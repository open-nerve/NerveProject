# M6 / P3a：工作项的迭代与模块归属

> 状态：预检通过（2026-11-08，[re-58](../../../reviews/re-58-M6P3a预检.md)），尚未实施。基线 main `ddd79a9f5362aedfc33c940acecc2a5bf0e3d9b8`（P2 的根因修复 noff，main CI 42458454710 通过）。设计见 [M6 设计](../M6-design.md)，负责人 2026-11-07 的决定 A/B/C 见设计 §15；加锁之路与拒绝次序沿用 M4（[硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md)）与 [P1 spec](P1-cycles-backend.md) §3。现状调研见本阶段台账 `research/write-path.md`、`research/list-engine.md`（ignored，随阶段归档）。

## 1. 目标与边界

设计 §12 的 P3 交付边界超出一段约 16 个 Task 的上限（调研：列表引擎的三份手写 SQL、前后端同一份排序夹具、偏好的两套新接口），按设计 §12“超限在实施前拆分”拆为两段：

- **P3a（本 spec）**：`issue` 模块的两张归属表 `cycle_issues`、`module_issues`；工作项与草稿的 `cycle_id`、`module_ids`（建立、PATCH、草稿发布）及回答里的这两个字段；归属的动态；删除迭代/模块时移除归属（`planning.cycle_deleted`、`planning.module_deleted` 两个事件）；删除工作项、项目、工作区时归属的软删与 60 天清理；自动归档跳过仍在未结束迭代或未到目标日模块里的工作项。覆盖设计 §2 的 G4、G11、G14、G16 的接口与数据库证据。
- **P3b（P3a 合入后再写 spec 与 plan）**：列表引擎的筛选 `cycle_id`/`module_id`、分组、排序 `cycle_name`/`module_name`、显示属性 `cycle`/`modules`（含 `api/testdata/issue-order.json` 的前后端同一夹具）、项目偏好的 CHECK；`cycle_user_properties`、`module_user_properties` 与 `/me/cycles|modules/{id}/preferences`、`cycle.preferences`/`module.preferences`。覆盖 G15。

P3a 不做：列表的筛选、分组、排序与显示（P3b，列表行只是多回答两个字段）；进度、冻结、转移（P4）；前端的属性与下拉框（P7）。前端只做五件事：重新生成的类型、手写的 `Issue`/`IssueDetail` 字面量补上两个字段、新错误码在工作项写入上的文案复用（码已有文案，不新增）、迭代动态渲染器的移出分支改读 `old_value`（设计 §8；否则移出迭代的动态显示空名）、草稿编辑保留两个字段（`core/services/issue/draft-input.ts` 的白名单原只有九个键，草稿契约加了两个字段后，网页一编辑就会丢掉经接口存入的迭代与模块；T8 实施时发现，随 T8 修）。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/migrations/sql/00043_issue_cycle_issues.sql`、`00044_issue_module_issues.sql` | 两张表、约束、索引，各含 Down（§2.2） |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql` | issue 条目的 schema 加两个迁移；运行角色对两张表的四种 DML |
| `server/internal/modules/issue/domain/` | `NewIssue`、`IssuePatch`、`Changes` 加迭代与模块（`Changes.None()` 认得它们）；`CheckNewIssue`/`CheckIssuePatch` 的 `module_ids` 上限与重复；归属的动态行（§2.4）；新的字段错误与 409 错误 |
| `server/internal/modules/issue/app/` | `PlanningTargets`、`OpenPlanning` 两个端口（消费方声明，R1、R9）；`IssueCreation.check/write` 与 `UpdateIssue.check/write` 的归属（建立与草稿发布共用 `IssueCreation`）；删除迭代/模块的订阅者（R6）；`issueRowSteps`、`DeletionRowSteps` 与清理步骤加两张表；`AutoArchiveRows` 加未结束的 id |
| `server/internal/modules/issue/adapter/postgres/` | 归属的读写语句；`GetIssue`、`GetIssues`、`SubIssues`、`SearchIssues` 与三条列表语句（平铺、分组、子分组）回答 `cycle_id`、`module_ids`（各条的列须逐一相同，`gen.GetIssueRow`、`gen.FlatIssuesRow` 互转）；软删、删除任务、剩余探查的 CASE 与清理语句；自动归档的排除 |
| `server/internal/modules/issue/adapter/http/` | 建立、PATCH、草稿载荷的解码；`issueDetail`、`issueRow` 的两个字段；每个新声明的错误码一个处理器测试 |
| `server/internal/modules/issue/{module.go,module_events.go}` | `Deps` 加 `Planning`（两个端口）；订阅 `planning.cycle_deleted`、`planning.module_deleted`；清理步骤在 `module.go` 的 `Provide`（`module_purge.go` 只有类型别名） |
| `server/internal/modules/planning/` | `app` 里两个端口的实现（`issue_targets.go`：目标事实、迭代是否已结束、未结束的 id，R1、R9）与批量读取的语句；`Provide(pool, projects)` 回答 `Provided{Purger, IssueTargets}`（`projects` 只需项目事实的读取；五处测试调用随之改：`bootstrap` 的 `purge_catalog_test`、`issue_retention_sets_test`、`asset_image_retention_fixture_test`，`planning/adapter/postgres` 的 `module_rows_test`、`store_test`，清理不用它，可传 nil）；`Deps.Events`，删迭代/模块在原事务发布事件（R6）；`EventTypes()` |
| `server/internal/bootstrap/` | 接线（`planning.Provide` 拿到项目端口；`eventTypes()` 加 planning；`ports.go` 的转换）；组合测试、交错、守卫与夹具（§2.5） |
| `server/migrations/` 测试 | `issue_schema_test` 的约束与索引名；`purge_indexes_test` 的表数；`trigram_schema_test` 的迁移总数、Down 循环起点与表清单 |
| `api/modules/issue.yaml` | `Issue`、`IssueDetail` 加必填 `cycle_id`（uuid 或 null）、`module_ids`（uuid 数组，按 id 升序）；`IssueCreate`、`IssueDraftPayload` 加可省略的 `cycle_id`（uuid，不可为 null）、`module_ids`；`IssueUpdate` 加 `cycle_id`（uuid 或 null）、`module_ids`；建立、修改、发布草稿声明新错误码（§2.3） |
| `web/` | `api-client` 重新生成；手写的 `Issue`/`IssueDetail` 字面量（13 个文件：测试与 `core/store/issue/fake-issues.ts`）补 `cycle_id: null, module_ids: []`；`issue-activity/activity/actions/cycle.tsx` 的移出分支读 `old_value` |
| `e2e/` | `furnishPlanningRows` 回答它建的迭代与模块的 id，`furnishIssueRows` 接收它们并经接口把工作项加入（调用者的令牌须是项目成员）；各调用处（`furnishWorkspace` 与 P4、W3、I21 等删除故事）改为先 `furnishPlanningRows` 后 `furnishIssueRows`；`assert/catalog.ts` 的 `asynchronousDeletionTable` 加两张表；`stories/planning/` 的 G4、G11、G14 接口与数据库故事 |
| `tools/keywords.json` | `phase` 为 `M6/P3`（P3a 与 P3b 共用，先例 M5 的 P7a/P7b 用 `M5/P7`：关键词工具只认 `M<n>/P<k>`） |
| `docs/` | P2 阶段记录 §7、P2 spec A9、设计 §12 与 §14（拆分与 P2 完成）的回填（T1）；`plane-diff.md`；输入交接的“处理结果（M6/P3a）” |

### 2.2 表与 Plane 差异

两张表照 `issue_labels`（`00017`）的写法：

| 表 | 列 | 唯一与索引 | 与 Plane |
|---|---|---|---|
| `cycle_issues` | `id`、`workspace_id`、`project_id`、`cycle_id`（`REFERENCES cycles ON DELETE CASCADE`）、`issue_id`（`REFERENCES issues ON DELETE CASCADE`）、审计四列、`deleted_at` | `cycle_issues_issue_id_key (issue_id) WHERE deleted_at IS NULL`（一个工作项最多在一个迭代，总体设计 §5.3）；四条 CASCADE 外键各一个无条件索引；`cycle_issues_deleted_at_idx` | Plane 只有部分唯一 `(cycle, issue)`，“一个迭代”只在代码里 |
| `module_issues` | 同上，`module_id REFERENCES modules ON DELETE CASCADE` | `module_issues_issue_id_module_id_key (issue_id, module_id) WHERE deleted_at IS NULL`；四个无条件索引；`module_issues_deleted_at_idx` | 同 Plane |

外键指向 `planning` 的表在 `TestSQLCSchemaScope` 下允许（先例 `issue_labels.label_id REFERENCES labels`）；`issue` 的语句不连 `cycles`、`modules`，名字与事实一律经端口或事件。

### 2.3 接口与错误

- **回答**：`Issue`、`IssueDetail` 的 `cycle_id` 是工作项当前未删除的归属所在迭代（已归档的迭代照常回答），没有时 `null`；`module_ids` 是未删除归属的模块 id，按 id 升序，没有时 `[]`。列表行、关系、子工作项、搜索里的 `Issue` 同样带上（列与 `GetIssue` 相同）。
- **建立与草稿发布**：`cycle_id` 省略即不加入；`module_ids` 省略或 `[]` 即不加入；`module_ids` 至多 100 个、不重复（同 `label_ids`，`checkIDs`）。
- **PATCH**：`cycle_id` 给 uuid 即加入或移到该迭代，给 `null` 即移出；`module_ids` 整体替换（移出的软删，新加入的插入，留下的不动）。与现值相同算没有改动。
- **草稿保存**：`IssueDraftPayload` 加同样的两个字段，只做结构检查（同其他九个字段）；类型与 `IssueCreate` 一致，使发布与建立的 400/422 逐值相同（`TestDraftPublishMatchesCreateBodyErrors`）。
- **新声明的错误**（建立、PATCH、发布草稿三处）：409 `planning.cycles_disabled`、`planning.modules_disabled`、`planning.archived`、`planning.cycle_completed`；422 字段 `cycle_id`、`module_ids[i]` 的 `not_allowed`（同 `label_ids[i]`）。`issue/domain/errors.go` 自己声明这四个码（跨模块重复声明的先例：`project.archived`）。`apitest.Main` 按（操作，码）核对，每个声明了某码的操作各要一个回答它的处理器测试：建立与 PATCH 各 4 个、发布草稿 4 个（`project.archived` 今天也是每个操作一个）。

### 2.4 动态

照 Plane 的字段名写进 `issue_activities`（设计 §3.3），前端现有的渲染器读它们（迭代渲染器的移出分支随本阶段改读 `old_value`，见 §1）：

| 变化 | 行 |
|---|---|
| 加入迭代 | `created`，`cycles`，`""` → 迭代名，`new_identifier` 为迭代 id，comment `added cycle ` |
| 移到别的迭代 | `updated`，`cycles`，旧名 → 新名，两个标识 |
| 移出迭代、迭代被删 | `deleted`，`cycles`，名字 → `""`，`old_identifier` 为迭代 id，comment `removed this work item from cycle ` |
| 加入模块 | `created`，`modules`，`""` → 模块名，`new_identifier` 为模块 id |
| 移出模块、模块被删 | `deleted`，`modules`，名字 → `""`，`old_identifier` 为模块 id |

PATCH 的行接在既有行（名字…负责人移出）之后：迭代一行，模块加入（按 id 升序）、模块移出（按 id 升序）。建立与草稿发布写创建行与负责人行之后，再写迭代的“加入”与每个模块的“加入”（R5）。`issue.updated` 的 `Changes` 由 `UpdateActivity` 投影，自动带上；`issue.created` 的 `Changes` 照标签的写法加 `cycles`、`modules` 两类。重复加入（已在其中）不写。

### 2.5 删除与清理

- **删迭代/模块**（R6）：`planning` 在删除的事务里发布 `planning.cycle_deleted` / `planning.module_deleted`（`Entity{cycle|module, id}`、工作区、项目、写者、删除时刻，`Changes` 一条 `{Field:"name", Old: 名字}`）；`issue` 的订阅者按 id 升序软删该目标未删除的归属（同一时刻、同一写者），为每个受影响的工作项写一条 `deleted` 动态，并把这些工作项记为被这次删除最后修改（`MarkChanged`，先例：附件与关系的变化）。不发 `issue.updated`（先例同上）。
- **删工作项**：`issueRowSteps` 在 `issue_labels` 之后加 `cycle_issues`、`module_issues`，随子树同一时刻软删，不写归属动态。
- **删项目、工作区**：`issue.delete_rows` 的 `DeletionRowSteps` 在 `issue_labels` 之后加两步，剩余探查加两个 CASE 分支。
- **60 天清理**：`issue` 的清理步骤在 `issues` 之前加 `module_issues`、`cycle_issues`（子先父；模块次序 asset → issue → planning 已使它们先于 `cycles`、`modules`）。
- **守卫与夹具**：`purge_catalog_test` 与 `purge_indexes_test` 30 张表；`trigram_schema_test` 44 个迁移；`issueNames`；运行角色；`projectIssueTable`（bootstrap）与 `asynchronousDeletionTable`（e2e）认得两张表（它们不以 `issue_` 开头，不改就会被当作同步删除）。`keysTo` 由目录推导，建表后这些守卫即要求两张表有行，而归属的接口要到 T6/T7 才有，所以 Go 的夹具一律以 SQL 种归属（夹具允许），在迭代与模块的种子之后：`seedIssues`（`twoProjects`、`seedProject` 已先跑 `seedCycles`/`seedModules`，它们不回答 id，`seedIssues` 按项目查出迭代与模块）覆盖项目、工作区、迟到删除与清理的守卫；`deletedWorkItem` 与 `issue_deletion_test` 里保留的工作项、`furnishDeletionIssue`（三个项目，其中两个在测试里建立）在工作项所在项目以 SQL 种一个迭代、一个模块与归属；`seedPurgeSparseRows` 与 `purgeScaleTables` 加两张表；`purge_runtime_test` 的 issue 组若加两张表，老化与核对的表清单同步加。P3a 以 `NERVE_PURGE_SCALE=1` 跑一次 `TestPurgeScale`。

## 3. 局部裁定

- **R1 目标的事实经 planning 的端口**：`issue/app` 声明 `PlanningTargets`：`Targets(ctx, projectID, cycles, modules []uuid.UUID)` 在工作项写已取得的项目行锁下回答项目的两个开关，以及所给 id 中属于本项目、未删除的迭代与模块（含已归档）的名字与是否归档；`CycleCompleted(ctx, projectID, cycleID, now)` 回答该迭代按项目时区在 `now` 是否已结束。实现在 `planning/app`（项目时区与开关经 planning 已有的项目端口，状态用 `domain.Today`/`StatusOf`，口径只此一处），由 `planning.Provide(pool, projects)` 交出，`bootstrap/ports.go` 转换。不扩展 `issue` 的 `LockedProject`：开关与“已结束”是 planning 的规则。
- **R2 拒绝的次序**（建立、PATCH、草稿发布三条路相同）：结构 400 → 不读数据的 422（`module_ids` 上限与重复）→ 404 → 403 → 409 `project.archived` →（PATCH）409 `issue.archived` → 无改动 200 → 409 `planning.cycles_disabled`（`cycle_id` 有变化）、`planning.modules_disabled`（`module_ids` 有变化）→ 409 `planning.archived`（新加入的目标已归档）→ 锁下的 422（既有各项与 `cycle_id`/`module_ids[i]` `not_allowed`：不存在、已删除或不在本项目）→ 资产的 404（`LockContent`）→ 读时钟 → 409 `planning.cycle_completed`（新加入的迭代已结束）→ 写。“已结束”要用这次写的时钟，所以在最后一把锁与读时钟之后判定（P1 改迭代时同样在锁下读时钟后判定状态，`update_cycle.go:54-59`）；建立只读一次时钟由 `issue_event_writes_test.go:120` 固定，不能为此再读一次。三条路的时钟都来自资产锁之后（`LockContent`、`LockDraftAssets`），判定点只有两处：`IssueCreation.writePreparedAt` 的开头（建立与草稿发布共用）与 `UpdateIssue.write` 的开头；PATCH 在检查与时钟之间的 `prepareDescription` 不产生回答给用户的错误。其余目标事实在时钟之前一次读出，未知目标与其他字段的问题合在一个 422 里。
- **R3 只检查新加入的**：留下的与移出的目标不检查（含已归档、已结束的迭代，已归档的模块），但仍经 `Targets` 读出名字，供“移到”“移出”的动态（同标签读加入与移出的两组，`update_issue.go:203`）；移到别的迭代只检查新迭代；开关看的是“有变化”，移出也受开关约束（设计 §3.8）。
- **R4 写法**：迭代变化时软删旧行、插入新行（同一时刻、同一写者，不就地改 `cycle_id`）；模块照标签的 `relink`。按 `(issue_id)` 与 `(issue_id, module_id)` 的部分唯一，同一工作项的并发写已由项目行锁串行，唯一冲突只可能是缺陷，不映射为业务错误。
- **R5 建立时的动态**：建立与草稿发布写迭代与模块的“加入”行。Plane 的建立弹窗先建工作项、再分别调加入迭代与模块的接口，历史里因此有这两类行；Nerve 一次写入，保留同样的历史（M4 对建立时的标签不写行，依据也是 Plane，两者不矛盾）。
- **R6 删除的事件**：`planning` 自己发布、`issue` 订阅（设计 §4.3、§6），名字随事件给出（`issue` 不能读 `cycles`/`modules`）。`planning.Deps` 加 `Events`；P1/P2 的删除用例测试固定了调用次序，加上“发布”一步，断言随之改这一处。订阅者要求已有事务（先例 `issue/adapter/postgres/attachment_events.go:12-27`），严格解析载荷（先例 `issue/app/delete_rows_on_deletion.go` 的 `deletedProjects`、`attachment_events.go` 的 `attachmentIssue`）。
- **R7 OwnOnly**：归属是工作项的写，照 `issue.update` 的规则（只看自己的调用者只能改自己建立的工作项，`HiddenFrom` 的 404）；不套用 `planning` 对一切写 403 的规则（那是迭代/模块本身的写）。
- **R8 开关默认关**：Go 组合测试与 e2e 先经接口打开 `cycle_view`、`module_view`；删除、清理、运行角色与规模的夹具以 SQL 种行。
- **R9 自动归档**：`issue/app` 声明 `OpenPlanning.OpenTargets(ctx, projectID, now)`，回答本项目未删除、结束日为空或不早于今天（项目时区）的迭代，与未删除、目标日为空或不早于今天的模块（不论归档与状态，Plane 同，设计 §7）。在 `Now` 之后、`AutoArchive` 之前调用（同一事务、同一业务时钟），id 数组进 `AutoArchiveRows`，语句以 `NOT EXISTS` 排除仍有这些归属的工作项。`auto_archive_test` 固定的调用次序加这一步。
- **R10 Plane 差异**：不照搬重复加入时重复写动态；加入已结束迭代 Plane 回答 400，Nerve 409 `planning.cycle_completed`；Plane 的 `module_ids` 不含已归档的模块，Nerve 含（设计 §3.6）。删迭代/模块时每个工作项一条“移出”动态并更新它的修改时间，与 Plane 相同（`bgtasks/issue_activities_task.py:822-858`；原写“Plane 只写一条迭代动态”有误，T12 核对 Plane 源码后更正，`plane-diff.md` 的 P1“删除迭代”一行同步更正）。登记进 `plane-diff.md`。

## 4. 验收标准

- [ ] A1 G4：建立、PATCH、草稿发布加入迭代；移到别的迭代、移出；已结束、已归档、已删除、别的项目的迭代被拒（码与字段如 §2.3）；开关关闭时有变化即 409；重复加入 200 不写；每次变化一条动态，`issue.updated`/`issue.created` 的 `Changes` 带上。
- [ ] A2 G11：模块的加入、移出、整体替换；已归档、已删除、别的项目的模块被拒；留下的不检查；上限与重复；开关；动态。
- [ ] A3 G14：删迭代/模块后工作项不受影响、归属移除，每个受影响的工作项一条 `deleted` 动态并记为被删除者此刻修改；删工作项、项目、工作区后归属以原 At/By 软删并可续跑；60 天清理有界、运行角色可执行、`TestPurgeScale` 认得新语句；全部守卫覆盖两张表。
- [ ] A4 G16：自动归档跳过仍在未结束迭代（含草稿迭代）或未到目标日模块（含目标日为空）里的工作项；“今天”按项目时区与任务的固定时钟；调用次序与锁不变。
- [ ] A5 加锁与交错：归属的检查与写在项目行锁下；“写归属”与“删迭代/删模块”两种次序的交错各有测试（先删则 422，先写则被订阅者移除并记动态）；“已结束”用这次写的时钟。
- [ ] A6 回答：`Issue`/`IssueDetail` 的两个字段在详情、列表行、关系、子工作项、搜索里一致；访客与 OwnOnly 读到的同样；前端类型与字面量更新后 `make lint-web`、`make test-web` 通过。
- [ ] A7 phase 为 `M6/P3`；每个新声明的码在 issue 的 apitest 二进制里有回答；输入交接写“处理结果（M6/P3a）”；P2 阶段记录 §7、spec A9 与设计 §12、§14 已回填。
- [ ] A8 七项本地门禁、deep-review 修复闭环、准确分支 CI 与 noff 后 main CI 全绿、归档清理。

## 5. 后续责任

P3b：列表引擎与两种偏好（G15），`PlanningDirectory` 端口只服务名字排序（调研：分组头由前端按自己的目录给出、筛选值只做语法检查）；P4：进度、冻结、转移（转移经 `issue` 的端口在同一事务移动归属并写动态）；P7：前端的属性、下拉框与新建弹窗；P8：规模测试重跑。

## 6. 风险

- **三条列表语句与四个读取投影**：`cycle_id`、`module_ids` 两列要在七处语句里逐一相同，一处不一致即编译失败（行类型互转），以 sqlc 生成与 `issue` 的适配器测试为守卫，单独一个 Task、先于任何写入。
- **前端类型**：`Issue` 加必填字段使手写字面量编译失败，在同一 Task 内修完并跑 `make lint-web`、`make test-web`。
- **事件把 planning 的删除与 issue 的写连在一个事务**：订阅者失败使删迭代回滚；订阅者只做软删、插动态与 `MarkChanged` 三条语句，按 id 升序，次序为项目行（planning 已持有）→ 迭代/模块行（它的 UPDATE）→ 归属行 → 动态 → 工作项行（`MarkChanged`）。工作项的写不显式锁工作项行（行锁来自 `write` 里的 UPDATE），但每个写一个项目的工作项或归属的写者都先取项目行 `FOR NO KEY UPDATE`（issue 与 planning 的 `Locks`、附件写的 `LockIssueProject`、自动归档的 `Projects.Lock`），所以它们在这把锁下串行，不会交叉等待。
- **按目录驱动的守卫**：两张表在建表的 Task 触发多道守卫；Go 的删除守卫在紧接的删除 Task 结束时转绿；e2e 的项目/工作区删除故事要等工作项能经接口加入迭代与模块（`furnishIssueRows`）之后才转绿，所以第一个整段七项检查点在归属的接口落地之后（plan 写明时点）。
