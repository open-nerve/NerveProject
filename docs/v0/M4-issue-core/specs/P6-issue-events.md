# M4/P6 `issue-events`：领域事件、事务投递与自动归档规格

日期：2026-10-26。基线：`main` `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`（P5 的 `--no-ff` 合并）。状态：**T1–T16 已实施并逐任务独评通过；内部整分支审查完成且无新发现；分支 CI 与外部终审待办**。配套：[plan](../plans/P6-issue-events.md)、[评审记录](../reviews/P6-issue-events-review.md)。

权威输入：[M4 设计](../M4-design.md) §3.5、3.8、3.19、3.20、6.5–6.6、12/P6、13.1、14；[推进方式](../../claude_to_gpt/02-method.md)、[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md)；前段 spec §5 / review §6 与本规格 §7 的交接。设计 D3、D4、D7 和架构问题 11.4、11.7、11.11 已批准，不重新裁定。

## 1. 目标

完整交付 P6：共享事件值与事务内同步分发器、P1–P5 已有写用例的发布点、只投递 River 客户端、两个队列与登记完整性检查、测试环境业务时钟偏移、自动归档及 I16 接口故事。事件失败与业务写同事务回滚；自动归档按锁下策略、状态及更新时间判断，系统执行者是真正 NULL。

本阶段不新增 HTTP 路由、公开错误码、权限动作、业务表、迁移或第三方依赖；沿用已锁定的 River 版本。`Project.archive_in` 与 `ProjectPatch.archive_in` 的说明随自动归档实现更新并重新生成。`plane/`、`refer/` 只读；架构守卫不放宽；既有用户请求全部沿 `Locks.run` 及其 across/either 形式。

P6 没有生产事件订阅者是设计明确的阶段过渡，P7 才有链接抓取。测试组合的真实订阅投递证明事务语义，不借此提前实现链接或项目删除任务。M4 未完成、M5–M8 交接尚不整体关闭。

## 2. 交付物

### 2.1 文件与职责

下表列主要手写文件；对应测试邻接放置。生成物仅由既有 `make gen` 产生，实际 SHA-256、行数及差异随实施记录，当前不预填。

| 文件 / 目录 | 职责 | Task |
|---|---|---|
| `server/internal/platform/postgres/tx.go`；`platform/jobs/inserter.go`；`bootstrap/jobs.go`；`modules/issue/adapter/river/inserter.go` | 提取事务、只投递客户端、跨平台组合的小接口 | T1 |
| `server/internal/shared/event.go`；`bootstrap/events.go`；`modules/issue/domain/events.go`；`modules/issue/module_events.go` | 共享值、事件名单、同步分发与模块出口 | T2 |
| `server/internal/modules/issue/app/events*.go` 与现有写用例 / `module.go` | 集中构造语义事件，各用例在事务末尾发布 | T3–T6 |
| `server/internal/platform/jobs/jobs.go`、`registration.go`、`config`；`modules/identity/adapter/river/cleanup.go` | 真实 worker 登记、独立周期声明、两队列、构造校验 | T7 |
| `server/internal/platform/clock/offset.go`；`platform/config`；`bootstrap/clock.go`、`app.go`、`users.go`、`workspaces.go` | 统一业务时钟，test-only 配置 | T8 |
| `server/internal/modules/project/app/issue_ports.go`、`adapter/postgres/queries/issue_ports.sql`；`issue/app/archive_ports.go`；`issue/adapter/postgres/queries/auto_archive.sql` | 项目候选、锁下策略、模块自身 SQL | T9 |
| `server/internal/modules/issue/app/auto_archive.go`、`rows.go`；`adapter/postgres/issues.go`；`adapter/river/auto_archive.go` | 每项目归档编排、nullable actor、worker | T10 |
| `server/internal/bootstrap/app.go`、`jobs_test.go`、`runtime_role_test.go`；`archtest/composition_test.go` | 最终任务组合、真实运行角色及命令隔离 | T11 |
| `server/internal/bootstrap/issue_auto_archive*_test.go` | 交错 18、锁下策略与时钟、真实回滚 | T12 |
| `e2e/fixtures/clock.ts`、`test.ts`；`stories/issue/i16-auto-archive.spec.ts` | 独立库 / 进程时钟故事；API 前置与三层断言 | T13–T14 |
| 现有权限 / 契约 / 架构与发布点回归测试；忽略目录中的证据 | 跨层矩阵、生产变异、完整门禁 | T15 |
| `api/modules/project.yaml` 及生成物；`README.md`；v0/M2/M4 设计、差异、交接与本阶段文档 | 说明、事实与后续责任同步 | T10/T16 |

### 2.2 T1：同事务只投递

稳定接口：`postgres.Tx(context.Context) (pgx.Tx, bool)`；`jobs.NewInserter(*pgxpool.Pool) (*Inserter, error)`；`(*Inserter).InsertTx(context.Context, pgx.Tx, river.JobArgs, *river.InsertOpts) error`。客户端不配 Workers / Queues、不调用 Start，不拥有池与 worker 生命周期。

`issue/adapter/river.Inserter` 只消费 `InsertTx(context.Context, river.JobArgs, *river.InsertOpts) error`；bootstrap 实现从 ctx 取 Tx，再调平台客户端。无事务立即返回程序错误，不回退 pool / 独立事务。`platform/jobs` 不导入 postgres/shared，app/domain 不导入 River/pgx。以真实 PostgreSQL 证明提交业务行与任务均留、回滚均无、提交前另一连接不可见、错误链保留。

### 2.3 T2：共享事件、分发与生产接线

```go
type Event struct {
    ID uuid.UUID
    Type string
    OccurredAt time.Time
    WorkspaceID uuid.UUID
    ProjectID uuid.UUID
    ActorID *uuid.UUID
    Entity EntityRef
    Changes []Change
}
type EntityRef struct { Kind string; ID uuid.UUID }
type Change struct {
    Field string
    Old, New *string
    OldID, NewID *uuid.UUID
}
type EventPublisher interface { Publish(context.Context, Event) error }
type EventSubscriber struct {
    Name string
    Types []string
    Handle func(context.Context, Event) error
}
```

上述在 `shared`，只依赖标准库。`bootstrap.newDispatcher([]string, []shared.EventSubscriber) (*dispatcher, error)` 校验类型声明、订阅类型、唯一非空名字、有效 Handle 和重复订阅；`Publish(ctx, shared.Event) error` 按登记顺序同步调用，首个错误原样返回且不调后续，未知发布类型拒绝，合法无订阅事件成功。登记数据与逐订阅者 Event 深拷贝，不能让切片或指针修改污染后续订阅者；不持久化事件、不建 outbox 表。

`issue.EventTypes() []string` 返回本期实际 12 类型；`issue.Subscribers(Inserter) []shared.EventSubscriber` 在 P6 返回空（设计明示过渡），参数是只投递适配器端口，不构造伪 LinkCrawls。bootstrap 在模块 New 前建完整 dispatcher，经 `issue.Deps.Events shared.EventPublisher` 注入全部写用例；不以 nil Publisher 静默跳过。P9 才让 project/workspace New 收分发器。测试订阅者仅在测试组合存在。

### 2.4 T3：创建、修改与描述事件

创建 / 快速添加复用 CreateIssue，一个 `issue.created`，Entity.Kind=`issue`、ID 为工作项真实 ID。Changes 依次为 name、description、state、priority、start_date、target_date、parent、sort_order、sequence_id、completed_at，再按 ID 升序每个 assignee、label；字段名分别 `assignees`、`labels`。采用清洗后、默认值和实际指派后的值，不复制只有创建/负责人的动态来冒充全字段。

创建 Old/OldID 为 nil；nullable 新值为空时 New/NewID 为 nil。文本为实际字符串；日期 YYYY-MM-DD；时间 UTC RFC3339Nano；float64 无损 `FormatFloat(v,'g',-1,64)`；整数十进制。状态、标签、负责人、父项分别带对应 ID 和锁下名称/编号，复用既有检查中取得的值。Entity/顶层已有 ID、actor、时刻不重复塞入 Changes；响应专用 SubIssuesCount 不进入事件。

修改一个 `issue.updated`；沿 `domain.UpdateActivity` 的字段名、Old/New/ID 形状与顺序，再加实际 sort_order 变化。completed_at 由 state 语义推导，不独立发 Change；排序不写动态仍必须发事件。事件记录本次真实变化，不因描述动态 / 版本 600 秒合并而合并。真实旧父项用已经取得的 `named.parent`，不从 seen 响应反推。

无改动不发布；`skip_activity=true` 且内容标识相同的保存即使写 HTML/关系也不发布。标识变化的 skip=true 是普通修改；显式 false 与其他字段的互斥沿 P4/P5 不变。所有事件的 OccurredAt 使用当前业务写已经读取的一次时刻。每个实际发布事件用 `uuid.NewV7()` 生成独立、非零 Event.ID，不复用 Entity.ID；一次批量内互异，同一事件对所有订阅者保持同一ID。Event.ID是事件身份，UUID内部时间不代替业务OccurredAt。

### 2.5 T4：删除、归档和恢复事件

直接删除工作项：全部子树与附属表软删除成功后，按子树 ID 升序逐项 `issue.deleted`，Changes 空；不额外发连带评论/表情/关联事件。管理员、创建者、隐藏 404、已归档允许删除的现有语义不变。

手动归档 / 恢复各一个 `issue.archived` / `issue.unarchived`，Entity 为 issue。`archived_at` Change 复用动态标志：手动 nil→`manual_archive`，自动 nil→`archive`，恢复 `archive`→`restore`；发生时刻取 OccurredAt。目标状态相同的幂等归档/恢复不发布。任一发布失败回滚原写及动态。

### 2.6 T5：父子与关联事件

修改父项沿 T3；批量加子项仅为实际 moved 子项各发 `issue.updated`（parent Change），不发父节点或已在父下的子项事件。全部 SetParents / 动态完成之后再循环发布；不是改一个即发布一个。

每条关联一个 `issue.relation_added` / `issue.relation_removed`，Entity.Kind=`issue_relation`、ID 为关系行。采用 `domain.NewRelation` 的存储方向：ProjectID=r.ProjectID；Changes 固定 `relation_type`、`issue`、`related_issue` 三项，增用 New/NewID、删用 Old/OldID。两端 ID 均明确，blocking 不改变 schema，不按授权成功的那一端改方向；双向动态不是两个事件。批量全部业务写成功后，沿既有确定顺序发布。

### 2.7 T6：评论与两类表情事件

评论增 / 改 / 删各一个 `issue.comment_created/updated/deleted`，Entity.Kind=`issue_comment`。Change.Field=`comment_html`；创建 nil→清洗后 HTML、修改锁下旧 HTML→新 HTML、删除旧 HTML→nil。修改在覆盖 current.HTML 前保留旧值；相同 HTML 不发布；删除者是实际 actor，可为他人评论的管理员。所属 issue 的审计时间虽改变，不额外发 issue.updated；评论表情连带删除不另发事件。

工作项 / 评论表情共同使用 `issue.reaction_added/removed`，Entity.Kind 分别 `issue_reaction` / `comment_reaction`，ID 为真实行 ID。Changes 固定 `reaction` 及容器 `issue` / `comment`，增取 New/NewID、删取 Old/OldID；删除保留当前查询返回行，不能以资源 ID 或新 UUID 替代。评论表情继续不改 issue.updated_at；重复新增、找不到、权限或归档拒绝不发布。

上述评论/关联/表情的四槽值固定如下；删除把增加一行中的 New/NewID 移到 Old/OldID，另一侧为 nil（评论修改例外，两侧都是对应 HTML）。这些字段不另加 UUID 文本或显示名。

| Change.Field / 场景 | Old | New | OldID | NewID |
|---|---|---|---|---|
| relation_type / 增关联 | nil | 规范存储类型字符串 | nil | nil |
| issue / 增关联或工作项表情 | nil | nil | nil | 存储源工作项ID或表情所属工作项ID |
| related_issue / 增关联 | nil | nil | nil | 存储目标工作项ID |
| reaction / 增表情 | nil | 规范表情字符串 | nil | nil |
| comment / 增评论表情 | nil | nil | nil | 所属评论ID |
| comment_html / 增评论 | nil | 清洗后HTML | nil | nil |
| comment_html / 改评论 | 锁下旧HTML | 清洗后新HTML | nil | nil |

### 2.8 T7：任务登记、队列与配置

worker 种类必须与实际登记绑定，不能信任外部填写、可能与 Add 不一致的 Kind。`jobs.Register[T river.JobArgs](river.Worker[T]) *WorkerRegistration` 封装私有 kind/add；`Job{Worker *WorkerRegistration, Periodic *PeriodicSpec}` 中周期声明独立于 worker，`PeriodicSpec{Args river.JobArgs, Interval time.Duration, ID string, RunOnStart bool, InsertOpts *river.InsertOpts}` 经公共 API 转成 River 周期对象，不反射私有字段。

`jobs.New(pool,cfg,definitions,enqueuedKinds)` 先登记，再独立检查每个 PeriodicSpec.Args.Kind 和 EnqueuedKinds 都有 worker；缺任一种构造失败。重复 worker/周期 ID、nil及typed-nil worker、无效 args、worker 数或周期同样失败。Register生成的描述即使收到无效worker也必须由New在登记/启动前拒绝，不得把它当成合法空Worker跳过；无需检查River私有字段。T7同步迁移现 identity cleanup、bootstrap的New四参调用/worker配置以及相关测试构造点，保证当Task可编译并启动；T10接入自动归档，T11负责最终完整性验收，不留旧回调兼容分支。漏一个完整定义由 bootstrap 的外部预期种类 completed 测试捕获，不能从实际定义反向造期望。

`jobs.default_workers` 默认10、`jobs.periodic_workers` 默认2，均正整数；identity cleanup 与 auto_archive 在 periodic，default 留给事件任务。`issue.auto_archive_interval` 默认24h、测试2s、最小1s（实际随T10归档worker落地；T7只负责worker数和身份清理迁移）。启动配置、环境变量覆盖、校验、日志与模板同改。现两处 Stop/Start 测试改为状态确认与期限，期限必须区分立即停止和 firstRetry 等待，不能固定 sleep 同步。

### 2.9 T8：统一业务时钟

`clock.Offset{Base interface{ Now() time.Time }, Offset time.Duration}.Now()` 对 Base.Now 加偏移后统一 UTC、截断到微秒。配置 `clock.offset` 默认0，仅 env=test 接受非0（正负均可），dev/prod 非0启动失败；测试真实配置载入和环境变量覆盖，不只调字段校验。

bootstrap 单一 `newClock` 同时供 serve、Users、Workspaces 的所有业务用例；不得只偏移自动任务。CLI 只共用纯时钟构造，不接 jobs/dispatcher/issue.New。River 调度、完成时间、重试、限速及等待期限继续真实时间；不提供运行时改时钟接口或信号。

### 2.10 T9：自动归档目录与 SQL

消费方 `issue/app.ArchivePolicies.ArchiveProjects(context.Context) ([]uuid.UUID,error)` 只返回未删除、未归档、archive_in∈[1,12] 的项目 ID，按 ID 排序。project 只查自家 projects；现有两种 Lock 的返回小值加 `ArchiveIn int`，bootstrap 转换；扫描值不是锁下策略的权威。

`AutoArchiveRows{ProjectID uuid.UUID, StateIDs []uuid.UUID, Cutoff time.Time, At time.Time}`；`AutoArchiveStore.AutoArchive(context.Context, AutoArchiveRows) ([]uuid.UUID,error)` 与 `InsertActivities(context.Context, []ActivityRow) error`。issue 自有一条 UPDATE RETURNING：同项目、未删除、未归档、state_id∈StateIDs、updated_at<=Cutoff；写 archived_at=At、updated_at=At、updated_by_id=NULL。空 StateIDs 安全返回零行；不跨模块 JOIN；RETURNING 顺序不作为契约。

T10 接入用例时，Project 与 ProjectPatch 两处 archive_in 说明更新为0关闭、1–12个月、每月30天、完成/取消状态及 updated_at 阈值；生成物不手改。没有 schema/grants 增项；实际 runtime grants 验证仍是 T11 必须完成的门。

### 2.11 T10：自动归档 app 与 worker

`AutoArchiveDeps{Policies ArchivePolicies, Projects ProjectGate, States StateCatalog, Issues AutoArchiveStore, Tx shared.TxManager, Clock Clock, Events shared.EventPublisher}`；`(*AutoArchive).Execute(context.Context) (int,error)` 返回已成功提交归档数，遇错返回已提交数量与原错误，不把已回滚项目计入。

按候选项目顺序逐项目独立事务：项目 `FOR NO KEY UPDATE` → 重判存在/未归档/ArchiveIn>0 → 同 ctx 状态目录选 completed/cancelled → 一次 Clock.Now → cutoff=now−ArchiveIn×30×24h → 一条归档 UPDATE → 全部动态 → 全部事件。无项目/禁用/无 eligible 状态跳过；错误原样返回，之前项目已提交保留，River 重试时已归档行排除，不重复动态/事件。

后台不进请求写闸门、不伪造用户 Actor、不取工作区锁；`issue.Deps.JobTx` 接原始 `postgres.TxManager`（同一个 `issue.write_lock_timeout` 配置），AutoArchiveDeps.Tx 消费它；不经过请求侧 busyTx 的55P03→ServerBusy转换，因此向River保留原PG错误链。请求侧Tx仍经busyTx，ReadTx仍独立快照；bootstrap复用同一底层构造，模块不写SET。当前项目 N 串行其活行写，归档不碰跨项目关系；不另外预锁每个 issue、不把项目事务拆成每行事务。排序返回 ID，所有动态和事件同刻，`ActivityRow.ActorID *uuid.UUID` 真正传 nil，普通写显式保留原非 nil Actor。动态 archive、事件 issue.archived 的 actor 均 NULL，不增关注、不变父子。

River `issue.auto_archive` worker 只调用 Execute，错误交 River；周期进入 periodic、RunOnStart 与既有 identity 一致。固定时钟验证 cutoff−1µs、cutoff、cutoff+1µs，完成/取消与各排除状态、删除/已归档/其他项目，以及两次运行幂等。

### 2.12 T11：最终组合与运行角色

服务顺序：各 Provide → 业务 Clock、只投递客户端 / wrapper、闸门 → access → dispatcher → issue.New → project/workspace/identity → 在所有声明任务的模块构造后工作客户端 jobs.New 一次 → instance/router；服务运行时再 Start。只投递客户端在模块前完整可用；工作客户端汇总全部 worker，不交给模块。

模块 `Jobs() []jobs.Job` 汇总 identity cleanup 与 auto_archive；`EnqueuedKinds() []string` 本期为空，不能把定时任务冒充请求投递声明。测试独立预期两种任务均实际 completed，校验 periodic 队列、并发上限与互不挤占。复用精确 runtime role/grants 环境实际自动归档；目录 grants 测试与真实用例运行都要通过。CLI 调用图继续不可达 jobs/River/issue.New。

### 2.13 T12：交错 18 与锁后重判

两个方向：请求先把旧完成项改进行中，任务随后不归档；任务先归档，请求锁后返回409 issue.archived。另覆盖扫描后 / 等锁期间 archive_in 改变、关闭、项目归档/删除；一律以锁下新值为准。用持锁、channel、pgtest.RowWaits 确认阶段，`-race -count=5` 无40P01。

时钟在锁等待期间推进，由同步屏障保证 clocktest.Fixed 的 Advance 与 Now 不并发；断言最后项目锁后才读取时钟。已成功项目与故障项目分开，真实发布失败证明该项目更新/动态/任务整体回滚。保持现有跨项目关系写的锁顺序与预检路径。

### 2.14 T13：独立 clock fixture

`e2e/fixtures/clock.ts` 提供可直接调用的普通函数与 typed ClockWorld，由 test.ts 接 fixture；复用 createDatabase/openDatabase/startNerve 等底层设施，独立创建唯一测试库、连接与日志，不复用 worker 绑定的 db/nerveWith/databaseSnapshot。

正常进程准备数据 → 完整停止 → 同库偏移进程；同库任何时刻只有一个 nerve。启动前记录真实 startedAt，finally 无论哪个步骤失败都停止自己进程、关闭自己连接并处理自己库；失败附件来自自己的库及两次进程日志。不可继承会误取 worker 库的自动快照 fixture，不影响其他测试/容器。

### 2.15 T14：I16 接口故事

全部前置经 API：创建账户、无到期 PAT、工作区、archive_in=1 项目、旧 completed/cancelled 项与旧未完成状态对照；另 archive_in=2 项目保留旧项。启动偏移744h（31天）进程，再经 API 创建/更新“最近完成”对照；用 PAT 访问，不复用失效 JWT。

最后一条recent对照写成功后，从ClockWorld自己的数据库读取 `SELECT clock_timestamp()::text AS watermark`，原字符串保留数据库微秒，不经JS Date截断。限时轮询 `issue.auto_archive` 的 completed、`attempted_at > $1::timestamptz`（参数为watermark），并保留finalized_at晚于偏移进程spawn前startedAt的条件。这证明至少一轮任务在全部对照写完后才开始；仅finalized_at晚于写入仍可能用了扫描过早的任务。随后核对目标 archived_at、默认列表消失/归档列表出现、详情与动态 actor=null，同戳数据库行和所有对照保持。准备期已完成任务不能满足断言；不等待真实天数、不固定 sleep。页面版重新登录及时间线显示归 P15。

### 2.16 T15：跨层证伪与全量门禁

12 类型逐项核对 Entity、Changes、actor、一次时钟、业务写全完成后的同 ctx 发布；无改动/skip/幂等/拒绝不发布，评论/表情/关联无伴随 issue.updated。测试三个订阅者“投递成功→sentinel失败→不得执行”，真实 PostgreSQL 中业务写与前项投递均回滚；批量最后一项失败整批回滚。

实际变异至少覆盖：Publish 移出事务/吞错、只发根、关系双发、sort_order漏发、skip误发、删除表情错ID、NULL换零UUID、漏worker两类、<=换<、锁前读策略/时钟、状态过滤遗漏、fixture误认历史完成。逐个记录断言失败层、恢复与复测；基础设施失败不算杀死变异。全部既有权限/隐藏404/契约/架构测试继续过；适用全量门禁及新增交错五遍通过后才记完成。

### 2.17 T16：文档、交接与交付门

同步设计§3.23/P6：v0§6.2/6.3 的事件事务约定、§6.7 的队列/调度/测试时钟；M2§3.15/3.17 只投递客户端与启动核对；plane-diff 的自动执行者为空与按间隔调度；README 默认 worker 数/autoarchive 间隔、test-only clock。处理本规格§7输入，交接整体 status 保持 open。

每 Task 记录实际测试/变异/独立评审；规格验收只据真实证据打勾。**准备期原安排：** 当时仅准备规格与预检，不执行产品任务，拟实施后交负责人安排整分支终审。**实施后结果：** T1–T16逐任务独评与内部整分支审查均已完成；后续分支CI、外部终审及获授权后的 --no-ff、精确 main SHA CI、归档/清理仍按交付门办理，不自行合并 P6。

## 3. 与设计的差异和补充（局部裁定）

| 裁定 | 结论 | 理由 | 判断错误的代价与验证 |
|---|---|---|---|
| 推进方法 | 紧凑文件/签名/断言计划，逐 Task 红绿、生产变异与独立评审，替代预制完整原型/逐字重放 | 02-method§3 允许调整；“只到预检”是准备期授权边界，后来已获T1–T16实施授权 | 接缝可能实施才暴露；附录不冒称运行，T15补跨层证据 |
| 规模 | 设计11项细化为16任务，估计最大手写差异约1200行并留10%余量；实施前重新量 | 发布点、归档与fixture分别可评审，不缩范围 | 估计偏差可按既有D2再切实施单元并记账 |
| 事件完整性 | 创建白名单；更新语义字段加sort_order；state涵盖派生completed_at | 动态既缺创建标签又不记排序，不能作为全部事件来源 | 漏/冗余字段影响后续通知；T3精确值与无动态反例 |
| 删除 payload | 具体类型表优先：issue.deleted空；评论/关联/表情删除带旧值上下文 | 消除设计“删除时为空”概括与具体表的歧义 | 后续消费者依赖该形状；T4–T6精确schema及M8交接 |
| 关系与表情标识 | 关系固定存储两端；两表情Kind分开，Entity为真实行ID | 事件不能依赖调用方向或资源ID猜实体 | 指向错误对象；跨项目反方向及三种不同ID反例 |
| 归档编码 | 复用manual_archive/archive/restore；事件时间在OccurredAt | 保守落实与动态同形，避免两种编码混用 | 外部要旧精确归档时刻需扩契约；本期明确不承诺 |
| 分发边界 | 未知发布拒绝，订阅值逐份深拷贝，无nil静默兜底 | 捕获漏登记与错误修改，保持确定性 | 增复制成本；类型/顺序/污染反例及文件规模核对 |
| P6空订阅 | Subscribers仅返回明确空集合，接收通用Inserter，不造LinkCrawls | 设计明确生产消费者在P7；当前用真实测试订阅证明 | 生产投递联通到P7再验；T1/T2/T15不冒称已有抓取 |
| 任务描述 | worker类型绑定实际注册，独立PeriodicSpec与EnqueuedKinds核对 | River公开API不提供已登记kind/周期构造函数的反向检查 | 平台内部API调整；两类真实漏登记变异与组合完成记录 |
| 系统归档锁 | 后台项目N、每项目事务、不进请求闸门/不造actor；独立JobTx保留原PG错误，同配置锁超时 | 设计§3.5已排除后台闸门；无用户可授权 | 误接请求路径或锁环；T12双向交错/超时和调用图 |
| 事件ID/四槽 | 独立非零Event.ID；评论/关系/表情逐槽冻结 | 避免实体ID冒充事件身份或同字段不同编码 | T2–T6置零/复用/错槽反例，未来消费者按固定shape接入 |
| 时钟覆盖 | serve与CLI共用纯newClock业务装饰器，River仍真实时间 | “全部用例”一致且不引入CLI任务组合 | 测试CLI时间变化；正负/零/精度/调用图验证 |
| 文档边界 | 当前无新路由/错误码/表/依赖，只改archive_in说明与本阶段约定 | 字段已存在，后台能力随消费者落地 | 实施若超范围必须重新分析，架构/高风险提交后上报 |

## 4. 逐项验收与证据

T1–T15 的运行命令、退出码、变异和精确提交见[评审 §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)及各任务报告/独评。准备期的静态预检只证明当时的文档；以下勾选依据后续真实实施证据。

| 完成 | Task | 完成线 |
|---|---|---|
| [x] | T1 | 同事务提交/回滚、无事务拒绝、提交前不可见，未启动worker也可投递 |
| [x] | T2 | 12类型目录、登记拒绝、固定顺序/深拷贝、生产注入与合法空订阅 |
| [x] | T3 | 创建完整字段、修改/描述/排序、skip与no-op事件边界 |
| [x] | T4 | 子树每项事件、归档恢复幂等、同戳与发布失败回滚 |
| [x] | T5 | 实际移动子项、关系一行一事件、存储方向、批量末尾发布 |
| [x] | T6 | 评论与两表情实体/actor/旧新值，拒绝无事件，无伴随issue.updated |
| [x] | T7 | 两队列实际并发、两种漏worker构造拒绝与变异、期限测试 |
| [x] | T8 | test-only offset、serve/CLI全部业务时钟一致，River真实时间不变 |
| [x] | T9 | 候选/锁下策略、自有SQL及恰好阈值、锁返回策略/生成物 |
| [x] | T10 | 每项目事务、系统NULL、固定时钟/状态反例、幂等与部分已提交后重试、Project/ProjectPatch说明 |
| [x] | T11 | 两种任务真实completed、精确runtime grants归档、CLI隔离 |
| [x] | T12 | 交错18两方向、策略变更/删除等待、锁后时钟，race五遍无40P01 |
| [x] | T13 | 独立普通函数/fixture、一库一进程、自己的失败日志/快照与清理 |
| [x] | T14 | I16 API前置/三层断言、31天与阈值对照、最后对照写入后开始且completed的任务 |
| [x] | T15 | 全发布矩阵、真实订阅回滚、实际变异恢复、全量门禁与旧故事回归 |
| [x] | T16 | 中文正式文档、设计/README/差异同步、输入交接、静态校验及提交；初评一处措辞问题由`6995d523`修复并经聚焦复审通过 |
| [x] | 内部整分支审查 | `ee26db29..6995d523` 的19提交/165文件已审，Critical/Important/Minor均0；见[评审 §5](../reviews/P6-issue-events-review.md#5-最终实现与-plan-的差异) |
| [ ] | 后续交付门 | push 后按精确 HEAD 查分支 CI；外部终审另待安排 |
| [ ] | 未授权集成门 | 仅经后续明确授权才 `--no-ff` 合并、查精确 main SHA 的 CI、归档与清理 |

## 5. 不在范围内与后续 Phase

| 接收方 | 明确保留的工作 |
|---|---|
| P7 | link事件3种、首个生产订阅者/LinkCrawls、出站防SSRF、抓取、registrationRows对river_job的请求任务核对 |
| P8 | 草稿保存不发事件，发布复用issue.created；搜索/link_count，不能预埋未来字段 |
| P9 | project/workspace.deleted、delete_rows订阅与任务，移除同步过渡删除端口，版本保留与物理清理 |
| P15 | I16页面版重新登录、NULL actor按站点名显示，不按邮箱/名字猜系统用户 |
| M5 | M2命令只投递原文交接：首个真正投递命令才接CLI，附平台客户端已在 |
| M6/M7 | 自动归档加入迭代/模块/收集箱真实表条件；通知发送前再判收件者权限 |
| M8 | Webhook消费明确事件名单/形状；项目删除连带与DetachLabel不另发issue事件的清单 |
| M4收尾 | 所有handoff逐条closed、总体§9.4、M4§14完整证据、M5–M8四份closeout；P6不提前关闭 |

## 6. 风险

| 风险 | 控制与证据责任 |
|---|---|
| 事件在事务外 / 批量半写即发布 | 原ctx、业务写全完成后发布；三个订阅者与批量末项故障真实回滚（T15） |
| 动态并非事件全集、变化条件失真 | 创建标签/排序/描述合并、skip/no-op与评论表情精确反例（T3/T6） |
| 注册元数据自证造成漏worker无人发现 | 强绑定真实登记、周期独立、外部预期completed及两类变异（T7/T11） |
| 扫描旧策略、空状态数组或跨模块JOIN | 只候选ID、锁下重新读、空集零行、SQL作用域守卫（T9/T12） |
| NULL actor被零UUID或原作者代替 | nullable持久化与真实读回、手动动作不丢actor的对照（T10） |
| 时钟偏移混入River真实时间/fixture误用worker库 | 统一业务时钟、startedAt真实时刻、独立资源生命周期及附件（T8/T13/T14） |
| 交错测试本身数据竞争或睡眠同步 | clocktest推进屏障、RowWaits与期限、race五遍（T12） |
| 规模估计不是原型实测 | 每任务实施前量级检查，保留修复余量；不以静态预检声称运行安全 |

## 7. 已知限制、交接与关闭条件

| 输入 | 本期接收 / 关闭证据 | 仍开放 |
|---|---|---|
| P1a/P1b/P5 spec§5、review§6 | archive_in消费；已有创建/修改/排序事件与审计同戳（T3/T9） | 页面/后续字段 |
| P2/P3 spec§5、review§6 | 删除/归档/恢复/父子/关系事件，自动归档与锁下时钟（T4/T5/T10/T12） | P9异步删除与交错24/25 |
| P4 spec§5、review§6 | 描述/评论/表情事件、skip免事件边界（T3/T6） | P15展示、M7通知/M8Webhook |
| M1-P5交接 | 服务端自动归档NULL执行者真实证据（T10） | 页面与收集箱创建者 |
| M2-closeout§1/3/6/13 | 只投递平台、固定时钟、登记变异、事务约定/grants、clock普通函数（T1/T7–T14） | CLI原文转M5；其他章节原落点 |
| M3-closeout§1/2/3/4/8 | 两处期限、后台锁论证、既有权限回归、archive_in两契约、phase关键词（T7/T9/T12/T15/T16） | 邀请删除保持同步；P7 registrationRows，P9删除任务；旧前端例外 |

阶段首提交已将关键词 phase 切 M4/P6；没有 until=M4/P6 的到期例外，P11/closeout 的前端例外保留。实现完成后上述输入只追加“处理结果（M4/P6）”及剩余责任，不改整体 status 为closed。M4§14“事件和任务”还含P7/P9条目，P6只能记子项证据。

实施验收依据§4 T1–T16 的实际证据；任务独评与内部整分支审查已完成，审查无新增发现。精确 HEAD 分支 CI、外部终审仍是后续交付门，当前不声称已完成。遵循用户指令，阶段完成先不合并；仅获明确授权后才进入 `--no-ff`、精确 main SHA CI、证据归档和清理的集成门。文档为中文，提交按用户最新AGENTS用详细中文说明；主目录HEAD始终main，阶段使用约定worktree，推送优先/usr/bin/git。

## 附录 A. 证据边界与准备历史

准备阶段只读调研、三个独立视角静态预检及修订复审曾记录为“未实施”，属于起步历史。后续 T1–T15 已在 `codex/m4-p6` 分别实施并独评；当前已验证源码 HEAD `4a489125f480402692772cbae483af26d9600792`。T15 在该源码哈希上顺序完成 `make test`、`make lint-go`、`make lint-web`、`make knip`、`make test-web`、`make e2e`、提交后 `make gen-check`；Go 为49个有测试包通过，E2E 149/149，Web 的缓存命中数见[评审 §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)，不把缓存称为新执行。20个真实生产变异与16写用例/12事件目录见评审所引矩阵。T2 早期聚焦命令曾空匹配，最终证据依赖后续实际匹配与完整包回归；T13最后门禁只留报告，T14/T15有完整原始输出。早期 testcontainers 可选 shell 探测及旧 Playwright 颜色警告保留原始日志；后续无警告的定向颜色验证已闭环，不能称所有历史输出 pristine。

P5 main `ee26db29c8a55cbb034a2da38c794c7f1af88e4e` 的 [CI 40257678959](https://github.com/open-nerve/NerveProject/actions/runs/40257678959) 仅证明基线。T16后的内部整分支审查已在`6995d523`完成且无新发现；精确 HEAD 分支 CI、外部终审与授权集成尚无结果，不在文档里预填成功。最终提交与 CI 的精确 SHA 由 Git 和交付答复给出，避免为文档自证自身 CI 无限提交。
