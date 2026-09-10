#!/usr/bin/env bash
set -e

REPO_DIR="/home/benzj/Projekte/untis-go"
BUILD_DIR="/tmp/untis-go-builds"
WORK_DIR="/tmp/untis-go-work"

# All release tags in order
TAGS=(v1.0 v1.1 v1.2 v1.3 v1.3.1 v1.4 v1.5 v1.5.1 v1.5.2 v1.6 v2.0 v2.1)

mkdir -p "$BUILD_DIR"

cd "$REPO_DIR"

# Save current branch to restore later
CURRENT=$(git rev-parse --abbrev-ref HEAD)

for TAG in "${TAGS[@]}"; do
  echo ""
  echo "======================================"
  echo "  Building $TAG"
  echo "======================================"

  # Friendly version without 'v' prefix for filenames
  VER="${TAG#v}"

  LINUX_OUT="$BUILD_DIR/untis-go-${TAG}-linux.tar.gz"
  WIN_OUT="$BUILD_DIR/untis-go-${TAG}-windows.zip"

  # Skip if both already exist
  if [[ -f "$LINUX_OUT" && -f "$WIN_OUT" ]]; then
    echo "  [SKIP] Artifacts already exist for $TAG"
    continue
  fi

  # Checkout this exact tag into a temp worktree
  rm -rf "$WORK_DIR"
  git worktree add "$WORK_DIR" "$TAG" 2>/dev/null || {
    echo "  [WARN] worktree add failed, trying checkout"
    git worktree remove --force "$WORK_DIR" 2>/dev/null || true
    git worktree add "$WORK_DIR" "$TAG"
  }

  cd "$WORK_DIR"

  # ---- Linux build (CGO for v2.0+ with GTK, pure Go for older) ----
  LINUX_BINARY="$WORK_DIR/untis-go-linux"
  if grep -q "gtk\|webkit\|webkitgtk" go.mod 2>/dev/null || \
     grep -q "go:build linux && cgo" gui_linux.go 2>/dev/null; then
    echo "  [LINUX] CGO build (GTK/WebKit detected)"
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
      go build -ldflags="-s -w" -o "$LINUX_BINARY" . 2>&1
  else
    echo "  [LINUX] Pure Go build"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags="-s -w" -o "$LINUX_BINARY" . 2>&1
  fi

  # Package Linux
  tar -czf "$LINUX_OUT" -C "$WORK_DIR" untis-go-linux --transform "s|untis-go-linux|untis-go|"
  echo "  [LINUX] -> $LINUX_OUT"

  # ---- Windows build (always pure Go, no GTK dependency) ----
  WIN_BINARY="$WORK_DIR/untis-go.exe"
  echo "  [WINDOWS] Cross-compile (CGO_ENABLED=0)"
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -ldflags="-s -w" -o "$WIN_BINARY" . 2>&1

  # Package Windows
  (cd "$WORK_DIR" && zip -q "$WIN_OUT" untis-go.exe)
  echo "  [WINDOWS] -> $WIN_OUT"

  # Cleanup worktree
  cd "$REPO_DIR"
  git worktree remove --force "$WORK_DIR" 2>/dev/null || true

  echo "  [OK] $TAG done"
done

# Restore original branch
git checkout "$CURRENT" 2>/dev/null || true

echo ""
echo "======================================"
echo "  All builds complete. Uploading..."
echo "======================================"

for TAG in "${TAGS[@]}"; do
  LINUX_OUT="$BUILD_DIR/untis-go-${TAG}-linux.tar.gz"
  WIN_OUT="$BUILD_DIR/untis-go-${TAG}-windows.zip"

  if [[ ! -f "$LINUX_OUT" && ! -f "$WIN_OUT" ]]; then
    echo "  [SKIP] No artifacts found for $TAG"
    continue
  fi

  echo "  Uploading assets to release $TAG ..."
  UPLOAD_ARGS=()
  [[ -f "$LINUX_OUT" ]] && UPLOAD_ARGS+=("$LINUX_OUT")
  [[ -f "$WIN_OUT"   ]] && UPLOAD_ARGS+=("$WIN_OUT")

  gh release upload "$TAG" "${UPLOAD_ARGS[@]}" --clobber \
    --repo benzjeremy/untis-go
  echo "  [OK] $TAG assets uploaded"
done

echo ""
echo "All done!"
ls -lh "$BUILD_DIR"/
