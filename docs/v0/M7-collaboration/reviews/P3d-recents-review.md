# M7 / P3d：最近访问（阶段记录）

> 状态：已完成，合入 main（noff `3d9293bd`），main CI 44989539530 全绿（§9）。spec 与 plan 见 [P3d spec](../specs/P3d-recents.md)、[P3d plan](../plans/P3d-recents.md)，预检 [re-92](../../../reviews/re-92-M7P3d预检.md)。基线 main `9d26b1c3`（P3c 的 noff），分支 `codex/m7-p3d`。台账 `.superpowers/sdd/P3d-recents/progress.md`（ignored，随阶段归档）。

## 1. 交付

- `IssuePriority` 纯搬进 `api/common.yaml`（R8），issue 的 3 处 Go 跟随；打包的 openapi 与 `schema.gen.ts` 逐字节不变，网页的类型照旧。
- 表 `user_recent_visits`（迁移 `00054`）：两种类型 `issue`、`project`，`project_id` 非空，两条 CHECK（类型两值；项目的访问 `project_id = entity_identifier`），三条带 `deleted_at IS NULL` 的部分索引（唯一 `…_entity_key`、连带、列表与裁剪）。
- 接口 2 个：`recordRecentVisit`（`POST /me/recent-visits`，新记与重访都 204）、`listRecentVisits`（`GET /workspaces/{slug}/recent-visits?entity_type=`，集合型 `{data}`，至多 20 条，项目的行 `issue: null`）；动作 2 个（`recent_visit.read`、`recent_visit.write`，工作区级，管理员、成员、访客）；没有新码，网页文案不改。
- 规则：一个可见谓词，记录与列出共用——项目在调用者的 `Readable` 里（含已归档的，不是成员的工作区管理员不算），工作项经 issue 的 `NavigationIssues` 判（未删、项目在授权里、`HiddenFrom` 不藏）；列出另去掉收集箱里的。已归档的能记、照列；收集箱里的条目能记、接受之后才列。记录上的每个 404 是目标自己的码（`issue.not_found`、`project.not_found`），与不存在的逐值相同、不写；工作区由目标推出；拒绝次序 400 → 422 → 查找。重访同一行，`visited_at`、`updated_at`、`updated_by_id` 同一时刻。列表先过滤后取 20，摘要取目标此刻的。GET 都没有副作用。
- 上限：每人每工作区未删的行至多 20 条，两种合计、读时藏起的也算；记录以本人与工作区的事务级咨询锁串行（键的公式只在 `LockRecentVisits` 一条 SQL 里），同一事务里物理删第 20 条之后的全部未删行，`FOR UPDATE SKIP LOCKED`。
- 锁：`RecentVisitLocks`：工作区 S → 项目 S → 锁下重读、重判 → 咨询锁 → 时钟 → upsert → 裁剪；持咨询锁的事务从不等行锁；不进写闸门（`Deps.Tx`）。
- 连带与删除：订阅者 `navigation_forget_deleted_issue_visits` 在工作项删除的事务里软删所有人的行（子树逐个、收集箱未接受条目的删除同此）；`navigation.delete_rows` 的项目范围加第三步（视图 → 收藏 → 最近访问），只动未删的行；归档不动；60 天清理接上（清理目录 36 → 37）。
- 端口：project 的 `NavigationTarget` 加 `Identifier`；issue 的新端口 `NavigationIssues`（地点与按调用者可见的摘要），它的 `NavigationIssueRows` 是第四条带 `Intake`（条目的 `status`）的查询。

## 2. Task 与提交

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `81b57eef` | spec、plan、预检 re-92；P3c 记录 §9 补记；设计 §14 |
| T2 | `e6db30e9` | `IssuePriority` 的纯搬移；检查点 0 |
| T3 | `8ba9f8e3` | 迁移、schema 测试、两个动作、清理一步；红窗口 (a)、(b) 起 |
| T4 | `576a2ba3` | 删除任务的第三步、Go 删除守卫的种子；红窗口 (a) 关闭；Go 两路中间检查 |
| T5 | `ae0b5b46`、`e938ea17` | 领域；存储 |
| T6 | `334b53c0` | 端口：项目目标的标识、issue 的 `NavigationIssues` |
| T7 | `91a2a0a2` | 用例：记录、列表、可见谓词、`RecentVisitLocks` |
| T8 | `ff3122fc` | 契约、处理器与接线；红窗口 (c) 起 |
| T9 | `98507b2d` | 工作项删除的连带 |
| T10 | `df50185a` | 权限矩阵、锁序测试；红窗口 (c) 关闭 |
| T11 | `0660c696` | e2e 删除夹具；红窗口 (b) 关闭；检查点 1 |
| T12 | `147ddcef` | 组合测试 H17 与工作项删除的连带 |
| T13 | `f9983159` | 交错 |
| T14 | `6a4e4e51` | e2e 接口故事 H17；检查点 2 |
| T15 | `95c8dbd3` | 文档与交接 |
| 修复轮 | `23c89903`、`55e89c76`、`76e26e06`、`903ed9ca`、`4012cd74` | 整分支审查的 m1、m6、m3、m4、m5（§9） |
| 限定复审的 O1 | `abe2f209` | 记录的写与裁剪失败时记录随之失败（§9） |

`e938ea17`、`df50185a`、`147ddcef` 是只改说明的重提交（树不变，原为 `af1bb9f8`、`46dd61f5`、`107ae4ae`）。

## 3. 局部裁定（实施中）

- **部分唯一的名字**（T3）：`user_recent_visits_user_id_workspace_id_entity_key`（50 字节；照列名全写 73 字节，超过 63 字节会被悄悄截断，名字的守卫就钉不住），迁移注释写明 `entity` 指两列；`project_id_check` 两列都非空，不用 `coalesce`。
- **删除任务的第三步**（T4）：端口 `RecentVisitDeletions` 与 `FavoriteDeletions` 并列；项目范围视图 → 收藏 → 最近访问，各自从头的游标、共用预算，工作区范围不加；只动未删的行，工作项先删、项目后删时连带写下的时刻与删除者不被改写。“视图与收藏”的陈旧注释在几个文件里一并补上最近访问。
- **四个端口、一个时刻**（T5）：`RecentVisitReader`、`RecentVisitLocker`、`RecentVisitWriter`、`RecentVisitCascades` 照收藏的分法；`RecentVisit` 只带 `VisitedAt`，存储以它同时写新记的建立、修改与重访的修改（“同一时刻”由一个字段保证）；`CheckRecentVisitKind` 答 `error`（唯一的检查，没有要并进同一个 422 的别的问题）；navigation 存储测试的行只经一个构造者 `seedVisit`。
- **冲突谓词多一个条件是等价变异**（T5）：plan 变异表“upsert”一行写反了——多一个 `AND` 的谓词更强，仍推得出同一条部分索引；42P10 的方向是推不出（去掉谓词、用 `OR` 放宽），两个都杀。领域与存储 40 个变异 39 杀、1 等价。硬规则 §5 新增一条。
- **issue 的端口**（T6）：读端口与值在新文件 `iss/app/navigation_issues.go`，行自带字段，以 `domain.Issue{CreatedByID, Intake}` 问 `HiddenFrom` 与 `InTriage`（谓词仍只在 issue）；条目行只取 `status`；优先级在两边的值里是 `string`（bootstrap 按类型直接转换，要求字段类型相同）；负责人用 `array(...)`，没有时是空数组；`navigation.New` 不读 `Issues`，`newNavigationRoute` 不补桥（re-92 m9）。
- **用例**（T7）：`RecentTargets{Readable, Projects, Issues}` 导出（模块根在包外构造它）；谓词 `sees`（记录与列出共用）、`shows`（再去掉收集箱里的与项目读不到的）、`visible`（先过滤后取 20）；锁下重读的地点须与预检的相同（拿着的锁是预检那个工作区与项目的）；项目 S 读不到即目标的 404；授权的不可见在预检与锁下都答目标自己的 404，从不答 `workspace.not_found`。
- **契约与处理器**（T8）：回答的 `assignee_ids` 为空时给 `[]`；`api/modules/navigation.yaml` 710 → 893 行、不拆（模块文件即标签，`TestEveryOperationIsTaggedWithItsModule`；`issue.yaml` 更长）。
- **连带的两个额外测试文件**（T9）：`TestTheModulesSubscribers`（“不注册”在 T9 就有确定的杀法）与存储上的 `TestAWorkItemsDeletionForgetsEveryonesVisitsOfIt`（“只删执行者自己的”变异由断言在 2.4 秒杀，T5 的同一变异靠 10 秒的等锁超时）；都不在 bootstrap，不占顶层名额。
- **锁序测试的第三段**（T10，re-92 I1）：持有者在 `WithinTx` 里经存储的 `LockRecentVisits` 取锁，`pgtest.WaitForStatementLockWaitBehind`（持有者的连接、语句名）等到请求，`readRequestLock` 认出它自己的等待；“不持咨询锁”用 `pg_locks` 数请求后端已授予的 advisory 锁，不复制键的公式。“不取咨询锁”的变异在这一段确定地死（另一个杀法是 T7 的调用序列）。矩阵的列表行复用唯一的造行者 `seedRecentVisits`；过去的、成批的访问另写 `seedPastVisits`，T13 把它扩展为可指定目标与 id。硬规则 §5 新增一条。
- **e2e 夹具只记项目一条的变异存活**（T11）：e2e 的删除目录按表计数，项目那一条就给了表的行；对照“第三步只删项目的行 + 只记项目一条”全过、“+ 两条都记”六条失败，证明工作项那一条守着第三步。硬规则 §5 新增一条。
- **组合层的等价变异**（T12）：四个变异在 app 一层有另一层给出同样的回答而存活，各在更早的 Task 有确定的杀法——重访不改 `updated_by_id`（经 app 一条访问只有它的主人写，T5 以另一个写者杀）、列表不按工作区限定（读时的谓词按这个工作区的 `Readable`，T5 杀）、订阅者拒空执行者（经 app 的删除都有执行者，T9 杀）、去掉同值的 `id DESC`（索引供给次序，T5 关掉索引扫描杀）。同一时刻的两条访问由一条 SQL `UPDATE` 造（每个请求只读一次时钟）。
- **交错**（T13）：复用 P3c 的 `issueWorld.race` 与 `favoriteRacer`（修复轮改名 `racedWrite`；另一个事务扣两个写都要的行、两个写都经真实的 app）；项目删除在 SQL 里的两层（项目的 `FOR SHARE`、`NavigationTargets`）在真实 app 里还有第三层（锁下的可见经 `Readable`），两层一起去存活、三层一起去才杀，两层的杀法在 T7 的假实现；死锁格由 500 加“记录等了”推断（同 P3c T12）。硬规则 §5 新增一条。
- **e2e 的时刻**（T14）：夹具把库里的时刻写成接口的格式（UTC 的 RFC 3339，秒的小数去掉末尾的 0），列表的 `visited_at` 与库里那一行、连带的删除时刻与工作项的 `deleted_at` 都逐字比较；另加 `listedNow`（由库里的行与目标此刻的回答拼出列表的一项）与 `issueDeletion`。

## 4. 向负责人说明

- 没有需要负责人裁定的事项。预检 re-92 的 4 个重要、11 个次要全部在 T1 采纳（m2 按主控的裁定，re-92 §6）。
- 第 1 路的两条线（spec §6，主控裁定）：本机停止线以检查点 0 在本机的第 1 路 668.2 秒为基线，停查线 698.2 秒；此后 T4 651.3、检查点 1 648.8、检查点 2 672.6（+4.4），都在线内。CI 的八成线（bootstrap 960 秒）只是报告线、不是停止线：三次分支 CI 第 1 路的 bootstrap 769.5、766.7、943.2 秒（64.1%、63.9%、78.6%），都没到；检查点 2 的 943.2 秒与本机只多 4.4 秒对不上，是 runner（同 P3c 终检的 938.8 对 main 的 752.5），趋势在 M7 收官时报告：此后 P4–P8 再加测试，第 1 路在慢 runner 上会碰到八成，届时带分析请负责人定分片或期限。
- 检查点 1 的分支 CI 第 0 路 Test 比检查点 0 多 112 秒，是慢的 runner，不是代码变重（§6）；本机第 0 路的 bootstrap 同期 465.6 → 459.6 秒。
- M6-M8-05 在检查点 1 出现一次（cp1）：本机 Docker 起 postgres 容器时报 `invalid port`（日志 433 处），五个各起容器的包（identity、issue、navigation、notification 的 `adapter/postgres` 与 `migrations`）一起失败，bootstrap 本身过；按规则整套重跑一次（cp1b），七项一次过。
- 文件大小：`bs/permission_matrix_test.go` 仍 452 行（`matrixRows()` 就地加一项）；`api/modules/navigation.yaml` 893 行（§3）；新文件都在 400 行以内（最长的 `bs/navigation_recent_visits_interleaving_test.go` 333、`e2e/stories/collaboration/h17-recent-visits.spec.ts` 331、`bs/navigation_recent_visit_locks_test.go` 327）。

## 5. 本地门禁与 CI

| 时点 | 本地七项 | 分支 CI |
|---|---|---|
| 检查点 0（`e6db30e9`） | 七项一次过（e2e 493 passed） | 44958948278 attempt 1 全绿 |
| T4 之后（`576a2ba3`） | Go 两路过（中间检查，e2e 在窗口 (b) 里） | — |
| 检查点 1（`0660c696`） | cp1 第 0 路因 M6-M8-05 失败（五个各起容器的包，bootstrap 本身过，447.5 秒）；整套重跑一次（cp1b），七项一次过（e2e 493 passed） | 44973647135 attempt 1 全绿 |
| 检查点 2（`6a4e4e51`） | 七项一次过（e2e 494 passed） | 44982376252 全绿 |
| 合并前的终检（`abe2f209`，`final.sh fin`） | 八张收据一次过（e2e 494 passed） | 44987439140 全绿（最终分支 CI） |
| main `3d9293bd`（合入之后） | — | 44989539530 全绿 |

## 6. 测试时间与两路（秒）

| 时点 | 本地 bootstrap 第 0 / 1 路 | CI server (0) Test | CI server (1) Test |
|---|---|---:|---:|
| main `9d26b1c3`（44947828240，基线） | | 662（bootstrap 475.1） | 773（bootstrap 752.5，62.7%） |
| 检查点 0（44958948278） | 465.6 / 668.2 | 741（bootstrap 514.2） | 791（bootstrap 769.5，64.1%） |
| T4 之后（本机中间检查） | 446.3 / 651.3 | — | — |
| 检查点 1（44973647135） | 459.6 / 648.8（cp1b） | 853（bootstrap 570.1，慢的 runner） | 788（bootstrap 766.7，63.9%） |
| 检查点 2（44982376252） | 468.1 / 672.6 | 673（bootstrap 489.9） | 974（bootstrap 943.2，期限的 78.6%） |
| 终检（44987439140） | 471.2 / 660.5 | 673（bootstrap 493.0） | 808（bootstrap 786.8，期限的 65.6%） |
| main `3d9293bd`（44989539530） | | 692（bootstrap 498.9） | 757（bootstrap 736.3，61.4%） |

- 本机两路整路（`make test-shard` 的回执）：检查点 0 595.4 / 673.0，检查点 1（cp1b）590.2 / 654.0，检查点 2 598.8 / 679.0；e2e 三次各 3.7–3.8 分钟。
- 新的 bootstrap 顶层名（桶 = `cksum % 100`，0–45 第 0 路），本机单跑：

| 名字 | 桶 | 路 | 本机 |
|---|---:|---:|---|
| `TestRecentVisitsFollowDeletedWorkItems` | 4 | 0 | 4.65–5.67 |
| `TestRecordingARecentVisitTakesItsLocks` | 17 | 0 | 3.05–5.86（同包跑 1.87） |
| `TestRecentVisitWritesInterleave` | 33 | 0 | 7.53–8.49（第一次绿 10.78） |
| `TestRecentVisitsThroughTheApp` | 36 | 0 | 5.62–6.60 |

- 被加重的已有测试：`TestPermissionMatrix`（桶 46，第 1 路）3673 → 3697 格（+24：列表 6、记录 18）；清理与删除守卫各多一张表。本机第 1 路的 bootstrap 由检查点 0 的 668.2 到检查点 2 的 672.6（+4.4）。
- 检查点 1 第 0 路的 runner 慢：P3d 没碰的 59 个包合计 543.4 秒（P3b 时的 main 404.2、检查点 0 477.7），1.34 倍；bootstrap 570.1 与之同比，不是代码变重。第 0 路 Test 从 741 到 853（+112）因此过了每路 90 秒的增量线；同一份代码在本机第 0 路的 bootstrap 459.6 秒。第 1 路两次 CI 都在 64% 上下（769.5、766.7），与 main 的 752.5 相近。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- 无新增。M6-M8-05 在检查点 1 出现一次，属已有的交接（§4、§5）。

## 8. 文档与交接（T15）

- plane-diff：表 `user_recent_visits` 一行（二、保留表的改动）、“最近访问（M7/P3d）”一行（四、行为差异），Plane 的行号已对过源码。D43 按源码改正：Plane 上限淘汰的软删墓碑由每日 `hard_delete` 末尾的通用循环在 60 天后物理删（`apps/api/plane/bgtasks/deletion_task.py:182-191`，按 `deleted_at` 清每个软删的模型），不是“墓碑永存”——调研、spec §2.5 与 R12、re-92 §3 第 10 条只读到 `:114-172` 的显式名单。Nerve 的差异不变：淘汰物理删、不留墓碑。
- 交接的处理结果（都保持 open 至收尾，页面随 P7）：M1-P2-trim-content（最近访问没有 `page`；首页的旧读者已随 `5c128973` 删除）、M2 §1（最近访问的列表）、M2 §2（`/me/recent-visits` 在 `navigation.yaml`，`fetchWorkspaceRecents` 已消失）、M2 §5（`user_recent_visits` 的外键去向）、M4 M7-04（删除工作项时清掉最近访问）。
- 硬规则 §4：`user_recent_visits` 各个写者的锁序、“持咨询锁的从不等行锁”、裁剪 `SKIP LOCKED` 的理由（re-92 m1 改正后的环）；P2a 一段加上 `NavigationIssueRows` 带条目的 `status`。§5 四条：锁下重判的层数按真实的 app 数；冲突谓词更强是等价变异；bootstrap 里等咨询锁的探针；e2e 的删除目录按表计数。
- 设计 §3.9（“Plane 相同”改为 Plane 单个删除时硬删、批量删除不清，D44；实现指针）、§4.2 的 `user_recent_visits` 一行、§4.3 的清理计数到 37、§5 的最近访问一行、§12 的 P3d 一行加 spec 链接、§14 的 P3d 一行。

## 9. 整分支审查与集成

- **整分支审查**（fable，只读，`9d26b1c3..95c8dbd3`）：没有严重、没有重要，可以合并。次要六条：m1 裁剪的删除计数只给测试用；m2 `RecordRecentVisit` 的写者总等于本人；m3 issue 里两处 `ownOnly` 循环；m4 删除窗口的工作区格里一条恒真的列表检查；m5 最近访问的交错用名为 `favoriteRacer` 的类型；m6 时钟回拨时新的访问会在自己的事务里被裁掉。
- **修复轮**（主控裁定：m2 不改——显式的写者与别的写端口一致；其余都改）：
  - `23c89903`（m1）：`TrimRecentVisits` 改 `:exec`，端口、存储、用例只答 `error`；存储测试改按 SQL 读表断言删了哪些行，原先由计数杀的 T5 裁剪变异重跑 15 个皆杀。plan T5 那句“答删了几条”是当时的计划，不改。
  - `55e89c76`（m6）：spec §6 记时钟回拨为已知风险（自愈、不越上限、不留孤行，不加测试）。
  - `76e26e06`（m3）：`seenByGrants`（`iss/app/issue_refs.go`）给 `IssueRefs` 与 `NavigationIssues` 共用；两个变异都使两边的测试失败。
  - `903ed9ca`（m4）：列表检查移进每格自己的删除里——项目格查 acme 不列 Dropped 与它的工作项，工作区格只留 doomed 的 `workspace.not_found`（逐值相同）；“列表不过滤”的变异使项目格失败，工作区格两层（`DirectoryWorkspace`、`ActiveRole`）一起去才失败、只去一层的对照存活。
  - `4012cd74`（m5）：`favoriteRacer` → `racedWrite`（P3c 与 P3d 的交错文件），纯改名。
- 以下在 P4 的 T1 补记，数字取自归档台账的末行：
  - **限定复审**（fable，`95c8dbd3..fd0fc6de`）：没有发现，Ready to merge。观察 O1（用例吞掉裁剪的错误测不出，早已存在）由主控当场以 `abe2f209` 补测试：记录的写与裁剪失败时记录随之失败，两个变异皆杀；O2（plan T5 的原话）照上文的修复轮记为不改。
  - **合并前的终检** `final.sh fin`（`abe2f209`）一次过：八张收据全 0；bootstrap 第 0 路 471.2 秒、第 1 路 660.5 秒（本机基线 668.2）；e2e 494 passed。
  - **最终分支 CI** 44987439140 第一次全绿：server (0) Test 673 / bootstrap 493.0，server (1) Test 808 / bootstrap 786.8（期限的 65.6%）；web、e2e 两片绿（§5、§6）。
  - **合并**：`--no-ff` 合入 main，`3d9293bd`（父 `9d26b1c3`、`abe2f209`），推送后以 `ls-remote` 核对。**main CI** 44989539530 第一次全绿：server (0) Test 692 / bootstrap 498.9，server (1) Test 757 / bootstrap 736.3（61.4%）；web、e2e 两片绿。
  - **归档与清理**：归档 `.superpowers/sdd/M7-P3d-integration-20261122`（载荷 3266 个文件，分类只有可复现与证据两种、没有未知，`source_unchanged` 为真，清理之后核过）；`cleanup.log`（2026-11-22T05:55:14Z）：worktree `m7-p3d` 已删，本地分支 `codex/m7-p3d`（在 `abe2f209`）与远端分支已删。
