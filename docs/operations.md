# Operations Notes

## Vault local persistence

Default `docker-compose.yml` still runs Vault dev mode for fast local development. Dev mode is ephemeral and loses keys when the container is recreated.

For local persistent Vault, run:

```bash
docker compose -f docker-compose.yml -f docker-compose.vault-persistent.yml up --build
```

The persistent Vault uses file storage at the named Docker volume `vault-data`. It must be initialized and unsealed with standard Vault commands before the API can sign events:

```bash
docker compose exec vault vault operator init
docker compose exec vault vault operator unseal
docker compose exec vault vault login
vault secrets enable -path=secret kv-v2
```

Use the generated root token in `.env` as `VAULT_TOKEN`. Do not use the dev token `root` for persistent or shared environments.

## Migration/backfill strategy

For existing data, run backfills in this order:

1. Backfill account keys for auth users that lack Vault key metadata.
2. Recreate missing Vault secrets for users whose Firestore metadata exists but Vault secret is missing.
3. Backfill `status` and `version` on legacy resources before enabling blockchain sync.
4. Backfill product catalog fields with empty defaults: `category`, `tags`, `imageUrls`, `freshnessNote`, `freshnessScore`.
5. Rebuild derived summaries like Trust Score from source collections: `pledges`, `buyer_checks`, `shop_reviews`, and `product_freshness_reports`.

Backfills should be idempotent and should emit signed event logs when they change user-visible state.

For pledge integrity metadata, use:

```bash
go run ./cmd/backfill-integrity --batch-size 200 --start-after=<pledge-id> --dry-run
```

With `BESU_ENABLED=false`, this fills `dataHash`, `chainAnchorStatus=pending_anchor`, and `integrityStatus=pending_anchor` for legacy pledges.

With `BESU_ENABLED=true`, it also attempts to anchor pending legacy pledges onto Besu.

## Firestore indexes

Keep `firestore.indexes.json` in sync with query patterns before production import. Deploy indexes with Firebase tooling for the target project.

## CI/CD

The baseline CI runs `go test ./...` on push and pull request. Add deploy jobs only after secrets and target environments are finalized.

For manual end-to-end smoke testing before release, run:

```bash
IMAGE_PATH=/abs/path/to/image.jpg ./scripts/e2e-mobile-flow.sh
```

## Staging and production compose

For staging-style local validation with persistent Vault and reverse proxy:

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.vault-persistent.yml \
  -f docker-compose.besu-qbft.yml \
  -f docker-compose.staging.yml \
  up --build
```

For production-oriented compose layering in this repo:

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.vault-persistent.yml \
  -f docker-compose.prod.yml \
  up -d
```

The production file is only a repo baseline. Replace local mounts, secrets, and host ports with environment-specific infrastructure controls before real deployment.

For frontend and QA handoff, use:

- `docs/frontend-handoff-final.md`
- `docs/mobile-handoff.md`
- `docs/mobile-api-playbook.md`

## Besu QBFT integrity anchoring

When `BESU_ENABLED=true`, the backend computes a canonical `dataHash` for each new pledge, submits `commitHash(...)` to the configured `IntegrityRegistry`, and stores chain metadata back into Firestore.

Required env vars:

```bash
BESU_RPC_URL=http://127.0.0.1:8545
BESU_CONTRACT_ADDRESS=0x...
BESU_FROM_ADDRESS=0x...
```

For production, prefer local signing with:

```bash
BESU_PRIVATE_KEY=...
```

When `BESU_PRIVATE_KEY` is set, the backend signs raw transactions locally and submits them with `eth_sendRawTransaction`, so the Besu RPC node does not need unlocked accounts.

Optional alerting:

```bash
ALERT_WEBHOOK_URL=https://your-alert-endpoint
ALERT_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
ALERT_TELEGRAM_BOT_TOKEN=...
ALERT_TELEGRAM_CHAT_ID=...
ALERT_SMTP_HOST=smtp.example.com
ALERT_SMTP_PORT=587
ALERT_SMTP_USERNAME=...
ALERT_SMTP_PASSWORD=...
ALERT_SMTP_FROM=alerts@example.com
ALERT_SMTP_TO=ops@example.com,security@example.com
```

The background worker retries `pending_anchor` pledges, periodically verifies `anchored` pledges against on-chain state, and emits webhook alerts on `mismatch_detected`.

## Kubo IPFS

When `IPFS_ENABLED=true`, the backend uploads raw image bytes from `seller/score` and `buyer/check` into Kubo and returns/stores `imageCid`.

For reusable uploads from mobile clients, use:

`POST /v1/media/images`

Required env vars:

```bash
IPFS_API_URL=http://127.0.0.1:5001
IPFS_GATEWAY_URL=http://127.0.0.1:8080
```

Current usage:

- `POST /v1/seller/score` returns `imageHash` and `imageCid`
- `POST /v1/seller/commit` accepts optional `imageCid`
- `POST /v1/buyer/check` stores uploaded image as `imageCid`
- product freshness reports accept optional `imageCid`
- `POST /v1/media/images` returns `imageHash`, `imageCid`, `gatewayUrl`, `contentType`, `sizeBytes`

For pledges, canonical `dataHash` now includes `imageCid` when present.

## Rate limiting

The API supports:

- `RATE_LIMIT_BACKEND=memory` for simple local/dev
- `RATE_LIMIT_BACKEND=firestore` for multi-instance deployments

Relevant env vars:

```bash
RATE_LIMIT_BACKEND=firestore
RATE_LIMIT_COLLECTION=rate_limits
RATE_LIMIT_MAX_REQUESTS=120
RATE_LIMIT_WINDOW_SEC=60
```

Authenticated requests are limited by `user:<userId>`; unauthenticated requests fall back to client IP.

## Integrity runbook

When a pledge enters `mismatch_detected`:

1. Check `GET /v1/shops/:shopId/pledges/:pledgeId/integrity`
2. For buyer-facing rendering, prefer `GET /v1/shops/:shopId/pledges/:pledgeId/proof`
2. Compare `dataHash`, `onChainDataHash`, and `mismatchReason`
3. If the DB version is correct and you want the chain to follow current DB state, use `POST /v1/admin/shops/:shopId/pledges/:pledgeId/reanchor`
4. If the pledge must be invalidated, use `POST /v1/admin/shops/:shopId/pledges/:pledgeId/revoke`

These actions create new immutable chain states and new signed audit events; they do not erase prior history.
