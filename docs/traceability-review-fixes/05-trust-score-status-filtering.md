# Luong 05: Loc status khi tinh trust score

Trang thai: `todo`

## Van de

Trust score hien lay tat ca reviews va buyer checks cua shop. Review da deleted/rejected co the van anh huong average rating. Buyer check rejected bi skip trong score con nhung `BuyerCheckCount` van dem tat ca, lam summary lech.

## Muc tieu

- Trust score chi tinh tin hieu hop le.
- Count trong summary phai khop voi tap du lieu duoc tinh.
- Ly do trust/risk khong bi anh huong boi record da bi moderation reject/delete.

## Pham vi code du kien

- `internal/service/shop/service.go`
- `internal/service/shop/service_test.go`
- Co the can cap nhat repository test neu list behavior duoc doi.

## Backend tasks

- Loc reviews chi `ReviewStatusActive` truoc khi tinh rating summary va review score.
- Loc buyer checks:
  - score chi tinh check khong `rejected`
  - count nen la eligible count, hoac them field rieng neu muon hien total/raw
  - flagged van tinh voi weight thap nhu hien tai
- Loc pledge theo status `committed` neu co pledge revoke/deleted trong tuong lai.
- Cap nhat reasons de phan biet `no_buyer_checks` voi `no_eligible_buyer_checks`.

## Tests can co

- Deleted review khong lam thay doi average rating.
- Rejected buyer check khong tang `BuyerCheckCount`.
- Flagged buyer check van tinh voi weight thap.
- High risk eligible check tang `HighRiskCheckCount`.
- `go test ./internal/service/shop`
- `go test ./...`

## Da lam

- Chua lam.

## Test da chay

- Chua chay.

## Commit

De xuat:

```text
Filter trust signals by status
```
