# M4/P7 生产变异矩阵

## 范围与可复核性

Task15在 `7b44d2ef7db03ab922b7a289fabdfe509951bb7d` 对既有原始证据逐项取证，选取 **21项不同实际生产补丁**，跨地址、网络、权限、锁、事务、guard、注册七类。没有重新执行历史变异，也没有把同一补丁的重复故事执行另算一类。每项历史真实红exit=1、恢复绿exit=0；T15最终正常源码全门禁见[行为矩阵](P7-issue-links-behavior-matrix.md)。

原始证据根 `E=.superpowers/sdd/P7-issue-links`（ignored）。`E/task-15-evidence/mutation-audit.json` 保存完整red/green argv、cwd、RUN计数、每次原始日志/元数据/补丁SHA256、原/变异/恢复/当前源码SHA。T15逐项读原始失败断言并核文件存在、exit与真实RUN，所有18个生产目标文件当前SHA均等于历史原/恢复SHA。下表不是最终HEAD上重新制造21次红。

历史开发过程中的HEAD不一定包含当时未提交源码；有原始HEAD字段时照录，没有时标“未录”，只给交付锚（用于定位历史，不冒称运行HEAD）。真正执行内容由每次运行source manifest和下列完整SHA识别。T5生产未变但后续组合测试补了后台完成同步；T6 policy_test随后仅注释/映射端点守卫加强；T7只用IDNA修后的fix-1；T9只用final-*；T13只用最终审计断言补齐后的M*-final-*。这些历史测试不等同最终测试字节，最终正常源码执行承担最终组合兼容性。

## 生产补丁、实际断言与层

计数为顶层/含子用例。除两项T6恢复运行 `TestPolicy` 全表5/200（覆盖原红选择）外，每项red/green argv相同。T6清单重叠变异仅在清单层被杀死，端点仍被父prefix拒绝，绝不推断HTTP/socket行为改变。

| 变异 / 类别 | 实际生产补丁 | 杀死它的实际断言 / 层 | 红→绿RUN | 源键 |
|---|---|---|---|---|
| `M13-precheck` / 锁 | 忽略预检错误继续进事务 | precheck acquired write transaction；真实PG/API组合 | 1/2→1/2 | S12 |
| `M14-clock` / 锁 | 创建时钟移到锁前 | clock read before project lock；真实PG/API组合 | 1/2→1/2 | S10 |
| `M15-author` / 权限 | 改删增加作者限制 | 管理员兼guest跨作者被错误403，期望200；真实PG/API组合 | 1/17→1/17 | S13,S11 |
| `m16-only-private` / 地址 | 仅以IsPrivate代替完整特殊地址表 | 100.64.0.0/100.127.255.255得到nil而非ErrBlocked；policy单元 | 1/4→5/200 | S16 |
| `m17-missing-overlap` / 地址 | 删192.0.0.9/32独立记录 | 独立IANA清单缺项；父/24仍阻断端点，不声称socket或HTTP行为改变；独立IANA清单 | 1/1→5/200 | S17 |
| `M19-first-only` / 网络 | 只检查首个DNS地址 | mixed DNS变成err=nil/dial=1；应blocked/0；outbound本地网络/私有dial | 1/2→1/2 | S15 |
| `M19-second-dns` / 网络 | 拨号前再次DNS | canonical hostname lookups=2；应1；outbound本地网络/私有dial | 1/2→1/2 | S18 |
| `M20-no-redirect-check` / 网络 | 第二跳绕过policy（双文件） | DNS仍2，dials=2；应blocked、dials=1；outbound本地网络/私有dial | 1/2→1/2 | S15,S18 |
| `M20-sixth-allowed` / 网络 | 允许第六次跳转 | GET=7；应初始加5跳共6；outbound本地网络/私有dial | 1/4→1/4 | S15 |
| `final-M25` / 网络 | 图标改用直接HTTP GET | userinfo listener在正对照后连接/请求增加至2/2、3/3且危险图标落入metadata；真实本地HTTP crawler | 1/5→1/5 | S1 |
| `final-M26-svg` / 网络 | 允许SVG MIME | SVG dataURL与预期两个icon槽null不等；真实本地HTTP crawler | 1/13→1/13 | S1 |
| `final-M26-bytes` / 网络 | 图标限额放宽至1MiB | 64KiB+1实际进入metadata；真实本地HTTP crawler | 1/6→1/6 | S1 |
| `final-M27-budget` / 网络 | 图标改用原parent重置预算 | 约8秒超过6.5秒上界，恢复约5秒；真实本地HTTP crawler | 1/2→1/2 | S1 |
| `final-M27-log-url` / 网络 | 诊断DEBUG输出rawURL | 完整日志含query/userinfo/password哨兵及原URL；真实本地HTTP crawler | 1/19→1/19 | S1 |
| `final-M27-log-error` / 网络 | 诊断DEBUG输出原err | malformed Location哨兵和完整包装错误进入日志；真实本地HTTP crawler | 1/19→1/19 | S1 |
| `M30` / 事务 | InsertTx改Background丢原事务 | requires a transaction；实际提交/注册组合失败；真实PG/API组合 | 2/13→2/13 | S8 |
| `M31-error` / 事务 | 吞Enqueue错误 | 第二subscriber仍被调用calls=[second]；应[]；真实PG/API组合 | 1/2→1/2 | S9 |
| `M34-worker` / 注册 | Module.Jobs漏crawl worker | 独立目录与真实app报enqueued kind issue.crawl_link has no worker；真实PG/API组合 | 2/3→2/3 | S14 |
| `M35` / guard | 去精确URL守卫 | 旧A完成后metadata变成Stale preview A，应该{}；真实PG/API组合 | 1/2→1/2 | S6,S4,S2 |
| `M36` / guard | 标题UPDATE写回app旧metadata | 标题RETURNING/API为{}，丢并发抓取的新metadata；真实PG/API组合 | 1/2→1/2 | S7,S5,S3 |
| `M37` / guard | 去ID守卫，仅URL与未删除 | old crawl changed deleted link or same-URL replacement；真实PG/API组合 | 1/2→1/2 | S6,S4,S2 |

## 原字节恢复与最终兼容

每个源键的完整SHA256同时是原字节、恢复字节和Task15 BASE当前字节的SHA；不把源码恢复误说成某次构建二进制原字节相同。SQL变异M35–37均实际make gen-go后运行，query/adapter/生成物一起恢复。

| 源键 | 路径 | 原=恢复=当前 SHA256 |
|---|---|---|
| S1 | `server/internal/modules/issue/adapter/crawler/crawler.go` | `21b766c22eeac6ac0b5f52230aa0f7069dc846279e0356f4e26513fa78a8c544` |
| S2 | `server/internal/modules/issue/adapter/postgres/gen/link_metadata.sql.go` | `9ec8da20fe1827eec2f2f0dc3bff7a53ff23949a7f416970b10838c0a211f0d5` |
| S3 | `server/internal/modules/issue/adapter/postgres/gen/links.sql.go` | `3c3b4ef1bbc12f2ec4c34f656f3d3fb4a7f1d198da15489516ef7e4b1d4d5478` |
| S4 | `server/internal/modules/issue/adapter/postgres/link_metadata.go` | `dc22668b017b89c239a7066418e97ebed8b61fa1e355fb2a5d2f17ec5e002440` |
| S5 | `server/internal/modules/issue/adapter/postgres/links.go` | `fafea48b10af009e6c8752166fcc9089a7856b2b0972ed9f0a0ba0c4cd7be6aa` |
| S6 | `server/internal/modules/issue/adapter/postgres/queries/link_metadata.sql` | `ff08151855d1bd997f3041aa632ea6acaf976b77ca890467fdf63318da8aa535` |
| S7 | `server/internal/modules/issue/adapter/postgres/queries/links.sql` | `8bfc4b09023755e790a622470d00e7e2bec5aebfe17815cde584468aa5220940` |
| S8 | `server/internal/modules/issue/adapter/river/link_crawls.go` | `6d89293f41b225403adacdfd31473732384f7998436a1980dcb74f1d57ec562f` |
| S9 | `server/internal/modules/issue/app/crawl_on_link_change.go` | `2bc8782c1f6293fdfd17a951fbd48d3f437c039930d9134d9cb8994b406a27db` |
| S10 | `server/internal/modules/issue/app/create_issue_link.go` | `3a5e27c3716ebfbdb0b16febc2b16ee0fdd13732bf4a29d05433c9b7b75fe8c2` |
| S11 | `server/internal/modules/issue/app/delete_issue_link.go` | `fdc30d075a7bfc79724f3feb4d3d1088a4dd92f76013e39bf815c03204c56fcc` |
| S12 | `server/internal/modules/issue/app/lock.go` | `633125a1d4f973b715448ab33995691fe8bd615f3dfd94e4ed3af389b935692f` |
| S13 | `server/internal/modules/issue/app/update_issue_link.go` | `f518519be3154e807c28f58e16586b950732dcbf119e2e5d76439a647311a0ec` |
| S14 | `server/internal/modules/issue/module_jobs.go` | `1464d73627962acab1b290b5d49e44a2e26d705c88414ee64a8986b8267a710b` |
| S15 | `server/internal/platform/outbound/client.go` | `1aed9b8c8b345d90403778343b6a4a72b30ca5de34a75d0127ea5338cd9b1b52` |
| S16 | `server/internal/platform/outbound/policy.go` | `188a47310700b652381b6f19b0b4b165b5d156858df7c6e299b6260287f79298` |
| S17 | `server/internal/platform/outbound/special_ranges.go` | `95751d9bf49a312328d85da3b7a3e9694300110a29a363e4e71d835769b68c90` |
| S18 | `server/internal/platform/outbound/transport.go` | `583d820ab2f04bc27fd207a397827c23773d2b77c3099a9640ab3c84595e5dee` |

## 每次运行、补丁与变异SHA

以下路径均相对E；红绿原始stdout/stderr与同名JSON配套，JSON保存完整cwd（本worktree/server）、argv与计数。T6为目录的manifest/stdout.log；T12为目录的command.json/stdout。变异SHA按源键列全，恢复SHA按上表。T11实际before/after补丁由mutations.py和.original保存，未伪造历史diff。

### M13-precheck

- 历史HEAD：`be3b8d4e9c59d6a60e397242c6a21161428a39ea`；交付锚：`8f05b68aa8b3d564ea5b74e682cb0beb26d45f5d`。
- 红 / 恢复绿：`task-5-evidence/M13-precheck-red.json` / `task-5-evidence/M13-precheck-restored-green.json`；恢复证明：`task-5-evidence/M13-precheck-restore.json`。
- 实际补丁：`task-5-evidence/M13-precheck.patch`。
- 红命令：`go test -v -count=1 -p=1 ./internal/bootstrap -run '^TestIssueLinkWritesLockAndRecheck$/^precheck_uses_no_write_resources$'`。恢复绿同命令。
- 变异SHA256：S12 `11cfa51f4488ae637fddbcb36e2896776fb2b48def9a4acde805de15f3ae13e2`。

### M14-clock

- 历史HEAD：`be3b8d4e9c59d6a60e397242c6a21161428a39ea`；交付锚：`8f05b68aa8b3d564ea5b74e682cb0beb26d45f5d`。
- 红 / 恢复绿：`task-5-evidence/M14-clock-red.json` / `task-5-evidence/M14-clock-restored-green.json`；恢复证明：`task-5-evidence/M14-clock-restore.json`。
- 实际补丁：`task-5-evidence/M14-clock.patch`。
- 红命令：`go test -v -count=1 -p=1 ./internal/bootstrap -run '^TestIssueLinkWritesLockAndRecheck$/^clock$/^createIssueLink$'`。恢复绿同命令。
- 变异SHA256：S10 `567e881411f486d7acc187b069950c25eb3e6fd03ce3bce1202a8df938aafde8`。

### M15-author

- 历史HEAD：`be3b8d4e9c59d6a60e397242c6a21161428a39ea`；交付锚：`8f05b68aa8b3d564ea5b74e682cb0beb26d45f5d`。
- 红 / 恢复绿：`task-5-evidence/M15-author-red.json` / `task-5-evidence/M15-author-restored-green.json`；恢复证明：`task-5-evidence/M15-author-restore.json`。
- 实际补丁：`task-5-evidence/M15-author.patch`。
- 红命令：`go test -v -count=1 -p=1 ./internal/bootstrap -run '^TestIssueLinkHiddenAnswers$'`。恢复绿同命令。
- 变异SHA256：S13 `538e3dbebc587c29eed28af56aba9650db92799aa321b450862eb3a77eb94d9a`；S11 `0cdfffb800a6070ea599b0a1c819a8468686a59780ce77d2b979887ffd9ca9d0`。

### m16-only-private

- 历史HEAD：原运行元数据未录；交付锚：`9fc903225b77209081c79925895d4a5cc6483115`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-6-evidence/m16-only-private/manifest.json` / `task-6-evidence/m16-only-private-restored/manifest.json`；恢复证明：`task-6-evidence/m16-only-private-record.json`。
- 实际补丁：`task-6-evidence/m16-only-private-record.json`。
- 红命令：`go test -count=1 -json ./internal/platform/outbound -run '^TestPolicyEndpoints$/^100[.]64[.]0[.]0$/^10$'`。恢复绿：`go test -count=1 -json ./internal/platform/outbound -run TestPolicy`（更广，含原红测试）。
- 变异SHA256：S16 `c9b71a63dda38c9687792bd5848296b83dc4020f70a1c8bf5e576d4f503d5979`。

### m17-missing-overlap

- 历史HEAD：原运行元数据未录；交付锚：`9fc903225b77209081c79925895d4a5cc6483115`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-6-evidence/m17-missing-overlap/manifest.json` / `task-6-evidence/m17-missing-overlap-restored/manifest.json`；恢复证明：`task-6-evidence/m17-missing-overlap-record.json`。
- 实际补丁：`task-6-evidence/m17-missing-overlap-record.json`。
- 红命令：`go test -count=1 -json ./internal/platform/outbound -run '^TestPolicyRegistryCoverage$'`。恢复绿：`go test -count=1 -json ./internal/platform/outbound -run TestPolicy`（更广，含原红测试）。
- 变异SHA256：S17 `105e43b048949118157a72efc454059d471f306158370ae2af8d5f74770cef86`。

### M19-first-only

- 历史HEAD：原运行元数据未录；交付锚：`128de7b620460d0161776b8dcc633f637cb746d5`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-7-evidence/fix-1/M19-first-only-red.json` / `task-7-evidence/fix-1/M19-first-only-restored.json`；恢复证明：`task-7-evidence/fix-1/M19-first-only-mutation.json`。
- 实际补丁：`task-7-evidence/fix-1/M19-first-only-client.go.diff`。
- 红命令：`go test -count=1 -v ./internal/platform/outbound -run '^TestOutboundPinsEveryHop$/all_candidates_before_dial$' -timeout 15s`。恢复绿同命令。
- 变异SHA256：S15 `d24f53a23796a826df3b349e8893857a044fd7f7d1ff7564af92b41d1c69909a`。

### M19-second-dns

- 历史HEAD：原运行元数据未录；交付锚：`128de7b620460d0161776b8dcc633f637cb746d5`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-7-evidence/fix-1/M19-second-dns-red.json` / `task-7-evidence/fix-1/M19-second-dns-restored.json`；恢复证明：`task-7-evidence/fix-1/M19-second-dns-mutation.json`。
- 实际补丁：`task-7-evidence/fix-1/M19-second-dns-transport.go.diff`。
- 红命令：`go test -count=1 -v ./internal/platform/outbound -run '^TestOutboundPinsEveryHop$/single_lookup_pinned_literal$' -timeout 15s`。恢复绿同命令。
- 变异SHA256：S18 `b8fd672fbc9e7574b0122bf904907f3de5071acaef39c3ea75c7aaa39bb7dac1`。

### M20-no-redirect-check

- 历史HEAD：原运行元数据未录；交付锚：`128de7b620460d0161776b8dcc633f637cb746d5`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-7-evidence/fix-1/M20-no-redirect-check-red.json` / `task-7-evidence/fix-1/M20-no-redirect-check-restored.json`；恢复证明：`task-7-evidence/fix-1/M20-no-redirect-check-mutation.json`。
- 实际补丁：`task-7-evidence/fix-1/M20-no-redirect-check-client.go.diff`、`task-7-evidence/fix-1/M20-no-redirect-check-transport.go.diff`。
- 红命令：`go test -count=1 -v ./internal/platform/outbound -run '^TestOutboundPinsEveryHop$/same_host_redirect_rechecks$' -timeout 15s`。恢复绿同命令。
- 变异SHA256：S15 `3d2ca28e3247c3c3cb9792426c165d50dad9b9165e5c7f476b461ea7d8277d0c`；S18 `674ce24eca1470497f98e2eaa9b8f586e8272cd16a0301172a4779e7b54febf9`。

### M20-sixth-allowed

- 历史HEAD：原运行元数据未录；交付锚：`128de7b620460d0161776b8dcc633f637cb746d5`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-7-evidence/fix-1/M20-sixth-allowed-red.json` / `task-7-evidence/fix-1/M20-sixth-allowed-restored.json`；恢复证明：`task-7-evidence/fix-1/M20-sixth-allowed-mutation.json`。
- 实际补丁：`task-7-evidence/fix-1/M20-sixth-allowed-client.go.diff`。
- 红命令：`go test -count=1 -v ./internal/platform/outbound -run '^TestOutboundLimitsAndHeaders$/redirect_limit$' -timeout 15s`。恢复绿同命令。
- 变异SHA256：S15 `da32e3f4e79a2213e4c7b1e0d2fbaf2a427c49e8abb1d2fcd2e1c5833580c199`。

### final-M25

- 历史HEAD：原运行元数据未录；交付锚：`15e2f251d6993491c71f5f0bc698a1b8e5c4beb0`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-9-evidence/final-M25-red.json` / `task-9-evidence/final-M25-restored.json`；恢复证明：`task-9-evidence/final-M25-restore.json`。
- 实际补丁：`task-9-evidence/final-M25.diff`。
- 红命令：`go test -count=1 -v ./internal/modules/issue/adapter/crawler -run TestCrawlerLimitsAndFailures/blocked_before_socket`。恢复绿同命令。
- 变异SHA256：S1 `0b9fd38eaec3cbdbfa66e53791a5ad9cb895fbe1675647f9b6fa9715bf1a0296`。

### final-M26-svg

- 历史HEAD：原运行元数据未录；交付锚：`15e2f251d6993491c71f5f0bc698a1b8e5c4beb0`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-9-evidence/final-M26-svg-red.json` / `task-9-evidence/final-M26-svg-restored.json`；恢复证明：`task-9-evidence/final-M26-svg-restore.json`。
- 实际补丁：`task-9-evidence/final-M26-svg.diff`。
- 红命令：`go test -count=1 -v ./internal/modules/issue/adapter/crawler -run TestCrawlerLimitsAndFailures/icon_mime`。恢复绿同命令。
- 变异SHA256：S1 `de0917b9768fd08e8c8a719b9a6be9d8cf376508ed279b6a9ab770dbd93a0022`。

### final-M26-bytes

- 历史HEAD：原运行元数据未录；交付锚：`15e2f251d6993491c71f5f0bc698a1b8e5c4beb0`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-9-evidence/final-M26-bytes-red.json` / `task-9-evidence/final-M26-bytes-restored.json`；恢复证明：`task-9-evidence/final-M26-bytes-restore.json`。
- 实际补丁：`task-9-evidence/final-M26-bytes.diff`。
- 红命令：`go test -count=1 -v ./internal/modules/issue/adapter/crawler -run TestCrawlerLimitsAndFailures/body_boundaries`。恢复绿同命令。
- 变异SHA256：S1 `20fc992b1d90c628b673f9f273810e483f045beaa59ec81d7e2b38a79708452a`。

### final-M27-budget

- 历史HEAD：原运行元数据未录；交付锚：`15e2f251d6993491c71f5f0bc698a1b8e5c4beb0`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-9-evidence/final-M27-budget-red.json` / `task-9-evidence/final-M27-budget-restored.json`；恢复证明：`task-9-evidence/final-M27-budget-restore.json`。
- 实际补丁：`task-9-evidence/final-M27-budget.diff`。
- 红命令：`go test -count=1 -v ./internal/modules/issue/adapter/crawler -run TestCrawlerLimitsAndFailures/shared_five_seconds`。恢复绿同命令。
- 变异SHA256：S1 `ee053d740fe53fb25eed0baca3ef27515c93cadfad891ac9606b35ee9d25458e`。

### final-M27-log-url

- 历史HEAD：原运行元数据未录；交付锚：`15e2f251d6993491c71f5f0bc698a1b8e5c4beb0`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-9-evidence/final-M27-log-url-red.json` / `task-9-evidence/final-M27-log-url-restored.json`；恢复证明：`task-9-evidence/final-M27-log-url-restore.json`。
- 实际补丁：`task-9-evidence/final-M27-log-url.diff`。
- 红命令：`go test -count=1 -v ./internal/modules/issue/adapter/crawler -run TestCrawlerLogsRedacted`。恢复绿同命令。
- 变异SHA256：S1 `a55e2af8bb891fecc8af7762676ddf039316666a1281e1e925a7bba55af91ae2`。

### final-M27-log-error

- 历史HEAD：原运行元数据未录；交付锚：`15e2f251d6993491c71f5f0bc698a1b8e5c4beb0`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-9-evidence/final-M27-log-error-red.json` / `task-9-evidence/final-M27-log-error-restored.json`；恢复证明：`task-9-evidence/final-M27-log-error-restore.json`。
- 实际补丁：`task-9-evidence/final-M27-log-error.diff`。
- 红命令：`go test -count=1 -v ./internal/modules/issue/adapter/crawler -run TestCrawlerLogsRedacted`。恢复绿同命令。
- 变异SHA256：S1 `d42484e223a57f4cadc51e9191e01b42dcc196c72bb7304de2601f15cf02192a`。

### M30

- 历史HEAD：原运行元数据未录；交付锚：`f2a6569b685c03541da24ec22b51b8a964d269b5`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-11-evidence/M30-red.json` / `task-11-evidence/M30-restored.json`；恢复证明：`task-11-evidence/M30-restoration.json`。
- 实际补丁：`task-11-evidence/mutations.py`。
- 红命令：`go test -v -count=1 -p=1 ./internal/bootstrap -run 'Test(IssueLinkSubscriberRegistration|LinkJobCommitRollback)$'`。恢复绿同命令。
- 变异SHA256：S8 `0ba0c42c5f6ca6f22a72681009b2c8fba369dd89d95efee85ffb9fec0c784793`。

### M31-error

- 历史HEAD：原运行元数据未录；交付锚：`f2a6569b685c03541da24ec22b51b8a964d269b5`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-11-evidence/M31-error-red.json` / `task-11-evidence/M31-error-restored.json`；恢复证明：`task-11-evidence/M31-error-restoration.json`。
- 实际补丁：`task-11-evidence/mutations.py`。
- 红命令：`go test -v -count=1 -p=1 ./internal/bootstrap -run 'TestLinkJobCommitRollback/enqueue_error$'`。恢复绿同命令。
- 变异SHA256：S9 `dbab7dc299f2584f7ede52bf9a33adf01e21ed68b94a149ec3787b4e3a200492`。

### M34-worker

- 历史HEAD：`f2a6569b685c03541da24ec22b51b8a964d269b5`；交付锚：`fdeadacf6d39340733d7e42f615229da3fdb6456`。
- 红 / 恢复绿：`task-12-evidence/M34-worker-red/command.json` / `task-12-evidence/M34-worker-restore-green/command.json`；恢复证明：`task-12-evidence/M34-worker-mutation/restore.json`。
- 实际补丁：`task-12-evidence/M34-worker-mutation/mutation.patch`。
- 红命令：`go test -json -count=1 -p=1 ./internal/bootstrap -run '^Test(NewAppCrawlJobCatalog|IssueLinkRuntimeJob)$'`。恢复绿同命令。
- 变异SHA256：S14 `329d85f0b7c3d2c8196f85e07533d4ed1e43e34a42fad613b9a18ae5c1fb9b05`。

### M35

- 历史HEAD：原运行元数据未录；交付锚：`a08777740f267ffcef45d0ee93c71a70de2d1b4c`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-13-evidence/M35-final-red.json` / `task-13-evidence/M35-final-restored-green.json`；恢复证明：`task-13-evidence/M35-restore.json`。
- 实际补丁：`task-13-evidence/M35.patch`。
- 红命令：`go test -json -count=1 -p=1 ./internal/bootstrap -run '^TestLinkCrawlURLInterleaving/URL_commits_before_old_crawl$'`。恢复绿：`g o ' ' t e s t ' ' - j s o n ' ' - c o u n t = 1 ' ' - p = 1 ' ' . / i n t e r n a l / b o o t s t r a p ' ' - r u n ' ' ''"'"'' '^' T e s t L i n k C r a w l U R L I n t e r l e a v i n g / U R L _ c o m m i t s _ b e f o r e _ o l d _ c r a w l '$' ''"'"''`（更广，含原红测试）。
- 变异SHA256：S6 `ec8f70a81012a2bc9bc03b45552e0066f8a88240f6967b2567f90ffe8a6a3dfc`；S4 `c8745a53fcbecb631bdeca98231943b329c36d15d6929ad4ea74d11bd6cd22ce`；S2 `4ca357e59677f6e4b300d835412036cf923d1fc6a698c0171b157ff7e698ee2e`。

### M36

- 历史HEAD：原运行元数据未录；交付锚：`a08777740f267ffcef45d0ee93c71a70de2d1b4c`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-13-evidence/M36-final-red.json` / `task-13-evidence/M36-final-restored-green.json`；恢复证明：`task-13-evidence/M36-restore.json`。
- 实际补丁：`task-13-evidence/M36.patch`。
- 红命令：`go test -json -count=1 -p=1 ./internal/bootstrap -run '^TestLinkTitlePreservesConcurrentMetadata/crawl_commits_after_title_read_before_title_write$'`。恢复绿：`g o ' ' t e s t ' ' - j s o n ' ' - c o u n t = 1 ' ' - p = 1 ' ' . / i n t e r n a l / b o o t s t r a p ' ' - r u n ' ' ''"'"'' '^' T e s t L i n k T i t l e P r e s e r v e s C o n c u r r e n t M e t a d a t a / c r a w l _ c o m m i t s _ a f t e r _ t i t l e _ r e a d _ b e f o r e _ t i t l e _ w r i t e '$' ''"'"''`（更广，含原红测试）。
- 变异SHA256：S7 `a2446b5f6f8d3304b0a1a2f7a0a0aaf8a1ceddc582254458aedf73db8154354b`；S5 `dc2cb936967ca5404b1df2552330cb537aed01c82bb3eabd91298abf062989a5`；S3 `804c8a9a45d6958f41f06f879cb8e620680c83caa3d47b50060d83049fa44390`。

### M37

- 历史HEAD：原运行元数据未录；交付锚：`a08777740f267ffcef45d0ee93c71a70de2d1b4c`，执行字节由source manifest与上表识别。
- 红 / 恢复绿：`task-13-evidence/M37-final-red.json` / `task-13-evidence/M37-final-restored-green.json`；恢复证明：`task-13-evidence/M37-restore.json`。
- 实际补丁：`task-13-evidence/M37.patch`。
- 红命令：`go test -json -count=1 -p=1 ./internal/bootstrap -run '^TestLinkCrawlAfterDeletion/same_URL_new_ID_rejects_old_result$'`。恢复绿：`g o ' ' t e s t ' ' - j s o n ' ' - c o u n t = 1 ' ' - p = 1 ' ' . / i n t e r n a l / b o o t s t r a p ' ' - r u n ' ' ''"'"'' '^' T e s t L i n k C r a w l A f t e r D e l e t i o n / s a m e _ U R L _ n e w _ I D _ r e j e c t s _ o l d _ r e s u l t '$' ''"'"''`（更广，含原红测试）。
- 变异SHA256：S6 `dcdfbf85e8bbaac0353f616bf0398215ddfda391a1a9fa63a60c35156e686500`；S4 `706e58651ae564558d47e2a785354ce42d475cf1dbb12c80c398b61f918c2ab6`；S2 `15e2659d64754a160a1fe2ff9d56275c2f40da6d60bddd868fd7c98fc4a79fed`。

## 计数排除与历史限制

- T12 M32/M33仅修改测试快照helper，属于测试守卫反例，不计生产变异；T7 carryover-test-guard也仅改测试helper，不计。
- T14再次执行M30证明I10实际201→500与恢复1/1，是已有同一生产补丁的故事归属证据，不给第22个生产变异名额。
- T6 m17-missing-overlap-endpoints历史163RUN仍绿；独立清单红精确保护192.0.0.9/32显式保留。删除父/24与删除子/32不是相同安全效果。
- 未选T7 IDNA修复前M19/M20；M21历史只证明origin absolute-URI错误，proxy fixture仍0，不在本矩阵声称发生代理连接。
- M25只证明图标绕outbound的实际本机危险请求，不将它扩大为全部DNS/IP策略证明；M19/M20私有dial将已验证目的地转本地，不称为真实私网socket。
- T5 M14-publish初次恢复绿为Ryuk INFRA失败，不能算通过；本次没有选该项。T8/T11两条计划selector文档Minor仍交T16，完整回归覆盖事实保留。
- T13 A→B→A按批准id+URL规则允许旧A晚覆盖同URL结果；M35只证明B不能被A覆盖，不虚构generation保证。
- T15不为降噪操作现有容器、不重写旧raw、不新增依赖；正常源码最终门禁与历史变异是两类证据。
