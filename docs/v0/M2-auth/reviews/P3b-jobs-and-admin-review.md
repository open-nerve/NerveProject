# M2/P3b River 与管理命令：评审记录

| 项 | 内容 |
|---|---|
| Phase | M2/P3b `jobs-and-admin` |
| 日期 | 2026-09-27 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完，修复的范围评审没有遗留） |
| spec / plan | [spec](../specs/P3b-jobs-and-admin.md) / [plan](../plans/P3b-jobs-and-admin.md) |
| 分支 | `worktree-m2-p3b-jobs-and-admin`，从 `main` 的 `d6f313b`（P3a 的合并）分出 |

## 1. 评审方式

- **拆分与负责人的决定**：P3 按负责人的裁定（2026-09-26）拆成 P3a 和 P3b。命令行的"只投递"River 客户端经负责人批准推迟到 M4（M2 设计 3.15、3.17、13.2 的 M4 一行），P3b 不建。决策点 1（邮箱只能由命令行修改）、2（`nerve users create`）、3（`deactivate`、`activate` 命令）按负责人的裁定实现。
- **spec 和 plan 由架构子任务产出**（`43bff3b`）：
  - 先在仓库外把整个 P3b 做出来（原型）。plan 中的代码从原型原样拼入，48 个差异块用 `patch -p1` 逐个核对。
  - 再从 `d6f313b` 的副本出发，按 plan 逐个 Task 重放 12 个 Task，每个 Task 之后 `make lint-go`、`make test` 都通过；结果与原型逐文件相同（2,608 个文件）。端到端 22 个测试中 21 个通过，S3 失败的原因与 P1–P3a 相同：副本不是 git 仓库。
  - 原型中 123 个变异有 122 个被发现；没被发现的是停机时迁移执行器与连接池的关闭顺序（spec 附录 A、第 6 节）。架构子任务的变异还发现了 4 处弱点，在 plan 中已改好。
- **执行前的核对**：
  - 控制者逐对核对共用文件和接口的 Task（15 行），没有发现冲突。
  - spec 第 3 节的 16 项差异全部接受。第 3、7 两项改了 M2 设计的文字，在控制者的提交 `777f1b1` 中同步（第 3 节）。
  - P2、P3a 评审交给 P3b 的事项落到各 Task（第 6 节）。
- **逐个 Task 评审**：12 个 Task 各由一个实现子任务写入代码，再由独立的评审子任务（sonnet）检查"是否符合 spec"和"代码质量"。
  - 实现者按要求对自己的代码做变异核对。**12 个 Task 中有 8 个**（Task 1、3–9）发现了以下问题之一，控制者逐项裁定，修正都在评审之前提交（第 3 节）：
    - plan 自带的测试没有咬住它声称守护的性质；
    - 测试会挂住而不是失败；
    - 注释、文字写了代码没有的行为；
    - plan 的代码与 spec 不符（Task 8 的 `readPassword`）。
  - 另外两个 Task 的问题不是由实现者的变异核对先发现的：
    - Task 11 的三处测试缺口由控制者在派发前读 brief 时发现（T11-a/b/c），实现者的变异证实了 plan 版本的故事在对应的变异下照样通过；
    - Task 12 按控制者的要求逐句对照代码核对新写的文字，改写了 4 处。
  - Task 2 的 plan 测试都成立（14 个变异都被发现），问题在导出文件的保护上；Task 10 没有问题。
  - 逐个 Task 评审没有要求返工。评审提出的 Minor 由控制者分拣：放进修复轮（第 4 节），或者接受。
  - 下一个 Task 的实现与上一个 Task 的评审并行，同一时间只有一个实现者提交。
- **C1 调查**：Task 11 实测停机时，20 次干净的停机中有 10 次 River 记了 ERROR。控制者读了 river@v0.47.0 的源码，要求先在仓库外调查、再裁定：
  - 210 次运行（spec 附录 A 的 C1）：ERROR 只出现在 River 启动后的最初几秒；运行 10 秒、30 秒后停机 0/40。换一种停止方式也不改变比例：这两处日志不看取消的原因。
  - 但取消的原因影响 River 的重建索引：停机打断重建时，原来的 runner 5/5 留下 INVALID 的 `_ccnew` 索引，River 此后不再重建它；改后 5/5 删掉。
  - 修正（`bca8347`，第 3 节 C1-a）另有一次范围评审（sonnet），通过，留下的一个 Minor 放进修复轮。
- **整分支评审**（opus，`d6f313b..2c6a94b`）：
  - 结论为"修复后可合并"：0 个 Critical、1 个 Important、2 个 Minor；没有发现生产代码的缺陷。
  - 评审跑了 `go test ./...`（32 个包）、`make lint-go`，交错测试和 `platform/jobs` 的 `-race -count=3`，bootstrap 的清理、停机顺序、连接池测试的 `-count=5`。它用锁定版本的命令行重新导出 River 的迁移，与 `00005` 逐字节相同；另在副本中探测了 archtest 的调用图规则和非 UTF-8 的地址。
  - 修复在一轮中完成：8 个提交，`b78c67a` 至 `281aba6`（基于 `2c6a94b`），包括 F1、F2 和逐个 Task 评审时停放的 6 项（第 4 节）。
  - 修复之后对 `2c6a94b..281aba6` 做了一次范围评审（sonnet）：8 项都已处理，没有新问题（第 4 节末）。
- **持续集成**：推送后通过 GitHub 公开接口（curl 和 jq）确认结果，见第 2 节。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | A12–A14、A16、A17 的接口版本通过，此前的故事仍然通过 | **通过。**<br>本机 `make e2e` 的 22 个测试全部通过，覆盖 S1–S4、A1–A17（Task 11；C1 修正之后又跑一次）。<br>五个新故事用 `--repeat-each=5` 跑，25 个全部通过。<br>T11-a/b/c 加的断言各有变异证明能失败。<br>持续集成的 e2e 任务通过（第 2 节末） |
| 2 | 交错测试 1–3 在真实数据库上通过，`-count=5`、`-race -count=5` 也通过；六个交错测试至此全部通过 | **通过。**<br>身份模块的 9 个交错测试函数（含 P3a 的交错 4、5）和 4 个子测试，`-count=5`、`-race -count=5` 都通过，没有 `DATA RACE`。第 6 个（P2 的 `TestTheCredentialLockBlocksLocksNotInserts`）在 `make test` 中通过。<br>每个锁的变异都在交错 1–3 中被发现：去掉签发方或管理员锁的 `FOR NO KEY UPDATE`，把登录、创建 PAT、重置的锁挪到事务外（登录和创建 PAT 的两个变异合起来覆盖 1–3）。失败都在约 10 秒内，不挂住。<br>Task 10 的评审另用 overlay 重跑了 3 个锁的变异 |
| 3 | 清理任务的测试通过：只删除过期的会话，跳过被锁住的会话；5 个迁移都能 up、down、再 up | **通过。**<br>`TestDeleteExpiredSessions`：`<` 改成 `<=`、忽略批大小、跳过已撤销的过期会话、删掉所有会话，这些变异都失败；已撤销而未过期的会话留下。<br>`TestDeleteExpiredSessionsSkipsLockedRows`：去掉 `SKIP LOCKED` 或行锁时，0.54 秒内因 `lock_timeout` 失败。<br>`TestMigrationsGoUpDownAndUpAgain`：10 个迁移变异都失败。<br>A14 在端到端中核对过期的会话被删、未过期的留下 |
| 4 | 实测的停机时间在端到端 fixture 的预算内，写进 review | **通过。**<br>Task 11 按 spec 附录 A 的 E5 实测（C1 修正之前的 runner；每种情形 10 次，最小/中位/最大）：就绪后立即停机 3.5/4.3/4.6 毫秒，就绪 3 秒后 4.1/9.1/17.6 毫秒；前一轮是 1.5/2.6/4.6 和 3.6/8.1/11.4 毫秒。都以 0 退出。<br>C1 调查的 210 次（其中改后的 runner 60 次）都在 3.3–21.7 毫秒之间，都以 0 退出。<br>fixture 的 `stopTimeoutMs` 是 30 秒，实测低三个数量级以上；worker 的预算是 70 秒。<br>按配置的最坏情况约 36 秒（HTTP 20 秒、任务 10 秒加 1 秒、连接池 5 秒），超过 30 秒：那时 fixture 在 30 秒 SIGKILL，报出日志路径，不会挂住（spec 第 6 节）。README 部署一节写明了这个最坏情况和容器的停止宽限期；M8 的部署文件要设 `stop_grace_period`（M2 设计 13.2）。<br>重建索引的实验另计：改后的 runner 等挡住删除的事务结束，SIGTERM 后约 0.6 秒以 0 退出；删除被挡满 15 秒时，15.0 秒以退出码 1 退出（第 7 节） |
| 5 | 命令行的组合只有连接池和 `Admin`（`bootstrap.Users`；`TestUsersCommands` 核对 `river_job` 是空的） | **通过。**<br>`TestUsersCommands` 核对 `river_job` 是空的。<br>但 `Users` 建出 River 客户端（不带任务）或 HTTP 一侧时 `river_job` 仍是空的，所以 Task 8 另加 archtest `TestUsersComposeNoServerAndNoJobs`：从 `bootstrap.Users` 沿静态调用图走，碰到 `identity.New`、`httpserver`、`ratelimit`、`jobs` 或 River 就失败，走不到 `identity.NewAdmin` 也失败。<br>整分支评审在副本中让 `Users` 调用 `jobs.New`，测试失败 |
| 6 | 存的哈希不可用时登录答 401，耗时与密码错误相同（`TestLoginTakesAsLongForAnUnknownAddress`） | **通过。**<br>改之前测试失败：`logins = 401, 401, 500`。<br>改之后三种登录的中位数 14.853、14.906、14.880 毫秒；`-count=3` 中最大相差 2.5%，容差 25%。<br>去掉替身、不做 argon2、替身用便宜的参数，这些变异都被这个测试发现。<br>关闭 P2 评审第 6 节交来的一项 |
| 7 | `make lint`、`make test`、`make gen-check`、`make knip`、`make e2e` 通过 | **通过。**<br>Task 1–11、C1 修正和修复轮提交之前，`make lint-go` 两段都是 `0 issues.`，`make test` 通过。<br>有生成物的 Task（2、4、6、8）核对了 SHA-256；提交后 `make gen-check` 干净。<br>改了 web 或 e2e 的 Task 8、11 跑了 `make lint-web`、`make knip`、`make e2e`；Task 12 和修复轮跑了 `make lint-web`。<br>`2c6a94b` 的持续集成三个任务都通过 |
| 8 | 3.20 中 P3b 的各行、8.7 中 P3b 的三行在同一次合并中写好；交接按第 7 节处理 | **通过**（Task 12 及其核对）。<br>Task 12 的评审对照 M2 设计 3.20（P3b 一行）、4.6（P3b 的五行）、8.7（P3b 的三行），确认都已写入；README 停机的两条逐句对照代码和 river@v0.47.0 核对。<br>整分支评审另核对了 README、总体设计 4.2、差异清单、M2 设计 3.3/3.17/12/13.2/M18、三份交接和两个契约说明。<br>三份交接追加"处理结果（M2/P3b）"，仍为 `open`（第 6 节） |

持续集成：
- 修复轮之前，`2c6a94b` 的分支 run 36260722067：web、server、e2e 三个任务都通过。
- 修复轮之后的分支持续集成在合并时记录。

## 3. 执行中的决定

spec 第 3 节的 16 项差异在执行前全部接受。它们都没有改动 M2 设计的架构，做错了也只是一个 Task 内的返工。其中两项改了 M2 设计的文字，在执行之前由控制者提交（`777f1b1`）：
- **3.3、3.17、6.2 的目录树、12 节、M18**：管理员的用例由 `identity.NewAdmin(AdminDeps)`（`admin.go`）组合，不是 `(*Module).Admin()`（第 3 项）。`cmd` 只导入模块入口、`bootstrap` 组合、`cmd` 解析参数的边界不变；做成方法会逼命令行先建 HTTP 那一侧，违背 3.17 的最小组合。
- **13.2 的 M8 一行**：默认配置下停机最坏约 36 秒，超过 Docker 默认的 10 秒宽限期，部署文件要设 `stop_grace_period`（第 7 项）。实测停机只有几毫秒，满足完成线。

执行中加了第 17 项（C1-a）。

执行中的裁定。除特别注明的以外，都只改测试、注释或文字，都在这个 Task 的评审之前提交。Task 1–9 的由实现者的变异核对发现；T11、T12 两组是控制者在派发前读 brief 时加的；C1-a 来自 Task 11 实测停机时的发现：

| Task | 决定 | 原因 |
|---|---|---|
| 1 | 加 `TestValidateAcceptsACleanupIntervalOfOneSecond`：恰好 1 秒的清理间隔合法。`config.go`、`validate.go`、`validate_test.go` 的注释照 River 的原话写：River 的文档说间隔不应少于 1 秒，但 River 不强制，由 nerve 的校验强制（改了生产代码中的注释） | 把 `<` 改成 `<=`（拒绝恰好 1 秒）的变异原来存活。plan 的注释说"River 的定时任务最多每秒一次"，River v0.47.0 并不这样执行（`periodic_job.go:12-14`） |
| 2 | `.gitattributes` 把 `00005` 标为 `-text linguist-generated=true`，`.editorconfig` 只对这个文件关掉行尾空白的修剪，照 `tools/plane-schema/*.sql` 的先例（改了仓库配置） | River 导出的 SQL 有两行带行尾空格，编辑器按 `.editorconfig` 保存就会改掉它们，文件与钉住的 SHA-256 不再相同 |
| 3 | 每次调用 `Stop` 都有期限（`stop` 帮助函数）；加 `TestRunnerWorksTwoJobsAtOnce`（最多 2 个 worker）、`TestStopWaitsForAnAttemptUnderWay`；钉住 1 秒的宽限和 1 秒的首次重试；断言 River 自己的 "River client started" 进入 runner 的日志；只用假客户端的测试移到 `runner_test.go`（纯移动，`jobs_test.go` 从 425 行降到 299 行）。接受两个存活的变异：`Stop` 取消进行中的尝试后多记一条 WARN；重试间隔的上限（30 秒）改掉 | `Stop` 不结束重试时测试挂 10 分钟；worker 的上限、`Stop` 等待进行中的尝试、宽限和首次重试的值、River 的日志去向，原来都没有测试看到。前一个接受的变异只多一条日志，后一个在合理的测试时间内看不出 |
| 4 | 假存储每次调用把时钟拨快 1 秒，与用例共用一个时钟；worker 的真实数据库测试加一条：取消的 ctx 让 `Work` 以 `context.Canceled` 失败；`TestDeleteExpiredSessionsSkipsLockedRows` 的每个等待（含持锁的事务）都在一个 10 秒的 ctx 里结束。接受空的定时任务 id；接线的 5 个变异交给 Task 5 | 每批重新读时钟的变异存活（`clocktest.Fixed` 不动）；两个假实现都忽略 ctx，丢掉 ctx 的变异在整个 server 的测试中存活；持锁的一方出问题时测试挂住而不是失败。定时任务的 id 只写进任务的元数据，看不到；接线要到 Task 5 才有运行的 River |
| 5 | `TestRunStopsTheJobsAfterHTTP` 的早期快照也拒绝 River 自己的 "River client stopped"；`TestCloseDoesNotWaitForAConnectionInUse` 先断言 `newApp` 设的是 5 秒。接受 `run` 丢掉 `Stop` 的错误这个变异。**不加** Task 3 评审停放的"`Stop` 先于 `Start` 时返回 nil" | River 随调用者的 ctx 停下的变异在 bootstrap 层存活；5 秒的期限没有钉住，设为 0 只因竞争才被发现。看到 `Stop` 的错误要一个测试专用的任务。`Stop` 唯一的调用者满足"先 `Start`"的前提，不写预防性的代码；做错的代价是以后调换顺序时停机会 panic，任何启动 bootstrap 的测试都会发现。Task 4 交来的接线变异都死在 `TestTheSessionCleanupRunsAsConfigured` |
| 6 | `admin_test.go` 的 `holdLock` 和 P3a `credentials_test.go` 的同样写法都加 10 秒的期限。接受"不区分大小写匹配"这个存活的变异 | 行已被锁住时测试挂住而不是失败（演示挂了 25 秒）；P3b 没有别的 Task 再改 `credentials_test.go`。存储按约定接收规范化的地址，规范化由用例负责（Task 7 核对） |
| 7 | 创建和重置都只哈希一次；加 `TestResetPasswordWhenTheHasherIsBusy`（哈希在事务外、锁外）；`set-email` 用未规范化的旧地址测"新旧相同"；撤销 PAT、数 PAT、按邮箱停用的写入，三条错误路径各有测试；补上 brief 的文字列出而代码缺的"常见密码"用例；注册关闭时用合法的输入也测一次：答 403，不做 argon2（P1 留下的缺口）。接受 `activate`、`deactivate` 的幂等 | 60 个变异中 8 个存活：哈希两次、在锁内哈希、用未规范化的旧地址比较、忽略三处错误，以及注册关闭时先哈希（原来的输入在哈希之前就校验失败）。探测不应占用哈希的名额 |
| 8 | 补上 CLI 的测试：`set-email` 的错误路径和退出码 1、`deactivate`/`activate` 的接线、`--email` 必填、迭代次数与并行度分开；加 `TestUsersPrintAndLogNoSecret`（DEBUG 下两个密码、两个哈希和它们的派生密钥，原文、转义、十六进制、两种 base64 都不出现）、`TestReadPasswordFromAPipe`；加 archtest `TestUsersComposeNoServerAndNoJobs`（第 2 节第 5 行）；**`readPassword` 只去掉行尾的 `\n` 或 `\r\n`**（改了生产代码）。接受终端路径的两个变异；`max_concurrent_hashes`、`max_wait` 的接线停放到修复轮 | 45 个变异中 plan 的测试只发现 28 个：CLI 层没有查密码的测试；`Users` 建出 River 客户端或 HTTP 一侧，按包的架构规则看不出。原来的写法会去掉单独的行尾 `\r`，违反 spec 第 3 节第 14 条。终端路径的测试要伪终端，得改 `go.mod` |
| 9 | `TestVerifyMatchesNothingAgainstAnotherFormat` 要求日志恰好是一条只有 `time`、`level`、`msg` 的 WARN，校验合格的哈希时什么都不记。接受等价的变异（去掉 WARN 之后的提前返回） | 23 个变异中 plan 的测试发现 15 个：WARN 带上密码或哈希的十六进制、base64、字节写法，记两次，或者每次校验都记，原来都通过。去掉提前返回后与替身的随机密钥比较，匹配的概率是 2^-256 |
| 10 | 无需裁定。接受评审的 Minor：读库的帮助函数 `credentials()` 只在一个账户的库上运行，它的 `WHERE` 看不出 | 17 个变异中每个锁的变异都在它涉及的交错中被发现。重置等锁的方向看不到管理员锁是否存在（重置自己的 `UPDATE` 也锁行），由重置持锁的方向钉住。`credentials()` 只是读取，写入由 Task 6 的两个账户的存储测试钉住 |
| 11 | 控制者在派发前加的三条：<br>**T11-a**：A12、A13、A16 在各步之后比较另一个账户的整行（`is_active`、密码、`updated_at`）和它的会话、PAT；<br>**T11-b**：`nerveUsers`、`nerveUsersFails` 都在输出中查密码的原文、十六进制和 base64；<br>**T11-c**：`nerveUsers` 断言命令在 DEBUG 下确实记了日志。<br>实现者另把 plan 的 `.trim()` 改为只去掉行尾的换行。<br>接受：3 秒后停机的 10 次中清理任务只完成过 8 次（Task 12 的文字不说它已运行）；A17 没有第二个账户的状态核对；评审的两个 Minor | plan 只查另一个账户的会话，没有 `WHERE` 的 `UPDATE users` 也能通过；失败路径拿到密码却不查输出；test 配置的日志级别是 `warn`，不开 DEBUG 时"日志里没有密码"不可能失败。三处都由变异证实：plan 版本的故事在对应的变异下通过。`.trim()` 会对首尾带空格的密码算错十六进制和 base64。A17 的 `create` 是插入，拒绝由 `expectNothingAdded` 钉住 |
| C1-a | **Runner 只在 River 的 `Start` 还在运行时取消它的 ctx；启动之后只调用 River 的 `Stop`，`Stop` 返回后才释放启动时的 ctx**（`bca8347`，改了生产代码；spec 2.5、第 3 节第 17 条、第 6 节两行、附录 A 的 C1）。不按消息过滤 River 的日志 | 原来的 runner 先取消 River 启动时的 ctx，River 的停止原因成了 `context.Canceled`；River 的重建索引只在原因是 `startstop.ErrStop` 时删掉没建完的索引（`reindexer.go:263`）。悄悄地永久失去索引维护，比罕见而响亮的退出码 1（第 7 节）更糟。过滤日志是补丁 |
| 12 | 控制者在派发前加的三条：<br>**T12-a**：README 部署加一条停机：顺序、默认值、最坏约 36 秒、宽限期要更长（Docker 默认 10 秒，`stop_grace_period`）、SIGKILL 的代价；<br>**T12-b**：M0-P6 交接的停机数字用 Task 11 实测的；<br>**T12-c**：新文字的每个事实都对照代码核对。<br>按 C1-b，README 另加"停机时 River 的日志"一条，含 `_ccnew` 的恢复方法。<br>核对改写了 4 处：总体设计 4.2 的账户行锁只列改动已有账户的四个命令；M0-P6 的 worker 预算是 `+ 10_000` 毫秒（70 秒）；M0-P6 的停机数字；README 重建索引一段分清 11 秒和 15 秒两个门槛。另加 Kubernetes 的 30 秒默认值、M0-P2 交接中第 17 条的一句 | P3b 的任务让最坏停机超过 Docker 的默认宽限期；运维文档是 README，13.2 的 M8 一行只管 M8 发布的 compose 文件。plan 的文字写了代码没有的行为：`create` 不锁行，管理员命令没有调用者的凭证可复核。控制者给 C1-b 的措辞把两个门槛混在了一起 |

控制者在过程中的其他决定：
- **修正只落在后面的 Task 不再改的文件里**：每次修正之前按 plan 的文件表核对（例如 Task 1 的 `validate_test.go`、Task 3 的 `jobs_test.go`、Task 5 的 `bootstrap/jobs_test.go`、Task 7 的 7 个测试文件）。不属于当前 Task 的修正停放到修复轮。
- **C1 调查只在仓库外进行**：调查期间工作区没有改动，Task 12 等调查和裁定之后才派发（它的 README 和 M0-P6 数字依赖结论）。
- **规则的偏离**：实现者 3 次、评审者 1 次用只读命令违反了 shell 规则（`cd` 连用、git 的管道、`git -C`），都没有改动文件，记在进度表中。

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| I1 `TestStopLeavesTheStartedClientToItsOwnStop` 偶发失败：`Stop` 可能落在 `unwatch()` 之前（spec 2.5 写明的窗口），测试把正确的代码当成错的。评审实测 `-race` 下 3,000 次失败 14 次、6,000 次失败 27 次，约 0.45% | Important | **已修（F1，`b78c67a`）。**<br>测试先等 runner 记下 "jobs started"，再调用 `Stop`。修复前 `-race` 下 6,000 次失败 34 次；只加等待时 6,000 次 0 次；最终的测试 3,000 次 0 次。<br>超出 brief 的一处，控制者接受：假客户端的 `Stop` 在 100 毫秒内看着启动时的 ctx。原来的单次读取看不到"去掉 `unwatch()`"的变异（20/20 通过），这个变异会随机地把 C1 带回来；现在 20/20 失败。正确的代码在客户端的 `Stop` 返回前不会取消这个 ctx；代价是这个测试多 100 毫秒 |
| M1 `bootstrap/users.go` 的 `fieldNames` 是包级的 `map`，违反总体设计 6.3 规则 4（禁止全局可变状态） | Minor | **已修（F2，`39c1f16`）。** 改为函数 `cliFieldName`，用 `switch`，其余的字段名原样返回。去掉任一分支或让默认分支返回空串的 4 个变异都被现有测试发现 |
| M2 四个管理员用例在等锁之前读时钟：命令等在一个登录或创建 PAT 后面时，撤销时刻可能早于那个凭证的创建时刻；`activate` 可能算上等锁期间过期的 PAT | Minor | **不改。** 没有安全上的影响：撤销仍然覆盖新的凭证。P3a 的用例也是这样写的，只改 P3b 会让两者不一致；两处一起改，又要为一个没有设计规则要求的时间先后改动 P3a。代价：撤销时刻可能比并发凭证的创建时刻早一个等锁的时间；以后要改，是每个用例挪一行 |

评审"不作判断"的清单，控制者照评审的写法接受，每一条都有 spec 第 3 节或进度表中的裁定：
- 管理员撤销 PAT 时只把 `updated_by_id` 置空，不动 `created_by_id`（spec 第 3 节第 5 条）。
- 手写的 PHC 字符串带离谱的参数时按它自己的参数校验：它不是 spec 2.11 说的"不可用的密码"，参数的变化由 M2 设计 §16 管。
- `Stop` 等一个已在进行的启动尝试（spec 2.5，`TestStopWaitsForAnAttemptUnderWay` 钉住）；`Stop` 先于 `Start` 时 panic（第 3 节 Task 5）。
- 对已停用的账户 `deactivate`、对已启用的账户 `activate` 都成功（第 3 节 Task 7，幂等）。
- 未知账户的 `set-email` 新旧相同时答 `email_unchanged` 而不是 `account_not_found`：spec 2.9 故意把这个检查放在锁之前。
- README 恢复步骤的第 1 步（逐个撤销不认识的 PAT）没提结束攻击者的会话：它逐字照抄 M2 设计 8.5，是设计层面的事。
- `DeactivateDeps` 带两个互为替代的锁（spec 第 3 节第 11 条）。
- M2 设计 3.15 仍写"默认 10 秒"，不带多出的 1 秒：spec 第 3 节第 7 条接受，13.2 的 M8 一行写 10+1。
- `cmd/nerve/main_test.go` 的 `TestServeUntilCancelled` 等 `<-done` 没有期限：这一行早于 P3b，它等的停机有配置的期限。
- 非 UTF-8 的地址：评审在副本中探测，三个命令都答 "No account has this e-mail address."，没有问题。

放进修复轮的另外 6 项，都是逐个 Task 评审和 C1 修正的评审中控制者裁定要改、但不属于当时那个 Task 的文件：
- **P1 `UNLOGGED` 表的所有权**（`f848b52`，Task 2 评审）：`sqlc_cases_test.go` 加一个用例：另一个模块的迁移 `ALTER` 一张 `UNLOGGED` 表，按所有权报错。它能发现一种只有它看得到的变体：把 `UNLOGGED` 表记为已知而没有所属模块，其余 archtest 都照样通过。
- **P2 `names()` 的遮蔽**（`605ae79`，Task 2 评审）：`schema_test.go` 的局部变量改名，行为不变。
- **P3 适配器的注释**（`050860f`，Task 4 评审）：`sessions.go` 去掉写出应用层批大小的注释。
- **P4 `max_concurrent_hashes`、`max_wait` 的接线**（`1754151`，Task 8 的 C4，P2 留下的缺口）：整程序测试 `TestPasswordHashingIsLimitedAsConfigured`，三组设置（1 个名额等 250 毫秒；1 个名额等 1.5 秒；2 个名额），5 个接线变异都失败。它用不可用的密码作闸门：hasher 在持有名额时记那条 WARN。控制者接受这个依赖：以后把这条日志挪到释放名额之后，测试会响亮地失败（200 而不是 503），注释写明了。
- **P5 失败的启动尝试释放 ctx**（`cd92453`，C1 修正的评审）：`TestAFailedAttemptReleasesItsContext`。去掉 `cancelAttempt()` 的变异失败，而其余 runner 测试在这个变异下都通过。
- **P6 spec 的两处更正**（`281aba6`，Task 12 的报告）：附录 A 的 C1 改为 210 次（原 runner 150 次，其中 50 次是 debug 级别的诊断运行；改后 60 次）；第 6 节重建索引的代价一行与 README 一样精确。

修复轮之后：`make lint-go` 两段 `0 issues.`；`make test` 32 个包通过；`jobs`、`bootstrap` 的 `-race -count=5` 通过；`make gen-check` 干净；`make lint-web` 通过。

修复的范围评审（sonnet，`2c6a94b..281aba6`）：8 项都已处理，没有新问题，也没有引入回退或覆盖的缺口。评审自己重跑了：
- F1：未改的代码 `-race -count=500` 0 次失败；去掉 `unwatch()` 的变异 500/500 失败。
- P5：未改的代码 `-race -count=200` 通过；去掉 `cancelAttempt()` 的变异 20/20 失败。
- P4：`-race -count=20` 通过；闸门确实在持有名额的区段里，每个等待都有期限。
- P6：对照 `reindexer.go:263-266` 和 River 的 `client.go` 核对，15 秒与 11 秒同时开始计时，连接池关闭的窗口总长过 River 的 15 秒。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改，以 spec 和代码为准。

1. **第 3 节的测试和文字修正**：spec 列出的测试因此与实际不同：
   - `TestUsersCommandsFail` 是 8 个用例，spec 写 6 个。
   - `TestUsersReadThePasswordFromStandardInput` 核对的哈希是 `m=64,t=2,p=1`（测试把迭代次数设为 2，与并行度分开），spec 写 test 配置的 `m=64,t=1,p=1`。
   - 各 Task 和修复轮加了 spec 没有列出的测试，见第 3、4 节，例如 `TestRunnerWorksTwoJobsAtOnce`、`TestStopWaitsForAnAttemptUnderWay`、`TestAFailedAttemptReleasesItsContext`、`TestUsersComposeNoServerAndNoJobs`、`TestPasswordHashingIsLimitedAsConfigured`。
   - plan 没有的新文件：`platform/jobs/runner_test.go`（Task 3 从 `jobs_test.go` 移出）、`archtest/composition_test.go`（Task 8）；`.gitattributes`、`.editorconfig` 各多一段（Task 2）。
2. **plan 中"River 的定时任务最多每秒一次"的说法保持原样**：plan 第 81 行，以及 `config.go`、`validate.go`、`validate_test.go` 差异块中的注释（第 115–116、178、366 行）。它说的是 `auth.session_cleanup_interval` 至少 1 秒的理由。River v0.47.0 的文档只说间隔不应少于 1 秒，并不强制（`periodic_job.go:12-14`），强制的是 nerve 的配置校验；提交的注释照此写（Task 1，`e96883a`）。
3. **生产代码的改动**（执行中的裁定）：
   - Task 1：`config.go`、`validate.go` 的注释。
   - Task 8：`readPassword` 只去掉 `\n` 或 `\r\n`。
   - C1-a（`bca8347`）：runner 在 River 启动之后只用它自己的 `Stop` 停止它。
4. **修复轮**（`b78c67a` 至 `281aba6`），改了生产代码的有：
   - F2：`cliFieldName` 代替包级的 `map`。
   - P3：`sessions.go` 的注释。
5. **生成物**：Task 2 导出的迁移（521 行，SHA-256 `03e6c6a0…9699`）、Task 4、6、8 的生成物都与 plan 记录的 SHA-256 相同；修复轮之后 `make gen-check` 仍然干净。
6. **父文档同步**：
   - **执行前的 M2 设计同步**：`43bff3b`（15 节的进度表）；`777f1b1`（3.3、3.17、6.2 的目录树、12 节、13.2 的 M8 一行、M18）。
   - **Task 12**（`2c6a94b`）：总体设计 4.2；差异清单二·按表（River 的表）、四（新增 4 行，改写 3 行）；README 部署一节（第一个账户、管理命令、令牌泄露后的恢复、停机、停机时 River 的日志）；三份交接的处理结果。
   - **spec 的更正**：C1 修正改了 2.5、第 3 节第 17 条、第 6 节两行、附录 A 的 C1（`bca8347`）；修复轮 P6 改了附录 A 的 C1 和第 6 节一行（`281aba6`）。
   - spec 的状态和 M2 设计的进度表在本评审的提交中改为已完成。

## 6. 移交事项

| 交接 | 交给 | 内容 |
|---|---|---|
| [M0-P2-platform-notes](../handoffs/M0-P2-platform-notes.md) | M8 | P3b 处理了第 5 条：River、停机顺序、连接池关闭的时限、River 的迁移，另记第 17 条（River 启动后只用它自己的 `Stop`）；留下第 2 条的接口调用日志（M8） |
| [M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) | M2/P4，M4、M5、M8 | P3b 处理了 River 停机与 fixture 的预算（Task 11 实测、C1 调查）、`runNerve` 的标准输入、A14 的等待期限；留下页面的登录状态、S2 的断言（P4），fixture 写法的延伸（M4、M5、M8） |
| [M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) | M2/P4 | P3b 处理了修改登录邮箱（`nerve users set-email`，A16）；留下 Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（P4） |

三份交接都保持 `open`。

P3a 评审第 6 节交给 P3b 的事项，都已处理：
- **交错 1–3 真的争锁**：由 Task 10 关闭（闸门在持锁的事务里，`pgtest.WaitForLockWait`，锁的变异都失败）。
- **管理员撤销全部 PAT 时的 `updated_by_id`**：Task 6 写 NULL（spec 第 3 节第 5 条）。
- **`deactivateMe` 的说明**：Task 8 写上 `nerve users activate` 和"之后未过期的 PAT 重新可用"。
- **README 中 8.5 的恢复步骤和 A12 的端到端**：Task 12、Task 11。
- **契约文件不受约 400 行的限制**：写进 plan 的 Global Constraints（spec 第 3 节第 16 条）。

P2 评审第 6 节交来的"不可用的密码保持登录耗时"由 Task 9 关闭（第 2 节第 6 行）。

交给后续的事项：

| 交给 | 事项 |
|---|---|
| M2/P5 | A12 的页面版本（spec 第 5 节） |
| M4 | 命令行的"只投递"River 客户端，随第一个投递任务的命令加入（M2 设计 3.15、3.17、13.2 的 M4 一行；负责人 2026-09-26 批准推迟） |
| M8 | 容器的停止宽限期：默认配置下停机最坏约 36 秒，超过 Docker 默认的 10 秒，部署文件要设 `stop_grace_period` 或调小这几个期限（13.2 的 M8 一行，`777f1b1`） |
| 负责人 | 调高 argon2 参数之后，登录耗时能区分沉睡账户（P2 评审 I2，M2 设计 §16），仍待负责人决定 |

## 7. 已知限制

- **River 在启动后的最初几秒内停机时记 ERROR**（spec 第 6 节，附录 A 的 C1）：`maintenance.PeriodicJobEnqueuer: Error starting transaction`，River 自己还没启动完时还有 `notifier.Notifier: Error running listener … conn closed`。这两处 River 不看取消的原因，runner 无从避免。实测就绪后立即停机 10/10，3 秒后 7/40（改后的 runner 4/40，差别是噪声），运行 10 秒、30 秒后 0/40；都以 0 退出，没有任务停在 `running`，没有连接泄漏。README 部署一节说明。
- **停机落在 River 的重建索引中，又有长事务挡住删除**（spec 第 6 节）：River 每天 00:00 UTC 重建 `river_job` 的索引。停机打断重建、又有访问过 `river_job` 的长事务挡住没建完的 `_ccnew` 索引的删除时，任务的停止在 `jobs.shutdown_timeout` 加 1 秒（默认 11 秒）放弃，nerve 记 ERROR 并以退出码 1 退出。River 自己的删除另有 15 秒的时限，连接池关闭时会等它的连接；只有删除被挡满 15 秒，索引才留下（实验中观察到一次，15.0 秒退出）。恢复方法写在 README：用 `DROP INDEX CONCURRENTLY` 删掉 WARN 的 `artifact_names` 中列出的索引。改之前的 runner 在这样的停机中无论有没有长事务都留下索引。
- **`Stop` 恰好落在 River 的 `Start` 返回时**，River 收到的仍是普通的取消，这一次的重建索引清理照旧失去（spec 2.5）：要能打断进行中的 `Start`，就避不开这个窗口。
- **终端上的密码提示没有自动化测试**（spec 第 6 节）：要伪终端，得改 `go.mod`。代码只用 `term.IsTerminal` 和 `term.ReadPassword`，两次输入不同时失败；"接受不同的两次输入""提示写到标准输出"两个变异存活。spec 的应对是人工核对一次，控制者在合并前做了：在伪终端上运行 `nerve users`（`281aba6` 构建，开发库上的一个临时库，用后删除），每次等提示出现后才输入：
  - 两次不同时，两个提示都出现，`nerve: the passwords do not match`，退出码 1，没有建账户；
  - 两次相同时，`created user tty@example.com`，退出码 0；
  - `reset-password` 同样两次提示，退出码 0；
  - 三次运行中，终端上都没有出现输入的密码。

  标准输入的路径由单元测试和 A13、A17 覆盖。
- **停机时"迁移执行器先于连接池关闭"没有测试能看出来**（spec 第 6 节）：两步都只在进程退出前运行一次，迁移执行器的 `*sql.DB` 不在两次调用之间保留连接池的连接（`SetMaxIdleConns(0)`）。把两步对调的变异在原型和 Task 5 中都存活；顺序写在 `close` 的注释和代码里。
- **管理员命令在等锁之前读时钟**（第 4 节 M2）：撤销时刻可能比并发凭证的创建时刻早一个等锁的时间，`activate` 可能算上等锁期间过期的 PAT。没有安全上的影响。
- **`Runner.Stop` 先于 `Start` 调用时 panic**（第 3 节 Task 5）：唯一的调用者满足前提；以后调换顺序，启动 bootstrap 的测试会发现。
- **River 的 `Start` 因配置错误失败时，runner 当作数据库不可达一样无限重试**（Task 5 的观察）：重试的间隔最多 30 秒，每次一条 WARN，nerve 照常服务但没有任务。现在模块总是注册自己的 worker；以后的模块加定时任务而漏了 worker，唯一的迹象就是这条 WARN。
- **数据库恢复之后真实的 River 客户端能重新启动**：只有读代码（spec 附录 A 的 E1）和假客户端的重试测试；真实客户端只测了"不可达时失败、`Stop` 立即结束重试"。
- **argon2 调参之后的登录耗时**：同 P2 评审第 7 节，仍然成立。
