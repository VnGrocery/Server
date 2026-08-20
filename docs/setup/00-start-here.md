# Start Here

> **Dùng hướng dẫn này:** [HUONG-DAN-TRIEN-KHAI.md](../HUONG-DAN-TRIEN-KHAI.md)
>
> Đó là tài liệu triển khai chính thức và đã được kiểm chứng bằng cách chạy thật.
> Các file `01-` đến `07-` trong thư mục này là bản cũ, còn sót nội dung không
> còn đúng (ví dụ hướng dẫn chọn Firestore — backend đó đã bị gỡ khỏi dự án,
> MongoDB hiện là backend duy nhất).

## Chạy nhanh

Đứng ở thư mục `server/`:

```bash
./scripts/vng chain-up          # 1. bật blockchain
./scripts/vng contract-deploy   # 2. deploy contract, tự ghi vào .env
./scripts/vng up                # 3. bật toàn bộ hệ thống
./scripts/vng health            # 4. kiểm tra
```

Chi tiết từng bước, các chế độ chạy (`--prod`, `--qbft`), và cách xử lý lỗi
thường gặp: xem [HUONG-DAN-TRIEN-KHAI.md](../HUONG-DAN-TRIEN-KHAI.md).

## Những giá trị phải giữ kỹ

- `JWT_SECRET`
- Vault unseal keys và root token
- `BESU_PRIVATE_KEY` (khoá owner của contract)
- `OPENAI_API_KEY`

Mất các giá trị này thì không đăng nhập được, không ký được, hoặc không neo
blockchain được nữa.
