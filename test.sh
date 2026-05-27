#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8081}"
CURL_CONNECT_TIMEOUT="${CURL_CONNECT_TIMEOUT:-5}"
CURL_MAX_TIME="${CURL_MAX_TIME:-60}"
PASSWORD="${PASSWORD:-Passw0rd!}"
RUN_ID="${RUN_ID:-$(date +%s)}"

# Backend role model:
# - auth role is admin/user only.
# - buyer mode is available to normal active users.
# - seller mode is ownership-based: one account owns one shop.
# This seed creates 1 admin, 1 buyer, and 3 seller-owner accounts.
ADMIN_EMAIL="${ADMIN_EMAIL:-admin+$RUN_ID@example.com}"
BUYER_EMAIL="${BUYER_EMAIL:-buyer+$RUN_ID@example.com}"
SELLER1_EMAIL="${SELLER1_EMAIL:-seller1+$RUN_ID@example.com}"
SELLER2_EMAIL="${SELLER2_EMAIL:-seller2+$RUN_ID@example.com}"
SELLER3_EMAIL="${SELLER3_EMAIL:-seller3+$RUN_ID@example.com}"

PRODUCT_STATUS="${PRODUCT_STATUS:-active}"
PRODUCT_CURRENCY="${PRODUCT_CURRENCY:-VND}"

# Optional. If set, exercises /seller/score and /buyer/check.
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
BUYER_TOKEN=""
BUYER_USER_ID=""

SELLER_TOKENS=()
SELLER_USER_IDS=()
SELLER_EMAILS=("$SELLER1_EMAIL" "$SELLER2_EMAIL" "$SELLER3_EMAIL")
SELLER_FIRST_NAMES=("Minh" "An" "Khoa")
SELLER_LAST_NAMES=("Seller" "Seller" "Seller")

SHOP_IDS=()
SHOP_NAMES=()
SHOP_OWNER_TOKENS=()
SHOP_OWNER_IDS=()

PRODUCT_IDS=()
PRODUCT_SHOP_IDS=()
PRODUCT_NAMES=()
PRODUCT_CATEGORIES=()
PRODUCT_PRICES=()
PRODUCT_OWNER_TOKENS=()
PRODUCT_OWNER_IDS=()
PRODUCT_FIRST_BATCH_IDS=()

BATCH_IDS=()
BATCH_PRODUCT_INDEXES=()
BATCH_SHOP_IDS=()
BATCH_CODES=()

PLEDGE_IDS=()
PLEDGE_SHOP_IDS=()
PLEDGE_PRODUCT_IDS=()
PLEDGE_BATCH_IDS=()
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

auth_header_name() {
  local token="$1"
  printf 'Authorization: Bearer %s' "$token"
}

mongo_eval() {
  if ! have_cmd mongosh; then
    echo "WARN: mongosh not found; skip Mongo direct data step" >&2
    return 1
  fi
  mongosh "$MONGODB_URI/$MONGODB_DATABASE" --quiet --eval "$1"
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

promote_admin_if_needed() {
  local user_id="$1"
  local email="$2"

  step "ensure admin role"
  mongo_eval "db.users.updateOne({userId:'$user_id'}, {\$set:{role:'admin', updatedAt:new Date()}}); printjson(db.users.findOne({userId:'$user_id'}, {_id:0,userId:1,email:1,role:1,status:1}));" || {
    echo "WARN: cannot promote admin via Mongo. Set BOOTSTRAP_ADMIN_EMAILS=$email before starting server if admin endpoints are needed." >&2
    return 0
  }
}

create_shop() {
  local index="$1"
  local token="$2"
  local owner_id="$3"
  local name="$4"
  local description="$5"
  local address="$6"
  local latitude="$7"
  local longitude="$8"

  step "create shop $index: $name"
  request POST /v1/shops -H "$(auth_header_name "$token")" "${BASE_JSON[@]}" \
    -d "{\"name\":\"$name\",\"description\":\"$description\",\"address\":\"$address\",\"latitude\":$latitude,\"longitude\":$longitude}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201

  local shop_id
  shop_id="$(printf '%s' "$LAST_BODY" | json_get shopId || true)"
  [[ -n "$shop_id" ]] || { echo "ERROR: missing shopId for $name" >&2; exit 1; }
  SHOP_IDS+=("$shop_id")
  SHOP_NAMES+=("$name")
  SHOP_OWNER_TOKENS+=("$token")
  SHOP_OWNER_IDS+=("$owner_id")
}

create_product() {
  local shop_index="$1"
  local product_number="$2"
  local name="$3"
  local description="$4"
  local category="$5"
  local tag1="$6"
  local tag2="$7"
  local note="$8"
  local score="$9"
  local price="${10}"

  local shop_id="${SHOP_IDS[$shop_index]}"
  local owner_token="${SHOP_OWNER_TOKENS[$shop_index]}"
  local image1="https://images.vnmeat.test/$RUN_ID/shop-$((shop_index + 1))/product-$product_number-main.jpg"
  local image2="https://images.vnmeat.test/$RUN_ID/shop-$((shop_index + 1))/product-$product_number-pack.jpg"

  step "create product shop=$((shop_index + 1)) product=$product_number: $name"
  request POST "/v1/shops/$shop_id/products" -H "$(auth_header_name "$owner_token")" "${BASE_JSON[@]}" \
    -d "{\"name\":\"$name\",\"description\":\"$description\",\"category\":\"$category\",\"tags\":[\"$tag1\",\"$tag2\",\"seed-$RUN_ID\"],\"imageUrls\":[\"$image1\",\"$image2\"],\"freshnessNote\":\"$note\",\"freshnessScore\":$score,\"price\":$price,\"currency\":\"$PRODUCT_CURRENCY\",\"status\":\"$PRODUCT_STATUS\"}"
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
  PRODUCT_OWNER_TOKENS+=("$owner_token")
  PRODUCT_OWNER_IDS+=("${SHOP_OWNER_IDS[$shop_index]}")
  PRODUCT_FIRST_BATCH_IDS+=("")
}

create_batch() {
  local product_index="$1"
  local batch_number="$2"
  local freshness="$3"
  local quantity="$4"
  local product_id="${PRODUCT_IDS[$product_index]}"
  local shop_id="${PRODUCT_SHOP_IDS[$product_index]}"
  local owner_token="${PRODUCT_OWNER_TOKENS[$product_index]}"
  local category="${PRODUCT_CATEGORIES[$product_index]}"
  local code="B-$RUN_ID-$((product_index + 1))-$batch_number"
  local day_offset=$((product_index + batch_number))
  local slaughtered_day
  local packed_day
  local received_day
  local expires_day
  slaughtered_day="$(printf '%02d' $((1 + day_offset % 20)))"
  packed_day="$(printf '%02d' $((2 + day_offset % 20)))"
  received_day="$(printf '%02d' $((3 + day_offset % 20)))"
  expires_day="$(printf '%02d' $((10 + day_offset % 15)))"

  step "create batch product=$((product_index + 1)) batch=$batch_number"
  request POST "/v1/shops/$shop_id/products/$product_id/batches" -H "$(auth_header_name "$owner_token")" "${BASE_JSON[@]}" \
    -d "{\"batchCode\":\"$code\",\"originName\":\"Trang trai doi tac $batch_number\",\"originAddress\":\"Tinh Long An, Viet Nam\",\"supplierName\":\"VNMeat Supply Hub $batch_number\",\"slaughteredAt\":\"2025-05-${slaughtered_day}T02:00:00Z\",\"packedAt\":\"2025-05-${packed_day}T06:00:00Z\",\"receivedAt\":\"2025-05-${received_day}T10:00:00Z\",\"expiresAt\":\"2026-06-${expires_day}T16:00:00Z\",\"quantity\":$quantity,\"quantityUnit\":\"kg\",\"storageTempMin\":0,\"storageTempMax\":4,\"currentFreshness\":$freshness,\"currentCategory\":\"$category\",\"status\":\"active\"}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201

  local batch_id
  batch_id="$(printf '%s' "$LAST_BODY" | json_get batchId || true)"
  [[ -n "$batch_id" ]] || { echo "ERROR: missing batchId for product ${PRODUCT_NAMES[$product_index]}" >&2; exit 1; }

  BATCH_IDS+=("$batch_id")
  BATCH_PRODUCT_INDEXES+=("$product_index")
  BATCH_SHOP_IDS+=("$shop_id")
  BATCH_CODES+=("$code")
  if [[ -z "${PRODUCT_FIRST_BATCH_IDS[$product_index]}" ]]; then
    PRODUCT_FIRST_BATCH_IDS[$product_index]="$batch_id"
  fi
}

create_trace_event() {
  local batch_index="$1"
  local event_type="$2"
  local title="$3"
  local description="$4"
  local day="$5"
  local temp="$6"
  local humidity="$7"
  local product_index="${BATCH_PRODUCT_INDEXES[$batch_index]}"
  local shop_id="${BATCH_SHOP_IDS[$batch_index]}"
  local product_id="${PRODUCT_IDS[$product_index]}"
  local batch_id="${BATCH_IDS[$batch_index]}"
  local owner_token="${PRODUCT_OWNER_TOKENS[$product_index]}"
  local location="VNMeat trace hub $((product_index + 1))"
  local lat
  local lng
  lat="$(awk "BEGIN { printf \"%.4f\", 10.70 + (($product_index % 6) * 0.013) }")"
  lng="$(awk "BEGIN { printf \"%.4f\", 106.65 + (($product_index % 6) * 0.014) }")"

  request POST "/v1/shops/$shop_id/products/$product_id/batches/$batch_id/trace-events" \
    -H "$(auth_header_name "$owner_token")" "${BASE_JSON[@]}" \
    -d "{\"type\":\"$event_type\",\"title\":\"$title\",\"description\":\"$description\",\"locationName\":\"$location\",\"latitude\":$lat,\"longitude\":$lng,\"temperature\":$temp,\"humidity\":$humidity,\"imageCid\":\"ipfs://seed/trace/$RUN_ID/$batch_id/$event_type\",\"imageHash\":\"trace-hash-$RUN_ID-$batch_id-$event_type\",\"dataHash\":\"trace-data-$RUN_ID-$batch_id-$event_type\",\"occurredAt\":\"2025-05-${day}T08:00:00Z\"}"
  echo "trace $event_type status=$LAST_STATUS batch=$batch_id"
  expect_status 201
}

seed_trace_events_for_batch() {
  local batch_index="$1"
  local base_day
  base_day="$(printf '%02d' $((1 + batch_index % 20)))"
  local next_day
  local third_day
  next_day="$(printf '%02d' $((2 + batch_index % 20)))"
  third_day="$(printf '%02d' $((3 + batch_index % 20)))"
  create_trace_event "$batch_index" "origin" "Nguon goc lo hang" "Ghi nhan trang trai, nha cung cap va dieu kien ban dau." "$base_day" "3.2" "68"
  create_trace_event "$batch_index" "packaging" "Dong goi va niem phong" "Lo hang duoc dong goi lanh va niem phong QR." "$next_day" "2.8" "65"
  create_trace_event "$batch_index" "storage_check" "Kiem tra bao quan" "Nhiet do kho lanh on dinh truoc khi ban." "$third_day" "2.4" "63"
}

seed_trace_events_directly() {
  if ! have_cmd mongosh; then
    echo "WARN: mongosh not found; skip direct trace_events seed" >&2
    return 0
  fi

  step "seed trace_events directly for all batches"
  local docs=""
  local batch_index
  for batch_index in "${!BATCH_IDS[@]}"; do
    local product_index="${BATCH_PRODUCT_INDEXES[$batch_index]}"
    local shop_id="${BATCH_SHOP_IDS[$batch_index]}"
    local product_id="${PRODUCT_IDS[$product_index]}"
    local batch_id="${BATCH_IDS[$batch_index]}"
    local actor_id="${PRODUCT_OWNER_IDS[$product_index]}"
    local base_day next_day third_day lat lng
    base_day="$(printf '%02d' $((1 + batch_index % 20)))"
    next_day="$(printf '%02d' $((2 + batch_index % 20)))"
    third_day="$(printf '%02d' $((3 + batch_index % 20)))"
    lat="$(awk "BEGIN { printf \"%.4f\", 10.70 + (($product_index % 6) * 0.013) }")"
    lng="$(awk "BEGIN { printf \"%.4f\", 106.65 + (($product_index % 6) * 0.014) }")"

    local origin packaging storage
    origin="{_id:'trace-$RUN_ID-$batch_id-origin',eventId:'trace-$RUN_ID-$batch_id-origin',batchId:'$batch_id',productId:'$product_id',shopId:'$shop_id',actorUserId:'$actor_id',type:'origin',title:'Nguon goc lo hang',description:'Ghi nhan trang trai, nha cung cap va dieu kien ban dau.',locationName:'VNMeat trace hub $((product_index + 1))',latitude:$lat,longitude:$lng,temperature:3.2,humidity:68,imageCid:'ipfs://seed/trace/$RUN_ID/$batch_id/origin',imageHash:'trace-hash-$RUN_ID-$batch_id-origin',dataHash:'trace-data-$RUN_ID-$batch_id-origin',status:'active',occurredAt:new Date('2025-05-${base_day}T08:00:00Z'),createdAt:new Date()}"
    packaging="{_id:'trace-$RUN_ID-$batch_id-packaging',eventId:'trace-$RUN_ID-$batch_id-packaging',batchId:'$batch_id',productId:'$product_id',shopId:'$shop_id',actorUserId:'$actor_id',type:'packaging',title:'Dong goi va niem phong',description:'Lo hang duoc dong goi lanh va niem phong QR.',locationName:'VNMeat trace hub $((product_index + 1))',latitude:$lat,longitude:$lng,temperature:2.8,humidity:65,imageCid:'ipfs://seed/trace/$RUN_ID/$batch_id/packaging',imageHash:'trace-hash-$RUN_ID-$batch_id-packaging',dataHash:'trace-data-$RUN_ID-$batch_id-packaging',status:'active',occurredAt:new Date('2025-05-${next_day}T08:00:00Z'),createdAt:new Date()}"
    storage="{_id:'trace-$RUN_ID-$batch_id-storage',eventId:'trace-$RUN_ID-$batch_id-storage',batchId:'$batch_id',productId:'$product_id',shopId:'$shop_id',actorUserId:'$actor_id',type:'storage_check',title:'Kiem tra bao quan',description:'Nhiet do kho lanh on dinh truoc khi ban.',locationName:'VNMeat trace hub $((product_index + 1))',latitude:$lat,longitude:$lng,temperature:2.4,humidity:63,imageCid:'ipfs://seed/trace/$RUN_ID/$batch_id/storage_check',imageHash:'trace-hash-$RUN_ID-$batch_id-storage_check',dataHash:'trace-data-$RUN_ID-$batch_id-storage_check',status:'active',occurredAt:new Date('2025-05-${third_day}T08:00:00Z'),createdAt:new Date()}"
    if [[ -z "$docs" ]]; then
      docs="$origin,$packaging,$storage"
    else
      docs="$docs,$origin,$packaging,$storage"
    fi
  done

  mongo_eval "const docs=[$docs]; for (const doc of docs) { db.trace_events.updateOne({_id:doc._id}, {\$set:doc}, {upsert:true}); } printjson({traceEventsSeeded: docs.length});" >/dev/null
  echo "trace_events seeded=$((${#BATCH_IDS[@]} * 3))"
}

commit_pledge() {
  local product_index="$1"
  local owner_token="${PRODUCT_OWNER_TOKENS[$product_index]}"
  local shop_id="${PRODUCT_SHOP_IDS[$product_index]}"
  local product_id="${PRODUCT_IDS[$product_index]}"
  local batch_id="${PRODUCT_FIRST_BATCH_IDS[$product_index]}"
  local product_name="${PRODUCT_NAMES[$product_index]}"
  local category="${PRODUCT_CATEGORIES[$product_index]}"
  local score
  local confidence
  local bundle_id
  local image_hash

  score="$(awk "BEGIN { printf \"%.1f\", 8.0 + (($product_index % 5) * 0.3) }")"
  confidence="$(awk "BEGIN { printf \"%.2f\", 0.86 + (($product_index % 4) * 0.03) }")"
  bundle_id="bundle-$RUN_ID-$((product_index + 1))"
  image_hash="seed-image-hash-$RUN_ID-$((product_index + 1))"

  step "commit pledge: $product_name"
  request POST /v1/seller/commit -H "$(auth_header_name "$owner_token")" "${BASE_JSON[@]}" \
    -d "{\"shopId\":\"$shop_id\",\"productId\":\"$product_id\",\"batchId\":\"$batch_id\",\"bundleId\":\"$bundle_id\",\"score\":$score,\"category\":\"$category\",\"confidence\":$confidence,\"imageHash\":\"$image_hash\",\"imageCid\":\"ipfs://seed/pledge/$RUN_ID/$((product_index + 1))\"}"
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
  PLEDGE_BATCH_IDS+=("$batch_id")
  PLEDGE_BUNDLE_IDS+=("$bundle_id")
  PLEDGE_TOKENS+=("$token")
}

create_review() {
  local shop_index="$1"
  local rating="$2"
  local comment="$3"
  local shop_id="${SHOP_IDS[$shop_index]}"

  step "create buyer review shop=$((shop_index + 1)) rating=$rating"
  request POST "/v1/shops/$shop_id/reviews" -H "$(auth_header_name "$BUYER_TOKEN")" "${BASE_JSON[@]}" \
    -d "{\"rating\":$rating,\"comment\":\"$comment\"}"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 201
}

create_freshness_report() {
  local product_index="$1"
  local score="$2"
  local comment="$3"
  local shop_id="${PRODUCT_SHOP_IDS[$product_index]}"
  local product_id="${PRODUCT_IDS[$product_index]}"
  local batch_id="${PRODUCT_FIRST_BATCH_IDS[$product_index]}"
  local category="${PRODUCT_CATEGORIES[$product_index]}"

  step "create freshness report: ${PRODUCT_NAMES[$product_index]}"
  request POST "/v1/shops/$shop_id/products/$product_id/freshness-reports" -H "$(auth_header_name "$BUYER_TOKEN")" "${BASE_JSON[@]}" \
    -d "{\"batchId\":\"$batch_id\",\"score\":$score,\"category\":\"$category\",\"confidence\":0.89,\"comment\":\"$comment\",\"imageHash\":\"buyer-report-hash-$RUN_ID-$((product_index + 1))\",\"imageCid\":\"ipfs://seed/report/$RUN_ID/$((product_index + 1))\"}"
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
    local verdict trusted status actual delta_abs location
    case $((i % 4)) in
      0) verdict="trusted"; trusted="true"; status="completed"; actual="8.7"; delta_abs="0.3"; location="near" ;;
      1) verdict="trusted"; trusted="true"; status="completed"; actual="8.1"; delta_abs="0.8"; location="near" ;;
      2) verdict="warning"; trusted="false"; status="completed"; actual="6.8"; delta_abs="1.7"; location="near" ;;
      *) verdict="high_risk"; trusted="false"; status="flagged"; actual="5.2"; delta_abs="3.0"; location="far" ;;
    esac

    local check_id="seed-check-$RUN_ID-$i"
    local pledged
    pledged="$(awk "BEGIN { printf \"%.1f\", $actual + $delta_abs }")"
    local doc
    doc="{_id:'$check_id',checkId:'$check_id',shopId:'${PLEDGE_SHOP_IDS[$i]}',productId:'${PLEDGE_PRODUCT_IDS[$i]}',batchId:'${PLEDGE_BATCH_IDS[$i]}',bundleId:'${PLEDGE_BUNDLE_IDS[$i]}',pledgeId:'${PLEDGE_IDS[$i]}',buyerUserId:'$BUYER_USER_ID',status:'$status',version:1,policyVersion:'freshness_policy_v1',trusted:$trusted,verdict:'$verdict',pledgedScore:$pledged,actualScore:$actual,scoreDelta:-$delta_abs,scoreDeltaAbs:$delta_abs,pledgedCategory:'${PRODUCT_CATEGORIES[$i]}',actualCategory:'${PRODUCT_CATEGORIES[$i]}',actualConfidence:0.88,locationStatus:'$location',categoryMatch:true,imageHash:'buyer-check-hash-$RUN_ID-$i',imageCid:'ipfs://seed/check/$RUN_ID/$i',reasons:['seeded_for_frontend','$verdict'],createdAt:new Date(),updatedAt:new Date()}"
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
  request POST /v1/seller/score -H "$(auth_header_name "${SELLER_TOKENS[0]}")" -F "image=@$IMAGE_PATH"
  echo "status=$LAST_STATUS"
  echo "$LAST_BODY"
  expect_status 200

  local token="${PLEDGE_TOKENS[0]}"
  if [[ -z "$token" ]]; then
    request POST "/v1/shops/${PLEDGE_SHOP_IDS[0]}/pledges/${PLEDGE_IDS[0]}/bundle-token" -H "$(auth_header_name "${SELLER_TOKENS[0]}")"
    expect_status 200
    token="$(printf '%s' "$LAST_BODY" | json_get bundleToken || true)"
  fi
  [[ -n "$token" ]] || { echo "ERROR: missing bundle token for buyer check" >&2; exit 1; }

  step "buyer check sample image"
  request POST /v1/buyer/check -H "$(auth_header_name "$BUYER_TOKEN")" \
    -F "pledgeId=${PLEDGE_IDS[0]}" \
    -F "batchId=${PLEDGE_BATCH_IDS[0]}" \
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
echo "BUYER_EMAIL=$BUYER_EMAIL"
echo "SELLER_EMAILS=${SELLER_EMAILS[*]}"
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

register_or_login "buyer" "$BUYER_EMAIL" "Bao" "Buyer"
BUYER_TOKEN="$REGISTER_TOKEN"
BUYER_USER_ID="$REGISTER_USER_ID"

for i in 0 1 2; do
  register_or_login "seller$((i + 1))" "${SELLER_EMAILS[$i]}" "${SELLER_FIRST_NAMES[$i]}" "${SELLER_LAST_NAMES[$i]}"
  SELLER_TOKENS+=("$REGISTER_TOKEN")
  SELLER_USER_IDS+=("$REGISTER_USER_ID")
done

step "admin me"
request GET /v1/me -H "$(auth_header_name "$ADMIN_TOKEN")"
expect_status 200
echo "$LAST_BODY"

create_shop 1 "${SELLER_TOKENS[0]}" "${SELLER_USER_IDS[0]}" "VNMeat Ben Thanh $RUN_ID" "Quay thit tuoi, cam ket AI va truy xuat tung lo." "Cho Ben Thanh, Quan 1, TP.HCM" "10.7721" "106.6983"
create_shop 2 "${SELLER_TOKENS[1]}" "${SELLER_USER_IDS[1]}" "VNMeat Thao Dien $RUN_ID" "Thuc pham cao cap, hai san va bo nhap khau." "Xuan Thuy, Thao Dien, TP Thu Duc" "10.8022" "106.7328"
create_shop 3 "${SELLER_TOKENS[2]}" "${SELLER_USER_IDS[2]}" "VNMeat Phu Nhuan $RUN_ID" "Cua hang gia cam, thit heo va thuc pham dong goi sach." "Phan Xich Long, Phu Nhuan, TP.HCM" "10.7998" "106.6823"

create_product 0 1 "Bo My Ribeye $RUN_ID" "Ribeye USDA Choice cat steak, van mo dep." "Thit bo" "bo-my" "steak" "Bao quan 0-4C, cat trong ngay" "8.8" "690000"
create_product 0 2 "Heo ba chi rut suon $RUN_ID" "Ba chi VietGAP ty le nac mo can bang." "Thit heo" "heo" "ba-chi" "Dong goi trong ngay" "8.1" "185000"
create_product 0 3 "Ga ta lam sach $RUN_ID" "Ga ta 1.4-1.6kg da vang tu nhien." "Gia cam" "ga-ta" "tuoi" "Giao trong ngay" "8.4" "165000"
create_product 0 4 "Bo xay premium $RUN_ID" "Bo xay tu phan nac vai, phu hop burger va sot." "Thit bo" "bo-xay" "burger" "Xay moi moi sang" "8.0" "235000"

create_product 1 1 "Ca hoi Na Uy fillet $RUN_ID" "Fillet ca hoi Na Uy cat phan lung." "Hai san" "ca-hoi" "na-uy" "Giao lanh" "9.0" "520000"
create_product 1 2 "Tom su song size 20 $RUN_ID" "Tom su song be oxy tai cua hang." "Hai san" "tom-su" "song" "Dong goi khi ban" "8.6" "420000"
create_product 1 3 "Uc vit hun khoi $RUN_ID" "Uc vit Phap hun khoi cat lat." "Thit gia cam" "uc-vit" "hun-khoi" "Bao quan mat" "8.2" "310000"
create_product 1 4 "So diep Nhat $RUN_ID" "So diep Nhat dong lanh nhanh, vi ngot." "Hai san" "so-diep" "nhat" "Ra dong cham truoc khi nau" "8.5" "450000"

create_product 2 1 "Suon non heo $RUN_ID" "Suon non cat khuc, phu hop kho va nuong." "Thit heo" "suon-non" "nuong" "Dong goi hut chan khong" "8.3" "210000"
create_product 2 2 "Dui ga goc tu $RUN_ID" "Dui ga goc tu trang trai co chung nhan." "Gia cam" "dui-ga" "trang-trai" "Bao quan mat lien tuc" "8.0" "125000"
create_product 2 3 "Thit bo luc lac $RUN_ID" "Bo cat vien san cho mon luc lac." "Thit bo" "luc-lac" "bo-vien" "Nen dung trong 24 gio" "8.7" "280000"
create_product 2 4 "Xuc xich tuoi $RUN_ID" "Xuc xich tuoi it phu gia, dung cho BBQ." "Che bien" "xuc-xich" "bbq" "Bao quan 0-4C" "7.9" "155000"

for product_index in "${!PRODUCT_IDS[@]}"; do
  create_batch "$product_index" 1 "$(awk "BEGIN { printf \"%.1f\", 8.4 + (($product_index % 3) * 0.2) }")" "45"
  create_batch "$product_index" 2 "$(awk "BEGIN { printf \"%.1f\", 7.8 + (($product_index % 4) * 0.2) }")" "32"
  create_batch "$product_index" 3 "$(awk "BEGIN { printf \"%.1f\", 7.2 + (($product_index % 5) * 0.2) }")" "24"
done

seed_trace_events_directly

for product_index in "${!PRODUCT_IDS[@]}"; do
  commit_pledge "$product_index"
done

create_review 0 5 "San pham tuoi, thong tin lo ro rang."
create_review 1 5 "Hai san va thit nhap khau dong goi tot."
create_review 2 4 "Gia cam sach, truy xuat duoc tung lo."

# One buyer account is used. Freshness reports are rate-limited to 10/hour per reporter.
for product_index in 0 1 2 3 4 5 6 7 8; do
  create_freshness_report "$product_index" "$(awk "BEGIN { printf \"%.1f\", 7.8 + (($product_index % 4) * 0.3) }")" "Buyer report: hang dung mo ta, lo co du lieu truy xuat."
done

seed_buyer_checks_directly
run_optional_image_checks

step "verify public shop/product/batch data"
request GET "/v1/shops?page=1&pageSize=20"
expect_status 200
echo "$LAST_BODY"

for shop_index in "${!SHOP_IDS[@]}"; do
  shop_id="${SHOP_IDS[$shop_index]}"
  step "list products shop=$((shop_index + 1))"
  request GET "/v1/shops/$shop_id/products"
  expect_status 200
  echo "$LAST_BODY"
done

step "verify sample product batches and trace"
request GET "/v1/shops/${PRODUCT_SHOP_IDS[0]}/products/${PRODUCT_IDS[0]}/batches"
expect_status 200
echo "$LAST_BODY"
request GET "/v1/shops/${PRODUCT_SHOP_IDS[0]}/products/${PRODUCT_IDS[0]}/batches/${PRODUCT_FIRST_BATCH_IDS[0]}/trace-events"
expect_status 200
echo "$LAST_BODY"

step "verify pledges"
for shop_id in "${SHOP_IDS[@]}"; do
  request GET "/v1/shops/$shop_id/pledges"
  expect_status 200
  echo "$LAST_BODY"
done

step "pledge proof sample"
request GET "/v1/shops/${PLEDGE_SHOP_IDS[0]}/pledges/${PLEDGE_IDS[0]}/proof"
expect_status 200
echo "$LAST_BODY"

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
echo "buyerEmail=$BUYER_EMAIL"
for i in "${!SHOP_IDS[@]}"; do
  echo "seller$((i + 1))Email=${SELLER_EMAILS[$i]} shop$((i + 1))Id=${SHOP_IDS[$i]}"
done
echo "shopCount=${#SHOP_IDS[@]}"
echo "productCount=${#PRODUCT_IDS[@]}"
echo "batchCount=${#BATCH_IDS[@]}"
echo "pledgeCount=${#PLEDGE_IDS[@]}"
echo "buyerCheckCount=${#PLEDGE_IDS[@]}"
