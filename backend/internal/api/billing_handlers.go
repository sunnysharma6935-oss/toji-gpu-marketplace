package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"gpumarketplace/backend/internal/billing"
	"gpumarketplace/backend/internal/db"
)

// POST /billing/topup  { "amount_paise": 2500, "method": "card" }
// Creates a mock PaymentIntent and immediately confirms it (since no real provider
// exists yet). Once Razorpay/Cashfree is wired in, this becomes two steps: initiate,
// then a webhook calls ConfirmTopUpSuccess — that split already exists in billing.go
// specifically so that future change doesn't touch this handler's shape.
func TopUp(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)
	customerID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "invalid user", http.StatusInternalServerError)
		return
	}

	var req struct {
		AmountPaise int64  `json:"amount_paise"`
		Method      string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountPaise <= 0 {
		http.Error(w, "amount_paise (positive) and method required", http.StatusBadRequest)
		return
	}
	if req.Method != "upi" && req.Method != "card" && req.Method != "netbanking" && req.Method != "manual" {
		http.Error(w, "method must be one of: upi, card, netbanking, manual", http.StatusBadRequest)
		return
	}

	piID, err := billing.InitiateTopUp(r.Context(), db.Pool, customerID, req.AmountPaise, req.Method)
	if err != nil {
		http.Error(w, "could not initiate top-up", http.StatusInternalServerError)
		return
	}
	if err := billing.ConfirmTopUpSuccess(r.Context(), db.Pool, piID); err != nil {
		http.Error(w, "top-up failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"payment_intent_id": piID.String(),
		"status":            "succeeded",
	})
}

// GET /billing/balance — returns TOJI balance (never call this a wallet in responses
// or copy that reach the frontend).
func GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)
	customerID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "invalid user", http.StatusInternalServerError)
		return
	}

	accountID, err := billing.GetOrCreateAccount(r.Context(), db.Pool, "customer_balance", "customer", &customerID)
	if err != nil {
		http.Error(w, "could not resolve account", http.StatusInternalServerError)
		return
	}

	balance, err := billing.Balance(r.Context(), db.Pool, accountID)
	if err != nil {
		http.Error(w, "could not compute balance", http.StatusInternalServerError)
		return
	}
	spendable, err := billing.SpendableBalance(r.Context(), db.Pool, accountID)
	if err != nil {
		http.Error(w, "could not compute spendable balance", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"balance_paise":   balance,
		"spendable_paise": spendable,
		"currency":        "INR",
	})
}

// GET /billing/transactions
func GetTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)
	customerID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "invalid user", http.StatusInternalServerError)
		return
	}

	rows, err := db.Pool.Query(r.Context(),
		`SELECT le.id, le.amount_paise, le.entry_type, le.status, le.created_at
		 FROM ledger_entries le
		 JOIN ledger_accounts la ON la.id = le.account_id
		 WHERE la.account_type = 'customer_balance' AND la.owner_id = $1
		 ORDER BY le.created_at DESC LIMIT 100`,
		customerID,
	)
	if err != nil {
		http.Error(w, "could not load transactions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type txRow struct {
		ID          string `json:"id"`
		AmountPaise int64  `json:"amount_paise"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
	}
	out := []txRow{}
	for rows.Next() {
		var t txRow
		if err := rows.Scan(&t.ID, &t.AmountPaise, &t.Type, &t.Status, &t.CreatedAt); err != nil {
			http.Error(w, "could not read transactions", http.StatusInternalServerError)
			return
		}
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /host/payout  { "amount_paise": 8500, "method": "manual" }
// Manual trigger only — nothing schedules this automatically, per the approved decision.
//
// NOTE, flagged rather than guessed: this resolves the authenticated user's id directly
// as the host_payable account's owner. Your schema has a separate `hosts` table (one
// user can own multiple host machines) — confirm whether the ledger's host owner_id
// should be the user id or a specific hosts.id before this ships, since I don't have
// your current hosts-table relationships to know which is correct here.
func RequestHostPayout(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)
	hostUserID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "invalid user", http.StatusInternalServerError)
		return
	}

	var req struct {
		AmountPaise int64  `json:"amount_paise"`
		Method      string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountPaise <= 0 {
		http.Error(w, "amount_paise (positive) and method required", http.StatusBadRequest)
		return
	}

	hostAccountID, err := billing.GetOrCreateAccount(r.Context(), db.Pool, "host_payable", "host", &hostUserID)
	if err != nil {
		http.Error(w, "could not resolve host account", http.StatusInternalServerError)
		return
	}

	piID, err := billing.RequestPayout(r.Context(), db.Pool, hostUserID, hostAccountID, req.AmountPaise, req.Method)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"payment_intent_id": piID.String(), "status": "initiated"})
}
