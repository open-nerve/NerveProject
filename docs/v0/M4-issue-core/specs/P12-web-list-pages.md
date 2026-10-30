# M4/P12 列表页面规格

> 状态：预检通过后已获负责人授权实施，当前从 T1 顺序推进。分支 `codex/m4-p12`，基线 main `ce3c39abef99a290157b282c23180983f8746bcb`。T1 将关键词 phase 切到 `M4/P12`；阶段完成先做整分支 deep-review，不合并。附录 A 保留实施前的准备快照。

权威输入：[M4 设计](../M4-design.md) §3.16–3.18、3.22、7.1、7.4、7.5、9.7、12/P12、13.1；[推进方法](../../claude_to_gpt/02-method.md)、[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md)；[P10b spec](P10b-web-list-data.md)、[P11 spec](P11-web-detail-data.md)及两段review的后续责任；M1-P4、M2-closeout §10/12、M3-closeout §6–8输入交接。配套[plan](../plans/P12-web-list-pages.md)、[review](../reviews/P12-web-list-pages-review.md)。

## 1. 目标

项目工作项的列表、看板及泳道、表格、日历在真实页面上完成分组分页、排序、拖放、快速添加、属性编辑、快捷操作和浮层交互。添加已有工作项改用原生 issue-search；键盘、焦点、拒绝和宿主结束行为可观察。完成 I3/I4/I6/I7 页面版、C1 三浏览器及 C3 列表一侧，保留既有 API 故事。

沿用 P10b/P11 的唯一 canonical、权限范围读取失效、确认写入和列表协调，不重新实现数据层。P13 的完整筛选/偏好/归档/个人主页，P14 的完整详情/创建与父项搜索，P15 的时间线/草稿/命令搜索和 P16 的全局清理仍由原阶段负责。

## 2. 交付物

### 2.1 文件与任务

`C/` 为 `web/apps/web/core/`，`L/` 为 `C/components/issues/issue-layouts/`，`P/` 为 `web/packages/`。标“新”的是实施目标。原设计15个粗项细化为16个可独立验证的Task；七个属性下拉按基础/标签选择拆成T9/T10，范围不变。

| Task | 文件/职责 | 验证落点 |
|---|---|---|
| T1 | 新 C/services/issue/issue-search.service.ts、C/hooks/use-issue-search.ts；C/store/issue/root.store.ts；C/components/core/modals/existing-issues-list-modal.tsx；日历quick-add-issue-actions、issue-detail-widget-modals三个消费者原子迁移 | 新service/hook测试；既有选择器、日历测试；详情添加子项/关联请求测试 |
| T2 | §2.9原22路径、L/utils.tsx及它的实际调用方；tools/keywords.json | 新 L/route-scope.test.tsx、规则样本；既有group-consumers/list-data/详情入口测试 |
| T3 | L/list/{base-list-root,default,list-group,blocks-list,block-root,block}.tsx及headers | 既有group-consumers、identifier-width；新list页面组件测试 |
| T4 | L/quick-add/{root,form/*,button/*}.tsx | 扩root.pending；新root.errors.test.tsx，真实Field/refused focus |
| T5 | L/kanban/*、L/utils.tsx、C/hooks/use-group-dragndrop.ts | 新kanban/drop.test.tsx；扩use-group-dragndrop与已有列表拖放测试 |
| T6 | L/spreadsheet/*，列头/子项链接/滚动 | 新spreadsheet/view.test.tsx、columns/header-column.test.tsx |
| T7 | L/calendar/*、quick-add/form/calendar.tsx | 新calendar/view.test.tsx；扩quick-add-issue-actions.test.tsx |
| T8 | C/components/dropdowns/date.tsx、L中的日期懒渲染调用方 | 新date.test.tsx，api-tokens自定义期限消费者兼容回归 |
| T9 | C/components/dropdowns/{date-range,priority,project/base,state/base}.tsx | 新/扩对应DOM测试，真实portal/ref/单状态 |
| T10 | C/components/issues/{issue-detail/label/select/label-select,issue-layouts/properties/label-dropdown,select/base}.tsx | 新标签/select组件测试；既有创建标签和拒绝回归 |
| T11 | L/quick-action-dropdowns/*、列表右键菜单实际共享实现、L/root及遮罩/拖动层叠调用点 | 新menus.test.tsx，既有CustomMenu回归；浏览器键盘/层叠 |
| T12 | C/components/issues/peek-overview/*、L/roots/project-layout-root.tsx及实际打开方 | 扩host-actions/detail-host；新peek生命周期DOM测试 |
| T13 | e2e/stories/issue/i3-list-groups.spec.ts、i7-quick-add.spec.ts；必要的共享fixtures | I3/I7页面，真实API/DB断言及watchPage |
| T14 | e2e/stories/issue/i4-list-order-calendar.spec.ts、i6-board-order.spec.ts；smoke/s2-web-app.spec.ts | I4/I6页面、精确请求/S2边界 |
| T15 | 新e2e/playwright.m4-list.config.ts、新stories/issue/list-interactions.spec.ts；本段改动的dead/oxlint/守卫收口 | C1/C3三浏览器、F七门禁、候选准确CI |
| T16 | spec/plan/review、整分支deep-review报告、设计§3.23/15、README、frontend-changes、输入handoff局部结果 | 完成线逐项实证、修复复审、最终准确分支CI；先不合并 |

### 2.2 原生搜索与唯一工作项

`IssueSearchService` 接本代 `ApiClient`，由 `IssueRootStore` 持有；`search(slug, query): Promise<IssueCollection>` 只调用 `GET /api/v0/workspaces/{slug}/issue-search` 并 unwrap。query 提取 `NonNullable<operations["searchIssues"]["parameters"]["query"]>`，不复制服务端DTO。应答为 `data: Issue[]`，没有分页cursor；默认50、limit最高100。空词打开即请求最近更新的候选，不先要求用户输入，不造无限滚动。

| 入口 | 查询合同 | 修改入口 |
|---|---|---|
| 日历添加已有 | `project_id`、`target_date: ["none"]`、可选q；无for/issue_id | 每个所选ID经现有root更新，只写目标target_date |
| 添加已有子项 | `project_id`、`for: "sub_issue"`、`issue_id: 父项ID` | 现有批量addSubIssues |
| 关联 | `for: "relation"`、`issue_id: 当前工作项ID`；项目范围带project_id，工作区开关打开时省略 | 现有createRelations与relation_type |

排除自己/祖先/后代/已关联由服务端完成。三处同Task迁移，取消旧 `sub_issue`、`issue_relation`、`workspace_search` 和模块级ProjectService实例。删除该组件的旧回调注入参数、正tabIndex及不再使用的旧类型import；P14/P15/M7仍有生产消费者的ProjectService/搜索DTO暂保留，逐路径登记，不把全仓旧搜索消失写成本段门禁。

搜索hook保存候选ID、查询状态及其读取证明；选择器自身保存选择ID和选择时的权限证明，展示始终从canonical派生，不在hook和组件各存一套选择。工作项实体只经root的既有读取接收进canonical。复用 `DetailResource<readonly string[]>` 的scope、`captureRead/isReadCurrent`与 `PermissionReads`，不用第二个角色观察者或实体cache。project查询scope包含workspace/project；工作区查询从空projectIDs的工作区权限水位捕获，发布前核所有返回项目及当前workspace。每条结果的workspace/project必须匹配请求允许范围，权限目录读取成功且当前可见后才能接收；无权/不匹配结果不入canonical。状态/项目显示来自已有目录，不构造`project__identifier`、`state__color`等旧字段；目录拒绝可见并可只重试读取。

打开、输入词、范围开关、source issue、项目/工作区、会话/权限代际变化都立即使旧候选不可用；防抖只延迟新HTTP，不延迟旧候选失效。同业务scope内换搜索词保留仍有权限的已选ID，以已选区域展示，允许跨词多选；范围/source/宿主/权限代际改变才清选择，不借保留已选使旧结果行可点击。关闭/卸载dispose，旧响应及旧错误无跟进；权限ABA不能复活候选。相同scope的读取失败结束loading并显示错误；首次失败不伪装无结果。提交前核每个已选ID的原scope证明、root当前可见性与当前业务scope，重复点击只提交一次；等待真正确认才关窗，拒绝保留可重试输入和选择。窗口有可访问名称，Tab/Escape/返回触发器走现有ModalCore与Headless UI机制。

提交回调明确为 `handleOnSubmit(ids: readonly string[]): Promise<SelectionSubmitResult>`；局部UI结果 `SelectionSubmitResult` 含 `confirmedIds: readonly string[]`、`failures: readonly { id: string; error: unknown }[]`。这只是本次选择命令的结果，不是HTTP DTO或第二份Issue。子项/关联的原子批量成功将本次全部ID标确认，整体拒绝保留全部；日历按每项真实结果汇总。选择器校验结果ID属于本次快照，移除已确认选择，仅保留失败项；全部确认才关窗。意外reject作为本次未确认失败处理；SessionChanged/旧scope无UI跟进。成功后的补读错误走既有资源的只读重试，不列为再次PATCH的失败项。T1即交付这份结果合同及部分成功语义，T7扩真实日历组合/页面边界，不延后修提前关闭。

### 2.3 四布局共用约束

从路由/显式props取得workspaceSlug/projectId，权限调用显式传两者；不能借effect稍后写入的router store决定当前渲染的项目。`useIssueListMount`、IssueListBinding和既有确认协调继续拥有查询/分页/版本；UI不自造游标、排序副本或手加减总数。每组续页保留原查询，加真实group_key/sub_group_key/cursor；A筛选与C组的交集不能退化成只筛C。`subgroup_totals`为泳道总数，不能把各格相加。加载失败有当前scope的错误/重试，成功清错误，不吞成空列表。

所有写等待服务端确认；拖动中的视觉预览可以跟随指针，但实体、列表次序和计数确认前不改。拒绝结束pending并保留原位置/输入；写成功后补读失败保留确认结果，提供只读重试，不重发写入。UI异步跟进同时受会话、权限水位和宿主生命周期约束；同账号切项目后也不能出现旧toast、焦点、scroll或关闭新弹窗。

P11编号列宽A方案原样保留：按当前布局已加载、可见的真实编号统一计算，新更长编号可增宽；scope/权限变更重算，无专用项目序号请求。多标签显示多处共享同一canonical；删除/权限失效及时遮住所有副本。

### 2.4 列表、快速添加和看板

列表的每组计数与续页可达，已加载行和展开子项编号一致，隐藏空组不丢续页。快速添加以选中组的原生字段创建；状态组末尾位置由服务端默认规则产生，不在UI加10000。空白标题有本地/服务端字段错误，FieldError、aria-invalid和aria-describedby先呈现，再由useRefusedFocus取得焦点；有效重试后清错。pending禁止重复提交，拒绝保留内容，group/route变更结束旧表单跟进；新键盘处理用isComposing。

看板支持同组、跨组、两张之间和列尾；多值标签/负责人移动按现有dropIssue build body语义保留其它成员，none显式清空，不带旧字段。目的地below不是只在末项才生效：移除源ID后按实际目标和上/下方向确定邻居，交给既有排序协调计算位置。Todo→Done只发一次含state_id/sort_order的PATCH，服务端写completed_at/updated_by_id和唯一state动态。确认之后卡片留在目标位置，后续读取不得把它重新按旧位置插回。排序不是sort_order时遵循现有可拖规则，不创建与当前排序矛盾的本地重排。

### 2.5 表格与日历

表格列头使用§3.16原生order_by映射，选中状态来自已确认displayFilters，不使用全局localStorage的`spreadsheetViewSorting`/`spreadsheetViewActiveSortingProperty`第二份真值。必要时删除仅本组件使用的旧键，不全局清空用户storage。正反排序、刷新、第二页、失败保留与切项目隔离有断言；label_name/assignee_name/sub_issues_count/target_date双向至少各一条真实续页。子项列直接链接本workspace的`/browse/PROJ-N`，不用旧ID地址和无消费者的#sub-issues。

日历月/周边界用当地日历日的YYYY-MM-DD，不经UTC转换造成前后一天；日期范围与普通筛选按AND进入请求，换月/周使旧页不可覆盖新页。拖动只改target_date，拒绝原日保留，成功只有一条对应动态。添加已有仅搜索没有截止日期的候选，目标日早于其start_date时不可选择；这个显示限制不代替服务端校验。多个选择按ID去重后更新，等待全部结果；部分成功保留已确认结果并从待重试选择中移除，只让失败项重试，不回滚或重复成功PATCH。目录/写/补读错误分别可见；用户关闭或换scope后无旧跟进。

### 2.6 下拉框、键盘、焦点与层叠

DateDropdown改成单一Popover打开状态，选中调同一close，保留现有日期/禁用/min/max/defaultOpen/closeOnSelect等实际调用合同。列表与表格懒渲染从键盘聚焦也能准备内容；api-tokens自定义有效期是必要的共享兼容回归。七个点名控件的popper ref挂实际浮层本身，不挂唯一子元素；离散无搜索选择用Listbox、有搜索用Combobox，一份打开状态。DateRangeDropdown的内容是Calendar网格及多个按钮，并非离散选项，因此与DateDropdown一样用Popover/Panel，不伪装成Listbox或Combobox.Options；保留原范围回调与P11一次PATCH合同。弹窗内的Listbox/Combobox经useInsideModal使列表`modal={false}`；Popover不引入另一份模态隔离，保持外层模态对背景的隔离；根节点不再是无名tab停留点。label-select的哨兵新建选项/详情表单完整正tabIndex审计仍归P14，但本Task改动不得引入新的嵌套控件或破坏标签创建确认合同。

快捷操作SubMenu可用键盘打开、选中、返回；看板/日历卡片在键盘打开菜单时有激活底色；issue-detail快捷菜单用customButtonClassName移除嵌套IconButton。关闭的右键菜单inert，关闭后无隐藏tab停留点。列表范围的pending遮罩、日历Popover/quick-add、拖动卡片/删除区在模态层下；焦点轮廓不被overflow裁切。沿用共有原语，若修到共享菜单，只修本设计已点名的键盘/状态责任并回归现有消费者，不迁M6/M7协议。

### 2.7 浮层与权限

peek打开/关上、切项、换项目/工作区、后退和宿主卸载结束自身状态。复用P11的owner及WorkItemLevelModals原provider位置，不移动到项目key下，不用全局reset关掉并存新owner。状态、项目、成员目录拒绝和当前工作项失权有可观察结束态，不显示前一项。成功/失败晚答不聚焦旧触发器、不关闭新peek、不重写新scope；键盘关闭能回到仍存在的本宿主触发器，触发器已卸载时交给页面正常焦点管理。

### 2.8 页面证据与浏览器

| 验收 | 必须观察的行为 |
|---|---|
| I3页面 | 状态分组逐组续页；优先级×标签泳道，多标签同一项出现在两道但总数不重复；A筛选/C组续页仍A∩C；切布局刷新保持 |
| I4页面 | 表格四种指定排序正反方向及第二页；日历月/周范围、拖到另一天及一条target_date动态；添加已有不含有截止日期项 |
| I6页面 | 成员Todo→Done放两邻居间，一次PATCH；DB完成/修改者/唯一动态与UI计数/位置一致；同列、below中项、尾部、拒绝都核对 |
| I7页面 | 选定状态快速添加，确认后列尾；空白标题描述与焦点，recordFocuses验证取焦瞬间已invalid；拒绝保留、双击只发一次 |
| C1 | 上述四布局的分组/续页/拖动/拒绝/列尾/添加已有，在Chromium、Firefox、WebKit各实际一遍 |
| C3列表侧 | DateDropdown键盘与单状态；七个列表/选择组件定位；外层modal背景隐藏；列表成员控件、peek页头CustomSelect；共享改动的api-tokens/自动归档/日期筛选消费者兼容实测或明确原阶段剩余项 |

每个页面故事导航前启动watchPage，核旧API、失败请求、console error/warning、pageerror、CSP、外域请求。主动制造的拒绝按精确verb/path/status/code及次数断言；基线已知第三方Canvas提示`EMOJI_CHECK_WARNING`继续由expectQuietConsole按实际加载编辑器次数精确期待，删除提示与编辑器分割仍属P14 Task10。Chromium的这条预期不可复制给Firefox/WebKit；逐浏览器观察后固定准确文本/次数，若该浏览器不发则期待空列表，不动态按收到的警告生成预期。除这些明确预期外全部为零，不能宽泛忽略403/422或console信息。只对实际触发的延迟/拒绝用route拦截，其余请求走真实服务器；数据库断言用既有fixtures/assert，不把响应本身当唯一oracle。S2仅扩大本段真实页面和准确REQUESTS，不删旧边界断言。

默认make e2e只跑Chromium全套+Firefox A4，不能据此宣称C1三浏览器通过。T15专用配置复用原globalSetup/fixtures，明确覆盖三个projects并取消继承A4 grep，只选本段页面/交互故事；`--list`核每浏览器非零且命中指定故事，再实际运行。保留每浏览器结果/trace和C3逐项记录，不能用jsdom或截图代替真实键盘/焦点行为。C3的详情/完整创建、C2/C4/C5及全应用C12保留后续责任。

### 2.9 原22路由文件的准确收口

基线盘点为14个仍存在、8个已在P10a删除；“22个改读路由”是原清单闭环，不是恢复删除文件再改。准备证据project-route-inventory.json保留每项路径/行数/哈希/相关命中。路径前缀为 `web/apps/web/`，下表core项省略`core/components/issues/`；app项省略`app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/`。

| 编号 | 原路径（省略上述前缀） | 处理 |
|---|---|---|
| 1–2 | app: issues/(list)/mobile-header.tsx；archives/issues/(detail)/header.tsx | 核显式路由；归档header已按projectId读取但本地变量名currentProjectDetails会命中守卫，局部改名，不弱化规则 |
| 3–6 | header.tsx；archived-issues-header.tsx；issue-layouts/spreadsheet/spreadsheet-view.tsx；issue-layouts/utils.tsx | 逐项核权限与调用链，utils移除router隐式目录选择 |
| 7–8 | issue-layouts/calendar/base-calendar-root.tsx；issue-layouts/empty-states/archived-issues.tsx | 显式slug/project及权限 |
| 9–10 | issue-layouts/empty-states/cycle.tsx；module.tsx | 已删除，只核删除和守卫防回归 |
| 11–12 | issue-layouts/empty-states/project-issues.tsx；project-view.tsx | 前者迁移；后者已删除 |
| 13–16 | issue-layouts/kanban/base-kanban-root.tsx；kanban/roots/cycle-root.tsx；list/base-list-root.tsx；list/roots/cycle-root.tsx | 两base核显式scope，两cycle根已删除 |
| 17–19 | issue-layouts/quick-action-dropdowns/archived-issue.tsx；cycle-issue.tsx；module-issue.tsx | archived补显式scope；后二者已删除 |
| 20–22 | issue-layouts/spreadsheet/base-spreadsheet-root.tsx；spreadsheet/roots/cycle-root.tsx；workspace-draft/delete-modal.tsx | base补scope；cycle已删除；draft已由P11使用workspace props，核正确性，不硬塞project权限 |

`getGroupByColumns`及内部project/state/label/created_by/assignee目录选择拆到 `L/group-columns.tsx`，与拖放/显示工具分责，接显式workspace/project或明确的workspace范围；无scope不退回router。同步rootStore读取仍只能同步，按需HTTP由现有宿主负责。更新全部调用方（包括profile/归档/子项），它们只做合同兼容，不宣称P13/P14页面验收。守卫保留M3原范围并覆盖原22路径，含已删除路径的样本以防复活；同步修actual test fixture省略scope的旧写法。剩余M7读者仍在，不提前删全局projectMemberIds getter。

## 3. 补充裁定

| ID | 裁定、理由与代价 |
|---|---|
| D1 | 只做文档静态预检，不做产品原型/逐Task重放；遵从负责人“暂不实现”，按02§3调整方法。代价是本次不能证明运行行为，全部A项保留开放。 |
| D2 | 15个设计粗项细分16Task，七下拉分两组，其余边界不变；预计约一万至一万七千行变更含测试删除文档，实施前据实际diff重估，超额先局部重切，不能借估算省验收。 |
| D3 | IssueSearchService随T1真实消费者同时建立；已有DetailResource/root权限水位承接搜索，不加新同步框架。工作区搜索前置水位用空project集合，返回后逐项目证明；代价是冷目录可能多一次现有目录读取，不能冒充旧DTO绕过。 |
| D4 | 原22路径按14存在/8已删逐项核验；删除是已完成收窄的证据。规则仍覆盖原路径，不延长旧getter例外；M7最后getter删除保留原归属。 |
| D5 | 单一日期组件本段统一修，覆盖api-tokens及M6/M7共享调用编译兼容；P13 R36共享DateDropdown根因随T8覆盖，P13仍须验其完整筛选页面，R38不提前关闭。 |
| D6 | @nerve/ui泛型清理、copy-hook全量、parent搜索/详情正tabIndex/C4按原P14–P16安排；本段只处理点名列表责任。C1/C3必须另跑三浏览器配置，默认Firefox A4不是验收证据。 |
| D7 | 日期范围是Calendar交互，沿DateDropdown单Popover状态/Panel定位，而非机械套无搜索Listbox；设计§7.5同步澄清，不改变调用props、日期语义或权限。代价是须回归共享日期范围消费者。 |
| D8 | 保留EMOJI_CHECK_WARNING的既有精确预期直至P14，不扩大P12到编辑器分割；三浏览器各自固定预期，其余警告/错误一律失败。独立builder-check指出此范围冲突，修订仅补足文档验收边界。 |
| D9 | 多选结果分确认ID与失败ID，T1三个调用方同时消费；同scope跨词保留已选、换业务/权限scope清空。只有ID/错误的UI结果不作为原生Issue转换层，避免日历部分成功重试重复写。 |
| D10 | T4 快建使用既有 DetailScope/权限水位和显式业务 scopeKey；标签 none 与负责人 none 即使预填载荷相同也结束旧表单。四布局共用 Field 控件；实际 DOM 挂载及解除 pending 后取焦，避免 RHF reset/ref 注册窗口。代价是目录/分组/权限身份改变时结束旧输入，符合当前宿主合同。 |
| D11 | T5 列表与看板的平面卡片统一用现有 closest-edge 产生上/下落点，失效目标不猜尾部。确认后的高亮/滚动由宿主持有，可取消 timer 和动画帧；日历、侧栏原高亮调用同步兼容，不扩展其业务写协议。泳道继续直接用 subgroup_totals，补原生续页/折叠按钮；独立的日历确认流程留 T7。 |
| D12 | T6 表格只投影确认偏好；metadata 用生成 IssueListOrder，旧筛选显示词汇留给其现有消费者，透过已有 projectListQuery 读取确认原生值，不迁公开协议。首屏重试移到布局骨架外，续页读空组 loader；表格 ref 精确为实际 div，issue-row 仅类型同步，其既有菜单键盘问题随 T11 处理。 |
| D13 | T7 日期拖动反馈统一由已存在根宿主持有并在确认后触发，日期格只负责目标和本地日期限制；CalendarChart 用实际滚动容器挂载/释放 adapter，移除重复移动端区块。确认写、权限归属、选择器部分结果及 API 均不变。 |

以上为已定设计内的执行细化，没有待负责人裁定的新协议/架构变更。若实施发现现有权限、搜索或排序合同不能实现这些行为，先提交已完成工作并上报，不加旧形状适配或放宽守卫。

## 4. 验收标准

- [ ] A1 原生搜索及三个消费者同Task接通；空词/none/for/source/query的准确请求和当前拒绝有测试，旧模块实例消失。
- [ ] A2 搜索候选只存ID，canonical及权限水位唯一；query防抖窗、scope/ABA/退役/晚答和目录拒绝不泄漏旧数据。
- [ ] A3 原22路径各有结论，守卫扩围且不放宽；同步helpers无HTTP、无router隐式选择，跨scope首次渲染正确。
- [ ] A4 列表及泳道逐组/格续页、唯一总数、过滤交集和P11编号列宽不回归；I3页面通过。
- [ ] A5 快速添加确认、字段提示/拒绝焦点/组预填/防重复通过；I7页面通过。
- [ ] A6 看板确认后移动、中间below/列尾稳定、多值none及拒绝通过；I6页面的UI/请求/DB三层证据一致。
- [ ] A7 表格原生排序、二页、偏好确认/失败隔离和子项browse链接通过；无旧排序localStorage真值。
- [ ] A8 日历月/周/local day/范围/单次拖动与添加已有通过；部分失败只重试失败项，I4页面通过。
- [ ] A9 DateDropdown及七下拉的一份状态、定位、模态兼容、共享消费者回归通过；菜单键盘/激活/inert/无嵌套按钮与焦点层叠可观察。
- [ ] A10 peek与实际宿主生命周期、并存owner、后退/切scope/权限/晚答处理通过。
- [ ] A11 watchPage/S2准确请求与零意外失败；C1三浏览器、C3列表侧逐项记录，无继承grep空过。
- [ ] A12 修改范围dead/oxlint/knip、完整本地F、至少15项不同生产性质变异、准确最终分支CI及整分支deep-review/修复复审完成；交接局部结果真实。未经后续授权先不合并。

## 5. 不在范围与交给后续

| 接收方 | 剩余责任 |
|---|---|
| P13 | 富筛选/显示设置完整I5、归档与个人主页、R38；日期共用根因已在P12实际修好才可引用，R36的筛选页面仍实测 |
| P14 | ParentIssuesListModal的for=parent、M7查重编译接通；详情/完整创建/父子关联完整故事，C3详情侧/C4/C5/C10/C11，详情正tabIndex/哨兵选项及copy-hooks所属项 |
| P15 | 评论/版本/草稿/命令搜索，最后ProjectService/WorkspaceService搜索与旧DTO，复制C8 |
| P16/收尾 | @nerve/ui下拉泛型最后清理、getTabIndex统计、全局dead/no-shadow、C12、设计§14/总体§9.4、输入交接总关闭及M5–M8 M4-closeout |
| M5–M7 | 文件与迭代/模块/视图/收集箱领域接口和页面；本段共享控件修复不等于这些领域已接通 |

## 6. 风险

拖放目标与过滤分页边界、native drag在三浏览器的事件差异、日历本地日期、搜索的即时失效与延迟发请求、七下拉共享调用是主要风险。对应测试必须看实际请求/焦点/DOM及DB，不用mock产物或只看类型检查推定通过。高风险共享协议缺口按负责人纪律上报；现有批准范围内的组件状态、错误、生命周期局部根因自主解决。

## 7. 已知限制、交接和关闭条件

本次准备不改变产品行为。P11已完成的数据层不等于P12的页面可用性证据；已有API故事和jsdom测试也不能代替C1/C3。准备证据在主目录 `.superpowers/sdd/P12-preparation-20261030/`，未来归档须与实施worktree证据一起保留。M4§14/总体§9.4、输入handoff顶层open和M5–M8最终closeout保持开放。

实施需A1–A12及Task16完整闭环；准确最终HEAD的CI后置回执存忽略证据目录，不在提交里循环记录自己的SHA。阶段完成先保留分支/worktree等待负责人集成安排。

## 附录 A：准备验证记录（2026-10-30）

本轮静态盘点基于main `ce3c39ab`，沿既有源码、生成schema、调用链和测试逐项核合同。没有原型、逐Task重放、编译、Vitest、E2E或浏览器操作记录；预检发现、独立builder-check和三视角deep-review见review。P11准确main CI及归档清理单独见P11 review集成节，不计为P12运行验证。
