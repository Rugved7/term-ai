
#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

mkdir -p dist

build() {
    local os="$1"
    local arch="$2"
    local extension="$3"

    local output="dist/termai-${os}-${arch}${extension}"

    echo "Building ${os}/${arch}..."

    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
        go build -trimpath -o "$output" .

    echo "Built: $output"
}

build linux amd64 ""
build linux arm64 ""

build windows amd64 ".exe"
build windows arm64 ".exe"

build darwin amd64 ""
build darwin arm64 ""

echo ""
echo "All builds completed successfully."
echo "Artifacts available in: dist/"
