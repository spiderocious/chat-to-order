// Wire types copied from the Go serialiser (backend/internal/httpapi/views.go).
// snake_case on the wire, kept snake_case here. Unknown fields are tolerated.
import { z } from 'zod'

export const OrderStatus = z.enum(['pending_payment', 'paid', 'paid_needs_attention', 'resolved', 'cancelled', 'expired'])
export type OrderStatus = z.infer<typeof OrderStatus>

export const NextWindow = z.object({ name: z.string(), opens_at: z.string() }).nullable()

export const NowView = z.object({
  wat: z.string(),
  window: z.string(),
  orders_open: z.boolean(),
  orders_close_at: z.string().optional(),
  next_window: NextWindow,
})
export type NowView = z.infer<typeof NowView>

export const Admin = z.object({ id: z.string(), email: z.string(), name: z.string() })
export type Admin = z.infer<typeof Admin>

export const LoginRes = z.object({ token: z.string(), expires_at: z.string(), admin: Admin })
export const MeRes = z.object({ admin: Admin, expires_at: z.string().nullable().optional() })

export const WindowView = z.object({ open: z.string(), close: z.string(), orders_close: z.string() })
export const Vendor = z.object({
  vendor_name: z.string(),
  pickup_address: z.string(),
  rider_phone: z.string(),
  order_prefix: z.string(),
  prep_minutes: z.number().int(),
  pending_ttl_minutes: z.number().int(),
  low_stock_threshold: z.number().int(),
  windows: z.object({ lunch: WindowView, evening: WindowView }),
  now: NowView,
  version: z.number().int(),
  updated_at: z.string(),
}).passthrough()
export type Vendor = z.infer<typeof Vendor>

export const Item = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  price_kobo: z.number().int(),
  price_display: z.string(),
  portions: z.number().int(),
  available: z.boolean(),
  window: z.enum(['lunch', 'evening', 'all']),
  sort: z.number().int(),
  photo_url: z.string().nullable(),
  offered_now: z.boolean(),
  low: z.boolean(),
  version: z.number().int(),
  updated_at: z.string(),
}).passthrough()
export type Item = z.infer<typeof Item>

export const MenuRes = z.object({ items: z.array(Item), now: NowView })

export const Zone = z.object({
  id: z.string(), name: z.string(), fee_kobo: z.number().int(), fee_display: z.string(), active: z.boolean(), sort: z.number().int(),
})
export type Zone = z.infer<typeof Zone>
export const ZonesRes = z.object({ zones: z.array(Zone) })

export const OrderRow = z.object({
  order_no: z.string(),
  status: OrderStatus,
  customer_name: z.string(),
  phone_masked: z.string().nullable(),
  fulfilment: z.enum(['delivery', 'pickup']),
  zone: z.object({ id: z.string(), name: z.string() }).nullable(),
  address: z.string().nullable(),
  window: z.string(),
  lines_summary: z.string(),
  total_kobo: z.number().int(),
  total_display: z.string(),
  reference: z.string().nullable(),
  paid_at: z.string().nullable(),
  ready_by: z.string().nullable(),
  ready_by_wat: z.string().nullable(),
  confirmed_by: z.string().nullable(),
  attention_reason: z.string().nullable(),
  expires_at: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
}).passthrough()
export type OrderRow = z.infer<typeof OrderRow>

export const Line = z.object({
  item_id: z.string(), name: z.string(), qty: z.number().int(), unit_price_kobo: z.number().int(), unit_display: z.string(), line_kobo: z.number().int(), line_display: z.string(),
})
export const OrderEvent = z.object({ type: z.string(), actor: z.string(), data: z.unknown(), at: z.string() })
export const PaymentCheck = z.object({
  checked_at: z.string(), source: z.string(), paystack_status: z.string().nullable(), outcome: z.string(),
  amount_kobo: z.number().nullable(), gateway_response: z.string().nullable(), latency_ms: z.number().nullable(),
})
export const ToolCallBrief = z.object({
  id: z.number(), tool: z.string(), ok: z.boolean(), error_code: z.string().nullable(), latency_ms: z.number(), created_at: z.string(),
})

export const OrderDetail = OrderRow.extend({
  phone: z.string().nullable(),
  note: z.string().nullable(),
  source: z.string(),
  authorization_url: z.string().nullable(),
  subtotal_kobo: z.number().int(),
  subtotal_display: z.string(),
  delivery_fee_kobo: z.number().int(),
  delivery_fee_display: z.string(),
  lines: z.array(Line),
  events: z.array(OrderEvent),
  payment_checks: z.array(PaymentCheck),
  tool_calls: z.array(ToolCallBrief),
  resolved_at: z.string().nullable(),
  resolution_note: z.string().nullable(),
  cancel_reason: z.string().nullable(),
})
export type OrderDetail = z.infer<typeof OrderDetail>

export const ToolCall = ToolCallBrief.extend({
  order_no: z.string().nullable(),
  request_id: z.string().nullable(),
  args: z.unknown(),
  result: z.unknown(),
})
export type ToolCall = z.infer<typeof ToolCall>

export const Stock = z.object({ id: z.string(), name: z.string(), portions: z.number().int(), low: z.boolean(), offered_now: z.boolean() })
export const Stats = z.object({
  date_wat: z.string(),
  window: z.string(),
  orders_open: z.boolean(),
  orders_close_at: z.string().optional(),
  next_window: NextWindow,
  paid_count: z.number().int(),
  paid_total_kobo: z.number().int(),
  paid_total_display: z.string(),
  awaiting_payment_count: z.number().int(),
  needs_attention_count: z.number().int(),
  cancelled_count: z.number().int(),
  expired_count: z.number().int(),
  stock: z.array(Stock),
  low_stock_threshold: z.number().int(),
  redis: z.string(),
}).passthrough()
export type Stats = z.infer<typeof Stats>

export const Page = <T extends z.ZodTypeAny>(item: T) =>
  z.object({ data: z.array(item), meta: z.object({ next_cursor: z.string().nullable(), has_more: z.boolean() }) })
export const Data = <T extends z.ZodTypeAny>(item: T) => z.object({ data: item })
