# Mobile API Playbook

## Seller flow
1. `POST /v1/media/images` with multipart `image`
2. `POST /v1/seller/score` with multipart `image`
3. `POST /v1/seller/commit` with `shopId`, optional `productId`, `score`, `category`, `confidence`, `imageHash`, optional `imageCid`
4. `GET /v1/shops/:shopId/pledges?productId=:productId` for pledge history
5. `GET /v1/shops/:shopId/pledges/:pledgeId/proof` for buyer-facing proof summary

## Buyer flow
1. `POST /v1/media/images` with multipart `image`
2. `POST /v1/buyer/check` with multipart `image` and optional `pledgeId`
3. `GET /v1/shops/:shopId/pledges/:pledgeId/proof` for final proof view
4. `GET /v1/shops/:shopId/products/:productId/freshness-reports` for public freshness context

## Shop detail screen
- `GET /v1/shops/:shopId`
- Read `trustSummary.score`, `trustSummary.grade`, `trustSummary.formulaVersion`
- Use `trustSummary.consistencyScore`, `trustSummary.recencyScore`, `trustSummary.coverageScore` for breakdown cards

## Product detail screen
- `GET /v1/shops/:shopId/products/:productId`
- `GET /v1/shops/:shopId/products/:productId/freshness-reports`
- `GET /v1/shops/:shopId/pledges?productId=:productId`

## Proof rendering rules
- `proofStatus=verified`: green verified badge
- `proofStatus=pending`: neutral syncing badge
- `proofStatus=warning`: warning state, reduce trust emphasis
- `proofStatus=revoked`: revoked state, hide verified badge

## Retry rules
- Media upload can be retried independently.
- Keep `imageHash` and `imageCid` locally until seller commit or buyer check succeeds.
- If proof is `pending`, poll again later instead of failing the user flow.

## Recommended polling
- `proofStatus=pending`: poll every `10-20s`
- `chainAnchorStatus=failed`: stop polling and surface retry/support UI
- trust summary on shop detail: refresh on screen open and after successful buyer check or seller commit
