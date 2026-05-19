#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8081}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
CURL_MAX_TIME="${CURL_MAX_TIME:-45}"

# Backend role model:
# - /auth/register creates role=user unless email is in BOOTSTRAP_ADMIN_EMAILS.
# - Seller capability is ownership-based: a normal user creates a shop, then can
#   create products and pledges for that owned shop.
SELLER_EMAIL="${SELLER_EMAIL:-${EMAIL:-}}"
BUYER_EMAIL="${BUYER_EMAIL:-}"
PASSWORD="${PASSWORD:-Passw0rd!}"
SELLER_FIRST_NAME="${SELLER_FIRST_NAME:-Seller}"
SELLER_LAST_NAME="${SELLER_LAST_NAME:-Smoke}"
BUYER_FIRST_NAME="${BUYER_FIRST_NAME:-Buyer}"
BUYER_LAST_NAME="${BUYER_LAST_NAME:-Smoke}"

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
PRODUCT_STATUS="${PRODUCT_STATUS:-active}"

PLEDGE_SCORE="${PLEDGE_SCORE:-8.5}"
PLEDGE_CATEGORY="${PLEDGE_CATEGORY:-$PRODUCT_CATEGORY}"
PLEDGE_CONFIDENCE="${PLEDGE_CONFIDENCE:-0.91}"
PLEDGE_IMAGE_HASH="${PLEDGE_IMAGE_HASH:-manual-test-image-hash}"
BUNDLE_ID="${BUNDLE_ID:-bundle-${RANDOM}-$(date +%s)}"

REVIEW_RATING="${REVIEW_RATING:-5}"
REVIEW_COMMENT="${REVIEW_COMMENT:-Smoke-test review from buyer account}"

REPORT_SCORE="${REPORT_SCORE:-7.2}"
REPORT_CATEGORY="${REPORT_CATEGORY:-$PRODUCT_CATEGORY}"
REPORT_CONFIDENCE="${REPORT_CONFIDENCE:-0.88}"
REPORT_COMMENT="${REPORT_COMMENT:-Buyer smoke-test report}"

# Optional. If set, the script exercises seller/score and buyer/check.
# Without an image file these flows are skipped because those endpoints require
# multipart image input and a configured vision scorer.
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
REGISTER_TOKEN=""

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

register_or_login() {
  local label="$1"
  local email="$2"
  local first_name="$3"
  local last_name="$4"
  local display_name="$last_name $first_name"
  local payload

  step "$label register/login"
  payload="$(cat <<JSON
{"email":"$email","password":"$PASSWORD","displayName":"$display_name","firstName":"$first_name","lastName":"$last_name"}
JSON
)"
  request POST /v1/auth/register "${BASE_JSON[@]}" -d "$payload"
  echo "register status=$LAST_STATUS"
  echo "$LAST_BODY"

  if [[ "$LAST_STATUS" != "201" ]]; then
    if [[ "$LAST_BODY" == *"email is already registered"* ]]; then
      echo "email exists -> login"
      request POST /v1/auth/login "${BASE_JSON[@]}" -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}"
      echo "login status=$LAST_STATUS"
      echo "$LAST_BODY"
      expect_status 200
    else
      expect_status 201
    fi
  fi

  local token
  token="$(printf '%s' "$LAST_BODY" | json_get accessToken || true)"
  [[ -n "$token" ]] || { echo "ERROR: cannot extract accessToken for $label" >&2; exit 1; }
  REGISTER_TOKEN="$token"
}

step "config"
echo "BASE_URL=$BASE_URL"
echo "SELLER_EMAIL=${SELLER_EMAIL:-<auto>}"
echo "BUYER_EMAIL=${BUYER_EMAIL:-<auto>}"
echo "SHOP_NAME=$SHOP_NAME"
echo "PRODUCT_NAME=$PRODUCT_NAME"
echo "PRODUCT_STATUS=$PRODUCT_STATUS"
echo "IMAGE_PATH=${IMAGE_PATH:-<empty>}"

if [[ -z "$SELLER_EMAIL" ]]; then
  SELLER_EMAIL="seller+${RANDOM}$(date +%s)@example.com"
fi
if [[ -z "$BUYER_EMAIL" ]]; then
  BUYER_EMAIL="buyer+${RANDOM}$(date +%s)@example.com"
fi
if [[ "$SELLER_EMAIL" == "$BUYER_EMAIL" ]]; then
  echo "ERROR: SELLER_EMAIL and BUYER_EMAIL must be different" >&2
  exit 1
fi

step "health"
request GET /health
expect_status 200
echo "$LAST_BODY"

register_or_login "seller-owner" "$SELLER_EMAIL" "$SELLER_FIRST_NAME" "$SELLER_LAST_NAME"
seller_token="$REGISTER_TOKEN"
SELLER_AUTH=(-H "Authorization: Bearer $seller_token")

step "seller me"
request GET /v1/me "${SELLER_AUTH[@]}"
expect_status 200
echo "$LAST_BODY"
seller_user_id="$(printf '%s' "$LAST_BODY" | json_get userId || true)"
seller_role="$(printf '%s' "$LAST_BODY" | json_get role || true)"
[[ -n "$seller_user_id" ]] || { echo "ERROR: missing seller userId" >&2; exit 1; }
echo "seller role=$seller_role (expected backend default: user; seller capability comes from shop ownership)"

step "create owned shop"
request POST /v1/shops "${SELLER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"name\":\"$SHOP_NAME\",\"description\":\"$SHOP_DESCRIPTION\",\"address\":\"$SHOP_ADDRESS\",\"latitude\":$SHOP_LATITUDE,\"longitude\":$SHOP_LONGITUDE}"
expect_status 201
echo "$LAST_BODY"
shop_id="$(printf '%s' "$LAST_BODY" | json_get shopId || true)"
shop_version="$(printf '%s' "$LAST_BODY" | json_get version || true)"
[[ -n "$shop_id" ]] || { echo "ERROR: missing shopId" >&2; exit 1; }

step "create second shop for same owner (expect conflict)"
request POST /v1/shops "${SELLER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d '{"name":"Duplicate Shop","description":"Duplicate","address":"Nowhere","latitude":0,"longitude":0}'
expect_status 409
echo "$LAST_BODY"

step "list public shops"
request GET "/v1/shops?query=$SHOP_NAME"
expect_status 200
echo "$LAST_BODY"

step "get shop"
request GET "/v1/shops/$shop_id"
expect_status 200
echo "$LAST_BODY"

step "create product in owned shop"
request POST "/v1/shops/$shop_id/products" "${SELLER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"name\":\"$PRODUCT_NAME\",\"description\":\"$PRODUCT_DESCRIPTION\",\"category\":\"$PRODUCT_CATEGORY\",\"tags\":[\"$PRODUCT_TAG\"],\"freshnessNote\":\"Fresh this morning\",\"freshnessScore\":8.4,\"price\":$PRODUCT_PRICE,\"currency\":\"$PRODUCT_CURRENCY\",\"status\":\"$PRODUCT_STATUS\"}"
expect_status 201
echo "$LAST_BODY"
product_id="$(printf '%s' "$LAST_BODY" | json_get productId || true)"
product_version="$(printf '%s' "$LAST_BODY" | json_get version || true)"
[[ -n "$product_id" ]] || { echo "ERROR: missing productId" >&2; exit 1; }

step "list public products"
request GET "/v1/shops/$shop_id/products?category=$PRODUCT_CATEGORY&tag=$PRODUCT_TAG&sort=freshness_desc"
expect_status 200
echo "$LAST_BODY"

step "get product"
request GET "/v1/shops/$shop_id/products/$product_id"
expect_status 200
echo "$LAST_BODY"

commit_image_hash="$PLEDGE_IMAGE_HASH"
if [[ -n "$IMAGE_PATH" ]]; then
  [[ -f "$IMAGE_PATH" ]] || { echo "ERROR: IMAGE_PATH not found: $IMAGE_PATH" >&2; exit 1; }
  step "seller score image"
  request POST /v1/seller/score "${SELLER_AUTH[@]}" -F "image=@$IMAGE_PATH"
  expect_status 200
  echo "$LAST_BODY"
  scored_hash="$(printf '%s' "$LAST_BODY" | json_get imageHash || true)"
  [[ -n "$scored_hash" ]] && commit_image_hash="$scored_hash"
else
  step "seller score image skipped"
  echo "Set IMAGE_PATH=/path/to/image.jpg to test /v1/seller/score"
fi

step "seller commit pledge"
request POST /v1/seller/commit "${SELLER_AUTH[@]}" "${BASE_JSON[@]}" \
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

step "pledge proof"
request GET "/v1/shops/$shop_id/pledges/$pledge_id/proof"
expect_status 200
echo "$LAST_BODY"

step "reissue bundle token"
request POST "/v1/shops/$shop_id/pledges/$pledge_id/bundle-token" "${SELLER_AUTH[@]}"
expect_status 200
echo "$LAST_BODY"
reissued_bundle_token="$(printf '%s' "$LAST_BODY" | json_get bundleToken || true)"
[[ -n "$reissued_bundle_token" ]] && bundle_token="$reissued_bundle_token"

register_or_login "buyer" "$BUYER_EMAIL" "$BUYER_FIRST_NAME" "$BUYER_LAST_NAME"
buyer_token="$REGISTER_TOKEN"
BUYER_AUTH=(-H "Authorization: Bearer $buyer_token")

step "buyer me"
request GET /v1/me "${BUYER_AUTH[@]}"
expect_status 200
echo "$LAST_BODY"

step "create shop review as buyer"
request POST "/v1/shops/$shop_id/reviews" "${BUYER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"rating\":$REVIEW_RATING,\"comment\":\"$REVIEW_COMMENT\"}"
expect_status 201
echo "$LAST_BODY"

step "list shop reviews"
request GET "/v1/shops/$shop_id/reviews"
expect_status 200
echo "$LAST_BODY"

step "create product freshness report as buyer"
request POST "/v1/shops/$shop_id/products/$product_id/freshness-reports" "${BUYER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"score\":$REPORT_SCORE,\"category\":\"$REPORT_CATEGORY\",\"confidence\":$REPORT_CONFIDENCE,\"comment\":\"$REPORT_COMMENT\",\"imageHash\":\"$commit_image_hash\"}"
expect_status 201
echo "$LAST_BODY"

step "list product freshness reports"
request GET "/v1/shops/$shop_id/products/$product_id/freshness-reports"
expect_status 200
echo "$LAST_BODY"

if [[ -n "$IMAGE_PATH" ]]; then
  [[ -n "$bundle_token" ]] || { echo "ERROR: missing bundleToken" >&2; exit 1; }
  step "buyer check image"
  request POST /v1/buyer/check "${BUYER_AUTH[@]}" \
    -F "pledgeId=$pledge_id" \
    -F "bundleId=$bundle_id" \
    -F "bundleToken=$bundle_token" \
    -F "locationStatus=near" \
    -F "image=@$IMAGE_PATH"
  expect_status 200
  echo "$LAST_BODY"
else
  step "buyer image check skipped"
  echo "Set IMAGE_PATH=/path/to/image.jpg to test /v1/buyer/check"
fi

step "update product"
request PUT "/v1/shops/$shop_id/products/$product_id" "${SELLER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"expectedVersion\":$product_version,\"name\":\"$PRODUCT_NAME Updated\",\"description\":\"$PRODUCT_DESCRIPTION\",\"category\":\"$PRODUCT_CATEGORY\",\"tags\":[\"$PRODUCT_TAG\",\"updated\"],\"freshnessNote\":\"Updated freshness note\",\"freshnessScore\":8.6,\"price\":$PRODUCT_PRICE,\"currency\":\"$PRODUCT_CURRENCY\",\"status\":\"$PRODUCT_STATUS\"}"
expect_status 200
echo "$LAST_BODY"
product_version="$(printf '%s' "$LAST_BODY" | json_get version || true)"

step "update shop"
request PUT "/v1/shops/$shop_id" "${SELLER_AUTH[@]}" "${BASE_JSON[@]}" \
  -d "{\"expectedVersion\":$shop_version,\"name\":\"$SHOP_NAME Updated\",\"description\":\"$SHOP_DESCRIPTION updated\",\"address\":\"$SHOP_ADDRESS\",\"latitude\":$SHOP_LATITUDE,\"longitude\":$SHOP_LONGITUDE}"
expect_status 200
echo "$LAST_BODY"

step "done"
echo "sellerEmail=$SELLER_EMAIL"
echo "buyerEmail=$BUYER_EMAIL"
echo "shopId=$shop_id"
echo "productId=$product_id"
echo "pledgeId=$pledge_id"
