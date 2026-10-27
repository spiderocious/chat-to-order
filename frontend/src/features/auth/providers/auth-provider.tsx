import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, auth as tokenStore } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { Data, LoginRes, MeRes, type Admin } from '@shared/api/types'
import { storage } from '@shared/utils/storage'

const KEY = 'ctpo.token'

interface AuthValue {
  status: 'loading' | 'anon' | 'authed'
  admin: Admin | null
  expiresAt: string | null
  login(email: string, password: string): Promise<void>
  logout(): Promise<void>
}

const AuthContext = createContext<AuthValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient()
  const [status, setStatus] = useState<AuthValue['status']>('loading')
  const [admin, setAdmin] = useState<Admin | null>(null)
  const [expiresAt, setExpiresAt] = useState<string | null>(null)

  const clear = useCallback(() => {
    tokenStore.set(null)
    storage.remove(KEY)
    setAdmin(null)
    setExpiresAt(null)
    setStatus('anon')
    qc.clear()
  }, [qc])

  useEffect(() => {
    tokenStore.onUnauthenticated(clear)
    const saved = storage.get(KEY)
    if (!saved) {
      setStatus('anon')
      return
    }
    tokenStore.set(saved)
    api(EP.AUTH.ME, { schema: Data(MeRes) })
      .then((res) => {
        const r = res as { data: { admin: Admin; expires_at?: string | null } }
        setAdmin(r.data.admin)
        setExpiresAt(r.data.expires_at ?? null)
        setStatus('authed')
      })
      .catch(clear)
  }, [clear])

  const login = useCallback(async (email: string, password: string) => {
    const res = (await api(EP.AUTH.LOGIN, { method: 'POST', body: { email, password }, schema: Data(LoginRes) })) as { data: { token: string; expires_at: string; admin: Admin } }
    tokenStore.set(res.data.token)
    storage.set(KEY, res.data.token)
    setAdmin(res.data.admin)
    setExpiresAt(res.data.expires_at)
    setStatus('authed')
  }, [])

  const logout = useCallback(async () => {
    try {
      await api(EP.AUTH.LOGOUT, { method: 'POST' })
    } catch {
      /* already out */
    }
    clear()
  }, [clear])

  const value = useMemo(() => ({ status, admin, expiresAt, login, logout }), [status, admin, expiresAt, login, logout])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth outside AuthProvider')
  return ctx
}
