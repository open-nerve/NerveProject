# M6 / P2：模块的后端

> 状态：待预检（2026-11-07）。基线 main `0a601f212e32057c4e48fe43e8c4f19691254573`（P1 noff，main CI 42343073949 通过）。设计见 [M6 设计](../M6-design.md)，负责人 2026-11-07 的决定 A/B/C 见设计 §15；P1 的加锁之路与拒绝次序见 [P1 spec](P1-cycles-backend.md) §3。

## 1. 目标与边界

在 P1 的 `planning` 模块里交付模块本身：三张表 `modules`、`module_members`、`module_links`；模块的建/读/改/删、归档与恢复、归档目录（`?archived=true`）；名字唯一、状态、日期；负责人与成员的校验；链接的增改删、列表与元数据的后台抓取；项目开关 `module_view`；权限矩阵；删除图（删模块连带成员与链接、项目/工作区删除的清理任务、60 天物理清理）。覆盖设计 §2 的 G9、G10、G13 的接口与数据库证据。

同时落实设计决定 17：链接地址的规整校验与元数据抓取从 `issue` 移出、两个模块共用一份，工作项链接的行为与测试不变（落点见 R12）。

不做：工作项归属 `module_issues`（P3）；`planning.module_deleted` 事件（P3 随第一个订阅者加入，同 P1 对 `cycle_deleted` 的处理）；`module_user_properties` 与 `module.preferences`（P3）；进度、分布、燃尽（P4）；前端页面（P6）。前端只做新错误码的文案，和 `planning.archived` 文案改为迭代与模块通用（R13）。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/shared/link.go` 与测试 | 由 `issue/domain/link.go`、`link_metadata.go` 移来：`CheckLinkURL`、`CheckLinkTitle`、`LinkMetadata`（R12），测试随之移动，用例不变 |
| `server/internal/platform/outbound/linkpreview/` | 由 `issue/adapter/crawler/` 原样移来（`charset`、`metadata`、`crawler` 及六个测试文件），只把返回类型改为不依赖任何模块的 `Metadata`（R12） |
| `server/internal/modules/issue/` | `domain.Link` 等改用 `shared` 的检查与元数据类型；`app.LinkCrawler` 返回 `shared.LinkMetadata`；`Deps.Outbound` 换为 `Deps.LinkCrawler`；删除 `adapter/crawler/` 与原 `link_metadata.go` |
| `api/common.yaml`、`api/modules/issue.yaml` | `IssueLinkMetadata`、`IssueLinkMetadataError` 移入 common 为 `LinkMetadata`、`LinkMetadataError`，`IssueLink` 引用它；`apitest` 的形状测试随之改名 |
| `server/migrations/sql/00040_planning_modules.sql`、`00041_planning_module_members.sql`、`00042_planning_module_links.sql` | 三张表、约束、索引，各含 Down 段（§2.2） |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql` | planning 条目的 schema 加三个迁移；运行角色对三张表的四种 DML |
| `server/internal/modules/planning/domain/` | `module.go`（名字、状态、日期、`member_ids`）、`module_patch.go`、`module_link.go`、`actions.go`、`errors.go` 的新增 |
| `server/internal/modules/planning/app/` | `decide.go`/`lock.go`/`read.go` 的三种对象（R1）；`create_module.go`、`update_module.go`、`archive_module.go`、`delete_module.go`、`list_modules.go`、`get_module.go`、`module_links.go`（建、改、删、列）、`crawl_link.go`；`delete_rows.go` 的命名步骤（R16）；`purge.go` 的新步骤 |
| `server/internal/modules/planning/adapter/postgres/` | `modules.sql`、`module_members.sql`、`module_links.sql`、`link_metadata.sql` 及 `delete_rows.sql`、`purge.sql` 的新语句与 `Store` |
| `server/internal/modules/planning/adapter/http/` | 处理器与每个错误码的处理器测试 |
| `server/internal/modules/planning/adapter/river/` | `planning.crawl_link` 的 worker 与投递器（R11） |
| `server/internal/modules/planning/{module.go,module_jobs.go,module_purge.go}` | 接线；`EnqueuedKinds` 加 `planning.crawl_link` |
| `server/internal/modules/project/` | `PlanningProjects` 加 `ActiveRoles`（复用现有 `Store.ActiveRoles`，工作项的负责人检查读同一个） |
| `server/internal/modules/access/domain/rules.go` | `module.read/write/delete`（`LevelProject`；delete 带 `AllowCreator`） |
| `server/internal/bootstrap/` | 接线；一个 `linkPreviews` 适配器给两个模块（R12）；权限矩阵的模块表与链接表（R10）；删除、清理、运行角色与规模的守卫和夹具（§2.4） |
| `server/migrations/` 测试 | `planning_schema_test.go` 的约束与索引名；`purge_indexes_test.go` 与 `purge_catalog_test.go` 的表数 |
| `api/modules/planning.yaml`、`api/openapi.yaml` | 新路径（§2.3） |
| `web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/{en,zh-CN}/errors.json` | 新错误码的文案；`planning_archived` 改为通用 |
| `e2e/` | `furnishPlanningRows` 再打开 `module_view`，建一个以管理员为负责人和成员、带一条链接的模块；`catalog.ts` 的 `asynchronousDeletionTable` 加三张表；`stories/planning/` 的 G9、G10、G13 接口故事 |
| `tools/keywords.json` | `phase` 为 `M6/P2`（没有到期的例外） |
| `docs/` | P1 阶段记录 §7 与 spec A8、设计 §14 的回填（T1）；设计 §3.2 的落点措辞（R12）；`plane-diff.md`；输入交接的“处理结果（M6/P2）” |

### 2.2 表与 Plane 差异

`modules`：

| 列 | 定义 | 与 Plane |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 同 |
| `name` | `varchar(255) NOT NULL CHECK (name <> '')` | 同；去掉首尾空白后不能为空；项目内未删除者不分大小写唯一（Plane 只在代码里按原样比较，数据库唯一 `(name, project_id)` 区分大小写） |
| `description` | `text NOT NULL DEFAULT ''` | 同（Plane 由代码填空串） |
| `start_date`、`target_date` | `date NULL`，CHECK 两者都有时开始 ≤ 目标 | Plane 只在序列化器检查 |
| `status` | `varchar(20) NOT NULL DEFAULT 'planned'`，CHECK 六种 | Plane 只在 Python 的 choices |
| `lead_id` | `uuid NULL REFERENCES users ON DELETE SET NULL` | 同 |
| `project_id`、`workspace_id` | `uuid NOT NULL`，`CASCADE` | 同 |
| `sort_order` | `double precision NOT NULL DEFAULT 65535` | 同；新建取本项目未删除模块（含已归档）的最小值 − 10000，没有时 65535（Plane `module.py:115-123`） |
| `archived_at`、`deleted_at`、审计四列 | 照 M2 设计 3.13 | 同 |
| 删除 | `view_props`、`logo_props`、`external_source`、`external_id`、`description_text`、`description_html` | 没有读者（设计 §4.2） |

`module_members`：`id`、`workspace_id`、`project_id`（`CASCADE`）、`module_id`（`REFERENCES modules ON DELETE CASCADE`）、`member_id`（`REFERENCES users ON DELETE CASCADE`）、审计四列、`deleted_at`；部分唯一 `(module_id, member_id) WHERE deleted_at IS NULL`。删除 Plane 多余的 `UNIQUE (module_id, member_id, deleted_at)`。

`module_links`：`id`、`workspace_id`、`project_id`、`module_id`（均 `CASCADE`）、`title varchar(255) NULL`、`url text NOT NULL CHECK (url <> '' AND length(url) <= 2048)`（Plane `varchar(200)`）、`metadata jsonb NOT NULL DEFAULT '{}'` 带与 `issue_links` 相同的键与类型 CHECK（Plane 不约束）、审计四列、`deleted_at`；`module_links_module_id_url_key EXCLUDE USING hash ((module_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)`（照 `issue_links`；Plane 只在代码里查，且建立时因 `module_id` 只读而查不到）。

索引：每条 `CASCADE` 外键（含 `module_members.member_id`）无条件索引，指向 `users` 的 `SET NULL` 外键不建（P1 的约定）；活行列表 `modules (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL`；名字唯一 `modules_project_id_name_key (project_id, lower(name)) WHERE deleted_at IS NULL`；三张表各一个清理用的 `<表>_deleted_at_idx (deleted_at) WHERE deleted_at IS NOT NULL`。命名列入 `planning_schema_test.go`。

### 2.3 接口与错误目录

`Module`：`id, project_id, workspace_id, name, description, start_date, target_date, status, lead_id, member_ids, sort_order, archived_at, created_at, updated_at, created_by_id, updated_by_id`；`status` ∈ `backlog|planned|in-progress|paused|completed|cancelled`；`member_ids` 按 id 升序。

`ModuleLink`：`id, module_id, url, title, metadata, created_by_id, created_at, updated_at`；`metadata` 引用 common 的 `LinkMetadata`（同 `IssueLink`）。

| 操作 | 成功 | 声明的错误 |
|---|---|---|
| `GET /projects/{project_id}/modules?archived=` | 200 `{data}`；默认未归档按 `sort_order, created_at, id`；`archived=true` 只返回已归档，按 `archived_at DESC, id` | 400、404、403 |
| `POST /projects/{project_id}/modules` `{name, description?, start_date?, target_date?, status?, lead_id?, member_ids?}` | 201 | 400、422、404、403、409 `project.archived`/`planning.modules_disabled`/`planning.module_name_taken`、503 |
| `GET /modules/{module_id}` | 200 | 404 `planning.module_not_found`、403 |
| `PATCH /modules/{module_id}` `{name?, description?, start_date?, target_date?, status?, lead_id?, member_ids?, sort_order?}` | 200 | 400、422、404、403、409 `project.archived`/`planning.modules_disabled`/`planning.archived`/`planning.module_name_taken`、503 |
| `DELETE /modules/{module_id}` | 204 | 404、403、409 `project.archived`/`planning.modules_disabled`、503 |
| `POST /modules/{module_id}/archive`、`/unarchive` | 200 `Module` | 404、403、409 `project.archived`/`planning.modules_disabled`、`planning.module_not_closed`（归档）、503 |
| `GET /modules/{module_id}/links` | 200 `{data}`，按 `created_at DESC, id DESC`（同工作项链接与 Plane） | 404、403 |
| `POST /modules/{module_id}/links` `{url, title?}` | 201，`metadata` 为 `{}` | 400、422、404、403、409 `project.archived`/`planning.modules_disabled`/`planning.archived`/`planning.module_link_exists`、503 |
| `PATCH /module-links/{link_id}` `{url?, title?}` | 200 | 400、422、404 `planning.module_link_not_found`、403、409 同上、503 |
| `DELETE /module-links/{link_id}` | 204 | 404、403、409 `project.archived`/`planning.modules_disabled`/`planning.archived`、503 |

看不到的项目沿用项目的 404；看不到或不存在的模块 404 `planning.module_not_found`，链接 404 `planning.module_link_not_found`（链接所在的模块已删除也是）。

### 2.4 删除与清理

- 删模块：软删本行及其未删除的成员与链接，同一时刻、同一写者（同 M4 删工作项连带子行）；P2 不发事件。已归档的模块可以删（同迭代）。
- 项目、工作区删除：`planning.delete_rows` 由单表改为命名步骤 `cycles → modules → module_members → module_links`（父先子后，同 `issue` 的 `DeletionRowSteps`），每步按项目、id 升序、`FOR NO KEY UPDATE`、有界、沿用原 At/By，预算边界按当前步骤判断剩余（R16）。任务参数不变。
- 60 天物理清理：步骤 `module_links → module_members → modules → cycles`（子先父），每张表单语句分页；运行角色可执行。
- 守卫与夹具同步扩展：`project_deletion_test`、`workspace_deletion_test`、`purge_late_deletion_test` 以 SQL 种三张表的行并核软删；`purge_runtime_test` 的 planning 组加三张表；`purge_catalog_test` 与 `purge_indexes_test` 的表数；`purgeScaleTables` 加三张表，规模夹具以 SQL 种入。P2 以 `NERVE_PURGE_SCALE=1` 跑一次 `TestPurgeScale` 证明追踪器认得新语句，P8 在最终代码上重跑。

## 3. 局部裁定

- **R1 加锁之路**：照 P1 R1。`see`、`Locks`、`Reads` 由只认迭代改为认三种对象：迭代、模块、模块链接（链接先读出它的模块，再到项目）；不加锁预检时按“对象 → 项目”读，锁下按“项目 → 对象重读”。`cycle.*` 的写看 `cycle_view`（`planning.cycles_disabled`），`module.*` 的写看 `module_view`（`planning.modules_disabled`）。迭代的回答与测试不变。
- **R2 拒绝的次序**（同 P1 的分法）：结构 400 → 不读数据的 422（名字、说明含 NUL、`status` 不在六种、POST 的日期顺序、`member_ids` 超过 100 或重复、`sort_order` 非有限数、链接的地址与标题）→ 404 → 403（角色、OwnOnly 的写）→ 409 `project.archived` → 409 `planning.modules_disabled` → 409 `planning.archived`（改已归档的模块、在已归档的模块上写链接）→ 锁下才能判定的 422（PATCH 合并后的日期、负责人与成员的资格）→ 409 `planning.module_name_taken`、`planning.module_link_exists`（写入时）。
- **R3 归档与恢复**：只有状态为已完成或已取消的模块能归档，否则 409 `planning.module_not_closed`；恢复不看状态；已是目标状态时 200、不写（P1 R3）。
- **R4 PATCH 的日期**：`start_date`、`target_date` 各自可给、可为 `null`，各自可空；合并后两者都有时开始 ≤ 目标（422 指向给出的字段）。空 PATCH 或没有实际改动 200、不写；`member_ids` 与现有集合相同算没有改动。
- **R5 负责人与成员**：资格是项目中角色为管理员或成员的有效成员（同工作项负责人的 `CanAssign`），不合格 422 `lead_id`/`member_ids[i]` `not_allowed`。建立时检查负责人与全部成员；PATCH 只检查改成的新负责人和新加入的成员，留下的成员和未改的负责人即使已离开项目也保持（设计 §3.2“成员离开项目不回写模块”，同 M4 只检查新增的负责人）。`member_ids` 至多 100 个、不重复（同工作项的 `MaxAssignees`），整体替换：移出的软删，新加入的插入，留下的不动。
- **R6 OwnOnly**：一切写 403（P1 R6）。
- **R7 开关**：`module_view=false` 时模块与链接的一切写 409；读照常。`module_view` 默认 `false`，Go 夹具、清理运行角色夹具、e2e 先经接口打开它。
- **R8 名字唯一**：比较本项目未删除（含已归档）的模块，不分大小写；由唯一索引保证，适配器把这条约束的冲突映射为 409 `planning.module_name_taken`（项目标签的先例）。写入在项目行锁下，并发两次同名建立只有一次成功，有交错测试。
- **R9 归档参数**：照 P1 R9。
- **R10 权限矩阵**：在 `projectTables` 下加模块表（`{module_id}`、成员建立的 `Mine` 与他人的模块，`prepareMatrix` 打开 `module_view`）与链接表（`{link_id}`），列同迭代表。
- **R11 抓取**：建链接与改地址在写入事务里经端口直接投递 `planning.crawl_link {link_id, url}`（planning 不为链接发事件，不经订阅者）；改地址把 `metadata` 重置为 `{}`，只改标题保留。worker 只在 `(id, url)` 仍对应未删除的链接时写入（同工作项链接）；最多 3 次尝试；没有新配置，两个模块共用同一个出站客户端。
- **R12 共用代码的落点**：设计 §3.2 写的是“移到 `platform`”，但架构测试规定 `domain`/`app` 只能导入标准库、本模块内层与 `internal/shared`，`platform` 不能导入 `shared` 与业务模块，`platform` 的包之间互不导入（`archtest/rules_test.go`）。因此地址与标题检查和元数据类型放 `internal/shared/link.go`（先例 `shared.ContainsURL`、`NormalizeEmail`），抓取放 `platform/outbound/linkpreview`（`outbound` 自己的子包），返回自有的 `Metadata`，由 `bootstrap` 的一个适配器转成 `shared.LinkMetadata` 交给两个模块。仍是一份代码，工作项链接的行为与测试不变；设计 §3.2 的措辞随之改。
- **R13 `planning.archived`**：码不变，说明与文案改为迭代与模块通用。
- **R14 动作与路由的次序**：照 P1 R12：动作与规则在建表的同一 Task 加入，`Register` 在第一个模块接口的 Task 加入。
- **R15 删模块的连带**：成员与链接在同一事务软删（§2.4），列表与清理不必再连父表判断。
- **R16 删除任务的步骤**：照 `issue/app/delete_rows.go` 的命名步骤与剩余判断；P1 的迭代删除测试作为回归，不改它们的断言。

## 4. 验收标准

- [ ] A1 G9 建立：名字去空白、不分大小写唯一（含已归档、不含已删除）；状态六种、默认 `planned`；日期顺序；负责人与成员的资格（访客、非成员、已离开者 422 指向下标）；`member_ids` 去重与上限；并发两次同名建立只成一次。
- [ ] A2 G9 读与改：两种顺序、归档目录与默认列表互斥；改各字段、成员整体替换（新加入的检查、留下的不动、移出的软删）、换负责人、日期合并、空改动不写；已归档 409。
- [ ] A3 G13：只有已完成、已取消的能归档；重复归档与恢复 200 不写；恢复不看状态；删除只限管理员或建立者，连带成员与链接同一时刻软删。
- [ ] A4 G10 链接：增改删与列表；地址规整与校验同工作项链接；同一模块内重复地址 409；标题可空可为空串；已归档的模块 409；投递在写入事务内（回滚不投递）；元数据只在地址仍匹配时写入；改地址重置元数据。
- [ ] A5 开关与权限：`module_view` 关闭时写 409、读 200；矩阵（管理员、成员、建立者、访客、OwnOnly、非成员、私有项目）与 `permission_matrix_coverage_test` 一致；看不到的一律 404；每个声明的错误码在 planning 的 apitest 二进制里有回答。
- [ ] A6 删除与清理：项目、工作区删除后三张表以原 At/By 软删并可续跑；60 天清理有界、运行角色可执行、`TestPurgeScale` 认得新语句；全部守卫覆盖三张表；迭代的删除与清理测试不变且通过。
- [ ] A7 共用链接：`shared` 与 `linkpreview` 各一份；工作项链接的全部测试只改导入路径与类型名并通过；架构测试通过。
- [ ] A8 phase 为 `M6/P2`；新错误码有中英文案；输入交接写“处理结果（M6/P2）”；P1 的阶段记录 §7、spec A8 与设计 §14 已回填。
- [ ] A9 七项本地门禁、deep-review 修复闭环、准确分支 CI 与 noff 后 main CI 全绿、归档清理。

## 5. 后续责任

P3：`module_issues`、`planning.module_deleted`、`module_user_properties`、工作项与草稿的 `module_ids`；P4：模块的进度、分布、燃尽；P6：模块页面（含链接的元数据显示）；P8：规模测试重跑。

## 6. 风险

- 共用链接代码动到 `issue`：纯移动，以 `bootstrap` 的 11 个工作项链接测试文件与 `crawler` 测试为回归，单独一个 Task、先于模块的任何代码。
- `see` 改为三种对象动到 P1 的迭代：迭代的 app、http、组合与矩阵测试作为回归。
- 一次建三张表触发多道按目录驱动的守卫：建表与删除任务放相邻的两个 Task，第一个模块接口 Task 之前不做整段七项检查点。
