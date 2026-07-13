#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.besu-qbft.yml}"
RECOVER_SERVICE="${RECOVER_SERVICE:-}"
RECOVERY_WAIT_SEC="${RECOVERY_WAIT_SEC:-20}"
CHECK_ATTEMPTS="${CHECK_ATTEMPTS:-12}"
CHECK_WAIT_SEC="${CHECK_WAIT_SEC:-5}"

validators=(besu-validator1 besu-validator2 besu-validator3 besu-validator4)

usage() {
  cat <<EOF
Usage:
  ./scripts/ensure-besu-peers.sh
  RECOVER_SERVICE=besu-validator2 ./scripts/ensure-besu-peers.sh

Without RECOVER_SERVICE this command is read-only. Recovery restarts exactly one
validator and refuses any other service name, preventing accidental quorum loss.
EOF
}

[[ "${1:-}" =~ ^(-h|--help|help)$ ]] && { usage; exit 0; }

if [[ -n "$RECOVER_SERVICE" ]]; then
  allowed=0
  for validator in "${validators[@]}"; do
    [[ "$RECOVER_SERVICE" == "$validator" ]] && allowed=1
  done
  (( allowed == 1 )) || { echo "Refusing to restart non-validator or multiple services: $RECOVER_SERVICE" >&2; exit 1; }
  echo "Rolling recovery: restarting only $RECOVER_SERVICE"
  docker compose -f "$SERVER_DIR/$COMPOSE_FILE" restart "$RECOVER_SERVICE"
  sleep "$RECOVERY_WAIT_SEC"
fi

for attempt in $(seq 1 "$CHECK_ATTEMPTS"); do
  if "$SCRIPT_DIR/check-besu-cluster.sh"; then
    exit 0
  fi
  if (( attempt < CHECK_ATTEMPTS )); then
    echo "Besu cluster not ready (attempt $attempt/$CHECK_ATTEMPTS); retrying in ${CHECK_WAIT_SEC}s..." >&2
    sleep "$CHECK_WAIT_SEC"
  fi
done

echo "Besu cluster remained unhealthy after $CHECK_ATTEMPTS attempts." >&2
exit 1
