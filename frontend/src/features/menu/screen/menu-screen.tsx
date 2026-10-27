import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'
import { useMenu, useUpdateItem, useCreateItem, type ItemPatch } from '../api/use-menu'
import { Board, BoardHead, EmptyState, ErrorState, Field, Input, RowSkeletons, SectionTitle, Select, Stepper, Switch } from '@shared/ui/primitives'
import { Button } from '@shared/ui/Button'
import { Chip } from '@shared/ui/StatusChip'
import { Drawer } from '@shared/ui/overlays'
import { errorMessage, isApiError } from '@shared/api/errors'
import { Pencil, Plus } from '@icons'
import { ROUTES } from '@shared/utils/routes'
import { cn } from '@shared/utils/cn'
import type { Item } from '@shared/api/types'

const windows = ['lunch', 'evening', 'all'] as const

export default function MenuScreen({ edit }: { edit?: string }) {
  const navigate = useNavigate()
  const q = useMenu()
  const update = useUpdateItem()
  const create = useCreateItem()
  const [creating, setCreating] = useState(false)
  const editing = q.data?.items.find((i) => i.id === edit)

  const closeEditor = () => void navigate({ to: ROUTES.MENU, search: {} })

  const patch = (it: Item, p: ItemPatch) =>
    update.mutate({ id: it.id, version: it.version, patch: p }, {
      onError: (e) => toast.error(isApiError(e) && e.code === 'version_conflict' ? 'Changed elsewhere. Reloaded the current values.' : errorMessage(e)),
    })

  return (
    <div className="space-y-4">
      <SectionTitle aside={<Button size="sm" icon={<Plus className="size-4" />} onClick={() => setCreating(true)}>Add item</Button>}>Menu</SectionTitle>
      <Board>
        <BoardHead><div className="hidden sm:grid grid-cols-[1fr_80px_90px_120px_96px_60px_32px] gap-x-3"><span>Item</span><span>Window</span><span className="text-right">Price</span><span className="text-center">Portions</span><span className="text-center">Available</span><span>Now</span><span /></div><span className="sm:hidden">Tap an item to edit</span></BoardHead>
        {q.isPending ? <RowSkeletons n={8} /> : q.isError ? <ErrorState error={q.error} onRetry={() => void q.refetch()} /> : q.data.items.length === 0 ? (
          <EmptyState title="No items yet" body="Add your first dish." action={<Button size="sm" onClick={() => setCreating(true)}>Add item</Button>} />
        ) : q.data.items.map((it) => (
          <div key={it.id} className={cn('hairline grid grid-cols-[1fr_auto] items-center gap-x-3 gap-y-2 px-3 py-2.5 text-[13px] sm:grid-cols-[1fr_72px_84px_180px_88px_56px_32px]', !it.available && 'opacity-70')}>
            <button className="min-w-0 text-left" onClick={() => void navigate({ to: ROUTES.MENU, search: { edit: it.id } })}>
              <span className="block truncate font-extrabold">{it.name}</span>
              <span className="block truncate text-[11.5px] text-ink-3">{it.description || it.id}</span>
            </button>
            <span className="hidden sm:inline text-ink-2">{it.window}</span>
            <span className="tnum text-right font-bold">{it.price_display}</span>
            <div className="flex flex-wrap items-center justify-center gap-1.5 col-span-2 sm:col-auto">
              <Stepper label={`${it.name} portions`} value={it.portions} onChange={(v) => patch(it, { portions: v })} />
              {it.portions === 0 ? <Chip tone="critical">Sold out</Chip> : it.low ? <Chip tone="caution">Low</Chip> : null}
            </div>
            <div className="flex justify-center"><Switch label={`${it.name} available`} checked={it.available} onChange={(v) => patch(it, { available: v })} /></div>
            <span className="hidden sm:inline">{it.offered_now ? <Chip tone="success">on</Chip> : <Chip tone="neutral">off</Chip>}</span>
            <button className="hidden sm:grid size-8 place-items-center rounded-md text-ink-3 hover:bg-paper-2" onClick={() => void navigate({ to: ROUTES.MENU, search: { edit: it.id } })} aria-label={`Edit ${it.name}`}><Pencil className="size-4" /></button>
          </div>
        ))}
      </Board>

      <ItemEditor open={!!editing} item={editing} onClose={closeEditor} onSave={(p) => editing && update.mutate({ id: editing.id, version: editing.version, patch: p }, { onSuccess: () => { toast.success('Saved'); closeEditor() }, onError: (e) => toast.error(errorMessage(e)) })} busy={update.isPending} />
      <ItemEditor open={creating} onClose={() => setCreating(false)} create onSave={(p) => create.mutate(p as ItemPatch & { id: string }, { onSuccess: () => { toast.success('Item added'); setCreating(false) }, onError: (e) => toast.error(errorMessage(e)) })} busy={create.isPending} />
    </div>
  )
}

function slug(s: string): string { return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 24) }

function ItemEditor({ open, item, create = false, onClose, onSave, busy }: { open: boolean; item?: Item; create?: boolean; onClose: () => void; onSave: (p: ItemPatch & { id?: string }) => void; busy: boolean }) {
  const [form, setForm] = useState({ id: '', name: '', description: '', price_naira: 0, portions: 0, available: true, window: 'all' as (typeof windows)[number], sort: 100 })
  const [errors, setErrors] = useState<Record<string, string>>({})
  useEffect(() => {
    if (!open) return
    setErrors({})
    if (item) setForm({ id: item.id, name: item.name, description: item.description, price_naira: Math.round(item.price_kobo / 100), portions: item.portions, available: item.available, window: item.window, sort: item.sort })
    else setForm({ id: '', name: '', description: '', price_naira: 0, portions: 10, available: true, window: 'all', sort: 100 })
  }, [open, item])

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const errs: Record<string, string> = {}
    if (form.name.trim().length < 1 || form.name.length > 80) errs.name = '1 to 80 characters'
    if (!Number.isInteger(form.price_naira) || form.price_naira < 1) errs.price_naira = 'Whole naira, at least 1'
    if (form.portions < 0) errs.portions = 'Cannot be negative'
    if (create && !/^[a-z0-9-]{2,24}$/.test(form.id)) errs.id = '2 to 24 lowercase letters, digits or hyphens'
    setErrors(errs)
    if (Object.keys(errs).length) return
    const p: ItemPatch & { id?: string } = { name: form.name.trim(), description: form.description, price_naira: form.price_naira, portions: form.portions, available: form.available, window: form.window, sort: form.sort }
    if (create) p.id = form.id
    onSave(p)
  }

  return (
    <Drawer open={open} onOpenChange={(o) => { if (!o) onClose() }} title={create ? 'New item' : item?.name ?? 'Edit item'} description={create ? 'The id is what the assistant uses; it is made from the name.' : `id ${item?.id}`}
      footer={<><Button variant="outline" onClick={onClose}>Cancel</Button><Button form="item-form" type="submit" loading={busy}>{create ? 'Add item' : 'Save'}</Button></>}>
      <form id="item-form" onSubmit={submit} className="space-y-3">
        <Field label="Name" error={errors.name}><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value, id: create ? slug(e.target.value) : form.id })} error={!!errors.name} /></Field>
        {create ? <Field label="Id" error={errors.id} hint="Lowercase, used in tool calls"><Input className="mono" value={form.id} onChange={(e) => setForm({ ...form, id: e.target.value })} error={!!errors.id} /></Field> : null}
        <Field label="Description"><Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} maxLength={200} /></Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Price (₦)" error={errors.price_naira}><Input type="number" inputMode="numeric" min={1} step={1} value={form.price_naira} onChange={(e) => setForm({ ...form, price_naira: Number(e.target.value) })} error={!!errors.price_naira} /></Field>
          <Field label="Window"><Select value={form.window} onChange={(e) => setForm({ ...form, window: e.target.value as (typeof windows)[number] })}>{windows.map((w) => <option key={w} value={w}>{w}</option>)}</Select></Field>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Portions" error={errors.portions}><div><Stepper label="portions" value={form.portions} onChange={(v) => setForm({ ...form, portions: v })} /></div></Field>
          <Field label="Available"><div className="h-10 flex items-center"><Switch label="available" checked={form.available} onChange={(v) => setForm({ ...form, available: v })} /></div></Field>
        </div>
        <Field label="Sort" hint="Lower comes first"><Input type="number" value={form.sort} onChange={(e) => setForm({ ...form, sort: Number(e.target.value) })} /></Field>
      </form>
    </Drawer>
  )
}
