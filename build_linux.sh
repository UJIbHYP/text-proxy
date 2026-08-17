#!/usr/bin/env bash
# Сборка бинарников textproxy для Linux (amd64, arm64, arm).
# Использование: ./build_linux.sh
set -euo pipefail

cd "$(dirname "$0")"

OUT="dist"
mkdir -p "$OUT"

build() {
    local goos="$1" goarch="$2"
    local name="textproxy-${goos}-${goarch}"
    echo "→ сборка $name"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -ldflags "-s -w" -o "$OUT/$name" .
}

build linux amd64
build linux arm64
build linux arm

echo
echo "Готово. Бинарники в $OUT/:"
ls -lh "$OUT"/textproxy-linux-*
