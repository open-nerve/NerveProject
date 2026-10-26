# M4/P6 领域事件与自动归档评审记录

日期：2026-10-26。状态：**T1–T16 实施并逐 Task 独评通过；历史内部整分支审查零新发现；分支 CI 发现的时钟断言已局部修复，聚焦复审及修复 HEAD CI、外部终审待办**。分支 `codex/m4-p6`；基线 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`。配套：[spec](../specs/P6-issue-events.md)、[plan](../plans/P6-issue-events.md)。

## 1. 评审方式

起步时按 [02-method](../../claude_to_gpt/02-method.md) 第3节采用紧凑计划，三个独立静态预检视角及修订复审只评文档与现状；其结果保留在第4节，不再冒充实施证据。之后 T1–T15 各自完成定位/影响回归、真实生产变异与恢复、中文实施报告及独立要求/质量评审，精确提交和命令索引在第2节。T16 正式记录与输入交接经独评发现一处措辞问题，修复提交 `6995d523` 已通过独立聚焦复审。此后唯一一次内部整分支审查覆盖基线至该HEAD的全部19提交/165文件，零新发现；外部终审另待负责人安排。

## 2. 验收标准逐条核对

实施基线为 P5 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`；其 [main CI 40257678959](https://github.com/open-nerve/NerveProject/actions/runs/40257678959) 只证明基线。下表 HEAD 为对应任务实施提交；每条命令均在对应 `task-N-report.md` 和证据目录有退出码/实际匹配/变异恢复，独立结论在 `task-N-review.md`。早期报告位于 worktree 的 ignored `.superpowers/sdd/P6-issue-events/`，不是 Git 跟踪的正文附件；最终证据归档按后续集成安排。

| Task / 实施 HEAD | 完成线的真实命令与结果 | 独评/边界 |
|---|---|---|
| T1 `d0ea1969` | `go test -count=1 -p=1 ./internal/platform/postgres ./internal/platform/jobs ./internal/bootstrap ./internal/modules/issue/adapter/river`，退出0；真实同事务提交/回滚/未提交不可见及无事务拒绝 | [spec §4](../specs/P6-issue-events.md#4-逐项验收与证据)；任务独评通过 |
| T2 `5de9238e` | `go test -count=1 ./internal/bootstrap`，退出0；`./internal/shared ./internal/modules/issue/app ./internal/modules/issue ./internal/archtest` 回归退出0 | 初次 focused 命令有空匹配，**不计**新 dispatcher 验证；后续实际匹配与包回归闭合 |
| T3 `98af1cb9` | `go test -v -count=1 -p 1 ./internal/modules/issue/app ./internal/bootstrap -run 'TestIssue(Created|Updated)Event'`，退出0；事件 race 五遍0 | 600秒描述窗口勘误见§3；独评通过 |
| T4 `eab93c6a` | `go test -count=1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(Issue(Deletion&#124;Archival)Events&#124;.*Deletion&#124;.*Archival)' -v`，退出0；137格权限矩阵 | 独评通过 |
| T5 `fcea0b95` | `go test -count=1 ./internal/bootstrap`，退出0（149.644秒）；app 回归及关系/父子定位实际匹配 | 独评通过 |
| T6 `99e45aed` | `go test -count=1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(CommentEvents|ReactionEvents|.*Comment|.*Reaction)' -v`，退出0；真实PG七条写路径回滚 | 独评通过 |
| T7 `8bf712b0` | `go test -p 1 -count=1 ./internal/platform/jobs ./internal/platform/config ./internal/modules/identity/adapter/river ./internal/bootstrap ./configs ./internal/archtest -v`，六包退出0；10/2队列隔离与两类漏worker变异 | 独评通过；归档interval由T10承担 |
| T8 `c01d9977` | `go test -p 1 -count=1 ./internal/platform/clock ./internal/platform/config ./internal/bootstrap ./internal/archtest`，退出0；服务/CLI业务时钟、River真时与负例 | 独评通过 |
| T9 `ad905df0` | `go test -p 1 -count=1 ./internal/modules/project/... ./internal/modules/issue/... ./internal/bootstrap ./internal/archtest`，退出0；SQL cutoff生产变异和`make gen-check`0 | deleted_at断言Minor由T10补并独评关闭 |
| T10 `e7e20464` | `go test -count=1 -v -p 1 ./internal/modules/issue/... ./internal/bootstrap ./internal/platform/config ./configs ./internal/archtest`，退出0（704顶层/5599 RUN）；JobTx锁超时race五遍0 | 活动/事件长度诊断Minor由T15补并独评关闭 |
| T11 `d9b9e21f` | `go test -count=1 -p 1 -v ./internal/bootstrap -run 'Test(AllPeriodicJobsRun|TheRuntimeRoleServesWithTheGrantsFile|TheGrantsFileCoversEveryRelationAndFunction|AutoArchive.*|IssueInserter.*|TheSessionCleanupRunsAsConfigured|CloseDoesNotWaitForAConnectionInUse|RunStopsTheJobsAfterHTTP)$'`，退出0 | 固定预期两任务 completed、真实运行角色与CLI隔离；独评通过 |
| T12 `7c3acaaf` | `go test -race -v -count=5 ./internal/bootstrap -run '^TestAutoArchive(AndStateChangeSerialize|UsesLockedPolicy|ReadsClockAfterLock)'`，退出0，19叶×5=95；无race/40P01 | 第二项目锁释放诊断Minor由T15补并独评关闭 |
| T13 `183ef1a7` | `make build`退出0；`pnpm --filter @nerve/e2e exec playwright test stories/smoke/clock-lifecycle.spec.ts --project=chromium`，6/6退出0 | 旧进程身份/附件两Minor由T15补；T13最终门禁仅报告，无完整rawlog |
| T14 `565fad12` | `make build`后I16定位1/1，完整`make e2e`149/149退出0；五个成功构建的真实变异被业务断言杀死 | 旧完成任务与跨写入扫描以微秒水位区别；独评通过 |
| T15 `4a489125` | `go test -race -count=5` 定向105 RUN；`GOFLAGS=-p=1 make test`、`make lint-go`、`make lint-web`、`make knip`、`make test-web`、`make e2e`、提交后`GOFLAGS=-p=1 make gen-check`依次退出0 | [16用例/12事件矩阵](P6-issue-events-publication-matrix.md)、[20变异矩阵](P6-issue-events-mutation-matrix.md)；四功能Minor独评关闭 |
| T16 `d6f341d2`、修复`6995d523` | 关键词、链接、diff、必要架构静态核对见任务报告；初评的一处SQL NULL/事件nil措辞问题经聚焦复审确认修复 | 任务独评完成；整分支结果见§5，分支CI仍待push后核对 |

T15 `make test` 是49个有测试包通过、16个无测试包；默认输出不列总测试用例数。`make lint-web` 54项成功，其中36缓存、18新执行；`make test-web` 18项成功，其中11缓存、7新执行；`make e2e` 是真实149/149（Chromium147、Firefox2）。七个被核文件的源码 SHA 与各全门禁执行时一致。20项生产变异中14项复用已核原始证据、6项T15新执行；没有以编译失败/INFRA充作业务检出。早期 testcontainers 可选 shell 探测日志、旧 Playwright `NO_COLOR`/`FORCE_COLOR` 警告照原样保留；T15报告末的 `14-color-environment` 在 `4a489125` 上同时移除两个变量后1/1、无警告，已闭环环境Minor，**不称全部历史输出 pristine**。

### 2.1 分支 CI 后的 clock 生命周期断言修复

首次分支 [CI 40291867930 / E2E job 121470998137](https://github.com/open-nerve/NerveProject/actions/runs/40291867930/job/121470998137) 在 `17b8a117dc084d2e8dc017b9f1dd68c0ccb79416` 的唯一失败是 `clock-lifecycle.spec.ts:39`：`startedAt=1793003729090` 大于 `birthtimeMs=1793003729089.3872`，其余148项通过（包括I16），不是W11失败。实际调用顺序一直是 `new Date → logFile → openSync → spawn → ready`。根因是测试把应用墙钟与文件系统创建时间当成严格执行顺序的共同标尺；Node明确指出stat时间精度依平台而异。现有日志不足以确定具体Linux内核、文件系统粒度或时钟校正机制，不能归因为已证实的浮点舍入或某一种FS行为。

本地诊断在已存在的临时日志上先取真实Date再以 `openSync(..., "w")` 打开，确认birthtime仍是原创建时刻，旧oracle为false。它证明旧判据不健全，**不是macOS复现本次Linux触发机制**。修复只为测试fixture增加可选 `now?: () => Date`（默认真实 `new Date`），以已有logFile回调记录启动前边界；独立故事断言初代与advance代均为采样→logFile，并返回对应采样值。正常切换故事保留真实墙钟范围、进程exitCode、业务偏移、DB隔离与清理；I16和独立附件处理原样保留。没有epsilon、取整、sleep、跳过或新测试框架。

验证原始证据位于 ignored `.superpowers/sdd/P6-issue-events/ci-clock-fix-evidence/`，完整中文报告为同级 `ci-clock-fix-report.md`。所有命令串行，移除NO_COLOR与FORCE_COLOR，使用既有浏览器缓存；每条JSON含精确argv、退出码、匹配、基线HEAD和源码SHA。

| 证据 | 实际结果 |
|---|---|
| `01-birthtime-counterexample` | exit0；正确调用顺序下旧文件时间oracle为false，仅作诊断 |
| `02-order-red` → `03-order-and-original-green` | 新故事先缺now事件明确断言红1/1、exit1；最小实现后新故事与原失败故事2/2、exit0 |
| `04-after-ready-mutation` → `05-order-restored` | 真正把fixture采样移到await startNerve/ready后；收到`[log:0, now]`而期望`[now, log:0]`，1/1失败、exit1；逐字恢复后1/1、exit0，SHA一致 |
| `06-clock-and-i16` | 7个clock生命周期故事与I16共8/8、exit0；I16保留跨写入任务与新任务的数据库微秒水位诊断 |
| `07-types` / `08-lint` / `09-format` | @nerve/e2e类型、0-warning lint和110文件格式检查均exit0 |
| `10-make-e2e` | 唯一一轮完整make e2e，150/150（Chromium148、Firefox2）、exit0；Playwright41.8秒，命令43.55秒。Web构建11/11缓存命中，E2E全为本次真实执行；无颜色警告 |

此次只有两个E2E源码文件和spec/plan/review变更，Go/业务代码未变，沿用T15对应Go/race门禁证据而不伪称重跑。历史内部整分支审查范围与结论保留；当前修复由主控接续独立聚焦复审和修复后精确HEAD公开CI，不将本地150/150替代CI结果。

## 3. 执行中的决定

局部实现决定及若判断错误的代价逐条记录如下，依据 ignored `progress.md` 中全部 Ruling；它们没有重开已批准的 D3/D4/D7。任务中发现的测试Minor及环境Minor的闭环见§5。

| 裁定 | 实际处理与理由 | 若错的代价/核对 |
|---|---|---|
| 起步状态 | 起步“未实施”只属准备历史，用户已授权后续T1–T16实施 | 若把旧时态当现状，会错误关闭或重做任务；本表与§2按真实进度更新 |
| 门禁分配 | 各Task跑定位/影响包/变异，完整仓库门禁集中T15 | 若局部回归遗漏跨层问题，T15的完整Go/Web/E2E门禁与矩阵兜底 |
| 集成授权 | 合并/mainCI/清理仍待单独授权，P6分支CI也不能用P5基线CI替代 | 若越权或误认CI会造成错误交付；§7明列后续门 |
| T3事件目录 | 类型常量集中 `issue/domain/events.go`，模块 `EventTypes` 消费它，app不反向导入模块 | 若层次不符只需局部命名/依赖重构；架构测试和T3评审核过 |
| T3恰好600秒 | 继承P4描述版本窗口，599秒与恰好600秒合并，600秒+1微秒开新版本；初错测试期望不算生产红 | 若判错只影响测试/文档；P4原行为及T3实际用例/变异守住 |
| T7阶段接线 | 四参runner先迁移identity；有真实autoarchive才在T10加issue Jobs/EnqueuedKinds，T11固定外部期望证明两者，不造空占位 | 若漏最终声明会使周期任务不运行；T11真实completed和漏worker变异核对 |
| T7归档周期 | worker数与身份迁移归T7，`issue.auto_archive_interval`默认24h/test2s及校验归T10 | 代价是间隔验证延后至真实worker；T10配置/组合已验证 |
| T11构造顺序 | 所有声明任务的模块构造后创建一次runner；instance/router按原顺序在runner后构造，服务运行时再Start | 若误写为“最后一个factory”，可能诱发无须的组合重排；T11检查生命周期/关闭分支 |
| T14诊断屏障 | 临时读锁屏障仅协调已运行worker，业务数据准备/写入仍全走API，SQL只作诊断断言 | 若越界会弱化I16 API证据；T14真实跨写入尝试水位变异和独评核查 |
| T15颜色环境 | 后续Playwright命令同时移除`NO_COLOR`与`FORCE_COLOR`，旧日志不改 | 若只在父进程删其一，子进程仍警告；`14-color-environment`实际1/1无警告，产品代码无变 |

## 4. 预检发现与处理

独立首审来源：事件/事务评审 `preflight-events.md`（E1–E3），平台/任务评审 `preflight-jobs.md`（J1/J2），范围/证据评审 `preflight-scope.md`（S1/S2）。报告保存在主目录 ignored `.superpowers/sdd/P6-issue-events/`；作者未担任这三份预检的评审者。首审合计 High 0、Medium 3、Low 4，全部局部，均接受；未发现需要重新裁定的架构问题。

| 来源 / 级别 | 问题与裁定理由 | 修订 / 错判代价 | 复审 |
|---|---|---|---|
| E1 / Medium | 请求busyTx会把55P03转ServerBusy，违背后台保留错误链；接缝分开 | issue.Deps.JobTx接同LockTimeout的raw Tx，bootstrap共用底层构造；请求映射不变。T10真实PG错误/请求503双对照。代价仅额外消费方端口，不加锁或SQL旁路 | ADDRESSED，独立复审通过 |
| E2 / Low | relation/reaction的文本与ID槽不明确，逐值测试可各自自证 | spec四槽表：容器只有ID，文本nil；T5/T6错槽变异。代价为后续消费者必须遵循固定shape | ADDRESSED，独立复审通过 |
| E3 / Low | Event.ID与Entity.ID、批量/连续事件身份未区分 | app生成非零独立ID；跨订阅者不变，同实体连续写/批量互异；置零/复用Entity.ID变异。代价是每事件一次UUID生成 | ADDRESSED，独立复审通过 |
| J1 / Medium | River AddWorkerSafely不会拒绝nil/typed-nil worker，登记集合可能假完整 | New在真实登记/启动前拒绝无效registration，两种nil反例要错误接受变异而非运行panic；不反射River私有字段。代价为平台少量输入校验 | ADDRESSED，独立复审通过 |
| J2 / Low | T7改New四参与worker配置，却把旧bootstrap调用留到T11 | T7同时最小迁移全部调用/配置/测试，T11只最终汇总；代价为T7包含必要接线，保证独立可编译验收 | ADDRESSED，独立复审通过 |
| S1 / Medium | recent对照可能未经过任何任务扫描，RunOnStart的completed可提前满足 | 最后recent写后取本库微秒真实水位，要求attempted_at晚于它且completed；保留spawn门/业务断言；M4 I16与时钟说明同步。代价为最多等下一周期，测试不用固定sleep | ADDRESSED，独立复审通过 |
| S2 / Low | P5当前表/计划仍说保留已删除分支 | 标明实施位置/交付门为历史，最终授权记录优先；不改旧过程事实。代价仅时态澄清 | ADDRESSED，独立复审通过 |

聚焦复审确认上述7项的实质修订闭合，另发现两处 Low 文档接缝并已修订：

- E4/J3（同一处）：T10错误链验证改用 `postgres.LockNotAvailable`，遵守03对锁等待测试的规则；定位命令显式加入既有 `TestABusyTransactionIsServerBusy` 和 `TestAProjectTooBusyForAWriteAnswersServerBusy`，确保用户503对照确实运行。
- S3：M4设计I16的数据库对照列与spec/plan统一：旧非终态在偏移前创建，recent终态在偏移后经API创建/更新，分别证明状态和时间过滤；不让同一新行同时承担两个性质。

这两项同属局部文档修订，未改变产品范围；最终聚焦复核已确认两项闭合。

独立范围评审另核 P5 双亲与tree相同、公开CI三任务全绿、6,517文件/184,269,398字节SHA均符以及远端P5不存在。预检当时本地关键词守卫72规则/23例外无命中、架构测试通过；这些仅证明起步状态，不证明P6运行功能。

### 最终预检结论与本地准备检查

三个独立聚焦复审 `preflight-events-rereview.md`、`preflight-jobs-rereview.md`、`preflight-scope-rereview.md` 均为静态 APPROVED，High/Medium/Low 未关闭数均0。原始报告与文件SHA留在同一 ignored 台账；正式记录汇总9项局部修订（首审7、复审新增2），无架构待决点。

- `node tools/keywords.mjs`：退出0，72规则、23例外、无命中/过期例外；phase仅由M4/P5改为M4/P6。
- `go test -count=1 ./internal/archtest`：退出0，包测试1.607s；完整命令用时2.299s，stderr空。证据为台账 `verification/archtest.json` / stdout / stderr。它只验证预检当时的基线架构守卫；当时P6运行行为尚未实现。
- 准备期文档自检确认16个Task顺序完整、当时16条产品验收全部未勾，新增本地链接有效，`git diff --check` 无错误；无产品/迁移/依赖文件变动。M4原有M5–M8未来closeout链接不提前创建。
- 仅文档与关键词标记不重跑已经精确通过的P5完整门禁；main基线CI及内容哈希证明与上表一致。准备提交/推送的精确SHA由交付答复和Git记录给出，不循环提交文档自证自身CI。

## 5. 最终实现与 plan 的差异

准备计划的文件/规模/命令为预估，实施按现有模块职责落地；各任务真实差异与提交见第2节的报告。T3事件词汇集中于domain；T7不加空issue.Jobs，T10接真正worker；T11 runner构造在全部声明任务的模块后，instance/router仍在其后。T3描述窗口含600秒，计划中若有“600秒即新版本”的措辞由此更正。T2初次 focused 空匹配只作为诊断记录；T10/T11实际测试名按报告核实，不复写计划正则为“通过”。T13末轮门禁原始输出缺失，T14/T15的原始全E2E与门禁覆盖后续状态。

独评记录的功能Minor均闭环：T9成功归档行 `deleted_at` 保留由T10真实断言/SQL变异证明；T10 activities/events长度先精确检查由T15重复活动变异证明；T12第二项目屏障、首项目锁释放/Execute计数由T15整批事务变异与race五遍证明；T13旧进程身份取真实child退出状态、故障附件分别尝试由T15漏stop变异及故障探针证明。T15评审另指出旧颜色警告，后续定向环境验证无警告，原日志照留。第4节的九项预检修订、这里的实施修复与第2节的最终差异/验证是不同阶段，不将“预检未关闭0”替代真实Minor记录。

手写发布点的完整性依据T15矩阵：15处`Publish`调用覆盖16个写用例/12种类型，包含各Change四槽nil/值、无事件例外；20项生产变异和三订阅者真实PG/River回滚见相邻矩阵。P6生产Subscribers仍为空集合，这是设计过渡，不能声称P7生产抓取已联通。没有新增HTTP operation、错误码、业务表、迁移或依赖。内部整分支审查结果与边界见下文。

T16任务独评修复：plane-diff、v0设计和M1/P5输入交接曾把未持久化的自动归档事件 `ActorID` 写成 SQL NULL，或把事件与活动行一起归于 SQL 读回。现已分清：T10真实 SQL/组合读回证明 `issue_activities.actor_id` 为 SQL NULL；`shared.Event.ActorID` 是同步发布的内存值 nil，由事件组合断言核对，不存在事件表或事件 SQL 读回。本修复仅更正文案，不改变产品行为或既有运行证据。

### 内部整分支审查与取舍（T16后）

唯一一次内部整分支审查以 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e..6995d523831f89936a7c70a94079651315a9f783` 为对象，覆盖19提交、165文件及完整产品/测试/生成物/文档差异。审者先核正式三文档、两矩阵与任务台账，再逐段审完整diff；围绕真实风险追踪事务错误/回滚、Runner与关闭、项目锁/活动SQL、发布payload、E2E进程和数据库生命周期。另核T15七个源码文件SHA与门禁manifest一致，没有重跑产品套件、变异或性能测试。结论：Critical 0、Important 0、Minor 0，无新增测试要求；这是内部技术审查，不替代公开分支CI或负责人外部终审。原始报告在ignored `.superpowers/sdd/P6-issue-events/final-review.md`。

审查报告的11项“考虑后不裁定为缺陷”均经主控接受，**不是发现，也不是豁免已承诺验收**：

| 项 | 取舍及边界 |
|---|---|
| 1–2 事件存储/重放、生产通知与发送鉴权/外链安全 | P6只交同步分发和事务投递接缝；P7链接首个生产订阅者、M7通知、M8 Webhook及外链规则各在其阶段验收，不提前造outbox或通知模型。 |
| 3 级联和DetachLabel单项事件 | 本期矩阵明列无事件副作用；P9按自身目标再定删除任务，不能擅加issue事件。 |
| 4 单项目极大批量性能 | 每项目一锁一事务是已批准原子性；没有规模基准就不声称任意规模吞吐，也不为猜测改分块。 |
| 5 项目失败后停止本轮 | 返回已提交数与原错，由River重试；幂等条件避免重复，继续越过故障项目属于新策略。 |
| 6 锁下未另判`ArchiveIn>12` | 候选扫描限定1–12且生产schema有约束；测试移除约束不代表生产有效状态。 |
| 7 并发/重复任务无额外唯一键 | 项目锁、未归档谓词及事务回滚已控制重复副作用，无新增schema必要。 |
| 8 fixture未逐测试DROP DATABASE | world关闭自身连接，整轮容器回收数据库；资源所有权明确。 |
| 9 准备期旧措辞 | 当时“当前仅准备/只到预检”是历史；本次进一步标记，不能据此重做T1或重索授权。 |
| 10 极端并发篡改登记配置/不稳定Args.Kind | 当前生产类型与配置受控，无实际触发链，不加推测性反射/约束。 |
| 11 尚无push、精确HEAD CI、外部终审/合并 | 是安排中的交付门，内部技术结论不代替其结果或授权。 |

## 6. 移交事项

- P1a–P5 的 spec§5/review§6 所列事件、时钟、归档输入已分别落本期；接收方可从各输入页“处理结果（M4/P6）”到本 review §2/§5 核对真实证据。输入 handoff 顶部 `status: open` 保持不变，因为其余条目仍未完成。
- [M1/P5](../handoffs/M1-P5-brand.md) 的服务端自动归档活动行 `actor_id` 为 SQL NULL，未持久化的事件 `ActorID` 为 nil；页面显示/收集箱创建者仍归 P15/M7。
- [M2 closeout](../handoffs/M2-closeout.md) §3/6/13 的 `updated_at`、worker登记/事务投递、clock fixture已接；§1命令行只投递的原文在M4收尾转M5，当前CLI不构造jobs。
- [M3 closeout](../handoffs/M3-closeout.md) §1两处期限已改，领域事件只覆盖issue写；`registrationRows` 对请求任务的`river_job`核对留P7，project/workspace删除订阅及任务留P9；§4 `archive_in` 两契约说明已生成。其他权限/页面/清理原落点不提前关。
- P7：链接首个生产订阅者与请求投递；P9：删除任务、版本保留、物理清理；P15：I16页面重登录与NULL actor展示；M6/M7按真实表条件再扩归档。M4§14事件任务整条及总体§9.4仍开放，不创建M5–M8最终closeout。

## 7. 已知限制与后续门

当前经证据确认 T1–T16 实施及逐Task独评，T16一处文档措辞问题已由`6995d523`修复并经聚焦复审通过；内部整分支审查也已完成且零新发现。首轮分支CI已在 `17b8a117` 报告clock生命周期断言失败，局部修复与本地验证见§2.1；**尚未**完成修复后聚焦复审、精确HEAD分支CI与负责人外部终审。P5 main CI仅为基线。原始证据的早期SDK fallback、T2空匹配、T13缺完整末轮rawlog、旧颜色警告已在§2/§5分开标出，不改写为 pristine。

交付时向负责人提供分支 `codex/m4-p6`、基线 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`、本 [spec](../specs/P6-issue-events.md)/[plan](../plans/P6-issue-events.md)/review 和精确 Git HEAD。最终push/CI的实际SHA由交付答复及Git记录给出，不为文档自证自身CI无限提交。本阶段先保留分支/worktree，不合并；`--no-ff`、精确main SHA CI、归档和清理仅在后续授权后进行。
