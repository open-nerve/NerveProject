# M4/P3 父子与关联：评审记录

| 项 | 内容 |
|---|---|
| Phase | M4/P3 `issue-hierarchy` |
| 日期 | 2026-10-24 |
| 结论 | **通过**：16 个 Task 逐个实现、逐个评审，都通过。Task 评审的 Important 共 17 条（T1 2 条、T3 2 条、T4 2 条、T5 1 条、T6 2 条、T10、T11、T12、T13 各 1 条、T14 2 条、T16 2 条），都出在 plan 写定的代码、测试或文档里，都在当轮改掉；T2 的评审要改的只是记录（plan 写定的表里一行的层），T7、T8、T9、T15 的评审通过，只有 Minor。整分支评审 Critical 0、Important 1（I1）、Minor 2，另有分诊的九项；I1 是设计层面的访客规则，报负责人，负责人裁定 A（2026-10-24）；修复轮一轮（两个提交）落实 I1、两条 Minor 和分诊 1–5；范围复审（opus）：八条裁定都已落实、对 HEAD 都对，新的问题 0 条，四条观察不需要改，"可以写评审记录、合并" |
| spec / plan | [spec](../specs/P3-issue-hierarchy.md) / [plan](../plans/P3-issue-hierarchy.md) |
| 设计 | [M4 设计](../M4-design.md) 第 12 节 P3；本 Phase 改过的设计各节见 spec 2.1、附录 A.2 和第 5 节 |
| 分支 | `worktree-m4-p3`，从 `main` 的 `f8696490`（P2 的合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`c13249f6`，2026-10-21，`$M4TMP/p3-architect-report.md`）：在 `$M4TMP/p3proto` 上逐 Task 做出原型、存快照（`p3snap/T0`…`T16`），plan 的块由相邻两份快照的差异生成。设计 12 节 P3 的十一个草稿任务切成 16 个 Task（原型先做成 14 个，两个超出约 1,500 行，拆过两次，spec 第 3 节第 1 条），最大的 Task 12 1,531 行、Task 3 1,511 行，没有停下；plan 17,482 行，spec 471 行。从基点逐 Task 重放 673 个块，照原样跑 131 个 `Run:`（2,330 秒，没有重跑），最后的树与原型差的只有提交的三份文档（3,474 对 3,472 个文件）；186 个变异都被发现，没有等价的；76 个交错、竞争、取锁形状、连接池和写闸门的测试 `-race -count=5` 380 次顶层通过，没有 40P01；矩阵 1,455 个叶子。架构子任务在 T12 被一次断线（ECONNRESET）打断，控制者恢复同一个子任务接着做。控制者的裁定（`$M4TMP/p3-architect-rulings.md`）：spec 第 3 节 1–23 条照建议，第 5、7、10、18 条带条件，第 20 条以预检为准（第 3 节）。
- **执行前的预检**（opus，只读，`$M4TMP/p3-preflight/preflight.md`）：High 0、Medium 3、Low 11，另有六条记录。独立重放 673 个块、131 个 `Run:`（2,360 秒），每个 Task 的树与快照相同；76 个测试 `-race -count=5` 380 次顶层通过（551 秒）；它自己的 44 个变异；在目标规模上量了子树的删除（每个项目 2 万个工作项、51 万行，整个项目挂在一个工作项下：请求平铺 2.58 秒、一条链 2.83 秒，同样的数据上删除项目 3.04、3.18 秒），不分批。M1：只看自己的访客的工作项的回答照存的写出看不到的父工作项的 `parent_id`；M2：加子工作项"在 N 之下读祖先链"没有组合一层的钉子；M3：裁定 10 的前提不存在（`issue.read` 要项目的有效成员关系，没有人不是成员而读公开项目的工作项）。
- **修订**（`650dc819`，2026-10-21）：架构的四个条件和预检的裁定（`$M4TMP/p3-preflight-rulings.md`：M1–M3、L1–L11、N1–N6）写进原型、快照、plan、spec 和设计（设计 25 处）；spec 的"待裁定"都换成裁定（487 行），plan 17,910 行、680 个块。变异 192 个（整张表在修订之后的快照上重跑），每一层都被发现；完整重放：680 个块、132 个 `Run:`（2,401 秒，没有重跑）；81 个测试 405 次顶层通过（584 秒），`platform/gate` 13 秒。修订的 Pass C 跑到 T8.10 时网络断开（ECONNRESET），负责人重启了客户端；会话在主检出里重新开始，控制者回到 m4-p3 的 worktree，恢复同一个架构子任务，它从 T8.10 续跑（`amend-passC-rest.sh`），已做完的没有重做。它的 D1–D11（裁定没有预见的决定）由控制者接受（第 3 节）。
- **执行**（2026-10-21 至 2026-10-23）：
  - 16 个 Task 各由一个实现子任务完成（不指定模型，即 Opus 5.5），用 `planapply.mjs` 应用 plan 的块；各由一个评审子任务（opus）检查是否符合 spec 和质量。每个 Task 都有评审之前的修改：派发带上的（常设的裁定和前面 Task 带过来的条目：Task 5、7、9–14、16），或控制者对实现者交回的报告的裁定（Task 1–8、10、12–16）。评审要改的在修复轮改，修复轮恢复同一个实现者（19 轮）；修复轮之后有范围复审：sonnet 12 次（Task 1、3、4、5、6、9、10、11、12、13、14、15），opus 1 次（Task 8：锁的形状靠计划）；只改文档或记录、或裁定由控制者核对的六轮（Task 2、7 的修复轮 1，Task 8、9、10 的修复轮 2，Task 16 的修复轮 1）由控制者自己核对差异，不再复审；控制者另有两个只改文档的提交（`f4b34b97`、`74182225`，复审的新发现）。同一时刻只有一个写入者。执行 60 个提交（`dbba1951`…`f8e4c0d7`，含控制者的两个）。Task 1 的实现者在评审之前的修改开始之前被一次断线打断（树在 `dbba1951` 干净），控制者恢复同一个子任务。
  - **plan 是"执行时的 plan"**（P1b 的裁定，从 Task 1 起）：每个修复轮在同一个提交里重新生成它的 Task 的块（`taskblocks.py`、`compose_plan.py`），执行中加的变异行经 `mutants_p3*.py` 和结果的覆盖文件进 plan 的表；块数从 680 到 807（第 5 节）。重新组装时 plan 头写的大小由一个脚本保持并核对：Task 5 起 `keepsizes-t5pf.py`，Task 11 的修复轮 1 起 `keepsizes-t11pf.py`，它核对头里说到的每个大小（三个契约、`keywords.json`、三个例外和它们的基点、改过的最长的、新建的最长的）对执行之后的最终的树，对不上就失败。
  - 实现者在每个 Task 里按 plan 的变异表跑自己的行，在 `$M4TMP/p3sdd/` 里没有 git 的副本上跑（`mut.py`：单元、存储、模式、组合、端到端、守卫各层）。标"（Task n 起）"的行由 Task n 的实现者在自己的树上重跑（`MUT_LATER=n`）：Task 3 的 T2.1、T2.2，Task 4 的 T2.4，Task 6 的 29 行、31 层（含 T5.26 在 M2 的交错），Task 11 的 42 行、43 层，Task 12 的 T4.12，Task 14 的 34 行、39 层（修复之后加 T13.33，35 行、40 层），Task 15 的 40 行、42 个端到端的层（评审之前的修改之后 41 行，修复轮之后 I8、I15 的 15 行重跑），都被发现。端到端的发现由 `e2ekills.py` 核对是断言失败（Task 15：42 个、0 个可疑）。容器起不来不算发现（`mut.py` 的 INFRA）：T1.9 的组合一层两次 INFRA（Ryuk 的端口），之后被发现；T3.11 的组合一层一次；T10.18 一次；T10.22 两次（"invalid port"），Docker 空闲时重跑被发现。
  - **与 P2 不同的做法**：P2 把执行中加的变异行放进 plan 的表，spec 附录 A.3 留作原型的记录，由评审记录数它的过时的数。P3 同样把行放进 plan 的表，另外让 A.3 在每轮之后都对：只在一层被发现的行在加它的那一轮写进 A.3，带真的理由（Task 10 的修复轮 2 补了之前各轮的，Task 11 起每轮写自己的；裁定在第 3 节）。整分支评审的逐行核对在 A.3 之外找到七行，修复轮都处理了（第 4 节）；HEAD 上 A.3 与 plan 的表一致（第 5 节）。
  - **执行中改了行为的生产代码只有整分支评审之后的修复轮**：I1（负责人裁定 A）和两句拒绝的文字；其余是错误的文字、行为不变的重构、说明，和一条语句的写法（同样的行，读法不同）（第 5 节）。
  - **规则的失误**（违反书面规则的调用）：都没有改动 worktree 或任何提交：git 命令都是只读的，被拦下的没有运行，写了文件的只写 scratch。
    - 实现者 52 次（台账和各 Task 的报告逐条记下的）：Task 1 2 次、Task 2 1 次、Task 4 6 次、Task 6、7、8 各 1 次、Task 9 3 次、Task 10 4 次、Task 11 7 次、Task 12 10 次、Task 13 4 次、Task 14 3 次、Task 15 4 次、Task 16 5 次；Task 3、5 和整分支评审之后的修复轮没有。按类：连起来的 git 命令 9 次（一次被拦下）、`git -C` 3 次、`cd` 2 次、行内的 `python3 -c` 6 次、行内的循环 13 次（两次被拦下）、路径里的 shell 变量 2 次、zsh 的 `=` 开头的 `echo` 8 次（zsh 拒绝执行）、`wait-for.sh` 不带参数或接管道 3 次，另有被拦下的 `Monitor` 循环、`$(grep -l …)` 的列表、`diff && cmp`、没先跑 `wait-idle.sh` 的一次 `go test`、被拦下的 `docker ps … | sort` 管道、被拦下的写完就运行的一个调用各 1 次。
    - 评审者约 26 次：Task 1 的复审 `git -C` 4 次和路径带 shell 变量的行内循环；Task 3 的复审一次写错路径（在 `/private/tmp/claude-501` 下建了一个目录，随即删掉）；Task 5 的评审一个行内的 `for`、复审一次 `cd`；Task 6 的复审在 `/private/tmp/claude-501` 下留了一个文件，随即删掉；Task 8 的评审约 6 次连起来的非 git 命令、起了又删了一个容器 `m4p3-rv-t8`，复审为点名的疑点在两个会话里核对（`pgrowlocks`、持着的删除），容器删掉；Task 9 的评审几次连起来的非 git 命令，复审一次 `git -C status`、一个随即删掉的探针目录、一次 `mkdir && ls`；Task 10 的评审在变异的副本上跑存储测试起了一个 testcontainers 的 Postgres（Ryuk 删掉，核对过）；Task 11 的复审一次 `cd /dev/null`、一个行内的 `while`、一次 `git status | head`；Task 12 的复审一次 `git -C status`。台账的"约""几次"照原样，不再细数。
    - 控制者：台账没有记下违反书面规则的调用。台账和裁定里写错、当场改正的五处：2–3 一行把 `Loops` 记在 Task 2（Task 2 的评审指出，:143 改正）；N3 的探针先写成 Task 4 的 T4.15，`held.grant` 从 Task 2 起才有，改为 T2.12（:128）；Task 9 的修复轮 1 一条写了不存在的 `TestRelationTypes`（:227，:228 改正）；Task 15 的评审之前的修改一条照实现者的话记下"报告写不进"，被拒的其实是 scratch 的文件（:358、:360，:365 和控制者的笔记 :164 改正）；I15 的行号早了一行（复审指出，:367）。
    - 执行之前：架构子任务的报告记下 13 次（`cd` 前缀 6 次、路径里的 shell 变量 5 次、一次在变异重跑时改了原型的一份文档、一次连在 `sed` 之后的 `git -C log`），修订又 2 次（路径里的 shell 变量、一个带 shell 变量的行内循环）；预检 1 次（行内的 `while`）。
    - 本文的作者 3 次，都只读：一个 `grep` 在路径里用了 shell 变量（运行了）；一个带 zsh 的 `echo =` 的调用和一个连起来的非 git 命令被拦下，没有运行。
- **整分支评审**（opus，`f8696490..f8e4c0d7`，62 个提交，`$M4TMP/p3final/review.md`）：Critical 0、Important 1（I1）、Minor 2，另有分诊的九项。它在 `git archive` 的副本上重跑全部门禁（gen-same 0 个差异，`make test` 47 个 `ok`，`make e2e` 126 个中 123 个通过：副本不是 git 检出的 S3 和 A4 的两个 firefox 测试）；Task 14 的 34 个测试和 `raceall.sh` 的 45 个的并集 64 个 `-race -count=5`，320 次顶层通过，`platform/gate` 50 次，0 行 40P01、死锁、DATA RACE；plan 的重放（780 个块，`make gen` 之后 3,475 对 3,475 个文件、0 个差异）；28 个变异的 59 层都被发现；T13.2、T13.7、T13.8 在 I9（`i9-relations.spec.ts:147`）被发现；A.3 说按性质看不到的三个（T13.4、T12.20、T10.23）在整个 `bootstrap` 活着，与 A.3 相符；只在一层的行逐个在每一层跑过（七行在 A.3 之外）。它在 HEAD 上重推了取锁的路径、逐个写的访客规则和死锁（第 4 节）。I1 是访客规则在设计层面的语义，控制者报负责人（用平实的话，附建议），修复轮等裁定；负责人裁定 A（2026-10-24）。
- **修复**：一轮，一个新的实现子任务（`final-fix-brief.md`），两个提交（`85f88b91`：代码、测试、契约、生成物和重新组装的 plan，21 个文件；`dcb99363`：spec、设计、差异清单和 M3 交接，4 个文件）。范围复审一次（opus，`f8e4c0d7..dcb99363`，`$M4TMP/p3final/rereview.md`）：八条裁定都已落实；新的问题 0 条；四条观察不需要改；"可以写评审记录、合并"。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

本文的作者在 HEAD `dcb99363`（本提交之前，树干净）上逐个运行七个门禁（`$M4TMP/p3sdd/gates-rd.sh`，日志在 `$M4TMP/p3sdd/rd/`，`summary.txt`），容器的两套之前先等机器上没有 Go 的测试（`wait-idle.sh`）。

| # | 标准 | 结果 |
|---|---|---|
| 1 | I8、I9 的接口版本和 I17 的接口版本（子树和它的负责人、标签、关联（两个方向）、关注、动态同一时刻删除，创建者的规则）、I15 的子工作项和关联的 409 通过；`make e2e` 126 个：此前的 123 个，加 I8、I9、I17 | **通过。** T15 `2a2a3419`（`29623915`、`4944e989`）：`make e2e` 124/126，失败的两个是 A4 的 firefox 测试；`MUT_LATER=15` 的 40 行死在全部 42 个端到端的层，`e2ekills.py` 0 个可疑；评审之前的修改让 I9 的数据两个方向都有（T9.8 加 I9 一层，T15.1），修复轮让 I15 读拒绝之后的 Fix the logout、T13.1 记下 I9 一层（`i9-relations.spec.ts:147`）。整分支评审在 `f8e4c0d7` 的副本上 123/126（多出的失败是 S3：副本不是 git 检出）；修复轮在 HEAD 上第二次 124/126（第一次另有 A9、P3、S8 三个页面故事在负载 14 时超时，第 7 节）。本文的作者在 HEAD 上：`make e2e` "Running 126 tests using 9 workers"，124 个通过（37.7 秒），I1（`i1-create-issue.spec.ts:23`）、I2（`i2-update-issue.spec.ts:30`）、I8（`i8-sub-issues.spec.ts:37`）、I9（`i9-relations.spec.ts:26`）、I15（`i15-archive-issue.spec.ts:46`）、I17（`i17-delete-issue.spec.ts:30`）在其中；失败的两个是 A4 的 firefox 测试（`stories/identity/a4-refresh.spec.ts:85:3`），错误是 `Error: browserType.launch: Executable doesn't exist at /Users/xiaoruan/Library/Caches/ms-playwright/firefox-1543/firefox/Nightly.app/Contents/MacOS/firefox`（机器上没有 Firefox；`rd/e2e.log`） |
| 2 | 交错 9（两次移动；加子工作项与移动）、10、11、13、23、26，加关联与删除工作区，两个删除同一条关联，两种顺序 `-count=5 -race` 通过，没有 40P01；加关联不锁它到不了的项目、预检没到达的锁下也不到达（`TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`）、写闸门、连接池的测试和 `platform/gate` 的单元测试同样五遍；P2、P1b、P1a、M3 的交错照旧通过 | **通过。** T14 `2c093568`：plan 的正则选的 34 个测试各 `-race -count=5`，170 次顶层通过，`platform/gate` 五遍，没有 40P01、死锁、DATA RACE。整分支评审在 `f8e4c0d7` 的副本上：64 个测试（Task 14 的 34 个和 `raceall.sh` 的 45 个的并集：P3 的交错、P2、P1b、P1a、M3 的交错、竞争、取锁形状、闸门和连接池的测试）各五遍，376 秒，320 次顶层全部通过，`platform/gate` 50 次，0 行 40P01、死锁、DATA RACE。之后的 Go 改动只有修复轮的 `85f88b91`（修改的父工作项的比较、`parentsOf`、两句拒绝的文字）；修复轮在 HEAD 上照 Task 14 的正则再跑（`$M4TMP/p3sdd/race-fw.sh`，日志 `p3sdd/logs/race-fw-*.log`，写于 `dcb99363` 之后）：`bootstrap` 199.6 秒，170 次顶层 PASS、0 FAIL、0 行 40P01、0 个 DATA RACE、0 行 deadlock；`platform/gate` 50 次 PASS（本文的作者重新数过这两份日志） |
| 3 | 加关联指向看不到的项目时不取锁、不进闸门（`TestARelationLocksNoProjectItDoesNotReach`，单元的 `TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches`） | **通过。** 两个测试（`bootstrap/issue_relation_locks_test.go:21`、`issue/app/lock_test.go:162`）在本文的作者的 `make test` 里通过（`bootstrap`、`issue/app` `ok`），`TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`（`:79`，Task 11 执行中加的）同样；前两个也在上一行的五遍里。整分支评审：反过来的变异 T10.7（跨项目的写只锁自己的项目）、T10.8（只进自己的闸门）在单元和组合两层被发现；T10.27（预检没到达的锁下再到达）在单元和那个交错被发现 |
| 4 | 一个看不到工作项的调用者在每个列表、数目、内嵌的 `Issue` 里都看不到它，作为目标时它的拒绝与不存在的逐值相同：矩阵的"访客·别人""管理员·访客"两列（五个新操作），`TestTheGuestRuleHoldsForSubItems`、`TestTheGuestRuleHoldsForRelations`、`TestTheGuestRuleHoldsForDeletingARelation`、`TestEachAnswerCountsTheSubItemsTheCallerSees`、`TestAHiddenParentIsNotInTheAnswer`、`TestARelationListShowsWhatTheCallerSees` | **通过。** 整分支评审之前这一条有一处不成立：修改时调用者看不到的现在的父工作项不算作"不存在"（I1），修复轮照负责人裁定 A 改正，`TestAHiddenParentCountsAsNoneInAChange`（`bootstrap/issue_parents_test.go:246`）逐字节比较指名它和指名随机 id 的回答，送回 `null` 什么都不写；T4.23（同存着的比）修复之前在 `issue/app` 和整个 `bootstrap` 活着，之后在单元和组合两层被发现（范围复审重跑）。本文的作者在 HEAD 上另跑矩阵（`go -C server test -count=1 -v -run '^TestPermissionMatrix$' ./internal/bootstrap/`，`rd/matrix.sh`、`rd/matrix.log`、`rd/guestcols.py`）："访客·别人"（`project_guest,_another's`）和"管理员·访客"（`workspace_admin_and_project_guest`）两列在五个操作的每一行都通过，0 个失败（`listSubIssues` 各 1 格、`addSubIssues` 各 2 格、`createIssueRelations` 各 2 格、`listIssueRelations` 各 1 格、`deleteIssueRelation` 各 2 格）。点名的六个组合测试在 `make test` 里通过（`bootstrap` `ok`） |
| 5 | 矩阵（P3 的 19 行）、按目录的删除测试（没有豁免）、完整性测试通过 | **通过。** 同一次矩阵的运行（`p3tools/matrixops.py`）：1,455 个叶子全部通过、0 个失败；P3 的五个操作 19 行、196 格（`listSubIssues` 20、`addSubIssues` 52、`createIssueRelations` 64、`listIssueRelations` 20、`deleteIssueRelation` 40），P1a–P2 的工作项的操作 393 格，其余 865 格是 M3 的，另有模板的 `prepare`；顶层 1.58 秒。与 spec A.4 记的格数相同。`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（没有 `subItems` 的豁免）、`TestThePermissionMatrixCoversEveryOperation`、`TestMatrixViolationsCatchesEachGap` 在 `make test` 里通过；`git grep` 在 `e2e/` 和 `bootstrap` 里找不到 `unwritten`、`withoutRows`，`subItems` 只剩 `issue_sub_issue_count_test.go` 里读子工作项数的帮手，不是豁免 |
| 6 | `make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e` 通过 | **通过。** 本文的作者在 HEAD `dcb99363` 上（`rd/*.log`）：`make gen-check` exit 0；`make lint-go` 两段 `0 issues.`；`make test` exit 0，47 个 `ok`、0 个 `FAIL`，`go test -count=1`（没有缓存），`bootstrap` 91.1 秒；`make lint-web` exit 0（"keywords: 72 rules, 23 exceptions, no hits."；i18n-keys 1,300 个键；turbo 54/54，其中 38 个读缓存，`--force` 不用缓存再跑 54/54，`rd/lint-web-turbo-force.log`）；`make knip` exit 0；`make test-web` 17/17（都读缓存；`--force` 再跑 17/17，`rd/test-web-force.log`）；`make e2e` 见第 1 行。跑完之后树仍干净（`git status --porcelain` 没有输出）。`server/go.mod`、`go.sum`、`server/tools/go.mod`、`go.sum`、`pnpm-lock.yaml` 自 `f8696490` 没有变化（`git diff --stat` 没有输出）。本提交的文档改完之后（提交之前）：`make lint-web` 和 `doclinks.py` 见本文的作者的报告；文档不进生成物 |

spec 第 4 节的六个框在本提交勾上。

持续集成：分支在执行中没有推送（`git branch -r` 只有 `origin/HEAD`、`origin/main`）。合并之后的持续集成在合并时记录（P2 的做法）。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，写明理由和裁定错了的代价（台账写了的照录）。完整的记录在执行台账（`.superpowers/sdd/P3-issue-hierarchy/progress.md`）和控制者的笔记（`$M4TMP/p3-controller-notes.md`）中。

**执行前**
- **负责人的裁定**（设计第 10 节，2026-10-17，P3 照用，不重开；理由和以后改的代价写在设计各条里）：
  - 决策点 1 的 B：工作项的写只锁项目行（N），不锁工作区行；预检答出一切会被拒绝的写，取连接之前进项目的写闸门。P3 的父子、关联都走 P1b 的同一条路（`Locks.run`），只多两种形式（跨项目的 `lock_across.go`、两端任一个的 `lock_either.go`），不另写第二条；spec 第 3 节第 3 条把闸门的位置算作资源，对 HEAD 的每个写重推死锁。错了的代价（设计）：B → A 只改取锁的一步和测试，但要重新面对饥饿。
  - 决策点 2 的 A：一个 M4；遇到超过上限的 Phase，控制者直接拆成 P<k>a、P<k>b 继续，不停下来问（预先批准）。P3 没有拆 Phase：十一个草稿任务切成 16 个 Task（spec 第 3 节第 1 条），最大的两个算"约 1,500 行"（同 P2 的 1,519 行）；执行之后五个 Task 长过 1,500 行（第 5 节），不重新切。错了的代价（设计）：中途要拆时在 Phase 的边界上收尾，成本是一次额外的收尾。
  - 决策点 7 的 A：删除项目、工作区时工作项一侧的行由 P9 的任务删除，P1–P8 在请求的事务里同步（过渡）。P3 的关联行的连带同步，`ProjectGate.DeletedAt` 到 P9（spec 第 3 节第 19 条：同步的连带之下没有可跳过的行，端口没有读者）；`eitherUnder` 在判定之前核对两个项目都锁住、`LockMany` 不回答等锁期间删除的项目，在 P9 改为任务之后才分得出（第 6 节）。错了的代价（设计）：A → B 删掉订阅者和任务，把连带的语句放回请求的事务。
  - 架构子任务和控制者的裁定都在负责人的裁定（决策点 1 的 B、2 的 A、7 的 A，架构问题 11.1–11.11）之内，没有一条要报负责人。
- **对架构的报告的裁定**（`$M4TMP/p3-architect-rulings.md`，2026-10-21）：spec 第 3 节 1–23 条都照建议，理由在各条里；带条件的四条：
  - 第 5 条（子树在锁下读、一个时刻、已归档的也在、不分批）：预检在目标规模上量；整个项目挂在一个工作项下的删除超出 15 秒的期限就改为分批（键集、一个时刻，同删除项目）。预检量过，远在期限之内，不分批（spec 第 6 节）。
  - 第 7 条（加关联的另一端的预检，它自己的测试 `TestARelationLocksNoProjectItDoesNotReach`）：设计里说它是"交错 21 的同一个测试"的句子（3.5、交错 21、9.3）改为交错 21 的关联的一例、与取锁形状的两行并列；6.5 `ProjectGate` 一行的 `LockMany` 的检查照实写。理由：取锁形状的测试锁的是表，造不出"看得到工作项、看不到对方的项目"。
  - 第 10 条（关联的列表逐个项目判定，加不加锁的窄端口 `ArchivedProjects.Archived`）：设计 3.3、3.4、6.5 照它改；原来 spec 第 5 节 P5 一行的条件被预检 M3 改过（见下）。
  - 第 18 条（`withoutRows`、`unwritten` 是 Task 之间的状态）：预检核对最终的树两者都没有，按目录的删除测试和 `expectIssueDeleted` 也没有别的豁免。预检核对过，成立；HEAD 上同样（第 2 节第 5 行）。
  - 第 20 条（按性质只在单元一层的）以预检为准：教训说"只在单元一层"的说法常常是错的，预检在组合一层跑 T7.1、T10.4 和 `eitherUnder` 的项目的核对。结果：T7.1 的理由是错的（L2），另两个成立。
  - 架构子任务的 13 次失误都只读，记下；修订的派发重申规则。
- **预检**（`$M4TMP/p3-preflight-rulings.md`，2026-10-21）：全部照建议，没有报负责人（都在 M4 的实现之内；M1 把设计 3.4 自己的规则用到多一个字段；M3 改正控制者的一条裁定；其余是测试和文档的事实）：
  - M1（看不到的父工作项的 id 在回答里）：遮住（选项 a）。理由：设计 3.4 说访客规则处处执行，3.4 和 3.13 已经不给他指名看不到的工作项的动态行，回答里的 `parent_id` 是同一个指名；选项 b 会把已知的泄露写进文档。条件：遮挡在 `seen` 一处，每个回答都经它；用访客规则的一个谓词 `domain.Issue.HiddenFrom`；只对 `OwnOnly` 下有父工作项的读一次（`GetIssues` 从 Task 5 提前到 Task 4，不另加查询）；读不到的也遮住，读失败是回答的错误；单元的五行、组合的 `TestAHiddenParentIsNotInTheAnswer` 和 Task 12 内嵌的一例；T4.12、T4.13；spec 第 5 节 P4、P5、P14 的交接（P14：页面只在用户改了父工作项时发 `parent_id`，否则遮住的 `null` 会把工作项拿出来）。
  - M2（加子工作项的祖先链在 N 之下读没有组合一层）：照提议加 `TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize`（Task 6）和 T5.26，进 Task 14 的竞争的列表。
  - M3（裁定 10 的前提是假的，已核对 `access/domain/decide.go:77-104`、`rules.go:137`）：照预检的文字改 spec 第 5 节 P5 一行（`seeEach` 和 `ArchivedProjects` 可以换成 `VisibleProjects`，`ArchivedProjects` 没有别的读者时删去）和设计 3.4、3.10 的两句。
  - L1–L11 照预检：L1 竞争的列表加写闸门的测试和 `platform/gate` 的一行；L2 T7.1 的理由是错的，组合的 `TestAWriteAcrossProjectsWaitsMaxWaitInAll`（Task 11），计时按"期限不会早到、上沿在对的答复和错的答复的中点"（一个教训的类）；L3 删除关联与删除它另一端的工作项成为交错 26（不重编号）和 T13.25；L4 一个泛型的 `listIn[T]`；L5–L7、L10 设计的句子，L8 spec 第 5 节 P9 一行的句子（T7.5、T13.4 在 P9 的组合测试）；L9 量过的数写进 spec 第 6、7 节；L11 形状不对的请求在闸门之前答 422（写闸门的组合测试两行，T5.16、T10.22）。N1 `relations.sql` 的注释在 Task 9 结束时就成立；N3 子树整个删除，含删除者看不到的子工作项（写进第 5 条，不改代码）。
  - 修订的 D1–D11（裁定没有预见的决定：`GetIssues` 周围的编号、遮挡的写法、写闸门测试的拆分、L2 的测试放进新文件 `gate_across_test.go`、T13.25 照 PF.E3 的写法、Task 12 1,560 行不拆等）：照报告接受。理由：每条都把一条裁定放在它自然的位置，或让文件留在约 400 行以内。代价：错了是那个 Task 的一个修复轮。

**过程的裁定**
- **plan 是"执行时的 plan"**（P1b 的裁定，从 Task 1 起）：每个修复轮在同一个提交里改它的 Task 的块，执行中加的变异行进表，plan 从基点复现到 HEAD。理由：plan 是能复现的记录。代价：每个修复轮多几分钟。
- **spec 附录 A 是注明日期的原型记录**（2026-10-21，修订同日）：修复轮只改 spec 正文的事实；执行中加的变异行进 plan 的表，评审记录数它们（第 5 节）。**A.3 是例外，保持为真**（Task 10 的修复轮 2，取代原来"留到最后一轮"的裁定）：A.3 现在就补上之前各轮的只在一层的行，各写理由，清扫的和执行中的分开数，A.6 写明是清扫的记录；从 Task 11 起每轮在同一轮写自己的条目。理由：spec 在每个 Task 结束时自身一致。代价：一次 spec 的整理；之后每轮几行。没有记下按性质的理由的只在一层的八行（存储 T4.17、T5.27、T5.28、T9.20，组合 T2.12、T4.21、T4.22、T6.5）：T9.20 交给 Task 11 的 `MUT_LATER=11`，其余七行交给最后一轮在每一层跑过，找到的层记下，否则写真的按性质的理由，再不然加测试（第 4 节分诊 4）。
- **模型**：实现者、文档的作者不指定模型；Task 的评审 opus；小的范围复审 sonnet，共用、并发的部分 opus（Task 8 的复审）。只改文档或记录的修复轮由控制者自己核对差异。
- **`MUT_LATER`**：较早的 Task 的"（Task n 起）"的行照 plan 用 `MUT_LATER=n` 跑。**INFRA**（P2 加的 `mut.py` 的结论）：容器起不来的层不算发现、也不算留下，Docker 空闲时重跑一次。
- **常设的错误的裁定**（Task 4 的修复轮 1 之后）：一组的语句的错误写个数、不写 id，Task 5、7、9、10、12 的实现者在评审之前照改、重新生成块。理由：负载下无界的日志行（Task 1 的评审：2 万个 id 约 740 KB）。代价：每个 Task 几个格式串。
- **plan 头的大小**（Task 11 的关切 3）：那一句量的是 plan 的块建出的树（执行之后的最终的树），不是原型；`keepsizes-t11pf.py` 核对每个说到的大小，对不上就失败，之后的重新组装都用它。理由：plan 是"执行时的"，那句话告诉执行者他会遇到的大小。代价：一句、一个检查。
- 常设的数据、"只在单元一层"和句子的规则（台账开头）：数据分得出每个性质（访客与成员、自己的与别人的、管理员·访客、已归档、已删除、两层以上的父子、两个方向的关联、两种 `id` 次序的项目对、调用者不总是最小的 id、隐藏的 404 与不存在的逐值比较）；"只在单元一层"的说法在组合一层核对；文档的句子不宽于代码，交接和差异清单也是。

**执行中**（补上缺口的，都先由变异证明缺口存在，补上之后再证明测试咬得住）
- **Task 1（评审 Important 2 条）**：
  - 评审之前（`d3353443`）：N3（子树连同删除者看不到的子工作项）没有测试：实现者的探针（只看自己的删除者去掉看不到的子工作项）在单元和整个 `bootstrap` 都活着 → 组合的 `TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee`（walt 删他自己的工作项，下面有 alice 的两层子工作项，同一时刻由 walt 删除），探针成为 T2.12（`held.grant` 从 Task 2 起才有），spec 第 5 条点名测试。代价：一个测试、一行。T1.9 的组合一层两次 INFRA → Docker 空闲时重跑，被发现。`adminGuest` 从 Task 2 移到 Task 1 的 `issue_world_test.go`：批准（共用帮手的扩展；代价：Task 2 的一个块重新生成）。留下的 `Created` 容器：不动。
  - 评审（Important ×2，plan 写定的，不改行为）：两句说明比数据宽（`issue_deletion_test.go:79-80` 的"the same rows"、`delete_issue_test.go:90` 的"in no order"）；`delete_issue.go:37` 把子树的每个 id 写进错误。裁定都接受（`59468ade`）：两句照评审改（Task 8 的块重新生成）；错误写个数。理由：比数据宽的句子误导下一个读者；负载下无界的日志行是真的运维代价。代价：三行和块的重新生成。复审（sonnet）：已落实；Nit "1 work items" → 不改。
- **Task 2（评审只改记录）**：
  - 评审之前（`8cffaa2b`）：`TestTheGuestRuleHoldsForAParent` 的说明写 bob 在 alice 的 Up 下建，测试里没有 bob，Task 2 的组合数据分不出"父工作项要是他自己建的"与"他看得到的" → 加 bob 的一例，T2.13（单元；组合）；`stopped` 闭包近似 `wantStoppedAt` → 一个核心帮手，`wantStoppedAt`、`wantStoppedOn` 经它（Task 3 的块随之改）。理由：自己的与别人的分得开；复制的帮手是缺陷的一类。代价：一例、一行、小的重构。
  - 评审：T2.10 的组合一层在 Task 2 的树上就被发现，不是"（Task 3 起）"；Minor：第 3 节第 12 条（创建不写第 5 行）没有行。裁定都接受（`ecbdf2b0`，只改 plan，控制者核对）：T2.10 的层移到本 Task；T2.14（单元；组合）。理由：错的"起"让后面的 Task 白跑；说了的规则没有行是没钉住的说法。代价：表的几行。
- **Task 3（评审 Important 2 条）**：
  - 评审之前（`4df19c98`）：`hierarchy.go:39-40` 的"and so does an addition of sub-items"在 Task 5 之前为假 → Task 3 的块去掉，Task 5 的块加回。代价：两个块。
  - 评审：Important (1)（plan 写定的）每个允许的移动的目标都是根，`|| len(ancestry) > 1` 的变异活过单元、按构造也活过组合；Important (2) 领域的 `sameID` 抄了 `sameDay`（也是 plan 写定的，`650dc819` 的 plan :2481）；Minor：spec 2.5、plan :2102 说 `IssueReader.Ancestry`，代码声明在 `IssueUpdater` 和 `SubIssueAdder`。裁定都接受（`9eabe3d4`）：单元的目标有祖先，组合把 Up 放到 Top 下、再把 Low 放到 Up 下，T3.16（单元；组合）；一个泛型的 `same[T comparable]` 取代两个（`dates.go` 加入 Task 3）；两处事实修正。理由：数据分得出性质；不重复的逻辑；文档不宽于代码。代价：两例、小的重构、一行。复审（sonnet）：已落实；新的：spec 2.1 的文件数在 `dates.go` 加入之后不对 → 控制者改（`f4b34b97`）。
- **Task 4（评审 Important 2 条）**：
  - 评审之前（`07516159`）：实现者的三个关切：修改、按编号取的回答吞掉数子工作项的失败；`TestGetIssues` 分不出 `ORDER BY id` 与扫描的次序；没有最后修改者不是创建者的子工作项 → T4.15、T4.16、T4.19、T4.20（单元，端口的失败）、T4.17（存储，`ANALYZE` 前后）、T4.18（存储；组合）；两个 Nit 改；`seen` 的多个工作项的分支留着（`seenOne` 用 `seen`，有单元测试，Task 5 是计划中的调用者）。理由：端口的错误不吞；数据分得出性质。代价：几行测试。
  - 评审：Important (1)（plan 写定的）契约的 `IssueDetail.parent_id` 只说"null for none"，遮住之后看不到的也是 null（`Issue` 同样）；Important (2)（plan 写定的）`postgres/sub_issues.go:18`、`issues.go:131` 打出每个 id；Minor (3) 写里的 `hideParents` 的读在事务的连接上只在单元钉住。裁定都接受（`1a9e2bdd`）：两个模式的契约一句，Task 4、6、8、11、12、14 的钉值重算；错误写个数；一个连接的池的测试加 walt 在看不到的父工作项下的工作项上的写，T4.21、T4.22（组合）。理由：契约是代理唯一读的文字；无界的日志行；spec 说"同一个连接"。代价：钉值的连锁、两个格式串、一例。常设的错误的裁定由此而来（见上）。复审（sonnet）：已落实；新的 Minor："issue.yaml（848 行）"不对 → 控制者改为 854（`74182225`）；附录的数落后于执行的行 → 照常设的裁定，评审记录数。
- **Task 5（评审 Important 1 条）**：
  - 评审之前（`93bf8714`：常设的裁定，三个错误写个数；`5214e682`）：实现者的关切：按最后修改者算"自己的"活着（T4.18 一类），`, i.id` 的次序没钉住，没有管理员·访客的单元一例，`sub_issues_test.go` 近似地抄了 `wantStopped`、`wantActivity`，说明指名 Task 6 才有的路由 → T5.27（存储）、T5.28（存储，`ANALYZE` 前后）、T5.29（单元；组合 Task 6 起）、实现者加的 T5.30（写的范围里访客规则豁免管理员·访客：单元；组合）接受；共用的帮手取代副本（Task 10、13 由派发带上）；说明不指名路由。代价：三行、小的重构。
  - 评审：Important（plan 写定的）`sub_issues_test.go` 的 `losing` 重写了假实现的 `lost`；Minor：生产代码两处写了"原来的父工作项在锁下读，不在是错误"；三条说了的规则没有行。裁定都接受（`680dada5`）：扩展 `lost`、删去 `losing`；`parents.go` 一个 `parentsBefore`，修改和加子工作项都用；T5.31（单元，从 Task 5 起就是 T3.13 的编辑）、T5.32（单元：Task 6 的整个 `bootstrap` 放过它，一条语句里移动的次序在组合看不到，接受）、T5.33（存储）。理由：生产和测试都不重复逻辑；说了的规则有行。代价：小的重构、三行。带给 Task 10：`losingRelations` → 共用的 `lost`。复审（sonnet）：已落实。
- **Task 6（评审 Important 2 条）**：
  - 评审之前（`732d99c4`）：没有测试列已归档的工作项的子工作项（两个探针活到最终的树）→ Shelved 下放 Kept，T6.5（组合）、T6.6（存储；组合）；T6.4 加组合一层（T11.5 的组合一层交给 Task 11）；实现者的探针（删除在锁之前读子树）成为 Task 1 表里的 T1.11（单元；组合 Task 6 起）；矩阵的检查经 `listIn`；一句说明改准。理由：数据分得出性质；行记下实际发现它的地方。代价：一个种子、三行、小的测试改动。
  - 评审：Important ×2（plan 写定的）：三个交错的终态检查和准备是复制的；两句说明不对。裁定都接受（`63b7ae6f`）：一个终态帮手和一个准备；两句改准。理由：复制的检查会漂；文档不宽于代码。代价：测试的重构、两句。复审（sonnet）：已落实；一个三行的 422 的核对出现两次（可选，不裁定）。
- **Task 7（评审通过）**：
  - 评审之前（`6c67e11e`：常设的裁定；`a333f755`）：T7.9 也在存储一层被发现 → 加存储一层；`Enter` 的 `slices.Clone` 没钉住（探针 P7.b）→ 留着并写明理由（返回的 `leave` 在事务之后让出键，不读调用者之后改过的切片），`TestACallerLeavesTheKeysHeEntered`，T7.10（单元）；两句说明改准。理由：行记下实际发现的；没有理由的代码不留；文档不宽于代码。代价：一个测试、一行。
  - 评审：通过；Minor ×2：没有测试钉住 `writeGate` 照给的次序交给闸门；`platform/gate` 的 `Enter` 的说明比代码宽。裁定当轮改（`6b0b73bf`，控制者核对，不复审）：`TestTheWriteGateAnswersItsRefusalsAsServerBusy` 表的循环之后加次序的一例（另写一个测试会让 `gate_test.go` 超过 400 行），T7.11（单元）；说明改准。理由：次序正是让闸门上的等待没有环的东西。代价：一例、一句。控制者核对：这一例杀死倒序和升序排序两种变异。
- **Task 8（评审通过）**：
  - 评审之前（`f6b6f212`）的六条：(1) 端到端的 Firefox：环境，同 Task 1、4；(2) P3 之前手写的 `55P03` 是 P1b 交给收尾的条目，`workspace` 的 `deactivation_test.go` 的 `waitsFor` 是 `pgtest.RowWaits` 的近似副本，一起在收尾合并（第 6 节）；(3) `pgtest.RowWaits` 用 `postgres.LockNotAvailable`，T8.14 重新锚定到它说的缺陷；(4) 由 (3) 一并解决；(5) `RowWaits` 的语句找不到行时测试失败，T8.15；(6) 删除测试存每一种类型（`newRelation`、`relate` 带类型），T8.16–T8.18（存储；组合，交错 11 Task 14 起），Task 9、12、13 的调用处随之改。理由：共用的帮手不能替缺陷作答（空的"不锁"的核对）；数据在带来代码的 Task 分得出每种类型。代价：一个帮手、测试数据、后面的调用处、四行。
  - 评审：通过；Minor ×4：spec 的文件数；子树里不是根的工作项没有跨项目的关联（关联一步只给 `[:1]` 的变异只死在单元）；`DeleteProjectRelationRows` 的 `OR … IN (SELECT …)` 用不上索引，每批读每个工作区的关联（评审的数据：733 条的项目滤掉 265,767 行，约 32 ms，对工作项一批约 5 ms），要保持一条语句；`seedIssues` 的关联 INSERT 是 `relate` 的副本。裁定当轮全改（`2fff6efb`）：计数；子树里的 c 关联到子树之外的 o，T8.19（单元；组合）；`UNION ALL` 的两支各用自己那一端的索引，仍是一条语句、按 `id` 的锁、键集、同样的行（500 个项目上核对），定制计划 26.1 → 6.2 ms、通用计划 43.3 → 23.5 ms；`seedIssues` 调 `relate`。理由：代价随每个工作区增长的一批是 I2 一类；语句是 P3 自己的。复审（opus）：都已落实（PG 18.6 上两种计划的 `LockRows` 都在 `Limit` 之下，只锁关联行，等锁时被删的行跳过）；新的 Minor ×2 → 修复轮 2（`c1f41480`，只改文档，控制者核对）：注释写上 P5 之前通用计划的例外，spec 第 7 节写出每个数的计划模式和通用计划的一对（146 → 429 ms）：P5 之前这一步在那样的数据上比原来慢，好处在 P5 强制定制计划之后才成立；P5 的 `force_custom_plan` 已在设计 3.16，不再改 SQL。
- **Task 9（评审通过）**：
  - 评审之前（`045fb636`，派发带上的）：五个错误写个数；`TestRelated` 的数据三种存储类型、`TestRelationAdded` 四种，T9.16（存储）、T9.17（单元）。对实现者的五个关切：(1) T9.16、T9.17 也在组合一层被发现 → 记为 Task 11 起、Task 14 起；(2) 带给 Task 13：`TestRelationRemoved` 覆盖四种；(3) `ChangedRows` 的"adds or removes"不改：类型的契约（设计 3.7），同 Task 7 的端口说明；(4) `storedRelations`、`relationText` 与 `linkTable`、`linkLine` 不改：形状不同，不是 Global Constraints 的帮手；(5) `TestInsertRelations` 的"in their order"不改：对代码成立，不是安全的性质（冲突的插入在两端的项目锁上排队）。
  - 评审：通过；Minor ×4（plan 写定的）：空列表的规则写了两次；`relation_test.go:32` 遍历生产的类型列表；存储测试共用一个 `&alice`，分不出指针的比较；"asked for in no order"。裁定当轮全改（`5dfc7848`）：`checkSomeIDs` 两处共用，T9.18（单元；组合）；四种类型逐个写出，T9.19（单元；组合 Task 11 起）；各自的指针（`domain.NewRelation`），T9.20（存储）；措辞；`relations.sql` 的契约说"addition or removal"。P2 的优先级的循环（同类，P2 的文件）→ 整分支评审之后的修复轮。理由：生产里不重复规则；去掉性质时测试要失败。代价：一个帮手、测试数据、四行。复审（sonnet）：已落实；新的 Minor：plan :10244 的指针一句比数据宽 → 修复轮 2（`4637c60f`，一行，控制者核对）。
- **Task 10（评审 Important 1 条）**：
  - 派发带上的（`8a466e05`）：错误写个数；共用的活动核对；`losingRelations` → 共用的 `lost`、`lose`；`CreateIssueRelations` 的说明不指名 Task 11 才加的路由（Task 11 的块写上）。实现者另加的 `wantRefused`：照裁定 2 的原则确认。实现者的关切（`0c3a4f76`）：P10.a（共用的 `lost` 丢掉活动指名的每个工作项，读回的核对弱化成"一个也没有"活着）接受、不改：核对守的是按位置配对，部分丢失只会以 panic 杀死弱化；新 id 在 `Execute` 里生成，丢一个子集要给共用的假实现加开关，为锁排除了的分支加设计的分量；错了的代价：弱化没人发现，部分的读回会在 `answer` 里 panic 而不是报错，而那个状态锁排除了。P10.b（跨项目的每个写都是 web 的，web < ops，把自己的项目放在最前的 `reachAll` 活过单元和组合的 8 层）→ 从较高的项目的工作项关联较低的一例，T10.23（单元）；P10.c（只以 blocking 关联，存的类型等于对方读到的）→ blocking、blocked_by 各一例，T10.24（单元；组合 Task 11 起）；两处三行的"什么都没写"合成一个帮手。info 4（Task 11 的块拼接、不重新生成：只差 `issue.yaml` 一对的上下文对齐）、info 6（T10.19 只以 panic 死：按性质）接受。
  - 评审：Important（plan 写定的）`reachUnder` 的失败没测（只在预检的第一次调用让端口失败）：R1（端口的错误答成 422）、R2（`lockAndDecide` 丢掉 `reachUnder` 的错误）活过 `issue/app`；Minor：R4（"预检没到达的锁下也不到达"没钉住）；R3（回答的 `grants` 里工作项自己的项目从没被读）；`from := end{…}` 违背 `end` 的说明。裁定（`2be08ad6`）：锁下也让读和判定失败，T10.25、T10.26（单元，按性质，A.3）；预检看不到、锁下看得到的一例，T10.27（单元）；`grants` 从空的开始（没有可改坏的，不加行）；`held.own()`（带给 Task 13）。理由：端口的错误绝不能作为拒绝到达客户端；设计 3.5 定了预检的拒绝；缺的是指名自己那一端的办法。
  - 复审（sonnet）：已落实；(1) 附录 A 自相矛盾；(2) T10.27 的 A.3 理由是假的（`serializeOn`、`WaitForLockWaitOn` 造得出）。裁定：(1) 修复轮 2（`103dacf4`，只改 spec，控制者核对）补 A.3（单元清扫 16 + 执行 10、存储 17 + 2、组合 6），T7.11、T10.23 进闸门次序一项，第 3 节第 20 条点名，A.6 写明是清扫的记录；(2) T10.27 离开 A.3，Task 11 加组合的交错，做不到则回报。T10.24 不进 A.3（Task 11 给它组合一层）。
- **Task 11（评审 Important 1 条）**：
  - 派发带上的（`98292afd`）：T11.5 的组合一层；`relatedTo` 经 `listIn[answeredRelation]`；矩阵文件头用子工作项的真话；`TestWorkItemsAreRelatedAcrossProjects` 也以 blocked_by 关联（T10.24）；T10.27 的组合交错 `TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`；T9.20 进 `MUT_LATER=11`。关切 1（那个交错在 alice 的 PATCH 下打开 Web 的 `guest_view_all_features`，不是提升 walt）接受：性质是调用者等锁时看得到的变化不改预检的拒绝，两种触发翻转同一个 `Grant` 位（`OwnOnly = guest && !GuestViewAll`，`access/domain/decide.go:89`），PATCH 持着关联等的那一行；提升要复制 `bobsMembership` 或加宽一个大小例外的文件；错了的代价：等锁时的成员关系变更靠同一个位和 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused` 对写的部分。关切 2（名字合进 `issue_world_test.go` 的 `names`）确认；关切 3（plan 头的大小，见上）；info 5（局部的 `relates` 遮住包的 `relates`）改名；info 6–8 接受。
  - 评审：Important（plan 写定的）`relating` 抄了 `subIssueIDs` 带引号的 id 的循环，`relatesTo` 手拼体；Minor：Task 14 的竞争的正则漏了新交错；矩阵文件头从 Task 14 起不对。⚠️ 组合的数据分不出升序的闸门次序：不算发现（A.3 写了闸门的次序按性质只在单元一层）；错了的代价：最后一轮的 A.3 核对再看（整分支评审重看：T10.23 活过整个 `bootstrap`，A.3 成立）。裁定（`0fe250aa`）：`idArray`（`issue_world_test.go`）由 `relating`、`subIssueIDs` 读，另并掉 cascade、pool 的循环和空体（体逐字节相同）；Task 14 的正则和列表写上新交错（34 个），spec 第 4、6 节和 A.2、A.4、A.5 写 81（原型）、82（执行）；Task 14 加宽矩阵头；大小一句在最终的树上；`relateCells`。另：R1 T10.22 的一层两次 INFRA → Docker 空闲时再跑一次，被发现；R2（23 个单个 id 的数组内联）接受：别的操作的测试数据，不是那两个体或循环的副本；R3 带给 Task 16。复审（sonnet）：已落实；新的 Minor：`permission_matrix_issue_updates_test.go:93-94` 手拼三个 id 的数组 → 最后一轮（分诊 2）。
- **Task 12（评审 Important 1 条）**：
  - 派发带上的（`f80c034f`）：错误写个数；`listsRelations` 经 `listIn`；一个由体建项目的帮手 `createdProjectFrom`（`createdProject` 读它，Lab、Task 11 和 P2 的私有项目由它建）。理由：第三处内联的副本会由 Task 12 加，帮手属于第三个用处出现的地方。实现者的关切（`6020f200`）：没有测试列已归档的工作项的关联（P12.b 活在每一层）→ 单元一例、`TestARelationListShowsWhatTheCallerSees` 里归档 Shelved 时列它的关联，T12.19（单元；组合）；`seeEach` 对被拒的项目只判定一次没钉住（P12.a）→ lab 的第二个工作项，T12.20（单元，A.3：回答相同，只有判定的调用不同）；M3 的三处做了 `createdProjectFrom` 的事 → 并掉；`keyOfRow`；info 6（工作项自己的项目之外没有对方时也查一次空的 `ArchivedIssueProjects`）接受：读路径上一次便宜的查询，守卫要多一个调用记录要钉住的分支，看不出好处；错了的代价：每个列表一次往返。R1、R3 确认。
  - 评审：Important（plan 写定的）`ascending` 还有两处逐字的内联副本（`seen.go:74`、`create_issue.go:186`）。裁定（`0badb50b`）：两处读 `ascending`，它的说明写上使用者，spec 2.14 和 plan 的 Interfaces 点名。F1（`ascending` 留在 `lock_across.go`）接受：它的形式是加锁次序的约定（约定七），`reachAll` 是定义它的使用者，别的文件复用；不建 utils 文件，移到另一个锁文件只是外观；错了的代价：最后一轮移一个函数。F2–F4 接受。复审（sonnet）：已落实；Task 13 的两个使用者 → 带给 Task 13（spec 2.15 点名）。
- **Task 13（评审 Important 1 条）**：
  - 派发带上的七条（`07b97170`）：`TestRelationRemoved` 四种（T13.26）；`wantActivityRows`；扩展 `wantNothingWritten` 到假实现记下的每个关联的写；`first, second := h.own(), h.instead`；`DeleteIssueRelation` 的说明不指名路由（Task 14 的块写上）；锁下非第一次的调用失败（T13.27–T13.30，单元，A.3）；spec 2.15 点名 `ascending` 的两个使用者。关切 1（**两端任一个的形式的 grant**：`held.grant` 是放行的那次判定的，`h.own()` 把它放进第一端，即使判定在第二端的项目上；`held.instead.grant` 总是零；没有读者）照说明接受：没有读者；给每端它自己项目的 grant 会让没判定的一端是零（`OwnOnly` 为假，读作"看得到全部"），是更宽松的潜伏陷阱；按形式分 `held` 为没有的读者改共用的路径。错了的代价：出现读者时 `eitherUnder` 放各端的 grant。关切 2（P13.f 不排序、P13.g 先标第二个：单元唯一的关联存为 far，按构造就是升序）→ 存为 near 的关联一例，T13.31、T13.32（单元，A.3）（`65da0827`）。关切 3（**`inserted` 的分开**：假实现把插入的关联记在新字段 `inserted`，`relations` 只是读的夹具）确认：读的夹具和写的记录是两件事，裁定 3 要它。info 4 接受。
  - 评审：Important（plan 写定的）第 3 节第 11 条的次序只在一个交叉方向钉住（逐端核对、另一端在先的写法通过 `issue/app`，也活过后面各层）。裁定（`9833c09c`）：单元一例"ops archived, near archived too"，T13.33（单元），组合一层带给 Task 14，做不到则进 A.3。理由：裁定的次序两个交叉方向都要。代价：一例、一行、一个组合的关联。复审（sonnet）：已落实。
- **Task 14（评审 Important 2 条）**：
  - 派发带上的（`0a638081`）：`TestWhoMayNotDeleteARelation` 把 Old 的工作项关联到 Shelved，删除答 409 `project.archived`，T13.33 的 Task 14 一层。关切 1（P14.a，T13.33 的镜像，只死在单元）→ Old 的第二个工作项从 Shelved 关联，T14.5（单元；组合）（`a946ad9c`）。理由：裁定的原则是两个交叉方向、每个能造出的层；派发只写了一个方向。T13.12 的 `TestWhoMayNotDeleteARelation` 一层（P14.b）确认。
  - 评审：Important ×2（文档，plan 写定的）：`relationOf` 的说明对访客、WM-公、已归档的项目的成员不对；交错 26 的"先删 Far"一支说删除在锁下重读关联，实际更早在 `eitherUnder` 重读 Far 时被拒绝。Minor：spec 2.16 的等待帮手；`nearAndFar` 第六次手拼成员请求。裁定（`51d3375d`）：三句改准、spec 2.16 照实；`makeMember`，世界的九处读它（**M3 的世界照旧经它们自己的世界**：`project_*`、`membership_world`、`ending_world`、`interleaving_*`、`permission_matrix_*` 不动，不是 P3 的文件）。设计 :351 的一句带给 Task 16。复审（sonnet）：已落实；spec A.3 T13.4 的理由 → 带给 Task 16。
- **Task 15（评审通过）**：
  - 评审之前（`29623915`）：I9 说"either way""from either work item"，数据各只一个方向（P15.a = T9.8 的编辑活过 I9；P15.b 只看第二个工作项活过每个故事）→ 扩展数据：Fix the login 指名 Deploy the site（存为 Deploy the site 的）→ 422 `duplicate`；bob 删 admin 的 Fix the login 到 Probe the cache 的关联（他能改 Fix the login、看不到 Probe the cache）→ 204；T9.8 加 I9（Task 15 起），T15.1（单元；矩阵；I9）。`issueRelationsListed` 改为 `listIssueRelations`（spec 2.17、文件表、Interfaces）；两处措辞；`changed` 下一个私有的行的写法 `activityRow`；一个 `madeBy` 给 I8、I9；Interfaces 的次序一句改准。p3tools 的变异文件头说 Task 14、15 不带行 → 改（接受）。
  - 评审：通过；Minor ×3：I15 的"none of the four writes anything"没断言 Fix the logout；I8 的 Plane 说明挂在每一行；T13.1 在 I9 死（推理），没记。裁定当轮改（`4944e989`，I15 是端到端的改动，不是 Task 16 的文档）。复审（sonnet）：已落实；T13.2、T13.7、T13.8 也应在 `i9:147` 死 → 最后一轮跑。
- **Task 16（评审 Important 2 条）**：
  - 派发带上的（`ac426aeb`）：M3 交接第 2 节点名 `TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`；设计 :351 的交错 26 一句；spec A.3 T13.4 的理由。实现者的关切 1–3、5（规定的四句比代码宽）→ 改准；关切 4（M2 交接第 3 节说 `TestWorkItemsArePutUnderAWorkItem` 断言一个时刻，它没有；P16.a 活过四个组合测试）→ 交接点名真断言它的单元测试和 I8，T16.1（单元；I8）（`62c5d0b2`）。理由：交接只引测试断言了的。Interfaces 加的一句：对 HEAD 成立就接受（评审核对过）。
  - 评审：Important ×2（plan 写定的，Plane 一侧，抄自设计的格子）：差异清单 :250 子工作项数的 Plane 一列、:252 关联的动态的 Plane 一列与 Plane 的代码不符。裁定（`f8e4c0d7`，只改文档，控制者核对 Plane 的代码：`db/models/issue.py:92-101`、`issue_activities_task.py:1355-1378`）：两格和它们抄的设计的几行（3.13 第 23 行、4.17）照 Plane 的代码写。理由：Plane 的行为是要照实写的事实，不是设计的决定。代价：三处两个从句。

**整分支评审之后**
- **I1 报负责人**（负责人的 /goal：设计层面的访客规则的语义是负责人的决定点）：控制者用平实的话报告，建议"同调用者得到的回答比"，另一种是保留现在的行为、把 spec 第 3 节第 9 条、第 4 节第 4 条和契约收窄。**负责人裁定 A（2026-10-24）**：修改的父工作项同调用者得到的回答比；`OwnOnly` 之下他看不到的现在的父工作项（`HiddenFrom`）算没有：指名它像指名不存在的一样被拒绝（422，逐值相同），送回遮住的 `null` 什么都不改，移到他看得到的父工作项下照旧写。
- 一个修复轮（`final-fix-brief.md`）的控制者的裁定：I1 照裁定（从根上改，一个谓词，只在需要时在锁下读，单元和组合各一例，变异行"同存着的比"，契约和 `make gen`，spec 第 3 节第 9 条、第 5 节 P14 一行，设计 3.4、3.9（修复轮只改了 3.4，3.9 的一般写法照旧成立，范围复审核对过），差异清单的访客一行，M3 交接第 3 节）；Minor 1（`parentsBefore` 经 `ascending`）；Minor 2（两句拒绝的文字加"that you see"，期望随之）；分诊 1（优先级逐个写出，加单元一行）、2（`idArray`）、3（子工作项的路由写进说明）、4（T4.21、T4.22 记下找到的层；T4.17、T5.28 进 A.3 的存储；T5.27 加组合的数据；T2.12 加单元一例；T6.5 加单元一例，不是 A.3 的一行；T13.2、T13.7、T13.8 的 I9 一层）、5（A.3 的数随之，A.6 照旧，第 3 节第 20 条照旧）；分诊 6 接受（Task 14 的裁定）；7、8 记给 M4 的收尾；9 交给本文（spec 第 4 节的框、设计第 15 节一行）。理由：评审的发现，和"每句话对 HEAD 都对"。
- 修复轮之后的裁定：实现者的关切（T3.13 的编辑在 Task 4 的树上不再能应用：那里原来的父工作项的读在 Task 4 的 `parentBefore` 里，Task 5 才有 `parentsOf`；没有哪一遍在那棵树上跑它，它的层是 Task 3 自己的和最终的树的，两处都能应用）接受，本文记下（第 7 节）。第一次运行的超时：环境（别的会话的容器，负载 14），第二次都通过；范围复审核对日志并在 Docker 空闲时重跑（第 7 节）。
- 合并说明的观察（控制者的笔记）：合并用 `commit-tree`，第一个父提交是 `origin/main`；`origin/main` 若已越过 `f8696490`，合并的树不一定是分支的树，合并前先核对。

## 4. 整分支评审的发现与处理

结论"一轮修复之后可以合并"：Critical 0、Important 1、Minor 2；分诊九项。评审逐条核对过：
- 一条取锁路径：每个写经 `Locks.run`（`lock.go:90`）：预检（不加锁读，访客规则先于任何角色：`HiddenFrom` → 写的 not-found）→ `gate.Enter`（在取连接之前）→ 一个事务：`lockAndDecide`（项目行 `FOR NO KEY UPDATE`，锁下重读、再判定）→ 用例的正文；三种范围：一个项目、跨项目（自己的和到达的项目，`ascending`，`LockMany` 一条语句 `ORDER BY id`）、两端任一个（两端的项目，`ascending`）。
- 逐个写的访客规则：创建、修改、加子工作项、加关联、删除关联、两个列表，看不到的与不存在的同一个答复；**修改的现在的父工作项是唯一的例外（I1）**。
- 把闸门的位置、池的连接和行锁都当作资源，重推死锁：闸门只在什么都不持时进、键升序一次进齐；项目行升序一条语句；工作项一侧的行只在它的项目的 N 之下写；关联行是另一个项目的锁唯一到得了的行，删除的一方按 `id` 升序先锁，加、删关联的一方持两端的 N；外键取 `KEY SHARE`，与 `FOR NO KEY UPDATE` 不冲突。没有环。
- 门禁、竞争的运行、plan 的重放、变异的抽样（第 1 节）；没有一句仍把 P3 的工作说成将来的（I1、分诊 3、4 的几处之外）。

| 发现 | 处理 |
|---|---|
| **I1**（Important）`issue/domain/patch.go:124` 拿补丁的 `parent_id` 同存着的父工作项比（`update_issue.go:84-87`、`parents.go:45-65`）：在项目里是访客的工作区管理员（`OwnOnly`、能 `issue.update`）读自己的工作项得到遮住的 `parent_id: null`；他再指名那个看不到的父工作项答 200（什么都不改），指名别的看不到的或随机的 id 答 422：分得出"X 是不是我的工作项的父工作项"，违背 spec 第 3 节第 9 条和第 4 节第 4 条；送回他得到的 `null`（读-改-写的客户端、代理）把工作项从它下面拿走（walt 的第 5 行，`updated_at` 变了，这本身也透露有一个隐藏的父工作项），spec 第 5 节只把它交给页面（P14），契约没说。评审的探针（`p3final/logs/probe.log`）证实；控制者核对了 `patch.go:124`、`parents.go:45-65` | `85f88b91`（负责人裁定 A）：`issue/app/parents.go:48` 的 `asAnswered`：补丁设了父工作项、锁下判定的 `Grant` 是 `OwnOnly`、工作项有父工作项时，锁下经 `parentsOf`（已有的 `GetIssues`，不另加查询）读一次父工作项，访客规则的谓词 `HiddenFrom` 藏起的算没有；`update_issue.go:85` 把这样的工作项交给 `domain.ChangesOf`（它的签名和 `patch.go` 不变），什么都不改时照旧答 `seenOne`（遮住），写照旧从存着的工作项写；读到的父工作项交给 `parentChange`（`:71`，`update_issue.go:172`），第 5 行原来的父工作项是存着的那个，不再读；没有 `OwnOnly` 的调用者照旧，同样的调用、同样的次序。契约 `IssueUpdate.parent_id`（`api/modules/issue.yaml:850-855`）写上比较的规则，`make gen` 只改三个生成物的描述行。测试：单元 `TestUpdateIssueComparesTheParentAsItsCallerIsAnswered`（`parents_test.go:232`：指名藏起的 `up` 与随机 id 的拒绝 `reflect.DeepEqual`、什么都不写；`null` 不读时钟、什么都不写；移到 `mine` 下第 5 行是 `WEB-3 → WEB-10`），`TestAParentsFailingReadEndsTheWrite`（`:280`）加比较的读失败；组合 `TestAHiddenParentCountsAsNoneInAChange`（`bootstrap/issue_parents_test.go:246`）。变异 T4.23（同存着的比）修复之前在 `issue/app` 和整个 `bootstrap` 活着，之后在单元和组合（`issue_parents_test.go:255`）被发现；T4.24（比较的读失败当作没读：单元，按性质，A.3）。`dcb99363`：spec 第 3 节第 9 条（:202）、第 5 节 P14 一行（:261）、2.1、2.6，设计 3.4（:215），差异清单的访客一行（:242），M3 交接第 3 节（:155），都标"负责人裁定 A，2026-10-24"；第 4 节第 4 条照原样就成立 |
| **Minor 1** `parents.go:74-85` 的 `parentsBefore` 重写了 `ascending` | `85f88b91`：`parentsOf`（`parents.go:121`）收集每个父工作项的 id，`GetIssues(ctx, ascending(ids))`（`:132`）；`parentsBefore`（`:101`）给它们起名字。先排序再去重与先去重再排序相同，行为不变；T3.13、T5.31 的锚没有动 |
| **Minor 2** `hierarchy.go:36`（`ParentNotAllowed`）、`:57`（`SubIssueNotAllowed`）的文字没有"that you see"（`relation.go:69` 和契约有） | `85f88b91`：两句都是"…of the project that you see, not archived, and neither this one nor under/above it"；各仍是一句固定的文字，看不到的与不存在的照样同一个答复；比较它们的测试随之改（`hierarchy_test.go:60`、`adapter/http/sub_issues_test.go:142-143`、`bootstrap/issue_parents_test.go:18-19`），没有故事或前端比较这些文字 |
| **分诊 1** P2 的优先级的循环（`issue/domain/issue_test.go:47`、`patch_test.go:39` 遍历生产的 `priorities`） | `85f88b91`：两张表逐个写出 urgent、high、medium、low、none（Task 2、3 的块，写在 P2 的文件上）；T2.15（优先级少了 medium）修复之前活过 domain、app、http 和整个 `bootstrap`，之后在单元（`TestCheckNewIssueAccepts`）和组合（Task 6 起：`TestTheGuestRuleHoldsForSubItems` 里 alice 把 walt 的 Mine 改成 medium，`issue_sub_issues_test.go:197`）被发现，不是只在一层 |
| **分诊 2** `bootstrap/permission_matrix_issue_updates_test.go:93-94` 手拼三个 id 的数组 | `85f88b91`：经 `idArray`（Task 11 的一个块，写在基点的文件上），体逐字节相同；文件表、spec 2.1 的 Task 11 一行和计数随之（165 个） |
| **分诊 3** `issue/app/sub_issues.go` 的两个用例的说明不写路由（另外三个写） | `85f88b91`：`:13-14` 写 GET、`:58-59` 写 POST `/api/v0/issues/{issue_id}/sub-issues`（Task 6 的块） |
| **分诊 4** A.3 之外只在一层的七行 | `85f88b91`、`dcb99363`：T4.21 记下单元一层（`seen_test.go` 的 `TestAnAnswerNamesTheParentTheCallerSees`）；T4.22 记下 Task 6、Task 11 起的组合层（`TestWorkItemsArePutUnderAWorkItem`、`TestWorkItemsAreRelatedAcrossProjects` 和矩阵的格子）和 I8（`i8-sub-issues.spec.ts:72`，Task 15 起）；T4.17、T5.28 进 A.3 的存储"读"一项（T4.17 的理由由修复轮的组合探针支持：它活着）；T5.27 加组合的数据（Task 6 起，alice 最后改 walt 的 Mine）；T2.12 加单元一例（`delete_issue_test.go:48` 的测试里 alice 是只做访客的管理员，carol 的工作项在她的子树里）；T6.5 加单元一例（`sub_issues_test.go:44` 的测试，已归档的工作项的子工作项）；T13.2、T13.7、T13.8 记下"故事 `i9-relations`（Task 15 起）"（修复轮自己跑：都在 `i9-relations.spec.ts:147:65` 被发现，`e2ekills.py` 0 个可疑）。plan 的行：:2152、:5565、:5569-5572、:6927-6928、:8239、:17035、:17040-17041 |
| **分诊 5** A.3、A.6、第 3 节第 20 条在分诊 4 之后 | `dcb99363`：A.3 的存储"执行中加的 4 个"（T5.33、T8.15、T4.17、T5.28），修复轮另加的 T4.24 让单元"执行中加的 18 个"、端口的失败 18 个；A.6 和第 3 节第 20 条不改（它们的两类，闸门的次序和 `eitherUnder`，在组合一层都看不到：T10.23、T13.4 活过整个 `bootstrap`） |
| **分诊 6** M3 的世界手拼项目成员的请求 | 接受、不改：Task 14 的裁定（M3 的世界经它们自己的世界，不是 P3 的文件）；要改是 M4 收尾的测试整理 |
| **分诊 7** P1b 交给收尾的手写 `55P03`；`workspace` 的 `waitsFor` | 记给 M4 的收尾（第 6 节） |
| **分诊 8** 空的 `m4-p1b` worktree 和分支 | 记下（第 7 节）：删不删由负责人定 |
| **分诊 9** 台账的其余：:273（闸门的次序只在单元一层）重看，T10.23 活过整个 `bootstrap`，A.3 成立；:308（F1）接受，见 Minor 1；合并的文档一步 | spec 第 4 节的框、设计第 15 节 P3 一行：本提交 |

**修复轮**（`85f88b91`：21 个文件，代码、测试、契约、生成物和重新组装的 plan；`dcb99363`：4 个文件，spec、设计、差异清单和 M3 交接）：
- 检查（`final-fix-report.md`）：`make lint-go` 两段 `0 issues.`；`make test` 第一次一个 `FAIL`（四个 M3 的成员关系的矩阵格子在别的会话的 testcontainers 运行时 32 秒超时），第二次没有；`make lint-web` 没有命中，`make knip`、`make test-web` 通过；`make e2e` 第一次 121 个通过（A9、P3、S8 三个页面故事在负载 14 时超时，另有 A4 的两个），第二次 124/126；Task 14 的正则 `-race -count=5`：170 次顶层 PASS，`platform/gate` 50 次，没有 40P01、DATA RACE、死锁。变异：Pass A 35 行 45/45 层，Pass B 6 行 10/10，Pass C（最终的树）52 行 106/106（读 `patch.go`、`update_issue.go`、`parents.go`、`hierarchy.go`、`app/sub_issues.go` 的 47 行全在其中），没有 INFRA；修复之前的探针 B-T4.23、B-T2.15 和组合探针 P-T4.17 照预期活着。plan：807 个块都对，逐 Task 重放之后只差 13 个生成物，`compose_plan.py` 逐字节重现；`mutcheck` 247 个变异、0 个问题。
- 范围复审（opus，`f8e4c0d7..dcb99363`，`$M4TMP/p3final/rereview.md`，副本在 `p3final/rereview/`）：八条裁定都已落实、对 HEAD 都对；新的问题 0 条。它核对了：I1 的读在锁下、用锁下判定的 `Grant` 和那一个谓词，只在需要时读、只读一次，指名藏起的父工作项与随机 id 的回答字节相同，`null` 不写、没有动态、`updated_at` 不变，没有 `OwnOnly` 的调用者同样的调用和次序，`lock.go`、`lock_across.go`、`lock_either.go`、`seen.go` 和别的模块不在差异里；T4.23 在单元和组合被发现、修复之前活着；T4.24 是真的只在单元一层的端口失败；分诊 1–5 的每一行，在 HEAD 上重跑 T2.12、T6.5、T2.15、T4.23、T4.24；plan 的重放（807 个块：replace 741、whole 7、file 59，只差 13 个生成物，逐字节组装），`spectable` 0、`filetablecheck` 0、`mutcheck` 247/0、`countfiles` 165；`doclinks.py` 只有设计的 4 个已知的链接；`make lint-go` 两次 0；第一次的超时在 Docker 空闲时重跑：69 个成员关系的格子通过（超时的四个各 0.14–0.21 秒），A9、P3、S8 的故事 10 个通过：环境。四条观察，都不需要改：(1) `parentChange` 的 `old` 为空既表示"没读"也表示"没有父工作项"：`parentsBefore` 对没有父工作项的工作项什么都不读，说明写了；(2) "回答的样子的父工作项"在两处判定（`seen.hideParents` 给回答，不加锁，读不到算藏起；`asAnswered` 给比较，锁下，读不到是错误），由一个谓词连着，锁下两者不会不同；(3) 时间差：`OwnOnly` 的修改设 `parent_id` 时只在有父工作项时多一次带索引的读，同 `hideParents` 的已有的不对称，不在 P3 的性质里（第 7 节）；(4) `update_issue.go:60` 的说明换行不齐，只是外观。

## 5. 最终实现与 plan 的差异

plan 是"执行时的 plan"：每个修复轮也改了它的 Task 的块，代码块写的是提交的代码（第 3 节）。所以这里不列逐个提交的漂移表，只写复现的结果、执行中改了的生产代码、注明日期的记录在 HEAD 上的数和执行中加的变异。比较的是修订之后第一次提交的 plan（`650dc819`）。

**复现**（本文的作者，`$M4TMP/p3sdd/rd/`）：
- 一份新的解压 `$M4TMP/p3tools/base-f8696490.tar`（`rd/base`）上 `node planapply.mjs <plan> check`：HEAD 的 plan "807 blocks {"replace":741,"whole":7,"file":59}; all ok"，`650dc819` 的 plan "680 blocks {"replace":616,"whole":5,"file":59}; all ok"。
- HEAD 的 13 个生成物与 plan 最后的钉值相同（`rd/pins.py`：SHA-256 和行数，13/13）：`api/dist/openapi.yaml` `457981a6…` 3,672 行（plan :17165）、`issue/…/http/gen/server.gen.go` `cf8a4091…`（:17166）、`schema.gen.ts` `5d6d5e24…`（:17167）、`bodyshape.gen.go` `de9219db…`（:12672）、`delete_issue.sql.go` `46638be2…`（:9330）、`delete_rows.sql.go` `47522c36…`（:9331）、`models.go` `d4082fbd…`（:9332）、`issues.sql.go` `b918552b…`（:3830）、`parents.sql.go` `651cd6d0…`（:2293）、`update.sql.go` `909672b1…`（:2294）、`sub_issues.sql.go` `fbbb6493…`（:5638）、`relations.sql.go` `e20111ef…`（:15899）、`project/…/issue_ports.sql.go` `7f5da04e…`（:14166）。
- 逐 Task 的完整重放由范围复审在 HEAD 上做过（第 4 节），整分支评审在 `f8e4c0d7` 上做过（780 个块，`make gen` 之后 0 个差异）；本文不再重做。
- 执行中改过的设计（`git diff 650dc819 HEAD`）：3.4（:215，修复轮）、3.5 交错 26（:351，`ac426aeb`）、3.13 第 23 行（:523，`f8e4c0d7`）、4.17 的子工作项数和关联的动态两行（:1090、:1092，`f8e4c0d7`），和本提交的第 15 节一行。

**执行中改了的生产代码**（`git log --name-status f8696490..dcb99363`，非测试的 Go、SQL、契约和配置，各 Task 的第一个提交之外；`rd/prodfiles.py`，每个差异逐个读过）：
- **改了行为的**：都在整分支评审之后的修复轮 `85f88b91`：
  - I1：`asAnswered`（负责人裁定 A）：`OwnOnly` 的调用者指名藏起的现在的父工作项答 422（与不存在的同一个值），送回 `null` 什么都不写；契约 `IssueUpdate.parent_id` 的描述随之。
  - 两句拒绝的文字（`ParentNotAllowed`、`SubIssueNotAllowed`）加"that you see"：422 的字段错误的文字变了，看不到的与不存在的仍是同一句。
- **语句的写法**：`2fff6efb`（Task 8 的修复轮 1）：`DeleteProjectRelationRows` 的 `OR … IN (SELECT …)` 改为 `UNION ALL` 的两支，同样的行、同样的 `id` 次序和锁、仍是一条语句，变的是计划和代价（第 3 节；spec 第 7 节）；`c1f41480` 只改它的注释。
- **错误的文字**（内部错误，回答照旧是 500）：写个数、不写 id：`59468ade`（删除子树）、`1a9e2bdd`（`GetIssues`、`SubIssueCounts`）、`93bf8714`（`SetParents`、加子工作项的读回）、`6c67e11e`（`LockIssueProjects`）、`045fb636`（`Related`、`InsertRelations`、`MarkChanged`）、`8a466e05`（加关联的读回）、`f80c034f`（`ArchivedIssueProjects`）。
- **改了代码、行为不变**：`9eabe3d4`（一个泛型的 `same[T]` 取代 `sameID`、`sameDay`）；`680dada5`（`parentsBefore`：修改改用 `GetIssues` 读原来的父工作项，答复相同）；`5dfc7848`（`checkSomeIDs`）；`2be08ad6`（`held.own()`；回答的 `grants` 从空的开始：自己的项目那一项从没被读）；`0badb50b`（`hideParents` 和创建的订阅者读 `ascending`：同样的集合和次序，输入不变）；`07b97170`（删除关联用 `h.own()`）；`85f88b91`（`parentsOf` 经 `ascending`，Minor 1）。HEAD 上 `ascending` 有七个调用者：`reachAll`、`hideParents`、`projectsBut`、创建的订阅者、`eitherOf`、删除关联的最后修改、`parentsOf`。
- **契约的描述**：`1a9e2bdd`（`IssueDetail.parent_id`，Task 6 的 `Issue.parent_id` 同样）、`85f88b91`（`IssueUpdate.parent_id`）。
- **只改注释和说明**：`4df19c98`（`ParentActivity`）、`07516159`（`Issue.SubIssuesCount`）、`a333f755`（`slices.Clone` 的理由、`enter`、`bootstrap` 的 `writeGate.Enter`）、`6b0b73bf`（`Keyed.Enter`）、`5dfc7848`（`relations.sql` 的 `MarkChanged`）、`2be08ad6`、`07b97170`（`own`、`end`、`run`）；用例的说明不指名之后的 Task 才加的路由（`5214e682`、`8a466e05`、`07b97170` 去掉，Task 11、14 的块和 `85f88b91` 写上）。
- **两端任一个的拒绝次序两个方向都钉住**（T13.33、T14.5）改的是测试：代码的次序本来就照裁定，测试的数据只走一个交叉方向。
- 测试的设施：`f6b6f212` 的 `pgtest.RowWaits`（用 `postgres.LockNotAvailable`，找不到行时测试失败；共用的 `fatalOf` 带上 `t`，`lockwait_test.go`、`lockwait_behind_test.go` 随之进 Task 8），不是生产代码。

**执行中并掉或加的测试帮手**（都是共用、扩展，取代复制的或会被复制的写法；各 Task 的块随之重新生成）：
- 组合（`bootstrap`）：`adminGuest` 从 Task 2 移到 Task 1 的 `issue_world_test.go`（`d3353443`）；`webWorkItems` 的准备和一个终态检查给三个交错共用（`63b7ae6f`）；矩阵的检查经 Task 6 的 `listIn`（`732d99c4`），`relatedTo`、`listsRelations` 经 `listIn[answeredRelation]`（`98292afd`、`f80c034f`）；`seedIssues` 调 `relate`（`2fff6efb`）；`names` 并进 `issue_world_test.go`，`workItemStamps`、`parentRows` 读它（`98292afd`）；`idArray` 给请求体的 id 数组，`relating`、`subIssueIDs` 读它，cascade、pool 的循环和空体并掉（`0fe250aa`），P2 的 `permission_matrix_issue_updates_test.go` 的三个 id 的数组也经它（`85f88b91`）；`relateCells`（改名，不遮住包的 `relates`，`0fe250aa`）；`createdProjectFrom`（`f80c034f`），M3 的三处并进它或 `createdProject`（`6020f200`）；`keyOfRow`，`projectOfRow`、`workspaceOfRow`、`listsRelations` 读它（`6020f200`）；`makeMember`，世界的九处读它（`51d3375d`）。
- 单元（`issue/app`）：`wantStopped` 一个核心，`wantStoppedAt`、`wantStoppedOn` 经它（`8cffaa2b`）；`sub_issues_test.go` 用共用的 `wantStopped*` 和动态的核对（`5214e682`）；假实现的 `lost` 扩展、`losing` 删去（`680dada5`），加关联的 `losingRelations` 改用共用的 `lost`、`lose`，`wantActivityRows`、`wantRefused`，一个本地的 `wantNothingWritten`（`8a466e05`、`0c3a4f76`）；`wantNothingWritten` 扩展到假实现记下的每个关联的写，假实现插入的关联记在 `inserted`（`07b97170`）；`fromNear` 的数据（`65da0827`）；管理员·访客的 `Grant` 一个值（`07516159`），修复轮移到 Task 2 的 `guest_rule_test.go`（`85f88b91`）。
- 存储和端到端：删除测试的 `newRelation`、`relate`、`relationRow` 带存储的类型（`f6b6f212`）；`assert/issue.ts` 的 `activityRow`（`changed`、`unrelatedRow` 和 `createdFor` 的创建一行经它）和 `madeBy`（I8、I9 共用，`asMade` 删去）（`29623915`）。

**改写的句子**（只改措辞，不改行为；裁定都在第 3 节）：测试的说明（`59468ade`、`63b7ae6f`、`5dfc7848`、`51d3375d`）；端到端的注释（`29623915`、`4944e989`：I9、I15 的注释，I8 的 Plane 说明只挂在 Tag the build 的一行）；生产代码的注释（上面"只改注释和说明"一条）；契约的描述（`1a9e2bdd`、`85f88b91`）；plan 和 spec 的句子（`f4b34b97`、`74182225` 的数，`9eabe3d4` 的 `Ancestry`，`4637c60f` 的指针一句，`103dacf4` 的 A.3，`0fe250aa` 的竞争的名单和大小一句，`51d3375d` 的交错 26 和 2.16，`c1f41480` 的第 7 节）；设计、差异清单和交接（`ac426aeb`、`62c5d0b2`、`f8e4c0d7`、`dcb99363`）。

**注明日期的记录在 HEAD 上的数**（spec 附录 A 照裁定只由执行的各轮改过 A.3，其余照旧；数都是本文的作者在 HEAD 上量的：`rd/planlines.py`、`rd/blockfiles.py`、`rd/filecount.py`、`rd/rows.py`、`rd/later.py`、`rd/onelayer.py`、`rd/sizes.py`、`rd/tablelines.py`）：
- **A.1**（:304）：
  - 量法与 A.1 相同（各 Task 从它的标题到下一个标题，Task 16 到文件末尾）；在 `650dc819` 的 plan 上量出的数与 A.1 逐个相同。A.1 → HEAD：Task 1 之前 193 → 197；1 877 → 939；2 938 → 1,022；3 1,511 → 1,558；4 1,471 → 1,861；5 1,170 → 1,359；6 1,215 → 1,309；7 829 → 887；8 1,366 → 1,528；9 660 → 703；10 1,079 → 1,146；11 1,361 → 1,543；12 1,560 → 1,792；13 1,068 → 1,225；14 1,307 → 1,458；15 1,130 → 1,208；16 175 → 179。plan 17,910 → 19,914 行（`wc -l`），多 2,004 行，都是修复轮的块、执行中加的变异行和各 Task 的说明。
  - 长过约 1,500 行的：A.1 写 Task 12（1,560）和 Task 3（1,511）；HEAD 上 Task 4 1,861、Task 12 1,792、Task 3 1,558、Task 11 1,543、Task 8 1,528。最大的增长在 Task 4（M1 的遮挡之后又有修复轮 1 的契约和连接的测试、整分支评审之后的 I1）和 Task 12（`createdProjectFrom`、`keyOfRow`、`ascending` 的两处、T12.19、T12.20）。
  - 文件：A.1 写手写的 153 个（新建 59、修改 94），HEAD 上 plan 的块写 165 个（59、106），spec 2.1 写 165；多出的 12 个都是已有的文件：`issue/domain/dates.go`（Task 3 的修复轮 1）、`pgtest/lockwait_test.go`、`lockwait_behind_test.go`（Task 8，`fatalOf` 带上测试）、`bootstrap/issue_cascade_test.go`、`issue_pool_test.go`（Task 11 的修复轮 1，`idArray`）、`demotion_test.go`、`ending_world_test.go`、`project_wiring_test.go`、`issue_identifier_test.go`、`project_writes_test.go`（Task 12，`createdProjectFrom`）、`issue_updates_test.go`（Task 14 的修复轮 1，`makeMember`）、`permission_matrix_issue_updates_test.go`（整分支评审之后的修复轮，`idArray`）。`git diff --name-status f8696490 HEAD` 列出的手写的是 163 个（59、104）：`e2e/fixtures/assert/project.ts`、`workspace.ts` 在 Task 8 改（`withoutRows`）、Task 11 还原，最后与基点相同。生成物 13 个；文档 6 个和设计（本提交加本文）。
- **A.3**（:337–371，执行中保持为真）：HEAD 上 A.3 写的数与 plan 的表一致（`rd/onelayer.py`：表里只在一层的每一行 A.3 都点名）：只在单元一层的清扫 16 个、执行中加的 18 个（端口的失败 11、闸门的次序 3、一条语句里的次序 2、调用者的切片 1、判定的次数 1），另有 T5.31（与 T3.13 同一个编辑，不另算），表里 35 行；只在存储一层的清扫 17 个、执行中加的 4 个，表里 21 行；只在组合一层的清扫的 6 个；模式 4 个（T8.1–T8.4）；守卫 1 个；只在端到端的没有。A.3 末尾"端到端发现的 39 个变异"是清扫时的数：HEAD 的表里 48 行有端到端一层，每一行另有 Go 的一层。
- **A.4**（:373–379）：矩阵对 HEAD 仍成立：1,455 个叶子、P3 的 196 格、P1a–P2 的 393 格、M3 的 865 格（本文的作者的运行，第 2 节；顶层 1.58 秒，A.4 写 1.54 秒）。竞争的名单：A.4 写原型 81 个、执行之后 82 个，plan 的 Task 14 跑 34 个（A.4 自己写了执行之后的数）。A.4 的端到端 123/126 是原型上的（不是 git 检出，S3 失败）；worktree 里是 124/126。
- **A.5**（:383）："680 个块"今天是 807 个（{"replace":741,"whole":7,"file":59}）；`Run:` 的行与修订时相同（119 行，A.5 的 132 个含提交之后的 13 个 `make gen-check`）。
- **A.6**（:387 和变异表）：变异 192 个，今天 247 个（`rd/rows.py`：表的行与 `mutants_p3*.py` 的定义一一对应）：执行中加的 55 行（5.1），没有去掉的。各层发现的（每行在它列的每一层各算一次）：组合 152 → 185，单元 131 → 173，存储 48 → 61，端到端 39 → 48，模式 4 → 4，守卫 1 → 1。只在一层的：单元 18 → 35，存储 17 → 21，组合 7 → 6，模式 4，守卫 1（见 A.3）。有"（Task n 起）"一层的 105 → 123 行：Task 3 起 3 → 2（T2.10 在 Task 2 的树上就被发现），Task 4 起 1，Task 6 起 27 → 33，Task 11 起 35 → 43，Task 12 起 1，Task 14 起 28 → 35，Task 15 起 39 → 46；表里点名的端到端的发现 41 → 50 次。各 Task 的"本 Task 的检查也发现较早的 Task 的变异"一句点名的数与这些相同（Task 3 2、Task 4 1、Task 6 33、Task 11 43、Task 12 1、Task 14 35、Task 15 46）。执行中每行在它的 Task 的树和最终的树上跑过、被发现（各 Task 的报告和台账；整分支评审的抽样 59/59；修复轮的 Pass A、B、C）；本文没有重跑变异。清扫的表里说"清扫时"的句子照旧成立；"原型的每个新的 Go 测试至少被一个变异弄失败（98 个）"是原型的数，没有在 HEAD 上重数。
- **Global Constraints 的大小**（plan :24）：Task 11 的裁定之后它量的是执行之后的最终的树；HEAD 上 `wc -l`（`rd/sizes.py`）与它写的每个数相同：`api/modules/issue.yaml` 857（修订时写 848，Task 4 之后 854，修复轮之后 857）、`project.yaml` 1363、`workspace.yaml` 872、`tools/keywords.json` 2670、`permission_matrix_test.go` 410（基点 406）、`issue_interleaving_test.go` 407（基点 393）、`migrations/schema_test.go` 401（基点 400）；改过的其余代码文件最长的 `bootstrap/gate_test.go` 399、`issue/app/fakes_test.go` 398；新建的最长的 `issue/adapter/postgres/relations_test.go` 327、`issue/app/sub_issues_test.go` 317、`issue/app/parents_test.go` 314。修订时（原型上量的）写的是 `fakes_test.go` 398、`bootstrap/ports_test.go` 391，新建的 327、327（`sub_issues_test.go`）、306（`issue_interleaving_relations_test.go`），今天是 391、317、297。超过约 400 行的手写文件只有三个契约、`keywords.json` 和三个点名的例外。

### 5.1 执行中加的变异

执行中加的变异是 plan 的变异表的行（不进 spec 附录 A 的表；只在一层的进 A.3），定义在 `$M4TMP/p3tools/mutants_p3_{a,b,c,d}.py`（每行前面的注释写明是哪一轮加的），中文在 `mutants_zh_p3.py`，结果在覆盖文件。各 Task 的数目和位置（行号是 HEAD 的 plan 的，本提交不改 plan）：

| Task | 表的行 | 原型的 | 执行中加的 | plan 的表（行号） |
|---|---|---|---|---|
| 1 | 11 | 10 | T1.11 | :1123–1133 |
| 2 | 15 | 11 | T2.12–T2.15 | :2141–2155 |
| 3 | 16 | 15 | T3.16 | :3696–3711 |
| 4 | 24 | 14 | T4.15–T4.24 | :5549–5572 |
| 5 | 33 | 26 | T5.27–T5.33 | :6901–6933 |
| 6 | 6 | 4 | T6.5、T6.6 | :8235–8240 |
| 7 | 11 | 9 | T7.10、T7.11 | :9119–9129 |
| 8 | 19 | 14 | T8.15–T8.19 | :10639–10657 |
| 9 | 20 | 15 | T9.16–T9.20 | :11341–11360 |
| 10 | 27 | 22 | T10.23–T10.27 | :12480–12506 |
| 11 | 5 | 5 | — | :14043–14047 |
| 12 | 20 | 18 | T12.19、T12.20 | :15820–15839 |
| 13 | 33 | 25 | T13.26–T13.33 | :17034–17066 |
| 14 | 5 | 4 | T14.5 | :18518–18522 |
| 15 | 1 | 0 | T15.1 | :19730 |
| 16 | 1 | 0 | T16.1 | :19914 |
| 合计 | 247 | 192 | 55 | |

每行（"类别"是 `mutants_p3*.py` 的 `class`；"层"照 plan 的表，"Task n 起"是较晚的 Task 的检查；A.3 指只在一层、写在 A.3 的）：

| 行 | 改坏 | 类别 | 层 | 哪一轮加的 |
|---|---|---|---|---|
| T1.11 | 删除工作项在锁之前读子树 | `subtree` | 单元；组合（交错 13，Task 6 起） | Task 6，评审之前（实现者的探针） |
| T2.12 | 删除者只看自己的工作项时，子树去掉他看不到的子工作项 | `guest` | 单元（整分支评审之后的修复轮加的）；组合（`TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee`） | Task 1，评审之前（N3 的探针） |
| T2.13 | 父工作项只能是调用者自己建的 | `guest` | 单元；组合（`TestTheGuestRuleHoldsForAParent`） | Task 2，评审之前 |
| T2.14 | 带父工作项的创建写第 5 行 | `activity` | 单元；组合 | Task 2 的修复轮 1（评审） |
| T2.15 | 优先级少了 medium | `priority` | 单元；组合（Task 6 起） | 整分支评审之后的修复轮（分诊 1） |
| T3.16 | 移动只允许到没有父工作项的工作项下 | `parent` | 单元；组合 | Task 3 的修复轮 1（评审 Important） |
| T4.15、T4.16、T4.19、T4.20 | 修改、按编号取、什么都没改的修改、什么都没改的归档或恢复的回答数子工作项失败时略过 | `port` | 单元（A.3） | Task 4，评审之前（实现者的关切） |
| T4.17 | `GetIssues` 照读到行的次序回答，不按 `id` | `store` | 存储（A.3，修复轮写进） | Task 4，评审之前 |
| T4.18 | 最后修改子工作项的账户算作"自己的" | `guest` | 存储；组合 | Task 4，评审之前 |
| T4.21 | `hideParents` 经连接池读父工作项 | `connection` | 单元（修复轮记下）；组合 | Task 4 的修复轮 1（评审 Minor） |
| T4.22 | `GetIssues` 经连接池读 | `connection` | 组合（一个连接的池；Task 6、Task 11 起的组合测试和矩阵，修复轮记下）；端到端（I8，Task 15 起） | Task 4 的修复轮 1 |
| T4.23 | `updateIssue` 拿要设的父工作项同存着的比 | `guest` | 单元；组合（`TestAHiddenParentCountsAsNoneInAChange`） | 整分支评审之后的修复轮（I1，负责人裁定 A） |
| T4.24 | 读调用者得到的父工作项失败，当作没读 | `port` | 单元（A.3） | 整分支评审之后的修复轮 |
| T5.27 | `SubIssues` 给只看自己的访客回答他最后修改的 | `guest` | 存储；组合（Task 6 起，修复轮加的数据） | Task 5，评审之前 |
| T5.28 | `SubIssues` 对同一时刻建的照读到的次序回答 | `store` | 存储（A.3，修复轮写进） | Task 5，评审之前 |
| T5.29 | 只是访客的管理员列子工作项要所有人的 | `guest` | 单元；组合（Task 6 起） | Task 5，评审之前 |
| T5.30 | 写的范围里访客规则对只是访客的管理员不成立 | `guest` | 单元；组合（Task 6 起也在加子工作项的测试） | Task 5，评审之前（实现者加的） |
| T5.31 | 不在的原来的父工作项被略过 | `port` | 单元（A.3 写明与 T3.13 同一个编辑，不另算） | Task 5 的修复轮 1（评审） |
| T5.32 | 加子工作项按请求的次序移动 | `order` | 单元（A.3） | Task 5 的修复轮 1 |
| T5.33 | `SetParents` 把没写到的当作写了 | `store` | 存储（A.3） | Task 5 的修复轮 1 |
| T6.5 | `listSubIssues` 对已归档的工作项回答空的 | `archived` | 单元（修复轮加的）；组合 | Task 6，评审之前 |
| T6.6 | `SubIssues` 不要已归档的工作项下的 | `archived` | 存储；组合 | Task 6，评审之前 |
| T7.10 | `Enter` 不另存一份键 | `gate` | 单元（A.3） | Task 7，评审之前（探针 P7.b） |
| T7.11 | 写闸门把一个写的几个项目倒过来进 | `gate` | 单元（A.3） | Task 7 的修复轮 1（评审 Minor） |
| T8.15 | `pgtest.RowWaits` 对不存在的行答"不等" | `lock` | 存储（A.3） | Task 8，评审之前（P8.a） |
| T8.16、T8.17、T8.18 | 删除的工作项的关联只删 `relates_to` 的、略过 `duplicate` 的；删除的项目的一批略过 `duplicate` 的 | `deletion` | 存储；组合（按目录的删除测试或删除项目的关联测试，另有交错 11，Task 14 起） | Task 8，评审之前（P8.b–P8.d） |
| T8.19 | 删除的关联只算子树的第一个工作项的 | `deletion` | 单元；组合 | Task 8 的修复轮 1（评审 Minor） |
| T9.16 | 只读 `relates_to` 一种类型的关联 | `store` | 存储；组合（Task 11、Task 14 起） | Task 9，评审之前（派发带上的数据） |
| T9.17 | 加关联的动态不论类型都写成 `blocking` | `activity` | 单元；组合（Task 11、Task 14 起） | Task 9，评审之前 |
| T9.18 | 加子工作项、加关联都可以一个也不列 | `check` | 单元；组合（Task 11 起也在关联） | Task 9 的修复轮 1（评审 Minor） |
| T9.19 | 加关联不能给 `duplicate` 类型 | `check` | 单元；组合（Task 11 起） | Task 9 的修复轮 1 |
| T9.20 | `InsertRelations` 比较账户的指针 | `store` | 存储；组合（Task 11 起） | Task 9 的修复轮 1；组合一层 Task 11 |
| T10.23 | 跨项目的写把自己的项目放在最前 | `lock` | 单元（A.3） | Task 10，评审之前（P10.b） |
| T10.24 | 对方的动态写存储的类型 | `activity` | 单元；组合（Task 11 起） | Task 10，评审之前（P10.c） |
| T10.25、T10.26 | 锁下读、判定另一个工作项失败当作没到达；`lockAndDecide` 丢掉 `reachUnder` 的错误 | `port` | 单元（A.3） | Task 10 的修复轮 1（评审 Important，R1、R2） |
| T10.27 | 预检没到达的在锁下再判定，放过就算到达 | `reach` | 单元；组合（Task 11 起，`TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`） | Task 10 的修复轮 1（R4）；A.3 的理由被复审否定，Task 11 加组合 |
| T12.19 | 已归档的工作项的关联列表答成空的 | `archived` | 单元；组合 | Task 12，评审之前（P12.b） |
| T12.20 | `seeEach` 对被拒的项目里每个工作项再判定一次 | `guest` | 单元（A.3） | Task 12，评审之前（P12.a） |
| T13.26 | 删除关联的动态不论类型都写成 `blocked_by` | `activity` | 单元；组合（Task 14 起）；端到端（I9，Task 15 起） | Task 13，评审之前（派发带上的） |
| T13.27–T13.30 | 锁下再读两端、再读关联、第二端的判定、锁下的判定失败时答成拒绝 | `port` | 单元（A.3） | Task 13，评审之前（派发带上的） |
| T13.31 | 两端任一个的写不按 `id` 排序进闸门、取锁 | `lock` | 单元（A.3） | Task 13，评审之前（P13.f） |
| T13.32 | 删除关联不按 `id` 排序改最后修改 | `changed` | 单元（A.3） | Task 13，评审之前（P13.g） |
| T13.33 | 逐端核对归档，另一端在先 | `archived` | 单元；组合（Task 14 起） | Task 13 的修复轮 1（评审 Important） |
| T14.5 | 逐端核对归档，第一端在先 | `archived` | 单元；组合 | Task 14，评审之前（P14.a） |
| T15.1 | 只有能改存为 `related_issue_id` 的那个工作项的人才能删 | `either` | 单元；组合（矩阵）；端到端（I9） | Task 15，评审之前（P15.b） |
| T16.1 | 加子工作项把移动的工作项写在动态之后 1 毫秒 | `moment` | 单元；端到端（I8） | Task 16，评审之前（P16.a） |

**原型的行在执行中加的层**（都由运行确认）：T2.10 的组合一层在 Task 2 的树上就被发现，"（Task 3 起）"去掉（Task 2 的修复轮 1）；T6.4 加组合（`TestBodiesThatBreakTheStructureAnswer400`，Task 6，评审之前）；T7.9 加存储（Task 7，评审之前）；T11.5 加组合（同 T6.4，Task 11 的派发）；T9.8 加 I9（Task 15 起，Task 15 评审之前）；T13.1 加 I9（Task 15 起，Task 15 的修复轮 1）；T13.2、T13.7、T13.8 加 I9（Task 15 起，整分支评审之后的修复轮）；T13.12 加 `TestWhoMayNotDeleteARelation`（Task 14 起，Task 14 评审之前）。重新锚定的（缺陷不变）：T8.14（Task 8，评审之前）、T8.8、T8.9（Task 8 的修复轮 1）、T9.6（Task 9 的修复轮 1）、T4.21 的编辑（Task 12 的修复轮 1）。

**台账点名的、没有成为行的探针**（结果在各 Task 的报告、预检和评审的记录里）：P10.a（接受、不改，第 3 节）；R3（回答的 `grants` 的自己的项目：改代码之后没有可改坏的）；P13.h（删除关联的另外 8 个组合层不读 T13.26 改的那一行，它在单元和它列下的较晚的层死）；P14.b（成为 T13.12 的一层）；P15.a（成为 T9.8 的 I9 一层）；修复轮的 P-T4.17（组合一层的探针，活着：支持 A.3 的理由）和 B-T4.23、B-T2.15（修复之前的同一个编辑，活着：证明缺口）；评审者的 m-order（成为 T13.33）；预检的 44 个 `PF.*`（第 1 节）；整分支评审的探针 `TestFinalReviewProbeAHiddenParentNamedAgain`（I1）。没有等价的变异：247 个都被发现。

## 6. 移交事项

同一个里程碑之内的后续 Phase（spec 第 5 节的表和第 7 节的"留下的条目"照录，另加执行中交出的）：
- **P4**：动态的列表对按标识指名别的工作项的行（第 5、20–23 行）执行访客规则（spec 第 3 节第 22 条）；`IssueDetail` 加 `description_html` 之后 `Issue` 与它分开，HTTP 的 `issueDetail → issue` 的转换在编译时故意断开，`apitest` 核对"`IssueDetail` 是 `Issue` 加一个字段"；分开之后的回答仍经 `seen`。M2 交接第 3 节的其余写工作项的用例（P4、P7）的 `updated_at`（取得全部锁之后读时钟）。
- **P5**：`VisibleProjects` 来了之后，`listIssueRelations` 的 `seeEach` 和 `ArchivedProjects` 可以换成它（它的项目正是 `seeEach` 放行的，这个工作项自己的项目已归档时同项目的对方照样列出），`ArchivedProjects` 没有别的读者时删去；列表的行按调用者的规则数子工作项、遮住 `parent_id`，P5 的访客规则的一致性测试核对它；连接池上设 `force_custom_plan`（设计 3.16）之后，删除项目的关联一批的好处才成立（spec 第 7 节）；M3 交接第 3 节的个人主页一行。
- **P6**：加关联、删除关联、加子工作项、移动、删除子树的事件（3.19）；M2 交接第 3 节的自动归档推进固定时钟的测试；M3 交接第 4 节的 `archive_in` 的说明。
- **P7**：链接的表进删除的连带，在全局表序里关联之后、关注之前。
- **P8**：选择器、搜索排除自己和后代、已有关联的（3.18）；M2 交接第 4 节的草稿发布。
- **P9**：`ProjectGate.DeletedAt`：删除项目、工作区改为任务之后，删除工作项跳过另一端的项目已删除的关联行；`deleteIssueRelation` 的描述加上"任一端的项目已删除、它的任务还没删这一行时也是 `issue.relation_not_found`"；交错 24、25；T7.5（`LockMany` 不回答等锁期间删除的项目）、T13.4（`eitherUnder` 在判定之前核对两个项目都锁住）各加一个组合测试；通过线的量测包括关联的一步，库里同时有别的工作区的行（spec 第 7 节）；M3 交接第 2 节的物理删除账户、第 4 节的删除断言的限时轮询。
- **P14**：父子、关联的页面（I8、I9 的页面版本），选择器；页面只在用户改了父工作项时在 PATCH 里发 `parent_id`，同别的字段（回答里遮住的 `null` 发回去什么都不改，spec 第 3 节第 9 条；负责人裁定 A 之后这一行只是页面的写法，不再是安全的条件）。
- **P3 之后各后端的 Phase**（到 P9）：各自的批量写按约定五核对（M3 交接第 2 节），各自的操作进规则表、操作名、矩阵（M3 交接第 3 节），其余工作项列表的访客规则在各列表的 Phase。
- **M2 交接第 4 节的 map 型、递归的请求体 schema**：P3 没有用到（spec 第 3 节第 21 条），用到它们的 Phase 或 M4 的 review 写明。
- **收尾**：3.23 中 P3 的行由 Task 16 写好，收尾逐行核对；两份交接的 `status` 在 M4 收尾随 review 关闭；P1b spec 第 5 节"收尾"一行（:223）的手写的 `55P03` 改用 `postgres.LockNotAvailable`：P3 的 Task 8 改了 `project` 的 `demote`（经 `pgtest.RowWaits`），HEAD 上剩 13 个测试文件里的 16 处（`git grep '"55P03"'`，去掉 `tx.go` 自己、它的测试和 `gate_test.go` 造的错误）；`workspace` 的 `deactivation_test.go` 的 `waitsFor` 是 `pgtest.RowWaits` 的近似副本（300 毫秒的 `lock_timeout`，找不到行时也失败），在收尾一起合并（Task 8 的裁定）；M3 的世界手拼项目成员的请求（分诊 6），要改是收尾的测试整理；设计 13.2 指向 M5–M8 的 `handoffs/M4-closeout.md` 的 4 个链接由收尾写那些文件；空的 `m4-p1b` worktree 由负责人定（第 7 节）。

**交接的处理结果**：P3 在两份交接里写下"处理结果（M4/P3）"：[M2 收尾交接](../handoffs/M2-closeout.md)（第 3 节 P3 的写的 `updated_at`；第 4 节的 map 型和递归的结构没有用到）、[M3 收尾交接](../handoffs/M3-closeout.md)（第 2 节的加关联的另一端的预检和约定五，第 3 节的规则、矩阵、码和工作项列表的访客规则，第 4 节的表和删除、`expectIssueDeleted` 的豁免）（Task 16 `76086a89`，`ac426aeb`、`62c5d0b2`，负责人裁定 A 的一句 `dcb99363`）。两份的 `status` 都照旧是 `open`，在 M4 收尾随 review 关闭（P1a、P1b、P2 的做法）；各自仍未处理的条目写在它们的"处理结果（M4/P3）"末尾的"仍未处理"一行：M2 收尾交接 :135（第 3 节其余写工作项的用例（P4、P7）、自动归档推进固定时钟的测试（P6）；第 4 节的草稿发布（P8）、map 型和递归的结构；第 1、2、5–13 节按 M4 设计 13.1 的落点），M3 收尾交接 :159（第 2 节 P3 之后的批量写（到 P9）、物理删除账户（P9）；第 3 节 P3 之后各 Phase 的操作、个人主页一行（P5）、其余工作项列表的访客规则；第 4 节的 `archive_in`（P6）和删除断言的限时轮询（P9）；第 1、5–9 节），与 spec 第 7 节的表一致。M4 设计 13.1 落在 P3 的（M2 交接第 3 节 P3 的用例，M3 交接第 2 节中约定五的逐个核对和加关联的预检）都在其中（spec 第 7 节）。

## 7. 已知限制

- **删除子树不分批**（spec 第 6、7 节）：每张表一条语句，子树可以是整个项目；预检在目标规模上量过（请求平铺 2.58 秒、一条链 2.83 秒，删除项目 3.04、3.18 秒），远在 15 秒之内。需要时分批的写法与删除项目相同（键集、一个时刻）。
- **删除项目的关联一批在 P5 之前用通用计划**（spec 第 7 节）：`UNION ALL` 的写法定制计划快（26 → 6 ms），`plan_cache_mode = auto` 下同一个连接执行五次之后服务器换用通用计划；评审的数据上目标规模的项目这一步在 P5 之前从约 184 ms 变成约 429 ms，好处（不随别的工作区的行增长）在 P5 强制定制计划之后才成立。
- **动态的读在 P4**：P3 写的第 21、23 行按标识指名对方，P3 没有读动态的接口，所以不透露；P4 要对它们执行访客规则。
- **P3 的写不发布事件**：P6 加。
- **`DeletedAt` 在 P9**：同步的连带之下没有读者；T7.5、T13.4 在 P9 改为任务之后才分得出（第 6 节）。
- **按性质只在一层的**（spec 附录 A.3，HEAD 上与 plan 的表一致）：单元 34 个（清扫 16、执行 18）、存储 21 个（17、4）、组合 6 个、模式 4 个、守卫 1 个。其中闸门的次序（T10.4、T7.11、T10.23、T13.31）组合一层看不出：两个跨项目的写按相反的次序排进对方的队列，至多等到 `max_wait` 答 503，不会挂住；整分支评审确认 T10.23、T13.4、T12.20 活过整个 `bootstrap`，与 A.3 相符。
- **时间差**（范围复审的观察 3）：`OwnOnly` 的修改设 `parent_id` 时只在有父工作项时多一次带索引的读，所以藏起的父工作项比没有父工作项多一次读；`hideParents` 读回答已有同样的不对称；回答（码、体、头、数、次序、422 的位置）没有不同，不在 P3 的性质里，不处理。
- **两端任一个的形式的 grant**（Task 13 的关切 1）：`held.grant` 是放行的那次判定的，`held.instead.grant` 是零；没有读者。出现读者时由 `eitherUnder` 放各端的 grant。
- **接受、不改的**：`ArchivedIssueProjects` 在没有别的项目时也查一次空的（Task 12 的 info 6）；共用的 `lost` 丢掉活动指名的每个工作项（P10.a）；T10.27 的交错翻转 `guest_view_all_features`、不是提升成员（Task 11 的关切 1）；23 个单个 id 的数组内联（Task 11 的 R2）；M3 的世界手拼成员的请求（分诊 6）；`ChangedRows` 的"adds or removes"、`storedRelations` 与 `linkTable`、`TestInsertRelations` 的"in their order"（Task 9 的裁定）；一个三行的 422 的核对出现两次（Task 6 的复审，没有裁定）；`update_issue.go:60` 的说明换行不齐（范围复审的观察 4）；Task 3、4、8、11、12 长过约 1,500 行（第 5 节）。
- **T3.13 的编辑在 Task 4 的树上不能应用**（修复轮的关切）：那里原来的父工作项的读在 Task 4 的 `parentBefore` 里；没有哪一遍在那棵树上跑它，它在 Task 3 的树和最终的树上都能应用。
- **持续集成没有跑过**：原型、复现和执行都在 macOS 上（spec 第 6 节最后一行）；ubuntu 上的结果在合并时看。
- **环境**：机器上没有 Firefox，A4 的两个 firefox 测试在每次 `make e2e` 都失败（装它是全局的安装，规则不许）。第一次运行在别的会话的负载下超时、Docker 空闲时重跑通过：修复轮的第一次 `make test`（四个 M3 成员关系的矩阵格子 32 秒超时，范围复审在空闲时重跑 69 格通过，那四格各 0.14–0.21 秒）和第一次 `make e2e`（A9、P3、S8 三个页面故事在负载 14 时 5 秒的期望超时，重跑 10 个通过）；Task 4 的修复轮 1 前两次 `make test` 起不来容器（Ryuk 不见了、套接字超时），第三次通过；Task 6 的评审之前的修改一次 `platform/jobs` 的容器起不来。本文的作者的运行时，别的会话的 12 个 `postgres:18.6` 容器开着、CPU 0%，七个门禁第一次就通过（`make e2e` 只有 A4 的两个 firefox 测试失败），没有要重跑的超时。留下的 `Created` 的 testcontainers（台账记到 152 个）照规则不动。
- **空的 `m4-p1b` worktree**：客户端的重启重新建过 `.claude/worktrees/m4-p1b`（P2 的评审记录第 7 节），HEAD 上它仍在 `7db78788`、分支 `worktree-m4-p1b`，没有自己的提交（`git log 7db78788..worktree-m4-p1b` 没有输出）；不是本 Phase 的，删不删由负责人定（控制者问过，还没有答复）。
- **注明日期的记录里过时的数**：spec 附录 A.1、A.4 的原型的数、A.5、A.6 的数（第 5 节逐个数出）照裁定不改；A.3 和 Global Constraints 的大小在执行中保持为真。
