#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <tag> <checksums.txt> > waitx.yaml" >&2
  exit 1
fi

tag="$1"
checksums_file="$2"
version="${tag#v}"
repo="mist714/kubectl-waitx"
hack_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
platform_template="$hack_dir/platform.tmpl.yaml"
manifest_template="$hack_dir/waitx.tmpl.yaml"

tmp_platforms="$(mktemp)"
trap 'rm -f "$tmp_platforms"' EXIT

for platform in "darwin amd64" "darwin arm64" "linux amd64" "linux arm64"; do
  set -- $platform
  os="$1"
  arch="$2"
  asset="kubectl-waitx_${version}_${os}_${arch}.tar.gz"
  sha256="$(awk -v asset="$asset" '$2 == asset { print $1 }' "$checksums_file")"
  if [ -z "$sha256" ]; then
    echo "missing checksum for $asset" >&2
    exit 1
  fi

  OS="$os" \
  ARCH="$arch" \
  REPO="$repo" \
  TAG="$tag" \
  ASSET="$asset" \
  SHA256="$sha256" \
  envsubst < "$platform_template" >> "$tmp_platforms"
  printf '\n' >> "$tmp_platforms"
done

PLATFORMS="$(cat "$tmp_platforms")" \
REPO="$repo" \
TAG="$tag" \
envsubst < "$manifest_template"
