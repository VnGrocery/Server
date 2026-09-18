#!/usr/bin/env python3
"""Draw the QR codes a pledge can carry.

There are two, and they do different jobs:

  --label   the lot code, as printed on a crate. Never expires, anyone can
            scan it, and it resolves to what the seller pledged.
  (default) a real bundle token, as the seller's screen shows it. Expires and
            is consumed by the first check, which is what a recorded buyer
            check needs.

The token is the genuine one the server issues to a seller, not a stand-in: the
app parses its claims and the server verifies its signature, so a faked payload
would be rejected at the first check and prove nothing.

Two consequences of that worth knowing before you use the default mode:

  * the token expires (30 minutes by default), so regenerate rather than
    keeping an old PNG around
  * it is single use - one successful check consumes it, and scanning the same
    image again is a replay

    ./scripts/make_bundle_qr.py                   # newest pledge in the demo shop
    ./scripts/make_bundle_qr.py --pledge <id>
    ./scripts/make_bundle_qr.py --all             # one PNG per pledge in the shop
    ./scripts/make_bundle_qr.py --label --all     # the printed labels instead

To scan it on the Android emulator, hang it in the virtual scene:

    emulator -avd <avd> -virtualscene-poster wall=<the png>

Two things that are easy to lose an hour to:

  * the AVD needs `hw.camera.back = virtualscene` in its config.ini. The
    default `emulated` draws a fixed colour-block pattern and silently ignores
    posters, so the flag looks broken rather than unused.
  * the scene camera does not start facing that wall, and nothing on the
    console moves it - not `sensor set`, not `physics`. Drag with the left
    mouse button inside the emulator window to turn around until the QR is in
    view. `wall` and `table` are the only poster names the stock scene has.

Scanning the PNG off the host screen with a real phone works too, and skips
all of the above.
"""

import argparse
import json
import pathlib
import sys
import urllib.error
import urllib.request

try:
    import qrcode
except ImportError:
    sys.exit("qrcode is missing. Install it with: pip3 install qrcode pillow")

API = "http://localhost:5050"
PASSWORD = "Passw0rd!"

# The shop the rest of the demo scripts use, so the QR points at a product that
# actually has content on its page.
DEFAULT_SHOP = "1b291ade-cc48-4295-b642-e0e3213d1fb2"
DEFAULT_EMAIL = "seller7.262007@vngrocery.demo"

OUT_DIR = pathlib.Path(__file__).parent / ".qr"


def call(method, path, token=None, body=None):
    request = urllib.request.Request(
        f"{API}{path}",
        method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={
            "Content-Type": "application/json",
            **({"Authorization": f"Bearer {token}"} if token else {}),
        },
    )
    try:
        with urllib.request.urlopen(request) as response:
            return json.loads(response.read() or "{}")
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")
        sys.exit(f"{method} {path} failed: {error.code} {detail}")
    except urllib.error.URLError as error:
        sys.exit(f"{method} {path} could not reach {API}: {error.reason}")


def write_label_qr(pledge, out_dir):
    """The QR that goes on a printed crate label: just the lot code.

    Tiny next to the token QR - a lot code is 18 characters, so this comes out
    around version 2 instead of 18 - which is why it scans off paper, off a
    curved crate and off a screen at arm's length.
    """
    code = qrcode.QRCode(
        # High, because this one gets printed and then knocked about.
        error_correction=qrcode.constants.ERROR_CORRECT_H,
        box_size=14,
        border=4,
    )
    code.add_data(pledge["bundleId"])
    code.make(fit=True)
    image = code.make_image().convert("RGB")
    out_dir.mkdir(parents=True, exist_ok=True)
    path = out_dir / f"label-{pledge['bundleId']}.png"
    image.save(path)
    return path, code.version


def write_qr(token, pledge, out_dir):
    # A bundle token is ~670 characters, which is a dense QR however it is
    # drawn. Two things keep it readable off a screen:
    #
    #   * the lowest error correction, which drops it from 101 modules square
    #     to 89 - nothing here is printed on a crate or smudged, so the
    #     redundancy buys less than the density costs
    #   * a large box size, because the usual reason a QR will not scan is that
    #     the camera cannot resolve individual modules
    code = qrcode.QRCode(
        error_correction=qrcode.constants.ERROR_CORRECT_L,
        box_size=14,
        border=4,
    )
    code.add_data(token)
    code.make(fit=True)
    # Saved as RGB: the 1-bit PNG Pillow writes by default reads back as an
    # empty image in some decoders.
    image = code.make_image().convert("RGB")
    out_dir.mkdir(parents=True, exist_ok=True)
    path = out_dir / f"bundle-{pledge['bundleId']}.png"
    image.save(path)
    return path, code.version


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--shop", default=DEFAULT_SHOP)
    parser.add_argument("--email", default=DEFAULT_EMAIL)
    parser.add_argument("--password", default=PASSWORD)
    parser.add_argument("--pledge", help="Pledge id; default is the newest one")
    parser.add_argument("--all", action="store_true", help="One QR per pledge")
    parser.add_argument(
        "--label",
        action="store_true",
        help="The printed-label QR: the lot code, no token, never expires",
    )
    parser.add_argument("--out", default=str(OUT_DIR))
    args = parser.parse_args()

    session = call(
        "POST",
        "/v1/auth/login",
        body={"email": args.email, "password": args.password},
    )
    access = session.get("accessToken")
    if not access:
        sys.exit(f"login for {args.email} returned no access token")

    listing = call("GET", f"/v1/shops/{args.shop}/pledges", token=access)
    pledges = listing.get("items") or []
    if not pledges:
        sys.exit(f"shop {args.shop} has no pledges to build a QR from")

    if args.pledge:
        pledges = [p for p in pledges if p.get("pledgeId") == args.pledge]
        if not pledges:
            sys.exit(f"pledge {args.pledge} is not in shop {args.shop}")
    elif not args.all:
        pledges = pledges[:1]

    out_dir = pathlib.Path(args.out)
    for pledge in pledges:
        if args.label:
            path, version = write_label_qr(pledge, out_dir)
            print(f"{path}")
            print(f"  bundle    {pledge['bundleId']}")
            print(f"  product   {pledge.get('productId', '')}")
            print(f"  qr        version {version}, never expires")
            continue
        issued = call(
            "POST",
            f"/v1/shops/{args.shop}/pledges/{pledge['pledgeId']}/bundle-token",
            token=access,
        )
        token = issued.get("bundleToken")
        if not token:
            sys.exit(f"pledge {pledge['pledgeId']} returned no bundle token")
        path, version = write_qr(token, pledge, out_dir)
        print(f"{path}")
        print(f"  bundle    {pledge['bundleId']}")
        print(f"  product   {pledge.get('productId', '')}")
        print(f"  expires   {issued.get('bundleTokenExpiresAt', '')}")
        print(f"  qr        version {version}, single use")


if __name__ == "__main__":
    main()
