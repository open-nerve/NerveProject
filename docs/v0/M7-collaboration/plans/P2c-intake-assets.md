# M7 / P2c plan：Triage 宿主的资产判定

> spec：[P2c spec](../specs/P2c-intake-assets.md)（预检 [re-87](../../../reviews/re-87-M7P2c预检.md) 修正后通过）。基线 main `f25cccad`。worktree `.codex/worktress/codex/m7-p2c`，分支 `codex/m7-p2c`。台账 `.superpowers/sdd/P2c-intake-assets/progress.md`（ignored），现状调研 `research/p2c-facts.md`。10 个 Task。

## Global Constraints

- spec §2 是合同，§3 的 R1–R6 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；声明的码有处理器测试回答，契约与处理器测试同一 Task。
- 模块之间只经端口；资产模块的 app 层不改；SQL 只经各模块自己的 sqlc；夹具不进产品包；手写文件约 400 行以内。
- 编排：资产写照旧走 `Access.Within`；issue 一侧的判定经 `Locks` 的变体，不写近似副本；提交者规则与 P2b 的内容修改共用；锁下复用 `refuseIntakeOff` 与 `CheckOpen`。
- 允许改的已有测试（改之前在台账逐条写明，只照这里写的改法）：
  - **编译跟随**：纯搬移（T2）与资产项目读的加宽（T3）带来的构造处与假实现（调研 §7 列的两个假实现与三处断言），断言的意思不变。
  - 资产处理器的 `business_errors_test.go` 加两行；契约测试里受新码影响的清单；生成物。
  - 宿主矩阵（T6）：在已有的矩阵测试里加“在收集箱里的工作项”的宿主与它的世界（开收集箱、提交者与非提交者的账号；放在归档的子测试之前），不改已有的行与断言。
  - 交错（T8）：`TestAssetLateHostInterleaving` 的封口表加三行（拒绝、关收集箱、接受）；真实锁等待的一格照 `assets_membership_interleaving_test.go` 的做法，写在已有的资产交错测试里作子测试。
  - 权限矩阵：`createAsset` 的一行与种子。

  其余 M4–M7 的断言是回归，不改。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`；下表每个性质至少一个能让测试失败的变异。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒，两路各自记。新的 bootstrap 顶层测试名只有 T7 的一个；T6、T8 是已有测试的子测试或表行。T6 提交后按 `tools/go-test-shard.sh` 的算法（`go test -list` 排序后按奇偶）算出 T7 新名字让哪些测试换路、记入台账，检查点 1 对照；不靠改名调路。资产处理器的测试在第 0 路。造好数据后扣锁的先 `waitForStartupRuns`，锁检查前 `waitForFanouts`。本机两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 为准并记录。

## 性质与变异

| 性质 | 变异 | Task |
|---|---|---|
| 在收集箱里的宿主按 `intake.update` | 照旧用 `issue.update` | T5、T6 |
| 非管理员须是提交者 | 非提交者放行 | T5、T6、T7 |
| 开关在锁下判 | 不判；判到评论宿主；挪进 `Inspect` | T5、T8 |
| 关闭的宿主 409 | 不判；挪进 `Inspect` | T5、T7、T8 |
| 归档先于关着、关着先于关闭 | 次序对调 | T5 |
| 403 先于 409 | 时段与开关先于提交者规则 | T5 |
| 已接受的照 `issue.update` | 已接受也按 `intake.update` | T5、T8 |
| 只看自己的访客对别人的条目 404 | 选了 `intake.update` 就先判角色 | T5 |
| `notFound` 仍是 `ErrIssueNotFound` | 换成 `ErrIntakeNotFound` | T5 |
| 读不变 | 读也按 `intake.update` | T5、T7 |
| 开关与归档同一次读 | 开关恒开；转换丢开关 | T3 |

## Tasks

### T1 基线与文档（主控）
spec、plan、预检 re-87；P2b 阶段记录 §9 回填集成、设计 §14 的 P2b 一行为已完成、加 P2c 进行中。

### T2 两次纯搬移
`Locks` 的变体（读一次、按工作项选动作、再判定），`seeAs` 委托给它；从 `intakeEditor` 抽出共用的提交者规则。断言不改。

### T3 资产的项目读带开关
SQL 与端口加宽为具名结构，bootstrap 转换与它的测试；编译跟随。adapter 测试与变异。

### T4 契约与处理器测试
`asset.yaml` 四个写的码与说明（含重复删除、重复完成）；`make gen`；`business_errors_test.go` 两行。

### T5 issue 一侧的判定
资产宿主按 `InTriage` 选动作；共用的提交者规则；`Check` 里只对在收集箱里的工作项宿主复用 `refuseIntakeOff` 与 `CheckOpen`；Go 文档注释加宽。单测（含交叉格）与变异。

### T6 宿主矩阵
真实宿主矩阵加“在收集箱里的工作项”（四个写 × 角色与提交者）。算好 T7 新名字的换路清单。

### T7 HTTP 组合测试（H11）→ **检查点 1**
一个新的顶层测试：访客提交者上传描述图片并经 `updateIntakeIssue` 绑定、加附件；非提交者 403；拒绝后 409；关掉收集箱 409；接受后照 `issue.update`；到期的暂缓可写；读不变；评论宿主不受影响。**检查点 1**：七项本地门禁、分支 CI、两路各自的增量。

### T8 交错与权限矩阵的一行
封口表三行；一格真实地在 `LockIssueProjects` 上等待（语句名的锁探针）；`createAsset` 以收集箱里的工作项为宿主的矩阵行（带 `check`）。

### T9 e2e 故事 → **检查点 2**
H11 的资产故事（PAT 与会话对等），写明落在哪个 e2e 文件与哪一片。**检查点 2**：七项本地门禁、两路各自的增量、e2e 两片的时间。

### T10 文档与交接（主控）
plane-diff（资产在收集箱里的工作项上的判定、删除已绑定附件的人）；硬规则；设计 §3.6（`:109` 前半句“附件按它们自己的规则照常可写”与后半句矛盾，改写）、§3.11、§14；M5-M7-01 的处理结果；阶段记录（§1 的四条收窄向负责人说明）。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 纯搬移确为纯搬移；资产模块的 app 层没动。
- 在收集箱里的宿主的判定、开关与时段只对它判，评论宿主与已接受的照旧。
- 提交者规则只有一份；拒绝次序与码与 spec §2.2 一致。
- 测试时间的预算与换路。
