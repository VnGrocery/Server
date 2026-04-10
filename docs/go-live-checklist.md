# Go-Live Checklist

## Infra
- Deploy Besu QBFT validators on private hosts or private subnets.
- Run Kubo IPFS with persistent volumes and pinning policy.
- Run Vault with persistent storage and unseal process documented.
- Put reverse proxy/TLS in front of the API.
- Restrict Besu RPC, Vault, and IPFS API to private network access only.
- Validate persistent disk sizing for Firestore export, Vault snapshots, and IPFS pinned content.

## Secrets
- Store `BESU_PRIVATE_KEY`, Vault token, and Firebase credentials in a secret manager.
- Do not keep production secrets in `.env`.
- Rotate deployer and alert credentials before go-live.
- Verify only the API service account can submit integrity writes.

## Backend
- Set `RATE_LIMIT_BACKEND=firestore`.
- Enable alert routing with at least one of Slack, Telegram, SMTP, or generic webhook.
- Run `go run ./cmd/backfill-integrity --batch-size 200` on legacy data.
- Verify `GET /v1/shops/:shopId/pledges/:pledgeId/proof` returns sane proof states.
- Run [scripts/e2e-mobile-flow.sh](/home/dora/VNGrocery/server/scripts/e2e-mobile-flow.sh) against staging with a real image.
- Verify bootstrap admin policy if `BOOTSTRAP_ADMIN_EMAILS` is used.

## Data
- Deploy Firestore indexes from `firestore.indexes.json`.
- Confirm backup strategy for Firestore, Vault, and IPFS.
- Validate retention policy for `event_logs`, `buyer_checks`, and `product_freshness_reports`.
- Confirm `imageCid` is present in new seller score, buyer check, and freshness report flows when `IPFS_ENABLED=true`.

## Monitoring
- Scrape `/metrics`.
- Alert on anchor failures, verify mismatches, and repeated rate-limit rejections.
- Keep API, Vault, IPFS, and Besu logs centralized.
- Confirm alert delivery on at least one real destination.

## Rollout
- Run staging for at least several days with anchor and verify workers enabled.
- Cut over with production `BESU_CONTRACT_ADDRESS` and `BESU_PRIVATE_KEY`.
- Keep a rollback plan for contract address, signer key, and proxy config.
- Freeze schema-affecting changes during initial cutover window.
- Record final contract address, signer wallet, and IPFS gateway in the release note.

## Sign-off
- Backend sign-off
- Mobile sign-off
- Ops sign-off
- Security sign-off
