# 05. Deploy Contract

## Mục tiêu
Deploy `contracts/IntegrityRegistry.sol` lên Besu local để backend có thể anchor hash.

## Chạy script deploy
```bash
./scripts/deploy-integrity.sh
```

Script sẽ in ra:
- `contractAddress`
- `transactionHash`
- `deployer`

## Copy contract address vào `.env`
Ví dụ:

```dotenv
BESU_CONTRACT_ADDRESS=0x42699a7612a82f1d9c36148af9c77354759b210b
```

## Nếu backend dùng địa chỉ gửi tx thay vì private key
Điền:
```dotenv
BESU_FROM_ADDRESS=0xfe3b557e8fb62b89f4916b721be55ceb828dbd73
```

## Nếu backend dùng private key để ký local
Điền:
```dotenv
BESU_PRIVATE_KEY=<your-private-key>
```

Bạn chỉ cần một trong hai:
- `BESU_FROM_ADDRESS`
- hoặc `BESU_PRIVATE_KEY`

Nhưng cho môi trường nghiêm túc, nên dùng `BESU_PRIVATE_KEY`.

## Sau khi sửa `.env`
Khởi động lại stack:

```bash
make run-all
```
