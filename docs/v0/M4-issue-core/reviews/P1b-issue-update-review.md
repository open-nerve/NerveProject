# M4/P1b 工作项的取、改与写闸门：评审记录

| 项 | 内容 |
|---|---|
| Phase | M4/P1b `issue-update`（设计中的 P1 由控制者按决策点 2 的预先批准拆成 P1a、P1b，2026-10-17；P1a 已合并，`5ce30a1a`） |
| 日期 | 2026-10-19 |
| 结论 | **通过**：15 个 Task 逐个实现、逐个评审，都通过（T1、T3、T6、T8、T11 的评审各有一条 Important，T5 的评审有两条，都出在 plan 写定的代码或测试里，都在当轮改掉）；整分支评审"一个修复轮之后可以合并"，Critical 0、Important 0、Minor 5，另有 Nit 4；修复轮一轮（两个提交）落实 M1–M5、N1、N2 和台账留给评审记录的各项；范围复审：Critical 0、Important 0、Minor 3、Nit 3，另有可选的两条，"本提交带上三条 Minor 的修改之后可以合并"；三条 Minor、三条 Nit 和可选的第一条在本提交改正（第 4 节） |
| spec / plan | [spec](../specs/P1b-issue-update.md) / [plan](../plans/P1b-issue-update.md) |
| 设计 | [M4 设计](../M4-design.md) 第 12 节 P1b；P1 拆成 P1a、P1b 见 `d693f539` |
| 分支 | `worktree-m4-p1b`，从 `main` 的 `5ce30a1a`（P1a 的合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`a04d479d`，`$M4TMP/p1b-architect-report.md`）：在 `$M4TMP/p1bproto` 上逐 Task 做出原型、存快照（`p1bsnap/T0`…`T15`），plan 的块由相邻两份快照的差异生成。设计草稿的 12 个任务切成 15 个 Task（P1 拆分的裁定 3），没有超出约 16 个、每个约 1,500 行。plan 8,769 行（`wc -l`），spec 420 行。从基点逐 Task 重放 267 个块，照原样跑 84 个 `Run:`（1,823 秒，没有重跑），`treediff` 与原型的 3,372 个文件没有差异；100 个变异 99 个被发现，T8.11 是等价的。控制者的裁定（`$M4TMP/p1b-architect-rulings.md`）：spec 第 3 节 1–16 条照建议；第 13 条的计时在执行之前加固（第 3 节）。
- **执行前的预检**（opus，只读，`$M4TMP/p1b-preflight/preflight.md`）：High 0、Medium 4、Low 6，外观 1。闸门的证明对，把闸门的位置算进等待图也没有环。独立重放 267 个块（1,767 秒），`treediff` 与原型的 3,372 个文件没有差异；`raceall` 的 45 个测试 225 次顶层、930 个子测试，没有 40P01；矩阵 1,055 格；I1、I2 各通过 8 次。它自己的 35 个变异：PF.G12 是坏的变异，不计；34 个计入，31 个被发现，留下的 PF.G4、PF.G5、PF.C3 成为 M1、M2、L5。
- **修订**（`f094ced6`）：M1–M4、L1–L6、c1 和裁定 10 的 P3 一句照预检接受（`$M4TMP/p1b-preflight-rulings.md`），由架构子任务改 spec、plan 和设计 11.11、13.2。之后是 273 个块，15 个 Task 带全部 84 个 `Run:` 再完整重放一次（1,595 秒），快照随之重做；104 个变异 103 个被发现（新的行 T5.21、T5.22、T10.17、T10.18）。控制者另提交 `e5df62f6`：设计 3.5、8.8 在 K 旁写上同时热的项目数（预检的 M4）。
- **执行**：
  - 15 个 Task 各由一个实现子任务完成，用 `planapply.mjs` 应用 plan 的块；各由一个评审子任务检查是否符合 spec 和质量；修复轮之后有范围复审。同一时刻只有一个写入者。台账记下的评审模型只有：Task 1–3 的评审 opus，Task 1 的复审 sonnet，Task 13 的评审 sonnet。
  - 实现者在每个 Task 里按 plan 的变异表再跑变异，在 `$M4TMP/p1bsdd/` 里没有 git 的副本上跑（`mut.py`；单元、存储、组合、端到端即 I2、守卫各层）。标"（Task n 起）"的行由 Task n 的实现者在自己的树上重跑（`MUT_LATER`）。端到端的发现由 `e2ekills.py` 核对是断言失败。容器起不来不算发现，出过两次：Task 4 的矩阵第一次起不来容器（63.5 秒），Docker 空闲时重跑 9.0 秒 `ok`；Task 5 的 PF5.7 的组合一层一次 417 个 `FAIL` 都是容器，Docker 空闲时重跑，留下（`task-4-report.md`、`task-5-report.md`）。
  - **plan 是"执行时的 plan"**（Task 4 的裁定，`e7714025`）：每个修复轮也改它的 Task 的块，从基点应用 plan 得到 HEAD 的每个手写文件；块数随修复轮从 273 到 308（第 5 节）。
  - Task 自己的测试缺口和说错的注释，由实现者在提交之后提出，或由 Task 的评审发现，控制者逐项裁定（第 3 节）。补上缺口的都先由变异证明缺口存在，补上之后再证明测试咬得住；执行中加的变异是 plan 的变异表和 spec 附录 A.6 的行（5.1）。
  - **执行中改了生产代码的行为的只有一处**：Task 9 的修复轮（`06bd96b4`），配置的规则从"锁的期限短于请求的期限"改成"`max_wait` 加锁的期限短于请求的期限"。另有两处只换写法、不改行为，几处只改注释（第 5 节）。其余都是测试、测试的数据和文档。
  - **规则的失误**：44 次，都是只读的，都没有改动什么。
    - 实现者 5 次：Task 7 评审之前的修改（`784443f8` 那一轮）在一个调用里连了两个只读的 git 命令；Task 10 第一次的 `git -C <worktree> status`；Task 10 的修复轮（`745a5233`）用一个调用连了 `git status --short` 和 `git log --oneline -4`；Task 11 一次 `git -C <worktree> status`；Task 12 用 `;` 连了一个 `ls` 和 `git log --oneline -3`。
    - 评审者 37 次：Task 10 的复审一次 `cd`；Task 13 的评审一次 `cd`；Task 14 的复审三次 `cd`；Task 15 的评审 25 次 `cd` 和一次连起来的 git 命令；整分支评审一次 `git -C … rev-parse`、一次 `cd … && grep`；范围复审两次失败的 `cd /dev/null`、两次对它自己的克隆用的 `git --git-dir`。
    - 控制者 1 次（2026-10-19）：一个调用用 `;` 连了两个只读的 git 命令（台账）。
    - 本文的作者 1 次：一个调用用 `;` 连了一个 `grep` 和只读的 `git show --stat`。
- **整分支评审**（opus，`5ce30a1a..3693abc7`，43 个提交，`$M4TMP/p1bfinal/review.md`）：结论"一个修复轮之后可以合并"，Critical 0、Important 0、Minor 5、Nit 4。它在 `git archive` 的副本上重跑全部门禁；`bootstrap` 的 48 个测试（交错、竞争、取锁形状、忙的项目、池、事务的连接和两个换码的测试）`-race -count=5`，其中 4 个另在 `GOMAXPROCS=2` 下各 3 次；plan 的重放（308 个块）；29 个变异、59 层。它把闸门的位置当作资源重推了死锁，没有环；逐条判断执行中搁置的各项（第 4 节）。
- **修复**：一轮，两个提交（`203f1f94`：矩阵的一行、三处注释和它们的 plan 块；`f18de99d`：spec、plan、设计、README、总体设计、M3 交接和 P1a spec 的标记）。范围复审一次（opus，`$M4TMP/p1bfinal/rereview.md`）：13 条裁定都照裁定落实，第 14 条遵守；Minor 3、Nit 3、可选 2 条；本提交改正（第 4 节）。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | I2 的接口版本通过；`make e2e` 122 个：此前的 121 个，加 I2 | **通过。** T14 `f90fa155`：`make e2e` 122 个中 120 个通过，失败的两个是 A4 的 firefox 测试；I2 单独跑 30/30；`MUT_LATER=14` 的 35 个变异都在 I2 被发现，`e2ekills.py` 0 个可疑。`3dc77e26`、`d4c99f3f` 之后同样 120/122，I1、I2 各重复 5 次、3 次通过。整分支评审在 `3693abc7` 的副本上 119/122，多出的失败是 S3（副本不是 git 检出）。本文的作者在 HEAD `f18de99d`（本提交之前，树干净）上：`make e2e` "Running 122 tests"，120 个通过，I1（`i1-create-issue.spec.ts:23`）、I2（`i2-update-issue.spec.ts:27`）、S3 在其中；失败的两个是 A4 的 firefox 测试（`stories/identity/a4-refresh.spec.ts:85:3`），错误是 `Error: browserType.launch: Executable doesn't exist at /Users/xiaoruan/Library/Caches/ms-playwright/firefox-1543/firefox/Nightly.app/Contents/MacOS/firefox`（机器上没有 Firefox；`$M4TMP/p1bsdd/rd/gate-e2e.log`） |
| 2 | `getIssue`、`updateIssue` 的矩阵格子通过 | **通过。** T4 `f81de68b`：`getIssue` 15 格、`createIssue` 75 格。T6 `f536f7e4`：1,055 格（`updateIssue` 九行 99 格）。M1 之后（`203f1f94`）已归档一行发 `{}`：`go -C server test -count=1 -run '^TestPermissionMatrix$' -v ./internal/bootstrap/` 1,056 个 PASS（顶层、`prepare` 和 1,054 格），0 个 FAIL；范围复审在 `f18de99d` 上同样，三个已归档的 `updateIssue` 格子通过。T5.29（PT5.7）在矩阵上被发现：`archived_project_admin` 答 200，期望 409 `project.archived`（`permission_matrix_test.go:383`）。本文的作者的 `make test` 里 `bootstrap` `ok`（80.0 秒） |
| 3 | 交错 1u、3–7 的修改一支、等锁期间工作项不在了的修改和交错 20 `-count=5 -race` 通过，没有 40P01；P1a、M3 的交错照旧通过 | **通过。** T11 在 `3cfef5a9` 的树上：十个测试 `-race -count=5`，50 次顶层、170 个子测试；`c46e8363` 之后 3u 再跑（5 次顶层、60 个子测试），`1dea6d99` 之后改过的三个再跑（15 次顶层、10 个子测试），都通过。T12 `beb66942`：交错 20 的两个测试带 `-race` 5 次、不带 10 次、`GOMAXPROCS=2 -race` 3 次都通过。整分支评审在 `3693abc7` 的副本上：`bootstrap` 的 48 个测试（`raceall.sh` 的 45 个：M3 的 30 个、P1a 的 9 个、P1b 的 6 个；加事务的连接和两个换码的测试）各 `-race -count=5`，290 秒，240 次顶层、930 个子测试，没有 40P01、死锁、DATA RACE、FAIL、panic。之后的修复轮在 Go 里只改了矩阵的一行（`TestPermissionMatrix` 不在这 48 个里）和三处注释（`git show 203f1f94`），所以这次运行对 HEAD 仍成立 |
| 4 | 闸门满时、等到 `max_wait` 时、请求在闸门上到期时 503 带 `Retry-After: 1`，`lock_timeout` 到时 503 | **通过。** T10 的三个提交（`0c5cd505`、`e25e2b5e`、`745a5233`）之后各跑一次：`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy` `-race -count=5` 10/10。整分支评审：它们在 48 个的 `-race -count=5` 里，另与交错 20 的两个在 `GOMAXPROCS=2 -race -count=3` 下 12 次通过；T10.5、T10.16、T8.7、T10.19（PT10.1）、T9.3 在那里被发现。范围复审在 `f18de99d` 上跑了这两个测试，通过；T10.21（排满答成 500）在单元一层、`TestAProjectTooBusyForAWriteAnswersServerBusy` 和交错 20 被发现（范围复审的抽跑） |
| 5 | 规模目标上的删除耗时写进本 spec | **通过。** spec 附录 A.4（:327–334）：架构子任务量了四次，种下 300 万行 38.6–44.3 秒；删除 Web（一个项目）一侧 4.83–6.32 秒、请求 4.40–4.71 秒；删除 beta（五个项目的工作区）一侧 25.1–29.7 秒、请求 23.0–24.6 秒。执行中的运行（A.4 :336）：Task 13 第一次请求 4.95 秒、24.99 秒，修复轮之后 Docker 空闲时 5.08 秒、27.45 秒，都在 ±2–3 倍之内。结论写在第 7 节：一个这样的项目在 15 秒之内，五个的工作区不在 |
| 6 | `make gen-check`、`lint-go`、`test`、`lint-web`、`knip`、`test-web`、`e2e` 通过 | **通过。** 本文的作者在 HEAD `f18de99d`（本提交之前，树干净）上逐个运行（`$M4TMP/p1bsdd/rd/gates.sh`，日志 `gate-*.log`）：`make gen-check` exit 0；`make lint-go` 两段 `0 issues.`；`make test` exit 0，47 个 `ok`、0 个 `FAIL`、没有缓存，`bootstrap` 80.0 秒；`make lint-web` exit 0（"keywords: 72 rules, 23 exceptions, no hits."；i18n-keys 1,297 个键；turbo 54/54）；`make knip` exit 0；`make test-web` 17/17；`make e2e` 见第 1 行。`server/go.mod`、`go.sum`、`server/tools`、`pnpm-lock.yaml` 自 `5ce30a1a` 没有变化（`git diff --stat` 没有输出）。本提交的文档改完之后（提交之前，`$M4TMP/p1bsdd/rd/post.sh`）：`TURBO_FORCE=1 make lint-web` exit 0（不用缓存，turbo 54/54、0 个缓存；关键词守卫没有命中）；`TURBO_FORCE=1 make test-web` 17/17、0 个缓存；`make gen-check` exit 0；`doclinks.py` 在 README、总体设计、差异清单、两份交接、P1b 的 spec 和 plan、P1a 的 spec 和本文这 9 个文件上 0 个断的链接，M4 设计单独查，断的只有 13.2 指向 M5–M8 的 `M4-closeout.md` 的 4 个（第 6 节） |

持续集成：分支在执行中没有推送（`git branch -r` 只有 `origin/HEAD`、`origin/main`）。推送之后和合并之后的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，写明理由和裁定错了的代价。完整的记录在执行台账（`.superpowers/sdd/P1b-issue-update/progress.md`）中。

**执行前**
- **架构的报告**（`$M4TMP/p1b-architect-rulings.md`）：spec 第 3 节 1–16 条照建议，理由在各条里；没有一条重开负责人的决定，都在设计或拆分裁定之内。第 13 条（交错 20 的读的上限 500 毫秒）照建议，但依赖计时的计数要在执行之前改成按条件，不留持续集成的退路：交错 20 的"恰好 134 个"和"66 个"、两个忙的项目的测试的窗口、锁的期限的测试的 80 毫秒、链的 300 毫秒的开始，在较慢的机器上会假失败或让变异留下。预检据此提出 H1–H4，修订照做。错了的代价：几个测试多等到条件成立（有失败的期限）。
- 架构子任务的四个本地决定一并接受：`platform/config/load_test.go` 414 行（基点 403 行）；`doclinks.py` 跳过代码，Task 15 不查 M4 设计的链接（设计里 M5–M8 的收尾交接在 M4 收尾才有）；Task 8 不跑 `make test`（`platform/gate` 到 Task 10 才有使用者）；已归档项目里的修改在 P1b 就答 409 `project.archived`（同 P1a 的创建）。错了的代价：一个测试文件多出 14 行；工具少查一个文件；Task 8 少一次全量的测试；一个拒绝提前到 P1b。
- **预检**（`$M4TMP/p1b-preflight-rulings.md`）全部照建议接受，一个修订提交改入：
  - M1：离开闸门在事务结束之后，没有一层钉住（PF.G4 留下）。Task 10 的 `fakeTx` 在事务里设 `callLog.inTx`，假闸门把事务里的离开记作 `Leave <项目> in tx`；T10.17 只在单元一层，A.3 写明理由（提交的时刻与闸门的交接经 HTTP 不靠计时看不到）。
  - M2：读不进写闸门，没有一层钉住（PF.G5 留下）。`newGet` 用记下进出的闸门，交错 20 也读热的工作项；T10.18 在单元和（Task 12 起）组合一层。
  - M3：H1–H4 按条件计时，删去 spec 里持续集成的退路。
  - M4：作为文档的事实修正，不是升级。闸门是设计 11.11 已批准的每个项目的份额；总的情形要长时间持行锁，今天只有过渡的同步连带会这样（决策点 7 的 A，到 P9）；它让别的请求慢几秒，不丢数据、不死锁。README、总体设计 4.2、M3 交接、spec 第 5、7 节和设计 11.11、13.2 写明 ⌈`max_conns` / K⌉ = 5 个同时热的项目能占满池，spec 第 5 节 P9 一行加再量的交接。
  - L1 负载测试每个 goroutine 1,000 次；L2"2–49 毫秒"；L3 端口的说法；L4 `issue/module.go` 的说明（Task 6）；L5 已取消组的状态（Task 5 的数据，T5.21）；L6 盖戳的数据（Task 7，T5.22 的组合一层；PF.C1 是数据的规则，只在单元一层，A.6 的完成时间一行写明）；c1 折行（生成物的摘要不变）；裁定 10 的 P3 一行加"进第二个闸门时不持连接、不持锁，只持着前一个闸门的位置"。
  - 错了的代价：M1、M2、L5、L6 是测试和数据的几十行；M3 让四个测试按条件等；其余是文字。
- spec 的 12 个"待裁定"标记（和 P1a 的 6 个）在裁定之后就过时：留到评审记录的阶段改（实际在修复轮 `f18de99d` 改；P1a spec 的状态一行在本提交）。代价：一个过时的标签。
- 变异在 `$M4TMP/p1bsdd/` 里没有 git 的副本上跑（同 P1a）：保留架构子任务的证据，不让 git 进副本；错了只是工具的路径。

**执行中**（补上缺口的，都先由变异证明缺口存在，补上之后再证明测试咬得住）
- **Task 1（评审 Important 1 条）**：
  - 实现者指出 `TestChangesOf` 只有负责人一侧有"给出空集合就清空"的一行：标签一半（PT1.1：给出的空标签集合当作没给）在单元一层留下，到 Task 7 才被发现。评审之前补上（`7644759f`，"every label removed"一行）。理由：一条规则的两半都要在单元一层钉住；代价一行。
  - `CheckIssuePatch`、`Changes`、`None` 的三句说明引用设计 3.7 和 spec 第 3 节第 4 条，是 Task 5 的锚点，照留；错了不影响用户。变异的参数写成不带点的 `TN`，常设的 brief 随之改正。
  - 评审：`UpdateActivity` 加负责人的循环复制了 `CreationActivity` 的第 2 行的（`activity.go:104-107` 对 `:43-46`），测试也重写了期望的行。控制者裁定改（`3866bf13`：一个 `assigneeAdded`）。理由：设计 3.13 的第 2、12 行是同样的列，两份写法会悄悄分开；代价约 10 行。复审（sonnet）：已落实。
- **Task 2**：评审通过。标签目录的 `ORDER BY id` 不钉住（PT2.4 留下）：没有读者依赖这个次序（`issue` 的端口不承诺次序，创建和 `labelNames` 按 `id` 找标签），它让两个目录的查询一样、输出确定；错了没有代价。spec A.6 清扫 16 的两半（假实现按请求的次序回答，从 Task 2 起不对）留给修复轮，`f18de99d` 改成"假实现按目录的次序、存储按 `id`，端口不承诺次序"。T2.1–T2.3 在较晚的 Task 上的层（Task 7 的盖戳测试、Task 14 的 I2）由 `MUT_LATER` 确认；T2.2、T2.3 的盖戳一层由修复轮记进表。
- **Task 3（评审 Important 1 条）**：
  - 实现者的三条：说明里提到 Task 5–6 的 `updateIssue`（同 Task 1 的说明，照留）；`unlinked` 与 `linked` 对称、删除的行的内联循环（交给评审判断：不是缺陷）。
  - 评审：`update_test.go` 每次去掉只点一个目标，"只去第一个"的变异留下（标签在每一层，负责人到 Task 6）。裁定一次去掉两个（加 dave、Docs，`057cc421`），两个缩小的探针 PT3.6、PT3.7 在存储一层被发现，成为 T3.12、T3.13。理由：去掉的标签留在工作项上、动态却说去掉了，是不出声的数据错误；代价是测试的两行数据。
- **Task 4**：
  - 实现者指出 `newGet` 的工作项由调用者自己建，调用者、创建者、最后修改者分不出（PT4.1 到 Task 6 的矩阵才被发现，PT4.2 只在矩阵）；另有五句说明比代码宽。评审之前补上（`b3d45197`：carol 建、alice 读；五句改准；Task 6、10 的块随之改锚点），PT4.1、PT4.2 在单元一层被发现。
  - 判定的动作（`issue.read` 还是 `issue.create`）到 P2 的访客规则之前只在单元一层分得出（PT4.3）：A.3 加一条、Task 4 的表加一行（修复轮做成 T4.7）。已归档的工作项照样读（PT4.4）由 Task 7 起的 `TestAWorkItemReadsAsStored` 钉住，接受。
  - **"执行时的 plan"**：后面的 Task 的块若改到修复过的文字，从基点重放就要求前面的 Task 的块也写修复过的文字。所以 Task 1、3、4 的块在一个文档提交里改成提交的代码（`e7714025`，281 个块），从基点应用到 Task 4 再用 `treediff` 与 worktree 比较；此后每个修复轮同时改它的 Task 的块（常设的 brief 随之改）。理由：plan 是能复现的记录。错了的代价：每个修复轮多几分钟。
  - 评审通过。留给修复轮：spec 2.6、A.6 清扫 10 和 35、plan Task 4 的说法（种子由工作区的管理员写、指派 PM 和离开的成员，`getIssue` 有两行）；"读不取锁"的"（Task 12 起）"一行（成为 T4.8、T4.9）。
- **Task 5（评审 Important 2 条）**：
  - 实现者的检查过的提议，评审之前补上（`d6c22d7e`）：一次加两个账户、已归档的什么都不改（409）、锁下重读和判定的失败、不是第一个的被拒绝的标签。每一个都是哪一层都看不到的数据或状态码的缺口（PT5.1、PT5.3–PT5.7 在单元一层被发现）。测试的近似副本（拒绝的核对、逐个端口的失败）收成共用的 `wantRefusal`、`wantStoppedAt`；`lock.go` 预检的说明改成对每个写都对；遮住外层的局部 `named` 改名 `accounts`。PT5.2、PT5.8 成为"（Task 6 起）"的行（T5.24、T5.30）。
  - 评审：(1) 没有单元测试在改标签时留着一个标签（RV5.1：删掉全部再插回，到 I2 才被发现）；(2) 设计 3.7 的状态规则写了两份（修改的 `stateOf && !triage` 与创建的 `pickState`）。裁定改（`9076e2be`）：改标签的一行旁留着 Docs；`pickState` 移到 `writes.go`，两个写都用它（`c.StateID` 不为空时两者相同，行为不变）；T5.5、T5.6 在 `p1btools/mutants_b5.py` 里重新锚定（架构子任务的工具一次性的修改，意图不变，原样留在 `p1bsdd/mutants_b5.before-t5-fix2.py`）。理由：留着的行被修改者重新盖戳会丢掉谁加的它；一条规则两份写法会分开。错了的代价：一行测试数据和一个函数的位置。复审：已落实。
- **Task 6（评审 Important 1 条）**：
  - T1.15、T5.2 从 Task 6 起也在矩阵被发现，记成"（Task 6 起）"的层（修复轮做了）。一个连接的测试的"每个写"一句由 Task 7 重写（锚点），分支之内接受。`toIssue` 与 `withIssueBody` 等交给评审：不是重复。
  - 评审：`issue_writes_test.go:43-48` 内联了 `newWorkItem`（会一直留到 Task 15）；取锁形状的每行的目标是随意的字符串，发的路径没有绑到它的操作的契约路径上。裁定改（`fe4497e0`）：用 `w.newWorkItem`；每行发它的操作的路径（`pathOf`），PT6.F2a、PT6.F2b 因此被发现（T6.4）。理由：P2 会加同一个方法的行（归档、恢复），会溜过去；两处都是共用的测试帮手，便宜。
- **Task 7**：
  - 实现者指出回答如存的测试的工作项从不完成，两边的 `completed_at` 都是空（PT7.1 在每一层都留下）。评审之前补上（`784443f8`）：bob 的修改把它移进 Done；盖戳的说明改写；五张表的行的查询收成 `issueWorld.workItemStamps`；创建的盖戳测试也核对每行是谁的（PC7.5 在旧的测试上留下）。
  - 评审通过；Minor：plan 的说法；数据里 `completed_at` 等于 `updated_at`，`getIssue` 把两者对调会在组合一层通过。裁定当轮改（`6e851157`：两个 PATCH，四个不同的时刻依次），PT7.4（把完成的时刻答成最后修改的）被发现。理由：说明对数据不成立；代价几行。PT7.1、PT7.4 成为 T7.5、T7.6。
  - 实现者提示盖戳测试的工作项也可以由 `newWorkItemIn` 建，带给 Task 11（`e52d98d8`）。
- **Task 8（评审 Important 1 条）**：
  - 实现者指出 PT8.1（放弃时正好被交到位置的等待者把位置空出来，不交给下一个：等待者排在空位之后，空闲的项目上出假的 503）在每次运行都留下。评审之前加测试里的不变式核对（`ce314a94`：有人等时没有空位，在闸门的锁下核对；spec 第 3 节第 11 条），PT8.1 带和不带 `-race` 10/10 被发现。理由：闸门的活性必须钉住。`Limits` 的说明从 Task 9 起才对，分支之内接受。
  - 评审：等待者放弃之后的先来先得没有测试（RV8.1，放弃的人由队尾顶替，留下）；Minor：该进入的调用者没有核对（RV8.2 只因 nil 的 panic 失败）、两个"等着的时候"的情形可能根本没在等、说明的措辞。四条都改（`a50c9c5b`：`held(t)`；放弃的情形先确认已排队；说明），PT8.1、RV8.1、RV8.2 10/10 被发现，成为 T8.12–T8.14。理由：先来先得是闸门在过载时的承诺；panic 不算发现；可能走不到自己的路的情形是不稳的钉子。错了的代价：测试的几十行。复审的 Nit（`gate_test.go:191-193` 的说明可能读成也说 `MaxWait`）由整分支评审判定：接受、不改。
- **Task 9**：
  - 实现者指出日志测试的值与别的键共用或是别的键的前缀（PT9.3–PT9.5）、两个下限没有测试接受（PT9.1、PT9.2）、向上取整与取最近分不出（PT9.6）。评审之前照提议补上（`c8c76dff`，只改测试，不让 414 行的 `load_test.go` 再长），成为 T9.11–T9.16。配置的说明从 Task 10 起才对，分支之内接受。
  - 评审通过；Minor (a)：校验不看 `max_wait`，`max_wait` 加锁的期限不短于请求的期限也能通过，那样的写到期时答 500 而不是 503。裁定当轮改（`06bd96b4`）：规则改成 `max_wait + write_lock_timeout < request_timeout`。理由：设计的等待要在期限之前以 503 结束；只有配置的加载校验，测试的世界不受影响。**不**在 `busyTx` 里把 `DeadlineExceeded` 换成 503：提交的超时也包着 `DeadlineExceeded`，那时的结果不知道，让客户端重试可能写两次。(b) `tx.go` 的说明（`d` 必须为正；`LockNotAvailable` 是任何 55P03）一并改。(c) P5 的交接（`BeginQuery` 盖过 `IsoLevel`、`AccessMode`）写进 spec 第 5 节（修复轮）。(d) identity、workspace 的测试里手写的锁的期限和 55P03 的检查交给整分支评审（成为收尾的一条）。错了的代价：一条配置的规则。
  - 实现者的观察：这条规则盖住的是闸门的等待加**一次**等锁（`lock_timeout` 是每次等待的），写进第 7 节。T9.6 在 `p1btools/mutants_b8_10.py` 里重新锚定（同 T5.5、T5.6）。PT9.7（旧的规则）在单元一层被发现，那时没有排进表（本提交的 T9.17）。复审：已落实；两个 Nit 没有改（第 7 节）。
- **Task 10**：
  - 实现者指出没有测试钉住闸门按项目取键（PT10.1：整个进程一个闸门，在 Task 10 和 Task 15 的 `bootstrap` 上都留下），`recover()` 的值没有核对（PT10.2），两句说明不准。评审之前补上（`e25e2b5e`：单元和组合两层，Web 被持着时 Ops 的创建答 201），PT10.1 在单元和组合、PT10.2 在单元一层被发现，成为 T10.19、T10.20。理由：按项目是闸门的全部意义。
  - 评审通过；Minor：请求到期的测试里，alice 离她自己的期限只剩 0.5 秒减去她的路程；两个 `Execute` 的说明没有提闸门和它的 503；`ErrFull` 换成 `server_busy` 没有变异的一行。前两条当轮改（`745a5233`：锁的期限 2 秒、偏移 2 秒，余量约 1 秒；bob 的 [3 秒, 4 秒) 窗口和变异的 ≥ 4 秒照旧）。理由：裁定 13 的意图，持续集成更慢。第三条成为 T10.21（修复轮）。
- **Task 11（评审 Important 1 条）**：
  - 带上 Task 7 的提示（`e52d98d8`）；`serializeOn` 的说明不再说第一个写等过了它的写（`316d8dff`）。
  - 实现者指出 3u 的数据分不出删除的写者（alice 既写又删，PT11.1 在 3u 留下）。评审之前改由 bob 写（`c46e8363`），成为 T11.1。spec 2.1、2.13 的名字留给修复轮；`workItemRows` 的宽度为 Task 13 服务，接受。
  - 评审：活动的读取逐字复制（`issue_interleaving_change_test.go:42-53` 对 `issue_updates_test.go:67-79`）；Minor："工作项不在了"的测试的 `WaitForLockWaitBehind` 也能匹配在 Fix 的行上的等待，锁工作项的预检会不经锁下重读就通过。裁定改（`1dea6d99`）：一个 `issueWorld.activityBy`；改用 `WaitForLockWaitOn("projects")`，PT11.2（预检锁工作项）因此失败（与 T6.3 是同一个改法，记成 T6.3 的 Task 11 一层）。"各两种顺序"的措辞留给修复轮（清扫 12）。
- **Task 12**：`ed45745e` 的说明照留；折行、窄的测试名和 `replyTo` 对 `sendInBackground` 交给评审：前者可留，后者不同。PT12.2、PT12.3（`GetIssue`、`see` 取项目行的锁）在交错 20 被发现，成为 Task 4 评审要的"（Task 12 起）"的读的一行（T4.8、T4.9）；没有闸门的变异的读最慢 3.19 秒。评审通过；第 3 节第 13 条的措辞留给修复轮（134 由构造成立；66 的真正的界是 5 秒的期限，强制持到 2.9 秒时最慢的一个 3.13 秒答；至少一秒的持有让在被持的行上等的读超过上限）。
- **Task 13**：控制者接受实现者的测量作为证据（在 ±2–3 倍之内），评审用 sonnet：这个测试可选、只测量、持续集成跳过。评审：种子的行数没有核对（空的种子也通过）、文档的命令没有 `-count=1`、访问令牌 15 分钟而期限 1 小时、文档的小问题。当轮改（`500e316c`：核对 6 × 50 万行；`-count=1`；令牌 1 小时），PT13.1 在种子的核对上被发现（T13.1）。理由：P9 在这个种子和命令上再量。复审：已落实；`workItemRows` 在 `pgtest.Soon`（5 秒）下数到 240 万行，是这个测量里唯一剩下的慢机器上的期限（交给 P9）。
- **Task 14**：
  - 实现者：I1、I2 重复 `byId` 和期望的行；措辞；"在任何读之前拒绝"只在单元一层看得到（PT14.1）。评审之前改（`3dc77e26`：`byId`、`assigned`、`createdFor` 移到 `assert/issue.ts`；I2 加一个随机 id 上的无效体，期望 422），PT14.1 在 I2 被发现（T5.32）。
  - 评审通过；Minor：清扫 14 的说法（修复轮）；`assigned` 手写了 `changed` 的行的形状，I2 重复两行期望。裁定当轮改（`d4c99f3f`：`changed` 与 `assigned` 并列在 `assert/issue.ts`，`bugRemoved`、`bobUnassigned` 只写一次）。理由：P14 的 I2 页面版本复用这些构造。p6、p7 的局部 `changed` 留给整分支评审（接受、不改）。
- **Task 15**：
  - 评审之前改（`3693abc7`）："请求到期 → 503"只在闸门的队列里成立，README、总体设计 4.2、M3 交接改成"在队列里请求到期"；差异清单的引用改准（另两行同类的一起）；"K 个连接"是这个项目的工作项的写的。理由：文档的事实；代价几个字。
  - 评审通过；Minor 交给修复轮："在闸门上请求到期"才准（到闸门时已经到期的请求也得到 503），"在队列里"更窄；`v0-design.md:246`、`M3-closeout.md:130` 的"几十毫秒"改成"不到 50 毫秒"；`README.md:141` 漏掉恰好 5 个热的项目；可选：M3 交接点名两个池的测试。

**整分支评审之后**
- 一个修复轮，14 条裁定（`final-fix-brief.md`）：M1 矩阵的已归档一行发 `{}`；M2 Task 5 的探针和 RV5.1 成为 T5.23 起的行，A.3 加 PT5.1、PT5.4、PT5.6；M3"P1b 之前…"；M4 81 个文件（24、57）；M5 设计的 `WriteGate` 在 P1b 一次一个项目、P3 加多个；N1 `gate.go` 的说明写"在闸门上"；N2 `queued` 自己的注释；N3、N4 不改；"在闸门上请求到期"处处一样；Task 15 评审的 Minor，M3 交接点名两个池的测试；台账留给评审记录的每一项；分诊进 spec 第 5 节（P5 的 `BeginQuery`、P9 的 `workItemRows`、收尾的 55P03 和 M8 的等待）；设计第 15 节和本文留给下一个提交。理由：评审的发现，和"每句话对 HEAD 都对"。错了的代价：矩阵一行的请求体和文档的文字。
- 范围复审之后的裁定（`review-doc-brief.md`，本提交）：Minor 1–3、Nit a–c、可选 1 照改；可选 2（PX9.J 成为 T9.18）不改，A.6 和清扫 45 已写明它是只为核对而跑的变异；P1b spec 的状态一行改成"已完成"。错了的代价：变异表少一行。
- 控制者的观察（台账 2026-10-19）：`origin/main` 仍是 `5ce30a1a`，合并是基点上的 `--no-ff`。
- 本文的作者在同一个提交里另改两处 spec 的文字，与 Nit a 同类：2.1（:33）把 P1a spec 的修改记到"整分支评审的修复轮和评审记录的提交"（状态一行是本提交改的）；状态一行（:7）把设计 3.5、8.8 的一句记到 `e5df62f6`。错了的代价是几个字。

## 4. 整分支评审的发现与处理

结论"一个修复轮之后可以合并"：Critical 0、Important 0、Minor 5、Nit 4。评审逐条核对过：
- 取锁的路径照 HEAD 运行的样子：读工作项、预检、闸门、事务、项目行、锁下重读、锁下判定；凡判定拒绝的在预检里答，不进闸门；闸门的离开在提交、回滚之后，panic 也离开；`getIssue` 不进闸门、不取锁；`createIssue` 也在这条路上，没有近似的副本；
- 把闸门的位置、池的连接和数据库的锁都当作资源的死锁推导：闸门只在什么都不持时等，连接的持有者只等数据库的锁，工作项的写只持一个项目的 N、只等 N 的持有者和改地址（改地址什么都不等），所以没有环。它更正了预检的一句：identity 的停用经 `LockActiveMemberProjects` 也取项目的 N，结论不变（停用等写，写不等停用；仓库里没有那句话）；
- 约定五、闸门的实现、各个等待的界和规则、连接池的总代价、模块的边界、组合和契约；
- 门禁、竞争的运行、plan 的重放和 29 个变异（第 1 节）。

| 发现 | 处理 |
|---|---|
| **M1** "已归档 409 在什么都不改之前"只在单元一层钉住：`update_issue.go:74-81` 定下这个次序，而矩阵的已归档一行发 `{"name":"Mend"}`（一个改动），`issue_writes_test.go:145` 也发改动。PT5.7（已归档的检查挪到 `c.None()` 之后）在单元一层被发现，在矩阵、每个 `bootstrap` 测试（75 秒）和 I2 都留下；在它之下，已归档项目里什么都不改的 PATCH 答 200（设计 3.8 是"一切写 409"），两边都不写 | `203f1f94`：矩阵的已归档一行发 `{}`，注释写明已归档的项目在衡量改动之前就被拒绝；spec 2.8、A.6 清扫 11 改成"已归档（什么都不改也 409）"（`f18de99d`）。PT5.7 在一份新的副本上：单元一层由 `TestUpdateIssueRefusesUnderTheLock/archived` 发现，矩阵由 `TestPermissionMatrix/updateIssue,_archived/archived_project_admin` 发现（答 200，期望 409 `project.archived`）；T5.1 照样在单元和矩阵被发现（`final-fix-report.md`）。范围复审在 `f18de99d` 上重跑，相同。它另写的 RR1（只在什么都不改时拒绝已归档的项目）在矩阵留下，在单元一层和一个连接的测试被发现（那里的已归档一步发 `{"name":"Mend"}`、期望 409，`issue_writes_test.go:175`），所以真的改动仍在组合一层钉住 |
| **M2** Task 5 的修复轮的探针和 RV5.1 没有变异表的行，A.3 缺几个只在单元一层的：PT5.1、PT5.4、PT5.6、PT5.7 在每个 `bootstrap` 测试和 I2 都留下，RV5.1 到 I2 才被发现 | `f18de99d`：plan Task 5 的表加 T5.23–T5.32（PT5.1–PT5.8、RV5.1 和路由来的 PT14.1），每行写它的层；A.3 加 T5.23（PT5.1：失败的重读，同 T5.13、T5.14）、T5.26（PT5.4：数据的规则，没有一个组合的请求一次加两个负责人）、T5.28（PT5.6：422 的下标是契约，单元的表有不是第一个的被拒绝的标签）；PT5.7 在 M1 之后不在 A.3 |
| **M3** 两句"今天"被 P1b 变得不对：spec :243"今天它们会一直等到请求的期限"，设计 :2011"（今天它们会一直等到请求期限）"。P1b 起这些写在排满、`max_wait`、在闸门上到期或等锁到期时得到 503 | `f18de99d`：两处改成"P1b 之前它们会一直等到请求的期限" |
| **M4** spec 2.1（:33）和 A.1（:262）写 79 个文件（新建 24、修改 55），HEAD 是 81 个（24、57）：执行中加了 Task 4 的修复轮改的 `permission_matrix_project_test.go`（`b3d45197`）和 Task 11 改的 `interleaving_test.go`（`316d8dff`）；2.1 的 Task 4、Task 11 两行缺它们 | `f18de99d`：两处改成 81 个（24、57）；Task 4 一行写出矩阵的六个文件，Task 11 一行加 `interleaving_test.go`、`issue_updates_test.go`；其余各行按 HEAD 上每个 Task 的文件重写（`ffw/taskfiles.py`） |
| **M5** 设计给写闸门的端口几个项目：3.5（:262）、6.5（:1332）、11.11（:2005）的 `Enter(ctx, projectIDs)`，9.1（:1685）的"多个键按升序一次取齐"。HEAD 的端口一次一个项目（spec 第 3 节第 10 条，P3 加多个），合并之后 main 的设计在 P3 之前不对 | `f18de99d`：3.5、6.5、11.11 加"P1b 一次一个项目（`Enter(ctx, projectID) (leave, error)`），P3 加多个（P1b spec 第 3 节第 10 条）"；9.1 的那一句标"（P3 的；P1b 一次一个键，…）"。spec 第 3 节第 10 条的标记把设计的节写错了（范围复审的 Minor 1），本提交改正 |
| **N1** `bootstrap/gate.go:36-39`"请求在等的时候到期"：到闸门时已经过了期限的请求也得到 `server_busy`（`platform/gate/gate.go:67` 先查 `ctx.Err()`） | `203f1f94`："…or whose request's deadline has passed when it comes to the gate or passes while it waits, is server_busy"；plan Task 10 的块随之（67 行） |
| **N2** `platform/gate/gate_test.go:67-68` 一段注释说了 `waiting` 和 `queued` 两个 | `203f1f94`：`waiting`（:67）、`queued`（:77-78）各一段 |
| **N3** `issue/app/ports.go:113-149`：`IssueCreator`、`IssueUpdater` 各自声明四个插入，说明相同 | 不改：这是设计的按用例的端口 |
| **N4** `platform/config/load_test.go` 414 行 | 不改：在"约 400 行"之内。基点是 403 行（`git cat-file -p 5ce30a1a:server/internal/platform/config/load_test.go` 存成文件再 `wc -l`；评审写的"约 401 行"不准），Task 9 加了 11 行 |

分诊表的其余各行：
- **交给 P5**：pgx 的 `BeginQuery` 盖过 `TxOptions` 的 `IsoLevel`、`AccessMode`（Task 9 评审的 (c)）。spec 第 5 节 P5 一行（`f18de99d`）：工作项一侧的快照事务把隔离级别和锁的期限放在同一个 `BeginQuery` 里。范围复审核对了 pgx v5.11.0 的 `tx.go:47-62`。
- **交给 M4 收尾**：手写的 55P03 检查（Task 9 评审的 (d)）。spec 第 5 节收尾一行列出 14 个测试文件里的 17 处，改用 `postgres.LockNotAvailable`；P2 起新的测试用平台的帮手；测试自己设的 8 处 `SET LOCAL lock_timeout` 不动。评审写的是 13 个文件，而它自己的清单是 14 个；修复轮和范围复审数到 14 个，基点上同样。
- **交给 P9**：规模的测量的 `workItemRows` 每次计数都在 `pgtest.Soon`（5 秒）之下。spec 第 5 节 P9 一行：再量时给计数一个自己的 `context`。
- **接受、不改**：
  - 规则只盖住闸门的等待和一次等锁：文档说的都是"一次"或"每次"等待，都对；默认值给第二次等待留 3 秒（2 + 5 + 5 = 12 < 15）；池里的等待写进设计 13.2 的 M8 一行和 spec 第 5 节收尾一行（第 7 节）。
  - 测试的配置是加载会拒绝的值：有意的，测试的世界不校验（`testIssueConfig` 2 + 4 ≥ 5；请求到期的测试 10 + 2 ≥ 3；池的测试的 `max_wait` 一分钟）。只改文字：`issue_writes_test.go:64-69`、spec 2.12 和第 3 节第 15 条、plan Task 10 的 `testIssueConfig`。
  - p6、p7 的局部 `changed`：M3 的三个相同的帮手（`p6-states.spec.ts:59`、`p6-states-page.spec.ts:53`、`p7-labels.spec.ts:64`），改状态、标签的行，与 `assert/issue.ts` 的动态的 `changed` 无关，没有文件两个都导入；要合并归 M4 收尾。
  - Task 8 复审的 Nit、Task 12 的折行和测试名：外观，HEAD 上读起来对。M3 交接点名两个池的测试：控制者裁定点名，修复轮做了。
- **"文档对照代码"**：评审列出的过时句子（M2–M5 和 spec :128 的一句）、台账留给评审记录的各项和 Task 15 评审的三条 Minor 都在 `f18de99d` 改。

**修复轮**（`203f1f94`：5 个文件，矩阵的一行、`gate.go` 的说明、两处测试注释和 plan 里它们的块；`f18de99d`：spec、plan、设计、README、总体设计、M3 交接和 P1a spec）：
- 检查（`final-fix-report.md`）：矩阵 1,056 个 PASS、0 个 FAIL；`go -C server test -race ./internal/platform/gate/` `ok`；`make lint-go` 两段 `0 issues.`；`make test` 47 个 `ok`，`bootstrap` 78.2 秒；每个提交之后 `make gen-check` 通过，树干净；`make lint-web` 没有命中；`doclinks.py` 在 8 个文件上 0 个断的链接；三次重放（`ffw/replay.sh`）308 个块都对，差的只有生成物、plan、spec、设计和 P1a spec；PT5.7 在新的副本上在矩阵被发现；路由来的探针在 `203f1f94` 上都在它们的行写的层被发现（`ffw/mutants_ffw2.py`–`mutants_ffw7.py`）。
- 与裁定不同的地方，范围复审都接受：PT9.7 没有成为行（控制者的 brief 只列了 PT9.1–PT9.6，是控制者的遗漏，成为范围复审的 Minor 2）；55P03 是 14 个文件的 17 处；PT12.2、PT12.3 是两行（T4.8、T4.9，两个不同的改法）；T6.4 一行含 PT6.F2a、PT6.F2b；PT11.2 记成 T6.3 的 Task 11 一层（两个改法逐字相同）；T11.1 在 Task 11 新的表里。
- 照原样留下的：P1a spec 的状态一行（成为 Minor 3）；设计的 4 个指向 M5–M8 的 `M4-closeout.md` 的链接（M4 收尾写它们，第 6 节）；`TestValidateAcceptsWriteWaitsJustUnderTheRequestTimeout` 只被只为核对而跑的变异弄失败（5.1）。

**修复的范围复审**（opus，`3693abc7..f18de99d`，`$M4TMP/p1bfinal/rereview.md`，副本在 `$M4TMP/p1bfinal/rr/`）：结论"本提交带上三条 Minor 的修改之后可以合并"，Critical 0、Important 0、Minor 3、Nit 3，另有可选的两条。它核对了：13 条裁定照裁定落实，第 14 条遵守；`f18de99d` 的克隆上 `make gen-check`、`lint-go`、`lint-web`、`test`（47 个 `ok`，`bootstrap` 75.2 秒）通过，`doclinks.py` 在 8 个文件上 0 个断的链接；plan 从基点重放，与 `f18de99d` 的归档差 10 处（6 个生成物、设计、P1a spec、P1b 的 spec 和 plan），都允许；PT5.7 在矩阵由真实的断言发现；修复轮写的每个数字它都自己量过（81、24、57；plan 9,756 行和各 Task 的行数；8,885、8,769；308 个块；A.6 的合计；执行中加的 32 行；执行中新加的 Go 测试两个；14 个文件的 17 处；引用的 49 个测试名都在；标记；pgx 的 `BeginQuery`；`workItemRows`；"等到请求的期限的写答 500"；设计的 4 个链接）；抽跑 9 行变异（T5.29、T5.1、T9.16、T6.3 的 Task 11 一层、T4.8、T4.9、T5.32、T8.13、T10.21），都在它们写的层被发现；另跑了 PT9.7 和它自己的 RR1。

**本提交改正的遗留**：

| 项 | 本提交 |
|---|---|
| **Minor 1** spec 第 3 节第 10 条的标记说设计在 3.5、6.7、11.11 写明 P1b 一次一个项目；写明的地方是 3.5、6.5、9.1、11.11，6.7 没有 | spec :175："偏离设计 3.5、6.5、6.7、11.11 的 `Enter(ctx, projectIDs)`，设计在 3.5、6.5、9.1、11.11 写明 P1b 一次一个项目"。本文的作者核对了设计 :262（3.5）、:1332（6.5）、:1685（9.1）、:2005（11.11）有这句，:1376（6.7）没有 |
| **Minor 2** PT9.7（和的规则丢掉 `max_wait`）由 `TestValidateCrossKeyRules` 钉住，没有一行记下它 | plan Task 9 的表在 T9.16 之后加 T9.17（:6552），Global Constraints（:35）写"`T9.11`–`T9.17`"；spec A.3（:295）"35 个：原型的 21 个和执行中加的 14 个"，（:304）"配置 12 个"并加上 T9.17 和它的说法；A.6（:443）"33 个"、`mutants_t9_fix2.py`（T9.17 是 PT9.7）和"T9.17 由范围复审在 `f18de99d` 上跑"，（:461）6/6 加 `TestValidateCrossKeyRules`，（:468）137 行、136 个被发现、单元一层 101 个、只在单元一层 35 个。A.6 表前的一句（:411）"执行中加的 33 个"不在复审的清单里，是同一个数，一并改。"（Task n 起）"的数目和其他合计不变。T9.17 的层是范围复审在 `f18de99d` 上的运行（`internal/platform/config` 被发现，`validate_test.go:223`、`:229`；`configs`、`cmd/nerve` 留下）和 Task 9 的修复轮（`$M4TMP/p1bsdd/mut-t9-fix2.log:24`） |
| **Minor 3** P1a spec 的状态一行仍是"待评审" | `specs/P1a-issues.md:7` 换成范围复审给的一行（"已完成（评审记录）：…"）；P1b spec :33 说 P1a spec 的那一句改成"…标成已裁定、改了它的状态一行"。本文的作者核对了那一行引的提交 `c37a3480`、`9e9db923`、`91e54133` 都在 |
| **Nit a** spec :33 把设计 3.5、8.8、11.11、13.2 都记到 `e5df62f6` | :33"（预检的 M4：`f094ced6` 改 11.11、13.2，`e5df62f6` 改 3.5、8.8）"；:262"3.5、8.8、11.11 和 13.2 的连接池的总代价" |
| **Nit b** A.1 执行之前的各 Task 的行数是 `compose_plan.py` 量的（不含 Task 之间三行的分隔），执行完的是从标题到标题量的 | 本文的作者从 `git show e5df62f6:docs/v0/M4-issue-core/plans/P1b-issue-update.md` 重新量（从 Task 的标题到下一个标题，Task 15 到文件末尾）：1 495、2 438、3 333、4 1,005、5 1,484、6 748、7 353、8 495、9 472、10 1,007、11 605、12 327、13 229、14 623、15 126，Task 1 之前 145 行，共 8,885 行，与复审的相同。:262 照此写，"最大的是 Task 5（1,484 行…），其次 Task 10（1,007 行）、Task 4（1,005 行）"；"预检之前 8,769 行"是 `a04d479d` 的 `wc -l`，与 8,885 同一种量法。执行完的 plan 因 T9.17 一行变成 9,757 行、Task 9 523 行，同样量过 |
| **Nit c** 第 3 节第 9 条说 P1a 第 5 节 P5 的一行做完了；P5 一行是"`Issue`（列表的行）；状态的名称进 `StateCatalog`"，做完的只有后一半 | spec :174"P1a 第 5 节 P5 一行的"状态的名称"由此做完" |
| **可选 1** 清扫 30（:383）"请求到期" | 照改："在闸门上请求到期" |
| **可选 2** PX9.J 成为 T9.18 | 不改（第 3 节） |
| P1b spec 的状态一行（:7） | "已完成（评审记录）：…"，照 Minor 3 的写法，链接到本文；保留原来仍对的事实，加上范围复审和本提交的遗留 |
| 设计第 15 节 P1b 一行（:2514） | "已完成：[spec]…、[plan]…、[评审]…（执行时 15 个 Task；整分支评审之后一个修复轮）"，照 P1a 一行的写法 |
| 本文的作者另改的，与 Nit a 同类（第 3 节） | spec :33 把 P1a spec 的修改记到"整分支评审的修复轮和评审记录的提交"；:7 写明设计 3.5、8.8 的一句在 `e5df62f6` |

## 5. 最终实现与 plan 的差异

plan 是"执行时的 plan"：每个修复轮也改了它的 Task 的块，代码块写的是提交的代码（第 3 节 Task 4）。所以这里不列逐个提交的漂移表（P1a 评审的 5.2），只写复现的结果和不同的地方。

**复现**（本文写好之后、提交之前，本文的作者重跑：`$M4TMP/p1bsdd/rd/replay.sh`，日志 `replay.log`、`replay-treediff.txt`）：
- 两份新的解压 `$M4TMP/p1btools/base-5ce30a1a.tar`。一份上 `node planapply.mjs <plan> check`："308 blocks {"file":24,"replace":282,"whole":2}; all ok"。
- 另一份上按顺序 `apply` Task 1–15，每个 exit 0；写的文件数依次是 4、12、4、25、16、12、3、2、9、19、8、1、1、4、5。
- `node treediff.mjs <复现> <worktree>`："3369 and 3442 files; 80 differences"：7 个两边都有、内容不同（下面的 5 个生成物、设计和 P1a 的 spec），5 个只在 worktree（`.git`、P1b 的 plan 和 spec、本文、生成的 `update.sql.go`），另 68 个在 `.superpowers/`。

与 plan 不同的地方，都是允许的：
- **生成物 6 个**：`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`issue/adapter/http/gen/bodyshape.gen.go`、`server.gen.go`、`project/adapter/postgres/gen/issue_ports.sql.go`（基点就有，内容不同），`issue/adapter/postgres/gen/update.sql.go`（只在 HEAD）。plan 不写生成物，按 SHA-256 和行数钉住，由各 Task 的 `Run:` 生成。HEAD 的六个与 plan 最后的钉值相同（`shasum -a 256`）：`issue_ports.sql.go` `44b659b2…`（plan :746）、`update.sql.go` `89190635…`（:1188）、`bodyshape.gen.go` `d7008328…`（:4310）、`server.gen.go` `bc130a65…`（:4311）、`openapi.yaml` `2a9bead7…`（:6623）、`schema.gen.ts` `5d4bef92…`（:6624）。
- **后来的几轮改过的文档**：`M4-design.md`（执行之前的 `a04d479d`、`f094ced6`、`e5df62f6`，修复轮的 `f18de99d`，本提交的第 15 节一行；plan 的块不写设计）；P1a 的 spec（`f18de99d` 的六个标记，本提交的状态一行）。
- **只在 HEAD 的**：P1b 的 spec 和 plan（plan 的块不写它们自己）和本文；`.git`；`.superpowers/`（执行的台账、brief 和报告）。

README、总体设计、差异清单和两份交接与复现的逐字节相同：Task 15 的块随 `3693abc7`、`f18de99d` 改过。

**执行中改了的生产代码**（`git log --name-status 5ce30a1a..HEAD`，Go 的非测试文件、SQL、契约、配置和文案；`$M4TMP/p1bsdd/rd/prodfiles.py`）。Task 自己的提交之外：
- 改了行为的只有 `06bd96b4`（Task 9 的修复轮）：`platform/config/validate.go` 的和的规则，连同 `config.go`、`server/configs/config.yaml`、`platform/postgres/tx.go` 的说明。
- 只换写法、不改行为：`3866bf13`（`issue/domain/activity.go`：动态的第 2、12 行一个 `assigneeAdded`）；`9076e2be`（`pickState` 从 `create_issue.go` 移到 `writes.go`，`update_issue.go` 也用它）。
- 只改注释或局部的名字：`d6c22d7e`（`lock.go` 预检的说明；`create_issue.go` 的局部 `named` 改名 `accounts`）、`a50c9c5b`（`platform/gate/gate.go` 的说明）、`e25e2b5e`（`issue/app/ports.go`）、`745a5233`（两个 `Execute` 的说明写上闸门）、`203f1f94`（`bootstrap/gate.go`）。

**文件大小**（HEAD，`wc -l`）：分支改过的手写的代码、测试文件都在约 400 行以内。最长的是 `platform/config/load_test.go` 414 行（N4）、`bootstrap/permission_matrix_test.go` 399 行、`issue_interleaving_test.go` 377 行、`ports_test.go` 376 行；新建的文件中最长的是 `issue/app/update_issue_test.go` 322 行、`platform/gate/gate_test.go` 312 行。契约 `api/modules/issue.yaml` 307 行。plan 执行之前 8,885 行、执行完 9,757 行；Task 5 是 1,619 行，长过约 1,500 行（它的两个修复轮的单元测试和代码，变异表执行中加的十行；spec A.1）。

### 5.1 执行中加的变异

执行中加的变异是 plan 的变异表和 spec A.6 的行（修复轮的裁定 11、12，本提交加了 T9.17），每行写明改法、必须失败的检查和层。这里只给各 Task 的数目和位置：

| Task | 表的行 | 原型的 | 执行中加的 | plan 的表（行号） |
|---|---|---|---|---|
| 1 | 15 | 15 | — | :678–692 |
| 2 | 3 | 3 | — | :1128–1130 |
| 3 | 13 | 11 | T3.12、T3.13 | :1455–1467 |
| 4 | 9 | 6 | T4.7–T4.9 | :2550–2558 |
| 5 | 32 | 22 | T5.23–T5.32 | :4146–4177 |
| 6 | 4 | 3 | T6.4 | :4966–4969 |
| 7 | 6 | 4 | T7.5、T7.6 | :5485–5490 |
| 8 | 14 | 11 | T8.12–T8.14 | :6016–6029 |
| 9 | 17 | 10 | T9.11–T9.17 | :6536–6552 |
| 10 | 21 | 18 | T10.19–T10.21 | :7639–7659 |
| 11 | 1 | — | T11.1 | :8364 |
| 12 | 1 | 1 | — | :8691 |
| 13 | 1 | — | T13.1 | :8926 |
| 合计 | 137 | 104 | 33 | |

（行号是本提交的 plan 的，按每行第一格的编号数：`$M4TMP/p1bsdd/rd/rows2.py`。）每行的改法在 `$M4TMP/p1bsdd/` 的探针文件里，spec A.6（:443）列出每个文件和每行在哪棵树上跑过；它们的缺陷类别、失败的检查和层在 A.6 的第二张表（:445–466）。合计（A.6 :468）：137 行，136 个被发现，T8.11 是等价的；单元一层发现 101 个，组合 63 个，端到端（I2）40 个，存储 21 个，守卫 1 个；只在一层发现的：单元 35 个、存储 6 个（都在 A.3，各有理由），组合 12 个，守卫 1 个，端到端没有。较早的行在执行中新记的层（T1.15、T5.2 在矩阵；T2.2、T2.3 在盖戳的测试；T6.3 在"工作项不在了"）写在各自的行里，各 Task 的表下"也发现较早的 Task 的变异"一段列出。

**不是行的探针**（实现者、评审者在各轮写的，在 `$M4TMP/p1bsdd/` 的探针文件里，结果在各 Task 的报告；控制者只把台账标给评审记录的、整分支评审的 M2 和范围复审的 Minor 2 排进表）：
- **对照**，核对工具和副本，不是代码的性质：PT1.2（PT1.1 的改法放在负责人一侧，原来就被发现）、PT7.2、PX9.1。
- **核对提议或重构的**，在修改之前证明提议的测试或重写咬得住；采纳之后由表里的行或原有的测试钉住：PT1.3（`assigneeAdded` 的重构，两个动态测试都失败）、PT6.F1（改用 `newWorkItem` 之后盖戳测试仍看得到不同的时刻）、PC7.1–PC7.5（重写的创建盖戳测试仍钉住 P1a 的性质）、PX7.P、PT7.1P、PX7.P3、PT9.1P–PT9.6P、PX9.P、PX9.T10、PX9.R、PX9.R2、PR11.1、PR11.2、PR11.2x、ST2.1–ST2.3（`e52d98d8` 之后 T2 的行在修改的盖戳测试上的层）、OPTB.0、OPTB.1（Task 14 的选项 (b)）、范围复审的 RR1（M1 之后真的改动仍在组合一层钉住）。
- **plan 原有的测试发现的**，同一个性质已经钉住：PT2.1–PT2.3、PT3.1–PT3.5、PT7.3（存储一层和矩阵）、PT8.2b。PT8.2 编译得过，却以 panic 失败，不算发现，改写成 PT8.2b。
- **数据的缺口**，修复轮改了数据之后在单元一层被发现，台账没有标给评审记录，没有排进表：PT1.1（`7644759f`）、PT4.1、PT4.2（`b3d45197`；PT4.1 另在 Task 6 起的矩阵，PT4.2 在 Task 4 的矩阵）。
- **由较晚的 Task 钉住，裁定接受的**：PT4.4（已归档的工作项照样读；Task 7 起 `TestAWorkItemReadsAsStored`）。
- **裁定不钉住的**：PT2.4（标签目录按 `id` 的次序；第 3 节 Task 2）。
- **测量，不是发现**：PT12.1（没有闸门时读最慢 3.19 秒，spec 第 3 节第 13 条）。
- PF4.1、PF4.3、PF4.4、PF5.x 是同名探针在最终的快照上的运行，不是别的探针。预检的 35 个 `PF.*` 在执行之前：留下的三个和 PF.D4 由修订做成 T10.17、T10.18、T5.21、T5.22，PF.C1 是数据的规则（A.6 的完成时间一行），其余被发现。

**等价的变异 T8.11**（有人等时新来的拿走空位）：`release` 把位置直接交给队首、不腾出空位，调用者只在位置全被持有时排队，所以有空位时一定没有人等，去掉的条件不改变行为（spec 第 3 节第 11 条、A.3）。它在闸门的单元测试和 `-race` 的负载测试上都通过；留着那个条件，是为了把"先来先得"写在进入的地方。

**只为核对而跑的变异 PX9.J**（`$M4TMP/p1bsdd/ffw/mutants_ffw5.py`）：把 `validate.go` 的 `g.MaxWait+i.WriteLockTimeout >= requestTimeout` 改成 `>= requestTimeout-time.Millisecond`（规则多留一毫秒），在 `internal/platform/config` 被发现，唯一失败的是 `TestValidateAcceptsWriteWaitsJustUnderTheRequestTimeout`（`ffw/mut-logs-check/PX9.J-unit-0.log:1`）。表里没有一行让这个测试失败；它证明这个测试会失败，不进表（spec A.6 :468、清扫 45；范围复审的可选 2 不采纳）。

## 6. 移交事项

同一个里程碑之内的后续 Phase（写进它们的 spec；spec 第 5 节各行照录，另加执行中交出的）：
- **P2**：访客规则：`issue.read`、`issue.update` 的访客（P1b 一律 403），`see` 和预检照样在锁之前答访客规则的 404；已归档的工作项的修改 409 `issue.archived`（P1b 没有归档）；起止日期进 `IssuePatch`（合并之后的值，3.7）；`project.state_in_use` 的补救（P1a 交给 P2 的一条照旧：修改在 P1b 有了，恢复在 P2）。执行中的：`getIssue` 判定的动作（T4.7）今天只在单元一层分得出，访客规则让 `issue.read` 与 `issue.create` 的角色不同之后，组合一层才看得到（A.3）；从 P2 起新的测试用平台的 `postgres.LockNotAvailable`、`postgres.LockTimeout`，不再手写（收尾一行）。
- **P3**：`WriteGate` 取多个项目：按 `id` 升序一次取齐、失败时放回；进第二个闸门时不持连接、不持锁（设计 11.11），只持着前一个闸门的位置（spec 第 3 节第 10 条；架构的裁定要 P3 的架构 brief 点名这一条）。设计 3.5、6.5、9.1、11.11 的多个项目的写法是 P3 的（M5）。`parent_id` 进 `IssuePatch`。
- **P4**：描述、`skip_activity` 进 `IssuePatch`；动态的第 4 行（描述，合并）；I2 的描述的断言。
- **P5**：`sort_order` 的修改；状态的名称已在目录里（P1a 第 5 节 P5 一行的这一半由 Task 2 做完）；工作项一侧的快照事务把隔离级别和锁的期限放在同一个 `BeginQuery` 里（`BEGIN ISOLATION LEVEL REPEATABLE READ; SET LOCAL lock_timeout = …`），否则只加 `postgres.LockTimeout` 会悄悄丢掉隔离级别。；**P5接收结果（2026-10-26）**：T8完成sort_order零/省略/同值/审计及state组合、skip true/false互斥，T12精度/快照、T14锁路径race五遍与T15 I6实证；直接复用P4已与LockTimeout正确组合的ReadOnlySnapshot，不另写BeginQuery。P10b拖放计算/比较器与页面、后续事件仍开放，实际证据见 [P5 review §2](../reviews/P5-issue-list-review.md#2-验收标准逐条核对)。
- **P9**：规模的测量在同一个测试里加任务的一支，种子和命令照旧（`-count=1`，种子的核对）；同步连带的两个方法删去之后这个测试量任务；在规模的数字旁再量第 7 节连接池的情形（删除一个大的工作区、写在它的项目上等行时，另一个项目的读）；连带进了 River 的任务之后，已删除的项目上的写应在预检答出、而不是在它的行上等，由 P9 核对；`workItemRows` 的计数给一个自己的 `context`。
- **P14**：I2 的页面版本，复用 `assert/issue.ts` 的 `byId`、`assigned`、`changed`、`createdFor`。
- **收尾**：3.23、8.8 中 P1b 的行由 Task 15 写好，收尾逐行核对；M8 的部署交接写明连接池的大小与多进程时的 K、同时热的项目数，和一个工作项的写的等待（闸门上至多 `max_wait` 或到请求的期限；每次等锁各至多 `write_lock_timeout`；同时热的项目达到 ⌈池 / K⌉ 个时池里的等待只以请求的期限为界；配置的规则只盖住闸门和一次等锁，等到请求的期限的写答 500）——设计 13.2 的 M8 一行已经写上（`f18de99d`）；14 个测试文件里的 17 处手写的 `pgErr.Code == "55P03"` 改用 `postgres.LockNotAvailable`（spec 第 5 节列出文件）；p6、p7 的局部 `changed` 若要合并在收尾做（整分支评审的分诊）。
- **M4 收尾写 M5–M8 的 `handoffs/M4-closeout.md`**：设计 13.2（:2477）每一行写进接收者的交接，连同各 Phase 的 spec 第 5 节、评审第 6 节交给它的各条。设计 :2482–2485 已经链接到这四个文件，它们还不存在：`doclinks.py` 对设计只报这 4 个链接；架构的本地决定让 Task 15 不查设计的链接，范围复审认为它们是收尾的，不是 P1b 的。P1b 交给 M8 的是上面收尾一行的池和等待，和删除的请求的数字（spec 附录 A.4）。

**交接的处理结果**：P1b 在两份交接里写下"处理结果（M4/P1b）"：[M2 收尾交接](../handoffs/M2-closeout.md)（第 3 节修改的 `updated_at`）、[M3 收尾交接](../handoffs/M3-closeout.md)（第 2 节的连接池、写闸门和 P1b 的批量写按约定五）（Task 15 `5e2945b5`，M3 的一段由 `3693abc7`、`f18de99d` 改准）。两份的 `status` 都照旧是 `open`，在 M4 收尾随 review 关闭（P1a 的做法）；各自仍未处理的条目写在它们的"处理结果"末尾。M4 设计 13.1 落在 P1b 的两项都在其中（spec 第 7 节）。

### P8接收回执（2026-10-27）

T3/T6草稿复用原Locks的authorizer/gate/写事务与统一timeout映射，普通NewLocks五参与项目N路径保持。T7真实gate排队无连接/事务、拒绝零gate/写锁、W/P/D等待重判与D后唯一时钟；T15交错race五遍65/65。T11沿已有ReadOnlySnapshot，不另写BeginQuery；P9规模/连接池后续、P14页面与M8部署仍开放。 实际命令/计数/变异/检查点见[P8 review §8](P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)。产品HEAD `10f44790`、首次整审 `166b9efd..10f44790`；文档独评/最终准确HEAD CI按后续收据追踪。未合并，不改变本页历史验收与集成事实，输入handoff总体仍open。

### 处理结果（M4/P9，2026-10-28）

T11已删项目写在预检拒绝；T14升级原SP2为六项目每个660000行，正常15s请求期限，每job一次running且attempt1/errors空/snoozes0；同workspace邻项目与另存活workspace两类读写探针保存全部160样本。测量仍是4c7379f1+dirty源码绑定d26，附录B保留机器/池/observer与525ms最大样本，不覆盖本页历史同步SP2/SP4数字。

证据：[P9 review §8–12](P9-purge-review.md)、[P9 spec附录B/C/D](../specs/P9-purge.md)；产品身份 `e003e96625ba51264d3aeb77729729d53a770103`，独立整审与两次局部复审结果按上述入口，最终检查点/文档独评分别记账。

仍开放：P14 I2页面、旧手写55P03收尾核对及M8池/K/多进程/超目标性能与等待上界仍开放。 本回执只承接P9后端子项，不关闭整份交接或M4，不改变本页历史运行/集成身份。

## 7. 已知限制

- **规模目标上删不掉一个工作区**（决策点 7 的已知，spec 第 7 节、附录 A.4）：五个 2 万个工作项的项目的工作区，请求约 23–27 秒，超过生产的 15 秒期限，到期时回滚、什么都没删；一个这样的项目在期限之内（spec 第 7 节写约 4.4–4.8 秒，执行中 Docker 空闲时的两次是 4.95、5.08 秒）。P9 换成任务。数字是一台机器（macOS、Docker Desktop）上的，按 ±2–3 倍读。
- **连接池的总代价**（设计 11.11，spec 第 7 节，预检的 M4）：K 是每个进程的；闸门限的是每个项目的份额，一个进程里 ⌈池 / K⌉ 个项目同时热（默认 5 个）就能占满池，另一个项目的读和写在池里等。要占满池得长时间持行锁，今天只有过渡的同步连带会这样（到 P9）；它让别的请求慢几秒，不丢数据、不死锁。M8 按同时热的项目数和进程数定池的大小。
- **热的项目上超出的写得到 503**（11.11 的代价）：排满、`max_wait`、在闸门上到期、等锁到期；客户端按 `Retry-After` 重试。
- **等到请求的期限的写答 500，不是 503**：配置的规则（`max_wait` 加 `write_lock_timeout` 短于 `server.request_timeout`，`validate.go:152-159`）只盖住闸门的等待和**一次**等锁。等池里的连接（⌈池 / K⌉ 个项目同时热时）、或等不止一次锁（例如排在改地址的事务后面等 `users` 的一行）的写还会到期；期限是 `platform/httpserver/api.go:168` 的 `context.WithTimeout`，闸门之外的 `DeadlineExceeded` 不是 ProblemError，落到 `apierrors.go:189-195` 的默认分支，答 500（范围复审）。不在 `busyTx` 里换成 503：提交的超时也包着 `DeadlineExceeded`，结果不知道，重试可能写两次（Task 9 的裁定）。默认值给第二次等待留 3 秒（2 + 5 + 5 = 12 < 15）。写进 M8 的交接（第 6 节）。
- **两处裁定过的固定等待**：`bootstrap/gate_test.go:242`（请求在闸门上到期的测试：alice 的创建在 bob 的修改发出之后 2 秒才发，Task 10 的修复轮）；`bootstrap/issue_pool_test.go:223`（交错 20 的持有至少一秒，每 10 毫秒查一次；架构的裁定 13：让没有闸门的变异的读超过上限）。其余的等待都按条件、有失败的期限（整分支评审）。
- **测试的配置是加载会拒绝的值**：`testIssueConfig` 2 + 4 秒不短于 `testConfig` 的 5 秒，请求到期的测试 10 + 2 ≥ 3，池的测试的 `max_wait` 一分钟；测试的世界从不校验配置（`validate()` 只在 `config.Load` 里），说明写明（`issue_writes_test.go:64-69`、spec 第 3 节第 15 条）。
- **只在单元、存储一层的性质**：spec 附录 A.3 逐条写明理由（按性质只在单元一层 35 个，只在存储一层 6 个）。其中 `getIssue` 判定的动作到 P2 才在组合一层分得出（T4.7），离开闸门在事务结束之后只在单元一层看得到（T10.17）。
- **T8.11 是等价的变异，PX9.J 只为核对而跑，PT2.4 不钉住**（5.1）：标签目录按 `id` 的次序没有读者依赖。
- **持续集成没有跑过**：原型、复现和执行都在 macOS 上（spec 第 6 节最后一行）；ubuntu 上的结果在合并时看。
- **接受、不改的**：N3（两个按用例的端口各自声明四个插入）；N4（`load_test.go` 414 行）；p6、p7 的局部 `changed`；外观：Task 8 复审的 `gate_test.go:191-193` 的说明、Task 12 的折行和窄的测试名、Task 9 复审的两个 Nit（和的溢出要在约 292 年之外，与 M2 的刷新规则同形；`validate_test.go:228` 的"well under"：11 秒对 15 秒）。

### M4/P6 接收记录

修改/排序事件及无改动边界已按P6规格实施，提交、真实测试/变异与独立评审见 [P6 review §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)及[§5](../reviews/P6-issue-events-review.md#5-最终实现与-plan-的差异)。P7/P9/P15和M4整体交接仍按原文开放。
