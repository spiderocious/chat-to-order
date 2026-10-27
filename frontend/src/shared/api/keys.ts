export interface OrdersFilter { status?: string; today?: boolean }
export interface ActivityFilter { tool?: string; ok?: boolean; order_no?: string }

export const K = {
  me: ['me'] as const,
  vendor: ['vendor'] as const,
  menu: ['menu'] as const,
  zones: ['zones'] as const,
  stats: ['stats', 'today'] as const,
  orders: (f: OrdersFilter) => ['orders', f] as const,
  ordersAll: ['orders'] as const,
  order: (no: string) => ['order', no] as const,
  activity: (f: ActivityFilter) => ['activity', f] as const,
  activityAll: ['activity'] as const,
}
