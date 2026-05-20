#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8081}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
CURL_MAX_TIME="${CURL_MAX_TIME:-60}"
PASSWORD="${PASSWORD:-Passw0rd!}"
RUN_ID="${RUN_ID:-$(date +%s)}"

# Backend role model:
# - register creates role=user unless email is in BOOTSTRAP_ADMIN_EMAILS.
# - seller capability is ownership-based: an account owns exactly one shop.
# This script therefore creates 2 seller-owner accounts for 2 shops.
ADMIN_EMAIL="${ADMIN_EMAIL:-admin+$RUN_ID@example.com}"
SELLER1_EMAIL="${SELLER1_EMAIL:-seller1+$RUN_ID@example.com}"
SELLER2_EMAIL="${SELLER2_EMAIL:-seller2+$RUN_ID@example.com}"
BUYER1_EMAIL="${BUYER1_EMAIL:-buyer1+$RUN_ID@example.com}"
BUYER2_EMAIL="${BUYER2_EMAIL:-buyer2+$RUN_ID@example.com}"

PRODUCT_STATUS="${PRODUCT_STATUS:-active}"
PRODUCT_CURRENCY="${PRODUCT_CURRENCY:-VND}"

# Optional. If set, exercises /seller/score and /buyer/check.
# Without this, buyer_checks are inserted directly into Mongo for frontend/admin data.
IMAGE_PATH="${IMAGE_PATH:-}"

MONGODB_URI="${MONGODB_URI:-$(sed -n 's/^MONGODB_URI=//p' .env 2>/dev/null | tail -n1)}"
MONGODB_DATABASE="${MONGODB_DATABASE:-$(sed -n 's/^MONGODB_DATABASE=//p' .env 2>/dev/null | tail -n1)}"
MONGODB_URI="${MONGODB_URI:-mongodb://127.0.0.1:27017}"
MONGODB_DATABASE="${MONGODB_DATABASE:-vngrocery}"

BASE_JSON=(-H "Content-Type: application/json")
LAST_BODY=""
LAST_STATUS=""
REGISTER_TOKEN=""
REGISTER_USER_ID=""

ADMIN_TOKEN=""
ADMIN_USER_ID=""
SELLER1_TOKEN=""
SELLER1_USER_ID=""
SELLER2_TOKEN=""
SELLER2_USER_ID=""
BUYER1_TOKEN=""
BUYER1_USER_ID=""
BUYER2_TOKEN=""
BUYER2_USER_ID=""

SHOP1_ID=""
SHOP2_ID=""
SHOP1_VERSION=""
SHOP2_VERSION=""

SHOP_IDS=()
SHOP_NAMES=()
SHOP_OWNER_TOKENS=()
SHOP_OWNER_IDS=()
SHOP_OWNER_LABELS=()

PRODUCT_IDS=()
PRODUCT_SHOP_IDS=()
PRODUCT_NAMES=()
PRODUCT_CATEGORIES=()
PRODUCT_PRICES=()

PLEDGE_IDS=()
PLEDGE_SHOP_IDS=()
PLEDGE_PRODUCT_IDS=()
PLEDGE_BUNDLE_IDS=()
PLEDGE_TOKENS=()

have_cmd() { command -v "$1" >/dev/null 2>&1; }

step() {
  printf '\n== %s ==\n' "$1"
}

json_get() {
  local key="$1"
  if have_cmd jq; then
    jq -r ".${key} // empty"
  else
    sed -nE "s/.*\"${key}\"[[:space:]]*:[[:space:]]*\"?([^\",}]*)\"?.*/\1/p" | head -n 1
  fi
}

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
      request POST /v1/auth/login "${BASE_JSON[@]}" -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}"
      echo "login status=$LAST_STATUS"
      echo "$LAST_BODY"
      expect_status 200
    else
      expect_status 201
    fi
  fi

  REGISTER_TOKEN="$(printf '%s' "$LAST_BODY" | json_get accessToken || true)"
  REGISTER_USER_ID="$(printf '%s' "$LAST_BODY" | json_get userId || true)"
  [[ -n "$REGISTER_TOKEN" ]] || { echo "ERROR: missing token for $label" >&2; exit 1; }
  [[ -n "$REGISTER_USER_ID" ]] || { echo "ERROR: missing userId for $label" >&2; exit 1; }
}

mongo_eval() {
  if ! have_cmd mongosh; then
    echo "WARN: mongosh not found; skip Mongo direct data step" >&2
    return 1
  fi
  mongosh "$MONGODB_URI/$MONGODB_DATABASE" --quiet --eval "$1"
}

promote_admin_if_needed() {
  local user_id="$1"
  local email="$2"

  step "ensure admin role"
  mongo_eval "db.users.updateOne({userId:'$user_id'}, {\$set:{role:'admin', updatedAt:new Date()}}); printjson(db.users.findOne({userId:'$user_id'}, {_id:0,userId:1,email:1,role:1,status:1}));" || {
    echo "WARN: cannot promote admin via Mongo. Set BOOTSTRAP_ADMIN_EMAILS=$email before starting server if admin endpoints are needed." >&2
    return 0
  }
}

auth_header_name() {
  local token="$1"
  printf 'Authorization: Bearer %s' "$token"
}

create_shop() {
  local label="$1"
  local token="$2"
  local name="$3"
  local description="$4"
  local address="$5"
  local latitude="$6"
  local longitude="$7"

  step "$label create shop"
  request POST /v1/shops -H "$(auth_header_name "$token")" "${BASE_JSON[@]}" \
    -d "{\"name\":\"$name\",\"description\":\"$description\",\"address\":\"$address\",\"latitude\":$latitude,\"longitude\":$longitude}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201

  local shop_id
  local version
  shop_id="$(printf '%s' "$LAST_BODY" | json_get shopId || true)"
  version="$(printf '%s' "$LAST_BODY" | json_get version || true)"
  [[ -n "$shop_id" ]] || { echo "ERROR: missing shopId" >&2; exit 1; }
  if [[ "$label" == "shop1" ]]; then
    SHOP1_ID="$shop_id"
    SHOP1_VERSION="$version"
  else
    SHOP2_ID="$shop_id"
    SHOP2_VERSION="$version"
  fi
}

create_product() {
  local shop_id="$1"
  local owner_token="$2"
  local name="$3"
  local description="$4"
  local category="$5"
  local tag1="$6"
  local tag2="$7"
  local note="$8"
  local score="$9"
  local price="${10}"

  step "create product: $name"
  request POST "/v1/shops/$shop_id/products" -H "$(auth_header_name "$owner_token")" "${BASE_JSON[@]}" \
    -d "{\"name\":\"$name\",\"description\":\"$description\",\"category\":\"$category\",\"tags\":[\"$tag1\",\"$tag2\"],\"freshnessNote\":\"$note\",\"freshnessScore\":$score,\"price\":$price,\"currency\":\"$PRODUCT_CURRENCY\",\"status\":\"$PRODUCT_STATUS\"}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201

  local product_id
  product_id="$(printf '%s' "$LAST_BODY" | json_get productId || true)"
  [[ -n "$product_id" ]] || { echo "ERROR: missing productId for $name" >&2; exit 1; }
  PRODUCT_IDS+=("$product_id")
  PRODUCT_SHOP_IDS+=("$shop_id")
  PRODUCT_NAMES+=("$name")
  PRODUCT_CATEGORIES+=("$category")
  PRODUCT_PRICES+=("$price")
}

commit_pledge() {
  local index="$1"
  local owner_token="$2"
  local shop_id="${PRODUCT_SHOP_IDS[$index]}"
  local product_id="${PRODUCT_IDS[$index]}"
  local product_name="${PRODUCT_NAMES[$index]}"
  local category="${PRODUCT_CATEGORIES[$index]}"
  local score
  local confidence
  local bundle_id
  local image_hash

  score="$(awk "BEGIN { printf \"%.1f\", 8.0 + (($index % 4) * 0.3) }")"
  confidence="$(awk "BEGIN { printf \"%.2f\", 0.88 + (($index % 3) * 0.03) }")"
  bundle_id="bundle-$RUN_ID-$index"
  image_hash="seed-image-hash-$RUN_ID-$index"

  step "commit pledge: $product_name"
  request POST /v1/seller/commit -H "$(auth_header_name "$owner_token")" "${BASE_JSON[@]}" \
    -d "{\"shopId\":\"$shop_id\",\"productId\":\"$product_id\",\"bundleId\":\"$bundle_id\",\"score\":$score,\"category\":\"$category\",\"confidence\":$confidence,\"imageHash\":\"$image_hash\",\"imageCid\":\"ipfs://seed/$RUN_ID/$index\"}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201

  local pledge_id
  local token
  pledge_id="$(printf '%s' "$LAST_BODY" | json_get pledgeId || true)"
  token="$(printf '%s' "$LAST_BODY" | json_get bundleToken || true)"
  [[ -n "$pledge_id" ]] || { echo "ERROR: missing pledgeId for $product_name" >&2; exit 1; }
  PLEDGE_IDS+=("$pledge_id")
  PLEDGE_SHOP_IDS+=("$shop_id")
  PLEDGE_PRODUCT_IDS+=("$product_id")
  PLEDGE_BUNDLE_IDS+=("$bundle_id")
  PLEDGE_TOKENS+=("$token")
}

create_review() {
  local buyer_token="$1"
  local shop_id="$2"
  local rating="$3"
  local comment="$4"

  step "create review shop=$shop_id rating=$rating"
  request POST "/v1/shops/$shop_id/reviews" -H "$(auth_header_name "$buyer_token")" "${BASE_JSON[@]}" \
    -d "{\"rating\":$rating,\"comment\":\"$comment\"}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201
}

create_freshness_report() {
  local buyer_token="$1"
  local index="$2"
  local score="$3"
  local comment="$4"
  local shop_id="${PRODUCT_SHOP_IDS[$index]}"
  local product_id="${PRODUCT_IDS[$index]}"
  local category="${PRODUCT_CATEGORIES[$index]}"

  step "create freshness report: ${PRODUCT_NAMES[$index]}"
  request POST "/v1/shops/$shop_id/products/$product_id/freshness-reports" -H "$(auth_header_name "$buyer_token")" "${BASE_JSON[@]}" \
    -d "{\"score\":$score,\"category\":\"$category\",\"confidence\":0.89,\"comment\":\"$comment\",\"imageHash\":\"buyer-report-hash-$RUN_ID-$index\",\"imageCid\":\"ipfs://seed/report/$RUN_ID/$index\"}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201
}

seed_buyer_checks_directly() {
  if ! have_cmd mongosh; then
    echo "WARN: mongosh not found; skip direct buyer_checks seed" >&2
    return 0
  fi

  step "seed buyer_checks directly for trust/admin data"
  local docs=""
  local i
  for i in "${!PLEDGE_IDS[@]}"; do
    local verdict trusted status actual delta_abs location buyer_id
    case $((i % 3)) in
      0) verdict="trusted"; trusted="true"; status="completed"; actual="8.4"; delta_abs="0.4"; location="near"; buyer_id="$BUYER1_USER_ID" ;;
      1) verdict="warning"; trusted="false"; status="completed"; actual="6.7"; delta_abs="1.6"; location="near"; buyer_id="$BUYER2_USER_ID" ;;
      *) verdict="high_risk"; trusted="false"; status="flagged"; actual="5.4"; delta_abs="3.1"; location="far"; buyer_id="$BUYER1_USER_ID" ;;
    esac

    local check_id="seed-check-$RUN_ID-$i"
    local pledged
    pledged="$(awk "BEGIN { printf \"%.1f\", $actual + $delta_abs }")"
    local doc
    doc="{_id:'$check_id',checkId:'$check_id',shopId:'${PLEDGE_SHOP_IDS[$i]}',productId:'${PLEDGE_PRODUCT_IDS[$i]}',bundleId:'${PLEDGE_BUNDLE_IDS[$i]}',pledgeId:'${PLEDGE_IDS[$i]}',buyerUserId:'$buyer_id',status:'$status',version:1,policyVersion:'freshness_policy_v1',trusted:$trusted,verdict:'$verdict',pledgedScore:$pledged,actualScore:$actual,scoreDelta:-$delta_abs,scoreDeltaAbs:$delta_abs,pledgedCategory:'${PRODUCT_CATEGORIES[$i]}',actualCategory:'${PRODUCT_CATEGORIES[$i]}',actualConfidence:0.88,locationStatus:'$location',categoryMatch:true,imageHash:'buyer-check-hash-$RUN_ID-$i',imageCid:'ipfs://seed/check/$RUN_ID/$i',reasons:['seeded_for_frontend','$verdict'],createdAt:new Date(),updatedAt:new Date()}"
    if [[ -z "$docs" ]]; then
      docs="$doc"
    else
      docs="$docs,$doc"
    fi
  done

  mongo_eval "const docs=[$docs]; for (const doc of docs) { db.buyer_checks.updateOne({_id:doc._id}, {\$set:doc}, {upsert:true}); } printjson({buyerChecksSeeded: docs.length});" >/dev/null
  echo "buyer_checks seeded=${#PLEDGE_IDS[@]}"
}

run_optional_image_checks() {
  if [[ -z "$IMAGE_PATH" ]]; then
    step "image flows skipped"
    echo "Set IMAGE_PATH=/path/to/image.jpg to run /v1/seller/score and /v1/buyer/check."
    return 0
  fi
  [[ -f "$IMAGE_PATH" ]] || { echo "ERROR: IMAGE_PATH not found: $IMAGE_PATH" >&2; exit 1; }

  step "seller score sample image"
  request POST /v1/seller/score -H "$(auth_header_name "$SELLER1_TOKEN")" -F "image=@$IMAGE_PATH"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 200

  local token="${PLEDGE_TOKENS[0]}"
  if [[ -z "$token" ]]; then
    request POST "/v1/shops/${PLEDGE_SHOP_IDS[0]}/pledges/${PLEDGE_IDS[0]}/bundle-token" -H "$(auth_header_name "$SELLER1_TOKEN")"
    expect_status 200
    token="$(printf '%s' "$LAST_BODY" | json_get bundleToken || true)"
  fi
  [[ -n "$token" ]] || { echo "ERROR: missing bundle token for buyer check" >&2; exit 1; }

  step "buyer check sample image"
  request POST /v1/buyer/check -H "$(auth_header_name "$BUYER1_TOKEN")" \
    -F "pledgeId=${PLEDGE_IDS[0]}" \
    -F "bundleId=${PLEDGE_BUNDLE_IDS[0]}" \
    -F "bundleToken=$token" \
    -F "locationStatus=near" \
    -F "image=@$IMAGE_PATH"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 200
}

step "config"
echo "BASE_URL=$BASE_URL"
echo "RUN_ID=$RUN_ID"
echo "ADMIN_EMAIL=$ADMIN_EMAIL"
echo "SELLER1_EMAIL=$SELLER1_EMAIL"
echo "SELLER2_EMAIL=$SELLER2_EMAIL"
echo "BUYER1_EMAIL=$BUYER1_EMAIL"
echo "BUYER2_EMAIL=$BUYER2_EMAIL"
echo "MONGODB_URI=$MONGODB_URI"
echo "MONGODB_DATABASE=$MONGODB_DATABASE"
echo "IMAGE_PATH=${IMAGE_PATH:-<empty>}"

step "health"
request GET /health
expect_status 200
echo "$LAST_BODY"

register_or_login "admin" "$ADMIN_EMAIL" "Admin" "Seed"
ADMIN_TOKEN="$REGISTER_TOKEN"
ADMIN_USER_ID="$REGISTER_USER_ID"
promote_admin_if_needed "$ADMIN_USER_ID" "$ADMIN_EMAIL"

register_or_login "seller1" "$SELLER1_EMAIL" "Minh" "Seller"
SELLER1_TOKEN="$REGISTER_TOKEN"
SELLER1_USER_ID="$REGISTER_USER_ID"

register_or_login "seller2" "$SELLER2_EMAIL" "An" "Seller"
SELLER2_TOKEN="$REGISTER_TOKEN"
SELLER2_USER_ID="$REGISTER_USER_ID"

register_or_login "buyer1" "$BUYER1_EMAIL" "Bao" "Buyer"
BUYER1_TOKEN="$REGISTER_TOKEN"
BUYER1_USER_ID="$REGISTER_USER_ID"

register_or_login "buyer2" "$BUYER2_EMAIL" "Linh" "Buyer"
BUYER2_TOKEN="$REGISTER_TOKEN"
BUYER2_USER_ID="$REGISTER_USER_ID"

step "admin me"
request GET /v1/me -H "$(auth_header_name "$ADMIN_TOKEN")"
expect_status 200
echo "$LAST_BODY"

create_shop "shop1" "$SELLER1_TOKEN" "VNMeat Ben Thanh $RUN_ID" "Quay thit tuoi song co cam ket AI" "Cho Ben Thanh, Quan 1, TP.HCM" "10.7721" "106.6983"
create_shop "shop2" "$SELLER2_TOKEN" "VNMeat Thao Dien $RUN_ID" "Thuc pham cao cap va hai san tuoi" "Xuan Thuy, Thao Dien, TP Thu Duc" "10.8022" "106.7328"

SHOP_IDS=("$SHOP1_ID" "$SHOP2_ID")
SHOP_NAMES=("VNMeat Ben Thanh $RUN_ID" "VNMeat Thao Dien $RUN_ID")
SHOP_OWNER_TOKENS=("$SELLER1_TOKEN" "$SELLER2_TOKEN")
SHOP_OWNER_IDS=("$SELLER1_USER_ID" "$SELLER2_USER_ID")
SHOP_OWNER_LABELS=("seller1" "seller2")

create_product "$SHOP1_ID" "$SELLER1_TOKEN" "Bo My Ribeye $RUN_ID" "Ribeye USDA Choice cat steak" "Thit bo" "bo-my" "steak" "Bao quan 0-4C" "8.8" "690000"
create_product "$SHOP1_ID" "$SELLER1_TOKEN" "Heo ba chi rut suon $RUN_ID" "Ba chi VietGAP ty le nac mo can bang" "Thit heo" "heo" "ba-chi" "Dong goi trong ngay" "8.1" "185000"
create_product "$SHOP1_ID" "$SELLER1_TOKEN" "Ga ta lam sach $RUN_ID" "Ga ta 1.4-1.6kg da vang tu nhien" "Gia cam" "ga-ta" "tuoi" "Giao trong ngay" "8.4" "165000"

create_product "$SHOP2_ID" "$SELLER2_TOKEN" "Ca hoi Na Uy fillet $RUN_ID" "Fillet ca hoi Na Uy cat phan lung" "Hai san" "ca-hoi" "na-uy" "Giao lanh" "9.0" "520000"
create_product "$SHOP2_ID" "$SELLER2_TOKEN" "Tom su song size 20 $RUN_ID" "Tom su song be oxy tai cua hang" "Hai san" "tom-su" "song" "Dong goi khi ban" "8.6" "420000"
create_product "$SHOP2_ID" "$SELLER2_TOKEN" "Uc vit hun khoi $RUN_ID" "Uc vit Phap hun khoi cat lat" "Thit gia cam" "uc-vit" "hun-khoi" "Bao quan mat" "8.2" "310000"

step "list shops"
request GET "/v1/shops?page=1&pageSize=20"
expect_status 200
echo "$LAST_BODY"

step "list products shop1"
request GET "/v1/shops/$SHOP1_ID/products"
expect_status 200
echo "$LAST_BODY"

step "list products shop2"
request GET "/v1/shops/$SHOP2_ID/products"
expect_status 200
echo "$LAST_BODY"

for i in "${!PRODUCT_IDS[@]}"; do
  if [[ "${PRODUCT_SHOP_IDS[$i]}" == "$SHOP1_ID" ]]; then
    commit_pledge "$i" "$SELLER1_TOKEN"
  else
    commit_pledge "$i" "$SELLER2_TOKEN"
  fi
done

for shop_id in "${SHOP_IDS[@]}"; do
  step "pledges for shop=$shop_id"
  request GET "/v1/shops/$shop_id/pledges"
  expect_status 200
  echo "$LAST_BODY"
done

step "pledge proof sample"
request GET "/v1/shops/${PLEDGE_SHOP_IDS[0]}/pledges/${PLEDGE_IDS[0]}/proof"
expect_status 200
echo "$LAST_BODY"

create_review "$BUYER1_TOKEN" "$SHOP1_ID" 5 "Bo ribeye tuoi, dung cam ket."
create_review "$BUYER2_TOKEN" "$SHOP1_ID" 4 "Ba chi va ga dong goi sach."
create_review "$BUYER1_TOKEN" "$SHOP2_ID" 5 "Ca hoi mau dep, giao lanh tot."
create_review "$BUYER2_TOKEN" "$SHOP2_ID" 4 "Tom song khoe, cua hang phuc vu nhanh."

for shop_id in "${SHOP_IDS[@]}"; do
  step "reviews for shop=$shop_id"
  request GET "/v1/shops/$shop_id/reviews"
  expect_status 200
  echo "$LAST_BODY"
done

for i in "${!PRODUCT_IDS[@]}"; do
  if (( i % 2 == 0 )); then
    create_freshness_report "$BUYER1_TOKEN" "$i" "8.3" "Buyer report: hang tuoi va dung mo ta."
  else
    create_freshness_report "$BUYER2_TOKEN" "$i" "7.8" "Buyer report: chat luong on, can theo doi them."
  fi
done

seed_buyer_checks_directly
run_optional_image_checks

step "admin users"
request GET "/v1/admin/users?page=1&pageSize=20" -H "$(auth_header_name "$ADMIN_TOKEN")"
expect_status 200
echo "$LAST_BODY"

step "admin buyer checks"
request GET "/v1/admin/buyer-checks?page=1&pageSize=20" -H "$(auth_header_name "$ADMIN_TOKEN")"
expect_status 200
echo "$LAST_BODY"

step "final shop trust summaries"
for shop_id in "${SHOP_IDS[@]}"; do
  request GET "/v1/shops/$shop_id"
  expect_status 200
  echo "$LAST_BODY"
done

step "done"
echo "adminEmail=$ADMIN_EMAIL"
echo "seller1Email=$SELLER1_EMAIL shop1Id=$SHOP1_ID"
echo "seller2Email=$SELLER2_EMAIL shop2Id=$SHOP2_ID"
echo "buyer1Email=$BUYER1_EMAIL"
echo "buyer2Email=$BUYER2_EMAIL"
echo "productCount=${#PRODUCT_IDS[@]}"
echo "pledgeCount=${#PLEDGE_IDS[@]}"
