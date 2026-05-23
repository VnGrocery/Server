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
- Uu tien OpenStreetMap/osmdroid thay vi Google Maps de tranh API key, billing va phu thuoc Google Play Services.
- Map-like UI hien tai la fallback bat buoc neu OSM tile loi, thieu toa do hoac emulator khong tai duoc tile.

## 3. Hien trang can kiem tra

Trang thai: `done`

- `ExploreScreen` hien dang render list shop.
- `ShopResponse` da co `latitude`, `longitude`.
- `ExploreViewModel`/repository da load shop that tu API.
- Chua co location permission flow.
- Chua co dependency osmdroid/OpenStreetMap.
- Chua co bottom sheet map/list.

## 4. Thu tu uu tien

1. Kiem tra va sua logic backend/data shop location neu toa do khong hop le.
2. Them mapping UI cho shop location/trust marker.
3. Redesign `ExploreScreen` thanh map-like UI dung Compose thuong.
4. Them bottom sheet danh sach shop + selected shop preview.
5. Them search/filter tren map.
6. Them location permission va fallback location.
7. Sau khi UI on dinh, tich hop OpenStreetMap/osmdroid.
8. Giu map-like UI lam fallback neu OSM khong kha dung.

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

## 9. Luong 5: OpenStreetMap/osmdroid integration

Trang thai: `pending`

### Muc tieu

Thay phan nen map-like bang OpenStreetMap that bang `osmdroid`, khong can Google Maps API key.

### Dieu kien bat dau

- MVP map-like UI da pass build va manual flow.
- Xac dinh tile policy:
  - MVP/dev co the dung tile public mac dinh cua OSM voi user-agent ro rang.
  - Production can tile provider rieng hoac self-host, khong abuse public tile server.

### Tasks

- Them dependency `osmdroid-android`.
- Them cau hinh user-agent/cache cho osmdroid.
- Tao composable bridge Android View:
  - `OsmExploreMap`
  - `rememberMapViewWithLifecycle` neu can quan ly lifecycle.
- Render `MapView` voi center mac dinh TP.HCM hoac user location neu co.
- Render marker theo `ShopResponse.latitude/longitude`.
- Marker mau/icon theo trust score neu osmdroid cho phep custom drawable; neu khong, dung default marker + title/snippet.
- Dong bo tap marker voi selected shop preview.
- Giu bottom sheet va filter logic tu luong 2/3.
- Giu map-like fallback:
  - khi khong co shop co toa do
  - khi map view loi render
  - khi muon chay offline/dev khong can tile

### Acceptance criteria

- OSM map render khong blank tren emulator/may that co internet.
- Marker shop co toa do render dung tren OSM.
- Shop thieu toa do khong lam crash, van nam trong bottom sheet.
- Tap marker cap nhat selected shop preview.
- Search/filter van hoat dong.
- Khong can commit API key.
- Map-like UI van la fallback co the bat lai nhanh neu OSM gap loi.

### Tests

- `./gradlew assembleDevDebug`
- Manual tren emulator/may that co internet:
  - vao tab Kham pha
  - thay OSM tile
  - tap marker
  - search/filter
  - tap shop preview vao detail
- Manual fallback:
  - data shop thieu toa do hoac tat OSM flag neu co.

### Ket qua da lam

- Chua thuc hien.

### Test da chay

- Chua chay.

## 9.1 Luong 5a: Tach map UI thanh fallback va OSM adapter

Trang thai: `done`

### Muc tieu

Giam rui ro khi them OSM bang cach tach phan map hien tai thanh fallback rieng, sau do moi bridge MapView.

### Tasks

- Doi `ExploreMapCanvas` hien tai thanh `FallbackExploreMap`.
- Tao interface/composable input chung:
  - shops
  - selectedShopId
  - onSelectShop
- Tao `ExploreMapSurface` de quyet dinh render fallback hay OSM.
- Chua them osmdroid o luong nay neu muon commit nho.

### Acceptance criteria

- UI hien tai khong doi hanh vi.
- Build pass.
- Code san sang gan `OsmExploreMap`.

### Tests

- `./gradlew assembleDevDebug`

### Ket qua da lam

- Doi loi goi map chinh sang `ExploreMapSurface`.
- Doi map-like UI hien tai thanh `FallbackExploreMap`.
- Giu chung input `shops`, `selectedShopId`, `onSelectShop` de luong sau gan `OsmExploreMap`.
- Chua them dependency osmdroid trong luong nay de giu commit nho va build on dinh.

### Test da chay

- `./gradlew assembleDevDebug`

## 9.2 Luong 5b: Them osmdroid MapView va marker

Trang thai: `done`

### Muc tieu

Render OpenStreetMap that va marker shop that trong tab Kham pha.

### Tasks

- Them dependency osmdroid.
- Them AndroidView cho `MapView`.
- Set tile source, zoom, center va multi-touch.
- Them marker cho shop co toa do.
- Marker click set selected shop va consume event.
- Clear/update overlays khi list filter thay doi.
- Quan ly lifecycle neu can de tranh memory leak.

### Acceptance criteria

- OSM tile render.
- Marker update theo search/filter.
- Tap marker cap nhat preview.
- Build pass.

### Tests

- `./gradlew assembleDevDebug`
- Manual tren emulator/may that co internet.

### Ket qua da lam

- Them dependency `org.osmdroid:osmdroid-android:6.1.18`.
- Them `OsmExploreMap` bang `AndroidView` bridge toi `MapView`.
- Cau hinh tile source `MAPNIK`, multi-touch, zoom va center.
- Set user-agent cho osmdroid bang package name.
- Render marker cho shop co toa do.
- Tap marker cap nhat selected shop preview va hien info window.
- Giu fallback map-like khi khong co shop co toa do.
- Search/filter tiep tuc dieu khien danh sach marker vi OSM nhan cung `filteredShops`.

### Test da chay

- `./gradlew assembleDevDebug`

## 9.3 Luong 5c: OSM fallback, cache va production note

Trang thai: `done`

### Muc tieu

Dam bao OSM integration khong lam app mong manh va co ghi chu dung cho production.

### Tasks

- Them flag/fallback de quay ve map-like UI khi can.
- Hien empty/fallback khi khong co shop co toa do.
- Ghi note trong plan ve tile server production.
- Kiem tra app van build neu khong co API key nao.

### Acceptance criteria

- Khong co API key van build/run.
- Co fallback ro.
- Plan ghi ro production tile provider/self-host.

### Tests

- `./gradlew assembleDevDebug`

### Ket qua da lam

- Them flag `UseOpenStreetMap` de co the quay ve map-like fallback nhanh neu can debug/offline.
- `ExploreMapSurface` chi render OSM khi flag bat va co shop co toa do; nguoc lai dung fallback map-like.
- Cau hinh osmdroid cache trong `context.cacheDir/osmdroid`.
- Giu production note trong plan: dev/MVP dung tile public nhe, production can tile provider rieng hoac self-host.
- Khong can API key; build van doc lap voi Google services.

### Test da chay

- `./gradlew assembleDevDebug`

## 10. Luong 6: Polish UI va accessibility

Trang thai: `done`

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

- Doi marker positioning sang responsive theo kich thuoc map thay vi offset co dinh.
- Them state marker selected bang kich thuoc/outline noi bat.
- Them state marker thieu toa do bang mau xam rieng va fallback point.
- Them content description cho marker, selected shop preview va shop list item.
- Giu score/trust label tren marker de khong chi phu thuoc vao mau.
- Dam bao text trong shop list/preview gioi han max line va ellipsis.

### Test da chay

- `./gradlew assembleDevDebug`

## 10.1 Luong 7: Nut ve vi tri hien tai va dia chi mac dinh

Trang thai: `done`

### Muc tieu

Nguoi dung co the dua map ve vi tri hien tai ro rang, khong phai dua vao gesture hoac filter `Gan ban`. Dia chi/mac dinh khu vuc tren map uu tien dia chi hien tai cua user khi app lay duoc location.

### Tasks

- Them nut noi tren map de xin quyen/lien ket ve vi tri hien tai.
- Khi bam nut:
  - neu da co quyen location, lay `lastKnownLocation` va center map ve user.
  - neu chua co quyen, mo permission request.
  - clear selected shop de map khong tiep tuc center vao marker dang chon.
- Lay dia chi hien tai bang Android `Geocoder` khi co toa do user.
- Hien fallback `Vi tri hien tai` neu co toa do nhung khong reverse geocode duoc.
- Giu fallback xem shop/map khi khong co location permission.

### Acceptance criteria

- Co nut vi tri hien tai tren tab Kham pha.
- Bam nut khi chua cap quyen se xin quyen location.
- Bam nut khi da co quyen se dua map ve vi tri user neu lay duoc location.
- UI hien dia chi hien tai/fallback thay vi chi phu thuoc trung tam TP.HCM.
- Build pass.

### Tests

- `./gradlew assembleDevDebug`
- Manual:
  - bam nut khi chua cap quyen.
  - bam nut khi da cap quyen.
  - emulator khong co last known location.

### Ket qua da lam

- Them `CurrentLocationMapButton` noi tren map.
- Dung lai permission flow hien co cho nut vi tri hien tai.
- Khi lay duoc location, clear selected shop de `OsmExploreMap` center ve user.
- Them reverse geocode bang `Geocoder` va hien dia chi hien tai trong control tren map.
- Co fallback text neu thiet bi khong tra ve dia chi.

### Test da chay

- `./gradlew assembleDevDebug`

## 10.2 Luong 8: Hoi quyen vi tri mot lan khi vao app

Trang thai: `done`

### Muc tieu

App chu dong xin quyen location mot lan khi user vao man hinh chinh lan dau, de tab Kham pha co the mac dinh theo vi tri user ma khong doi den khi user bam nut map.

### Tasks

- Them runtime permission request o `MainScreen`.
- Luu co da hoi permission vao `SharedPreferences`.
- Chi launch permission dialog neu:
  - chua tung hoi trong install hien tai.
  - app chua co quyen location.
- Khong hoi lai neu user da tu choi.
- Giu nut vi tri hien tai tren map de user co the bat/refresh thu cong sau nay.

### Acceptance criteria

- User vao app chinh lan dau thi thay dialog xin location neu chua cap quyen.
- Dialog khong bi lap lai moi lan compose/reopen app.
- Neu da co quyen location thi khong hien dialog thua.
- Build pass.

### Tests

- `./gradlew assembleDevDebug`
- Manual:
  - install moi/clear app data.
  - login vao app chinh.
  - accept permission.
  - clear app data va deny permission.
  - reopen app, khong thay prompt lap lien tuc.

### Ket qua da lam

- Them `rememberLauncherForActivityResult` xin `ACCESS_FINE_LOCATION` va `ACCESS_COARSE_LOCATION` trong `MainScreen`.
- Them `SharedPreferences` flag `initial_location_permission_requested`.
- Mark da hoi truoc khi launch dialog de tranh launch lap neu recomposition.
- Giu flow thu cong trong tab Kham pha qua nut vi tri hien tai.

### Test da chay

- `./gradlew assembleDevDebug`

## 10.3 Luong 9: Bottom sheet shop co the keo thu gon

Trang thai: `done`

### Muc tieu

Danh sach shop o tab Kham pha co the keo xuong de thu gon khi nguoi dung muon xem map rong hon, dac biet khi thao tac bang chuot tren emulator.

### Tasks

- Them drag state theo truc doc cho `ShopBottomSheet`.
- Dinh nghia hai trang thai:
  - expanded: sheet cao day du de xem danh sach.
  - collapsed: chi con phan dau sheet/handle.
- Snap sheet ve expanded/collapsed khi tha chuot/tay.
- Giu `LazyColumn` shop list va cac action hien co.
- Them content description cho handle drag.

### Acceptance criteria

- Keo sheet xuong se thu gon.
- Keo sheet len se mo lai.
- Danh sach shop van scroll/xem/mo shop binh thuong.
- Build pass.

### Tests

- `./gradlew assembleDevDebug`
- Manual:
  - keo sheet xuong bang chuot tren emulator.
  - keo sheet len de mo lai.
  - scroll danh sach shop.
  - bam `Xem` vao chi tiet shop.

### Ket qua da lam

- Them `Modifier.draggable` cho `ShopBottomSheet`.
- Them offset snap giua expanded va collapsed.
- Doi sheet sang chieu cao on dinh `280.dp` va peek height `56.dp`.
- Giu handle tren sheet va them content description cho hanh vi keo.

### Test da chay

- `./gradlew assembleDevDebug`

## 11. Commit plan

- Commit 1: `Audit explore shop location data`
- Commit 2: `Redesign explore tab as map view`
- Commit 3: `Add explore map search filters`
- Commit 4: `Add location fallback for explore map`
- Commit 5a: `Prepare explore map fallback adapter`
- Commit 5b: `Integrate OpenStreetMap for explore`
- Commit 5c: `Harden explore OSM fallback`
- Commit 6: `Polish explore map interactions`
- Commit 7: `Add explore current location control`
- Commit 8: `Request location permission on first main entry`
- Commit 9: `Make explore shop sheet draggable`

Moi commit phai cap nhat `Trang thai` va `Ket qua da lam` cua luong tuong ung trong file nay truoc khi commit.

## 12. Rui ro va cach giam rui ro

- OSM public tile policy: dev/MVP dung nhe; production dung tile provider rieng hoac self-host.
- OSM tile loi/network loi: giu map-like UI fallback.
- osmdroid la Android View, khong Compose-native: can bridge bang `AndroidView` va quan ly lifecycle can than.
- Shop thieu toa do: fallback sang list/bottom sheet va khong render marker sai.
- Permission location gay crash: location la optional, khong chan flow.
- UI map qua nang: giu logic search/filter trong ViewModel/composable nho, tranh gom het vao mot file qua lon.
- Thay doi backend ngoai pham vi: chi sua seed/data validation neu can cho marker dung.
