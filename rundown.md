# Rundown: what was built, and how to test it

Handover notes for the TSG Labs take-home. Everything below is live as of 27 Oct 2026. The long-form write-up is [README-v2.md](README-v2.md); this file is the short version with the logins and the click-by-click test paths.

## 1. What was done

- **The business.** TPlus Kitchen, a real food and small-chops vendor in Osogbo that takes orders on WhatsApp ([Instagram](https://www.instagram.com/teeplus_events/), [X](https://x.com/Tplusevent)). The demo menu, zones and prices are illustrative; the vendor has not been contacted.
- **The agent.** A Chord teammate ("TPlus Kitchen") with one skill (`food-vendor`, revision 6) and four tools over MCP: `get_menu`, `create_order`, `verify_payment`, `confirm_order`. It shows the menu, takes the order, sends a Paystack test-mode link, verifies the payment with Paystack and confirms the order with stock reduced. It never confirms an order Paystack has not confirmed.
- **The backend.** Go + Gin on Railway: `/mcp` (Streamable HTTP, bearer-protected) for Chord, `/api/v1` for the panel, `/paid` for the customer after checkout, `/healthz`. Postgres for truth, Redis fail-open for cache, locks, rate limits, idempotency and live events. A reconciler re-verifies pending orders every minute, so a customer who pays and never says "paid" is still confirmed.
- **The admin panel.** React on Netlify: Board (tiles, today's orders, stock strip, assistant activity), Orders (filters, cursor paging, detail with timeline and payment checks), Menu (inline portions and availability), Settings (kitchen, windows, zones), Activity (every tool call with args and result). Live over SSE; phone-first.
- **Testing.** Unit and integration tests (`go test -race`), a 52-check end-to-end smoke (52/52 locally, 46/46 against the live Railway service with the real Paystack test key), real conversations in Chord's simulator with real Paystack test payments, every panel screen checked on the live site after signing in.
- **Docs.** [README-v2.md](README-v2.md) (problem, stats, design, field notes, go-live note, links), the build plan and two technical designs in [docs/](docs/) and hosted at https://chat-to-order-986.netlify.app/docs/plan.html, screenshots in [docs/screenshots/](docs/screenshots/).

## 2. Logins

| Where | URL | Email | Password |
|---|---|---|---|
| Admin panel, live | https://chat-to-order-986.netlify.app | `devferanmi@gmail.com` | `tplus-demo-2026` |
| Admin panel, local | http://localhost:5173 | `devferanmi@gmail.com` | `admin12345` |

The live password is a demo credential set on Railway as `ADMIN_PASSWORD`; change it there and restart to rotate (older sessions are revoked automatically). The MCP bearer token, JWT secret and Paystack key are in `.deploy.env` / `.secrets.env` on this machine only, never in git.

## 3. Test the deployed version (10 minutes)

**A. The conversation (the main deliverable).**

1. Open https://chat.tsglabs.dev/c/161x1iBUglWnrJc3APYog6u- (sign in with the Chord account if asked). Enter any name, "Start chatting".
2. Send: `Good evening, what do you have tonight?` → menu with prices, portions left, delivery zones and the pickup address.
3. Send: `2 jollof and 1 chapman, deliver to 12 Gbongan Road, Oke-Fia. My name is Tobi.` → lines, delivery ₦500, total ₦8,500, a Paystack link, and a line `Order TPK-0xxx · ref TPK-0xxx-xxxx`.
4. Send: `paid` **before** paying → "not confirmed yet, the link is still open until …". It never says "failed".
5. Open the Paystack link. Pick the **Success** test option (test mode; no real card). You land on a "Payment received" page with the order number.
6. Send: `paid now` → "Your payment is confirmed. Order TPK-0xxx … ready by about HH:MM. The rider will call you."
7. Send: `Also add 5 biryani for pickup, same name` → "only 2 left … I can do up to 2 or a swap" (stock guard).

**B. The panel, while the conversation runs.** Sign in at https://chat-to-order-986.netlify.app with the live login above.

- Board: the new order appears as *Awaiting pay*, flips to *Paid* when confirmed, with no reload (the "Live" chip is the SSE connection). Jollof's portion count drops by 2.
- Orders → click the order: timeline (created, link issued, verified, confirmed), the Paystack checks (reference, amount, fees), the tool calls that touched it.
- Activity: every tool call with latency, arguments and result; filter by tool or outcome.
- Menu: change portions with the stepper or switch an item off; send "what do you have" again in the chat and the reply reflects it.
- "Reset demo" (top right) wipes orders and restores the six demo orders. Use it before a recording.

**C. Health.** https://backend-production-c54f.up.railway.app/healthz → `db ok`, `redis ok`, `paystack configured`, `mcp_tools 4`.

## 4. Test it locally (5 minutes)

Both servers are already running on this machine (backend on :8080, panel on :5173). To start from cold:

```bash
cd backend
cp .env.example .env            # PAYSTACK_FAKE=1: a built-in fake checkout, no Paystack account needed
docker compose up -d            # Postgres on 5433, Redis on 6380
make dev                        # migrates, seeds the menu and six demo orders, listens on :8080
make test                       # go vet + go test -race (needs docker compose up)
make smoke                      # 52 end-to-end checks through /mcp and /api/v1
```

```bash
cd frontend
cp .env.example .env            # VITE_API_BASE_URL=http://localhost:8080
pnpm install && pnpm dev        # http://localhost:5173
pnpm check                      # typecheck, lint, tests
```

Then: sign in at http://localhost:5173 with the local login. `make smoke` is the fastest full demo: it takes an order through MCP, pays on the fake checkout page, verifies, confirms, and exercises the admin API; watch the board update while it runs.

To talk to the local tools directly:

```bash
curl -s localhost:8080/mcp -H "Authorization: Bearer $MCP_BEARER_TOKEN" \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_menu","arguments":{}}}'
```

Chord itself only talks to the deployed `/mcp`, so the local backend is for development and the smoke; the conversation is always tested against Railway.

## 5. Deploy again after a change

- Backend: commit, then `railway up --service backend --detach` from the **repo root** (the service's root directory is `backend`). Auto-deploy on push needs the Railway GitHub app granted on the repo, which only the account owner can do in the browser.
- Panel: `cd frontend && VITE_API_BASE_URL=https://backend-production-c54f.up.railway.app VITE_DEMO=1 pnpm build && netlify deploy --prod --dir dist --no-build --site 8fae3503-0533-4a29-a577-5305ac7ba9de`. The build copies `docs/*.html` into the site, so the hosted plan and specs update too.
- Skill: edit `backend/skill/food-vendor/`, then draft + publish through the Chord skills API (the exact calls are in `backend/scripts/chord/00-create-teammate.sh`).
- Live demo data: `cd backend && make reset-live` restores the six demo orders on Railway.

## 6. Known limits

- Railway deploys are pushed from this machine, not from GitHub (see above).
- The teammate's `persona` (its original system prompt, set at creation) still names the old demo vendor. Once a teammate is in skills mode the persona is frozen by design: `PUT …/playbook` returns `409 SKILLS_ENABLED`, and the generic `PUT /v1/teammates/{id}` returns `500` for any non-empty `persona` instead of that 409 (a Chord bug, field note 5.10). It is inert: the system prompt is built from the published `food-vendor` skill, and the migrated "Existing instructions" skill that holds the old text is excluded. Only Chord's admin UI shows it.
- Paystack is in test mode; real cards do not work and no money moves.
