# M7 / P3b：navigation 模块与视图

> 状态：spec 经预检 [re-90](../../../reviews/re-90-M7P3b预检.md) 修正后通过。基线 main `17e0b1ac`（P3a 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H12–H14）、§3.7、§3.10、§3.11、§4.1–§4.3、§5、§6.3、§9、§10、§15 D3/D4/E；预检 re-81 的 I7、I10、I11。现状调研 `research/p3-facts-base-7dfaa8a1.md`（P3 调研，基线 `7dfaa8a1`；DP-n 指它 §9 的决定点；re-90 已按 `17e0b1ac` 重核，行号以 re-90 为准）。前一段 [P3a](P3a-workspace-list.md) 已交付工作区跨项目列表与它的查询词 `WorkspaceList`。

## 1. 目标与边界

新模块 `navigation`：一张 `issue_views`（`project_id` 为空即工作区视图），项目视图与工作区视图的建、列、取、改、删，规则、开关，删除图与清理，守卫与组合测试、e2e 接口故事。

不做：收藏与最近访问（P3c；删除视图时收藏的连带随 P3c，DP-26）；页面与 store（P6a、P6b；旧 `view.service.ts`、`WorkspaceService` 的四个方法随页面删）；“保存为视图”入口的收起（P4）。

**P3b 合入后 main 的样子**：视图接口可用，网页仍打旧地址（P6a/P6b 接）；用户看不到变化。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/migrations/sql/00052_navigation_issue_views.sql` + schema 测试 | 表、CHECK、索引（§2.3） |
| `server/internal/modules/navigation/` | 模块（照 `notification`：`module.go`、`module_events.go`、`module_jobs.go`、`module_purge.go`；`domain/`、`app/`、`adapter/{postgres,river,http}`） |
| `server/internal/modules/issue/` | `WorkspaceListSettings`（工作区视图的已存词，同 `ListSettings.Validate` 的签名，DP-8 (a)）；项目视图直接用已有的 `ListSettings`（`issue/app/list_settings.go:19`，它写死 `ProjectList`） |
| `server/internal/modules/access/` | 8 个动作的规则 |
| `server/internal/modules/project/` | navigation 要的端口：项目的事实（工作区、`issue_views_view`）与项目 `FOR SHARE`（照 `adapter/postgres/queries/planning_ports.sql:1-26`）；不给“归档”（R2，没有人读） |
| `server/internal/modules/workspace/` | 不改：bootstrap 转换已有的 `WorkspaceDirectory`（按 slug 找、两种 `FOR SHARE`，`workspace/module.go:46-58`） |
| `api/common.yaml`、`server/internal/shared/` | `LogoProps` 的纯搬移（R5：`LogoProps`、`LogoEmoji`、`LogoIcon` 三个 schema；Go 的类型与 `Check`，连同 NUL 检查；project 改用 `shared.LogoProps`、处理器改用 `apigen` 的类型，不留别名） |
| `api/modules/navigation.yaml`、`api/openapi.yaml` | 契约（§2.4） |
| `server/internal/bootstrap/` | 接线、端口转换、清理表、任务；守卫（§4） |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql`、config | 模块的 sqlc 条目、运行角色、删除任务的批量与预算（config 七处与 README 的配置表、清理次序，plan T7 点名） |
| `web/` | 两个新码的文案（`web/apps/web/core/lib/error-messages.ts` 与 `web/packages/i18n/src/locales/{en,zh-CN}/errors.json`）；`make gen` 的 `schema.gen.ts` |
| `e2e/` | 删除夹具经接口建两种视图；H12、H14（含 H13 已存工作区视图的往返）的接口故事 |
| `docs/` | plane-diff、交接、硬规则、设计 §3.7/§14、阶段记录 |

### 2.2 动作、角色与相对规则（设计 §3.7、§3.10）

| 动作 | 级别 | 角色 | 用例里的相对规则 |
|---|---|---|---|
| `view.read` | 项目 | 管理员、成员、访客 | 只看自己（`Grant.OwnOnly`）的人只看见自己所有的视图 |
| `view.create` | 项目 | 管理员、成员 | 开关开着 |
| `view.update` | 项目 | 管理员、成员 | 只有所有者（D29）；开关开着 |
| `view.delete` | 项目 | 管理员、成员 | 所有者或 `Grant.ProjectAdmin`；开关开着 |
| `workspace_view.read/create` | 工作区 | 管理员、成员 | — |
| `workspace_view.update` | 工作区 | 管理员、成员 | 只有所有者 |
| `workspace_view.delete` | 工作区 | 管理员、成员 | 所有者或 `Grant.WorkspaceRole` 为管理员 |

- “项目管理员”是 `Grant.ProjectAdmin`（项目管理员，或是项目有效成员的工作区管理员，访客也算，`access/domain/decide.go:85`）。在项目里是访客的工作区管理员通过 `view.create/update/delete` 的规则（`:99`）：能建、能改自己的、能删自己的；别人的在 `OwnOnly` 为真时一律 404，为假（开了 `guest_view_all_features`）时能删（他是 `ProjectAdmin`）。不是项目成员的工作区管理员看得到项目，读写都是 403。
- **工作区访客与已存工作区视图（R1，DP-14 (a)）**：照 §3.10 的表与 H14，访客的列、取、建、改、删都 403（`decideWorkspace` 对规则之外的角色答 403，不用写代码）；§3.7 那句“访客只看自己的（照 Plane，也就是没有）”改写以免歧义（访客只有四个默认视图，D3）。
- **只看自己（DP-15）**：只看自己的人（没开 `guest_view_all_features` 的访客，含工作区管理员兼项目访客）对别人的视图，取、改、删都是 404 `navigation.view_not_found`，列表里没有（在 SQL 里按 `owned_by_id = 调用者` 过滤）。依据：`view.read` 通过时看 `Grant.OwnOnly`；改、删上 `Authorize` 对访客答带 `Grant` 的 `Forbidden`，用例先看其中的 `OwnOnly` 与所有者，藏起的答 404，然后才轮到 403。开了的访客看全部。
- **只看自己的人建视图（DP-16 (a)）**：照角色表可建；视图不暴露工作项，执行时列表照样按只看自己过滤。
- **已归档项目（R2，DP-13 (a)）**：视图的读写照常（M3 3.19 通则；`Authorize` 对已归档项目照常判定）；登记差异（Plane 滤掉已归档项目的视图）。
- **所有者离开**之后视图照常列出，没有人能改，管理员能删（Plane 相同，plane-diff 登记一句）。
- 在 `/projects/{project_id}/views`、`/workspaces/{slug}/views` 上：看不到项目或工作区答 404 `project.not_found` / `workspace.not_found`；看得到没权限答 403。`/views/{view_id}` 见 §2.5。

### 2.3 表 `issue_views`（DP-21）

| 列 | 定义 |
|---|---|
| `id` | `uuid PRIMARY KEY` |
| `workspace_id` | `uuid NOT NULL → workspaces ON DELETE CASCADE` |
| `project_id` | `uuid NULL → projects ON DELETE CASCADE`（为空即工作区视图） |
| `name` | `varchar(255) NOT NULL`（按字符数 1–255，去首尾空白后非空） |
| `description` | `text NOT NULL DEFAULT ''`（纯文本，没有资产，不设长度上限，Plane 相同） |
| `logo_props` | `jsonb NOT NULL DEFAULT '{}'`（`{}` 即没有图标；CHECK 照 `00010` 的项目图标） |
| `filters` | `jsonb NOT NULL`（CHECK：键集按 `project_id` 是否为空分两种；值为字符串数组；`CASE` 保护照 `00046:15-22`） |
| `display_filters`、`display_properties` | `jsonb NOT NULL`（CHECK 照 `00046`，日历的形状照常、七个键齐） |
| `owned_by_id` | `uuid NOT NULL → users ON DELETE CASCADE`（Plane 相同） |
| `created_by_id`、`updated_by_id` | `uuid NULL → users ON DELETE SET NULL` |
| `created_at`、`updated_at` | `timestamptz NOT NULL` |
| `deleted_at` | `timestamptz NULL` |

**已存的筛选键**（re-90 I1）：项目视图 **12** 个（`shared.ListFilters` 的 10 个共用键 + `cycle_id`、`module_id`）；工作区视图 **11** 个（10 个 + `project_id`）；契约的并集 **13** 个。`sub_issue` 不是已存的筛选键，只在 `display_filters.sub_issue`（布尔）；`archived` 不存。

索引：`(project_id, created_at DESC, id DESC) WHERE deleted_at IS NULL AND project_id IS NOT NULL`、`(workspace_id, created_at DESC, id DESC) WHERE deleted_at IS NULL AND project_id IS NULL`；外键的不带条件索引（`workspace_id`、`project_id`、`owned_by_id`）；`deleted_at` 的清理索引。约束与索引的名字照 `<table>_<column>_idx`、`…_deleted_at_idx` 与已有 CHECK 的命名，schema 测试按名字钉。

**表名的陷阱（R3，DP-19 (a)）**：保留 `issue_views`；Go 的 `projectIssueTable`（`bs/issue_project_deletion_jobs_test.go:28-31`，按 `issue_` 前缀认 issue 模块的表）与 e2e 的 `asynchronousDeletionTable`（`e2e/fixtures/assert/catalog.ts:94-112`）改为不误认它；视图以 navigation 自己的谓词留在 `parentAsyncTable` 里。与建表同一个 Task。

**bootstrap 的 `navigation` 冲突（R4，DP-20 (a)）**：测试包里包级的 `type navigation`（`bs/permission_matrix_workspace_test.go:227`）改名；在任何 bootstrap 文件 import `modules/navigation` 之前（第一个是 `actions_test.go` 的 `moduleActions()`），即建模块目录的同一个 Task。

### 2.4 接口（设计 §5，DP-11 (a)）

| 操作 | 路由 | 回答 | `x-problem-codes` |
|---|---|---|---|
| `listProjectViews` | `GET /projects/{project_id}/views` | `{data: [View]}` | `project.not_found`、`forbidden` |
| `createProjectView` | `POST /projects/{project_id}/views` | 201 `View` | `validation_failed`、`project.not_found`、`forbidden`、`navigation.views_disabled` |
| `listWorkspaceViews` | `GET /workspaces/{slug}/views` | `{data: [View]}` | `workspace.not_found`、`forbidden` |
| `createWorkspaceView` | `POST /workspaces/{slug}/views` | 201 `View` | `validation_failed`、`workspace.not_found`、`forbidden` |
| `getView` | `GET /views/{view_id}` | `View` | `navigation.view_not_found`、`forbidden` |
| `updateView` | `PATCH /views/{view_id}` | `View` | `validation_failed`、`navigation.view_not_found`、`forbidden`、`navigation.views_disabled` |
| `deleteView` | `DELETE /views/{view_id}` | 204 | `navigation.view_not_found`、`forbidden`、`navigation.views_disabled` |

- 视图的写不进项目的写闸门，不声明 503 `server_busy`。
- `View`（DP-23）：`{id, workspace_id, project_id|null, name, description, logo_props, filters, display_filters, display_properties, owned_by_id, created_at, updated_at}`；不带 `is_favorite`（设计 §3.8）。
- 列表集合型、不分页（设计 §3.7），`created_at` 倒序、同值按 `id`；`listProjectViews` 只列这个项目的，`listWorkspaceViews` 只列 `project_id IS NULL` 的，两条部分索引各对一条。
- 一个 `View` schema，`filters` 为 13 键的并集，按视图种类 422（DP-10 (a)）；`LogoProps` 引 `../common.yaml`。
- 词（设计 §3.7“按页面校验”）：项目视图的筛选与显示 = `ListSettings`（项目列表的已存词，12 键）；工作区视图的筛选 = `WorkspaceListSettings` 的 11 键，显示（DP-9 (a)）：`layout` 只收 `spreadsheet`、`list`，`group_by`、`sub_group_by` 必须为 null，`order_by` 照 `ListOrderAllowed`，日历的形状照常。
- 建（DP-22）：`name` 必填；`filters` 缺省 `{}`；显示两项缺省为 `shared` 的默认值；所有者是调用者。
- PATCH：给出的顶层字段整体替换；在规范化之后的领域值上与锁下重读的行比，什么都没变（含请求体 `{}`）→ 200 原样回答、不写、不动 `updated_at`。没有 PUT（D24）。
- 新码：`navigation.view_not_found`（404）、`navigation.views_disabled`（409）。

### 2.5 拒绝次序（DP-17，re-90 I2）

- **`/views/{view_id}` 上的每个 404 都是 `navigation.view_not_found`**，与不存在的 id 逐值相同（码、体、头）：不存在、已软删、它的项目或工作区看不到或已删、只看自己藏起。`Authorize` 给的 `ErrNotVisible` 不带码，由用例换成这个码。
- `{view_id}` 选动作：不加锁读视图行（navigation 自己的表）→ `project_id` 为空用 `workspace_view.*` 与 `Target{WorkspaceID}`，否则 `view.*` 与 `Target{WorkspaceID, ProjectID}`（P2c `seeChoosing` 的写法）。
- 建：请求体 400/422（名字、图标、按路由种类的词——纯函数，先于任何读）→ 404 项目或工作区 → 403 角色 → 锁 → 锁下重读、重判 → 409 `views_disabled`（项目视图）。
- 取：读视图 → 404 → 403 角色。
- 改：读视图 → 404 → 403 角色 → 403 不是所有者（`forbidden`，不加新码）→ 422（名字、图标、按种类的词，一个回答）→ 锁 → 重读、重判 → 409 `views_disabled`（项目视图）→ 无改动。
- 删：读视图 → 404 → 403 角色 → 403 不是所有者也不是管理员 → 锁 → 重读、重判 → 409 `views_disabled`（项目视图）。

### 2.6 锁序（DP-12 (a)，re-90 I7）

- M3 约定：工作区 `FOR SHARE` → 项目 `FOR SHARE`（项目视图）→ 视图行 `FOR UPDATE`；工作区视图只锁工作区 S → 视图行。不进项目的写闸门。开关、归档、成员与角色的变化取项目行或工作区行的 `FOR NO KEY UPDATE`，与之串行，锁下判定为准。
- 项目写锁表按**操作 id** 分类：`createProjectView`、`updateView`、`deleteView` 各一行，指向项目视图；`rowPaths` 加 `/api/v0/views/`；准备时打开两个项目的 `issue_views_view`（默认关着）。矩阵里 `getView`、`updateView`、`deleteView` 各两个 `variant`（项目视图、工作区视图）。
- 工作区视图那一支另写一条锁序测试，证明：工作区被 `FOR NO KEY UPDATE` 扣住时，工作区视图的改、删在等；等的时候不持有视图行；视图行的锁在工作区 S 之后。

### 2.7 删除图与清理（设计 §4.3，DP-18 (a)，re-90 m4）

- 订阅者：`project.deleted` 按事件里的项目逐个、`workspace.deleted` 一步，在同一事务投递 `navigation.delete_rows`（参数：范围、id、时刻、删除者，照 `issue/app/delete_rows_on_deletion.go:20-35`）。任务不等工作项；分批按 `id` 升序加锁（M3 约定五）；软删时 `deleted_at`、`updated_at` 取事件的时刻，`updated_by_id` 取删除者；工作区那一步只删 `project_id IS NULL` 的行；显式 `Timeout`、有限次数，次数用尽记日志。config `navigation.delete_batch`、`delete_budget`。
- 立即不可见：读写都经 `Authorize`（项目或工作区已删 → 看不到 → 404），不依赖任务何时跑完。
- 清理：模块的清理表（插在 notification 与 issue 之间）与清理器；清理目录的计数 34 → 35 与各处名单。

## 3. 局部裁定

- **R1** 工作区访客对已存工作区视图 403（§2.2）；设计 §3.7 那句改写。
- **R2** 已归档项目上的视图读写照常（§2.2）。
- **R3** 表名保留 `issue_views`，两处前缀启发式改为不误认（§2.3）。
- **R4** bootstrap 测试包的 `type navigation` 改名，在建模块目录的同一个 Task（§2.3）。
- **R5** `LogoProps` 跨模块纯搬移，单独提交，不改行为（§2.1）。
- **R6** 导航操作的删除窗口：H12 组合测试的一个子测试，用 `holdNavigationDeletion` 扣住 navigation 的删除任务，在项目删除、工作区删除之后、视图行还活着时，每个视图操作答 404，与不存在的 id 逐值相同（DP-24 (a)）。
- **R7** 矩阵的“所有者”列复用 `issueColumns`（“建了它的成员” = 所有者；访客·全部、访客·自己、访客·别人、管理员兼访客两格现成）；工作区视图的行用两个 `variant`（视图是成员的、是管理员的）；`{view_id}` 的目标规则排在 `underProject` 之前，三条反例（工作区列指了项目视图、项目列指了工作区视图、别的工作区的视图）。访客所有的视图用 SQL 种。
- **R8** `TestProjectDeletionPublishesAndEnqueuesAtomically`（扣住删除任务后断言异步表没变）：加 `holdNavigationDeletion`，这条测试的连接数加一，断言不改（re-90 I6 的甲）。

## 4. 验收标准

- 契约、生成物、处理器逐码（§2.4 的表）；`make gen-check` 无差异；网页的两个码有文案。
- 迁移的 schema 测试（列、可空、默认、索引与约束名、CHECK 反例：两种 `filters` 键集、`sub_issue` 不在筛选里、图标、显示两项）。
- 单测：领域（名字、图标、两种词、所有者规则、错误）、用例（拒绝次序、锁下重判、开关、只看自己、无改动）。交叉格：建——坏词 × 看不到的项目 → 422；改——坏词 × 看不到的视图 → 404；关着的开关 × 不是成员 → 403；不是所有者 × 坏词 → 403。
- 守卫：权限矩阵（每个操作的行、`{view_id}` 两种归属与反例、R7 的格）、项目写锁表（三行与开关的准备）、工作区视图的锁序测试、动作 ↔ 规则与模块目录、删除（Go 守卫、清理目录与名单）、导航操作的删除窗口（R6）。
- 组合测试（真实的 app）：
  - H12：项目视图的建、列、取、改、删；所有者与 `ProjectAdmin`；只看自己（访客·别人、管理员兼访客·别人）；开关关着写 409、读照常；已归档项目照常；列表次序（两条同一时刻的视图）；所有者不是删除者。
  - H14：工作区视图；工作区访客 403；工作区管理员删别人的、成员删不了别人的。
  - H13 已存工作区视图的往返：存得下的每个筛选键、显示词，原样作为查询发给 `GET /workspaces/{slug}/issues` 都被接受并筛出预期的集合（`sub_issue` 从显示里取）；项目视图对项目列表同理（含 `cycle_id`、`module_id`）。
- 交错：视图写 × 项目删除 / 工作区删除 / 开关 / 成员移除（锁下重判）。
- e2e：删除夹具经接口建两种视图（项目视图进 `furnishPlanningRows`，工作区视图进 `furnishWorkspace` 与 W3 自己的 `furnish`），P4（两条）、W2、W3（两个文件）、I21 转绿；H12、H14（含 H13 的往返）的接口故事 `e2e/stories/collaboration/h12-project-views.spec.ts`、`h14-workspace-views.spec.ts`（PAT 与会话对等，数据库断言）。
- 每个性质至少一个能让测试失败的变异（plan 的变异表）。
- 测试时间：两路各自的增量 = 落在这一路的新顶层名 + 这一路里被加重的已有测试（`TestPermissionMatrix` 桶 46、`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 桶 52、`TestPurgeCatalogCoversTablesAndIncomingKeys` 桶 59、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow` 桶 66 与删除守卫，多在第 1 路）；按作业的 Test 步实测，每路 ≤ 90 秒；第 1 路的 bootstrap 包到 `go test` 期限的八成先报告。

## 5. 后续责任与交接

- P3c：收藏与最近访问；`deleteView` 加“删除所有人对它的收藏”一步（DP-26）。
- P6a/P6b：页面、store、旧 service 的删除、`viewsEntryOn`；M6-M7-04、M6-M7-07、M3 交接 §2–§4 的视图项都还在那边。
- 本段写处理结果的交接（T14）：M2 §1（后端部分完成：集合型不适用、游标在 P3a、跨模块联表的结论是经端口；保持 open）、M2 §5（`owned_by_id CASCADE`、审计列 `SET NULL`）、M4 M7-03（视图的接口一支；页面与旧方法仍随 P6a/P6b）、M6-M7-03（项目视图有迭代、模块词）、M6-M7-04（只到接口与两个码的文案）、M3 §1 的筛选列（`project_id` 在工作区视图上，不在 `workspace_user_properties`）、M5-M7-03（视图的说明是纯文本）。

## 6. 风险

- 删除图与守卫面广（P1b 同量级）：三个红窗口写进 plan，各带测试名。
- 测试时间：第 1 路的余量（P3a 阶段记录 §6）；超出预算按 P2b 的做法报告，不自行改作业结构。
