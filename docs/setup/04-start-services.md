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

## Chọn IP hoặc URL đúng theo cách bạn chạy

### Trường hợp 1: mọi thứ chạy cùng trong Docker Compose
Đây là cách dễ nhất.

Dùng:

```dotenv
VAULT_ADDR=http://vault:8200
IPFS_API_URL=http://ipfs:5001
IPFS_GATEWAY_URL=http://ipfs:8080
BESU_RPC_URL=http://besu-validator1:8545
```

Lý do:
- `vault`, `ipfs`, `besu-validator1` là tên service trong Docker network
- backend container gọi nhau bằng tên service, không dùng `127.0.0.1`

### Trường hợp 2: backend chạy trên máy host, còn services chạy bằng Docker trên cùng máy
Dùng:

```dotenv
VAULT_ADDR=http://127.0.0.1:8200
IPFS_API_URL=http://127.0.0.1:5001
IPFS_GATEWAY_URL=http://127.0.0.1:8080
BESU_RPC_URL=http://127.0.0.1:8545
```

Lý do:
- backend không ở trong Docker network
- backend host phải gọi vào cổng publish ra localhost

### Trường hợp 3: blockchain ở server khác
Ví dụ server blockchain có IP `10.10.10.20`

Dùng:

```dotenv
BESU_RPC_URL=http://10.10.10.20:8545
```

Nếu Vault/IPFS cũng ở server khác thì thay tương tự:

```dotenv
VAULT_ADDR=http://10.10.10.30:8200
IPFS_API_URL=http://10.10.10.40:5001
IPFS_GATEWAY_URL=http://10.10.10.40:8080
```

### Trường hợp 4: 4 validator nằm trên 4 server riêng
Ví dụ:
- validator1: `10.0.0.11`
- validator2: `10.0.0.12`
- validator3: `10.0.0.13`
- validator4: `10.0.0.14`

Backend chỉ cần 1 RPC endpoint để ghi và đọc chain.
Thường chọn validator1 hoặc một RPC gateway riêng:

```dotenv
BESU_RPC_URL=http://10.0.0.11:8545
```

Backend không cần gọi cả 4 validator cùng lúc.
4 validator chỉ cần nhìn thấy nhau qua mạng riêng để chạy QBFT.

## Khi nào dùng `127.0.0.1`, khi nào không
- Dùng `127.0.0.1` khi backend chạy trên chính máy đó
- Dùng tên service như `vault`, `ipfs`, `besu-validator1` khi backend chạy trong Docker Compose
- Dùng IP private như `10.x.x.x` khi service chạy ở máy khác trong mạng nội bộ

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
