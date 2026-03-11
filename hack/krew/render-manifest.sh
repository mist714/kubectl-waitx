#!/bin/sh
set -eu

tag="$1"
checksums_file="$2"
version="${tag#v}"
script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
trap 'rm -f "$script_dir/kustomize/vars.env"' EXIT

cat > "$script_dir/kustomize/vars.env" <<EOF
VERSION=$version
SHA_DARWIN_AMD64=$(awk -v asset="kubectl-waitx_${version}_darwin_amd64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")
SHA_DARWIN_ARM64=$(awk -v asset="kubectl-waitx_${version}_darwin_arm64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")
SHA_LINUX_AMD64=$(awk -v asset="kubectl-waitx_${version}_linux_amd64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")
SHA_LINUX_ARM64=$(awk -v asset="kubectl-waitx_${version}_linux_arm64.tar.gz" '$2 == asset { print $1 }' "$checksums_file")
EOF

kustomize build "$script_dir/kustomize" > "$script_dir/../../plugins/waitx.yaml"
