# M6 / P4 进度、冻结与转移实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（修正后，2026-11-10，[re-62](../../../reviews/re-62-M6P4预检.md)）：H1 矩阵行随操作进入 T5/T9、H2 转移格子按格子建目标、M1 根契约、M2 `embed_test`、M3 删去“改日期清掉冻结”、M4 对照测试带 `archived=false`、L1–L6 已改入。

**Goal：**
- 迭代与模块各有 `/progress`：按状态组的计数、负责人与标签的分布、按项目时区的燃尽；迭代另带冻结；
- 迭代与模块的列表行带实时计数，迭代行另带负责人 id 与冻结的计数；
- 已结束的迭代由周期任务或转移冻结一次；
- 已结束迭代里未完成的工作项可以转到另一个迭代，每个一条动态。

**Architecture：**
- **事实在 issue，计算在 planning**：issue 的语句按状态 id 回答工作项事实与计数（不连 `states`），planning 经项目的状态读取对到状态组，在 `planning/domain` 用纯函数算出全部数字。两个方向的端口都由消费方声明、`bootstrap` 转换。
- **读**走 planning 的读之路（一个只读快照）；**转移**走写之路（项目行锁），在同一事务里经 ctx 调 issue 的转移语句；**冻结任务**照自动归档，每项目一个事务、锁后读时钟。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/River/Playwright，不加依赖。

**Spec：** [P4-progress-transfer](../specs/P4-progress-transfer.md)（§2 的合同与 §3 的 R1–R12 是本计划的约束）；设计 [M6](../M6-design.md) §3.4、§3.5、§3.7、§5、§6、§9；模板：自动归档（`issue/app/auto_archive.go`、`issue/adapter/river/auto_archive.go`）、删迭代的订阅者（`issue/app/membership_events.go`）、P3a 的归属写法与动态、P3b 的目录端口（`issue/app/planning_ports.go`、`planning/app/issue_directory.go`、`bootstrap/ports.go`）。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p4`，分支 `codex/m6-p4`，基线 main `ed8f0df4`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/docs/test/refactor(M6/P4)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 由 T1 改为 `M6/P4`；不延长任何 `until`。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内。
- SQL 只经本模块的 sqlc：planning 的语句不连 `issues`、`cycle_issues`、`states`；issue 的语句不连 `cycles`、`states`。
- 端口由消费方声明；模块不互相 import。
- 不放宽 `TestSQLCSchemaScope`、`rawsql_test`、`archtest`，以及删除、清理、运行角色、权限矩阵覆盖的守卫。
- 删掉不用的代码；不加开关、桩或兼容别名。

**测试纪律**
- 每个 Task 先写会失败的测试，看到它因所测性质失败，再实现到通过。
- 安全、加锁、访客、口径的每个性质至少一个能让测试失败的变异，只计能编译的变异。
- 每个 Task 跑聚焦回归：planning 全部包、P1–P3b 的迭代与模块 bootstrap 测试（按子串匹配 `-run 'Cycle|Module|Planning|PermissionMatrix|Purge|RuntimeRole|Grants|ProjectDeletion'`，`-timeout=40m`；bootstrap 的测试名多为动词在前，如 `TestReadCyclesThroughTheApp`，以 `TestCycle` 开头的正则只匹配到 9 个，T2 发现）、受影响的 issue 包。

**回归测试的边界**
- 回归测试指 M4 的工作项与列表、P1–P3b 的迭代、模块、归属、偏好的测试。
- 只在 spec 说契约变化的地方改：列表行的类型名（`Cycle` → `CycleListItem`、`Module` → `ModuleListItem`）、迁移与列的个数、周期种类清单、`planning.Deps` 的新字段、手搭用例的新依赖、两条网页文案。其他断言不改。

**检查点与证据**
- 检查点：T6 结束时（读的部分全部完成：进度与列表行）串行完整七项，推送并核分支 CI。T14 收尾再串行一次。
- 证据放在 worktree 的 `.superpowers/sdd/P4-progress-transfer/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，调研笔记在 `research/`。实现代理先读同目录的 `implementer-brief.md`。

## Review Focus

1. **口径（R1、R3、R4）**：范围（未删除归属、未删除、未归档、可见）；五组之和即总数；分布不计已取消、`null` 行、多值；燃尽的时区、今天、`scope`、完成日、过时的 `completed_at`、上限与 `null`。
2. **可见范围（R1、R10）**：OwnOnly 只算自己建立的，冻结为 `null`；访客的计数；与列表引擎的总数相同。
3. **冻结（R5、R6）**：只写一次、先写者为准；任务与转移都在项目行锁下；锁后读时钟；不改 `updated_at`；写下后不清（时区回退的边角接受，spec R5）。
4. **转移（R7、R8）**：转谁、留谁；拒绝次序；先冻结再移动；不就地改；每个一条动态与 `MarkChanged`；0 也是 200；锁序。
5. **端口（R2）**：消费方声明；issue 的语句不连 `states`/`cycles`；同一事务经 ctx；列表一次取整页。
6. **回答的形状（R9）**：列表行另立 schema，`Cycle`/`Module` 与写入的回答不变；字段名沿用 Plane。

## Task 1（T1）：P3b 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P3b-list-engine-review.md`：抬头与 §7。
- `specs/P3b-list-engine.md`：抬头与 A8。
- `plans/P3b-list-engine.md`：抬头。
- `M6-design.md`：抬头、§14 的 P3b 行“已完成”、P4 行“进行中”。
- `tools/keywords.json`：顶层 `phase` 改为 `M6/P4`。

- [ ] Step 1：按 P3b 归档（主目录 `.superpowers/sdd/M6-P3b-integration-20261109`）里的台账、CI 收据与 `cleanup.log` 回填 §7：
  - **检查点 CI**：`28f7bbb92b3d3f6837580732289fe0f5635df5bf` 的 42594813667，attempt 1 成功（server (1) 129084768215、web 129084768598、server (0) 129084768608、e2e 129090608431）。
  - **最终门禁**：t14-gate1 在 `4738850c` 上 `platform/postgres` 一包因本机 Docker API 超时失败（环境，未改代码）；修复 Minor 后的 `295a6f07` 上 t14-gate2 七项通过（test 906 秒、66 包，e2e 432）。
  - **最终候选**：`79a540f99e1518ca1746b202aac850ebae1f7fcf` 的分支 CI 42616111421，attempt 1 成功（web 129156280178、server (0) 129156280351、server (1) 129156280411、e2e 129162795112）。
  - **合入与 main CI**：noff `ed8f0df4e8982dc50a4560a31fdf9b2964d84ac7`，双父 `2648a739`、`79a540f9`，树与候选相同（`a18207d1`）；main CI 42621199100 attempt 1 成功（web 129173481052、server (1) 129173481263、server (0) 129173481411、e2e 129181980131）。
  - **归档**：2403 个文件、402,178,603 字节、未知 0、清单 sha256 `5fc24b37fd80dcc9db74d34f3d6f4ae432e94ebb8e710b767016353346699166`；P4 开工时复核 `archive.py verify`（收据 `t1-p3b-archive-verify`）。
  - **清理**：`cleanup.log` 记录分支已含于 origin/main，worktree `m6-p3b`、本地与远端分支 `codex/m6-p3b` 已删除，主目录停在 `ed8f0df4`。
  - 勾选 A8；把阶段记录、spec、plan 的抬头改为已完成。
- [ ] Step 2：设计 §14 的 P3b 行改为“已完成”并附证据，P4 行改为“进行中”，链接 P4 的 spec 与 plan；抬头的状态句随之改。`keywords.json` 的 phase 改为 `M6/P4`，跑 `node tools/keywords.mjs`（没有到期的例外）。
- [ ] Step 3：检查文档链接（`doc-links.py`），然后提交 `docs(M6/P4): 回填P3b的集成结果，进入P4阶段`。

## Task 2（T2）：`progress_snapshot` 列与 planning 的存储

**Files：**
- `server/migrations/sql/00048_planning_cycles_progress_snapshot.sql`（spec §2.4）。
- `server/migrations/`：`planning_schema_test.go` 的 `TestCyclesSchema` 加列、CHECK 与索引；`trigram_schema_test.go` 的迁移总数 47→48 与 Down 循环；上行、下行、再上行。
- `server/sqlc.yaml`：planning 条目加 `00048`。
- `planning/adapter/postgres/queries/`：新文件 `progress.sql`：
  - `CycleSnapshot(id)`：整份；
  - `CycleSnapshotCounts(ids)`：列表行用，只取六个数与 `frozen_at`（`jsonb` 取键，不读分布与燃尽）；
  - `FreezeCycle(id, snapshot)`：`WHERE progress_snapshot IS NULL`，回答是否写入；
  - `FreezeDueProjects(utc_today)`：候选项目 id；
  - `CyclesToFreeze(project_id, today)`：本项目未删除、未冻结、`end_date < today` 的迭代（含已归档），按 id。
- `planning/adapter/postgres/`：上述语句的存储方法与适配器测试。

- [ ] Step 1：写迁移与存储的测试（列、CHECK 拒绝非对象、部分索引进 `TestCyclesSchema` 的 `wantIndexes`、只在为空时写入、列表计数的形状、候选与待冻结的边界：已删除、已冻结、结束日等于今天、已归档的迭代、别的项目），看到失败。
- [ ] Step 2：实现，`make gen`，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - `FreezeCycle` 去掉 `progress_snapshot IS NULL`；
  - `CyclesToFreeze` 的 `<` 改为 `<=`；
  - `CyclesToFreeze` 漏掉 `deleted_at IS NULL`。
  
  然后提交。

## Task 3（T3）：进度的纯计算

**Files：**
- `planning/domain/progress.go` 与 `progress_test.go`（spec §2.1、R1、R3、R4）。
- 冻结的 JSON 编解码（字段名同接口），往返测试。

- [ ] Step 1：写表驱动测试，至少覆盖：
  - 五组计数与总数；范围外的状态组回答错误；
  - 分布：多负责人、多标签、没有的进 `null` 行且排最后、已取消不计、只回答总数大于 0 的行、按 id 升序；
  - 燃尽：东八区的完成时刻落在 UTC 的前一天；今天之后为 `null`；开始前完成的从第一天起算；状态为已完成而 `completed_at` 为空的从第一天起算；状态不是已完成而带 `completed_at` 的不算；已取消不在 `scope`；草稿、缺日期、跨度（结束日减开始日）恰为 3660 天（有，3661 个点）与 3661 天（`null`）；一天的迭代；
  - 冻结的编解码往返。
  
  看到失败。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 完成日按 UTC 取日期；
  - `scope` 含已取消；
  - 分布计入已取消；
  - 完成按 `completed_at` 是否为空而不按状态组；
  - `null` 行排在最前；
  - 跨度上限的比较差一。
  
  然后提交。

## Task 4（T4）：两个方向的统计端口

**Files：**
- `planning/app/`：端口 `IssueStats`、`Projects.StateGroups` 与值类型（spec R2）；`Deps` 的新字段。
- `issue/adapter/postgres/queries/`：新文件 `planning_stats.sql`：`CycleItems`、`ModuleItems`（每个工作项的状态 id、`completed_at`、未删除负责人与标签的 id 数组，按工作项 id）、`CycleStateCounts`、`ModuleStateCounts`（按目标与状态 id）、`CycleAssignees`（按迭代去重的负责人 id）；OwnOnly 以可空的 `viewer` 参数（`viewer IS NULL OR i.created_by_id = viewer`）。
- `issue/` 的实现与 `issue.Provide` 交出（只用连接池）。
- `project/`：`PlanningProjects` 加状态的读取（复用 `IssueStates`）。
- `bootstrap/ports.go`：`planningProjects.StateGroups`、`planningIssues` 的转换；`app.go` 接线；手搭 `planning.New` 的测试补字段（`planning_cycles_test.go` 的 `newPlanningRoute`、`planning_preferences_lock_test.go` 的 `cyclePreferencesChange` 等）。
- 假实现随端口扩展：`planning/app/lock_test.go` 的 `lockWorld`、`bootstrap/planning_ports_test.go` 的 `fakePlanningProjects`；`project/adapter/postgres/planning_ports_test.go` 加状态读取的用例（re-60 L6 的先例）。
- 本 Task 较大：可以分两个提交（先 issue 的语句与适配器测试，再端口、项目读取与转换），收据与台账照常。

- [ ] Step 1：写适配器测试（每条语句）：已删除的工作项、已归档的工作项、已删除的归属、别的迭代/模块、别的项目、OwnOnly（自己的、别人的、建立者为空的）、已删除的负责人与标签行、多负责人与多标签、空输入；再写 bootstrap 的转换测试（不经 planning、不转换、不传错误三种失败）。看到失败。
- [ ] Step 2：实现，`make gen`，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - `CycleItems` 漏掉 `i.archived_at IS NULL`；
  - `CycleItems` 漏掉归属的 `deleted_at IS NULL`；
  - `viewer` 条件恒真；
  - 负责人数组不排除已删除的行；
  - `CycleStateCounts` 漏掉 `i.deleted_at IS NULL`；
  - `ModuleItems` 的模块条件恒真。
  
  然后提交。

## Task 5（T5）：迭代与模块的 `/progress`

**Files：**
- `api/modules/planning.yaml`、`api/openapi.yaml`（根契约列出两条新路径）：`getCycleProgress`、`getModuleProgress` 与 spec §2.3 的 schema（`Progress`、`FrozenProgress`、`CycleProgress`、两种分布、燃尽）。
- `planning/app/`：`GetCycleProgress`、`GetModuleProgress`（读之路：一个快照里读迭代/模块、状态组、工作项事实、冻结；OwnOnly 用自己的范围、冻结为 `null`；今天按项目时区）。
- `planning/adapter/http/`：两个处理器与映射；apitest 每个（操作，码）。
- `bootstrap/`：HTTP 组合测试；权限矩阵的两行（`permission_matrix_cycles_test.go`、`permission_matrix_modules_test.go`，`onACycle` 与模块的同类，读的格子），覆盖守卫不留红。

- [ ] Step 1：写测试：
  - 用例单测：读之路的拒绝（404、403），OwnOnly 的范围与冻结为 `null`，非 OwnOnly 回答冻结；
  - bootstrap：成员、访客、OwnOnly 的计数；已归档的项目与迭代、开关关闭照常读；与列表引擎 `cycle_id=<id>&sub_issues=true&archived=false` 的总数相同（同一调用者，spec R1）；已删除的迭代/模块 404；模块缺日期时燃尽为 `null`。
  
  看到失败。
- [ ] Step 2：实现，`make gen`，网页 `check:types` 通过，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - OwnOnly 时 `viewer` 传空；
  - OwnOnly 时照常回答冻结；
  - 今天按 UTC；
  - 进度的读不在同一个快照（改用写的事务管理器）。
  
  然后提交。

## Task 6（T6）：列表行的计数与检查点

**Files：**
- `api/modules/planning.yaml`：`CycleListItem`、`ModuleListItem`、`FrozenCounts`；`CycleList`、`ModuleList` 的行改用它们（spec R9）。
- `planning/app/list_cycles.go`、`list_modules.go`：整页一次取计数、负责人 id 与冻结的计数（每页固定条数的语句）。
- `planning/adapter/http/`：列表的映射。
- e2e 与 Go 测试里把列表行当 `Cycle`/`Module` 用的地方改类型（断言不改）。

- [ ] Step 1：写测试：两种列表与归档目录的六个数、`assignee_ids`（去重、按 id、只含范围内工作项的）、冻结的计数与时刻、OwnOnly 只算自己的且冻结为 `null`、没有工作项的行全为 0；语句条数不随行数变（以计数的假端口记录调用次数）。看到失败。
- [ ] Step 2：实现，`make gen`，网页 `check:types`、`lint-web` 通过，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 列表逐行调统计端口；
  - OwnOnly 的列表行回答冻结；
  - `assignee_ids` 不去重。
  
  提交。
- [ ] Step 4（检查点）：串行完整七项（`gates.sh t6-gate`），推送准确的 HEAD，核分支 CI。

## Task 7（T7）：冻结的周期任务

**Files：**
- `planning/app/freeze_cycles.go`：`FreezeCycles`（spec §2.5、R6）。
- `planning/adapter/river/freeze_cycles.go`：`planning.freeze_cycles` 周期任务（照 `issue/adapter/river/auto_archive.go`）；`planning/module_jobs.go` 的 `Jobs()`。
- `platform/config/`：`PlanningConfig.FreezeInterval`（`koanf:"freeze_interval"`），校验 ≥1s、日志键、测试；`server/configs/config.yaml` 1h，`config.test.yaml` 2s（dev/prod 的 YAML 没有 `planning:` 一节，继承 `config.yaml`）；`server/configs/embed_test.go:113` 三种环境的期望（test 2s，dev/prod 1h）。
- `bootstrap/`：接线；`app_test.go:95` 的配置（1h）；六处周期种类清单（`all_periodic_jobs_test`、`issue_world_test` 的 `waitForStartupRuns`、`issue_draft_publish_test`、`request_jobs_test`、`purge_scale_fixture_test`、`runtime_role_test`）。

- [ ] Step 1：写测试：
  - 用例单测：候选不加锁；每项目先锁、锁后读时钟；已删除的项目跳过；归档的项目、开关关闭、归档的迭代照样冻结；已冻结的不再写；某项目失败时回答已冻结的个数、之前的已提交；
  - bootstrap：固定时钟（UTC 16:30 即东八区次日）下东八区项目的迭代结束日为 UTC 今天时被冻结、UTC 项目的不冻结；冻结的内容等于当时 `/progress` 的 `live`（非 OwnOnly）；不改 `updated_at`/`updated_by_id`；运行角色下可执行；锁超时保留 PG 错误、时钟未在锁前读；
  - 周期种类清单、配置。
  
  看到失败。
- [ ] Step 2：实现，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 先读时钟再锁项目；
  - 跳过归档的项目；
  - 写入时不带 `progress_snapshot IS NULL`；
  - 冻结按 OwnOnly 的范围算；
  - 某项目失败时整个运行回滚。
  
  然后提交。

## Task 8（T8）：issue 一侧的转移

**Files：**
- `planning/app/`：端口 `IssueTransfer` 与值类型（spec R2）。
- `issue/domain/activity.go`：导出 `CycleMoved(from, to Named)`；测试。
- `issue/adapter/postgres/queries/memberships.sql`：`TransferCycleIssues`（一条语句：按 id 锁源迭代里给定状态 id 的、未删除未归档工作项的未删除归属行，软删，在目标插入新行，同一时刻、同一写者，回答工作项 id 按 id）。
- `issue/app/`：转移的实现：语句 → 每个工作项一条 `CycleMoved` 动态 → `MarkChanged`；不发布事件（spec R8）。经 `issue.Provide` 交出，`bootstrap` 转换。

- [ ] Step 1：写测试：适配器（只转给定状态 id 的；已归档、已删除的工作项与已删除的归属不动；目标插入新行而不是改旧行；时刻与写者；回答的 id 次序）；用例（动态的字段、新旧名字与标识、写者与时刻；`MarkChanged` 的 id；0 个时不写动态）。看到失败。
- [ ] Step 2：实现，`make gen`，聚焦通过（含 P3a 的归属与动态测试）。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 就地改 `cycle_id`；
  - 不过滤状态 id；
  - 漏掉 `archived_at IS NULL`；
  - 动态的新旧名字对调；
  - 不调 `MarkChanged`。
  
  然后提交。

## Task 9（T9）：转移的接口

**Files：**
- `api/modules/planning.yaml`、`api/openapi.yaml`（根契约列出新路径）：`transferCycleIssues`、`CycleTransfer`、`CycleTransferResult`（spec §2.3）。
- `planning/app/transfer_cycle_issues.go`：写之路，拒绝次序（spec §2.3、R7），锁下读状态组与目标，读时钟，先冻结（同 T7 的计算）再经端口移动。
- `planning/domain/errors.go`：转移用的两条服务端消息（同码）。
- `planning/adapter/http/`：处理器；apitest 每个（操作，码）。
- `web/packages/i18n` 的 en 与 zh-CN 两条文案（spec R11）。
- `bootstrap/`：HTTP 组合测试；权限矩阵的转移行：目标迭代与源里的归属由该行的 `prepare` 按格子建立（先例 `unarchiveCycle` 的 `cycleArchived`），不改共用种子（spec §6）。

- [ ] Step 1：写测试：
  - 用例单测：拒绝次序逐项（`lock_test.go` 的 `lockWorld` 记录调用次序）；锁后读时钟；先冻结后移动；已冻结的不再写；0 个时 200；
  - bootstrap：转走积压/未开始/进行中，留下已完成、已取消、已归档；动态与修改时间；回答个数；第二次转移（新加入一个未完成的工作项后）；源未结束、已归档，目标不存在、别的项目、源本身、已删除、已归档、已结束；开关关闭；已归档的项目；OwnOnly 403；冻结的数字是移动之前的。
  
  看到失败。
- [ ] Step 2：实现，`make gen`，`lint-web`、`test-web` 通过，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 不查源是否已结束；
  - 目标为源本身时照转；
  - 先移动后冻结；
  - 目标已结束的判定在读时钟之前（用锁前的时钟）；
  - 转移不经项目写闸门。
  
  然后提交。

## Task 10（T10）：交错与锁

**Files：**
- `bootstrap/` 新测试文件（照 `planning_deletion_interleaving_test.go`、`issue_interleaving_test.go` 的 `inBothOrdersAt`、`requestBlocked`、`heldTx`；周期任务照 `issue_auto_archive_test.go`）。

- [ ] Step 1：写交错测试，各两种次序，都在 `projects` 行上串行：
  - 冻结任务与转移：冻结的数字两种次序相同，只写一次，转走的相同；
  - 转移与工作项 PATCH（把源里一个未完成的工作项改为已完成、或移到别的迭代）；
  - 转移与删目标迭代：先删则 422，先转则被删除的订阅者移出并记动态；
  - 冻结任务与工作项改状态：冻结反映先发生的那个；
  - 转移的锁超时回答 503 `server_busy`。
  
  看到它们在缺锁的变异下失败。
- [ ] Step 2：需要时修正实现，聚焦通过。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 转移不取项目行锁（直接开事务）；
  - 冻结任务不取项目行锁。
  
  然后提交。

## Task 11（T11）：权限矩阵（G17）

**Files：**
- `bootstrap/permission_matrix_cycles_test.go`、`permission_matrix_modules_test.go`：三个新操作的行已在 T5、T9 加入；本 Task 加计数的可见性格子：列表行与进度的数字对访客、OwnOnly（工作区管理员兼访客）只算可见的，冻结对 OwnOnly 为 `null`。归属与冻结由格子的 `prepare` 建立（spec §6），共用种子与已有格子不变。

- [ ] Step 1：写格子，先以一个期望错误的版本看到失败，再改正。
- [ ] Step 2：聚焦通过（`TestPermissionMatrix` 与覆盖守卫）。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - `transferCycleIssues` 的动作改为 `cycle.read`；
  - 进度对 OwnOnly 用全部范围。
  
  然后提交。

## Task 12（T12）：G5、G6、G7、G12 的接口故事

**Files：**
- `e2e/stories/planning/g5-cycle-progress.spec.ts`、`g6-cycle-freeze.spec.ts`、`g7-cycle-transfer.spec.ts`、`g12-module-progress.spec.ts`。
- `e2e/fixtures/` 里需要的调用与断言（spec §2.7）。

- [ ] Step 1：写故事。数据库断言：计数对照 `cycle_issues` 与状态；冻结的列；转移后的归属行与动态（`cycleMoved`、`expectPlanningActivity`）。先跑一个期望错误的版本，看到断言失败，再改正。冻结任务在测试环境 2s 一次，故事以轮询数据库等待，不睡固定时间。
- [ ] Step 2：`make build build-s3-fixture` 后，聚焦跑本目录（`cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test --project=chromium stories/planning/`），通过后提交。

## Task 13（T13）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`（spec R12）：改写 `cycles` 表与迭代接口两行（`progress_snapshot` 回来、进度在 P4），新增行为各行。
- 输入交接的“处理结果（M6/P4）”，status 保持 open：M1-P2（无点数）、M1-P3（无工作区进度）、M1-closeout（口径定义）、M2-closeout 第 1 节（三个端口）、M4-closeout M6-04（燃尽序列的形状）。逐条核对，没有相关的不写。
- 设计：§3.5（燃尽的跨度、完成日、上限；分布只回答总数大于 0 的行，“无”行在有这样的工作项时出现；冻结写下后不清与时区回退的边角；“转移看实时未完成数”的意思）、§5（列表行的 schema 与 `assignee_ids`）、§6（端口的名字与状态组的读取）、§9（配置与测试间隔）、§14 的 P4 行。

- [ ] Step 1：写差异与回执；Plane 的行号对照主目录的只读 `/Users/xiaoruan/project/nerve-project/plane/` 逐一核过（worktree 里没有 `plane/`）。
- [ ] Step 2：检查文档链接与关键词，然后提交。

## Task 14（T14）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（留收据），推送准确的 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账；
  - 写阶段记录 `reviews/P4-progress-transfer-review.md`；
  - 勾选 spec §4；A8 待合入后由 P5 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
