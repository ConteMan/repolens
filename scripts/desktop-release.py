"""Build, notarize and verify the macOS and Windows desktop artifacts and generate the desktop cask."""
import argparse
import base64
import hashlib
import json
import os
import plistlib
import re
import shutil
import stat
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
package = ROOT / "cmd/repolens-desktop"
config = json.loads((package / "mygo.json").read_text())
version = config["version"]
if not re.fullmatch(r"(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)", version):
    raise SystemExit("Expected a stable desktop version")
notes = (package / "CHANGELOG.md").read_text()
if re.search(r"^## (\S+)", notes, re.M)[1] != version:
    raise SystemExit("Version must match the first desktop changelog section")

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--verify-only", action="store_true", help="Verify and stage an existing local build")
output = ROOT / "bin/desktop-release" / version
if not parser.parse_args().verify_only:
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT):
        raise SystemExit("Commit the release sources before building desktop packages")
    output.mkdir(parents=True, exist_ok=True)
    (output / "commit.txt").unlink(missing_ok=True)
    key = Path(os.environ.get("REPOLENS_UPDATE_KEY_PATH",
                              Path.home() / "Library/Application Support/Repolens/update-keys/mygo-update.key"))
    if not key.is_file() or stat.S_IMODE(key.stat().st_mode) != 0o600:
        raise SystemExit("Expected a readable private updater key with mode 0600")
    subprocess.run(["go", "tool", "mygo", "build", "-platform", "darwin/universal,windows/amd64",
                    "-o", str(output), str(package)], cwd=ROOT, check=True,
                   env=dict(os.environ, MYGO_UPDATER_PRIVATE_KEY=key.read_text().strip()))
    (output / "commit.txt").write_text(commit + "\n")

stage = output / "darwin-universal"
app = stage / "Repolens.app"
archive = stage / f"repolens-{version}-darwin-universal.tar.gz"
manifest = stage / "update-darwin-universal.json"
dmg = stage / f"repolens-{version}.dmg"
tag = f"v{version}"
repo = "https://github.com/ConteMan/repolens"
download = f"{repo}/releases/download/{tag}/"

info = plistlib.loads((app / "Contents/Info.plist").read_bytes())
if info["CFBundleShortVersionString"] != version or info["CFBundleIdentifier"] != config["identifier"]:
    raise SystemExit("Unexpected application identity or version")
architectures = subprocess.check_output(["lipo", "-archs", app / "Contents/MacOS/Repolens"], text=True)
if set(architectures.split()) != {"arm64", "x86_64"}:
    raise SystemExit("Expected a Universal application")
subprocess.run(["codesign", "--verify", "--strict", "--deep", app], check=True)
subprocess.run(["spctl", "--assess", "--type", "execute", app], check=True)
for item in (app, dmg):
    subprocess.run(["xcrun", "stapler", "validate", item], check=True)

def sha256(path):
    return hashlib.sha256(path.read_bytes()).digest()


def verify_update(archive, manifest):
    update = json.loads(manifest.read_text())
    if (update["version"] != version or update["url"] != download + archive.name
            or update["size"] != archive.stat().st_size or not update["notes"]):
        raise SystemExit(f"Unexpected desktop update manifest: {manifest}")
    # MyGo signs the raw 32-byte SHA-256 digest with Ed25519.
    with tempfile.TemporaryDirectory() as tmp:
        tmp = Path(tmp)
        public_key = bytes.fromhex("302a300506032b6570032100") + base64.b64decode(config["updates"]["publicKey"])
        (tmp / "public.der").write_bytes(public_key)
        (tmp / "signature").write_bytes(base64.b64decode(update["signature"]))
        (tmp / "digest").write_bytes(sha256(archive))
        subprocess.run(["openssl", "pkeyutl", "-verify", "-pubin", "-rawin", "-keyform", "DER",
                        "-inkey", tmp / "public.der", "-sigfile", tmp / "signature",
                        "-in", tmp / "digest"], check=True)


win = output / "windows-amd64"
installer = win / f"repolens-setup-{version}-amd64.exe"
win_archive = win / f"repolens-{version}-windows-amd64.tar.gz"
win_manifest = win / "update-windows-amd64.json"
verify_update(archive, manifest)
verify_update(win_archive, win_manifest)

publish = output / "publish"
publish.mkdir(exist_ok=True)
artifacts = [dmg, archive, manifest, installer, win_archive, win_manifest, *sorted(stage.glob("*.delta")), *sorted(win.glob("*.delta"))]
expected = {p.name for p in artifacts} | {"SHA256SUMS"}
if any(p.name not in expected for p in publish.iterdir()):
    raise SystemExit(f"Unexpected files in {publish}; use a fresh output directory")
for artifact in artifacts:
    shutil.copy2(artifact, publish / artifact.name)
(publish / "SHA256SUMS").write_text("".join(f"{sha256(p).hex()}  {p.name}\n" for p in artifacts))
(output / "repolens-desktop.rb").write_text(f'''cask "repolens-desktop" do
  version "{version}"
  sha256 "{sha256(dmg).hex()}"

  url "{repo}/releases/download/v#{{version}}/repolens-#{{version}}.dmg"
  name "Repolens"
  desc "Build and preview documentation sites from Git repositories"
  homepage "{repo}"

  livecheck do
    url :url
    regex(/^v(\\d+(?:\\.\\d+)+)$/i)
    strategy :github_releases
  end

  auto_updates true
  depends_on macos: :ventura

  app "Repolens.app"
end
''')
print(f"Release assets: {publish}\nHomebrew cask: {output / 'repolens-desktop.rb'}")
