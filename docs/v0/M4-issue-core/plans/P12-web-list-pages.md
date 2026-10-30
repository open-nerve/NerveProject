# M4/P12 列表页面实施计划

> 给执行代理：负责人已授权实施，按通过预检的 T1–T16 顺序推进。使用executing-plans；是否委派另遵负责人及适用技能要求。阶段完成做整分支deep-review、修复复审，先不合并。

**目标：** 四布局和原生添加已有选择器在真实页面完成确认修改、分页/排序/拖放、键盘/焦点/宿主结束及I3/I4/I6/I7、C1/C3列表侧。

**结构：** 复用IssueRootStore/canonical/PermissionReads/DetailResource/IssueListBinding及既有写队列；只增加有真实消费者的搜索薄服务和宿主查询状态。使用锁定的React/MobX、Headless UI、react-popper、pragmatic drag、Vitest/Playwright，不新增依赖或后端/API字段。

**权威：** [spec](../specs/P12-web-list-pages.md)、[M4设计](../M4-design.md) §3.16–3.18/3.22/7.4/7.5/9.7/12/P12；[预检记录](../reviews/P12-web-list-pages-review.md)。基线 main `ce3c39abef99a290157b282c23180983f8746bcb`，工作树 `.codex/worktress/codex/m4-p12`，分支 `codex/m4-p12`。C/L/P路径缩写和文件结构见spec§2.1。

## 全局约束

- 主目录保持main；准备不改变产品、phase或依赖。实施开始重核基线差异；T1才把phase从M4/P11改为M4/P12，并核实际到期例外，不把全部M4/closeout改期。
- plane/refer只读，不动已有容器/Docker；无全局安装/设计外依赖，无stash/clean/reset --hard/force，不取钥匙串或浏览器凭据。系统`/usr/bin/git`负责push；详细中文提交保留`feat/fix/test/docs(M4/P12)`前缀。
- 每个Task末类型/受影响回归可通过；迁接口/props同步所有生产调用和既有测试。新文件约400行；超过时按职责拆，测试表格确需例外记真实原因。不用any/双重断言/宽Partial假实体迁就旧协议。
- 当前错误按已有Problem码及文案表显示，新增文案同时en/zh-CN与keys同步；SessionChanged/旧宿主错误无旧提示。修改拒绝不改canonical，修改成功后补读拒绝不回滚、不重复写。
- 用现有FakeNerve、fake-issues/issueStoreTab、fake-projects、heldChange/fake-time；浏览器用e2e/fixtures/{test,api,issues,browser}及assert/issue。新helper须有两处真实复用，不复制整套假客户端/服务器；不使用sleep碰时序。
- 共享DateDropdown/菜单等局部修复在已批准范围内；若发现要改公共协议/权限归属或架构回归，先提交已完成工作，再上报。P11列宽A方案及P10b权限A方案不重开。
- 只读准备代替原型/逐Task重放，按02§3记录方法调整；本计划不伪造未来代码、生成hash、红绿或运行数。实施按真实字节/argv/SHA补证据。

## 验证协议

**G（逐Task）：** 写必要行为断言并确认红→最小实现→绿→对生产性质作有效变异红→精确恢复字节/hash→绿→受影响回归/类型/diff-check→要求与质量审查、修复复审→精确路径提交。缺文件/编译红单记，不算行为红；同性质换输入不冒算不同变异。台账保留cwd、argv、源码SHA/dirty文件、raw、exit、执行数/skip和恢复hash。普通低风险机械文档变更只静态核，不写镜像测试。

**聚焦：** 从worktree根运行`pnpm --filter web exec vitest run core/...test.tsx`，目标见各Task；类型用`pnpm --filter web check:types`。共享包有真实改动时先`pnpm exec turbo run build --filter=web^... --force`。无匹配/全skip不是绿。页面聚焦先用当前源码`make build`，再进入e2e运行`pnpm exec playwright test stories/issue/<文件>.spec.ts --project=chromium --grep '(page)'`；以实际参数和非零故事数为准。

**F（本地七项，严格串行）：**

```sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
```

每个产品检查点push前跑F并记录准确源码，建议检查点T5/T10/T15；Task16若有产品修复再次按差异完成必要验证，最终交付F须覆盖全部最终产品输入。连续纯文档提交可核input hash复用已通过F，不能冒称重跑。服务端/端到端数据库测试不并跑，不使用旧bin。默认E2E只有Chromium全套和Firefox A4；C1/C3另见T15。准确分支CI独立于本地F，main CI只在未来授权集成后核。

## Task 1：原生搜索、选择器和三个真实调用方（约1100–1500行）

**Files：** 新C/services/issue/issue-search.service.ts/.test.ts、C/hooks/use-issue-search.ts/.test.tsx；C/store/issue/root.store.ts；C/components/core/modals/existing-issues-list-modal.tsx/.test.tsx；L/calendar/quick-add-issue-actions.tsx/.test.tsx；C/components/issues/issue-detail-widgets/issue-detail-widget-modals.tsx及其实际测试；tools/keywords.json。
**Interfaces：** service签名及三种query见spec§2.2；hook接受显式scope/查询/打开状态，公开当前ID集合、loading/error/retry及读取证明；DetailResource负责值/证明，canonical负责Issue，选择ID/权限证明只归选择器一处。`handleOnSubmit(ids): Promise<SelectionSubmitResult>`签名按spec§2.2，三个调用方一起迁，不能继续ISearchIssueResponse或Promise<void>导致部分成功无法表达。

- [x] 重核基线、phase及exceptions；按真正现存调用建立表，三个入口与旧测试一起迁。生成query日期必须数组，for/issue_id成对；service只unwrap原生data，没有cursor/旧形状转换。
- [x] service真实请求测试`usesNativeSearchQueriesForAllThreeConsumers`核空词、none、for/source、关联workspace省略project；403/422/SessionChanged原错交回。hook用root/readtoken，目录当前证明后acceptListIssues，resource仅存ID，不新建权限观察者。
- [x] 测试`invalidatesCandidatesBeforeDebounceFires`、`doesNotPublishAcrossScopeOrPermissionABA`、`doesNotAcceptForeignSearchRows`：q刚变化旧答即使先回来也不可显示/选择；项目A→B/工作区开关/source变/关闭重开/退役/同账号权限降再恢复均隔离；冷项目/状态目录拒绝可观察，错误重试只读。
- [x] 选择器开时空词搜索；同scope跨词保留已选ID与有效证明，换业务/权限scope清空，旧候选立即不可点；迁真实选择、禁用pending、重复点击、失败保留、当前scope确认关窗；保持原有拒绝/旧项目晚答/切target清选择测试性质，去除测试专用旧callback注入。用ModalTitle命名，取消正tabIndex，候选只含可选选项，不保留hover嵌套链接。
- [x] 日历callback真正await全部写并返回确认/失败ID；部分成功移除已成功选择，只保留失败项，全成功才关。T1测试双选一成功一失败后重试只发失败ID；T7扩真实日历日期/组合边界，当前Task不得留下fire-and-forget或重复成功写。子项/关联query同Task改，不影响P14完整页面验收；删除唯一对应模块实例，不删其它旧搜索消费者。
- [x] 聚焦新service/hook及既有两modal/calendar测试、detail-host/子项关联受影响测试、types；M01旧搜索URL、M02延迟到debounce才失效、M03跳过返回scope证明各行为红；G。提交`feat(M4/P12): 接通列表添加已有工作项的原生搜索`。

## Task 2：原22路径与显式路由scope（约650–1100行）

**Files：** spec§2.9表中14现存文件及utils调用方、tools/keywords.json；新L/route-scope.test.tsx，迁group-consumers.test.tsx省略scope的fixture。

- [ ] 从准备inventory核22项当前存在/删除；不恢复8已删文件。已正确按props/route读取的只复核，归档header的local currentProjectDetails变量改名避免误命中，不能豁免规则。
- [ ] 目录helper明确workspace/project参数，project/state/label/created_by/assignee均不回退router store；profile/归档/子项真实调用同步。rootStore的同步读取不发HTTP；workspace级调用显式标范围，draft删除保留P11的workspace权限。
- [ ] `rendersTheNewRouteBeforeRouterStoreEffect`使route=B/store仍A，核地址、目录、权限、候选与组列全B；覆盖项目成员/访客、project_id缺失不能回落A。当前读错误仍可观察，不自行批量预取目录。
- [ ] project-address-route保留M3已守卫路径并扩至原22项，已删项留files样本阻回归；命中/不命中样本包含多行allowPermissions、显式slug/project、合法workspace删除。故意复原隐式权限使守卫红；核M7 getter读者不提前删除。
- [ ] 跑route-scope/group-consumers/list-data/detail-entry及相关权限回归、关键词样本/types；M04错路由、M05同步helper发请求，G。提交`fix(M4/P12): 让列表地址和目录选择使用显式路由范围`。

## Task 3：列表分组和续页消费者（约450–850行）

**Files：** L/list、group-consumers.test.tsx、list-data.test.tsx；新L/list/view.test.tsx；仅确有缺口时改现有binding职责内实现。

- [ ] 真实ListGroup/加载控件接同组cursor，保留原筛选；加载中去重、失败可重试、不重复/漏行、末页停止、空组显隐均测试。错误属于当前scope，不清其它组。
- [ ] 多标签/负责人组保留完整数组；UI总数读权威值，不能按已加载长度推total。A筛选/C组第2页保持A∩C；已有子项/编号宽度测试同时回归。
- [ ] 可见区域外已加载行、展开子项和分页后更长编号保持P11同一宽度与重置合同；不新增宽度取项目请求。加载更多按钮有可访问名称，键盘可达。
- [ ] 聚焦group-consumers/list-data/list/view/list/identifier-width，M06续页丢filter、M07总数改已加载长度各红；G。提交`fix(M4/P12): 闭合列表分组续页和失败重试`。

## Task 4：快速添加字段错误与当前宿主（约450–850行）

**Files：** L/quick-add/root与form/button；扩root.pending，新root.errors.test.tsx；错误文案/en/zh-CN仅有新码/文案时改。

- [ ] 名称Field/FieldError、aria-invalid/describedby、useRefusedFocus；关闭RHF默认抢焦，空白与服务端422字段拒绝都先显示错误再焦点。有效修改/成功清错，403不冒充name校验错误；IME组字不误提交。
- [ ] 确认前输入/pending保留、防双提交、拒绝可重试；成功后只在当前scope重置/聚焦。原root.pending的A成功后允许旧toast断言按新宿主要求迁成无旧toast，保留“B pending不会被A释放”的原性质。
- [ ] state/priority/labels/assignees/none的既有原生预填保持，状态列末尾位置不在UI硬加；switch group/route/关闭/权限ABA后不执行旧回调或创建第二条。
- [ ] 聚焦两root测试/group-consumers和真实Field/refused-focus回归；M08先focus后标invalid、M09丢宿主检查、M10移除pending锁分别红；G。提交`fix(M4/P12): 完成快速添加的字段反馈与确认焦点`。

## Task 5：看板和泳道拖放（约850–1400行）

**Files：** L/kanban、L/utils.tsx、C/hooks/use-group-dragndrop.ts/.test.tsx，新L/kanban/drop.test.tsx；现有列表drop/confirm回归。

- [ ] 真实源/目标构造覆盖同组前/后、跨组两邻居间、列尾和空列；移除源后求邻居，中间below也前进一位。目标失效/不存在给当前错误或安全不发，不能猜任意尾部掩盖事件缺口。
- [ ] 多值group/subgroup用既有makeBody/root.dropIssue，不覆写其它标签/负责人；none清空及优先级none按原生合同。非sort_order布局不可擅自做手工排序。
- [ ] deferred PATCH确认前原列表/计数不变；成功一次state+sort_order，卡片留目标；403/422原位、补读失败保留成功并只读重试。实际hook的新toast按码且同时核session/scope/permission；旧timer/highlight/scroll卸载后取消。
- [ ] 泳道总数只用subgroup_totals，组/格各自cursor，加载控件键盘可达；一个多标签卡片多处与唯一canonical一致。
- [ ] 聚焦drop/hook/group-consumers及root排序/确认测试；M11中间below变above、M12泳道格total相加、M13先乐观改序、M14尾部被旧读复位各行为红；G与检查点F。提交`fix(M4/P12): 按确认结果完成看板和泳道拖放`。

## Task 6：表格排序、滚动和编号链接（约600–1000行）

**Files：** L/spreadsheet全调用链、新view.test.tsx/columns/header-column.test.tsx，spec指出的两独占localStorage键。

- [ ] 映射现有可排序列到原生order_by，仅使用3.16支持值；箭头、aria-sort来自确认displayFilters。移除第二份全局排序真值，失败保留旧偏好/箭头/数据；刷新与同账号换项目不串。
- [ ] 单元格可编辑值仍用现有确认入口；无限滚动保留order/cursor、去重加载和最后一页结束，空/失败/重试不是假末页。四种指定排序正反方向与真实第二页留T14页面验证，本Task真实生成客户端/组件断言。
- [ ] 子项列用project catalog中的真实identifier和sequence_id构建browse链接，无#sub-issues；目录未就绪不能造错误编号/旧ID链接。
- [ ] 聚焦view/header-column/list-data及相关filters偏好回归；M15排序码旧字段、M16失败仍改箭头或跨项目localStorage污染、M17链接旧片段各红；G。提交`fix(M4/P12): 使用原生排序和确认偏好驱动表格`。

## Task 7：日历月周、拖动与添加已有（约550–1000行）

**Files：** L/calendar、quick-add/form/calendar；新view.test.tsx，扩quick-add-issue-actions.test.tsx。

- [ ] 日历月/周范围、当地日YYYY-MM-DD、跨月/年/时区边界，current scope旧查询/旧错误不覆盖新日历；日期范围与基础筛选AND，不丢none的独立搜索合同。
- [ ] 拖动只有target_date PATCH；确认后日期/计数/位置更新，拒绝原日保留；同日不重复写。状态/标签/负责人字段不随拖动误发。
- [ ] 添加已有空词/输入搜索都带project_id与target_date:[none]；开始日期晚于目标日禁选，已含target_date项不进候选。多选去重，await allSettled等价完整结果；保存已成功ID、只保留失败ID待重试，错误按码显示；确认后的读失败不再次PATCH已成功项。
- [ ] host关闭/换日/换项目/退出后不关闭新selector或toast，失败保留当前选择。真实modal+calendar组合测试，不能只spy update被调用就称完成。
- [ ] 聚焦view/quick-add/selector/root日期确认测试；M18UTC错一天、M19target_date条件丢失、M20部分成功重试再次写成功项各红；G。提交`fix(M4/P12): 完成日历日期与添加已有的确认流程`。

## Task 8：DateDropdown一份状态与懒渲染（约500–900行）

**Files：** C/components/dropdowns/date.tsx、新date.test.tsx；L日期属性/表格单元格实际懒渲染调用，api-tokens自定义有效期消费者测试。

- [ ] 用Popover提供唯一打开状态，去掉useDropdown与ComboDropDown互相同步；选中、Escape、Tab、外部点击只结束同一state。保留实际public props与禁用/范围/defaultOpen/closeOnSelect合同。
- [ ] 键盘focus/Enter/Space触发能装载懒内容；先开日期再开旁边属性不会残留两个面板。无嵌套button，portal/ModalCore内可点可Tab，背景仍受外层modal隔离。
- [ ] DOM测试对应上述事件；实际浏览器由T15核定位和focus。回归api-tokens自定义有效期/minmax，M6/M7调用编译契约不变，不运行未接通领域接口。
- [ ] M21保留双state、M22只pointer才能渲染各红；G。记录P13 R36共享根因在本Task覆盖，筛选页面验收仍P13；提交`fix(M4/P12): 让日期下拉由单一打开状态控制`。

## Task 9：四个基础属性下拉（约650–1100行）

**Files：** C/components/dropdowns/date-range.tsx、priority.tsx、project/base.tsx、state/base.tsx及对应测试/直接消费者。

- [ ] priority/project/state按有无搜索选择Combobox/Listbox，popper ref/styles/attributes落Options列表元素；date-range的Calendar用单Popover/Panel，ref落Panel，不能将网格按钮伪装成Listbox选项。只由Headless状态决定open，不手动第二份state/点击模拟。打开搜索框获焦，Escape还给触发器。
- [ ] useInsideModal→modal=false只适用内层列表，避免它解除/覆盖外层modal对背景的隔离；下拉触发label/disabled保持。DateRange沿P11一次PATCH两个字段的合同，拒绝不清旧值。
- [ ] 目录当前拒绝可观察/可重试，旧scope不跟进；成员/访客按现有role精确禁用，不通过控件重写权限逻辑。M6/M7共享project/date-range消费者只兼容props。
- [ ] 四DOM测试+既有project/base及日期原子修改回归；M23ref放子项、M24内层modal破坏外层各红；G。提交`fix(M4/P12): 修正基础属性下拉的定位和键盘状态`。

## Task 10：三个标签与通用选择控件（约600–1100行）

**Files：** C/components/issues/issue-detail/label/select/label-select.tsx、L/properties/label-dropdown.tsx、C/components/issues/select/base.tsx及测试；标签创建hook调用方仅确需兼容处。

- [ ] 同T9的单state、Options ref、modal内行为；select根不再无名停留点，触发器以当前工作项命名。P11只读分支disabled和集合pending锁保留。
- [ ] 新建标签仍经useLabelChanges().create，重复名field拒绝/失败留窗/旧scope无跟进；搜索和新建触发的职责不混淆。P14哨兵选项及详情表单正tabIndex审计如未实际覆盖，准确移交，不据本Task宣称全关。
- [ ] 聚焦三控件DOM、P11标签创建/拒绝/只读回归和types；M25从只读分支漏disabled、M26绕统一create、M27根无名tab点各红；G与检查点F。提交`fix(M4/P12): 完成标签选择器的单状态和模态兼容`。

## Task 11：菜单键盘、右键和层叠（约650–1100行）

**Files：** L/quick-action-dropdowns/*、P/ui/src/dropdowns/custom-menu.tsx及context-menu/root.tsx；L/roots/project-layout-root、calendar Popover、quick-add/form/calendar、calendar/issue-block、list/block、kanban/block/delete-drop-zone；新L/menus.test.tsx与受影响共享测试。

- [ ] 逐个仍存在的SubMenu写键盘事件断言：打开、箭头进入/返回、Escape、焦点归还；看板/日历卡片键盘激活底色与pointer一致。已删cycle/module菜单只登记删除，不恢复五个历史文件凑数。
- [ ] issue-detail菜单去IconButton嵌套，使用既有customButtonClassName；ContextMenu在基线已实现inert，核真实列表关闭时不参与Tab、打开时可达；无缺口不重写已有行为。新处理经isComposing，不用keyCode229。
- [ ] 点名pending/日历/拖动层在z-30下或有实际模态遮挡证明；Button/IconButton/CustomMenu焦点轮廓不被overflow裁。共享组件改动必须保留其它实际调用回归，不用禁动画规避生命周期问题。
- [ ] DOM+共享菜单回归，浏览器T15核C3；M28子菜单仅pointer、M29关闭菜单可tab、M30嵌套button各红；G。提交`fix(M4/P12): 完成列表快捷菜单和模态层叠的键盘行为`。

## Task 12：peek宿主结束与焦点归属（约400–750行）

**Files：** C/components/issues/peek-overview/root/header/view/issue-detail、实际打开方及L/roots/project-layout-root；host-actions.test.tsx/detail-host.test.tsx、新peek生命周期测试。

- [ ] 对照P11owner机制清点每个open/close/switch/back/unmount；保留WorkItemLevelModals所在provider，不能全局清新owner。切scope同步隐藏旧项，旧HTTP无UI跟进。
- [ ] 目录/工作项拒绝、权限降级/ABA、退役结束当前加载/内容；错误可观察。切A→B时A成功/失败/cleanup均不关B，不执行旧scroll/focus；只给仍存在的本宿主触发器归还焦点。
- [ ] 两并存owner、StrictMode重复cleanup、同账号route变更、删除当前项/后退；props输入来自实际route不从router effect。CustomSelect页头键盘功能在T15实际核。
- [ ] 聚焦host-actions/detail-host/新测试，M31旧owner关新peek、M32卸载仍执行focus各红；G。提交`fix(M4/P12): 让工作项浮层随实际宿主结束`。

## Task 13：I3与I7真实页面故事（约650–1100行）

**Files：** e2e/stories/issue/i3-list-groups.spec.ts、i7-quick-add.spec.ts；必要共享fixture（新issue-list-page.ts，实际两故事复用）。

- [ ] 保留API故事，加(page)故事：API准备独立账户/项目/状态标签/足够跨页数据，浏览器登录后真实UI操作。让组有超过真实页面limit的数据；不得只拦截造假分页响应代替端点。
- [ ] I3状态组加载更多/泳道label×priority的唯一计数、完整数组、A∩C续页、切布局刷新；请求query与DOM顺序同时断言，不在断言前排序服务端答案。
- [ ] I7选组快速添加，DB真实state/默认尾部sort和活动；空白field可访问描述、recordFocuses取焦时invalid；pending双提交/拒绝留输入。少量错误拦截精确期待watchPage失败，其它均零。
- [ ] watchPage在导航前接，全部页面拒绝/console/CSP/旧URL/外域结果独立断言；沿expectQuietConsole按每次真正加载编辑器精确期待已知EMOJI_CHECK_WARNING，保留探针和次数断言，除此以外零警告；删除提示/编辑器分割归P14。新fixture不放宽全局watch。当前build跑两(page)故事Chromium，记录非零数；变异M06/M08至少各有页面层捕获证据。G，提交`test(M4/P12): 用真实页面验证分组续页和快速添加`。

## Task 14：I4、I6页面与S2（约800–1300行）

**Files：** e2e/stories/issue/i4-list-order-calendar.spec.ts、i6-board-order.spec.ts；既有smoke/s2-web-app.spec.ts，T13共享fixture。

- [ ] I4四种排序双向真实二页、月/周/日范围、拖到另一天且DB一条target_date动态；添加已有开窗空词、只无日期候选、选中确认后出现，错日期拒绝不关。时区用browser context明确设置，并核目标日期不漂移。
- [ ] I6成员从Todo放Done两卡之间，UI计数/位置、准确一次PATCH与DB completed_at/updated_by/state动态；同列below中项/尾部、拒绝回位、确认后刷新稳定。pointer真实拖动，不直接调用store假扮页面拖放。
- [ ] S2实际打开项目工作项列表/切必要布局、继承已有身份和REQUESTS白名单；新精确请求按真实合同添加。M6/M7未实现领域不能通过mock空成功混入本段范围。
- [ ] 当前build下I4/I6页面+S2；保留原API全部断言。M11/M18/M19至少页面捕获；G，提交`test(M4/P12): 验证表格日历拖动及页面请求边界`。

## Task 15：C1/C3三浏览器、dead与完整门禁（约600–1000行）

**Files：** 新e2e/playwright.m4-list.config.ts、新e2e/stories/issue/list-interactions.spec.ts；改到的barrel/成员/keywords/oxlint cap；测试/浏览器逐项台账。

- [ ] 专用config复用default globalSetup及fixtures设置，明确projects为chromium/firefox/webkit并覆盖默认project列表；testMatch仅I3/I4/I6/I7文件与list-interactions文件，标题grep为(page)或C1/C3交互名（testMatch只能筛文件），不继承Firefox A4 grep。未接入全局CI之前不扩大默认全套浏览器矩阵；没有专用配置消费者则不留无效文件。
- [ ] 从e2e执行`pnpm exec playwright test --config=playwright.m4-list.config.ts --list`，核三个projects、I3/I4/I6/I7/C3每类实际非零；再分别`--project=chromium`、`--project=firefox`、`--project=webkit`串行实跑，准备当前make build。缺浏览器只通过锁定本地Playwright支持的安装方式准备，不全局装，不把缺依赖写成验收通过。
- [ ] C1逐行为/浏览器记录，C3逐点名控件记录：键盘开DateDropdown、单状态/相邻第二控件、七个Options锚点、外层modal背景inert/隐藏、CustomSelect peek/自动归档/日期筛选与成员控件实际入口；某入口属于后续完整页面验收时精确写覆盖的共享行为与剩余，不能整项假关。各浏览器的EMOJI_CHECK_WARNING/人为拒绝console预期按实际观察固定文本与次数，没有则期待空，不使用Chromium宽泛忽略或把watch结果当预期；P14仍负责移除该提示。截图辅助，焦点与选择用可观察断言。
- [ ] 按实际修改路径跑oxlint零警告、对应cap同步降低，不提高上限；dead按真实调用链删除或逐项判定，knip为零；守卫没有新增宽泛例外，阶段内旧模块实例/旧查询键的目标路径零命中。
- [ ] 独立变异至少15性质，含协议/隔离/权限/分页/排序/焦点/确认/生命周期，逐条保留生产diff、首失败、恢复hash与绿。F七项串行、无缓存web实际测试数、E2E默认非零；push检查点核准确CI。G，提交`test(M4/P12): 核验三浏览器列表交互并完成阶段门禁`。

## Task 16：整分支deep-review、修复与交付（约250–500行文档，修复另计）

**Files：** 本spec/plan/review、新docs/reviews/re-NN-M4P12整体审核.md、M4-design§3.23/15、README前端、docs/v0/frontend-changes.md、M2/M3输入handoff局部结果；真实修复按发现归属。

- [ ] 从准确base到交付HEAD审整分支，顺序逻辑正确性/集成安全/测试覆盖；全文清单、相关依赖、未覆盖如实记录。每发现写根因、影响范围、验证，局部修复后复审；架构问题先提交再上报。
- [ ] 核A1–A12实证及所有写的确认/拒绝/旧scope/成功补读失败四列。旧证据仅在源码输入hash未变时复用；后续修复不能冒用旧审批。候选M01–M32不等于已运行32项，最终至少15不同性质实际有效。
- [ ] README前端/设计3.23/P12行及frontend-changes按实际完成更新，输入handoff只标对应局部处理；原22清单、M2列表下拉/搜索实例、M3 watchPage/菜单/路由逐项证据。顶层open、M4§14/总体§9.4和最终M5–M8 closeout保留。
- [ ] 最终产品变化按差异完成必要重验，F覆盖全部最终输入；准确文稿HEAD分支CI后置核server/web/e2e，写忽略目录final-delivery.json防自引用循环。只有这些通过才能报告阶段完成。
- [ ] 提交`docs(M4/P12): 汇总列表页面验证与整分支审核`；报告分支/base/spec/plan/review/CI，保留worktree，不合并、不开始P13，等待负责人安排。

## 规模、预检与证据限制

16Task，预计变更约一万至一万七千行，含测试/删除/文档；数字是准备估算，不是实现实测。T1原子迁移、T5真实拖放与T9/T10共有组件是重点规模风险；实施开Task前据消费者重新量，必要时在16内重排，若仍不能承载则先提交上报阶段拆分，不能无声超范围。

预检以合同/文件/调用链/既有测试静态推导与一次独立builder-check、主控三视角deep-review完成；未跑产品、未生成测试字节/hash、未做原型或逐Task回放。所有实施复选框保持未勾；准备完成状态在review登记。
