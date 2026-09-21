import { http } from './httpClient'
import type { AuthResponse } from '@/types/api'

// Confirmed real endpoints: POST /signup, POST /login (backend/internal/api/handlers.go)
//
// IMPORTANT ARCHITECTURE NOTE, confirmed from the backend source: there is only ONE
// user table and ONE signup/login flow. "Customer" and "host" are not separate account
// types — any signed-in user can register a host machine via a SEPARATE authenticated
// call (POST /hosts/register) after signing up normally. There is no "host signup" or
// "host login" endpoint distinct from the regular one. The UI below reflects this: the
// host-onboarding flow assumes the visitor is already authenticated (or sends them
// through the same signup/login as a customer), then calls registerHost() as a second
// step — it does not pretend a separate host account system exists.

export async function signup(email: string, password: string): Promise<AuthResponse> {
  return http.post<AuthResponse>('/signup', { email, password })
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  return http.post<AuthResponse>('/login', { email, password })
}

export function saveSession(res: AuthResponse) {
  localStorage.setItem('toji_token', res.token)
  localStorage.setItem('toji_user_id', res.user_id)
}

export function clearSession() {
  localStorage.removeItem('toji_token')
  localStorage.removeItem('toji_user_id')
}

export function getSession(): { token: string; userId: string } | null {
  const token = localStorage.getItem('toji_token')
  const userId = localStorage.getItem('toji_user_id')
  if (!token || !userId) return null
  return { token, userId }
}
