# Chat to paid order

**A Chord AI teammate on WhatsApp for a Lagos food vendor.** A customer asks what is on the menu, orders, gets a Paystack link, pays, and the order is confirmed with stock reduced. The vendor watches it happen on an admin panel.

Built in one day for the TSG Labs build exercise. **Role: Forward-deployed Engineering** (second choice: Technical Product Management).

| | |
|---|---|
| **Try the agent** | https://chat.tsglabs.dev/c/jIMwkWWy_4MoJukWyBnCZ3h1 (WhatsApp simulator; pick "Success" on the Paystack test checkout) |
| **Admin panel** | https://chat-to-order-986.netlify.app |
| **Backend + MCP server** | https://backend-production-c54f.up.railway.app |
| **Code** | https://github.com/spiderocious/chat-to-order |
| **Recording** | 3-minute walkthrough in the Drive folder |

---

## Contents

1. [The problem and the business](#1-the-problem-and-the-business)
2. [What the agent does](#2-what-the-agent-does)
3. [How it is built](#3-how-it-is-built)
4. [Testing and results](#4-testing-and-results)
5. [Deployment field notes](#5-deployment-field-notes)
6. [Go-live note for the business owner](#6-go-live-note-for-the-business-owner)
7. [Links](#7-links)
8. [Run it yourself](#8-run-it-yourself)
9. [Repository layout](#9-repository-layout)
10. [How AI was used](#10-how-ai-was-used)

---

## 1. The problem and the business

### Who

Iya Bisi runs a home kitchen at 14 Adelabu Street, Surulere, Lagos. She cooks lunch and dinner in fixed portions and sells to offices and homes nearby, entirely over WhatsApp. She is a made-up vendor, but the kitchen is a type that exists on every street in Lagos: one cook, one phone, a menu that changes daily, and money that arrives by bank transfer.

### What her day looks like (illustrative, not measured)

A kitchen like this typically takes 20 to 40 orders a day across two service windows. Each order is three to five WhatsApp messages: "what do you have", "how much with delivery", "I've sent it, check". That is roughly 100 to 200 messages a day typed by the same person who is cooking. The same three questions account for most of them.

### Where she loses money and trust

- **Oversold dishes.** The portion count lives in her head. Two customers order the last ofada at the same time; one gets a refund argument.
- **Lost orders.** "I want" never becomes "I've paid", or a transfer screenshot arrives with no name and nobody can match it.
- **Fake proof of payment.** A screenshot is not a payment. Doctored transfer alerts are a known pattern on WhatsApp commerce, and a busy cook has no way to check.
- **Time.** Every minute typing is a minute not cooking, and the replies slow down exactly when demand peaks.

### Why she would pay

If the assistant takes even half the repetitive messages and stops one oversold dish a day, that is an hour of her time and one refund argument a day. A flat fee of a few thousand naira a month, or a small per-order fee, is cheap against that. The platforms she could use instead (delivery aggregators) take a percentage of every order and put her under their brand; this keeps her on the WhatsApp number her customers already have.

### What the agent solves

It shows today's menu with live portion counts, takes the order, prices it with the delivery fee, sends a Paystack link, confirms the order only when Paystack says the money arrived, reduces the portions, and gives her a board she can watch from the kitchen. She never confirms an order that was not paid, and she never answers "how much is jollof" again.

### Deliberately left out

Delivery dispatch and rider tracking, refunds, multiple vendors, building the menu from photos, customer accounts and order history, Pidgin or Yoruba beyond a light touch, and gadget-vendor needs (serial numbers, warranties, pay on delivery). Gadget vendors are the stated next vertical: same engine, more trust and stock rules.

---

## 2. What the agent does

One teammate on Chord, one skill, four tools over MCP. The conversation, as it ran in the simulator on 27 October:

| Customer | Teammate |
|---|---|
| "Good evening, what do you have tonight?" | Calls `get_menu`. Lists the evening items with prices and portions left, delivery zones and fees, pickup address, and the 40-minute prep time. |
| "1 fried rice and 2 moi moi, delivery to 7 Herbert Macaulay Way, Yaba. I'm Ngozi." | Calls `create_order`. Replies with the lines, delivery ₦1,200, total ₦6,300, the Paystack link, and "Order IBK-0240 · ref IBK-0240-01bd". |
| "paid" (before actually paying) | Calls `verify_payment`. "Your payment isn't confirmed yet. The link is still open until 6:03 PM." Never the word "failed". |
| Pays on Paystack's test checkout | Lands on a "Payment received, reply paid on WhatsApp" page. |
| "paid now" | Calls `verify_payment` then `confirm_order`. "Order IBK-0240 is confirmed… ready by about 6:20 PM. The rider will call you." |
| "Also add 12 egusi for pickup" | `create_order` refuses with `item_unavailable, max_qty: 10`. "We only have 10 egusi left tonight. I can do up to 10, or a swap." |

### The four tools

| Tool | What it does | External call |
|---|---|---|
| `get_menu` | Current window, whether orders are open, every offered item with price and portions left, delivery zones, pickup address | none |
| `create_order` | Validates items and quantities against stock and the cut-off, prices the cart in kobo, writes a pending order, creates the Paystack transaction, returns the link. Idempotent for the same cart | Paystack `transaction/initialize` |
| `verify_payment` | Asks Paystack about a reference; maps the answer to `success`, `not_confirmed`, `failed` or `amount_mismatch` | Paystack `transaction/verify` |
| `confirm_order` | Verifies again, locks the order and the menu rows, decrements portions, marks paid with a ready-by time. Idempotent on the reference. A sold-out line after payment becomes `paid_needs_attention` for the vendor | Paystack `transaction/verify` |

### Rules the code enforces regardless of what the model does

- Money is integer kobo everywhere. Every money field is returned twice, as `*_kobo` and as a display string the model repeats.
- An order is confirmed only when Paystack's verify says success and the amount and currency match.
- Stock moves once per reference, inside a transaction with row locks. Two customers paying for the last portion at the same instant: one is confirmed, the other is flagged for the vendor.
- "Pending" and "abandoned" are "not confirmed yet", never "failed".
- The model cannot set a price, a status or a count. Its arguments are validated and normalised; its errors are full sentences with a `next_step`.
- A reconciler re-verifies pending orders every minute, so a customer who pays and never says "paid" is still confirmed.

---

## 3. How it is built

```
WhatsApp simulator ──► Chord teammate (skill: food-vendor)
                             │  MCP over Streamable HTTP, bearer auth
                             ▼
                     Go + Gin service on Railway ──► Paystack (test mode)
                     ├── Postgres: orders, stock, audit, tool calls
                     ├── Redis: cache, locks, rate limits, idempotency, pub/sub
                     └── REST + SSE ──► React admin panel on Netlify
```

| Part | Stack | Notes |
|---|---|---|
| Backend + MCP | Go 1.27, Gin, pgx, goose, go-redis, MCP Go SDK | One service: `/mcp` for Chord, `/api/v1` for the panel, `/paid` for the customer, `/healthz`, `/metrics` |
| Database | Postgres on Railway | Schema and the default menu seeded by migrations; `version` columns for optimistic concurrency |
| Redis | Railway | Every use fails open; correctness lives in Postgres |
| Admin panel | React 19, TypeScript, Vite, TanStack Router + Query, Tailwind v4, motion | Light mode, phone-first, live updates over SSE, polling when degraded |
| Skill | `SKILL.md` + two references | Revision 5 published after five rounds of real-conversation fixes |
| Hosting | Railway (backend, from `/backend`), Netlify (panel, from `/frontend`) | Two deployables, same repo, nothing at the root |

No Paystack webhook: the account is shared with other projects, so payment truth comes from calling verify, and the reconciler covers the silent-payer case. Full designs: [docs/tech-spec-backend.html](docs/tech-spec-backend.html) and [docs/tech-spec-web.html](docs/tech-spec-web.html).

---

## 4. Testing and results

Tested at every layer before the next was built on it, then again on the hosts. Full log with screenshots in [docs/QA.md](docs/QA.md).

| Layer | What ran | Result |
|---|---|---|
| Backend unit + integration | `go test -race`: money, windows, Paystack mapping, alias decoding; real-Postgres tests for create/verify/confirm, idempotent cart, validation matrix, mismatch and declined, Paystack down, admin transitions, six goroutines racing for the last two portions | green |
| End-to-end smoke, local | 52 checks through `/mcp` and `/api/v1` with the fake checkout | 52 / 52 |
| End-to-end smoke, live on Railway | Same with the real Paystack test key | 46 / 46 |
| Chord registration | Tools discovered (4), health probe `ok` at 252 ms | ok |
| Real conversations | 7 turns by API, 6 turns in the simulator, two real Paystack test payments, both confirmed | confirmed |
| Admin panel | Every screen at desktop and 390 px; four UI bugs found and fixed; live CORS verified | ok |
| Credits | 13 teammate turns in total | under 1 of 50 used |

Tool latency on the live service: `get_menu` 250 to 290 ms, `create_order` 240 to 250 ms, `verify_payment` 560 to 600 ms (one Paystack round trip), `confirm_order` 260 to 420 ms.

---

## 5. Deployment field notes

Eight problems hit while building and deploying, in the order they happened. Each: what happened, how the cause was found, the fix, and what Chord, Paystack or Railway could change.

### 5.1 Chord's model guessed the tool argument names, and the customer saw the retries

**What happened.** First real order. The teammate called `create_order` three times and every call failed schema validation: `missing properties: ["customer_name" "items" "fulfilment"]`, then `unexpected additional properties ["id"]`, then `["quantity"]`. It gave up and sent its own narration to the customer: *"Trying the quantity field under a different name."*

**How I found the cause.** The `steps` array on `POST /v1/external-chat/messages` lists every tool call with its result text, so all three attempts were visible in one response. The MCP Go SDK infers a schema with `additionalProperties: false`, so `id` for `item_id` and `quantity` for `qty` were hard rejections, with validator text as the error.

**Fix.** The server publishes an explicit, lenient schema (an example in the description, no `additionalProperties: false`, no hard `required` list) and the handler normalises aliases (`id`/`item`/`name` → `item_id`, `quantity`/`count` → `qty`, `customer`/`name` → `customer_name`, `fulfillment` → `fulfilment`, a bare order number where a reference is expected). Validation errors became full sentences with a `next_step`. The skill gained: never narrate tool attempts to the customer. The next conversation succeeded on the second call and the customer saw a clean reply.

**What I'd change in Chord.** Surface the tool's `inputSchema` example to the model more prominently, or let a server mark a tool lenient so near-miss arguments pass through. And treat a reply containing tool-retry narration as a failed turn rather than delivering it.

### 5.2 Paystack passes its fee to the customer, so the verified amount is not the order total

**What happened.** The order total was ₦8,700. Checkout showed **NGN 8,934.02**. `verify_payment` returned `amount_mismatch`, the guard refused to confirm, and the customer was told the vendor would be in touch.

**How I found the cause.** The checkout page itself showed the larger figure. Paystack's verify response carries `amount` (what was charged, fee included when the merchant passes it to the customer), `requested_amount` (what we asked for) and `fees`. The code compared `amount`.

**Fix.** Compare the order total against `requested_amount`, falling back to `amount`, and record both plus `fees` on the payment check. A unit test pins the fee-bearer case. The same order confirmed on the next "paid".

**What I'd change.** Paystack: put `requested_amount` next to `amount` in the verify docs with one sentence on fee bearer. Chord: the built-in Paystack connector should say which figure it checks.

### 5.3 Paystack rejects a `.example` payer email

**What happened.** Live `create_order` failed with `paystack_rejected: "email" must be a valid email.` The server sent `orders+ibk-0231@iyabisi.example` because WhatsApp customers give no email and `.example` is the reserved documentation TLD.

**How I found the cause.** The tool returned Paystack's message verbatim and the Railway log had the same text. Local runs use a fake Paystack, so the first real initialize was on the deployed service.

**Fix.** A configurable `ORDER_EMAIL_DOMAIN` with a valid default; the per-order address stays synthetic and nothing is emailed.

**What I'd change.** Paystack: accept RFC 2606 reserved domains in test mode, or say in the docs that the TLD is validated. For me: one real initialize before trusting the fake.

### 5.4 The Skills API refuses a new skill until the teammate is "migrated"

**What happened.** `POST /v1/teammates/{id}/skills` on a teammate created seconds earlier returned `400 FAILED_PRECONDITION: Initialize skills from existing instructions before modifying them`.

**How I found the cause.** The skills guide mentions `skills/migrate` for legacy teammates; a brand-new teammate created with a `persona` is also "legacy" in this sense. Migrate turned the persona into an "Existing instructions" skill and unlocked the endpoint.

**Fix.** Create → migrate → create skill → publish → exclude the placeholder skill. Scripted in `backend/scripts/chord/00-create-teammate.sh`.

**What I'd change in Chord.** Auto-migrate on first skill create, or accept skill files on `POST /v1/teammates`. Also, two skills responses contain raw control characters inside JSON strings, which strict parsers reject.

### 5.5 Railway: the GitHub app had no access to the new repo, and `railway up` from a subfolder breaks the root directory

**What happened.** The first deployment built from the repo root and failed (no Dockerfile there). After setting `rootDirectory: backend` through the GraphQL API, `railway up` from inside `backend/` uploaded that folder as the snapshot root, so the builder looked for `backend/backend`. A push to `main` triggered nothing: `railway service source connect` answered `User does not have access to the repo`.

**How I found the cause.** `railway logs --build` per deployment id, and a GraphQL `serviceInstance` query to confirm `rootDirectory` and `source.repo`.

**Fix.** `railway up` from the repository root with the root directory set to `backend`. Auto-deploy on push needs the Railway GitHub app granted on the repo, a browser-only step for the account owner.

**What I'd change.** Railway CLI: a `--root-directory` flag, and a clear message when `up` runs from a subfolder of a service with a root directory.

### 5.6 The demo depended on the time of day

**What happened.** The live smoke failed 13 checks at 16:05 WAT because the headline dish was seeded lunch-only and the lunch window had closed. A reviewer trying the agent in the evening would have been told jollof is not available.

**How I found the cause.** `get_menu` on the live service listed only evening items; the smoke asserted on jollof.

**Fix.** A migration makes jollof and fried rice all-day; ofada stays lunch-only and egusi evening-only so the window rule is still demonstrable. The smoke uses unique customer names per run so it never collides with an older open order.

**What I'd change.** Nothing on Chord's side. For me: seed demo data for the reviewer's clock, not the builder's.

### 5.7 The model used the example reference from the skill as if it were real

**What happened.** Asked to "check again", the teammate called `verify_payment` with `IBK-0231-a1b2`, the placeholder from the skill's examples and the tool description, not the real `IBK-0235-82e2`. On `unknown_reference` it told the customer it had never created the order and that the link "didn't come from the kitchen's system", which was false.

**How I found the cause.** The `steps` for that turn show the exact argument. The reply that carried the link had omitted the reference, and tool results are not part of the conversation history on later turns, so the only `IBK-` strings the model could see were the examples.

**Fix.** Every reply that sends a link must carry the order number and reference on their own line. All examples use obvious placeholders (`IBK-0000-xxxx`) and say so. Hard rules: a reference only comes from `create_order` in this conversation or from the order number the customer reads on the payment page (the tools accept the order number alone); never retract or deny an order unless a tool proves it. `unknown_reference` now says exactly what to ask the customer. On the next turn the teammate verified by order number and confirmed correctly.

**What I'd change in Chord.** Keep tool results, or at least their `next_step` and ids, in the context of later turns, or give tools a small per-conversation memory. Today a builder has to smuggle state through the model's own replies.

### 5.8 The reconciler confirmed the order before the model did

**What happened.** After the fee fix deployed, the next "paid" led to `confirm_order` returning `already_confirmed: true, confirmed_by: reconciler`. The same happened in the simulator run. The background reconciler had re-verified the pending order on its one-minute tick and confirmed it first.

**Why it matters.** This is the no-webhook design working: a shared Paystack account cannot point a webhook at this service, so payment truth comes only from verify, and the reconciler covers customers who pay and never say so. The model path and the reconciler path converged on the same row under the row lock, with one stock decrement.

**What I'd change.** Nothing; this is the behaviour I wanted. The go-live note explains the lag of up to about three minutes.

---

## 6. Go-live note for the business owner

**What it does.** Customers message your WhatsApp number. The assistant shows today's menu from your admin panel, takes the order, sends a Paystack payment link, and confirms the order only after Paystack says the money arrived. Your portion counts go down by themselves. You watch orders arrive on the board, on your phone or laptop.

**What it does not do.** It does not send riders, give refunds, handle complaints about food already delivered, or add dishes by itself. You set the menu, prices and portions each morning in the panel. If a dish runs out after someone has paid, the order turns red as "Needs you" and you call the customer to swap it or refund by transfer yourself.

**Before real customers use it.**
1. Your own Paystack business account with live keys and your settlement bank account. Today it runs on test keys; test cards work, real cards do not.
2. Your WhatsApp Business number connected to Chord. Today it runs in Chord's simulator.
3. Real prices and portions in the panel, and the service windows set to your real hours.
4. A Chord credit top-up and somebody watching the balance. When credits run out the assistant goes silent.
5. A one-week trial with five regulars you trust, with you watching the board.

**Money.** Prices in the panel are what the customer pays, plus Paystack's fee if your Paystack account is set to pass the fee to the customer (today it is, so a ₦8,700 order shows as ₦8,934 on the payment page). Settle that choice in Paystack before launch. The assistant never takes an account number or a transfer screenshot as proof of payment.

**When a payment fails.** The customer sees the bank's reason and gets a fresh link. Nothing is recorded and no food is reserved. If a customer closes the payment page without finishing, the link stays open for 30 minutes, then the order expires on its own.

**When something breaks.** If Paystack or the ordering system is down, the assistant tells the customer nothing was charged and to try again in a minute, and the Activity page shows the error. If the assistant cannot answer at all, customers get no reply; call Feranmi. Alerts to your phone when an order needs you or the service is down are the first thing to add after launch.

**Honest risks.** Anyone who knows your number can order; the order number is the identity. The assistant can misread a message and ask again rather than guess. Two customers paying for the last portion at the same instant: one is confirmed, the other is flagged for you. A payment the customer never mentions is confirmed within about three minutes, not instantly. The panel runs on a single small server, fine for one kitchen at today's volume.

**Next.** Adding dishes by sending a photo of your handwritten menu, and a WhatsApp alert when an order needs you. Gadget sellers are the next vertical: same assistant, plus stock counts, warranties and pay-on-delivery.

---

## 7. Links

**Live**
- WhatsApp simulator chat with the teammate: https://chat.tsglabs.dev/c/jIMwkWWy_4MoJukWyBnCZ3h1
- Admin panel: https://chat-to-order-986.netlify.app
- Backend health: https://backend-production-c54f.up.railway.app/healthz
- MCP endpoint (bearer-protected): https://backend-production-c54f.up.railway.app/mcp

**Code and documents**
- Repository: https://github.com/spiderocious/chat-to-order
- Backend and MCP technical design: [docs/tech-spec-backend.html](docs/tech-spec-backend.html)
- Admin panel technical design: [docs/tech-spec-web.html](docs/tech-spec-web.html)
- Build plan as it happened: [docs/plan.html](docs/plan.html)
- QA log with screenshots: [docs/QA.md](docs/QA.md), [docs/qa/screenshots/](docs/qa/screenshots/)
- The skill: [backend/skill/food-vendor/](backend/skill/food-vendor/)

**Reference**
- Chord API docs: https://api.chordhq.xyz/docs
- Paystack transaction API: https://paystack.com/docs/api/transaction/
- Paystack test payments: https://paystack.com/docs/payments/test-payments/

---

## 8. Run it yourself

### Backend

```bash
cd backend
cp .env.example .env            # PAYSTACK_FAKE=1 by default; no Paystack account needed locally
docker compose up -d            # Postgres on 5433, Redis on 6380
make dev                        # migrates, seeds the menu and six demo orders, listens on :8080
make test                       # go vet + go test -race (needs docker compose up)
make smoke                      # 52 end-to-end checks through /mcp and /api/v1
make skill                      # zip the Chord skill package
```

Local admin login: `devferanmi@gmail.com` / `admin12345` (from `.env.example`).

### Admin panel

```bash
cd frontend
cp .env.example .env            # VITE_API_BASE_URL=http://localhost:8080
pnpm install && pnpm dev        # http://localhost:5173
pnpm check                      # typecheck, lint, tests
```

### Deploy

- Backend: Railway service with root directory `backend`, Dockerfile build, Postgres and Redis attached, variables as in `backend/.env.example` plus `DATABASE_URL=${{Postgres.DATABASE_URL}}` and `REDIS_URL=${{Redis.REDIS_URL}}`. `railway up` from the repo root, or a push to `main` once the Railway GitHub app has the repo.
- Panel: Netlify site with base `frontend`, `netlify.toml` in that folder, `VITE_API_BASE_URL` set to the Railway domain.
- Chord: `backend/scripts/chord/00-create-teammate.sh` then `01-register.sh`, with `CHORD_API_KEY`, `CHORD_WORKSPACE_ID`, `MCP_URL` and `MCP_BEARER_TOKEN` in the environment.

---

## 9. Repository layout

```
README.md                         this file
PROBLEM-STATEMENT.md · FIELD-NOTES.md · GO-LIVE.md   the submission pieces, also merged above
backend/                          Go service: cmd/server, internal/{config,money,clock,store,redisx,paystack,ordering,reconcile,activity,mcpserver,httpapi}
backend/skill/food-vendor/        SKILL.md + references/vendor-rules.md + references/conversation-examples.md
backend/scripts/                  smoke.sh, chord/00-create-teammate.sh, chord/01-register.sh, chord/02-usage.sh, chord/03-chat.sh
frontend/                         Vite app: src/routes, src/features/{auth,live,board,orders,menu,settings,activity}, src/shared/{api,ui,utils}
docs/                             plan.html, tech-spec-backend.html, tech-spec-web.html, QA.md, qa/screenshots/, chord-docs/
```

---

## 10. How AI was used

Claude Code wrote the plan and the two technical designs from the brief, scaffolded the backend and the panel from the tool contracts, drove the browser for Chord setup, the simulator conversation and the Paystack test checkout, and drafted these documents. Feranmi made the product decisions, supplied the accounts and keys, debugged the deployment with it, ran the recording, and edited everything sent to TSG. Every problem in the field notes was hit for real and fixed by hand; none is invented.
