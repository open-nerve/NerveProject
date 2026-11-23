# M7 / P2a：收集箱条目的生命周期——提交、分诊与工作项的边界

> 状态：spec 经预检 [re-85](../../../reviews/re-85-M7P2a预检.md) 修正后通过。基线 main `31f38580629280b8b33a81e770fb5b39e734f045`（P1c 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H8、H10、H11）、§3.5、§3.6、§3.10、§3.11、§4、§5、§6.1、§9、§15 A/B/D2/E。现状调研见本阶段台账 `research/p2a-facts.md`（ignored，随阶段归档；下文 DP-n 指它 §15 的决定点）。

## 1. 目标与边界（含 P2a / P2b 的重新划分）

设计 §12 原把“删除图”放在 P2b。调研发现：`intake_issues` 一建，按目录核对的删除守卫（Go 的 `keysTo`/`deletionViolations`、e2e 的 `expectIssueDeleted`）就要求每张表在被删的父行下各有一行；e2e 只经接口造数据，而一个能经 `DELETE /issues/{id}` 删除的工作项要带条目行，只能是**已接受**的条目（在收集箱里的工作项对工作项的删除答 409）。所以表、删除图与接受必须在同一段；设计 §4.3 不许为此开守卫的例外。按“条目的生命周期”与“列表和其余”重新划分（R1）：

- **P2a（本 spec）**：`intake_issues` 表、删除图与 60 天清理；动态 `verb` 的放宽；提交（Triage 的新建路径）与开关；分诊（五个状态的迁移、接受、动态、重复目标）；条目作为工作项的写入边界（409 `issue.in_triage`、另一端 422）与搜索的排除；详情的 `intake`；两个新操作的权限矩阵；H8、H10 的接口证据与 e2e 故事，H11 的读写边界部分。
- **P2b**：收集箱列表、内容的修改 `PATCH /intake-issues/{id}`、删除 `DELETE /intake-issues/{id}`、待处理数、Triage 宿主的资产判定（`intake.update`，与内容修改同一个动作与时段，含 `asset.yaml` 与资产路径读开关）、它们的权限矩阵与 H9、H11 其余的证据。

P2a 不做：网页（P5）；Plane 的 `source`、`source_email`、`external_*`、`extra`（M7-01，不建）；分诊状态的接口与前端（设计 §3.6，M3 交接 §1.2 随之关闭）。

**P2a 合入后、P2b 之前 main 的已知状态**：拒绝、重复的条目是终态，P2b 之前删不掉、改不了内容、列不出来（只能按 id、编号与通知找到）；待处理的可以接受再删。Triage 宿主的资产照旧按 `issue.update`（管理员、成员）判定。不算不安全：`intake_view` 默认关（`00010`），网页的收集箱还在打 Plane 的旧地址，没有页面能造出条目；项目删除与清理照常带走它们。已接受的工作项的动态里会有一行 `intake`，旧的 `inbox.tsx` 走通用文字的分支，不崩。阶段记录向负责人说明。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/migrations/sql/00050_issue_intake_issues.sql` | 表、约束、索引（§2.2），含 Down |
| `server/migrations/sql/00051_issue_issue_activities_intake_verbs.sql` | 放宽 `issue_activities_verb_check`（§2.2），含 Down |
| `server/migrations/` 测试 | 新表的约束与索引名、CHECK 的反例；verb 的两条新反例；`purge_indexes_test` 33 → 34；`trigram_schema_test` 迁移数 49 → 51、表清单、Down 起点 |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql` | issue 的 schema 清单加两个迁移；运行角色对新表的四种 DML |
| `server/internal/modules/issue/` | 领域（状态与迁移、有效状态、条目的事实、分诊请求的规则）；存储（条目行的插入与更新、`GetIssue` 系三条查询带条目、搜索的排除、删除与清理）；用例（提交、分诊、修改核心的接缝、六个源端守卫）；HTTP（两个操作、详情的 `intake`）；`Locks` 加时钟 |
| `server/internal/modules/project/` | `LockIssueProject`、`LockIssueProjects` 带 `intake_view`（§2.4） |
| `server/internal/modules/access/` | 动作 `intake.submit`、`intake.triage` 的规则（§2.9） |
| `server/internal/bootstrap/` | 端口转换；删除与清理的守卫、夹具；权限矩阵、写锁表；组合测试、交错 |
| `api/modules/issue.yaml`、`api/modules/notification.yaml` | 两个操作、`IssueDetail.intake`、新码与受影响操作的码；通知契约里 `state_group` 的说明加 `triage`（DP-17） |
| `web/` | 生成的类型、`error-messages.ts` 与两份 `errors.json` 的新码；`IssueDetail` 的夹具 `fake-issues.ts`、`issue.store.types.test.ts`；`issue.store.ts` 把列表行盖到详情上时保留 `intake`（P1a 的 `is_subscribed` 同一改法）及其测试 |
| `e2e/` | 删除守卫的目录（`asynchronousDeletionTable`）与夹具（§2.10）；H8、H10 的接口故事 |
| `docs/` | plane-diff；硬规则 03 §4；M3 交接 §1.2、M4 交接 M7-01 的处理结果；设计 §12、§14；`tools/keywords.json` 的 phase；阶段记录 |

### 2.2 表与迁移

`intake_issues`（`00050`）：

| 列 | 类型与约束 |
|---|---|
| `id` | uuid 主键 |
| `workspace_id`、`project_id` | 非空，外键 `CASCADE` |
| `issue_id` | 非空，外键 `issues` `CASCADE` |
| `status` | 非空，CHECK `pending`、`snoozed`、`accepted`、`declined`、`duplicate` |
| `snoozed_till` | 可空 timestamptz |
| `duplicate_to_id` | 可空 uuid，**不是外键**（§15 D2） |
| `created_by_id`、`updated_by_id` | 账户外键 `SET NULL`（提交者 = `created_by_id`，与工作项的创建者相同） |
| `created_at`、`updated_at`、`deleted_at` | 审计与软删 |

- CHECK：`status = 'snoozed'` ⇔ `snoozed_till IS NOT NULL`；`status = 'duplicate'` ⇔ `duplicate_to_id IS NOT NULL`。
- 部分唯一 `(issue_id) WHERE deleted_at IS NULL`；每个外键一个不带条件的索引；`deleted_at` 的部分索引（清理）；`(project_id, status) WHERE deleted_at IS NULL`（P2b 的列表与待处理数，P2a 建好）。

`issue_activities`（`00051`）：`issue_activities_verb_check` 改为 `verb IN ('created','updated','deleted') OR (field IS NOT DISTINCT FROM 'intake' AND verb IN ('pending','snoozed','accepted','declined','duplicate'))`，名字不变（`field` 可空，`=` 会让空的 `field` 放行，预检 I1）。Down 先删 `field = 'intake'` 的动态再收回 CHECK。

### 2.3 Triage 与条目的事实（R2）

- **权威的标记是条目行**：工作项“在收集箱里”当且仅当它有未删除的条目行且 `status <> 'accepted'`；这样的工作项的状态就是项目的 Triage 状态。两者按构造等价：只有 `InsertIssue`、`UpdateIssue` 写 `state_id`；只有提交把工作项放进 Triage，只有接受把它移出；状态接口碰不到 Triage 状态，工作项的修改不能以 Triage 为目标。一个 bootstrap 不变式测试在提交、各种分诊、删除之后钉住等价，并钉住“在收集箱里的工作项没有父、没有子、没有关联行、不在迭代模块里”（DP-7）。
- `domain.Issue` 加 `Intake *IntakeFacts{Status, SnoozedTill, DuplicateToID}`：`GetIssue`、`GetIssues`、`SubIssues` 三条查询共用一种行，同步 `LEFT JOIN intake_issues ii ON ii.issue_id = i.id AND ii.deleted_at IS NULL`（条件在 `ON` 里，用得上部分唯一索引）。注释写明：只有这三条的行带 `Intake`；列表与搜索的行不带，不得在它们上面调 `InTriage()`（靠不变式安全：它们不含在收集箱里的工作项）。
- **有效状态**：存的状态为 `snoozed` 且 `snoozed_till <= now` 时为 `pending`（设计 §3.5，按读那一刻到期）。分诊、回答、守卫都用有效状态，存的值不因到期而改写。`now`：写用那一次写在最后一把锁之后读的时刻；读的回答由 `Locks` 新加的时钟按需读（只有存的状态为暂缓时要此刻，每个回答至多一次；每个回答都读会让无改动的归档也读时钟，打破 M4 的 `TestIssueArchivalEvents`，T9 的裁定）。

### 2.4 提交 `POST /projects/{project_id}/intake-issues`

- 请求体（平的，`additionalProperties: false`，DP-13）：`name`（必填）、`description_html`、`priority`、`label_ids`、`assignee_ids`、`target_date`。校验照 `CheckNewIssue`。
- 授权 `intake.submit`（项目的管理员、成员、访客）。次序：请求体 400/422 → 项目看不到 404 `project.not_found` → 403 → 409 `project.archived` → 409 `issue.intake_disabled` → 锁下的 422（标签、负责人等，照新建核心）。
- 编排：`Locks.run(write{project, intake.submit})`；锁下读 `LockedProject.IntakeView`（project 的两条锁读加列；bootstrap 按类型转换，结构同步；DP-3：只在锁下判，issue 的 `readMembership` 先例）；归档先于开关（与 planning 一致）。
- 写：新建核心 `createLocked` 加一个放置参数，取项目的 Triage 状态（锁下 `StateCatalog` 的 `GroupTriage`）代替 `pickState`；缺 Triage 状态是内部错误（同缺默认状态）。草稿发布与普通新建照旧取默认状态。编号、默认负责人、描述、提及、自动关注、创建动态、描述版本、`issue.created` 照常；同一事务里再插条目行（`pending`，提交者），再读回一次作回答（核心的 `answer` 不变）。事件先于条目行发布：事务内的订阅者不读条目行（DP-2，写在用例的注释里）。
- 回答 201 `IssueDetail`（`intake.status = pending`）。通知照 P1b：负责人、被提及的人按 `issue.read` 复核后收到；快照的状态是 Triage。

### 2.5 分诊 `POST /intake-issues/{issue_id}/triage`

- 请求体 `{status, snoozed_till?, duplicate_to_id?}`；`{issue_id}` 是工作项 id（Plane 相同）。`status` 在契约里是五值的枚举；值不合法 422（domain 判定，与 P1c 全部已读的请求体同一口径），类型不对 400（bodyshape）。
- 授权 `intake.triage`（项目管理员，以及是项目有效成员的工作区管理员，§15 B）。
- 次序（I9）：请求体 400/422 → 工作项看不到 404 `issue.intake_not_found` → 403 → 没有条目行 404 `issue.intake_not_found`（`write.check`，预检与锁下各一次）→ 409 `project.archived` → 409 `issue.intake_disabled` → 409 `issue.intake_transition_not_allowed` → 锁下的 422（`snoozed_till`、`duplicate_to_id`）。两种 404 与随机 id 逐值相同；所以成员对一个看得到的普通工作项得 403，管理员得 404。
- 迁移（R3，按有效状态）：

| 从 | 可到 |
|---|---|
| `pending` | `accepted`、`declined`、`duplicate`、`snoozed` |
| `snoozed` | `accepted`、`declined`、`duplicate`、`snoozed`（改日期）、`pending`（取消暂缓） |
| `accepted`、`declined`、`duplicate` | 无（终态） |

  其余 409（含 `pending → pending`；重复的请求在终态后得 409，不做幂等的 200）。
- 字段：`snoozed` 须给 `snoozed_till`（缺 `required`），截到微秒后须晚于 `now`（`must_be_future`）；`duplicate` 须给 `duplicate_to_id`：未删除、同一项目、调用者看得到（同项目的授予与 `HiddenFrom`）、不在收集箱里、不是自己，否则 422 `not_allowed`（字段 `duplicate_to_id`，各种不合格同一个码与说明，排在 403 与迁移 409 之后，不透露存在；“是自己”也在锁下这一步，不提前到请求体）。已归档的目标允许（设计的四条，选择器经搜索不列已归档的，登记）。别的状态给了这两个字段 422 `not_allowed`。
- 编排：`Locks.run(write{issue, intake.triage, check: 有条目行})`，条目的事实来自锁下重读的 `h.issue.Intake`（不另锁条目行，写条目行的每条路径都持项目锁）。时钟在最后一把锁之后读一次，整个请求用这一个时刻。
- 写：条目行改状态、`snoozed_till`、`duplicate_to_id`、`updated_*`；一条 `field = 'intake'`、`verb = 新状态` 的动态，`old_value`、`new_value` 为有效的旧状态与新状态的名字（到期的暂缓被接受记 `pending → accepted`），不存重复目标的 id（R7）。**接受**另经修改核心的接缝（R6）以补丁 `{state_id: 默认状态}`、同一个时刻改状态：默认状态在锁下经 `StateCatalog` 读（`pickState(nil)`），状态动态、`completed_at`（随默认状态的组）、`issue.updated`（只有状态变化）照常，状态变化不重算 `sort_order`（与普通修改一致）；`intake` 动态写在状态动态之前。拒绝、重复、暂缓、取消暂缓不发布事件、不通知（Plane 相同）。
- 回答 200 `IssueDetail`。

### 2.6 条目作为工作项的边界（R4）

- **源端 409 `issue.in_triage`**（选择性的守卫，经 `write.check`，DP-6；链接、评论、表情、关注不加）：`updateIssue`、`archiveIssue`、`unarchiveIssue`、`deleteIssue`、`addSubIssues`（父端）、`createIssueRelations`（源端）六个。`deleteIssueRelation` 不加：任一端的编排不跑 `write.check`，而在收集箱里的工作项按不变式没有关联行（预检 I3）。次序在 404、403 之后，`project.archived`、`issue.archived` 之前（DP-9；硬规则 03 §4 的拒绝次序登记这一例外）。`unarchiveIssue` 的守卫先于“已是所求状态”的空操作。
- **另一端 422**（沿用已有的码，DP-8）：父工作项（新建、修改、草稿发布：`ParentNotAllowed`）、子工作项（`SubIssueNotAllowed`）、关联的另一端（`RelatedNotAllowed`），经 `inReach` 与关联的另一端判定。
- **搜索**：候选不含在收集箱里的工作项（`NOT EXISTS` 未删除且 `status <> 'accepted'` 的条目行，issue 自己的表）；`issue_id` 指向在收集箱里的工作项时 404，同已归档的源（DP-4、DP-5）。重复目标的选择器（P5）用同一个搜索。
- **列表**：项目列表、成员列表照旧由状态目录排除；迭代、模块与 `sub_issues_count` 靠不变式，由测试钉住。
- **读**：详情、按编号取、评论、动态、表情、链接、附件照工作项的读权限；描述版本照 M4 的 `issue_description_version.read`（登记差异）。

### 2.7 修改核心的接缝（R6）

把 `UpdateIssue.Execute` 闭包里的那一段原样抽成 `updateLocked(ctx, actor, h held, p domain.IssuePatch, clock Clock) (domain.Issue, error)`（实施时由 `at time.Time` 改为时钟：照字面由调用者先读时刻，会让不变的修改也读时钟、把读时钟提前到检查之前、带图片时早于最后那把内容锁；传时钟则搬移的代码一字不改，T6 的裁定）；`Execute` 只剩 `CheckIssuePatch`、`write{…, check: 源端守卫}` 与调用它（传业务时钟）。接受以补丁 `{StateID: 默认状态}` 与一个只答分诊那一刻的时钟调它（一个请求一个时刻）；源端守卫挂在 `Execute` 的 `write.check` 上，不在抽出的那一段里，所以接受不经它。P2b 的内容修改用同一个接缝换白名单与授权。

### 2.8 详情的 `intake`（DP-16）

`IssueDetail.intake`：`{status, snoozed_till, duplicate_to_id}` 或 `null`（必填、可为 null）。有条目行（含已接受）时给出；`status` 为有效状态；`snoozed_till` 只在有效状态为 `snoozed` 时给；`duplicate_to_id` 只在目标未删除且读的人看得到时给（同项目的授予与 `HiddenFrom`；只在状态为 `duplicate` 时多一次按主键的读），否则 `null`。七个回答 `IssueDetail` 的操作都带（`seenOne`；提交与分诊的回答读回后同样经它）；新建核心的 `answer` 不变（普通新建与草稿发布没有条目）；列表行不带。

### 2.9 契约、动作与码

- 操作：`submitIntakeIssue`（201 `IssueDetail`）、`triageIntakeIssue`（200 `IssueDetail`）。
- 动作：`intake.submit`（管理员、成员、访客）、`intake.triage`（管理员）。`intake.read`、`intake.update`、`intake.delete` 在 P2b 随它们的操作加。
- 新码：`issue.in_triage`（409）、`issue.intake_disabled`（409）、`issue.intake_not_found`（404）、`issue.intake_transition_not_allowed`（409）。各操作写明 `x-problem-codes`；六个受源端守卫影响的已有操作加 `issue.in_triage`；处理器测试逐码回答。
- 通知契约 `state_group` 的说明加 `triage`（只改文字，DP-17）。

### 2.10 删除图与清理

- 工作项删除：条目行随工作项在原事务里软删（`issueRowSteps` 里紧跟 `issues` 的子表之后）。P2a 里只有已接受的条目能经 `DELETE /issues` 删除。
- 项目、工作区删除：issue 的分批任务加 `intake_issues`（`DeletionRowSteps`、`deletion_remaining`，位置写死并由已有的次序测试钉住）。
- 60 天清理：issue 的清理步骤在 `issues` 之前加 `intake_issues`；清理目录 33 → 34。
- `duplicate_to_id` 指向的工作项被删：不动，读时为 `null`（§2.8）。
- Go 守卫：删除守卫以 SQL 种条目行（P1b 先例）；`issue_project_deletion_jobs_test` 的 `projectIssueTable` 显式加 `intake_issues`（前缀不是 `issue_`）。
- e2e（预检 I2）：`asynchronousDeletionTable` 加 `intake_issues`；新夹具 `furnishAcceptedIntake`（管理员打开开关、提交、接受、再经 `updateIssue` 放到指定工作项之下）由 `expectIssueDeleted` 的四处调用（i15 两处、i17 两处）各调一次，i21 以它代替孙工作项的新建（那里断言根之下恰为三个工作项，断言不变）；`furnishPlanningRows` 在已有的 `changeProject` 里打开开关，待处理的那一次提交放在 `furnishIssueRows`（放进 `furnishPlanningRows` 会让 i15 写死的编号后移一位；项目、工作区的删除故事都经 `furnishIssueRows`），供项目、工作区的删除断言（T10 的两条裁定）。

## 3. 局部裁定

- **R1 重新划分 P2a、P2b**：理由见 §1；不开守卫的例外。Triage 宿主的资产（预检 I5 的三处缺口：资产契约、开关的来源、动作的选择次序）随内容修改移到 P2b。设计 §12 的两行与 §14 随之改，阶段记录向负责人说明。
- **R2 条目行是权威的标记**：issue 自己的表、`precheck` 里就能判定，并给详情的 `intake` 用；与状态组的等价由不变式测试钉住。
- **R3 迁移规则与有效状态**：照设计 §3.5；终态与 `pending → pending` 都是 409；到期的暂缓按待处理迁移。
- **R4 边界**：六个源端 409、另一端沿用 422；守卫按用例选择性加，链接等照常可写；搜索在 issue 的 SQL 里用条目行排除。
- **R5 资产移到 P2b**：见 R1。
- **R6 接受经修改核心的接缝**：§2.7；一个请求一个时刻。
- **R7 分诊的动态不指名重复目标**：只存状态名，免得动态里出现读的人看不到的工作项。

## 4. 验收标准

- 契约、生成物、处理器逐码；`make gen-check` 无差异；网页的类型检查与 store 的测试（`intake` 随列表行保留）。
- 迁移测试：约束、索引名、CHECK 的反例（暂缓无日期、重复无目标、别的状态带日期或目标、状态值、同一工作项两行未删除；`field` 为 NULL 配 `accepted`、`field = 'state'` 配 `accepted` 都被拒；已有的 `commented` 反例不变）。
- 组合测试（真实的 app，经接口）：
  - H8：管理员、成员、访客各提交一次，回答 201、`intake.status = pending`、状态是 Triage；项目列表、成员列表、搜索都没有它；负责人与被提及的人收到通知；开关关着 409 而读照常；已归档的项目 409 `project.archived` 先于开关；请求体里的状态、父工作项、迭代等 400。
  - H10：五种迁移逐一、非法迁移 409、终态后 409；接受之后在项目列表与搜索里出现、状态为默认状态、`intake` 动态在状态动态之前、关注者收到状态通知、一个请求一个时刻；拒绝、重复、暂缓不通知；`snoozed_till` 过去、等于此刻、缺失 422；到期的暂缓按待处理迁移且动态记 `pending`；重复目标的各种不合格 422 同一回答、已归档的目标可以；拒绝次序：成员对普通工作项 403、管理员对普通工作项 404、只看自己的管理员兼访客对别人的条目 404，两种 404 与随机 id 逐值相同。
  - H11 的边界：六个源端写 409 `issue.in_triage`（次序先于归档）；做父、子、关联另一端 422；评论、表情、链接、关注照常；搜索的源 404。
  - 详情的 `intake`：七个操作都带；有效状态；看不到或已删除的重复目标为 `null`。
  - 不变式：条目行未接受 ⇔ 状态是 Triage；在收集箱里的工作项没有父、子、关联、迭代模块。
- 删除与清理：工作项、项目、工作区删除的守卫转绿；清理目录 34；清理与删除的规模夹具含新表；e2e 的五处删除故事与项目、工作区的删除故事转绿。
- 权限矩阵：两个新操作的行（开关在种子里打开；先核已有各行的 `check` 不数新种的 Triage 工作项）；六个源端守卫在矩阵里至少各一行；写锁表两行。
- 交错：两次并发的分诊（第二次得 409）；分诊与开关的关闭；接受与默认状态的改设；分诊与项目删除。
- e2e：H8、H10 的接口故事（PAT 与会话对等，H1 的写法），本机各约 3 秒。
- 每个可见、迁移、守卫、锁的性质至少一个能让测试失败的变异（含“接受读了第二次时钟”）。
- 测试时间：CI 第 0 路、第 1 路 Test 步各自的增量 ≤ 90 秒（本机约 60 秒）；两个检查点记本机与分支 CI 两路。

## 5. 后续责任

- P2b：列表、内容修改（经 §2.7 的接缝）、删除接口、待处理数（含到期暂缓）、Triage 宿主的资产、`intake.read`、`intake.update`、`intake.delete`、它们的矩阵与 H9、H11 的其余证据。
- P5：网页的收集箱改用新接口与状态名，删除旧的 `inbox-issue` 服务与 `FORMS` 分支，`intake` 的动态渲染。
  - **处理结果（M7/P5a）：** 网页的收集箱改用新接口（`IntakeService`、issue 根 store 的子 store `root.intake`，T2–T5）与状态名（`effectiveStatus`、`inbox_issue.*` 按状态名的文案）；旧的 `inbox-issue` 服务在 T6c、`FORMS` 分支在 T6a 删除；`intake` 的动态按状态名渲染（T9）（[P5a spec](P5a-web-intake.md) §2.1–§2.10）。

## 6. 风险

- 新建核心加一个参数、修改核心抽出接缝：草稿发布与普通新建、修改的回归由已有测试覆盖。
- `GetIssue` 系三条查询同步加左连接：每次读工作项多一个按部分唯一索引的连接，规模测试复核。
- 测试时间：两路都在 670 秒左右；按 §4 的预算，检查点各量一次。
