# M7 / P2a plan：收集箱条目的生命周期——提交、分诊与工作项的边界

> spec：[P2a spec](../specs/P2a-intake-submit-triage.md)（预检 [re-85](../../../reviews/re-85-M7P2a预检.md) 修正后通过）。基线 main `31f38580`。worktree `.codex/worktress/codex/m7-p2a`，分支 `codex/m7-p2a`。台账 `.superpowers/sdd/P2a-intake-submit/progress.md`（ignored），现状调研 `research/p2a-facts.md`。17 个 Task（T2、T12 各拆为两段，超过约 16 的上限一个，预检 m9 认可）。

## Global Constraints

- spec §2 是合同，§3 的 R1–R7 是裁定；超出 spec 的契约或架构改动先停下报告。
- 契约先行：改了 `api/` 就 `make gen`，`make gen-check` 无差异；每个声明的码有处理器测试回答。
- 模块之间只经端口；SQL 只经各模块自己的 sqlc，`TestSQLCSchemaScope` 不加例外（issue 的 SQL 不读 `states`）；夹具不进产品包；手写文件约 400 行以内。
- 编排：提交与分诊走 `Locks.run`，不在用例里写近似副本；新建核心只加一个放置参数，修改核心按 spec §2.7 抽出接缝。访客规则只有 `HiddenFrom` 一个谓词。一个请求一个时刻，在最后一把锁之后读。
- 守卫：删除与清理的目录守卫一律不加例外（设计 §4.3）。
- 允许改的已有测试（改之前在台账逐条写明，每处只照这里写的改法）：
  - 迁移：`issue_schema_test.go` **加**两条 verb 反例（已有的 `commented` 不改）、新表的名字；`purge_indexes_test.go`（33 → 34）；`trigram_schema_test.go`（迁移数、表清单、Down 起点）。
  - issue 模块里钉表序的：`adapter/postgres/deletion_remaining_test.go`、`delete_issue_test.go`、`delete_rows_test.go`、`purge_test.go`、`app/delete_rows_test.go`、`app/cascade_test.go`——表序、步骤表加 `intake_issues` 一项。
  - 删除与清理（bootstrap）：`purge_catalog_test.go`（33 → 34）、`purge_runtime_test.go` 与 `purge_scale_fixture_test.go`（加新表）、`issue_project_deletion_jobs_test.go` 的 `projectIssueTable`（显式加 `intake_issues`）、删除守卫的种子辅助（加条目行）。
  - 详情的形状：`platform/httpserver/apitest/issue_shapes_test.go` 的 `own` 加 `intake`；`issue/adapter/http/issues_test.go` 的金样加 `"intake":null`。
  - 权限矩阵与写锁表：新行、种子（开关打开、种一个在收集箱里的工作项与条目行）；不改已有行与断言（先核已有的 `check` 不数新种的行）。`access/domain/rules_test.go` 的 `tableCells` 加两行。
  - 处理器测试与契约测试里受新码影响的 `x-problem-codes` 清单。
  - 编译跟随：接口加方法、函数加参数带来的假实现与调用处（如 `create_core_test.go` 直接调 `createLocked`/`prepareLocked`、`NewLocks` 的构造处）只为编译而改，不改断言，台账逐个记。
  - 网页：`fake-issues.ts`、`issue.store.types.test.ts` 的 `IssueDetail` 字面量加 `intake: null`；`issue.store.ts` 的合并保留 `intake`，加一条与 `keeps_is_subscribed_when_a_list_row_arrives` 同形的测试。
  - e2e：`fixtures/assert/catalog.ts` 的 `asynchronousDeletionTable`；`fixtures/furnish.ts` 的 `furnishPlanningRows`（加一次提交）与新夹具 `furnishAcceptedIntake`；`stories/issue/i15-archive-issue.spec.ts`、`i17-delete-issue.spec.ts`、`i21-purge.spec.ts` 只加“调一次 `furnishAcceptedIntake`”这一步（逐处确认故事里有管理员的 token）。
  - 生成物（Go 的 `gen/`、网页的 `schema.gen.ts`）；`error-messages.ts` 与两份 `errors.json`。

  其余 M4–M6、P1a–P1c 的断言是回归，不改。
- 红的窗口（其间这几项不作为证据，台账记下）：e2e 的删除断言 T2b → T10；路由、逐码与写锁表的守卫 T8 → T9 / T11；权限矩阵的覆盖守卫 T8 → T11。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`，失败的运行也留着；每个可见、迁移、守卫、锁的性质至少一个能让测试失败的变异。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒（本机约 60 秒）；两个检查点记本机与分支 CI 两路。本机两路因 M6-M8-05 失败且重跑一次仍不过时，按负责人 2026-11-18 的决定以分支 CI 为准并记录。

## Tasks

### T1 基线与文档（主控）
P1c 阶段记录 §9 回填集成、设计 §14 的 P1c 一行；spec、plan、预检 re-85；设计 §12 的 P2a、P2b 两行按 R1 改、§14 加 P2a 进行中；`tools/keywords.json` 的 phase 改 `M7/P2`（`node tools/keywords.mjs` 通过，不延长任何 `until`）。

### T2a 迁移
`00050`、`00051`（spec §2.2，含 Down）；sqlc、运行角色；迁移测试（约束、索引名、CHECK 的反例、verb 的两条新反例、`purge_indexes`、`trigram_schema`）。变异：CHECK 的 `IS NOT DISTINCT FROM` 改回 `=`（`field` 为 NULL 的反例要杀它）、去掉部分唯一。

### T2b 删除图与清理
issue 的清理步骤（`issues` 之前）、工作项删除的同事务软删、项目与工作区的分批（spec §2.10）；issue 模块里钉表序的测试与 bootstrap 的删除、清理守卫以 SQL 种条目行转绿。变异：漏软删、清理次序在 `issues` 之后、分批漏表。

### T3 领域与动作
状态与迁移表、有效状态、分诊请求的校验（`snoozed_till`、`duplicate_to_id` 的必填与不许、`status` 的值）、`IntakeFacts` 与 `InTriage()`；`intake.submit`、`intake.triage` 的动作声明与规则表、`rules_test` 两行（同一提交，免得动作守卫红）。表驱动单元测试，每条规则一个变异。

### T4 存储
条目行的插入与更新（按 `issue_id`，不另锁）；`GetIssue`、`GetIssues`、`SubIssues` 同步左连接（spec §2.3）；搜索的排除；project 的两条锁读带 `intake_view`（bootstrap 的类型转换随之）。adapter 测试；变异：漏 `ii.deleted_at`、`accepted` 也算在收集箱里、搜索不排除。

### T5 提交
新建核心的放置参数（Triage 状态）；提交用例（spec §2.4 的次序，开关在锁下、归档先于开关，插条目行后读回）；单元测试钉住次序与核心的复用（草稿发布与普通新建照旧取默认状态）。变异：开关不看、归档在开关之后、状态取默认。

### T6 分诊与修改核心的接缝
抽出 `updateLocked`（spec §2.7，`Execute` 的行为不变，已有的修改测试照旧过）；分诊用例（spec §2.5）：拒绝次序、迁移、字段、重复目标（锁下读，同项目的授予与 `HiddenFrom`）、`intake` 动态在状态动态之前、接受经接缝改到默认状态、一个时刻。单元测试与变异：终态可改、重复目标看不到也收、接受不发事件、接受读了第二次时钟、动态记存的状态而不是有效状态。

### T7 工作项的边界
选择性的 Triage 守卫加到六个源端写（spec §2.6，`unarchiveIssue` 的守卫先于空操作）；`inReach` 与关联的另一端拒绝在收集箱里的工作项；搜索的源 404。单元测试；变异：每个源端去掉守卫、另一端放行、链接被误拦、守卫排到归档之后。

### T8 契约、网页的码与类型
两个操作、`IssueDetail.intake`、四个新码、六个已有操作加 `issue.in_triage`、通知契约的说明；`make gen`；网页的码、两个 `IssueDetail` 夹具与 `issue.store.ts` 的合并及其测试。红的窗口从这里开始。

### T9 处理器、接线与详情的 `intake`
两个处理器；`Locks` 加时钟；`seenOne` 带有效状态与重复目标的可见（spec §2.8）；bootstrap 接线；逐码的处理器测试；详情形状的两处已有测试；“七个回答都带 `intake`”照 `TestEachDetailAnswerSaysWhetherTheCallerSubscribes` 写。路由与逐码的守卫转绿。

### T10 e2e 的删除夹具
`asynchronousDeletionTable` 加 `intake_issues`；`furnishAcceptedIntake` 与五处调用；`furnishPlanningRows` 加一次提交（spec §2.10）。i15、i17、i21 与项目、工作区的删除故事转绿。

### T11 权限矩阵与写锁表
两个新操作的行（开关打开）；六个源端守卫在矩阵里至少各一行（种一个在收集箱里的工作项）；写锁表两行；覆盖守卫与写锁表守卫转绿。

### T12a 组合测试：提交、详情与不变式
H8；详情的 `intake`（七个操作、有效状态、重复目标的可见）；不变式（spec §2.3）。

### T12b 组合测试：分诊与边界 → **检查点 1**
H10（含拒绝次序与两种 404 逐值相同）；H11 的边界。**检查点 1**：七项本地门禁、分支 CI、两路时间。

### T13 交错
两次并发的分诊、分诊与开关的关闭、接受与默认状态的改设、分诊与项目删除；语句名的锁探针。变异：分诊不在项目锁下、迁移不在锁下复核。

### T14 e2e 故事 → **检查点 2**
H8、H10 的接口故事（H1 的写法，等分发用 P1c 的 `notificationsSent`）。**检查点 2**：七项本地门禁、两路时间。

### T15 文档与交接（主控）
plane-diff（提交、状态与迁移、接受写动态、重复目标的校验与已归档、删除图、描述版本、只有应用内来源、平的请求体与独立的分诊端点、回答的形状与 201、状态名、Plane 的提交与修改不看开关、Plane 现建 Triage 状态、Plane 接受时缺默认状态答 400）；硬规则 03 §4（选择性的 Triage 守卫与它先于归档的次序、条目行是权威标记）；M3 交接 §1.2、M4 交接 M7-01 的处理结果；设计 §14；阶段记录（向负责人说明 R1 与 P2b 之前 main 的状态）。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 新建与修改核心的复用（没有近似副本）；接受经接缝、一个时刻。
- 条目行与 Triage 状态的等价；有效状态在迁移、回答、守卫里一致。
- 六个源端守卫与另一端的拒绝；链接、评论等照常；搜索的排除与源。
- 不透露：两种 404 与随机 id 逐值相同；重复目标的不合格同一回答；详情里看不到的重复目标为 `null`；分诊动态不指名目标。
- 删除图与清理不加例外；e2e 的夹具只经接口。
- 测试时间的预算（两路各自）。
