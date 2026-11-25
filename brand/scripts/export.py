"""Export the folio master and palette samples. Requires Python 3 and ImageMagick 7."""
import argparse
import hashlib
import json
import subprocess
import tempfile
from pathlib import Path
from xml.etree import ElementTree as ET

BRAND = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("output", type=Path, help="Destination for generated assets")
parser.add_argument("--palette", choices=["moss", "plum", "ochre"], help="Export one palette")
parser.add_argument("--check", action="store_true", help="Check committed assets against a fresh export")
args = parser.parse_args()
scratch = tempfile.TemporaryDirectory() if args.check else None
destination = Path(scratch.name) if scratch else args.output
doc = json.loads((BRAND / "repolens-logo.pen").read_text())
master, = [n for n in doc["children"] if n.get("reusable") and n["name"].startswith("LG-01")]
micro, = [n for n in doc["children"] if n.get("ref") == master["id"] and n["name"].startswith("LG-02")]
colors = {k: v["value"] for k, v in doc["variables"].items()}
canvas = f'0 0 {master["width"]} {master["height"]}'


def symbol(color, small=False):
    group = ET.Element("g", fill=color)
    for part in master["children"]:
        if small:
            part = {**part, **micro["descendants"].get(part["id"], {})}
        if part["type"] != "path" or part.get("rotation") or part.get("flipX") or part.get("flipY"):
            raise ValueError(f'Unsupported master layer or transform: {part["name"]}')
        if not part.get("enabled", True):
            continue
        vx, vy, vw, vh = part["viewBox"]
        transform = (f'translate({part["x"]} {part["y"]}) '
                     f'scale({part["width"] / vw} {part["height"] / vh}) translate({-vx} {-vy})')
        ET.SubElement(group, "path", d=part["geometry"], transform=transform, fill=color,
                      **{"fill-rule": part.get("fillRule", "nonzero")})
    return ET.tostring(group, encoding="unicode")


def svg(body, size, viewbox):
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="{viewbox}">{body}</svg>\n'


def raster(source, target, size):
    subprocess.run(["magick", "-background", "none", str(source), "-resize", f"{size}x{size}",
                    "-strip", str(target)], check=True)


for key in sorted(k.removesuffix("-mark") for k in colors if k.endswith("-mark")):
    if args.palette and key != args.palette:
        continue
    out = destination / key
    out.mkdir(parents=True, exist_ok=True)
    mark = colors[key + "-mark"]
    light = colors[key + "-light"]
    paper = colors[key + "-paper"]
    for filename, color, small in [("mark", mark, False), ("dark", light, False), ("mono", "#202321", False),
                                    ("favicon", mark, True)]:
        (out / (filename + ".svg")).write_text(svg(symbol(color, small), 1024, canvas))
    icon = f'<rect x="10" y="10" width="80" height="80" rx="18" fill="{paper}"/>'
    icon += f'<g transform="translate(17 17) scale({66/master["width"]} {66/master["height"]})">{symbol(mark)}</g>'
    (out / "desktop.svg").write_text(svg(icon, 1024, "0 0 100 100"))
    for size in (128, 1024):
        raster(out / "desktop.svg", out / f"desktop-{size}.png", size)
    for size in (16, 32):
        raster(out / "favicon.svg", out / f"favicon-{size}.png", size)
    raster(out / "mark.svg", out / "mark-32.png", 32)

    files = sorted(path for path in out.iterdir() if path.suffix in {".svg", ".png"})
    manifest = "".join(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}\n" for path in files)
    (out / "SHA256SUMS").write_text(manifest)
    if args.check:
        expected = args.output / key
        mismatches = [path.name for path in out.iterdir()
                      if not (expected / path.name).is_file() or path.read_bytes() != (expected / path.name).read_bytes()]
        if mismatches:
            raise SystemExit(f"{key}: missing or modified assets: {', '.join(sorted(mismatches))}")
