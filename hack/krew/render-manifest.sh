#!/bin/sh
set -eu

tag="$1"
checksums_file="$2"
version="${tag#v}"
script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
HOMEPAGE="https://github.com/mist714/kubectl-waitx" \
VERSION="$version" \
SHA_DARWIN_AMD64="$(awk -v asset="kubectl-waitx_${version}_darwin_amd64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")" \
SHA_DARWIN_ARM64="$(awk -v asset="kubectl-waitx_${version}_darwin_arm64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")" \
SHA_LINUX_AMD64="$(awk -v asset="kubectl-waitx_${version}_linux_amd64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")" \
SHA_LINUX_ARM64="$(awk -v asset="kubectl-waitx_${version}_linux_arm64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")" \
envsubst '${HOMEPAGE} ${VERSION} ${SHA_DARWIN_AMD64} ${SHA_DARWIN_ARM64} ${SHA_LINUX_AMD64} ${SHA_LINUX_ARM64}' \
  < "$script_dir/plugin.tmpl.yaml" \
  > "$script_dir/../../plugins/waitx.yaml"
