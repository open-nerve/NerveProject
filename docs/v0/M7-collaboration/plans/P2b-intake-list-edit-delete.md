# M7 / P2b plan：收集箱的列表、内容修改、删除与待处理数

> spec：[P2b spec](../specs/P2b-intake-list-edit-delete.md)（预检 [re-86](../../../reviews/re-86-M7P2b预检.md) 修正后通过）。基线 main `670399c4`（合入 M5 测试竞态的修复后前移，见台账）。worktree `.codex/worktress/codex/m7-p2b`，分支 `codex/m7-p2b`。台账 `.superpowers/sdd/P2b-intake-list/progress.md`（ignored），现状调研 `research/p2b-facts.md`。16 个 Task。

## Global Constraints

- spec §2 是合同，§3 的 R1–R12 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；每个声明的码有处理器测试回答（spec §2.7 逐操作的码）。
- 模块之间只经端口；SQL 只经各模块自己的 sqlc（issue 的 SQL 不读 `states`）；夹具不进产品包；手写文件约 400 行以内。
- 编排：写走 `Locks.run`，读走已有的读编排，不写近似副本；修改与删除经抽出的核心。访客规则在 Go 里只有 `HiddenFrom`；列表与计数的 SQL 用 M4 列表的登记写法；列表行的裁剪复用 `hideParents`、`projectAttachments`。一个请求一个时刻，在最后一把锁之后（读在只读事务里）读。
- 守卫：删除与清理的目录守卫一律不加例外。按目录驱动的守卫随契约同一 Task 补齐（spec §2.8）。
- 允许改的已有测试（改之前在台账逐条写明，只照这里写的改法）：
  - **编译跟随**：纯搬移（T2、T7 前半）与接口加方法（T5–T8）带来的构造处与假实现（如 `fakes_intake_test.go`、`bootstrap/issue_list_directories_test.go`、各处 `UpdateIssue`/`DeleteIssue` 的构造），只为编译而改，断言不改，台账逐个记。
  - m2：`submit_intake_test.go` 期望的调用清单少了核心回答的读（m2 的 RED），`create_core_test.go` 的调用签名。
  - n1：`triage_intake_test.go` 加新格。
  - 权限矩阵与写锁表：新行与种子；不改已有行与断言。`access/domain/rules_test.go` 的 `tableCells`、`creatorCells` 加行。
  - 删除窗口：`issue_deletion_window_cases_test.go`（计数与新行、`request()` 的条目分支）、`issue_deletion_window_test.go`（`getIntakeCounts` 在项目删除时 200 与它的 `check`）、`issue_deletion_window_fixture_test.go`（`f.q` 开收集箱、提交一个条目）。
  - 空创建者：`issue_null_creator_test.go` 的跳过清单与泄漏的目录。
  - 详情的回答：`issue_intake_answers_test.go` 加修改为第八个。
  - 处理器测试与契约测试里受新码影响的 `x-problem-codes` 清单；生成物；`error-messages.ts` 与两份 `errors.json`。

  其余 M4–M7 的断言是回归，不改。
- 红的窗口（台账记下）：契约与处理器在同一 Task（生成的服务端接口要求处理器，P2a 的教训）；写锁表、矩阵覆盖、删除窗口与空创建者的守卫 T9 → T10。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`；下表每个性质至少一个能让测试失败的变异。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒；两路各自的增量与新测试自己的耗时分开记（按名字轮流分路，新测试名会让后面的换路；某一路因换路超过时照设计 §9 报告，不靠改名调路）。T11、T12 里删除之后断言行与墓碑的测试、T12 的每个交错先 `waitForStartupRuns`；修改与删除之后做锁检查的先 `waitForFanouts`（两者都发事件，硬规则 03 §5）。本机两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 为准并记录。

## 性质与变异

| 性质 | 变异 | Task |
|---|---|---|
| 有效状态（到期的暂缓算待处理） | SQL 里只看存的状态 | T5a、T11a |
| 列表的只看自己 | 去掉 `own_only` 的谓词 | T5a、T11a |
| 列表行的裁剪 | 去掉 `hideParents`；子数去掉谓词 | T6、T11a |
| 重复目标看不到为 `null` | 列表行不经 `HiddenFrom` | T6、T11a |
| 列表与回答同一个时刻 | 读第二次时钟 | T6 |
| 日的半开区间 | 区间改闭、用 `::date` | T5a |
| 计数的范围 | 关着的、归档的被列出 | T5b、T11a |
| 只看自己的计数 | 去掉谓词 | T5b、T11a |
| 修改的相对规则、访客字段 | 非提交者放行；访客字段放行 | T8 |
| 关闭的 409 | 已接受可改 | T8 |
| 开关在锁下再判 | 只在预检判 | T8、T12 |
| 已接受只删条目行 | 已接受删了工作项；未接受只删了行 | T7 |
| 源端守卫仍在 | `updateIssue` 的守卫被拆 | T8 |
| n1 | 去掉检查 | T3 |
| 修改、删除在锁下复核 | 状态与条目行只在预检判 | T12 |

## Tasks

### T1 基线与文档（主控）
P2a 阶段记录 §9 回填集成、设计 §14 的 P2a 一行为已完成；spec、plan、预检 re-86；设计 §12 的 P2b 一行与新的 P2c 一行（含 H11 的图片证据）、§14 加 P2b 进行中；设计 §3.5 的计数“未归档”、§3.10 的规则表加 `intake.count`。

### T2 修改核心的类型（纯搬移，m3）
`IssueUpdate` 自己的类型（照 `IssueCreation`），`UpdateIssue` 与分诊改为持有它；断言不改。

### T3 m2 与 n1
`createLocked` 交回读回的工作项与时刻，`CreateIssue` 自己组回答；`acceptance` 的“默认即当前”内部错误。单测与变异。

### T4 领域与动作
`ParseIntakeQuery`（CSV 的 `status`、日的半开 UTC 区间、`none` 422）、`IntakeCursor`；`IntakeIssueUpdate` 的校验（访客的字段）；关闭的状态与 `issue.intake_closed`；四个动作的声明与规则、`rules_test` 的行（同一提交）。表驱动单测与变异。

### T5a 存储：列表
列表 SQL（筛选、有效状态、访客规则、键集、`total`、子数只数看得到的）。adapter 测试与变异。

### T5b 存储：计数、条目行的软删、项目端口
计数 SQL；条目行的软删；project 的“哪些开着收集箱”端口与 SQL、bootstrap 转换。adapter 测试与变异。

### T6 列表与计数的用例
`ListIntakeIssues`（只读事务、时刻一次、`hideParents`、`projectAttachments`、重复目标按页批量读与 `HiddenFrom`）、`CountIntakeIssues`（`Visible` → 开关 → 计数，一个只读事务）。单测与变异。

### T7 删除核心（纯搬移）与删除
先把 `DeleteIssue` 锁下的一段抽成删除核心（纯搬移、断言不改），再加 `DeleteIntakeIssue`（四种状态删工作项、已接受只删条目行，含已归档的）。单测与变异。

### T8 内容修改
`UpdateIntakeIssue`：相对规则与访客字段在 `write.check`、关闭的 409、开关与归档、经 `IssueUpdate`。单测与变异。

### T9 契约、处理器与网页
四个操作、四个类型、新码（spec §2.7 逐操作的码）；`make gen`；四个处理器、bootstrap 接线、逐码的处理器测试；网页的码文案。写锁表、矩阵覆盖、删除窗口与空创建者的守卫在此转红，台账记下。

### T10 权限矩阵、写锁表、删除窗口、空创建者
spec §2.8 的全部守卫转绿：矩阵行（列表 `write: true` 与 `prepare`）、写锁表两行、删除窗口的两读两写（期望、夹具、条目分支与控制变异）、空创建者的跳过与泄漏目录、`creatorCells`。

### T11a 组合测试：列表与计数
H9 的列表（筛选、次序、游标、`total`、有效状态、日的边界、只看自己、行的裁剪、重复目标、列表行与详情的 `intake` 相同）与计数。

### T11b 组合测试：修改、删除与 n1 → **检查点 1**
内容修改与删除（spec §4）、交叉的拒绝次序四格、修改为第八个详情回答。**检查点 1**：七项本地门禁、分支 CI、两路各自的增量。

### T12 交错
内容修改与分诊、删除与分诊、计数与提交；语句名的锁探针；每个先 `waitForStartupRuns`。变异：修改不在锁下复核状态、删除不在锁下复核条目行。

### T13 e2e 故事 → **检查点 2**
H9 的接口故事（列表、筛选、计数、修改、删除，PAT 与会话对等）。**检查点 2**：七项本地门禁、两路各自的增量。

### T14 文档与交接（主控）
plane-diff（列表的默认状态与词、日按 UTC、行的形状、修改的字段与访客 403、关闭 409、删除的时段与已接受只删条目行、计数的形状、动作与归档的项目不列）；硬规则；设计 §14；阶段记录（向负责人说明 R1 与 P2c 之前 main 的状态）。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 两次纯搬移确为纯搬移；第三个调用者不复制编排。
- 有效状态在 SQL、Go 与回答里一致；访客规则的 SQL 写法与 `HiddenFrom` 一致（只看自己、空创建者）；列表行的裁剪与项目列表相同。
- 内容修改的相对规则与访客字段、关闭的 409、两种 404 逐值相同、交叉的拒绝次序；源端守卫仍在。
- 删除：四种状态删工作项、已接受只删条目行，连带与 `deleteIssue` 相同。
- 计数：范围、到期的暂缓、只看自己、不另写成员关系。
- 按目录驱动的守卫全部补齐；测试时间的预算。
