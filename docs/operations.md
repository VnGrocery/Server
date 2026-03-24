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

## Firestore indexes

Keep `firestore.indexes.json` in sync with query patterns before production import. Deploy indexes with Firebase tooling for the target project.

## CI/CD

The baseline CI runs `go test ./...` on push and pull request. Add deploy jobs only after secrets and target environments are finalized.
