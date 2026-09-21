// The single place every request to the backend goes through. Nothing in
// components/pages should ever call fetch() directly — see services/*.ts.

const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
    this.name = 'ApiError'
  }
}

function authHeaders(): Record<string, string> {
  const token = localStorage.getItem('toji_token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function request<T>(
  path: string,
  options: { method?: string; body?: unknown; auth?: boolean } = {}
): Promise<T> {
  const { method = 'GET', body, auth = false } = options

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(auth ? authHeaders() : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new ApiError(res.status, text || `Request failed with status ${res.status}`)
  }

  // Some endpoints (e.g. HostHeartbeat-style acks) may return no body; guard for that.
  const contentType = res.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export const http = {
  get: <T>(path: string, auth = false) => request<T>(path, { method: 'GET', auth }),
  post: <T>(path: string, body?: unknown, auth = false) =>
    request<T>(path, { method: 'POST', body, auth }),
}
