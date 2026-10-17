# M4/P1a 工作项：评审记录

| 项 | 内容 |
|---|---|
| Phase | M4/P1a `issues`（设计中的 P1 由控制者按决策点 2 的预先批准拆成 P1a、P1b，2026-10-17） |
| 日期 | 2026-10-18 |
| 结论 | **通过**：15 个 Task 逐个实现、逐个评审，都通过（T2、T3、T12 的评审各有一条 Important，T15 的评审有三条 Important，都是文字；都在当轮改掉）；整分支评审"修复轮之后可以合并"，Critical 0、Important 1、Minor 8；修复轮一轮（两个提交）落实 I1、M1–M8 和设计里过时的 D1–D8，范围复审：全部落实，没有新的问题，可以合并（第 4 节） |
| spec / plan | [spec](../specs/P1a-issues.md) / [plan](../plans/P1a-issues.md) |
| 设计 | [M4 设计](../M4-design.md) 第 12 节 P1a；P1 拆成 P1a、P1b 见 `d693f539` |
| 分支 | `worktree-m4-p1`，从 `main` 的 `1eb9686a`（M4 设计合并）分出 |

## 1. 评审方式

- **拆分**（`d693f539`）：P1 的架构子任务按 M3 已合并的 plan 实测，估出约 22–25 个任务、约 23,000–26,000 行，两个上限都超出（设计第 12 节 P1a 的"拆分"一段）。控制者按决策点 2 的预先批准裁定八条（2026-10-17，`$M4TMP/p1-split-rulings.md`）：拆成 P1a、P1b；P1a 仍超出时的退路（第 11、13 个任务移到 P1b）预先批准；任务可以在约 16 个、每个约 1,500 行以内重新切分；规模的测量在 P1b；两段的关键词 `phase` 都写 `M4/P1`；拆分先作为一个文档提交记进设计。
- **spec 和 plan 由架构子任务产出**（`5396a085`）：在 `$M4TMP/p1proto` 上逐 Task 做出原型、存快照（`p1snap/T0`…`T15`），plan 的块由相邻两份快照的差异生成。15 个 Task，最大的是 Task 2（1,435 行）；没有用到退路。从 `d693f539` 逐 Task 重放 408 个块，最后的树与原型的 3,343 个文件没有差异；136 个变异全部被发现（spec 附录 A.5、A.6）。控制者对 spec 第 3 节的裁定（`$M4TMP/p1a-architect-rulings.md`）：第 4–10、12–16 条照建议接受或确认，第 1–3、11 条是拆分裁定的落实。
- **执行前的预检**（opus，只读，`$M4TMP/p1a-preflight/preflight.md`）：High 0、Medium 0、Low 6。删除工作区的新顺序与 M3 设计 3.6 的取锁表、停用、P1a 的写都构不成等待环；独立重放 408 个块，与原型只差 spec 和 plan 两个文件（原型里是它们较早的稿）；30 个变异（取自变异表的 24 个，加上自己写的 PF.1–PF.6）全部被发现；39 个交错和取锁测试 `-race -count=5` 干净；矩阵 941 格全部通过。
- **修订**（`c37a3480`）：L1–L6 照预检的原文接受（`$M4TMP/p1a-preflight-rulings.md`），由架构子任务改进 spec 和 plan。L1 改的注释会被 sqlc 复制，`gen/delete_rows.sql.go` 重新钉住。从 `d693f539` 再重放一次，3,343 个文件没有差异。
- **执行**：
  - 15 个 Task 各由一个实现子任务完成（不指定模型，继承 Opus 5.5），用 `planapply.mjs` 原样应用 plan 的块；各由一个评审子任务（opus）检查是否符合 spec 和质量。Task 的修复轮之后有范围复审（sonnet 或 opus）。同一时刻只有一个写入者。
  - 实现者在每个 Task 里按 plan 的变异表再跑变异，在 `$M4TMP/p1sdd/` 里没有 git 的副本上跑。从 Task 9 起也跑前面的 Task 标"（Task n 起）"、落在自己这个 Task 的行。因容器起不来而失败的 Go 测试不算发现：实现者 grep 日志，重跑一次（Task 2 的 T2.10 出过一次这样的假发现）。
  - 实现者每个 Task 用 `treediff` 比较自己的树与原型的快照。Task 6 起，"已知的差异"就是修复轮的提交改过的文件（`git show --stat`），加上 spec、plan、设计。
  - Task 自己的测试缺口和说错的注释，由实现者在提交之后提出，或由 Task 评审发现，控制者逐项裁定（第 3 节）。每一轮修复之后，实现者在副本上证明后面的 Task 的块仍然能应用。
  - **执行中改了生产代码的只有两处**：Task 5 的修复轮（`604d4e66`：给出的空优先级被当作没给，而契约的枚举不收它）和整分支评审的 I1（`9e9db923`：`project.state_in_use` 的说明和文案给了一个谁都做不到的补救）。另有两处只改生产代码的注释（`5c0114b6` 的 `project/app/cascade.go`，`9e9db923` 的 `invitations.sql`）。其余都是测试的缺口、说错的注释和文档。
  - 规则的失误：六次，都是只读的，都没有影响（一次 `git -C`、一次被会话拒绝的 `git -C`、三次用 `;` 连接的命令、一次多余的 `cd`；台账 Task 1、3、8、12、14 和整分支评审的修复轮）。
- **整分支评审**（opus，`1eb9686a..25f5b9fb`，37 个提交，`$M4TMP/p1final/review.md`）：结论"修复轮之后可以合并"，Critical 0、Important 1、Minor 8，另列出设计里过时的句子 D1–D8。它在 `git archive` 的副本上重跑了全部门禁、39 个交错和取锁测试的 `-race -count=5`、25 个变异的样本（24 个被发现，留下的 T8P.4 就是 M1），逐条判断执行中搁置的各项（第 4 节）。
- **修复**：一轮，两个提交（`9e9db923`：代码、测试、契约、文案；`91e54133`：设计、spec、plan、交接）。范围复审一次（opus）：I1、M1–M8、D1–D8 都已落实，可以合并。它指出的一处遗留（设计第 12 节 P1a 任务 14 的"`api.ts` 的建工作项"）在本提交改正。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | I1 的接口版本通过；M3 的 P4、W2、W3 带着工作项一侧的行通过，没有留下端到端的过渡（`make e2e` 121 个） | **通过。** T14 `0ea7a4d8`：`make e2e` 121 个中 119 个通过，失败的两个是 A4 的 firefox 测试（`browserType.launch: Executable doesn't exist`，机器上没有 Firefox）；I1、P4（接口、页面）、W2、W3（三个）、W11 都通过。T14 的修复轮 `fbefd9ad` 之后、整分支评审的修复轮之后同样 119/121。`withoutRows` 删去；T14.3（留着过渡的排除）在 P4、W3 失败，T14.1、T14.2 在结构一层和故事失败（T14 的报告）。整分支评审在副本上：118/121，多出的失败是 S3（副本不是 git 检出，F4）；T3.1 的端到端一层在 P4、W3 的断言上失败（`e2ekills.py` 0 个可疑） |
| 2 | 交错 1、2、3、8、21、22 和交错 4–7 的创建一支 `-count=5 -race` 通过，没有 40P01 | **通过。** T12 `5c15156e`：本 Task 的五个测试 25 次顶层、60 个子测试；`raceall` 的 37 个测试 185 次顶层、815 个子测试。T13 `eb027a2c`：`raceall` 39 个测试（M3 的 30 个、P1a 的 9 个）195 次顶层、835 个子测试。T12 的修复轮 `c4c10107`：七个测试 35 次顶层、190 个子测试。整分支评审在 `25f5b9fb` 的副本上：39 个测试 195 次顶层、835 个子测试，203.6 秒。每一次都没有 40P01、死锁、DATA RACE、FAIL。修复轮的 `9e9db923` 改了其中一个测试（M2：`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects` 改用 `holding`），之后它在 `make test`（不带 `-race`）里通过；修复轮的报告没有记 `-race` 的重跑，控制者 2026-10-18 在 `dc2b17e2`（代码与 `91e54133` 相同）上补跑：`go -C server test -race -count=5 -v -run '^TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects$' ./internal/bootstrap/`，5 次顶层通过，`ok`（3.547 秒），没有 DATA RACE、40P01、FAIL |
| 3 | 看不到项目、看得到而不能写的写不取锁；每个工作项的写在 `issueWrites` 里有一行（T10.8） | **通过。** `TestEachIssueWriteLocksItsProjectFirst`（T10 `37dc7899`，`62feefdc` 加强）：另一个事务以 `EXCLUSIVE` 持着写读写的每张表（`users` 除外）时，olga 照样得到 404，walt、wanda 照样得到 403；测试证明的是"不取行锁、不写"（措辞在 `91e54133` 改准，M5）。T10.8 在完整性核对上失败（T10 的报告）；整分支评审：T8.1、T10.1、T10.2、T10F.1、T10F.2、T12.1 在这个测试上失败 |
| 4 | 本 Phase 的矩阵格子通过（`createIssue` 七行） | **通过。** `permission_matrix_issues_test.go`：T9 `f26ea437` 六行，T12 `5c15156e` 加"the triage state"一行（12 格通过）。`TestPermissionMatrix` 在 HEAD 的 `make test` 里通过。预检的重放数到 941 格（`createIssue` 七行 75 格）；之后没有一个修复轮的提交碰矩阵的文件（`git show --stat`） |
| 5 | 删除在用的状态 409 `project.state_in_use`；删除标签从工作项上摘下它和它的子标签 | **通过。** T11 `7e93b59b`：`TestDeletingAStateAWorkItemIsInIsRefused`、`TestDeletingALabelTakesItOffTheWorkItems`；存储的 `TestStateInUse`、`TestDetachLabels`（`556e045e` 加强）；交错 6、7 的创建一支（T13 `eb027a2c`）。整分支评审：T11.1、T11.7、T11F.1 被发现。拒绝的说明和文案原来给了一个做不到的补救（I1），`9e9db923` 改为中性的措辞 |
| 6 | 迁移 `00015`–`00019` 升、降、再升；`issue` 的 `apitest.Main` 两个方向 | **通过。** `make test` 的 `server/migrations`（`TestMigrationsGoUpDownAndUpAgain`，19 个迁移）和 `issue/adapter/http`（`apitest.Main(m, "issue")`）都是 `ok`（整分支评审的门禁表；HEAD 上 `make test` 46 个 `ok`） |
| 7 | `make gen-check`、`lint-go`、`test`、`lint-web`、`knip`、`test-web`、`e2e` 通过 | **通过。** 修复轮之后在工作树上（`final-fix-report.md`）：`make gen` 之后只有三个生成物变，各只差 `state_in_use` 的那句；`lint-go` 两段 `0 issues.`；`make test` 46 个 `ok`；`lint-web`（关键词守卫 72 条规则、23 个例外、没有命中；turbo 54/54）、`knip`、`test-web`（17/17）通过；`make e2e` 119/121（只有 A4 的两个 firefox 测试；S3 在工作树上通过）；两个提交之后 `make gen-check` 通过，树干净。`server/go.mod`、`go.sum`、`server/tools`、`pnpm-lock.yaml` 自 `d693f539` 没有变化 |

持续集成：分支在执行中没有推送（每个 Task 的报告都写"Not pushed"；本地没有 `origin/worktree-m4-p1`）。推送之后和合并之后的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 拆分的八条（第 1 节）。spec 第 3 节：第 4 条（`DeleteWorkspaceIssues` 不带工作区的 id，P8 加）、第 5 条（端口只带 P1a 读的东西）、第 6 条（`PublicProfiles` 在 P1a 接上）、第 7 条（契约只有 `IssueDetail`）、第 12 条（夹具在 `furnish.ts`、`issues.ts`、`assert/catalog.ts`，W3 拆成两个文件）照建议接受；第 8–10、13–16 条确认。设计第 2 节原来的夹具写法由 spec 取代，`91e54133` 把设计改成与代码一致。
- 预检的 L1–L6 照原文接受：L1 `delete_rows.sql` 的说明（等锁期间被删除的行被跳过、取下一行）；L2 M3 设计 3.3 补一句；L3、L4、L6 是 spec 第 5 节的 P2、组合、P3 三行；L5 是第 3 节第 1 条。都是文字，预检逐条核对过；错了的代价是一句注释或交接。
- 变异在 `$M4TMP/p1sdd/` 里没有 git 的副本上跑：保留架构子任务的证据，不让 git 进副本；错了只是工具的路径。

**执行中**（补上缺口的，都先由变异证明缺口存在，补上之后再证明测试咬得住）
- **Task 1**：评审通过。⚠️ 由控制者解决：Task 2 把锁住的项目 id 传进 `DeleteWorkspaceProjects`，`cascadeCalls` 核对它。两个 Minor 搁置到整分支评审：spec 2.3 的"两条例外"（实为一条例外、命中两处）；`invitations.sql:74` 的"the cascade"（成为 M3）。
- **Task 2（评审 Important 1 条）**：`project/app/cascade.go` 的说明说每个方法只用 `project` 自己的仓储，Task 2 起 `DeleteWorkspaceProjects` 还调 `IssueCascade`。当轮改正（`5c0114b6`，只改注释）；plan 的块保留原样，后面没有块锚在这句上。理由：每句注释在它的 Task 结束时都要对，这句会进 main。同一个 Task 里控制者裁定：容器起不来造成的失败不算发现（T2.10 的组合一层出过一次，重跑之后照 A.3 留下）。
- **Task 3（评审 Important 1 条）**：`fakeRows` 总是最大的 `id` 在前回答，下一批从"回答的第一个 `id`"之后开始的变异（T3F.1）在两个单元测试上都留下，而 A.3 让单元测试是这个键集性质唯一的守卫。假实现改为轮流按两种次序回答（`5956467e`），T3F.1、T3F.2 都被发现。plan 的 T3.6 一行写了一个不会失败的测试名，搁置（`91e54133` 改正）。
- **Task 4**：评审通过。Minor (a)：分批语句的 `FOR NO KEY UPDATE` 没有测试钉住，评审写的两个连接的探针能杀掉去掉它的变异。控制者裁定把探针做成存储测试（`516961f9`，`TestABatchPassesOverARowDeletedWhileItWaits`，五张表各一个子测试）：语句自己的注释声称这个行为，P9 的任务会在项目的锁之外调这些语句，所以就地钉住，不交出去。T4F.1–T4F.5 修复之前都留下，之后都被发现。没有读者的 `issueAlice` 一并删去。spec A.3 中 T4.1 的一句搁置（`91e54133` 改正）。
- **Task 5**：评审通过，四条 Minor 一轮落实（`604d4e66`）：
  - 恰好 101 个负责人或标签没有测试（T5P.2）；
  - "不用默认负责人"一行的默认负责人是访客，可指派的默认负责人被加在请求的负责人旁边的变异留下（T5P.1）；
  - `VerbDeleted` 在 P1a 没有用者，删去；
  - **给出的空优先级 `""` 被当作没给，而契约的枚举不收它**：在根上改，`NewIssue.Priority` 改为 `*Priority`（与 `StateID` 同一个"没给"的约定），`checkPriority` 拒绝 `""`。plan 中建、读 `NewIssue.Priority` 的块（Task 5、8、9）随之修订，HTTP 加一个 `"priority": ""` 答 422 的测试。实现者在副本上证明修订后的 Task 6–15 都能应用、Task 9 的树上生成物与钉值相同、`issue` 的测试通过；复审另外重放了修订后的 Task 6–9。理由：枚举由领域检查，一个"没给"的约定好过两个。错了的代价：一个字段的类型和几处测试的字面值改回去。
  - ⚠️ 留到 Task 8 再查：T5P.1 经用例到不了，只因为用例只读所问账户的角色。Task 8 的报告证实（`create_issue.go` 的 `check`，由 `TestCreateIssueDefaults/assignees_asked_for` 钉住）。
- **Task 6**：评审通过，Minor (a)、(b) 当轮落实（`11708a08`）：`IssueLabels`、`ActiveRoles` 的 `= ANY` 没有钉住（数据里没有没被请求的未删除标签、有效成员，两个 `OR true` 的变异都留下），Web 加没被请求的 Feature 和 frank；`issue_ports_test.go` 复制了 `waits` 的 NOWAIT/55P03 逻辑，改为一个 `rowWaits`，`waits` 包它，13 个调用者不变；`ids` 换成包里已有的 `jsonOf`。(c) 预言式的注释、(d) spec 2.6 的探针写法搁置。
- **Task 7**：
  - 实现者把 plan 的 `newLabel` 与 Task 4 的 `aLabel` 合并（移到 `store_test.go`），控制者接受：标准不许复制测试辅助，后面的 plan 仍能应用。
  - 实现者指出 `GetIssue` 的标签子查询有两个谓词没有钉住（T7P.1 读已删除的标签行；T7P.2 读每个工作项的标签，在 I1 及以下每一层都留下）。评审之前补上（`8f53cd70`）。理由：`getIssue` 的回答是 `createIssue` 和 P1b 的取、改都读的。
  - 评审的三条 Minor 当轮落实（`f6f53dee`）：`GreatestSortOrder` 的项目过滤（T7R.1）；"没有时为 NULL"（T7R.2），取消的读挪到删除之前；两处读文本行的循环合成 `textRows`。
- **Task 8**：
  - 实现者指出关注按 `id` 升序（约定五的后一半）没有测试：调用者 alice 总是最小的 `id`。评审之前补上（`3ba93508`：bob 在 alice 之前取，期望 `{bob, alice, carol}`）。理由：约定五是取锁约定写明的规则，单元测试是它在这里的钉子。
  - 实现者自审的 `308c1833`（`callLog.add` 复用 `line`，一句说明）接受，条件是后面的 Task 仍能应用（证实了）。
  - 评审的三条 Minor 当轮落实（`c82d5f2d`）：标签问题的下标（请求 `{other, bug}`，T8P.6）；`fakeRows` 嵌入 `callLog`；没有读者的假状态（`fakeAuthorizer.grant`、`fakeIssues.subscribers`）删去。
  - T8P.4（关注的去重没有钉住）搁置（成为 M1）。同一轮控制者裁定：从 Task 9 起，实现者也跑前面的"（Task n 起）"行，因为 T8 的一些组合一层到 Task 9–14 才有。
- **Task 9**：
  - 实现者指出 `issueDetail` 的最后修改者、时刻没有钉住（夹具里它们与创建的相同，T9Q.1、T9Q.2 留下）。评审之前补上（`a20ccf33`）。理由：P1b 的取、改经 `issueDetail` 回答。
  - ⚠️ 由控制者解决：设计 5.2 的 `bodyshape` 代价测试加字符串的数组排在 P2（设计 6.1 的表），不是 Task 9 的。
  - 评审的 Minor (a)、(b) 当轮落实（`bdbddf17`）：`parent_id`、`archived_at` 的映射（T9R.1、T9R.2）；一句说"账户不在了"的注释（v0 不删除账户）。
  - (c) `orNull` 和各模块的 HTTP 测试骨架搁置（第 7 节）。plan 的"五行"搁置（`91e54133` 改为六行）。
- **Task 10**：
  - 实现者自审的 `32fdd478` 改正两句注释：`reads` 是"锁下最先读、预检没有读的表"，不是"锁下最先读的表"（锁下的再判定先读成员关系）；`issueTables` 不含 `users`。
  - 评审的 Minor 当轮落实（`62feefdc`）：(a) 编号挪到检查之前（T8.8）时，编号的 `UPDATE` 自己就行锁项目，掩盖了锁的强度。写停在 `reads` 之后，测试核对它没有持任何表的 `ROW EXCLUSIVE`（`writtenTables`）；双重变异 T10F.1、T10F.2 修复之前留下，之后被发现。P1b 的行复用这个测试。(b) `users` 的说明补全。
  - spec、plan 同样宽松的措辞搁置（`91e54133` 改正，M5）。
- **Task 11**：评审的 Minor (a)、(b) 当轮落实（`556e045e`）：`TestDetachLabels` 改用包里的 `rowsAsText`、`audits`、`deletion`，比较全部 10 列；`TestStateInUse` 先问一个没归档的工作项（T11F.1 修复之前在存储一层留下）。(c) `project.state_in_use` 的补救说法（P1a 没有移动工作项的写，已归档的工作项是只读的）搁置到整分支评审，一次定下全部文案（成为 I1）。
- **Task 12（评审 Important 1 条）**：`issueWorld.serialize` 复制了 M3 的 `serializeOnWeb` 的七步门协议。提到 `interleaving_test.go` 的 `serializeOn`，两边都调它（`c4c10107`）；`heldTx` 的说明改准（它只开一个事务，第二个会 panic）。理由：标准要求 M3 的交错辅助被延伸、不被复制，P1b 的 3u–7u 要在它上面写。T12F.3 在 P1a 和 M3 两族都被发现。**T12F.5 裁定为接受的留存**（5.1）。M3 较早的交错测试仍手写门的几步，搁置：整分支评审把 `interleaving_writes_test.go` 交给 P1b；`interleaving_deletion_test.go` 实为 P1a 的，台账原来写错（成为 M2）。
- **Task 13**：评审的一条 Minor 当轮落实（`df324adb`）：`issue_interleaving_catalog_test.go` 重复了 `issue_cascade_test.go` 的"随删除、由删除者摘下"的谓词。提成 `issueWorld.workItemLabels` 和 `stateKept`，两个文件共用；T13F.9、T13F.11、T13F.12 表明交错 7 的先建一支钉住了时刻、写者、`updated_at` 三部分。
- **Task 14**：评审的五条 Minor 当轮落实（`fbefd9ad`）：`furnishIssueRows` 的说明（"两个账户关注"对五个调用者中的四个不对）；`tablesUnder` 的例子（nerve 没有 `issue_views` 表，改为 `(issues, issue_subscribers)`）；I1 内联了 P6 的 `stateIds`，于是有了新的 `e2e/fixtures/states.ts`（`createState` 从 `api.ts`、`listStates` 和 `stateIds` 从 P6 的两个文件移来，`api.ts` 由 403 行减到 386 行）；`furnishIssueRows` 不再回答没人读的值；一段注释挪到它说的那一行上。T14F.1 在 I1 和 P6 的页面版本都被发现。
- **Task 15（评审 Important 3 条，都是文字）**：M3 收尾交接的证据说法和"（完成）"、M1/P3 交接的 Phase、M3 设计 3.3 的 `ProjectCascade` 一条，连同五条 Minor，一轮改正（`25f5b9fb`）。理由：留着已知不对的句子只会多一轮。plan 的 doclinks 期望（Task 5 代码块里三处 `ptr[Priority]` 调用被读成链接）搁置（`91e54133` 改正）。

**整分支评审之后**
- 一个修复轮，照评审的 Findings、Stale sentences 和分诊表做 I1、M1–M8；I1 用评审给的中性措辞（第 4 节）。plan 的代码块保留原样，漂移列在 5.2。`interleaving_writes_test.go` 的门交给 P1b，写进 spec 第 5 节。错了的代价：文案和文档的文字。
- 更正：台账把 `interleaving_deletion_test.go:297` 记成 M3 的旧测试，它是 P1a 的 Task 1 加的，在修复轮里改（M2）。
- 本文的作者在同一个提交里改正设计第 12 节 P1a 任务 14 的一处（`api.ts` → `issues.ts`），与 D3、D7 同类；错了的代价是一个词。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 1、Minor 8。评审逐条核对过：
- 取锁协议：一条共用的路径（`Locks`），预检回答判定的每个拒绝，锁下的判定是最终的，时刻在最后一把锁之后，一次创建的每一行同一时刻、同一个写者；
- 逐个写的锁和次序（创建、删除项目、删除工作区的新顺序、删除状态、删除标签、M3 的成员关系收缩、停用、改地址、增长），没有等待环；全局的表顺序；约定五；键集分批不重读、不漏行；
- 预检第 3 节的死锁分析在 HEAD 上仍成立；
- 门禁、交错的竞争运行和 25 个变异（第 1 节）。

| 发现 | 处理 |
|---|---|
| **I1** `project.state_in_use` 的说明、契约、两份文案说"先把它们移到别的状态"。P1a 没有移动工作项的写（`updateIssue` 在 P1b）；已归档的工作项在恢复之前只读（设计第 2 节 I15）。对谁都做不到，违反"每句话对 HEAD 都对"和 spec 的清扫 30 | `9e9db923`。Go 的说明："Work items are in the state, archived ones too, so it cannot be deleted."；`project.yaml` 的 `deleteState` 一句；en："Work items are in this state, archived ones too, so it cannot be deleted."；zh-CN："这个状态下还有工作项（包括已归档的），不能删除。"；`state_writes_test.go` 的 `stateInUseJSON`；`make gen`（bundle 和 `schema.gen.ts` 只差这句）。spec 的清扫 30 写明为什么不给补救；补救交给 P2 的文案（spec 第 5 节的 P2 一行） |
| **M1** `create_issue.go:195` 的 `slices.Compact` 没有钉住：T8P.4 在整个 `issue/app` 留下（没有一行的调用者也是负责人） | `9e9db923`：`TestCreateIssueDefaults` 加"the caller among the assignees asked for"，期望的关注用 `slices.Compact` 求出。T8P.4 修复之前留下，之后由这一行发现 |
| **M2** `interleaving_deletion_test.go:286-295` 手写了 `holding` 的几步；这个测试是 P1a 新加的 | `9e9db923`：改用 `holding`（它另外加 NOWAIT，并核对锁住一行），保留 `holder.Commit(ctx)` |
| **M3** `invitations.sql:73-74` 说"the cascade takes them"，而锁现在在 `Execute` 里、`cascade()` 之前 | `9e9db923`：改为"the deletion"，`make gen` 复制进 `gen/invitations.sql.go` |
| **M4** 设计里被 P1a 弄过时的句子 D1–D8 | `91e54133`：D1（:248，"P1a 起时刻在两步锁之后读"）、D2（`cascade.go` 的行号）、D3（`furnishIssueRows` 的签名、文件、调用处）、D4（`catalog.ts` 的 `tablesUnder`）、D5、D6（`app.go` 的行号）、D7（夹具按资源分的文件）、D8（"不取行锁、不写"）；另改 :2053（一条例外、命中两处）和同类的 :103、:649、:1307、:1363、:1732 |
| **M5** spec、plan 说取锁形状的测试让写停在"锁下最先读的表"，而锁下最先读的是再判定的成员关系；"不出现在 `pg_locks` 里"多于测试证明的 | `91e54133`：spec 2.10、A.2、清扫 13 和 plan :8437 改为"锁下最先读、预检没有读的表（`states`）"，拒绝的写"不取行锁、不写"；设计 :260、:345、:2061 同改（D8） |
| **M6** spec 的其他文字：2.1 的计数和 Task 14 一行、2.3、2.6、2.11、A.3 的 T4.1、A.3 缺修复轮的变异 | `91e54133`：2.1 重新数（手写 140 个：新建 60、修改 80；生成物 14；文档 9），Task 14 一行加 `states.ts` 和 P6 的两个文件；2.3 一条例外、命中两处；2.6 `FOR SHARE` 等它、`FOR KEY SHARE` 不等；2.11 加 `deleteLabel` 的描述；A.3 的 T4.1 由核对每批按 `id` 升序回答的存储测试钉住；A.3 加修复轮按性质只在单元一层的 6 个 |
| **M7** plan 的说明不再对：GC 的文件长度、两处"两条例外"、:81、:113、:116、T3.6 一行、Task 15 Step 2 的 doclinks 期望、文件表缺只有修复轮碰过的文件 | `91e54133`：照改，另改同类的 :143 和 T2.5 一行；Task 15 Step 2 期望 plan 有 3 个"断的链接"（Task 5 代码块里的三处 `ptr[Priority]` 调用）；文件表加一行，列出只有修复轮碰过的五个文件；代码块没有动 |
| **M8** M3 收尾交接 :124 的未处理清单缺 `expectIssueDeleted`（P2）和删除断言的限时轮询（P9） | `91e54133`：补上 |

分诊表的其余各行：
- **接受、不改**：`workspace_deletion_test.go:222-223` 的标签一句（Task 3 改过，HEAD 上对）；`bootstrap/users.go:26`、`workspace_deletion_test.go:192` 的短注释行；迁移 00016–00018 第 2 行的注释（HEAD 上对）；Task 6、7、8 的预言式注释（`app/issue_ports.go`、`module.go`、`failures_test.go`、`issues.go`、`issues.sql`、`lock.go`、`ports.go`、`create_issue.go`，HEAD 上都对）；`orNull` 和各模块的 HTTP 测试骨架；T12F.5（第 7 节）。
- **交给 P1b**：`interleaving_writes_test.go:199-211` 手写的门（第 6 节）。
- **写进本文**：修复轮的变异（5.1），其中 T8P.5 按性质只在单元一层、已被发现。
- **评审考虑过、不算发现**：`issueTables` 不含 `users`（第 7 节）。

**修复轮**（`9e9db923`：11 个文件，代码、测试、契约、文案和 3 个生成物；`91e54133`：4 个文档）：
- 门禁见第 2 节第 7 行。T8P.4 修复之前留下，之后被发现（`$M4TMP/p1sdd/mut-ffw.sh`，单元一层，修复前后各一份没有 git 的副本）。
- 清单之外的改动，都与清单里的同类：设计 :103、:649、:1307、:1363、:1732；plan :143 和 T2.5 一行；spec A.1 指向 2.1 的一句、第 3 节第 12 条的"原写"、第 5 节的 P2 一行（`state_in_use` 的补救交给 P2）。
- 规则的失误一次：`make gen` 之后用 `;` 接了一个 `echo`，只读，没有影响。

**修复的范围复审**（opus）：I1、M1–M8、D1–D8 都已落实；清单之外的改动都对；spec 2.1 的重数对；没有新的问题；可以合并。遗留一处：设计 :2066（第 12 节 P1a 的任务 14）"`api.ts` 的建工作项"，建工作项的夹具在 `e2e/fixtures/issues.ts`。本提交改正。

## 5. 最终实现与 plan 的差异

plan 的代码块记录的是当时运行的文字，保留原样，本提交不改 plan。plan 自己改过三次：`c37a3480`（预检的 L1–L6，Task 1 之前）；`604d4e66`（Task 5、8、9 中建、读 `NewIssue.Priority` 的块随优先级的指针修订，Task 5 的块与提交的文件逐字节相同）；`91e54133`（M7 的说明文字，不碰代码块）。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：`9e9db923` 的 `project.state_in_use` 的说明、契约、文案（plan 的 Task 11 块里仍是旧的补救说法，全仓库只有那里还有它）和 `invitations.sql` 的一句注释；`5c0114b6` 的 `project/app/cascade.go` 的说明。`604d4e66` 的优先级随 plan 一起改，不是漂移。
- **测试**：Task 一层的修复轮、实现者自审的提交和整分支评审的修复轮（5.2）。
- **文档**：Task 15 的六个文件经 `25f5b9fb` 再改，M3 收尾交接经 `91e54133` 再改。
- **钉住的生成物**：plan 钉住的 SHA-256 中，Task 1 的 `gen/invitations.sql.go`、Task 11 的 `api/dist/openapi.yaml` 和 `schema.gen.ts` 与 HEAD 不同（`9e9db923` 改了它们的源）。
- **文件大小**（HEAD，`wc -l`）：分支改过的手写的代码、测试文件都在约 400 行以内，最大的是 `server/migrations/schema_test.go` 400 行、`bootstrap/permission_matrix_test.go` 392 行、`project_write_locks_test.go` 和 `interleaving_growth_test.go` 各 389 行、`e2e/fixtures/api.ts` 386 行。契约 `api/modules/project.yaml` 1,357 行（M3 的裁定 B1：契约文件不拆）。

### 5.1 修复轮的变异

这些变异由 Task 一层的修复轮、实现者自审和整分支评审的修复轮写下，定义在 `$M4TMP/p1sdd/` 的探针文件里（Task 6 的两个在评审的 `review-t6/`），不在 plan 的变异表（`mutants_p1a.py` 的 136 个）里。"被发现的层"是修复之后的；括号里写修复之前的结果。最后一列只在 spec 附录 A.3 或 Task 报告写明了理由的行填"是"。

| Task | id | 变异 | 被发现的层 | 守的文件 | 按性质只在单元一层 |
|---|---|---|---|---|---|
| 3 | T3F.1 | 下一批从这一批回答的第一个 `id` 之后开始，不是最大的 | 单元：`TestTheCascadeDeletesEachProjectsRowsInBatches`（`5956467e` 之前留下） | `issue/app/delete_project_rows.go` | 是：同 T3.4，语句自己跳过已删除的行，真实环境上只是扫得多（A.3） |
| 3 | T3F.2 | 下一批从最后一个 `id` 之后开始 | 单元：同上，另 `TestTheCascadeStopsAtAFailingBatch` | 同上 | 是：同上 |
| 4 | T4F.1–T4F.5 | 工作项、负责人、标签、关注、动态的分批语句各去掉子查询的 `FOR NO KEY UPDATE` | 存储：`TestABatchPassesOverARowDeletedWhileItWaits` 中各自那张表的子测试（`516961f9` 之前都留下） | `issue/adapter/postgres/queries/delete_rows.sql` | — |
| 5 | T5P.1 | 请求了负责人时，可指派的默认负责人也被加上 | 单元：领域的 `TestAssignees`（`604d4e66` 之前留下）；`issue/app` 留下 | `issue/domain/issue.go`（`Assignees`） | 是：经用例到不了，用例只读所问账户的角色（Task 8 的报告） |
| 5 | T5P.2 | 上限多一个：101 个负责人或标签照收 | 单元：领域的 `TestCheckNewIssueReportsEveryProblem`（`604d4e66` 之前留下） | `issue/domain/issue.go`（`CheckNewIssue`） | — |
| 5 | T5P.3 | 给出的空优先级当作没给 | 单元：领域的 `TestCheckNewIssueReportsEveryProblem`；`issue/app` 留下 | `issue/domain/issue.go`（`checkPriority`） | — |
| 6 | `labels-any-dropped` | `IssueLabels` 的 `id = ANY` 恒真 | 存储：`TestIssueCatalogs`（`11708a08` 之前留下）；只在这一层跑过 | `project/adapter/postgres/queries/issue_ports.sql` | — |
| 6 | `roles-any-dropped` | `ActiveRoles` 的 `member_id = ANY` 恒真 | 存储：同上 | 同上 | — |
| 7 | T7P.1 | `GetIssue` 读已删除的标签行 | 存储：`TestCreatingAWorkItemWritesItsRows`（`8f53cd70` 之前每一层都留下） | `issue/adapter/postgres/queries/issues.sql`（`GetIssue`） | — |
| 7 | T7P.2 | `GetIssue` 读每个工作项的标签 | 存储：同上（之前在存储、组合、I1 都留下） | 同上 | — |
| 7 | T7P.3 | `GetIssue` 读每个工作项的负责人（对照） | 存储：同上（修复前后都被发现） | 同上 | — |
| 7 | T7R.1 | `GreatestSortOrder` 的项目过滤恒真 | 存储：`TestGreatestSortOrder`（`f6f53dee` 之前留下） | `issues.sql`（`GreatestSortOrder`） | — |
| 7 | T7R.2 | 没有负责人、标签时回答空数组，不是 NULL | 存储：`TestAFailedIssueStatementIsAnError`（`f6f53dee` 之前留下） | `issues.sql`（`GetIssue`） | — |
| 8 | T8P.1 | 用例在所问账户之外也读默认负责人的角色 | 单元：`TestCreateIssueDefaults/assignees_asked_for` | `issue/app/create_issue.go` | — |
| 8 | T8P.2 | T8P.1 加 T5P.1：默认负责人在所问账户之外也被指派 | 单元：领域的 `TestAssignees`；`issue/app` | `create_issue.go`、`domain/issue.go` | — |
| 8 | T8P.3 | 预检在写的事务里做 | 单元：五个 `TestCreateIssue…` 测试 | `create_issue.go` | — |
| 8 | T8P.4 | 关注不先去重（去掉 `slices.Compact`） | 单元：`TestCreateIssueDefaults/the_caller_among_the_assignees_asked_for`（`9e9db923` 起；之前在整个 `issue/app` 留下，即 M1） | `create_issue.go` | 是：存储的 `ON CONFLICT DO NOTHING` 跳过第二行，组合出的 app 里等价，同 T7.3（A.3） |
| 8 | T8P.5 | 关注按调用者、负责人的次序写，不按 `id` | 单元：`TestCreateIssueWritesTheWorkItemAndItsRows`、`TestCreateIssueDefaults/the_default_assignee,_a_member`（`3ba93508` 之前留下） | `create_issue.go` | 是：同 T8.12，同一个项目的行在项目 N 之下没有第二个写者（A.3） |
| 8 | T8P.6 | 不认识的标签的问题按标签的 `id` 次序编下标，不按请求里的位置 | 单元：`TestCreateIssueRefusesUnderTheLock` 的分诊一行（`c82d5f2d` 之前留下） | `create_issue.go` | — |
| 9 | T9P.1 | 处理函数把给出的空优先级当作没给交下去 | 单元：HTTP 的 `TestCreateIssueTakesAnEmptyPriorityAsGiven` | `issue/adapter/http/issues.go` | — |
| 9 | T9P.2 | 处理函数丢掉请求的优先级（T9.1 在指针上的写法，T9.1 的旧文字在修订之后不在了） | 单元：HTTP 的 `TestCreateIssue`、`TestCreateIssueTakesAnEmptyPriorityAsGiven`；组合：矩阵的"each field given" | 同上 | — |
| 9 | T9Q.1 | 回答把创建者当作最后修改者 | 单元：HTTP 的 `TestCreateIssue`（`a20ccf33` 起）；矩阵留下 | 同上（`issueDetail`） | 是：P1a 只有创建，经接口造出的工作项这两列与创建的相同（A.3） |
| 9 | T9Q.2 | 回答把创建的时刻当作最后修改的 | 同上 | 同上 | 是：同上 |
| 9 | T9R.1 | 回答不给父工作项 | 单元：HTTP 的 `TestCreateIssue`（`bdbddf17` 起） | 同上 | — |
| 9 | T9R.2 | 回答不给归档的时刻 | 同上 | 同上 | — |
| 10 | T10F.1 | T8.8（编号在检查之前）加 T6.1（锁取 `FOR SHARE`） | 组合：`TestEachIssueWriteLocksItsProjectFirst` 的 `writtenTables`（`62feefdc` 之前留下） | `issue/app/create_issue.go` 的次序；`project/adapter/postgres/queries/issue_ports.sql`（`LockIssueProject`） | — |
| 10 | T10F.2 | T8.8 加 T12.1（锁换成普通的读） | 同上 | 同上 | — |
| 11 | T11F.1 | `StateInUse` 只数已归档的工作项 | 存储：`TestStateInUse`（`556e045e` 之前留下）；组合：`TestDeletingAStateAWorkItemIsInIsRefused`（修复前后都被发现） | `issue/adapter/postgres/queries/cascade.sql` | — |
| 12 | T12F.3 | 共用核心的探针等 `workspaces` 上的等待 | 组合：`TestTwoCreationsOnAProjectSerialize`，M3 的 `TestStateWritesOnOneProjectSerialize`、`TestLabelWritesOnOneProjectSerialize` | `bootstrap/interleaving_test.go`（`serializeOn`） | — |
| 12 | T12F.5 | 共用核心不做锁等待的核对（去掉 `WaitForLockWaitOn`） | **留下**（七个交错）；接受，裁定见表下 | 同上 | — |
| 13 | T13P.1 | `deleteState` 在它的锁之前问工作项 | 单元：`project/app`；组合：交错 6 的先建一支 | `project/app/delete_state.go` | — |
| 13 | T13P.2 | `createIssue` 在项目的锁之前检查依赖的行 | 单元：`issue/app`；组合：取锁形状，交错 6、7 的先删一支 | `issue/app/create_issue.go` | — |
| 13 | T13F.9 | T11.9 放到交错 7：删除标签以自己的时刻摘下 | 组合：交错 7 的先建一支（`df324adb` 起） | `project/app/delete_label.go` | — |
| 13 | T13F.11 | T11.11 放到交错 7：`DetachLabels` 保留行原来的写者 | 同上 | `issue/adapter/postgres/queries/cascade.sql` | — |
| 13 | T13F.12 | T11.12 放到交错 7：`DetachLabels` 的 `updated_at` 写数据库的时刻 | 同上 | 同上 | — |
| 14 | T14F.1 | `stateIds` 把名称映到状态所在项目的 id，不是状态的 id | 端到端：I1（422 `state_id`）、P6 的页面版本（等不到 `PATCH` 的回答） | `e2e/fixtures/states.ts` | — |

一共 40 个：39 个至少在一层被发现，T12F.5 是接受的留存。另有对照 T12F.5B：在修复之前的树（`5c15156e`）上删去两份副本各自的探针，七个交错同样都通过，所以 T12F.5 早于这次提取。整分支评审在 `25f5b9fb` 的副本上重跑了其中 10 个（T3F.1、T4F.1、T4F.5、T5P.3、T8P.4、T8P.5、T10F.1、T10F.2、T11F.1、T12F.3），结果与上表相同，只是 T8P.4 那时还留下（M1）。

**T12F.5 的裁定**（控制者，台账 Task 12）：接受的留存。它改的是测试自己判断前提的探针，不是被测的代码。去掉锁的生产代码变异在探针在时都被发现（T12.1 在取锁形状、交错 8、交错 4/5 三处），探针自己的两种失败方式由 T12F.3（等错了表）和 T12.2（门在写之前扣住）钉住。裁定错了的代价：要给这个辅助函数写一个元测试，那是过度设计。

### 5.2 plan 与 HEAD 的差异

plan 的代码块照运行时的文字保留。下表是动过 plan 的块所写的文件的提交，读者从这里找到漂移。文件取自 `git show --stat`；路径省略 `server/internal/modules/` 和 `server/internal/`。

| 提交 | Task | 改的文件 | plan 的块 |
|---|---|---|---|
| `5c0114b6` | 2（评审 Important） | `project/app/cascade.go`（说明） | Task 2 的块是旧的说明 |
| `5956467e` | 3（评审 Important） | `issue/app/cascade_test.go`（`fakeRows` 轮流两种次序） | Task 3 的块；Task 8、11 锚在这个文件的别处，照样应用 |
| `516961f9` | 4 | `issue/adapter/postgres/delete_rows_test.go`（新测试、`deletion`）；`server/migrations/issue_schema_test.go`（删去 `issueAlice`） | Task 4 的块 |
| `604d4e66` | 5 | `issue/domain/activity.go`、`issue.go`、`issue_test.go`；plan | plan 随之修订：Task 5 的块与提交的文件逐字节相同，Task 8、9 中建、读 `NewIssue.Priority` 的块改为指针；`0baf7879`、`f26ea437` 照修订后的块应用，不是漂移 |
| `11708a08` | 6 | `project/adapter/postgres/demote_test.go`（`rowWaits`）、`issue_ports_test.go` | Task 6 的块；`demote_test.go` 只有修复轮碰过 |
| `c7798d12` | 7（Task 的提交里实现者自审的合并） | `issue/adapter/postgres/issues_test.go`（少了 `newLabel`）、`store_test.go`（多了它）、`delete_rows_test.go`（`aLabel` 包它）；其余文件与块相同 | Task 4、7 的块 |
| `8f53cd70` | 7 | `issue/adapter/postgres/issues_test.go` | Task 7 的块 |
| `f6f53dee` | 7 | 同上（`textRows`） | 同上 |
| `308c1833` | 8（实现者自审） | `issue/app/create_issue_test.go`、`fakes_test.go` | Task 8 的块 |
| `3ba93508` | 8 | `issue/app/create_issue_test.go` | 同上 |
| `c82d5f2d` | 8 | `issue/app/cascade_test.go`、`create_issue_test.go`、`fakes_test.go` | Task 3、8 的块；Task 11 在 `cascade_test.go` 上的两个旧块照样应用 |
| `a20ccf33` | 9 | `issue/adapter/http/issues_test.go` | Task 9 的块 |
| `bdbddf17` | 9 | 同上 | 同上 |
| `32fdd478` | 10（实现者自审） | `bootstrap/issue_write_locks_test.go`（四段注释） | Task 10 的块 |
| `62feefdc` | 10 | 同上（`writtenTables`） | 同上 |
| `556e045e` | 11 | `issue/adapter/postgres/cascade_test.go` | Task 11 的块 |
| `c4c10107` | 12（评审 Important） | `bootstrap/interleaving_test.go`（`serializeOn`）、`interleaving_states_test.go`、`issue_interleaving_test.go` | Task 11、12 的块；`interleaving_test.go` 只有修复轮碰过 |
| `df324adb` | 13 | `bootstrap/issue_cascade_test.go`、`issue_interleaving_catalog_test.go` | Task 11、13 的块 |
| `fbefd9ad` | 14 | `e2e/fixtures/api.ts`、`assert/catalog.ts`、`assert/workspace.ts`、`furnish.ts`、`states.ts`（新建）；`e2e/stories/issue/i1-create-issue.spec.ts`、`project/p6-states-page.spec.ts`、`project/p6-states.spec.ts` | Task 14 的块；`states.ts` 和 P6 的两个文件只有修复轮碰过 |
| `25f5b9fb` | 15（评审 Important 3 条） | `docs/v0/M3-workspace-project/M3-design.md`；`docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md`、`M1-P3-trim-platform.md`、`M3-closeout.md`；`docs/v0/plane-diff.md`；`docs/v0/v0-design.md` | Task 15 的块（六个文件都是它的目标） |
| `9e9db923` | 整分支评审的修复轮 | I1：`project/domain/errors.go`、`api/modules/project.yaml`、`web/packages/i18n/src/locales/en/errors.json`、`zh-CN/errors.json`、`project/adapter/http/state_writes_test.go`；M1：`issue/app/create_issue_test.go`；M2：`bootstrap/interleaving_deletion_test.go`；M3：`workspace/adapter/postgres/queries/invitations.sql`；生成的 `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`workspace/adapter/postgres/gen/invitations.sql.go` | Task 1（M2、M3）、Task 8（M1）、Task 11（I1）的块；Task 11 的块里仍是旧的补救说法，钉住的三个生成物的 SHA-256 不再相同 |
| `91e54133` | 整分支评审的修复轮 | `docs/v0/M4-issue-core/M4-design.md`、`handoffs/M3-closeout.md`、`plans/P1a-issues.md`（说明文字）、`specs/P1a-issues.md` | Task 15 在 `M3-closeout.md` 上的块；plan 的代码块没有动 |

## 6. 移交事项

同一个里程碑之内的后续 Phase（写进它们的 spec；spec 第 5 节各行照录，下面另加执行中交出的）：
- **P1b**（spec 第 5 节"交给 P1b"一表）：
  - 取锁的一步：`issue/app/lock.go` 的 `Locks`、`write`、`precheck`、`lockAndDecide`。P1b 让 `lockAndDecide` 带上锁下读到的工作项；按资源寻址的写在这条路径上加"读工作项（不加锁）得到项目、锁下重读确认仍在这个项目"，不另写一份。写闸门插在预检之后、事务之前。
  - `server_busy`：写闸门和 `lock_timeout` 的 503 随闸门在每个工作项的写上声明。
  - 端口：`ProjectGate.Find`、`Lock` 照用；`IssueCreator.GetIssue` 已回答带负责人、标签的工作项；修改要的存储方法、差集写法在 P1b 加。领域：创建一侧的规则照用，修改的规则和动态的第 3、6、7、10–13 行在 P1b。契约：`IssueDetail` 照用，`IssueUpdate` 在 P1b。
  - 组合：`issueWrites` 每个工作项的写一行，P1b 的写不加行时 `TestEachIssueWriteLocksItsProjectFirst` 失败。每行的 `reads` 是写在锁下最先读、预检没有读的表，而且要在写自己的任何写之前读（`updateIssue` 在锁之前读 `issues`，它的 `reads` 不能是 `issues`；Task 10 的报告）。`62feefdc` 的 `writtenTables` 核对停下的写没有持任何表的 `ROW EXCLUSIVE`，P1b 的行同样受它约束。`writesOnAProject` 按标签 `issue` 去掉工作项的写。矩阵在 `permission_matrix_issues_test.go`。
  - 交错：`heldTx`、`exchange`、`racingWrite`、`serialize`（经 `serializeOn`）、`inBothOrders`、`createsIssue`、`deletedWith`。交错 3u–7u 照它们写（修改代替创建），交错 20 另经真实的连接池和闸门。`heldTx` 每个写只开一个事务，第二个会 panic（`c4c10107` 的说明）；Task 12 的评审核对了每个 `WithinTx`，认为 P1b 的 `updateIssue` 也只开一个顶层事务（台账）；P1b 的写若嵌套 `WithinTx`，`heldTx` 要在 context 已带事务时跳过门。
  - 交错的门（整分支评审的分诊，`91e54133` 写进 spec 第 5 节）：`interleaving_writes_test.go` 的 `TestTwoWritesOnAProjectSerialize`（:199-211）仍手写门的几步；P1b 的 3u–7u 改到这个文件时，把它们收进 `serializeOn`。
  - `issueDetail` 的最后修改者、时刻、父工作项、归档时刻今天只在 HTTP 的单元测试里分得出（T9Q.1–2、T9R.1–2）。P1b 的修改让 `updated_*` 与创建的不同之后，组合一层也看得到。
  - 规模：同步的删除（`deleteProjectRows`，每批 1,000 行）在 2 万个工作项（每个 20 条动态）的项目和五个这样的项目的工作区上量请求的耗时，记进 P1b 的 spec。
  - 端到端：I2 的接口版本照 `expectIssueCreated` 的写法加 `expectIssueUpdated`。关键词守卫的 `phase` 照旧 `M4/P1`。
- **P2**：
  - 访客规则：判定的 `Grant` 重新由 `decide`、`precheck`、`lockAndDecide` 带出（`OwnOnly` 的拒绝也在预检里答）；`OwnOnly` 由 `access` 的 `ProjectFacts` 算出，`ProjectGate.Lock` 要不要 `guest_view_all_features` 由 P2 定。起止日期进接口；按编号取（`identifier`）。
  - **`project.state_in_use` 的补救**：只有 P1b 的修改和 P2 的恢复都有了才做得到。交给 P2 的文案一并定（例如"先把它们移到别的状态，已归档的先恢复"），说明、契约、两份文案同改。
  - `expectIssueDeleted`：照 `expectProjectDeleted` 从目录自查、沿外键图递归（M3 收尾交接的未处理清单）。
  - 有了 `deleteIssue` 之后，按目录的删除测试的种子加一个在项目之前已删除的工作项：分批语句的 `deleted_at IS NULL` 今天只由存储测试钉住（预检的 PF.3）。
  - `bodyshape` 的代价测试加字符串的数组（设计 6.1 的表；Task 9 的评审确认在 P2）。
- **P3**：`LockMany`（约定七）、`DeletedAt`；关联进删除的语句；加关联与删除工作区的交错（两个项目同属一个工作区，两种顺序）：`LockWorkspaceProjects` 的 `id` 升序从这时起承重（P1a 只由存储测试钉住，预检的 PF.1、L6）。
- **P5**：`Issue`（列表的行）；状态的名称进 `StateCatalog`。
- **P6**：`archive_in`。
- **P8**：`DeleteWorkspaceIssues` 加工作区的 id（没有项目的草稿）；草稿的写和发布草稿先取工作区 S，不是 `issueWrites` 的形状，从 `writesOnIssues` 分出，另写它们的形状测试。
- **P9**：`DeleteProjectIssues`、`DeleteWorkspaceIssues` 换成事件和任务，删去这两个方法；删除断言对工作项一侧的表的限时轮询（M3 收尾交接的未处理清单）。
- **P14**：I1 的页面版本。
- **收尾**：3.23 中 P1a 的行由 Task 15 写好，收尾逐行核对。

**交接的处理结果**：P1a 在三份交接里写下"处理结果（M4/P1a）"：[M1/P2 交接](../handoffs/M1-P2-trim-content.md)、[M1/P3 交接](../handoffs/M1-P3-trim-platform.md)、[M3 收尾交接](../handoffs/M3-closeout.md)（Task 15 `e6532863`，`25f5b9fb`、`91e54133` 改准）。三份的 `status` 都照旧是 `open`，在 M4 收尾随 review 关闭（M3 的做法）；各自仍未处理的条目写在它们的"处理结果"末尾。

## 7. 已知限制

- **授权与时刻的先后可以不一致**（B 的代价一，设计 3.5，spec 第 7 节）：只改工作区角色的写不锁项目，降为访客、结束成员关系的连带在锁项目之前读时刻。写在总体设计 4.2。
- **拒绝的回答在预检时定下**（spec 第 7 节）：预检拒绝的请求在那一刻排定先后，锁下不再看。
- **改地址与创建会等一次**：`users set-email` 的 `FOR UPDATE` 与创建的外键 `FOR KEY SHARE` 冲突；改地址只持这一行，不等项目一侧的任何锁，所以没有环（设计 3.5，整分支评审的表）。
- **删除项目、工作区的工作项一侧是同步的**：在请求的事务里分批软删除，规模目标上的耗时还没有量（P1b 量，P9 换成任务）。
- **只在单元、存储一层的性质**：spec 附录 A.3 逐条写明理由（按性质只在单元一层 20 个，只在结构、存储一层 28 个，另加修复轮的 6 个）。其中 `LockWorkspaceProjects` 的 `id` 升序在 P3 之前只由存储测试钉住。
- **T12F.5 是接受的留存**（5.1 的裁定）：共用的 `serializeOn` 去掉锁等待的核对时，七个交错都通过。
- **`orNull` 和各模块的 HTTP 测试骨架**：`issue/adapter/http` 照 M2、M3 以来各模块的写法，有自己的 `orNull`、`newServer`、`do`、`request`、`fakeAuth`。把它们提到 `platform/httpserver` 是跨模块的重构，不是 P1a 的（整分支评审的分诊：接受、不改）。
- **外观的行宽**：`bootstrap/users.go:26`、`workspace_deletion_test.go:192` 的短注释行（plan 原文），`api/modules/workspace.yaml` 中 `deleteWorkspace` 描述里较长的一行（折叠的标量，渲染不受影响）。接受、不改。
- **`issueTables` 不含 `users`**：取锁形状的测试里，另一个事务以 `EXCLUSIVE` 持的表不含 `users`（创建不加锁地读它，只经外键取 `FOR KEY SHARE`）。拒绝的路径不写、不锁 `users`，所以没有加（整分支评审考虑过，不算发现）。
