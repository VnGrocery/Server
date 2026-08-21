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
- `MongoDB`: backend lưu trữ duy nhất
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
- lọc cửa hàng theo bán kính (`lat`/`lng`/`radiusKm`), trả kèm `distanceKm`
- lịch sử thay đổi sản phẩm + chuỗi giá 30 ngày, dựng từ event log đã ký

## Quick Start
1. Tạo `.env`
```bash
cp .env.example .env
```

2. Điền `JWT_SECRET` và `OPENAI_API_KEY`

MongoDB là backend duy nhất; thiếu nó Server dừng ngay lúc khởi động:
```dotenv
MONGODB_ENABLED=true
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=vngrocery
```

3. Chạy stack local
```bash
./scripts/vng up
```

API nằm ở cổng `5050` của máy host (cổng 5000 bị AirPlay Receiver của macOS
chiếm sẵn). Đổi bằng `API_PORT` trong `.env` nếu cần.

4. Nếu Vault chưa init/unseal, làm theo hướng dẫn script in ra

5. Deploy contract Besu
```bash
./scripts/deploy-integrity.sh
```

6. Cập nhật `.env` với `VAULT_TOKEN` và `BESU_CONTRACT_ADDRESS`, rồi chạy lại:
```bash
./scripts/vng up
```

7. Test flow:
```bash
./test.sh
IMAGE_PATH=/abs/path/to/image.jpg ./scripts/e2e-mobile-flow.sh
```

## Important Docs

Mobile cấu hình Server bằng `--dart-define=API_BASE_URL=<server-url>`; mặc định là `http://10.0.2.2:5050`. Các route tích hợp bổ sung gồm `GET /v1/me/shop`, `GET /v1/seller/shops/:shopId/products`, `GET /v1/shops/:shopId/products/:productId/history`, `/v1/vouchers/check` và `/v1/me/vouchers`. Contract đầy đủ có tại `/docs` và `/openapi.json` khi Server đang chạy.
- setup từ đầu: [docs/setup/00-start-here.md](docs/setup/00-start-here.md)
- triển khai một máy, từng bước: [docs/HUONG-DAN-TRIEN-KHAI.md](docs/HUONG-DAN-TRIEN-KHAI.md)
- vận hành: [docs/operations.md](docs/operations.md)
- mobile API handoff: [docs/mobile-api-playbook.md](docs/mobile-api-playbook.md)

## Useful Commands
```bash
make help
./scripts/vng --prod vault-init
./scripts/vng chain-up
make ipfs-up
./scripts/vng up
./scripts/vng seed
./scripts/vng seed-history
./scripts/vng health
make logs service=api
make clean
make clean-all
```

## Notes
- Nếu backend chạy trong Docker Compose, dùng tên service như `vault`, `ipfs`, `besu-validator1`
- Nếu backend chạy trên host, dùng `127.0.0.1`
- Nếu blockchain ở server khác, dùng private IP hoặc domain nội bộ của RPC endpoint
