import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { api } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { K, type ActivityFilter, type OrdersFilter } from '@shared/api/keys'
import { Data, OrderRow, Page, Stats, ToolCall, type OrderRow as OrderRowT, type Stats as StatsT, type ToolCall as ToolCallT } from '@shared/api/types'
import { usePollInterval } from '@features/live/providers/live-provider'

type PageOf<T> = { data: T[]; meta: { next_cursor: string | null; has_more: boolean } }

export function useStatsToday() {
  const refetchInterval = usePollInterval()
  return useQuery({
    queryKey: K.stats,
    queryFn: () => api<{ data: StatsT }>(EP.STATS, { schema: Data(Stats) }).then((r) => r.data),
    refetchInterval,
  })
}

export function useOrders(filter: OrdersFilter, limit = 50) {
  const refetchInterval = usePollInterval()
  return useInfiniteQuery({
    queryKey: K.orders(filter),
    queryFn: ({ pageParam }) =>
      api<PageOf<OrderRowT>>(EP.ORDERS.LIST, { schema: Page(OrderRow), query: { status: filter.status, today: filter.today ? 'true' : undefined, cursor: pageParam || undefined, limit } }),
    initialPageParam: '',
    getNextPageParam: (last) => (last.meta.has_more && last.meta.next_cursor ? last.meta.next_cursor : undefined),
    refetchInterval,
  })
}

export function useActivity(filter: ActivityFilter, limit = 30) {
  const refetchInterval = usePollInterval()
  return useInfiniteQuery({
    queryKey: K.activity(filter),
    queryFn: ({ pageParam }) =>
      api<PageOf<ToolCallT>>(EP.ACTIVITY, { schema: Page(ToolCall), query: { tool: filter.tool, ok: filter.ok === undefined ? undefined : String(filter.ok), order_no: filter.order_no, cursor: pageParam || undefined, limit } }),
    initialPageParam: '',
    getNextPageParam: (last) => (last.meta.has_more && last.meta.next_cursor ? last.meta.next_cursor : undefined),
    refetchInterval,
  })
}
