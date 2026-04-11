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

## Tạo thư mục chứa Firebase credentials
```bash
mkdir -p secrets
```

Đặt file Firebase vào:

```text
secrets/firebase-service-account.json
```

## Những dòng cần sửa ngay trong `.env`
```dotenv
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_CREDENTIALS_FILE=./secrets/firebase-service-account.json
JWT_SECRET=replace-with-a-long-random-secret
OPENAI_API_KEY=your-openai-api-key
BOOTSTRAP_ADMIN_EMAILS=admin@example.com
```

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
