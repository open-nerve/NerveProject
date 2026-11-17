# M7 / P1a：关注接口与通知要用的事件

> 状态：spec 经预检 [re-82](../../../reviews/re-82-M7P1a预检.md) 修正后通过。基线为设计合入 main 的 noff（合入后回填）。设计见 [M7 设计](../M7-design.md) §3.1、§3.2、§6.1、§12，负责人 2026-11-17 的决定见设计 §15；加锁之路与拒绝次序沿用 M4（[硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md)），关注是其中一个写明理由的例外（§2.4）。现状调研见设计分支台账 `research/p1a-facts.md`（ignored，随阶段归档）。

## 1. 目标与边界

设计 §12 的 P1 拆为两段：

- **P1a（本 spec）**：`issue` 一侧。评论、链接的六个事件带上工作项 id；`issue.created`、`issue.updated`、评论的建立与修改带上这一次新提及的账户；手动关注、取消关注的接口；`IssueDetail.is_subscribed`；描述与评论的保存只让新被提及的人自动关注（R7）。覆盖设计 §2 的 H1 的接口与数据库证据。
- **P1b（P1a 合入后再写 spec 与 plan）**：`notification` 模块与它要的端口、表、订阅者、分发任务与接口（H2–H7）。

P1a 不做：通知本身（P1b）；网页的关注按钮（P4）。网页只做四件事：重新生成的类型、手写的 `IssueDetail` 字面量补上 `is_subscribed`、工作项 store 在列表行覆盖详情时保留 `is_subscribed`（同 `description_html` 的写法）、`make test-web` 照跑。新的事件变化在 P1a 没有生产订阅者，由形状测试钉住，P1b 消费。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/issue/app/` | 评论、链接事件加 `issue` 变化（`collaboration_events.go`、`link_events.go`）；`issue.created`、`issue.updated` 与评论事件的 `mention` 变化（`issue_events.go`、`update_issue.go`、`comments.go`、`comment_writes.go`）；自动关注收窄为新被提及的人（`description.go`、`mentions.go`，R7）；`Locks.share`（新文件 `lock_share.go`，§2.4）与关注、取消关注的用例（只写行）；`IssueReader` 加“调用者是否关注”的读，`seenOne` 与建立、发布草稿的回读带上它；`lock.go` 里 `Locks` 的说明加上 `share` |
| `server/internal/modules/issue/domain/` | `Issue` 加每个调用者的 `Subscribed`（同 `SubIssuesCount` 的写法，只在详情回答里有值） |
| `server/internal/modules/issue/adapter/postgres/` | `ShareIssue`（`SELECT … FROM issues WHERE id = $1 AND deleted_at IS NULL FOR SHARE`，不连别的表）、`RemoveSubscriber`（软删一行）、`IsSubscribed`（点查）；`InsertSubscribers` 复用；`DeleteIssueSubscriberRows` 等语句头注释里“写这些行都先持项目锁”一句改为写明关注、取消关注持的是工作项行的 `FOR SHARE`，并写下 §2.4 的锁序不变式 |
| `server/internal/modules/issue/adapter/http/` | 两个处理器；`issueDetail` 映射 `is_subscribed`；每个新声明的码一个处理器测试；两份详情 JSON 金样加字段 |
| `server/internal/modules/issue/module.go` | 用例接线 |
| `server/internal/bootstrap/` | 组合测试（§4）；权限矩阵两行；`IssueReader` 的测试实现 `snapshotSearchStore`；写的锁目录加第三类 `subscriptionWrites`（`issue_draft_write_locks_test.go`、`issue_write_locks_test.go`，形状“无工作区锁、无项目锁、工作项行 `FOR SHARE`、不进闸门、被拒的调用者不取锁”，以真实探针核对）；删除窗口加两例（`issue_deletion_window_cases_test.go`，22 → 24）；`gate_test.go` 说明“每个工作项的写都进闸门”一句写明关注、取消关注与草稿同为例外 |
| `server/internal/platform/httpserver/apitest/issue_shapes_test.go` | `TestIssueDetailShape` 改为 `IssueDetail` 恰比 `Issue` 多 `description_html`、`is_subscribed` 两个必填字段 |
| `api/modules/issue.yaml`、`api/openapi.yaml`、`api/dist/openapi.yaml` | `POST /api/v0/issues/{issue_id}/subscribe`、`/unsubscribe`（`subscribeIssue`、`unsubscribeIssue`，204，无请求体，`x-problem-codes: [issue.not_found, forbidden, server_busy]`）；`IssueDetail.is_subscribed`（必填布尔，说明“调用者此刻是否关注这个工作项”）；根文件的 `$ref`；打包产物 |
| `web/` | `api-client` 重新生成（`schema.gen.ts`）；三处手写字面量加 `is_subscribed: false`（`core/store/issue/fake-issues.ts:47` 的 `issueDetailOf`，`core/store/issue/issue.store.types.test.ts:23`、`:43`，其余以 `check:types` 为准）；`core/store/issue/issue.store.ts` 的 `choose` 保留 `is_subscribed` 与它的类型测试 |
| `e2e/stories/collaboration/` | H1 (API) 故事（新目录，§4） |
| `tools/keywords.json` | `phase` 为 `M7/P1`（P1a、P1b 共用，关键词工具只认 `M<n>/P<k>`） |
| `docs/` | `plane-diff.md` 的关注两行与“自动关注”“工作项的写的取锁”两行的补充（R3、R7、§2.4）；[硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md) 加关注的例外一段（P8 为草稿写过同样的一段）；M4 设计的事件表（`M4-design.md` 的评论、链接两行）指向本 spec §2.2；M4 交接 M7-02 与 M5 交接 M5-M7-02 的“处理结果（M7/P1a）”；设计 §14 的 P1a 行；阶段记录 |

### 2.2 事件的变化（issue 的公开约定）

- **`issue` 变化**：评论的建立、修改与链接的建立、修改，在已有变化之后追加 `{Field: "issue", NewID: 工作项 id}`；评论的删除与链接的删除追加 `{Field: "issue", OldID: 工作项 id}`（删除类事件的变化都在 Old 一侧，先例：表情、关联的移除与 `asset.attachment_deleted`）。`Entity` 仍是评论或链接。
- **`mention` 变化**：每个这一次新被提及的账户一个 `{Field: "mention", NewID: 账户}`，按账户 id 升序，追加在业务变化之后、`issue` 变化之前（`issue.created`、`issue.updated` 没有 `issue` 变化，追加在最后，`sort_order` 之后）。评论的删除不带。集合的定义：
  - 描述（建立、发布草稿、PATCH）：M4 提及同步已算出的集合——建立时为 `checked.mentions`，PATCH 时为 `mentions.Added`（成员过滤后的提及减去现存的 `issue_mentions` 行）；都是项目的有效成员（任何角色），未经可见性过滤（只看自己的访客也在内）。所以一个在加入项目之前就被提及的人，下一次改描述时算新提及。PATCH 的“写入但未改变”分支（`update_issue.go:286-297`）照旧不发布事件，也就没有 `mention` 变化，那时补写的提及行不会带来通知（M4 交接 M7-02：无改动与 ID 迁移的保存不发布新事件）。
  - 评论的建立：正文的全部提及，经同一个成员过滤（`descriptionTargets` 的 `mentions`，不加执行者）。
  - 评论的修改：新正文的提及减去旧正文的提及，再经成员过滤；旧正文是项目锁下重读的那一份（`comment_writes.go:77`），重新清洗得出它的提及。评论不存提及，所以加入项目之前被提及的人在之后的修改里不算新提及。正文清洗后不变时照旧不写、不发布。
- 不加事件类型；不改已有变化的内容与次序；`asset`、`crawl_on_link_change` 两个生产订阅者不看新变化（`crawl_on_link_change` 只认 `url`）。

### 2.3 接口与错误

- **关注** `POST /issues/{id}/subscribe`、**取消关注** `POST /issues/{id}/unsubscribe`：204，无回答体。已关注再关注、未关注就取消，都是 204 的空操作。
- **判定**：`issue.read`（管理员、成员、访客）；看不到的工作项（不存在、已删除、项目看不到、只看自己的访客看别人的，以及项目或工作区正在删除）404 `issue.not_found`；看得到项目但角色不在规则里 403 `forbidden`；锁等待超过 `issue.write_lock_timeout` 503 `server_busy`。已归档的工作项与项目照常可关注、取消（R3）。
- **回答的 `is_subscribed`**：七个回答 `IssueDetail` 的操作都带——取、按编号取、修改、归档、恢复（经 `seenOne`，为调用者此刻是否有未删除的关注行）与建立、发布草稿（在同一事务里回读，建立者恒为关注：M4 的自动关注）。列表行（`Issue`）不带它。

### 2.4 关注的写法与它的例外

编排放在 `Locks` 上（新的 `Locks.share(ctx, actor, issueID, action, fn)`，与 `Locks.run`、`DraftLocks.run` 并列，硬规则 §4“新的写使用相应的集中编排”），用例只写行：

1. 不加锁的预检：`see(issue.read)`，判定拒绝的在这里答 404、403，不取任何锁。
2. 事务（与工作项写同一个 `busyTx`，锁等待超时 503）：`ShareIssue` 以 `FOR SHARE` 锁住工作项行；没有（已删除）答 404；在锁住的行上再判定一次（`decide`，含 `HiddenFrom`）。
3. 关注：`InsertSubscribers`（`ON CONFLICT … DO NOTHING`）；取消：`RemoveSubscriber` 软删这一行（`deleted_at`、`updated_at`、`updated_by_id` 为这次的时刻与执行者）。
4. 不写动态、不 `MarkChanged`、不改工作项的 `updated_at`、不发布事件（Plane 也不）。

**它偏离的两条约定与为什么安全**：总体设计的约定 2（写先锁父行——项目——再判定）与硬规则 §4（普通工作项的写都经 `Locks.run`：写闸门加项目的 `FOR NO KEY UPDATE`）。关注不取项目锁、不进写闸门（设计 §3.1，re-81 m9），所以它的锁下判定不与成员关系的变更、项目删除串行。这是安全的：一个刚被撤权的人留下的关注行，与“先被自动关注、后被移出项目”留下的行是同一种状态（M4 交接 M7-02），P1b 的通知在发送时按读权限复核，列表的 `subscriber_id` 按当时的成员关系过滤；而取项目锁会让每一次关注排在这个项目所有的写之后。工作项的删除仍与它串行：`FOR SHARE` 与删除的 `UPDATE issues`（`FOR NO KEY UPDATE`）冲突——删除先提交，关注在新版本上找不到行、答 404；关注先提交，删除事务里软删关注行的语句在之后执行、看得到它。项目、工作区的删除由分批任务先软删工作项、后软删关注行，同理；它们待处理期间的关注答 404（删除窗口的两例）。

**锁序不变式**：每个写 `issue_subscribers` 的路径都先锁工作项行（评论的 `MarkChanged`、PATCH 的 `UpdateIssue`/`WriteDescription`、删除的 `DeleteIssueRows`，建立是新行），关注先 `FOR SHARE` 工作项行、再写关注行，所以没有锁环。以后加写关注行的路径（P2 的收集箱）必须同样先锁工作项行；写进语句头注释与 §5。

## 3. 局部裁定

- **R1 删除类事件的 `issue` 在 Old 一侧**：与已有的移除类事件一致，消费方按事件类型取 `OldID` 或 `NewID`（设计 §6.1 已随之细化）。
- **R2 提及集合经成员过滤**：评论与描述同一个过滤，去掉不是项目有效成员的 id；可见性留给通知发送时（设计 §3.3）。两者“新”的口径不同（§2.2），照现状接受。
- **R3 已归档的照常可关注**：关注是个人的状态，Plane 只看项目成员关系；先例是 `cycle.preferences` 在归档时照常可写。登记差异：Plane 不核对看得到（访客可关注看不到的工作项），Nerve 按 `issue.read` 与只看自己。
- **R4 不另设动作**：关注用 `issue.read`（设计 §3.10），`actions_test` 不变；权限矩阵加两行。
- **R5 `is_subscribed` 只在详情**：读在 `seenOne` 与建立、发布草稿的回读处，一次点查；不加到 `GetIssue` 的 SQL（它也是锁下重读用的、没有调用者）。
- **R6 不发布关注事件**：设计没有关注的通知；P1b 的通知按关注行读。
- **R7 自动关注只给新被提及的人**：M4 在每次写描述时把描述里全部可见的被提及者插入关注（`description.go:41-49`、`update_issue.go:347-351`，含“写入但未改变”的分支），评论修改时把新正文里全部可见的被提及者插入（`comment_writes.go:73`）；没有取消关注时这是空操作，有了取消关注，任何一次无关的保存都会把取消掉的人加回来。改为：描述的写只关注 `可见 ∩ mentions.Added`，评论的修改只关注 `可见 ∩ (新 − 旧)` 加执行者；建立（工作项、草稿发布、评论）不变。设计 §3.1 的“被提及会重新加入”按此读作“被新提及”。登记差异（plane-diff“自动关注”行）。

## 4. 验收标准

- 六个事件与 `issue.created`、`issue.updated` 的形状测试（单元与 bootstrap）按 §2.2 改，并新增：描述带提及的建立与 PATCH（新增、已有、访客、非成员、“写入但未改变”无事件）、评论带提及的建立与修改（差集、旧正文重读、非成员被滤掉）；清洗对已清洗的 HTML 提及不变的单元测试。
- 关注：关注、再关注、取消、再取消；`is_subscribed` 在七种详情回答里；列表的 `subscriber_id` 与成员列表的 `relation=subscribed` 随之变化；取消之后评论、被指派、被**新**提及会重新关注，而仍被提及的人在描述保存、评论修改之后不会（R7 的成对断言）；看不到的 404（不存在、已删除、看不到的项目、只看自己的访客看别人的）、403（工作区管理员、成员看未加入的公开项目）；已归档的工作项与项目照常；数据库的行（软删的时刻与执行者）。
- 交错：关注与工作项删除两种先后（`subscriptionWrites` 的锁探针）；关注与项目删除的分批任务、删除窗口的两例；扣住工作项行时撤权，放开后 404 且没有行；锁等待超时 503。每个加锁、可见的性质至少一个能让测试失败的变异（去掉锁下再判定的变异由“扣住时撤权”杀掉）。
- 权限矩阵两行（工作项的列、归档的列、已归档的工作项）。
- e2e：`e2e/stories/collaboration/h1-subscription.spec.ts`，“H1 (API)”：PAT 与浏览器登录对等地关注、取消，经接口读 `is_subscribed` 与 `subscriber_id` 的列表，数据库断言关注行。本机时间写进阶段记录（≤ 3 秒）。
- 本地七项、分支 CI、deep-review 与修复复审、`--no-ff` 后的 main CI；Go 的第 0 路的预估写进阶段记录（设计 §9：接近 960 秒时先报告负责人）。

## 5. 后续责任

- P1b 消费 §2.2 的变化，复用 `ShareIssue` 给分发任务锁工作项行（设计 §6.2）；给通知的变化编号时跳过 `mention` 与 `issue` 两种字段（它们不是可通知的变化本身）。
- P2a 对 Triage 条目的关注作出裁定（设计 §3.6 的读照常，提交者自动关注，推定可以关注）并加矩阵行；P2 新的写关注行的路径守 §2.4 的锁序不变式。
- P4 在页面上用 `is_subscribed` 与两个接口。

## 6. 风险

- 加入 `bootstrap` 的测试会改变排在其后的测试所在的路（按名字排序轮流分路）：最终名单定下后重算两路的预估。
- `TestAHiddenParentIsNotInTheAnswer` 比较两个调用者的详情：两人的 `is_subscribed` 本来不同，比较时去掉这个字段（只此一处的改动，断言的本意不变）。
- R7 改了 M4 的自动关注：M4 的评论、描述测试里断言“保存后关注行存在”的，若依赖重复插入，随之改为断言新提及（只改这些，计划 T3、T4 点名）。
