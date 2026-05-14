#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:5000}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
CURL_MAX_TIME="${CURL_MAX_TIME:-45}"

EMAIL="${EMAIL:-}"
AUTO_EMAIL=0
PASSWORD="${PASSWORD:-Passw0rd!}"
DISPLAY_NAME="${DISPLAY_NAME:-}"
FIRST_NAME="${FIRST_NAME:-Test}"
LAST_NAME="${LAST_NAME:-User}"

SHOP_NAME="${SHOP_NAME:-Green Shop}"
SHOP_DESCRIPTION="${SHOP_DESCRIPTION:-Fresh daily}"
SHOP_ADDRESS="${SHOP_ADDRESS:-123 Main St}"
SHOP_LATITUDE="${SHOP_LATITUDE:-10.762622}"
SHOP_LONGITUDE="${SHOP_LONGITUDE:-106.660172}"

PRODUCT_NAME="${PRODUCT_NAME:-Morning Greens}"
PRODUCT_DESCRIPTION="${PRODUCT_DESCRIPTION:-Fresh vegetables for smoke test}"
PRODUCT_CATEGORY="${PRODUCT_CATEGORY:-fresh_produce}"
PRODUCT_TAG="${PRODUCT_TAG:-leafy}"
PRODUCT_PRICE="${PRODUCT_PRICE:-25000}"
PRODUCT_CURRENCY="${PRODUCT_CURRENCY:-VND}"
PRODUCT_STATUS="${PRODUCT_STATUS:-published}"

PLEDGE_SCORE="${PLEDGE_SCORE:-8.5}"
PLEDGE_CATEGORY="${PLEDGE_CATEGORY:-fresh_produce}"
PLEDGE_CONFIDENCE="${PLEDGE_CONFIDENCE:-0.91}"
PLEDGE_IMAGE_HASH="${PLEDGE_IMAGE_HASH:-manual-test-image-hash}"
BUNDLE_ID="${BUNDLE_ID:-bundle-${RANDOM}-$(date +%s)}"

REPORT_SCORE="${REPORT_SCORE:-7.2}"
REPORT_CATEGORY="${REPORT_CATEGORY:-fresh_produce}"
REPORT_CONFIDENCE="${REPORT_CONFIDENCE:-0.88}"
REPORT_COMMENT="${REPORT_COMMENT:-Buyer smoke-test report}"

IMAGE_PATH="${IMAGE_PATH:-}"

have_cmd() { command -v "$1" >/dev/null 2>&1; }

json_get() {
  local key="$1"
  if have_cmd jq; then
    jq -r ".${key} // empty"
  else
    sed -nE "s/.*\"${key}\"[[:space:]]*:[[:space:]]*\"?([^\",}]*)\"?.*/\1/p" | head -n 1
  fi
}

step() {
  printf '\n== %s ==\n' "$1"
}

LAST_BODY=""
LAST_STATUS=""

request() {
  local method="$1"
  local path="$2"
  shift 2

  local raw
  raw="$(curl -sS \
    --connect-timeout "$CURL_CONNECT_TIMEOUT" \
    --max-time "$CURL_MAX_TIME" \
    -X "$method" \
    "$BASE_URL$path" \
    "$@" \
    -w $'\n__HTTP_STATUS__:%{http_code}')"

  LAST_STATUS="$(printf '%s' "$raw" | sed -n 's/^__HTTP_STATUS__://p' | tail -n1)"
  LAST_BODY="$(printf '%s' "$raw" | sed '/^__HTTP_STATUS__:/d')"
}

expect_status() {
  local expected="$1"
  if [[ "$LAST_STATUS" != "$expected" ]]; then
    echo "ERROR: expected HTTP $expected, got $LAST_STATUS" >&2
    echo "Body: $LAST_BODY" >&2
    exit 1
  fi
}

BASE_JSON=(-H "Content-Type: application/json")

step "config"
echo "BASE_URL=$BASE_URL"
echo "EMAIL=${EMAIL:-<auto>}"
echo "SHOP_NAME=$SHOP_NAME"
echo "PRODUCT_NAME=$PRODUCT_NAME"
echo "IMAGE_PATH=${IMAGE_PATH:-<empty>}"

if [[ -z "$EMAIL" ]]; then
  EMAIL="test+${RANDOM}$(date +%s)@example.com"
  AUTO_EMAIL=1
fi

if [[ -z "$DISPLAY_NAME" ]]; then
  DISPLAY_NAME="${LAST_NAME} ${FIRST_NAME}"
fi

step "health"
request GET /health
expect_status 200
echo "$LAST_BODY"

step "register"
register_payload="$(cat <<JSON
{"email":"$EMAIL","password":"$PASSWORD","displayName":"$DISPLAY_NAME","firstName":"$FIRST_NAME","lastName":"$LAST_NAME"}
JSON
)"
request POST /v1/auth/register "${BASE_JSON[@]}" -d "$register_payload"
echo "register status=$LAST_STATUS"
echo "$LAST_BODY"

if [[ "$LAST_STATUS" != "201" ]]; then
  # If auto-generated email accidentally collides, rotate once then retry register.
  if [[ "$AUTO_EMAIL" == "1" ]] && [[ "$LAST_BODY" == *"email is already registered"* ]]; then
    EMAIL="test+${RANDOM}$(date +%s)@example.com"
    register_payload="$(cat <<JSON
{"email":"$EMAIL","password":"$PASSWORD","displayName":"$DISPLAY_NAME","firstName":"$FIRST_NAME","lastName":"$LAST_NAME"}
JSON
)"
    request POST /v1/auth/register "${BASE_JSON[@]}" -d "$register_payload"
    echo "register retry status=$LAST_STATUS"
    echo "$LAST_BODY"
  fi
fi

if [[ "$LAST_STATUS" != "201" ]]; then
  if [[ "$LAST_STATUS" =~ ^5 ]]; then
    echo "ERROR: register failed by server/infrastructure; skip login fallback" >&2
    exit 1
  fi
  echo "register failed -> try login with same credentials" >&2
  request POST /v1/auth/login "${BASE_JSON[@]}" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
  echo "login status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 200
fi

auth_body="$LAST_BODY"
token="$(printf '%s' "$auth_body" | json_get accessToken || true)"
if [[ -z "$token" ]]; then
  echo "ERROR: cannot extract accessToken" >&2
  echo "$auth_body" >&2
  exit 1
fi
AUTH=(-H "Authorization: Bearer $token")

step "me"
request GET /v1/me "${AUTH[@]}"
expect_status 200
echo "$LAST_BODY"

step "create shop"
request POST /v1/shops "${AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"name\":\"$SHOP_NAME\",\"description\":\"$SHOP_DESCRIPTION\",\"address\":\"$SHOP_ADDRESS\",\"latitude\":$SHOP_LATITUDE,\"longitude\":$SHOP_LONGITUDE}"
expect_status 201
echo "$LAST_BODY"
shop_id="$(printf '%s' "$LAST_BODY" | json_get shopId || true)"
[[ -n "$shop_id" ]] || { echo "ERROR: missing shopId" >&2; exit 1; }

step "create second shop (expect 409)"
request POST /v1/shops "${AUTH[@]}" "${BASE_JSON[@]}" -d '{"name":"Duplicate Shop","address":"Nowhere"}'
expect_status 409
echo "$LAST_BODY"

step "create product"
request POST "/v1/shops/$shop_id/products" "${AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"name\":\"$PRODUCT_NAME\",\"description\":\"$PRODUCT_DESCRIPTION\",\"category\":\"$PRODUCT_CATEGORY\",\"tags\":[\"$PRODUCT_TAG\"],\"freshnessNote\":\"Fresh this morning\",\"freshnessScore\":8.4,\"price\":$PRODUCT_PRICE,\"currency\":\"$PRODUCT_CURRENCY\",\"status\":\"$PRODUCT_STATUS\"}"
expect_status 201
echo "$LAST_BODY"
product_id="$(printf '%s' "$LAST_BODY" | json_get productId || true)"
[[ -n "$product_id" ]] || { echo "ERROR: missing productId" >&2; exit 1; }

step "list products"
request GET "/v1/shops/$shop_id/products?category=$PRODUCT_CATEGORY&tag=$PRODUCT_TAG&sort=freshness_desc"
expect_status 200
echo "$LAST_BODY"

commit_image_hash="$PLEDGE_IMAGE_HASH"
if [[ -n "$IMAGE_PATH" ]]; then
  [[ -f "$IMAGE_PATH" ]] || { echo "ERROR: IMAGE_PATH not found: $IMAGE_PATH" >&2; exit 1; }
  step "seller score"
  request POST /v1/seller/score "${AUTH[@]}" -F "image=@$IMAGE_PATH"
  expect_status 200
  echo "$LAST_BODY"
  scored_hash="$(printf '%s' "$LAST_BODY" | json_get imageHash || true)"
  [[ -n "$scored_hash" ]] && commit_image_hash="$scored_hash"
else
  step "seller score skipped"
  echo "Set IMAGE_PATH to test image flows"
fi

step "seller commit"
request POST /v1/seller/commit "${AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"shopId\":\"$shop_id\",\"productId\":\"$product_id\",\"bundleId\":\"$BUNDLE_ID\",\"score\":$PLEDGE_SCORE,\"category\":\"$PLEDGE_CATEGORY\",\"confidence\":$PLEDGE_CONFIDENCE,\"imageHash\":\"$commit_image_hash\"}"
expect_status 201
echo "$LAST_BODY"
pledge_id="$(printf '%s' "$LAST_BODY" | json_get pledgeId || true)"
bundle_id="$(printf '%s' "$LAST_BODY" | json_get bundleId || true)"
bundle_token="$(printf '%s' "$LAST_BODY" | json_get bundleToken || true)"
[[ -n "$pledge_id" ]] || { echo "ERROR: missing pledgeId" >&2; exit 1; }
[[ -n "$bundle_id" ]] || bundle_id="$BUNDLE_ID"

step "pledge history"
request GET "/v1/shops/$shop_id/pledges?productId=$product_id&category=$PLEDGE_CATEGORY"
expect_status 200
echo "$LAST_BODY"

step "pledge integrity"
request GET "/v1/shops/$shop_id/pledges/$pledge_id/integrity"
expect_status 200
echo "$LAST_BODY"

if [[ -n "$IMAGE_PATH" ]]; then
  if [[ -z "$bundle_token" ]]; then
    step "reissue bundle token"
    request POST "/v1/shops/$shop_id/pledges/$pledge_id/bundle-token" "${AUTH[@]}"
    expect_status 200
    echo "$LAST_BODY"
    bundle_token="$(printf '%s' "$LAST_BODY" | json_get bundleToken || true)"
  fi

  [[ -n "$bundle_token" ]] || { echo "ERROR: missing bundleToken" >&2; exit 1; }

  step "buyer check"
  request POST /v1/buyer/check "${AUTH[@]}" \
    -F "pledgeId=$pledge_id" \
    -F "bundleId=$bundle_id" \
    -F "bundleToken=$bundle_token" \
    -F "image=@$IMAGE_PATH"
  expect_status 200
  echo "$LAST_BODY"

  step "create freshness report"
  request POST "/v1/shops/$shop_id/products/$product_id/freshness-reports" "${AUTH[@]}" "${BASE_JSON[@]}" \
    -d "{\"score\":$REPORT_SCORE,\"category\":\"$REPORT_CATEGORY\",\"confidence\":$REPORT_CONFIDENCE,\"comment\":\"$REPORT_COMMENT\",\"imageHash\":\"$commit_image_hash\"}"
  expect_status 201
  echo "$LAST_BODY"

  step "list freshness reports"
  request GET "/v1/shops/$shop_id/products/$product_id/freshness-reports"
  expect_status 200
  echo "$LAST_BODY"
else
  step "buyer/image flows skipped"
fi

step "done"
echo "email=$EMAIL"
echo "shopId=$shop_id"
echo "productId=$product_id"
echo "pledgeId=$pledge_id"
