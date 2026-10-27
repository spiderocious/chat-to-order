import { useEffect, useRef } from 'react'
import { Link } from '@tanstack/react-router'
import { AnimatePresence, motion } from 'motion/react'
import { useStatsToday, useOrders, useActivity } from '../api/use-board'
import { Board, BoardHead, EmptyState, ErrorState, RowSkeletons, SectionTitle, StatTile, StatTileSkeleton, Flip } from '@shared/ui/primitives'
import { OrderRowItem } from '@features/orders/widgets/OrderRow'
import { Chip } from '@shared/ui/StatusChip'
import { ROUTES } from '@shared/utils/routes'
import { cn } from '@shared/utils/cn'
import { Bot, CircleCheck, CircleAlert } from '@icons'
import { relativeTime } from '@shared/utils/format'

const stagger = { hidden: { opacity: 0, y: 8 }, show: (i: number) => ({ opacity: 1, y: 0, transition: { duration: 0.28, delay: i * 0.035, ease: [0.22, 0.61, 0.36, 1] as const } }) }

export default function BoardScreen() {
  const stats = useStatsToday()
  const orders = useOrders({ today: true }, 50)
  const activity = useActivity({}, 6)
  const rows = orders.data?.pages.flatMap((p) => p.data) ?? []
  const seen = useRef<Set<string>>(new Set())
  const first = useRef(true)
  useEffect(() => {
    if (rows.length && first.current) { rows.forEach((r) => seen.current.add(r.order_no)); first.current = false }
  }, [rows])
  const sorted = [...rows].sort((a, b) => (a.status === 'paid_needs_attention' ? -1 : 0) - (b.status === 'paid_needs_attention' ? -1 : 0))

  return (
    <div className="min-w-0 space-y-5">
      <motion.section custom={0} variants={stagger} initial="hidden" animate="show">
        <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
          {stats.isPending ? <><StatTileSkeleton /><StatTileSkeleton /><StatTileSkeleton /><StatTileSkeleton /></> : stats.isError ? (
            <div className="col-span-2 sm:col-span-4 pass-xs"><ErrorState error={stats.error} onRetry={() => void stats.refetch()} compact /></div>
          ) : (
            <>
              <StatTile label="Paid today" value={stats.data.paid_count} hint={`${stats.data.cancelled_count + stats.data.expired_count} cancelled or expired`} />
              <StatTile label="Taken" value={stats.data.paid_total_display} tone="success" />
              <StatTile label="Awaiting pay" value={stats.data.awaiting_payment_count} tone={stats.data.awaiting_payment_count > 0 ? 'caution' : 'neutral'} />
              <StatTile label="Needs you" value={stats.data.needs_attention_count} tone={stats.data.needs_attention_count > 0 ? 'critical' : 'neutral'} hint={stats.data.needs_attention_count > 0 ? 'Paid, but an item ran out' : 'Nothing waiting'} />
            </>
          )}
        </div>
      </motion.section>

      <motion.section custom={1} variants={stagger} initial="hidden" animate="show">
        <SectionTitle aside={<Link to={ROUTES.ORDERS} className="text-[12.5px] font-extrabold text-leaf-700 hover:underline">All orders</Link>}>Orders today</SectionTitle>
        <Board>
          <BoardHead><div className="hidden sm:grid grid-cols-[112px_96px_1fr_110px_64px_16px] gap-x-3"><span>Status</span><span>Order</span><span>Customer · lines</span><span className="text-right">Total</span><span className="text-right">Ready</span><span /></div><span className="sm:hidden">Today</span></BoardHead>
          {orders.isPending ? <RowSkeletons n={5} /> : orders.isError ? <ErrorState error={orders.error} onRetry={() => void orders.refetch()} /> : rows.length === 0 ? (
            <EmptyState title="No orders yet today" body="They'll appear here the moment a customer pays." />
          ) : (
            <AnimatePresence initial={false}>
              {sorted.map((o) => {
                const isNew = !seen.current.has(o.order_no)
                if (isNew) seen.current.add(o.order_no)
                return <OrderRowItem key={o.order_no} o={o} isNew={isNew} />
              })}
            </AnimatePresence>
          )}
        </Board>
      </motion.section>

      <div className="grid min-w-0 gap-5 lg:grid-cols-[1fr_380px]">
        <motion.section className="min-w-0" custom={2} variants={stagger} initial="hidden" animate="show">
          <SectionTitle aside={<Link to={ROUTES.MENU} className="text-[12.5px] font-extrabold text-leaf-700 hover:underline">Edit menu</Link>}>Stock</SectionTitle>
          <div className="flex gap-2 overflow-x-auto pb-1 [-webkit-overflow-scrolling:touch]">
            {stats.data?.stock.map((s) => (
              <Link key={s.id} to={ROUTES.MENU} search={{ edit: s.id }} className={cn('pass-xs flex shrink-0 items-center gap-2 px-3 py-2 text-[12.5px] transition-colors', !s.offered_now && 'opacity-60')}>
                <span className="font-bold">{s.name}</span>
                <span className={cn('tnum font-display text-[16px] font-extrabold', s.portions === 0 ? 'text-critical' : s.low ? 'text-caution-on' : 'text-leaf-700')}><Flip value={s.portions} /></span>
                {s.portions === 0 ? <Chip tone="critical">Sold out</Chip> : s.low ? <Chip tone="caution">Low</Chip> : null}
              </Link>
            ))}
          </div>
        </motion.section>

        <motion.section className="min-w-0" custom={3} variants={stagger} initial="hidden" animate="show">
          <SectionTitle aside={<Link to={ROUTES.ACTIVITY} className="text-[12.5px] font-extrabold text-leaf-700 hover:underline">See all</Link>}>Assistant activity</SectionTitle>
          <Board>
            {activity.isPending ? <RowSkeletons n={4} /> : activity.isError ? <ErrorState error={activity.error} compact /> : (activity.data.pages[0]?.data.length ?? 0) === 0 ? (
              <EmptyState title="No tool calls yet" body="When the WhatsApp assistant works, its calls show here." />
            ) : (
              activity.data.pages[0]?.data.map((t) => (
                <Link key={t.id} to={ROUTES.ACTIVITY} className="hairline flex items-center gap-2.5 px-3 py-2 text-[12.5px] hover:bg-paper-2">
                  <Bot className="size-4 text-ink-4" />
                  <span className="mono font-semibold">{t.tool}</span>
                  {t.ok ? <CircleCheck className="size-3.5 text-leaf" /> : <span className="inline-flex items-center gap-1 text-critical"><CircleAlert className="size-3.5" /><span className="mono text-[11px]">{t.error_code}</span></span>}
                  <span className="ml-auto tnum text-[11px] text-ink-3">{t.latency_ms} ms · {relativeTime(t.created_at)}</span>
                </Link>
              ))
            )}
          </Board>
        </motion.section>
      </div>
    </div>
  )
}
