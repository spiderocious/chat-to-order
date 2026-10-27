import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { AnimatePresence, motion } from 'motion/react'
import { toast } from 'sonner'
import { useActivity } from '@features/board/api/use-board'
import { Board, BoardHead, EmptyState, ErrorState, RowSkeletons, SectionTitle, Select } from '@shared/ui/primitives'
import { Button } from '@shared/ui/Button'
import { Chip } from '@shared/ui/StatusChip'
import { wat, relativeTime } from '@shared/utils/format'
import { ROUTES } from '@shared/utils/routes'
import { Bot, ChevronDown, Copy } from '@icons'
import { cn } from '@shared/utils/cn'
import type { ToolCall } from '@shared/api/types'

const tools = ['', 'get_menu', 'create_order', 'verify_payment', 'confirm_order'] as const

export default function ActivityScreen({ tool, ok, orderNo }: { tool?: string; ok?: boolean; orderNo?: string }) {
  const navigate = useNavigate()
  const q = useActivity({ tool, ok, order_no: orderNo }, 30)
  const rows = q.data?.pages.flatMap((p) => p.data) ?? []
  const setSearch = (patch: { tool?: string; ok?: string; order_no?: string }) => {
    const okv = patch.ok === undefined ? (ok === undefined ? undefined : ok ? 'true' : 'false') : patch.ok === 'true' ? 'true' : patch.ok === 'false' ? 'false' : undefined
    void navigate({ to: ROUTES.ACTIVITY, search: { tool: patch.tool ?? tool ?? undefined, ok: okv, order_no: patch.order_no ?? orderNo ?? undefined } })
  }

  return (
    <div className="space-y-4">
      <SectionTitle>Assistant activity</SectionTitle>
      <div className="flex flex-wrap items-center gap-2">
        <Select className="h-8 w-44 text-[12.5px]" value={tool ?? ''} onChange={(e) => setSearch({ tool: e.target.value || undefined })}>{tools.map((t) => <option key={t} value={t}>{t || 'All tools'}</option>)}</Select>
        <Select className="h-8 w-36 text-[12.5px]" value={ok === undefined ? '' : String(ok)} onChange={(e) => setSearch({ ok: e.target.value })}><option value="">Any outcome</option><option value="true">OK</option><option value="false">Errors</option></Select>
        {orderNo ? <Chip tone="info">order {orderNo} <button className="ml-1" onClick={() => setSearch({ order_no: '' })} aria-label="clear">×</button></Chip> : null}
      </div>
      <Board>
        <BoardHead><div className="grid grid-cols-[52px_1fr_auto] gap-x-3"><span>Time</span><span>Tool</span><span>Latency</span></div></BoardHead>
        {q.isPending ? <RowSkeletons n={8} /> : q.isError ? <ErrorState error={q.error} onRetry={() => void q.refetch()} /> : rows.length === 0 ? <EmptyState title="No tool calls match" body="When the WhatsApp assistant calls a tool, it shows here with the exact arguments and result." /> : rows.map((t) => <Row key={t.id} t={t} />)}
        {q.hasNextPage ? <div className="flex justify-center p-3"><Button variant="outline" size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>Load more</Button></div> : null}
      </Board>
    </div>
  )
}

function Row({ t }: { t: ToolCall }) {
  const [open, setOpen] = useState(false)
  const copy = (v: unknown) => { void navigator.clipboard.writeText(JSON.stringify(v, null, 2)); toast.success('Copied') }
  return (
    <div className="hairline">
      <button className="grid w-full grid-cols-[52px_1fr_auto] items-center gap-x-3 px-3 py-2 text-left text-[12.5px] hover:bg-paper-2" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        <span className="tnum text-ink-3" title={relativeTime(t.created_at)}>{wat(t.created_at)}</span>
        <span className="flex min-w-0 items-center gap-2"><Bot className="size-4 shrink-0 text-ink-4" /><span className="mono font-semibold">{t.tool}</span><Chip tone={t.ok ? 'success' : 'critical'}>{t.ok ? 'ok' : t.error_code ?? 'error'}</Chip>{t.order_no ? <span className="mono hidden text-[11px] text-ink-3 sm:inline">{t.order_no}</span> : null}</span>
        <span className="flex items-center gap-2 tnum text-[11.5px] text-ink-3">{t.latency_ms} ms <ChevronDown className={cn('size-4 transition-transform duration-180', open && 'rotate-180')} /></span>
      </button>
      <AnimatePresence initial={false}>
        {open ? (
          <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.18 }} className="overflow-hidden">
            <div className="grid gap-2 bg-paper-2 px-3 py-3 md:grid-cols-2">
              {(['args', 'result'] as const).map((k) => (
                <div key={k} className="min-w-0">
                  <div className="mb-1 flex items-center justify-between text-[10.5px] font-extrabold uppercase tracking-[0.08em] text-ink-3">{k}<button className="inline-flex items-center gap-1 normal-case tracking-normal" onClick={() => copy(t[k])}><Copy className="size-3" /> copy</button></div>
                  <pre className="mono max-h-72 overflow-auto rounded-[6px] border border-line bg-card p-2 text-[11px] leading-relaxed">{JSON.stringify(t[k], null, 2)}</pre>
                </div>
              ))}
            </div>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  )
}
