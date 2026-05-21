# Ke hoach thiet ke lai tab Kham pha dang map

## 0. Quy tac thuc hien

- Lam theo tung luong nho, moi luong co code, test va commit rieng.
- Xong luong nao cap nhat trang thai vao file nay truoc khi commit.
- Khong stage/commit cac thay doi ngoai pham vi, dac biet `.gitignore` va `test.sh` neu dang la thay doi rieng cua nguoi dung.
- Neu test khong chay duoc, ghi ro ly do va test da chay.
- Uu tien sua cac lo hong logic backend truoc UI nang cao.

## 1. Muc tieu

Thiet ke lai tab `Kham pha` tu danh sach shop thanh trai nghiem tim shop tren map, dung du lieu shop that tu API hien co.

Ket qua mong muon:

- Nguoi mua thay shop gan minh tren ban do.
- Marker the hien trust score/trang thai cua shop.
- Co bottom sheet de xem nhanh shop va danh sach shop.
- Tap marker hoac shop card de vao `StoreDetailScreen`.
- Co fallback khi chua co quyen location hoac shop thieu toa do.

## 2. Nguyen tac san pham

- Map la man hinh chinh cua tab, khong lam landing page.
- Thong tin uu tien tren marker/card:
  - ten shop
  - trust score/grade
  - dia chi ngan
  - so danh gia
  - pledge moi nhat neu co
- Khong chan nguoi dung neu chua co location permission.
- Neu chua tich hop Google Maps API key, dung map-like UI truoc de demo nghiep vu va giam rui ro.

## 3. Hien trang can kiem tra

Trang thai: `done`

- `ExploreScreen` hien dang render list shop.
- `ShopResponse` da co `latitude`, `longitude`.
- `ExploreViewModel`/repository da load shop that tu API.
- Chua co location permission flow.
- Chua co dependency Google Maps Compose.
- Chua co bottom sheet map/list.

## 4. Thu tu uu tien

1. Kiem tra va sua logic backend/data shop location neu toa do khong hop le.
2. Them mapping UI cho shop location/trust marker.
3. Redesign `ExploreScreen` thanh map-like UI dung Compose thuong.
4. Them bottom sheet danh sach shop + selected shop preview.
5. Them search/filter tren map.
6. Them location permission va fallback location.
7. Sau khi UI on dinh, tich hop Google Maps Compose neu co API key.

## 5. Luong 1: Audit du lieu shop location va trust

Trang thai: `done`

### Muc tieu

Dam bao UI map co data dung de render marker va card.

### Tasks

- Kiem tra `ShopResponse` tu API co `latitude`, `longitude`, `trustSummary`, `ratingSummary`.
- Kiem tra seed data/dev data co toa do hop le.
- Neu shop thieu toa do:
  - dung toa do fallback theo thanh pho.
  - gan marker vao list fallback, khong render marker sai.
- Xac dinh rule mau marker:
  - trust score >= 80: xanh.
  - trust score 50-79: vang.
  - trust score < 50 hoac chua co: do/xam.

### Files du kien

- `Server/cmd/seed-mongo/main.go` neu seed thieu toa do.
- `VNMeat/app/src/main/java/com/example/vnmeat/data/api/dto/ShopDTO.kt`
- `VNMeat/app/src/main/java/com/example/vnmeat/ui/viewmodel/ExploreViewModel.kt`
- `VNMeat/app/src/main/java/com/example/vnmeat/ui/screens/ExploreScreen.kt`

### Tests

- Backend: `go test ./...` neu co sua Server.
- Android: `./gradlew assembleDevDebug` neu co sua VNMeat.

### Ket qua da lam

- `ShopResponse` tren Android da co `latitude`, `longitude`, `trustSummary`, `ratingSummary`.
- `domain.Shop` va API shop response backend da co toa do.
- Seed Mongo hien co toa do hop le cho `VNMeat Ben Thanh` va `VNMeat Thao Dien`.
- Chua can sua backend cho luong audit.

### Test da chay

- Khong chay test vi luong nay chi audit du lieu va cap nhat ke hoach.

## 6. Luong 2: Map-like UI MVP bang Compose thuong

Trang thai: `done`

### Muc tieu

Co giao dien map demo duoc nghiep vu ma chua phu thuoc Google Maps API key.

### Tasks

- Thay layout list-first cua `ExploreScreen` bang map-first layout.
- Tao composable:
  - `ExploreMapScreen`
  - `ShopMapMarker`
  - `SelectedShopPreview`
  - `ShopBottomSheet`
- Render marker theo toa do shop da normalize vao khung map.
- Tap marker set `selectedShop`.
- Tap selected shop card goi `onNavigateToStore(shopId)`.
- Giu search bar noi tren map.
- Giu danh sach shop trong bottom sheet.

### Acceptance criteria

- Vao tab Kham pha thay map chiem phan lon man hinh.
- Marker hien theo du lieu shop.
- Tap marker hien card shop.
- Tap card vao chi tiet cua hang.
- Khong crash khi shop khong co toa do.

### Tests

- `./gradlew assembleDevDebug`
- Manual:
  - login buyer.
  - vao Kham pha.
  - tap marker.
  - tap shop preview.
  - quay lai tab Kham pha.

### Ket qua da lam

- Thay `ExploreScreen` tu list-first sang map-first UI bang Compose thuong.
- Them map background custom, marker theo shop, selected shop preview va bottom sheet danh sach shop.
- Marker dung toa do shop da normalize trong tap du lieu hien co.
- Tap marker set selected shop; tap preview/list action mo `StoreDetailScreen`.
- Giu search bar tren map va filter chip co ban.
- Them fallback marker cho shop thieu toa do, khong crash khi latitude/longitude bang 0.

### Test da chay

- `./gradlew assembleDevDebug`

## 7. Luong 3: Search va filter tren map

Trang thai: `done`

### Muc tieu

Nguoi dung loc nhanh shop theo ten, dia chi va trust.

### Tasks

- Search theo:
  - ten shop
  - dia chi
  - mo ta
- Them filter chips:
  - Gan toi
  - Trust cao
  - Co cam ket
  - Nhieu danh gia
- Khi filter thay doi:
  - marker list cap nhat.
  - bottom sheet list cap nhat.
  - selected shop bi clear neu khong con trong ket qua.

### Acceptance criteria

- Search/filter tac dong dong thoi map va list.
- Empty state ro rang khi khong co shop phu hop.
- Khong mat navigation vao store detail.

### Tests

- `./gradlew assembleDevDebug`
- Manual search/filter voi data seed.

### Ket qua da lam

- Doi filter sang enum ro nghia gom `Tat ca`, `Gan ban`, `Uy tin cao`, `Co cam ket`, `Nhieu danh gia`.
- Search theo ten, dia chi va mo ta shop.
- Filter cap nhat dong thoi marker tren map va list trong bottom sheet.
- Selected shop tu dong clear neu khong con nam trong ket qua filter/search.
- Them summary so luong shop va mo ta filter dang ap dung.
- Them empty state ro ly do va nut xoa search khi dang co tu khoa.
- Sort rieng theo tung filter:
  - `Gan ban` dung khoang cach tu center mac dinh TP.HCM.
  - `Nhieu danh gia` uu tien rating count.
  - `Co cam ket` uu tien pledge count.
  - mac dinh uu tien trust score.

### Test da chay

- `./gradlew assembleDevDebug`

## 8. Luong 4: Location permission va fallback

Trang thai: `done`

### Muc tieu

Ho tro trai nghiem gan toi nhung khong phu thuoc bat buoc vao location.

### Tasks

- Them permission runtime cho location neu can.
- Khi co location:
  - tinh khoang cach tu user den shop.
  - sort theo gan toi khi filter `Gan toi`.
- Khi khong co location:
  - dung center mac dinh TP.HCM.
  - hien copy nhe trong bottom sheet: "Bat vi tri de sap xep theo khoang cach".
- Khong hoi permission lien tuc.

### Acceptance criteria

- Tu choi permission van xem duoc map va shop list.
- Cho phep permission thi hien khoang cach.
- App khong crash tren emulator khong co GPS.

### Tests

- `./gradlew assembleDevDebug`
- Manual:
  - deny permission.
  - allow permission.
  - emulator khong co location.

### Ket qua da lam

- Them permission `ACCESS_COARSE_LOCATION` va `ACCESS_FINE_LOCATION` vao Android manifest.
- Them request permission trong tab Kham pha khi nguoi dung can sap xep `Gan ban`.
- Doc last known location tu `LocationManager` neu da co quyen.
- Sap xep filter `Gan ban` theo khoang cach haversine tu vi tri nguoi dung khi co du lieu.
- Neu chua co quyen/khong lay duoc GPS, giu fallback center TP.HCM va hien banner trong bottom sheet.
- Hien khoang cach tren shop list khi co user location.
- Khong tu dong hoi permission lien tuc; chi hoi khi nguoi dung bam nut bat vi tri.

### Test da chay

- `./gradlew assembleDevDebug`

## 9. Luong 5: Google Maps Compose integration

Trang thai: `pending`

### Muc tieu

Thay map-like UI bang Google Maps that khi da co API key va dependency.

### Dieu kien bat dau

- Da co Google Maps API key.
- Da quyet dinh noi luu key:
  - `local.properties`
  - manifest placeholder
  - hoac BuildConfig theo flavor.
- MVP map-like UI da pass build va manual flow.

### Tasks

- Them dependency `maps-compose`.
- Them config API key khong commit secret.
- Render `GoogleMap`.
- Render `Marker` theo shop.
- Dong bo tap marker voi selected shop preview.
- Giu bottom sheet va filter logic tu luong 2/3.

### Acceptance criteria

- Map Google render khong blank.
- Marker shop render dung.
- Search/filter van hoat dong.
- Khong commit API key that.

### Tests

- `./gradlew assembleDevDebug`
- Manual tren emulator co Google Play services.

### Ket qua da lam

- Chua thuc hien.

## 10. Luong 6: Polish UI va accessibility

Trang thai: `pending`

### Muc tieu

Lam UI map du dung cho demo/san pham ma khong anh huong logic.

### Tasks

- Marker state:
  - selected
  - normal
  - disabled/no-coordinate
- Bottom sheet:
  - collapsed preview
  - half expanded list
  - empty state
- Content description cho marker/card button.
- Dam bao text khong tran tren mobile.
- Dam bao mau marker khong chi phu thuoc vao mau, co label/score.

### Tests

- `./gradlew assembleDevDebug`
- Manual tren man hinh nho va man hinh lon.

### Ket qua da lam

- Chua thuc hien.

## 11. Commit plan

- Commit 1: `Audit explore shop location data`
- Commit 2: `Redesign explore tab as map view`
- Commit 3: `Add explore map search filters`
- Commit 4: `Add location fallback for explore map`
- Commit 5: `Integrate Google Maps for explore`
- Commit 6: `Polish explore map interactions`

Moi commit phai cap nhat `Trang thai` va `Ket qua da lam` cua luong tuong ung trong file nay truoc khi commit.

## 12. Rui ro va cach giam rui ro

- Google Maps API key thieu hoac sai: lam map-like UI truoc.
- Shop thieu toa do: fallback sang list/bottom sheet va khong render marker sai.
- Permission location gay crash: location la optional, khong chan flow.
- UI map qua nang: giu logic search/filter trong ViewModel/composable nho, tranh gom het vao mot file qua lon.
- Thay doi backend ngoai pham vi: chi sua seed/data validation neu can cho marker dung.
