# Mobile Handoff

## Main screens
- `Auth`: `POST /v1/auth/register`, `POST /v1/auth/login`, `POST /v1/auth/google`, `POST /v1/auth/refresh`
- `Shop feed`: `GET /v1/shops`
- `Shop detail`: `GET /v1/shops/:shopId`, `GET /v1/shops/:shopId/pledges`
- `Product detail`: `GET /v1/shops/:shopId/products/:productId`, `GET /v1/shops/:shopId/products/:productId/freshness-reports`
- `Seller pledge`: `POST /v1/media/images`, `POST /v1/seller/score`, `POST /v1/seller/commit`
- `Buyer check`: `POST /v1/media/images`, `POST /v1/buyer/check`
- `Proof screen`: `GET /v1/shops/:shopId/pledges/:pledgeId/proof`

## Per-screen data contract
- `Shop feed`: use `trustSummary.score`, `trustSummary.grade`, `trustSummary.formulaVersion`
- `Shop detail`: show latest pledge history and proof CTA when a pledge exists
- `Product detail`: show `freshnessScore`, `freshnessNote`, product images, and recent freshness reports
- `Seller pledge`: upload once, then reuse `imageHash`, `imageCid`, and `gatewayUrl`
- `Buyer check`: prefer image upload first, then attach `pledgeId` when the user is checking against a known seller pledge
- `Proof screen`: render `proofStatus`, `proofHeadline`, `proofSummary`, `recommendedActions` as primary copy

## UI state mapping
- `proofStatus=verified`: green verified state, show trust badge
- `proofStatus=pending`: neutral syncing state, allow pull-to-refresh
- `proofStatus=warning`: warning banner, reduce trust emphasis
- `proofStatus=revoked`: revoked badge, disable verified UI
- `grade=excellent|good`: positive trust treatment
- `grade=watch|risk`: warning treatment

## Frontend rules
- Always upload image first with `POST /v1/media/images` for retryable flows.
- Cache `imageCid` and `gatewayUrl` locally until the user finishes commit or check.
- Treat `recommendedActions` as machine-friendly hints for UI logic.
- Treat `reasons` arrays as diagnostics for support or admin tooling, not primary user copy.
- Do not show blockchain wording by default. Use `Cam ket da duoc xac thuc`, `Dang dong bo`, `Can xem xet`, `Da thu hoi`.

## Example proof payload
```json
{
  "pledgeId": "pledge-1",
  "shopId": "shop-1",
  "proofStatus": "verified",
  "proofHeadline": "Cam ket da duoc xac thuc",
  "proofSummary": "Du lieu hien tai trung khop voi ban ghi da duoc neo.",
  "recommendedActions": ["show_verified_badge"],
  "integrity": {
    "chainAnchorStatus": "anchored",
    "integrityStatus": "anchored",
    "onChainMatch": true
  }
}
```
