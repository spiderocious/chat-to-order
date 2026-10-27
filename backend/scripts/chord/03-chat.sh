#!/usr/bin/env bash
# Send one customer message to the teammate through Chord's external chat API
# and print the reply plus the tool steps. Same conversation = same SESSION.
# Usage: SESSION=<uuid> SENDER=customer-1 scripts/chord/03-chat.sh "message text"
set -euo pipefail
BASE=${CHORD_BASE_URL:-https://api.chordhq.xyz}
: "${CHORD_API_KEY:?}" "${CHORD_WORKSPACE_ID:?}" "${CHORD_TEAMMATE_ID:?}" "${SESSION:?}"
SENDER=${SENDER:-customer-1}
curl -sS "$BASE/v1/external-chat/messages" -H "Authorization: Bearer $CHORD_API_KEY" -H 'Content-Type: application/json' -H "X-Request-ID: $(uuidgen)" \
  --data "$(jq -n --arg w "$CHORD_WORKSPACE_ID" --arg t "$CHORD_TEAMMATE_ID" --arg s "$SENDER" --arg sess "$SESSION" --arg text "$1" --arg name "${DISPLAY_NAME:-Tobi}" \
   '{workspace_id:$w, teammate_id:$t, sender_id:$s, session_id:$sess, display_name:$name, text:$text}')" \
  | jq '{reply, agent_invoked, reason, steps: (.steps // [] | map({title, tool_name, success, latency_ms, result: (.result|tostring|.[0:200])}))}'
