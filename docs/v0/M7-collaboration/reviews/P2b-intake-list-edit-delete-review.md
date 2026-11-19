# M7 / P2b 阶段记录：收集箱的列表、内容修改、删除与待处理数

> 状态：实施完成，待整分支审查。spec 与 plan 见 [P2b spec](../specs/P2b-intake-list-edit-delete.md)、[P2b plan](../plans/P2b-intake-list-edit-delete.md)，预检 [re-86](../../../reviews/re-86-M7P2b预检.md)（修正后通过，6 处重要、11 处次要全部采纳）。基线 main `057176e5`（P2a 的 noff `670399c4` 加 M5 测试竞态的修复）。

## 1. 交付

- 四个操作：`GET /projects/{project_id}/intake-issues`（`listIntakeIssues`）、`PATCH`、`DELETE /intake-issues/{issue_id}`（`updateIntakeIssue`、`deleteIntakeIssue`）、`GET /workspaces/{slug}/intake-counts`（`getIntakeCounts`）；动作 `intake.read`、`intake.update`、`intake.delete`、`intake.count`；新码 `issue.intake_closed`（409）。
- 列表：有效状态（到期的暂缓算待处理）、七种筛选、三种次序、键集游标与 `total`；行与项目列表同样裁剪，看不到的重复目标为 `null`；只看自己的访客只列自己创建的。
- 内容修改：不是管理员的须是提交者（工作项的创建者）；访客提交者只能改名字与描述；关闭的条目 409；经修改核心 `IssueUpdate`，动态、提及、描述版本、事件与普通修改相同。
- 删除：管理员或成员以上的提交者、任何状态；未接受的连工作项一起删（经删除核心 `IssueRemoval`），已接受的只删条目行。
- 待处理数：调用者看得到、开着收集箱的每个项目一项（含 0）。
- P2a 留下的 m2（新建核心交回读回的工作项）、m3（修改核心成自己的类型）、n1（接受时工作项已在默认状态是内部错误）。
- 按目录驱动的守卫：权限矩阵 10 行 89 格、写锁表两行、删除窗口读写各加两个、空创建者测试；e2e 的 H9 接口故事。

## 2. Task 与提交

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `f1834445` | spec、plan、预检 re-86；设计 §3.5、§3.10、§12（P2b、P2c）、§14；P2a 阶段记录 §9 |
| T2 | `0d2a4db0` | 修改核心抽成 `IssueUpdate`（纯搬移，m3） |
| T3 | `6f6e6305` | m2、n1 |
| T4 | `ed14833e` | 领域：列表的查询与游标、修改的请求体、关闭的状态、四个动作与规则 |
| T5a | `518935d4` | 存储：列表 |
| T5b | `47b86604` | 存储：计数、条目行的软删、project 的“开着收集箱”端口 |
| T6 | `2d8f75f0` | 列表与计数的用例 |
| T7 | `70414b0f`、`ab3b6d35` | 删除核心抽成 `IssueRemoval`（纯搬移）；收集箱的删除 |
| T8 | `dbce5620` | 内容修改 |
| T9 | `1a909a1f` | 契约、处理器、网页的码文案与客户端的 CSV |
| T10 | `bcaef0b5` | 权限矩阵、写锁表、删除窗口、空创建者 |
| T11a | `a502f72d` | 组合测试：列表与计数 |
| T11b | `3ade4bc0` | 组合测试：修改、删除、第八个详情回答 |
| T12 | `3b4c1e8e` | 交错 |
| T13 | （待填） | e2e 的 H9 故事 |
| T14 | （待填） | 文档与交接 |

## 3. 局部裁定（实施中）

- T3：n1 的三格是新的测试函数（m1 的表没有工作项状态与条目状态的列）。
- T4：`created_at`、`updated_at` 各收一个条件，重复给 422；`ContentOnly` 看“给了哪些字段”，不看值变没变（给了当前值的 `priority` 也算）。
- T5a/T6：SQL 交回存的条目行，回答里的有效状态由用例按同一时刻 `AsOf` 算；越过末尾的空页照 `Listing` 重读 `total`。
- T5b：project 的端口也排除已删除、已归档的项目（`Visible` 的未归档条件在这里重复一次，这一层的变异因此能被测出）；用例仍从 `Visible` 出发。
- T6：把 `ListIssues` 里“找项目 → 判定”两步原样提成 `Locks.seeProject`（调用与次序不变），列表与收集箱列表共用；重复目标经已有的 `IssueRefs`（与详情同一套条件，11 格逐值比对）。
- T7：删除核心命名 `IssueRemoval`（`IssueDeletion` 已是行的戳记）；未接受的删除路径里项目归档检查两次（`refuseIntakeOff` 与核心各一次），为保持纯搬移不动。
- T8：“管理员”即 `Grant.ProjectAdmin`，与 `intake.triage` 同一判定（项目管理员，或是项目有效成员的工作区管理员，§15 B），所以以访客身份加入的工作区管理员也能改自己条目的全部字段；“访客”是非管理员且项目角色为访客，不看 `OwnOnly`；`Issue.CreatedBy` 放在领域，`HiddenFrom` 也用它。
- T9：`IntakeCount` 是具名的 schema（线上形状不变）；`IntakeIssue` 逐个列出 `Issue` 的属性（`Issue` 是 `additionalProperties: false`，`allOf` 不成），形状测试钉住两者一致。**偏离 plan**：网页客户端 `@nerve/api-client` 按路由决定数组参数的发法，不改则 `status` 发成重复参数、服务端答 422；把路由清单改为 `csvListPaths` 并加 `status`，已有两条路由的行为不变（`client.test.mjs` 按契约钉住）。
- T10：删除窗口测试**不加** `waitForStartupRuns`：它的屏障只是草稿行的 `FOR UPDATE`，启动任务排不到它后面，P2a 的连接池饥饿机制不成立（P2a 的 `a4bc78d5` 对同一测试也选了 `barriersOutlasted`）；加上要多约 4 秒。
- T11：日的边界与次序的区分用测试时钟提交，不改 SQL 的 `created_at`；“与普通修改相同”以一个经 `updateIssue` 同样修改的孪生工作项比对，不重述修改核心的规则；`issue.deleted` 以通知订阅者的效果观察。
- T12：计数与提交以确定的次序测（提交持锁未提交时读计数），只有这一个方向（计数的只读事务没有能停下的点）；“锁等待时开关被关”不重复（T7、T8 的单测与 P2a 的 `TestTheIntakeToggleAndItsWritesSerialize` 已钉住）。

## 4. 向负责人说明

- **R1 再划分**：资产判定跨模块（资产的契约、项目的端口与 SQL、资产读路径的判定次序），照原范围约 20 个 Task，划为 P2c。P2b 合入后、P2c 之前：Triage 宿主的资产照旧按 `issue.update`（管理员、成员）判定，访客提交者不能给自己的条目加图片与附件，非提交者的成员能；`intake_view` 默认关、网页还在打 Plane 的旧地址，不算不安全。拒绝、重复的条目从此可以列出、可以删除。
- **与 Plane 的几处不同**（plane-diff 已登记）：不给 `status` 列全部状态（Plane 只列待处理）；访客提交者给了别的字段 403（Plane 静默丢弃）；访客提交者不能删（Plane 的创建者不分角色）；关闭的条目不能改内容（Plane 任何状态都能改）；待处理数含到期的暂缓、按只看自己数、归档的项目不列。
- **测试时间**：见 §6。

## 5. 本地门禁与 CI

- 检查点 1（`3ade4bc0`）：本机七项全过（cp1）；分支 CI 44470009970 attempt 1 全绿。
- 检查点 2：（待填）

## 6. 测试时间与两路

| 时点 | 本机 bootstrap 第 0 路 | 第 1 路 | 合计 | CI server (0) Test | CI server (1) Test |
|---|---|---|---|---|---|
| P2a 基线（P2a fin2；分支 CI 44436473289） | 474.3 秒 | 566.1 秒 | 1040.4 | 671 秒 | 681 秒 |
| 检查点 1（`3ade4bc0`） | 547.8 | 534.5 | 1082.3 | 713 | 812 |
| 检查点 2 | （待填） | | | | |

P2b 新增的 bootstrap 测试本机合计约 10 秒（组合 3.4–4 秒、交错 2–5 秒、矩阵 +1.6 秒、删除窗口 +0.5 秒），加 T9 的 HTTP 测试约 3 秒（不在 bootstrap）。新的顶层测试名让排在后面的测试换路（T11 换了 298 个、T12 又换了 366 个，两路名字数仍各半），所以本机检查点 1 约 100 秒从第 1 路挪到第 0 路。检查点 1 的分支 CI 上 web Unit 575 秒（P2a 498 秒）、e2e (1) 749 秒（P2a 628 秒，而 P2b 那时还没加 e2e），运行机器偏慢约 15–20%；按这个比例折算，server (1) 的 812 秒约是 +30 秒，没有能归于 P2b 的单路超 90 秒。同一份代码在 P2a 的两次 main CI 上是 686/671 与 857/881 秒，运行机器的快慢约 25%。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- `bootstrap/issue_write_locks_test.go` 403 行，再加行之前先拆。
- 生成的网页客户端按路由决定数组参数的发法（`csvListPaths`），新的 CSV 列表要登记（硬规则 03 §5）。
- 非接受的收集箱删除路径里项目归档检查两次（无害，保持了纯搬移）。

## 8. 文档与交接（T14）

- plane-diff：收集箱的列表、内容修改、删除、待处理数四行。
- 硬规则 03：§4 加 P2b 一段（提交者的定义、相对规则在 `write.check`、两个核心、有效状态）；§5 加两条（客户端的 CSV 路由、契约与处理器同一 Task 与先纯搬移）。
- 设计 §14 的 P2b 一行在合入时改为已完成。

## 9. 整分支审查与集成

（待填）
