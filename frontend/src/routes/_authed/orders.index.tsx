import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import OrdersScreen from '@features/orders/screen/orders-screen'

const Search = z.object({ status: z.string().optional() })

export const Route = createFileRoute('/_authed/orders/')({
  validateSearch: Search,
  component: function OrdersIndexPage() {
    const { status } = Route.useSearch()
    return <OrdersScreen status={status ?? ''} />
  },
})
