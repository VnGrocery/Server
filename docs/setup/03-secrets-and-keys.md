# 03. Secrets And Keys

## File này giải thích từng key để làm gì

### 1. Firebase service account file
- Dạng: file `.json`
- Lưu ở: `secrets/firebase-service-account.json`
- Dùng để: backend đọc và ghi Firestore
- Phải giữ kín: `có`
- Chỉ cần khi: `MONGODB_ENABLED=false`

### 1a. MongoDB URI
- Dạng: connection string
- Ví dụ: `mongodb://127.0.0.1:27017`
- Dùng để: backend kết nối MongoDB
- Phải giữ kín: `nên giữ kín nếu có user/password`

### 1b. MongoDB database name
- Dạng: tên database
- Ví dụ: `vngrocery`
- Dùng để: chọn database trong MongoDB
- Phải giữ kín: `không`

### 2. JWT secret
- Dạng: chuỗi random dài
- Dùng để: ký access token và refresh token
- Phải giữ kín: `có`

### 3. OpenAI API key
- Dạng: key của OpenAI
- Dùng để: AI score ảnh seller và buyer
- Phải giữ kín: `có`

### 4. Vault unseal keys
- Dạng: 5 key được tạo khi `vault operator init`
- Dùng để: mở khóa Vault sau khi khởi động lại
- Phải giữ kín: `rất quan trọng`
- Nên lưu ở đâu: password manager hoặc nơi an toàn ngoài máy

### 5. Vault root token
- Dạng: token sinh khi `vault operator init`
- Dùng để: backend truy cập Vault local/persistent
- Phải giữ kín: `rất quan trọng`
- Sau khi có token, điền vào `.env`:

```dotenv
VAULT_ENABLED=true
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=<vault-root-token>
VAULT_KV_MOUNT=secret
VAULT_KEYS_PATH_PREFIX=account-keys
```

Nếu chạy backend trực tiếp trên máy host:
```dotenv
VAULT_ADDR=http://127.0.0.1:8200
```

### 6. Besu deployer address
- Dạng: địa chỉ EVM
- Dùng để: gửi transaction deploy contract hoặc anchor hash
- Phải giữ kín: `không`, vì address là public

### 7. Besu private key
- Dạng: private key EVM
- Dùng để: backend tự ký raw transaction gửi lên Besu
- Phải giữ kín: `rất quan trọng`

Nếu dùng local signing:
```dotenv
BESU_ENABLED=true
BESU_RPC_URL=http://besu-rpc-proxy:8545
BESU_RPC_URLS=http://besu-rpc-proxy:8545,http://besu-rpc1:8545,http://besu-rpc2:8545
BESU_CHAIN_ID=1337
BESU_PRIVATE_KEY=<your-besu-private-key>
```

### 8. Besu contract address
- Dạng: địa chỉ contract sau khi deploy
- Dùng để: backend gọi `IntegrityRegistry`
- Phải giữ kín: `không`

Ví dụ:
```dotenv
BESU_CONTRACT_ADDRESS=0x1234...
```

## Những gì phải cất kỹ
- Firebase service account file
- JWT secret
- OpenAI API key
- Vault unseal keys
- Vault root token
- Besu private key

## Những gì có thể public
- `BESU_CONTRACT_ADDRESS`
- `BESU_FROM_ADDRESS`
- `IPFS_GATEWAY_URL`
