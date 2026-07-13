# VNGrocery Server

Backend cho ứng dụng trust/freshness review giữa người bán và người mua.

Người bán chụp ảnh hàng đang bán, AI chấm điểm chất lượng, rồi tạo cam kết.  
Người mua chụp ảnh tại quầy để kiểm tra chất lượng hiện tại và so sánh với cam kết đó.  
Hệ thống lưu audit trail, proof, image CID, và integrity anchor để tăng tính minh bạch.

## Core Idea
- `Seller`: chụp ảnh, nhận AI score, xác nhận cam kết chất lượng
- `Buyer`: chụp ảnh, kiểm tra độ tươi/chất lượng hiện tại
- `System`: so sánh kết quả hiện tại với cam kết, lưu proof và history

## Stack
- `Go`: API, worker, business logic
- `MongoDB`: backend mặc định
- `Firestore`: backend tùy chọn khi tắt MongoDB
- `Vault`: lưu private key account
- `Besu QBFT`: anchor integrity hash
- `Kubo IPFS`: lưu image CID
- `OpenAI Vision`: chấm điểm ảnh

## Main Features
- auth email/password + Google
- shop, product, review, freshness report
- seller score + seller commit
- buyer check + proof view
- signed event log + verify API
- trust score cho shop/seller
- Besu integrity anchoring
- asynchronous Besu anchoring with RPC failover and retry backoff
- IPFS image CID flow

## Quick Start
1. Tạo `.env`
```bash
cp .env.example .env
```

2. Điền `JWT_SECRET`, `OPENAI_API_KEY`, và chọn storage backend

MongoDB là mặc định:
```dotenv
MONGODB_ENABLED=true
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=vngrocery
FIREBASE_ENABLED=false
```

Chỉ dùng Firestore khi:
```dotenv
MONGODB_ENABLED=false
FIREBASE_ENABLED=true
FIREBASE_PROJECT_ID=your-project
FIREBASE_CREDENTIALS_FILE=./secrets/firebase-service-account.json
```

3. Chạy stack local
```bash
make run-all
```

4. Nếu Vault chưa init/unseal, làm theo hướng dẫn script in ra

5. Deploy contract Besu
```bash
./scripts/deploy-integrity.sh
```

6. Cập nhật `.env` với `VAULT_TOKEN` và `BESU_CONTRACT_ADDRESS`, rồi chạy lại:
```bash
make run-all
```

7. Test flow:
```bash
./test.sh
IMAGE_PATH=/abs/path/to/image.jpg ./scripts/e2e-mobile-flow.sh
```

## Important Docs

Mobile cấu hình Server bằng `--dart-define=API_BASE_URL=<server-url>`. Các route tích hợp bổ sung gồm `GET /v1/me/shop`, `GET /v1/seller/shops/:shopId/products`, `/v1/vouchers/check` và `/v1/me/vouchers`. Contract đầy đủ có tại `/docs` và `/openapi.json` khi Server đang chạy.
- setup từ đầu: [docs/setup/00-start-here.md](/home/dora/VNGrocery/server/docs/setup/00-start-here.md)
- setup MongoDB: [docs/setup/02a-mongodb-or-firestore.md](/home/dora/VNGrocery/server/docs/setup/02a-mongodb-or-firestore.md)
- vận hành: [docs/operations.md](/home/dora/VNGrocery/server/docs/operations.md)
- mobile API handoff: [docs/mobile-api-playbook.md](/home/dora/VNGrocery/server/docs/mobile-api-playbook.md)
- mobile design handoff: [tmp/mobile-design-handoff/00-start-here.md](/home/dora/VNGrocery/server/tmp/mobile-design-handoff/00-start-here.md)

## Useful Commands
```bash
make help
make vault-up
make besu-up
make ipfs-up
make run-all
make logs service=api
make clean
make clean-all
```

## Notes
- Nếu backend chạy trong Docker Compose, dùng tên service như `vault`, `ipfs`, `besu-validator1`
- Nếu backend chạy trên host, dùng `127.0.0.1`
- Nếu blockchain ở server khác, dùng private IP hoặc domain nội bộ của RPC endpoint
