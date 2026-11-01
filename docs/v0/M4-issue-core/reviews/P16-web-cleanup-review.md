# M4/P16 前端清理：预检与后续实施记录

日期：2026-11-01。状态：预检通过，已授权实施，T1–T3 本地任务完成，T2 检查点 CI 待核。基线 main `313728cbc61003d7ed072c1e1e191e1cc2a3d75a`，分支 `codex/m4-p16`；[spec](../specs/P16-web-cleanup.md)、[plan](../plans/P16-web-cleanup.md)、[准备盘点](P16-web-cleanup-inventory.md)、[deep-review预检](../../../reviews/re-26-M4P16预检审核.md)。

## 1. 前置与方法

P15 的授权合并、准确 main CI、证据归档及正常清理见 [P15 review §13](P15-web-collaboration-drafts-search-review.md#13-授权集成main-ci-与归档清理2026-11-01)。本期分支从该 main 基线独立建 worktree，主目录仍 main；原生 worktree 工具不能指定负责人的路径，因此用系统 git worktree add。

预检时授权只有准备；按 02 §3 用源码/测试/类型合同、只读 lint 和 P15 AST 输入等价核对替代产品原型/逐 Task 重放。没有安装 P16 依赖、产品改动、P16测试执行或 phase 变更。原 AST 报告的 raw hash、tested input 中三份非文稿 dirty 文件已核实；后续集成修复不改变 web/，最终基线以 integration-baseline-equivalence.json 绑定；不把复用说成新运行。

brainstorming 要求的 fresh builder-check 只读完整 spec/plan/inventory 与相关源码，返回 `nothing important left`；主控按 deep-review 三视角顺序核验，不声称三位独立审者。builder之后由主控明确 no-shadow 根规则键、当前any数量、类型文件不进生产入口及新config的显式lint，核准config忽略项需用相同规则临时配置去除ignorePatterns；进一步对8处旧no-shadow屏蔽作同规则副本对照，补入隐藏8条及无效2处的删除验收，再核最终链接/状态；范围与产品均未改变。阅读范围与静态证据保存在主目录 `.superpowers/sdd/P16-preparation-20261101/`。

## 2. 验收映射

下表按实施证据逐项更新；准备通过不等于 A 条款完成。

| 标准 | Task与证据落点 | 当前状态 |
|---|---|---|
| A1 | T1完整last-reader表、M6/M7子交接与静态反例，见§9 | 已通过 |
| A2 | T2 类型/消费者/运行与反例见§10 | 本地通过 |
| A3 | T3 全仓0/error、词法/运行/反例见§11 | 已通过 |
| A4 | T4新AST逐行ledger、历史52行、动态读者/删除链 | 待实施 |
| A5 | T5 M4领域/全期改动零告警与各包精确cap | 待实施 |
| A6 | T6关键词、旧搜索具名扫描、正反例与未来边界 | 待实施/回归 |
| A7 | T7隐式getter/显式方法区分、M7三文件与原store测试 | 待实施/回归 |
| A8 | T8 README/两清单/交接子项与实际能力对账 | 待实施 |
| A9 | T9 S2入口/四身份精确请求与拥有者测试 | 待实施 |
| A10 | T10三引擎同范围清单、影响回归、最终C12留收尾 | 待实施 |
| A11 | T11完整F、deep-review/修复复审、准确分支HEAD CI | 待实施 |

## 3. 准备决定

spec D1–D8均为已批准设计内的落地决定，无待裁定项。保持十一Task，当前实际影响为32个选择器生产文件、46条已知no-shadow（38显示+8隐藏，含无效屏蔽共35待核文件）与453个M4 dead候选；历史数量保留来源、不拿它们当实施上限。共享last-reader接口留到M6/M7最后迁移；原生API/权限所有权不变。类型判别与UI运行、dead候选与实死、C12准备与最后验收分开记录。

## 4. 预检发现与修订

[deep-review报告](../../../reviews/re-26-M4P16预检审核.md)逐项列文稿修订。主要是补当前所有消费者和真实菜单event类型、保留主题对象/null/0/多选语义、约束类型负例进入真实tsc但不进入生产入口、全期改动文件告警与原AST逐行审计、0例外合法终态和C12边界。

没有产品缺陷被声称已修复，没有待决 Structural Issue/Needs Decision。T1/T7现存代码已符合“共享接口尚有读者则留”的选择，后续只需当前复核/交接，不能为凑任务重写接口。

## 5. 与原方法和设计的差异

遵用户“暂不实现”，不做产品原型/逐任务重放、不预填生成物或测试通过数；正式实施执行每Task五步与必要反证。设计十一粗项保持，但按基线数明确影响文件；32文件/34直接调用不是新增业务范围。所需共享泛型/no-shadow本就是P16批准范围，未扩大后端协议。

## 6. 移交事项

T1/T7把M6/M7真实读者与最后删除条件写入现有输入，closeout再汇成最终M4-closeout。M7视图服务/版本旧DTO、完整工作区标签、收集箱来源/系统创建者和lazy pending discard仍在各自阶段。全M4终审、控制者全路径C12、§3.23/§14、总体9.4、性能数字与八份输入顶部关闭继续留closeout。

## 7. 已知限制与待验证

静态候选453行尚未逐一完成T4裁定；103条显示+8条隐藏诊断的63文件待核范围不是固定上限，当前TS接口编译绿也不能证明泛型新增合同。类型子元素与父组件的泛型关系不能由独立JSX Option保证。现存M6/M7页面未接通的责任未改变，旧URL规则零命中不能代表全应用零旧请求。三浏览器准备还未运行P16测试，不能把P12–P15的旧矩阵当本期结果。

## 8. 准备交付核验

十一Task/五十五步骤和十一A均未勾，tools/keywords.json仍M4/P15；提交范围限文稿。相对链接、既有/新增路径、基线/主目录、实际盘点数字与未实施状态均由最终静态收据核对。准确准备提交及远端一致性以提交后的 `preparation-delivery.json` 为准，不能将文稿内预填SHA作为自证。

本地证据根 `.superpowers/sdd/P16-preparation-20261101/`：builder-check.json、baseline-inventory-summary.json、ast-input-equivalence.json、dropdown-callers.json、lint-baseline-summary.json及逐包原始diagnostics、planned-lint-scope.json、shadow-suppression-scan.json及原样/去屏蔽对照raw、历史/最后读者清单、source-review-inventory.json、static-preflight.json。产品实现证据从 T1 起另记于 worktree `.superpowers/sdd/P16-web-cleanup/`。

## 9. T1 焦点索引与实施入口

2026-11-01，以准备提交 `52a312f8` 为 Task BASE。独立 worktree 安装锁定依赖；phase 从 M4/P15 改为 M4/P16，不改规则/例外到期日。重新检索所有 tracked web TS/TSX，并用 TypeScript JSX AST 逐点核菜单属性与正索引；不把候选审计称为永久产品守卫。

| 实际读者（core/components/ 下） | 组/归属 | 处理 |
|---|---|---|
| cycles/form.tsx | PROJECT_CYCLE / M6 | 保留 |
| modules/form.tsx | PROJECT_MODULE / M6 | 保留 |
| views/form.tsx | PROJECT_VIEW / M7 | 保留；M6 共用弹窗宿主分工不变 |
| inbox/modals/create-modal/create-root.tsx | INTAKE_ISSUE_FORM / M7 | 保留 |
| 同目录 issue-properties.tsx | INTAKE_ISSUE_FORM / M7 | 保留 |
| 同目录 issue-title.tsx | INTAKE_ISSUE_FORM / M7 | 保留 |
| 同目录 issue-description.tsx | INTAKE_ISSUE_FORM / M7 | 保留 |

`ISSUE_FORM` 只有自身映射读取，M4 创建窗已自然 Tab；删除独占数组、枚举与映射共22行。四个存活组、utils 函数、constants/utils 两份 barrel 不动。44处 CustomMenu 直接 JSX 中没有 tabIndex 或展开属性，仅表格列头 header-column 有 customButtonTabIndex=-1；这个非正焦点用途仍保留。菜单默认0及 `tabIndex ?? customButtonTabIndex` 按批准 D2 保持，不能把旧收集箱父项入口当现存读者。两个字面正索引仍在 M6 的归档弹窗，写入 M6/M7 现有 M3-closeout 子回执，顶层 open 不变。

原始证据均在本 worktree `.superpowers/sdd/P16-web-cleanup/`：`t1-focus-inventory.json`（7个函数读者、44个菜单、2个未来域字面正索引），M01 给审计输入追加 M4 button tabIndex=1，被准确标为一条；源文件没有变异，无需冒称运行行为失败。`t1-baseline-dropdowns`/`t1-final-dropdowns` 原始收据及日志均22/22；`t1-final-build` 10任务成功；`t1-final-types` 22任务成功（含web typegen/tsc及共享包）；`t1-keywords` 76规则/13例外无命中。改动 constants 的 oxlint 0/0，diff --check通过。产品变更只有无读者组删除，未修改存活表单键盘/菜单行为。

## 10. T2 共享选择器合同与检查点

Task BASE `31733739`。`CustomSelect<T>` 保留任意值与可空当前值，`CustomSearchSelect<T extends string | number>` 把 options/value/onChange 按 multiple 字面量关联；内部两条 Combobox 边界明确单值与数组。菜单展开仍无参，item 为按钮 MouseEvent。common.ts 的 option 默认 string 保持其余旧读者。32生产文件/34调用逐项结论见 [inventory 的 T2 表](P16-web-cleanup-inventory.md#t2-实施消费者核验2026-11-01)，实际 TS checker 核 value/onChange 没有 any；独立 JSX Option 与父组件 T 的一致性仍由消费者回归证明，不能由该类型接口单独保证。

基线 null 多选崩溃的根因是公开合同允许 null，原 any 原样交给 Headless UI multiple 分支，其选项判定对 null 调 some；数组基线正常。现在在第三方边界将多选空态归一为 []，单选清空搜索仍不外发 null；主题对象引用、0/空串/false、菜单焦点/IME/表单语义保持。消费者将尚未加载的值归一 null，成员值去 any，rich filters 明确 string 数组与禁用哨兵。module status 回调按字段类型传递 status，消除一条遮蔽；web cap108→107，其余不抬高。

类型 fixture 被 UI tsconfig 实际包含，knip 只有该文件的具名 entry；未加入生产 barrel，函数不执行。基线 t2-red-types-corrected 是七个 unused expect-error；t2-baseline-values 为1失败/10成功（null多选）。终态 t2-values-r1 11/11、t2-consumers-runtime 42/42、t2-rich-sentinel 16/16，依赖构建/全类型通过。M02/M03 产生指定 TS2322；M04吞0/空串、M05多选误关闭、M06清空外发null、M07对象变字符串均有实际 AssertionError。M06首次先被缺失option助手拦下，不计有效断言；调整断言顺序后r2才计通过。所有变异原字节恢复，有补丁/输入输出hash。误在变异中采到的11条UI诊断不用于cap，恢复后是10条。

完整本地 F 七项串行通过，统一产品输入 hash `5c373a5a409a6197308b9f4abf8ef58cab37e94e3dcaa4b6b7e444d75e188c98`，每项输入/输出一致。gen-check、lint-go、Go test、lint-web(54任务)、knip、强制 test-web(18任务，零缓存)、E2E **279/279**，包含 C3 列表/表单/peek/设置与 C2 真实筛选及兼容夹具。默认 E2E 是 Chromium 主集加两条 Firefox A4，不是三浏览器矩阵。原始 t2-f1至t2-f7 收据、日志与 t2-full-gates-summary.json 保存在 worktree 证据根；本段文稿不改变受测产品。检查点提交后再核准确远端 HEAD/CI，结果写提交后收据和后续记录，当前不宣称该 CI 已完成。

准备提交52a312f8的 run41270383724已核server/web/e2e三项success；仅作为准备提交状态，不替代本产品检查点。

## 11. T3 遮蔽升错与共享旧链清理

Task BASE `f0e2ac3c`。预检46条中的module status已在T2修正；本期全包/tools/E先实测37条显示，再移除8处旧屏蔽，得到45条/32文件。38个词法绑定改名覆盖29文件：其中外层插件工厂改名一次清除四处view遮蔽；内部引用及shorthand的 element/editor/className/view 等key保持。两个私有Props类型改名而不改公共组件。另两处无效屏蔽一并删除；逐文件表见[inventory T3](P16-web-cleanup-inventory.md#t3-遮蔽与屏蔽逐文件处理2026-11-01)。最后根规则 no-shadow=error，全仓该诊断0、总error0，无专项或全规则disable绕过。

局部行为缺口均先由真实DOM回归证实：旧LayoutDropDown缓存旧翻译、Enter无法打开列表；日期/成员移除项只有div，没有原生键盘按钮；收藏夹Disclosure内又嵌套button。LayoutDropDown复用现有CustomSelect，仍传同一布局枚举、onChange和禁用布局集合，删除整个旧dropdown目录及独占键盘hook/barrel。knip随即找出UI的三个无读者依赖（@nerve/hooks、lodash-es、@types/lodash-es），已连同锁文件该importer九行删除；其余解析版本未改变。筛选项使用具名type=button，逐值移除与清全部传值保持；收藏夹拖动握柄移到Disclosure按钮之外。M6两张表单使用Modal的data-autofocus保留标题初始焦点；现存Tab组及M6/M7业务责任不变。cycle分组无成员/无标签项使用稳定key，默认tab映射内联。

四个新增拥有者测试共7项通过：layout同挂载语言更新/键盘选择和关闭回焦、两个筛选项精确剩余值/无表单提交、两张真实表单在Modal内初始聚焦、收藏夹无嵌套按钮且Space可展开。类型及格式/lint54任务通过；共享构建10任务、knip通过、强制完整test-web18任务（零缓存）通过。首次绿色尝试因脚本语法错误未发生产品写入，原样红记录保留；修复后的测试环境缺ResizeObserver导致一项失败，按现有jsdom测试补环境能力后7项通过，没有修改生产以迎合测试。E2E影响回归继续在T5/T10检查点执行。

M08一次性合法TS中的嵌套遮蔽只产生一条no-shadow/error并exit1；M09不同绑定无诊断/exit0。M10把插件返回对象的view键错误改成createView，真实Editor消费者在SideMenu创建时报dragHandleView is not a function（1失败/13未选）；这明确证明公共key没有随局部名重命名。变异字节已精确恢复，随后完整前端测试通过。

| 包 | T2上限 | T3实际/上限 |
|---|---:|---:|
| web | 107 | 79 |
| editor | 48 | 32 |
| ui | 10 | 4 |
| propel | 10 | 6 |
| utils | 6 | 5 |
| hooks | 3 | 3 |
| 其它含tools/E | 0 | 0 |

总计184→129；八条被屏蔽的隐藏诊断未计入原cap，不拿解除屏蔽后的临时192作为上限。所有本Task实际变更文件零诊断，完整M4期间文件与M4领域剩余仍在T5处理。原始证据在本worktree证据根：t3-before/unsuppressed/after-r2的逐包JSON与rows、t3-removed-suppressions、t3-suppression-final、t3-rule-examples、t3-M10补丁/恢复hash、t3-owned-ui-green、t3-shared-build、t3-final-lint-web-r2、t3-final-knip-r2、t3-final-test-web及对应raw。阅读发现的编辑器生命周期问题已记待核清单，后续T4/T5继续根因核验，不提前宣称整分支终审通过。


## 12. T4 死成员与历史 52 行

Task BASE `a3af031b`。typegen 后按 M1 附录原脚本重新扫描，未改变 DOMAINS、过滤测试或重新定义 dead。准备453行到T4起点449行的四项差异、每条声明和读者/验证、历史52行与实际删除提交见 [逐项审计](P16-dead-members.md)。449项中删除91项及独占链，保留358项；全仓 member875→801、prop365→334、export21→2。剩余两个 export 是编译负例入口，不执行。没有以候选为零要求删除公开 ref、合并 Commands、NodeView/Plugin 或结构类型合同。表格六列不再构造无人读取的第三参数，FilterInstance 的无读者 id 删除后同步删独占 uuid 依赖和对应 lock importer 三行。

历史52行分别在 P11/P14/P15 已删除，原始报告、文件内 owner.symbol 与删除 diff 逐名核准。WorkspaceService 四个 M7 view 方法仍有 store 读者和传输契约测试；类型文件的其它角色/引导声明保持，不能把旧候选消失写成整文件消失。T4 新增 ledger 是逐项审计表，行数来自449项记录而非组件职责扩张。

验证：`t4-static-final-types-r2` 11任务通过；删 uuid 后 `t4-knip-r2` 通过；`t4-web-tests` 18任务通过、零缓存；`t4-lint-web` 54任务通过。web cap79→78、utils5→4，其余不变，总127；custom-image/block 两条既有告警仍按已批准 T5 处理，不宣称 T4 所碰文件全零。`t4-final-ast` 的 M4 报告 SHA256 为 `70710c06dcc6fa61088e6b9e6ae5182bf61420071bc5734a823c126a5d174df3`；AST/knip/全前端输入相同 `2b814a18bb5c31087bba5116de2a811721429bcf95325b460219ade78422eae4`。lint 在补 ledger/精确 cap 后执行，原始 receipt 记录差异。此组输入还包含下节路由修复的三个未提交文件；不将它伪称仅 T4 提交的独立测试，T5 全门禁将验证合并输入。

M11 删除审计输入 D001，真实 ledger checker 报 missing rows；恢复449行及原脚本hash后通过。M12 将 TableView.contentDOM 改为 undefined，类型仍通过，但真实编辑器表格单元格消失、编辑测试失败；恢复字节后测试通过。该测试基线已绿，只补动态协议保护，不宣称原实现有缺陷。M11 为审计反证、M12 为真实 DOM 行为反证，补丁/红日志/恢复hash分别保留。对照原报告的第三方 code/codeBlock 合并声明被证明仍有类型消费者，尝试删除造成指定 TS2717 后原字节恢复，没有为了降数留破坏。

## 13. T2 检查点 CI 的 I13 失败与后续处理

准确提交 `f0e2ac3c599e4e79f4ca5c0c9aae6b756fa19d39` 的 [CI 41272608224](https://github.com/open-nerve/NerveProject/actions/runs/41272608224) server/web 成功，E2E 278通过、I13历史分页1失败，不能记为该检查点全绿。连接器取得作业124750553462日志与报告13070691374，原报告zip SHA256 `595f454dd220ab19b705b52072fa0430a450acdeeaa3bd020d09021f973ec3b4`。trace显示 cursor 第二页已成功返回，随后当前编号路由的验证读完成导致历史重新挂载，首页又覆盖成两项。

本地有效红 `ci-history-warm-route-red-r2` 证实：返回访问过的编号路由，SWR已有旧数据时 isLoading=false，但本次读取尚在 isValidating，页头/正文提前进入交互。两个路由宿主等待本次 lookup 完成，`ci-history-warm-route-green` 14条通过；首次红因假时钟尚未过 dedupe cleanup 未抵达被验行为，不计有效红。此为既有路由边界的局部修复，沿用原 owner 和读取合同，不改权限/分页协议。真实 I13 页面复验与准确后续提交 CI 归 T5，当前仍待验证。


## 14. T5 告警终态与局部行为修复

Task BASE `5b3472bd`。重新取全部14个包/tools/E实际诊断，按原 domains M4 与 M3收尾 `53a6a05b` 以来整个M4改动路径并集核验，T4之后剩余41条/26文件，不能只取P16增量。当前范围零诊断；总数127→86，web78→69、editor32→0，ui4/propel6/utils4/hooks3保持，所有其它包0。规则、ignore、豁免和cap算法未放宽；临时给web上限68/70均被原lint-cap分别以“超上限/必须下调”拒绝。

局部修复沿现有拥有者：CalendarStore 将原 reaction disposer 交回 IssueRootStore.retire；SideMenu 将 DragHandlePlugin 的真实销毁函数接回 ProseMirror view 生命周期；BubbleMenu 释放未结束选择的mousemove/mouseup并在editor切换时复位。真实编辑器测试证实旧监听/DOM未释放，恢复后通过。表格行列菜单原先任意键都阻止并关闭，改为只处理非IME的Escape并回焦；丢失drag marker时仍释放自身window监听。链接输入确认避开IME并阻止父表单提交，label与input关联，主动打开时显式聚焦。原静态外层拦点击删去；子项工具栏实际是折叠trigger的兄弟，不再preventDefault吞子按钮默认动作。

通知筛选为原生button，Enter/Space一次切换且不提交父表单。编辑器空白区域用原生“继续输入”操作把焦点交给正文，图片选择交给真实img；缩放握柄保留指针拖动并支持方向键和原生键盘激活。两条新增名称由宿主en/zh-CN文案传入轻量EditorLabelsContext，真实NodeView在切换名称后保持同一控件；不增依赖/网络/业务状态拥有者，不启用M4图片。该回归还证实旧size>2判断误把单张图当空文档，首次改尺寸才生成id导致控件重建；现在只跳过默认空段落，实际叶子块在挂载前获得既有id，旧迁移meta保持。

机械清理明确保留所读字段：autolink.value/href/from/to、slash section.key/title/items；使用findLast/flatMap/一次性FileReader事件，未改变序列化格式。各进度分布的唯一无负责人/无标签桶用固定key，列表骨架用既有不同分组数量作key。Editor NodeView共享同一真实工厂，T5生命周期用例拆到 editor-lifecycle.test.tsx，旧 editor-interaction.test.ts 仍保留原15条，避免复制一套假编辑器。

证据根同前：`t5-ci-page-build`与`t5-ci-i13-page`在相同输入上真实I13重复10/10；T2检查点失败原因见§13，后续CI尚待核。`t5-all-types-r2`跨包构建/类型11任务通过；`t5-editor-lifecycle-final`真实编辑两文件22条通过，后补marker收尾的`t5-last-hosts`7条通过；`t5-mutants-restored`日历/通知6条通过。M13恢复不可键盘访问div被真实按钮/激活断言拒绝，M14保留旧周起始偏好被实际星期断言拒绝；补丁/红日志/字节恢复在t5-mutations.json。最初Bubble测试直接卸载被第三方挪出的根DOM导致NotFoundError，不计行为红；加真实外层宿主后r2确实检出未移除监听。单图测试曾带已存id作为对照通过，最终恢复无id旧文档重现并修正根因，不掩盖失败。

冻结输入后完整 F 七项串行通过：gen-check、lint-go、Go test（455.775秒）、lint-web（54任务）、knip、强制test-web（18任务、零缓存）和默认E2E 279/279。七项输入hash均为 `97771a1fb8edf91f81b0e37ebd39c3653ab9a3894f4249b209c26ddc52ce50b7`，各自输入/输出逐文件状态一致；`t5-f1`至`t5-f7`原始收据与`t5-full-gates-summary.json`可复核。该默认集仍非三引擎矩阵。仅在F结束后更新本文稿；准确检查点提交的CI后置核验，不提前记全绿。

## 15. T6 关键词终态

Task BASE `c4881cf5`。phase仍M4/P16，原76规则/13例外与全部samples通过。旧地址八类（issues、work-items、archived-issues、draft-issues、draft-to-issue、search-issues、entity-search、user-issues）和否定筛选均0例外、0未覆盖命中；M4复制0例外。copy-hook余10条：M6的cycle quick-actions、module links/list-item与quick-actions、两个context-based cycle/module commands共5条；M7的inbox header、project view quick-actions、workspace两份view quick-action共4条；M8 webhook secret-key1条。对应M6/M7/M8 closeout到期日不改，另3条为analytics两条与来源品牌说明一条。

具名全web TS/TSX扫描：projectIssuesSearch无出现；searchWorkspace只剩workspace.service.test的“不再存在”断言，没有生产声明/调用。不能把它写成连测试字符串也不存在。规则不覆盖deleted-issues、cycle-issues、inbox-issues等独立未来领域地址；编辑器包自己的剪贴板也不属于web copy-hook范围。没有借清理扩大这些已批准边界。

M15/M16在真实git扫描可见的M4临时TS文件中分别加入旧工作项URL、allowNegative，原 `node tools/keywords.mjs` 均exit1并准确报告各一命中；原生/api/v0、issues-detail及合法同名片段正例通过。临时文件已删除，原规则/配置字节hash相等，整份守卫恢复通过。`t6-keyword-probes.json`、各原始receipt/raw和`t6-final-inventory.json`保存源码、诊断及复原证据。本Task无产品/规则改动，既有终态合法，不强造例外或伪称行为修复。

## 16. T7 成员getter最后读者

Task BASE `4c9685cd`。沿原TypeScript program解析property access/object binding的属性声明，生产3读取都指向IProjectMemberStore，分别是M7 mobile-header、view-list-header、inbox filter-selection；测试2读取在project-member.store.test。声明/interface、computed注册不算读者；M4同名局部数组是显式getProjectMemberIds调用，历史utils.tsx两读者已无。未发现无法分类的同名属性。保留仍活getter及消费者；准确路径、两个收集箱使用点与最后删除条件已写[M7回执§7](../../M7-collaboration/handoffs/M3-closeout.md#7-m4p16-成员-getter-最后读者回执2026-11-01)。

M17以同一M4临时文件中的显式方法调用为正例，再替换为store.projectMemberIds。真实checker明确新增一个M4属性读取并exit1；去掉临时文件，原store字节未变，重扫3生产/2测试、0M4/0未分类通过。`t7-member-probes.json`和三个原始receipt保存输入/诊断/恢复；`t7-member-store` 21/21、`t7-web-types` typegen/tsc通过。本Task没有产品改动，成员目录权限与会话责任保持。

## 17. T8 前端说明与子交接

Task BASE `565a2927`。README把P12分支和P13–P15待验收历史字样改为已集成的四布局、富筛选、归档/个人三关系、详情/创建/协作/草稿/命令两宿主能力；P16仍明确为实施分支。前端改动清单保留P12原验收数字并修正当前集成标题，新增P16实证；plane-diff对应泛型、旧dropdown链、动态协议、no-shadow与关键词边界。评论只跟进已载增量窗口、HTTP复制失败、M4图片/附件禁用和M6–M8自身页未接通均保持准确。

M1-P3/M1-closeout/M3-closeout只补当前子回执，历史“当时未合并”和失败记录不覆盖；链接P15实际main313728cb/CI/归档说明现状。八份输入顶部全部open，M4设计§14和总体9.4字节未改，未创建未来M4-closeout。静态逐新链接及章节核验14项通过、diff --check通过，A1–A8勾选、A9–A11未勾；`t8-doc-check.json`保存检查输入hash和逐链接。无产品改动，不重复运行整套产品测试。

## 18. T9 S2 最终入口与请求

Task BASE `e0c75a6b`。S2 REQUESTS新增管理员/成员归档、各自个人三关系及草稿；访客补三关系/自己个人页和草稿。四条身份故事共49次冷加载（13/13/15/8），仍按method+path+query和重复数逐次、结尾再次对账。全文件保持13个故事，含未登录、错误工作区、归档项目、新手引导、个人设置及详情三身份；未复制世界夹具，前置仍全部API。

| 入口 | 身份和真实覆盖 | 请求/边界 |
|---|---|---|
| 项目列表及四布局 | S2四身份，I3/I4/I6/I7、C3补数据/交互 | 成员一次列表GET及既有两次不同宿主偏好GET；非成员只读项目，不读工作项 |
| 归档列表 | S2 admin/member，I15/C9补恢复/删除/Back | APP+WORKSPACE及一次issues?order_by=sort_order&sub_issue=true&limit=100&archived=true；归档不经过普通ProjectLayout，不取项目包装层资源 |
| 个人三关系 | S2 admin/member各自assigned/created/subscribed；I19补跨人/可见项目与续页 | 一次member issues GET，relation准确、order_by=sort_order、sub_issue=false、limit=100；不额外取偏好或标签目录 |
| 访客个人页 | S2别人三关系/自己assigned/两个原入口；I19桌面手机 | 无个人列表GET、Display/Add filter/布局/关系导航；可见无权限说明 |
| 草稿 | S2 admin/member/guest；I14和C9补保存/发布/取消/Back | 成员一次无query的个人drafts GET；访客零草稿请求、创建禁用且解释拒绝 |
| 编号/项目ID详情 | S2(detail)成员200、访客404、非成员403；I1/I2、editor-loading C10两地址 | 按实际回答与DOM/console核身份；不是把动作请求冒充冷加载清单 |
| 顶部/弹窗命令搜索 | I18两宿主、C7原生动作/IME/Back | 由既有真实动作故事验证，S2无伪造搜索请求名单 |

新增断言先证实访客草稿正文空白：组件 `t9-guest-refusal-red` 与实际 `t9-s2-baseline` 均缺无权限说明。WorkspaceDraftIssuesRoot沿既有canRead判断，在已知访客角色时显示现存双语拒绝文字；加载/退役仍不发请求，未改角色/协议。首次S2管理员/成员失败另是测试预期错误：误套普通项目包装层和服务端偏好默认值；沿core.ts归档独立路由、getComputedDisplayFilters及profile/archived store重新核准本地默认值，未把它记成产品缺陷或照抄实际请求白名单。

修复后两组件文件5/5、S2 13/13、I14/I19页面9/9，make lint-web54任务通过；统一产品/测试输入 `6a97e025ffdae1e84a34a034493bdf3517bb8dbe1316ec91b4ec17b4fb2798f3`，收据输入/输出相同。M18在页面注入一次重复归档GET，M19在访客个人页注入实际列表GET，两次原精确数组断言均显示Received多1；这是浏览器网络故障注入，未声称两个生产实现曾有该错误。临时测试字节恢复、受测bin hash未变，S2恢复13/13。原始补丁/红原因/恢复hash在`t9-request-probes.json`，初次报告和各反例trace另存，不覆盖失败证据。三引擎复验归T10。
