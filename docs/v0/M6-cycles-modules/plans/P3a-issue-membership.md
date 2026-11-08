# M6 / P3a 工作项的迭代与模块归属实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：T1–T13 已实施（2026-11-08），整分支审查 [re-59](../../../reviews/re-59-M6P3a整体审核.md)，阶段记录 [P3a 阶段记录](../reviews/P3a-issue-membership-review.md)；预检 [re-58](../../../reviews/re-58-M6P3a预检.md)。

**Goal：** 工作项经自己的字段加入、移动、移出迭代与模块（建立、PATCH、草稿发布），每次变化一条动态；删除迭代/模块、工作项、项目、工作区时归属随之移除与清理；自动归档跳过仍在未结束迭代或未到目标日模块里的工作项。

**Architecture：** `issue` 拥有 `cycle_issues`、`module_issues`；目标的事实（开关、名字、归档、是否已结束、未结束的 id）经 `issue/app` 声明、`planning/app` 实现的端口，在工作项写已取得的项目行锁下读取；删迭代/模块由 `planning` 在原事务发布事件，`issue` 订阅后移除归属并写动态；`bootstrap` 唯一接线。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/River/Playwright，不加依赖。

**Spec：** [P3a-issue-membership](../specs/P3a-issue-membership.md)（§2.5 的守卫清单与 §3 的 R1–R10 是本计划的约束）；设计 [M6](../M6-design.md) §3.3、§3.8、§4、§6、§7；M4 的工作项写（`issue/app/create_core.go`、`update_issue.go`）与标签、负责人的写法是模板。

## Global Constraints

- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p3a`，分支 `codex/m6-p3a`，基线 main `ddd79a9f`；主目录永远 main。提交详细中文，前缀 `feat/fix/docs/test/refactor(M6/P3a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`，推送用 `/usr/bin/git`；noff 已获连续授权。
- `plane/`、`refer/` 只读且不在其内跑 git；不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。
- T1 改 phase 为 `M6/P3`（P3a 与 P3b 共用，先例 `M5/P7`；关键词工具只认 `M<n>/P<k>`）；不延长任何 `until`。
- 手写文件约 400 行内；SQL 只经本模块 sqlc（`issue` 的语句不连 `cycles`、`modules`）；端口由消费方声明；模块不互相 import；不放宽 `TestSQLCSchemaScope`、`rawsql_test`、`archtest`、删除/清理/运行角色守卫。
- 每个（操作，新声明的码）在同一 Task 有 issue 的 apitest 处理器测试回答（`apitest.Main` 按操作核对）；码已有中英文案，不新增；`make test-web` 照跑。
- 测试夹具：`cycle_view`、`module_view` 默认关，Go 组合测试与 e2e 先经接口打开；删除、清理、运行角色与规模的夹具以 SQL 种行（夹具允许，含 T3 起各删除守卫要求的归属行，归属的接口要到 T6/T7 才有）；故事的前置数据只经接口。
- 守卫转绿的时点：表级守卫在 T2 内转绿；Go 的删除与清理守卫在 T3 结束时转绿；e2e 的项目/工作区删除故事要求两张表各有一行随项目软删，等 T9 的 `furnishIssueRows` 能经接口加入迭代与模块才转绿。因此 T9 之前不做整段七项检查点（只跑聚焦回归，`make test` 的 bootstrap 包用 `make test-shard` 两路或聚焦正则），T9 起有产品变化的检查点推送前串行完整七项。
- 每个 Task：先写会失败的测试并看到它因所测性质失败，再实现到通过；安全、加锁、访客性质至少一个能让测试失败的变异（只计能编译的变异）。每 Task 跑聚焦回归。
- M4 的工作项测试与 P1/P2 的迭代、模块测试是回归：只改它们的夹具、假实现与固定的调用次序（spec R6、R9 点名的两处），不改其他断言。
- 证据放 worktree 的 `.superpowers/sdd/P3a-issue-membership/`（台账 `progress.md`、收据经 `run-check.py`、变异经 `mutate.py`；调研笔记在 `research/`）。

## Review Focus

1. 拒绝次序（spec R2）：开关与归档的 409 在锁下 422 之前；未知目标与其他字段问题合在一个 422；“已结束”在最后一把锁与读时钟之后、用同一个时钟。
2. 只检查新加入的（R3）；移到别的迭代只检查新迭代；开关看“有变化”。
3. 端口（R1、R9）：消费方声明、planning 实现、bootstrap 转换；“已结束”与“未结束”的口径只在 planning；读取在工作项写的事务与项目行锁下。
4. 动态（spec §2.4）：字段名、动词、标识、次序；重复加入不写；`issue.created`/`issue.updated` 的 `Changes`。
5. 删除：事件在原事务、订阅者按 id 升序软删并写动态与 `MarkChanged`；删工作项、项目、工作区与清理覆盖两张表；守卫没有被放宽。
6. 回答的两个字段在七处投影里一致；访客与 OwnOnly 读到的同样。

## Task 1（T1）：P2 回填、拆分与阶段

**Files：** `docs/v0/M6-cycles-modules/reviews/P2-modules-backend-review.md`（抬头与 §7）；`specs/P2-modules-backend.md`（A9）；`M6-design.md`（抬头、§12 的 P3 拆为 P3a/P3b 两行、§14 的 P2 行“已完成”与 P3a 行“进行中”）；`tools/keywords.json`。

- [ ] Step 1：按 P2 归档（`.superpowers/sdd/M6-P2-integration-20261108`）的台账与 CI 收据回填 §7：分支 CI（含失败与修复的去向）、noff `861ea1b5`、main CI 42399566014 两次失败与负责人的指示、修复分支 `codex/m6-p2-ci` 的提交与分支 CI、第二次 noff `ddd79a9f`、main CI 42458454710、归档（2707 个文件、未知 0、清单 sha256 `a7ba04f0…`，复核 `archive.py verify`）、清理；勾选 A9；抬头改为已完成。
- [ ] Step 2：设计 §12 把 P3 拆为 P3a（本 plan）与 P3b（列表引擎与偏好，P3a 合入后写 spec 与 plan），§14 加两行；phase 改 `M6/P3`，`node tools/keywords.mjs` 通过；文档链接检查；提交 `docs(M6/P3a): 回填P2的集成结果，P3拆为P3a与P3b并进入P3a阶段`。

## Task 2（T2）：两张归属表与表级守卫

**Files：** `server/migrations/sql/00043_issue_cycle_issues.sql`、`00044_issue_module_issues.sql`；`server/sqlc.yaml`；`deploy/runtime-grants.sql`；`server/migrations/{issue_schema_test.go,purge_indexes_test.go,trigram_schema_test.go}`（迁移总数两处、Down 循环起点与表清单）；`bootstrap/purge_catalog_test.go`（表数）。

- [ ] Step 1：先改守卫（RED）：约束与索引名并入 `issueNames`、表数 28→30、迁移总数 42→44；新的约束测试（`cycle_issues` 一个工作项只在一个未删除的迭代、已删除的不占；`module_issues` 同一对只一行未删除的；外键去向：删迭代/模块/工作项/项目/工作区 CASCADE，审计列 SET NULL）。
- [ ] Step 2：两个迁移（含 Down 与 spec §2.2 的全部约束、索引）、sqlc schema、运行角色授权；`TestMigrationsGoUpDownAndUpAgain`、`TestConstraintAndIndexNames`、`TestPurgeIndexes`、运行角色授权覆盖通过。`purge_catalog_test` 的“没有清理步骤”与删除守卫在 T3 转绿（台账记基线）。
- [ ] Step 3：变异：`cycle_issues` 的唯一改为 `(cycle_id, issue_id)`、`module_issues` 的唯一去掉 `WHERE deleted_at IS NULL`、`issue_id` 改 `SET NULL` 时测试失败。提交。

## Task 3（T3）：删工作项、删除任务与清理的两步，删除守卫

**Files：** `issue/app/{delete_issue.go,delete_rows.go,delete_project_rows.go}` 与测试；`issue/adapter/postgres/queries/{delete_issue.sql,delete_rows.sql,deletion_remaining.sql,purge.sql}`、`Store` 与测试（`deletion_remaining_test`、`delete_rows_test`、`delete_issue_test`、`purge_test` 的固定清单）；`issue/module.go` 的 `Provide` 里的清理步骤；`bootstrap` 的 `issue_project_deletion_jobs_test.go`（`projectIssueTable` 认两张表）、种子 `seedIssues`（以 SQL 给 Web/Ops 的工作项各一个迭代与模块归属：`twoProjects`、`seedProject` 已先跑 `seedCycles`/`seedModules`，它们不回答 id，按项目查出）、`project_deletion_test.go` 与 `issue_deletion_test.go`（`deletedWorkItem` 与保留的工作项在其项目以 SQL 种迭代、模块与归属）、`issue_deletion_interleaving_test.go`（`furnishDeletionIssue`，三个项目各种一份）、`purge_scale_fixture_test.go`（`purgeScaleTables`、`seedPurgeSparseRows`）、`purge_runtime_test.go`（issue 组与老化、核对的表清单同步加两张表）。

- [ ] Step 1：守卫与单元测试（RED）：删工作项连带两张表同一时刻；`DeletionRowSteps` 的步骤清单与剩余探查；清理步骤的次序；项目/工作区删除、迟到删除与清理的守卫随种子扩展而要求两张表。
- [ ] Step 2：实现（`issueRowSteps` 与 `DeletionRowSteps` 在 `issue_labels` 之后加 `cycle_issues`、`module_issues`；清理在 `issues` 之前加 `module_issues`、`cycle_issues`）；bootstrap 的删除、清理、运行角色守卫转绿（聚焦正则，收据）。
- [ ] Step 3：本地跑一次 `NERVE_PURGE_SCALE=1` 的 `TestPurgeScale`（收据）；变异：删工作项漏掉 `cycle_issues`、删除任务漏掉 `module_issues` 步骤、清理把 `issues` 放到两张表之前时测试失败。提交。

## Task 4（T4）：回答里的 `cycle_id` 与 `module_ids`

**Files：** `api/modules/issue.yaml`（`Issue`、`IssueDetail` 两个必填字段）、生成物（`gen`、`dist`、`schema.gen.ts`）；`issue/domain/issue.go`（`Issue.CycleID`、`ModuleIDs`）；`issue/adapter/postgres/queries/{issues.sql,sub_issues.sql,search_issues.sql,list_issues.sql,list_issue_groups.sql,list_issue_subgroups.sql}` 与映射（`issueOf`、`listRecord`、`search_issues.go`）；`issue/adapter/http/issues.go`（`issueDetail`、`issueRow`）；`web/` 中手写的 `Issue`/`IssueDetail` 字面量（以 `make lint-web` 的类型检查找全）。

- [ ] Step 1：测试（RED）：适配器测试以 SQL 种归属（含已删除的归属不回答、已归档的迭代照常回答、`module_ids` 升序）；详情、列表三种形态、关系、子工作项、搜索各回答两个字段且一致；访客与 OwnOnly 读到的一样。
- [ ] Step 2：实现（七处投影的列逐一相同）；`make gen`；前端字面量补字段；`make lint-go gen-check lint-web test-web` 与 issue 适配器、bootstrap 的工作项读取测试通过。
- [ ] Step 3：变异：`cycle_id` 不排除已删除的归属、`module_ids` 不排序时测试失败。提交。

## Task 5（T5）：planning 的目标端口与未结束的 id

**Files：** `issue/app/planning_ports.go`（`PlanningTargets`、`OpenPlanning` 与值类型，spec R1、R9）；`planning/app/issue_targets.go` 与测试；`planning/adapter/postgres/queries/issue_targets.sql`（按 id 批量读本项目未删除的迭代与模块、未结束的 id）与 `Store`；`planning/module.go`（`Provide(pool, projects)` 回答 `IssueTargets`，`projects` 只需项目事实的读取）；`bootstrap/{app.go,ports.go}` 的转换与接线（`issue.Deps.Planning`）；`planning.Provide` 的五处测试调用（`bootstrap` 的 `purge_catalog_test.go`、`issue_retention_sets_test.go`、`asset_image_retention_fixture_test.go`，`planning/adapter/postgres` 的 `module_rows_test.go`、`store_test.go`；清理不用项目端口，可传 nil）。

- [ ] Step 1：测试（RED）：`Targets` 只回答本项目、未删除的（含已归档）并带名字与归档；别的项目、已删除的不回答；两个开关；`CycleCompleted` 按项目时区（以上海时区跨日的用例区分 UTC）、草稿与未开始与进行中为否；`OpenTargets` 的迭代（结束日为空或不早于今天）与模块（目标日为空或不早于今天，不论归档与状态），已删除的不在内。
- [ ] Step 2：实现；`bootstrap` 接线（此 Task 端口尚无调用者，下一 Task 使用）。
- [ ] Step 3：变异：`CycleCompleted` 用 UTC 的今天、`Targets` 不排除别的项目、`OpenTargets` 把今天结束的迭代算作已结束时测试失败。提交。

## Task 6（T6）：工作项加入、移动、移出迭代

**Files：** `api/modules/issue.yaml`（`IssueCreate`、`IssueUpdate` 的 `cycle_id`；建立与 PATCH 声明 `planning.cycles_disabled`、`planning.archived`、`planning.cycle_completed`）、生成物；`issue/domain/{issue.go,patch.go,activity.go,errors.go}` 与测试；`issue/app/{create_core.go,update_issue.go,writes.go,issue_events.go}` 与测试（假实现加端口）；`issue/adapter/postgres/queries/memberships.sql`（迭代的软删与插入）与 `Store`；`issue/adapter/http/{create_payload.go,issues.go}` 与处理器测试（三个码 × 建立、PATCH 各一个）；bootstrap 的组合测试（新文件 `issue_cycle_membership_test.go`）；`web/apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/cycle.tsx` 的移出分支读 `old_value`（设计 §8）与它的单元测试。

- [ ] Step 1：测试（RED）：建立时加入；PATCH 加入、移到别的迭代、移出（含移出已归档、已结束的迭代）；相同值不写；已结束（用这次写的时钟，以项目时区跨日区分）、已归档 409；不存在、已删除、别的项目 422 `cycle_id not_allowed` 并与其他字段问题合在一个 422；`cycle_view` 关闭时有变化 409、无变化 200；拒绝次序（spec R2）的相邻对各一个用例；动态的三种行与 `issue.created`/`issue.updated` 的 `Changes`；OwnOnly 只能改自己建立的（R7）；归属行与工作项的审计同一时刻；竞态：同一工作项两个并发 PATCH 指向不同迭代，最后只有一行未删除的 `cycle_issues`、两条动态（`racingWrite`/`w.sending`，`planning_cycles_test.go:192-196` 的写法）。
- [ ] Step 2：实现（R1–R5）；`make gen`；聚焦回归（issue 全部包、bootstrap 的工作项写与事件测试、`make lint-go gen-check test-web`）。
- [ ] Step 3：变异：“已结束”在读时钟之前判定（以跨日交错证明）、开关只在加入时看、移到别的迭代也检查旧迭代、重复加入仍写动态时测试失败。提交。

## Task 7（T7）：工作项加入、移出模块

**Files：** `api/modules/issue.yaml`（`module_ids`；建立与 PATCH 声明 `planning.modules_disabled`）、生成物；`issue/domain` 的 `module_ids` 上限、重复与 `Changes`；`issue/app` 的检查与 `relink`；`memberships.sql` 的模块语句；处理器测试（新码 × 建立、PATCH 各一个）；bootstrap 组合测试（`issue_module_membership_test.go`）。

- [ ] Step 1：测试（RED）：建立时加入多个；PATCH 整体替换（新加入的检查、留下的已归档模块不检查、移出的软删且同一时刻）；集合相同不写；上限与重复（下标）；已归档 409；不存在、已删除、别的项目 422 `module_ids[i] not_allowed`（下标指向请求里的位置）；开关；动态的两种行（按 id 升序）。
- [ ] Step 2：实现；聚焦回归。
- [ ] Step 3：变异：留下的模块也检查、移出的物理删除、下标指向排序后的位置时测试失败。提交。

## Task 8（T8）：草稿的迭代与模块

**Files：** `api/modules/issue.yaml`（`IssueDraftPayload` 两个字段；`publishDraft` 声明四个新码）、生成物；处理器测试（四个码 × 发布草稿各一个）；bootstrap 的草稿测试（`issue_draft_publish_test.go` 等）。

- [ ] Step 1：测试（RED）：保存只做结构检查（指向已结束、已删除的迭代也能保存）；发布按建立的规则（已结束、已归档、已删除、开关关闭被拒、草稿留着可改）；发布成功写归属与“加入”动态；`TestDraftPublishMatchesCreateBodyErrors` 覆盖两个字段的 400/422 用例。
- [ ] Step 2：实现（解码随 `newIssueInput` 共用，检查与写入随 `IssueCreation` 共用，不另写）；提交。

## Task 9（T9）：删迭代/模块的事件与订阅者；第一个检查点

**Files：** `planning/{module.go,module_events.go}`（`Deps.Events`、`EventTypes()`）；`planning/app/{delete_cycle.go,delete_module.go}` 与测试（固定次序加“发布”）；`issue/{module_events.go}`、`issue/app/membership_events.go`（订阅者）与测试；`issue/adapter/postgres` 的移除语句；`bootstrap/{app.go,events.go,events_test.go}`（事件目录）；bootstrap 组合测试（`planning_deletion_memberships_test.go`）与交错测试；`e2e/fixtures/furnish.ts`（`furnishPlanningRows` 回答所建迭代与模块的 id，`furnishIssueRows` 接收它们、以项目成员的令牌经接口把工作项加入；`furnishWorkspace` 改为先 planning 后 issue）、调用两者的删除故事（`p4-archive.spec.ts`、`w3-workspace-settings.spec.ts`、`i21-purge.spec.ts` 等，以 grep 找全，调用次序对调）、`e2e/fixtures/assert/catalog.ts`（`asynchronousDeletionTable` 加两张表）。

- [ ] Step 1：测试（RED）：删迭代后归属同一时刻软删、每个工作项一条 `deleted` 动态（名字来自事件）、工作项记为被删除者此刻修改、不发 `issue.updated`；删模块同理；删一个没有归属的迭代不写；载荷严格解析（类型、实体、缺名字被拒）；交错：写归属与删迭代两种次序（先删则 PATCH 422，先写则被移除并记动态），删模块同理；事件目录。
- [ ] Step 2：实现（R6）；e2e furnish 与目录。
- [ ] Step 3：变异：订阅者不按事件的时刻盖章、漏写动态、不发布事件时测试失败；串行完整七项（第一个检查点，项目/工作区删除故事此时转绿）后推送并核分支 CI。提交。

## Task 10（T10）：自动归档跳过仍在未结束迭代或模块里的工作项

**Files：** `issue/app/{auto_archive.go,archive_ports.go}` 与测试（固定次序加一步）；`issue/adapter/postgres/queries/auto_archive.sql` 与测试；`issue/module.go` 的接线；bootstrap 的四个直接构造 `AutoArchiveDeps` 的测试与新的组合测试。

- [ ] Step 1：测试（RED）：在进行中、未开始、草稿迭代里的不归档，在已结束迭代里的照常归档；在目标日为空、未到的模块里的不归档，目标日已过的照常；已删除的迭代/模块不算；项目时区跨日；调用次序 `Lock → States → Now → OpenTargets → Update`。
- [ ] Step 2：实现（R9）；变异：用 UTC 的今天、把今天结束的迭代当作已结束、排除条件漏掉模块时测试失败。提交。

## Task 11（T11）：接口故事

**Files：** `e2e/stories/planning/`（G4、G11、G14 的接口与数据库故事）、`e2e/fixtures/`（归属的调用与数据库断言）。

- [ ] Step 1：G4、G11、G14 的接口故事（PAT 与浏览器令牌对等；数据库断言：归属行、动态行、软删与审计）；先跑一个期望错误的版本看到断言失败，再修正。
- [ ] Step 2：`make build` 后聚焦本目录与项目/工作区删除故事通过；提交。

## Task 12（T12）：文档与交接回执

**Files：** `docs/v0/plane-diff.md`；输入交接 M2 §1–§2、M4 M6-03、M5-M6-03 的“处理结果（M6/P3a）”（逐条核对与 P3a 相关的条款，没有相关的不写）；设计 §14 的 P3a 行。

- [ ] Step 1：差异（spec R10、§2.2 的唯一）与回执，status 保持 open。
- [ ] Step 2：文档链接检查；提交。

## Task 13（T13）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（收据），推送准确 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review（新的只读审者），Critical/Important 一次修复轮各有 RED/GREEN，Minor 入台账；阶段记录 `reviews/P3a-issue-membership-review.md`；spec §4 勾选（A8 待合入后由 P3b 的 T1 回填）。
- [ ] Step 3：noff 合入、准确 main CI、归档与正常清理。
