# M4/P6 领域事件、任务与自动归档实施计划

> **执行记录：** T1–T16 已按任务实施并逐项独立评审，内部整分支审查已完成且无新发现。以下 Task 清单保留当时的计划语气和命令，实际运行与差异以[评审 §2/§5](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)和任务报告为准；分支 CI、外部终审仍待后续。

**目标：** 实现 M4 §12/P6 的事件发布、事务投递、worker 完整性、两队列、测试时钟、自动归档与 I16 接口故事。

**架构：** shared 持事件值和发布端口；bootstrap 同步分发、连接 postgres.Tx 与 jobs.Inserter。issue/app 维护写事务内发布与每项目自动归档编排，project 提供自己表的目录；River 适配器只调用用例。请求沿 Locks.run，后台归档沿已批准的项目 N 事务路径。

**技术栈：** 仓库锁定 Go 1.27 / toolchain go1.27.1、pgx 5.11.0、River 0.47.0、sqlc 1.31.1、PostgreSQL 18.6、Node 24、pnpm 11.10.0、Playwright。不新增依赖。

**规格：** [P6 spec](../specs/P6-issue-events.md)；[P6 review](../reviews/P6-issue-events-review.md)。基线 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`，P5 的 `--no-ff` 合并。基线 CI 的最终结论见 review，不把它作为 P6 运行证据。

**方法：** 延续 P5 经 [02-method §3](../../claude_to_gpt/02-method.md#3-哪些是要求哪些是做法) 允许的紧凑计划：固定文件、消费/产出签名、检查和命令，以逐任务真实红绿、生产变异、恢复对照及独立评审代替完整预制原型/逐字重放。静态预检不声称已有运行证据。M4 已批准设计为架构依据，局部细化见 spec §3；真正架构问题先提交已完成工作再上报。

## Global Constraints

- 工作树 `.codex/worktress/codex/m4-p6`，分支 `codex/m4-p6`，主目录 HEAD 始终 main。第一提交落 spec/plan/review、关键词 phase=M4/P6；目前无到期的 M4/P6 例外，不延后现有例外。
- 不改 plane/refer、不在其中跑 git，不操作既有容器和 Docker，不全局安装，不增加设计未点名依赖；不 stash/clean/reset --hard。提交中文且保留 feat/fix/docs(M4/P6) 前缀；push 使用 `/usr/bin/git`。
- 每 Task 真实检查→实现→实际变异/恢复→提交→独立要求/质量评审→修复/复审。阶段实现完成先交负责人安排整分支终审，不自动合并。merge --no-ff / main CI / 清理须有相应授权与真实结果。
- 请求写继续经 Locks.run 的已有三种形式。隐藏404先于403，拒绝不进闸门/事务；锁下重读、项目归档→工作项归档→无改动→依赖→时钟→写。不得用事件发布改变既有拒绝顺序或 HTTP 形状。
- 系统自动归档按设计§3.5后台例外：不进请求闸门、不锁工作区、不伪造账户；每项目一个事务、项目N→状态目录→时钟→issues→activities→events。事务用 issue.Deps.JobTx 的原始postgres.TxManager（同LockTimeout配置，不经busyTx）；原PG错误链返回River。用户请求Tx仍保留唯一的HTTP 503转换，bootstrap共用底层构造。
- 模块不互相 import、不跨模块 JOIN。app/domain 不导入 River/pgx；platform/jobs 不导入 platform/postgres。SQL 仅 sqlc，模块不写 SET，不放宽任何架构/CLI/SQL 守卫。
- 事件按完整业务写之后、同 ctx、同已读时刻发布；不吞发布错误、不使用 Background/异步 goroutine、不以 nil publisher 静默跳过。不持久化事件、不建 outbox 表。
- 无新 HTTP operation、权限动作、错误码、业务表或迁移。Project/ProjectPatch 的说明经生成链同步；若真实实现出现新增错误码须当 Task 配 error-messages.ts/en/zh-CN 并 test-web，不能在此预设例外。
- 新手写逻辑约400行内；已有较大 module/ports 文件仅增消费所需接缝，按职责分 module_events.go/module_jobs.go、archive_ports.go 等，不顺手全模块重构；生成物沿既有例外，仅生成命令修改。
- 复用 issueWorld、callLog/fakeTx/clockAt、pgtest/RowWaits、已有权限矩阵和 runtime-role 框架、e2e/db.ts/server.ts/api.ts；不另造认证/SQL/测试基础设施。容器 suite 一次一套，确认退出后再开下一套。
- 任何测试命令须确认实际匹配；下文新测试名是计划名，实施时按真实文件同步，不把空匹配当通过。性质测试数据要能区分变异，拒绝/时间/锁必须在组合层核对。固定时钟不并发读写，交错靠条件和期限，不靠固定 sleep。

## Review Focus

1. “有动态才有事件”漏掉排序、创建标签和描述合并，删除表情又用容器ID：T3/T5/T6 的具体 payload/计数及反例。
2. 分发离开原事务、部分批量写即发布、前订阅者污染后订阅者：T1/T2/T4–6/T15 的真实 InsertTx、晚失败与深复制测试。
3. worker kind 与真实注册脱节、删 worker 连周期声明一起消失、期望集合从实际注册反推：T7 的独立描述/两种漏登记，T11 的固定预期 completed 集合。
4. 扫描时旧 archive_in、时钟在锁前、<= 变成 <、actor零UUID、首项目提交后重试重复：T9/T10/T12 的固定边界、两项目故障和真实交错。
5. I16 用 worker 库或旧 completed 记录、混用真实/偏移时间、两个 nerve 同库运行、失败泄漏进程：T8/T13/T14 的独立资源、spawn前真实时间与失败清理对照。

## 文件结构、依赖与规模

路径相对本 worktree；I=`server/internal/modules/issue`，P=`server/internal/modules/project`，B=`server/internal/bootstrap`，J=`server/internal/platform/jobs`，C=`server/internal/platform/config`。新文件名为计划，实施变更同步文档。

| Task | 文件与职责 | 前置 | 预计手写差异行 |
|---|---|---|---:|
| T1 | platform/postgres/tx.go、tx_test.go；J/inserter.go/test；B/jobs.go、jobs_inserter_test.go；I/adapter/river/inserter.go：原事务投递 | 基线 | 400–700 |
| T2 | shared/event.go；B/events.go/test；I/module_events.go、module.go：事件目录/分发器/注入 | T1 | 500–850 |
| T3 | I/app/events.go、issue_events.go；create_issue.go/update_issue.go及对应测试：创建/修改全部语义与事件身份 | T2 | 750–1150 |
| T4 | I/app/delete_issue.go/archive_issue.go、events_deletion_test.go；B/issue_events_deletion_test.go | T3 | 600–1000 |
| T5 | I/app/sub_issues.go/create_relations.go/delete_relation.go、relation_events.go/test | T4 | 750–1150 |
| T6 | I/app/comments.go/comment_writes.go/reactions.go/comment_reactions.go、collaboration_events.go/test | T5 | 750–1200 |
| T7 | J/registration.go/jobs.go及测试；identity/adapter/river/cleanup.go；C/jobs配置及config.yaml/test.yaml；B/app.go与现有构造测试最小签名迁移 | T1 | 700–1100 |
| T8 | platform/clock/offset.go/test；C/clock配置；B/clock.go/app.go/users.go/workspaces.go及测试 | T7 | 450–750 |
| T9 | I/app/archive_ports.go/ports.go；P/app/issue_ports.go、adapter/postgres/issue_ports.go/queries/issue_ports.sql；B/ports.go；I/adapter/postgres/auto_archive.go/queries/auto_archive.sql/test | T8 | 650–1000 |
| T10 | I/app/auto_archive.go/rows.go/writes.go；postgres/issues.go；domain/activity.go；I/adapter/river/auto_archive.go；I/module_jobs.go；B/gate.go/app.go的JobTx接线及测试 | T2/T7/T9 | 650–1150 |
| T11 | B/app.go/jobs_test.go/runtime_role_test.go；issue/identity Jobs/EnqueuedKinds；archtest组合 | T10 | 550–900 |
| T12 | B/issue_auto_archive_interleaving_test.go、既有锁帮手扩展 | T11 | 650–1000 |
| T13 | e2e/fixtures/clock.ts、test.ts、stories/smoke/clock-lifecycle.spec.ts：独立fixture验证 | T8/T11 | 400–700 |
| T14 | e2e/stories/issue/i16-auto-archive.spec.ts、assert/issue.ts必要小扩展 | T12/T13 | 350–650 |
| T15 | B/issue_events_test.go等跨层缺口、最终变异与全门禁记录 | T1–14 | 450–800 |
| T16 | spec/plan/review、M4/v0/M2设计、README、plane-diff、相关handoff | T15 | 文档为主 |

这是起步估计而非原型实测。参考 main 实读 create_issue.go 212行、update_issue.go 311、delete_issue.go111、create_relations.go202；jobs_test.go235、runtime_role_test.go214、e2e/test.ts97、server.ts181。每 Task 实施前核真实职责/差异，留约10%修复余量；超过约1500手写行时依D2重切并同步文件，不减反例/回归。原设计11粗项映射：1→T1；2→T2；3→T3–5；4→T3/T6；5→T7；6→T11；7→T8；8→T9/T10；9→T12；10→T13/T14；11→T16；完成线统一由T15复核。

## 跨 Task 接口

以下是起步时固定的跨任务接口；T1–T15 已实现，真实签名以源码及评审 §5 为准。uuid 是本仓库标准库 `"uuid"`。

```go
// platform/postgres；平台之间不互导。
func Tx(context.Context) (pgx.Tx, bool)
// platform/jobs；不配 Workers/Queues、不 Start、不拥有 pool 生命周期。
func NewInserter(*pgxpool.Pool) (*Inserter, error)
func (*Inserter) InsertTx(context.Context, pgx.Tx, river.JobArgs, *river.InsertOpts) error
// issue/adapter/river（消费方）；bootstrap.jobsInserter 以 postgres.Tx 接平台实现。
type Inserter interface { InsertTx(context.Context, river.JobArgs, *river.InsertOpts) error }

// shared；仅标准库。
type EntityRef struct { Kind string; ID uuid.UUID }
type Change struct { Field string; Old, New *string; OldID, NewID *uuid.UUID }
type Event struct {
    ID uuid.UUID
    Type string
    OccurredAt time.Time
    WorkspaceID, ProjectID uuid.UUID
    ActorID *uuid.UUID
    Entity EntityRef
    Changes []Change
}
type EventPublisher interface { Publish(context.Context, Event) error }
type EventSubscriber struct { Name string; Types []string; Handle func(context.Context, Event) error }
// bootstrap；稳定登记顺序，登记值复制、逐订阅者事件深复制。
func newDispatcher(types []string, subscribers []shared.EventSubscriber) (*dispatcher, error)
func (*dispatcher) Publish(context.Context, shared.Event) error
// issue 模块目录；12类型见spec，未来P7/P9才扩展。
func EventTypes() []string
func Subscribers(inserter riveradapter.Inserter) []shared.EventSubscriber
```

P6 `Subscribers` 明确为空的阶段过渡（M4§3.19），T2 经 NewInserter→jobsInserter→Subscribers 建分发器；不提前造 LinkCrawls、RowDeletions 空业务适配，也不把 Inserter 放进 app/domain。EventPublisher 注入每个实际写用例与 `issue.Deps`；不是在仓储 InsertActivities 内暗发事件。Publish未知类型报程序错误，已声明但无人订阅成功；失败错误保留链。所有业务字段/payload白名单在 spec，不反射整个模型。

```go
// platform/jobs；worker.kind/add 私有、由同一T绑定，不允许两份可填元数据。
type Job struct { Worker *WorkerRegistration; Periodic *PeriodicSpec }
func Register[T river.JobArgs](river.Worker[T]) *WorkerRegistration
type PeriodicSpec struct {
    Args river.JobArgs
    Interval time.Duration
    ID string
    RunOnStart bool
    InsertOpts *river.InsertOpts
}
// Config 保留ShutdownTimeout/Logger，新增DefaultWorkers/PeriodicWorkers int。
func New(*pgxpool.Pool, Config, []Job, []string /* enqueuedKinds */) (*Runner, error)
// issue / identity；当前请求无投递，EnqueuedKinds为空；周期种类独立核对。
func (*Module) Jobs() []jobs.Job
func EnqueuedKinds() []string

// platform/clock；bootstrap newClock(cfg)满足各模块Clock，不引入shared依赖。
type Offset struct { Base interface { Now() time.Time }; Offset time.Duration }
func (Offset) Now() time.Time

// issue/app；project侧等形小值/接口由bootstrap转换。
type ArchivePolicies interface { ArchiveProjects(context.Context) ([]uuid.UUID, error) }
// 现有 LockedProject 新增 ArchiveIn int，ProjectGate.Lock/LockMany同时回答。
type AutoArchiveRows struct { ProjectID uuid.UUID; StateIDs []uuid.UUID; Cutoff, At time.Time }
type AutoArchiveStore interface {
    AutoArchive(context.Context, AutoArchiveRows) ([]uuid.UUID, error)
    InsertActivities(context.Context, []ActivityRow) error
}
// issue.Deps 新增 JobTx shared.TxManager；AutoArchiveDeps.Tx取JobTx，不能取请求busyTx。
// ActivityRow.ActorID 改为 *uuid.UUID；其余手动写者值保持强制非空。
type AutoArchiveDeps struct {
    Policies ArchivePolicies
    Projects ProjectGate
    States StateCatalog
    Issues AutoArchiveStore
    Tx shared.TxManager
    Clock Clock
    Events shared.EventPublisher
}
func NewAutoArchive(AutoArchiveDeps) *AutoArchive
func (*AutoArchive) Execute(context.Context) (int, error)
// issue/adapter/river：AutoArchiveArgs.Kind()=="issue.auto_archive"，worker只调Execute。
```

每个Event由app用uuid.NewV7生成独立非零ID，不能复用实体ID；批量互异、分发各订阅者同一值，OccurredAt仍取业务写时刻。关系/表情容器字段只填ID槽（文本nil），完整四槽表见spec。

Jobs 构造先拒绝nil/typed-nil worker等无效描述，再实际登记 worker、保留成功 kind 集合，独立检查周期 Args.Kind 和 EnqueuedKinds 后再造工作客户端；不可从 River 私有字段反射。periodic 与 worker 可独立缺失，漏接整个 Job 由 bootstrap 固定预期集合测试捕获。默认10/2，两个配置正整数；自动归档默认24h、test2s、至少1s；clock.offset默认0，非test非0拒绝。身份清理和归档均 periodic，RunOnStart=true。River调度/退避、限速用真实时间，业务 Clock 统一偏移UTC微秒。

```ts
// e2e/fixtures/clock.ts：普通函数可被全局准备调用，不依赖testInfo。
type ClockRun = { nerve: Nerve; api: Api; startedAt: Date };
type ClockWorld = {
  db: Database;
  current: ClockRun;
  advance(offset: string): Promise<ClockRun>;
  close(): Promise<void>;
};
async function createClockWorld(options: {
  databaseName: string;
  logFile: (generation: number) => string;
  now?: () => Date; // 默认new Date，仅供生命周期测试观察采样顺序
}): Promise<ClockWorld>;
// test.ts 导出由Playwright原始base扩展的clockTest；不继承普通test的自动db快照。
```

fixture独立建库/连接/进程，advance先await旧进程stop再启动；startedAt在spawn前取真实Date。失败快照/日志/最终关闭由clockTest统一拥有，底层启动失败也清理已获得资源。数据准备仅API；SQL只断言。生命周期验证沿已有Playwright，不引入node:test的TS加载链或新runner；T14真正消费fixture。 CI修复追加：专用顺序故事注入now并通过既有logFile边界核对每代采样早于启动、返回采样值；正常故事与I16仍走默认真实墙钟。文件birthtime不作顺序oracle，ready后采样的真实变异必须断言红并恢复。

## 共用实施收口 G

每 Task 的最后一步执行 G：运行定位测试并确认匹配→记录真实红因/绿结果→运行指定生产变异（不是改测试断言）→恢复再绿→按影响范围执行 lint/gen/回归→只 add 本 Task 文件→中文提交→以精确BASE..HEAD交独立评审并修复复审。已有正确行为补测可以baseline绿，敏感性用真实变异业务红证明，不虚造初红。生成物提交后 `make gen-check`。所有记录含命令、退出码、HEAD、范围、变异差异及恢复；S4历史、静态预检、当前运行证据分开。

Go命令在 `server/`；node/Makefile命令在 worktree根。`go test -count=1` 用现有pgtest。新锁交错均 `-race -count=5`，不得有40P01。全量门禁结果可沿未受影响的内容复用；新修改/故障必须验证，不为文档SHA无限重跑。

## Task 1：原事务内只投递客户端

- [ ] 先写 TestInserterCommitRollback / TestInserterRequiresTransaction：无worker启动，原Tx写业务probe+投递；独立观察连接提交前不可见、提交两者留、sentinel回滚两者无；无tx错误，错误链可识别。
- [ ] 运行 `go test -count=1 ./internal/platform/postgres ./internal/platform/jobs ./internal/bootstrap -run 'Test(Tx|Inserter)'`，确认业务缺失红；实现Tx getter、NewInserter/InsertTx及B小适配，恢复绿。
- [ ] 变异：wrapper用pool另开事务、ctx丢事务、吞InsertTx错误；组合层须失败，恢复绿。S4既有事实只作设计依据，单列本次实测。
- [ ] G；跑archtest，jobs与postgres不互导、app不见River，CLI不达投递。

## Task 2：共享事件与确定性分发器

- [ ] 写 TestDispatcher：3订阅者交叉类型/逆字母登记、第二出错第三不执行、无订阅成功；未知订阅/发布、重名/空名/nil handle/重复订阅类型拒绝；各订阅者收到同一Event.ID；外部登记切片修改无效；首订阅者修改指针/切片不污染后者与原事件。
- [ ] 实现shared值/端口、12类型目录和dispatcher；在app组合经只投递适配建空生产订阅列表，注入issue与现有写构造点的明确fake，不加nil跳过。
- [ ] `go test -count=1 ./internal/shared ./internal/bootstrap ./internal/modules/issue/app ./internal/archtest -run 'Test(Dispatcher|.*Event|CommandsCompose)'`；编译缺接线不当作语义红。
- [ ] 变异：按名字排序/丢复制/吞subscriber错误/取消目录检查；相应独立断言失败，G。

## Task 3：创建与修改事件（含描述、排序）

- [ ] 写 TestIssueCreatedEvent / TestIssueUpdatedEvent：初值含2标签、默认负责人、parent/dates/description；修改逐字段、集合增减、sort_order=0及相邻可表示float；非零Event.ID≠Entity.ID、批量/连续写事件ID互异；no-op与skip=true只补id零事件；描述599秒与恰好600秒合并、600秒+1微秒开新版本；每个真实修改仍各发布一个事件。
- [ ] 从锁下真实值/已算Changes构造payload，使用现有s.at/by，业务SQL/动态/版本全成功后Publish；不以seen响应或动态是否插入判断，不新增动态行。
- [ ] `go test -count=1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(Issue(Created|Updated)Event|.*Description|.*SortOrder)'`；原HTTP隐藏/拒绝矩阵保持。
- [ ] 变异：事件只取activity、忽略sort_order/创建label、版本合并时不发、额外读Clock、先于版本写发布、Event.ID置零/复用Entity.ID；app及至少一个真实组合失败，G。

## Task 4：子树删除、归档与恢复事件

- [ ] 写 TestIssueDeletionEvents / TestIssueArchivalEvents：三层树含隐藏/已归档子项，每个实际删除一个；手动archive/unarchive marker与真实actor，幂等零事件；直接删除与项目/workspace同步cascade不发布的区别。
- [ ] 所有业务批写后按ID发布，不随连带发评论/表情/关联事件；末尾某事件失败整树/所有动态/前投递回滚。
- [ ] `go test -count=1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(Issue(Deletion|Archival)Events|.*Deletion|.*Archival)'`。
- [ ] 变异：仅根事件、按请求可见性漏子树、幂等也发、Publish移事务外；真实DB回滚反例失败，G。

## Task 5：父子与关联事件

- [ ] 写 TestHierarchyEvents / TestRelationEvents：两个移动子项+已在父下对照；父无额外事件；relation两端跨项目两种ID序、blocking反向存储、删除从任一授权端进入。
- [ ] 逐槽核对spec的nil文本/ID载荷（错误填UUID文本或错侧也须有生产变异），固定Entity真ID和存储ProjectID，两端Changes以canonical存储方向，关系每行一个事件，不按两条动态变两件；先完成所有关系/两端审计/动态再发布。
- [ ] `go test -count=1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(HierarchyEvents|RelationEvents|.*Relation.*Locks|.*SubItems)'`。
- [ ] 变异：按授权端改payload、双向发两次、为未移动子项发、锁下提前发；数量/逐值/回滚断言失败，G。

## Task 6：评论与两类表情事件

- [ ] 写 TestCommentEvents / TestReactionEvents：评论HTML旧/新清洗值、同HTML零事件；管理员删除他人评论actor是管理员；两类表情rowID≠containerID≠actorID，增加/移除都取真实rowID；评论表情仍不推进主项审计。
- [ ] 保留锁下读取的旧实体再写，不重新生成删除实体ID；评论删除保留旧comment_html，无子表情连带事件，评论/关联/表情变化不再另发issue.updated。
- [ ] `go test -count=1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(CommentEvents|ReactionEvents|.*Comment|.*Reaction)'`。
- [ ] 变异：actor取作者、Entity取容器、旧HTML被新值覆盖、容器ID字段错误填字符串、给评论表情推进issue.updated_at、吞Publish错误；真实行为断言失败，G。

## Task 7：任务登记、两队列与期限

- [ ] 写 TestJobRegistration / TestJobQueues：周期保留但worker移除、EnqueuedKinds保留但worker移除分别构造失败；错误register/重复worker/周期ID/nil或typed-nil worker/无效args与interval/worker数；配置载入/env/default的正负例。
- [ ] 实现强类型Register+独立PeriodicSpec，实际AddWorkerSafely成功才记kind。同一Task改身份清理descriptor/periodic队列、bootstrap.New四参/worker配置与全部旧测试构造点，保持各Task可编译启动，不拖到T11；默认/周期队列10/2；各放超过上限的阻塞probe，一队释放仅推进本队且另一队可并行。
- [ ] 将 TestStartWithoutADatabase / TestStopEndsTheAttemptsToStart 的固定余量改为条件屏障+与firstRetry分得开的期限，不能把sleep放大充修复。
- [ ] `go test -count=1 ./internal/platform/jobs ./internal/platform/config ./internal/modules/identity/adapter/river ./internal/bootstrap`；一次一套，注意现有runner重试/关停回归。
- [ ] 两种生产完整性检查分别移除，使各负例失败；省略整个job由T11抓住。容量与取消等待变异/恢复，G。

## Task 8：统一业务偏移时钟

- [ ] 写 TestOffset / TestClockConfig / TestBusinessClock：正负零、跨日期、微秒截断；dev/prod非0失败、test±允许、env覆盖；服务与users/workspaces业务用例同一来源，limiter/River保持真实时间。
- [ ] 增clock.Offset与clock.offset配置/有效配置日志；B/newClock统一注入全部业务Clock（包括CLI），CLI仅加时钟不加issue.New/jobs/dispatcher。
- [ ] `go test -count=1 ./internal/platform/clock ./internal/platform/config ./internal/bootstrap ./internal/archtest -run 'Test(Offset|ClockConfig|BusinessClock|CommandsCompose|.*Config)'`。
- [ ] 变异：只偏移worker、放过prod非0、某模块仍System、精度不截断；各有可失败检查，G。

## Task 9：归档候选、锁下配置与静态SQL

- [ ] 写 TestArchivePolicies / TestAutoArchiveRows：候选仅有效1–12项目；Lock/LockMany读回archive_in；旧/恰好/新cutoff行，完成+取消与旧的其他状态、删除/已归档/外项目/空StateIDs各独立反例。
- [ ] P只扫自己projects并返回ID；I接小端口，锁下经已有StateCatalog得到终态ID，不JOIN states/projects。I一条UPDATE设置archived_at/updated_at=At、updated_by_id=NULL，RETURNING实际ID；空状态集永不匹配。
- [ ] `make gen-go`；`go test -count=1 ./internal/modules/project/adapter/postgres ./internal/modules/issue/adapter/postgres ./internal/archtest -run 'Test(ArchivePolicies|AutoArchiveRows|.*IssueProject|SQLCSchemaScope|ModulesRunSQL)'`。
- [ ] 变异：<=改<、忽略state/archived/deleted/project、空数组成无筛选、锁回答不带archive_in；store真实SQL失败，G。

## Task 10：自动归档用例、空执行者与worker

- [ ] 写 TestAutoArchive：逐项目独立tx/锁/目录/时钟/批写/动态/事件顺序；scan之后锁下关闭/改阈值/归档/删除跳过；全部端口错误传播；第二项目失败，首项目已提交保留，重试不重复；ctx取消。
- [ ] ActivityRow.ActorID显式指针，stamp手动动作保持非空，系统nil；核adapter传入真实SQL NULL且普通动作身份未丢。bootstrap给issue.Deps.JobTx注入raw LockTimeout TxManager，AutoArchive消费它；真实锁超时测试用既有 `postgres.LockNotAvailable(err)` 确认原错误链，并核非shared.ServerBusy；不手写55P03判断。请求busyTx旧映射回归保留。新增AutoArchiveActivity使用archive标志；不新增关注、不改父子。
- [ ] 按锁下ArchiveIn算30*24h阈值，一次Now；返回ID排序后先全部活动再全部事件。worker仅调用Execute返回错误，periodic24h/test2s。配置与Project/ProjectPatch两处说明同步，`make gen`。
- [ ] `go test -count=1 ./internal/modules/issue/... ./internal/bootstrap -run 'Test(AutoArchive|.*Activity|.*Archival|ABusyTransactionIsServerBusy|AProjectTooBusyForAWriteAnswersServerBusy)'`；固定时钟30天前1秒/恰好/后1秒与微秒边界，真实发布失败回滚UPDATE+动态+river_job。
- [ ] 变异：30天改自然月/31天、提前读clock、使用扫描阈值、actor取创建者/零UUID、吞任一端口错误、首项目回滚或重复动态；定位失败后恢复，G。

## Task 11：最终任务组合、运行角色与命令守卫

- [ ] 写 TestAllPeriodicJobsRun：固定期望identity.cleanup_expired_sessions与issue.auto_archive，不从Jobs返回值反推；实际serve启动、每种river_job completed，队列为periodic；多个间隔配置实测。
- [ ] 模块Jobs/EnqueuedKinds经bootstrap一次登记，工作客户端在所有声明任务的模块完成构造后构造一次，随后仍构造 instance/router；NewInserter在模块前、不得Start或交工作client给模块；失败构造关闭pool等原资源。
- [ ] 扩现有runtime角色框架，以实际授予角色执行自动归档并断言业务行/空actor，不只看到cleanup completed。无新表不凭空改grants。CLI现有调用图继续拒绝jobs/River/issue.New/InsertTx。
- [ ] `go test -count=1 ./internal/bootstrap ./internal/archtest -run 'Test(AllPeriodicJobsRun|TheRuntimeRoleServesWithTheGrantsFile|TheGrantsFileCoversEveryRelationAndFunction|CommandsComposeNoServerAndNoJobs)'`。
- [ ] 变异：bootstrap漏接自动归档Job、所有任务进default、CLI引入inserter、临时撤去活动写权限（仅测试角色fixture）；实际检查失败，恢复G。

## Task 12：交错18及配置/删除等待

- [ ] 写 TestAutoArchiveAndStateChangeSerialize（两方向）：状态先改非终态则不归档；归档先提交则用户修改409/issue.archived。用真实项目N和pgtest.RowWaits，不sleep。
- [ ] 写 TestAutoArchiveUsesLockedPolicy / TestAutoArchiveReadsClockAfterLock：候选后改archive_in/禁用/归档项目/删除项目与工作区，两种先后；等待期间安全推进Fixed，检查每行最终戳/actor/事件及无残留写。
- [ ] 核完整资源图：任务只持单项目N、不持workspace、不等gate；请求先取齐gate再项目升序；没有跨模块SQL和新的锁反向边。约定五用项目N串行论证，不额外逐issue预锁。
- [ ] `go test -race -count=5 ./internal/bootstrap -run 'TestAutoArchive(AndStateChangeSerialize|UsesLockedPolicy|ReadsClockAfterLock)'`，必须确有三组测试、所有方向通过且无40P01。
- [ ] 去项目锁/锁前Now/旧策略/提前不再重读各变异须在对应组合测试业务失败而非超时；恢复再race，G。

## Task 13：独立时间流逝fixture

- [ ] 实现createClockWorld普通函数与test.ts独立clockTest，直接复用db/server底层；不继承worker db/nerveWith/auto databaseSnapshot。数据库名带worker/test/retry唯一值。
- [ ] 验证stop旧进程完成才start新进程、startedAt取spawn前真实时间、每代独立日志、偏移值走NERVE_CLOCK__OFFSET；startup/advance/use失败均关闭当前进程和pool，失败dump自己的库。
- [ ] 新增smoke/clock-lifecycle.spec.ts，直接使用Playwright原始base和createClockWorld验证实际切换与非法offset导致advance失败后的清理；同时断言独立库标记未出现在worker库。失败仍调用close且不能遗留当前进程/数据库连接；不另加runner或依赖。真实fixture在紧邻T14消费。
- [ ] 跑 `pnpm --filter @nerve/e2e check:types`，`make build` 后 `pnpm --filter @nerve/e2e exec playwright test stories/smoke/clock-lifecycle.spec.ts --project=chromium`；`make lint-web`/`make knip`，漏stop/错误库归属变异须有失败对照，G。

## Task 14：I16自动归档接口故事

- [ ] 经API准备PAT、工作区/项目archive_in=1、旧completed和cancelled目标、旧非终态、archive_in=2旧终态对照；切偏移744h进程，再API建/改recent终态。每个性质用独立对照行。
- [ ] 用PAT；最后recent写成功后，自己的DB读 `SELECT clock_timestamp()::text AS watermark`（保留微秒原字符串），轮询 `attempted_at > $1::timestamptz` 且completed的auto_archive，并保留finalized_at>spawn前真实startedAt；此后再核业务结果，默认列表消失/归档列表出现、GET/activities系统actor=null、DB每行同戳且仅一条归档活动；不是历史completed或偏移业务时间。
- [ ] `make build` 后`pnpm --filter @nerve/e2e exec playwright test stories/issue/i16-auto-archive.spec.ts --project=chromium`定位I16，再 `PLAYWRIGHT_BROWSERS_PATH=<项目内已归档缓存绝对路径> make e2e`。保留Chromium全故事/Firefox A4，不全局安装浏览器。
- [ ] 真实生产变异：自动归档actor改为项目作者、去状态过滤、阈值错误，以及fixture误认旧completed/只按finalized而任务早于recent开始的诊断对照；固定暂停任务在开始后、对照写入前的屏障或等价可控时序证明这种旧运行不能满足条件。必须是成功build后的断言失败；恢复完整I16。未来页面在偏移nerve重登录留P15，G。

## Task 15：全发布点、回滚与最终门禁

- [ ] 建已实现写用例清单对12事件与无事件例外逐行核验（Event.ID独立性、四槽nil/值也要逐项）；所有新增或改动拒绝仍与不存在逐值相同，publisher零调用；每Task的代表DB回滚不能被仅fake日志替代。
- [ ] 真实三订阅者：第一InsertTx、第二失败、第三不得调用；批量最后事件失败之前所有投递/业务/活动一起回滚。成功对照确认任务提交留下，原ctx内能看到最终业务状态。
- [ ] 从T1–14抽不少于15个生产变异（安全/事务/锁/完整性至少各一，剩余事件/时钟/阈值），登记实际失败层与恢复；未检出的先根因修复，不把INFRA当发现。
- [ ] 顺序跑 `make test`、`make lint-go`、`make lint-web`、`make knip`、`make test-web`、`make e2e`；带生成物提交后 `make gen-check`。沿已验证未变化内容复用，不重复缓存数字冒充新结果。
- [x] 全Task独评后完成一次内部整分支审查：19提交/165文件，Critical/Important/Minor均0，无新增产品修复；负责人外部终审仍待安排。G。

## Task 16：正式记录、输入交接与阶段交付

- [x] spec§4每条证据映射至真实命令/Task/HEAD，plan按实际实现同步；review七节记录局部决定、预检、实现/修复、最终差异与限制。
- [x] 同步v0设计§6.2/6.3/6.7、M2设计§3.15/3.17、M4§3.23/P6与§15、plane-diff第四节、README部署配置；API说明已在T10落地。
- [x] M1-P5服务端空actor、M2§3/6/13、M3§1部分/§4写处理结果；P1a–P5相应接收链接指向review。handoff总体status仍open。M2§1原文转M5、registrationRows=P7、删除/保留/清理=P9、页面=P15都不提前关闭。
- [x] M4§14事件任务整条仍含P7/P9未实现部分，不整体勾选；总体§9.4保持M4进行中，不创建M5–M8最终收尾文件。
- [x] 核中文文档链接/命令/签名/状态与git diff并提交，告知负责人分支、基线、spec/plan/review；本阶段不合并。
- [x] T16初评Important措辞问题由`6995d523`修复并经聚焦复审通过；内部整分支审查完成，零新发现。
- [ ] push后按精确HEAD查分支CI；外部终审另待安排。合并/mainCI/清理须再获授权。

## 实施状态与历史边界

准备阶段三个视角的独立静态预检与修订复审均通过，当时“全部未实施”属历史状态。T1–T16 后续已逐 Task 实施与独评，真实命令、变异和差异见 review §2/§5；T16 初评的一处措辞问题已修复并经聚焦复审通过。T15 最终门禁在源码 HEAD `4a489125f480402692772cbae483af26d9600792` 上全部退出0。T2早期聚焦空匹配不计新测试证明；真实描述版本窗口含恰好600秒，600秒+1微秒进入新版本。内部整分支审查已完成且无新发现；push后精确HEAD分支CI、外部终审及授权合并/mainCI/清理仍待办；P5基线main CI不能替代P6证据。
