#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:5000}"

EMAIL="${EMAIL:-}"
PASSWORD="${PASSWORD:-Passw0rd!}"
DISPLAY_NAME="${DISPLAY_NAME:-Test User}"

SHOP_NAME="${SHOP_NAME:-Green Shop}"
SHOP_DESCRIPTION="${SHOP_DESCRIPTION:-Fresh daily}"
SHOP_ADDRESS="${SHOP_ADDRESS:-123 Main St}"
SHOP_LATITUDE="${SHOP_LATITUDE:-10.762622}"
SHOP_LONGITUDE="${SHOP_LONGITUDE:-106.660172}"

PRODUCT_NAME="${PRODUCT_NAME:-Morning Greens}"
PRODUCT_DESCRIPTION="${PRODUCT_DESCRIPTION:-Fresh vegetables for trust-flow smoke test}"
PRODUCT_CATEGORY="${PRODUCT_CATEGORY:-fresh_produce}"
PRODUCT_TAG="${PRODUCT_TAG:-leafy}"
PRODUCT_PRICE="${PRODUCT_PRICE:-25000}"
PRODUCT_CURRENCY="${PRODUCT_CURRENCY:-VND}"
PRODUCT_STATUS="${PRODUCT_STATUS:-published}"

PLEDGE_SCORE="${PLEDGE_SCORE:-8.5}"
PLEDGE_CATEGORY="${PLEDGE_CATEGORY:-fresh_produce}"
PLEDGE_CONFIDENCE="${PLEDGE_CONFIDENCE:-0.91}"
PLEDGE_IMAGE_HASH="${PLEDGE_IMAGE_HASH:-manual-test-image-hash}"

REPORT_SCORE="${REPORT_SCORE:-7.2}"
REPORT_CATEGORY="${REPORT_CATEGORY:-fresh_produce}"
REPORT_CONFIDENCE="${REPORT_CONFIDENCE:-0.88}"
REPORT_COMMENT="${REPORT_COMMENT:-Buyer smoke-test report}"

IMAGE_PATH="${IMAGE_PATH:-}"

have_cmd() { command -v "$1" >/dev/null 2>&1; }

json_get() {
  local key="$1"
  if have_cmd jq; then
    jq -r ".$key // empty"
    return 0
  fi
  sed -nE "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"?([^\",}]*)\"?.*/\\1/p" | head -n 1
}

request() {
  local method="$1"
  local path="$2"
  shift 2
  curl -sS -X "$method" "$BASE_URL$path" "$@"
}

print_step() {
  printf '\n== %s ==\n' "$1"
}

extract_token() {
  local body="$1"
  printf '%s' "$body" | json_get accessToken || true
}

BASE_HEADERS=(-H "Content-Type: application/json")

echo "BASE_URL=$BASE_URL"
echo "EMAIL=$EMAIL"
echo "PASSWORD=$PASSWORD"
echo "DISPLAY_NAME=$DISPLAY_NAME"
echo "SHOP_NAME=$SHOP_NAME"
echo "PRODUCT_NAME=$PRODUCT_NAME"
echo "IMAGE_PATH=${IMAGE_PATH:-<empty>}"

print_step "health"
curl -sS -D- "$BASE_URL/health" -o /dev/null | sed -n '1,5p'

if [[ -z "$EMAIL" ]]; then
  EMAIL="test+$RANDOM$(date +%s)@example.com"
fi

print_step "register or login"
register_body=$(request POST /v1/auth/register \
  "${BASE_HEADERS[@]}" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"displayName\":\"$DISPLAY_NAME\"}")
echo "$register_body"

token="$(extract_token "$register_body")"
if [[ -z "$token" ]]; then
  login_body=$(request POST /v1/auth/login \
    "${BASE_HEADERS[@]}" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
  echo "$login_body"
  token="$(extract_token "$login_body")"
fi

if [[ -z "$token" ]]; then
  echo "ERROR: could not extract accessToken" >&2
  exit 1
fi

AUTH_HEADER=(-H "Authorization: Bearer $token")

print_step "me"
request GET /v1/me "${AUTH_HEADER[@]}"
echo

print_step "create shop"
shop_body=$(request POST /v1/shops \
  "${AUTH_HEADER[@]}" \
  "${BASE_HEADERS[@]}" \
  -d "{\"name\":\"$SHOP_NAME\",\"description\":\"$SHOP_DESCRIPTION\",\"address\":\"$SHOP_ADDRESS\",\"latitude\":$SHOP_LATITUDE,\"longitude\":$SHOP_LONGITUDE}")
echo "$shop_body"

shop_id="$(printf '%s' "$shop_body" | json_get shopId || true)"
if [[ -z "$shop_id" ]]; then
  echo "ERROR: could not extract shopId" >&2
  exit 1
fi

print_step "create second shop (should fail with 409)"

dup_body=$(request POST /v1/shops \

  "${AUTH_HEADER[@]}" \

  "${BASE_HEADERS[@]}" \

  -d "{\"name\":\"Duplicate Shop\",\"address\":\"Nowhere\"}")

echo "$dup_body"

if ! printf '%s' "$dup_body" | grep -q "Account already owns a shop"; then

  echo "ERROR: missing expected 409 error for duplicate shop"

  exit 1

fi

echo
print_step "create product"
product_body=$(request POST "/v1/shops/$shop_id/products" \
  "${AUTH_HEADER[@]}" \
  "${BASE_HEADERS[@]}" \
  -d "{\"name\":\"$PRODUCT_NAME\",\"description\":\"$PRODUCT_DESCRIPTION\",\"category\":\"$PRODUCT_CATEGORY\",\"tags\":[\"$PRODUCT_TAG\"],\"freshnessNote\":\"Fresh this morning\",\"freshnessScore\":8.4,\"price\":$PRODUCT_PRICE,\"currency\":\"$PRODUCT_CURRENCY\",\"status\":\"$PRODUCT_STATUS\"}")
echo "$product_body"

product_id="$(printf '%s' "$product_body" | json_get productId || true)"
if [[ -z "$product_id" ]]; then
  echo "ERROR: could not extract productId" >&2
  exit 1
fi

print_step "list products"
request GET "/v1/shops/$shop_id/products?category=$PRODUCT_CATEGORY&tag=$PRODUCT_TAG&sort=freshness_desc"
echo

commit_image_hash="$PLEDGE_IMAGE_HASH"

if [[ -n "$IMAGE_PATH" ]]; then
  if [[ ! -f "$IMAGE_PATH" ]]; then
    echo "ERROR: IMAGE_PATH does not exist: $IMAGE_PATH" >&2
    exit 1
  fi

  print_step "seller score"
  seller_score_body=$(request POST /v1/seller/score \
    "${AUTH_HEADER[@]}" \
    -F "image=@$IMAGE_PATH")
  echo "$seller_score_body"

  scored_hash="$(printf '%s' "$seller_score_body" | json_get imageHash || true)"
  if [[ -n "$scored_hash" ]]; then
    commit_image_hash="$scored_hash"
  fi
else
  print_step "seller score skipped"
  echo "Set IMAGE_PATH=/abs/path/to/image.jpg to exercise /v1/seller/score and image-based flows"
fi

print_step "seller commit"
commit_body=$(request POST /v1/seller/commit \
  "${AUTH_HEADER[@]}" \
  "${BASE_HEADERS[@]}" \
  -d "{\"shopId\":\"$shop_id\",\"productId\":\"$product_id\",\"score\":$PLEDGE_SCORE,\"category\":\"$PLEDGE_CATEGORY\",\"confidence\":$PLEDGE_CONFIDENCE,\"imageHash\":\"$commit_image_hash\"}")
echo "$commit_body"

pledge_id="$(printf '%s' "$commit_body" | json_get pledgeId || true)"
if [[ -z "$pledge_id" ]]; then
  echo "ERROR: could not extract pledgeId from seller commit" >&2
  exit 1
fi

print_step "pledge history"
request GET "/v1/shops/$shop_id/pledges?productId=$product_id&category=$PLEDGE_CATEGORY"
echo

print_step "pledge integrity"
request GET "/v1/shops/$shop_id/pledges/$pledge_id/integrity"
echo

if [[ -n "$IMAGE_PATH" ]]; then
  print_step "buyer check"
  request POST /v1/buyer/check \
    "${AUTH_HEADER[@]}" \
    -F "pledgeId=$pledge_id" \
    -F "image=@$IMAGE_PATH"
  echo

  print_step "create product freshness report"
  freshness_body=$(request POST "/v1/shops/$shop_id/products/$product_id/freshness-reports" \
    "${AUTH_HEADER[@]}" \
    "${BASE_HEADERS[@]}" \
    -d "{\"score\":$REPORT_SCORE,\"category\":\"$REPORT_CATEGORY\",\"confidence\":$REPORT_CONFIDENCE,\"comment\":\"$REPORT_COMMENT\",\"imageHash\":\"$commit_image_hash\"}")
  echo "$freshness_body"

  print_step "list product freshness reports"
  request GET "/v1/shops/$shop_id/products/$product_id/freshness-reports"
  echo
else
  print_step "buyer/image flows skipped"
  echo "Set IMAGE_PATH to run /v1/buyer/check and freshness report smoke tests"
fi

print_step "done"
echo "shopId=$shop_id"
echo "productId=$product_id"
echo "pledgeId=$pledge_id"
