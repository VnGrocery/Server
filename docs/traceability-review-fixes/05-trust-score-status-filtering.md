# Luong 05: Loc status khi tinh trust score

Trang thai: `done`

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

- Loc pledge truoc khi tinh trust summary: chi tinh pledge status `committed`.
- Loc review truoc khi tinh rating/trust score: chi tinh `active`.
- Loc buyer check truoc khi tinh trust score/count: chi tinh `completed` va `flagged`.
- `BuyerCheckCount`, `TrustedCheckCount`, `HighRiskCheckCount`, coverage, recency va consistency deu dung tap eligible sau loc.
- Cap nhat test trust summary hien co de set status ro rang.
- Them test dam bao draft pledge, deleted review va rejected buyer check khong bi tinh vao summary.

## Test da chay

- `go test ./internal/service/shop`
- `go test ./...`

## Commit

Da commit:

```text
Filter trust signals by status
```
