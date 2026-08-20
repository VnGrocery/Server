#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:5050}"
EMAIL="${EMAIL:-}"
PASSWORD="${PASSWORD:-Passw0rd!}"
DISPLAY_NAME="${DISPLAY_NAME:-E2E User}"
IMAGE_PATH="${IMAGE_PATH:-}"

SHOP_NAME="${SHOP_NAME:-E2E Trust Shop}"
PRODUCT_NAME="${PRODUCT_NAME:-E2E Fresh Produce}"
PRODUCT_CATEGORY="${PRODUCT_CATEGORY:-fresh_produce}"
PRODUCT_TAG="${PRODUCT_TAG:-trusted}"
# /v1/seller/commit rejects a pledge without a bundleId.
BUNDLE_ID="${BUNDLE_ID:-e2e-bundle-$RANDOM$(date +%s)}"

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

step() {
  printf '\n== %s ==\n' "$1"
}

BASE_HEADERS=(-H "Content-Type: application/json")

if [[ -z "$EMAIL" ]]; then
  EMAIL="e2e+$RANDOM$(date +%s)@example.com"
fi

step "health"
curl -sS "$BASE_URL/health"
echo

step "auth"
register_body=$(request POST /v1/auth/register "${BASE_HEADERS[@]}" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"displayName\":\"$DISPLAY_NAME\"}")
token="$(printf '%s' "$register_body" | json_get accessToken || true)"
if [[ -z "$token" ]]; then
  login_body=$(request POST /v1/auth/login "${BASE_HEADERS[@]}" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
  token="$(printf '%s' "$login_body" | json_get accessToken || true)"
fi
if [[ -z "$token" ]]; then
  echo "ERROR: auth failed"
  exit 1
fi
AUTH_HEADER=(-H "Authorization: Bearer $token")

step "create shop"
shop_body=$(request POST /v1/shops "${AUTH_HEADER[@]}" "${BASE_HEADERS[@]}" \
  -d "{\"name\":\"$SHOP_NAME\",\"description\":\"E2E shop\",\"address\":\"123 E2E St\",\"latitude\":10.76,\"longitude\":106.66}")
shop_id="$(printf '%s' "$shop_body" | json_get shopId || true)"
if [[ -z "$shop_id" ]]; then
  echo "ERROR: missing shopId"
  exit 1
fi

step "create product"
product_body=$(request POST "/v1/shops/$shop_id/products" "${AUTH_HEADER[@]}" "${BASE_HEADERS[@]}" \
  -d "{\"name\":\"$PRODUCT_NAME\",\"description\":\"E2E product\",\"category\":\"$PRODUCT_CATEGORY\",\"tags\":[\"$PRODUCT_TAG\"],\"freshnessNote\":\"Fresh today\",\"freshnessScore\":8.8,\"price\":25000,\"currency\":\"VND\",\"status\":\"published\"}")
product_id="$(printf '%s' "$product_body" | json_get productId || true)"
if [[ -z "$product_id" ]]; then
  echo "ERROR: missing productId"
  exit 1
fi

image_hash="e2e-manual-image-hash"
image_cid=""

if [[ -n "$IMAGE_PATH" ]]; then
  if [[ ! -f "$IMAGE_PATH" ]]; then
    echo "ERROR: IMAGE_PATH not found: $IMAGE_PATH"
    exit 1
  fi

  step "upload media"
  media_body=$(request POST /v1/media/images "${AUTH_HEADER[@]}" -F "image=@$IMAGE_PATH")
  echo "$media_body"
  image_hash="$(printf '%s' "$media_body" | json_get imageHash || true)"
  image_cid="$(printf '%s' "$media_body" | json_get imageCid || true)"

  step "seller score"
  request POST /v1/seller/score "${AUTH_HEADER[@]}" -F "image=@$IMAGE_PATH"
  echo
else
  step "image upload skipped"
  echo "Set IMAGE_PATH=/abs/path/to/image.jpg to cover upload, seller score, and buyer check image flow"
fi

step "seller commit"
commit_payload="{\"shopId\":\"$shop_id\",\"productId\":\"$product_id\",\"bundleId\":\"$BUNDLE_ID\",\"score\":8.6,\"category\":\"$PRODUCT_CATEGORY\",\"confidence\":0.92,\"imageHash\":\"$image_hash\""
if [[ -n "$image_cid" ]]; then
  commit_payload+=",\"imageCid\":\"$image_cid\""
fi
commit_payload+="}"
commit_body=$(request POST /v1/seller/commit "${AUTH_HEADER[@]}" "${BASE_HEADERS[@]}" -d "$commit_payload")
echo "$commit_body"
pledge_id="$(printf '%s' "$commit_body" | json_get pledgeId || true)"
if [[ -z "$pledge_id" ]]; then
  echo "ERROR: missing pledgeId"
  exit 1
fi

step "proof"
request GET "/v1/shops/$shop_id/pledges/$pledge_id/proof"
echo

step "pledge history"
request GET "/v1/shops/$shop_id/pledges?productId=$product_id&category=$PRODUCT_CATEGORY"
echo

if [[ -n "$IMAGE_PATH" ]]; then
  step "buyer check"
  request POST /v1/buyer/check "${AUTH_HEADER[@]}" -F "pledgeId=$pledge_id" -F "image=@$IMAGE_PATH"
  echo
fi

step "create freshness report"
report_payload="{\"score\":7.4,\"category\":\"$PRODUCT_CATEGORY\",\"confidence\":0.87,\"comment\":\"E2E freshness report\",\"imageHash\":\"$image_hash\""
if [[ -n "$image_cid" ]]; then
  report_payload+=",\"imageCid\":\"$image_cid\""
fi
report_payload+="}"
request POST "/v1/shops/$shop_id/products/$product_id/freshness-reports" "${AUTH_HEADER[@]}" "${BASE_HEADERS[@]}" -d "$report_payload"
echo

step "list freshness reports"
request GET "/v1/shops/$shop_id/products/$product_id/freshness-reports"
echo

step "shop detail"
request GET "/v1/shops/$shop_id"
echo

step "summary"
echo "email=$EMAIL"
echo "shopId=$shop_id"
echo "productId=$product_id"
echo "pledgeId=$pledge_id"
