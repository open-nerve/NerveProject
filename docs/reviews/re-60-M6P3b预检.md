# M6 / P3b 预检

> 2026-11-09：预检通过（修正后），可以按 P3b 的 T1 开始实施。
> - **目标**：[P3b spec](../v0/M6-cycles-modules/specs/P3b-list-engine.md)、[P3b plan](../v0/M6-cycles-modules/plans/P3b-list-engine.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md)。
> - **基线**：main `2648a73940f053940b3f321f241b1a25eaac8209`（P3a 的 noff，main CI 42487991301 首次通过）。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：两个只读调研代理对照 main 的代码，核对了 P3a 之前写的列表引擎笔记，并补写现状笔记，随阶段台账归档：
   - 后端：列表引擎在 P3a 之后的插入点、“我的工作项”的目录、按需目录的先例、项目偏好的模板、planning 的加锁之路与删除清理；
   - 网页与 e2e：显示属性的保存路径、服务端新词在网页的去向、生成类型的编译影响、共用夹具的行、e2e 的钉子、Plane 的偏好接口。
2. **写 spec 与 plan**：据此写出 P3b 的 spec 与 14 个 Task 的 plan（`b10571b0`）。
3. **审查**：一次 fresh 只读审查（`fable`）对照分支上的实际代码逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在本 Task 内变红而计划不修；
   - R1–R10 是否站得住；
   - 是否有须负责人决定的事；
   - A1–A8 的覆盖。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **列表引擎**：
  - `listvocab.go` 的筛选、分组、排序都没有迭代与模块；
  - `appendID` 遇到 `none` 时只分标签与负责人两路，新字段的 `none` 会被记到负责人；
  - 三条列表语句各自手写目录、筛选、名字键与分组；P3a 已在三者的投影里加了 `cycle_id`、`module_ids`；
  - 排序不分范围，“我的工作项”的状态、标签、负责人目录已覆盖调用者的全部可见项目；
  - 附件计数是唯一按需取的目录，只在按附件数排序时取。
- **共用夹具**：`api/testdata/issue-order.json` 有四个消费者，Go 三个、网页一个，各自钉着场景数或排序数；e2e 的 `i4` 以 `satisfies Record<IssueListOrder,…>` 钉着排序词。
- **项目偏好**：
  - 筛选是十个键的类型化结构，显示属性是 12 个必填布尔；
  - 网页保存显示属性时回传服务端上次给的整个对象，加键不会使网页的保存被拒；
  - 改偏好是项目行 `FOR SHARE`、不进写闸门，不拒绝已归档的项目与 OwnOnly。
- **planning**：
  - 读之路（`Reads.run`）没有写时的拒绝；
  - 写之路（`Locks.run`）拒绝 OwnOnly、已归档的项目与开关关闭；
  - 删模块时在同一事务软删成员与链接；
  - 删除任务与清理按命名步骤推进。
- **`common.yaml`**：已有被两个模块共用的 `LinkMetadata`；生成器解析外部引用（`IsExternalRefsAllowed`），bodyshape 走解析后的 schema。

## 3. 预检结论与处理

**结论：修正后可实施。**
- **R2、R3**：按需目录与行在同一个只读快照；“我的工作项”的目录只覆盖调用者的有效成员项目，`cycle.read`/`module.read` 对访客开放，名字只用来排位、不出服务端；排序保持不分范围是对的。
- **R5**：共用显示设置合乎架构规则：`domain`/`app` 可导入 `internal/shared`，`shared` 只用标准库；按 P2 的先例不算超范围，T6 不改行为。
- **R6**：
  - 偏好的共享锁之路与项目偏好一致；
  - `busyTx` 的锁超时是 `server_busy`，PATCH 须声明，GET 不需要；
  - 锁序：PATCH 先取项目行 S 锁，再写自己的偏好行；删迭代/模块与工作项的写先取项目行 N 锁；删除任务在项目已删除后才运行。所以没有死锁。
- **R4、R7、R8、R9**：成立。

处理均已写入 spec/plan：

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| M1 | Medium | T8 建表后，一批 Go 守卫立即变红，原计划到 T9 才修：每张表恰好一个清理者（`purge_catalog_test.go:99-103`）、删除目录（`projectPlanningTable`）、两个父范围下都要有种下的行（`workspace_deletion_test.go:61-75`）、`purge_runtime_test` 的清单。T8 会留下一个红的提交 | 删除任务与清理的步骤、守卫与夹具并入 T8，在同一个 Task 内转绿；T9 只剩删迭代/模块时软删偏好行 |
| M2 | Medium | T6 的改名漏了几处，`make lint-web` 的类型检查（含 `@nerve/e2e`）会在 T6 内变红：`filter.store.ts:233`、e2e `i5` 的 `:2`、`:91`、`:374`；Go 的 `issue/module.go:44-50` 别名与导出接口、`project/app/issue_preferences_test.go:14-26` 的假实现 | T6 的 Files 补全，并加一步提交前 grep |
| L1 | Low | `Validate` 留着显示属性参数却不再读它 | 签名改为 `Validate(filters, display)`（spec R5、§2.2） |
| L2 | Low | `00045` 的 Down 没写要先删 14 键的 CHECK，剥键的 UPDATE 本身就会违反它 | spec R4 与 T7 写明次序：删新 CHECK → 剥键 → 旧默认值 → 重建旧 CHECK |
| L3 | Low | T11 漏了根 `api/openapi.yaml` 的两条路径（`TestRootListsEveryModulePath`）；T7 没点名 `customProperties` 与项目矩阵的偏好行 | 补入 T11、T7 |
| L4 | Low | e2e `i5:199-205` 把 `group_by:"cycle_id"` 当作被拒的词，T3 起服务端接受它，原计划到 T7 才改；`IssueListGroupField` 的说明也要写新词 | 移到 T3，与 `project_issue_preferences_test.go:114` 一起改 |
| L5 | Low | 行号漂移：清理规模“必须有过期行”在 `:96-99`；T5 写 `seedListOrder` 要打开开关，但读取不看开关 | 改正行号；T5 改为以 SQL 种，不提开关 |
| L6 | Low | T10 给 `Projects`/`PlanningProjects` 加 `Share`，实现它们的假实现要跟着长：`lock_test.go:60-66`、`bootstrap/planning_ports_test.go:20-43` | 补入 T10 |
| L7 | Low | R6 让 OwnOnly 的工作区管理员兼访客也能改迭代/模块的偏好，与设计 §3.7“OwnOnly 时所有 planning 的写 403”的字面不一致 | 那句的理由是写会动到看不见的工作项、或把看不见的数量写进回答，个人显示设置两者都不涉及；项目偏好同样放行。spec R6 写明这一澄清，T13 写进设计 §3.7。属于局部澄清，不改架构 |

**提给负责人留意、不须决定的两点**：
- `ProjectDisplayProperties` 多两个必填键：在设计 §5 范围内，先例是 `00038` 的 `attachment_count`。部署前打开的网页标签页，第一次保存显示属性会被拒一次，刷新即好。
- 生成的类型名由 `Project*` 改为 `IssueList*`：JSON 不变。

## 4. 计划覆盖

| 验收 | Task |
|---|---|
| A1 | T2 |
| A2 | T3 |
| A3 | T4、T5 |
| A4 | T6、T7 |
| A5 | T10、T11 |
| A6 | T8、T9、T11 |
| A7 | T1、T6、T11、T13 |
| A8 | T14 |

- **G15 与 G17**：G15 的接口与数据库证据在 T2–T5、T12；G17 的偏好矩阵行在 T11，含已归档的变体。
- **变异**：都对准所测性质，包括：`appendID` 的两路 `else`；漏掉 `deleted_at IS NULL`；`none` 与 id 取交；漏掉的 `CASE` 分支；最大取代最小；没有的排在最前；目录排除已归档；每种排序都取目录；Down 不剥键；PATCH 走写之路；`Share` 不加锁；时钟在锁之前；GET 写入；upsert 的冲突目标；夹具不写偏好。
- **检查点**：完整七项在 T7（建新表之前）与 T14 各一次。e2e 的删除故事要求每张带 `project_id` 的表都有行，所以从 T8 到 T11 按设计保持红，在 T11 的 `furnishPlanningRows` 写偏好后转绿。
