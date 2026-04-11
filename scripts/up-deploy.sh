#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
TMP_CONFIG_FILE="$(mktemp "${TMPDIR:-/tmp}/vngrocery-deploy-compose.XXXXXX.txt")"
trap 'rm -f "$TMP_CONFIG_FILE"' EXIT

echo "Validating deploy compose..."
docker compose -f "$COMPOSE_FILE" config >"$TMP_CONFIG_FILE"

echo "Starting deploy stack..."
docker compose -f "$COMPOSE_FILE" up -d --build "$@"

echo
echo "Stack started. Useful commands:"
echo "  docker compose -f $COMPOSE_FILE ps"
echo "  docker compose -f $COMPOSE_FILE logs -f api"
echo "  docker compose -f $COMPOSE_FILE logs -f vault"
echo "  docker compose -f $COMPOSE_FILE logs -f besu-validator1"
