-- +goose Up
CREATE TABLE vendor_settings (
  id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  vendor_name text NOT NULL,
  pickup_address text NOT NULL,
  rider_phone text NOT NULL DEFAULT '',
  order_prefix text NOT NULL DEFAULT 'IBK',
  prep_minutes int NOT NULL DEFAULT 40 CHECK (prep_minutes BETWEEN 5 AND 240),
  lunch_open text NOT NULL, lunch_close text NOT NULL, lunch_orders_close text NOT NULL,
  evening_open text NOT NULL, evening_close text NOT NULL, evening_orders_close text NOT NULL,
  pending_ttl_minutes int NOT NULL DEFAULT 30,
  low_stock_threshold int NOT NULL DEFAULT 3,
  version int NOT NULL DEFAULT 1,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE menu_items (
  id text PRIMARY KEY CHECK (id ~ '^[a-z0-9-]{2,24}$'),
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  price_kobo bigint NOT NULL CHECK (price_kobo > 0),
  portions int NOT NULL CHECK (portions >= 0),
  available boolean NOT NULL DEFAULT true,
  "window" text NOT NULL CHECK ("window" IN ('lunch','evening','all')),
  sort int NOT NULL DEFAULT 100,
  photo_url text,
  version int NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE delivery_zones (
  id text PRIMARY KEY CHECK (id ~ '^[a-z0-9-]{2,24}$'),
  name text NOT NULL,
  fee_kobo bigint NOT NULL CHECK (fee_kobo >= 0),
  active boolean NOT NULL DEFAULT true,
  sort int NOT NULL DEFAULT 100
);

CREATE SEQUENCE order_no_seq START 231;

CREATE TABLE orders (
  id text PRIMARY KEY,
  order_no text NOT NULL UNIQUE,
  status text NOT NULL CHECK (status IN
    ('pending_payment','paid','paid_needs_attention','resolved','cancelled','expired')),
  customer_name text NOT NULL,
  phone text,
  note text,
  fulfilment text NOT NULL CHECK (fulfilment IN ('delivery','pickup')),
  zone_id text REFERENCES delivery_zones(id),
  address text,
  "window" text NOT NULL CHECK ("window" IN ('lunch','evening')),
  subtotal_kobo bigint NOT NULL,
  delivery_fee_kobo bigint NOT NULL,
  total_kobo bigint NOT NULL,
  reference text UNIQUE,
  authorization_url text,
  cart_hash text,
  source text NOT NULL DEFAULT 'mcp' CHECK (source IN ('mcp','admin')),
  paid_at timestamptz,
  ready_by timestamptz,
  confirmed_by text,
  attention_reason text,
  resolved_at timestamptz,
  resolution_note text,
  cancel_reason text,
  expires_at timestamptz NOT NULL,
  reconciled_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX orders_open_cart ON orders (cart_hash) WHERE status = 'pending_payment';
CREATE INDEX orders_status_created ON orders (status, created_at DESC);
CREATE INDEX orders_created ON orders (created_at DESC, id DESC);

CREATE TABLE order_lines (
  id bigserial PRIMARY KEY,
  order_id text NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  item_id text NOT NULL REFERENCES menu_items(id),
  name_snapshot text NOT NULL,
  unit_price_kobo bigint NOT NULL,
  qty int NOT NULL CHECK (qty > 0),
  line_kobo bigint NOT NULL
);
CREATE INDEX order_lines_order ON order_lines (order_id);

CREATE TABLE order_events (
  id bigserial PRIMARY KEY,
  order_id text NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  type text NOT NULL,
  actor text NOT NULL,
  data jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX order_events_order ON order_events (order_id, id);

CREATE TABLE payment_checks (
  id bigserial PRIMARY KEY,
  order_id text REFERENCES orders(id) ON DELETE CASCADE,
  reference text NOT NULL,
  source text NOT NULL,
  paystack_status text,
  amount_kobo bigint,
  currency text,
  gateway_response text,
  outcome text NOT NULL,
  latency_ms int,
  raw jsonb,
  checked_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX payment_checks_order ON payment_checks (order_id, id);

CREATE TABLE tool_calls (
  id bigserial PRIMARY KEY,
  tool text NOT NULL,
  args jsonb NOT NULL,
  result jsonb,
  ok boolean NOT NULL,
  error_code text,
  latency_ms int NOT NULL,
  order_no text,
  request_id text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tool_calls_created ON tool_calls (created_at DESC, id DESC);

CREATE TABLE admins (
  id text PRIMARY KEY,
  email text NOT NULL UNIQUE,
  name text NOT NULL,
  password_hash text NOT NULL DEFAULT '',
  token_version int NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS admins, tool_calls, payment_checks, order_events, order_lines, orders, delivery_zones, menu_items, vendor_settings;
DROP SEQUENCE IF EXISTS order_no_seq;
