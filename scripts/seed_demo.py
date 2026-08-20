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
    # The set below spreads across the city so that anywhere in central Ho Chi
    # Minh City has something inside the 5 km ring. A demo run from one office
    # should not depend on standing next to one particular market.
    {
        "name": "Rau Củ Sạch Bình Thạnh",
        "description": "Rau củ theo mùa, nhập từ hợp tác xã Củ Chi mỗi sáng.",
        "address": "215 Điện Biên Phủ, Quận Bình Thạnh, TP.HCM",
        "latitude": 10.8010,
        "longitude": 106.7110,
        "category": "vegetables",
        "tags": ["Củ Chi", "theo mùa"],
        "products": [
            ("Rau muống nước", 12000, 8.6, "Cọng nhỏ, hái sáng cùng ngày."),
            ("Bí đao xanh", 21000, 8.4, "Trái non, vỏ còn phấn."),
            ("Mồng tơi", 11000, 8.5, "Lá dày, không dập."),
            ("Khổ qua rừng", 38000, 8.2, "Trái nhỏ, vị đắng thanh."),
        ],
        "pledges": 3,
        "reviews": [(5, "Rau tươi, giá mềm hơn siêu thị."),
                    (4, "Giao đúng giờ, đóng gói gọn.")],
        "voucher": ("RAUBT12", "Giảm 12% đơn rau củ", 12, True, 80000),
    },
    {
        "name": "Thủy Hải Sản Tân Bình",
        "description": "Hải sản đông lạnh và tươi sống, có tem truy xuất từng lô.",
        "address": "42 Cộng Hòa, Quận Tân Bình, TP.HCM",
        "latitude": 10.8015,
        "longitude": 106.6520,
        "category": "seafood",
        "tags": ["cấp đông nhanh", "có tem lô"],
        "products": [
            ("Cá hồi Na Uy phi lê", 520000, 8.9, "Nhập nguyên con, cắt theo yêu cầu."),
            ("Bạch tuộc baby", 245000, 8.3, "Làm sạch, cấp đông -40°C."),
            ("Nghêu Bến Tre", 48000, 8.1, "Ngâm nhả cát 6 tiếng."),
        ],
        "pledges": 2,
        "reviews": [(5, "Cá hồi tươi, quét mã ra đúng ngày nhập."),
                    (4, "Nghêu sạch cát, nấu ngọt nước.")],
        "voucher": None,
    },
    {
        "name": "Trái Cây Nhập Khẩu Quận 7",
        "description": "Trái cây nhập khẩu, bảo quản lạnh từ kho tới quầy.",
        "address": "1058 Nguyễn Văn Linh, Quận 7, TP.HCM",
        "latitude": 10.7290,
        "longitude": 106.7180,
        "category": "fruit",
        "tags": ["nhập khẩu", "bảo quản lạnh"],
        "products": [
            ("Táo Envy New Zealand", 165000, 9.0, "Size 70-80, giòn ngọt."),
            ("Nho mẫu đơn Hàn Quốc", 480000, 8.8, "Hộp 1kg, còn nguyên phấn."),
            ("Kiwi vàng Zespri", 128000, 8.5, "Chín tới, ăn được ngay."),
            ("Cam vàng Úc", 89000, 8.4, "Nhiều nước, ít hạt."),
        ],
        "pledges": 3,
        "reviews": [(5, "Nho ngon, đóng hộp cẩn thận."),
                    (5, "Táo giòn đúng như mô tả."),
                    (4, "Giá cao nhưng chất lượng ổn định.")],
        "voucher": ("NHAPKHAU20", "Giảm 20.000đ đơn từ 200k", 20000, False, 200000),
    },
    {
        "name": "Thịt Tươi Gò Vấp",
        "description": "Thịt heo và gà ta, giết mổ tập trung có kiểm dịch.",
        "address": "88 Quang Trung, Quận Gò Vấp, TP.HCM",
        "latitude": 10.8380,
        "longitude": 106.6650,
        "category": "meat",
        "tags": ["có kiểm dịch", "mổ trong ngày"],
        "products": [
            ("Gà ta nguyên con", 145000, 8.7, "Gà thả vườn, 1.5-1.8kg."),
            ("Nạc dăm heo", 138000, 8.5, "Thớ mềm, ít mỡ."),
            ("Xương ống heo", 62000, 8.0, "Ninh nước dùng ngọt."),
        ],
        "pledges": 2,
        "reviews": [(4, "Gà chắc thịt, luộc lên thơm.")],
        "voucher": None,
    },
    {
        "name": "Nông Sản Thủ Đức",
        "description": "Nông sản Lâm Đồng về thẳng kho Thủ Đức mỗi đêm.",
        "address": "12 Võ Văn Ngân, TP. Thủ Đức, TP.HCM",
        "latitude": 10.8500,
        "longitude": 106.7570,
        "category": "fresh_produce",
        "tags": ["Lâm Đồng", "về đêm"],
        "products": [
            ("Khoai tây Đà Lạt", 32000, 8.6, "Củ đều, vỏ mỏng."),
            ("Cà rốt Đà Lạt", 28000, 8.5, "Ngọt, không xơ."),
            ("Hành tây tím", 35000, 8.3, "Vỏ khô, không mọc mầm."),
            ("Súp lơ xanh", 45000, 8.7, "Bông chặt, cuống non."),
        ],
        "pledges": 4,
        "reviews": [(5, "Khoai tây bở, nấu canh ngon."),
                    (4, "Rau về đêm nên sáng mua rất tươi.")],
        "voucher": ("THUDUC10", "Giảm 10% nông sản Đà Lạt", 10, True, 100000),
    },
    {
        "name": "Chợ Quê Quận 10",
        "description": "Đặc sản vùng miền: gạo, trứng, đồ khô có nguồn gốc rõ.",
        "address": "301 Sư Vạn Hạnh, Quận 10, TP.HCM",
        "latitude": 10.7720,
        "longitude": 106.6680,
        "category": "fresh_produce",
        "tags": ["đặc sản vùng miền"],
        "products": [
            ("Gạo tám Điện Biên", 210000, 8.4, "Túi 5kg, vụ mới."),
            ("Trứng vịt Đồng Tháp", 48000, 8.2, "Hộp 10 quả, vịt chạy đồng."),
            ("Nấm hương khô Sa Pa", 155000, 8.6, "Cánh dày, thơm đậm."),
        ],
        "pledges": 2,
        "reviews": [(4, "Gạo dẻo, nấu lên thơm."),
                    (5, "Nấm hương thơm, không bị mốc.")],
        "voucher": None,
    },
    {
        "name": "Rau Hữu Cơ Quận 3",
        "description": "Rau hữu cơ trồng nhà kính, có chứng nhận từng luống.",
        "address": "175 Võ Văn Tần, Quận 3, TP.HCM",
        "latitude": 10.7780,
        "longitude": 106.6870,
        "category": "vegetables",
        "tags": ["hữu cơ", "nhà kính"],
        "products": [
            ("Cải kale xoăn", 68000, 9.0, "Lá non, trồng nhà kính."),
            ("Rau chân vịt", 55000, 8.8, "Cắt gốc, rửa sẵn."),
            ("Cà chua beef", 72000, 8.7, "Trái to, thịt dày."),
        ],
        "pledges": 3,
        "reviews": [(5, "Rau sạch thật, ăn sống được."),
                    (5, "Có chứng nhận rõ ràng, yên tâm."),
                    (4, "Giá cao nhưng xứng đáng.")],
        "voucher": ("HUUCO15", "Giảm 15% rau hữu cơ", 15, True, 120000),
    },
]


def log(message):
    print(message, flush=True)


def seed(api, suffix, only=None):
    created_shops = []
    wanted = [shop for shop in SHOPS if only is None or shop["name"] in only]

    for index, shop in enumerate(wanted, start=1):
        email = f"seller{index}.{suffix}@vngrocery.demo"
        token = api.register(email, shop["name"])
        log(f"\n[{index}/{len(wanted)}] {shop['name']}")

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
    parser.add_argument(
        "--only",
        help=(
            "Comma-separated shop names to seed. Without it every shop is "
            "seeded, which duplicates any already on the server -- use this to "
            "top up an existing demo database instead of doubling it."
        ),
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="Print the shop names this script can seed, then exit.",
    )
    args = parser.parse_args()

    if args.list:
        for shop in SHOPS:
            log(shop["name"])
        return 0

    only = None
    if args.only:
        only = {name.strip() for name in args.only.split(",") if name.strip()}
        unknown = only - {shop["name"] for shop in SHOPS}
        if unknown:
            print(f"Không có cửa hàng: {', '.join(sorted(unknown))}", file=sys.stderr)
            return 1

    api = Api(args.base_url)
    try:
        api.get("/health")
    except RuntimeError as error:
        print(f"Server không phản hồi: {error}", file=sys.stderr)
        return 1

    suffix = str(int(time.time()))[-6:]
    log(f"Seeding {args.base_url} (hậu tố tài khoản: {suffix})")

    try:
        shops = seed(api, suffix, only)
        report(api, shops)
    except RuntimeError as error:
        print(f"\nSeed thất bại: {error}", file=sys.stderr)
        return 1

    log("\nXong. Đăng nhập bằng bất kỳ tài khoản nào ở trên, mật khẩu: " + PASSWORD)
    return 0


if __name__ == "__main__":
    sys.exit(main())
