# M7 / P3d 预检

> 2026-11-22：结论 **修正后通过**（Ready after fixes）。没有严重问题；4 处重要、11 处次要。合同的每一项在 `9d26b1c3` 上都做得出来：`00054` 空着，计数与名单和 plan 列的一致，sqlc 实测收下 `hashtextextended` 的咨询锁、带部分唯一谓词的 upsert 与 `MATERIALIZED … SKIP LOCKED` 的裁剪；记录的锁序（工作区 S → 项目 S → 咨询锁 → upsert → 裁剪）对每个锁同一批行的写者都不成环，“工作项的每条删除路径整个事务持项目 N”属实；可见谓词恰是 `issue.read` 加 `HiddenFrom`；`issue.deleted` 的发布点只有删除核心一处，没有漏掉的删除路径；四个新顶层名的桶复算无误，第 1 路的增量接近零；Plane 的事实抽查全部成立。重要问题都在测试一侧：bootstrap 的锁序测试用错了等待探针（应用在跑时它数的是任何后端）；变异表里“咨询锁”一行写的后果一条不会发生、一条靠竞态；可见与上限的几条性质没有杀得死的变异（只列自己的、先过滤后取 20、裁剪删到只剩 20、已软删的行不被改写）；交错里“裁剪 × 扣着的一行”照字面做不出来。不需要改设计的取舍，R1–R14 没有一条被代码推翻，没有要负责人新定的事。
> - **目标**：[P3d spec](../v0/M7-collaboration/specs/P3d-recents.md)、[P3d plan](../v0/M7-collaboration/plans/P3d-recents.md)（未提交的草稿），次要目标 `.superpowers/sdd/P3d-recents/implementer-brief.md`。依据 [M7 设计](../v0/M7-collaboration/M7-design.md) §3.9、§3.10、§4.1–§4.3、§5、§6、§9、§10（H17）、§12、§15，调研 `research/backend.md`（DP-n）、`research/plane-e2e.md`（PE-n），台账的主控裁定，先例 P3c 的 [spec](../v0/M7-collaboration/specs/P3c-favorites.md)、[plan](../v0/M7-collaboration/plans/P3c-favorites.md)、[阶段记录](../v0/M7-collaboration/reviews/P3c-favorites-review.md)、预检 [re-91](re-91-M7P3c预检.md)，[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md) §4、§5。
> - **基线**：main `9d26b1c3`（P3c 的 noff），worktree `codex/m7-p3d`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git 状态（只用了 `git status`、`git log`、`git show --stat` 读先例提交）、不起容器、不跑测试。跑过的有三样：

- `go test -list . ./internal/bootstrap`（编译、列名字：781 个顶层测试，没有带 `Recent`、`Visit` 的名字）。
- `printf %s 名字 | cksum`（查桶）与 `wc -c`（查标识符长度）。
- **一次 sqlc 探针**，任务书没有点名，这里写明：在会话的 scratchpad 里放一份草拟的 `00054` 表、四条语句（两种写法的咨询锁、upsert、裁剪）和一份只指向 scratchpad 的 `sqlc.yaml`，用仓库钉住的 sqlc（`server/tools/go.mod`，带 `GOFLAGS=-mod=mod`）生成到 scratchpad。仓库里没有写入任何文件，探针前后 `git status --short` 都只有两份草稿。它只证明 sqlc 肯编译这些语句，不证明 PostgreSQL 的运行行为。

对照代码核了：issue 的 `Locks.run`、`lockAndDecide`、删除核心、两个删除用例、子树与删除的 SQL、`HiddenFrom` 与 `InTriage`、`IssueRefs`；access 的 `Readable` 与规则表；navigation 的 `FavoriteLocks`、`delete_rows`、订阅者、清理、SQL 与模块根；notification 的同类订阅者与全部已读的 SQL；workspace、project 两个模块全部的行锁语句；事件的发布者、订阅者与每个自己拼 dispatcher 的测试；`contract.Operations()` 的全部使用者；矩阵的运行器、覆盖规则、目标规则与收藏的行；项目写锁表的分类函数；清理、删除、迁移的各处计数；`pgtest` 的等待探针与请求探针；e2e 的 `furnish.ts`、`catalog.ts`、项目删除断言与每个调用者；分路脚本；契约里可空对象、查询枚举、码集的先例与 `IssuePriority` 的全部引用；Plane 的记录任务、列表、序列化、模型与两个删除接口。下文路径相对 worktree 根，`si/` = `server/internal/`，`bs/` = `si/bootstrap/`，`nav/`、`iss/`、`pj/`、`ws/` = `si/modules/{navigation,issue,project,workspace}/`，`mig/` = `server/migrations/`，`pa/` = `plane/apps/api/plane/`。

没有核实的：

- 任何测试的实际耗时与两路、两片的余量（没有运行）。78.2%、62.7%、655.4 秒取自 plan，只在 P3c 阶段记录里对上了检查点 2 的 649.0 秒（`:77`）。
- 红窗口 (a) 的六个名字是否就是全部：静态读不出，只有 T3 跑一遍才知道。
- `make gen` 之后生成物的差异（`IssuePriority` 搬移后 `api/dist` 与 `schema.gen.ts` 里名字不变，只按 `LogoProps` 的先例 `ea92002a` 推断）。
- 裁剪语句在真实库上的行为（`READ COMMITTED` 下行被并发改写时带 CTE 引用的条件怎样重判）、`ON CONFLICT` 对部分唯一的推断（与 P3c 的 `InsertFavoriteItem` 同一写法，只按它推理）。
- m10 里 PostgreSQL 对共享行锁不排队的说法（按实现的记忆写的，没有在库上试）。
- 调研 `plane-e2e.md` 里前端一侧的事实，与 T1 要回填的 P3c 数字。

## 2. 发现

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §4“锁序”；plan T10；brief 的“The advisory lock”一条 | **bootstrap 的锁序测试用 `pgtest.WaitForLockWait` 等咨询锁，这个探针在应用运行时不成立。** 它数的是“这个库里任何一个在等锁的后端”，自己的文档写明只适合“没有应用在跑”的测试：应用的 River 任务（通知分发先 `FOR SHARE` 工作项行、插入时外键再 `KEY SHARE` 父行，还有启动即跑的周期任务）也会等锁。锁序测试起的是真实的应用（`startObservedApp`）。一个任务恰好在等时探针就返回，此刻请求可能还没走到咨询锁：接着的 `requestRowLock` 要求请求“正在等”，直接 `Fatal`；或者采到的是请求走到一半的状态。结果是偶发失败，而且失败的样子像产品的锁序错了。存储测试（没有应用）用它没有问题。 | `si/platform/postgres/pgtest/lockwait.go:14-31`（适用范围）、`:83-92`（按语句认等待者、按持有者认等待）、`:125-134`（三条查询）；`bs/request_locks_test.go:169-196`（请求探针：`wait_event_type='Lock'` 且有未授予的锁，咨询锁也算）；`bs/request_row_locks_test.go:36-41`；硬规则 §5 的 R10 一条 | bootstrap 那一段改用认得出“谁在等谁”的探针：持有者事务的连接已知，用 `WaitForStatementLockWaitBehind(pool, 持有者的 PgConn, 时限, 咨询锁那条语句)`（`pg_blocking_pids` 对咨询锁同样成立），或直接用请求探针（`readRequestLock` 已认任何未授予的锁）。spec §4、plan T10、brief 三处改口；`WaitForLockWait` 只留在 T5 的存储测试。 |
| I2 | plan 变异表“咨询锁”一行；spec §4“交错”的两格 | **“不取咨询锁”写的两个后果，一个不会发生，一个靠竞态。** ① “并发同一目标两行或 23505”：`INSERT … ON CONFLICT … DO UPDATE` 对着部分唯一索引本身就并发安全——后到的插入等先到的事务，再走 `DO UPDATE`，一行、不报错，与取不取咨询锁无关。“同一人同时记同一目标两次（一行、两个 204）”因此杀不死这个变异。② “同一人并发越过 20”：要两条 upsert 都落在对方裁剪的快照之前。持咨询锁的事务按不变式从不等行锁，夹具没有行可扣来把它停在临界区里；两个请求一起放行后谁先提交是竞态，变异时而存活。真正确定的杀法在别处：T10 锁序测试的第三段（持有者扣着咨询锁时请求必须在等，变异体不等、直接 204）与 T7 的调用序列。 | spec §2.3 的 upsert、§2.8“持咨询锁的事务从不等行锁”；sqlc 探针里的同一条语句；`nav/adapter/postgres/queries/favorites.sql:34-43`（同一张索引形状的先例）；硬规则 §5“只有能编译、能失败的变异才算” | 变异表这一行改为：后果只写“同一人并发越过 20”，杀它的测试写 T10 第三段与 T7；删掉“两行或 23505”。T13 的“19 条时同时记两个新目标（恰 20 条）”写明构造：一个持有者事务经存储的方法扣着咨询锁，两个请求都被看到等在它后面，放开后恰 20 条、两个 204——它是正向的结果，不当作这个变异的杀法。键的公式漏掉人或工作区只是多串行、结果不变，登记为等价变异，或在存储测试加一格“别人的、别的工作区的锁不等”。 |
| I3 | plan 变异表；spec §4 的“存储”“端口”“组合测试” | **几条可见与上限的性质没有杀得死的变异（硬规则 §5、设计 §9）。** ① **只列自己的、只列这个工作区的**：这是这个接口的隐私性质，变异表没有，T5 给列表查询列的测试只有次序；H17 有多个账户，但没有写“别人的行、他在别的工作区的行不在回答里”。② **先过滤后取 20**：未删的行不超过 20 条时，“先取 20 再过滤”与它结果相同（取 20 就是全取）。表里写的“藏起的行占名额时少列”正是两种次序答得一样的情形。只有 21 条未删行、且藏起的一条排在前 20 里才分得出。③ **裁剪删到只剩 20**：平时只多出一条，“只删最旧的一条”（Plane 的写法）与“删掉第 20 条之后的全部”等价；spec §2.5、§6 说回滚留下的第 21 条“到他下一次记录为止”，这句话没有测试。④ **锁下重读对工作项删除有两层**：地点的重读与可见的重读都带 `deleted_at IS NULL`，只去一层是等价变异；plan 只给项目删除写了“两层一起去掉、留对照”。⑤ **删除任务的第三步不改写已软删的行**：工作项先删（连带在那一刻软删它的访问行）、项目后删是真实的次序；第三步若不带 `deleted_at IS NULL`，会把那些行的删除时刻与删除者改成项目的，清理的时点跟着变。表里没有，Go 的删除守卫只查“没有未删的行”。⑥ 裁剪的“前 20”把已软删的行算进去（活的行被提前删掉）；列表的 `entity_type` 筛选被忽略；spec §4“端口”给地点写了“不读已删的”，给摘要没有写。 | plan 变异表 `:35-62`、T5 的测试清单 `:86`；spec §2.5（列表、上限）、§2.7（锁下重读）、§4“端口”；`nav/adapter/postgres/queries/delete_rows.sql:25-35`（`DeleteProjectFavorites` 带 `r.deleted_at IS NULL`）；硬规则 §5“要证明‘只有这一个’时准备两个”“锁下的重判有两层时…” | 变异表补行并各配测试：① “列表不按人限定”“不按工作区限定”——T5 存储测试与 H17 各一格（两个人访问同一目标、同一人在两个工作区）。② T7 的假存储交出 21 条、藏起的一条在前 20 里，写进 Task；表里的措辞改掉。③ T5 存储测试从 23 条裁到恰 20 条；同一组数据证明“下一次记录补回 20”。④ 表里“去掉锁下重读”一行写明两层一起去掉、留只去一层的对照。⑤ T4 存储测试加一条早先软删的行，断言时刻与删除者不变；表里加“第三步不带 `deleted_at IS NULL`”。⑥ 三条各加进表，spec §4“端口”补“摘要不读已删的”。 |
| I4 | spec §4“交错”的“裁剪 × 别的事务扣着的一行”“裁剪 × 子树删除”；plan T13 | **两格照字面做不出来。** ① “那个事务提交后不超过 20 条”：别的事务若只是扣锁（`FOR UPDATE`），提交后那一行还是未删的，这个人有 21 条，断言失败。要它成立，那个事务得像连带那样把行软删再提交。而“两个写都经真实的 app”在这里办不到：navigation 是 `issue.deleted` 的最后一个订阅者（`bs/events.go:165-175`），连带拿到行锁之后只剩提交，没有东西可扣。② “裁剪 × 不按全局 `id` 次序锁行的子树删除”照 P1c 的测试构造“行的 `id` 次序与子树次序相反”：全部已读一次锁很多行，裁剪在上限之下通常只锁一行（第 21 条），一行成不了环。要复现死锁得有两条以上“第 20 条之后”的行，只能靠 SQL 种出 22 条，或先造出回滚留下的第 21 条。经接口造不出这个状态，实施者会在这里卡住或写出一个什么都不证明的格。 | spec §4 `:169`、§2.5（上限、瞬时第 21 条）；`iss/app/delete_core.go:53-65`（行步骤全部做完才逐个发事件）；`bs/notification_mark_all_read_interleaving_test.go:82-112`（先例的构造：扣住第二个工作项的一条通知，把删除停在两个工作项之间） | ① 写明“别的事务”是测试的 SQL 事务，经存储的 `DeleteIssueRecentVisits` 软删那一行：提交 → 恰 20 条；另加回滚一格 → 库里 21 条、列表 20 条、下一次记录后 20 条（这也是 I3 ③ 的组合层证据）。② 写明两种杀法各要什么：“去掉 `SKIP LOCKED`”由“不等”杀——照先例扣住子树第二个工作项的一条通知，把删除停在两次连带之间，此时记录必须立刻答 204；要演示死锁（500）的一格用 SQL 种 22 条，两条多出的行分属子树的两个工作项、`id` 次序与子树相反。记录的目标要在删除所在项目之外，否则先被项目 S 挡住。 |

### 次要

| # | 位置 | 发现 | 建议 |
|---|---|---|---|
| m1 | spec §2.8“若裁剪会等……成环” | 写的那条环在代码里走不到：它要子树删除“扣着访问行、又等别的项目的锁”。删除先取齐项目锁（`iss/app/lock.go:179-183`），再做完全部行步骤，最后才逐个发 `issue.deleted`（`iss/app/delete_core.go:53-65`）；拿到访问行之后它不再等任何项目。真实的环是 I4 ② 那一种：多出两条以上时裁剪与连带直接互等。平时（只多出一条）等待的裁剪不成环，但会让持着咨询锁与项目 S 的记录挂在一个慢删除后面。结论（`SKIP LOCKED`）不变。 | 改写这一句；T15 要把它登记进硬规则 §4，先改对再登记。DP-12 的同一条链也不要引用。 |
| m2 | spec §6、plan“测试时间”的第 1 路门槛 | “分支 CI 第 1 路超过 960 秒先报告”落在 runner 的波动里：同一份代码分支 CI 938.8 秒、main CI 752.5 秒，相差 186 秒，离 960 只有 21 秒，P3d 一行不加也可能触发。本机基线写的是 P3c 检查点 2 的 649.0 秒，plan T1 记的终检是 655.4 秒。 | 判定用检查点 0 在本机实测的第 1 路作基线，多出 30 秒以上先报告。CI 另设一条绝对线作报警（例如 1020 秒，期限的 85%），并写明“CI 过线而本机增量不到 30 秒”记为 runner 波动、报告但不停。改路数、改期限仍要负责人同意。 |
| m3 | spec R6、§5 的 P7 交接 | 藏起的行照占上限，分诊的人连着打开 20 个待处理条目，他的 20 条全是不列的行，首页的“最近”变空，直到再访问别的东西。拒绝、标重复的条目永远不列，占位到被挤掉为止。R6 本身成立（`issue.read` 读得到条目，记录不拒）。 | R6 不动。§5 给 P7 加一句：收集箱页打开条目时不记录，只在工作项详情页记；理由写上面这一条。 |
| m4 | spec §4 H17“删除窗口（`holdNavigationDeletion`）”；plan T12 的种子 | 这个帮手扣的是项目的一条**活的视图**，没有视图时 `Scan` 失败（`bs/navigation_fixture_test.go:98-105`）。任务的步骤是视图 → 收藏 → 最近访问，扣住视图才停得住第三步。 | T12 的种子写明：删除窗口那个项目开着视图并有一条视图。 |
| m5 | spec R11；问题 8 的矩阵列表行 | 列表一行只查空集合，可以接受：读时过滤、只列自己的都归 H17（I3 ① 补上之后）。它的理由“不占副本”只说对一半：在模板库里种数据不占副本，代价是 `prepareMatrix` 里多一行调用，而那个函数在已 452 行的 `bs/permission_matrix_test.go` 里；按格准备则必须是 `write` 行（`bs/permission_matrix_coverage_test.go:84-86`）。 | 保持现状即可。想让这一行不空，便宜的做法是把它写成带 `prepare` 的 `write` 行（SQL 往格的副本里插一条对 `acme/public` 的访问）：访客列出它，管理员、成员不是那个项目的成员、答空。多 6 个副本，按现有 280 个 `write` 行约 85 秒折算不到 1 秒，不碰任何封顶的文件。 |
| m6 | spec §2.8、plan T5、brief：“sqlc 收就……不收就……” | 已实测：仓库钉住的 sqlc 收下 `pg_advisory_xact_lock(hashtextextended('user_recent_visits/' \|\| sqlc.arg(user_id)::uuid::text \|\| '/' \|\| sqlc.arg(workspace_id)::uuid::text, 0))`，生成 `LockRecentVisits(ctx, {UserID, WorkspaceID})`。 | 三处删掉“不收就在适配器里算”的分支，键的公式定在 SQL 一处；T5 不必再试。 |
| m7 | spec §2.8“（`iss/app/lock.go:12-37`）” | 这一段是 `Locks` 类型的注释。持锁的代码在 `Locks.run`（`:103-120`）与 `lockAndDecide`（`:179-183`），语句是 `pj/adapter/postgres/queries/issue_ports.sql:1-11`。 | 换成这三处。 |
| m8 | spec §4“交错”的“× 成员移除” | 通用断言（不留指向已删目标的活行、不答 500）对成员移除是空的：两种次序都不产生那样的行。 | 写出它自己的断言：移除先提交 → 记录 404、不写行；记录先提交 → 移除等它，行在、读时不列。 |
| m9 | plan“允许改的已有测试”的“接线”一条 | `newNavigationRoute` 写成“编译跟随”。`navigation.Deps` 是具名字段的结构体，加一个字段不需要调用处跟着改；另外两处只跑任务的 `navigation.New`（`bs/navigation_fixture_test.go:178`、`:197`）也不必动。 | 写清为什么要补：只有 `navigation.New` 会解引用空的 `Issues` 时才需要。T6 核实后二选一，记台账。 |
| m10 | spec §6“项目 S 的争用” | 只写了“互等毫秒级”。按 PostgreSQL 的实现，后来的共享行锁不在等待的 `NO KEY UPDATE` 后面排队；记录又比收藏频繁得多（每次打开页面一次）。同一个项目上的记录若持续重叠，工作项的写会等到写锁期限、答 503。v0 的量上不现实，本项没有在库上验证。 | §6 加一句，登记为已知风险；不加测试。 |
| m11 | brief“Commits” | 提交尾注写死了 `Co-Authored-By: Claude Opus 5.5`。实施者未必跑在这个模型上。 | 改为“用派发时会话给出的署名行”。 |

## 3. 逐项回答

1. **合同的每一项能不能在 `9d26b1c3` 上做出来**（都能）：
   - **迁移号**：`mig/sql/` 有 53 个文件，最后一个是 `00053_navigation_user_favorites.sql`，`00054` 空着。
   - **sqlc**：探针里四条语句都生成成功——`hashtextextended` 的咨询锁（`:exec`）、单个 `bigint` 键的写法、upsert、两段 `MATERIALIZED` CTE 加 `FOR UPDATE SKIP LOCKED` 的裁剪（`:execrows`）。`pg_advisory_xact_lock`、`hashtextextended` 都在 `pg_catalog`，对 `PUBLIC` 可执行，不进授权文件的守卫（它只看 `public` 模式）。见 m6。
   - **`ON CONFLICT` 的推断**：写出的谓词 `WHERE deleted_at IS NULL` 与索引的谓词逐字相同，和 P3c 的 `InsertFavoriteItem`（`nav/adapter/postgres/queries/favorites.sql:43`）同一形状；sqlc 肯编译，真实库上的推断没有试。
   - **索引名长度**：部分唯一 50 字节、连带索引 53、列表与裁剪索引 54，都在 63 以内；照列名全写的是 73，spec 的缩写是必要的。其余约束名最长 37。
   - **计数**：迁移 53 → 54 的三处（`mig/trigram_schema_test.go:44-45`、`:84`、`:106-107`），表名名单按字母序插在 `user_favorites` 与 `users` 之间（`:52-54`）；清理目录 36 → 37（`bs/purge_catalog_test.go:154-155`）；`TestPurgeIndexes` 36 → 37（`mig/purge_indexes_test.go:40-41`）；授权加在 `deploy/runtime-grants.sql:29`；sqlc 的 schema 名单在 `server/sqlc.yaml:209-211`。
   - **删除任务的步骤**：`steps` 的项目范围今天是视图 → 收藏（`nav/app/delete_rows.go:49-57`），加第三步、各自游标、共用预算做得到，种类与载荷不变，在途的旧任务升级后顺带删新表。`DeleteRowsDeps` 加一项，只有 `nav/app/delete_rows_test.go:100` 与 `nav/module.go:139` 两处构造。
   - **清理的次序**：模块次序是 asset、notification、navigation、issue、planning、project、workspace、identity（`bs/issue_retention_sets_test.go:23`），navigation 先于它的三个父表；表内放在 `user_favorites` 之后，两张表互不引用。
2. **锁**（不成环）：
   - **“工作项的每条删除路径整个事务持项目 N”属实。** 软删工作项的语句只有两条：删除核心的 `DeleteIssueRows`（`iss/adapter/postgres/queries/delete_issue.sql:13-22`）与项目删除任务的分批（`delete_rows.sql:8`）。前者只经 `deleteLocked`，它只有两个调用者（`iss/app/delete_issue.go:94-95`、`delete_intake.go:57-62`），都在 `Locks.run` 里；`run` 在一个事务里先 `lockAndDecide`（`iss/app/lock.go:113-118`、`:179-183`），取的是 `LockIssueProject` 的 `FOR NO KEY UPDATE`（`pj/adapter/postgres/queries/issue_ports.sql:1-11`）。后者在项目删除之后才跑、不发事件，由任务的第三步管。
   - **工作项删除（含子树）与连带**：子树在同一个项目里（`delete_issue.sql:1-11`），连带在发布者的事务里、项目 N 之下。删除先提交：记录等到项目 S，锁下重读见已删 → 404。记录先提交：删除等它，连带的语句在拿到锁之后取快照，看得到刚写的行。
   - **项目、工作区删除与任务**：项目删除改写项目行；工作区删除先取工作区 N、再按 `id` 取它每个项目的 N（`pj/adapter/postgres/queries/cascade.sql:94-105`）。都与记录的 S 冲突，次序同为工作区先于项目。任务不持父行的锁，只在项目已删之后跑，记录在锁下读到项目未删，两者不碰同一行。
   - **项目归档**：工作区 S → 项目 N，与记录同向；归档不动最近访问。
   - **成员变更**：工作区成员的移除、改角色、接受邀请都在工作区 N 之下（`ws/adapter/postgres/queries/members.sql:33-34`、`:67-68`）；项目成员的变更在项目 N 之下（`pj/adapter/postgres/queries/members.sql:8`、`:24`、`:46`、`:54`）；停用账户先取账户行 `FOR NO KEY UPDATE`、再取他的工作区 N（`ws/adapter/postgres/queries/deactivation.sql:1-13`），与插入时外键对账户行的 `KEY SHARE` 不冲突。都与记录串行。
   - **收藏的写**：工作区 S → 项目 S → 视图 S → 收藏行，与记录共享前两把、同向，不碰对方的表。
   - **同一人的并发记录**：先到的拿咨询锁，后到的持着自己的工作区 S、项目 S 等它；先到的不再等任何行，所以不成环。别的工作区是别的键、别的行。
   - **清理**：只删已软删的行，upsert 只可能撞未删的行，裁剪只选未删的行。
   - **两条不变式**：① 持咨询锁的事务从不等行锁——成立。upsert 撞的那一行只能是本人对这个目标的未删行；能锁它的另外三方（同一人的记录、这个工作项的连带、这个项目的任务）分别被咨询锁、项目 S、“项目未删”挡住。外键检查要的工作区、项目的 `KEY SHARE` 已被自己的 S 盖住；对账户行的 `KEY SHARE` 今天没有冲突方（全仓库对 `users` 只有 `FOR NO KEY UPDATE`）。② 裁剪 `SKIP LOCKED`——必要，理由见 m1、I4（spec 写的那条环不对，真实的环在多出两条以上时）。
3. **可见**：谓词恰是 `issue.read` 加 `HiddenFrom`。issue 自己读一个工作项就是这样判的（`iss/app/lock.go:294-308`：`Authorize`，再 `HiddenFrom`），收集箱里的条目读的也是 `issue.read`，没有另一条规则。`Readable` 就是在每个他是有效成员的项目上判 `issue.read`，含已归档的，不是成员的工作区管理员读不到（`si/modules/access/app/readable.go:40-75`）。项目按 `Readable`，与收藏、通知一致；公开项目的非成员 `GET /projects/{id}` 是 200、记录是 404，这是 DP-7 定下的。
   - **Triage 算得便宜也算得对**：未删的条目行每个工作项至多一条（部分唯一 `intake_issues_issue_id_key`，`mig/sql/00050_issue_intake_issues.sql:27`），`LEFT JOIN` 不会把行变多，至多 20 个 id、有索引。`InTriage()` = 有未删的条目行且状态不是 `accepted`（`iss/domain/issue.go:113-115`），拒绝、标重复都算在内。
   - **各状态下“能记、不列”**：待处理、暂缓——合理，接受之后出现。已接受——列。拒绝、标重复——永远不列，占一个位置直到被挤掉，可以接受，但要给 P7 留话（m3）。未接受的条目被删——发 `issue.deleted`，连带带走。已接受的条目被删——只删条目行，工作项与访问行照旧。
4. **连带**：发布 `issue.deleted` 的只有删除核心一处（`iss/app/delete_core.go:60-65`），对子树的每个工作项各发一次。
   - 子树：在内。未接受的收集箱条目：在内（同一个核心）。已接受的条目：不发，工作项还在，对。
   - 项目删除：issue 的分批任务不发事件，由 navigation 任务的第三步按 `project_id` 删。工作区删除：事件里列出它的每个项目，各投一个项目范围的任务（`nav/app/delete_rows_on_deletion.go:18-33`）；`project_id` 非空，所以不需要工作区范围的一步。
   - 没有找到漏掉的删除：仓库里没有批量删除工作项的接口，也没有别的语句软删 `issues`。
   - **订阅者的校验与发布者相符**：发布者交的是类型、实体 `issue` 与 id、工作区、项目、非空的执行者、时刻，变化为 `nil`（`iss/app/issue_events.go:14-17`、`delete_core.go:62`）。“执行者为空或非零”与 notification 的同类订阅者一致（`si/modules/notification/app/deletion_events.go:17-19`）；“没有变化”比它多一条，发布者满足。没有测试经带 navigation 订阅者的 dispatcher 发手写的 `issue.deleted`。
   - **按 `id` 加锁与 `SKIP LOCKED` 相容**：连带可能等裁剪锁住的行，裁剪的持有者不等；裁剪提交后那一行已物理删除，连带在等待之后重判、跳过它。
5. **红窗口**：
   - (b)、(c) 完整。`contract.Operations()` 的使用者共七个文件；与 navigation 标签有关的只有矩阵覆盖（`bs/permission_matrix_coverage_test.go:146`）与项目写锁表（`bs/project_write_locks_test.go:302`）。后者不红：两条路径都没有 `{project_id}`，矩阵行只用工作区列。issue 的删除窗口、写锁表与 planning 的访客行只看各自的标签。
   - (a) 的六个名字静态核不出是否齐全；plan 的做法（T3 跑一遍读删除目录的测试、名单之外的红先停）是对的。调研另点过的四个也跑一遍：`TestWorkspaceDeletionSubscriberRollsBackFullCatalog`、`TestDeletionInterleaving25`、`TestAWriteOnAProjectDeletedWhileItWaitsFindsNoProject`、`TestLateDeleteRowsAndPurgeRetry`。
   - **事件目录不红**：`TestIssueEventDirectory` 用不带订阅者的 dispatcher（`bs/events_test.go:201-244`）；`TestPlanningEventDirectory` 用 `navigation.Provide(nil)` 的订阅者，但只发 planning 的事件（`:262`），新订阅者不被调用。没有测试按名字或个数枚举 navigation 的订阅者。
   - **`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 不红**，理由同上。
   - **清理的运行与规模测试**：各有自己的表名单（`bs/purge_runtime_test.go:53`、`:113`、`bs/purge_scale_fixture_test.go:24`），T3 连同种子一起改，不留窗口。
   - **契约的自动守卫**：查询参数是枚举时不产生“绑定失败”的格（`si/platform/httpserver/apitest/operations.go:74-79` 明写枚举照绑），所以 422 由用例判与守卫不冲突；矩阵的目标规则只看路径参数，`/me/recent-visits` 没有参数。
6. **允许清单与文件大小**：清单盖住了每个 Task 要动的已有测试，没有发现缺口；与“不再加长”的约束没有相撞。
   - `bs/permission_matrix_test.go`（452）只在 `matrixRows()` 那一行就地加一项，做得到；`bs/permission_matrix_seeded_test.go`（430）、`_seed_test.go`（390）不必动——记录的路径没有 id，列表行不种数据。
   - `iss/app/ports.go`（395）不动，端口进新文件，可行。`nav/module.go` 154、`bs/navigation_ports.go` 114、`bs/app_modules.go` 168、`iss/module.go` 317，加上去都远在 400 以内。
   - `api/modules/navigation.yaml` 710 行，加两个操作、六个 schema 后约 900 行；契约文件没有行数上限（`issue.yaml` 约 3000 行）。
   - e2e：`fixtures/furnish.ts` 213、`fixtures/assert/catalog.ts` 118，够用；I21（411）、W3（404）不改。
   - 一处措辞见 m9。
7. **变异**：表里多数杀得死。杀不死或等价的见 I2（咨询锁）与 I3（先过滤后取 20、裁剪只删一条、锁下重读的两层）。缺的对照：只列自己的、只列这个工作区的；第三步不改写已软删的行；裁剪的前 20 不数已软删的；筛选被忽略；摘要不读已删的。“连带不带 `entity_type = 'issue'`”是等价的（工作项的 id 不会等于项目的 id），不必列。
8. **测试**：
   - **上限在并发下恰 20**：照现在的写法是结果测试，不是杀法（I2）。结果本身成立：咨询锁单独一条语句，后面的 upsert 与裁剪各取新快照，看得到先到者已提交的行。
   - **裁剪 × 子树删除**：要照 I4 ② 改写构造，否则做不出来。
   - **记录 × 工作项删除的两种次序**：证明得了。先让一方等在夹具扣着的项目行上、再让另一方排到它后面，次序由排队决定；“子工作项也一格”有意义，它证明连带是逐个工作项发的。
   - **桶**：四个新名 36、4、33、17，都小于 46，在第 0 路；`TestPermissionMatrix` 在桶 46、第 1 路，`TestThePermissionMatrixCoversEveryOperation` 在桶 15。
   - **矩阵的列表行不种数据**：可以接受，便宜的种法见 m5。
9. **契约**：
   - **形状**：`project` 必在、`issue` 可空，与 P1c 的 `oneOf: [{$ref}, {type: 'null'}]` 同一写法（`api/modules/notification.yaml:396-405`），生成器已经处理过它。
   - **`IssuePriority` 的搬移**：Go 的手写引用恰三处（`iss/adapter/http/issues.go:95`、`:105`、`intake_list.go:45`），都是类型转换，没有用到枚举常量；`issue.yaml` 的定义在 `:2196`、引用恰七处，行号与 spec 写的一致。网页有三个文件用到这个类型，都经 `@nerve/api-client` 导出的名字；打包后名字不变是按 `LogoProps` 的先例推断的，没有跑 `make gen`。
   - **各操作的码**：记录不答 `workspace.not_found`、两个操作都不答 `forbidden`，成立——三种角色全收，规则表里只有“三种之外的角色”才是 403，库里没有那样的角色；`listWorkspaceIssues` 是同样的声明（`api/modules/issue.yaml:404`）。`bad_request` 照例不写进 `x-problem-codes`。navigation 另声明一份 `issue.not_found` 有先例（它已有 `project.not_found`）。
   - **`entity_type` 的 422**：请求体的枚举由用例判是收藏的做法（`nav/adapter/http/favorite_refusals_test.go:93-118`）；查询参数用枚举有先例（`api/modules/issue.yaml:1538`、`:1594-1598`）。
10. **Plane 的事实**（抽查全部成立）：
    - D35：列表按模型默认的 `-created_at`（`pa/db/models/recent_visit.py:36`），淘汰也按 `created_at`（`pa/bgtasks/recent_visited_task.py:39-44`）。
    - D36：`== 20` 才删一条，没有唯一约束（DDL 只有主键，`tools/plane-schema/plane-v1.4.2-schema.sql:3822-3826`）。
    - D37、D38：十处 GET 调 `recent_visited_task.delay`；读出时只认 `issue`、`page`、`project`（`pa/app/views/workspace/recent_visit.py:33`）。
    - D39：序列化用不滤归档、不看权限的 `objects.get`（`pa/app/serializers/workspace.py:316-329`）。
    - D43：淘汰是软删（`recent_visited.delete()`），硬删任务里没有这张表。
    - D44：批量删除不清最近访问（`pa/app/views/issue/base.py:773-797`）；列表先取 20 再解析（`recent_visit.py:35`）。
    - 单个删除是硬删：`delete(soft=False)`（`pa/app/views/issue/base.py:716-727`）。项目归档只删收藏（`pa/app/views/project/base.py:429-441`）。列表给访客（`recent_visit.py:24`）。
11. **时间**：第 1 路的增量接近零，plan 做到了。四个新顶层名都在第 0 路。第 1 路变重的只有：`TestPermissionMatrix` 多 24 格、其中 18 格各起一个应用，按现有 280 个 `write` 行约 85 秒折算是 1–2 秒；`TestPurgeScale`（桶 48）、清理目录、几个删除守卫各多一张表、几行种子；每次删除工作项多一条走索引的 `UPDATE`。门槛的写法见 m2：本机的 +30 秒合理，基线应取检查点 0；CI 的 960 秒在 runner 的波动之内，会误报。

## 4. 核实为正确的事实（摘要）

- `00054` 空着；关键词的 phase 已是 `M7/P3`（`tools/keywords.json:2`）；bootstrap 现有 781 个顶层测试，没有重名。
- sqlc 肯编译 SQL 一侧的咨询锁键、upsert 与裁剪；三条部分索引的名字都在 63 字节以内。
- 发布 `issue.deleted` 的只有删除核心，每个子树工作项一次，变化为空、执行者非空；行步骤全部完成之后才发。
- 工作项不换项目：没有改 `issues.project_id` 的语句。
- 行锁强度：`FOR SHARE` 与 `FOR NO KEY UPDATE` 冲突，不与外键的 `KEY SHARE` 冲突；workspace、project 两个模块的成员写都取父行的 `FOR NO KEY UPDATE`。
- 请求探针认得出等咨询锁的请求；`WaitForLockWait` 认得出但不分是谁在等。
- 矩阵的目标规则只看路径参数；按格准备的行必须是 `write` 行；`projectOf` 对工作区的六列给出 `acme/public` 与 `gone/project`。
- spec 引用的行号抽查都对得上（m7 的一处是注释而不是代码）。
- Plane 一侧七条缺陷编号与“单个删除硬删”成立。

## 5. 需要负责人的事

没有。I1–I4 都是测试的写法与变异表；m2 的门槛数字是主控层面的选择，只有真要改路数或期限时才到负责人。

## 6. 处理（主控，2026-11-22）

全部采纳（m2 按主控的裁定，m5 取便宜的那种种法），改入 spec、plan 与 brief：

- I1：采纳。spec §4“锁序”的第三段改用 `pgtest.WaitForStatementLockWaitBehind`（持有者的连接、语句 `LockRecentVisits`），再由 `readRequestLock` 认出是请求自己的等待；写明 `WaitForLockWait` 与 `requestBlocked` 为什么不用（`bs/request_locks_test.go:141-150` 只认行与表的等待）。plan T10 写出持有者的造法（`WithinTx` 里经存储的方法，`postgres.Tx(ctx)` 给连接）；`WaitForLockWait` 只留在 T5 的存储测试；brief 的咨询锁一条同改。
- I2：采纳。变异表“咨询锁”一行只写“同一人并发越过 20”，杀法是 T10 第三段与 T7 的调用序列，删掉“两行或 23505”；T13 的“恰 20 条”与“同一目标两次”写明持有者事务加两个请求各自探针的构造，记为正向的结果。键少人或少工作区不登记为等价，T5 加一格“别人的、别的工作区的不等”杀它。
- I3：采纳，六项各配变异与测试。① 变异表加“列表只列自己的、这个工作区的”，T5 存储与 H17 各一格。② “先过滤后取 20”改为 T7 的假存储交出 21 条、藏起的一条在前 20 里。③ T5 用 SQL 种 23 条裁到恰 20、21 条加一个新目标裁到恰 20（愈合）；变异“只删最旧的一条”。④ 锁序一行写明工作项删除的两层一起去掉、留只去一层的对照。⑤ T4 存储测试加一条早先软删的行，spec §2.6 写明第三步只动未删的，H17 的连带加“先删工作项再删项目”；变异“第三步不带 `deleted_at IS NULL`”。⑥ 变异表加“前 20 数进已软删的”“忽略 `entity_type`”“摘要读已删的”，spec §4“端口”补摘要不读已删的。
- I4：采纳。spec §4“交错”与 plan T13 重写两格：① 扣行的是测试的 SQL 事务、经存储的 `DeleteIssueRecentVisits` 软删那一行，提交一格、回滚一格（库里 21、列表 20、下一次记录后 20）。② 扣子树第二个工作项的一条通知把删除停在两次连带之间，记录的目标在别的项目；“去掉 `SKIP LOCKED`”由不等的一格杀，另加死锁的一格（SQL 种成记录之后 22 条、两条多出的行 `id` 与子树次序相反，先收齐两个回答再断言，变异体 40P01 → 500）。死锁格与不等格共用世界，只多种一条访问、并指定两条的 `id`，本机约多 1–2 秒，所以做。
- m1：采纳。spec §2.8 改写：DP-12 的那条经别的项目的环走不到（`iss/app/lock.go:179-183`、`iss/app/delete_core.go:53-65`、`bs/events.go:165-175`），真实的环是多出两条以上时裁剪与连带互等；只多一条时等待不成环、但会挂住记录。plan T15 写明硬规则 §4 照改正后的登记。
- m2：采纳主控的裁定（与建议不同）。本机停止线：检查点 0 的本机第 1 路作基线（不用 P3c 的 649.0），多出 30 秒以上停下查。CI 的八成线保持 960 秒，是常设的报告规则、不停：过线时主控向负责人报告并附 runner 的分析（第 0 路非 bootstrap 各包的时间、与本机的对照），只有本机也过了停止线才停。不另设 1020 秒。spec §6、plan 的“测试时间”与 T2、brief 同改；P3c 阶段记录 §6 记下同一份代码 78.2% 与 62.7% 之差。
- m3：采纳。R6 不动；spec §5 给 P7 加一句：收集箱页打开条目时不记，只在工作项详情页记，理由照写。
- m4：采纳。spec §4 H17 的删除窗口与 plan T12 的种子写明那个项目开着视图并有一条视图。
- m5：采纳便宜的种法。列表一行改为带 `prepare` 的 `write` 行（先例 `bs/permission_matrix_issue_lists_test.go:15`、`:17`），SQL 往格的副本插一条本格调用者对 `acme/public` 的访问，访客列出它、管理员与成员答空；spec R11 的理由改写（6 个副本，第 1 路不到 2 秒，准备在新文件，`prepareMatrix` 不改）；变异表“矩阵”加“列表不按可见过滤”。
- m6：采纳。spec §2.8、plan T5、brief 删掉“不收就在适配器里算”，键的公式定在 SQL 一处，注明 re-92 已实测。
- m7：采纳。spec §2.8 改指 `Locks.run`（`iss/app/lock.go:103-120`）、`lockAndDecide`（`:179-183`）与 `pj/adapter/postgres/queries/issue_ports.sql:1-11`。
- m8：采纳。spec §4“交错”的成员移除写成项目成员移除（`removeProjectMember`，项目 N），两种次序各有自己的断言。
- m9：采纳。plan 允许清单的“接线”一条写清只有 `navigation.New` 在构造时解引用空的 `Issues` 才补桥，`bs/navigation_fixture_test.go:178`、`:197` 不动；T6 核实后二选一、记台账。
- m10：采纳。spec §6“项目 S 的争用”加一句，登记为已知风险（未在库上验证），不加测试。
- m11：采纳。brief 的提交尾注改为“用派发时会话给出的署名行”。
- §3 第 5 条（红窗口 (a) 另跑调研点过的四个测试）：采纳，写进 plan 的红窗口 (a)。
