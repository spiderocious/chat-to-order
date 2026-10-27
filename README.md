# Chat to paid order

A Chord AI teammate on WhatsApp for a Lagos food vendor. A customer asks what's on the menu, orders, gets a Paystack link, pays, and the order is confirmed with stock reduced. The vendor watches it happen on an admin panel.

Built for the TSG Labs build exercise, role: Forward-deployed Engineering.

| Part | Where | Stack | Deploys to |
|---|---|---|---|
| Backend + MCP server | [`backend/`](backend/) | Go, Gin, Postgres, Redis, MCP Go SDK | Railway |
| Admin panel | [`frontend/`](frontend/) | React, TypeScript, TanStack Router + Query, Tailwind | Netlify |
| Chord skill | [`backend/skill/food-vendor/`](backend/skill/food-vendor/) | SKILL.md + references | Chord |
| Docs | [`docs/`](docs/) | plan, technical designs, QA log, field notes | |

## Run locally

```bash
cd backend
cp .env.example .env            # PAYSTACK_FAKE=1 by default; no Paystack account needed
docker compose up -d            # Postgres on 5433, Redis on 6380
make dev                        # migrates, seeds, listens on :8080
make test                       # go vet + go test -race (needs docker compose up)
make smoke                      # 50 end-to-end checks through MCP and the admin API
```

```bash
cd frontend
cp .env.example .env            # VITE_API_BASE_URL=http://localhost:8080
pnpm install && pnpm dev        # http://localhost:5173
```

## How AI was used

Claude Code wrote the plan and technical designs from the brief, scaffolded the backend and panel from the tool contracts, and drafted the docs. Feranmi made the product decisions, ran every Chord, Paystack and hosting step, debugged the deployment, and edited everything sent to TSG.
