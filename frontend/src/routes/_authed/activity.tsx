import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import ActivityScreen from '@features/activity/screen/activity-screen'

export const Route = createFileRoute('/_authed/activity')({
  validateSearch: z.object({ tool: z.string().optional(), ok: z.enum(['true', 'false']).optional(), order_no: z.string().optional() }),
  component: () => {
    const { tool, ok, order_no } = Route.useSearch()
    return <ActivityScreen tool={tool} ok={ok === undefined ? undefined : ok === 'true'} orderNo={order_no} />
  },
})
