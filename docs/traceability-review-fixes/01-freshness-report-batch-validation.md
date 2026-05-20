# Luong 01: Validate batch cho freshness report

Trang thai: `done`

## Van de

`CreateFreshnessReport` hien chi validate shop/product public, sau do luu thang `batchId` tu request. Neu client gui batch khong ton tai, batch thuoc product khac, shop khac, hoac batch da deleted/recalled/expired, freshness history theo lo se sai.

## Muc tieu

- Neu request co `batchId`, backend phai validate batch ton tai va thuoc dung `shopId/productId`.
- Batch khong duoc `deleted`.
- Can quyet dinh policy cho `expired/recalled/sold_out`:
  - Khuyen nghi: cho phep tao report nhung gan reason/status rieng neu la buyer report lich su.
  - MVP an toan hon: chi cho `active`, reject cac status con lai.
- Response va list freshness history khong tra ve report gan batch sai.

## Pham vi code du kien

- `internal/service/product/service.go`
- `internal/service/product/service_test.go`
- Co the can inject `ProductBatchRepository` vao product service.
- `cmd/server/main.go` neu constructor thay doi.
- Handler test neu response/error mapping thay doi.

## Backend tasks

- Them dependency `ProductBatchRepository` vao `product.Service`.
- Khi `FreshnessReportInput.BatchID` khong rong:
  - load batch theo `batchId`
  - batch phai thuoc dung `shopId`
  - batch phai thuoc dung `productId`
  - batch status phai hop le theo policy da chon
- Neu `batchId` rong:
  - giu fallback cho data/flow cu neu can
  - khuyen nghi chi cho phep khi product chua co batch active, hoac ghi reason ro trong plan neu van cho phep
- Cap nhat test cho Mongo/Firestore neu repository behavior bi anh huong.

## Tests can co

- Tao report thanh cong voi active batch dung shop/product.
- Reject batch khong ton tai.
- Reject batch khac product.
- Reject batch khac shop.
- Reject batch deleted/recalled/expired theo policy.
- Fallback khong batch neu con support.
- `go test ./internal/service/product`
- `go test ./...`

## Da lam

- Them dependency `ProductBatchRepository` cho `product.Service` qua setter.
- Wire `productBatchRepository` vao product service trong `cmd/server/main.go`.
- Validate `batchId` khi tao freshness report:
  - batch phai ton tai
  - batch phai thuoc dung shop
  - batch phai thuoc dung product
  - batch phai co status `active`
- Giu fallback legacy: freshness report khong co `batchId` van duoc tao.
- Them test cho active batch, missing batch, sai product, sai shop, inactive batch va legacy report khong batch.

## Test da chay

- `go test ./internal/service/product`
- `go test ./...`

## Commit

Da commit:

```text
Validate freshness report batches
```
