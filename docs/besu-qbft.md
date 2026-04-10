# Besu QBFT Local Network

This repo includes a 4-validator Besu QBFT network for local development.

## Start

Run Besu together with the existing API and Vault stack:

```bash
docker compose -f docker-compose.yml -f docker-compose.besu-qbft.yml up --build
```

The default compose stack now also includes Kubo IPFS:

- API: `5000`
- Vault: `8200`
- IPFS API: `5001`
- IPFS gateway: `8080`

Validator RPC ports on the host:

- `8545` -> `besu-validator1`
- `8546` -> `besu-validator2`
- `8547` -> `besu-validator3`
- `8548` -> `besu-validator4`

P2P ports:

- `30303` to `30306`

## Backend config

For the Go API, point the backend at validator 1:

```dotenv
BESU_ENABLED=true
BESU_RPC_URL=http://besu-validator1:8545
BESU_CHAIN_ID=1337
```

If the API runs on the host instead of Docker, use:

```dotenv
BESU_RPC_URL=http://127.0.0.1:8545
```

`BESU_CONTRACT_ADDRESS` must be set after deploying `contracts/IntegrityRegistry.sol`.

## Deploy IntegrityRegistry

Use the helper script:

```bash
chmod +x scripts/deploy-integrity.sh
./scripts/deploy-integrity.sh
```

Defaults:

- RPC: `http://127.0.0.1:8545`
- chain id: `1337`
- owner/deployer address: `0xfe3b557e8fb62b89f4916b721be55ceb828dbd73`

The script uses a Dockerized Foundry image to:

- compile `contracts/IntegrityRegistry.sol`
- sign a raw transaction with the prefunded dev private key from genesis
- deploy the contract
- print the `BESU_CONTRACT_ADDRESS` you should copy into `.env`

For production-like backend setup on the private chain, also set:

```dotenv
BESU_PRIVATE_KEY=8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63
```

This lets the backend sign raw transactions locally instead of depending on unlocked RPC accounts.

For production-like API limits across multiple replicas, prefer:

```dotenv
RATE_LIMIT_BACKEND=firestore
RATE_LIMIT_COLLECTION=rate_limits
```

Override values if needed:

```bash
BESU_RPC_URL=http://127.0.0.1:8545 \
BESU_CHAIN_ID=1337 \
BESU_OWNER_ADDRESS=0xfe3b557e8fb62b89f4916b721be55ceb828dbd73 \
BESU_PRIVATE_KEY=8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63 \
./scripts/deploy-integrity.sh
```

## Reset chain data

To restart from genesis:

```bash
rm -rf besu/qbft/data/validator{1,2,3,4}/*
touch besu/qbft/data/validator{1,2,3,4}/.gitkeep
```

## Notes

- The network is private and intended for local/dev use only.
- Validator 1 is the bootnode.
- Genesis prefunds sample dev accounts from the upstream Besu example network.
- After changing `IntegrityRegistry.sol` to add revoke support, redeploy the contract and update `BESU_CONTRACT_ADDRESS`.
