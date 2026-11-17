# M7 / P1a 关注接口与通知要用的事件 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：plan 待预检。

**Goal：** 评论、链接事件带工作项 id，工作项与评论的事件带这一次新提及的账户；手动关注、取消关注；详情回答带 `is_subscribed`。

**Architecture：** 全在 `issue` 模块：事件的变化在发布处追加；关注照 `DraftLocks.run` 的写法（不进写闸门，事务里 `FOR SHARE` 锁工作项行再判定）；`is_subscribed` 是每个调用者的读，挂在 `seenOne` 与建立的回读上。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/Playwright，不加依赖。

**Spec：** [P1a-subscriptions-events](../specs/P1a-subscriptions-events.md)（§2 的合同与 §3 的 R1–R6 是本计划的约束）；设计 [M7](../M7-design.md) §3.1、§3.2、§6.1；M4 的评论写（`comment_writes.go`）、草稿的不进闸门之路（`lock_draft.go`）与表情事件（`collaboration_events.go`）是模板。

## Global Constraints

- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m7-p1a`，分支 `codex/m7-p1a`，基线为设计合入 main 的 noff；主目录永远 main。提交详细中文，前缀 `feat/fix/docs/test/refactor(M7/P1a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`，推送用 `/usr/bin/git`；noff 已获连续授权。
- `plane/`、`refer/` 只读且不在其内跑 git；不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。
- T1 改 phase 为 `M7/P1`；不延长任何 `until`。
- 手写文件约 400 行内；SQL 只经本模块 sqlc；不放宽 `TestSQLCSchemaScope`、`rawsql_test`、`archtest`、删除/清理/运行角色守卫。
- 每个（操作，新声明的码）在同一 Task 有 issue 的 apitest 处理器测试回答；不新增问题码（三个码都已有中英文案）；`make test-web` 照跑。
- 接口说明的文字会进入 `web/packages/api-client/src/schema.gen.ts`，避开网页关键词规则的词（`plane`、`upgrade`、`estimate`、`integration`、`timeline`、`auto close` 等）。
- 每个 Task：先写会失败的测试并看到它因所测性质失败，再实现到通过；加锁、可见、幂等性质至少一个能让测试失败的变异（只计能编译的变异）。每 Task 跑聚焦回归。
- M4 的工作项、评论、链接测试是回归：只改 spec §4 点名的形状断言、两份详情金样、`TestIssueDetailShape` 与 `TestAHiddenParentIsNotInTheAnswer` 的比较，不改其他断言。
- 证据放 worktree 的 `.superpowers/sdd/P1a-subscriptions-events/`（台账 `progress.md`、收据经 `run-check.py`、变异经 `mutate.py`）。

## Review Focus

1. 变化的次序与 Old/New 一侧（spec §2.2、R1）；没有改已有变化；“写入但未改变”与评论正文不变时没有事件。
2. 提及集合：描述用 M4 已算出的集合，评论是成员过滤后的集合与差集；旧正文是锁下重读的。
3. 关注的锁：预检、`FOR SHARE`、锁下再判定；与删除的两种先后；不进写闸门；没有动态、事件、`updated_at` 的变化。
4. `is_subscribed` 在六种详情回答一致，列表行没有。
5. 拒绝：404 与 403 的区分、只看自己的访客、已归档照常、503。

## Task 1（T1）：阶段与台账

**Files：** `tools/keywords.json`；`docs/v0/M7-collaboration/M7-design.md`（§14 加 P1a 行“进行中”）。

- [ ] Step 1：在 worktree 建台账（工具取自设计台账：`run-check.py` 改台账路径、`doc-links.py`、`archive.py`、`wait-run.sh`、`mutate.py` 取自 M6 的归档）。
- [ ] Step 2：phase 改 `M7/P1`，`node tools/keywords.mjs` 通过；设计 §14 加 P1a 行；文档链接检查；提交 `docs(M7/P1a): 进入 P1a 阶段`。

## Task 2（T2）：评论、链接事件的工作项 id

**Files：** `server/internal/modules/issue/app/{collaboration_events.go,link_events.go}` 与 `{collaboration_events_test.go,links_test.go,link_writes_test.go}`；`server/internal/bootstrap/{issue_comment_events_test.go,issue_links_test.go,issue_link_writes_test.go,issue_link_race_helpers_test.go}`。

- [ ] Step 1：先改单元与 bootstrap 的形状断言为 spec §2.2（建立、修改 `NewID`，删除 `OldID`，追加在最后），看到失败。
- [ ] Step 2：`commentEvent`、`linkEvent` 追加变化（一处函数，按事件类型选一侧）；`crawl_on_link_change_test.go` 加一例带 `issue` 变化的事件仍只投一次抓取。
- [ ] Step 3：聚焦回归（`issue/app`、bootstrap 的评论、链接、链接竞争、资产删除测试）；变异：删除类放到 New 一侧、漏掉一种事件；提交。

## Task 3（T3）：描述的提及进入工作项事件

**Files：** `server/internal/modules/issue/app/{issue_events.go,update_issue.go,create_core.go}` 与单元测试；`server/internal/bootstrap/issue_event_writes_test.go`。

- [ ] Step 1：失败的测试：建立（含发布草稿）带两个成员、一个只看自己的访客、一个非成员的提及，事件有三个 `mention`（升序、在最后）；PATCH 新增一个、保留一个，事件只有新增的那个；“写入但未改变”的保存没有事件；不带描述的 PATCH 没有 `mention`。
- [ ] Step 2：`createdChanges` 收 `checked.mentions`，`updatedChanges` 收 `relations.mentions.Added`，各追加升序的 `mention`。
- [ ] Step 3：聚焦回归（`issue/app`、bootstrap 的工作项事件与草稿发布）；变异：用未经成员过滤的 id、在“写入但未改变”分支发布；提交。

## Task 4（T4）：评论的提及进入评论事件

**Files：** `server/internal/modules/issue/app/{comments.go,comment_writes.go,mentions.go}` 与单元测试；`server/internal/bootstrap/issue_comment_events_test.go`。

- [ ] Step 1：失败的测试：建立带两个成员与一个非成员的提及，事件有两个 `mention`；修改从 `{a}` 到 `{a, b}` 只有 `b`，从 `{a, b}` 到 `{a}` 没有；修改所用的旧正文是锁下重读的那一份（照 `collaboration_events_test.go:61-68` 的写法）；清洗对已清洗 HTML 的提及不变。
- [ ] Step 2：评论的写取得成员过滤后的提及集合（复用 `descriptionTargets` 的成员读，自动关注照旧用可见的那部分加执行者），修改时对旧正文清洗求差。
- [ ] Step 3：聚焦回归；变异：修改时用全集而非差集、旧正文用预检时的；提交。

## Task 5（T5）：关注的存储

**Files：** `server/internal/modules/issue/adapter/postgres/queries/{issues.sql,subscriptions.sql}`（新文件放 `ShareIssue`、`RemoveSubscriber`、`IsSubscribed`）、对应的 Go 与 adapter 测试；`delete_issue.sql` 等语句头注释。

- [ ] Step 1：adapter 测试：`ShareIssue` 对已删除的行无结果、与另一事务的 `UPDATE issues` 互相等待（锁探针）；`RemoveSubscriber` 只软删未删除的那一行、写时刻与执行者，重复调用无影响；`IsSubscribed` 只认未删除的行；`InsertSubscribers` 重复插入不报错。
- [ ] Step 2：写语句、`make gen-go`；更新语句头的锁说明。
- [ ] Step 3：adapter 包回归；提交。

## Task 6（T6）：关注、取消关注的用例

**Files：** `server/internal/modules/issue/app/subscription.go`（新）与单元测试；`app/ports.go`（存储端口加三个方法）。

- [ ] Step 1：失败的单元测试（假存储）：预检拒绝 404、403；锁下行已删除 404；锁下再判定（只看自己的访客、撤权）404；关注写一行、取消软删、重复都是空操作；不调用 `MarkChanged`、不发布事件。
- [ ] Step 2：照 `DraftLocks.run` 写：预检 → `busyTx` → `ShareIssue` → `decide` → 写。
- [ ] Step 3：回归；变异：去掉锁下再判定、去掉 `FOR SHARE`、在锁前写；提交。

## Task 7（T7）：接口与处理器

**Files：** `api/modules/issue.yaml`、`api/openapi.yaml`；`server/internal/modules/issue/adapter/http/{handler.go,subscriptions.go}` 与测试；`server/internal/modules/issue/module.go`。

- [ ] Step 1：契约（两条路径，`x-problem-codes: [issue.not_found, forbidden, server_busy]`，说明避开关键词）；`make gen-go`、`make gen-web`。
- [ ] Step 2：处理器与接线；处理器测试覆盖 204 与三个码（`server_busy` 用假错误）。
- [ ] Step 3：`apitest` 回归、`node tools/keywords.mjs`；提交。

## Task 8（T8）：`is_subscribed`

**Files：** `api/modules/issue.yaml`（`IssueDetail` 必填布尔）；`issue/domain/issue.go`；`issue/app/{ports.go,seen.go,create_core.go,publish_draft.go}`；`issue/adapter/postgres`（`IsSubscribed` 的读）；`issue/adapter/http/issues.go` 与 `issues_test.go` 的两份金样；`platform/httpserver/apitest/issue_shapes_test.go`；`bootstrap/{issue_parents_test.go,issue_search_snapshot_ports_test.go}`；`issue/app/fakes_parents_test.go`。

- [ ] Step 1：失败的测试：六种详情回答的 `is_subscribed`（建立者 true，修改者未关注 false，关注后 true）；`TestIssueDetailShape` 改为恰多两个必填字段；`TestAHiddenParentIsNotInTheAnswer` 比较前去掉 `is_subscribed`。
- [ ] Step 2：`IssueReader` 加一个方法（三个实现），`seenOne` 与建立、发布草稿的回读调用它；HTTP 映射；金样加字段。
- [ ] Step 3：`issue` 各包与 `apitest` 回归；提交。

## Task 9（T9）：网页的类型

**Files：** `web/packages/api-client/src/schema.gen.ts`（生成）；`web/apps/web/core/store/issue/{fake-issues.ts,issue.store.types.test.ts}`。

- [ ] Step 1：`make gen-web`；两处字面量加 `is_subscribed: false`；`check:types`、`make test-web`、`make lint-web`、`knip` 通过；提交。

## Task 10（T10）：组合测试

**Files：** `server/internal/bootstrap/issue_subscriptions_test.go`（新）。

- [ ] Step 1：经接口：关注 / 取消 / 重复；列表 `subscriber_id` 与成员列表 `relation=subscribed`；取消后评论、被指派、被提及重新关注；已归档的工作项与项目照常；只看自己的访客、撤权、私有项目 404；数据库的行（软删的时刻与执行者）。
- [ ] Step 2：交错：关注与工作项删除两种先后（锁探针）；关注与项目删除的分批任务；锁等待超时 503（`issue.write_lock_timeout` 调小的组合）。
- [ ] Step 3：变异各一（见 T6）；记录新测试的本机时间与它们分到的路；提交。

## Task 11（T11）：权限矩阵

**Files：** `server/internal/bootstrap/permission_matrix_collaboration_test.go`（或新文件 `permission_matrix_subscriptions_test.go`）。

- [ ] Step 1：两行：`issueColumns`（看不到 404，工作区管理员、成员看未加入的公开项目 403）、归档的列、已归档的工作项（`inDone(true)`）都成功；`TestThePermissionMatrixCoversEveryOperation` 通过；记录矩阵耗时的增量；提交。

## Task 12（T12）：H1 (API) 故事

**Files：** `e2e/stories/collaboration/h1-subscription.spec.ts`（新）；需要时 `e2e/fixtures/issues.ts` 加两个调用。

- [ ] Step 1：PAT 与浏览器登录对等：关注、取消、`is_subscribed`、`subscriber_id` 列表、数据库的关注行；本机 chromium 时间 ≤ 3 秒；提交。

## Task 13（T13）：文档与交接

**Files：** `docs/v0/plane-diff.md`；`docs/v0/M7-collaboration/handoffs/{M4-closeout.md,M5-closeout.md}`；`docs/v0/M7-collaboration/reviews/P1a-subscriptions-events-review.md`（阶段记录）；设计 §14。

- [ ] Step 1：plane-diff 加关注两行（spec R3）；M4 交接 M7-02 与 M5 交接 M5-M7-02 写“处理结果（M7/P1a）”（事件已带 id 与提及，消费在 P1b，保持 open）；阶段记录（Task、收据、变异、两路预估）；文档链接检查；提交。

## 检查点与收尾

- T8 之后做一次整段七项检查点（gen-check、lint-go、test、lint-web、knip、test-web、build+e2e），推送分支、等分支 CI（`wait-run.sh`）。
- T13 之后整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。
