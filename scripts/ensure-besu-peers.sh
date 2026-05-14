#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-8}"
SLEEP_SEC="${SLEEP_SEC:-2}"
MAX_BLOCK_DRIFT="${MAX_BLOCK_DRIFT:-24}"
HEAL_ON_DRIFT="${HEAL_ON_DRIFT:-1}"
HEAL_ON_ZERO_PEER="${HEAL_ON_ZERO_PEER:-1}"
HEAL_WAIT_SEC="${HEAL_WAIT_SEC:-5}"
HEAL_MAX_RESTARTS="${HEAL_MAX_RESTARTS:-2}"
NODEINFO_MAX_ATTEMPTS="${NODEINFO_MAX_ATTEMPTS:-30}"
NODEINFO_SLEEP_SEC="${NODEINFO_SLEEP_SEC:-2}"
SERVICES=(besu-validator1 besu-validator2 besu-validator3 besu-validator4)

usage() {
  cat <<EOF
Usage:
  ./scripts/ensure-besu-peers.sh

Environment:
  COMPOSE_FILE   Compose file to use (default: docker-compose.deploy.yml)
  MAX_ATTEMPTS           Max retry loops (default: 8)
  SLEEP_SEC              Sleep between loops in seconds (default: 2)
  MAX_BLOCK_DRIFT        Max allowed block gap between nodes (default: 24)
  HEAL_ON_DRIFT          Restart lagging nodes when drift is high (default: 1)
  HEAL_ON_ZERO_PEER      Restart nodes that keep peerCount=0 (default: 1)
  HEAL_WAIT_SEC          Wait after each healing restart (default: 5)
  HEAL_MAX_RESTARTS      Max restart count per node in one run (default: 2)
  NODEINFO_MAX_ATTEMPTS  Max retries waiting for admin_nodeInfo per node (default: 30)
  NODEINFO_SLEEP_SEC     Sleep between admin_nodeInfo retries (default: 2)
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

service_name_from_index() {
  local index="$1"
  echo "${SERVICES[$index]}"
}

restart_service_once() {
  local service="$1"
  local count="$2"
  echo "HEAL: restarting $service (restart count: $count/$HEAL_MAX_RESTARTS)"
  run_compose restart "$service" >/dev/null
  sleep "$HEAL_WAIT_SEC"
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
  local nodeinfo_attempt

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
    node_info=""
    for nodeinfo_attempt in $(seq 1 "$NODEINFO_MAX_ATTEMPTS"); do
      if node_info="$(json_rpc "$port" "admin_nodeInfo" 2>/dev/null || true)"; then
        enode="$(printf '%s' "$node_info" | extract_json_string_field "enode")"
        if [[ -n "$enode" ]]; then
          break
        fi
      fi
      if (( nodeinfo_attempt < NODEINFO_MAX_ATTEMPTS )); then
        echo "  waiting admin_nodeInfo on $service ($port) attempt $nodeinfo_attempt/$NODEINFO_MAX_ATTEMPTS ..."
        sleep "$NODEINFO_SLEEP_SEC"
      fi
    done
    if [[ -z "${enode:-}" ]]; then
      echo "ERROR: admin_nodeInfo failed on $service ($port) after $NODEINFO_MAX_ATTEMPTS attempts" >&2
      exit 1
    fi
    enodes+=("$enode")
  done

  local attempt all_good add_result
  local -a peer_counts=()
  local -a block_heights=()
  local -a zero_peer_services=()
  local -a drift_services=()
  local -A restart_counts=()
  local i service_i port_i
  local peer_hex peer_dec block_hex block_dec
  local min_block max_block drift
  local restart_key restart_next_count
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

    peer_counts=()
    block_heights=()
    zero_peer_services=()
    min_block=-1
    max_block=-1

    all_good=1
    for i in "${!SERVICES[@]}"; do
      service_i="$(service_name_from_index "$i")"
      port_i="${ports[$i]}"

      peer_hex="$(json_rpc "$port_i" "net_peerCount" | extract_json_result_hex || true)"
      peer_dec="$(hex_to_dec "${peer_hex:-0x0}")"
      echo "  $service_i peers: $peer_dec (${peer_hex:-0x0})"
      peer_counts+=("$peer_dec")
      if (( peer_dec < 1 )); then
        all_good=0
        zero_peer_services+=("$service_i")
      fi

      block_hex="$(json_rpc "$port_i" "eth_blockNumber" | extract_json_result_hex || true)"
      block_dec="$(hex_to_dec "${block_hex:-0x0}")"
      block_heights+=("$block_dec")
      if (( min_block < 0 || block_dec < min_block )); then
        min_block="$block_dec"
      fi
      if (( max_block < 0 || block_dec > max_block )); then
        max_block="$block_dec"
      fi
      echo "  $service_i block: $block_dec (${block_hex:-0x0})"
    done

    drift=$((max_block - min_block))
    echo "  cluster drift: $drift blocks (min=$min_block max=$max_block threshold=$MAX_BLOCK_DRIFT)"
    if (( drift > MAX_BLOCK_DRIFT )); then
      all_good=0
      drift_services=()
      for i in "${!SERVICES[@]}"; do
        service_i="$(service_name_from_index "$i")"
        if (( block_heights[$i] < max_block - MAX_BLOCK_DRIFT )); then
          drift_services+=("$service_i")
        fi
      done
      if (( ${#drift_services[@]} > 0 )); then
        echo "  WARN: lagging nodes: ${drift_services[*]}"
      fi
    fi

    if (( all_good == 1 )); then
      echo "Besu peer mesh is healthy."
      exit 0
    fi

    if (( HEAL_ON_ZERO_PEER == 1 )); then
      for service in "${zero_peer_services[@]}"; do
        restart_key="zero_peer:$service"
        restart_next_count=$(( ${restart_counts[$restart_key]:-0} + 1 ))
        if (( restart_next_count <= HEAL_MAX_RESTARTS )); then
          restart_counts[$restart_key]="$restart_next_count"
          restart_service_once "$service" "$restart_next_count"
        else
          echo "HEAL: skip restart for $service (zero peer) because restart limit reached."
        fi
      done
    fi

    if (( HEAL_ON_DRIFT == 1 && drift > MAX_BLOCK_DRIFT )); then
      for service in "${drift_services[@]}"; do
        restart_key="drift:$service"
        restart_next_count=$(( ${restart_counts[$restart_key]:-0} + 1 ))
        if (( restart_next_count <= HEAL_MAX_RESTARTS )); then
          restart_counts[$restart_key]="$restart_next_count"
          restart_service_once "$service" "$restart_next_count"
        else
          echo "HEAL: skip restart for $service (drift) because restart limit reached."
        fi
      done
    fi

    sleep "$SLEEP_SEC"
  done

  echo "ERROR: Besu peers are still not connected after $MAX_ATTEMPTS attempts." >&2
  echo "Hint: if one node keeps drifting, check node data/genesis consistency and consider resyncing that validator data dir." >&2
  echo "Check logs: docker compose -f $COMPOSE_FILE logs --tail=100 besu-validator1 besu-validator2 besu-validator3 besu-validator4" >&2
  exit 1
}

main "$@"
