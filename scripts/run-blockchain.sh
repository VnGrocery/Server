#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.besu-qbft.yml}"
COMPOSE_PATH="$SERVER_DIR/$COMPOSE_FILE"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/run-blockchain.sh up [docker-compose up args...]
  ./scripts/run-blockchain.sh down [docker-compose down args...]
  ./scripts/run-blockchain.sh restart
  ./scripts/run-blockchain.sh status
  ./scripts/run-blockchain.sh logs [service]
  ./scripts/run-blockchain.sh reset

Environment:
  COMPOSE_FILE   Compose file name relative to server/ (default: docker-compose.besu-qbft.yml)

Examples:
  ./scripts/run-blockchain.sh up -d
  ./scripts/run-blockchain.sh logs besu-validator1
  ./scripts/run-blockchain.sh reset
EOF
}

run_compose() {
  docker compose -f "$COMPOSE_PATH" "$@"
}

require_compose_file() {
  if [[ ! -f "$COMPOSE_PATH" ]]; then
    echo "Compose file not found: $COMPOSE_PATH" >&2
    exit 1
  fi
}

reset_chain_data() {
  local validator_dir

  echo "Resetting QBFT chain data..."
  for validator_dir in \
    "$SERVER_DIR/besu/qbft/data/validator1" \
    "$SERVER_DIR/besu/qbft/data/validator2" \
    "$SERVER_DIR/besu/qbft/data/validator3" \
    "$SERVER_DIR/besu/qbft/data/validator4"; do
    find "$validator_dir" -mindepth 1 -delete
    touch "$validator_dir/.gitkeep"
  done
}

main() {
  local command="${1:-}"
  shift || true

  require_compose_file

  case "$command" in
    up)
      echo "Validating compose file: $COMPOSE_FILE"
      run_compose config >/dev/null
      echo "Starting Besu QBFT network..."
      run_compose up --build "$@"
      ;;
    down)
      echo "Stopping Besu QBFT network..."
      run_compose down "$@"
      ;;
    restart)
      echo "Restarting Besu QBFT network..."
      run_compose down
      run_compose up --build -d
      run_compose ps
      ;;
    status)
      run_compose ps
      ;;
    logs)
      if [[ $# -gt 0 ]]; then
        run_compose logs -f "$1"
      else
        run_compose logs -f
      fi
      ;;
    reset)
      echo "Stopping network before reset..."
      run_compose down -v --remove-orphans || true
      reset_chain_data
      echo "Chain data reset complete."
      ;;
    ""|-h|--help|help)
      usage
      ;;
    *)
      echo "Unknown command: $command" >&2
      echo >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
