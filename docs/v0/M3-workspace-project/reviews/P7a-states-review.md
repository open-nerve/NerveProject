# M3/P7a 状态与按资源寻址的共用取锁路径：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P7a `states` |
| 日期 | 2026-10-06 |
| 结论 | **通过**：10 个 Task 逐个实现、逐个评审，都通过（T3 有两条 Important、T6 有一条 Important，都是 plan 自己的测试缺口，当轮补上）；整分支评审"修复轮之后可以合并"，Critical 0、Important 0、Minor 3；修复轮一轮落实它的发现和执行中搁置的 P1–P12，范围复审通过、没有要改的（第 4 节） |
| spec / plan | [spec](../specs/P7a-states.md) / [plan](../plans/P7a-states.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P7a；P7 拆成 P7a、P7b 见 `3f87fcd3`；3.6（删除状态的一行）、3.17 的修订见 `b6dc2639`；3.6 加锁表的状态几行见修复轮（P1） |
| 分支 | `worktree-m3-p7`，从 `main` 的 `46abb231`（P6 合并）分出 |

## 1. 评审方式

- **拆分**：第一次运行的架构子任务把状态和标签放在一个 P7 里做原型（T1–T9），计划的块已超出一个 Phase 能逐 Task 评审的大小；控制者上报证据，负责人 2026-10-05 裁定拆成 P7a `states`、P7b `labels`，"一步一步来，每一步做扎实"。设计随 `3f87fcd3` 拆出两个 Phase；控制者的裁定 S1–S8 定下拆分的细节（`$M3TMP/p7-split-rulings.md`）。第一次运行的 T7 之后的快照（`p7snap/T7`→`T8`→`T9`→`p7proto`）留给 P7b 重放（第 6 节）。
- **spec 和 plan 由第二次的架构子任务产出**（`a1dab910`）：由第一次运行的 T1–T7 重新切分、逐块评审，10 个 Task；135 个变异全部被发现；从 `3f87fcd3` 重放，最后的树与原型相同。控制者的裁定 A1–A9（`$M3TMP/p7a-architect-rulings.md`）。
- **执行前的预检**（opus，只读，`$M3TMP/p7a-preflight/preflight.md`）：高 0、中 1、低 10，独立重放 1,199 秒干净，竞争的一组（`race5`）干净。M1：`createState` 在锁下的判定、与删除竞争时的 404 只在单元一层被发现：`createState` 加进 `rowWrites`（按项目寻址，`creates`）。L1–L10：L1 的前提写进代码（`seededStates`），L4 保留代码的次序、修订设计 3.6 和 3.17（spec 第 3 节第 13 条），L6 的说明也说分诊，其余是文字。
- **修订**（`b6dc2639`：设计 3.6、3.17，spec 和 plan）：10 个 Task、9,613 行；137 个变异全部被发现（20 个按性质只在单元一层）；从 `3f87fcd3` 重放 306 个块 1,008 秒，与原型相同；`race5` 485 个子测试，没有 40P01。开始前控制者在预检的跨 Task 表上核对修订加的行，并按 plan 的块算出 25 个共用的文件和 Task 的先后（台账）。
- **执行**：
  - 10 个 Task 各由一个实现子任务完成（不指定模型），各由一个评审子任务（opus）检查是否符合 spec 和质量；Task 的修复轮用 sonnet 做范围复审，整分支的修复轮用 opus。同一时刻只有一个写入者；评审只读，与下一个 Task 的实现流水进行，改同一批文件的 Task 等前一个的评审回来；有写入者时评审在 `git archive` 的副本上跑。
  - 实现者在每个 Task 里按 plan 的变异表再跑变异，并加上自己的；T9、T10 没有生产代码，它们跑别的 Task 的变异表里标"（Task 9 起）""（Task 10 起）"的行（55 行、52 行），每一行在点名的测试失败。
  - Task 自己的测试缺口和说错的注释由实现者在提交之后、评审之前提出，或由 Task 评审发现，控制者逐项裁定（第 3 节），在同一个 Task 里补上；会与后面 Task 冲突的、或只是加固的，搁置到修复轮（P1–P12）。
  - **执行中没有发现生产代码的缺陷**：Task 一层的裁定都是测试的缺口、说错的注释、契约的文字或去重（T1-c 的 500 文字、T2-d 的长度检查）。
- **整分支评审**（opus，`46abb231..3d7157c8`）：结论"修复轮之后可以合并"，Critical 0、Important 0、Minor 3；它逐条判断了搁置的 P1–P12（P3、P6 换了修法），并列出 P7b 的交接（第 6 节）。
- **修复**：一轮（`441b1ae4`、`f27b1ed1`、`20ec2874`、`8612c99e`，按生产代码、Go 测试、端到端、设计文档四组），范围复审一次（第 4 节）。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | P6、W11 的接口版本通过，此前的每个故事仍然通过（共 69 个） | **通过。** T10 `c67c9984`：`make e2e` 69 passed；"（Task 10 起）"的 52 个变异都在 P6（`s5-create-guests` 在 W11）失败。T10-b 之后 W11 读工作区的每张表和 `workspaces` 一行，gus 的调用多写 `project_user_properties` 的变异在 W11 失败。修复轮之后（`8612c99e`）`make e2e` 69 passed；P12 之后 P5、P6、W11 经夹具的 `newAccount` 注册账户，步骤不变。 |
| 2 | 交错 10 和一组最后两个状态，两种顺序，`-count=5 -race`，没有 40P01 | **通过。** T9 `123dbdd9`：竞争的一组 `-count=5 -race` 没有 40P01、没有 DATA RACE；新的四个组合测试 `-count=3`（147 个子测试）。合并之前在最后的树上（`8612c99e`）：架构子任务的 `race5` 一组（交错 10 和组里最后两个的成对写、按行寻址的写的竞争和锁强度、P5b 的成员交错、最先锁工作区、每个写只写它的行，12 个测试）`-count=5 -race` 167 秒，60 次、495 个子测试（比修订时多 P6′ 的两种顺序）全部通过，没有 40P01、没有 DATA RACE。 |
| 3 | 项目成员的三个写的竞争和锁强度照旧；`createState` 也在竞争的测试里 | **通过。** T1 把 P5b 的两个按行寻址的写移到共用路径（逐步对照删除的代码，码、次序、锁不变）；T9 把 P5b 的两个测试推广为 `rowWrite`，P5b 的 15 个子测试改名保留、断言不少（T9 评审逐行对照）；`createState` 在竞争的后三行（裁定 M1）。T9-d、T9-e 之后建状态由 `creates` 点名、不带 `member` |
| 4 | 每个状态的写在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 有一行，少一行完整性核对失败 | **通过。** T4、T6、T7 各加行；`s34-first-lock-row` 在完整性核对失败 |
| 5 | `project` 的 `apitest.Main` 两个方向：四个状态码和 `listWorkspaceStates` 的 `workspace.not_found` 各由 `project` 自己的 HTTP 测试返回 | **通过。** T4、T6、T7、T8；T8-e 之后 `workspace.not_found` 的回答体在包里写一次、四处用，各处照旧比较整个回答体 |
| 6 | 整程序测试覆盖六个操作（矩阵、最先锁工作区、盖戳、连接、每个写只写它的行） | **通过。** T4–T9；T6-a 之后连接测试跑改组被拒绝的一行（组的计数在事务上，组合一层），T7-e′ 之后盖戳测试在 Web 归档时跑删除和设为默认（3.19 的"照常"在组合一层） |
| 7 | 状态的矩阵行和已归档项目的小表中状态的行通过，格子数和耗时记下 | **通过。** 合并之前在最后的树上：`TestPermissionMatrix` 721 格（P6 结束时 532 格，P7a 加 189 格，都在状态的行和已归档小表中状态的行），全部通过，1.58 秒（P6 结束时 2.02 秒）。 |
| 8 | `make gen-check`、`lint-go`、`test`、`lint-web`、`knip`、`test-web`、`e2e` 通过 | **通过。** 修复轮之后（`8612c99e`）：`lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`（`platform/jobs` 第一次就通过）；`gen-check` 通过，没有生成物变化；`lint-web`、`knip`、`test-web` 通过；`make e2e` 69 passed。`go.mod`、`go.sum`、`server/tools`、`pnpm-lock.yaml` 自 `3f87fcd3` 没有变化 |

持续集成：分支上每次推送都跑，server、web、e2e 三个任务。T3 修复之后的 `1a8b7eec`：run 37307860261；T6 修复之后的 `66d8918e`：run 37322842655；T7 修复之后的 `ac7bc11b`：run 37334478541；T8 修复之后的 `8883be86`：run 37343097180；T9、T10 的 `c67c9984`：run 37350677615；10 个 Task 完成的 `3d7157c8`：run 37352391350；修复轮之后的 `8612c99e`：run 37358059874；都通过。本提交推送之后分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 拆分（负责人）和 S1–S8、A1–A9、预检的 M1、L1–L10（第 1 节）。spec 第 3 节的 13 条由这些裁定定下，执行中没有再议。

**执行中：补上 plan 自己的测试缺口和说错的注释**（每条都先证明缺口，再证明补上之后测试咬得住）
- **T1-a、T1-c**：`MemberRoleChanger` 的说明说它整个在项目的锁下，嵌入的 `MemberByID` 先在锁外读：改为与 `MemberRemover` 一致；共用路径的 500 文字说是哪一种行（`project_member.update: row … read as …`），状态共用这条路径之后日志看得出是哪个写。**T1-b（不改）**：alice 不变的一行由调用记录蕴含，保留与 P5b 拒绝表的前后约定一致。
- **T2-a、T2-c、T2-d**：`TestCheckNewStateReportsEveryField` 说"每个字段报它的第一个问题"而没有一行展示：加两行；全部有效的修改用带标点的名称（项目的 `checkName` 混进来的变异过得了每一层）；长度检查（空白 `too_short`、超过 255 个字符 `too_long`）由 `checkStateText` 和项目的 `checkName` 共用一个 `checkLength`，码、文字、次序不变。**T2-b**：plan 的 `t-triage-allowed` 的说法搁置（P3）。
- **T3-a … T3-g（评审 Important 2 条）**：
  - **I1**：`updateState` 改不了 `default` 在任何一层都没有 Go 测试，"恰好一个默认"缺一角：`TestUpdateState` 在 Backlog（默认）上跑，行和回答里 `default` 仍是真。
  - **I2**：失败的行没有要求 `errors.Is(err, context.Canceled)`、不是 `*shared.Error`：数据库失败可能答成 `project.state_default` 或 `state_not_found` 而测试不知道；补上（P4a–P6 的行照旧通过，变异证明）。
  - 其余：存储测试的 4 个存活变异各一行（T3-a），共用的测试辅助移到 `store_test.go`（T3-d），去掉蕴含的断言（T3-e、T3-g），`TestUpdateStateWritesNoDeletedOrTriageState` 拒绝 `*shared.Error`（T3-f）。
- **T4-a … T4-d**：`state()` 的 `Default` 映射到 T8 才有测试钉住（T4-a，交给 T8 的 `webBacklogJSON`，T8 评审确认 `Default: false` 在单元一层失败）；`createState` 生成的状态 id 在单元一层没有测试：建第二个状态、核对 id 不同、非零、是新的（T4-d）；两处说明（T4-b、T4-c："membership states"与工作流的状态分开）。
- **T5-a、T5-b**：搁置（P6、P7）。
- **T6-a … T6-c（评审 Important 1 条）**：
  - **I1**：改组时数原来的组的那次读只在单元一层核对它在事务上：连接测试先把 QA 改到 started（409），再改名；存储走池的变异和用例的外层 context 变异都在组合一层失败。
  - 契约说"请求体在看状态之前核对"只在单元一层：矩阵加一行（分诊状态上 `{"name":""}`，每一列 422，T6-b）；组里最后一个的一行改为只有原来的组能拒绝（Todo 改到 started，T6-c）。
- **T7-a … T7-e′**：spec 第 3 节第 13 条的组合一层的钉子在矩阵和单元的行名里点名（T7-a）；`6` 改为 `len(stateLocked(...))`（T7-b）；假实现的"gone"说明锁在组合的 app 里挡住删除（T7-c）；`markDefaultState` 的契约文字搁置到 T8（T7-d → T8-d）。**T7-e′**：3.19 的"已归档项目的状态照常删除、设为默认"只在单元一层：盖戳测试在删除和设为默认两行前后归档、恢复 Web（控制者原先的 T7-e 以为矩阵读得到效果，204 没有回答，实现者 NEEDS_CONTEXT，改为方案 B）。删除被组拒绝时回滚的组合一层的钉子交给 T9（`o-delete-refusal-committed`）。
- **T8-a … T8-g**：`createProject` 重复 `decide` 的身体（T8-a，搁置 P10）；`TestListingWorkspaceStatesKeepsToItsWorkspace` 照它的说明核对已删除的状态不在任何列表（T8-b，`x-both-deleted` 之前在组合一层存活）；`listWorkspaceStates` 的次序说"按项目 id"（T8-c，实现者把同样的说法改到四处注释，T8-f 接受）；`markDefaultState` 的契约文字只说默认（T8-d，T8 是最后一个钉生成物 SHA 的 Task，一次 `make gen`）；`workspace.not_found` 的回答体写一次（T8-e）；`forbidden` 的回答体搁置（T8-g，P11）。
- **T9-a … T9-e**：`memberWorld` 把 Ops 的 Cancelled 移到 100000 是 Serialize 看得见 `s1-gs-project` 的前提，原来只看 200：读回答核对 100000（T9-a）；`TestEachStateWriteChangesItsRowsAlone` 的说明只说它展示的（分诊的两个拒绝什么都不写），"在事务之前"交给单元的两个子测试（T9-b）；`s21-last-wrapped` 的变异文字缺 `fmt`（T2-d 之后），重放用 `-fmt`（T9-c）；建状态在竞争里由 `creates` 点名，`row()` 对建状态失败（T9-d），建状态的一行不再带没人读的 `member`（T9-e）。T7 留下的 `o-delete-refusal-committed` 在 Serialize 的三个"删除在后"的子测试失败。
- **T10-a、T10-b、T10-d**：`plane-diff.md` 4.11 一组中唯一的状态一行说默认状态先答 `project.state_default`（新项目的 Backlog）；W11 由 `workspaceTables` 加 `workspaces` 一行取快照，标题照写；P6 标题对分诊状态不说"不再"。**T10-c**：三个故事里相同的 `account` 搁置（P12）。

**执行中：不改的**
- T1-b（上面）。T5 的两条搁置到修复轮（P6、P7）。T7-d 移到 T8。T10 评审更正了报告的一句（`o-state-in-another-workspace` 也在存储一层失败），没有只在端到端一层的性质。
- 搁置项 P1–P12 在修复轮处理（第 4 节）。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 0、Minor 3。评审逐条看过：组合起来的锁的次序和强度（工作区 S → 项目 N → 行；设为默认先写原来的默认）、时钟在最后一把锁之后、拒绝和失败什么都不留、项目恰好一个默认、非分诊的组不会空、分诊不可见、P5b 的写移到共用路径之后行为不变、`findWorkspaceAndDecide` 与 P4a 的两个读等价、模块之间没有引用和跨库的 JOIN。它用变异证明了 P6 是承重的：`lock.go` 的 `return h, first, nil`（用锁前读到的行做决定）只经成员关系的行失败，状态的测试没有一个看得见。

| 发现 | 处理 |
|---|---|
| **M1** `DefaultMarker` 的说明说它整个在项目的锁下，嵌入的 `StateByID` 先在锁外读（T1-a 同类，plan 规定的文字） | F1：改为"Its writes run …"，与 `StateUpdater`、`StateDeleter` 一致 |
| **M2** `project.state_name_taken` 的回答体在 HTTP 测试里手写两遍 | F2：并入 P11，加 `stateNameTakenJSON` |
| **M3** 三个状态的写各写一遍 `rowWrite[domain.State]` 的字面量，只差操作名；存储的两个列表各写一遍转换的循环 | F3：一个状态自己的构造函数（不放进 `lock.go`），P7b 的两个标签的写照它。F4（不改）：两个循环在 sqlc 生成的两种行类型上，共用的部分已在 `stateOf`，为四行写泛型得不偿失 |
| 搁置项 P1–P12 | 都同意，两条换了修法：**P3**（plan 的文字）记在第 5 节，不改 plan；**P6′** 不改假实现，在 `TestStateWritesOnOneProjectSerialize` 加一行"QA 被第一个写改到 started，第二个写在等锁时它自己的行变了，改到 backlog"：数的是锁下的组；上面的变异在新行的两种顺序都失败（评审的原型）。代价：状态的锁下重读在单元一层不钉，组合一层钉 |

**修复轮**（`441b1ae4`、`f27b1ed1`、`20ec2874`、`8612c99e`，按生产代码、Go 测试、端到端、设计文档四组）：一次分派，落实 F1–F3 和 P1、P2、P4、P6′、P7、P8、P10–P12（P3、P5 记在第 5 节）。
- **生产代码**（重构，行为不变）：状态的行的写由 `stateWrite(id, action, states)` 构造（`state_ports.go`，照 `member_ports.go` 的先例放在端口旁，不进 `lock.go`，F3）；`createProject` 经 `decide` 判定，目标和拒绝的次序不变（P10）；`DefaultMarker` 的说明（F1）。
- **Go 测试**：P6′ 的一行（`return h, first, nil` 在新行的两种顺序失败，旧的文件上存活）；项目名称的长度先于禁用字符（P4，`f-name-forbidden-first` 失败）；`updateState` 的每一行带自己的改动，不再重写假实现的合并（P7）；连接测试的说明（P8）；`project` 的 HTTP 测试里已有常量的回答体都改用常量（P11、F2，扫到另外 7 个文件）。
- **P2 的根因**是测试的时间假设，不是生产代码：River 按间隔精确地排定每次运行（`ScheduledAt` 相差恰好 1 秒），实际开始的时刻另加插入和取出的等待（每秒轮询一次），重负载下实际的间隔低到 101 毫秒。测试改为要求相邻两次的 `ScheduledAt` 恰好差 1 秒，没有余量；24 个忙循环旁 `-race -count=50` 50/50 通过。
- **端到端**：P5、P6、W11 共用夹具的 `newAccount`（P12）。**设计**：3.6 的加锁表把修改状态、设为默认分成两行，新建状态写出它的读（P1）。
- 24 个变异（都经 `go test -overlay`）都在预期的地方失败。

**修复的范围复审**：opus，`3d7157c8..8612c99e`：通过，没有要改的。它逐项对照裁定，在副本上重跑了 P6′、P4、P7、P10 的变异（都在说的地方失败），核对 P11 的常量与它们替换的字面量逐字相同，在 River v0.47.0 的源码里确认了 P2 的根因（`nextRunAt` 每次恰好加一个间隔、不合并、两个时刻同样截到微秒）。它的一条观察：测试把先收到的两次运行当作先排定的两次，饥饿超过 1 秒时第三次的 goroutine 可能先送到，测试会看到 2 秒的间隔。与实现者报告的 `start` 辅助函数的 100 毫秒余量（7 个测试共用，只在人为的极端负载下 160 次失败 6 次）同一类，都是 M2 的 `platform/jobs`，一起移交收尾（第 6 节，裁定 FW-a、FW-c）。`project.identifier_taken`、`project.name_taken` 的回答体各手写两遍、没有常量（P4a 的测试），也移交收尾（FW-b）。

## 5. 最终实现与 plan 的差异

plan 的块记录的是当时的文字，保留原样，本提交不改 plan。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：
  - 共用路径的 500 文字说是哪一种行（T1-c）；`checkLength` 由状态和项目的名称共用（T2-d，`project.go`、`state.go`）。
  - 契约：`listWorkspaceStates` 的次序"按项目 id"（T8-c、T8-f），`markDefaultState` 的 403 说明只说默认（T8-d）；生成物的 SHA 因此与 plan 的表不同（`openapi.yaml` `8261b02d…`、`schema.gen.ts` `9186c891…`、`gen/states.sql.go` `b9a78cbe…`）。
  - 修复轮：`stateWrite`（F3）、`createProject` 经 `decide`（P10），都是重构；`DefaultMarker` 的说明（F1）。
- **测试**：T1-a … T10-d 各轮（第 3 节）和修复轮。
- **文档**：设计 3.6 的加锁表（P1）；`plane-diff.md` 4.11（T10-a）。
- **plan 自己的文字**（不改，记在这里）：
  - T1、T2 的块是修复之前的文字（plan:282 的 500 文字；T2 的三个文件）。
  - 变异表：`t-triage-allowed` 说分诊不再是 422，它照旧是 422（`invalid_format`），在问题的码上失败（P3）；T5 的说明说 `fakeStates.StateByID` 用 `answersAs`，代码没有（对的，S6），`g-update-unchecked`、`g-update-n`、`s5-update-members` 的 `TestPermissionMatrix` 少"（Task 6 起）"（P5）；`s4-create-frozen-clock`、`s4-update-frozen-clock` 的锚点从 T7 起才匹配，在各自的 Task 重新锚定之后在组合一层失败；`s22-row-id` 的锚点是 T1-c 之前的文字；`s21-last-wrapped` 缺 `fmt`，重放用 `s21-last-wrapped-fmt`（T9-c）。
  - T9：建状态的一行没有 `member`（T9-e）；T10：W11 读的表多于 plan 的五张（T10-b）；P6 的假实现不改（P6′）。
- 文件大小：分支改过的源文件都在约 400 行以内（最大的是 `permission_matrix_seed_test.go` 390 行、`permission_matrix_test.go` 387 行、`e2e/fixtures/assert/workspace.ts` 378 行；`lock.go` 273 行）；契约 `api/modules/project.yaml` 1,139 行（见第 6 节 P7b 第 10 条）。

## 6. 移交事项

同一个里程碑之内的后续 Phase（写进它们的 spec；spec 第 5 节各行照录，下面是执行之后的补充）：
- **P7b**（`labels`）：
  - **spec 第 5 节 P7b 一行照录**：共用路径（`lockRowAndDecide`、`rowWrite`、`placed`）、竞争和阶梯、矩阵、假实现、已归档的小表、第一次运行 T8–T10 的重放、承接的 P4a、P4b、P5b、P6 条目。
  - **重放的补充**：P7a 改过 `state.go`（T2-d 的 `checkLength`），spec 第 5 节原来的"P7a 没有改 `state.go`"已随本提交改正：第一次运行的 T9 把 `checkShortText` 移出 `state.go`，重放时手工合并，标签的名称用 `checkLength`。契约里状态的两句改过（T8-c、T8-d）；`fakes_write_test.go`、`deletion.go` 照 spec 第 5 节手工合并。第一次运行的快照在 `.superpowers/m3tmp-backup/nerve-m3.tgz`（`/tmp` 下的副本可能被清理），P7b 用过之前保留。
  - **共用件**：标签的两个按行寻址的写照状态的 `stateWrite`（`state_ports.go`，F3）写一个构造函数；`rowWrite.read` 用于守卫的写之后的重读（`deletedNothing`）；`createLabel` 经 `lockAndDecide`、`listLabels` 经 `findAndDecide`；路径的键的核对已由 `lock_test.go` 钉一次，不再加行（S6）。
  - **测试的骨架**：`rowPaths`、`underRow`（加 `labelNamed`）、`underProject`（加 `{label_id}`）；`projectOfRow`、`seeded.state`、`matrixStates` 和 `seededStates` 式的前提；`ofState`、`ofArchivedState`、`toState`；`project_row_races_test.go` 的 `row()` 和 `creates`；`stateWrittenHolding`、`memberWorld.projectLocks()`、`tx()`、`rowsBut`、`tableRows`；e2e 的 `workspaceTables`（加 `labels`）和 `deletion.go` 的"P7b adds the labels at the end"。
  - **这里踩过的坑**：
    1. 锁下重读回答不了变化的值的假实现（P6）：标签的组合测试要有一行"第二个写自己的标签在等锁时变了"（例如 `parent_id`），不只是旁边的一行。
    2. 嵌入了查找的端口，说明不能说整个在锁下（T1-a、F1）。
    3. 连接测试要真的跑每一个决定所依赖的读，例如被拒绝的层级改动（T6-a）。
    4. 承重的世界的设置要读回自己的回答（T9-a）。
    5. 每个映射到 HTTP 的字段在映射它的 Task 钉住（T4-a：`Default` 四个 Task 没有钉）。
    6. 204 的写在矩阵里看不到效果；已归档项目上的效果由盖戳测试在归档前后跑（T7-e′）。
    7. 失败的行要求 `errors.Is(err, context.Canceled)`、不是 `*shared.Error`（T3-c）。
    8. 存储按唯一索引的确切名字映射冲突（`stateTaken`），标签的名称索引不分大小写也一样。
    9. 变异的锚点：fix 轮改过的地方 plan 的块是之前的文字；`s21-last-wrapped` 用 `-fmt`。
    10. `api/modules/project.yaml` 1,139 行，加标签会到约 1,400 行：P7b 决定约定是否允许按资源拆分。
- **P8**、**P11**、**M4**、**M7**：spec 第 5 节各行照录，执行中没有变化。
- **收尾**：spec 第 5 节的一行照录，另加修复轮移交的三条（都是测试，不是 P7a 的代码）：
  - **`platform/jobs` 的测试去掉时间的余量**（FW-a、FW-c，M2 的包）：`start` 辅助函数要求 Start 在 100 毫秒内返回（7 个测试共用），只在人为的极端负载下 160 次失败 6 次，改为期限；两个 Stop 的测试的上界同类；`TestRunnerWorksAPeriodicJobUntilStopped` 收集到 Stop 为止每次运行的 `ScheduledAt`、排序之后要求相邻的恰好差 1 秒，不再假定先收到的就是先排定的。
  - **`project` 的 HTTP 测试**里 `project.identifier_taken`、`project.name_taken` 的回答体各手写两遍（FW-b，P4a 的测试）：照 P11 写成常量。
  - e2e 的故事里内联的 `createPAT(api, (await register(…` 约 52 处（P7a 只把三个故事里相同的那一个提到夹具的 `newAccount`，P12）：一并收拢到夹具。

## 7. 已知限制

- **`createProject` 锁工作区的回答没有键可核对**（spec 第 3 节第 1 条）：按 slug 锁，P4a 的形状；真实的存储按 slug 的唯一索引回答。
- **分诊状态的名称占用名称**（第 6 条）：建或改名为 Triage 答 409，而调用者列不出它；M7 取出分诊状态时再议。
- **`SetDefaultState` 的守卫在组合一层到不了**（第 4 条）：纵深防御，由单元和存储一层钉住。
- **契约不说工作项**（第 7 条）：P4a 的几个字段照旧，收尾一次清扫。
- **状态的锁下重读只在组合一层钉住**（P6′）：单元的假实现回答不了等锁时变了的组。
