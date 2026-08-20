# Deploying VnGrocery

One command, one entry point: `./scripts/vng`.

```bash
./scripts/vng chain-up          # 1. start the blockchain
./scripts/vng contract-deploy   # 2. deploy IntegrityRegistry (writes .env for you)
./scripts/vng up                # 3. start everything else
./scripts/vng health            # 4. verify
```

That is the whole happy path. `./scripts/vng help` lists everything.

## What runs

| Service | Port | Purpose |
|---|---|---|
| `api` | 5000 | Go backend |
| `mongo` | 27017 | primary datastore |
| `vault` | 8200 | account keys (dev mode auto-unseals) |
| `ipfs` | 5001 / 8080 | product image storage |
| `redis` | 6379 | read cache |
| `besu` | 8545 | single-node QBFT chain for hash anchoring |

## Profiles

| Command | Compose files used |
|---|---|
| `./scripts/vng up` | `docker-compose.yml` + `docker-compose.besu.yml` |
| `./scripts/vng up --prod` | the above + `docker-compose.prod.yml` (nginx proxy, persistent Vault, `restart: always`, log rotation) |
| `./scripts/vng up --qbft` | `docker-compose.yml` + `docker-compose.besu-qbft.yml` (4 validators, demo only) |
| `./scripts/vng up --no-chain` | `docker-compose.yml` only, no hash anchoring |

## Why a single Besu node by default

QBFT tolerates `f` faulty nodes only when `n >= 3f+1`. The 4-validator cluster
therefore survives exactly one failure: the moment two validators desync, block
production stops entirely — which is what made the old setup feel unreliable.

A single node has nothing to stay in sync with, so that failure mode disappears.
The chain still produces real blocks, the contract still anchors real hashes, and
`IntegrityView` still verifies them. Use `--qbft` when a multi-validator demo is
the point; use the default for everything else.

## Production notes

`--prod` swaps dev Vault for the file-backed one, which must be initialised once:

```bash
./scripts/vng --prod vault-init
```

It also requires `VAULT_TOKEN` in `.env` — the dev root token is deliberately not
accepted there.

## Resetting

```bash
./scripts/vng reset             # drops all containers and volumes, asks first
./scripts/vng contract-deploy   # redeploy the contract afterwards
```

## Files

```
docker-compose.yml            base stack
docker-compose.besu.yml       single-node chain (default)
docker-compose.besu-qbft.yml  4-validator cluster (demo)
docker-compose.prod.yml       production overlay
scripts/vng                   entry point
scripts/deploy-integrity.sh   contract deployment (called by vng)
scripts/check-blockchain.sh   chain health (called by vng health)
scripts/e2e-mobile-flow.sh    end-to-end API test (called by vng e2e)
scripts/check-besu-cluster.sh  }  only used by --qbft
scripts/ensure-besu-peers.sh   }
```
