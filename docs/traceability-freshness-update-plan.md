# Ke hoach cap nhat VNMeat: lo hang, lich su do tuoi va truy xuat nguon goc

## 0. Quy tac thuc hien

- Trien khai theo tung luong nho, moi luong phai co code, test lien quan va commit rieng.
- Sau khi xong moi luong, cap nhat trang thai trong file ke hoach nay truoc khi commit.
- Khong gop nhieu luong lon vao mot commit neu co the tach rieng theo backend, Android UI, migration hoac docs.
- Khong revert cac thay doi co san cua nguoi dung trong worktree.
- Neu mot luong khong test duoc day du, ghi ro ly do va test da chay trong commit/final summary.

## 1. Muc tieu

Cap nhat ung dung VNMeat tu mo hinh hien tai:

- Product chi luu thong tin san pham chung.
- Pledge/cam ket gan voi product va bundle.
- Freshness report gan voi product.
- Buyer check so sanh anh thuc te voi cam ket.
- Shop trust score da co nhung UI hien thi con don gian.

Thanh mo hinh day du hon cho ung dung truy xuat nguon goc va do uy tin:

- Moi san pham co nhieu lo hang.
- Moi lo hang co lich su do tuoi rieng.
- Moi lo hang co timeline truy xuat nguon goc.
- Cam ket, buyer check, freshness report deu gan duoc voi lo hang cu the.
- Nguoi mua xem duoc ly do tin cay, proof va canh bao bat thuong.
- Seller quan ly lo hang, dieu kien bao quan va QR theo tung lo.

## 2. Pham vi cap nhat

### Backend

- Them domain model cho lo hang.
- Them domain model cho su kien truy xuat.
- Mo rong pledge, buyer check, freshness report de ho tro batchId.
- Them repository cho batch va trace event.
- Them service xu ly batch, trace timeline va freshness history.
- Them REST API cho batch, timeline, freshness history.
- Cap nhat OpenAPI docs.
- Cap nhat audit event va integrity hash cho du lieu batch/trace quan trong.

### Android UI

- Them tab "Lo hang & do tuoi" trong man hinh chi tiet san pham.
- Them man hinh/section timeline truy xuat nguon goc.
- Them UI xem proof that su.
- Them trust score breakdown trong chi tiet cua hang.
- Them flow seller tao va quan ly lo hang.
- Cap nhat flow tao cam ket de chon lo hang.
- Cap nhat buyer check result de hien thi so sanh chi tiet.

### Du lieu va van hanh

- Them migration/backfill cho du lieu cu.
- Them seed data cho batch va trace event.
- Cap nhat indexes Firestore/MongoDB.
- Cap nhat test backend va Android.

## 3. Thiet ke du lieu moi

### 3.1 ProductBatch

Tao entity `ProductBatch` de dai dien cho tung lo hang cua mot san pham.

Truong de xuat:

```go
type ProductBatch struct {
    BatchID           string
    ProductID         string
    ShopID            string
    OwnerUserID       string
    BatchCode         string
    OriginName        string
    OriginAddress     string
    SupplierName      string
    SlaughteredAt     *time.Time
    PackedAt          *time.Time
    ReceivedAt        *time.Time
    ExpiresAt         *time.Time
    Quantity          float64
    QuantityUnit      string
    StorageTempMin    float64
    StorageTempMax    float64
    CurrentFreshness  float64
    CurrentCategory   string
    Status            string
    Version           int
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

Trang thai de xuat:

- `active`
- `sold_out`
- `expired`
- `recalled`
- `deleted`

### 3.2 TraceEvent

Tao entity `TraceEvent` de luu timeline truy xuat nguon goc.

Truong de xuat:

```go
type TraceEvent struct {
    EventID       string
    BatchID       string
    ProductID     string
    ShopID        string
    ActorUserID   string
    Type          string
    Title         string
    Description   string
    LocationName  string
    Latitude      float64
    Longitude     float64
    Temperature   *float64
    Humidity      *float64
    ImageCID      string
    ImageHash     string
    DataHash      string
    Status        string
    OccurredAt    time.Time
    CreatedAt     time.Time
}
```

Loai su kien de xuat:

- `origin`
- `slaughter`
- `packaging`
- `shipping_started`
- `shipping_checkpoint`
- `received_at_shop`
- `storage_check`
- `freshness_check`
- `quality_warning`
- `recall`

### 3.3 Mo rong entity hien co

Them `BatchID` vao:

- `Pledge`
- `BuyerCheck`
- `ProductFreshnessReport`
- DTO request/response tuong ung tren backend va Android.

Mo rong `ProductFreshnessReport`:

- `BatchID`
- `ObservedAt`
- `Temperature`
- `Humidity`
- `Source`: `seller`, `buyer`, `admin`, `system`
- `WarningLevel`: `none`, `watch`, `warning`, `critical`

Mo rong `SellerCommitRequest`:

- `batchId`
- `traceEventId` optional

Mo rong `BuyerCheckResponse`:

- `batchId`
- `batchCode`
- `pledgedScore`
- `actualScore`
- `scoreDelta`
- `reasons`
- `traceSummary`

## 4. Backend tasks

### 4.1 Domain

- Tao file `internal/domain/product_batch.go`.
- Tao file `internal/domain/trace_event.go`.
- Them `BatchID` vao `internal/domain/pledge.go`.
- Them `BatchID` vao `internal/domain/buyer_check.go`.
- Them `BatchID`, `ObservedAt`, `Temperature`, `Humidity`, `Source`, `WarningLevel` vao `internal/domain/product_freshness_report.go`.

### 4.2 Repository contracts

- Them `ProductBatchRepository` vao `internal/repository/contracts.go`.
- Them `TraceEventRepository` vao `internal/repository/contracts.go`.
- Them filter:

```go
type ProductBatchListFilter struct {
    ShopID    string
    ProductID string
    Status    string
}

type TraceEventListFilter struct {
    ShopID    string
    ProductID string
    BatchID   string
    Type      string
}
```

- Mo rong `ProductFreshnessReportListFilter` them `BatchID`.
- Mo rong `BuyerCheckListFilter` them `BatchID`.

### 4.3 Repository Firestore

- Tao `internal/repository/firestore/product_batch_repository.go`.
- Tao `internal/repository/firestore/trace_event_repository.go`.
- Cap nhat `collections.go` voi collections moi:
  - `productBatches`
  - `traceEvents`
- Cap nhat query freshness reports theo `batchId`.
- Cap nhat query buyer checks theo `batchId`.
- Them index Firestore:
  - `productBatches`: `shopId`, `productId`, `status`, `updatedAt`
  - `traceEvents`: `batchId`, `occurredAt`
  - `productFreshnessReports`: `batchId`, `createdAt`
  - `buyerChecks`: `batchId`, `createdAt`

### 4.4 Repository MongoDB

- Tao `internal/repository/mongo/product_batch_repository.go`.
- Tao `internal/repository/mongo/trace_event_repository.go`.
- Tao indexes MongoDB:
  - `product_batches.shopId`
  - `product_batches.productId`
  - `product_batches.status`
  - `trace_events.batchId`
  - `trace_events.occurredAt`
  - `product_freshness_reports.batchId`
  - `buyer_checks.batchId`

### 4.5 Service layer

Tao service moi `internal/service/batch`.

Chuc nang:

- `CreateBatch`
- `UpdateBatch`
- `DeleteBatch`
- `ListBatches`
- `GetBatch`
- `GetBatchFreshnessHistory`
- `GetBatchTraceTimeline`
- `AddTraceEvent`
- `AddStorageCheck`
- `MarkRecalled`
- `MarkSoldOut`

Quy tac nghiep vu:

- Chi owner cua shop moi tao/sua/xoa batch.
- Batch phai thuoc product trong cung shop.
- Batch active khong duoc co `expiresAt` trong qua khu neu tao moi.
- Neu freshness score moi thap hon score truoc qua nguong, tao `quality_warning`.
- Neu batch bi recall, buyer UI phai hien thi canh bao ro.

### 4.6 Product service

- Cap nhat `ProductService` de co the tra ve latest active batch summary cho product detail.
- Cap nhat freshness report creation:
  - validate `batchId`
  - update `ProductBatch.CurrentFreshness`
  - tao trace event `freshness_check`

### 4.7 Seller service

- Cap nhat `seller/commit`:
  - nhan `batchId`
  - validate batch active
  - pledge gan voi batch
  - bundle token payload chua `batchId`

### 4.8 Buyer service

- Cap nhat `buyer/check`:
  - doc `batchId` tu bundle token hoac request
  - validate pledge dung batch
  - luu buyer check voi `batchId`
  - tao freshness report hoac trace event tu buyer check neu ket qua hop le
  - neu batch expired/recalled thi verdict phai canh bao manh

### 4.9 Bundle token

- Them `batchId` vao token claims.
- Cap nhat verifier de doi chieu `batchId`.
- Cap nhat QR payload Android `VNMeatQrPayload`.

### 4.10 API routes

Them route public:

- `GET /v1/shops/:shopId/products/:productId/batches`
- `GET /v1/shops/:shopId/products/:productId/batches/:batchId`
- `GET /v1/shops/:shopId/products/:productId/batches/:batchId/freshness-history`
- `GET /v1/shops/:shopId/products/:productId/batches/:batchId/trace-events`

Them route seller authenticated:

- `POST /v1/shops/:shopId/products/:productId/batches`
- `PUT /v1/shops/:shopId/products/:productId/batches/:batchId`
- `DELETE /v1/shops/:shopId/products/:productId/batches/:batchId`
- `POST /v1/shops/:shopId/products/:productId/batches/:batchId/trace-events`
- `POST /v1/shops/:shopId/products/:productId/batches/:batchId/storage-checks`
- `POST /v1/shops/:shopId/products/:productId/batches/:batchId/recall`

Them route admin:

- `GET /v1/admin/product-batches`
- `PATCH /v1/admin/product-batches/:batchId/moderation`
- `GET /v1/admin/trace-events`

### 4.11 DTO backend

- Tao `internal/api/dto/product_batch.go`.
- Tao `internal/api/dto/trace_event.go`.
- Mo rong `product.go`, `seller_commit.go`, `buyer_check.go`.
- Cap nhat docs handler OpenAPI.

### 4.12 Audit va integrity

- Log audit event cho:
  - `batch.created`
  - `batch.updated`
  - `batch.recalled`
  - `trace_event.created`
  - `freshness_report.created`
- Dua `batchId` vao resource metadata.
- Can nhac neo hash cho trace event quan trong:
  - origin
  - slaughter
  - packaging
  - recall

## 5. Android tasks

### 5.1 DTO Android

Them file:

- `data/api/dto/ProductBatchDTO.kt`
- `data/api/dto/TraceEventDTO.kt`

Mo rong:

- `ProductDTO.kt`
- `SellerDTO.kt`
- `BuyerDTO.kt`
- `VNMeatQrPayload.kt`

### 5.2 API service Android

Cap nhat `VNMeatApiService.kt` them:

- `getProductBatches`
- `getProductBatch`
- `getBatchFreshnessHistory`
- `getBatchTraceEvents`
- `createProductBatch`
- `updateProductBatch`
- `createTraceEvent`
- `createStorageCheck`
- `recallBatch`

### 5.3 Repository Android

Them:

- `ProductBatchRepository`
- `TraceabilityRepository`

Cap nhat:

- `ProductRepository` de load batch summary/freshness history.
- `SellerRepository` de commit pledge theo batch.
- `BuyerRepository` de buyer check voi batch.

### 5.4 UI model

Them model:

- `ProductBatch`
- `TraceEvent`
- `FreshnessHistoryPoint`
- `BatchFreshnessSummary`

### 5.5 ProductDetailScreen

Cap nhat man hinh chi tiet san pham:

- Them tab hoac section `Lo hang & do tuoi`.
- Hien thi danh sach lo hang active.
- Moi batch card hien thi:
  - ma lo
  - han dung
  - diem tuoi hien tai
  - trang thai
  - canh bao neu expired/recalled/warning
- Khi chon batch:
  - hien thi freshness history
  - hien thi trace timeline
  - hien thi pledge/proof gan voi batch
  - nut `Kiem tra lo nay`

### 5.6 Freshness chart UI

Them component:

- `FreshnessHistoryChart`
- `FreshnessTimeline`
- `FreshnessStatusBadge`

MVP co the dung Compose Canvas don gian truoc, chua can thu vien chart.

Thong tin can hien thi:

- score theo thoi gian
- category
- source: seller/buyer/admin/system
- warning level
- comment
- image/proof neu co

### 5.7 Trace timeline UI

Them component:

- `TraceTimelineSection`
- `TraceEventItem`

Hien thi:

- nguon goc
- dong goi
- van chuyen
- nhap cua hang
- kiem tra bao quan
- kiem tra do tuoi
- recall/canh bao neu co

### 5.8 StoreDetailScreen

Cap nhat:

- Bo card hard-code "Cam ket chat luong dat 8.5+".
- Load pledge moi nhat tu API.
- Nut `Xem Proof` mo proof viewer.
- Them section `Vi sao cua hang nay dang tin?`
  - pledge score
  - review score
  - buyer check score
  - consistency score
  - recency score
  - coverage score
  - reasons

### 5.9 Proof viewer

Them bottom sheet hoac screen:

- `PledgeProofScreen`
- `PledgeProofBottomSheet`

Hien thi:

- `proofStatus`
- `proofHeadline`
- `proofSummary`
- `chainTxHash`
- `chainBlockNumber`
- `dataHash`
- `onChainMatch`
- `lastCheckedAt`
- recommended actions

### 5.10 Seller create product

Cap nhat form:

- Upload anh that su thay placeholder.
- Them thong tin nguon goc co ban:
  - nha cung cap
  - noi xuat xu
  - chung nhan
  - dieu kien bao quan khuyen nghi

### 5.11 Seller batch management

Them man hinh moi:

- `SellerBatchListScreen`
- `SellerCreateBatchScreen`
- `SellerBatchDetailScreen`

Luon uu tien workflow:

1. Tao product.
2. Tao batch cho product.
3. Them trace events dau vao.
4. Chup anh AI va tao pledge cho batch.
5. In QR theo batch.

### 5.12 Seller create pledge

Cap nhat:

- Chon shop.
- Chon product.
- Chon batch active.
- Chup anh.
- AI score.
- Seller confirm.
- Commit pledge voi `batchId`.
- QR payload co `batchId`.

### 5.13 Buyer check

Cap nhat flow:

- QR scan doc `shopId`, `productId`, `batchId`, `pledgeId`, `bundleId`.
- Product detail hien thi dung batch dang quet.
- Buyer check gui/verify batch.
- Result screen hien thi:
  - batch code
  - seller pledged score
  - actual score
  - delta
  - reasons
  - freshness history gan nhat
  - canh bao expired/recalled

## 6. UI/UX de xuat

### Product detail moi

Cau truc:

```text
Chi tiet san pham
- Tong quan
- Lo hang & do tuoi
- Nguon goc
- Proof
- Cua hang
```

Trong `Lo hang & do tuoi`:

```text
[Dropdown/List lo hang]

Ma lo: BATCH-2026-001
Do tuoi hien tai: 86
Trang thai: Tot
Han dung: 24/05/2026

[Bieu do lich su do tuoi]

[Timeline lan kiem tra]
```

Trong `Nguon goc`:

```text
Nguon hang
Dong goi
Van chuyen
Nhap cua hang
Bao quan
Kiem tra gan nhat
```

### Store detail moi

Cau truc:

```text
Header cua hang
Trust score
Breakdown diem uy tin
Cam ket moi nhat
San pham
Danh gia
```

## 7. Migration/backfill

### 7.1 Du lieu cu

Vi du lieu hien tai chua co batch:

- Tao batch mac dinh cho moi product active.
- `batchCode = "DEFAULT-" + productId`.
- Copy `freshnessScore` tu product sang `currentFreshness`.
- Gan tat ca pledge cu vao default batch theo product.
- Gan freshness report cu vao default batch theo product.
- Buyer check cu neu co productId thi gan vao default batch.

### 7.2 Script

Them command:

- `cmd/backfill-batches/main.go`

Chuc nang:

- scan products
- tao default batch neu chua co
- update pledge/freshness/buyer check neu thieu batchId
- dry-run mode
- resume mode

## 8. Test plan

### Backend unit tests

- Batch service:
  - tao batch hop le
  - khong cho tao batch voi product khac shop
  - update version conflict
  - recall batch
  - list batch theo product
- Trace service:
  - tao trace event
  - list timeline dung thu tu
  - validate event type
- Freshness report:
  - create report gan batch
  - update current freshness cua batch
  - tao warning khi score giam bat thuong
- Seller commit:
  - commit pledge voi batch active
  - reject batch expired/recalled
- Buyer check:
  - token co batchId
  - pledge mismatch batch bi reject
  - expired/recalled batch tra verdict canh bao

### Backend handler tests

- Test tat ca API moi.
- Test auth/owner permissions.
- Test public read endpoints.
- Test admin list/moderation.

### Android tests

- Mapper DTO -> UI model.
- ViewModel product detail load batch/freshness/timeline.
- Seller create pledge yeu cau batch.
- Buyer check result hien thi delta va reasons.

### Manual test flow

1. Seller tao product.
2. Seller tao batch.
3. Seller them origin + packaging trace event.
4. Seller tao pledge cho batch.
5. Seller in QR.
6. Buyer scan QR.
7. Buyer xem product detail voi batch.
8. Buyer chup anh check.
9. App hien thi result, delta, reasons.
10. Store trust score cap nhat.

## 9. Thu tu trien khai de xuat

### Phase 1: Batch foundation

- Them ProductBatch domain/repository/service/API.
- Them Android DTO/API/repository.
- Them seller UI tao/list batch.
- Backfill default batch cho du lieu cu.

Ket qua: he thong bat dau quan ly theo lo.

### Phase 2: Freshness history by batch

- Them batchId vao freshness report.
- API freshness history theo batch.
- Android product detail them tab `Lo hang & do tuoi`.
- Them chart/timeline freshness don gian.

Ket qua: nguoi dung xem duoc lich su do tuoi tung lo.

### Phase 3: Batch-aware pledge and QR

- Them batchId vao pledge.
- Them batchId vao bundle token/QR payload.
- Cap nhat seller create pledge.
- Cap nhat buyer check.

Ket qua: QR va AI check gan dung lo hang.

### Phase 4: Trace timeline

- Them TraceEvent domain/repository/service/API.
- Seller them trace event.
- Buyer xem timeline nguon goc.
- Trace event quan trong co audit/integrity.

Ket qua: app co truy xuat nguon goc that su.

### Phase 5: Proof and trust transparency

- Proof viewer UI.
- Store trust score breakdown UI.
- Product/batch warning badges.
- Buyer result hien thi pledged vs actual vs delta.

Ket qua: nguoi mua hieu duoc vi sao nen tin hoac can canh bao.

### Phase 6: Admin and operations

- Admin list/moderate batch.
- Admin list trace events.
- Recall workflow.
- Alert khi batch co nhieu buyer check xau.
- Dashboard canh bao.

Ket qua: san sang van hanh va xu ly su co.

## 10. Definition of Done

Hoan thanh khi:

- Product co the co nhieu batch.
- Seller tao batch va tao pledge cho batch.
- QR payload chua batchId.
- Buyer scan QR thay dung batch.
- Product detail hien thi lich su do tuoi theo batch.
- Product detail hien thi timeline truy xuat nguon goc.
- Buyer check luu va hien thi batchId.
- Store detail co trust score breakdown.
- Proof viewer doc duoc API proof hien co.
- Du lieu cu duoc backfill batch mac dinh.
- Backend test pass.
- Android build pass.

## 10.1 Trang thai thuc hien

### Luong 1: Backend batch foundation

Trang thai: `done`

Da lam:

- Them `ProductBatch` domain model.
- Them DTO create/update/list/detail cho product batch.
- Them `ProductBatchRepository` contract.
- Them Firestore repository cho `product_batches`.
- Them MongoDB repository cho `product_batches`.
- Them `internal/service/batch` voi create/update/delete/get/list.
- Them handler REST cho product batch.
- Them route public:
  - `GET /v1/shops/:shopId/products/:productId/batches`
  - `GET /v1/shops/:shopId/products/:productId/batches/:batchId`
- Them route authenticated:
  - `POST /v1/shops/:shopId/products/:productId/batches`
  - `PUT /v1/shops/:shopId/products/:productId/batches/:batchId`
  - `DELETE /v1/shops/:shopId/products/:productId/batches/:batchId`
- Wire service/handler/repository vao `cmd/server`.

Test da chay:

```sh
go test ./internal/service/batch ./internal/api/handler ./internal/api/router ./cmd/server
```

### Luong 2: Android batch client foundation

Trang thai: `done`

Da lam:

- Them `ProductBatchDTO.kt` cho request/response batch.
- Them Retrofit endpoints:
  - `getProductBatches`
  - `getProductBatch`
  - `createProductBatch`
  - `updateProductBatch`
  - `deleteProductBatch`
- Them `ProductBatchRepository`.
- Them UI model `ProductBatch`.
- Them mapper `ProductBatchResponse.toUiModel()`.

Test da chay:

```sh
./gradlew :app:compileDevDebugKotlin
```

### Luong 3: Product detail batch visibility

Trang thai: `done`

Da lam:

- Cap nhat `ProductDetailViewModel` de load batches theo product.
- Them state `batches` cho man hinh chi tiet san pham.
- Them section `Lo hang & do tuoi` trong `ProductDetailScreen`.
- Hien thi empty state khi san pham chua co batch.
- Hien thi batch cards voi ma lo, do tuoi hien tai, trang thai, han dung va nguon.

Test da chay:

```sh
./gradlew :app:compileDevDebugKotlin
```

### Luong 4: Backend freshness reports by batch

Trang thai: `done`

Da lam:

- Them `batchId` vao `ProductFreshnessReport` domain.
- Them `batchId` vao create request va response DTO.
- Them `batchId` vao `ProductFreshnessReportListFilter`.
- Cap nhat Firestore/Mongo freshness report repositories de filter theo `batchId`.
- Cap nhat product service de luu `batchId` khi tao freshness report.
- Cap nhat API list freshness reports de ho tro query `?batchId=...`.
- Cap nhat handler/router tests va product service tests.

Test da chay:

```sh
go test ./...
```

### Luong 5: Android freshness reports by batch

Trang thai: `done`

Da lam:

- Them `batchId` vao `ProductFreshnessReportResponse`.
- Them `batchId` vao `CreateFreshnessReportRequest`.
- Cap nhat Retrofit `getFreshnessReports` de ho tro query `batchId`.
- Them repository methods:
  - `listFreshnessReports(shopId, productId, batchId)`
  - `createFreshnessReport(shopId, productId, request)`

Test da chay:

```sh
./gradlew :app:compileDevDebugKotlin
```

### Luong 6: Backend batch-aware seller pledge

Trang thai: `done`

Da lam:

- Them `batchId` vao `Pledge` domain.
- Them `batchId` vao seller commit request/response.
- Them `batchId` vao pledge history response va pledge proof response.
- Cap nhat seller service de luu `batchId` vao pledge.
- Cap nhat seller handler de truyen `batchId` vao commit va bundle token issue.
- Them `batchId` vao bundle token issue input va JWT claims.
- Them `batchId` vao bundle token verify claims output.
- Cap nhat shop proof bundle de expose `batchId`.

Test da chay:

```sh
go test ./...
```

### Luong 7: Android batch-aware seller pledge DTO and QR

Trang thai: `done`

Da lam:

- Them `batchId` vao `SellerCommitRequest`.
- Them `batchId` vao `SellerCommitResponse`.
- Them `batchId` vao `PledgeResponse`.
- Them `batchId` vao `PledgeProofBundleResponse`.
- Them `batchId` vao `VNMeatQrPayload`.
- Cap nhat QR URI parse/render de giu `batchId`.
- Cap nhat seller create pledge ViewModel de dua `batchId` tu response vao QR payload.

Test da chay:

```sh
./gradlew :app:compileDevDebugKotlin
```

## 11. Rủi ro va luu y

- Them batchId vao token/QR co the lam QR cu khong tuong thich. Can giu fallback cho QR cu khong co batchId.
- Backfill phai chay dry-run truoc khi ghi that.
- Neu freshness score dang co thang 0-10 va product UI hien nhu phan tram, can chuan hoa ro thang diem.
- Freshness history nen luu score raw va display scale rieng de tranh nham lan.
- Trace event co the tang nhanh ve so luong, can index tot theo `batchId` va `occurredAt`.
- Recall la tinh nang nhay cam, can audit log va chi owner/admin moi duoc thao tac.
