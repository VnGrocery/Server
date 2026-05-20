# Ke hoach tiep theo: hoan thien batch, freshness history va traceability

## 0. Quy tac thuc hien

- Lam theo tung luong nho, moi luong co code, test va commit rieng.
- Xong luong nao cap nhat trang thai vao file nay truoc khi commit.
- Khong stage/commit cac thay doi ngoai pham vi, dac biet `.gitignore` va `test.sh` neu dang la thay doi rieng cua nguoi dung.
- Neu test khong chay duoc, ghi ro ly do va test da chay.
- Uu tien sua cac lo hong logic backend truoc UI nang cao.

## 1. Muc tieu

Hoan thien cac phan con thieu sau khi da co nen tang batch:

- Seller tao cam ket phai chon batch that.
- Backend phai validate batch khi seller commit pledge.
- Buyer check phai validate batch khop token va pledge.
- Product detail phai xem duoc lich su do tuoi theo batch.
- Du lieu cu phai co default batch qua backfill.
- Them trace event/timeline nguon goc cho moi batch.

## 2. Van de hien tai

### 2.1 Seller UI chua chon batch

Android da co DTO/repository batch, backend da nhan `batchId`, nhung man hinh seller create pledge hien chi chon shop + product. Vi vay pledge moi co the van khong gan batch neu UI khong gui batch.

### 2.2 Seller commit backend chua validate batch

Backend hien luu `batchId` vao pledge, nhung chua kiem tra:

- batch co ton tai khong
- batch co thuoc dung shop/product khong
- batch co active khong
- batch co expired/recalled khong

### 2.3 Buyer check chua validate batch consistency

Buyer check da luu `batchId`, token da co claim `batchId`, pledge da co `batchId`, nhung service chua chan cac truong hop mismatch:

- request `batchId` khac token `batchId`
- pledge `batchId` khac token `batchId`
- pledge `batchId` khac request `batchId`

### 2.4 Product detail chua co freshness history

Product detail moi chi hien batch cards. Chua co:

- danh sach lan kiem tra do tuoi theo batch
- chart
- timeline freshness
- warning neu diem giam bat thuong

### 2.5 Chua co migration/backfill

Du lieu cu khong co batch nen can default batch cho moi product va gan lai pledge/report/check cu.

### 2.6 Chua co TraceEvent

Chua co entity va API cho timeline nguon goc: origin, slaughter, packaging, shipping, received, storage check, recall.

## 3. Thu tu uu tien

1. Backend validate seller commit batch.
2. Android seller create pledge chon batch.
3. Backend validate buyer check batch consistency.
4. Android buyer/result hien batch info ro hon.
5. Freshness history UI theo batch.
6. Backfill default batch.
7. TraceEvent backend.
8. TraceEvent Android UI.
9. Trust/proof UI polish.

## 4. Luong 1: Backend validate seller commit batch

Trang thai: `done`

### Backend tasks

- Inject `ProductBatchRepository` vao `seller.Service`.
- Cap nhat constructor `seller.NewService`.
- Cap nhat `cmd/server/main.go` de truyen `productBatchRepository`.
- Khi `CommitInput.BatchID` khong rong:
  - load batch theo `batchId`
  - batch phai thuoc `shopId`
  - batch phai thuoc `productId`
  - batch owner phai la seller
  - batch status phai la `active`
- Neu batch khong active:
  - `sold_out`, `expired`, `recalled`, `deleted` deu reject.
- Neu `productId` rong nhung `batchId` co ton tai:
  - co the derive `productId` tu batch hoac reject. Khuyen nghi: reject de API ro rang trong MVP.

### Tests

- Seller commit thanh cong voi active batch.
- Reject batch khong ton tai.
- Reject batch khac product.
- Reject batch khac shop.
- Reject batch expired/recalled.
- Full `go test ./...`.

### Da lam

- Inject `ProductBatchRepository` vao `seller.Service`.
- Cap nhat `seller.NewService` va `cmd/server/main.go` de truyen repository batch.
- Validate `batchId` khi seller commit:
  - bat buoc co `productId` neu co `batchId`
  - batch phai ton tai
  - batch phai thuoc dung shop/product
  - batch owner phai khop seller neu co owner
  - batch status phai la `active`
- Them test cho active batch, missing batch, product mismatch, shop mismatch, inactive batch va batch khong co product.

### Test da chay

- `go test ./internal/service/seller ./cmd/server`
- `go test ./...`

### Commit

Commit message de xuat:

```text
Validate seller pledge batches
```

## 5. Luong 2: Android seller create pledge chon batch

Trang thai: `done`

### Android tasks

- Inject `ProductBatchRepository` vao `SellerCreatePledgeViewModel`.
- Them state:
  - `batches`
  - `selectedBatchId`
- Khi chon product:
  - load batches active cua product.
  - auto select batch dau tien neu co.
- Trong `SellerCreatePledgeScreen`:
  - them dropdown `Lo hang`.
  - hien batch code, freshness, expiry.
  - disable nut chup/commit neu chua co batch.
- Khi commit:
  - gui `batchId` trong `SellerCommitRequest`.
- QR payload:
  - da co `batchId`, dam bao lay tu response hoac selected batch.

### Tests

- `./gradlew :app:compileDevDebugKotlin`
- Manual flow:
  - tao product
  - tao batch
  - tao pledge chon batch
  - QR co batchId

### Da lam

- Inject `ProductBatchRepository` vao `SellerCreatePledgeViewModel`.
- Them state `batches` va `selectedBatchId`.
- Load danh sach batch `active` khi load/chon product.
- Auto select batch active dau tien neu co.
- Them dropdown `Lo hang` trong `SellerCreatePledgeScreen`.
- Disable buoc chup/commit neu chua co batch active.
- Gui `batchId` trong `SellerCommitRequest` va giu `batchId` trong QR payload tu response.

### Test da chay

- `./gradlew :app:compileDevDebugKotlin`

### Commit

```text
Require batch selection for seller pledges
```

## 6. Luong 3: Backend validate buyer check batch consistency

Trang thai: `done`

### Backend tasks

- Trong buyer service:
  - lay `batchId` tu request.
  - lay `batchId` tu token claims.
  - lay `batchId` tu pledge neu co.
- Rule:
  - neu request va token deu co batchId nhung khac nhau -> reject.
  - neu pledge va token deu co batchId nhung khac nhau -> reject.
  - neu pledge va request deu co batchId nhung khac nhau -> reject.
  - neu chi token co batchId -> dung token batchId.
  - neu chi pledge co batchId -> dung pledge batchId.
  - neu khong co batchId -> cho fallback cho QR cu.
- Them helper `resolveBatchIDForCheck`.
- Ghi reason/log neu mismatch.

### Tests

- Buyer check pass khi request/token/pledge batch khop.
- Reject request batch khac token batch.
- Reject pledge batch khac token batch.
- Fallback QR cu khong batch van pass.
- Replay retry van tra result co batchId.
- Full `go test ./...`.

### Da lam

- Them helper `resolveBatchIDForCheck`.
- Validate mismatch giua request/token/pledge truoc khi cham diem AI va truoc khi persist buyer check.
- Rule fallback:
  - request va token khac nhau thi reject
  - pledge va token khac nhau thi reject
  - pledge va request khac nhau thi reject
  - uu tien token batchId, sau do pledge batchId, sau do request batchId
  - QR cu khong co batchId van pass
- Dam bao replay retry tra lai result co `batchId` tu check da luu.

### Test da chay

- `go test ./internal/service/buyer`
- `go test ./...`

### Commit

```text
Validate buyer check batch consistency
```

## 7. Luong 4: Android buyer/result hien batch info

Trang thai: `done`

### Android tasks

- `BuyerCheckResponse` da co `batchId`; them hien thi trong `BuyerCheckResultScreen`.
- Neu co batchId:
  - hien "Lo hang: <batchId>" hoac batchCode neu load duoc.
- Can nhac load batch detail sau buyer check:
  - shopId, productId, batchId -> `ProductBatchRepository.getBatch`.
- Hien:
  - batchCode
  - currentFreshness
  - expiry
  - status

### Tests

- `./gradlew :app:compileDevDebugKotlin`

### Da lam

- `BuyerCheckResultScreen` doc `batchId` tu `BuyerCheckResponse`.
- Neu co `batchId`, load batch detail bang `ProductBatchRepository.getBatch`.
- Hien card `Lo hang` gom:
  - batch code hoac batch id
  - current freshness
  - status
  - expiry neu co
- Neu khong load duoc chi tiet batch, van hien `batchId` va trang thai loi gon.

### Test da chay

- `./gradlew :app:compileDevDebugKotlin`

### Commit

```text
Show batch context in buyer check results
```

## 8. Luong 5: Freshness history UI theo batch

Trang thai: `done`

### Backend da co

- `GET /v1/shops/:shopId/products/:productId/freshness-reports?batchId=...`

### Android tasks

- Tao UI model:
  - `FreshnessHistoryPoint`
- Tao mapper tu `ProductFreshnessReportResponse`.
- Cap nhat `ProductDetailViewModel`:
  - selected batch
  - freshness reports cua selected batch
  - load reports khi selected batch thay doi
- Cap nhat `ProductDetailScreen`:
  - batch selector
  - list/timeline freshness reports
  - compact chart bang Compose Canvas hoac progress trend don gian
- Hien:
  - score
  - category
  - confidence
  - comment
  - createdAt
  - source tam thoi la reporter/admin neu chua co field source

### Tests

- `./gradlew :app:compileDevDebugKotlin`

### Da lam

- Them UI model `FreshnessHistoryPoint`.
- Them mapper tu `ProductFreshnessReportResponse`.
- Cap nhat `ProductDetailViewModel`:
  - `selectedBatchId`
  - `freshnessHistory`
  - load freshness reports theo batch dang chon
  - doi batch thi load lai history
- Cap nhat `ProductDetailScreen`:
  - batch card co selected state va click chon batch
  - section `Lich su do tuoi theo lo`
  - trend bars compact
  - timeline/list hien score, category, confidence, comment, reporter va createdAt

### Test da chay

- `./gradlew :app:compileDevDebugKotlin`

### Commit

```text
Show batch freshness history
```

## 9. Luong 6: Backfill default batch

Trang thai: `done`

### Backend tasks

- Tao `cmd/backfill-batches/main.go`.
- Flags:
  - `--dry-run`
  - `--start-after`
  - `--batch-size`
  - `--default-status`
- Logic:
  - list products.
  - neu product chua co batch thi tao default batch.
  - `batchCode = DEFAULT-<productId>`.
  - `currentFreshness = product.FreshnessScore` sau khi chuan hoa scale.
  - gan pledge cu vao default batch neu pledge.ProductID khop.
  - gan freshness report cu vao default batch.
  - gan buyer check cu vao default batch neu productId khop.
- Khong overwrite batchId neu da co.

### Tests

- Unit test logic backfill voi repo fake.
- Dry-run khong save.
- Full `go test ./...`.

### Da lam

- Tao service `internal/service/batchbackfill` de chay backfill idempotent va test bang fake repo.
- Tao command `cmd/backfill-batches/main.go`.
- Ho tro flags:
  - `--dry-run`
  - `--start-after`
  - `--batch-size`
  - `--default-status`
- Ho tro wiring theo config:
  - MongoDB khi `MONGODB_ENABLED=true`
  - Firestore khi MongoDB disabled
- Logic:
  - list products
  - neu product chua co batch thi tao `default-<productId>`
  - `batchCode = DEFAULT-<productId>`
  - normalize `FreshnessScore` ve percent cho `currentFreshness`
  - gan pledge/report/buyer check cu vao default batch neu cung product va dang rong `batchId`
  - khong overwrite record da co `batchId`

### Test da chay

- `go test ./internal/service/batchbackfill ./cmd/backfill-batches`
- `go test ./...`

### Commit

```text
Add default batch backfill
```

## 10. Luong 7: Chuan hoa thang diem do tuoi

Trang thai: `done`

### Van de

He thong dang co diem 0-10 cho AI/seller pledge, trong khi batch `currentFreshness` dang validate 0-100. Can chuan hoa truoc khi data lon.

### Quyet dinh de xuat

- AI score va pledge score tiep tuc dung 0-10.
- Batch current freshness dung 0-100 percentage.
- Khi tao batch tu product/AI score:
  - neu input <= 10, co the convert sang `score * 10` hoac reject tuy API.
- UI phai ghi ro:
  - `Score AI` cho 0-10.
  - `Độ tươi %` cho 0-100.

### Backend tasks

- Them helper normalize freshness percent.
- Validate ro field nao 0-10, field nao 0-100.
- Cap nhat docs/DTO comments neu can.

### Android tasks

- Doi label UI:
  - "Score AI" cho pledge/check.
  - "Độ tươi %" cho batch.

### Tests

- Unit tests normalize helper.
- `go test ./...`
- `./gradlew :app:compileDevDebugKotlin`

### Da lam

- Backend:
  - them `batch.NormalizeFreshnessPercent`
  - batch create/update chap nhan input 0-10 va normalize sang percent 0-100
  - batch create/update van reject input am hoac >100
  - backfill default batch dung chung helper normalize
  - them DTO comment cho `currentFreshness`
- Android:
  - product card/list dung label `Score AI` cho product freshness score 0-10
  - doi nguong mau score AI ve 8/6 thay vi 80/60
  - batch detail hien `currentFreshness%` ro la do tuoi percent
  - product detail ghi ro score cam ket thang 0-10

### Test da chay

- `go test ./internal/service/batch ./internal/service/batchbackfill ./cmd/backfill-batches`
- `go test ./...`
- `./gradlew :app:compileDevDebugKotlin`

### Commit

```text
Clarify freshness score scales
```

## 11. Luong 8: TraceEvent backend foundation

Trang thai: `done`

### Domain

Them `TraceEvent`:

```go
type TraceEvent struct {
    EventID      string
    BatchID      string
    ProductID    string
    ShopID       string
    ActorUserID  string
    Type         string
    Title        string
    Description  string
    LocationName string
    Latitude     float64
    Longitude    float64
    Temperature  *float64
    Humidity     *float64
    ImageCID     string
    ImageHash    string
    DataHash     string
    Status       string
    OccurredAt   time.Time
    CreatedAt    time.Time
}
```

### Repository

- `TraceEventRepository`
- Firestore implementation.
- Mongo implementation.
- Filter by `shopId`, `productId`, `batchId`, `type`.

### Service

- `traceability.Service`
- `CreateTraceEvent`
- `ListTraceEvents`
- Validate:
  - batch exists
  - owner can create event
  - public can list active events

### API

- Public:
  - `GET /v1/shops/:shopId/products/:productId/batches/:batchId/trace-events`
- Seller:
  - `POST /v1/shops/:shopId/products/:productId/batches/:batchId/trace-events`

### Tests

- service tests
- handler tests
- router test
- `go test ./...`

### Da lam

- Them domain `TraceEvent`.
- Them `TraceEventRepository` vao repository contracts.
- Them collection `trace_events`.
- Them repository:
  - Firestore `TraceEventRepository`
  - Mongo `TraceEventRepository`
- Them `traceability.Service`:
  - `CreateTraceEvent`
  - `ListTraceEvents`
  - validate batch/shop/product ton tai
  - validate seller owner khi create
  - public list chi lay active events
- Them DTO va handler:
  - `CreateTraceEventRequest`
  - `TraceEventResponse`
  - `TraceEventListResponse`
  - `TraceEventHandler`
- Them routes:
  - `GET /v1/shops/:shopId/products/:productId/batches/:batchId/trace-events`
  - `POST /v1/shops/:shopId/products/:productId/batches/:batchId/trace-events`
- Wire repository/service/handler vao `cmd/server/main.go`.
- Them service tests, handler tests va router route tests.

### Test da chay

- `go test ./internal/service/traceability ./internal/api/handler ./internal/api/router ./cmd/server`
- `go test ./...`

### Commit

```text
Add trace event backend foundation
```

## 12. Luong 9: Android trace timeline UI

Trang thai: `done`

### Android tasks

- DTO:
  - `TraceEventDTO.kt`
- API:
  - `getTraceEvents`
  - `createTraceEvent`
- Repository:
  - `TraceabilityRepository`
- UI model:
  - `TraceEvent`
- Product detail:
  - tab/section `Nguon goc`
  - timeline vertical
- Seller:
  - basic form add trace event for batch

### Tests

- `./gradlew :app:compileDevDebugKotlin`

### Da lam

- Them DTO:
  - `TraceEventResponse`
  - `TraceEventListResponse`
  - `CreateTraceEventRequest`
- Them API:
  - `getTraceEvents`
  - `createTraceEvent`
- Them `TraceabilityRepository`.
- Them UI model `TraceEvent` va mapper tu `TraceEventResponse`.
- Cap nhat `ProductDetailViewModel`:
  - load trace events theo selected batch
  - reload trace events khi doi batch
  - tao trace event va reload timeline
- Cap nhat `ProductDetailScreen`:
  - section `Nguon goc theo lo`
  - timeline vertical cho trace events
  - dialog seller co ban de them trace event cho batch dang chon

### Test da chay

- `./gradlew :app:compileDevDebugKotlin`

### Commit

```text
Show trace timeline for batches
```

## 13. Luong 10: Proof/trust UI polish

Trang thai: `todo`

### Android tasks

- Store detail:
  - bo card hard-code "Cam ket chat luong dat 8.5+"
  - hien latest pledge that tu API.
- Proof viewer:
  - load `getPledgeProof`
  - hien proof status, hash, tx, block.
- Trust breakdown:
  - hien pledge/review/buyerCheck/consistency/recency/coverage.

### Tests

- `./gradlew :app:compileDevDebugKotlin`

### Commit

```text
Add proof and trust breakdown UI
```

## 14. Definition of Done cho giai doan tiep theo

Hoan thanh khi:

- Seller khong the tao pledge cho batch sai/xoa/expired/recalled.
- Seller UI bat buoc chon batch active khi tao pledge.
- Buyer check bi reject neu batch trong request/token/pledge khong khop.
- Buyer result hien batch context.
- Product detail hien freshness history theo selected batch.
- Du lieu cu co default batch qua backfill.
- TraceEvent backend va UI timeline co MVP.
- Full backend test pass.
- Android compile pass.

## 15. Trang thai thuc hien

- Da hoan thanh Luong 1: Backend validate seller commit batch.
- Da hoan thanh Luong 2: Android seller create pledge chon batch.
- Da hoan thanh Luong 3: Backend validate buyer check batch consistency.
- Da hoan thanh Luong 4: Android buyer/result hien batch info.
- Da hoan thanh Luong 5: Freshness history UI theo batch.
- Da hoan thanh Luong 6: Backfill default batch.
- Da hoan thanh Luong 7: Chuan hoa thang diem do tuoi.
- Da hoan thanh Luong 8: TraceEvent backend foundation.
- Da hoan thanh Luong 9: Android trace timeline UI.
- Luong tiep theo: Luong 10 Proof/trust UI polish.
