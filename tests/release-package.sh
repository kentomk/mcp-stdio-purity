#!/bin/sh
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
release_root=$(mktemp -d)
cleanup() {
  rm -rf "$release_root"
}
trap cleanup EXIT HUP INT TERM

SOURCE_DATE_EPOCH=0 "$project_root/scripts/package-release.sh" v0.1.0 "$release_root/first"
SOURCE_DATE_EPOCH=0 "$project_root/scripts/package-release.sh" v0.1.0 "$release_root/second"

for target in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do
  archive="mcp-stdio-purity_v0.1.0_${target}.tar.gz"
  test -s "$release_root/first/$archive"
  cmp "$release_root/first/$archive" "$release_root/second/$archive"
done
cmp "$release_root/first/SHA256SUMS" "$release_root/second/SHA256SUMS"
[ "$(wc -l < "$release_root/first/SHA256SUMS")" -eq 4 ]

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$release_root/first" && sha256sum -c SHA256SUMS)
else
  (cd "$release_root/first" && shasum -a 256 -c SHA256SUMS)
fi

single_archive=mcp-stdio-purity_v0.1.0_linux_amd64.tar.gz
mkdir "$release_root/single"
cp "$release_root/first/$single_archive" "$release_root/single/"
cp "$release_root/first/SHA256SUMS" "$release_root/single/"
(
  cd "$release_root/single"
  grep "  ${single_archive}$" SHA256SUMS |
    sha256sum --check --strict -
)
if command -v shasum >/dev/null 2>&1; then
  (
    cd "$release_root/single"
    grep "  ${single_archive}$" SHA256SUMS |
      shasum -a 256 --check -
  )
fi

mkdir "$release_root/extract"
host_target="$(go env GOOS)_$(go env GOARCH)"
case "$host_target" in
  linux_amd64|linux_arm64|darwin_amd64|darwin_arm64) ;;
  *)
    echo "release execution test does not support host target: $host_target" >&2
    exit 1
    ;;
esac
tar -xzf "$release_root/first/mcp-stdio-purity_v0.1.0_${host_target}.tar.gz" -C "$release_root/extract"
[ "$("$release_root/extract/mcp-stdio-purity" version)" = v0.1.0 ]
cmp "$project_root/LICENSE" "$release_root/extract/LICENSE"
