# Deployment field notes

Problems hit while building and deploying the agent, in the order they happened. Each one: what happened, how the cause was found, what fixed it, and what Chord (or Paystack, or Railway) could change so the next builder does not hit it. Logged as they happened on 27 Oct 2026.

## 1. Chord's model guessed the tool argument names, and the customer saw the retries

**What happened.** First real order through Chord. The teammate called `create_order` three times and every call failed schema validation:

```
validating "arguments": validating root: required: missing properties: ["customer_name" "items" "fulfilment"]
validating /properties/items/items: unexpected additional properties ["id"]
validating /properties/items/items: unexpected additional properties ["quantity"]
```

Then it gave up and sent its own narration to the customer: *"Trying the quantity field under a different name."*

**How I found the cause.** The `steps` array on `POST /v1/external-chat/messages` shows every tool call with its result text, so the three attempts were visible in one response. The MCP Go SDK infers a JSON schema from the Go input struct with `additionalProperties: false`, so `id` instead of `item_id` and `quantity` instead of `qty` were hard rejections with validator text as the error, which the model then tried to reason about.

**Fix.** Two changes. The server now publishes an explicit, lenient schema (an example in the description, no `additionalProperties: false`, no hard `required` list) and the handler normalises aliases (`id`/`item`/`name` → `item_id`, `quantity`/`count` → `qty`, `customer`/`name` → `customer_name`, `fulfillment` → `fulfilment`, a bare order number where a reference is expected). Validation errors are now full sentences with a `next_step`. The skill gained a hard rule: never narrate tool attempts to the customer; fix the arguments silently, and after two failures say "something went wrong on my side". The next conversation succeeded on the second call and the customer saw a clean reply.

**What I'd change in Chord.** Show the tool's `inputSchema` example to the model more prominently, or let the server flag a tool as "lenient" so Chord passes near-miss arguments through. And treat a reply that contains tool-retry narration as a failed turn rather than delivering it; the customer should never read the model's inner voice.

## 2. Paystack passes its fee to the customer, so the amount on verify is not the order total

**What happened.** The order total was ₦8,700. Paystack checkout showed **NGN 8,934.02**. After paying, `verify_payment` returned `outcome: amount_mismatch`, the guard refused to confirm, and the customer was told Iya Bisi would be in touch.

**How I found the cause.** The checkout page itself showed the larger figure. Paystack's verify response carries `amount` (what the customer was charged, fee included when the merchant's settings put the fee on the customer), `requested_amount` (what we asked for) and `fees`. My code compared `amount` with the order total.

**Fix.** Compare the order total against `requested_amount` (falling back to `amount` when absent) and record both plus `fees` on the payment check. A unit test pins the fee-bearer case. The same order confirmed on the next "paid".

**What I'd change.** For builders: Paystack's docs for verify should put `requested_amount` next to `amount` with one sentence on fee bearer. For Chord: the built-in Paystack connector should document which figure it checks, because a merchant that moves the fee to the customer will see the same thing.

## 3. Paystack rejects a `.example` payer email

**What happened.** Live `create_order` failed with `paystack_rejected: "email" must be a valid email.` The server sent `orders+ibk-0231@iyabisi.example` because customers never give an email on WhatsApp and `.example` is the reserved documentation TLD.

**How I found the cause.** The MCP tool returned Paystack's own message verbatim, and the Railway log line for the call had the same text. Local runs use a fake Paystack client, so the first real initialize call was on the deployed service.

**Fix.** A configurable `ORDER_EMAIL_DOMAIN` with a valid default; the per-order address is still synthetic (`orders+ibk-0235@iyabisi-kitchen.com`) and nothing is emailed.

**What I'd change.** Paystack: accept RFC 2606 reserved domains in test mode, or say in the initialize docs that the TLD is validated. For me: run one real initialize before calling the fake "good enough".

## 4. Skills API refuses a new skill until the teammate is "migrated"

**What happened.** `POST /v1/teammates/{id}/skills` on a teammate created seconds earlier by API returned `400 FAILED_PRECONDITION: Initialize skills from existing instructions before modifying them`.

**How I found the cause.** The skills guide mentions `POST .../skills/migrate` for "legacy" teammates, but a brand-new teammate created with a `persona` is also legacy in this sense. Calling migrate turned the persona into an "Existing instructions" skill and unlocked the endpoint.

**Fix.** Create → migrate → create skill → publish → set the migrated placeholder skill `included: false`. Scripted in `backend/scripts/chord/00-create-teammate.sh`.

**What I'd change in Chord.** Auto-migrate on first skill create, or let `POST /v1/teammates` accept skill files directly. Also, two responses from the skills API contain raw control characters inside JSON string values (the file contents), which strict JSON parsers reject; `jq` and Python both failed until I parsed leniently.

## 5. Railway: the GitHub app had no access to the new repo, and `railway up` from a subfolder breaks the root directory

**What happened.** The first deployment built from the repo root (no Dockerfile there) and failed. After setting the root directory to `backend` through the GraphQL API, `railway up` from inside `backend/` uploaded that folder as the snapshot root, so the builder looked for `backend/backend` and failed again. A push to `main` did not trigger a build at all: `railway service source connect` answered `User does not have access to the repo`, because the Railway GitHub app was not granted the newly created repository.

**How I found the cause.** `railway logs --build` for each deployment id, and the GraphQL `serviceInstance` query to confirm `rootDirectory` and `source.repo`.

**Fix.** `railway up` from the repository root with the service's root directory set to `backend`. Auto-deploy on push needs the GitHub app granted on the repo, which is a browser-only step for the account owner.

**What I'd change.** Railway CLI: a `--root-directory` flag on `railway add`/`service`, and a clear message when `up` is run from a subfolder of a service with a root directory.

## 6. The demo depends on the time of day

**What happened.** The live smoke test failed 13 checks at 16:05 WAT because the headline dish was seeded as lunch-only and the lunch window had closed. A reviewer trying the agent in the evening would have been told jollof is not available.

**How I found the cause.** `get_menu` on the live service listed only evening items; the smoke test asserted on jollof.

**Fix.** A migration makes jollof and fried rice all-day; ofada stays lunch-only and egusi evening-only so the window rule is still demonstrable. The smoke test also uses unique customer names per run so it never collides with an older open order.

**What I'd change.** Nothing on Chord's side. For me: seed demo data for the reviewer's clock, not the builder's.

## 7. The model used the example reference from the skill as if it were real

**What happened.** Two turns after the order was created, the customer said "I've paid, can you check again?". The teammate called `verify_payment` with `IBK-0231-a1b2`, which is the placeholder reference in my skill's conversation examples and in the tool description, not the real `IBK-0235-82e2`. The tool answered `unknown_reference`. The model then told the customer it had never created the order and that the payment link "didn't come from the kitchen's system", which was false.

**How I found the cause.** The `steps` for that turn show the exact argument. Looking back, the reply that carried the payment link never included the reference (the model had dropped it), and tool results are not part of the conversation history on later turns, so the only `IBK-` strings the model could see were the examples.

**Fix.** Three changes. Every reply that sends a link must carry the order number and reference on their own line, so later turns can find them. All examples now use obvious placeholders (`IBK-0000-xxxx`) and say so. The hard rules gained: a reference only ever comes from `create_order` in this conversation or from the order number the customer reads on the payment page (the tools accept the order number alone); and never retract or claim an order does not exist unless a tool result proves it. `unknown_reference` now tells the model exactly what to ask the customer. On the next turn the customer quoted the order number, and the teammate verified and confirmed it correctly.

**What I'd change in Chord.** Keep tool results (or at least the tool's `next_step` and ids) in the context of later turns in the same conversation, or expose a small key-value memory per conversation that tools can write to. Today a builder has to smuggle state through the model's own replies.

## 8. The reconciler confirmed the order before the model did

**What happened.** After the fee fix deployed, the next "paid" from the customer led to `confirm_order` returning `already_confirmed: true, confirmed_by: reconciler`. The background reconciler had re-verified the pending order on its one-minute tick and confirmed it first.

**Why it matters.** This is the no-webhook design working: a shared Paystack account cannot point a webhook at this service, so payment truth comes only from calling verify, and the reconciler covers customers who pay and never say so. The model path and the reconciler path converged on the same row under the row lock, with one stock decrement.

**What I'd change.** Nothing; this is the behaviour I wanted to see. The go-live note explains the up-to-three-minute lag.
