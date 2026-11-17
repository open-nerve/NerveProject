# M7 / P1a：关注接口与通知要用的事件

> 状态：spec 待预检。基线为设计合入 main 的 noff（合入后回填）。设计见 [M7 设计](../M7-design.md) §3.1、§3.2、§6.1、§12，负责人 2026-11-17 的决定见设计 §15；加锁之路与拒绝次序沿用 M4（[硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md)）。现状调研见设计分支台账 `research/p1a-facts.md`（ignored，随阶段归档）。

## 1. 目标与边界

设计 §12 的 P1 拆为两段：

- **P1a（本 spec）**：`issue` 一侧。评论、链接的六个事件带上工作项 id；`issue.created`、`issue.updated`、评论的建立与修改带上这一次新提及的账户；手动关注、取消关注的接口；`IssueDetail.is_subscribed`。覆盖设计 §2 的 H1 的接口与数据库证据。
- **P1b（P1a 合入后再写 spec 与 plan）**：`notification` 模块与它要的端口、表、订阅者、分发任务与接口（H2–H7）。

P1a 不做：通知本身（P1b）；网页的关注按钮（P4）。网页只做三件事：重新生成的类型、手写的 `IssueDetail` 字面量补上 `is_subscribed`、`make test-web` 照跑。新的事件变化在 P1a 没有生产订阅者，由形状测试钉住，P1b 消费。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/issue/app/` | 评论、链接事件加 `issue` 变化（`collaboration_events.go`、`link_events.go`）；`issue.created`、`issue.updated` 与评论事件的 `mention` 变化（`issue_events.go`、`update_issue.go`、`comments.go`、`comment_writes.go`）；关注、取消关注的用例（新文件，§2.4）；`IssueReader` 加“调用者是否关注”的读，`seenOne` 与建立、发布草稿的回答带上它 |
| `server/internal/modules/issue/domain/` | `Issue` 加每个调用者的 `Subscribed`（同 `SubIssuesCount` 的写法，只在详情回答里有值） |
| `server/internal/modules/issue/adapter/postgres/` | `ShareIssue`（`SELECT … FROM issues WHERE id = $1 AND deleted_at IS NULL FOR SHARE`，不连别的表）、`RemoveSubscriber`（软删一行）、`IsSubscribed`（点查）；`InsertSubscribers` 复用；`DeleteIssueSubscriberRows` 等语句头注释里“写这些行都先持项目锁”一句改为写明关注、取消关注持的是工作项行的 `FOR SHARE` |
| `server/internal/modules/issue/adapter/http/` | 两个处理器；`issueDetail` 映射 `is_subscribed`；每个新声明的码一个处理器测试；两份详情 JSON 金样加字段 |
| `server/internal/modules/issue/module.go` | 用例接线 |
| `server/internal/bootstrap/` | 组合测试（§4）、权限矩阵两行、`snapshotSearchStore` 等 `IssueReader` 的测试实现 |
| `server/internal/platform/httpserver/apitest/issue_shapes_test.go` | `TestIssueDetailShape` 改为 `IssueDetail` 恰比 `Issue` 多 `description_html`、`is_subscribed` 两个必填字段 |
| `api/modules/issue.yaml`、`api/openapi.yaml` | `POST /api/v0/issues/{issue_id}/subscribe`、`/unsubscribe`（`subscribeIssue`、`unsubscribeIssue`，204，无请求体，`x-problem-codes: [issue.not_found, forbidden, server_busy]`）；`IssueDetail.is_subscribed`（必填布尔，说明“调用者此刻是否关注这个工作项”）；两条路径在根文件的 `$ref` |
| `web/` | `api-client` 重新生成；`core/store/issue/fake-issues.ts` 的 `issueDetailOf` 与 `core/store/issue/issue.store.types.test.ts` 两处字面量加 `is_subscribed: false` |
| `e2e/stories/collaboration/` | H1 (API) 故事（新目录，§4） |
| `tools/keywords.json` | `phase` 为 `M7/P1`（P1a、P1b 共用，关键词工具只认 `M<n>/P<k>`） |
| `docs/` | `plane-diff.md` 的关注两行（§3 R3、R4）；M4 交接 M7-02 与 M5 交接 M5-M7-02 的“处理结果（M7/P1a）”；设计 §14 的 P1a 行；阶段记录 |

### 2.2 事件的变化（issue 的公开约定）

- **`issue` 变化**：评论的建立、修改与链接的建立、修改，在已有变化之后追加 `{Field: "issue", NewID: 工作项 id}`；评论的删除与链接的删除追加 `{Field: "issue", OldID: 工作项 id}`（删除类事件的变化都在 Old 一侧，先例：表情、关联的移除与 `asset.attachment_deleted`）。`Entity` 仍是评论或链接。
- **`mention` 变化**：每个这一次新被提及的账户一个 `{Field: "mention", NewID: 账户}`，按账户 id 升序，追加在业务变化之后、`issue` 变化之前（`issue.created`、`issue.updated` 没有 `issue` 变化，追加在最后，`sort_order` 之后）。集合的定义：
  - 描述（建立、发布草稿、PATCH）：M4 提及同步已算出的集合——建立时为 `checked.mentions`，PATCH 时为 `mentions.Added`；都是项目的有效成员（任何角色），未经可见性过滤（只看自己的访客也在内）。PATCH 的“写入但未改变”分支（`update_issue.go:286-297`）照旧不发布事件，也就没有 `mention` 变化（M4 交接 M7-02：无改动与 ID 迁移的保存不发布新事件）。
  - 评论的建立：正文的全部提及，经同一个成员过滤（`descriptionTargets` 的 `mentions`，不加执行者）。
  - 评论的修改：新正文的提及减去旧正文的提及，再经成员过滤；旧正文是项目锁下重读的那一份（`comment_writes.go:77`），重新清洗得出它的提及。正文清洗后不变时照旧不写、不发布。
- 不加事件类型；不改已有变化的内容与次序；`asset`、`crawl_on_link_change` 两个生产订阅者不看新变化（`crawl_on_link_change` 只认 `url`）。

### 2.3 接口与错误

- **关注** `POST /issues/{id}/subscribe`、**取消关注** `POST /issues/{id}/unsubscribe`：204，无回答体。已关注再关注、未关注就取消，都是 204 的空操作。
- **判定**：`issue.read`（管理员、成员、访客）；看不到的工作项（不存在、已删除、项目看不到、只看自己的访客看别人的）404 `issue.not_found`；看得到项目但角色不在规则里 403 `forbidden`；锁等待超过 `issue.write_lock_timeout` 503 `server_busy`。已归档的工作项与项目照常可关注、取消（R3）。
- **回答的 `is_subscribed`**：取、按编号取、修改、归档、恢复的详情回答为调用者此刻是否有未删除的关注行；建立与发布草稿的回答在同一事务里读，建立者恒为关注（M4 的自动关注）。列表行（`Issue`）不带它。

### 2.4 关注的写法

照 `DraftLocks.run` 的“不进写闸门”写法（`lock_draft.go`）：

1. 不加锁的预检：`Locks.see(issue.read)`，判定拒绝的在这里答 404、403。
2. 事务（与工作项写同一个 `busyTx`，锁等待超时 503）：`ShareIssue` 以 `FOR SHARE` 锁住工作项行；没有（已删除）答 404；在锁住的行上再判定一次（`decide`，含 `HiddenFrom`）。
3. 关注：`InsertSubscribers`（`ON CONFLICT … DO NOTHING`）；取消：`RemoveSubscriber` 软删这一行（`deleted_at`、`updated_at`、`updated_by_id` 为这次的时刻与执行者）。
4. 不写动态、不 `MarkChanged`、不改工作项的 `updated_at`、不发布事件（Plane 也不）。

`FOR SHARE` 与工作项的删除（`UPDATE issues` 取 `FOR NO KEY UPDATE`）冲突：删除先提交，关注在新版本上找不到行、答 404；关注先提交，删除事务里软删关注行的语句在之后执行、看得到它。项目、工作区的删除由分批任务先软删工作项、后软删关注行，同理。锁序与 v0 的总序一致（工作项先于关注行）。

## 3. 局部裁定

- **R1 删除类事件的 `issue` 在 Old 一侧**：与已有的移除类事件一致，消费方按事件类型取 `OldID` 或 `NewID`（设计 §6.1 写的 `NewID` 按此细化）。
- **R2 提及集合经成员过滤**：评论与描述同一个定义，去掉不是项目有效成员的 id；可见性留给通知发送时（设计 §3.3）。
- **R3 已归档的照常可关注**：关注是个人的状态，Plane 只看项目成员关系；先例是 `cycle.preferences` 在归档时照常可写。登记差异：Plane 不核对看得到（访客可关注看不到的工作项），Nerve 按 `issue.read` 与只看自己。
- **R4 不另设动作**：关注用 `issue.read`（设计 §3.10），`actions_test` 不变；权限矩阵加两行。
- **R5 `is_subscribed` 只在详情**：读在 `seenOne`（取、按编号取、修改、归档、恢复都经它）与建立、发布草稿的回读处，一次点查；不加到 `GetIssue` 的 SQL（它也是锁下重读用的、没有调用者）。
- **R6 不发布关注事件**：设计没有关注的通知；P1b 的通知按关注行读。

## 4. 验收标准

- 六个事件与 `issue.created`、`issue.updated` 的形状测试（单元与 bootstrap）按 §2.2 改，并新增：描述带提及的建立与 PATCH（新增、已有、访客、非成员、“写入但未改变”无事件）、评论带提及的建立与修改（差集、旧正文重读、非成员被滤掉）；清洗对已清洗的 HTML 提及不变的单元测试。
- 关注：关注、再关注、取消、再取消；`is_subscribed` 在六种详情回答里；列表的 `subscriber_id` 与成员列表的 `relation=subscribed` 随之变化；取消之后评论、被指派、被提及会重新关注；看不到的 404（不存在、已删除、看不到的项目、只看自己的访客看别人的）、403（工作区管理员、成员看未加入的公开项目）；已归档的工作项与项目照常；数据库的行（软删的时刻与执行者）。
- 交错：关注与工作项删除两种先后（`FOR SHARE` 的探针测试，同 M4 的锁探针写法）；关注与项目删除的分批任务；锁等待超时 503。每个加锁、可见的性质至少一个能让测试失败的变异。
- 权限矩阵两行（工作项的列、归档的列、已归档的工作项）。
- e2e：`e2e/stories/collaboration/h1-subscription.spec.ts`，“H1 (API)”：PAT 与浏览器登录对等地关注、取消，经接口读 `is_subscribed` 与 `subscriber_id` 的列表，数据库断言关注行。本机时间写进阶段记录（≤ 3 秒）。
- 本地七项、分支 CI、deep-review 与修复复审、`--no-ff` 后的 main CI；Go 的第 0 路的预估写进阶段记录（设计 §9：接近 960 秒时先报告负责人）。

## 5. 后续责任

- P1b 消费 §2.2 的变化，复用 `ShareIssue` 给分发任务锁工作项行（设计 §6.2）。
- P4 在页面上用 `is_subscribed` 与两个接口；网页的 store 合并列表行时只保留 `description_html`（`issue.store.ts:27-36`），P4 要让它也保留 `is_subscribed`。

## 6. 风险

- 加入 `bootstrap` 的测试会改变排在其后的测试所在的路（按名字排序轮流分路）：最终名单定下后重算两路的预估。
- `TestAHiddenParentIsNotInTheAnswer` 比较两个调用者的详情：两人的 `is_subscribed` 本来不同，比较时去掉这个字段（只此一处的改动，断言的本意不变）。
