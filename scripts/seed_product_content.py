#!/usr/bin/env python3
"""Fill a shop's products with specification rows and a long-form description.

Demo content for the product detail page: the seeded products carry a single
flat sentence, which leaves the spec table and the description blocks empty on
every screen built to show them.

What it writes is seller copy - where the goods come from, how to keep them,
how to prepare them. It does not invent certifications, ratings, freshness
scores or anything else a buyer is meant to be able to verify; those belong to
the pledge and report records, not to text a seller types.

Each write goes through the API, so every change is signed and lands in the
product's change log like any other edit.

    ./scripts/seed_product_content.py                       # default shop
    ./scripts/seed_product_content.py --shop <shopId> --email <seller>
"""

import argparse
import json
import sys
import urllib.error
import urllib.request

API = "http://localhost:5050"
PASSWORD = "Passw0rd!"

# The shop the demo walks through.
DEFAULT_SHOP = "1b291ade-cc48-4295-b642-e0e3213d1fb2"
DEFAULT_EMAIL = "seller7.262007@vngrocery.demo"

REASON = "Bổ sung thông số và mô tả chi tiết cho sản phẩm"

# Keyed by a word in the product name. Keeping the copy per-product rather than
# generated from a template is the point: a demo of a description feature reads
# as filler the moment every product says the same thing.
CONTENT = {
    "cải kale": {
        "specs": [
            ("Xuất xứ", "Đà Lạt, Lâm Đồng"),
            ("Trọng lượng", "300 g / bó"),
            ("Bảo quản", "Ngăn mát 2-5 °C, bọc túi giấy, không rửa trước khi cất"),
            ("Hạn dùng", "5 ngày kể từ ngày hái"),
            ("Quy cách", "Bó buộc dây lạt, lót giấy thấm"),
        ],
        "blocks": [
            ("heading", "Điểm nổi bật"),
            ("paragraph", "Kale xoăn trồng ở vùng cao Đà Lạt, lá dày và giòn hơn kale trồng đồng bằng. Hái vào sáng sớm rồi chuyển thẳng về quầy trong ngày."),
            ("bullets", ["Lá dày, cuống giòn, không bị dai", "Hái sáng cùng ngày", "Rửa sạch đất trước khi giao"]),
            ("heading", "Sơ chế & bảo quản"),
            ("paragraph", "Tước bỏ phần cuống già ở gốc lá. Vò nhẹ với chút dầu ô liu trước khi trộn salad để lá mềm và bớt hăng. Nếu xào thì cho vào sau cùng, đảo nhanh 1-2 phút."),
            ("paragraph", "Không rửa trước khi cất tủ lạnh. Bọc túi giấy hoặc khăn khô rồi để ngăn mát, dùng trong 5 ngày."),
        ],
    },
    "rau chân vịt": {
        "specs": [
            ("Xuất xứ", "Đà Lạt, Lâm Đồng"),
            ("Trọng lượng", "400 g / bó"),
            ("Bảo quản", "Ngăn mát 2-5 °C, để nguyên bó, không ngâm nước"),
            ("Hạn dùng", "3 ngày kể từ ngày hái"),
            ("Quy cách", "Bó có rễ, giữ nguyên gốc cho lá tươi lâu"),
        ],
        "blocks": [
            ("heading", "Điểm nổi bật"),
            ("paragraph", "Rau chân vịt còn nguyên rễ, cách giữ lá tươi lâu nhất mà không cần hoá chất. Cắt rễ ngay trước khi nấu."),
            ("bullets", ["Còn nguyên rễ, lá không héo rũ", "Lá non, cuống mảnh, ít xơ", "Hái sáng cùng ngày"]),
            ("heading", "Sơ chế & bảo quản"),
            ("paragraph", "Rửa dưới vòi nước chảy để trôi hết cát ở kẽ lá, không ngâm lâu vì rau sẽ nhũn. Luộc hoặc xào nhanh trong 1 phút, để lâu lá mất màu và mất vị ngọt."),
            ("paragraph", "Lá này mỏng nên xuống nhanh hơn các loại rau khác: dùng trong 3 ngày, để nguyên bó trong ngăn mát."),
        ],
    },
    "cà chua beef": {
        "specs": [
            ("Xuất xứ", "Đà Lạt, Lâm Đồng"),
            ("Trọng lượng", "800 g - 1 kg / trái"),
            ("Bảo quản", "Nơi thoáng mát 12-15 °C, chỉ cho tủ lạnh khi đã cắt"),
            ("Hạn dùng", "10 ngày nếu để nguyên trái"),
            ("Quy cách", "Từng trái lót lưới xốp, xếp một lớp trong thùng"),
        ],
        "blocks": [
            ("heading", "Điểm nổi bật"),
            ("paragraph", "Cà chua beef trái to, thịt dày và ít hạt, cắt lát không bị chảy nước nên hợp làm burger, salad hoặc áp chảo."),
            ("bullets", ["Trái to 800 g - 1 kg", "Thịt dày, ít hạt, ít nước", "Hái khi vừa chín tới, không ủ ép"]),
            ("heading", "Hướng dẫn sử dụng"),
            ("paragraph", "Cắt lát dày 1 cm để kẹp bánh mì hoặc burger. Nếu nấu sốt thì trụng qua nước sôi 30 giây rồi bóc vỏ, thịt cà sẽ tan mịn hơn."),
            ("heading", "Lưu ý & bảo quản"),
            ("paragraph", "Để nguyên trái ở nơi thoáng mát, không cho vào tủ lạnh khi chưa cắt: lạnh làm thịt cà bở và mất vị. Trái đã cắt thì bọc kín, để ngăn mát và dùng trong 2 ngày."),
        ],
    },
    "trứng gà": {
        "specs": [
            ("Xuất xứ", "Trại gà thả vườn, Củ Chi, TP.HCM"),
            ("Quy cách", "Hộp giấy 10 quả, có vách ngăn từng quả"),
            ("Trọng lượng", "55-60 g / quả"),
            ("Bảo quản", "Ngăn mát 4-8 °C, để đầu to hướng lên"),
            ("Hạn dùng", "21 ngày kể từ ngày thu"),
        ],
        "blocks": [
            ("heading", "Điểm nổi bật"),
            ("paragraph", "Trứng từ gà thả vườn, lòng đỏ đậm màu và dẻo hơn trứng gà nuôi lồng. Thu mỗi sáng, không rửa để giữ lớp màng bảo vệ tự nhiên trên vỏ."),
            ("bullets", ["Gà thả vườn, ăn thóc và rau", "Thu trứng mỗi sáng", "Không rửa vỏ, giữ màng bảo vệ tự nhiên"]),
            ("heading", "Lưu ý & bảo quản"),
            ("paragraph", "Chỉ rửa ngay trước khi dùng. Rửa sớm làm mất lớp màng trên vỏ, vi khuẩn dễ thấm qua các lỗ khí và trứng hỏng nhanh hơn nhiều."),
            ("paragraph", "Xếp đầu to hướng lên để lòng đỏ nằm giữa. Để ngăn mát, tránh cánh cửa tủ vì nhiệt độ ở đó thay đổi mỗi lần mở."),
        ],
    },
    "rau muống": {
        "specs": [
            ("Xuất xứ", "Ruộng nước Hóc Môn, TP.HCM"),
            ("Trọng lượng", "500 g / bó"),
            ("Bảo quản", "Ngăn mát 5-8 °C, bọc khăn ẩm"),
            ("Hạn dùng", "2 ngày kể từ ngày hái"),
            ("Quy cách", "Bó buộc lạt, đã nhặt bỏ lá già"),
        ],
        "blocks": [
            ("heading", "Điểm nổi bật"),
            ("paragraph", "Rau muống nước cọng to và giòn, hái buổi sáng và bán trong ngày. Đã nhặt sẵn lá già và gốc cứng."),
            ("bullets", ["Cọng giòn, không bị xơ", "Đã nhặt sẵn lá già", "Hái và bán trong ngày"]),
            ("heading", "Sơ chế & bảo quản"),
            ("paragraph", "Ngâm nước muối loãng 5 phút rồi rửa lại, vớt ra để ráo. Luộc thì cho vào lúc nước sôi mạnh và mở vung để rau giữ màu xanh."),
            ("paragraph", "Rau muống xuống rất nhanh, nên mua vừa đủ ăn trong hai ngày. Bọc khăn ẩm rồi để ngăn mát nếu chưa nấu ngay."),
        ],
    },
    "chuối": {
        "specs": [
            ("Xuất xứ", "Vườn chuối Đồng Nai"),
            ("Trọng lượng", "1,2 - 1,5 kg / nải"),
            ("Bảo quản", "Nhiệt độ phòng, treo nơi thoáng, không cho tủ lạnh khi còn xanh"),
            ("Hạn dùng", "4-6 ngày tuỳ độ chín khi nhận"),
            ("Quy cách", "Nguyên nải, cắt sát cuống và bọc cuống"),
        ],
        "blocks": [
            ("heading", "Điểm nổi bật"),
            ("paragraph", "Chuối già hương cắt lúc còn ương rồi để chín tự nhiên, nên vị ngọt thanh và thơm hơn chuối giấm thuốc. Nhận về thường còn hơi xanh ở cuống."),
            ("bullets", ["Chín tự nhiên, không giấm thuốc", "Nguyên nải, cuống được bọc để chín đều", "Cắt tại vườn Đồng Nai"]),
            ("heading", "Hướng dẫn sử dụng"),
            ("paragraph", "Treo nải ở nơi thoáng cho chín dần. Muốn chín nhanh thì để cạnh quả táo hoặc quả bơ chín; muốn chậm lại thì tách rời từng quả."),
            ("heading", "Lưu ý & bảo quản"),
            ("paragraph", "Không cho vào tủ lạnh khi chuối còn xanh, vỏ sẽ thâm đen mà ruột vẫn sượng. Chuối đã chín thì cất ngăn mát được thêm 2 ngày, vỏ thâm nhưng ruột vẫn ngon."),
        ],
    },
}


def call(method, path, body=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    request = urllib.request.Request(
        API + path,
        method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers=headers,
    )
    with urllib.request.urlopen(request) as response:
        return json.load(response)


def content_for(name):
    lowered = name.lower()
    for key, value in CONTENT.items():
        if key in lowered:
            return value
    return None


def to_blocks(pairs):
    blocks = []
    for kind, value in pairs:
        if kind == "bullets":
            blocks.append({"type": "bullets", "items": value})
        else:
            blocks.append({"type": kind, "text": value})
    return blocks


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--shop", default=DEFAULT_SHOP)
    parser.add_argument("--email", default=DEFAULT_EMAIL)
    parser.add_argument(
        "--force",
        action="store_true",
        help="rewrite products that already carry specs",
    )
    args = parser.parse_args()

    try:
        token = call(
            "POST", "/v1/auth/login", {"email": args.email, "password": PASSWORD}
        )["accessToken"]
    except urllib.error.URLError as error:
        sys.exit(f"Không đăng nhập được ({error}). Stack đã chạy chưa: ./scripts/vng up")

    products = call("GET", f"/v1/shops/{args.shop}/products")["items"]
    written = skipped = 0

    for product in products:
        content = content_for(product["name"])
        if content is None:
            print(f"  bỏ qua {product['name']}: chưa có nội dung soạn sẵn")
            skipped += 1
            continue
        if product.get("specs") and not args.force:
            print(f"  bỏ qua {product['name']}: đã có thông số (--force để ghi đè)")
            skipped += 1
            continue

        # Read-modify-write on the whole record: an update replaces it, so
        # anything omitted here - the photo, the freshness score - is erased
        # from the signed product.
        payload = {
            "changeReason": REASON,
            "expectedVersion": product["version"],
            "name": product["name"],
            "description": product["description"],
            "category": product["category"],
            "tags": product.get("tags") or [],
            "imageUrls": product.get("imageUrls") or [],
            "freshnessNote": product.get("freshnessNote") or "",
            "freshnessScore": product.get("freshnessScore") or 0,
            "price": product["price"],
            "currency": product.get("currency") or "VND",
            "status": product["status"],
            "specs": [{"key": k, "value": v} for k, v in content["specs"]],
            "descBlocks": to_blocks(content["blocks"]),
        }
        saved = call(
            "PUT",
            f"/v1/shops/{args.shop}/products/{product['productId']}",
            payload,
            token,
        )
        print(
            f"  {saved['name']}: v{saved['version']}, "
            f"{len(saved.get('specs') or [])} thông số, "
            f"{len(saved.get('descBlocks') or [])} khối mô tả"
        )
        written += 1

    print(f"\nXong. Ghi {written} sản phẩm, bỏ qua {skipped}.")
    if written:
        print("Mỗi thay đổi đã được ký và nằm trong lịch sử thay đổi của sản phẩm.")


if __name__ == "__main__":
    main()
