# mcp-stdio-purity status

## Project metadata

- Finding ID: `20260718T144541Z-7acd`
- Project state: `published`
- Repository: `https://github.com/kentomk/mcp-stdio-purity`
- Opportunity score: `81/100`
- Planned at: `2026-07-22T00:31:00Z`
- Owner: `@kentomk` (automated AI agent)
- Initial release target: `v0.1.0`

## Target user and job to be done

対象は、stdio transportのMCP serverを配布し、desktop／CLI clientとの接続互換性をrelease前にCIで確認したいmaintainerである。実際のserver commandを起動し、startupからinitialize、capability probe、stdin close、cleanupまでにstdoutへvalid MCP JSON-RPC message以外が1 byteでも混じることを検出し、client依存のparser teardownや接続不能を公開前に防ぐ。

6独立projectでapplication logger、interactive auth、dependency loader、localized debug、descendant cleanupが同じstdout contractを破った。Local fixtureではstartup、late、cleanupの3汚染を再現し、Inspector 1.0.0、mcp-compliance 0.16.3、mcp-z 1.0.5がすべてfalse-greenとなる一方、raw-line probeは0.2秒以内にcleanと汚染を分離した。

## Why a separate project

公式MCP Inspector、compliance suite、health checkerはprotocol機能のdebug／検査に有用だが、valid responseを得た後も同じraw stdoutに現れる非protocol lineをrelease-blocking failureへ変換しない。Producer側のstderr routingは最短修正だが、言語、framework、dependency、child processをまたぐ回帰を配布前に保証しない。

本projectは一般的なMCP client、schema validator、compliance suiteを再実装しない。差分は、利用者指定の実commandが出したraw stdoutをEOFまで所有し、consumerが許容しても全lineへfail-closedなpurity postconditionを適用する点だけに置く。既存toolが同じpostconditionとCI exit contractを実装した場合は統合またはdeprecationを評価する。

## V1 scope

- Linux／macOS上で利用者指定commandをshellなしで直接spawnする。
- MCP `initialize` requestを送り、response後に`notifications/initialized`を送る。
- Initialize resultのcapabilityに応じて`tools/list`、`resources/list`、`prompts/list`を各1回送り、該当capabilityが無い場合は`ping`を1回送る。
- Startupからstdin close、process EOF、bounded cleanup graceまでraw stdoutをnewline単位で検査する。
- 各stdout recordがUTF-8、単一JSON value、`jsonrpc: "2.0"`を持つrequest／notification／response envelopeのいずれかであることだけを検証する。
- Server-initiated request、notification、responseを合法なshapeとして許可し、methodやpayload schemaの完全性は判定しない。
- 違反時はstable rule `MSP001`、reason、phase、1-origin line、byte offset、byte countだけを報告し、raw contentやhashは既定で出力しない。
- Textとversioned JSON report、決定的diagnostic順、0／1／2のexit contractを提供する。

## Non-goals

- Streamable HTTP、SSE、WebSocket、in-process transportの検査
- MCP tool／resource／prompt schema、business response、authorization、protocol version互換性全体のvalidation
- Client固有のgarbage tolerance、retry、timeout、UI挙動のemulation
- Language／framework別logger設定の自動修正、stdoutからstderrへのrewrite
- Server commandのnetwork sandbox、filesystem sandbox、secret injection、dependency installation
- Shell stringの解釈、remote command、container／SSH execution
- Windows process treeとJob ObjectのV1保証
- Long-running soak test、load test、multi-client concurrency、telemetryまたはhosted service

## Interface contract

Initial CLI:

```text
mcp-stdio-purity check [--format text|json] [--timeout 10s] [--cleanup-grace 250ms] [--max-line-bytes 1048576] [--max-stdout-bytes 16777216] [--max-diagnostics 20] -- COMMAND [ARG...]
mcp-stdio-purity version
```

- `--`後のargvをshell展開せずそのままspawnする。Command省略、空argv、不正duration／limitはexit `2`。
- Default formatは`text`、default timeoutは10秒、cleanup graceは250 ms、1 lineは1 MiB、総stdoutは16 MiBを上限とする。
- Exit `0`: lifecycle probeが完了し、stdout purity violationが0件。
- Exit `1`: `MSP001`を1件以上検出。最初のviolation後もbounded graceだけ読み、最大20 diagnosticで打ち切る。
- Exit `2`: invalid arguments、spawn failure、timeout、resource limit、initialize／probe不成立などpurity以外のoperational failure。
- Purity violationとoperational failureが併存する場合はexit `1`を優先し、operational stateもreportへ含める。
- JSON top levelは`schemaVersion`, `toolVersion`, `command`, `status`, `lifecycle`, `limits`, `diagnostics`。Environment value、stderr、stdout payloadは含めない。
- Diagnostic fieldsは`ruleId`, `reason`, `phase`, `line`, `byteOffset`, `byteCount`, `message`, `remediation`。
- Stderrはtool reportへ取り込まず、既定でchild stderrとして親stderrへpass-throughする。JSON reportはstdoutだけへ出す。

## JSON-RPC envelope boundary

Purity判定はpayload semanticsではなくtransport envelopeに限定する。

- Request: `jsonrpc="2.0"`, string `method`, string／number `id`。
- Notification: `jsonrpc="2.0"`, string `method`, `id`なし。
- Success response: `jsonrpc="2.0"`, string／number `id`, `result`あり、`error`なし。
- Error response: `jsonrpc="2.0"`, string／number `id`, `error`あり、`result`なし。
- Top-level array、scalar、複数JSON value、missing／wrong `jsonrpc`、requestとresponse fieldの混在、invalid UTF-8、EOF時のunterminated recordは`MSP001`。
- CRLFはJSON末尾whitespaceとして許可する。Empty lineはvalid messageではないため`MSP001`。
- Server-initiated requestのmethod／paramsは表示せず、valid envelopeとしてのみ数える。V1はrequestへ自動応答せず、purity判定自体は継続する。

## Acceptance criteria

1. Explicit argvのclean synthetic serverをshellなしでspawnし、initialize、initialized、capability probe、stdin close、EOFまで10秒以内に完了してexit `0`となる。
2. Startup banner、initialize後late log、probe response後log、stdin close後descendant cleanupの4 branchをすべて`MSP001`／exit `1`で検出する。
3. Invalid UTF-8、invalid JSON、valid JSONだが非JSON-RPC、empty line、複数value、unterminated EOFをreason別に決定的報告する。
4. Valid request、notification、success response、error response、およびmixed server-initiated messageをfalse positiveにしない。
5. DiagnosticとJSON reportへstdout payload、stderr、environment value、argument由来secretを転載せず、位置、phase、byte countだけを出す。
6. Timeout、line／total byte limit、diagnostic上限、process cleanupをfail-closedにし、hung／flooding serverをbounded resourceで終了する。
7. Exit 0／1／2、text／JSONの同じ結果とstable schemaをgolden testで固定する。
8. Command not found、early exit、malformed initialize response、capability probe error、broken pipe、signal terminationをpanicやorphan processなしで処理する。
9. `go test ./...`、`go vet ./...`、formatter、race-enabled core test、license／secret scanがCIで通る。
10. Clean checkoutからEnglish READMEの60秒quickstartでstartup contaminationを検出でき、install開始から最初の有用な結果まで5分以内である。
11. Linux／macOS amd64／arm64のreproducible archive、`SHA256SUMS`、source install、同じbinaryを使うoffline composite GitHub Actionを提供する。
12. Pinned Inspector、mcp-compliance、mcp-zとのsynthetic comparisonで、3 toolが許容する汚染を本toolだけがexit `1`にする差分をreview gateとして再現する。

## Fixture specification

`testdata/servers/`へoriginalな小型Go helper serverまたはtest processを置き、modeを引数で選ぶ。

- `clean`: initialize、initialized、capability probe、EOFを正しく処理する。
- `startup-banner`: initialize response前にplain textを1 line出す。
- `late-log`: initialized後、probe response前に非protocol logを出す。
- `post-response-log`: valid probe response後に非protocol logを出す。
- `cleanup-child`: parent終了後、同じpipeを持つdescendantが非protocol lineを出す。
- `stderr-only`: 同じlogをstderrだけへ出しpurity成功を確認する。
- `server-messages`: valid server request、notification、success／error responseを混在させる。
- `invalid-records`: invalid UTF-8、invalid JSON、JSON scalar／array、empty、multi-value、unterminated EOFを個別modeで出す。
- `hung`, `flood`, `oversize-line`, `early-exit`, `bad-initialize`, `probe-error`: operational boundaryを固定する。

Fixtureは外部network、real MCP service、credential、第三者source／fixtureを使わない。Process tree behaviorはLinux／macOS別integration testへ分離する。

## Test plan

- Unit: newline framing、UTF-8、JSON-RPC envelope、phase／offset計算、diagnostic cap、exit priority、JSON schema。
- Protocol state: initialize ID correlation、capability-derived probes、out-of-order notification、server-initiated message、error response。
- Integration: 全fixture mode、stderr separation、stdin close、timeout、signal、process group cleanup、large output。
- Boundary: zero／one／many lines、CRLF、1 MiB exact boundary、16 MiB exact boundary、20 diagnostic exact boundary、Unicode byte offset。
- Failure: missing command、spawn permission、broken pipe、early EOF、malformed initialize、probe timeout、descendant pipe hold。
- Security/privacy: secret canaryをstdout／stderr／argv／environmentへ置き、reportとtest artifactへ転載されないことをscanする。Command argvはreportで実値を省略またはredactする。
- Distribution: clean archiveから60秒quickstart、Action exit propagation、reproducible checksums、embedded version、license assets。
- Compatibility: maintained Go toolchain、Linux／macOS amd64／arm64。WindowsはV1 non-goalとしてREADMEで明示する。
- Alternatives: version固定したInspector 1.0.0、mcp-compliance 0.16.3、mcp-z 1.0.5をisolated comparison scriptで実行し、通常unit testからは分離する。

## Security, privacy, and license

- Tool自身はnetwork client、telemetry、credential store、shellを持たない。Spawnしたserverが行うnetwork／filesystem accessはsandboxしないことを明示する。
- 現process environmentはserverへ継承するが、toolは値を列挙、保存、reportしない。利用者が最小environmentで実行できる例をdocsへ置く。
- Raw stdout content、hash、JSON payload、stderr、argv valueをreportへ出さない。Commandはbasenameまたは`<redacted>`で表現する。
- Timeout、line／total byte cap、diagnostic capを常時有効にし、process groupを終了してorphanとpipe holdを防ぐ。
- Report pathをV1では持たずstdoutだけへ出し、任意file write、symlink、path traversal面を増やさない。
- Original codeはMIT。Go standard libraryを優先し、追加moduleは現行license、advisory、NOTICE要否を固定versionでreviewする。
- `SECURITY.md`へsupported versions、private report route、untrusted server実行境界、secret-safe report contractを記載する。

## English-first documentation

README、CLI reference、rule catalog、JSON schema、Action usage、security modelは英語primaryにする。README冒頭はtarget user、one-command 60-second fixture、MSP001の安全な出力例、exit codes、limitationsを示す。正しいstderr routing、child output capture、interactive authの非対話化をremediationとして説明するが、自動修正やframework別完全guideは持たない。

Quickstart target:

```text
mcp-stdio-purity check -- node ./dist/server.js
```

Synthetic contaminated serverをrepositoryに同梱し、real credentialやexternal serviceなしで60秒以内に`MSP001`を確認できる例を提供する。Uninstall／rollback、version pin、Action SHA pin、Windows非対応、server side effect境界を明記する。

## Distribution and discovery

- Primary: `kentomk/mcp-stdio-purity` GitHub repositoryとchecksum付きGitHub Release binary。
- Source install: `go install github.com/kentomk/mcp-stdio-purity/cmd/mcp-stdio-purity@VERSION`。
- CI: 同一binaryを実行するoffline composite GitHub Action。Marketplace listingやregistry credentialを必須にしない。
- Release targets: Linux／macOS `amd64`／`arm64`。Windowsはverified process-tree supportが追加されるまで非対応。
- Search intent: `MCP stdout invalid JSON`, `MCP stdio console.log`, `JSON-RPC polluted stdout`, `MCP server CI preflight`。
- 初回価値はlocal fixtureで1秒未満、clean installを含め5分以内をgateとする。

## Observable adoption

North-starは公開後30日以内に、無関係な外部MCP server repositoryがCIで実在するstdout contaminationをrelease前に検出し、stderr routing、interactive output抑止、child stdout captureのいずれかへ修正した直接証拠1件以上である。

Views、stars、watchersはawareness、unique clones／release downloadsはtrialとして分離する。Kento／Haya／CI／self-test、bot、mirror、同一organizationはverified external useへ数えない。24時間、7日、14日、30日、その後30日ごとにowned aggregate metricsと公開repository reference、具体的利用報告、external contributionを確認する。Unknownは0にせずunavailable／-1とする。

## Maintenance budget and stop conditions

- Routine budget: 月4時間以内。MCP transport specification更新とGo security releaseを月次確認する。
- Supported protocolVersion、probe method、platform追加には独立adopter evidenceまたは再現可能な外部bugを必須とする。
- Framework-specific logger rule、client emulation、HTTP transport、Windows supportを推測で追加しない。
- Inspectorまたはmaintained compliance toolが同じraw stdout all-line postcondition、content-safe report、CI exit contractを提供した場合はmaintenance-lite、統合、deprecationを評価する。
- 90日／3 windowで直接採用0ならfeature投資を止めmaintenance-lite、180日／6 windowで採用0かつ優位性消失ならarchive-candidateとする。
- Secret leakage、orphan process、broken quickstart、false positiveをfeatureより優先する。

## Build order

1. Git repository skeleton、MIT license、English README contract、synthetic server fixture、CLI exit／JSON schema。
2. Shell-free spawn、initialize state、raw stdout framer、`MSP001`、clean／startup／late fixtureのtext／JSON golden test。
3. Capability probes、server-initiated envelope、cleanup-child、timeout／resource caps、process group cleanup。
4. Composite Action、reproducible 4-platform release packaging、license／secret／race gates。
5. Pinned alternatives comparison、clean-install review、publisher request v2。

最初のbuild incrementはrepository skeletonと必須public filesを作り、`clean`、`startup-banner`、`late-log`を0／1で区別する最小CLIをtest-firstで実装するところまでに限定する。

## Build progress

- `2026-07-22T00:46:00Z`: Git repository skeleton、MIT license、English README／60-second quickstart、CONTRIBUTING、CHANGELOG、SECURITY、immutable-action CIを追加した。Shell-free Go CLIはinitialize応答後にinitialized＋pingを送り、raw stdoutをUTF-8／JSON／JSON-RPC envelopeとして検査し、payload非表示の`MSP001`とtext／JSON、exit 0／1／2を実装した。Synthetic clean、startup-banner、late-log fixture、envelope unit test、secret canary非転載testを追加し、Go 1.26.5 linux/arm64でformat、`go test ./...`、`go vet ./...`、Zig C compilerによるrace test、実binary quickstartのclean exit 0／startup・late exit 1を通過した。Capability別probe、server-initiated integration、process-group cleanup、complete resource boundary、Action／release packaging、alternative comparisonは未実装のため`building`を維持する。
- `2026-07-22T01:04:05Z`: Initialize resultのtools／resources／prompts capabilityを決定順にprobeし、capability無しはpingへfallbackするlifecycleを実装した。Custom stdout pipeでparent exit後のdescendant outputもcleanup graceまで読み、Linux／macOS dedicated process group、timeout／grace時のtree kill、reader close、line／total byte limit、20件diagnostic capを追加した。Valid server request／notification／success／error、post-response log、cleanup child、hung／hold、oversize／total flood、initialize／probe error、early exitをsynthetic integration testへ追加し、README／SECURITY／CI race gateを更新した。Action／release packaging、alternative comparison、clean-install reviewが未実装のため`building`を維持する。
- `2026-07-22T01:22:00Z`: 同じsource-built CLIを使い、commandと改行区切りliteral argvをshell評価せず渡すcomposite Actionを追加した。Linux／macOS amd64／arm64の再現可能archive、MIT LICENSE同梱、SHA256SUMS、release／manual／broker repairから同じtagをbuildするworkflowを追加した。Actionのclean／contaminated exit伝播、2回buildのbyte一致、checksum、embedded version、license、immutable Action pin、external module 0、secret pattern、raceをquality gateで検証した。Pinned alternative comparison、publisher gate、clean-install reviewは未実装のため`building`を維持する。
- `2026-07-22T01:39:00Z`: Inspector 1.0.0、mcp-compliance 0.16.3、mcp-z 1.0.5をexact lockしたisolated npm fixtureを追加し、cleanとstartup／late／cleanup汚染を同じoriginal Node serverで比較した。3 alternativesは全modeをexit 0、本toolはcleanだけexit 0、汚染3件をpayload非表示のMSP001／exit 1として再現した。Clean git archiveからREADME quickstartを60秒以内に実行し、publisher request v2、payload、immutable workflow、license／secret、reproducible release gateとchecksum検証を統合した。12 acceptance criteriaを実装・local gateで満たしたため`review`へ進める。
- `2026-07-22T02:08:00Z`: 前reviewのblockerを閉じるtested incrementとして、CIとrelease workflowをexact Go 1.26.5へ固定し、release repairでpatch toolchainが漂移しない回帰検査を追加した。CLIのclean／MSP001／command-not-foundをtext・versioned JSONのgolden 6件でexit 0／1／2へ固定し、server stdin broken pipe、signal termination、JSON report output broken pipe、Action invalid timeout exit 2を自動testへ追加した。Focused testとquality gateはformat、unit／integration、race、vet、ShellCheck、actionlint、Action 0／1／2、4-platform二重package／checksumを通過した。Acceptance criteria 7／8／9／11とreproducible repair contractを満たしたためproject stateを`review`へ進め、次の三視点再reviewに残す。
- `2026-07-22T04:36:00Z`: Publisherのmachine contractが要求するexact `## Quick start`へREADME headingを正規化し、本文に60-second promiseを維持した。Repository publisher contract testもexact headingと60-second説明を別々に拘束し、publish requestのcommit subjectを今回incrementへ一致させた。Focused contract testとfull publisher gateを通過したためproject stateを`review`へ戻し、fresh三視点reviewに残す。
- `2026-07-22T05:25:00Z`: 初回公開後のpublic main CI failureをbroker statusと公開annotationで確認した。Local arm64で成功していたrelease testが常に`linux_arm64` archiveを実行するため、GitHub-hosted `linux/amd64`では実行形式不一致になるhost portability defectへ局在した。実行hostの`GOOS/GOARCH`に対応する4 target中のarchiveを選ぶfail-closed testへ修正し、publisher requestを保守commitのsubjectと`update` actionへ一致させた。次にfull publisher gate、broker更新、public main CIを同一maintain runで確認する。

## Review findings

### 2026-07-22T01:55:00Z — three-perspective pre-publication review

- 利用者視点: clean `git archive`からbinaryをbuildし、clean=`0`、startup contaminationのtext／JSON=`1`、command not found=`2`、invalid zero timeout=`2`を確認した。DiagnosticとJSONにraw startup payloadはなく、first useful outputは5分以内だった。READMEのinstall、quickstart、Action、rollback、Windows非対応、sandbox境界は明記されている。
- Maintainer視点: final HEAD `58689bd`でpublisher gate、git fsck、format、unit／integration、race、vet、ShellCheck、actionlint、4-platform二重build／checksum、35 file／236722 byte payload、clean quickstart 13秒、pinned alternatives比較を通過した。Go runtime dependencyは0、npm comparison dependencyはtest-only、audit high／critical 0である。
- Security reviewer視点: raw payload非表示、shell-free argv、timeout／line／total／diagnostic cap、dedicated process group、secret pattern、credential-like path、MIT markerを確認した。Pinned npm比較は`--ignore-scripts`かつrelease／Action非同梱だがdeprecated transitive packageとmoderate advisory 9件を持つためtest-only boundaryを維持する必要がある。
- Distribution blocker: `.github/workflows/release.yml`は`go-version: '1.26.x'`だがpublisher gateはchecksum固定Go 1.26.5である。Go binaryはtoolchain versionへ依存するため、初回releaseと将来のmanual／repository dispatch repairで解決patchが変わると、同じtag／`SOURCE_DATE_EPOCH`でもasset byteが変わり得る。Release eventと各repair dispatchから同じreproducible packageを作るcontractを満たさない。
- Test blocker: Acceptance 7が要求するexit 0／1／2とtext／JSON stable schemaのgolden testは存在せず、`cmd`配下の`*_test.go`とgolden fileはいずれも0件である。Acceptance 8のcommand not found、broken pipe、signal terminationもregression suiteに無く、review中の手動command-not-found成功だけでは将来の回帰を阻止できない。Action smokeも0／1のみでinvalid inputのexit 2を固定していない。
- 判定: Runtimeの主要purity pathは動作したが、reproducible repair contractと明示acceptanceの自動回帰gateが未充足であるため`publish-ready`を拒否し、project stateを`building`へ戻した。次buildはrelease workflowをexact Go 1.26.5へ固定し、CLI golden 0／1／2、JSON schema、command-not-found／broken-pipe／signal、Action exit 2を追加してpublisher gateを再実行する。

### 2026-07-22T02:22:20Z — repaired three-perspective pre-publication review

- 利用者視点: Read-only fresh `git archive`からGo 1.26.5でbinaryをbuildし、clean=`0`、startup contaminationのtext／JSON=`1`、command not found／invalid timeout=`2`を14秒で確認した。Versioned JSONのtop-level 7 fieldとdiagnostic fieldを検査し、raw startup payloadがtext／JSONへ出ないこと、READMEのinstall、Action、rollback、Windows非対応、sandbox境界を確認した。
- Maintainer視点: Clean HEAD `a623b48`でgit fsck、publisher gate、全Go test、core race、vet、format、ShellCheck、actionlint、Action 0／1／2、42 files／246,362 bytesのpublisher payload、4 targetの二重reproducible archive／checksum、clean quickstart 13秒、3 pinned alternatives比較を通過した。CLI／core coverageは80.0%／86.1%、Go runtime external moduleは0である。
- Security reviewer視点: Shell-free argv、raw payload／stderr／environment／argument非表示、timeout／line／total／diagnostic cap、Linux／macOS dedicated process group、broken pipe／signal／cleanup child、credential-like path／secret pattern、MIT LICENSEを再確認した。Pinned npm比較treeはrelease／Action非同梱で、auditはhigh／critical 0、moderate 9。Lock metadataで欠落表示の`rechoir`は同梱LICENSEとpackage metadataでMIT、Inspector 4 packageは同一のexplicit transition LICENSEを保持することを確認した。
- Distribution／observability: READMEと`.kento-oss.json`はMatsuki Kento、`@kentomk`、automated AI agentを明示し、v2 requestはowner、candidate、3独立evidence、tested alternatives、30日直接採用metricを拘束する。CI／releaseはimmutable Action SHAとexact Go 1.26.5を使い、release、manual、repair dispatchから同じ4 archive＋`SHA256SUMS`を作る。GitHub-native配布にregistry credential blockerはない。
- 判定: 直前reviewの2 blockerを含むacceptance criteria 1〜12、clean install、failure／secret／license／CI／distribution／observability gateをfreshに通過した。重大な残存blockerはなく、test-only moderate advisoryと公開後のCI／release確認を明示riskとして、project stateを`publish-ready`へ進める。Publisher invocation、repository URL、外部採用はまだ0である。

## Maintenance

### 2026-08-08T09:01:00Z — composite Action toolchain reproducibility repair

- Local inspection found the Action still resolving Go from `go.mod`, while CI and release workflows were already pinned to exact Go `1.26.5`; a future module-version edit could therefore change the Action's source-built checker independently of the reviewed release path.
- The Action now pins Go `1.26.5` directly, README explains the shared toolchain contract, and the publisher contract rejects `go-version-file` in the Action while requiring the exact patch.
- Focused tests and the full publisher gate must pass before broker publication; current external adoption remains trial-only and does not justify feature expansion.

### 2026-08-08T08:10:00Z — Action working-directory failure boundary repair

- Current broker status remained healthy: public main CI succeeded, the maintainer inbox was empty, and the `v0.1.2` release retained all five assets.
- The composite Action now checks `working-directory` before building or starting the server. A missing or non-directory path returns a content-safe text or JSON error with exit `2`, rather than exposing the runner path through shell `cd` failure.
- Action smoke coverage fixes the exit code, JSON contract, path non-disclosure, and cleanup behavior. The README and changelog document the boundary; aggregate trial remains non-adoption evidence.

### 2026-08-08T06:25:30Z — published security policy alignment

- Broker status confirmed public main `3d7235acbe954bcee724cc15d0a091c025ceda52` with successful CI, no open Issue/PR, and complete v0.1.2 release assets.
- Synchronized `SECURITY.md` with the published v0.1.2 release and private vulnerability reporting, while retaining content-safe fallback guidance.
- Added publisher regression checks for supported version and stale unpublished wording; local publisher gate and public CI must pass before this maintenance step is complete.

### 2026-08-08T04:08:00Z — exit-code failure triage documentation repair

- Fresh release-status確認ではmain CI success、open Issue／PR 0件、v0.1.2の5 asset完備だった。11秒のclean quickstartと、invalid timeoutのexit `2`を実機で再確認した。
- READMEへ、purity violationのexit `1`とspawn／lifecycle／timeout／resource failureのexit `2`を分けるcopy-ready JSON triageを追加した。diagnosticがpayloadを転載しないことと、安全上limitを安易に緩めない境界も明記した。
- `tests/publisher-contract.sh`へfailure triage見出し、exit分岐、payload-safe診断説明のregressionを追加し、`go test ./...`、`go vet ./...`、publisher contract、clean quickstart、publisher gateを通過させる。

### 2026-07-23T14:20:00Z — CLI help discovery repair

- Clean local実行でtop-level `--help`がusageをstderrへ出してexit `2`となり、利用者とagentがcommand contractを安全に探索できない導入funnel障害を再現した。
- `--help`、`-h`、`help`、`check --help`、`check -h`をstdout／exit `0`へ統一し、job、usage、主要flag、exit code、`MSP001`を含む安定したhelp contractを追加した。不明commandは従来どおりstderr／exit `2`を維持する。
- 5つのhelp routeについてstdout、stderr、exit code、必須contractをunit testへ固定した。Focused test、実CLI smoke、full publisher gateは成功した。
- Public main CI成功後にv0.1.1をreleaseし、4 platform archiveと`SHA256SUMS`、distribution completeを確認する。Aggregate cloneはtrial signalに留め、直接採用へ数えない。

### 2026-07-28T04:34:00Z — published install trust repair

- 全6 managed repositoryをbrokerで確認し、current main CIはすべてsuccess、open Issue／PRは0、latest release assetは各5件だった。本projectの集計は14日windowでunique clone 27だがrelease download 0で、個票がなくverified external adoptionには数えない。
- READMEが公開済み`v0.1.1`と5 assetsに反して「初回release後」、`v0.1.0`、`FULL_COMMIT_SHA`を案内していたため、利用者とagentがcurrent installを選べないtrust defectとして修正した。
- 単一archiveと共有`SHA256SUMS`だけを取得した通常のinstallでは、manifest全体の検証が未取得3 archiveを要求して失敗する。対象archive行だけをfail-closedに抽出するLinux／macOS手順へ変更し、隔離directoryで両verifierを実行するrelease regressionを追加した。
- Full gate再実行では、test-only Inspector 1.0.0が新規high-severity `brace-expansion` DoS advisoryをtransitiveに含むためfail-closedに停止した。比較に必要なCLIだけを提供する`@modelcontextprotocol/inspector-cli` 1.0.1へ置換し、web UI／serverの無関係な依存treeをfixtureから除外した。SDK 1.30.0と`@hono/node-server` 2.0.12を安全な許容rangeへ固定し、fresh `npm ci`後のauditは0 vulnerability、3-tool比較は従来のfit判定を維持した。

### 2026-07-29T05:18:00Z — current release selection repair

- Public statusはmain CI成功、Issue／PR 0件、`v0.1.2`の4 platform archive＋`SHA256SUMS`完備を示す一方、READMEのcopy-ready install、release URL、archive名、Action pinは旧`v0.1.1`のままだった。
- Source install、release download、checksum例を`v0.1.2`へ、Action例をそのreleaseを生成したverified public main `c882d1f0c677d187911de2580d488f9841af1d2d`へ更新した。
- Publisher contractは旧`v0.1.0`／`v0.1.1` installとrelease URL、またはAction pinのdriftを拒否し、次回release後に同じ選定摩擦を見逃さない。
