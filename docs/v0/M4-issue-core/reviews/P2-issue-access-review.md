# M4/P2 访客规则、创建者、删除、按编号取、归档、起止日期：评审记录

| 项 | 内容 |
|---|---|
| Phase | M4/P2 `issue-access` |
| 日期 | 2026-10-21 |
| 结论 | **通过**：14 个 Task 逐个实现、逐个评审，都通过（T1、T2、T7、T8、T9 的评审各有一条 Important，T14 的评审有两条，都出在 plan 写定的代码、测试或文档里，都在当轮改掉）；整分支评审"修复轮之后可以合并"，Critical 0、Important 0、Minor 2，另有分诊的六项；修复轮一轮（两个提交）落实 M1、M2 和分诊的六项；范围复审：七条裁定都已落实，新的 Minor 1 条（N1）、Nit 1 条（N2，不改），"N1 的一句加上之后可以合并"；N1 在本提交改正（第 4 节） |
| spec / plan | [spec](../specs/P2-issue-access.md) / [plan](../plans/P2-issue-access.md) |
| 设计 | [M4 设计](../M4-design.md) 第 12 节 P2；本 Phase 改过的设计各节见 spec 2.1 |
| 分支 | `worktree-m4-p2`，从 `main` 的 `7db78788`（P1b 的合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`64c2acc9`，2026-10-19，`$M4TMP/p2-architect-report.md`）：在 `$M4TMP/p2proto` 上逐 Task 做出原型、存快照（`p2snap/T0`…`T14`），plan 的块由相邻两份快照的差异生成。设计 12 节 P2 的十四个草稿任务切成 14 个 Task：按编号取拆成 Task 5、6，起止日期拆成 Task 10、11，矩阵的列提前成 Task 3，规则的行分在 Task 1、2；最大的 1,454 行，没有停下。plan 11,744 行，spec 434 行。从基点逐 Task 重放 538 个块，照原样跑 103 个 `Run:`（1,796 秒，没有重跑），`treediff` 与原型差的只有提交的三份文档（3,410 对 3,408 个文件）；114 个变异都被发现，没有等价的；`bootstrap` 的 63 个交错、竞争、取锁形状和连接池的测试 `-race -count=5` 315 次顶层通过；矩阵 1,259 个叶子。控制者的裁定（`$M4TMP/p2-architect-rulings.md`）：spec 第 3 节 1–19 条照建议，第 3、8、10、14 条带条件（第 3 节）。
- **执行前的预检**（只读，`$M4TMP/p2-preflight/preflight.md`）：High 0、Medium 1、Low 12。访客规则在六个操作上的证明对；把闸门的位置算作资源也没有等待环；约定五对每个写者成立。独立重放 538 个块、103 个 `Run:`（1,875 秒），每个 Task 的树与快照相同；63 个交错和竞争的测试 315 次顶层通过；I1、I2、I15 各重复 4 次，12 次通过；核对了变异表的 114 行。它自己的 31 个变异：26 个在所说或所料的层被发现，5 个显示了缺口：PF.D5（删除跳过闸门，每一层都活着）成为 M1，PF.G4b（创建者不明的工作项在 `AllowCreator` 上算调用者的，每一层都活着）成为 L1，PF.A3（恢复看状态组，只在单元一层）成为 L2，PF.I1、PF.I2（已删除的项目的标识、不折叠，只在存储、领域一层）成为 L3。
- **修订**（`dfe4c257`，2026-10-20）：预检的 M1、L1–L12 照控制者的 15 条裁定（`$M4TMP/p2-preflight-rulings.md`）改进原型、快照、plan、spec 和设计 3.3、3.4、6.5；spec 的"待裁定"标记都换成裁定（449 行），plan 11,973 行、547 个块。变异 119 个（新的 T2.9、T4.21、T5.16、T7.19、T7.20），改过的层和新的行在各自的树上重跑 101 次、116 层，全部被发现。之后再完整重放一次：547 个块、103 个 `Run:`（1,839 秒，没有重跑）；交错和竞争 63 个测试 315 次顶层通过（459 秒）；矩阵 1,259 个叶子；端到端 120/123（失败的三个与 P2 无关：副本不是 git 检出的 S3，没有 Firefox 的 A4 两个）。
- **执行**（2026-10-20）：
  - 14 个 Task 各由一个实现子任务完成（不指定模型），用 `planapply.mjs` 应用 plan 的块；各由一个评审子任务（opus）检查是否符合 spec 和质量。11 个 Task 在评审之前由控制者对实现者的报告裁定、当轮改（"评审之前的修改"：Task 1、2、5、6、7、8、9、11、12、13、14）；评审要改的在修复轮改，修复轮之后有范围复审：sonnet 11 次（Task 1–8、11、13、14）；只改文档的三个修复轮（Task 5 的第 4 轮、Task 9 的第 2 轮、Task 12 的第 2 轮）由控制者自己核对差异，不再复审；控制者另改了 spec 的一行（`a94a5629`，Task 6 的复审的 Nit）。同一时刻只有一个写入者。执行 46 个提交。
  - 实现者在每个 Task 里按 plan 的变异表再跑变异，在 `$M4TMP/p2sdd/` 里没有 git 的副本上跑（`mut.py`：单元、存储、组合、端到端即 I1、I2、I15、守卫各层）。标"（Task n 起）"的行由 Task n 的实现者在自己的树上重跑（`MUT_LATER=n`，第 3 节）：Task 2 3 行、Task 3 6 行（它的修复轮又给 T2.11 加了矩阵两层，HEAD 上 7 行）、Task 4 7 行、Task 5 1 行（修复轮 3 加的 T2.12）、Task 6 15 行、Task 7、8 各 1 行、Task 11 18 行、Task 12 6 行、Task 13 33 行（含 T8.8 的 I15 一层）。端到端的发现由 `e2ekills.py` 核对是断言失败（Task 13：34 个、0 个可疑；评审之前的修改之后 29/0，修复轮之后 17/0）。容器起不来不算发现：Task 4 的实现者自查（`96994b5f`）之后的 `make test` 两次起不来容器（44、45 个 `ok`），第三次 47 个 `ok`；Task 11 的修复轮 2 的第一次 `make test` 一个包起不来容器，Docker 空闲时重跑 47 个 `ok`；T8.9 第一次的 Pass C 起不来容器（重跑是真的发现）。
  - **`mut.py` 的 INFRA**（Task 8 之后，第 3 节）：`mut.py` 原来把起不来的容器算作发现（pgtest 的 `t.Fatal` 打出 `--- FAIL`）。加了 INFRA 的结论之后，扫出的只有一份过时的架构的日志：`p2tools/mut-logs-c/T5.1-composition-1.log`（T5.1 的组合一层，容器没起来）；T5.1 在最终的树上重跑，在 `issue_identifier_test.go:85` 的断言上被发现。那份日志原样留着。
  - **plan 是"执行时的 plan"**（从 P1b 带来的裁定，从 Task 1 起）：每个修复轮在同一个提交里重新生成它的 Task 的块（`taskblocks.py`、`compose_plan.py`），执行中加的变异行经 `mutants_p2*.py` 和结果的覆盖文件（`mut-results-{a,b,c}-amend.json`）进 plan 的表；块数从 547 到 592（第 5 节）。spec 附录 A 是注明日期的原型记录，不改（第 5 节数它）。
  - Task 自己的测试缺口和说错的注释，由实现者在提交之后提出，或由 Task 的评审发现，控制者逐项裁定（第 3 节）。补上缺口的都先由变异证明缺口存在，补上之后再证明测试咬得住；执行中加的变异是 plan 的变异表的行（5.1）。
  - **执行中没有改生产代码的行为**：三处改了代码、行为不变，契约的两段描述改了文字，几处只改注释（第 5 节）。其余都是测试、测试的数据和文档。
  - **规则的失误**（违反书面规则的调用）：执行中 31 次，都没有改动 worktree 或任何提交：git 命令都是只读的，被拦下的没有运行，写了文件的只写 scratch。
    - 实现者 16 次：Task 5 的修复轮 3（`faacec8b` 那一轮）两次只读的 `git -C`；Task 6 评审之前的修改一次用 `;` 把只读的 `git diff --stat` 接在 `grep` 之后；Task 7 评审之前的修改一次用 `&&` 把只读的 `git diff --stat` 接在 `cp` 之后；Task 8 的修复轮 2 一次 `git -C … check-ignore`、一次把 `ls … \| wc -l` 和 `git show --stat …` 连在一个调用里；Task 13 一次 `cd`、两次用 shell 变量（`D` 一次，被拦下；路径里的 `L` 一次）、五个行内的循环（两个 `for`，三个等它自己的日志的 `until`）；Task 14 一个行内的 `for`；整分支评审之后的修复轮一次空的行内 Python heredoc（没有运行任何东西）。
    - 评审者 14 次：Task 6 的复审一次 `cd`；Task 8 的评审一次失败的 `cd /dev/null`、一个带 `$f` 的行内循环、一次带 shell 变量被拦下的命令；Task 9 的评审三次路径里带 shell 变量（其中两次是行内的循环）；Task 11 的复审两个行内的 `for`；Task 14 的复审一次 `cd`；整分支评审一个调用里的 `cd`、行内循环和 `git -C`，一次 `cd … && python3 -c`，一次用 `;` 连起来的两个只读的 git 命令；范围复审一个 `python3 -I -c` 里的行内循环。
    - 控制者 1 次（2026-10-20）：一次 `git -C <主检出> worktree list`，被拦下，没有运行（台账）。另外，控制者给 Task 3 的派发说明写错了跑法（第 3 节的 `MUT_LATER`）。
    - 不算在内的：评审者另记下的几次连起来的只读的非 git 命令（Task 8 的复审三次，Task 11 的复审和范围复审各几次；书面的规则只禁连起来的 git 命令）；Task 14 的实现者一次 `cp` 盖掉 scratch 里的一份备份（重做，没有丢东西）；Task 14 的复审一次写错路径的 `python3`（没有作用）。
    - 执行之前：架构子任务的报告记下四次（两次 `cd`，一次失败；两次行内的 Python heredoc，写的是它自己的 scratch 文件），预检记下一次（`shasum` 与只读的 `git status` 连在一个调用里）。
    - 本文的作者 0 次；一次只读的 `awk` 调用被拦下，没有运行。
  - **周的限额**：Task 13 的实现者在块已应用（五个端到端的文件没有提交）、`make e2e` 已跑完（121 个通过，失败的两个是 A4 的 firefox 测试）之后被周的限额停下。会话恢复之后控制者回到 m4-p2 的 worktree，恢复同一个实现者从那里接着做（提交、T8.8 的 I15 一层、`MUT_LATER=13`、重新组装 plan、重放、报告）。客户端的重启重新建了一个空的 m4-p1b 的 worktree（第 7 节）。
  - **负责人的暂停**：Task 13 之后负责人说"当前后台任务完结之后暂停一下，我重启一下客户端"（2026-10-20），之后"重启好了,继续"。重启又从 m4-p1b 开始，控制者回到 m4-p2：HEAD `5874a47f`，树干净；Task 14 从那里开始，带上暂停时记下的两条提示（第 3 节）。
- **整分支评审**（opus，`7db78788..fd546f35`，48 个提交，`$M4TMP/p2final/review.md`）：结论"修复轮之后可以合并"，Critical 0、Important 0、Minor 2，另有分诊的六项（都不改行为）。它在 `git archive` 的副本上重跑全部门禁（gen-same 0 个差异，`make test` 47 个 `ok`，`make e2e` 120/123：副本不是 git 检出的 S3 和 A4 的两个）；`bootstrap` 的 69 个测试 `-race -count=5`；plan 的重放（588 个块，`make gen` 之后 3,410 对 3,410 个文件、0 个差异）；矩阵 1,259 个叶子；27 个变异的 57 层和两个端到端的层，都是断言失败。它在 HEAD 上重推了取锁的路径、逐个操作的访客规则、把闸门的位置当作资源的死锁和约定五（第 4 节）。
- **修复**：一轮，两个提交（`b57ae344`：代码、测试和重新组装的 plan；`8c42351b`：设计和 spec）。范围复审一次（opus，`$M4TMP/p2final/rereview.md`）：七条裁定都已落实；新的 Minor N1、Nit N2；本提交改 N1（第 4 节）。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | I15 的接口版本（归档、恢复、两条动态、`issue.not_archivable`、已归档时改字段 409、访客 403、删除已归档的）、I1 和 I2 的起止日期（含合并之后的日期 422 的两个方向）、I2 的已归档 409 通过；`make e2e` 123 个：此前的 122 个，加 I15 | **通过。** T13 `e17b20b1`（`b402016d`、`0db90e1a`）：`make e2e` 两次各 123 个中 121 个通过，失败的两个是 A4 的 firefox 测试；`MUT_LATER=13` 的 33 行、34 层（含 T8.8 的 I15 一层）都被发现，`e2ekills.py` 0 个可疑。`5b38471f`、`5874a47f` 之后同样 121/123，重跑的 29 层、17 层都被发现。整分支评审在 `fd546f35` 的副本上 120/123（多出的失败是 S3：副本不是 git 检出），T8.8 在 I15（`i15-archive-issue.spec.ts:110`，答 409，期望 204）、T10.3 在 I2（`i2-update-issue.spec.ts:248`）被发现。本文的作者在 HEAD `8c42351b`（本提交之前，树干净）上：`make e2e` "Running 123 tests"，121 个通过，I1（`i1-create-issue.spec.ts:23`）、I2（`i2-update-issue.spec.ts:30`）、I15（`i15-archive-issue.spec.ts:36`）、S3 在其中；失败的两个是 A4 的 firefox 测试（`stories/identity/a4-refresh.spec.ts:85:3`），错误是 `Error: browserType.launch: Executable doesn't exist at /Users/xiaoruan/Library/Caches/ms-playwright/firefox-1543/firefox/Nightly.app/Contents/MacOS/firefox`（机器上没有 Firefox；`$M4TMP/p2sdd/rd/gate-e2e.log`） |
| 2 | 判定表、矩阵（P2 的格子）、按目录的删除测试、`bodyshape` 和生成器的测试通过 | **通过。** 本文的作者的 `make test`（第 5 行）：`access/domain` `ok`（`TestEveryRuleDecidesItsCells`、`TestDecideAtTheProjectLevels`、`TestAllowCreatorAllowsAMemberWhoCreatedIt`、`TestARefusalCarriesTheGrant`、`TestTheGrantCarriesTheRoles`），`bootstrap` `ok`（83.8 秒；`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、`TestThePermissionMatrixCoversEveryOperation`、`TestMatrixViolationsCatchesEachGap` 在其中），`platform/httpserver/bodyshape` `ok`，`server/tools/bodyshapegen` `ok`。矩阵在 HEAD 上另跑一次（`go -C server test -count=1 -v -run '^TestPermissionMatrix$' ./internal/bootstrap/`，`$M4TMP/p2sdd/rd/matrix.log`，`matrixcount.py`）：1,259 个叶子全部通过、0 个失败；工作项的操作 393 格（`getIssue` 20、`getIssueByIdentifier` 20、`createIssue` 80、`updateIssue` 136、`deleteIssue` 40、`archiveIssue` 57、`unarchiveIssue` 40），其余 866 个是 M3 的 865 格和模板的 `prepare`；顶层 1.5 秒。与 spec A.4 记的、整分支评审在 `fd546f35` 上数的相同 |
| 3 | 一个看不到工作项的访客对它的每个读和写得到 404、不是 403：矩阵的"访客·别人""管理员·访客"两列（六个操作），交错 5 的访客一支 | **通过。** 同一次矩阵的运行（`guestcols.py`）："访客·别人"（`project_guest,_another's`）和"管理员·访客"（`workspace_admin_and_project_guest`）两列在六个操作的每一行都通过，0 个失败：`getIssue`、`getIssueByIdentifier` 各一行，`updateIssue`、`deleteIssue`、`unarchiveIssue` 各两行，`archiveIssue` 三行。交错 5 的访客一支在 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（bob 改 alice 的 Fix 或 walt 的工作项时被降为只看自己的项目访客，锁下 404；创建、改和删他自己的 403），在 `make test` 和整分支评审的 `-race -count=5` 里通过（第 4 行）。整分支评审的抽样：T1.7（`OwnOnly` 按工作区的角色）在 `decide_test.go:313` 和矩阵的 `getIssue`、`getIssueByIdentifier` 格子（答 200，期望 404）被发现，T2.1、T2.2、T2.5 在矩阵或交错 5（`issue_interleaving_test.go:388`）被发现；执行中加的 T2.10、T2.11（按 `id` 的次序认他自己的工作项）在矩阵和交错 5 被发现（5.1） |
| 4 | 交错 12、删除与修改的两种顺序、交错 4、5 的删除和访客一支 `-count=5 -race` 通过，没有 40P01；P1b、P1a、M3 的交错照旧通过 | **通过。** T12 `3132203b`：plan 的 `-race -count=5`（94 秒），另把 63 个交错、竞争和取锁形状的测试各跑 5 次（449 秒），都没有 40P01、死锁、DATA RACE；`eff79cad` 之后改过的两个测试再跑 5 次，Task 12 的评审对三个测试再跑 5 次（33 秒），都通过。整分支评审在 `fd546f35` 的副本上：`bootstrap` 的 69 个测试（`racenames.py` 的 63 个，即 M3、P1a、P1b、P2 的交错、竞争、取锁形状和连接池的测试；加 `gate_test.go` 的和事务的连接的测试；`raceall.sh` 的 45 个都在其中）各 `-race -count=5`，490 秒，345 次顶层全部通过，0 行 40P01、死锁、DATA RACE。之后的 Go 改动只有修复轮的 `b57ae344`：单元一层的代码和测试（构造、`issueState` 的移动）和裁剪过的 `TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`；它和 `TestDeletingAWorkItemAndChangingItSerialize` 由修复轮和范围复审各 `-race -count=5` 再跑，10 次顶层、10 个子测试，没有 40P01、死锁、DATA RACE，所以那次运行对 HEAD 仍成立 |
| 5 | `make gen-check`、`lint-go`、`test`、`lint-web`、`knip`、`test-web`、`e2e` 通过 | **通过。** 本文的作者在 HEAD `8c42351b`（本提交之前，树干净）上逐个运行（`$M4TMP/p2sdd/rd/gates.sh`，日志 `gate-*.log`，`gates.summary`）：`make gen-check` exit 0；`make lint-go` 两段 `0 issues.`；`make test` exit 0，47 个 `ok`、0 个 `FAIL`，`go test -count=1`（没有缓存），`bootstrap` 83.8 秒；`make lint-web` exit 0（"keywords: 72 rules, 23 exceptions, no hits."；i18n-keys 1,299 个键；turbo 54/54）；`make knip` exit 0；`make test-web` 17/17；`make e2e` 见第 1 行。跑完之后树仍干净。`server/go.mod`、`go.sum`、`server/tools/go.mod`、`go.sum`、`pnpm-lock.yaml` 自 `7db78788` 没有变化（`git diff --stat` 没有输出）。本提交的文档改完之后（提交之前，`$M4TMP/p2sdd/rd/post.sh`，日志 `post*-*.log`）：`make lint-web` exit 0（关键词守卫 72 条规则、23 个例外、没有命中；它读 git 列出的文件，未跟踪而不被忽略的也读，本文在其中）；`make gen-check` exit 0；`doclinks.py` 在本文、设计、spec 和 plan 上：本文 6 个链接、spec 9 个、plan 0 个，都不断；设计 19 个，断的只有 13.2 指向 M5–M8 的 `handoffs/M4-closeout.md` 的已知 4 个（M4 收尾写那些文件） |

持续集成：分支在执行中没有推送（`git branch -r` 只有 `origin/HEAD`、`origin/main`）。推送之后和合并之后的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，写明理由和裁定错了的代价。完整的记录在执行台账（`.superpowers/sdd/P2-issue-access/progress.md`）中。

**执行前**
- **负责人的裁定**（设计第 10 节，2026-10-17，P2 照用，不重开；理由和以后改的代价写在设计各条里）：
  - 决策点 1 的 B：工作项的写只锁项目行（N），不锁工作区行；取锁之前一个不加锁的预检答出一切会被拒绝的写，取连接之前进项目的写闸门（11.11）。P2 的删除、归档、恢复都走 P1b 的同一条路（`Locks.run`），访客规则在 `Locks.decide` 里，预检和锁下各执行一次；spec 第 3 节第 7 条把闸门的位置算作资源重推死锁。错了的代价（设计）：B → A 只改取锁的一步和测试，但要重新面对饥饿。
  - 决策点 2 的 A：一个 M4；Phase 超过约 16 个任务、每个约 1,500 行时，控制者可以直接拆，不问负责人。P2 没有拆：架构子任务切成 14 个 Task，最大的 1,454 行；修订之后 Task 7 1,519 行，跨 Task 的核对接受、不重新切；执行的修复轮让它长到 1,630 行（第 5 节）；不重新切的代价是一个 Task 长过约 1,500 行。错了的代价（设计）：选 A 之后若中途要拆，在某个 Phase 的边界上收尾即可，成本是一次额外的收尾。
  - 决策点 7 的 A：删除项目、工作区时工作项一侧的行由 P9 的后台任务删除，P1–P8 在请求的事务里同步（过渡）。P2 的约定五的论证靠它：P9 的任务只写已删除的项目的行，那时已没有写到得了它们（spec 第 3 节第 7 条）；删除一个工作项本身在请求的事务里同步（3.6）。错了的代价（设计）：A → B 删掉订阅者和任务，把 `delete_rows.go` 的语句放回请求的事务。
- **架构的报告**（`$M4TMP/p2-architect-rulings.md`，2026-10-19）：spec 第 3 节 1–19 条照建议，理由在各条里；没有一条重开负责人的决定，都在设计、它的交接（P1a 第 5 节的"由 P2 定"、P1b 第 5 节）或控制者自己的裁定之内。带条件的四条：第 3 条（`ProjectGate.Lock` 不读 `guest_view_all_features`：修订设计 3.3、6.5 的 `ProjectGate`）、第 8 条（归档、恢复的动态的 `comment`：差异清单写上"动态的 `comment` 不同"）、第 10 条（`checkDate` 与 `ParseDate` 各带一份两行的规则：两处的说明互相指名）、第 14 条（两处 `parent_id` 的豁免写进第 5 节 P3 一行，作为有约束力的交接）。第 8、9 条是产品的文字（两行动态的 `comment`；`issue.not_archivable`、`issue.archived`、`project.state_in_use` 的文案和补救）：本地的决定；错了的代价是两句文字、它们的单元测试和端到端的两行，和三个码的几处文案。预检要特别核对第 2、3、5、7、9、11 条。
- **预检**（`$M4TMP/p2-preflight-rulings.md`，2026-10-20）：15 条裁定全部照建议，一个修订提交改入（`dfe4c257`）：
  - M1（删除、归档、恢复进写闸门没有钉住）：照预检加单元的 `TestEachWriteOnAWorkItemEntersItsProjectsGate`；另加组合一层（裁定加的）：只在单元一层看得到的闸门的性质是缺口，除非 A.3 说明它按性质只在单元一层，而这个不是；`gate_test.go` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`，闸门满时三个写各立刻答 503 `server_busy`、`Retry-After: 1`；新行 T4.21、T7.19 各在单元和组合两层。
  - L1：`creatorOf`，T2.9（单元，Task 4 起），T2.3 加 `issue/app` 一层；两个按性质只在单元一层，到 P9（spec 第 5 节 P9 一行）。L2：恢复不看状态组的组合一层，T7.20。L3：按编号取的数据（ŞI、已删除的 Şİ、beta 的 İŞ），T5.9 离开 A.3 的只在存储一层的清单，新行 T5.16。L4–L8、L12 照预检的措辞；L9 也修订设计 3.3；L10 第 7 条的 `users` 一句；L11 第 6 节的时间差一句；裁定 2 的可选措辞（设计 3.4）、裁定 5 的矩阵的行用它们自己的名字。
  - 错了的代价：M1 是两个测试和两行变异；L1–L3 是测试的数据和三行变异；其余是文字。

**过程的裁定**
- **plan 是"执行时的 plan"**（P1b 的裁定，从 Task 1 起）：每个修复轮在同一个提交里改它的 Task 的块，执行中加的变异行经 `mutants_p2*.py` 和结果的覆盖文件进表，plan 从基点复现到 HEAD。理由：plan 是能复现的记录。代价：每个修复轮多几分钟。
- **spec 附录 A 是注明日期的原型记录**（2026-10-19，修订 2026-10-20）：修复轮只改 spec 正文的事实；执行中加的变异行进 plan 的表，评审记录数它们（第 5 节）。理由：附录 A 记的是原型证明了什么，改写它会让记录与那时的运行对不上。代价：附录里有过时的数，由评审记录逐个数出。
- 模型：实现者、文档的作者不指定模型；Task 的评审 opus；小的范围复审 sonnet，共用、并发的部分 opus。只改文档的修复轮由控制者自己核对差异（Task 5 的修复轮 4 起）。
- **`MUT_LATER`**（Task 3）：较早的 Task 的"（Task n 起）"的行照 plan 用 `MUT_LATER=n` 跑（`mutants_p2.py` 读它，只留那些变异和它们在 Task n 的层），常设的 brief 随之改正。控制者给 Task 3 的派发说明写的"`MUT_ONLY`，不是 `MUT_LATER`"是错的；实现者照 plan 用了 `MUT_LATER`。代价：没有，结果相同。
- **INFRA**（Task 8 之后，工具的修正）：Task 8 的实现者指出 `mut.py` 把起不来的容器算作发现：它碰上 T8.9 第一次的 Pass C（重跑是真的发现）和架构子任务的 T5.1 组合一层的记录（`p2tools/mut-logs-c/T5.1-composition-1.log`；在最终的树上的探针在断言上发现 T5.1：说法成立，证据不对）。控制者扫了一遍：容器起不来的签名（"pgtest: start "）只在那份架构的日志和改了名的 T8.9 的日志里。裁定（从根上改，由同一个实现者）：`mut.py` 的结论加 INFRA：Go 一层的日志带它就不算发现、也不算留下，重跑（保守：带它的运行即使另有真的失败也不算）；五份日志上核对，0 个错；T5.1 的组合一层在最终的树上重跑（`MUT_FINAL=1`）进 `mut-results-c-amend.json`，容器起来，在 `issue_identifier_test.go:85` 被发现；plan 逐字节不变，没有提交。架构的那份日志原样留着（合并的结果不再依赖它）。代价：每次容器起不来多一次重跑。

**执行中**（补上缺口的，都先由变异证明缺口存在，补上之后再证明测试咬得住）
- **Task 1（评审 Important 1 条）**：
  - 评审之前（`71f42734`）：T1.12（角色的拒绝丢掉判定读到的 `Grant`：访客规则 404 先于 403 的 `access` 一半值得自己一行；单元，组合 Task 2 起）；`TestARefusalCarriesTheGrant` 加三个之外的角色一行，T1.13（按性质只在单元一层：表的 `role` 有 `CHECK`）；`Grant.OwnOnly`、`Decide` 的两句说明写的是 Task 2 的使用者，改成不随时间变的（字段的契约，"让用例能先答"）。代价：两行变异和两句说明。
  - 评审（spec 一处缺）：`creatorRoles` 的管理员一半没有测试钉住（`creatorRoles = [member]` 通过 `access`、`shared` 的全部测试）。裁定（`8603355b`）：照评审在 `TestAllowCreatorAllowsAMemberWhoCreatedIt` 加管理员的创建者一行，T1.14（按性质只在单元一层，同 T1.2）；Minor 一并改：`TestAuthorizeKnowsTheCreator` 在三行上比较 `Grant`，T1.15（单元，组合 Task 2 起）。`CreatorID` 的说明与 Task 2 之前 `lock.go` 的零值：不改（字段的契约，Task 2 的 `Locks.decide` 填它；一个 Task 的过渡，没有读者）。代价：一行测试数据和一个断言。复审（sonnet）：已落实。
- **Task 2（评审 Important 1 条）**：
  - 评审之前（`e51f609b`）：领域和 `issue/app` 的单元数据里调用者的 id 总比别的创建者的小，按次序比较（`Compare(actor) > 0`）的变异在两层单元都通过（矩阵和交错 5 发现它）：改一行数据，每层单元都有在调用者两边的创建者，T2.10。教训的一类"调用者总是最小"；代价一行数据。A4 的 Firefox：环境，接受。
  - 评审：T2.10 的镜像（`Compare(actor) < 0`）在每个组合测试都通过（组合里被拒绝的工作项的创建者都比访客先注册）；Minor：`lock.go:174` 的 404 引设计 8.2，应为 8.4；`guest_rule_test.go` 的说明说"角色被拒绝与否"，而预检的行角色都被拒绝。裁定都接受（`2622d189`）：交错 5 加 bob（已降级）改 walt 建的工作项 → 404，T2.11（单元两层、交错 5；Task 3 起也在矩阵）；8.4 改在 `lock.go`、spec 第 3 节第 2 条和设计 6.7；加"别人的、在预检、角色允许"一行；`bobsWorkItem`、`waltsWorkItem` 移到 `issue_world_test.go`。复审：已落实；Nit：Global Constraints 写的 `issue_interleaving_test.go` 394 行是原型的量 → 不改，评审记录写执行的数（第 5 节）；成员关系的 POST 写了三次 → 不改：每次是共用的 `send` 的一次调用，不是近似的副本。
- **Task 3**：
  - 评审（Minor，plan 写定的）：(1) 种子的事实核对列的是字面的对，不是每列各自的情形（`accountOf(callerCreator)=PA` 时矩阵通过、T1.10 留下）；(2) 三处说明不对（`columns_test.go:85-86`、`issues_test.go:16-18`、`issue_columns_test.go:68-69`）；(3) `guest_view_all` 的标志由 acme 的管理员设，规则会拒绝他；(4) 工作项的列重写了 9 个共用的列。另提议：`matrixAccounts` 里让管理员·访客先注册，矩阵就能发现 T2.11。裁定全部接受，一个修复轮（`dcd30b8d`）：逐列的事实核对，T3.2（矩阵的自检）；三处说明；由项目的管理员 PA 设标志（spec 2.5 的事实修正）；工作项的列建在 `issueRows.of`、`ofProject` 上；`matrixAccounts` 改次序，T2.11 加矩阵的两层（Task 3 起）。代价：测试的几十行。复审：已落实；提示：清扫 35 仍写"acme 的管理员"（注明日期的记录，第 5 节）。
- **Task 4**：
  - 实现者的自查（`96994b5f`）：新的删除测试让 `workspace_deletion_test.go`、`workspace_deletion_catalog_test.go` 的四句说明不对：批准（事实的修正，只改注释）。容器起不来（第 1 节）：环境，记下。
  - 评审（Minor ×3，不改生产代码）：(1) T4.8 不是只在单元一层：取锁形状的 `deleteIssue` 一行发现它（`issue_write_locks_test.go:176`）；(2) 存储和目录的测试里被删的工作项的 id 总是最小的，`issue_id <= $3` 通过专门的测试；(3) `waits` 的定义"锁下第一个预检没读的表"对删除（和恢复）不对：它们的预检读 `issues`，所以模式是 `SHARE`。裁定全部接受（`e0a52e1b`）：T4.8 的行加取锁形状一层，spec 第 3 节第 6 条的事实修正（四张子表之间的次序仍只在单元一层；A.3、A.6 是注明日期的记录）；存储测试按 gone、other、item 的次序建，T4.22（存储）；`waits` 照评审改写（测试和 spec 第 3 节第 13 条）。复审：已落实；目录的测试仍放过 `<=`（谓词在存储一层，照裁定）。
- **Task 5**：
  - 评审之前（`b02c7d5f`）：`get_issue_by_identifier.go` 的类型说明指名 Task 6 才有的路由 → 去掉（Task 5 结束时为假）；`handler.go` 的"UseCases are the use cases behind the module's operations" → 不改（结构体的契约，同 `CreatorID`）；实现者的 `ORDER BY id` 的探针成为 T5.17（存储）。
  - 评审（Minor ×4，不改行为）：(1) `identifier.go` 的 `cutLast` 抄了 go1.27.1 的 `strings.CutLast`；(2) T5.17 的存储数据只凭规划器的选择分得出（`ANALYZE` 之后顺序扫描按 id 的次序，留下）；(3) 单元的拒绝表只比较 Kind 和 Code，"一个值"只由 Task 6 的组合测试钉住，没有行；(4) `seeWorkspace` 重写了 `decide` 的看不到的映射。裁定全部接受（`2a51963f`）：用 `strings.CutLast`；评审的数据，T5.17 在 `ANALYZE` 前后都被发现；单元表逐值比较（`reflect.DeepEqual`），细节的变异成为 T5.18（单元，组合 Task 6 起）；`seeWorkspace` 交给 `decide`（零的项目 id 在工作区判定）。实现者另指出 Task 2 的 `guest_rule_test.go`（和 P1a 的 `create_issue_test.go`）仍只比较码。
  - 裁定（修复轮 3，`faacec8b`）：访客规则隐藏的工作项的 404 与不存在的是同一个值，是隐私的性质（设计 8.3、8.4），那时哪一层都没有钉住（单元比较码，矩阵比较状态和码）：`guest_rule_test.go` 逐值比较（Task 5 的块），T2.12（单元 Task 5 起；组合 Task 6 起：`TestAWorkItemIsFoundByItsIdentifier` 逐字节比较 walt 对 alice 的 WEB-1 的回答）；P1a 的 `create_issue_test.go` 没有隐藏与不存在之分 → 不改。代价：测试的比较方式和一行变异。
  - 复审（sonnet）：已落实；Nit：`issue_ports_test.go:92-96` 和 plan 的一句说 id 的次序不是"按工作区的索引的"，对普通的 `projects_workspace_id_idx` 不对 → 修复轮 4（`5af85d54`，只改说明，控制者核对）。
- **Task 6**：
  - 评审之前（`3d9a9aa5`）：契约写"两个项目的标识对得上时……两个都不是时没有"，比代码窄（II、İI、Iİ、İİ 能同时存在），而代理读这段文字 → 改成"不止一个项目的标识对得上时，取逐字母等于它的大写的那个……没有就没有"，设计 3.7 写"两个或更多"；打包结果和 `schema.gen.ts` 各一行，Task 6、7、8、11 的钉值重算。照原样留下：`MatchIdentifier` 的说明、`identifier-fold.json`、spec 和 plan 的"两个以上"（对）；设计 9.1（:1679）、9.3（:1744）说的是数据恰好两个的测试 → 不改。
  - 评审（Minor ×2，Nit ×2）：(1) T6.1（`x-problem-codes` 缺一个码）不是只在单元一层：每个组合的请求经 `call`，它的 `CheckResponse` 跑 `checkProblemCode`；A.3 的理由"组合一层不读契约的声明"是错的，T4.10、T7.14、T8.6 也靠它；(2) HTTP 的测试处处用 slug `acme`，把常量交给用例的处理函数也通过。Nit：三处"不在任何项目表的列"的循环；描述没说编号怎样写（`WEB-01` → 404）。裁定全部接受（`eb69fc36`）：T6.1 加组合两层（按编号取的组合测试、矩阵的 X 格子），T4.10 加矩阵的已归档的 `deleteIssue` 格子（A.3 照旧，第 5 节数）；Task 7、8 的实现者核对 T7.14、T8.6 的组合一层；HTTP 加 beta 的一例，T6.3（单元、组合）；`noProjectTable(op, param)` 换掉三个循环（另一处一个调用者的也换）；描述写编号"十进制，0 到 9，不带符号、不以 0 开头，1 到 2147483647"（与 `ParseIdentifier` 相同）。复审：已落实；spec 2.1 的 Task 6 行漏了 `permission_matrix_columns_test.go` → 控制者改（`a94a5629`）。
- **Task 7（评审 Important 1 条）**：
  - 评审之前（`e8a20f15`，五条都照做）：(1) `SetArchivedAt` 的 `WHERE id <=` 在每一层都活着（未请求的工作项都在被归档的之后建）→ 存储测试加一个之前建的，T7.21（存储；Task 8 起组合，经写闸门的测试）；(2) 存储测试的说明说"其余每列照旧"，而 `GetIssue` 不读描述和日期 → 用已有的行的转储逐列比较，T7.22（存储）；(3) 组合测试只看 409 的状态 → 也看码 `issue.not_archivable`，T7.24；(4) T7.14 加组合两层（矩阵"Todo 里的"格子、`TestArchivingAWorkItemStampsItsRows`）；(5) 矩阵"只有写的行能 `prepare`"的检查的探针，T7.23（自检）。
  - 评审：归档的"已归档的先于状态组"只在单元一层钉住（R7-order23：`archivable` 放到不写之前，在整个 `bootstrap` 都通过：组合里再归档的都是 Done 里的工作项）；Minor：`asArchived` 重写了矩阵的 `answered`；`archive_issue.go:117-120` 重复了 `update_issue.go:116-118` 的"状态不是项目的"的检查；spec 第 3 节第 5 条说恢复的"已归档的项目里已归档的"一行钉住 T7.2（实为 T7.1）。裁定全部接受（`1e12f05e`）：Cancelled 改到开始组之后 bob 再归档已归档的 Todo 工作项 → 200、戳不变，T7.25（单元、组合）；`asArchived` 建在 `answered` 上；一个 `issueState` 由归档和修改共用（取代 `stateOf`，它只有那两个副本用）；spec 的事实修正。代价：一步测试和一个函数。复审：已落实；Nit：`issueState` 两个写共用，却在 `update_issue.go`，而 `writes.go` 放"每个写共用的部分"（`pickState`）→ 留给整分支评审之后的修复轮（只是移动，不改行为和测试）。
- **Task 8（评审 Important 1 条）**：
  - 评审之前（`b58b3870`）：(1) T8.6 加组合一层（矩阵 `updateIssue` 的"已归档的工作项"），同 T6.1、T4.10、T7.14 的错的"只在单元一层"；(2) "已归档的工作项照样删除"（spec 第 3 节第 5 条、清扫 36，评审敏感）成为 T8.8（实现者的探针 P8-del：单元 `TestDeleteIssueDeletesTheWorkItemAndItsRows` 的 `an archived work item` 一例，组合矩阵 `deleteIssue`，端到端 I15 Task 13 起）；(3) 已归档的项目里 `state_in_use` 的补救：措辞不改，它在那里的第一个拒绝 `project.archived` 的文字（"unarchive it to change it"）补上之前的一步，spec 第 3 节第 9 条和清扫 30 写上这一种；代价：一句文案的五处；(4) 页面的文案只按键核对 → 不改（同之前的每个码）。实现者的偏离：T8.8 的 I15 一层不能先写进行（工具不接受没有运行记录的较晚的层）→ 接受，由 Task 13 的派发带上；忘了的代价是一个端到端的层没记（矩阵在组合一层钉住它）。
  - 评审：`ErrIssueArchived` 的说明"answers a write on an archived work item, but its unarchiving and its deletion"对再归档不对（代码的话里那也是写，答 200）；Minor：spec 2.9 的"删除标签照样从已归档的工作项上摘下它"哪里都没钉住（`DetachLabels` 跳过已归档的，`issue/adapter/postgres` 和整个 `bootstrap` 都通过）；spec 的几句。裁定全部接受（`c6a277c9`）：说明改成"a change of an archived work item, even one that changes nothing"（3.8；再归档、恢复、删除不是改动，不列例外）；`TestDeletingALabelTakesItOffTheWorkItems` 在 alice 删 Bug 之前把 bob 的一个工作项归档，T8.9（组合）；spec 2.1、2.9、第 3 节第 5、9 条。实现者的 `mut.py` 的关切见上（INFRA）。复审：已落实；spec A.1 仍写 138 个文件、Task 8 346 行 → 注明日期的记录（第 5 节数）；plan 的提交信息的 Nit → 不改。
- **Task 9（评审 Important 1 条）**：
  - 评审之前（`73990ae9`）：(1) 探针 P9-len（同长度的下标只按位数比较）每一层都活着 → `TestComparePaths` 加"a[1].z"在"a[2].a"之前，T9.14（单元）；(2) spec 第 3 节第 10 条、2.10 和 plan 说 `TestGenerateRejectsWhatItCannotCheck` 钉住"没有映射的 `date` 被拒绝"，实为 `TestGenerateFollowsTheTypeMapping` → 事实修正；(3) 清扫 59：`date.go` 的类型说明不再说"工作项的起止日期"（Task 10 之前没有）；`cost_test.go` 的"第一个字符串的数组和请求体的第一个日期"不对（M3/P4b 的 `hide_in_more_menu` 才是第一个字符串的数组）→ 不说"第一个"，按情形点名。
  - 评审（只改文档）：`bodyshape.go:66-67` 的 `Open` 的说明"不检查它们，也不检查没有 item schema 的数组的元素"自 Task 9 起不对（`walk` 为数读它们）；Minor：两处说按名字读成员（第一层以下按文档的次序）；"生成的解码器把每个数读进 float64"只对某些形状成立。裁定（`2a7bad0e`，只改文档，控制者核对，不复审）：三处改准。
- **Task 10**：
  - 裁定（不改）：`domain/issue.go:53`、`queries/issues.sql:52` 的"as the API shows it (M4 design 5.2)"和 `issue.go:89`、`patch.go:12` 的"without the fields of later phases"对只到 Task 10 的代码说宽了（接口在 Task 11 才有日期）→ 不改：清扫 59 约束的是写下它的 Task 结束时的注释（这些是 P1 写的），它们是类型的契约，Task 11 结束时又准确（同 Task 1 的 `CreatorID`）。错了的代价：评审者再指出一次。插入时对调日期的探针死在数据库的 `issues_dates_check`、不在读回：是存储一层的发现，不改。
  - 评审：通过，没有问题；它对裁定 (1) 的第一个理由的更正（"Task 10 没动它们"只对裁定的那几句成立，Task 10 扩展了 `NewIssue`、`IssuePatch` 的说明的段落）接受，另一个理由撑得住；不改。
- **Task 11**：
  - 评审之前（`37f1be1e`）：(1) `TestDatesAreKeptInOrder` 的说明和 spec 2.12 说"给它已有的日期什么都不写"，测试只看动态和存的日期（探针 P11-noop-row 重写 `updated_at`、`updated_by_id`，通过单元和整个 `bootstrap`）→ 加断言，T11.5（组合）；(2) 失败信息把日期打成指针地址 → 打日期的文字。500 代替 422 的发现（T9.4、T10.1、T10.3、T10.4、T10.6 死于数据库的日期检查答的 500，期望 422）→ 不改：数据库的检查是兜底，那些发现是断言。
  - 评审：通过；Minor：(1)（plan 写定的）修改的"已归档 409 先于日期的 422"对日期哪一层都没钉住（"先看日期"的变异只在单元的 `dates_test.go:92-100` 经"一个 422"死），与 R7-order23 同类；(2) `update_issue_test.go:118-128` 的"每个字段原样"漏了 Task 10 加的两个日期；(3) `given[T]` 是 `projects.go:118` 的 `named` 的泛型版。裁定：(1) 改（`93c0d878`）：`TestUpdateIssueRefusesUnderTheLock` 加"已归档的工作项给次序不对的日期"一行，`TestDatesAreKeptInOrder` 加一步，T11.6（单元、组合）；(2) 当轮改（P1b 的测试的说明被 Task 10 的字段变宽了），T11.5 加单元一层；(3) 不改：模块之间不导入、不建 helpers 包（P1a 评审第 7 节对按适配器的 `orNull` 的裁定），评审记录列出（第 7 节）。复审（sonnet）：已落实；plan 的提交信息的 Nit → 不改。
- **Task 12**：
  - 实现者的自查（`573feee7`：`deletedWith`、`itemWrite`、`deletesIssue` 的说明，清扫 59）批准。评审之前（`eff79cad`）：(1) `TestDeletingAWorkItemAndChangingItSerialize` 的说明说"bob 的动态在其中"没有断言 → `activityBy` 的断言（先改是 `["name: Fix → Mend"]`，先删是空）；(2) 交错 12 的失败信息一边打 id、一边打名字 → 一种写法；(3) T1.4、T2.4 在交错 4、5 死于 `serializeOn` 的等待（预检拒绝，写从不等，是测试的前提），矩阵给出有名的 403 → 不改：要有名的回答就得让 P1b 共用的 `serializeOn` 报出在等之前就结束的写，评审记录列出；(4) T12-probe.1 成为 T12.2（单元，组合交错 4、5：未删除的行数，spec 第 3 节第 18 条）。
  - 评审：通过；Minor（plan 写定的）：`issue_interleaving_test.go:20-26` 的文件头比文件窄（Task 12 起有删除的一支）→ 修复轮 2（`7a33943d`，只改文档，控制者核对）；Nit：P1b 的 `TestAWorkItemGoneWhileItsChangeWaitsIsNotFound` 的"deleted"一支（裸 SQL）与新的"先删"经路由的重叠 → 交给整分支评审（修复轮删了它，第 4 节）。
- **Task 13**：
  - 周的限额让实现者停下（第 1 节），恢复的是同一个实现者。提交 `e17b20b1`、`b402016d`（T8.8 的 I15 一层，照 Task 8 的裁定）、`0db90e1a`（实现者的自查：`changed` 接受空的旧值，`archivedRow` 经它写，不抄字面）。评审之前（`5b38471f`）：批准 `0db90e1a`（共用帮手的规则）；I15 说再归档回答工作项原样，却只看 `archived_at` 是字符串 → 对第一次的回答 `toEqual`（清扫 15；单元已发现那个探针，不加行）；I15 的 `issue.not_archivable` 的 409 和 I2 的已归档的 409 之前各读一次 `expectIssueArchival`（清扫 27、38）。
  - 评审：通过；Minor ×2（plan 写定的）：I15 第三次归档只看状态；`assert/issue.ts:299` 的 `cardinality(c.conkey) = 1` 悄悄丢掉多列的外键（Go 的 `keysTo` 会失败）。可选：外键图的读移进 `catalog.ts`。裁定都接受（`5874a47f`）：第三次归档之后 `expectIssueArchival`；`keysBelow` 移到 `catalog.ts`、`tablesUnder` 旁边，多列的键按名字失败（探针：`issues` 下一个两列的键让 I15 在 `catalog.ts:65` 失败，之前通过）。复审（sonnet）：已落实；两个 Nit → 不改。
- **Task 14（评审 Important 2 条）**：
  - 实现者带上暂停时记下的两条提示（M2 设计"报哪 16 个"加上只为数读的按文档的次序；M2 交接的"第一个数组的请求体"不说"第一个"）和六份文档对照 HEAD 的逐句核对。评审之前（`df7380e4`）：(1) M3 交接的处理结果加"第 2 节，约定五"一项（P1b 的有），spec 第 7 节写第 2 节；(2) 差异清单的两行（"工作项的创建者的权限""访客删除自己建的工作项"）：在项目里是访客的工作区管理员按管理员的规则删他自己建的（204），不是凭 `AllowCreator` → "不能再凭它删"和一句括号，设计 4.17 同样；(3) M2 设计 3.12 多出的一句（`date` 映射到 `shared.Date`）：对，不改，评审记录记下；(4) A.1 的大小：注明日期的记录。另两条：设计 12 节 P2 的关闭一行漏了 13.1 要的逐 Phase 的约定五的核对（P1a 的也漏）→ 留给整分支评审之后的修复轮；设计的 4 个断的链接：M4 收尾写那些文件 → 不改。
  - 评审：Important (1) 差异清单 :242 漏了按编号取的 403 `forbidden`（看得到项目、不能读：WA-、WM-公），设计 :1084（"都是 404"）、:1133（5.1 没有 403）同样；(2) M3 交接 :146 的"仍未处理"丢了第 3 节的未完项，M1-P3 交接 :58 可以点名 P11、P14。Minor：(3) 设计 :1082"除恢复、删除外一切写 409 `issue.archived`"对再归档不对；(4) M2 设计 :1639 仍把 `format: date` 列在生成器失败的例子里，:503 缺 P2 的日期；Nit (5)、(6)。裁定六条都在修复轮 2 一次改（`fd546f35`，只改文档；比分到最后的修复轮便宜）；差异清单 :241 的"恢复没归档的 404"靠预检读的 Plane（spec A.2）→ 不改。实现者为一致多改的（5.1 的 `getIssue` 一行加 403 F、5.3 的 `issue.archived` 一行、spec 2.1 列出执行中改的设计各节）接受：同一个事实处处一样。复审（sonnet）：已落实；三个 Nit 交给整分支评审的分诊。

**整分支评审之后**
- 一个修复轮（`final-fix-brief.md`），七条裁定：M1 `ErrIssueNotFound` 的说明写全它回答的情形；M2 一个用例一个构造（`deleteOver`、`archiveOver`，`gated` 的创建、修改的构造是 P1b 的，不动）；分诊 1 `issueState` 移到 `writes.go`；分诊 2 删去 P1b 的裸 SQL 的"deleted"一支，留"moved"一支和测试的名字，P1b 的 T5.16 在路由的测试上重新核对，成为 T12.3；分诊 6 设计 12 节 P2、P1a 的关闭一行；分诊 7a–7c；第 7 条 spec 2.1 和文件表。不改：spec 附录 A、Global Constraints 的大小、清扫 21、35、52（注明日期的记录，本文数）；设计第 15 节和本文留给下一个提交。分诊 3（`serializeOn`）、4（`given`、`named`）接受、不改；5（m4-p1b）是负责人的。理由：评审的发现，和"每句话对 HEAD 都对"。错了的代价：两个测试的构造、一个函数的位置、一个交错的一支和文档的文字。
- 范围复审之后的裁定（`review-doc-brief.md`，本提交）：N1 的一句在本提交，不再复审；N2 不改。
- 控制者的观察（整分支评审的合并说明）：本地的 `main` 在 `5ce30a1a`（P1a 的合并），落后于 `origin/main` 的 `7db78788`；`--no-ff` 合并的第一个父提交要是 `7db78788`。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 0、Minor 2；分诊另加修复轮的六项（1、2、6、7a、7b、7c），都不改行为。评审逐条核对过：
- 一条取锁路径照 HEAD 运行的样子：五个写（创建、修改、删除、归档、恢复）都经 `Locks.run`，它是 `WriteGate.Enter` 唯一的调用者；两个读经 `see` 和 `decide`；没有 `precheck`、`lockAndDecide`、`decide` 的近似副本；
- 逐个操作、按项目角色的访客规则：一条规则在 `Locks.decide`（`ErrNotVisible` → 404；否则 `HiddenFrom` → 404，在角色的 403 之前），预检与锁下用同一个 `decide`；没有泄露：隐藏的与不存在的是同一个值，在预检答（不进闸门、不等锁、没有 409 或合并之后的 422），唯一的时间差是多一次判定的读（spec 第 6 节接受）；
- 把闸门的位置、池的连接和行锁都当作资源，在 HEAD 的每个写上重推死锁：闸门只在什么都不持时进，没有连带进闸门，每条边沿全局的次序或从闸门的持有者到行锁，没有环；约定五：P2 的每个写者先持项目的 N；
- 门禁、竞争的运行、plan 的重放和变异的抽样（第 1 节）。

| 发现 | 处理 |
|---|---|
| **M1** `issue/domain/errors.go:20-21` 的 `ErrIssueNotFound` 的说明比代码窄：只说不存在、已删除、看不到项目，而它也回答访客规则隐藏的工作项（`lock.go:196`）和按编号取在工作区之后的每种找不到（`get_issue_by_identifier.go:66-91`）；访客规则所依赖的"与不存在的同一个值"只在这里声明 | `b57ae344`：`errors.go:20-23`"…answers a work item that does not exist or is deleted, one whose project the caller does not see, one the guest rule hides from him, and every miss of a work item's identifier after the workspace, all with this one value (M4 design 3.4, 3.7, 5.3)"，由 Task 5 的块写（整句从 Task 5 起才成立；那句原是 P1b 的文字，P2 的 Task 没写过它）。范围复审逐个核对了 HEAD 答这个值的每一处（`lock.go:158-159`、`:161`、`:196-197`、`:127-128`、`:137`、`:142`，`get_issue_by_identifier.go:69`、`:81`、`:88`），都在说明里；N2 见下 |
| **M2** `issue/app/lock_test.go:66-80` 的闸门测试三次在行内重写删除、归档的构造（`newDelete`、`newArchive` 的副本）：构造按值捉住 `w.locks`，`gated` 之后才换；用例加一个端口要改三处 | `b57ae344`：`deleteOver(w)`（`delete_issue_test.go:17`）、`archiveOver(w)`（`archive_issue_test.go:35`）；`newDelete`、`newArchive` 和闸门测试（`lock_test.go:67`、`:70`、`:76`，在 `gated` 之后）都调它们；`gated` 自己的创建、修改的构造是 P1b 的，不动。范围复审：T4.21、T7.19 在 HEAD（`lock_test.go:90`）和 plan 的 Task 4、Task 7 的树上都被发现，都是断言失败 |
| **分诊 1**：`issueState` 两个写共用，却在 `update_issue.go` | `b57ae344`：逐字节移到 `writes.go:31-40`（`pickState` 旁边，范围复审比较了 487 个字节），文件头（`:14-16`）写上"the state it may put the work item in and the state it is in"；调用者是 `update_issue.go:122` 的 `check` 和 `archive_issue.go:116` 的 `archivable`。T7.25、T7.4、T7.18、T7.20、T7.24（单元一层读它的）照样被发现 |
| **分诊 2**：P1b 的 `TestAWorkItemGoneWhileItsChangeWaitsIsNotFound` 的"deleted"一支（裸 SQL）与 Task 12 的"先删"经路由是同一个交错，又造出没有操作造得出的状态（工作项删了、它的行没删） | `b57ae344`：删去那一支，"moved"一支成为测试的正文（`issue_interleaving_change_test.go:46-87`），名字不变（`raceall.sh`、plan 的 `Run:` 照旧）；说明写明没有操作移动工作项、所以在数据库里移，删除的是 `TestDeletingAWorkItemAndChangingItSerialize` 的。P1b 的 T5.16（锁下重读留下已删除的工作项）成为 T12.3（组合），在 `issue_interleaving_deletion_test.go:123` 被发现（PATCH 答 500，期望 404）；另有两个单元测试发现它，行只记裁定的组合一层。范围复审：路由的测试比删去的一支更强（bob 的动态为空；Web 的三行都在删除的时刻由 alice 删）；P1b 的 T5.17、T6.3 照样在裁剪过的测试被发现（`:85`、`:71`），所以"moved"一支仍逼出它的交错、仍是"仍在这个项目"的组合一层；两个交错 `-race -count=5` 10/10 |
| **分诊 3**：`serializeOn` 不能指名在等之前就结束的写 | 接受、不改：前提的发现是真的期限失败（T1.4 在交错 4、5："no statement waited for a row lock of projects within 5s"）；矩阵对同一个变异给出有名的 403；`serializeOn` 是 M3 共用的帮手，改它要动每个交错 |
| **分诊 4**：`given[T]` 与 `named` | 接受、不改：按适配器的转换帮手（P1a 评审第 7 节的先例）；共用要一个 helpers 包或跨模块的导入 |
| **分诊 5**：空的 `m4-p1b` worktree | 只记下（第 7 节）：删不删由负责人定 |
| **分诊 6**：设计 12 节 P2 的关闭一行（:2108）漏了 13.1 要的约定五的逐个核对；P1a 的（:2068）也漏 | `8c42351b`：P2 一行加"M3 交接第 2 节中约定五的逐个核对（P2 的批量写：删除工作项的五条语句，归档、恢复的 `SetArchivedAt`）"；P1a 一行加"…和约定五的逐个核对（P1a 的批量写：…）"，范围复审核对了它与 P1a 评审 :97 相符 |
| **分诊 7a**：设计 5.1 的六个 P3、P4 的读只写 404，而 9.2 给"看得到而不是项目成员的工作区管理员"403 | `8c42351b`：:1136、:1138、:1141、:1145、:1149、:1154 改成"404；403 F"，同 9.2 :1701 |
| **分诊 7b**：设计 3.4 :207"项目访客创建的工作项，他不能删"读起来像是 `AllowCreator` 的效果 | `8c42351b`："…，他不能凭 `AllowCreator` 删（在项目里是访客的工作区管理员照管理员的规则删他自己建的；Plane 的页面…）" |
| **分诊 7c**：spec 第 7 节的"留下的条目"与交接自己的"仍未处理"一行不一致 | `8c42351b`：M2 交接第 4 节一行（:262）加草稿发布（P8）；M3 交接第 2 节一行（:263）加加关联的另一端的预检（P3）、物理删除账户（P9）；第 4 节一行（:265）加 `archive_in` 的说明（P6） |
| **分诊 8**：附录 A、Global Constraints 的大小、清扫 21、35、52；架构的那份 INFRA 日志；M2 设计 3.12 多出的一句；设计的 4 个断的链接 | 不改：附录和大小是注明日期的记录（第 5 节数）；那份日志在仓库外，合并的结果靠重跑的；3.12 的一句对（五个模块的映射；`checkDate` 照 `ParseDate` 的规则、不导入 `shared`）；链接指向 M5–M8 的 `M4-closeout.md`，M4 收尾写它们 |

**修复轮**（`b57ae344`：7 个代码、测试文件和重新组装的 plan；`8c42351b`：设计和 spec）：
- 检查（`final-fix-report.md`）：`issue/domain`、`issue/app` `ok`；两个交错 `-race -count=5`：10 次顶层、10 个子测试，0 行 40P01、死锁、DATA RACE；矩阵 `ok`；`make lint-go` 两段 `0 issues.`；`make test` 47 个 `ok`；`make gen-check` 通过、树干净；`make lint-web` 没有命中；`doclinks.py` 在设计、spec 和 plan 上只有设计的 4 个已知的断的链接。变异，每个在新的副本里：T12.3 在 Task 12 的树和最终的树上被发现；T4.21、T7.19 在构造改过之后仍在单元一层被发现；读 `issueState` 的 T7.4、T7.18、T7.20、T7.24、T7.25 照样被发现；另有 Task 4、Task 7 每行的单元一层和最终的树上全部 42 行带单元一层的，都被发现，没有 INFRA。plan：从基点 592 个块都对，逐 Task 应用之后只差生成物，`compose_plan.py` 逐字节重现 plan（12,727 行）。
- 裁定没写死、由实现者选的：M1 的说明由 Task 5 的块写（理由同 Task 10 的裁定：清扫 59 约束写下它的 Task 结束时的句子）。范围复审认为对。
- 实现者记下的：P1b 的注明日期的记录（P1b plan :4161，spec :137、:427）仍把裁剪过的测试写作 T5.16 的组合一层；在 HEAD 上那个发现是 Task 12 的路由的测试（T12.3）。

**修复的范围复审**（opus，`fd546f35..8c42351b`，`$M4TMP/p2final/rereview.md`，副本在 `$M4TMP/p2final/rereview/`）：结论"N1 的一句加上之后可以合并"：七条裁定都照裁定落实、对 HEAD 都对；新的 Minor 1 条（N1）、Nit 1 条（N2）。它核对了：`ErrIssueNotFound` 回答的每一处；没有剩下构造的副本，T4.21、T7.19 在 HEAD 和 plan 的 Task 4、Task 7 的树上被发现；`issueState` 逐字节移动；T12.3 在路由的测试上被发现，删去的一支没有丢掉路由的测试不钉住的性质，"moved"一支仍逼出它的交错；P1a 的关闭一句与它的评审相符；142 个文件（新建 34、修改 108）、生成物 11 个、文档 8 个和设计；执行中改的设计各节；plan 592 个块，逐 Task 应用、`make gen` 之后与 HEAD 3,410 对 3,410 个文件、0 个差异，`compose_plan.py` 逐字节重现（12,727 行）；`make lint-go`、`make test`（47 个 `ok`，89 秒）、`make lint-web` 的各部分、`doclinks.py`；两个交错 `-race -count=5` 10/10。它指出实现者的报告说 spec 第 7 节"每行照交接的次序"对 M3 第 4 节一行不全对：只是报告的说法，次序不是事实的问题。

**本提交改正的遗留**：

| 项 | 本提交 |
|---|---|
| **N1**（Minor，文档；7c 的同类）spec 第 7 节 M1-P3 一行（:260）的"留下的条目"漏了描述历史的地址（P4）：交接自己的"仍未处理"一行（`handoffs/M1-P3-trim-platform.md:58`）留着"描述历史的地址（M4 的 P4）"，它的 P2 的处理结果以 spec 第 7 节为来源（:60）；7c 只查了引 M2、M3 交接的行 | spec :260 的那一格以"描述历史的地址（P4）；"开头。本文的作者核对了交接 :58 和地址一项的两半（:38"描述历史和按编号查找的地址"；P2 做了按编号查找的一半） |
| **N2**（Nit）`lock.go:137` 的一支（锁下读到的工作项在别的项目里）`ErrIssueNotFound` 的说明没点名 | 不改：没有操作走得到（设计 :1406：工作项的项目不会变），`lockAndDecide` 自己的说明（`lock.go:118-121`）写了它 |
| 设计第 15 节 P2 一行（:2515） | "已完成：[spec]…、[plan]…、[评审]…（执行时 14 个 Task；整分支评审之后一个修复轮）"，照 P1b 一行的写法 |

## 5. 最终实现与 plan 的差异

plan 是"执行时的 plan"：每个修复轮也改了它的 Task 的块，代码块写的是提交的代码（第 3 节）。所以这里不列逐个提交的漂移表，只写复现的结果、执行中改了的生产代码、注明日期的记录在 HEAD 上的数和执行中加的变异。比较的是修订之后第一次提交的 plan（`dfe4c257`）。

**复现**（本文写之前，树干净；`$M4TMP/p2sdd/rd/replay.sh`，日志 `replay.log`、`replay-treediff.txt`）：
- 两份新的解压 `$M4TMP/p2tools/base-7db78788.tar`。一份上 `node planapply.mjs <plan> check`："592 blocks {"replace":552,"whole":6,"file":34}; all ok"。
- 另一份上放进 HEAD 的 spec、plan 和设计（phase 的文档，没有块写它们），按顺序 `apply` Task 1–14，每个 exit 0；写的文件数依次是 16、16、12、23、30、11、35、14、20、21、11、4、6、6。
- `node treediff.mjs <复现> <worktree>`："3409 and 3474 files; 75 differences"：10 个生成物两边都有、内容不同，生成的 `delete_issue.sql.go` 只在 worktree，另 `.git` 和 `.superpowers/` 下的 63 个；没有一个手写的文件或文档不同。复现不跑生成命令；范围复审在它的复现里 `make gen` 之后 3,410 对 3,410 个文件、0 个差异。
- HEAD 的 11 个生成物与 plan 最后的钉值相同（`pins.py`：SHA-256 和行数）：`project/…/access.sql.go` `5f707683…`（plan :298）、`project/…/server.gen.go` `61cbb8a6…`（:1056）、`delete_issue.sql.go` `58673549…`（:2734）、`issue_ports.sql.go` `b3a2acdc…`（:4174）、`issues.sql.go` `235bd7f3…`（:9446）、`models.go` `59e0fe88…`（:9447）、`update.sql.go` `5beaddd3…`（:9448）、`openapi.yaml` `0ddf0747…`（:10679）、`bodyshape.gen.go` `132fca76…`（:10680）、`issue/…/server.gen.go` `4607cfaf…`（:10681）、`schema.gen.ts` `9b409bb5…`（:10682）。
- 后来的几轮改过的设计：`64c2acc9`（第 15 节的一行）、`dfe4c257`（3.3、3.4、6.5）、`2622d189`（6.7）、`3d9a9aa5`（3.7）、`df7380e4`、`fd546f35`（4.17、5.1、5.3）、`8c42351b`（3.4、5.1、第 12 节）和本提交的第 15 节一行。

**执行中改了的生产代码**（`git log --name-status 7db78788..8c42351b`，非测试的 Go、SQL、契约、配置和文案；`prodfiles.py`、`codelines.py`）。各 Task 的第一个提交之外：
- **没有一处改了行为。**
- 改了代码、行为不变：`2a51963f`（Task 5 的修复轮 2）：`identifier.go` 用 `strings.CutLast`，去掉自己的 `cutLast`；`lock.go` 的 `seeWorkspace` 交给 `decide`（零的项目 id，同样把 `ErrNotVisible` 答成 `workspace.not_found`）。`1e12f05e`（Task 7 的修复轮 2）：`issueState`（错误的文字与原来两处相同）取代 `update_issue.go` 的 `stateOf` 和 `archive_issue.go` 的同一个检查。`b57ae344`（修复轮）：`issueState` 逐字节移到 `writes.go`。
- 契约的描述（`api/modules/issue.yaml` 的 `getIssueByIdentifier`，生成的打包结果和 `schema.gen.ts` 随之）：`3d9a9aa5`（标识的平手规则对不止一个项目，原来写两个）、`eb69fc36`（编号的写法，多两行）。
- 只改注释：`71f42734`（`shared/authorize.go`、`access/domain/decide.go`：`OwnOnly`、`Decide`）、`2622d189`（`lock.go` 引 8.4）、`b02c7d5f`（`get_issue_by_identifier.go` 不指名路由）、`c6a277c9`（`ErrIssueArchived`）、`73990ae9`（`shared/date.go` 的类型说明）、`2a7bad0e`（`bodyshape.go` 的三处）、`b57ae344`（`ErrIssueNotFound`；`writes.go` 的文件头）。
- 其余各轮只改测试、测试的数据和文档。Task 8 的 `c6a277c9` 改的是删除标签的组合测试，钉住已有的行为（已归档的工作项上照样摘下已删除的标签），不是连带本身。

**注明日期的记录在 HEAD 上的数**（spec 附录 A 和 Global Constraints 照旧不改；数都是本文的作者在 HEAD 上量的：`planlines.py`、`filecount.py`、`rows.py`、`verdicts.py`、`sizes.py`、`matrixcount.py`，在 `$M4TMP/p2sdd/rd/`）：
- **A.1**（:279）：
  - 量法：A.1 的各 Task 的行数是 `compose_plan.py` 量的，不含 Task 之间的三行（空行、`---`、空行），比它自己说的"从它的标题到下一个标题"少三行（Task 14 到文件末尾，相同）：`dfe4c257` 的 plan 从标题到标题量是 Task 1 之前 178、Task 1 785……，减去三行正是 A.1 的数。整分支评审的表把 A.1 的数与 HEAD 从标题到标题的数并列，差里含这三行（Task 10 没改，1,147 → 1,150）。
  - 用 A.1 的量法，A.1 → HEAD：Task 1 之前 175 → 180；1 782 → 793；2 729 → 746；3 859 → 885；4 1,430 → 1,486；5 1,296 → 1,369；6 582 → 637；7 1,519 → 1,630；8 346 → 420；9 1,126 → 1,145；10 1,147 → 1,147；11 800 → 888；12 241 → 398；13 731 → 768；14 168 → 193。plan 11,973 → 12,727 行（`wc -l`），多 754 行，都是修复轮的块、执行中加的变异行和各 Task 的说明。
  - 长过约 1,500 行的只有 Task 7：1,630 行（修订之后 1,519；它的两个修复轮，整分支评审之后的修复轮的 `archiveOver` 和 `issueState`）。"其次 Task 4（1,430 行）、Task 5（1,296 行）"今天是 1,486、1,369。
  - 文件：A.1 写手写的 138 个（新建 34、修改 104），HEAD 是 142 个（34、108；整分支评审时是 140 个，spec 2.1 现在写 142）：执行中多改了四个已有的文件：`bootstrap/issue_cascade_test.go`（Task 8 的修复轮 2）、`e2e/fixtures/assert/catalog.ts`（Task 13 的修复轮 2）、`issue/app/writes.go`、`bootstrap/issue_interleaving_change_test.go`（整分支评审之后的修复轮）。生成物 11 个、文档 8 个照旧（`git diff --name-status 7db78788 8c42351b`）。
- **A.3**（:310–335）：
  - "只在单元一层被发现的（按性质，25 个）"今天是 24 个：A.3 列的 25 个里 5 个不再只在单元一层，执行中加了 4 个。
  - 不在了的 5 个，A.3 的理由是错的，运行显示它们在组合一层死：T4.8（"同 P1b 的 T5.11……表的次序在它之下等不起来"：取锁形状的 `deleteIssue` 一行发现它，变异先写负责人的行，在等 `issues` 时已写了 `issue_assignees`，`issue_write_locks_test.go:176`，Task 4 的评审）；T4.10、T6.1、T7.14、T8.6（"组合一层和端到端不读契约的声明"：每个组合的请求经 `call`，它的 `CheckResponse`（`apitest.go:154`）跑 `checkProblemCode`（`problems.go:66`），答出一个没声明的码就失败；T6.1 在按编号取的组合测试和矩阵的 X 格子，T4.10 在矩阵的已归档的 `deleteIssue` 格子，T7.14 在矩阵的"Todo 里的"格子和归档的盖戳测试，T8.6 在矩阵的 `updateIssue`"已归档的工作项"；Task 6 的评审发现，Task 7、8 的实现者核对）。
  - 执行中加的、只在单元一层的 4 个：T1.13（三个之外的角色：表的 `CHECK`）、T1.14（`creatorRoles` 的管理员，同 T1.2）、T7.23（矩阵的自检）、T9.14（同长度的下标：`bodyshape` 的测试直接驱动，同 T9.5）。
  - 只在存储一层的：A.3 写 4 个，今天 7 个：加了 T4.22（删除按 id 的次序取工作项）、T5.17（`ProjectIdentifiers` 的次序：`MatchIdentifier` 不靠它）、T7.22（归档也写 `GetIssue` 不读的一列）。只在守卫一层的 3 个照旧。只在组合一层的：A.3 写 3 个，今天 6 个：加了 T3.2（矩阵的逐列的事实核对）、T8.9（删除标签跳过已归档的）、T12.3（它的行只记裁定的组合一层，另有两个单元测试发现它，第 4 节）。
- **A.4**（:339–343）：对 HEAD 仍成立：矩阵 1,259 个叶子、工作项 393 格、每个操作的格数相同，其余 865 格是 M3 的（本文的作者的运行，第 2 节；顶层 1.5 秒，A.4 写 1.53 秒）；`racenames.py` 的定义在 HEAD 上仍是 63 个测试。A.4 的端到端 120/123 是原型上的（原型不是 git 检出，S3 失败）；在 worktree 里是 121/123。
- **A.5**（:347）："547 个块"今天是 592 个（{"replace":552,"whole":6,"file":34}，本文的复现）：Task 的修复轮从 547 起逐轮改到 588 个（Task 14 的修复轮 2 之后），整分支评审之后的修复轮 592 个。
- **A.6**（:418 和清扫的表）：
  - 变异 119 个，今天 143 个（`rows.py`）：执行中加的 24 行（5.1）。合并的结果（原型的结果文件加覆盖文件，`verdicts.py`）：Pass A 209 层、Pass B 100 层、Pass C 275 层，都是 `killed`，没有留下的、没有 INFRA。
  - 各层发现的：单元 90 → 107，组合 84 → 106，端到端 32 → 33，存储 22 → 26，守卫 4 → 4。只在一层的：单元 25 → 24，存储 4 → 7，守卫 3 → 3，组合 3 → 6（见 A.3）。"（Task n 起）"的：Task 2 起 1 → 3，Task 3 起 6 → 7，Task 4 起 7，Task 5 起 0 → 1，Task 6 起 13 → 15，Task 7 起 1，Task 8 起 0 → 1，Task 11 起 18，Task 12 起 6，Task 13 起 32 → 33；端到端的发现 33 次 → 34 次（T8.8 在 I15）。
  - 清扫 9（:365）、18（:374）和类别表的"清扫 18：契约声明的码"一行（:450，层只写"单元"）照 A.3 说 T4.8、四个契约的变异只在单元一层：同上，不对。
  - 清扫 21（:377）"单元的拒绝表用 `errors.Is` 比较码"：今天访客规则、按编号取和 `getIssue` 的单元拒绝表逐值比较（`reflect.DeepEqual`：`guest_rule_test.go:59`、`:98`，`get_issue_by_identifier_test.go:152`，`get_issue_test.go:78`；Task 5 的修复轮 2、3），比码更强，正是那个性质（隐藏的与不存在的同一个值）。
  - 清扫 35（:391）说 `guestViewAll` 由 acme 的管理员经项目的存储设，并判定"规则到得了的状态"：这个判定连原型也不对：原型的种子确实由 acme 的管理员设（`dfe4c257` 的 plan :2041-2052），而他不是 acme 任何项目的成员，规则不让他改项目（`updateProject`）。Task 3 的评审发现，修复轮（`dcd30b8d`）改由项目的管理员 PA 设（`permission_matrix_seed_issues_test.go:99-112`："acme's admin, a member of no project of acme, may not"），改了 2.5 的同一句，没改这一行，所以它的说法对 HEAD 的种子也不对了。台账留给评审记录的清单漏了它，整分支评审补上。
  - 清扫 52（:408）"没有重写的辅助"：原型就不对：修订加的单元闸门测试在行内重写删除、归档的构造（整分支评审的 M2）；修复轮之后对 HEAD 成立（`deleteOver`、`archiveOver`）。
- **Global Constraints 的大小**（plan :24，标明是最终的原型上量的；HEAD 上 `wc -l`）：`api/modules/issue.yaml` 481 → 483（`eb69fc36` 的编号的写法）；`issue_interleaving_test.go` 394 → 393；新建的文件中最长的 `issue/app/archive_issue_test.go` 195 → 203（修复轮的 `archiveOver`）、`get_issue_by_identifier_test.go` 170 → 168、`permission_matrix_issue_columns_test.go` 162 → 166；不变的：`project.yaml` 1363、`tools/keywords.json` 2670、`permission_matrix_test.go` 406、`issue/app/fakes_test.go` 400。分支改过的手写的代码、测试文件中，超过约 400 行的只有点名的例外 `permission_matrix_test.go`（406）；其次是 `fakes_test.go` 400、`issue_interleaving_test.go` 393、`permission_matrix_coverage_test.go` 387、`bootstrap/ports_test.go` 378。

### 5.1 执行中加的变异

执行中加的变异是 plan 的变异表的行（不进 spec 附录 A），定义在 `$M4TMP/p2tools/mutants_p2_{a,b,c}.py`，中文在 `mutants_zh_p2.py`，结果在覆盖文件 `mut-results-{a,b,c}-amend.json`。各 Task 的数目和位置（行号是 HEAD 的 plan 的，本提交不改 plan）：

| Task | 表的行 | 原型的 | 执行中加的 | plan 的表（行号） |
|---|---|---|---|---|
| 1 | 15 | 11 | T1.12–T1.15 | :962–976 |
| 2 | 12 | 9 | T2.10–T2.12 | :1712–1723 |
| 3 | 2 | 1 | T3.2 | :2610–2611 |
| 4 | 22 | 21 | T4.22 | :4079–4100 |
| 5 | 18 | 16 | T5.17、T5.18 | :5455–5472 |
| 6 | 3 | 2 | T6.3 | :6110–6112 |
| 7 | 25 | 20 | T7.21–T7.25 | :7721–7745 |
| 8 | 9 | 7 | T8.8、T8.9 | :8160–8168 |
| 9 | 14 | 13 | T9.14 | :9305–9318 |
| 10 | 14 | 14 | — | :10455–10468 |
| 11 | 6 | 4 | T11.5、T11.6 | :11352–11357 |
| 12 | 3 | 1 | T12.2、T12.3 | :11756–11758 |
| 合计 | 143 | 119 | 24 | |

每行（"类别"是 `mutants_p2*.py` 的 `class`；"层"里"Task n 起"是较晚的 Task 的检查；交错 4、5 即 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`）：

| 行 | 改坏 | 类别 | 层 | 谁发现 |
|---|---|---|---|---|
| T1.12 | 角色的拒绝丢掉判定读到的 `Grant`：别人的工作项给访客答 403 | `guest` | 单元（`access/domain`）；组合（矩阵 `updateIssue`、交错 4、5，Task 2 起） | Task 1，评审之前 |
| T1.13 | 三个之外的角色的拒绝丢掉 `Grant` | `guest` | 单元（`access/domain`；按性质） | Task 1，评审之前 |
| T1.14 | `creatorRoles` 没有管理员 | `creator` | 单元（`access/domain`；按性质） | Task 1 的评审（Important） |
| T1.15 | `Authorizer` 拒绝时丢掉 `Grant` | `guest` | 单元（`access/app`）；组合（矩阵 `updateIssue`、交错 4、5，Task 2 起） | Task 1 的评审 |
| T2.10 | 访客规则按 id 的次序认他自己的：创建者在调用者之前的别人的工作项给他看 | `guest` | 单元（`issue/domain`、`issue/app`）；组合（矩阵 `getIssue`、`updateIssue`，交错 4、5） | Task 2，评审之前（实现者） |
| T2.11 | 同上，创建者在调用者之后（T2.10 的镜像） | `guest` | 单元（`issue/domain`、`issue/app`）；组合（交错 4、5；矩阵 `getIssue`、`updateIssue`，Task 3 起） | Task 2 的评审（Important）；矩阵两层是 Task 3 的评审的提议 |
| T2.12 | 访客规则隐藏的工作项答一个文字不同的 `issue.not_found` | `guest` | 单元（`issue/app`，Task 5 起）；组合（`TestAWorkItemIsFoundByItsIdentifier`，Task 6 起） | Task 5 的修复轮 3（控制者对实现者的提示的裁定） |
| T3.2 | 工作项表的"建了它的项目成员"一列用 PA 的账户 | `matrix` | 组合（`TestPermissionMatrix` 的自检） | Task 3 的评审 |
| T4.22 | 删除按 id 的次序（`<=`）取工作项 | `predicate` | 存储（`issue/adapter/postgres`） | Task 4 的评审 |
| T5.17 | `ProjectIdentifiers` 不按 `id` 的次序 | `order` | 存储（`project/adapter/postgres`） | Task 5，评审之前（实现者的探针）；数据由 Task 5 的评审改 |
| T5.18 | 没有项目的标识对得上时答一个文字不同的 `issue.not_found` | `scope` | 单元（`issue/app`）；组合（`TestAWorkItemIsFoundByItsIdentifier`，Task 6 起） | Task 5 的评审 |
| T6.3 | 处理函数不论路径的工作区，都把 `acme` 交给用例 | `scope` | 单元（`issue/adapter/http`）；组合（`TestAWorkItemIsFoundByItsIdentifier`） | Task 6 的评审 |
| T7.21 | 归档按 id 的次序（`<=`）取工作项 | `predicate` | 存储；组合（`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`，Task 8 起） | Task 7，评审之前（实现者） |
| T7.22 | 归档也写 `GetIssue` 不读的一列（`description_html`） | `store` | 存储 | Task 7，评审之前 |
| T7.23 | 矩阵的自检放过没有 `write` 而带 `prepare` 的行 | `matrix` | 单元（`TestMatrixViolationsCatchesEachGap`） | Task 7，评审之前 |
| T7.24 | 状态不能归档的工作项答项目已归档的 409 | `interface` | 单元（`issue/app`）；组合（矩阵 `archiveIssue`、`unarchiveIssue`，`TestArchivingAWorkItemStampsItsRows`） | Task 7，评审之前 |
| T7.25 | 归档先看状态组、再答已归档的工作项 | `order` | 单元（`issue/app`）；组合（`TestArchivingAWorkItemStampsItsRows`） | Task 7 的评审（Important，R7-order23） |
| T8.8 | `deleteIssue` 对已归档的工作项答 409 `issue.archived` | `decision` | 单元（`issue/app`）；组合（矩阵 `deleteIssue`）；端到端（I15，Task 13 起） | Task 8，评审之前（实现者的探针 P8-del） |
| T8.9 | 删除标签只从没归档的工作项上摘下它 | `predicate` | 组合（`TestDeletingALabelTakesItOffTheWorkItems`） | Task 8 的评审 |
| T9.14 | 下标与下标只按位数比较 | `paths` | 单元（`bodyshape`） | Task 9，评审之前（实现者的探针 P9-len） |
| T11.5 | 什么也不改但给了日期的修改照样重写工作项 | `no-op` | 单元（`issue/app`，Task 11 的评审加的）；组合（`TestDatesAreKeptInOrder`） | Task 11，评审之前 |
| T11.6 | 先查日期的次序、再拒绝已归档的工作项 | `order` | 单元（`issue/app`）；组合（`TestDatesAreKeptInOrder`） | Task 11 的评审 |
| T12.2 | 在锁下被拒绝的删除照样删除并提交 | `decision` | 单元（`issue/app`）；组合（交错 4、5：未删除的行数） | Task 12，评审之前（实现者的探针） |
| T12.3 | 等锁期间被删除的工作项照样被写（P1b 的 T5.16） | `decision` | 组合（`TestDeletingAWorkItemAndChangingItSerialize`） | 整分支评审的分诊 2（修复轮） |

**原型的行在执行中加的层**（错的"只在单元一层"之外，都由运行确认）：T4.8 加组合（取锁形状 `TestEachIssueWriteLocksItsProjectFirst`；Task 4 的评审）；T4.10 加组合（矩阵 `deleteIssue` 的已归档的格子；Task 6 的评审的裁定）；T6.1 加组合（`TestAWorkItemIsFoundByItsIdentifier`、矩阵 `getIssueByIdentifier`；Task 6 的评审）；T7.14 加组合（矩阵 `archiveIssue`、`unarchiveIssue`，`TestArchivingAWorkItemStampsItsRows`；Task 7，评审之前）；T8.6 加组合（矩阵 `updateIssue`；Task 8，评审之前）；T8.8 加端到端（I15，Task 13 起：照 Task 8 的裁定，由 Task 13 的实现者跑，`b402016d`）。

**台账点名的、没有成为行的探针**（在 `$M4TMP/p2sdd/` 的探针文件里，结果在各 Task 的报告）：P9-any、P9-quad（Task 9，被原有的测试发现）；T8.6-c、P8-code（Task 8，被矩阵或单元发现）；插入时对调日期的探针（Task 10，死在 `issues_dates_check`：存储一层）；T12-probe.2（未删除的行数撑得住 T12.2）、T12-probe.3（P1b 的 `TestTwoChangesOfAWorkItemSerialize` 已发现它，`activityBy` 的断言之后也在删除的交错被发现）；P13-asis（单元已发现，`toEqual` 之后也在 I15）；评审者的探针 R7-order23（成为 T7.25）、P13r-early 和"锁项目之前读工作项"（Task 12 的评审，两种顺序都被发现）；范围复审重跑的 P1b 的 T5.17、T6.3（裁剪过的测试）；预检的 31 个 `PF.*`（第 1 节）。没有等价的变异：143 个都被发现。

## 6. 移交事项

同一个里程碑之内的后续 Phase（写进它们的 spec；spec 第 5 节各行和第 7 节的"留下的条目"照录，另加执行中交出的）：
- **P3**：删除工作项连带子树（3.6、3.9）：`DeleteIssue` 的步骤加子树和关联行，`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 和 `expectIssueDeleted` 都去掉 `subItems`（`issues.parent_id`）的豁免（spec 第 3 节第 14 条的裁定：有约束力的交接），I17 用 `expectIssueDeleted`；已归档的父工作项、关联的两端 409 `issue.archived`，已归档的不能成为父工作项、关联的对方（3.8）；删除工作项与关联的交错（11、13）重推第 3 节第 7 条的死锁论证（关联行按 `id` 升序，约定五的例外）；M3 交接第 2 节的加关联的另一端的预检。`expectIssueDeleted` 走到的外键多于一列时按名字失败（`catalog.ts` 的 `keysBelow`，同 Go 的 `keysTo`）。设计 5.1 的 P3、P4 的读的行已是"404；403 F"（分诊 7a），P3、P4 的 spec 照抄。
- **P4**：已归档的工作项上评论 409 `issue.archived`；访客规则用到评论、表情、描述版本的读写（挂在工作项下面的一切）；M1-P3 交接的地址一项中描述历史的地址（N1）；同一交接的动态的字段名、评论的 `access`。
- **P5**：列表按访客规则过滤（`VisibleProjects` 带 `guest_view_all_features`，3.4）；`?archived=true` 的列表（规则 `issue.archive`）；`Issue` 列表行的 `start_date`、`target_date`；端到端里访客取别人的工作项的 404（I20 的接口版本，设计 12 节 P2 的第 13 个任务）；M3 交接第 3 节的个人主页的一行。；**P5接收结果（2026-10-26）**：T3/T9/T10/T12及T14已验证日期/none/重复AND、issue.read与issue.archive分支、OwnOnly/隐藏父计数和个人主页；T15 I15/I19/I20补归档恢复、三关系及隐藏404四格接口故事。搜索P8与页面P12–15仍开放，实际证据见 [P5 review §2](../reviews/P5-issue-list-review.md#2-验收标准逐条核对)。
- **P6**：删除、归档、恢复、修改发布事件（`issue.deleted`、`issue.archived`、`issue.unarchived`、`issue.updated`，3.19）；自动归档（第 15 行）；`ProjectGate.Lock` 加 `archive_in`，M3 交接第 4 节的 `archive_in` 的说明；M2 交接第 3 节的自动归档推进固定时钟的测试。
- **P8**：`searchIssues` 照用 `issue.search` 和 `foldIdentifier`（3.18）；M2 交接第 4 节的草稿发布。
- **P9**：账户物理删除之后 `created_by_id` 为空的工作项：访客规则隐藏它，`AllowCreator` 不认任何人；P2 只在单元一层核对这两半（假存储的 `creatorOf`，T2.3、T2.9，附录 A.3），P9 加组合一层的用例；M3 交接第 2 节的物理删除账户、第 4 节的删除断言的限时轮询。
- **P10a、P11、P13、P14**：旧的工作项地址的守卫（P10a）；前端的两处 `/work-items/` 字面量（两个 service 在 P11 换掉）；`/:workspaceSlug/browse/:workItem` 经 `getIssueByIdentifier`（P14）；I15 的页面版本、已归档的详情只读，I1、I2 的页面版本带日期（P13、P14）。
- **P3 之后各后端的 Phase**（到 P9）：各自的批量写按约定五核对（M3 交接第 2 节），各自的操作进规则表、操作名、矩阵（M3 交接第 3 节，完整性测试和 `apitest.Main` 随之通过）；工作项列表的访客规则在各列表的 Phase；M2 交接第 3 节的 P3、P4、P7 的写工作项的用例用时钟写 `updated_at`。
- **收尾**：3.23 中 P2 的行由 Task 14 写好，收尾逐行核对；三份交接的 `status` 在 M4 收尾随 review 关闭；第 3 节第 3、8 条的修订已做（设计 3.3 的 `ProjectGate` 一句和 6.5 的 `ProjectGate` 一行，差异清单的"归档、恢复"一行），收尾核对它们；M2 交接第 4 节的 map 型、递归的请求体 schema 在 P2 没有用到（Task 14 写进交接），用到它们的 Phase 或 M4 的 review 再写；设计 13.2 指向 M5–M8 的 `handoffs/M4-closeout.md` 的 4 个链接由收尾写那些文件；空的 `m4-p1b` worktree 由负责人定（第 7 节）。

**交接的处理结果**：P2 在三份交接里写下"处理结果（M4/P2）"：[M1-P3 交接](../handoffs/M1-P3-trim-platform.md)（地址，按编号查找的一半）、[M2 收尾交接](../handoffs/M2-closeout.md)（第 3 节 P2 的写的 `updated_at`；第 4 节的路径的排序、数的范围、`date`、代价测试，map 型和递归的结构没有用到）、[M3 收尾交接](../handoffs/M3-closeout.md)（第 2 节约定五、第 3 节的规则和矩阵、第 4 节的 `expectIssueDeleted` 和 `guest_view_all_features` 的说明）（Task 14 `8c43625f`，约定五的一项 `df7380e4`，按编号取的 403 和各交接的未完项 `fd546f35`）。三份的 `status` 都照旧是 `open`，在 M4 收尾随 review 关闭（P1a、P1b 的做法）；各自仍未处理的条目写在它们的"处理结果（M4/P2）"末尾的"仍未处理"一行：M1-P3 交接 :58、M2 收尾交接 :126、M3 收尾交接 :146，与 spec 第 7 节的表（N1 之后）一致。其中不在那张表里的：M1-P3 交接的动态的字段名和评论的 `access`（P4，上面 P4 一行）、富文本筛选的显示层（P10b，关键词守卫的终态在 P16）、收集箱的来源（以转交关闭：收尾写进 M7 的交接，删除工作项动态中显示来源的一支在 P15）；M2 收尾交接第 1、2、5–13 节和 M3 收尾交接第 1、5–9 节，按 M4 设计 13.1 的落点。M4 设计 13.1 落在 P2 的（M1-P3 的地址一项的一半，M2 交接第 3 节 P2 的用例和第 4 节，M3 交接第 2 节 P2 的批量写、第 3 节和第 4 节的两项）都在其中（spec 第 7 节）。

### P8接收回执（2026-10-27）

T10–T12搜索消费issue.search、ParseIdentifier/MatchIdentifier及共享Unicode折叠fixture，在同一可见项目/OwnOnly快照内匹配，guest/adminGuest不放宽。六草稿操作统一draft.write，发布另按issue.create并复用原引用/访客边界。T12真实HTTP/矩阵/契约与T14 I18/I14已有证据；P9物理删除账户组合与P13–P15页面仍开放。 实际命令/计数/变异/检查点见[P8 review §8](P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)。产品HEAD `10f44790`、首次整审 `166b9efd..10f44790`；文档独评/最终准确HEAD CI按后续收据追踪。未合并，不改变本页历史验收与集成事实，输入handoff总体仍open。

## 7. 已知限制

- **访客取别人的工作项的端到端格子不在 P2**（spec 第 7 节）：本 Phase 由矩阵（Go）核对，端到端的那一格随 I20 的接口版本在 P5。
- **删除的语句不分批**（spec 第 7 节）：一个工作项的动态随它的修改增长；目标规模上一条语句是几十行；极端的工作项（几万行动态）由 `lock_timeout` 之外的请求期限兜底。
- **删除、归档、恢复不发布事件**：P6 加（设计 3.19 的用例的发布点）。
- **按编号取的时间差和代价**（spec 第 6 节）：每种 404 的回答逐字节相同，但看不到的比不存在的多一次判定的读（与 M3 看不到的项目相同），不处理；每次读工作区的全部项目的标识（v0 的规模下是几十个），需要时 P8 的搜索一起加索引。
- **创建者不明的工作项只在单元一层**（T2.3、T2.9，到 P9）；其余按性质只在单元、存储一层的性质见 spec 附录 A.3，HEAD 上 24 个、7 个（第 5 节）。
- **持续集成没有跑过**：原型、复现和执行都在 macOS 上（spec 第 6 节最后一行）；ubuntu 上的结果在合并时看。
- **空的 `m4-p1b` worktree**：客户端的重启重新建了 `.claude/worktrees/m4-p1b`，在 `7db78788`、分支 `worktree-m4-p1b`，没有自己的提交（`git log 7db78788..worktree-m4-p1b` 没有输出）；不是本 Phase 的，删不删由负责人定。
- **注明日期的记录里过时的句子**：spec 附录 A（A.1、A.3、A.5、A.6，清扫 9、18、21、35、52）和 Global Constraints 的大小，照裁定不改，第 5 节逐个数出；P1b 的 spec（:137、:427）、plan（:4161）仍把裁剪过的测试写作 P1b 的 T5.16 的组合一层（HEAD 上是 T12.3 的路由的测试）；架构的那份 INFRA 日志 `p2tools/mut-logs-c/T5.1-composition-1.log` 原样留着（第 1 节）。
- **差异清单的"恢复没归档的 404"**（:241）靠预检读的 Plane（spec A.2），没有 `plane/` 无法再核对。
- **接受、不改的**：`serializeOn` 不能指名在等之前就结束的写（T1.4、T2.4 在交错 4、5 死于前提的期限，矩阵给出有名的 403）；`given[T]` 与 `named` 各在自己的适配器；500 代替 422 的发现（T9.4、T10.1、T10.3、T10.4、T10.6 的组合一层和 T10.3 的 I2 死于数据库的 `issues_dates_check` 答的 500：那个检查是兜底，发现是断言）；N2（`ErrIssueNotFound` 的说明不点名没有操作走得到的"锁下在别的项目里"）；Task 7 1,630 行，长过约 1,500 行；`issue/app/fakes_test.go` 400 行；`permission_matrix_test.go` 406 行（Global Constraints 点名的例外）；设计 9.1（:1679）、9.3（:1744）说两个折叠相同的标识，对它们的数据成立。

### M4/P6 接收记录

子树删除、手动归档/恢复事件已按P6规格实施，提交、真实测试/变异与独立评审见 [P6 review §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)及[§5](../reviews/P6-issue-events-review.md#5-最终实现与-plan-的差异)。P7/P9/P15和M4整体交接仍按原文开放。
