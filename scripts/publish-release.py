"""Stage CLI and signed desktop assets together, then optionally publish the verified draft."""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("bundle", type=Path, help="Extracted release workflow artifact")
parser.add_argument("notes", type=Path, help="Release notes")
parser.add_argument("--publish", action="store_true", help="Publish the draft after verification")
args = parser.parse_args()
root = Path(__file__).resolve().parents[1]
version = json.loads((root / "cmd/repolens-desktop/mygo.json").read_text())["version"]
tag = f"v{version}"
repo = "ConteMan/repolens"
sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
tag_sha = subprocess.check_output(["git", "rev-parse", f"{tag}^{{commit}}"], cwd=root, text=True).strip()
if sha != tag_sha or (args.bundle / "commit.txt").read_text().strip() != sha:
    raise SystemExit("CLI bundle, checkout and release tag must identify the same commit")
desktop_dir = root / "bin/desktop-release" / version
if (desktop_dir / "commit.txt").read_text().strip() != sha:
    raise SystemExit("Build the desktop packages from the tagged commit")
desktop = desktop_dir / "publish"
cli = args.bundle / "cli"
subprocess.run(["python3", root / "scripts/desktop-release.py", "--verify-only"], check=True)
expected_cli = {f"repolens_{version}_{os}_{arch}.{ext}"
                for os, ext in [("darwin", "tar.gz"), ("linux", "tar.gz"), ("windows", "zip")]
                for arch in ["amd64", "arm64"]}
if {p.name for p in cli.iterdir()} != expected_cli | {"checksums.txt"}:
    raise SystemExit("The CLI bundle must contain the six release archives and checksums.txt")
for directory, checksum in [(cli, "checksums.txt"), (desktop, "SHA256SUMS")]:
    subprocess.run(["shasum", "-a", "256", "-c", checksum], cwd=directory, check=True)
assets = sorted(cli.iterdir()) + sorted(desktop.iterdir())
local = {p.name: "sha256:" + hashlib.sha256(p.read_bytes()).hexdigest() for p in assets}


def gh(*arguments):
    return subprocess.check_output(["gh", *arguments, "--repo", repo], text=True)


# List both drafts and published releases to avoid treating a failed lookup as permission to overwrite.
releases = json.loads(gh("release", "list", "--limit", "100", "--json", "tagName,isDraft"))
existing = next((r for r in releases if r["tagName"] == tag), None)
if existing and not existing["isDraft"]:
    raise SystemExit(f"{tag} is already public")
if not existing:
    gh("release", "create", tag, "--verify-tag", "--draft", "--title", f"Repolens {version}",
       "--notes-file", str(args.notes))


def remote_assets():
    return {asset["name"]: asset["digest"] for asset in json.loads(
        gh("release", "view", tag, "--json", "assets"))["assets"]}


uploaded = remote_assets()
if set(uploaded) - set(local):
    raise SystemExit("The existing draft contains unexpected assets")
pending = [str(p) for p in assets if uploaded.get(p.name) != local[p.name]]
if pending:
    gh("release", "upload", tag, *pending, "--clobber")
if remote_assets() != local:
    raise SystemExit("Uploaded files do not match the verified release assets")
if args.publish:
    gh("release", "edit", tag, "--draft=false", "--latest")
print(f"{tag}: {len(local)} verified assets, {'published' if args.publish else 'draft'}")
