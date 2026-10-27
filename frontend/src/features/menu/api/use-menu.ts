import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, newIdempotencyKey } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { K } from '@shared/api/keys'
import { Data, Item, MenuRes, Zone, ZonesRes, type Item as ItemT, type NowView, type Zone as ZoneT } from '@shared/api/types'
import { isApiError } from '@shared/api/errors'

type MenuData = { items: ItemT[]; now: NowView }

export function useMenu() {
  return useQuery({ queryKey: K.menu, queryFn: () => api<{ data: MenuData }>(EP.MENU.LIST, { schema: Data(MenuRes) }).then((r) => r.data) })
}

export function useZones() {
  return useQuery({ queryKey: K.zones, queryFn: () => api<{ data: { zones: ZoneT[] } }>(EP.ZONES.LIST, { schema: Data(ZonesRes) }).then((r) => r.data.zones) })
}

export interface ItemPatch { name?: string; description?: string; price_naira?: number; portions?: number; available?: boolean; window?: 'lunch' | 'evening' | 'all'; sort?: number }

/** Optimistic for portions and available; rolls back and reloads on 409. */
export function useUpdateItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, version, patch }: { id: string; version: number; patch: ItemPatch }) =>
      api<{ data: ItemT }>(EP.MENU.ITEM(id), { method: 'PATCH', body: patch, ifMatch: version, idempotencyKey: newIdempotencyKey(), schema: Data(Item) }).then((r) => r.data),
    onMutate: async ({ id, patch }) => {
      await qc.cancelQueries({ queryKey: K.menu })
      const prev = qc.getQueryData<MenuData>(K.menu)
      if (prev) qc.setQueryData<MenuData>(K.menu, { ...prev, items: prev.items.map((it) => (it.id === id ? { ...it, ...('price_naira' in patch ? {} : patch) } : it)) })
      return { prev }
    },
    onError: (err, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(K.menu, ctx.prev)
      if (isApiError(err) && err.code === 'version_conflict') void qc.invalidateQueries({ queryKey: K.menu })
    },
    onSettled: () => { void qc.invalidateQueries({ queryKey: K.menu }); void qc.invalidateQueries({ queryKey: K.stats }) },
  })
}

export function useCreateItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: ItemPatch & { id: string }) => api<{ data: ItemT }>(EP.MENU.LIST, { method: 'POST', body, idempotencyKey: newIdempotencyKey(), schema: Data(Item) }).then((r) => r.data),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: K.menu }); void qc.invalidateQueries({ queryKey: K.stats }) },
  })
}

export function useUpsertZone() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, create, body }: { id: string; create: boolean; body: { name?: string; fee_naira?: number; active?: boolean; sort?: number } }) =>
      create
        ? api<{ data: ZoneT }>(EP.ZONES.LIST, { method: 'POST', body: { id, ...body }, idempotencyKey: newIdempotencyKey(), schema: Data(Zone) }).then((r) => r.data)
        : api<{ data: ZoneT }>(EP.ZONES.ZONE(id), { method: 'PATCH', body, idempotencyKey: newIdempotencyKey(), schema: Data(Zone) }).then((r) => r.data),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: K.zones }); void qc.invalidateQueries({ queryKey: K.menu }) },
  })
}
