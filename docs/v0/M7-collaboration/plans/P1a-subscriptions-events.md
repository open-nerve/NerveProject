# M7 / P1a 关注接口与通知要用的事件 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：plan 经预检 [re-82](../../../reviews/re-82-M7P1a预检.md) 修正后通过。

**Goal：** 评论、链接事件带工作项 id，工作项与评论的事件带这一次新提及的账户；手动关注、取消关注；详情回答带 `is_subscribed`；自动关注只给新被提及的人。

**Architecture：** 全在 `issue` 模块：事件的变化在发布处追加；关注的编排是 `Locks` 上的新方法 `share`（不进写闸门、不取项目锁，事务里 `FOR SHARE` 锁工作项行再判定），用例只写行；`is_subscribed` 是每个调用者的读，挂在 `seenOne` 与建立的回读上。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/Playwright，不加依赖。

**Spec：** [P1a-subscriptions-events](../specs/P1a-subscriptions-events.md)（§2 的合同、§2.4 的例外与锁序不变式、§3 的 R1–R7 是本计划的约束）；设计 [M7](../M7-design.md) §3.1、§3.2、§6.1；M4 的评论写（`comment_writes.go`）、草稿的不进闸门之路（`lock_draft.go`）与表情事件（`collaboration_events.go`）是模板。

## Global Constraints

- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m7-p1a`，分支 `codex/m7-p1a`，基线为设计合入 main 的 noff；主目录永远 main。提交详细中文，前缀 `feat/fix/docs/test/refactor(M7/P1a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`，推送用 `/usr/bin/git`；noff 已获连续授权。
- `plane/`、`refer/` 只读且不在其内跑 git；不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。
- T1 改 phase 为 `M7/P1`；不延长任何 `until`。
- 手写文件约 400 行内；SQL 只经本模块 sqlc；不放宽 `TestSQLCSchemaScope`、`rawsql_test`、`archtest`、删除/清理/运行角色守卫。
- 每个（操作，新声明的码）在同一 Task 有 issue 的 apitest 处理器测试回答；不新增问题码（三个码都已有中英文案）；`make test-web` 照跑。
- 接口说明的文字会进入 `web/packages/api-client/src/schema.gen.ts`，避开网页关键词规则的词（`plane`、`upgrade`、`estimate`、`integration`、`timeline`、`auto close` 等）。
- 每个 Task：先写会失败的测试并看到它因所测性质失败，再实现到通过；加锁、可见、幂等性质至少一个能让测试失败的变异（只计能编译的变异）。每 Task 跑聚焦回归。
- 允许改的既有测试（其余 M4 测试是回归，不改断言）：spec §4 点名的事件形状断言；两份详情金样；`TestIssueDetailShape`；`TestAHiddenParentIsNotInTheAnswer` 的比较；写的锁目录（`issue_draft_write_locks_test.go`、`issue_write_locks_test.go`）与删除窗口（`issue_deletion_window_cases_test.go`）；`gate_test.go` 的说明；依赖重复插入关注行的 M4 断言（R7，T3、T4 逐个点名）。
- 守卫转绿的时点：T7 加了两个契约操作，写的锁目录与删除窗口两个守卫随即要求它们有行，T8 让它们转绿；所以 T7 只跑聚焦回归，T8 之后才跑 bootstrap 的整包。T9 使契约的 `IssueDetail` 多一个必填字段，网页的生成与字面量在同一个 Task，检查点放在 T9 之后。
- 证据放 worktree 的 `.superpowers/sdd/P1a-subscriptions-events/`（台账 `progress.md`、收据经 `run-check.py`、变异经 `mutate.py`）。

## Review Focus

1. 变化的次序与 Old/New 一侧（spec §2.2、R1）；没有改已有变化；“写入但未改变”与评论正文不变时没有事件。
2. 提及集合：描述用 M4 已算出的集合，评论是成员过滤后的集合与差集；旧正文是锁下重读的；R7 的自动关注只给新被提及的人。
3. 关注的锁：编排在 `Locks.share`；预检、`FOR SHARE`、锁下再判定；与删除的两种先后；不进写闸门；没有动态、事件、`updated_at` 的变化；锁序不变式写在注释里。
4. `is_subscribed` 在七种详情回答一致，列表行没有，网页 store 保留它。
5. 拒绝：404 与 403 的区分、只看自己的访客、删除窗口、已归档照常、503。

## Task 1（T1）：阶段与台账

**Files：** `tools/keywords.json`；`docs/v0/M7-collaboration/M7-design.md`（§14 加 P1a 行“进行中”）。

- [ ] Step 1：在 worktree 建台账（`run-check.py` 改台账路径、`doc-links.py`、`archive.py`、`wait-run.sh` 取自设计台账，`mutate.py` 取自 M6 的归档）。
- [ ] Step 2：phase 改 `M7/P1`，`node tools/keywords.mjs` 通过；设计 §14 加 P1a 行；文档链接检查；提交 `docs(M7/P1a): 进入 P1a 阶段`。

## Task 2（T2）：评论、链接事件的工作项 id

**Files：** `server/internal/modules/issue/app/{collaboration_events.go,link_events.go}` 与 `{collaboration_events_test.go,links_test.go,link_writes_test.go,crawl_on_link_change_test.go}`；`server/internal/bootstrap/{issue_comment_events_test.go,issue_links_test.go,issue_link_writes_test.go,issue_link_race_helpers_test.go}`。

- [ ] Step 1：先改单元与 bootstrap 的形状断言为 spec §2.2（建立、修改 `NewID`，删除 `OldID`，追加在最后），看到失败。
- [ ] Step 2：`commentEvent`、`linkEvent` 追加变化（一处函数，按事件类型选一侧）；`crawl_on_link_change_test.go` 加一例带 `issue` 变化的事件仍只投一次抓取。
- [ ] Step 3：聚焦回归（`issue/app`、bootstrap 的评论、链接、链接竞争、资产删除测试）；变异：删除类放到 New 一侧、漏掉一种事件；提交。

## Task 3（T3）：描述的提及进入工作项事件，描述的自动关注只给新被提及的人

**Files：** `server/internal/modules/issue/app/{issue_events.go,update_issue.go,create_core.go,description.go}` 与单元测试；`server/internal/bootstrap/issue_event_writes_test.go` 与描述提及、自动关注的组合测试。

- [ ] Step 1：失败的测试：建立（含发布草稿）带两个成员、一个只看自己的访客、一个非成员的提及，事件有三个 `mention`（升序、在最后）；PATCH 新增一个、保留一个，事件只有新增的那个；“写入但未改变”的保存没有事件；不带描述的 PATCH 没有 `mention`；R7：仍被提及的人在关注行被软删之后，再保存描述不会被加回，新被提及的会。
- [ ] Step 2：`createdChanges` 收 `checked.mentions`，`updatedChanges` 收 `relations.mentions.Added`，各追加升序的 `mention`；`prepareDescription` 的 `subscribers` 改为 `可见 ∩ Added`（建立不变）。先列出 M4 里依赖重复插入的断言（预计没有），逐个写进台账再改。
- [ ] Step 3：聚焦回归（`issue/app`、bootstrap 的工作项事件、描述、提及、草稿发布）；变异：用未经成员过滤的 id、在“写入但未改变”分支发布、关注全部可见被提及者；提交。

## Task 4（T4）：评论的提及进入评论事件，评论修改的自动关注只给新被提及的人

**Files：** `server/internal/modules/issue/app/{comments.go,comment_writes.go,mentions.go}` 与单元测试；`server/internal/bootstrap/issue_comment_events_test.go` 与评论提及的组合测试。

- [ ] Step 1：失败的测试：建立带两个成员与一个非成员的提及，事件有两个 `mention`；修改从 `{a}` 到 `{a, b}` 只有 `b`，从 `{a, b}` 到 `{a}` 没有；修改所用的旧正文是锁下重读的那一份（照 `collaboration_events_test.go:61-68` 的写法）；清洗对已清洗 HTML 的提及不变；R7：`a` 取消关注后，评论被修改（仍提及 `a`）不会把 `a` 加回，新提及的 `b` 会。
- [ ] Step 2：评论的写取得成员过滤后的提及集合（复用 `descriptionTargets` 的成员读）；建立照旧关注“执行者 + 可见的被提及者”，修改只关注“执行者 + 可见 ∩ (新 − 旧)”；修改时对旧正文清洗求差。
- [ ] Step 3：聚焦回归；变异：修改时用全集而非差集、旧正文用预检时的；提交。

## Task 5（T5）：关注的存储

**Files：** `server/internal/modules/issue/adapter/postgres/queries/{issues.sql,subscriptions.sql}`（新文件放 `ShareIssue`、`RemoveSubscriber`、`IsSubscribed`）、对应的 Go 与 adapter 测试；`delete_issue.sql` 等语句头注释。

- [ ] Step 1：adapter 测试：`ShareIssue` 对已删除的行无结果、与另一事务的 `UPDATE issues` 互相等待（锁探针）；`RemoveSubscriber` 只软删未删除的那一行、写时刻与执行者，重复调用无影响；`IsSubscribed` 只认未删除的行；`InsertSubscribers` 重复插入不报错。
- [ ] Step 2：写语句、`make gen-go`；更新语句头的锁说明与锁序不变式（spec §2.4）。
- [ ] Step 3：adapter 包回归；提交。

## Task 6（T6）：`Locks.share` 与关注、取消关注的用例

**Files：** `server/internal/modules/issue/app/{lock_share.go,subscription.go}`（新）与单元测试；`app/lock.go`（`Locks` 的说明）；`app/ports.go`（存储端口加三个方法）。

- [ ] Step 1：失败的单元测试（假存储、在两次调用之间翻转的假授权）：预检拒绝 404、403 且不开事务；锁下行已删除 404；锁下再判定（只看自己的访客、两次调用之间撤权）404；关注写一行、取消软删、重复都是空操作；不调用 `MarkChanged`、不发布事件。
- [ ] Step 2：`Locks.share`：预检 → `busyTx` → `ShareIssue` → `decide` → `fn`；两个用例只在 `fn` 里写行。
- [ ] Step 3：回归；变异：去掉锁下再判定、去掉 `FOR SHARE`、在锁前写；提交。

## Task 7（T7）：接口与处理器

**Files：** `api/modules/issue.yaml`、`api/openapi.yaml`、`api/dist/openapi.yaml`（生成）；`web/packages/api-client/src/schema.gen.ts`（生成）；`server/internal/modules/issue/adapter/http/{handler.go,subscriptions.go}` 与测试；`server/internal/modules/issue/module.go`。

- [ ] Step 1：契约（两条路径，`x-problem-codes: [issue.not_found, forbidden, server_busy]`，说明避开关键词）；`make gen-go`、`make gen-web`。
- [ ] Step 2：处理器与接线；处理器测试覆盖 204 与三个码（`server_busy` 用假错误）。
- [ ] Step 3：`apitest`、`issue` 各包的聚焦回归、`node tools/keywords.mjs`、`make check:types` 一类的网页类型检查；提交（bootstrap 的两个目录守卫在 T8 转绿）。

## Task 8（T8）：写的锁目录与删除窗口

**Files：** `server/internal/bootstrap/{issue_draft_write_locks_test.go,issue_write_locks_test.go,issue_deletion_window_cases_test.go,gate_test.go}`。

- [ ] Step 1：第三类目录 `subscriptionWrites`：两行，形状“无工作区锁、无项目锁、工作项行 `FOR SHARE`、不进闸门、被拒的调用者不取锁”，以真实探针核对（同既有两类的写法）；`writesOnIssues` 的并集包含它。
- [ ] Step 2：删除窗口加两例（22 → 24）：项目、工作区删除待处理时关注、取消答 404，与不存在的逐字节相同；`gate_test.go` 的说明写明关注、取消关注与草稿同为不进闸门的例外。
- [ ] Step 3：bootstrap 整包（两路）；变异：关注取项目锁、不锁工作项行；提交。

## Task 9（T9）：`is_subscribed`

**Files：** `api/modules/issue.yaml`（`IssueDetail` 必填布尔）与生成物（`make gen-go`、`make gen-web`）；`issue/domain/issue.go`；`issue/app/{ports.go,seen.go,create_core.go,publish_draft.go}` 与 `fakes_parents_test.go`；`issue/adapter/postgres`（`IsSubscribed` 的读）；`issue/adapter/http/issues.go` 与 `issues_test.go` 的两份金样；`platform/httpserver/apitest/issue_shapes_test.go`；`bootstrap/{issue_parents_test.go,issue_search_snapshot_ports_test.go}`；`web/apps/web/core/store/issue/{fake-issues.ts,issue.store.types.test.ts,issue.store.ts}`。

- [ ] Step 1：失败的测试：七种详情回答的 `is_subscribed`（建立者 true，修改者未关注 false，关注后 true）；`TestIssueDetailShape` 改为恰多两个必填字段；`TestAHiddenParentIsNotInTheAnswer` 比较前去掉 `is_subscribed`；网页 store 的 `choose` 在列表行覆盖详情时保留 `is_subscribed`。
- [ ] Step 2：`IssueReader` 加一个方法（三个实现），`seenOne` 与建立、发布草稿的回读调用它；HTTP 映射；金样加字段；网页的三处字面量与 `choose`。
- [ ] Step 3：`issue` 各包、`apitest`、`check:types`、`make test-web`、`make lint-web`、`knip`；提交。

## 检查点（T9 之后）

整段七项（gen-check、lint-go、test、lint-web、knip、test-web、build+e2e），推送分支、等分支 CI（`wait-run.sh`）。

## Task 10（T10）：组合测试

**Files：** `server/internal/bootstrap/issue_subscriptions_test.go`（新）。

- [ ] Step 1：经接口：关注 / 取消 / 重复；列表 `subscriber_id` 与成员列表 `relation=subscribed`；取消后评论、被指派、被新提及重新关注，仍被提及的人在描述保存、评论修改后不被加回；已归档的工作项与项目照常；只看自己的访客、撤权、私有项目 404；数据库的行（软删的时刻与执行者）。
- [ ] Step 2：交错：关注与工作项删除两种先后；扣住工作项行时撤权，放开后 404 且没有行；关注与项目删除的分批任务；锁等待超时 503（`issue.write_lock_timeout` 调小的组合）。
- [ ] Step 3：记录新测试的本机时间与它们分到的路；提交。

## Task 11（T11）：权限矩阵

**Files：** `server/internal/bootstrap/permission_matrix_subscriptions_test.go`（新）。

- [ ] Step 1：两行：`issueColumns`（看不到 404，工作区管理员、成员看未加入的公开项目 403）、归档的列、已归档的工作项（`inDone(true)`）都成功；`TestThePermissionMatrixCoversEveryOperation` 通过；记录矩阵耗时的增量；提交。

## Task 12（T12）：H1 (API) 故事

**Files：** `e2e/stories/collaboration/h1-subscription.spec.ts`（新）；需要时 `e2e/fixtures/issues.ts` 加两个调用。

- [ ] Step 1：PAT 与浏览器登录对等：关注、取消、`is_subscribed`、`subscriber_id` 列表、数据库的关注行；本机 chromium 时间 ≤ 3 秒；提交。

## Task 13（T13）：文档与交接

**Files：** `docs/v0/plane-diff.md`；`docs/v0/claude_to_gpt/03-rules-and-pitfalls.md`（§4）；`docs/v0/M4-issue-core/M4-design.md`（事件表的评论、链接两行）；`docs/v0/M7-collaboration/handoffs/{M4-closeout.md,M5-closeout.md}`；`docs/v0/M7-collaboration/reviews/P1a-subscriptions-events-review.md`（阶段记录）；设计 §14。

- [ ] Step 1：plane-diff：关注两行（R3）、“自动关注”行补 R7、“工作项的写的取锁”行补关注的例外；硬规则 §4 加关注的例外一段（spec §2.4）；M4 设计的事件表指向本 spec §2.2；M4 交接 M7-02 写“处理结果（M7/P1a）”（关注接口、`is_subscribed`、事件已带 id 与提及，通知在 P1b，保持 open）；M5 交接 M5-M7-02 写“附件事件本已只带 id，P1a 不改，消费在 P1b”；阶段记录（Task、收据、变异、两路预估）；文档链接检查；提交。

## 收尾

T13 之后整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。
