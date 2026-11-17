# M7 / P1b 预检

> 2026-11-17：结论 **修正后通过**（Ready after fixes）。没有严重问题；7 处重要、10 处次要，都只涉及 spec 与 plan 的文字、Task 的边界与验收，没有须负责人决定的事（I4 若选“扩展事件”一路则是对 P1a 公开约定的追加，按设计 §15 D1 的口径属已授权范围，仍建议在阶段记录里写明）。
> - **目标**：[P1b spec](../v0/M7-collaboration/specs/P1b-notification-core.md)、[P1b plan](../v0/M7-collaboration/plans/P1b-notification-core.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2、§3.2–§3.4、§4、§6、§8–§10、§12、§15，[P1a spec](../v0/M7-collaboration/specs/P1a-subscriptions-events.md) §2.2、§2.4、§5，台账 `research/p1b-facts.md` 与[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。
> - **基线**：main `9a3a388a`（P1a 的 noff），worktree `codex/m7-p1b`。

## 1. 方法

一次 fresh 只读审查（`fable`）：不构建、不跑测试、不动 git，只写本文件。对照代码核了删除与关注的 SQL、事件的真实形状、动态的动词、River v0.47.0 的 snooze 实现（模块缓存里的源码）、access 的判定、bootstrap 的接线与守卫、e2e 的目录。下文路径相对 worktree 根，`si/` = `server/internal/`。没有核实的：各测试的实际耗时（没有运行），e2e 删除故事在 T2–T12 之间的实际红绿（按代码推断）。

## 2. 发现

### 重要

| # | 发现与证据 | 对 spec / plan 的修正 |
|---|---|---|
| I1 | **`notification.delete_rows` 的等待没有写时长，也没有上限，可以永远等下去。** River v0.47.0 的 snooze 把 `attempt` 减回去（`river@v0.47.0/internal/jobexecutor/job_executor.go:417-419`），不消耗次数，所以 spec §2.8 的“次数用尽记日志”对等待这一支永不触发；`issue.delete_rows` 若 25 次用尽被丢弃（`si/modules/issue/adapter/river/delete_rows.go` 默认 25），项目里永远有未软删的工作项，通知的任务就无限 snooze。先例里的 `river.JobSnooze(0)`（`si/modules/planning/adapter/river/delete_rows.go:60-62`）是“预算用完立刻续跑”，照抄到等待上是空转（snooze ≤ 调度间隔时任务直接回到 `available`，`job_executor.go:413-417`）。另外 Go 的两个等待有 28 处调用（`waitProjectDeletion`/`waitWorkspaceDeletion`），两个任务同事务投递、并行起跑，通知的任务几乎每次都会先 snooze 一轮：等待时长直接加到每个删除测试上。 | spec §2.8 写明：等待用固定的正时长（建议 500 ms–1 s，常量，不进配置），与预算续跑的 `JobSnooze(0)` 分开；等待的上限以 `job.CreatedAt` 计（建议 1 小时），超过后不再 snooze 而是返回错误，走 River 的退避与 25 次上限，用尽时记错误日志（此时留下的行靠读时过滤隐藏、随项目 60 天清理的 `CASCADE` 物理删除）。验收加一条：`issue.delete_rows` 被扣住时通知的任务 `metadata.snoozes` 递增、`attempt` 不超过 1、不写行。plan T3 的变异加“等待用 `JobSnooze(0)`”不必，但 T12 的时间记录要单列这 28 处等待的增量。 |
| I2 | **动词规则引用的出处与代码不符。** spec §2.3 说 `issue.updated` 的动词是“`domain.UpdateActivity` 的同一套：加入 `created`、移出 `deleted`、其余 `updated`”。实际 `UpdateActivity` 的每一行都是 `VerbUpdated`：标签的加入与移出（`si/modules/issue/domain/activity.go:199-206`）、负责人的加入（`:46-48`）与移出（`:210-213`）、父工作项（`hierarchy.go:72-74`）；事件本身不带动词（`si/shared/events.go:15-20`）。按原文写表驱动测试，T6 会钉住一个不存在的规则，或者实施者去改 M4 的动态。 | spec §2.3 改为自己的显式规则，不引 `UpdateActivity`：`labels`、`assignees` 的变化，只有 New 一侧（`NewID` 非空）为 `created`，只有 Old 一侧为 `deleted`；其余字段 `updated`；并写一句“与 M4 动态行的 `verb`（一律 `updated`）不同，登记差异”（或者干脆照动态一律 `updated`，由 P4 的卡片按新旧值判断——二选一写死）。T6 的表逐行列出。 |
| I3 | **“去掉 HTML 的变化”与变化序号冲突。** §2.9 说任务参数是“去掉 `description`、`comment_html` 的变化”，R4 说序号是“源变化在事件里的位置”。若订阅者把这两种变化从列表里删掉，后面的位置全部前移；评论事件里 `comment_html` 恰是位置 0、也就是可通知的变化本身（`si/modules/issue/app/collaboration_events.go:12-17`），删掉后任务无从编号。 | §2.9 改为“保留每个变化的位置，只把 `description`、`comment_html` 的 Old、New 置空”（或参数里每个变化带原始下标）；R4 加一句“序号在订阅者里按事件原始列表算出”。T8 的变异“参数里留 HTML”之外加一个“删掉变化使位置前移”的断言（`issue.created` 带描述与两个负责人时，负责人的序号与事件里一致）。 |
| I4 | **关联事件的 `issue` 不一定是发起的一端。** spec §2.3 称“关联事件的 `issue` 是发起的工作项”。`relationEvent` 用的是存储的规范行，“独立于授权这次请求的一端”（`si/modules/issue/app/issue_events.go:97-112`）；以 `blocking` 发起的关联反向存为对方的 `blocked_by`（`si/modules/issue/domain/relation.go:14-15,95-99`）。于是在 A 上加“A blocking B”，事件的 `issue` 是 B，通知发给 B 的关注者、字段 `blocked_by`，A 的关注者不收——与设计 §3.2“只在发起的那个工作项上一条”相反。 | spec 二选一写明并登记差异：(a) 通知发到存储行的 `issue` 一端（不改事件），§2.3 的措辞改为“存储行的起点；以 `blocking` 发起时是对方”，验收加这一例；(b) `relationEvent` 追加一个“发起端”的变化（只加不改，P1a §2.2 的做法），通知按它发并以请求的类型为字段。推荐 (b)，否则 H2 的“关联”对 `blocking` 的行为反直觉。另写明另一端的 id 取自 `related_issue` 变化（不是 `relation_type` 那一条的 `NewID`，它没有），序号取 `relation_type` 的位置 0。 |
| I5 | **快照端口照 spec 的写法接不出来。** §2.1、§2.6 说端口“由 `issue.Provide` 交出”“在 issue 里经它已有的项目端口组成”。`issue.Provide` 只有连接池（`si/modules/issue/module.go:66`，`si/bootstrap/app.go:138`），项目端口是 `issue.New` 的 `Deps`（`app.go:166-185`）；状态名与组、项目标识是 project 的表，issue 的 sqlc 不能联（`TestSQLCSchemaScope`）。同时 spec 没有写快照的读不得取锁：issue 手里读项目标识的现成路径之一是带锁的 `ProjectGate.Lock`（`si/modules/issue/app/ports.go:20-36`），分发持工作项行 `FOR SHARE` 时再等项目锁，会与“持项目锁、等工作项行”的每个写形成死锁环。 | §2.6 改为：issue 交出的 `NotificationSnapshot` 只答 issue 自己的列（名字、编号、`state_id`、创建者、当前负责人，一条 sqlc），由 `Provide` 交出；状态名与组、项目标识由 bootstrap 的 `notification_ports.go` 经 project 已有的不加锁的读（`StateCatalog.States`、项目标识）补齐。§2.9 加锁序的一句：“分发在 `ShareIssue` 之后只做不加锁的读与一条 INSERT，不得经 `ProjectGate.Lock`/`LockMany`”；Review Focus 已有这一条，T7 的单元测试用一个记录调用的假件钉住。 |
| I6 | **“允许改的已有测试”漏了几处会被后台分发打破的断言。** (1) 整表的 `river_job` 快照：`si/bootstrap/issue_draft_snapshot_test.go:32`、`issue_draft_publish_test.go:160`（草稿发布会发布 `issue.created`，T8 之后多一行 `notification.fanout`）；(2) 自带任务种类清单的运行器：`si/bootstrap/app_test.go:282`、`issue_delete_job_fixture_test.go:118`（清单里没有通知的两个种类；被扣住的删除夹具里通知的任务没有 worker 或不在预期里）、`request_jobs_test.go:18-20`；(3) `issue_retention_sets_test.go:21` 自己的模块清单；(4) 规模与清理夹具里的“待处理任务为 0”（`purge_scale_fixture_test.go:84-86`）现在也包含分发。另一面，清单里“R10 涉及的锁探针”没有给出文件名，偏松。 | plan Global Constraints 的清单加上 (1)–(4)，每处写明只允许的改法（快照按 `kind` 过滤或多认一种；种类清单加两个种类）；锁探针一项改为 T9 开头用 `grep -lE "pg_blocking_pids|wait_event_type='Lock'|WaitForLockWait"` 列出的文件名单写进台账后才允许改（见 m9）。 |
| I7 | **守卫转红的窗口与检查点的位置。** (1) T8 上线后才做 T9：T8 的提交处在“探针可能不稳”的状态，而 T8 自己要跑整包；应先加固探针再上线。(2) e2e 的删除故事从 T2（表出现）起就缺“被删的行”（`e2e/fixtures/assert/project.ts:213-223`、`workspace.ts:280-281` 要求父行下每张表都有被删的行），直到 T12 才修，其间 T3–T11 都无法以 e2e 为证；plan 只记了 Go 守卫 T2→T3 的红。(3) 唯一的检查点在 T12，离上线（T8）隔了四个 Task，若两路时间超线要到很晚才知道。 | T9 移到 T8 之前（先把探针改为按语句文本认等锁者，这一步不依赖分发存在），或并入 T8 并写明“T8 结束时整包两遍为绿”。T12 的 e2e 目录与夹具分两步：`asynchronousDeletionTable` 加 `notifications` 与 Go 夹具同在 T3（那时 e2e 仍红，台账记“e2e 删除故事 T2→T8 之后的夹具 Task 为红”），经接口产生通知行的夹具紧跟 T8（新 T9），在那里设第一个检查点（七项、分支 CI、两路时间）；T12 原处保留第二个检查点。 |

### 次要

| # | 发现与证据 | 修正 |
|---|---|---|
| m1 | 提及通知的 `change_seq` 没有写。 | §2.3 加：取收件人那一条 `mention` 变化的位置（每个收件人唯一）；`issue.created`、`issue.updated` 的提及字段为 `description`，评论事件的为 `comment`。 |
| m2 | 链接“取第一个业务变化、新旧值为地址”：只改标题的修改只有 `title` 一个变化，也不写动态（`si/modules/issue/app/update_issue_link.go:79-87`）。 | §2.3 写明：`issue.link_updated` 只有带 `url` 变化时通知（与动态一致），序号取 `url` 的位置；只改标题不通知。 |
| m3 | `DecideMany(ctx, action, project, accounts)` 没有工作区参数；`Authorize` 还比对项目属于目标的工作区（`si/modules/access/app/authorizer.go:48`），并带 `Creator` 事实（`:41`）。 | §2.7 写明工作区取自分发锁住的工作项行并传入、不属于它的项目按“没有项目”；`Creator` 恒为假（`issue.read` 没有 `AllowCreator`，`rules.go:140`），所以 `DecideMany` 只接受没有 `AllowCreator` 的规则，否则报错。等价测试加“别的工作区的项目”“已删除的工作区”两格；T5 的变异加“漏工作区比对”。 |
| m4 | `notifications` 的写者的锁序没有写下。同事务软删按 `(workspace_id, issue_id)` 一条语句、按扫描次序取锁；子树删除在一个事务里逐个事件执行；P1c 的“全部已读”是另一个多行写者，两者次序不同会死锁。 | §2.8 的软删语句照 `DeleteIssueLinkRows`（`si/modules/issue/adapter/postgres/queries/delete_issue.sql` 的“先按 id 升序 `FOR NO KEY UPDATE`”）；§5 交给 P1c：多行的写每批按 id 升序并 `SKIP LOCKED` 或以锁超时重试，语句头注释写不变式。 |
| m5 | 分发写行时用哪里的 `workspace_id`、`project_id` 没写；§2.2 没写 `project_id`、`workspace_id`、`issue_id`、`event_id`、`change_seq`、`sender`、`data` 的 `NOT NULL`。 | §2.9：行的工作区、项目取 `ShareIssue` 回答的行（`si/modules/issue/app/ports.go:156-161`），与参数不一致时记日志并以行为准；§2.2 补 `NOT NULL` 与 `change_seq >= -1` 的 CHECK。 |
| m6 | 分发持 `FOR SHARE` 期间，同一工作项的每个写在项目锁与写闸门之内等它（`write_lock_timeout` 5 秒，`server/configs/config.yaml:118`）；规模测试只“记录”持有时间。 | §4 给一个界：1000 个关注者一次分发的行锁持有时间不超过 1 秒，超过先报告；另记“持锁期间对同一工作项的一次 PATCH 的等待”。 |
| m7 | R9：`scaleSamples` 在同一个探针工作项上连续 20 次 PATCH（`si/bootstrap/issue_scale_evidence_test.go:67-99`），接入后上一次的分发可能正持着行锁，测到的是 m6 的等待而不是 `InsertTx` 的成本。 | R9 写明两组数字：每次 PATCH 之前等队列空（`InsertTx` 的成本），与不等（含行锁的等待）；“接入前”在 T1 同一台机器上先跑并记入台账。 |
| m8 | T10 过大：约 20 种变化 × 字段级断言再加去重、不通知的写、自动归档、草稿。 | 拆为 T10a（`issue.created`、`issue.updated`、归档与自动归档、不通知的写）与 T10b（评论、链接、关联、附件、草稿发布、提及与去重）；连同 I7 的新夹具 Task 共 16 个，仍在上限内。T11 大小合适。 |
| m9 | “14 个用探针的文件”不可复核：`pg_blocking_pids` 与 `wait_event_type='Lock'` 的字面匹配是 12 个文件，算上 `WaitForLockWait*` 辅助函数是 35 个。 | R10 与 T9 不写死数字，改为 T9 开头按固定的 grep 生成名单并记入台账。 |
| m10 | 测试时间没有预算数字。bootstrap 现有 716 个 `Test` 函数，CI 两路的 Test 步约 730–780 秒，八成线是 960 秒。 | plan 写明：P1b 新增的 bootstrap 测试每路的增量预算 ≤ 150 秒（含 I1 的等待），T9（新检查点）与 T12 各记一次两路的实测；接近 960 秒先报告（设计 §9）。规模、基准类一律选择性运行（已如此）。 |

## 3. 逐项回答

1. **并发论证（§2.8、§2.9、R3）成立**，前提是 I1、I5 改入。`issue.delete_rows` 对每个工作项行取冲突的锁：内层 `SELECT … FOR NO KEY UPDATE` 加外层 `UPDATE`（`si/modules/issue/adapter/postgres/queries/delete_rows.sql:8-14`），单个删除是 `UPDATE issues`（`delete_issue.sql` 的 `DeleteIssueRows`），都与 `FOR SHARE` 冲突；`ShareIssue` 带 `deleted_at IS NULL`，等待之后在新版本上复核（`subscriptions.sql:1-11`）。因此一个分发要么在该工作项软删提交之前提交（它的行对之后才开始的分批软删可见），要么在其后取锁、读不到行、什么都不写；`ProjectIssuesRemain` 为假之后不会再有分发写出这个项目的行。逐个情形：已软删的工作项（读不到行）；草稿（另一张表 `draft_issues`，不发通知，发布是新工作项且须过项目锁）；工作项不换项目（issue 的 SQL 里只有草稿改 `project_id`，`drafts.sql:31`）；工作区删除按事件列出的每个项目各投一个任务（`si/modules/issue/app/delete_rows_on_deletion.go:18-33`，planning 同），与项目删除同理；已归档的工作项照样被分批软删、照样计入“还有未软删的”。锁序：分发只持工作项行的 `FOR SHARE`，INSERT 的外键只取 `FOR KEY SHARE`，不与项目的 `FOR NO KEY UPDATE` 冲突；删除事务先项目锁、后工作项行、最后在订阅者里写通知行，没有环（只要快照不取项目锁，I5）。
2. **可通知的变化**：与设计 §3.2 的集合一致；`issue.created` 的形状（`issue_events.go:24-56`）、评论与链接末尾的 `issue`（`collaboration_events.go:19-29`）、附件只有 `issue` 一个变化（`si/modules/asset/app/attachment_publication.go:44-55`）、自动归档执行者为空且新值 `archive`（`auto_archive.go:94`）都与 spec 相符。不符的是 I2（动词）、I3（序号）、I4（关联的一端）、m1、m2。序号在事件落入任务参数之后是稳定的（重试读同一份参数）。快照对 P4 够用（台账 §6：卡片只读工作项的 id、名字、编号、标识与变化的字段、动词、新旧值）。收件人与类型与设计 §3.3 一致。
3. **批量判定**：可以复用 `domain.Decide`——它是纯函数（`si/modules/access/domain/decide.go:48-104`），`Authorize` 只是读事实后调用它（`authorizer.go:32-55`）；两条新的读放在 workspace、project 各自的模块里是对的；等价测试可行。补 m3。
4. **模块边界与文件清单**：sqlc、授权、清理目录、River 种类、配置、e2e 目录、删除守卫、运行角色都列到了；R2 与守卫一致（每个模块目录须在 `moduleActions` 或 `actionlessModules` 之一，`si/bootstrap/actions_test.go:22-60`，空的 `Register` 不触发规则测试），加入时照惯例写一行理由。缺的是 I5、I6。单独的 `notification.delete_batch/delete_budget` 照 M6 的先例（`server/configs/config.yaml:108-125`），不算过度设计。
5. **次序与门禁**：见 I7、m8、m10。T3 先于 T4 使用 `ProjectIssuesRemain` 已在 plan 里写明，可以。
6. **过度设计或缺口**：没有发现过度设计。缺口是 I1 的上限、m6 的界、m7 的口径。

## 4. 核实为正确的事实（摘要）

- River 为 v0.47.0（`server/go.mod:25`）；snooze 减回 `attempt` 并在 `metadata.snoozes` 计数；Go 的等待断言 `attempt > 1` 即失败（`si/bootstrap/planning_deletion_fixture_test.go` 的 `waitPlanningDeletion`），snooze 不会触发它。
- `planning.delete_rows` 的写法（订阅者校验整份载荷、参数三项、每批一个事务、预算后 `ProjectDeletionRemaining`）如 spec 所引（`si/modules/planning/app/delete_rows_on_deletion.go`、`delete_rows.go`、`adapter/river/delete_rows.go`）。
- 订阅者按登记次序同步投递，现为 issue、planning、asset（`si/bootstrap/app.go:153`）；清理模块次序现为 asset → issue → planning → project → workspace → identity（`app.go:253`），插入 notification 的位置可行。
- `issue.read` 是项目级、三种角色（`si/modules/access/domain/rules.go:140`）；只看自己 = 项目角色访客且未开 `guest_view_all_features`（`decide.go:92`）。
- `asynchronousDeletionTable` 现不含 `notifications`（`e2e/fixtures/assert/catalog.ts:94-110`）；`furnishWorkspace` 里指派的是执行者本人（`e2e/fixtures/furnish.ts:115`），不会产生通知，T12 的夹具改动是必要的。
- `parentAsyncTable`、`waitProjectDeletion`、`waitWorkspaceDeletion` 的现状如 spec 所述（`si/bootstrap/issue_project_deletion_jobs_test.go:37-39,103-109,227-237`）。
- 评论的纯文本存为 `Text`（`si/modules/issue/domain/comment.go:21`），R6 的摘录取它是可行的。

## 5. 处理结果

2026-11-17 由主控逐条改入 spec 与 plan（同一提交）：

- I1：spec §2.8 等待改为 500 毫秒的固定 snooze、以 `job.CreatedAt` 计 1 小时的上限后返回错误；§4 加“被扣住时只 snooze、`attempt` 不超过 1”；§6 与 plan 的时间预算计入删除测试的等待。
- I2：`issue.updated` 的动词一律 `updated`，与 M4 动态一致，加入与移出由新旧值区分（spec §2.3、§5 交给 P4）。
- I3：参数保留全部位置，`description`、`comment_html` 只置空 Old、New（spec §2.9、R4）；T6 的变异含“删掉变化使位置前移”。
- I4：选 (b)：关联的两个事件追加 `requested_on`（建立为请求所在的工作项，移除为被授权的那一端，即用例的 `h.own()`），通知只发到这一端（spec §2.3、R11；设计 §6.1 同改；plan T4）。
- I5：issue 交出 `NotificationIssue`（只答自己的列），状态名与组、项目标识由 bootstrap 经 project 不加锁的读补齐；锁序一句写进 spec §2.6。
- I6：plan Global Constraints 的允许清单补全，每处写明只允许的改法；锁探针名单由固定的 grep 生成。
- I7：探针加固提到上线之前（T8），上线为 T9，e2e 经接口的夹具紧跟（T10）并设检查点 1；T13 之后检查点 2；e2e 删除故事 T2–T10 为红写进 plan 与 spec §6。
- m7：R9 改为一个轻量的写入基准（M4 的数字来自 66 万行的删除规模测试，太重），在基线的临时 worktree 与分支上各跑一次。
- m1–m3、m5、m6：写进 spec §2.3、§2.7、§2.2、§2.9、§4、R9。m4：§2.8 的软删按 id 升序加锁，§5 交给 P1c。m8：组合测试拆为 T11、T12，共 15 个 Task。m9：不写死文件数。m10：每路增量预算 ≤ 150 秒写进 plan 与 spec §4。
