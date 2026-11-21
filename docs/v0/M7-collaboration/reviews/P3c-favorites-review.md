# M7 / P3c：收藏与三种归档事件（阶段记录）

> 状态：已完成，合入 main（noff `9d26b1c3`），main CI 44947828240 全绿（§9）。spec 与 plan 见 [P3c spec](../specs/P3c-favorites.md)、[P3c plan](../plans/P3c-favorites.md)，预检 [re-91](../../../reviews/re-91-M7P3c预检.md)。基线 main `1168fd25`，分支 `codex/m7-p3c`。台账 `.superpowers/sdd/P3c-favorites/progress.md`（ignored，随阶段归档）。

## 1. 交付

- 拆 `bootstrap/app.go`（397 行）：`app.go`、`app_ports.go`、`app_modules.go`、`app_serve.go` 与 `events.go` 的订阅者清单，保持行为的提取（R2 的六条次序、失败清理一一对应）。
- 三种归档事件：`planning.cycle_archived`、`planning.module_archived`、`project.archived`，在归档的事务里、写下 `archived_at` 之后发布，时刻就是写下的那一刻；迭代、模块再归档不写也不发，项目再归档写新时刻、照发；恢复不发。
- 表 `user_favorites`（迁移 `00053`）：五种类型（`view` 兼指两种视图），文件夹 ⇔ 列的搭配等 CHECK，两条部分唯一都带 `deleted_at IS NULL`。
- 接口 4 个：`listFavorites`、`createFavorite`（新建 201、已有 200）、`updateFavorite`、`deleteFavorite`；动作 2 个（`favorite.read`、`favorite.write`，工作区级，管理员、成员）；新码 `navigation.favorite_not_found`（404）、`navigation.favorite_folder_name_taken`（409），网页有文案。
- 规则：目标须存在、在这个工作区、此刻看得到、未归档，不合格一律同一个 422；`project_id` 由目标推出；`/favorites/{favorite_id}` 上的 404 一个码，含目标已看不到；次序按人；文件夹一层、名字去空白后不分大小写唯一；全部软删，删文件夹连它里面的；开关不看；读时过滤看不到、已归档、已删除的目标。
- 连带：迭代、模块的删除与归档、项目归档在发布者的事务里软删**所有人**的收藏（订阅者 `navigation_favorite_cascades`，按类型校验）；`deleteView` 同一时刻、同一写者带走它的收藏；项目、工作区删除经 `navigation.delete_rows`（任务改为按表的步骤：视图 → 收藏，工作区那一步先文件夹、后条目）；60 天清理接上（清理目录 35 → 36）。
- 锁：`FavoriteLocks`（建、改、删一个 run）：工作区 S → 项目 S → 视图 S → 父文件夹 S → 收藏行；收藏行的全序是文件夹先于条目、同类按 `id`。

## 2. Task 与提交

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `3fb8fc53` | spec、plan、预检 re-91；P3b 记录 §9；设计 §11、§12、§14 |
| — | `b4a792ec` | spec §2.3 目标的部分唯一只带 `deleted_at IS NULL`（T3a 实测 42P10） |
| T2 | `dab60dd0` | 拆 `bootstrap/app.go`；检查点 0 |
| T3a | `07150593` | 迁移、schema 测试、两个动作、清理一步；红窗口 (a)、(b) 起 |
| T3b | `7d8d6215` | 删除任务的收藏一步、Go 删除守卫的种子；红窗口 (a) 关闭；Go 两路中间检查 |
| T4 | `f6e3dcd9`、`4cce4e57` | 领域；存储 |
| T5 | `8c740444`、`b580e1da` | 端口；`FavoriteLocks`、列表与建 |
| T6 | `5b634fa4` | 改与删 |
| T7 | `193ac983` | 契约、处理器、接线与网页的码；红窗口 (c) 起 |
| T8a | `54cf7eac` | 三种归档事件 |
| T8b | `6d5687f9` | 连带、`deleteView` 的连带、R12、R13 |
| T9 | `b0ab7465` | 权限矩阵、锁序测试；红窗口 (c) 关闭 |
| T10 | `6fd1da8c` | e2e 删除夹具；红窗口 (b) 关闭；检查点 1 |
| T11 | `68eab95b` | 组合测试 H15（含 R14）、H16 |
| T12 | `c894e4af` | 交错 |
| T13 | `f962f3a0` | e2e 接口故事 H15、H16；检查点 2 |
| T14 | `2fafe043` | 文档与交接 |
| 修复轮 | `0de46327`、`5d0a6d45`、`1259cf8a` | 建与删的竞态（§9）、连带的类型由表推出、设计的一个空格 |

## 3. 局部裁定（实施中）

- **`app.go` 的拆分**（T2）：一个承载局部变量的结构 `wiring`（约 40 个字段，只活在构造期），步骤函数 `provide`、`dispatchEvents`、`share`、`modules`、`schedule`、`route`；`a.close()` 只在 `newAppWithFactories` 里、紧跟失败的那一步。
- **目标的部分唯一只带 `deleted_at IS NULL`**（T3a）：多写 `entity_identifier IS NOT NULL` 会让 `ON CONFLICT … WHERE deleted_at IS NULL` 推断不到索引（42P10）；spec 随之改（`b4a792ec`），硬规则 §5 新增一条。
- **清理守卫的盲点**（T3a）：外键 `CASCADE` 由表的主人执行，父先于子被清时子行的物理删除会被运行角色的清理守卫记成“运行角色之外的 DELETE”；夹具让收藏早于它的文件夹（逐行清），不加豁免，`TestPurgeUserFavorites` 两种次序都钉住；硬规则 §5 新增一条。
- **删除任务工作区那一步分两遍**（T3b）：单一 `ORDER BY (类型, id)` 会让按最大 `id` 前进的游标跳过未删的条目；“条目先于文件夹”在库里看不出（每批一个事务、只一种行），由单测钉 spec §2.8 的次序。
- **父规则只在存储的 `ShareFolder` 条件里**（T4）：找不到即父不合格，锁到的必是文件夹行，守住全序。
- **`NavigationFacts` 不加 `Archived`、项目批量不带标识**（T5）：锁下的归档由同一批量读与同一谓词定；标识随 P3d。plan T5 的那句按此读。
- **改只在给出非空 `parent_id` 时锁新的父**；视图收藏的改删也取视图 S（T6）。
- **`createFavorite` 的 `parent_id` 只收 uuid**（T7）：`null` 答 400，与改的 `null`（移出文件夹）分开。
- **项目归档的时钟读一次**（T8a）：锁后一次读数写进 `archived_at` 并发布；`LockCaller` 保留自己更早的读数（凭证锁只判有效、不写，同 `createProject`）。`project.archived` 用命名常量，注释写明与 409 码分属两处。
- **R12、R13**（T8b）：Go 在归档之后重种（重种撞部分唯一即证明连带接上）；e2e 的 `removedBefore` 还断言没有随项目删的行；R13 以缺池的 panic 判“到了写”。
- **矩阵的 `prepareMatrix`** 实际在 `permission_matrix_test.go`（plan 与 re-91 写成 `permission_matrix_seed_test.go`），一次调用使它 451 → 452 行（T9）。
- **交错的做法**（T12）：夹具扣住两个写都要的行、两个写都经真实的 app（planning 没有带 app 发布者的路由构造器；只有两个写同时在途才能在真实库上证明全序——“改先锁本行再锁父”的变异在真实库上死锁，约 1 秒后答 500）。
- **畸形归档的组合证据只做项目**（T11）：迭代、模块留在单元层（T8b 73 格、R13、T8a）。

## 4. 向负责人说明

- 没有需要负责人裁定的事项。预检 re-91 的 6 个重要、14 个次要全部在 T1 采纳（re-91 §6）。
- 实施前把 P3c 再划为 P3c（收藏）与 P3d（最近访问）（R1）：不拆约 25 个 Task。
- 实施中记下的罕见竞态（T4：同一个人同时删与重新收藏同一目标，建的 `ON CONFLICT DO NOTHING` 遇上只被删除锁住、还没改写的行不等待，跟着的读读不到 → 500）经整分支审查确认，在修复轮按根因修掉（§9），不交出去。
- 文件大小：`bs/permission_matrix_test.go` 452 行（一次调用，见 §3）、`e2e/stories/workspace/w3-workspace-settings.spec.ts` 404 行（两个导入与一行注释）；`e2e/fixtures/assert/project.ts` 在 T10 由 413 回到 352（`expectProjectDeleted` 纯搬到 `project-deletion.ts`）。

## 5. 本地门禁与 CI

| 时点 | 本地七项 | 分支 CI |
|---|---|---|
| 检查点 0（`dab60dd0`） | 六项过；e2e 490 passed 1 failed（F11 page 的 `net::ERR_ABORTED`，已知 M6-M8-07），只重跑这个文件一次 7 passed | 44905363500 全绿 |
| T3b 之后（`7d8d6215`） | Go 两路过（中间检查，e2e 在窗口 (b) 里） | — |
| 检查点 1（`6fd1da8c`） | 七项一次过（e2e 491 passed） | 44929502521 全绿 |
| 检查点 2（`f962f3a0`） | 七项一次过（e2e 493 passed） | 44938594594 全绿 |
| 合并前的终检（`d2511e56`，`final.sh fin`） | 七项一次过（e2e 493 passed；test-web 全部命中缓存，16/16） | 44945103629 全绿（最终分支 CI） |
| main `9d26b1c3`（合入之后） | — | 44947828240 全绿 |

## 6. 测试时间与两路（秒）

| 时点 | 本地 bootstrap 第 0 / 1 路 | CI server (0) Test | CI server (1) Test |
|---|---|---:|---:|
| main `1168fd25`（44820762025） | | 650（bootstrap 469.1） | 850（bootstrap 827.9） |
| 检查点 0（44905363500） | 422.5 / 637.5 | 820（bootstrap 548.5，慢的 runner） | 766（bootstrap 745.2） |
| 检查点 1（44929502521） | 417.3 / 662.2 | 835（bootstrap 555.2，慢的 runner） | 933（bootstrap 905.8） |
| 检查点 2（44938594594） | 453.9 / 649.0 | 723（bootstrap 496.1） | 753（bootstrap 733.0） |
| 终检（44945103629） | 453.2 / 655.4 | 650（bootstrap 465.5） | 967（bootstrap 938.8，期限的 78.2%） |
| main `9d26b1c3`（44947828240） | | 662（bootstrap 475.1） | 773（bootstrap 752.5，62.7%） |

- 新的 bootstrap 顶层名（桶 = `cksum % 100`，0–45 第 0 路）：

| 名字 | 桶 | 路 | 本机 |
|---|---:|---:|---|
| `TestFavoriteWritesInterleaveWithArchives` | 2 | 0 | 4.1–6.9 |
| `TestFavoritesOfEveryKindThroughTheApp` | 6 | 0 | 1.2–4.4 |
| `TestFavoritesFollowTheirTargets` | 13 | 0 | 3.9–5.9 |
| `TestFavoriteWritesTakeTheirTargetsLocks` | 37 | 0 | 2.8–4.3 |

- 被加重的已有测试：`TestPermissionMatrix`（桶 46，第 1 路）3631 → 3673 格；清理与删除守卫各多一张表（本机第 1 路从检查点 0 到 1 约 +25 秒）。
- 第 0 路的 runner 在检查点 0、1 都慢：P3c 没碰的非 bootstrap 包合计 492.8（main）→ 681.9 → 687.5（1.39 倍）。第 1 路在 CI 上 61–75.5% 摆动（检查点 1 的 905.8 秒是期限的 75.5%，未到八成；检查点 2 回到 733.0 秒、61.1%）；第 1 路只有 bootstrap，量不出 runner。检查点 2 第 0 路的 runner 也慢 1.21 倍（598.1 / 492.8），bootstrap 496.1。
- 终检与 main 是同一份代码（`9d26b1c3` 的第二个父就是 `d2511e56`），第 1 路的 bootstrap 却是 938.8 秒（78.2%，至今最高，未到八成）对 752.5 秒（62.7%），差 186 秒在 runner；本机第 1 路稳定在 649–662 秒（终检 655.4）。所以 CI 第 1 路的单次数字不足以判断增量：P3d 以本机检查点 0 的第 1 路作基线，多出 30 秒以上才停，CI 过 960 秒只报告（P3d spec §6）。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- 无新增。

## 8. 文档与交接（T14）

- plane-diff：表 `user_favorites` 一行（二、保留表的改动）、“收藏（M7/P3c）”一行（四、行为差异）。
- 交接的处理结果：M6-M7-02（迭代、模块的收藏与归档连带：后端完成，模块归档 Plane 同样移除——答“是”；页面随 P7）、M3 §1.4（项目收藏与归档项目时的处理）、M1-P3-trim-platform（契约的 `FavoriteKind` 五值）、M1-P2-trim-content（收藏没有 `page`；最近访问随 P3d）、M2 §1（收藏的列表）、M2 §5（`user_favorites` 的外键去向）。都保持 open（页面随 P7，最近访问随 P3d）。
- 硬规则 §5 三条：部分唯一上 `ON CONFLICT` 的推断；同表父子两类行的加锁全序；清理守卫看不见外键 CASCADE 的删除。
- 设计 §3.8 的实现指针、§4.2 的 `user_favorites` 一行、§4.3 的清理计数进度、§5 的收藏一行（201/200、`{favorite_id}`）、§14 的 P3c 一行；README 的删除一段与链接预览的指向；三个测试注释（`project_access_test.go`、`access_readable_projects_test.go`、`access_decide_many_test.go`）里 Authorizer 与 Readable 的构造处由 `app.go` 改指 `app_ports.go`。

## 9. 整分支审查与集成

- **整分支审查**（fable，只读，`1168fd25..2fafe043`）：没有严重、没有重要。次要三条：m1 T4 的竞态没有交出去；m2 连带订阅者的事件类型在 `favorite_cascades.go` 里写了两遍（表的键与 `Types` 字面量）；m3 设计 §3.8 新加的一句前多一个空格。T4 竞态的判断：属实、窄、不坏数据、不挡合并，可在修复轮以一次 `FOR SHARE` 的读修掉。
- **修复轮**（主控裁定：按根因修，不留已知的 500）：
  - `0de46327`：建一个目标的收藏在父文件夹的 S 之后、写之前，对本人对这个目标的未删收藏取 `FOR SHARE`（`ShareFavoriteOfTarget`，全序里条目行在文件夹行之后）；找到即 200 原样、不读时钟、不写，它的删除等建结束；没有（含等到的删除已提交）就插入。`ON CONFLICT` 的回读留给同一个人同时两次建；剩下的只有同一个人同一刻建、建、删三者交叠。spec §2.5、§2.8、§4 随之改。测试：存储一条、用例两格、锁序测试新的一段（16 个写都断言等待时没有写任何表）、交错新的一组 4 格（删与文件夹删 × 两种次序：删先则 201 且只剩新的一条；建先则 200 与原 201 逐字节相同、删 204、不留活的；从不 500）。变异 3 个点名的 + 6 个皆杀。
  - `5d0a6d45`：`Types` 由 `cascades` 的键排序得出；加第六个键的变异使 `TestFavoriteCascades` 与 `TestPlanningEventDirectory` 都失败。
  - `1259cf8a`：设计 §3.8 去掉空格。
- 以下在 P3d 的 T1 补记，数字取自归档台账的末行：
  - **限定复审**（fable，`2fafe043..49aa9fb2`）：没有严重、没有重要，Ready to merge；三条次要（两段注释重排、spec §2.5 的措辞、`shareExisting` 记下被拒的建也取锁）由主控当场在 `d2511e56` 修，`rr-lint-go`、`rr-keywords`、`rr-doc-links` 全过。
  - **合并前的终检** `final.sh fin`（`d2511e56`）一次过：七项全 0；bootstrap 第 0 路 453.2 秒、第 1 路 655.4 秒；e2e 493 passed；test-web 全部命中缓存（16/16，网页的输入自上次通过未变）。
  - **最终分支 CI** 44945103629 第一次全绿：server (0) Test 650 / bootstrap 465.5，server (1) Test 967 / bootstrap 938.8（期限的 78.2%）；web、e2e 两片绿（§5、§6）。
  - **合并**：`--no-ff` 合入 main，`9d26b1c3`（父 `1168fd25`、`d2511e56`），推送后以 `ls-remote` 核对。**main CI** 44947828240 第一次全绿：server (0) Test 662 / bootstrap 475.1，server (1) Test 773 / bootstrap 752.5（62.7%）；web、e2e 两片绿。
  - **归档与清理**：归档 `.superpowers/sdd/M7-P3c-integration-20261122`（清单 3507 个文件、828 个目录，分类只有可复现与证据两种、没有未知，`source_unchanged` 为真）；`cleanup.log`：worktree `m7-p3c` 已删，本地分支 `codex/m7-p3c`（在 `d2511e56`）与远端分支已删。
