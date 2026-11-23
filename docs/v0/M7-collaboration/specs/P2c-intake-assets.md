# M7 / P2c：Triage 宿主的资产判定

> 状态：spec 经预检 [re-87](../../../reviews/re-87-M7P2c预检.md) 修正后通过。基线 main `f25cccad`（P2b 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H11）、§3.6、§3.10、§3.11、§15；P2b 见 [P2b spec](P2b-intake-list-edit-delete.md) §1（再划分与 P2c 之前 main 的状态）与 [P2b 阶段记录](../reviews/P2b-intake-list-edit-delete-review.md)；M5 交接 [M5-M7-01](../handoffs/M5-closeout.md#M5-M7-01)。现状调研见本阶段台账 `research/p2c-facts.md`（ignored，随阶段归档；下文 DP-n 指它的决定点）。

## 1. 目标与边界

设计 §3.6：资产（描述图片、附件）的写对 Triage 宿主按 `intake.update` 判定（管理员与提交者），不按 `issue.update`。P2b 之后 main 照旧按 `issue.update`。P2c 把这一句做完，并补 H11 里“描述的图片同此”的证据。

- 做：资产的四个写（`createAsset`、`completeAsset`、`duplicateAsset` 的目标、`deleteAsset`）在宿主是“在收集箱里的工作项”时的判定、拒绝次序与码；`asset.yaml` 的码与说明；它们的单测、宿主矩阵、组合测试、交错、权限矩阵的一行与 e2e 故事。
- 不做：资产的读（照旧 `issue.read`，与 `intake.read` 同样的角色与访客规则，DP-7；`duplicateAsset` 的来源、`getAsset`、内容、附件列表都不变）；评论宿主的资产（不变）；网页（P5）。
- 没有新操作，按操作清单检查的守卫（写锁表、删除窗口、空创建者、`rules_test`）不变。

**这收窄了 main 今天的行为（R1，设计已批准，阶段记录向负责人逐条说明）**，只对在收集箱里的工作项（待处理、暂缓、拒绝、重复）的宿主：

1. 访客提交者从 403 变为可以给自己的条目加图片与附件；
2. 不是提交者的成员从可以变为 403；
3. 拒绝、重复的条目上四个写对所有人（含管理员）都答 409 `issue.intake_closed`——已绑定的附件只能随删除整个条目一起去掉；
4. 收集箱关着时四个写对所有人都答 409 `issue.intake_disabled`。

另：接受之后照 `issue.update`，访客提交者不能再删自己没绑定的预备资产（403），同内容修改在接受之后的规则。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/issue/app/` | 资产宿主的判定（`asset_access.go`）：先读工作项、按 `InTriage` 选动作；提交者规则与内容修改共用；锁下复用 `refuseIntakeOff` 与 `CheckOpen` |
| `server/internal/modules/issue/app/lock*.go` | `Locks` 的“读一次、按工作项选动作、再判定”的变体，`seeAs` 委托给它（纯搬移） |
| `server/internal/modules/project/` | 资产的项目读同时交回 `intake_view`（DP-4） |
| `server/internal/bootstrap/` | 端口转换与它的测试；宿主矩阵、组合测试、交错、权限矩阵的一行 |
| `api/modules/asset.yaml`、生成物 | 四个写的 `x-problem-codes` 加 `issue.intake_disabled`、`issue.intake_closed`，说明各一句 |
| `server/internal/modules/asset/adapter/http/` | 逐码的处理器测试（`business_errors_test.go` 两行） |
| `e2e/` | H11 的资产故事 |
| `docs/` | plane-diff、硬规则、设计 §3.6、§3.11、§14、M5-M7-01 的处理结果、阶段记录 |

资产模块的 app 层不改（宿主的判定在 issue 模块，经 bootstrap 的端口，DP-2）。

### 2.2 判定（DP-1、DP-3、DP-5、DP-6）

宿主是工作项（`AssetIssue`）、且工作项在收集箱里（`InTriage()`：有未删除的条目行、存的状态不是 `accepted`）时：

- 动作 `intake.update`（项目的管理员、成员、访客）加提交者规则：不是管理员（`Grant.ProjectAdmin`）的须是工作项的创建者（`Issue.CreatedBy`），否则 403 `forbidden`；与 P2b 内容修改同一个规则，从 `intakeEditor` 抽成共用的函数（纯搬移），不复制。访客字段的规则不适用（资产没有字段之分）。
- 锁下（`Check`）另判，复用 P2a/P2b 的 `refuseIntakeOff`（项目归档 409 `project.archived` → 收集箱关着 409 `issue.intake_disabled`）与 `IntakeFacts.CheckOpen`（拒绝、重复 409 `issue.intake_closed`；只看存的状态，不读时钟，到期的暂缓仍可写）。只对这种宿主判，评论宿主与不在收集箱里的工作项照旧；时段与开关不进 `Inspect`。
- 不在收集箱里的工作项（含已接受的）照旧 `issue.update`。

拒绝次序：

- `Inspect`（不加锁）：宿主不在或调用者看不到 404 `asset.not_found`（只看自己的访客对别人的条目，与随机 id 逐值相同）→ 角色不许 403 → 不是提交者 403。
- `Check`（锁下）：上面三项再判 → `asset.scope_changed` → 项目不在 404 → 409 `project.archived` → 409 `issue.intake_disabled` → 409 `issue.intake_closed` → 409 `issue.archived`。

码：只用 `forbidden`、`issue.intake_disabled`、`issue.intake_closed`；不答 `issue.in_triage`、`issue.intake_not_found`（资产路径的 `notFound` 仍是 `ErrIssueNotFound`，`AssetHostNotFound` 不含条目的 404）。两个码的错误文字不变（只加宽 Go 的文档注释，T5）。

### 2.3 读一次、按工作项选动作（DP-2）

今天 `read` 在读工作项之前就定了 `issue.update`（`issue/app/asset_access.go:106-109`），然后 `Locks.seeAs` 读工作项并判定。给 `Locks` 加一个变体：读一次工作项，由调用者按工作项选动作，再判定；`seeAs` 委托给它（4 处调用，`NewLocks` 的签名不变）。纯搬移单独提交（断言不改）。先选动作再判定不泄漏：两种动作都先过访客规则的 404。

### 2.4 开关的来源（DP-4）

资产的项目读（今天的 `AssetProjectArchived`，只有资产这一个调用者，只交回归档与是否存在）加宽为一次读交回一个具名的结构（如 `AssetProject{Archived, IntakeOn}` 与是否存在），同一条 SQL、同一个快照；不用多个按位置的布尔值，不另调 P2b 的 `IntakeOn`。端口形状变了，两个假实现与三处断言跟随（台账逐个记）；bootstrap 的转换加一个测试（开关不被丢）。

### 2.5 删除与已关闭的宿主（DP-8、DP-9）

- 已关闭（拒绝、重复）或收集箱关着的宿主上，删除自己没绑定的预备资产、重复的完成同样 409，没有例外（与今天已归档的项目相同）。
- 删除已绑定的附件照 M5 的规则：宿主允许写的人都能删（所以访客提交者能删管理员在他的条目上传的附件）。Plane 只许上传者与管理员，plane-diff 登记。

### 2.6 契约与文字（DP-10）

- `asset.yaml` 的四个写各加 `issue.intake_disabled`、`issue.intake_closed`（`forbidden` 已声明），说明各加一句“宿主在收集箱里时按 `intake.update` 与提交者判定，收集箱关着或条目已关闭答 409”；重复删除、重复完成的说明也涵盖这两个 409。处理器测试逐码回答（`business_errors_test.go`），与契约同一 Task（`apitest.Main` 对声明了却没回答的码失败）。
- Go 的文档注释：`ErrIntakeDisabled`、`ErrIntakeClosed`、`ActionUpdateIntake`、`refuseIntakeOff`、`CheckOpen`、`rules.go` 那一句加宽到资产（与行为同一 Task）。
- 网页的两条码文案已有，不改。

## 3. 局部裁定

- **R1 规则与设计相同**：图片与附件同一个规则（DP-1 (a)）；§1 的四条收窄设计已批准，阶段记录向负责人逐条说明。
- **R2 共用提交者规则**，不复制；资产路径不用 `hasIntakeRow` 与访客字段的规则。
- **R3 开关与时段只在锁下判**，只对在收集箱里的工作项宿主，复用 `refuseIntakeOff` 与 `CheckOpen`。
- **R4 开关与归档同一次读**（加宽资产的项目读，具名结构）。
- **R5 资产的读不变**（`issue.read`）。
- **R6 已关闭的宿主没有删除的例外**；已绑定附件的删除照 M5。

## 4. 验收标准

- 契约、生成物、处理器逐码；`make gen-check` 无差异。
- 纯搬移的提交断言不改。
- 单测（issue/app 的资产宿主判定）：管理员、成员提交者、访客提交者可写；非提交者的成员、空创建者对非管理员 403；开关关着、拒绝、重复、项目归档 409；到期的暂缓可写；已接受的工作项照 `issue.update`；评论宿主不变；交叉格：归档 × 关着（答归档）、关着 × 已关闭（答关着）、非提交者 × 已关闭、非提交者 × 关着（都答 403）；只看自己的访客对别人的条目 404 与随机 id 相同。
- 宿主矩阵（真实的 app）：在收集箱里的工作项作为宿主，四个写 × 角色与提交者。
- 组合测试（HTTP）：H11——访客提交者给自己待处理的条目上传描述图片并经 `updateIntakeIssue` 绑定、加附件；非提交者的成员 403；拒绝后再写 409；关掉收集箱后 409；接受后照 `issue.update`；读不变（附件列表、内容）；评论宿主不受影响。
- 交错：资产写与条目被拒绝、收集箱被关、条目被接受（接受后按 `issue.update` 判定）；至少一格真实地在 `LockIssueProjects` 上等待（语句名的锁探针）。
- 权限矩阵：`createAsset` 以收集箱里的工作项为宿主的一行，带对回答范围的 `check`。
- e2e：H11 的资产故事（PAT 与会话对等）。
- 每个性质至少一个能让测试失败的变异（plan 的变异表）。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒，两路各自记；新的顶层测试名只有一个（T7），检查点 1 之前按 `tools/go-test-shard.sh` 的算法算好换路的清单。

## 5. 后续责任

- P5：网页的收集箱详情按状态开放图片与附件的编辑（Plane 网页只对管理员与提交者、待处理与暂缓开放）。
  - **处理结果（M7/P5a）：** 随 P5b（Q21）：P5a 的新详情（`inbox/content/detail.tsx`）不给描述编辑器 `images`，没有附件一栏（[P5a spec](P5a-web-intake.md) §1）。

## 6. 风险

- `Locks` 的变体与资产项目读的加宽触及资产的全部写：先纯搬移，回归由已有的宿主矩阵与资产测试覆盖。
- 开关的判定若放进 `Check` 的共用部分会连评论宿主一起拦：只对 `AssetIssue` 且在收集箱里的宿主判，变异钉住。
- 第 0 路的时间：资产处理器的测试在第 0 路（其余的包都在那一路），新的 bootstrap 顶层测试会让后面的换路；只加一个顶层名，其余为已有测试的子测试或表行。
