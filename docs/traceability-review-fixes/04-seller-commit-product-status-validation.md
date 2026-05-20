# Luong 04: Seller commit validate trang thai product

Trang thai: `done`

## Van de

Seller commit da validate shop owner va batch active, nhung product validation moi chi check product ton tai va thuoc shop. Product bi archived/deleted/pending van co nguy co tao pledge neu repository tra ve record.

## Muc tieu

- Seller chi tao pledge cho product hop le.
- Product phai thuoc shop, owner khop seller, va status nam trong tap cho phep.
- Neu co batch, batch/product/shop/owner/status phai khop nhu hien tai.

## Policy de xuat

- Cho phep: `active`, co the `published` neu he thong dung status nay cho public product.
- Reject: `draft`, `archived`, `deleted`.

## Pham vi code du kien

- `internal/service/seller/service.go`
- `internal/service/seller/service_test.go`

## Backend tasks

- Sau khi load product:
  - reject neu product empty
  - product shop khop input shop
  - product owner khop seller neu field co gia tri
  - product status hop le
- Dung error `ErrInvalidCommit` hoac `ErrShopOwnership` tuy tinh huong.
- Cap nhat test hien co ve seller commit batch neu bi anh huong.

## Tests can co

- Commit thanh cong voi active product.
- Commit thanh cong/that bai voi published tuy policy.
- Reject deleted product.
- Reject archived/draft product.
- Reject product owner khac seller.
- `go test ./internal/service/seller`
- `go test ./...`

## Da lam

- Them validate product rieng cho seller commit.
- Product co `productId` phai ton tai, thuoc dung shop va co status hop le.
- Cho phep product status `active` va `published`.
- Reject product status rong, `draft`, `archived`, `deleted`.
- Reject product co `OwnerUserID` khac seller neu field nay co gia tri.
- Cap nhat test fixture product sang active.
- Them test cho published product, inactive product va owner mismatch.

## Test da chay

- `go test ./internal/service/seller`
- `go test ./...`

## Commit

Da commit:

```text
Validate seller pledge product status
```
