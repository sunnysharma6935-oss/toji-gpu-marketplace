package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gpumarketplace/backend/internal/billing"
	"gpumarketplace/backend/internal/db"
)

// --- RentalReady: provisioning -> running -------------------------------------------

func RentalReady(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}
	rentalIDStr := chi.URLParam(r, "rental_id")
	rentalID, err := uuid.Parse(rentalIDStr)
	if err != nil {
		http.Error(w, "invalid rental id", http.StatusBadRequest)
		return
	}

	var req struct {
		SSHPort      int    `json:"ssh_port"`
		JupyterToken string `json:"jupyter_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SSHPort == 0 || req.JupyterToken == "" {
		http.Error(w, "ssh_port and jupyter_token required", http.StatusBadRequest)
		return
	}

	var hostID string
	if err := db.Pool.QueryRow(r.Context(), `SELECT id FROM hosts WHERE agent_token = $1`, token).Scan(&hostID); err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	tx, err := db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	var currentStatus string
	var ratePaisePerHour int64
	err = tx.QueryRow(r.Context(),
		`SELECT r.status, l.price_paise_per_hour
		 FROM rentals r
		 JOIN listings l ON l.id = r.listing_id
		 JOIN gpus g ON g.id = l.gpu_id
		 WHERE r.id = $1 AND g.host_id = $2`,
		rentalID, hostID,
	).Scan(&currentStatus, &ratePaisePerHour)
	if err != nil {
		http.Error(w, "rental not found for this host", http.StatusNotFound)
		return
	}

	if currentStatus == "running" {
		// Idempotent: agent may have retried this call after a network blip.
		writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
		return
	}
	if currentStatus != "provisioning" {
		http.Error(w, "rental is not in a state that can become running", http.StatusConflict)
		return
	}

	now := time.Now()
	if _, err = tx.Exec(r.Context(),
		`UPDATE rentals SET status='running', started_at=$2, ssh_port=$3, jupyter_token=$4 WHERE id=$1`,
		rentalID, now, req.SSHPort, req.JupyterToken,
	); err != nil {
		http.Error(w, "could not update rental", http.StatusInternalServerError)
		return
	}

	// Billing state is initialized ONLY here — never at CreateRental (that step only
	// reserves funds). This is what makes "billing begins only after running state"
	// true by construction: ProcessBillingTick requires this row and it doesn't exist
	// before this point.
	if err := billing.InitBillingState(r.Context(), tx, rentalID, ratePaisePerHour, now); err != nil {
		http.Error(w, "could not initialize billing state", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "commit failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

// --- UsageHeartbeat: the 30-second billing trigger -----------------------------------

func UsageHeartbeat(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}
	rentalID, err := uuid.Parse(chi.URLParam(r, "rental_id"))
	if err != nil {
		http.Error(w, "invalid rental id", http.StatusBadRequest)
		return
	}

	var hostID string
	if err := db.Pool.QueryRow(r.Context(), `SELECT id FROM hosts WHERE agent_token = $1`, token).Scan(&hostID); err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	var customerIDStr, status string
	err = db.Pool.QueryRow(r.Context(),
		`SELECT r.customer_id, r.status
		 FROM rentals r
		 JOIN listings l ON l.id = r.listing_id
		 JOIN gpus g ON g.id = l.gpu_id
		 WHERE r.id = $1 AND g.host_id = $2`,
		rentalID, hostID,
	).Scan(&customerIDStr, &status)
	if err != nil {
		http.Error(w, "rental not found for this host", http.StatusNotFound)
		return
	}
	if status != "running" {
		http.Error(w, "rental is not running", http.StatusConflict)
		return
	}

	var body struct {
		GPUUtilPct *int `json:"gpu_util_pct"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // body is optional; ignore decode errors on empty body

	if body.GPUUtilPct != nil {
		// Finally gives the existing (previously unused) usage_heartbeats table a writer.
		if _, err := db.Pool.Exec(r.Context(),
			`INSERT INTO usage_heartbeats (rental_id, gpu_util_pct) VALUES ($1, $2)`,
			rentalID, *body.GPUUtilPct,
		); err != nil {
			// Non-fatal — utilization logging failing shouldn't block billing.
		}
	}

	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	hostUUID, err := uuid.Parse(hostID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	custAccountID, err := billing.GetOrCreateAccount(r.Context(), db.Pool, "customer_balance", "customer", &customerID)
	if err != nil {
		http.Error(w, "could not resolve customer account", http.StatusInternalServerError)
		return
	}

	hostOwnerUserID, err := resolveHostOwnerUserID(r.Context(), hostUUID)
	if err != nil {
		http.Error(w, "could not resolve host owner", http.StatusInternalServerError)
		return
	}
	hostAccountID, err := billing.GetOrCreateAccount(r.Context(), db.Pool, "host_payable", "host", &hostOwnerUserID)
	if err != nil {
		http.Error(w, "could not resolve host account", http.StatusInternalServerError)
		return
	}

	result, err := billing.ProcessBillingTick(r.Context(), db.Pool, rentalID, custAccountID, hostAccountID, time.Now(), false)
	if err != nil {
		http.Error(w, "billing tick failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"must_stop":     result.MustStopRental,
		"charged_paise": result.ChargedPaise,
	})
}

// --- Shared stop/finalize core, used by both entry points ----------------------------

// finalizeRental is the ONE place a rental ever transitions into a terminal state
// ('stopped' or 'failed'). Both StopRental (customer) and AgentStopRental /
// ProvisioningFailed (agent) call this — not their own copies of the logic.
//
// CONCURRENCY FIX: this now holds a SINGLE transaction across the entire operation —
// the initial row lock on `rentals`, the final billing tick (via ProcessBillingTickTx,
// which accepts this same transaction instead of opening its own), and the status
// flip, all commit together or not at all. A second, near-simultaneous call to
// finalizeRental for the same rental blocks on the `SELECT ... FOR UPDATE` below until
// this entire operation commits — then it reads the now-terminal status and takes the
// idempotent no-op branch. No queue, no new infrastructure: one transaction instead of
// the previous two is the entire fix, exactly as scoped.
func finalizeRental(ctx context.Context, rentalID uuid.UUID, targetStatus string, reason string) (map[string]interface{}, error) {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) // no-op once committed below

	var status, listingID, customerIDStr string
	err = tx.QueryRow(ctx,
		`SELECT status, listing_id, customer_id FROM rentals WHERE id = $1 FOR UPDATE`,
		rentalID,
	).Scan(&status, &listingID, &customerIDStr)
	if err != nil {
		return nil, err
	}

	if status == "stopped" || status == "failed" {
		// The row lock we just acquired guarantees this read is current: if another
		// finalize call got here first, it already committed by the time we could
		// acquire this lock, so "already final" is genuinely true, not a stale read.
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": status, "already_final": true}, nil
	}

	if status == "provisioning" {
		if err := billing.ReleaseReservation(ctx, tx, rentalID); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx,
			`UPDATE rentals SET status=$2, stopped_at=now(), status_reason=$3 WHERE id=$1`,
			rentalID, targetStatus, reason,
		); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": targetStatus, "charged_paise": 0}, nil
	}

	// status == "running": everything from here happens in the SAME transaction we
	// opened above — no intermediate commit. The rentals row lock stays held for the
	// full duration of the billing tick and the status flip.
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return nil, err
	}
	custAccountID, err := billing.GetOrCreateAccount(ctx, tx, "customer_balance", "customer", &customerID)
	if err != nil {
		return nil, err
	}

	var hostIDStr string
	if err := tx.QueryRow(ctx,
		`SELECT g.host_id FROM listings l JOIN gpus g ON g.id = l.gpu_id WHERE l.id = $1`,
		listingID,
	).Scan(&hostIDStr); err != nil {
		return nil, err
	}
	hostUUID, err := uuid.Parse(hostIDStr)
	if err != nil {
		return nil, err
	}
	var hostOwnerUserIDStr string
	if err := tx.QueryRow(ctx, `SELECT user_id FROM hosts WHERE id=$1`, hostUUID).Scan(&hostOwnerUserIDStr); err != nil {
		return nil, err
	}
	hostOwnerUserID, err := uuid.Parse(hostOwnerUserIDStr)
	if err != nil {
		return nil, err
	}
	hostAccountID, err := billing.GetOrCreateAccount(ctx, tx, "host_payable", "host", &hostOwnerUserID)
	if err != nil {
		return nil, err
	}

	tickResult, err := billing.ProcessBillingTickTx(ctx, tx, rentalID, custAccountID, hostAccountID, time.Now(), true)
	if err != nil {
		return nil, err
	}

	// No `WHERE status='running'` guard needed here — the row lock has been held
	// continuously since the SELECT above, so nothing else could have changed this
	// row's status in the meantime. That guard was needed in the old two-transaction
	// version specifically because the lock was released early; it no longer is.
	if _, err = tx.Exec(ctx,
		`UPDATE rentals SET status=$2, stopped_at=now(), status_reason=$3 WHERE id=$1`,
		rentalID, targetStatus, reason,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":          targetStatus,
		"charged_paise":   tickResult.ChargedPaise,
		"shortfall_paise": tickResult.ShortfallPaise,
	}, nil
}

// --- Customer-facing stop -------------------------------------------------------------

func StopRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)
	rentalID, err := uuid.Parse(chi.URLParam(r, "rental_id"))
	if err != nil {
		http.Error(w, "invalid rental id", http.StatusBadRequest)
		return
	}

	var ownerID string
	if err := db.Pool.QueryRow(r.Context(), `SELECT customer_id FROM rentals WHERE id=$1`, rentalID).Scan(&ownerID); err != nil {
		http.Error(w, "rental not found", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "not your rental", http.StatusForbidden)
		return
	}

	result, err := finalizeRental(r.Context(), rentalID, "stopped", "customer_requested")
	if err != nil {
		http.Error(w, "could not stop rental", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// --- Agent-facing stop (called immediately after a must_stop=true heartbeat) ----------

func AgentStopRental(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}
	rentalID, err := uuid.Parse(chi.URLParam(r, "rental_id"))
	if err != nil {
		http.Error(w, "invalid rental id", http.StatusBadRequest)
		return
	}

	var hostID string
	if err := db.Pool.QueryRow(r.Context(), `SELECT id FROM hosts WHERE agent_token=$1`, token).Scan(&hostID); err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}
	if !rentalBelongsToHost(r.Context(), rentalID, hostID) {
		http.Error(w, "rental not found for this host", http.StatusNotFound)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := req.Reason
	if reason == "" {
		reason = "insufficient_balance"
	}

	result, err := finalizeRental(r.Context(), rentalID, "stopped", reason)
	if err != nil {
		http.Error(w, "could not stop rental", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// --- Provisioning failure --------------------------------------------------------------

func ProvisioningFailed(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}
	rentalID, err := uuid.Parse(chi.URLParam(r, "rental_id"))
	if err != nil {
		http.Error(w, "invalid rental id", http.StatusBadRequest)
		return
	}

	var hostID string
	if err := db.Pool.QueryRow(r.Context(), `SELECT id FROM hosts WHERE agent_token=$1`, token).Scan(&hostID); err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}
	if !rentalBelongsToHost(r.Context(), rentalID, hostID) {
		http.Error(w, "rental not found for this host", http.StatusNotFound)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := "provisioning_failed"
	if req.Reason != "" {
		reason = "provisioning_failed: " + req.Reason
	}

	// Uses the SAME finalizeRental core as a stop — a provisioning failure is just a
	// terminal transition with a different target status ('failed' instead of
	// 'stopped'). No parallel failure-handling mechanism was invented.
	result, err := finalizeRental(r.Context(), rentalID, "failed", reason)
	if err != nil {
		http.Error(w, "could not record provisioning failure", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// --- Active-rental reconciliation, for the agent's stop-side polling ------------------

func ListActiveRentalsForHost(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")
	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}
	var hostID string
	if err := db.Pool.QueryRow(r.Context(), `SELECT id FROM hosts WHERE agent_token=$1`, token).Scan(&hostID); err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	rows, err := db.Pool.Query(r.Context(),
		`SELECT r.id FROM rentals r
		 JOIN listings l ON l.id = r.listing_id
		 JOIN gpus g ON g.id = l.gpu_id
		 WHERE g.host_id = $1 AND r.status IN ('provisioning', 'running')`,
		hostID,
	)
	if err != nil {
		http.Error(w, "could not load active rentals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			http.Error(w, "could not read active rentals", http.StatusInternalServerError)
			return
		}
		ids = append(ids, id)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"active_rental_ids": ids})
}

// --- shared helpers ---------------------------------------------------------------------

func resolveHostOwnerUserID(ctx context.Context, hostID uuid.UUID) (uuid.UUID, error) {
	var userIDStr string
	if err := db.Pool.QueryRow(ctx, `SELECT user_id FROM hosts WHERE id=$1`, hostID).Scan(&userIDStr); err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(userIDStr)
}

func rentalBelongsToHost(ctx context.Context, rentalID uuid.UUID, hostID string) bool {
	var ownerHostID string
	err := db.Pool.QueryRow(ctx,
		`SELECT g.host_id FROM rentals r JOIN listings l ON l.id=r.listing_id JOIN gpus g ON g.id=l.gpu_id WHERE r.id=$1`,
		rentalID,
	).Scan(&ownerHostID)
	return err == nil && ownerHostID == hostID
}
