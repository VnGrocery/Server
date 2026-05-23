# Role Permission Update Plan

## 0. Quy tac thuc hien

- Trien khai theo tung luong nho, moi luong phai co code, test lien quan va commit rieng.
- Sau khi xong moi luong, cap nhat trang thai trong file ke hoach nay truoc khi commit.
- Khong gop nhieu luong lon vao mot commit neu co the tach rieng theo backend, Android UI, migration hoac docs.
- Khong revert cac thay doi co san cua nguoi dung trong worktree.
- Neu mot luong khong test duoc day du, ghi ro ly do va test da chay trong commit/final summary.

## Muc tieu

Don gian hoa phan quyen ve 2 role chinh:

- `admin`: quan tri he thong, moderation, quan ly user, xem du lieu van hanh.
- `user`: tai khoan nguoi dung thong thuong. Mot user co the mua hang, review, kiem tra chat luong, va cung co the so huu shop de thuc hien nghiep vu seller.

Huong nay phu hop hon voi ung dung co cung mot tai khoan nhung sau nay co the chuyen qua lai giua giao dien buyer va seller. Role khong nen bi tach thanh `buyer` va `seller` vi buyer/seller luc nay la **context nghiep vu** hoac **UI mode**, khong phai quyen he thong co dinh.

## Nguyen tac thiet ke

1. `role` chi dung de phan biet quyen he thong: `admin` va `user`.
2. Buyer/seller khong phai role trong DB.
3. Seller capability duoc xac dinh bang viec user co so huu shop hay khong.
4. Buyer capability la mac dinh cua moi active `user`.
5. Admin khong nen mac dinh di qua luong seller/buyer; admin nen dung cac endpoint admin rieng.
6. Ownership check van la lop bao ve quan trong nhat cho shop/product/batch/pledge.
7. Status user phai duoc kiem tra tren moi protected route de suspend/delete co hieu luc som.

## Role model de xuat

| Role | Y nghia | Ghi chu |
| --- | --- | --- |
| `admin` | Quan tri he thong | Chi duoc tao qua bootstrap hoac admin action |
| `user` | Tai khoan nguoi dung | Co the la buyer, seller, hoac ca hai tuy theo du lieu so huu shop |

Khong can `buyer` va `seller` trong `users.role`.

Neu sau nay can onboarding chi tiet hon, nen them field rieng thay vi them role:

- `sellerProfile.status`: `none`, `pending`, `active`, `suspended`
- `buyerProfile.status`: neu that su can, mac dinh co the la `active`
- `activeMode` tren frontend/local preference: `buyer` hoac `seller`

## Role luu o dau

Role tiep tuc luu o user profile:

```go
domain.User.Role
```

Trong DB:

```json
{
  "userId": "user-1",
  "email": "user@example.com",
  "role": "user",
  "status": "active"
}
```

Khong nen luu role lam source of truth trong JWT. JWT co the chua role de hien thi nhanh, nhung middleware quan trong nen doc lai user tu DB khi can kiem tra quyen.

## Capability model

### Buyer capability

Moi active `user` deu co buyer capability:

- buyer check
- review shop
- report freshness
- xem lich su check cua chinh minh neu co endpoint nay sau nay

Khong can role `buyer`.

### Seller capability

Seller capability den tu resource ownership:

- User tao shop thi tro thanh owner cua shop do.
- User co shop active/pending thi frontend co the hien thi seller mode.
- Product, batch, trace event, pledge phai check `OwnerUserID`.
- Neu shop bi suspended/deleted, seller actions tren shop do bi chan.

Khong can role `seller`.

### Admin capability

Admin capability den tu `role == admin`:

- quan ly user
- moderate shop/product/buyer check/report
- xem admin list/report
- rotate/recover/backfill account key
- reanchor/revoke integrity

## Route permission matrix

### Public routes

| Route group | Access |
| --- | --- |
| `GET /health`, `GET /v1/health` | Public |
| `GET /docs`, `GET /openapi.json` | Public hoac environment-gated tren production |
| `POST /v1/auth/register`, `login`, `google`, `refresh`, password reset | Public |
| Public shop/product/batch/trace/review/proof reads | Public, chi tra active/published resource |

### Shared authenticated routes

| Route group | Required checks |
| --- | --- |
| `/v1/me`, change password, delete account | Active authenticated user |
| `POST /v1/media/images` | Active authenticated user, sau nay co the them upload context |
| `GET /v1/events` | Can xem lai muc dich. Neu chua can public user audit thi nen admin-only hoac filter theo actor |

### User as buyer

| Route group | Required checks |
| --- | --- |
| `POST /v1/buyer/check` | Active `user`, valid bundle token, quota |
| `POST /v1/shops/:shopId/reviews` | Active `user` |
| `DELETE /v1/shops/:shopId/reviews/me` | Active `user`, own review |
| `POST /v1/shops/:shopId/products/:productId/freshness-reports` | Active `user`, quota |

### User as seller

| Route group | Required checks |
| --- | --- |
| `POST /v1/shops` | Active `user`; optional business validation |
| `PUT/DELETE /v1/shops/:shopId` | Active `user`, shop owner |
| Product create/update/delete/bulk | Active `user`, shop owner |
| Batch create/update/delete | Active `user`, product/shop owner |
| Trace event create | Active `user`, batch/shop owner |
| `POST /v1/seller/score` | Active `user`; ideally require a shop context later |
| `POST /v1/seller/commit` | Active `user`, shop owner |
| Reissue bundle token | Active `user`, shop owner |

### Admin

| Route group | Required checks |
| --- | --- |
| `/v1/admin/**` | Active authenticated user, `role == admin` |
| User role/status/key management | `admin`, plus last-admin protection |
| Moderation endpoints | `admin` |
| Admin reports/lists | `admin` |

## Backend implementation plan

### 1. Chuan hoa role constants

Tao constants dung chung:

```go
const (
    RoleAdmin = "admin"
    RoleUser  = "user"

    UserStatusActive    = "active"
    UserStatusSuspended = "suspended"
    UserStatusDeleted   = "deleted"
)
```

Dung constants nay trong auth, middleware, user admin, seed va tests.

### 2. Giu register mac dinh la user

Dang ky moi:

- Bootstrap admin email -> `admin`
- Con lai -> `user`

Khong can client gui `role` khi register. Neu co gui thi backend nen ignore hoac reject de tranh tu dang ky admin.

### 3. Strengthen auth middleware

`AuthRequired` nen:

- verify JWT
- load current user tu `UserRepository`
- reject user missing/suspended/deleted
- set `Principal` va `domain.User` vao request context

Ly do: khi admin suspend/delete user, access token cu khong nen tiep tuc goi API den khi het han.

### 4. Giu admin middleware rieng

`AdminRequired` nen doc current user tu context, fallback doc DB neu can, va check:

```go
user.Role == "admin"
user.Status == "active"
```

Khong nen tin role trong JWT lam source of truth.

### 5. Khong them buyer/seller middleware

Khong can:

- `RequireSeller`
- `RequireBuyer`
- role `seller`
- role `buyer`

Thay vao do:

- Buyer actions chi can active user + business checks.
- Seller actions can active user + ownership checks.

### 6. Lam ro seller ownership

Seller endpoints hien tai nen tiep tuc check:

- shop owner
- product owner
- batch owner
- pledge belongs to owned shop

Can review va bo sung test cho cac case:

- user A khong sua duoc shop cua user B
- user A khong tao product trong shop cua user B
- user A khong commit pledge cho shop cua user B
- user A khong tao trace event cho batch cua user B

### 7. Them seller profile neu can

Neu can quy trinh duyet seller, khong nen bien seller thanh role. Nen them model rieng:

```go
type SellerProfile struct {
    UserID string
    Status string // pending, active, suspended
    BusinessName string
    TaxID string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Luc do seller actions can:

- active `user`
- `sellerProfile.status == active`
- ownership

Neu chua can duyet seller, co the bo qua seller profile trong giai do dau.

### 8. Frontend mode switching

Frontend co the quyet dinh mode:

- Buyer mode: mac dinh cho moi user.
- Seller mode: hien thi khi user co shop hoac bam "Create shop".

De ho tro UI, backend nen co endpoint:

```http
GET /v1/me/capabilities
```

Response de xuat:

```json
{
  "role": "user",
  "canUseBuyerMode": true,
  "canUseSellerMode": true,
  "ownedShopCount": 1,
  "ownedShopIds": ["shop-1"],
  "admin": false
}
```

Voi admin:

```json
{
  "role": "admin",
  "canUseBuyerMode": false,
  "canUseSellerMode": false,
  "admin": true
}
```

Co the cho admin test UI buyer/seller sau nay bang impersonation/debug mode rieng, nhung khong nen tron vao role model chinh.

## Data migration

Vi hien tai seed/data co the co `seller` va `buyer`, migration nen map ve `user`:

| Old role | New role |
| --- | --- |
| `admin` | `admin` |
| `seller` | `user` |
| `buyer` | `user` |
| `user` | `user` |
| empty/unknown | `user` hoac manual review |

Seller identity sau migration duoc suy ra tu shop ownership, khong tu role.

Nen chay migration co dry-run:

- dem so user theo old role
- dem so seller-role user dang own shop
- dem so buyer-role user co buyer checks/reviews
- report unknown role
- apply map ve `admin/user`

## Testing plan

### Auth/middleware

- Missing token -> 401
- Invalid token -> 401
- Active user -> pass
- Suspended/deleted user -> 403
- JWT role khac DB role -> dung DB role
- Admin route chi admin active vao duoc
- User route active user vao duoc

### Admin user management

- Admin co the doi role `user <-> admin`
- Non-admin khong doi duoc role/status
- Khong duoc demote/suspend/delete active admin cuoi cung
- Role khong hop le bi reject

### Seller ownership

- User tao shop thanh cong
- User khong tao/sua/xoa product trong shop cua user khac
- User khong sua/xoa shop cua user khac
- User khong commit pledge cho shop cua user khac
- User khong tao trace event cho batch cua user khac

### Buyer actions

- Active user tao buyer check duoc neu token hop le
- Active user tao review duoc
- Active user tao freshness report duoc
- Suspended/deleted user bi chan

## Rollout plan

### Phase 1: Chuan hoa concept

Trang thai: Done - da them role/status constants, cap nhat cac check admin dung constants, va map seed data `seller/buyer` ve `user`.

- Cap nhat docs va team convention: chi co `admin/user`.
- Khong dung `seller/buyer` trong `users.role` nua.
- Cap nhat seed/test data moi theo `admin/user`.

### Phase 2: Middleware active user

- Auth middleware load user tu DB.
- Protected route reject suspended/deleted user.
- Admin middleware dung current user tu context.

### Phase 3: Admin safety

- Admin role update chi cho `admin/user`.
- Them last-active-admin protection.

### Phase 4: Ownership hardening

- Audit seller endpoints.
- Bo sung test ownership.
- Dam bao shop/product/batch/pledge khong the cross-owner mutation.

### Phase 5: Capabilities endpoint

- Them `/v1/me/capabilities`.
- Frontend dung endpoint nay de hien buyer/seller/admin mode.

### Phase 6: Migration

- Dry-run role migration.
- Map `seller/buyer` ve `user`.
- Verify user co shop van vao seller UI duoc qua capabilities.

## Ket luan

Huong hop ly hon la:

- Role he thong: `admin`, `user`
- Buyer/seller: UI mode va business capability
- Seller access: active user + shop ownership
- Buyer access: active user
- Admin access: admin role

Cach nay tranh viec mot nguoi vua mua vua ban bi ket trong mot role, va giup sau nay tach giao dien seller/buyer ma khong phai doi lai auth model.
