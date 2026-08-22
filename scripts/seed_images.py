#!/usr/bin/env python3
"""Gives every product a photo.

Downloads a freely licensed photo per product name from Wikimedia Commons,
pushes it into the project's own IPFS node through POST /v1/media/images, and
then updates the product through the normal seller API so the change is signed
and hash-chained like any other edit - the new image shows up in the product's
block-scan history rather than appearing from nowhere.

    ./scripts/vng seed-images              # everything that has no image yet
    ./scripts/vng seed-images --force      # replace images that are already set

Images are cached under scripts/.image-cache so a re-run does not re-download.

The URL written onto the product is whatever the API reports as the gateway
URL, which is IPFS_PUBLIC_GATEWAY_URL. That has to be an address the phone can
reach: a compose service name resolves inside the network only. See .env.
"""
import argparse
import json
import mimetypes
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

BASE = os.environ.get("VNG_API", "http://localhost:5050") + "/v1"
ADMIN_EMAIL = os.environ.get("VNG_ADMIN_EMAIL", "admin@admin.com")
ADMIN_PASSWORD = os.environ.get("VNG_ADMIN_PASSWORD", "admin")
SELLER_PASSWORD = os.environ.get("VNG_SELLER_PASSWORD", "Passw0rd!")
CACHE = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".image-cache")
UA = {"User-Agent": "VnGrocery-demo-seed/1.0 (university project)"}

# Vietnamese listing name -> the exact file on Wikimedia Commons.
#
# Pinned by name rather than searched at run time. Search ranking drifts, and it
# does not know what the listing is selling: asking for "potatoes" returned a
# sweet potato, "beef tenderloin" returned a plate of steak tartare, and both
# egg listings came back with the same photo of a cracked egg. Pinning also
# makes the credits below exact, which CC BY-SA obliges us to get right.
FILES = {
    "Ba rọi heo rút sườn": "File:Sliced pork belly - Bo Ssam Boiled Pork Belly - Sydney Madang Restaurant AUD27 small.jpg",
    "Bí đao xanh": "File:A winter melon, gourd , wax gourd , KyaukFaYonTee.jpg",
    "Bạch tuộc baby": "File:Fresh seafood display showcasing a variety of fish and octopus at a market undefined.jpg",
    "Cam vàng Úc": "File:Oranges - whole-halved-segment.jpg",
    "Chuối già hương": "File:Bunch of bananas on sale.jpg",
    "Chôm chôm nhãn": "File:Rambutans with seed.jpg",
    "Cà chua beef": "File:CDP IMG 4093.JPG",
    "Cà chua bi hữu cơ": "File:Yellow cherry tomatoes.jpg",
    "Cà rốt Đà Lạt": "File:Carrots of many colors.jpg",
    "Cá chẽm nguyên con": "File:Apahap.jpg",
    "Cá hồi Na Uy phi lê": "File:Raw salmon fillets.jpg",
    "Cải kale xoăn": "File:Olives attractively served in purple cabbage leaves.jpg",
    "Cải ngọt Đà Lạt": "File:Choy Sum with Soy Sauce.jpg",
    "Dưa leo baby": "File:Ryerson Market cucumbers on sale.jpg",
    "Gà ta nguyên con": "File:Whole raw chicken - Japan Dec 22 2019.jpeg",
    "Gạo ST25 túi 5kg": "File:Uncooked ST25 rice on bamboo surface.jpg",
    "Gạo tám Điện Biên": "File:JasmineRice.jpg",
    "Hành tây tím": "File:Red Onion on White.JPG",
    "Khoai tây Đà Lạt": "File:Patates.jpg",
    "Khổ qua rừng": "File:Ripe bitter melon (Momordica charantia).jpg",
    "Kiwi vàng Zespri": "File:Kiwifruit 'Gold' cross section.jpg",
    "Mồng tơi": "File:Basella alba leaves 27052014.jpg",
    "Mực lá Cần Giờ": "File:Squid and fish caught by local fisherman.jpg",
    "Nghêu Bến Tre": "File:Clams muscles shellfish food.jpg",
    "Nho mẫu đơn Hàn Quốc": "File:Shine muscat grapes (20240913).jpg",
    "Nạc dăm heo": "File:HK food ingredient red meat frozen pork chop raw butt steak October 2021 SS2 013.jpg",
    "Nấm hương khô Sa Pa": "File:Dried shiitake mushrooms (20240907).jpg",
    "Rau chân vịt": "File:Spinach leaves.jpg",
    "Rau muống nước": "File:N Ipoa D1600.JPG",
    "Súp lơ xanh": "File:Broccoli and cross section edit.jpg",
    "Sườn non heo": "File:Raw pork spareribs.jpg",
    "Sầu riêng Ri6": "File:Durian Fruit in Yunnan.jpg",
    "Thăn bò Úc nhập khẩu": "File:20241123 Rindsuppe 004 (54250401408).jpg",
    "Trứng gà thả vườn": "File:Eggs in basket 2020 G1.jpg",
    "Trứng vịt Đồng Tháp": "File:Mangkuk telur bebek asin.jpg",
    "Táo Envy New Zealand": "File:Red Apple.jpg",
    "Tôm sú tươi size 20": "File:Penaeus monodon.jpg",
    "Xoài cát Hòa Lộc": "File:Mangos - single and halved.jpg",
    "Xà lách Romaine": "File:Romaine lettuce.jpg",
    "Xương ống heo": "File:Raw Pork Meat in Bloody Butchery of Corpses.jpg",
}


def slug(name):
    out = [c if (c.isascii() and c.isalnum()) else "-" for c in name.lower()]
    return re.sub(r"-+", "-", "".join(out)).strip("-")


# ------------------------------------------------------------------ commons
def fetch(url, timeout=30):
    """GET with a slow retry.

    Commons rate-limits anonymous callers and answers 429 when it has had
    enough. Retrying immediately just earns another 429, so back off; a seed
    run is not in a hurry.
    """
    delay = 5
    for attempt in range(5):
        try:
            return urllib.request.urlopen(urllib.request.Request(url, headers=UA), timeout=timeout).read()
        except urllib.error.HTTPError as exc:
            if exc.code not in (429, 503) or attempt == 4:
                raise
            wait = int(exc.headers.get("Retry-After", delay))
            print("    Commons asked us to slow down, waiting %ds" % wait)
            time.sleep(wait)
            delay = min(delay * 2, 60)
    raise RuntimeError("gave up on " + url)


def commons(params):
    url = "https://commons.wikimedia.org/w/api.php?" + urllib.parse.urlencode(params)
    return json.loads(fetch(url).decode())


def file_info(title):
    """Everything we need about one pinned Commons file."""
    result = commons({
        "action": "query", "format": "json", "titles": title,
        "prop": "imageinfo", "iiprop": "url|mime|extmetadata", "iiurlwidth": 800,
    })
    pages = list(((result.get("query") or {}).get("pages") or {}).values())
    if not pages or "missing" in pages[0]:
        return None
    info = (pages[0].get("imageinfo") or [{}])[0]
    if not info.get("thumburl"):
        return None
    meta = info.get("extmetadata") or {}
    return {
        "title": pages[0]["title"],
        "url": info["thumburl"],
        "licence": (meta.get("LicenseShortName") or {}).get("value", "unknown"),
        "author": re.sub("<[^>]+>", "", (meta.get("Artist") or {}).get("value", "unknown")).strip()[:80],
        "page": "https://commons.wikimedia.org/wiki/" + urllib.parse.quote(pages[0]["title"].replace(" ", "_")),
    }


def photo_for(name):
    """Bytes of the photo for a product, downloading it only once."""
    os.makedirs(CACHE, exist_ok=True)
    path = os.path.join(CACHE, slug(name) + ".jpg")
    if os.path.exists(path) and os.path.getsize(path) > 20000:
        return open(path, "rb").read()

    title = FILES.get(name)
    if not title:
        return None
    hit = file_info(title)
    if not hit:
        print("    %s is no longer on Commons" % title)
        return None
    data = fetch(hit["url"], timeout=60)
    with open(path, "wb") as fh:
        fh.write(data)
    # Most of these are CC BY or CC BY-SA, which oblige us to say where each
    # one came from and who took it. Recorded as they arrive so the credits
    # cannot drift out of step with the files actually in use.
    record_credit(name, hit)
    print("    downloaded %s (%d KB) from %s" % (slug(name), len(data) // 1024, hit["title"]))
    time.sleep(0.4)
    return data


CREDITS = os.path.join(CACHE, "CREDITS.json")


def record_credit(name, hit):
    credits = {}
    if os.path.exists(CREDITS):
        try:
            credits = json.load(open(CREDITS))
        except ValueError:
            credits = {}
    credits[name] = {k: hit[k] for k in ("title", "licence", "author", "page")}
    with open(CREDITS, "w") as fh:
        json.dump(credits, fh, ensure_ascii=False, indent=2, sort_keys=True)


# ---------------------------------------------------------------------- api
def call(method, path, token=None, body=None, raw=None, content_type=None):
    data, headers = None, {}
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    if raw is not None:
        data, headers["Content-Type"] = raw, content_type
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    for _ in range(6):
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                text = resp.read().decode()
                return json.loads(text) if text else {}
        except urllib.error.HTTPError as exc:
            if exc.code == 429:
                # The API rate-limits; it says how long to wait, so wait.
                time.sleep(int(exc.headers.get("Retry-After", "5")) + 1)
                continue
            raise RuntimeError("%s %s -> %d %s" % (method, path, exc.code, exc.read().decode()[:200]))
    raise RuntimeError("%s %s kept hitting the rate limit" % (method, path))


def multipart(filename, data):
    boundary = "----vngrocery" + os.urandom(8).hex()
    mime = mimetypes.guess_type(filename)[0] or "application/octet-stream"
    body = b"".join([
        ("--%s\r\n" % boundary).encode(),
        ('Content-Disposition: form-data; name="image"; filename="%s"\r\n' % filename).encode(),
        ("Content-Type: %s\r\n\r\n" % mime).encode(),
        data,
        ("\r\n--%s--\r\n" % boundary).encode(),
    ])
    return body, "multipart/form-data; boundary=" + boundary


def login(email, password):
    return call("POST", "/auth/login", body={"email": email, "password": password})["accessToken"]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--force", action="store_true",
                        help="replace images on products that already have one")
    args = parser.parse_args()

    try:
        admin = login(ADMIN_EMAIL, ADMIN_PASSWORD)
    except Exception as exc:
        print("Could not sign in as %s: %s" % (ADMIN_EMAIL, exc))
        print("Uploading needs an account; set VNG_ADMIN_EMAIL / VNG_ADMIN_PASSWORD.")
        return 1

    on_ipfs = {}
    tokens = {}
    attached = skipped = failed = 0

    for shop in call("GET", "/shops?limit=100")["items"]:
        products = call("GET", "/shops/%s/products?limit=100" % shop["shopId"])["items"]
        pending = [p for p in products if args.force or not p.get("imageUrls")]
        if not pending:
            skipped += len(products)
            continue

        # Each product is edited by the shop that owns it, so the signed event
        # names the seller rather than an administrator.
        owner = shop.get("ownerEmail") or seller_email(shop["shopId"])
        if not owner:
            print("  no owner login for %s, skipping" % shop["name"])
            failed += len(pending)
            continue
        if owner not in tokens:
            try:
                tokens[owner] = login(owner, SELLER_PASSWORD)
            except Exception as exc:
                print("  cannot sign in as %s: %s" % (owner, exc))
                failed += len(pending)
                continue

        print("%s" % shop["name"])
        for product in pending:
            name = product["name"]
            if name not in on_ipfs:
                data = photo_for(name)
                if not data:
                    print("    no photo for %s" % name)
                    failed += 1
                    continue
                body, content_type = multipart(slug(name) + ".jpg", data)
                uploaded = call("POST", "/media/images", token=admin, raw=body, content_type=content_type)
                on_ipfs[name] = uploaded.get("gatewayUrl") or ""
                print("    ipfs %s %s" % (uploaded.get("imageCid", "?")[:20], name))
            url = on_ipfs.get(name)
            if not url:
                failed += 1
                continue
            try:
                call("PUT", "/shops/%s/products/%s" % (shop["shopId"], product["productId"]),
                     token=tokens[owner], body={
                         "expectedVersion": product["version"],
                         "name": name,
                         "description": product.get("description", ""),
                         "category": product.get("category", ""),
                         "tags": product.get("tags") or [],
                         "imageUrls": [url],
                         "freshnessNote": product.get("freshnessNote", ""),
                         "freshnessScore": product.get("freshnessScore", 0),
                         "price": product.get("price", 0),
                         "currency": product.get("currency") or "VND",
                         "status": product.get("status") or "published",
                     })
                attached += 1
            except RuntimeError as exc:
                print("    FAILED %s: %s" % (name, exc))
                failed += 1

    print("\n%d products given a photo, %d already had one, %d failed" % (attached, skipped, failed))
    return 1 if failed else 0


_owner_cache = {}


def seller_email(shop_id):
    """Owner login for a shop, read from the seeded accounts."""
    if not _owner_cache:
        try:
            admin = login(ADMIN_EMAIL, ADMIN_PASSWORD)
            for user in call("GET", "/admin/users?limit=500", token=admin).get("items", []):
                _owner_cache[user.get("userId")] = user.get("email")
        except Exception:
            return None
    shop = call("GET", "/shops/%s" % shop_id)
    return _owner_cache.get(shop.get("ownerUserId"))


if __name__ == "__main__":
    sys.exit(main())
