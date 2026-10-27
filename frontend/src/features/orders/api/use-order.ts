import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, newIdempotencyKey } from '@shared/api/client'
import { EP } from '@shared/api/endpoints'
import { K } from '@shared/api/keys'
import { Data, OrderDetail, type OrderDetail as OrderDetailT } from '@shared/api/types'

export const orderQuery = (no: string) =>
  queryOptions({ queryKey: K.order(no), queryFn: () => api<{ data: OrderDetailT }>(EP.ORDERS.DETAIL(no), { schema: Data(OrderDetail) }).then((r) => r.data) })

export function useOrder(no: string) {
  return useQuery(orderQuery(no))
}

function useOrderAction(kind: 'resolve' | 'cancel') {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ no, text }: { no: string; text: string }) =>
      api(kind === 'resolve' ? EP.ORDERS.RESOLVE(no) : EP.ORDERS.CANCEL(no), { method: 'POST', body: kind === 'resolve' ? { note: text } : { reason: text }, idempotencyKey: newIdempotencyKey() }),
    onSuccess: (_d, v) => {
      void qc.invalidateQueries({ queryKey: K.order(v.no) })
      void qc.invalidateQueries({ queryKey: K.ordersAll })
      void qc.invalidateQueries({ queryKey: K.stats })
    },
  })
}

export const useResolveOrder = () => useOrderAction('resolve')
export const useCancelOrder = () => useOrderAction('cancel')
