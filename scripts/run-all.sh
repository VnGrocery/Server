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
run_compose up -d vault

echo "Waiting for Vault status..."
for _ in $(seq 1 30); do
  if run_compose exec -T vault vault status -address="$VAULT_HTTP_ADDR" >"$TMP_STATUS_FILE" 2>/dev/null; then
    break
  fi
  sleep 2
done

if ! run_compose exec -T vault vault status -address="$VAULT_HTTP_ADDR" >"$TMP_STATUS_FILE" 2>/dev/null; then
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
echo "  docker compose -f $COMPOSE_FILE logs -f besu-validator1"
