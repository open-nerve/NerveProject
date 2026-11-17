# M7 / P1a 阶段记录：关注接口与通知要用的事件

> 状态：T1–T13 已完成，待整分支审查、最终候选与合入（§7 在合入后回填）。spec 与 plan 见 [P1a spec](../specs/P1a-subscriptions-events.md)、[P1a plan](../plans/P1a-subscriptions-events.md)，预检 [re-82](../../../reviews/re-82-M7P1a预检.md)。基线 main `8d17eed7`（M7 设计的 noff）。原始证据在 worktree 的 `.superpowers/sdd/P1a-subscriptions-events/`（台账 `progress.md`、每次检查的日志与收据、变异），随阶段归档。

## 1. 交付

- 评论、链接的六个事件追加所在工作项的 `issue` 变化（建立、修改在 New 一侧，删除在 Old 一侧，R1）。
- `issue.created`、`issue.updated`、评论的建立与修改追加这一次新提及的 `mention` 变化（成员过滤、未经可见性过滤；评论的修改为新正文减旧正文，旧正文是锁下重读的那一份）。
- R7：描述的保存只让 `可见 ∩ 新提及` 自动关注，评论的修改只让执行者与 `可见 ∩ (新 − 旧)` 自动关注；建立不变。
- `POST /issues/{id}/subscribe`、`/unsubscribe`：编排是 `Locks.share`（不进写闸门、不取项目锁，预检后事务里 `FOR SHARE` 锁工作项行、在锁住的行上再判定），用例只写关注行；没有动态、事件、`updated_at` 的变化；已归档照常。
- `IssueDetail.is_subscribed`：七个回答详情的操作都带，读在最后；网页 store 在列表行覆盖详情时保留它。
- 守卫：写的锁目录第三类 `subscriptionWrites`（真实探针三段）、删除窗口 22 → 24、闸门说明、权限矩阵两个操作各三行（74 格）。

## 2. Task 与提交

| Task | 提交 | 内容 | 变异 |
|---|---|---|---|
| T1 | `223e70a1` | phase `M7/P1`、设计 §14、基线回填 | — |
| T2 | `c07765cd` | 六个事件的 `issue` 变化 | 4 |
| T3 | `459c9807` | 描述的 `mention`、R7（描述） | 5 |
| T4 | `46626813` | 评论的 `mention`、R7（评论修改）、清洗幂等 | 5 |
| T5 | `65d743b2` | `ShareIssue`、`RemoveSubscriber`、`IsSubscribed`，语句头的锁序不变式 | 7 |
| T6 | `7f499a69` | `Locks.share` 与两个用例 | 5 |
| T7 | `7bd4b237` | 契约、生成物、处理器与接线 | 1 |
| T8 | `01384c6e` | 第三类锁目录与探针、删除窗口两例、闸门说明 | 8 |
| T9 | `55920ae1` | `is_subscribed`（服务端、契约、网页三处字面量与 `choose`） | 4 |
| T11 | `83daecb2` | 权限矩阵两个操作各三行 | 2 |
| T10 | `9db03a35` | 六个组合测试（接口与行、R7 经真实接口、看不到的 404、与删除两种先后、等锁时撤权、项目删除的分批任务、503） | 12 |
| T12 | `6ee25714` | H1 (API) 故事（本机 chromium 172 毫秒） | 2 |
| T13 | 本记录所在提交 | 文档与交接（§6） | — |

T11 提到 T9 之后、检查点之前：plan 漏了权限矩阵的覆盖守卫也随 T7 加的两个操作转红（T8 的实施者发现，主控裁定）。

## 3. 局部裁定（实施中）

- T5、T6：`ShareIssue` 答专门的 `SharedIssue`（四列），放在 `IssueReader` 端口上而不给 `Locks` 加字段（改 `NewLocks` 的签名会动约 20 处 M4 的回归测试）；存储端口是新的 `SubscriptionWriter`；用例不持事件、动态、`MarkChanged` 的端口，§2.4 第 4 条在结构上成立。
- T8：探针分三段——被拒者不取锁且先等 `issues` 表；写关注行时只持本工作项行的 `FOR SHARE`、不持项目与工作区行；项目与工作区行被别的事务 `FOR NO KEY UPDATE` 扣住时照常 204。探针放新文件 `issue_subscription_write_locks_test.go`（原文件会过 400 行）。
- T9：建立与发布草稿共用 `IssueCreation.answer`；单元假件的 `IsSubscribed` 只在测试给定关注者时记调用（M4 的单元测试逐条比较调用序列，同 `fakeGate` 的先例）。
- T10：删除的时刻早于关注的提交（删除取项目锁后读时钟，再等工作项行），交错的断言不以关注的提交为下界。
- 新 Go 文件不加版权头：仓库里的 Go 文件都没有。
- 证据：失败的运行也是证据，不删日志（T3 删过一个夹具错误的运行日志，台账已记，此后照此执行）。

## 4. 本地门禁与 CI

- 检查点 cp1（`83daecb2`）七项全过：gen-check、lint-go、test（922 秒）、lint-web、knip、test-web、build+e2e（233 秒，480 passed）。
- 检查点的分支 CI 44021715408 attempt 1 全绿：server (0) 作业 699 秒（Test 步 580 秒）、server (1) 618 秒（Test 587 秒）、web 911 秒、e2e (2) 789 秒、e2e (1) 883 秒。
- 本机的 Docker 启动失败两次（t8-shard0、t11-shard0，testcontainers 连 `docker.sock` 超时、reaper 容器消失，不同的测试），各只重跑那一路一次即通过；与产品无关，归 M6-M8-05 一类的环境问题。

## 5. 测试时间与两路

- `bootstrap` 包 702 → 708 个测试（T10 后），两路各 354 个；一个排在第 19 位的新测试让其后 630 个旧测试换了路。
- 本机两路（T10 后，`9db03a35`）：第 0 路 618 秒（其中 bootstrap 490.1 秒，其余为 `bootstrap` 以外的全部包），第 1 路 411 秒（bootstrap 407.7 秒）。
- CI 的两路：设计合入的 main CI 44004492566 第 0 路 Test 步 912 秒（只改文档，慢在运行器；前三次 686–755 秒）；本段检查点的分支 CI 第 0 路 580 秒、第 1 路 587 秒。第 0 路作业的约束是 25 分钟（1500 秒），最慢的一次 1094 秒（73%）；`bootstrap` 那一半的单次 `go test` 离 20 分钟很远。spec §4 原写的“960 秒”把单次 `go test` 的期限套在了整路上，T13 已改。负责人已收到重新平衡两路的建议（不加作业、不改期限），未决。
- 权限矩阵：74 个新格子使 `TestPermissionMatrix` 本机由 30.7 秒到 40.8 秒（+10.1 秒，预检估的 1.3 秒按每格 17 毫秒算，实际每格约 136 毫秒：取消关注的格子先在副本上放一行）。
- e2e：H1 (API) 本机 chromium 172 毫秒。

## 6. 文档与交接（T13）

- `plane-diff.md`：“工作项的写的取锁”补关注的例外，“自动关注”补 R7，新增“手动关注、取消关注”“工作项详情的 `is_subscribed`”两行。
- [硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md) 加关注的例外一段（`Locks.share`、安全的理由、锁序不变式）。
- M4 设计的事件表：评论、链接、建立、修改四行指向 P1a spec §2.2。
- 输入交接：[M4 交接 M7-02](../handoffs/M4-closeout.md#M7-02)、[M5 交接 M5-M7-02](../handoffs/M5-closeout.md#M5-M7-02) 写“处理结果（M7/P1a）”，保持 open（通知在 P1b）；动态的 `comment` 列不进公开接口（设计 §11）。
- spec §4 与设计 §9 的测试时间写法更正（§5）。

## 7. 集成

（合入后回填：整分支审查、修复轮、最终候选、分支 CI、noff、main CI、归档与清理。）
