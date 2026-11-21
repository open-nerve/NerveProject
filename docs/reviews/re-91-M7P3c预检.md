# M7 / P3c 预检

> 2026-11-21：结论 **修正后通过**（Ready after fixes）。没有严重问题；6 处重要、14 处次要。合同的每一项在 `1168fd25` 上都做得出来：三个归档用例的模块 `Deps` 早有 `Events`，只差传进用例；项目 S 确实与迭代、模块的归档和删除串行；视图 `FOR SHARE` 与 `deleteView` 的 `FOR UPDATE` 冲突；`00053` 空着；清理、删除图、事件目录要改的计数与名单与 plan 列的一致；四个新顶层名的桶复算无误；Plane 的事实抽查全部成立。重要问题落在：`app.go` 的拆分做不成“纯搬移”（265 行的函数只能提取），验收写法要换；改与删文件夹的锁序相反，同一人的两个请求能成环；允许清单少两段（矩阵的种子文件、navigation 模块自己的测试），且与“不再加长”的约束相撞；T3、T10 各装了两件可单独评审的事；矩阵的目标变体照现有种子全是 422，H19 的项目级格没有落点；变异表缺六条（最要紧的是“活的子行留在已删的文件夹下”）。不需要改设计的取舍，没有要负责人新定的事。
> - **目标**：[P3c spec](../v0/M7-collaboration/specs/P3c-favorites.md)、[P3c plan](../v0/M7-collaboration/plans/P3c-favorites.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2（H15、H16）、§3.8、§3.10、§4.1–§4.3、§5、§6.1、§6.3、§9、§10、§12、§15，调研 `research/backend.md`（DP-n）、`research/plane-e2e.md`（PE-n），先例 P3b 的 [spec](../v0/M7-collaboration/specs/P3b-views.md)、[plan](../v0/M7-collaboration/plans/P3b-views.md)、[阶段记录](../v0/M7-collaboration/reviews/P3b-views-review.md)、预检 [re-90](re-90-M7P3b预检.md)，[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §3–§5。
> - **基线**：main `1168fd25`（P3b 的 noff），worktree `codex/m7-p3c`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git 状态（只用了 `git status`、`git diff` 读草稿）、不起容器、不跑测试。跑过的只有 `go test -list . ./internal/bootstrap`（编译、列名字：777 个顶层测试，没有带 `Favorite` 的名字）与 `printf %s 名字 | cksum`（查桶）。对照代码核了：`bootstrap/app.go` 整份与 `events.go`；三个归档用例、两条锁路（planning、project）与它们的 SQL；navigation 的 `locks.go`、`delete_view.go`、`delete_rows.go` 与 SQL、模块根；access 的 `Readable`；事件类型的全部声明、订阅者与使用者；每个自己拼 dispatcher、自己建 planning / project 模块的测试；项目写锁表的分类函数；矩阵的覆盖、目标规则与种子结构；`contract.Operations()` 的全部使用者；清理的四处名单与目录守卫；迁移的三处计数；Go 的删除守卫与 `holdNavigationDeletion`；e2e 的 `catalog.ts`、`furnish.ts`、两个删除断言与每个调用它们的故事；分路脚本；Plane 的 `favorite.py`、模型与七处连带。下文路径相对 worktree 根，`si/` = `server/internal/`，`bs/` = `si/bootstrap/`，`nav/`、`pl/`、`pj/` = `si/modules/{navigation,planning,project}/`，`mig/` = `server/migrations/`，`pa/` = `plane/apps/api/plane/`。

没有核实的：

- 任何测试的实际耗时与两路、两片的余量（没有运行）；69%、73% 取自 P3b 阶段记录 §9。
- `make gen` 之后生成物的差异；一个操作两个 2xx 状态在 oapi-codegen 与 `openapi-typescript` 里的样子（契约里没有先例，见 m6）。
- 带部分唯一索引的 `ON CONFLICT` 的写法（m2 只按 PostgreSQL 的规则推理）。
- 锁序测试扣三把锁时测试应用的 4 个连接够不够。
- 调研 `plane-e2e.md` 里前端一侧的事实（只核了后端与 `favorite.store.ts` 的重排一段）。

## 2. 发现

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §2.1 第一行、R2、§4 第一条；plan T2 | **`app.go` 的拆分做不成“纯搬移”，验收写法验不出东西。** 397 行里 265 行是一个函数 `newAppWithFactories`：密钥、池、`Provide` 步、事件、各模块的 `New`、清理与任务、路由注册，全靠约 25 个局部变量串着（`pool`、`tx`、`readTx`、`issueTx`、八个 `…Ports`、`authorizer`、`decider`、`readable`、`inserter`、`events`、`projectWrites`、`previews`、`imageBindings`、`imageReader`、`assetPurger`、`limiter`、`businessClock`、各模块的值），中间有 9 处 `a.close()` 的失败清理。把它分到 `app_ports.go`、`app_modules.go`、`app_serve.go` 只能**提取函数**：新的函数名、参数表、一个承载局部变量的结构——“名字不变”“`git diff --stat` 只有搬移”都不成立，也验不出行为没变。真正要守的是次序：`Provide` 全部先于 `New`；project 先于 workspace（`proj.Cascade()`）；workspace 先于 identity（`ws.SignupInvitations()`、`ws.Deactivator()`）；asset 在 issue、project、workspace 之后（`AssetAccess()`）；清理模块的次序；任务与种类的拼接次序；路由注册的次序；每个失败点的清理。T2 的“一组代表性的 bootstrap 测试”也不够：这是 777 个测试共用的组合根。没有测试依赖 `app.go` 的行位置或文件名（只有三处注释提到它）；测试引用的未导出名 `newApp`、`newAppWithPoolFactory`、`newAppWithFactories`、`appFactories`、`poolCloseTimeout`、`publicOperations`、`a.jobs` 不动即可。订阅者清单进 `events.go` 是合理的（`eventTypes()` 已在那里），它也是一个新函数。 | `bs/app.go:83-348`（函数）、`:132-162`（Provide 步）、`:169-170`（订阅者）、`:228`（“project before workspace”）、`:250-277`、`:278-287`、`:288-295`、`:335-343`；`bs/events.go:158-161`；`bs/project_access_test.go:21` 等三处注释 | R2 与 §2.1 改写为“保持行为的提取”：新增的只有函数名、参数与一个承载局部变量的结构；构造的次序、每个构造的实参、失败时的清理路径不变，测试不改。spec 列出上面六条次序。§4 的验收改为：`go build`、`go vet`、`lint-go`；**两路 bootstrap 全量**（本机或分支 CI 的 server (0)、(1)，e2e 此时还绿）在 T3 之前过一次；台账记每段的去处。订阅者的函数签名写进 T2（T10 再给它加 `navigationPorts`）。 |
| I2 | spec §2.8 第三条；plan T5、T8、Review Focus | **改与删文件夹的锁序相反，同一人的两个请求能成环。** 改：“本行与新的父文件夹一条语句按 `id` 升序 `FOR UPDATE`”；删文件夹：“先锁文件夹行，再按 `id` 升序锁它的子行”。子行 c 已在文件夹 F 里、`c.id < F.id`（id 是时间序的 v7，先收藏、后建文件夹再拖进去就是这样）时：`PATCH c {parent_id: F, sequence: …}` 先锁 c 再等 F；`DELETE F` 先锁 F 再等 c。PostgreSQL 判死锁、回滚其一，答 500。给出的字段“整体替换”，所以带着没变的 `parent_id` 一起发是合法的请求。其余组合核过不成环：连带与两种范围的删除任务只按 `id` 锁条目行或与用户的写被工作区、项目的锁隔开；建只对父文件夹取 `FOR SHARE`；移入别的文件夹时 `DELETE F` 的子行语句在等待之后按新版本重判 `parent_id`（READ COMMITTED）。spec §4 的交错“删文件夹 × 移入它”是另一种情形（c 还不是 F 的子行），抓不到这个。 | spec §2.8；`si/shared` 的 id（`uuid.NewV7()`，`pl/app/deletion_event.go:16`）；`nav/adapter/postgres/queries/delete_rows.sql:6-12`（现有的按 id 加锁写法）；硬规则 03 §4“M3 的约定”约定五 | §2.8 给 `user_favorites` 定一个全序：**文件夹行先于条目行，同类按 `id` 升序**。改：先锁新的父文件夹（`FOR SHARE` 即可，与建相同），再锁本行 `FOR UPDATE`；删文件夹照旧；连带与任务只锁条目行或被外层锁隔开，写一句理由。交错加一格“删文件夹 × 改它里面的子行（带同一个 `parent_id`）”，变异表加“改按 id 升序锁本行与父”。 |
| I3 | plan Global Constraints（允许清单；“不再加长”的文件） | **允许清单少两段，其中一段与文件大小的约束相撞。** ① 矩阵：`/favorites/{favorite_id}` 落在目标规则的通用分支（参数以 `_id}` 结尾 → 必须是 `seeded.workspaceOfRow` 认得的、种在这一列工作区下的行）。`workspaceOfRow` 只认 `memberships`、`invitations`、`drafts`、`assets`、`notifications` 五张表，所以要改 `permission_matrix_seeded_test.go`：`seeded` 加一个字段、`newSeeded` 的初始化、`workspaceOfRow` 的名单；`prepareMatrix`（`permission_matrix_seed_test.go`，390 行）加一次调用。plan 把 `permission_matrix_seeded_test.go`（428 行）列为“不再加长”，允许清单里又只有“新行、种子、`matrixRows()` 的拼接一项”。② navigation 模块自己的已有测试没有一条在清单里，而删除任务改成按表的步骤、清理加表、`deleteView` 加连带都会碰到：`nav/app/delete_rows_test.go`（端口的假实现与步骤）、`nav/app/delete_view_test.go`（调用序列多一次连带）、`nav/adapter/postgres/purge_test.go`（`Tables()` 恰为 `issue_views`）、`nav/adapter/postgres/delete_rows_test.go`（按范围选语句的表）。③ `nav/app/delete_rows_on_deletion_test.go:58` 钉了订阅者的名字与两种类型：连带订阅者要另起名字（spec 已写“一个订阅者五种事件”，与它并列），这条不改。 | `bs/permission_matrix_targets_test.go:163-168`；`bs/permission_matrix_seeded_test.go:139-146,173-176,402-410`；`bs/permission_matrix_notifications_test.go:40-44`（先例）；`nav/app/delete_rows_test.go:42-46`；`nav/adapter/postgres/purge_test.go:109-110`；`nav/adapter/postgres/delete_rows_test.go:42-43,105-106`；`nav/app/delete_rows_on_deletion_test.go:58` | 允许清单加：`permission_matrix_seeded_test.go`（三处，写明各加什么；字段与初始化并进已有的行，净增不超过 2 行）、`permission_matrix_seed_test.go`（一次调用，种子函数进新文件）、navigation 模块的四个测试文件（各写改法）。“不再加长”一句改成“除这两处之外不再加长”。 |
| I4 | plan（“16 个 Task”）、T3、T10、检查点 | **T3、T10 各装了两件可单独评审的事；T3 到 T12 之间没有任何 CI 证据。** T3 = 迁移与 schema 测试 + sqlc 与运行角色 + 两个动作与规则 + 清理一步与四处名单 + **删除任务改成按表的步骤**（app、SQL、adapter、三个测试）+ 两处目录谓词 + Go 守卫的种子：P3b 用了两个 Task（T4、T7），re-90 的 I10 正是这一条。T10 = 两个模块三种事件的声明与发布、三组单测 + 连带订阅者与签名 + `deleteView` 的连带 + R12（Go 与 e2e）+ R13。硬规则要“评审敏感的内容不放在同一段”，事件的发布时机与连带的锁序是这一段最敏感的两件。另外窗口 (b) 从 T3 开到 T12，分支 CI 的 e2e 九个 Task 都红；Go 两路不在窗口里，却也要等到 T12 才第一次看。 | plan T3、T10；[P3b plan](../v0/M7-collaboration/plans/P3b-views.md) T4、T7；re-90 I10；设计 `M7-design.md:289`；硬规则 03 §4“设计第 12 节的约束”2、3 | T3 分 a（迁移、schema、sqlc、运行角色、动作、清理——清理守卫不红）与 b（删除任务的步骤、两处谓词、Go 种子——删除守卫的红窗口只跨 a → b，写进窗口清单并带测试名：`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 及用 `parentAsyncTable` 的几条）；T10 分 a（三种事件：声明、发布、单测、`TestIssueEventDirectory`）与 b（订阅者、`deleteView`、R12、`TestPlanningEventDirectory`）。数目到 18，回到 16 的办法由主控选：把 T2 做成单独的插入段（调研 DP-2 (b)，纯提取、自己合入），或把 T6 并进 T7（re-90 的“端口与读、建”）。T3b 之后加一次“Go 两路”的中间检查（e2e 预期红）。 |
| I5 | spec §4（守卫、H15）；plan T11、变异表“矩阵” | **矩阵的目标变体照现有种子全是 422，H19 的项目级格没有落点。** ① 矩阵的行只能用工作区的列（用了项目级的列，这个操作就整体算“项目上的写”，这是 `writesOnAProject` 按**操作 id** 的分类）。而种子里 acme 的管理员、成员一个项目都读不到（管理员不是任何项目的成员，成员在私有项目里的成员关系已删），只有访客读公开项目——所以“建的变体：项目、迭代、项目视图”三行在管理员、成员两格都是 422，访客 403，什么也分不出；能答 201 的只有文件夹与工作区视图。通知的行靠 `prepare` 把他们加进公开项目（`prepareDraftPublisher`）。② 因此 H19 点名的项目级的格——只看自己的访客、**工作区管理员兼项目访客**、私有项目、撤权——矩阵装不下，只能在 H15 的组合测试里。spec §4 的 H15 写了私有项目非成员、只看自己的人与别人的视图、离开项目，没有写管理员兼访客（硬规则：“访客不能 X”对他往往不成立，要写清）：他在项目里是访客，`OwnOnly` 为真时能收藏项目、迭代、模块与自己的视图，收藏不了别人的视图；开了 `guest_view_all_features` 之后能。 | `bs/project_write_locks_test.go:150-167`（按 `r.op` 记 `projectLevel`）；`bs/permission_matrix_notifications_test.go:20-33`（种子里谁读得到什么、`prepare` 的做法）；`si/modules/access/app/readable.go:42-49`；硬规则 03 §4“访客规则”管理员兼访客一条；设计 `M7-design.md:155`、H19 | spec §4 写明：矩阵四个操作、工作区列，建的变体为文件夹、工作区视图（201 / 201 / 403）与一个带 `prepare` 的项目目标（加入公开项目后 201，不加 422）；改、删用每列自己的文件夹。H19 的项目级格写进 H15 的清单：只看自己的访客（项目、迭代、模块、自己的视图可收藏，别人的视图 422）、管理员兼访客的两种开关、私有项目非成员、撤权后读时过滤。变异表“矩阵”一行据此改。 |
| I6 | plan“性质与变异”；spec §4 | **变异表缺六条。** ① **活的子行不留在已删的文件夹下**：建不对父文件夹取 `FOR SHARE`、改不锁新的父——与删文件夹交错后留下活的子行；清理物理删文件夹时外键 `CASCADE` 会把这条**活的**行一起物理删掉（spec §2.6 “子行与文件夹同一时刻软删过”正依赖它）。交错只有“删文件夹 × 移入它”，要加“删文件夹 × 在它里面新建”。② **读时过滤已归档**：归档的连带会把行删掉，所以“列表不滤已归档的目标”只有在行还在时才杀得死——种子要有一条用 SQL 留下的、目标已归档的活行（设计 §3.8 说的“写入与归档交错留下的行”）；表里的“不查归档”只挂在建上。③ **开关不看**（R5）：变异“关着 `cycle_view`、`module_view`、`issue_views_view` 时拒收藏或不列”，H15 加一格。④ **项目再归档照发**、`OccurredAt` 等于写下的 `archived_at`（表里只有“迭代再归档也发”）。⑤ **`deleteView` 的连带与视图同一时刻、同一写者**（表里的“时刻、写者不取事件的”只管事件）。⑥ **一个 422 装全部问题**（见 m1）：`entity_id` 与 `parent_id` 同时不合格时两个字段都在。 | plan 的表；spec §2.5、§2.6、R5；`pj/app/archive_project.go:33-37,56`；`nav/app/delete_view.go:31-33`；硬规则 03 §4 第 7 步、§5“测试数据分不出性质” | 表补六行，各标 Task；spec §4 的交错与 H15、H16 的清单跟上。 |

### 次要

| # | 位置 | 发现与证据 | 修正 |
|---|---|---|---|
| m1 | spec §2.7（建：“422 目标 → 422 父”） | 写成了先后两步。硬规则要“全部问题在一个 422 里”（03 §4 第 7 步）：目标与父同时不合格时，一个 422 带 `entity_id`、`parent_id` 两个 `not_allowed`。分两步也多一个次序上的差别可看。 | 改成“锁下重判 → 一个 422（目标、父）”；改的“422 父”同理与别的字段错误合并。 |
| m2 | spec §2.5（再收藏、文件夹名）；plan T5（“`ON CONFLICT DO NOTHING` 后读出已有”） | 两条部分唯一要分开处理：不带目标的 `ON CONFLICT DO NOTHING` 会把文件夹名的冲突也吞掉（之后按目标读不到“已有”）。条目的插入要写冲突目标 `(user_id, entity_type, entity_identifier) WHERE deleted_at IS NULL AND entity_identifier IS NOT NULL`；文件夹的插入、改名不带 `ON CONFLICT`，把这条索引的 23505 换成 409——它在事务里是最后一条语句（23505 之后事务已废）。 | T5 写明两条语句与 23505 的位置；adapter 测试各一例。 |
| m3 | spec §2.6（“先整体校验载荷”） | 两种删除事件带一个 `name` 变化（`pl/app/deletion_event.go:15-18`），三种归档事件没有 `Changes`。校验要按类型写：删除不看（或恰一条 `name`），归档要求为空；`project.archived` 要求 `Entity.ID == ProjectID`。 | §2.6 加一句；`TestPlanningEventDirectory` 的畸形例各按类型。 |
| m4 | spec §2.6（事件名 `project.archived`） | 这个串已经是一个问题码：已归档的项目上的写答 409 `project.archived`（`e2e/stories/project/p4-archive.spec.ts:135`、`issue/i1-create-issue.spec.ts:372` 等）。事件类型与问题码是两个命名空间，代码里不冲突，名字也是设计 §6.1 定的；只是全仓检索会混在一起。 | 不改名。事件类型用具名常量（如 project 的 `EventArchived`），声明处写一句“与问题码同名、无关”。 |
| m5 | plan T6（“bootstrap 转换”）、T9（“接线”） | `lint-go` 开着 `unused`（`server/.golangci.yml:5`）。T6 在 `bs/navigation_ports.go` 里加的桥是未导出的类型，到 T9 才被引用的话，T6–T8 的提交 lint 不过。 | T6 同时给 `navigation.Deps` 加字段并在 `navigation.New(…)` 的调用处传入；T9 只剩用例与 `Register`。 |
| m6 | spec §2.4（`createFavorite` 201 与 200） | `api/modules/*.yaml` 里没有一个操作声明两个 2xx，这是第一个。契约规则与 `apitest` 没有“只一个成功状态”的假设（`rules_test.go`、`operations.go` 核过），但处理器测试、e2e 的辅助函数、生成的 TS 类型都是第一次遇到。 | T9 的处理器测试两个状态各一例；e2e 的 `createFavorite` 辅助函数返回状态；plane-diff 记“新建 201、已有 200”。 |
| m7 | plan T12；允许清单的 e2e 一条 | `furnishIssueRows` 的调用者有 11 处，不都是建项目的人：P4 的接口故事里项目视图是管理员建的（`furnishPlanningRows`），收藏的是成员（`p4-archive.spec.ts:179-180`）；W3 两个都是成员。收藏要求调用者是工作区管理员或成员、项目的有效成员、且看得到那个视图（不是只看自己）。现有的调用者都满足（都在项目里建工作项，工作区角色 15 或 20），但这是夹具的前置条件。`i15`、`i17` 只调 `furnishPlanningRows`，不受影响。 | `furnishIssueRows` 的注释写明前置条件；T12 逐个调用点核一遍。 |
| m8 | spec R14、§2.6（“立即不可见靠读时过滤”） | 删除窗口里（项目已删、任务还没跑）删一个文件夹，会把它里面属于这个项目的子行以**用户的**时刻与写者软删；任务随后跳过它们。行为无害，但与“随项目删除的行都带项目的时刻与删除者”的字面不同（Go 的 `unstamped`、e2e 的 `other` 都这样比，只是现有夹具不这么做）。 | §2.6 写一句这是允许的；R14 的子测试不必测，但别让后来的评审当成缺陷。 |
| m9 | spec §5；plan T16 | 交接少一条、两条的结果要写具体：① **M1-P2-trim-content**（“收藏的接口没有 `page` 类型”，`handoffs/M1-P2-trim-content.md:12,18`）——收藏的一半在 P3c 落定，最近访问随 P3d；设计 §11 把它的关闭阶段写成 P4、P7，要加 P3c。② **M6-M7-02** 留了一问“模块归档时 Plane 是否同样移除”（`handoffs/M6-closeout.md:25`）：是，`pa/app/views/module/archive.py:553-558` 按目标删所有人的。③ **M3 §1.4** 的“归档项目时的收藏处理”（`handoffs/M3-closeout.md:18`）点名写：后端在归档的事务里移除。 | T16 与 spec §5 各加。 |
| m10 | plan T16（plane-diff 的行为一行） | 抽查 Plane 后要补的差异：列表含文件夹里的（Plane 的列表只给顶层，另有按文件夹取子项的接口，`pa/app/views/workspace/favorite.py:26,85-97`）；改只收三个字段（Plane 的序列化器收任意字段，`:70-76`）；Plane 的列表只按有效的项目成员过滤，不看归档（`:26-33`）。R4 的“Plane 网页收藏不了工作区视图”是对的，后端收（删除工作区视图时连带删它的收藏，`pa/app/views/view/base.py:130-135`）。 | 清单加这三条。 |
| m11 | spec §4 末条；plan Global Constraints（测试时间） | 四个新顶层名全在第 0 路，第 0 路还另跑 navigation、planning、project、access、migrations 各包的新测试；“每路 ≤ 90 秒”是对两路各自说的。交错一条是 3 种写 × 6 种对手再加文件夹的两格，约 20 个配对，每个配对一份库的话单这一条就可能占掉大半。第 1 路只加重已有的守卫，估计几秒（P3b 实测 +4 秒）。 | plan 给四条各写一个预估；交错的配对共用一份准备好的库或分组；T13、T14 各记本机时间，接近 90 秒先报告。 |
| m12 | plan T1、T16（设计的更新） | 要改的设计文字还有：§4.3 的“32 改为 37”（今天 35，P3c 后 36，P3d 后 37）；§5 的表（`POST` 的 201 / 200、`/favorites/{id}`）；§11 里关闭阶段写“P3c”的两格（M1-P3、M1-P2）在再拆之后哪些归 P3d；§12 的 P3c 一行拆成 P3c、P3d。§4.2 的两条唯一与审计列、§3.8、§14 plan 已列。 | T1 做 §12、§11、§14；T16 做 §3.8、§4.2、§4.3、§5。 |
| m13 | spec §2.6（“`OccurredAt`（归档写下的时刻）”） | 项目的归档用例今天读两次时钟：`LockCaller` 一次、`SetArchived` 一次（`pj/app/archive_project.go:50,56`）；事件的时刻要与写进 `archived_at` 的是同一个值。事件还要工作区 id：`lockAndDecide` 的返回值今天被丢掉（`:53`），它带着 `project.WorkspaceID`。迭代、模块的用例在锁下读一次（`pl/app/archive_cycle.go:49`、`archive_module.go:59`），不用动。 | T10a 写明：锁后读一次，写与事件共用；单测断言两者相等。 |
| m14 | spec §2.3（“名字照已有命名，schema 测试按名字钉”） | 约束与索引的名字没有列出（re-90 m3 同一条）：schema 测试、`navigationNames`、m2 里按约束名识别 23505 都要它。 | 逐条写名字：`user_favorites_<列>_idx`、`…_deleted_at_idx`、两条部分唯一、各 CHECK。 |

## 3. 逐项回答

1. **可行性**（都成立）：
   - **拆 `app.go`**：见 I1。四个文件名不与已有的冲突（现有 `app_fixture_test.go`、`app_pool_test.go`、`app_test.go`）；`events.go` 加订阅者的函数要多 import `notification`、`navigation`。
   - **归档用例与发布**：三个用例的 `Deps` 都没有 `Events`（`pl/app/archive_cycle.go:18-22`、`archive_module.go:18-22`、`pj/app/archive_project.go:14-21`），模块的 `Deps` 有（`pl/module.go:156`、`pj/module.go:295`），构造处加一个实参（`pl/module.go:219-220,226-227`、`pj/module.go:333-334`）。事务是对的：迭代、模块在 `Locks.run` 的写事务里、持项目 N（`pl/app/lock.go:89-106`）；项目在 `WithinTx` 里、持工作区 S 与项目 N（`pj/app/archive_project.go:49-56`、`pj/app/lock.go:173-200`）。再归档：迭代、模块已是所求的状态时不写（`archive_cycle.go:57-59`、`archive_module.go:53-55`），自然不发；项目每次都写新时刻（`projects.sql:98-105`），照发、连带幂等——与 spec 一致。恢复共用同一个结构，只在 `u.archive` 时发。
   - **事件声明与目录**：`pl/module_events.go:12-14`、`pl/domain/events.go:9-12`、`pj/module_events.go:4`；未声明的类型发布与订阅都被拒（`bs/events.go:62-65,83-86`）。
   - **端口**：`Readable` 的桥照 `bs/notification_ports.go:83-98`；它不看归档（`readable.go:42-45`），所以项目的事实要加 `Archived`（`pj/app/navigation_ports.go:9-12`、`navigation_ports.sql:1-16`）并另加批量；planning 的批量是新查询；视图的批量与 `ShareView` 是 navigation 自己的。`FOR SHARE` 与 `LockView` 的 `FOR UPDATE`（`nav/adapter/postgres/queries/views.sql:30-36`）冲突。
   - **迁移、sqlc、运行角色**：最新是 `00052_navigation_issue_views.sql`；`mig/trigram_schema_test.go:44-45,84,106-107`（52 → 53）、`:52`（表名）；`server/sqlc.yaml:208-210` 加一行；`deploy/runtime-grants.sql:29` 加表。T3 加 sqlc 条目时查询目录已在，没有 P3b 那个“条目先于查询”的窗口。
   - **删除任务**：今天一张表、按范围选语句（`nav/app/delete_rows.go:44-78`）；改成“视图 → 收藏”两步、各自游标、共用预算做得到，种类与载荷不变（`nav/adapter/river/delete_rows.go`），在途的旧任务升级后顺带删新表。工作区一步的项目列表来自事件的 `Changes`（`nav/app/delete_rows_on_deletion.go:16-32`），每个项目各有一个任务。
   - **清理 35 → 36**：`bs/purge_catalog_test.go:154-155`、`mig/purge_indexes_test.go:40-41`；目录守卫对自引用的外键自动核对，`CASCADE` 是允许的动作（`purge_catalog_test.go:139-141`）。
2. **规则与拒绝次序**：
   - 可见谓词与设计 §3.8 一致。项目按 `Readable`：只认有效成员，不是成员的工作区管理员不算，访客也在里面（带 `OwnOnly`）；项目视图再加 `View.HiddenFrom`（`nav/domain/view.go:52-54`）；工作区视图恒可见（`favorite.*` 的角色 ⊆ `workspace_view.read`，`access/domain/rules.go:235`）。
   - 存在性：建的 422 对不存在、别的工作区、看不到、已归档逐值相同，再收藏的 200 在校验之后，父不合格（别人的、不是文件夹、已删）同一个 422——没有找到剩下的探测口。409 只关自己的文件夹名。`/favorites/{id}` 上先读**本人的**行，别人的行与不存在的同一条路；访客只有在自己有行时才到 403，没有行时是 404，与通知的先例相同（`notification/app/update_notification.go:101-124`）。两步 422 的问题见 m1。
   - 再收藏 200、次序、文件夹名、无改动的位置（409 之后）都自洽；“改成只差大小写的名字”算改动、不算冲突。
   - 工作区访客 403 与 Plane 相同（`pa/app/views/workspace/favorite.py:23,37,69,78`）。
3. **锁序**：
   - 项目 S 够：planning 的一切写（含归档、删除）先取项目行 `FOR NO KEY UPDATE`（`pj/adapter/postgres/queries/planning_ports.sql:9-15`），`SetCycleArchived`、`SetModuleArchived`、`DeleteCycle`、`DeleteModule` 各只有一个调用者，都在 `Locks.run` 里。项目的归档、成员、开关取项目 N（`projects.sql:57-63`）。`LockCaller` 取账户行 `FOR NO KEY UPDATE`（`identity/provide.go:37`），不与收藏插入时外键的 `KEY SHARE` 冲突。
   - 全序：工作区 S → 项目 S → 视图 S → 收藏行，与 `deleteView`（工作区 S → 项目 S → 视图 U → 连带）、迭代与模块的删除归档（项目 N → 连带）、项目归档（工作区 S → 项目 N → 连带）都同向。唯一的环在收藏表内部，见 I2。
   - 留在 `writesOnAProject` 之外是对的：分类 = 不是 GET、标签不是 issue / planning / asset、并且（路径含 `{project_id}` 或它的任何一行用了项目级的列）（`bs/project_write_locks_test.go:150-167`）。收藏的两条路径都没有 `{project_id}`，行只用工作区的列，就不被收进去；代价见 I5。
4. **事件与连带**：同事务、按注册次序（`bs/events.go:17-28`），navigation 排最后，迭代删除时 issue 的 `issue_membership_removals` 先跑；订阅者报错即整个发布失败、归档回滚。除 R13 的两条，没有别的测试枚举这三种类型：`planning_cycles_test.go:207` 的观察者订全部类型，用它的 `planning_deletion_memberships_test.go` 只删除不归档；`Events` 为空的三处 planning 路由（`planning_list_rows_test.go:275`、`planning_progress_snapshot_test.go:106`、`planning_deletion_fixture_test.go:156,184`）都不走归档接口（用 SQL 归档）；project 的路由都经 `projectDeletionPublisher`（`bs/issue_project_deletion_jobs_test.go:249-262`），它用 `eventTypes()` 并带着 navigation 的订阅者。项目归档按 `project_id` 删，CHECK `project ⇒ project_id = entity_identifier` 保证删得到项目自己。恢复不发。
5. **删除图与守卫**：Go 的窗口见 I4；R12 的名单是全的——先归档再断言删除目录的 Go 测试只有 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`（`bs/project_deletion_test.go:209`），e2e 只有 P4 的两条（`p4-archive.spec.ts:181-188,314-334`）。W3 的工作区断言用 `...projectTables(db)` 作 `deletedAlone`（`w3-workspace-settings.spec.ts:300-304`），新表自动在内；I21 的 `parentIDs` 要每张表经接口有行（`i21-purge.spec.ts:88-100`），夹具到位即过。`holdNavigationDeletion` 扣的是一行视图（`bs/navigation_fixture_test.go:50-57`），任务先做视图一步，收藏一步排在后面，仍然扣得住。计数与名单与 plan 列的一致（`purge_runtime_test.go:52,111`、`purge_scale_fixture_test.go:24`、`issue_project_deletion_jobs_test.go:37-39`、`e2e/fixtures/assert/catalog.ts:97-115`）。
6. **枚举操作、表、事件、任务的守卫**：`contract.Operations()` 的使用者 12 处逐个看过——矩阵覆盖会红（窗口 (c)）；路由、401、请求体 400、参数 400、问题响应的头自动跟随；删除窗口只认 `issue` 标签、访客见全部只认 `planning`、项目写锁表与草稿写锁表不涉及。根路径与标签规则（`apitest/rules_test.go:53-84`）随契约同一 Task。网页的码恰是 plan 列的三个文件加生成物。缺的见 I3。
7. **plan 的质量**：次序与数目见 I4、m5；红窗口 (b)、(c) 的名单与 P3b 的一致、是全的；变异见 I6；检查点 1 挂在关窗的 T12 是对的。四个新名的桶复算：6、13、2、37，都在第 0 路（`p0=46`，`tools/go-test-shard.sh:18`）；被加重的已有测试：`TestPermissionMatrix` 46、`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 52、`TestPurgeCatalogCoversTablesAndIncomingKeys` 59、`TestPurgeCatalogRejectsEachGap` 65、`TestNewAppNavigationJobCatalog` 68 在第 1 路。第 1 路 bootstrap 离八成（960 秒）在 main 上有 132 秒、慢 runner 上 82 秒；时间见 m11。
8. **交接与设计**：见 m9、m12。设计分给 P3c 而 spec 没有的：没有找到——最近访问整体移到 P3d（R1，设计 §12 允许实施前再拆）。
9. **Plane 的事实**（抽查后端，全部成立）：四个接口只给工作区管理员、成员（`favorite.py:23,37,69,78`）；已有与新建都答 200（`:54,64`）；`project_id` 取请求体（`:62`）；`DELETE` 是硬删（`:81`）；`is_folder` 与 `sequence` 默认 65535、按全工作区最大值加 10000（`pa/db/models/favorite.py:23-24,55-63`）；迭代、模块删除只删当前用户的（`pa/app/views/cycle/base.py:503-508`、`module/base.py:746-751`），归档删所有人的（`cycle/archive.py:598-603`、`module/archive.py:553-558`）；项目归档按 `project` 全删（`project/base.py:433`）；两种视图删除都连带（`view/base.py:130-135,386-391`）。

## 4. 核实为正确的事实（摘要）

- `00053` 空着；关键词 phase 已是 `M7/P3`，P3c 共用。
- 三种归档今天不发事件；模块的 `Deps` 里 `Events` 已接到 bootstrap（`bs/app.go:203,232`）。
- 行锁强度：`FOR SHARE` 与 `FOR NO KEY UPDATE`、`FOR UPDATE` 冲突，不与外键的 `KEY SHARE` 冲突；现有 SQL 的注释与此一致。
- 没有测试依赖 `app.go` 的位置；解析源码的测试只读测试文件与别的目录。
- 清理目录、删除目录都按 `pg_constraint` 驱动，新表的外键自动进入；自引用的 `CASCADE` 不被拦。
- e2e 的删除断言对新表自动生效，`removedBefore` 只需要 P4 的两条。
- 四个新顶层名的桶与 spec 写的一致；基线 777 个顶层测试。
- Plane 一侧九条事实成立。

## 5. 需要负责人的事

没有。I1–I6 都是 spec、plan 的写法与次序；Task 数回到 16 的两种办法（T2 做成插入段，或合并 T6、T7）是主控层面的选择。

## 6. 处理（主控，2026-11-21）

全部采纳，改入 spec 与 plan：

- I1：R2 与 §2.1 改为“保持行为的提取”，写出六条次序与失败清理、测试引用的未导出名不动；§4 的验收改为 `go build`、`go vet`、`lint-go` 与两路全量；plan T2 带**检查点 0**（本机两路全量 + 分支 CI，e2e 此时也绿），台账记每段的去处与六条次序的位置。拆分留在本段（不另做插入段）。
- I2：§2.8 定收藏行的全序——文件夹行先于条目行，同类按 `id` 升序；改先锁新的父文件夹 `FOR SHARE`、再锁本行；连带与两种范围的删除任务的理由写明。交错加“删文件夹 × 改它里面的子行（带同一个 `parent_id`）”，变异表加“改按 `id` 升序一条语句锁本行与父”。
- I3：允许清单加 `permission_matrix_seeded_test.go` 的三处（净增不超过 2 行）、`permission_matrix_seed_test.go` 的一次调用（种子函数进新文件）、navigation 模块的四个测试文件；`delete_rows_on_deletion_test.go` 不改（连带订阅者另起名字）；“不再加长”一句写明这两处例外。
- I4：T3 分 a（迁移、schema、sqlc、运行角色、动作、清理）与 b（删除任务的步骤、两处谓词、Go 种子），窗口 (a) 只跨 a → b、带测试名，T3b 之后加一次 Go 两路的中间检查；T10 分为 T8a（三种事件）与 T8b（连带、`deleteView`、R12、R13 的两组订阅者）。回到 16 个：领域并存储（T4），端口并读与建（T5，re-90 的先例）。
- I5：spec §4 写明矩阵只用工作区列（理由：`writesOnAProject` 按操作 id 分类），建的变体为文件夹、工作区视图与一个带 `prepare` 的项目目标；改、删用每列自己的文件夹。H19 的项目级格（只看自己的访客、管理员兼访客的两种开关、私有项目非成员、撤权）写进 H15。
- I6：变异表补六行（活的子行与“删文件夹 × 在它里面新建”、读时过滤已归档的活行、开关不看、项目再归档照发与时刻相等、`deleteView` 的连带同一时刻与写者、一个 422 装两个字段）；spec §4 的 H15、H16 与交错跟上。
- m1：建的目标与父合成一个 422。m2：条目写明冲突目标，文件夹按约束名换 409。m3：订阅者按类型校验、另起名字。m4：事件名不改，命名常量加注释。m5：端口的桥在加它的 Task 里接进 `navigation.Deps`。m6：§2.4 记两个 2xx。m7：plan 写明 `furnishIssueRows` 的调用者条件。m8：§2.6 写明删除窗口里删文件夹的时刻允许。m9：§5 加 M1-P2，M6-M7-02 的一问答“是”，M3 §1.4 写明归档的处理。m10：T14 的 plane-diff 清单加三条。m11：plan 给四条预估，交错的配对共用库、分组，实测记台账。m12：T1 改设计 §11、§12、§14，T14 改 §3.8、§4.2、§4.3、§5。m13：项目归档锁后读一次时钟，写与事件共用，工作区 id 取 `lockAndDecide` 的返回值。m14：§2.3 逐条写名字。

