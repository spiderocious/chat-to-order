#!/usr/bin/env bash
# Register the MCP server in Chord, assign it to the teammate, enable it, probe
# tools and health, and restrict the teammate to the four tools.
# Reads: CHORD_API_KEY, CHORD_WORKSPACE_ID, CHORD_TEAMMATE_ID, MCP_URL, MCP_BEARER_TOKEN
# Optional: CHORD_BASE_URL (default https://api.chordhq.xyz), MCP_NAME
set -euo pipefail
BASE=${CHORD_BASE_URL:-https://api.chordhq.xyz}
: "${CHORD_API_KEY:?}" "${CHORD_WORKSPACE_ID:?}" "${CHORD_TEAMMATE_ID:?}" "${MCP_URL:?}" "${MCP_BEARER_TOKEN:?}"
NAME=${MCP_NAME:-"Iya Bisi ordering"}
H=(-H "Authorization: Bearer $CHORD_API_KEY" -H 'Content-Type: application/json')

echo "== 1. register (disabled)"
REG=$(curl -sS -X POST "$BASE/v1/workspaces/$CHORD_WORKSPACE_ID/mcp-servers" "${H[@]}" --data "$(jq -n --arg n "$NAME" --arg u "$MCP_URL" --arg t "$MCP_BEARER_TOKEN" \
  '{name:$n,url:$u,auth_type:"bearer",auth_token:$t,transport:"streamable_http",enabled:false}')")
echo "$REG" | jq .
ID=$(echo "$REG" | jq -r '.id // empty')
if [ -z "$ID" ]; then
  echo "registration failed; if 409, list existing servers:"; curl -sS "$BASE/v1/workspaces/$CHORD_WORKSPACE_ID/mcp-servers" "${H[@]}" | jq .
  exit 1
fi
echo "server id: $ID"

echo "== 2. assign to teammate"
curl -sS -X PUT "$BASE/v1/teammates/$CHORD_TEAMMATE_ID" "${H[@]}" --data "{\"set_mcp_server_ids\":[\"$ID\"]}" | jq '{id, mcp_server_ids}'

echo "== 3. enable"
curl -sS -X PUT "$BASE/v1/workspaces/$CHORD_WORKSPACE_ID/mcp-servers/$ID" "${H[@]}" --data '{"enabled":true}' | jq '{id,enabled,url}'

echo "== 4. discover tools"
curl -sS -X POST "$BASE/v1/workspaces/$CHORD_WORKSPACE_ID/mcp-servers/$ID/tools" "${H[@]}" | jq '[.[].name]'

echo "== 5. health (read status even on HTTP 200)"
curl -sS -X POST "$BASE/v1/workspaces/$CHORD_WORKSPACE_ID/mcp-servers/$ID/health" "${H[@]}" | jq .

echo "== 6. restrict to the four tools"
curl -sS -X PUT "$BASE/v1/workspaces/$CHORD_WORKSPACE_ID/mcp-servers/$ID" "${H[@]}" --data '{"enabled_tools":["get_menu","create_order","verify_payment","confirm_order"]}' | jq '{id,enabled,enabled_tools}'

echo; echo "done. CHORD_MCP_SERVER_ID=$ID"
