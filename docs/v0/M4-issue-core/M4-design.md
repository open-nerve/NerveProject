# M4 工作项核心：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M4 工作项核心（`docs/v0/M4-issue-core`） |
| 日期 | 2026-10-16 |
| 状态 | 第三稿，已按聚焦复核修订（17.3）。第 10 节的八个决策点都已由负责人裁定（2026-10-17，全部按建议：1 选 B，2 至 7 选 A，8 选 B）；第 11 节的十一个架构问题都已由负责人批准（2026-10-17）；裁定记在 17.4。P1 在实施之前由控制者按决策点 2 的预先批准拆成 P1a、P1b（2026-10-17，第 12 节） |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.1、3、4.2、5、6、7.7、8.2、9；[差异清单](../plane-diff.md)；[前端改动清单](../frontend-changes.md)；[M2 设计](../M2-auth/M2-design.md)、[M3 设计](../M3-workspace-project/M3-design.md)（它们定下的平台约定和加锁约定约束 M4，本文照做，不重新裁定；要修订的写在第 11 节） |
| 前置交接 | `handoffs/` 中的 8 份：M3-closeout（9 节）、M2-closeout（13 节）、M1-closeout、M1-P2、M1-P3、M1-P4、M1-P5、M0-P4。逐条落到 Phase，见第 13 节 |
| 设计评审 | 第一稿（`b2ca1de6`）经独立评审（2026-10-16）：Ready with fixes，Critical 0、Important 8、Minor 26，评审另在 `postgres:18.6` 上做了 S1、S4、S5 和 S2 的一部分；第二稿按控制者对每条发现的裁定修订，逐条的落实在 17.1。第二稿（`e8256b58`）经对抗性评审（2026-10-16）：Ready with fixes, but not ready for the owner as written，Critical 0、Important 7、Minor 22，评审另做了 SP1–SP3（0.3）；第三稿按控制者的裁定修订，逐条的落实在 17.2。第三稿（`b33bfd9c`）经聚焦复核（2026-10-16）：Ready with fixes，Critical 0、Important 2、Minor 9，17.2 的 30 行中 25 行完整、5 行部分，复核另做了 SP4（0.3）；第三稿按控制者的裁定再修订，逐条的落实在 17.3 |

---

## 0. 目标与范围

### 0.1 目标
M4 是 v0 中最大的一个领域：工作项和它下面的一切（父子、关联、链接、评论、表情回应、操作动态、描述的历史、草稿、归档），加上列表引擎和搜索。它也是第一个有异步副作用、第一个有业务定时任务的里程碑。M4 结束时：
1. 任何调用方都能通过接口创建、查看、修改、删除、归档、恢复工作项，管理父子、关联、链接、评论、表情回应，查看操作动态和描述的历史，保存和发布草稿，按编号取工作项、搜索工作项；用 PAT 能做的事和页面一样多。
2. **列表引擎**：项目的工作项列表、已归档的列表和个人主页的列表在服务端完成筛选、分组、子分组、排序和按游标分页（总体设计 3.3、3.4）。一条 SQL 返回每组的第一页和总数；每种排序有自己的游标载荷，前端以同一个全序在本地重排。
3. **加锁约定延伸到工作项**：工作项和它下面的每个写只锁项目行（`FOR NO KEY UPDATE`），不锁工作区行（决策点 1，负责人裁定 B）；锁之前一个不加锁的预检挡掉一切会被拒绝的写，取连接之前一个按项目的写闸门挡住一个热的项目耗尽连接池（3.5，架构问题 11）；删除工作区先按 `id` 升序锁住它的项目；一个改变一个时刻对工作项一侧的行同样成立（3.5、3.6）。
4. **删除项目、工作区的工作项一侧由后台任务完成**（决策点 7，负责人裁定 A）：请求的事务做 M3 的连带、发布事件，工作项一侧的行由 River 的任务分批软删除，写下请求的时刻和删除者；v0 的规模目标是一个项目至多约 2 万个未删除的工作项（3.6、3.16）。
5. **权限框架的扩展**：规则表加入"创建者本人"（`AllowCreator`）和访客只看自己创建的工作项（`OwnOnly`，`guest_view_all_features`），一条访客规则用在每个读、写和列表上（3.4）。
6. **领域事件与第一个异步订阅者**：`shared` 有事件的值和 `EventPublisher` 端口；工作项的用例在事务里发布事件；链接的元数据抓取是第一个订阅者，经 River 的任务和防 SSRF 的出站客户端完成；`project`、`workspace` 发布 `project.deleted`、`workspace.deleted`，`issue` 订阅它们、投递删除工作项一侧的任务（决策点 3、7，3.19、3.11、3.6）。
7. **定时任务**：自动归档、描述历史的保留、60 天物理清理（覆盖 M2、M3、M4 的表），经 River 运行；运行中的 nerve 可以在测试环境里把时钟拨快，时间流逝的故事不等真实的时间（3.20、3.21）。
8. 前端把工作项的类型、service、store 换成生成的客户端和类型，按总体设计 7.7 分代，列表的分页、分组扩展 `core/lib/reconciled.ts`；M4 能到达的页面挂载时不请求 M5 及以后的旧接口。
9. 本领域的每个用户故事都有端到端测试：页面、数据库、异步结果三层断言；接口版本用 PAT，调用同一组数据库断言。持续集成的全部门禁通过；交给 M4 的交接全部关闭；给 M5–M8 的交接写清接收条件。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 工作项：创建、查看、修改（标题、描述、状态、优先级、负责人、标签、起止日期、父工作项、排序值）、删除（连同子树）、按编号取（`PROJ-12`）；默认状态、默认负责人、编号、完成时间 | 附件、描述中的图片、`attachment_count`（M5）；`Issue.cycle_id`、`module_ids` 和按迭代、模块的筛选、分组、排序（M6）；关注和取消关注的接口、`is_subscribed`、通知（M7） |
| 父子：设父、取消父、批量加子工作项、子工作项列表；不成环 | 收集箱、分诊状态的工作项、收集箱的来源（`IN_APP`、`FORMS`、`EMAIL`，M7）；自动归档中迭代、模块、收集箱的条件（M6、M7） |
| 关联：关联、重复、阻塞、被阻塞四种，可以跨项目（同一工作区） | 保存的视图、工作区视图和跨项目列表 `GET /workspaces/{slug}/issues`（M7）；工作区一级的显示设置的筛选列（M7） |
| 链接：增删改查，元数据由后台任务抓取（防 SSRF） | Webhook、接口调用日志、对外的事件投递（M8；M4 只发布事件） |
| 评论（增删改查，没有回复，决策点 6）、工作项和评论的表情回应 | 工作项快照 `issue_versions`（决策点 5，负责人裁定 A：不做） |
| 操作动态：M4 的各类变更写进 `issue_activities`，按游标列出；描述的历史版本：写入、合并、列出、查看，还原用普通的修改 | 评论的回复（决策点 6，负责人裁定 A：不做） |
| 草稿：个人草稿的增删改查，发布为工作项（与创建同一条代码路径） | 批量删除、批量归档、`issue-dates`（M1-P3、M1-P2 交接：前端没有入口，不提供） |
| 归档、恢复、已归档的列表；项目的自动归档（每天） | 账户的物理删除（v0 不做，M2 设计 4.7） |
| 列表引擎：项目的列表（含已归档）、个人主页的三个标签页；筛选、分组、子分组、排序、游标；项目的工作项显示设置（筛选、显示方式、显示属性）存在 `project_user_properties` | 已归档、个人主页、草稿页的显示设置照 Plane 只存在浏览器（`localStorage`），不进服务端 |
| 搜索：命令面板、选父工作项、加子工作项、选关联的工作项；按编号精确匹配，标题按 pg_trgm 模糊匹配 | 命令面板中迭代、模块、视图、收集箱的搜索结果（M6、M7 随各自的接口加回） |
| 领域事件、River 的事务内投递、两个异步订阅者（链接元数据；删除项目、工作区之后的工作项一侧，决策点 7）；自动归档、描述历史的保留、60 天物理清理三个定时任务；测试环境的时钟偏移；工作项的写的闸门（3.5） | 命令行的"只投递" River 客户端接进命令的组合（M4 没有投递任务的命令，M2 交接第 1 节原样转交，13.2） |
| 前端：工作项的类型、service、store；四种布局、筛选、显示设置、拖动、快速添加；详情、浮层、创建弹窗、子工作项、关联、链接、评论、表情、动态、描述历史、草稿、归档页、个人主页的工作项、命令面板的工作项命令和搜索、`/browse/PROJ-12`；M4 领域的死成员、死 prop 和 oxlint；M3 交接的键盘、弹窗、下拉框各项 | 迭代、模块、视图、收集箱的页面（M6、M7）：迭代、模块、项目视图、工作区视图的工作项列表（它们的 store、布局根和页面上的列表部分）在 M4 删除（P10a，7.3），M6、M7 照 Plane 的源码在新的列表引擎上加回；收集箱的组件改到能编译 |

### 0.3 关于文中的 spike
写第一稿时做了六份只读的调研（不进仓库，结论和关键的行号都已写在正文中）：Plane 的表（R1）、Plane 的业务规则（R2）、操作动态的格式（R3）、列表引擎（R4）、Nerve 后端的现状（R5）、前端的测量（R6）。第一稿的独立评审在两个临时的 `postgres:18.6` 容器上做了 S1、S4、S5 和 S2 的一部分（Go 程序用仓库的 `go.mod`，即 River v0.47.0、pgx v5.11.0；sqlc v1.31.1 经 `server/tools/go.mod`）；程序和输出不进仓库，结论写在下面和正文的各节。编号与冒烟故事 S1、S2 重名，文中按上下文区分：说迁移版本、`REQUESTS`、`watchPage` 的是冒烟故事。
- **前端的测量**（第 1 节、7.10）在仓库之外的副本上做（`git archive` 的副本，链上主检出的 `node_modules`）：M1 收尾的 `deadsym.mjs`、`domains.mjs`（全文在 [M1 收尾计划](../M1-frontend-trim/plans/closeout.md)附录 A），按 `domains.mjs` 分领域统计 oxlint，按文件组数行数，`git grep` 数旧接口的调用、store 和类型的使用方。校准：`deadsym.mjs` 得到 `prop 359, member 1000, export 2`，`--rows M3` 227 行，与 M3 收尾记录的头上相同（`docs/v0/M3-workspace-project/specs/closeout.md:985`）；oxlint 各包的条数等于各包的上限。
- **S1 列表的 SQL（已做）**：
  - 可行，要改一处写法：sqlc 拒绝多个数组的 `unnest`（"function unnest(unknown, unknown) does not exist"）；`ROWS FROM (unnest(@a::uuid[]), unnest(@b::text[])) AS t(id, key)` 可以，它的列被定为 `interface{}`，只用来连接，或在取出之前转型。3.16 的其余写法都能生成。
  - 总数放在每一行的窗口列里：`UNION ALL` 把"行"和"总数"拼在一起、`GROUPING SETS` 都能生成，但运行时 pgx 不能把 `NULL` 扫进非空类型的列。
  - 正确：21 个排序码 × 5 种筛选，每页 7 行翻完；多值的组；144 个格的子分组，都与逐行的比较相同。
  - 性能：按 CTE 里的键 `CASE` 排序用不上索引，每个请求扫描、排序这个项目满足筛选的全部行。1 万个工作项 2–100 ms。10 万个工作项，改写三处之后（`filtered AS MATERIALIZED`、窄的 CTE、每一级的总数用窗口）：默认的一页约 55–125 ms，`label_name` 约 210–430 ms，按 `state_id`、`label_id` 分组约 0.2–0.5 s、0.45–0.7 s，泳道约 0.7–1.0 s（机器有负载，误差 ±20–30%）。JIT（每次 150–500 ms 的编译）和预备语句的通用计划都让它更慢。
  - 一个陷阱：Go 的 `nil` 切片作为 `NULL` 传出，`cardinality(NULL) = 0` 的谓词滤掉全部行。
  - 写法、规模的目标和连接的参数写进 3.16；P5 不再有 spike 的任务。
- **S2 pg_trgm（做了一部分）**：sqlc 只在扩展的迁移列在模块的 `schema:` 里时才把 `similarity()` 定为 `float32`，否则静默地是 `interface{}`；名为 `q` 的参数与接收者同名，生成为 `q_`；10 万行上 `'%a3f9%'` 走位图索引扫描（0.09 ms）、`'webhook'` 12 ms，两个字符的 `'ab'` 退到顺序扫描（45 ms）；只有 `SELECT`、`USAGE` 的角色经 PUBLIC 的 `EXECUTE` 能调用 `similarity()`，授权测试的预期不变。迁移角色在镜像里建扩展、中文的查询（多是两个字）的耗时留给 P8 的第 6、12 个任务（3.18）。
- **S3 bluemonday（未做）**：放行表是局部的，在 P4（描述、评论、清洗到来的 Phase）的第 1 个任务做；样例除了编辑器每种节点的输出，还要有经接口写入的 HTML（列表、标题、嵌套的标记）和编辑器重新序列化之后的结果（3.7）。
- **S4 River（已做）**：只投递的客户端（`&river.Config{}`，没有 `Workers`）在事务里 `InsertTx` 一个没有登记 worker 的种类可以，回滚不留、提交留下一条；以空的 `river.NewWorkers()` 建的工作客户端在 `AddWorker` 之前 `InsertTx` 失败（`UnknownJobKindError`，`river@v0.47.0/client.go:2270-2277`），`NewClient` 之后 `AddWorkerSafely` 可以（共用同一个 `*Workers`，可用但没有写进文档），`Start` 之前 `PeriodicJobs().Add`、`InsertTx` 都可以；一个 worker 都没有时 `Start` 失败。组合因此用只投递的客户端投递（6.6）。
- **S5 取锁的等待（已做）**：E1：方案 A 之下，新的 `FOR SHARE` 在等待中的 `FOR NO KEY UPDATE` 之前 1 ms 得到；60 ms 一个、每 30 ms 新来一个的 `FOR SHARE` 持续 3 秒，工作区的 N 等了 2.753 秒（M3 17.4 的探针 B 复现）。E2：方案 B 之下，N 的等待者先来先得，3 秒首尾相连的工作项写之后，删除工作区只等了 319 ms（排在它前面的写）。E3：工作项写的链进行中，10 次 `UPDATE workspaces` 最长 2 ms；插入经外键取工作区行的 `FOR KEY SHARE`，与 `FOR NO KEY UPDATE` 不冲突。E4：`… WHERE id = ANY($1) ORDER BY id FOR NO KEY UPDATE` 按 `id` 的顺序取锁。E5、E5b、E6：约定七和关联行按 `id` 先锁的论证（3.5）。这些结论是决策点 1 的依据；交错 20、21 是它们的回归测试。
- 第二稿的对抗性评审另做了三个 spike（同样在临时的 `postgres:18.6` 上，pgx v5.11.0，程序和输出不进仓库）：
  - **SP1 连接池**：10 个连接的 pgxpool，每个写取项目 N、取号、插入工作项和一条动态、3 ms 的 Go 计算、提交，另一个探针每 20 ms 经同一个池读另一个项目。10 个写分在 10 个项目：每秒 2375 个，探针约 0 ms；40 个写在同一个项目：每秒 236 个，写的中位 155 ms，**探针 141 ms**；200 个（一个 PAT 的突发额度）在同一个项目：每秒 328 个，写 887 ms，**探针 875 ms**；200 个分在 10 个项目：写和探针都约 90 ms。一个热的项目让整个实例的请求排在池的队列里，A、B 两个方案都一样（3.5 的写闸门，架构问题 11）。另一个结果：经外键取账户行 `FOR KEY SHARE` 的插入首尾相连 3 秒时，`UPDATE users SET display_name` 等 1 ms，`UPDATE users SET email`（全表唯一的列，取 `FOR UPDATE`）等 2.913 秒（3.5）。
  - **SP2 删除的连带**：4.2–4.14 的索引和 P9 的清理索引，每个项目 2 万个工作项，每个工作项 20 条动态（五分之一带约 1 KB 的描述）、2 个负责人、2 个标签、3 个关注者、2 条评论、3 个描述版本；按项目（工作区）一条 `UPDATE … SET deleted_at, updated_at, updated_by_id … WHERE … AND deleted_at IS NULL`，一个事务、回滚：一个项目（66 万行）**8.6 秒**（动态 6.1 秒），五个项目的工作区（330 万行）**36.5 秒**（动态 25.7 秒）。不含关联、链接、提及、表情和 M3 自己的步骤；机器是笔记本上的 Docker，按 ±2–3 倍读。请求的期限是 15 秒（`server.request_timeout`），所以在规模目标上，同步的连带删不掉一个工作区（决策点 7）。
  - **SP3 出站的地址表**：`net/netip` 的判断加 6.8 原来的显式表放过了 `192.0.0.192`、`192.0.0.0/24`、`198.18.0.0/15`、`240.0.0.0/4`、`192.0.2.0/24`、`198.51.100.0/24`、`203.0.113.0/24`、`192.88.99.0/24`、`100::/64`、`2001:db8::/32`、`64:ff9b:1::/48`（6.8 改为写出 IANA 的整张特殊用途登记表）。
- 第三稿的聚焦复核另做了一个 spike（同样的环境）：
  - **SP4 分批删除的语句**：`issue_activities` 形状的表带 4.11 的索引，`uuidv7()` 的 `id` 让各项目在 `id` 的顺序上交错，每个项目 40 万行（SP2 的形状）。第三稿的分批语句（`WHERE id IN (SELECT id … WHERE <范围> AND deleted_at IS NULL ORDER BY id LIMIT 1000)`）每批都从最小的 `id` 走过前面各批已删除的行，代价随已做的行数增长：一个项目 **53.9 秒**（每批 7 → 295 ms）；同一个循环加键集（`AND id > $after`）**1.8 秒**，每批一直约 3 ms。按工作区的一个任务在五个项目（200 万行）上、每次 60 秒（River 的默认期限）：第一次做了 67.7 万行，第二次只有 28.7 万行，余下的 103.6 万行又要 385.7 秒，这一张表约 506 秒，约 9 次超时，加上 River 的退避约 2.4 小时。3.21 的清理语句同一个形状：80 万行 31.4 秒（每批 8 → 113 ms）。3.6、3.21 因此改为键集，任务另有时间的预算（决策点 7）。

---

## 1. 现状基线（2026-10-16 实测，`main` 为 `53a6a05b`）

| 项 | 数值与出处 |
|---|---|
| 服务端的模块 | `identity`、`workspace`、`project`、`access`、`instance`（`server/internal/modules/`）。手写的生产代码 15,657 行、测试 33,082 行、生成的 13,232 行；最大的手写生产文件 `platform/httpserver/bodyshape/bodyshape.go` 382 行，都在 400 行以内 |
| 迁移 | 14 个：`00001`–`00004` 是 `identity`，`00005` 是 River，`00006`–`00009` 是 `workspace`，`00010`–`00014` 是 `project`（`server/migrations/sql/`）；下一个是 `00015`。`server/sqlc.yaml` 有三个条目；`projects.last_issue_sequence` 已建、没有读写（`00010_project_projects.sql:29`） |
| 接口 | 60 个操作：`identity` 13、`instance` 2、`workspace` 19、`project` 26（`api/modules/*.yaml`）；没有 `format: date` 的字段，没有 map 型的结构 |
| 项目一侧的取锁 | `project/app/lock.go`（273 行）的 `lockAndDecide`、`lockRowAndDecide` 是 `project/app` 私有的；项目级的写最先以 `FOR SHARE` 锁工作区行（方案 E，`lock.go:173-201`）。别的模块不能调用它（`archtest/rules_test.go:39`） |
| `project.Provide` | 只提供 `ProjectAccess`、`ProjectMembershipCounts`（`project/module.go:87-96`）：没有项目的锁、取号、状态和标签的目录、成员的角色 |
| 删除的连带 | `project/app/deletion.go:11-20` 的步骤：项目 → 项目成员 → 项目显示设置 → 状态 → 标签，每张表一条 `UPDATE`（`queries/cascade.sql`）；删除工作区在工作区 N 之后读时刻（`workspace/app/delete_workspace.go:59`） |
| 权限 | `access/domain/rules.go` 39 行规则；没有 `AllowCreator`、没有创建者的事实；`ProjectFacts` 只读 `workspace_id, network, member_role`（`project/adapter/postgres/queries/access.sql:1-8`），不读 `guest_view_all_features`（列在 `00010_project_projects.sql:22`） |
| 事件与 River | 非测试的 Go 代码里没有事件；`platform/jobs/jobs.go`（167 行）只把 River 客户端的 `Start`、`Stop` 交出去（`:48-51`），没有 `Insert`、`InsertTx`；默认队列 `maxWorkers = 2`（`:38`）；唯一的任务是 `identity.cleanup_expired_sessions`（`identity/adapter/river/cleanup.go:16`），`bootstrap/app.go:150` 只传 `ident.Jobs()` |
| 时钟 | `clock.System{}` 写死在组合根里（`bootstrap/app.go:111,116,127,159`）；配置没有时钟（`server/configs/config.yaml`、`config.test.yaml`）；`jobs` 只有 `shutdown_timeout`（`config.yaml:77-79`） |
| 请求体的结构检查 | `bodyshape` 只认 `time.Time`、`uuid.UUID` 两种格式（`bodyshape.go:52-60`），生成器遇到 `format: date` 失败（`server/tools/bodyshapegen/generate.go:145-152`）；问题的路径按字典序排序（`bodyshape.go:189-191`） |
| 页大小 | `PageSize`、`DefaultPageSize = 50`、`MaxPageSize = 100` 在 `identity/domain/api_token.go:124-140`，唯一的分页列表是 PAT |
| 请求的期限与连接池 | `server.request_timeout: 15s`（到期取消请求的 context，数据库调用随之结束）、`database.max_conns: 10`（`server/configs/config.yaml:15,27`）；503 `server_busy` 已有（`shared.ServerBusy`，`shared/error.go:147-150`），M2 的密码哈希用它：名额满时最多等 `auth.password.max_wait`（`config.yaml:54-56`，`identity/adapter/argon2/hasher.go:159`） |
| CSP | `img-src 'self' data: blob:`，没有 `worker-src`、`frame-src`（`server/internal/platform/webui/csp.go:24-48`） |
| Plane 的表 | `issues` 28 列、`issue_assignees` 10、`issue_labels` 10、`issue_relations` 11、`issue_links` 12、`issue_subscribers` 10、`issue_mentions` 10、`issue_comments` 20、`issue_reactions` 11、`comment_reactions` 11、`issue_activities` 20、`issue_versions` 33、`issue_description_versions` 15、`draft_issues` 24 加 4 张关联表（`tools/plane-schema/plane-v1.4.2-schema.sql`）。`issues` 没有任何唯一约束和三元组索引（R1 第 1、19 节） |
| M4 领域的前端代码 | 564 个文件、54,644 行（不含测试；含 `@nerve/shared-state` 581 个、56,570 行，另有 `@nerve/editor` 179 个、18,392 行）：stores 33 个、8,428 行，services 11 个、905 行，组件 375 个、36,163 行，路由 24 个、1,485 行，hooks 23 个、2,034 行，类型、常量、工具 98 个、5,629 行。是 M3 开始时的 2.5 倍文件、2.3 倍行数（R6 第 1 节） |
| M4 要替换的旧接口调用 | 51 个 service 方法（7 个没有调用方，可以直接删）：`issue.service.ts` 19、`workspace_draft.service.ts` 6、`issue_reaction.service.ts` 6、`issue_archive.service.ts` 4、`issue_comment.service.ts` 4、`issue_relation.service.ts` 3、`project.service.ts` 3、`work_item_version.service.ts` 2、`workspace.service.ts` 2、`issue_activity.service.ts` 1、`user.service.ts` 1（R6 第 2 节） |
| 模块级的 service 实例 | M4 领域 13 处（M2 交接第 12 节的同一份）：`IssueService` 2、`WorkItemVersionService` 2、`WorkspaceDraftService` 1、`ProjectService` 3、`WorkspaceService` 5（R6 第 4 节） |
| store 和类型的使用方 | `useIssueDetail` 105 个文件、`useIssues` 62、`useCommandPalette` 32、`useIssueStoreType` 15；`TIssue` 124、`EIssuesStoreType` 93、`IIssueDisplayProperties` 52、`IIssueDisplayFilterOptions` 40、`TFilterProperty` 40（`grep -rlw`，R6 第 3 节） |
| 死成员和死 prop | `domains.mjs --rows M4` 545 行（成员 426、prop 119），另有按路径归到 M3 的 52 行（`workspace-draft-issues/base.ts` 16、`types/src/workspace.ts` 24、`project/projects.ts` 9、`workspace.service.ts` 3），共约 597 行（R6 第 5 节） |
| oxlint | 按 `domains.mjs` 的划分 M4 领域 197 条、分布在 89 个文件（web 131、editor 63、utils 2、constants 1）：`no-shadow` 74、`no-static-element-interactions` 33、`click-events-have-key-events` 27、`exhaustive-deps` 14 等。全仓 `no-shadow` 128 条，M4 的文件组占 108 条（R6 第 6 节） |
| 关键词守卫 | `tools/keywords.json` 72 条规则、23 个例外，`phase` 是 `M3/closeout`；到期的：`/user-properties/` 两处 `until: "M4"`（`keywords.json:2526`），`copy-hook` 10 个 `until: "M4/closeout"`（`:2533-2597`）；e2e 的 `workItemFilters`（`e2e/fixtures/mounts.ts:56`）和 `EMOJI_CHECK_WARNING`（定义在 `e2e/fixtures/browser.ts:166`；29 个故事文件 96 行，另有 `e2e/fixtures/settings-pages.ts` 一处） |
| 端到端 | 52 个故事文件（identity 17、project 14、smoke 9、workspace 12）；`e2e/fixtures/` 没有工作项的帮助函数、没有 `clock.ts`；`workspaceTables`（`e2e/fixtures/assert/workspace.ts:222-231`）只有 M3 的八张表；`expectProjectDeleted` 从目录读出每张有 `project_id` 外键的表（`e2e/fixtures/assert/project.ts:190-228`） |

---

## 2. 用户故事（M4 的验收范围）

每个故事一个 Playwright 测试文件，放在 `e2e/stories/issue/`（I1–I22）。按总体设计 8.2 断言。

**约定**：
- **三层断言**：页面、数据库、异步结果。M4 的异步结果有三种：链接的元数据（I10）、自动归档（I16）、物理清理（I21）；用"限时轮询数据库"等结果，不用固定的等待。其余故事的"异步"一栏为空（M4 没有通知和 Webhook，事件只有链接一个订阅者，3.19）。
- **接口版本**：一律用 PAT；页面版本和接口版本调用同一个断言函数（`e2e/fixtures/assert/issue.ts`，按表分函数；动态和描述版本另放 `assert/history.ts`）。
- **前置数据**经接口准备，不写 SQL：建工作项、改工作项、加子工作项、加关联、加评论、存草稿的帮助函数放在 `e2e/fixtures/` 下按资源分的文件里（P1a 起：`issues.ts` 建工作项，`states.ts` 建、读状态），都走生成的客户端。**访客建的工作项**（I17、I20 要它）经接口的路径是：他以项目成员的身份建工作项，再经 `updateProjectMember` 把他改为角色 5（访客）；接口里访客不能建工作项（3.4）。
- **`furnishIssueRows(api, token, projectId, assigneeId, labelId)`**（`e2e/fixtures/furnish.ts`，P1a 起）：经接口给一个项目的 `issue` 模块的每张表各造至少一行，负责人和标签由调用者给出，随每个加表的 Phase 长大（P1a 工作项、负责人、标签、关注、动态；P3 关联；P4 评论、表情、提及、描述版本；P7 链接；P8 草稿）。M3 的删除故事在删除之前调用它（`e2e/stories/project/p4-archive.spec.ts:179,249`、`e2e/stories/workspace/w3-workspace-settings.spec.ts:92,168`；W2 和 W3 的页面版本经 `furnishWorkspace`，`e2e/fixtures/furnish.ts:46`）：`expectProjectDeleted` 要求每张有 `project_id` 的表都有随项目删除的行（`e2e/fixtures/assert/project.ts:190-214`），`expectWorkspaceDeleted` 对 `workspaceTables` 的每一张同样要求（`e2e/fixtures/assert/workspace.ts:230-279`），两者的表由 `e2e/fixtures/assert/catalog.ts` 的 `tablesUnder` 从目录读出，新表一建出，这些故事不造行就失败（3.6）。`workspaceTables` 照 `projectTables` 从目录读出（有 `workspace_id` 外键的表），不再手写（P1a 起）。
- **故事随 Phase 长大**：一个故事的文件在它的核心断言所要的表和操作都已存在的第一个 Phase 加入（最后一列：接口版本 / 页面版本）；之后的 Phase 扩充同一个故事，断言和前置数据随之增加，"断言的增长"一列写明哪些断言在哪个 Phase 加入。每个 Phase 的完成线只写那时已有的断言，扩充的步骤是那个 Phase 的任务（第 12 节）。
- **时间流逝**的故事（I16、I21）用 `e2e/fixtures/clock.ts`（3.20）：先在一个新库上用正常时钟的 nerve 准备数据，停下它，再在同一个库上以时钟偏移起一个 nerve，等定时任务的结果。同一个库上只有一个 nerve 在运行，River 的选主不会落到时钟不对的进程上。
- 每个有页面的故事都用 `watchPage`（`e2e/fixtures/browser.ts`）断言：没有失败的接口请求、没有发往旧接口的请求（`oldApiRequests`）、控制台安静、没有 CSP 违规（3.1）。故意触发的失败（403、409 等）按请求逐个放行。
- 下表中 403 不带码的都是平台码 `forbidden`，422 不带码的都是 `validation_failed`（5.3）。

| 编号 | 故事 | 页面 | 数据库 | 异步 | 接口版本 | 断言的增长 | Phase（接口 / 页面） |
|---|---|---|---|---|---|---|---|
| I1 | 创建工作项 | 成员在项目的工作项页按 `c` 打开创建弹窗，填标题、描述（含一个 @提及）、状态、优先级、两位负责人、两个标签、起止日期，提交；新的工作项出现在列表里，提示里有编号 `PROJ-1`。标题为空、开始晚于截止：字段下方提示，焦点到第一个出错的字段（`useRefusedFocus`）；工作项的弹窗里打开"Save this draft?"再按 Escape，里层关上、外层留着（M3 交接第 7 节） | `issues` 新增一行：`sequence_id = 1`、`projects.last_issue_sequence = 1`，`state_id` 是选的状态，`sort_order` 按 3.7，`description_stripped` 是纯文本，`created_by_id`、`updated_by_id` 是当前账户；`issue_assignees` 两行、`issue_labels` 两行；`issue_activities` 一条 `field` 为空的创建行和两条负责人行（3.13），没有标签行；`issue_subscribers` 有创建者、两位负责人和被提及的成员；`issue_mentions` 一行；`issue_description_versions` 一行 | — | `POST /api/v0/projects/{project_id}/issues`，同一组断言；不带 `assignee_ids` 时项目的默认负责人被指派（他仍是项目中不是访客的有效成员时），他离开项目之后再建：没有负责人；负责人是项目访客 422（`assignee_ids[0]`，`not_allowed`）；访客 403；已归档的项目 409 `project.archived`（分诊状态 422 经接口造不出：M3 不许建分诊状态，这一格在 Go 的集成测试里，由仓储准备数据） | P1a：工作项的行、编号、默认状态和默认负责人、负责人、标签、关注（创建者、负责人）、创建和负责人的动态、接口的各种拒绝；P2：起止日期；P4：描述（清洗、`description_stripped`）、@提及的 `issue_mentions` 一行和被提及者的关注、描述版本一行；P14：页面 | P1a / P14 |
| I2 | 在详情页改每个字段 | 打开 `/…/issues/{id}` 改标题、描述、状态、优先级、负责人（加一个、去一个）、标签（加一个、去一个）、起止日期；时间线依次出现对应的动态，连续两次改描述只有一条描述动态。打开一条描述没有块 id 的工作项，不做改动：没有新的动态（M1-P2 交接） | `issues` 各列、`updated_at`、`updated_by_id`；改到"已完成"组的状态时 `completed_at` 有值，改回时为空；`issue_activities` 每个改动一行（标签、负责人的增减各一行），连续两次改描述只有一行（3.13 第 4 条）；只补块 id 的保存没有新的动态、版本，`updated_at` 不变（3.7） | — | `PATCH /api/v0/issues/{issue_id}`，同一组断言；开始晚于截止 422（只传一个日期时与存着的另一个比较）；已归档的工作项 409 `issue.archived`；`skip_activity` 带别的字段 422 | P1b：标题、状态、优先级、负责人、标签的修改和它们的动态，`completed_at`，`updated_at`；P2：起止日期的修改和动态、合并之后的日期 422、已归档 409；P4：描述的修改、连续两次的合并、只补块 id 的保存不写动态和版本、不改 `updated_at`，只改链接地址或勾选框的保存照常记下，`skip_activity` 带别的字段 422；P14：页面 | P1b / P14 |
| I3 | 列表与看板：分组、子分组 | 列表按状态分组，每组有计数，一组超过一页时"加载更多"；看板按优先级分组、按标签子分组（泳道），一个带两个标签的工作项出现在两条泳道里；切换布局后刷新仍是看板 | 数据库不变（读） | — | `GET /api/v0/projects/{project_id}/issues?group_by=state_id`、`group_by=priority&sub_group_by=label_id`：每组的 `total` 是不重复的工作项数，`label_ids` 是工作项完整的标签（不只是所在的组）；"加载更多"是同一个请求加 `group_key=<组的键>&cursor=<组的 next_cursor>`，翻完不重复、不遗漏，`total` 等于组的；带 `label_id=A&group_by=label_id` 时，另一个标签的组"加载更多"仍只有带 A 的；未知的 `order_by` 422 | P5：全部的接口断言；P12：页面 | P5 / P12 |
| I4 | 表格排序与日历 | 表格按标签名、负责人名、子工作项数、截止日期升降排序，翻到第二页仍是同一个顺序；日历的月视图按截止日期显示，把一个工作项拖到另一天 | 拖动之后 `target_date` 和一条 `target_date` 动态 | — | 每种 `order_by` 两个方向各一个翻页测试（3.16 的游标）；`target_date=2026-10-01..2026-10-31&group_by=target_date` | P5：`link_count` 之外的每个排序码、日期的范围和 `none`；P8：`link_count` 的两个方向；P12：页面（含日历的"添加已有"：只列出没有截止日期的） | P5 / P12 |
| I5 | 筛选与显示设置的保存 | 加筛选：状态组"进行中"、负责人"我"、标签"无"、截止日期两个范围；关显示属性"标签"；刷新之后筛选、显示方式、显示属性都还在；另一个成员打开同一个项目看到的是他自己的设置 | `project_user_properties` 中这位成员那一行的 `filters`、`display_filters`、`display_properties`（3.17）；另一位成员的行不变 | — | `PATCH /api/v0/me/projects/{project_id}/preferences` 带三个新字段；未知的分组键 422；`GET` 列表带同样的参数，结果相同 | P5：接口；P13：页面 | P5 / P13 |
| I6 | 在看板上拖动（总体设计 8.2 的示例） | 成员把一张卡片从"Todo"拖到"Done"、放在两张卡片之间：卡片出现在"Done"，两列的计数各变一 | `issues.state_id`、`completed_at` 有值、`sort_order` 在相邻两张之间、`updated_by_id` 是当前账户；`issue_activities` 一条 `field = state`，`old_identifier`、`new_identifier` 是两个状态的 id，`actor_id` 是当前账户 | — | `PATCH /api/v0/issues/{issue_id}` 带 `state_id`、`sort_order`，同一组断言 | P5：接口；P12：页面 | P5 / P12 |
| I7 | 快速添加 | 在列表某一组的底部输入标题回车，工作项出现在那一组的末尾；只有空白时字段下方提示 | `issues` 新增一行，状态是那一组的状态 | — | 与 I1 的接口版本共用 | P5：接口（同 I1）；P12：页面 | P5 / P12 |
| I8 | 子工作项 | 在父工作项的子工作项区"添加已有"选两个，"新建"一个；把其中一个移到别的父工作项；去掉一个的父工作项；选父工作项时列表里没有它自己的后代 | 子工作项的 `parent_id`；每个被改的子工作项一条 `parent` 动态（`old_value` 是原来的父工作项的编号）；父工作项的 `sub_issues_count` 由 `getIssue` 读出 | — | `POST /api/v0/issues/{issue_id}/sub-issues`、`PATCH` 子工作项的 `parent_id`；把祖先加为子工作项 422（`sub_issue_ids[0]`，`not_allowed`）；已归档的父工作项 409 `issue.archived` | P3：接口；P14：页面（两个选择器经 `issue-search` 的 `for` 排除，P8 已有） | P3 / P14 |
| I9 | 关联 | 在详情页加"阻塞"（同项目）和"关联"（另一个项目的工作项）；从对方的详情页看到反向的"被阻塞"；删除一条 | `issue_relations` 一对工作项一行，"阻塞"存为反向的 `blocked_by`；每次增减两条动态（正向、反向，3.13 第 20–23 行），反向一条的 `project_id` 是对方的项目 | — | `POST /api/v0/issues/{issue_id}/relations`、`DELETE /api/v0/issue-relations/{relation_id}`；已有关联的一对 422（`issue_ids[0]`，`duplicate`）；对方看不到的项目里的工作项 422（`not_allowed`）；关联列表不含看不到的工作项 | P3：接口；P14：页面 | P3 / P14 |
| I10 | 链接与元数据 | 加一个链接（只写了 `127.0.0.1:<端口>/page`，端口是本机的页面服务器的），列表里先显示地址，随后显示抓到的标题和图标；改标题；删除 | `issue_links` 一行，`url` 补上 `http://`；`metadata` 由空对象变为 `{title, favicon, url, favicon_url}`；改地址之后重新抓取；三条链接动态 | `river_job` 有一条 `issue.crawl_link`，完成；指向 `169.254.169.254` 的链接得到 `{error: "blocked", …}`（"没有发出连接"由 6.8 的 Go 表测试证明，e2e 只断言 `blocked`）。放行 `127.0.0.1/32` 写在 `config.test.yaml` 的 `outbound.allow_cidrs`，worker 自己的 nerve 和 `nerveWith` 起的 nerve 都带着它，谁取到任务结果都一样（3.11） | `POST /api/v0/issues/{issue_id}/links` 等，同一组断言；同一个地址两次 409 `issue.link_exists` | P7：接口；P14：页面 | P7 / P14 |
| I11 | 评论与表情回应 | 发一条带 @提及 的评论，编辑（显示"已编辑"），给工作项和评论各加一个表情再去掉；空评论的发送按钮不可用；成员在别人的评论上没有编辑、删除，项目管理员在别人的评论上只有删除（决策点 8，负责人裁定 B） | `issue_comments` 一行、`edited_at`；`issue_reactions`、`comment_reactions` 加了又软删除；评论者和被提及的人进入 `issue_subscribers`；工作项的 `updated_at` 随评论、工作项的表情而变，评论的表情不变它 | — | `POST /api/v0/issues/{issue_id}/comments` 等；空评论 422（`comment_html`，`required`）；改别人的评论 403 `issue.not_author`，成员删别人的评论 403 `issue.not_author`、项目管理员删别人的评论 204；同一个表情两次 409 `issue.reaction_exists` | P4：接口；P15：页面 | P4 / P15 |
| I12 | 操作动态的时间线 | 时间线按时间合并动态和评论，切换新旧顺序、只看评论或只看动态；超过一页的时间线打开时读完全部页，两页的行都显示（3.13）；另一个标签页加了评论之后，回到这一页的时间线在下一次修改之后读到它 | 数据库不变（读） | — | `GET /api/v0/issues/{issue_id}/activities`、`/comments`：按 `(created_at, id)` 升序翻页，不重复、不遗漏；从最后一页的游标再读得到新的行（3.13）；引用了看不到的工作项的动态不在列表里（3.13） | P4：接口；P15：页面 | P4 / P15 |
| I13 | 描述的历史与还原 | 成员甲建工作项（带描述），成员乙连着改两次描述，甲再改一次：历史里有三个版本，乙的两次合为一个；乙打开甲的第一版，还原（最新的一版是甲的，所以不合并）：描述变回去，历史多一个版本、时间线多一条描述动态 | `issue_description_versions`：同一个人 600 秒内的修改合并为一行（`last_saved_at` 更新），换了人新的一行（3.14）；还原经普通的 `PATCH`；600 秒的边界由后端的集成测试用时钟覆盖 | — | `GET /api/v0/issues/{issue_id}/description-versions`、`GET /api/v0/issue-description-versions/{version_id}`；项目访客 403 | P4：接口；P15：页面 | P4 / P15 |
| I14 | 草稿发布 | 在创建弹窗里写了一半，选"存为草稿"；在草稿页打开它，补上标题，"移到项目"：工作项出现在项目里，草稿消失 | `draft_issues` 一行（`payload` 是创建的请求体）；发布之后与 I1 相同的工作项各表，草稿软删除，同一个事务 | — | `POST /api/v0/workspaces/{slug}/drafts`、`POST /api/v0/drafts/{draft_id}/publish`；`payload` 缺标题时发布 400，与直接创建同一个请求体的 400 相同（M2 交接第 4 节）；草稿引用的状态已删除 422（`state_id`）；别人的草稿 404 `issue.draft_not_found`；项目访客发布 403 | P8：接口；P15：页面 | P8 / P15 |
| I15 | 归档、恢复与归档页 | "进行中"的工作项的菜单里没有"归档"；改到"已完成"后归档，它从列表消失，出现在项目的归档页；在详情页它是只读的，只能恢复和删除；恢复 | `archived_at` 有值、`updated_at`；动态 `archived_at` 的 `manual_archive`、`restore` 两条；恢复后为空 | — | `POST /api/v0/issues/{issue_id}/archive`、`/unarchive`；"进行中"的工作项 409 `issue.not_archivable`；已归档的工作项上评论、加关联、改字段都是 409 `issue.archived`；`GET …/issues?archived=true` 只有已归档的；项目访客 403 | P2：归档、恢复、两条动态、`issue.not_archivable`、改字段 409、访客 403；P3：加关联、加子工作项 409；P4：评论 409；P5：`?archived=true` 的列表；P13：页面 | P2 / P13 |
| I16 | 自动归档 | 项目设"1 个月"自动归档；一个"已完成"的工作项 31 天没有更新之后（时钟偏移）从列表消失，时间线显示"Nerve 归档了工作项" | `archived_at` 有值；动态 `archived_at` 的 `archive`，`actor_id` 为空（决策点 4）；没有更新够久的、状态不是完成或取消的不变（这两个对照经偏移之后的 nerve 写下，它们的 `updated_at` 是偏移之后的时刻，3.20） | 一条 `issue.auto_archive` 在偏移的 nerve 起来之后完成（`finalized_at` 晚于它的启动；准备数据的 nerve 每 2 秒也跑过它）；轮询 `issues.archived_at` | 只有任务一条路，接口版本只用 PAT 读：`GET /api/v0/issues/{issue_id}` 的 `archived_at`、`GET …/activities` 那一条的 `actor_id` 为 `null`。页面版本在偏移的 nerve 上重新登录（两个 nerve 的测试签名密钥不同，`session_ttl` 短于 31 天） | P6：接口；P15：页面 | P6 / P15 |
| I17 | 删除（连同子树） | 父工作项有两层子工作项、一条跨项目的关联、评论；删除确认框写明它的子工作项（含更下层的、已归档的）一并删除；删除之后它和子树都不在列表里，对方的关联列表里也没有它 | 父工作项、子树、它们的负责人、标签、关联（两个方向）、链接、评论、表情、关注、提及、动态、描述版本的 `deleted_at` 都是同一时刻、`updated_by_id` 是删除者（`expectIssueDeleted`，按目录找到每张指向 `issues` 的表，与 `expectProjectDeleted` 同一写法） | — | `DELETE /api/v0/issues/{issue_id}`；项目成员删别人建的 403；他自己建的 204；项目访客删自己建的 403（页面相同，3.4）；已离开项目的创建者 404 | P3：子树和它们的负责人、标签、关注、动态、两个方向的关联同一时刻删除，创建者的规则；P4：前置数据加评论、表情、提及、描述版本（`expectIssueDeleted` 按目录覆盖新表）；P7：前置数据加链接；P14：页面 | P3 / P14 |
| I18 | 搜索 | 命令面板输入 `proj-12` 得到那一个工作项，输入标题的一段（三个字以上）得到模糊匹配；选父工作项、加关联的搜索不列出自己、后代、已关联的；已归档的不出现 | 数据库不变（读） | — | `GET /api/v0/workspaces/{slug}/issue-search?q=…`；`GET /api/v0/workspaces/{slug}/issues/PROJ-12`（标识不分大小写）；看不到的项目里的编号 404 `issue.not_found` | P8：接口；P15：页面 | P8 / P15 |
| I19 | 个人主页的工作项 | 成员打开同事的主页，"已分配""已创建""已关注"三个标签页各列出他的工作项，只有调用者是有效成员的项目里的；按项目分组；访客打开别人的主页看到"没有权限"，页面不请求列表（M3 决策点 3） | 数据库不变（读） | — | `GET /api/v0/workspaces/{slug}/members/{user_id}/issues?relation=assigned`；访客 403；不是成员的项目里的工作项不出现 | P5：接口；P13：页面 | P5 / P13 |
| I20 | 访客的边界（权限矩阵的页面抽样，9.2） | 项目访客（"访客可见全部"关）只看到自己建的工作项，打开别人的工作项的地址"找不到"；不能建、不能改；能评论自己建的工作项。项目设置里打开"访客可见全部"之后他看到全部 | 数据库不变（除了评论） | — | 抽样四格：访客 `GET` 别人的工作项 404、`POST /issues` 403、列表只有自己的、他的工作项上评论 201 | P5：接口（评论的一格用 P4 的评论）；P14：页面（列表只有自己的、别人的工作项的地址"找不到"、不能建、不能改）；P15：页面上评论自己建的工作项 | P5 / P14 |
| I21 | 物理清理 | — | 软删除的工作项（带动态、评论）、PAT、标签、项目，61 天之后（时钟偏移）从表里消失；动态随工作项一起删除（外键 `CASCADE`，3.21）；60 天之内的仍在 | 定时任务 `cleanup.purge_deleted` 完成；轮询各表 | 只有任务一条路，接口版本没有 | P9：全部（这时全部的表都已在） | P9 / — |
| I22 | 工作项页上的会话切换 | 在工作项详情页改状态，请求经 `route.fetch()` 先到达 nerve 并成功；另一个标签页退出并以另一个账户登录之后，才经 `route.fulfill()` 把回答交给页面：原标签页不提示、不改页面（总体设计 7.7，M3 的 W3 同一写法）；后退到另一个项目的工作项页时，开着的浮层和弹窗不跟过去（M3 交接第 6 节） | 修改照常写入 | — | — | P14：页面 | — / P14 |

**冒烟故事的更新**：
- S1：`goose_db_version` 的最新版本等于最后一个迁移文件（照旧，M4 加 19 个，4.1）。
- S2：`REQUESTS` 的清单（`e2e/stories/smoke/s2-web-app.spec.ts:139`）随挂载路径上的取数改：P10b 删去项目页挂载时的 `user-properties` 请求（`workItemFilters`，M3 交接第 5 节）、加上工作项列表的请求；P14 起打开一个管理员建的工作项的详情，按角色有各自的预期：管理员、成员看到它；"只看自己"的访客得到"找不到"的页面，那一个 404 按请求放行；不是项目成员的工作区成员（S2 的项目是公开的，`s2-web-app.spec.ts:205`，他看得到项目而不能读工作项）得到"没有权限"的页面，那一个 403 `forbidden` 按请求放行（3.5 的预检正是这样回答他）；S2 不许有别的失败的请求。S2 在四个角色下仍断言没有旧接口的请求。

---

## 3. 设计裁定

以下裁定都在总体设计、M2 设计和 M3 设计定下的范围内；改写已批准规则的地方列在 3.23，按 Phase 同步到上级文档。与 Plane 不同的地方逐条登记到差异清单（4.17）。3.3、3.4、3.5、3.16、3.19 是后续各 M 照做的约定。Plane 的出处中，`API/` 指 `plane/apps/api/plane/`，`WEB/` 指 Plane 原来的 `apps/web/core/`（Nerve 的副本在 `web/apps/web/core/`，文件相同时直接写 Nerve 的路径）。

### 3.1 挂载时的规则延伸到 M4 能到达的页面（M2 设计 3.1；M3 设计 3.1）
**规则：M4 能到达的页面，挂载时不向 M5 及以后的旧接口发任何请求。** 这是 M2 设计 3.1、M3 设计 3.1 的同一条规则，范围扩大到 M4 的页面。

- **M4 能到达的页面**：M3 的页面，加上项目的工作项页（列表、看板、日历、表格四种布局，含浮层）、`/:workspaceSlug/projects/:projectId/issues/:issueId`（只做跳转，7.4）、`/:workspaceSlug/browse/:workItem`（详情）、项目的已归档工作项（列表和详情）、`/:workspaceSlug/drafts`、成员和管理员看到的个人主页的三个标签页（已分配、已创建、已关注）、命令面板的工作项页和搜索。
- **不在其内**（属于各自的 M）：迭代、模块、视图、收集箱的页面（M6、M7），工作区视图 `workspace-views`（M7），通知（M7），项目已归档的迭代、模块（M6）。M4 页面上指向它们的链接是已知的过渡（与 M3 设计 3.1 相同）。详情页"找不到"的空状态里有一个按钮跳到 `workspace-views/all-issues`（`browse/[workItem]/page.tsx:95`、`issues/(detail)/[issueId]/page.tsx:57`），改为跳到项目列表，M7 不必改回（工作区视图不是更合适的落点）。
- **删掉的挂载时请求和入口**（各自的 M 随新接口加回，写进 13.2）：
  - 详情和浮层的附件区（`components/issues/attachment/`、`issue-details/attachment.store.ts`，M5）：不挂载、不取数；附件区和"添加附件"的按钮在 M5 之前不显示。
  - 编辑器的图片上传（M5）：M4 的每个编辑器（描述、评论、创建弹窗、草稿）传 `disabledExtensions` 含 `image`（编辑器已有的参数，`web/packages/editor/src/types/extensions.ts:7`；`core/components/editor/rich-text/editor.tsx:50`），斜杠命令和粘贴、拖入都不再走 `FileService`。M5 去掉这一项。这不是开关：参数和它的两种取值都是编辑器原有的，M4 只是不给出上传的能力。
  - 详情的关注按钮和 `issue-details/subscription.store.ts`（M7）：不挂载、不取 `is_subscribed`。M4 的自动关注照样写 `issue_subscribers`（3.12），个人主页的"已关注"标签页读得到它。
  - 详情的"在收集箱中"跳转（`browse/[workItem]/page.tsx:79-85`，读 `is_intake`，M7）：删除。
  - 筛选和分组的迭代、模块选项（M6）：接口的枚举里没有它们（3.16），前端的选项由生成的类型约束，删除（7.3）；项目开着迭代、模块功能时也不出现。
- **M4 自己的挂载时取数一次改完**：工作项列表、显示设置、详情的六个区（工作项、动态、评论、子工作项、关联、链接）、表情、描述的历史都在前端的数据层（P10a、P10b、P11）改接新接口，页面的 Phase 在这三段之后。项目页挂载时取 `user-properties`（`workItemFilters`，`e2e/fixtures/mounts.ts:56`）在 P10b 改为读项目的显示设置（3.17），e2e 的放行一并删除。
- **理由与代价**：与 M2、M3 相同。代价：M5 之前描述和评论里不能放图片；M7 之前不能手动关注、取消关注。
- **守卫**：第 2 节每个有页面的故事都经 `watchPage` 断言；收尾按页面逐个列出守着它的故事。README"前端"一节"M3 中看到的页面"一条改写为 M4 能到达的页面（3.23）。

### 3.2 后续 M 的字段何时进入接口（照 M2 设计 3.2）
| 字段 | 规则 | 处理 |
|---|---|---|
| `Issue.attachment_count`（M5） | 3 | 不定义。删除前端的读取：表格的附件列、显示属性 `attachment_count`、按附件数排序的两个选项、卡片上的附件数；M5 随附件的接口加回（13.2）。`issues.attachment_count` 一类的计数在 M5 用查询算出，不加列 |
| `Issue.cycle_id`、`Issue.module_ids`（M6） | 3 | 不定义。表格的迭代、模块列，显示属性 `cycle`、`modules`，按迭代、模块分组、排序、筛选，卡片上的迭代、模块下拉框，在 M4 删除读取（P10a）；迭代、模块、项目视图、工作区视图的工作项列表（`store/issue/{cycle,module,project-views,workspace}/`、`roots/cycle-layout-root.tsx` 等四个布局根、页面上的列表部分）在 P10a 删除，M6、M7 照 Plane 的源码在新的列表引擎上加回（7.3，13.2） |
| `Issue.is_subscribed`（M7） | 3 | 不定义，关注按钮不挂载（3.1） |
| `Issue.is_intake`（M7） | 3 | 不定义，详情的收集箱跳转删除（3.1） |
| `Issue.estimate_point`、`type_id`、`is_draft`（Plane） | — | 不定义，前端的读取删除：估算、工作项类型 v0 不做（总体设计 1.2），草稿是另一张表（3.15） |
| `IssueComment.attachments`、`access`、`parent_id`、`comment_json` | — | 不定义（3.12、决策点 6）：`access` 随 M1 删掉的公开发布没有写入方；`comment_json` 没有写入方（M1 收尾交接）；附件 M5 经资产表关联，不在评论行上 |
| `IssueActivity.actor_detail`、`issue_detail`、`project_detail`、`workspace_detail` | — | 不定义（3.13）：渲染路径只读 `actor_detail.id`、`display_name` 和 `workspace_detail.slug`（R3 第 5.2 节），由工作区成员列表（含已结束的成员关系）和路由给出 |
| `Project` 的工作项计数（`total_issues` 一类） | 3 | 照 M3 设计 3.2，仍不定义 |

### 3.3 模块划分与跨模块的协作（架构问题 1，第 11 节）
- **新模块 `issue`**，拥有 13 张表：`issues`、`issue_assignees`、`issue_labels`、`issue_relations`、`issue_links`、`issue_subscribers`、`issue_mentions`、`issue_comments`、`issue_reactions`、`comment_reactions`、`issue_activities`、`issue_description_versions`、`draft_issues`。总体设计 6.2 的模块清单本来就有它；`server/sqlc.yaml` 加第四个条目（P1a）。
  - 理由：这 13 张表都以工作项为根，彼此有外键、同事务写入（动态、关注、提及随工作项的写入写在同一个事务，总体设计 6.6）。拆成"工作项""评论""草稿"几个模块，每个写都要跨模块端口，互相成环（评论要锁工作项，工作项的删除要连带评论）。
  - 不拆的代价：模块较大，用 `app` 层的文件按用例拆开（6.2），每个文件不超过约 400 行。
- **`issue` 经端口读写别的模块**，端口声明在 `issue/app`，由别的模块的 `Provide` 实现，`bootstrap` 接上（M3 设计 6.6 的两段组合）：
  - **向 `project` 要**：
    - `ProjectGate`：`Find`：不加锁地按项目 id 读出它的工作区（第一把锁之前，3.5）；`Lock`：以 `FOR NO KEY UPDATE` 锁住项目行（带 `deleted_at IS NULL`），回答 `archived`、`identifier`、`default_assignee_id`、`archive_in`；`Lock` 不回答 `guest_view_all_features`：锁下的判定经 `Authorize` 读它（P2 spec 第 3 节第 3 条）；`LockMany`：一条语句按 `id` 升序锁住几个项目（约定七，3.5）；两个不加锁的读：这个工作区未删除的项目的 id（草稿的读，3.15），`DeletedAt(projectIDs)`（P9 起）：其中已删除的项目和它们的 `deleted_at`（关联行属于先删除的那一端，3.5）。删除工作区锁它的全部项目不经这个端口，经 `project` 给 `workspace` 的 `ProjectCascade.LockWorkspaceProjects`（见下）。
    - `Sequences`：在项目行的锁之下 `UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 … RETURNING`（总体设计 5.3，M3 交接第 4 节）。
    - `StateCatalog`：项目的未删除状态（id、名称、组、`default`、是否分诊）；按 id 取一个；列表引擎按目录排序、分组用（3.16）。
    - `LabelCatalog`：项目的未删除标签（id、名称）；按 id 集合核对。
    - `ProjectMembers`：一组账户在这个项目里是否是有效成员、角色（负责人的检查、提及的过滤、默认负责人）。
    - `VisibleProjects`：调用者在一个工作区里是有效成员的项目、各自的角色和 `guest_view_all_features`（个人主页、搜索按它过滤，3.4；关联列表不经它：逐个项目照单个工作项的判定，3.4、3.10）；另一个方法给动态的过滤：调用者的成员关系（`is_active`）所在的全部项目，不看项目和成员关系的 `deleted_at`、不看归档（3.13）。
    - `ArchivePolicies`：设了 `archive_in`、未归档、未删除的项目（自动归档，3.8）。
  - **向 `workspace` 要**：`WorkspaceDirectory`：`workspace.Provide` 今天交给 `project` 的同一个目录（`workspace/module.go:33-49`），已有 `WorkspaceBySlug`（不加锁，个人主页、搜索、草稿、按编号取）和 `ShareWorkspaceByID`（`FOR SHARE`，草稿的写的第一把锁，3.5），`workspace` 不必加方法。
  - **向 `identity` 要**：`PublicProfiles`：按 id 批量读显示名。按负责人名排序的目录要它：目录是这个项目（个人主页：可读的项目）里**出现在负责人行上的每个账户**的显示名，含已离开、已降为访客的（他们仍是已有工作项的负责人，3.6），不只是有效成员（第二稿用 `ProjectMembers` 的有效成员，对他们排序键为空，与页面从含已结束的成员关系的成员列表取名字的本地重排对不上，对抗性评审 M22）。端口在 `bootstrap/ports.go` 由 `identity` 的读接成（P5）。
  - **`issue` 给 `project`**：两个端口，声明在 `project/app`，由 `issue.Provide` 交出（都不要 `issue.New` 的东西）：
    - `IssueCascade`，由 `issue/app` 的 `cascade.go` 实现，`project` 的状态、标签的写调用它：
      - `StateInUse(ctx, stateID) (bool, error)`：删除状态时，在项目 N 之下数用它的未删除工作项（含已归档的），有则 `project.state_in_use`（3.7）；
      - `DetachLabel(ctx, labelIDs, by, now)`：删除标签（连同子标签）时软删除它们的 `issue_labels` 行。
      - P1–P8 的过渡：另有 `DeleteProjectIssues(ctx, projectID, by, now)`、`DeleteWorkspaceIssues(ctx, workspaceID, projectIDs, by, now)`（这次锁住、删除的项目，加没有项目的草稿），在删除项目、工作区的事务里同步软删除工作项一侧的行，调的是任务用的同一份编排和语句（6.2）。P9 把它们换成事件和后台任务（3.6，决策点 7），这两个方法随之删除。过渡是因为后台任务要 P6 的事件和只投递的客户端，而 P1a 起删除项目就要带走工作项的表（M3 的删除故事，2 节）。
    - `IssueListSettings`：检查项目的显示设置的取值（`filters`、`display_filters`、`display_properties`），由 `issue/domain` 的列表词汇实现：纯函数，不要连接池（3.17）。
  - **`project`、`workspace` 发布事件**（P9，决策点 3、7）：`deleteProject` 在事务里发布 `project.deleted`，`deleteWorkspace` 发布 `workspace.deleted`，带这次锁住、删除的项目（3.19）；`issue` 的订阅者在同一个事务里投递删除工作项一侧的任务。两个模块的 `Deps` 加 `EventPublisher`，不知道谁订阅。
  - **写闸门**（P1b，架构问题 11）：`issue/app` 声明端口 `WriteGate`，由 `bootstrap` 用 `platform/gate` 接成（3.5）。
  - **`project` 给 `workspace`**：`ProjectCascade` 加一个方法 `LockWorkspaceProjects(ctx, workspaceID)`：删除工作区在工作区 N 之后、读时刻之前，按 `id` 升序以 `FOR NO KEY UPDATE` 锁住它的未删除项目（决策点 1，3.5）。
- **组合的顺序**（M3 设计 6.6 的延伸，6.6）：全部模块先 `Provide`（`identity`、`workspace`、`project`、`issue` 各自交出端口的实现），然后 `access.New` → `issue.New` → `project.New` → `workspace.New` → `identity.New`。`issue.New` 要 `project`、`workspace` 的 `Provide` 交出的端口、只投递的 River 客户端（在全部模块之前由连接池建出，3.20）和写闸门；`project.New` 要 `issue.Provide` 交出的 `IssueCascade`、`IssueListSettings`；`project.New`、`workspace.New` 要分发器（P9 起，它们发布删除的事件）。没有构造的环：`Provide` 只建适配器（查询对象）和纯函数，不要别的模块的东西；分发器由 `issue.Subscribers(Inserter)` 建出，在三个模块的 `New` 之前（6.6）。
- **跨模块只经端口**：`issue` 的 sqlc 查询只读写自己的 13 张表（`TestSQLCSchemaScope`）。列表引擎要的状态、标签、成员的名字由端口读出，作为参数传进查询（架构问题 5，3.16）；不在 sqlc 的守卫里开例外。
- **领域事件**（决策点 3，3.19）：`issue` 的用例在事务里经 `shared.EventPublisher` 发布事件；订阅者由各模块交出、`bootstrap` 组成分发器。要能拒绝、要在发起者的锁之下执行的连带仍是同步的端口（`StateInUse`、`DetachLabel`，M3 交接第 1 节"连带同步"）；删除项目、工作区之后工作项一侧的行不需要这两样，在 P9 改为事件的订阅者投递的后台任务（3.6，决策点 7）。

### 3.4 权限：创建者、访客只看自己的工作项（架构问题 3）
- **`shared` 的扩展**（`shared/authorize.go`）：
  - `Target` 加 `CreatorID uuid.UUID`：按工作项（或它下面的行）判定时是工作项的 `created_by_id`，别的判定为零值。
  - `Grant` 加 `OwnOnly bool`：调用者是项目访客（项目角色 5）、项目的 `guest_view_all_features` 为假时为真。用例拿它过滤（下面），不再读一次项目。
  - **按项目角色算，不看工作区角色**（对抗性评审 M4）：工作区管理员在一个项目里只是访客时，`access/domain/decide.go:76-87` 让他通过这个项目的每条规则（`ProjectAdmin`），而他的项目角色是 5，`OwnOnly` 仍为真。两者同时成立时 404 在先：别人建的工作项对他是"看不到"（404），他自己建的照管理员的规则可以写。Plane 的列表同样按项目角色过滤。9.2 的矩阵为他单列一列。
- **规则表的扩展**（`access/domain/rules.go`）：`Rule` 加 `AllowCreator bool`。项目级的规则带它时，角色不在集合里的调用者，若 `actor == Target.CreatorID`，并且他是项目的**有效成员、项目角色不低于成员**，也允许。
  - Plane 的 `creator=True` 只看工作区成员关系、不看项目成员关系（`API/app/permissions/base.py:24-38`），已离开项目的创建者仍能读、改、删"他的"工作项。Nerve 要求创建者仍是项目的有效成员（看不到项目的人 404，与别人相同）。
  - "不低于成员"：项目访客创建的工作项，他不能凭 `AllowCreator` 删（在项目里是访客的工作区管理员照管理员的规则删他自己建的；Plane 的页面只给项目管理员和成员显示删除，`WEB/components/issues/issue-layouts/quick-action-dropdowns/project-issue.tsx:60-69`，删除框再要求是创建者或项目管理员，`delete-issue-modal.tsx:54-82`）。登记差异。
  - 今天只有 `issue.delete` 带 `AllowCreator`（规则 `{LevelProject, [Admin], AllowCreator}`），效果是"项目管理员，或建了它的项目成员"。
- **`ProjectFacts` 多读一列**：`access` 读项目事实的端口（`project/adapter/postgres/queries/access.sql:1-8`）加读 `guest_view_all_features`，判定时算出 `OwnOnly`。
- **一条访客规则，用在每个地方**：访客（`OwnOnly`）只看到自己创建的工作项；看不到的工作项和挂在它下面的一切（评论、链接、关联、子工作项、动态、表情、描述的历史）都是 404。Plane 只在约 6 处执行这条规则，别处泄露（R2 第 0.8 节：动态和评论列表、子工作项、关联、链接、表情、全局搜索、个人主页都不检查），Nerve 处处执行。
  - 单个工作项：用例读工作项（得到项目和 `created_by_id`）→ 判定 → `grant.OwnOnly && issue.CreatedByID != actor` 时答 `issue.not_found`（与不存在相同，不透露存在；角色不允许时也先看访客规则，404 先于 403，6.7）。按资源 id 寻址的子资源（`/issue-comments/{id}`、`/issue-links/{id}`、`/issue-relations/{id}`、`/issue-description-versions/{id}`）先读子资源得到工作项，同样处理，答子资源自己的 404 码。
  - 列表和搜索：同一条规则，在 SQL 中（`OwnOnly` 的项目加 `created_by_id = $actor`：子工作项的列表、P5 的列表引擎），或者读出之后逐个（关联的列表：`seeEach` 按每个对方的项目的 `Grant`）。个人主页、搜索跨项目，每个项目按它自己的 `OwnOnly` 过滤（`VisibleProjects` 回答每个项目的角色和标志）；关联列表照单个工作项的判定（`see`）对每个对方的项目判定一次 `issue.read`、再照那个项目的访客规则（`seeEach`），别的已归档的项目经不加锁的 `ArchivedProjects` 读出（3.8、3.10）。
  - 关联：对方工作项必须对调用者可见（他在它的项目上有 `issue.read`，即那个项目的有效成员，公开项目也一样；并且过了那个项目的访客规则），不然加关联 422（`issue_ids[i]`，`not_allowed`），关联列表不含它（3.10）。
  - 计数：`sub_issues_count` 按调用者的规则数（`OwnOnly` 的项目里只数他自己建的子工作项），按它排序也用同一个数（3.9、3.16）。
  - 父工作项：回答里 `parent_id` 指名调用者看不到的工作项时为 `null`（每个回答经同一处，P3 的 `seen`；与动态的行不指名看不到的工作项同一个规则，3.13）。修改的 `parent_id` 同调用者得到的回答比，他看不到的父工作项算没有：送回这个 `null` 什么都不改，指名它像指名不存在的工作项一样 422（负责人裁定 A，2026-10-24）。
  - 动态：一行动态引用的工作项（父子的行、关联的行的 `old_identifier`、`new_identifier`）对调用者看不到时，这一行不在他的动态列表里（与关联列表不含看不到的对方同一个规则，3.13）。
  - 关注：自动关注只加能读这个工作项的人（3.12）：被提及的"只看自己"的访客不是创建者时不关注。关注之后读的权限还会变（降为访客、项目关掉"访客可见全部"、被移出），关注行不随之删除，所以"关注者都读得到"只在加入的那一刻成立；M7 发通知时按那一刻的 `issue.read` 加访客规则再判定一次（写进 M7 的交接，13.2；对抗性评审 M5）。
  - 一个集成测试对 9.2 的每种身份核对：列表、个人主页、搜索的结果等于"对每个工作项逐个判定 `issue.read` 并应用访客规则"的结果（M3 设计 3.4 的同一种测试）；计数和动态的过滤各有一组用例。
- **操作名和规则**（`issue/domain/actions.go`，每个操作名在规则表一行；9.2 的矩阵逐格）：

  | 操作名 | 级别 | 允许 | 说明 |
  |---|---|---|---|
  | `issue.create` | 项目 | 管理员、成员 | 创建、快速添加、发布草稿（另要工作区的 `issue_draft.write`） |
  | `issue.read` | 项目 | 管理员、成员、访客（访客受 `OwnOnly`） | 取单个、列表、子工作项、关联、链接、评论、表情、动态 |
  | `issue.update` | 项目 | 管理员、成员 | 改字段、父工作项、负责人、标签、加子工作项、关联和链接的写（R2：链接任何成员都可改别人的，照搬） |
  | `issue.delete` | 项目 | 管理员；`AllowCreator` | 见上 |
  | `issue.archive` | 项目 | 管理员、成员 | 归档、恢复、已归档的列表（Plane 的归档列表只许管理员、成员，`API/app/views/issue/archive.py:106`） |
  | `issue_comment.create` | 项目 | 管理员、成员、访客（访客只能评论看得到的工作项） | Plane 相同（`API/app/views/issue/comment.py:67-81`） |
  | `issue_comment.write` | 项目 | 管理员、成员、访客 | 改、删评论；用例另有相对的规则：改只有作者（403 `issue.not_author`）；删是作者，或项目管理员（`Grant.ProjectAdmin`）删别人的（决策点 8，负责人裁定 B；页面今天只给作者显示，`comments/quick-actions.tsx:41-43`，P15 给项目管理员加"删除"；Plane 服务端允许管理员改、删，登记差异） |
  | `issue_reaction.write` | 项目 | 管理员、成员、访客 | 工作项和评论的表情回应；只能去掉自己的 |
  | `issue_description_version.read` | 项目 | 管理员、成员 | 页面只给管理员、成员显示历史（`main-content.tsx:141-159`），Plane 服务端允许访客，登记差异 |
  | `issue_draft.write` | 工作区 | 管理员、成员 | 草稿的全部操作，只操作自己的草稿（Plane 的页面只给管理员、成员显示草稿入口，`workspace/sidebar/user-menu.tsx:48-54`） |
  | `member_issue.list` | 工作区 | 管理员、成员 | 个人主页的工作项（M3 决策点 3：访客看别人的主页没有工作项列表），行再按调用者可见的项目过滤 |
  | `issue.search` | 工作区 | 管理员、成员、访客 | 结果按可见性和访客规则过滤；按编号取（`getIssueByIdentifier`）按目标项目判定 `issue.read` |
  | `project_preferences.read`、`.update` | 项目 | 任何有效成员（M3 已有） | 显示设置的三列随 M3 的操作（3.17） |

- **相对的规则在用例里**：评论的作者（改）、作者或项目管理员（删，决策点 8）、去掉自己的表情、草稿的拥有者（别人的草稿 404 `issue.draft_not_found`）、负责人和默认负责人的资格（3.7）。
- **判定的时机**：写在项目行的锁之后判定（3.5，M3 约定二），锁之前另有不加锁的预检，拒绝的在那里就答（3.5）；单个的读不开事务，直接判定；列表的读（目录和查询）在一个只读的 `REPEATABLE READ` 事务里，用同一个快照（3.16）。
- **Plane 里服务端比页面宽松的地方**，按页面收紧，逐条登记差异清单第四节：访客创建者改、删工作项（服务端允许，页面不显示）；管理员改别人的评论（删除按决策点 8 放开给项目管理员，页面随之加入口）；访客读描述历史；访客用草稿；访客读个人主页的工作项；访客关注、取消关注（M7 的接口照此）；已归档的工作项上评论、表情、链接、关联（3.8）；评论为空（3.12）；只传一个日期时不与存着的另一个比较（3.7）。

### 3.5 加锁顺序与并发（决策点 1 负责人裁定 B；架构问题 2）
M2 设计 3.5 的全局顺序和 M3 设计 3.6 的六条约定照旧。M4 的表接在后面；工作项的写取哪一把锁是决策点 1，负责人裁定 B，正文按它写定。

- **全局顺序**：`users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels` → `draft_issues` → `issues` → `issue_assignees` → `issue_labels` → `issue_relations` → `issue_links` → `issue_subscribers` → `issue_mentions` → `issue_comments` → `issue_reactions` → `comment_reactions` → `issue_activities` → `issue_description_versions`。同一张表的多行按 `id` 升序取锁。
- **约定二的延伸：工作项的写最先锁项目行，不锁工作区行**（决策点 1，负责人裁定 B）。
  - 工作项和它下面的每个写（创建、修改、删除、归档、恢复，父子、关联、链接、评论、表情，发布草稿）：不加锁地读出资源（按项目 id 的写读项目，按资源 id 的写读资源、得到工作项和项目；在事务之外，用完就还回连接）→ 不加锁的预检（下面：会被拒绝的在这里就答）→ 进入项目的写闸门（下面：不持连接地等）→ 开始事务 → 以 `FOR NO KEY UPDATE` 锁住项目行（带 `deleted_at IS NULL`，回答 3.3 列的几列）→ 重读资源，确认仍存在、仍属于这个项目 → 判定 → 依赖行的检查 → 读时刻 → 写 → 提交 → 离开闸门。
  - 为什么是项目的 N 而不是 S：一个项目里的工作项的写都要串行。取号要改项目行；父工作项不成环要在没有别的父子改动时沿祖先走（3.9）；`sort_order` 取同一状态的最大值；描述的合并、版本的合并读"最近的一行"再改它（3.13、3.14）；每个工作项的动态按提交的顺序得到不减的时刻（3.13 的增量读取靠它）。S 与 S 不互斥，这些都要另外加锁；N 一把锁全部做到。代价：同一个项目里的写一个接一个，每个是一个短事务。
  - 为什么不锁工作区行：M3 设计 17.4 的探针 B：新的 `FOR SHARE` 越过等待中的 `FOR NO KEY UPDATE`。工作项的写是最频繁的写，若照方案 E 先取工作区 S，连续重叠的工作项写可以让工作区一级的写（改工作区、删工作区、改角色、接受邀请、移出、停用）一直等到请求期限（M3 设计 17.4 的代价，"M4 的工作项写入……不默认沿用"）。N 与 N 互斥、排队，不会越过等待者：删除工作区在数据库里最多等每个项目上正在进行和排在它前面的写。S5 的 E1、E2、E3 在 PG 18.6 上证实了这两点（0.3）。这一句只在连接池不被排队的写占满时成立：没有写闸门时，200 个在同一个项目上排队的写占住全部 10 个连接，删除工作区先在池的队列里等（SP1，对抗性评审 I3），所以要下面的写闸门。
  - **删除工作区仍先锁它的项目**：工作区 N → 判定 → 邀请（N，`id` 升序，约定五的例外，照旧）→ 这个工作区的全部未删除项目（N，`id` 升序，`ProjectCascade.LockWorkspaceProjects`）→ 读时刻 → 软删除工作区 → 删除锁住的邀请 → 成员 → 显示设置 → `DeleteWorkspaceProjects`（项目 → 项目成员 → 项目显示设置 → 状态 → 标签）→ 发布 `workspace.deleted`（带锁住的项目），`issue` 的订阅者为每个项目投递一个删除工作项一侧的任务、为没有项目的草稿投递一个（P9 起；P1–P8 的过渡是在这里同步调 `IssueCascade.DeleteWorkspaceIssues`，3.3）。P1a 起时刻在两步锁之后读（`workspace/app/delete_workspace.go:68-76`）。
    - **在决策点 7 的 A 之下重新推一遍，这一步仍要**：工作项一侧的行不再在这个事务里写，但任务随后写下的是这个事务读的时刻 `t1`（一个改变一个时刻）。一个已持项目 N 的工作项的写若在 `t1` 之后读自己的时刻 `t2`、提交，任务随后把那个工作项写成 `updated_at = deleted_at = t1 < t2`：时刻倒退，与同步的连带一样。先锁全部项目再读时刻，`t1` 就晚于每个已提交的工作项的写；删除提交之后，工作项一侧的每个写在锁下都看到项目已删除（404），任务写的行没有更晚的时刻。代价比同步的时候小得多：事务只做 M3 的几步，项目的锁持几毫秒，不再持到全部工作项一侧的行写完（SP2：五个项目的工作区 36.5 秒）。
    - 删除项目同理：M3 的删除项目本来就是工作区 S → 项目 N → 读时刻，`t1` 晚于这个项目上每个已提交的写，不必改。
    - 邀请先于项目：全局顺序里 `workspace_member_invites` 在 `projects` 之前。锁项目之后的成员、显示设置是按谓词的批量，只有持这个工作区锁的事务能写到它们（M3 约定五的论证不变）。
  - **M3 的其他连带与工作项的写**：`DemoteToGuest`、`EndMemberships` 按 `id` 升序锁住他的项目（N），与工作项的写在项目行上串行：先提交的写按它判定时的角色生效，后到的写在锁下读到新的角色。只改工作区角色（不降为访客）的写不锁项目，与工作项的写不串行：判定读到的是判定那一刻已提交的角色。授权的结果等于两者按某个顺序先后发生；**时刻则不一定**（下一条的代价一）。M3 的项目级的写都先取工作区 S，与取工作区 N 的角色变更串行，所以在 M3 里两者的时刻也按生效的顺序；B 之下工作项的写不取工作区的锁，这一点不再成立。
  - **B 的两个代价**（独立评审 I1，决策点 1 写进选项）：
    - **代价一：授权与时刻的先后可以不一致。** `DemoteToGuest`、`EndMemberships` 的时刻由调用方传入，调用方在工作区 N 之后、锁项目之前读它（`server/internal/modules/project/app/cascade.go:58-94`）。一个已持项目 N 的工作项的写在这之后读到自己的时刻 `t2 > t1`、按旧的角色判定、提交；连带随后以 `t1` 记下移出或降级。结果：X 在 `t2` 写的动态，挨着他在 `t1 < t2` 结束的成员关系，而生效的顺序是"先写、后移出"。每一行自己的列不倒退（M3 的保证仍成立），丢掉的是"工作项的写与成员关系的变更之间，时刻与生效的顺序一致"；只改角色的写不锁项目，同样如此。方案 A 保住这一点。要在 B 之下也保住，就是决策点 1 的 B′（连带先锁项目、列举、再读时刻），它改写 M3 已批准的约定六的收缩路径和删除、离开、停用的用例，不建议。
    - **代价二：不是每条工作项一侧的路径都离开了工作区 S。** 草稿的写和发布草稿（下面）、M3 的全部项目管理类的写仍先取工作区 S，M3 17.4 的探针 B 的饥饿在这些较少的路径上仍在；B 只让最频繁的写（工作项和它下面的行）不再挡住工作区一级的写。
  - **只读的端口不加锁**：状态、标签的目录，成员的角色，在项目 N 之下读出就是稳定的：状态、标签的写和项目成员的写都持这个项目的 N（M3 设计 3.6 的取锁表）。
  - **不加锁的可见性预检**（M3 交接第 2 节的"先锁后判定的副作用"；独立评审 M8）：先锁后判定让任何已认证的账户在得到 404 之前取得一个锁。M3 取的是工作区 S（共享），B 之下是项目 N（排他）：一个知道某个看不到的项目的 id 的工作区成员，可以让那个项目的全部写排队，限速按凭证计数（每个 PAT 一个桶）挡不住他。所以每个工作项一侧的写在第一把锁之前做一次不加锁的预检：读出资源和项目之后调 `Authorize`，**凡是判定拒绝的都在这里答**（对抗性评审 M1）：看不到（不是项目的有效成员，或访客规则挡住）答资源的 404，看得到而不能做答 403（例如公开项目里不是成员的工作区成员、不是项目成员的工作区管理员、做不了这个写的访客，`access/domain/decide.go:86-88`），都不取任何锁、不进写闸门。
    - 为什么在锁之前拒绝是对的：拒绝的请求什么都没写，它在预检的那一刻排定先后（可线性化），等价于在那一刻到达、被拒绝。只有放行要在锁下重新判定：等锁期间被移出、降级的人照样在锁下得到 404 或 403。预检不放行任何锁下会拒绝的写。
    - **加关联的对方**：预检同样不加锁地读出每个对方工作项、判定它的可见性（3.10 的条件）；看不到的对方不进 `LockMany`，它的错误（`issue_ids[i]`，`not_allowed`）在预检时定下，与锁下发现的错误一起在 422 里返回。所以知道别的项目里的工作项 id 的人，不能经加关联去锁一个他看不到的项目（一次最多 20 个 id、可以分在 20 个项目里）。
    - 测试：看不到项目的账户、看得到而不能写的账户发出的写不取行锁、不写；加关联指向一个看不到的项目时不锁它（交错 21 的关联的一例：自己的测试 `TestARelationLocksNoProjectItDoesNotReach`，与取锁形状的测试的 `createIssueRelations`、`deleteIssueRelation` 两行并列）。
  - **写闸门：一个热的项目不能耗尽连接池**（对抗性评审 I3，架构问题 11；A、B 两个方案都要）：每个工作项的写在等项目 N 时持着一个池里的连接。连接池是 10 个（`database.max_conns`），`authenticated` 的额度让一个凭证一次突发 200 个请求；SP1 量到 200 个写在同一个项目上排队时，别的项目上无关的读也要等约 875 ms（池的等待者先来先得，读排在被锁挡住的写后面）。
    - 做法：进程内按项目的信号量（`platform/gate`，带 `context` 的按键的信号量，约 100 行），`issue` 经它声明的端口 `WriteGate` 使用（`Enter(ctx, projectIDs) (release, error)`）；P1b 一次一个项目（`Enter(ctx, projectID) (leave, error)`），P3 加多个（P1b spec 第 3 节第 10 条）。每个工作项的写在**取连接之前**进入它的项目的闸门，每个项目在一个进程里至多 K 个写同时在数据库里（`issue.write_gate.per_project`，默认 2：一个持 N 在写，一个已在等 N，提交之后不必再等调度）；其余的在进程里等，不持连接。关联的写按项目 `id` 升序进入它到达的每个项目的闸门（加关联：工作项自己的和到达的对方的，至多 21 个；删除关联：两端的）。
    - 等的上限：至多等 `issue.write_gate.max_wait`（默认 2 秒，与 `auth.password.max_wait` 同一种写法），一个项目的等待者至多 `issue.write_gate.max_queue` 个（默认 64），超过或请求的期限先到，就答 503 `server_busy`（`Retry-After: 1`，`shared.ServerBusy`，M2 已有）。另有一道兜底：工作项一侧的写的事务开始时 `SET LOCAL lock_timeout`（`issue.write_lock_timeout`，默认 5 秒），等锁超过它（SQLSTATE `55P03`）同样答 503 `server_busy`。这一句由平台发：`platform/postgres` 的 `TxManager` 加一个选项 `LockTimeout(d)`，`BEGIN` 之后发 `SET LOCAL`，模块的 SQL 里没有 `SET`（`TestModulesRunSQLOnlyThroughSQLC` 不许模块在 `gen` 之外发语句，聚焦复核 M6）；`55P03` 换成 503 只在 `bootstrap/gate.go` 一处，挨着闸门的 503（6.1）。没有选 sqlc 的 `SELECT set_config('lock_timeout', $1, true)`：那要每个写用例自己记得先调它，选项让给 `issue` 的写用例的 `TxManager` 一律带上。
    - 闸门不改变加锁的顺序：它不是数据库的锁，持着它的请求从不在持数据库的锁或连接时再等别的闸门（多个闸门在开始事务之前按 `id` 升序一次取齐），不走闸门的写（M3 的项目管理类的写、删除项目和工作区、后台的任务）不等闸门，所以闸门与数据库的锁之间不会成环。
    - 范围：v0 是单个进程（M2 设计 3.15），K 是每个进程的；多个进程时每个进程各 K 个；闸门限的是每个项目的份额（11.11），连接池的大小与 K、同时热的项目数一起由 M8 的部署交接写明（13.2）。闸门只管工作项一侧的写（含发布草稿）；草稿的保存持工作区 S、项目 S，不等 N，不进闸门。
    - 测试：交错 20 经真实的连接池跑：200 个写在同一个项目上，另一个项目上的读的耗时不超过一个上限（SP1 的回归）；闸门满时 503 带 `Retry-After`；`lock_timeout` 的兜底。
  - **外键的 `FOR KEY SHARE` 挡住改账户的邮箱**（对抗性评审 M2，不改代码，写明）：M4 的每个插入经指向 `users` 的外键（`created_by_id`、`actor_id`、`assignee_id`……）取账户行的 `FOR KEY SHARE`。`users.email` 是全表唯一的列（`00001_identity_users.sql:8`），`nerve users set-email`（`ChangeEmail`）改它要取 `FOR UPDATE`，与 `FOR KEY SHARE` 冲突。SP1 量到：首尾相连 3 秒的插入，改显示名等 1 ms，改邮箱等 2.913 秒。所以一个持续写入的代理，要等它停一下才能改它的邮箱；没有死锁。S5 的 E3"无害"的结论只对唯一键是部分唯一的表（`workspaces`、`projects`）成立。M4 不改；给 `set-email` 加 `lock_timeout` 和重试的提示交给 M8（13.2）。
  - **没有新的成员关系增长**（M3 交接第 2 节）：指派、提及、关注都不让谁成为项目成员（负责人必须已是成员，3.7），所以约定六不加新的路径，也不加与停用的交错测试。
  - **不物理删除账户**（M3 交接第 2 节）：60 天清理不删除账户（3.21），`member_id`、`user_id` 上的 `ON DELETE CASCADE` 不会不经工作区的锁删除成员关系。
- **约定七（新）：跨项目的写按 `id` 升序同时锁住涉及的每个项目。** 今天只有关联：加关联、删关联涉及两个工作项，它们可以在两个项目里（3.10）。
  - 做法：不加锁地读出两边的工作项、得到它们的项目 → 这些项目（N，`id` 升序，一条语句）→ 重读 → 判定 → 写。
  - 为什么两边都要 N（设计简报原写"源一侧 N、别的工作项行 S"，这里改了，第 11.2 节）：关联的写会写到对方一侧的行：对方工作项的 `updated_at`（3.7）和对方工作项上的反向动态（它的 `project_id` 是对方的项目，3.13）。只锁源一侧，删除对方项目的连带（持对方项目 N、已读了时刻）可以与它交错，独立评审的 spike E5 在 PG 18.6 上复现了三种后果：
    - 对方工作项的 `updated_at` 倒退：连带先读时刻（10:00），关联的写后写对方的 `updated_at = 10:01`，连带随后把它写成 10:00；
    - 反向动态"先删后建"：10:01 建的反向动态被连带以 10:00 软删除；
    - **最坏的一种：留下一条活的关联行，指向已删除的工作项。** 删除项目的连带先锁关联行、再处理工作项（下面的表），连带等在对方工作项的行上时插入的关联行不在它锁住的集合里，连带结束之后它仍是活的。
    - 锁住对方的项目（约定七，E5b），关联的写等在对方项目的 N 上；删除提交之后它看到对方的项目已删除，什么都不写。对方工作项的行在它的项目 N 之下是稳定的，不必另锁行。交错 23 是这一条的回归测试。
  - 不会成环：多个项目一次按 `id` 升序取，与删除工作区（全部项目，`id` 升序）同一个顺序；M3 的项目级的写持工作区 S 和一个项目的 N，不等别的项目。
- **约定五的例外延伸到关联行**：删除工作项和删除项目的任务（P1–P8 是同步的连带）要软删除跨项目的关联行，而另一个项目上的写（删除另一端的工作项、删除另一端的项目）不持本项目的锁也能写到同一行。两个批量各按扫描顺序取锁会互相等成环（A 删除时锁 r1 等 r2，对方项目的删除锁 r2 等 r1）。所以这两处由一条语句按 `id` 升序锁住要删的关联行（`SELECT id … ORDER BY id FOR NO KEY UPDATE`），再只删锁住的那些（M3 约定五的例外的同一写法）。
  - 锁关联行的这一步在全局顺序里关联行的位置（`issues`、`issue_assignees`、`issue_labels` 之后，`issue_links` 之前），不提前到工作项之前（第二稿的删除项目先锁关联行、再删草稿和工作项，与全局顺序不合，对抗性评审 M13）。
  - 等锁之后 Postgres 在最新的版本上重新判断 `deleted_at IS NULL`：已被对方删除的行跳过，留着对方的时刻和删除者。关联行只被创建和软删除，不被修改，跳过它不会让任何一列的时刻倒退。
  - 删除工作区不另有特例（第三稿说"删除工作区的任务不需要它"，不对，聚焦复核 I1）：删除工作区是每个项目一个任务（3.6），同一工作区的两个项目的任务、更早删除的一个项目的任务、另一个活的项目上的删除工作项都可能写同一条跨项目的关联行，各自按 `id` 升序先锁，不会成环。
- **一条关联行属于先删除的那一端**（聚焦复核 I1）：另一端的项目已删除、它的任务还没写到这一行时，这一行属于那次删除，要写那次删除的时刻和删除者（同步的写法里它在那次删除的事务里就写了）。
  - 活的写：只有删除工作项会软删除别人那一端的关联行（删除关联对另一端已删除的关联答 404，3.10）。`deleteIssue` 锁住关联行之后，经 `ProjectGate.DeletedAt`（P9 起）不加锁地读出另一端的项目里已删除的那些，跳过它们的关联行（不写，锁随提交释放）。读到另一端的项目未删除、而它的删除在这之后才提交的，这一行按这次删除工作项写下，那个项目的任务随后看到它已删除、跳过：删除工作项在那次读的那一刻排在项目的删除之前，与交错 11 的同步写法同一个结果。不用约定七去锁另一端的项目：那要在锁本项目之前知道全部另一端的项目、一次取齐，给每个删除工作项加锁，而这里只需要"那一刻另一端删了没有"。
  - 任务：项目 P 的任务（时刻 `at`）对另一端的项目在 `at` 之前已删除的关联行跳过，留给那次更早的删除的任务；另一端在同一时刻随同一个工作区删除的，两个任务谁先锁到谁写，写下的时刻和删除者相同。另一端的项目的 `deleted_at` 同样经 `ProjectGate.DeletedAt` 读出。
  - 交错 24、25 断言写下的是谁的时刻和删除者，不只是"只删了一次"。
- **删除标签摘下 `issue_labels` 不需要按 `id` 先锁**（M3 交接第 4 节）：那里说 `issue_labels` 的写若不持项目或工作区的 `FOR NO KEY UPDATE`，就落在约定五的例外里。B 之下每个写活的项目的 `issue_labels` 的事务都持这个项目的 N（工作项的写、删除标签），删除项目、工作区之后的任务只写已删除的项目的行（3.6），`DetachLabel` 在项目 N 之下按谓词批量写到的行没有别的写能同时写到，所以不加按 `id` 的预锁，约定五的论证照旧。
- **草稿的写持工作区 S，有项目时再持项目 S**：草稿属于工作区（`issue_draft.write` 是工作区级的操作），先锁父行（工作区 S，M3 约定二"往父行下面加行、改下面的行"）、判定；草稿带项目时再以 `FOR SHARE` 锁住它的项目（改项目时新旧两个，`id` 升序），确认项目未删除、调用者看得到（3.15），然后锁草稿行（`FOR UPDATE`）、重读。
  - 为什么要项目 S：删除项目软删除这个项目的草稿（3.6，P9 起由任务写下删除请求的时刻）。草稿的写若不持项目的锁，删除读时刻之后草稿的写可以写下较晚的 `updated_at`，随后被写下较早的 `deleted_at`。持项目 S 时，删除项目（N）等它提交之后才读时刻；删除提交之后，草稿的写在锁下看到项目已删除（422 `project_id`）。
  - 重读时草稿的项目已不是读时的那个（另一个标签页改了它）：409 `issue.draft_changed`，调用者重试。
- **发布草稿**：读草稿（不加锁，得到项目）→ 预检 → 项目的写闸门 → 工作区 S → 判定 `issue_draft.write` → 项目 N → 判定 `issue.create` → 草稿行 `FOR UPDATE` → 重读（已删除 404 `issue.draft_not_found`；项目变了 409 `issue.draft_changed`）→ 与创建相同的检查和写入 → 软删除草稿。与 M3 的项目级的写同一个形状（工作区 S → 项目 N）。
- **不取项目锁的后台写**：
  - 链接的抓取（3.11）：一条带守卫的 `UPDATE issue_links SET metadata = $m WHERE id = $id AND url = $url AND deleted_at IS NULL`，只写 `metadata`，不写 `updated_at`、`updated_by_id`。不持别的锁，与任何事务都不会成环；链接已删除或地址已改时写 0 行。项目已删除、任务还没软删除这个链接时，写回照样写下 `metadata`：这一列没有时刻，不破坏一个改变一个时刻，链接随后被任务删除。
  - 60 天清理（3.21）：只删除软删除超过 60 天的行；没有任何写会再改一行已软删除的行（每个写都带 `deleted_at IS NULL`），清理与写不相遇。
  - **删除项目、工作区之后的任务**（3.6，决策点 7）：它只写一个已删除的项目的行（或一个已删除的工作区里没有项目的草稿）。项目已删除，工作项一侧的每个写在项目锁下都看到它已删除（404），所以任务不取项目的锁；还能写到这些行的只有别的任务和另一个活的项目上删除工作项的写，都只在跨项目的关联行上相遇，按 `id` 先锁（上面的约定五的例外），谁写由"一条关联行属于先删除的那一端"定。每批一个短事务，在全局顺序上逐张表、按 `id` 的键集进行。
  - 版本的保留不在其列：它按项目取 N（3.14），与自动归档相同，只看未删除的项目。它只删未删除的工作项的未删除的版本，清理只删软删除超过 60 天的行，两者碰不到同一行：两个任务都在 `periodic` 队列的两个 worker 上，若碰到同一批行，按不同的扫描顺序的物理删除会互相等成死锁（40P01）。
- **时刻的单调**：同一个项目里的写都在项目 N 之下、取得全部锁之后读时刻，后一个写在前一个提交之后才拿到锁，所以同一个工作项的动态、评论、版本行按提交的顺序得到不减的 `created_at`（墙钟不回拨时）。跨项目的关联的反向动态也在对方项目的 N 之下写。3.13 的增量读取（从最后一页的游标再读）靠这一条不漏行。
- **各操作的取锁**（S 是 `FOR SHARE`，N 是 `FOR NO KEY UPDATE`，"判定"是调用 `Authorize`；"读"是不加锁的读，没有时答资源的 404，工作项一侧的写在读之后另做不加锁的预检（判定拒绝的答 404 或 403、不取锁）、再进入项目的写闸门（上面），这两步不再写出；每个按资源寻址的写在锁之后重读资源，确认仍存在、仍属于这个项目，不再写出）：

  | 操作 | 依次 |
  |---|---|
  | 创建工作项、快速添加 | 读项目 → 项目 N → 判定 `issue.create` → 项目已归档 409 → 检查（状态、标签、负责人、父工作项，3.7、3.9）→ 取号（同一项目行的 `UPDATE … RETURNING`）→ 时刻 → `sort_order`（同一状态的最大值，3.7）→ 插入工作项 → 负责人（含默认负责人）→ 标签 → 关注 → 提及 → 动态 → 描述版本 → 事件 |
  | 修改工作项 | 读工作项 → 项目 N → 判定 `issue.update`（访客规则）→ 项目已归档 409 → 工作项已归档 409 → 检查（合并后的日期，状态、标签、负责人，父工作项不成环）→ 时刻 → 改工作项 → 负责人、标签的增删（差集）→ 关注 → 提及 → 动态（描述的合并，3.13）→ 描述版本（合并，3.14）→ 事件 |
  | 删除工作项 | 读工作项 → 项目 N → 判定 `issue.delete`（`AllowCreator`，访客规则）→ 项目已归档 409 → 子树（`WITH RECURSIVE`，在锁下）→ 时刻 → 软删除子树的工作项 → 负责人 → 标签 → 关联行（N，`id` 升序，两个方向，约定五的例外）→ 另一端的项目已删除的跳过（`ProjectGate.DeletedAt`，上面，P9 起）→ 软删除其余锁住的关联行 → 链接 → 关注 → 提及 → 评论 → 表情 → 评论的表情 → 动态 → 描述版本（批量，同一个时刻、同一个删除者）→ 事件 |
  | 归档、恢复 | 读工作项 → 项目 N → 判定 `issue.archive` → 项目已归档 409 →（归档）已是归档：不写，200；状态组不是完成、取消 409 `issue.not_archivable`；（恢复）未归档：不写，200 → 时刻 → 改 → 动态 → 事件 |
  | 加子工作项 | 读父工作项 → 项目 N → 判定 `issue.update` → 项目已归档 409 → 父已归档 409 → 子工作项的检查（同项目、未删除、未归档、不是父本身或它的祖先，422 `sub_issue_ids[i]`）→ 时刻 → 子工作项（`id` 升序）改 `parent_id` → 每个一条 `parent` 动态 → 事件 |
  | 加关联 | 读工作项和对方（得到项目）→ 预检含每个对方的可见性，看不到的对方不进下一步 → 其余的项目 N（`id` 升序，约定七）→ 判定 `issue.update` → 项目已归档 409 → 工作项已归档 409 → 对方的检查（可见、未删除、未归档、不是自己、这一对还没有关联，422 `issue_ids[i]`）→ 时刻 → 插入关联行（按对方的 `id` 排序，约定五的后一半）→ 两边的 `updated_at` → 正反两条动态 → 事件 |
  | 删除关联 | 读关联（得到两个工作项、两个项目）→ 两个项目 N（`id` 升序）→ 判定（两边任一个的 `issue.update`，3.10）→ 项目已归档 409 → 两边任一个已归档 409 → 时刻 → 软删除 → 两边的 `updated_at` → 正反两条动态 → 事件 |
  | 加、改、删链接 | 读工作项（加）或链接（改、删）→ 项目 N → 判定 `issue.update` → 项目已归档 409 → 工作项已归档 409 → 地址的唯一（409 `issue.link_exists`）→ 时刻 → 写 → 工作项的 `updated_at` → 动态 → 事件（地址变了时投递抓取，3.11） |
  | 发评论 | 读工作项 → 项目 N → 判定 `issue_comment.create`（访客规则）→ 项目已归档 409 → 工作项已归档 409 → 时刻 → 插入评论 → 工作项的 `updated_at` → 关注（评论者、被提及的有效成员）→ 事件 |
  | 改、删评论 | 读评论（得到工作项、项目）→ 项目 N → 判定 `issue_comment.write`（访客规则）→ 改别人的 403 `issue.not_author`；删别人的而不是项目管理员 403 `issue.not_author`（决策点 8）→ 项目已归档 409 → 工作项已归档 409 → 时刻 → 写（删除连带评论的表情）→ 工作项的 `updated_at` →（改）关注被新提及的成员 → 事件 |
  | 加、去表情（工作项、评论） | 读工作项或评论 → 项目 N → 判定 `issue_reaction.write`（访客规则）→ 项目已归档 409 → 工作项已归档 409 →（加）已有 409 `issue.reaction_exists`；（去）没有 404 `issue.reaction_not_found` → 时刻 → 写 →（工作项的表情）工作项的 `updated_at` → 事件 |
  | 存草稿、改草稿、删草稿 | （改、删）读草稿 → 工作区 S → 判定 `issue_draft.write` →（带项目时）项目 S（新旧两个，`id` 升序）→（改、删）草稿行 `FOR UPDATE`（不是自己的、已删除 404；项目变了 409 `issue.draft_changed`）→ 检查（3.15）→ 时刻 → 写 |
  | 发布草稿 | 见上 |
  | 改项目的显示设置（M3，加三列） | 工作区 S → 项目 S → 判定 → 改（M3 照旧，3.17） |
  | 删除状态（M3，加一步） | 工作区 S → 项目 N → 重读状态 → 判定 → `IssueCascade.StateInUse`（不加锁的读，项目 N 之下稳定）→ 在用 409 `project.state_in_use` → 守卫的写 → 数它原来的组（M3 照旧） |
  | 删除标签（M3，加一步） | 工作区 S → 项目 N → 重读标签 → 判定 → 层级检查 → 时刻 → 标签和子标签 → `IssueCascade.DetachLabel`：它们的 `issue_labels`（批量） |
  | 删除项目（M3，加一步） | 工作区 S → 项目 N → 判定 → 时刻 → 项目 → 项目成员 → 显示设置 → 状态 → 标签 → 发布 `project.deleted`（订阅者投递 `issue.delete_rows`，P9 起；P1–P8 的过渡是在这里同步调 `IssueCascade.DeleteProjectIssues`，同下一行的顺序） |
  | 删除项目、工作区之后的任务 `issue.delete_rows`（3.6） | 不取项目的锁；一个项目一个任务（另有一个工作区里没有项目的草稿的），按全局顺序逐张表、按 `id` 的键集，每批一个事务：草稿 → 工作项 → 负责人 → 标签 → 关联行（N，`id` 升序，含另一端在这个项目里的；另一端的项目更早删除的跳过）→ 软删除锁住的其余 → 链接 → 关注 → 提及 → 评论 → 表情 → 评论的表情 → 动态 → 描述版本；每行写请求的时刻和删除者 |
  | 删除工作区（M3，改顺序） | 见上"删除工作区仍先锁它的项目"；发布 `workspace.deleted`，带这次锁住、删除的项目（P9 起；P1–P8 同步调 `DeleteWorkspaceIssues`，用任务的同一组语句，按这些项目和没有项目的草稿） |
  | 自动归档（每个项目一个事务，3.8） | 项目 N（未删除、未归档）→ 时刻 → 一条 `UPDATE` 归档符合条件的工作项 `RETURNING` → 动态（`actor_id` 为空）→ 事件 |
  | 版本的保留（每个项目一个事务，3.14） | 项目 N（未删除）→ 一条 `DELETE` 删掉未删除的工作项的、未删除的版本中每个工作项最新 20 行以外的（软删除的行归清理，3.21） |

- **关于"删除项目"和任务的时刻**：M3 的删除项目在项目 N 之后读时刻；任务（过渡期是同步的连带）在这之后才锁关联行。按上面的论证，等到的关联行要么仍在（随后写下这次删除的时刻），要么已被对方删除（跳过），没有一行的时刻倒退，所以不必把读时刻移到关联行的锁之后。
- **交错测试**（9.3）在真实数据库上确定性地覆盖 26 种交错：
  1. 同一项目两次创建（编号连续、不重复；第二个等第一个提交）；
  2. 创建与删除项目（两种顺序：先建的随项目删除，与项目同一时刻，P9 起由任务写下；后建的 404）；
  3. 修改工作项与删除工作区（两种顺序；删除先拿到工作区 N 而工作项的写已持项目 N 时，删除等它提交，删除的时刻晚于它的 `updated_at`，任务写下的也是这个时刻）；
  4. 修改工作项与移出调用者（移出先提交：写在锁下判定得到 404）；
  5. 降为访客与修改工作项（降级先提交：403，或访客规则下 404）；
  6. 删除状态与创建、修改用它的工作项（状态先删：422 `state_id`；工作项先写：409 `project.state_in_use`）；
  7. 删除标签与给工作项加这个标签；
  8. 归档项目与创建工作项（归档先提交：409 `project.archived`）；
  9. 两次改父工作项（两次移动；加子工作项与移动）各自合法、合起来成环（第二个在锁下沿祖先走，422 `parent_id` 或 `sub_issue_ids[i]`）；
  10. 两个方向相反的跨项目关联同时加（同一对只留一行，后到的 422 `duplicate`；两个项目按 `id` 升序锁，不死锁）；
  11. 删除带跨项目关联的工作项与删除对方项目（关联行按 `id` 升序锁，不死锁；关联行只有一次删除）；
  12. 归档与把它的状态改到"进行中"（先改状态：409 `issue.not_archivable`）；
  13. 删除父工作项与给它加子工作项（删除先提交：404；加先提交：新的子工作项一并被删除）；
  14. 发评论与删除工作项（删除先提交：404；评论先提交：评论被连带删除，同一时刻）；
  15. 同一个人连续两次改描述（动态合并为一行、版本合并为一行，时刻不减）；
  16. 发布草稿与删除它的项目（删除先提交：草稿已随项目删除，P9 起任务还没到时草稿的读也把它当作不存在，404 `issue.draft_not_found`）；
  17. 发布草稿与另一个请求把草稿改到别的项目（409 `issue.draft_changed`）；
  18. 自动归档与把工作项的状态改回"进行中"（改先提交：不归档；归档先提交：修改 409 `issue.archived`）；
  19. 抓取与改链接地址（抓取写 0 行，新的地址另有一次抓取）；
  20. 饥饿与连接池：经真实的连接池和写闸门，一串首尾相连的工作项写（同一工作区的若干项目）不让改工作区、改工作区成员的角色等到期限；删除工作区只等每个项目上正在进行和闸门放进来的写；200 个写压在同一个项目上时，另一个项目上无关的读的耗时不超过上限，多出的写得到 503 `server_busy`（S5、SP1 的回归测试）；
  21. 取锁的形状：每个工作项的写等第一把锁时在工作区行上什么都不持，它取的第一把锁是项目的 N，之后也不取与 `FOR NO KEY UPDATE` 冲突的工作区行的锁（插入经外键取工作区行的 `FOR KEY SHARE`，与它不冲突，S5 的 E3）；看不到项目的账户、看得到而不能写的账户（公开项目里不是成员的工作区成员、不是项目成员的工作区管理员）的写不取行锁、不写；加关联指向看不到的项目时不锁那个项目（不加锁的预检；这一例是自己的测试 `TestARelationLocksNoProjectItDoesNotReach`，与取锁形状的测试的 `createIssueRelations`、`deleteIssueRelation` 两行并列：取锁形状的测试按操作锁表，造不出"看得到工作项、看不到对方的项目"）。用 `pg_locks` 观察，与 M3 的同类测试同一写法；
  22. 移出项目的默认负责人与不带负责人的创建（移出先提交：新的工作项没有负责人；创建先提交：他被指派，随后的移出不改这一行，与 Plane 相同）；
  23. 加关联 A→B 与删除 B 的项目（两种顺序：删除先提交：关联的写在锁下看到 B 的项目已删除，422，不写任何行；关联先提交：随项目删除这条关联行和反向的动态，与项目同一时刻（P9 起由任务写下），没有活的关联行指向已删除的工作项；S5 的 E5、E5b）；
  24. 删除项目 P 的任务与另一个项目 Q 上删除关联另一端的工作项 Y（P9）：P 先删除、任务还没写到时删除 Y，Y 的删除不写这条关联行，P 的任务随后写下 P 的时刻和删除者；Y 先删除（P 还在）时这一行是 Y 的时刻和删除者，P 的任务跳过它；任务与 Y 的删除同时锁关联行时按 `id` 先锁，两种顺序都不死锁；断言每一行的 `deleted_at`、`updated_by_id`，不只是"只删了一次"。任务挂起期间，从另一个项目对已删除项目里的工作项加关联 422、删除指向它的关联 404 `issue.relation_not_found`；
  25. 删除项目 Old，它的任务还没做完时删除它的工作区（P9，聚焦复核 I1）：Old 的每一行工作项一侧的行（含 Old 与工作区另一个项目之间的关联行）都是 Old 的时刻和删除者，工作区其余项目的行都是工作区的；没有项目的草稿是工作区的。M3 的 W3（先删 Old、再删工作区，断言 Old 的行留着 Old 的删除）在 P9 的轮询之下因此确定地通过。
  26. 删除关联与删除它另一端的工作项（P3 加，编号接在最后：20–25 在各文档里被引用）：删除关联不论在哪一端判定都锁两端的项目，与删除工作项在那个工作项的项目行上排队；关联先删，工作项的删除读到它已删除、跳过，留着那次删除；工作项先删，关联的删除在锁下重读两端，读到那个工作项已删除（关联已随它删除），404 `issue.relation_not_found`，什么都不写。
  - 凡有两种顺序的（谁先拿到锁），两种都跑。

### 3.6 连带（总体设计 5.5；M3 交接第 1、2 节）
每一行写下发起者的时刻和删除者（`deleted_at`、`updated_at`、`updated_by_id`），与 M3 相同。删除工作项（连同子树）、删除状态、删除标签的连带在发起它的事务里同步完成；删除项目、删除工作区时，M3 的行在请求的事务里同步完成，**工作项一侧的行由后台任务分批完成，写下请求的时刻和删除者**（决策点 7，负责人裁定 A；P9 起，P1–P8 是同步的过渡，3.3）。Plane 的连带由 Celery 在事务之外异步完成（`API/bgtasks/deletion_task.py:17-105`），差异清单第四节登记的"同一事务"随之改为只对前三种成立（3.23）。

- **删除工作项连带整棵子树**（Plane 相同：`parent` 是 `CASCADE`，删除父工作项递归软删除子工作项，`API/db/models/issue.py:114-120`；R2 第 1 节）。
  - 子树在项目 N 之下用 `WITH RECURSIVE` 列出，含已归档的子工作项；父子都在同一项目（3.9），子树不出这个项目。
  - 子树里每个工作项和挂在它们下面的每一行（负责人、标签、两个方向的关联、链接、关注、提及、评论和评论的表情、表情、动态、描述版本）一并软删除，同一个时刻、同一个删除者。
  - 子工作项不另写动态（它们的动态也一并删除）；子树里每个被删除的工作项各发布一个 `issue.deleted`（3.19），与加子工作项时每个被改的子工作项一个 `issue.updated` 相同，M7 的通知、M8 的 Webhook 按工作项接收。
  - 页面的删除确认框写明子工作项（含更下层的、已归档的）会一并删除。Plane 的确认框不提（`delete-issue-modal.tsx:117-126`），这是页面的改动，登记前端改动清单。
  - **动态随工作项删除**：Plane 的 `issue_activities.issue_id` 是 `DO_NOTHING`，软删除不连带它，动态在工作项删除之后仍是"活的"行（R1 第 11 节）。Nerve 一并软删除，物理清理时由外键 `CASCADE` 删除（4.16）；没有任何读取路径会读一个已删除工作项的动态。登记差异。
  - 删除不写"已删除"的动态（Plane 写一条 `field = issue` 的行，`API/app/views/issue/base.py:728-739`，页面不渲染，R3 第 3 节第 2 行）：那一行写下之后随工作项一起软删除，没有读者。事件 `issue.deleted` 是 M7、M8 的来源。登记差异。
  - **不发布工作项事件的连带**：删除项目、工作区之后由任务删除的工作项，和删除标签摘下的 `issue_labels`（`DetachLabel`），不发布 `issue.deleted`、`issue.updated`。前者由 `project.deleted`、`workspace.deleted` 两个事件说出（P9，3.19），M8 的 Webhook 可以订阅它们；后者没有事件，属于 M8 给标签加的那一批（M3 交接 M8 的一条）。这份清单写进 M8 的交接（13.2）。
  - e2e 的 `expectIssueDeleted`（`e2e/fixtures/assert/issue.ts`）照 `expectProjectDeleted`（`e2e/fixtures/assert/project.ts:190-214`，它的表由 `e2e/fixtures/assert/catalog.ts` 的 `tablesUnder` 读出）的写法从目录读表，但**沿外键图从 `issues` 往下递归地走**：`comment_reactions` 指向 `issue_comments` 而不是 `issues`，只看直接指向 `issues` 的表就断言不到它（对抗性评审 M16）；断言子树的每一行都在同一时刻、由同一个账户软删除。
- **删除项目、删除工作区**（决策点 7，负责人裁定 A；P9 起）：
  - **请求的事务**：M3 的连带照旧（项目或工作区的行、成员、显示设置、状态、标签；删除工作区的邀请一步照旧同步，M3 交接第 1 节），然后发布 `project.deleted`（删除项目）或 `workspace.deleted`（删除工作区；`Changes` 是这次锁住、删除的每个项目一项，正是在这个时刻删除的那些，3.19）。`issue` 的订阅者在同一个事务里经只投递的客户端投递任务 `issue.delete_rows{scope, id, at, by}`，`at`、`by` 是请求的时刻和删除者：删除项目投递一个（`scope = project`）；删除工作区为每个这次删除的项目投递一个（`scope = project`），另加一个 `scope = workspace_drafts`（`id` 是工作区，只写 `project_id IS NULL AND workspace_id = $w` 的草稿）。事务回滚，任务也不在。
    - **一个已删除的项目只属于它自己的任务**（聚焦复核 I1）：更早删除的项目（时刻 `t0`）不在 `workspace.deleted` 的项目里，它的行只由它自己的任务写下 `t0` 和它的删除者。第三稿的"删除工作区是一个按 `workspace_id` 的任务"会把这样的项目里任务还没写到的行写成工作区的时刻和删除者（两个任务在 `default` 队列的 10 个 worker 上通常同时运行），打破下面的"一个改变一个时刻"，M3 的 W3（先删项目 Old、再删工作区，断言 Old 的行留着 Old 的删除）在 P9 时过时不过。按项目的任务也让每个任务有界：一个目标规模的项目约 66 万行（下面）。
  - **任务**（worker 在 `issue/adapter/river`，调 `issue/app/delete_project_rows.go` 的编排，编排经仓储的端口调 `issue/adapter/postgres/delete_rows.go` 的语句；`default` 队列）：按全局顺序逐张表，每张表一个按 `id` 的键集循环，每批一个短事务、一条语句：`UPDATE t SET deleted_at = $at, updated_at = $at, updated_by_id = $by WHERE id IN (SELECT id FROM t WHERE project_id = $p AND deleted_at IS NULL AND id > $after ORDER BY id LIMIT $batch FOR NO KEY UPDATE) RETURNING id`，下一批的 `$after` 是这一批最大的 `id`，直到 0 行；主键的索引直接服务它，不加索引。第三稿的语句没有 `id > $after`，每批都从最小的 `id` 走过前面各批已删除的行：SP4 量到一个项目的 40 万行动态 53.9 秒（每批 7 → 295 ms），键集 1.8 秒（每批约 3 ms，聚焦复核 I2）。关联行另按 `related_issue_id` 匹配另一端在这个项目里的行（R1 第 4 节的陷阱），按 `id` 先锁，另一端的项目更早删除的跳过（3.5"一条关联行属于先删除的那一端"）。带 `deleted_at IS NULL` 所以幂等；重试从每张表第一个未删除的 `id` 重新开始（每次运行一遍扫描）。每批 `issue.delete_batch` 行（默认 1000）。
    - **时间的预算**：worker 覆盖 River 的 `Timeout()`，取 `issue.delete_budget` 加 5 秒（预算默认 45 秒；River 默认的期限是 1 分钟，`river@v0.47.0/client.go:53`，`platform/jobs` 没有改它）。它在每批之前看已用的时间，预算用完而还有行时返回 `river.JobSnooze(0)`：任务重新排队、立即可取，不算一次尝试（`river@v0.47.0/error.go:31-40`），所以一个大的删除不会耗尽尝试的次数，也不会被期限打断在半批上。真的出错（数据库不可用一类）照 River 默认的重试（25 次，`attempt^4` 秒的退避）。
    - **规模**：键集之下每 1000 行约 3 ms（SP4），一个目标规模的项目的任务在一次运行里做完。P9 的量测是一条通过线：五个目标规模的项目的工作区，每个任务都在一次运行里做完（没有 snooze、没有重试），每个任务的耗时记进 spec。
  - **一个改变一个时刻仍成立**：每一行写的是删除它的那次请求的时刻和删除者，与同步的连带写下的相同，只是晚一些写：每个项目的行只由它自己的任务写；跨项目的关联行属于先删除的那一端，活的删除工作项和别的任务都跳过它（3.5）；请求先锁项目、再读时刻（3.5 重新推过），所以没有一行的时刻倒退。交错 24、25 断言每一行写下的是谁的删除。
  - **窗口期间的读写**：请求提交之后、任务做完之前，工作项一侧的行还是"活的"，但它们的项目已删除。每个读写都经项目的门（`ProjectGate`、`access` 的 `ProjectFacts` 都带 `deleted_at IS NULL`，`server/internal/modules/project/adapter/postgres/queries/access.sql:1-8`）或可读的项目的目录（`VisibleProjects` 只列未删除的项目），所以这些行对每个读写都等于不存在，**答案与任务做完之后相同**：按 id 的读和写（预检 404）、按编号取（只认未删除的项目的标识）、搜索和个人主页（`VisibleProjects`）、项目的列表（项目 404）、别的项目里的关联列表（对方看不到）、从别的项目加关联（422）和删除指向它的关联（404），删除别的项目里的工作项不写它与这里的工作项之间的关联行（3.5，交错 24）、草稿（草稿的读按这个工作区未删除的项目过滤：`project_id` 为空或在 `ProjectGate` 给出的未删除项目里，否则当作不存在）、游标（每次按请求重新施加可见性，8.6）。动态里引用这些工作项的行按 3.13 的规则判定，那条规则本来就不看 `deleted_at`，任务前后答案相同。9.3 的"窗口"测试逐条核对：在任务跑之前和之后各问一次，答案相同。
  - **任务失败**：真的出错时 River 按它的退避重试（预算用完的 snooze 不是失败）；重试用完时记一条错误日志，行仍是窗口期间的样子（对任何读写都不存在），60 天之后项目（工作区）的行被物理清理时，外键的 `CASCADE` 把它们物理删除（3.21）。
  - **与清理的先后**：任务写的 `deleted_at` 是请求的时刻，清理按 `deleted_at < 现在 − 60 天` 删除，所以任务在 60 天之内做完时清理碰不到它还没写的行。任务挂起超过 60 天时：`issue` 的清理者跳过仍是活的行，`project`（`workspace`）的清理者物理删除项目（工作区）的行，外键的 `CASCADE` 把剩下的工作项一侧的行一起物理删除；任务随后写 0 行、完成。两者若恰好同时碰到同一批行，按不同的顺序可能死锁（40P01），一方回滚、由 River 重试。
  - **e2e**：M4 的每张表都有 `project_id`（草稿可空）、`deleted_at`、`updated_by_id`。`expectProjectDeleted`、`expectWorkspaceDeleted` 从目录读表（`workspaceTables` 改为从目录读出，2 节）；P9 起它们对工作项一侧的表**限时轮询数据库**，等到每一行都与项目同一时刻、由同一个账户软删除（总体设计 8.2 允许"限时轮询数据库"）。M3 的删除故事在删除之前经 `furnishIssueRows` 给每张表造行（2 节，每个加表的 Phase 扩充它；对抗性评审 I6）。
- **删除状态**：有未删除的工作项（含已归档的）用它时拒绝，409 `project.state_in_use`（Plane 同样拒绝，计 `Issue.objects`，`API/app/views/state/base.py:123-130`；M3 的 `deleteState` 还没有这个检查，P1a 加入，M3 交接第 4 节）。所以 `issues.state_id` 的外键 `CASCADE` 不会删掉活的工作项（4.2）。
- **删除标签**：连同子标签，软删除它们的 `issue_labels` 行（Plane 相同，经 `CASCADE` 的遍历；M3 交接第 4 节）。不写动态（Plane 也不写）。
- **项目成员离开、被移出、账户停用**：不改工作项。他仍是已有工作项的负责人、关注者（Plane 相同）；他不能被新指派（3.7）。Plane 的读取只返回"在某个项目里仍有效"的负责人（`API/app/views/issue/base.py:541-551`，范围不限于这个项目，是缺陷）；Nerve 返回全部未删除的负责人行，页面从工作区成员列表（含已结束的成员关系）取他的显示名。登记差异。
- **归档项目**：不改工作项。已归档的项目里的工作项可以读，一切写 409 `project.archived`（3.8）。
- **连带的数量**（决策点 7）：一个 2 万个工作项的项目约有 66 万行工作项一侧的行，SP2 量到在一个事务里软删除它们要 8.6 秒，五个这样的项目的工作区 36.5 秒，请求的期限是 15 秒。同步的做法在 v0 自己的规模目标（一个项目至多约 2 万个未删除的工作项，3.16）上删不掉一个工作区，每次重试还先锁住工作区和它的每个项目 15 秒。所以选 A（负责人裁定）：请求只做 M3 的几步，工作项一侧交给任务。不论怎样裁定，P1b 在目标的规模上量：一个 2 万个工作项的项目、一个五个这样的项目的工作区，请求的耗时和工作项一侧的软删除的耗时（P1b 时还是 P1a 写下的同步的过渡）分开写进 P1b 的 spec；P9 在同样的数据上量任务做完的时间和它期间别的项目的写的延迟。

### 3.7 工作项的字段规则
- **标题**（`name`）：创建时必填；去掉首尾空白之后 1–255 个字符（Plane 的页面，`issue-modal/components/title-input.tsx:45-63`）；修改时带上就不能为空。数据库 `CHECK (name <> '')`。
- **描述**（`description_html`）：
  - **随它的数据一起进入接口**（第 12 节约束 5）：描述、清洗、提及、描述的版本在 P4 一起到来（评论也要清洗）。P1a 建的 `issues` 已有 `description_html`、`description_stripped` 两列（取默认值），但 P1–P3 的 `IssueCreate`、`IssueUpdate`、`IssueDetail` 没有描述，`IssueDetail` 那时与 `Issue` 同形；P4 给三个结构加上它（5.2）。
  - 用 bluemonday 清洗（负责人 2026-10-25 批准 P4/A1：实现集中在 `issue/adapter/content`，app 声明 `HTMLSanitizer` 小端口，domain 只保留纯值与规则，由 `issue/module.go` 组装；不放宽架构守卫）：放行表是 Plane 编辑器用到的标签和属性，照 Plane 的 nh3 放行表（`API/utils/content_validator.py:71-160`：nh3 的默认表，加 `mention-component`、`label`、`input`、`image-component`；按标签的属性表；网址只许 `http`、`https`、`mailto`、`tel`）。清洗不拒绝，只去掉（Plane 相同）。P4 的 S3 做往返测试（9.1）：编辑器每种节点产生的 HTML 经清洗不变；经接口写入的 HTML（列表、标题、嵌套的标记，没有块 id）经清洗、再经编辑器重新序列化，结果记进样例，下面的 `skip_activity` 用它们；另有只改了一个链接地址、只勾了一个勾选框的样例，它们必须算作修改。`image-component` 留在放行表里：M5 之前编辑器不产生它（3.1），M5 不必再改放行表。
  - `description_stripped`：清洗后的纯文本（bluemonday 的 `StrictPolicy` 加空白的规整），描述为空时为空。M4 只写不读；读者是 M7 通知的摘要和 M8 的 Webhook 载荷（13.2）。总体设计 6.9"供搜索使用"改写（3.23）：搜索只匹配编号和标题（3.18）。
  - 长度：接口描述 `maxLength: 500000`（字符），请求体的上限 1 MiB 照旧（`server.max_body_bytes`，`server/configs/config.yaml`）；Plane 是 10 MB（`content_validator.py:16`）。登记差异。
  - 没有 `description_json`、`description_binary`：Plane 的 `description_json` 没有读者（R1 第 1 节），Nerve 的编辑器还在发它（`core/components/editor/rich-text/description-input/root.tsx:157`），P11 删去；`description_binary` 是协同编辑的遗留，差异清单已登记删除。
  - 创建时不带描述：`<p></p>`（Plane 的默认值）。
- **只补块 id 的保存（`skip_activity`）**（P4，与描述一起）：
  - Nerve 的编辑器在可编辑的状态下打开一段没有块 id（`data-id`，`web/packages/editor/src/extensions/unique-id/extension.ts:63,78,85`）的描述时，会补上 id 并保存一次（`use-editor.ts:93` 的 `isMigrationUpdate`；只读的编辑器不补）。这不是用户的修改。在不分人和代理的 Nerve 里，这是常态而不是少数：经 PAT 建的工作项，描述都没有块 id，第一个打开它的成员就触发这次保存；编辑器重新序列化列表、标题、嵌套的标记时，HTML 与清洗之后存下的也不逐字相同。
  - `IssueUpdate` 有一个布尔字段 `skip_activity`。它只能与 `description_html` 一起出现、不能带别的字段，否则 422（`skip_activity`，`not_allowed`）。
  - **服务端的判断**（控制者的裁定，独立评审 I6 的选项 b，对抗性评审 I4 补全）：带 `skip_activity` 的保存总是写下新的描述；它的**内容标识**与存着的相同时，这次保存算"内容不变"：不写动态、不写版本、不发布事件、**不改 `updated_at`、`updated_by_id`**。内容标识是清洗之后的 HTML 的一个规范的投影（由 `issue/adapter/content` 解析生成，`issue/domain` 的纯值承载并比较）：纯文本（`description_stripped`）、按出现顺序的提及的 id、按出现顺序的 `a[href]`、按出现顺序的每个 `input` 的 `checked` 状态（M5 打开图片之后再加按顺序的 `image-component[src]`，写进 M5 的交接，13.2）。任何一项不同，就是一次普通的描述修改（动态、版本、事件、`updated_at` 都照常），所以这个字段不能用来隐藏文字、提及、链接的去处或勾选框的状态的改动。
    - 为什么要后两项：放行表留着 `a` 的 `href` 和 `input` 的 `checked`（Plane `API/utils/content_validator.py:113,153`）。只比较纯文本和提及时，把描述里的链接从 `https://docs.example` 换成 `https://evil.example`、勾上或去掉一个任务的勾，纯文本和提及都不变：任何成员或 PAT（含被盗的）带上 `skip_activity` 就能改，不留动态、不留可还原的版本、不发事件（第二稿的写法，对抗性评审 I4）。
  - **剩下的口子**：只改格式（加粗、列表的层次、标题的级别）而内容标识不变的保存，带上 `skip_activity` 就不留动态和版本。这是比较规范投影的代价，换来经接口建的工作项被打开时不产生一条算在打开者名下的修改；格式不改变描述说了什么、指向哪里、勾了什么。登记差异（4.17）、写进 8.2。
  - Plane 只看客户端的标志，并照样改 `updated_at`（`API/app/views/issue/base.py:632,684-712`）。不改 `updated_at` 是因为打开一个旧工作项不应让它在"最近更新"的排序里前移，也不应推迟它的自动归档。登记差异。
  - 创建弹窗也要读 `isMigrationUpdate`（M1 收尾交接）：问题出在弹窗**编辑**一个已有的工作项或打开一份草稿时（新建时描述是空的），打开没有块 id 的描述、不做改动，表单不应被标成已改；P14 修正，vitest 和 C4 都写这两种情形。
- **状态**（`state_id`）：这个项目未删除、不是分诊的状态，否则 422（`state_id`，`not_allowed`）。创建时不带：项目的默认状态（M3 保证每个项目恰有一个，差异清单二·按表 `states`）。Plane 相同（`API/app/serializers/issue.py:169-176`；默认值 `API/db/models/issue.py:228-238`）。
- **优先级**（`priority`）：`urgent`、`high`、`medium`、`low`、`none`，默认 `none`。
- **起止日期**（`start_date`、`target_date`）：`YYYY-MM-DD`（`format: date`）；可以为 `null`。两列在 P1a 随 `issues` 建出，接口在 P2 才有这两个字段，同一个 Phase 让 `bodyshape` 和它的生成器认这个格式（6.1；第 12 节约束 5：字段随它的数据和检查进入接口；不放在 P1a、P1b，是为了不与它们的加锁约定同一段评审，对抗性评审 M21）。开始不晚于截止，按**合并之后的值**检查：只传一个日期时与存着的另一个比较，否则 422（`target_date`，`not_allowed`；只传了 `start_date` 时指向 `start_date`）。Plane 只在两个都在请求里时检查（`API/app/serializers/issue.py:128-133`），页面两个选择器互相约束（`issue-detail/sidebar.tsx:74-78`）。相等允许。数据库另有 `CHECK (start_date <= target_date)`。登记差异。
- **负责人**（`assignee_ids`）：每一个都必须是这个项目的有效成员、项目角色不低于成员，否则 422（`assignee_ids[i]`，`not_allowed`）；重复 422（`duplicate`）；最多 100 个。Plane 静默地去掉不合格的（`API/app/serializers/issue.py:150-156`），页面只列出项目中不是访客的成员（`dropdowns/member/dropdown.tsx:36-40`）。登记差异。修改时带上就是新的集合，服务端按差集增删行（Plane 删除全部再插入，`:286-326`，每次保存都换一批行），只为增删的写动态。
- **默认负责人**：创建时 `assignee_ids` 不带或为空、项目的 `default_assignee_id` 现在是这个项目中不是访客的有效成员时，指派给他；否则没有负责人，**不报错**（Plane 相同，`API/app/serializers/issue.py:232-253`）。成员离开项目之后 `default_assignee_id` 仍指向他（M3 照 Plane 不清空），所以创建时要现查他的成员关系（M3 交接第 4 节）。这条规则写进 `createIssue` 的接口描述。默认负责人也自动关注（3.12）；Plane 不让他关注（R2 第 7 节，缺陷）。
- **标签**（`label_ids`）：每一个都是这个项目未删除的标签，否则 422（`label_ids[i]`，`not_allowed`）；重复 422；最多 100 个。Plane 静默地去掉（`:159-166`），登记差异。
- **编号**（`sequence_id`）：`projects.last_issue_sequence` 在项目 N 之下加一（总体设计 5.3）。数据库 `UNIQUE (project_id, sequence_id)` 是全表唯一（不是部分唯一）：计数器从不回退，编号从不重用，软删除、物理清理之后也一样（Plane 的 `issue_sequences` 行在删除后保留，`API/db/models/issue.py:557-571`）。客户端不能写它（Plane 的序列化器允许写，R2 第 1 节，缺陷）。
- **排序值**（`sort_order`，`double precision`）：创建时取这个项目、这个状态的未删除工作项（含已归档的）的最大值加 10000，没有时 65535（Plane `API/db/models/issue.py:158,206-210`；总体设计 5.5）。创建的请求体没有这个字段；修改可以写它（拖动，3.16）。
- **完成时间**（`completed_at`）：创建时状态在"已完成"组为当前时刻；改状态时，新状态在"已完成"组为当前时刻（从一个完成状态改到另一个完成状态也重置，Plane 相同，`:240-255`），否则为空；"已取消"不设。客户端不能写它。
- **`updated_at` 的含义**：工作项本身或挂在它下面的行的每一个写都改工作项的 `updated_at`、`updated_by_id`：改字段、负责人、标签、父工作项（被改的子工作项）、评论的增删改、链接的增删改、关联的增删（两边）、工作项的表情、归档和恢复。评论的表情不改（Plane 相同：评论的表情的动态 `issue_id` 为空，`API/bgtasks/issue_activities_task.py:1531-1537`）。只补块 id 的保存不改（上面）。自动归档写 `updated_by_id = NULL`（决策点 4）。
  - 为什么照 Plane 让评论、链接也改它：自动归档按 `updated_at` 判断"多久没有动静"（3.8）；有人还在评论的工作项不应被自动归档（Plane 的同一个效果，R2 第 0.7 节）。
- **不能移动到别的项目**：`project_id` 只在路径里，请求体没有它（Plane 相同，`API/app/serializers/issue.py:100`）。
- **按编号取**：`GET /workspaces/{slug}/issues/{identifier}`，`identifier` 是"项目标识-编号"，标识不分大小写。M3 的标识只有大写的 `A`–`Z`、数字和 `ÇŞĞİÖÜ`（M3 设计 3.19），`I` 与 `İ` 是两个字母；Go 的 `strings.ToUpper("iş")` 是 `"IŞ"` 而不是 `"İŞ"`，只转大写会让 `/browse/iş-12` 找不到 `İŞ-12`（对抗性评审 M12）。所以两边经同一个折叠比较（`issue/domain` 的 `foldIdentifier`）：`a`–`z` 转成 `A`–`Z`，`çşğöü` 转成 `ÇŞĞÖÜ`，`i`、`ı`、`I`、`İ` 都折成同一个字母。一个工作区里只有 `I`、`İ` 之别的两个或更多标识折叠之后相同时，逐字等于 `strings.ToUpper` 之后的那一个优先，否则 404。样例写在 `api/testdata/identifier-fold.json`（`issue-order.json` 的写法），Go 的测试读它；前端不折叠，把路由里的文字原样交给接口。搜索的编号匹配用同一个折叠（3.18）。格式不对、项目不存在、看不到、编号不存在，都是 404 `issue.not_found`（Plane 对格式不对答 400、对不是成员答 403，透露项目存在，`API/app/views/issue/base.py:1202-1230`；登记差异）。已归档的工作项照样返回（Plane 相同）。
- **客户端不能写的字段**：Plane 的序列化器 `fields = "__all__"`，客户端可以直接写 `sequence_id`、`archived_at`、`deleted_at` 等几乎一切（R2 第 1 节）。Nerve 的请求体是封闭的结构（总体设计 3.2：未知字段 400），只有上面列出的可写字段。差异清单二·全局已有"封闭的请求体"一条，这里不另登记。

### 3.8 归档、恢复与自动归档（决策点 4 负责人裁定 A）
- **归档**：工作项的状态组必须是"已完成"或"已取消"，否则 409 `issue.not_archivable`（Plane 400，`API/app/views/issue/archive.py:258-263`）。`archived_at` 是 `timestamptz`（差异清单已登记；Plane 是日期）。已归档的再归档：不写，200 返回工作项（Plane 404）。
- **恢复**：不检查状态组（Plane 相同，`:281-302`）；未归档的恢复：不写，200。父工作项是否归档不影响子工作项，反之亦然（Plane 相同）。
- **归档、恢复的响应**是完整的 `IssueDetail`（总体设计 3.1；Plane 返回 `{archived_at}` 和 204）。
- **已归档的工作项只读**：除恢复和删除之外，它和它下面的一切写（改字段、加子工作项、关联的两端、链接、评论、表情）都是 409 `issue.archived`。Plane 只挡住了修改（`issue_objects`，404），评论、链接、表情、关联不挡（R2 第 11 节），页面全部禁用（`main-content.tsx:92-183`）。按页面收紧，登记差异。
  - 已归档的工作项不能成为新的父工作项（422 `parent_id`，`not_allowed`；Plane 的 PATCH 允许，R2 第 11 节）；不能被加为子工作项、不能成为关联的对方（422）。
  - 已归档的工作项不出现在：项目的工作项列表（默认）、子工作项列表、关联列表（3.10）、搜索和选择器（3.18）、个人主页；出现在 `?archived=true` 的列表。Plane 的 `issue_objects` 同样排除它们（R2 第 0.3 节）。父工作项的 `sub_issues_count` 不计已归档的子工作项（Plane 相同）。
- **归档的列表**：`GET /projects/{project_id}/issues?archived=true`，只返回已归档的，与项目的列表同一个引擎（3.16）；规则 `issue.archive`（访客 403）。
- **已归档的项目**：项目的列表照常返回它的工作项（可以读，总体设计 5.5"仍然可以查看"）；一切写 409 `project.archived`；跨项目的列表（个人主页、搜索、选择器、跨项目关联的对方）不含已归档的项目里的工作项（Plane 的 `issue_objects`）。页面遇到已归档的项目显示 M3 的界面，不请求列表（M3 设计 3.19）。Plane 的项目列表对已归档的项目返回空（`issue_objects`），登记差异。
- **自动归档**（总体设计 5.5，Plane `API/bgtasks/issue_automation_task.py:22-86`）：
  - 定时任务 `issue.auto_archive`（间隔可配置，`issue.auto_archive_interval`，默认 `24h`，测试 `2s`；3.20）。
  - 对每个设了 `archive_in`（1–12 个月）、未归档、未删除的项目，一个事务：项目 N → 读时刻 → 一条 `UPDATE` 归档符合条件的工作项：未删除、未归档、状态组是"已完成"或"已取消"、`updated_at <= now - archive_in × 30 天` → 每个一条动态 → 每个一个事件 `issue.archived`。
  - 不在 M4 的条件（转交，13.2）：工作项不在未结束的迭代里、不在目标日期未过的模块里（M6）；收集箱的工作项只有已接受、已拒绝、重复的（M7）。M4 没有迭代、模块、收集箱，这三个条件对 M4 的数据恒为真，由各自的 M 随表加入任务的查询。
  - 写 `archived_at = now`、`updated_at = now`、`updated_by_id = NULL`；动态 `field = archived_at`、`new_value = archive`、`actor_id = NULL`（3.13）；事件的 `actor_id` 为 `null`。
  - **`actor_id = NULL` 表示 nerve 自己**（决策点 4，负责人裁定 A）：Plane 记为项目的创建者（`issue_automation_task.py:73`），把一个自动的动作算在一个人头上，PAT 的读者、通知（M7）、Webhook（M8）都会读到它。Nerve 不建系统账户（没有人和代理之分）。页面的规则改为通用的：动态的 `actor_id` 为空时显示站点名（`SITE_NAME`，今天是 "Nerve"，`web/packages/constants/src/metadata.ts:7`），不再看 `new_value === "archive"`（`archived-at.tsx:39`）。v0 不物理删除账户（M2 设计 4.7），所以 `actor_id` 为空只有这一种含义；`ON DELETE SET NULL` 永不触发。
  - 自动归档不让任何人关注（Plane 相同，`subscriber=False`），不改父工作项、子工作项。

### 3.9 父子
- **父工作项**（`parent_id`，创建和修改都可以带）：同一个项目、未删除、未归档；不是自己（数据库 `issues_not_own_parent_check CHECK (parent_id <> id)`）；不是自己的后代。不合规 422（`parent_id`，`not_allowed`）。不限层数（Plane 相同）。
  - 后代的检查在项目 N 之下沿新父工作项的祖先链向上走（`WITH RECURSIVE`），遇到自己就是成环。同一个项目里的父子改动都持这个项目的 N（3.5），走的时候祖先链不会变。
  - Plane 不检查自己、不检查成环（`API/app/serializers/issue.py:179-186`），选择器只排除直接的子工作项，从页面就能造出环（R2 第 2 节）。登记差异。
- **加子工作项**：`POST /issues/{issue_id}/sub-issues`，`{sub_issue_ids: [...]}`（1–100 个，不重复）。每一个都要：同一个项目、未删除、未归档、不是父本身、不是父的祖先（否则成环），否则 422（`sub_issue_ids[i]`，`not_allowed`）；已有别的父工作项的，改为这个父工作项（Plane 相同）。父已归档 409 `issue.archived`。每个被改的子工作项一条 `parent` 动态，`old_value` 是它原来的父工作项的编号（Plane 写成子工作项自己的编号，`API/app/views/issue/sub_issue.py:258`，缺陷；R3 第 8 节第 6 条）。响应 `{data: Issue[]}`：改过的子工作项。
- **去掉父工作项**：`PATCH` 子工作项的 `parent_id: null`（Plane 相同，没有专门的接口）。
- **子工作项列表**：`GET /issues/{issue_id}/sub-issues`，直接的子工作项，不含已归档的（Plane 相同），按访客规则过滤；集合型，不分页（3.16 末尾）。Plane 的响应带 `state_distribution`（按状态组分的 id），Nerve 不带：页面从子工作项的 `state_id` 和状态目录算出（P11）。
- **`sub_issues_count`**：未删除、未归档的直接子工作项中调用者看得到的个数（Plane 数全部的）：项目里"只看自己"的访客只数他自己建的（3.4），别人数全部；按它排序用同一个数（3.16）。

### 3.10 关联
- **四种类型**，与 Plane 的页面相同：`relates_to`（关联）、`duplicate`（重复）、`blocked_by`（被阻塞）、`blocking`（阻塞）（`web/packages/types/src/issues/issue_relation.ts:17`）。Plane 的模型另有 `start_before`、`finish_before`、`implemented_by`，属于已砍的时间线和企业版（R2 第 3 节），接口的枚举里没有。登记差异。
- **存储**：一对工作项一行。`relates_to`、`duplicate` 对称，存为（本工作项，对方）；`blocked_by` 存为（本工作项，对方）；`blocking` 反过来存为（对方，本工作项，`blocked_by`）（Plane 相同，`API/app/views/issue/relation.py:229-246`）。数据库 `CHECK (relation_type IN ('blocked_by', 'duplicate', 'relates_to'))`、`CHECK (issue_id <> related_issue_id)`，部分唯一 `ON (LEAST(issue_id, related_issue_id), GREATEST(issue_id, related_issue_id)) WHERE deleted_at IS NULL`：一对工作项之间最多一条关联，不论类型和方向。Plane 的唯一键有方向、不看类型，A 被 B 阻塞和 B 被 A 阻塞可以并存（缺陷；页面的选择器排除已有关联的工作项，R2 第 3 节），按页面收紧，登记差异。
- **行的 `project_id`** 是存储的 `issue_id` 那一端的项目（Plane 写成请求路径里的项目，`relation.py:237-238`，缺陷）。
- **加关联**：`POST /issues/{issue_id}/relations`，`{relation_type, issue_ids: [...]}`（1–20 个，不重复）。每个对方都要：同一个工作区、对调用者可见（他在它的项目上有 `issue.read`、并且过了那个项目的访客规则，3.4）、未删除、未归档、所在项目未归档、不是自己，否则 422（`issue_ids[i]`，`not_allowed`）；这一对已有关联（任何类型、任何方向）422（`duplicate`）。可以跨项目（Plane 的关联选择器有工作区开关，R2 第 3 节）。Plane 不检查类型（任意字符串）、不检查自己、不检查调用者能否看到对方的项目（R2 第 3 节）。登记差异。
- **响应**：201，`{data: IssueRelation[]}`：新建的关联，从本工作项看。
- **关联列表**：`GET /issues/{issue_id}/relations`，集合型，`{data: IssueRelation[]}`，按关联行的 `(created_at, id)` 升序。只含对调用者可见、未删除、未归档的对方（Plane 显示调用者看不到的项目里的工作项的标题和状态，泄露）。
- **`IssueRelation`**：`{id, relation_type, issue, created_at}`（5.2）。`relation_type` 是从本工作项看的类型（存为对方 `blocked_by` 本工作项的，本工作项看到的是 `blocking`）；`issue` 是对方的列表行（`Issue`，5.2）。这是总体设计 3.6"关联对象只返回 ID"的又一个例外（M3 的成员内嵌用户资料是第一个，M3 设计 11.3）：对方可以在另一个项目里，页面没有它的数据；只给 id，页面要为每个对方再取一次。理由与 M3 相同，登记到总体设计 3.6（3.23）。对方的状态名由页面从工作区的状态列表（M3 的 `listWorkspaceStates`）读出。
- **删除关联**：`DELETE /issue-relations/{relation_id}`，204。调用者能改两端任一个工作项（`issue.update`，在那一端的项目上判定）即可：先判定存储的 `issue_id` 那一端，看不到或不能做时判定另一端；两端都看不到 404 `issue.relation_not_found`，看得到而都不能做 403。Plane 的 `remove-relation` 从任一端的页面调用、只看那个项目的角色（`relation.py:271-293`），关联不存在时 500（缺陷）。两端任一个已归档 409 `issue.archived`。任一端的项目已删除（它的删除任务还没软删除这一行，3.6）时也是 404 `issue.relation_not_found`：与任务做完之后的答案相同。
- **动态**：每次增减两条，正向一条写在本工作项上、反向一条写在对方工作项上，各用从自己看的类型（3.13 第 20–23 行）。两边的 `updated_at` 都改（Plane 的动态任务对两边都改）。
- **对方被删除**：关联行随之软删除（3.6）。**对方被归档**：关联行保留，列表不显示它；恢复之后重新出现（Plane 相同）。

### 3.11 链接与元数据的抓取（架构问题 8）
- **字段**：`url`（必填）、`title`（可选，至多 255 个字符，可为 `null`）、`metadata`（服务端写，客户端不能写）。
- **地址的规整**：去掉首尾空白；不以 `http://`、`https://` 开头（不分大小写）时在前面加 `http://`（Plane 的服务端相同，`API/app/serializers/issue.py:566-572`；页面在不以 `http` 开头时加，`create-update-link-modal.tsx:65-68`）；然后解析：协议只能是 `http`、`https`，必须有主机名，规整后至多 2048 个字符，否则 422（`url`，`not_allowed`）。Plane 的 `URLValidator` 另放行 `ftp`、`ftps`，登记差异。
- **同一个工作项上地址唯一**（规整之后逐字比较，未删除的行之间）：重复 409 `issue.link_exists`，创建和修改都是（Plane 在代码里检查，400，`:585-598`）。数据库另有部分唯一 `ON (issue_id, url) WHERE deleted_at IS NULL`。登记差异。
- **谁能改**：项目的管理员和成员都能改、删任何人的链接（Plane 相同，没有创建者规则，R2 第 4 节）。
- **抓取**（第一个异步订阅者，3.19）：
  - 创建、改地址发布事件 `issue.link_created`、`issue.link_updated`（`changes` 里有 `url`）；订阅者 `crawl_on_link_change` 在同一个事务里投递 River 任务 `issue.crawl_link{link_id, url}`（`InsertTx`，3.20）。事务回滚，任务也不在。只改标题不重新抓取（Plane 相同，`API/app/views/issue/link.py:76-83`）。
  - 改地址时 `metadata` 重置为 `{}`：旧地址的标题不应显示在新地址旁边（Plane 保留到新的抓取覆盖它）。
  - worker 经 `platform/outbound` 的客户端取页面（防 SSRF，8.1、11.8）：只发 `GET`；连接 2 秒、整体 5 秒的期限；至多 5 次跳转，每一跳重新检查；响应体至多读 1 MiB（Plane 不设上限，`work_item_link_task.py:128`，缺陷）。取 `<title>`（反转义、去空白、至多 255 个字符）。图标：页面里 `rel` 为 `icon`、`shortcut icon`、`apple-touch-icon`、`apple-touch-icon-precomposed` 的 `<link>`，按最终地址解析，没有时取 `/favicon.ico`；同样经出站客户端取，至多 64 KiB、类型是 `image/png`、`image/x-icon`、`image/vnd.microsoft.icon`、`image/gif`、`image/jpeg`、`image/webp` 之一，存为 `data:` 地址；否则 `favicon` 为 `null`，页面显示默认的链接图标。Plane 不限大小和类型，默认存一个内嵌的 SVG（`:205-245`）；Nerve 不存 SVG。
  - 写回：`{title, favicon, url, favicon_url}`；失败时 `{error, url}`，`error` 只有两个值：`blocked`（地址被出站客户端拒绝，6.8）、`failed`（其余的一切：超时、连不上、状态码、超过上限、不是 HTML）。第二稿分成六种，让抓取成了一个扫端口的回显（对抗性评审 M3）；页面只用"有没有标题"，不读 `error` 的细分，所以收粗到页面要的程度，失败的种类只写进 nerve 的日志。写回用 3.5 的带守卫的 `UPDATE`：链接已删除或地址已改时写 0 行。不写 `updated_at`、不写动态、不发布事件（抓取不是谁的修改）。
  - 网络上的失败不重试（写下 `error`，Plane 也不重试）；数据库的失败让任务返回错误，由 River 重试（至多 3 次）。
  - 页面显示 `title || url`，后面灰色的 `metadata.title`，`metadata.favicon` 作图标（`links/link-item.tsx:46-47,58-75`），不变。CSP 的 `img-src 'self' data: blob:` 不变：图标是 `data:` 地址，页面不从外部取图。
- **测试的出口**：e2e 的 I10 要让抓取到达本机起的页面服务器，而出站客户端拒绝回环地址。配置 `outbound.allow_cidrs`（只在 `env = test` 时接受，别的环境启动时拒绝，与 `clock.offset` 同一种做法，3.20）列出放行的网段；`config.test.yaml` 放行 `127.0.0.1/32`，所以 worker 自己的 nerve 和故事起的 nerve 都带着它（两个 nerve 都在工作 `default` 队列，谁取到抓取的任务结果都一样，对抗性评审 M18）。I10 输入 `127.0.0.1:<端口>/page`（页面服务器的端口），规整之后是 `http://127.0.0.1:<端口>/page`；被拒绝的一例用 `169.254.169.254`（链路本地，云的元数据地址），e2e 只断言 `error: blocked`，"没有发出连接"由 6.8 的 Go 表测试证明。
- **限流**：不另设桶（5.4）。每个链接最多引起页面和图标两次出站请求（加跳转），抓取在队列里由固定数量的 worker 执行（3.20），出站的速率不随请求的速率增长。

### 3.12 评论、表情回应、关注与提及
- **评论**（`issue_comments`）：
  - `comment_html` 用描述的同一个放行表清洗；`comment_stripped` 是纯文本；至多 500000 个字符。清洗之后没有文字、没有 `mention-component` 和 `image-component` 时 422（`comment_html`，`required`；页面的规则，`web/packages/utils/src/string.ts:224-244`）。Plane 的服务端接受空评论，登记差异。
  - 改评论时内容变了才写 `edited_at`（Plane 相同，`API/app/views/issue/comment.py:116-119`）。
  - 只有作者能改；作者或项目管理员能删（决策点 8，3.4）；删除连带它的表情。
  - 没有 `access`、`parent_id`（决策点 6）、`comment_json`、`attachments`、`description_id`（3.2、4.7）。
  - 评论列表：`GET /issues/{issue_id}/comments`，按 `(created_at, id)` 升序、游标分页（`limit` 1–100，默认 50），每条内嵌它的表情 `reactions`（5.2，评论自己的子行，与 M3 的成员内嵌资料同一类例外）。Plane 的页面从 `/history/?activity_type=issue-comment` 读评论（R2 第 5 节），Nerve 用这个列表；它与动态列表的增量读取用同一种做法（3.13）。
  - 评论不写动态（3.13）；发评论、改、删都改工作项的 `updated_at`（3.7）。
- **表情回应**（`issue_reactions`、`comment_reactions`）：
  - 值是表情的码位，十进制，用 `-` 连接（`128077`、`10084-65039`；页面的编码，`web/packages/propel/src/emoji-icon-picker/helper.tsx:139-160`）。接口和数据库都检查格式：`^[0-9]{1,7}(-[0-9]{1,7}){0,15}$`，至多 128 个字符。Plane 接受任意文本，登记差异。
  - 一个人对同一个工作项（评论）的同一个表情只有一行：重复 409 `issue.reaction_exists`（Plane 400）。去掉：`DELETE /issues/{issue_id}/reactions/{reaction}`、`DELETE /issue-comments/{comment_id}/reactions/{reaction}`，只去掉调用者自己的，没有 404 `issue.reaction_not_found`。
  - 项目的任何有效成员（含访客）可以对看得到的工作项和评论加表情（Plane 相同）；已归档的工作项 409。
  - 工作项的表情列表：`GET /issues/{issue_id}/reactions`，集合型；评论的表情内嵌在评论里，没有单独的列表。
  - 表情不写动态（Plane 写，但从不显示、不通知，R3 第 7 节），登记差异。工作项的表情改工作项的 `updated_at`，评论的表情不改（3.7）。
- **自动关注**（`issue_subscribers`，总体设计 6.6）：在同一个事务里加入：创建者；实际被指派的负责人（含默认负责人，创建和修改时）；发评论的人；描述和评论里被提及的、能读这个工作项的人（项目的有效成员；"只看自己"的访客只在他是创建者时，3.4）。前三种人本来就能读它（负责人不低于成员，评论者读得到才能评论）。
  - 不是"每个改过工作项的人"：Plane 让每个产生通知的动态的执行者都关注（`API/bgtasks/notification_task.py:292-299`），改一次优先级就关注了，这是它的动态任务的副作用（R2 第 7 节）。登记差异。
  - 插入用 `ON CONFLICT DO NOTHING`（部分唯一索引 `WHERE deleted_at IS NULL`），按账户 id 排序。
  - 去掉负责人不取消他的关注（Plane 相同）。
  - 关注、取消关注的接口和 `is_subscribed` 在 M7（13.2）。
- **提及**（`issue_mentions`）：
  - 每次描述被写入（创建、改描述、还原），从**保存之后**的描述里取出 `<mention-component entity_name="user_mention" entity_identifier="…">`，过滤为项目的有效成员（任何角色），与已有的行做集合同步：新的插入，不在的软删除。
  - Plane 拿请求体与旧描述比较，一个不带描述的修改（改状态）会删掉全部提及行，之后不再补回（R2 第 8 节，缺陷）。登记差异。
  - 评论里的提及不存进 `issue_mentions`，只让被提及的、能读这个工作项的人关注（Plane 让被提及的人都关注）。
  - 被提及而读不到它的"只看自己"的访客：`issue_mentions` 照样有他一行（列表的 `mention_id` 筛选按调用者的可见性过滤，他自己看不到这个工作项），但不关注它；M7 发通知时另按那一刻的读权限再判定（3.4）。
  - 读者：列表的筛选 `mention_id`（3.16）。
  - 编辑器的 @提及 候选由前端从项目成员列表（M3 的 store）过滤得到，不再请求 `entity-search`（R2 第 8 节；它是 `WorkspaceService.searchEntity` 的唯一用途，P11 删除，连同 4 个模块级实例中的 `WorkspaceService`，M2 交接第 12 节）。没有项目的草稿的编辑器用工作区成员。

### 3.13 操作动态
- **写在同一个事务里**（总体设计 6.6；Plane 在 Celery 里异步写，差异清单第四节已登记）。值的格式照 Plane（显示用的字符串，引用的对象的 id 在 `old_identifier`、`new_identifier`），因为页面的渲染器按它判断：增还是减看 `old_value === ""`，设还是去看 `new_value` 是否为空（R3 第 5.2 节）。M4 修正 Plane 写错的几处，渲染器不受影响。
- **M4 写的行**（`verb` 都是 Plane 的值；"—"是 `NULL`）：

  | # | 何时 | `field` | `verb` | `old_value` | `new_value` | `old_identifier` | `new_identifier` | 与 Plane |
  |---|---|---|---|---|---|---|---|---|
  | 1 | 创建 | — | `created` | — | — | — | — | 相同。`actor_id` 是创建者，`created_at` 是工作项的 `created_at`（`issue_activities_task.py:577-579`） |
  | 2 | 创建时的每个负责人 | `assignees` | `updated` | `""` | 显示名 | — | 账户 id | 只为实际指派的写（含默认负责人）；Plane 按请求里的 id 写，含被静默去掉的（`:375-408`，缺陷） |
  | 3 | 改标题 | `name` | `updated` | 旧标题 | 新标题 | — | — | 相同 |
  | 4 | 改描述 | `description` | `updated` | 旧描述 | 新描述 | — | — | 合并：这个工作项最新的一行是调用者的描述行时不新写，改那一行的 `new_value`、`created_at`、`updated_at`，保留它的 `old_value`。Plane 只改 `created_at`，值是陈旧的（`:88-96`） |
  | 5 | 改父工作项（每个被改的子工作项） | `parent` | `updated` | 旧父的编号或 `""` | 新父的编号或 `""` | 旧父 id | 新父 id | 加子工作项时 `old_value` 是原来的父（Plane 写成子工作项自己，缺陷） |
  | 6 | 改优先级 | `priority` | `updated` | 旧值 | 新值 | — | — | 相同 |
  | 7 | 改状态 | `state` | `updated` | 旧状态名 | 新状态名 | 旧状态 id | 新状态 id | 相同（Nerve 的状态不为空，旧值总有） |
  | 8 | 改开始日期 | `start_date` | `updated` | `YYYY-MM-DD` 或 `""` | 同 | — | — | 相同 |
  | 9 | 改截止日期 | `target_date` | `updated` | 同 | 同 | — | — | 相同 |
  | 10 | 加标签（每个） | `labels` | `updated` | `""` | 标签名 | — | 标签 id | 相同；创建时的标签不写（Plane 相同） |
  | 11 | 去标签（每个） | `labels` | `updated` | 标签名 | `""` | 标签 id | — | 相同 |
  | 12 | 加负责人（每个） | `assignees` | `updated` | `""` | 显示名 | — | 账户 id | 相同 |
  | 13 | 去负责人（每个） | `assignees` | `updated` | 显示名 | `""` | 账户 id | — | 相同 |
  | 14 | 手动归档 | `archived_at` | `updated` | — | `manual_archive` | — | — | 相同 |
  | 15 | 自动归档 | `archived_at` | `updated` | — | `archive` | — | — | `actor_id` 为空（决策点 4）；Plane 是项目的创建者 |
  | 16 | 恢复 | `archived_at` | `updated` | `archive` | `restore` | — | — | 相同（手动归档的恢复也写 `archive`，Plane 相同） |
  | 17 | 加链接 | `link` | `created` | — | 地址 | — | 链接 id | 相同 |
  | 18 | 改链接的地址 | `link` | `updated` | 旧地址 | 新地址 | 链接 id | 链接 id | 相同；只改标题不写（Plane 相同） |
  | 19 | 删链接 | `link` | `deleted` | 地址 | `""` | — | — | 相同 |
  | 20 | 加关联，本工作项上 | 从本工作项看的类型 | `updated` | `""` | 对方的编号 | — | 对方 id | 对方 id 在 `new_identifier`（Plane 写进 `old_identifier`，`:1303`） |
  | 21 | 加关联，对方工作项上 | 从对方看的类型 | `updated` | `""` | 本工作项的编号 | — | 本工作项 id | `project_id` 是对方的项目（Plane 是本项目，`:1317`） |
  | 22 | 删关联，本工作项上 | 从本工作项看的类型 | `deleted` | 对方的编号 | `""` | 对方 id | — | 相同 |
  | 23 | 删关联，对方工作项上 | 从对方看的类型 | `deleted` | 本工作项的编号 | `""` | 本工作项 id | — | Plane 的 `comment` 用正向的类型（`field` 只把 `blocked_by`、`blocking` 反过来）、`old_identifier` 是它自己、`project_id` 是本项目（`:1362-1374`），修正 |

  - "编号"是 `PROJ-12`，用写入时的项目标识；名字是写入时的名字（Plane 相同）。
  - **不写的**：删除工作项（Plane 的 `field = issue` 一行，3.6）；评论的增改删、工作项和评论的表情（Plane 写，从不显示，R3 第 7 节；评论在评论列表里，M7 的通知从事件来，3.19）。登记差异。迭代、模块（M6）、附件（M5）、收集箱（M7）的行由各自的 M 写。估算、投票、草稿的行已砍。
- **列与外键**：`issue_id NOT NULL`、`ON DELETE CASCADE`（Plane 是可空、`DO_NOTHING`）；不要 `issue_comment_id`（评论不写动态）、`epoch`（只写不读）、`attachments`（从不写）；`actor_id` `SET NULL`（照搬，永不触发）；保留 `comment` 列，写 Plane 的英文句子去掉品牌（自动归档写 `archived the work item`，不写 `Plane has archived the issue`），接口不返回它，M7 的通知决定用不用（13.2）。4.11 逐列。
- **列表**：`GET /issues/{issue_id}/activities`，按 `(created_at, id)` 升序、游标分页（`limit` 1–100，默认 50），访客规则同工作项（看不到的工作项 404）。每行只给 `actor_id`，不给 `actor_detail`、`issue_detail`、`project_detail`、`workspace_detail`（R3 第 5.2 节：渲染路径只读 `actor_detail` 的 id 和显示名、`workspace_detail.slug`）。Plane 的列表不分页（`API/app/views/issue/activity.py:24-86`），不带 `activity_type` 时 500。
  - **引用了看不到的工作项的行不在列表里**（3.4）：`parent` 的行和关联的行（第 5、20–23 行）的 `old_identifier`、`new_identifier` 是别的工作项；它所在的项目调用者读不到，或它被那个项目的访客规则挡住时，这一行从调用者的列表里略去（`old_value`、`new_value` 里的编号也是信息）。
    - **"读得到"按被引用的工作项所在项目的成员关系判定，不看任何 `deleted_at`**（对抗性评审 M8）：调用者在那个项目里有 `is_active` 的成员关系（项目、成员关系、被引用的工作项是否已软删除都不看，归档也不看），并过了那个项目的访客规则（角色 5 而 `guest_view_all_features` 为假时，只有他自己建的）。所以已删除的对方、已删除的项目里的对方的历史行照常显示给当时就读得到它的人（删除项目软删除成员关系，但不改 `is_active`；离开、被移出才改 `is_active`，M3 的 `00011_project_project_members.sql`）；这条判定在删除项目的任务之前和之后答案相同（3.6）。被引用的工作项已被物理清理（60 天之后）时连接不到它，这一行略去。
    - 目录：`VisibleProjects` 的另一个方法，按上面的口径给出调用者的项目、角色、`OwnOnly`（3.3）；在 SQL 里与访客规则同一种写法，游标照常（略去的行不占位置）。测试：引用已删除的工作项、已删除的项目里的工作项、已物理清理的工作项的行，各一例（9.3）。Plane 显示这些行，登记差异。
- **增量读取**：页面打开详情时读完动态和评论的全部页；之后在每次修改得到应答时，从**产生最后一页的那个游标**再读一次（最后一页的 `next_cursor` 为空时也保留它的请求游标），按 `id` 合并、按 `(created_at, id)` 重排。
  - 为什么不会漏：同一个工作项的动态、评论按提交的顺序得到不减的 `created_at`（3.5"时刻的单调"），新的行都排在已读的行之后。描述的合并把旧行的 `created_at` 改到现在，它也出现在再读的结果里，按 `id` 覆盖。
  - Plane 的页面用 `created_at__gt` 取增量（`activity.store.ts:137-142`）。游标是 Nerve 统一的分页形式（总体设计 3.4），按 `(created_at, id)` 的游标还排除了同一时刻的行被跳过。
  - **只读到新增的行**：别的标签页（或别的调用者）改、删了一条已读到的评论，这一页上那条评论要到重新打开详情（整体重读）才更新；描述的合并改了旧行的 `created_at`，所以它会被读到。Plane 的 `created_at__gt` 也只取新增；这一条写明并登记（4.17），I12 只断言新增。
- **页面**：动态的执行者为空时显示站点名（3.8）；执行者的显示名和主页链接从工作区成员列表（含已结束的成员关系）取；`IssueUser` 与 `IssueCreator` 两个相同的组件合并为一个（R3 第 5.4 节）。

### 3.14 描述的历史（决策点 5 负责人裁定 A）
- **只做描述的历史，不做工作项快照**（决策点 5，负责人裁定 A）：Plane CE 在运行时从不写 `issue_versions`（唯一的写入方 `issue_task` 没有调用者，`API/bgtasks/issue_version_sync.py:33-63`），页面也不读（`work_item_version.service.ts` 只有描述的历史）。总体设计 1.1"历史版本：工作项快照、描述修改历史"改为"描述修改历史（可查看、可还原）"，5.2 的表数 45 改为 44，6.6"同时写 `issue_versions`"删去（3.23）。
- **写入**：创建时一行；每次描述的修改（只补块 id 而内容相同的保存除外）一行，内容是修改之后的描述。**合并**：这个工作项最新的版本（按 `last_saved_at`）属于同一个人、`last_saved_at` 在 600 秒之内时，原地改它（描述、纯文本、`last_saved_at`、`updated_at`、`updated_by_id`），这是一个滑动的窗口（Plane 相同，`API/bgtasks/issue_description_version_task.py:15-40,56-66`）。
  - 写入在修改的同一个事务里（Plane 在 Celery 里）。
  - `owned_by_id`、`created_by_id` 是写这一版的人（Plane 的审计列抄工作项的，`API/db/models/issue.py:806-818`，与 `owned_by_id` 不一致）。
- **保留**：每个工作项只留最新的 20 行（按 `last_saved_at`），每天一次的定时任务 `issue.trim_description_versions`，每个未删除的项目一个事务（项目 N），物理删除多出的行（P9，与清理同一段，两者不能相遇，对抗性评审 M21）（Plane 相同，`API/bgtasks/cleanup_task.py:128-146`，按 `created_at`）。它只看未删除的工作项的未删除的版本（`deleted_at IS NULL` 的两边）；软删除的版本（随工作项删除）只归 60 天清理，两个任务碰不到同一行（3.5）。
- **读取**：`GET /issues/{issue_id}/description-versions`，最新的在前（`last_saved_at` 降序、`id` 降序），游标分页，每行不带描述；`GET /issue-description-versions/{version_id}` 带描述。规则 `issue_description_version.read`（管理员、成员，3.4）；工作项的访客规则同样适用。
- **还原**：没有专门的接口（Plane 也没有）。页面把旧的描述放进编辑器，经普通的 `PATCH` 保存：产生新的版本、一条描述动态。已归档的工作项不能还原（409，页面也禁用）。

### 3.15 草稿（总体设计 5.4）
- **表**：`draft_issues(id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at, deleted_at)`（4.13）。Plane 是一张与工作项同形的表加四张关联表（R1 第 14 节），差异清单已登记"一张表 + payload"。
- **`payload`** 是创建工作项的请求体，同一个结构 `IssueCreate`，但全部字段可选（`IssueDraftPayload`，5.2；标题可以为空串）。`project_id` 不在载荷里而是一列：创建的路径里有项目、请求体没有。
- **保存**（创建、修改）：`{project_id, payload}`。载荷只检查结构（`bodyshape`，未知字段 400），不检查它引用的状态、标签、负责人、父工作项（总体设计 5.4"保存时只检查类型"）。`project_id` 可为空；不为空时必须是这个工作区里调用者看得到的、未删除的项目，否则 422（`project_id`，`not_allowed`）。
- **只属于创建者**：列表、读、改、删、发布都只对自己的草稿；别人的草稿 404 `issue.draft_not_found`（Plane 的删除不看是不是自己的，管理员能删别人的，登记差异）。
- **列表**：`GET /workspaces/{slug}/drafts`，自己的草稿，最新的在前（`created_at` 降序、`id` 降序），游标分页，带 `total`。草稿的每个读（列表、取、改、删、发布）只认 `project_id` 为空或属于这个工作区未删除的项目的草稿（项目的目录经 `ProjectGate` 不加锁地读出）：项目已删除而删除的任务还没软删除它时，它与已删除的同样不存在（3.6）。
- **发布**：`POST /drafts/{draft_id}/publish`，没有请求体，用存着的载荷：
  - 没有项目 422（`project_id`，`required`）。
  - 载荷经 `createIssue` 的同一个结构检查：同一个不合规的请求体得到与直接创建相同的 400（M2 交接第 4 节）；再经同一组语义检查：同样的 422，路径相同（`state_id`、`label_ids[i]`、`parent_id` 等，引用的对象已删除时就是这样）。
  - 调用者要能在那个项目里创建（`issue.create`）。Plane 的发布不检查（`API/app/views/workspace/draft.py:224-234`），登记差异。
  - 与创建同一条代码路径（`app` 层的同一个函数），同一个事务里软删除草稿；M5 在这一个事务里移交附件（总体设计 5.4）。
  - Plane 用请求体而不是存着的草稿（页面把表单和草稿合并后发出，`issue-modal/form.tsx:270-293`），Nerve 照总体设计 5.4 用存着的载荷；页面先 `PATCH` 草稿、再发布（P15）。登记差异。
  - 响应 201，`IssueDetail`。
- **连带**：草稿随它的项目、工作区软删除（Plane 的 `project` 外键 `CASCADE`；P9 起由删除的任务写下请求的时刻，3.6）。Plane 的草稿另有指向父工作项的外键，父工作项删除时草稿被软删除；Nerve 的父工作项在载荷里，没有外键，发布时 422（`parent_id`）。登记差异。
- **页面**：草稿的入口只给工作区的管理员、成员（Plane 相同）；创建弹窗的"存为草稿"、草稿页的列表、编辑、删除、"移到项目"。草稿页没有筛选和显示设置（Plane 的 `store/issue/workspace-draft/issue.store.ts:212` 是 `const filterParams = {}`），不存任何设置（3.17）。

### 3.16 列表引擎（总体设计 3.3、3.4；架构问题 5、6）
本节是 P5 的规格；后端的接口约定（筛选的写法、分组的响应、游标的载荷）是后续 M 的列表照做的约定（M6 的迭代、模块列表，M7 的视图和工作区列表）。

**接口**
- `GET /projects/{project_id}/issues`（`listIssues`）：项目的工作项；`archived=true` 时只有已归档的（规则 `issue.archive`），否则只有未归档的（`issue.read`）。
- `GET /workspaces/{slug}/members/{user_id}/issues`（`listMemberIssues`）：个人主页；`relation` 必填：`assigned`（他是负责人）、`created`（他建的）、`subscribed`（他关注的）。范围是调用者在这个工作区里是有效成员的、未归档的项目（3.4 的"可读的集合"，Plane 相同，`API/app/views/workspace/user.py:139-147`），每个项目按它自己的访客规则过滤；规则 `member_issue.list`。`user_id` 不是这个工作区的成员（含已结束的成员关系）时 404 `workspace.member_not_found`（M3 的码）。
- 两者都不含分诊状态的工作项（状态目录不含分诊状态，M7 起才有这样的工作项；Plane 的 `issue_objects` 相同）。
- 两者都另有 `group_key`、`sub_group_key` 两个参数，只用于一组（一格）的"加载更多"（下面"响应"的最后一条）。

**筛选参数**（v0 3.3 的写法，补上三处：`none`、可重复的日期、单日）

| 参数 | 取值 | 含义 | 哪里 |
|---|---|---|---|
| `state_id` | 状态 id，逗号分隔 | 属于其中之一 | 项目 |
| `state_group` | `backlog`、`unstarted`、`started`、`completed`、`cancelled` | 状态在这些组里 | 两者 |
| `priority` | `urgent`、`high`、`medium`、`low`、`none` | 优先级是其中之一（`none` 是一个优先级，不是空） | 两者 |
| `label_id` | 标签 id 或 `none` | 有其中一个标签；`none`：没有标签 | 两者 |
| `assignee_id` | 账户 id、`me` 或 `none` | 负责人含其中一个；`none`：没有负责人 | 项目 |
| `created_by_id` | 账户 id 或 `me` | 创建者是其中之一 | 项目 |
| `mention_id` | 账户 id 或 `me` | 描述里提及了其中一个（`issue_mentions`） | 项目 |
| `subscriber_id` | 账户 id 或 `me` | 其中一个关注了它 | 项目 |
| `project_id` | 项目 id | 属于其中之一（个人主页按项目分组的"加载更多"） | 个人主页 |
| `start_date`、`target_date` | `YYYY-MM-DD`（那一天）、`a..b`、`..b`、`a..`、`none`；**可重复**（`?target_date=..2026-10-31&target_date=2026-10-01..`） | 每一个都要满足（AND）；范围含两端；空的日期不满足任何日期条件，只满足 `none`（没有这个日期）；`none` 不能与别的日期条件同时出现（422）。日历的"添加已有"列出没有截止日期的工作项要它（`calendar/quick-add-issue-actions.tsx:46-50` 的 `target_date: "none"`，对抗性评审 I5） | 两者 |
| `sub_issue` | `true`（默认）、`false` | `false`：只有没有父工作项的 | 两者 |
| `archived` | `true`、`false`（默认） | 见上 | 项目 |

- 逗号分隔的列表是"属于其中之一"，不同参数之间是 AND（v0 3.3）。`none` 可以与 id 并列（`label_id=none,<id>`：没有标签，或有这个标签）。
- 日期是不带时区的日期（工作项的日期本来就是），不涉及时区。Plane 的富筛选对 `created_at`、`updated_at` 的单日按用户的时区算（`API/utils/filters/filterset.py:171-189`），页面没有启用这两个筛选（R4 第 B.1 节），Nerve 不提供。
- 取值不合格（不是 id、不认识的词、日期写错、范围倒置）422，`errors[].field` 是参数名、`code` 是 `not_allowed`。同一个非日期参数出现两次 422。
- 迭代、模块的筛选（`cycle_id`、`module_id`）在 M6 随它们的表加入；在 M4 是未知的参数。
- Plane 的富筛选只生成"条件的 AND"，每个条件是等于、属于、范围（R4 第 B.1 节），上面的写法完整表达它；前端的 store 把筛选的表达式翻译成这些参数（3.22），接口不接收表达式。

**分组**
- `group_by`：`state_id`、`state_group`、`priority`、`label_id`（多值）、`assignee_id`（多值）、`created_by_id`、`target_date`（项目，日历用）；个人主页另有 `project_id`、没有 `state_id`、`assignee_id`、`created_by_id`（Plane 页面的选项，R4 第 E.1 节）。`sub_group_by`（只有项目的列表，看板的泳道）取同一组值，不能等于 `group_by`、不能单独出现，否则 422。键名用字段名（`created_by_id`、`label_id`），与筛选参数一致；Plane 是 `created_by`、`labels__id`，前端的映射表（`EIssueGroupByToServerOptions`）删除。
- 多值的分组：一个有两个标签的工作项出现在两个组里；没有标签的在 `none` 组（Plane 的键是 `"None"`）。行里的 `label_ids`、`assignee_ids` 总是工作项完整的集合，不受分组影响（Plane 按分组拆开的数组只有当前页里的那部分，R4 第 A.6 节，缺陷）。
- 组的键：id（状态、标签、账户、项目）、优先级、状态组、日期，空值是 `none`。
- 只返回非空的组；空的组由页面按自己的列补上（`show_empty_groups` 本来就只在页面里起作用，R4 第 A.8 节）。组的顺序：优先级 `urgent` → `none`，状态组 `backlog` → `cancelled`，日期升序，其余按键的文本升序；`none` 在最后。页面按自己的列顺序摆放，不依赖它。
- `limit` 在分组时是每组（子分组时是每格）的条数。

**响应**
- 不分组：`{data: Issue[], next_cursor, total}`。`total` 是满足筛选的工作项数，不受游标影响（v0 3.4 的列表封套加 `total`：单组"加载更多"的应答要给出这一组的总数，页面的组头和空状态读它，R4 第 E.3 节）。
- 分组：`{groups: [{key, total, data, next_cursor}], total}`；子分组：`{groups: [{key, total, subgroups: [{key, total, data, next_cursor}]}], subgroup_totals: [{key, total}], total}`。组的 `total` 是这一组里不重复的工作项数，格的 `total` 是这一格的；`subgroup_totals` 是每个子组（看板的一条泳道）里不重复的工作项数；最外层的 `total` 是全部不重复的工作项数。泳道的行头读 `subgroup_totals`，不再把这一行的格加起来：按多值的字段（标签、负责人）分组时，一个工作项在一行的两个格里，相加就多算（`swimlanes.tsx:171`、`base-issues.store.ts:399-407`，对抗性评审 M7）。Plane 的组总数把 0 报成 1，归档列表的总数恒错，分组时的 `total_count` 按"工作项 × 组"计（R4 第 A.5 节），都不照搬。
- 一组（一格）"加载更多"：**同一个请求**（同样的筛选、`group_by`、`sub_group_by`、`order_by`）加上 `group_key=<这一组的键>`（子分组的格再加 `sub_group_key`），带这一组（格）的 `next_cursor`。键与响应里给出的完全相同，含 `none`。回答是不分组的形状 `{data, next_cursor, total}`，只有这一组（格）的行，`total` 等于这一组（格）的 `total`。
  - 为什么不是"不分组的列表加上这一组的筛选"（v0 3.4 的写法）：多值的字段同时被筛选和分组时它表达不了。`label_id=A&group_by=label_id` 时组 C 的成员是"有 A **并且**有 C"，而"加载更多"的 `label_id=C` 得到的是有 C 的（不论有没有 A），`label_id=A&label_id=C` 又因非日期参数不能重复而是 422；`target_date` 的 `none` 组也不能接着读（日期的筛选没有 `none`）。Plane 的分组参数与筛选的表达式（`filters` 的 JSON）分开传，Nerve 把两者放进一个参数的空间，所以单列一个 `group_key`（独立评审 I2，11.6）。
  - `group_key` 不合格（不是这个分组码的键的写法）422；带 `group_key` 而没有 `group_by`，或带 `sub_group_key` 而没有 `group_key`、`sub_group_by`，都是 422。带 `group_by` 和 `cursor` 而没有 `group_key`（子分组时没有 `sub_group_key`）也是 422（`cursor`，`not_allowed`）：分组的第一页没有游标，每组的游标只能接着读那一组（对抗性评审 M6）。

**排序与游标**
- `order_by`：`sort_order`（只有升序）、`created_at`、`-created_at`（默认）、`updated_at`、`-updated_at`、`start_date`、`-start_date`、`target_date`、`-target_date`、`priority`、`-priority`、`state_name`、`-state_name`、`label_name`、`-label_name`、`assignee_name`、`-assignee_name`、`sub_issues_count`、`-sub_issues_count`、`link_count`、`-link_count`。不认识的 422（Plane 静默地换成 `-created_at`，R4 第 A.4 节）。`link_count` 的两个码随 `Issue.link_count` 在 P8 加入枚举（链接的数据在 P7，字段和排序放在 P8 的一个列表一侧的任务里，不与 P7 的出站客户端同一段评审，对抗性评审 M21；第 12 节约束 5：字段不早于它的数据），P5 时还没有；按附件数排序在 M5、按迭代、模块名在 M6 加入枚举。
- 键：
  - `priority` 的等级 `urgent` 4、`high` 3、`medium` 2、`low` 1、`none` 0；`priority` 升序（`none` 在前）、`-priority` 降序（`urgent` 在前），与 Plane 的页面的标签一致（`web/packages/constants/src/issue/common.ts:242-249`）。
  - `state_name`：状态名的比较键；`label_name`、`assignee_name`：工作项的标签（负责人）中比较键最小的那个，降序也取最小的（Plane 相同，`order_queryset.py:179-193`）；负责人用显示名（Plane 用 `first_name`，Nerve 的页面显示的是显示名，登记差异）。负责人的目录是这个项目（个人主页：可读的项目）里出现在负责人行上的每个账户，含已离开、已降为访客的，名字经 `identity` 的 `PublicProfiles` 读出（3.3）；页面从含已结束的成员关系的成员列表取名字，两边是同一份名字（对抗性评审 M22）。
  - 名字的比较键：只把 `A`–`Z` 转成小写，然后按码位比较。在 Go 里算出、作为目录传进 SQL，SQL 里用 `COLLATE "C"` 比较（UTF-8 的字节序就是码位序）；前端用同一个比较。JavaScript 的 `<` 和 `localeCompare` 都不行：`<` 比较 UTF-16 的码元，基本平面以外的字符（如 U+1F600）与 U+E000–U+FFFF 之间的字符（如 U+FF5E）顺序相反；前端逐个码位（`codePointAt`）比较。一份共用的样例文件 `api/testdata/issue-order.json`（名字、日期、空值、同键时的 `id`，含 U+FF5E 与 U+1F600 这样的一对）由 Go 的测试和 vitest 各读一次，两边的顺序必须与它相同。
  - 计数：未删除、未归档的直接子工作项中调用者看得到的个数（3.9）；未删除的链接数。
- **全序**：键按方向排，空值总在最后（两个方向都是，Plane 相同），同键时按 `id`，方向与键相同。前端在本地重排用同一个全序（3.22），否则本地插入的行与服务端的页对不上（R4 第 E.2 节第 8 条）。
- **游标的载荷**是一个 JSON 数组 `[排序码, 键, id]`，例如 `["-created_at", "2026-10-16T08:00:00.000000Z", "0199…"]`、`["target_date", null, "0199…"]`、`["priority", 3, "0199…"]`、`["label_name", "backend", "0199…"]`。载荷以 `id` 结尾（M2 交接第 2 节）、写明它的排序（`created_at` 与 `updated_at` 的载荷同形，不能互相冒用，R4 第 E.2 节第 8 条）；排序码与请求的 `order_by` 不同、形状不对，都是 400 `bad_request`（v0 3.4）。封套照旧（`shared/cursor.go`），不签名：改成另一个合格的位置只是从那里接着读。
  - 以名字、计数排序时，两页之间改了名字、加了子工作项，行会在页之间移动（Plane 相同）；游标只保证同一个全序上不重复地接着读。

**SQL 的写法**（S1 已在 sqlc v1.31.1、PG 18.6 上验证，0.3）
- **静态的 sqlc 查询**，不拼接 SQL（`TestModulesRunSQLOnlyThroughSQLC`，`server/internal/archtest/rawsql_test.go:19-26`：`gen` 之外的语句一律不许，所以没有"手写的查询"这条退路）：三条查询，不分组、分组、子分组；排序码、分组码是参数，筛选是数组参数（空数组表示不筛选）。
- **一个快照**（对抗性评审 M10）：列表的读在一个只读的 `REPEATABLE READ` 事务里（`platform/postgres` 的 `TxManager` 的选项 `ReadOnlySnapshot()`，6.1）：经端口读目录和跑列表的查询用同一个快照。不这样做时，两次读之间新建了一个状态、把一个工作项改到它，这个工作项在这一页上消失（状态目录是内连接），总数也对不上。
- **目录作为参数**（架构问题 5）：用例经端口读出状态（id、组、名字的比较键）、标签（id、比较键）、负责人（出现在负责人行上的账户的 id、显示名的比较键）、可读的项目（id、是否 `OwnOnly`），作为并列的数组传入。sqlc 不接受多个数组的 `unnest(a, b)`，写成 `ROWS FROM (unnest(@state_ids::uuid[]), unnest(@state_keys::text[])) AS st(id, key)`；它的列被定为 `interface{}`，只用来连接，要取出时在 `SELECT` 里转型。`issue` 的查询不读别的模块的表，sqlc 的守卫不开例外（3.3）。分诊状态不在目录里，连接时自然排除。
- **数组参数从不为 `nil`**：Go 的 `nil` 切片作为 `NULL` 传出，`cardinality(NULL) = 0` 为假，会滤掉全部行（S1 验证过）。仓储把每个数组参数规整成非 `nil` 的空切片，查询里另写 `coalesce(cardinality(@x), 0) = 0` 作为"不筛选"；两样都有测试。
- **形状**：
  1. `filtered AS MATERIALIZED`：满足筛选、可见性（`OwnOnly` 的项目只有调用者建的）的工作项，每个一行，只带排序、分组要的列；`MATERIALIZED` 让后面的几次引用不重算它。
  2. 窄的 CTE 算出排序键：每行 `(k_null, k_num, k_text)`：数值的键（时间的微秒数、日期的天数、排序值、优先级的等级、计数）在 `k_num`，名字的键在 `k_text`，空值时 `k_null` 为真。`ORDER BY k_null, CASE WHEN 升序 THEN k_num END, CASE WHEN 降序 THEN k_num END DESC, …, id` 和游标的谓词各只写一次，不为每个排序码写一份。
  3. 多值的分组：`LEFT JOIN LATERAL` 取工作项的标签（负责人），没有时一行空值，就是 `none` 组。日期的组键用 `to_char(target_date, 'YYYY-MM-DD')`，与其余的键同是文本。
  4. 第一页：`row_number() OVER (PARTITION BY 组键[, 子组键] ORDER BY 全序)`，每组取前 `limit + 1` 行（多出的一行只用来判断有没有下一页、给出 `next_cursor`，不返回）。
  5. 总数是每一行上的窗口列：每一级的总数在那一级恰好一个工作项一行的阶段用 `count(*) OVER (PARTITION BY …)` 算出：全部的总数在 `filtered` 上，组的总数在按组键展开之后、按子组键展开之前，子组（泳道）的总数在只按子组键展开的一级（每个工作项每个子组键一行），格的总数在两次展开之后；随行带出。仍是一条 SQL（v0 3.4）。`UNION ALL` 拼"行"和"总数"、`GROUPING SETS` 都能生成，但运行时 pgx 不能把 `NULL` 扫进非空类型的列（S1），不用。
  6. 一页为空时没有行带出总数：不带游标的请求就是 0；带游标（`next_cursor` 之后已没有行，或篡改过的游标）时，用例以同样的筛选去掉游标、`limit 1` 再查一次取 `total`。只在这种少见的情况多一次往返。
- **行的完整数组和计数**只为返回的行算（关联子查询 `ARRAY(SELECT label_id …)`、`count(*)`）；按计数、名字排序时，排序键要为全部满足筛选的行算出（Plane 干脆不支持，R4 第 E.2 节第 6 条）。
- **代价与规模的目标**：按 CTE 里的键 `CASE` 排序用不上索引，每个请求扫描、排序这个项目满足筛选的全部行（S1）。实测：1 万个工作项 2–100 ms；10 万个工作项，默认的一页约 55–125 ms，`label_name` 约 210–430 ms，分组约 0.2–0.7 s，泳道约 0.7–1.0 s。**v0 的目标**（决策点 7 的一部分，负责人已裁定）：一个项目至多约 2 万个未删除的工作项时，列表在这个实测的代价之内；更大的项目是 M8 的性能工作（13.2）。删除在这个规模上能否完成见决策点 7，不在 M4 加索引式的特化（例如为 `-created_at` 单列一条可用 `(project_id, created_at DESC, id DESC)` 的查询，S1 测到 0.3 ms，留给 M8 按数字决定）。
- **连接的参数由 nerve 自己设**：JIT 的编译（每次 150–500 ms）和预备语句的通用计划都让这几条查询更慢（S1）。`platform/postgres` 建连接池时在每个连接的运行参数里设 `jit = off`、`plan_cache_mode = force_custom_plan`（pgx 的 `RuntimeParams`），不交给部署；一个测试从池里的连接读回这两个值（P5）。M8 的部署交接只核对它们（13.2）。
- **索引**：`issues_project_id_state_id_idx` 等部分索引（4.2）服务筛选和单组的等值条件，不服务排序；P5 的 spec 按 S1 的计划记下实际用到的索引，没有用到的不建。
- **两个已知的代价写进 P5 的 spec**（独立评审 M26）：个人主页的跨项目列表按 `VisibleProjects` 的项目哈希连接、顺序扫描这个工作区的全部工作项，随工作区变大；通用计划和 JIT 的耗时（上一条已处理）。前者交给 M8 测量（13.2）。

**集合型的列表**（M3 设计 3.12、11.3 的延伸，3.23 同步总体设计 3.4）：子工作项、关联、链接、工作项的表情一次返回全部，`{data}`，不分页：它们的大小由人对一个工作项的操作决定，页面要整个集合。评论、动态、描述的历史、草稿、工作项的列表按游标分页。

**一处不同于 v0 3.4 的示例**：示例写 `group_by=state_id&sub_group_by=priority`，结构 `{groups: [{key, total, data, next_cursor}]}`；子分组时组里是 `subgroups` 而不是 `data`（上面）。3.23 同步。

### 3.17 项目的工作项显示设置（架构问题 10）
- **存在 `project_user_properties`**（M3 设计 3.18 留给 M4 的三列）：迁移 `00026_project_project_user_properties_issue_display.sql` 归 `project` 模块（M2 设计 3.14 的所有权规则），加三列 `jsonb NOT NULL`，各有默认值和 CHECK（4.15）：
  - `filters`：筛选，Nerve 的列表参数的格式：一个对象，键是 3.16 的筛选参数名（项目页可用的那些），值是字符串的数组，每个元素就是查询参数里逗号分隔的一项（日期是一个条件）：`{"state_group": ["started"], "assignee_id": ["me"], "label_id": ["none"], "target_date": ["2026-10-01..2026-10-31"]}`。默认 `{}`。页面从它直接拼出查询串，不再翻译。
  - `display_filters`：`{layout, group_by, sub_group_by, order_by, sub_issue, show_empty_groups, calendar: {layout, show_weekends}}`，键全有。默认取 Plane 模型的默认值去掉已砍的键（`type`、`calendar_date_range`），补上页面要的 `calendar`：`{"layout": "list", "group_by": null, "sub_group_by": null, "order_by": "-created_at", "sub_issue": true, "show_empty_groups": true, "calendar": {"layout": "month", "show_weekends": false}}`（`API/db/models/issue.py:61-70`；页面的默认值，`web/packages/utils/src/work-item/base.ts:126-143`）。
  - `display_properties`：显示哪些属性，M4 的 11 个键：`assignee`、`start_date`、`due_date`、`labels`、`key`、`priority`、`state`、`sub_issue_count`、`link`、`created_on`、`updated_on`，都是布尔值，默认全为真（Plane `API/db/models/issue.py:73-88` 去掉 `estimate`）。`attachment_count` 由 M5、`modules`、`cycle` 由 M6 随各自的迁移加键（改默认值和 CHECK，给已有的行补上键）。
  - Plane 的 `rich_filters`（筛选的表达式）不建：前端的 store 把表达式翻译成上面的 `filters`（3.22）。Plane 的旧 `filters` 格式（`state`、`labels`、`assignees`……）也不建。登记差异清单二·按表。
- **数据库的 CHECK 只查形状**（键的集合、值的类型，M2 设计 3.13），不查取值：取值由 `issue` 的列表词汇检查（下一条）。M6 给分组、排序加 `cycle_id`、`module_id` 一类的取值时不必改 CHECK。
- **列表的词汇在 `issue/domain`，`project` 经端口用它**（控制者的裁定，独立评审 M13）：`issue/domain/listvocab.go` 定义筛选参数名和每个参数的取值语法（id、`none`、`me`、优先级、状态组、日期的条件）、分组码和排序码，以及每种页面（项目、个人主页）允许哪些。`issue` 用它解析列表的查询参数；`project` 检查显示设置的 `PATCH` 时经它声明的端口 `IssueListSettings`（`Validate(filters, displayFilters, displayProperties) []FieldError`），由 `issue.Provide` 交出、就是这份词汇（纯函数，不要连接池，与 `IssueCascade` 放在一起，3.3）。两处用同一份规则：存下来的设置就是下一次的查询参数。
  - 为什么不放进 `shared`（第一稿的做法，M3 设计 11.4 的"两个模块必须一致的纯取值规则"）：这份词汇随 M5–M7 一直长（附件数、迭代、模块、视图），放进 `shared` 就让所有模块共享一份不断变大的工作项领域的词汇；端口的方向（`project` 声明、`issue` 实现）与 `IssueCascade` 相同，没有新的依赖方向（11.10）。
- **接口**：M3 的 `getProjectPreferences`、`updateProjectPreferences`（`api/modules/project.yaml:744-797`）的结构加三个字段（5.2）；`PATCH` 带上哪一个就整体替换哪一个（与 `navigation` 相同）。不合格 422（`filters.label_id[0]`、`display_filters.group_by` 等，`not_allowed`）。谁能读写：项目的任何有效成员，只有自己那一行（M3 照旧）。
- **只存在浏览器的**（Plane 相同，R4 第 C.8 节）：已归档的列表、个人主页的筛选和显示设置（`localStorage` 的 `issue_local_filters`；草稿页没有筛选和显示设置，3.15，对抗性评审 M11）；看板折叠的组（`kanban_filters`）。Plane 的已归档列表读写的键不一致（`archived/filter.store.ts:131` 读 `richFilters`、`:175` 写 `rich_filters`），筛选从不恢复；P13 改为同一个键。
- **M3 的关键词守卫的例外**：`/user-properties/` 两处 `until: "M4"`（`tools/keywords.json:2526`）在 `phase` 改为 `M4/P1` 的那一刻就到期（`tools/keywords.mjs:3-6,195`：`until` 不晚于 `phase` 即到期，守卫失败）。所以 P1a 开始时把 `phase` 改为 `M4/P1` 的**同一个提交**里，把它的 `until` 改为 `M4/P11`（P1b 的 `phase` 照旧是 `M4/P1`：守卫只认 `M<n>/P<k>`，第 12 节）：P10b 删除这两个旧方法和例外（项目页的筛选 store 改接新接口的 Phase，M3 交接第 5 节），`until` 取删除它的 Phase 的下一个，P10b 进行中它仍有效（7.9）。

### 3.18 搜索与按编号取（总体设计 6.9）
- **一个搜索接口**：`GET /workspaces/{slug}/issue-search`，参数 `q`（可选，去掉首尾空白之后至多 100 个字符）、`project_id`（可选，只搜这个项目）、`for`（可选：`parent`、`sub_issue`、`relation`）与 `issue_id`（`for` 出现时必填）、`start_date`、`target_date`（可选，3.16 的日期条件，含 `none`）、`limit`（1–100，默认 50）。响应 `{data: Issue[]}`，最好的在前，不分页（取前 N 个，同 Plane）。
  - **`q` 为空**（不带或去掉空白之后为空）：按 `updated_at` 降序、`id` 给出范围里最近更新的工作项，排除照旧。两个选择器一打开就用空的词搜索（`core/modals/existing-issues-list-modal.tsx:98-131`、`issues/parent-issues-list-modal.tsx:61-79`），只有命令面板等着输入（`power-k/ui/modal/search-menu.tsx:49`）；第二稿要求 `q` 必填，选择器每次打开都是 422，或者打开是空的（对抗性评审 I5）。
  - **日期的条件**：日历的"添加已有"只列出没有截止日期的（`target_date=none`，`calendar/quick-add-issue-actions.tsx:46-50`），解析与列表的日期参数同一份词汇（3.17）。
- **范围**：调用者是有效成员的、未归档的项目里，未删除、未归档的工作项，每个项目按它的访客规则过滤（3.4）。Plane 的命令面板的搜索不执行访客规则（泄露），选择器的搜索对一切访客只给自己建的（比别处更严）（R2 第 12 节）；Nerve 处处是同一条规则。`project_id` 不在范围里 404 `project.not_found`。
- **匹配与顺序**：
  1. `q` 形如"项目标识-编号"（标识经 3.7 的折叠比较）时，那一个工作项排在最前（总体设计 6.9"支持按编号搜索"）；
  2. `q` 全是数字时，编号等于它的工作项（Plane 对 `q` 里的每个整数都这样匹配，`API/utils/issue_search.py:14-24`），排在其次；
  3. 标题包含 `q`（不分大小写），按 `similarity(name, q)` 降序；三个字符以上的 `q` 用 `issues.name` 的 trigram 索引（4.2）。
  - 同分时按 `created_at` 降序、`id`。Plane 把 `PROJ-12` 当作"标题含 PROJ-12、或项目标识含它、或编号是 12"，得到每个项目的 12 号（R2 第 12 节）；Nerve 精确匹配。登记差异。
- **选择器的排除**（`for` 与 `issue_id`；`issue_id` 看不到时 404 `issue.not_found`）：
  - `parent`（给 `issue_id` 选父工作项）：只在它的项目里；排除它自己、它现在的父工作项、它的全部后代（Plane 只排除直接的子工作项，从页面就能造出环，R2 第 2 节）。
  - `sub_issue`（给 `issue_id` 加子工作项）：只在它的项目里；只列没有父工作项的（Plane 相同）；排除它自己和它的全部祖先（Plane 不排除祖先，也能造出环）。
  - `relation`（给 `issue_id` 加关联）：排除它自己和与它已有关联的（两个方向）；可以跨项目，`project_id` 不带时是整个工作区（页面的工作区开关）。
- **命令面板**：工作项的结果来自这个接口；项目的结果由页面在已加载的项目列表里过滤（M3 的 store）。迭代、模块、视图、收集箱的结果在 M6、M7 随各自的接口加回（13.2）。Plane 的全局搜索 `GET /workspaces/{slug}/search/` 删除（`WorkspaceService.searchWorkspace`）。
- **pg_trgm**：迁移 `00029_issue_issues_name_trgm.sql` 执行 `CREATE EXTENSION IF NOT EXISTS pg_trgm` 并建 `issues_name_trgm_idx ON issues USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL`（M0-P4 交接）。PG 13 起它是受信任的扩展，有 `CREATE` 权限的库属主就能建，不要超级用户；部署说明归 M8（13.2）。
  - S2 已验证的（0.3）：`00029` 要列在 `issue` 条目的 `schema:` 里，否则 sqlc 静默地把 `similarity()` 定为 `interface{}`（P8 加一个生成之后的类型检查）；SQL 里的命名参数不叫 `@q`（与生成的接收者同名，成为 `q_`），叫 `@query`，接口的参数仍是 `q`；三个字符以上走索引（10 万行 0.09–12 ms），两个字符退到顺序扫描（10 万行 45 ms）；运行时的角色经 PUBLIC 的 `EXECUTE` 调用扩展的函数，授权测试的预期不变。
  - 留给 P8 的：迁移角色在 `postgres:18.6` 里建扩展（第 6 个任务）；**中文**：常见的中文查询只有两个字，总走顺序扫描，P8 的第 12 个任务另测中文标题和两个字的查询；pg_trgm 取三元组的方式取决于库的 `LC_CTYPE`，部署的 locale 要求写进 M8 的交接（13.2）。
- **按编号取**：见 3.7。`/:workspaceSlug/browse/:workItem` 用它。

### 3.19 领域事件（决策点 3 负责人裁定 A；架构问题 4）
- **`shared` 中的值和端口**（`shared/event.go`，只依赖标准库）：
  - `Event{ID, Type, OccurredAt, WorkspaceID, ProjectID, ActorID *uuid.UUID, Entity EntityRef, Changes []Change}`：`ActorID` 为空表示 nerve 自己（决策点 4）；`Entity{Kind, ID}` 是事件说的那个东西（工作项、评论、链接、关联、表情）；`Changes` 与动态同形（`Field`、`Old`、`New`、`OldID`、`NewID`），创建时是创建的各字段，删除时为空。
  - `EventPublisher.Publish(ctx, Event) error`：用例在自己的事务里、写完之后调用。
  - `EventSubscriber{Name string; Types []string; Handle func(ctx, Event) error}`：模块经 `Subscribers()` 交出它的订阅者。
- **分发器在组合根**（`bootstrap/events.go`）：由各模块交出的订阅者建出，构造时检查：每个订阅的类型都是某个模块声明过的（`EventTypes()`），名字不重复。`Publish` 在调用者的上下文里（带着事务）按固定的顺序同步调用订阅者；一个订阅者返回错误，`Publish` 返回它，用例的事务回滚。没有订阅者的事件什么都不做。
  - 为什么同步、在事务里：订阅者典型的事是往 River 里投递任务（总体设计 6.3 第 3 条），`InsertTx` 与修改同一个事务提交，事务就是 outbox：改动提交了任务一定在，回滚了任务一定不在。
  - 不持久化事件本身，不建 outbox 表。M8 的 Webhook 订阅同一批事件、在同一个事务里投递投送的任务。
- **M4 的事件类型**（`issue` 模块发布工作项的；`project`、`workspace` 各发布一个删除的，P9）：

  | 类型 | 何时 | `Entity` | `Changes` |
  |---|---|---|---|
  | `project.deleted` | 删除项目（`deleteProject` 的事务里，M3 的连带之后） | 项目 | 空 |
  | `workspace.deleted` | 删除工作区（`deleteWorkspace` 的事务里，M3 的连带之后；它的项目不另发 `project.deleted`） | 工作区 | 这次锁住、删除的每个项目一项（`Field = project`，`OldID` 是项目的 id）；更早删除的项目不在其中 |
  | `issue.created` | 创建、快速添加、发布草稿 | 工作项 | 创建时的各字段（含负责人、标签） |
  | `issue.updated` | 修改、加子工作项（每个被改的子工作项一个）、只补块 id 之外的描述修改 | 工作项 | 每个改了的字段一项，负责人、标签的增减各一项 |
  | `issue.deleted` | 删除（子树里每个被删除的工作项一个，3.6） | 工作项 | 空 |
  | `issue.archived`、`issue.unarchived` | 手动归档、自动归档（`ActorID` 为空）、恢复 | 工作项 | `archived_at` |
  | `issue.comment_created`、`issue.comment_updated`、`issue.comment_deleted` | 评论的增改删 | 评论 | `comment_html` |
  | `issue.link_created`、`issue.link_updated`、`issue.link_deleted` | 链接的增改删 | 链接 | `url`、`title` |
  | `issue.relation_added`、`issue.relation_removed` | 关联的增删（一个关联一个事件，不分正反） | 关联 | `relation_type`、对方的 id |
  | `issue.reaction_added`、`issue.reaction_removed` | 工作项和评论的表情 | 表情 | `reaction`、它所在的工作项或评论 |

  - 不发布：只补块 id 的保存、抓取写回的元数据、草稿的保存（草稿不是工作项）、版本的保留和清理；删除项目、工作区之后由任务删除的工作项（`project.deleted`、`workspace.deleted` 说出它们），删除标签摘下的 `issue_labels`（3.6，这份清单写进 M8 的交接）。
- **M4 的两个订阅者**，都是 `issue` 模块的：
  - `crawl_on_link_change`：订阅 `issue.link_created` 和 `Changes` 里有 `url` 的 `issue.link_updated`，经端口 `LinkCrawls` 在同一个事务里投递 `issue.crawl_link`（3.11）。设计简报中的 `issue.link_url_changed` 不另设类型：一次修改一个事件，订阅者按 `Changes` 判断（"每次改动一个事件"的规则不例外）。
  - `delete_rows_on_deletion`（P9）：订阅 `project.deleted`、`workspace.deleted`，经端口在同一个事务里投递 `issue.delete_rows{scope, id, at, by}`（3.6）：`project.deleted` 投递一个 `{project, 项目}`；`workspace.deleted` 为 `Changes` 里的每个项目投递一个 `{project, 项目}`，另加一个 `{workspace_drafts, 工作区}`（没有项目的草稿）。`at`、`by` 取事件的 `OccurredAt`、`ActorID`，就是删除请求的时刻和删除者。第三稿按工作区投递一个任务，会写到更早删除的项目的行（聚焦复核 I1，3.6）。
- **P6 时分发器没有生产中的订阅者**：P6 建事件、分发器和全部发布点，订阅者在 P7 随链接到来（审查敏感的两件分开，第 12 节）。P6 的测试用测试组合里的订阅者证明发布点和回滚。这是写明的过渡，不是空的扩展点：P7 在同一个 M 里接上。
- **哪些连带走事件，哪些走端口**（决策点 3、7；M3 交接第 1 节）：要能拒绝（`project.state_in_use`）、要在发起者的锁之下执行的连带仍是同步的端口（`IssueCascade` 的 `StateInUse`、`DetachLabel`）：事件的订阅者不拒绝、不依赖调用顺序之外的锁。删除项目、工作区之后工作项一侧的行两样都不要：它们只需要在提交之后、以请求的时刻和删除者被删除，正是"跨模块的副作用经领域事件，订阅者投递任务"（总体设计 6.3 第 3 条）的形状。所以 M3 设计 13.2 设想的"把删除工作区的连带改挂到 `WorkspaceDeleted` 事件上"对工作项一侧做了：`project`、`workspace` 不再知道 `issue` 有哪些表（`IssueCascade` 的两个删除的方法在 P9 删除），M5、M6 的表（附件、迭代的工作项）在同一个事件上加自己的订阅者，不必再扩一个端口；M8 的 Webhook 订阅同一个事件（13.2）。
- **决策点 3 的另一个选项**（B：不建事件，M4 直接在用例里投递任务）的代价写在第 10 节：M7 的通知、M8 的 Webhook 要回头改 M4 的每个用例；选 B 时删除项目、工作区经 `IssueCascade` 的一个投递任务的方法（`ScheduleRowDeletion(projectIDs, workspaceDrafts, at, by)`）在请求的事务里投递同样的任务（每个删除的项目一个，删除工作区另加没有项目的草稿一个），3.6 的其余不变。

### 3.20 任务平台与测试的时钟（M2 设计 3.15；M2 交接第 1、6、13 节；M3 交接第 1 节；架构问题 4、7）
- **事务内投递经只投递的客户端**（S4，0.3；独立评审 M25）：`platform/jobs` 交出 `NewInserter(pool) (*Inserter, error)`：一个不配 `Workers`、不调用 `Start` 的 River 客户端（M2 交接第 1 节说的那一个），和它的 `InsertTx(ctx, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error`。只投递的客户端不检查种类是否登记（S4），所以它在全部模块之前由连接池建出，模块的投递端口拿到它时不需要任何 worker 已经登记，组合里没有"先建客户端、后登记"的环（第一稿要把 `jobs.New` 拆成两步，6.6）。工作的客户端（`jobs.New`，配队列、worker、定时任务，`Start`）照 M2 的写法在最后建出，不交给模块。
  - `platform/postgres` 交出 `Tx(ctx) (pgx.Tx, bool)`，读出 `TxManager` 放在上下文里的事务。`platform/jobs` 不导入 `platform/postgres`（`archtest/rules_test.go:43`：平台包之间不互相导入，`config` 除外；今天事务只能经 `postgres.DB(ctx, pool)` 取出，`platform/postgres/tx.go:26-31`）；`bootstrap` 把两者接成模块要的投递端口：不在事务里调用就是程序错误，返回错误。
  - **投递的种类都有 worker，启动时检查**：只投递的客户端什么种类都收，漏登记的种类会安静地留在 `river_job` 里没人做。每个模块经 `EnqueuedKinds()` 声明它可能投递的种类（M4：`issue.crawl_link`；P9 起加 `issue.delete_rows`），`bootstrap` 在建工作的客户端时核对每一个都有登记的 worker，缺了就构造失败（与"定时任务没有 worker 时失败"同一处，下面），有一个变异测试。
- **队列与并发**：两个队列：`default`（事件触发的任务：抓取，P9 起删除工作项一侧的行）、`periodic`（定时任务）。配置 `jobs.default_workers`（默认 10）、`jobs.periodic_workers`（默认 2）；今天默认队列写死 `maxWorkers = 2`（`platform/jobs/jobs.go:38`）。抓取一次最多等 5 秒（3.11），10 个 worker 是每秒约 2 个链接的下限，足够一个团队；删除工作项一侧的任务每个项目一个，按键集分批（SP4：每 1000 行约 3 ms，一个目标规模的项目几秒），每次运行至多 45 秒的预算、用完就 snooze（3.6），期间只占一个 worker；删除一个五个项目的工作区是五个任务，可以分在五个 worker 上。不另设队列。River 默认的 1 分钟期限只有这个 worker 覆盖（3.6），抓取有它自己的 5 秒。
- **构造时的检查**：定时任务的种类、模块声明会投递的种类没有对应的 worker 时，`jobs.New` 拒绝启动（River 要到运行时才报错，只投递的客户端根本不报）。
- **测试的余量**（M3 交接第 1 节）：M3 收尾把 `platform/jobs` 的测试中时间的余量改成了期限，留下两处：`jobs_test.go:265`（`TestStartWithoutADatabase`）、`runner_test.go:136`（`TestStopEndsTheAttemptsToStart`）。P6 改 `platform/jobs` 时照样改成期限：分得开"立即返回"和"等过了重试的间隔"（`firstRetry`，1 秒）。
- **命令行的只投递客户端**：只投递的客户端从 P6 起就有了（上面），但 M4 没有投递任务的命令，命令行的组合仍不需要它；M2 交接第 1 节原样转交 M5，附一句"客户端已在 `platform/jobs`，第一个投递任务的命令加入时交给命令行的组合"（13.2）。
- **M4 的定时任务**（间隔都可配置，测试配置为 2 秒）：

  | 种类 | 队列 | 间隔（默认） | 做什么 |
  |---|---|---|---|
  | `identity.cleanup_expired_sessions` | `periodic` | M2 照旧 | — |
  | `issue.auto_archive` | `periodic` | `issue.auto_archive_interval`，`24h` | 3.8 |
  | `issue.trim_description_versions` | `periodic` | `issue.trim_versions_interval`，`24h` | 3.14（P9） |
  | `cleanup.purge_deleted` | `periodic` | `cleanup.purge_interval`，`24h` | 3.21（P9） |

  - Plane 在每天固定的时刻运行（01:00、03:15 UTC，`API/celery.py`）；Nerve 按间隔，从启动算起（River 的 `PeriodicInterval`）。差别不对用户可见，登记差异。
  - 只有 River 的主节点运行定时任务（M2 设计 3.15）；v0 是单个进程。
- **测试的时钟**（架构问题 7）：
  - 配置 `clock.offset`（时长，默认 0），只在 `env = test` 时接受，别的环境不为 0 时启动失败（与 M3 的测试专用配置同一种做法）。组合根用 `clock.Offset{Base: clock.System{}, Offset: d}` 代替今天写死的 `clock.System{}`（`bootstrap/app.go:112,118,124,135,167`），全部用例的时钟一起偏移。River 自己的调度用真实的时间（不受影响）：定时任务按真实的间隔运行，任务里读到的"现在"是偏移之后的。
  - e2e 的 `fixtures/clock.ts`：时间流逝的故事（I16、I21）先从模板建一个新库，在它上面用正常时钟起一个 nerve 准备数据，停下；再以 `NERVE_CLOCK__OFFSET` 起一个 nerve（`nerveWith` 的同一种启动，`e2e/fixtures/test.ts:70`），页面和断言都对着它。同一个库上任何时刻只有一个 nerve，River 的选主不会落到时钟不同的进程上；worker 自己的 nerve 在另一个库上，不受影响。
  - `clock.ts` 的几处细节（对抗性评审 M20）：它自己建库、建连接、读写这个库，不能复用 `db`、`nerveWith`、`databaseSnapshot`（它们都绑在 worker 的库上）；测试环境的签名密钥是每次启动临时生成的（`auth.jwt.private_key_file` 为空），`session_ttl` 是 720 小时，短于 31 天，所以页面版本在偏移的 nerve 上重新登录（接口版本用 PAT，PAT 默认不过期）；断言"定时任务完成"要找一条在偏移的 nerve 起来之后才完成的任务（准备数据的 nerve 每 2 秒也跑过同一个种类）；对照的行（"没有更新够久的不变""60 天之内的仍在"）要经偏移的 nerve 写下，在准备数据的 nerve 上写下的一切都是同一个时刻（或用 `archive_in = 2` 这样的设定拉开）。
  - 为什么不在运行中改时钟：那要一个改时钟的接口或信号，是只为测试存在的生产代码路径；以启动参数给出，生产代码里只有一个装饰器。

### 3.21 60 天的物理清理（总体设计 5.5、6.7；M2 交接第 5 节；架构问题 9）
- **每个模块清理自己的表**：`identity`、`workspace`、`project`、`issue` 各交出一个 `Purger`：按它的表从子到父，删除 `deleted_at < cutoff` 的行，每张表一个按 `id` 的键集循环，每批一个事务、一条语句：`DELETE FROM t WHERE id IN (SELECT id FROM t WHERE deleted_at < $1 AND id > $after ORDER BY id LIMIT $2) RETURNING id`，下一批的 `$after` 是这一批最大的 `id`，直到删除 0 行。第三稿的语句没有 `id > $after`，与删除的任务同一个毛病：SP4 量到 80 万行 31.4 秒、每批 8 → 113 ms（聚焦复核 I2）。描述版本的保留是每个项目一条语句（3.14），不分批，不受影响。模块的 sqlc 查询只碰自己的表（`TestSQLCSchemaScope`）。
- **一个定时任务**：`bootstrap` 把四个清理者组成一个任务 `cleanup.purge_deleted`，按 `issue` → `project` → `workspace` → `identity` 的顺序运行（子模块先清理，父表的 `CASCADE` 多半已无事可做）。一个任务而不是四个：四个各自运行时，清理项目的批量与清理工作项的批量可以在同一批行上按不同的顺序相遇（外键的 `CASCADE` 在父表的删除里删除子行），一个任务按顺序执行就没有并发。
- **时刻**：`cutoff = 现在 - cleanup.purge_after`（默认 `1440h`，即 60 天，Plane 的 `HARD_DELETE_AFTER_DAYS`）；"现在"取用例的时钟（测试里偏移，3.20）。每批 `cleanup.purge_batch` 行（默认 1000）。
- **与写不相遇**：没有任何写会改一行已软删除的行（每个写都带 `deleted_at IS NULL`），清理删除的行不会有人在写。一个活的行也不会引用一个已软删除 60 天的行：引用都随删除连带软删除（3.6），或在删除时被拒绝（状态在用 409）。唯一的例外是删除项目、工作区之后还没做完的任务（3.6）：那些行仍是活的，清理不碰它们；任务挂起超过 60 天时，清理项目（工作区）的行，外键的 `CASCADE` 把它们一起物理删除，任务随后写 0 行。
- **外键的图**（4.16）：M4 的每个子表经 `CASCADE` 指向它的父表，都有不带条件的索引（M3 设计第 4 节的补充规则）；`issue_activities.issue_id` 是 `CASCADE`（不是 Plane 的 `DO_NOTHING`），`issue_comment_id` 不建（M2 交接第 5 节的阻碍由此消除）。`bootstrap` 的目录测试（`workspace_deletion_catalog_test.go` 的同一种写法）核对：每张有 `deleted_at` 的表都有清理者，每条指向被清理的表的外键都是 `CASCADE` 或 `SET NULL`。
- **索引**：每张被清理的表一个 `<表>_deleted_at_idx ON (deleted_at) WHERE deleted_at IS NOT NULL`，迁移按模块（P9：`00030_identity_purge_indexes.sql`、`00031_workspace_purge_indexes.sql`、`00032_project_purge_indexes.sql`、`00033_issue_purge_indexes.sql`）。
- **不清理的**：账户（v0 不物理删除，M2 设计 4.7）；过期的会话（M2 的任务另行物理删除）；未删除的描述版本超过 20 的（3.14 的任务，不等 60 天；软删除的版本只归清理，两个任务不碰同一行，3.5）。
- **e2e**：I21（时钟偏移 61 天）。

### 3.22 前端的列表：在 `reconciled.ts` 上分页、分组（总体设计 7.7；M3 交接第 6 节）
总体设计 7.7 要求工作项的分页、分组列表扩展 `core/lib/reconciled.ts`，不另写一份。它今天的 `Reconciled<V>` 是"整个值由最新的一次取数写下，取数在外期间确认的修改在应答上重做"（`reconciled.ts:26-81`，重做在 `:56-76`）。列表引擎要多四样：按组接着取的页、确认的修改在组之间移动工作项、组的总数、一个工作项只有一份。第二稿的规则在重做、窗口的移动和那一份工作项上会算错（对抗性评审 I2 的九个反例：双减、判断不了版本、重做时的"修改之前"无定义、窗口后退留下看不见的卡片、旧的应答覆盖那一份、读到别人的修改之后一张卡片在两列、列表的行覆盖详情的描述、空的组没有地方插入、放在最后或日历的满格时卡片消失）。下面的数据模型照评审的改法重写，九个反例各是 vitest 表里的一行；仍是一份实现，扩展 `reconciled.ts`（总体设计 7.7）。

- **`Reconciled.fetchPart(part, read, merge)`**（加在 `reconciled.ts`，约 40 行）：取值的一部分（一组、一格的下一页）。
  - 值还没取到时什么都不做。
  - 它在外期间开始了一次整体的 `fetch`，或同一个 `part` 开始了更新的一次 `fetchPart`，它的应答不写（与 `fetch` 被超越的规则相同），返回 `undefined`。
  - 应答到达时，`merge(当前的值, 应答)` 得到新的值（被回答的那一组的 `total` 取应答的 `total`，不在本地已加减过的数上接着算），再把它在外期间确认的修改按确认的顺序在新的值上重做一次；重做时修改拿到这一次的应答（下面"重做"一条按它判断）。
  - 整体的 `fetch` 照旧：它的应答写下全部的组（第一页、总数、游标），在它开始之前发出、仍在外的 `fetchPart` 随之作废。
- **列表的值**（`PagedGroups`，`core/lib/paged-groups.ts`，纯函数，不依赖 MobX）：`{total, groups: Map<键, Group>, subgroupTotals?: Map<子组键, 数>}`，`Group = {total, rows: Row[], nextCursor, boundary, stale, subgroups?: Map<子组键, Group>}`，`Row = {id, key, updatedAt}`。
  - `rows` 是那一组**按应答写下的**行：id、它在这个列表的全序里的排序键、应答给出的它的 `updated_at`。版本的判断看它，不看那一份工作项（第二稿只存 id，那一份早已写着修改之后的版本，判断无从做起）。
  - `boundary` 是这一组回答过的页的最后一行的（排序键，id），**固定下来，不随活的行重算**（第二稿的"已载入的最后一行之后"会随那一行被拖走而后退，后来放进去的卡片既不显示、下一页也不返回它）。它只往后移：并入一页时取 `max(boundary, 这一页的最后一行)`（按全序），所以不会退到拖放扩出去的卡片之前（聚焦复核 M3）。`nextCursor` 为空的组是"载入完"的。
  - **在窗口内** = 组载入完，或按全序 `(键, id) ≤ boundary`。
  - **没有回答的组、格是空的、载入完的**：服务端只返回非空的组（3.16），所以不在应答里的组就是 `{total: 0, rows: [], nextCursor: null}`，修改需要时按这个样子建出它（拖进空列、空泳道的第一张卡片）。
  - 不分组的列表是一个键为空的组；泳道的总数是 `subgroupTotals`（3.16 的 `subgroup_totals`），不再把一行的格加起来。
- **一个工作项只有一份**（按 id 的 `ReconciledByKey`，详情的读取和列表的应答都写它）：
  - 两个版本取**服务端的 `updated_at` 大的**，相等时取后应答的。第二稿只比 `answeredAt`，那是写入的顺序，不是服务端的顺序：在一次修改之前读、在它之后才回答的看板的重取会把修改之前的行写进那一份。Plane 的 `addIssue` 用任何应答覆盖（`store/issue/issue.store.ts:61-72`）。
  - `description_html` 只由 `IssueDetail` 的应答写；列表的行（`Issue`，没有描述）只改别的字段，留着那一份的描述（详情、浮层的编辑器读它）。
  - 一次**读**（取单个、任何一份列表的应答）写进那一份的新版本若改了决定列表成员关系的字段（状态、优先级、标签、负责人、起止日期、父工作项、`archived_at`、`created_by_id`，或某份列表的排序键用到的字段），那是别人的修改，这个会话的列表的总数无从确知：每份含有它的列表里，它旧、新两个版本所属的组标 `stale`。这个会话自己有一个修改在路上（已发出、还没确认）的工作项除外：读到的新版本可能就是这个修改，它确认时按下面的规则摆放，不当作别人的修改，不白白重取（聚焦复核 M3）。
- **确认的修改**：一个修改在确认的那一刻记下 `Change = {id, beforeSnapshot, after, drop?}`：`beforeSnapshot` 是确认之前那一份里的它（新建时为"无"，这个会话从没取到过它时为"不知道"）；`after` 是确认的应答（删除时为"无"；归档、恢复是 `archived_at` 变了，列表的 `archived` 决定它属不属于）；`drop` 是用户拖放的落点（哪一组、放在哪两张卡片之间），只有拖放才有。
  - 对每一份列表的每一组（格）`g`：`新属 = after 满足列表的筛选、它的组键含 g`（在本地判断：筛选、分组的键、可见性都在工作项的行上）；`旧属` 先看这一份列表自己：`g.rows` 里有它而 `updatedAt` 早于 `after` 的，它在 `g` 里（是）；没有它而 `g` 载入完的，它不在（否）；`g.rows` 里有它而 `updatedAt` 不早于 `after` 的，这一份已经含着修改，只摆放；其余按 `beforeSnapshot` 算（"不知道"时这一组标 `stale`）。
  - 按 `(旧属, 新属)`：

    | 旧属 | 新属 | `g.total` | `g.rows` |
    |---|---|---|---|
    | 是 | 是 | 不变 | 在 `rows` 里：换成新的键和版本、按全序重排，不在窗口内就移出（它在后面的页里）；不在而在窗口内：插入 |
    | 是 | 否 | 减一 | 在就移出 |
    | 否 | 是 | 加一 | 在窗口内就插入；否则不动（它随后面的页来） |
    | 否 | 否 | 不变 | — |

  - **用户的拖放留在放下的地方**：带 `drop` 的修改在目标组里总是把卡片插入到落点，落点在窗口之外时把这一组的 `boundary` 扩到这张卡片。放在一列的最后（`sort_order` 是最后一张加 65535，`issue-layouts/utils.tsx:479-514`）、放进日历上已满的一天（月视图一天只载入 4 张，`calendar/base-calendar-root.tsx:81-90`）都是这样；之后的页按服务端的游标接着取，按全序、按 id 去重并入，落在它前面的行排到它前面。卡片不会在确认之后消失、只剩计数变了（I6 的"卡片出现在 Done"）。
  - 最外层的 `total`、泳道的 `subgroupTotals` 同样按"修改前在不在、修改后在不在"加减。
  - 看板上最常见的一步（I6、总体设计 8.2：把卡片拖进一个没有载入完的列）因此是确知的：不重取整个看板，已载入的页和滚动的位置都留着。
- **重做：在一次取数的应答上**（取数在外期间确认的修改，`Reconciled` 在它的应答上重做）：应答可能是修改之前读的，也可能已经含着它。**只按应答自己判断**它在修改之前还是之后：
  1. 应答的任何一组里有这个工作项的行：`updatedAt` 不早于 `after` 是之后，早于是之前。
  2. 应答里没有它的行，但应答覆盖了它修改之后该在的位置（它新属于的某组 `g`，应答对 `g` 是载入完的整组，或 `after` 的键落在应答对 `g` 回答的那一段里）：是之前（之后的话它就在那里）。应答覆盖了它修改之前所在的位置而它不在：是之后。
  3. 都看不出：不知道。
  - **重做时旧属只来自应答和 `beforeSnapshot`**，不来自应答之前已并入的行（例如拖放已插入的、`updatedAt = after` 的那一行；把它当作"这一份已经含着修改"会让一个在修改之前读的应答的总数少算一个，聚焦复核 M3）。应答回答了的组，旧属按应答：应答里有它的行是"是"，应答对这一组载入完、或覆盖了它修改之前的键而没有它是"否"；应答没有回答的组按 `beforeSnapshot`。
  - 之后：每组只按全序摆放（和拖放的落点），不加减。之前：照上表加减，旧属按上一条。不知道：`旧属 ≠ 新属` 的组标 `stale`，其余只摆放。
  - **第 2 条判断时另取一次这个工作项**（`getIssue(X)`）：第 2 条把"应答里没有它"读成"应答在修改之前"，这对这个会话自己的修改成立，但若第三方在 `after` 之后又改了它（它已不在新属的位置），第 2 条会把它放回服务端说它不在的地方。取到的版本经"一个工作项只有一份"的规则写进那一份：比 `after` 新、改了成员关系的，按那里的规则标 `stale`，第三方的移动由此被发现（聚焦复核 M3）。
  - 一组的"加载更多"的应答只回答那一组，只在那一组上判断。
- **只在这几种情况下标 `stale`，而且只标受影响的组**：`beforeSnapshot` 是"不知道"；列表的筛选有本地判断不了的参数（`mention_id`、`subscriber_id`、个人主页的 `relation=subscribed`：提及、关注的集合不在工作项的行上，这样的列表对任何一个确认的修改都标 `stale`）；重做时看不出应答在修改之前还是之后；读到别人改了成员关系（上面"一个工作项只有一份"）。
- **`stale` 的组**在下一次显示时（已显示的立即）重取：一组的用 `fetchPart` 取那一组的第一页（`group_key`，3.16），整个列表的（筛选判断不了时）用 `fetch`；重取作废在外的同一部分。总数因此不会偏离：要么确知地加减，要么重取。
- **vitest 的表**（9.5）：每一行是 `(列表当前的值, 修改或读到的版本, 应答?) → 期望的值`：上表四种组合各配在、不在 `rows`，窗口内、外，载入完与否；新建、删除、归档和恢复；多值的组（一个标签换成另一个、两个标签去掉一个）；子分组的格和泳道的总数；最外层的总数；拖放（窗口外、空的组、空的泳道格）；重做的三种判断；每种 `stale`；对抗性评审 I2 的九个反例各一行；聚焦复核 M3 的五行：重做时旧属不取应答之前并入的行（反例一的变体：拖放先插入、看板的重取在修改之前读，那一组的总数是应答的加一）、`fetchPart` 的并入取应答的 `total`、第 2 条判断之后第三方又移动了它（`getIssue` 的版本让那一组 `stale`）、拖放扩出 `boundary` 之后并入一页 `boundary` 不后退、自己的修改在路上时读到的新版本不标 `stale`。
- **全序**：`paged-groups.ts` 的比较与服务端相同（3.16）：名字的比较键逐个码位比较（`codePointAt`，不用 `<` 和 `localeCompare`）、优先级的等级、空值在最后、同键按 `id`。vitest 读同一份样例 `api/testdata/issue-order.json`。
- **每份列表都改**：一个确认的修改作用于这个会话里全部的列表（项目的、已归档的、个人主页的）和工作项的那一份（总体设计 7.7"写进每一份"）。Plane 的乐观更新只改当前的列表、失败时回滚（`base-issues.store.ts:538-572`），与 7.7 的"只用确认的结果、一次一个"相反，删除。拖动（3.16 的 `sort_order`、跨组改字段）也是先发修改、确认之后才移动卡片：页面在请求在路上时让卡片停在放下的位置、显示更新中的遮罩（`list/block.tsx`、`calendar/issue-block.tsx` 已有），被拒绝时回到原处并提示。
  - **`sort_order` 在轮到它发出时才算**（总体设计 7.7）：新的值取放下位置前后两张卡片的 `sort_order` 的中点，而这两张的值来自 store；同一个工作项（或同一列）的修改经 `oneAtATime()` 排队，排在前面的修改确认之后可能移动了相邻的卡片。所以拖放只记下"放在哪两张之间"（它们的 id），请求的体在这次修改轮到发出时按那时的列表算出；vitest 覆盖"前一个拖动确认之后，后一个按新的位置算"。
- **一组的"加载更多"**：`fetchPart(组的键, 同一个请求 + group_key[ + sub_group_key] + 这一组的 next_cursor, 追加)`（3.16）；回答是这一组的 `{data, next_cursor, total}`，并入时这一组的 `total` 取它的 `total`，`boundary` 取 `max(boundary, 它的最后一行)`。Plane 在本地按已加载的条数算偏移量的游标（`base-issues.store.ts:420-430`），从不用服务端的 `next_cursor`（R4 第 C.4 节），删除。
- **筛选的表达式翻译成参数**：富筛选的 store（`@nerve/shared-state` 的 `work-item-filters`）照旧维护 `TWorkItemFilterExpression`（一层 AND，R4 第 B.1 节），项目的列表 store 把它翻译成 3.17 的 `filters` 对象再拼查询串；翻译是一个纯函数，vitest 覆盖每种条件。
- **子工作项的筛选留在浏览器里**（对抗性评审 M11）：子工作项的列表是集合型、不分页（3.16），它的筛选在浏览器里做（`sub_issues_filter.store.ts:131` 经 `base-issues-utils.ts:227-242` 的 `getFilteredWorkItems`），用到相对日期的选项（`fromnow`，`packages/constants/src/filter.ts:11-43`）。P10b 重写 `base-issues-utils.ts` 时保留 `getFilteredWorkItems` 和这些选项。
- **否定的筛选删除**（M1-P3 交接）：Nerve 的接口没有"不是""不属于"，页面也生成不了（`useFiltersOperatorConfigs` 恒为 `allowNegative: false`，R4 第 B.2 节）。删除 `TAllAvailableOperatorsForDisplay`、`TAllAvailableDateFilterOperatorsForDisplay`（恒等的别名）、`TNegativeOperatorConfig`、`isNegation` 一路的参数和 `useFiltersOperatorConfigs`，调用方直接用 `TSupportedOperators`。
- **规模**：`reconciled.ts` 增约 40 行，`paged-groups.ts` 约 450 行（第二稿估 300 行；多了行的版本、固定的边界、拖放的落点和重做的判断），取代 `base-issues.store.ts`（1,775 行）中分页、分组、排序、乐观更新的部分（7.3）。

### 3.23 需要同步到上级文档的地方
**规则**照 M3 设计 3.20：每处偏离，在实现它的那个 Phase 的同一次合并中同步到上级文档和差异清单；建表、改表的 Phase 逐列更新差异清单。收尾只按本表逐行核对。总体设计 9.4 中 M4 的状态随本文的合并改为"进行中"（2026-10-17，已改）。负责人已裁定第 10 节、批准第 11 节（2026-10-17，全部按建议），下表每一行的内容都已确定，按标出的 Phase 同步。

| Phase | 文档 | 位置 | 内容 |
|---|---|---|---|
| P1a | 总体设计 | 4.2 | 加锁顺序延伸到 M4 的表；工作项的写最先锁项目行、不锁工作区行，取锁之前做不加锁的预检（凡拒绝的都在那里答）；删除工作区先锁它的项目再读时刻；写明 B 的两个代价（3.5，决策点 1） |
| P1b | 总体设计 | 4.2 | 取连接之前进入按项目的写闸门、`lock_timeout` 的兜底，满时 503 `server_busy`；写明连接池的代价（3.5，架构问题 11） |
| P1b | 总体设计 | 6.2 | 平台包 `platform/gate`（带 `context` 的按键的信号量，3.5，架构问题 11） |
| P1a | 总体设计 | 6.2 | 模块 `issue` 和它的 13 张表（3.3，架构问题 1） |
| P1a | M3 设计 | 3.6 的取锁表、17.4 | "删除工作区"一行改为先锁项目、再读时刻；17.4"M4 的工作项写入……不默认沿用"后补一句 M4 的选择（3.5） |
| P1a | M3 设计 | 3.17 | 删除状态：有工作项（含已归档的）用它时 409 `project.state_in_use`（3.6） |
| P1a | 差异清单 | 二·按表 | `issues`、`issue_assignees`、`issue_labels`、`issue_subscribers`、`issue_activities` 逐列（4.2–4.5、4.11） |
| P1a | 差异清单 | 四 | 4.17 中标 P1a 的行 |
| P1b | 差异清单 | 四 | 4.17 中标 P1b 的行 |
| P2 | 总体设计 | 6.5 | `AllowCreator` 要求创建者仍是项目的有效成员、角色不低于成员；`OwnOnly` 与"一条访客规则"（3.4，架构问题 3） |
| P2 | M2 设计 | 3.11 | 格式检查器登记 `date`；字段错误的路径按数组下标的数值排序；不限类型的节点检查数的范围（6.1，M2 交接第 4 节） |
| P2 | 差异清单 | 四 | 4.17 中标 P2 的行（访客规则、创建者、归档只读、按编号取的 404 和折叠、起止日期） |
| P3 | 总体设计 | 3.4 | 集合型的列表加上子工作项、关联、链接、工作项的表情（3.16 末尾）。P3 有第一个（`listSubIssues`） |
| P3 | 总体设计 | 3.6 | "关联对象只返回 ID"的第二个例外：关联内嵌对方工作项的列表行（3.10） |
| P3 | 总体设计 | 4.2 | 约定七（跨项目的写按 `id` 升序锁住涉及的项目）和关联行的约定五例外（3.5） |
| P3 | 差异清单 | 二·按表、四 | `issue_relations` 逐列（4.6）；4.17 中标 P3 的行 |
| P4 | 总体设计 | 1.1 | "评论（支持回复）"改为"评论"（决策点 6）；"历史版本：工作项快照、描述修改历史"改为"描述修改历史（可查看、可还原）"（决策点 5） |
| P4 | 总体设计 | 5.2、5.3、6.6、6.7 | 表数 45 改为 44，删去 `issue_versions`；6.6 删去"同时写 `issue_versions`"；6.7 的保留任务写明"每个工作项最新的 20 个描述版本"（3.14） |
| P4 | M2 设计 | 3.12 | 页大小的规则移到 `shared`（M3 交接第 9 节） |
| P4 | 差异清单 | 一、二·按表、四 | 一：`issue_versions` 不保留；二：`issue_comments`、`issue_reactions`、`comment_reactions`、`issue_mentions`、`issue_description_versions` 逐列（4.7、4.9、4.10、4.12、4.14）；四：4.17 中标 P4 的行 |
| P5 | 总体设计 | 3.3 | 筛选的写法补上 `none`（含日期的 `none`）、可重复的日期参数、单日、不合格的取值 422（3.16） |
| P5 | 总体设计 | 3.4 | 不分组的列表带 `total`；子分组的结构 `subgroups` 和泳道的总数 `subgroup_totals`；只返回非空的组；分组时不带 `group_key` 的游标 422；列表的读在一个 `REPEATABLE READ` 的快照里；一组（一格）的"加载更多"是同一个请求加 `group_key`（`sub_group_key`），不再是"加上这一组的筛选"；游标的载荷写明排序、以 `id` 结尾；未知的 `order_by`、`group_by` 422（3.16，架构问题 6） |
| P5 | 总体设计 | 6.2 | 列表的词汇在 `issue/domain`，`project` 经它声明、`issue.Provide` 实现的 `IssueListSettings` 检查显示设置（3.17，架构问题 10） |
| P5 | M3 设计 | 3.18 | 项目的显示设置加 `filters`、`display_filters`、`display_properties` 三列，格式见本文 3.17 |
| P5 | 差异清单 | 二·按表、三、四 | `project_user_properties` 加三列、不建 `rich_filters`（4.15）；三：列表的响应、排序键、分组键的名字；四：4.17 中标 P5 的行 |
| P6 | 总体设计 | 6.2、6.3 | `shared` 加事件的值和 `EventPublisher`；6.3 第 3 条写明订阅者的写法：同步、在事务里、出错回滚，异步的事经 `InsertTx`（3.19，决策点 3） |
| P6 | 总体设计 | 6.7 | 定时任务的清单和间隔、两个队列；测试的时钟偏移（3.20，架构问题 7） |
| P6 | M2 设计 | 3.15、3.17 | 事务内投递经只投递的 River 客户端（在模块之前建出）；队列与并发；定时任务和模块声明会投递的种类没有 worker 时构造失败（3.20） |
| P6 | 差异清单 | 四 | 自动归档的执行者为空（决策点 4）；按间隔而不是固定时刻运行 |
| P7 | 总体设计 | 6.2、8（安全） | `platform/outbound`：防 SSRF 的出站客户端，M8 复用（3.11、8.1，架构问题 8） |
| P7 | 差异清单 | 二·按表、四 | `issue_links` 逐列（4.8）；抓取的上限、`metadata` 的形状 |
| P7 | README | "安全"一节 | 8.8 中 P7 的行；P6、P9 的"部署"行同样按 8.8 |
| P8 | 总体设计 | 5.4 | 草稿：`project_id` 是一列；发布用存着的载荷、要求 `issue.create`；409 `issue.draft_changed`（3.15） |
| P8 | 总体设计 | 6.9 | `description_stripped` 不用于搜索；搜索的匹配和顺序、选择器的排除、`q` 为空时给出最近更新的、日期的条件、编号的折叠（3.7、3.18） |
| P8 | 差异清单 | 二·按表、四 | `draft_issues` 逐列（4.13）；`issues` 的 trigram 索引落地；4.17 中标 P8 的行 |
| P8 | README | "部署"一节 | 迁移执行 `CREATE EXTENSION pg_trgm`（库属主即可，8.8） |
| P9 | 总体设计 | 5.5、6.7 | 60 天清理：每个模块清理自己的表，一个任务按顺序运行；删除关系图（3.21、4.16，架构问题 9）；描述版本的保留与清理在同一段，两者不相遇 |
| P9 | 总体设计 | 5.5、6.3 | "连带删除在同一个事务里同步完成"改为：删除项目、工作区时工作项一侧的行由事件的订阅者投递的任务分批软删除，写下请求的时刻和删除者；第一个跨模块的异步副作用（`project.deleted`、`workspace.deleted`）；v0 的规模目标（一个项目至多约 2 万个未删除的工作项）（3.6、3.19，决策点 7） |
| P9 | M3 设计 | 3.6、13.2 | 删除项目、工作区的步骤后面是发布事件，不再调 `IssueCascade` 的删除；13.2 设想的"删除工作区的连带挂到事件上"对工作项一侧做了（3.19） |
| P9 | 差异清单 | 四 | 动态随工作项物理删除（`CASCADE`，不是 `DO_NOTHING`）；已登记的"连带在同一事务"改为只对删除工作项、状态、标签成立，删除项目、工作区的工作项一侧与 Plane 一样在事务之外完成，但写下请求的时刻（3.6） |
| P10b | 总体设计 | 7.7 | `Reconciled.fetchPart` 和 `paged-groups.ts`；一个工作项只有一份，取服务端 `updated_at` 大的（3.22） |
| P10a–P16 | 前端改动清单 | 3.1、3.2 | M4 一行的状态；删到 M5、M6、M7 的入口和取数（3.1、3.2）；迭代、模块、项目视图、工作区视图的工作项列表在 P10a 删除，M6、M7 加回（7.3） |
| P12 | README | "前端"一节 | "M3 中看到的页面"改写为 M4 能到达的页面（3.1） |
| 收尾 | 总体设计 | 9.4 | M4 的状态改为"已完成" |

---

## 4. 数据模型

按 M2 设计 3.13 的表结构约定和 M3 设计第 4 节开头的两处补充建表：外键的 `ON DELETE` 照搬模型的 `on_delete`，不用 `DEFERRABLE`；默认值和 CHECK 从模型读来、在差异清单中写为"新加"；审计时间由用例的时钟写入；不建 `*_like` 索引；`created_by_id`、`updated_by_id` 不建索引；只保留部分唯一索引，不要 Django 的 `UNIQUE (…, deleted_at)`；物理级联用到的外键列（指向 `workspaces`、`projects`、`issues`、`issue_comments`、`labels`、`states` 的 `CASCADE`）有不带条件的 `<表>_<列>_idx`。Plane 的表结构出处是 `tools/plane-schema/plane-v1.4.2-schema.sql`（下称快照）和 `API/db/models/issue.py`、`draft.py`。

M4 的 13 张表都带 `workspace_id`、`project_id`（草稿的可空）、`deleted_at`、`updated_by_id`：按目录的删除守卫要求 `workspace_id`（`bootstrap/workspace_deletion_catalog_test.go`），`expectProjectDeleted` 要求有 `project_id` 的表的行与项目同一时刻、由同一个人软删除（3.6）。每张表的 `id` 由应用生成（UUIDv7），没有默认值。

### 4.1 迁移文件
每张表随它的第一个操作所在的 Phase 建出；文件名 `<版本>_<模块>_<内容>.sql`（M2 设计 3.14），版本从 `00015` 起。

| 文件 | 内容 | Phase |
|---|---|---|
| `00015_issue_issues.sql` | `issues` | P1a |
| `00016_issue_issue_assignees.sql` | `issue_assignees` | P1a |
| `00017_issue_issue_labels.sql` | `issue_labels` | P1a |
| `00018_issue_issue_subscribers.sql` | `issue_subscribers` | P1a |
| `00019_issue_issue_activities.sql` | `issue_activities` | P1a |
| `00020_issue_issue_relations.sql` | `issue_relations` | P3 |
| `00021_issue_issue_comments.sql` | `issue_comments` | P4 |
| `00022_issue_issue_reactions.sql` | `issue_reactions` | P4 |
| `00023_issue_comment_reactions.sql` | `comment_reactions` | P4 |
| `00024_issue_issue_mentions.sql` | `issue_mentions` | P4 |
| `00025_issue_issue_description_versions.sql` | `issue_description_versions` | P4 |
| `00026_project_project_user_properties_issue_display.sql` | `project_user_properties` 加三列（4.15） | P5 |
| `00027_issue_issue_links.sql` | `issue_links` | P7 |
| `00028_issue_draft_issues.sql` | `draft_issues` | P8 |
| `00029_issue_issues_name_trgm.sql` | `CREATE EXTENSION IF NOT EXISTS pg_trgm`；`issues_name_trgm_idx` | P8 |
| `00030_identity_purge_indexes.sql`、`00031_workspace_purge_indexes.sql`、`00032_project_purge_indexes.sql`、`00033_issue_purge_indexes.sql` | 每张被清理的表的 `<表>_deleted_at_idx ON (deleted_at) WHERE deleted_at IS NOT NULL`（3.21） | P9 |

- `server/sqlc.yaml` 加 `issue` 条目（P1a），`schema:` 列本模块的迁移；`00026` 归 `project` 的条目。
- 每个建表的 Phase 同时把新表加进 `deploy/runtime-grants.sql`（M2 交接第 6 节；`TestTheGrantsFileCoversEveryRelationAndFunction`）；`00029` 的扩展函数在 `public`，靠 PUBLIC 的默认权限（S2 核对）。
- 冒烟故事 S1 照旧断言最新的版本等于最后一个迁移文件（M4 加 19 个）。

### 4.2 `issues`（Plane 28 列 → 20 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 主键名 `issues_pkey`（Plane 是 `issue_pkey`，快照 :3274） |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `project_id` | `uuid NOT NULL REFERENCES projects ON DELETE CASCADE` | — |
| `parent_id` | `uuid REFERENCES issues ON DELETE CASCADE`，`issues_not_own_parent_check CHECK (parent_id <> id)` | CHECK 新加（3.9；多列的 CHECK 有名字，M2 设计 3.13） |
| `state_id` | `uuid NOT NULL REFERENCES states ON DELETE CASCADE` | Plane 可空；Nerve 的创建总有状态（项目恰有一个默认状态）。`CASCADE` 照搬：删除状态在有工作项用它时被拒绝（3.6），所以只会删掉已软删除的工作项 |
| `name` | `varchar(255) NOT NULL`，`CHECK (name <> '')` | CHECK 新加 |
| `description_html` | `text NOT NULL DEFAULT '<p></p>'` | — （P1a 建列，接口在 P4 才有这个字段，3.7） |
| `description_stripped` | `text` | — |
| `priority` | `varchar(30) NOT NULL DEFAULT 'none'`，`CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none'))` | 默认值、CHECK 新加 |
| `start_date`、`target_date` | `date`，`issues_dates_check CHECK (start_date <= target_date)` | CHECK 新加（3.7）；P1a 建列，接口在 P2 才有这两个字段 |
| `sequence_id` | `integer NOT NULL`，`CHECK (sequence_id > 0)` | 没有默认值（取自计数器）；CHECK 新加 |
| `sort_order` | `double precision NOT NULL DEFAULT 65535` | — |
| `completed_at` | `timestamptz` | — |
| `archived_at` | `timestamptz` | Plane 是 `date`（差异清单已登记） |
| `created_by_id`、`updated_by_id` | `uuid REFERENCES users ON DELETE SET NULL` | — |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 由应用写入 |
| `deleted_at` | `timestamptz` | — |

- **删除的列**：`point`、`estimate_point_id`、`type_id`（估算、工作项类型已砍）、`is_draft`（草稿是另一张表）、`description_binary`（协同编辑的遗留）、`external_source`、`external_id`（差异清单已登记）；`description_json`（没有读者，3.7；新登记）。
- **索引**：`issues_project_id_sequence_id_key UNIQUE (project_id, sequence_id)`（全表唯一，3.7；也服务按编号取）；`issues_workspace_id_idx`、`issues_project_id_idx`、`issues_parent_id_idx`、`issues_state_id_idx`（不带条件：物理级联、子工作项、`StateInUse`）；`issues_project_id_state_id_idx ON (project_id, state_id) WHERE deleted_at IS NULL`（`sort_order` 的最大值、按状态筛选）；列表的排序用不上索引（S1，3.16），P5 的 spec 按实测的计划只加用得到的；`issues_name_trgm_idx ON issues USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL`（P8）。Plane 只有外键列的 btree 索引，没有任何唯一约束（R1 第 1 节）。

### 4.3 `issue_assignees`（Plane 10 列 → 10 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2 | — |
| `issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE` | — |
| `assignee_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | 照搬（v0 不物理删除账户，不触发） |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`issue_assignees_issue_id_assignee_id_key ON (issue_id, assignee_id) WHERE deleted_at IS NULL`（照搬，快照 :5473）；`issue_assignees_issue_id_idx`、`issue_assignees_workspace_id_idx`、`issue_assignees_project_id_idx`（不带条件）；`issue_assignees_assignee_id_idx ON (assignee_id) WHERE deleted_at IS NULL`（个人主页"已分配"、`assignee_id` 的筛选）。
- 写法：修改按差集增删（3.7），不再每次保存删除全部再插入。

### 4.4 `issue_labels`（Plane 10 列 → 10 列）
列与 4.3 同形，`assignee_id` 换成 `label_id uuid NOT NULL REFERENCES labels ON DELETE CASCADE`。
- **索引**：**新加**部分唯一 `issue_labels_issue_id_label_id_key ON (issue_id, label_id) WHERE deleted_at IS NULL`（Plane 没有任何约束，可以重复，R1 第 3 节；差异清单已登记"新增"）；`issue_labels_issue_id_idx`、`issue_labels_label_id_idx`（不带条件：标签的物理级联）、`workspace_id`、`project_id` 的索引。
- 删除标签软删除它的行（3.6）。

### 4.5 `issue_subscribers`（Plane 10 列 → 10 列）
列与 4.3 同形，`assignee_id` 换成 `subscriber_id uuid NOT NULL REFERENCES users ON DELETE CASCADE`。
- **索引**：`issue_subscribers_issue_id_subscriber_id_key ON (issue_id, subscriber_id) WHERE deleted_at IS NULL`（照搬）；`issue_subscribers_issue_id_idx` 等不带条件的三个；`issue_subscribers_subscriber_id_idx ON (subscriber_id) WHERE deleted_at IS NULL`（"已关注"、`subscriber_id` 的筛选）。
- 写入只有自动关注（3.12）；关注的接口在 M7。

### 4.6 `issue_relations`（Plane 11 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2；`project_id` 是 `issue_id` 那一端的项目 | Plane 写请求路径的项目（3.10） |
| `issue_id`、`related_issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE`，`issue_relations_not_self_check CHECK (issue_id <> related_issue_id)` | CHECK 新加 |
| `relation_type` | `varchar(20) NOT NULL`，`CHECK (relation_type IN ('blocked_by', 'duplicate', 'relates_to'))` | Plane 默认 `blocked_by`、不检查取值；Nerve 不设默认（接口必填）、只有页面的三种存储值（3.10） |
| 审计列 | 同 4.2 | — |

- **索引**：`issue_relations_pair_key ON (LEAST(issue_id, related_issue_id), GREATEST(issue_id, related_issue_id)) WHERE deleted_at IS NULL`（新加，取代 Plane 有方向的 `(issue_id, related_issue_id)`，3.10）；`issue_relations_issue_id_idx`、`issue_relations_related_issue_id_idx`（不带条件：两个方向的读取和物理级联）、`workspace_id`、`project_id`。

### 4.7 `issue_comments`（Plane 20 列 → 13 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2 | — |
| `issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE` | — |
| `actor_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | Plane 可空（"系统也能评论"）；Nerve 没有不是人写的评论，是作者 |
| `comment_html` | `text NOT NULL DEFAULT '<p></p>'` | — |
| `comment_stripped` | `text NOT NULL DEFAULT ''` | 默认值新加 |
| `edited_at` | `timestamptz` | — |
| 审计列 | 同 4.2 | — |

- **删除的列**：`comment_json`（没有读者，M1 收尾交接）、`attachments`（从未写过）、`access`（M1-P3 交接：没有内外之分）、`parent_id`（决策点 6）、`description_id` 和它的唯一约束（`descriptions` 表不保留，差异清单已登记）、`external_source`、`external_id`。
- **索引**：`issue_comments_issue_id_idx`（不带条件）；`issue_comments_issue_id_created_at_idx ON (issue_id, created_at, id) WHERE deleted_at IS NULL`（评论列表的游标）；`workspace_id`、`project_id`。

### 4.8 `issue_links`（Plane 12 列 → 12 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2 | — |
| `issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE` | — |
| `title` | `varchar(255)` | — |
| `url` | `text NOT NULL`，`CHECK (url <> '' AND length(url) <= 2048)` | CHECK 新加（3.11） |
| `metadata` | `jsonb NOT NULL DEFAULT '{}'`，CHECK：是对象，键是 `title`、`favicon`、`url`、`favicon_url`、`error` 的子集，每个值是字符串或 `null` | CHECK 新加（M2 设计 3.13） |
| 审计列 | 同 4.2 | — |

- **索引**：**新加**部分唯一 `issue_links_issue_id_url_key ON (issue_id, url) WHERE deleted_at IS NULL`（Plane 只在代码里检查）；`issue_links_issue_id_idx` 等。

### 4.9 `issue_reactions`（Plane 11 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2 | — |
| `issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE` | — |
| `actor_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | — |
| `reaction` | `text NOT NULL`，`CHECK (reaction ~ '^[0-9]{1,7}(-[0-9]{1,7}){0,15}$' AND length(reaction) <= 128)` | CHECK 新加（3.12） |
| 审计列 | 同 4.2 | — |

- **索引**：`issue_reactions_issue_id_actor_id_reaction_key ON (issue_id, actor_id, reaction) WHERE deleted_at IS NULL`（照搬）；`issue_reactions_issue_id_idx` 等。

### 4.10 `comment_reactions`（Plane 11 列 → 11 列）
列与 4.9 同形，`issue_id` 换成 `comment_id uuid NOT NULL REFERENCES issue_comments ON DELETE CASCADE`。没有 `issue_id`（Plane 相同），列出一个工作项的评论的表情经评论连接。
- **索引**：`comment_reactions_comment_id_actor_id_reaction_key`（照搬）；`comment_reactions_comment_id_idx`（不带条件：评论的物理级联）等。

### 4.11 `issue_activities`（Plane 20 列 → 17 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2；`project_id` 是这一行的工作项的项目 | Plane 的反向关联行写本项目（3.13） |
| `issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE` | Plane 可空、`DO_NOTHING`（R1 第 11 节）。登记差异 |
| `verb` | `varchar(255) NOT NULL DEFAULT 'created'`，`CHECK (verb IN ('created', 'updated', 'deleted'))` | CHECK 新加；M7 的收集箱状态（`"-2"`…`"2"`）随它的迁移放宽 |
| `field` | `varchar(255)` | — |
| `old_value`、`new_value` | `text` | — |
| `old_identifier`、`new_identifier` | `uuid`（不是外键，指向状态、标签、账户、链接、工作项） | — |
| `comment` | `text NOT NULL DEFAULT ''` | 默认值新加；文字去掉品牌（3.13） |
| `actor_id` | `uuid REFERENCES users ON DELETE SET NULL` | 照搬；为空表示 nerve 自己（决策点 4） |
| 审计列 | 同 4.2 | Plane 的 `created_by_id`、`updated_by_id` 实际总为空（Celery 里没有当前用户）；Nerve 写执行者，删除时写删除者 |

- **删除的列**：`issue_comment_id`（评论不写动态，3.13；它的 `DO_NOTHING` 本会挡住物理清理，M2 交接第 5 节）、`epoch`（只写不读）、`attachments`（从不写）。
- **索引**：`issue_activities_issue_id_created_at_idx ON (issue_id, created_at, id)`（不带条件：列表的游标、描述的合并找"最新的一行"、物理级联）；`workspace_id`、`project_id`。Plane 只有外键列的索引。
- 这是最大的表（R1 第 18 节）；Plane 不清理活的动态，Nerve 也不（总体设计 6.7 的保留只列描述版本、日志）。

### 4.12 `issue_mentions`（Plane 10 列 → 10 列）
列与 4.3 同形，`assignee_id` 换成 `mention_id uuid NOT NULL REFERENCES users ON DELETE CASCADE`。
- **索引**：`issue_mentions_issue_id_mention_id_key`（照搬）；`issue_mentions_issue_id_idx` 等；`issue_mentions_mention_id_idx ON (mention_id) WHERE deleted_at IS NULL`（`mention_id` 的筛选）。

### 4.13 `draft_issues`（Plane 24 列加 4 张关联表 → 9 列，1 张表）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `project_id` | `uuid REFERENCES projects ON DELETE CASCADE` | 照搬（可空） |
| `payload` | `jsonb NOT NULL DEFAULT '{}'`，`CHECK (jsonb_typeof(payload) = 'object')` | 取代 Plane 与工作项同形的 18 列和 4 张关联表（总体设计 5.4，差异清单已登记）；键由接口的结构检查（3.15） |
| `created_by_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | Plane 可空、`SET NULL`；草稿的拥有者，每个读取都按它过滤。`NOT NULL` 与 `SET NULL` 矛盾，改为 `CASCADE`（不触发）。登记差异 |
| `updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`draft_issues_workspace_id_created_by_id_created_at_idx ON (workspace_id, created_by_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`（列表的游标，PAT 列表的同一写法）；`draft_issues_workspace_id_idx`、`draft_issues_project_id_idx`（不带条件）。
- M5 加 `file_assets.draft_issue_id`（`CASCADE`）指向它（R1 第 15 节）。

### 4.14 `issue_description_versions`（Plane 15 列 → 13 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id`、`workspace_id`、`project_id` | 同 4.2 | — |
| `issue_id` | `uuid NOT NULL REFERENCES issues ON DELETE CASCADE` | — |
| `description_html` | `text NOT NULL DEFAULT '<p></p>'` | — |
| `description_stripped` | `text` | — |
| `last_saved_at` | `timestamptz NOT NULL` | 由应用写入（Plane 默认 `now()`） |
| `owned_by_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | — |
| 审计列 | 同 4.2；`created_by_id`、`updated_by_id` 是写这一版的人 | Plane 抄工作项的（3.14） |

- **删除的列**：`description_binary`、`description_json`（同 4.2）。
- **索引**：`issue_description_versions_issue_id_last_saved_at_idx ON (issue_id, last_saved_at, id)`（不带条件：列表、合并、保留、物理级联）；`workspace_id`、`project_id`。
- `issue_versions`（Plane 33 列）不建（决策点 5）；差异清单第一节"未保留的表"加一行。

### 4.15 `project_user_properties` 的三列（M3 的 11 列 → 14 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `filters` | `jsonb NOT NULL DEFAULT '{}'`；CHECK：是对象，键是项目页的筛选参数名的子集（`state_id`、`state_group`、`priority`、`label_id`、`assignee_id`、`created_by_id`、`mention_id`、`subscriber_id`、`start_date`、`target_date`），每个值是字符串的数组（`jsonb_path_exists` 写成，函数是 immutable 的） | Plane 的 `filters`（旧格式）和 `rich_filters`（表达式）合为这一列，格式是 Nerve 的列表参数（3.17） |
| `display_filters` | `jsonb NOT NULL DEFAULT '{"layout": "list", "group_by": null, "sub_group_by": null, "order_by": "-created_at", "sub_issue": true, "show_empty_groups": true, "calendar": {"layout": "month", "show_weekends": false}}'`；CHECK：恰好这些键，`layout`、`order_by` 是字符串，`group_by`、`sub_group_by` 是字符串或 `null`，两个布尔值，`calendar` 恰好两个键、类型各自对 | 去掉 `type`、`calendar_date_range`，加 `calendar`；取值由 `issue` 的列表词汇经 `IssueListSettings` 检查（3.17） |
| `display_properties` | `jsonb NOT NULL DEFAULT`（11 个键都为 `true`）；CHECK：恰好这 11 个键，都是布尔值 | 去掉 `estimate`；`attachment_count`（M5）、`modules`、`cycle`（M6）随各自的迁移加回 |

- 每个 CHECK 照 M2 设计 3.13 包在 `CASE WHEN jsonb_typeof(…) = 'object'` 里，数组、标量、JSON `null` 也得到 `check_violation`；P5 的迁移测试逐条试（M3 的 `preferences` 的同一写法）。
- 已有的行取默认值（`ADD COLUMN … DEFAULT` 不重写表）。

### 4.16 删除关系图（M4 的表）
在 M3 设计 4.12 的图上延伸。M4 的表没有 `DO_NOTHING` 的外键。
```
users
 ├── issue_assignees.assignee_id、issue_subscribers.subscriber_id、issue_mentions.mention_id   ON DELETE CASCADE
 ├── issue_comments.actor_id、issue_reactions.actor_id、comment_reactions.actor_id            ON DELETE CASCADE
 ├── issue_description_versions.owned_by_id、draft_issues.created_by_id                      ON DELETE CASCADE
 ├── issue_activities.actor_id                                                               ON DELETE SET NULL
 └── 13 张表的 created_by_id（草稿除外）、updated_by_id                                       ON DELETE SET NULL
workspaces
 └── 13 张表的 workspace_id                                                                  ON DELETE CASCADE
projects
 ├── 12 张表的 project_id                                                                    ON DELETE CASCADE
 └── draft_issues.project_id                                                                 ON DELETE CASCADE（可空）
states
 └── issues.state_id                                                                         ON DELETE CASCADE（在用时不能删除状态，3.6）
labels
 └── issue_labels.label_id                                                                   ON DELETE CASCADE
issues
 ├── issues.parent_id                                                                        ON DELETE CASCADE
 ├── issue_assignees、issue_labels、issue_links、issue_subscribers、issue_mentions、
 │   issue_comments、issue_reactions、issue_description_versions 的 issue_id                  ON DELETE CASCADE
 ├── issue_relations.issue_id、issue_relations.related_issue_id                              ON DELETE CASCADE
 └── issue_activities.issue_id                                                               ON DELETE CASCADE（Plane DO_NOTHING）
issue_comments
 └── comment_reactions.comment_id                                                            ON DELETE CASCADE
（draft_issues.payload 里的状态、标签、负责人、父工作项的 id 不是外键：发布时检查，3.15）
```
- 每条指向 `workspaces`、`projects`、`states`、`labels`、`issues`、`issue_comments` 的外键都有不带条件的索引；清理的级联按它们找子行（3.21）。
- 以后的 M 加入的反向外键（`ON DELETE` 照搬 Plane，与这张图一致）：M5 `file_assets.issue_id`、`comment_id`、`draft_issue_id`（`CASCADE`）；M6 `cycle_issues.issue_id`、`module_issues.issue_id`（`CASCADE`，`cycle_issues` 另有 `(issue_id)` 的部分唯一）；M7 `intake_issues.issue_id`（`CASCADE`）、`duplicate_to_id`（`SET NULL`）（R1 第 15 节；13.2）。M4 不指向它们的任何表。
- 通知、最近访问、收藏按 `entity_identifier` 引用工作项、没有外键（M7）：工作项删除时 Plane 删除最近访问（`API/app/views/issue/base.py:721-727`），M7 决定（13.2）。

### 4.17 行为差异（登记到差异清单第四节）
| 方面 | Plane | Nerve | Phase |
|---|---|---|---|
| 工作项的写的取锁 | Celery 里异步写动态，写之间不串行 | 项目行 N，同一事务里写动态、关注、提及、版本（3.5，决策点 1） | P1a |
| 一个项目上的并发写 | 不限 | 每个进程里一个项目至多 K 个写同时在数据库里，多的在进程里等，等不到 503 `server_busy`（`Retry-After`）（3.5，架构问题 11） | P1b |
| 删除项目、工作区时工作项一侧的行 | Celery 在事务之外异步软删除，各自的时刻 | 后台任务分批软删除，写下删除请求的时刻和删除者；完成之前它们对任何读写都不存在（3.6，决策点 7） | P9 |
| 负责人、标签不合格 | 静默去掉 | 422 `not_allowed`（3.7） | P1a |
| 起止日期 | 两个都在请求里才检查 | 按合并之后的值检查；数据库另有 CHECK（3.7） | P2 |
| 默认负责人 | 指派但不关注 | 指派并关注（3.7、3.12） | P1a |
| 负责人的动态 | 按请求里的 id 写，含被去掉的 | 只为实际指派的写（3.13） | P1a |
| 只补块 id 的保存 | 看客户端的标志，改 `updated_at` | 内容标识（纯文本、提及、链接的地址、勾选框的状态）不变时不写动态、版本，不改 `updated_at`；否则是普通的修改。剩下的口子：只改格式的保存带上标志就不留动态和版本（3.7） | P4 |
| 描述的动态的合并 | 只改 `created_at`，值是陈旧的 | 保留旧的 `old_value`，改 `new_value`（3.13） | P4 |
| 自动关注 | 每个产生通知的动态的执行者；被提及的人都关注 | 创建者、负责人、评论者、被提及的成员中能读这个工作项的（"只看自己"的访客只在他是创建者时，3.12） | P1a（评论、提及 P4） |
| 编号 | 序列化器允许写 `sequence_id` | 客户端不能写；`(project_id, sequence_id)` 全表唯一（3.7） | P1a |
| 删除状态 | 有工作项时 400 | 409 `project.state_in_use`（3.6） | P1a |
| 修改的响应 | 204 | 200 `IssueDetail`（总体设计 3.1） | P1b |
| 负责人的读取 | 只返回"在某个项目里仍有效"的 | 返回全部未删除的负责人行（3.6） | P1b |
| 创建者的权限 | 只看工作区成员关系，已离开项目也能读、改、删 | 要求是项目的有效成员、角色不低于成员（3.4） | P2 |
| 访客只看自己的工作项 | 只在约 6 处执行 | 处处执行，挂在看不到的工作项下的一切 404；按项目角色算，工作区管理员在项目里是访客时同样（3.4） | P2（各列表在各自的 Phase） |
| 访客删除自己建的工作项 | 服务端允许，页面不显示 | 403（工作区管理员是项目的有效成员时不论项目角色都通过项目的每条规则，总体设计 6.5 的约定；所以只是项目访客的工作区管理员删得了自己建的，204，不是凭 `AllowCreator`）（3.4） | P2 |
| 已归档的工作项 | 只挡修改；评论、链接、表情、关联不挡；能成为父工作项 | 修改 409 `issue.archived`，什么都没改的修改也是，子资源的写同样；恢复、删除照常；在已归档的项目里先答 409 `project.archived`；不能成为父工作项、关联的对方（3.8） | P2（子资源在各自的 Phase） |
| 归档、恢复 | 归档答 `{archived_at}`，恢复 204；再归档 404 | 都返回 `IssueDetail`；重复的不写、200（3.8） | P2 |
| 按编号取 | 格式不对 400，不是成员 403 | 格式不对、不存在、看不到的都是 404，看得到项目而不是它的有效成员的 403（与取工作项相同）；标识经同一个折叠比较，`i`、`ı`、`I`、`İ` 一样（3.7） | P2 |
| 删除工作项 | 动态保留；写一条"已删除"；确认框不提子工作项 | 动态一并软删除；不写"已删除"；确认框写明子树（3.6） | P3（确认框 P14） |
| 父工作项 | 不检查自己、不检查成环 | 422（3.9） | P3 |
| 加子工作项的动态 | `old_value` 是子工作项自己的编号 | 是原来的父工作项（3.13） | P3 |
| 子工作项数 | 数 `issue_objects` 里的直接子工作项（不含已删除、已归档、草稿、分诊状态的，也不含已归档的项目里的），不论调用者看不看得到 | 按调用者的规则数（"只看自己"的访客只数自己建的），按它排序同样（3.9） | P3 |
| 关联 | 六种类型、不检查类型和自己；一对工作项可有两条；显示看不到的项目里的工作项；删除不存在的 500 | 页面的四种；一对一条；对方要可见；按 id 删除（3.10） | P3 |
| 关联的动态 | 对方 id 在 `old_identifier`；反向行的 `project_id` 是本项目；删除的反向行的 `comment` 用正向的类型（`field` 只把 `blocked_by`、`blocking` 反过来），`old_identifier` 是它自己 | 修正（3.13） | P3 |
| 空评论 | 服务端接受 | 422（3.12） | P4 |
| 改、删别人的评论 | 项目管理员可以改、删 | 只有作者能改；作者和项目管理员能删（决策点 8，3.4） | P4 |
| 表情的值 | 任意文本；重复 400 | 码位的格式；重复 409（3.12） | P4 |
| 评论、表情的动态 | 写，从不显示 | 不写（3.13） | P4 |
| 提及的同步 | 与请求体比较，不带描述的修改删掉全部 | 从保存后的描述做集合同步（3.12） | P4 |
| 描述的历史 | 访客能读 | 管理员、成员（3.4） | P4 |
| 动态的列表 | 不分页；不带类型时 500；每行带完整的工作项、项目、工作区；引用看不到的工作项的行照常显示 | 游标分页，只带 `actor_id`；引用调用者看不到的工作项的行略去（3.13） | P4 |
| 时间线的增量读取 | `created_at__gt`，只取新增 | 从最后一页的游标再读，同样只取新增：别的标签页对评论的改、删要到重新打开才更新（3.13，写明的限制） | P4 |
| 列表的分组、排序 | 未知的排序换成 `-created_at`；组的总数 0 报成 1；按标签、负责人分组时数组只有当前页的；按偏移量翻页 | 422；总数准确；数组完整；键集的游标（3.16） | P5 |
| 已归档的项目的工作项列表 | 空（`issue_objects`） | 照常返回，写 409（3.8） | P5 |
| 个人主页的工作项 | 不执行访客规则，访客也能读 | 规则 `member_issue.list` 加访客规则（3.4） | P5 |
| 显示设置 | `filters`、`rich_filters`、`display_filters`、`display_properties` 四列 | 三列，格式是列表的参数（3.17） | P5 |
| 自动归档的执行者 | 项目的创建者 | 空：nerve 自己（3.8，决策点 4） | P6 |
| 定时任务的时刻 | 每天固定的时刻 | 按间隔（3.20） | P6 |
| 链接的地址 | 放行 `ftp`；重复 400 | 只有 `http`、`https`；重复 409（3.11） | P7 |
| 链接的抓取 | 不限响应和图标的大小，存 SVG；改地址时保留旧的元数据 | 1 MiB、64 KiB 的上限，只存位图；改地址时清空；失败只记 `blocked` 或 `failed`；拒绝的地址是 IANA 的整张特殊用途登记表和启动时本机网卡上的地址（3.11、6.8） | P7 |
| 草稿 | 访客也能用；管理员能删别人的；发布用请求体、不检查能否创建；指向父工作项的外键 | 只有管理员、成员，只操作自己的；发布用存着的载荷、要求 `issue.create`；父工作项在载荷里（3.15） | P8 |
| 搜索 | `PROJ-12` 匹配每个项目的 12 号；命令面板不执行访客规则；选择器能造出环 | 精确匹配编号；处处执行访客规则；选择器排除后代、祖先；没有搜索词时给出最近更新的（3.18） | P8 |
| 物理清理 | `DO_NOTHING` 的动态挡住工作项的物理删除 | 动态随工作项 `CASCADE`；每个模块清理自己的表（3.21） | P9 |

### 4.18 授权与 sqlc
- `deploy/runtime-grants.sql`：每张新表 `SELECT, INSERT, UPDATE, DELETE`（物理清理要 `DELETE`），与 M3 的表相同；P1a 起每个建表的 Phase 加。
- sqlc v1.31.1：`issue` 条目的查询放在 `issue/adapter/postgres/queries/`，按表或按用例分文件，每个不超过约 400 行；列表引擎的三条查询（3.16）各一个文件。`date` 列映射到 P2 选定的 Go 类型（6.1，M2 交接第 4 节：不能是 `openapi_types.Date`）。

---

## 5. 接口

### 5.1 操作
一个新模块文件 `api/modules/issue.yaml`（模块文件名等于模块目录名，M2 设计 3.12），37 个操作；`api/modules/project.yaml` 改三个操作。路径都不带结尾 `/`。每个操作写 `security` 和 `x-problem-codes`；下表"主要错误"一栏就是后者的内容，不再列出所有操作都可能返回的平台码。全部带 `bearer`（另隐含 `unauthorized`）；带请求体的都可能返回 400 `bad_request`（结构）。`forbidden` 写作 403 F，`validation_failed` 写作 422 V。按资源 id 寻址的操作，资源所在的项目已归档时答 409 `project.archived`（写），下表写作"409 PA"；工作项已归档时 409 `issue.archived`，写作"409 IA"。工作项一侧的每个写（下表里带 409 PA 的，和 `createIssue`、`publishDraft`）另在 `x-problem-codes` 里声明 503 `server_busy`（写闸门满、`lock_timeout`，3.5；M2 的登录已这样声明），下表不再逐行写出。

契约按模块一个文件（M3 设计 5.1 的约定和它的例外）：`issue.yaml` 预计约 2,500 行，超过约 400 行的规则，是 M3 写明的同一个例外。

**工作项**（`issue`）：

| 方法与路径 | operationId | 请求 | 成功 | 主要错误 |
|---|---|---|---|---|
| `GET /api/v0/projects/{project_id}/issues` | `listIssues` | 查询：3.16 的筛选、`group_by`、`sub_group_by`、`group_key`、`sub_group_key`、`order_by`、`archived`、`limit`、`cursor` | 200 `IssueList` | 404 `project.not_found`；403 F（`archived=true` 的访客）；422 V |
| `POST /api/v0/projects/{project_id}/issues` | `createIssue` | `IssueCreate {name, description_html?, state_id?, priority?, assignee_ids?, label_ids?, start_date?, target_date?, parent_id?}` | 201 `IssueDetail` | 404 `project.not_found`；403 F；422 V；409 `project.archived` |
| `GET /api/v0/issues/{issue_id}` | `getIssue` | — | 200 `IssueDetail` | 404 `issue.not_found`；403 F |
| `PATCH /api/v0/issues/{issue_id}` | `updateIssue` | `IssueUpdate {name?, description_html?, skip_activity?, state_id?, priority?, assignee_ids?, label_ids?, start_date?, target_date?, parent_id?, sort_order?}`（三个可空的用 `nullable.Nullable`） | 200 `IssueDetail` | 404；403 F；422 V；409 IA；409 PA |
| `DELETE /api/v0/issues/{issue_id}` | `deleteIssue` | — | 204 | 404；403 F；409 PA |
| `GET /api/v0/workspaces/{slug}/issues/{identifier}` | `getIssueByIdentifier` | — | 200 `IssueDetail` | 404 `workspace.not_found`；404 `issue.not_found`；403 F |
| `POST /api/v0/issues/{issue_id}/archive` | `archiveIssue` | — | 200 `IssueDetail` | 404；403 F；409 `issue.not_archivable`；409 PA |
| `POST /api/v0/issues/{issue_id}/unarchive` | `unarchiveIssue` | — | 200 `IssueDetail` | 404；403 F；409 PA |
| `GET /api/v0/issues/{issue_id}/sub-issues` | `listSubIssues` | — | 200 `IssueCollection` | 404；403 F |
| `POST /api/v0/issues/{issue_id}/sub-issues` | `addSubIssues` | `SubIssuesAdd {sub_issue_ids}`（1–100） | 200 `IssueCollection` | 404；403 F；422 V（`sub_issue_ids[i]`）；409 IA；409 PA |
| `GET /api/v0/issues/{issue_id}/relations` | `listIssueRelations` | — | 200 `IssueRelationCollection` | 404；403 F |
| `POST /api/v0/issues/{issue_id}/relations` | `createIssueRelations` | `IssueRelationsCreate {relation_type, issue_ids}`（1–20） | 201 `IssueRelationCollection` | 404；403 F；422 V（`issue_ids[i]` 的 `not_allowed`、`duplicate`）；409 IA；409 PA |
| `DELETE /api/v0/issue-relations/{relation_id}` | `deleteIssueRelation` | — | 204 | 404 `issue.relation_not_found`；403 F；409 IA；409 PA |
| `GET /api/v0/issues/{issue_id}/links` | `listIssueLinks` | — | 200 `IssueLinkCollection` | 404；403 F |
| `POST /api/v0/issues/{issue_id}/links` | `createIssueLink` | `IssueLinkCreate {url, title?}` | 201 `IssueLink` | 404；403 F；422 V；409 `issue.link_exists`；409 IA；409 PA |
| `PATCH /api/v0/issue-links/{link_id}` | `updateIssueLink` | `IssueLinkUpdate {url?, title?}` | 200 `IssueLink` | 404 `issue.link_not_found`；403 F；422 V；409 `issue.link_exists`；409 IA；409 PA |
| `DELETE /api/v0/issue-links/{link_id}` | `deleteIssueLink` | — | 204 | 404 `issue.link_not_found`；403 F；409 IA；409 PA |
| `GET /api/v0/issues/{issue_id}/comments` | `listIssueComments` | 查询：`limit`、`cursor` | 200 `IssueCommentPage` | 404；403 F |
| `POST /api/v0/issues/{issue_id}/comments` | `createIssueComment` | `IssueCommentCreate {comment_html}` | 201 `IssueComment` | 404；403 F；422 V；409 IA；409 PA |
| `PATCH /api/v0/issue-comments/{comment_id}` | `updateIssueComment` | `IssueCommentUpdate {comment_html}` | 200 `IssueComment` | 404 `issue.comment_not_found`；403 F；403 `issue.not_author`；422 V；409 IA；409 PA |
| `DELETE /api/v0/issue-comments/{comment_id}` | `deleteIssueComment` | — | 204 | 404 `issue.comment_not_found`；403 F；403 `issue.not_author`；409 IA；409 PA |
| `GET /api/v0/issues/{issue_id}/reactions` | `listIssueReactions` | — | 200 `IssueReactionCollection` | 404；403 F |
| `POST /api/v0/issues/{issue_id}/reactions` | `addIssueReaction` | `ReactionCreate {reaction}` | 201 `IssueReaction` | 404；403 F；422 V；409 `issue.reaction_exists`；409 IA；409 PA |
| `DELETE /api/v0/issues/{issue_id}/reactions/{reaction}` | `removeIssueReaction` | — | 204 | 404；404 `issue.reaction_not_found`；403 F；409 IA；409 PA |
| `POST /api/v0/issue-comments/{comment_id}/reactions` | `addCommentReaction` | `ReactionCreate {reaction}` | 201 `CommentReaction` | 404 `issue.comment_not_found`；403 F；422 V；409 `issue.reaction_exists`；409 IA；409 PA |
| `DELETE /api/v0/issue-comments/{comment_id}/reactions/{reaction}` | `removeCommentReaction` | — | 204 | 404 `issue.comment_not_found`；404 `issue.reaction_not_found`；403 F；409 IA；409 PA |
| `GET /api/v0/issues/{issue_id}/activities` | `listIssueActivities` | 查询：`limit`、`cursor` | 200 `IssueActivityPage` | 404；403 F |
| `GET /api/v0/issues/{issue_id}/description-versions` | `listDescriptionVersions` | 查询：`limit`、`cursor` | 200 `DescriptionVersionPage` | 404；403 F |
| `GET /api/v0/issue-description-versions/{version_id}` | `getDescriptionVersion` | — | 200 `DescriptionVersionDetail` | 404 `issue.description_version_not_found`；403 F |
| `GET /api/v0/workspaces/{slug}/drafts` | `listDrafts` | 查询：`limit`、`cursor` | 200 `DraftPage` | 404 `workspace.not_found`；403 F |
| `POST /api/v0/workspaces/{slug}/drafts` | `createDraft` | `DraftCreate {project_id?, payload}` | 201 `Draft` | 404；403 F；422 V（`project_id`） |
| `GET /api/v0/drafts/{draft_id}` | `getDraft` | — | 200 `Draft` | 404 `issue.draft_not_found` |
| `PATCH /api/v0/drafts/{draft_id}` | `updateDraft` | `DraftUpdate {project_id?, payload?}` | 200 `Draft` | 404 `issue.draft_not_found`；403 F；422 V；409 `issue.draft_changed` |
| `DELETE /api/v0/drafts/{draft_id}` | `deleteDraft` | — | 204 | 404 `issue.draft_not_found`；403 F；409 `issue.draft_changed` |
| `POST /api/v0/drafts/{draft_id}/publish` | `publishDraft` | — | 201 `IssueDetail` | 404 `issue.draft_not_found`；404 `project.not_found`；403 F；400（载荷的结构，与 `createIssue` 相同）；422 V；409 `issue.draft_changed`；409 `project.archived` |
| `GET /api/v0/workspaces/{slug}/members/{user_id}/issues` | `listMemberIssues` | 查询：`relation`、3.16 的筛选（个人主页的那些）、`group_by`、`group_key`、`order_by`、`limit`、`cursor` | 200 `IssueList` | 404 `workspace.not_found`；404 `workspace.member_not_found`；403 F；422 V |
| `GET /api/v0/workspaces/{slug}/issue-search` | `searchIssues` | 查询：`q`（可选）、`project_id`、`for`、`issue_id`、`start_date`、`target_date`、`limit` | 200 `IssueCollection` | 404 `workspace.not_found`；404 `project.not_found`；404 `issue.not_found`；422 V |

**改动的 `project` 操作**：

| 方法与路径 | operationId | 改动 |
|---|---|---|
| `GET /api/v0/me/projects/{project_id}/preferences` | `getProjectPreferences` | `ProjectPreferences` 加 `filters`、`display_filters`、`display_properties`（5.2） |
| `PATCH /api/v0/me/projects/{project_id}/preferences` | `updateProjectPreferences` | `ProjectPreferencesUpdate` 加同样三个可选字段，各自整体替换 |
| `DELETE /api/v0/states/{state_id}` | `deleteState` | `x-problem-codes` 加 409 `project.state_in_use` |

- `Project` 的 `identifier`、`default_assignee_id`、`guest_view_all_features`、`archive_in` 的说明改写为它们对工作项的作用（M3 交接第 4 节）：编号的前缀；不带负责人的新工作项指派给他（他仍是项目中不是访客的有效成员时）；关时访客只看到自己建的工作项；自动归档的月数。`State.default` 的说明加"新工作项不带状态时取它"。
- 各列表的顺序：`listIssues`、`listMemberIssues` 按 `order_by`（3.16）；子工作项按 `sort_order`、`id`（父工作项详情里的列表，与 Plane 的页面默认相同）；关联、链接、工作项的表情按 `(created_at, id)`；评论、动态按 `(created_at, id)` 升序；描述的历史按 `(last_saved_at, id)` 降序；草稿按 `(created_at, id)` 降序；搜索按 3.18。

### 5.2 结构
- **关联字段只给 id，名字带 `_id`**（总体设计 3.2、3.6）：Plane 的 `state`、`parent`、`created_by`、`actor`、`owned_by` 改为 `state_id`、`parent_id`、`created_by_id`、`actor_id`、`owned_by_id`。
- **`Issue`**（列表行）：`{id, workspace_id, project_id, sequence_id, name, state_id, priority, start_date, target_date, parent_id, sort_order, completed_at, archived_at, assignee_ids, label_ids, sub_issues_count, link_count, created_by_id, updated_by_id, created_at, updated_at}`。日期是 `format: date`，时间是 `date-time`。没有 Plane 的 `estimate_point`、`cycle_id`、`module_ids`、`attachment_count`、`is_draft`、`is_intake`、`is_subscribed`、`state__group`（3.2；状态组由页面从状态目录读）。字段随它的数据进入接口（第 12 节约束 5）：`start_date`、`target_date` 在 P2，`sub_issues_count` 在 P3，`link_count` 在 P8，其余在 P1a（随 `createIssue`）。
- **`IssueDetail`** = `Issue` 加 `description_html`（取单个、创建、修改、归档、恢复、发布）。不用 `allOf`（生成器不支持，M2 设计 3.11），字段写两遍，`apitest` 的一个检查核对 `IssueDetail` 的字段是 `Issue` 的加一个。描述在 P4 才进入接口（3.7，第 12 节约束 5）：P1–P3 的 `IssueDetail` 与 `Issue` 同形，`IssueCreate`、`IssueUpdate` 没有 `description_html`、`skip_activity`；P4 加上它们和这个检查。
- **`IssueCreate`**：`name`（必填，1–255）、`description_html`（≤ 500000）、`state_id`、`priority`、`assignee_ids`、`label_ids`（各 ≤ 100，`uniqueItems` 不用：重复由用例答 422 `duplicate`，路径带下标）、`start_date`、`target_date`、`parent_id`。**`IssueUpdate`**：同样的字段都可选，加 `skip_activity`（布尔）和 `sort_order`；`start_date`、`target_date`、`parent_id` 可清空。**`IssueDraftPayload`**：`IssueCreate` 的字段全部可选，`name` 可以为空串（3.15）。
- **`IssueList`**：`{total, data?, next_cursor?, groups?, subgroup_totals?}`：不带 `group_by`、或带 `group_key`（一组的"加载更多"）时有 `data`、`next_cursor`，否则有 `groups`；子分组时另有 `subgroup_totals`（3.16）。不用 `oneOf`（同上），接口描述写明哪一种出现；前端按 `groups` 是否存在分支（7.2）。`IssueGroup {key, total, data?, next_cursor?, subgroups?}`、`IssueSubgroup {key, total, data, next_cursor}`、`IssueSubgroupTotal {key, total}`。
- **`IssueCollection`**、`IssueRelationCollection`、`IssueLinkCollection`、`IssueReactionCollection`：`{data: [...]}`（集合型，3.16）。
- **`IssueRelation`**：`{id, relation_type, issue: Issue, created_at}`：`relation_type` 是 `relates_to`、`duplicate`、`blocked_by`、`blocking`，从路径里的工作项看；`issue` 是对方（3.10）。**`IssueRelationsCreate`**：`{relation_type, issue_ids}`。
- **`IssueLink`**：`{id, issue_id, url, title, metadata, created_by_id, created_at, updated_at}`；**`IssueLinkMetadata`**：封闭的对象，字段 `title`、`favicon`、`url`、`favicon_url`、`error` 都可选、可为 `null`（`error` 是枚举 `blocked`、`failed`），`{}` 表示还没有抓取（3.11）。
- **`IssueComment`**：`{id, issue_id, actor_id, comment_html, edited_at, created_at, updated_at, reactions: CommentReaction[]}`：内嵌它的表情（评论自己的子行；与 M3 的成员内嵌资料同一类例外，3.12）。
- **`IssueReaction`**：`{id, issue_id, actor_id, reaction, created_at}`；**`CommentReaction`**：`{id, comment_id, actor_id, reaction, created_at}`；**`ReactionCreate`**：`{reaction}`，`pattern` 与数据库的 CHECK 相同。
- **`IssueActivity`**：`{id, issue_id, project_id, verb, field, old_value, new_value, old_identifier, new_identifier, actor_id, created_at}`。不返回 `comment`（3.13）。`verb` 是字符串（M7 的收集箱状态会加取值，不做成枚举）。
- **`DescriptionVersion`**：`{id, issue_id, owned_by_id, last_saved_at, created_at}`；**`DescriptionVersionDetail`** 加 `description_html`。
- **`Draft`**：`{id, workspace_id, project_id, payload: IssueDraftPayload, created_at, updated_at}`；**`DraftCreate`** `{project_id?, payload}`、**`DraftUpdate`** `{project_id?, payload?}`（`project_id` 可清空）。
- **分页的封套**：`IssueCommentPage`、`IssueActivityPage`、`DescriptionVersionPage` 是 `{data, next_cursor}`；`DraftPage` 另带 `total`（草稿页的计数）。
- **`ProjectPreferences`** 加：`filters: IssueFilters`（封闭的对象，3.17 的十个键都可选，值是字符串的数组）、`display_filters: IssueDisplayFilters`（`layout`：`list`、`kanban`、`calendar`、`spreadsheet`；`group_by`、`sub_group_by`：`IssueGroupBy` 或 `null`；`order_by`：`IssueOrderBy`；`sub_issue`、`show_empty_groups`；`calendar: {layout: month | week, show_weekends}`）、`display_properties: IssueDisplayProperties`（11 个布尔值，都必有）。
- **枚举**：`IssuePriority`、`IssueGroupBy`、`IssueOrderBy`、`IssueRelationType`、`IssueMemberRelation`（`assigned`、`created`、`subscribed`）、`IssueSearchFor`。前端的 `TIssuePriorities`、`TIssueGroupByOptions`、`TIssueOrderByOptions`、`TIssueRelationTypes` 由它们取代（7.2）。
- **查询参数**写在 `issue.yaml` 的 `components/parameters`：筛选参数是 `style: form, explode: false` 的字符串数组（逗号分隔），日期参数是 `explode: true`（可重复），值的语法由用例按 `issue/domain` 的列表词汇检查（3.17），不在接口描述里用 `pattern`：一个值可以是 id、`none` 或 `me`。
- **结构中新的开放 map**：没有（`IssueFilters` 的键是已知的十个，是封闭的对象；M2 交接第 4 节的 map 一项写明"没有用到"）。请求体中第一次出现字符串的数组（`assignee_ids` 等）和可空的日期：`bodyshape` 的代价测试加上它们（M2 交接第 4 节）。

### 5.3 错误码
| 码 | 状态 | 含义 |
|---|---|---|
| `issue.not_found` | 404 | 工作项不存在、已删除，或调用者看不到（看不到项目，或访客规则，3.4）；按编号取时格式不对、项目不存在也是它 |
| `issue.archived` | 409 | 已归档的工作项的修改（什么都没改的也是）和它的子资源的写（3.8） |
| `issue.not_archivable` | 409 | 归档一个状态组不是"已完成""已取消"的工作项（Plane 400） |
| `issue.relation_not_found` | 404 | 关联不存在、已删除，或两端都看不到 |
| `issue.link_not_found` | 404 | 链接不存在、已删除，或它的工作项看不到 |
| `issue.link_exists` | 409 | 同一个工作项上已有这个地址（Plane 400） |
| `issue.comment_not_found` | 404 | 评论不存在、已删除，或它的工作项看不到 |
| `issue.not_author` | 403 | 改别人的评论；不是项目管理员而删别人的评论（3.4，决策点 8） |
| `issue.reaction_exists` | 409 | 已有同一个表情（Plane 400） |
| `issue.reaction_not_found` | 404 | 去掉一个调用者没有加过的表情 |
| `issue.description_version_not_found` | 404 | 版本不存在，或它的工作项看不到 |
| `issue.draft_not_found` | 404 | 草稿不存在、已删除、已发布，或不是调用者的 |
| `issue.draft_changed` | 409 | 草稿的项目在等锁期间被另一个请求改了，重试（3.5） |
| `project.state_in_use` | 409 | 删除有工作项用着的状态（3.6；Plane 400） |

- 取值不合规一律 422 `validation_failed`，字段错误的码取自封闭集合（总体设计 3.5），M4 不新增字段错误码：负责人不合格是 `assignee_ids[i]` 的 `not_allowed`，重复是 `duplicate`；成环的父工作项是 `parent_id` 的 `not_allowed`；空评论是 `comment_html` 的 `required`；草稿没有项目是 `project_id` 的 `required`；列表的查询参数不合格是参数名的 `not_allowed`。
- `apitest` 核对两个方向（M2 设计 3.11）和前缀（M3 设计 11.7）；`issue.yaml` 的操作声明的 `project.*`、`workspace.*` 码由 `issue` 自己的 HTTP 测试经假实现返回过（M3 交接第 3 节）。

### 5.4 限流
M4 不加新的桶：需要登录的操作按凭证计数（`authenticated`，`server/configs/config.yaml` 每分钟 1200、突发 200）。
- 链接：每个链接最多引起页面和图标两次出站请求，抓取在固定数量的 worker 里排队（3.20），出站的速率不随请求的速率增长；队列的长度由 `authenticated` 的额度限制。
- 搜索：三个字符以上走 trigram 索引，少于三个字符的在调用者可读的项目里顺序扫描；每次返回至多 100 个。
- 先锁后判定的拒绝（3.5）：凡判定拒绝的（看不到的 404、不能做的 403）被不加锁的预检挡住，不取锁、不进写闸门；限速按凭证计数挡不住一个有很多 PAT 的账户，所以靠的是预检而不是额度。
- 一个项目上的写（3.5）：`authenticated` 的突发额度（200）比连接池（10）大得多，所以工作项一侧的写在取连接之前经按项目的写闸门，满时 503 `server_busy`；额度管的是一个凭证的速率，闸门管的是一个项目在一个进程里同时占的连接。
- M8 若发现需要单独的桶，按总体设计 3.6 加。

---

## 6. 后端结构

### 6.1 新增和修改的包
| 包 | 内容 | 依赖 |
|---|---|---|
| `internal/shared` | `authorize.go`：`Target.CreatorID`、`Grant.OwnOnly`（3.4）；新 `event.go`（`Event`、`Change`、`EntityRef`、`EventPublisher`、`EventSubscriber`，3.19）；新 `date.go`（`Date`：不带时区的日期，文本形式 `YYYY-MM-DD`，比较和解析；工作项的起止日期在领域、接口、筛选里都用它，P2）；从 `identity/domain` 移来的 `PageSize`（M3 交接第 9 节，P4） | 只有标准库 |
| `modules/issue` | 6.2 | `internal/shared`；适配器另外依赖平台、pgx、River |
| `modules/access` | `Rule.AllowCreator`；判定读 `guest_view_all_features`、算出 `OwnOnly`；创建者的判定（3.4） | `internal/shared` |
| `modules/project` | `Provide` 加 6.5 中 `issue` 要的七个端口的实现；`deletion.go` 的步骤加 `IssueCascade.DeleteProjectIssues`、`DeleteWorkspaceProjects` 的末尾调 `IssueCascade.DeleteWorkspaceIssues`（P1–P8 的过渡），P9 改为删除项目发布 `project.deleted`、两处调用删除（3.6）；`ProjectCascade` 加 `LockWorkspaceProjects`；`EventTypes()`（`project.deleted`）；`delete_state.go` 调 `StateInUse`；`delete_label.go` 调 `DetachLabel`；显示设置的三列，取值经它声明的 `IssueListSettings` 检查（3.17）；`Purger`（3.21） | 同 M3 |
| `modules/workspace` | `delete_workspace.go`：先锁邀请和项目、再读时刻（3.5），P9 起发布 `workspace.deleted`；`EventTypes()`；`ProjectCascade` 端口加 `LockWorkspaceProjects`；`Purger`。`issue` 用的目录就是已有的 `WorkspaceDirectory`（`WorkspaceBySlug`、`ShareWorkspaceByID`，`workspace/module.go:33-49`），不加方法 | 同 M3 |
| `modules/identity` | `PageSize` 移出；`Purger`（`api_tokens`）；`PublicProfiles` 的读：负责人名的目录要它，端口由 `issue` 声明（3.3，P5） | 同 M2 |
| `platform/jobs` | `NewInserter(pool)`：只投递的客户端和它的 `InsertTx`；两个队列和它们的 worker 数（配置）；定时任务的种类和声明会投递的种类没有 worker 时 `New` 失败；测试的余量改成期限（3.20） | 同 M2 |
| `platform/postgres` | `Tx(ctx) (pgx.Tx, bool)`（3.20）；`NewTxManager` 的两个选项：`LockTimeout(d)`（`BEGIN` 之后由平台发 `SET LOCAL lock_timeout`，P1b）、`ReadOnlySnapshot()`（`BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY`，经 pgx 的 `TxOptions`，P5），模块的 SQL 里没有 `SET`（`TestModulesRunSQLOnlyThroughSQLC`）；`LockNotAvailable(err)`（SQLSTATE `55P03`，与 `UniqueViolation` 同一种写法，`unique.go`）；连接池在每个连接上设 `jit = off`、`plan_cache_mode = force_custom_plan`（3.16，P5） | 同 M2 |
| `platform/clock` | `Offset{Base, Offset}`（3.20） | 只有标准库 |
| `platform/outbound` | 新：防 SSRF 的 HTTP 客户端（6.8） | 标准库 |
| `platform/gate` | 新：带 `context` 的按键的信号量（`Keyed[K comparable]`：每个键至多 K 个持有者、等待者至多若干个、等待有上限），工作项的写闸门用它（3.5，架构问题 11，P1b） | 标准库 |
| `platform/config` | `jobs.default_workers`、`jobs.periodic_workers`；`clock.offset`（只在 `env = test`）；`outbound.allow_cidrs`（只在 `env = test`）；`issue.auto_archive_interval`、`issue.trim_versions_interval`、`issue.delete_batch`、`issue.delete_budget`；`issue.write_gate.per_project`、`issue.write_gate.max_wait`、`issue.write_gate.max_queue`、`issue.write_lock_timeout`（3.5）；`cleanup.purge_after`、`cleanup.purge_interval`、`cleanup.purge_batch` | — |
| `platform/httpserver/bodyshape` 与 `tools/bodyshapegen` | `FormatDate` 的检查器（`shared.Date` 的规格）；字段错误的路径按下标的数值排序；不限类型的节点检查数的范围；代价测试加上字符串的数组（M2 交接第 4 节，P2） | — |
| `bootstrap` | 接线（6.6）；`events.go`（分发器，3.19）；`jobs.go`（投递端口的接合：`postgres.Tx` 加只投递的客户端的 `InsertTx`；建工作的客户端时核对各模块 `EnqueuedKinds()` 的种类都有 worker）；`gate.go`（`platform/gate` 接成 `issue` 的 `WriteGate`；给 `issue` 的写用例一个带 `LockTimeout` 的 `TxManager`，并把它的 `postgres.LockNotAvailable` 换成 `shared.ServerBusy`：闸门的 503 和锁的期限的 503 都在这一个文件里，3.5）；`purge.go`（一个清理任务，3.21）；`ports.go` 的转换；测试：操作名的完整性、权限矩阵、交错、取锁的形状、清理的目录 | 全部 |
| `internal/archtest` | `issue` 在现有规则下检查；组合测试：命令行的组合不到达 `issue.New`、River 的 `InsertTx`（P1–P8 它为 `project.NewCascade` 拿 `issue.Provide` 交出的 `IssueCascade`，6.6） | — |
| `deploy/runtime-grants.sql` | 13 张新表的读写（4.18） | — |
| `server/configs/config.yaml`、`config.test.yaml` | 上面的配置项；测试配置的间隔都是 `2s` | — |

- **新的依赖只有一个**：`github.com/microcosm-cc/bluemonday`（HTML 的清洗，总体设计 6.9 早已写明；`server/go.mod` 今天没有它），连同它带来的 `golang.org/x/net`、`github.com/aymerick/douceur`、`github.com/gorilla/css`；P4 随描述和评论加入，`archtest/binary_test.go` 的禁用清单不受影响。HTML 的解析（取 `<title>`、图标、提及）用 `golang.org/x/net/html`。River、pgx 已有。

### 6.2 `issue` 模块的结构
```
modules/issue/
  domain/
    issue.go                Issue；标题、优先级、日期、负责人和标签的集合的规则（3.7）；完成时间、排序值的计算
    hierarchy.go            父子：不是自己、不成环（输入是祖先链，纯函数，3.9）
    relation.go             关联的四种类型、存储的方向和反向（3.10）
    link.go                 地址的规整和检查（3.11）
    comment.go              评论为空的判断（3.12）
    reaction.go             表情的格式（3.12）
    activity.go             动态的行：由一次修改的前后两个值算出 3.13 的行（纯函数）；描述的合并的判断
    versions.go             描述版本的合并的判断（600 秒、同一个人，3.14）
    rich_text.go            清洗结果和内容标识的纯值；只补块 id 的保存是否"内容不变"（纯文本、按顺序的提及、链接地址、勾选框状态）；提及的集合规则（3.7、3.12），不解析 HTML、不依赖第三方
    identifier.go           按编号取和搜索的标识折叠（3.7）
    draft.go                草稿：载荷的项目、发布的检查（3.15）
    listvocab.go            列表的词汇：参数名和取值的语法、分组码、排序码、各页面允许的集合；IssueListSettings 的实现（3.17）
    list.go                 列表的请求：解析好的筛选、分组、group_key、排序；游标的载荷（3.16）
    events.go               事件的类型和 EventTypes()（3.19）
    actions.go              本模块的操作名和 Actions()（3.4）
    errors.go               本模块的错误码（5.3）
  app/
    ports.go                仓储（按用例拆成小接口）、Clock、EventPublisher、ProjectGate、Sequences、StateCatalog、LabelCatalog、ProjectMembers、VisibleProjects、ArchivedProjects、ArchivePolicies、WorkspaceDirectory、PublicProfiles、WriteGate、LinkCrawls、RowDeletions、HTMLSanitizer（消费方的小接口，返回领域纯值）
    lock.go                 工作项一侧的写共用的一步：读资源 → 预检 → 写闸门 → 项目 N → 重读 → 判定 → 访客规则（3.5）
    create_issue.go         创建、快速添加、发布草稿共用的核心（createCore），和 createIssue 的入口
    get_issue.go  get_issue_by_identifier.go  update_issue.go  delete_issue.go  archive_issue.go
    issue_writes.go         修改共用的写：负责人、标签的差集，关注，提及，动态，版本，事件（update、create、addSubIssues 都用它）
    list_issues.go  list_member_issues.go  search_issues.go
    sub_issues.go           listSubIssues、addSubIssues
    relations.go            list、create、delete
    links.go                list、create、update、delete；crawl_link.go：抓取的写回（worker 调用）
    comments.go  reactions.go  activities.go  description_versions.go
    drafts.go               list、create、get、update、delete；publish_draft.go
    cascade.go              project 的 IssueCascade 的实现：StateInUse、DetachLabel（P1–P8 另有 DeleteProjectIssues、DeleteWorkspaceIssues，3.3）
    delete_project_rows.go 删除项目、工作区之后工作项一侧的行的编排：表的顺序、按 `id` 的键集、关联行属于先删除的那一端（经 `ProjectGate.DeletedAt`，P9 起）、时间的预算；语句经仓储的端口在 `adapter/postgres/delete_rows.go`（3.6；P1–P8 的同步连带也调它）
    auto_archive.go         自动归档（3.8）；trim_versions.go（3.14）
    crawl_on_link_change.go 链接的订阅者；delete_rows_on_deletion.go 删除的订阅者（3.19）
    purge.go                本模块的 Purger（3.21）
  adapter/
    content/                HTMLSanitizer 的实现；bluemonday 的唯一放行表、StrictPolicy 纯文本、x/net/html 解析与内容投影；描述和评论共用（3.7、3.12）
    postgres/               仓储；queries/*.sql（按表、按用例分文件，列表引擎三个文件）；gen/（sqlc）；delete_rows.go：删除的任务每张表的一批（`queries/delete_rows.sql`，键集），清理的批同一种写法
    http/                   handler，按资源分文件；查询参数到 domain/list.go 的转换；gen/
    river/                  任务的参数和 worker：crawl_link、delete_rows（default 队列）、auto_archive、trim_versions（periodic 队列）；LinkCrawls、RowDeletions 的实现（经投递端口）
    crawler/                抓取：经 platform/outbound 取页面和图标、解析 title 和 link（3.11）
  module.go                 Provide(pool)（IssueCascade、IssueListSettings、Purger）；New(Deps)；Register；Actions()；EventTypes()；Subscribers()；Jobs()；EnqueuedKinds()
```
- 一个用例一个文件或一组紧密的用例一个文件（总体设计 6.1），每个不超过约 400 行；`update_issue.go` 只编排，写的细节在 `issue_writes.go`。
- `cascade.go` 只用本模块的仓储，删除者和时刻由调用方传入，事务取自调用方的 `ctx`（M3 的 `ProjectCascade` 的同一写法）。它在 `Provide` 里建（只依赖连接池），所以 `project.New` 能在 `issue.New` 之前或之后拿到它：组合的顺序由别的端口决定（6.6）。
- 删除的语句（`adapter/postgres/delete_rows.go`）和编排（`app/delete_project_rows.go`）在 P1a 写成：同步的过渡里由 `cascade.go` 在发起者的事务里调（循环到 0 行，不看预算），P9 由任务的 worker 调（每批一个事务，带预算）；同一组语句、同一份编排，不是两份。SQL 只在 `adapter/postgres`，`app` 经仓储的端口调它（`TestModulesRunSQLOnlyThroughSQLC`）。
- `lock.go` 是 `issue/app` 私有的（M3 的 `project/app/lock.go` 也是私有的）；项目行的锁本身经 `ProjectGate` 端口由 `project` 实现。

### 6.3 别的模块的改动
- **`project`**：
  - `Provide` 的新端口（6.5）都是 `project/adapter/postgres` 里的查询，只依赖连接池：`ProjectGate`（不加锁的读、`FOR NO KEY UPDATE` 的锁、按 `id` 升序锁几个项目）、`Sequences`、`StateCatalog`、`LabelCatalog`、`ProjectMembers`、`VisibleProjects`、`ArchivedProjects`（P3）、`ArchivePolicies`。锁项目行的那条查询与 M3 的 `lockRowAndDecide` 用的是同一张表的同一种锁，但 M3 的那一个先锁工作区（方案 E），这一个不锁（3.5）。
  - `project.New` 的 `Deps` 加 `IssueCascade`、`IssueListSettings`（都由 `issue.Provide` 交出），P9 起加 `EventPublisher`（删除项目发布 `project.deleted`）。`ProjectCascade` 加 `LockWorkspaceProjects`（删除工作区用，3.5），它的查询与 `ProjectGate.LockMany` 同一种写法。`ProjectGate` 另有两个不加锁的读：这个工作区未删除的项目的 id（草稿的读按它过滤，3.15）；`DeletedAt(projectIDs)`（P9 起），其中已删除的项目的 `deleted_at`（删除工作项和删除的任务据它跳过属于先删除的那一端的关联行，3.5）。`deleteWorkspace` 发布的 `workspace.deleted` 带 `LockWorkspaceProjects` 锁住的项目（3.19）。
  - `NewCascade(CascadeDeps)` 在 P1–P8 要 `IssueCascade`（`DeleteWorkspaceProjects` 的末尾调它）；命令行的组合（`bootstrap/users.go:45-46`，`nerve users deactivate` 只用 `EndMemberships`）为它传 `issue.Provide(pool)` 交出的 `IssueCascade`，不构造 `issue.New`。P9 起 `DeleteWorkspaceProjects` 不再调 `issue`（删除工作区的事件由 `workspace` 的用例发布），这一项随之从 `CascadeDeps` 删除（对抗性评审 M9）。
  - `access.sql` 的 `ProjectFacts` 加 `guest_view_all_features`（3.4）。
- **`workspace`**：删除工作区的用例改顺序（3.5），P9 起在事务里发布 `workspace.deleted`（`Deps` 加 `EventPublisher`）；`ProjectCascade` 端口加 `LockWorkspaceProjects`。
- **`access`**：6.1 的一行；规则表加 M4 的 13 行（3.4）；`decide.go` 的输入加创建者、标志；判定表测试加创建者和访客规则的每一种。
- **`identity`**：`PageSize` 移出（P4）；`Purger`（P9）。

### 6.4 事件、任务和清理的接缝
- **发布**：`issue/app` 的用例（P9 起还有 `project` 的删除项目、`workspace` 的删除工作区）在事务里调 `EventPublisher.Publish`，在全部写之后、事务提交之前。用例不知道谁订阅。
- **分发**：`bootstrap/events.go` 的 `Dispatcher` 实现 `shared.EventPublisher`，由 `issue.Subscribers()`（以后各模块的）建出，构造时检查类型（各模块的 `EventTypes()`：`issue`、P9 起 `project`、`workspace`）和名字（3.19），按登记的顺序同步调用；P6 起每个用例的发布点由用例测试经假的发布者核对（类型、`Entity`、`Changes`）。
- **投递**：订阅者要投递任务时经一个端口（`issue/app` 的 `LinkCrawls`、P9 的 `RowDeletions`），实现在 `issue/adapter/river`，它要一个 `Inserter`（`InsertTx(ctx, args, opts) error`），由 `bootstrap/jobs.go` 用 `postgres.Tx(ctx)` 加只投递的客户端（`jobs.NewInserter(pool)`，3.20）接成；不在事务里时返回错误。模块经 `EnqueuedKinds()` 声明它会投递的种类，建工作的客户端时逐个核对有 worker。
- **任务**：模块经 `Jobs()` 交出 `jobs.Job`（M2 的写法，`identity/adapter/river/cleanup.go`），worker 只调用例；`bootstrap/app.go` 把各模块的 `Jobs()` 和 `purge.go` 的清理任务交给 `jobs.New`（今天只传 `ident.Jobs()`，`app.go:158`）。
- **清理**：四个模块各交出一个 `Purger`（`Purge(ctx, cutoff, batch) (int, error)`，删完为止），`bootstrap/purge.go` 把它们按 3.21 的顺序组成一个任务。`Purger` 的接口声明在 `bootstrap`（消费方），模块只交出满足它的值，不互相导入。
- **测试的时钟**：`bootstrap/app.go` 按配置建出 `clock.Offset` 或 `clock.System`，交给每个模块的 `Deps.Clock`（3.20）。

### 6.5 端口
| 端口 | 声明在 | 实现 | 测试 |
|---|---|---|---|
| `Authorizer`（`Target.CreatorID`、`Grant.OwnOnly`） | `internal/shared` | `access/app` | `access/domain` 的判定表（创建者、访客规则的每一种）；9.2 的矩阵 |
| `ProjectGate`（`Find(projectID)` 不加锁 → 工作区；`Lock(projectID)` `FOR NO KEY UPDATE`，带 `deleted_at IS NULL`，回答 `archived`、`identifier`、`default_assignee_id`、`archive_in`，不回答 `guest_view_all_features`（判定经 `Authorize` 读它：锁下的写、P3 的两个列表；P5 的列表引擎经 `VisibleProjects`）；`LockMany(projectIDs)` 一条语句按 `id` 升序；不加锁的读：工作区未删除的项目的 id，`DeletedAt(projectIDs)`（P9 起）） | `issue/app` | `project/adapter/postgres`（`project.Provide`） | 集成测试；取锁的形状测试（交错 21：每个写的第一把锁是项目的 N，跨项目的写经 `LockMany`）；`LockMany` 的 `id` 顺序和等锁期间被删除的项目（`project` 的存储测试，S5 的 E4）；看不到的项目不锁（`TestARelationLocksNoProjectItDoesNotReach`，交错 21 的关联的一例） |
| `Sequences`（`Next(projectID) (int, error)`：`UPDATE … RETURNING`，在项目 N 之下） | `issue/app` | 同上 | 集成测试；交错 1 |
| `StateCatalog`（项目的未删除状态：id、名字、组、默认、分诊；名字的比较键） | `issue/app` | 同上 | 集成测试 |
| `LabelCatalog`（项目的未删除标签：id、名字的比较键；按 id 集合核对） | `issue/app` | 同上 | 同上 |
| `ProjectMembers`（一组账户在项目里的有效成员关系和角色） | `issue/app` | 同上 | 同上；交错 4、5、22 |
| `VisibleProjects`（调用者在一个工作区里是有效成员的未归档项目、角色、`OwnOnly`；另一个方法给动态的过滤：`is_active` 的成员关系所在的全部项目，不看 `deleted_at` 和归档，3.13） | `issue/app` | 同上 | 访客规则的一致性测试（3.4）；动态引用已删除、已删除项目里、已清理的工作项的三例 |
| `ArchivedProjects`（`Archived(projectIDs)`：给的项目中未删除、已归档的，不加锁；P3 的关联列表读它，3.4、3.10） | `issue/app` | 同 `ProjectGate`（`project` 的 `ArchivedIssueProjects`） | 存储测试；组合的转换测试；关联列表的组合测试（别的已归档的项目的对方不在其中） |
| `PublicProfiles`（按 id 批量读显示名） | `issue/app` | `identity` 的读（`bootstrap/ports.go` 接成） | 负责人名的目录含已离开、已降为访客的负责人（3.16） |
| `WriteGate`（`Enter(ctx, projectIDs) (release, error)`；P1b 一次一个项目（`Enter(ctx, projectID) (leave, error)`），P3 加多个（P1b spec 第 3 节第 10 条）） | `issue/app` | `bootstrap/gate.go` 接 `platform/gate` | `platform/gate` 的单元测试；交错 20 经真实的连接池；满时 503 |
| `ArchivePolicies`（设了 `archive_in`、未归档、未删除的项目） | `issue/app` | 同上 | 自动归档的集成测试 |
| `WorkspaceDirectory`（已有的 `WorkspaceBySlug`；`ShareWorkspaceByID` 以 `FOR SHARE` 锁工作区行） | `issue/app`（照 `workspace/module.go:33-49` 的方法声明自己要的小接口） | `workspace.Provide` 交给 `project` 的同一个目录 | 已有的集成测试；交错 16、17 |
| `IssueCascade`（`StateInUse`、`DetachLabel`；P1–P8 另有 `DeleteProjectIssues`、`DeleteWorkspaceIssues`，P9 删除） | `project/app` | `issue/app` 的 `cascade.go`（`issue.Provide`） | 用例测试；删除状态、标签的集成测试核对连带与发起者在同一事务、同一时刻；交错 6、7（P1–P8 另有 2、3、11、16、23 的同步版本） |
| `IssueListSettings`（`Validate(filters, displayFilters, displayProperties) []FieldError`） | `project/app` | `issue/domain` 的 `listvocab.go`（`issue.Provide` 交出，纯函数） | `issue/domain` 的表测试；`updateProjectPreferences` 的 422 经它 |
| `ProjectCascade.LockWorkspaceProjects` | `workspace/app` | `project/app`（委托 `ProjectGate` 的同一条查询） | 交错 3、20 |
| `EventPublisher` | `internal/shared` | `bootstrap/events.go` | 分发器的单元测试；用例测试用假的发布者 |
| `LinkCrawls`（`Enqueue(ctx, linkID, url)`） | `issue/app` | `issue/adapter/river`（经 `bootstrap` 用只投递的客户端接成的 `Inserter`） | 集成测试：提交的事务留下 `issue.crawl_link`，回滚的不留（S4 已在 River v0.47.0 上验证） |
| `RowDeletions`（`Enqueue(ctx, scope, id, at, by)`，`scope` 是 `project` 或 `workspace_drafts`，P9） | `issue/app` | 同上 | 集成测试：删除项目、工作区提交时留下每个删除的项目一个 `issue.delete_rows`（工作区另加草稿的一个），回滚的不留；任务做完之后每一行是删除它的那次请求的时刻和删除者；交错 2、3、11、16、23、24、25；窗口期间的读与任务之后相同（3.6） |
| `Purger` | `bootstrap` | 四个模块 | 清理的集成测试（3.21、9.3） |

- **跨边界的值**：端口的参数和结果只用标准库、`shared` 的类型，或调用方 `app` 层声明的小值类型（`LockedProject`、`StateRef`、`LabelRef`、`MemberRole`、`VisibleProject`）；转换在 `bootstrap/ports.go`（M3 设计 6.5 的写法）。
- **每个端口只有一种真实的实现**；用例测试用手写的假实现。

### 6.6 组合：没有构造的环
`bootstrap` 按下面的顺序接线（M3 设计 6.6 的延伸）：
1. **载入签名密钥**（M3 照旧）。
2. **适配器**：`identity.Provide(pool)`、`workspace.Provide(pool)`、`project.Provide(pool)`（加 6.5 的七个端口）、`issue.Provide(pool)`（`IssueCascade`、`IssueListSettings`、`Purger`）。都只依赖连接池。
3. **平台**：时钟（3.20）；只投递的 River 客户端 `jobs.NewInserter(pool)`，和由它加 `postgres.Tx` 接成的 `Inserter`（3.20）；`outbound` 的客户端；写闸门（`platform/gate`，按配置的 K、等待的上限，3.5）。
4. `access.New`（`WorkspaceRoles`、`ProjectAccess`）→ `Authorizer`。
5. **分发器**：`bootstrap/events.go` 由 `issue.Subscribers()` 建出。订阅者是 `issue` 模块的函数，它要的 `LinkCrawls`、`RowDeletions` 由 `issue/adapter/river` 在 `issue.Subscribers(Inserter)` 里建出，不要 `issue.New` 的东西，所以分发器在 `issue.New`、`project.New`、`workspace.New` 之前建出、交给它们。
6. `issue.New`（`Authorizer`、分发器、时钟、`project` 的七个端口、`WorkspaceDirectory`、`PublicProfiles`、写闸门、`Inserter`、`outbound` 的客户端；两个 `shared.TxManager`：工作项一侧的写用的带锁的期限、`55P03` 换成 503（`bootstrap/gate.go`），列表的读用的只读快照（`postgres.ReadOnlySnapshot()`，3.16））。
7. `project.New`（`Authorizer`、`WorkspaceDirectory`、`WorkspaceMembers`、`IssueCascade`、`IssueListSettings`；P9 起分发器）→ `Cascade()`。
8. `workspace.New`（M3 照旧，`ProjectCascade` 多一个方法；P9 起分发器）→ `Deactivator()`、`SignupInvitations()`。
9. `identity.New`（M3 照旧）。
10. **任务**：`jobs.New(pool, cfg, identity.Jobs() + issue.Jobs() + purge.Job(四个 Purger), 各模块的 EnqueuedKinds())` 建工作的客户端，照 M2 的写法一次登记全部 worker 和定时任务，核对定时的种类和声明会投递的种类都有 worker（缺了就构造失败），然后 `Start`。它不交给任何模块。

- **S4 在 River v0.47.0 上的事实**（0.3）：只投递的客户端（没有 `Workers`）不检查种类，事务里的 `InsertTx` 回滚不留、提交留下；工作的客户端拒绝没有登记的种类（`UnknownJobKindError`，`river@v0.47.0/client.go:2270-2277`），`NewClient` 之后 `AddWorker` 可用（共用 `*Workers` 指针，没有写进文档）、`Start` 之前 `PeriodicJobs().Add` 和 `InsertTx` 都可用；一个 worker 都没有时 `Start` 失败。第一稿的做法是先建工作的客户端交给投递端口、最后再登记 worker（`NewRunner`、`Register` 两步），它依赖那个没有写进文档的行为，而且投递必须在登记完成之后：请求可能在服务开始接收之前就到达。改用只投递的客户端，这两点都不存在，投递端口在第 3 步就完整，第 10 步与 M2 的写法相同。
- **只投递的客户端的代价**是它不替我们检查种类：漏登记的种类会安静地留在 `river_job`。所以第 10 步的核对和它的变异测试是必需的（3.20）。

- 没有构造的环：`issue.New` 不要 `project.New`、`workspace.New` 的东西；`project.New` 要的 `IssueCascade` 来自 `issue.Provide`。
- **命令行的组合**（M3 设计 6.6）：`nerve users`、`nerve workspaces` 不构造 `issue.New`、分发器、`jobs`。停用（`nerve users deactivate`）经 `EndMemberships` 改项目成员关系，不碰工作项的表；但 `project.NewCascade(project.CascadeDeps{Pool})`（`bootstrap/users.go:45-46`）在 P1–P8 要 `IssueCascade`（`DeleteWorkspaceProjects` 调它），所以命令行在这段时间为它传 `issue.Provide(pool)` 交出的 `IssueCascade`（只是查询对象，不构造 `issue.New`）；P9 起 `DeleteWorkspaceProjects` 不再调 `issue`，这一项删除（对抗性评审 M9，6.3）。`archtest/composition_test.go` 加上"不到达 `issue.New`、`InsertTx`"。M4 没有投递任务的命令（M2 交接第 1 节，13.2）。

### 6.7 一个写请求：`PATCH /api/v0/issues/{issue_id}` 改状态和标签
```
请求 → …（M2 设计 6.4 的固定链、路由、参数绑定、认证、限流、请求体结构：未知字段、类型、日期的格式 → 400）
     → handler：RequireActor；IssueUpdate 转成用例的入参（nullable 区分"没传"和"清空"）
     → UpdateIssue.Execute(ctx, actor, issueID, in)：
         0. 值的校验（事务之前，只看请求）：标题去空白后 1–255；skip_activity 只与描述同在；
            label_ids 不重复（422 label_ids[i] duplicate）
         1. issues.Find(issueID)（事务之外，用完就还回连接）：未删除 → project_id、created_by_id；没有 → 404 issue.not_found
            预检（不加锁）：Authorizer.Authorize(…, "issue.update", …) 加访客规则：
              看不到、访客规则挡住 → 404 issue.not_found；看得到而不能做 → 403 forbidden
              都不取任何锁、不进闸门                                      （3.5：凡拒绝的都在这里答）
         2. WriteGate.Enter(ctx, [project_id])：这个项目在本进程里已有 K 个写在数据库里时在进程里等，
            不持连接；等过 max_wait、队列满或请求到期 → 503 server_busy（Retry-After: 1）（3.5）
       TxManager.WithinTx {                                                  （带 LockTimeout 的 TxManager：BEGIN 之后平台发 SET LOCAL lock_timeout）
         3. ProjectGate.Lock(project_id)                                   （3.5：最先锁项目行，不锁工作区）
              SELECT … FROM projects WHERE id = $1 AND deleted_at IS NULL FOR NO KEY UPDATE
              → 0 行：404 issue.not_found；等锁超过 lock_timeout → 503 server_busy
         4. issues.Get(issueID) 重读：等锁期间被删除 → 404 issue.not_found
         5. Authorizer.Authorize(ctx, actor, "issue.update",
                                  Target{WorkspaceID, ProjectID, CreatorID: issue.CreatedByID})
              → 看不到：ErrNotVisible → 404 issue.not_found
              → Grant.OwnOnly 且 CreatedByID ≠ actor：404 issue.not_found      （访客规则，3.4）
              → 看得到、角色不是管理员或成员：403 forbidden
         6. 项目已归档 → 409 project.archived；工作项已归档 → 409 issue.archived
         7. 依赖行的检查（项目 N 之下稳定）：
              StateCatalog：state_id 是这个项目未删除、不是分诊的状态 → 否则 422 state_id not_allowed
              LabelCatalog：每个 label_id 是这个项目未删除的标签 → 否则 422 label_ids[i] not_allowed
              合并后的起止日期（这次没改，跳过）
         8. 锁下读时刻 now
         9. issues.Update：state_id、completed_at（新状态在"已完成"组为 now，否则空）、updated_at = now、updated_by_id
            issue_labels：差集 → 插入新增的（按 label_id 排序）、软删除去掉的
        10. 动态（domain/activity.go 由前后两个值算出）：state 一行（旧名、新名、旧 id、新 id）；每个加、去的标签一行
        11. EventPublisher.Publish(issue.updated, Changes = [state, labels+, labels-])
              → 分发器：没有订阅 issue.updated 的订阅者（M4），什么都不做
       }
       release()：离开闸门
     → 200 IssueDetail（重读一次，带完整的 label_ids、assignee_ids、计数）
```
- **要点**：
  - 锁在判定之前（M3 约定二）；工作项的写最先锁项目行、不锁工作区行（3.5，决策点 1）。锁之前的预检答一切拒绝，放行的那一个由锁下的第 5 步权威地再判定一次。
  - 闸门在开始事务（取连接）之前进、在提交之后出：在项目上排队的写不占连接池（3.5，架构问题 11）。
  - 资源先读、锁、再重读：工作项的项目不会变（不能移动），重读只为确认它仍在。
  - 访客规则在判定之后、作为"看不到"处理：访客改别人的工作项与改不存在的工作项得到同一个 404（8.4）。
  - 值的校验在事务之前，依赖行的检查在判定之后（M3 约定二）：不存在的标签 id 对没有权限的人不透露任何东西。
  - 动态、关注、提及、版本、事件都在同一个事务里（总体设计 6.6）；事件的订阅者出错时整个修改回滚。
  - 响应重读：差集写完之后，完整的数组和计数由同一条查询给出，与列表的行一致。

### 6.8 `platform/outbound`：防 SSRF 的出站客户端（架构问题 8）
- **接口**：`Client.Get(ctx, url, limits) (Response, error)`：`limits` 给出期限、最多读的字节数；`Response` 有最终的地址、状态、类型、读到的字节（不超过上限）。错误分成几类（`Blocked`、`Timeout`、`Unreachable`、`TooLarge`），抓取把 `Blocked` 写成 `metadata.error = blocked`，其余都写成 `failed`，细分只进日志（3.11）。
- **规则**（照 Plane 的 `API/utils/url_security.py`、`ip_address.py`，补上它缺的上限）：
  - 只许 `http`、`https`；不带用户信息（`user@host`）；端口不限（Plane 也不限）。
  - 解析主机名，**任何一个**解析出的地址落在拒绝的表里就拒绝，并递归检查 IPv6 里嵌着的 IPv4（IPv4 映射的 IPv6、NAT64、6to4、Teredo，`ip_address.py:16-95,105-156`）。
  - **拒绝的表写成显式的 IANA 特殊用途登记表**（IPv4 和 IPv6 两张，`iana-ipv4-special-registry`、`iana-ipv6-special-registry`），不靠 `net/netip` 的判断：`netip` 没有"保留"的判断，`IsPrivate` 比 Python 的 `is_private`（Plane 靠它）窄得多。SP3 量到第二稿的"`netip` 的判断加一张小表"放过了 `192.0.0.192`（Oracle Cloud 旧的元数据地址）、`192.0.0.0/24`、`198.18.0.0/15`、`240.0.0.0/4`、三个文档网段、`192.88.99.0/24`、`100::/64`、`2001:db8::/32` 和 Plane 自己列着的 `64:ff9b:1::/48`（对抗性评审 M3）。表里另有 `0.0.0.0/8`、`100.64.0.0/10`、`169.254.0.0/16`、`255.255.255.255/32`、`fec0::/10`（已废弃的站点本地）和组播；登记表里"全局可达"一栏为真的网段也拒绝（例如 `192.0.0.9/32`），出站只许普通的公网地址。
  - **拒绝本机网卡上的地址**：启动时读出本机各网卡上的地址（`net.InterfaceAddrs`）加进拒绝的表。它只覆盖启动那一刻在本机网卡上的地址：容器里和多数云上，公网地址不在本机的网卡上（在 NAT 或负载均衡上），网卡的地址也可能在启动之后变；这些由网络层的出站规则覆盖（README，8.8；聚焦复核 M9）。
  - **连接到检查过的那个地址**：自己的 `DialContext` 只拨检查过的 IP，Host 头和 TLS 的 SNI 仍是主机名，挡住 DNS 重绑定（`url_security.py:160-272`）。
  - **每一次跳转重新检查**，至多 5 次；不跟随到非 `http(s)`。
  - 不读环境变量里的代理（`Transport.Proxy = nil`）。
  - 期限：连接 2 秒、整体 5 秒（由调用方的 `limits` 给，抓取用这两个值）；读到上限就停，返回 `TooLarge`。
  - 不发 cookie，不带凭证；`User-Agent` 是 `Nerve/<版本> (+link preview)`。
- **测试的出口**：`outbound.allow_cidrs`（只在 `env = test`，3.11）把列出的网段从拒绝的表里拿掉；生产配置里不为空时启动失败。
- **测试**：登记表的每个网段一行（两端的地址各一个，含 IPv6 的嵌入形式），本机的接口地址一行；DNS 重绑定（第一次解析公网、第二次解析回环，客户端只拨第一次检查过的）；跳转到内网被拒绝；超过 5 次；超时；超过上限；代理的环境变量不生效。用 `httptest` 和自定义的解析器，不访问外网。
- **M8 复用它**（Webhook 的投送），所以在 `platform`，不在 `issue`（13.2）。

---

## 7. 前端

前端先做数据层（P10a 换型和删除 M6、M7 的工作项列表，P10b 列表一侧，P11 详情一侧），再做页面（P12 列表与看板、P13 筛选、归档与个人主页、P14 详情、P15 协作、草稿与搜索），最后清理（P16）：三段数据层把 M4 的全部类型、service、store 和挂载时的取数迁到新接口，页面的 Phase 加入故事时挂载路径上已经没有旧接口的请求（3.1、第 12 节）。M3/P8 量到它较小的类型集换型时有 200 个 `tsc` 错误、因此拆成 P8a、P8b（R6 第 9 节）；M4 的 `TIssue` 有 131 个使用方（`git grep -lw TIssue -- web`）、`EIssuesStoreType` 93 个。第二稿按"列表一侧、详情一侧"分两段换型，编译不过：列表和详情读的是同一份工作项（`store/issue/issue.store.ts` 的 `issuesMap`），它的类型一换，全部使用方要在同一段里改（对抗性评审 I7），所以换型单独成段（P10a，7.2）。

### 7.1 按会话分代（总体设计 7.7；M2 交接第 12 节；M3 交接第 6 节）
- **service 按代建**：M4 的 service 都是生成的客户端之上的薄封装，构造时接收这一代的 `ApiClient`，由 store 建（M3 的写法）。M2 交接列出的 13 处模块级实例全部消失（R6 第 4 节），各在改它所在的代码的 Phase，`git grep` 的门禁在最后一处所在的 P15：
  - P11（数据层）：`WorkItemVersionService` 2、`WorkspaceDraftService` 1；`WorkspaceService.searchEntity` 4（编辑器的 @提及，改为前端从成员列表过滤，3.12；收集箱的一处是 M7 的组件，同样改）。
  - P13：`IssueService` 1（已归档的详情的页头，7.4）。
  - P12：`ProjectService.projectIssuesSearch` 1：添加已有工作项的弹窗（`existing-issues-list-modal.tsx`）随日历的"添加已有"改接 `issue-search`（7.5）。
  - P14：`IssueService` 1（`issues/[issueId]` 的跳转，7.4）；`ProjectService.projectIssuesSearch` 2：选父工作项（`parent-issues-list-modal.tsx`）随 I8 的页面版本改接 `issue-search`（7.5），收集箱的查重（M7 的组件）随关联的区改接、改到能编译、行为不变。
  - P15：`WorkspaceService.searchWorkspace` 1（命令面板的搜索，3.18）；P15 的完成线是 `git grep` 不再找到 M4 的这 13 处。
- **`IssueRootStore` 的 `autorun`**（`store/issue/root.store.ts:157`，在 action 之外写 `router` 的副本）：改为 `rootStore.router` 上的计算属性，没有要释放的反应；`store-context.test.ts` 的两条 MobX 严格模式警告随之消失（M3 交接第 6 节，P10b）。
- **`rootStore` 的三处同步读取**（`issue-layouts/utils.tsx`、`command-palette.store.ts:94`、`store/issue/issue.store.ts:66`）：接上新接口之后仍只同步读、不发请求；要发请求的改经自己的 `RootStore`（M2 交接第 12 节）。
- **SWR 键**：M4 的文件里直接从 `swr` 导入的 15 处（R6 第 8 节）全部改经 `useSessionSWR`，键是 `[取数的名称, loginId, ...参数]`（M3 设计 7.1）；`.oxlintrc.json` 的路径限制扩到 M4 的目录。
- **修改一个接一个**：同一个工作项的修改经 `oneAtATime()` 排队（拖动、连续改属性、描述的保存），只用确认的结果写进每一份（3.22）；Plane 的乐观更新和回滚删除。拖动的 `sort_order` 依赖相邻的卡片，在这次修改轮到发出时才按那时的列表算（总体设计 7.7，3.22）。
- **组件在 `await` 之后的跟进经 `followInSession`**（M3 设计 7.1）：创建之后的提示和"查看"按钮（`create-issue-toast-action-items.tsx`）、删除之后的跳转、发布草稿之后的跳转、归档之后的跳转。I22 的页面版本覆盖详情页的修改。
- **权限决定取数**：访客的个人主页不取工作项（M3 决策点 3，I19）；已归档的列表、描述的历史、草稿只在调用者是项目（工作区）的管理员、成员时取；不是项目成员的人打开项目的工作项页不取列表（M3 的 `ProjectAuthWrapper` 已按 `member_role` 启用）。
- **按需的取数被拒绝时**（M3 交接第 6 节）：状态、成员、标签的下拉框打开时取，子工作项的 store 取别的项目的状态、成员、标签（调用者可能不是那些项目的成员）。被拒绝时下拉框显示错误、结束加载，不留未处理的拒绝；工作项 store 不等结果的取数（`issue.store.ts` 取状态的两处、`sub_issues.store.ts` 的三处）改为等待并交回调用方。跨项目的状态由工作区的状态列表（M3 的 `listWorkspaceStates`）给出，不再按项目逐个取。vitest 覆盖每一处的拒绝。
- **界面状态在以 id 为 key 的布局之下**（M3 交接第 6 节，负责人 2026-10-13 的 B）：
  - `WorkItemLevelModals`（创建、删除工作项的开关在命令面板的 store）的宿主**留在原处**（`power-k/projects-app-provider.tsx:93`，在 `(projects)/layout.tsx` 之下）：它以 `browse/[workItem]` 的 `workItemIdentifier` 为参数（`:32`），而 `browse/` 与 `projects/` 是同级的路由，不在 `[projectId]/layout.tsx` 之下；新建工作项又是工作区一级的命令（首页、草稿页、个人主页都能用）。移到项目的 key 之下，`/browse/PROJ-12`、首页、草稿页、个人主页上就没有这两个弹窗了（独立评审 I8）。所以照 M3 交接第 6 节给的另一种做法：宿主在 `projectId ?? workItemIdentifier` 改变时和卸载时（`useLayoutEffect` 的清理）关上它渲染的每个开关，后退的检查照旧。
  - 工作项详情的根 store 的 `peekIssue` 和弹窗开关（`issue-details/root.store.ts:126,143-150`）、`issueCrudOperationState`、`issueLinkData`、`relationKey`：四个宿主在卸载时关上它们渲染、设的开关并重设这三份数据，浮层的宿主清空 `peekIssue`。`isArchiveIssueModalOpen` 没有谁设它，删除，连同 `isAnyModalOpen` 中读它的一处。
  - 工作项的三个页面（`issues/(detail)/[issueId]`、`archives/issues/(detail)/[archivedIssueId]`、`browse/[workItem]`）和个人主页（`profile/[userId]`、`[profileViewId]`）以页面自己的 id 给子树 key。
  - I22 和 P14、P13 的后退检查：开着一个弹窗或浮层，后退到另一个项目、工作项或人之后它不再开着。

### 7.2 类型
生成的类型（`@nerve/api-client`）取代下面的 Plane 类型，没有转换层（总体设计 7.2）：

| Plane 类型（`web/packages/types/src/`） | 生成的类型 | 说明 |
|---|---|---|
| `TIssue`、`TBaseIssue`、`TIssueMap` | `Issue`、`IssueDetail` | 关联字段带 `_id`（`created_by` → `created_by_id`）；没有 `estimate_point`、`cycle_id`、`module_ids`、`attachment_count`、`is_draft`、`is_intake`、`is_subscribed`、`is_epic`、`type_id`、`description_binary`（3.2，M1-P2、M1-P3 交接） |
| `TIssuesResponse`、`TGroupedIssues`、`TSubGroupedIssues`、`TGroupedIssueCount`、`TIssuePaginationData`、`TPaginationData` | `IssueList`、`IssueGroup`、`IssueSubgroup`；本地的 `PagedGroups`（3.22） | 页面按 `groups` 是否存在分支（5.2） |
| `TIssueOrderByOptions`、`TIssueGroupByOptions`、`TIssuePriorities`、`TIssueRelationTypes` | `IssueOrderBy`、`IssueGroupBy`、`IssuePriority`、`IssueRelationType` | 键名改为 3.16 的（`state_name`、`label_name`、`assignee_name`、`created_by_id`、`label_id`……）；`EIssueGroupByToServerOptions`、`EServerGroupByToFilterOptions` 删除 |
| `IIssueDisplayProperties`、`IIssueDisplayFilterOptions`、`IIssueFilters`、`IIssueFiltersResponse`、`TWorkItemFilterExpression` 的存储 | `IssueDisplayProperties`、`IssueDisplayFilters`、`IssueFilters`（`ProjectPreferences` 的三个字段） | 表达式只留在富筛选的 store 里，存储和请求用 `IssueFilters`（3.17、3.22） |
| `TIssueComment`、`TIssueCommentReaction`、`TIssueReaction` | `IssueComment`、`CommentReaction`、`IssueReaction` | 没有 `actor_detail`、`access`、`comment_json`、`is_member`、`issue_detail`、`project_detail`、`workspace_detail`；`JSONContent` 删除（M1 收尾交接） |
| `TIssueLink` | `IssueLink`、`IssueLinkMetadata` | — |
| `TIssueRelation`、`TIssueRelationIdMap` | `IssueRelation` | 对方是内嵌的 `Issue` |
| `TIssueActivity` 和 `base.ts` 的细节类型 | `IssueActivity` | `actor_detail`、`issue_detail`、`project_detail`、`workspace_detail`、`source_data`、`epoch`、`attachments`、`is_bot` 都没有（3.13、M3 交接第 5 节） |
| `TDescriptionVersion`、`TDescriptionVersionDetails` | `DescriptionVersion`、`DescriptionVersionDetail` | — |
| `TWorkspaceDraftIssue` 和 `workspace-draft-issues/base.ts` | `Draft`、`IssueDraftPayload` | 草稿页从 `payload` 读字段 |
| 工作区搜索的类型（`workspace.ts` 24 行、`project/projects.ts` 9 行，M3 交接第 5 节） | `Issue`（搜索的结果） | 项目的结果由页面在项目列表里过滤（3.18） |
| `EIssuesStoreType`、`EIssueLayoutTypes` | 保留为前端的常量 | 它们是前端的 store 和布局的种类，不是接口的值 |

- **使用方**：`TIssue` 131 个文件（`issue-layouts` 47、`issue-modal` 9、`issue-detail-widgets` 9、`inbox` 10、`store/issue` 15、`power-k` 6、`packages/types` 7，其余散在）、`EIssuesStoreType` 93、`IIssueDisplayProperties` 52、`IIssueDisplayFilterOptions` 40、`TFilterProperty` 40（R6 第 3 节，第三稿重数）。
- **哪一段删哪一个类型**（对抗性评审 I7）：
  - `TIssue`、`TBaseIssue`、`TIssueMap` 在 **P10a** 换成 `Issue`（读描述的地方是 `IssueDetail`）并删除。列表、详情、浮层、弹窗、收集箱（M7 的组件）都读同一份工作项，所以它们的使用方在同一段里改。做法是"先收窄、后换型"：P10a 先删掉对 `Issue` 没有的字段的读取（3.2 的迭代、模块、附件数、估算、`is_subscribed`、`is_intake`、`description_binary` 等），这时 `TIssue` 还在，每个任务都能编译；最后一个任务把类型和那一份工作项一起换掉，只剩导入、`created_by` → `created_by_id` 一类的改名和可空性，机械的改动、按目录核对。
  - 列表的形状和枚举（`TIssuesResponse`、`TGroupedIssues` 一类，`TIssueOrderByOptions`、`TIssueGroupByOptions`、`TIssuePriorities`）在 **P10b** 随 `base-issues.store.ts` 的重写换掉：它们只有列表一侧在用。
  - 详情的类型（评论、表情、链接、关联、动态、描述版本、草稿、搜索）在 **P11** 随各自的 store 换掉，`JSONContent`、`comment_json`、`is_bot` 一起删。
  - **过渡的类型，一个写明、有界的例外**（聚焦复核 M7）：P10a 换型之后、新的 service 到来之前，旧的 `core/services/issue/issue.service.ts`、`issue_archive.service.ts`、`issue_relation.service.ts`、`workspace_draft.service.ts` 和读它们的旧 store 仍调 Plane 的地址，签名改成了 `Issue`、`IssueDetail`（`TIssuesResponse` 里装的也是 `Issue`）。这正是下面对 M6、M7 拒绝的"把 Plane 的响应套上生成的类型"，所以只作为过渡：它能编译，Nerve 没有这些地址，运行时到不了。界限：`issue.service.ts` 的列表方法和 `issue_archive.service.ts` 在 P10b 随列表的 store 换掉；`issue.service.ts` 的其余方法、`issue_relation.service.ts`、`workspace_draft.service.ts` 在 P11 换掉；P11 的完成线核对 `core/services/issue/` 里没有用 `Issue`、`IssueDetail` 去类型 Plane 的响应的方法。界限之外只有 M7 的 `services/inbox/inbox-issue.service.ts`：它的 `create`、`update` 的参数是 `Partial<TIssue>`，换成 `Partial<Issue>` 之后留到 M7（它本来就是 `old-issue-urls` 的例外，`until` 是 M7），写进 M7 的交接（13.2）。
- **M6、M7 的工作项列表在 P10a 删除，不保留旧的形状**（对抗性评审 I7 的 b）：迭代、模块、项目视图、工作区视图的列表 store（`store/issue/{cycle,module,project-views,workspace}/` 的 `issue.store.ts` 和筛选的 store）都继承 P10b 要重写的 `helpers/base-issues.store.ts`，又由仍调 Plane 接口的 service 喂数据（`cycle.service.ts`、`module.service.ts`、`workspace.service.ts` 的 `TIssuesResponse`）：基类改成解析新的契约之后它们不可能"行为不变"，留着就要一层兼容（项目的规则不许），把 Plane 的响应套上生成的 `Issue` 又违反"类型来自 OpenAPI"。所以 P10a 删除它们、四个布局根（`roots/{cycle,module,project-view,all-issue}-layout-root.tsx`）、这些页面上的列表部分（迭代、模块、项目视图的详情页，`workspace-views/[globalViewId]`，已归档的迭代、模块的根）、`EIssuesStoreType` 的 `CYCLE`、`MODULE`、`PROJECT_VIEW`、`GLOBAL` 和读它们的分支（53 个文件），以及共用的布局组件里只为它们存在的分支（分组头的"添加已有"、日历的 `addIssuesToView`、迭代和模块的空状态、活动迭代和进度侧栏对列表 store 的调用、迭代的转移弹窗）。M3 删最近访问的小组件是同一种做法。代价：M6、M7 之前，迭代、模块、视图、工作区视图的页面只剩它们自己的页头和详情，没有工作项列表；M6、M7 照 Plane 的源码在新的列表引擎上加回（13.2），前端改动清单记下。

### 7.3 services 与 stores（P10a、P10b、P11）
- **services**（`core/services/issue/` 下按资源的新 service，每个方法一次调用生成的客户端）：`issue`（取、建、改、删、按编号取、归档、恢复）、`issue-list`（项目、个人主页的列表）、`sub-issue`、`issue-relation`、`issue-link`、`issue-comment`、`issue-reaction`（工作项和评论的）、`issue-activity`、`description-version`、`draft`、`issue-search`；项目的显示设置并入 M3 的 `preferences` service。M4 的 51 个旧方法中 7 个没有调用方，直接删除（R6 第 2 节）；旧 `issue.service.ts` 里属于 M6（`addIssueToCycle`、`removeIssueFromCycle`）、M7（关注的三个）、M5（附件）的方法移到各自的旧 service 文件，等各自的 M 替换（它们不在 M4 的挂载路径上，3.1）。旧 `ProjectService`（`core/services/project/project.service.ts`）的三个方法都被替换，文件删除（M3 交接第 5 节）；旧 `WorkspaceService` 的两个搜索方法删除，视图的方法（M7）留着。

| store（`core/store/issue/`） | 做法 |
|---|---|
| `helpers/base-issues.store.ts`（1,775 行）、`helpers/base-issues-utils.ts`、`helpers/issue-filter-helper.store.ts` | 重写在 `Reconciled` 和 `paged-groups.ts` 之上（3.22）：第一页、一组的下一页、确认的修改、`stale`；请求的参数由 `IssueFilters` 和显示设置拼出；本地重排用 3.16 的全序。Plane 的偏移量游标、乐观更新、按已加载条数算的游标删除；`base-issues-utils.ts` 的 `getFilteredWorkItems`（子工作项在浏览器里的筛选，含相对日期）保留（3.22） |
| `issue.store.ts` | 工作项的一份：按 id 的 `ReconciledByKey`，服务端的 `updated_at` 大的胜出、相等时后应答的胜出；`description_html` 只由 `IssueDetail` 写；读到别人改了成员关系时标 `stale`（3.22） |
| `project/`（`filter.store.ts`、`issue.store.ts`） | 显示设置读写项目的显示设置（3.17）；`PATCH` 整体替换一个字段；看板折叠的组照旧在 `localStorage` |
| `archived/`、`profile/`、`workspace-draft/` | 列表接新接口；已归档、个人主页的筛选和显示设置在 `localStorage`（已归档的键修正，3.17；草稿页没有筛选）；草稿的发布先 `PATCH` 再 `publish`（3.15） |
| `issue-details/`（`root`、`issue`、`sub_issues`、`relation`、`link`、`comment`、`comment_reaction`、`reaction`、`activity`） | 各接新 service；动态、评论读完全部页，修改之后从产生最后一页的游标再读（3.13）；子工作项的状态分布由状态目录算；附件（M5）、关注（M7）的 store 不挂载 |
| `issue_kanban_view.store.ts`、`issue_calendar_view.store.ts` | 拖动只用确认的结果；`PATCH` 的请求体不带 `id`、`project_id`（封闭的请求体，R4 第 C.7 节） |
| `cycle/`、`module/`、`project-views/`、`workspace/`（M6、M7） | 删除（P10a，7.2），连同它们的布局根和页面上的列表部分；M6、M7 在新的引擎上加回 |
| `command-palette.store.ts` | 工作项弹窗的开关不变，由宿主结束（7.1） |
| `@nerve/shared-state` 的 `rich-filters`、`work-item-filters` | 否定的一层删除（3.22）；翻译成 `IssueFilters` 的纯函数 |

- **改调的使用方**：store 的方法签名和字段一变，调用它们的页面和组件在改这个 store 的 Phase 改到能编译、行为不变；页面的新行为在 P12–P15。
- **错误经文案表**：工作项的创建、修改读 `error?.error` 的两处（`issue-modal/base.tsx:233,323`）改按 `code` 经 `errorMessageKey`（M3 交接第 6 节）；新码的文案与码在同一个任务里加（第 12 节的约束）。
- **工作项页面上新建标签**（`issue-detail/label/root.tsx`、`label/select/root.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`select/dropdown.tsx`、`select/base.tsx`）改经 `useLabelChanges().create`（M3 交接第 6 节）。
- **M3 交接第 6 节的另外三件**：工作区一级显示的工作项不显示没有取过标签的项目的标签（M7 加回）；`fetchParentStats` 在工作项的修改动了项目的字段时加一个不等的重取；工作项的键宽读 `next_work_item_sequence`、`TIssueIdentifierProps.projectIdentifier` 在项目未到时可以是 `undefined`。P11 逐个处理。

### 7.4 页面（P12–P15）
- **项目的工作项页**（P12、P13）：列表、看板（含泳道）、日历、表格四种布局，分组、子分组、每组"加载更多"、快速添加、快捷操作、浮层；拖动（I6）；表格的列排序；日历的月、周和拖动、"添加已有"（列出没有截止日期的，7.5）；筛选的工具栏、显示设置、显示属性。迭代、模块的分组、列、筛选不出现（3.2）。
- **`/:workspaceSlug/projects/:projectId/issues/:issueId`**（P14）：去掉 `clientLoader` 和模块级的 `IssueService`（`getIssueMetaFromURL`，`/issues/{id}/meta/`）；页面经会话的 store 取工作项、从项目列表取标识，跳到 `/browse/PROJ-12`。找不到时的按钮跳到项目列表（3.1）。
- **`/:workspaceSlug/browse/:workItem`**（P14）：经 `getIssueByIdentifier`；`useSWR` 改经 `useSessionSWR`；收集箱的跳转删除（3.1）。详情的各区：属性、描述、子工作项、关联、链接、表情、时间线（评论和动态）、描述的历史（只给管理员、成员）、归档和删除。
- **项目的已归档工作项**（P13）：列表（只有列表布局，Plane 相同）、详情（只读，只能恢复和删除，3.8）；页头的模块级 `IssueService` 删除。
- **草稿页 `/:workspaceSlug/drafts`**（P15）：列表、编辑、删除、"移到项目"。只读工作区的地址的 4 个文件改用 `useRouteWorkspace()` 或路由的 slug（M3 交接第 8 节）：草稿页的 3 个（`drafts/header.tsx`、`workspace-draft/empty-state.tsx`、`workspace-draft/root.tsx`）和命令面板的 `power-k/config/creation/command.ts`（7.9）。
- **个人主页的三个标签页**（P13）：成员和管理员看到工作项列表（列表、看板，按项目、状态组、优先级、标签分组）；访客看到"没有权限"，筛选的工具栏不在它之上（M3 交接第 6 节，M3/P11 评审 C10）。
- **命令面板**（P15）：工作项的命令（`power-k/ui/pages/context-based/work-item/commands.ts`，418 行，改写时按命令的种类拆成几个文件，M3 交接第 8 节）；搜索（3.18）；主题命令在应答成功之后才应用，失败时按 `code` 提示（M2 交接第 9 节，经 `inSession()`）。
- **评论的删除**（P15，决策点 8，负责人裁定 B）：项目管理员在别人的评论的菜单里也有"删除"（没有"编辑"）；今天的页面只给作者显示菜单（`comments/quick-actions.tsx:41-43`）。登记前端改动清单。
- **删除确认框**写明子树（3.6）；**归档**的菜单项只在状态组是完成、取消时出现（Plane 相同）；**自动归档**的动态显示站点名（3.8）。
- **表格的"子工作项"列**直接链接 `/browse/PROJ-12`，去掉 `#sub-issues` 片段（没有元素读它，M1-P4 交接）。
- **个人设置页不挂载工作项的弹窗**（M1-P4 交接）：`app/(all)/settings/profile/layout.tsx:16` 的 `ProjectsAppPowerKProvider` 换成不含工作项弹窗的命令面板（那里没有工作区，弹窗本来就打不开）；这同时去掉了它在打开之前建出的编辑器（M2 交接第 7 节）。

### 7.5 弹窗、下拉框和键盘（M3 交接第 7、8 节；M2 交接第 10、11 节）
M4 的页面上逐项照 M3 收尾的写法改，并由故事或浏览器核对（9.7）。路径在 `web/apps/web/core/components/` 下。
- **弹窗**：工作项的全部弹窗经 Headless UI v2 的 `ModalCore`：Tab 在弹窗里转、Escape 和点外面关上、关上之后焦点回到打开它的控件、弹窗里 portal 的日期、下拉框、表情能用；`AlertModalCore` 的确认框被拒绝时留着，按钮文字经 `t()`。
- **弹窗的名字**：以普通标题的四个（`issues/archive-issue-modal.tsx:73`（P13）、`issues/confirm-issue-discard.tsx:40`（P14，嵌套的弹窗一起）、`issues/issue-detail/links/create-update-link-modal.tsx:82`（P14）、`issues/issue-modal/form.tsx:267`（P14））改经 `ModalTitle`；没有标题的两个（`core/description-versions/modal.tsx`、`issues/parent-issues-list-modal.tsx`）给 `sr-only` 的；`start-date.tsx`、`due-date.tsx` 给日期弹窗的 `title` 经 `t()`。故事断言 `toHaveAccessibleName`。
- **正的 `tabIndex`**：工作项的表单（`getTabIndex(ETabIndices.ISSUE_FORM)`）、`existing-issues-list-modal.tsx:70,156`、`parent-issues-list-modal.tsx`、`label-select.tsx` 的 `baseTabIndex`（`:53,162`）、`default-properties.tsx` 给父工作项菜单的、`archive-issue-modal.tsx`、`description-versions/modal.tsx` 写死的 `tabIndex={1}`，在 M4 删去。`existing-issues-list-modal.tsx` 是 M4 的选择器（独立评审 I7）：子工作项区的"添加已有"（`sub_issue: true`）和关联的选择（`issue_relation: true`，带工作区的开关），见 `issues/issue-detail-widgets/issue-detail-widget-modals.tsx:136-194`，另有项目日历的"添加已有"（`calendar/quick-add-issue-actions.tsx`）；它的 `tabIndex` 归 M4（P12，随它改接）。迭代、模块经它加已有的工作项的调用（`list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx` 的迭代、模块分支，日历的 `addIssuesToView`，`empty-states/cycle.tsx`、`module.tsx`）随 M6 的列表在 P10a 删除（7.2），M6 加回。M4 删去自己的之后 `getTabIndex` 若已没有调用方，M4 删除 `getTabIndex`、两份 `tab-indices.ts`（`web/packages/constants/src/`、`web/packages/utils/src/`）和 `CustomMenu` 读 `tabIndex` 的一支；还有 M6、M7 的调用方时写进它们的交接（M6、M7 的 M3 交接说"最后删去正的 `tabIndex` 的那一个删除 `getTabIndex`"，13.2）。P16 的第一个任务数一次。
- **两个选择器经 `issue-search`**（3.18；独立评审 I7，对抗性评审 I5）：
  - `ExistingIssuesListModal` 在 **P12** 改接：它在项目日历上就用得到（"添加已有"：`project_id` 加 `target_date=none`，只列没有截止日期的），所以随日历的那一段改；同一个任务把它的另两个调用方的参数换成 `for=sub_issue`、`for=relation`（关联的工作区开关打开时不带 `project_id`），去掉它的正的 `tabIndex`。它一打开就以空的词搜索，`q` 为空时服务端给出最近更新的（3.18）。
  - `ParentIssuesListModal` 在 **P14** 改接（`for=parent`），随 I8 的页面版本。
  - 排除自己、后代、祖先、已关联的都由服务端做。
- **嵌套的按钮**：`DropdownButton` 放在 Headless UI 的按钮里的写法随属性下拉框和页头改；菜单的按钮里放 `IconButton` 的三处（`comments/quick-actions.tsx`、`issue-layouts/quick-action-dropdowns/issue-detail.tsx`、`issue-activity/activity-filter.tsx`）照 `navigation/project-actions-menu.tsx` 改（`customButtonClassName`）；`peek-overview/header.tsx:188` 的 `customButton`、`default-properties.tsx` 给父工作项菜单的 `customButton` 不再是按钮里的按钮。
- **`CustomMenu` 的键盘**：快捷操作的 `SubMenu`（5 个文件）的触发项键盘打得开；看板、日历卡片的快捷操作在键盘打开时也显示"激活"的底色。
- **下拉框**：`DateDropdown` 只有一份打开状态（Popover，选中时 `close`），连同列表、表格单元格里的懒渲染，键盘打得开（M2 交接第 10 节）；7 个下拉框（`dropdowns/{date-range,priority,project/base,state/base}.tsx`、`issue-detail/label/select/label-select.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`select/base.tsx`）的 popper 的 ref 放在列表元素上，只要 Headless UI 的一份打开状态（没有搜索框的 `Listbox`、有的 `Combobox`），在弹窗里 `modal={false}`（`useInsideModal()`，M3 交接第 7 节的"弹窗里的列表"）；`select/base.tsx` 的根不再是一个没有名字的停留点；`CustomSearchSelect` 的 `defaultOpen` 在富筛选里核对；`rich-filters/` 的 4 个调用方传经 `t()` 的 `searchPlaceholder`、`noResultsMessage`、`loadingMessage`（M3 交接第 8 节）；成员下拉框的 8 个文件、4 个筛选的新行为在页面上核对。
- **`@nerve/ui` 下拉框的 `any`**（`dropdowns/helper.tsx` 7 处、`custom-search-select.tsx` 2 处）改成泛型，13 个调用方一起改（M3 交接第 8 节；多数在 M4 的页面上，余下的 M6、M7 调用方改到能编译）。
- **筛选的按钮**：`FiltersDropdown` 的 7 个调用方，故事断言按钮的名字、`aria-expanded`、Escape 之后焦点回到它。
- **组字的判断**：新加的键盘处理只经 `isComposing`（关键词规则 `composing-check`）。
- **右键菜单**关着时 `inert`，在列表、草稿上核对。
- **层叠顺序**：`project-layout-root.tsx:80`、日历的两个 `Popover.Panel`、`quick-add/form/calendar.tsx:22`、更新中的遮罩（`calendar/issue-block.tsx:110`、`list/block.tsx:217`）、拖动中的 `kanban/block.tsx:261`、`kanban-delete-drop-zone.tsx:52` 降到 `z-30` 之下，或开着弹窗核对它们不画在弹窗上（附件的 `attachment-item-list.tsx:123` 是 M5 的，不挂载）。
- **焦点的轮廓**：`Button`、`IconButton`、`getButtonStyling`（`rich-filters/add-filters/button.tsx`、`dropdowns/layout.tsx`）、`CustomMenu` 不被 `overflow-hidden` 裁掉，键盘核对。
- **每一行的控件以它那一行为名**：评论的菜单按钮和"加表情"以评论的作者为名（"Options for {name}'s comment" 一类，en 与 zh-CN 同时加）；新加的 `IconButton`、`EmojiPicker` 都有名字。
- **字段的提示和被拒绝时的焦点**：工作项的标题（`issue-modal/components/title-input.tsx:85`）、链接的地址（`create-update-link-modal.tsx:112`）、项目的选择（`issue-modal/components/project-select.tsx`）、快速添加的标题（`quick-add/root.tsx`，给出提示）、详情的标题（`issues/title-input.tsx:147,173,180`）、筛选的日期范围（`rich-filters/filter-value-input/date/range.tsx:56`）照 M3 的写法：`Field` 里的 `FieldError`，控件 `aria-invalid`，焦点只经 `useRefusedFocus`。故事断言焦点、`toHaveAccessibleDescription`，并用 `recordFocuses` 断言取得焦点的那一刻已 `aria-invalid`。
- **列表里不是选项的控件**：`select/base.tsx:299`、`label-select.tsx:201` 的"新建标签"改为哨兵值的选项；`parent-issues-list-modal.tsx:162` 选项里只在指针经过时显示的链接删除（打开工作项另有菜单）。
- **嵌套的弹窗**：工作项的弹窗里打开"Save this draft?"和选父工作项，在里层按 Escape，里层关上、外层留着（I1 的一步）。**描述编辑器的 Escape 监听**（`issue-modal/components/description-editor.tsx:90` 监听整个 `document`）改为只理会目标在它自己的弹窗里的 Escape，选父工作项的 Escape 不再关上或打开外层的东西；故事核对。
- **复制只经 hook**：M2 交接的 6 处和 M3 交接的 10 个文件（有重叠）改经 `useCopyLink`、`useCopyText`，失败时经 `t()` 提示；`copy-hook` 的 10 个例外随之删除（7.9）。

### 7.6 编辑器（`@nerve/editor` 与 `core/components/editor`）
- **代码分割**（M2 交接第 7 节）：编辑器的模块到用它的页面才加载（`React.lazy` 包住富文本、轻量编辑器的入口）；未登录时打开需要登录的页面，不再先加载 tiptap 的 Emoji 节点和 `is-emoji-supported`。个人设置页不再挂载工作项的弹窗（7.4），它在打开之前建出的编辑器随之没有。之后删除 e2e 的 `EMOJI_CHECK_WARNING`：定义（`e2e/fixtures/browser.ts:166`）、29 个故事文件的 96 行，和 `e2e/fixtures/settings-pages.ts` 的一处（它也要清掉）。
- **标注块的默认表情**（M2 交接第 8 节）：默认值的 `EMOJI_URL` 指向 `cdn.jsdelivr.net`（`editor/src/extensions/callout/utils.ts:19`），被 CSP 挡住。改为不带图片地址、只用 `EMOJI_UNICODE`（`128161`）以原生表情显示；标注块的显示在没有地址时用原生表情。表情回应的选择器（M3 改为本站的数据）在 P15 核对没有 CSP 违规。
- **`worker-src`**：编辑器和它的扩展没有 `blob:` 的 worker（`git grep` 没有 `new Worker`；`URL.createObjectURL` 只在 `@nerve/utils` 的 CSV 下载），CSP 不加 `worker-src`。P14 的评审在打包之后的产物上复核一次，写进评审记录（M2 交接第 8 节）。
- **Markdown 的节点序列化删除**（M1 收尾交接）：5 个扩展（标注块、`custom-color`、`custom-image`、`emoji`、`mentions`）的 `addStorage` 的 `markdown.serialize` 没有调用方（`getMarkdown()` 不被调用，`transformCopiedText` 为假），删除；`git grep -n "markdown: {" -- web/packages/editor/src` 没有输出。
- **只补块 id 的保存**：详情的描述照旧发 `skip_activity`（布尔值，不再是字符串 `"true"`）；工作项的弹窗在**编辑已有的工作项、打开草稿**时读 `isMigrationUpdate`，不把表单标成已改（新建时描述是空的，不涉及；M1 收尾交接，P14；vitest 和 C4 写这两种情形；I2 的一步）。
- **描述保存的"撤销再重做"**（M1-P2 交接）：`description-input/root.tsx` 在 1.5 秒的保存延迟内先撤销再重做，最终发出撤销后的内容。重写保存时，待发的保存总是取最后一次的内容（重做回到上次保存的内容时取消待发的那一次），vitest 覆盖。
- **不再发 `description_json`**（3.7）。
- **@提及**：候选来自项目成员（没有项目的草稿用工作区成员），不再请求 `entity-search`（3.12）。
- **图片**：M4 的编辑器关闭 `image` 扩展（3.1）。

### 7.7 删除的代码
- 3.1、3.2 列出的入口和取数：附件区、关注按钮、收集箱跳转、迭代和模块的分组、列、筛选、显示属性（各自的 M 加回）。
- 迭代、模块、项目视图、工作区视图的工作项列表（P10a，7.2）：四个列表 store 和它们的筛选 store、三个旧 service 里返回 `TIssuesResponse` 的方法、四个布局根、页面上的列表部分、`EIssuesStoreType` 的四个值和读它们的分支、共用的布局组件里只为它们存在的分支（M6、M7 加回）。
- `TIssue`、`TBaseIssue`、`TIssueMap` 和 Plane 的列表、详情的类型（7.2，P10a、P10b、P11）。
- Plane 的列表引擎在前端的部分：偏移量游标、乐观更新、`EIssueGroupByToServerOptions`、`EServerGroupByToFilterOptions`、本地按已加载条数算游标（3.22）。
- 否定的筛选的一层：`TAllAvailableOperatorsForDisplay`、`TAllAvailableDateFilterOperatorsForDisplay`、`TNegativeOperatorConfig`、`isNegation` 的参数、`useFiltersOperatorConfigs`（M1-P3 交接，3.22）。
- 7 个没有调用方的 service 方法；旧 `ProjectService` 文件；`WorkspaceService` 的两个搜索方法；`issue_versions` 没有前端代码（Plane 的页面本来就不读）。
- 类型：`JSONContent` 和 `comment_json`（`comment-create.tsx` 编辑器回调里的同名参数一起）；`is_bot`（`types/src/issues/activity/base.ts:50`、`comments/card/display.tsx:65`）；`TIssueActivity*Detail` 一类；工作区搜索的类型。
- 动态：`IssueCreator`（与 `IssueUser` 相同）；`default.tsx` 读 `source_data.source` 的"via 来源"一支（接口没有 `source_data`，收集箱的来源在 M7 定，3.2、13.2）；`archived-at.tsx` 按 `new_value === "archive"` 的特判（改为执行者为空的通用规则）。
- `use-issues-actions.tsx:354` 的 `as TProfileViews`：个人主页的路由段在路由处收窄一次，`TProfileViews` 往下传（M1 收尾交接）。
- `isArchiveIssueModalOpen`（7.1）。
- 草稿的四张关联表对应的前端字段（`draft_issue_assignees` 一类，Plane 的草稿结构）：草稿的字段从 `payload` 读。
- 删除之后 knip、tsc 照旧通过；`domains.mjs --rows M4` 的行随之减少（7.8）。

### 7.8 死成员、死 prop 和 oxlint（M1 收尾交接；M3 交接第 8 节）
- **死成员**：`domains.mjs --rows M4` 545 行（成员 426、prop 119），加上按路径归到 M3 的 52 行（`workspace-draft-issues/base.ts` 16、`types/src/workspace.ts` 24、`project/projects.ts` 9、`workspace.service.ts` 3），共约 597 行（R6 第 5 节）。做法照 M1 收尾：改到一个文件时删掉其中列出的行；多数随 types、services、stores 的重写消失。收尾时每一行都已消失，或写进收尾的评审（文件、名称、谁在读或传它）。大户：`@nerve/types` 149、`@nerve/editor` 120、`issue-details` 44、表格的列 29。
- **oxlint**：
  - **谁改谁清**：M4 改到的文件在 M4 结束时没有警告；按 `domains.mjs` 的划分 M4 领域 197 条、89 个文件（R6 第 6 节）。点名的：`issue-activity/helper.tsx` 2、`description-versions/modal.tsx` 1、`issue-activity/activity-filter.tsx` 2（M3 交接第 8 节）。
  - **按规则清一类**：`eslint(no-shadow)` 全仓清零（全仓 128 条，M4 的文件组占 108 条，余下约 20 条在 M4 之外，R6 第 6 节）；P16 的一个任务，`.oxlintrc.json` 把它升为错误。
  - 各包的上限随之调低（`tools/lint-cap.mjs`），收尾的评审写明每个包的变化。
- **两个工作项的删除确认框的 `onSubmit` 必传**，改写时保持（M3 交接第 8 节）。

### 7.9 关键词守卫（`tools/keywords.json`）
- **阶段**：每个 Phase 开始时 `phase` 改为 `M4/P<k>`，收尾开始时改为 `M4/closeout`（M3 交接第 8 节）。
- **到期的例外**：
  - **规则**：一个例外的 `until` 不晚于 `phase` 就到期，守卫失败（`tools/keywords.mjs:3-6,195`）。所以改 `phase` 的那个提交同时改到期的例外的 `until`，`until` 取删除这个例外的 Phase 的下一个（在删除它的 Phase 进行中仍有效）。
  - `plane-workspace-urls` 的 `/user-properties/` 两处 `until: "M4"`（`keywords.json:2526`）：P1a 把 `phase` 改为 `M4/P1` 的同一个提交里把 `until` 改为 `M4/P11`（3.17；P1b 的 `phase` 照旧），P10b 删除这两个方法和例外；e2e 的 `workItemFilters`（`e2e/fixtures/mounts.ts:56`）和三个故事里的放行一起删除（M3 交接第 5 节）。
  - `copy-hook` 的 10 个 `until: "M4/closeout"`（`:2533-2597`）：P14、P15 改完复制之后删除，收尾时守卫要求它们已不在。
- **扩大范围**：`project-address-route` 加上工作项页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节的 6 个和裁定 F-12 补上的 16 个），改成读路由参数；`workspace-address-route` 加上只读工作区的 4 个（M3 交接第 8 节），其中 `power-k/config/creation/command.ts` 是 `.ts`，而规则的文件正则以 `\.tsx$` 结尾（`keywords.json:2381`），P15 把它放宽为 `\.tsx?$`；`copy-hook` 的范围加上 M4 的组件目录。
- **新规则**（P10a 起）：
  - `old-issue-urls`：`web/` 里不再出现 Plane 的工作项地址（`/issues/`、`/work-items/`、`/archived-issues/`、`/draft-issues/`、`/draft-to-issue/`、`/search-issues/`、`/entity-search/`、`/user-issues/` 前面带 `/api/workspaces/` 的写法）；M5、M6、M7 的旧 service 文件按例外登记，`until` 是它们各自的 M。
  - `issue-optimistic`：工作项的 store 不在应答之前写值（`runInAction` 出现在 `await` 之前的写法，按文件范围），只经 `Reconciled`。规则的写法在 P10b 的 spec 定，做不到静态可靠时改为 vitest 的表。
  - `label-create-direct`：工作项的组件不直接调 store 的 `createLabel`，只经 `useLabelChanges().create`（M3 交接第 6 节）。
  - `negation-filters`：`isNegation`、`TNegativeOperatorConfig`、`allowNegative` 不再出现。

### 7.10 规模估计（按 M3 的实际校准，R6 第 9 节）
M3 的校准：组内改动的行数估得准（估 9,550、实 9,879）；组外（跟随类型的使用方）是估计的 4.7 倍；vitest 是 4 倍（每行生产代码约 0.78 行测试）；任务的个数估得对（55），但每个任务更大：约 287 行生产代码、223 行测试、10 个文件。

| 范围 | 领域的文件 / 行 | 按 M3 的比例 | 改动的行（删 / 重写 / 加） | 约改到的文件 | 约任务 |
|---|---|---|---|---|---|
| services 和生成的类型（51 个旧方法 → 约 12 个按资源的 service；工作项的类型 → 生成的） | 45 / 2,578 | 120% / 60% | 约 2,100 | 约 45 | 8 |
| 列表的 stores（helpers 2,485 行、project、archived、profile、draft、workspace、root、issue、calendar、kanban；显示设置；`reconciled.ts` 的扩展和 `paged-groups.ts`） | 22 / 5,818 | 72%（另加约 340 行新的） | 约 4,500 | 约 25 | 15 |
| 详情的 stores（工作项、评论、表情、链接、关联、子工作项、动态、版本；弹窗的开关由宿主结束） | 10 / 2,401 | 72% | 约 1,700 | 约 12 | 6 |
| 筛选和共用的工具（rich-filters 18 / 1,306、work-item-filters 6 / 610、issue-layouts/filters 23 / 1,441、`@nerve/shared-state` 17 / 1,926、constants 13 / 1,097、utils 51 / 2,859） | 128 / 9,239 | 30% | 约 2,800 | 约 75 | 9 |
| 布局的组件（列表、看板、表格、日历、快速添加、快捷操作、属性、根、空状态、`utils.tsx`） | 117 / 12,955 | 28% | 约 3,600 | 约 85 | 12 |
| 详情、浮层、组件、弹窗、评论、版本、编辑器的包装、标题、选择、关联 | 163 / 14,702 | 30% | 约 4,400 | 约 120 | 15 |
| 草稿、归档、个人主页、`browse`（路由 24 / 1,485 加组件）、命令面板、hooks 23 / 2,034 | 74 / 5,753 | 35% | 约 2,000 | 约 55 | 7 |
| 下拉框（17 / 2,320）、`core/filters`、`core/modals`、命令面板的 store，加 7.5 的键盘、弹窗各项 | 22 / 3,124 | 45% | 约 1,400 | 约 18 | 5 |
| `@nerve/editor` 的定点修改（序列化、标注块、代码分割、弹窗的迁移、120 行死成员） | 179 / 18,392 | 定点 | 约 800 | 约 40 | 3 |
| **组内小计** | 581 / 56,570（另 editor 179 / 18,392） | 约 41% | **约 23,300** | **约 475** | **80** |
| 组外：M5、M6、M7 的代码里 `TIssue`（124）、`EIssuesStoreType`（93）、`useIssueDetail`（105）、`useIssues`（62）的使用方；迭代、模块、视图、收集箱的根；`@nerve/ui` 下拉框的泛型（13 个调用方）；一类 oxlint 规则；i18n；复制 | — | M3：领域行数的 25% | 约 7,000 | 约 250 | 8（多数随 store 的任务） |
| vitest（每个修改一张表：会话、权限、拒绝；`paged-groups.ts` 的规则表；翻译筛选的纯函数） | — | 0.78 × 生产 | **约 23,600 新增** | 约 220 | 在上面的任务里 |
| 每个 Phase 的评审和文档任务 | | | | | 约 7 |
| **合计** | | | **约 30,300 行生产代码、约 23,600 行测试** | **约 950** | **约 95（85–110）** |

- 第 12 节的前端 Phase 合计 106 个任务（P10a–P16），落在这个区间的上端：比第二稿多的是 P10a 删除 M6、M7 的列表（7.2）和第三稿移进的任务（I20 的页面、评论的删除、`ExistingIssuesListModal` 的改接）；收尾另 14 个任务，多半是前端（M3 的收尾 16 个任务，远超它的原型，预留一轮修复、一轮浏览器核对之后的修复，第 16 节）。
- 没有计入：端到端的故事和 fixtures（M3 是估计的 2.3 倍）、生成的 `schema.gen.ts`（M3 为 45 个操作加了 2,360 行）、后端。

---

## 8. 安全

### 8.1 出站请求（链接的抓取，架构问题 8）
- 抓取是 nerve 第一次按用户给的地址发出请求，它是 SSRF 的入口：任何项目的成员都能让服务器去取一个地址。6.8 的客户端照 Plane 的防护（只许 `http(s)`、解析之后拒绝内网和保留的地址、连接到检查过的那个地址、每一跳重新检查、不用环境的代理），把拒绝的表写成 IANA 的两张特殊用途登记表的全部网段，加上启动时本机网卡上的地址（不是公网地址不在网卡上的部署，那里靠网络层的出站规则，8.8），并补上 Plane 缺的上限（响应 1 MiB、图标 64 KiB、整体 5 秒）。
- 抓取的结果只有标题（文本，React 转义显示）、图标（类型受限的 `data:` 地址，只作 `<img>` 的 `src`）、最终的地址（文本）；不把页面的任何内容当作 HTML 显示。不存 SVG（3.11）。
- 抓取不带 cookie、不带凭证，`User-Agent` 写明是 nerve 的链接预览，被抓的站点能识别它。
- 测试专用的放行（`outbound.allow_cidrs`）只在 `env = test` 时接受（3.11）；生产配置带着它启动失败。
- `metadata.error` 只有 `blocked`、`failed` 两个值（3.11），不写响应的内容、不写失败的细分；细分只进服务器的日志。被拒绝的地址（内网、保留、本机）在连接之前就是 `blocked`，与那里的端口开不开无关，所以抓取不是探测内网的回显。
- **剩余的风险**：
  - 公网上的地址由调用者决定，nerve 会去请求它（Plane 相同）。
  - **公网地址上的端口探测仍可做**（对抗性评审 M3）：调用者写 `http://<公网地址>:<端口>/`，从有没有标题、`failed` 来得多快，能推断那个端口上有没有 HTTP 服务；第二稿的六种错误让这个回显更细，第三稿收成两种，但不消除它。它多半只到达任何人都能从公网直接探测的地址，探测一个端口要建一个链接、等一个任务。例外是本机自己的公网地址不在网卡上的部署（容器、NAT、云的负载均衡，6.8）：抓取可以经它到达只在外部防火墙上挡着的本机服务；这一点和只出不进的网络一样，由管理员在网络层限制 nerve 的出站（README，8.8；聚焦复核 M9）。

### 8.2 富文本
- 描述和评论的 HTML 在写入时用 bluemonday 清洗（3.7），放行表照 Plane 的 nh3 表（`content_validator.py:71-160`）；页面照旧经编辑器渲染。清洗不拒绝、只去掉，所以带脚本、事件属性、`javascript:` 地址的请求被存成去掉了它们的 HTML（P4 的测试逐类核对）。
- **`skip_activity` 留下的口子**（3.7）：带这个标志的保存，内容标识（纯文本、按顺序的提及的 id、按顺序的链接地址、按顺序的勾选框状态）不变时不留动态和版本、不改 `updated_at`。所以只改格式（加粗、列表的层次、标题的级别、代码块的语言）的修改可以不留痕迹；文字、提及、链接的去处、勾选框的任何改动都照常记下（第二稿只比较纯文本和提及，改一个链接的去处、勾一个勾选框都能不留痕迹，对抗性评审 I4）。S3 的样例含只改链接地址、只改勾选框的两种，必须判为修改。内容仍经清洗，这一条不涉及注入，只涉及审计的完整；登记差异（4.17）。
- 提及的 `entity_identifier` 只在同步 `issue_mentions` 时被读，读成 UUID（不是 UUID 的忽略）。
- 链接的地址只许 `http`、`https`（3.11），页面上的链接不会是 `javascript:`。
- 动态的 `old_value`、`new_value` 是存下的名字和地址，页面以文本显示（描述的那两列不显示，3.13）。

### 8.3 枚举
| 入口 | 回答 | 说明 |
|---|---|---|
| 工作项、评论、链接、关联、版本的 id 不存在、看不到、被访客规则挡住 | 各自同一个 404 | 6.7；id 是随机的 UUID |
| 按编号取 `PROJ-12` | 格式不对、项目不存在、看不到、编号不存在：同一个 404 `issue.not_found` | Plane 对不是成员答 403，透露项目和编号存在（3.7） |
| 加关联的对方、加子工作项、负责人、标签 | 不存在与看不到同一个 422 `not_allowed` | 不透露别的项目里有没有这个 id |
| 搜索 | 只返回可读的项目里过了访客规则的 | Plane 的命令面板不执行访客规则（3.18） |
| 个人主页的工作项 | 只返回调用者可读的项目里的 | Plane 不执行访客规则（3.4） |
| 动态、子工作项数 | 引用看不到的工作项的动态行略去；子工作项数按调用者的规则数 | Plane 显示这些行里的编号，计数含看不到的（3.4、3.13） |
| 草稿 | 别人的草稿 404 | 只属于创建者（3.15） |
| 已删除的项目里的工作项（删除的任务还没做完时） | 与不存在相同：按编号取、搜索、列表、个人主页、别的项目的关联、草稿的发布都当它不存在 | 项目一删除就是"看不到"（3.6），不等那些行被软删除 |

### 8.4 404 与 403 的边界
- 看不到的一律 404，看得到而不能做的 403 `forbidden`（3.4）。访客规则挡住的工作项算"看不到"：访客改别人的工作项与改不存在的工作项同一个 404。访客规则按项目的角色判：工作区管理员在这个项目里只是访客时同样只看自己的，404 优先于 403（3.4，对抗性评审 M4）。
- 规则只有一处（`access/domain/decide.go`）加一个过滤（`OwnOnly` 在用例和 SQL 里的同一个含义）；矩阵逐格测试（9.2），列表、个人主页、搜索的过滤与逐个判定一致（9.3）。
- 写在项目行的锁之后判定（3.5）：等锁期间被降级、被移出的人拿不到过期的授权。锁之前另有不加锁的预检，凡会被拒绝的（404 或 403）都在取锁之前得到回答：看不到项目的人、只能读的人都不能排他地锁住项目，也不占项目的写闸门（3.5、11.11）；锁下的判定只拦等锁期间的变化。
- 请求体的结构检查挡住越权的字段：`PATCH /issues/{id}` 带 `project_id`、`sequence_id`、`archived_at`、`completed_at`、`created_by_id` 都是 400（Plane 的序列化器允许写它们，R2 第 1 节）。

### 8.5 CSP
- `img-src 'self' data: blob:` 不变：链接的图标是 `data:` 地址（3.11），标注块的默认表情改为原生表情（7.6），表情回应的选择器用本站的数据（M3）。
- 不加 `worker-src`（编辑器没有 `blob:` 的 worker，7.6）、不加 `frame-src`（M7 的收集箱、视图要嵌 iframe 时再加，M2 交接第 8 节）。
- 编辑器的图片在 M5 之前关闭（3.1），页面不从外部取图。

### 8.6 游标的篡改（M2 交接第 2 节）
- 游标不签名（总体设计 3.4）：改成另一个合格的位置只是从那里接着读。列表的可见性（可读的项目、访客规则、`archived`、筛选）每次都按请求和调用者重新施加，不来自游标：游标的载荷只有排序码、键和 `id`（3.16），不含项目、工作区、筛选。
- 篡改的游标能做到的只是跳到同一个结果集里的另一个位置。测试：拿别人（能看到更多的人）的游标、拼一个指向看不到的工作项的 `id` 和键，结果里没有任何调用者看不到的行（9.3）。
- 每个游标分页的列表都有自己的载荷和这样一个测试（M2 交接第 2 节）：工作项的列表（P5）、评论和动态（P4）、描述的历史（P4，`[last_saved_at, id]`）、草稿（P8，`[created_at, id]`；把另一个人的草稿的 `id` 和时刻放进游标，结果仍只有调用者自己的草稿，不透露那一份是否存在）。
- 排序码与请求不符、形状不对：400 `bad_request`（3.16）。

### 8.7 其他
- **草稿的载荷**：只在发布时按创建的规则检查（3.15）；保存时它是调用者自己的数据，只给他自己看。大小受请求体的上限（1 MiB）约束。
- **描述历史的大小**：每个工作项至多 20 个版本（3.14），软删除的随工作项在 60 天之后物理删除（3.21）。
- **动态的增长**：没有保留的上限（Plane 也没有，4.11）；物理清理只删除随工作项删除的。第 16 节记为风险。
- **物理清理不删账户**（M3 交接第 2 节）：成员关系不会不经工作区的锁被级联删除（3.21）。
- **测试专用的配置**（`clock.offset`、`outbound.allow_cidrs`）在非测试环境里不为空时启动失败（3.20、3.11）。

### 8.8 README 的新增内容
| Phase | 位置 | 内容 |
|---|---|---|
| P1b | 部署 | 工作项的写闸门和锁的期限：`issue.write_gate.per_project`（默认 2）、`issue.write_gate.max_wait`（2 秒）、`issue.write_gate.max_queue`（64）、`issue.write_lock_timeout`（5 秒）；超过时答 503 `server_busy`，客户端按 `Retry-After` 重试；闸门限的是每个项目的份额，同时热的项目达到池的大小除以 K 时等锁的写仍能占满池；连接池的大小与多进程部署时的 K、同时热的项目数由 M8 的部署说明定（11.11） |
| P6 | 部署 | 定时任务（自动归档）的间隔配置；任务队列的 worker 数（`jobs.default_workers`、`jobs.periodic_workers`） |
| P7 | 安全 | 链接的抓取会从服务器发出请求：防护的规则、上限、公网地址上仍可探测端口（8.1）；公网地址不在本机网卡上的部署，在网络层挡住 nerve 访问自己的公网地址；要完全禁止时在网络层禁止 nerve 的出站（抓取得到 `failed`，不影响别的功能） |
| P8 | 部署 | 迁移执行 `CREATE EXTENSION IF NOT EXISTS pg_trgm`：迁移的账户要有库的 `CREATE` 权限（库属主即可），不要超级用户（M0-P4 交接；完整的部署说明在 M8） |
| P9 | 部署 | 60 天的物理清理：`cleanup.purge_after` 等三个配置；它不删除账户。描述历史的保留（`issue.trim_versions_interval`）。删除项目、工作区之后，工作项一侧的行由后台任务分批软删除（`issue.delete_batch`，默认 1000；每次运行的预算 `issue.delete_budget`，默认 45 秒，用完就重新排队，不算失败；决策点 7）：请求立即返回，那些行从那一刻起就看不到；任务用尽重试时写错误日志，行在 60 天之后随项目的物理清理消失 |

- "前端"一节的改写在 P12（3.23）。

---

## 9. 测试

### 9.1 后端单元测试
- **`issue/domain`**（纯函数，表驱动）：字段规则（3.7：标题、日期按合并之后的值、负责人和标签的集合的差集、完成时间、排序值）；父子不成环（给出祖先链）；关联的类型、方向和反向；地址的规整；空评论；表情的格式；动态的行（3.13 的每一行一个用例，含描述的合并、关联的正反两行）；版本的合并（600 秒的边界，同一个人和换人）；提及的取出；只补块 id 的保存是否"内容不变"（内容标识的每一项：纯文本相同而格式不同算不变；文字不同、提及不同或换了顺序、只改链接的地址、只改勾选框的状态都算修改）；编号的折叠（`foldIdentifier`，读 `api/testdata/identifier-fold.json`：大小写、土耳其语的 i、ı、I、İ 四种折成一种，两个编号只差 I 与 İ 时精确匹配优先、否则 404）；列表的词汇（每个页面允许的集合，`IssueListSettings` 的每种 422）；列表的请求的解析（每个参数的每种取值、不合格的 422、重复的参数、`group_by` 与 `sub_group_by` 的组合、`group_key` 的写法和它要的参数）；游标的载荷（每个排序码的往返、排序码不符 400）；全序的比较读 `api/testdata/issue-order.json`（含基本平面以外的字符）。
- **清洗**（S3，P4）：编辑器每种节点的 HTML（由 `@nerve/editor` 的测试导出一份样例）经清洗不变；经接口写入的 HTML（列表、标题、嵌套的标记，没有块 id）经清洗、再经编辑器重新序列化的结果也在样例里，`skip_activity` 的判断对它们答"内容不变"；只改一个链接的地址、只改一个勾选框的两份样例必须答"修改"（对抗性评审 I4）；带脚本、事件属性、`javascript:`、`data:` 的地址、不在放行表的标签的输入被去掉；`description_stripped` 的空白规整。
- **`access/domain`**：判定表加 `AllowCreator`（创建者是成员、是访客、已离开、不是创建者）和 `OwnOnly`（访客、标志开关、不是访客、工作区管理员在项目里是访客）的每一种；`Grant.ProjectAdmin`（决策点 8）。
- **`shared`**：`Date` 的解析和比较；`EventPublisher` 的值；`PageSize`（随移动）。
- **分发器**（`bootstrap/events.go`）：按登记的顺序调用；一个订阅者出错时返回它、后面的不调用；构造时未声明的类型、重复的名字失败。
- **`platform/outbound`**：6.8 的表（IANA 两张登记表的每个网段、本机的接口地址）。
- **`platform/gate`**（11.11）：K 个之内不等；第 K+1 个等到有人离开；等过 `max_wait`、队列满、`context` 到期各返回各自的错误，等待者被移出队列；多个键按升序一次取齐，任一个失败时已取的全部放回（P3 的；P1b 一次一个键，P1b spec 第 3 节第 10 条）；`release` 调两次无害；`-race` 下的并发表。
- **`platform/jobs`**：只投递的客户端的 `InsertTx`；定时任务的种类、声明会投递的种类没有 worker 时 `New` 失败（M2 交接第 6 节：漏注册的变异让它失败，两种各一个变异）；两个队列的 worker 数；测试的余量改成期限（M3 交接第 1 节）。
- **`platform/postgres`**：池里的连接读回 `jit = off`、`plan_cache_mode = force_custom_plan`（3.16）；`TxManager` 的两个选项：`LockTimeout` 的事务里读回 `lock_timeout`、事务结束之后连接上不留它，`ReadOnlySnapshot` 的事务里写被拒绝、两次读看到同一个快照；`LockNotAvailable` 认出 `55P03`。
- **`bodyshape`**：`date` 的检查器（不合格的日期得到带路径的 400）；路径按下标的数值排序（`tags[2]` 在 `tags[10]` 之前）；开放节点里的 `1e400` 得到带路径的 400；代价测试加上字符串的数组（M2 交接第 4 节）。
- **`clock.Offset`**：偏移之后的时刻；非测试环境的配置不为零时启动失败。

### 9.2 权限矩阵（逐格）
**测法**：照 M3 设计 9.2（`bootstrap/permission_matrix_*_test.go`，真实的组合和 Postgres，读的格子共用模板库，写的格子各自复制）。准备的数据加：每个项目里一个由项目成员建的工作项（带评论、链接、关联、版本）、一个由访客建的工作项、一个已归档的工作项；一个"只看自己"的项目（`guest_view_all_features = false`）和一个"可见全部"的项目；每个人一份草稿。新加的身份：建了它的项目成员（`AllowCreator`）、"只看自己"的访客（对自己建的、对别人建的分两列）、在"只看自己"的项目里只是访客的工作区管理员（对抗性评审 M4）。约 480 格，写的约 320 格。

**项目级**（✓ 成功；403 是 `forbidden`；404 是该资源的 not_found 码；"访客·全部"是标志开着的项目里的访客，"访客·自己""访客·别人"是标志关着的项目里的访客对他自己建的、别人建的工作项；"管理员·访客"是标志关着的项目里只是访客的工作区管理员，对别人建的工作项）：

| 操作 | 项目管理员 | 项目成员 | 建了它的项目成员 | 访客·全部 | 访客·自己 | 访客·别人 | 管理员·访客 | 看得到而不是项目成员的工作区管理员 | 已离开项目 | 项目已删除 |
|---|---|---|---|---|---|---|---|---|---|---|
| `listIssues` | ✓ | ✓ | ✓ | ✓ 全部 | ✓ 只有自己建的 | — | ✓ 只有自己建的 | 403 | 404 | 404 |
| `listIssues?archived=true` | ✓ | ✓ | ✓ | 403 | 403 | — | ✓ 只有自己建的 | 403 | 404 | 404 |
| `createIssue` | ✓ | ✓ | ✓ | 403 | 403 | — | ✓ | 403 | 404 | 404 |
| `getIssue`、`getIssueByIdentifier`、`listSubIssues`、`listIssueRelations`、`listIssueLinks`、`listIssueComments`、`listIssueReactions`、`listIssueActivities` | ✓ | ✓ | ✓ | ✓ | ✓ | 404 | 404 | 403 | 404 | 404 |
| `updateIssue`、`addSubIssues`、`createIssueRelations`、`deleteIssueRelation`、链接的写 | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 403 | 404 | 404 |
| `deleteIssue` | ✓ | 403 | ✓ | 403 | 403 | 404 | 404 | 403 | 404 | 404 |
| `archiveIssue`、`unarchiveIssue` | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 403 | 404 | 404 |
| `createIssueComment` | ✓ | ✓ | ✓ | ✓ | ✓ | 404 | 404 | 403 | 404 | 404 |
| `updateIssueComment`、`deleteIssueComment`（作者本人） | ✓ | ✓ | ✓ | ✓ | ✓ | 404 | 404 | 403 | 404 | 404 |
| `updateIssueComment`，别人的评论 | 403 `issue.not_author` | 403 `issue.not_author` | 403 `issue.not_author` | 403 `issue.not_author` | 403 `issue.not_author` | 404 | 404 | 403 | 404 | 404 |
| `deleteIssueComment`，别人的评论（决策点 8，负责人裁定 B） | ✓ | 403 `issue.not_author` | 403 `issue.not_author` | 403 `issue.not_author` | 403 `issue.not_author` | 404 | 404 | 403 | 404 | 404 |
| 表情的加、去 | ✓ | ✓ | ✓ | ✓ | ✓ | 404 | 404 | 403 | 404 | 404 |
| `listDescriptionVersions`、`getDescriptionVersion` | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 403 | 404 | 404 |
| `getProjectPreferences`、`updateProjectPreferences`（三个新字段） | ✓ | ✓ | ✓ | ✓ | ✓ | — | ✓ | 403 | 404 | 404 |

- "建了它的项目成员"一列只在 `deleteIssue` 上与"项目成员"不同；另有一行"建了它、后来被降为访客"（403）和"建了它、已离开项目"（404）。
- "管理员·访客"一列核对访客规则按项目的角色判（3.4，`access/domain/decide.go:76,81`）：工作区管理员只要有一个有效的项目成员关系就通过每一条项目的规则（`Grant.ProjectAdmin`），访客规则仍按他在项目里的角色让他只看自己建的，别人建的一律 404（404 优先于规则）。所以创建、已归档的列表（只有自己建的）是 ✓；他自己建的工作项上的格子与"项目管理员"一列相同（改、删、归档、描述的历史，和删除别人在上面的评论，决策点 8），另有一行核对。
- "项目已删除"一列在删除的任务完成之前和之后各跑一次（决策点 7）：两次的回答相同。
- 写的格子另有"工作项已归档"（409 `issue.archived`）和"项目已归档"（409 `project.archived`）两种前置，每个写操作各一格。

**工作区级**：

| 操作 | 管理员 | 成员 | 访客 | 从来不是成员 | 已被移出 | 工作区已删除 |
|---|---|---|---|---|---|---|
| `listDrafts`、`createDraft` 和自己的草稿的读、改、删、发布 | ✓ | ✓ | 403 | 404 | 404 | 404 |
| 别人的草稿 | 404 | 404 | 403 | 404 | 404 | 404 |
| `publishDraft`，草稿的项目里他是访客 | — | 403 | 403 | 404 | 404 | 404 |
| `listMemberIssues`（别人的主页） | ✓ 他可读的项目里的 | ✓ 同左 | 403 | 404 | 404 | 404 |
| `searchIssues` | ✓ 可读的 | ✓ 可读的 | ✓ 可读的，按访客规则 | 404 | 404 | 404 |

- 测试另外核对：`issue.yaml` 的每个操作在表里有一行；`issue/domain/actions.go` 的 `Actions()` 与规则表的键一一对应（M3 的完整性测试）。

### 9.3 集成测试（`pgtest`）
- **每个用例**：在真实的 Postgres 上核对它写的每一行（主表、负责人、标签、关注、提及、动态、版本、`updated_at`），时刻等于固定时钟（M2 交接第 3 节）；事件经假的发布者核对。经接口准备不了的格子在这里经仓储准备：分诊状态的 `state_id` 得到 422（`createState` 拒绝分诊，对抗性评审 M17）。
- **交错**：3.5 的 26 种，照 M3 的写法（两个连接，用 `pg_locks` 等到对方阻塞再放行，确定性）；两种顺序都跑。取锁的形状（交错 21）照 M3 的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（`bootstrap/project_write_locks_test.go:265`）改写：每个工作项的写等第一把锁时在工作区行上什么都不持，第一把锁是项目行的 N，之后不取与 `FOR NO KEY UPDATE` 冲突的工作区行的锁（外键的 `FOR KEY SHARE` 不算）；看不到项目的账户、看得到而不能写的账户的写不取行锁、不写，加关联指向一个看不到的项目时不锁它（预检，对抗性评审 M1；这一例是交错 21 的关联的一例，自己的测试 `TestARelationLocksNoProjectItDoesNotReach`，与取锁形状的测试的 `createIssueRelations`、`deleteIssueRelation` 两行并列）。M3 的那个测试改为只覆盖项目管理类的写。
- **饥饿与连接池**（交错 20，S5、SP1 的回归，经真实的连接池和写闸门）：16 个连接首尾相连地写同一个工作区的几个项目的工作项，期间改工作区、改成员的角色在 1 秒之内完成；删除工作区等到的只是每个项目上正在进行和闸门放进来的写（每个项目至多 K 个）。另 200 个写同时打到一个项目，另一个项目上的读的耗时不超过一个上限（第二稿量到约 875 ms，上限在 P1b 的 spec 按实测定）；闸门满时 503 `server_busy` 带 `Retry-After`；把 `issue.write_lock_timeout` 设小，等 N 超过它的写得到 503 而不是一直等（11.11）。
- **连带**：删除工作项连带子树和每一张子表（同一时刻、同一个删除者，按外键的图从 `issues` 传递地找表，评论的表情一类也在内）；删除状态在用 409；删除标签软删除 `issue_labels`；连带失败时发起者的写一起回滚。
- **删除项目、工作区**（决策点 7，负责人裁定 A，P9；P1–P8 是同步的过渡写法，测同样的终态）：请求的事务只做 M3 的连带并经事件为每个删除的项目投递一个 `issue.delete_rows`（删除工作区另加没有项目的草稿一个；更早删除的项目不在其中），回滚时不留任务；任务把 `issue` 的 13 张表（含跨项目的关联和没有项目的草稿）的行软删除，`deleted_at`、`updated_by_id` 是删除它的那次请求的时刻和删除者，跨项目的关联行属于先删除的那一端；任务重做无害（谓词 `deleted_at IS NULL`），重试从每张表第一个未删除的 `id` 开始；预算用完时返回 `JobSnooze(0)`，River 的尝试次数不增加（把预算设小，看任务分几次运行做完、`attempt` 不变）；任务做完之前与之后，3.6 列出的每个读（按编号取、搜索、列表、个人主页、别的项目的关联、引用它的动态、草稿、游标）回答相同，每个写（加关联、发布草稿、删除关联、删除另一端的工作项）的结果相同；交错 24、25；任务在清理之后才执行时（项目行已物理删除）什么也不做。
- **规模的目标**（决策点 7）：P1b 在 2 万个工作项（每个 20 条动态）的项目和 5 个这样的项目的工作区上量删除的耗时（P1a 写下的同步的过渡写法，P1b 不改它），记进 P1b 的 spec。P9 是一条通过线：同样的工作区，每个任务在一次运行里做完（没有 snooze、没有重试），每个任务的耗时和期间同一工作区别的项目上的写的等待记进 P9 的 spec。
- **访客规则的一致性**（3.4）：对 9.2 的每种身份（含"管理员·访客"），列表、个人主页、搜索的结果等于逐个判定 `issue.read` 加访客规则的结果；"只看自己"的访客看到的 `sub_issues_count` 只数他自己建的，回答里的 `parent_id` 不指名他看不到的工作项（3.4）；他的动态列表里没有引用看不到的工作项（别人建的父工作项、看不到的项目里的关联对方）的行；被提及的、读不到这个工作项的访客没有关注它。
- **动态引用的工作项**（3.13，对抗性评审 M8）：调用者在被引用的工作项的项目里是有效成员时显示，与那个工作项、项目是否已删除、已归档无关；已离开那个项目的不显示；被引用的工作项已物理清理时不显示。三种各一个测试。
- **列表引擎**：每个排序码两个方向各一个翻页测试（数据里有同键、空值，翻完不重复、不遗漏、顺序与 `issue-order.json` 的比较相同）；每个分组码的第一页和每组的总数（多值的组、`none` 组、子分组的格、每个子分组的总数 `subgroup_totals` 不重复计算一个在两列里的工作项）；一组（一格）的"加载更多"带 `group_key`（`sub_group_key`）翻完不重复、不遗漏，`total` 等于组的 `total`，含"`label_id=A&group_by=label_id` 的组 C"和 `target_date` 的 `none` 组；分组的请求带 `cursor` 而不带 `group_key` 得到 422；每个筛选参数（含 `none`、`me`、`target_date=none`、`start_date=none`、可重复的日期、单日）；数组参数为空（不筛选）与为 `nil` 的仓储都得到全部行；空的一页的 `total`；一次读在一个只读的 `REPEATABLE READ` 事务里（在读目录与读行之间另一个连接建一个状态、把工作项移进去，这次的应答仍自洽：行和总数出自同一个快照）；篡改的游标（8.6）；1 万、10 万个工作项的耗时（S1 的回归，上限取 3.16 的实测加余量）；`assignee_name` 的排序对已离开的负责人有键。
- **事务内投递**（S4）：经只投递的客户端，提交的事务留下 `issue.crawl_link`、`issue.delete_rows`，回滚的不留；`registrationRows`（`bootstrap/table_rows_test.go`）的"被拒的注册不写任何表"另核对 `river_job` 里没有由请求投递的任务（按种类，M3 交接第 1 节）。
- **定时任务**：自动归档（推进固定时钟：30 天减一秒不归档、加一秒归档；状态不对、项目没设、项目已归档的不动；动态的执行者为空；事件）；版本的保留（P9：第 21 个被删；已删除的工作项、已删除的项目里的版本不动）；物理清理（61 天的删、59 天的留；带动态和评论的工作项、PAT、标签、项目、工作区；`CASCADE` 带走子行；四个模块按顺序；M2 交接第 5 节）；`bootstrap/jobs_test.go` 等每个种类在 `river_job` 中 `completed`（M2 交接第 6 节的另一半）。
- **清理的目录**（3.21）：每张有 `deleted_at` 的表都有清理者；每条指向被清理的表的外键是 `CASCADE` 或 `SET NULL`。
- **抓取**：`httptest` 的页面（标题、图标、跳转、超大、超时）经 `outbound.allow_cidrs` 放行；写回的守卫（链接已删除、地址已改时 0 行）；失败只写 `blocked`、`failed`。
- **编号的折叠**：`İŞ-12`、`iş-12`、`IŞ-12` 取到同一个工作项；只差 I 与 İ 的两个项目并存时精确匹配的那个胜出。
- **授权**：`TestTheGrantsFileCoversEveryRelationAndFunction`、`TestTheRuntimeRoleServesWithTheGrantsFile` 在每个建表的 Phase 通过（M2 交接第 6 节）。

### 9.4 契约测试和整程序测试
- `apitest.Main`（M2 设计 3.11）：`issue.yaml` 的每个码两个方向（声明的都被 `issue` 自己的 HTTP 测试返回过，含经假实现返回的 `project.*`、`workspace.*`）；前缀规则（M3 设计 11.7）。
- `bootstrap/contract_test.go` 的七个整程序测试自动覆盖新操作：路由、默认拒绝（401）、请求体的结构（含数组里的元素、日期的格式）、参数绑定（路径的 UUID、`limit`）、错误响应的头。
- 草稿发布与创建的同一个 400（M2 交接第 4 节）：同一个不合格的请求体，`createIssue` 与先存为草稿再 `publishDraft` 得到相同的 400（状态、码、问题的路径）。
- 前端的生成的类型与契约同步（`make gen-web` 之后没有差别，M2 的门禁）。

### 9.5 前端单元测试（vitest）
- `core/lib/reconciled.ts` 的 `fetchPart`：值还没取到时不做；被整体的取数超越、被同一部分更新的取数超越时不写；在外期间确认的修改在合并之后按确认的顺序重做，重做拿到这次的应答；会话改变。
- `core/lib/paged-groups.ts`：3.22 的表（`(列表当前的值, 修改或读到的版本, 应答?) → 期望的值`）：旧属、新属的四种组合各配在、不在 `rows`，窗口内、外（固定的 `boundary`），载入完与否；旧属先按这一份列表的行的 `updatedAt` 判断、再按 `beforeSnapshot`；新建（`beforeSnapshot` 为无）、删除（`after` 为无）、归档和恢复；没有回答的组、格按空的建出；多值的组；子分组的格和 `subgroupTotals`；最外层的总数；拖放的落点（窗口外扩 `boundary`、空的列、空的泳道格、日历已满的一天）；重做的三种判断（应答里有它的行、应答覆盖了它的位置、看不出）；每种 `stale`；对抗性评审 I2 的九个反例各一行；聚焦复核 M3 的五行（重做时旧属只来自应答和 `beforeSnapshot`、并入取应答的 `total`、第 2 条之后的 `getIssue`、`boundary` 只往后移、自己的修改在路上时不标 `stale`）。
- 一个工作项的那一份（`ReconciledByKey`）：服务端的 `updated_at` 大的胜出、相等时后应答的胜出；列表的行不覆盖 `description_html`；读到的新版本改了成员关系的字段时，含它的列表里它旧、新两个版本的组标 `stale`，这个会话自己有修改在路上的工作项除外。
- 拖动的 `sort_order` 在轮到发出时按那时的列表算：前一个拖动确认、移动了相邻的卡片之后，后一个按新的位置算。
- 一组的"加载更多"带 `group_key`（`sub_group_key`）和这一组的游标，应答只并入这一组，`total` 取应答的，`boundary` 取 `max(boundary, 它的最后一行)`；拖放留在窗口外的卡片在后面的页到来时按 id 去重。
- 子工作项在浏览器里的筛选（`getFilteredWorkItems`，含相对日期 `fromnow`）在重写 `base-issues-utils.ts` 之后行为不变。
- 全序的比较读 `api/testdata/issue-order.json`，逐个码位比较（基本平面以外的字符与 U+E000–U+FFFF 的一对）。
- 筛选的表达式翻译成 `IssueFilters`（每种条件）；`IssueFilters` 拼成查询串。
- 每个修改一张表（M3 的写法）：会话改变时不写、被拒绝时不写并交回、确认之后写进每一份列表和工作项的那一份。
- 7.1 的各项：`autorun` 的替代、按需取数被拒绝、宿主卸载时关上开关和重设数据、`WorkItemLevelModals` 在 `projectId ?? workItemIdentifier` 改变时关上开关、`followInSession` 的跟进。
- 描述保存的"撤销再重做"（M1-P2 交接）；工作项的弹窗编辑已有的工作项、打开草稿时读 `isMigrationUpdate`，不把表单标成已改（M1 收尾交接）。
- 个人主页的路由段收窄（`TProfileViews`）。

### 9.6 端到端
- **fixture**：`e2e/fixtures/` 下按资源分的文件加工作项、子工作项、关联、链接、评论、表情、草稿的帮助函数（生成的客户端，PAT；P1a 起有 `issues.ts`、`states.ts`，`furnish.ts` 放 `furnishIssueRows`、`furnishWorkspace`）；`assert/issue.ts`、`assert/history.ts` 按表的断言函数，页面版本和接口版本共用；`expectIssueDeleted`（3.6，沿外键图递归）；`workspaceTables` 从目录读出（`assert/catalog.ts`，P1a 起；M4 的表自动在内）；`db.ts` 加这些表的读取；`fixtures/clock.ts`（3.20：自己的数据库、自己的 nerve，偏移之后重新登录，只认偏移的 nerve 启动之后完成的任务）；本机的页面服务器（I10，`127.0.0.1`，`config.test.yaml` 放行）。
- **`furnishIssueRows(api, token, projectId, assigneeId, labelId)`**（`e2e/fixtures/furnish.ts`；对抗性评审 I6，2 节）：给一个项目的每张工作项一侧的表造至少一行；P1a 起每个加表的 Phase 扩充它，并在同一个 Phase 让 M3 的删除故事（P4、W2、W3）在删除之前调它，所以 `expectProjectDeleted`、`expectWorkspaceDeleted` 每一次都断言到新加的表。P9 起这两个断言对工作项一侧的表限时轮询（决策点 7）。M5、M6 加表时照样扩充它（13.2）。
- **故事**：第 2 节的 22 个；S1、S2 的更新（S2 在 P10b 起打开项目的列表，P14 起按身份打开一个工作项的详情：成员看到，"只看自己"的访客得到 404、公开项目里不是项目成员的工作区成员得到 403 的"没有权限"，都按请求放行、不留别的失败的请求）。每个故事的接口版本和页面版本在第 2 节标出的 Phase 加入，之后按"断言的增长"一列在后面的 Phase 扩充。
- **会话切换**（I22，P14）：M3 的 W3 的写法（`route.fetch()` 先到达 nerve，换账户之后 `route.fulfill()`）；变异核对：去掉 `followInSession` 的核对，这个故事必须失败。
- **后退的检查**（7.1）：开着弹窗、浮层后退到另一个项目、工作项或人，它们不再开着（P13、P14）。
- **键盘、名字、焦点**：I1 的嵌套弹窗的 Escape；字段的提示与焦点（`recordFocuses`）；弹窗里的列表打开时 `pageBehindHidden`；筛选按钮的名字和 `aria-expanded`（7.5）。
- **`EMOJI_CHECK_WARNING` 删除之后**，S2 和个人设置的故事仍断言控制台没有别的警告（M2 交接第 7 节）。

### 9.7 控制者的浏览器核对
| 编号 | 内容 | Phase |
|---|---|---|
| C1 | 列表、看板（泳道）、日历、表格：分组、每组加载更多、拖动（同组排序、跨组、放在两张之间）、被拒绝时卡片回到原处、放在一列最后的卡片确认之后留在那里；日历的"添加已有"只列没有截止日期的（3.18）；Chromium、Firefox、WebKit 各一遍 | P12 |
| C2 | 筛选的工具栏和显示设置：每种筛选、日期的两个范围、清空；刷新之后还在；`FiltersDropdown` 的名字和 Escape 之后的焦点；`CustomSearchSelect` 的 `defaultOpen` 和 4 个富筛选的调用方（M2 交接第 10 节） | P13 |
| C3 | 7.5 的下拉框：`DateDropdown` 用键盘打开、只有一份状态；7 个下拉框的列表在按钮旁；在弹窗里打开时后面的页面隐藏；`CustomSelect` 的 3 个调用方（自动归档、日期筛选、浮层的页头）；成员下拉框的 8 个文件 | P12、P14 |
| C4 | 工作项的弹窗：Tab 在里面转、Escape、点外面、焦点回到打开它的控件；嵌套的"Save this draft?"和选父工作项的 Escape；描述编辑器的 Escape 不再误关；弹窗里 portal 的日期、下拉框、表情能用；`ModalTitle` 的名字；在弹窗里编辑一个描述没有块 id 的工作项、打开一份这样的草稿，不做改动就关上，不问"是否保存"（`isMigrationUpdate`）；Firefox、WebKit 也看一遍焦点 | P14 |
| C5 | 详情：属性的每个下拉框、子工作项（添加已有、新建、移走）、关联（跨项目）、链接（元数据出现）；快捷操作的子菜单用键盘打开；层叠顺序（开着弹窗时页面上的元素不画在它之上）。表情、时间线、描述的历史在 C6（它们的页面在 P15，对抗性评审 M19） | P14 |
| C6 | 评论：发、改（"已编辑"）、删、表情；每条评论的控件以作者为名（读屏）；只有作者有编辑，作者和项目管理员有删除（决策点 8）；空评论不能发；打开表情回应的选择器时 Network 面板没有 `cdn.jsdelivr.net`、控制台没有 CSP 违规（M2 交接第 8 节）。工作项的表情、时间线（排序、只看评论或动态）、描述的历史和还原 | P15 |
| C7 | 命令面板：工作项的命令、搜索（编号、标题）、主题命令（失败时不改主题）；在中文界面下 | P15 |
| C8 | 复制：工作项链接、评论链接、编号、子工作项和关联的链接，在局域网 HTTP 地址上失败时有提示（M2 交接第 11 节、M3 交接第 8 节） | P15 |
| C9 | 草稿：存、改、删、移到项目；归档页；个人主页的三个标签页（成员、访客）；访客在"只看自己"的项目里 | P13、P15 |
| C10 | 标注块：Network 面板没有 `cdn.jsdelivr.net`，控制台没有 CSP 违规；未登录打开详情的地址不先加载编辑器（M2 交接第 7、8 节）。表情回应的同一项在 C6（P15，表情的页面在那里） | P14 |
| C11 | 两个账户两个标签页：在工作项页上切换账户，旧的一代不再写入（7.1） | P14 |
| C12 | 全部 M4 的页面在 Chromium、Firefox、WebKit 下各走一遍主路径；焦点的轮廓不被裁掉；右键菜单（M3 交接第 7 节） | 收尾 |

---

## 10. 负责人决策点

下面八项改变用户能做什么、能看到什么，或改变上级文档承诺的范围、M4 的交付方式，而 Plane 和上级文档都没有定下。八项都已由负责人裁定（2026-10-17），都选了建议的选项；选项和代价留在下面备查，正文各节都已按裁定写定。第三稿加了两项：删除项目、工作区时工作项一侧的行何时删除和 v0 的规模目标（决策点 7，对抗性评审 I1、M15），评论的管理（决策点 8，对抗性评审 M14，第二稿把它当作按页面定下的一件）。

下面几件按 Plane 已有的规则或页面裁定，没有交负责人（理由在各节）：
- **删除工作项连带整棵子树**（3.6）：Plane 相同；确认框写明。另一种（子工作项脱离父工作项）是 Plane 没有的行为。
- **一对工作项最多一条关联**（3.10）：页面本来就不让造出第二条，按页面收紧服务端。
- **已归档的工作项只读**（3.8）：页面全部禁用，按页面收紧。
- **自动关注不含"每个改过的人"**（3.12）：Plane 的副作用，不是它的规则；总体设计 6.6 列的是"创建人、负责人、评论人"。
- **草稿发布用存着的载荷**（3.15）：总体设计 5.4 已定。
- **抓取的上限**（3.11）：Plane 缺的防护，补上不改变用户能做的事。
- **筛选不支持否定**（3.22）：页面生成不了（M1-P3 交接让 M4 定，结论是删掉那一层）。

### 决策点 1：工作项的写取哪一把锁（M3 设计 17.4；M3 交接第 2 节）
**负责人裁定（2026-10-17）：B，工作项的写只锁项目行。** 落在 3.5、3.6、6.3、6.5、6.7、9.2、9.3（交错 3、4、5、20、21）、11.2、11.11。

**背景**：
- M3 的项目级的写最先以 `FOR SHARE` 锁住工作区行（方案 E，负责人 2026-10-02 的裁定），为的是与工作区一级的连带互斥、让"一个改变一个时刻"成立。17.4 的探针 B 测到它的代价：新的 `FOR SHARE` 越过等待中的 `FOR NO KEY UPDATE`，连续重叠的项目级的写可以让工作区一级的写一直等到请求期限。负责人当时写明"M4 的工作项写入不默认沿用"。
- 工作项的写比项目管理类的写频繁得多。`authenticated` 的限速是每个凭证每分钟 1200 次（每秒 20 次），一个忙的团队、或一个循环调用接口的 PAT，就能让工作项的写首尾相连。
- 改工作项一侧的行的连带只有删除项目、删除工作区两种（3.6）。

| 选项 | 对用户 | 能防住什么 | 对代码 |
|---|---|---|---|
| A 照方案 E：工作区 S → 项目 N | 与 M3 一致 | 时刻不倒退。但连续的工作项写可以让改工作区、删工作区、改角色、接受邀请、移出、停用一直等到请求期限（15 秒），管理命令一直等 | 不改 M3 的约定；要另外证明 M4 的负载下不饿死，或加限流 |
| **B 只锁项目 N；删除工作区先按 `id` 锁住它的项目（建议）** | 最频繁的写（工作项和它下面的行）不再挡住工作区一级的写；删除工作区最多等每个项目上已在数据库里的写（写闸门之下每个项目至多 K 个，11.11） | 每一行的时刻不倒退：两种连带都与工作项的写在项目行上互斥（3.5）。N 与 N 排队，不越过等待者。**代价**（3.5）：一、工作项的写与成员关系的变更之间，时刻可以与生效的顺序不一致（按旧角色判定、时刻 `t2` 的写，挨着在 `t1 < t2` 结束的成员关系）；二、草稿的写、发布草稿和 M3 的全部项目管理类的写仍先取工作区 S，探针 B 的饥饿在这些较少的路径上仍在 | 删除工作区多一步、读时刻的位置后移；工作项的写与 M3 的项目级的写是两种第一把锁（各自的理由写在 3.5）；锁之前一个不加锁的可见性预检 |
| B′ 同 B，另让连带先锁项目、再读时刻（不建议） | 同 B | 去掉连带一侧的代价一：`DemoteToGuest`、`EndMemberships` 先锁成员的行（N）、列举、锁项目，然后才读时刻，与项目上的工作项的写按生效的顺序得到时刻。只改工作区角色（不降为访客）的写仍不锁项目，那一种仍在 | 改写 M3 已批准的约定六的收缩路径的顺序，和删除、离开、停用这几个用例（它们今天在工作区 N 之后就读时刻）；换来的只是审计时刻的一致 |
| C 工作区 S → 项目 S | 同一项目里的写可以并行 | 时刻不倒退；仍有 A 的饥饿；同一项目里的取号、不成环、合并都要另加锁 | 每个不变式一把专门的锁，约定更多 |

- **证据**（S5，0.3，PG 18.6）：E1 复现探针 B：60 ms 一个、每 30 ms 新来一个的 `FOR SHARE` 持续 3 秒，工作区的 `FOR NO KEY UPDATE` 等了 2.753 秒（方案 A 之下，工作项写首尾相连就是这个形状）。E2：N 的等待者先来先得，3 秒首尾相连的工作项写（B）之后，删除工作区只等了 319 ms。E3：B 之下工作项写的链进行中，改工作区最长 2 ms；插入经外键取的 `FOR KEY SHARE` 与 `FOR NO KEY UPDATE` 不冲突。
- **理由**：B 同时满足两件事：每一行的时刻不倒退（M3 交接第 2 节要求说明的"不出现 F-M3 那样的时刻倒退"），最频繁的写不再饿死工作区一级的写（M3 17.4 的代价不扩大到工作项的写）。代价一只影响"不同的表之间的时刻是否按生效的顺序"这一种审计的读法，授权本身在锁下判定、不会过期；代价二留在本来就少的路径上。B′ 能去掉连带一侧的代价一，但要改一条已批准的约定和三个 M3 的用例，所以不建议；负责人若要审计的时刻一致，选 B′（只改角色的那一种另要改角色的写也锁项目，在 B′ 的 spec 里定）。同一项目里的写串行是 N 的代价，每个写是一个短事务；一个项目里每秒数十个写仍够用。
- **连接池的代价，A、B 相同**（对抗性评审 I3，SP1）：等锁的写持着池里的一个连接（池是 10 个）。一个凭证的突发额度（200 个请求）打在同一个项目上时，A、B 之下都是 N 上的一条队，别的项目上无关的读排在池的等待者后面约 875 ms；40 个时约 141 ms。所以不论选哪个，都要 11.11 的写闸门：每个项目每个进程至多 K 个写在数据库里，其余在进程里等、不持连接，等不到就 503 `server_busy`（另有 `lock_timeout` 兜底）。它的代价：热的项目上超过 K 的写排队，排得太久的得到 503、要由客户端重试，而不是一直等到请求期限；多一个约 100 行的平台包、一个端口、每个写用例多一步（3.5）。选 A 时闸门同样必要（S 与 S 不互斥，等的是工作区一级的写，但写和写之间的 N 仍排队）。
- **测试**：交错 21（每个工作项的写等第一把锁时在工作区行上什么都不持，第一把锁是项目的 N；看不到项目、不能写的账户不取锁）；交错 20（饥饿和连接池：经真实的连接池和闸门，首尾相连的工作项写不延迟工作区一级的写，一个热的项目不拖慢别的项目的读，S5、SP1 的回归）；交错 3（删除工作区与修改工作项的两种顺序）；交错 4、5（移出、降级与修改工作项）。
- **以后改的代价**：B → A 只改取锁的一步和测试，但要重新面对饥饿；B 之下若某个项目真的成为热点（每秒上百个写），再按行细分锁，是 3.5 的改写。

### 决策点 2：M4 作为一个里程碑交付，还是拆成两个
**负责人裁定（2026-10-17）：A，一个 M4。** 落在第 12、15、16 节。

**背景**：M4 的后端约 37 个操作加列表引擎、事件、River、清理；前端是 M3 的 2.5 倍文件（564 个、54,644 行，R6 第 1 节），估计约 95 个前端任务（7.10），第 12 节排出 106 个。M3 是 45 个操作、约 55 个前端任务，用了 13 个 Phase 加收尾。

| 选项 | 对用户 | 对交付 | 对代码 |
|---|---|---|---|
| **A 一个 M4，后端 → 前端的数据层 → 页面 → 收尾，17 个 Phase 加收尾（建议）** | M4 结束时工作项一次可用 | 约 241 个任务；控制者遇到超过上限的 Phase 时直接拆成 P<k>a、P<k>b 继续，不停下来问，记进第 15、16 节（预先批准） | 一份设计、一份交接、一次收尾 |
| B 拆成 M4a（后端和接口版本的故事）和 M4b（前端和页面版本的故事） | M4a 结束时只有接口可用，页面上的工作项要等 M4b | 两次收尾、两份交接；M4a 的收尾不能核对页面上的任何故事 | 设计拆成两份，交接里多一份 M4a → M4b |

- **理由**：M3 的经验是数据层和页面之间的交接最贵（P8 拆成 P8a、P8b，收尾远超原型）；拆成两个 M 把这段交接变成跨 M 的交接，代价更大。后端和前端已在 Phase 上分开，评审一次只看一段。
- **以后改的代价**：选 A 之后若中途要拆，在某个 Phase 的边界上收尾即可（后端的 Phase 在前），成本是一次额外的收尾。

### 决策点 3：现在引入领域事件，还是让用例直接投递任务（M2 设计 3.15；M2 交接第 6 节；M3 交接第 1 节）
**负责人裁定（2026-10-17）：A，现在引入。** 落在 3.19、3.20、6.4、11.4、P6、P7、P9。

**背景**：总体设计 6.3 第 3 条说"跨模块的副作用（通知、Webhook、动态）经领域事件"，典型的订阅者往 River 投递任务；M2、M3 都把"事件怎样写"留给第一个需要它的 M。M4 的第一个异步副作用是链接的抓取，只在 `issue` 模块之内；M7 的通知、M8 的 Webhook 都要订阅工作项的每一种变化。

| 选项 | 对用户 | 对以后的 M | 对代码 |
|---|---|---|---|
| **A 现在引入：`shared` 的事件和发布端口，用例在事务里发布每一种变化，分发器在组合根（建议）** | 无可见的不同 | M7、M8 只加订阅者，不改 M4 的用例 | `shared/event.go`、`bootstrap/events.go`、每个写用例的发布点（约 20 处）；P6 时还没有生产中的订阅者（3.19）；P9 起删除项目、工作区也发布事件，工作项一侧的删除任务由订阅者投递（决策点 7） |
| B 不建事件：抓取由链接的用例经端口直接投递 | 无可见的不同 | M7 引入事件时回头改 M4 的每个写用例、补测试 | 少一个分发器、少约 20 个发布点；决策点 7 的任务由 `project` 经端口 `IssueCascade.ScheduleRowDeletion` 投递（3.19） |

- **理由**：M4 是定义"一次改动是什么"的地方（3.13 的动态就是它的一种投影），现在把事件和动态一起定下最便宜；M7、M8 回头改 M4 的约 20 个用例，要重新评审它们的事务和锁。同步、在事务里的分发不引入新的一致性问题：`InsertTx` 让任务与改动同生同灭（3.19）。
- **以后改的代价**：A → B 删掉发布点即可；B → A 是上面说的回头改。

### 决策点 4：自动的动作记在谁名下（M1-P5 交接）
**负责人裁定（2026-10-17）：A，`actor_id` 为空表示 nerve 自己。** 落在 3.8、3.13、3.19、4.11、7.7。

**背景**：v0 里只有自动归档是 nerve 自己做的动作（自动关闭已砍）。Plane 把它记在项目的创建者名下（`issue_automation_task.py:73`），页面按 `new_value === "archive"` 把执行者显示为站点名。Nerve 不区分人和代理、不建系统账户（M1-P5 交接：不要按邮箱或名字猜系统用户）。同一个值会流进 PAT 的读者、M7 的通知、M8 的 Webhook。

| 选项 | 对用户 | 对 PAT 和以后的 M | 对代码 |
|---|---|---|---|
| **A `actor_id` 为空表示 nerve 自己（建议）** | 时间线显示"Nerve 归档了工作项"（与 Plane 的页面相同） | 读者看到 `actor_id: null`；事件、Webhook 同样 | 页面的通用规则：执行者为空显示站点名；不再按值特判 |
| B 照 Plane：项目的创建者 | 页面靠 `new_value` 特判显示站点名 | 通知和 Webhook 里一个人"做了"他没做的事；创建者被删除之后又是空 | 不改 |
| C 为空，另加 `actor_kind`（`account`、`system`） | 同 A | 不靠"空"的含义，契约明确 | 动态、事件多一列（一个字段）；与 Plane 的表多一处不同 |

- **理由**：v0 不物理删除账户（M2 设计 4.7），`actor_id` 的 `SET NULL` 永不触发，"空"只有一种含义，A 已经无歧义；C 多出的一列在 v0 里没有第二个取值的用处。
- **收集箱的来源和代为创建的工作项**（M1-P5 交接的另一半）：属于 M7（收集箱在 M7，总体设计 9.2），转交（13.2）；M4 删掉动态里读来源的分支（7.7）。
- **以后改的代价**：A → C 加一列和一个字段，存量的空值都是 `system`；物理删除账户若在 v0 之后出现，那时再区分。

### 决策点 5：工作项快照（`issue_versions`）做不做
**负责人裁定（2026-10-17）：A，不做。** 落在 3.14、4.14、3.23（总体设计 1.1、5.2、6.6）。

**背景**：总体设计 1.1 承诺"历史版本：工作项快照、描述修改历史"，6.6 写"同时写 `issue_versions`"。Plane CE 有这张表（33 列）和读取的接口，但运行时没有任何代码写它（唯一的写入方没有调用者），页面也不读（R2 第 9 节、R1 第 12 节）。

| 选项 | 对用户 | 对代码 |
|---|---|---|
| **A 不做：只有描述的历史（建议）** | 能看、能还原描述的历史（Plane 的页面就是这样）；没有整个工作项的快照 | 不建表；总体设计 1.1、5.2（45 → 44 张表）、6.6 改写 |
| B 做：每次修改写快照，可查看、可还原 | 多一个"工作项历史"的页面 | 要自己定义触发、合并、保留、读取的接口、页面和还原的语义，Plane 都没有；约多一个 Phase |

- **理由**：快照在 Plane 里是一张没有写入方、没有页面的表；照搬"删掉没人用的"的原则，它就不该存在。动态（3.13）已经记下每个字段的每次改动。
- **以后改的代价**：B 以后可以加回：一张表、一个写入点（`issue_writes.go`）、一个读取的接口和页面，与 M4 的其余部分无关。

### 决策点 6：评论的回复做不做
**负责人裁定（2026-10-17）：A，不做。** 落在 3.12、4.7、3.23（总体设计 1.1）。

**背景**：总体设计 1.1 写"评论（支持回复）"。Plane 的模型有 `parent`，但 CE 的页面没有回复的界面（`enableReplies` 收下了但从不读），服务端也没有任何回复的规则（R2 第 5 节）。

| 选项 | 对用户 | 对代码 |
|---|---|---|
| **A 不做（建议）** | 评论是平铺的（Plane 的页面就是这样） | 不建 `parent_id`；总体设计 1.1 改为"评论" |
| B 做 | 能回复一条评论 | 要自己定义层数、列表的形状、删除的连带、通知（M7），并从头写页面；约多半个 Phase |

- **理由**：这是 Nerve 的新功能，不是对 Plane 的保留；v0 的目标是保留 Plane 有的。
- **以后改的代价**：加一列（`parent_id`，`CASCADE`）、一个规则、页面；与 M4 的其余部分无关。

### 决策点 7：删除项目、工作区时工作项一侧的行何时删除；v0 的规模目标（对抗性评审 I1、M15）
**负责人裁定（2026-10-17）：A，工作项一侧的行由后台任务删除；规模目标是一个项目至多约 2 万个未删除的工作项。** 落在 3.5、3.6、3.15、3.19、3.20、3.21、3.23（总体设计 5.5）、6.2、6.3、6.6、8.8、9.2、9.3（交错 24、25）、9.6、P1a、P1b、P9。

**背景**：
- 总体设计 5.5 写"连带删除在同一个事务里同步完成"，M3 的删除项目、删除工作区照此做。工作项一侧加进来之后，一个事务要软删除的行随工作项的数量线性增长：每个工作项平均约 33 行（动态、负责人、标签、关注、评论、描述的版本……）。
- **规模目标**：v0 承诺一个项目至多约 2 万个未删除的工作项（3.16，按独立评审的裁定定下；S1 量了 1 万、10 万）。它是对部署者的承诺，也决定这一项该怎么做，所以与这一项一起交负责人（对抗性评审 M15）；更大的项目是 M8 的性能工作。
- **SP2**（0.3，PG 18.6，按 ±2–3 倍读）：一个 2 万个工作项的项目约 66 万行，在一个事务里软删除 8.6 秒；五个这样的项目的工作区 330 万行，36.5 秒。请求的期限是 15 秒（`server.request_timeout`）。删除工作区先锁住它的每个项目（决策点 1 的 B），这段时间里这些项目上的每个写都在等；到期时事务回滚、什么也没删，重试是同样的结果。

| 选项 | 对用户 | 在目标的规模上 | 对代码 |
|---|---|---|---|
| **A 请求只做 M3 的连带，工作项一侧的行由后台任务分批软删除（建议）** | 删除立即返回；项目、工作区一删除，其中的工作项对每个读写都不存在（3.6），与删完之后没有可见的差别 | 请求是 M3 的几步（毫秒级），与工作项的数量无关。任务每个删除的项目一个（删除工作区另加没有项目的草稿一个），按 `id` 的键集每批 1000 行、一个短事务，不持项目、工作区的锁，不挡别的项目的写；SP4：每批约 3 ms，一个目标规模的项目在一次运行里做完（第三稿的语句没有键集，是平方的：一个项目的 40 万行动态 53.9 秒，改为键集 1.8 秒；一个五个项目的工作区约 9 次超时、2.4 小时的退避，聚焦复核 I2）；每次运行有 45 秒的预算，用完就重新排队，不算一次尝试 | `project.deleted`、`workspace.deleted` 两个事件，`issue` 的一个订阅者、一个任务种类和一组分批的语句（`delete_rows.go`，P1a 已写，P1–P8 由同步的过渡写法调用）；改总体设计 5.5 的一句（3.23）。**代价**：这是第一个跨模块的异步效果；请求提交到任务做完之间有一个窗口，那些行仍是"活的"，正确性靠每个读写都经项目的门（3.6 逐条列出、9.3 逐条测试）；一条跨项目的关联行属于先删除的那一端，删除工作项和任务都要据 `ProjectGate.DeletedAt` 跳过属于别人的行（3.5，交错 24、25）；任务用尽重试时行留在窗口的样子，60 天之后随项目的物理清理消失；与清理的先后要写明（3.6）；e2e 的删除断言改为限时轮询 |
| B 两个删除各有自己的长期限（几分钟），另写明规模的上限 | 删大的项目、工作区要等几十秒；HTTP 的客户端和反向代理（常见 60 秒）可能先超时，用户看到失败而删除在服务器上仍在进行 | 一个项目约 9 秒、五个项目的工作区约 37 秒，都在一个事务里；这段时间里工作区的每个项目的写都在等（闸门之下排满就 503）；规模超过上限的不保证删得掉 | 两个路由的期限的配置；README 写上限；不改 5.5 |
| C 超过一个规模（例如 2,000 个工作项）的删除拒绝（409），大的经 CLI 删除 | 大的项目、工作区在页面上删不掉，要找有服务器权限的人 | 请求里的删除有界；CLI 不受请求的期限约束，但同样持锁几十秒 | 一次计数、一个 409 的码、一个 CLI 命令和它的组合 |

- **在目标的规模上什么成立**：
  - **列表**（三个选项相同，S1）：1 万个工作项 2–100 ms；10 万时默认的一页 55–125 ms、按组 0.2–0.7 s、泳道 0.7–1.0 s；2 万在两者之间，页面可用。
  - **删除**（SP2、SP4）：只有 A 让请求的耗时与规模无关、不持锁几十秒，任务按键集每个项目几秒；B 能删完但要几分钟的期限和挡住整个工作区的写；C 在上限以下与 B 的小规模相同，以上的删不掉。
  - 不论怎样裁定，P1b 在目标的规模上量（一个 2 万个工作项的项目、一个五个这样的项目的工作区），请求的耗时和工作项一侧的软删除的耗时分开写进 P1b 的 spec。选 A 时 P9 的量测是一条通过线：同样的工作区，每个任务都在一次运行里做完（没有 snooze、没有重试），每个任务的耗时和期间别的项目的写的延迟记进 spec。
- **理由**：同步的做法在 v0 自己的规模目标上删不掉一个工作区，B、C 都是把这一点变成用户或运维要承担的限制。A 的窗口对用户不可见：项目的门本来就把已删除的项目当作不存在，A 只是让工作项一侧的行晚一些写下同一个时刻。A 的代价（异步、窗口、失败时的兜底）都写明、都有测试。
- **删除工作区的取锁**：选 A 时仍先锁住它的每个项目、再读时刻（决策点 1 的 B，3.5 重新推过）：任务写的是请求的时刻，一个在那之后提交的工作项的写若不被挡住，就会让它的行的时刻倒退。这一步现在只挡请求里 M3 的几步，代价很小。
- **以后改的代价**：A → B 删掉订阅者和任务，把 `delete_rows.go` 的语句放回请求的事务（P1–P8 的过渡写法就是这样）；B → A 是反过来。改规模目标只改 3.16 的数字和测量的数据，不改 A 的做法。

### 决策点 8：项目管理员能不能删除别人的评论（对抗性评审 M14）
**负责人裁定（2026-10-17）：B，项目管理员也能删除（不能改）别人的评论；时间线上不显示删除。** 落在 3.4、3.12、5.3、7.4、9.2、P4、P15。

**背景**：Plane 的服务端允许项目管理员改、删任何人的评论（`allow_permission([ROLE.ADMIN], creator=True, model=IssueComment)`），页面只给作者显示菜单（`comments/quick-actions.tsx:41-43`）。第二稿按页面收紧为"只有作者"，并把它列在"按页面定下、不交负责人"的几件里。Nerve 不区分人和代理：一个失控的账户（例如循环发评论的代理）留下的评论，在"只有作者"之下只能由它自己删，或删掉整个工作项。这是去掉了一种管理的能力，属于负责人。

| 选项 | 对用户 | 对代码 |
|---|---|---|
| A 照页面：只有作者能改、删 | 管理员清理不了别人的评论 | 不改页面；服务端一条相对的规则 |
| **B 项目管理员也能删除（不能改）别人的评论（建议）** | 管理员的别人的评论的菜单里有"删除"，没有"编辑"。**代价**：删除在时间线上不留痕迹：评论不写动态（3.13，Plane 写而从不显示），谁删的只在评论行的 `updated_by_id` 和 `issue.comment_deleted` 事件（`ActorID`）里；在"只看自己"的项目里只是访客的工作区管理员也有 `Grant.ProjectAdmin`（`access/domain/decide.go:76,81`），能删别人在他自己建的工作项上的评论 | 服务端：删是作者或 `Grant.ProjectAdmin`，改只有作者（403 `issue.not_author`）；页面给项目管理员显示"删除"（P15，登记前端改动清单）；矩阵多一行 |
| C 照 Plane 的服务端：管理员能改、删 | 管理员能改写别人的评论 | 页面另加"编辑"；被改过的评论只显示"已编辑"，看不出是谁改的，读者会以为是作者的话 |

- **理由**：B 是 Plane 服务端规则的一个子集，给出清理的能力而不让一个人替另一个人说话；删除本身在页面上可见（评论不见了），而"是谁删的"只在 `updated_by_id` 和事件里。C 的改写没有记下改写者，在不区分人和代理的产品里会让评论的归属不可信。
- **删除留不留一条动态**（聚焦复核 M8）：不在 B 里加。3.13 定下评论不写动态（4.17：Plane 写、从不显示），给"别人删的"另开一种动态，要动态的格式（`field = comment` 的一行）、页面的渲染器和引用一条已删除的评论的显示一起定下，不是 B 顺手能带上的；所以 B 的痕迹是 `updated_by_id` 加 `issue.comment_deleted` 事件（M7 的通知、M8 的 Webhook 都能看到执行者）。负责人按建议裁定 B（2026-10-17），时间线上不显示。
- **以后改的代价**：B → A 删掉一个分支和页面的菜单项；B → C 加"编辑"的分支，但要先给评论加"改写者"。

---

## 11. 架构问题

以下十一条改变总体设计、M2 设计或 M3 设计的架构约定，都已由负责人批准（2026-10-17；第三稿加了 11.11 的写闸门，对抗性评审 I3）。每条写问题、做法和另一种做法；正文各节按做法写定，上级文档按 3.23 的 Phase 同步。

### 11.1 新模块 `issue` 和它要的端口（3.3、6.2、6.5、6.6；负责人已批准，2026-10-17）
- **问题**：工作项的 13 张表要一个归属；它要项目的锁、取号、状态和标签的目录、成员的角色，项目、工作区的删除又要连带它的表，而模块之间不能互相导入。
- **做法**：一个模块 `issue` 拥有 13 张表；`project.Provide` 交出七个端口（项目的锁和读取、取号、状态目录、标签目录、成员、可读的项目、自动归档的策略），`workspace.Provide` 交出已有的目录；`issue.Provide` 交出 `IssueCascade`（状态在用，标签的连带；P1–P8 另有删除项目、工作区的过渡的同步连带）和 `IssueListSettings`（显示设置的取值检查）；P9 起删除项目、工作区经 `project.deleted`、`workspace.deleted` 两个事件到达 `issue` 的订阅者（决策点 7）。组合的顺序：全部 `Provide` → 平台（时钟、只投递的客户端、出站的客户端、写闸门）→ `access.New` → 分发器 → `issue.New` → `project.New` → `workspace.New` → `identity.New` → 任务。
- **另一种做法**：拆成"工作项""评论""草稿"几个模块。它们彼此有外键、在一个事务里写（动态、关注、提及），拆开之后每个写要跨模块的端口，删除工作项的连带要回调评论模块，评论又要锁工作项的项目：端口成环。

### 11.2 加锁约定的延伸（3.5；负责人已批准，2026-10-17，取锁的选择是决策点 1）
- **问题**：M3 的全局顺序和六条约定要延伸到 13 张新表；工作项的写取哪一把锁见决策点 1；关联可以跨项目，删除可以跨项目地软删除关联行。
- **做法**：
  - 全局顺序接上 `draft_issues` → `issues` → 子表（3.5）。
  - 工作项的写最先锁项目行（N），不锁工作区行；锁之前一个不加锁的预检，凡会被拒绝的（看不到的 404、不能做的 403，含关联的另一端看不到的）在取锁之前回答，不取锁、不进写闸门（11.11）；删除工作区先按 `id` 锁住它的项目、再读时刻（决策点 1；决策点 7 选 A 时仍需要，3.5）。B 的两个代价写在 3.5 和决策点 1：工作项的写与成员关系的变更之间时刻可以与生效的顺序不一致；草稿和 M3 的项目管理类的写仍先取工作区 S。
  - **约定七（新）**：跨项目的写按 `id` 升序同时锁住涉及的每个项目（今天只有关联）。设计简报写的是"源一侧的项目 N，别的工作项行 S"；本文改为两边的项目都 N，因为关联的写会写到对方一侧的行（对方的 `updated_at`、对方工作项上的反向动态）。只锁源一侧时，与删除对方项目的连带交错有三种后果（S5 的 E5 复现）：对方的 `updated_at` 倒退、反向动态"先删后建"，最坏的是留下一条活的关联行指向已删除的工作项（连带先锁关联行、再处理工作项，等锁期间插入的行不在它锁住的集合里）。约定七之下关联的写等在对方项目上，删除提交之后什么都不写（E5b）；交错 23。
  - **约定五的例外延伸到关联行**：删除工作项、删除项目的连带（决策点 7 选 A 时是任务的那一批）先按 `id` 锁住要删的关联行，再只删锁住的那些；锁关联行的一步排在全局顺序里关联表的位置上（对抗性评审 M13）。
  - 草稿的写持工作区 S，有项目时再持项目 S。
  - 后台的写不取项目的锁（抓取的带守卫的写、清理、版本的保留、删除工作项一侧的任务），理由在 3.5。
- **另一种做法**：关联的写只锁源一侧、对方的工作项行 `FOR SHARE`（设计简报）。它挡不住对方项目的连带在锁之前读了时刻、在锁之后写（3.5），需要再给连带加一步"锁关联的对方"，比锁两个项目更绕。

### 11.3 权限：创建者、访客只看自己的工作项（3.4；负责人已批准，2026-10-17）
- **问题**：总体设计 6.5 只有"`AllowCreator`（创建者本人可以修改或删除）"一句；`guest_view_all_features` 的约束 Plane 只在约 6 处执行。
- **做法**：`Rule.AllowCreator`：创建者要是项目的有效成员、角色不低于成员才算；`Target.CreatorID`、`Grant.OwnOnly`；一条访客规则在单个的读写（用例里，"看不到"）、列表和搜索（SQL 里）上执行，挂在看不到的工作项下的一切都是 404；一个一致性测试核对两处不走散。
- **另一种做法**：照 Plane 的 `creator=True` 只看工作区成员关系（已离开项目的创建者仍能删"他的"工作项）；访客规则只在 Plane 执行的地方执行（其余泄露）。

### 11.4 领域事件与事务内的任务投递（3.19、3.20、6.4；负责人已批准，2026-10-17，引入与否是决策点 3）
- **问题**：事件的形状、订阅者的写法（M2 设计 3.15 留给 M4）；River 的事务内投递的接缝（`platform/jobs` 不能导入 `platform/postgres`）；漏登记 worker 没有响亮的迹象（M2 交接第 6 节）。
- **做法**：`shared` 的 `Event`、`EventPublisher`、`EventSubscriber`；分发器在组合根，同步、在调用者的事务里、出错回滚；订阅者要做异步的事时经端口投递 River 任务（`InsertTx`）；`platform/postgres` 交出 `Tx(ctx)`，`platform/jobs` 交出只投递的客户端（`NewInserter(pool)`，在全部模块之前建出）和它的 `InsertTx`，`bootstrap` 接合；工作的客户端照 M2 的写法最后建出、一次登记全部 worker；两个队列；定时任务的种类和模块声明会投递的种类没有 worker 时启动失败（6.6，S4 在 River v0.47.0 上验证）。P9 起 `project`、`workspace` 也发布删除的事件，`issue` 的订阅者投递工作项一侧的删除任务：这是第一个跨模块的订阅（决策点 7）。
- **另一种做法**：事件写进一张 outbox 表、由后台任务分发（异步的分发器）。它多一张表、一个轮询、一种"已提交而未分发"的状态；River 的 `InsertTx` 已经是事务内的 outbox，再加一层没有好处。

### 11.5 列表的查询以目录为参数（3.16；负责人已批准，2026-10-17）
- **问题**：列表要按状态名、标签名、负责人名排序和分组，按状态组筛选，这些都在别的模块的表里；`TestSQLCSchemaScope` 不许 `issue` 的查询读它们。
- **做法**：用例经端口读出目录（id 和比较键），作为并列的数组参数传进静态的 sqlc 查询，查询里 `unnest` 成表再连接。不开 sqlc 的例外，名字的比较键在 Go 里算（与前端的比较同一个规则，3.16）。
- **另一种做法**：给 `issue` 开一个读 `states`、`labels`、`profiles` 的例外。守卫变弱，`issue` 依赖别的模块的表结构，M3 为 `access` 拒绝过同样的例外（M3 设计 3.3）。
- **S1 的结论**（0.3）：能生成，要把多个数组的 `unnest` 写成 `ROWS FROM`，总数放在每行的窗口列；结果对全部排序码、筛选、多值的组、子分组正确。代价是排序用不上索引，每个请求扫描、排序项目里满足筛选的行：v0 的目标是一个项目至多约 2 万个未删除的工作项（3.16），更大的项目是 M8 的性能工作。连接的参数（`jit = off`、`plan_cache_mode = force_custom_plan`）由 nerve 设。

### 11.6 列表的接口约定（3.16、5.2；负责人已批准，2026-10-17）
- **问题**：总体设计 3.3、3.4 的写法表达不了页面要的几样：空值（"没有标签"）、同一个日期的两个条件、单组"加载更多"的总数、子分组的结构；Plane 对未知的排序静默降级。
- **做法**：
  - 筛选：`none` 表示空值；日期的参数可重复（AND），`YYYY-MM-DD` 是单日；不合格的取值 422。
  - 不分组的列表带 `total`；分组 `{groups: [{key, total, data, next_cursor}], total}`，子分组 `{groups: [{key, total, subgroups: [...]}], total}`，只返回非空的组。
  - 一组（一格）的"加载更多"是同一个请求加 `group_key`（`sub_group_key`）和这一组的 `next_cursor`，回答不分组的形状。v0 3.4 写的"不分组的列表加上这一组的筛选"在多值的字段同时被筛选和分组时表达不了（`label_id=A&group_by=label_id` 的组 C，3.16）。
  - 未知的 `order_by`、`group_by` 422。
  - 游标的载荷是 `[排序码, 键, id]`：写明排序、以 `id` 结尾；空值在最后；前端用同一个全序在本地重排（共用的样例文件）。
  - 集合型的列表加上子工作项、关联、链接、工作项的表情；关联内嵌对方的列表行、评论内嵌它的表情（总体设计 3.6 的例外）。
- **另一种做法**：接口接收 Plane 的筛选表达式（JSON）。它能表达否定和嵌套，而页面生成不了它们（3.22）；表达式的解析和 SQL 的生成都更大，PAT 的使用者也更难写。

### 11.7 测试的时钟（3.20；负责人已批准，2026-10-17）
- **问题**：自动归档、物理清理的故事要"30 天、61 天之后"，不能等真实的时间（M2 交接第 13 节）；M2 只有用例测试里的固定时钟。
- **做法**：配置 `clock.offset`（只在 `env = test`）让运行中的 nerve 的用例时钟整体偏移；e2e 的 `clock.ts` 在新库上先用正常的 nerve 准备数据、再以偏移起一个 nerve，同一个库上任何时刻只有一个 nerve。
- **另一种做法**：一个只在测试环境存在的"拨钟"接口。它是只为测试存在的生产代码路径，还要处理拨钟期间在路上的请求。

### 11.8 `platform/outbound`：防 SSRF 的出站客户端（3.11、6.8、8.1；负责人已批准，2026-10-17）
- **问题**：链接的抓取按用户给的地址发出请求；M8 的 Webhook 也是。
- **做法**：一个平台包，照 Plane 的规则（解析后拒绝内网和保留的地址、连接到检查过的地址、每一跳重新检查、不用代理），加上响应和图标的上限；测试环境可以经 `outbound.allow_cidrs` 放行本机的测试服务器。
- **另一种做法**：放在 `issue/adapter/crawler` 里。M8 要复制一份，两份防护会走散。

### 11.9 60 天的物理清理按模块（3.21；负责人已批准，2026-10-17）
- **问题**：总体设计 5.5、6.7 只写了"60 天后物理清理"；它要删除 M2、M3、M4 的表，而模块的查询只能碰自己的表。
- **做法**：每个模块交出一个清理者，删除自己的表里软删除超过期限的行（分批、子先于父）；组合根把它们组成一个任务、按 `issue` → `project` → `workspace` → `identity` 的顺序运行；指向被清理的表的外键都是 `CASCADE` 或 `SET NULL`（动态改为 `CASCADE`），一个目录测试守住。账户不清理。
- **另一种做法**：一个清理任务直接按目录删除全部表（不分模块）。它要读写每个模块的表，越过 sqlc 的守卫。

### 11.10 工作项的显示设置与列表的词汇（3.17；负责人已批准，2026-10-17）
- **问题**：M3 把显示设置的筛选和显示列的格式留给 M4；存下的设置就是下一次的查询，两个模块（`project` 存、`issue` 用）必须一致。
- **做法**：`project_user_properties` 加 `filters`、`display_filters`、`display_properties` 三列，`filters` 用列表参数的格式；数据库只查形状。列表的词汇（参数名、取值的语法、分组码、排序码、各页面允许的集合）在 `issue/domain`；`project` 检查显示设置时经它声明的端口 `IssueListSettings`，由 `issue.Provide` 交出这份词汇（纯函数，不要连接池，与 `IssueCascade` 同一个方向，3.17）。已归档、个人主页、草稿页的设置照 Plane 只存在浏览器。
- **另一种做法一**：词汇放进 `shared`，作为两个模块必须一致的纯取值规则（M3 设计 11.4 的延伸；第一稿的做法）。不取：这份词汇随 M5–M7 一直长（附件数、迭代、模块、视图），`shared` 会积起一份工作项领域的、不断变化的词汇，每个模块都看得见它（独立评审 M13）。
- **另一种做法二**：照 Plane 存筛选的表达式（`rich_filters`），由 `issue` 在每次请求时翻译。存下的东西与接口的参数不同形，PAT 的使用者读到的设置不能直接拿去请求列表。

### 11.11 工作项的写闸门（3.5、6.1、6.7；负责人已批准，2026-10-17；对抗性评审 I3）
- **问题**：每个工作项的写在等项目行的 N 时持着池里的一个连接（池是 10 个），一个热的项目上排着的写能占满整个池，别的项目上无关的读写排在池的等待者后面：SP1 量到一个凭证的突发额度（200 个写）打在一个项目上时，另一个项目的读等约 875 ms。决策点 1 的 A、B 都一样（它们决定等哪一把锁，不决定等的时候持不持连接）。M3 的写都是罕见的管理操作，没有遇到这一点。
- **做法**：
  - 一个平台包 `platform/gate`（`Keyed[K comparable]`，带 `context` 的按键的信号量，约 100 行）；`issue` 在 `issue/app` 声明端口 `WriteGate.Enter(ctx, projectIDs) (release, error)`（P1b 一次一个项目（`Enter(ctx, projectID) (leave, error)`），P3 加多个（P1b spec 第 3 节第 10 条）），`bootstrap/gate.go` 用它实现。
  - 每个工作项一侧的写（含发布草稿）在预检之后、**取连接之前**进入它的项目的闸门：每个项目在一个进程里至多 K 个写在数据库里（`issue.write_gate.per_project`，默认 2），其余在进程里等，不持连接。跨项目的写（关联）按项目 `id` 升序一次取齐它到达的每个项目的闸门，任一个失败时放回已取的。
  - 等的上限：`issue.write_gate.max_wait`（默认 2 秒）、每个项目的等待者 `issue.write_gate.max_queue`（默认 64）、请求的期限，先到的那个让请求得到 503 `server_busy`（`Retry-After: 1`，M2 已有的码）。兜底：工作项一侧的写的事务 `SET LOCAL lock_timeout`（`issue.write_lock_timeout`，默认 5 秒），由 `platform/postgres` 的 `TxManager` 的选项 `LockTimeout` 在 `BEGIN` 之后发，超过时（`55P03`）在 `bootstrap/gate.go` 换成同一个 503（6.1）。
  - 闸门不是数据库的锁：持着它的请求不在持数据库的锁或连接时再等别的闸门，不走闸门的写（M3 的项目管理类的写、删除项目和工作区、后台的任务）不等它，所以它不改变 3.5 的取锁顺序、不与数据库的锁成环。
  - K 是每个进程的；多个进程时每个进程各 K 个。闸门限的是每个项目的份额，同时热的项目达到池的大小除以 K（默认 5 个）时，等锁的写仍能占满池；连接池的大小与 K、同时热的项目数由 M8 的部署交接写明（13.2）。
  - 测试：`platform/gate` 的单元测试（9.1）；交错 20 经真实的连接池跑，断言另一个项目上的读的耗时（9.3）。
- **代价**（决策点 1、第 16 节也写明，A、B 相同）：热的项目上超过 K 的写排队，排得太久的得到 503、由客户端重试（P1b 之前它们会一直等到请求期限）；每个写用例多一步；多一个平台包和一个端口。
- **另一种做法一**：只有 `lock_timeout`。等锁的写仍持着连接，只是等得短一些；池在那几秒里照样被占满，别的项目的读照样排队。
- **另一种做法二**：把连接池按用途分开（工作项的写一个池、其余一个池）。别的项目的读不再受影响，但同一个池里不同项目的写仍互相挤占，池的大小要按项目数调，而 pgx 的池没有按键的限额。

---

## 12. Phase 划分与实施规划

所有 Phase 依次推进，每个 Phase 按 M2、M3 的节奏（worktree → spec 和 plan → 按任务实现、逐个评审 → 整分支评审 → 修复 → review、交接 → `--no-ff` 合并、推送 → 持续集成通过 → 清理 worktree 和分支）。每个 Phase 合并时：持续集成的全部门禁、此前已加入的全部故事都通过；3.23 和 8.8 中标着这个 Phase 的行已在同一次合并中同步。

**约束**：
1. **每个故事的前置数据在它加入时都能经接口准备**（总体设计 8.2），不写 SQL。权限矩阵是 Go 测试，用仓储准备数据，每个 Phase 都能加它的行（9.2）；经接口准备不了的格子放进 Go 的集成测试（分诊状态的 `state_id`：`createState` 拒绝分诊，对抗性评审 M17）。访客建的工作项经接口准备：成员建，再经 `updateProjectMember` 把他降为访客（2 节）。
2. **每段约 16 个任务以内，plan 不超过约 1,500 行**。M3 有四段在实施前超出上限、各拆成两段（M3 设计第 12 节约束 2）。M4 按任务数分好；任务比 M3 的大（7.10：每个前端任务约 287 行生产代码、223 行测试）。第三稿把 P10 预先拆成 P10a、P10b（对抗性评审 I7）。P1（16 个任务）在实施前按 M3 已合并的 plan 实测估出约 22–25 个任务、约 23,000–26,000 行，两个上限都超出，由控制者按下面的预先批准裁定（2026-10-17）拆成 P1a、P1b，并预先批准 P1a 仍超出时的退路（见 P1a 一节）。**预先批准**（决策点 2）：某段的架构子任务按已合并的 Phase 的 plan 实测估出超过上限时，控制者直接拆成 P<k>a、P<k>b 继续，不停下来请示，拆法和理由记进第 15、16 节。
3. **评审敏感的内容不在同一段**：加锁约定和连带（P1a 起步：项目 N、预检、删除工作区的新顺序；P1b：写闸门、连接池；P3 延伸：约定七、关联行的例外）；权限的扩展（P2；起止日期和 `bodyshape` 的生成器也在这一段，与锁分开）；HTML 的清洗（P4）；列表引擎的 SQL（P5）；事件和 River（P6）；抓取和出站（P7；`link_count` 改列表的 SQL，放在 P8）；后台的批量删除（P9：物理清理、描述版本的保留、删除项目和工作区之后工作项一侧的任务放在一段，一起证明它们与请求的写、与彼此不相遇；对抗性评审 M21）。
4. **新的错误码随它的文案进来**：声明新错误码的任务，在同一个任务里把它加进 `PROBLEM_MESSAGES`（`web/apps/web/core/lib/error-messages.ts`）和 `en`、`zh-CN` 两份 `errors.json`，并运行 `make test-web`（M3 设计第 12 节约束 4）。
5. **字段随它的数据进入接口**（M2 设计 3.2 的规则 3 在 M4 之内同样适用）：`start_date`、`target_date` 在 P2，与 `bodyshape` 的 `date` 检查器一起（P1a 的迁移已建这两列）；`description_html`、`skip_activity` 在 P4，与清洗、提及、描述的版本一起（3.7）；`sub_issues_count` 在 P3（父子）；`link_count` 和它的两个排序码在 P8（链接的表 P7 已有；它改的是列表的 SQL，不与抓取同段）。之前的 Phase 不定义它们。
6. **每个 Phase 开始的提交**把 `tools/keywords.json` 的 `phase` 改为 `M4/P<k>`，同一个提交改这时到期的例外的 `until`（取删除它的 Phase 的下一个，7.9）。
7. **故事随 Phase 长大**（第 2 节"断言的增长"一列）：一个故事在它的核心断言都已存在的第一个 Phase 加入；之后的 Phase 在它的端到端任务里扩充同一个故事的断言和前置数据。每段的完成线只写那时已有的断言。
8. **每个给工作项加表的 Phase**（P1a、P3、P4、P7 加工作项的子表，P8 加草稿）同时延伸：`deleteIssue` 自己的连带（P2 起；草稿除外）；删除项目、工作区的语句（`issue/adapter/postgres/delete_rows.go`：P1–P8 由 `IssueCascade` 的过渡方法在请求的事务里调用，P9 起由任务分批调用）；e2e 的 `furnishIssueRows`，并让 M3 的删除故事（P4、W2、W3）在删除之前调它（对抗性评审 I6）。`expectIssueDeleted`（沿外键图）、`expectProjectDeleted`、`expectWorkspaceDeleted`（`workspaceTables` 从目录读出）自动断言到新表。

**顺序**：
- **后端十段，前端八段（P10 预先拆成 P10a、P10b），然后收尾**（先后端、后前端，M2、M3 的做法：前端对接真实的接口）。设计简报估计后端七到八段；P6（事件和 River）与 P7（抓取和出站）各是评审敏感的内容，分成两段，所以定稿时是九段；P1 在实施前拆成 P1a、P1b（约束 2），现在是十段。
- **P1a 工作项的表与创建**让"工作项的写只锁项目行"、锁之前的预检（凡拒绝的都在锁之前回答）和删除工作区的新顺序第一次落地，带上创建和它的动态、关注，I1 的接口版本先有这些断言。连带（删除项目、工作区、状态、标签）随表加入；删除项目、工作区在 P1–P8 是同步的过渡写法：M3 的删除故事从 P1a 起就要工作项一侧的行被删除，而决策点 7 的任务要 P6 的事件和只投递的客户端，P9 改为任务。
- **P1b 取、改与写闸门**：取、改两个操作（不含描述、日期）和它们的动态，I2 的接口版本；写闸门和 `lock_timeout` 的兜底（插在 P1a 的预检和事务之间）；连接池的交错 20；在规模目标上量 P1a 的删除（决策点 7）。
- **P2 权限与日期**：访客规则、创建者、按编号取、删除、归档，它要 P1a、P1b 的读写才有东西可以挡；起止日期和 `bodyshape` 的 `date`（约束 5）。
- **P3 父子与关联**：父子不成环、子树的删除、跨项目的关联（约定七）。
- **P4 协作**：描述和它的清洗、提及、版本（I1、I2 的断言在这里补全），评论、表情、动态的列表；第二个分页的列表（评论）在这里，页大小移到 `shared`（M3 交接第 9 节）。
- **P5 列表**：引擎要的过滤（访客规则、已归档）、排序键（子工作项数在 P3 之后有了）都已就位；显示设置的三列随它；连接的参数（3.16）。
- **P6 事件与任务**：给 P1–P4 的写加上发布点；只投递的客户端；自动归档是 River 的第一个业务定时任务；测试的时钟。
- **P7 链接**：第一个订阅者和第一个事务内投递的任务，出站的客户端。
- **P8 草稿与搜索**：发布复用 P1a 的创建核心；搜索要 P5 的可读集合和 P3 的父子（选择器的排除）；`link_count` 进入列表。
- **P9 后台的批量删除**：60 天的清理、描述版本的保留、删除项目和工作区的任务。全部表都已在，事件和任务（P6）已在。
- **P10a、P10b、P11 数据层**（换型和删除 M6、M7 的工作项列表、列表一侧、详情一侧）先于全部页面（3.1）：页面的故事加入时挂载路径上已经没有旧接口。
- **P12–P15 页面**，P16 清理（死成员、`no-shadow`、守卫的终态）。

**规模**：257 个任务（后端 137、前端 106、收尾 14）。M3 的设计是 148 个、实施时因拆分成为 171 个加 16 个的收尾；M4 的前端 106 个任务，约是 M3 前端实际（55 个）的 1.9 倍（7.10），后端的操作少一些（37 对 45），但列表引擎、事件、任务、清理是 M3 没有的。第二稿比第一稿多两个后端任务（描述、清洗移到 P4）。第三稿比第二稿多 12 个：P2 加上日期（+2）；版本的保留从 P6 移到 P9（P6 −1）；P8 加上 `link_count`（+1）；P9 加上版本的保留、两个删除的事件、删除的任务、窗口的测试和 e2e 的轮询（+4）；P10 预先拆成两段，加上删除 M6、M7 的列表和"先收窄、后换型"（+7）；I20 的页面从 P13 移到 P14（P13 −1）。P1 的写闸门和规模的测量占了移出的日期的位置。定稿时是 241 个（后端 121），P1 一段 16 个；P1 拆成 P1a（15 个）、P1b（12 个）之后后端多 11 个（约束 2）；P3 在实施之前在约 16 个任务、每个约 1,500 行以内重新切成 16 个（+5，[P3 spec](specs/P3-issue-hierarchy.md) 第 3 节第 1 条）。

### P1a `issues`：工作项的表、项目行的锁、创建与连带（后端，15 个任务）
- **拆分**（P1a、P1b 共用）：设计定稿时这里是一段 P1 `issues`（16 个任务）。实施之前，P1 的架构子任务按 M3 已合并的 plan 实测的每个任务的行数（P4a、P4b、P7a、P7b 的同类任务）和每行手写代码在 plan 中占的行数（P4a 1.41、P4b 1.47、P7a 1.62、P7b 1.72）逐个估出约 22–25 个任务、约 23,000–26,000 行，草稿的第 1 个（与第 5 个分不开）、第 9、11、13、15 个各超过约 1,500 行，两个上限都超出；由控制者按决策点 2 的预先批准裁定（2026-10-17）拆成 P1a、P1b，依次合并：P1a → P1b，每段合并之后下一段才开始（约束 2、约束 3 和"顺序"）。边界由代码的耦合定下：新表一建出，按目录的删除测试要每张表有一行准备的数据和它的连带（"seed a row under the deleted project"），M3 的删除故事（P4、W2、W3）要经接口造出的行（`furnishIssueRows`，要 `createIssue`）；第一个工作项的操作要它的矩阵行和取锁形状的测试（`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 按契约列出每个带 `{project_id}` 的写，要收窄）；删除工作区的新顺序要与同步的连带一起到来（不然持项目 N 的创建在删除读了时刻之后提交，连带写下的 `deleted_at` 早于它的 `created_at`）。所以表、项目行的锁和预检、创建、连带、删除工作区的新顺序在 P1a；取、改、写闸门和连接池在 P1b（写闸门插在预检和事务之间，P1a 的测试都不要它）。两段的任务列表同样是 plan 的草稿：两段的架构子任务可以在约 16 个任务、每个任务约 1,500 行以内重新切分，不再停下来请示。**预先批准的退路**（M3 的 P5a 的 A′ 的写法）：P1a 的原型仍超出约 16 个任务，或某个任务超过约 1,500 行而重新切分解决不了时，第 11、13 个任务（`StateInUse`、`DetachLabel` 和交错 6、7 的创建一支，连同 M3 交接第 4 节中删除状态、标签的去向的关闭）移到 P1b，仍不够时 I1 的故事文件也移过去，不再停下来请示，记进 P1a 的 spec 第 3 节和第 15 节。规模目标上的删除的测量在 P1b（它量 P1a 的代码，P1b 不改它；只为两段的大小，退路移出足够多时可以留在 P1a，记进 spec）。关键词守卫的 `phase` 只认 `M<n>/P<k>`（`tools/keywords.mjs:23`），两段都写 `M4/P1`（约束 6）。每个加操作的任务同时加它的操作名、规则行、矩阵行和取锁形状的测试中的一行，声明新码的任务同时加前端文案（约束 4）：完整性核对要求契约的每个操作从它出现起就有这些。
- **目标**：任何调用方都能建工作项（描述在 P4、起止日期在 P2），带默认值、编号、排序值、完成时间、关注和动态；工作项的写只锁项目行，凡会被拒绝的在锁之前回答；删除项目、工作区、状态、标签连带工作项的表；删除工作区先锁它的项目、再读时刻。
- **评审重点**（约束 3）：取锁的一步（读 → 预检 → 项目 N → 重读 → 判定），预检拒绝的写不取任何锁，锁下的判定仍是最终的（交错 4、5 的创建一支）；删除工作区的新顺序与 M3 设计 3.6 的取锁表中的每个写没有新的等待环；新表进入两个连带，同一时刻、同一个删除者，按目录的删除测试不开豁免；每个新的批量写按约定五核对，没有新的增长。
- **任务**：
  1. 本 Phase 的第一个提交：`phase` 改为 `M4/P1`，`/user-properties/` 的一条例外（命中两处）的 `until` 改为 `M4/P11`（7.9）；迁移 `00015`–`00019` 和它们的约束、索引名；`sqlc.yaml` 的 `issue` 条目，grants；`issue` 模块的骨架（`Provide` 交出 `IssueCascade`；`actions.go` 的 `issue.create` 和它的规则行）；删除项目、工作区的语句（`adapter/postgres/delete_rows.go`，按 `id` 的键集）和编排（`app/delete_project_rows.go`），`IssueCascade` 的过渡方法 `DeleteProjectIssues`、`DeleteWorkspaceIssues`（在请求的事务里调这些语句），接进 `project` 的删除步骤和命令行的组合；按目录的删除测试给五张表准备行（不开豁免）；端到端的过渡（`expectProjectDeleted` 暂不读这五张表，到第 14 个任务结束）。
  2. 键集的存储测试（分批、`after`、时刻和删除者、别的项目不变）；CHECK 的反例；部分唯一和全表唯一的键。
  3. `issue/domain` 创建一侧的规则（标题、优先级、负责人和标签的集合、排序值、完成时间、默认负责人）和动态的第 1、2 行（3.13）；本模块的错误码。
  4. `project.Provide` 的 `ProjectGate`（`Find`、`Lock`）、`Sequences`、`StateCatalog`、`LabelCatalog`、`ProjectMembers` 和它们的集成测试。
  5. 取锁的一步和不加锁的预检（`issue/app/lock.go`，3.5）：读 → 预检（凡拒绝的 404、403 在这里答）→ 项目 N → 重读 → 判定；用例测试。
  6. 创建的存储：插入工作项、负责人、标签、关注、动态，同一状态的最大排序值，详情的重读（完整的数组）；存储测试和失败测试。
  7. `createIssue` 的用例（默认状态、默认负责人、编号、排序值、完成时间、关注、动态，锁下读时钟）。
  8. `createIssue` 的接口：`issue.yaml`（`Issue`、`IssueDetail`、`IssueCreate`）、HTTP、`issue.New`、`bootstrap/ports.go` 的转换、矩阵的行；M3 的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 收窄到项目管理类的写；契约里 `Project.identifier`、`default_assignee_id`、`State.default` 的说明改成它们的作用（5.1，M3 交接第 4 节）。
  9. 取锁的形状（交错 21）：每个工作项的写第一把锁是项目的 N，工作区行上只有外键的 `FOR KEY SHARE`；看不到项目、看得到而不能写的账户（公开项目里不是成员的工作区成员、不是项目成员的工作区管理员）的写不取行锁、不写；盖戳和事务的连接的行。
  10. 删除工作区的新顺序：`ProjectCascade.LockWorkspaceProjects`，读时刻移到两步锁之后（3.5）；交错 3（以创建为持项目 N 的写，两种顺序）；M3 的交错照旧通过。
  11. `project` 的 `deleteState` 加 `StateInUse`（409 `project.state_in_use`，文案）、`deleteLabel` 加 `DetachLabel`。
  12. 交错 1、2、8、22；交错 4、5 的创建一支（移出、降级先提交时锁下的判定为准）；分诊状态 422 的集成测试（经仓储准备）。
  13. 交错 6、7 的创建一支。
  14. 端到端：`e2e/fixtures/issues.ts` 的建工作项、`assert/issue.ts`、`furnish.ts` 的 `furnishIssueRows`（五张表）和 M3 的 P4、W2、W3 调它，`workspaceTables` 从目录读出，过渡结束；I1 的接口版本（本 Phase 的断言）。
  15. 3.23 中 P1a 的行；review。
- **关闭**（13.1）：M1-P2 交接的接口字段一项（没有 `estimate_point`、`description_binary`、`/issue-dates/`）；M1-P3 交接的接口字段一项（没有 `is_epic`、`type_id`，没有批量的接口）；M3 交接第 2 节的预检一项（凡拒绝的写不取锁）和约定五的逐个核对（P1a 的批量写：创建插入的负责人、标签、关注，删除项目、工作区的连带的分批语句，删除工作区按 `id` 锁它的项目，删除标签的 `DetachLabel`）；M3 交接第 4 节（删除状态、标签的去向，默认负责人的契约，`identifier`、`default_assignee_id`、`State.default` 的说明，`workspaceTables`、`furnishIssueRows`）。
- **完成线**：I1 的接口版本（工作项的行、编号、默认状态和默认负责人、负责人、标签、关注、创建和负责人的动态、各种拒绝）通过；M3 的 P4、W2、W3 带着工作项一侧的行通过，没有留下端到端的过渡；交错 1、2、3、8、21、22 和交错 4–7 的创建一支 `-count=5 -race` 通过，没有 40P01；看不到项目、不能写的写不取锁；本 Phase 的矩阵格子通过；删除在用的状态 409，删除标签从工作项上摘下。

### P1b `issue-update`：工作项的取、改与写闸门（后端，12 个任务）
- **拆分**：见 P1a 的同一条；P1b 在 P1a 合并之后开始。
- **目标**：任何调用方都能取、改工作项（描述在 P4、起止日期在 P2）；一个热的项目不占满连接池：取连接之前进入按项目的写闸门，`lock_timeout` 兜底，满时 503。
- **评审重点**（约束 3）：写闸门（取连接之前进入，不持连接地等，从不在持数据库的锁时再等别的闸门；每个项目至多 K 个、`max_wait`、`max_queue`、请求的期限，503 `server_busy` 带 `Retry-After: 1`）；`55P03` 换成 503 只在 `bootstrap/gate.go` 一处；修改的批量写（负责人、标签的差集）按约定五核对；预检与锁下的判定（交错 4、5 的修改一支）。
- **任务**：
  1. `issue/domain` 修改一侧的规则（补丁、负责人和标签的差集、完成时间的转换）和动态的第 3、6、7、10–13 行（3.13）。
  2. 修改的存储（补丁；负责人、标签按差集增删，按 `id` 的顺序；关注）和测试。
  3. `getIssue`（用例、契约、HTTP、矩阵的行；响应的重读：完整的数组）。
  4. `updateIssue` 的用例。
  5. `updateIssue` 的接口（`IssueUpdate`、HTTP、矩阵的行、取锁形状和盖戳的行）。
  6. `platform/gate`（`Keyed[K]`）和它的单元测试（表、`-race`）。
  7. 写闸门接入：`lock.go` 的 `WriteGate` 端口、`bootstrap/gate.go`、四个配置；`platform/postgres` 的 `TxManager` 选项 `LockTimeout` 和 `LockNotAvailable`，`bootstrap/gate.go` 把 `55P03` 换成 503 `server_busy`；工作项一侧的写在契约里声明 `server_busy`。
  8. 交错 3、4、5、6、7 的修改一支。
  9. 饥饿与连接池（交错 20，经真实的连接池和闸门）：一串首尾相连的工作项写不让工作区一级的写等到期限（S5 的回归），200 个写压在同一个项目上时另一个项目上无关的读的耗时不超过上限（SP1 的回归），闸门满时 503 带 `Retry-After`，`lock_timeout` 的兜底。
  10. 在规模目标上量删除（2 万个工作项的项目、五个这样的项目的工作区，P1a 的同步连带，决策点 7），请求的耗时和工作项一侧的软删除的耗时分开记进 spec。
  11. 端到端：I2 的接口版本（本 Phase 的断言）。
  12. 3.23、8.8 中 P1b 的行；review。
- **关闭**（13.1）：M2 交接第 3 节的一部分（P1 的用例用时钟写 `updated_at`：创建在 P1a，修改在这里）；M3 交接第 2 节（方案的选择和代价，批量写逐个按约定五核对，没有新的增长，连接池的代价和写闸门；预检一项在 P1a）。
- **完成线**：I2 的接口版本（标题、状态、优先级、负责人、标签的修改和动态、`completed_at`、`updated_at`）通过；`getIssue`、`updateIssue` 的矩阵格子通过；交错 3–7 的修改一支和交错 20 通过；闸门满时 503 带 `Retry-After: 1`，`lock_timeout` 到时 503；规模目标上的删除耗时写进 spec。

### P2 `issue-access`：访客规则、创建者、删除、按编号取、归档、起止日期（后端，14 个任务）
- **目标**：访客只看自己的工作项、创建者的权限、删除、按编号取、归档和恢复；已归档的工作项只读；起止日期进入接口。
- **任务**：
  1. `shared` 的 `Target.CreatorID`、`Grant.OwnOnly`（按项目的角色：工作区管理员在项目里只是访客时同样为真）；`access` 读 `guest_view_all_features`、`AllowCreator` 的判定；判定表测试（3.4）。
  2. 规则表：`issue.read` 加访客；`issue.delete`（管理员 + `AllowCreator`）；`issue.archive`；契约里 `Project.guest_view_all_features` 的说明。
  3. `lock.go`：预检和锁下的判定都加上访客规则（404 优先），`getIssue`、`updateIssue` 改走它。
  4. `deleteIssue` 和它自己的连带（P1a 的四张子表；子树在 P3）；`expectIssueDeleted` 沿外键图递归（对抗性评审 M16）。
  5. `getIssueByIdentifier`（解析、`foldIdentifier` 和 `identifier-fold.json`、各种 404）。
  6. `archiveIssue`、`unarchiveIssue`（409 `issue.not_archivable` 和文案；动态第 14、16 行）。
  7. 已归档只读：`updateIssue` 的 409 `issue.archived`（文案）；工作项的写在已归档的项目里 409 `project.archived`。
  8. `shared/date.go`；`bodyshape` 和生成器：`format: date`（`shared.Date`）、路径按下标排序、开放节点的数的范围、代价测试加字符串的数组；`oapi-codegen` 的类型映射（M2 交接第 4 节）。
  9. 起止日期进入接口：`IssueCreate`、`IssueUpdate`、`Issue` 加 `start_date`、`target_date`；合并之后的日期 422；动态第 8、9 行。
  10. 交错 12；交错 5 加上访客规则下 404 的一支（P1a、P1b 时访客一律 403）。
  11. 矩阵：访客的三列、"管理员·访客"一列、创建者的两行。
  12. 整程序测试（含日期的格式）；`apitest` 两个方向。
  13. 端到端：I15 的接口版本（本 Phase 的断言）；I1、I2 加上起止日期和合并之后的日期 422；I2 加上已归档 409。访客取别人的工作项的 404 在本 Phase 只由矩阵（Go）核对，端到端的那一格随 I20 的接口版本在 P5（聚焦复核 M5）。
  14. 3.23 中 P2 的行（含 M2 设计 3.11）；review。
- **关闭**：M1-P3 交接的地址一项的一半（按编号取统一到 `/workspaces/{slug}/issues/{identifier}`）；M2 交接第 3 节的一部分（P2 的用例）、第 4 节（路径的排序、数的范围、`date`、代价测试；map 型、递归的结构写明"没有用到"）；M3 交接第 2 节中约定五的逐个核对（P2 的批量写：删除工作项的五条语句，归档、恢复的 `SetArchivedAt`）、第 3 节（`AllowCreator`、`guest_view_all_features`、矩阵、码；个人主页的一行在 P5）、第 4 节的 `expectIssueDeleted` 和 `guest_view_all_features` 的说明。
- **完成线**：I15 的接口版本（归档、恢复、两条动态、`issue.not_archivable`、改字段 409、访客 403）、I1 和 I2 的起止日期（含合并之后的日期 422）、I2 的已归档 409 通过；判定表、矩阵、完整性测试、`bodyshape` 的测试通过。

### P3 `issue-hierarchy`：父子与关联（后端，16 个任务）
- **目标**：父工作项、子工作项、子树的删除、跨项目的关联。
- **任务**：
  1. 迁移 `00020`，grants；删除的语句加关联（按 `id` 先锁，含另一端在这个项目里的，锁的一步排在全局顺序里关联表的位置）；`deleteIssue` 自己的连带加关联（两个方向，按 `id` 先锁）；`furnishIssueRows` 加关联。
  2. 父工作项：创建、修改带 `parent_id`，不成环（`WITH RECURSIVE`）；`Issue.sub_issues_count`（按调用者的规则数，3.9）。
  3. `listSubIssues`、`addSubIssues`（动态第 5 行，`old_value` 是原来的父）。
  4. `deleteIssue` 的子树（同一时刻，含已归档的）。
  5. 关联的领域；`ProjectGate.LockMany`；`createIssueRelations`（预检另一端的可见性，看不到的不锁、不进闸门；约定七，正反两条动态，两边的 `updated_at`）。
  6. `listIssueRelations`（对方可见、内嵌 `Issue`）、`deleteIssueRelation`（两边任一个的判定）。
  7. 归档与父子、关联的交互：父工作项已归档时加子工作项 409 `issue.archived`；已归档的子工作项不进列表和计数；已归档的对方不在关联列表、恢复之后重新出现（3.9、3.10）。
  8. 交错 9、10、11、13、23、26；交错 21 加上"加关联指向看不到的项目时不锁它"。
  9. 矩阵；整程序测试。
  10. 端到端：I8、I9、I17 的接口版本（本 Phase 的断言）；I15 加上加关联、加子工作项的 409。
  11. 3.23 中 P3 的行（集合型、内嵌的例外、约定七）；review。
- **关闭**：M2 交接第 3 节的一部分（P3 的用例）；M3 交接第 2 节中约定五的逐个核对（关联行的例外）。
- **完成线**：I8、I9 的接口版本和 I17 的接口版本（子树和它的负责人、标签、关注、动态、两个方向的关联同一时刻删除，创建者的规则）、I15 的关联和子工作项的 409 通过；交错 9–11、13、23、26 通过；加关联指向看不到的项目时不取锁。

### P4 `issue-collaboration`：描述、清洗、提及、版本、评论、表情、动态的列表（后端，16 个任务）
- **目标**：描述随它的清洗、提及、版本一起进入接口；评论和表情；动态和评论按游标列出。
- **任务**：
  1. S3 和清洗（`issue/adapter/content`；app 的 `HTMLSanitizer` 端口、domain 的纯值）：加入 bluemonday；放行表、纯文本、内容投影，`issue/module.go` 组装，保留现有架构守卫；往返的样例：编辑器每种节点的输出，经接口写入的 HTML（列表、标题、嵌套的标记）再经编辑器重新序列化的结果，只改链接地址、只改勾选框的两份（3.7）。
  2. 迁移 `00021`–`00025`，grants；删除的语句、`deleteIssue` 自己的连带、`furnishIssueRows` 加五张表。
  3. 页大小移到 `shared`（`identity` 改用它，M3 交接第 9 节）。
  4. 描述进入接口：`IssueCreate`、`IssueUpdate`、`IssueDetail` 加 `description_html`，`apitest` 核对 `IssueDetail` = `Issue` 加它；动态第 4 行和它的合并；`skip_activity`（内容标识：纯文本、提及、链接的地址、勾选框的状态，3.7）。
  5. 提及的集合同步（创建、改描述），被提及而能读这个工作项的人关注（3.12）。
  6. 描述版本的写入和合并（创建、改描述，600 秒）。
  7. `listDescriptionVersions`、`getDescriptionVersion`（规则 `issue_description_version.read`，404 码和文案）；版本的游标载荷和篡改的测试（8.6）。
  8. `createIssueComment`、`listIssueComments`（清洗、空评论 422、关注评论者和被提及的人、内嵌表情）。
  9. `updateIssueComment`、`deleteIssueComment`（改只有作者，删是作者或项目管理员，决策点 8；403 `issue.not_author` 和文案、`edited_at`、连带表情）。
  10. 工作项的表情（409 `issue.reaction_exists`、404 `issue.reaction_not_found` 和文案，`updated_at`）。
  11. 评论的表情。
  12. `listIssueActivities`：引用的工作项按调用者在它的项目里的成员关系判定，不看 `deleted_at`，已清理的略去（3.13）；同一个工作项的 `created_at` 单调的集成测试；评论、动态的游标载荷和篡改的测试。
  13. 交错 14、15。
  14. 矩阵（含删除别人的评论的一行）；整程序测试。
  15. 端到端：I11、I12、I13 的接口版本；I1 加上描述、提及、版本；I2 加上描述的修改与合并、只补块 id 的保存、只改链接地址或勾选框的保存照常记下；I15 加上评论的 409；I17 的前置数据加上评论、表情、提及、版本。
  16. 3.23 中 P4 的行（决策点 5、6 裁定之后的总体设计 1.1、5.2、6.6）；review。
- **关闭**：M1-P2 交接的 `skip_activity` 一项（服务端一侧）；M1-P3 交接的接口字段一项的另一半（评论没有 `access`、动态没有 `WORKLOG` 一类）和地址的另一半（描述历史的地址）；M1 收尾交接的 `comment_json`（服务端一侧不定义）；M2 交接第 2 节的一部分（评论、动态、版本的游标）、第 3 节的一部分（P4 的用例）；M3 交接第 9 节（页大小在 `shared`）。
- **完成线**：I11–I13 的接口版本，I1 的描述、提及、版本，I2 的描述的修改与合并、只补块 id 的保存和只改链接地址或勾选框的保存，I15 的评论的 409，I17 加上评论、表情、提及、版本之后通过；交错 14、15 通过；S3 的样例全部往返。

### P5 `issue-list`：列表引擎与显示设置（后端，15 个任务）
- **目标**：项目的列表（含已归档）、个人主页的列表：筛选、分组、子分组、排序、游标、一组的"加载更多"；项目的工作项显示设置。
- **任务**：
  1. `platform/postgres` 的连接池在每个连接上设 `jit = off`、`plan_cache_mode = force_custom_plan`，一个测试读回（3.16）。
  2. `issue/domain/listvocab.go`（列表的词汇）和测试；`IssueListSettings` 由 `issue.Provide` 交出（3.17）。
  3. 列表的请求的解析（422，含 `group_key`、`sub_group_key`、分组时不带 `group_key` 的 `cursor`、日期的 `none`）、游标的载荷、`api/testdata/issue-order.json`（含基本平面以外的字符的一对）和 Go 的比较。
  4. 目录端口加比较键；`VisibleProjects`；`PublicProfiles`（`assignee_name` 的目录含每个当过负责人的账户，3.16）。
  5. 不分组的查询和窗口的 `total`；空的一页的 `total`；数组参数从不为 `nil`；一次读在一个只读的 `REPEATABLE READ` 事务里（`platform/postgres` 的 `TxManager` 选项 `ReadOnlySnapshot`）。
  6. 分组的查询（`ROWS FROM`、`filtered AS MATERIALIZED`、窄的 CTE、每一级的窗口总数、`to_char` 的日期键、多值的组、`none`）。
  7. 子分组的查询和 `subgroup_totals`。
  8. `listIssues`（含 `archived=true`、`group_key`）和 `IssueList`。
  9. `listMemberIssues`（`relation`、可读的项目、各项目的访客规则；`member_issue.list`）。
  10. 访客规则的一致性测试（含子工作项数、"管理员·访客"）；篡改的游标（8.6）。
  11. 每个排序码两个方向的翻页、每个分组码、每个筛选参数的集成测试；`group_key` 的"加载更多"（含"`label_id=A` 的组 C"和日期的 `none` 组）；快照的测试；1 万、10 万个工作项的耗时；P5 的 spec 记下 S1 的计划和两个已知的代价（3.16）。
  12. 迁移 `00026`（`project`）：三列和 CHECK；`get/updateProjectPreferences` 加三个字段，取值经 `IssueListSettings`。
  13. 矩阵（含个人主页的一行）；整程序测试。
  14. 端到端：I3、I4（`link_count` 之外，含 `target_date=none`）、I5、I6、I7、I19、I20 的接口版本；I15 加上已归档的列表。
  15. 3.23 中 P5 的行；review。
- **关闭**：M2 交接第 2 节的列表一部分（每种排序有自己的载荷和测试，改过的游标只能在看得到的数据里换起点）；M3 交接第 3 节的个人主页的一行、第 5 节的服务端一侧（显示设置的格式）。
- **完成线**：I3、I4（`link_count` 之外的排序码、日期的 `none`）、I5–I7、I19、I20 的接口版本和 I15 的已归档的列表通过；10 万个工作项的耗时在 3.16 的实测之内；连接的参数读回为预期的值。

### P6 `issue-events`：领域事件、只投递的客户端、自动归档与测试的时钟（后端，11 个任务）
- **目标**：事件和分发器；经只投递的客户端在事务里投递；两个队列和启动时的检查；自动归档；运行中的 nerve 可以偏移时钟。
- **任务**：
  1. `platform/jobs` 的 `NewInserter(pool)` 和 `InsertTx`；`platform/postgres` 的 `Tx`；`bootstrap/jobs.go` 接成 `Inserter`；提交留任务、回滚不留的集成测试（S4 的结论写进 spec）。
  2. `shared/event.go`；`bootstrap/events.go` 的分发器和测试。
  3. P1–P3 的用例的发布点（创建、修改、删除：子树里每个工作项一个、归档、恢复、父子、关联）和假的发布者的测试。
  4. P4 的用例的发布点（评论、表情、描述的修改）。
  5. `platform/jobs`：两个队列、worker 数、定时任务的种类和 `EnqueuedKinds()` 的种类没有 worker 时构造失败（两种变异测试）；测试的余量改成期限（M3 交接第 1 节）。
  6. `bootstrap` 的接合（各模块的 `Jobs()`、`EnqueuedKinds()`）；`jobs_test.go` 等每个种类 `completed`。
  7. `clock.Offset`、`clock.offset`（只在测试环境）、组合用它。
  8. 自动归档（`ArchivePolicies`，每个项目一个事务，执行者为空，动态、事件）和固定时钟的测试；契约里 `Project.archive_in` 的说明。
  9. 交错 18。
  10. 端到端：`fixtures/clock.ts`（自己的数据库和 nerve，不复用 worker 的 `db`、`nerveWith`、`databaseSnapshot`；偏移的 nerve 上重新登录；只认偏移的 nerve 启动之后完成的任务；对照的数据经偏移的 nerve 写下，对抗性评审 M20）；I16 的接口版本。
  11. 3.23 中 P6 的行、README 的 P6 行；review。
- **关闭**：M1-P5 交接的服务端一侧（自动归档的执行者为空）；M2 交接第 3 节（自动归档推进固定时钟）、第 6 节（漏登记 worker 的变异让测试失败；订阅者的写法写进设计；grants）、第 13 节（`clock.ts`）；M3 交接第 1 节的一部分（邀请的一步留在删除工作区的事务里；`platform/jobs` 的两处余量）、第 4 节的 `archive_in` 的说明。
- **完成线**：I16 的接口版本通过；两种漏登记的变异测试失败；提交留任务、回滚不留。

### P7 `issue-links`：链接、出站客户端与抓取（后端，12 个任务）
- **目标**：链接的增删改查；第一个订阅者经 River 抓取元数据；防 SSRF 的出站客户端。
- **任务**：
  1. 迁移 `00027`，grants；删除的语句、`deleteIssue` 自己的连带、`furnishIssueRows` 加链接。
  2. 链接的领域（规整、协议、长度）；`createIssueLink`、`listIssueLinks`。
  3. `updateIssueLink`、`deleteIssueLink`（409 `issue.link_exists`、404 `issue.link_not_found` 和文案；动态第 17–19 行；工作项的 `updated_at`）。
  4. `platform/outbound`（6.8：IANA 两张特殊用途登记表的每个网段、本机的接口地址）和它的表测试。
  5. `outbound.allow_cidrs`（只在测试环境，别的环境启动失败）；`config.test.yaml` 放行 `127.0.0.1/32`。
  6. 抓取：取页面和图标、解析、上限（`issue/adapter/crawler`）；`metadata.error` 只有 `blocked`、`failed`，细分进日志。
  7. `issue.crawl_link` 的任务和 worker，`EnqueuedKinds()` 声明它；带守卫的写回。
  8. `LinkCrawls` 端口、订阅者 `crawl_on_link_change`；提交留任务、回滚不留的集成测试。
  9. `registrationRows` 核对 `river_job`（M3 交接第 1 节）。
  10. 交错 19。
  11. 端到端：本机的页面服务器；I10 的接口版本（写 `127.0.0.1:<端口>/page`，被拒绝的只断言 `error: blocked`，对抗性评审 M18）；I17 的前置数据加上链接。
  12. 3.23 中 P7 的行、README 的安全一行；review。
- **关闭**：M2 交接第 3 节的一部分（链接的写改工作项的 `updated_at`）、第 6 节的订阅者一项（第一个订阅者落地）；M3 交接第 1 节的一部分（由请求投递的任务进入 `registrationRows` 的核对）。
- **完成线**：I10 的接口版本（含被拒绝的 `169.254.169.254` 写下 `blocked`）、I17 加上链接之后通过；6.8 的表测试通过。

### P8 `drafts-search`：草稿、搜索与 `link_count`（后端，13 个任务）
- **目标**：个人草稿和发布；工作项的搜索和选择器的排除；pg_trgm；`link_count` 进入列表。
- **任务**：
  1. 迁移 `00028`，grants；删除的语句加草稿（项目的，和按工作区的、没有项目的）；`furnishIssueRows` 加草稿。
  2. 草稿的领域；`listDrafts`、`createDraft`、`getDraft`（只列项目未删除的或没有项目的，3.15）；草稿的游标载荷和篡改的测试（另一个人的草稿的 `id` 放进游标，8.6）。
  3. `updateDraft`、`deleteDraft`（项目 S、409 `issue.draft_changed` 和文案、404 `issue.draft_not_found`、已删除的项目 422 `project_id`）。
  4. `publishDraft`（创建的核心，经写闸门，存着的载荷经 `bodyshape`）；与创建同一个 400 的测试（M2 交接第 4 节）。
  5. 交错 16、17。
  6. S2 的余下部分（迁移角色在 `postgres:18.6` 里建扩展）；迁移 `00029`，列进 `issue` 的 `schema:`，生成的 `similarity()` 的类型检查；grants 的测试。
  7. 搜索的领域：编号的解析和折叠、数字的匹配、顺序；`q` 为空时给出最近更新的（3.18）。
  8. `searchIssues`（可读的项目、访客规则、`for` 的排除、日期的条件含 `none`）。
  9. `Issue.link_count` 和它的两个排序码（列表的 SQL、游标的载荷、`issue-order.json` 加行；约束 5）。
  10. 矩阵（草稿、搜索）；整程序测试。
  11. 端到端：I14、I18 的接口版本；I4 加上 `link_count` 的两个方向。
  12. 10 万个工作项上的搜索耗时：英文、中文的标题，两个字和三个字以上的查询（3.18），写进 spec。
  13. 3.23 中 P8 的行、README 的部署一行；review。
- **关闭**：M0-P4 交接的服务端一侧（迁移建出扩展，确认镜像里有它；部署说明在收尾转给 M8）；M1-P2 交接（搜索结果没有 `page`）；M2 交接第 2 节的草稿的游标、第 4 节的草稿发布一项。
- **完成线**：I14、I18 的接口版本和 I4 的 `link_count` 通过；交错 16、17 通过；中文查询的耗时写进 spec。

### P9 `purge`：物理清理、描述版本的保留与工作项一侧的删除任务（后端，13 个任务）
- **目标**：软删除超过 60 天的行按模块物理删除；每个工作项至多 20 个描述版本；删除项目、工作区之后工作项一侧的行由任务软删除（决策点 7）；三种后台的批量删除与请求的写、与彼此不相遇，有测试守住。
- **任务**：
  1. 迁移 `00030`–`00033`（每个模块的 `deleted_at` 索引）。
  2. `issue` 的清理者（子先于父，每张表按 `id` 的键集分批，3.21）。
  3. `project`、`workspace` 的清理者（同样的键集）。
  4. `identity` 的清理者（`api_tokens`，同样的键集）。
  5. `bootstrap/purge.go` 的一个任务和三个配置。
  6. 清理的目录测试（每张有 `deleted_at` 的表有清理者，外键是 `CASCADE` 或 `SET NULL`）。
  7. 清理的集成测试：61 天删、59 天留，带动态和评论的工作项，PAT、标签、项目、工作区。
  8. 描述版本的保留（`issue.trim_description_versions`，每个项目一个事务，只看未删除的项目里未删除的工作项的未删除的版本；从 P6 移来，对抗性评审 M21）。
  9. `project.deleted`、`workspace.deleted`：两个删除的用例在事务里发布（`workspace.deleted` 的 `Changes` 是这次锁住、删除的项目），`EventTypes()`；`project`、`workspace` 的 `New` 收分发器。
  10. 订阅者 `delete_rows_on_deletion`（每个删除的项目一个任务，删除工作区另加没有项目的草稿一个）、`RowDeletions` 端口、任务 `issue.delete_rows`（键集分批、幂等、45 秒的预算和 `JobSnooze(0)`，`EnqueuedKinds()` 声明它）；`ProjectGate.DeletedAt`，任务和 `deleteIssue` 的关联一步据它跳过属于先删除的那一端的关联行（3.5）；删除 `IssueCascade` 的两个过渡方法和命令行的组合里的那一项（6.6）。
  11. 窗口的测试（任务之前、之后每个读写的回答相同，3.6）、任务的测试（重做、用尽重试、预算用完时 snooze 而不算一次尝试、在清理之后执行）、交错 24、25；规模的通过线：五个目标规模的项目的工作区，每个任务在一次运行里做完（没有 snooze、没有重试），耗时和期间别的项目的写的延迟记进 spec。
  12. 端到端：I21；`expectProjectDeleted`、`expectWorkspaceDeleted` 对工作项一侧的表限时轮询，M3 的删除故事照旧通过。
  13. 3.23 中 P9 的行（含总体设计 5.5）、README 的部署一行；review（后端的交接项在此逐条核对）。
- **关闭**：M2 交接第 5 节（PAT 和带动态的工作项的清理，删除关系图）；M3 交接第 1 节的另一半（删除工作区的连带中工作项一侧挂到事件上，邀请的一步仍在事务里）、第 2 节的物理删除账户一项（不删除）、第 4 节的清理包括 M3 的表。
- **完成线**：I21 通过；M3 的删除故事（P4、W2、W3）在轮询之下确定地通过（W3 的先删项目、再删工作区由交错 25 保证）；目录测试在漏掉一个清理者或一条外键时失败；窗口的测试和交错 24、25 通过；五个目标规模的项目的工作区的每个删除任务在一次运行里做完，耗时写进 spec。

### P10a `web-issue-types`：换型与删除 M6、M7 的工作项列表（9 个任务）
- **目标**：`TIssue`、`TBaseIssue`、`TIssueMap` 换成生成的 `Issue`、`IssueDetail`，每个任务结束时能编译；迭代、模块、项目视图、工作区视图的工作项列表删除（7.2）。
- **任务**：
  1. 生成的类型（`make gen-web`）；本段只用 `Issue`、`IssueDetail`，service 在 P10b、P11 随它们的 store 加入（不留没有调用方的导出）。
  2. 删除 M6、M7 的四个布局根和页面上的列表部分（迭代、模块、项目视图的详情页，`workspace-views/[globalViewId]`，已归档的迭代、模块的根），和共用的布局组件里只为它们存在的分支（分组头的"添加已有"、日历的 `addIssuesToView`、迭代和模块的空状态、活动迭代和进度侧栏对列表 store 的调用、迭代的转移弹窗）。
  3. 删除 M6、M7 的列表 store 和筛选 store、旧 service 里返回 `TIssuesResponse` 的方法、`EIssuesStoreType` 的 `CYCLE`、`MODULE`、`PROJECT_VIEW`、`GLOBAL` 和读它们的分支。
  4. 布局的读取收窄：迭代、模块的列、显示属性、分组删除（3.2）。
  5. 详情、浮层、弹窗、收集箱（M7）、M5 的组件的读取收窄：不再读 `Issue` 没有的字段（`cycle_id`、`module_ids`、`attachment_count`、`estimate_point`、`is_subscribed`、`is_intake`、`description_binary` 等）。
  6. 换型：`Issue`、`IssueDetail` 取代 `TIssue`、`TBaseIssue`、`TIssueMap`（131 个文件，含工作项的那一份的 map），只剩导入、改名（`created_by` → `created_by_id` 一类）和可空性，按目录核对；旧类型从 `@nerve/types` 删除；旧的 service 的签名随之改成 `Issue`、`IssueDetail`，这是 7.2 写明的过渡的类型，到 P10b、P11 为止。
  7. 改到的文件的死成员、oxlint；knip；新的关键词规则 `old-issue-urls`（M5、M6、M7 的旧 service 文件按例外登记）。
  8. vitest 照旧通过；S2 在四个角色下照旧通过；迭代、模块、视图的页面打开时没有失败的请求。
  9. 前端改动清单（M6、M7 的工作项列表删除和它的可见代价，3.23）；review。
- **关闭**：没有整项关闭的交接；M6、M7 要加回的列表记进第 13.2 节，收尾写进它们的交接。
- **完成线**：`tsc`、knip、vitest 通过；`git grep -lw TIssue -- web` 没有输出；`EIssuesStoreType` 没有 `CYCLE`、`MODULE`、`PROJECT_VIEW`、`GLOBAL`；S2 通过。

### P10b `web-list-data`：列表一侧的数据层（14 个任务）
- **目标**：列表的 services 和 stores 改接新接口；`reconciled.ts` 的扩展和 `paged-groups.ts`；项目页挂载时不再请求旧接口。
- **任务**：
  1. `issue`、`issue-list` 的 service，换掉旧 `issue.service.ts` 的列表方法和 `issue_archive.service.ts`（7.2 的过渡在列表一侧结束）；项目显示设置的三个字段并入 `preferences` 的 service；列表的形状和枚举换型（`IssueList`、`IssueOrderBy`、`IssueGroupBy`、`IssuePriority`；`TIssuesResponse` 一类、`EIssueGroupByToServerOptions`、`EServerGroupByToFilterOptions` 删除）。
  2. `Reconciled.fetchPart` 和 vitest。
  3. `paged-groups.ts` 的数据模型：组的行 `{id, key, updatedAt}`、固定的 `boundary`、没有回答的组按空的建出、`subgroupTotals`；全序逐个码位比较，读 `issue-order.json`。
  4. `paged-groups.ts` 的修改和重做：旧属、新属、拖放的落点、重做的三种判断、`stale`；3.22 的 vitest 表（含对抗性评审 I2 的九个反例）。
  5. `base-issues.store.ts` 重写（第一页、一组带 `group_key` 的下一页、`stale` 的重取），保留 `getFilteredWorkItems`。
  6. 工作项的一份（`ReconciledByKey`：服务端的 `updated_at`，描述只由 `IssueDetail` 写，读到成员关系改变时标 `stale`）。
  7. 项目的 `filter.store.ts` 读写显示设置；删除 `/user-properties/` 的两个旧方法、关键词的例外、e2e 的 `workItemFilters` 和三个故事里的放行（M3 交接第 5 节）。
  8. 筛选的表达式翻译成 `IssueFilters`；否定的一层删除（M1-P3 交接）。
  9. 已归档、个人主页的筛选存在 `localStorage`（已归档的键修正）；草稿页没有筛选：删除 `WorkspaceDraftIssuesFilter`（`store/issue/workspace-draft/filter.store.ts`，它继承本段重写的 `IssueFilterHelperStore`）和 `store/issue/root.store.ts`、`hooks/store/use-issues.ts` 里对它的引用（聚焦复核 M7）。
  10. `IssueRootStore` 的 `autorun` 改为计算属性；`rootStore` 的三处同步读取；列表一侧的 `useSessionSWR`。
  11. 看板、日历的 store：只用确认的结果，`oneAtATime`，拖动的 `sort_order` 在轮到发出时才算，`PATCH` 不带 `id`、`project_id`。
  12. 确认的修改写进每一份列表和工作项的那一份；每个修改一张 vitest 的表。
  13. 挂载时的取数：项目页取列表和显示设置；S2 的 `REQUESTS`；改到的文件的死成员、oxlint；关键词规则 `issue-optimistic`、`negation-filters`。
  14. 3.23 中 P10b 的行（总体设计 7.7）、前端改动清单；review。
- **关闭**：M1-P3 交接的筛选显示层（否定的一层删除；终态在 P16 核对）；M2 交接第 12 节的 `autorun`、三处同步读取、列表一侧的 SWR 键；M3 交接第 5 节（例外和 `workItemFilters` 删掉）、第 6 节（`autorun`、`reconciled.ts` 的扩展、S2 的 `REQUESTS`）。
- **完成线**：`tsc`、knip、vitest 通过；S2 在四个角色下没有旧接口的请求；`store-context.test.ts` 没有严格模式的警告；`/user-properties/` 的例外已删除。

### P11 `web-detail-data`：详情一侧的数据层（14 个任务）
- **目标**：详情的 services 和 stores 改接新接口；数据层的 7 处模块级实例消失；store 持有的弹窗开关由宿主结束。
- **任务**：
  1. 子工作项、关联、链接、评论、表情、动态、描述版本、草稿、搜索的 service；按编号取。
  2. 详情的类型（评论、表情、链接、关联、动态、描述版本、草稿、搜索）换成生成的；`JSONContent`、`comment_json`、`is_bot`、动态的细节类型删除。
  3. 详情一侧的使用方改到能编译（含 M5、M7 的组件）。
  4. `issue`、`sub_issues`（状态分布由目录算）、`relation` 的 store。
  5. `link`、`comment`、`comment_reaction`、`reaction` 的 store。
  6. `activity` 的 store：读完全部页、从最后一页的游标再读，与评论合并（3.13）。
  7. 描述版本、草稿的 store（发布先 `PATCH` 再 `publish`）。
  8. 模块级实例中数据层的 7 处（`WorkItemVersionService` 2、`WorkspaceDraftService` 1、`searchEntity` 4，7.1）；`getUserProfileIssues` 改接；编辑器的 @提及 改为从成员过滤，`entity-search` 删除。
  9. 按需的取数被拒绝时有提示或交回（M3 交接第 6 节），vitest。
  10. 弹窗的宿主：`WorkItemLevelModals` 留在原处，在 `projectId ?? workItemIdentifier` 改变时和卸载时关上开关；详情根 store 的开关、`peekIssue` 和三份数据由宿主结束；`isArchiveIssueModalOpen` 删除（7.1）。
  11. 错误按码经文案表（`issue-modal/base.tsx` 的两处）；工作项页面上新建标签经 `useLabelChanges().create`。
  12. M3 交接第 6 节的另外三件（工作区一级的标签、`fetchParentStats`、键宽）。
  13. 详情一侧的关键词规则（`label-create-direct`）；改到的文件的死成员、oxlint；`workspace-draft-issues/base.ts` 的 16 行随类型删除或写明。
  14. 每个修改一张 vitest 的表；review。
- **关闭**：M1 收尾交接的 `JSONContent`、`comment_json`；M2 交接第 12 节的 7 处实例、`getUserProfileIssues`、详情一侧的 SWR 键；M3 交接第 5 节的 `is_bot`、第 6 节（按需的取数、错误的文案、标签的新建、宿主、三件）、第 8 节的 `workspace-draft-issues/base.ts` 的 16 行。
- **完成线**：`git grep` 不再找到数据层的那 7 处模块级实例；`is_bot` 不在 `web/` 里；`core/services/issue/` 里没有用 `Issue`、`IssueDetail` 去类型 Plane 的响应的方法（7.2 的过渡结束）；vitest 覆盖每一处拒绝和宿主的清理。

### P12 `web-list-pages`：列表、看板、表格、日历（15 个任务）
- **目标**：项目的工作项页的四种布局、拖动、快速添加、快捷操作、浮层；"添加已有"的选择器经 `issue-search`。
- **任务**：
  1. 列表布局：分组、每组加载更多、快速添加（字段的提示和焦点）；I7 的页面版本。
  2. 看板和泳道（泳道的总数读 `subgroup_totals`）；确认之后才移动、被拒绝时回到原处、拖放的卡片留在放下的地方；I6 的页面版本。
  3. 表格：列、按 3.16 的排序码排序、无限滚动。
  4. 日历的月、周和拖动；I4 的页面版本（含"添加已有"）。
  5. 快捷操作的菜单（子菜单用键盘打开、"激活"的底色）；`quick-action-dropdowns/issue-detail.tsx` 菜单按钮里的 `IconButton` 不再是按钮里的按钮；右键菜单关着时 `inert`。
  6. 行里的属性下拉框：7 个下拉框的 popper 和一份打开状态、`DateDropdown`（M2 交接第 10 节）。
  7. 列表页的层叠顺序和焦点的轮廓（7.5）。
  8. 浮层的打开、关上，宿主卸载时清空。
  9. `ExistingIssuesListModal` 改接 `issue-search`（7.5）：日历的"添加已有"（`target_date=none`）；子工作项、关联的两个调用方的参数换成 `for=sub_issue`、`for=relation`；去掉它的正的 `tabIndex`（`:70,156`）；它的 `projectIssuesSearch` 实例消失（7.1）。
  10. I3 的页面版本。
  11. `project-address-route`：22 个文件改读路由参数，规则的范围扩大（M3 交接第 8 节）。
  12. 列表页的 `watchPage`；S2 的更新。
  13. 改到的布局文件的死成员、oxlint。
  14. vitest；C1、C3 的准备。
  15. 3.23 中 P12 的行（README 的"前端"一节）；review。
- **关闭**：M2 交接第 10 节的列表一侧（`DateDropdown`、7 个下拉框）、第 12 节的一处实例（`existing-issues-list-modal.tsx` 的 `projectIssuesSearch`）；M3 交接第 6 节的 `watchPage` 一项、第 7 节的列表一侧（`CustomMenu` 的键盘、右键菜单、层叠顺序、快捷操作的嵌套按钮、`existing-issues-list-modal.tsx` 的 `tabIndex`）、第 8 节的 22 个文件。
- **完成线**：I3、I4（含日历的"添加已有"）、I6、I7 的页面版本通过；C1 和 C3 的列表一侧通过。

### P13 `web-filters-archives-profile`：筛选、显示设置、归档页、个人主页（12 个任务）
- **目标**：筛选的工具栏和显示设置；已归档的工作项；个人主页的工作项。
- **任务**：
  1. 富筛选的工具栏（4 个 `CustomSearchSelect` 调用方的文字经 `t()`、`defaultOpen` 的核对）；`FiltersDropdown` 的名字；日期弹窗的 `title` 经 `t()`。
  2. 显示设置（布局、分组、排序、空组、子工作项、显示属性）的保存；I5 的页面版本。
  3. 日期范围的字段提示（`range.tsx`）。
  4. 已归档的列表和详情（只读），页头的模块级 `IssueService` 删除；I15 的页面版本。
  5. 归档、恢复的操作；归档弹窗的 `ModalTitle`、去掉 `tabIndex={1}`。
  6. 个人主页的三个标签页；访客不取列表、工具栏不在"没有权限"之上；`as TProfileViews` 删除；I19 的页面版本。
  7. 以页面自己的 id 给子树 key：个人主页（`userId`、`profileViewId`）、已归档的详情（`archives/issues/(detail)/[archivedIssueId]`）；后退的检查。
  8. 已归档、个人主页的 `localStorage` 设置的页面一侧。
  9. `watchPage`；vitest。
  10. C2、C9 的归档页和个人主页。
  11. 改到的文件的死成员、oxlint。
  12. review。
- **关闭**：M1 收尾交接的 `as TProfileViews`；M2 交接第 10 节的 `CustomSearchSelect` 一项、第 12 节的一处实例（已归档的页头）；M3 交接第 6 节（个人主页的工作项、访客的工具栏、个人主页和已归档的详情的 key）、第 7 节的筛选按钮和日期弹窗、第 8 节的 `CustomSearchSelect` 的文字。
- **完成线**：I5、I15、I19 的页面版本通过；`git grep -n "as TProfileViews" -- web` 没有输出；C2 和 C9 的归档页、个人主页通过。

### P14 `web-issue-detail`：详情、创建弹窗、父子、关联、链接（16 个任务）
- **目标**：工作项的详情页和浮层、创建弹窗、子工作项、关联、链接、删除；选父工作项经 `issue-search`；访客的边界；编辑器的定点修改。
- **任务**：
  1. `issues/:issueId` 的跳转（去掉 `clientLoader` 和模块级的 `IssueService`）；`browse/:workItem` 经按编号取、`useSessionSWR`，以 id 给子树 key；收集箱的跳转删除。
  2. 详情的属性和标题（`title-input.tsx` 的提示照 M3 的写法）；`label-select.tsx` 的 `baseTabIndex` 删去；I2 的页面版本。
  3. 描述的保存：`skip_activity` 为布尔值；"撤销再重做"的修复（M1-P2 交接）；不发 `description_json`。
  4. 创建弹窗：`ModalTitle`、去掉正的 `tabIndex`、编辑已有的工作项和打开草稿时读 `isMigrationUpdate`（vitest 写这两种）、按码的错误、字段的提示和 `useRefusedFocus`、项目选择的提示、弹窗里的列表 `modal={false}`；I1 的页面版本。
  5. 嵌套的弹窗："Save this draft?"（`confirm-issue-discard.tsx:40` 改经 `ModalTitle`）和选父工作项的 Escape；描述编辑器的 Escape 监听只理会自己的弹窗。
  6. 子工作项的区（添加已有、新建、去掉父工作项、状态分布）、父工作项的选择：`ParentIssuesListModal` 经 `for=parent`（`sr-only` 的名字、去掉 `tabIndex`、去掉悬停才显示的链接，它的 `projectIssuesSearch` 实例消失）；I8 的页面版本。
  7. 关联的区（跨项目、两边任一个删除；选择器是 P12 改接的 `ExistingIssuesListModal`）；收集箱的查重改接 `issue-search`、改到能编译（`projectIssuesSearch` 的最后一处）；I9 的页面版本。
  8. 链接的区（弹窗的 `ModalTitle` 和地址的提示、元数据、复制）；I10 的页面版本。
  9. 删除确认框写明子树（`onSubmit` 保持必传）；I17 的页面版本。
  10. 编辑器的代码分割；个人设置页不挂载工作项的弹窗（M1-P4、M2 交接第 7 节）；删除 `EMOJI_CHECK_WARNING`（定义、29 个故事文件的 96 行、`settings-pages.ts` 的一处）。
  11. 标注块的原生表情；5 个扩展的 Markdown 序列化删除；`worker-src` 的复核写进 review。
  12. 编辑器关闭图片；附件区、关注按钮不挂载（3.1）。
  13. 表格的子工作项列直接链接 `/browse/…`，去掉 `#sub-issues`（M1-P4 交接）。
  14. I20 的页面版本（列表只有自己的、别人的工作项的地址"找不到"、不能建、不能改）；S2 按身份打开一个工作项的详情（2 节）；I22；后退的检查（弹窗、浮层、`WorkItemLevelModals`）。
  15. 详情的复制改经 hook（`identifier-text.tsx`、`links/link-item.tsx`、关联和子工作项的 `helper`、`issue-detail-quick-actions.tsx`、浮层的页头），删除它们的 `copy-hook` 例外。
  16. C3 的详情一侧、C4、C5、C10、C11；review。
- **关闭**：M1-P2 交接（撤销再重做）；M1-P4 交接（弹窗的挂载、`#sub-issues`）；M1 收尾交接（`isMigrationUpdate`、Markdown 序列化）；M2 交接第 7 节、第 8 节（标注块、`worker-src`）、第 11 节的详情一侧、第 12 节的 3 处实例（跳转 1、`projectIssuesSearch` 2）；M3 交接第 6 节的工作项页面的 key、第 7 节的弹窗、名字、`tabIndex`（`parent-issues-list-modal.tsx`、`label-select.tsx`）、嵌套的按钮、字段的提示、弹窗里的列表、列表里不是选项的控件、嵌套的弹窗、描述编辑器的 Escape，第 8 节的删除确认框。
- **完成线**：I1、I2、I8、I9、I10、I17、I20、I22 的页面版本通过（选择器经 `issue-search` 的排除）；S2 按身份打开详情；C3 的详情一侧、C4、C5、C10、C11 通过。

### P15 `web-collaboration-drafts-search`：时间线、评论、表情、描述的历史、草稿、搜索、命令面板（15 个任务）
- **目标**：时间线和评论、表情、描述的历史；草稿页；搜索；命令面板；自动归档的显示。
- **任务**：
  1. 时间线：`IssueUser` 与 `IssueCreator` 合并，执行者为空显示站点名，`archived-at.tsx` 的特判、`default.tsx` 的来源一支删除；`git grep` 核对没有按邮箱或名字猜测系统用户的代码（M1-P5 交接）；I12 的页面版本。
  2. 动态的筛选：嵌套的按钮（`activity-filter.tsx`）、`helper.tsx` 的两条 oxlint。
  3. 评论：发（空的不能发）、改（只有作者）、删（作者，和项目管理员删别人的，决策点 8）；菜单按钮不再嵌按钮；每条评论的控件以作者为名（en、zh-CN）；I11 的页面版本；I20 加上评论自己建的工作项。
  4. 工作项和评论的表情；表情选择器没有 CSP 违规（C6）。
  5. 描述的历史（`ModalTitle`、去掉 `tabIndex={1}`、一条 oxlint）和还原；I13 的页面版本。
  6. 草稿页：列表、编辑、删除、移到项目；草稿页的 3 个只读工作区的文件改用 `useRouteWorkspace()`；I14 的页面版本。
  7. 创建弹窗的"存为草稿"。
  8. 命令面板：工作项的命令拆开（`commands.ts`）；主题命令在应答成功之后才应用（M2 交接第 9 节）；`power-k/config/creation/command.ts` 读路由的 slug；`workspace-address-route` 的文件正则放宽为 `\.tsx?$`、加上这 4 个文件。
  9. 搜索：命令面板经 `issue-search`，项目在列表里过滤，`WorkspaceService.searchWorkspace` 的模块级实例消失；I18 的页面版本。
  10. 旧的 `ProjectService` 文件、`WorkspaceService` 的搜索方法、工作区搜索的类型和参数（36 行）删除。
  11. I16 的页面版本（时钟）。
  12. 其余的复制改经 hook（`create-issue-toast-action-items.tsx`、`issue-activity/helper.tsx`、快捷操作的 `helper.tsx`、命令面板），`copy-hook` 的例外全部删除。
  13. C6（含时间线、表情、描述的历史）、C7、C8、C9 的草稿。
  14. 改到的文件的死成员、oxlint。
  15. review。
- **关闭**：M1-P3 交接的收集箱来源的一支（`default.tsx`；取值转交 M7）；M1-P5 交接（自动归档的显示、没有猜测的代码）；M2 交接第 8 节的表情回应、第 9 节、第 11 节的余下部分、第 12 节（最后一处实例，`git grep` 的门禁）；M3 交接第 5 节的旧 `ProjectService`、`WorkspaceService`、搜索的类型，第 7 节的每一行的控件、评论的嵌套按钮，第 8 节的复制、只读工作区的 4 个文件、3 个文件的 oxlint、`commands.ts`。
- **完成线**：I11–I14、I16、I18 的页面版本和 I20 的评论通过；`copy-hook` 没有 M4 的例外；`git grep` 不再找到 M4 的 13 处模块级实例；C6–C8 和 C9 的草稿通过。

### P16 `web-cleanup`：死成员、`no-shadow` 与守卫的终态（11 个任务）
- **目标**：M4 领域的死成员、oxlint、关键词守卫到达终态；跨 M 的共用件按交接处理。
- **任务**：
  1. 数一次 `getTabIndex` 的调用方：没有了就删除它和常量、`CustomMenu` 读 `tabIndex` 的一支；还有 M6、M7 的就写进它们的交接（M3 交接第 7 节）。
  2. `@nerve/ui` 下拉框的泛型和 13 个调用方（M3 交接第 8 节）。
  3. `eslint(no-shadow)` 全仓清零，升为错误（M1 收尾交接）。
  4. 死成员：`--rows M4` 的余下的行和按路径归到 M3 的 52 行，清零或写进 review（M1 收尾交接、M3 交接第 8 节）。
  5. oxlint 各包的上限调低（`tools/lint-cap.mjs`）。
  6. 关键词守卫的终态：`old-issue-urls` 的例外只剩 M5、M6、M7 的旧 service 文件；`negation-filters` 没有命中。
  7. 成员 store 的 `projectMemberIds`：M4 的两个读者（`issue-layouts/utils.tsx`）改完；M7 的读者还在，删除转交 M7（M3 交接第 8 节）。
  8. 前端改动清单的 M4 一行、README 的"前端"一节的终稿。
  9. 评审找到的 vitest 的缺口；S2 的终稿。
  10. C12 的准备（三种浏览器）。
  11. review。
- **关闭**：M1-P3 交接的筛选显示层的终态；M1 收尾交接（死成员、oxlint 的两项）；M3 交接第 7 节的 `getTabIndex`、第 8 节的共用件和死成员。
- **完成线**：`domains.mjs --rows M4` 清零或逐行写进 review；`no-shadow` 全仓 0；各包的警告数等于新的上限。

### 收尾 `closeout`（14 个任务）
1. `tools/keywords.json` 的 `phase` 改为 `M4/closeout`，守卫通过（`copy-hook` 没有 M4 的例外）。
2. 全部故事、门禁在主干上通过；不稳定的故事修正。
3. 控制者的浏览器核对 C12（Chromium、Firefox、WebKit）。
4. 整个 M4 的独立评审；修复的第一轮。
5. 浏览器核对之后的修复轮。
6. 3.23 逐行核对。
7. 差异清单、前端改动清单的终稿（4.17 的每一行都已登记）。
8. 13.1 的每一条交接逐条写结论，交接的 `status` 改为 `closed`。
9. 写给 M5–M8 的交接（13.2）：M2 交接第 1 节原样转进 M5 的交接（附一句：只投递的客户端已在 `platform/jobs`）；M0-P4 交接的部署要求原样转进 M8 的交接；M5、M6 加表时扩充 `furnishIssueRows`、订阅删除的事件；M6、M7 加回 P10a 删除的工作项列表。
10. 本文和总体设计 9.4 标为完成；第 15 节的进度表。
11. 死成员、oxlint 的终数，写进收尾的评审。
12. 性能的数字（列表对照 3.16 的 2 万个工作项的目标、搜索、删除的请求和任务、交错 20 的连接池、清理）写进收尾的评审，交给 M8。
13. 收尾的评审记录。
14. 合并的准备。

---

## 13. 交接的落点

### 13.1 交给 M4 的交接
**M4 自己的 8 份**（`docs/v0/M4-issue-core/handoffs/`）。各 Phase 的"关闭"一行与这张表的 Phase 一一对应（第 12 节）：

| 交接与节 | 落在哪里 | 关闭条件（在哪个 Phase 满足） |
|---|---|---|
| M0-P4 `pg_trgm` 的索引 | 3.18、4.1（`00029`）、8.8 | 迁移建出扩展和 `issues.name` 的 GIN 索引，在持续集成的 Postgres 18 镜像上运行；S2 的余下部分（迁移角色建扩展、生成的类型）写进 spec（P8）。部署要求（迁移账户不需要超级用户）不在 M4 关闭：原样写进 M8 的交接，另加 `LC_CTYPE` 的要求（3.18；13.2，收尾） |
| M1-P2 不再使用的字段和接口 | 4.2、5.1、5.2、3.18 | 接口没有 `estimate_point`、`description_binary`、`/issue-dates/`（P1a）；搜索结果只有工作项，没有 `page`（P8） |
| M1-P2 只补 id 的迁移更新 | 3.7、7.6 | `skip_activity` 为真、内容标识（纯文本、提及、链接的地址、勾选框的状态）不变时不记动态、不写版本、不改 `updated_at`，有测试，S3 的样例含经接口写入的 HTML（P4）；页面发布尔值（P14）；I2 的一步 |
| M1-P2 描述保存的"撤销再重做" | 7.6 | `description-input/root.tsx` 的保存在撤销、重做之后发出编辑器里显示的内容，vitest 覆盖（P14） |
| M1-P3 不再读、不再写的字段 | 4.2、4.7、4.11、5.2 | 接口没有 `is_epic`、`type_id`（P1a），评论没有 `access`、动态的字段名里没有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`（3.13 的表，P4） |
| M1-P3 地址 | 5.1 | 按编号取是 `/workspaces/{slug}/issues/{identifier}`（P2），描述历史是 `/issues/{issue_id}/description-versions`（P4），与其余工作项地址统一；没有批量删除、批量归档（P1a） |
| M1-P3 富文本筛选的显示层 | 3.22、7.7 | 不支持否定的运算符：`TAllAvailable*ForDisplay` 的否定一层和 `useFiltersOperatorConfigs` 删除（P10b），关键词规则 `negation-filters` 看住，终态没有命中（P16） |
| M1-P3 收集箱的来源 | 3.2、7.7 | **以转交关闭**（同 M2 交接第 1 节的做法）：收集箱属于 M7（总体设计 9.2），来源的取值由 M7 定。收尾把这一项连同它的细节写进 M7 的交接（13.2）：`EInboxIssueSource` 的 `IN_APP`、`FORMS`、`EMAIL`，前端只创建 `IN_APP`（`inbox-issue.service.ts`）；收集箱详情对 `FORMS` 显示"Intake Form user"的一支（`inbox/content/issue-root.tsx`）；项目功能设置里收集箱的说明只描述保留的用法，保留表单或邮件来源时要改写（M1-P3 交接第 31 行）。M4 自己的部分：删除工作项动态中显示来源的一支（`issue-activity/activity/actions/default.tsx`，P15），工作项的动态不再读来源 |
| M1-P4 个人设置页上的工作项弹窗 | 7.4、7.6 | 去掉挂载（P14）；它在打开之前建的编辑器随之没有（M2 交接第 7 节） |
| M1-P4 `#sub-issues` 片段 | 7.4 | 去掉片段：表格的子工作项列直接链接 `/browse/…`（P14） |
| M1-P5 系统代为创建和操作的工作项 | 决策点 4、3.8、7.7 | 负责人的裁定写进 3.8；自动归档的动态执行者为空（P6），页面显示站点名，`archived-at.tsx` 的特判删除，`git grep` 核对前端没有按邮箱或名字猜测系统用户的代码（P15）。收集箱的创建者一半**以转交关闭**：收集箱在 M7（总体设计 9.2），收尾把"由系统代为创建的工作项记在谁名下、怎样显示，不要恢复按邮箱或名字猜测的写法"连同 M1/P5 删掉的两处（`inbox/sidebar/inbox-list-item.tsx` 的 "Plane" 头像、`issues/peek-overview/properties.tsx` 的 `-intake` 一支）写进 M7 的交接（13.2） |
| M1-closeout 死成员和死 prop | 7.7、7.8 | `domains.mjs --rows M4` 的每一行消失或写进 review（P10a–P16，P16 汇总）；`JSONContent` 和 `comment_json` 一起删除（P11） |
| M1-closeout `as TProfileViews` | 7.7 | 个人主页的视图段在路由处收窄，`git grep -n "as TProfileViews" -- web` 没有输出（P13） |
| M1-closeout 弹窗把描述的 ID 迁移当作改动 | 3.7、7.6 | 弹窗编辑已有的工作项、打开草稿时读 `isMigrationUpdate`，不把表单标成已改；vitest 写这两种，C4 核对（P14） |
| M1-closeout 标注块的 Markdown 序列化 | 7.6 | 删除：`git grep -n "markdown: {" -- web/packages/editor/src` 没有输出（P14） |
| M1-closeout oxlint | 7.8 | 改到的文件 0 条；`eslint(no-shadow)` 全仓清零并升为错误；各包上限调低，review 写明（P10a–P16，P16 汇总） |
| M2-closeout §1 只投递的 River 客户端 | 3.20、6.6 | 只投递的客户端在 P6 加入 `platform/jobs`（服务的组合用它在事务里投递）；M4 没有投递任务的命令（6.6），命令行的组合不需要它：M4 的 review 写明，本节**原样**写进 M5 的交接，附一句"客户端已在 `platform/jobs`，第一个投递任务的命令加入时交给命令行的组合"（13.2，收尾的第 9 个任务）；之后本交接才能改为 `closed`（负责人："做好记录，以后别漏了"） |
| M2-closeout §2 列表的游标 | 3.16、8.6 | 每种排序有自己的载荷，翻页不重复、不遗漏，排序键相同时按 `id`；改过的游标只能在调用者看得到的数据里换起点：评论、动态、描述的历史（P4），工作项的列表（P5），草稿（P8，另一个人的草稿的 `id` 不泄露任何东西） |
| M2-closeout §3 自动归档读 `updated_at` | 3.7、3.8 | 写工作项的每个用例用时钟写 `updated_at`，测试断言等于固定时钟（P1–P4、P7）；自动归档的测试推进固定时钟（P6） |
| M2-closeout §4 请求体检查与草稿发布 | 3.7、5.2、6.1 | 路径按下标的数值排序、开放节点的数的范围、`date` 的检查器各有测试；代价测试加上字符串的数组（P2）；map 型、递归的 schema 写明"没有用到"（5.2，P2 的 review）；草稿发布与创建工作项给出相同的 400（P8） |
| M2-closeout §5 物理清理与删除关系图 | 3.21、4.16 | 4.16 延伸到 M4 的表；清理的集成测试覆盖超过 60 天的 PAT 和带动态的工作项（`issue_activities.issue_id` 在 M4 是 `CASCADE`，不再挡住工作项的物理删除，4.16）（P9） |
| M2-closeout §6 定时任务和事件订阅者 | 3.19、3.20、4.18 | 定时任务的种类和模块声明会投递的种类没有 worker 时启动失败，两种漏登记的变异让测试失败（P6）；订阅者的写法写在 3.19（本文；第一个订阅者在 P7）；每张新表在同一个提交里进 `deploy/runtime-grants.sql`，两个权限测试通过（每个加表的 Phase） |
| M2-closeout §7 编辑器的代码分割 | 7.6 | 编辑器到用它的页面才加载；`EMOJI_CHECK_WARNING` 的定义、29 个故事文件的 96 行和 `e2e/fixtures/settings-pages.ts` 的一处删除，S2 和个人设置的故事仍断言控制台没有别的警告（P14） |
| M2-closeout §8 CSP | 7.6、8.5 | 标注块的默认表情改为原生表情，插入标注块时没有 CSP 违规、不请求 `cdn.jsdelivr.net`（P14、C10）；表情回应的选择器同样核对（P15、C6）；编辑器不用 `blob:` 的 worker，不加 `worker-src`，P14 的 review 在打包之后的产物上复核并写明；`frame-src` 归 M7（M7 的交接已有） |
| M2-closeout §9 命令面板的主题命令 | 7.4 | 应答成功、且标签页仍在发出修改时的会话才 `setTheme`，失败时按 `code` 提示；vitest 和 C7（P15） |
| M2-closeout §10 下拉框 | 7.5 | `DateDropdown` 只有一份打开状态、能用键盘打开；7 个下拉框的列表在按钮旁（P12）；`CustomSelect` 的 3 个调用方、`CustomSearchSelect` 的 4 个富筛选调用方和 `defaultOpen` 在浏览器中核对（C2、C3；P12、P13、P14） |
| M2-closeout §11 复制到剪贴板 | 7.5 | 6 处改经 `useCopyLink`、`useCopyText`，失败时经 `t()` 提示（P14、P15，与 M3 交接第 8 节的 10 个文件一起）；C8 |
| M2-closeout §12 stores 按会话分代 | 7.1、7.3 | `IssueRootStore` 的 `autorun` 改为 `rootStore.router` 上的计算属性，没有要释放的反应，"随退役的一代释放"由此满足，`store-context.test.ts` 没有严格模式的警告（P10b）；13 处模块级实例按 7.1 的分配消失（P11 7 处、P12 1 处、P13 1 处、P14 3 处、P15 1 处），`git grep` 的门禁在 P15；三处同步读取不发请求（P10b）；SWR 键带 `loginId`（P10b、P11）；`getUserProfileIssues` 改接（P11） |
| M2-closeout §13 e2e 的 `clock.ts` | 3.20 | `e2e/fixtures/clock.ts` 按 M0-P6 的写法加入，I16 用它，不等真实的时间；nerve 一侧的 `clock.offset` 只在测试环境（P6） |
| M3-closeout §1 事件与删除工作区的连带 | 3.6、3.19、3.20 | M3 自己的连带留在删除的事务里，邀请的一步照旧同步（P6 写明）；工作项一侧在决策点 7 选 A 时挂到 `project.deleted`、`workspace.deleted` 上、由任务完成，写明窗口和失败的处理（P9，3.6、3.19）；`registrationRows` 核对 `river_job`（P7）；`jobs_test.go:265`、`runner_test.go:136` 改为期限（P6） |
| M3-closeout §2 加锁的约定和方案 E | 3.5、决策点 1、11.2、11.11 | 方案的选择和代价写在 3.5、决策点 1（负责人裁定 B：只锁项目行；两个代价和 B′ 写明；S5 的证据；连接池的代价和写闸门，A、B 相同）；锁之前的预检拒绝一切会被拒绝的写、不取锁（P1a，加关联的另一端在 P3）；连接池的代价和写闸门（P1b）；每个新的批量写逐个按约定五核对（3.5，P1a–P9 各自的 review；删除标签摘下 `issue_labels` 不需要预锁，理由在 3.5）；没有新的成员关系增长（3.5），不加与停用的交错测试（P1a）；M4 不物理删除账户（3.21，P9） |
| M3-closeout §3 权限规则和错误码 | 3.4、5.3、9.2、9.4 | 工作项的每个操作在规则表、操作名、矩阵中各有一行（P1a 起每个后端的 Phase）；`AllowCreator`、`guest_view_all_features`，访客规则按项目的角色（"管理员·访客"一列）（P2）；个人主页的规则行（访客看不到别人的）在矩阵（P5）；完整性测试和 `apitest.Main` 通过（每个后端的 Phase） |
| M3-closeout §4 状态、标签和项目的数据 | 3.6、3.7、4.16、5.1 | 删除状态时还有工作项 409 `project.state_in_use`；删除标签时从工作项上摘下（`DetachLabel`）；默认负责人只在仍是项目当前的、不是访客的成员时指派，写进契约；`identifier`、`default_assignee_id`、`State.default` 的说明；`workspaceTables` 从目录读出，`furnishIssueRows` 让 M3 的删除故事断言到每张新表（P1a 起每个加表的 Phase）；`expectIssueDeleted` 照 `expectProjectDeleted` 的目录自查、沿外键图递归，`guest_view_all_features` 的说明（P2）；`archive_in` 的说明（P6）；删除断言的限时轮询（P9）；4.16 的图 |
| M3-closeout §5 显示设置与旧的 service | 3.17、7.3、7.9 | 显示设置的三列和格式（P5）；`/user-properties/` 的两个方法、关键词的例外和 `workItemFilters` 删除（P10b；例外的 `until` 在 P1a 的第一个提交里改为 `M4/P11`，7.9）；`projectIssuesSearch` 的三个调用方：添加已有的弹窗改接 `issue-search`（P12），选父工作项改接、收集箱的查重改到能编译（P14）；旧 `ProjectService` 文件、`WorkspaceService` 的搜索方法、工作区搜索的类型和参数（36 行）删除（P15）；`is_bot` 删除（P11） |
| M3-closeout §6 stores、会话和页面的取数 | 7.1、7.3、7.4 | `autorun`（P10b）；按需的取数被拒绝时有提示或交回（P11）；`error?.error` 的两处经文案表、标签的新建经 `useLabelChanges().create`（P11）；工作区一级的标签、`fetchParentStats`、键宽（P11）；`WorkItemLevelModals` 留在原处、在 `projectId ?? workItemIdentifier` 改变时和卸载时关上开关，详情 store 的开关、`peekIssue` 由宿主结束（P11）；S2 的 `REQUESTS`（P10b）、`watchPage`（P12）；个人主页的工作项和访客的工具栏，个人主页和已归档的详情以 id 给子树 key（P13）；工作项页面以 id 给子树 key、后退的检查（P14） |
| M3-closeout §7 键盘、下拉框和弹窗 | 7.5 | 每一项的浏览器核对写进 review（C1–C6、C12）；`CustomMenu` 的键盘、右键菜单、层叠的元素降到 `z-30` 之下、快捷操作的嵌套按钮、`existing-issues-list-modal.tsx` 的正的 `tabIndex`（P12）；筛选按钮、日期弹窗、归档弹窗（P13）；工作项的弹窗、名字、正的 `tabIndex`（含 `parent-issues-list-modal.tsx`、`label-select.tsx`）、字段的提示和 `useRefusedFocus`、弹窗里的列表 `modal={false}`、列表里不是选项的控件、嵌套的弹窗和描述编辑器的 Escape（P14）；每一行的控件以那一行为名、评论和动态筛选的嵌套按钮、描述历史的弹窗（P15）；`getTabIndex` 数一次调用方（P16） |
| M3-closeout §8 地址、复制、文案和留下的行 | 7.5、7.8、7.9 | `project-address-route` 的范围含 22 个文件（P12），`workspace-address-route` 含 4 个文件，文件正则放宽为 `\.tsx?$` 以含 `power-k/config/creation/command.ts`（P15）；`copy-hook` 没有 `M4/closeout` 的例外（P14、P15，收尾核对）；3 个文件的 oxlint 清零、`commands.ts` 拆开（P15）；删除确认框的 `onSubmit` 保持必传（P14）；`workspace-draft-issues/base.ts` 的 16 行有结论（P11）；`CustomSearchSelect` 的 4 个调用方传经 `t()` 的文字（P13）；`@nerve/ui` 下拉框的泛型（P16）；`projectMemberIds` 的 M4 读者改完、删除转给 M7（P16，13.2）；每个 Phase 的第一个提交改 `phase` 和到期的 `until` |
| M3-closeout §9 页大小移到 `shared` | 3.16、6.1 | 第二个分页列表（评论）合并时页大小在 `shared`，`identity` 改用它（P4） |

**别的 M 的交接中点名 M4 的**：

| 交接与节 | M4 的部分 | 落点 |
|---|---|---|
| M5 的 M3-closeout"嵌套的 id 没有 key" | 附件的确认框和上传的弹窗在工作项的页面上，随 M4 给这些页面的 key 结束 | P14 的第 1 个任务以工作项的 id 给子树 key；附件区在 M4 不挂载（3.1），M5 接上时只需核对 |
| M6 的 M3-closeout 添加已有工作项 | `existing-issues-list-modal.tsx` 改用 M4 的工作项搜索 | 它是 M4 的选择器（项目日历的"添加已有"、子工作项的"添加已有"、关联的选择，7.5）：P12 让它改接 `issue-search`（日历带 `target_date=none`，另两个经 `for=sub_issue`、`for=relation`），它的正的 `tabIndex` 由 M4 删去；迭代、模块经它加已有工作项的分支（分组头的两处、日历的 `addIssuesToView`、`empty-states/cycle.tsx`、`module.tsx`）随 M6 的列表在 P10a 删除，M6 照 Plane 的源码在新的引擎上加回（7.2、13.2） |
| M6 的 M3-closeout 正的 `tabIndex`（第 34 行） | "M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`" | P16 的第 1 个任务数调用方；M6、M7 还有时写进它们的交接（13.2） |
| M6 的 M2-closeout 日期范围和项目的下拉框 | M4 改好的 `dropdowns/date-range.tsx`、`dropdowns/project/base.tsx` 也在迭代、模块的表单里 | P12 改；M6 接上时核对（13.2 的 M6 一行复述） |
| M7 的 M3-closeout 收集箱查重 | `select-duplicate.tsx` 改用 M4 的工作项搜索 | `issue-search` 在 P8；P14 让它改接之后能编译（同一个任务去掉 `projectIssuesSearch` 的模块级实例），M7 核对它的页面 |
| M7 的 M3-closeout `getTabIndex` 和 `projectMemberIds` | "M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex`"；"最后改完读者的那一个删除 `projectMemberIds`" | P16 的第 1、7 个任务数调用方；M6、M7 还有读者时，删除写进它们的交接（13.2） |
| M7 的 M2-closeout `frame-src` | M4 的交接第 8 节同一项 | M4 不加；M7 照它自己的交接 |
| M8 的 M3-closeout 事件 | 工作区、项目、成员的 Webhook 事件加在 M4 引入的事件机制上 | 3.19 的事件名单和 `EventPublisher`；`project.deleted`、`workspace.deleted` 在 P9 加入（决策点 7），M8 的 Webhook 可以直接订阅；不发布事件的连带（3.6）的清单写进 M8 的交接；决策点 3 若选 B，M8 要先回头补事件（第 10 节），写进 M8 的交接 |
| M8 的 M3-closeout 层叠顺序、弹窗的名字、`AlertModalCore`、`CustomSearchSelect` 的文字 | M4 的页面上 `z-30` 及以上的元素、M4 自己的弹窗经 `ModalTitle`、M4 的调用方传经 `t()` 的文字 | P12–P15（7.5）；M4 改完之后 M8 只剩全应用的核对和把 prop 改为必传（13.2） |
| M8 的 M1-closeout 发往第三方的请求（标注块的默认表情图） | 同 M4 的 M2 交接第 8 节 | P14 改为原生表情之后，M8 对这一项只剩核对（13.2） |

### 13.2 M4 交给后续 M 的事项（收尾时写成交接）
每一行写进接收者的 `handoffs/M4-closeout.md`（收尾），连同各 Phase 的 spec 第 5 节、评审第 6 节交给它的各条和收尾找到的。

| 接收者 | 事项 |
|---|---|
| [M5](../M5-files/handoffs/M4-closeout.md) | **M2 收尾交接第 1 节原样**（命令行的只投递 River 客户端；负责人："做好记录，以后别漏了"），注明 M3、M4 都没有投递任务的命令，并附一句：只投递的客户端已在 `platform/jobs`（`NewInserter`，M4/P6），第一个投递任务的命令加入时把它交给命令行的组合，并照 3.20 声明它会投递的种类。**附件**：`Issue.attachment_count` 和它的排序码、显示属性"附件"（3.2）；编辑器去掉 `disabledExtensions` 里的 `image`，图片的上传和显示经 M5 的文件接口（3.1、7.6）；工作项详情的附件区、评论的附件；草稿若带附件，发布时随之移到工作项；`file_assets` 指向工作项、评论、草稿的外键在 4.16 的图上延伸，`deleteIssue` 的连带和删除的语句随之延伸（第 12 节约束 8），60 天清理加上 M5 的清理者（3.21 的写法）。**内容标识**：打开图片之后，`skip_activity` 的内容标识加上按顺序的 `image-component[src]`（3.7，对抗性评审 I4），换一张图不能不留痕迹。**删除**：M5 的表加进 `furnishIssueRows`，M3、M4 的删除故事随之断言到它们；挂在项目、工作区下的行在删除项目、工作区时照 3.6 的写法由订阅 `project.deleted`、`workspace.deleted` 的任务软删除，写下请求的时刻和删除者（决策点 7）。**事件**：附件的增删是否另有事件，由 M5 照 3.19 的写法定 |
| [M6](../M6-cycles-modules/handoffs/M4-closeout.md) | **加回 P10a 删除的工作项列表**（7.2，对抗性评审 I7）：迭代、模块的列表 store 和筛选 store、两个布局根、迭代和模块的详情页与已归档的根上的列表部分、`EIssuesStoreType` 的 `CYCLE`、`MODULE` 和它们的分支、分组头的"添加已有"、日历的 `addIssuesToView`、迭代和模块的空状态、活动迭代和进度侧栏对列表 store 的调用、迭代的转移弹窗；照 Plane 的源码写在 `paged-groups.ts` 和新的列表接口上，不恢复 Plane 的列表引擎；`existing-issues-list-modal.tsx` 的迭代、模块调用方随之加回（弹窗本身 M4 已改接 `issue-search`、删去正的 `tabIndex`）。`Issue.cycle_id`、`module_ids` 和它们的写入（3.2）；列表的筛选、分组、排序加上迭代、模块（3.16 的词汇在 `issue/domain/listvocab.go`：加取值、加 SQL 的一支，`issue-order.json` 加行；`IssueListSettings` 随之认它们）；显示属性的"迭代""模块"；自动归档的条件加上迭代、模块的条件（3.8）；`cycle_issues`、`module_issues` 的外键、连带、清理者，加进 `furnishIssueRows`，删除项目、工作区时照 3.6 订阅两个删除的事件；`getTabIndex` 若 M6 还有调用方，最后一个删去的删除它和两份 `tab-indices.ts`（P16 的计数）；`dropdowns/date-range.tsx`、`dropdowns/project/base.tsx` 在迭代、模块的表单里核对 |
| [M7](../M7-collaboration/handoffs/M4-closeout.md) | **收集箱**（M1-P3、M1-P5 交接以转交关闭，原文的细节一并写进）：来源的取值（`EInboxIssueSource` 的 `IN_APP`、`FORMS`、`EMAIL`；前端只创建 `IN_APP`；收集箱详情对 `FORMS` 显示"Intake Form user"的一支，`inbox/content/issue-root.tsx`；保留表单或邮件来源时改写项目功能设置里收集箱的说明）；由系统代为创建的工作项记在谁名下、怎样显示（照决策点 4 的结论；不要恢复按邮箱或名字猜测的写法；M1/P5 删掉了 `inbox-list-item.tsx`、`peek-overview/properties.tsx` 的两处）；自动归档的条件加上收集箱的状态（3.8）；工作项动态的动词 CHECK 为收集箱的状态放宽（4.11）；`select-duplicate.tsx` 的页面核对。**通知**：订阅、取消订阅的接口和 `is_subscribed`（3.12）；M4 的关注只由服务端写，关注者后来可能读不到这个工作项（降为"只看自己"的访客、标志关掉、被移出），所以通知在发出的那一刻按 `issue.read` 加访客规则重新判定（对抗性评审 M5）；通知由 3.19 的事件产生（删除时子树里每个工作项一个 `issue.deleted`）；动态的 `comment` 列是否进入接口（3.13）。**视图**：加回 P10a 删除的项目视图、工作区视图的工作项列表（`EIssuesStoreType` 的 `PROJECT_VIEW`、`GLOBAL`，两个布局根，`workspace-views/[globalViewId]` 和项目视图详情页的列表部分，7.2），照 Plane 的源码写在新的引擎上；跨项目的列表（3.16 的引擎以项目为参数，加工作区的一支）；个人主页按标签分组时工作区一级的标签（3.16）。**其他**：`services/inbox/inbox-issue.service.ts` 的参数在 P10a 换成 `Partial<Issue>`，仍调 Plane 的收集箱地址（7.2 写明的过渡的类型，M7 随收集箱的接口换掉）；命令面板的迭代、视图、收集箱结果；删除工作项时清掉最近访问（4.16）；`frame-src`；`projectMemberIds` 的最后一个读者改完时删除它；`getTabIndex` 同 M6 一行 |
| [M8](../M8-open-release/handoffs/M4-closeout.md) | **Webhook** 订阅 3.19 的事件（工作项、评论、关联、链接，和 P9 的 `project.deleted`、`workspace.deleted`；事件名单是契约的一部分）；**不发布事件的连带**只剩删除标签摘下的 `issue_labels`（3.6），要由 M8 给标签加的事件覆盖；`platform/outbound` 给 Webhook 的投递复用（6.8，IANA 的两张登记表和本机的接口地址）。**部署**：`pg_trgm` 的部署要求（M0-P4 交接，原样）和库的 `LC_CTYPE`（pg_trgm 取三元组依赖它，中文的搜索，3.18）；`outbound.allow_cidrs` 只在测试环境（8.8）；清理和删除任务的配置（3.21、3.6）；连接的参数 `jit = off`、`plan_cache_mode = force_custom_plan` 由 nerve 自己设，部署只核对它们没有被覆盖（3.16）；连接池的大小与写闸门的 K、同时热的项目数（多个进程时每个进程各 K 个；同时热的项目达到池的大小除以 K 时等锁的写仍能占满池，11.11），以及一个工作项的写的等待：闸门上至多 `issue.write_gate.max_wait`（或到请求的期限），数据库里每一次等锁各至多 `issue.write_lock_timeout`，同时热的项目达到 ⌈池 / K⌉ 个时另有池里的等待，只以请求的期限为界；配置的规则（`max_wait` 加 `write_lock_timeout` 短于 `server.request_timeout`）只盖住闸门和一次等锁，等到请求的期限的写答 500（M4/P1b spec 第 5 节）；`nerve users set-email` 在持续写入的账户上要等外键的 `FOR KEY SHARE`（3.5，对抗性评审 M2）：给它加 `lock_timeout` 和重试的提示。**性能**：列表的 v0 目标是一个项目至多约 2 万个未删除的工作项（3.16 的实测，决策点 7），更大的项目、按 `-created_at` 的单列索引式查询、个人主页的跨项目列表（顺序扫描整个工作区）由 M8 按收尾的数字决定；搜索、删除的请求和任务（P1b、P9 的 spec）、清理的数字。**核对**：标注块的默认表情图在 M4 改为原生表情之后，M8 的 M1 收尾交接的那一项只剩核对；`CustomSearchSelect` 的三个文字和 `AlertModalCore` 的按钮文字在 M4–M7 的调用方都传了之后改为必传 |

---

## 14. 完成标准
- [ ] 第 2 节的 22 个故事（页面版本、接口版本，含"断言的增长"一列的全部断言）和冒烟故事 S1、S2 在持续集成中通过。
- [ ] 权限矩阵的全部格子（含创建者、访客的两种、在项目里只是访客的工作区管理员、已不是成员的身份、不是成员的工作区管理员）、访客规则一致的测试（含子工作项数、动态的过滤、关注）、3.5 的 26 个交错测试在真实数据库上通过；看不到项目、不能写的写和加关联指向看不到的项目都不取锁；交错 20 经真实的连接池和写闸门，闸门满时 503 `server_busy`。
- [ ] 每个模块的 `apitest.Main` 两个方向核对通过（`issue` 模块和 `project` 新的 `project.state_in_use`）；`access/domain` 的判定表、规则表与各模块操作名的完整性测试通过；每个操作在矩阵里有一行。
- [ ] 列表：每个排序码两个方向的翻页、每个分组码和子分组、每个筛选参数、带 `group_key` 的"加载更多"的集成测试通过；`api/testdata/issue-order.json` 由 Go 和 vitest 各读一次，结果相同；改过的游标只在调用者看得到的数据里换起点（8.6）；10 万个工作项的耗时在 3.16 的实测之内，连接的参数读回为预期的值；`paged-groups.ts` 的 vitest 表（含对抗性评审 I2 的九个反例）通过。
- [ ] 事件和任务：经只投递的客户端，提交留任务、回滚不留的集成测试；定时任务的种类、声明会投递的种类漏登记 worker 的两种变异让测试失败；`registrationRows` 核对 `river_job`；自动归档、版本的保留推进固定时钟；`platform/jobs` 的测试没有固定的余量。
- [ ] `platform/outbound` 的表测试（6.8：IANA 两张登记表的每个网段、本机的接口地址、跳转、重绑定）通过；I10 中被拒绝的 `169.254.169.254` 写下 `blocked`。
- [ ] 清理的目录测试（每张有 `deleted_at` 的表有清理者，外键是 `CASCADE` 或 `SET NULL`）和 I21 通过；删除项目、工作区的任务的窗口测试、交错 24、25 通过，M3 的删除故事在限时轮询之下通过。
- [ ] 持续集成的全部门禁通过（含架构测试、`TestSQLCSchemaScope`、grants 的两个测试、`apitest` 的写法规则、命令行的组合测试、关键词守卫、oxlint 上限、knip、`make gen-check`）。
- [x] 第 10 节的八个决策点有负责人的裁定，本文按裁定修改；第 11 节的十一个架构问题已由负责人批准。（2026-10-17：八个决策点都按建议裁定，每个决策点之后写着裁定，正文各节按它们写定；11.1–11.11 各标"负责人已批准，2026-10-17"；本文开头的状态一行和 17.4 同样写明。）
- [ ] 13.1 的每一项已关闭（或以写明的转交关闭）并写进相应 Phase 的 review，8 份交接 `status: closed`；13.2 的交接已写好，M2 交接第 1 节原样在 M5 的交接里，M1-P3、M1-P5 的收集箱一项连同细节在 M7 的交接里。
- [ ] 3.23 的每一行已在对应的 Phase 同步；差异清单按 4.17 逐行核对过；前端改动清单有 M4 的一节；README 按 8.8 更新。
- [ ] 浏览器核对 C1–C12 写进各 Phase 的 review（C12 在收尾）。
- [ ] `domains.mjs --rows M4` 没有输出（或剩下的每一行写明是误报或测试自己读的）；`eslint(no-shadow)` 全仓为 0 并升为错误；oxlint 各包等于调低的上限。
- [ ] M4 能到达的页面挂载时没有旧接口请求，也没有调用者无权读的请求（故事的 `watchPage`，S2 的 `REQUESTS` 在四个角色下）；`copy-hook` 没有 M4 的例外；`git grep` 不再找到 M4 的 13 处模块级 service 实例。
- [ ] M4 页面的组件在 `await` 之后的跳转、提示先核对会话；每个修改的 hook 的 vitest 有"换账户之后兑现"的一行；C11 通过。
- [ ] 性能的数字写进收尾的 review：1 万、2 万、10 万个工作项的列表第一页（不分组、分组、子分组），搜索（含中文、两个字的查询），在规模目标上删除项目和工作区（2 万个工作项的项目、五个这样的项目的工作区：请求和任务各自的耗时），一次清理；交给 M8。
- [ ] 规模估计与实际的对比写进收尾的 review。
- [ ] 总体设计 9.4 中 M4 的状态改为"已完成"。

## 15. Phase 进度表
| Phase | 内容 | 状态 |
|---|---|---|
| 设计 | 本文（第三稿，按对抗性评审和聚焦复核修订） | 第一稿经独立评审（2026-10-16，17.1），第二稿按控制者的裁定逐条落实；第二稿经对抗性评审（2026-10-16，17.2），第三稿照控制者的裁定逐条落实；第三稿经聚焦复核（2026-10-16，17.3），再照裁定修订；第 10 节的八个决策点已由负责人裁定、第 11 节的十一个架构问题已由负责人批准（2026-10-17，全部按建议，17.4）；验证 S1、S4、S5、SP1–SP4 已做，S2 做了一部分，S3 在 P4（0.3） |
| P1a `issues` | 工作项的表、项目行的锁、创建与连带（后端，15）；设计中的 P1 由控制者按决策点 2 的预先批准拆出（2026-10-17，第 12 节） | 已完成：[spec](specs/P1a-issues.md)、[plan](plans/P1a-issues.md)、[评审](reviews/P1a-issues-review.md)（执行时 15 个 Task；整分支评审之后一个修复轮） |
| P1b `issue-update` | 工作项的取、改与写闸门（后端，12）；同上 | 已完成：[spec](specs/P1b-issue-update.md)、[plan](plans/P1b-issue-update.md)、[评审](reviews/P1b-issue-update-review.md)（执行时 15 个 Task；整分支评审之后一个修复轮） |
| P2 `issue-access` | 访客规则、创建者、删除、按编号取、归档、起止日期（后端，14） | 已完成：[spec](specs/P2-issue-access.md)、[plan](plans/P2-issue-access.md)、[评审](reviews/P2-issue-access-review.md)（执行时 14 个 Task；整分支评审之后一个修复轮） |
| P3 `issue-hierarchy` | 父子与关联（后端，16） | 已完成：[spec](specs/P3-issue-hierarchy.md)、[plan](plans/P3-issue-hierarchy.md)、[评审](reviews/P3-issue-hierarchy-review.md)（执行时 16 个 Task；整分支评审之后一个修复轮） |
| P4 `issue-collaboration` | 描述、清洗、提及、版本、评论、表情、动态的列表（后端，16） | 进行中：[spec](specs/P4-issue-collaboration.md)、[plan](plans/P4-issue-collaboration.md)；负责人已批准清洗适配器方案（2026-10-25），保留纯层架构守卫，见[评审记录 A1](reviews/P4-issue-collaboration-review.md)；完成后由负责人安排 Claude 整分支终审，尚未合并 |
| P5 `issue-list` | 列表引擎与显示设置（后端，15） | 未开始 |
| P6 `issue-events` | 领域事件、只投递的客户端、自动归档与测试的时钟（后端，11） | 未开始 |
| P7 `issue-links` | 链接、出站客户端与抓取（后端，12） | 未开始 |
| P8 `drafts-search` | 草稿、搜索与 `link_count`（后端，13） | 未开始 |
| P9 `purge` | 物理清理、描述版本的保留与工作项一侧的删除任务（后端，13） | 未开始 |
| P10a `web-issue-types` | 换型与删除 M6、M7 的工作项列表（9） | 未开始 |
| P10b `web-list-data` | 列表一侧的数据层（14） | 未开始 |
| P11 `web-detail-data` | 详情一侧的数据层（14） | 未开始 |
| P12 `web-list-pages` | 列表、看板、表格、日历（15） | 未开始 |
| P13 `web-filters-archives-profile` | 筛选、显示设置、归档页、个人主页（12） | 未开始 |
| P14 `web-issue-detail` | 详情、创建弹窗、父子、关联、链接（16） | 未开始 |
| P15 `web-collaboration-drafts-search` | 时间线、评论、表情、描述的历史、草稿、搜索、命令面板（15） | 未开始 |
| P16 `web-cleanup` | 死成员、`no-shadow` 与守卫的终态（11） | 未开始 |
| 收尾 `closeout` | 交接、同步核对、浏览器核对 C12、规模和性能的数字（14） | 未开始 |

## 16. 风险
| 风险 | 影响 | 应对 |
|---|---|---|
| M4 是 v0 最大的里程碑：37 个新操作、19 个迁移、前端 106 个任务，约是 M3 前端实际的 1.9 倍（7.10） | 周期拉长；某段的 plan 超出约 1,500 行 | 按任务数分成 17 段加收尾，每段 9–16 个任务（第 12 节，P10 已预先拆开）；超出时按约束 2 预先批准拆成 P<k>a、P<k>b（决策点 2）；决策点 2 的 B（拆成两个 M）写明了代价。已发生一次：P1 在实施前按 M3 已合并的 plan 实测估出约 22–25 个任务、约 23,000–26,000 行，控制者按决策点 2 的预先批准裁定（2026-10-17）拆成 P1a（15 个）、P1b（12 个），并预先批准 P1a 仍超出时把状态、标签的连带（连同交错 6、7 的创建一支）和 I1 的故事文件移到 P1b；两段在上限之内可以重新切分任务。现在是 18 段加收尾 |
| 列表的代价：排序用不上索引，每个请求扫描、排序项目里满足筛选的全部行（S1，3.16） | 很大的项目上列表慢（10 万个工作项时分组 0.2–0.7 s、泳道 0.7–1.0 s） | 写法已在 S1 验证（`ROWS FROM`、`MATERIALIZED`、窄的 CTE、窗口总数），没有绕过 sqlc 的退路（`TestModulesRunSQLOnlyThroughSQLC`）；v0 的规模目标是一个项目至多约 2 万个未删除的工作项，交负责人确认（决策点 7）；连接的参数由 nerve 设（`jit`、通用计划）；更大的项目交给 M8 的性能工作（13.2） |
| 删除一个大的项目、工作区（SP2：2 万个工作项的项目 8.6 秒，五个这样的项目的工作区 36.5 秒，请求的期限 15 秒） | 同步的连带在规模目标上删不掉一个工作区，期间工作区的每个项目的写都在等 | 决策点 7（负责人裁定 A）：工作项一侧的行由任务分批软删除，请求只做 M3 的几步；每个删除的项目一个任务，按 `id` 的键集（SP4：没有键集的语句是平方的，一个项目的 40 万行动态 53.9 秒，键集 1.8 秒），每次运行 45 秒的预算、用完就 snooze 而不算一次尝试；窗口里的读写经项目的门当作不存在（3.6，逐条测试）；任务失败时行留在窗口的样子，随项目的物理清理消失；P1b 在规模目标上量，P9 的通过线是每个任务一次运行做完 |
| 工作项一侧的删除任务是第一个跨模块的异步效果（决策点 7 的 A） | 窗口里某个读写漏了项目的门，就会看到已删除的项目里的工作项；一条行写成了别的删除的时刻和删除者 | 3.6 逐条列出读写的路径和它经过的门；9.3 的窗口测试在任务之前、之后各问一次；矩阵的"项目已删除"一列在任务前后各跑一次；每个项目只由它自己的任务写，跨项目的关联行属于先删除的那一端（3.5），交错 24、25 断言每一行写下的是谁的删除；P1–P8 的过渡写法与任务用同一组语句和编排 |
| 一个热的项目占满连接池（SP1：200 个写排在一个项目上时，别的项目的读等约 875 ms；A、B 相同） | 整个实例的请求变慢 | 写闸门（11.11）：每个项目每个进程至多 K 个写在数据库里，其余在进程里等，不持连接，等不到 503 `server_busy`；`lock_timeout` 兜底；交错 20 经真实的连接池断言别的项目的读的耗时。代价是热的项目上排得太久的写得到 503、要由客户端重试（决策点 1）；K 和池的大小在多进程时由 M8 定 |
| `issue_activities` 的增长（每个改动一行，描述的动态存完整的 HTML，3.13） | 表和索引变大，列表的动态读得慢 | 动态按工作项分页读（`(issue_id, created_at, id)` 的索引）；描述的动态 600 秒内合并；Nerve 与 Plane 一样不清理活的动态；M8 的性能测量量它的大小 |
| 前端换型（`Issue` 取代 Plane 的类型）：131 个使用方，列表、详情、浮层、收集箱读同一份工作项 | 一个任务改不完；中途不能编译 | P10a"先收窄、后换型"：先删掉对 `Issue` 没有的字段的读取（`TIssue` 还在，每个任务能编译），最后一个任务只剩导入、改名和可空性的机械替换（7.2）；列表的形状在 P10b、详情的类型在 P11 随各自的 store 换掉 |
| M6、M7 的工作项列表在 P10a 删除（7.2） | M6、M7 之前，迭代、模块、项目视图、工作区视图的页面没有工作项列表 | 可见的代价写进前端改动清单；不留兼容层（项目的规则）；M6、M7 照 Plane 的源码在新的引擎上加回，写进它们的交接（13.2） |
| 收尾超出（M3 的收尾实际 16 个任务，另有多轮修复和浏览器核对之后的修复） | 合并推迟 | 浏览器核对分在各页面段（C1–C11），收尾只做 C12；收尾的任务数按 M3 的实际定为 14，修复轮按 M3 的写法另计 |
| 决策点 1 选 B 之后某个项目成为写的热点 | 同一项目里的写排队，延迟上升，超过闸门的上限时 503 | 每个写是一个短事务；S5、SP1 和交错 20 测饥饿和连接池；热点出现时按行细分锁（3.5 的改写，决策点 1 的"以后改的代价"） |
| 决策点 1 选 B 的两个代价（3.5） | 审计时工作项的写与成员关系的变更的时刻可以与生效的顺序不一致；草稿和 M3 的项目管理类的写仍可能被饿 | 写明在决策点 1，由负责人在 B、B′、A 之间选；授权本身在锁下判定，不受影响 |
| 只投递的客户端不检查种类（S4） | 漏登记的种类安静地留在 `river_job` 里没人做 | 模块声明会投递的种类，建工作的客户端时逐个核对，有变异测试（3.20、6.6） |
| `pg_trgm` 与迁移角色、中文的查询（S2 的余下部分） | 迁移失败；两个字的中文查询走顺序扫描 | S2 已验证 sqlc、类型、授权（0.3）；P8 的第 6 个任务在镜像里建扩展，第 12 个任务量中文的耗时；`LC_CTYPE` 写进 M8 的交接 |
| bluemonday 的放行表与编辑器的输出不一致（S3） | 保存之后格式丢失，或放过不该放的属性 | S3 的往返测试以 `@nerve/editor` 导出的样例为准，每个扩展一例，另有经接口写入的 HTML；放行表只在 `issue/adapter/content` 一处；新扩展加入时往返测试先失败 |
| `skip_activity` 的内容标识不含格式（3.7） | 只改格式的保存带上标志就不留动态和版本 | 写明的口子，登记差异（4.17）和 8.2；文字、提及、链接的去处、勾选框的改动都照常记下，S3 的样例含只改链接地址、只改勾选框的两份；M5 打开图片时加上图片的地址（13.2） |
| 拖动只在服务端确认之后移动卡片（3.22） | 慢网络下拖动显得迟钝 | 请求在路上时卡片停在放下的位置、显示更新中的遮罩，被拒绝时回到原处（3.22）；确认通常在一次往返之内；C1 在节流的网络下核对一次 |
| 前端的列表（分组、每组分页）与服务端的总数走散（3.22） | 页面的数字与实际不符，卡片在两列里出现或消失 | 第三稿按对抗性评审 I2 重写：组存应答写下的行和它们的 `updatedAt`、固定的 `boundary`，重做只按应答自己的证据判断，那一份工作项按服务端的 `updated_at` 取；确知时加减，看不出时只把受影响的组标 `stale` 重取；九个反例各是 vitest 表的一行 |
| 测试时间 | 矩阵、交错、列表的集成测试和 22 个故事的页面、接口两个版本；删除断言的轮询 | 矩阵并行、写的格子各用复制的库（M3 的做法）；端到端按故事并行；10 万个工作项、规模目标上的删除的测量只在单独的性能测试里跑，不进每次的持续集成 |

## 17. 评审的落实

### 17.1 第一稿的独立评审（2026-10-16）
评审结论：Ready with fixes（Critical 0、Important 8、Minor 26）。评审在两个临时的 `postgres:18.6` 容器上做了 S1、S4、S5 和 S2 的一部分（0.3）。控制者接受全部发现，对其中几条给出具体的做法（I1 的 B′ 不建议、I3 用"断言的增长"、I6 选 b、M13 改为端口、M25 用只投递的客户端），第二稿照裁定落实：

| 编号 | 问题 | 处理 |
|---|---|---|
| I1 | 决策点 1 低估了 B 的代价；3.5"与 M3 相同"一句不对：B 之下工作项的写与成员关系的变更之间，时刻可以与生效的顺序不一致；草稿和 M3 的项目管理类的写仍取工作区 S | 3.5 改正那一句，加"B 的两个代价"；决策点 1 的 B 一行写两个代价，加 B′（连带先锁项目、再读时刻，不建议），加 S5 的证据；11.2；0.3 记 S5 |
| I2 | 单组"加载更多"用"不分组的列表加上这一组的筛选"，在多值的字段同时被筛选和分组时表达不了，日期的 `none` 组也接不下去 | 3.16 加 `group_key`、`sub_group_key`，"加载更多"是同一个请求加它们；5.1、5.2、11.6、3.23 的 P5 一行（总体设计 3.4）；3.22 的 `fetchPart` 用它；I3 的接口版本；9.3、9.5 |
| I3 | 几个故事的接口版本排在它们断言的表和操作存在之前（I1、I2 的提及和版本，I15 的评论和关联，I17 的评论和链接） | 第 2 节加"断言的增长"一列，故事在核心断言都已存在的 Phase 加入、之后扩充；第 12 节约束 7，每段的端到端任务写扩充的步骤，完成线只写那时已有的断言；9.6 |
| I4 | S1 必须在设计时定下：§16 的"手写的 pgx 查询"违反 `TestModulesRunSQLOnlyThroughSQLC`；结果要写进 3.16、11.5、§16 | 0.3 记 S1、S2 的结果；3.16 写下可行的形状、`nil` 切片的规则、空的一页的总数、规模的目标（约 2 万）、连接的参数由 nerve 设（P5 的任务和读回的测试）；§16 删去那条退路；11.5 记结论；M8 的交接只核对连接的参数 |
| I5 | `listChange` 让看板最常见的一步（拖进没有载入完的列）变成整个列表的重取 | 3.22 按修改前的那一份和确认的应答判断成员关系，四种组合都确知地加减；只在 `before` 不知道、本地判断不了的筛选（和取数与修改重叠时的版本判断）标 `stale`；写成 vitest 的表（9.5） |
| I6 | 经接口写入的描述没有块 id，第一个打开它的成员触发的保存在逐字比较下算一次修改 | 选 b：`skip_activity` 的保存在纯文本和提及的集合都不变时"内容不变"（3.7）；剩下的口子（只改格式不留痕迹）登记差异（4.17）、写进 8.2；S3 移到 P4，样例含经接口写入的 HTML 和编辑器的重新序列化（0.3、9.1）；§16 改写这一行 |
| I7 | `ExistingIssuesListModal` 是 M4 的选择器，第一稿当成 M6 的，到 P15 才让它编译；选父工作项的排除也在 P15 | 7.5 写明两个选择器经 `issue-search`（`for=parent`、`for=sub_issue`、`for=relation`），在 I8、I9 的页面版本所在的 P14 改接；它的正的 `tabIndex` 归 M4；M6 只留迭代、模块的调用方；7.1、P14 的任务 6、7，P15 的任务 10 改为删除旧 service 和类型；13.1 的第二张表、13.2 的 M6 一行 |
| I8 | 把 `WorkItemLevelModals` 移到项目的 key 之下会让 `/browse/…`、首页、草稿页、个人主页没有创建、删除的弹窗 | 7.1 改为宿主留在原处，在 `projectId ?? workItemIdentifier` 改变时和卸载时关上开关，后退的检查照旧；P11 的任务 10；9.5 |
| M1 | 约定七的理由漏了最坏的后果（活的关联行指向已删除的工作项），交错里没有"加关联与删除对方的项目" | 3.5 的约定七写三种后果（S5 的 E5、E5b）；加交错 23（P3）；11.2；交错数改为 23（9.3、第 14 节） |
| M2 | 版本的保留与清理可以在同一批行上死锁 | 3.14、3.5（取锁表和后台写的一条）、3.21：保留只删未删除的工作项的未删除的版本，软删除的只归清理；9.3 |
| M3 | 端口不一致：`ProjectGate.LockWorkspace` 没有人用；`IssueCascade` 在 3.3 是一个方法、别处是两个；`ShareWorkspaceByID` 已存在 | 3.3、6.1、6.3、6.5 对齐：`ProjectGate` 只有 `Find`、`Lock`、`LockMany`；`IssueCascade` 两个删除的方法；`workspace` 用已有的目录，不加方法 |
| M4 | 多列的 CHECK 没有名字 | 4.2、3.9：`issues_not_own_parent_check` |
| M5 | 访客规则没有处处执行：子工作项数、动态里引用的工作项、自动关注被提及的访客 | 3.4 加三条；3.9 的计数按调用者的规则数；3.13 略去引用看不到的工作项的动态行（"看得到"只看项目和访客规则，不看是否已删除）；3.12 只关注能读的人；4.17、8.3；9.3 的一致性测试 |
| M6 | `IssueRelation` 的字段两处不一致 | 3.10 与 5.2 同为 `{id, relation_type, issue, created_at}` |
| M7 | `issue.deleted` 把子树放进一个事件；连带不发布事件 | 3.6、3.19：子树里每个工作项一个 `issue.deleted`；不发布事件的连带列在 3.6，写进 M8 的交接（13.2） |
| M8 | 交错 21 的"从不取工作区行的锁"不严格；B 之下先锁后判定让知道 id 的旁人排他地锁住项目 | 交错 21 改写（3.5、9.3、决策点 1）；3.5 加不加锁的可见性预检，锁下的判定仍是权威的；6.7 的流程、5.4、8.4；P1 的任务 8 |
| M9 | P1 把锁、清洗、请求体检查放在一起；P4、P7 没写 `deleteIssue` 自己的连带的延伸 | 描述、清洗、提及、版本一起移到 P4（第 12 节约束 5，3.7、5.2、6.1），P1 的工作项没有描述字段；约束 8 和 P2、P3、P4、P7 的任务写明 `deleteIssue` 的延伸；P4 成为 16 个任务，合计 229 |
| M10 | 增量读取只读到新增；拖动的 `sort_order` 要在发出时才算 | 3.13 写明只读到新增并登记（4.17）；3.22、7.1 写明 `sort_order` 在轮到发出时按那时的列表算；9.5 |
| M11 | JavaScript 的 `<` 比较 UTF-16 码元，与 Go、`COLLATE "C"` 的码位序不同 | 3.16、3.22：前端逐个码位比较；`issue-order.json` 加 U+FF5E 与 U+1F600 这样的一对；9.1、9.5 |
| M12 | 中文的查询多是两个字，总走顺序扫描；pg_trgm 依赖 `LC_CTYPE` | 3.18；P8 的任务 11 量中文；M8 的交接写 locale 的要求（13.2） |
| M13 | 11.10 的另一种做法不全；改为词汇在 `issue/domain`，`project` 经端口检查 | 控制者裁定改做法：3.17、11.10 重写（另一种做法是放进 `shared`，不取的理由）；3.3、6.1、6.2、6.3、6.5、6.6 加 `IssueListSettings`；4.15、5.2、9.1、3.23 的 P5 一行、P5 的任务 2、13.2 的 M6 一行 |
| M14 | 13 处模块级实例分在三个 Phase，各处说法不一 | 7.1 定一个分配（P11 7、P13 1、P14 4、P15 1），`git grep` 的门禁在 P15；13.1、各 Phase 的任务和完成线照它 |
| M15 | 关键词守卫的 `until` 两处差一 | 7.9 写明规则；改 `phase` 的提交同时改到期的 `until`，`/user-properties/` 的改为 `M4/P11`；3.17、第 12 节约束 6、P1 的任务 1 |
| M16 | 只读工作区的文件是 4 个还是 3 个；`command.ts` 没有任务，规则的正则不含 `.ts` | 7.4、7.9：4 个，`workspace-address-route` 的文件正则放宽为 `\.tsx?$`；P15 的任务 8 |
| M17 | 弹窗的 `isMigrationUpdate` 针对的是编辑已有的工作项、打开草稿，不是新建；C4 没有这一项 | 3.7、7.6、9.5 写明两种情形；C4 加上；P14 的任务 4 |
| M18 | 描述的历史、草稿的游标没有载荷和篡改的测试 | 8.6；P4 的任务 7、P8 的任务 2（另一个人的草稿的 `id` 不泄露）；13.1 的 M2 第 2 节一行 |
| M19 | 13.1 的"`DetachLabel` 按 `id` 先锁"在 3.5、P1 里没有依据 | 3.5 写明 B 之下每个写 `issue_labels` 的事务都持项目 N，不需要预锁；13.1 去掉那一句 |
| M20 | 收集箱的转交单薄，引的是总体设计 9.4（应为 9.2） | 13.1 的 M1-P3、M1-P5 两行以转交关闭，带上来源的取值、"Intake Form user"、功能设置的说明、M1/P5 删掉的两处；13.2 的 M7 一行；引 9.2 |
| M21 | 第 12 节几段的"关闭"漏了 13.1 放在那一段的项 | 第 12 节每段的"关闭"按 13.1 重写（含 M2 第 3 节的 P7、第 8 节表情回应在 P15 的 C6） |
| M22 | 几项有落点、没有任务 | M1-P5 的 `git grep`（P15 的任务 1）；`Project` 四个字段和 `State.default` 的说明（P1 的任务 9、P2 的任务 2、P6 的任务 8）；已归档的详情以 id 给 key（P13 的任务 7）；`label-select.tsx` 的 `baseTabIndex`（P14 的任务 2）；`confirm-issue-discard.tsx:40` 的 `ModalTitle`（P14 的任务 5，7.5）；`quick-action-dropdowns/issue-detail.tsx` 的嵌套按钮（P12 的任务 5） |
| M23 | 引用和计数有误 | 3.1 的标题改引 M2 设计 3.1；3.3 的 `Sequences` 改引 M3 交接第 4 节；`EMOJI_CHECK_WARNING` 是 29 个故事文件的 96 行，另有定义和 `settings-pages.ts` 的一处（第 1 节、7.6、13.1、P14 的任务 10） |
| M24 | 13.1 的第二张表漏了 M6 交接的 `getTabIndex` 一句；13.2 没有告诉 M8 标注块的默认表情图只剩核对 | 13.1 加 M6 的一行（第 34 行）和 M8 的 M1 收尾交接一行；13.2 的 M8 一行；7.5 写明两份 `tab-indices.ts` |
| M25 | S4 的结果要写进 6.6；可以用只投递的客户端投递 | 6.6 记 S4 的事实；投递经只投递的客户端（在模块之前建出，组合没有环，不再拆 `jobs.New`），另加启动时核对模块声明会投递的种类都有 worker（3.20、6.1、6.4、6.5、11.4、P6 的任务 1、5）；M2 交接第 1 节照旧原样转交，附一句客户端已在（13.1、13.2） |
| M26 | S1 的两个运行上的发现 | 3.16 的"两个已知的代价"写进 P5 的 spec（P5 的任务 11）；M8 的交接记跨项目的成员列表（个人主页）的代价（13.2） |

第二稿另做的、裁定之外的改动（都在正文里写明）：
- 一页为空时没有行带出总数（窗口总数的后果）：带游标的空页另查一次总数（3.16）。
- `listChange` 的重做按版本判断，在取数与修改重叠时只标那一组 `stale`（3.22），使确知的加减在 `Reconciled` 的重做下仍幂等。
- 动态的过滤中"看得到"不看对方是否已删除，已删除的对方的历史行照常显示（3.13）。
- B′ 只去掉连带一侧的代价一；只改工作区角色的那一种另要改角色的写也锁项目（决策点 1）。
- 搜索的 SQL 命名参数叫 `@query`，接口的参数仍是 `q`（3.18，S2 的发现）。
- 被提及、但读不到工作项的访客仍记提及的行，只是不关注（3.12）。

### 17.2 第二稿的对抗性评审（2026-10-16）
评审结论：Ready with fixes, but not ready for the owner as written（Critical 0、Important 7、Minor 22）。加锁的约定、约定七、清理与活的写不相遇、事件和事务内投递的形状经受住了评审构造的每一种交错；七条 Important 改变了要负责人裁定的内容，或改变了 P1、P10 要做的事。评审另在临时的 `postgres:18.6` 上做了 SP1–SP3（0.3）。控制者接受全部发现，I1 改为决策点 7（并入 M15）、M14 改为决策点 8，对 I2、I3、I4、I7 和几条 Minor 给出具体的做法；第三稿照裁定落实：

| 编号 | 问题 | 处理 |
|---|---|---|
| I1 | 同步的连带在规模目标上删不掉一个工作区（SP2：2 万个工作项的项目 8.6 秒、五个这样的项目的工作区 36.5 秒，请求的期限 15 秒） | 新的决策点 7（建议 A：工作项一侧的行由任务分批软删除，写下请求的时刻和删除者），连同规模目标（M15）；3.6 写请求的事务、任务、"一个改变一个时刻"、窗口期间的每个读写、任务失败、与清理的先后、e2e 的轮询；3.19 的 `project.deleted`、`workspace.deleted` 和订阅者；3.5 重新推出删除工作区的取锁、加任务的一行和交错 24；3.15、3.20、3.21；6.2、6.3、6.6；8.3、8.8；9.2、9.3、9.6；3.23 的 P9 行（总体设计 5.5）；P1 量规模、P9 落地；§16 |
| I2 | 3.22 的重做、窗口和那一份工作项会算错（九个反例） | 3.22 重写：组存应答写下的 `{id, key, updatedAt}` 和固定的 `boundary`；修改记 `(beforeSnapshot, after)`；重做只按应答自己的证据判断；没有回答的组按空的建出；那一份取服务端 `updated_at` 大的、描述只由 `IssueDetail` 写、读到成员关系改变时标 `stale`；拖放留在落点并扩 `boundary`；九个反例各是 vitest 表的一行（9.5）；仍是 `reconciled.ts` 的一个扩展；7.3 的 store 表；P10b 的任务 3–6；§14、§16 |
| I3 | 一个热的项目占满 10 个连接的池（SP1：别的项目的读等约 875 ms） | 架构问题 11.11 写闸门（`platform/gate`、`WriteGate` 端口、每个项目每个进程 K 个、等的上限、503 `server_busy`、`lock_timeout` 兜底）；3.5、6.1、6.2、6.5、6.6、6.7、8.8；交错 20 经真实的连接池（9.3）；决策点 1 和 §16 写代价（A、B 相同）；池的大小和 K 交给 M8（13.2）；P1 的任务 8、14 |
| I4 | 内容标识只有纯文本和提及，改链接的去处、勾选框不留痕迹 | 内容标识加按顺序的 `a[href]` 和勾选框的状态（3.7、6.2 的清洗与内容投影，P4/A1 裁定后位于 `adapter/content`）；S3 加只改链接、只改勾选框的样例，必须算修改（9.1，P4 的任务 1）；8.2、4.17、§16 改写；I2 的 P4 断言；M5 打开图片时加 `image-component[src]`（13.2） |
| I5 | 选择器接到的 `issue-search` 要求 `q`；日历的"添加已有"要筛出没有截止日期的 | `q` 可选，空时给出最近更新的可见工作项（3.18、5.1）；列表和搜索都有 `target_date=none`、`start_date=none`（3.16、3.18）；`ExistingIssuesListModal` 在 P12 随日历改接（7.5，P12 的任务 9），选父工作项在 P14；I4 的页面版本含"添加已有"；7.1 的实例分配改为 P12 一处、P14 三处 |
| I6 | 每个加表的 Phase 让 M3 的删除故事断言不到新表 | e2e 的 `furnishIssueRows(api, token, projectId, assigneeId, labelId)`（`e2e/fixtures/furnish.ts`），每个加表的 Phase 扩充它并让 M3 的 P4、W2、W3 在删除之前调它（2 节、9.6、第 12 节约束 8）；`workspaceTables` 从目录读出；P9 起删除的断言轮询；M5、M6 的交接（13.2） |
| I7 | P10 的两段换型编译不过；M6、M7 的列表 store 没有能"行为不变"的类型 | (a) P10 预先拆成 P10a（9 个任务：先收窄、后换型，`TIssue` 一族在这里换掉）和 P10b（14 个任务）；列表的形状在 P10b、详情的类型在 P11（7.2 的"哪一段删哪一个类型"）；(b) P10a 删除 M6、M7 的列表 store、布局根、页面上的列表部分、`EIssuesStoreType` 的四个值和它们的分支（7.2、7.3、7.7）；可见的代价写进前端改动清单和 §16；M6、M7 照 Plane 的源码加回（13.2） |
| M1 | 预检有两个洞：加关联的另一端先被锁；注定 403 的写也先取锁 | 预检回答一切会被拒绝的（404、403），加关联先预检另一端的可见性、看不到的不锁（3.5、6.7、8.4、11.2）；交错 21 加"指向看不到的项目时不锁它"（9.3）；P1 的任务 7、P3 的任务 5、8 |
| M2 | 外键的 `FOR KEY SHARE` 挡住 `nerve users set-email` | 3.5 写明（SP1 的数字），M4 不改代码；M8 的交接（13.2） |
| M3 | SSRF 的拒绝表漏了特殊用途的网段；`error` 的细分让抓取成为端口探测的回显 | 6.8 写出 IANA 两张特殊用途登记表和本机的接口地址，每个网段一个测试；`metadata.error` 收成 `blocked`、`failed`（3.11、5.2）；8.1 改写，承认公网地址上的端口探测仍可做；4.17、8.8；P7 的任务 4、6 |
| M4 | 在项目里只是访客的工作区管理员的 `OwnOnly` 没有定义 | 3.4：按项目的角色，`OwnOnly` 为真，404 优先；矩阵加"管理员·访客"一列（9.2）；9.1、8.4；P2 的任务 1、11 |
| M5 | 关注在失去读权之后仍在，M7 的交接说反了 | 3.4 写明；M7 的交接：通知发出时按 `issue.read` 加访客规则重新判定（13.2） |
| M6 | 分组的请求带 `cursor` 而不带 `group_key` 没有定义 | 422（3.16、5.1）；9.3；P5 的任务 3 |
| M7 | 泳道的行总数没有返回，页面把格加起来会重复计算 | `subgroup_totals`（3.16、5.2）；3.22 的 `subgroupTotals`；9.3；P5 的任务 7、P12 的任务 2 |
| M8 | 动态引用的工作项在已删除的项目里、已物理清理时怎样显示没有定义 | 3.13：按调用者在被引用的工作项的项目里是否有效成员判定，不看 `deleted_at`；已清理的略去；`VisibleProjects` 加一个读成员关系历史的方法（3.3、6.5）；9.3 三个测试；P4 的任务 12 |
| M9 | 命令行的组合要 `issue.Provide` | P1–P8 命令行为 `project.NewCascade` 传 `issue.Provide` 的 `IssueCascade`，P9 起删除（6.3、6.6）；P1 的任务 4、P9 的任务 10 |
| M10 | 列表的读跨两个快照 | 一次读在一个只读的 `REPEATABLE READ` 事务里（3.4、3.16）；9.3；3.23 的 P5 一行；P5 的任务 5 |
| M11 | 草稿页没有筛选；相对日期存在，`getFilteredWorkItems` 要保留 | 3.17、7.3、P10b 的任务 9 去掉草稿页的筛选；3.22、7.3 保留 `getFilteredWorkItems` 和 `fromnow`（9.5，P10b 的任务 5） |
| M12 | 按编号取的"转成大写"错过 `İ` | 两边按同一个规则折叠（`foldIdentifier`：i、ı、I、İ 折成一种，并列时精确匹配优先），`api/testdata/identifier-fold.json`（3.7、3.18、6.2、9.1、9.3）；P2 的任务 5、P8 的任务 7 |
| M13 | 删除项目的连带先锁关联行、后删草稿，违背全局顺序 | 锁关联行的一步排到全局顺序里关联表的位置（3.5 的取锁表，`deleteIssue` 同样）；任务按全局顺序逐张表（3.6）；11.2 |
| M14 | 管理员不能删除别人的评论是一项没说出的产品决定 | 新的决策点 8（建议 B：项目管理员也能删除，不能改）；3.4、3.12、5.3、7.4、9.2、4.17；从第 10 节"不交负责人"的清单删去；P4 的任务 9、P15 的任务 3 |
| M15 | 规模目标是负责人一级的承诺 | 并入决策点 7，写明每个选项在目标上什么成立；3.16、§16 |
| M16 | `expectIssueDeleted` 不递归，断言不到评论的表情一类 | 沿外键图从 `issues` 递归（3.6、9.6）；P2 的任务 4 |
| M17 | I1 的分诊状态一格经接口准备不了 | 移到 Go 的集成测试，经仓储准备（第 12 节约束 1、9.3，P1 的任务 13）；I1 删去那一格 |
| M18 | I10 写的地址到不了页面服务器，`allow_cidrs` 放在哪里没说，"没有发出请求"在 e2e 里看不到 | I10 写 `127.0.0.1:<端口>/page`；`allow_cidrs` 在 `config.test.yaml`（3.11）；只断言 `error: blocked`，不连接由 Go 的表测试证明（2 节；P7 的任务 5、11） |
| M19 | 完成线点名后面的 Phase 才做的东西 | C5 只留详情一侧，时间线、表情、描述的历史移到 C6（9.7，P15）；I20 的页面版本移到 P14，评论自己建的在 P15；I12 改写为打开时读完全部页；S2 按身份打开详情在 P14 的任务 14，每个身份有预期（2 节、9.6）；C2、C3、C9 写进完成线（P12–P15）；13.1 的 C3 改为 P12、P14；访客建的工作项经降级准备（2 节、约束 1） |
| M20 | 时钟偏移的故事有几个缺口 | `clock.ts` 用自己的数据库和 nerve，偏移的 nerve 上重新登录，只认偏移的 nerve 启动之后完成的任务，对照的数据经偏移的 nerve 写下（3.20；2 节的 I16、I21；9.6；P6 的任务 10） |
| M21 | 评审敏感的内容仍混在一起 | 版本的保留移到 P9（与清理、删除的任务一起）；`link_count` 移到 P8；日期和 `bodyshape` 的生成器移到 P2（第 12 节约束 3、5） |
| M22 | `assignee_name` 的目录只有有效成员 | 目录由每个当过负责人的账户组成，经 `PublicProfiles`（3.3、3.16、6.5；P5 的任务 4） |

裁定留给本文选择的，第三稿的选择和理由（都在正文里写明）：
- **事件而不是端口**（I1）：决策点 3 建议 A，删除项目、工作区经 `project.deleted`、`workspace.deleted` 到达 `issue` 的订阅者（3.19）；选 B 时由 `IssueCascade.ScheduleRowDeletion` 投递同一个任务。第三稿写的"删除工作区是**一个**按 `workspace_id` 的任务"**在 17.3 改了**（聚焦复核 I1）：它会把更早删除的项目里还没写到的行写成工作区的时刻，现在是每个这次删除的项目一个任务，加没有项目的草稿一个（3.6）。
- **任务在 P9 落地，P1–P8 是同步的过渡写法**：M3 的删除故事从 P1 起就要工作项一侧的行被删除，任务要 P6 的事件和只投递的客户端；两种写法调用同一组语句（`delete_rows.go`），P9 只换调用者（3.6、第 12 节）。
- **删除工作区仍先锁它的项目、再读时刻**（I1 要求重新推出）：任务写的是请求的时刻，一个在那之后提交的工作项的写若不被挡住，它的行的时刻会倒退；这一步现在只挡 M3 的几步，代价小（3.5、决策点 7）。
- **I7 选预先拆开 P10，而不是把类型的删除推到最后一个使用方的 Phase**：最后的使用方在 P15，推迟会让 P10b–P14 的新 store 都先写在 Plane 的类型上、再换一次；"先收窄、后换型"让 P10a 的每个任务都能编译（7.2）。
- **日期和 `bodyshape` 的生成器放在 P2**（M21）：I1、I2 的日期断言是第一批要日期的故事，P2 的敏感项是权限，与请求体的检查无关；不另开一个早期任务。
- **`link_count` 放在 P8**（M21）：它改的是列表的 SQL 和游标，P8 本来有列表一侧的搜索。
- **`ExistingIssuesListModal` 在 P12 改接**（I5）：它的第一个调用方的页面（项目日历）在 P12；子工作项、关联的两个调用方的参数在同一个任务里换掉。
- **写闸门的默认值**：K = 2（一个持 N 在写，一个已在等 N）、`max_wait` 2 秒（照 `auth.password.max_wait` 的写法）、`max_queue` 64、`lock_timeout` 5 秒（3.5、11.11）。
- **`metadata.error` 收成 `blocked`、`failed` 两个值**（M3 的"保留枚举、收粗"）：页面只看有没有标题（3.11）。
- **编号的折叠在并列时精确匹配优先，否则 404**（M12）：只差 I 与 İ 的两个项目标识可以并存（M3 的唯一性是逐字的），折叠之后会撞在一起。
- **动态引用的可见性看成员关系的 `is_active`，不看 `deleted_at`**（M8）：删除项目时成员关系被软删除而 `is_active` 不变，离开、被移出时 `is_active` 为假（3.13）。
- **决策点 8 的 B 用已有的 `Grant.ProjectAdmin`**，不加新的规则（3.4）。
- **3.22 的重做的第 2 条判断**（应答覆盖了它修改之后该在的位置而它不在，就是修改之前）是对评审的方向的补全：只看应答里有没有它的行，会把"应答在修改之前、而它不在这一页"当作看不出。聚焦复核认为它对这个会话自己的修改成立、对第三方随后的修改不成立，17.3 给它加了一次 `getIssue`（3.22）。

### 17.3 第三稿的聚焦复核（2026-10-16）
复核结论：Ready with fixes（Critical 0、Important 2、Minor 9）。17.2 的 30 行（I1–I7、M1–M22 和"另外"一行）中 25 行完整，5 行部分：I1、I6（同一个原因：按工作区的任务，下表 I1）、M4（"管理员·访客"一列的两格和说明，下表 M1）、M13（两行的顺序，下表 M4）、M19（S2 对不是项目成员的人的预期，下表 M2）。复核核对过而成立的：窗口期间的读写、重新推出的删除工作区的取锁、写闸门与锁和连接池不成环、3.22 对第二稿九个反例的回答、任务放在 P9、每个 Phase 能编译、计划的算术。复核另做了 SP4（0.3）。控制者接受全部发现；第三稿照裁定再修订一次：

| 编号 | 问题 | 处理 |
|---|---|---|
| I1 | 按 `workspace_id` 的一个任务会把更早删除的项目里还没写到的行写成工作区的时刻和删除者；另一个项目上活的删除工作项也会把属于已删除项目的关联行写成自己的时刻；M3 的 W3 在 P9 时过时不过 | 每个删除的项目一个任务（`workspace.deleted` 的 `Changes` 带这次锁住、删除的项目），另加没有项目的草稿一个（3.6、3.19、3.3、6.3、6.5）；一条跨项目的关联行属于先删除的那一端：`deleteIssue` 和任务经 `ProjectGate.DeletedAt` 跳过属于别人的行（3.5，取锁表的两行）；交错 24 断言写下的是谁的删除，新加交错 25（先删项目、在它的任务做完之前删工作区）；3.5 删去"删除工作区的任务不需要它"；决策点 7 的 A 一行；§16；17.2 的选择一行记下这里改了；P9 的任务 9–11 和完成线 |
| I2 | 分批的语句是平方的，River 默认 1 分钟的期限没有写进设计（SP4：一个项目的 40 万行动态 53.9 秒，一个五个项目的工作区约 9 次超时、2.4 小时的退避） | 每张表按 `id` 的键集（`AND id > $after`），重试从第一个未删除的 `id` 开始（3.6）；worker 覆盖 `Timeout()`，45 秒的预算（`issue.delete_budget`）用完返回 `river.JobSnooze(0)`，不算一次尝试（`river@v0.47.0/error.go:31-40`），真的出错照 River 默认的重试（3.6、3.20、6.1、8.8）；3.21 的清理同样的键集；SP4 的数字写进 0.3 和决策点 7 的 A 一行；P9 的量测改成一条通过线：五个目标规模的项目的工作区，每个任务一次运行做完（9.3、P9 的任务 2–4、10、11 和完成线、§16） |
| M1 | "管理员·访客"一列的 `createIssue`、`listIssues?archived=true` 和说明与 3.4、`decide.go` 不合 | 两格改为 ✓（已归档的列表只有自己建的）；说明改为：别人建的 404，他自己建的照"项目管理员"一列（9.2） |
| M2 | S2 里不是项目成员的人对一个工作项的详情得到的是 403，不是 404（S2 的项目是公开的） | S2 对他预期"没有权限"、放行那一个 403，项目仍是公开的（2 节的 S2、9.6；P14 的任务 14 照 2 节） |
| M3 | 3.22 的重做留了四个缺口 | 重做时旧属只来自应答和 `beforeSnapshot`；`fetchPart` 的并入取应答的 `total`；应答回答了的组旧属按应答，第 2 条判断时另取 `getIssue(X)`，第三方的移动由那一份的规则发现；`boundary` 取 `max(boundary, 这一页的最后一行)`；自己的修改在路上的工作项读到新版本不标 `stale`（3.22）；五个各是 vitest 表的一行（3.22、9.5） |
| M4 | 删除工作项和任务两行的顺序把评论的表情排在表情之前，与全局顺序不合 | 两行都改为评论 → 表情 → 评论的表情（3.5 的取锁表） |
| M5 | 三处记账：9.3 的一句、`delete_rows.go` 的位置两处不一、P2 的访客那一格没有落点 | 9.3 改为"正在进行和闸门放进来的写（每个项目至多 K 个）"；语句在 `issue/adapter/postgres/delete_rows.go`（sqlc），编排在 `issue/app/delete_project_rows.go`，经仓储的端口调（3.6、6.2、约束 8、P1 的任务 5）；P2 的端到端不再有访客那一格，P2 由矩阵（Go）核对，端到端随 I20 的接口版本在 P5（P2 的任务 13） |
| M6 | `SET LOCAL lock_timeout` 碰上不许模块在 `gen` 之外发语句的守卫；`55P03` 在哪里换成 503 没说 | `platform/postgres` 的 `TxManager` 加选项 `LockTimeout(d)`（`BEGIN` 之后由平台发 `SET LOCAL`）和 `LockNotAvailable(err)`；`bootstrap/gate.go` 给 `issue` 的写用例这个 `TxManager`，把 `55P03` 换成 503，挨着闸门的 503（3.5、6.1、6.6、6.7、9.1、11.11、P1 的任务 8）；列表的只读快照同样是一个选项 `ReadOnlySnapshot()`（3.16、P5 的任务 5） |
| M7 | P10a 让旧的 service 用生成的 `Issue` 去类型 Plane 的响应，没有写明；`WorkspaceDraftIssuesFilter` 的删除没有任务 | 7.2 写明这是有界的过渡：点名四个旧 service，列表一侧在 P10b、其余在 P11 换掉，P11 的完成线核对；界限之外只有 M7 的 `inbox-issue.service.ts`，写进 M7 的交接（13.2）；P10a 的任务 6、P10b 的任务 1、9（删除 `WorkspaceDraftIssuesFilter`） |
| M8 | 决策点 8 的 B 有两个代价没写：删除别人的评论在时间线上不留痕迹；只是项目访客的工作区管理员也能删别人在他的工作项上的评论 | 两个代价写进 B 一行；删除不加一条动态（3.13 定下评论不写动态，加一种动态要动态的格式和页面一起定下），痕迹是 `updated_by_id` 和 `issue.comment_deleted` 事件，要在时间线上显示留给负责人（决策点 8） |
| M9 | "拒绝本机的接口地址"在 NAT 之下说过了头 | 改为"启动时本机网卡上的地址"；公网地址不在网卡上的部署靠网络层的出站规则（6.8、8.1、4.17、8.8 的 P7 一行） |

裁定留给本文选择的，这次修订的选择和理由：
- **怎样知道关联行的另一端已删除**（I1）：在 `deleteIssue` 锁住关联行之后经 `ProjectGate.DeletedAt` 不加锁地读，不用约定七去锁另一端的项目。约定七要在锁本项目之前知道全部另一端、一次取齐，给每个删除工作项加锁；这里只需要"那一刻另一端删了没有"，读之后才提交的删除排在这次删除工作项之后，与交错 11 的同步写法同一个结果（3.5）。同一个读也让任务跳过属于更早的删除的关联行。
- **没有项目的草稿是工作区的一个任务**（`scope = workspace_drafts`），不并进某个项目的任务：它们不属于任何项目。
- **预算 45 秒、`Timeout()` 取预算加 5 秒**：预算在每批之前看，SP4 的一批约 3 ms，任务在期限之前就停下返回 `JobSnooze(0)`，不会被期限打断在半批上（3.6）。
- **`lock_timeout` 走 `TxManager` 的选项，不走 sqlc 的 `set_config`**（M6）：选项让给 `issue` 的写用例的 `TxManager` 一律带上，`set_config` 要每个写用例自己记得先调；`55P03` 的转换在 `bootstrap/gate.go`，挨着闸门的 503。列表的只读快照用同一种选项。
- **删除的语句和编排分开放**（M5）：SQL 在 `issue/adapter/postgres/delete_rows.go`，编排（表的顺序、键集、关联行的跳过、预算）在 `issue/app/delete_project_rows.go`，同步的过渡和任务都调这一份编排。
- **P2 的访客那一格移到 P5**（M5）：端到端随 I20 的接口版本，P2 由矩阵核对；不另在 I20 之前加一个故事的增长。
- **删除别人的评论不加动态**（M8）：理由在决策点 8；要在时间线上留痕迹由负责人另定（负责人 2026-10-17 按建议裁定 B，不加，17.4）。
- **S2 保留公开的项目**（M2）：对不是项目成员的人预期 403 的"没有权限"，它正是 3.5 的预检给他的回答，S2 由此也核对了这一格。

### 17.4 负责人的裁定（2026-10-17）
第三稿 `720428ea` 连同三轮评审的落实交负责人。负责人的回答是"全部按建议"：
- 决策点 1 选 B：工作项和它下面的写只锁项目行，不锁工作区行；删除工作区先按 `id` 锁住它的项目、再读时刻。B 的两个代价和连接池的代价（写闸门，11.11）照 3.5 写明。
- 决策点 2 选 A：一个 M4，17 个 Phase 加收尾；控制者遇到超过上限的 Phase 直接拆成 P<k>a、P<k>b 继续，记进第 15、16 节。
- 决策点 3 选 A：现在引入领域事件；第一个异步的订阅者是链接的抓取，P9 起还有删除项目、工作区之后工作项一侧的删除任务；只投递的 River 客户端在模块之前建出，启动时核对每种会投递的任务都有 worker。M2 交接第 1 节原样转给 M5。
- 决策点 4 选 A：`actor_id` 为空表示 nerve 自己。
- 决策点 5 选 A：不做 `issue_versions`，只有描述的历史。
- 决策点 6 选 A：不做评论的回复。
- 决策点 7 选 A：请求只做 M3 的连带，工作项一侧的行由后台任务按项目分批软删除，写下请求的时刻和删除者；v0 的规模目标是一个项目至多约 2 万个未删除的工作项。
- 决策点 8 选 B：项目管理员也能删除（不能改）别人的评论；删除不写动态，时间线上不显示，痕迹是 `updated_by_id` 和 `issue.comment_deleted` 事件。
- 第 11 节的十一个架构问题全部批准。

正文各节本来就按建议写定，这次只把"待负责人裁定""待负责人批准"的标记改为裁定、批准的记录（开头的状态、第 10、11 节、3.23、第 14、15 节）；3.23 的每一行内容随之确定。总体设计 9.4 中 M4 的状态随本文的合并改为"进行中"。
