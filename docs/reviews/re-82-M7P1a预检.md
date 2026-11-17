# M7 / P1a 预检

> 2026-11-17：预检通过（修正后）：Ready after fixes，没有 Critical，四处 Important、八处 Minor 都只涉及 spec 与 plan 的文字、Task 的边界与验收，全部接受并改入；没有须负责人决定的事（R7 是 M4 自动关注的局部收窄，按负责人的授权自行决定并登记差异）。
> - **目标**：[P1a spec](../v0/M7-collaboration/specs/P1a-subscriptions-events.md)、[P1a plan](../v0/M7-collaboration/plans/P1a-subscriptions-events.md)，依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §3.1、§3.2、§6.1、§12 与设计分支台账的 `research/p1a-facts.md`。
> - **基线**：main `d5b5134e03436781d2efd6cb9a483f66e3f95a8f`；审查对象为分支 `codex/m7-design` 的 `93daa77d`。

## 1. 方法

一次 fresh 只读审查（`fable`），不改动文件与 git 状态、不跑工具；对照代码核了 spec 与 plan 的每个路径、函数、行号、测试名与查询名，审了事件变化的位置与次序、提及集合的定义、关注的事务与锁序、`is_subscribed` 的每条回答之路、Task 的大小与次序、检查点与守卫。原始报告没有另存文件，逐条内容与处理见 §2。

## 2. 结论与处理

结论 **Ready after fixes**。

| # | 级别 | 发现 | 处理 |
|---|---|---|---|
| I1 | Important | 两个由契约推导的目录守卫在 T7 加了两个操作后转红而没有 Task 负责：写的锁目录要求每个 `issue` 标签的非 GET 操作属于已有两类之一，而普通一类要求项目的 `FOR NO KEY UPDATE`；删除窗口钉住 22 例、每个操作一例 | 接受：新 T8 加第三类 `subscriptionWrites`（无工作区锁、无项目锁、工作项行 `FOR SHARE`、不进闸门、被拒者不取锁，真实探针核对）与删除窗口的两例（22 → 24）；这些测试列进“允许改的既有测试”；plan 写明守卫转绿的时点 |
| I2 | Important | 任何一次描述的保存都会把取消了关注的人加回来：M4 在每次写描述时把全部可见的被提及者插入关注，评论修改同理；没有取消关注时这是空操作 | 接受：裁定 R7——描述的写只关注 `可见 ∩ mentions.Added`，评论的修改只关注 `可见 ∩ (新 − 旧)` 加执行者，建立不变；T3、T4 加成对的断言（仍被提及的不加回、新被提及的加入）；登记差异；设计 §3.1 随之改为“被新提及” |
| I3 | Important | T8 使 `is_subscribed` 成为必填，网页的生成与字面量却在 T9，T8 之后的检查点必然失败；T7 的文件没列生成物 | 接受：网页的生成、字面量与 store 并入同一个 Task（新 T9），检查点移到它之后；T7、T9 列出 `api/dist/openapi.yaml`、`schema.gen.ts` 与 `make gen-go`、`make gen-web` |
| I4 | Important | spec 说关注“照 `DraftLocks.run`”，却没写明它偏离了总体设计的约定 2（写先锁父行）与硬规则 §4（普通工作项的写都经 `Locks.run`、新的写用集中编排）；T6 把编排写进了用例文件 | 接受：spec §2.4 写明两处偏离与为什么安全（撤权留下的行与“自动关注后被移出”同一状态，通知发送时复核，列表按成员关系过滤；取项目锁会让关注排在项目所有的写之后）；编排放在 `Locks.share`（`lock_share.go`），用例只写行；T13 在硬规则 §4 加一段，`lock.go` 的说明、`gate_test.go` 的说明与 plane-diff“工作项的写的取锁”行随之补 |
| m1 | Minor | 回答 `IssueDetail` 的操作是七个不是六个 | 接受：写为七个并逐一列名 |
| m2 | Minor | 网页的手写字面量是两个文件里的三处 | 接受：写为三处，其余以 `check:types` 为准 |
| m3 | Minor | 不会死锁依赖一个没写下的不变式：每个写关注行的路径都先锁工作项行 | 接受：写进 spec §2.4、语句头注释与 §5（交给 P2） |
| m4 | Minor | 描述与评论“新提及”的口径不同，“写入但未改变”分支补写的提及行不会带来通知 | 接受：§2.2 各写一句，照现状接受 |
| m5 | Minor | P1a 到 P4 之间，网页 store 在列表行覆盖详情时会丢掉 `is_subscribed` | 接受：T9 让 `choose` 保留它（一行与已有的类型测试） |
| m6 | Minor | T13 的文档同步缺 M4 设计的事件表；M5-M7-02 说的是附件事件，P1a 不碰 | 接受：T13 加 M4 事件表的指向；M5-M7-02 写为“附件事件本已只带 id，P1a 不改，消费在 P1b” |
| m7 | Minor | “去掉锁下再判定”的变异只有在预检与事务之间发生变化时才杀得掉 | 接受：T10 加“扣住工作项行时撤权，放开后 404 且没有行”；T6 的假授权在两次调用之间翻转 |
| m8 | Minor | §5 缺 Triage 与 P1b 的后续责任 | 接受：P2a 裁定 Triage 条目的关注并加矩阵行，守锁序不变式；P1b 给通知编号时跳过 `mention`、`issue` 两种字段 |

审者对询问的几点的回答（无需改动）：评论的删除不带提及；P1a 不必拒绝 Triage；13 个 Task 大小合适，次序除 I3 外正确；建立与发布草稿回读 `is_subscribed` 而不是写死为真是对的。

## 3. 审者核实为正确的事实（摘要）

- 事件：`commentEvent`、`linkEvent`、表情与关联移除的 Old 一侧先例（`collaboration_events.go:11-29`、`issue_events.go:80-96`）；`createdChanges` 收到带 `mentions` 的 `checked`（`create_core.go:105-109,276`），草稿发布共用 `prepareLocked`/`writePreparedAt`；`updatedChanges` 在 `update_issue.go:368` 被调用时 `relations` 在作用域内，另一个调用者 `sub_issues.go:153` 只有父工作项。
- “写入但未改变”分支（`update_issue.go:286-297`）回答空事件；评论修改的旧正文是锁下重读的（`comment_writes.go:77`）；正文不变时不写、不发布。
- 六个事件的生产订阅者不受影响（`crawl_on_link_change.go:24` 只认 `url`）；`descriptionTargets` 回答成员过滤后、含只看自己的访客、升序的集合。
- `DraftLocks.run` 不进闸门、用 `l.base.tx`（`lock_draft.go:12-13,131`），它是 `busyTx`（`app.go:173`），锁超时映射为 `server_busy`（`gate.go:62-78`）；`decide` 执行 `HiddenFrom`。
- 删除：工作项删除先 `DeleteIssueRows` 后 `DeleteIssueSubscriberRows`；分批任务按 `project_id` 先 `issues` 后 `issue_subscribers`；与现有写者没有锁环。
- `InsertSubscribers` 带 `ON CONFLICT … DO NOTHING`；还没有单行软删的语句；`seenOne` 恰覆盖取、按编号取、修改、归档、恢复，建立与发布草稿绕过它；`IssueReader` 恰三个实现。
- 两个会被打破的测试确如所说，没有找到别的跨调用者整份详情的比较；契约、访问规则、`cycle.preferences` 与 `server_busy` 的先例、权限矩阵的辅助函数、关键词的 phase 规则、e2e 的标题约定与目录都存在。
