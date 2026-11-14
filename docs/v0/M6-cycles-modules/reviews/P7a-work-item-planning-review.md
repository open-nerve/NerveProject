# M6/P7a 阶段评审记录

> 实现与审查已完成，待分支 CI 与合入：整分支审查 Ready to merge（[re-73](../../../reviews/re-73-M6P7a整体审核.md)：没有 Critical、Important，三处 Minor 与六条观察），主控仍以修复轮 F（F1–F4）处理三处 Minor，经限定复审 Ready to merge（没有 Minor，re-73 §4）；最终候选 `52c04d69`（最后一个改代码的提交是 F3 的 `9b4a5f77`，改 e2e 故事；最后一个改产品代码的是 F2 的 `72c39da1`；`52c04d69` 只改文档），其后只有本记录所在的提交（只改文档）。最终 HEAD 的分支 CI、合入、main CI 与归档清理见 §7（由 P7b 的 T1 回填）。基线 main `954a1838bff44e98cc44f15df524da27e42fb1d8`（P6b 的 noff，双父 `06b72d4b`、`c0572192`）；P6b 合入后 CI 的修复 noff `7868a6b496d2587e2ed2cac7360c3ca6a86963be`（main CI 43557098427 attempt 1 通过）在 T1 之前并入（`4b93ffd0`）。

## 1. 评审方式与范围

P7a 交付设计 §12 的 P7a 一行（P7 因超出一段的上限在实施之前拆为 P7a/P7b/P7c，[spec](../specs/P7a-work-item-planning.md) §1、R1）：写入单元的范围必给、去掉 M4 起的 router 退路；模块集合的变化以增减在写队列轮到时作用于所持的集合，迭代是单值字段；按项目的规划目录（下拉框的选项在打开时读，值的名字按需读、含已结束与已归档的）；迭代与模块的下拉框；工作项的详情侧栏、预览、列表与看板的行、表格的两列与显示属性 `cycle`/`modules`；新建与修改的弹窗（表单优先、修改发出差异、拒绝落到字段）、分组头的新建弹窗归命令宿主、复制与草稿；要加代码的三个到上限的文件先拆；G4、G11 的页面故事（工作项一侧）。列表词表与“添加已有”在 P7b，命令面板、移动端页头与遗留在 P7c（spec §1、§5）。

- **调研**：两个只读调研代理在 `c0572192` 的树（即 main `954a1838` 的树）上写出现状笔记（`research/web.md` 606 行：网页的现状、写入单元、行、弹窗、草稿、命令宿主、目录的读、交接各项 A1–A24 与 X1–X2、e2e 与守卫、拆分的估算；`research/plane-contract.md` 375 行：Plane 的工作项一侧、nerve 的契约、差异与风险），随台账归档。
- **界定**：主控在起草 spec 之前裁定十件事（D1–D10，台账 `controller-rulings.md`）：P7 拆为三段（设计 §8 的四个视图文件交 P7c）；表单优先于本页；模块集合以增减在轮到时算、迭代是单值；选项与值分开；跨项目的行照状态下拉框按项目读；草稿不发空的 `cycle_id`、换项目清空；三个到上限的文件先拆；Plane 的缺陷不照搬；e2e 预算约 20 秒本机 chromium，估算或上界过 30 分钟即停下报告负责人；只改网页。起草代理据此写 [spec](../specs/P7a-work-item-planning.md)（R1–R21，R1–R10 照录 D1–D10；A1–A12）与 [plan](../plans/P7a-work-item-planning.md)（15 个 Task）（`ab4ef06b`）。
- **预检**：实施前一次 fresh 只读预检（[re-72](../../../reviews/re-72-M6P7a预检.md)），结论为修正后可实施：四处 Important（F1 变化的类型声明 `module_ids?: never`、命令逐个点名写请求体、类型测试用 `Partial<IssueDetail>` 的值；F2 新建的初值在弹窗一级并进表单的默认值，P5b、P6b 的新建测试因此仍是 R2 的回归保护；F3 原 T5 拆为 T5、T6，计划为 16 个 Task；F4 分组头的新建经命令宿主时宿主改三处）与六处 Minor（F5 慢 runner 与 27 分 41 秒的数据点、P7b/P7c 的 e2e 预算；F6 草稿的 `draftPayload` 不改、`module_ids: []` 照发；F7 初值的过滤等两份未归档的列表答到；F8 调用者 `data` 的过滤移到 T12；F9 `issue-list-binding.ts` 交 P7b 先拆；F10 表格两列的表头的两处改法），主控全部接受，另令 T1 回填 P6b 合入后 main CI 的失败与修复分支，在实施前改入（`0f3b5dc4`）。P6b 的 CI 修复合入 main 之后，本分支以 `git merge --no-ff` 并入它（`4b93ffd0`）。
- **实施**：T1–T15 由实现代理完成，主控逐 Task 复核并记台账。T2 先于 T1（T1 只改文档、要等修复的 main CI 与 P6b 的归档；预检核过没有在 `M6/P7` 到期的关键词例外，次序不影响守卫）。拆分与只改接收者的 Task（T2、T4）先以既有测试留基线收据，结束时逐文件同名同数。主控复核时另要了三个修正提交：T9 两格关上列表时焦点回到格子（`ca0f10be`）、T11 的 R17（`cc5dafea`）、T12 两份列表的读被拒（`16473c1b`）。
- **门禁与审查**：T9 一次检查点（七项门禁与分支 CI，CI 的实测作 T16 估算的校准点）；T16 在 `01a10e66` 上的七项门禁与三种引擎的页面配置，一次 fresh 只读整分支审查（[re-73](../../../reviews/re-73-M6P7a整体审核.md)），修复轮 F 另经一次限定复审；最终候选的七项门禁、三种引擎的页面配置，与推送前的 CI 时长估算（spec §2.11）。

## 2. 验收状态

A1–A11 满足（[spec §4](../specs/P7a-work-item-planning.md#4-验收标准)已勾选），A12 待合入后由 P7b 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 写入单元：`updateIssue` 的范围必给，`WritesRoot` 没有 `workspaceSlug`、没有 router 退路，`issue-writes.ts` 在 `project-address-route` 里（样例与变异）；8 个回归文件 32 处只加范围、断言不改（逐文件同数 272 例）；根 320 行、写入单元 302 行 | T2、T3 |
| A2 | 模块集合：增减在轮到时作用于所持的集合（连续两次、前一个被拒、已归档与不认识的 id 保留），迭代是单值；`IssueChange` 声明 `module_ids?: never`（`Partial<IssueDetail>` 的值赋不进去），`changeBody` 逐个点名；P6b 的“移出模块”经同一命令（三个测试文件不改） | T3、T7、T10 |
| A3 | 目录：选项在打开时经 `useDirectoryRead` 读、按项目与权限分范围、迟到的不写、读失败可重试；名字含已结束与已归档的，按需、一个项目一次，与包装层、归档页同键；跨项目的行按各自的项目；`use-planning-names.ts` 在 swr 限制里 | T5–T9、T12 的修正 |
| A4 | 下拉框：选项、值、“无迭代”、已归档的已选项只能去掉、模块的 `onChange` 是增减、以行为名、表格的格子关上时焦点回到格子；列表开着时换项目关上并改读新项目；M2 交接 §4、§5(a) 与 M3 交接 §3 的两个下拉框在三种引擎的浏览器里核对 | T5、T7、T9 的修正、T14、F3 |
| A5 | 各处：详情、预览、行、表格、显示属性（类型、常量、默认、本地存的页、显示面板）；开关关闭时不显示、只读时不可改、被拒时说明、表头没有排序 | T5–T9 |
| A6 | 弹窗：初值在弹窗一级并进默认值、等两份列表答到才按选项过滤（在途时两栏加载、保存禁用，读被拒时说明并可重试）、表单优先、只发一个请求、为空时省略；修改不预填本页、发出差异；拒绝落到字段（R17）；分组头的新建弹窗在命令宿主、活过核验；复制带源的可加入的值并本页的 | T10–T12、T11 与 T12 的修正、F1 |
| A7 | 草稿：表单与行的两个字段、换项目清空（换回打开时的项目时回到存下的值）、不发空的迭代（`module_ids: []` 照发）、行的模块增减在轮到时算、发布的拒绝落到字段 | T13、F2 |
| A8 | 拆分：`base.tsx` 452 → 307、`form.tsx` 432 → 334、`utils.tsx` 399 → 184，新件各在约 400 行以内，测试只改导入（基线逐文件同名同数） | T4 |
| A9 | G4、G11 的页面故事（三种引擎） | T14、F3 |
| A10 | 守卫：`project-address-route`（各 Task 的新件与改写的件）、swr 限制、告警上限 44、knip、关键词（phase `M6/P7`）、i18n（新键 `common.archived`、`module.count`，删去的五个键）、`root.store.test.ts` 21 个 store | T1、各 Task |
| A11 | 文档：P6b 的回填（阶段记录 §7 含 CI 的修复、spec A8、设计 §14）；交接回执（M2 §4、§5(a)，M3 §3 的两个下拉框，M4 M6-02 的写入与显示属性部分，M6-M8-08 的行数）；差异清单、前端改动清单、设计 §8/§12（P7 一行拆为三行）/§14；re-73 的记录 | T1、T15、F4、T16 |

## 3. 局部裁定

台账中 122 条 `Ruling:`（T1–T15 与三个修正提交 107 条、修复轮 F 7 条、主控 8 条）与主控在复核时的裁定，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **写入单元**（R3、R10；T2、T3）：`updateIssue(id, makeBody, drop?, operationScope)` 的范围必给，拖拽权限检查直接读给出的范围（原来 `??` 的右边已不可达）；测试的 32 处取接收者已有的 `.scope("ws", "A")`。变化的类型 `IssueChange`（`Omit<IssueUpdate, "module_ids"> & { module_ids?: never; modules?: ModuleSetChange }`）放在 `@nerve/types` 的 `issues/issue-change.ts`；逐字段点名的 `changeBody` 与模块集合的 `modulesAfter`（T10 移到 `core/lib/planning-directory.ts`，与写入单元、表单共用）在轮到时以所持的集合算，没有 `modules` 时不发 `module_ids`。tsc 由此找出一处整集合的写——没有读者的 `IQuickActionProps.handleUpdate`，删去；`root.store.planning.test.ts:146` 一处只把整集合参数改写为增减（断言不改）。T2 与 T4 各有一个 plan 写明的变异在既有测试里存活，各补一个钉住的测试，以同一变异 RED。行数：根 320 不变，写入单元 300 → 293 → 302，请求体的纯函数 75 → 105 → 98（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)）。
- **三个文件的拆分**（R7；T4）：`base.tsx` 拆出描述的读 `useDescriptionRead` 与新建、修改、发布三个命令的 hook，弹窗一级的提交次序与关闭留在 base；`form.tsx` 拆出提交件 `useFormSubmission` 与页脚 `IssueFormFooter`；`utils.tsx` 拆出分组拖放 `group-drop.ts` 与表格列的 `spreadsheet-columns.tsx`。新件保留 Plane 的版权头（内容来自 Plane 的文件），新测试用 OpenNerve 的；9 个路径加入 `project-address-route`。
- **选项与名字**（R4、R5、R13、R14、R18；T5–T7，主控裁定名字的读由显示值的各面各自调用）：选项规则 `cycleOptions`（“无迭代”在首，未归档且未结束的）与 `moduleOptions`（未归档的，任何状态）是 `core/lib/planning-directory.ts` 的纯函数，经 `useDirectoryRead` 在打开时读（`core/hooks/use-planning-options.ts`，按项目与权限身份分范围，下拉框以项目为 key）；名字是 `usePlanningNames`（`core/hooks/use-planning-names.ts`）：store 叫不出且有角色、项目未归档、开关打开时，先读未归档的列表、仍叫不出再读归档，键与包装层、归档页共用。按钮名为“字段: 值, 行”（不加新键）；“已归档”的说明是新键 `common.archived`；多于一个模块时按钮显示个数（新键 `module.count`），触发器的名字另以 sr-only 带上各模块的名字。模块下拉框的每一下恰给一个增减，已归档的已选项在选项之后、只能去掉，去掉之后不再列出；写在途时照常可选（不照 Plane 禁用），显示的是 nerve 确认的值。R18 的五个键删去。
- **显示属性、行与表格**（R11、R20、R21；T8、T9）：显示面板与表格的开关门共用 `core/lib/planning-switch.ts` 的 `displayPropertyOn`（T9 从 T8 的面板移来，一条规则一处）；行的一件 `IssueRowPlanningProperties`（模块在前、迭代在后，以行为名）；`SPREADSHEET_PROPERTY_DETAILS` 的类型只让两项的四个排序键可省（省就四个一起省），没有排序键的表头只有标题与图标。T9 起初不接 `onClose`（选后焦点留在格里的按钮上），主控不接受，`ca0f10be` 给两个下拉框加可选的 `onClose`，焦点照别的列回到格子。
- **弹窗**（R2、R15–R17；T10–T12）：初值在 `create-command.tsx` 的 `useCreateStart`（表单挂上之前并进默认值，只非草稿的新建；本页的并法与“确认的工作项在本页”经 `planning-page-domains.ts` 的两列 `start`、`holds`，没有按域的分支）；表单的新建是 `IssueActions` 的新一项 `createIssueAsGiven`，快速新建照旧经本页的注入；请求体的两键为空或开关关时省略。修改以表单首次渲染时的模块为起点算差（`moduleSetChange`），经提交选项交给修改命令、在轮到时作用；带模块变化的修改不当作纯描述迁移。R17：主控取 A，表单有了迭代字段之后 409 `planning.cycle_completed` 落到迭代一栏，P5b `cycle-create.test.tsx` 页头一行改断言新合同（`cc5dafea`）。命令宿主的新建请求按组的 `project_id`、路由的项目或工作区持有，锁定项目由 `isPlanningPageStoreType` 推出，`DIALOG_STORES` 覆盖 PROJECT/CYCLE/MODULE/PROFILE。调用者给的值等两份列表答到才以 `givenOptions` 过滤、以 `resetField` 设为默认值（不算改动）；T12 起初在列表读失败时悄悄丢掉复制的值，主控不接受（re-72 F7：值留着、不丢），`16473c1b` 改为栏下说明并给重试、值留在弹窗、该域不发，重试答到后照样套用。
- **草稿**（R6；T13）：草稿的修改发表单的整集合（草稿 payload 整体替换，`module_ids: []` 是清空的写法）；换项目清空分两处（表单的效果清空两栏；提交件在项目不是打开时的项目时点名发两键，盖过打开之后行上写进的）；发布的拒绝照 R17 落到字段；内容复制在弹窗开着时开关被关掉、发布被拒、复制的草稿留着（设计 §3.3 B），以测试钉住。草稿的行的每个写都是“以所持的 payload 算的修改”。
- **e2e**（T14）：见 §5 的教训。
- **修复轮 F**：主按钮的禁用条件与两栏的加载同一个（表单换到别的项目之后，值是那个项目的、经那个项目的选项选的）；“存为草稿”在列表在途时仍可用（负责人决定 B：草稿保存只查结构），主控接受；F2 以项目一栏不是改动判断“回到打开时的项目”，与提交件的 `moved` 同一起点；F3 的一步放在行的模块控件上、修改之后（保住 spec §2.11 写着的修改请求体），主控接受；F1 的三处测试准备的改动（断言不改）主控接受。
- 其余见台账。

## 4. 实施记录

变异数为能编译的产品变异（e2e 的为能编译、能构建的），守卫（关键词、swr）的变异另记在台账；例数为网页包（`web`）的文件数/用例数之差。

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `ab4ef06b`、`0f3b5dc4` | spec 203 行、plan 261 行（15 个 Task）；预检 re-72 的修正（F1–F4 Important、F5–F10 Minor 与主控的 T1 回填），计划为 16 个 Task |
| 并入修复 | `4b93ffd0` | `git merge --no-ff` 并入 P6b 合入后 CI 的修复 noff `7868a6b4`（双父 `0f3b5dc4`、`7868a6b4`），T1 的前提 |
| T2 写入单元的范围（先于 T1） | `6f543e6b` | 范围必给，删去 router 退路与 `WritesRoot.workspaceSlug`，`issue-writes.ts` 加入 `project-address-route`；8 个回归文件 32 处只加范围、逐文件同数（272 例）；+1/1；变异 2 个；写入单元 293 行 |
| T3 模块集合的变化命令 | `bbbdaa80` | `IssueChange`、`changeBody`、`changeIssue`；列表、看板、表格、详情的写改用它，P6b 的移出经同一命令；删去没有读者的 `handleUpdate`；基线 60 个文件 932 例同数；+1/17；变异 6 个；写入单元 302 行 |
| T1 回填与阶段 | `c1921520` | P6b 阶段记录 §7（分支 CI、合入、第一次 main CI 的 I4 失败、修复分支与第二次 main CI、归档、清理）、P6b spec A8、设计抬头与 §14、前端改动清单、关键词 phase `M6/P7`；原件逐项核对（收据 `t1-*`） |
| T4 三个文件先拆 | `9225f6e1` | `base.tsx` 452 → 307、`form.tsx` 432 → 334、`utils.tsx` 399 → 184，新件 8 个；基线 12 个文件逐个同名同数，测试只改导入；+1/1（存活的变异补测）；变异 3 个 |
| T5 选项规则与迭代下拉框 | `43d53c35` | `cycleOptions`、`useCycleOptions`、迭代下拉框、详情侧栏的迭代一栏；+3/17；变异 8 个 |
| T6 名字与预览 | `34d902a0` | `usePlanningNames`、“已归档”的说明、预览的迭代一栏、R18 删去五个键；+2/13；变异 10 个 |
| T7 模块下拉框 | `45889a0b` | `moduleOptions`、`useModuleOptions`、模块下拉框、详情与预览的模块一栏、`module.count`；+1/21；变异 13 个 |
| T8 显示属性与行 | `adb53eb6` | 显示属性 `cycle`/`modules`、显示面板的开关门、行的两个属性；+3/21，两个页头测试的断言改为本页的显示属性（契约变化）；变异 9 个 |
| T9 表格两列（检查点） | `f1646370` | 两列的格子、表头没有排序、`displayPropertyOn` 移到 `planning-switch.ts`；+1/9；变异 11 个 |
| T9 的修正 | `ca0f10be` | 两个下拉框加可选的 `onClose`，两格选后与 Escape 后焦点回到格子；+0/1；变异 2 个 |
| T10 新建弹窗的两栏 | `2a8faa01` | `planning-fields.tsx`、`useCreateStart`、打开时读两份列表、`createIssueAsGiven`（表单优先）、请求体为空或开关关时省略；+1/13，`close-policy.test` 新草稿的 payload 补 `module_ids: []`（契约变化）；变异 15 个 |
| T11 修改的差异与拒绝 | `12070947` | `moduleSetChange` 从表单的起点算差、经 `changeIssue` 在轮到时作用，422 落到字段；+2/13；变异 8 个 |
| T11 的 R17 | `cc5dafea` | 409 `planning.cycle_completed` 落到迭代一栏，P5b `cycle-create.test.tsx` 页头一行改断言新合同；+0/1；变异 1 个 |
| T12 分组头的弹窗与复制 | `5d3ec8c9` | 分组头的“+”经命令宿主的新建请求、`DIALOG_STORES`、`copyOf` 带源的迭代与模块、`givenOptions`；+2/9，`module-quick-actions.test.tsx:264` 照 R15 改契约；变异 14 个 |
| T12 的读失败 | `16473c1b` | `usePlanningListReads` 取代 `usePlanningListsAnswered`：读被拒时栏下说明与重试，值留在弹窗、该域不发；+0/4；变异 3 个 |
| T13 草稿 | `45b4c2ab` | 草稿表单的两栏、换项目清空、发布的拒绝落到字段、草稿的行的两个下拉框；+2/17；变异 11 个 |
| T14 页面故事 | `40cef888` | G4（138 行）、G11（167 行）；期望错误的 RED 2 个，e2e 变异 6 个都失败 |
| T15 文档 | `01a10e66` | 差异清单扩四行、新八行；前端改动清单改五行、新五行；M2、M3、M4 交接的回执；M6-M8-01、-03、-08 补记，新 M6-M8-13；设计 §8、§12（P7 拆三行）、§14；spec §2.11、§5 |
| F1 | `f113759f` | 两份列表在途时主按钮禁用（re-73 M1）；+0/1；变异 1 个；三处测试准备的改动（断言不改） |
| F2 | `72c39da1` | 草稿的项目回到打开时的项目时两栏回到存下的值（re-73 M2）；+0/1；变异 2 个 |
| F3 | `9b4a5f77` | G11 在行上点已归档的 Gamma 只移出它、确认后不再列出（re-73 M3，187 行）；e2e 变异 1 个、期望错误的 RED 1 个 |
| F4 | `52c04d69` | re-73 的记录、spec §5 的修复轮、spec 与 plan 的状态、设计 §14 |
| T16 记录 | 本记录所在提交 | re-73 §4（修复轮的限定复审）、本记录、spec §4 的勾选与 §5、spec 与 plan 的状态、设计 §14、前端改动清单 |

网页包从基线的 429 个文件 4972 例到 449 个文件 5132 例。

## 5. 门禁与 CI

- **检查点**（T9 之后，`f1646370`）：
  - `t9-gate`：`make gen-check`、`make lint-go`、`make test`（本机 922 秒通过，没有出现 M6-M8-05 的签名）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 470 过，W1 (page) 与 W6 (API) 以 “too many clients already” 失败，即 [M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03) 的签名，只重跑这两个文件一次 14 条通过（T15 记入该条）。
  - 分支 CI 43573631430 attempt 1 全绿：web 132383074803、server (0) 132383074931、server (1) 132383075050、e2e 132386476220（e2e 作业 08:13:18Z–08:32:50Z，19 分 32 秒）。这时还没有本段的页面故事，比基线 20 分 20 秒少 48 秒，作 T16 估算的校准点（T15 记入 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)）。检查点推送之后的提交留在本地，直到 T16。
- **三种引擎的页面配置**（`playwright.m6-pages.config.ts`）：
  - T14：两个新文件 6 例、`--repeat-each=3` 18 例；两个文件的时长 chromium 3.9 + 2.6 = 6.5 秒（预算约 20 秒）、firefox 5.6、3.9 秒、webkit 4.6、3.8 秒（6 个 worker）。整套第一次 117 例有 116 过：G11 在 WebKit 上预览的模块列表离按钮 44 像素，查明是描述编辑器约晚 0.25 秒挂上、属性一栏下移，不是本段引入（交 M8，[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)），故事改为先等上方的内容到齐，之后三种引擎 `--repeat-each=6` 36 例与整套 117 例通过。
  - 修复轮：F3 之后 G11 三种引擎 3 例（chromium 2.6、firefox 3.2、webkit 3.6 秒）、与 G4 同跑 6 例、chromium `--repeat-each=3` 3 例通过；两个新文件 chromium 合计仍约 6.5 秒。
  - T16：`01a10e66` 的 `t16-m6-pages` 与最终候选的 `f-final-m6-pages` 整套 117 例通过（各 2.3 分钟）。
- **T16 的门禁**（`01a10e66`，与整分支审查并行）：`t16-gate` 七项全过——`make test` 本机 948 秒通过，`make build && make e2e` 474 过（3.6 分钟，没有间歇失败）。修复轮在门禁跑完之后才开始。修复轮的收尾另跑一次全量 e2e（F1 改了保存何时可按）：473 过，F11 (page) [s3] 以 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 的签名失败一次，照该条的验收语只重跑该文件一次 7 条通过。
- **最终候选**：`f-final-gate`（`52c04d69020e4c6acb15a2b47fc200e6dbcaebbb`）——`make gen-check`、`make lint-go`、`make test`（本机 913 秒通过）、`make lint-web`、`make knip`、`make test-web` 第一次就通过；`make build && make e2e` 第一次在 18 秒时失败于 e2e 的全局准备：testcontainers 等 Postgres 容器的端口映射超时（“Timed out after 10000ms while waiting for container ports to be bound to the host”，`e2e/fixtures/db.ts:36`），一条故事也没有跑，是本机 Docker 的环境（[M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05) 的签名）；只重跑这一项一次 474 过（3.7 分钟，没有间歇失败，`f-final-gate-build-e2e-rerun`）。三种引擎的页面配置 117 例通过。
  - 修复轮的限定复审与这次门禁并行，结论见 §6。
  - 本记录所在提交只改文档，另跑文档链接、关键词与 `make lint-web`（收据 `t16-record-*`），其余以准确 HEAD 的分支 CI 为准。
- **CI 的 e2e 时长估算**（推送前，spec §2.11）：本段新的页面故事本机 chromium 6.5 秒（G4 3.9、G11 2.6，F3 之后不变）。
  - **估算**：基线 20 分 20 秒（修复合入后的 main CI 43557098427）加 6.5 秒 × 1.67–2.25、两个 worker 摊开，约 20 分 25–27 秒。
  - **上界**：最慢的 27 分 41 秒（那次还没有 I4 的拆分，加它约 5 秒本机 chromium 的份）加 (5 + 6.5) 秒 × 3.6–4.1，约 28 分 22–28 秒。
  - **校准点**：T9 检查点 19 分 32 秒，与基线之比 0.961，换算约 19 分 37–39 秒；慢的 runner（×1.31）约 26 分 45–48 秒。
  - 都低于 30 分钟的报告线，所以推送；实测超过 30 分钟仍先停下报告负责人。实际的 e2e 作业时长见 §7。
- **测试与 e2e 的教训**（交 P7b、P7c）：
  - **先等上方的内容再打开列表**（T14）：预览的描述编辑器、上次编辑与创建者一行在第一次渲染之后约 0.25 秒才到，属性一栏随之下移，react-popper 不随祖先的布局移动重算位置，已打开的列表会离按钮约 40 像素（[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)）。核对“列表在按钮旁”的故事先等这些内容与过渡结束（预览等描述的编辑器、上次编辑与创建者一行和 peek 的过渡；详情页等侧栏的 “Created by”；弹窗等描述的编辑器与 `transitionsEnded`）。
  - **路由在载入之后才装**（T14）：扣住回答、读请求体的路由（`holdAnswer`、`bodiesSentTo`）在页面载入之前装上时，WebKit 报 50 多条 “preloaded using link preload but not used”，控制台不安静；在页面载入、预览打开之后才装，之后只在同一页上开预览与弹窗（P6b 的同一教训）。
  - **可访问名的空格与次序**（T14）：值、说明、行各是 flex 里的一个盒子，Playwright 的名字计算在逗号、冒号、分号前多一个空格，故事以 ` ?,`、` ?:`、` ?;` 的正则匹配（G11 的 `modulesNamed`、`inRow`）；选项照 nerve 的次序（`sort_order`，新建的在前）写死，按钮名里的多个模块按 id（UUIDv7，即创建的次序）排，请求体以 `ascendingIds` 写出确切的次序。
  - **弹窗自己的读**（T14）：项目页上弹窗读到的工作项与页所持的不同时多一次列表的读，`dialogReads` 的序列只合迭代页、模块页；项目页的故事以 `answerTo` 等弹窗对 `GET /api/v0/issues/<id>` 的回答。
  - **e2e 变异要落在被证的一步**（F3）：主控举的 `t7-mut-archived-not-listed` 让故事在旧的一步就失败，证不到新的一步；“点 Gamma 发整集合”在没有在途的写时等价存活。取只有新的一步能挡住的变异（点已归档的已选项不发移出），没有能编译的产品变异时以期望错误的 RED 证实断言。变异之后以干净的源重建，`bin/nerve` 才不带变异。
  - **单测的 SWR 缓存**（F1）：同一文件的表单在同一会话读同一份键时，全局 SWR 缓存的去重会让后一个测试的新 store 拿不到列表（逐个跑过、整文件跑败）；渲染时给每个测试自己的 `SWRConfig` 缓存（`draft-planning.test.tsx`、`planning-copy.test.tsx`、F1 起 `issue-modal/fake-form.tsx`）。
  - **全局准备的环境失败**：T14 与最终候选各有一次 testcontainers 等端口映射超时（全局准备里，没有故事跑），都只重跑一次即过。
  - **e2e 预算**：本段两个新文件 chromium 合计 6.5 秒，少于约 20 秒的预算，P7b、P7c 两段合计约剩 28.5–50.5 秒（spec §5）；e2e 作业的结构（例如分片）是负责人的决定。
- **最终 HEAD 的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-73](../../../reviews/re-73-M6P7a整体审核.md)，`fable`，范围 `7868a6b4..01a10e66`，21 个提交、163 个文件，与 `01a10e66` 的本机门禁并行）逐项核对了：

- 写入（增减在轮到时算、逐字段点名、`module_ids?: never` 的类型测试、P6b 的移出经同一命令；修改发表单的差异；迭代是固定字段；草稿照 re-72 F6，对照接口合同；复制与调用者的值等列表答到才过滤；表单优先；命令宿主的新建）；
- 会话分代与迟到的回答（选项经 `useDirectoryRead`，名字与列表经带 `loginId` 的 `useSessionSWR`，写都经 `run`、`useDetailCommand`、`useDetailScope`）；
- 界面（选项与值分开、已归档的只能去掉、不重排、各面的开关、可访问性、只读）；
- 设计质量（文件的行数、没有值的循环导入、没有开关与兼容、类型由 `IssueUpdate`、`IssueDetail` 推出、守卫）；
- 测试（真实的 store 与 FakeNerve，plan 列的变异都有失败的收据，存活的两个补了测试，改过的回归断言都有理由，e2e 不重新载入）；
- 文档（Plane 的引用抽查 11 处，差异清单、前端改动清单、设计与交接回执与代码和台账一致）。

结论 Ready to merge：没有 Critical、Important，三处 Minor（M1 两份列表在途时保存会发未核过的调用者的值；M2 草稿的项目换走又换回时保存丢掉存下的迭代与模块；M3 G11 没走“Gamma 只能去掉”）与六条观察，没有须负责人决定的事。主控仍以修复轮 F 处理三处 Minor（同 P6b 的先例）：F1 加载中禁用主按钮（不取“加载中不发两键”，那会静默丢掉持有的值）、F2 换回打开时的项目时两栏 `resetField` 回到存下的值、F3 G11 在行上点 Gamma、F4 写 re-73 的记录；六条观察不需动作。

修复轮的限定复审（`fable`，范围 `01a10e66..52c04d69`，与最终候选的七项门禁并行）结论 Ready to merge，没有 Critical、Important、Minor：F1–F4 各只做所说的、关掉 M1–M3，主控接受的三项取舍照实现；F1 的条件恰为“在途的是表单所在项目的列表”，修改与草稿不受影响，产品代码没有动 store 与 SWR；F2 回到默认项目时两栏不再是改动、不发；F3 没有重新载入、锚是 nerve 确认的名字与记下的请求体；RED、GREEN 与变异的收据都在台账，三处测试准备的改动不掩盖行为，没有削弱的断言；re-73 的记录与链接相符。六条观察都不需动作（F1 只守按钮、F2 在打开时项目就关着某个域时的既有角落属设计 §3.3 B 等），见 re-73 §4。

## 7. 集成、归档与接续

（待回填：由 P7b 的 T1 按归档里的台账、CI 原件与 `cleanup.log` 回填，作业号与时刻以 GitHub 公开的 jobs 接口核对。）

- **最终 HEAD 的分支 CI**：待回填（含 e2e 作业的实际时长，与 §5 的估算对照）。
- **合入与 main CI**：待回填。
- **归档**：待回填。
- **清理**：待回填。

**接续**：P7b 列表词表与“添加已有”（G15 的工作项一侧：目录、按页的显示选项常量、偏好的筛选按页收窄、筛选/分组/排序/拖拽、空态与分组头与日历的“添加已有”，表格两列表头的排序；`issue-list-binding.ts` 先拆）、P7c 命令面板、移动端页头与遗留（G19 的工作项一侧，设计 §8 的四个视图文件，`fake-planning-page.tsx` 的 `opened`），各在 P7a 合入之后写自己的 spec、plan 与预检；P7a 留下的接缝与 e2e 的余量见 [spec](../specs/P7a-work-item-planning.md) §5。
