#!/usr/bin/env python3
"""Fill a running server with believable demo data.

Creates sellers with shops, products, freshness pledges (which the worker then
anchors on chain), buyer reviews and vouchers, so the app has something real to
show instead of one product called "E2E Fresh Produce".

Usage:
    python3 scripts/seed_demo.py [--base-url http://localhost:5050]

Buyer checks are deliberately not seeded: they need the vision provider, and
without OPENAI_API_KEY the endpoint returns "provider unavailable".
"""

import argparse
import json
import random
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timedelta, timezone

PASSWORD = "Passw0rd!"


class Api:
    def __init__(self, base_url):
        self.base_url = base_url.rstrip("/")
        self.token = None

    def _call(self, method, path, body=None, token=None):
        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode() if body is not None else None
        request = urllib.request.Request(url, data=data, method=method)
        request.add_header("Content-Type", "application/json")
        bearer = token if token is not None else self.token
        if bearer:
            request.add_header("Authorization", f"Bearer {bearer}")
        try:
            with urllib.request.urlopen(request, timeout=20) as response:
                raw = response.read().decode()
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as error:
            detail = error.read().decode()[:200]
            raise RuntimeError(f"{method} {path} -> {error.code}: {detail}") from None
        except urllib.error.URLError as error:
            raise RuntimeError(f"cannot reach {url}: {error.reason}") from None

    def get(self, path, token=None):
        return self._call("GET", path, token=token)

    def post(self, path, body, token=None):
        return self._call("POST", path, body, token=token)

    def register(self, email, display_name):
        auth = self.post(
            "/v1/auth/register",
            {"email": email, "password": PASSWORD, "displayName": display_name},
        )
        return auth["accessToken"]


# Real districts of Ho Chi Minh City so the map has something sensible to show.
SHOPS = [
    {
        "name": "Rau Sạch Cô Ba",
        "description": "Rau củ Đà Lạt giao trong ngày, có ghi nhận độ tươi từng lô.",
        "address": "Chợ Bến Thành, Quận 1, TP.HCM",
        "latitude": 10.7721,
        "longitude": 106.6980,
        "category": "vegetables",
        "tags": ["Đà Lạt", "giao trong ngày"],
        "products": [
            ("Cải ngọt Đà Lạt", 18000, 9.1, "Thu hoạch sáng nay, lá còn phấn."),
            ("Xà lách Romaine", 32000, 8.7, "Bảo quản lạnh 4°C từ vườn."),
            ("Cà chua bi hữu cơ", 45000, 8.9, "Giống cherry, độ ngọt cao."),
            ("Dưa leo baby", 25000, 8.4, "Vỏ mỏng, không hạt."),
        ],
        "pledges": 4,
        "reviews": [(5, "Rau tươi, quét mã ra đúng lô luôn."),
                    (5, "Mua 3 lần đều ổn định, giao nhanh."),
                    (4, "Chất lượng tốt, giá hơi cao một chút.")],
        "voucher": ("RAUSACH10", "Giảm 10% đơn rau củ", 10, True, 100000),
    },
    {
        "name": "Thịt Sạch Minh Anh",
        "description": "Thịt heo, bò nhập từ trang trại đạt chuẩn VietGAP.",
        "address": "234 Nguyễn Trãi, Quận 5, TP.HCM",
        "latitude": 10.7546,
        "longitude": 106.6634,
        "category": "meat",
        "tags": ["VietGAP", "mổ trong ngày"],
        "products": [
            ("Ba rọi heo rút sườn", 165000, 8.8, "Heo mổ trong ngày, cấp đông nhanh."),
            ("Thăn bò Úc nhập khẩu", 420000, 9.0, "Nhập nguyên khối, cắt theo yêu cầu."),
            ("Sườn non heo", 189000, 8.5, "Sườn mềm, tỉ lệ nạc mỡ cân đối."),
        ],
        "pledges": 3,
        "reviews": [(5, "Thịt tươi, có mã truy xuất rõ ràng."),
                    (4, "Đóng gói kỹ, sẽ quay lại.")],
        "voucher": ("THIT50K", "Giảm 50.000đ cho đơn từ 300k", 50000, False, 300000),
    },
    {
        "name": "Hải Sản Cần Giờ",
        "description": "Hải sản đánh bắt tại Cần Giờ, về chợ mỗi sáng.",
        "address": "12 Duyên Hải, Huyện Cần Giờ, TP.HCM",
        "latitude": 10.4114,
        "longitude": 106.9548,
        "category": "seafood",
        "tags": ["Cần Giờ", "đánh bắt tự nhiên"],
        "products": [
            ("Tôm sú tươi size 20", 380000, 8.6, "Tôm còn nhảy, bảo quản đá vảy."),
            ("Mực lá Cần Giờ", 295000, 8.2, "Mực dày mình, đánh bắt đêm qua."),
            ("Cá chẽm nguyên con", 178000, 8.0, "Cá 1.2-1.5kg, làm sạch tại chỗ."),
        ],
        "pledges": 2,
        "reviews": [(4, "Tôm chắc thịt, giao còn lạnh."),
                    (3, "Mực hơi nhỏ so với hình.")],
        "voucher": None,
    },
    {
        "name": "Trái Cây Nhà Vườn Cái Mơn",
        "description": "Trái cây miền Tây, hái tại vườn và chuyển thẳng lên TP.",
        "address": "56 Lê Văn Việt, TP. Thủ Đức, TP.HCM",
        "latitude": 10.8462,
        "longitude": 106.7803,
        "category": "fruit",
        "tags": ["miền Tây", "chín cây"],
        "products": [
            ("Sầu riêng Ri6", 145000, 9.2, "Cơm vàng hạt lép, chín cây."),
            ("Xoài cát Hòa Lộc", 89000, 8.8, "Trái 400-500g, ngọt đậm."),
            ("Chôm chôm nhãn", 62000, 8.3, "Hái sáng, cuống còn xanh."),
        ],
        "pledges": 3,
        "reviews": [(5, "Sầu riêng ngon đúng chuẩn Ri6.")],
        "voucher": ("TRAICAY15", "Giảm 15% trái cây theo mùa", 15, True, 150000),
    },
    {
        "name": "Nông Sản Hữu Cơ An Nhiên",
        "description": "Cửa hàng mới mở, đang xây dựng dữ liệu truy xuất.",
        "address": "88 Phan Xích Long, Quận Phú Nhuận, TP.HCM",
        "latitude": 10.7995,
        "longitude": 106.6822,
        "category": "fresh_produce",
        "tags": ["hữu cơ"],
        "products": [
            ("Gạo ST25 túi 5kg", 185000, 8.0, "Gạo vụ mới, đóng túi hút chân không."),
            ("Trứng gà thả vườn", 42000, 8.1, "Hộp 10 quả, gà nuôi thả."),
        ],
        # Deliberately none: gives the app a shop with no trust data, which is a
        # different state from a shop that scored badly.
        "pledges": 0,
        "reviews": [],
        "voucher": None,
    },
]


def log(message):
    print(message, flush=True)


def seed(api, suffix):
    created_shops = []

    for index, shop in enumerate(SHOPS, start=1):
        email = f"seller{index}.{suffix}@vngrocery.demo"
        token = api.register(email, shop["name"])
        log(f"\n[{index}/{len(SHOPS)}] {shop['name']}")

        created = api.post(
            "/v1/shops",
            {
                "name": shop["name"],
                "description": shop["description"],
                "address": shop["address"],
                "latitude": shop["latitude"],
                "longitude": shop["longitude"],
            },
            token=token,
        )
        shop_id = created["shopId"]
        log(f"    shop {shop_id}")

        product_ids = []
        for name, price, score, note in shop["products"]:
            product = api.post(
                f"/v1/shops/{shop_id}/products",
                {
                    "name": name,
                    "description": note,
                    "category": shop["category"],
                    "tags": shop["tags"],
                    "freshnessNote": note,
                    "freshnessScore": score,
                    "price": price,
                    "currency": "VND",
                    "status": "published",
                },
                token=token,
            )
            product_ids.append((product["productId"], score))
        log(f"    {len(product_ids)} sản phẩm")

        for product_id, score in product_ids[: shop["pledges"]]:
            api.post(
                "/v1/seller/commit",
                {
                    "shopId": shop_id,
                    "productId": product_id,
                    "bundleId": f"lot-{suffix}-{random.randint(1000, 9999)}",
                    # Slightly below the listed score: a pledge is what the
                    # seller measured, not the ideal figure on the listing.
                    "score": round(score - random.uniform(0, 0.4), 1),
                    "category": shop["category"],
                    "confidence": round(random.uniform(0.86, 0.97), 2),
                    "imageHash": f"seed-{suffix}-{product_id[:8]}",
                },
                token=token,
            )
        if shop["pledges"]:
            log(f"    {shop['pledges']} cam kết (đang neo lên blockchain)")

        voucher = shop["voucher"]
        if voucher:
            code, title, value, is_percent, min_spend = voucher
            expires = datetime.now(timezone.utc) + timedelta(days=45)
            api.post(
                f"/v1/shops/{shop_id}/vouchers",
                {
                    "code": code,
                    "title": title,
                    "discountValue": value,
                    "isPercent": is_percent,
                    "minSpend": min_spend,
                    "expiresAt": expires.isoformat().replace("+00:00", "Z"),
                    "note": "Áp dụng cho đơn mua tại quầy.",
                },
                token=token,
            )
            log(f"    voucher {code}")

        created_shops.append((shop_id, shop))

    # Reviews come from buyers, and a shop only accepts one per account, so each
    # review needs its own reviewer.
    review_count = 0
    reviewer_index = 0
    for shop_id, shop in created_shops:
        for rating, comment in shop["reviews"]:
            reviewer_index += 1
            token = api.register(
                f"buyer{reviewer_index}.{suffix}@vngrocery.demo",
                f"Khách hàng {reviewer_index}",
            )
            api.post(
                f"/v1/shops/{shop_id}/reviews",
                {"rating": rating, "comment": comment},
                token=token,
            )
            review_count += 1
    log(f"\n{review_count} đánh giá từ {reviewer_index} tài khoản người mua")

    return created_shops


def report(api, shops):
    log("\nChờ worker neo hash lên blockchain...")
    time.sleep(18)

    log("\nKết quả:")
    for shop_id, shop in shops:
        detail = api.get(f"/v1/shops/{shop_id}")
        trust = detail.get("trustSummary") or {}
        rating = detail.get("ratingSummary") or {}
        score = trust.get("score", 0)
        grade = trust.get("grade", "-")
        pledges = trust.get("pledgeCount", 0)
        stars = rating.get("averageRating", 0)
        reviews = rating.get("ratingCount", 0)
        log(
            f"  {shop['name']:<32} tin cậy {score:>5.1f} ({grade:<9}) "
            f"· {pledges} cam kết · {stars:.1f}★ / {reviews} đánh giá"
        )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://localhost:5050")
    args = parser.parse_args()

    api = Api(args.base_url)
    try:
        api.get("/health")
    except RuntimeError as error:
        print(f"Server không phản hồi: {error}", file=sys.stderr)
        return 1

    suffix = str(int(time.time()))[-6:]
    log(f"Seeding {args.base_url} (hậu tố tài khoản: {suffix})")

    try:
        shops = seed(api, suffix)
        report(api, shops)
    except RuntimeError as error:
        print(f"\nSeed thất bại: {error}", file=sys.stderr)
        return 1

    log("\nXong. Đăng nhập bằng bất kỳ tài khoản nào ở trên, mật khẩu: " + PASSWORD)
    return 0


if __name__ == "__main__":
    sys.exit(main())
