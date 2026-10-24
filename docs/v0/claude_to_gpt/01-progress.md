# 01 进度（截至 2026-10-25）

## 1. 项目一句话
Nerve 是从 Plane（AGPL-3.0，`plane/` 只读）裁剪出来的轻量项目管理系统：复用 Plane 的前端（`web/`），后端用 Go 重写（`server/`，模块化单体、端口与适配器、OpenAPI 优先的 `/api/v0`、每个模块自己的 sqlc、PostgreSQL 18.6、River 只投递），一个二进制内嵌前端。账户不区分人和代理：同一个账户，人用界面、代理用接口。保留的功能照 Plane 一比一（含表结构），砍掉的功能删干净。总体设计是 [v0-design.md](../v0-design.md)，它的 9.4 是里程碑进度表。

## 2. 里程碑
| M | 状态 | 说明 |
|---|---|---|
| M0 基础骨架、M1 前端瘦身、M2 账户认证、M3 工作区与项目 | 已完成 | 各自的设计、spec、plan、评审在 `docs/v0/M0-…`–`M3-…`；M3 收尾合并 `53a6a05b`（2026-10-16） |
| M4 工作项核心 | 进行中 | 本交接的对象 |
| M5–M8 | 未开始 | M4 收尾时给它们写 `handoffs/M4-closeout.md` |

## 3. M4 到哪了
**设计**：[M4-design.md](../M4-issue-core/M4-design.md)，2026-10-17 合并（`1eb9686a`，持续集成第 336 次通过）。三稿三审（17.1–17.3）。负责人 2026-10-17 "全部按建议" 裁定（17.4）：
- D1 = B：工作项的写只锁项目行（不锁工作区行）；删除工作区先锁它的项目。
- D2 = A：M4 作为一个里程碑；某段的 plan 实测超过上限时，控制者可以直接拆成 P<k>a、P<k>b，不用请示（第 12 节约束 2）。
- D3 = A：现在引入领域事件。D4 = A：自动的动作记在 nerve 名下（actor 为 NULL）。D5 = A：不做 `issue_versions`。D6 = A：不做评论的回复。
- D7 = A：删除项目、工作区之后工作项一侧的行由按项目的 River 任务软删除；规模目标每个项目约 2 万个未删除的工作项。
- D8 = B：项目管理员可以删除、不能编辑别人的评论，不记动态。
- 第 11 节的十一个架构问题（11.1–11.11，含 11.11 每个项目的写闸门）已批准。

**Phase**（设计第 12 节分段，第 15 节是进度表）：

| Phase | 状态 | 合并 | 持续集成 | 文档 |
|---|---|---|---|---|
| P1a `issues` | 已完成（15 个 Task） | `5ce30a1a`（2026-10-18） | 第 337 次通过 | [spec](../M4-issue-core/specs/P1a-issues.md)、[plan](../M4-issue-core/plans/P1a-issues.md)、[评审](../M4-issue-core/reviews/P1a-issues-review.md) |
| P1b `issue-update` | 已完成（15 个 Task） | `7db78788`（2026-10-19） | 第 338 次通过 | [spec](../M4-issue-core/specs/P1b-issue-update.md)、[plan](../M4-issue-core/plans/P1b-issue-update.md)、[评审](../M4-issue-core/reviews/P1b-issue-update-review.md) |
| P2 `issue-access` | 已完成（14 个 Task） | `f8696490`（2026-10-21） | 第 339 次通过 | [spec](../M4-issue-core/specs/P2-issue-access.md)、[plan](../M4-issue-core/plans/P2-issue-access.md)、[评审](../M4-issue-core/reviews/P2-issue-access-review.md) |
| P3 `issue-hierarchy` | 已完成（16 个 Task） | `dfc9378b`（2026-10-24） | **第 340 次失败**：一个 M3 的老故事 W11 偶发失败，不是 P3 的改动，见 [06](06-ci-w11-flake.md) | [spec](../M4-issue-core/specs/P3-issue-hierarchy.md)、[plan](../M4-issue-core/plans/P3-issue-hierarchy.md)、[评审](../M4-issue-core/reviews/P3-issue-hierarchy-review.md) |
| P4–P9（后端）、P10a–P16（前端）、收尾 | 未开始 | — | — | 设计第 12 节各段 |

P1 在实施前按 D2 拆成 P1a、P1b。P3 在实施前重新切成 16 个 Task。每段都是整分支终审之后一个修复轮。

## 4. HEAD 上已经有的
- **接口**（`api/modules/issue.yaml`）：`createIssue`、`getIssue`、`getIssueByIdentifier`、`updateIssue`、`deleteIssue`（连同子树）、`archiveIssue`、`unarchiveIssue`、`listSubIssues`、`addSubIssues`、`createIssueRelations`、`listIssueRelations`、`deleteIssueRelation`。
- **表**：迁移 `00015`–`00020`：`issues`、`issue_assignees`、`issue_labels`、`issue_subscribers`、`issue_activities`、`issue_relations`。
- **机制**：
  - 加锁的路 `Locks.run`，有三种形式：单项目、`lock_across.go`（加关联）、`lock_either.go`（删除关联）；
  - 每个项目的写闸门 `platform/gate`，等锁超时、闸门满答 503 `server_busy`，只在 `bootstrap/gate.go` 映射；
  - 访客规则（`Grant.OwnOnly`、`domain.Issue.HiddenFrom`，`seen` 遮住看不到的父工作项）；
  - 创建者规则 `AllowCreator`；
  - 删除项目、工作区、状态、标签时工作项一侧的同步连带（P9 改为任务）；
  - 起止日期和 `bodyshape` 的 `date` 检查；
  - 权限矩阵里工作项的各行；
  - 设计 3.5 的交错测试中落在 P1a–P3 的各个（哪个在哪一段，见各 spec 第 2 节）。
- **端到端**：I1、I2、I8、I9、I15、I17 的接口版本，加上此前的全部故事，共 126 个（124 个 chromium，2 个 A4 的 firefox）。页面版本从 P12 起。
- **动态**：已写入，还没有读动态的接口（P4 的 `listIssueActivities`）。

## 5. 本机仓库的状态
- 本地 `main` 与 `origin/main` 一致，P3 的合并（`dfc9378b`）之后只有本交接的文档。远端只有 `main`，本地只有 `main` 和主目录一个工作树（空的 `m4-p1b` 已在 2026-10-25 删除）。
- 交接文件的 `status` 都还是 `open`，按惯例在 M4 收尾时随评审关闭；P1a–P3 在其中写了"处理结果（M4/P<k>）"。
