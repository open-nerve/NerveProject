# M6 / P7a 预检

> 2026-11-14：预检通过（修正后），可以按 P7a 的 T1 开始实施。
> - **目标**：[P7a spec](../v0/M6-cycles-modules/specs/P7a-work-item-planning.md)、[P7a plan](../v0/M6-cycles-modules/plans/P7a-work-item-planning.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `954a1838bff44e98cc44f15df524da27e42fb1d8`（P6b 的 noff，双父 `06b72d4b`、`c0572192`）；审查对象为分支 `codex/m6-p7a` 的 `ab4ef06b`。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：两个只读调研代理在 `c0572192` 的树（与 main `954a1838` 的树相同）上写出现状笔记（台账 `research/web.md`、`research/plane-contract.md`，随阶段归档）：
   - 网页：工作项一侧读写 `cycle_id`、`module_ids` 的现状（写入单元、行与表格的写、弹窗与草稿、命令宿主、显示属性）、规划 store 与目录的读、到上限的文件、交给 P7 的交接各项（P5a–P6b 的 §5、M1–M3 的交接）、e2e 的夹具与时长、守卫的现值、P7 的量与拆缝；
   - Plane 与契约：Plane 的迭代、模块下拉框与工作项上的属性、新建弹窗与草稿、列表的筛选与分组、“添加已有”、命令面板，nerve 的接口（工作项的 `cycle_id` 单值、`module_ids` 整体替换、草稿的 payload 整体替换、拒绝的错误码），两者的差异。
2. **裁定与起草**：主控在写 spec 之前裁定 D1–D10（台账 `controller-rulings.md`）——P7 拆为 P7a、P7b、P7c，表单优先于本页，模块集合以增减在轮到时算、迭代是单值，选项与值分开，跨项目的列表按项目读，草稿不发空的迭代、换项目清空，三个到上限的文件先拆，Plane 的缺陷不照搬，e2e 预算约 20 秒本机 chromium，只改网页——再由起草代理写出 P7a 的 spec（R1–R21，R1–R10 照录 D1–D10；A1–A12）与 15 个 Task 的 plan（`ab4ef06b`）。
3. **审查**：一次 fresh 只读审查（`fable`）对照 HEAD 的代码与只读的 `plane/` 逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在某个 Task 内变红而计划不修（knip、告警上限、关键词规则、i18n、`.oxlintrc.json` 的 swr 限制）；
   - R1–R21 是否站得住、类型与测试能否真的挡住所裁定的错误、有没有须负责人决定的事；
   - A1–A12 与 e2e 对 G4、G11 的页面部分与各交接验收条件的覆盖，以及 e2e 作业的时长；
   - 与接口契约、总体设计 §7.7 是否一致，对 Plane 行为的描述是否准确。
4. **裁定**：主控逐条核过审查的结论并裁定，另加 T1 回填 P6b 合入后的 CI 修复一项，再由起草代理改入。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **工作项一侧**：网页上没有迭代、模块的下拉框，没有一处组件读工作项的 `cycle_id`、`module_ids`（`research/web.md` §0）。
- **写入单元**：`core/store/issue/issue-writes.ts` 300 行、工作项根 store 320 行（P6b 的 A1，M6-M8-08）；`updateIssue` 在没有给出范围时退回 router 的工作区（`issue-writes.ts:260-261`）；`updateBody` 的白名单放行 `cycle_id`、`module_ids`（`issue-write-fields.ts:35-36`）；行与表格的写的类型是 `Partial<IssueDetail>`（`properties/all-properties.tsx:37`、`spreadsheet/columns/label-column.tsx:20`）。
- **到上限的文件**：`issue-modal/base.tsx` 452 行、`form.tsx` 432 行、`issue-layouts/utils.tsx` 399 行；`core/store/issue/helpers/issue-list-binding.ts` 487 行（P7b 要往里加）。
- **新建与分组头**：迭代页、模块页的新建测试把 `IssueDefaultProperties` 换成 `() => null`（`roots/cycle-create.test.tsx:66-68`、`module-create.test.tsx:66-68`），断言（`cycle-create.test.tsx:155-160`、`module-create.test.tsx:162`）今天靠 store 的注入成立；分组头把新建弹窗放在组里，列表核验时整个布局换成加载（`issue-layout-HOC.tsx:52-58`），命令宿主只在 `holds(dialog, dialog.data.project_id)` 时渲染对话（`command-host.tsx:160-165`、`:196`）。
- **表格的表头**：`columns/header-column.tsx:39` 在 `SPREADSHEET_PROPERTY_DETAILS` 没有条目时给 `null`；`spreadsheet-header-column.tsx:33` 总给 `aria-sort`。
- **守卫现值**：告警上限 44；`root.store.test.ts` 21 个 store；关键词 78 条规则、7 条例外，phase `M6/P6`。
- **e2e 的时长**：P6b 的最终分支 CI 43545459485 的 e2e 作业 23 分 09 秒；合入后的 main CI 43549053092 attempt 1 的 e2e 作业 132311928038 为 27 分 41 秒（03:14:42Z–03:42:23Z，故事本身约 25.4 分钟），是已记下的最慢一次，只有 M4 的 I4 (page) 的表格故事因时间预算失败（没有重跑），由修复分支 `codex/m6-p6b-ci` 拆为四个故事（本机 chromium 约多 5 秒）；期限 35 分钟，报告线 30 分钟。

## 3. 结论与处理

结论 **Ready after fixes**：四处 Important、六处 Minor，没有 Critical，没有架构问题，没有须负责人决定的事。全部在实施前改入 spec 与 plan（本记录所在的提交）；F3 把一个 Task 拆成两个，Task 由 15 个变为 16 个，下表的 Task 号是改后的（括号里是 `ab4ef06b` 的号）：

| 编号 | 问题 | 处理 |
|---|---|---|
| F1（Important） | R3 说编译器挡住整集合的写，只有变化的类型让整集合赋不进去才成立；今天行与表格的写的类型是 `Partial<IssueDetail>`（`all-properties.tsx:37`、`label-column.tsx:20`），`updateBody` 放行 `module_ids`（`issue-write-fields.ts:35-36`），一个 `Partial<IssueDetail>` 的值照样能把整集合带进请求体 | spec §2.2、R3：变化的类型声明 `module_ids?: never`；写入单元的命令逐个点名字段组出 `IssueUpdate`，不写 `...change`；`updateBody` 的白名单只给命令自己算出的整集合用。plan T3：类型测试另用一个类型为 `Partial<IssueDetail>` 的值（不只对象字面量）；变异 `t3-mut-spread-change`（命令以 `...change` 写请求体）；Review Focus 第 1 条 |
| F2（Important） | `cycle-create.test.tsx`、`module-create.test.tsx` 在 `:66-68` 把 `IssueDefaultProperties` 换成 `() => null`，它们的断言只靠 store 的注入成立；表单优先之后若初值由字段的组件去填，这些测试或失败、或要改，就不再是 R2 的回归保护 | spec §2.6：初值在弹窗一级（`base.tsx` 或 T4 拆出的新建命令）在表单挂上之前并进默认值——调用者给的值中是选项的，否则本页可加入时为本页；开关关闭时没有这个键。plan T10（原 T9）：RED 含字段换成 `() => null` 时请求体仍带本页；`t10-mut-page-overwrites-form` 与新的 `t10-mut-prefill-in-field` 落在这一处；回归边界写明这两个文件的替身与它们一行不改仍通过 |
| F3（Important） | 原 T5 一个 Task 里有规则、选项的读、名字的按需读、迭代下拉框、侧栏与预览的两栏、“已归档”的说明与五个键的删除，超出一个实现者的量 | 拆为 T5（规则模块、选项的读、迭代下拉框与侧栏的迭代一栏——它们的第一个产品读者）与 T6（名字的按需读与它的 SWR 键、`.oxlintrc.json`，“已归档”的说明，预览的一栏，i18n 的删除）。计划为 16 个 Task，其后顺延；检查点仍在中间（T9，原 T8）；spec 与 plan 的 Task 号与收据名随之改 |
| F4（Important） | 原 T11 让分组头经命令宿主请求新建弹窗，但宿主只在 `holds(dialog, dialog.data.project_id)` 时渲染（审者引 `command-host.tsx:165-170`、`:199`；HEAD 上是 `:160-165`、`:196`），分组头的 payload 多半没有 `project_id`（按状态分组是 `{state_id}`），弹窗永远不显示；锁定项目与 `DialogStore` 的类型也没有写 | spec §2.7 与 plan T12（原 T11）写明宿主改三处：(a) 加一种带分组 `data` 与 store 类型的“新建”请求，对话由路由的项目持有，按项目分组时由分组的 `project_id` 持有（起草时另补：没有路由的项目的个人页由工作区的权限身份持有）；(b) 宿主按 `isPlanningPageStoreType` 给出 `isProjectSelectionDisabled`，同今天的两个分组头（`list/headers/group-by-card.tsx:99`、`kanban/headers/group-by-card.tsx:63`）；(c) `DialogStore` 包括分组头会给“+”的每一种 store 类型（按 `viewFlags.enableIssueCreation`：项目、迭代、模块、个人页）。RED：项目页按状态分组时“+”打开弹窗、活过核验；变异 `t12-mut-hold-by-data-project`、`t12-mut-host-no-lock` |
| F5 | spec §2.11 漏了 P6b spec §2.10 的慢 runner 一句（×1.31，M6-M8-01）；主控另给出新的数据点：43549053092 的 e2e 作业 27 分 41 秒 | spec §2.11：补回慢 runner 一句；记下 27 分 41 秒与 I4 的修复；上界改以已测到的最慢的数据点算（约 29.2–29.4 分钟），直说在那样的 runner 上可能越过 30 分钟；T9 的检查点是校准，估算、上界或任一次实测越过 30 分钟时停下报告负责人、不推送。spec §5：P7b 与 P7c 两段合计约剩 15–37 秒本机 chromium，e2e 作业的结构（例如分片）是负责人的决定、两段多半要提出来。plan 的“检查点与证据”与 T16 Step 1 随之改 |
| F6 | spec §2.8 说“表单为空的两个值在草稿里同样省掉”，与 `draftPayload` 对 `module_ids: []` 的处理和清空草稿模块的需要相矛盾 | 主控裁定：`draftPayload`（`core/services/issue/draft-input.ts`）不改。去掉迭代是带 `cycle_id: null`（或 `""`）的修改，键随之从存下的 payload 去掉（修改里有的键以 `Object.hasOwn` 盖过 `before`）；`module_ids: []` 照留照发（契约允许空集合，`[]` 正是清空草稿模块的写法，省掉键反而留下 `before` 的集合）；新草稿没有迭代时没有 `cycle_id`，可以带 `module_ids: []`。spec §2.8 删去那一句、写成一条规则，R6 注明；plan T13：RED 钉住两点，变异 `t13-mut-null-cycle-draft`、`t13-mut-empty-modules-omitted`；回归边界写明逐字核对新草稿请求体的断言可以补上 `module_ids: []` |
| F7 | 弹窗按选项过滤初值时，两份未归档的列表可能还没答到，过滤会把值丢掉 | spec §2.6、§2.7：弹窗带着项目打开时经同一份 SWR 键（`PROJECT_CYCLES`、`PROJECT_MODULES`）读两份未归档的列表，只在答到之后过滤（以改默认值的方式，不算改动），之前字段显示加载、值留着不丢。plan T10、T12 的 RED 与变异 `t12-mut-filter-before-lists` |
| F8 | 调用者给的 `data` 的过滤与它的 RED 放在新建弹窗的 Task（原 T9），而第一个真正的调用者是复制 | 移到 T12（原 T11）；T10 只有本页的初值 |
| F9 | `core/store/issue/helpers/issue-list-binding.ts` 已 487 行，P7b 要往里加目录的工作 | spec §5 的 P7b：加之前先拆（D7） |
| F10 | plan 的表格 Task（原 T8）没有写两处必要的改动：新条目没有排序键时类型过不去、表头总给排序菜单与 `aria-sort` | plan T9：`SPREADSHEET_PROPERTY_DETAILS` 两项的排序键在类型上可省（`columns/header-column.tsx:39` 在没有条目时给 `null`，映射类型是同态的，显示属性的 Task T8 不给条目也能编译）；表头在没有排序键时只给标题与图标，不渲染 `CustomMenu`、不带 `aria-sort`（`spreadsheet-header-column.tsx:33`）。变异 `t9-mut-header-sorts`、`t9-mut-aria-sort` |
| 主控 | P6b 合入后的 main CI 43549053092 attempt 1 只在 I4 (page) 的表格故事失败，由修复分支 `codex/m6-p6b-ci` 处理；T1 原先只回填到 P6b 的 noff | plan T1：Step 0 先核对修复的 noff 已并入本分支（主控在 T1 之前做）；§7 回填最终分支 CI 43545459485、noff `954a1838`、43549053092 的失败（没有重跑）、修复分支（`0cfe405d` 的测试、`9ae1517f` 的文档）与它的分支 CI、noff、main CI，均以 P6b 的归档 `.superpowers/sdd/M6-P6b-integration-<日期>`（含修复的台账 `ci-fix/`）为准；修复的 main CI 没有全绿或归档不在时停下。M6-M8-01 的新段随修复合入，T1 只链接；设计 §14 的 P6b 行写上修复。spec 抬头与 §1 随之改 |

## 4. 审者核实为正确的事实（摘要）

审者的报告没有另存文件，下面只记主控转达、起草时又在 HEAD 上逐一核过的：

- 行与表格的写的类型、`updateBody` 的白名单（F1）：`all-properties.tsx:37`（审者引 `:36`，为同一处的 props）、`label-column.tsx:20`、`issue-write-fields.ts:35-36`。
- 迭代页、模块页的新建测试的替身与断言的行号（F2）。
- 命令宿主按 `data.project_id` 持有对话、两个分组头锁定项目的行（F4；宿主的行号以 HEAD 为准改过）。
- `issue-list-binding.ts` 487 行（F9）；表格表头的两处（F10）。
- 43549053092 的 e2e 作业号、起止时刻与唯一的失败（F5）。
- 其余 R1–R21 与 A1–A12 没有被指出事实错误；没有守卫会在 Task 的次序里变红而计划不修（F3 拆分之后的次序同样）。

## 5. 负责人须知

开工不须负责人决定任何事。R1–R10 是主控在已批准的设计之内、写 spec 之前的裁定（D1–D10），R11–R21 都在设计与 P5a–P6b 的先例之内；F6 的草稿规则是主控的局部裁定。本段只改网页，不改 `server/`、`api/`。

需要负责人知道的两件事：

- **e2e 的时长**：P6b 合入后的 main CI 在一个慢的 runner 上 e2e 作业用了 27 分 41 秒。本段新增两个页面故事（本机 chromium 约 20 秒）之后，以那样的 runner 算上界约 29.2–29.4 分钟，可能越过 30 分钟的报告线（期限 35 分钟）。T9 的检查点先实测一次（校准点），T16 推送之前再算估算与上界；估算、上界或任一次实测越过 30 分钟时停下报告负责人、不推送；改 CI 的期限或作业结构须负责人同意（M6-M8-06 的教训）。
- **P7b、P7c 的余量**：本段之后两段合计约剩 15–37 秒本机 chromium；e2e 作业的结构（例如分片）多半要由那两段提请负责人决定。
