# Production Stack

## Target topology
- `api`: Go backend behind TLS proxy
- `vault`: HA or managed Vault for account keys
- `ipfs`: Kubo node with persistent volume and gateway policy
- `besu`: private QBFT validators plus one internal RPC endpoint
- `firestore`: source of truth for app data

## Network patterns

### Pattern A: everything on one machine
- API, Vault, IPFS, Besu đều chạy cùng máy
- nếu backend ở Docker: dùng tên service
- nếu backend ở host: dùng `127.0.0.1`

### Pattern B: blockchain on another server
- API chạy server A
- Besu RPC chạy server B
- dùng:
  - `BESU_RPC_URL=http://<server-b-private-ip>:8545`

### Pattern C: one validator per server
- validator1: server B1
- validator2: server B2
- validator3: server B3
- validator4: server B4
- backend chỉ gọi 1 RPC endpoint:
  - validator1
  - hoặc RPC gateway riêng

Ví dụ:
```dotenv
BESU_RPC_URL=http://10.0.0.11:8545
```

### Pattern D: services all on separate servers
Ví dụ:
```dotenv
VAULT_ADDR=http://10.0.0.30:8200
IPFS_API_URL=http://10.0.0.40:5001
IPFS_GATEWAY_URL=http://10.0.0.40:8080
BESU_RPC_URL=http://10.0.0.11:8545
```

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
- Track `/metrics` for:
  - anchor attempts/failures/mismatches
  - rate-limit rejections
  - `bundle_token_issued_total`
  - `bundle_token_replay_total`
  - `buyer_check_retried_total`
- Retention:
  - `bundle_token_uses` is cleaned in background every 10 minutes (batch 500) by `expiresAt`
- Run `go run ./cmd/backfill-integrity --batch-size 200` after data migrations that affect pledges
- Run `IMAGE_PATH=/abs/path/to/image.jpg ./scripts/e2e-mobile-flow.sh` against staging before each release

## Minimum go-live checklist
- TLS in front of API and private RPC access for Besu
- secret rotation for `BESU_PRIVATE_KEY`, Vault token, Firebase credentials
- off-site backup for Vault and IPFS volumes
- documented contract redeploy and re-anchor runbook
