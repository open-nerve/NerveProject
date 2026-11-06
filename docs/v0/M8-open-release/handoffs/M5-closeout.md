---
status: open
from: M5/closeout
to: M8
created: 2026-11-06
---

# M5 交给 M8：文件存储的部署、配置、备份与 CSP，附件事件的 Webhook，发布前的演练，仍开放的观察，首个投递任务的 CLI

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M5 设计 §13](../../M5-files/M5-design.md#13-输入交接与后续交接)、[P8 阶段记录](../../M5-files/reviews/P8-final-acceptance-review.md)与其中的 T10–T12；M5 的最终集成与 main CI 由 M5 收尾记录，不以本页替代。资产用途与宿主接入见 [M5 → M6 交接](../../M6-cycles-modules/handoffs/M5-closeout.md#M5-M6-01)。

<a id="M5-M8-01"></a>
## M5-M8-01 真实支持的存储与部署条件

- **local（默认）**：`files.driver: local`，字节在 `files.local.root`（默认 `./data/files`，相对进程工作目录）。浏览器经同源 `/static/local/{key}` 的 HMAC 签名地址读写，签名与期限用运行时钟、不用业务时间偏移；CSP 只有 `'self'`。部署须把根目录放在持久卷上，并且为本实例独占。默认部署按 README“部署”节一步步运行，由 F18 演示在三浏览器证明（`test:m5-demo`）。
- **s3**：经 S3 兼容接口直传对象存储。真实验证过的提供方只有固定的 SeaweedFS 4.52（P3），AWS 与其他品牌未验收。约束（`platform/config/files_s3.go`）：endpoint 为 http(s)，不带凭据、路径、查询或片段，端口 1–65535；IPv4 字面量可用，IPv6 字面量被拒（浏览器 CSP 无法表达），末段为数字或十六进制的 DNS 名被拒；region 1–64 字符；bucket 为 DNS 名而非 IP；prefix 以 `/` 结尾且为实例独占；`path_style` 默认 true。凭据只经 `NERVE_FILES__S3__*` 环境变量或 `_file` 路径给出（YAML 内联秘密被拒），二者择一。启动时核对桶**从未开启过版本控制**（删除后旧版本仍存在会使清理失效）。最低 IAM、精确 CORS、迁移前提与故障排查见 [`deploy/s3/README.md`](../../../../deploy/s3/README.md)。s3 模式下 `/static` 回答 404。
- **CSP**：只有 s3 的实际来源（SDK 解析后的 scheme 与 host，含虚拟主机式桶名）加入 `img-src` 与 `connect-src`（`platform/webui/csp.go`）；其余指令与 local 相同。
- **传输限额**：同时进行的传输至多 `max_concurrent_transfers`（默认 16，满额 503，没有等待队列）；每 IP 的传输速率桶 `transfer_rate`（每分钟 60、突发 20），独立于 JSON API 的速率桶；单次传输的总预算为 `transfer_grace + ceil(实际大小 / min_transfer_bytes_per_second)` 秒，默认最大文件 55 秒。业务 JSON 仍受 `/api/v0` 的 1 MiB/15 秒中间件约束，文件字节不经过它。

**何时验收：** M8 的部署文档与发布说明写入上述条件；若要宣布支持 AWS 或其他 S3 兼容服务，须先在该服务上跑通 P3 的双存储合同（`m5-s3` 与 `asset_s3_*` 测试）。

<a id="M5-M8-02"></a>
## M5-M8-02 配置

`files.*` 的默认值在 `server/configs/config.yaml`，dev/prod/test 都不覆盖；校验在 `platform/config/files.go`。

| 键 | 默认 | 取值 |
|---|---|---|
| `upload_ttl` | 15m | 10s–15m，新上传授权的寿命 |
| `operation_timeout` | 10s | 1s–10s，确认/复制的外部 I/O 段 |
| `cleanup_interval` | 1h | 1m–24h，`asset.cleanup` 的周期 |
| `cleanup_batch` | 100 | 1–1000 |
| `cleanup_budget` | 10s | 1s–30s；`asset.cleanup` 与共享清理 `cleanup.purge_deleted` 每次运行都用这个预算 |
| `size_limit` | 5242880 | 1–104857600；实例配置以 `file_size_limit` 报告，超限 413 `asset.too_large` |
| `driver` | local | local 或 s3 |
| `local.root` | ./data/files | 非空 |
| `min_transfer_bytes_per_second` | 131072 | 1024–2^30 |
| `transfer_grace` / `transfer_idle_timeout` | 15s / 30s | 大于 0，至多 1m |
| `max_transfer_duration` | 15m | 至多 1h，且不短于 grace + ceil(size_limit / 最低速率) |
| `max_concurrent_transfers` | 16 | 1–256 |
| `transfer_rate` | {60, 20} | 各至少 1 |

命名提醒：共享清理的批量与周期在 `cleanup.*`，它的每次运行预算却在 `files.cleanup_budget`（M5/P2 复用）。M8 写配置说明时把这一点写清，或在有理由时改名并迁移配置。

**何时验收：** 发布的配置参考与样例逐项一致；`nerve serve` 启动时记录的有效配置（秘密已遮蔽）与文档一致。

<a id="M5-M8-03"></a>
## M5-M8-03 备份、恢复与实例独占

- 文件由数据库中的 `file_assets` 行与存储中的对象共同构成；已确认的对象不可被旧上传地址改写（封存复制），对象键只由行决定。
- `asset.cleanup` 的对象阶段会删除存储中**比运行时钟早 24 小时以上、且没有行引用**的对象（`asset/app/cleanup_inventory.go`）。因此：把恢复出的数据库、预发或演练实例指向生产的根目录或 S3 前缀，会删掉生产中在该数据库之后上传的文件。根目录与前缀必须为每个实例独占；恢复演练使用独立的存储副本。
- 只恢复数据库而缺少对象时，引用这些对象的读取会失败；只恢复对象时，没有行的对象在 24 小时后被清理。S3 桶不能开版本控制（启动检查），备份不能依赖桶版本。删除 60 天后（`cleanup.purge_after`）行与对象被物理清理。

**何时验收：** M8 的备份/恢复手册给出数据库与存储的一致备份顺序，并以一次真实恢复演练核对：恢复后的读取、清理不删除仍被引用的对象、独占约束写入部署检查。

<a id="M5-M8-04"></a>
## M5-M8-04 附件事件与 Webhook

事件 `asset.attachment_added`、`asset.attachment_deleted` 的载荷、发出点与事务边界见 [M5 → M7 交接](../../M7-collaboration/handoffs/M5-closeout.md#M5-M7-02)：只有已绑定的普通工作项附件发出；在写入事务内同步投递，没有 outbox；载荷只有 ID，没有文件名、大小或地址。Webhook 订阅它们时须在同一事务投递 River 任务，由任务在事务外发出请求；载荷不得携带签名地址或文件内容，接收方需要文件时经 API 按当前权限读取。

**何时验收：** Webhook 开放附件事件时：增删各一次投递、回滚不投递、重试幂等（事件 `ID` 为 UUIDv7）、载荷不含签名地址。

<a id="M5-M8-05"></a>
## M5-M8-05 发布前运行、不在默认 CI 中的验收

默认 CI 的 e2e 作业已用约 21.2/20.5 分钟，预算 25 分钟（P5 经批准由 20 改为 25）；server 作业中 bootstrap 包在 CI 上为 756–978 秒（包期限 20 分钟、作业期限 25 分钟，P8 观察型世界的启动任务等待使其本地由约 650 秒增至约 820 秒）；新增默认故事前先核余量，调整预算或拆分作业须负责人批准。下面这些因此留在默认 CI 之外，发布候选上逐项运行并保留原件：

- `pnpm --filter @nerve/e2e test:m5-demo`：F18 默认生产部署演示（空库、Node 生成的签名密钥、`migrate up`、`users create`、重启后文件仍在），三浏览器。
- `test:m5-slow`：页面慢流（大于 1 MiB、超过 15 秒的附件上传），Chromium，local 与 S3。
- `test:m5-profile`、`test:m5-content`、`test:m5-s3`：资料页、内容页与 S3 的三浏览器矩阵（默认 CI 只跑 Chromium，Firefox 只跑少数故事）。
- `NERVE_FILES_DRILL=1 go test -count=1 -run '^TestFilesCleanupDrill$' ./internal/bootstrap`：全链路删除与清理演练，local 与 s3。
- `NERVE_PURGE_SCALE=1 go test -p=1 -count=1 -timeout=30m -run '^TestPurgeScale$' ./internal/bootstrap` 与 `NERVE_MEASURE_SCALE=1 go test -p=1 -count=1 -timeout=40m -run '^TestTheDeletionsAtTheScaleTarget$' ./internal/bootstrap`：M4 的两项规模测试；P8 在 M5 代码上的结果：660,026 行清理一次运行 5.0 秒；六个 66 万行项目的删除，单项目 7.6 秒、五项目并发 14.3 秒。

**何时验收：** 每个发布候选；结果写入发布记录。

<a id="M5-M8-06"></a>
## M5-M8-06 仍开放的观察与已知限制

- 复制为 Markdown 省略图片（签名地址不外泄）。
- 删除附件的回答丢失、核验得 404（文件确已删除）后，附件区标题的数字在下一次规范读取前仍是旧值（P8 T5 的取舍：这条路径不触发工作项的权威重读）。
- 规划器统计陈旧时，物理清理仍有界并续跑完成，但更慢（P8 实测 6 次运行合计 52 秒，统计新鲜时 5.0 秒）；部署中 autovacuum 照常运行即可。
- 五个项目同时软删时，邻项目写延迟上升（M4 记最大 576ms，P8 在 M5 代码上最大 68ms）；多热项目、多进程的全局公平未证明（M4 交接原样保留）。
- WebKit 的 F8 页面故事偶发“点击选择文件后没有 filechooser 事件”（P7b 与 P8 各一次，隔离重复从未复现，判为自动化层间歇）；默认 CI 不跑 WebKit。
- `TestAllPeriodicJobsRun` 的独立清单未含 `asset.cleanup`（其启动运行由十余个默认测试覆盖）。
- 启动时运行的周期任务与锁测试：服务启动时运行的五个周期任务通常在应用启动后1–3秒才完成，晚于测试世界建好，可能插到测试自己持有的锁后面（`issue.trim_description_versions` 逐个锁住所有项目，`asset.cleanup` 的清点阶段锁住存储中每个对象的资产行）。P8 在 CI 上遇到三次，现在观察型世界（`newObservedIssueWorld*` 等带请求追踪的 `newIssueWorldOnRequests`）交出前等这些任务各完成一次，另两项测试各自等待；其他世界构造仍靠各测试自己的等待（让所有世界都等会使 bootstrap 包的时间翻倍）。若再出现锁测试的阻塞者不是测试自己的夹具，先查是不是启动任务；缩短启动任务的延迟需要调 River 的取任务间隔，属平台配置。
- CI 上的 Chromium 网络变化：P8 的四次完整 e2e 运行中有三次，一个 [s3] 页面故事的浏览器里所有在途请求（包括发往 127.0.0.1 的页面脚本和 API）同时以 `net::ERR_NETWORK_CHANGED` 失败。推断原因是每个 [s3] 故事自起自停 SeaweedFS 容器，Linux 运行器上的容器启停增删宿主网络接口。P8 按负责人决定改为整次运行共用一个在浏览器之前启动的 S3 服务、每个故事一个桶，S3 夹具自身的生命周期测试在浏览器之前跑完（见 P8 阶段记录 T14）。仍开放：页面脚本加载失败后应用停在启动画面，既不提示也不重试；若以后再出现这一错误，先查运行期间是否有容器启停。
- M4 的 C12 (review) 页面故事（`e2e/stories/issue/review-interactions.spec.ts` 中按布局生成的 “group controls fold, reopen and create through native keyboard focus”）2026-11-06 在 P8 最终候选的 CI 上首次失败一次：“Create new work item”弹窗打开约 150–500 毫秒后按 Escape，5 秒内弹窗没有关闭。P8 对这个弹窗只改了关闭副本时的提示文字，不涉及 Escape 与普通新建；本地两种布局各重复 20 次全部通过，原因未查实。负责人决定重跑后通过，并要求记入本交接。若再出现，先看失败时弹窗内的焦点位置与 Escape 的处理者。
- CI 的 e2e 作业余量：2026-11-06 两次运行的“安装 Playwright 浏览器”一步因运行器 Ubuntu 镜像的软件包下载停滞（单包停顿约 1–3 分钟，一次这一步用了 22.5 分钟）而耗尽作业的 25 分钟期限被取消，同一输入重跑后通过；绿色 e2e 作业已用 23–24 分钟（最慢一次 24.1 分钟，测试本身 21–22 分钟）。浏览器的系统依赖每次运行都重新下载；缓存它或调整作业期限属 CI 配置，调整预算须负责人同意。

**何时验收：** 发布说明列出前两项限制；其余在 M8 的性能与 CI 工作中复核或关闭。

<a id="M5-04"></a>
## M5-04 首个投递任务的 CLI（原文转交）

下面完整引用 M4 交给 M5 的 M5-04，其中又完整引用 M2 原第 1 节；相对链接按原出处解释，原文不改。可点击来源：[M4 → M5 原交接](../../M5-files/handoffs/M4-closeout.md#M5-04)。

<!-- M4-M5-04-BEGIN -->
````text
## M5-04 首个投递任务的 CLI

下面完整引用 M2 原第 1 节；其中相对链接按原出处解释，原文不改。可点击来源：[M2 → M4 原交接](../../M4-issue-core/handoffs/M2-closeout.md)。

<!-- M2-SECTION-1-BEGIN -->
```text
## 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）

- **负责人的要求**：2026-09-26 批准把它推迟到 M4 时说，"做好记录，以后别漏了"。M4 用不到它，也要把这一节原样转进下一个 M 的交接，不能随本交接关闭而丢掉。
- **做什么**：第一个"用例会投递 River 任务、又能从命令行触发"的命令加入时，`platform/jobs` 提供一个只投递的客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`），`bootstrap` 的命令组合把它交给这些用例。
- **为什么**：命令行的组合只有连接池和 `identity` 的管理用例，没有 River 客户端（`server/internal/bootstrap/users.go:22-26`，M2 设计 3.17）。服务用的客户端（`server/internal/platform/jobs/jobs.go:70` 的 `New`）配队列、启动 worker，命令行不能用它。用例在事务里用 `InsertTx` 投递（M2 设计 3.15）；命令行上没有客户端时，任务要么没投递，要么用例在运行中失败。M2 的命令都不投递任务，所以推迟（M2 设计 3.15、3.17、13.2）。
- **可能早于 M4**：M3 给停用加端口（[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 6 节），`nerve users deactivate` 也走这个用例；端口的实现投递任务时，这一项提前到 M3。
- **建议的守法**：投递任务的端口在用例的构造参数里必填，命令的组合漏了它，在构造时就失败，而不是运行到一半；`server/internal/bootstrap/users_test.go` 这样的组合测试覆盖每个命令。
- **关闭条件**：M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。

```
<!-- M2-SECTION-1-END -->

**当前事实与责任：** 只投递的客户端已在 `platform/jobs`（`NewInserter`，M4/P6）；M3、M4 没有投递任务的命令，现有 CLI 组合无需构造它。**第一个投递任务的命令加入时，把它交给命令行的组合**，按 §3.20 显式声明投递的 kinds，启动时验证对应 worker。

**何时验收：** 首个投递任务的 CLI 提交中完成。真实命令经运行角色执行，验证 `river_job` 的 kind/args、提交留任务、回滚不留任务；少登记一个 worker 的负例必须失败。不能把服务端已有 NewInserter 当 CLI 已接通。
````
<!-- M4-M5-04-END -->

**M5 结束时的事实：** M5 仍没有投递任务的命令。CLI 的组合（`bootstrap/users.go`、`workspaces.go`、`commands.go`）只构造连接池与管理用例，不接事件发布者；`jobs.NewInserter` 唯一的非测试调用在 `bootstrap/app.go`（serve）。

**何时验收：** 首个投递任务的 CLI 提交中完成，验收同上文原文；M6、M7、M8 谁先加入这样的命令，谁完成并在另两份交接中写明。
