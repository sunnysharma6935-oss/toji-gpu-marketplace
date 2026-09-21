import { useState, useCallback } from 'react'
import { getSession, saveSession, clearSession } from '@/services/authService'
import type { AuthResponse } from '@/types/api'

export function useAuth() {
  const [session, setSession] = useState(getSession())

  const login = useCallback((res: AuthResponse) => {
    saveSession(res)
    setSession({ token: res.token, userId: res.user_id })
  }, [])

  const logout = useCallback(() => {
    clearSession()
    setSession(null)
  }, [])

  return { session, isAuthenticated: !!session, login, logout }
}
