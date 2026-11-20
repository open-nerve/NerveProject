# M7 / P3a plan：工作区跨项目列表

> spec：[P3a spec](../specs/P3a-workspace-list.md)（预检 [re-88](../../../reviews/re-88-M7P3a预检.md) 修正后通过）。基线 main `7dfaa8a1`。worktree `.codex/worktress/codex/m7-p3a`，分支 `codex/m7-p3a`。台账 `.superpowers/sdd/P3a-views/progress.md`（ignored），现状调研 `research/p3a-facts.md`。9 个 Task。

## Global Constraints

- spec §2 是合同，§3 的 R1–R7 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；每个声明的码有处理器测试回答；契约与处理器同一 Task。
- 模块之间只经端口；SQL 只经 issue 自己的 sqlc；不写列表引擎的近似副本（工作区列表 = 成员列表去掉目标成员与 `relation`）；手写文件约 400 行以内。
- 允许改的已有测试（改之前在台账逐条写明，只照这里写的改法）：
  - 词表的表驱动单测：`WorkspaceList` 另起一张表（`listvocab_test.go` 里范围是列，不能“只加行”）；解析的单测加新范围的格。
  - `access/domain/rules_test.go` 的 `tableCells` 一行。
  - `adapter/http/list_query_test.go` 里列举两张按操作清单的期望（`:24`、`:41`）。
  - 权限矩阵一行；删除窗口：`issue_deletion_window_cases_test.go`（读 20 → 21）、`issue_deletion_window_test.go`（项目删除时 200 的工作区读加新操作与一个 `check`）；空创建者：`issue_null_creator_test.go` 的跳过清单与泄漏目录（T5 写明四处的确切位置）。
  - `TestIssueListScale` 与它的 oracle：加工作区场景、路由的第三种、子种子（不新增顶层名）。
  - 生成物。

  其余断言是回归，不改。
- 红的窗口（台账记下）：T4 加了操作，`TestThePermissionMatrixCoversEveryOperation` 与 `TestDeletionWindowCoversEveryIssueOperation` 红到 T5；T4 验证处理器包、契约、网页与非 bootstrap 包，这两个守卫在 T5 转绿。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`；下表每个性质至少一个能让测试失败的变异。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒，两路各自记；新的 bootstrap 顶层名只有 T6 的一个（合并 main 后分路改为按名字的哈希，下面的换路清单作废，见阶段记录 §4；当时的轮流分：以 `TestWorkspaceIssueList` 开头，第 764 位、落第 1 路，换 7 个：第 1 → 0 路 `TestWorkspaceProfilesConvertsIdentitysAnswer`、`TestWorkspacesCreateErrors`、`TestWorkspacesReactivateMemberErrors`、`TestWritesOnTwoProjectsOfAWorkspaceDoNotWait`；第 0 → 1 路 `TestWorkspacesCreate`、`TestWorkspacesReactivateMember`、`TestWritesOnAProjectAreEachShape`），检查点对照。造好数据后扣锁或写墓碑的测试先 `waitForStartupRuns`。本机两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 为准并记录。

## 性质与变异

| 性质 | 变异 | Task |
|---|---|---|
| 不收 `cycle_id`、`module_id`、`archived`、`relation` | 任一被收 | T2 |
| 只平铺 | `group_by`/`sub_group_by`/`group_key`/`sub_group_key` 任一被收 | T2 |
| 解析先于授权 | 解析挪到判定之后（交叉格） | T3、T6 |
| 范围是有效成员的未归档项目 | `Visible` 换成同接口的 `Historical`；含已归档项目 | T3、T6 |
| 动作 | 用 `member_issue.list` | T3 |
| 两张按操作的清单 | `withListQuery` 漏新操作（答 200 不带筛选）；`listSingleQuery` 漏新操作 | T4 |
| 网页客户端的 CSV | `csvListPaths` 漏新路由 | T4 |
| 逐项目只看自己 | 去掉 `OwnOnly`（游标翻页也换出看不到的行） | T6 |
| 只读快照 | 不在只读快照里读 | T6 |
| 守卫 | 去掉矩阵行、删除窗口行 | T5 |

## Tasks

### T1 基线与文档（主控）
spec、plan、预检 re-88；关键词 phase `M7/P3`；设计 §12 的再划分（P3a 工作区列表、P3b 视图、P3c 收藏与最近访问；P6b 一行加工作区标签目录；`:277` 的“P3b”改 P3c）、§14；P2c 阶段记录 §9 回填集成。

### T2 领域：`WorkspaceList`
`ListScope` 加 `WorkspaceList`；筛选词（spec §2.2）、只平铺的四个词、不收 `relation`；`ParseListQuery` 认它。另起的表驱动单测与变异。

### T3 用例与动作
`ListWorkspaceIssues`（只读快照、`Visible`、`Listing.Read` 不带目标成员）；动作 `workspace_issue.list` 与规则、`rules_test` 一行。单测与变异（接线随 T4 的处理器）。

### T4 契约、处理器与网页客户端
`listWorkspaceIssues` 的契约（码、恒为平铺的说明）；`make gen`；处理器与接线、两张按操作的清单、逐码的处理器测试；`csvListPaths` 加新路由；`make test-web`。变异见表。矩阵覆盖与删除窗口覆盖两个守卫在此转红（台账记下）。

### T5 守卫
权限矩阵一行；删除窗口读 20 → 21（项目删除时 200、不含被删项目的工作项）；空创建者的跳过与泄漏目录；两个覆盖守卫转绿。

### T6 组合测试 H13 → **检查点 1**
一个新顶层名的组合测试（spec §4 的各格，含交叉格、不是成员的项目、只读快照、末页之后的空页），种子写明（多项目、不同角色与只看自己、收集箱、归档、离开的项目）。变异见表。**检查点 1**：七项本地门禁、分支 CI、两路各自的增量与换路清单对照。

### T7 规模
`TestIssueListScale` 下的工作区场景、路由的第三种与多项目子种子；本机 `NERVE_LIST_SCALE=1` 的证据、EXPLAIN 与通过线的判定（spec §2.4）；先重估运行时间。不达标先查明，要靠索引或特化 SQL 时停下上报。

### T8 e2e 故事 → **检查点 2**
H13 的接口故事 `e2e/stories/collaboration/h13-workspace-list.spec.ts`（四个默认视图、逐项目只看自己、跨项目翻页，PAT 与会话对等）。**检查点 2**：七项本地门禁、两路各自的增量、e2e 两片的时间。

### T9 文档与交接（主控）
plane-diff（工作区列表的词与只平铺、范围、默认视图）；三个交接各写部分的处理结果（M2 §1：工作区列表的游标部分，视图列表与跨模块联表留 P3b；M4 M7-03：工作区列表一支，视图留 P3b、页面与标签目录留 P6b；M6-M7-03：设计 D4 已关，工作区列表不收迭代、模块的筛选与分组词，次序词照收）；硬规则；设计 §3.7、§14；阶段记录。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 工作区列表不复制列表引擎；范围、只看自己、排除与成员列表一致；两张按操作的清单都有新操作。
- 词表与只平铺；游标不能换出看不到的行；只读快照。
- 规模的证据与通过线；测试时间与换路。
