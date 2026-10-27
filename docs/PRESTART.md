# Pre-start: everything I need before writing code

Fill the blanks at the bottom (section 5). Secrets go in `.secrets.env` at the repo root, never in this file. When section 5 is complete, say "start" and I begin stage 1 without asking anything else until deploy.

What I already have on this machine, verified today: Go 1.27, Node 25, pnpm, Docker (running), Railway CLI (logged in as devferanmi@gmail.com, team "adewaleadeniji's Projects"), Netlify CLI (logged in as aoadeniji81@student.lautech.edu.ng, team "AdewaleAdeniji's team"), `gh` (logged in as spiderocious).

---

## 1. Accounts and clicks you do once

Most of the clicking I can do in your Chrome (section 4). What stays with you is anything that is a secret, and being signed in.

### 1.1 Paystack (plan §10)
1. Sign in at https://dashboard.paystack.com. Create an account if you don't have one; test mode needs no business verification.
2. Turn on **Test Mode** (toggle beside the business name; the banner turns orange).
3. **Settings → API Keys & Webhooks** → reveal and copy the **Test Secret Key** (`sk_test_…`).
4. Put it in `.secrets.env` (section 5). Do not change the webhook URL; we don't use one.
5. Note for the recording: success test card `4084 0840 8408 4081`, CVV `408`, any future expiry, PIN `0000`, OTP `123456`. Check https://paystack.com/docs/payments/test-payments/ on the day for a declined card.

### 1.2 Chord (plan §11)
Only one thing by hand: **be signed in to https://www.chordhq.xyz with devferanmi@gmail.com in Chrome** and leave the tab open. I drive your Chrome for the rest: Connected Apps toggle, simulator sign-in, reading the credit balance, creating the teammate from the ZIP, publishing, activating, copying the ids, and running the test conversation.

One exception, because it is a secret and should not pass through my context: **create the API key yourself.** Admin Panel → API keys (or Developers) → new key named `chat-to-order` with scopes `teammate:read`, `teammate:update`, `mcp:create`, `mcp:read`, `mcp:manage`, `orchestrator:chat`, `thread:read`, `usage:read` (or the profile that includes MCP management). Paste it into `.secrets.env`. I use it for the register / probe / usage scripts, which are faster and more precise than the UI and produce output I can paste into the field notes.

### 1.3 Railway
Nothing to click. I will create the project `chat-to-order`, the Postgres and Redis services, the backend service from `/backend`, and generate the domain, all with the CLI. You add nothing by hand; I set every variable except `PAYSTACK_SECRET_KEY`, which I take from `.secrets.env` and set with `railway variables set`.

### 1.4 Netlify
Nothing to click. I will run `netlify sites:create` for `chat-to-order` under your team and `netlify link` inside `/frontend`, then deploy with `netlify deploy --prod`. Env var `VITE_API_BASE_URL` set by CLI.

### 1.5 GitHub
Nothing to click. I will run `gh repo create spiderocious/chat-to-order --public` and push. Confirm the name and visibility in section 5.

---

## 2. Decisions (plan §16, updated)

Defaults are what I'll use unless you change them in section 5.

| # | Decision | Default |
|---|---|---|
| D1 | Vendor name | Iya Bisi's Kitchen |
| D2 | Pickup address | 14 Adelabu Street, Surulere, Lagos |
| D3 | Menu (8 items) | Jollof rice & chicken ₦3,500 (20) · Fried rice & chicken ₦3,500 (15) · Ofada rice & ayamase ₦4,000 (2) · Beans & plantain ₦2,500 (12) · Moi moi ₦800 (30) · Egusi & pounded yam ₦4,500 evening only (10) · Chapman ₦1,000 (25) · Bottled water ₦300 (40) |
| D4 | Delivery zones | Surulere ₦700 · Yaba ₦1,200 · Ikeja ₦1,800 · pickup free |
| D5 | Windows | Lunch 11:00–15:00, orders close 14:30 · Evening 17:00–20:30, orders close 20:00 · prep 40 min |
| D6 | Order prefix | IBK |
| D7 | Teammate name in Chord | Iya Bisi's Kitchen |
| D8 | MCP server name in Chord | Iya Bisi ordering |
| D9 | GitHub repo | spiderocious/chat-to-order, public |
| D10 | Railway project | chat-to-order (Postgres + Redis + backend) |
| D11 | Netlify site | chat-to-order |
| D12 | Admin login email | devferanmi@gmail.com |
| D13 | Admin display name | Iya Bisi |
| D14 | Seed demo orders on the deployed instance | yes (6 orders so the board has variety) |
| D15 | Stop building at credits used | 35 of 50 |

---

## 3. What I do myself, in order, once you say "start"

1. Create `.secrets.env.example`, `backend/`, `frontend/` skeletons, `README.md`; `git init`; create the GitHub repo; first push.
2. Backend stage 1 (about 2 h): schema, migrations, seed, domain, Paystack client + fake, four MCP tools, REST + SSE, Redis, reconciler, tests green locally against Docker Postgres and Redis.
3. Skill package `food-vendor.zip`.
4. Railway: project, Postgres, Redis, backend service, variables, domain, deploy, `/healthz` green.
5. In your Chrome: Chord Connected Apps, simulator sign-in, create the teammate from the ZIP, publish, activate. Then by API: register the MCP server, assign, enable, probe tools and health, set the four-tool allowlist. Probe output goes straight into the field notes.
6. Frontend stage 5: build, Netlify site, deploy, CORS origin set on Railway.
7. QA in your Chrome (section 4), then field notes written as things break, then README, go-live note, problem statement.
8. Arrange the windows and reset the demo for your recording.

## 4. QA and the parts I run in your Chrome

I have browser tools and will use them with the qa-frontend and qa-backend personas:

- **Chord:** every click in plan §11 Passes A–D, including creating the teammate from the ZIP, checking the Instructions badge, publishing, activating, and the "Why this reply" inspector when a reply looks wrong.
- **Simulator:** the full conversation from plan §02, the "paid before paying" path, the sold-out path.
- **Paystack checkout:** open the link, pay with the test card, return and say "paid". I also open the Paystack Transactions page to confirm the record.
- **Admin panel QA:** every screen in all four states, at 390 px and 1440 px, keyboard-only pass, and the live-update check (an order paid in the simulator flips on the board without a reload).
- **Backend QA:** `make smoke` against the deployed URL, the Chord health probe, cursor pagination walk, 409 on stale `If-Match`.

Credits: each simulator turn spends Chord credits. I run the conversation locally against the fake Paystack first, then once in the simulator for real, then once more for the recording. Hard stop at D15.

**The one thing only you do:** the 3-minute screen recording (plan §12), with your voice. I will have the windows arranged and the demo state reset when you sit down.

---

## 5. Fill these in

### 5.1 `.secrets.env` (repo root, gitignored; create it with exactly these lines)

```
PAYSTACK_SECRET_KEY=sk_test_
CHORD_API_KEY=chord_live_
```

### 5.2 Non-secret values (write here)

```
Chrome Dev: signed in to chordhq.xyz as devferanmi@gmail.com:        yes
Chrome Dev: signed in to dashboard.paystack.com:                     yes (Test Mode: to confirm)
Recording: silent screen capture by me, or narrated by you:          ____
```

### 5.3 Decisions you want to change (leave empty to accept all defaults in section 2)

```
D__ : ______
D__ : ______
```

### 5.4 Permissions (write yes or no)

```
I may create the GitHub repo spiderocious/chat-to-order (public):   yes
I may create the Railway project, Postgres, Redis and backend service: yes
I may create the Netlify site and deploy to it:                       yes
I may set Railway and Netlify environment variables, including the
  Paystack key read from .secrets.env:                                yes
```

### 5.5 Start time

```
Say "start" in chat when 5.1, 5.2 and 5.4 are done. If the Chord API key is the
only thing missing, say "start" anyway; I need it at step 5 of section 3 (about H+2:30).
```
