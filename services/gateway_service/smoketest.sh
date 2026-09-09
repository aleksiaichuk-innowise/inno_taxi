#!/bin/sh
# Integration smoke test for Gateway Service, run against a live
# docker-compose stack (this is README's "Testing: Integration tests
# using HTTP clients" for a service with no Go code to unit test).
#
# Run from inside the compose network, e.g.:
#   docker compose up -d mongo redis user_service auth_service driver_service gateway_service
#   docker run --rm --network innotaxi_default \
#     -v "$(pwd)/services/gateway_service/smoketest.sh:/smoketest.sh" \
#     curlimages/curl:8.10.1 sh /smoketest.sh
#
# Talks to the gateway (GATEWAY, default gateway_service:8000) for
# everything a real client would go through, and directly to
# driver_service's internal-only endpoint (DRIVER_SERVICE) only to seed a
# driver record - that endpoint is deliberately not reachable through the
# gateway, so a real client could never do this step.

set -eu

GATEWAY="${GATEWAY:-http://gateway_service:8000}"
DRIVER_SERVICE="${DRIVER_SERVICE:-http://driver_service:8081}"

fail() { echo "FAIL: $1" >&2; exit 1; }

json_field() { sed -n "s/.*\"$1\":\"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p"; }

echo "== /healthz =="
code=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/healthz")
[ "$code" = 200 ] || fail "expected /healthz=200, got $code"

echo "== register a fresh user through the gateway =="
# $$ is useless for uniqueness here: this script's shell is always PID 1
# inside its own container, so every run would collide on the same email
# against Mongo's data volume, which persists across container restarts.
SUFFIX=$(od -An -N4 -tu4 /dev/urandom | tr -d ' \n')
EMAIL="gw-smoketest-$SUFFIX@example.com"
# The phone number is uniquely indexed too - it needs to vary run-to-run
# just as much as the email does.
PHONE="+375$((SUFFIX % 900000000 + 100000000))"
REGISTER_RESP=$(curl -s -X POST "$GATEWAY/users/register" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"Gateway Smoketest\",\"email\":\"$EMAIL\",\"phone\":\"$PHONE\",\"password\":\"password123\",\"role\":\"User\"}")
USER_ID=$(echo "$REGISTER_RESP" | json_field id)
[ -n "$USER_ID" ] || fail "register did not return an id: $REGISTER_RESP"

echo "== login through the gateway =="
LOGIN_RESP=$(curl -s -X POST "$GATEWAY/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"login\":\"$EMAIL\",\"password\":\"password123\"}")
ACCESS=$(echo "$LOGIN_RESP" | json_field access_token)
[ -n "$ACCESS" ] || fail "login did not return an access_token: $LOGIN_RESP"

echo "== /auth/login rate limit trips under a burst =="
saw_429_or_503=0
i=0
while [ "$i" -lt 30 ]; do
    code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/auth/login" \
        -H 'Content-Type: application/json' \
        -d "{\"login\":\"$EMAIL\",\"password\":\"wrongpass1\"}")
    if [ "$code" = 503 ] || [ "$code" = 429 ]; then
        saw_429_or_503=1
        break
    fi
    i=$((i + 1))
done
[ "$saw_429_or_503" = 1 ] || fail "expected a burst of /auth/login requests to eventually be rate-limited"

echo "== /users/profile without a token is rejected by the gateway (never reaches user_service) =="
code=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/users/profile")
[ "$code" = 401 ] || fail "expected /users/profile without auth=401, got $code"

echo "== /users/profile with a token routes through =="
PROFILE_RESP=$(curl -s -w '\n%{http_code}' "$GATEWAY/users/profile" -H "Authorization: Bearer $ACCESS")
code=$(echo "$PROFILE_RESP" | tail -n1)
body=$(echo "$PROFILE_RESP" | sed '$d')
[ "$code" = 200 ] || fail "expected /users/profile with auth=200, got $code ($body)"
echo "$body" | grep -q "\"id\":\"$USER_ID\"" || fail "profile response did not echo the logged-in user: $body"

echo "== admin-only route rejects a plain 'user' token (role gate via auth_service) =="
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/users/admin/users/$USER_ID/analyst-role" -H "Authorization: Bearer $ACCESS")
[ "$code" = 403 ] || fail "expected admin route with a user-role token=403, got $code"

echo "== seed a driver record directly against driver_service's internal endpoint (never through the gateway) =="
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$DRIVER_SERVICE/internal/drivers" \
    -H 'Content-Type: application/json' \
    -d "{\"user_id\":\"$USER_ID\",\"type\":\"economy\"}")
[ "$code" = 201 ] || fail "expected driver seed=201, got $code"

echo "== /drivers/profile without a token is rejected =="
code=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/drivers/profile")
[ "$code" = 401 ] || fail "expected /drivers/profile without auth=401, got $code"

echo "== /drivers/profile with the same user's token routes through =="
DRIVER_RESP=$(curl -s -w '\n%{http_code}' "$GATEWAY/drivers/profile" -H "Authorization: Bearer $ACCESS")
code=$(echo "$DRIVER_RESP" | tail -n1)
body=$(echo "$DRIVER_RESP" | sed '$d')
[ "$code" = 200 ] || fail "expected /drivers/profile with auth=200, got $code ($body)"
echo "$body" | grep -q "\"user_id\":\"$USER_ID\"" || fail "driver profile did not match the logged-in user: $body"

echo "== internal-only endpoints are not reachable through the gateway =="
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/users/internal/verify-credentials" -H 'Content-Type: application/json' -d '{}')
[ "$code" = 404 ] || fail "expected /users/internal/verify-credentials through the gateway=404, got $code"

echo
echo "ALL GATEWAY SMOKE TESTS PASSED"
