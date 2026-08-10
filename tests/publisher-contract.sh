#!/bin/sh
# shellcheck disable=SC2016
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

jq -e '
  .schemaVersion == 2
  and .action == "update"
  and .owner == "kentomk"
  and .name == "mcp-stdio-purity"
  and (.description | type == "string" and length >= 20 and length <= 160)
  and (.topics | type == "array" and length >= 1 and length <= 10 and index("kento-oss") != null and all(type == "string"))
  and .candidateId == "20260718T144541Z-7acd"
  and (.targetUsers | type == "string" and length >= 10 and length <= 500)
  and (.jobToBeDone | type == "string" and length >= 10 and length <= 1000)
  and (.distributionPath | type == "string" and length >= 10 and length <= 500)
  and (.successMetric | type == "string" and length >= 10 and length <= 500)
  and (.reviewAfterDays | type == "number" and floor == . and . >= 1 and . <= 30)
  and .opportunityScore == 81
  and (.demandEvidence | type == "array" and length >= 3 and
       all(type == "object" and (.url | type == "string" and startswith("https://")) and
           (.kind | type == "string" and test("^[a-z][a-z0-9-]{2,49}$")) and
           (.independenceKey | type == "string" and length >= 3 and length <= 200)))
  and ((.demandEvidence | map(.independenceKey | ascii_downcase) | unique | length) >= 3)
  and ((.demandEvidence | map(.kind) | unique | length) >= 2)
  and (.alternatives | type == "array" and length >= 3 and
       all(type == "object" and (.name | type == "string" and length >= 2 and length <= 200) and
           (.url | type == "string" and startswith("https://")) and .tested == true and
           (.gap | type == "string" and length >= 10 and length <= 1000)))
  and ((.alternatives | map((.name | ascii_downcase) + "\n" + .url) | unique | length) >= 3)
  and .duplicateSearch.completed == true
  and (.duplicateSearch.summary | type == "string" and length >= 20)
  and (.differentiation | type == "string" and length >= 20)
  and .testCommand == "scripts/publisher-gate.sh"
  and .license == "MIT"
  and (.commitMessage | length >= 10 and length <= 120)
' publish-request.json >/dev/null

jq -e --slurpfile request publish-request.json '
  .schemaVersion == 1
  and .candidateId == $request[0].candidateId
  and .owner == $request[0].owner
  and (.createdBy | test("Matsuki Kento") and test("@kentomk") and test("AI|automated"; "i"))
' .kento-oss.json >/dev/null

grep -Eq '^## Installation\b' README.md
grep -Eq '^## Quick start\b' README.md
grep -Eq '60-second quick start' README.md
quick_start_line=$(grep -n -m 1 '^## Quick start$' README.md | cut -d: -f1)
quick_start_block=$(tail -n "+$quick_start_line" README.md | sed -n '1,/^```$/p')
printf '%s\n' "$quick_start_block" | grep -Fxq 'mkdir -p ./bin'
printf '%s\n' "$quick_start_block" | grep -Fxq 'go build -o ./bin/mcp-stdio-purity ./cmd/mcp-stdio-purity'
grep -Fq 'releases/tag/v0.1.4' README.md
grep -Fq 'mcp-stdio-purity@v0.1.4' README.md
grep -Fq 'uses: kentomk/mcp-stdio-purity@4724c0203a400c6b26e99d7cc00e17f4a5112eff # v0.1.4 release revision' README.md
grep -Fq "grep -E \"^[0-9a-fA-F]{64}  \$archive\$\" SHA256SUMS | sha256sum --check --strict -" README.md
grep -Fq 'expected exactly one checksum row' README.md
grep -Fq 'archive contains an unsafe member path' README.md
grep -Fq "extract_dir=\$(mktemp -d)" README.md
grep -Fq "tar -xzf \"\$archive\" -C \"\$extract_dir\"" README.md
# shellcheck disable=SC2016
grep -Fq 'expected_binary="$extract_dir/mcp-stdio-purity_v0.1.4_linux_amd64/mcp-stdio-purity"' README.md
grep -Fq 'test -f "$expected_binary" && test ! -L "$expected_binary"' README.md
grep -Fq 'curl -fsSLo SHA256SUMS' README.md
grep -Fq "mkdir -p \"\$HOME/.local/bin\"" README.md
grep -Fq 'install -m 0755 "$expected_binary" "$HOME/.local/bin/mcp-stdio-purity.new"' README.md
grep -Fq '"$expected_binary" version' README.md
grep -Fq 'shasum -a 256 --check -' README.md
checksum_checks=$(grep -Fc "checksum_matches=\$(grep -Ec" README.md)
unsafe_path_checks=$(grep -Fc "unsafe_member=\$(tar -tzf" README.md)
test "$checksum_checks" -ge 3
test "$unsafe_path_checks" -ge 3
# shellcheck disable=SC2016
macos_block=$(sed -n '/On macOS, replace the verification command with:/,/Replace `linux_amd64`/p' README.md)
printf '%s\n' "$macos_block" | grep -Fq 'shasum -a 256 --check -'
# shellcheck disable=SC2016
printf '%s\n' "$macos_block" | grep -Fq 'unsafe_member=$(tar -tzf "$archive"'
# shellcheck disable=SC2016
printf '%s\n' "$macos_block" | grep -Fq 'tar -xzf "$archive" -C "$extract_dir"'
# shellcheck disable=SC2016
printf '%s\n' "$macos_block" | grep -Fq 'expected_binary="$extract_dir/${archive%.tar.gz}/mcp-stdio-purity"'
printf '%s\n' "$macos_block" | grep -Fq 'test -f "$expected_binary" && test ! -L "$expected_binary"'
grep -Fq 'The published' SECURITY.md
grep -Fq 'v0.1.4' SECURITY.md
if grep -Fq 'not published yet' SECURITY.md; then
  echo 'SECURITY.md still claims the public project is unpublished' >&2
  exit 1
fi
grep -Eq '^## Failure triage$' README.md
grep -Fq 'case "' README.md
grep -Fq '" in' README.md
grep -Fq 'For exit' README.md
grep -Fq 'diagnostics' README.md
if grep -Eq 'After the first release|FULL_COMMIT_SHA|mcp-stdio-purity@v0\.1\.[0-2]|releases/tag/v0\.1\.[0-2]' README.md; then
  echo 'README contains a stale pre-publication or prior-release install path' >&2
  exit 1
fi
grep -q 'Matsuki Kento' README.md
grep -q '@kentomk' README.md
grep -Eiq 'AI|automated' README.md

grep -Eq 'uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
grep -Eq 'uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
if grep -Eq 'uses: actions/(checkout|setup-go)@v[0-9]' .github/workflows/*.yml action.yml; then
  echo 'mutable GitHub Action reference found' >&2
  exit 1
fi
grep -Fq "go-version: '1.26.5'" action.yml
if grep -Fq 'go-version-file:' action.yml; then
  echo 'composite Action must use the reviewed exact Go patch' >&2
  exit 1
fi

test -x scripts/publisher-gate.sh
test -x scripts/release-gate.sh
sh -n scripts/publisher-gate.sh scripts/release-gate.sh
