#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

RPC_URL="${BESU_RPC_URL:-http://127.0.0.1:8545}"
CHAIN_ID="${BESU_CHAIN_ID:-1337}"
OWNER_ADDRESS="${BESU_OWNER_ADDRESS:-0xfe3b557e8fb62b89f4916b721be55ceb828dbd73}"
PRIVATE_KEY="${BESU_PRIVATE_KEY:-8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63}"
FOUNDRY_IMAGE="${FOUNDRY_IMAGE:-ghcr.io/foundry-rs/foundry:stable}"
ARTIFACT_PATH="out/IntegrityRegistry.sol/IntegrityRegistry.json"
GAS_LIMIT="${BESU_DEPLOY_GAS_LIMIT:-6000000}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required" >&2
    exit 1
  fi
}

require_cmd docker
require_cmd jq
require_cmd curl

echo "Deploying IntegrityRegistry to $RPC_URL using chainId $CHAIN_ID"

docker run --rm \
  --entrypoint forge \
  -v "$ROOT_DIR:/src" \
  -w /src \
  "$FOUNDRY_IMAGE" \
  build >/dev/null

if [[ ! -f "$ARTIFACT_PATH" ]]; then
  echo "artifact not found at $ARTIFACT_PATH" >&2
  exit 1
fi

bytecode="$(jq -r '.bytecode.object' "$ARTIFACT_PATH")"
if [[ -z "$bytecode" || "$bytecode" == "null" ]]; then
  echo "failed to read bytecode from $ARTIFACT_PATH" >&2
  exit 1
fi

owner_arg="${OWNER_ADDRESS#0x}"
owner_arg="$(printf '%064s' "$owner_arg" | tr ' ' '0')"

send_output="$(
  docker run --rm \
    --network host \
    --entrypoint cast \
    -v "$ROOT_DIR:/src" \
    -w /src \
    "$FOUNDRY_IMAGE" \
    send \
      --json \
      --legacy \
      --rpc-url "$RPC_URL" \
      --chain "$CHAIN_ID" \
      --private-key "$PRIVATE_KEY" \
      --gas-price 0 \
      --gas-limit "$GAS_LIMIT" \
      --create "${bytecode}${owner_arg}"
)"

printf '%s\n' "$send_output"

tx_hash="$(printf '%s' "$send_output" | jq -r '.transactionHash // .hash // empty')"
if [[ -z "$tx_hash" ]]; then
  echo "failed to parse transaction hash from cast send output" >&2
  exit 1
fi

receipt_output="$(
  docker run --rm \
    --network host \
    --entrypoint cast \
    "$FOUNDRY_IMAGE" \
    receipt \
      --json \
      --rpc-url "$RPC_URL" \
      "$tx_hash"
)"

printf '%s\n' "$receipt_output"

contract_address="$(printf '%s' "$receipt_output" | jq -r '.contractAddress // empty')"
if [[ -z "$contract_address" ]]; then
  echo "failed to parse contract address from receipt" >&2
  exit 1
fi

cat <<EOF

Deployed IntegrityRegistry
  contractAddress: $contract_address
  deployer: $OWNER_ADDRESS
  transactionHash: $tx_hash

Update your .env:
  BESU_ENABLED=true
  BESU_RPC_URL=$RPC_URL
  BESU_CHAIN_ID=$CHAIN_ID
  BESU_CONTRACT_ADDRESS=$contract_address
  BESU_FROM_ADDRESS=$OWNER_ADDRESS
EOF
