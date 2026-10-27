import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useAuth } from '@features/auth/providers/auth-provider'
import { LiveProvider } from '@features/live/providers/live-provider'
import { AppShell } from '@app/shell/AppShell'
import { ROUTES } from '@shared/utils/routes'
import { Loader2 } from '@icons'

export const Route = createFileRoute('/_authed')({ component: Authed })

function Authed() {
  const { status } = useAuth()
  const navigate = useNavigate()
  useEffect(() => {
    if (status === 'anon') {
      const next = window.location.pathname + window.location.search
      void navigate({ to: ROUTES.LOGIN, search: { next: next === '/' ? undefined : next }, replace: true })
    }
  }, [status, navigate])
  if (status !== 'authed') {
    return (
      <div className="grid min-h-dvh place-items-center text-ink-3"><Loader2 className="size-6 animate-spin" aria-label="Loading" /></div>
    )
  }
  return (
    <LiveProvider>
      <AppShell><Outlet /></AppShell>
    </LiveProvider>
  )
}
