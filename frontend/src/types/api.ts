// Types mirror the ACTUAL current backend contracts, confirmed by reading
// backend/internal/api/*.go directly — not assumed, not invented.

export interface AuthResponse {
  user_id: string
  token: string
}

// GET /listings — confirmed: backend/internal/api/handlers.go, ListActiveListings.
// This is the ONLY listing-read endpoint that currently exists. There is no
// GET /listings/{id} single-item endpoint yet — see services/api.ts for how the
// GPU detail page works around that honestly rather than inventing one.
export interface Listing {
  listing_id: string
  gpu_id: string
  model: string
  vram_mb: number
  driver_version: string
  machine_label: string
  price_paise_per_hour: number
}

// POST /rentals — confirmed: handlers.go, CreateRental.
export interface CreateRentalResponse {
  rental_id: string
  status: 'provisioning'
}

// POST /rentals/{rental_id}/stop — confirmed: rental_lifecycle.go, StopRental.
export interface StopRentalResponse {
  status: string
  charged_paise?: number
  shortfall_paise?: number
  already_final?: boolean
}

// GET /billing/balance — confirmed: billing_handlers.go, GetBalance.
export interface BalanceResponse {
  balance_paise: number
  spendable_paise: number
  currency: 'INR'
}

// POST /billing/topup — confirmed: billing_handlers.go, TopUp.
export type TopUpMethod = 'upi' | 'card' | 'netbanking' | 'manual'

export interface TopUpResponse {
  payment_intent_id: string
  status: string
}

// GET /billing/transactions — confirmed: billing_handlers.go, GetTransactions.
export interface Transaction {
  id: string
  amount_paise: number
  type: string
  status: string
  created_at: string
}

// POST /hosts/register — confirmed: handlers.go, RegisterHost.
export interface RegisterHostResponse {
  host_id: string
  agent_token: string
}
