# M4 / P16 `web-cleanup`：前端清理与守卫终态

状态：预检通过，已获授权实施，T1–T3 本地任务完成，T2 检查点 CI 待核。基线 `313728cbc61003d7ed072c1e1e191e1cc2a3d75a`（P15 集成修复后的 main 提交），分支 `codex/m4-p16`。本次按已通过预检的计划实施；各 A 项凭实际证据勾选，`tools/keywords.json.phase` 已推进 `M4/P16`。阶段完成后 deep-review，先不合并。

## 1. 目标

完成 [M4 设计 §12 P16](../M4-design.md#p16-web-cleanup死成员no-shadow-与守卫的终态11-个任务) 的十一项：清理真实死代码、收紧共享下拉框的类型、全仓 `no-shadow` 清零并升错、降低各包警告上限，核关键词终态、S2 和 C12 准备。行为、权限、会话所有权、原生 API 和后续里程碑边界保持设计已批准的合同。

P16 完成不等于 M4 完成。控制者的全页面 C12、完整 §3.23/§14、规模数字、总体设计 9.4 以及最终交接关闭属于独立 `closeout`。

## 2. 交付物

路径缩写：C=`web/apps/web/core`，P=`web/packages`，E=`e2e`。现有路径的完整清单见 [准备盘点](../reviews/P16-web-cleanup-inventory.md)；新增路径明确写“新增”，不冒称已存在。

| Task | 文件与产物 | 核心边界 |
|---|---|---|
| T1 | P/{utils,constants}/src/tab-indices.ts、P/ui/src/dropdowns/custom-menu.tsx；M6/M7 交接子记录 | 数真实消费者，按最后读者决定保留/删除；实施起才推进 phase |
| T2 | P/ui/src/dropdowns/{helper,custom-select,custom-search-select,custom-menu}.tsx、P/types/src/common.ts、实际类型/渲染消费者；新增泛型类型与行为检查 | 不以 any、类型断言或字符串化掩盖值/回调错配 |
| T3 | `.oxlintrc.json`、全仓 `no-shadow` 文件、相关包 package.json | 变量词法含义不变；共享事件、序列化属性名与 API 不改 |
| T4 | 原附录 AST 的 M4 候选及历史 52 行；新增 `reviews/P16-dead-members.md` | 真死成员连同独占分支删除，保留逐行写实际读者 |
| T5 | M4 领域与本 M 已改文件的剩余告警、各包 package.json | 同提交精确下降 cap，不改 lint-cap 计数规则、不忽略文件 |
| T6 | tools/keywords.json、既有 samples、关键词核验记录 | 旧工作项地址和否定筛选终态；无 M4 copy-hook 例外 |
| T7 | C/store/member/project/project-member.store.ts 及实际读者；M7 子交接 | 不删除 M7 尚在使用的 getter，不误判同名局部数组 |
| T8 | README.md、docs/v0/frontend-changes.md、docs/v0/plane-diff.md、输入 handoff 子回执 | 内容反映 P12–P16 实际能力，M4 总状态仍开放 |
| T9 | E/stories/smoke/s2-web-app.spec.ts、现有消费者测试与发现的必要缺口 | 原生请求精确核对、权限四身份与 S2 终稿 |
| T10 | 新增 `reviews/P16-C12-preparation.md`；必要时新增 E/playwright.m4-cleanup.config.ts 与脚本 | 显式三个项目，逐路径/身份/焦点矩阵；最终 C12 留收尾 |
| T11 | spec/plan/review、整分支 deep-review、七门禁、准确分支 HEAD CI | 完成后停在集成待办，另候合并安排 |

### 2.1 T1：焦点索引与最后读者

当前 `getTabIndex` 的七个生产文件均在 M6/M7：cycles/form、modules/form、views/form，以及 inbox 创建弹窗的 create-root、issue-properties、issue-title、issue-description。M4 原表单读者已迁移；共享函数、常量和 CustomMenu 的 `tabIndex ?? customButtonTabIndex` 分支目前保留。不得为达到“零命中”删除后续领域行为或放宽焦点守卫。

实施重新计数，额外核枚举、两份 barrel、CustomMenu 直接 tabIndex 的消费者。只有确无任何实际消费者的独立成员可删除；整函数/常量文件/菜单分支须最后读者消失才删除。把精确路径、所有者、删除条件写进 M6/M7 的现有输入交接，保留到 closeout 汇入 M4-closeout；不提前新建最终交接或关闭其 status。

### 2.2 T2：共享下拉框类型

历史“13 个调用方”不是执行上限。本基线两个组件共 32 个生产文件、34 个 JSX 调用点：CustomSelect 20 文件/20 点，CustomSearchSelect 13 文件/14 点（一个文件同时使用两者）。还要核类型引用、包装器及 `.Option`/`.MenuItem`，不能只改十三处编译碰巧报错的地方。

- `CustomSelect<T>`：保留任意实际选择值 T（主题使用对象，角色/日期等有数字或联合字面量）、可空的当前 value、`onChange(value:T):void`；`Option<T>` 的 value 同样强类型。独立 JSX children 的类型不能可靠证明父子泛型相等，不能声称已获得这种保证；具体消费者和运行测试负责真实选项对应。保留当前 Listbox 选择语义，不把对象 JSON 化或强转字符串。
- `CustomSearchSelect<T>`：实际选项为可作 React key 的 string/number，`ICustomSearchSelectOption<T>` 同步传递 value 类型。以 `multiple:true` 区分 `value:T[] | null`、`onChange(values:T[]):void` 与单选 `multiple?:false`、可空当前值、`onChange(value:T):void`。旧调用允许的 undefined 空态在拥有者处显式处理；不让任意 boolean 或 T/T[] 联合回调继续冲淡判别。泛型默认值不许回退 any。
- 单选 Combobox 的 null 表示清空输入或内部空态，沿既有行为不外发、不关闭；合法空串/0（以及 CustomSelect 的 false）不能被真值判断误吞。多选空值规范为内部空数组，选择/取消返回新数组且继续开着；单选选择后关闭。保留搜索 query、loading/options=undefined、空列表、disabled option、footer、onOpen/onClose 各一次、defaultOpen 和 focusSearchOnOpen。
- 删除当前 helper.tsx 八处、custom-search-select.tsx 两处 any 及 common.ts 的选项 value:any。`menuButtonOnClick` 实际是无参展开通知；MenuItem.onClick 实际收到 `React.MouseEvent<HTMLButtonElement>`，按真实派发定类型，不能一律改无参。无新增 any/双断言/ts-ignore；第三方适配在内部按判别分支传递，不用泛型外观包住 any 实现。
- 调用方明确单/多模式并移除原值链中的 any（如 MemberSelect.value），保持清空/none/custom 哨兵和服务端请求语义。字段 invalid/describedBy、IME/229、Enter 不提交外层表单、Escape/Tab/回焦、portal 外层隔离均保持。

新增 ui `src/dropdowns/select-contract.typecheck.tsx` 由既有 ui tsc include 真实编译，覆盖正例与带理由的 `@ts-expect-error` 负例：string/number/对象、单/多值、错误 callback/option，禁止运行负例，也不从生产 barrel 导出该文件；沿 api-client 的未调用类型检查函数模式。若需让 knip 识别该专用输入，只在既有项目配置中精确登记编译入口并证明 tsc 确实编译；不扩 ignore。运行测试使用真实 Listbox/Combobox，不 mock 掉被验泛型组件或只断言 props。

### 2.3 T3：全仓 no-shadow

只读基线显示 38 条：web 14、editor 16、propel 4、ui 3、utils 1，涉及 27 个文件。另有 8 个文件各一处 no-shadow 屏蔽注释；在忽略的准备目录中对逐字副本仅用空格替换这些注释，以同一锁定工具/规则扫描，发现隐藏 8 条（6 文件），另外两处注释已无作用。合计当前已知 46 条/33 个诊断文件，连同无效屏蔽共 35 个待核文件；源码未改，详细位置和对照 raw 见 inventory 与 shadow-suppression-scan.json。旧设计 128 是历史数，不能当当前待办。

按词法绑定重命名，保持 shorthand 的序列化键、回调入参含义、闭包、setter、枚举与导出的 API 名。删除全部既有 no-shadow 屏蔽（含已无效两处），重新枚举解除屏蔽后的诊断；不能只修显示的 38 条。同步处理本 Task 所碰文件的其它告警；依赖/事件变化必须有真实行为回归，不能把 effect 依赖整行忽略。最终在根规则设置 `"no-shadow": "error"`（实施用真实诊断核准），所有包/tools/E 真实扫描为零，并扫描确认无 no-shadow 专项或全规则 disable 残留；正反例验证新嵌套遮蔽报错、合法不遮蔽通过。禁止通过 ignorePatterns、override、eslint-disable、ts-ignore 或上抬 cap 达标。

实施局部裁定：旧 UI Dropdown 的唯一生产消费者是 views/form 使用的 LayoutDropDown。以现有 CustomSelect 接管该包装组件，保留布局枚举、onChange 与禁用布局合同，删除失去读者的旧 dropdown 目录、专属键盘 hook 与 barrel；真实键盘选择/关闭与同挂载实例切换语言的回归先记录缺口。此为既有共享 UI 清理，不新增状态所有者、依赖或后端协议。

### 2.4 T4：死成员与历史误归属

P15 最终 AST 的 web/ 扫描输入与本基线相同，经 hash 核验后可用于准备：全仓 member 875、prop 369、export 21；原 domains 脚本 M4 member 295、prop 158，共 453 候选/123 文件。这里包含测试形状、第三方插件回调和按路径误归 M4 的其它领域。候选不等于死代码；export 不属于 `--rows M4` 计数，仍由 knip/实际读者判断。

T4 用 M1 closeout plan 附录原版 deadsym/domains/lib/program、当前 typegen 重新产出，禁止改 DOMAINS 或过滤测试来降数。每行记录 kind、文件+symbol（行号仅定位）、删除或保留、生产/测试真实读取位置、动态机制或具体协议、验证。测试行不一律豁免；EditorRefApi、TipTap/ProseMirror 插件结构、表格列配置、DTO 的动态/结构类型读者须具名核对；无读者的成员、分支、只为它存在的参数/导出/文件一起删除。

历史 52 行按四个源文件逐项追溯：workspace-draft-issues/base.ts 16、types/src/workspace.ts 24、types/src/project/projects.ts 9、workspace.service.ts 3。当前这四路径在 AST 中均无 member/prop 候选；这只是盘点，不证明所有成员都被删除。T4 对照原历史行/旧版本，分别写已删除的提交或现存真实读者，尤其 WorkspaceService 的 M7 view 方法仍保留。P14 的公开编辑器 ref、P15 的 M7 版本/视图合同不得被脚本误删。

完成线为 M4 新报告每行均有结论，不能只给一个“无新增”的旧 ledger；旧项换名/移动要跟到新位置。M5/M6/M7 的存活领域合同写清所有者与读者，不为本期重写其业务；真无任何读者的局部残留可根因删除。

### 2.5 T5：告警与上限

基线各包 cap 和实际数完全一致：web108、editor48、ui10、propel10、utils6、hooks3，其余有 lint 的包/E/tools 均0，总185。原 domains 的 M4 匹配60条；另核 `53a6a05b`（M3 closeout）至基线已改文件及 T1–T4/T7 后续实际改动。准备时 M4匹配、全局 no-shadow 文件、两个下拉框消费者、M4已改文件的并集有103条/55文件；加上八个屏蔽文件后为63文件、103显示+8隐藏诊断（含两处无效屏蔽）。它是初始影响表，不是可永久冻结的清理上限。

保持“本 M 改到的文件最终零告警”。语义/键盘问题在所属组件修，不塞空 handler/role、移除行为或 suppress；静态展示元素的错误交互应改真实按钮或明确无交互。editor 上传/图片未在 M4 启用，机械与局部组件清理不能顺便启用它或引入新上传合同。每次修复或删除同时按实际结果下调 owning package cap；规则升级造成 warning 变 error 不算清理成功。`tools/lint-cap.mjs` 保持警告数严格等于上限。

### 2.6 T6：关键词终态

T1 开始才改 phase 为 `M4/P16`，T6 再核终态。当前 old-issue-urls 和 negation-filters 均无例外，copy-hook 只余 M6五/M7四/M8一条；没有 M4 例外。保持现有正则、samples、范围与到期日；“只剩 M5/M6/M7”允许零条，不能为符合历史描述凭空添加例外。M6/M7 现存不在 old-issue-urls 匹配范围的领域地址仍归原所有者，不能把此规则零命中声称全应用零旧 API。

准确核规则匹配、现存例外和真实请求三层，追加或复用必要正/反例证明 M4 旧地址/否定名会失败、原生 /api/v0 和现存合法词通过。P15 已删除的 `/search/` 方法虽不匹配此规则，仍以具名搜索合同扫描核零；不重建专用空服务。没有删除需求的新规则不凭空加，全局守卫不放宽。

### 2.7 T7：成员 getter

M4 issue-layouts/utils.tsx 的两处旧 `projectMemberIds` getter 读者已消失；当前 M4 的同名局部变量来自显式 projectId 的 `getProjectMemberIds(projectId,false)`，不可误删。M7 仍有三文件实际 getter 读者：项目视图 mobile-header、views/view-list-header、inbox-filter/filters/filter-selection（两处）。保留 getter 和既有 M3 测试，在 M7 输入交接写明最后读者迁移后删除；不改目录/权限所有权。

### 2.8 T8：文档终稿与交接

README 的“P12 分支”“P13–P15 待验收”已是历史文字，本期改为准确 main 能力：四布局/富筛选/归档/个人三关系、详情/创建/编辑器、评论/表情/历史/草稿/命令搜索，以及共享组件边界。README、前端改动清单、差异清单互相对应，不宣称未实现的附件、迭代/模块/视图列表、通知/收集箱等已可用。

P16 对 M1-P3/M1-closeout/M3-closeout 写子项处理证据，M6/M7 接收 last-reader 的具体删除条件。八份 M4 输入 handoff 顶部仍 open；M5–M8 最终 M4-closeout、M4设计§14勾选/总体9.4完工只在收尾处理。

### 2.9 T9：消费者测试与 S2 终稿

保持既有 P12 basic-dropdowns、P13 rich-filter、P14 详情/表单、P15 协作搜索和 M2/M3 设置控件回归。T2 的类型负例与真实值选择、T3/T5 的具体行为变化优先补在拥有者旁边，不复制 FakeNerve 或完整服务世界。

S2 保留 admin/member/guest/project-non-member 四身份、实际冷加载的精确请求名单、200/404/403 与跨工作区错误地址；补齐 M4 最终页面入口覆盖表：项目列表、归档列表、个人三关系、草稿、编号/项目详情、命令搜索（后两者的动作合同可引用 I18/C7，不能假称 S2 冷加载包含动作请求）。对未在 S2 的主入口补真实页面分支与该阶段精确请求名单；访客个人页无列表/工具栏、访客草稿拒绝及非成员项目无工作项读取分别核。只按当前已实现合同维护 expected requests，不能加旧 API/任意失败白名单或忽略重复请求来凑绿。

### 2.10 T10：C12 准备

建立表：项目四布局与快速添加/peek、富筛选/归档/个人三关系、编号/项目详情/子项/关联/链接/编辑器、评论/表情/历史、草稿创建/编辑/发布、两处命令搜索宿主；每行绑定已有故事或明确新缺口、账号身份、三浏览器、焦点轮廓不裁切、键盘与右键菜单、Back/关闭、请求与控制台验证。P12–P15 config 有重叠，合并清单按实际 test id 去重，默认 Firefox 仅 A4 不能代替矩阵。

必要时新增只组合既有故事的 `playwright.m4-cleanup.config.ts` 和 `test:m4-cleanup`，`--list` 必须三个引擎各有同一范围的非零测试；配置复用 base 的数据库/进程生命周期，不重写夹具。跑受 T2/T3/T5/T9 影响的三浏览器回归并记录实际 executed，不必在准备任务重复无变化的完整旧矩阵。可补当前实际遗漏的焦点/右键测试，但三浏览器全页面控制者最终 C12 保留到 closeout，在矩阵中明确“准备/自动回归已做，最终控制者核对待办”。

### 2.11 T11：终审与交付

完整本地七门禁串行，记录准确输入 SHA/dirty hash、raw/exit、运行/跳过数；整分支 deep-review 覆盖 baseline..candidate 的逻辑、集成、测试三视角，修复局部问题并限定复审。静态清理不强造行为红；需要保护的类型/守卫/行为应有真实负例或变异，至少抽查十五个有意义断言，保留精确恢复。

最终分支 HEAD 与远端一致，三个 CI job 全绿后才称阶段完成；若失败按范围定位修复重验。按照负责人要求先不合并。获后续授权再 --no-ff 集成、验 main CI、归档/清理。

## 3. 准备期决定

| 决定 | 裁定与理由 | 错判代价与验证 |
|---|---|---|
| D1 | 保持设计十一 Task；历史数量改为当前静态清单，不改业务范围 | 设计要求每段约16 Task/计划约1500行以内；单Task实现约1500行作为本期复核参考，超范围先按职责重切 |
| D2 | 两个 last-reader 接口暂留，M6/M7 具体接收 | 提前删会破坏未迁领域；T1/T7 当前读者/全类型/存活回归 |
| D3 | 下拉框区分任意值 Select 与 string/number 搜索值，单/多模式判别 | 类型窄化误伤主题/空态；T2 正负编译及真实消费者 |
| D4 | 死成员按原脚本全部候选逐行审，不按总数裁切 | 脚本误报导致协议/插件损坏；T4 真读者与对应回归 |
| D5 | 全仓 no-shadow 与 M4 全期改动文件告警一起核；cap 只降 | 仅警告转错误会假清零；T3/T5 实际 diagnostics与正反例 |
| D6 | old-issue-urls 当前零例外即合法终态，保留未来领域边界 | 不增加无意义例外，不用规则零命中冒称全页面接通 |
| D7 | C12 先准备矩阵和受影响回归，最终全路径控制者核对归 closeout | 准备和最终执行必须分列；T10 与收尾交接 |
| D8 | 本次用源码/类型/告警/历史证据静态预检替代产品原型与重放 | 符合“暂不实现”；不将 P15 绿或静态扫描记成 P16 产品验收 |

这些是已批准设计范围内的落地决定。若需要改接口含义、跨模块业务责任或发现架构回归，先提交已完成工作再上报，不以本表预先授权。

## 4. 验收标准

- [x] A1：T1 当前 getTabIndex/常量/菜单完整读者清单，M4零读者，存活接口及 M6/M7 删除条件准确。
- [x] A2：T2 无点名 any，泛型/单多判别和所有当前消费者编译；类型负例及空态/数字/对象/真实键盘回归通过。
- [x] A3：T3 全仓 no-shadow 为0且升 error；全部8处既有屏蔽已删、无专项或全规则disable绕过；合法/遮蔽正反例有效，词法与序列化含义不变。
- [x] A4：T4 新报告每个 M4 候选与历史52行可追溯；449 行中删除91、保留358，读者与动态合同见 [逐项审计](../reviews/P16-dead-members.md)，类型/knip/全前端与 M11/M12 见 review §12；T11 继续核新增候选。
- [x] A5：T5 M4领域及本M实际改动文件零告警，各包实际警告等于下降后的cap，无抬高/抑制。
- [x] A6：T6 phase=M4/P16，旧URL/否定筛选及M4复制例外终态符合设计，正反例和完整关键词门禁通过。
- [x] A7：T7 M4旧getter读者为0，M7三文件及getter/测试保留，最后删除条件交接完成。
- [x] A8：T8 README/两清单/子交接逐句为真，未提前关闭M4或未来领域。
- [ ] A9：T9 实际消费者缺口补齐，S2主入口和四身份精确请求/拒绝/无旧地址通过。
- [ ] A10：T10 C12表完整，三浏览器同范围非零清单、受影响回归有实际证据，最终C12仍明确留收尾。
- [ ] A11：T11 本地七门禁、deep-review修复复审、最终准确HEAD CI和证据清单完成；交付后先不合并。

## 5. 后续范围

不新增服务端字段/接口、不改 IssueRootStore/PermissionReads 所有权，不实现 M5 文件、M6迭代模块、M7收集箱/通知/视图/完整工作区标签或 M8 功能。P10a 决定A继续有效：保留的未接通领域自身页面由拥有者验收。输入 handoff 总关闭、M5–M8 `handoffs/M4-closeout.md`、全M4终审、§14与总体9.4、最终C12和性能数字归 closeout。

## 6. 风险

主要风险是以候选数代替读者证明、泛型表面收紧但用 any 逃逸、对象/空态/多选回调被改义、机械重命名改了对象键、为清 lint 损坏 effect/键盘、脚本/矩阵空匹配及把准备写成完成。分别由 T4逐行ledger、T2编译+运行、T3局部差异与负例、T5真实行为、T9精确请求、T10清单/执行与T11证据审计处理。

## 7. 交接与关闭落点

| 输入 | P16 子项 | 仍开放 |
|---|---|---|
| M1-P3筛选显示 | T6否定守卫终态 | 顶部总关闭留closeout |
| M1-closeout死成员/oxlint | T3–T5原报告、实际读者、全仓规则与cap | M4汇总和最终数字 |
| M3-closeout§7/§8 | T1泛型/索引、T2、T4历史误归、T7、T9/T10 | M6/M7存活读者；控制者C12 |
| P15 review§6/§13 | 总守卫/no-shadow、C12准备；复用已归档历史证据 | 评论增量窗口、M7旧合同等限制不扩大 |

## 附录 A：准备证据界线

主目录 `.superpowers/sdd/P16-preparation-20261101/` 保存实际源码检索、P15 AST等价来源、只读告警输出、builder-check、文档deep-review和静态核验。实施证据将位于 P16 worktree 的 `.superpowers/sdd/P16-web-cleanup/`；T1 起另记本阶段产品实现/测试证据。P15 main CI 和当前静态 inventory 只证明基线及计划输入，不勾上面的 A 条款。
