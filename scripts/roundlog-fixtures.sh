#!/bin/bash
# Regenerates engine/roundlog/testdata/*.json.gz from engine/roundlog/fixtures.tsv with cmd/roundlog.
# The demos are on the corpus share; DEMO_ROOT is its mount.
set -eu
cd "$(dirname "$0")/.."
ROOT=${DEMO_ROOT:-/mnt/cs2-demos}
BIN=$(mktemp)
trap 'rm -f "$BIN"' EXIT
go build -o "$BIN" ./engine/cmd/roundlog
rm -f engine/roundlog/testdata/*.json.gz
awk -F'\t' '!/^#/ && NF >= 2 { if (!($1 in r)) order[n++] = $1; r[$1] = r[$1] " " $2 } END { for (i = 0; i < n; i++) print order[i] "\t" r[order[i]] }' \
  engine/roundlog/fixtures.tsv |
  while IFS=$'\t' read -r demo rounds; do
    # shellcheck disable=SC2086
    "$BIN" -o engine/roundlog/testdata "$ROOT/$demo" $rounds
  done
