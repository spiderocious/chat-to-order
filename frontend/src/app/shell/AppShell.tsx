import { useState, type ReactNode } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import { motion } from 'motion/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { cn } from '@shared/utils/cn'
import { ROUTES } from '@shared/utils/routes'
import { LayoutDashboard, ReceiptText, UtensilsCrossed, Settings, Activity, LogOut, RotateCcw, ChevronDown } from '@icons'
import { useAuth } from '@features/auth/providers/auth-provider'
import { LiveChip } from '@features/live/widgets/LiveChip'
import { api } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { K } from '@shared/api/keys'
import { Data, Vendor } from '@shared/api/types'
import { Chip } from '@shared/ui/StatusChip'
import { ConfirmTyped } from '@shared/ui/overlays'
import { errorMessage } from '@shared/api/errors'

const nav = [
  { to: ROUTES.BOARD, label: 'Board', icon: LayoutDashboard, group: 'Today', exact: true },
  { to: ROUTES.ORDERS, label: 'Orders', icon: ReceiptText, group: 'Today' },
  { to: ROUTES.MENU, label: 'Menu', icon: UtensilsCrossed, group: 'Setup' },
  { to: ROUTES.SETTINGS, label: 'Settings', icon: Settings, group: 'Setup' },
  { to: ROUTES.ACTIVITY, label: 'Activity', icon: Activity, group: 'System' },
] as const

function useVendor() {
  return useQuery({ queryKey: K.vendor, queryFn: () => api<{ data: Vendor }>(EP.VENDOR, { schema: Data(Vendor) }).then((r) => r.data), staleTime: 30_000 })
}

function WindowChip() {
  const { data } = useVendor()
  if (!data) return null
  const now = data.now
  if (now.window === 'closed') return <Chip tone="neutral">Closed{now.next_window ? ` · ${now.next_window.name} opens ${now.next_window.opens_at}` : ''}</Chip>
  if (!now.orders_open) return <Chip tone="caution">{now.window} · orders closed{now.next_window ? `, ${now.next_window.name} at ${now.next_window.opens_at}` : ''}</Chip>
  return <Chip tone="success">{now.window} · orders close {now.orders_close_at}</Chip>
}

function ResetButton() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const m = useMutation({
    mutationFn: () => api(EP.RESET, { method: 'POST' }),
    onSuccess: () => { toast.success('Demo reset'); setOpen(false); void qc.invalidateQueries() },
    onError: (e) => toast.error(errorMessage(e)),
  })
  if (import.meta.env.VITE_DEMO !== '1') return null
  return (
    <>
      <button className="hidden sm:inline-flex h-7 items-center gap-1 rounded-full border border-line-2 bg-card px-2.5 text-[11px] font-extrabold text-ink-2 hover:bg-paper-2" onClick={() => setOpen(true)}>
        <RotateCcw className="size-3" /> Reset demo
      </button>
      <ConfirmTyped open={open} onOpenChange={setOpen} title="Reset the demo data?" description="Clears every order and tool call, restores seed stock and re-adds the six demo orders. Settings stay." word="RESET" confirmLabel="Reset" onConfirm={() => m.mutate()} loading={m.isPending} />
    </>
  )
}

function AccountMenu() {
  const { admin, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const initials = (admin?.name ?? 'IB').split(' ').map((s) => s[0]).join('').slice(0, 2).toUpperCase()
  return (
    <div className="relative">
      <button className="flex h-8 items-center gap-1.5 rounded-full border border-line-2 bg-card pl-1 pr-2 text-[12px] font-extrabold" onClick={() => setOpen((o) => !o)} aria-haspopup="menu" aria-expanded={open}>
        <span className="grid size-6 place-items-center rounded-full bg-leaf text-[10px] text-white">{initials}</span>
        <ChevronDown className="size-3.5 text-ink-3" />
      </button>
      {open ? (
        <div role="menu" className="absolute right-0 mt-1 w-52 pass-xs p-1 z-[600]" onMouseLeave={() => setOpen(false)}>
          <div className="px-3 py-2 text-[12px]"><div className="font-extrabold">{admin?.name}</div><div className="text-ink-3 truncate">{admin?.email}</div></div>
          <div className="px-3 pb-1 text-[10.5px] text-ink-4">v{import.meta.env.VITE_APP_VERSION ?? 'dev'}</div>
          <button role="menuitem" className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-[13px] font-bold hover:bg-paper-2" onClick={() => void logout()}><LogOut className="size-4" /> Sign out</button>
        </div>
      ) : null}
    </div>
  )
}

export function AppShell({ children }: { children: ReactNode }) {
  const { data: vendor } = useVendor()
  const path = useRouterState({ select: (s) => s.location.pathname })
  const isActive = (to: string, exact?: boolean) => (exact ? path === to : path.startsWith(to))

  return (
    <div className="dvh flex bg-paper">
      {/* rail */}
      <aside className="hidden lg:flex w-[220px] shrink-0 flex-col border-r border-line-2 bg-paper-2 px-3 py-4">
        <Link to={ROUTES.BOARD} className="flex items-center gap-2.5 px-2">
          <span className="grid size-8 place-items-center rounded-[9px] bg-leaf font-display text-[14px] font-extrabold text-white">IB</span>
          <span className="leading-tight"><span className="block font-display text-[14px] font-extrabold">{vendor?.vendor_name ?? 'Iya Bisi'}</span><span className="block text-[9.5px] font-extrabold uppercase tracking-[0.1em] text-ink-3">Orders</span></span>
        </Link>
        <nav className="mt-4 flex flex-col">
          {['Today', 'Setup', 'System'].map((g) => (
            <div key={g}>
              <div className="mt-3 mb-1 px-2 text-[9.5px] font-extrabold uppercase tracking-[0.12em] text-ink-4">{g}</div>
              {nav.filter((n) => n.group === g).map((n) => {
                const active = isActive(n.to, 'exact' in n && n.exact)
                return (
                  <Link key={n.to} to={n.to} className={cn('relative flex items-center gap-2.5 rounded-[8px] px-2.5 py-2 text-[13px] font-bold transition-colors duration-100', active ? 'text-white' : 'text-ink-2 hover:bg-paper-3')}>
                    {active ? <motion.span layoutId="rail-active" className="absolute inset-0 rounded-[8px] bg-leaf" transition={{ duration: 0.18 }} /> : null}
                    <n.icon className="relative size-4" /><span className="relative">{n.label}</span>
                  </Link>
                )
              })}
            </div>
          ))}
        </nav>
        <div className="mt-auto px-2 text-[11px] text-ink-4">Light mode · {new Date().getFullYear()}</div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        {/* topbar */}
        <header className="sticky top-0 z-[200] flex h-14 items-center gap-2 border-b border-line bg-paper/90 px-4 backdrop-blur sm:px-5">
          <Link to={ROUTES.BOARD} className="lg:hidden grid size-8 place-items-center rounded-[9px] bg-leaf font-display text-[13px] font-extrabold text-white">IB</Link>
          <div className="hidden sm:block"><WindowChip /></div>
          <div className="ml-auto flex items-center gap-2">
            <LiveChip compact />
            <ResetButton />
            <AccountMenu />
          </div>
        </header>
        <div className="sm:hidden px-4 pt-3"><WindowChip /></div>

        <main className="mx-auto w-full max-w-[1180px] flex-1 px-4 pb-24 pt-4 sm:px-5 lg:pb-8">{children}</main>

        {/* bottom tab bar on phones and tablets */}
        <nav className="lg:hidden fixed inset-x-0 bottom-0 z-[200] border-t border-line-2 bg-card safe-bottom">
          <div className="grid grid-cols-5">
            {nav.map((n) => {
              const active = isActive(n.to, 'exact' in n && n.exact)
              return (
                <Link key={n.to} to={n.to} className={cn('relative flex flex-col items-center gap-0.5 py-2 text-[10.5px] font-extrabold', active ? 'text-leaf-700' : 'text-ink-3')}>
                  {active ? <motion.span layoutId="tab-active" className="absolute top-0 h-0.5 w-10 rounded-full bg-leaf" transition={{ duration: 0.18 }} /> : null}
                  <n.icon className="size-5" />{n.label}
                </Link>
              )
            })}
          </div>
        </nav>
      </div>
    </div>
  )
}
