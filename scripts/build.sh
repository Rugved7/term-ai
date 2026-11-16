
#!/usr/bin/env bash

set -euo pipefail

# --------------------------------------------------
# Project configuration
# --------------------------------------------------

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

VERSION="${1:-dev}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------
# Validate inputs and dependencies
# --------------------------------------------------

if [[ ! "$VERSION" =~ ^[a-zA-Z0-9][a-zA-Z0-9._+-]*$ ]]; then
    echo "Error: invalid version '$VERSION'." >&2
    exit 1
fi

for tool in go git tar zip sha256sum date; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "Error: required command '$tool' is not installed." >&2
        exit 1
    fi
done

# --------------------------------------------------
# Build metadata
# --------------------------------------------------

LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"

# --------------------------------------------------
# Prepare output directory
# --------------------------------------------------

mkdir -p dist
rm -f dist/termai-* dist/SHA256SUMS

# --------------------------------------------------
# Cross-platform build function
# --------------------------------------------------

build() {
    local os="$1"
    local arch="$2"
    local extension="$3"

    local output="dist/termai-${VERSION}-${os}-${arch}${extension}"

    echo "Building ${os}/${arch} (${VERSION})..."

    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
        go build -trimpath -ldflags "$LDFLAGS" -o "$output" .

    echo "Built: $output"
}

# --------------------------------------------------
# Build all supported targets
# --------------------------------------------------

build linux amd64 ""
build linux arm64 ""

build windows amd64 ".exe"
build windows arm64 ".exe"

build darwin amd64 ""
build darwin arm64 ""

# --------------------------------------------------
# Package Linux and macOS binaries
# --------------------------------------------------

echo ""
echo "Packaging Linux and macOS archives..."

for os in linux darwin; do
    for arch in amd64 arm64; do
        binary="termai-${VERSION}-${os}-${arch}"
        archive="dist/${binary}.tar.gz"

        tar -czf "$archive" -C dist "$binary"

        echo "Packaged: $archive"
    done
done

# --------------------------------------------------
# Package Windows binaries
# --------------------------------------------------

echo ""
echo "Packaging Windows archives..."

for arch in amd64 arm64; do
    binary="termai-${VERSION}-windows-${arch}.exe"
    archive="dist/termai-${VERSION}-windows-${arch}.zip"

    zip -j "$archive" "dist/$binary"

    echo "Packaged: $archive"
done

# --------------------------------------------------
# Generate SHA-256 checksums
# --------------------------------------------------

echo ""
echo "Generating SHA-256 checksums..."

(
    cd dist

    sha256sum \
        termai-"${VERSION}"-*.tar.gz \
        termai-"${VERSION}"-*.zip \
        > SHA256SUMS
)

# --------------------------------------------------
# Build summary
# --------------------------------------------------

echo ""
echo "========================================"
echo "       TermAI Build Completed"
echo "========================================"
echo "Version:   $VERSION"
echo "Commit:    $COMMIT"
echo "Build date: $BUILD_DATE"
echo "========================================"
echo "Release artifacts:"
ls -lh dist/
echo "========================================"
echo "Checksums: dist/SHA256SUMS"
