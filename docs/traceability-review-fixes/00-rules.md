# Rules

## Quy tac chung

- Lam theo tung luong nho, moi luong co code, test va commit rieng.
- Truoc khi sua code, doc file plan cua luong do va cac test hien co lien quan.
- Khong sua ngoai pham vi luong dang lam neu khong bat buoc.
- Khong revert thay doi cua nguoi khac.
- Neu phat hien bug lien quan nhung ngoai pham vi, ghi vao file plan hoac tao file plan moi.
- Sau khi sua code, cap nhat file plan cua luong:
  - `Trang thai`
  - `Da lam`
  - `Test da chay`
  - `Commit`
- Neu test khong chay duoc, ghi ro lenh da chay va ly do fail/block.

## Code

- Uu tien pattern san co trong service/handler/repository.
- Validate nghiep vu o service layer, handler chi parse request va map response.
- Error can wrap vao error domain hien co de handler tra dung status code.
- Khong them abstraction neu chi dung mot noi va khong lam logic ro hon.
- Voi batch/product/shop, luon trim ID truoc khi so sanh.
- Voi public API, khong expose resource da `deleted`.

## Test

- Moi luong backend can co unit test service truoc.
- Neu thay doi handler/router contract, them/cap nhat handler/router test.
- Chay test nho truoc, sau do chay full:

```bash
go test ./internal/service/<module>
go test ./internal/api/handler
go test ./...
```

- Neu co script e2e phu hop, chay them sau khi unit test pass.

## Commit

- Mot commit cho mot luong.
- Commit message ngan, imperative, neu duoc theo dang:

```text
Validate freshness report batches
Require active batches for buyer checks
Sync batch freshness from checks
Filter trust signals by status
```

- Truoc commit:

```bash
go test ./...
git status --short
```

- Chi stage cac file thuoc luong dang lam va file plan da cap nhat.

## Trang thai hop le

- `todo`: Chua lam.
- `in_progress`: Dang lam.
- `blocked`: Bi chan, co ghi ly do.
- `done`: Da code, test, cap nhat plan va commit.
