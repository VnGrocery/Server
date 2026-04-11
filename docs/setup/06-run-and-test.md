# 06. Run And Test

## Kiểm tra container đang chạy
```bash
make ps
```

## Kiểm tra log
```bash
make logs service=api
make logs service=vault
make logs service=besu-validator1
```

## Test flow nhanh không cần ảnh
```bash
./test.sh
```

## Test flow đầy đủ với ảnh
```bash
IMAGE_PATH=/abs/path/to/image.jpg ./scripts/e2e-mobile-flow.sh
```

## Bạn nên thấy gì
- auth register/login thành công
- tạo shop thành công
- tạo product thành công
- seller commit thành công
- proof endpoint trả dữ liệu
- nếu có ảnh: buyer check và freshness report cũng thành công

## Nếu muốn tạo admin ngay khi đăng ký
Đặt email vào `.env`:

```dotenv
BOOTSTRAP_ADMIN_EMAILS=admin@example.com
```

Khi account mới tạo có email này, hệ thống sẽ tự gán `role=admin`.
