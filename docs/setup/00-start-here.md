# Start Here

Đây là file chính. Nếu bạn là người mới, chỉ cần đi theo đúng thứ tự dưới đây.

## Mục tiêu
Bạn sẽ làm được 5 việc:

1. Clone mã nguồn
2. Tạo file `.env`
3. Chuẩn bị các key và token cần thiết
4. Chạy toàn bộ stack local: API, Vault, IPFS, Besu
5. Gọi test flow để kiểm tra hệ thống sống

## Thứ tự đọc
1. [01-prepare-machine.md](/home/dora/VNGrocery/server/docs/setup/01-prepare-machine.md)
2. [02-clone-and-env.md](/home/dora/VNGrocery/server/docs/setup/02-clone-and-env.md)
3. [03-secrets-and-keys.md](/home/dora/VNGrocery/server/docs/setup/03-secrets-and-keys.md)
4. [04-start-services.md](/home/dora/VNGrocery/server/docs/setup/04-start-services.md)
5. [05-deploy-contract.md](/home/dora/VNGrocery/server/docs/setup/05-deploy-contract.md)
6. [06-run-and-test.md](/home/dora/VNGrocery/server/docs/setup/06-run-and-test.md)
7. [07-troubleshooting.md](/home/dora/VNGrocery/server/docs/setup/07-troubleshooting.md)

## Nếu bạn chỉ muốn chạy nhanh
Làm lần lượt:

1. copy `.env.example` thành `.env`
2. điền Firebase và OpenAI key
3. chạy `make run-all`
4. init và unseal Vault theo hướng dẫn script in ra
5. deploy contract Besu
6. cập nhật `BESU_CONTRACT_ADDRESS` và `VAULT_TOKEN` trong `.env`
7. chạy lại `make run-all`
8. test bằng `./scripts/e2e-mobile-flow.sh`

## Bạn cần giữ lại những thứ gì
- `Firebase service account json`
- `JWT_SECRET`
- `Vault unseal keys`
- `Vault root token`
- `BESU private key` nếu dùng signing local
- `OpenAI API key`

Không được làm mất các giá trị này. Nếu mất, hệ thống có thể không login được, không ký được, hoặc không anchor blockchain được.
