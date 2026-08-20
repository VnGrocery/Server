# Besu development and hardened QBFT profiles

## Local development

Local development uses one QBFT validator and a separate disposable chain:

```bash
./scripts/vng chain-up
```

RPC is available at `http://127.0.0.1:8545`. This profile has no Byzantine
fault tolerance and must never be used outside a developer machine.

Reset it with:

```bash
docker compose -f docker-compose.besu.yml down -v
```

## Hardened cluster profile

The cluster profile contains four validators, two non-validator RPC nodes and
an HAProxy endpoint:

```bash
make besu-cluster-up
make besu-health
```

Only HAProxy publishes RPC on host port `8545`. Validator RPC is disabled.
Validators use a static peer mesh plus two bootnode references; API traffic is
served by `besu-rpc1` and `besu-rpc2`.

`BESU_IMAGE` is pinned by default. Upgrade it first in staging, run the cluster
health/failover checks, then perform a one-validator-at-a-time rolling upgrade.

Backend configuration inside Compose:

```dotenv
BESU_ENABLED=true
BESU_RPC_URL=http://besu-rpc-proxy:8545
BESU_RPC_URLS=http://besu-rpc-proxy:8545,http://besu-rpc1:8545,http://besu-rpc2:8545
BESU_CHAIN_ID=1337
```

The Go client rotates across the list and retries the same signed raw
transaction against another endpoint when an RPC endpoint fails.

Run `BESU_WORKER_ENABLED=true` on exactly one API/worker replica. Set it to
`false` on other API replicas so multiple signers do not compete for the same
account nonce.

## Asynchronous anchoring

Creating a shop or pledge stores `pending_anchor` in the application database
and returns without waiting for Besu. The integrity worker performs the chain
write and persists:

- `chainAnchorOperation`
- `chainAnchorAttempts`
- `chainAnchorNextAttemptAt`
- `chainAnchorLastError`

Retries use exponential backoff capped at five minutes. Before resubmitting,
the worker queries the contract and reconciles an already committed matching
record, making recovery idempotent after an ambiguous RPC failure.

## Deploy IntegrityRegistry

Start the desired profile, then run:

```bash
./scripts/deploy-integrity.sh
```

Copy the printed `BESU_CONTRACT_ADDRESS` into `.env`. The repository's funded
accounts and validator keys are disposable development credentials. Generate
new keys outside Git and load the signer from a secret manager for every real
environment.

## Consensus and existing data

The hardened genesis uses a 5-second block period and 10-second request timeout.
Changing genesis is not compatible with an existing chain database. Back up
required data, stop the old network, reset all validator/RPC data, redeploy the
contract and run the integrity backfill before starting this profile.

## Safe recovery

Health checks validate chain ID, peer count, block progression and RPC drift:

```bash
BESU_RPC_URLS=http://127.0.0.1:8545 ./scripts/check-besu-cluster.sh
```

Recovery may restart exactly one validator:

```bash
make besu-recover service=besu-validator2
```

Never restart two validators at once. A four-validator QBFT cluster loses
quorum when two validators are unavailable.

## Real production topology

The Compose cluster proves the topology but all containers still share one
host. For real fault tolerance, place each validator on an independent host or
failure zone, replace Docker IPs in `static-nodes.json` with private routable
addresses, use persistent SSD storage, and keep RPC/metrics on private networks.
