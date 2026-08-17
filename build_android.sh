#!/usr/bin/env bash
# Сборка бинарника textproxy для Android (arm64) с системным резолвером.
#
# Чтобы Go на Android резолвил DNS "как браузер" (через системный netd), нужен
# CGO + Android NDK: бинарник линкуется с bionic, и getaddrinfo() идёт в netd.
#
# Требуется Android NDK (переменная ANDROID_NDK_HOME или ANDROID_NDK_ROOT).
# Скачать: https://developer.android.com/ndk/downloads
#
# Использование: ANDROID_NDK_HOME=/opt/android-ndk ./build_android.sh
set -euo pipefail

cd "$(dirname "$0")"

OUT="dist"
mkdir -p "$OUT"

# Ищем NDK: сначала переменные окружения, затем типовые пути.
NDK="${ANDROID_NDK_HOME:-${ANDROID_NDK_ROOT:-}}"
if [[ -z "$NDK" ]]; then
    for p in "$HOME"/Android/Sdk/ndk/* "$HOME"/android-ndk* "$HOME"/.local/android-ndk* /opt/android-ndk* /usr/local/android-ndk*; do
        if [[ -d "$p" ]]; then NDK="$p"; break; fi
    done
fi

if [[ -z "$NDK" ]]; then
    echo "Ошибка: Android NDK не найден." >&2
    echo "Установите NDK и укажите путь:" >&2
    echo "  export ANDROID_NDK_HOME=/path/to/android-ndk" >&2
    echo "Скачать: https://developer.android.com/ndk/downloads" >&2
    exit 1
fi

# clang для aarch64 (API 24 = Android 7.0+).
CC="$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android24-clang"
if [[ ! -x "$CC" ]]; then
    echo "Ошибка: не найден $CC" >&2
    echo "Проверьте путь к NDK и имя хоста (linux-x86_64 / darwin-x86_64)." >&2
    exit 1
fi

echo "→ сборка textproxy-android-arm64 (NDK: $NDK)"
CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC="$CC" \
    go build -trimpath -ldflags "-s -w" -o "$OUT/textproxy-android-arm64" .

echo
echo "Готово. Бинарник в $OUT/:"
ls -lh "$OUT"/textproxy-android-arm64
