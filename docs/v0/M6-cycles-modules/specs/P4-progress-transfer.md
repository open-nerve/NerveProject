# M6 / P4：进度、冻结与转移

> 状态：待预检（2026-11-10）。基线 main `ed8f0df4e8982dc50a4560a31fdf9b2964d84ac7`（P3b 的 noff，main CI 42621199100 首次通过）。设计见 [M6 设计](../M6-design.md) §2 的 G5–G7、G12、G17，§3.4、§3.5、§3.7、§3.8、§5、§6、§9；负责人 2026-11-07 的决定 A/B/C 见设计 §15（第 5、6、7、8、11、14、15 条与本段直接相关）。加锁之路沿用 M4（[硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md)）与 [P1 spec](P1-cycles-backend.md) §3；归属的写法与动态沿用 [P3a spec](P3a-issue-membership.md)。现状调研见本阶段台账 `research/backend.md`、`research/plane-web.md`（ignored，随阶段归档）。

## 1. 目标与边界

P4 交付设计 §12 P4 一行：

- **进度**：`GET /cycles/{cycle_id}/progress`、`GET /modules/{module_id}/progress`，回答按状态组的计数、负责人与标签的分布、燃尽序列；迭代另带冻结。
- **列表行的计数**：迭代与模块的列表（含归档目录）每行带实时计数；迭代行另带负责人 id 与冻结的计数。
- **冻结**：`cycles.progress_snapshot`，由周期任务 `planning.freeze_cycles` 与转移写入，先写者为准，之后不覆盖。
- **转移**：`POST /cycles/{cycle_id}/transfer-issues`，把已结束迭代里未完成的工作项转到另一个迭代，每个转走的工作项一条动态。

覆盖 G5、G6、G7、G12 的接口与数据库证据，G17 中进度、计数与转移的矩阵行；冻结与转移的交错。页面在 P5（迭代）与 P6（模块）。

**不做：**
- 进度百分比：由网页每域一个函数计算（设计 §3.5，P5/P6）；服务端只给计数。
- 迭代详情 `GET /cycles/{id}` 与各写入的回答不加计数（R9）。
- 模块不冻结（Plane 同，设计 §3.5）。
- 估算点数（M1 已删）；工作区级的进度（M7 无此需求，M1-P3 交接）。

网页只做三件事：重新生成的类型；e2e 夹具随列表行的新类型改名；`planning.cycle_completed`、`planning.cycle_not_completed` 两条文案改为在每个回答它们的场合都成立（R11）。

## 2. 交付合同

### 2.1 数字的口径（R1–R4）

**范围**：迭代/模块里有未删除归属、未删除、未归档、调用者看得见的工作项。子工作项照常计入；草稿不是工作项（`draft_issues` 另表）。调用者是只看自己的访客（OwnOnly）时只算他建立的工作项（同列表引擎的谓词，建立者为空的不算）。

| 数 | 定义 |
|---|---|
| 计数 | `total_issues` 与五个状态组 `backlog_issues`、`unstarted_issues`、`started_issues`、`completed_issues`、`cancelled_issues`；`total` 恒等于五者之和 |
| 分布 | 负责人、标签各一张表；已取消的不计入。每行 `total_issues = completed_issues + pending_issues`（pending = 积压、未开始、进行中）。一个工作项有几个负责人/标签就计入几行；没有的计入 id 为 `null` 的一行。只回答总数大于 0 的行，按 id 升序，`null` 行最后 |
| 燃尽 | 迭代从开始日到结束日、模块从开始日到目标日，按项目时区逐日一点 `{date, remaining}`。`scope` = 范围内除已取消外的工作项数；某日的 `remaining` = `scope` − 该日结束前已完成的数量；今天（项目时区）之后为 `null`。已完成指状态组为已完成，完成日为 `completed_at` 换到项目时区的日期；状态组为已完成而 `completed_at` 为空的（状态改组后才会出现）从第一天起算已完成，状态组不是已完成的即使带着旧的 `completed_at` 也不算。缺日期（草稿迭代、缺任一日期的模块）或跨度超过 3660 天时整个燃尽为 `null` |

### 2.2 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/planning/domain/` | `progress.go`：状态组、`WorkItem`（状态 id、`completed_at`、负责人 id、标签 id）、`Counts`、分布行、`Burndown`、`Progress`、`Frozen`；纯函数由工作项事实、状态组、日期跨度、时区与今天算出 `Progress`（§2.1）；冻结的 JSON 编解码（类型化，字段名同接口）。`BurndownDays = 3660` |
| `server/internal/modules/planning/app/` | 端口（消费方声明，R2）：`IssueStats`（设计 §6 的 `PlanningIssueStats`）、`IssueTransfer`（设计 §6 的 `TransferCycleIssues`），`Projects` 加 `StateGroups`。用例：`GetCycleProgress`、`GetModuleProgress`（读之路）、列表两个用例带计数（R9）、`FreezeCycles`（周期任务，R6）、`TransferCycleIssues`（写之路，R7、R8）；`UpdateCycle` 改日期时清掉冻结（R5） |
| `server/internal/modules/planning/adapter/` | `postgres`：冻结的读（列表行只取计数与时刻，进度取整份）、只在为空时写入、待冻结的项目与迭代、改日期时清空。`http`：三个新处理器，列表处理器回答新行。`river`：`planning.freeze_cycles` 周期任务。`module.go`、`module_jobs.go`：接线与 `Jobs()` |
| `server/internal/modules/issue/` | 实现 planning 的两个端口：`adapter/postgres` 的统计语句（单个迭代/模块的工作项事实；一批迭代/模块按状态的计数；一批迭代的负责人 id）与转移语句（锁源迭代的归属行、软删、插入目标行、回答工作项 id）；`domain` 导出 `CycleMoved(from, to Named)`（`cycleActivity` 的 `updated`）；转移写动态并 `MarkChanged`。经 `issue.Provide` 交出（只用连接池，先例 `MembershipRemovals`） |
| `server/internal/modules/project/` | `PlanningProjects` 加状态的读取（id 与状态组），复用 `IssueStates` 的语句，不新增 SQL |
| `server/internal/bootstrap/` | 接线与转换（`planningProjects.StateGroups`、`planningIssues`）；`planning.Deps` 新字段；周期任务进入六处周期种类清单与 `waitForStartupRuns`；组合测试、矩阵、交错 |
| `server/internal/platform/config/`、`server/configs/` | `planning.freeze_interval`（默认 1h，最少 1s，日志键）；`config.test.yaml` 为 2s |
| `server/migrations/sql/00048_planning_cycles_progress_snapshot.sql` 与测试 | §2.4；迁移总数 47→48（`trigram_schema_test`）、`TestCyclesSchema` 的列；sqlc 的 planning 条目加 `00048` |
| `api/modules/planning.yaml` | §2.3 的三个操作与 schema；`CycleList`、`ModuleList` 的行改为 `CycleListItem`、`ModuleListItem` |
| `web/` | 重新生成；两条文案（R11） |
| `e2e/` | §2.7 |
| `docs/` | P3b 回填（T1）；`plane-diff.md`；输入交接的“处理结果（M6/P4）”；设计 §3.5、§5、§6、§9、§14 |

### 2.3 接口与错误

| 操作 | 路径 | 回答 | `x-problem-codes` |
|---|---|---|---|
| `getCycleProgress` | `GET /api/v0/cycles/{cycle_id}/progress` | `CycleProgress` | `planning.cycle_not_found`、`forbidden` |
| `getModuleProgress` | `GET /api/v0/modules/{module_id}/progress` | `Progress` | `planning.module_not_found`、`forbidden` |
| `transferCycleIssues` | `POST /api/v0/cycles/{cycle_id}/transfer-issues`，体 `{target_cycle_id}` | `CycleTransferResult` | `validation_failed`、`planning.cycle_not_found`、`forbidden`、`project.archived`、`planning.cycles_disabled`、`planning.archived`、`planning.cycle_not_completed`、`planning.cycle_completed`、`server_busy` |
| `listCycles`、`listModules` | 不变 | 行改为 `CycleListItem`、`ModuleListItem` | 不变 |

**Schema**（字段名尽量沿用 Plane 与网页现读的名字，R9）：

“六个数”指 `total_issues`、`backlog_issues`、`unstarted_issues`、`started_issues`、`completed_issues`、`cancelled_issues`（int64，必填），在下列 schema 里逐个写出。

| 名字 | 字段 |
|---|---|
| `AssigneeDistribution` | `assignee_id`（uuid 或 null）、`total_issues`、`completed_issues`、`pending_issues` |
| `LabelDistribution` | `label_id`（uuid 或 null）、同上三数 |
| `Burndown` | `start_date`、`end_date`（日）、`scope`、`points: [{date, remaining: int64 或 null}]` |
| `Progress` | 六个数、`assignees`、`labels`、`burndown`（`Burndown` 或 null） |
| `FrozenProgress` | `Progress` 的全部字段与 `frozen_at` |
| `CycleProgress` | `status`、`live: Progress`、`progress_snapshot: FrozenProgress 或 null` |
| `FrozenCounts` | 六个数与 `frozen_at` |
| `CycleListItem` | `Cycle` 的全部字段、六个实时数、`assignee_ids`（范围内工作项的未删除负责人，去重、按 id 升序）、`progress_snapshot: FrozenCounts 或 null` |
| `ModuleListItem` | `Module` 的全部字段、六个实时数 |
| `CycleTransfer` | `target_cycle_id`（uuid，必填，封闭） |
| `CycleTransferResult` | `transferred_issues`（int64） |

代码库没有 `allOf` 的先例，列表行照 `Issue`/`IssueDetail` 的做法写全字段。OwnOnly 的调用者在列表行与进度里的 `progress_snapshot` 恒为 `null`（冻结算的是全部工作项）。

**读之路**：三个读操作走 `Reads.run`（一个只读快照）：迭代/模块行、项目的状态组、工作项事实、冻结都在同一个快照里读。动作沿用 `cycle.read`、`module.read`；已归档的项目与迭代/模块、开关关闭照常读（设计 §3.8）。

**转移的拒绝次序**（R7）：结构 400 → 404 源迭代 → 403（含 OwnOnly）→ 409 `project.archived` → 409 `planning.cycles_disabled` →（锁下）409 `planning.archived`（源已归档）→ 422 `target_cycle_id` `not_allowed`（目标不存在、已删除、不在本项目、就是源本身）→ 409 `planning.archived`（目标已归档）→ 读时钟 → 409 `planning.cycle_not_completed`（源未结束）→ 409 `planning.cycle_completed`（目标已结束）→ 冻结（若尚无）→ 移动。

每个（操作，码）在 planning 的 apitest 有一个回答它的处理器测试。不新增错误码；`planning.cycle_not_completed`、`planning.cycle_completed` 各加一条只用于转移的服务端消息（同码异消息的先例：`issue/domain/errors.go:65-69`）。

### 2.4 表

`00048_planning_cycles_progress_snapshot.sql`：

- `cycles` 加 `progress_snapshot jsonb NULL`，CHECK `progress_snapshot IS NULL OR jsonb_typeof(progress_snapshot) = 'object'`。`NULL` 即未冻结（Plane 用 `{}`）。
- 部分索引 `cycles_freeze_due_idx (end_date) WHERE deleted_at IS NULL AND progress_snapshot IS NULL AND end_date IS NOT NULL`，供周期任务找待冻结的项目。
- Down：删索引、删列。运行角色已有 `cycles` 的 DML，不改授权。

存的内容就是 `FrozenProgress` 的 JSON（字段名同接口），算的是全部工作项（不按调用者过滤）。

### 2.5 冻结的周期任务（R6）

- **种类与配置**：`planning.freeze_cycles`，`periodic` 队列，`RunOnStart`，间隔 `planning.freeze_interval`（默认 1h；测试环境 2s）。不进 `EnqueuedKinds`（那里只列请求触发的种类）。
- **一次运行**：
  1. 不加锁取候选项目：有未删除、未冻结、`end_date ≤` UTC 今天的迭代的项目 id（任何时区的“今天”至多比 UTC 日期晚一天，所以这是“已结束”的超集），按 id。
  2. 每个项目一个事务（`JobTx`，不进写闸门，先例自动归档）：先取项目行锁；项目不在（已删除）则跳过；锁后读时钟，按项目时区得今天；取本项目未删除、未冻结、结束日早于今天的迭代（含已归档的）；逐个读全部工作项事实与状态组，算出冻结，`UPDATE … WHERE progress_snapshot IS NULL`；提交。
  3. 某个项目失败即返回错误与已冻结的个数，之前的项目已提交，下次运行续上。
- **不看**：项目是否归档、开关是否关闭、迭代是否归档。冻结是记录，不是用户的写。
- **不改** `cycles.updated_at`、`updated_by_id`（任务没有写者；冻结的时刻在 `frozen_at` 里）。

### 2.6 转移（R7、R8）

- **加锁之路**：`Locks.run`（动作 `cycle.write`）：不加锁预检 → 项目写闸门 → 项目行 `FOR NO KEY UPDATE` → 锁下重读源迭代、再判定 → 锁下读目标迭代 → 读时钟 → 判定已结束 → 写。
- **转走谁**：源迭代里有未删除归属、未删除、未归档、状态组为积压/未开始/进行中的工作项（不按调用者过滤：OwnOnly 已被 403）。状态组到状态 id 的对应在项目锁下读（改状态组也取项目行锁，所以锁下稳定）。已完成、已取消、已归档的留下。
- **写**：
  1. 源尚未冻结则先冻结：用这次的时钟、移动之前的数字（同周期任务的计算）。
  2. 一条语句：按 id 锁源迭代的这些归属行（`FOR NO KEY UPDATE ORDER BY id`），软删，在目标迭代插入新行（同一时刻、同一写者；不就地改 `cycle_id`，P3a R4），回答工作项 id（按 id）。
  3. 每个工作项一条动态：`updated`/`cycles`/源名→目标名，标识为两个迭代的 id（`CycleMoved`）。
  4. `MarkChanged(ids, by, at)` 更新这些工作项的修改时间与修改者。不发布 `issue.updated`（先例：删迭代的订阅者 `membership_events.go:35-37`；现在没有订阅者）。
- **回答**：`200 {transferred_issues: n}`；没有可转的时回答 0（只冻结，不写别的）。可以多次转移。
- **锁序**：项目行 → `cycles`（冻结的 UPDATE）→ `cycle_issues` → `issue_activities` → `issues`。工作项的写、删迭代、冻结任务都先取项目行，所以按项目行串行。

### 2.7 网页与 e2e

**网页**：
1. 重新生成 `api-client`。
2. `packages/i18n` 的 en 与 zh-CN：`planning_cycle_completed`、`planning_cycle_not_completed` 两条改为在改迭代、工作项加入、归档、转移各处都成立的说法（R11）。

**e2e**：
- 夹具：`fixtures/cycles.ts` 的列表随新行类型；进度、转移的调用；数据库断言读 `progress_snapshot`、转移后的归属与动态（沿用 `assert/issue-planning.ts` 的 `cycleMoved`、`expectPlanningActivity`）。
- 新故事放 `stories/planning/`：G5（迭代进度、分布、燃尽、访客与 OwnOnly 的计数）、G6（冻结：任务写入后改状态、移出，冻结不变，实时变化）、G7（转移：转走的与留下的、动态、多次转移、拒绝）、G12（模块进度）。
- 让迭代成为已结束：建一个进行中的迭代、经 PATCH 把工作项加入，再把迭代的日期改到过去（P1 只拒绝“改之前已结束”的迭代改日期）。

## 3. 局部裁定

- **R1 口径**：
  - 范围同设计 §3.5：只算调用者看得见、未归档、未删除的工作项与未删除的归属。子工作项计入（Plane 同：它的统计不排除子工作项）。
  - 五个状态组之和就是总数。分诊组不收工作项（`issue/domain/issue.go:36-45`），遇到范围外的组是缺陷，回答 500，不静默丢弃。
  - OwnOnly 只看到自己建立的工作项（`HiddenFrom`、`list_issues.sql:24` 同一谓词）。同一调用者的进度总数等于列表引擎 `cycle_id=<id>&sub_issues=true` 的总数，T5 以一个对照测试钉住。
- **R2 计算在 planning，事实来自 issue**：
  - issue 的语句不能连 `states`，planning 的语句不能连 `issues`（`TestSQLCSchemaScope`）。所以 issue 只回答按状态 id 的事实，planning 经项目的状态读取把状态 id 对到状态组，在 `planning/domain` 用纯函数算出全部数字（时区逻辑与 `domain.Today` 在一处，可单测）。
  - **`IssueStats`**（planning/app 声明）：
    - `CycleItems(ctx, projectID, cycleID, viewer)`、`ModuleItems(ctx, projectID, moduleID, viewer)`：每个工作项一行（id、状态 id、`completed_at`、未删除负责人 id、未删除标签 id）。
    - `CycleCounts(ctx, projectID, cycleIDs, viewer)`、`ModuleCounts(...)`：按（迭代/模块，状态 id）的个数；`CycleAssignees(ctx, projectID, cycleIDs, viewer)`：按迭代的去重负责人 id。列表一次取整页，没有 N+1。
    - `viewer` 为 OwnOnly 调用者的 id，否则为空（看全部）。
  - **`IssueTransfer`**：`TransferCycleIssues(ctx, Transfer{Project, Workspace, From, To Named, StateIDs, By, At}) (int, error)`，在调用者的事务里（经 ctx，先例 `Cascade.DetachLabel`）。
  - **项目的状态组**：`Projects` 加 `StateGroups(ctx, projectID)`，bootstrap 经 project 的 `IssueStates` 转换。
  - issue 一侧经 `issue.Provide`（只用连接池）交出，`bootstrap/ports.go` 转换值类型；`newPlanningRoute` 等手搭 `planning.New` 的测试随之补字段。
- **R3 分布**：负责人与标签两张，已取消的不计入（设计 §3.5），已完成按状态组（Plane 按 `completed_at` 是否为空，取消的算未完成）。只给 id，不给名字、头像、颜色：同 `Issue.assignee_ids`/`label_ids`，网页按成员与标签目录显示（Plane 的网页本来就按成员 id 取头像）；冻结里引用的成员离开或标签删除时，由网页显示兜底名字（P5）。离开项目的负责人照常计入（他的归属行还在）。
- **R4 燃尽**：
  - **跨度**：点覆盖整个迭代（开始到结束），今天之后为 `null`。设计 §3.5 的“从开始日到 min(结束日, 今天)”指有值的范围，坐标轴仍是整个迭代（Plane 同），T13 改写设计的这句话。
  - **范围不含已取消**（设计 §3.5）；用当前归属，后加入的从第一天起计入（Plane 同）。
  - **完成日**：见 §2.1。状态改组后 `completed_at` 可能过时（`project/app/update_state.go:48-55` 只改状态，不回写工作项）。按状态组判断是否已完成，与计数一致；完成日缺失时从第一天起算，使最后一个点与计数一致、过去的点不随日子变。
  - **上限**：日期没有跨度上限（年份 0001–9999），不设上限时一个迭代的燃尽可达数百万个点，写进回答与冻结。跨度超过 3660 天（约十年）时燃尽为 `null`，计数与分布照常（Plane 没有上限，登记差异）。
  - **今天**：项目时区，用读之路的时钟（冻结用锁后的时钟）。
- **R5 冻结**：
  - 写一次：`UPDATE cycles SET progress_snapshot = … WHERE id = … AND progress_snapshot IS NULL`；周期任务与转移都在项目行锁下写，先写者为准。
  - 读：列表行只回答冻结的六个数与时刻（不读分布与燃尽，列表轻）；进度回答整份。OwnOnly 回答 `null`。
  - 不改 `updated_at`/`updated_by_id`（R6）。
  - **改日期即清掉冻结**：已结束的迭代只能改排序（P1），所以冻结一般不会过时；唯一的例外是改项目时区让“今天”往回退，已冻结的迭代暂时回到进行中、又能改日期。这时 `UpdateCycle` 改了开始或结束日就在同一写里把 `progress_snapshot` 置空，迭代再次结束时重新冻结。不改日期则冻结保留（读者按 `status` 决定用哪份，设计 §3.5）。
- **R6 冻结任务**：先例自动归档（`issue/app/auto_archive.go:35-108`）：候选不加锁、权威判定在项目锁下、锁后读时钟、每项目提交。冻结不看项目归档、开关与迭代归档：它记录迭代结束时的事实，用户随后照样能读（设计 §3.8 读取照常）。
- **R7 转移的规则与次序**：
  - 源：须已结束（Plane 的公共接口规则；应用接口不查，不照搬）且未归档。目标：同项目、未删除、未归档、不是源本身、不是已结束。
  - 目标的问题在 422 与 409 之间按 P3a R2 的写法：不存在类的合成字段错误 `target_cycle_id` `not_allowed`（不以 404 泄露），归档 409，已结束在读时钟之后 409。
  - “已结束”用这次写的时钟，在最后一把锁之后判定（P1 `update_cycle.go:54-59`、P3a R2 同）。
  - 没有可转的：200、0。双击或并发的第二次转移不该报错；源若尚未冻结，这次照样冻结（源已结束，冻结总是对的）。设计 §3.5 的“转移本身看实时的未完成数”指转哪些，不是拒绝的条件，T13 写明。
- **R8 转移的写**：§2.6。不就地改 `cycle_id`，使归属行的历史同 P3a。不发布 `issue.updated`：批量移动的先例（删迭代）也不发布，现在没有订阅者；M7 的通知若需要，再由 M7 定。
- **R9 回答的形状**：
  - **列表行另立 schema**：`Cycle` 也是每个写入（建、改、归档、恢复）的回答，加必填计数会让每个写都要统计。列表行（`CycleListItem`、`ModuleListItem`）另立，`getCycle`、`getModule` 与写入的回答不变；详情页的数字来自 `/progress`（设计 §5 只要求列表行带计数）。
  - **迭代行带 `assignee_ids`**：Plane 的迭代列表行有它，网页的列表行头像与侧栏成员读它（`cycle-list-item-action.tsx:237-241`、`sidebar-details.tsx:78-81`）；在同一批统计里取，范围同计数。设计未写，T13 补进 §5。
  - **燃尽改为数组**：Plane 是 `{日期: 数}` 的映射，横轴次序靠 JSON 键序，生成的类型也差；数组带 `scope`，理想线从 `scope` 起画。P5 本来就要按 M4 交接 M6-04 改图表（0、1、多个点，`null` 不画成 0）。
  - **`{}` 改为 `null`**；没有估算点数的字段（M1-P2 交接）。
- **R10 权限**：不加新动作。进度沿用 `cycle.read`/`module.read`（管理员、成员、访客）；转移是 `cycle.write`（管理员、成员；OwnOnly 403，设计 §3.7）。列表行与进度的数字按调用者的可见范围算，OwnOnly 不读冻结（设计 §3.5）。
- **R11 两条文案**：`planning.cycle_completed` 的网页文案“只能改排序”在 P3a 起已用于“工作项不能加入已结束的迭代”，P4 再用于转移的目标；`planning.cycle_not_completed` 的“只有已结束的迭代能归档”再用于转移的源。码表达的是同一个事实（迭代已结束/未结束），改文案而不新开码：两条改成各场合都成立的说法（en 与 zh-CN）。服务端消息按场合各写一条（§2.3）。
- **R12 Plane 差异**（登记进 `plane-diff.md`）：
  - 燃尽按项目时区切天、判断今天（Plane 按 UTC，跨度因存的时刻偏一天）；范围不含已取消（Plane 含，取消的永不燃尽）；完成按状态组；跨度上限；形状为数组。
  - 分布的“无负责人”“无标签”是真实数字（Plane 实时分布恒为 0）；已取消不计入；已完成按状态组；只给 id。
  - 冻结：任务或转移先写者为准、只写一次（Plane 只在转移时写、每次覆盖、移动 0 个也写、没有任务）；改日期清掉；`NULL` 而非 `{}`。
  - `/progress` 同时回答实时与冻结（Plane 有快照后只回答快照）；一个接口代替 `progress` 与 `analytics?type=`；模块也有 `/progress`（Plane 只在详情里）。
  - 列表行六个数（Plane 应用接口的迭代列表三个）；数字按访客的可见范围（Plane 不过滤）。
  - 转移：源须已结束且未归档；目标的五条规则（Plane 缺失或跨项目为 500，归档或同一迭代照收）；每个工作项一条动态（Plane 因崩溃一条不写）；回答转走的个数（Plane `{"message": "Success"}`）；字段名 `target_cycle_id`（Plane `new_cycle_id`）；可以多次转移（Plane 网页有快照后禁用按钮）。

## 4. 验收标准

- [ ] A1 G5：迭代的计数、分布、燃尽按 §2.1：五组与总数；进度的分母由网页算（不在服务端）；分布的“无”行、多负责人/多标签、取消不计；燃尽按项目时区（东八区与 UTC 不同日的时刻）、今天之后为 `null`、开始前完成的从第一天起算、状态改组后的两种过时、草稿与超长跨度为 `null`。
- [ ] A2 G6：周期任务冻结已结束的迭代，之后改状态、移出，冻结不变而实时变化；冻结只写一次；任务与转移两种次序交错，冻结的数字相同；改日期清掉冻结；任务不改 `updated_at`。
- [ ] A3 G7：转移只转积压、未开始、进行中的，已完成、已取消、已归档的留下；每个转走的一条动态、修改时间与修改者更新；回答个数，0 也是 200；可以多次转移；拒绝次序与码（§2.3）；转移前冻结用移动之前的数字。
- [ ] A4 G12：模块的计数、分布、燃尽（开始到目标日），缺日期为 `null`；模块不冻结。
- [ ] A5 列表行：迭代与模块列表（含归档目录）每行六个实时数；迭代行 `assignee_ids` 与冻结的计数；没有 N+1（每页固定条数的语句）；OwnOnly 只算自己的、冻结为 `null`。
- [ ] A6 G17 与加锁：三个新操作进入权限矩阵（管理员、成员、建立者、访客、工作区管理员兼访客、非成员、私有项目、撤权）；访客与 OwnOnly 的数字只算可见的；交错：冻结任务与转移、转移与工作项改迭代/改状态、转移与删目标迭代、冻结任务与工作项改状态，各两种次序；转移与冻结都在最后一把锁之后读时钟；转移的锁超时回答 503，任务的锁超时保留错误、已提交的项目不回滚。
- [ ] A7 契约与文档：每个新声明的码在 planning 的 apitest 有回答；`00048` 上行、下行、再上行；差异清单、输入交接的“处理结果（M6/P4）”、设计 §3.5/§5/§6/§9/§14 已写；P3b 的阶段记录 §7、spec A8 与设计 §14 已回填。
- [ ] A8 七项本地门禁、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P5 的 T1 勾选）

## 5. 后续责任

**P5**（迭代页面）：每域一个进度函数与它的调用者（M1-closeout“进度的口径”）；读 `/progress` 与列表行；冻结只在 `status` 为已结束时展示；燃尽图按 `points` 画、`null` 不画成 0、0/1/多个点（M4 交接 M6-04）；转移弹窗与按钮（看实时的未完成数）；冻结里离开的成员与删除的标签的兜底名字。M1-P2、M1-P3 两份交接在网页改用新接口后关闭。

**P6**（模块页面）：同上的模块部分。

**P8**：冻结任务的规模（候选的部分索引、每项目的迭代数）随清理规模一起重跑。

## 6. 风险

- **跨模块的同一事务**：转移在 planning 的写事务里调 issue 的语句（经 ctx）。锁序见 §2.6；以交错测试与“锁后读时钟”的测试钉住，变异证明它们能失败。
- **周期任务插到持锁的测试后面**（M5 P8 的教训）：新种类进入 `waitForStartupRuns` 与其余五处周期种类清单；bootstrap 测试的间隔取 1h，只有启动时的一次运行。
- **矩阵的种子**：现有矩阵的两个迭代都已结束、没有归属。转移与计数的格子需要一个未结束的目标迭代与种下的归属，加在矩阵的世界里，不改已有格子的断言。
- **`Cycle` 的回答不变**：列表行换成新 schema，e2e 与 Go 测试里把列表行当 `Cycle` 用的地方随之改类型，断言不改。
