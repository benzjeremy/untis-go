#!/usr/bin/env bash
set -e

REPO_DIR="/home/benzj/Projekte/untis-go"
BUILD_DIR="/tmp/untis-builds"
WORK_DIR="/tmp/untis-work"

mkdir -p "$BUILD_DIR"
cd "$REPO_DIR"

build_tag() {
  local TAG="$1"
  local SKIP_LINUX="${2:-no}"
  local LINUX_OUT="$BUILD_DIR/untis-go-${TAG}-linux.tar.gz"
  local WIN_OUT="$BUILD_DIR/untis-go-${TAG}-windows.zip"

  echo ""
  echo "====== Building $TAG ======"

  rm -rf "$WORK_DIR"
  git worktree add "$WORK_DIR" "$TAG"
  cd "$WORK_DIR"

  # --- Linux ---
  if [[ "$SKIP_LINUX" == "yes" ]]; then
    echo "  [LINUX] Skipping (already built)"
  else
    echo "  [LINUX] Building..."
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
      go build -ldflags="-s -w" -o "$WORK_DIR/untis-go" . 2>&1
    tar -czf "$LINUX_OUT" -C "$WORK_DIR" untis-go
    echo "  [LINUX] OK -> $(du -sh $LINUX_OUT | cut -f1)"
  fi

  # --- Windows (pure Go, kein CGO) ---
  echo "  [WINDOWS] Cross-compiling..."
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -ldflags="-s -w" -o "$WORK_DIR/untis-go.exe" . 2>&1
  (cd "$WORK_DIR" && python3 -m zipfile -c "$WIN_OUT" untis-go.exe)
  echo "  [WINDOWS] OK -> $(du -sh $WIN_OUT | cut -f1)"

  cd "$REPO_DIR"
  git worktree remove --force "$WORK_DIR"

  # --- Upload ---
  echo "  [UPLOAD] Uploading to release $TAG ..."
  gh release upload "$TAG" "$LINUX_OUT" "$WIN_OUT" --clobber --repo benzjeremy/untis-go
  echo "  [UPLOAD] Done for $TAG"
}

# v1.6: Linux already built, only redo Windows + upload both
build_tag v1.6 yes
build_tag v2.0
build_tag v2.1

echo ""
echo "=== All done ==="
for tag in v1.6 v2.0 v2.1; do
  echo "$tag: $(gh release view $tag --json assets -q '[.assets[].name] | join(", ")' --repo benzjeremy/untis-go)"
done
