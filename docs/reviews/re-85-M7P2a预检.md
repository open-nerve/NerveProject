# M7 / P2a 预检

> 2026-11-19：结论 **修正后通过**（Ready after fixes）。没有严重问题；9 处重要、10 处次要。R1 的重新划分成立，R2 的等价按构造成立。重要问题都落在 spec 与 plan 的文字、文件清单、允许清单与 Task 次序上，不需要改设计的取舍。须向负责人说明的有两件：R1 改的是已批准设计 §12 的两行（理由成立）；I5、I6 使 P2a 触及 `api/modules/asset.yaml` 与网页的三个手写文件，超出 spec §2.1 现在写的范围。
> - **目标**：[P2a spec](../v0/M7-collaboration/specs/P2a-intake-submit-triage.md)、[P2a plan](../v0/M7-collaboration/plans/P2a-intake-submit-triage.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2、§3.5、§3.6、§3.10、§3.11、§4、§5、§6.1、§9、§12、§15，台账 `research/p2a-facts.md` 与[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §4。
> - **基线**：main `31f38580`（P1c 的 noff），worktree `codex/m7-p2a`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git、不起容器、不跑测试；在 `server/` 跑了一次 `go build ./...`（通过）。对照代码核了 `Locks` 的四种编排、新建与修改两条写路径、`pickState`、三条读工作项的 SQL 与它们的行转换、搜索、资产宿主的判定、工作项与项目的删除步骤、清理步骤、Go 与 e2e 的目录守卫、写锁表、契约与形状测试、网页的详情类型，以及 Plane 的 `views/intake/base.py`、`serializers/intake.py`。下文路径相对 worktree 根，`si/` = `server/internal/`，`ia/` = `si/modules/issue/app/`。

没有核实的：

- 各测试的实际耗时与两路 CI 的余量（没有运行）。
- 权限矩阵种一个 Triage 工作项之后，已有各行的 `check` 是否受影响（没有逐行读）。
- sqlc 对 `LEFT JOIN` 列生成的可空类型，以及 `go build` 之外的任何门禁。
- I1 的反例没有在数据库里实际执行，是按 SQL 的三值逻辑推出的。

## 2. 发现

### 重要

| # | 发现与证据 | 对 spec / plan 的修正 |
|---|---|---|
| I1 | **`00051` 的 CHECK 按 spec 的写法挡不住 `field` 为空的行。** spec §2.2 写 `verb IN ('created','updated','deleted') OR (field = 'intake' AND verb IN (…五个))`。`field` 可空（`server/migrations/sql/00019_issue_issue_activities.sql:13`），创建动态的 `field` 就是 NULL（`si/modules/issue/domain/activity.go:56`）。`field` 为 NULL、`verb = 'accepted'` 时表达式是 `false OR (NULL AND true)` = NULL，CHECK 对 NULL 放行。所以 §4 要的反例“`field ≠ intake` 时的新 verb 被拒”在最常见的一种行上不成立。 | §2.2 改为 `… OR (field IS NOT DISTINCT FROM 'intake' AND verb IN (…))`（或 `coalesce(field, '') = 'intake'`）。迁移测试的反例写两条：`field` 为 NULL 配 `accepted`、`field = 'state'` 配 `accepted`，都要被拒。plan 的“verb 的反例改为…”改成“**加**两条”：现有的 `'commented'` 一条（`server/migrations/issue_schema_test.go:318`）仍然成立，不要改。 |
| I2 | **e2e 的删除夹具改错了地方：`expectIssueDeleted` 的五处调用都不经 `furnishIssueRows`。** R1 的结论本身成立（见 §3 第 1 条），但 plan T11 与允许清单写的是“`furnishIssueRows` 经提交与接受造一个可删除的、带条目行的工作项”。`expectIssueDeleted` 要求被删的那个工作项之下每个键都有一行随它删除（`e2e/fixtures/assert/issue.ts:415-465`）。五处调用各自搭自己的树：`e2e/stories/issue/i15-archive-issue.spec.ts:233,460`、`i17-delete-issue.spec.ts:125,299`、`i21-purge.spec.ts:215`（`i17` 的 `root`/`plan`、`i21` 的 `root` 都是故事里 `createIssue` 出来的，`i17-delete-issue.spec.ts:59-83`）。`furnishIssueRows` 只喂项目、工作区的删除断言。另外 `fixtures/assert/project.ts`、`workspace.ts` 是按目录读行的，自己不造行，不需要改。 | T11 与允许清单改为：① 加一个夹具（如 `furnishAcceptedIntake(api, adminToken, projectId, parentId)`）：管理员打开 `intake_view`、提交、接受，再经 `updateIssue` 把它放到指定工作项之下（`issues.parent_id` 是图里的键，子工作项的条目行算“在它之下”）；② 五处故事各调一次（故事文件进允许清单，写明“只加这一步”）；③ `furnishIssueRows` 里加一次提交（待处理即可），供项目、工作区的断言；它现在只收一个成员的 token（`e2e/fixtures/furnish.ts:66`），开关要管理员打开，放进 `furnishPlanningRows`（已收 `adminToken`、已在改项目的开关，`furnish.ts:106-108`）或另加参数；④ 允许清单去掉 `project.ts`、`workspace.ts`，加 `e2e/fixtures/` 的接口辅助。五处里删除者有成员（`i17:299` 是 `ann`），接受要管理员的 token，逐处确认故事里有。 |
| I3 | **“七个源端守卫”里 `deleteIssueRelation` 既到不了，也挂不上 `write.check`。** ① `write.check` 在“任一端”的编排里根本不运行：`precheck` 在 `w.instead != nil` 时直接返回 `eitherOf`（`ia/lock.go:129-132`），`lockAndDecide` 直接返回 `eitherUnder`（`lock.go:177-179`），两处都在调 `w.check` 之前。② 按 §2.3 的不变式，Triage 的工作项永远没有关联行：提交不建关联，`createIssueRelations` 对源端 409、对另一端 422。所以这个守卫经接口到不了，§4“七个源端写 409”与“七个源端守卫至少在一个矩阵行里出现”对它写不出不靠 SQL 造违规状态的测试。`unarchiveIssue` 则到得了（未归档的 Triage 工作项上，守卫先于“已是所求状态”的空操作，`ia/archive_issue.go:77-83`）。 | 源端守卫改为六个：`updateIssue`、`archiveIssue`、`unarchiveIssue`、`deleteIssue`、`addSubIssues`、`createIssueRelations`。`deleteIssueRelation` 不加守卫、不声明 `issue.in_triage`；不变式测试（DP-7 那一个）加一条“在收集箱里的工作项没有关联行、没有父、不在迭代模块里”。§2.6、§2.9、§4、plan T7、T12 的“七”都改。 |
| I4 | **R6 的“修改核心”今天不是一个可以调用的核心，接缝没有定形；时钟会读两次。** 新建有核心（`IssueCreation.createLocked`，`ia/create_core.go:64`，并把时刻交还调用者）。修改没有：`UpdateIssue.Execute` 自己调 `Locks.run`，动作写死 `issue.update`（`ia/update_issue.go:87-89`），归档检查、`asAnswered`、`check`、时钟（`:137`）、`write`、事件、读回都在它的闭包里（`:90-158`）。分诊在自己的 `Locks.run`（动作 `intake.triage`）里，不能调 `Execute`。plan 说“修改核心各只加一个参数或接缝”，没有说接缝是什么。若分诊自己读一次时钟写条目行与 `intake` 动态，修改的那一段再读一次，就违反“这个请求写的每一行同一个时刻”（硬规则 03 §4 第 9 条）。 | spec R6、§2.5 写明：把 `Execute` 闭包里的那一段原样抽成 `(u *UpdateIssue) updateLocked(ctx, actor, h held, p domain.IssuePatch, at *time.Time) (domain.Issue, time.Time, error)`（或把时刻交还，像 `createLocked`）；`Execute` 只剩 `CheckIssuePatch`、`write{…}` 与调用它。接受以补丁 `{StateID: 默认状态}` 调它，时刻是分诊在最后一把锁之后读的那一个。源端守卫挂在 `Execute` 的 `write.check` 上，不进抽出的那一段，所以接受不经它（R6 的这句话因此成立）。P2b 的内容修改用同一个接缝换白名单与授权，现在定形免得 P2b 再改。写明接受时两条动态的先后（`intake` 在前或状态在前，写死并由测试钉住；id 是 v7，时刻相同）。T6 加变异“接受读了第二次时钟”。核实过的事实：`pickState(states, &默认状态)` 不会被拒（默认状态不可能是 Triage，`si/modules/project/adapter/postgres/queries/states.sql:79`），`issueState` 认 Triage 作当前状态（`ia/writes.go:34-40`），`completed_at` 随默认状态的组（默认状态可以是完成组），状态变化不重算 `sort_order`（`si/modules/issue/domain/patch.go:142-147`，与普通修改一致，spec 写一句）。 |
| I5 | **R5 的资产判定缺三样：契约、开关的来源、动作的选择次序。** ① 资产的四个操作把 issue 的码写在自己的契约里：`createAsset`、`deleteAsset`、`completeAsset`、`duplicateAsset` 都声明 `project.archived, issue.archived`（`api/modules/asset.yaml:53,106,132,168`），资产的处理器测试按码列操作（`si/modules/asset/adapter/http/business_errors_test.go:85`）。§2.7 新增的 `issue.in_triage`、`issue.intake_disabled` 要进这四处与这个测试；spec §2.1 与 plan 的允许清单都没有 `asset.yaml`。② 开关无处可读：`Inspect` 不加锁、没有 `LockedProject`；`Check` 在资产自己的项目锁之下，只经 `AssetProjectReader.AssetProjectArchived` 读归档（`ia/asset_access.go:40-42,67`）。§2.4 只给 `LockIssueProject(s)` 加了 `intake_view`，资产这条路读不到。③ 动作在读工作项之前就定了：`read` 先选 `ActionUpdate` 再 `seeAs`（`asset_access.go:106-109,131`），而 Triage 宿主要先读到条目才知道用 `intake.update`。 | §2.1 加 `api/modules/asset.yaml` 与资产的处理器测试；§2.7 写明：`read` 改为先 `GetIssue`、按 `InTriage()` 选动作、再 `decide`（同包可用 `Locks` 的两个方法，一次读）；相对规则（非管理员须是提交者，403）在 `Inspect` 与 `Check` 都判；`AssetProjectReader` 改为同时交出 `intake_view`（project 的端口、SQL、bootstrap 适配各加一列，进 §2.1 的 project 一行），只在 `Check`（锁下）判，与 DP-3 一致；次序写死：404 → 403（角色、再提交者）→ `project.archived` → `issue.intake_disabled` → 终态的 `issue.in_triage`。若想把 P2a 收窄，R5 整段可以随内容修改移到 P2b（同一个动作、同一个时段；P2a 里没有任何路径能把准备好的描述图片绑上去）——这是可选项，移的话在 R1 里写明期间 Triage 宿主照 `issue.update`。 |
| I6 | **`IssueDetail.intake` 会打破 spec 说“不改”的网页文件与三处 Go 测试。** `IssueDetail` 的每个属性都是必填（`api/modules/issue.yaml:1705-1707`）。网页：`web/apps/web/core/store/issue/fake-issues.ts:47-51` 的 `issueDetailOf` 与 `issue.store.types.test.ts:23,52` 都是带类型的 `IssueDetail` 字面量，少了 `intake` 过不了类型检查；`issue.store.ts:31-34` 在列表行盖到详情上时只保留 `description_html` 与 `is_subscribed`，`intake` 会被丢掉（P1a 加 `is_subscribed` 时改的就是这三个文件）。Go：`si/platform/httpserver/apitest/issue_shapes_test.go:36` 钉着“`IssueDetail` = `Issue` 加 `description_html`、`is_subscribed`”；`si/modules/issue/adapter/http/issues_test.go:64-66` 是详情的逐字节金样；`si/bootstrap/issue_subscribed_answers_test.go:33` 是“七个回答都带”的先例。spec §2.1 写网页“只有生成的类型、`error-messages.ts` 与两份 `errors.json`”，plan 的允许清单只提了码。 | §2.1 的 `web/` 一行加这三个文件（`issue.store.ts` 的合并保留 `intake`，并有一条与 `keeps_is_subscribed_when_a_list_row_arrives` 同形的测试）。允许清单加：`issue_shapes_test.go` 的 `own` 加 `intake`；`issues_test.go` 的金样加 `"intake":null`；网页的两个夹具。T10 的“七个操作都带”照 `TestEachDetailAnswerSaysWhetherTheCallerSubscribes` 写。 |
| I7 | **允许清单漏了 issue 模块里钉着表序的测试，也没有“跟着签名编译”的一类。** 加一张 issue 的表、一个删除步骤、一个清理步骤，下面这些已有断言必然要改，清单里一处都没有：`si/modules/issue/adapter/postgres/deletion_remaining_test.go:15-17`（逐项钉 `DeletionRowSteps()` 的次序）、`delete_issue_test.go:104` 与 `delete_rows_test.go:99`（逐表的步骤表）、`purge_test.go:36`（逐表的清理函数）、`ia/delete_rows_test.go:233` 与 `ia/cascade_test.go:66,99,112`（语句次序与假实现）。另外接口与签名一变，假实现与直接调用处要跟着编译：`IssueRowsDeleter` 等接口的假实现；`ia/create_core_test.go:83,106` 直接调 `createLocked`、`prepareLocked`；`NewLocks` 若加时钟（m1），八个测试文件的构造处（`ia/create_issue_test.go:56` 等）。 | 允许清单加上面六个文件，改法各一句（“表序、步骤表加 `intake_issues` 一项”）。另加一类：“接口加方法、函数加参数带来的假实现与调用处的编译跟随，不改断言”，台账逐个记。spec §2.10 写死 `intake_issues` 在 `issueRowSteps`、`DeletionRowSteps`、清理步骤里的位置（建议紧跟 `issues` 之后；清理在 `issues` 之前）。 |
| I8 | **三个守卫的红的窗口没有写，或者写短了；关键词的 phase 没有改。** ① 动作：T5–T8 的用例要用 `intake.submit`、`intake.triage`、`intake.update`，而 plan 把“三个动作进规则表”放在 T9。动作一旦进 issue 的 `Actions()` 而规则表里没有，`TestEveryActionHasARuleAndEveryRuleAnAction`（`si/bootstrap/actions_test.go:149`）就红，从 T5 到 T9，plan 没有列。② 写锁表：`issueWrites` 要求契约里每个带 `issue` 标签的非 GET 操作有一行（`si/bootstrap/issue_write_locks_test.go:44-46,151-159`），T9 加契约后红到 T12；plan 只列了路由、逐码与矩阵覆盖三项。③ `tools/keywords.json:2` 还是 `M7/P1`；设计 §12 末段与硬规则 03 §4“每段开始的提交改关键词守卫的 phase”要求 P2a 的第一个实现提交改为 `M7/P2`（P2a、P2b 共用），plan T1 没有这一步。 | ① 三个动作与规则表、`rules_test` 的三行提到 T3（领域）里，与动作的声明同一个提交。② 红的窗口加一条“写锁表的覆盖守卫 T9 → T12”。③ T1 加“phase 改 `M7/P2`，`node tools/keywords.mjs` 通过，不延长任何 `until`”。 |
| I9 | **分诊的两个 404 与 403 的先后没有写死。** §2.5 写“工作项看不到或没有条目行 404；看得到而不是管理员 403”。照 `Locks.run` 自然的次序，`seeAs` 先判定（访客规则的 404、角色的 403），`w.check` 在它之后（`ia/lock.go:134-142`）。所以成员对一个看得到的普通工作项调分诊，得到的是 403，不是“没有条目行”的 404。两种次序都不透露什么（他读得到这个工作项，详情里就有 `intake`），但它是拒绝次序的合同，测试要逐值钉。 | §2.5 的次序改为：请求体 422 → 看不到 404 → 403 → 没有条目行 404（`w.check`，预检与锁下各一次）→ `project.archived` → `issue.intake_disabled` → 迁移 409 → 锁下的 422。验收加一格：成员对普通工作项 403、管理员对普通工作项 404、只看自己的管理员兼访客对别人的条目 404，且两种 404 与随机 id 逐值相同。 |

### 次要

| # | 发现与证据 | 修正 |
|---|---|---|
| m1 | 读的路径没有时钟。有效状态要 `now`，而回答详情的 `seenOne` 在 `Locks` 上（`ia/seen.go:96`），`Locks` 没有时钟（`ia/lock.go:34-41`）；`getIssue`、`getIssueByIdentifier` 都经它。 | spec §2.8 写明 `now` 从哪里来（建议给 `Locks` 一个 `Clock`，一次回答读一次；写的回答用写的那个时刻）。动态的 `old_value` 用有效状态还是存的状态，写死（建议有效状态：到期的暂缓被接受记 `pending → accepted`）。 |
| m2 | `Intake` 为 nil 有两种意思。三条共用 `gen.GetIssueRow` 的查询会带条目；搜索的行是手工拼的 `GetIssueRow`（`si/modules/issue/adapter/postgres/search_issues.go:38-44`），列表引擎走别的转换，这两处的 `Intake` 恒为 nil，即使有已接受的条目行。今天靠不变式安全（搜索与列表都不含收集箱里的工作项）。 | `domain.Issue.Intake` 的注释写明“只有 `GetIssue`、`GetIssues`、`SubIssues` 的行带；列表与搜索的行不带，不得在它们上面调 `InTriage()`”。连接条件写成 `ON ii.issue_id = i.id AND ii.deleted_at IS NULL`（放在 `ON` 里才用得上部分唯一索引，也才是左连接）。 |
| m3 | T4 的“条目行的锁读”不需要。`lockAndDecide` 在项目锁下重读工作项（`ia/lock.go:187`），带出来的 `h.issue.Intake` 就是锁下的事实；写条目行的每条路径都持项目锁，项目删除的分批在项目行已删之后，分诊等到锁时读不到项目、答 404。 | T4 去掉锁读，分诊直接用 `h.issue.Intake`、按 `issue_id` 更新。少一把锁，时钟的位置也更清楚。 |
| m4 | P2a 与 P2b 之间 main 上的状态没有写下来。拒绝、重复的条目是终态，P2b 之前删不掉、改不了、也列不出来（只有按 id、编号与通知能到）；待处理的可以“接受再删”。不算不安全：开关默认关（`server/migrations/sql/00010_project_projects.sql:21`），网页的收集箱还在打 Plane 的旧地址，没有页面能造出条目；项目删除与清理照常带走它们。已接受的工作项的动态里会有 `intake` 一行，旧的 `inbox.tsx` 走 `default` 分支显示一句通用文字，不崩（`web/apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/inbox.tsx:31-42`）。 | spec §5 或 R1 加一段“P2a 合入后、P2b 之前 main 的已知状态”，阶段记录向负责人说明。 |
| m5 | `00051` 的 Down 把 CHECK 收回三个动词，表里若已有 `intake` 的动态会失败。 | Down 先删 `field = 'intake'` 且动词不在三个之内的行，或写明 Down 只对空表负责（照仓库里已有的放宽迁移的口径）。 |
| m6 | 重复目标能不能是已归档的工作项，没有写。设计只列四条（同项目、看得到、不在收集箱、不是自己）；选择器用的搜索不给已归档的（`search_issues.sql:22`）。 | §2.5 写死（建议允许，照设计的四条；或与选择器一致不允许，二选一并登记）。统一的 422 经核不透露：隐藏的、别的项目的、不存在的目标同一个回答，且它排在 403 与迁移 409 之后。 |
| m7 | `status` 的非法值、`duplicate_to_id` 等于自己各在哪一步、什么码，没有写。 | 写死：`status` 不在五个之内由契约的枚举回答（400 还是 422 照仓库里枚举请求字段的现有口径，写出来）；“是自己”归入锁下的统一 422（已是 spec 的意思，写明不提前到请求体）。 |
| m8 | 提交的回答与新建核心的 `answer`。条目行在 `createLocked` 返回之后才插入（§2.4），核心的回答里 `Intake` 是 nil。plan T10 写“新建的 `answer` 读条目与重复目标的可见”，其实普通新建与草稿发布永远没有条目，提交也不可能有重复目标。 | T5 写明提交在插入条目行之后读回一次再回答；T10 改为“只有 `seenOne` 处理重复目标的可见；`answer` 不变”。重复目标的读是一次按主键的 `GetIssue`，只在状态为 `duplicate` 时发生，目标与条目同项目、用同一个授予。 |
| m9 | Task 的大小。T2 一个 Task 里有两个迁移、sqlc、授权、迁移测试、清理、单个删除、分批删除、Go 守卫的种子，再加 I7 的六个测试文件；T13 有 H8、H10、H11、详情、不变式五组。 | T2 拆为 T2a（迁移、sqlc、授权、迁移测试）与 T2b（删除图、清理、守卫与夹具）；T13 照 P1c 拆为 a、b。超过 16 个时，把 R5 移到 P2b（I5 的可选项）或把 T11 并进 T10 之后的同一个提交组，实施前定。e2e 的红窗口 T2 → T11 可以接受：检查点 1 在 T13，Go 的目录守卫在 T2 当场转绿。 |
| m10 | plane-diff 的清单（T16）少几条：Plane 的提交与修改都不看 `intake_view`（`plane/apps/api/plane/app/views/intake/base.py:176-332`）；Plane 缺 Triage 状态时现建一个（`:240-252`），Nerve 当作内部错误；分诊在 Plane 是同一个 PATCH，Nerve 是独立的 `POST …/triage`；回答的形状（Plane 是 `{id, status, …, issue}` 的封套、200，Nerve 是 `IssueDetail` 带 `intake`、提交 201）；Plane 接受时没有默认状态答 400（`app/serializers/intake.py:54-63`），Nerve 的默认状态总在；Triage 守卫先于 `project.archived`（硬规则 03 §4 第 7 条的次序之外的登记例外）。 | T16 的清单与硬规则的登记补这几条。矩阵的种子加 Triage 工作项时，先核已有各行的 `check` 不数它（本次没有核）。 |

## 3. 逐项回答

1. **R1**：推理成立。Go 的 `keysTo` 对每个通到 `issues`、`projects`、`workspaces` 的外键都要一行（`si/bootstrap/workspace_deletion_catalog_test.go:43-111`），可以用 SQL 种。e2e 的 `expectIssueDeleted` 按目录走外键图，要求每个键在被删的工作项之下有一行随它删除（`issue.ts:457-464`），故事只经接口造数据；Triage 的工作项对 `DELETE /issues` 答 409，所以带条目行又能删的只有已接受的。表、删除图、清理、接受必须同段。新划分是自洽的：P2b 剩列表、内容修改、条目的删除、待处理数与它们的动作，仍是完整的一段，并复用 I4 的接缝。夹具的落点见 I2；期间 main 的状态见 m4。
2. **R2**：等价按构造成立。写 `issues.state_id` 的只有 `InsertIssue` 与 `UpdateIssue` 两条语句（`si/modules/issue/adapter/postgres/queries/issues.sql:14`、`update.sql:8`）；`pickState` 不给 Triage（`ia/writes.go:22-29`），草稿发布走同一个 `check`；状态接口的读、改、删、设默认都排除 Triage（`states.sql:15,48,64,79`），所以它删不掉、也成不了默认状态。三条查询确实共用一种行（`issues.go:156`、`sub_issues.go:38` 的类型转换，编译器会逼三条同步）。左连接的代价是每行一次部分唯一索引的探测，每个写多两次（预检与锁下），可以忽略。有效状态只在两处真正要时钟：`→ pending` 的迁移与回答；守卫与资产只看存的状态是否 `accepted` 或终态，不受到期影响，所以一致。读的时钟见 m1。
3. **新建接缝与 R6**：新建的放置参数可行，草稿发布不受影响（`publish_draft.go:43` 传默认）。修改一侧见 I4。没有近似副本的前提是按 I4 抽出，而不是在分诊里重写状态的写。
4. **开关**：只在锁下判与 issue 自己的先例一致（`readMembership`）。归档先于开关要由用例自己先判（核心里的归档检查在后，`create_core.go:115`），spec 已写。`issue.LockedProject(locked)` 是类型转换（`si/bootstrap/ports.go:183,193`），两边同名同型加一列即可；31 处 `LockedProject{…}` 字面量都是带字段名的，能编译。资产一路读不到开关，见 I5。
5. **分诊**：迁移表与设计一致，`pending → pending` 与终态之后都是 409。字段规则可行；请求体一级的 422 先于 404 是仓库的惯例。次序见 I9。动态的形状与表的列相符（`verb`、`field` 是 `varchar(255)`），不带标识，访客规则的动态过滤不涉及它；契约里 `IssueActivity.verb` 是普通字符串（`issue.yaml:2413-2414`），通知的 `NotificationVerb` 枚举不受影响（分诊不发事件，接受只发状态变化）。CHECK 的写法见 I1；约束名不变，没有测试比对约束的定义文本。
6. **边界**：写路径核了一遍 `Locks.run` 的全部调用处，没有调研漏掉的：描述版本只有读，排序经 `updateIssue`，迭代转移只动迭代成员，提及同步在修改与新建里，草稿的父工作项在发布时经 `inReach`，自动归档只碰终态。源端的数目见 I3。`write.check` 在预检与锁下各跑一次，位置对；它先于 `project.archived`，要登记（m10）。另一端 422、搜索的 `NOT EXISTS`（issue 自己的表，在 sqlc 范围内）、搜索的源 404 都可行。列表靠状态目录的内连接，迭代模块进度与 `sub_issues_count` 靠不变式，成立。
7. **资产**：可行，但缺三样，见 I5。访客提交者：`intake.update` 放行访客，只看自己的访客对别人的条目先得 404，对自己的是提交者。评论的两种宿主不动。
8. **详情**：五个经 `seenOne`（`get_issue.go:40`、`get_issue_by_identifier.go:79`、`update_issue.go:119,157`、`archive_issue.go:82,104`），两个经 `answer`，共七个。细节见 m8、I6。
9. **删除图与清理**：服务端的文件清单与调研一致，缺的是 I7 的测试。e2e 见 I2。
10. **plan**：见 I2、I6、I7、I8、m9。`TestSQLCSchemaScope`、`archtest` 不受影响（`sqlc.yaml` 的 issue 清单加两个迁移即可）。
11. **Plane**：spec 对 Plane 的描述与源码相符：五个数字状态、没有迁移规则、嵌套的请求体、接受到默认状态、拒绝与重复留在 Triage（`app/serializers/intake.py:72-81`）、状态动态 `notification=False`。漏登记的见 m10。

## 4. 核实为正确的事实（摘要）

- `pickState` 是 Triage 进不去的唯一原因，`StateCatalog` 的回答含 Triage（`si/modules/project/adapter/postgres/queries/issue_ports.sql:30-37`）；`CompletedAt` 对 Triage 组为空。
- `Grant.ProjectAdmin` = 项目管理员，或是项目成员的工作区管理员（`si/modules/access/domain/decide.go:85`），与 §15 B 一致；他若在项目里只是访客，`OwnOnly` 仍为真，别人的条目对他是 404。
- `projectIssueTable` 与 e2e 的 `asynchronousDeletionTable` 都按 `issue_` 前缀，`intake_issues` 要显式加（`si/bootstrap/issue_project_deletion_jobs_test.go:26-29`、`e2e/fixtures/assert/catalog.ts:94-111`）。
- 最新迁移是 `00049`；`trigram_schema_test.go` 钉 49、`purge_indexes_test.go` 钉 33。
- 新建核心在 `write` 的最后发布 `issue.created`（`create_core.go:294`），事务内的订阅者不读条目行，DP-2 的次序安全。
- `go build ./...` 在基线上通过。

## 5. 处理

待主控。

## 5. 处理（主控，2026-11-19）

全部采纳，改入 spec 与 plan（与本文件同一个提交）：

- I1：CHECK 改为 `field IS NOT DISTINCT FROM 'intake'`；迁移测试加两条反例，已有的 `commented` 不改。
- I2：e2e 夹具改为新夹具 `furnishAcceptedIntake` 由五处 `expectIssueDeleted` 的故事各调一次，`furnishPlanningRows` 加一次提交；`project.ts`、`workspace.ts` 不改。
- I3：源端守卫改为六个，`deleteIssueRelation` 不加，不变式测试钉“没有关联行”。
- I4：spec §2.7 定形修改核心的接缝 `updateLocked(…, at)`；接受以同一个时刻调它；`intake` 动态写在状态动态之前；T6 加“读了第二次时钟”的变异。
- I5：取预检给的可选项——Triage 宿主的资产判定整段随内容修改移到 P2b（R1、R5），P2a 不改 `asset.yaml`；P2b 之前照 `issue.update`，§1 写明 main 的已知状态。
- I6：网页的两个 `IssueDetail` 夹具与 `issue.store.ts` 的合并（及测试）、`issue_shapes_test` 与详情金样进 §2.1 与允许清单；T9 照 `TestEachDetailAnswerSaysWhetherTheCallerSubscribes` 写七个回答。
- I7：允许清单加 issue 模块里钉表序的六个测试与“编译跟随”一类；§2.10 写死位置。
- I8：动作与规则表提到 T3；红窗口加写锁表的守卫；T1 改关键词的 phase 为 `M7/P2`。
- I9：§2.5 写死拒绝次序（422 → 看不到 404 → 403 → 没有条目行 404 → 归档 → 开关 → 迁移 → 锁下 422），验收加三格与逐值相同。
- m1：`Locks` 加时钟，读的回答读一次，写的回答用写的时刻；动态记有效状态。
- m2：`Intake` 的注释与 `ON` 条件。
- m3：不另锁条目行，用锁下重读的 `h.issue.Intake`。
- m4：§1 加“P2a 合入后、P2b 之前 main 的已知状态”，阶段记录向负责人说明。
- m5：`00051` 的 Down 先删 `intake` 动态再收回 CHECK。
- m6：已归档的重复目标允许，登记。
- m7：`status` 的值 422（domain）、类型 400（bodyshape）；“是自己”在锁下的统一 422。
- m8：提交插条目行后读回；`answer` 不变，只有 `seenOne` 处理重复目标。
- m9：T2、T12 各拆两段，共 17 个 Task。
- m10：T15 的 plane-diff 清单补齐，矩阵种子先核已有 `check`。
