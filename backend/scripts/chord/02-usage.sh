#!/usr/bin/env bash
# Print Chord credit usage so we know what a turn costs.
set -euo pipefail
BASE=${CHORD_BASE_URL:-https://api.chordhq.xyz}
: "${CHORD_API_KEY:?}"
curl -sS "$BASE/v1/usage?include_breakdowns=true" -H "Authorization: Bearer $CHORD_API_KEY" | jq '{request_count, debited_credits, quoted_credits, by_teammate: (.by_teammate // [] | map({teammate_id, requests, debited_credits}))}'
