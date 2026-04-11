#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BASE_COMPOSE_FILE="${BASE_COMPOSE_FILE:-docker-compose.yml}"
BLOCKCHAIN_COMPOSE_FILE="${BLOCKCHAIN_COMPOSE_FILE:-docker-compose.besu-qbft.yml}"
EXTRA_COMPOSE_FILE="${EXTRA_COMPOSE_FILE:-}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/run-full-stack.sh up [docker-compose up args...]
  ./scripts/run-full-stack.sh down [docker-compose down args...]
  ./scripts/run-full-stack.sh restart
  ./scripts/run-full-stack.sh status
  ./scripts/run-full-stack.sh logs [service]
  ./scripts/run-full-stack.sh config

Default compose set:
  - docker-compose.yml
  - docker-compose.besu-qbft.yml

Environment:
  BASE_COMPOSE_FILE         Base compose file under server/ (default: docker-compose.yml)
  BLOCKCHAIN_COMPOSE_FILE   Blockchain compose file under server/ (default: docker-compose.besu-qbft.yml)
  EXTRA_COMPOSE_FILE        Optional extra compose file under server/, for example docker-compose.staging.yml

Examples:
  ./scripts/run-full-stack.sh up -d
  EXTRA_COMPOSE_FILE=docker-compose.staging.yml ./scripts/run-full-stack.sh up -d
  ./scripts/run-full-stack.sh logs api
EOF
}

compose_args=()

append_compose_file() {
  local file_name="$1"
  local file_path="$SERVER_DIR/$file_name"

  if [[ ! -f "$file_path" ]]; then
    echo "Compose file not found: $file_path" >&2
    exit 1
  fi

  compose_args+=(-f "$file_path")
}

run_compose() {
  docker compose "${compose_args[@]}" "$@"
}

init_compose_files() {
  append_compose_file "$BASE_COMPOSE_FILE"
  append_compose_file "$BLOCKCHAIN_COMPOSE_FILE"

  if [[ -n "$EXTRA_COMPOSE_FILE" ]]; then
    append_compose_file "$EXTRA_COMPOSE_FILE"
  fi
}

print_stack_summary() {
  echo
  echo "Stack started. Useful endpoints:"
  echo "  API:                http://localhost:5000"
  echo "  Vault:              http://localhost:8200"
  echo "  IPFS API:           http://localhost:5001"
  echo "  IPFS gateway:       http://localhost:8080"
  echo "  Besu validator 1:   http://localhost:8545"
  echo "  Besu validator 2:   http://localhost:8546"
  echo "  Besu validator 3:   http://localhost:8547"
  echo "  Besu validator 4:   http://localhost:8548"
  echo
  echo "Useful commands:"
  echo "  ./scripts/run-full-stack.sh status"
  echo "  ./scripts/run-full-stack.sh logs api"
  echo "  ./scripts/run-full-stack.sh logs besu-validator1"
  echo "  ./scripts/deploy-integrity.sh"
}

main() {
  local command="${1:-}"
  shift || true

  init_compose_files

  case "$command" in
    up)
      echo "Validating compose stack..."
      run_compose config >/dev/null
      echo "Starting full stack..."
      run_compose up --build "$@"
      print_stack_summary
      ;;
    down)
      echo "Stopping full stack..."
      run_compose down "$@"
      ;;
    restart)
      echo "Restarting full stack..."
      run_compose down
      run_compose up --build -d
      run_compose ps
      print_stack_summary
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
    config)
      run_compose config
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
