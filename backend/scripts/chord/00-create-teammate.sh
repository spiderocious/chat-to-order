#!/usr/bin/env bash
# Create the teammate, upload the skill package, publish it, activate.
# Reads: CHORD_API_KEY, CHORD_WORKSPACE_ID. Optional: CHORD_BASE_URL, TEAMMATE_NAME, TEAMMATE_EMAIL, SKILL_DIR
set -euo pipefail
BASE=${CHORD_BASE_URL:-https://api.chordhq.xyz}
: "${CHORD_API_KEY:?}" "${CHORD_WORKSPACE_ID:?}"
NAME=${TEAMMATE_NAME:-"Iya Bisi's Kitchen"}
EMAIL=${TEAMMATE_EMAIL:-"iya-bisi-orders@chat-to-order.chord"}
SKILL_DIR=${SKILL_DIR:-"$(cd "$(dirname "$0")/../../skill/food-vendor" && pwd)"}
H=(-H "Authorization: Bearer $CHORD_API_KEY" -H 'Content-Type: application/json')

echo "== 1. create teammate"
PERSONA="You are the ordering assistant for Iya Bisi's Kitchen on WhatsApp. Follow the food-vendor skill exactly: every price, total, reference and time comes from a tool result; never confirm an order before confirm_order returns status paid."
TM=$(curl -sS -X POST "$BASE/v1/teammates" "${H[@]}" --data "$(jq -n --arg e "$EMAIL" --arg n "$NAME" --arg w "$CHORD_WORKSPACE_ID" --arg p "$PERSONA" \
  '{email:$e, first_name:"Iya Bisi", last_name:"Orders", display_name:$n, role:"ordering_assistant", workspace_id:$w, persona:$p, start_paused:false, creation_request_id:("ctpo-"+($n|ascii_downcase|gsub("[^a-z0-9]+";"-")))}')")
echo "$TM" | jq '{id, display_name, workspace_id, lifecycle_state}'
TID=$(echo "$TM" | jq -r '.id // empty')
[ -n "$TID" ] || { echo "create failed: $TM"; exit 1; }

echo "== 2. upload skill files"
FILES=$(cd "$SKILL_DIR" && find . -type f \( -name '*.md' -o -name '*.txt' \) | sed 's#^\./##' | sort | while read -r f; do jq -n --arg p "$f" --rawfile c "$f" '{path:$p, content:$c}'; done | jq -s .)
SK=$(curl -sS -X POST "$BASE/v1/teammates/$TID/skills" "${H[@]}" --data "$(jq -n --argjson files "$FILES" '{name:"food-vendor", entrypoint:"SKILL.md", files:$files, idempotency_key:"ctpo-skill-v1"}')")
echo "$SK" | jq '{id, name, version, published_version, included}' 2>/dev/null || echo "$SK"
SID=$(echo "$SK" | jq -r '.id // empty'); VER=$(echo "$SK" | jq -r '.version // 1')
[ -n "$SID" ] || { echo "skill create failed: $SK"; exit 1; }

echo "== 3. publish"
curl -sS -X POST "$BASE/v1/teammates/$TID/skills/$SID/publish" "${H[@]}" --data "$(jq -n --argjson v "$VER" '{expected_version:$v, summary:"First version: menu, order, pay, verify, confirm", idempotency_key:"ctpo-skill-publish-v1"}')" | jq '{id, version, published_version, included}' 2>/dev/null

echo "== 4. activate"
curl -sS -X POST "$BASE/v1/teammates/$TID/lifecycle" "${H[@]}" --data '{"action":"activate"}' | jq '{id, lifecycle_state}' 2>/dev/null
curl -sS "$BASE/v1/teammates/$TID" "${H[@]}" | jq '{id, display_name, lifecycle_state, mcp_server_ids}'
echo; echo "CHORD_TEAMMATE_ID=$TID"
