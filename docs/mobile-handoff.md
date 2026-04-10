# Mobile Handoff

## Core screens
- `Auth`: `POST /v1/auth/register`, `POST /v1/auth/login`, `POST /v1/auth/google`, `POST /v1/auth/refresh`
- `Shop list/detail`: `GET /v1/shops`, `GET /v1/shops/:shopId`
- `Product list/detail`: `GET /v1/shops/:shopId/products`, `GET /v1/shops/:shopId/products/:productId`
- `Seller pledge flow`: `POST /v1/media/images`, `POST /v1/seller/score`, `POST /v1/seller/commit`
- `Buyer check flow`: `POST /v1/media/images`, `POST /v1/buyer/check`
- `Proof view`: `GET /v1/shops/:shopId/pledges/:pledgeId/proof`

## Suggested mobile flow
1. Upload image with `POST /v1/media/images`.
2. Reuse returned `imageHash`, `imageCid`, `gatewayUrl` in seller commit, buyer check, or freshness report.
3. Show trust summary from shop detail. Treat `formulaVersion=trust_score_v2` as the current score contract.
4. For pledge proof, prefer `proofStatus`, `proofHeadline`, `proofSummary`, and `recommendedActions`. Do not expose Besu/IPFS jargon by default.

## UI mapping
- `proofStatus=verified`: show green verified state.
- `proofStatus=pending`: show neutral "dang dong bo" state.
- `proofStatus=warning`: show warning banner and mismatch copy.
- `proofStatus=revoked`: show revoked badge and reduce trust emphasis.

## Sample proof payload
```json
{
  "pledgeId": "pledge-1",
  "shopId": "shop-1",
  "proofStatus": "verified",
  "proofHeadline": "Cam ket da duoc xac thuc",
  "proofSummary": "Hash du lieu trong co so du lieu trung khop voi ban ghi da duoc neo len blockchain.",
  "recommendedActions": ["show_verified_badge"],
  "integrity": {
    "chainAnchorStatus": "anchored",
    "integrityStatus": "anchored",
    "onChainMatch": true
  }
}
```

## Frontend rules
- Always call `POST /v1/media/images` before sending image-based actions from mobile.
- Persist `imageCid` and `gatewayUrl` locally if the UI needs preview/retry.
- Treat `reasons` arrays as diagnostics, not primary user copy.
