# Production Stack

## Target topology
- `api`: Go backend behind TLS proxy
- `vault`: HA or managed Vault for account keys
- `ipfs`: Kubo node with persistent volume and gateway policy
- `besu`: private QBFT validators plus one internal RPC endpoint
- `firestore`: source of truth for app data

## Required env
- `BESU_ENABLED=true`
- `BESU_RPC_URL=https://besu-rpc.internal`
- `BESU_PRIVATE_KEY` from secret manager
- `BESU_CONTRACT_ADDRESS` deployed `IntegrityRegistry`
- `IPFS_ENABLED=true`
- `IPFS_API_URL=http://ipfs:5001`
- `IPFS_GATEWAY_URL=https://ipfs.example.com`
- `MEDIA_MAX_IMAGE_BYTES=10485760`
- `MEDIA_ALLOWED_TYPES=image/jpeg,image/png,image/webp`

## One-command local deploy baseline
- `./scripts/init-vault.sh`
- `./scripts/up-deploy.sh`
- `./scripts/run-all.sh`
- stack entrypoint file: `docker-compose.deploy.yml`

## Storage rules
- Keep raw images off-chain.
- Upload image to IPFS, store `imageCid`, include `imageCid` in pledge hash payload, anchor only canonical `dataHash` on Besu.
- Use persistent volumes for Vault and Kubo. Do not run either in dev-only mode in production.

## Operational checks
- Alert on `integrityStatus=mismatch_detected`
- Track `/metrics` for anchor attempts, anchor failures, verify mismatches, and rate-limit rejections
- Run `go run ./cmd/backfill-integrity --batch-size 200` after data migrations that affect pledges
- Run `IMAGE_PATH=/abs/path/to/image.jpg ./scripts/e2e-mobile-flow.sh` against staging before each release

## Minimum go-live checklist
- TLS in front of API and private RPC access for Besu
- secret rotation for `BESU_PRIVATE_KEY`, Vault token, Firebase credentials
- off-site backup for Vault and IPFS volumes
- documented contract redeploy and re-anchor runbook
