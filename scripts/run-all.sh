#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
VAULT_HTTP_ADDR="${VAULT_HTTP_ADDR:-http://127.0.0.1:8200}"
TMP_STATUS_FILE="$(mktemp "${TMPDIR:-/tmp}/vngrocery-vault-runall.XXXXXX.txt")"
trap 'rm -f "$TMP_STATUS_FILE"' EXIT

run_compose() {
  docker compose -f "$COMPOSE_FILE" "$@"
}

echo "Checking deploy compose..."
run_compose config >/dev/null

echo "Starting Vault first..."
run_compose up -d redis vault

echo "Waiting for Redis health..."
redis_ready=0
for _ in $(seq 1 30); do
  if run_compose exec -T redis sh -lc "redis-cli ping" 2>/dev/null | grep -q "PONG"; then
    redis_ready=1
    break
  fi
  sleep 1
done

if [[ "$redis_ready" -ne 1 ]]; then
  echo "Redis is not ready yet."
  echo "Check:"
  echo "  docker compose -f $COMPOSE_FILE logs redis"
  exit 1
fi

echo "Waiting for Vault status..."
vault_reachable=0
for _ in $(seq 1 30); do
  if run_compose exec -T vault sh -lc "vault status -address='$VAULT_HTTP_ADDR' || true" >"$TMP_STATUS_FILE" 2>/dev/null; then
    if grep -Eq "Initialized|Sealed|Storage Type|HA Enabled" "$TMP_STATUS_FILE"; then
      vault_reachable=1
      break
    fi
  fi
  sleep 2
done

if [[ "$vault_reachable" -ne 1 ]]; then
  if run_compose exec -T vault sh -lc "vault status -address='$VAULT_HTTP_ADDR' || true" >"$TMP_STATUS_FILE" 2>/dev/null; then
    if grep -Eq "Initialized|Sealed|Storage Type|HA Enabled" "$TMP_STATUS_FILE"; then
      vault_reachable=1
    fi
  fi
fi

if [[ "$vault_reachable" -ne 1 ]]; then
  echo "Vault is not reachable yet."
  echo "Check:"
  echo "  docker compose -f $COMPOSE_FILE logs vault"
  exit 1
fi

cat "$TMP_STATUS_FILE"

if grep -q "Initialized[[:space:]]\+false" "$TMP_STATUS_FILE"; then
  echo
  echo "Vault is not initialized yet. Run these commands first:"
  echo "  docker compose -f $COMPOSE_FILE exec vault vault operator init -address=$VAULT_HTTP_ADDR"
  echo "  docker compose -f $COMPOSE_FILE exec vault vault operator unseal -address=$VAULT_HTTP_ADDR"
  echo "  docker compose -f $COMPOSE_FILE exec vault vault login -address=$VAULT_HTTP_ADDR"
  echo "  docker compose -f $COMPOSE_FILE exec vault vault secrets enable -address=$VAULT_HTTP_ADDR -path=secret kv-v2"
  exit 0
fi

if grep -q "Sealed[[:space:]]\+true" "$TMP_STATUS_FILE"; then
  echo
  echo "Vault is initialized but sealed. Unseal it first:"
  echo "  docker compose -f $COMPOSE_FILE exec vault vault operator unseal -address=$VAULT_HTTP_ADDR"
  exit 0
fi

echo
echo "Vault is ready. Starting full stack..."
run_compose up -d --build

echo
echo "Stack started. Useful commands:"
echo "  docker compose -f $COMPOSE_FILE ps"
echo "  docker compose -f $COMPOSE_FILE logs -f api"
echo "  docker compose -f $COMPOSE_FILE logs -f vault"
echo "  docker compose -f $COMPOSE_FILE logs -f redis"
echo "  docker compose -f $COMPOSE_FILE logs -f besu-validator1"
