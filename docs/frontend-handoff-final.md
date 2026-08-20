# Frontend Handoff Final

## Goal
Build the mobile app without needing blockchain-specific knowledge. The app should present trust, freshness, and proof in plain language.

## Screen map
- `Auth`: register, login, refresh token
- `Shop list`: search shops, show trust badge and summary score
- `Shop detail`: show pledge history, latest proof state, product list
- `Product detail`: show freshness metadata, product images, freshness reports
- `Seller flow`: upload image, score image, confirm pledge
- `Buyer flow`: upload image, check product, compare against pledge if available
- `Proof detail`: show verified, pending, warning, or revoked state
- `My account`: basic profile and seller-owned content

## Primary API sequence
- Seller:
  1. `POST /v1/media/images`
  2. `POST /v1/seller/score`
  3. `POST /v1/seller/commit`
  4. `GET /v1/shops/:shopId/pledges`
  5. `GET /v1/shops/:shopId/pledges/:pledgeId/proof`
- Buyer:
  1. `POST /v1/media/images`
  2. `POST /v1/buyer/check`
  3. `GET /v1/shops/:shopId/pledges/:pledgeId/proof`
  4. `GET /v1/shops/:shopId/products/:productId/freshness-reports`

## UI text contract
- `verified`: show "Cam ket da duoc xac thuc"
- `pending`: show "Dang dong bo"
- `warning`: show "Du lieu can duoc kiem tra lai"
- `revoked`: show "Cam ket da bi thu hoi"
- `no_pledge`: show "Chua co cam ket tu nguoi ban"

## What frontend should ignore
- Besu, QBFT, contract addresses
- Vault signing details
- event log chain internals
- raw integrity mismatch diagnostics unless building admin UI

## QA handoff
- Use [scripts/e2e-mobile-flow.sh](../scripts/e2e-mobile-flow.sh) for smoke tests.
- Use [docs/mobile-api-playbook.md](mobile-api-playbook.md) for endpoint-by-endpoint payloads.
- Use [docs/mobile-handoff.md](mobile-handoff.md) for UI mapping.
