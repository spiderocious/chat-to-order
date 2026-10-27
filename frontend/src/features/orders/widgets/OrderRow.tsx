import { Link } from '@tanstack/react-router'
import { motion } from 'motion/react'
import { cn } from '@shared/utils/cn'
import { ROUTES } from '@shared/utils/routes'
import { relativeTime } from '@shared/utils/format'
import { StatusChip } from '@shared/ui/StatusChip'
import { Truck, Store, ChevronRight } from '@icons'
import type { OrderRow as OrderRowT } from '@shared/api/types'

export function OrderRowItem({ o, isNew = false }: { o: OrderRowT; isNew?: boolean }) {
  const attention = o.status === 'paid_needs_attention'
  return (
    <motion.div layout initial={isNew ? { opacity: 0, height: 0, backgroundColor: '#dff1e6' } : false} animate={{ opacity: 1, height: 'auto', backgroundColor: '#ffffff' }}
      transition={{ duration: 0.28, ease: [0.22, 0.61, 0.36, 1], backgroundColor: { duration: 1.2 } }} className="overflow-hidden">
      <Link to={ROUTES.ORDER} params={{ orderNo: o.order_no }}
        className={cn('hairline group grid grid-cols-[auto_1fr_auto] items-center gap-x-3 gap-y-1 px-3 py-2.5 transition-colors duration-100 hover:bg-paper-2 sm:grid-cols-[112px_96px_1fr_110px_64px_16px]', attention && 'border-l-[3px] border-l-critical')}>
        <StatusChip status={o.status} />
        <span className="mono text-[12.5px] font-semibold tnum hidden sm:inline">{o.order_no}</span>
        <span className="min-w-0 truncate text-[13px] col-span-1 sm:col-auto">
          <span className="font-extrabold">{o.customer_name}</span>
          <span className="text-ink-3"> · {o.lines_summary}</span>
          <span className="text-ink-4 hidden md:inline"> · {o.fulfilment === 'delivery' ? o.zone?.name ?? 'delivery' : 'pickup'}</span>
        </span>
        <span className="tnum text-right text-[13px] font-extrabold">{o.total_display}</span>
        <span className="col-span-2 flex items-center gap-1.5 text-[11.5px] text-ink-3 sm:col-auto sm:justify-end">
          <span className="sm:hidden mono">{o.order_no}</span>
          {o.fulfilment === 'delivery' ? <Truck className="size-3.5" aria-label="delivery" /> : <Store className="size-3.5" aria-label="pickup" />}
          <span className="tnum">{o.ready_by_wat ? `ready ${o.ready_by_wat}` : relativeTime(o.created_at)}</span>
        </span>
        <ChevronRight className="hidden size-4 text-ink-4 transition-transform duration-100 group-hover:translate-x-0.5 sm:block" />
        {attention && o.attention_reason ? <span className="col-span-3 sm:col-span-6 text-[12px] font-bold text-critical-on">{o.attention_reason}</span> : null}
      </Link>
    </motion.div>
  )
}
