# Traceability Review Fixes

Thu muc nay gom cac luong can sua sau dot review tong the backend truy xuat nguon goc, do tuoi san pham va uy tin shop.

## Cach dung

- Doc `00-rules.md` truoc khi bat dau.
- Moi luong sua nam trong mot file rieng.
- Lam xong luong nao thi cap nhat `Trang thai`, `Da lam`, `Test da chay` va `Commit`.
- Uu tien sua backend logic truoc UI.

## Danh sach luong

1. `01-freshness-report-batch-validation.md`
2. `02-batch-current-freshness-sync.md`
3. `03-buyer-check-active-batch-validation.md`
4. `04-seller-commit-product-status-validation.md`
5. `05-trust-score-status-filtering.md`
6. `06-shop-onboarding-verification.md`
7. `07-trace-event-taxonomy-validation.md`

## Thu tu uu tien de xuat

1. Validate `batchId` cho freshness report.
2. Buyer check phai validate batch con hop le tai thoi diem scan.
3. Dong bo/derive current freshness cua batch.
4. Loc status khi tinh trust score.
5. Siết seller commit theo trang thai product.
6. Chuan hoa shop onboarding/verification.
7. Enum hoa trace event va validate timeline.
