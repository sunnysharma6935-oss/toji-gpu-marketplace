package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"gpumarketplace/backend/internal/api"
	"gpumarketplace/backend/internal/db"
)

func main() {
	ctx := context.Background()

	if err := db.Connect(ctx); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	r := chi.NewRouter()

	// CORS for the local TOJI frontend.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	// Health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Authentication
	r.Post("/signup", api.Signup)
	r.Post("/login", api.Login)

	// Host
	r.Post("/hosts/register", api.RequireAuth(api.RegisterHost))
	r.Post("/hosts/heartbeat", api.HostHeartbeat)
	r.Post("/hosts/gpu-report", api.GPUReport)

	// Listings
	r.Post("/hosts/gpus/list", api.RequireAuth(api.CreateListing))
	r.Get("/listings", api.ListActiveListings)

	// Rentals
	r.Post("/rentals", api.RequireAuth(api.CreateRental))
	r.Get("/rentals", api.RequireAuth(api.ListCustomerRentals))
	r.Get("/rentals/{rental_id}", api.RequireAuth(api.GetCustomerRental))
	r.Get("/agent/rentals/pending", api.GetPendingRentals)

	// Rental lifecycle
	// These agent endpoints authenticate using X-Agent-Token,
	// so they must NOT use RequireAuth.
	r.Post("/agent/rentals/{rental_id}/ready", api.RentalReady)
	r.Post("/agent/rentals/{rental_id}/usage-heartbeat", api.UsageHeartbeat)
	r.Post("/agent/rentals/{rental_id}/stop", api.AgentStopRental)
	r.Post("/agent/rentals/{rental_id}/provisioning-failed", api.ProvisioningFailed)
	r.Get("/agent/rentals/active", api.ListActiveRentalsForHost)

	// Customer rental stop uses normal JWT authentication.
	r.Post("/rentals/{rental_id}/stop", api.RequireAuth(api.StopRental))

	// Billing
	r.Post("/billing/topup", api.RequireAuth(api.TopUp))
	r.Get("/billing/balance", api.RequireAuth(api.GetBalance))
	r.Get("/billing/transactions", api.RequireAuth(api.GetTransactions))

	// Host payouts
	r.Post("/host/payout", api.RequireAuth(api.RequestHostPayout))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
