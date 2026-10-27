#!/usr/bin/env bash
# End-to-end smoke test through the MCP tools and the admin API.
# Usage: BASE=http://localhost:8080 MCP_TOKEN=... ADMIN_EMAIL=... ADMIN_PASSWORD=... scripts/smoke.sh
set -euo pipefail
BASE=${BASE:-http://localhost:8080}
MCP_TOKEN=${MCP_TOKEN:?}
ADMIN_EMAIL=${ADMIN_EMAIL:?}
ADMIN_PASSWORD=${ADMIN_PASSWORD:?}
FAKE=${FAKE:-1}
pass=0; failn=0
RUN=$RANDOM   # unique customer names so reruns never hit an older open order
check() { if [ "$2" = "$3" ]; then pass=$((pass+1)); echo "  ok   $1 = $2"; else failn=$((failn+1)); echo "  FAIL $1: got '$2' want '$3'"; fi; }
mcp() { curl -s "$BASE/mcp" -H "Authorization: Bearer $MCP_TOKEN" -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"$1\",\"arguments\":$2}}"; }
tooltext() { jq -r '.result.content[0].text'; }

echo "== health"
H=$(curl -s "$BASE/healthz"); check "healthz.ok" "$(echo "$H" | jq -r .ok)" "true"

echo "== mcp auth"
check "mcp.no_bearer" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/mcp" -H 'Content-Type: application/json' -d '{}')" "401"
check "mcp.get" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/mcp" -H "Authorization: Bearer $MCP_TOKEN")" "405"

echo "== tools/list"
TL=$(curl -s "$BASE/mcp" -H "Authorization: Bearer $MCP_TOKEN" -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}')
check "tools.count" "$(echo "$TL" | jq '.result.tools | length')" "4"
check "tools.names" "$(echo "$TL" | jq -r '[.result.tools[].name] | sort | join(",")')" "confirm_order,create_order,get_menu,verify_payment"

echo "== get_menu"
M=$(mcp get_menu '{}' | tooltext)
check "menu.vendor" "$(echo "$M" | jq -r .vendor_name)" "Iya Bisi's Kitchen"
check "menu.orders_open" "$(echo "$M" | jq -r .orders_open)" "true"
check "menu.has_jollof" "$(echo "$M" | jq -r '[.items[].id] | index("jollof") != null')" "true"
check "menu.price_display" "$(echo "$M" | jq -r '.items[] | select(.id=="jollof") | .price_display')" "₦3,500"
JOLLOF_BEFORE=$(echo "$M" | jq -r '.items[] | select(.id=="jollof") | .portions_left')

echo "== create_order validation"
E=$(mcp create_order '{"customer_name":"Tobi","items":[{"item_id":"nope","qty":1}],"fulfilment":"pickup"}' | tooltext)
check "create.item_unknown" "$(echo "$E" | jq -r .error)" "item_unknown"
E=$(mcp create_order '{"customer_name":"Tobi","items":[{"item_id":"ofada","qty":5}],"fulfilment":"pickup"}' | tooltext)
check "create.item_unavailable" "$(echo "$E" | jq -r .error)" "item_unavailable"
check "create.max_qty_present" "$(echo "$E" | jq -r '.max_qty | type')" "number"
E=$(mcp create_order '{"customer_name":"Tobi","items":[{"item_id":"jollof","qty":1}],"fulfilment":"delivery"}' | tooltext)
check "create.zone_required" "$(echo "$E" | jq -r .error)" "zone_required"
E=$(mcp create_order '{"customer_name":"Tobi","items":[{"item_id":"jollof","qty":1}],"fulfilment":"delivery","zone_id":"surulere"}' | tooltext)
check "create.address_required" "$(echo "$E" | jq -r .error)" "address_required"

echo "== create_order lenient argument names"
L=$(mcp create_order '{"customer":"Lenient '$RUN'","items":[{"id":"Jollof rice","quantity":1}],"fulfillment":"Delivery","zone":"Surulere","delivery_address":"5 Ogunlana Drive"}' | tooltext)
check "lenient.total" "$(echo "$L" | jq -r .total_display)" "₦4,200"
check "lenient.by_order_no" "$(mcp verify_payment "{\"order_no\":\"$(echo "$L" | jq -r .order_no)\"}" | tooltext | jq -r .outcome)" "not_confirmed"

echo "== create_order happy"
CART='{"customer_name":"Tobi '$RUN'","items":[{"item_id":"jollof","qty":2},{"item_id":"chapman","qty":1}],"fulfilment":"delivery","zone_id":"surulere","address":"22 Bode Thomas, Surulere","phone":"08012345678"}'
O=$(mcp create_order "$CART" | tooltext)
check "create.total" "$(echo "$O" | jq -r .total_display)" "₦8,700"
check "create.fee" "$(echo "$O" | jq -r .delivery_fee_display)" "₦700"
check "create.reused" "$(echo "$O" | jq -r .reused)" "false"
REF=$(echo "$O" | jq -r .reference); NO=$(echo "$O" | jq -r .order_no); URL=$(echo "$O" | jq -r .authorization_url)
echo "  order $NO ref $REF"
check "create.ref_prefix" "${REF:0:${#NO}}" "$NO"
O2=$(mcp create_order "$CART" | tooltext)
check "create.idempotent" "$(echo "$O2" | jq -r .reused)" "true"
check "create.same_ref" "$(echo "$O2" | jq -r .reference)" "$REF"

echo "== verify before paying"
V=$(mcp verify_payment "{\"reference\":\"$REF\"}" | tooltext)
check "verify.not_confirmed" "$(echo "$V" | jq -r .outcome)" "not_confirmed"
C=$(mcp confirm_order "{\"reference\":\"$REF\"}" | tooltext)
check "confirm.refuses_unpaid" "$(echo "$C" | jq -r .error)" "payment_not_confirmed"

if [ "$FAKE" = "1" ]; then
  echo "== pay (fake checkout page)"
  check "paid.page" "$(curl -s -o /dev/null -w '%{http_code}' "$URL")" "200"
  V=$(mcp verify_payment "{\"reference\":\"$REF\"}" | tooltext)
  check "verify.success" "$(echo "$V" | jq -r .outcome)" "success"
  C=$(mcp confirm_order "{\"reference\":\"$REF\"}" | tooltext)
  check "confirm.paid" "$(echo "$C" | jq -r .status)" "paid"
  check "confirm.ready_by" "$(echo "$C" | jq -r '.ready_by_wat | length')" "5"
  C2=$(mcp confirm_order "{\"reference\":\"$REF\"}" | tooltext)
  check "confirm.idempotent" "$(echo "$C2" | jq -r .already_confirmed)" "true"
  M2=$(mcp get_menu '{}' | tooltext)
  check "stock.decremented" "$(echo "$M2" | jq -r '.items[] | select(.id=="jollof") | .portions_left')" "$((JOLLOF_BEFORE-2))"
fi
check "verify.unknown" "$(mcp verify_payment '{"reference":"IBK-9999-zzzz"}' | tooltext | jq -r .error)" "unknown_reference"

echo "== admin api"
check "login.bad" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"wrong\"}")" "401"
TOK=$(curl -s "$BASE/api/v1/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" | jq -r .data.token)
check "login.token" "$([ ${#TOK} -gt 20 ] && echo yes)" "yes"
A="Authorization: Bearer $TOK"
check "me" "$(curl -s "$BASE/api/v1/auth/me" -H "$A" | jq -r .data.admin.email)" "$ADMIN_EMAIL"
check "unauth" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/orders")" "401"
check "order.detail" "$(curl -s "$BASE/api/v1/orders/$NO" -H "$A" | jq -r .data.order_no)" "$NO"
check "order.events" "$(curl -s "$BASE/api/v1/orders/$NO" -H "$A" | jq -r '.data.events | length > 2')" "true"
check "order.404" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/orders/IBK-9999" -H "$A")" "404"
check "orders.list" "$(curl -s "$BASE/api/v1/orders?limit=2" -H "$A" | jq -r '.meta.has_more')" "true"
CUR=$(curl -s "$BASE/api/v1/orders?limit=2" -H "$A" | jq -r '.meta.next_cursor')
check "orders.cursor" "$(curl -s "$BASE/api/v1/orders?limit=2&cursor=$CUR" -H "$A" | jq -r '.data | length > 0')" "true"
check "stats" "$(curl -s "$BASE/api/v1/stats/today" -H "$A" | jq -r '.data.paid_count >= 0')" "true"
check "activity" "$(curl -s "$BASE/api/v1/activity?limit=5" -H "$A" | jq -r '.data[0].tool | length > 0')" "true"
V1=$(curl -s "$BASE/api/v1/menu" -H "$A" | jq -r '.data.items[] | select(.id=="water") | .version')
P=$(curl -s "$BASE/api/v1/menu/water" -X PATCH -H "$A" -H 'Content-Type: application/json' -H "If-Match: \"$V1\"" -d '{"portions":39}')
check "patch.ok" "$(echo "$P" | jq -r .data.portions)" "39"
check "patch.stale" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/menu/water" -X PATCH -H "$A" -H 'Content-Type: application/json' -H "If-Match: \"$V1\"" -d '{"portions":38}')" "409"
check "patch.validation" "$(curl -s "$BASE/api/v1/menu/water" -X PATCH -H "$A" -H 'Content-Type: application/json' -d '{"portions":-1}' | jq -r .error.code)" "validation"
# make a fresh pending order to cancel so the check does not depend on seed state
PEND=$(mcp create_order '{"customer_name":"Smoke Cancel '$RUN'","items":[{"item_id":"water","qty":1}],"fulfilment":"pickup"}' | tooltext | jq -r .order_no)
K=$(uuidgen)
R1=$(curl -s "$BASE/api/v1/orders/$PEND/cancel" -X POST -H "$A" -H "Idempotency-Key: $K" -H 'Content-Type: application/json' -d '{"reason":"smoke"}')
check "cancel.ok" "$(echo "$R1" | jq -r .data.status)" "cancelled"
check "cancel.replay" "$(curl -s "$BASE/api/v1/orders/$PEND/cancel" -X POST -H "$A" -H "Idempotency-Key: $K" -H 'Content-Type: application/json' -d '{"reason":"smoke"}' | jq -r .data.status)" "cancelled"
check "cancel.twice" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/orders/$PEND/cancel" -X POST -H "$A" -H 'Content-Type: application/json' -d '{"reason":"again"}')" "409"
check "resolve.wrong_state" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/orders/$NO/resolve" -X POST -H "$A" -H 'Content-Type: application/json' -d '{"note":"x"}')" "409"
check "sse" "$(curl -s -N -m 2 "$BASE/api/v1/events" -H "$A" | head -c 200 | grep -c 'event: hello')" "1"
check "logout" "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/v1/auth/logout" -X POST -H "$A")" "204"
check "revoked" "$(curl -s "$BASE/api/v1/auth/me" -H "$A" | jq -r .error.code)" "token_revoked"

echo; echo "passed $pass, failed $failn"
[ "$failn" = "0" ]
