import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { motion } from 'motion/react'
import { toast } from 'sonner'
import { useOrder, useResolveOrder, useCancelOrder } from '../api/use-order'
import { Board, BoardHead, ErrorState, Field, Input, Pass, Skeleton } from '@shared/ui/primitives'
import { Button } from '@shared/ui/Button'
import { Chip, StatusChip } from '@shared/ui/StatusChip'
import { ConfirmTyped, Modal } from '@shared/ui/overlays'
import { ROUTES } from '@shared/utils/routes'
import { wat, relativeTime, titleCase } from '@shared/utils/format'
import { errorMessage } from '@shared/api/errors'
import { Copy, Truck, Store, Phone, MapPin, ExternalLink, Ban, CircleCheck } from '@icons'
import { cn } from '@shared/utils/cn'

const actorTone = { mcp: 'info', reconciler: 'neutral', system: 'neutral' } as const

export default function OrderDetailScreen({ orderNo }: { orderNo: string }) {
  const q = useOrder(orderNo)
  const resolve = useResolveOrder()
  const cancel = useCancelOrder()
  const [resolveOpen, setResolveOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [note, setNote] = useState('')

  if (q.isPending) return <DetailSkeleton />
  if (q.isError) return <Pass><ErrorState error={q.error} onRetry={() => void q.refetch()} /></Pass>
  const o = q.data
  const canResolve = o.status === 'paid_needs_attention'
  const canCancel = o.status === 'pending_payment' || o.status === 'expired'

  const copyRef = () => { if (o.reference) { void navigator.clipboard.writeText(o.reference); toast.success('Reference copied') } }

  return (
    <motion.div initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.28 }} className="space-y-4">
      <nav className="text-[12.5px] text-ink-3"><Link to={ROUTES.ORDERS} className="font-bold text-leaf-700 hover:underline">Orders</Link> / <span className="mono">{o.order_no}</span></nav>

      <div className="grid gap-4 lg:grid-cols-[1fr_380px]">
        <div className="space-y-4">
          {/* PASS: what one person holds */}
          <Pass className={cn(o.status === 'paid_needs_attention' && 'border-critical shadow-[3px_4px_0_#d64545]')}>
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="flex items-center gap-2"><h1 className="mono text-[22px] font-extrabold tnum">{o.order_no}</h1><StatusChip status={o.status} /></div>
                <div className="mt-1 text-[13px] text-ink-3">{o.customer_name} · {relativeTime(o.created_at)} · {o.window}</div>
              </div>
              <div className="text-right"><div className="text-[10px] font-extrabold uppercase tracking-[0.08em] text-ink-4">Total</div><div className="font-display text-[30px] font-extrabold leading-none tnum">{o.total_display}</div></div>
            </div>
            {o.attention_reason ? <div className="mt-3 rounded-[6px] bg-critical-soft px-3 py-2 text-[13px] font-bold text-critical-on">{o.attention_reason}{o.resolution_note ? ` · Resolved: ${o.resolution_note}` : ''}</div> : null}
            <div className="mt-4 grid gap-2 text-[13px] sm:grid-cols-2">
              <div className="flex items-start gap-2">{o.fulfilment === 'delivery' ? <Truck className="mt-0.5 size-4 text-ink-3" /> : <Store className="mt-0.5 size-4 text-ink-3" />}<span>{o.fulfilment === 'delivery' ? <><b>Delivery</b> · {o.zone?.name}</> : <b>Pickup</b>}{o.ready_by_wat ? <span className="text-ink-3"> · ready by <b className="tnum">{o.ready_by_wat}</b></span> : null}</span></div>
              {o.address ? <div className="flex items-start gap-2"><MapPin className="mt-0.5 size-4 text-ink-3" /><span>{o.address}</span></div> : null}
              {o.phone ? <div className="flex items-start gap-2"><Phone className="mt-0.5 size-4 text-ink-3" /><a href={`tel:${o.phone}`} className="font-bold text-leaf-700">{o.phone}</a></div> : null}
              {o.note ? <div className="text-ink-2 sm:col-span-2"><b>Note:</b> {o.note}</div> : null}
            </div>
            {(canResolve || canCancel) ? (
              <div className="mt-4 flex flex-wrap gap-2">
                {canResolve ? <Button icon={<CircleCheck className="size-4" />} onClick={() => setResolveOpen(true)}>Mark resolved</Button> : null}
                {canCancel ? <Button variant="outline" destructive icon={<Ban className="size-4" />} onClick={() => setCancelOpen(true)}>Cancel order</Button> : null}
              </div>
            ) : null}
          </Pass>

          <Board>
            <BoardHead>Lines</BoardHead>
            {o.lines.map((l) => (
              <div key={l.item_id} className="hairline grid grid-cols-[1fr_auto_auto] gap-3 px-3 py-2 text-[13px]"><span><b>{l.qty}×</b> {l.name}</span><span className="tnum text-ink-3">{l.unit_display}</span><span className="tnum font-bold w-20 text-right">{l.line_display}</span></div>
            ))}
            <div className="grid grid-cols-[1fr_auto] gap-3 px-3 py-2 text-[12.5px] text-ink-3"><span>Subtotal</span><span className="tnum">{o.subtotal_display}</span></div>
            <div className="grid grid-cols-[1fr_auto] gap-3 px-3 pb-2 text-[12.5px] text-ink-3"><span>Delivery</span><span className="tnum">{o.delivery_fee_display}</span></div>
            <div className="grid grid-cols-[1fr_auto] gap-3 border-t border-line px-3 py-2 text-[13.5px] font-extrabold"><span>Total</span><span className="tnum">{o.total_display}</span></div>
          </Board>

          <Board>
            <BoardHead>Payment</BoardHead>
            <div className="hairline flex flex-wrap items-center gap-2 px-3 py-2 text-[12.5px]">
              <span className="text-ink-3">Reference</span>
              <span className="mono font-semibold">{o.reference ?? '·'}</span>
              {o.reference ? <button className="grid size-6 place-items-center rounded text-ink-3 hover:bg-paper-2" onClick={copyRef} aria-label="Copy reference"><Copy className="size-3.5" /></button> : null}
              {o.authorization_url && o.status === 'pending_payment' ? <a href={o.authorization_url} target="_blank" rel="noreferrer" className="ml-auto inline-flex items-center gap-1 text-[12px] font-bold text-leaf-700 hover:underline">Open link <ExternalLink className="size-3" /></a> : null}
            </div>
            {o.payment_checks.length === 0 ? <div className="px-3 py-3 text-[12.5px] text-ink-3">Not checked with Paystack yet.</div> : o.payment_checks.map((c, i) => (
              <div key={i} className="hairline grid grid-cols-[52px_1fr_auto] items-center gap-2 px-3 py-2 text-[12.5px]">
                <span className="tnum text-ink-3">{wat(c.checked_at)}</span>
                <span><Chip tone={c.outcome === 'success' ? 'success' : c.outcome === 'failed' || c.outcome === 'amount_mismatch' ? 'critical' : 'caution'}>{titleCase(c.outcome)}</Chip> <span className="text-ink-3">{c.paystack_status ?? ''}{c.gateway_response ? ` · ${c.gateway_response}` : ''}</span></span>
                <span className="tnum text-[11px] text-ink-4">{c.source}{c.latency_ms != null ? ` · ${c.latency_ms} ms` : ''}</span>
              </div>
            ))}
          </Board>
        </div>

        <div className="space-y-4">
          <Board>
            <BoardHead>Timeline</BoardHead>
            <ol className="px-3 py-2">
              {o.events.map((e, i) => (
                <li key={i} className="relative flex gap-3 pb-3 pl-4 text-[12.5px] last:pb-1">
                  <span className="absolute left-0 top-1.5 size-2 rounded-full bg-leaf" aria-hidden />
                  {i < o.events.length - 1 ? <span className="absolute left-[3px] top-3.5 h-full w-px bg-line" aria-hidden /> : null}
                  <span className="font-bold">{titleCase(e.type)}</span>
                  <Chip tone={actorTone[e.actor as keyof typeof actorTone] ?? 'neutral'}>{e.actor.replace('admin:', 'admin ')}</Chip>
                  <span className="ml-auto tnum text-ink-4">{wat(e.at)}</span>
                </li>
              ))}
            </ol>
          </Board>
          <Board>
            <BoardHead>Tool calls</BoardHead>
            {o.tool_calls.length === 0 ? <div className="px-3 py-3 text-[12.5px] text-ink-3">None linked to this order.</div> : o.tool_calls.map((t) => (
              <Link key={t.id} to={ROUTES.ACTIVITY} search={{ order_no: o.order_no }} className="hairline flex items-center gap-2 px-3 py-2 text-[12.5px] hover:bg-paper-2">
                <span className="mono font-semibold">{t.tool}</span>
                <Chip tone={t.ok ? 'success' : 'critical'}>{t.ok ? 'ok' : t.error_code ?? 'error'}</Chip>
                <span className="ml-auto tnum text-[11px] text-ink-4">{t.latency_ms} ms · {wat(t.created_at)}</span>
              </Link>
            ))}
          </Board>
        </div>
      </div>

      <Modal open={resolveOpen} onOpenChange={setResolveOpen} title="Mark as resolved" description="Say what you did so the record is complete."
        footer={<><Button variant="outline" onClick={() => setResolveOpen(false)}>Back</Button><Button loading={resolve.isPending} disabled={note.trim().length < 3} onClick={() => resolve.mutate({ no: o.order_no, text: note.trim() }, { onSuccess: () => { toast.success('Order resolved'); setResolveOpen(false); setNote('') }, onError: (e) => toast.error(errorMessage(e)) })}>Resolve</Button></>}>
        <Field label="Note" hint="For example: called Tobi, swapped ofada for jollof, he agreed."><Input autoFocus value={note} onChange={(e) => setNote(e.target.value)} /></Field>
      </Modal>
      <ConfirmTyped open={cancelOpen} onOpenChange={setCancelOpen} title={`Cancel ${o.order_no}?`} description="The customer's link stops working. This cannot be undone." word="CANCEL" confirmLabel="Cancel order" loading={cancel.isPending}
        onConfirm={() => cancel.mutate({ no: o.order_no, text: 'cancelled by admin' }, { onSuccess: () => { toast.success('Order cancelled'); setCancelOpen(false) }, onError: (e) => toast.error(errorMessage(e)) })} />
    </motion.div>
  )
}

export function DetailSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-4 w-40" />
      <div className="pass p-5 space-y-3"><Skeleton className="h-7 w-48" /><Skeleton className="h-4 w-64" /><Skeleton className="h-10 w-full" /></div>
      <div className="board p-3 space-y-2"><Skeleton className="h-4 w-full" /><Skeleton className="h-4 w-full" /><Skeleton className="h-4 w-2/3" /></div>
    </div>
  )
}
