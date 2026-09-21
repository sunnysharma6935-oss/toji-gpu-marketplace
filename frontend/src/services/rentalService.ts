import { http } from './httpClient'
import type { CreateRentalResponse, StopRentalResponse } from '@/types/api'

// Confirmed real endpoint: POST /rentals (backend/internal/api/handlers.go, CreateRental)
// Known real error responses from the backend, worth handling explicitly in the UI:
//   402 — insufficient TOJI balance to start this rental
//   409 — listing not available (already rented, or inactive)
export async function createRental(listingId: string): Promise<CreateRentalResponse> {
  return http.post<CreateRentalResponse>('/rentals', { listing_id: listingId }, true)
}

// Confirmed real endpoint: POST /rentals/{rental_id}/stop (rental_lifecycle.go, StopRental)
export async function stopRental(rentalId: string): Promise<StopRentalResponse> {
  return http.post<StopRentalResponse>(`/rentals/${rentalId}/stop`, undefined, true)
}

// NOT YET AVAILABLE ON THE BACKEND: no GET /rentals (list mine) or GET /rentals/{id}
// (single rental status/polling) endpoint exists yet. "My Rentals" and the active
// rental detail/workspace view cannot be data-driven until these exist — the
// corresponding pages are built and structured, but clearly render a "waiting on
// backend" state rather than fabricated rental data. See pages/customer/MyRentals.tsx
// and pages/customer/RentalDetail.tsx.
//
// BACKEND TODO: GET /rentals (customer's own, from JWT) and GET /rentals/{id}.
