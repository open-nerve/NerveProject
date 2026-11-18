# M7 / P1c 预检

> 2026-11-18：结论 **修正后通过**（Ready after fixes）。没有严重问题；8 处重要、9 处次要。都只涉及 spec 与 plan 的文字、Task 的次序与验收。须向负责人说明的只有一件：m1 的 `{all, mentions}` 改名是对已批准设计文字的改动（行为不变），建议在阶段记录里写明。I6 若选“`type` 声明为数组”一路，要改网页客户端的源码，超出 spec §2.1 写的网页范围；推荐的一路不需要。
> - **目标**：[P1c spec](../v0/M7-collaboration/specs/P1c-notification-api.md)、[P1c plan](../v0/M7-collaboration/plans/P1c-notification-api.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2、§3.3、§3.4、§3.10、§5、§6.3、§9、§12、§15，[P1b spec](../v0/M7-collaboration/specs/P1b-notification-core.md) 与[阶段记录](../v0/M7-collaboration/reviews/P1b-notification-core-review.md)，台账 `research/p1c-facts.md` 与[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §4。
> - **基线**：main `1a23f6f8`（P1b 的 noff），worktree `codex/m7-p1c`。

## 1. 方法

一次 fresh 只读审查（`fable`）：不构建、不跑测试、不动 git，只写本文件。对照代码核了通知的表与三条 SQL、分发写行的时刻、工作项删除的事件循环、access 的判定与批量判定、project 的可见项目查询、issue 的列表 SQL 与 `HiddenFrom`、权限矩阵的种子与守卫、契约的参数规则、网页客户端的测试、CI 的分路脚本，以及 Plane 的 `views/notification/base.py`。下文路径相对 worktree 根，`si/` = `server/internal/`。

没有核实的：

- 各测试的实际耗时（没有运行）。
- I1 的死锁没有实际复现，是按两边的取锁次序推出的。
- PostgreSQL 的 `LockRows` 在 `Limit` 之下（被跳过的行不占 `LIMIT`）是按文档与 P1b 的同形语句判断的，没有另跑。

## 2. 发现

### 重要

| # | 发现与证据 | 对 spec / plan 的修正 |
|---|---|---|
| I1 | **全部已读与子树删除会死锁；spec §2.7、R5 的“同一锁序、没有环”不成立。** 删除一个工作项时，`DeleteIssue` 在一个事务里对子树的每个工作项各发一个 `issue.deleted`（`si/modules/issue/app/delete_issue.go:114-119`）。订阅者每个事件调一次 `DeleteIssueNotifications`（`si/modules/notification/app/deletion_events.go:16-24`）。每次调用只在“这一个工作项的通知”之内按 id 升序加锁（`si/modules/notification/adapter/postgres/queries/deletion.sql:8-13`）。所以整个事务的取锁次序是“A 的通知升序，再 B 的通知升序”，不是全局升序：B 的旧通知的 id 常小于 A 的。全部已读按收件人的全部未读行全局升序加锁。同一个收件人在父、子工作项上都有未读通知时：全部已读锁住 b（小）后等 a；删除已持有 a，再去锁 b。两边互等，PostgreSQL 1 秒后以 40P01 中止其一，表现为全部已读或删除工作项的 500。plan T12 只测了“全部已读与项目删除的分批软删”，那一对确实没有环，测不到这个。 | spec §2.7 的批改为 `FOR NO KEY UPDATE SKIP LOCKED`（P1b spec §5 已把它列为可选）。全部已读从不等锁，也就没有环。被跳过的行只有两种：正被软删的（不必标），正被单条修改或另一次全部已读持有的（毫秒级，写明“不保证标到”）。R5 与语句头的不变式改写为“全部已读不等待任何行锁”。T12 改为：子树删除持有通知行时全部已读立即返回、标了其余的、`marked` 不含被持有的行、没有 40P01；变异“去掉 `SKIP LOCKED`”要让这个测试失败。不选这一路的话，就得改 P1b 的删除（一条语句按 id 升序锁住整个子树的通知），那是改已合入的阶段，不推荐。 |
| I2 | **`created_at` 是事件的时刻，不是写出的时刻；§2.7 的“之后才写出的不标”做不到。** 分发写行时 `created_at = sqlc.arg(at)`，即事件的时刻（`si/modules/notification/adapter/postgres/queries/notifications.sql:9-13`；`app/ports.go` 的 `Notifications.At` 注明“the event's moment”）。行的 id 是分发运行时生成的 `uuid.NewV7()`（`si/modules/notification/app/fanout.go:112`）。所以事件早于请求、分发迟到的行，`created_at <= now` 成立、id 又大于已处理的批，会被后面的批标为已读，用户没有见过它。T12 的“全部已读与分发写出新行（新行不标）”按字面写不出稳定的测试。同一事实也影响列表：迟到的行按事件时刻插在已翻过的页中间。§2.3 的“页与页之间都在同一个只读快照里”也不成立，页是不同的请求。 | §2.7 改为如实的口径：“只标事件时刻不晚于 `now` 的行；事件早于 `now` 而分发在批循环结束前才写出的行可能被标（与用户点击前一刻才到的通知同一类，登记为已知性质）”。T12 的这一例改为“请求开始之后发生的事件（时刻晚于 `now`）不标”，另加一例钉住迟到的行的行为（标或不标都可以，但要写死）。§2.3 改为“一次请求之内可见的集合与行在同一个只读快照里取；跨页是键集分页，迟到的分发可能落在已翻过的位置”，并交给 P4（不能靠第一页的游标判断有没有新通知）。 |
| I3 | **检查点 1 设在守卫还红的位置。** T6 加契约之后有三处守卫变红：`TestAPIRoutesAreTheContractsOperations`（`si/bootstrap/contract_test.go:47`，要到 T8 接线）、`TestEveryModuleRunsItsHTTPTestsThroughMain` 与 `apitest.Main` 的逐码完整性（T8）、`TestThePermissionMatrixCoversEveryOperation`（`si/bootstrap/permission_matrix_coverage_test.go:145`，要到 T11）。检查点 1 在 T10，要求“七项本地门禁、分支 CI”，那时矩阵覆盖守卫必然是红的。plan 也没有写红的窗口（re-83 I7 的同一类问题）。 | 权限矩阵提到接线之后、组合测试之前：T6 契约与动作 → T7 用例 → T8 处理器与接线 → 新 T9 权限矩阵（覆盖守卫转绿）→ 组合测试 → 检查点 1。plan 写明红的窗口：路由与逐码守卫 T6→T8，矩阵覆盖守卫 T6→新 T9，其间这几项不作为证据。 |
| I4 | **权限矩阵里单条修改与列表的格子按 spec 的写法填不出来。** 矩阵的工作区三列里，管理员与成员没有有效的项目成员资格：`callerAdmin` 不在 `matrixProjectMembers`，`callerMember` 在私有项目的那一行由 `partingStates` 删除（`si/bootstrap/permission_matrix_seeded_test.go:81-94`；`permission_matrix_search_test.go:15-16` 的注释与 `permission_matrix_drafts_test.go:41-43` 的 `prepareDraftPublisher` 都写了这一点）。按 §2.5，他们一个项目也读不到：单条修改得到 404 `notification.not_found` 而不是 200，列表为空，全部已读 `marked = 0`。访客列在 acme/public 是只看自己（`callerGuest` 是它的访客），种的行若 `issue_created_by_id` 不是他，也是 404。§2.8 没有给单条修改的格子，只说“每列一条种好的通知”。 | §2.8 写出格子。单条修改两行：(a) 带 `prepare`（`prepareDraftPublisher` 的先例，在写格自己的库上给管理员、成员补 acme/public 的成员资格）：200、200、200、404、404、404；访客的行种在他读得到的工作项上（acme/private，或 acme/public 且 `issue_created_by_id` 是他）。(b) 不带 `prepare` 的变体：管理员、成员 404 `notification.not_found`，正是“工作区管理员不是项目成员读不到”的一格。列表与全部已读的行加 `check`，断言各列的条数与 `marked`（否则 200 的格子没有区分度）。种子的位置：gone 的那一行要在删除 gone 之前插入（草稿循环的位置，`permission_matrix_test.go:326-337`），由 `waitWorkspaceDeletion` 等它软删。行只用 `workspaceColumns`：非 GET 的行一旦用了项目级的列，`writesOnAProject`（`si/bootstrap/project_write_locks_test.go:134-150`）会把它当作“项目上的写”，要求它锁项目行。 |
| I5 | **单条修改的流程缺三处，其中一处会透露存在。** (1) 地址里没有工作区，工作区要从行读出，再授权、再取可见的集合，然后才能锁；§2.7 只写“先锁这一行（调用者的、未删除的、可见的）”，没有写锁之前的读。(2) 次序：若先按行的工作区 `Authorize` 再比收件人，别人的通知在“调用者是成员的工作区”与“不是成员的工作区”可能走不同的路；422 `must_be_future` 若先于 404，也会透露这个 id 存在（硬规则 03 §4：“422 的位置都不透露”）。(3) 单行上的只看自己不需要 SQL：行已在手里，可以直接问 P1b 的 `GuestRule.HiddenFrom`（`si/modules/notification/app/ports.go:136-143`），少一处谓词的副本。另外没有写用哪个事务：普通的 `tx` 没有锁超时（`si/bootstrap/app.go:129`），单条修改等工作项删除的事务时会等到请求超时；§2.4 的码里没有 `server_busy`。 | §2.7 写明次序：① 不加锁按 `(id, receiver_id = 调用者, deleted_at IS NULL)` 读行，没有即 404；② 以行的工作区 `Authorize(notification.write)`，`ErrNotVisible` 即同一个 404；③ 取可见的集合，项目不在其中、或 `GuestRule.HiddenFrom(issue_created_by_id, 调用者, own_only)` 为真即 404；④ 开事务，`FOR NO KEY UPDATE` 锁行并复核收件人与未删除（没有即 404）；⑤ 读时钟；⑥ 校验（此时才可能 422）；⑦ 写或不写。验收加：别人的通知配过去的 `snoozed_till` 回答 404 而不是 422；四种 404（别人的、已删除的、别的工作区的、项目读不到的）与随机 id 的回答逐字节相同（体、头、码）。事务二选一写死：带锁超时并经 `busyTx` 回答 503（则声明 `server_busy`），或写明“等待以被等的事务为界”的理由。 |
| I6 | **查询参数的契约类型没有定，其中一种写法会打破网页的测试。** (1) `type` 若声明为 `explode: false` 的数组，`web/packages/api-client/test/client.test.mjs:192-232` 对每个 GET 的每个数组参数断言客户端按契约的 `explode` 发出；而客户端只对两个地址用 CSV 序列化（`web/packages/api-client/src/index.ts:34-43`），通知的地址不在其中，测试会红。修它要改网页源码，不在 plan 的允许清单里，也超出 §2.1“网页只有生成的类型与码”。(2) `read`、`archived`、`snoozed` 若声明为 `boolean`，非法值由绑定回答 400 `bad_request`（`TestParametersThatDoNotBindAnswer400`，`si/bootstrap/contract_test.go:211-237`；`apitest/operations.go:91-92`），与 §2.3 的“参数不合法 422”矛盾。issue 列表的 `archived` 为此声明为字符串（`api/modules/issue.yaml:509-512`）。(3) 未知参数、重复的标量没有写：现有的列表两种口径都有（动态 400，草稿与工作项列表 422，`issue.yaml:23,167,394`）。(4) §2.4 没有逐操作列 `x-problem-codes`；`forbidden` 对三种角色都放行的操作不可达（`workspace_members.role` 的 CHECK 只有三个值），先例两种都有（`getWorkspacePreferences` 不声明，`searchIssues` 声明）。 | spec §2.3、§2.4 写死：`tab`、`type`、`read`、`archived`、`snoozed` 都声明为字符串，由 domain 解析，非法值、重复的 `type` 成员 422（`type` 用逗号分隔的字符串，客户端不必改）；未知与重复的参数照草稿列表的口径 422；逐操作列出码。建议：列表 `[workspace.not_found, validation_failed]`，未读数 `[workspace.not_found]`，全部已读 `[workspace.not_found, validation_failed]`，单条 `[notification.not_found, validation_failed]`，`forbidden` 照 `getWorkspacePreferences` 不声明（或都声明并各有处理器测试，二选一）。全部已读的请求体里同名字段可以是真正的布尔与枚举，由 bodyshape 把关（400）。 |
| I7 | **“哪些变化指向别的工作项”是按名单隐去、名单之外放行，新增一种关联就会漏出 id。** 现状与 spec §2.6 一致：`parent` 只留 id（`si/modules/notification/domain/changes.go:174-182`），关联的字段是请求端读到的类型、另一端的 id 在 `new_id` 或 `old_id`（`changes.go:214-235`），类型只有四种（`si/modules/issue/domain/relation.go:23-31`）。但回答对其余字段的 `old_id`/`new_id` 照存：以后 issue 加一种关联类型，通知的名单没跟上，另一端的 id 就原样给了看不到它的人。 | §2.4、§2.6 改为白名单放行：只有 `assignees`、`labels`、`state` 的 `old_id`/`new_id` 照存（`state` 的变化也带 id，spec 只写了负责人与标签）；`parent` 与关联的四种经 `IssueRefs` 解析；其余字段的 id 一律不给。加一个 bootstrap 测试：issue 的每一种关联类型都在通知的“工作项引用字段”集合里。变异：从集合里去掉一种，测试要红。 |
| I8 | **R1 的等价测试在矩阵的原始种子上杀不掉 plan 自己列的变异；plan 引了不存在的 `access.Provide`。** 种子里唯一已删除的项目是 gone 的，它的工作区也删了，工作区成员资格已无效，`Decide` 不看项目就拒绝（`si/modules/access/domain/decide.go:83-85`）。所以“把已删除的项目算进来”的变异在原始种子上不会让任何格子不同。P1b 的等价测试为此有第二遍，用 SQL 造出请求单独造不出的状态：项目单独删除而成员行还在、工作区成员行删除而项目成员行还在、撤权的人的项目成员行又变为有效（`si/bootstrap/access_decide_many_test.go:80-93`）。另外 access 没有 `Provide`，只有 `New`、`NewDecider`（`si/modules/access/module.go:28-52`）。 | spec §2.5、plan T2：等价测试照 `TestDecideManyIsAuthorizeOnTheMatrixSeed` 跑两遍（第二遍的造状态抽成共用的辅助），并要求每一遍都有“放行、只看自己、拒绝”三种答案。写明批量读的范围（二选一）：只读调用者有有效成员资格的未删除项目（`issue.read` 没有成员资格就不放行，`decide.go:99-101`；此时 `ReadableProjects` 只接受不放行非成员的规则，照 `DecideMany` 拒绝 `AllowCreator` 的写法），或读工作区全部未删除的项目。plan 的“`access.Provide` 交出”改为“`access.NewReadableProjects(deps)`，照 `NewDecider`”。 |

### 次要

| # | 发现与证据 | 修正 |
|---|---|---|
| m1 | `{all, mentions}` 的改名有依据：Plane 的 `total` 不含提及（`plane/apps/api/plane/app/views/notification/base.py:201-234`），网页也是有提及就显示提及数、否则显示另一个数（`web/apps/web/core/components/navigation/top-navigation-root.tsx:28-31`），`all` 与 `tab=all` 同名。但它改的是负责人批准的设计文字（§3.4、§5、§15 C）。 | 设计 §14 的裁定里写一行；阶段记录向负责人说明“只改名字，行为照 Plane”；H7 的“总数”改为“‘全部’页签的未读数”。 |
| m2 | R2 的做法可以接受，没有更好的设计：分页与计数必须在 SQL 里过滤，通知的 sqlc 只看得到自己的表。issue 的先例有出处：M4 设计 3.4 写明“列表和搜索：同一条规则，SQL 按各项目 `OwnOnly` 加 `created_by_id = $actor`”（`docs/v0/M4-issue-core/M4-design.md:215`），`list_issues.sql:24`、`search_issues.sql:23` 等如此。`issue_created_by_id` 为空时 `NOT own_only OR … = 调用者` 得 NULL、不选中，与 `HiddenFrom` 对未知创建者一致（`si/modules/issue/domain/issue.go:106-108`）。不足：这是 issue 之外第一处副本，而 P1b 的整分支审查刚删过一处（阶段记录 §9 I1）。 | R2 引 M4 设计 3.4 作依据；T14 在硬规则 03 §4 的访客规则下加一句“通知的列表、未读数、全部已读三条 SQL 是登记的例外，由六格等价测试钉住”。三条语句各一个变异（各去掉过滤）。单条修改不写 SQL 副本（I5）。 |
| m3 | 时间预算的口径不清。第 0 路的 Test 步除了 bootstrap 的一半，还跑 bootstrap 以外的全部包与 tools（`tools/go-test-shard.sh:27-31`），通知的 adapter、处理器、access、project、issue 的新测试都只加在第 0 路。台账基线取的是本机的 523.3 / 500.0 秒，CI 是 749–780 / 517–564 秒，约 1.5 倍。若 90 秒是本机数，CI 上约 135 秒，第 0 路到约 915 秒，离 960 只剩约 45 秒。按名字轮流分路，新增测试会让已有测试换路。矩阵本身便宜：新增约 12–24 个写格，现有约 250 行写。 | spec §4、plan 写明：预算是 CI 第 0 路 Test 步的总增量 ≤ 90 秒（含非 bootstrap 的包），折合本机约 60 秒；两个检查点记本机两路与分支 CI 两路的实测。若 P1c 之后第 0 路超过 870 秒，在阶段记录里向负责人提出加一路（设计 §9），不等 P2a。 |
| m4 | T9 过大：H2–H4 的列出、H5 的全部筛选与分页、七种可见、三种另一端，再加 R2 的六格乘三条 SQL。 | 拆为 T9a（H2–H5：列出、筛选、分页）与 T9b（可见、另一端、R2 等价）。连同 I3 的调序共 15 个 Task。 |
| m5 | 允许清单漏一处，另有一处应写明不发生。`si/bootstrap/notification_deletion_test.go:38` 直接写 `notification.New(notification.Deps{…})`，不经 `notificationDeps`，`New` 若对新字段判空它会坏。P1c 不应加迁移或索引（`migrations/notification_schema_test.go` 钉住索引名，不在清单里）。 | 清单加这一处，改法为“补新字段或改经 `notificationDeps`”；Global Constraints 加“P1c 不加迁移与索引；要加先停下报告”。 |
| m6 | 另一端的项目标识，§2.6 写“单行读或一次按工作区的读”，没有定。一页最多 50 条，逐项目单行读是 N+1；P1b 的审查 M2 又刚去掉“读整个工作区的项目”。 | 定为一页一次：`IssueRefs` 回答里带 `ProjectID`，bootstrap 对去重后的项目 id 做一次批量读（project 加一条按 id 数组读标识的查询），页里没有引用时不读。 |
| m7 | 全部已读的时刻在请求开始时读，与“在最后一把锁之后读时钟”（03 §4 第 8 条）不同。理由成立（它同时是界，各批同一个时刻），但 spec 没有写。批的 `ORDER BY id` 没有索引可用（未读索引只有 `(receiver_id, workspace_id)`，`migrations/sql/00049_notification_notifications.sql:50-51`），每批对剩余未读排序一次；`archived=true` 时不走未读索引。上万条时每批是毫秒到十毫秒级，可以接受。 | R5 加一句理由。T5 的 adapter 测试记一次 1 万条未读时单批的耗时，不设门槛。 |
| m8 | 文字：§2.7 的“空的请求体合法”应为“`{}` 合法”（请求体必须是对象，没有字节是 400）；`read`、`archived` 给 `null` 由 bodyshape 回答 400，应写明并有处理器测试。同一事件的几条通知 `created_at` 相同、id 是随机的 v7，彼此的先后不按 `change_seq`。 | 照改；最后一条写一句“同一事件的几条之间次序稳定但不保证按变化的位置”。 |
| m9 | plane-diff 的清单（T14）少几条与 Plane 的差异：Plane 有 `DELETE <pk>/` 与 `GET <pk>/`（`plane/apps/api/plane/app/urls/notification.py:16-46`），Nerve 没有；Plane 的基础排序是 `snoozed_till, -created_at`（`base.py:77`），不给 `per_page` 与 `cursor` 时不分页；Plane 的未读数要求 `snoozed_till IS NULL`，Nerve 计入已到期的；回答里没有 `triggered_by_details`、`is_inbox_issue`、`is_mentioned_notification`。另外通知的两个写不在 `Locks.run` 之内，03 §4 应像 P1a 的关注那样登记，并写下 `notifications` 各个写者的锁序（分发只插入、两种软删、单条、全部已读、清理）。 | T14 的清单补这几条；03 §4 加一段。 |

## 3. 逐项回答

1. **设计的 P1c 一行与 H2–H7**：都有交付。没有越界：没有网页、保留期、单条读取，没有新表与迁移。缺的是 I4（H19 在矩阵里的格子）与 I6（契约的细节）。
2. **R1**：语义对。`issue.read` 是项目级、三种角色（`si/modules/access/domain/rules.go:140`），放行要求有效的项目成员资格（`decide.go:99-101`）。已归档的项目算数（`decide.go:30` 的注释，`project/adapter/postgres/queries/access.sql:1-8` 只排除已删除的）。只看自己按项目角色，所以工作区管理员兼项目访客是只看自己（`decide.go:86-89`）。批量读在 project 的 sqlc 范围内可行：`projects` 与 `project_members` 都是它的表，`visible_issue_projects.sql` 去掉 `archived_at IS NULL` 再带上 `network`、`guest_view_all_features` 就是。可以与 `DecideMany` 一样共用 `domain.Decide`：它是纯函数。等价测试有意义，但要补第二遍（I8）。
3. **R2**：见 m2。六格测试足以钉住取舍，前提是三条 SQL 各有变异、单条修改走端口。
4. **不透露**：过滤在 SQL 的 `LIMIT` 之前，`next_cursor` 取自这一页最后一条可见的行，没有总数，所以次序与游标不透露隐藏的行；未读数与 `marked` 只计可见的。另一端读不到时两对都是 `null`，但放行的名单要改为白名单（I7）。404 的逐值相同要靠 I5 的次序保证。
5. **全部已读**：锁序见 I1，界见 I2。每批一个事务、部分失败不回滚是合理的（重试幂等，§6 已写）。锁下复核由 READ COMMITTED 下带锁子查询对新版本重新求值完成，条件（未读、未删除）必须写在带锁的子查询里，照 `DeleteProjectNotifications` 的形状；T5 的“批的跳过”测试对应它。
6. **单条修改**：三态解码照 `given` 的先例可行（`read`、`archived` 是 `*bool`，`snoozed_till` 是 `nullable.Nullable[time.Time]`）。`must_be_future` 截到微秒后比较，照 `CheckAPIToken`（`si/modules/identity/domain/api_token.go:84-104`），业务时钟本身也截到微秒，一致。幂等与“没变不写”的规则清楚。只锁通知行、不进写闸门与仓库的写编排不冲突：03 §4 的编排管的是工作项的写，通知行不决定任何别的写的判定，但要登记（m9）。时钟在锁之后读是对的；流程的缺口见 I5。
7. **回答的形状**：改名见 m1。`old_issue`/`new_issue` 的四个字段够 P4 的卡片用。引用字段经核实是 `parent` 加四种关联（I7）。演员只给 id 与动态、评论一致（`api/modules/issue.yaml` 的 `actor_id`）。
8. **权限矩阵**：`workspaceOfRow` 加一张按列命名的 `notifications` 表可行，`{notification_id}` 走 `_id}` 的通用分支（`si/bootstrap/permission_matrix_targets_test.go:144-149`）。格子见 I4，预算见 m3。
9. **plan**：次序见 I3，大小见 m4，清单见 I6（网页客户端）、m5。变异的清单基本完整，补 I1、I7、m2 的几个。检查点调序后两个够用。`TestSQLCSchemaScope` 不受影响；关键词工具没有会命中新文件的规则，phase 不变。
10. **Plane**：spec 对 Plane 行为的描述与源码相符（两个页签互斥、未读数的两个数、全部已读连提及一起标、三个状态独立、三种角色）。漏登记的差异见 m9。

## 4. 核实为正确的事实（摘要）

- 通知表、四个索引与审计列如 spec 与台账所述（`server/migrations/sql/00049_notification_notifications.sql`）；`updated_by_id` 是到 `users` 的外键，收件人的写取账户行的 `FOR KEY SHARE`，不与别的写冲突。
- P1b 的两条软删语句都是“带锁的子查询按 id 升序，再更新”（`deletion.sql`）；分批的那条每批一个事务、全局升序，与全部已读没有环。有环的只有 I1 的子树删除。
- `GuestRule.HiddenFrom` 由 issue 交出（`si/modules/issue/module_notification_issues.go:47`），等价测试可以直接问它。
- `GetIssues` 读未删除的工作项、含已归档的、不看可见（`si/modules/issue/app/ports.go:141-143`）；`IssueRefs` 在它之上按授予的项目与 `HiddenFrom` 过滤是可行的。关联列表隐去已归档的另一端（`list_relations.go:63-67`），R4 与它不同是有意的。
- `listDrafts` 是“按 slug 找工作区、工作区级授权、同一个只读快照”的先例（`si/modules/issue/app/draft_reads.go:26-75`）。
- 三种角色都放行的工作区级规则有先例（`rules.go:64,155`）；`notification` 现在在 `actionlessModules`（`si/bootstrap/actions_test.go:46`）。
- 分路脚本按测试名排序轮流分配，其他包只在第 0 路（`tools/go-test-shard.sh`）；作业期限 25 分钟、单次 `go test` 20 分钟（`.github/workflows/ci.yml:23`）。

## 5. 处理（主控，2026-11-18）

全部采纳，改入 spec 与 plan（与本文件同一个提交）：

- I1：全部已读的批改为 `FOR NO KEY UPDATE SKIP LOCKED`，从不等待行锁；R5 改写，T12 改为子树删除持有通知行时立即返回、没有 40P01，变异去掉 `SKIP LOCKED`。
- I2：§2.7 的界改为“事件时刻不晚于 `now`”，迟到的行登记为已知性质、测试写死；§2.3 改为“一次请求之内同一快照，跨页是键集分页”，交给 P4。
- I3：plan 调序为 契约 → 用例 → 处理器与接线 → 权限矩阵 → 组合测试 → 检查点 1，写明红的窗口。
- I4：§2.8 写出格子：单条修改带与不带 `prepare` 两行、列表等三个操作加 `check`，访客的种子行、gone 的插入位置、只用 `workspaceColumns`。
- I5：§2.7 写死七步次序（404 先于 422），单行的只看自己走 `GuestRule`，事务带 `issue.write_lock_timeout`、超时 503 `server_busy`（声明并测试）。
- I6：五个筛选参数都声明为字符串，`type` 逗号分隔；未知与重复参数 422；逐操作列出码，`forbidden` 不声明。
- I7：回答里的 id 改为白名单（`assignees`、`labels`、`state`），加“关联类型全在引用集合里”的测试与变异。
- I8：等价测试两遍、共用造状态的辅助、每遍三种答案；`NewReadableProjects`；批量读只读有有效成员资格的未删除项目，只接受不放行非成员的规则。
- m1：R3 写明只改名字，阶段记录向负责人说明；T14 改设计 §3.4、§5 与 H7 的措辞。
- m2：R2 引 M4 设计 3.4；三条语句各一个变异；T14 在硬规则 03 §4 登记例外。
- m3：预算改为 CI 第 0 路 Test 步的总增量 ≤ 90 秒（本机约 60 秒），超过 870 秒向负责人提出加一路。
- m4：T10 拆为 T10a、T10b，共 15 个 Task。
- m5：允许清单补 `notification_deletion_test.go` 的直接构造；不加迁移与索引。
- m6：另一端的项目标识一页一次批量读。
- m7：R5 写明全部已读在请求开始时读时钟的理由；T5 记 1 万条未读时单批的耗时。
- m8：`{}` 合法、`null` 由 bodyshape 400、同一事件几条之间的次序，照改。
- m9：T14 的 plane-diff 清单补齐；硬规则 03 §4 登记 `notifications` 的写者与锁序。
