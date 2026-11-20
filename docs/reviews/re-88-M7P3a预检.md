# M7 / P3a 预检

> 2026-11-20：结论 **修正后通过**（Ready after fixes）。没有严重问题；6 处重要、12 处次要。合同的每一项在 `7dfaa8a1` 上都做得出来：列表引擎不用改，`WorkspaceList` 只是词表、解析与一个用例；R1 的再划分与依赖方向成立，两段之间 main 的样子写得对。重要问题落在：HTTP 一层有第二张按操作的名单没有写进合同（漏了它筛选会被静默丢掉）、规模的通过线没有数、交接点名的“只读快照”没有验收格、T4 → T5 的红窗口没有声明、变异没有成表且缺拒绝次序的交叉格、三条交接在再划分之后的归属没有分清。不需要改设计的取舍，没有要负责人新定的事。
> - **目标**：[P3a spec](../v0/M7-collaboration/specs/P3a-workspace-list.md)、[P3a plan](../v0/M7-collaboration/plans/P3a-workspace-list.md)（未提交的草稿），依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §2（H13）、§3.7、§3.10、§4.1、§5、§9、§10、§12、§15，台账 `research/p3a-facts.md`（DP-1–26），交接 [M4-closeout](../v0/M7-collaboration/handoffs/M4-closeout.md) M7-03、[M2-closeout](../v0/M7-collaboration/handoffs/M2-closeout.md) §1、[M6-closeout](../v0/M7-collaboration/handoffs/M6-closeout.md) M6-M7-03，[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §3–§5。
> - **基线**：main `7dfaa8a1`（P2c 的 noff），worktree `codex/m7-p3a`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git、不起容器、不跑测试。跑过的只有 `go test -list . ./internal/bootstrap`（编译、列名字，不运行测试），用来核换路。对照代码核了词表与解析（`listvocab.go`、`list_query.go`、`list_filters.go`）、成员列表的用例与引擎（`list_member_issues.go`、`listing.go`）、平铺的 SQL 与 adapter、`Visible` 的 SQL、HTTP 的绑定与两张名单（`list_query.go`）、处理器测试、契约里几个工作区级读的码、access 的规则与判定、`contract.Operations()` 的全部使用者、删除窗口与空创建者、权限矩阵的覆盖守卫、列表的快照测试、`TestIssueListScale` 的种子与期望、分路脚本、网页客户端与它的契约测试。下文路径相对 worktree 根，`si/` = `server/internal/`，`id/` = `si/modules/issue/domain/`，`ia/` = `si/modules/issue/app/`，`ih/` = `si/modules/issue/adapter/http/`，`bs/` = `si/bootstrap/`。

没有核实的：

- 任何测试的实际耗时、换路的 7 个测试各自多重、两路 CI 与 e2e 两片的余量（没有运行）。
- 规模的任何数字（没有跑 `NERVE_LIST_SCALE=1`）；下文的数字都取自 M4 的记录。
- 调研 §4 写的 Plane 的行为（没有读 `plane/`）。
- `issueWorld` 与删除窗口夹具里各人的项目成员关系细节（T5、T6 的期望集合要实施时按种子定）。
- `make gen` 之后 `schema.gen.ts` 的差异是否触到网页的其他测试（只核了 `client.test.mjs`）。

## 2. 发现

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §2.1（`adapter/http` 一行）、§4；plan T4、允许清单 | **HTTP 一层有两张按操作的名单，合同只写了一张；漏掉另一张，筛选被静默丢掉。** `listSingleQuery` 按路由列单值参数（合同写了）。另一张是 `withListQuery`：只有名单里的操作 id 才把原始查询放进 context；不在名单里，`rawListQuery(ctx)` 是 nil，处理器照样答 200——没有筛选、默认次序。四个默认视图里“分配给我”“我创建的”“我关注的”就会各答全部工作项，正是那段注释防的“丢了筛选、执行更宽的查询”。T4 没有列任何变异；两张名单各有一个已有测试按名字枚举（`TestListQueryCopy` 的循环、`TestListBindingErrorIsLocal` 的表），都不在允许清单里（清单那一条写的是代码，不是测试）。`listSingleQuery` 现在的 `switch` 把 `group_by`、`group_key` 与项目、成员共用一支，工作区路由要自己的一支（它不声明这两个参数）。 | `ih/list_query.go:72-75`（操作 id 的名单）、`:93-96`（nil）、`:52-70`；`ih/list_member_issues.go:10`；`ih/list_query_test.go:24`（循环）、`:41` 起（表）；硬规则 03 §5“允许清单”与“每个性质一个变异” | spec §2.1 的 `adapter/http` 一行写成“处理器；`withListQuery` 的操作名单；按路由的单值参数清单”。T4 加变异：新操作不在 `withListQuery` 名单里（带筛选的请求，假用例要收到原始查询）；新路由不在 `listSingleQuery` 里（重复的已声明参数答 400 而不是 422）；`csvListPaths` 不加（`client.test.mjs` 红）。允许清单加：`TestListQueryCopy` 的循环加一个名字、`TestListBindingErrorIsLocal` 的表加行，不改已有行。 |
| I2 | spec §2.4；plan T7 | **规模的通过线没有数，“停下上报”的条件因此不可判定；两个点名的场景照现在的种子是空的。** ①“与项目列表在同等总量上的实测同级”没有说比什么（模块时间还是列表 SQL 的中位）、在哪个规模上判（目标是 2 万，10 万是验证、不承诺）、差多少算同级。`TestIssueListScale` 本身不断言时间，只记录；M4 的记录还明写了“普通 Visible 跨项目连接可能扫描整个工作区”“成员分配 / 订阅相对旧实现 +87.2% / +83.2%”都交 M8，所以“同级”不给数就会在量出来之后才争。②种子里负责人、关注者只有 alice、bob，请求固定以 walt 的身份发；`assignee_id=me`、`subscriber_id=me` 对 walt 匹配 0 行，撞上 `empty performance scenario` 的 `t.Fatal`。③场景的路由靠 `Project` 一个整数分（≥0 项目、<0 成员），工作区场景要第三种；期望集合的范围（`r.Project > 2` 排除、第 2 个项目只看自己）可以与成员场景共用，但要写明。④现有种子是“Web 一个项目 N 个根、另外四个各 200”，walt 的工作区范围 ≈ N + 300 行，与项目 0 的 `default` 场景几乎同量——所以“同一次运行里的同等总量”现成就有对照，均匀分布的子种子是补充，不是对照的前提。 | `bs/issue_list_scale_test.go:80-158`（无时间断言）、`:32-38`（路由与 walt）、`:108-110`；`bs/issue_list_scale_oracle_test.go:50-51`（负责人、关注者）、`:105-110,136-152`；`bs/issue_list_scale_seed_test.go:33-55`；M4 设计 `docs/v0/M4-issue-core/M4-design.md:639`（目标：每项目约 2 万）、`:641`（10 万：默认 / `label_name` 列表 SQL 中位 105.53 / 231.55 ms）、`:642`（跨项目扫描交 M8）；`docs/v0/M4-issue-core/specs/P5-issue-list.md:131-140`（10k 默认 11.34 ms、成员分配 6.65 / 96.59 ms）；设计 `M7-design.md:256,265` | spec §2.4 在量之前写下：比的量（建议：列表 SQL 的中位，1 次预热 + 5 次，与 M4 同法）；判定的规模（建议 20000，10 万只记录）；成对的对照（工作区 `default` ↔ 项目 `default`、`label_name` ↔ `label_name`、按负责人 / 关注者 ↔ `member_assigned` / `member_subscribed`）与倍数上限（主控定一个数，如 1.5 倍），外加 EXPLAIN 的计划形状与对照同类。场景写成 `assignee_id=<bob>`、`subscriber_id=<bob>`（`me` 在解析时就换成 id，SQL 相同），或给请求换身份，二选一写明。T7 写明场景怎么选路由、期望的范围、子种子的项目数与角色，以及已有 13 个场景的期望不变。 |
| I3 | spec §2.2（“一个只读快照、时刻读一次”）、§4；plan T3 | **交接点名的只读快照没有验收格，也没有变异；“时刻读一次”这句代码里没有对应物。** M4 交接 M7-03 的原话是“保持只读 REPEATABLE READ 快照”。两个已有的列表各有一条快照测试（用例跑在核对隔离级别与只读的事务里、数事务次数、带游标的空页也在同一个快照里取总数）；spec §4 的单测、组合测试都没有工作区列表的这一格，T3 的变异只有“范围用了别的端口、动作错”。另外列表引擎不读时钟（成员列表的依赖里没有 `Clock`，读时钟的是收集箱列表与待处理数），“时刻读一次”是句子比代码宽。 | `docs/v0/M7-collaboration/handoffs/M4-closeout.md:51`；`bs/issue_lists_snapshot_test.go:23-43`（`checkedListSnapshot`）、`:86-128`（成员列表：`tx.calls != 3`）；`ia/list_member_issues.go:18-25`（没有 `Clock`）；`si/modules/issue/module.go:260`（`ReadTx`）、`:277-278`（读时钟的两个）；硬规则 03 §5“文档的坑” | spec §4 加一格：工作区列表的整条链（找工作区、授权、`Visible`、目录、行、空页的总数）在一个 REPEATABLE READ 只读事务里，照 `TestMemberIssueReadSnapshot` 的写法，作为 T6 那个新顶层名的子测试（不加名字）。变异：`Visible` 挪到事务外；带游标的空页另开事务。§2.2 删去“时刻读一次”。 |
| I4 | plan Global Constraints、T4、T5 | **T4 → T5 是红窗口，plan 没有声明。** T4 把 `listWorkspaceIssues`（`issue` 标签）加进契约，这个提交上两条守卫立刻红：矩阵覆盖（“operation … has no row”）、删除窗口（操作集合与计数 `reads=20`）。T5 才补。P1c、P2a、P2b 的 plan 都在 Global Constraints 里写了红窗口与“其间不作为证据”；硬规则要求“红窗口的清单按 `contract.Operations()` 的使用者找全”。其余使用者自动跟随，不红（见 §3 第 2 条）。 | `bs/permission_matrix_coverage_test.go:96-100`；`bs/issue_deletion_window_cases_test.go:76-98`；`docs/v0/M7-collaboration/plans/P2b-intake-list-edit-delete.md:23`、`plans/P1c-notification-api.md:20`；硬规则 `03-rules-and-pitfalls.md` §5（“加 issue 的操作时……”一条） | Global Constraints 加一条：红的窗口 T4 → T5——`TestThePermissionMatrixCoversEveryOperation`、`TestDeletionWindowCoversEveryIssueOperation`（台账记下，其间不作为证据）；T4 的验证写明跑哪些（`ih/` 的处理器测试、`make gen-check`、`make test-web`、bootstrap 里路由与 401、参数 400 的几条）。 |
| I5 | spec §2.3（拒绝次序）、§4；plan T2–T6 的变异 | **变异是散在各 Task 里的三两个词，没有“性质 → 变异 → 所在 Task”的表；拒绝次序没有交叉格。** spec §2.3 定了“查询 400/422 → 404 → 403”，§4 只有“拒绝次序”四个字。要钉的交叉格：坏查询 × 看不到的工作区（外人、已移除的成员、不存在的 slug 都答 422，不是 404）；重复的已声明参数 × 没有令牌（绑定先于认证，422）；看不到的工作区与不存在的 slug 逐值相同。缺的变异：解析挪到授权之后；`Visible` 换成同一个接口上的 `Historical`（这才是“用了别的端口”最容易写错的那一个，T6 要有已离开的项目与已归档的项目各一个来分辨）；`archived` 被收；`sub_group_key` 被收；动作换成 `member_issue.list`（访客从 200 变 403，矩阵的访客格分辨）。“游标带出看不到的行”没有说改哪一行代码——这一格与“去掉逐项目的只看自己”是同一个变异，游标指向看不到的那一行时照样不给，应照实写。 | spec §2.3、§4；plan T2、T3、T6；`ia/visible_projects.go:19-26`（`Visible` 与 `Historical`）；`ia/list_member_issues.go:36-55`（解析在事务之前）；`id/list_query.go:128-145`；硬规则 03 §5（“拒绝的次序要在两个交叉方向上都钉住”） | plan 加一张“性质与变异”表（照 P2c plan），每行标 Task。spec §4 的单测一条写出三格交叉；T6 的种子写明：已离开的项目、已归档的项目、同一人一个项目只看自己另一个是成员、游标指向看不到的行。 |
| I6 | spec §1、R5、§5；plan T9（“M2 交接 §1、M4 交接 M7-03、M6 交接 M6-M7-03 的处理结果”） | **三条交接在再划分之后各只有一部分属 P3a，plan 把它们并成一句“处理结果”。** 调研 §1.9 是按没拆的 P3a 写的。逐条：① **M2 §1** 的关闭条件有三：每种排序的载荷（视图列表是集合型、不分页——这是 P3b 的表与接口）、改过的游标只在看得到的数据里换起点（工作区列表，P3a 的 H13）、跨模块联表的结论（navigation，P3b）。P3a 只能写第二条，不能写“本条关闭”。② **M7-03**：P3a 只完成“工作区跨项目列表分支”一句；视图在 P3b，页面与旧 `WorkspaceService` 四个方法在 P6a/P6b；标签目录留给 P6b（R5）可以，但交接的原话是“**个人主页**按标签分组需要完整的工作区标签目录”，消费方是已有的个人主页，另一个是工作区列表的标签筛选（设计“目录取看得到的项目的并集”）；“随页面定”要写明两个消费方、可能要加一个 project 模块的接口（网页阶段加契约），并写进设计 §12 的 P6b 一行，否则只留在 P3a 的 spec 里。③ **M6-M7-03**：设计 §11 已写“关闭”（§15 D4）；P3a 能写的只是“工作区列表不收迭代、模块的筛选词”，“项目视图有迭代、模块词”是 P3b。照 R3，工作区列表仍收 `cycle_name`、`module_name` 的排序词（行里也带 `cycle_id`、`module_ids`），处理结果与 plane-diff 要照实写，不能写成“工作区列表没有迭代、模块词”。 | `handoffs/M2-closeout.md:14-17`；`handoffs/M4-closeout.md:49-54`；`handoffs/M6-closeout.md:30-38`；设计 `M7-design.md:119,121,277-281,285`（“保持 open 至收尾关闭”）、`:351`（D4）；`id/listvocab.go:71-80`；`ia/listing.go:93-100` | T9 分三句写：M2 §1——P3a 的部分（游标只决定起点、测试所在），其余指明 P3b；M7-03——P3a 的部分，标签目录的落点与两个消费方，其余指明 P3b、P6a、P6b；M6-M7-03——筛选词不收、排序词照旧。spec §5 的 P3b 一条加“M2 §1 的视图列表与联表结论”；P6b 一条写明标签目录的两个消费方。T1 改设计 §12 时把标签目录写进 P6b 一行。 |

### 次要

| # | 位置 | 发现与证据 | 修正 |
|---|---|---|---|
| m1 | spec §2.3（`forbidden`“T4 定”） | 现在就能定：到不了。`workspace_issue.list` 三种角色都允许；工作区一级只在角色不在规则里时答 403（`si/modules/access/domain/decide.go:58-66`），而 `workspace_members.role` 有 `CHECK (role IN (5, 15, 20))`（`server/migrations/sql/00007_workspace_workspace_members.sql:10`）。“三种之外的角色”只是规则单测的一格（`access/domain/rules_test.go:21`）。先例两边都有：`listNotifications`（`api/modules/notification.yaml:40`）、`listWorkspaceStates`（`api/modules/project.yaml:614`）不声明；`getIntakeCounts`（`api/modules/issue.yaml:948`）声明了，只能靠处理器测试里的假用例回答。 | spec 写定“不声明 `forbidden`”，拒绝次序去掉“→ 403”；码是 `validation_failed`、`workspace.not_found`（`bad_request` 在顶层）。 |
| m2 | spec §2.2（“只平铺”） | 只列了 `group_by`、`sub_group_by`、`group_key`，少 `sub_group_key`。四个都不用写代码：前两个因 `ListGroupAllowed`、`ListSubGroupAllowed` 对新范围答否，后两个由 `validateGroups` 拒绝（`id/list_query.go:74-87,132-137`）。 | 列全四个；单测四格。 |
| m3 | plan 允许清单（“词表与解析的表驱动单测：新范围的行；不改已有行”） | 已有的两张表里范围是**列**不是行（`{field, project, member}` 的位置字面量，`id/listvocab_test.go:31-46,59-70`），加一列就要动每一行。 | 写成“另起 `WorkspaceList` 的表（子测试），已有的表不动”。`ListScope(3)` 今天没有测试当作未知范围用（用的是 0 与 255），不冲突。 |
| m4 | plan T3（“接线”） | 接线要 `httpadapter.UseCases` 上有 `ListWorkspaceIssues` 字段与它的接口（`ih/handler.go:157-189`），那是 T4 的处理器的一部分；放在 T3 要么编译不过，要么留一个没人读的字段。 | “接线”移到 T4（契约、处理器、字段、`module.go` 一行同一 Task）；T3 只有用例、动作、规则与单测。 |
| m5 | plan 允许清单（“`client.test.mjs` 受新路由影响的期望”） | 这个测试按契约逐个 GET 自动生成，没有要改的期望（`web/packages/api-client/test/client.test.mjs:207-233`）；`csvKeys` 已经有工作区的全部多值词（`src/index.ts:49-62`），只要加一条路径（`:35-39`）。 | 清单这一条改成“`csvListPaths` 加一条路径；`client.test.mjs` 不改”。 |
| m6 | spec §1、§4；plan T1 | ① H13 的原文含“与已存的工作区视图”（`M7-design.md:40`），这一半在 P3b；设计 §12“必须同时完成的验证”一列要随段拆开（P3a：H13 的列表、默认视图、规模；P3b：H12、H14 与 H13 的已存视图；P3c：H15–H17）。② 把收藏一段改叫 P3c，设计里还有一处旧号：§11 的 `M1-P3-trim-platform` 一行“P3b、P7”（`:277`）。`:265` 的“P3a 的规模测试”仍然对。M7 目录里没有别的引用。 | T1 点名 §12 的两列与 §11 `:277`；spec §1 写一句“H13 的已存工作区视图属 P3b”。 |
| m7 | spec §4（组合测试的各格） | `Visible` 只认有效的项目成员关系：工作区管理员不是成员的项目、公开但没加入的项目都不出现（`si/modules/project/adapter/postgres/queries/visible_issue_projects.sql:5-11`）。这是它与 access 的 `Readable` 的另一处差别（DP-5），各格里没有。 | 加一格：工作区管理员对自己不是成员的项目、成员对公开但未加入的项目，都不出现。 |
| m8 | plan Global Constraints（换路） | 核过：基线 770 个顶层测试；`TestWorkspaceIssueList…` 排在第 764 位（`TestWorkspaceImagesReadsAndRollback` 之后），自己落在第 1 路，后面 7 个换路：`TestWorkspaceProfilesConvertsIdentitysAnswer`、`TestWorkspacesCreateErrors`、`TestWorkspacesReactivateMemberErrors`、`TestWritesOnTwoProjectsOfAWorkspaceDoNotWait` 从第 1 路到第 0 路；`TestWorkspacesCreate`、`TestWorkspacesReactivateMember`、`TestWritesOnAProjectAreEachShape` 从第 0 路到第 1 路。plan 说的“约 7 个”成立。第 0 路另跑 bootstrap 以外的全部包，T2–T4 的单测与处理器测试都加在这一路。 | 把这张清单写进台账，检查点 1 对照；I3 的快照格作子测试，不另起名字。 |
| m9 | plan T8、检查点 2 | e2e 按条数平分两片（`.github/workflows/ci.yml:88-136`），新故事文件会改变划分；检查点 2 只写了 Go 的两路。 | T8 写明故事放哪（建议 `e2e/stories/collaboration/h13-workspace-issues.spec.ts`）；检查点 2 加 e2e 两片各自的时间。 |
| m10 | plan T7 | M4 收尾的命令给整个 `TestIssueListScale` 40 分钟（`docs/v0/M4-issue-core/plans/closeout.md:137`，三个规模各 13 个场景）；再加约 5 个场景乘三个规模与一个子种子，会不会超没有估。 | T7 写明命令、期限与证据目录；子种子只在判定的规模上跑，或给出预估。 |
| m11 | spec §2.2（范围）、§2.3 | ① 回答沿用 `IssueListPage`，它的 `data` 不是必填（`api/modules/issue.yaml:2611-2623`）；工作区列表永远是平铺。② 没有写带游标的空页怎么答（引擎会去掉游标再读一次取 `total`，`ia/listing.go:116-128`）。 | 契约的说明写“总是 `total`、`data`、`next_cursor`”；处理器测试钉空页的 `data: []`、`next_cursor: null`；H13 的翻页加末页之后的空页与 `total`。 |
| m12 | plan T6（`waitForStartupRuns`）、T5 | 删除窗口里项目删除时 200 的名单与 `check` 是两处 `switch`（`bs/issue_deletion_window_test.go:60,73-82`），空创建者是跳过名单加路径循环两处（`bs/issue_null_creator_test.go:37,51-52`）；plan 各写成一句。跳过名单不加，这一行会按“期望 404”去跑而失败。 | T5 逐处点名四个位置；删除窗口的 `check` 用 `windowPageIDs`，期望集合按夹具定。 |

## 3. 逐项回答

1. **可行性**（都成立）：
   - `ListScope` 加一项：`id/listvocab.go:13-18`；`ParseListQuery` 的入口守卫 `id/list_query.go:44-46` 加一个范围即可。`relation` 对新范围自然 422（`:88-92`，只给成员），也不会被要求（`:104-106`）。
   - 词：`ListFilterAllowed` 加一支（`id/listvocab.go:28-42`）；`ListFilters` 与 `parse`、`appendID` 不分范围，十二个词的字段都在（`id/list_query.go:19-27`、`id/list_filters.go:8-99`），`me` 在解析时换成调用者。
   - 只平铺：不用写拒绝的代码（m2）。引擎自己不认范围，只平铺完全靠解析这一处。
   - `Listing.Read` 不带目标成员：`ia/listing.go:68-71` 要求“没有 `relation` 就没有目标成员”，工作区列表正是；项目集合为空直接答空页（`:85-87`）。SQL 的 `relation = ''` 一支（`si/modules/issue/adapter/postgres/queries/list_issues.sql:49`）、成对的项目数组与逐项目的 `own_only`（`:4-5,24`）、筛选项目与范围相交（`:26`）、内连接状态目录排除 Triage（`:21`）、已归档（`:23`）都现成。不用新 SQL、不用新端口，所以运行角色的授权、sqlc 范围、`TestVisibleIssueProjectsConversion` 等端口转换测试都不动。
   - `Visible`：有效成员、项目未删未归档、`own_only` 按项目角色（`visible_issue_projects.sql:1-12`），与设计 `:121` 一致；成员列表、搜索、关联、待处理数已在用（`ia/list_member_issues.go:63`、`search_issues.go:53`、`list_relations.go:53`、`count_intake.go:82`）。
   - 用例的模板：`ia/list_member_issues.go:31-74`，去掉 `:56-62` 的目标成员与 `TargetMemberID`；它的单测是按调用记录写的（`ia/list_member_issues_test.go:46`），拒绝次序与“`Visible` 为空的空页”都能在这一层钉。
   - 回答的形状：`listPage(page, false, false)`（`ih/list_issues.go:23-29`）。
   - 单值参数清单与 `withListQuery`：见 I1。
   - `csvListPaths`：见 m5。
   - 码：`validation_failed`、`workspace.not_found` 到得了；`forbidden` 到不了（m1）。`limit=x` 等绑定错误是顶层的 `bad_request`，`TestParametersThatDoNotBindAnswer400` 自动覆盖。
   - 路由：与 `GET /api/v0/workspaces/{slug}/issues/{identifier}`（`bs/issue_deletion_window_cases_test.go:30`）少一段，不冲突。
2. **守卫**（`contract.Operations()` 的使用者逐个核过）：
   - 会红、要补：权限矩阵的覆盖（`bs/permission_matrix_coverage_test.go:96-100`；一行，`inWorkspace(cellOK, cellOK, cellOK)`，模板 `bs/permission_matrix_search_test.go:18-37`，带分辨访客范围的 `check`）；删除窗口（`bs/issue_deletion_window_cases_test.go:96` 的 20 → 21，加 `:18-68` 的一行；项目删除时 200 在 `bs/issue_deletion_window_test.go:60`，`check` 在 `:73-82`；工作区删除时默认 404 且与不存在的 slug 逐值相同，自动成立）；空创建者（`bs/issue_null_creator_test.go:37` 的跳过名单、`:51-52` 的路径循环）。plan 的 T5 三样都有，位置见 m12。
   - 自动跟随、不红：路由（`bs/contract_test.go:47-73`、`bs/asset_contract_test.go:35`，处理器与契约同一 Task）；401（`:76-98`）；参数 400（`:216-244`）；问题响应的头（`:252-260`）；根路径与标签（`apitest/rules_test.go`，`api/openapi.yaml` 加路径）；处理器逐码（`ih/handler_test.go:25` 的 `apitest.Main`）。
   - 不涉及：`writesOnAProject`（只认非 GET，`bs/project_write_locks_test.go:150-155`）；`writesOnIssues`（非 GET，`bs/issue_write_locks_test.go:195-203`）与草稿写锁表；访客见全部的守卫（只管 planning 的写，`bs/permission_matrix_guests_see_all_test.go:114-116`）；限流的测试（按桶，不按操作）；`bs/issue_list_directories_test.go`（端口转换，没有新端口）。
   - 动作：`workspace_issue.list` 进 `id/actions.go:60` 的 `Actions()` 与规则表，同一 Task，`bs/actions_test.go` 的两条守卫保持绿；`rules_test.go` 的 `tableCells` 一行（七格：允许三格、看不到三格、三种之外的角色 403）。
   - plan 没有漏守卫；漏的是 HTTP 的第二张名单（I1）与红窗口的声明（I4）。
3. **规模**：结构上放得下——场景是 `TestIssueListScale` 之下每个规模里的 `t.Run`，不加顶层名，CI 上仍是一次跳过（`bs/issue_list_scale_test.go:81-83`）；路由、身份与期望要各动一处，通过线没有数，见 I2。M4 的数字在 `M4-design.md:639-642` 与 `specs/P5-issue-list.md:129-140`。
4. **换路**：成立，见 m8。
5. **交接**：见 I6。标签目录留给 P6b 可以；M2 §1 与 M7-03 在 P3a 只能写部分结果；M6-M7-03 设计已关闭。
6. **R1**：两段之间 main 的样子写得对——网页没有调用新路径（`web/` 里除生成物外没有这条路径的读者），没有 navigation 模块与 `issue_views` 表，P3a 不加表所以删除守卫、清理计数、迁移计数都不动。依赖方向对：P3b 的工作区视图校验用 `ListFilterAllowed(·, WorkspaceList)` 的词；P3a 不依赖 P3b 的任何东西。要补的是 H13 的已存视图一半与设计里的旧号（m6）。
7. **plan**：次序基本对（T2 → T3 → T4 → T5 → T6）；“接线”的位置见 m4；红窗口见 I4；变异见 I5；允许清单的缺口见 I1、m3、m5；检查点 1 放在 T6（组合测试之后）、检查点 2 在 e2e 之后，位置对，检查点 2 少 e2e 两片（m9）。9 个 Task 现实：T4 最重（契约约 13 个参数、两张名单、处理器测试、网页客户端），T6 次之（约 12 格加快照）；修复轮之后约 10 个，远在 16 之内。

## 4. 核实为正确的事实（摘要）

- 列表引擎、SQL、游标、目录都不分范围，成员列表已经在多个项目上用同一个 `[order, key, id]` 游标；P3a 不需要改 `ia/listing.go` 与任何 SQL。
- `ListOrderAllowed` 没有范围参数，也是游标键的校验（`id/listvocab.go:71-80`）；R3 沿用它成立。
- 按 `cycle_name`、`module_name` 排序时引擎对全部可见项目读一次名字（`ia/listing.go:96-99`），成员列表已如此。
- 调用者看不到的工作区答 `workspace.not_found`，与不存在的相同（模板 `ia/list_member_issues.go:46-52`）。
- 工作区管理员兼项目访客仍只看自己：`own_only` 只按项目角色算（`visible_issue_projects.sql:3`）。
- 空创建者的工作项对只看自己的人不出现：`i.created_by_id = actor` 对 NULL 不成立（`list_issues.sql:24`）。
- 重复的已声明参数在绑定时就拒绝、先于认证；未声明的词（`group_by`、`cycle_id`、`relation`）经原始查询到解析器答 422——前提是 I1 的名单加了。
- 基线 bootstrap 770 个顶层测试，与调研 §7.1 一致；`TestIssueListScale` 排在第 453 位，加场景不改任何名字。
- 网页一侧 P3a 只碰 `web/packages/api-client/src/index.ts` 的一行与生成物；没有新的问题码，不用加文案。
- 再划分后设计里要改的旧号只有 §11 `:277` 与 §12 `:299-300`。

## 5. 处理

待主控。

## 处理（主控，2026-11-20）

全部采纳，改入 spec 与 plan：

- I1：spec §2.1 写明 `list_query.go` 的两张按操作的清单（`withListQuery`、`listSingleQuery`）；T4 的变异表各一条与 `csvListPaths` 一条；`list_query_test.go` 的两张期望进允许清单。
- I2：spec §2.4 写明判定的规模（总量 20000）、配对的场景、度量（SQL 中位，1 次预热 + 5 次实测）与通过线（配对的项目列表的 1.5 倍以内，且不超过 M4 的 105.53 ms）；请求的人在一部分行上是负责人与关注者；路由的选择扩为三种。
- I3：只读快照一格进组合测试与变异表；删去“时刻读一次”。
- I4：T4 → T5 的红窗口写进 Global Constraints。
- I5：性质与变异表；交叉格写进 spec §4；T6 的种子写明。
- I6：T9 的三个交接各写部分的处理结果；spec §5 与设计 §12 的 P6b 一行写进标签目录。
- m1–m12：不声明 `forbidden`、拒绝次序去掉 403（R7）；只平铺加 `sub_group_key`；词表单测另起一张表；接线随 T4；`client.test.mjs` 不改期望；H13 的已存视图属 P3b、设计 `:277` 改 P3c；不是成员的项目一格；换路清单写进 plan；检查点 2 加 e2e 两片、故事文件的位置；规模跑重估时间；恒为平铺的说明与末页之后的空页；T5 写明四处位置。
