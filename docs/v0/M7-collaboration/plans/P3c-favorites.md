# M7 / P3c plan：收藏与三种归档事件

> spec：[P3c spec](../specs/P3c-favorites.md)（预检 [re-91](../../../reviews/re-91-M7P3c预检.md) 修正后通过）。基线 main `1168fd25`。worktree `.codex/worktress/codex/m7-p3c`，分支 `codex/m7-p3c`。台账 `.superpowers/sdd/P3c-favorites/progress.md`（ignored）。16 个 Task（T3、T8 各分 a、b；领域并存储、端口并读与建，re-91 I4）。

## Global Constraints

- spec §2 是合同，§3 的 R1–R14 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；每个声明的码有处理器测试回答；契约、处理器与两个新码的文案同一 Task（`error-messages.test.ts` 要求键恰等于契约的码）。
- 模块之间只经端口；navigation 的 SQL 只经自己的 sqlc（不能联表 `projects`、`cycles`、`modules`、`workspaces`）；新代码进新文件，手写文件约 400 行以内。已近上限、不再加长的：`bs/ports.go` 378、`project/module.go` 371、`issue/app/ports.go` 395、`bs/permission_matrix_test.go` 451、`bs/permission_matrix_seeded_test.go` 428（例外：允许清单里写明的三处，净增不超过 2 行）、`bs/permission_matrix_seed_test.go` 390（例外：一次调用）。新桥进 `bs/navigation_ports.go`，project 的新端口进新文件（照 `module_notification_names.go`），矩阵的新行与种子函数进新文件。
- 端口的 bootstrap 转换在加它的同一个 Task 里接进 `navigation.Deps`（`lint-go` 开着 `unused`，re-91 m5）。
- `bootstrap/app.go` 先拆（T2），之后的接线加在拆出的文件里。
- 允许改的已有测试（改之前在台账逐条写明，只照这里写的改法）：
  - 迁移：`server/migrations/trigram_schema_test.go`（迁移数 52 → 53、回滚循环的上界、表名名单加 `user_favorites`）、`purge_indexes_test.go`（35 → 36）、`navigation_schema_test.go` 的 `navigationNames`（加新表的名字）；新 schema 测试。
  - 动作：`access/domain/rules_test.go` 的 `tableCells`。
  - 清理：`bs/purge_catalog_test.go`（35 → 36、`providedPurgeTables`）、`bs/purge_runtime_test.go`（navigation 一组加表、变老的表）、`bs/purge_scale_fixture_test.go`、`bs/issue_retention_sets_test.go`；`deploy/runtime-grants.sql` 与 `bs/runtime_role_test.go` 的表。
  - navigation 模块自己的已有测试（re-91 I3）：`nav/app/delete_rows_test.go`（端口的假实现与按表的步骤）、`nav/adapter/postgres/delete_rows_test.go`（按范围选语句的表加收藏）、`nav/adapter/postgres/purge_test.go`（`Tables()` 加 `user_favorites`）、`nav/app/delete_view_test.go`（调用序列多一次连带）。`nav/app/delete_rows_on_deletion_test.go` 不改（连带订阅者另起名字）。
  - 删除图：`bs/issue_project_deletion_jobs_test.go` 的 `navigationTable` 与订阅者构造的调用点；Go 删除守卫的夹具（`bs/project_deletion_test.go`、`bs/workspace_deletion_test.go` 的种子用 SQL 加项目下、工作区下的收藏；R12：`TestDeletingAProjectLeavesNoUndeletedRowUnderIt` 在归档之后再种）；e2e 的 `asynchronousDeletionTable`。
  - 事件（R13）：`bs/events_test.go` 的 `TestIssueEventDirectory`（project 目录加 `project.archived`）、`TestPlanningEventDirectory`（目录四种；T8a 先只改目录，T8b 改为 issue 与 navigation 两组订阅者、畸形的删除与归档事件各被拒绝）；planning、project 归档用例的单测（调用序列多一次 `Publish`，断言载荷与时刻）。
  - 矩阵：新行（只用工作区列）与种子；`permission_matrix_seeded_test.go` 的三处（`seeded` 加一个字段、`newSeeded` 的初始化、`workspaceOfRow` 的名单——字段与初始化并进已有的行）；`permission_matrix_seed_test.go` 的 `prepareMatrix` 加一次调用（种子函数进新文件）；`matrixRows()` 的拼接一项。
  - e2e：`furnish.ts`（`furnishIssueRows` 建收藏、`PlanningRows` 带回视图 id、`furnishWorkspace` 建文件夹与里面的工作区视图收藏）、W3 自己的 `furnish`（同上）、`assert/project.ts` 的 `expectProjectDeleted` 加 `removedBefore`、P4 两条传 `["user_favorites"]`（R12）。`furnishIssueRows` 的 11 个调用者都是工作区管理员或成员、且是项目成员（re-91 m7）；新增调用照此。
  - 生成物。

  其余断言是回归，不改。
- 红的窗口（台账记下）：
  - (a) T3a → T3b：Go 的 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 与用 `parentAsyncTable` 的几条（表有了，删除任务与种子还没有；T3a 在台账列出实际红的测试名）。
  - (b) T3a → T10：e2e 的 P4（`p4-archive.spec.ts` 两条）、W2（`w2-landing.spec.ts`）、W3（`w3-workspace-settings.spec.ts`、`w3-workspace-settings-page.spec.ts`）、I21（`i21-purge.spec.ts`）——表有了，夹具只能经 T7 的接口造行。
  - (c) T7 → T9：`TestThePermissionMatrixCoversEveryOperation`。
  - 清理守卫、动作 ↔ 规则不红（T3a 一并加）；事件目录不红（加类型与改测试同一 Task）。每个 Task 写明它验证什么、哪些守卫预期红。
- 检查点：**0**（T2 之后：本机两路全量 + 分支 CI 全绿，e2e 此时也绿）；T3b 之后一次 Go 两路的中间检查（本机；e2e 预期红）；**1**（T10）；**2**（T13）。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`；下表每个性质至少一个能让测试失败的变异。
- 测试时间：新顶层名只有四个，都在第 0 路（`TestFavoritesOfEveryKindThroughTheApp` 桶 6、`TestFavoritesFollowTheirTargets` 13、`TestFavoriteWritesInterleaveWithArchives` 2、`TestFavoriteWritesTakeTheirTargetsLocks` 37）；预估本机 H15 约 5–8 秒、H16 约 4–6 秒、交错约 15–25 秒、锁序约 2–3 秒，T11、T12 各记实测，单条接近预估的两倍先报告。别的新测试作已有顶层名的子测试或放进非 bootstrap 包。改名前用 `printf %s 名字 | cksum` 复算、记进台账。检查点按作业的 Test 步实测，每路 ≤ 90 秒，第 1 路的 bootstrap 包到 `go test` 期限的八成先报告。造好数据后扣锁或写墓碑的测试先 `waitForStartupRuns`；等真实周期任务的窗口要盖住约 6 秒。本机两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 为准并记录。

## 性质与变异

| 性质 | 变异 | Task |
|---|---|---|
| `app.go` 的拆分保持行为 | （无变异：两路全量与分支 CI；台账记六条次序的去处） | T2 |
| CHECK 与部分唯一 | 去掉文件夹 ⇔ 列的搭配；部分唯一不带 `deleted_at IS NULL`（软删后重建被拒）；文件夹名区分大小写 | T3a |
| 清理 | 清理表漏收藏 | T3a |
| 删除任务 | 收藏一步缺失；工作区那一步删了项目里的收藏；时刻、删除者取 `now()` 与任务自己 | T3b |
| 字段搭配与文件夹名 | 文件夹收 `entity_id`；别的收 `name`；名字不去空白；文件夹收父 | T4 |
| 再收藏的冲突目标 | `ON CONFLICT` 不写冲突目标（文件夹名的冲突被吞） | T4 |
| 次序 | 新建不取本人最大值（取全工作区的）；列表不按 `id` 打破 | T4、T5、T11 |
| 可见与合格 | 不查 `Readable`（不是成员的工作区管理员能收藏项目）；不查归档；不查 `HiddenFrom`；不查同一工作区 | T5、T11 |
| 读时过滤已归档 | 列表不滤已归档的目标（SQL 留下的活行） | T5、T11 |
| 开关不看 | 关着 `cycle_view` / `issue_views_view` 时拒收藏或不列 | T5、T11 |
| 一个 422 | 已归档另答别的码；看不到答 404；目标与父同时不合格只带一个字段 | T5 |
| 再收藏 | 已有时再插一行或答 201；已有时改了父 | T5、T11 |
| 父规则与活的子行 | 父可以是别人的文件夹、非文件夹、已删的文件夹；建不对父取 `FOR SHARE`（删文件夹 × 在它里面新建留下活的子行） | T5、T6、T12 |
| 一个 404 | 目标已看不到的行可改、删；别人的行答 403 | T6、T11 |
| 无改动 | 无改动仍写、仍动 `updated_at` | T6 |
| 文件夹名 409 | 改名撞上答 422 或 200；改成自己（大小写不同）答 409 | T4、T6 |
| 删文件夹 | 子项不删；子项的时刻不同 | T6、T11 |
| 收藏行的全序 | 改按 `id` 升序一条语句锁本行与父（删文件夹 × 改它里面的子行带同一个 `parent_id` 死锁） | T6、T12 |
| 拒绝次序 | 建：体的校验挪到读之后（看不到的工作区 + 坏体应答 422）；改：体的校验挪到读行之前（别人的行 + 坏体应答 404） | T5、T6 |
| 锁下重判 | 去掉锁下重读 | T6、T12 |
| 锁序 | 不 `FOR SHARE` 工作区；不 `FOR SHARE` 项目；视图目标不 `FOR SHARE` 视图 | T5、T9 |
| 归档事件 | 不发布；恢复也发；迭代再归档也发；项目再归档不发；`OccurredAt` 与写下的 `archived_at` 不同；载荷缺项目或工作区 | T8a |
| 连带 | 只删当前用户的；项目归档漏它的视图收藏；畸形事件被接受（按类型）；时刻、写者不取事件的 | T8b、T11 |
| `deleteView` 的连带 | 不删收藏；另读一次时钟或写者不同 | T8b、T11 |
| 立即不可见 | 读不过滤已删项目（R14，扣住任务时） | T11 |
| 矩阵 | 去掉一行；访客答 200；带 `prepare` 的项目目标不加入项目也答 201 | T9 |

## Tasks

### T1 基线与文档（主控）
spec、plan、预检 re-91；P3b 阶段记录 §9 回填集成；设计 §14 的 P3b 一行改为已完成、加 P3c 一行；设计 §12 的 P3c 一行拆成 P3c、P3d（R1），§11 里关闭阶段写 P3c 的格（M1-P3、M1-P2）按再拆分好。

### T2 拆 `bootstrap/app.go`（R2）→ **检查点 0**
保持行为的提取：四个文件 + 订阅者清单进 `events.go`（新函数，签名写明，T8b 给它加 navigation 的端口）；R2 的六条次序与每个失败点的清理不变；`go build`、`go vet`、`lint-go`；台账写每段的去处、六条次序各在哪里。**检查点 0**：本机两路全量、推送、分支 CI 全绿（e2e 也绿）。

### T3a 迁移、动作与清理
`00053` 与 schema 测试（spec §2.3 的名字逐条）、迁移的计数与名单；sqlc 条目、运行角色；`favorite.read`、`favorite.write`（`domain/actions.go`、规则表、`tableCells`）；模块的清理一步（35 → 36）、名单与 `purge_test.go`；e2e 的 `asynchronousDeletionTable`。红窗口 (a)、(b) 起。

### T3b 删除任务的收藏一步 → Go 两路的中间检查
删除任务改为按表的步骤（视图 → 收藏；项目范围、工作区范围；全序加锁）；`navigationTable`；Go 删除守卫的 SQL 种子；navigation 的两个删除测试。红窗口 (a) 在此关闭。之后本机跑 Go 两路（e2e 预期红）。

### T4 领域与存储
领域：收藏、五种类型、建与改的字段搭配、文件夹名、次序、父规则、无改动、错误；单测与变异。存储：sqlc 查询与 adapter 测试——列表、取本人的行、插入（条目写明冲突目标后读出已有；文件夹按约束名换 409）、本人最大次序、改、软删一行与文件夹连子项、按目标与按项目的连带、锁（全序：父文件夹 `FOR SHARE`、本行 `FOR UPDATE`、文件夹与子行）。

### T5 端口、读与建
端口：project 的事实加 `Archived`、批量（名字、标识、`logo_props`、归档、工作区）；planning 的迭代、模块批量（工作区、项目、名字、归档、已删）；access 的 `Readable` 桥；navigation 自己的视图批量与 `ShareView`；bootstrap 转换并接进 `navigation.Deps`；各自的 adapter 测试。用例：`favoriteLocks`；列表（一个只读快照、spec §2.5 的可见谓词、开关不看）；建（合格、一个 422、再收藏、次序、父）；拒绝次序（建）。单测与变异。

### T6 改与删
改（父、次序、文件夹名、无改动）、删（文件夹连子项）；一个 404（含目标已看不到）；全序加锁、锁下重读重判；拒绝次序（改、删）。单测与变异。

### T7 契约、处理器、接线与网页的码
`navigation.yaml`（四个操作、`Favorite` 等 schema；`createFavorite` 的 201 与 200）、根 `openapi.yaml` 的标签说明与路径；`make gen`；处理器逐码（spec §2.4）；`New` 的用例与 `Register`；两个新码的文案。红窗口 (c) 起。

### T8a 三种归档事件
planning、project 声明并发布（spec §2.6、R3）：项目归档的时钟读一次、写与事件共用，工作区 id 取 `lockAndDecide` 的返回值；`project.archived` 的命名常量与注释；单测（发布、恢复不发、迭代再归档不发、项目再归档照发、时刻相等）；`TestIssueEventDirectory`、`TestPlanningEventDirectory` 的目录。

### T8b 连带
连带订阅者（五种事件、按类型校验、另起名字）、订阅者构造的签名与调用点；`deleteView` 的连带（同一时刻、同一写者）；R12（Go 在归档后再种；e2e 的 `removedBefore` 与 P4 两条的传参，P4 仍在窗口 (b) 里）；`TestPlanningEventDirectory` 改为两组订阅者（R13）。

### T9 守卫
权限矩阵（四个操作，只用工作区列；建的变体：文件夹、工作区视图、带 `prepare` 的项目目标；改、删用每列自己的文件夹，SQL 种）；锁序测试 `TestFavoriteWritesTakeTheirTargetsLocks`（spec §2.8）。红窗口 (c) 在此关闭。

### T10 e2e 删除夹具 → **检查点 1**
`furnishIssueRows` 经接口收藏项目、迭代、模块、项目视图（`PlanningRows` 带回视图 id）；`furnishWorkspace` 与 W3 自己的 `furnish` 建文件夹、把工作区视图收进去；验证 P4（两条，含 R12 的 `removedBefore`）、W2、W3（两个文件）、I21。红窗口 (b) 在此关闭。**检查点 1**：七项本地门禁、分支 CI、两路各自的增量。

### T11 组合测试
H15 `TestFavoritesOfEveryKindThroughTheApp`（spec §4 的各格，含 H19 的项目级格、开关、已归档的活行、一个 422、R14 的子测试）、H16 `TestFavoritesFollowTheirTargets`（含项目再归档、`deleteView` 的同一时刻）。种子写明：两个同 `sequence` 的收藏、两个人收藏同一目标、只看自己的人与别人的视图、SQL 留下的已归档目标的活行。

### T12 交错
`TestFavoriteWritesInterleaveWithArchives`：建、改、删 × 项目归档 / 迭代归档 / 迭代删除 / 视图删除 / 项目删除 / 成员移除；删文件夹 × 移入它 / 在它里面新建 / 改它里面的子行（带同一个 `parent_id`）；配对共用准备好的库、分组；`waitForStartupRuns`。

### T13 e2e 接口故事 → **检查点 2**
`h15-favorites.spec.ts`（PAT 与会话对等）、`h16-favorite-cascades.spec.ts`；数据库断言。**检查点 2**：七项本地门禁、两路各自的增量、e2e 两片的时间。

### T14 文档与交接（主控）
plane-diff（表一行；行为一行：访客、写入校验、`project_id` 由服务端推出、连带对所有人、工作区视图可收藏（Plane 后端收、网页收藏不了）、次序按人、文件夹名在后端、软删（Plane 的 DELETE 硬删）、读时过滤（Plane 只按有效成员滤、不看归档）、列表含文件夹里的（Plane 只给顶层、另有子项接口）、改只收三个字段、回答形状、POST 201、开关不看）；交接（spec §5 的六条）；硬规则；设计 §3.8、§4.2（两条唯一加 `deleted_at IS NULL`、审计列）、§4.3（清理计数：今天 36，P3d 后 37）、§5（201/200、`/favorites/{favorite_id}`）、§14；阶段记录。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 一个 404 与一个 422：看不到、不存在、别人的、目标已看不到逐值相同；不合格的目标不泄露存在。
- 锁序与锁下重判；收藏行的全序（文件夹先于条目）；连带与删除任务。
- 三种归档事件的发布时机、时刻与载荷；连带对所有人、恢复不加回；`deleteView` 的连带。
- 删除图与两条“先归档再删除”的守卫（R12）；清理。
- 模块边界：navigation 只经端口拿项目、迭代、模块、可读项目；`bootstrap/app.go` 的拆分保持行为（R2 的六条次序）。
