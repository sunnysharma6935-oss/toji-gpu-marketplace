import { http } from './httpClient'
import type { Listing } from '@/types/api'

// Confirmed real endpoint: GET /listings (backend/internal/api/handlers.go, ListActiveListings)
export async function getActiveListings(): Promise<Listing[]> {
  return http.get<Listing[]>('/listings')
}

// NOT YET AVAILABLE ON THE BACKEND: there is no GET /listings/{id} endpoint.
// The GPU detail page needs a single listing by id. Per the mission brief's rule
// ("do not invent backend endpoints... build the UI, isolate a clearly-marked
// placeholder"), this fetches the full active list and finds the match client-side —
// which is a real, working implementation today, not a mock — while remaining
// something you should replace with a real GET /listings/{id} call once the backend
// adds one, since fetching the entire list to view one GPU won't scale.
//
// BACKEND TODO: add GET /listings/{id} returning a single Listing.
export async function getListingById(listingId: string): Promise<Listing | null> {
  const all = await getActiveListings()
  return all.find((l) => l.listing_id === listingId) ?? null
}
