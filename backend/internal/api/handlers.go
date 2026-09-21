package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gpumarketplace/backend/internal/auth"
	"gpumarketplace/backend/internal/billing"
	"gpumarketplace/backend/internal/db"
)

// --- Signup ---

type signupReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Signup(w http.ResponseWriter, r *http.Request) {
	var req signupReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "email required, password must be 8+ chars", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id := uuid.New()

	_, err = db.Pool.Exec(
		r.Context(),
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		id,
		req.Email,
		hash,
	)

	if err != nil {
		http.Error(w, "could not create user (email may already exist)", http.StatusConflict)
		return
	}

	token, _ := auth.GenerateToken(id.String())

	writeJSON(w, http.StatusCreated, map[string]string{
		"user_id": id.String(),
		"token":   token,
	})
}

// --- Login ---

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var id, hash string

	err := db.Pool.QueryRow(
		r.Context(),
		`SELECT id, password_hash FROM users WHERE email = $1`,
		req.Email,
	).Scan(&id, &hash)

	if err != nil || !auth.CheckPassword(hash, req.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, _ := auth.GenerateToken(id)

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id": id,
		"token":   token,
	})
}

// --- Host registration ---

type registerHostReq struct {
	MachineLabel string `json:"machine_label"`
}

func RegisterHost(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)

	var req registerHostReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MachineLabel == "" {
		http.Error(w, "machine_label required", http.StatusBadRequest)
		return
	}

	hostID := uuid.New()
	agentToken := randomToken(32)

	_, err := db.Pool.Exec(
		r.Context(),
		`INSERT INTO hosts (id, user_id, machine_label, agent_token, status)
		 VALUES ($1, $2, $3, $4, 'pending')`,
		hostID,
		userID,
		req.MachineLabel,
		agentToken,
	)

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"host_id":     hostID.String(),
		"agent_token": agentToken,
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// --- Host heartbeat ---

func HostHeartbeat(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")

	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}

	result, err := db.Pool.Exec(
		r.Context(),
		`UPDATE hosts
		 SET last_heartbeat_at = NOW(), status = 'online'
		 WHERE agent_token = $1`,
		token,
	)

	if err != nil {
		http.Error(w, "heartbeat failed", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// --- GPU report ---

func GPUReport(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")

	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}

	var req struct {
		GPUIndex      int    `json:"gpu_index"`
		Model         string `json:"model"`
		VRAMMB        int    `json:"vram_mb"`
		DriverVersion string `json:"driver_version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var hostID string

	err := db.Pool.QueryRow(
		r.Context(),
		`SELECT id FROM hosts WHERE agent_token = $1`,
		token,
	).Scan(&hostID)

	if err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	_, err = db.Pool.Exec(
		r.Context(),
		`INSERT INTO gpus
			(id, host_id, gpu_index, model, vram_mb, driver_version)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (host_id, gpu_index)
		 DO UPDATE SET
			model = EXCLUDED.model,
			vram_mb = EXCLUDED.vram_mb,
			driver_version = EXCLUDED.driver_version`,
		uuid.New(),
		hostID,
		req.GPUIndex,
		req.Model,
		req.VRAMMB,
		req.DriverVersion,
	)

	if err != nil {
		http.Error(w, "gpu report failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// --- Create listing ---

func CreateListing(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)

	var req struct {
		GPUID          string `json:"gpu_id"`
		PricePaiseHour int64  `json:"price_paise_per_hour"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.GPUID == "" || req.PricePaiseHour <= 0 {
		http.Error(w, "gpu_id and positive price required", http.StatusBadRequest)
		return
	}

	var gpuID string

	err := db.Pool.QueryRow(
		r.Context(),
		`SELECT g.id
		 FROM gpus g
		 JOIN hosts h ON h.id = g.host_id
		 WHERE g.id = $1
		   AND h.user_id = $2`,
		req.GPUID,
		userID,
	).Scan(&gpuID)

	if err != nil {
		http.Error(w, "GPU not found or not owned by user", http.StatusNotFound)
		return
	}

	listingID := uuid.New()

	_, err = db.Pool.Exec(
		r.Context(),
		`INSERT INTO listings
			(id, gpu_id, price_paise_per_hour, is_active)
		 VALUES ($1, $2, $3, true)`,
		listingID,
		gpuID,
		req.PricePaiseHour,
	)

	if err != nil {
		http.Error(w, "could not create listing", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"listing_id":           listingID.String(),
		"gpu_id":               gpuID,
		"price_paise_per_hour": req.PricePaiseHour,
		"is_active":            true,
	})
}

// --- Active listings ---

func ListActiveListings(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(
		r.Context(),
		`SELECT
			l.id,
			g.id,
			g.model,
			g.vram_mb,
			COALESCE(g.driver_version, ''),
			h.machine_label,
			l.price_paise_per_hour
		 FROM listings l
		 JOIN gpus g ON g.id = l.gpu_id
		 JOIN hosts h ON h.id = g.host_id
		 WHERE l.is_active = true
		   AND h.status = 'online'
		 ORDER BY l.price_paise_per_hour ASC`,
	)

	if err != nil {
		http.Error(w, "could not load listings", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type Listing struct {
		ListingID         string `json:"listing_id"`
		GPUID             string `json:"gpu_id"`
		Model             string `json:"model"`
		VRAMMB            int    `json:"vram_mb"`
		DriverVersion     string `json:"driver_version"`
		MachineLabel      string `json:"machine_label"`
		PricePaisePerHour int64  `json:"price_paise_per_hour"`
	}

	listings := []Listing{}

	for rows.Next() {
		var item Listing

		if err := rows.Scan(
			&item.ListingID,
			&item.GPUID,
			&item.Model,
			&item.VRAMMB,
			&item.DriverVersion,
			&item.MachineLabel,
			&item.PricePaisePerHour,
		); err != nil {
			http.Error(w, "could not read listings", http.StatusInternalServerError)
			return
		}

		listings = append(listings, item)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "could not read listings", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, listings)
}

// --- Create rental ---

// CreateRental reserves the customer's TOJI balance in the SAME
// transaction as creating the rental row.
func CreateRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)

	custUUID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var req struct {
		ListingID string `json:"listing_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ListingID == "" {
		http.Error(w, "listing_id required", http.StatusBadRequest)
		return
	}

	tx, err := db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	defer tx.Rollback(r.Context())

	var listingID string
	var ratePaisePerHour int64

	err = tx.QueryRow(
		r.Context(),
		`SELECT l.id, l.price_paise_per_hour
		 FROM listings l
		 WHERE l.id = $1
		   AND l.is_active = true
		   AND NOT EXISTS (
		       SELECT 1
		       FROM rentals r
		       WHERE r.listing_id = l.id
		         AND r.status IN ('provisioning', 'running')
		   )`,
		req.ListingID,
	).Scan(&listingID, &ratePaisePerHour)

	if err != nil {
		http.Error(w, "listing not available", http.StatusConflict)
		return
	}

	rentalID := uuid.New()

	// Insert rental row FIRST because reservations.rental_id
	// is a foreign key to rentals(id).
	_, err = tx.Exec(
		r.Context(),
		`INSERT INTO rentals
			(id, listing_id, customer_id, status)
		 VALUES ($1, $2, $3, 'provisioning')`,
		rentalID,
		listingID,
		userID,
	)

	if err != nil {
		http.Error(w, "could not create rental", http.StatusInternalServerError)
		return
	}

	custAccountID, err := billing.GetOrCreateAccount(
		r.Context(),
		tx,
		"customer_balance",
		"customer",
		&custUUID,
	)

	if err != nil {
		http.Error(w, "could not resolve customer account", http.StatusInternalServerError)
		return
	}

	if err := billing.ReserveAtRentalCreation(
		r.Context(),
		tx,
		custAccountID,
		rentalID,
		ratePaisePerHour,
	); err != nil {

		if errors.Is(err, billing.ErrInsufficientBalance) {
			http.Error(
				w,
				"insufficient TOJI balance to start this rental",
				http.StatusPaymentRequired,
			)
			return
		}

		log.Printf(
			"CreateRental: reservation failed for rental %s: %v",
			rentalID,
			err,
		)

		http.Error(w, "could not reserve balance", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "could not create rental", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"rental_id": rentalID.String(),
		"status":    "provisioning",
	})
}

// --- Customer rentals ---

func ListCustomerRentals(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)

	rows, err := db.Pool.Query(
		r.Context(),
		`SELECT
			r.id,
			r.listing_id,
			r.status,
			r.ssh_port,
			r.jupyter_token,
			r.started_at,
			r.stopped_at,
			r.total_paise_billed,
			r.created_at,
			g.model,
			g.vram_mb,
			COALESCE(g.driver_version, ''),
			h.machine_label,
			l.price_paise_per_hour
		 FROM rentals r
		 JOIN listings l ON l.id = r.listing_id
		 JOIN gpus g ON g.id = l.gpu_id
		 JOIN hosts h ON h.id = g.host_id
		 WHERE r.customer_id = $1
		 ORDER BY r.created_at DESC`,
		userID,
	)

	if err != nil {
		http.Error(w, "could not load rentals", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type Rental struct {
		RentalID          string     `json:"rental_id"`
		ListingID         string     `json:"listing_id"`
		Status            string     `json:"status"`
		SSHPort           *int       `json:"ssh_port,omitempty"`
		JupyterToken      *string    `json:"jupyter_token,omitempty"`
		StartedAt         *time.Time `json:"started_at,omitempty"`
		StoppedAt         *time.Time `json:"stopped_at,omitempty"`
		TotalPaiseBilled  int64      `json:"total_paise_billed"`
		CreatedAt         time.Time  `json:"created_at"`
		Model             string     `json:"model"`
		VRAMMB            int        `json:"vram_mb"`
		DriverVersion     string     `json:"driver_version"`
		MachineLabel      string     `json:"machine_label"`
		PricePaisePerHour int64      `json:"price_paise_per_hour"`
	}

	rentals := []Rental{}

	for rows.Next() {
		var item Rental

		if err := rows.Scan(
			&item.RentalID,
			&item.ListingID,
			&item.Status,
			&item.SSHPort,
			&item.JupyterToken,
			&item.StartedAt,
			&item.StoppedAt,
			&item.TotalPaiseBilled,
			&item.CreatedAt,
			&item.Model,
			&item.VRAMMB,
			&item.DriverVersion,
			&item.MachineLabel,
			&item.PricePaisePerHour,
		); err != nil {
			http.Error(w, "could not read rentals", http.StatusInternalServerError)
			return
		}

		rentals = append(rentals, item)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "could not read rentals", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, rentals)
}

// --- Customer rental detail ---

func GetCustomerRental(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxUserID).(string)
	rentalID := chi.URLParam(r, "rental_id")

	if rentalID == "" {
		http.Error(w, "rental_id required", http.StatusBadRequest)
		return
	}

	type RentalDetail struct {
		RentalID          string     `json:"rental_id"`
		ListingID         string     `json:"listing_id"`
		Status            string     `json:"status"`
		SSHPort           *int       `json:"ssh_port,omitempty"`
		JupyterToken      *string    `json:"jupyter_token,omitempty"`
		StartedAt         *time.Time `json:"started_at,omitempty"`
		StoppedAt         *time.Time `json:"stopped_at,omitempty"`
		TotalPaiseBilled  int64      `json:"total_paise_billed"`
		CreatedAt         time.Time  `json:"created_at"`
		Model             string     `json:"model"`
		VRAMMB            int        `json:"vram_mb"`
		DriverVersion     string     `json:"driver_version"`
		MachineLabel      string     `json:"machine_label"`
		PricePaisePerHour int64      `json:"price_paise_per_hour"`
	}

	var item RentalDetail

	err := db.Pool.QueryRow(
		r.Context(),
		`SELECT
			r.id,
			r.listing_id,
			r.status,
			r.ssh_port,
			r.jupyter_token,
			r.started_at,
			r.stopped_at,
			r.total_paise_billed,
			r.created_at,
			g.model,
			g.vram_mb,
			COALESCE(g.driver_version, ''),
			h.machine_label,
			l.price_paise_per_hour
		 FROM rentals r
		 JOIN listings l ON l.id = r.listing_id
		 JOIN gpus g ON g.id = l.gpu_id
		 JOIN hosts h ON h.id = g.host_id
		 WHERE r.id = $1
		   AND r.customer_id = $2`,
		rentalID,
		userID,
	).Scan(
		&item.RentalID,
		&item.ListingID,
		&item.Status,
		&item.SSHPort,
		&item.JupyterToken,
		&item.StartedAt,
		&item.StoppedAt,
		&item.TotalPaiseBilled,
		&item.CreatedAt,
		&item.Model,
		&item.VRAMMB,
		&item.DriverVersion,
		&item.MachineLabel,
		&item.PricePaisePerHour,
	)

	if err != nil {
		log.Printf("GetCustomerRental failed: rental_id=%s user_id=%s error=%v", rentalID, userID, err)
		http.Error(w, "rental not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// --- Agent pending rentals ---

func GetPendingRentals(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Agent-Token")

	if token == "" {
		http.Error(w, "missing agent token", http.StatusUnauthorized)
		return
	}

	var hostID string

	err := db.Pool.QueryRow(
		r.Context(),
		`SELECT id FROM hosts WHERE agent_token = $1`,
		token,
	).Scan(&hostID)

	if err != nil {
		http.Error(w, "invalid agent token", http.StatusUnauthorized)
		return
	}

	rows, err := db.Pool.Query(
		r.Context(),
		`SELECT r.id, g.gpu_index
		 FROM rentals r
		 JOIN listings l ON l.id = r.listing_id
		 JOIN gpus g ON g.id = l.gpu_id
		 WHERE g.host_id = $1
		   AND r.status = 'provisioning'`,
		hostID,
	)

	if err != nil {
		http.Error(w, "could not load pending rentals", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type PendingRental struct {
		RentalID string `json:"rental_id"`
		GPUIndex int    `json:"gpu_index"`
	}

	pending := []PendingRental{}

	for rows.Next() {
		var item PendingRental

		if err := rows.Scan(
			&item.RentalID,
			&item.GPUIndex,
		); err != nil {
			http.Error(w, "could not read pending rentals", http.StatusInternalServerError)
			return
		}

		pending = append(pending, item)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "could not read pending rentals", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, pending)
}
