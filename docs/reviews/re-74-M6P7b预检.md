# M6 / P7b 预检

> 2026-11-14：预检通过（修正后），可以按 P7b 的 T1 开始实施。
> - **目标**：[P7b spec](../v0/M6-cycles-modules/specs/P7b-list-vocabulary.md)、[P7b plan](../v0/M6-cycles-modules/plans/P7b-list-vocabulary.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `f42f9d6457ccc9540a687019f3530e253fe0f7b5`（P7a 的 noff，双父 `7868a6b4`、`5d6dab04`）；审查对象为分支 `codex/m6-p7b` 的 `cf92e3d5`。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：一个只读调研代理在 P7a 的最终树 `5d6dab04`（与 main `f42f9d64` 的树相同）上写出 P7a 合入之后的增量笔记（台账 `research/web-delta.md`，随阶段归档）：P7a 造好的接缝（写入单元的 `changeIssue` 与 `IssueChange`、规划目录与 `usePlanningNames`、命令宿主）、列表的目录与词表的现状、“添加已有”的弹窗与三个入口、e2e 的余量、Task 的估算。它更新 P7 的两份旧笔记 `research/web.md`、`research/plane-contract.md`（写于 P7a 之前的 `c0572192`）。
2. **裁定与起草**：主控在写 spec 之前裁定 E1–E15（台账 `controller-rulings.md`）——范围与 16 个 Task，“添加已有”排除本页的并显示当前的迭代，拖拽的请求体在轮到时算、模块拖到“无”只移出来源，分组的列的次序与拒收，分组头只加入本页，一批逐个 `PATCH` 至多 2 个在途，筛选含“无”，开关关闭时存下的照旧，删去排序键的死分支，归档页不加，“添加已有”的弹窗在命令宿主，分组里的快速新建带组的值，e2e 预算 14 秒，只改网页，行数与质量——再由起草代理写出 P7b 的 spec（R1–R22，R1–R15 照录 E1–E15；A1–A15）与 16 个 Task 的 plan（`cf92e3d5`）。spec 的 R17（列表的目录以列表自己的读为门、不看项目的开关）与主控“新的规划数据的读者都以开关为门”的说明在开关关闭的边上不同，起草时写明请主控确认；主控在预检之前确认了它（台账的 Ruling：开关关掉后保存的名字排序 nerve 仍回答，按开关把门会让整个列表失败，合 E8；代价是开关关掉时目录仍会为名字排序读一次规划列表）。
3. **审查**：一次 fresh 只读审查（`fable`）对照 HEAD 的代码与只读的 `plane/` 逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在某个 Task 内变红而计划不修（knip、告警上限、关键词规则、i18n、`.oxlintrc.json` 的 swr 限制）；
   - R1–R22 是否站得住、类型与测试能否真的挡住所裁定的错误、有没有须负责人决定的事；
   - A1–A15 与 e2e 对 G15 的工作项一侧、G4 与 G11 的“添加已有”、M4 交接 M6-02 与 M3 交接 §1、§3 的覆盖，以及 e2e 作业的时长；
   - 与接口契约、总体设计 §7.7 是否一致，对 Plane 行为的描述是否准确。
4. **裁定**：主控逐条核过审查的结论并裁定（全部接受，F9 取“照实写、不加机制”），另给出 P7a 合入后的 main CI 与归档的数据，再由起草代理改入。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **列表的词表与目录**：网页的筛选、分组、排序词表里没有迭代与模块（`core/lib/issue-filters.ts:6-67`）；列表的目录的两项是空的 `Map`（`core/store/issue/helpers/issue-list-binding.ts:84-86`），按名字排序、行上有值时 `prepareCatalog` 抛“Required issue directory is unavailable”（`:278-288`）；状态、标签的先例在 `:242-264`（项目不在 store 时先读项目，叫不出就重读）。
- **偏好的类型**：基类 `PagePreferences = Pick<ProjectPreferences, …>`，service 的 `update` 是方法写法（`page-preferences-filter.store.ts:21-28`），每一页都以 `toProjectFilters` 转换（`:222-223`）；生成的 `CycleIssueFilters`、`ModuleIssueFilters` 各少一个键（`web/packages/api-client/src/schema.gen.ts:4152-4165`、`:4179-4192`），`ProjectIssueFilters` 两者都有（`:3493-3507`）。
- **快速新建**：预填类型 `QuickAddPrefill` 七个字段、提交的白名单同七个（`issue-layouts/quick-add/root.tsx:34-36`、`:179-180`），没有 `cycle_id`、`module_ids`；`createBody` 自 P7a 起放行两者（`core/store/issue/issue-write-fields.ts:17-18`）。
- **到上限的文件**：`issue-list-binding.ts` 487 行、`use-work-item-filters-config.tsx` 353 行、`command-host.tsx` 255 行；工作项根 store 320 行、写入单元 302 行（M6-M8-08）。
- **守卫现值**：告警上限 44；`root.store.test.ts` 21 个 store；关键词 78 条规则、7 条例外，phase `M6/P7`。
- **e2e 的时长**：P7a 的最终分支 CI 43600392398 的 e2e 作业 23 分 48 秒；P7a 合入后的 main CI 43603772626 attempt 1 全绿，e2e 作业 132478430823 为 14:31:09Z–14:54:52Z，23 分 43 秒；记下的最慢一次仍是 43549053092 的 27 分 41 秒；期限 35 分钟，报告线 30 分钟。

## 3. 结论与处理

结论 **Ready after fixes**：四处 Important、五处 Minor，没有 Critical，没有架构问题，没有须负责人决定的事。全部在实施前改入 spec 与 plan（本记录所在的提交）；F4 把一个 Task 拆成两个，Task 由 16 个变为 17 个（主控接受，在设计 §12 的“约 16”之内），T7 起顺延，检查点仍在中间的拖拽（T9，原 T8），收据名随之改；下表的 Task 号是改后的（括号里是 `cf92e3d5` 的号）：

| 编号 | 问题 | 处理 |
|---|---|---|
| F1（Important） | R18 的类型保证不成立：生成的 `CycleIssueFilters` 是 `ProjectIssueFilters` 的结构子集（`schema.gen.ts:4152-4165` 对 `:3493-3507`，属性都可省），一个类型为 `ProjectIssueFilters` 的值（不是字面量）不论参数的变型都赋得进去，`toProjectFilters` 照样当得了迭代页的转换，T4 的 `@ts-expect-error` 会成为未用的指令。spec 把原因写成方法参数的双变，不对 | spec §2.3、R18：网页一侧推出本页的类型 `Omit<ProjectIssueFilters, "cycle_id"> & { cycle_id?: never }`（模块页同）；转换回答本页的类型；service 的 `update` 写成属性、收本页的类型（由 `CyclePreferencesUpdate`、`ModulePreferencesUpdate` 推出），配错的 service 赋不进去；原因改为结构子类型（与 re-72 F1 同一个教训）。A3 随之改。plan T4：类型测试为转换配错、service 配错与一个不是字面量的 `ProjectIssueFilters` 的值；收据 `t4-type-never`（去掉 `never`，三处都成为未用的指令）、`t4-type-property`（方法写法，“service 配错”一处成为未用的指令）；Review Focus 第 2 条 |
| F2（Important） | 目录的读法让寻常的流程大声失败：两份列表都已在 store 之后，别处新建了迭代（模块）、一个看得见的行加入它，按 `cycle_name` 排序的读就抛错，写的确认也重读进同一个失败 | spec §2.2、R17、A2：目录照状态、标签的先例（审者引 `issue-list-binding.ts:249-262`，HEAD 上判断与读在 `:249-260`）——有叫不出的就重读未归档的列表，store 已有也读；仍叫不出再重读已归档的；仍叫不出才失败。共用的纯函数只定次序（先未归档、后已归档）；`usePlanningNames` 保留自己的“已持则不读”（hook 随渲染反复求值，不能无尽地重读）。“新鲜”一段改写。plan T3：RED 的“store 已有未归档的列表时只读已归档的”换为“两份都在 store 里、行带别处新建的迭代 → 重读未归档的一次、叫出；重读两份后仍叫不出 → 失败”；变异 `t3-mut-no-reread`，另加 `t3-mut-hook-no-held-guard`（hook 不能失去已持的守卫） |
| F3（Important） | E12、R12 说错了机制：`createBody` 放行 `cycle_id`、`module_ids`（`issue-write-fields.ts:17-18`），真正的门是快速新建的根的 `QuickAddPrefill`（`quick-add/root.tsx:34-36`，七个字段）与提交的白名单（`:179-180`） | spec §2.5 的现状与做法、§2.11、R12、A6 引 `root.tsx`，写明 E12 的机制更正、裁定不变；plan T7（原 T6 的一半）的 Files 加这个文件（类型与白名单各加两键）；“根不传两键”与 `t7-mut-quick-add-key` 落在同一个测试上，是等价变异，记一个、台账写明 |
| F4（Important） | 原 T6 太大：常量的推出、开关的词表与谓词、它的读者、分组与子分组的面板，再加分组里的快速新建与拒收的列，超出一个实现者的量 | 拆为 T6（A8 的推出，R8 的 `planning-switch.ts` 的表与谓词、它的读者、分组与子分组的面板）与 T7（R12 的快速新建含 `quick-add/root.tsx`，拒收的列不给“+”与快速新建）。计划为 17 个 Task，其后顺延：筛选的界面 T8、拖拽 T9（检查点 `t9-gate`）、表头 T10、“添加已有”T11–T13、页面故事 T14、T15、文档 T16、门禁与合入 T17（`t17-gate`）；spec 与 plan 的 Task 号、收据名与回归边界随之改（P7a 的 Task 号不改） |
| F5 | G15 的页面故事加模块一列的一个排序方向（约 1 秒），核 nerve 的多值取最小的名字（`api/modules/issue.yaml:1868-1872`） | spec §2.14：世界改为 W1 在 Alpha 与 Gamma、W4 在 Beta，模块一列 Z→A 一次：行依次 W2、W4、W1、W3（取最大的名字时 W1 会在 W4 之前）；两文件的分配改为 9 + 5 秒，合计仍 ≤ 14 秒（约束的是合计）。plan T14、T15 的压缩线随之改；T14 的压缩改为移出按模块分组一步（原写的泳道不在故事里） |
| F6 | 审者说 T7（现 T8）引的 `filter-config.test.tsx` 不存在，配置 hook 既有的测试是 `core/components/work-item-filters/filters-hoc/base.test.tsx` | 起草时核对：这个文件存在，在 `core/components/work-item-filters/filter-config.test.tsx`（plan 只写了文件名），它经 `WorkItemFiltersHOC` 用真的 `useWorkItemFiltersConfig`；`filters-hoc/base.test.tsx` 则把这个 hook 换成替身。照审者的意思改：plan T8 写出完整的路径与这两个文件的分别，新 hook 的测试是新文件；变异 `t8-mut-own-key-offered` 不再点那个文件名 |
| F7 | T2 的基线清单：`permission-reads.test.ts` 不导入绑定；漏了 `store/issue/project-directory-order.test.ts`（目录的测试）、`hooks/use-issue-list-refresh.test.tsx`、`hooks/use-group-dragndrop.test.tsx`、`workspace-draft/issue.store.test.ts`、`issue-details/detail-changes.test.ts` | plan 回归边界“拆分”：照 grep 改为十个文件；另记 grep 不到而执行绑定的 `cycle/priority-issues.store.test.ts`（它的 store 构造绑定，`priority-issues.store.ts:5`、`:67`）；仍以 grep 为准 |
| F8 | spec §2.2 按项目的循环少了先例的“项目在 store，否则 `fetchProject`”一步（审者引 `issue-list-binding.ts:247-248`，HEAD 上是 `:245-246`）；`getCycleById` 只给项目 store 仍给的项目的迭代 | spec §2.2、§2.12、A2：先读不在 store 的项目；plan T3：RED 加“成员列表跨两个项目，B 不在项目 store 里：先读 B，再各读自己的”，变异 `t3-mut-no-project-fetch` |
| F9（主控裁定：照实写） | spec 说“冷的页上……`Reconciled` 留较新的”，不确切：被较新的读超过的那一次 `Reconciled.fetch` 回答 `undefined`（`core/lib/reconciled.ts:52-56`），目录这一次可能失败，写的确认随之 `markUnavailable` 再重读 | spec §2.2 改为“被超过的读”一段：目录这一次可能失败，列表的读可重试，写的确认 `markUnavailable`（`issue-writes.ts:178`，绑定 `:468`）后重读叫出；这是状态、标签今天已有的行为，不加机制。§6 加一行 |
| 主控 | P7a 合入后的 main CI 与归档的数据；e2e 的余量在刀口上 | spec 抬头与 §2.14 的数据表：main CI 43603772626 attempt 1 全绿、e2e 作业 132478430823 为 23 分 43 秒（1423 秒），最慢的仍是 27 分 41 秒，锚不变；估算约 23 分 55–59 秒；P7b 14 + P7c 8 = 22.0 秒对最坏一端的 22.4 秒，14 秒是硬上限，停下报告的规则不变。plan 的抬头与 T1 的 Step 0、Step 3 写上归档的路径与主控记下的数（3133 个文件、清单 sha256 `d16a2da0…`、清单的 `head` `5d6dab04`、`cleanup.log`），T1 以收据再核 |

## 4. 审者核实为正确的事实（摘要）

审者的报告没有另存文件，下面只记主控转达、起草时又在 HEAD 上逐一核过的：

- 生成的三个筛选类型与 service 的参数（F1）：`schema.gen.ts:3493-3507`、`:4152-4165`、`:4179-4192`；`core/services/planning/cycle-preferences.service.ts:22`、`module-preferences.service.ts:22`；`page-preferences-filter.store.ts:21-28`、`:222-223`。
- 状态、标签的读法（F2、F8）：`issue-list-binding.ts:245-246` 先读项目，`:249-260` 叫不出就重读（行号以 HEAD 为准改过）；`use-planning-names.ts:59-61` 的已持守卫。
- 快速新建的两道门（F3）：`quick-add/root.tsx:34-36`、`:179-180`；`issue-write-fields.ts:17-18`。
- `Reconciled.fetch` 对被超过的读回答 `undefined`（F9）：说明在 `reconciled.ts:52-56`，`return undefined` 在 `:65`；`issue-writes.ts:178`、绑定的 `markUnavailable` 在 `:468`。
- T2 的 grep（F7）：十个测试文件命中，`permission-reads.test.ts` 不在其中。
- nerve 的 `module_name` 取工作项的模块中最小的名字，含已归档的，没有的在最后（F5，`api/modules/issue.yaml:1868-1872`）。
- 其余 R1–R22 与 A1–A15 没有被指出事实错误；没有守卫会在 Task 的次序里变红而计划不修（F4 拆分之后的次序同样）。

## 5. 负责人须知

开工不须负责人决定任何事。R1–R15 是主控在已批准的设计之内、写 spec 之前的裁定（E1–E15），R16–R22 都在设计与 P5b–P7a 的先例之内；R17（目录不看开关）由主控确认；F3 只更正了 E12 所说的机制，裁定不变。本段只改网页，不改 `server/`、`api/`。

需要负责人知道的一件事：

- **e2e 的余量在刀口上**：以记下的最慢一次（27 分 41 秒）为锚，P7b 与 P7c 两段合计的新页面故事至多约 22.4–27.1 秒本机 chromium；预算是 P7b 14 秒加 P7c 8 秒，共 22.0 秒，离最坏的一端只差 0.4 秒。本段 X = 14 秒时上界约 29 分 13–26 秒，估算约 23 分 55–59 秒。T1 重算、T9 的检查点校准、T17 推送之前再算；估算、上界或任一次实测越过 30 分钟，或 T1 算出的余量小于 22 秒时，停下报告负责人、不推送；e2e 作业的结构（例如分片）、期限与超时是负责人的决定，P7c 多半要提出来。
