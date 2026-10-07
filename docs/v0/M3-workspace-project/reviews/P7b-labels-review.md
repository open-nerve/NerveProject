# M3/P7b 标签：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P7b `labels` |
| 日期 | 2026-10-07 |
| 结论 | **通过**：11 个 Task 逐个实现、逐个评审，都通过（T3 有一条 Important、T5 有一条 Important、T11 有一条 Important，都是 plan 自己的测试缺口，当轮补上）；整分支评审"修复轮之后可以合并，加上 I1"，Critical 0、Important 1、Minor 2；修复轮一轮落实 I1、M1 和执行中搁置的 P1–P6，范围复审 通过、没有要改的（第 4 节） |
| spec / plan | [spec](../specs/P7b-labels.md) / [plan](../plans/P7b-labels.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P7b；P7 拆成 P7a、P7b 见 `3f87fcd3` |
| 分支 | `worktree-m3-p7b`，从 `main` 的 `0deb8c34`（P7a 合并）分出 |

## 1. 评审方式

- **拆分**：负责人 2026-10-05 裁定 P7 拆成 P7a `states`、P7b `labels`，"一步一步来，每一步做扎实"。P7a 合并（`0deb8c34`）之后 P7b 开工。
- **spec 和 plan 由架构子任务产出**（`c2943340`）：从第一次运行的标签部分（`p7snap/T7`→`T8`→`T9`→`p7proto`）接过来，放在 P7a 之上重新切成 11 个 Task；105 个变异全部被发现；从 `0deb8c34` 重放，最后的树与原型相同（3,130 个文件）。架构子任务被每周的限流打断一次，经 SendMessage 续上原来的上下文。控制者的裁定 B1–B4（`$M3TMP/p7b-architect-rulings.md`）：B1 契约文件不拆；**B2 不取架构子任务的做法**：`createLabel` 不收 `sort_order`，照设计 4.10（新标签在最大值加 10000，没有时 65535），与 Plane 的网页一致；B3 十一个 Task；B4 端到端的排除留到 Task 11，附带条件。
- **执行前的预检**（opus，只读，`$M3TMP/p7b-preflight/preflight.md`）：高 0、中 2、低 4，独立重放干净（3,130 个文件没有差异）。M1：端到端排除的删去不承重：`expectProjectDeleted` 加自查（读出的表等于有 `project_id` 列的每张表）和 Task 11 的变异表。M2：在锁前读最大 `sort_order` 只在单元一层被发现：交错核对未删除的标签的 `sort_order` 各不相同。L1–L4：标记、同一工作区的父标签反例、**L3 现在就把假实现的按 id 重读和矩阵的行做成泛型**（`readRow`、`rowsByID`）、`plane-diff` 的一行。
- **修订**（`1b7b8966`）：plan 7,842 行；108 个变异全部被发现（11 个按性质只在单元一层）；PF-L3 之后 P7a 的测试不变（同样的子测试，P7a 的变异在同样的测试失败）；重放干净（291 个块）；`race5` 670 个子测试，没有 40P01；矩阵 853 格；端到端 69/70，只有 S3（F4，与 P7b 无关）。
- **执行**：
  - 11 个 Task 各由一个实现子任务完成（不指定模型），各由一个评审子任务（opus）检查是否符合 spec 和质量；Task 的修复轮用 sonnet 做范围复审。同一时刻只有一个写入者；评审只读，在 `git archive` 的副本上跑，与下一个 Task 的实现流水进行；改同一批文件的 Task 等前一个的评审和修复回来（台账里的文件重叠表和流水次序）。
  - 实现者在每个 Task 里按 plan 的变异表再跑变异（含前面的 Task 标"（Task N 起）"的行），并加上自己的；每个 Task 都跑过 `make lint-go`、`make test`，接口的 Task 加跑 `gen-check` 和前端的检查。
  - Task 自己的测试缺口和说错的注释由实现者在提交之后提出，或由 Task 评审发现，控制者逐项裁定（第 3 节），在同一个 Task 里补上；跨 Phase 的去重和早先 Phase 的单行缺口搁置到修复轮（P1–P6）。
  - **执行中没有发现生产代码的缺陷**：Task 一层的裁定都是测试的缺口、说错的注释、契约的文字或去重。
- **整分支评审**（opus，`0deb8c34..a12e7295`）：结论"修复轮之后可以合并，把 I1 加进修复轮"，Critical 0、Important 1、Minor 2；逐条判断搁置的 P1–P6（都同意），逐条核对 P7a 留下的十个坑（都照办），列出后续的交接（第 6 节）。
- **修复**：一轮（`67dfa9d3`、`7651c601`、`d200a408`，按安全测试、早先 Phase 的单行缺口、测试辅助的合并三组），范围复审一次（第 4 节）。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | P7 的接口版本、断言标签的 W3 和 P4 通过，此前的每个故事仍然通过（共 70 个） | **通过。** T11 `b4fdc71e`：`make e2e` 70 passed。`expectProjectDeleted` 的自查：留着排除、少读一张表（`states`）、一张有 `project_id` 而没有外键的新表，三个错误各在 P4、W3 的自查失败，去掉自查时三个都通过（预检 M1 关闭）；`k-no-labels-step` 在 P4、W3、W2 失败。T11-b 之后两个列表都按 "C" 排序（数据库的默认排序规则下 `issues` 与 `issue_views` 的次序不同，M4 会碰到）。修复轮之后 修复轮之后（`d200a408`）`make e2e` 70 passed，P7 多一个拒绝（F-I1）。 |
| 2 | 交错 11 和同名的并发创建两种顺序 `-count=5 -race` 通过，没有 40P01 | **通过。** T10 `7cee413c`：六个测试 `-count=5 -race` 30/30、570 个子测试，没有 FAIL、40P01、DATA RACE。合并之前在最后的树上（`d200a408`）：`race5.sh` 一组（交错 11 和标签的其余六对、交错 10 和组里最后两个的成对写、按行寻址的写的竞争和锁强度、P5b 的成员交错、最先锁工作区、状态和标签的写各只写它的行，14 个测试）`-count=5 -race` 254 秒，70 次、670 个子测试全部通过，没有 40P01、没有 DATA RACE；修复轮的串行化两个测试 `-count=5 -race` 130/130。 |
| 3 | 迁移 `00014` 升、降、再升通过 | **通过。** T1；`make test` 每个 Task 都跑 |
| 4 | 每个标签的写在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 里有一行，少一行时完整性核对失败 | **通过。** T5、T7、T8 各加行（`writesOnAProject` 由矩阵的行得出）；`rowPaths` 加 `/api/v0/labels/`，按行寻址的第一步探测标签行；T8 评审的变异（事务外的 TxManager）在这里失败 |
| 5 | `project` 的 `apitest.Main` 两个方向（含两个标签码） | **通过。** T5（`project.label_name_taken`）、T7（`project.label_not_found`） |
| 6 | 整程序测试覆盖本 Phase 的操作（矩阵、最先锁工作区、盖戳、连接、竞争、每个写只写它的行） | **通过。** T5、T7、T8、T9、T10；盖戳的三个标签的写在 Web 归档期间（3.19 的"照常"在组合一层） |
| 7 | 项目级矩阵的全部格子通过，已归档项目的小表含每个状态和标签的操作；格子数和耗时记下 | **通过。** T9 之后 853 格。合并之前在最后的树上（`d200a408`）：`TestPermissionMatrix` 865 格（P7a 结束时 721 格，P7b 加 144 格：标签的四个操作 15 行和已归档小表中标签的行），全部通过，1.58 秒（P7a 结束时 1.58 秒）。 |
| 8 | `make gen-check`、`lint-go`、`test`、`lint-web`、`knip`、`test-web`、`e2e` 通过 | **通过。** 修复轮之后（`d200a408`）：`lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`；`go test -race ./internal/bootstrap/` 一次通过；`gen-check` 通过，没有生成物变化；`lint-web`、`knip`、`test-web` 通过；`make e2e` 70 passed。`go.mod`、`go.sum`、`server/tools`、`pnpm-lock.yaml` 自 `0deb8c34` 没有变化 |

持续集成：分支上每次推送都跑，server、web、e2e 三个任务，每次都通过：T3 修复 `22fcb2b9` run 37445647728；T4 `31800846` run 37447071813、修复 `e0828f66` run 37464912927；T5 `ff61e1c4` run 37463824986、修复 `1233bdd1` run 37471032854；T6 `d5f638fd` run 37465979209、修复 `d94883b5` run 37472245072；T7 `5b306efb` run 37475486972、修复 `50503a2c` run 37496661856；T8 `ae170f3b` run 37498632804、修复 `975df8db` run 37503101752；T9 `50592f16` run 37507167102、修复 `b51341f8` run 37513023468；T10 `7cee413c` run 37511470493、修复 `59dae991` run 37516857232；T11 `b4fdc71e` run 37514811126、修复 `a12e7295` run 37544832335；修复轮 `d200a408` run 37552155824。本提交推送之后分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 拆分（负责人）、B1–B4、预检的 M1、M2、L1–L4（第 1 节）。spec 第 3 节的 13 条由这些裁定定下，执行中没有再议。

**执行中：补上 plan 自己的测试缺口和说错的注释**（每条都先证明缺口，再证明补上之后测试咬得住）
- **T1-a … T1-d**：e2e 的注释点名故事（T1-a）；删除工作区的种子里标签的写者是项目的管理员（角色 20）而不是不能写标签的成员（T1-b），并且删除之前读回"项目下没有一行最后由删除者写"的前提（T1-d，一个 `noRowLastWrittenBy`，两个删除测试共用）；`labels.project_id NOT NULL` 由 sqlc 的类型和 `gen-check` 钉住（T1-c，不加行）。
- **T2-a … T2-e**：`CheckParent` 在两个拒绝同时成立时答哪个，没有一行展示：加四行，每对一行（T2-a）；`checkIdentifier` 用 `checkMaxLength`，行为不变（T2-b）；"no parent"在这个包里指顶层，改为"父标签不在或已删除"（T2-c）；P4a 的标识的两个行为搁置（T2-d，P1）；第五对不可能同时成立，不加行（T2-e）。
- **T3-a … T3-e（评审 Important 1 条）**：
  - 实现者先提出的存储测试的四个缺口（之前的写者、截到同一微秒的"再次删除"、`created_at`/`updated_at` 的映射、`UpdateLabel` 违反 CHECK 和外键）在评审之前补上（T3-a）；P7a 的 `stateOf` 同一个缺口搁置（T3-b，P2）。
  - **I1**：`TestListLabels` 的并列分不出按 id 还是按名称（早 id 的标签按名称也排在前）：改名为 Wiki；`ORDER BY sort_order, name` 的变异在旧的测试上存活、新的失败（T3-c）。
  - 内部违反的三行检查复制了五处：一个 `internalViolation`（T3-d）；`stateWorld` 的说明（T3-e）。
- **T4-a … T4-c**：`LabelCreator` 的说明说整个端口在锁下而它嵌入 `LabelFinder`（P7a 坑 2 同类）：说明只说它的写（T4-a）；第二个标签只核对 id，`x-greatest-cached`（结构上缓存最大值）过得了每个单元测试：核对它的 `sort_order`（T4-b）；P7a 的 `TestCreateState` 同一个缺口搁置（T4-c，P3）。
- **T5-a … T5-f（评审 Important 1 条）**：
  - **I1**：矩阵的种子没有读回 `workspace_id`，"另一个项目的父标签"一行是 `r-parent-same-workspace` 在任何一层唯一的杀手，靠一个没人读的种子：`seededLabels` 读回 id、工作区、父标签（T5-a、T5-b），五个种子的变异之前都通过整个 bootstrap 包，之后都在 prepare 失败。
  - 连接测试的注释说"两层"而只核对 422：核对回答的说明；同一个测试里 P7a 的 409 也核对码（T5-c）。实现者为了不碰 T7 的锚点用了一个一次性的变量、`send` 读了再清空：**控制者不接受**，改为一行之内显式的 `refusedWith(t, send(…), problem)`，并修订 plan 里 T7 的两个块随之（T5-e，`1233bdd1`）。理由：隐式地把一行和下一行连起来是乱接线；后面的块的锚点是 plan 的文字，跟着代码走。
  - 种子的写者的说明（T5-d）；复审说"WA- 能写标签"，控制者驳回：`createLabel` 的格子里 WA- 是 403（T5-f）。
- **T6-a … T6-d**：`labelLocked` 复制 P7a 的 `stateLocked`：一个 `rowLocked`，两者是一行的包装（T6-a）；唯一给颜色的一行给的是 `""`，与"清空颜色"的变异同值：另一行给 `#FF0000`（T6-b）；`parentRead` 由 `labelUpdated` 取前 8 个（T6-c）；"标签不变"由调用记录蕴含，与 P7a、T4 的形状一致，不改（T6-d）。
- **T7-a … T7-e**：改名的一行在已在顶层的 Feature 上，"不给 `parent_id` 时留在原处"看不出来：改在 Bug 下的 UI 上，核对父标签、`sort_order`、颜色（T7-a、T7-b；UI 的种子带颜色，`seededLabels` 读回）；`labelID` 复制 `stateID`：一个 `namedRowID`（T7-d）；"把 UI 改名 Widgets 到顶层"读回答（T7-e）；矩阵的 12 列格子写了三遍搁置（T7-c，P4）。
- **T8-a、T8-b**：连接测试的注释说删除 Bug 连同 Widgets，没有读回：经池读回两行同一时刻删除、Widgets 在 Bug 下（T8-a）；`LabelDeleter` 的说明与两个兄弟同形（T8-b）。
- **T9-a … T9-d**：列出的标签的颜色、`sort_order`、工作区在任何一层都没有核对：存储的 `TestListLabels` 比较整个标签（T9-a）；P4b 的 `refusedAs` 的 `errDisk` 一支也要求链里没有 `*shared.Error`（T9-b，失败的行的规则）；假实现的次序还要挡住按名称和"顶层在前"的排序，守卫列出它挡住的七种（T9-c）；读的拒绝表写了四遍搁置（T9-d，P5）。
- **T10-a … T10-d**：行的测试的说明不叙述它没读的效果，交错的说明不说"一条语句"（组合一层看不到机制）（T10-a）；409、422 的行核对码（T10-b）；失败时打印两边的字段（T10-c）；串行化测试的循环复制 P7a 的搁置（T10-d，P6）。
- **T11-a … T11-f（评审 Important 1 条）**：
  - 故事 P7 的标题说成员的写被拒绝而没有试删除：加一行 403（T11-a）。
  - **I1**：自查比较的两个列表按不同的排序规则排（`regclass::text` 按数据库默认 en_US，`information_schema` 按 "C"），今天的四张表碰巧一致，M4 的 `issues`、`issue_views` 会让集合相等的检查失败：两个列表都按 "C"（T11-b）。控制者的原话 `ORDER BY 1 COLLATE "C"` PostgreSQL 不收（序号不能带排序规则），实现者把 `COLLATE "C"` 放在选出的名称上，控制者接受（T11-f：裁定的意思对，字面写错了）。
  - W3 删除 Old 之前读回 Old 的标签（T11-c）；P7 一步的注释不说颜色（`""` 是默认，看不出来，T11-d）；Acme 的 Web 的每个标签逐行核对删除的时刻（T11-e，之前早 1 秒的变异在 W3 存活）。
- **T3 留下的一条**：端到端 P7 是否依赖 `sort_order` 的并列：唯一的并列（已删除的 Bug 与新的 bug 都在 65535）只在 `expectLabels` 的数据库读里，按名称打破（两边都是），API 的列表不含已删除的标签。

**执行中：不改的**
- T1-c、T2-e、T5-f、T6-d（上面）。搁置项 P1–P6 在修复轮处理（第 4 节）。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并，把 I1 加进修复轮"：Critical 0、Important 1、Minor 2。评审逐条看过：组合起来的锁的次序和强度（工作区 S → 项目 N → 行）、时钟在最后一把锁之后、拒绝和失败什么都不留、没有第三层和环、删除的父标签不留未删除的子标签、名称不分大小写唯一、P7a 的状态的矩阵行移到 `rowsByID`、假实现的 `StateByID` 移到 `readRow` 之后行为不变、四个端口的说明一致、模块之间没有引用和跨库的 JOIN、契约的说明对每个调用者都对、两个连带对标签完整；P7a 留下的十个坑逐条照办。

| 发现 | 处理 |
|---|---|
| **I1** 给一个已经有父标签的标签换一个规则不允许的父标签，在任何一层都没有测试：`update_label.go:49` 加 `&& l.ParentID == nil`（有父标签的标签跳过子标签和父标签的检查）过得了单元、全部组合的标签测试，按阅读也过得了每个故事。代码今天是对的，没有测试守着（设计 12 节约束 3） | F-I1：单元两行（Bug 下的 UI 放到 Ops 的 Docs 下、放到自己下面），矩阵一行（UI 给 `acme/archived` 的 Bug），P7 一个拒绝（Widgets 放到 Ops 的 Bug 下）；变异在三层都失败 |
| **M1** `bootstrap` 里两个新的读问题回答的辅助（`refusedWith`、`refusalOf`），旁边已有 `problemCode`、`oneError` | F-M1：并进修复轮的辅助合并 |
| **M2** 端到端的 `workspaceTables` 是手工维护的列表，没有像 `expectProjectDeleted` 那样的目录自查 | 不改，移交 M4（第 6 节）：列表在 P2、P3 就有；Go 的删除工作区的测试已经是目录驱动的；M4 的工作项的表加进来时一起加自查 |
| 搁置项 P1–P6 | 都同意，没有一条比说的更糟、没有一条承重 |

**修复轮**（`67dfa9d3`、`7651c601`、`d200a408`）：一次分派，落实 F-I1、F-M1 和搁置的 P1–P6，全部是测试。
- **安全测试的缺口**（`67dfa9d3`，F-I1）：单元两行（Bug 下的 UI 放到 Ops 的 Docs 下、放到自己下面）、矩阵一行（UI 给 `acme/archived` 的 Bug，格子与另一个父标签的行相同）、P7 一个拒绝（Widgets 放到 Ops 的 Bug 下）。`m1` 在修复之前过得了单元、组合和 P7，之后在三层都失败。
- **早先 Phase 的单行缺口**（`7651c601`）：P4a 的标识空白答 `invalid_format`、过长又不合格式答 `too_long`（P1）；P7a 的 `stateOf` 的两个时刻在一个改过的状态上各对各（P2，放在 `state_writes_test.go` 的 `TestUpdateState`：`states_test.go` 里没有建了之后再写的状态，实现者指出，控制者接受）；P7a 的 `TestCreateState` 核对第二个状态的 `Sequence`（P3）。四个变异修复之前存活、之后失败。
- **测试辅助的合并**（`d200a408`）：矩阵的格子一个辅助（按不可见的格子，项目、行、成员关系三处用，P4）；读的拒绝表一个构造（四个测试用，P5）；串行化的一对一个运行者（状态和标签两个测试用，P6）；`bootstrap` 读问题回答一个（F-M1）。865 格逐字节相同，2,117 个子测试名不变，17 个变异修复前后在同样的测试失败。
- 实现者的两条更正（都是简报的笔误，接受）：状态的串行化测试是 12 个子测试，不是 14；P2 的落点见上。

**修复的范围复审**：opus，`a12e7295..d200a408`：通过，没有要改的。它在副本上重跑了 `m1`（基线上存活，修复之后在单元的两行、矩阵的两格失败）、P1 的两个变异和自己写的 P3 的缓存变异（基线上存活、修复之后失败）；自己写了导出每一行格子的测试，三个提交上比较（C 前后逐字节相同，A 只多一行 12 格）；比较 `7651c601` 与 `d200a408` 的子测试名（相同）；T10 评审的锁前读父标签的变异和一个只改说明的变异照旧在原来的地方失败。它的一条观察，不用改："UI 放到自己下面"同时触犯两条规则（有父标签的标签做自己的父标签必然也违反两层），`sameError` 不比较说明；这一行的用处是杀掉 `m1`，"不能做自己的父标签"单独由"Feature 放到自己下面"钉住。

## 5. 最终实现与 plan 的差异

plan 的块记录的是当时的文字，保留原样（T5-e 的修订除外，见下），本提交不改 plan。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：没有与 plan 不同的地方。端口的说明（T4-a、T8-b）是文字。
- **测试**：T1-b … T11-e 各轮（第 3 节）和修复轮。
- **plan 自己的文字**：
  - **T5-e 修订了 plan**（`1233bdd1`）：T7 在连接测试 Icons 一行上的两个块随 `refusedWith` 改写，Bug 放到 UI 下的一行也用它；后面的块逐个核对仍恰好匹配一次。
  - T11-b 的 `COLLATE "C"` 放在选出的名称上（T11-f），plan 里是排除之前的文字。
  - `p7bsnap/T*` 是修复之前的快照：T7 起实现者比较快照时，差异都追溯到修复轮。
  - Task 10 的说明提到交错第 7 对的变异在 95535 相撞，实际种子在 1–4，相撞在 65535、10004；测试不写字面值，由唯一性核对杀掉（T10 评审核对）。
- 文件大小：分支改过的源文件都在约 400 行以内（最大的是 `server/migrations/schema_test.go` 399 行、`permission_matrix_test.go` 392 行、`project_write_locks_test.go` 382 行、`e2e/fixtures/assert/project.ts` 381 行）；契约 `api/modules/project.yaml` 约 1,350 行（裁定 B1 的写明的例外）。

## 6. 移交事项

同一个里程碑之内的后续 Phase 和之后的里程碑（写进它们的 spec；spec 第 5 节各行照录，下面是执行之后的补充）：
- **P8（数据层）**：
  - `Label`、`LabelList`、`LabelCreate`、`LabelUpdate` 取自 `schema.gen.ts`；Plane 的 `parent`、`project`、`workspace` 对应 `parent_id`、`project_id`、`workspace_id`；没有 `description`；没有颜色是 `""`。
  - 建：顶层的标签不发 `parent_id`（发 `null` 答 400），不发 `sort_order`（400），新标签一律在最后（最大值加 10000，或 65535）。改：`parent_id: null` 移到顶层；只发 `LabelUpdate` 有的字段（`additionalProperties: false`）。
  - 标签只按项目取；没有 Plane 的 `/workspaces/{slug}/labels/`（M7 把各项目的列表拼起来）。`listLabels` 对已归档的项目照常列出（与 `listStates` 不同）。
  - 两个码随 `PROBLEM_MESSAGES` 搬迁。
- **P11（页面）**：
  - 写标签的只有项目管理员和项目成员兼工作区管理员；成员和访客只能列出，控件照此隐去。
  - 409 `project.label_name_taken`（不分大小写）落到名称字段：Plane 的表单在 `errors.name` 里找 `LABEL_NAME_ALREADY_EXISTS`，要改映射。422 `parent_id not_allowed` 显示服务端的说明。
  - 拖进空的子列表时只发 `parent_id`（位置不变）；中间位置的 `sort_order` 是任意数。删除一个标签连同它的子标签：刷新 store。
- **M4**：
  - 工作项带标签（`issue_labels`）：定下删除标签时工作项上的标签怎么办。
  - `issue_labels` 的写如果不持有项目的 N 或工作区的 N，就落在约定五的例外里，`DeleteLabel` 的一条语句要改为按 id 先锁。
  - 任何新的设 `parent_id` 的路径都要再核对两层和已删除的父标签的规则；I1 表明"给已有父标签的标签换父标签"要有自己的拒绝行。
  - 工作项的表加进端到端的 `workspaceTables`，并考虑像 `expectProjectDeleted` 那样的目录自查（整分支评审 M2）。
  - `expectProjectDeleted` 对任何有 `project_id` 而没有指向 `projects` 的外键的新表失败（有意的）；两个列表保持 `COLLATE "C"`。
  - 60 天的物理清理可以用不带条件的 `labels_*_idx`。
  - 工作项的写是否取工作区的 S 由 M4 决定（负责人 2026-10-02），照旧。
- **M7**：跨项目的标签列表是各项目列表的并集，没有工作区级的标签。
- **M8**：部署说明写明数据库用 UTF-8 的 `LC_CTYPE`（`lower(name)` 按数据库的 locale 折叠大小写，spec 第 7 节）。
- **收尾**：
  - 核对 3.20 的 P7b 两行（总体设计 5.3、`plane-diff.md` 两节）。
  - 裁定 B1（契约文件不拆）落到设计。
  - 架构子任务的重放里 `make gen-check` 在 git 检出之外什么都不查：以后的架构简报在重放里重新生成并比较（P7b 用了 gen-same）。
  - P7a 移交收尾的三条照旧（`platform/jobs` 的余量、`project` 的 HTTP 测试里两个回答体、e2e 里内联的 `createPAT(register)`）。

## 7. 已知限制

- **不分大小写跟随数据库的 `LC_CTYPE`**（spec 第 7 节）：以 `C` 初始化的数据库只折叠 ASCII。
- **`createProject` 锁工作区的回答没有键可核对**（P7a 第 3 节第 1 条）：照旧。
- **标签的锁下重读只在组合一层钉住**（P7a 的 P6′ 同类）：单元的假实现回答不了等锁时变了的父标签；交错的七对在组合一层。
- **"一条语句"是存储的机制**：`DeleteLabel` 用一条语句删除标签和它的子标签，由存储测试钉住；组合一层看到的是结果（在项目的锁下，刚放到 Bug 下的标签随它删除），先另读子标签再按 id 删除在项目的 N 下等价（T10 评审的 M2）。
- **端到端的 `workspaceTables` 是手工的列表**（整分支评审 M2）：移交 M4。
