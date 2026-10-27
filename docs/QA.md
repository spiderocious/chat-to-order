# QA log

Progressive testing as the build went: each layer tested before the next was built on it, then the whole thing again on the hosts. Dates are 27 Oct 2026 WAT.

## 1. Backend, local (Docker Postgres + Redis, fake Paystack)

- `go vet`, `go test -race ./...`: unit tests for money, clock windows, Paystack status mapping and typed errors, argument alias decoding; integration tests on a real Postgres schema per test: create → verify → confirm, idempotent cart, validation matrix, amount mismatch and declined card, Paystack down, admin transitions, and six goroutines confirming the last two portions (exactly 2 paid, 4 needs-attention, stock ends at 0).
- `scripts/smoke.sh` locally: **52 checks, 0 failures** through `/mcp` (tools/list, all four tools, lenient argument names, idempotent cart, unknown reference, unpaid confirm refused, pay on the fake page, verify success, confirm, double confirm, stock decrement) and `/api/v1` (login bad and good, me, 401 without token, order detail and 404, cursor pagination, stats, activity, PATCH with If-Match and stale 409, validation 400, cancel with Idempotency-Key replay and 409 on repeat, resolve wrong state 409, SSE hello frame, logout and revoked token).

## 2. Backend, live on Railway (real Paystack test key)

- `/healthz`: db ok, redis ok, paystack configured, ready.
- `scripts/smoke.sh` with `FAKE=0`: **44 checks, 0 failures** (the fake-checkout section is skipped against the real Paystack).
- First live run found two real bugs (field notes 2 and 3): the `.example` payer email and the lunch-only headline dish after 16:00. Both fixed and re-verified.

## 3. Chord: teammate, skill, MCP registration

- Teammate created by API, skill uploaded and published (after the `skills/migrate` precondition, field note 4), revisions 1 to 4 published during testing.
- MCP server registered, assigned, enabled. Tool discovery returned the four tools; health probe `{"status":"ok","latency_ms":252}`; `enabled_tools` restricted to the four.

## 4. Conversation through Chord's chat API (same credits as the simulator)

| Turn | Customer | What the teammate did | Result |
|---|---|---|---|
| 1 | "Good afternoon, what do you have today?" | `get_menu`, read `vendor-rules.md` | Listed evening items with exact prices and counts |
| 2 | "2 jollof and 1 chapman, deliver to 22 Bode Thomas, Surulere. My name is Tobi." | `create_order` ×3, all schema-rejected (`id`, `quantity`) | Leaked narration to the customer. Field note 1. Fixed: lenient schema + aliases + skill rule |
| 3 | "Sorry, did that go through? …" | `get_menu`, `create_order` (one alias miss, then success, `reused: true`) | Link sent with lines and ₦8,700 total |
| 4 | Paid on Paystack checkout (test "Success" option) | | Checkout showed NGN 8,934.02: fee passed to customer. Field note 2 |
| 5 | "paid" | `verify_payment` → `amount_mismatch` | Refused to confirm, told the customer plainly, flagged the vendor. Correct behaviour on the old comparison; fix deployed (`requested_amount`) |
| 6 | "I've paid, can you check again?" | `verify_payment` with the example reference from the skill → `unknown_reference` | Confabulated a retraction. Field note 7. Fixed: references only from this conversation; order number in every link reply |
| 7 | "I did pay. The payment page says order IBK-0235." | `verify_payment` by order number → success; `confirm_order` → `paid`, `already_confirmed: true, confirmed_by: reconciler` | Confirmed with order number, lines, address, ready-by 17:15. The reconciler had confirmed it first (field note 8) |

Credits after 7 turns: 0.75 of 50.

## 5. Admin panel, local against the local backend

- Login, board (tiles, today's orders, stock strip, activity feed), orders list with filters and cursor "Load more", order detail with timeline and payment checks, menu with inline stepper and switch, settings, activity with expandable args/result. Live chip reads "Live"; an order paid through the smoke flips on the board without a reload.
- Bugs found and fixed: lines summary showed "0.02×" (quantity formatted as money); a deep link bounced to login because the route loader fetched before the token was in memory; the board overflowed horizontally at 390 px; the Ofada row's price collided with the stepper when the Low chip appeared.
- Phone width checked at 390 px through an iframe viewport: bottom tab bar, 2×2 tiles, two-line order rows, menu rows wrap. No page-level horizontal scroll after the fix.

Screenshots in [`qa/screenshots/`](qa/screenshots/).

## 6. Admin panel, live on Netlify

- Site deployed from `frontend/dist` to https://chat-to-order-986.netlify.app with `VITE_API_BASE_URL` pointing at Railway.
- CORS verified by curl: preflight 204 with `access-control-allow-origin: https://chat-to-order-986.netlify.app`; a real POST carries the same header.
- Sign-in on the live site is left to the owner (password in `.deploy.env`); every screen was verified against the same API locally.

## 7. WhatsApp simulator (chat.tsglabs.dev)

Connected Apps enabled, the simulator app allowed, chat link created for the teammate: https://chat.tsglabs.dev/c/jIMwkWWy_4MoJukWyBnCZ3h1. Conversation as the customer "Ngozi", 17:32 to 17:36 WAT, on skill revision 4:

| Customer | Teammate | Evidence |
|---|---|---|
| "Good evening, what do you have tonight?" | Evening menu with prices and portions left, delivery zones, pickup address, 40-minute prep | `05-simulator-menu.jpg` |
| "1 fried rice and 2 moi moi, delivery to 7 Herbert Macaulay Way, Yaba. I'm Ngozi." | Lines, delivery ₦1,200, total ₦6,300, Paystack link, "Order IBK-0240 · ref IBK-0240-01bd", link expiry | `06-simulator-order-link.jpg` |
| "paid" (before paying) | "Your payment isn't confirmed yet… the link is still open until 6:03 PM" | `07-simulator-paid-before-paying.jpg` |
| Paid on checkout (test Success; page showed NGN 6,497.47 with the customer-borne fee) | Landed on the "Payment received" page with the order number | `02-paid-page-live.jpg` |
| "paid now" | "Your payment is confirmed… Order IBK-0240… ready by about 6:20 PM. The rider will call you." | `08-simulator-confirmed.jpg` |

Admin API for IBK-0240 afterwards: `status: paid`, `confirmed_by: reconciler`, three payment checks (mcp not_confirmed, reconciler success, mcp success), stock reduced. The reconciler confirmed the order between the payment and the customer's next message; the model's `confirm_order` returned `already_confirmed: true`.

One cosmetic finding: the teammate used markdown bold (`**Total**`) in a WhatsApp-style chat. Skill revision 5 adds a plain-text rule.
