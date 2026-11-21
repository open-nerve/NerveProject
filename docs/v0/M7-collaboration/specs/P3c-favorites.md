# M7 / P3c：收藏与三种归档事件

> 状态：spec 经预检 [re-91](../../../reviews/re-91-M7P3c预检.md) 修正后通过。基线 main `1168fd25`（P3b 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H15、H16）、§3.8、§3.10、§4.1–§4.3、§5、§6.1、§6.3、§9、§10、§15 D1/E。现状调研在台账 `research/backend.md`（后端，DP-n 指它 §12）与 `research/plane-e2e.md`（Plane、e2e、契约，DP-n 写作 PE-n）；Plane 缺陷编号 D31–D42 指设计调研 `plane-facts.md` §6 与 `plane-e2e.md` §1.5。前一段 [P3b](P3b-views.md) 交付了 navigation 模块与视图。

## 1. 目标与边界

设计 §12 的 P3（视图、工作区列表、收藏、最近访问）再拆：P3a 工作区列表、P3b 视图之后，**P3c = 收藏**，**P3d = 最近访问**（R1）。P3c：先拆 `bootstrap/app.go`（P3b 整分支审查 m4），planning 与 project 发布三种归档事件，navigation 加 `user_favorites` 与它的四个接口、目标的校验与读时过滤、目标删除与归档时的连带，守卫、组合测试、交错、e2e 接口故事。

不做：最近访问（P3d：`user_recent_visits`、`POST /me/recent-visits`、`GET /workspaces/{slug}/recent-visits`、`issue.deleted` 的连带、M4 交接 M7-04 的最近访问部分）；页面、store、旧 `favorite.service.ts` 与手写的 `IFavorite`（P7）；拖动次序的前端缺陷 D40（P7）。

**P3c 合入后 main 的样子**：收藏接口与连带可用；三种归档事件已发布（只有 navigation 订阅）；`bootstrap/app.go` 已拆；清理目录 36 张表；网页仍打旧地址（P7 接）；用户看不到变化。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/bootstrap/app.go` → `app.go`、`app_ports.go`、`app_modules.go`、`app_serve.go`，订阅者清单进 `events.go` | 保持行为的提取（R2）：新增的只有函数名、参数与一个承载局部变量的结构；构造的次序、每个构造的实参、失败时的清理路径不变，测试不改 |
| `server/internal/modules/planning/`、`project/` | 三种归档事件的声明与发布（§2.6）；navigation 要的批量端口（§2.5） |
| `server/migrations/sql/00053_navigation_user_favorites.sql` + schema 测试 | 表、CHECK、索引（§2.3） |
| `server/internal/modules/navigation/` | 收藏的领域、存储、用例、处理器；连带订阅者；`deleteView` 的连带；删除任务的收藏一步；清理一步 |
| `server/internal/modules/access/` | `favorite.read`、`favorite.write` 的规则 |
| `server/internal/bootstrap/` | 接线与端口转换（`navigation_ports.go`）；守卫（§4） |
| `api/modules/navigation.yaml`、`api/openapi.yaml` | 契约（§2.4）；根标签 `navigation` 的说明加收藏 |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql` | sqlc 条目、运行角色 |
| `web/` | 两个新码的文案（`error-messages.ts`、两份 `errors.json`，与契约同一提交）；`make gen` 的 `schema.gen.ts` |
| `e2e/` | 删除夹具经接口建收藏；`expectProjectDeleted` 的 `removedBefore`（R12）；H15、H16 的接口故事 |
| `docs/` | plane-diff、交接、硬规则、设计 §3.8/§4.2/§4.3/§12/§14、阶段记录 |

### 2.2 动作、角色与相对规则（设计 §3.8、§3.10）

| 动作 | 级别 | 角色 | 用例里的相对规则 |
|---|---|---|---|
| `favorite.read` | 工作区 | 管理员、成员 | 只列自己的；目标看不到的不列（§2.5） |
| `favorite.write` | 工作区 | 管理员、成员 | 只写自己的；目标须合格（§2.5） |

- 工作区访客：列、建、改、删都 403（Plane 相同：收藏接口只给工作区管理员、成员）。降为访客的人的行留着，升回即重现。
- 不是工作区成员（含已移除）：路由上的工作区答 404 `workspace.not_found`；`/favorites/{favorite_id}` 上答 404 `navigation.favorite_not_found`（§2.7）。
- 成员离开项目、被降为只看自己：行不删，读时过滤（设计 §3.8）；重新加入即重现。

### 2.3 表 `user_favorites`（设计 §4.2，DP-17、PE-1）

| 列 | 定义 |
|---|---|
| `id` | `uuid PRIMARY KEY` |
| `workspace_id` | `uuid NOT NULL → workspaces ON DELETE CASCADE` |
| `project_id` | `uuid NULL → projects ON DELETE CASCADE`（目标所属的项目；项目收藏为项目自己；工作区视图与文件夹为空） |
| `user_id` | `uuid NOT NULL → users ON DELETE CASCADE` |
| `entity_type` | `text NOT NULL`，CHECK 五值 `project`、`cycle`、`module`、`view`、`folder`（`view` 兼指两种视图，R4） |
| `entity_identifier` | `uuid NULL`（目标的 id，不是外键；文件夹为空） |
| `name` | `varchar(255) NULL`（只文件夹有：去首尾空白后 1–255 字，非空） |
| `parent_id` | `uuid NULL → user_favorites ON DELETE CASCADE`（所在的文件夹） |
| `sequence` | `double precision NOT NULL`（有限值） |
| `created_by_id`、`updated_by_id` | `uuid NULL → users ON DELETE SET NULL`（最后一次写的人：自己，或连带时触发的人） |
| `created_at`、`updated_at` | `timestamptz NOT NULL`（不设默认，漏给即报错，同 P3b） |
| `deleted_at` | `timestamptz NULL` |

- CHECK：`entity_type` 五值；文件夹 ⇔ `entity_identifier`、`project_id`、`parent_id` 都空且 `name` 非空；不是文件夹 ⇔ `entity_identifier` 非空且 `name` 为空；`project` ⇒ `project_id = entity_identifier`；`cycle`、`module` ⇒ `project_id` 非空；`sequence` 不是 NaN、正负无穷。
- 索引：部分唯一 `(user_id, entity_type, entity_identifier) WHERE deleted_at IS NULL`（文件夹的 `entity_identifier` 为空，空值在唯一索引里不相撞；谓词多写 `entity_identifier IS NOT NULL` 会让下面的 `ON CONFLICT … WHERE deleted_at IS NULL` 推断不到这条索引，42P10）；部分唯一 `(user_id, workspace_id, lower(name)) WHERE entity_type = 'folder' AND deleted_at IS NULL`（设计 §4.2 的两条加 `deleted_at IS NULL`，软删后能重建）；按目标 `(entity_type, entity_identifier) WHERE deleted_at IS NULL`（连带）；列表 `(user_id, workspace_id) WHERE deleted_at IS NULL`；外键的不带条件索引 `workspace_id`、`project_id`、`user_id`、`parent_id`；`deleted_at` 的清理索引。名字照已有命名，schema 测试按名字钉。
- 名字（schema 测试、`navigationNames` 与 23505 的识别都按它）：主键 `user_favorites_pkey`；外键 `user_favorites_{workspace_id,project_id,user_id,parent_id,created_by_id,updated_by_id}_fkey`；CHECK `user_favorites_entity_type_check`（五值）、`user_favorites_shape_check`（文件夹 ⇔ 列的搭配）、`user_favorites_project_id_check`（`project` 的 `project_id` 与 `cycle`、`module` 的非空）、`user_favorites_name_check`（非空）、`user_favorites_sequence_check`（有限）；部分唯一 `user_favorites_user_id_entity_type_entity_identifier_key ON (user_id, entity_type, entity_identifier) WHERE deleted_at IS NULL`、`user_favorites_user_id_workspace_id_name_key ON (user_id, workspace_id, lower(name)) WHERE entity_type = 'folder' AND deleted_at IS NULL`；索引 `user_favorites_entity_type_entity_identifier_idx`、`user_favorites_user_id_workspace_id_idx`（两条都 `WHERE deleted_at IS NULL`）、`user_favorites_{workspace_id,project_id,user_id,parent_id}_idx`、`user_favorites_deleted_at_idx`（`WHERE deleted_at IS NOT NULL`）。
- 插入条目写明冲突目标（`ON CONFLICT (user_id, entity_type, entity_identifier) WHERE deleted_at IS NULL DO NOTHING`，不吞文件夹名的冲突）；文件夹的插入不带 `ON CONFLICT`，按约束名把 23505 换成 409。
- 没有 Plane 的 `is_folder`（由 `entity_type` 定）。表名不以 `issue_` 开头，Go 的 `navigationTable` 与 e2e 的 `asynchronousDeletionTable` 各显式加一项。

### 2.4 接口（设计 §5）

| 操作 | 路由 | 回答 | `x-problem-codes` |
|---|---|---|---|
| `listFavorites` | `GET /workspaces/{slug}/favorites` | `{data: [Favorite]}` | `workspace.not_found`、`forbidden` |
| `createFavorite` | `POST /workspaces/{slug}/favorites` | 201 `Favorite`（新建）；200 `Favorite`（已有，原样） | `validation_failed`、`workspace.not_found`、`forbidden`、`navigation.favorite_folder_name_taken` |
| `updateFavorite` | `PATCH /favorites/{favorite_id}` | `Favorite` | `validation_failed`、`navigation.favorite_not_found`、`forbidden`、`navigation.favorite_folder_name_taken` |
| `deleteFavorite` | `DELETE /favorites/{favorite_id}` | 204 | `navigation.favorite_not_found`、`forbidden` |

- `Favorite`（DP-12 (a)，R8）：`{id, workspace_id, project_id|null, entity_type, entity_id|null, parent_id|null, sequence, name, logo_props, created_at}`。`name`：文件夹为它自己的名字，其余为目标此刻的名字；`logo_props`：项目、视图取目标的，迭代、模块（Nerve 没有图标）与文件夹为 `{}`；`LogoProps` 引 `../common.yaml`。不带 Plane 的 `entity_data`、`is_folder`，资源也不加 `is_favorite`（设计 §3.8）。
- `FavoriteCreate`：`entity_type`（必填）；`entity_id`（不是文件夹时必填，文件夹不许给）；`name`（文件夹必填，别的不许给）；`parent_id`（可选；文件夹不许给）。不收 `project_id`（服务端由目标推出；Plane 取请求体）。
- `FavoriteUpdate`：`parent_id`（uuid 或 null）、`sequence`（有限数）、`name`（只文件夹）。给出的字段整体替换；在规范化之后与锁下重读的行比，什么都没变（含 `{}`）→ 200 原样、不写、不动 `updated_at`。
- 列表集合型、不分页：本人在这个工作区的全部收藏（含文件夹里的），按 `sequence DESC, id DESC`（Plane 网页按 `sequence` 降序显示）；目标看不到的不列（§2.5）。
- 新码：`navigation.favorite_not_found`（404）、`navigation.favorite_folder_name_taken`（409）。目标或父不合格是 `validation_failed` 加已有的字段码 `not_allowed`（字段 `entity_id` 或 `parent_id`）。
- 收藏的写不进项目的写闸门，不声明 503。
- `createFavorite` 是契约里第一个声明两个 2xx（201、200）的操作；处理器测试两种都答，生成的回答类型两种都有。

### 2.5 目标的合格、可见与次序（设计 §3.8，DP-7、DP-8、DP-10、DP-14、PE-6、PE-7、PE-14）

**可见**（列表、改、删用同一谓词）：

| 目标 | 可见 |
|---|---|
| `project` | 在调用者的 `Readable` 里（项目的有效成员，含只看自己；不是成员的工作区管理员不算），未归档、未删除 |
| `cycle`、`module` | 未删除、未归档；所属项目可见 |
| `view`（项目视图） | 未删除；所属项目可见；不被 `View.HiddenFrom` 藏起（只看自己的人看不到别人的视图） |
| `view`（工作区视图） | 未删除（`favorite.*` 的角色已含 `workspace_view.read`） |
| `folder` | 总可见；里面的每项各按自己的谓词 |

- **合格**（建）= 可见，且目标在路由的工作区里。不存在、在别的工作区、看不到、已归档一律同一个 422（`entity_id` 的 `not_allowed`），不泄露存在（R7）。目标与父同时不合格时，一个 422 带两个字段的 `not_allowed`（硬规则 §4：全部问题在一个 422 里）。
- **开关不看**（`cycle_view`、`module_view`、`issue_views_view`，DP-10 (a)）：收藏是个人的导航，开关只管页面入口；Plane 也不看。
- **父**（`parent_id`）：本人、同一工作区、未删除的文件夹；文件夹不能有父（一层，D42）。不合格同为 422（`parent_id` 的 `not_allowed`）。
- **再收藏**：同一目标已有未删的收藏 → 200 原样回答（不改它的父与次序），不写（Plane 也原样返回；Plane 新建与已有都答 200，Nerve 新建 201）。“已有”以锁下 `FOR SHARE` 读到的为准（§2.8），从读到它到提交之前它删不掉。
- **次序**（D33、PE-7）：新建的 `sequence` = 本人在这个工作区未删收藏的最大值 + 10000，没有则 65535；同一人并发建可能同值，列表按 `id` 打破，不加锁。PATCH 收任意有限数，不重排、不查重。
- **文件夹名**（D41、PE-14）：去首尾空白后 1–255 字、无 NUL；同一人在同一工作区未删的文件夹里 `lower(name)` 唯一，建与改名撞上都答 409 `navigation.favorite_folder_name_taken`（由部分唯一索引兜底）；改成自己的名字（含只差大小写）不算冲突。
- **删除**：软删（R10；Plane 的 DELETE 是硬删）；删文件夹在同一时刻软删它里面的全部。

### 2.6 归档事件与连带（设计 §3.8、§4.3、§6.1，R3、R11）

- **事件**：`planning.cycle_archived`、`planning.module_archived`（planning 声明）、`project.archived`（project 声明），在归档的事务里、写下 `archived_at` 之后发布；载荷 `Entity{cycle|module|project, id}`、`WorkspaceID`、`ProjectID`、`ActorID`、`OccurredAt`，没有 `Changes`。`OccurredAt` 与写进 `archived_at` 的是同一个值：项目的归档用例今天读两次时钟（`project/app/archive_project.go:50,56`），改为锁后读一次、写与事件共用，工作区 id 取 `lockAndDecide` 的返回值（今天被丢掉）；迭代、模块的用例已在锁下读一次。`project.archived` 这个串同时是一个 409 问题码（已归档项目上的写）：事件名照设计不改，用命名常量并加注释，两处不混用。迭代、模块已归档时再归档不写，就不发；项目再归档写新时刻（照 Plane），照发（连带幂等）。恢复不发（设计 §6.1、§15 D1）。
- **连带订阅者**（navigation，一个订阅者五种事件，DP-15 (a)）：`planning.cycle_deleted`、`planning.module_deleted`、`planning.cycle_archived`、`planning.module_archived` → 软删所有人对这个目标的收藏；`project.archived` → 软删 `project_id` 为它的全部收藏（项目自己、它的迭代、模块、视图）。在发布者的事务里同步执行；先整体按类型校验载荷（删除事件带恰一条 `name` 的变化，归档事件 `Changes` 为空，`project.archived` 的实体就是 `ProjectID`；畸形的拒绝，事务回滚）；订阅者另起名字，与已有的 `navigation_delete_rows_on_deletion` 并列；按 `id` 升序加锁；`deleted_at`、`updated_at` 取事件的 `OccurredAt`，`updated_by_id` 取 `ActorID`。迭代、模块的删除与归档对所有人生效（Plane 的应用接口删除时只删当前用户的，D31）。
- **视图删除**：`deleteView` 在同一个事务、与视图同一时刻、同一写者软删所有人对它的收藏（P3b spec §5）。
- **恢复不加回**：归档时已软删，恢复后无物可回；读时过滤保证归档期间不显示。
- **项目、工作区删除**（R11，DP-16 (a)）：`navigation.delete_rows` 的种类与载荷 `{scope, id, at, by}` 不变，任务改为按表的步骤（视图 → 收藏），各自游标、共用预算；项目范围删 `project_id = P` 的收藏，工作区范围只删 `workspace_id = W AND project_id IS NULL` 的（文件夹、工作区视图的收藏）；项目里的由每个列出的项目的任务删。立即不可见靠读时过滤。删除窗口里（项目已删、任务还没跑）用户删一个文件夹，会以用户的时刻与写者软删它里面属于这个项目的子行，任务随后跳过它们——允许；Go 与 e2e 的“随项目删除的行带项目的时刻”断言只看夹具造的行，不覆盖这种交错。
- **清理**：模块的清理加一步 `user_favorites`（在 `issue_views` 之后），清理目录 35 → 36；物理删文件夹时外键连带物理删它的子行（子行与文件夹同一时刻软删过）。

### 2.7 拒绝次序（DP-28；P3b §2.5 的写法）

- **`/favorites/{favorite_id}` 上的每个 404 都是 `navigation.favorite_not_found`**，与不存在的 id 逐值相同（码、体、头）：不存在、已软删、别人的、它的工作区他已看不到、它的目标他已看不到（DP-9 (a)）。工作区访客答 403。
- 列：404 工作区 → 403 角色 → 读（一个只读快照）。
- 建：请求体 400/422（结构；`entity_type` 与各字段的搭配；文件夹名——纯函数，先于任何读）→ 404 工作区 → 403 角色 → 预读目标（不加锁，只为知道要锁哪个项目、哪个视图）→ 锁 → 锁下重读、重判 → 422（目标与父，一个回答）→ 已有 → 200 原样 → 插入 → 409 文件夹名。
- 改：读本人的行（不加锁）→ 404 → 403 角色 → 目标看不到 → 404 → 422（按行的种类：`name` 只给文件夹、文件夹不能有父、`sequence` 有限）→ 锁 → 重读、重判 → 422 父 → 409 文件夹名 → 无改动 200 → 写。
- 删：读本人的行 → 404 → 403 角色 → 目标看不到 → 404 → 锁 → 重读、重判 → 写。

### 2.8 锁序（DP-5 (a)、DP-6 (a)、DP-30 (a)）

- 新的 `favoriteLocks`（navigation 的 app 里，与视图的 `Locks` 共用工作区、项目的端口；视图的 `Locks.run` 不动，它写死了视图的开关）。M3 约定：工作区 `FOR SHARE` → 项目 `FOR SHARE`（目标或行有项目时）→ 视图行 `FOR SHARE`（目标是视图时，新端口 `ShareView`）→ 收藏行；锁下重读、重判；一次时钟读数在最后一把锁之后。
- 迭代、模块不锁本行：它们的一切写先取项目行 `FOR NO KEY UPDATE`（planning 的锁路），项目 S 已与它们的归档、删除串行。
- **收藏行的全序**（re-91 I2）：文件夹行先于条目行，同类按 `id` 升序。建：父文件夹行 `FOR SHARE`、锁下重判它未删；建目标（不是文件夹）的再锁本人对这个目标的未删收藏 `FOR SHARE`（条目行，写之前的最后一把锁）——找到即 200 原样、不插，它的删除等到建提交；找不到（含等到一个扣着它的删除提交之后，READ COMMITTED 的重查排除已删的版本）才插入；插入撞上已有（同一人同时两次建，后一次的读在前一次插入之前）照旧不插、读回那一条答 200，读回落空（500，可重试）只剩同一人同时建、建、删一种情形。改：先锁新的父文件夹 `FOR SHARE`，再锁本行 `FOR UPDATE`。删文件夹：先锁文件夹行 `FOR UPDATE`，再按 `id` 升序锁它的子行。连带与项目范围的删除任务只锁条目行（按 `id` 升序；与用户的写被项目的锁隔开）；工作区范围的那一步在工作区删除之后才跑（用户的写已被工作区的锁隔开），也照全序锁文件夹与工作区视图的收藏。
- 收藏是工作区级的写：矩阵行只用工作区的列，不进项目写锁表（`writesOnAProject`）；另写一条锁序测试 `TestFavoriteWritesTakeTheirTargetsLocks`：工作区被 `FOR NO KEY UPDATE` 扣住时建、改、删在等，等的时候不持有项目与收藏行；项目被扣住时持工作区 S、不持视图与收藏行；视图行被扣住时持项目 S；本人对目标的收藏被 `FOR UPDATE` 扣住时，再建它的等在这把 `FOR SHARE` 上，持工作区、项目、视图（目标是视图时）与父（给了父时）的 S，还没写任何行。

## 3. 局部裁定

- **R1** 再拆段：P3c 收藏（约 16 个 Task），P3d 最近访问（约 10 个）；表与它的接口同段（e2e 的删除夹具只经接口造数）。关键词的 phase 仍是 `M7/P3`。
- **R2** 先拆 `bootstrap/app.go`（397 行，`newAppWithFactories` 265 行，约 25 个局部变量、9 处失败清理）：四个文件 + 订阅者清单进 `events.go`，**保持行为的提取**，单独提交，测试不改。要守的次序：① 全部 `Provide` 先于各模块的 `New`；② project 先于 workspace（`proj.Cascade()`）；③ workspace 先于 identity（`ws.SignupInvitations()`、`ws.Deactivator()`）；④ asset 在 issue、project、workspace 之后（`AssetAccess()`）；⑤ 清理模块、任务与种类的拼接、路由注册的次序；⑥ 每个失败点的清理（`a.close()`）。测试引用的未导出名（`newApp`、`newAppWithPoolFactory`、`newAppWithFactories`、`appFactories`、`poolCloseTimeout`、`publicOperations`、`a.jobs`）不动。
- **R3** 归档事件写了 `archived_at` 才发；恢复不发；载荷没有 `Changes`（§2.6）。
- **R4** 五种类型，`view` 兼指两种视图；工作区视图可收藏（设计 §3.8；Plane 网页收藏不了）；没有 `issue`、`page`（M1-P3 交接）。
- **R5** 收藏不看开关；已归档的目标收藏不了，归档时连带删除。
- **R6** 收藏是工作区级的写：不进项目写锁表，另写锁序测试（§2.8）。
- **R7** 目标不合格一个 422；`/favorites/{favorite_id}` 上的 404 一个码，含目标已看不到的行。
- **R8** 接口的字段名 `entity_id`（与 P3d 的最近访问一致），列名照设计 `entity_identifier`；回答平铺，名字与图标取目标此刻的。
- **R9** 次序与文件夹名的规则（§2.5）。
- **R10** 全部软删（用户删除与各连带）；两条部分唯一加 `deleted_at IS NULL`。
- **R11** 删除任务同种类、同载荷，按表的步骤；工作区那一步只删 `project_id IS NULL` 的收藏。
- **R12** 先归档再删除的两条守卫：Go 的 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt` 在归档之后用 SQL 种收藏；e2e 的 `expectProjectDeleted` 加 `removedBefore: string[]`（与 `deletedAlone` 对称），P4 的两条传 `["user_favorites"]`，断言有更早删的行、没有未删的——兼作 H16 的数据库证据。与连带同一个 Task。
- **R13** 事件目录测试：`TestIssueEventDirectory` 的 project 目录加 `project.archived`；`TestPlanningEventDirectory` 改为 issue 与 navigation 两组订阅者，畸形的删除与归档事件各被拒绝（归档类型没有 issue 的订阅者）。
- **R14** 收藏的删除窗口：H15 组合测试的一个子测试，`holdNavigationDeletion` 扣住删除任务，项目删除、工作区删除之后行还活着时：列表不给，改、删答 404 与不存在的逐值相同，建以它为目标答 422。

## 4. 验收标准

- `bootstrap/app.go` 的拆分（R2）：`go build`、`go vet`、`lint-go`；bootstrap 测试不改，两路全量在 T3a 之前过一次（本机与分支 CI，检查点 0；此时 e2e 也应全绿）；台账记每段的去处与 R2 的六条次序各在哪里。
- 契约、生成物、处理器逐码（§2.4）；`make gen-check` 无差异；两个新码有文案。
- 迁移的 schema 测试（列、可空、默认、索引与约束名、各 CHECK 的反例、两条部分唯一在软删之后能重建、文件夹名不分大小写撞上、外键的连带）。
- 单测：领域（类型与字段的搭配、文件夹名、次序、父规则、无改动）、用例（拒绝次序、锁下重判、可见与合格、再收藏、删文件夹）；planning、project 的归档用例发布事件（再归档的两种情形、恢复不发）。交叉格：建——坏体 × 看不到的工作区 → 422；改——坏体 × 别人的行 → 404；目标已看不到 × 坏体 → 404。
- 守卫：权限矩阵（四个操作，只用工作区列——用了项目级的列，`writesOnAProject` 就按操作 id 把它整体算作项目上的写；建的变体：文件夹、工作区视图（201 / 201 / 403）与一个带 `prepare` 的项目目标（照通知行的 `prepareDraftPublisher`，加入公开项目后 201）；改、删用每列自己的文件夹，种子用 SQL）、锁序测试（§2.8）、动作 ↔ 规则、事件目录（R13）、删除（Go 守卫与 R12、清理目录与名单、`navigationTable`、e2e 目录）、删除窗口（R14）。
- 组合测试（真实的 app）：
  - H15 `TestFavoritesOfEveryKindThroughTheApp`：五种各收藏，`project_id`、名字、图标由服务端推出；再收藏 200 原样；不合格的四种（不存在、别的工作区、看不到——私有项目非成员、只看自己的人收藏别人的视图——已归档）同一个 422；工作区访客 403；文件夹的名字、重名 409（建与改名、大小写）、一层、父的四种不合格；进出文件夹与改次序；无改动；列表次序（两个同 `sequence` 的收藏）；读时过滤（离开项目后不列、行在，重新加入又列；降为只看自己后别人的视图收藏不列）；项目级的格（H19，矩阵装不下）：只看自己的访客（项目、迭代、模块、自己的视图可收藏，别人的视图 422）、工作区管理员兼项目访客（`guest_view_all_features` 关着时同只看自己的访客，开着时别人的视图也可收藏）、私有项目的非成员（422）、撤权后读时过滤；开关关着（`cycle_view`、`module_view`、`issue_views_view`）照常收藏、照常列出（R5）；目标已归档而行还在（用 SQL 留下，设计 §3.8 说的写入与归档交错留下的行）时不列；目标与父同时不合格的一个 422；别人的、不存在的、目标已看不到的 `/favorites/{id}` 同一 404；删文件夹连子项；R14 的子测试。
  - H16 `TestFavoritesFollowTheirTargets`：两个人收藏同一目标；迭代删除、模块删除、迭代归档、模块归档、项目视图删除、工作区视图删除 → 所有人的收藏软删（时刻、写者 = 触发者），别的目标的收藏不动；恢复不加回；项目再归档照发（`OccurredAt` 等于新写下的 `archived_at`），迭代再归档不发；`deleteView` 的连带与视图同一时刻、同一写者；项目归档移除它里面的四种、文件夹与别的项目的留下；项目删除、工作区删除经任务删除（时刻、删除者取事件的），工作区那一步不删项目里的；归档事件畸形时归档回滚。
- 交错 `TestFavoriteWritesInterleaveWithArchives`：建、改、删 × 项目归档 / 迭代归档 / 迭代删除 / 视图删除 / 项目删除 / 成员移除（锁下重判）；删文件夹 × 移入它、× 在它里面新建（不留活的子行在已删的文件夹下：清理物理删文件夹时外键会连带物理删掉活行）、× 改它里面的子行并带同一个 `parent_id`（全序，不死锁）；同一人删收藏、删装着它的文件夹 × 再建同一目标（删先 201、只剩新的一条活行；建先 200 原样、删照删，不留活行；不答 500）。配对共用准备好的库、分组跑，本机时间记进台账。
- e2e：删除夹具经接口建收藏（`furnishIssueRows`：项目、迭代、模块、项目视图；`furnishWorkspace` 与 W3 自己的 `furnish`：文件夹与里面的工作区视图收藏）；P4（两条）、W2、W3（两个文件）、I21 转绿；H15、H16 的接口故事 `e2e/stories/collaboration/h15-favorites.spec.ts`（PAT 与会话对等）、`h16-favorite-cascades.spec.ts`，数据库断言。
- 每个性质至少一个能让测试失败的变异（plan 的变异表）。
- 测试时间：四个新顶层名都在第 0 路（桶 6、13、2、37），预估本机 H15 约 5–8 秒、H16 约 4–6 秒、交错约 15–25 秒、锁序约 2–3 秒；第 0 路另跑 navigation、planning、project、access、migrations 各包的新测试。第 1 路只因已有守卫变重（`TestPermissionMatrix` 桶 46 等，估几秒）。按作业的 Test 步实测，每路增量 ≤ 90 秒；第 1 路的 bootstrap 包到 `go test` 期限的八成先报告。

## 5. 后续责任与交接

- P3d：最近访问；删除任务加最近访问一步；清理 36 → 37。
- P7：侧栏收藏、星、命令面板、删旧 `favorite.service.ts` 与 `IFavorite`、`old-favorite-urls` 关键词规则、D40 的拖动次序。
- 本段写处理结果的交接：M6-M7-02（迭代、模块的收藏与归档连带：后端完成，页面随 P7；它留的一问“模块归档时 Plane 是否同样移除”答是，`module/archive.py:553-558` 按目标删所有人的）、M3 §1.4（项目收藏：接口完成；“归档项目时的收藏处理”= 后端在归档的事务里移除；星与菜单随 P7）、M1-P3-trim-platform（收藏五种：契约定下，前端用生成类型随 P7）、M1-P2-trim-content（收藏的接口没有 `page` 类型：P3c 落定；最近访问一半随 P3d）、M2 §1（收藏的列表：集合型、经端口拿目标；最近访问仍 open → P3d）、M2 §5（`user_favorites` 的外键去向：`user_id CASCADE`、审计列 `SET NULL`、自引用 `CASCADE`）。

## 6. 风险

- 删除图与守卫面广，另有两条“先归档再删除”的守卫（R12）：红窗口写进 plan，各带测试名。
- 三种新事件在发布者的事务里同步执行订阅者：连带失败会让归档失败（与删除事件同）；订阅者严格校验、按 `id` 升序加锁。
- 测试时间：第 1 路余量（P3b 记录 §9：main 69%，慢 runner 73%）；新顶层名全在第 0 路；超出预算照 P2b 的做法报告。
