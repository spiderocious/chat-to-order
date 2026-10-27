export const EP = {
  AUTH: { LOGIN: '/api/v1/auth/login', LOGOUT: '/api/v1/auth/logout', ME: '/api/v1/auth/me' },
  VENDOR: '/api/v1/vendor',
  MENU: { LIST: '/api/v1/menu', ITEM: (id: string) => `/api/v1/menu/${encodeURIComponent(id)}` },
  ZONES: { LIST: '/api/v1/zones', ZONE: (id: string) => `/api/v1/zones/${encodeURIComponent(id)}` },
  ORDERS: {
    LIST: '/api/v1/orders',
    DETAIL: (no: string) => `/api/v1/orders/${encodeURIComponent(no)}`,
    RESOLVE: (no: string) => `/api/v1/orders/${encodeURIComponent(no)}/resolve`,
    CANCEL: (no: string) => `/api/v1/orders/${encodeURIComponent(no)}/cancel`,
  },
  STATS: '/api/v1/stats/today',
  ACTIVITY: '/api/v1/activity',
  EVENTS: '/api/v1/events',
  RESET: '/api/v1/admin/reset',
} as const
