import { createFileRoute, Link } from '@tanstack/react-router'
import OrderDetailScreen, { DetailSkeleton } from '@features/orders/screen/order-detail-screen'
import { orderQuery } from '@features/orders/api/use-order'
import { ErrorState, Pass } from '@shared/ui/primitives'
import { ROUTES } from '@shared/utils/routes'
import { isApiError } from '@shared/api/errors'

export const Route = createFileRoute('/_authed/orders/$orderNo')({
  loader: ({ context: { queryClient }, params }) => {
    if (!/^[A-Z]{2,5}-\d{4,}$/.test(params.orderNo)) throw new Error('not_found')
    return queryClient.ensureQueryData(orderQuery(params.orderNo)).catch(() => undefined)
  },
  pendingComponent: DetailSkeleton,
  errorComponent: ({ error }) => (
    <Pass>
      {isApiError(error) && error.code === 'not_found' ? null : <ErrorState error={error} />}
      <div className="px-6 pb-6 text-center text-[13px]">No such order. <Link to={ROUTES.ORDERS} className="font-bold text-leaf-700 underline">Back to orders</Link></div>
    </Pass>
  ),
  component: () => {
    const { orderNo } = Route.useParams()
    return <OrderDetailScreen orderNo={orderNo} />
  },
})
