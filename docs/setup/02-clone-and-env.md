# 02. Clone And Env

## Clone mã nguồn
```bash
git clone <YOUR_REPO_URL> VNGrocery
cd VNGrocery/server
```

## Tạo file `.env`
```bash
cp .env.example .env
```

## Tạo thư mục chứa credentials
```bash
mkdir -p secrets
```

Nếu bạn dùng Firestore, đặt file Firebase vào:

```text
secrets/firebase-service-account.json
```

## Những dòng cần sửa ngay trong `.env`
```dotenv
MONGODB_ENABLED=true
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=vngrocery
JWT_SECRET=replace-with-a-long-random-secret
OPENAI_API_KEY=your-openai-api-key
BOOTSTRAP_ADMIN_EMAILS=admin@example.com
```

Nếu bạn muốn dùng Firestore thay vì MongoDB, đọc tiếp:

- [02a-mongodb-or-firestore.md](/home/dora/VNGrocery/server/docs/setup/02a-mongodb-or-firestore.md)

## Chưa cần sửa ngay lúc này
Các biến sau sẽ được điền ở bước sau:
- `VAULT_TOKEN`
- `BESU_CONTRACT_ADDRESS`
- `BESU_PRIVATE_KEY` hoặc `BESU_FROM_ADDRESS`

## Tạo JWT secret nhanh
Ví dụ:

```bash
openssl rand -hex 32
```

Copy kết quả vào:
```dotenv
JWT_SECRET=<gia-tri-vua-tao>
```
