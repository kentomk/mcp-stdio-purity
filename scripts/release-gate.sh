#!/bin/sh
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

scripts/quality-gate.sh
tests/publisher-contract.sh
tests/publisher-payload.sh
tests/quickstart-clean.sh
tests/alternatives/compare.sh
