# 07. Troubleshooting

## 1. Vault unhealthy
Nguyên nhân thường gặp:
- Vault chưa init
- Vault chưa unseal
- `VAULT_TOKEN` trong `.env` sai

Kiểm tra:
```bash
docker compose -f docker-compose.deploy.yml exec vault vault status -address=http://127.0.0.1:8200
```

Bạn cần thấy:
- `Initialized true`
- `Sealed false`

## 2. Besu chạy nhưng backend không anchor được
Kiểm tra:
- `BESU_ENABLED=true`
- `BESU_RPC_URL` đúng
- `BESU_CONTRACT_ADDRESS` đúng
- `BESU_PRIVATE_KEY` hoặc `BESU_FROM_ADDRESS` đúng

Nếu backend chạy trong Docker mà bạn lại dùng:

```dotenv
BESU_RPC_URL=http://127.0.0.1:8545
```

thì thường sẽ lỗi, vì `127.0.0.1` lúc đó trỏ vào chính container API.

Khi backend chạy trong Docker Compose, hãy dùng:

```dotenv
BESU_RPC_URL=http://besu-validator1:8545
```

Khi backend chạy trên host, hãy dùng:

```dotenv
BESU_RPC_URL=http://127.0.0.1:8545
```

Khi backend chạy ở server khác, hãy dùng IP hoặc domain thật của Besu RPC:

```dotenv
BESU_RPC_URL=http://10.0.0.11:8545
```

## 3. IPFS chạy nhưng không có `imageCid`
Kiểm tra:
- `IPFS_ENABLED=true`
- `IPFS_API_URL` đúng
- image upload có thực sự đi qua `POST /v1/media/images`

Giống Besu:
- backend trong Docker: `IPFS_API_URL=http://ipfs:5001`
- backend trên host: `IPFS_API_URL=http://127.0.0.1:5001`
- backend ở máy khác: `IPFS_API_URL=http://<IP_IPFS>:5001`

## 3b. Vault chạy nhưng app không ký được
Kiểm tra:
- backend trong Docker: `VAULT_ADDR=http://vault:8200`
- backend trên host: `VAULT_ADDR=http://127.0.0.1:8200`
- backend ở máy khác: `VAULT_ADDR=http://<IP_VAULT>:8200`

## 4. Firebase lỗi
Kiểm tra:
- `FIREBASE_PROJECT_ID`
- file `secrets/firebase-service-account.json`
- `FIREBASE_CREDENTIALS_FILE=./secrets/firebase-service-account.json`

## 5. Muốn xóa local stack và chạy lại
```bash
make down
docker compose -f docker-compose.deploy.yml down --remove-orphans
```

Nếu cần reset Vault local:
```bash
docker volume ls | grep vault
docker volume rm server_vault-data
```

## 6. Quên mất Vault unseal keys
Nếu chỉ là local dev, cách đơn giản nhất là xóa volume Vault và init lại từ đầu.

## 7. Docker báo lỗi network not found
Chạy:
```bash
docker compose -f docker-compose.deploy.yml down --remove-orphans
docker network prune -f
make run-all
```
