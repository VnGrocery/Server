# RUN-BLOCKCHAIN

Tài liệu chạy cụm blockchain (Besu QBFT) trong môi trường deploy của VNGrocery.

## 1) Chuẩn bị

```bash
cd /home/dora/VNGrocery/server
```

## 2) Chạy toàn bộ stack

```bash
make run-all
```

`run-all` sẽ:
- validate compose
- khởi động `redis` + `vault` trước
- kiểm tra Vault readiness
- nếu Vault sẵn sàng thì mới bật full stack (bao gồm Besu)

## 3) Nếu Vault chưa sẵn sàng

### 3.1 Kiểm tra trạng thái Vault

```bash
docker compose -f docker-compose.deploy.yml exec vault vault status -address=http://127.0.0.1:8200
```

### 3.2 Trường hợp `Initialized: false` (chưa init)

```bash
docker compose -f docker-compose.deploy.yml exec vault vault operator init -address=http://127.0.0.1:8200
```

Lưu lại:
- Unseal Keys
- Initial Root Token

### 3.3 Unseal Vault (cần đủ ngưỡng key, thường 3)

```bash
docker compose -f docker-compose.deploy.yml exec vault vault operator unseal -address=http://127.0.0.1:8200 "<UNSEAL_KEY_1>"
docker compose -f docker-compose.deploy.yml exec vault vault operator unseal -address=http://127.0.0.1:8200 "<UNSEAL_KEY_2>"
docker compose -f docker-compose.deploy.yml exec vault vault operator unseal -address=http://127.0.0.1:8200 "<UNSEAL_KEY_3>"
```

### 3.4 (Tuỳ chọn) Login Vault

```bash
docker compose -f docker-compose.deploy.yml exec vault vault login -address=http://127.0.0.1:8200 "<ROOT_TOKEN>"
```

### 3.5 Chạy lại full stack

```bash
make run-all
```

## 4) Chỉ bật blockchain (không bật toàn bộ stack)

```bash
docker compose -f docker-compose.deploy.yml up -d besu-validator1 besu-validator2 besu-validator3 besu-validator4
```

## 5) Kiểm tra RPC blockchain

```bash
for p in 8545 8546 8547 8548; do
  printf ":%s " "$p"
  curl -s -H 'Content-Type: application/json' \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    http://127.0.0.1:$p
  echo
done
```

Nếu trả về `result` dạng hex (`0x...`) là RPC đang hoạt động.

## 6) Sửa tình trạng node chưa bắt peer

```bash
COMPOSE_FILE=docker-compose.deploy.yml ./scripts/ensure-besu-peers.sh
```

Kiểm tra peer count:

```bash
for p in 8545 8546 8547 8548; do
  printf ":%s " "$p"
  curl -s -H 'Content-Type: application/json' \
    --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
    http://127.0.0.1:$p
  echo
done
```

## 7) Lệnh kiểm tra nhanh

### 7.1 Trạng thái container

```bash
docker compose -f docker-compose.deploy.yml ps
```

### 7.2 Log validator

```bash
docker compose -f docker-compose.deploy.yml logs -f besu-validator1
```

### 7.3 Log cả cụm Besu

```bash
docker compose -f docker-compose.deploy.yml logs --tail=200 besu-validator1 besu-validator2 besu-validator3 besu-validator4
```

## 8) Lỗi thường gặp

### Lỗi: Blockchain không chạy, chỉ thấy Redis/Vault
Nguyên nhân phổ biến: Vault đang `Sealed=true`, nên `run-all` dừng trước khi bật full stack.

Cách xử lý:
1. `vault status`
2. unseal đủ key
3. chạy lại `make run-all`

### Lỗi: RPC không phản hồi
- kiểm tra container `besu-validator*` đã Up chưa
- xem logs validator
- chạy `ensure-besu-peers.sh`

---

Gợi ý: Sau khi máy restart, luôn kiểm tra Vault trước, vì Vault sealed sẽ làm bạn tưởng blockchain bị hỏng dù thực ra cụm chưa được start.
