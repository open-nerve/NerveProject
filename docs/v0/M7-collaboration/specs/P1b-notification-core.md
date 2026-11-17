# M7 / P1b：通知模块的核心（生成与删除）

> 状态：spec 经预检 [re-83](../../../reviews/re-83-M7P1b预检.md) 修正后通过。基线 main `9a3a388a7a3be57df0c4da76d539ed7a5cead34a`（P1a 的 noff）。设计见 [M7 设计](../M7-design.md) §3.2–§3.4、§4、§6、§8、§9，负责人 2026-11-17 的决定 C（通知的规则）见设计 §15；P1a 交来的事件变化与后续责任见 [P1a spec](P1a-subscriptions-events.md) §2.2、§5。现状调研见本阶段台账 `research/p1b-facts.md`（ignored，随阶段归档）。

## 1. 目标与边界

设计 §12 的 P1b 约 18–20 个 Task，超过一段的上限，实施之前拆为两段（共用关键词 phase `M7/P1`）：

- **P1b（本 spec）**：`notification` 模块的核心。`notifications` 表；事件的订阅者与分发任务；它要的 issue、project、workspace、access 端口（工作项行锁、关注者、快照、评论正文、收件人的批量判定）；关联事件追加请求所在的一端；工作项删除时的同事务软删、项目与工作区删除的分批任务、60 天清理；规模与写入基准。H2–H4 的数据库证据（经真实的写入产生通知行）。
- **P1c（P1b 合入后再写 spec 与 plan）**：通知的接口（列表、修改、全部已读、未读数）、读时的可见过滤（含已归档项目）与另一端的解析、权限矩阵、H2–H7 的接口证据与 e2e 故事。

P1b 不做：任何通知的接口与读（P1c）；网页（P4）。模块在 P1b 没有接口，列为没有动作的模块（`actions_test` 的 `actionlessModules`，写一行理由），P1c 随第一个接口加动作与规则（R2）。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/notification/` | 新模块：`module.go`（`Provide`、`New`、`Register` 空、`Jobs`、`EnqueuedKinds`）、`module_events.go`（`Subscribers`）、`module_purge.go`；`domain/`（可通知的变化、动词、变化序号、收件人、类型、快照的形状，§2.3–§2.5）；`app/`（端口、订阅者、分发用例、删除与清理）；`adapter/postgres/`（sqlc）；`adapter/river/`（分发与分批删除的任务） |
| `server/migrations/sql/00049_notification_notifications.sql` | 表、约束、索引（§2.2），含 Down |
| `server/migrations/` 测试 | `notification_schema_test.go`（约束与索引名、CHECK 的反例）；`purge_indexes_test` 32 → 33；`trigram_schema_test` 迁移数 48 → 49、表清单、Down 起点 |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql` | 新模块的条目；运行角色对新表的四种 DML |
| `server/internal/modules/issue/` | 端口（§2.6）与关联事件追加的 `requested_on` 变化（§2.3）；由 `issue.Provide` 交出 |
| `server/internal/modules/{workspace,project,access}/` | 收件人的批量判定（§2.7） |
| `server/internal/bootstrap/` | 接线（`Provide`、订阅者的次序、`New`、任务与种类、清理次序 asset → notification → issue → …）；端口转换 `notification_ports.go`（含快照的补齐，§2.6）；`job_inserter.go` 的断言；`actions_test` 的 `actionlessModules`；删除与清理的守卫、夹具（§2.8）；组合测试、交错、规模 |
| `server/internal/platform/config/`、`server/configs/`、`README.md` | `notification.delete_batch`、`notification.delete_budget`（M6 `planning` 的先例） |
| `e2e/fixtures/` | `assert/catalog.ts` 的 `asynchronousDeletionTable` 加 `notifications`；删除故事的夹具经接口产生一行通知（§2.8） |
| `docs/` | `plane-diff.md`；README；M4 交接 M7-02、M5 交接 M5-M7-02 的“处理结果（M7/P1b）”；设计 §6.1、§12、§14；阶段记录 |

### 2.2 表

`notifications`（设计 §4.2）：

- `id`；`workspace_id NOT NULL REFERENCES workspaces ON DELETE CASCADE`；`project_id NOT NULL REFERENCES projects ON DELETE CASCADE`；`receiver_id NOT NULL REFERENCES users ON DELETE CASCADE`；`triggered_by_id REFERENCES users ON DELETE SET NULL`（空 = nerve 自己）。
- `issue_id NOT NULL`、`issue_created_by_id`（都不是外键）；`sender NOT NULL`（CHECK：`created`、`assigned`、`subscribed`、`mentioned`）；`data jsonb NOT NULL`（CHECK：顶层恰为 `issue`、`change` 两个对象，§2.5）；`event_id NOT NULL`、`change_seq smallint NOT NULL CHECK (change_seq >= -1)`。
- `read_at`、`archived_at`、`snoozed_till`、`created_at NOT NULL`、`updated_at NOT NULL`、`deleted_at`。审计的 `created_by_id`、`updated_by_id` 不建：行由 nerve 写，之后只有收件人自己改。

索引：部分唯一 `(event_id, change_seq, receiver_id) WHERE deleted_at IS NULL`（任务重试幂等）；`(receiver_id, workspace_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`（列表）；`(receiver_id, workspace_id) WHERE deleted_at IS NULL AND read_at IS NULL AND archived_at IS NULL`（未读数）；`(workspace_id, issue_id) WHERE deleted_at IS NULL`（工作项删除）；三条外键各一个不带条件的索引；`notifications_deleted_at_idx (deleted_at) WHERE deleted_at IS NOT NULL`（清理）。

### 2.3 可通知的变化、动词与序号

变化序号是那个变化在事件的变化列表（订阅者收到的原始列表）里的位置，从 0 起；创建本身用 `-1`（R4）。

| 事件 | 可通知的变化 → `change.field`、`change.verb`、序号 |
|---|---|
| `issue.created` | 创建本身 → `issue`、`created`、`-1`；每个 `assignees` 变化 → `assignees`、`created`，新值为显示名、`new_id` 为账户，序号为它的位置 |
| `issue.updated` | `name`、`parent`、`priority`、`state`、`start_date`、`target_date`、`labels`、`assignees` 的每个变化 → 同名字段、`updated`（与 M4 动态行的动词一致：动态一律 `updated`，加入与移出由新旧值区分，P4 的卡片照此判断）；`description`、`cycles`、`modules`、`sort_order`、`mention` 不是 |
| `issue.archived`、`issue.unarchived` | `archived_at` → `archived_at`、`updated`（新值 `manual_archive`/`archive`/`restore`，照动态） |
| `issue.comment_created/updated/deleted` | `comment_html` → `comment`、`created`/`updated`/`deleted`，带评论 id；建立与修改带摘录（§2.5） |
| `issue.link_created`、`issue.link_deleted` | `url` → `link`、`created`/`deleted`，新旧值为地址 |
| `issue.link_updated` | 只有带 `url` 变化时一条（与动态一致）→ `link`、`updated`；只改标题不通知 |
| `issue.relation_added/removed` | `requested_on`（本段追加，见下）→ 字段为请求那一端读到的关联类型（如 `relates_to`、`duplicate`、`blocking`、`blocked_by`），`created`/`deleted`；另一端的 id 取 `issue`、`related_issue` 两者中不是请求端的那个，记在 `new_id`/`old_id`（只存 id，P1c 读时解析） |
| `asset.attachment_added/deleted` | `issue` → `attachment`、`created`/`deleted` |
| 各事件的 `mention` 变化 | 提及通知 → `issue.created`、`issue.updated` 的为 `description`，评论事件的为 `comment`，动词 `mentioned`；序号为收件人那一条 `mention` 的位置（每个收件人唯一） |

`issue.deleted`、表情、迭代转移不通知；评论、链接事件末尾的 `issue` 变化是所在工作项，不是可通知的变化（P1a §5）。

**关联事件的请求端**：`relationEvent` 用存储的规范行，以 `blocking` 发起的关联存为对方的 `blocked_by`，事件的 `issue` 因此可能是对方。本段在关联的两个事件末尾追加一个变化 `requested_on`（只加不改，设计 §6.1 的口径）：建立时是请求所在的工作项，移除时是移除被授权的那一端（用例里的 `h.own()`，与动态的第一行同一端）；值为那一端读到的类型（`Relation.TypeFrom`），id 为那一端；建立在 New 一侧、移除在 Old 一侧。通知只发到这一端的关注者（设计 §3.2：另一端的动态被跳过）。

草稿发布带附件时，创建一条之外每个附件另有一条（照事件，登记差异，R5）。

### 2.4 收件人与类型

对一个事件（工作项 I、执行者 A，可为空）：

- **提及通知**：每个 `mention` 变化的账户，去掉 A。
- **关注通知**：I 此刻的关注者（任务里读，含这次写自动加入的），去掉 A、去掉这个事件里收提及通知的人；每个可通知的变化一条。
- **复核**：上面两类合起来，按任务那一刻的 `issue.read` 与只看自己批量判定（§2.7），读不到的去掉；只看自己的人再按快照里的创建者执行 `HiddenFrom`。
- **类型**：提及为 `mentioned`；否则收件人是 I 的创建者为 `created`，是 I 当前的负责人为 `assigned`，其余为 `subscribed`（设计 §3.3；Plane 的 D14 不照搬）。发送时定下。

### 2.5 快照（`data`）

```json
{"issue": {"id": "…", "name": "…", "sequence_id": 12, "identifier": "PROJ", "state_name": "In Progress", "state_group": "started"},
 "change": {"field": "state", "verb": "updated", "old_value": "Todo", "new_value": "In Progress", "old_id": "…", "new_id": "…", "comment_id": "…", "excerpt": "…"}}
```

`change` 里没有的键不写。`parent` 与关联的另一端只存 id，不存编号与名字（设计 §3.3，P1c 读时按读的人的可见解析）。评论的摘录是评论存着的纯文本（`comment_stripped`）的前 200 个字符：提及在纯文本里不出现（清洗只留提及的 id），照此接受（R6）；删除评论的通知没有摘录。

### 2.6 端口

issue 由 `Provide` 交出（只依赖连接池，照 `module_planning_stats.go`）：

- `IssueRowLock.ShareIssue`：P1a 的语句不变，从 `IssueReader` 移到它内嵌的小接口（R8），分发任务与 `Locks` 共用。
- `SubscriberIDs(issue)`：未删除的关注者，升序。
- `NotificationIssue(issue)`：issue 自己的列——名字、编号、`state_id`、创建者、当前的负责人（一条 sqlc，不加锁）。
- `CommentText(comment)`：未删除评论的纯文本；已删除的答没有。
- `ProjectIssuesRemain(project)`：项目里还有没有未软删的工作项（§2.8）。

状态名与组、项目标识由 bootstrap 的 `notification_ports.go` 经 project 已有的不加锁的读补齐（状态目录、项目标识）。**锁序**：分发在 `ShareIssue` 之后只做不加锁的读与一条 INSERT，不得经 `ProjectGate.Lock`、`LockMany` 或任何取项目、工作区行锁的端口（否则与“持项目锁、等工作项行”的写形成环）。

### 2.7 收件人的批量判定

access 加 `DecideMany(ctx, action, workspace, project, accounts)`：workspace 一次读出这些账户在这个工作区的有效角色，project 一次读出项目的事实（是否删除、所属工作区、公开、`guest_view_all_features`）与这些账户的项目角色；项目不属于这个工作区按“没有项目”。然后对每个账户调与 `Authorize` 同一个 `domain.Decide(RuleFor(action), Facts{…})`，回答每个账户的允许与只看自己。`Creator` 恒为假，所以 `DecideMany` 只接受没有 `AllowCreator` 的规则，否则报错（`issue.read` 没有）。工作区与项目取自分发锁住的工作项行。

一个等价测试在权限矩阵的种子上逐格比对 `DecideMany` 与 `Authorize`（`issue.read`），另加“别的工作区的项目”“已删除的工作区”两格。

### 2.8 删除与清理

- **工作项删除**：订阅者 `forget_deleted_issue_notifications`（`issue.deleted`，原事务里）软删这个工作项的通知：先按 id 升序 `FOR NO KEY UPDATE` 再更新（`DeleteIssueLinkRows` 的写法），时刻与删除者取事件的；执行者为空也接受（先例 asset）。分发与它的先后由工作项行的锁决定（§2.9）。
- **项目、工作区删除**：订阅者在原事务里为每个项目投递 `notification.delete_rows`（M6 `planning.delete_rows` 的写法，保留原请求的时刻与删除者）。任务先问 `ProjectIssuesRemain`：项目里还有未软删的工作项（`issue.delete_rows` 还在删）就 snooze 一个固定的正时长（常量 500 毫秒，不进配置；与预算用完续跑的 `JobSnooze(0)` 分开）再来；等待以 `job.CreatedAt` 计满 1 小时仍未删完，就不再 snooze 而返回错误，走 River 的退避与次数上限，次数用尽记错误日志（River v0.47.0 的 snooze 不消耗次数，没有这个上限会永远等）。没有了才分批软删这个项目的通知（每批一个事务、按 id 升序 `FOR NO KEY UPDATE`、预算用完 `JobSnooze(0)`）。这样不必让分发任务锁项目行（R3）。
- **清理**：清理步骤一张表；模块次序 asset → notification → issue → planning → project → workspace → identity；`TestPurgeCatalogCoversTablesAndIncomingKeys` 32 → 33。
- **守卫与夹具**：Go 的项目、工作区删除守卫要求每张表都有被删的行：`twoProjects`、`seedProject` 以 SQL 种通知行（夹具允许）；`parentAsyncTable`、`waitProjectDeletion`、`waitWorkspaceDeletion` 认得新表与新任务；清理的运行角色、迟到删除、规模夹具同步。e2e 的 `asynchronousDeletionTable` 加 `notifications`；删除故事的夹具经接口让第二个账户被指派（产生一行通知），轮询到行写出再删除。

### 2.9 分发任务

- 订阅者 `notify_on_issue_events`：只看 §2.3 的事件；按 §2.3 在原始变化列表上算出有没有可通知的变化（含 `mention`），没有就不投；有就在原事务 `InsertTx` 一个 `notification.fanout`。参数：事件 id、类型、时刻、工作区、项目、工作项、执行者、实体（评论 id 等），以及**保留全部位置**的变化列表，其中 `description`、`comment_html` 的 Old、New 置空（HTML 不进 `river_job`，序号不变）。
- 任务：`default` 队列，`Timeout` 30 秒，`MaxAttempts` 5，次数用尽记错误日志（`planning.delete_rows` 的先例）；事务是工作项写的那个（READ COMMITTED、读写、`lock_timeout`）。在事务里：`ShareIssue` 锁工作项行（已删除则什么都不写、完成）→ 读 issue 的列、关注者、评论正文与补齐的状态名、项目标识 → `DecideMany` → `HiddenFrom` → 按 §2.4 算出行 → 一条 `INSERT … ON CONFLICT DO NOTHING`。行的工作区、项目取锁住的行（与参数不一致时记日志并以行为准）。重试与重复投递不多写。
- 与工作项删除串行：删除先提交，任务的 `FOR SHARE` 在新版本上复核 `deleted_at IS NULL`、读不到行；任务先提交，删除事务里的订阅者软删掉刚写的行。与项目删除的先后见 §2.8。

## 3. 局部裁定

- **R1 拆段**：P1b 是生成与删除，P1c 是接口与读；设计 §12 随之改两行。
- **R2 P1b 的模块没有动作**：列入 `actionlessModules`，P1c 加 `notification.read`、`notification.write` 与规则，同时移出。
- **R3 项目删除不锁项目行**：让分发任务锁项目行会与这个项目的每一次写冲突；改为分批删除等工作项删完再删通知，正确性来自工作项行的锁（`issue.delete_rows` 软删工作项时 `FOR NO KEY UPDATE`，与分发的 `FOR SHARE` 冲突；`ShareIssue` 等锁之后复核）：工作项全删完之后不会再有分发写出这个项目的行。工作项不换项目；工作区删除按项目逐个投递，同理。
- **R4 变化序号**：订阅者在事件的原始列表上算出，参数保留全部位置；创建本身 `-1`；幂等键 `(event_id, change_seq, receiver_id)`。
- **R5 草稿发布带附件**：照事件，每个附件一条通知；登记差异。
- **R6 评论摘录**：存着的纯文本，提及不显示名字；P4 若要显示名字，读时再解析，不在发送时存。
- **R7 批量判定与 `Authorize` 同一个函数**：等价测试钉住。
- **R8 锁端口**：`IssueRowLock` 由 `IssueReader` 内嵌，`NewLocks` 的签名不变（它有 29 处调用）。
- **R9 写入基准**：M4 记录的 PATCH 数字来自 6 个 66 万行项目的删除规模测试，太重，不拿来做前后对比。另写一个轻量的选择性运行的基准（一个项目、探针工作项带若干关注者、M4 的 `scaleSamples` 20 次 PATCH 的 p50、p95、最大），只用基线已有的辅助函数，记两组：每次 PATCH 之前等队列空（`InsertTx` 的成本）与不等（含分发持行锁时的等待）；在基线 `9a3a388a` 的临时 worktree 里（复制这个测试文件）与分支上各跑一次，同一台机器。自动归档一轮另加一个选择性运行的计时（仿 `TestTheFreezeAtTheScaleTarget`），同一次运行里比较带与不带通知订阅者的发布者。
- **R10 锁探针与后台分发**：分发任务会在工作项行上取 `FOR SHARE`，bootstrap 里数等锁者的探针可能看到它。上线之前（plan T8）按固定的 grep 列出用探针的文件，探针一律按自己语句的文本认等锁者（已有先例 `-- name: LockIssueProject`），或在探针之前等队列里的 `notification.fanout` 跑完。
- **R11 关联的请求端**：追加 `requested_on` 变化而不是改用存储行的起点，使“以 `blocking` 发起”也通知发起端的关注者（与 Plane 一致）；对 P1a §2.2 的公开约定只加不改。

## 4. 验收标准

- 组合测试（经真实的写入，读 `notifications` 表）：§2.3 每一种可通知的变化各产生应有的行（字段、动词、新旧值、id、摘录、类型、时刻、`triggered_by`、序号）；执行者不收；提及与关注的去重；只改描述、迭代、模块、`sort_order`、链接标题、表情、转移、删除不产生；自动归档（执行者为空）；关联只发到请求端，含以 `blocking` 发起与经关联 id 移除；附件；草稿发布带附件。
- 复核：写入之后、任务运行之前撤权或降为只看自己，那个人不收；私有项目的移出；工作区管理员兼访客。
- 幂等：同一个任务跑两次只有一份行；参数里没有 HTML 且序号与事件一致（`issue.created` 带描述与两个负责人时负责人的序号）。
- 交错：分发与工作项删除两种先后；分发与项目删除（等工作项删完再删通知）；`issue.delete_rows` 被扣住时通知的删除任务 `metadata.snoozes` 递增、`attempt` 不超过 1、不写；锁超时让分发重试。
- 删除与清理：工作项删除的同事务软删；项目、工作区删除的分批任务与守卫；60 天清理与 `TestPurgeScale`。
- 等价：`DecideMany` 与 `Authorize` 在矩阵种子上逐格一致。
- 规模：一个工作项 1000 个关注者的一次分发，行锁持有不超过 1 秒（`write_lock_timeout` 是 5 秒；超过先报告），另记持锁期间同一工作项一次 PATCH 的等待；写入基准两组与自动归档一轮写进阶段记录。
- 每个加锁、可见、幂等性质至少一个能让测试失败的变异。本地七项、分支 CI、deep-review 与修复复审、`--no-ff` 后的 main CI；两路的时间写进阶段记录（每路增量预算 ≤ 150 秒，含删除等待；接近单次 `go test` 的 20 分钟或作业 25 分钟的八成先报告）。

## 5. 后续责任

- P1c：接口、读时的可见过滤（新的含已归档项目的可见集合）、另一端的解析、未读数、全部已读（多行的写每批按 id 升序加锁或 `SKIP LOCKED`，与 §2.8 的软删同一锁序，语句头写不变式）、权限矩阵、H2–H7 的接口证据与 e2e。
- P4：卡片按 §2.5 的字段与动词重写；`issue.updated` 的加入与移出由新旧值区分。

## 6. 风险

- 后台分发与 bootstrap 的锁探针（R10）。
- 每次写多一个 `InsertTx`（R9 量化）；自动归档一轮的事务里每个被归档的工作项多一个。
- 新表让删除守卫要求每张表都有被删的行：Go 的在 T3 转绿；e2e 的删除故事从 T2 起红，到经接口的夹具（plan T10）才转绿。
- 删除测试多出通知删除任务的等待（每次约一个 500 毫秒），计入两路的预算。
