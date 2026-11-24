#!/bin/sh
# Builds the GoDownload browser extension for each target browser.
#
#   ./build.sh            build chrome and firefox
#   ./build.sh chrome     build one target
#
# Output: dist/<target>/ (unpacked, loadable) and dist/godownload-<target>-<version>.zip
set -eu

ROOT=$(cd "$(dirname "$0")" && pwd)
SRC="$ROOT/src"
ICONS="$ROOT/icons"
DIST="$ROOT/dist"
ICON_SIZES="16 32 48 128"

log() { printf '%s\n' "$*"; }
warn() { printf 'WARNING: %s\n' "$*" >&2; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

# Reads "version" from a manifest without needing jq.
manifest_version() {
  sed -n 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$1" | head -n 1
}

make_zip() {
  # $1 = directory to pack, $2 = absolute output path
  rm -f "$2"
  if command -v zip >/dev/null 2>&1; then
    (cd "$1" && zip -q -r -X "$2" .)
  elif command -v python3 >/dev/null 2>&1; then
    # -m zipfile -c would store the directory name as a prefix; archive the contents instead.
    (cd "$1" && python3 -c 'import os, sys, zipfile
with zipfile.ZipFile(sys.argv[1], "w", zipfile.ZIP_DEFLATED) as z:
    for base, dirs, files in os.walk("."):
        dirs.sort()
        for name in sorted(files):
            path = os.path.join(base, name)
            z.write(path, os.path.relpath(path, "."))' "$2")
  else
    warn "neither zip nor python3 found; skipping $2"
    return 0
  fi
  log "  packed $(basename "$2")"
}

build_target() {
  target=$1
  manifest="$ROOT/manifests/$target.json"
  [ -f "$manifest" ] || die "unknown target '$target' (no $manifest)"
  out="$DIST/$target"

  log "Building $target..."
  rm -rf "$out"
  mkdir -p "$out"
  cp -R "$SRC/." "$out/"
  cp "$manifest" "$out/manifest.json"

  mkdir -p "$out/icons"
  missing=""
  for size in $ICON_SIZES; do
    icon="$ICONS/icon-$size.png"
    if [ -f "$icon" ]; then
      cp "$icon" "$out/icons/"
    else
      missing="$missing icon-$size.png"
    fi
  done
  if [ -n "$missing" ]; then
    warn "missing icons in $ICONS:$missing"
    # Chrome refuses to load an extension whose manifest names a missing icon,
    # so drop the icon keys from this build to keep it loadable for development.
    if command -v python3 >/dev/null 2>&1; then
      python3 -c 'import json, sys
p = sys.argv[1]
m = json.load(open(p))
m.pop("icons", None)
m.get("action", {}).pop("default_icon", None)
json.dump(m, open(p, "w"), indent=2)
open(p, "a").write("\n")' "$out/manifest.json"
      warn "removed icon references from $target/manifest.json; the browser will show a default icon"
    else
      warn "python3 not found; Chrome will refuse to load this build until the icons exist"
    fi
  fi

  # Never ship editor/OS junk.
  find "$out" \( -name '.DS_Store' -o -name '*~' -o -name '*.swp' \) -exec rm -f {} +

  version=$(manifest_version "$manifest")
  [ -n "$version" ] || version="dev"
  make_zip "$out" "$DIST/godownload-$target-$version.zip"
  log "  unpacked: $out"
}

[ -d "$SRC" ] || die "source directory $SRC not found"
mkdir -p "$DIST"

if [ "$#" -eq 0 ]; then
  set -- chrome firefox
fi
for t in "$@"; do
  build_target "$t"
done
log "Done."
