import { useNavigate } from '@tanstack/react-router'
import { AnimatePresence } from 'motion/react'
import { useOrders } from '@features/board/api/use-board'
import { Board, BoardHead, EmptyState, ErrorState, RowSkeletons, SectionTitle } from '@shared/ui/primitives'
import { Button } from '@shared/ui/Button'
import { OrderRowItem } from '@features/orders/widgets/OrderRow'
import { cn } from '@shared/utils/cn'
import { ROUTES } from '@shared/utils/routes'

export const filters = [
  { key: '', label: 'All' },
  { key: 'paid,resolved', label: 'Paid' },
  { key: 'pending_payment', label: 'Awaiting' },
  { key: 'paid_needs_attention', label: 'Needs you' },
  { key: 'cancelled,expired', label: 'Cancelled' },
] as const

export default function OrdersScreen({ status }: { status: string }) {
  const navigate = useNavigate()
  const q = useOrders({ status: status || undefined }, 50)
  const rows = q.data?.pages.flatMap((p) => p.data) ?? []
  const empty = { '': 'No orders yet.', 'paid,resolved': 'No paid orders yet.', pending_payment: 'Nobody is mid-payment right now.', paid_needs_attention: 'No orders need you right now.', 'cancelled,expired': 'Nothing cancelled or expired.' }[status] ?? 'No orders match.'
  return (
    <div className="space-y-4">
      <SectionTitle>Orders</SectionTitle>
      <div className="flex gap-1.5 overflow-x-auto pb-1" role="tablist" aria-label="Filter orders">
        {filters.map((f) => (
          <button key={f.key} role="tab" aria-selected={status === f.key} onClick={() => void navigate({ to: ROUTES.ORDERS, search: { status: f.key || undefined } })}
            className={cn('h-8 shrink-0 rounded-full border px-3 text-[12.5px] font-extrabold transition-colors duration-100', status === f.key ? 'border-ink bg-leaf text-white' : 'border-line-2 bg-card text-ink-2 hover:bg-paper-2')}>
            {f.label}
          </button>
        ))}
      </div>
      <Board>
        <BoardHead><div className="hidden sm:grid grid-cols-[112px_96px_1fr_110px_64px_16px] gap-x-3"><span>Status</span><span>Order</span><span>Customer · lines</span><span className="text-right">Total</span><span className="text-right">Ready</span><span /></div><span className="sm:hidden">Newest first</span></BoardHead>
        {q.isPending ? <RowSkeletons n={8} /> : q.isError ? <ErrorState error={q.error} onRetry={() => void q.refetch()} /> : rows.length === 0 ? <EmptyState title={empty} /> : (
          <AnimatePresence initial={false}>{rows.map((o) => <OrderRowItem key={o.order_no} o={o} />)}</AnimatePresence>
        )}
        {q.hasNextPage ? <div className="flex justify-center p-3"><Button variant="outline" size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>Load more</Button></div> : null}
      </Board>
    </div>
  )
}
