# M7 / P3d plan：最近访问

> spec：[P3d spec](../specs/P3d-recents.md)；预检 [re-92](../../../reviews/re-92-M7P3d预检.md) 修正后通过（处理见它的 §6）。基线 main `9d26b1c3`（P3c 的 noff）。worktree `.codex/worktress/codex/m7-p3d`，分支 `codex/m7-p3d`。台账 `.superpowers/sdd/P3d-recents/progress.md`（ignored）。15 个 Task。下文 `nav/` = `server/internal/modules/navigation/`，`iss/` = `server/internal/modules/issue/`，`pj/` = `server/internal/modules/project/`，`bs/` = `server/internal/bootstrap/`，`mig/` = `server/migrations/`。

## Global Constraints

- spec §2 是合同，§3 的 R1–R14 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；每个声明的码有处理器测试回答（`nav/adapter/http/handler_test.go:27` 的 `apitest.Main`）；契约与处理器同一 Task。P3d 没有新码，网页文案不改（spec §2.4）。
- 模块之间只经端口；navigation 的 SQL 只经自己的 sqlc（不联表 `issues`、`projects`、`intake_issues`）；新代码进新文件，手写文件约 400 行以内。**不再加长的文件**（行数为基线）：`bs/permission_matrix_test.go` 452（只许 `matrixRows()` 那一行就地加一项，净增 0 行）、`bs/permission_matrix_seeded_test.go` 430、`bs/permission_matrix_coverage_test.go` 416、`bs/permission_matrix_cycles_test.go` 394、`bs/permission_matrix_seed_test.go` 390、`bs/permission_matrix_columns_test.go` 389、`bs/project_write_locks_test.go` 426、`bs/ports.go` 378、`iss/app/ports.go` 395、`pj/module.go` 373、`pj/app/ports.go` 363、`e2e/stories/issue/i21-purge.spec.ts` 411、`e2e/stories/workspace/w3-workspace-settings.spec.ts` 404（这些都不改）。新桥进 `bs/navigation_ports.go`（114），issue 的新端口进新文件（R3），navigation 的新代码进新文件（`nav/app/favorite_locks.go` 283、`favorite_sight.go` 227 不改）。
- 端口的 bootstrap 转换在加它的同一个 Task 里接进 `navigation.Deps` 与 `bs/app_modules.go` 的 `navigationModule`（`lint-go` 开着 `unused`，re-91 m5）。
- 允许改的已有测试（改之前在台账逐条写明，只照这里写的改法）：
  - 迁移：`mig/trigram_schema_test.go`（迁移数 53 → 54 的三处 `:44-45`、`:84`、`:106-107`，表名名单 `:52-54` 加 `user_recent_visits`）、`mig/purge_indexes_test.go`（36 → 37，`:40-41`）、`mig/navigation_schema_test.go` 的 `navigationNames`（加 spec §2.3 的每个名字）。
  - 动作：`server/internal/modules/access/domain/rules_test.go` 的 `tableCells`（两行，`{allowed, allowed, allowed, invisible, invisible, invisible, forbidden}`）。
  - 清理：`bs/purge_catalog_test.go`（36 → 37，`:154-155`）、`bs/purge_runtime_test.go`（navigation 一组 `:53` 与 `tables` `:113` 加表，准备里加 `seedRecentVisits` 一次调用）、`bs/purge_scale_fixture_test.go`（`purgeScaleTables` `:24` 加表、准备里加一次调用）；`bs/navigation_fixture_test.go` 加 `seedRecentVisits`（照 `seedFavorites`：项目的一行与工作项的一行，`entity_identifier` 可取任意 uuid（不是外键），各随父删除而删）。`bs/runtime_role_test.go` 按目录读表，不改（P3c T3a `07150593` 的提交说明）。
  - navigation 模块自己的已有测试：`nav/adapter/postgres/purge_test.go`（`Tables()` 加 `user_recent_visits`）、`nav/app/delete_rows_test.go`（端口的假实现加第三步、步骤的次序）、`nav/adapter/postgres/delete_rows_test.go`（按范围选语句的表加最近访问）。`nav/app/delete_rows_on_deletion_test.go`、`favorite_cascades_test.go` 不改（新订阅者另起名字）。
  - 删除图：`bs/issue_project_deletion_jobs_test.go` 的 `navigationTable`（`:37-39`）；`bs/project_deletion_test.go` 的 `twoProjectsWith`（`:68-69` 之后一次调用，`twoProjects` 的注释 `:30-39` 一并）与 `bs/workspace_deletion_test.go` 的 `seedProject`（`:249-250` 之后一次调用，它的注释 `:209-219` 一并）；e2e 的 `asynchronousDeletionTable`（`e2e/fixtures/assert/catalog.ts:98-117`，含 `:93-97` 的注释）。
  - 端口：`pj/adapter/postgres/navigation_targets_test.go`（期望值加 `Identifier`）。
  - 接线：`bs/navigation_views_interleaving_test.go` 的 `newNavigationRoute`（`:72-85`）**只在需要时**补 issue 的桥。`navigation.Deps` 是具名字段的结构体，加 `Issues` 不要求调用处跟着改；只有 `navigation.New` 在构造时解引用空的 `Issues` 才需要补。只跑任务的两处 `navigation.New`（`bs/navigation_fixture_test.go:178`、`:197`）不改。T6 核实后二选一、记台账；补的话断言不改。
  - 矩阵：`bs/permission_matrix_test.go` 的 `matrixRows()`（`:219-221`）就地加一项；新行与列表一行的 `prepare` 都在新文件。`permission_matrix_seeded_test.go`、`permission_matrix_seed_test.go`、`prepareMatrix` 不改（记录的路径不带 id，不需要 `workspaceOfRow`；列表一行按格在副本上准备，spec R11）。
  - e2e：`e2e/fixtures/furnish.ts` 的 `furnishIssueRows`（`:78-114`，记两次与注释）。P4、W2、W3、I21 的故事文件不改。
  - 生成物。

  其余断言是回归，不改。编译跟随（假实现加方法、直接调用者补参数）只为编译，不改断言，每处记台账。
- 红的窗口（台账记下）：
  - **(a) T3 → T4**：Go 的删除守卫——表有了，删除任务的第三步与种子还没有。预期红的六个（P3c T3a 实测同一组，归档台账）：`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestProjectDeletionPublishesAndEnqueuesAtomically`、`TestLateDeleteRowsAfterPurge`、`TestPurgeParentCascadesUnprocessedIssueRows`。T3 跑读删除目录的全部测试，另跑调研点过的四个（`TestWorkspaceDeletionSubscriberRollsBackFullCatalog`、`TestDeletionInterleaving25`、`TestAWriteOnAProjectDeletedWhileItWaitsFindsNoProject`、`TestLateDeleteRowsAndPurgeRetry`，re-92 §3 第 5 条），台账逐条列实际红的名字与原因；名单之外的红先停下报告。
  - **(b) T3 → T11**：e2e 的 P4（`p4-archive.spec.ts` 的 “P4 (API)” `:90` 与 “P4 (page)” `:222`）、W2（`w2-landing.spec.ts` 的 “W2 (API)” `:40`）、W3（`w3-workspace-settings.spec.ts` 的 “W3 (API)” 主条 `:32`、`w3-workspace-settings-page.spec.ts` 的第一条 `:14`）、I21（`i21-purge.spec.ts` 的 “I21 (API)” `:289`）——表进了目录、`asynchronousDeletionTable` 认它，夹具要等 T8 的接口与 T11 才造得出行。T3 到 T10 不跑 e2e。
  - **(c) T8 → T10**：`TestThePermissionMatrixCoversEveryOperation`。
  - 清理目录与索引、迁移的计数与名字、动作 ↔ 规则不红（T3 一并改）；事件目录不红（没有新的事件类型，`bs/events_test.go:201-300` 只发 planning 的事件）；契约的自动守卫（路由、401、请求体 400、参数 400、问题响应的头、根路径与标签）随 T8 绿；项目写锁表不红（记录的路径没有 `{project_id}`，矩阵行只用工作区列，`bs/project_write_locks_test.go:150-169`）。每个 Task 写明它验证什么、哪些守卫预期红。
- 检查点：**0**（T2 之后：本机七项 + 分支 CI 全绿，e2e 也绿——窗口打开之前最后一次全绿）；T4 之后一次 Go 两路的中间检查（本机，e2e 在窗口 (b) 里）；**1**（T11）；**2**（T14）。七项用 `sh .superpowers/sdd/P3d-recents/gates.sh <前缀>`。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`；下表每个性质至少一个能让测试失败的变异。两层都在查同一件事时，变异两层一起改，并留只改一层的对照（硬规则 §5）。
- 测试时间：新顶层名只有四个，都在第 0 路（`TestRecentVisitsThroughTheApp` 桶 36、`TestRecentVisitsFollowDeletedWorkItems` 4、`TestRecentVisitWritesInterleave` 33、`TestRecordingARecentVisitTakesItsLocks` 17，`printf %s 名字 | cksum` 取余 100，0–45 是第 0 路）；预估本机 H17 约 3–6 秒、连带约 3–5 秒、交错约 6–12 秒、锁序约 2–4 秒，T10、T12、T13 各记实测，单条接近预估的两倍先报告。别的新测试作已有顶层名的子测试或放进非 bootstrap 包；改名前复算桶、记台账。检查点按作业的 Test 步实测，每路增量 ≤ 90 秒。**第 1 路**（spec §6，主控裁定）：本机停止线——检查点 0 在本机实测的第 1 路是 P3d 的基线（不用 P3c 的数），之后本机第 1 路比它多出 30 秒以上，先停下查原因、报告；CI 的八成线——分支 CI 第 1 路的 bootstrap 超过 960 秒是常设的报告规则、不停，主控向负责人报告并附 runner 的分析（第 0 路非 bootstrap 各包的时间、与本机的对照），只有本机也过了停止线才停。不自行改路数、片数或期限。造好数据后扣锁或写墓碑的测试先 `waitForStartupRuns`（`bs/issue_world_test.go:105`），扣表锁、行锁之前 `waitForFanouts`（`bs/notification_fixture_test.go:182`）；等真实周期任务的窗口盖住约 6 秒。本机两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 为准并记录。

## 性质与变异

| 性质 | 变异 | Task |
|---|---|---|
| `IssuePriority` 的搬移保持行为 | （无变异：`gen-check`、issue 的处理器测试与检查点 0） | T2 |
| CHECK 与部分唯一 | 去掉 `project_id_check`；部分唯一不带 `deleted_at IS NULL`（软删后重建被拒）；部分唯一的列少 `workspace_id`（schema 测试按名字比索引的定义） | T3 |
| 清理 | `Provide` 漏最近访问一步；截止的 `<` 改 `<=` | T3 |
| 动作 | `recent_visit.write` 拒访客（`tableCells` 与矩阵的访客格） | T3、T10 |
| 删除任务 | 项目范围缺第三步；第三步只删 `entity_type = 'project'` 的行；时刻、删除者取 `now()` 与任务自己；第三步不带 `deleted_at IS NULL`（连带早先软删的行被改写成项目的时刻与删除者，清理的时点跟着变） | T4、T12 |
| 领域 | 类型收第三个值；上限不是 20 | T5 |
| 列表的次序 | 去掉 `id DESC` 的打破（关掉索引扫描的读杀得死） | T5 |
| 列表只列自己的、这个工作区的 | 列表的语句不按人限定；不按工作区限定 | T5、T12 |
| 列表的筛选 | 忽略 `entity_type`（筛选的条件恒真） | T5、T12 |
| upsert | 冲突时 `DO NOTHING`（重访不记）；重访改 `id` 或 `created_at`；重访不改 `updated_at` / `updated_by_id`；冲突谓词多一个条件（42P10） | T5、T12 |
| 裁剪 | 留 21 条；只删最旧的一条（Plane 的 `== 20`：23 条只减到 22 条、愈合的 21 条加一条新的只减到 21 条）；前 20 把已软删的行算进去（更新的已删行占位，活的行被提前删）；按 `created_at` 排（D35）；软删而不是物理删（D43）；不按工作区或不按人限定；去掉 `SKIP LOCKED`（存储：被锁的行等到超时；交错：不等的一格等住，死锁的一格 40P01、其一 500） | T5、T13 |
| 咨询锁 | 不取咨询锁（后果：同一人并发越过 20）——确定的杀法是 T10 锁序测试的第三段（持有者扣着咨询锁时请求必须在等，变异体直接 204）与 T7 的调用序列；T13 的“恰 20 条”是正向的结果，不算它的杀法（不取咨询锁时两个请求同时放行，谁先提交是竞态）。键少了人或少了工作区（存储测试“别人的、别的工作区的不等”） | T5、T7、T10 |
| 裁剪与清理不相交 | 裁剪不带 `deleted_at IS NULL`（删掉清理该删的）；清理不带截止 | T5 |
| 连带的语句 | 只删一个人的；不按工作区限定；时刻、写者不取参数的 | T5、T9 |
| 项目的标识 | `NavigationTargets` 不读 `identifier`（空串） | T6 |
| issue 的端口 | 不按授权过滤；不调 `HiddenFrom`（只看自己的访客读到别人的）；`InTriage` 恒假；负责人含已删的；地点不读已归档的；摘要读已删的工作项 | T6 |
| 可见与列出 | 列表照列 Triage；列表藏已归档（收藏的谓词漏进来）；记录拒已归档；项目只看未删、不看授权（不是成员的工作区管理员能记公开项目） | T7、T12 |
| 先过滤后取 20 | 先取 20 再过滤——T7 的假存储交出 21 条未删行、藏起的一条排在前 20 里，变异体只列出 19 条（不超过 20 条时两种次序答得一样，杀不死） | T7 |
| 拒绝次序 | 记录：`entity_type` 的检查挪到读目标之后（不存在的目标 + 坏类型应答 422）；列：挪到工作区之后（看不到的工作区 + 坏类型应答 422） | T7、T12 |
| 一个 404 | 看不到答 403 或别的码；记录对不是成员的工作区答 `workspace.not_found` | T7、T8、T12 |
| 锁序与锁下重判 | 不 `FOR SHARE` 工作区；不 `FOR SHARE` 项目；工作项删除的两层（锁下地点的重读与可见的重读，都看 `deleted_at IS NULL`）一起去掉，留只去一层的对照（仍 404）——杀它的是 T7 的“锁下重读看到已删”与 T13 的“删除先提交 × 记录”（活行）；项目删除的两层（项目 S 读不到 + 重读）一起去掉，留只去一层的对照；时钟在咨询锁之前读 | T7、T10、T13 |
| 记录的回答 | 重访答 200 或 201 | T8 |
| 契约 | 去掉记录的一个码（`CheckResponse` 拒绝）；逐码表漏答一个码（`apitest.Main`） | T8 |
| 连带订阅者 | 不注册；畸形的载荷被接受（逐项）；执行者为空被拒；只删执行者自己的 | T9、T12 |
| 矩阵 | 去掉一行；访客的记录答 404；不准备的“不作为成员”一行答 204；列表不按可见过滤（管理员、成员的格列出 `acme/public`） | T10 |
| e2e 夹具 | 夹具不记（I21 的 `parentIDs` 与五条 `deletedWithIt` 红）；删除任务缺第三步（六条红） | T11 |
| GET 没有副作用 | 列表写行（例如列表里记一次） | T12、T14 |
| 上限数藏起的行 | 上限只数看得到的 | T12 |

## Tasks

### T1 基线与文档（主控）
提交 spec、plan、预检 re-92（`docs/reviews/re-92-M7P3d预检.md`，含 §6 的处理），以及随预检的处理已写进工作区的 P3c 阶段记录与设计 §14 的改动（主控审过再提交）。P3c 阶段记录：状态行改为已完成；§9 回填修复轮之后的限定复审（fable，`2fafe043..49aa9fb2`：Ready to merge，三条次要在 `d2511e56` 修，`rr-lint-go`、`rr-keywords`、`rr-doc-links` 过）、终检 `final.sh fin`（`d2511e56`，七项全 0；bootstrap 第 0 路 453.2、第 1 路 655.4 秒；e2e 493 passed）、最终分支 CI 44945103629（server (0) Test 650 / bootstrap 465.5，server (1) Test 967 / bootstrap 938.8，期限的 78.2%，web、e2e 两片绿）、`--no-ff` 合入 `9d26b1c3`（父 `1168fd25`、`d2511e56`）、main CI 44947828240（server (1) bootstrap 752.5，62.7%，全绿）、归档 `.superpowers/sdd/M7-P3c-integration-20261122`（3507 个文件，未知 0；worktree、本地与远端分支已删，`cleanup.log`）。数字取自归档台账的末行；§5、§6 的表加最终分支 CI 与 main CI 两行（本机 `fin` 453.2 / 655.4 秒），写明同一份代码 78.2% 与 62.7% 之差在 runner。设计 §14 的 P3c 一行改为已完成（noff `9d26b1c3`、main CI 44947828240、阶段记录），加 P3d 一行（进行中）。

### T2 `IssuePriority` 的纯搬移（R8）→ **检查点 0**
文件：`api/common.yaml`（加 `IssuePriority`，说明照原文）、`api/modules/issue.yaml`（删 schema `:2196-2199`，7 处 `$ref` 改指 `../common.yaml#/components/schemas/IssuePriority`）、`iss/adapter/http/issues.go:95`、`:105`、`iss/adapter/http/intake_list.go:45`（`gen.IssuePriority` → `apigen.IssuePriority`）；`make gen`（issue 的 `server.gen.go`、`apigen/components.gen.go`、`api/dist`、`schema.gen.ts`）。步骤：搬、生成、`gen-check`、`lint-go`、issue 的处理器测试、`lint-web`、`knip`、`test-web`；`api/dist` 里的名字不变（打包保留名字，同 `LogoProps`），网页的 `IssuePriority` 类型照旧。断言不改。**检查点 0**：本机七项、推送、分支 CI 全绿（e2e 也绿），台账记两路的本机与 CI 时间作 P3d 的基线（第 1 路的本机停止线由这一次的本机数算）。

### T3 迁移、动作与清理
文件：`mig/sql/00054_navigation_user_recent_visits.sql`（含 Down）与新 `mig/navigation_recent_visits_schema_test.go`（`TestUserRecentVisitsSchema`：spec §2.3 的列与名字逐条；两条 CHECK 各一个只破它的反例；部分唯一在软删之后能重建；一条真实的 `INSERT … ON CONFLICT (user_id, workspace_id, entity_type, entity_identifier) WHERE deleted_at IS NULL DO UPDATE` 推得出它；外键的连带与 `SET NULL`）；迁移的三处计数与名单（允许清单）；`server/sqlc.yaml`（navigation 的 schema 加 `00054`，`:210-211` 之后）；`deploy/runtime-grants.sql:29`；动作 `recent_visit.read`、`recent_visit.write`（`nav/domain/actions.go` 的常量与 `Actions()`、`server/internal/modules/access/domain/rules.go` 两行（注释照设计 §3.10：三种角色、只对自己的、目标须看得到在用例）、`tableCells`）；清理一步（`nav/adapter/postgres/queries/purge.sql` 的 `PurgeUserRecentVisits`、`purge.go`、`nav/module.go` 的 `Provide` 在 `user_favorites` 之后、包注释）、`purge_test.go`；清理守卫的名单与种子（允许清单）；e2e 的 `asynchronousDeletionTable`。部分唯一的名字照 spec（`…_entity_key`，63 字节的上限），迁移注释写明。测试与变异见上表“CHECK 与部分唯一”“清理”。检查：迁移包、navigation 与 access 各包、bootstrap 的清理目录与运行角色清理、`TestSQLCSchemaScope`、`lint-go`、`gen-check`，e2e 的类型检查、lint 与格式（`catalog.ts`）。红窗口 (a)、(b) 起；台账列 (a) 实际红的名字。

### T4 删除任务的第三步 → Go 两路的中间检查
文件：`nav/app/ports.go`（新端口 `RecentVisitDeletions{DeleteProjectRecentVisits(ctx, RowBatch)}`，文档照 `FavoriteDeletions`）、`nav/app/delete_rows.go`（`DeleteRowsDeps` 加一项，`steps` 的项目范围加第三步，文档）、`nav/adapter/postgres/queries/delete_rows.sql`（`DeleteProjectRecentVisits`：`project_id = P`、未删、`id > after`、`ORDER BY id LIMIT n FOR NO KEY UPDATE`，照 `DeleteProjectFavorites`）、`delete_rows.go`、`nav/module.go`（`NewDeleteRows` 的实参）、`nav/module_jobs.go` 的注释；`navigationTable`；Go 删除守卫的种子调用（允许清单）；navigation 的两个删除测试（允许清单）。测试：项目范围三步的次序与各自从头的游标、共用预算；一批只删这个项目的两种行、按 `id` 升序加锁、时刻与删除者取参数；一条早先已软删的行（别的时刻、别的删除者，如连带写下的）不被改写（spec §2.6）；工作区范围不变。变异见上表“删除任务”。红窗口 (a) 在此关闭。之后本机跑 Go 两路（e2e 预期红），台账记时间。

### T5 领域与存储
领域（新文件 `nav/domain/recent_visit.go` + 测试）：`RecentVisitKind`（`issue`、`project`）、`CheckRecentVisitKind`（不是两值之一 → `validation_failed`、`entity_type` 的 `invalid_format`）、`RecentVisit`、上限常量 `RecentVisitCap = 20`；`nav/domain/errors.go` 加 `ErrIssueNotFound`（码与文案同 `iss/domain/errors.go:44`）。存储（新 `nav/adapter/postgres/queries/recent_visits.sql`、`recent_visits.go` + 测试；端口声明在新 `nav/app/recent_ports.go`）：
- `RecentVisits(user, workspace, kind *RecentVisitKind)`：本人在这个工作区的未删行（有 `kind` 时只那一种）、`visited_at DESC, id DESC`，不加锁、不限条数（上限限住）。
- `LockRecentVisits(user, workspace)`：spec §2.8 的那一条语句，键在 SQL 里由 `hashtextextended` 推出（re-92 已实测 sqlc 收下）。
- `RecordRecentVisit(v, by)`：spec §2.3 的 upsert。
- `TrimRecentVisits(user, workspace, keep)`：spec §2.8 的语句，答删了几条。
- `DeleteIssueRecentVisits(workspace, issue, at, by *uuid)`：连带，按 `id` 升序 `FOR NO KEY UPDATE`。

测试：
- 列表：次序（同一时刻两条，事务里 `SET LOCAL enable_indexscan = off` 等再读一遍，硬规则 §5）；只读本人的、只读这个工作区的（两个人访问同一目标、同一人在两个工作区）；`kind` 只读那一种。
- 新记与重访的列。
- 裁剪（数据用 SQL 种）：23 条未删行再记一次 → 恰 20 条；21 条（回滚留下的样子）再记一个新目标 → 恰 20 条（瞬时第 21 条愈合）；前 20 只数未删的（几条更新的已删行之下仍留 20 条活的）；物理删（连已删的也数不到它）；只动本人本工作区；一行被另一个事务 `FOR UPDATE` 扣住时裁剪在 `lock_timeout` 之内答完且留下它。
- 咨询锁：同一人同一工作区的两个事务串行（第二个等，`pgtest.WaitForLockWait`——存储测试没有应用在跑，它可用；bootstrap 不用它，见 T10）；别人的、同一人别的工作区的在 `lock_timeout` 之内取到。
- 连带只动这个工作区这个工作项的、所有人的。
- 新测试 `TestTrimAndPurgeOfRecentVisitsTakeDisjointRows`（四种行：第 20 条之后的未删行、截止之前删的、截止之后删的、前 20 条；先裁后清与先清后裁各删恰自己的一组）。

变异见上表“列表的次序”“列表只列自己的、这个工作区的”“列表的筛选”“upsert”“裁剪”“咨询锁”“裁剪与清理不相交”“连带的语句”。

### T6 端口
- project（R4）：`pj/app/navigation_ports.go` 与 `nav/app/favorite_ports.go` 的两个结构同一位置加 `Identifier string`（文档改为收藏与最近访问共用）；`pj/adapter/postgres/queries/navigation_ports.sql:18-24` 多选 `identifier`、`navigation_ports.go:48-62` 填它；`navigation_targets_test.go` 的期望值（允许清单）。`pj/module.go` 不改。
- issue（R3）：新 `iss/module_navigation_issues.go`（接口 `NavigationIssues`：`NavigationIssuePlaces`、`NavigationIssues`，值的别名，文档照 `module_issue_refs.go`）、`iss/app/navigation_issues.go`（读端口在此声明；`HiddenFrom` 与 `InTriage` 在此判，照 `issue_refs.go:45-65`）、`iss/adapter/postgres/queries/navigation_issues.sql`（地点：未删、含已归档、按 id；摘要：未删、含已归档，项目、编号、名字、状态、优先级、创建者、未删负责人的数组（同 `issues.sql:89-90`）、未删条目行的 `status`（`LEFT JOIN intake_issues`，同 `issues.sql:99`），按 `id`）、`iss/adapter/postgres/navigation_issues.go`；`iss/module.go` 的 `Provided` 与 `Provide` 各一行。测试（app 与 adapter）：spec §4“端口”一条的每格。
- navigation：`nav/app/recent_ports.go` 加 `Issues` 端口与它的值（字段与 issue 的相同，bootstrap 按类型转换）；`nav/module.go` 的 `Deps.Issues`；`bs/navigation_ports.go` 的桥 `navigationIssues`（授权照 `navigationReadable` 逐个转换）；`bs/app_modules.go` 的 `navigationModule` 传入。核实 `navigation.New` 是否在构造时解引用 `d.Issues`，按允许清单“接线”一条决定 `newNavigationRoute` 改不改，记台账。
- 变异见上表“项目的标识”“issue 的端口”。

### T7 用例
新文件（都在 `nav/app/`）：`recent_sight.go`（`recentTargets{Readable, Projects, Issues}`；一次读 `Readable`、一次工作项、一次项目；谓词 `sees`（记录）与 `shows`（列出 = `sees` ∧ 不在收集箱 ∧ 项目读得到），spec §2.5）、`recent_locks.go`（`RecentVisitLocks`：预检与 `run`，spec §2.7、§2.8；复用 `decideWorkspace`、`allow`）、`record_recent_visit.go`（`RecordRecentVisit`）、`list_recent_visits.go`（`ListRecentVisits`，`ReadTx` 的快照）+ 各自的测试与假实现（新文件 `recent_fakes_test.go`）。时钟在咨询锁之后读一次；新记与重访的时刻、写者照 spec §2.5；先过滤后取 20。单测钉：两条拒绝次序与交叉格；锁的调用序列（工作区 S → 授权 → 项目 S → 重读 → 咨询锁 → 时钟 → upsert → 裁剪；少了咨询锁这一步即失败，“不取咨询锁”的两个确定杀法之一）；锁下重读看到已删、看不到 → 404 且不取咨询锁、不读时钟（工作项的两层——地点的重读与可见的重读——变异一起去掉，只去一层的对照仍 404）；可见谓词逐格（已归档、Triage、只看自己、不是成员的工作区管理员）；先过滤后取 20：假存储交出 21 条未删行、藏起的一条排在前 20 里，答 20 条、含第 21 条。变异见上表“可见与列出”“先过滤后取 20”“拒绝次序”“一个 404”“锁序与锁下重判”。

### T8 契约、处理器与接线
`api/modules/navigation.yaml`（两个操作、六个 schema，说明写谁能用、拒绝次序、锁、写的时刻与上限，文件头一句）、根 `api/openapi.yaml`（两条路径、标签说明）；`make gen`（之后 `lint-web`、`knip`、`test-web`）；处理器（新 `nav/adapter/http/recent_visits.go`：错误一律交 `api.Errors.Write`；回答把 `issue` 在项目的行上给 `null`）、`handler.go` 的 `UseCases` 两项；`nav/module.go` 的 `New` 与包注释；`newNavigationRoute` 的桥（允许清单）。测试（新 `recent_visits_test.go`）：每个声明的码一个失败；204；回答全键（`issue: null`、`logo_props: {}`）；请求体到用例的映射（`entity_type` 原样交给用例）；结构 400；查询的 `entity_type` 原样交给用例。红窗口 (c) 起；自动跟随的契约守卫照常绿。变异见上表“记录的回答”“契约”。

### T9 工作项删除的连带
新文件 `nav/app/recent_cascades.go`（`NewForgetDeletedIssueVisits`：名字 `navigation_forget_deleted_issue_visits`、`Types: ["issue.deleted"]`；先整体校验，spec §2.6；经 `RecentVisitCascades` 端口写）+ 测试（每种畸形各一格被拒且没写；执行者为空照写、写者为空；所有人的、只这个工作项的）；`nav/module_events.go` 的 `Subscribers` 加它、文档。事件目录与 `bs/events.go` 不改（类型已声明，`Subscribers` 的签名不变）。变异见上表“连带订阅者”。

### T10 守卫
权限矩阵（新文件 `bs/permission_matrix_recent_visits_test.go`，`recentVisitMatrixRows()`；`matrixRows()` 就地加一项）：列表一行与记录三行，格与理由照 spec §4“守卫”。列表一行是 `write` 加 `prepare`（spec R11，先例 `bs/permission_matrix_issue_lists_test.go:15`）：`prepare` 用 SQL 往格的副本里插一条 `s.account(accountOf(c))` 对 `acme/public` 的项目访问（照 `inDone` 开副本的池，`bs/permission_matrix_issue_archive_test.go:61-72`），`check` 断言访客恰列出它、管理员与成员答空。记录的请求体按列取目标：`projectOf(c)` 的项目（`s.project`），或它的 `matrixIssue`（工作区管理员建的，`s.issue(projectOf(c), matrixIssue)`，`bs/permission_matrix_issue_columns_test.go:43-54`；`gone/project` 也有）。

锁序测试 `TestRecordingARecentVisitTakesItsLocks`（新文件 `bs/navigation_recent_visit_locks_test.go`，照 `TestFavoriteWritesTakeTheirTargetsLocks` 的骨架，`bs/navigation_favorite_locks_test.go:115-264`；spec §4 的四段，两种目标）：`requestBlocked` 认第一、二段的行等待；`requestWrittenTables`（`bs/request_locks_test.go:303`）断言等待时没写任何表，`requestRowLock`（`bs/request_row_locks_test.go:14`）断言持哪些 S。**第三段**（咨询锁）：持有者在 `postgres.NewTxManager(...).WithinTx` 里经 `navigationpg` 存储的 `LockRecentVisits` 取锁，`postgres.Tx(ctx)`（`server/internal/platform/postgres/tx.go:28`）给出它的连接；等待用 `pgtest.WaitForStatementLockWaitBehind(t, pool, 持有者的 PgConn, 时限, "LockRecentVisits")`，再用 `readRequestLock` 认出是这个请求在 `LockRecentVisits` 里等（`locktype` 为 `advisory`）。不用 `pgtest.WaitForLockWait`（数任何等锁的后端，应用的 River 任务也算）与 `requestBlocked`（只认行与表的等待，`bs/request_locks_test.go:141-150`）。第三段是“不取咨询锁”的确定杀法。红窗口 (c) 在此关闭。变异见上表“矩阵”“咨询锁”“锁序与锁下重判”。

### T11 e2e 删除夹具 → **检查点 1**
新 `e2e/fixtures/recent-visits.ts`（`recordVisit(api, token, body)` 断言 204；之后 T14 再加调用包与库的读）；`furnishIssueRows` 在收藏之后由调用者记 `{entity_type: "project", entity_id: projectId}` 与 `{entity_type: "issue", entity_id: fix.id}`，注释写明两行各证明什么（项目一行随项目、工作区删除；工作项一行证明第三步不只删项目的行）与前置条件（调用者是项目里看得到 `fix` 的成员，现有调用者都是；每人每工作区远在 20 条以内）。`furnishWorkspace` 与 W3 自己的 `furnish` 经 `furnishIssueRows` 带上，不改。验证 P4（两条）、W2、W3（两个文件）、I21。红窗口 (b) 在此关闭。变异见上表“e2e 夹具”。**检查点 1**：七项本地门禁、分支 CI、两路各自的增量与第 1 路的余量。

### T12 组合测试
H17 `TestRecentVisitsThroughTheApp`（桶 36）与 `TestRecentVisitsFollowDeletedWorkItems`（桶 4），新文件（照 P3c 按主题分文件：`navigation_recent_visits_world_test.go` 的世界与帮手、`navigation_recent_visits_test.go`、`navigation_recent_visits_sight_test.go`、`navigation_recent_visits_follow_test.go`，各约 400 行以内）；格照 spec §4 的两条。种子写明：创建者不是删除者；三层子树；两个人访问同一工作项；同一人在第二个工作区也有行（只列这个工作区的）；一条访问最早而建立不最早的行（上限的区分）；一个藏起的行在上限里；工作区管理员兼项目访客的两种开关；Triage 条目的待处理、接受、拒绝三种；删除窗口那个项目开着视图并有一条视图（`holdNavigationDeletion` 扣的是它，`bs/navigation_fixture_test.go:98-105`）；先删工作项、再删它的项目的一组（第三步不改写连带写下的时刻与删除者）。GET 没有副作用的快照逐列比较。变异见上表“可见与列出”“一个 404”“连带订阅者”“GET 没有副作用”“上限数藏起的行”“upsert”“列表只列自己的、这个工作区的”“列表的筛选”“删除任务”。

### T13 交错
`TestRecentVisitWritesInterleave`（桶 33，新文件 `bs/navigation_recent_visits_interleaving_test.go`，必要时再分一个文件）：spec §4“交错”的各格；配对共用准备好的库、分组；`waitForStartupRuns`、`waitForFanouts`。各组的构造：
- 记录 × 删除、成员移除、归档：两个写都经真实的 app，夹具扣住两个写都要的行（P3c T12 的做法）；成员移除两种次序各有自己的断言（spec §4）。
- 恰 20 条、同一目标两次：持有者事务经 `navigationpg` 存储的 `LockRecentVisits` 扣着咨询锁（同 T10 第三段的持有者），两个请求各用自己的请求探针（`w.requests.observe`、`readRequestLock`）都看到等在 `LockRecentVisits`，再放开。正向的结果，不当作“不取咨询锁”的杀法。
- 裁剪 × 扣着的一行：测试的 SQL 事务经存储的 `DeleteIssueRecentVisits` 软删那一行、开着；提交一格、回滚一格（回滚格：库里 21 条、列表 20 条、下一次记录后 20 条）。
- 裁剪 × 子树删除：子树两个工作项都有 bob 的通知（照先例由指派造），扣住第二个工作项的一条，等删除停在它后面（`pgtest.WaitForStatementLockWaitBehind`，持有者的连接、语句 `DeleteIssueNotifications`，同先例 `bs/notification_mark_all_read_interleaving_test.go:146-148`）；记录的目标在别的项目。不等的一格：第 20 条之后恰一条（第一个工作项的访问；补足 20 条的行可用 SQL 种，`entity_identifier` 不是外键），记录用 `answeredUnlessWaiting`（语句 `TrimRecentVisits`）。死锁的一格：SQL 种成记录之后 22 条，第 20 条之后两条分属两个工作项、行的 `id` 与子树次序相反；先放开、收齐两个回答再断言（两个都 204、本人 20 条、记录没等），变异体的 40P01 才看得见。两格与先例共用世界的构造；死锁格比不等格只多种一条（第二个工作项的访问）、并指定两条的 `id`，本机约多 1–2 秒。

变异见上表“裁剪”（去掉 `SKIP LOCKED`：不等的一格与死锁的一格各杀一次）、“锁序与锁下重判”（工作项删除的两层一起去掉）。

### T14 e2e 接口故事 → **检查点 2**
`e2e/stories/collaboration/h17-recent-visits.spec.ts`：照 `research/plane-e2e.md` §8 的提纲 A–J（记录、重访、筛选、此刻的摘要、读不写、看不到、访客、离开、上限、删除），两轮（PAT、会话），每轮 `slugFor(testInfo, credential)` 自己的工作区；头注释照 H15 写明哪些归 bootstrap 与矩阵、页面归 P7。`fixtures/recent-visits.ts` 加 `recentVisitCalls(api)`（成功 `{status, visits}`，失败 `answer(...)`）与只读的 `recentVisitRows(db, userId, workspaceId)`（含已删，`{id, type, identifier, project, created, visited, deleted, writer, stamped}`）。I 步依赖 B 步之后 bob 没有别的记录，注释写明。**检查点 2**：七项本地门禁、两路各自的增量、e2e 两片的时间。

### T15 文档与交接（主控）
plane-diff：表一行（`user_recent_visits`：`entity_type` CHECK 两值、`project_id` 非空、部分唯一、审计列、没有 `entity_name`）；行为一行（显式记录、204、只记两种（D37）、按访问时刻（D35）、咨询锁守住上限（D36）、淘汰物理删（D43）、读时过滤更宽（D39：只看自己的访客、私有化、Triage）、已归档照列、工作项删除软删且每条删除路径都清（Plane 单个删除硬删、批量不清，D44）、先过滤后取 20、工作区访客能用）。交接（spec §5 的五条）。硬规则 §4：`user_recent_visits` 的写者与锁序（spec §2.8 的表、持咨询锁者从不等行锁、裁剪 `SKIP LOCKED` 的理由照 spec §2.8 改正后的那条环——多出两条以上时裁剪与连带互等，不是 DP-12 经别的项目的那条）、P2a 一段的“三条查询带 `Intake`”加 `NavigationIssueRows`（或实施时的名字）；§5 加实施中新发现的坑。设计 §3.9（“Plane 相同”改为“Plane 单个删除时硬删、批量删除不清”；实现指针）、§4.2 的 `user_recent_visits` 一行（`project_id` 非空、CHECK、三条部分索引、审计列）、§4.3（清理计数到 37）、§5 的最近访问一行（204、集合型、`entity_type` 可选）、§12 的 P3d 一行加 spec 链接、§14；阶段记录 `reviews/P3d-recents-review.md`。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 一个 404：看不到、不存在、已删除、别的工作区、只看自己时别人的工作项逐值相同；工作区由目标推出，记录不透露工作区的存在。
- 锁序与锁下重判；持咨询锁的事务从不等行锁（upsert 不等的理由、裁剪的 `SKIP LOCKED`、外键对账户行的 `KEY SHARE`）；每个写者的锁序。
- 可见谓词只有一个：工作项的判定在 issue（`HiddenFrom`、`InTriage`），navigation 不复制；列出与记录的差别只在 Triage。
- 上限：数未删的全部行（前 20 不数已删的）、先过滤后取 20、删到恰 20（不只最旧的一条）、物理删、与清理不相交；列表只列本人在这个工作区的。
- 删除图：`issue.deleted` 覆盖每条删除路径；删除任务的第三步（不改写已软删的行）；归档不动。
- 测试的构造：bootstrap 里等咨询锁用按持有者与语句的探针（不用 `WaitForLockWait`）；app 里扣不住的访问行用测试的 SQL 事务经存储的方法扣；每个变异由一个确定的格杀，不靠竞态。
- 模块边界：navigation 只经端口拿工作项、项目与可读项目；`IssuePriority` 的搬移保持行为。
