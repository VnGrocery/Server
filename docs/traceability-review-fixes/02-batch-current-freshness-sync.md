# Luong 02: Dong bo current freshness cua batch

Trang thai: `done`

## Van de

`ProductBatch.CurrentFreshness` chi duoc set khi create/update batch. Buyer check va freshness report tao ra nhieu tin hieu moi nhung batch card van co the hien diem cu, lam UI va trust signal khong phan anh tinh trang hien tai.

## Muc tieu

- Batch detail/card hien freshness gan voi tin hieu moi nhat.
- Lich su freshness van duoc giu trong report/check.
- Co policy ro ve nguon du lieu nao duoc cap nhat batch:
  - seller/manual report
  - buyer check trusted
  - buyer check warning/high_risk
  - admin moderation

## Phuong an de xuat

MVP nen dung approach an toan:

- Khi seller/owner tao freshness report hop le cho batch, update `ProductBatch.CurrentFreshness` va `CurrentCategory`.
- Buyer check khong update truc tiep neu chua co moderation; chi luu history.
- Neu buyer check `high_risk`, co the tao warning/reason trong response va trust score, khong ghi de batch state.

Sau MVP co the them derived endpoint tinh current freshness tu report/check moi nhat.

## Pham vi code du kien

- `internal/service/product/service.go`
- `internal/service/product/service_test.go`
- Co the can `ProductBatchRepository` dependency tu luong 01.
- `internal/service/buyer/service.go` neu quyet dinh buyer check update batch.

## Backend tasks

- Sau khi tao freshness report active va co batch:
  - load batch
  - validate version/update policy
  - update `CurrentFreshness`, `CurrentCategory`, `UpdatedAt`, tang `Version`
- Dam bao khong lam mat cac field batch khac.
- Xem xet audit log cho batch update neu he thong can audit batch state.
- Dinh nghia ro score scale:
  - product report score hien dang 0-10
  - batch current freshness dang normalize 0-100
  - can convert nhat quan truoc khi save

## Tests can co

- Report score 8.2 update batch current freshness thanh 82 neu batch dung scale 0-100.
- Report category update `CurrentCategory`.
- Report bi reject khong update batch.
- Report khong batch khong update batch.
- Loi save batch thi report co rollback duoc khong? Neu khong co transaction, can document behavior.
- `go test ./internal/service/product`
- `go test ./...`

## Da lam

- Khi freshness report active co `batchId` duoc tao, backend sync lai batch da validate:
  - `CurrentFreshness = report.Score * 10`
  - `CurrentCategory = report.Category`
  - tang `Version`
  - `UpdatedAt = report.CreatedAt`
- Giu nguyen cac field batch khac khi save lai batch.
- Buyer check chua update batch truc tiep theo policy MVP trong file nay.
- Report khong co `batchId` khong lookup/sync batch, giu fallback legacy.
- Luu y hien tai chua co transaction cross-repo: report duoc save truoc, sau do moi sync batch. Neu save batch loi, service tra error nhung report co the da duoc luu.
- Them test cho sync score 0-10 sang percent 0-100, category, version, updatedAt, preserve field cu, no-batch fallback va loi batch sync.

## Test da chay

- `go test ./internal/service/product`
- `go test ./...`

## Commit

Da commit:

```text
Sync batch freshness from reports
```
