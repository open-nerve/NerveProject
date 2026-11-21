# M7 / P3d：最近访问

> 状态：草稿；预检 [re-92](../../../reviews/re-92-M7P3d预检.md) 修正后通过（处理见它的 §6），T1 提交。基线 main `9d26b1c3`（P3c 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H17、H19）、§3.9、§3.10、§4.1–§4.3、§5、§6.3、§9、§10、§12、§15 E。现状调研在台账 `research/backend.md`（后端，DP-n 指它 §12）与 `research/plane-e2e.md`（Plane、e2e、契约，写作 PE-n）；Plane 缺陷编号 D35–D39 指设计调研 `plane-facts.md` §6，D43、D44 是 `plane-e2e.md` §1.6 新记的。前一段 [P3c](P3c-favorites.md) 交付了收藏。下文 `nav/` = `server/internal/modules/navigation/`，`iss/` = `server/internal/modules/issue/`，`pj/` = `server/internal/modules/project/`，`bs/` = `server/internal/bootstrap/`，`pa/` = `plane/apps/api/plane/`。

## 1. 目标与边界

设计 §12 的 P3 拆成四段，P3d 是最后一段：**最近访问**（R1）。P3d：`IssuePriority` 搬进 `api/common.yaml`；navigation 加 `user_recent_visits` 与它的两个接口；每人每工作区 20 条的上限（事务级咨询锁、同一事务里物理删）；读时的可见过滤；issue 交给 navigation 的新端口（工作项的地点与按调用者可见的摘要），项目目标加标识；`issue.deleted` 的连带；删除任务的第三步；60 天清理的一步；守卫、组合测试、交错、e2e 接口故事。T1 另补 P3c 阶段记录 §9。

不做：页面（首页的“最近”、打开工作项详情与进入项目时各记一次、失败不打扰，都随 P7）；工作项与项目之外的类型（迭代、模块、视图、页，设计 §3.9、D37）；删除接口（设计没有）；工作区级的行（R2）。

**P3d 合入后 main 的样子**：两个接口可用；工作项删除、项目删除、工作区删除都带走最近访问；清理目录 37 张表（设计 §4.3 的终数）；网页没有最近访问的任何代码（M3/P8a 的 `5c128973` 已删首页小部件，`web/apps/web/core/components/home/home-body.tsx:13-24`），P7 接；用户看不到变化。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `api/common.yaml`、`api/modules/issue.yaml`、`iss/adapter/http/{issues,intake_list}.go` | `IssuePriority` 的纯搬移（R8）：schema 搬进 `common.yaml`，`issue.yaml` 的 7 处 `$ref`（`:2245`、`:2373`、`:2445`、`:2562`、`:2638`、`:2888`、`:2956`）改指它，Go 的 3 处 `gen.IssuePriority`（`issues.go:95`、`:105`、`intake_list.go:45`）改用 `apigen` 的；单独提交，行为与断言不变 |
| `server/migrations/sql/00054_navigation_user_recent_visits.sql` + schema 测试 | 表、CHECK、索引（§2.3） |
| `nav/` | 最近访问的领域、存储、用例、处理器；`issue.deleted` 的订阅者；删除任务的第三步；清理一步 |
| `iss/` | 新端口 `NavigationIssues`（§2.5、R3），接口与实现都在新文件（`iss/app/ports.go` 已 395 行） |
| `pj/` | `NavigationTarget` 加 `Identifier`（R4） |
| `server/internal/modules/access/` | `recent_visit.read`、`recent_visit.write` 的规则 |
| `bs/` | 接线与端口转换（`navigation_ports.go`）；守卫（§4） |
| `api/modules/navigation.yaml`、`api/openapi.yaml` | 契约（§2.4）；根标签 `navigation` 的说明加最近访问 |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql` | sqlc 条目、运行角色 |
| `web/` | 只有 `make gen` 的生成物（`schema.gen.ts`）；没有新码，文案不改 |
| `e2e/` | 删除夹具经接口记最近访问（新 `fixtures/recent-visits.ts`）；`asynchronousDeletionTable` 加一项；H17 的接口故事 |
| `docs/` | P3c 阶段记录 §9、plane-diff、交接、硬规则、设计 §3.9/§4.2/§4.3/§5/§12/§14、阶段记录 |

### 2.2 动作、角色与相对规则（设计 §3.10）

| 动作 | 级别 | 角色 | 用例里的相对规则 |
|---|---|---|---|
| `recent_visit.read` | 工作区 | 管理员、成员、访客 | 只列自己的；目标此刻看不到的不列（§2.5） |
| `recent_visit.write` | 工作区 | 管理员、成员、访客 | 只记自己的；目标须此刻看得到（§2.5） |

- 工作区访客能记、能列：设计 §3.10 的表（`M7-design.md:153`）明写三种角色，Plane 的列表也给访客（`pa/app/views/workspace/recent_visit.py:24`）。与收藏不同（收藏的访客 403）。
- 三种角色全收，两个操作都没有 403，契约不声明 `forbidden`（`workspace_issue.list` 与通知的列表同此：`server/internal/modules/access/domain/rules.go:183-187`、`api/modules/issue.yaml:404` 的 `listWorkspaceIssues` 声明 `[validation_failed, workspace.not_found]`）。
- 不是工作区成员（含已移除）：列表答 404 `workspace.not_found`；记录答目标那一种的 404（工作区由目标推出，§2.7）。
- 离开项目、被降为只看自己：行不删，读时过滤；重新加入即重现。

### 2.3 表 `user_recent_visits`（设计 §4.2，DP-1、DP-2）

| 列 | 定义 |
|---|---|
| `id` | `uuid PRIMARY KEY` |
| `workspace_id` | `uuid NOT NULL → workspaces ON DELETE CASCADE`（目标所在的工作区） |
| `project_id` | `uuid NOT NULL → projects ON DELETE CASCADE`（目标所属的项目；项目的访问为项目自己） |
| `user_id` | `uuid NOT NULL → users ON DELETE CASCADE`（照 Plane；v0 不物理删除账户，不触发） |
| `entity_type` | `text NOT NULL`，CHECK 两值 `issue`、`project` |
| `entity_identifier` | `uuid NOT NULL`（目标的 id，不是外键） |
| `visited_at` | `timestamptz NOT NULL` |
| `created_by_id`、`updated_by_id` | `uuid NULL → users ON DELETE SET NULL`（建的人是访问者；改的人是重访者或连带时删除的人） |
| `created_at`、`updated_at` | `timestamptz NOT NULL`（不设默认，漏给即报错，同 `00052`、`00053`） |
| `deleted_at` | `timestamptz NULL` |

- `project_id` 非空（R2）：两种目标都在项目里（Plane 的 `project_id` 只有工作区视图的访问为空，`pa/app/views/view/base.py:111-117`，Nerve 不记它）。所以删除任务没有工作区范围的一步。
- CHECK：`entity_type` 两值；`entity_type = 'project'` ⇒ `project_id = entity_identifier`（两列都非空，不用 `coalesce`）。
- 名字（schema 测试、`navigationNames` 都按它）：主键 `user_recent_visits_pkey`；外键 `user_recent_visits_{workspace_id,project_id,user_id,created_by_id,updated_by_id}_fkey`；CHECK `user_recent_visits_entity_type_check`、`user_recent_visits_project_id_check`；部分唯一 `user_recent_visits_user_id_workspace_id_entity_key ON (user_id, workspace_id, entity_type, entity_identifier) WHERE deleted_at IS NULL`；连带 `user_recent_visits_workspace_id_entity_identifier_idx ON (workspace_id, entity_identifier) WHERE deleted_at IS NULL`（设计 §4.3 点名的索引）；列表与裁剪 `user_recent_visits_user_id_workspace_id_visited_at_idx ON (user_id, workspace_id, visited_at DESC, id DESC) WHERE deleted_at IS NULL`（照 `issue_views_project_id_created_at_idx` 不把 `id` 写进名字，`00052_navigation_issue_views.sql:81`）；外键的不带条件索引 `user_recent_visits_{workspace_id,project_id,user_id}_idx`；清理 `user_recent_visits_deleted_at_idx ON (deleted_at) WHERE deleted_at IS NOT NULL`。
- 部分唯一的名字缩写为 `…_entity_key`：照列名全写的 `user_recent_visits_user_id_workspace_id_entity_type_entity_identifier_key` 有 73 字节，超过 PostgreSQL 标识符的 63 字节（`NAMEDATALEN − 1`），建索引时会被悄悄截断（只发 NOTICE），名字的守卫就钉不住；仓库里最长的现有名字是 56 字节（`user_favorites_user_id_entity_type_entity_identifier_key`）。迁移的注释写明 `entity` 指 `(entity_type, entity_identifier)` 两列。
- 记录的写是一条 `INSERT … ON CONFLICT (user_id, workspace_id, entity_type, entity_identifier) WHERE deleted_at IS NULL DO UPDATE SET visited_at, updated_at, updated_by_id`：冲突谓词与索引谓词逐字相同（硬规则 §5，P3c T3a 实测多一个条件即 42P10）；schema 测试钉住这条语句推得出这条索引。
- 没有 Plane 的 `entity_name`（改为 `entity_type`，CHECK 两值）；Plane 没有任何唯一（DDL 只有主键，`tools/plane-schema/plane-v1.4.2-schema.sql:3822-3826`，D36）。表名不以 `issue_` 开头：Go 的 `navigationTable`（`bs/issue_project_deletion_jobs_test.go:37-39`）与 e2e 的 `asynchronousDeletionTable`（`e2e/fixtures/assert/catalog.ts:98-117`）各显式加一项。
- 迁移数 53 → 54，清理目录 36 → 37，`deploy/runtime-grants.sql:29` 的 navigation 一行加这张表（含 `DELETE`：裁剪与清理都物理删）。

### 2.4 接口（设计 §5，PE-20）

| 操作 | 路由 | 回答 | `x-problem-codes` |
|---|---|---|---|
| `recordRecentVisit` | `POST /me/recent-visits` | 204（新记与重访同） | `validation_failed`、`issue.not_found`、`project.not_found` |
| `listRecentVisits` | `GET /workspaces/{slug}/recent-visits?entity_type=` | `{data: [RecentVisit]}` | `validation_failed`、`workspace.not_found` |

- 两条路径都在 `api/modules/navigation.yaml`、`tags: [navigation]`（M2 交接 §2：`/me/recent-visits` 属于最近访问自己的模块文件，`handoffs/M2-closeout.md:28`）；根 `api/openapi.yaml` 加两条 `$ref`（`:234-243` 之后），标签说明（`:35-36`）加“and his recent visits”；文件头注释加一句。
- `RecentVisitKind`：`enum [issue, project]`，请求体与查询共用（命名照 `FavoriteKind`，`navigation.yaml:603-609`）。
- `RecentVisitCreate`：`{entity_type: RecentVisitKind, entity_id: uuid}`，两个都必填，`additionalProperties: false`。没有 slug、没有 `project_id`：工作区与项目由目标推出（R7）。
- `RecentVisit`（DP-10 (a)）：`{id, entity_type, entity_id, visited_at, project: RecentVisitProject, issue: RecentVisitIssue | null}`，全部必填。`project` 总在：项目的访问是项目自己，工作项的访问是它的项目；`issue` 恰在 `entity_type` 为 `issue` 时非空（可空对象照 P1c 的 `oneOf: [{$ref}, {type: 'null'}]`，`api/modules/notification.yaml:396-405`）。
- `RecentVisitProject`：`{id, name, identifier, logo_props}`，`logo_props` 引 `../common.yaml` 的 `LogoProps`（`api/common.yaml:122`）。不带 Plane 的 `project_members`（网页的项目列表已带 `member_ids`，PE-4）。
- `RecentVisitIssue`：`{id, sequence_id, name, state_id, priority, assignee_ids}`，`priority` 引 `../common.yaml` 的 `IssuePriority`（R8）。不带项目的 id 与标识：同一行的 `project` 给出，工作项的编号是 `project.identifier` 与 `sequence_id`。不带 Plane 的 `type`、`is_epic`（Nerve 没有工作项类型，`docs/v0/plane-diff.md:23`）。
- `RecentVisitList`：`{data: [RecentVisit]}`，集合型、不分页，至多 20 条，`visited_at DESC, id DESC`。
- 没有新码：`issue.not_found`、`project.not_found`、`workspace.not_found`、`validation_failed` 的文案都已在 `web/apps/web/core/lib/error-messages.ts` 与两份 `errors.json`，契约的码集不变（`error-messages.test.ts` 不受影响，PE-21）。navigation 另声明 `domain.ErrIssueNotFound`，码与文案与 issue 的相同（`iss/domain/errors.go:44`），照它已有的 `ErrProjectNotFound`（`nav/domain/errors.go:20`，与 `pj/domain/errors.go:19` 逐字相同）。
- 结构错（缺字段、多字段、`entity_id` 不是 UUID、类型不对）是 400 `bad_request`（生成的严格处理器与 `bodyshape` 判，同收藏，`nav/adapter/http/favorite_refusals_test.go:93-118`）；`entity_type` 不是两值之一是 422 `validation_failed`、`entity_type` 的 `invalid_format`，请求体与查询相同（PE-7；严格处理器不校验枚举值，值由用例判）。查询的未知参数照 navigation 的集合型列表忽略，不加查询守卫。
- 最近访问的写不进写闸门、不声明 `server_busy`，用 `Deps.Tx`（无锁期限，同收藏，`bs/app_modules.go:93-101`，DP-18）。没有事件、没有动态。

### 2.5 可见、列出、上限与写入（设计 §3.9，DP-5、DP-7、DP-15、PE-9–PE-13）

**看得到**（记录与列出共用一个可见谓词，`recentSight`）：

| 目标 | 看得到 |
|---|---|
| `project` | 在调用者的 `Readable` 里：工作区的未删项目、他是有效成员的（含访客），**已归档的也算**（`server/internal/modules/access/app/readable.go:40-48`）；不是成员的工作区管理员不算（DP-7） |
| `issue` | 未删除（**已归档的、已归档项目里的也算**），所在项目在 `Readable` 里，`domain.Issue.HiddenFrom` 按他在那里的授权不藏它（只看自己的访客只看自己建的）；**在收集箱里的（Triage）也算** |

- 工作项的判定在 issue 里（新端口 `NavigationIssues`，照 P1c 的 `IssueRefs`，`iss/app/issue_refs.go:45-65`）：navigation 把 `Readable` 的授权交过去，issue 用唯一的谓词 `HiddenFrom` 判（硬规则 §4 访客规则；不在 navigation 或 SQL 里复制“创建者 = 调用者”）。
- **列出** = 看得到，并且工作项不在收集箱里（`InTriage`，R6），并且它的项目读得到（`ProjectTargets` 只读未删的项目）。所以记录收 Triage 条目（204），列表在它被接受之前不列它，接受之后再列；拒绝、标重复的条目一直在收集箱里（`iss/domain/issue.go:113-115`），一直不列。
- 开关不看：没有开关管最近访问。
- **与收藏的差别**（R5，登记）：收藏不收、不列已归档的目标（P3c spec §2.5），最近访问收、列（设计 §3.9 只说“看不到的不列”；`Readable` 与 `issue.read` 都含已归档的；Plane 照列，`pa/app/serializers/workspace.py:316-329` 用不滤归档的 `objects`）。项目归档不动最近访问（PE-17，Plane 同，`pa/app/views/project/base.py:429-441` 只删收藏）。
- **列表**：一个只读快照里读本人在这个工作区的全部未删行（有 `entity_type` 时只读那一种），`visited_at DESC, id DESC`；一次读 `Readable`、一次按 id 读工作项摘要、一次按 id 读项目（全部行的项目），按上面的谓词过滤，**过滤之后**取前 20（R8：上限之外的瞬时第 21 条也不给）。行数由上限限住，不分页。
- **摘要取此刻的**：工作项的名字、编号、状态、优先级、未删的负责人（按 id 升序，同 `GetIssues` 的子查询，`iss/adapter/postgres/queries/issues.sql:89-90`）；项目的名字、标识、图标。工作项不换项目（issue 的 SQL 没有改 `issues.project_id` 的语句），行的 `project_id` 就是工作项此刻的项目。
- **上限**：每人每工作区未删的行最多 20 条，两种合计，看得到与否都算（DP-15、PE-11：读时藏起的行——离开的项目、Triage 条目——重新可见时不会凭空多出）。记录写完之后，按 `visited_at DESC, id DESC` 排在第 20 条之后的行在同一事务里**物理删除**（设计 §3.9，Plane 软删、墓碑永存，D43）。裁剪跳过被别的事务锁住的行（§2.8）：那只能是正被工作项删除的连带或项目删除的任务软删的行，提交后它就不在未删的集合里；那个事务回滚时，这个人暂时有 21 条，直到他下一次记录。
- **新记**：`id` 新 v7；`workspace_id`、`project_id` 取目标的；`user_id`、`created_by_id`、`updated_by_id` 是调用者；`visited_at`、`created_at`、`updated_at` 是同一个时刻。
- **重访**（PE-13）：同一行；`visited_at`、`updated_at`、`updated_by_id` 改为这一次的时刻与调用者，`id`、`created_at`、`created_by_id` 不变（Plane 只改 `visited_at`，`pa/bgtasks/recent_visited_task.py:29-35`）。新记与重访都答 204（DP-8）。
- **GET 没有副作用**（总体设计 §3.2）：列表在只读快照里；工作项详情、按编号取、项目详情、各种列表都不写这张表（Plane 的十个 GET 都写，D38）。

### 2.6 连带、删除与清理（设计 §4.3，DP-16、DP-17）

- **工作项删除**：新订阅者 `navigation_forget_deleted_issue_visits`（navigation，`Types: ["issue.deleted"]`，与 `navigation_delete_rows_on_deletion`、`navigation_favorite_cascades` 并列，`nav/module_events.go:15-17`）。在发布者的事务里同步执行：先整体校验载荷——类型 `issue.deleted`、实体 `issue` 且有 id、工作区与项目非空、时刻非零、执行者为空或非零（同 `forget_deleted_issue_notifications`，`server/internal/modules/notification/app/deletion_events.go:15-23`）、没有变化（issue 的删除核心不带变化，`iss/app/delete_core.go:62`）；畸形的拒绝，删除回滚。然后软删**所有人**指向这个工作项的未删行（`workspace_id = W AND entity_type = 'issue' AND entity_identifier = I`，按 `id` 升序 `FOR NO KEY UPDATE`），`deleted_at`、`updated_at` 取事件的 `OccurredAt`，`updated_by_id` 取 `ActorID`（为空则空）；`visited_at` 不动。
- **覆盖每条删除路径**：`deleteIssue`（`iss/app/delete_issue.go:95`）与 `deleteIntakeIssue` 删未接受的条目（`iss/app/delete_intake.go:62`）都经删除核心 `deleteLocked`，它对子树的每个工作项各发一个 `issue.deleted`（`iss/app/delete_core.go:60-65`），子树、收集箱条目都在内；`deleteIntakeIssue` 删已接受的条目只删条目行、不发事件（`delete_intake.go:64`），工作项还在，它的最近访问也在。Nerve 没有批量删除工作项的接口（`issue.yaml` 的删除只有 `deleteIssue` `:1197`、`deleteIntakeIssue` `:1024`；Plane 的批量删除不清，D44）。项目删除时 issue 的分批任务不发 `issue.deleted`（`iss/app/delete_project_rows.go` 没有 `Publish`），由下一条的任务一步删。
- **项目、工作区删除**：`navigation.delete_rows` 的种类与载荷 `{scope, id, at, by}` 不变；项目范围的步骤由视图 → 收藏改为视图 → 收藏 → 最近访问（`nav/app/delete_rows.go:49-57`），第三步删 `project_id = P` 的全部未删行（项目自己的与它的工作项的），按 `id` 升序分批、`FOR NO KEY UPDATE`，时刻与删除者取任务的参数；只动未删的行（照 `DeleteProjectFavorites` 的 `r.deleted_at IS NULL`，`nav/adapter/postgres/queries/delete_rows.sql:33`）：工作项先删、项目后删时，连带早先软删的行保留连带写下的时刻与删除者，清理的时点不跟着变；工作区范围不加步（工作区的删除逐个列出它删的项目、各投一个项目范围的任务，`nav/app/delete_rows_on_deletion.go:23-29`）。立即不可见靠读时过滤（`Readable` 与 `ProjectTargets` 都只读未删的项目）。
- **归档**：工作项归档、项目归档与恢复都不动最近访问（R5）。e2e 的 P4（先归档再删）不加 `removedBefore`。
- **清理**：模块的清理在 `user_favorites` 之后加一步 `user_recent_visits`（`nav/module.go:45-54`），清理目录 36 → 37。
- **裁剪与清理不相交**：裁剪只删未删的行（`deleted_at IS NULL`），清理只删截止之前删的（`deleted_at < cutoff`）；存储测试钉住两者的集合不相交、两种次序结果相同（照 `TestTrimAndPurgeUseDisjointVersionSets`，`bs/issue_retention_sets_test.go:29`；issue 的 `TrimDescriptionVersions` 是同类的先例，`iss/adapter/postgres/queries/trim_versions.sql`）。

### 2.7 拒绝次序（P3c §2.7 的写法）

- **记录上每个 404 都与不存在的 id 逐值相同**（码、体、头）：工作项为 `issue.not_found`，项目为 `project.not_found`——不存在、已删除、在他不是成员的工作区、他看不到（不是项目成员、只看自己时别人的工作项）、项目已删而任务还没跑完，都是同一个回答，也都不写行。
- 记录：请求体 400（结构）→ 422（`entity_type`，纯函数，先于任何读）→ 预检（不开事务、不加锁）：读目标的地点（工作项经 `NavigationIssuePlaces`，项目经 `ProjectTargets`；没有 → 404）→ `recent_visit.write` 在它的工作区上的判定（看不到 → 404）→ 看得到（`Readable` + 工作项的可见；不 → 404）→ 事务：工作区 `FOR SHARE`（已删 → 404）→ 再判 `recent_visit.write`（→ 404）→ 项目 `FOR SHARE`（已删 → 404）→ 锁下重读地点与可见、重判（→ 404）→ 咨询锁 → 时钟 → 写 → 裁剪 → 204。
- 列：422（`entity_type`，先于工作区，DP-19）→ 404 工作区（slug 不对、他不是有效成员）→ 读（一个只读快照）。
- 交叉格：记录——坏的 `entity_type` × 不存在的目标 → 422；列——坏的 `entity_type` × 看不到的工作区 → 422；记录——看不到的目标 × 工作区访客 → 404（不是 403）。

### 2.8 锁序（DP-11 (a)、DP-12、DP-13、DP-14）

- **记录**：工作区 `FOR SHARE` → 项目 `FOR SHARE`（M3 约定二，工作区先于项目）→ 锁下重读、重判 → 本人与这个工作区的**事务级咨询锁** → 读一次时钟 → upsert → 裁剪。新的 `RecentVisitLocks`（navigation 的 app 里、新文件），复用收藏已有的 `decideWorkspace`、`allow`（`nav/app/favorite_locks.go:255-283`），工作区、项目的端口与收藏、视图共用；`FavoriteLocks`、视图的 `Locks` 不动——`FavoriteLocks.run` 写死了视图行与收藏行的锁（`nav/app/favorite_locks.go:167-218`），视图的 `Locks.run` 写死了视图的开关，最近访问的最后一把锁是咨询锁，另起一个编排，不复制它们。
- **为什么项目 S 够、不锁工作项行**：工作项的每条删除路径在整个事务里持项目的 `FOR NO KEY UPDATE`（`Locks.run` 在一个事务里先 `lockAndDecide`，`iss/app/lock.go:103-120`、`:179-183`；语句是 `LockIssueProject`，`pj/adapter/postgres/queries/issue_ports.sql:1-11`），与项目 S 冲突；项目删除改写项目行、工作区删除先锁工作区再按 id 锁项目，也都与之冲突。删除先提交：记录等到项目 S 之后锁下重读，看到工作项或项目已删 → 404、不写。记录先提交：删除等它，之后连带在删除的事务里软删刚写的行，或项目的任务删它。两种次序都不留指向已删目标的活行（P1a 的工作项行 S 不够：navigation 的任务不等 issue 的任务，`nav/app/delete_rows.go:31-37`，没有项目 S 的记录会在任务扫过之后插进已删项目下的活行，DP-11）。成员关系与角色的变更锁工作区或项目行，同样串行。
- **咨询锁**：键由本人与工作区两个 id 在 SQL 里推出，加锁是存储的一个方法 `LockRecentVisits`：`pg_advisory_xact_lock(hashtextextended('user_recent_visits/' || sqlc.arg(user_id)::uuid::text || '/' || sqlc.arg(workspace_id)::uuid::text, 0))`（DP-13；预检 re-92 用仓库钉住的 sqlc 实测收下，生成 `LockRecentVisits(ctx, {UserID, WorkspaceID})`）。键的公式只此一处，测试经这个方法取锁。两个不相干的人撞了键只是多一次串行，不影响正确。它让同一个人在同一工作区的记录一个接一个，上限在并发下守得住（Plane 的 `== 20` 在并发下失守，D36）。
- **不变式：持咨询锁的事务从不等行锁**（DP-12，进硬规则 §4）。拿到咨询锁时，它要等的父行（工作区、项目）已经在手；upsert 冲突的那一行只能是本人对这个目标的未删行，同一人的别的记录被咨询锁挡在外面，这个工作项的连带要项目的 N 锁、被本事务的项目 S 挡住，项目的任务只在项目删除之后跑、而本事务锁下读到它未删，清理只碰已删的行——所以 upsert 不等；裁剪 `SKIP LOCKED`，不等。唯一可能的等待是外键检查对账户行的 `FOR KEY SHARE`，今天没有路径与它冲突（没有删除账户行的路径，identity 只取 `FOR NO KEY UPDATE`，同 P1c 的登记，硬规则 §4）。所以等咨询锁的人等的是一个不会再等的人，不成环。
- **裁剪为什么必须 `SKIP LOCKED`**：工作项删除先取齐项目锁（`iss/app/lock.go:179-183`），做完全部行步骤，最后才对子树逐个发 `issue.deleted`（`iss/app/delete_core.go:53-65`），navigation 又是它最后一个订阅者（`bs/events.go:165-175`）；连带拿到访问行之后不再等任何项目，所以调研 DP-12 写的那条经“别的项目的锁”的环走不到。真实的环在裁剪与连带之间：本人第 20 条之后有两条以上的行（回滚留下的瞬时行累积时），分属同一子树的两个工作项，且行的 `id` 次序与子树的次序（工作项 `id` 升序，`iss/adapter/postgres/queries/delete_issue.sql:1-11`）相反——按 `id` 等待的裁剪锁住一条、等另一条，连带锁住另一条、等这一条，40P01，其一答 500。平时只多出一条，等待的裁剪不成环，但会让持着工作区 S、项目 S 与咨询锁的记录挂在一个慢的删除后面，同一人的后续记录又排在咨询锁后面。`SKIP LOCKED` 把两种都去掉。
- **裁剪**：一条语句，`MATERIALIZED` 的 CTE 先取本人在这个工作区按 `visited_at DESC, id DESC` 的前 20 条（不加锁，本事务持咨询锁，同一人不会同时插入），再取其余未删行 `ORDER BY id FOR UPDATE SKIP LOCKED`，`DELETE … USING` 它们（照 P1c 全部已读的写法与理由，`server/internal/modules/notification/adapter/postgres/queries/mark_read.sql:1-36`：等待会与逐个工作项软删的子树删除成环，见上一条；`IN` 子查询会在每次重扫时再锁）。上限 20 是领域的常量，作参数传给语句。
- **`user_recent_visits` 的写者与锁序**（T15 照 P1c 为 `notifications` 的写法登记进硬规则 §4）：

| 写者 | 持有的锁 | 对本表行的锁 |
|---|---|---|
| 记录（upsert） | 工作区 S、项目 S、咨询锁 | 本人对这个目标的一行，从不等 |
| 记录（裁剪） | 同上 | 本人在这个工作区第 20 条之后的行，`FOR UPDATE SKIP LOCKED`，从不等 |
| 工作项删除的连带 | 项目 N（发布者的事务） | 子树逐个工作项，每个工作项的全部行按 `id` 升序 `FOR NO KEY UPDATE`（整体不是全局升序）；可能等裁剪锁住的行，裁剪的持有者从不等 |
| 项目删除的任务 | 无（项目已删） | `project_id = P` 的行分批按 `id` 升序 `FOR NO KEY UPDATE`；同上 |
| 60 天清理 | 无 | 只删已删的行，与上面三者的行不相交 |

- 最近访问是工作区级的写：矩阵行只用工作区的列（用了项目级的列，`writesOnAProject` 就按操作 id 把它整体算作项目上的写，`bs/project_write_locks_test.go:150-169`），不进项目写锁表；另写锁序测试 `TestRecordingARecentVisitTakesItsLocks`（§4）。

## 3. 局部裁定

- **R1** 范围：P3d 只做最近访问（约 15 个 Task）；关键词的 phase 仍是 `M7/P3`（`tools/keywords.json:2`）。T1 补 P3c 阶段记录 §9：限定复审 Ready to merge（三条次要在 `d2511e56` 修）、`final.sh fin` 七项全过、最终分支 CI 44945103629 全绿（第 1 路 bootstrap 938.8 秒，期限的 78.2%）、`--no-ff` 合入 `9d26b1c3`、main CI 44947828240 全绿（第 1 路 752.5 秒，62.7%）、归档 `.superpowers/sdd/M7-P3c-integration-20261122`（3507 个文件，未知 0）；设计 §14 的 P3c 一行改为已完成、加 P3d 一行。理由：P3c 的阶段记录在合并之前提交，集成的数字只能由下一段补。
- **R2** 表（§2.3）：`project_id` 非空，没有工作区范围的删除步（DP-1）；CHECK 两条；部分唯一的名字因 63 字节的上限缩写；连带、列表与裁剪各一条部分索引（DP-2）；`workspace_id`、`project_id`、`user_id` `CASCADE`，审计列 `SET NULL`，`entity_identifier` 不是外键（设计 §4.2、§15 D2 的同一做法）。
- **R3** issue 的新端口 `NavigationIssues`（DP-3 (a)）：`NavigationIssuePlaces(ids)` 读未删工作项的工作区与项目（已归档的也读，不加锁、不判定），给记录的预检知道判定哪个工作区、锁哪个项目；`NavigationIssues(reader, grants, ids)` 答其中他看得到的（§2.5），每个带项目、编号、名字、状态、优先级、未删的负责人、是否在收集箱里。判定在 issue：授权的项目、`HiddenFrom`；收集箱的标记由 issue 的 `InTriage()` 判，行经 `LEFT JOIN intake_issues`（未删的条目行，同 `GetIssues`，`iss/adapter/postgres/queries/issues.sql:99`）带 `Intake`——这是第四条行带 `Intake` 的查询，硬规则 §4 的 P2a 一段（“只有 `GetIssue`、`GetIssues`、`SubIssues` 三条”，`03-rules-and-pitfalls.md:82`）随 T15 改。接口在新文件 `iss/module_navigation_issues.go`，实现与它的读端口在 `iss/app/navigation_issues.go`（`iss/app/ports.go` 已 395 行），SQL 在新的 `navigation_issues.sql`。理由：设计 §6.3 说 issue 给 navigation 的是“摘要与可见判定”；只此一处有访客规则。
- **R4** 项目目标加标识（DP-4 (a)）：`pj/app.NavigationTarget`（`pj/app/navigation_ports.go:22-27`）与 `nav/app.ProjectTarget`（`nav/app/favorite_ports.go:35-40`）同一个 Task 各加 `Identifier string`、同一位置，`NavigationTargets` 的 SQL 多选 `identifier`（`pj/adapter/postgres/queries/navigation_ports.sql:18-24`）——bootstrap 按类型直接转换（`bs/navigation_ports.go:78-81`），两边字段必须相同。收藏不用它（P3c 阶段记录 §3“标识随 P3d”）。
- **R5** 已归档的工作项、已归档项目与其中的工作项：能记、照列（DP-5、PE-9），与收藏不同，登记；项目归档不动最近访问（PE-17）。理由：设计 §3.9 只排除看不到的，`issue.read` 与 `Readable` 都含已归档的，P1c R4 的“读得到就该认得出”。
- **R6** 收集箱里的工作项（PE-10 (a)）：能记（`issue.read` 读得到它，详情带 `intake`），在收集箱里时不列，接受之后再列；藏起的行照占上限。理由：设计 §3.6 条目“不出现”在各种列表里；Triage 状态不在状态接口里（`M7-design.md:108`），首页的行渲染不出它的状态。记录不加 Triage 的守卫（硬规则 §4 P2a：“以后给工作项加的写，要写明它对在收集箱里的工作项加不加这个守卫”；记录不是工作项的写，写明不加）。backend 调研的 DP-6 (b)（一个谓词、照列）不取。
- **R7** `POST /me/recent-visits`：204（DP-8）；看不到、不存在答目标自己的 404，与从未存在的 id 逐值相同（DP-9 (a)），没有新码；拒绝次序 400 → 422 → 查找（§2.7）；工作区由目标推出；工作区访客照设计 §3.10 能记；重访三列同一时刻（PE-13）；没有事件、动态、写闸门，`Deps.Tx`、没有 503（DP-18）。
- **R8** `GET /workspaces/{slug}/recent-visits`：集合型，形状 §2.4（DP-10 (a)，P1c 的可空对象）；`visited_at DESC, id DESC`，先过滤后取 20；坏的 `entity_type` 422 先于工作区（DP-19、PE-7）；GET 没有副作用；`RecentVisitKind`；`IssuePriority` 纯搬移到 `common.yaml`，单独一个 Task、一个提交，issue 的 3 处 Go 跟随（PE-5 (a)，P3b R5 搬 `LogoProps` 的先例 `ea92002a`）——第二个模块开始回答优先级，词表只留一处；搬移之后打包的名字不变，网页的 `IssuePriority` 类型照旧（`web/apps/web/core/lib/issue-order.test.ts:3`）。契约都在 `navigation.yaml`（PE-20）。
- **R9** 记录的锁序与上限（§2.8，DP-11 (a)–DP-15）：工作区 S → 项目 S → 锁下重判 → 咨询锁 → 时钟 → upsert → 裁剪；持咨询锁的从不等行锁；裁剪物理删、`SKIP LOCKED`；上限数未删的全部行；每个写者的锁序进硬规则 §4。
- **R10** 连带与删除（§2.6）：新订阅者 `navigation_forget_deleted_issue_visits`（DP-16 (a)，另起名字，与收藏的连带分开）；删除任务项目范围的第三步（DP-17 (a)，同种类、同载荷，P3c R11 的做法）；清理在收藏之后；裁剪与清理不相交由存储测试钉住。
- **R11** 守卫：权限矩阵两个操作只用工作区列（DP-20，P3c T9 的先例与理由）；列表一行是带 `prepare` 的 `write` 行（按格准备的行必须是 `write` 行，`bs/permission_matrix_coverage_test.go:84-86`；GET 的先例是 `listIssues` 的已归档两行，`bs/permission_matrix_issue_lists_test.go:15`、`:17`）：准备用 SQL 往格的副本里插一条本格调用者对 `acme/public` 的项目访问，访客列出它，不是那个项目成员的管理员、成员答空——矩阵因此查到按角色的读时过滤，其余的读时过滤归组合测试。代价是 6 个副本，按现有 `write` 行的折算第 1 路不到 2 秒，不碰任何封顶的文件（准备在新文件里，`prepareMatrix` 不改）。记录三行在各格自己的副本上（它写）；锁序测试另写。
- **R12** Plane 的差异（T15 的 plane-diff）：D35（按访问时刻排，Plane 按建立时刻）、D36（咨询锁与部分唯一守住上限与“一个目标一行”）、D37（只记两种）、D39（读时过滤，比设计写的更宽：只看自己的访客、私有化的项目、Triage、已归档的 Plane 都照列）、D43（Plane 淘汰软删、60 天硬删不含此表，墓碑永存；Nerve 淘汰物理删、删除的连带软删、60 天清理）、D44（Plane 的批量删除与工作区视图删除留行、先取 20 再解析，首页不足 20；Nerve 每条删除路径都经 `issue.deleted`、先过滤后取 20）。设计 §3.9 的“Plane 相同”只对一半：Plane 单个删除时**硬删**所有人的行（`pa/app/views/issue/base.py:716-727`）、批量删除不清（`:773-797`），T15 改设计的措辞。
- **R13** e2e：删除夹具只经接口造行，`furnishIssueRows` 由调用者记项目与它的工作项 `fix`（PE-15）；H17 一个接口故事两轮（PAT、会话），每轮用自己的工作区（没有删除接口，复位不了，PE-18）；工作项删除的 e2e 证据放在 H17（I17 只沿外键走，`entity_identifier` 不是外键，PE-19）。
- **R14** bootstrap 的新顶层名只有四个，都在第 0 路（`printf %s 名字 | cksum` 取余 100，桶 0–45 是第 0 路，`tools/go-test-shard.sh:18-20`）：`TestRecentVisitsThroughTheApp` 36、`TestRecentVisitsFollowDeletedWorkItems` 4、`TestRecentVisitWritesInterleave` 33、`TestRecordingARecentVisitTakesItsLocks` 17。第 1 路在 P3c 的最终分支 CI 到了期限的 78.2%，新名一个也不放进去。

## 4. 验收标准

- `IssuePriority` 的搬移（R8）：`make gen-check` 无差异、`lint-go`、issue 的处理器测试不改断言；检查点 0 的本机七项与分支 CI 全绿（窗口打开之前最后一次全绿）。
- 迁移的 schema 测试 `TestUserRecentVisitsSchema`（新文件 `server/migrations/navigation_recent_visits_schema_test.go`）：列、可空、无默认、每个名字（§2.3）；两条 CHECK 各一个只破它的反例；部分唯一在软删之后能重建；`ON CONFLICT … WHERE deleted_at IS NULL` 推得出部分唯一（一条真实的 upsert）；外键的连带（项目、工作区、账户）与 `SET NULL`。
- 单测：领域（两种类型、`entity_type` 的 422、上限常量）；用例（拒绝次序、锁的次序与锁下重判、可见谓词的每一格、列出与记录的差别、重访的三列、时钟在最后一把锁之后读一次；过滤之后取 20——假存储交出 21 条未删行、藏起的一条排在前 20 里，答前 20 条可见的、含第 21 条，因为不超过 20 条时“先取 20 再过滤”与它答得一样；锁下重读工作项的两层——地点的重读与可见的重读都看已删，变异两层一起去掉，留只去一层的对照仍答 404）；订阅者（载荷的每个畸形各被拒、执行者为空、只动这个工作项的、所有人的）；删除任务的三步次序与各自的游标。
- 存储（`nav/adapter/postgres`）：
  - 列表：次序（同一时刻按 `id` 打破，在事务里关掉索引扫描再读一遍，硬规则 §5）；只读本人的、只读这个工作区的（两个人访问同一目标、同一人在两个工作区各有行）；`entity_type` 的筛选（只读那一种）。
  - upsert 新记与重访（`id`、`created_at`、`created_by_id` 不变）。
  - 裁剪：SQL 种 23 条未删行再记一次，删到恰 20 条（删掉第 20 条之后的全部，不只最旧的一条——Plane 的 `== 20` 只删一条）；21 条（回滚留下的样子）再记一个新目标 → 恰 20 条（§2.5 的瞬时第 21 条在下一次记录愈合）；前 20 只数未删的（几条更新的已删行不占位置，活的行不被提前删）；物理删；不碰别人与别的工作区；跳过被锁的行且不等（`lock_timeout` 之内答完）。
  - 咨询锁：同一人同一工作区的两个事务串行（第二个等，没有应用在跑，`pgtest.WaitForLockWait` 可用）；别人的、同一人别的工作区的不等（`lock_timeout` 之内取到）——键漏掉人或工作区的变异由它杀。
  - 连带按 `id` 升序、只动这个工作项与这个工作区的；任务的一批，一条早先已软删的行的时刻与删除者不变；清理；**裁剪与清理不相交**（新测试 `TestTrimAndPurgeOfRecentVisitsTakeDisjointRows`）。
- 端口：project 的 `NavigationTargets` 带标识；issue 的两个方法——地点读已归档的、不读已删的；摘要不读已删的，按授权、`HiddenFrom`（只看自己的授权下别人的不在、自己的在、创建者未知不在）、已归档的在、Triage 的在且 `InTriage` 为真、接受之后为假、负责人只含未删的、别的工作区的不在。
- 守卫：权限矩阵两个操作，只用工作区列——列表一行（R11，`write` 加 `prepare`）`inWorkspace(200, 200, 200)`，检查访客的回答恰是 `acme/public` 那一条，管理员、成员的回答是空集合；记录三行，各格在自己的副本上：“一个项目，作为它的成员”（`prepare: prepareDraftPublisher` 把 acme 的管理员、成员加进 `acme/public`，`bs/permission_matrix_drafts_test.go:42-56`）→ 204 / 204 / 204 / 404 / 404 / 404；“一个项目，不作为它的成员”（不准备）→ 404 / 404 / 204 / 404 / 404 / 404（种子里管理员、成员一个项目都读不到，访客读 `acme/public`，`bs/permission_matrix_notifications_test.go:20-37`）；“别人的工作项”（准备）→ 204 / 204 / 404（`acme/public` 不给访客看全部，只看自己，`HiddenFrom` 经端口生效）/ 404 / 404 / 404。锁序测试（§2.8）。动作 ↔ 规则、删除目录（Go 守卫、`navigationTable`、e2e 目录）、清理目录与名单。
- 锁序 `TestRecordingARecentVisitTakesItsLocks`（照 `TestFavoriteWritesTakeTheirTargetsLocks`，`bs/navigation_favorite_locks_test.go:115`），四段，工作项与项目两种目标都走一遍：
  1. 工作区被 `FOR NO KEY UPDATE` 扣住时记录在等（`requestBlocked`），不持项目、没有咨询锁、没写表（`requestWrittenTables`）；
  2. 项目被扣住时持工作区 S（`requestRowLock`），没有咨询锁、没写表；
  3. 咨询锁被一个持有者事务经存储的 `LockRecentVisits` 扣住时，记录等在 `LockRecentVisits`，持工作区 S 与项目 S、没写表。这一段等待用 `pgtest.WaitForStatementLockWaitBehind`，传持有者的连接与语句名 `LockRecentVisits`（`pg_blocking_pids` 对咨询锁同样成立，`server/internal/platform/postgres/pgtest/lockwait.go:83-92`），再由请求探针 `readRequestLock` 认出是这个请求自己的等待（它认任何未授予的锁，`bs/request_locks_test.go:193`）。`pgtest.WaitForLockWait` 不用：它数库里任何一个等锁的后端，应用的 River 任务也算（`lockwait.go:14-31`），只适合没有应用在跑的存储测试；`requestBlocked` 也不用：它只认行与表的等待（`bs/request_locks_test.go:141-150`）。这一段是“不取咨询锁”的确定杀法：变异体不等、直接 204；
  4. 本人第 20 条之后的一行被 `FOR UPDATE` 扣住时记录不等、答 204，那一行还在。
- 组合测试（真实的 app）：
  - H17 `TestRecentVisitsThroughTheApp`：两种各记一次（204；库里一行，工作区与项目由服务端推出，写者本人，三个时刻相同）；重访（`id`、`created_at`、`created_by_id` 不变，三列同一时刻，次序跟着变）；筛选 `issue`、`project`、省略；只列自己的、只列这个工作区的（两个人记同一目标，各自的列表只有自己那一行；同一人在两个工作区各记一次，各列各的）；摘要整值比较且取此刻的（改名、优先级、负责人之后跟着变，已删的负责人不在）；GET 没有副作用（读工作项详情、按编号、项目详情、项目列表、工作区列表、最近访问列表之后，本表的行逐列不变）；一个 404（不存在、别的工作区、私有项目的非成员、公开项目的非成员（它 `GET /projects/{id}` 答 200）、不是项目成员的工作区管理员、只看自己时别人的工作项），各与随机 id 的回答逐值相同、不写行；工作区访客能记能列；H19 的项目级格：工作区管理员兼项目访客在 `guest_view_all_features` 关着时同只看自己、开着时看全部；离开项目后不列、行在、再记 404，重新加入又列；已归档的工作项、已归档项目与其中的工作项能记、照列（R5）；收集箱里的条目能记、不列，接受之后列，拒绝的一直不列（R6）；上限：第 21 次记录物理删访问最早的那一行（不是建立最早的，没有墓碑），藏起的行也算数，别的工作区的不动；拒绝次序的交叉格（§2.7）；删除窗口（`holdNavigationDeletion` 扣住任务，`bs/navigation_fixture_test.go:98-105`；它扣的是项目的一条活的视图，没有视图时 `Scan` 失败，而任务的步骤是视图 → 收藏 → 最近访问，扣住视图才停得住第三步——所以那个项目开着视图并有一条视图）：项目已删时列表不列它与它的工作项、记录答 404 与不存在的逐值相同。
  - `TestRecentVisitsFollowDeletedWorkItems`：两个人都访问过的工作项，三层子树删除（删除者不是创建者）→ 两人对子树每个工作项的行在删除的时刻、由删除者软删，`visited_at` 不变，不再列，再记 404；别的工作项、别的工作区的行不动；`deleteIntakeIssue` 删未接受的条目带走它的行，删已接受的条目只删条目行、行照旧；项目删除、工作区删除经任务删（时刻、删除者取事件的；工作项与项目两种行都删）；先删工作项、再删它的项目时，工作项的行保留工作项删除写下的时刻与删除者（任务的第三步只动未删的，§2.6）；项目归档与恢复、工作项归档不动。
- 交错 `TestRecentVisitWritesInterleave`（扣锁之前 `waitForStartupRuns`、`waitForFanouts`；配对共用准备好的库、分组跑）：
  - 记录 × 工作项删除（两种次序；目标是子树里的子工作项也一格，它证明连带逐个工作项发）、× 项目删除、× 工作区删除：两个写都经真实的 app，夹具扣住两个写都要的行、次序由排队决定；不留指向已删目标或已删项目下的活行，不答 500。
  - × 项目成员移除（`removeProjectMember`，项目 N，`pj/adapter/postgres/queries/members.sql:53-54`）：移除先提交 → 记录在项目 S 之后锁下重判看不到，404、不写行；记录先提交 → 移除等它，行在、此后的列表不列它。× 项目归档：行不动，两个都 2xx。
  - 同一人有 19 条时同时记两个新目标；同时记同一目标两次。构造：一个持有者事务经存储的 `LockRecentVisits` 扣着咨询锁，两个请求经各自的请求探针都看到等在 `LockRecentVisits`，放开后恰 20 条、两个 204；同一目标一行、两个 204、没有 23505。这两格是正向的结果，不是“不取咨询锁”的杀法：部分唯一上的 `ON CONFLICT … DO UPDATE` 本身并发安全，变异体也是一行；不取咨询锁时两个请求同时放行、谁先提交是竞态（确定的杀法是锁序测试的第三段与用例的调用序列）。
  - 裁剪 × 别的事务扣着的一行：那个事务是测试的 SQL 事务，经存储的 `DeleteIssueRecentVisits` 软删本人第 20 条之后的那一行，开着不提交（经 app 扣不住：navigation 是 `issue.deleted` 的最后一个订阅者，`bs/events.go:165-175`，连带拿到行锁之后只剩提交）。记录不等、答 204；那个事务提交 → 恰 20 条。回滚另一格 → 库里 21 条未删、列表 20 条、他下一次记录之后 20 条（§2.5 的瞬时第 21 条，也是存储测试“愈合”的组合层证据）。
  - 裁剪 × 子树删除：扣住子树第二个工作项的一条通知，把删除停在两次连带之间（照 `TestMarkAllReadPassesOverTheNotificationsAWorkItemsDeletionHolds` 的构造，`bs/notification_mark_all_read_interleaving_test.go:82-112`；删除按工作项 `id` 升序对子树逐个发 `issue.deleted`，`iss/app/delete_core.go:60-65`）；记录的目标在删除所在项目之外，否则先被项目 S 挡住。**不等的一格**：第 20 条之后恰一条、是第一个工作项的访问，删除停住时记录立刻答 204（照 `answeredUnlessWaiting`，`bs/notification_mark_all_read_interleaving_test.go:50`），放开后删除 204、本人 20 条——“去掉 `SKIP LOCKED`”在这里等住。**死锁的一格**：用 SQL 种成记录之后 22 条，第 20 条之后的两条分属子树的两个工作项、行的 `id` 次序与子树相反（经接口造不出：每次记录都裁到 20）；先收两个回答、再断言：两个都 204、本人 20 条、记录没等——变异体在这里 40P01、其一答 500（§2.8 的环）。
- e2e：删除夹具经接口记最近访问（`furnishIssueRows`）；P4（两条）、W2、W3（两个文件）、I21 转绿；H17 的接口故事 `e2e/stories/collaboration/h17-recent-visits.spec.ts`（PAT 与会话两轮，各用自己的工作区），数据库断言。
- 每个性质至少一个能让测试失败的变异（plan 的变异表）。
- 测试时间：四个新顶层名都在第 0 路（R14），预估本机 H17 约 3–6 秒、连带约 3–5 秒、交错约 6–12 秒、锁序约 2–4 秒；第 0 路另跑 navigation、issue、project、access、migrations 各包的新测试。第 1 路只因已有守卫变重（`TestPermissionMatrix` 桶 46 多 24 格、每格各起一个 app——列表一行也是 `write`，R11；清理与删除守卫各多一张表），交错约 6–12 秒的预估含死锁的一格。按作业的 Test 步实测，每路增量 ≤ 90 秒；第 1 路另按 §6 的两条线。

## 5. 后续责任与交接

- P7：首页的“最近”（工作项、项目两种行，状态经工作区的状态目录解析，已归档项目不在项目列表里时不显示成员头像，PE-4）；打开工作项详情、进入项目时各记一次，失败不打扰；预览、按编号的页、项目内各子页是否各记，由 P7 定（Plane 是都记，PE §2.3）。收集箱页打开条目时不记，只在工作项详情页记：藏起的行照占上限（R6），分诊的人连着打开 20 个待处理条目，他的 20 条就全是不列的行，首页的“最近”变空，直到再访问别的；拒绝、标重复的条目永远不列，占位到被挤掉为止。
- 本段写“处理结果（M7/P3d）”的交接（都保持 open 至收尾，设计 §11；页面部分随 P7）：
  - **M1-P2-trim-content**：最近访问的接口没有 `page`，只有 `issue`、`project`（`RecentVisitKind`）；“首页的最近访问目前读取 Plane 的 `…/recent-visits/`”已过时——首页小部件随 M3/P8a 的 `5c128973` 删除，首页随 P7 读新接口。
  - **M2 §1**：最近访问的列表是集合型、不分页，至多 20 条，按 `visited_at DESC, id DESC`；目标的摘要与可见经端口取（issue 的 `NavigationIssues`、project 的 `NavigationTargets`、access 的 `Readable`），navigation 的 SQL 不联表别的模块的表。
  - **M2 §2**：`/me/recent-visits` 在 `api/modules/navigation.yaml`（不在 `identity.yaml`）；`WorkspaceService.fetchWorkspaceRecents` 已随 `5c128973` 消失（`git grep 'fetchWorkspaceRecents' -- web` 为空）；新的 SWR 键随 P7。
  - **M2 §5**：`user_recent_visits` 的外键去向：`user_id → users` `CASCADE`（v0 不物理删除账户，不触发），`created_by_id`、`updated_by_id` `SET NULL`，`workspace_id`、`project_id` `CASCADE`，目标不是外键（`entity_identifier`），删除项目、工作区经 `navigation.delete_rows` 软删。navigation 的三张表至此都写明了去向；照设计 §11，输入交接保持 open 至收尾关闭。
  - **M4 M7-04**：“删除工作项时清掉最近访问”= navigation 的订阅者在删除的事务里软删所有人指向它的行，子树逐个、收集箱条目的删除同此（§2.6）。
- M3 §1 的首页“最近”小部件是页面，随 P7，本段不写。

## 6. 风险

- **第 1 路的余量**：P3c 的最终分支 CI 第 1 路 bootstrap 938.8 秒，期限（20 分钟）的 78.2%；同一份代码 main CI 是 752.5 秒、62.7%，差 186 秒在 runner。P3d 不加第 1 路的顶层名，只让已有守卫变重（矩阵多 24 格，R11）。两条线（主控裁定）：
  - **本机停止线**：检查点 0 在本机实测的第 1 路作 P3d 的基线（不用 P3c 的数）；之后本机第 1 路比它多出 30 秒以上，先停下查原因、报告。
  - **CI 的八成线**：分支 CI 第 1 路的 bootstrap 超过 960 秒（期限的 80%）是常设的报告规则，不是停止线：过线时主控向负责人报告，附 runner 的分析（第 0 路非 bootstrap 各包的时间、与本机的对照）；只有本机也涨过了停止线才停。
  - 加路、加片、改期限都要负责人同意（设计 §9）。
- **项目 S 的争用**：每次记录短暂持项目 S，与这个项目的工作项写（项目 N）互等毫秒级。页面的记录不等回答（P7），可接受；不加写闸门，所以没有 503。另一面（已知风险，未在库上验证，不加测试）：按 PostgreSQL 的实现，后来的共享行锁不在等待中的 `NO KEY UPDATE` 后面排队，记录又比收藏频繁得多（每次打开页面一次）；同一个项目上的记录若持续重叠，工作项的写会等到写锁期限、答 503。v0 的量上不现实。
- **瞬时的第 21 条**：裁剪跳过的行所在的事务回滚时出现，到他下一次记录为止（存储测试的“愈合”与交错的回滚格钉住）；列表照样至多 20 条。多出两条以上时裁剪若等待会与连带成环（§2.8），`SKIP LOCKED` 去掉它。
- **删除图与守卫面广**：红窗口写进 plan，各带测试名；e2e 六条红到夹具的 Task。
- **一个订阅者的失败让删除失败**：连带在发布者的事务里同步执行（与通知、资产的同类订阅者同）；严格校验、按 `id` 升序加锁。
