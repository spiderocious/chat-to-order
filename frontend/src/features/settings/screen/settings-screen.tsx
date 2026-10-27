import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { api, newIdempotencyKey } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { K } from '@shared/api/keys'
import { Data, Vendor, type Vendor as VendorT } from '@shared/api/types'
import { Board, BoardHead, ErrorState, Field, Input, Pass, SectionTitle, Skeleton, Switch } from '@shared/ui/primitives'
import { Button } from '@shared/ui/Button'
import { Chip } from '@shared/ui/StatusChip'
import { errorMessage, isApiError } from '@shared/api/errors'
import { useZones, useUpsertZone } from '@features/menu/api/use-menu'
import { Plus } from '@icons'

function useVendorQ() {
  return useQuery({ queryKey: K.vendor, queryFn: () => api<{ data: VendorT }>(EP.VENDOR, { schema: Data(Vendor) }).then((r) => r.data) })
}

const mins = (s: string) => { const [h, m] = s.split(':').map(Number); return Number.isFinite(h) && Number.isFinite(m) ? (h ?? 0) * 60 + (m ?? 0) : -1 }

export default function SettingsScreen() {
  const qc = useQueryClient()
  const q = useVendorQ()
  const [form, setForm] = useState<VendorT | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})
  useEffect(() => { if (q.data && !form) setForm(q.data) }, [q.data, form])

  const save = useMutation({
    mutationFn: (v: VendorT) => api<{ data: VendorT }>(EP.VENDOR, { method: 'PUT', body: { vendor_name: v.vendor_name, pickup_address: v.pickup_address, rider_phone: v.rider_phone, prep_minutes: v.prep_minutes, pending_ttl_minutes: v.pending_ttl_minutes, low_stock_threshold: v.low_stock_threshold, windows: v.windows }, ifMatch: v.version, idempotencyKey: newIdempotencyKey(), schema: Data(Vendor) }).then((r) => r.data),
    onSuccess: (data) => { qc.setQueryData(K.vendor, data); setForm(data); toast.success('Saved'); void qc.invalidateQueries({ queryKey: K.menu }); void qc.invalidateQueries({ queryKey: K.stats }) },
    onError: (e) => {
      if (isApiError(e) && e.code === 'version_conflict' && e.current) { const cur = Vendor.safeParse(e.current); if (cur.success) { setForm(cur.data); qc.setQueryData(K.vendor, cur.data) } toast.warning('Changed elsewhere. Reloaded the current values.'); return }
      if (isApiError(e) && e.fieldErrors.length) { setErrors(Object.fromEntries(e.fieldErrors.map((f) => [f.field, f.message]))) }
      toast.error(errorMessage(e))
    },
  })

  if (q.isPending || !form) return <div className="space-y-4"><Skeleton className="h-6 w-32" /><div className="pass p-5 space-y-3"><Skeleton className="h-10 w-full" /><Skeleton className="h-10 w-full" /></div></div>
  if (q.isError) return <Pass><ErrorState error={q.error} onRetry={() => void q.refetch()} /></Pass>

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const errs: Record<string, string> = {}
    if (!form.vendor_name.trim()) errs.vendor_name = 'Required'
    if (!form.pickup_address.trim()) errs.pickup_address = 'Required'
    if (form.prep_minutes < 5 || form.prep_minutes > 240) errs.prep_minutes = '5 to 240'
    for (const w of ['lunch', 'evening'] as const) {
      const o = mins(form.windows[w].open), oc = mins(form.windows[w].orders_close), c = mins(form.windows[w].close)
      if (o < 0 || oc < 0 || c < 0) errs[`windows.${w}`] = 'Times must be HH:MM'
      else if (!(o < oc && oc <= c)) errs[`windows.${w}`] = 'Open, then orders close, then kitchen closes'
    }
    if (mins(form.windows.lunch.close) > mins(form.windows.evening.open)) errs['windows.evening'] = 'Evening must open after lunch closes'
    setErrors(errs)
    if (Object.keys(errs).length) return
    save.mutate(form)
  }
  const set = <K extends keyof VendorT>(k: K, v: VendorT[K]) => setForm({ ...form, [k]: v })
  const setW = (w: 'lunch' | 'evening', k: 'open' | 'orders_close' | 'close', v: string) => setForm({ ...form, windows: { ...form.windows, [w]: { ...form.windows[w], [k]: v } } })

  return (
    <form onSubmit={submit} className="space-y-4">
      <SectionTitle aside={<Button type="submit" loading={save.isPending}>Save</Button>}>Settings</SectionTitle>
      <div className="grid gap-4 lg:grid-cols-2">
        <Pass>
          <h3 className="mb-3 text-[15px] font-extrabold">Kitchen</h3>
          <div className="space-y-3">
            <Field label="Name" error={errors.vendor_name}><Input value={form.vendor_name} onChange={(e) => set('vendor_name', e.target.value)} error={!!errors.vendor_name} /></Field>
            <Field label="Pickup address" error={errors.pickup_address}><Input value={form.pickup_address} onChange={(e) => set('pickup_address', e.target.value)} error={!!errors.pickup_address} /></Field>
            <Field label="Rider phone"><Input value={form.rider_phone} onChange={(e) => set('rider_phone', e.target.value)} inputMode="tel" /></Field>
            <div className="grid grid-cols-3 gap-3">
              <Field label="Prep (min)" error={errors.prep_minutes}><Input type="number" min={5} max={240} value={form.prep_minutes} onChange={(e) => set('prep_minutes', Number(e.target.value))} error={!!errors.prep_minutes} /></Field>
              <Field label="Link TTL (min)" error={errors.pending_ttl_minutes}><Input type="number" min={5} max={120} value={form.pending_ttl_minutes} onChange={(e) => set('pending_ttl_minutes', Number(e.target.value))} /></Field>
              <Field label="Low stock at" error={errors.low_stock_threshold}><Input type="number" min={0} max={50} value={form.low_stock_threshold} onChange={(e) => set('low_stock_threshold', Number(e.target.value))} /></Field>
            </div>
            <Field label="Order prefix" hint="Read only; set at launch"><Input readOnly value={form.order_prefix} /></Field>
          </div>
        </Pass>
        <Pass>
          <h3 className="mb-1 text-[15px] font-extrabold">Windows</h3>
          <p className="mb-3 text-[12.5px] text-ink-3">Now: <b>{form.now.wat}</b> · {form.now.window === 'closed' ? 'closed' : `${form.now.window}, orders ${form.now.orders_open ? `open till ${form.now.orders_close_at}` : 'closed'}`}{form.now.next_window ? ` · next ${form.now.next_window.name} at ${form.now.next_window.opens_at}` : ''}</p>
          {(['lunch', 'evening'] as const).map((w) => (
            <div key={w} className="mb-3">
              <div className="mb-1 flex items-center gap-2 text-[12px] font-extrabold uppercase tracking-[0.08em] text-ink-3">{w}{errors[`windows.${w}`] ? <span className="normal-case tracking-normal text-critical">{errors[`windows.${w}`]}</span> : null}</div>
              <div className="grid grid-cols-3 gap-2">
                <Field label="Opens"><Input type="time" value={form.windows[w].open} onChange={(e) => setW(w, 'open', e.target.value)} error={!!errors[`windows.${w}`]} /></Field>
                <Field label="Orders close"><Input type="time" value={form.windows[w].orders_close} onChange={(e) => setW(w, 'orders_close', e.target.value)} error={!!errors[`windows.${w}`]} /></Field>
                <Field label="Kitchen closes"><Input type="time" value={form.windows[w].close} onChange={(e) => setW(w, 'close', e.target.value)} error={!!errors[`windows.${w}`]} /></Field>
              </div>
            </div>
          ))}
        </Pass>
      </div>
      <ZonesBoard />
    </form>
  )
}

function ZonesBoard() {
  const q = useZones()
  const upsert = useUpsertZone()
  const [adding, setAdding] = useState(false)
  const [nz, setNz] = useState({ id: '', name: '', fee_naira: 500 })
  const fire = (fn: () => void) => fn()
  return (
    <Board>
      <BoardHead><div className="flex items-center justify-between"><span>Delivery zones</span><button type="button" className="inline-flex items-center gap-1 normal-case tracking-normal text-leaf-700" onClick={() => setAdding((a) => !a)}><Plus className="size-3.5" /> Add zone</button></div></BoardHead>
      {q.isPending ? <div className="p-3"><Skeleton className="h-4 w-full" /></div> : q.isError ? <ErrorState error={q.error} compact /> : q.data.map((z) => (
        <div key={z.id} className="hairline grid grid-cols-[1fr_110px_auto] items-center gap-3 px-3 py-2 text-[13px]">
          <span><b>{z.name}</b> <span className="mono text-[11px] text-ink-4">{z.id}</span></span>
          <span className="flex items-center gap-1 tnum"><span className="text-ink-3">₦</span><Input type="number" className="h-8 w-24" defaultValue={Math.round(z.fee_kobo / 100)} onBlur={(e) => { const v = Number(e.target.value); if (v !== Math.round(z.fee_kobo / 100)) fire(() => upsert.mutate({ id: z.id, create: false, body: { fee_naira: v } }, { onSuccess: () => toast.success('Zone saved'), onError: (er) => toast.error(errorMessage(er)) })) }} /></span>
          <span className="flex items-center gap-2"><Switch label={`${z.name} active`} checked={z.active} onChange={(v) => upsert.mutate({ id: z.id, create: false, body: { active: v } }, { onError: (er) => toast.error(errorMessage(er)) })} />{z.active ? <Chip tone="success">on</Chip> : <Chip tone="neutral">off</Chip>}</span>
        </div>
      ))}
      {adding ? (
        <div className="grid grid-cols-[1fr_1fr_110px_auto] items-end gap-2 bg-paper-2 px-3 py-2">
          <Field label="Name"><Input className="h-8" value={nz.name} onChange={(e) => setNz({ ...nz, name: e.target.value, id: e.target.value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 24) })} /></Field>
          <Field label="Id"><Input className="h-8 mono" value={nz.id} onChange={(e) => setNz({ ...nz, id: e.target.value })} /></Field>
          <Field label="Fee (₦)"><Input className="h-8" type="number" value={nz.fee_naira} onChange={(e) => setNz({ ...nz, fee_naira: Number(e.target.value) })} /></Field>
          <Button type="button" size="sm" loading={upsert.isPending} onClick={() => upsert.mutate({ id: nz.id, create: true, body: { name: nz.name, fee_naira: nz.fee_naira } }, { onSuccess: () => { toast.success('Zone added'); setAdding(false); setNz({ id: '', name: '', fee_naira: 500 }) }, onError: (er) => toast.error(errorMessage(er)) })}>Add</Button>
        </div>
      ) : null}
    </Board>
  )
}
