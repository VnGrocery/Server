#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
VAULT_SERVICE="${VAULT_SERVICE:-vault}"
VAULT_HTTP_ADDR="${VAULT_HTTP_ADDR:-http://127.0.0.1:8200}"
TMP_STATUS_FILE="$(mktemp "${TMPDIR:-/tmp}/vngrocery-vault-status.XXXXXX.txt")"
trap 'rm -f "$TMP_STATUS_FILE"' EXIT

run_compose() {
  docker compose -f "$COMPOSE_FILE" "$@"
}

echo "Using compose file: $COMPOSE_FILE"

run_compose up -d "$VAULT_SERVICE"

echo
echo "Waiting for Vault container to become reachable..."
for _ in $(seq 1 30); do
  if run_compose exec -T "$VAULT_SERVICE" vault status -address="$VAULT_HTTP_ADDR" >"$TMP_STATUS_FILE" 2>/dev/null; then
    break
  fi
  sleep 2
done

if run_compose exec -T "$VAULT_SERVICE" vault status -address="$VAULT_HTTP_ADDR" >"$TMP_STATUS_FILE" 2>/dev/null; then
  cat "$TMP_STATUS_FILE"
else
  echo "Vault is not reachable yet. Check logs with:"
  echo "  docker compose -f $COMPOSE_FILE logs $VAULT_SERVICE"
  exit 1
fi

if grep -q "Initialized[[:space:]]\+false" "$TMP_STATUS_FILE"; then
  echo
  echo "Vault is not initialized. Run:"
  echo "  docker compose -f $COMPOSE_FILE exec $VAULT_SERVICE vault operator init -address=$VAULT_HTTP_ADDR"
  echo
  echo "Save the unseal keys and root token, then unseal with:"
  echo "  docker compose -f $COMPOSE_FILE exec $VAULT_SERVICE vault operator unseal -address=$VAULT_HTTP_ADDR"
  echo
  echo "After unseal, enable KV v2 if needed:"
  echo "  docker compose -f $COMPOSE_FILE exec $VAULT_SERVICE vault login -address=$VAULT_HTTP_ADDR"
  echo "  docker compose -f $COMPOSE_FILE exec $VAULT_SERVICE vault secrets enable -address=$VAULT_HTTP_ADDR -path=secret kv-v2"
  exit 0
fi

if grep -q "Sealed[[:space:]]\+true" "$TMP_STATUS_FILE"; then
  echo
  echo "Vault is initialized but sealed. Unseal it with:"
  echo "  docker compose -f $COMPOSE_FILE exec $VAULT_SERVICE vault operator unseal -address=$VAULT_HTTP_ADDR"
  exit 0
fi

echo
echo "Vault is initialized and unsealed."
