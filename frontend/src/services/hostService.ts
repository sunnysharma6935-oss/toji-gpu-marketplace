import { http } from './httpClient'
import type { RegisterHostResponse } from '@/types/api'

// Confirmed real endpoint: POST /hosts/register (backend/internal/api/handlers.go)
export async function registerHost(machineLabel: string): Promise<RegisterHostResponse> {
  return http.post<RegisterHostResponse>('/hosts/register', { machine_label: machineLabel }, true)
}

// NOT YET AVAILABLE ON THE BACKEND — none of these exist yet:
//   GET /hosts/mine            — list the current user's registered machines
//   GET /hosts/{id}/gpus       — GPUs detected/reported on a machine
//   POST /hosts/gpus/{id}/listing — create/update a listing (CreateListing exists but
//                                   is keyed by gpu_id via POST /hosts/gpus/list; no
//                                   GET to list or edit an existing listing yet)
//   GET /host/earnings         — host earnings summary (ledger-backed, per prior
//                                   architecture work, but no HTTP endpoint exposes it)
//
// The Host Dashboard pages are built and structured to consume these once they exist,
// but currently render an explicit "waiting on backend" state — no invented machine
// lists, no invented earnings numbers, per the mission brief's explicit rule against
// fake financial/marketplace data.
