# Chat to paid order

**A Chord AI teammate on WhatsApp for a Nigerian food vendor.** A customer asks what is on the menu, orders, gets a Paystack link, pays, and the order is confirmed with stock reduced. The vendor watches it happen on an admin panel.

| | |
|---|---|
| **Try the agent** | https://chat.tsglabs.dev/c/161x1iBUglWnrJc3APYog6u- (WhatsApp simulator; pick "Success" on the Paystack test checkout) |
| **Admin panel** | https://chat-to-order-986.netlify.app · login `devferanmi@gmail.com` / `tplus-demo-2026` |
| **Backend + MCP server** | https://backend-production-c54f.up.railway.app |
| **Code** | https://github.com/spiderocious/chat-to-order |
| **Plan** | [Plan Document](https://chat-to-order-986.netlify.app/docs/plan.html) |
| **Backend + MCP technical design document** | [backend + MCP document](https://chat-to-order-986.netlify.app/docs/tech-spec-backend.html) |
| **Web application technical design document** | [web design doc](https://chat-to-order-986.netlify.app/docs/tech-spec-web.html) |
| **Recording** | 3-minute walkthrough in the Drive folder |


---


[![The live board after a real simulator order](docs/screenshots/03-phone-board-detail-menu.jpg)](docs/screenshots/03-phone-board-detail-menu.jpg)

---

## Contents

1. [The problem and the business](#1-the-problem-and-the-business)
2. [What the agent does](#2-what-the-agent-does)
3. [How it is built](#3-how-it-is-built)
4. [Testing and results](#4-testing-and-results)
5. [Go-live note for the business owner](#6-go-live-note-for-the-business-owner)
6. [Links](#7-links)
7. [Run it yourself](#8-run-it-yourself)
8. [Repository layout](#9-repository-layout)

---

## 1. The problem and the business

### The vendor

[TPlus Kitchen](https://www.instagram.com/teeplus_events/) ([@Tplusevent](https://x.com/Tplusevent)) is a food and small-chops vendor in Osogbo, Osun State. Daily food delivery, daily small chops delivery, food trays and cocktails for events, and customers can only order on whatsapp. There is no website, no ordering app and no payment page. Every order is a conversation, and every payment is a bank transfer that somebody has to match to a chat.

TPlus is one of a very large crowd. The menu, prices, zones and portion counts in this demo are illustrative.

### The size of the crowd

- Nigeria has **39.65 million micro, small and medium enterprises**, and 96% of them are nano or micro businesses, most of them sole proprietors ([SMEDAN / NBS national survey](https://www.nigerianstat.gov.ng/pdfuploads/MSMES.pdf)). Informal businesses push the real number higher ([FATE Foundation](https://fatefoundation.org/wp-content/uploads/2024/03/7.-SMEDAN-1.pdf)).
- Roughly **14 million Nigerian small businesses sold through Meta's apps (Facebook, Instagram, WhatsApp) in 2025**, which a Meta-commissioned study values at about $820 million a year to the economy ([Vanguard, 2026](https://www.vanguardngr.com/2026/05/how-facebook-instagram-whatsapp-pump-820-m-into-nigerias-economy-yearly/)).
- WhatsApp reaches about **95% of Nigerian internet users** ([Yazi](https://www.askyazi.com/articles/whatsapp-penetration-across-africa-statistics-by-country)). In an analysis of 74,901 listed Nigerian businesses, only 57.7% had a website; business happens over calls and WhatsApp ([myhustle.space](https://myhustle.space/insights/state-of-small-business-nigeria-2026)).
- The money is already digital. Electronic payments in Nigeria reached **₦284.99 trillion in the first quarter of 2025**, up 17.7% on the year ([Nairametrics](https://nairametrics.com/2025/07/26/e-payment-transactions-in-nigeria-hit-n284-9-trillion-in-q1-2025/)), after about ₦1.07 quadrillion in 2024. Instant transfer is the default way a customer pays a small vendor.
- Online food delivery alone is a **$1.14 billion market in 2025**, forecast to reach $2.7 billion by 2034 ([IMARC](https://www.imarcgroup.com/nigeria-online-food-delivery-market)); the aggregators that make up that number take a cut of every order and own the customer relationship. The WhatsApp vendor is the part of the market the aggregators have not reached.

Nobody publishes a count of orders placed over WhatsApp; the figures above are the boundaries of the problem, not a measurement of it.

### Where the money leaks

- **Fake proof of payment.** A screenshot is not a payment. Doctored transfer alerts are a known pattern against small sellers ([The Guardian Nigeria](https://guardian.ng/news/concerns-as-more-nigerians-fall-prey-to-fake-bank-alerts/), [ThisDay](https://www.thisdaylive.com/2025/06/28/how-not-to-fall-prey-to-fake-bank-alerts/)). NIBSS counted **67,518 digital-payment fraud cases in 2025 with ₦25.85 billion lost**, down from ₦52.26 billion in 2024 ([Legit.ng](https://www.legit.ng/business-economy/money/1706685-nigeria-banks-face-fraud-trend-attacks-bigger-losses/)); mobile-money agents put their own 2024 losses at up to ₦42.6 billion, "mainly through fake alerts" ([Techpoint](https://techpoint.africa/insight/major-hack-and-fraud-cases-in-nigeria/)). Most vendor losses are never reported.
- **Oversold dishes.** The portion count lives in the cook's head. Two customers order the last pan of biryani in the same minute; one gets a refund argument.
- **Lost orders.** "I want" never becomes "I've paid", or a transfer lands with no name and nobody can match it to a chat.
- **Time.** A kitchen like this answers the same three questions, "what do you have", "how much with delivery", "I've sent it, check", perhaps 100 to 200 times a day, typed by the same person who is cooking (illustrative, not measured).

### Why the vendor would pay

If the assistant takes even half the repetitive messages and stops one oversold dish a day, that is an hour of the owner's time and one refund argument a day. A flat fee of a few thousand naira a month, or a small per-order fee, is cheap against that. The alternative, a delivery aggregator, takes a percentage of every order and puts the kitchen under someone else's brand; this keeps TPlus on the WhatsApp number its customers already have.

### What the agent solves

It shows today's menu with live portion counts, takes the order, prices it with the delivery fee, sends a Paystack link, confirms the order only when Paystack says the money arrived, reduces the portions, and gives the owner a board to watch from the kitchen. It never confirms an order that was not paid, and nobody at TPlus answers "how much is jollof" again.

### Deliberately left out

Delivery dispatch and rider tracking, refunds, event catering and food trays (TPlus's other business line; the assistant hands those to the owner), multiple vendors, building the menu from photos, customer accounts, Pidgin or Yoruba beyond a light touch, and gadget-vendor needs (serial numbers, warranties, pay on delivery). Gadget vendors are the stated next vertical: same engine, more trust and stock rules.

---

## 2. What the agent does

One teammate on Chord, one skill, four tools over MCP. The conversation, as it ran in the simulator on 27 October after the vendor switch (screenshots linked):

| Customer | Teammate | |
|---|---|---|
| "Good evening, what do you have tonight?" | Calls `get_menu`. Lists the eight items with prices and "only 2 left" where it matters, the three delivery zones with fees, and free pickup in Oke-Fia. | [05](docs/screenshots/05-simulator-menu.jpg) |
| "2 jollof and 1 chapman, deliver to 12 Gbongan Road, Oke-Fia. My name is Tobi." | Calls `create_order`. Replies with the lines, delivery ₦500, total ₦8,500, the Paystack link, its expiry, and "Order TPK-0231 · ref TPK-0231-5efc". | [06](docs/screenshots/06-simulator-order-link.jpg) |
| "paid" (before actually paying) | Calls `verify_payment`. "I haven't seen your payment yet, so I'm not treating the order as paid." Never the word "failed". (This reply also carried a stray apology; field note 5.10.) | [07](docs/screenshots/07-simulator-paid-before-paying.jpg) |
| Pays on Paystack's test checkout (NGN 8,730.97 with the customer-borne fee) | Lands on a "Payment received, reply paid on WhatsApp" page showing TPK-0231. | [02](docs/screenshots/02-paid-page-live.jpg) |
| "paid now" | Calls `verify_payment` then `confirm_order`. "Payment received, Tobi. Your order is confirmed. Order TPK-0231 … ready by about 7:15pm, and the rider will call you." | [08](docs/screenshots/08-simulator-confirmed.jpg) |
| "Also add 5 biryani for pickup, same name" | `create_order` refuses with `item_unavailable, max_qty: 2`. "We only have 2 chicken biryani left tonight, so I can't do 5. 2 at ₦4,000 each, or a swap." | [09](docs/screenshots/09-simulator-stock-guard.jpg) |

On the panel the same order is [TPK-0231](docs/screenshots/10-order-detail-live.jpg): timeline created → link issued → confirmed (by the reconciler, 18:32) → verified (by the model, 18:32), two successful Paystack checks, four tool calls. The [activity screen](docs/screenshots/11-activity-live.jpg) lists every call with its latency.

### The four tools

| Tool | What it does | External call |
|---|---|---|
| `get_menu` | Current window, whether orders are open, every offered item with price and portions left, delivery zones, pickup address | none |
| `create_order` | Validates items and quantities against stock and the cut-off, prices the cart in kobo, writes a pending order, creates the Paystack transaction, returns the link. Idempotent for the same cart | Paystack `transaction/initialize` |
| `verify_payment` | Asks Paystack about a reference; maps the answer to `success`, `not_confirmed`, `failed` or `amount_mismatch` | Paystack `transaction/verify` |
| `confirm_order` | Verifies again, locks the order and the menu rows, decrements portions, marks paid with a ready-by time. Idempotent on the reference. A sold-out line after payment becomes `paid_needs_attention` for the vendor | Paystack `transaction/verify` |

### Rules the code enforces regardless of what the model does

- Money is integer kobo everywhere. Every money field is returned twice, as `*_kobo` and as a display string the model repeats.
- An order is confirmed only when Paystack's verify says success and the requested amount and currency match.
- Stock moves once per reference, inside a transaction with row locks. Two customers paying for the last portion at the same instant: one is confirmed, the other is flagged for the vendor.
- "Pending" and "abandoned" are "not confirmed yet", never "failed".
- The model cannot set a price, a status or a count. Its arguments are validated and normalised (item by id or name, zone by id or name in any spelling); its errors are full sentences with a `next_step`.
- A reconciler re-verifies pending orders every minute, so a customer who pays and never says "paid" is still confirmed.

---

## 3. How it is built

```
WhatsApp simulator ──► Chord teammate (skill: food-vendor, rev 7)
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
| Database | Postgres on Railway | Schema, the vendor and the menu seeded by migrations (0005 switched the vendor to TPlus without touching existing orders); `version` columns for optimistic concurrency |
| Redis | Railway | Every use fails open; correctness lives in Postgres |
| Admin panel | React 19, TypeScript, Vite, TanStack Router + Query, Tailwind v4, motion | Light mode, mobile-first, live updates over SSE, polling when degraded. Screens: [board](docs/screenshots/01-board-desktop-live.jpg), [order](docs/screenshots/10-order-detail-live.jpg), [menu](docs/screenshots/04-menu-desktop-live.jpg), [settings](docs/screenshots/12-settings-live.jpg), [activity](docs/screenshots/11-activity-live.jpg), [phone](docs/screenshots/03-phone-board-detail-menu.jpg) |
| Skill | `SKILL.md` + two references | Revision 7, after seven rounds of real-conversation fixes |
| Hosting | Railway (backend, from `/backend`), Netlify (panel, from `/frontend`) | Two deployables, same repo, nothing at the root |

No Paystack webhook: the account is shared with other projects, so payment truth comes from calling verify, and the reconciler covers the silent-payer case. The admin login is bootstrapped at boot from `ADMIN_EMAIL` / `ADMIN_PASSWORD` (keyed on email, rotated when the env value changes, sessions revoked on rotation); there is no endpoint that can mint or reset an admin.

Full designs: [backend + MCP](https://chat-to-order-986.netlify.app/docs/tech-spec-backend.html), [admin panel](https://chat-to-order-986.netlify.app/docs/tech-spec-web.html), and the [build plan](https://chat-to-order-986.netlify.app/docs/plan.html) (sources in [docs/](docs/)).

---

## 4. Testing and results

Tested at every layer before the next was built on it, then again on the hosts, then once more after the vendor switch.

| Layer | What ran | Result |
|---|---|---|
| Backend unit + integration | `go test -race`: money, windows, Paystack mapping, alias decoding, zone-by-name; real-Postgres tests for create/verify/confirm, idempotent cart, validation matrix, mismatch and declined, Paystack down, admin transitions, six goroutines racing for the last two portions | green |
| End-to-end smoke, local | 52 checks through `/mcp` and `/api/v1` with the fake checkout | 52 / 52 |
| End-to-end smoke, live on Railway | Same with the real Paystack test key | 46 / 46 |
| Chord registration | Tools discovered (4), health probe `ok` | ok |
| Real conversations | 7 turns by API and 12 turns in the simulator across the day, three real Paystack test payments, all confirmed | confirmed |
| Admin panel | Signed in on the live site; every screen at desktop and 390 px; CORS verified | ok |
| Credits | 16 teammate requests in total | 1.74 of 50 used |

Tool latency on the live service for TPK-0231: `get_menu` 4 to 13 ms, `create_order` 1,329 ms (one Paystack initialize) then 5 ms for the idempotent repeat, `verify_payment` 336 ms, `confirm_order` 11 ms.

---
## 5. Go-live note for the business owner

**What it does.** Customers message your WhatsApp number. The assistant shows today's menu from your admin panel, takes the order, sends a Paystack payment link, and confirms the order only after Paystack says the money arrived. Your portion counts go down by themselves. You watch orders arrive on the board, on your phone or laptop.

**What it does not do.** It does not send riders, give refunds, handle complaints about food already delivered, take event or tray bookings, or add dishes by itself. You set the menu, prices and portions each morning in the panel. If a dish runs out after someone has paid, the order turns red as "Needs you" and you call the customer to swap it or refund by transfer yourself.

**Before real customers use it.**
1. Your own Paystack business account with live keys and your settlement bank account. Today it runs on test keys; test cards work, real cards do not.
2. Your WhatsApp Business number connected to Chord. Today it runs in Chord's simulator.
3. Real prices, portions, zones and the pickup address in the panel, and the service windows set to your real hours.
4. A Chord credit top-up and somebody watching the balance. When credits run out the assistant goes silent.
5. A one-week trial with five regulars you trust, with you watching the board.

**When a payment fails.** The customer sees the bank's reason and gets a fresh link. Nothing is recorded and no food is reserved. If a customer closes the payment page without finishing, the link stays open for 30 minutes, then the order expires on its own.

**When something breaks.** If Paystack or the ordering system is down, the assistant tells the customer nothing was charged and to try again in a minute, and the Activity page shows the error. If the assistant cannot answer at all, customers get no reply; call Feranmi. Alerts to your phone when an order needs you or the service is down are the first thing to add after launch.

**Next.** Adding dishes by sending a photo of your handwritten menu, a WhatsApp alert when an order needs you, and event-tray enquiries routed to you with the details already collected. Gadget sellers are the next vertical: same assistant, plus stock counts, warranties and pay-on-delivery.

---

## 6. Links

**Live**
- WhatsApp simulator chat with the teammate: https://chat.tsglabs.dev/c/161x1iBUglWnrJc3APYog6u-
- Admin panel: https://chat-to-order-986.netlify.app (login `devferanmi@gmail.com` / `tplus-demo-2026`)
- Backend health: https://backend-production-c54f.up.railway.app/healthz
- MCP endpoint (bearer-protected): https://backend-production-c54f.up.railway.app/mcp

**Plan and designs (hosted)**
- Build plan: https://chat-to-order-986.netlify.app/docs/plan.html
- Backend and MCP technical design: https://chat-to-order-986.netlify.app/docs/tech-spec-backend.html
- Admin panel technical design: https://chat-to-order-986.netlify.app/docs/tech-spec-web.html

**Code and documents**
- Repository: https://github.com/spiderocious/chat-to-order
- Design sources: [docs/plan.html](docs/plan.html), [docs/tech-spec-backend.html](docs/tech-spec-backend.html), [docs/tech-spec-web.html](docs/tech-spec-web.html)
- Screenshots: [docs/screenshots/](docs/screenshots/)
- The skill: [backend/skill/food-vendor/](backend/skill/food-vendor/)

**The business**
- TPlus Kitchen on Instagram: https://www.instagram.com/teeplus_events/
- TPlus on X: https://x.com/Tplusevent

**Reference**
- Chord API docs: https://api.chordhq.xyz/docs
- Paystack transaction API: https://paystack.com/docs/api/transaction/
- Paystack test payments: https://paystack.com/docs/payments/test-payments/

---

## 7. Run it yourself

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
- Panel: Netlify site with base `frontend`, `netlify.toml` in that folder, `VITE_API_BASE_URL` set to the Railway domain. The build copies `docs/*.html` into the site under `/docs/`.
- Chord: `backend/scripts/chord/00-create-teammate.sh` then `01-register.sh`, with `CHORD_API_KEY`, `CHORD_WORKSPACE_ID`, `MCP_URL` and `MCP_BEARER_TOKEN` in the environment.

---

## 8. Repository layout

```
README.md                         this document
backend/                          Go service: cmd/server, internal/{config,money,clock,store,redisx,paystack,ordering,reconcile,activity,mcpserver,httpapi}
backend/skill/food-vendor/        SKILL.md + references/vendor-rules.md + references/conversation-examples.md
backend/scripts/                  smoke.sh, chord/00-create-teammate.sh, chord/01-register.sh, chord/02-usage.sh, chord/03-chat.sh
frontend/                         Vite app: src/routes, src/features/{auth,live,board,orders,menu,settings,activity}, src/shared/{api,ui,utils}
docs/                             plan.html, tech-spec-backend.html, tech-spec-web.html, screenshots/
```

---