#!/bin/sh
# Integration smoke test for Wallet Service, run against a live
# docker-compose stack - complements repository/pg_repo's dockertest
# suite (which proves the DB-level guarantees) by proving the HTTP
# surface actually wires them up correctly end to end.
#
# Migrations aren't run automatically on startup (see the design doc), so
# apply them first, e.g.:
#   docker compose up -d wallet_postgres redis wallet_service
#   for f in services/wallet_service/migrations/postgres/*.sql; do
#     sed '/-- +goose Down/,$d' "$f" | docker exec -i taxi-wallet-postgres psql -U postgres -d wallet
#   done
#   docker run --rm --network innotaxi_default \
#     -v "$(pwd)/services/wallet_service/smoketest.sh:/smoketest.sh" \
#     curlimages/curl:8.10.1 sh /smoketest.sh

set -eu

WALLET_SERVICE="${WALLET_SERVICE:-http://wallet_service:8084}"

fail() { echo "FAIL: $1" >&2; exit 1; }

json_field() { sed -n "s/.*\"$1\":\"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p"; }

SUFFIX=$(od -An -N4 -tu4 /dev/urandom | tr -d ' \n')
USER_ID="smoketest-user-$SUFFIX"

echo "== create wallet (expect 201, balance 0) =="
resp=$(curl -s -w '\n%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\"}")
code=$(echo "$resp" | tail -n1)
body=$(echo "$resp" | sed '$d')
[ "$code" = 201 ] || fail "expected create wallet=201, got $code ($body)"
[ "$(echo "$body" | json_field balance_minor_units)" = 0 ] || fail "expected balance 0, got: $body"

echo "== duplicate create is rejected (expect 422) =="
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets" -H 'Content-Type: application/json' -d "{\"user_id\":\"$USER_ID\"}")
[ "$code" = 422 ] || fail "expected duplicate create=422, got $code"

echo "== charging an empty wallet is rejected (expect 402) =="
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets/$USER_ID/charge" -H 'Content-Type: application/json' -d '{"amount_minor_units":500,"reference_id":"order-1"}')
[ "$code" = 402 ] || fail "expected insufficient-funds charge=402, got $code"

echo "== refund funds the wallet (expect 201) =="
resp=$(curl -s -w '\n%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets/$USER_ID/refund" -H 'Content-Type: application/json' -d '{"amount_minor_units":10000,"reference_id":"topup-1"}')
code=$(echo "$resp" | tail -n1)
[ "$code" = 201 ] || fail "expected refund=201, got $code ($(echo "$resp" | sed '$d'))"

echo "== charge now succeeds (expect 201) =="
resp=$(curl -s -w '\n%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets/$USER_ID/charge" -H 'Content-Type: application/json' -d '{"amount_minor_units":500,"reference_id":"order-1"}')
code=$(echo "$resp" | tail -n1)
body=$(echo "$resp" | sed '$d')
[ "$code" = 201 ] || fail "expected charge=201, got $code ($body)"
first_tx_id=$(echo "$body" | json_field id)

echo "== replaying the same charge is idempotent (expect 200, same transaction id) =="
resp=$(curl -s -w '\n%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets/$USER_ID/charge" -H 'Content-Type: application/json' -d '{"amount_minor_units":500,"reference_id":"order-1"}')
code=$(echo "$resp" | tail -n1)
body=$(echo "$resp" | sed '$d')
[ "$code" = 200 ] || fail "expected idempotent replay=200, got $code ($body)"
replay_tx_id=$(echo "$body" | json_field id)
[ "$replay_tx_id" = "$first_tx_id" ] || fail "expected replay to return the same transaction id ($first_tx_id), got $replay_tx_id"

echo "== balance reflects exactly one charge, not two (expect 9500) =="
resp=$(curl -s -w '\n%{http_code}' "$WALLET_SERVICE/internal/wallets/$USER_ID")
balance=$(echo "$resp" | sed '$d' | json_field balance_minor_units)
[ "$balance" = 9500 ] || fail "expected balance=9500, got $balance"

echo "== invalid amount is rejected (expect 422) =="
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$WALLET_SERVICE/internal/wallets/$USER_ID/charge" -H 'Content-Type: application/json' -d '{"amount_minor_units":-1,"reference_id":"order-2"}')
[ "$code" = 422 ] || fail "expected invalid-amount charge=422, got $code"

echo "== unknown wallet 404s =="
code=$(curl -s -o /dev/null -w '%{http_code}' "$WALLET_SERVICE/internal/wallets/does-not-exist")
[ "$code" = 404 ] || fail "expected unknown wallet=404, got $code"

echo "== transaction history has both entries =="
resp=$(curl -s -w '\n%{http_code}' "$WALLET_SERVICE/internal/wallets/$USER_ID/transactions")
code=$(echo "$resp" | tail -n1)
body=$(echo "$resp" | sed '$d')
[ "$code" = 200 ] || fail "expected transactions=200, got $code ($body)"
echo "$body" | grep -q '"reference_id":"order-1"' || fail "expected the charge to appear in history: $body"
echo "$body" | grep -q '"reference_id":"topup-1"' || fail "expected the refund to appear in history: $body"

echo
echo "ALL WALLET SMOKE TESTS PASSED"
