# M6 / P2 模块的后端实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：待预检（2026-11-07）。

**Goal：** 在 `planning` 模块交付模块本身的接口、规则、权限、链接抓取与删除/清理，并把链接的规整校验与抓取从 `issue` 移出共用。

**Architecture：** `planning` 拥有 `modules`、`module_members`、`module_links`；项目事实、项目行锁与成员角色经 `project` 的端口；写入沿用 P1 的加锁之路（同一个项目写闸门），`see` 认迭代、模块、模块链接三种对象；链接检查在 `internal/shared`，抓取在 `platform/outbound/linkpreview`，`bootstrap` 唯一接线。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/River/Playwright，不加依赖。

**Spec：** [P2-modules-backend](../specs/P2-modules-backend.md)（§2.4 的守卫清单与 §3 的 R1–R16 是本计划的约束）；设计 [M6](../M6-design.md) §3.2、§3.6–§3.8、§4、§5、§6、§9；P1 的 [spec](../specs/P1-cycles-backend.md) §3 与代码是模板。

## Global Constraints

- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p2`，分支 `codex/m6-p2`，基线 main `0a601f21`；主目录永远 main。提交详细中文，前缀 `feat/fix/docs/test/refactor(M6/P2)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`，推送用 `/usr/bin/git`；noff 已获连续授权。
- `plane/`、`refer/` 只读且不在其内跑 git；不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。
- T1 改 phase 为 `M6/P2`；不延长任何 `until`。
- 手写文件约 400 行内；SQL 只经本模块 sqlc；端口由消费方声明；模块不互相 import；不放宽 `TestSQLCSchemaScope`、`rawsql_test`、`archtest`、删除/清理/运行角色守卫。
- 新错误码在引入它的同一 Task 进 `web/apps/web/core/lib/error-messages.ts` 与 `web/packages/i18n/src/locales/{en,zh-CN}/errors.json`，跑 `make test-web`；每个声明的码在 planning 的 apitest 二进制里有处理器测试回答。
- 测试夹具：`module_view` 默认关，凡建模块的 Go 夹具与 e2e 先打开它；删除守卫的父/邻种子以 SQL 写入（夹具允许），故事的前置数据经接口。
- 每个 Task：先写会失败的测试并看到它因所测性质失败，再实现到通过；安全、加锁、访客性质至少一个能让测试失败的变异（只计能编译的变异）。每 Task 跑聚焦回归；T3、T4 之后的目录守卫在本 Task 内转绿；T7 之前不做整段七项检查点，T7 起有产品变化的检查点推送前串行完整七项。
- P1 的迭代测试是回归：T2、T4、T6 只能改它们的导入与夹具，不改断言。
- 证据放 worktree 的 `.superpowers/sdd/P2-modules-backend/`（台账 `progress.md`、收据经 `run-check.py`、变异经 `mutate.py`，照 P1）。

## Review Focus

1. 共用链接的移动：只是移动，工作项链接的行为、错误、元数据与抓取不变；架构测试通过。
2. 三种对象的加锁之路：链接经模块到项目；锁下重读；`cycle.*` 与 `module.*` 各看自己的开关；迭代的回答不变。
3. 拒绝次序与资格：spec R2 的分法；负责人与成员只检查新的；OwnOnly 的写 403。
4. 名字与链接的唯一：不分大小写的名字、同模块的地址；并发写在项目行锁下串行，交错测试只成一次。
5. 删除与清理：删模块连带子行同一时刻；任务的命名步骤沿用原 At/By、不取项目锁、可续跑、不误伤邻项目；清理子先父、合规模追踪器、运行角色可执行。
6. 抓取：投递在写入事务内，元数据只在地址仍匹配时写，改地址重置。

## Task 1（T1）：P1 回填与阶段

**Files：** `docs/v0/M6-cycles-modules/reviews/P1-cycles-backend-review.md`（抬头与 §7）；`specs/P1-cycles-backend.md`（状态、A8）；`M6-design.md` §14（P1 行“已完成”、P2 行“进行中”）；`tools/keywords.json`。

- [ ] Step 1：按 P1 归档的台账回填：最终候选 `72b83beb` 的分支 CI 42340200507（三个作业首次通过）、noff `0a601f21`（父 `910a55cd`、`72b83beb`，树与候选相同）、main CI 42343073949（首次通过）、归档 `.superpowers/sdd/M6-P1-integration-20261107`（清单与核验结果）、worktree 与分支已清理；勾选 A8。
- [ ] Step 2：phase 改 `M6/P2`，`node tools/keywords.mjs` 通过（没有到期的例外）；文档链接检查；提交 `docs(M6/P2): 回填P1的集成结果并进入P2阶段`。

## Task 2（T2）：链接的规整校验与抓取移出 issue

**Files：** `server/internal/shared/link.go`、`link_test.go`；`server/internal/platform/outbound/linkpreview/`（由 `issue/adapter/crawler/` 移来）；`issue/domain/{link.go,link_metadata.go,link_test.go,link_metadata_test.go}`、`issue/app/{crawl_ports.go,crawl_link.go,create_issue_link.go}`、`issue/adapter/{http/links.go,postgres/link_metadata.go,postgres/links.go}`、`issue/module.go`；`bootstrap/app.go` 与新的 `link_previews.go`；`api/common.yaml`、`api/modules/issue.yaml`、生成物；`platform/httpserver/apitest/issue_link_shapes_test.go`；设计 §3.2 的措辞。

- [ ] Step 1：先移测试（RED）：`link_test.go` 的地址与标题用例、`link_metadata_test.go` 移到 `shared`，`crawler` 的六个测试文件移到 `linkpreview`，改包名与调用名后看到编译失败。
- [ ] Step 2：移代码（spec R12）：`shared.CheckLinkURL`、`shared.CheckLinkTitle`、`shared.LinkMetadata`；`linkpreview.New`/`Crawl` 返回 `linkpreview.Metadata`；`issue` 改用它们、`Deps.Outbound` 换为 `Deps.LinkCrawler`；`bootstrap` 一个适配器；common 的 `LinkMetadata`、`LinkMetadataError` 与 `make gen`。`make lint-go` 与 `go test ./internal/shared/... ./internal/platform/outbound/... ./internal/modules/issue/... ./internal/archtest/...`、bootstrap 的工作项链接测试（`-run 'Link|Crawl'`）通过。
- [ ] Step 3：变异：`CheckLinkURL` 放过带凭据的地址、`linkpreview` 不再截断响应体时，移来的测试失败；设计 §3.2 的措辞改为 R12 的落点；提交 `refactor(M6/P2): 链接的规整校验移入shared、抓取移入outbound，供工作项与模块共用`。

## Task 3（T3）：三张表、动作与表级守卫

**Files：** `server/migrations/sql/00040_planning_modules.sql`、`00041_planning_module_members.sql`、`00042_planning_module_links.sql`；`server/sqlc.yaml`；`deploy/runtime-grants.sql`；`server/migrations/{planning_schema_test.go,purge_indexes_test.go}` 与迁移总数、表清单所在的测试；`planning/domain/actions.go`；`access/domain/rules.go` 与 `rules_test.go`；`bootstrap/actions_test.go`；`purge_catalog_test.go`。

- [ ] Step 1：先改守卫（RED）：约束与索引名、表数、`module.read/write/delete` 三个动作、运行角色授权覆盖。
- [ ] Step 2：三个迁移（含 Down 与 spec §2.2 的全部约束、索引）、sqlc schema、运行角色授权、动作与规则；`TestMigrationsGoUpDownAndUpAgain`、`TestConstraintAndIndexNames`、`TestTheGrantsFileCoversEveryRelationAndFunction`、`TestEveryActionHasARuleAndEveryRuleAnAction` 通过。
- [ ] Step 3：约束测试（状态 CHECK、日期 CHECK、名字不分大小写唯一且只约束未删除、成员与链接的唯一只约束未删除、元数据 CHECK、外键去向）；变异：删状态 CHECK、名字索引去掉 `lower`、`member_id` 改 `SET NULL`、去掉链接的 EXCLUDE 时测试失败。提交。

## Task 4（T4）：删除任务的命名步骤、清理与删除守卫

**Files：** `planning/app/{delete_rows.go,purge.go}` 与测试；`planning/adapter/postgres/queries/{delete_rows.sql,purge.sql}`、`Store`；`planning/module.go` 的清理步骤；`bootstrap` 的 `project_deletion_test.go`、`workspace_deletion_test.go`、`purge_late_deletion_test.go`、`purge_runtime_test.go`、`purge_scale_fixture_test.go`。

- [ ] Step 1：守卫与夹具（RED）：父/邻项目各以 SQL 种一个模块、一个成员、一条链接并核删除后软删；运行角色清理组 `{"planning", {"cycles", "modules", "module_members", "module_links"}}`；规模夹具。
- [ ] Step 2：`DeleteRows` 改为命名步骤 `cycles → modules → module_members → module_links`（spec R16）；清理步骤 `module_links → module_members → modules → cycles`；单元测试：每步原 At/By、续跑、预算边界的剩余判断按当前步骤、邻项目不受影响、重试幂等；P1 的迭代删除测试不改断言。
- [ ] Step 3：本地跑一次 `NERVE_PURGE_SCALE=1` 的 `TestPurgeScale`（收据）；变异：漏掉 `module_links` 步骤、清理把 `modules` 放到 `module_links` 之前、用 worker 时钟代替原 At 时测试失败。提交。

## Task 5（T5）：模块与链接的纯规则

**Files：** `planning/domain/{module.go,module_patch.go,module_link.go,errors.go}` 与测试。

- [ ] Step 1：表驱动测试（RED）：名字规整（复用 `CleanName`）、说明的 NUL、状态六种与默认、日期顺序（各自可空）、`member_ids` 上限与重复（下标）、`sort_order` 有限；PATCH 的合并日期（R4）、成员集合的加入/移出/留下、没有改动的判定；链接的建立与修改沿用 `shared` 的检查。
- [ ] Step 2：实现；变异：日期比较 `<=` 改 `<`、重复成员不报、成员集合相同仍算改动时测试失败。提交。

## Task 6（T6）：三种对象的加锁之路与成员角色端口

**Files：** `project/app/planning_ports.go`、`project/module.go`（`PlanningProjects.ActiveRoles`）；`bootstrap/ports.go` 的转换；`planning/app/{ports.go,decide.go,lock.go,read.go}` 与测试。

- [ ] Step 1：单元测试（RED）：模块与链接的预检（对象 → 项目）与锁下重读（项目 → 对象），链接所在的模块已删除为 `planning.module_link_not_found`；`module.*` 看 `module_view`、`cycle.*` 看 `cycle_view`；OwnOnly 的写 403；锁下再判定（等待期间被删、被移到别的项目、开关被关）；P1 的 `lock_test` 不改断言。
- [ ] Step 2：实现（spec R1）；变异：模块的写看 `cycle_view`、链接在锁下不重读、AllowCreator 读错对象的建立者时失败。提交。

## Task 7（T7）：建立模块（第一个模块接口）

**Files：** `api/modules/planning.yaml`、`api/openapi.yaml`、生成物；`planning/app/create_module.go`、`adapter/http/`、`adapter/postgres/`；`planning/module.go` 的接线；权限矩阵（`permission_matrix_targets_test.go`、`seeded_test.go`、`prepareMatrix`、新的模块表与列）；`e2e/fixtures/{furnish.ts,catalog.ts}`；错误文案（`planning.modules_disabled`、`planning.module_name_taken`、`planning.archived` 改为通用）。

- [ ] Step 1：契约与生成；矩阵的模块表（spec R10）与种子；e2e 的 furnish 打开 `module_view` 并经接口建模块（管理员为负责人与成员），`asynchronousDeletionTable` 加三张表。
- [ ] Step 2：组合与处理器测试（RED）：最少字段与全部字段、先验 422、负责人与成员的资格（访客、非成员、已离开者）、重名（不分大小写、已归档也算、已删除不算）409、开关 409、已归档项目 409、访客/OwnOnly 403、看不到的项目 404、`sort_order` 规则、成员行与审计同一时刻；交错测试：两个事务同时建立同名模块只有一个成功、无 `40P01`。
- [ ] Step 3：实现；变异：名字唯一的映射丢失（变 500）、资格检查放到锁外、成员不检查时失败；串行完整七项（第一个检查点）后推送并核分支 CI。提交。

## Task 8（T8）：读取与归档目录

**Files：** `planning.yaml`；`planning/app/{list_modules.go,get_module.go}`；处理器与组合测试；`planning.module_not_found` 的文案。

- [ ] Step 1：测试（RED）：两种顺序、互斥、`archived` 非布尔 400、已删除模块与已删除项目 404、`member_ids` 升序且不含已移出的、访客可读。
- [ ] Step 2：实现；提交。

## Task 9（T9）：修改模块

**Files：** `planning.yaml`；`planning/app/update_module.go`；测试。

- [ ] Step 1：测试（RED）：各字段、日期合并（R4）、空改动与相同成员集合不写、成员整体替换（新加入的检查、留下的已离开者保持、移出的软删且同一时刻）、换负责人与清空负责人、改名重名 409（排除自身、只改大小写可以）、已归档 409。
- [ ] Step 2：实现；变异：留下的成员也检查资格、移出的成员物理删除、重名不排除自身时失败。提交。

## Task 10（T10）：归档、恢复与删除模块

**Files：** `planning.yaml`；`planning/app/{archive_module.go,delete_module.go}`；测试；`planning.module_not_closed` 的文案。

- [ ] Step 1：测试（RED）：只有已完成、已取消能归档；重复归档与恢复 200 不写；恢复不看状态；`archived_at` 与审计同一时刻；管理员删任何、成员只删自己建立的、访客与 OwnOnly 403；删后 404；模块、成员、链接的 `deleted_at` 与 `updated_*` 同一时刻；开关与已归档项目 409。
- [ ] Step 2：实现；变异：归档不看状态、删除不连带链接、去掉建立者判断时失败。提交。

## Task 11（T11）：模块链接的增改删与列表

**Files：** `planning.yaml`；`planning/app/module_links.go`；`adapter/postgres/queries/module_links.sql`；处理器与组合测试；矩阵的链接表；`planning.module_link_not_found`、`planning.module_link_exists` 的文案。

- [ ] Step 1：测试（RED）：建立（补协议、校验同工作项链接、标题为 null 与空串）、列表顺序、改地址与改标题、删除、同模块重复地址 409（排除自身、已删除不算、另一个模块可以）、已归档的模块 409、链接所在模块已删除 404、开关与权限；交错测试：两个事务同时建立同地址只有一个成功。
- [ ] Step 2：实现（此 Task 元数据保持 `{}`，抓取在 T12）；变异：重复映射丢失、已归档的模块不拒时失败。提交。

## Task 12（T12）：链接元数据的抓取

**Files：** `planning/app/{crawl_link.go,module_links.go}`；`adapter/river/`（`planning.crawl_link` 的投递器与 worker）；`adapter/postgres/queries/link_metadata.sql`；`planning/{module.go,module_jobs.go}`；`bootstrap` 的组合测试（照 `issue_link_jobs_test.go`、`issue_crawl_jobs_test.go` 的假出站）。

- [ ] Step 1：测试（RED）：建立与改地址各投递一次、只改标题不投递、写入回滚不投递（spec R11）；worker 写入元数据、地址已改或链接已删除时不写；改地址重置 `metadata` 为 `{}`；`EnqueuedKinds` 含新种类。
- [ ] Step 2：实现；变异：在事务外投递、写元数据不比较地址、改地址不重置时失败。提交。

## Task 13（T13）：接口故事

**Files：** `e2e/stories/planning/`。

- [ ] Step 1：G9、G10、G13 的接口故事（PAT 与浏览器令牌对等、数据库断言：模块行、成员行、链接行、审计与软删）；先跑一个期望错误的版本看到断言失败，再修正。
- [ ] Step 2：`make build` 后聚焦本目录与 P4/W2/W3 删除故事通过；提交。

## Task 14（T14）：文档与交接回执

**Files：** `docs/v0/plane-diff.md`；输入交接 M2 §2、M3 §2、M4 M6-03、M5-M6-01、M5-M6-03 的“处理结果（M6/P2）”（逐条核对与 P2 相关的条款，没有相关的不写）；设计 §14 的 P2 行。

- [ ] Step 1：差异（spec §2.2 的表、名字不分大小写、链接元数据由后台抓取、负责人与成员的资格、归档与重复归档 200、已归档模块不能改与不能写链接）与回执，status 保持 open。
- [ ] Step 2：文档链接检查；提交。

## Task 15（T15）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（收据），推送准确 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review（新的只读审者），Critical/Important 一次修复轮各有 RED/GREEN，Minor 入台账；阶段记录 `reviews/P2-modules-backend-review.md`；spec §4 勾选（A9 待合入后由 P3 的 T1 回填）。
- [ ] Step 3：noff 合入、准确 main CI、归档与正常清理。
