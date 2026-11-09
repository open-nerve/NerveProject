# M6 / P3b 列表引擎的迭代与模块、迭代与模块页的显示设置实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（修正后，[re-60](../../../reviews/re-60-M6P3b预检.md)），实施中。

**Goal：**
- 工作项列表能按迭代、模块筛选（含“无”）与分组；
- 能按迭代名、模块名两向排序，Go 与网页共用同一份夹具；
- 显示属性多出“迭代”“模块”；
- 项目偏好认得这些新词；
- 迭代页与模块页各有调用者自己的显示设置，经 `/me/cycles|modules/{id}/preferences` 读写。

**Architecture：**
- **列表引擎**：词汇、解析、三条静态 SQL 都在 `issue`。按名字排序要的迭代/模块目录，经 `issue/app` 声明、`planning/app` 实现的端口，在读行的同一个快照里按需取得，以数组传入语句。
- **显示设置的值**：三页共用一份，契约放 `common.yaml`，Go 值类型放 `internal/shared`。
- **偏好**：两张偏好表归 `planning`。读走 planning 的读之路；改走新的共享锁之路（项目行 `FOR SHARE`），校验经 `issue.ListSettings`，由 `bootstrap` 接线。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/River/Playwright，不加依赖。

**Spec：** [P3b-list-engine](../specs/P3b-list-engine.md)（§2 的合同与 §3 的 R1–R10 是本计划的约束）；设计 [M6](../M6-design.md) §3.7、§4、§5、§6、§7；M4 的列表引擎（`issue/domain/listvocab.go`、`issue/app/listing.go`、三条列表语句）、标签的筛选、分组、名字排序与附件计数的按需目录是模板；项目偏好（`project/app/{get,update}_preferences.go`、`00026`、`00038`）是偏好的模板。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p3b`，分支 `codex/m6-p3b`，基线 main `2648a739`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/docs/test/refactor(M6/P3b)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 保持 `M6/P3`（P3a 已改）；不延长任何 `until`。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内。
- SQL 只经本模块的 sqlc：`issue` 的语句不连 `cycles`、`modules`，planning 的语句不连 `issues`。
- 端口由消费方声明；模块不互相 import。
- 不放宽 `TestSQLCSchemaScope`、`rawsql_test`、`archtest`，以及删除、清理、运行角色的守卫。

**测试纪律**
- 每个 Task 先写会失败的测试，看到它因所测性质失败，再实现到通过。
- 安全、加锁、访客的每个性质至少一个能让测试失败的变异，只计能编译的变异。
- 每个 Task 跑聚焦回归：issue 列表的平铺、分组、子分组三种形态都要跑。

**回归测试的边界**
- 回归测试指 M4 的列表与项目偏好、P1–P3a 的迭代、模块与归属的测试。
- 只在 spec 说契约变化的地方改：由拒绝翻为允许的词汇行、键与场景的个数、夹具、假实现、类型名、仍被拒绝的替代词。其他断言不改。

**检查点与证据**
- 检查点时点：T7 结束时串行完整七项，推送并核分支 CI。此时列表引擎与项目偏好已完成，新表尚未建。T8 建表后 e2e 的删除故事要等 T11 的 `furnishPlanningRows` 能经接口写偏好行才转绿，所以 T8–T11 之间只跑聚焦回归，不做整段检查点。T14 收尾再串行一次。
- 证据放在 worktree 的 `.superpowers/sdd/P3b-list-engine/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，调研笔记在 `research/`。实现代理先读同目录的 `implementer-brief.md`。

## Review Focus

1. **词汇的范围（R1、R3）**：筛选与分组只限项目列表，“我的工作项”回答 422；四个排序词两种列表都可用；`appendID` 的 `none` 不再落到负责人。
2. **三条列表语句**：新参数在三条语句与参数复制里逐一相同；模块的多值分组与标签同构；`none` 指没有未删除的归属；已归档的迭代/模块照常算。
3. **目录（R2）**：只在名字排序时取；与行在同一个快照里；“我的工作项”覆盖全部可见项目；包括已归档的，不包括已删除的。
4. **共用夹具（R7）**：同一个提交改 Go 的三个消费者、网页的比较器与 e2e 的 i4。
5. **共用显示设置（R5）**：重构不改任何回答与存储；bodyshape 对 `common.yaml` 的请求体有效。
6. **偏好之路（R6、R8）**：
   - GET 不写；PATCH 走共享锁、不进写闸门，OwnOnly、已归档、开关关闭都不拒绝；
   - 本页的键回答 400；拒绝次序；只碰自己的一行；
   - 与删除的两种交错。
7. **删除与清理**：两张表在删迭代/模块、删除任务、清理、守卫、规模夹具里都覆盖。

## Task 1（T1）：P3a 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P3a-issue-membership-review.md`：抬头与 §7。
- `specs/P3a-issue-membership.md`：抬头与 A8。
- `M6-design.md`：抬头、§14 的 P3a 行“已完成”、P3b 行“进行中”。

- [ ] Step 1：按 P3a 归档（主目录 `.superpowers/sdd/M6-P3a-integration-20261108`）里的台账与 CI 收据回填 §7：
  - **检查点 CI 42478883539**：attempt 1 的 server (0) 在下载 golangci-lint 时遇 GitHub 504；attempt 2 只重跑失败的作业，四个作业成功。
  - **最终候选**：`384a365f` 的分支 CI 42484965936 首次通过。
  - **合入与 main CI**：noff `2648a739`（两个父提交 `ddd79a9f`、`384a365f`，树与候选相同），main CI 42487991301 首次通过。
  - **归档**：2378 个文件、387,069,731 字节、未知 0、清单 sha256 `cb75d968fd03db3eed3f03a71efce10b938cc04ccb62947126a5a5289dc9bf44`，复核 `archive.py verify`。
  - **清理**：`cleanup.log` 记录 worktree 与本地、远端分支已删除。
  - 勾选 A8；把阶段记录与 spec 的抬头改为已完成。
- [ ] Step 2：设计 §14 的 P3a 行改为“已完成”并附证据，P3b 行改为“进行中”，链接 P3b 的 spec 与 plan；抬头的状态句随之改。
- [ ] Step 3：检查文档链接（`doc-links.py`），然后提交 `docs(M6/P3b): 回填P3a的集成结果，进入P3b阶段`。

## Task 2（T2）：按迭代、模块筛选

**Files：**
- `issue/domain/{listvocab.go,list_query.go,list_filters.go}` 与测试（`listvocab_test.go` 的 L41-42 翻转）。
- `issue/adapter/http/list_query.go` 与测试（`listSingleQuery`）。
- `issue/adapter/postgres/{list_parameters.go,list_issue_groups.go,list_issue_subgroups.go}`、`queries/{list_issues.sql,list_issue_groups.sql,list_issue_subgroups.sql}` 与适配器测试。
- `api/modules/issue.yaml`：`listIssues` 的 `cycle_id`、`module_id` 两个参数，照 `label_id` 的写法与说明。
- 生成物。
- bootstrap 的 HTTP 列表测试：新文件 `issue_list_planning_filters_test.go`。

- [ ] Step 1：先写测试（RED）：
  - 词汇：两个筛选只限项目列表，值为 uuid 或 `none`。
  - 解析：`appendID` 把 `none` 记到正确的字段（为 `cycle_id=none` 写一个断言 `Assignees.None` 仍为假的单元测试）；`ListFilterValueAllowed` 认两个字段。
  - HTTP：项目列表重复给出参数回答 422，“我的工作项”给出这两个参数回答 422。
  - 适配器的三种形态：按迭代、按模块、`none`、`none,<id>` 取并、与标签取交；已删除的归属不算；已归档的迭代/模块照常算。
  - bootstrap：访客与 OwnOnly 只看到自己能看见的，分组的总数也只算这些。
- [ ] Step 2：实现。`appendID` 改为按字段的 `switch`，四个分支各一。三条语句的筛选照标签的模板，参数复制三处。然后 `make gen`，跑聚焦回归。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - `appendID` 退回两路 `else`；
  - 模块筛选漏掉 `deleted_at IS NULL`；
  - `none` 与 id 改为取交；
  - 子分组语句漏掉迭代筛选。
  
  然后提交。

## Task 3（T3）：按迭代、模块分组

**Files：**
- `issue/domain/{listvocab.go,list_query.go}` 与测试：分组、子分组、分组键语法、`canonicalizeListGroupKey`；`listvocab_test` 与 `issue/app/list_settings_test.go:86` 的翻转。
- 两条分组语句的 `CASE`：分组一处，子分组两处。
- `api/modules/issue.yaml` 的 `IssueListGroupField`（含它的说明文字 `issue.yaml:1850`，写明两个新词只限项目列表），以及生成物。
- 网页（编译修正）：`lib/issue-membership.ts` 的 `groupKeys` 加 `module_id`，`lib/issue-filters.ts` 的 `issueFieldForGroup` 加 `module_id → module_ids`，各配一个单元测试。
- bootstrap 的分组测试：`issue_list_planning_groups_test.go`。
- 仍被拒的替代词：`bootstrap/project_issue_preferences_test.go:114` 与 e2e `stories/issue/i5-list-preferences.spec.ts:199-205` 把 `group_by:"cycle_id"`、`sub_group_by:"module_id"` 换成仍被拒的词，例如 `project_id`（服务端从本 Task 起接受原来的词）。

- [ ] Step 1：先写测试（RED）：
  - 迭代单值分组；模块多值分组，一个工作项进它的每个模块组；`none` 组；
  - 组的总数与去重后的外层 `total`；组的次序；只回答非空组；
  - 按迭代分组再按模块子分组，以及反过来；
  - 组内续页与双向分页；
  - “我的工作项”回答 422；项目偏好接受 `group_by:"cycle_id"`；
  - OwnOnly 的组总数。
- [ ] Step 2：实现，然后 `make gen`、`make lint-web test-web`，跑聚焦回归。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 模块分组只取第一个模块；
  - 空数组不进 `none`；
  - 子分组的第二个 `CASE` 漏掉 `module_id`。
  
  然后提交。

## Task 4（T4）：planning 的名字目录端口

**Files：**
- `issue/app/planning_ports.go`：`PlanningDirectory` 与 `PlanningName`，并入 `PlanningPorts`。
- `planning/app/issue_targets.go` 的同目录新文件 `issue_directory.go` 与测试。
- `planning/adapter/postgres/queries/issue_targets.sql`：两条按项目 id 数组读的语句，`ORDER BY id`，与 `Store`、适配器测试。
- `planning/module.go`：`IssueTargets` 加方法，或在 `Provided` 加一项。
- `bootstrap/ports.go` 的 `issuePlanning` 转换与 `planning_ports_test.go`。
- `issue/app/fakes_planning_test.go`。

- [ ] Step 1：先写测试（RED）：
  - 回答所给项目里未删除的迭代与模块，包括已归档的；
  - 别的项目、已删除的不回答；
  - 空数组回答空；
  - 次序按 id；
  - 开关关闭的项目照常回答。
- [ ] Step 2：实现并接线。这个 Task 的端口还没有调用者，T5 使用。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 漏掉 `deleted_at IS NULL`；
  - 排除已归档的；
  - 不按项目过滤。
  
  然后提交。

## Task 5（T5）：按迭代名、模块名排序与共用夹具

**Files：**
- `issue/domain/{listvocab.go,list_cursor_key.go}`，`listvocab_test.go:22` 翻转。
- `issue/adapter/postgres/list_cursor.go:47`。
- `issue/app/{list_ports.go,listing.go}`：`ListDirectories.Cycles/Modules`，只在名字排序时取，在同一个快照里。
- `list_parameters.go` 的成对数组与参数复制三处，以及三条语句的目录 CTE 与名字键。
- `api/modules/issue.yaml` 的 `IssueListOrder`，以及生成物。
- `api/testdata/issue-order.json`：两个场景（R7）。
- Go 的三个消费者：`issue/domain/list_order_test.go`（11→13）、`issue/adapter/postgres/list_order_test.go`（播种分支）、`bootstrap/issue_list_orders_test.go`（21→25；`seedListOrder` 的两个分支以 SQL 种迭代、模块与归属，读取不看开关，没有迭代的那一行照种）。
- 快照与证据：`bootstrap/issue_list_explain_test.go:41` 的目录分组；仿 `issue_list_attachment_snapshot_test.go` 的目录快照测试。
- 网页：
  - `lib/issue-order.ts` 与 `issue-order.test.ts`（13）；
  - `store/project/fake-projects.ts` 的 `cycleOf`、`moduleOf`；
  - `lib/paged-groups.test.ts:12`、`store/issue/fake-issues.ts:110`；
  - `store/issue/helpers/issue-list-binding.ts`：空 `Map` 与 `prepareCatalog` 的检查，并配一个单元测试证明检查会抛出（R9）。
- e2e：`stories/issue/i4-list-order-calendar.spec.ts`（27）。

- [ ] Step 1：先写测试（RED），并在夹具里加两个场景，看到 Go 与网页的测试同时失败：
  - 四种形态、两种列表；
  - 没有的排在最后；同键按 id 随方向排；
  - 游标的文本键跨页；
  - 目录与行在同一个只读快照里：照附件计数的快照测试，在目录语句处暂停，另一个连接改名或加入，回答不变；
  - 其余排序不发 planning 的目录语句：以语句探针断言。
- [ ] Step 2：实现，然后 `make gen`、`make lint-go gen-check lint-web test-web`，跑聚焦回归：issue 全部包、bootstrap 的列表排序与快照测试，以及 e2e 的 `i4`（先 `make build`）。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 模块名取最大而非最小；
  - 没有迭代的排在最前；
  - 目录排除已归档；
  - 每种排序都取目录：以语句探针证明；
  - 网页比较器对模块取第一个而非最小。
  
  然后提交。

## Task 6（T6）：显示设置的形状三页共用（重构，不改回答与存储）

**Files：**
- `api/common.yaml`：`IssueListCalendar`、`IssueListDisplayFilters`、`IssueListDisplayProperties`，由 `project.yaml` 搬来，旧名删除；`project.yaml` 改为引用它们。
- 生成物：`apigen/components.gen.go`、项目的 `server.gen.go` 与 bodyshape、`api/dist`、`schema.gen.ts`。
- `server/internal/shared/` 新文件：`ListFilters`（十个共同的筛选键与转为 map 的方法）、`ListDisplayFilters`、`ListCalendar`、`ListDisplayProperties` 与默认值，并配测试。
- `project/domain/{issue_preferences.go,preferences.go}`、`project/app/list_settings.go`、`project/adapter/{http,postgres}` 与测试：改用共用类型，删除被替代的类型。
- `issue/app/list_settings.go` 与测试：`Validate(filters, display)` 的新签名，去掉显示属性参数，删除手写的属性清单（R5）；`issue/module.go:44-50` 的别名 `ListCalendar`/`ListDisplayFilters` 与导出的 `ListSettings` 接口随之改。
- `project/app/issue_preferences_test.go:14-26` 的假实现 `capturedSettings`。
- `bootstrap/{ports.go,app.go}`：删除 `projectListSettings`，直接交 `issue.ListSettings`。
- 网页：`store/issue/project/filter.store.ts:8` 与 `:233` 的类型名。
- e2e：`stories/issue/i5-list-preferences.spec.ts` 的 `:2`、`:91`（`ProjectDisplayFilters`）、`:100-113`、`:374`（`keyof ProjectDisplayProperties`）改名。
- 提交前 grep：web 与 e2e（除 `schema.gen.ts`）不再有 `ProjectDisplayFilters`、`ProjectDisplayProperties`、`ProjectListCalendar`；server 不再有被删的 `ListDisplayFilters`、`ListCalendar` 旧类型。

- [ ] Step 1：先核实 bodyshape。在 `common.yaml` 加一个 schema，并让 `ProjectPreferencesUpdate` 引用它，`make gen` 后跑 project 的 bodyshape 与 apitest：多余的键与错误的类型仍回答 400，收据记为 RED/GREEN。如果生成器解析不了外部引用，就停下上报，不绕开。
- [ ] Step 2：完成搬移与 Go 的共用类型，跑全部的项目偏好测试（project 模块、`bootstrap/project_issue_preferences_test.go`、`project_preferences_*`、`p5_preferences_scope_test.go`）。它们只因类型名改动，断言不动。新增 `shared` 的单元测试：默认值，以及 `ListFilters` 转 map 时 nil 与空数组的区别。
- [ ] Step 3：`make gen-check lint-go lint-web test-web` 全过。再做变异，以下每一项都应让测试失败：
  - 共用默认值改一个布尔；
  - `ListFilters` 把 nil 当空数组。
  
  然后提交 `refactor(M6/P3b): …`。

## Task 7（T7）：项目偏好认得新词；第一个检查点

**Files：**
- `server/migrations/sql/00045_project_project_user_properties_planning_display.sql`（R4），与新的迁移测试（照 `attachment_display_test.go`：上行、下行、再上行，以及存过的行）。
- `server/sqlc.yaml` 的 project 条目，迁移总数 44→45（`trigram_schema_test`）。
- `shared.ListDisplayProperties` 加 `Cycle`、`Modules`（默认 `true`）。
- `project/domain.Filters` 加 `CycleID`、`ModuleID`，与它的转换、HTTP 映射。
- `api/common.yaml` 的显示属性加两个必填键；`api/modules/project.yaml` 的 `ProjectIssueFilters` 加两个可省略的键；生成物。
- 网页：`store/project/fake-projects.ts:81-94`、`components/issues/issue-layouts/roots/project-lifecycle.test.tsx:92-105`。
- e2e：`stories/issue/i5-list-preferences.spec.ts:100-113`（补两个键）、`stories/project/p8-project-preferences.spec.ts:40-54`。
- bootstrap 的项目偏好测试：`project_issue_preferences_test.go` 的默认与自定义 JSON 常量（`customProperties` 等）、`permission_matrix_project_test.go:154-160` 的偏好行。

- [ ] Step 1：先写测试（RED）：
  - 迁移：旧行补上两个属性；默认值；12 个筛选键与 14 个显示属性；下行的次序为删新 CHECK → 剥键 → 旧默认值 → 重建旧 CHECK（spec R4），存过新键的行下行后合法。
  - 项目偏好：存 `filters.cycle_id`/`module_id` 与两个属性；`DefaultPreferences` 等于列的默认值。
  - 未知的属性键与筛选键回答 400。
- [ ] Step 2：实现，然后 `make gen`，跑聚焦回归。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - Down 不剥新键：再上行或下行失败；
  - 旧行不补属性；
  - 默认值漏掉 `modules`。
  
  提交后串行完整七项（`gates.sh t7-gate1`，第一个检查点），推送，核分支 CI。

## Task 8（T8）：两张偏好表、删除任务与清理的两步，守卫

建表即触发的 Go 守卫（每张表恰好一个清理者、删除目录、两个父范围下都要有种下的行、清理规模必须有过期行）在本 Task 内一并转绿（预检 M1）。

**Files：**
- `server/migrations/sql/00046_planning_cycle_user_properties.sql`、`00047_planning_module_user_properties.sql`（spec §2.4）。
- `server/sqlc.yaml` 的 planning 条目；`deploy/runtime-grants.sql`。
- `server/migrations/{planning_schema_test.go,purge_indexes_test.go,trigram_schema_test.go}`：迁移总数 45→47，清理表数 30→32，`planningNames` 加两张表的约束与索引名。
- 删除任务：`planning/app/delete_rows.go` 的 `DeletionRowSteps`、`deletes` 与 `ProjectRows` 加两步；`queries/delete_rows.sql` 的两条语句与剩余探查 `CASE`（`:45-56`）；适配器 `delete_rows.go`；`delete_rows_steps_test.go:17` 的 `planningSteps` 与 `delete_rows_test.go`。
- 清理：`planning/module.go:73-78` 的步骤子先父（`module_user_properties`、`module_links`、`module_members`、`modules`、`cycle_user_properties`、`cycles`）；`purge.go`、`queries/purge.sql`；`store_test.go:160`、`module_rows_test.go:223` 固定的 `Tables()`。
- bootstrap：
  - `purge_catalog_test.go` 的表数；
  - `issue_project_deletion_jobs_test.go:31-33` 的 `projectPlanningTable` 认两张表；
  - `planning_deletion_fixture_test.go` 的 `seedCycles`、`seedModules` 各以 SQL 种一行偏好（`seedProject`、清理夹具已调用它们）；
  - `purge_scale_fixture_test.go` 的 `purgeScaleTables`、`seedPurgeSparseRows`；
  - `purge_runtime_test.go:49,103` 的 planning 组与老化、核对的表清单。

- [ ] Step 1：先改守卫并写测试（RED）：
  - 约束：同一迭代、同一用户只有一行未删除的，已删除的不占位；筛选不认本页的键；显示属性恰好 14 个键；默认值；外键的去向（删迭代、模块、用户、项目、工作区时 CASCADE，审计列 SET NULL）。
  - 删除任务的步骤清单与剩余探查；清理的次序子先父。
  - 项目/工作区删除、迟到删除与清理的守卫要求两张表有行并核对软删。
- [ ] Step 2：写两个迁移（含 Down）、sqlc 的 schema、运行角色的授权、删除任务与清理的语句和步骤、夹具。表级守卫与 bootstrap 的删除、清理、运行角色守卫都转绿，用聚焦正则跑，留收据。
- [ ] Step 3：本地跑一次 `NERVE_PURGE_SCALE=1` 的 `TestPurgeScale`，留收据。再做变异，以下每一项都应让测试失败：
  - 唯一约束去掉部分条件；
  - 筛选的 CHECK 认本页的键；
  - `cycle_id` 改为 `SET NULL`；
  - 删除任务漏掉 `module_user_properties` 一步；
  - 清理把 `cycles` 放在 `cycle_user_properties` 之前。
  
  然后提交。

## Task 9（T9）：删迭代/模块时软删偏好行

**Files：**
- `planning/app/{delete_cycle.go,delete_module.go}` 与测试：`CycleDeleter` 加一个方法、`ModuleDeleter` 再加一个，固定的调用次序加一步。
- `planning/adapter/postgres` 的两条软删语句（照 `DeleteModuleMemberRows`：一条语句，按 id 次序锁行后写）与 `Store`、适配器测试。
- bootstrap 的删迭代/删模块组合测试（`planning_deletion_*` 或 P1/P2 的删除测试，加偏好行的断言）。

- [ ] Step 1：先写测试（RED）：删迭代/模块时，未删除的偏好行在同一事务、同一时刻、同一写者软删；已删除的保持原来的删除；别的迭代/模块的行不动。
- [ ] Step 2：实现，跑聚焦回归。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - 删迭代漏掉偏好行；
  - 软删不排除已删除的行，改写它们的删除时刻。
  
  然后提交。

## Task 10（T10）：偏好的用例、共享锁之路与动作

**Files：**
- `planning/domain`：两种偏好、`CycleFilters`/`ModuleFilters`（嵌入 `shared.ListFilters`）、默认值、动作 `cycle.preferences`、`module.preferences`（`Actions()`、`ModuleAction`）与测试。
- `planning/app`：`get_preferences.go`、`update_preferences.go`、`list_settings.go`（消费方声明的 `IssueListSettings`）、`lock.go`。新的共享锁之路，并顺手把 `write` 改为嵌入 `named`（R6），与测试。
- `planning/adapter/postgres`：读与 upsert（`ON CONFLICT (cycle_id,user_id) WHERE deleted_at IS NULL`），与适配器测试，其中默认值等于列的默认值。
- `project/adapter/postgres/queries/planning_ports.sql` 与 `PlanningProjects.Share`（`FOR SHARE`），与测试。
- `bootstrap/{ports.go,app.go}`：`planningProjects.Share`；planning 的列表设置端口直接交 `issue.ListSettings`。
- 随端口长大的假实现：`planning/app/lock_test.go:60-66` 的 `lockWorld`、`bootstrap/planning_ports_test.go:20-43` 的 `fakePlanningProjects` 与它的转换测试。
- `access/domain/{rules.go,rules_test.go}`；`bootstrap/actions_test.go`。

- [ ] Step 1：先写测试（RED）：
  - 读：没有行时回答默认值且不写；有行时回答存的值。
  - 改：只改给出的字段；第一次修改写入默认值加改动；时钟在锁之后读。
  - 拒绝次序：词的 422 → 404 → 403。
  - 照常放行：OwnOnly、已归档的项目/迭代/模块、开关关闭都不拒绝。
  - 锁：项目行 `FOR SHARE`，不进写闸门。以锁探针断言 PATCH 会排在持有 `FOR NO KEY UPDATE` 的事务之后。
  - 共享锁之路的调用次序。
  - `write` 嵌入 `named` 的重构不改 P1/P2 的任何断言。
- [ ] Step 2：实现。先单独提交 `refactor` 的重构，再提交功能。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - PATCH 走 `Locks.run`：OwnOnly 或开关关闭时被拒；
  - `Share` 改为不加锁；
  - 时钟在锁之前读；
  - GET 写入默认行。
  
  然后提交。

## Task 11（T11）：偏好的接口、矩阵与交错

**Files：**
- `api/modules/planning.yaml`：四个操作与六个 schema（spec §2.3）；`api/openapi.yaml` 的根 `$ref` 两条新路径（`TestRootListsEveryModulePath`）；以及生成物。
- `planning/adapter/http`：四个处理器、`UseCases` 字段、处理器测试；每个（操作，码）一个。
- bootstrap：
  - 组合测试：新文件 `planning_preferences_test.go`；
  - 矩阵：`permission_matrix_cycles_test.go`、`permission_matrix_modules_test.go` 的行，并带已归档的变体；
  - 交错：新文件 `planning_preferences_interleaving_test.go`；
  - 运行角色。
- e2e：
  - `fixtures/furnish.ts` 的 `furnishPlanningRows` 以管理员的令牌 PATCH 两种偏好；
  - `fixtures/assert/catalog.ts` 的 `asynchronousDeletionTable` 加两张表。

- [ ] Step 1：先写测试（RED）：
  - 回答的形状；
  - 本页的键 400，多余的键 400；
  - 词的 422 字段路径；
  - 只读写自己的一行：两个用户各存一份，互不覆盖；
  - 访客与工作区管理员兼访客：放行；
  - 被移出项目后 403、行保留，重新加入后读回原值；
  - 开关关闭；已归档的项目与迭代/模块；
  - 与删迭代/删模块交错的两种次序：先删则 404；先写则行随删除软删、时刻相同；
  - 运行角色下读写。
- [ ] Step 2：实现，然后 `make gen`，跑聚焦回归，再跑 e2e 的项目/工作区删除故事（P4、W2、W3、I15、I17、I21，先 `make build`）。它们此时转绿。
- [ ] Step 3：变异，以下每一项都应让测试失败：
  - upsert 不带 `user_id` 冲突目标，改成按迭代覆盖；
  - `furnishPlanningRows` 不写偏好：删除故事失败。
  
  然后提交。

## Task 12（T12）：G15 的接口故事

**Files：**
- `e2e/stories/planning/g15-issue-lists.spec.ts`：筛选、分组、排序，经接口并做数据库断言。
- `e2e/stories/planning/g15-planning-preferences.spec.ts`：两种偏好的读写，PAT 与浏览器令牌对等。数据库断言：行、默认值、只碰自己的一行、随迭代删除软删。
- `e2e/fixtures/` 里需要的调用与断言。

- [ ] Step 1：写故事。先跑一个期望错误的版本，看到断言失败，再改正。
- [ ] Step 2：`make build` 后，聚焦跑本目录与 i3、i4、i5、p8，通过后提交。

## Task 13（T13）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`（spec R10）。
- 输入交接：M4 M6-02 的列表引擎部分、M5-M6-02 的附件列随共用设置出现，各写“处理结果（M6/P3b）”，status 保持 open。逐条核对与 P3b 相关的条款，没有相关的不写。
- 设计 §6 的 `PlanningDirectory` 一行收窄（R2）；§7 的列表引擎一句补“筛选与分组只限项目列表”（R1）；§3.7 写明个人显示设置不属于“OwnOnly 时所有 planning 的写 403”（R6 的澄清）；§14 的 P3b 行。

- [ ] Step 1：写差异与回执。
- [ ] Step 2：检查文档链接，然后提交。

## Task 14（T14）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（留收据），推送准确的 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账；
  - 写阶段记录 `reviews/P3b-list-engine-review.md`；
  - 勾选 spec §4；A8 待合入后由 P4 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
