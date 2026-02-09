#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

EMAIL="${EMAIL:-}"
PASSWORD="${PASSWORD:-Passw0rd!}"
DISPLAY_NAME="${DISPLAY_NAME:-Test User}"

SHOP_NAME="${SHOP_NAME:-Green Shop}"
SHOP_DESCRIPTION="${SHOP_DESCRIPTION:-Fresh daily}"
SHOP_ADDRESS="${SHOP_ADDRESS:-123 Main St}"
SHOP_LATITUDE="${SHOP_LATITUDE:-10.762622}"
SHOP_LONGITUDE="${SHOP_LONGITUDE:-106.660172}"

PLEDGE_SCORE="${PLEDGE_SCORE:-8.5}"
PLEDGE_CATEGORY="${PLEDGE_CATEGORY:-fresh_produce}"
PLEDGE_CONFIDENCE="${PLEDGE_CONFIDENCE:-0.91}"

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

echo "BASE_URL=$BASE_URL"

echo
echo "== health =="
curl -sS -D- "$BASE_URL/health" -o /dev/null | sed -n '1,5p'

if [[ -z "$EMAIL" ]]; then
  EMAIL="test+$RANDOM$(date +%s)@example.com"
fi

echo
echo "== register / login =="
register_body=$(request POST /v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"displayName\":\"$DISPLAY_NAME\"}")
echo "$register_body"

token=$(printf '%s' "$register_body" | json_get accessToken || true)
if [[ -z "$token" ]]; then
  login_body=$(request POST /v1/auth/login \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
  echo "$login_body"
  token=$(printf '%s' "$login_body" | json_get accessToken || true)
fi

if [[ -z "$token" ]]; then
  echo "ERROR: could not extract accessToken (install jq for reliability)" >&2
  exit 1
fi

echo
echo "== me =="
me_body=$(request GET /v1/me -H "Authorization: Bearer $token")
echo "$me_body"

echo
echo "== create shop =="
shop_body=$(request POST /v1/shops \
  -H "Authorization: Bearer $token" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$SHOP_NAME\",\"description\":\"$SHOP_DESCRIPTION\",\"address\":\"$SHOP_ADDRESS\",\"latitude\":$SHOP_LATITUDE,\"longitude\":$SHOP_LONGITUDE}")
echo "$shop_body"

shop_id=$(printf '%s' "$shop_body" | json_get shopId || true)
if [[ -z "$shop_id" ]]; then
  echo "ERROR: could not extract shopId from shop response" >&2
  exit 1
fi

echo
echo "== list shops =="
shops_body=$(request GET /v1/shops)
echo "$shops_body"

echo
echo "== get shop =="
shop_detail_body=$(request GET "/v1/shops/$shop_id")
echo "$shop_detail_body"

echo
echo "== seller commit =="
commit_body=$(request POST /v1/seller/commit \
  -H "Authorization: Bearer $token" \
  -H "Content-Type: application/json" \
  -d "{\"shopId\":\"$shop_id\",\"score\":$PLEDGE_SCORE,\"category\":\"$PLEDGE_CATEGORY\",\"confidence\":$PLEDGE_CONFIDENCE}")
echo "$commit_body"

pledge_id=$(printf '%s' "$commit_body" | json_get pledgeId || true)
if [[ -z "$pledge_id" ]]; then
  echo "WARN: could not extract pledgeId from commit response" >&2
fi

if [[ -n "$IMAGE_PATH" ]]; then
  if [[ ! -f "$IMAGE_PATH" ]]; then
    echo "ERROR: IMAGE_PATH does not exist: $IMAGE_PATH" >&2
    exit 1
  fi
  if [[ -z "$pledge_id" ]]; then
    echo "ERROR: pledgeId is required for buyer check" >&2
    exit 1
  fi

  echo
  echo "== buyer check =="
  buyer_body=$(request POST /v1/buyer/check \
    -F "pledgeId=$pledge_id" \
    -F "image=@$IMAGE_PATH")
  echo "$buyer_body"
else
  echo
  echo "== buyer check skipped =="
  echo "Set IMAGE_PATH=/abs/path/to/shop.jpg to run /v1/buyer/check"
fi
