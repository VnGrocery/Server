# Luong 06: Chuan hoa shop onboarding va verification

Trang thai: `todo`

## Van de

Shop moi tao hien duoc set `active` ngay. Voi bai toan check uy tin shop, shop chua duoc review/verify nhung da public active co the gay hieu nham voi nguoi mua.

## Muc tieu

- Co trang thai onboarding ro rang cho shop moi.
- Public listing va trust badge phan biet shop active verified voi shop moi/chua du thong tin.
- Khong pha luong seller local/dev neu MVP van can tao shop nhanh.

## Phuong an de xuat

Lua chon A - chat hon:

- Shop moi default `pending_review`.
- Admin approve sang `active`.
- Public listing chi hien `active`.

Lua chon B - it pha hon:

- Shop moi van `active`, nhung them trust reason/grade the hien `unverified_new_shop`.
- UI khong hien verified badge neu chua co proof/review/check/pledge.

Khuyen nghi: Chon B neu dang demo MVP, chon A neu chuan bi production.

## Pham vi code du kien

- `internal/service/shop/service.go`
- `internal/service/shop/service_test.go`
- `internal/api/handler/shop_handler_test.go`
- Mobile/UI neu co hien badge verification.

## Backend tasks

- Chon policy A hoac B va ghi vao file nay truoc khi code.
- Neu A:
  - doi default status trong `Create`
  - dam bao seller van quan ly duoc shop cua minh khi pending
  - admin moderation approve active
- Neu B:
  - them reason trong trust summary khi shop moi it du lieu
  - dam bao proof/verified badge dua vao integrity/trust data, khong dua moi status active

## Tests can co

- Shop moi co status dung policy.
- Public list chi hien shop hop le.
- Owner van get/update shop cua minh neu pending.
- Trust summary cho shop moi co reason ro.
- `go test ./internal/service/shop`
- `go test ./internal/api/handler`
- `go test ./...`

## Da lam

- Chua lam.

## Test da chay

- Chua chay.

## Commit

De xuat:

```text
Clarify shop verification status
```
