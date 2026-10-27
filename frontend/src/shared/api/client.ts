import type { z } from 'zod'
import { ApiError } from './errors'

const raw = import.meta.env.VITE_API_BASE_URL as string | undefined
if (!raw) throw new Error('VITE_API_BASE_URL is not set')
export const BASE = raw.replace(/\/$/, '')

// The auth token lives in memory; the auth provider mirrors it to storage.
let token: string | null = null
let onUnauthenticated: (() => void) | null = null
export const auth = {
  get token() { return token },
  set(t: string | null) { token = t },
  onUnauthenticated(fn: () => void) { onUnauthenticated = fn },
}

export interface ApiInit extends Omit<RequestInit, 'body' | 'method'> {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  schema?: z.ZodTypeAny
  idempotencyKey?: string
  ifMatch?: number
  timeoutMs?: number
  query?: Record<string, string | number | boolean | undefined>
}

export async function api<T>(path: string, init: ApiInit = {}): Promise<T> {
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), init.timeoutMs ?? 10_000)
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (init.body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (init.idempotencyKey) headers['Idempotency-Key'] = init.idempotencyKey
  if (init.ifMatch !== undefined) headers['If-Match'] = `"${init.ifMatch}"`

  let url = BASE + path
  if (init.query) {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(init.query)) if (v !== undefined && v !== '') qs.set(k, String(v))
    const s = qs.toString()
    if (s) url += (url.includes('?') ? '&' : '?') + s
  }

  let res: Response
  try {
    res = await fetch(url, { ...init, method: init.method ?? 'GET', headers, body: init.body === undefined ? undefined : JSON.stringify(init.body), signal: ctrl.signal })
  } catch (e) {
    clearTimeout(t)
    if (e instanceof DOMException && e.name === 'AbortError') throw new ApiError('timeout', 'The server took too long to answer.', 0)
    throw new ApiError('network', 'Could not reach the server. Check your connection.', 0)
  }
  clearTimeout(t)

  if (res.status === 204) return undefined as T
  const body: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const err = ApiError.from(res, body)
    if (res.status === 401 && !path.endsWith('/auth/login')) onUnauthenticated?.()
    throw err
  }
  if (init.schema) {
    const parsed = init.schema.safeParse(body)
    if (!parsed.success) {
      console.error('contract drift', path, parsed.error.issues.slice(0, 5))
      throw new ApiError('contract_drift', 'Something changed on the server. Reload the page.', res.status)
    }
    return parsed.data as T
  }
  return body as T
}

export function newIdempotencyKey(): string {
  return crypto.randomUUID()
}
