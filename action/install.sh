#!/usr/bin/env bash
# Download the shallnot release binary for this runner, verify its checksum,
# and put it on the PATH of the following steps.
set -euo pipefail

REPOSITORY="RachidChabane/shallnot"
VERSION="${SHALLNOT_VERSION:-latest}"

case "${RUNNER_OS:-$(uname -s)}" in
  Linux) os=linux ;;
  macOS | Darwin) os=darwin ;;
  Windows | MINGW* | MSYS*) os=windows ;;
  *) echo "shallnot: unsupported runner OS ${RUNNER_OS:-$(uname -s)}" >&2; exit 2 ;;
esac
case "${RUNNER_ARCH:-$(uname -m)}" in
  X64 | x86_64 | amd64) arch=amd64 ;;
  ARM64 | arm64 | aarch64) arch=arm64 ;;
  *) echo "shallnot: unsupported runner architecture ${RUNNER_ARCH:-$(uname -m)}" >&2; exit 2 ;;
esac

if [ "$VERSION" = "latest" ]; then
  base="https://github.com/${REPOSITORY}/releases/latest/download"
else
  base="https://github.com/${REPOSITORY}/releases/download/${VERSION}"
fi

archive="shallnot_${os}_${arch}.tar.gz"
destination="${RUNNER_TEMP:-$(mktemp -d)}/shallnot-bin"
mkdir -p "$destination"
cd "$destination"

curl -fsSL --retry 3 -o "$archive" "${base}/${archive}"
curl -fsSL --retry 3 -o checksums.txt "${base}/checksums.txt"
expected="$(grep " ${archive}\$" checksums.txt | cut -d ' ' -f 1)"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$archive" | cut -d ' ' -f 1)"
else
  actual="$(shasum -a 256 "$archive" | cut -d ' ' -f 1)"
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "shallnot: checksum mismatch for ${archive}" >&2
  exit 2
fi

tar -xzf "$archive"
if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$destination" >> "$GITHUB_PATH"
fi
echo "shallnot installed in $destination"
