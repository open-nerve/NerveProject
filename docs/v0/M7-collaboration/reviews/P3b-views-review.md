# M7 / P3b：navigation 模块与视图（阶段记录）

> 状态：已完成，合入 main（noff `1168fd25`），§9 为审查与集成的记录。spec 与 plan 见 [P3b spec](../specs/P3b-views.md)、[P3b plan](../plans/P3b-views.md)，预检 [re-90](../../../reviews/re-90-M7P3b预检.md)。基线 main `17e0b1ac`，分支 `codex/m7-p3b`。台账 `.superpowers/sdd/P3b-views/progress.md`（ignored，随阶段归档）。

## 1. 交付

- 新模块 `navigation`（照 `notification` 的形状）与表 `issue_views`（迁移 `00052`；`project_id` 为空即工作区视图；`filters` 的 CHECK 按种类分两种键集）。
- 接口 7 个：`listProjectViews`、`createProjectView`、`listWorkspaceViews`、`createWorkspaceView`、`getView`、`updateView`、`deleteView`；动作 8 个（`view.*` 项目级、`workspace_view.*` 工作区级）；新码 `navigation.view_not_found`（404）、`navigation.views_disabled`（409），网页有文案。
- 规则：建、改、删要角色；改只给所有者，删是所有者或项目管理员 / 工作区管理员；只看自己的人看不到别人的视图（列表里没有，取、改、删 404）；工作区访客对已存工作区视图 403；项目视图的写要 `issue_views_view` 开着，读照常；已归档项目上照常。
- `/views/{view_id}`：不加锁读出视图、按种类选动作；看不到、不存在、已删、只看自己藏起都答同一个 `navigation.view_not_found`，逐值相同。
- 写只有一条路 `Locks.run`：预检（404 → 403 → 不是所有者 / 管理员 403 → 422）→ 工作区 S → 项目 S（项目视图）→ 视图行 `FOR UPDATE` → 锁下重读重判 → 409 → 无改动不写。
- 删除图：`project.deleted`、`workspace.deleted` 的订阅者在原事务里投 `navigation.delete_rows`，任务分批软删，时刻与删除者取事件的；工作区那一步只删工作区自己的视图，项目视图由每个列出的项目的任务删。60 天清理接上（清理目录 34 → 35）。
- 共用层：`LogoProps`（契约三个 schema、Go 类型与 `Check`、`CheckText`）从 project 纯搬移到 common / shared（R5）；issue 交出 `WorkspaceListSettings`（工作区视图的已存词，11 键）。

## 2. Task 与提交

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `9dbe9119` | spec、plan、预检 re-90；P3a 记录 §9；设计 §3.7（R1）、§14 |
| T2 | `ea92002a` | `LogoProps` 的纯搬移（R5） |
| T3 | `4c39dfde` | issue 的 `WorkspaceListSettings` |
| T4 | `9f5901e6` | 迁移、模块骨架、8 个动作与规则、清理；R3、R4；红窗口 (a)、(b) 起 |
| T5 | `c6c7d8c0` | 领域：视图、内容与改动、名字与图标、所有者规则、错误 |
| T6 | `c7017160` | 存储：两种列表、取、锁、建、改、软删 |
| T7 | `7fbadb37` | 删除图、config、接线、R8；红窗口 (a) 关闭 |
| T8a | `f5713fbd` | 端口、读与建 |
| T8b | `b3c1aeea` | 改与删 |
| T9 | `ff1cdc63` | 契约、处理器、接线与网页的码；红窗口 (c) 起 |
| T10a | `530ca390` | 权限矩阵、项目写锁表、工作区视图的锁序；红窗口 (c) 关闭 |
| T10b | `896880ff` | e2e 删除夹具；红窗口 (b) 关闭；检查点 1 |
| T11 | `3835d00d` | 组合测试 H12（含 R6）、H14（含 H13 的往返） |
| T12 | `86ced65b` | 交错 |
| T13 | `39b5d501` | e2e 接口故事 H12、H14；检查点 2 |
| T14 | 本提交 | 文档与交接 |

## 3. 局部裁定（实施中）

- **词的校验在 app 层**（T5）：领域不声明端口（全仓的先例）；用例按视图种类选 issue 的 `ListSettings` / `WorkspaceListSettings`，一个 422 里领域问题在前、词在后（T8a、T8b 钉住）。一个 `Filters` 类型装契约的 13 键并集，按种类的拒绝由 issue 的两种词表与列的 CHECK 负责。
- **表不设时刻与三个 jsonb 列的默认值**（T4）：漏给即 `NOT NULL` 报错，不悄悄取数据库时钟；名字的 CHECK 只 `<> ''`，去空白与 255 字归领域。
- **R3**：Go 的删除守卫给 navigation 一个谓词，`issue_` 前缀不再把 `issue_views` 认作 issue 的表；e2e 的 `asynchronousDeletionTable` 只分异步与立即，`issue_views` 作为 navigation 的一项列出。
- **工作区删除时的项目视图**（T7）：`workspace.deleted` 只有一条、把锁住的项目列在 `Changes` 里，订阅者照 issue 为每个列出的项目投一条项目范围的任务，另投一条工作区范围的只删 `project_id IS NULL`。
- **允许清单外的两处跟随**（T7）：`server/configs/embed_test.go`（内置配置整体比较）、`bootstrap/app_test.go` 的 `testConfig`（不经 Load，缺这一行任务批量为 0 会一直 snooze），各只加 navigation 的一行。
- **开关只在锁下判**（T8a，照 spec §2.5）：预检时关、锁下开照样建。工作区视图的建在锁下按 slug 重读。视图的写用普通事务（不进写闸门，没有 `lock_timeout`，与项目的写同）。
- **`LockView` 拆成一方法的端口 `ViewLocker`**；`decide` 是预检与锁下共用的唯一判定（T8b）；无改动在锁下 409 之后、拿重读的行比。
- **400 与 422 的分界**（T9）：13 键之外的键（如 `sub_issue`、`archived`）是结构 400；另一种视图的键（`cycle_id`、`module_id`、`project_id`）是值 422。
- **矩阵**（T10a）：视图经 navigation 的存储种（访客所有的接口建不出）；只开 acme 公开项目的开关，私有项目的格顺带钉住“403 先于 409”与“读不看开关”；X 三列两级都有，指哪种视图都答同一个 404。
- **列表次序的同值打破**（T6、T11）：部分索引自带 `id` 次序，去掉 `id DESC` 的变异在小表上存活；存储测试关掉索引扫描再读，组合测试先 `ANALYZE`，各留对照（硬规则 §5 新增一条）。
- **锁下的删除重判有两层**（T12）：容器的 `FOR SHARE` 读不到，`Authorize` 也看已删；只去一层是等价变异，变异两层一起去，留对照（硬规则 §5 新增一条）。
- **e2e**（T13）：开关由项目管理员打开（`project.update` 只给管理员）；只看自己的访客经接口先以成员建、再降为访客。

## 4. 向负责人说明

- 没有需要负责人裁定的事项。预检 re-90 的 10 个重要、12 个次要全部在 T1 采纳（预检文末“处理”一节）。
- 设计与原样的不同，按设计 §3.10 的表定：工作区访客对已存的工作区视图一律 403（R1；设计 §3.7 有歧义的一句已改写）。
- `bootstrap/app.go` 397 行，下一个往里加接线的 Phase（P3c）要先拆。三个旧测试文件略超约 400 行（`permission_matrix_test.go` 451、`permission_matrix_seeded_test.go` 428、`project_write_locks_test.go` 426），留给整分支审查。

## 5. 本地门禁与 CI

| 时点 | 本地七项 | 分支 CI |
|---|---|---|
| 检查点 1（`896880ff`） | `cp1`、`cp1r` 两次第 0 路都以 M6-M8-05 失败（Docker 映射端口 “invalid port”，测试代码开跑之前；backend 日志同时刻 port forward 失败），第 0 路以分支 CI 为准（负责人 2026-11-18 的规则）；其余以 `cp1c` 续跑全过（e2e 489 passed） | 44777446942 全绿 |
| 检查点 2（`39b5d501`） | 七项一次过（e2e 491 passed） | 44793862354：第 1 次只有 e2e (1) 失败——测试开跑之前 globalSetup 起 S3 夹具容器，runner 的 Docker 答 500 “address already in use”（容器要 `HostPort` 0；P3b 没碰 S3 夹具，同一 runner 装 Playwright 浏览器用了 19 分钟），环境原因，只重跑失败的作业一次；第 2 次全绿 |

## 6. 测试时间与两路（秒）

| 时点 | 本地 bootstrap 第 0 / 1 路 | CI server (0) Test | CI server (1) Test |
|---|---|---:|---:|
| main `17e0b1ac`（44743168952） | | 618（bootstrap 456.4） | 766（bootstrap 746.7） |
| 检查点 1（44777446942） | 421.6 / 651.0 | 626（bootstrap 452.7） | 760（bootstrap 740.1） |
| 检查点 2（44793862354） | 432.9 / 667.9 | 855（bootstrap 578.8，慢的 runner，见下） | 777（bootstrap 757.0） |
| T14 `6f062a61`（44809593146，代码同检查点 2） | | 625 | 770 |

- 检查点 2 的第 0 路慢在 runner（同一份代码在 `6f062a61` 上 625 秒，比基线 +7）：P3b 没碰的非 bootstrap 包合计 605.5 → 965.0 秒（1.59 倍，如 `issue/adapter/http` 108.4 → 206.2），bootstrap 只 1.28 倍；第 1 路 757.0 秒约为 `go test` 期限的 63%。

- 新的 bootstrap 顶层名（桶 = `cksum % 100`，0–45 第 0 路）：

| 名字 | 桶 | 路 | 本机 |
|---|---:|---:|---|
| `TestProjectViewsThroughTheApp` | 0 | 0 | 3.6–5.6 |
| `TestViewWritesInterleave` | 6 | 0 | 2.1–5.6 |
| `TestEachWriteOnAWorkspaceViewSharesItsWorkspaceFirst` | 38 | 0 | 1.4–3.2 |
| `TestWorkspaceViewsThroughTheApp` | 42 | 0 | 1.0–2.1 |
| `TestNewAppNavigationJobCatalog` | 68 | 1 | 0.0 |
| `TestMatrixViolationsCatchesAViewOfAnotherKind` | 70 | 1 | 0.0 |

- 被加重的已有测试：`TestPermissionMatrix`（桶 46，第 1 路）3505 → 3632 格，本机单跑 +1.2 秒（约 3.6%，CI 上约 +3 秒）；删除守卫、清理测试各多等一个空任务或一张表，毫秒到一秒级。
- 两个组合测试与交错测试特意取落在第 0 路的名字（第 1 路的包约在期限的六成）。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- 删除订阅者里校验 `project.deleted` / `workspace.deleted` 载荷的 `deletedProjects` 已有四份逐字相同（issue、planning、notification、navigation），各模块的 `app/purge.go` 有七份除包注释外逐字相同（identity、issue、notification、planning、project、workspace、navigation；asset 另是一种）。P3b 照已有形状加了各自的又一份；是否抽到 shared 交 M8 评估（整分支审查 m5）。

## 8. 文档与交接（T14）

- plane-diff：表 `issue_views` 一行（二、保留表的改动）、“项目视图与工作区视图（M7/P3b）”一行（四、行为差异）。
- 交接的处理结果：M2 §1（视图的部分；剩收藏与最近访问的列表，保持 open）、M2 §5（`issue_views` 的外键去向；保持 open）、M4 M7-03（视图的接口一支；页面与旧方法随 P6a/P6b）、M6-M7-03（项目视图有迭代、模块词）、M6-M7-04（只到接口与两个码的文案）、M3 §1（工作区视图的筛选列）、M5-M7-03（视图的说明是纯文本）。M6-M7-04、M6-M7-07 与 M3 交接 §2–§4 的视图项都还在 P6a/P6b。
- 硬规则 §5：列表次序的同值打破在小表上的测法；两层重判的等价变异。
- 设计 §3.7 与实现对齐（键数、404、PATCH、已归档项目、收藏随 P3c）；§4 的表补 `issue_views` 的审计列；§14 的 P3b 一行。

## 9. 整分支审查与集成

- 检查点 2（`39b5d501`）：本地七项一次过；分支 CI 44793862354 第 1 次只有 e2e (1) 因 runner 的 Docker 端口失败，重跑一次全绿（§5、§6）。
- 整分支审查（fable，只读，P3b 对 main `17e0b1ac` 的差）：With fixes，I 2、m 5，产品代码不用改。I1 plane-diff 两行把 Plane 的视图删除写成物理删除（原样经 `SoftDeleteModel` 软删；工作区视图只连带删收藏）、I2 第 0 路的增量要按 spec §4 实测、m1 T8b 的提交号、m5 交 M8 的重复（§7）——主控改文档（`b44183be`）；m2 e2e 删除目录里不起作用的排除、m3 视图名 255 字的宽度钉在真实数据库上——主控改测试（`ac121cce`：列改 254、256 的两个变异都被杀死）；m4 文件大小不必现在拆，`bootstrap/app.go` 先拆写进 P3c 的约束。
- 限定复审（`b44183be`、`ac121cce`）：Ready to merge，发现 0；三条不阻塞的说明不改：N1 当时引用的 CI 的 e2e 两路未完（§6 只引 server 的 Test 步）、N2 spec 里 e2e 谓词的措辞稍旧（最终样子见 §3）、N3 一行注释 97 列。
- 第 0 路的实测（spec §4：每路 ≤ 90 秒）：同一份代码在 `6f062a61` 上 server (0) Test 625 秒（基线 618，+7）、server (1) 770 秒（+4）。检查点 2 的 855 秒与最终分支 CI 的 840 秒都是慢的 runner：P3b 没碰的非 bootstrap 包合计 1.57–1.59 倍。
- 最终七项 fin（`ac121cce`）全过：bootstrap 427.3 / 642.0 秒，e2e 491 passed；最终分支 CI 44815731936 attempt 1 全绿：server (0) 作业 1021 / Test 840 / bootstrap 574.7，server (1) 934 / 902 / 877.5（慢的 runner，期限的 73%，未到八成）。
- `--no-ff` 合入 main `1168fd25`（树与分支相同）；main CI 44820762025 attempt 1 全绿：server (0) 作业 790 / Test 650 / bootstrap 469.1，server (1) 878 / 850 / 827.9（第 1 路的包在 `go test` 20 分钟期限的 69%）。
- 归档 `.superpowers/sdd/M7-P3b-integration-20261121`（2828 个文件、unknown 0，清理后复核一致）；P3b 的 worktree、本地与远端分支已删除（先核实已含在 main）。
