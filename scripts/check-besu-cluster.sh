#!/usr/bin/env bash
set -euo pipefail

RPC_URLS="${BESU_RPC_URLS:-http://127.0.0.1:8545}"
EXPECTED_CHAIN_ID="${BESU_CHAIN_ID:-1337}"
MIN_PEERS="${MIN_PEERS:-2}"
MAX_BLOCK_DRIFT="${MAX_BLOCK_DRIFT:-2}"
PROGRESS_WAIT_SEC="${PROGRESS_WAIT_SEC:-6}"

IFS=',' read -r -a urls <<< "$RPC_URLS"

rpc() {
  local url="$1" method="$2"
  curl -fsS --max-time 5 -H 'Content-Type: application/json' \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":[],\"id\":1}" "$url"
}

hex_result() {
  sed -n 's/.*"result":"\(0x[0-9a-fA-F]*\)".*/\1/p' | head -n 1
}

hex_to_dec() {
  local value="${1#0x}"
  [[ -n "$value" ]] || value=0
  printf '%d\n' "$((16#$value))"
}

sample() {
  local url="$1" chain_hex peer_hex block_hex
  chain_hex="$(rpc "$url" eth_chainId | hex_result)"
  peer_hex="$(rpc "$url" net_peerCount | hex_result)"
  block_hex="$(rpc "$url" eth_blockNumber | hex_result)"
  [[ "$(hex_to_dec "$chain_hex")" -eq "$EXPECTED_CHAIN_ID" ]] || return 1
  printf '%s %s\n' "$(hex_to_dec "$peer_hex")" "$(hex_to_dec "$block_hex")"
}

first_blocks=()
for raw_url in "${urls[@]}"; do
  url="${raw_url//[[:space:]]/}"
  [[ -n "$url" ]] || continue
  read -r peers block < <(sample "$url") || {
    echo "UNHEALTHY $url: RPC unavailable or chain ID mismatch" >&2
    exit 1
  }
  if (( peers < MIN_PEERS )); then
    echo "UNHEALTHY $url: peers=$peers, required=$MIN_PEERS" >&2
    exit 1
  fi
  first_blocks+=("$block")
  echo "SAMPLE $url peers=$peers block=$block"
done

(( ${#first_blocks[@]} > 0 )) || { echo "No Besu RPC endpoint configured" >&2; exit 1; }
sleep "$PROGRESS_WAIT_SEC"

min=-1
max=-1
advanced=0
index=0
for raw_url in "${urls[@]}"; do
  url="${raw_url//[[:space:]]/}"
  [[ -n "$url" ]] || continue
  read -r peers block < <(sample "$url") || {
    echo "UNHEALTHY $url after wait" >&2
    exit 1
  }
  (( block > first_blocks[index] )) && advanced=1
  (( min < 0 || block < min )) && min=$block
  (( max < 0 || block > max )) && max=$block
  echo "HEALTH $url peers=$peers block=$block"
  index=$((index + 1))
done

(( advanced == 1 )) || { echo "UNHEALTHY: block height did not progress" >&2; exit 1; }
drift=$((max - min))
(( drift <= MAX_BLOCK_DRIFT )) || { echo "UNHEALTHY: block drift=$drift, max=$MAX_BLOCK_DRIFT" >&2; exit 1; }
echo "HEALTHY: endpoints=${#first_blocks[@]} block=$max drift=$drift"
