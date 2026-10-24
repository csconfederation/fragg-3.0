#!/usr/bin/env bash
# Download the CI demo set listed in export/testdata/ci/demos.json from the public bucket into
# DIR as <demSha256>.dem, verifying both hashes. Demos already in DIR are kept.
#
#   scripts/fetch-ci-demos.sh DIR
#   FRAGG_CI_DEMOS=DIR go test ./export -run TestCIDemos
set -euo pipefail

dir=${1:?usage: fetch-ci-demos.sh DIR}
manifest="$(dirname "$0")/../export/testdata/ci/demos.json"
bucket=$(jq -r .bucket "$manifest")
mkdir -p "$dir"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

check() { # file sha256
  [[ $(sha256sum "$1" | cut -d' ' -f1) == "$2" ]] || { echo "sha256 mismatch: $1" >&2; return 1; }
}

jq -r '.demos[] | select(.object) | [.object, .objectSha256, .demSha256] | @tsv' "$manifest" |
  while IFS=$'\t' read -r object object_sha dem_sha; do
    out="$dir/$dem_sha.dem"
    if [[ -f $out ]] && check "$out" "$dem_sha" 2>/dev/null; then
      continue
    fi
    echo "fetching $object"
    curl -fsS --retry 3 -o "$tmp/object" "$bucket/$object"
    check "$tmp/object" "$object_sha"
    if [[ $object == *.zip ]]; then
      unzip -p "$tmp/object" '*.dem' >"$tmp/demo"
    else
      mv "$tmp/object" "$tmp/demo"
    fi
    check "$tmp/demo" "$dem_sha"
    mv "$tmp/demo" "$out"
  done
