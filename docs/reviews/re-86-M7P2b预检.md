# M7 / P2b 预检

> 2026-11-19：结论 **修正后通过**（Ready after fixes）。没有严重问题；6 处重要、11 处次要。R1（资产判定移 P2c）成立，合同的每一项在 `670399c4` 上都做得出来，与设计的授权没有矛盾。重要问题落在 plan 的 Task 次序、允许清单与 spec 的三处文字上，不需要改设计的取舍。须向负责人说明的只有 spec 已写的一件：R1 再次改已批准设计 §12 的行。
> - **目标**：[P2b spec](../v0/M7-collaboration/specs/P2b-intake-list-edit-delete.md)、[P2b plan](../v0/M7-collaboration/plans/P2b-intake-list-edit-delete.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2（H9、H11）、§3.5、§3.6、§3.10、§3.11、§12、§15，台账 `research/p2b-facts.md`，[P2a 阶段记录](../v0/M7-collaboration/reviews/P2a-intake-submit-triage-review.md)、[re-85](re-85-M7P2a预检.md) 与[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §4、§5。
> - **基线**：main `670399c4`（P2a 的 noff），worktree `codex/m7-p2b`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git、不起容器、不跑测试、不跑 `go build`。对照代码核了 `Locks` 的编排与 `write.check`、修改核心、新建核心、删除用例、分诊与提交、`seenAt` / `intakeSeen`、列表的解析原子与 HTTP 绑定、列表引擎的 SQL 与行的裁剪、条目行的 SQL 与清理、access 的判定、project 的可见项目端口、按 `contract.Operations()` 驱动的全部守卫、删除窗口与空创建者测试、权限矩阵的帮手、契约里可比操作的码。下文路径相对 worktree 根，`si/` = `server/internal/`，`ia/` = `si/modules/issue/app/`，`bs/` = `si/bootstrap/`。

没有核实的：

- 各测试的实际耗时与两路 CI 的余量（没有运行；数字取自 P2a 阶段记录 §6）。
- 权限矩阵已有各行的 `check` 是否受新行影响（P2b 不加全局种子，按构造不受影响，没有逐行读）。
- sqlc 对新 SQL 生成的类型；列表 SQL 在大数据量下的计划（没有执行）。
- `bs/issue_activity_pages_test.go` 等文件里是否还有完整实现 project 端口的假实现（只按方法名搜过，见 I5）。

## 2. 发现

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | plan T9、T10；红窗口一条 | **T9 单独提交编译不过，不是“守卫转红”。** issue 的处理器实现生成的 `gen.StrictServerInterface`；契约加四个操作后 `make gen` 给接口加四个方法，`issue/adapter/http` 与整个 `bootstrap` 包在处理器写好之前不能编译，红窗口里一个 bootstrap 测试都跑不了。P2a 已经踩过并裁定合并。 | `si/modules/issue/adapter/http/handler.go:219`；P2a 阶段记录 `:48`（“处理器与逐码测试并入 T8（生成的服务端接口要求处理器）”），提交 `8d085bec` 是契约、处理器、网页的码一起 | T9、T10 合为一个 Task、一个提交（契约、`api/openapi.yaml`、`make gen`、四个处理器、接线、逐码测试、网页文案）。红窗口改为“写锁表、矩阵覆盖、删除窗口（及遍历它的空创建者测试）：合并后的 T9 → T11”；路由与逐码当场绿。照 P2a `:50` 的裁定，T11 紧跟其后、中间不插别的 Task。 |
| I2 | spec §2.4（“提交者（条目行的创建者）”）、§2.5、§2.3 | **“提交者”在三处用了两个定义，其中一个在 `write.check` 里读不到。** §2.4 说提交者是条目行的创建者；§2.5 用 `AllowCreator`，§2.3 的只看自己用 `i.created_by_id`，这两处都是工作项的创建者。`write.check` 只拿到 `domain.Issue` 与授予；`Issue.Intake`（`IntakeFacts`）只有状态、暂缓到、重复目标，没有条目行的创建者，三条 `GetIssue` 系查询也不读它。两者按构造相等（P2a 提交时同一个 `by`，都 `SET NULL`），但照 §2.4 的字面要给三条查询与 `IntakeFacts` 加一列。 | `si/modules/issue/domain/intake.go:83-87`；`ia/lock.go:75,145-149,206-210`（`check` 的签名与两次调用）、`:269-271`（`AllowCreator` 取 `i.CreatedByID`）；`si/modules/issue/adapter/postgres/queries/issues.sql:80`；`intake.sql:7-9` | §2.4 改为“提交者 = 工作项的创建者（`Issue.CreatedByID`，与条目行的创建者按构造相同，P2a spec §2.4）”，三处同一个定义，不加列。写明创建者为空时：非管理员一律不是提交者（修改 403、删除 403），验收与空创建者测试各加一格（成员对空创建者的条目 PATCH 403、DELETE 403；只看自己的仍是 404）。 |
| I3 | spec §2.3（回答）、plan T5、T6 | **列表行对只看自己的访客的裁剪没有写。** 范围含已接受的工作项，它们是普通工作项：可以有别人建的父、别人建的子。`IntakeIssue` = 列表行 `Issue`，就带 `parent_id`、`sub_issues_count`、`attachment_count`。项目列表靠三样守住访客规则：SQL 里子工作项数只数自己的、`hideParents` 把看不到的父答成 `null`、`projectAttachments`。新的平铺查询不经列表引擎，这三样不会自动跟来；spec 与 plan 只提了“访客规则的 SQL”与重复目标。漏了就是硬规则“列表和计数只算调用者能看到的”“回答里不指名看不到的工作项”的破口。 | `ia/listing.go:165-193`（`issues`：`hideParents`、`projectAttachments`）；`si/modules/issue/adapter/postgres/queries/list_issues.sql:124-125`（子工作项数带 `NOT own_only OR c.created_by_id = actor`）；硬规则 03 §4 访客规则第 4、5 条 | §2.3 写明列表行与项目列表的行同样裁剪：`sub_issues_count` 只数调用者看得到的、看不到的父 `parent_id` 为 `null`、附件数经已有的投影；T6 复用 `hideParents` 与 `projectAttachments`（不重写）。验收加一格：访客提交的条目被接受后，管理员给它挂上自己建的父与子，只看自己的访客列出它时 `parent_id` 为 `null`、子数为 0；变异（去掉 `hideParents`、去掉子数的谓词）各让它失败。 |
| I4 | plan 允许清单（删除窗口一条）、T11；spec §2.8 | **删除窗口的期望不在 `…_cases_test.go` 里，允许清单漏了它所在的文件。** 用例表只列操作；每例的期望在 `issue_deletion_window_test.go`：默认 404，项目删除时只有 `searchIssues`、`listMemberIssues`、`listDrafts` 三个工作区级的读是 200，各带一个 `check`。`getIntakeCounts` 是工作区级的读，项目删除时应答 200 且不含被删的项目，不改这个开关它就按 404 判、确定失败。要让 200 有内容，夹具里留下的项目 `f.q` 还得开着收集箱、有一个待处理的条目（现在没有）。另外 `request()` 把 `{issue_id}` 换成条目的分支只认 `triageIntakeIssue`（plan 的“指向条目”已是这个意思，这里点明位置）。 | `bs/issue_deletion_window_test.go:55-80`；`bs/issue_deletion_window_fixture_test.go:19,34`（`kept` 都是 `f.q` 的普通工作项）；`bs/issue_deletion_window_cases_test.go:92,127-133`；`bs/issue_null_creator_test.go:37,49-56` | 允许清单的删除窗口一条加 `issue_deletion_window_test.go`（`getIntakeCounts` 在项目删除时 200，`check` 钉“只有留下的项目、待处理数为 1”）与 `issue_deletion_window_fixture_test.go`（`f.q` 打开收集箱并提交一个条目）；`request()` 给 `updateIntakeIssue`、`deleteIntakeIssue` 加条目的分支，并照 P2a 的做法各留一个控制变异（指向普通工作项时测不出）。空创建者测试：跳过清单加 `listIntakeIssues`、`getIntakeCounts`，泄漏的目录加收集箱列表与计数（空创建者的条目不出现、不计数）。 |
| I5 | spec §2.2、§4（“已有测试一字不改地通过”）；plan 允许清单的“纯搬移”与 m2 两条 | **允许清单与 spec 的说法对不上，也少一类。** ① 两次搬移都要改已有测试的构造处：`NewUpdateIssue` / `UpdateIssueDeps` / `TriageIntakeIssueDeps{Update:}` 在 6 个测试文件里，`NewDeleteIssue` / `DeleteIssueDeps` 在 2 个里——spec 的“一字不改”不成立，plan 的“只为编译而改”才对。② “编译跟随”只给了 T2、T7 的前半，而 T5–T8 还要给接口加方法：`IntakeWriter` 加条目行的软删，假实现在 `fakes_intake_test.go`；project 的 `IssueProjects` 与 issue 的 `ProjectPorts` 加“开着收集箱的项目”，假实现在 `bs/issue_list_directories_test.go`（P2a 的 I7 就是漏了这一类）。③ m2 写“断言不改”，但提交的单测逐条钉着调用次序，里面有核心回答的那次 `IsSubscribed` 与附件数；m2 去掉它们，期望的调用表必须变短——这正是 m2 的 RED 证据，不是“不改”。 | ① `ia/update_issue_test.go`、`triage_intake_test.go`（两处）、`cycle_membership_test.go`、`description_mentions_test.go`、`issue_events_test.go`、`lock_test.go`；`ia/delete_issue_test.go`、`issue_lifecycle_events_test.go`。② `ia/ports.go:204-209`、`ia/fakes_intake_test.go:29,41`；`si/modules/project/module.go:141-143`、`bs/issue_list_directories_test.go:34-42`。③ `ia/submit_intake_test.go:40,63-66`；`ia/create_core_test.go:83,93` | spec §2.2、§4 改为“已有测试的断言不改，只改构造处”，并列出上面 8 个文件。允许清单加一类（全阶段适用）：“接口加方法、构造函数换依赖带来的假实现与调用处的编译跟随，不改断言，台账逐个记”，点名 `fakes_intake_test.go` 与 `issue_list_directories_test.go`。m2 一条改为“`submit_intake_test.go` 期望的调用表去掉核心回答的两次读（先红后绿）；`create_core_test.go` 只跟签名”。 |
| I6 | spec §2.7（“各操作写明 `x-problem-codes`”） | **四个操作的码没有列出来，而哪些码到得了要先裁定。** 每个声明的码都要有处理器测试回答，组合测试的每个回答也按声明核对；多声明一个到不了的码、少声明一个到得了的码都会红。内容修改经修改核心，核心能答的码比它到得了的多：`issue.archived`（Triage 的工作项归档不了，已接受的先答 `issue.intake_closed`）、四个 `planning.*`（白名单里没有迭代、模块）都到不了；`asset.not_found`、`asset.not_uploaded`（描述图片的绑定）、`server_busy` 到得了；`issue.in_triage` 不该有。spec 是合同，plan 又规定超出 spec 的契约改动要停下报告，所以要在 spec 里定。 | `api/modules/issue.yaml` 里 `updateIssue`、`deleteIssue`、`triageIntakeIssue`、`listIssues`、`listDrafts` 的 `x-problem-codes`；`ia/update_issue.go:112-117,134-141`；`si/platform/httpserver/apitest/problems.go:137,158-160`（`unanswered`） | §2.7 逐操作列码（建议）：列表 `validation_failed, project.not_found, forbidden`；计数 `workspace.not_found, forbidden`；修改 `validation_failed, issue.intake_not_found, forbidden, project.archived, issue.intake_disabled, issue.intake_closed, server_busy, asset.not_found, asset.not_uploaded`；删除 `issue.intake_not_found, forbidden, project.archived, issue.intake_disabled, server_busy`。写一句为什么修改不声明 `issue.archived` 与 `planning.*`。 |

### 次要

| # | 位置 | 发现与证据 | 修正 |
|---|---|---|---|
| m1 | spec §2.1 | 文件表只有 `api/modules/issue.yaml`。两条新路径（`/api/v0/intake-issues/{issue_id}`、`/api/v0/workspaces/{slug}/intake-counts`）要进根文件，否则 `TestRootListsEveryModulePath` 红（`api/openapi.yaml:142-145` 是 P2a 的两条；`si/platform/httpserver/apitest/rules_test.go:53`）。 | §2.1 加 `api/openapi.yaml`。 |
| m2 | spec §2.3（“`status` 可多值”） | 没说多值的写法。已有列表是一个参数里的逗号分隔，重复的参数答 422 `not_allowed`；这靠两处按路由写死的清单：`listSingleQuery`（`si/modules/issue/adapter/http/list_query.go:44-60`）与 `withListQuery` 的操作名单（`:63`，不在名单里的操作拿不到原始查询）。 | 写明“逗号分隔，重复的参数 422，同其他列表”；T10 点名这两处各加新路由。 |
| m3 | spec §2.3、R8 | “按 UTC 的日”落到 SQL 时，`timestamptz::date` 随会话时区。`ParseListDateFilter` 收 `none`（`si/modules/issue/domain/list_date.go:21-23`），spec 已说不收，解析处要自己拒。 | 写明日的范围在 Go 里换成 UTC 的半开时刻区间 `[lo 00:00, hi+1 00:00)` 再给 SQL；adapter 测试钉 23:59:59.999999 与次日 00:00 两条边界行；`none` 422 一格。 |
| m4 | spec §2.6 | ① 设计 §3.5 写“每个看得到、开着收集箱的项目”，spec 收窄为未归档的（Plane 连归档的也数）；取舍合理（同 `Visible`），但设计的字与 plane-diff 要跟上。② “专门端口”若另写一条成员关系的 SQL，就是 `(pm.role = 5 AND NOT p.guest_view_all_features)` 的第三份（`si/modules/project/adapter/postgres/queries/visible_issue_projects.sql:1-12`）。③ 没说项目范围与计数在同一个只读快照里（成员列表、搜索的先例）。 | ① T1 改设计 §3.5 的那句加“未归档”，T15 的 plane-diff 加“归档的项目不列”。② 端口定为“这些项目里哪些开着收集箱”（收 `Visible` 的结果、只读 `intake_view`），不重写成员关系与只看自己；若坚持新 SQL，加一条与 `Visible` 的等价测试。③ 写明一个只读事务。 |
| m5 | spec §2.4、§2.5 的拒绝次序 | 两个操作里“不是提交者”与“没有条目行”的先后相反，是机制决定的：删除的提交者规则在 `AllowCreator`（判定里，先于 `check`），修改的在 `check`（条目行之后）。成员对一个看得到的普通工作项：DELETE 403、PATCH 404。两者都不泄漏（他读得到详情里的 `intake`），与 P2a I9 一致，但没写出来，评审会当成不一致去“修”。另：已接受且已归档的工作项，删除只删条目行、不答 `issue.archived`，没有写。 | 两节各加一句点明这个先后与原因；验收加交叉的两格（成员非提交者 × 有无条目行 × PATCH / DELETE），照硬规则 03 §5“拒绝的次序在两个交叉方向上都钉住”。写明已归档的已接受工作项照删条目行。 |
| m6 | spec §2.4 | “至少给一个字段之外的校验照 `CheckIssuePatch`”读不通。`updateIssue` 收空补丁、按无改动回答（契约没有 `minProperties`，`api/modules/issue.yaml:2370-2374`；`ia/update_issue.go:130-143`）。 | 改为“字段的值照 `CheckIssuePatch`；`{}` 在全部拒绝之后按无改动答 200，同 `updateIssue`”。 |
| m7 | spec §4（n1 列在“组合测试（真实的 app）”下）、plan T12 | n1 的状态只有不变式已破时才到得了（条目未终态而工作项已在默认状态），经接口造不出来，组合测试要用 SQL 把 `state_id` 改掉。 | 写明 n1 的主证据在单测（T3），组合一格用 SQL 造违规状态并注明；或只留单测。 |
| m8 | plan T5、T6、T7、T11、T12 | 变异只在 T8、T13 点了名。合同里这些性质还没有对应的变异：重复目标看不到为 `null`、列表与回答同一个时刻、已接受只删条目行（两个方向：已接受删了工作项；未接受只删了行）、计数的范围（关着的、归档的被列出）、只看自己的计数、锁下再判开关。另 `PATCH` 是第八个回答详情的操作（`bs/issue_intake_answers_test.go:36-102` 是七个），列表行的 `intake` 应与同一工作项详情的 `intake` 逐值相同（同一套三个条件，`ia/seen.go:132-148`）。 | plan 加一张“性质 → 变异 → 所在 Task”的表；T12 加“PATCH 的回答带 `intake` 与 `is_subscribed`”“列表行与详情的 `intake` 相同”两格。 |
| m9 | plan（15 个 Task） | T4（两个解析器、游标、修改请求的校验、关闭的状态、四个动作与规则）、T5（两条 SQL、软删、跨模块的端口与转换）、T12（五组组合测试，P2a 的 m9 就为此拆过）偏大。I1 合并一个之后，拆 T5（issue 的存储 / project 的端口）与 T12（列表与计数 / 修改、删除与 n1）共 16 个，在设计 §12 的约 16 之内。 | 按此拆；检查点 1 放在 T12b。 |
| m10 | plan 测试时间一条 | ① bootstrap 的测试按名字排序后轮流分到两路（`tools/go-test-shard.sh:15-17`）：新加一个顶层测试会让排在它后面的全部换路，一路的增量可以与新测试自己的耗时无关。② `waitForStartupRuns` 只写在全局约束里，没点到 Task。 | ① 检查点 1 记两路各自的增量与新测试自己的耗时，分开写；某一路因换路超过 90 秒时照设计 §9 报告，不靠改名调路。② T13 的每个交错、T12 里删除之后断言行与墓碑的测试点名先等；修改与删除之后做锁检查的先 `waitForFanouts`（两者都发事件）。 |
| m11 | spec §1（P2c 之前 main 的状态） | 描述准确（见 §3 第 6 条），少两句：拒绝、重复的条目在 P2b 之后可以列出、可以删除（P2a 记录 §4 说的“删不掉、列不出来”到此结束）；设计 §3.6 的资产一句与 H11“描述的图片同此”的证据留到 P2c。 | 补这两句；T1 改设计 §12 时把 H11 的图片证据写进 P2c 一行。 |

## 3. 逐项回答

1. **可行性**（都成立）：
   - 列表的解析与游标：原子可以复用（`ListFilterValueAllowed`、`(*ListFilters).parse`、`ParseListDateFilter`、`shared.PageSize`）；`ListOrderAllowed` 没有范围参数（`si/modules/issue/domain/listvocab.go:71-80`），所以 `sequence_id` 走独立的 `ParseIntakeQuery` 与 `IntakeCursor` 是对的。HTTP 绑定见 m2。
   - 有效状态在 SQL：`status = 'pending' OR (status = 'snoozed' AND snoozed_till <= now)` 与 `EffectiveStatus` 的 `!After(now)` 同义（`si/modules/issue/domain/intake.go:71-76`）；业务时钟已截到微秒，Go 与 SQL 不会在边界上分歧。`AsOf` 收一个函数，传那一个时刻即可。
   - 访客规则的 SQL：`NOT own_only OR i.created_by_id = actor` 是登记的写法（`list_issues.sql:24`），创建者为空时不选中。列表的 `own_only` 来自 `Locks.decide` 的授予（同 `ia/list_issues.go:41-48`）。行的裁剪见 I3。
   - 重复目标的批量读：`IssueRefs` 一次读、内部用 `HiddenFrom`、只认授予里的项目（`ia/issue_refs.go:38-45`），与详情的三个条件一致。
   - 修改的相对规则：`check` 是闭包，拿得到补丁、调用者与授予；`ProjectAdmin`、`ProjectRole` 都在授予里（`si/modules/access/domain/decide.go:83-90`）。预检与锁下各跑一次。提交者的定义见 I2。
   - 关闭的 409：`h.issue.Intake.Status` 是锁下重读的事实，只看存的状态，不读时钟。
   - 删除核心：`DeleteIssue.Execute` 的闭包没有自己的守卫（`ia/delete_issue.go:101-123`），源端守卫在 `write.check`，可以原样抽出。已接受只删条目行：清理按条目行自己的 `deleted_at`（`queries/purge.sql:157-161`），部分唯一索引与不变式测试都不受影响，删除与清理的守卫不需要例外。
   - 计数：`seeWorkspace` 现成（`ia/lock.go:249-252`）；项目级的规则在没有项目时判不了，新的工作区级动作是必要的。端口见 m4。
   - m2、m3、n1：`createLocked` 的调用者只有新建、提交与一个单测；`updateLocked` 只有修改与分诊两处；n1 的检查与 m1 同位（`ia/triage_intake.go:176-192`）。测试的跟随见 I5、m7。
2. **拒绝次序**：与硬规则和 P2a 一致。访客规则的 404 在任何 403 之前（`ia/lock.go:276`）；两种 404 是同一个 `ErrIntakeNotFound`；归档先于开关（`refuseIntakeOff`）。两个操作之间的那处先后见 m5。
3. **与设计**：删除的授权（管理员、成员以上的提交者、任何状态）、`intake.update` 的相对规则、访客提交者只改名字与描述，都与 §3.5、§3.6、§3.10 一致；`skip_activity` 是控制位不是字段，访客可给是合理的。§3.10 的规则表除 `intake.count` 一行外不需要改；要跟着改字的是 §3.5 的待处理数一句（m4）与 §12 的两行（R1）。
4. **按目录驱动的守卫**（`contract.Operations()` 的全部使用者）：路由（`bs/contract_test.go`、`asset_contract_test.go`）、401 与参数绑定（自动）、权限矩阵覆盖、写锁表（`issue_draft_write_locks_test.go` 的目录核对）、删除窗口及遍历它的空创建者测试、逐码、根文件的路径清单。`project_write_locks_test.go` 与 `permission_matrix_guests_see_all_test.go` 只看 project、planning 的写，不涉及。spec §2.8 的清单加上 I4、m1 即全。
5. **权限矩阵的 GET 行**：`prepare` 只在 `write` 或 `config` 的行上跑（硬规则与台账已核），`write: true` 的 GET 行有先例；`inTheIntake` 只放本列自己的工作项，“自己的”与“别人的”两列正好分出只看自己（一列见 1 个、一列见 0 个）。预期的格子与 `onAWorkItem`、`onTheIssues` 对得上。
6. **P2b 与 P2c 之间**：安全，描述准确。资产对工作项宿主的写今天按 `issue.update` 判（`ia/asset_access.go:106-109`），没有条目的时段与开关；访客提交者的修改若带图片，因为他准备不了而被拒，不会绑上别人的图片（绑定只认本人为本宿主准备的）。成员提交者在待处理时能带图片，与最终规则一致。
7. **plan**：见 I1、I4、I5、m8–m10。动作与规则同一个提交（T4）吸取了 P2a 的 I8；关键词 phase 已是 `M7/P2`，不用改。

## 4. 核实为正确的事实（摘要）

- 删除窗口现在是读 18、写 26（`bs/issue_deletion_window_cases_test.go:92`）；加两读两写后 20、28，与 spec 相同。
- `NewLocks` 的签名不变，三十多处构造不受影响；`LockedProject`、`shared.Grant`、`VisibleProject` 都不加字段。
- `intake.delete` 的 `AllowCreator`：访客创建者 403，工作区管理员兼项目访客放行但别人的条目先 404（`decide.go:73-76,100-103`）。
- 修改核心不带调用者的守卫（`ia/triage_guard.go:10-24`），`updateIssue` 的源端守卫留在 `Execute` 的 `write.check`（`ia/update_issue.go:71`）。
- 列表不能经列表引擎：它内连接不含 Triage 的状态目录（`list_issues.sql:21`）；设计 §4.1 已定为专门的平铺查询。
- CI 第 0 路 Test 步在 P2a 为 732–791 秒，加 90 秒仍在 960 秒的提醒线以下。

## 5. 处理

待主控。

## 处理（主控，2026-11-19）

全部采纳，改入 spec 与 plan：

- I1：契约与处理器并为一个 Task（T9），守卫的红窗口 T9 → T10。
- I2：提交者定义为工作项的创建者 `Issue.CreatedByID`（R11），空创建者即没有提交者，验收加空创建者的格。
- I3：列表行与项目列表同一套裁剪（R10，复用 `hideParents`、`projectAttachments`、子数只数看得到的），验收加挂父与子的格，变异表列入。
- I4：允许清单加删除窗口的期望与夹具文件、`request()` 的条目分支与控制变异、空创建者的跳过与泄漏目录（spec §2.8）。
- I5：编译跟随一类（纯搬移与接口加方法）、m2 的调用清单作为 RED、n1 的新格写入允许清单；spec 删去“一字不改”。
- I6：spec §2.7 逐操作列码，写明修改不声明 `issue.archived`、`planning.*` 与 `issue.in_triage` 的理由。
- m1–m11：`api/openapi.yaml`；CSV 与按路由的参数清单；日在 Go 里换成半开的 UTC 时刻区间、`none` 422、两条边界；计数经 `Visible` 再读开关、一个只读事务、设计 §3.5 加“未归档”；两个操作的拒绝次序相反的原因与交叉四格、已归档的已接受照删条目行；`{}` 那句改写；n1 以单测为主证据；性质与变异表、第八个详情回答与列表行的 `intake` 相同；T5、T11 各拆两段（16 个 Task）；两路的增量分开记、点名要先等的 Task；P2c 之前 main 的状态补两句。
