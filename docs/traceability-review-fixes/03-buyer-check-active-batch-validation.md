# Luong 03: Buyer check validate batch con hop le

Trang thai: `done`

## Van de

Buyer check hien da validate batch consistency giua request, token va pledge. Tuy nhien service chua load batch de kiem tra batch con ton tai va con hop le tai thoi diem scan. Neu batch da `recalled`, `expired`, `deleted` sau khi QR duoc tao, buyer check van co the tiep tuc cham AI va tra trusted.

## Muc tieu

- Neu buyer check resolve duoc `batchId`, backend phai validate batch.
- Batch phai thuoc dung shop/product tu pledge hoac token claims.
- Batch inactive phai bi reject hoac tra verdict canh bao theo policy.
- Khong break QR cu khong co batch neu van can fallback.

## Policy de xuat

- `deleted`: reject.
- `recalled`: reject hoac verdict `high_risk` voi reason `batch_recalled`. Khuyen nghi reject neu QR khong nen dung nua.
- `expired`: verdict `high_risk` voi reason `batch_expired` neu van muon nguoi mua biet ly do.
- `sold_out`: reject neu QR tai quay khong con hop le.
- `active`: tiep tuc check binh thuong.

## Pham vi code du kien

- `internal/service/buyer/service.go`
- `internal/service/buyer/service_test.go`
- Inject `ProductBatchRepository` vao buyer service.
- `cmd/server/main.go` neu constructor thay doi.

## Backend tasks

- Them dependency batch repository.
- Sau `resolveBatchIDForCheck`, neu co batch:
  - load batch
  - validate batch shop/product khop pledge/token
  - validate status theo policy
- Dam bao token claims `ShopID/ProductID` duoc dung khi khong co pledge.
- Them reason ro rang cho batch inactive neu tra result thay vi reject.
- Replay retry phai tra lai result da luu, khong cham lai AI.

## Tests can co

- Pass voi active batch.
- Reject/flag recalled batch.
- Reject/flag expired batch.
- Reject deleted batch.
- Reject batch khac shop/product.
- QR cu khong batch van pass neu policy con support.
- Replay retry van tra result cu.
- `go test ./internal/service/buyer`
- `go test ./...`

## Da lam

- Them dependency `ProductBatchRepository` cho `buyer.Service` qua setter.
- Wire `productBatchRepository` vao buyer check service trong `cmd/server/main.go`.
- Sau khi resolve `batchId`, buyer check se load batch va validate:
  - batch phai ton tai
  - batch phai thuoc dung shop theo pledge/token
  - batch phai thuoc dung product theo pledge/token
  - batch status phai la `active`
- Giu fallback legacy: QR/check khong co `batchId` khong bi lookup batch.
- Replay retry van tra result cu truoc khi validate/cham lai.
- Them test cho active batch, inactive batch, sai shop, sai product.

## Test da chay

- `go test ./internal/service/buyer`
- `go test ./...`

## Commit

Da commit:

```text
Require active batches for buyer checks
```
