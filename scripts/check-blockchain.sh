#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="${ENV_FILE:-$SERVER_DIR/.env}"

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE"
fi

RPC_URL="${BESU_RPC_URL:-http://127.0.0.1:8545}"
EXPECTED_CHAIN_ID="${BESU_CHAIN_ID:-1337}"
CONTRACT_ADDRESS="${BESU_CONTRACT_ADDRESS:-}"
SECOND_SAMPLE_DELAY_SEC="${SECOND_SAMPLE_DELAY_SEC:-2}"

usage() {
  cat <<EOF
Usage:
  ./scripts/check-blockchain.sh

Environment:
  ENV_FILE                 Env file to source first (default: server/.env)
  BESU_RPC_URL             RPC endpoint to check (default: http://127.0.0.1:8545)
  BESU_CHAIN_ID            Expected chain id (default: 1337)
  BESU_CONTRACT_ADDRESS    Optional contract address to verify with eth_getCode
  SECOND_SAMPLE_DELAY_SEC  Delay between 2 block number samples (default: 2)
EOF
}

json_rpc() {
  local method="$1"
  local params="${2:-[]}"

  curl -sS --fail \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":$params,\"id\":1}" \
    "$RPC_URL"
}

extract_result() {
  sed -n 's/.*"result":"\([^"]*\)".*/\1/p'
}

hex_to_dec() {
  local hex="${1#0x}"

  if [[ -z "$hex" ]]; then
    echo "0"
    return
  fi

  printf "%d\n" "$((16#$hex))"
}

print_ok() {
  echo "[OK] $1"
}

print_warn() {
  echo "[WARN] $1"
}

print_fail() {
  echo "[FAIL] $1" >&2
}

main() {
  local chain_id_hex chain_id_dec block_1_hex block_1_dec block_2_hex block_2_dec
  local peer_count_hex peer_count_dec syncing_raw code_hex

  if [[ "${1:-}" =~ ^(-h|--help|help)$ ]]; then
    usage
    exit 0
  fi

  echo "Checking Besu RPC: $RPC_URL"

  if ! chain_id_hex="$(json_rpc "eth_chainId" | extract_result)"; then
    print_fail "Không gọi được RPC endpoint."
    print_fail "Kiểm tra Besu đang chạy và BESU_RPC_URL đúng."
    exit 1
  fi

  if [[ -z "$chain_id_hex" ]]; then
    print_fail "RPC trả về nhưng không đọc được chain id."
    exit 1
  fi

  chain_id_dec="$(hex_to_dec "$chain_id_hex")"
  print_ok "RPC phản hồi. Chain ID hiện tại: $chain_id_dec ($chain_id_hex)"

  if [[ "$chain_id_dec" != "$EXPECTED_CHAIN_ID" ]]; then
    print_warn "Chain ID không khớp giá trị mong đợi: $EXPECTED_CHAIN_ID"
  else
    print_ok "Chain ID khớp cấu hình mong đợi."
  fi

  block_1_hex="$(json_rpc "eth_blockNumber" | extract_result)"
  block_1_dec="$(hex_to_dec "$block_1_hex")"
  print_ok "Block hiện tại: $block_1_dec ($block_1_hex)"

  sleep "$SECOND_SAMPLE_DELAY_SEC"

  block_2_hex="$(json_rpc "eth_blockNumber" | extract_result)"
  block_2_dec="$(hex_to_dec "$block_2_hex")"

  if (( block_2_dec > block_1_dec )); then
    print_ok "Block đang tăng: $block_1_dec -> $block_2_dec"
  elif (( block_2_dec == block_1_dec )); then
    print_warn "Block chưa tăng trong ${SECOND_SAMPLE_DELAY_SEC}s. Có thể bình thường nếu mạng đang chậm hoặc proposer chưa tới lượt."
  else
    print_warn "Block giảm bất thường: $block_1_dec -> $block_2_dec"
  fi

  peer_count_hex="$(json_rpc "net_peerCount" | extract_result)"
  peer_count_dec="$(hex_to_dec "$peer_count_hex")"
  print_ok "Số peer đang thấy: $peer_count_dec ($peer_count_hex)"

  syncing_raw="$(curl -sS --fail \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
    "$RPC_URL")"

  if [[ "$syncing_raw" == *'"result":false'* ]]; then
    print_ok "Node không ở trạng thái syncing."
  else
    print_warn "Node đang syncing hoặc RPC trả về object syncing."
  fi

  if [[ -n "$CONTRACT_ADDRESS" && "$CONTRACT_ADDRESS" != "0x0000000000000000000000000000000000000000" ]]; then
    code_hex="$(json_rpc "eth_getCode" "[\"$CONTRACT_ADDRESS\",\"latest\"]" | extract_result)"

    if [[ -n "$code_hex" && "$code_hex" != "0x" ]]; then
      print_ok "Contract có code tại $CONTRACT_ADDRESS"
    else
      print_warn "Không thấy bytecode tại $CONTRACT_ADDRESS"
    fi
  else
    print_warn "Chưa kiểm tra contract vì BESU_CONTRACT_ADDRESS chưa được cấu hình hợp lệ."
  fi
}

main "$@"
