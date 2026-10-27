SHELL := /bin/bash
export TEST_DATABASE_URL ?= postgres://postgres:dev@localhost:5433/chat_to_order?sslmode=disable

.PHONY: dev test smoke skill lint build

dev: ## run locally with fake Paystack against docker compose
	docker compose up -d
	set -a; source .env; set +a; go run ./cmd/server

build:
	go build ./...

test: ## unit + integration (needs docker compose up)
	go vet ./...
	go test -race -count=1 ./...

smoke: ## end-to-end through MCP and the admin API; BASE overrides the target
	set -a; source .env; set +a; BASE=$${BASE:-http://localhost:8080} MCP_TOKEN=$$MCP_BEARER_TOKEN ADMIN_EMAIL=$$ADMIN_EMAIL ADMIN_PASSWORD=$$ADMIN_PASSWORD FAKE=$${FAKE:-1} scripts/smoke.sh

skill: ## zip the Chord skill package
	mkdir -p dist && rm -f dist/food-vendor.zip && (cd skill && zip -qr ../dist/food-vendor.zip food-vendor -x '*.DS_Store') && ls -la dist/food-vendor.zip
