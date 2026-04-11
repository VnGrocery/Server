# 04. Start Services

## Cách dễ nhất
Chạy:

```bash
make run-all
```

Script sẽ:
- check compose
- start Vault trước
- dừng lại nếu Vault chưa init hoặc chưa unseal
- start toàn bộ stack khi Vault đã sẵn sàng

## Nếu Vault chưa init
Script sẽ in ra lệnh kiểu này:

```bash
docker compose -f docker-compose.deploy.yml exec vault vault operator init -address=http://127.0.0.1:8200
```

Sau khi chạy lệnh đó:
- lưu `Unseal Key 1..5`
- lưu `Initial Root Token`

## Unseal Vault
Dùng bất kỳ 3 key:

```bash
docker compose -f docker-compose.deploy.yml exec vault vault operator unseal -address=http://127.0.0.1:8200 "<UNSEAL_KEY_1>"
docker compose -f docker-compose.deploy.yml exec vault vault operator unseal -address=http://127.0.0.1:8200 "<UNSEAL_KEY_2>"
docker compose -f docker-compose.deploy.yml exec vault vault operator unseal -address=http://127.0.0.1:8200 "<UNSEAL_KEY_3>"
```

## Login Vault
```bash
docker compose -f docker-compose.deploy.yml exec vault vault login -address=http://127.0.0.1:8200 "<ROOT_TOKEN>"
```

## Bật KV v2 nếu chưa có
```bash
docker compose -f docker-compose.deploy.yml exec vault vault secrets enable -address=http://127.0.0.1:8200 -path=secret kv-v2
```

## Điền lại `.env`
Sau khi có root token:

```dotenv
VAULT_ENABLED=true
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=<ROOT_TOKEN>
IPFS_ENABLED=true
IPFS_API_URL=http://ipfs:5001
IPFS_GATEWAY_URL=http://ipfs:8080
BESU_ENABLED=true
BESU_RPC_URL=http://besu-validator1:8545
BESU_CHAIN_ID=1337
```

## Chạy lại toàn bộ stack
```bash
make run-all
```

## Nếu chỉ muốn chạy riêng từng phần
```bash
make vault-up
make ipfs-up
make besu-up
make api-up
```
