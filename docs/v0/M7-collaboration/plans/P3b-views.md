# M7 / P3b plan：navigation 模块与视图

> spec：[P3b spec](../specs/P3b-views.md)（预检 [re-90](../../../reviews/re-90-M7P3b预检.md) 修正后通过）。基线 main `17e0b1ac`。worktree `.codex/worktress/codex/m7-p3b`，分支 `codex/m7-p3b`。台账 `.superpowers/sdd/P3b-views/progress.md`（ignored）。16 个 Task（T8、T10 各分 a、b）。

## Global Constraints

- spec §2 是合同，§3 的 R1–R8 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；每个声明的码有处理器测试回答；契约与处理器同一 Task。
- 模块之间只经端口；navigation 的 SQL 只经自己的 sqlc（不能联表 `projects`、`workspaces`）；新模块照 `notification` 的形状；手写文件约 400 行以内。
- 允许改的已有测试（改之前在台账逐条写明，只照这里写的改法）：
  - 迁移：`server/migrations/trigram_schema_test.go`（迁移数 51 → 52、回滚循环的上界、表名名单加 `issue_views`）、`purge_indexes_test.go`（34 → 35）、`schema_test.go`（拼接加 navigation 的名单）；新 schema 测试。
  - R3：`bs/issue_project_deletion_jobs_test.go` 的 `projectIssueTable` 与 `parentAsyncTable`、`e2e/fixtures/assert/catalog.ts` 的 `asynchronousDeletionTable`。R4：`bs/permission_matrix_workspace_test.go` 的 `type navigation` 改名（与它的几个使用处）。
  - 动作：`access/domain/rules_test.go` 的 `tableCells`；`bs/actions_test.go` 的 `moduleActions()`。
  - 清理：`bs/purge_catalog_test.go`（计数 34 → 35、`providedPurgeTables`）、`bs/purge_runtime_test.go`（分组、表、模块次序）、`bs/purge_scale_fixture_test.go`、`bs/issue_retention_sets_test.go`；`deploy/runtime-grants.sql` 与 `bs/runtime_role_test.go` 的表。
  - 删除图（每处只加 navigation 的一项）：`projectDeletionPublisher`（按模块逐个列订阅者）、`bs/issue_delete_job_fixture_test.go` 与 `bs/issue_lifecycle_events_test.go`（自拼的任务与种类）、`bs/request_jobs_test.go`（按名字断言种类）、`bs/job_inserter.go` 的断言；Go 删除守卫的夹具（`bs/project_deletion_test.go`、`bs/workspace_deletion_test.go` 的种子用 SQL 加两种视图）、等待删除完成的辅助；`TestProjectDeletionPublishesAndEnqueuesAtomically` 加 `holdNavigationDeletion` 与一个连接（R8，断言不改）。
  - config：`si/platform/config/validate_test.go`、`purge_test.go`、`load_test.go`（只加 navigation 的行）。
  - 矩阵：行（`issueColumns`）、`{view_id}` 的目标规则与三条反例（R7）。
  - 项目写锁表：`rowPaths` 一项、`projectWrites` 三行、开关的准备（建项目之后打开两个项目的 `issue_views_view`，不改已有的行）。
  - e2e 夹具：`furnishPlanningRows`（项目视图）、`furnishWorkspace` 与 W3 自己的 `furnish`（工作区视图）。
  - project 各层测试里 `LogoProps` 的类型名（R5，断言不改）。
  - 生成物。

  其余断言是回归，不改。
- 红的窗口（台账记下）：
  - (a) T4 → T7：Go 的 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 与同夹具的几条（表有了，删除图还没有）。
  - (b) T4 → T10b：e2e 的 P4（两条）、W2、W3（`w3-workspace-settings.spec.ts`、`w3-workspace-settings-page.spec.ts`）、I21。
  - (c) T9 → T10a：`TestThePermissionMatrixCoversEveryOperation`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`。
  - 清理的守卫不红（T4 一并接上清理）；动作 ↔ 模块目录的守卫不红（T4 一并加动作）。每个 Task 写明它验证什么、哪些守卫预期红。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`；下表每个性质至少一个能让测试失败的变异。
- 测试时间：两路各自的增量 = 落在这一路的新顶层名 + 这一路里被加重的已有测试（spec §4 列出）；新顶层名提交前用 `printf %s 名字 | cksum` 算好落在哪一路，记进台账；检查点按作业的 Test 步实测，每路 ≤ 90 秒，第 1 路的 bootstrap 包到 `go test` 期限的八成先报告。造好数据后扣锁或写墓碑的测试先 `waitForStartupRuns`；等真实周期任务的窗口要盖住约 6 秒。本机两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 为准并记录。

## 性质与变异

| 性质 | 变异 | Task |
|---|---|---|
| 两种词按种类校验 | 工作区视图收 `cycle_id`；项目视图拒 `cycle_id`；工作区视图的显示收 `group_by`；工作区视图的筛选收 `sub_issue` | T3、T5 |
| CHECK 的两种键集 | 去掉键集的区分（schema 测试的反例） | T4 |
| 名字、图标 | 名字收空白；图标收坏结构 | T5 |
| 所有者规则 | 改不查所有者；成员删别人的；`ProjectAdmin` 删不了别人的；工作区管理员删不了别人的工作区视图 | T5、T8b、T11 |
| `{view_id}` 选动作 | 工作区视图按 `view.*` 判（或反过来） | T8a |
| 404 逐值相同 | 看不到的项目里的视图答 `project.not_found` | T8a、T11 |
| 只看自己 | 列表不滤；取别人的答 403；改、删别人的（访客·别人、管理员兼访客·别人）答 403 或 204 | T8a、T8b、T11 |
| 开关 | 写不查 `issue_views_view`；读也查 | T8a、T8b、T11 |
| 工作区访客 | 访客能列 | T4（规则）、T11 |
| 拒绝次序（两个方向） | 建：词的校验挪到读之后；改：词的校验挪到读之前（看不到的视图 + 坏词应答 404）；409 挪到 403 之前（不是成员的人在关着开关的项目上应答 403） | T8a、T8b |
| 无改动 | 无改动仍写、仍动 `updated_at`；PATCH 合并而不是整体替换 | T8b |
| 锁下重判 | 去掉锁下重读 | T8b、T12 |
| 锁序 | 不 `FOR SHARE` 工作区 | T8a、T10a |
| 列表的次序 | `created_at` 同值时不按 `id`（两条同一时刻的视图） | T6、T11 |
| 已归档项目照常 | 已归档项目上的写被拒 | T11 |
| 删除图 | 订阅者不投 `navigation.delete_rows`；任务不删；时刻、删除者取 `now()` 与任务自己；工作区那一步缺失 | T7 |
| 立即不可见 | 读不经 `Authorize`（R6，扣住任务时） | T11 |
| 清理 | 清理表漏视图 | T4 |
| 矩阵与写锁守卫 | 去掉一行；`{view_id}` 的目标规则放在 `underProject` 之后 | T10a |
| 往返 | 存得下列表不收的词 | T11 |

## Tasks

### T1 基线与文档（主控）
spec、plan、预检 re-90；P3a 阶段记录 §9 回填集成（含修复轮 `r1-scale` 在 `c5629ec7` 上的数字，P3a 限定复审 R1）；设计 §14 的 P3a 一行改为已完成、加 P3b 一行；设计 §3.7 关于工作区访客的那句（R1）。

### T2 `LogoProps` 的纯搬移（R5）
三个 schema 搬到 `api/common.yaml`，`project.yaml` 的三处 `$ref` 改指它，说明改为不指项目的句子；Go 的类型与 `Check`（连同 NUL 检查）搬进 `shared`，project 各层直接用 `shared.LogoProps`、处理器用 `apigen` 的类型；`make gen`；断言不改。

### T3 issue 的 `WorkspaceListSettings`
工作区视图的已存词校验（11 键，排除 `sub_issue`、`archived`；显示按 spec §2.4）；与 `ListSettings` 同一签名；由 `issue.Provide` 交出；单测与变异。

### T4 迁移、模块骨架、动作与清理
`00052` 与 schema 测试、迁移的计数与名单；运行角色；R3、R4；模块骨架（`Provide` 交出清理器、`module_purge.go`、清理查询与 sqlc 条目）、bootstrap 的清理接线与清理名单；8 个动作（`domain/actions.go`、模块根的 `Actions()`）、规则表、`tableCells`、`moduleActions()` 一行——与建出模块目录同一个提交。红窗口 (a)、(b) 起。

### T5 领域
视图、两种筛选、显示、名字、图标、所有者规则、错误；单测与变异。

### T6 存储
sqlc 查询（两种列表、取、建、改、软删、锁）与 adapter 测试（列表次序、部分索引的两条列表）。

### T7 删除图
订阅者与 `navigation.delete_rows` 任务（spec §2.7）；config 七处与 README 的配置表、清理次序；bootstrap 里订阅者与任务的接线（`job_inserter.go` 的断言一行）；允许清单里删除图的各处；Go 删除守卫的夹具（SQL 种两种视图）；R8。红窗口 (a) 在此关闭。

### T8a 端口、读与建
端口（项目的事实与 `FOR SHARE`、`WorkspaceDirectory` 的转换、issue 的两种词）与 bootstrap 转换；列、取、建两种视图；`{view_id}` 选动作与 404 逐值相同；只看自己的读；拒绝次序（建）。单测与变异。

### T8b 改与删
改、删两种视图；所有者与管理员；只看自己藏起；锁序、锁下重读重判；开关；无改动；拒绝次序（改、删）。单测与变异。

### T9 契约、处理器、接线与网页的码
`navigation.yaml`、根 `openapi.yaml` 的标签与路径；`make gen`；处理器逐码（spec §2.4 的表）；`New` 的用例与 `Register`；两个新码的文案。红窗口 (c) 起。

### T10a 守卫
权限矩阵（R7）、项目写锁表（spec §2.6，含开关的准备）、工作区视图的锁序测试。红窗口 (c) 在此关闭。

### T10b e2e 删除夹具 → **检查点 1**
项目视图进 `furnishPlanningRows`，工作区视图进 `furnishWorkspace` 与 W3 自己的 `furnish`（先开 `issue_views_view`）；验证 P4（两条）、W2、W3（两个文件）、I21。红窗口 (b) 在此关闭。**检查点 1**：七项本地门禁、分支 CI、两路各自的增量。

### T11 组合测试
H12、H14、H13 的往返（spec §4 的各格；种子写明：两条同一时刻的视图、所有者不是删除者、访客所有的视图用 SQL 种）；R6 的子测试。新顶层名算好落路。

### T12 交错
视图写 × 项目删除 / 工作区删除 / 开关 / 成员移除；`waitForStartupRuns`。

### T13 e2e 接口故事 → **检查点 2**
`h12-project-views.spec.ts`、`h14-workspace-views.spec.ts`（含 H13 的往返；PAT 与会话对等，数据库断言）。**检查点 2**：七项本地门禁、两路各自的增量、e2e 两片的时间。

### T14 文档与交接（主控）
plane-diff（没有 PUT、建有角色、看不到 404、开关、已归档项目照常、工作区访客、列表次序、按页面校验的词、所有者离开之后）；交接（spec §5 的七条）；硬规则；设计 §3.7、§14；阶段记录。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 两种视图共用 `/views/{view_id}` 时的授权：先读出种类再按种类选动作，看不到与不存在逐值相同。
- 锁序与锁下重判；开关；只看自己；工作区访客。
- 删除图：订阅者、任务、清理、Go 与 e2e 的守卫；表名陷阱。
- 模块边界：navigation 只经端口拿项目、工作区、词表。
