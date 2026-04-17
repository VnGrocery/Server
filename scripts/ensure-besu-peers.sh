#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-8}"
SLEEP_SEC="${SLEEP_SEC:-2}"
SERVICES=(besu-validator1 besu-validator2 besu-validator3 besu-validator4)

usage() {
  cat <<EOF
Usage:
  ./scripts/ensure-besu-peers.sh

Environment:
  COMPOSE_FILE   Compose file to use (default: docker-compose.deploy.yml)
  MAX_ATTEMPTS   Max retry loops (default: 8)
  SLEEP_SEC      Sleep between loops in seconds (default: 2)
EOF
}

run_compose() {
  docker compose -f "$COMPOSE_FILE" "$@"
}

rpc_port() {
  local service="$1"
  local mapped
  mapped="$(run_compose port "$service" 8545 | head -n 1 || true)"
  if [[ -z "$mapped" ]]; then
    return 1
  fi
  echo "${mapped##*:}"
}

json_rpc() {
  local port="$1"
  local method="$2"
  local params="${3:-[]}"
  curl -sS --max-time 5 \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":$params,\"id\":1}" \
    "http://127.0.0.1:$port"
}

extract_json_string_field() {
  local key="$1"
  sed -n "s/.*\"$key\":\"\\([^\"]*\\)\".*/\\1/p" | head -n 1
}

extract_json_result_hex() {
  sed -n 's/.*"result":"\(0x[0-9a-fA-F]\+\)".*/\1/p' | head -n 1
}

hex_to_dec() {
  local hex="${1#0x}"
  if [[ -z "$hex" ]]; then
    echo "0"
    return
  fi
  printf "%d\n" "$((16#$hex))"
}

main() {
  local arg="${1:-}"
  if [[ "$arg" =~ ^(-h|--help|help)$ ]]; then
    usage
    exit 0
  fi

  local ports=()
  local enodes=()
  local service port node_info enode

  echo "Resolving Besu RPC ports from $COMPOSE_FILE ..."
  for service in "${SERVICES[@]}"; do
    if ! port="$(rpc_port "$service")"; then
      echo "ERROR: cannot resolve mapped RPC port for $service" >&2
      exit 1
    fi
    ports+=("$port")
    echo "  $service -> 127.0.0.1:$port"
  done

  echo "Reading enode addresses ..."
  local i
  for i in "${!SERVICES[@]}"; do
    service="${SERVICES[$i]}"
    port="${ports[$i]}"
    if ! node_info="$(json_rpc "$port" "admin_nodeInfo")"; then
      echo "ERROR: admin_nodeInfo failed on $service ($port)" >&2
      exit 1
    fi
    enode="$(printf '%s' "$node_info" | extract_json_string_field "enode")"
    if [[ -z "$enode" ]]; then
      echo "ERROR: cannot extract enode for $service ($port)" >&2
      exit 1
    fi
    enodes+=("$enode")
  done

  local attempt all_good peer_hex peer_dec add_result
  for attempt in $(seq 1 "$MAX_ATTEMPTS"); do
    echo "Peer bootstrap attempt $attempt/$MAX_ATTEMPTS ..."

    for i in "${!SERVICES[@]}"; do
      port="${ports[$i]}"
      for enode in "${enodes[@]}"; do
        add_result="$(json_rpc "$port" "admin_addPeer" "[\"$enode\"]" || true)"
        if [[ "$add_result" == *'"error"'* ]]; then
          echo "WARN: admin_addPeer returned error on ${SERVICES[$i]}: $add_result" >&2
        fi
      done
    done

    all_good=1
    for i in "${!SERVICES[@]}"; do
      port="${ports[$i]}"
      peer_hex="$(json_rpc "$port" "net_peerCount" | extract_json_result_hex || true)"
      peer_dec="$(hex_to_dec "${peer_hex:-0x0}")"
      echo "  ${SERVICES[$i]} peers: $peer_dec (${peer_hex:-0x0})"
      if (( peer_dec < 1 )); then
        all_good=0
      fi
    done

    if (( all_good == 1 )); then
      echo "Besu peer mesh is healthy."
      exit 0
    fi

    sleep "$SLEEP_SEC"
  done

  echo "ERROR: Besu peers are still not connected after $MAX_ATTEMPTS attempts." >&2
  echo "Check logs: docker compose -f $COMPOSE_FILE logs --tail=100 besu-validator1 besu-validator2 besu-validator3 besu-validator4" >&2
  exit 1
}

main "$@"
