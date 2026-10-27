# Backend: ordering domain, MCP server, admin API

Go 1.27, Gin, Postgres (pgx + goose), Redis (go-redis), MCP Go SDK. Deploys to Railway from this folder with the Dockerfile.

## Run

```bash
cp .env.example .env          # PAYSTACK_FAKE=1: no Paystack account needed locally
docker compose up -d          # Postgres :5433, Redis :6380
make dev                      # migrates, seeds the menu and six demo orders, listens on :8080
```

## Test

```bash
make test                     # go vet + go test -race; integration tests use TEST_DATABASE_URL
make smoke                    # 50 end-to-end checks through /mcp and /api/v1 (BASE=https://… for a deployed instance)
```

## Layout

```
cmd/server            boot: config → postgres → redis → migrate → domain → mcp → reconciler → http
internal/config       env → Config, fails on a missing key
internal/money        Kobo (int64) and Format("₦8,700")
internal/clock        Lagos time, service windows, ready-by rounding
internal/store        pgx queries, embedded goose migrations, demo seed
internal/redisx       cache, locks, rate limits, idempotency, JWT revocation, pub/sub; all fail open
internal/paystack     initialize + verify client, typed errors, and a Fake for local use
internal/ordering     the domain: menu, create order, verify, confirm, resolve, cancel
internal/reconcile    ticker: re-verify pending orders, cancel reference-less ones, expire old ones
internal/activity     tool-call log (phone and address masked)
internal/mcpserver    the four tools over Streamable HTTP, bearer-protected
internal/httpapi      REST + SSE for the admin panel, /paid page, /healthz, /metrics
skill/food-vendor     the Chord skill package (make skill → dist/food-vendor.zip)
scripts/chord         register the MCP server in Chord by API; read credit usage
scripts/smoke.sh      the end-to-end smoke test
```

## Environment

See `.env.example`. On Railway: `DATABASE_URL=${{Postgres.DATABASE_URL}}`, `REDIS_URL=${{Redis.REDIS_URL}}`, plus `PUBLIC_BASE_URL`, `CORS_ORIGINS`, `PAYSTACK_SECRET_KEY`, `MCP_BEARER_TOKEN`, `JWT_SECRET`, `METRICS_TOKEN`, `ADMIN_EMAIL`, `ADMIN_PASSWORD` (bootstrap only), `SEED_DEMO_ORDERS=1`.
