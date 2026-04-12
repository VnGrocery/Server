# 02a. MongoDB Or Firestore

## Quy tắc chọn backend lưu trữ

Repo này hiện hỗ trợ 2 backend:
- `MongoDB`
- `Firestore`

Rule thực tế:
- nếu `MONGODB_ENABLED=true` thì dùng `MongoDB`
- nếu cả `MONGODB_ENABLED=true` và `FIREBASE_ENABLED=true` thì vẫn ưu tiên `MongoDB`
- nếu bạn không chỉnh gì, mặc định vẫn là `MongoDB`
- chỉ dùng `Firestore` khi `MONGODB_ENABLED=false`

## Khuyến nghị

### Local mới, dễ nhất
Dùng MongoDB:

```dotenv
MONGODB_ENABLED=true
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=vngrocery
FIREBASE_ENABLED=false
```

Lúc này:
- không cần Firebase credentials
- không cần Firestore project

### Nếu muốn dùng Firestore
Dùng:

```dotenv
MONGODB_ENABLED=false
FIREBASE_ENABLED=true
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS_FILE=./secrets/firebase-service-account.json
```

Lúc này:
- backend sẽ bỏ MongoDB
- backend dùng Firestore như trước

## Ví dụ `.env` tối thiểu cho MongoDB local

```dotenv
PORT=5000
MONGODB_ENABLED=true
MONGODB_URI=mongodb://127.0.0.1:27017
MONGODB_DATABASE=vngrocery
FIREBASE_ENABLED=false
JWT_SECRET=replace-with-a-long-random-secret
OPENAI_API_KEY=your-openai-api-key
```

## Ví dụ `.env` tối thiểu cho Firestore

```dotenv
PORT=5000
MONGODB_ENABLED=false
FIREBASE_ENABLED=true
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS_FILE=./secrets/firebase-service-account.json
JWT_SECRET=replace-with-a-long-random-secret
OPENAI_API_KEY=your-openai-api-key
```

## Lưu ý quan trọng

- `backfill-integrity` hiện vẫn chỉ hỗ trợ Firestore
- nếu bạn đang dùng MongoDB, lệnh này sẽ báo rõ và dừng
- runtime `server` đã hỗ trợ MongoDB cho các repository chính
