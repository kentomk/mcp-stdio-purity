#!/bin/sh
set -eu

workflow=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)/.github/workflows/release.yml
grep -Fq 'types: [published]' "$workflow"
grep -Fq 'workflow_dispatch:' "$workflow"
grep -Fq 'repository_dispatch:' "$workflow"
grep -Fq 'types: [kento_release_repair]' "$workflow"
grep -Fq 'tagName:' "$workflow"
grep -Fq 'required: true' "$workflow"
grep -Fq "ref: \${{ github.event.release.tag_name || inputs.tagName || github.event.client_payload.tagName }}" "$workflow"
# shellcheck disable=SC2016
[ "$(grep -Fc 'TAG_NAME: ${{ github.event.release.tag_name || inputs.tagName || github.event.client_payload.tagName }}' "$workflow")" -eq 2 ]
grep -Fq 'contents: write' "$workflow"
grep -Fq "go-version: '1.26.5'" "$workflow"
if grep -Fq "go-version: '1.26.x'" "$workflow"; then
  echo 'release workflow must pin the exact reviewed Go patch version' >&2
  exit 1
fi
[ "$(grep -Ec 'uses: [^ ]+@[0-9a-f]{40}([[:space:]]|$)' "$workflow")" -eq 2 ]
# shellcheck disable=SC2016
grep -Fq 'gh release upload "$TAG_NAME"' "$workflow"
if grep -Eq 'uses: [^ ]+@(main|master|v[0-9]+)([[:space:]]|$)' "$workflow"; then
  echo 'mutable Action reference found in release workflow' >&2
  exit 1
fi
grep -Fq 'dist/SHA256SUMS' "$workflow"
grep -Fq -- '--clobber' "$workflow"
