# M7 / P2b：收集箱的列表、内容修改、删除与待处理数

> 状态：spec 经预检 [re-86](../../../reviews/re-86-M7P2b预检.md) 修正后通过。基线 main `670399c4`（P2a 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H9、H11）、§3.5、§3.6、§3.10、§3.11、§4、§5、§15 A/B/D2/E；P2a 见 [P2a spec](P2a-intake-submit-triage.md) 与 [P2a 阶段记录](../reviews/P2a-intake-submit-triage-review.md)。现状调研见本阶段台账 `research/p2b-facts.md`（ignored，随阶段归档；下文 DP-n 指它 §12 的决定点）。

## 1. 目标与边界（P2b / P2c 的划分）

P2a 把列表、内容修改、删除接口、待处理数与 Triage 宿主的资产判定留给 P2b。照原范围约 20 个 Task，超过设计 §12 约束 2 的约 16；其中资产判定跨模块（资产的契约、项目的端口与 SQL、资产读路径的判定次序，预检 re-85 I5），是评审敏感的一块（约束 3）。按此再划分（R1）：

- **P2b（本 spec）**：收集箱列表、内容修改 `PATCH /intake-issues/{issue_id}`、删除 `DELETE /intake-issues/{issue_id}`、各项目的待处理数；动作 `intake.read`、`intake.update`、`intake.delete`、`intake.count`；P2a 审查留下的 m2、m3、n1；它们的权限矩阵、写锁表、删除窗口与 H9、H11 其余的证据。
- **P2c**：Triage 宿主的资产判定（DP-21–23）：资产的四个操作按 `intake.update` 与同一时段判定、资产路径读开关、先读工作项再选动作的次序，`asset.yaml` 与资产处理器测试；H11 里“描述的图片同此”的证据。

P2b 不做：网页（P5）；Triage 宿主的资产（P2c）。

**P2b 合入后、P2c 之前 main 的已知状态**：拒绝、重复的条目从此可以列出、可以删除（P2a 记录 §4 说的“删不掉、列不出来”到此结束）。Triage 宿主的资产照旧按 `issue.update`（管理员、成员）判定：访客提交者不能给自己的条目加图片与附件，非提交者的成员能；设计 §3.6 的资产一句与 H11 的图片证据留到 P2c。`intake_view` 默认关、网页还在打 Plane 的旧地址，不算不安全。阶段记录向负责人说明。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/issue/` | 领域（列表的查询与游标、修改请求的校验、关闭的状态、四个动作的声明）；存储（列表与计数的 SQL、条目行的软删）；用例（修改核心与删除核心的类型、列表、计数、内容修改、删除；m2、n1）；HTTP（四个操作） |
| `server/internal/modules/project/` | “这些项目里哪些开着收集箱”的端口与 SQL（DP-20，只读 `intake_view`） |
| `server/internal/modules/access/` | 四个动作的规则（`intake.delete` 带 `AllowCreator`） |
| `server/internal/bootstrap/` | 端口转换；权限矩阵、写锁表、删除窗口、空创建者测试；组合测试、交错 |
| `api/modules/issue.yaml`、`api/openapi.yaml` | 四个操作与路径、`IntakeIssue`、`IntakeIssuePage`、`IntakeIssueUpdate`、`IntakeCounts`、新码 |
| `web/` | 生成的类型、新码的文案 |
| `e2e/` | H9 的接口故事 |
| `docs/` | plane-diff、硬规则、设计 §3.5、§3.10、§12、§14、阶段记录 |

### 2.2 修改核心、删除核心的类型、m2 与 n1（R2）

- **m3**：把 `(*UpdateIssue).updateLocked` 原样抽成自己的类型 `IssueUpdate`（新文件，照 `IssueCreation`），依赖里不带时钟，时钟按调用给；持有者：`UpdateIssue`、`TriageIntakeIssue`、新的 `UpdateIntakeIssue`。单独提交纯搬移：断言不改，只为编译改构造处与假实现（台账逐个记）。
- **删除核心**：`DeleteIssue` 锁下的那一段同样先纯搬移成自己的类型，`deleteIssue` 与本阶段的删除共用。
- **m2**：`createLocked` 交回读回的工作项与时刻，`CreateIssue` 自己组回答（`PublishDraft` 已如此），提交不再建一次丢掉的回答；`submit_intake_test.go` 期望的调用清单少了核心回答的读，这是 m2 的 RED，不是“断言不改”。
- **n1**：`acceptance` 在默认状态就是工作项当前的状态时返回内部错误，位置与 m1 的检查相同（任何写之前）。只在 SQL 破坏了行与状态的一致时才到得了，主要证据是单测。

### 2.3 列表 `GET /projects/{project_id}/intake-issues`（`listIntakeIssues`）

- 查询参数（DP-12、13、14、15、16）：
  - `status`（逗号分隔的多值，五个名字，按有效状态：`pending` 含已到期的暂缓，`snoozed` 只含未到期的；不给即全部状态，R9）、`priority`、`label_id`（uuid 或 `none`）、`assignee_id`（uuid、`me` 或 `none`）、`created_by_id`（uuid 或 `me`）；
  - `created_at`、`updated_at`：日的写法（`ParseListDateFilter`），在 Go 里换成 UTC 的半开时刻区间 `[起日 00:00, 止日次日 00:00)` 再交给 SQL（不用 `::date`，它随会话时区）；不收 `none`（422）；
  - `order_by`：`created_at`、`updated_at`、`sequence_id`，各带 `-`；默认 `-created_at`；同值按工作项 id 同向；
  - `limit`、`cursor`。
  - 解析是独立的 `ParseIntakeQuery`（复用原子的解析），HTTP 层的允许参数照 `adapter/http/list_query.go` 的两张按路由的清单另列一张；游标 `[order, key, id]` 用共享的封套；`sequence_id` 不进项目、成员列表与视图设置的词表。
- 回答（DP-11）：`IntakeIssuePage = {total, data: [IntakeIssue], next_cursor}`；`IntakeIssue` = 列表行 `Issue` 加上非空的 `intake`（`{status, snoozed_till, duplicate_to_id}`）。`intake` 与同一工作项详情里的 `intake` 逐值相同（同一套规则：有效状态；重复目标按页一次批量读、读的人看得到且未删除才给，否则 `null`，DP-17）。`total` 是同一筛选下的条数。整页在一个只读事务里，时刻读一次。
- **行的裁剪与项目列表相同（R10）**：`sub_issues_count` 只数调用者看得到的子工作项、看不到的父的 `parent_id` 为 `null`（复用 `hideParents`）、附件数经已有的 `projectAttachments`。不重写这些规则。
- 范围：项目里有未删除条目行的工作项，含已接受的（含之后归档的，DP-25）；不受开关影响（开关关着时仍可读，同 P2a 的读）。
- 授权 `intake.read`（项目的管理员、成员、访客）。只看自己的访客只列自己创建的（`NOT own_only OR i.created_by_id = 调用者`，M4 列表的登记写法，DP-18；空创建者不选中，空创建者测试随之扩展）。
- 拒绝次序：查询参数 400/422 → 项目看不到 404 `project.not_found` → 403。

### 2.4 内容修改 `PATCH /intake-issues/{issue_id}`（`updateIntakeIssue`）

- 请求体 `IntakeIssueUpdate`（平的，`additionalProperties: false`，DP-4）：`name`、`description_html`、`priority`、`assignee_ids`、`label_ids`、`target_date`（可空）、`skip_activity`（与 `updateIssue` 同一规则）。字段的校验照 `CheckIssuePatch`。
- **提交者**是工作项的创建者 `Issue.CreatedByID`（与 `AllowCreator`、列表的只看自己同一个定义，R11）；创建者为空时没有提交者。
- 授权 `intake.update`（项目的管理员、成员、访客）加相对规则（设计 §3.10）：不是管理员的须是提交者，否则 403 `forbidden`（DP-6）；访客提交者只能改 `name`、`description_html`、`skip_activity`，给了别的字段 403 `forbidden`（DP-5）。两者都在 `write.check`（预检与锁下各一次）。
- 时段：存的状态是 `pending` 或 `snoozed` 才能改；`accepted`、`declined`、`duplicate` 答 409 `issue.intake_closed`（DP-7；只看存的状态，不读时钟）。
- 拒绝次序（DP-8）：请求体 400/422 → 看不到 404 `issue.intake_not_found` → 403（角色）→ 没有条目行 404 `issue.intake_not_found` → 403（不是提交者）→ 403（访客的字段）→ 闸门、锁、锁下再判定 → 409 `project.archived` → 409 `issue.intake_disabled` → 409 `issue.intake_closed` → 修改核心的 422。两种 404 与随机 id 逐值相同。“没有条目行”先于“不是提交者”是机制决定的（两者都在 `write.check`，条目行先查）；与删除相反（见 2.5），两者都不透露什么：读得到工作项的人从详情的 `intake` 已知道它有没有条目。
- 写：经 `IssueUpdate` 核心、补丁限于上面的字段（不能改状态、父、迭代、模块）；动态、`issue.updated`、提及、描述版本、描述里的图片照修改核心。工作项修改的源端守卫（P2a）不变：`updateIssue` 对在收集箱里的工作项仍 409。
- 回答 200 `IssueDetail`（第八个回答详情的操作，带 `intake` 与 `is_subscribed`）。

### 2.5 删除 `DELETE /intake-issues/{issue_id}`（`deleteIntakeIssue`）

- 授权 `intake.delete`：项目管理员；不低于成员的提交者（`AllowCreator`，即工作项的创建者）；任何状态（设计 §3.5、§3.10，Plane 相同，DP-9）。
- 拒绝次序：看不到 404 `issue.intake_not_found` → 403 → 没有条目行 404 → 409 `project.archived` → 409 `issue.intake_disabled`。“不是提交者”在判定里（`AllowCreator`），所以先于“没有条目行”，与修改相反（见 2.4），同样不透露。
- 写（DP-10）：经删除核心：
  - `pending`、`snoozed`、`declined`、`duplicate`：删除工作项（条目行随之软删，`issue.deleted`，与 `deleteIssue` 相同的连带）；
  - `accepted`：只软删条目行（工作项留下，不写动态、不发事件）；工作项已归档的同样只删条目行，不答 `issue.archived`。
- 回答 204。

### 2.6 待处理数 `GET /workspaces/{slug}/intake-counts`（`getIntakeCounts`）

- 授权新的工作区级动作 `intake.count`（管理员、成员、访客，DP-19）；设计 §3.10 的规则表加一行。
- 范围（DP-20）：先取调用者看得到的项目（已有的 `Visible`：有效成员、未归档），再经 project 的新端口只读这些项目的 `intake_view`，列出开着的，每个都列（含 0），按项目 id；不另写成员关系与只看自己的 SQL，不改 `VisibleProject`。归档的项目不列（Plane 连归档的也数）：设计 §3.5 那句加“未归档”，plane-diff 登记。
- 数：有效状态为 `pending` 的条目（含已到期的暂缓）；只看自己的访客只数自己的。用 `intake_issues_project_id_status_idx`。范围与计数在一个只读事务里，时刻读一次。
- 回答 `IntakeCounts = {data: [{project_id, pending}]}`。拒绝次序：工作区看不到 404 → 403。

### 2.7 动作与码

- `intake.read`（项目：管理员、成员、访客）、`intake.update`（项目：管理员、成员、访客，加相对规则）、`intake.delete`（项目：管理员，`AllowCreator` 成员以上）、`intake.count`（工作区：管理员、成员、访客）。
- 新码 `issue.intake_closed`（409）。
- 各操作的 `x-problem-codes`（每个都有处理器测试回答）：
  - `listIntakeIssues`：`validation_failed`、`project.not_found`、`forbidden`；
  - `getIntakeCounts`：`workspace.not_found`、`forbidden`；
  - `updateIntakeIssue`：`validation_failed`、`issue.intake_not_found`、`forbidden`、`project.archived`、`issue.intake_disabled`、`issue.intake_closed`、`server_busy`、`asset.not_found`、`asset.not_uploaded`；不声明 `issue.archived`（在收集箱里的工作项归档不了，已接受的先答 `issue.intake_closed`）、`planning.*`（白名单里没有迭代、模块）与 `issue.in_triage`；
  - `deleteIntakeIssue`：`issue.intake_not_found`、`forbidden`、`project.archived`、`issue.intake_disabled`、`server_busy`。

### 2.8 按目录驱动的守卫（与契约同一 Task 补齐，R12）

- 权限矩阵的行：列表行 `write: true` 以 `prepare` 把本列项目的工作项放进收集箱（DP-24）；计数行判授权与项目集合；修改、删除行。
- 写锁表：修改、删除两行。
- 删除窗口：用例表读 18 → 20、写 26 → 28；期望在 `issue_deletion_window_test.go`：`getIntakeCounts` 在项目删除时 200、`check` 钉“只列留下的项目、待处理数为 1”；夹具 `issue_deletion_window_fixture_test.go` 让留下的项目 `f.q` 打开收集箱并提交一个条目；`request()` 给修改、删除加指向条目的分支（照 P2a 的分诊），各留一个控制变异（指向普通工作项时测不出）。
- 空创建者测试：跳过清单加 `listIntakeIssues`、`getIntakeCounts`；泄漏的目录加收集箱列表与计数（空创建者的条目对只看自己的访客不出现、不计数）。
- `rules_test` 的 `tableCells`、`creatorCells`；逐码的处理器测试；详情的回答测试加修改为第八个。
- 红窗口的清单按 `contract.Operations()` 的使用者找全（P2a 检查点 1 的教训，硬规则 03 §5）。

## 3. 局部裁定

- **R1 P2b、P2c 再划分**：见 §1；资产判定整段在 P2c。设计 §12、§14 随之改，阶段记录向负责人说明。
- **R2 修改核心、删除核心成自己的类型**（m3），m2、n1 一并做；搬移先单独提交。
- **R3 有效状态在 SQL 里**：`status = 'pending' OR (status = 'snoozed' AND snoozed_till <= now)` 为待处理，`now` 是请求读一次的时刻；列表的 `status` 筛选、计数用同一写法。
- **R4 访客提交者的字段**：给了不许的字段 403，不静默丢弃（Plane 丢弃）。
- **R5 关闭的条目**：新码 `issue.intake_closed`，内容修改（P2b）与资产（P2c）同一个意思。
- **R6 删除**：设计的授权，没有状态的时段；已接受的只删条目行；与 `deleteIssue` 共用删除核心。
- **R7 计数的授权**：新的工作区级动作 `intake.count`。
- **R8 日的筛选按 UTC**：同 Plane；Go 里换成半开时刻区间；网页按本地日计算的差异照旧登记。
- **R9 不给 `status` 即全部**：同其他列表的筛选，与 Plane 的默认（只列待处理）不同，登记。
- **R10 列表行的裁剪**：与项目列表同一套（看不到的父、子数、附件数），复用而不重写。
- **R11 提交者是工作项的创建者**：修改的相对规则、删除的 `AllowCreator`、列表的只看自己同一个定义；创建者为空即没有提交者。
- **R12 按目录驱动的守卫随契约同一 Task 补齐**。

## 4. 验收标准

- 契约、生成物、处理器逐码；`make gen-check` 无差异；网页的新码文案。
- 两次纯搬移（修改核心、删除核心）断言不改，只为编译改构造处。
- 组合测试（真实的 app）：
  - H9：管理员、成员、访客按各筛选与次序列出；有效状态（到期的暂缓算待处理）；日的两条边界（止日 23:59:59.999999 在内、次日 00:00 不在）与 `none` 422；只看自己的访客只见自己的；重复目标看不到为 `null`；游标翻页与 `total`；已接受、已归档的仍在；列表行的 `intake` 与详情的逐值相同。
  - 行的裁剪：访客提交的条目被接受后，管理员给它挂上自己建的父与子，只看自己的访客列出它时 `parent_id` 为 `null`、子数为 0。
  - 计数：开着收集箱、看得到的项目各一项（含 0），到期的暂缓计入，只看自己的访客只数自己的，关着收集箱或已归档的项目不列。
  - 内容修改：提交者、管理员、访客提交者（只两字段）、非提交者的成员 403、空创建者的条目非管理员 403、关闭的条目 409、开关与归档 409、源端守卫仍拦 `updateIssue`；提及、描述版本、`issue.updated` 照常；回答带 `intake` 与 `is_subscribed`。
  - 删除：四种状态删工作项、已接受的（含已归档的）只删条目行；提交者成员可删、访客提交者 403、管理员任何状态。
  - 交叉的拒绝次序：成员非提交者 × 有无条目行 × 修改与删除四格（修改：无行 404、有行 403；删除：两者都 403）。
- 权限矩阵、写锁表、删除窗口、空创建者测试补齐的行。
- 交错：内容修改与分诊、删除与分诊、计数读与提交（不要求串行，只要求不报错、数在两者之一）。
- e2e：H9 的接口故事（PAT 与会话对等）。
- 每个可见、迁移、守卫、锁的性质至少一个能让测试失败的变异（plan 的变异表）。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒；两路各自的增量与新测试自己的耗时分开记（按名字轮流分路，新测试名会让后面的换路）。

## 5. 后续责任

- P2c：Triage 宿主的资产判定（DP-21–23）与 H11 的图片证据。
- P5：网页的收集箱改用新接口（列表、修改、删除、计数、分诊）。

## 6. 风险

- 两次核心的搬移触及 `updateIssue`、`deleteIssue` 与分诊：先纯搬移，回归由已有测试覆盖。
- 列表 SQL 复制了访客规则的写法（M4 列表的登记写法），由只看自己与空创建者的测试钉住；行的裁剪复用列表的辅助。
- 删除窗口的计数与期望写死，随契约同一 Task 改。
