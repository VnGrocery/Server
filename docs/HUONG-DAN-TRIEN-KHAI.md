# Hướng dẫn triển khai VnGrocery

Tài liệu này là nguồn chính thức để chạy hệ thống. Nếu bạn là người mới, chỉ cần
làm đúng 4 bước ở phần "Chạy nhanh" là xong.

Mọi thứ đi qua **một lệnh duy nhất**: `./scripts/vng`.

---

## 1. Cần chuẩn bị gì

| Thứ | Dùng để làm gì |
|---|---|
| Docker Desktop (đang chạy) | chạy toàn bộ service |
| `jq`, `curl` | script deploy contract cần |
| File `.env` | cấu hình (script tự tạo từ `.env.example` nếu chưa có) |

Kiểm tra nhanh:

```bash
docker ps        # không báo lỗi là Docker đang chạy
jq --version
```

Không cần cài Go, MongoDB, hay Besu trên máy — tất cả chạy trong Docker.

---

## 2. Chạy nhanh (4 bước)

Đứng ở thư mục `server/`:

```bash
./scripts/vng chain-up          # 1. bật blockchain
./scripts/vng contract-deploy   # 2. deploy contract, tự ghi vào .env
./scripts/vng up                # 3. bật toàn bộ hệ thống
./scripts/vng health            # 4. kiểm tra
```

Xong. Mở http://localhost:5000/health, thấy `{"status":"ok"}` là chạy được.

> **Vì sao phải deploy contract trước khi `up`?**
> Server cần địa chỉ contract để neo hash lên blockchain. Bước 2 tự động ghi
> `BESU_CONTRACT_ADDRESS`, `BESU_FROM_ADDRESS` và `BESU_PRIVATE_KEY` vào `.env`,
> bạn không phải copy tay.

### Trước khi dùng thật, sửa 2 giá trị trong `.env`

```bash
JWT_SECRET=...        # chuỗi ngẫu nhiên dài, dùng để ký token đăng nhập
OPENAI_API_KEY=...    # để tính năng chấm điểm ảnh AI hoạt động
```

Sinh `JWT_SECRET` ngẫu nhiên:

```bash
python3 -c "import secrets; print(secrets.token_urlsafe(48))"
```

---

## 3. Hệ thống gồm những gì

| Service | Cổng | Vai trò |
|---|---|---|
| `api` | 5000 | Backend Go |
| `mongo` | 27017 | Cơ sở dữ liệu chính |
| `vault` | 8200 | Lưu khoá tài khoản |
| `ipfs` | 5001 / 8080 | Lưu ảnh sản phẩm |
| `redis` | 6379 | Cache |
| `besu` | 8545 | Blockchain (1 node) neo hash |

---

## 3b. Tạo dữ liệu demo

Server mới dựng thì rỗng, app sẽ trông trống trải. Đổ dữ liệu mẫu:

```bash
./scripts/vng seed
```

Tạo 5 cửa hàng có thật ở TP.HCM (rau, thịt, hải sản, trái cây, nông sản) kèm
sản phẩm, cam kết độ tươi (được neo lên blockchain), đánh giá và voucher. Mật
khẩu mọi tài khoản: `Passw0rd!`

Một cửa hàng cố ý **không có cam kết nào** để thấy trạng thái "chưa đủ dữ liệu"
— khác với cửa hàng bị chấm điểm thấp.

> Kiểm chứng của người mua (buyer check) **không** được seed: nó cần AI vision,
> mà không có `OPENAI_API_KEY` thì server trả về "provider unavailable".

---

## 4. Các lệnh hay dùng

```bash
./scripts/vng up                # bật hệ thống
./scripts/vng down              # tắt
./scripts/vng restart           # bật lại
./scripts/vng status            # xem service nào đang chạy
./scripts/vng logs api          # xem log (đổi 'api' thành service khác)
./scripts/vng health            # kiểm tra API + blockchain
./scripts/vng e2e               # chạy thử toàn bộ luồng nghiệp vụ
./scripts/vng seed              # đổ dữ liệu demo
./scripts/vng reset             # xoá sạch dữ liệu, làm lại từ đầu
./scripts/vng help              # xem tất cả lệnh
```

Có thể dùng `make` cho ngắn: `make up`, `make down`, `make logs service=api`.

---

## 5. Các chế độ chạy

Thêm cờ vào bất kỳ lệnh nào:

| Cờ | Khi nào dùng |
|---|---|
| *(không có)* | Mặc định. Blockchain 1 node. Dùng cho phát triển và demo. |
| `--prod` | Chạy thật: thêm nginx, Vault lưu bền, tự khởi động lại, xoay log. |
| `--qbft` | Demo blockchain 4 validator. Chỉ dùng khi cần trình diễn. |
| `--no-chain` | Chạy không có blockchain (không neo hash). |

Ví dụ: `./scripts/vng up --prod`

### Vì sao mặc định chỉ 1 node blockchain?

QBFT chỉ chịu lỗi được `f` node khi có `n ≥ 3f+1` node. Cụm 4 validator vì thế
chịu được **đúng 1** node hỏng — hễ 2 node mất đồng bộ là chain **ngừng tạo
block hoàn toàn**. Đó chính là lý do bản cũ hay "chết" khó hiểu.

Một node thì không có ai để mà mất đồng bộ, nên lỗi đó biến mất. Chain vẫn tạo
block thật, contract vẫn neo hash thật, verify vẫn chạy thật. Cần trình diễn
nhiều validator thì mới bật `--qbft`.

---

## 6. Chạy ở chế độ production

```bash
./scripts/vng up --prod
```

Khác biệt so với mặc định:

- Có nginx đứng trước API (cổng 80)
- Vault chuyển sang lưu bền (dev mode mất hết dữ liệu khi restart)
- `restart: always` + tự xoay log

Vault bản bền phải khởi tạo **một lần duy nhất**:

```bash
./scripts/vng --prod vault-init
```

Làm theo hướng dẫn script in ra: `operator init` → lưu kỹ unseal keys và root
token → `operator unseal` 3 lần với 3 key khác nhau.

Chế độ `--prod` **bắt buộc** có `VAULT_TOKEN` trong `.env` (không nhận token dev).

> **Mất unseal keys là mất toàn bộ khoá tài khoản.** Lưu ở nơi an toàn.

---

## 7. Kiểm tra hệ thống chạy đúng

```bash
./scripts/vng e2e
```

Lệnh này chạy hết luồng thật: đăng ký → tạo shop → tạo sản phẩm → seller commit
→ neo hash lên blockchain → tạo báo cáo độ tươi → xem trust score.

Muốn xác nhận blockchain neo thành công, lấy `shopId` và `pledgeId` ở cuối kết
quả rồi gọi:

```bash
curl -s http://localhost:5000/v1/shops/<shopId>/pledges/<pledgeId>/proof | jq .integrity
```

Kết quả đúng phải có:

```json
{
  "chainAnchorStatus": "anchored",
  "integrityStatus": "anchored",
  "onChainMatch": true,
  "chainTxHash": "0x..."
}
```

Nếu còn `pending_anchor`, đợi khoảng 15 giây rồi gọi lại — worker neo hash chạy
theo chu kỳ vài giây một lần.

---

## 8. Gặp lỗi thì làm gì

### `bind: address already in use` ở cổng 5000

Trên macOS, cổng 5000 bị AirPlay Receiver chiếm. Đổi cổng trong `.env`:

```bash
API_PORT=5050
```

Rồi `./scripts/vng up` lại. API sẽ ở `http://localhost:5050`.

(Cách khác: tắt AirPlay Receiver trong System Settings → General → AirDrop & Handoff.)

### API restart liên tục, log ghi `JWT_SECRET must be changed`

`.env` còn để giá trị mẫu. Sinh chuỗi ngẫu nhiên và thay vào (xem mục 2).

### Pledge kẹt mãi ở `pending_anchor`

Thường do `BESU_PRIVATE_KEY` trong `.env` chưa đúng. Contract dùng `onlyOwner`
nên giao dịch **phải được ký bằng đúng khoá đã deploy contract**. Chạy lại:

```bash
./scripts/vng contract-deploy
./scripts/vng restart
```

### `IntegrityRegistry is not deployed yet`

Bạn chạy `up` trước khi deploy contract. Làm đúng thứ tự ở mục 2, hoặc chạy
`./scripts/vng up --no-chain` nếu tạm thời không cần blockchain.

### Muốn làm lại sạch từ đầu

```bash
./scripts/vng reset             # xoá container + toàn bộ dữ liệu (có hỏi xác nhận)
./scripts/vng chain-up
./scripts/vng contract-deploy
./scripts/vng up
```

### Xem log để tìm nguyên nhân

```bash
./scripts/vng logs api
./scripts/vng logs besu
./scripts/vng status
```

---

## 9. Những giá trị phải giữ kỹ

Mất là hỏng hệ thống, không khôi phục được:

- `JWT_SECRET` — mất thì toàn bộ người dùng bị đăng xuất
- Vault **unseal keys** và **root token** — mất thì không mở được Vault
- `BESU_PRIVATE_KEY` — khoá owner của contract, mất thì không neo hash được nữa
- `OPENAI_API_KEY`

`.env` **không được** commit lên git.

---

## 10. Cấu trúc file triển khai

```
docker-compose.yml            hệ thống nền (api, mongo, vault, ipfs, redis)
docker-compose.besu.yml       blockchain 1 node — mặc định
docker-compose.besu-qbft.yml  blockchain 4 validator — chỉ để demo
docker-compose.prod.yml       lớp phủ production

scripts/vng                   lệnh duy nhất cần nhớ
scripts/deploy-integrity.sh   deploy contract  (vng gọi)
scripts/check-blockchain.sh   kiểm tra chain   (vng gọi)
scripts/e2e-mobile-flow.sh    test đầu-cuối    (vng gọi)
```

Bản tiếng Anh ngắn gọn hơn: [`../DEPLOY.md`](../DEPLOY.md)
