#!/bin/sh
set -eu

comparison_root=$(unset CDPATH; cd -- "$(dirname -- "$0")" && pwd)
project_root=$(unset CDPATH; cd -- "$comparison_root/../.." && pwd)
node_modules=${MSP_ALTERNATIVE_NODE_MODULES:-$comparison_root/node_modules}
install_root=$comparison_root

if [ ! -x "$node_modules/.bin/mcp-inspector-cli" ] ||
   [ ! -x "$node_modules/.bin/mcp-compliance" ] ||
   [ ! -x "$node_modules/.bin/mcp-z" ]; then
  command -v npm >/dev/null 2>&1 || { echo 'npm is required for the isolated alternative comparison' >&2; exit 2; }
  npm ci --prefix "$install_root" --ignore-scripts --no-audit --no-fund
  node_modules=$install_root/node_modules
fi

# shellcheck disable=SC2016
node -e '
const root = process.argv[1];
const expected = {
  "@modelcontextprotocol/inspector-cli": "1.0.1",
  "@yawlabs/mcp-compliance": "0.16.3",
  "@mcp-z/cli": "1.0.5",
};
for (const [name, version] of Object.entries(expected)) {
  const actual = require(`${root}/${name}/package.json`).version;
  if (actual !== version) throw new Error(`${name}: expected ${version}, got ${actual}`);
}
' "$node_modules"

npm audit --prefix "$comparison_root" --package-lock-only --omit=dev --audit-level=high >/dev/null

work_root=$(mktemp -d)
cleanup() {
  rm -rf "$work_root"
}
trap cleanup EXIT HUP INT TERM

binary=$work_root/mcp-stdio-purity
go build -trimpath -o "$binary" "$project_root/cmd/mcp-stdio-purity"

run_expect_zero() {
  name=$1
  shift
  set +e
  timeout 90 "$@" >"$work_root/$name.stdout" 2>"$work_root/$name.stderr"
  status=$?
  set -e
  if [ "$status" -ne 0 ]; then
    echo "$name unexpectedly exited $status" >&2
    sed -n '1,80p' "$work_root/$name.stderr" >&2
    exit 1
  fi
}

write_mcp_z_config() {
  mode=$1
  config=$2
  node -e '
const fs = require("node:fs");
const [output, server, mode] = process.argv.slice(1);
fs.writeFileSync(output, JSON.stringify({mcpServers: {fixture: {command: process.execPath, args: [server, "--mode", mode]}}}));
' "$config" "$comparison_root/server.mjs" "$mode"
}

for mode in clean startup late cleanup; do
  # shellcheck disable=SC2016
  run_expect_zero "inspector-$mode" \
    sh -c 'cd "$1" && shift && exec "$@"' sh \
      "$node_modules/@modelcontextprotocol/inspector-cli/build" \
      "$node_modules/.bin/mcp-inspector-cli" --cli node "$comparison_root/server.mjs" \
      --mode "$mode" --method tools/list

  run_expect_zero "compliance-$mode" \
    "$node_modules/.bin/mcp-compliance" test --format json --strict \
      --only transport,lifecycle --startup-timeout 15000 --timeout 5000 -- \
      node "$comparison_root/server.mjs" --mode "$mode"

  config=$work_root/mcp-z-$mode.json
  write_mcp_z_config "$mode" "$config"
  run_expect_zero "mcp-z-$mode" \
    "$node_modules/.bin/mcp-z" inspect --config "$config" --health --json

  set +e
  "$binary" check --timeout 5s --cleanup-grace 250ms -- \
    node "$comparison_root/server.mjs" --mode "$mode" \
    >"$work_root/purity-$mode.stdout" 2>"$work_root/purity-$mode.stderr"
  purity_status=$?
  set -e
  if [ "$mode" = clean ]; then
    [ "$purity_status" -eq 0 ]
  else
    [ "$purity_status" -eq 1 ]
    grep -q '^MSP001 ' "$work_root/purity-$mode.stdout"
    if grep -q 'fixture .*\(banner\|log\)' "$work_root/purity-$mode.stdout"; then
      echo "raw contamination leaked for mode $mode" >&2
      exit 1
    fi
  fi
done

printf '%s\n' 'alternative comparison: 3 pinned tools accepted clean and 3 contaminated modes; mcp-stdio-purity rejected all contaminated modes'
