# M4/P13 筛选、显示设置、归档页与个人主页

> 状态：文档预检通过，未实施。分支 codex/m4-p13；基线 main ff89d0a1219a8d09ff924f5bd5257c486cd7e2b8；工作树 .codex/worktress/codex/m4-p13。负责人本轮只授权准备 spec、plan 和预检。P12 已合并、准确 main CI 全绿并归档清理，见 [P12 review §28](../reviews/P12-web-list-pages-review.md#28-授权合并main-ci与归档清理2026-10-31)。

权威：[M4 设计](../M4-design.md) §3.16–3.17、3.22、7.4–7.5、9.7、12/P13、14；[plan](../plans/P13-web-filters-archives-profile.md)；[阶段评审](../reviews/P13-web-filters-archives-profile-review.md)。原型与逐 Task 重放按 [02 §3](../../claude_to_gpt/02-method.md#3-哪些是要求哪些是做法) 调整为静态合同、调用链、既有测试核对、一次独立 builder-check 和主控三视角 deep-review。本次不以静态结论代替产品测试。

## 1. 目标

让已有原生列表、偏好和确认写入合同在富筛选、显示设置、归档和个人主页真实页面上闭环。完成 I5、I15、I19 的页面故事及 C2、C9 中归档/个人主页部分；保留每个页面的权限边界、拒绝反馈、当前宿主和本地设置隔离。

P10b 的原生转换、分组分页、IssueRootStore 权限失效方案 A，P11 的详情资源、宿主和编号宽度方案 A，以及 P12 的共享控件修正均为输入。发现局部缺口在原职责内修复，不重造服务、权限观察者或第二份 Issue。

## 2. 交付物

### 2.1 文件结构与任务

路径缩写：C=web/apps/web/core；L=C/components/issues/issue-layouts；R=web/apps/web/app/(all)/[workspaceSlug]/(projects)；P=web/packages。表中的“新”是实施目标，本次不会创建产品文件。

| Task | 文件与职责 | 必要验证落点 |
|---|---|---|
| T1 | P/constants/src/issue/filter.ts；P/utils/src/work-item-filters/configs/filters/user.ts；C/hooks/work-item-filters/use-work-item-filters-config.tsx；原 rich-filter 注册链 | 新 work-item-filters/filter-config.test.tsx；已有 issue-filters.test.ts、filters-hoc/base.test.tsx |
| T2 | C/components/rich-filters/add-filters/dropdown.tsx、filter-item/root.tsx、filter-value-input/select/{single,multi,shared,selected-options-display}.tsx；新 select/use-filter-select-options.ts | 新 rich-filters/select-inputs.test.tsx、select/use-filter-select-options.test.tsx |
| T3 | rich-filters/filter-value-input/date/range.tsx；C/components/dropdowns/date-range.tsx；L/filters/header/filters/{start-date,due-date}.tsx；C/components/core/filters/date-filter-modal.tsx | 新 range.test.tsx、date-filter-modal.test.tsx；既有 basic-dropdowns.test.tsx |
| T4 | C/components/issues/filters.tsx；L/filters/header/helpers/dropdown.tsx 及 display-filters 真实控件；C/hooks/use-project-preferences-follow.ts | 新 preferences-controls.test.tsx；扩展已有 use-project-preferences-follow.test.tsx、project/filter.store.test.ts |
| T5 | L/roots/archived-issue-layout-root.tsx；C/components/issues/archived-issues-header.tsx；C/store/issue/archived/{filter,issue}.store.ts | 新 roots/archive-page.test.tsx；既有 archived/filter.store.test.ts |
| T6 | C/components/issues/archive-issue-modal.tsx；L/quick-action-dropdowns/{project-issue,issue-detail,archived-issue}.tsx；详情/peek 的实际 archive、restore 调用 | 新 archive-issue-modal.test.tsx；既有 restore-menu.test.tsx、detail-commands.test.tsx |
| T7 | R/projects/(detail)/[projectId]/archives/issues/(detail)/{layout,header}.tsx 与 [archivedIssueId]/page.tsx；详情只读及 owner 实际消费者 | 新 roots/archive-detail.test.tsx；既有 detail-host/宿主测试 |
| T8 | 新 C/lib/profile-view.ts；R/profile/[userId]/{layout,header,mobile-header}.tsx、[profileViewId]/page.tsx；C/hooks/use-issues-actions.tsx、C/hooks/use-issue-list-mount.ts；profile 两布局根 | 新 profile-view.test.ts、profile/profile-route.test.tsx；既有 list-mount/route-scope 测试 |
| T9 | C/components/profile/{profile-issues,profile-issues-filter}.tsx；C/store/issue/profile/{filter,issue}.store.ts；workspace-level filters HOC | 新 profile/profile-page.test.tsx；既有 profile/filter.store.test.ts、use-profile-member.test.ts |
| T10 | e2e/stories/issue/i5-list-preferences.spec.ts；复用 e2e/fixtures/issue-list-page.ts | I5(page)、UI/原生请求/数据库偏好三层证据 |
| T11 | e2e/stories/issue/i15-archive-issue.spec.ts | I15(page)、归档/恢复/只读/删除 |
| T12 | e2e/stories/issue/i19-member-issues.spec.ts | I19(page)、三关系/权限/隔离/分页 |
| T13 | 新 e2e/stories/issue/filter-archive-profile-interactions.spec.ts、e2e/playwright.m4-filters.config.ts；e2e/package.json；关键词、翻译、死代码检查 | C2/C9 三浏览器；完整本地七项门禁 |
| T14 | spec/plan/review、M4 设计§15、输入 handoff 对应子项、docs/reviews 下一号报告 | 整分支 deep-review、修复复审和准确分支 CI |

文件名以真实目录为准；T1前按 plan 再核一次路径。只为真实读者抽取小 helper。新生产文件约400行内；测试确需长表时记录理由。不增加设计外依赖或服务端/API/schema 变化。

### 2.2 富筛选的词汇与表达式

沿 C/lib/issue-filters.ts 的唯一转换路径：toProjectFilters、toMemberFilters、toFilterExpression 及 projectListQuery/memberListQuery。UI 表达式可以保留既有别名，网络和项目偏好只交原生字段；不得在控件里再写一个协议翻译器。

| 页面 | 可选条件 |
|---|---|
| 项目 | state_id、state_group、priority、label_id、assignee_id、created_by_id、mention_id、subscriber_id、start_date、target_date |
| 归档 | 现有 priority、state_group、state_id、assignee_id、created_by_id、label_id、start_date、target_date；所有列表查询仍带 archived=true |
| 个人主页 | state_group、priority、label_id、start_date、target_date；原生 project_id 范围约束可显示、修改和清空，不能因改另一条件丢失 |

subscriber_id 数据链已存在，T1补对应配置与成员选项，不能用 InvalidFilterItem 代替合法已保存条件。用户类值遵守现有转换：负责人支持 none/me/UUID；创建者、提及者、关注者支持 me/UUID，不能提交 none。标签支持 none/UUID。现有状态组、优先级取值不扩大；created_at、updated_at、cycle/module/estimate 等不恢复。

非日期同字段集合按既有 IN 语义；日期允许多个非 none 条件取 AND，none 不与同字段其它条件混用；单日、双端及现有原生开区间在反向显示时不损失。非法持久化字段和不支持的operator沿P10b既有合同整字段清除；不新增有损翻译，也不能把合法日期AND改成OR或丢掉有效条件。清空后提交空对象/空数组须按原接口语义，不发旧 rich_filters 或 user-properties 请求。

项目与归档候选来自当前项目目录；个人主页项目来自当前调用者可见、已加入且未归档项目。工作区标签仍遵守 P11 spec§2.10：仅投影已持有、当前仍可见项目的 labelMap，不逐项目预取，也不新增 workspace labels API。冷页面可无标签候选；已保存但目录未加载的合法标签 ID 保留条件和中性占位，不能因候选为空自动清空。完整工作区标签目录归 M7；本期测试明确这项限制，不通过偷偷先开所有项目伪造完整性。

### 2.3 四个选择器与可访问名称

四个 CustomSearchSelect 生产调用为 add-filters/dropdown、filter-item/root、select/single、select/multi。searchPlaceholder、noResultsMessage、loadingMessage 显式来自 t()；新增/改到的文案同步 en、zh-CN 和键。选项名称、清空、无可添加条件、加载失败/重试和日期弹窗名称同样可翻译。

defaultOpen 在空条件首次出现时生效；异步候选未到时仍有可见加载状态，到达后不要求多点一次，不因重渲染/清空搜索反复开关。每次打开的回调仅一次。键盘 Enter/Space、输入搜索、选择、Escape/Tab 与焦点归还在真实页面验收，保持 P12 的单一打开状态和外层 ModalCore 隔离。

single/multi 共用一个局部 options 请求生命周期：配置/条件宿主切换、卸载后旧成功、旧错误和 finally 均不能覆盖新状态。当前读取失败显示可重试反馈，不只 console.error，不冒充“无结果”。局部 hook 只管理 options/loading/error，不缓存另一份领域目录，不引入权限事实；父级身份来自原 root/route。

FiltersDropdown 默认按钮及 menuButton 分支都有翻译后的 title、aria-expanded、disabled；七个M4生产调用先盘点，再覆盖实际可达入口；全仓其余共享消费者保留兼容，不扩大其领域验收。图标按钮不能依赖默认 “Dropdown”。Escape 后焦点回触发器，无嵌套按钮；定位 ref/style 落在实际面板，避免只给内容定位而命中区留在原处。

### 2.4 日期输入

富筛选范围保留本地未完成草稿，第一次选起日不能因 onChange([]) 丢掉起日；新建或开始编辑双端范围时，只有两个有效且顺序正确的日期才提交完整范围。未完成/反向/越界时有文字错误，Field/FieldError 与触发按钮 aria-invalid、aria-describedby 对应，不能只有红色。清空是明确动作；不能把暂未选完当成清空已保存条件。

草稿归属 condition 身份，换条件、关闭宿主、换页后不写入新条件。日期按用户本地日处理，UTC±时区、闰日、年界不移一天。已有单日及开区间（例如target_date__range值为“2026-10-01,”或“,2026-10-31”）原样回显；开区间本身有效，未编辑时不标错误、不发onChange。开始用日历改为双端范围后才按未完成草稿校验，完成后一次替换；清空仍是显式动作。

DateRangeDropdown 可沿既有 select 字段约定增加可选 invalid/describedBy，并传到实际触发按钮；保持 onSelect、受控 value、min/max、portal/defaultOpen、disabled 及 P11 双日期原子修改合同。不能将共享控件改成全局乐观缓存。旧 DateFilterModal 的 start/due 调用 title 经 t()；核对其范围比较确实读取 date2，局部修正后保留 M3/M6/M7 使用的原 callback 格式，不把旧格式接回 M4 原生筛选请求。

### 2.5 显示设置与两类保存

项目设置仍经 ProjectIssuesFilter + ProjectPreferencesService 的三个字段；layout 为 list/kanban/spreadsheet/calendar。分组/子分组/排序使用现有原生词汇和页面能力；空组、子工作项、11个显示属性、calendar.layout/show_weekends 均保留 false、null、嵌套未改字段。主分组清空或与子分组相同按现有规范清理；看板没有分组时按现有 state_id 默认，不发送非法组合。

PATCH 确认前显示最后确认值；拒绝保留原设置和数据并按码显示一次错误。同字段连续修改沿已有队列，不同字段确认不得互相覆盖。useProjectPreferencesFollow 只跟进当前 session、路由及 root 权限身份；卸载、A→B→A、权限降再升后，旧答不改新 UI、不提示。PATCH 已确认但列表 GET 失败时保留确认值，显示读取错误且重试只 GET，不能回滚或再次 PATCH。

归档/个人主页仍写 issue_local_filters，由现有 helper 以账号、workspace 和项目/目标用户隔离。归档统一 rich_filters；旧 richFilters 迁移只在已有兼容边界中。个人主页同一个目标用户的三个标签页共享筛选/显示偏好，切目标/账号/工作区隔离；relation 始终来自当前标签页，不存进偏好。storage 损坏/不可用时沿既有安全默认与会话内状态，不发偏好 PATCH，不因为恢复失败留下空白页。

本地设置已保存但列表读取拒绝，不能假称“保存失败并还原”；读取失败由当前列表的可见错误/重试处理。桌面和手机头部、富筛选 callback 都必须消费 Promise，不能无处理拒绝或吞成成功。归档固定 list，个人主页只 list/kanban；不支持的历史布局/子分组按所属页面规范化，不影响 M6/M7 的 helper 语义。

### 2.6 归档与详情

归档列表只允许项目管理员/成员；访客不发归档列表 GET。详情按既有 issue.read 及 OwnOnly 判断，可见访客仍是只读；不能把列表权限直接改成整个详情的权限。归档行/详情/peek 的标题、描述、属性、子项、关联、链接、评论等不允许写；保留有权角色的恢复和删除。字段只读与命令权限分别判断，不能一改 viewFlags 就把恢复/删除同时禁掉。

归档只对终态允许；服务端 issue.not_archivable、issue.archived、403/404/422 等仍按原码反馈。ArchiveIssueModal 用 ModalTitle，去正 tabIndex；稳定 issue 快照、必需异步 onSubmit、同步 pending 锁，缺失 item/回调不伪造成功。两个直接生产调用 project-issue/issue-detail 同步迁移。确认前留窗并阻重复；拒绝保留可重试；成功只在当前宿主提示/关闭，当前权限/宿主结束使旧跟进失效。

archive/unarchive 命令继续走既有 root/队列/确认投影。归档弹窗负责其一次成功/拒绝反馈；传入的命令不再套第二层反馈。恢复复用 P12 已闭合的一处命令反馈，不吞拒绝后再显示成功。确认后的补读拒绝只重试 GET，列表和详情不能把已归档/恢复的事实回滚。

归档详情子树 key 包含 workspace、project、archivedIssueId；页头和正文的弹窗/菜单一并属于该身份。Back/同路由换 id/关闭重开/权限 ABA 结束旧宿主，旧完成不能关闭后来打开的同一 id。P11 已移除页头模块级 IssueService，只检查不回归；本期不重复“删除”不存在的实例。

### 2.7 个人主页

在路由边界显式判断 assigned、created、subscribed，得到 TProfileViews 后向下传递。不能用 value in 普通对象接受 constructor/toString；非法值不发成员工作项请求。删除 use-issues-actions 的 as TProfileViews，同 Task 更新所有生产调用及测试；不把 string 断言挪到另一层。

有效 workspace 管理员/成员可看三关系，列表和看板支持项目、状态组、优先级、标签分组；保持每组真实 total、同组 cursor 与原筛选。按访问者授权逐项目执行 OwnOnly；workspace admin 兼 project guest 也不能越权；隐藏、已离开、归档项目不出现在结果、组名或计数。自动关注导致关系有交集是正常合同，不把三个集合编造为互斥。

workspace 访客显示“没有权限”；桌面/手机筛选工具栏与列表一起不挂载、不发列表/筛选目录取数。成员摘要的现有 M3 数据链保留，不为此新增接口。目标非成员、读失败、空结果、首读/续页失败各有正确 UI 和读取重试。

页面身份包含 workspace/userId/profileViewId，页头、手机头部、筛选、列表、peek 和它们拥有的弹层同步结束。Back、同用户换 view、同 view 换用户和权限变更都测试；key 不改变本地偏好“同目标三个 view 共享”的合同。旧列表和迟到的旧 finally 不得闪到新页面。沿 IssueRootStore.permissionReads 和已有 UI owner，不新增全局清理器。

### 2.8 页面故事与证据

| 故事 | 真实用户动作与必要断言 |
|---|---|
| I5(page) | A 操作筛选/显示设置并刷新，B 无 A 的偏好；两个日期范围取 AND、none/me、清空；确认/拒绝/确认后读失败；DOM、精确 PATCH/query 与数据库三列一致 |
| I15(page) | 从项目行归档终态项，归档列表/详情只读；恢复返回活动列表；不允许非终态和无权写；有权删除归档项仍按已有子树断言。请求、DB archived_at/活动与 UI 一致 |
| I19(page) | 三关系由真实 API 种子产生且允许交集；换目标、分页、过滤/刷新/本地设置隔离；workspace guest 无工具栏/列表 GET，admin兼项目guest仍 OwnOnly |
| C2/C9 | 每种允许筛选和清空、保存后恢复；四选择器、两个日期控件、显示按钮的键盘/名称/focus/定位；归档与profile的Back/切页/弹层；桌面和手机工具栏权限 |

种子只通过既有 API fixture；DB 仅断言。不能以 API 代替要验收的 UI 操作。导航前安装 watchPage，区分精确预期的拒绝与意外失败；零 pageerror/旧 API/无处理拒绝。既有 EMOJI_CHECK_WARNING 由 P14 清理，本段只按实际入口/浏览器精确枚举，不能加宽忽略。

新 m4-filters 配置显式建立 chromium/firefox/webkit 三个 project，限定 I5/I15/I19 页面及新交互故事；不得继承默认 Firefox 的 A4 grep。分别记录非零执行、通过、失败/skip、raw/trace、源码输入。默认 make e2e 不能代替三浏览器 C2/C9。

## 3. 补充裁定

| 编号 | 裁定、理由与错误代价 |
|---|---|
| D1 | 设计12项粗任务细分为14项，未拆新阶段。异步控件、范围输入、项目保存、归档/profile各自可独立验收；估计每项约300–1300行含测试，修复轮预留10%。切分不扩大功能边界。 |
| D2 | 本轮只有文档静态预检，phase保持M4/P12。已批准设计不重新做产品原型；代价是实际运行风险必须在T1起红绿和检查点F中验证，不称已有运行证据。 |
| D3 | 保持P11的工作区标签目录边界，仅接已持有且可见labelMap，未知合法ID不丢失；不引入逐项目扩取数或M7接口。冷页面候选可能不完整，完整目录仍交M7。 |
| D4 | typed view从route一次收窄；已有各层接口同步迁移属于设计点名工作，不另建view协议或权限归属。若发现必须改API/根权限设计，提交后上报。 |
| D5 | 归档页头旧service、R36日期单开关和R38注册根因已在P11/P12完成，本期只验真实页面与剩余局部缺口，不重复记功。 |
| D6 | 两类偏好分别维持服务器确认与本地保存；失败GET不等于写失败。单一反馈和已有scope水位防止重复操作，禁止catch空数组/假成功。 |
| D7 | 同目标三个profile view共享本地偏好，UI身份仍按view重建；不能为了key把存储范围也改变。保留P10b既有project_id约束。 |
| D8 | 旧DateFilterModal范围比较错误属于现有共享控件的局部修正，核真实调用并保留callback语义；不扩到M6/M7业务验收。 |

无新增架构决定待负责人裁定；文档预检发现的可执行性缺口在本范围内修订。若后续事实推翻此结论，先提交已完成工作再上报。

## 4. 验收标准

- [ ] A1 三页允许筛选配置与原生词汇一致；subscriber 正常显示，日期AND/none/me/清空与既有非法字段清洗保持。
- [ ] A2 四选择器的翻译、defaultOpen、异步当前性、失败重试及按钮名称/键盘/焦点在真实页面通过。
- [ ] A3 范围草稿不丢起日、错误文字与aria关联、本地日无偏移；旧日期弹窗标题/范围与共享消费者不回归。
- [ ] A4 项目三个偏好字段确认写入、并发不覆盖、拒绝不变、确认后读失败只GET重试；I5(page)三层一致。
- [ ] A5 归档本地设置恢复、固定列表、只读字段与有权命令分离；I15(page)归档/恢复/删除成立。
- [ ] A6 归档弹窗pending/ModalTitle/无正tabIndex/单次反馈，归档详情key与Back/晚答归属成立。
- [ ] A7 三view从路由严格收窄并传递，git grep “as TProfileViews”在web零命中；非法路由不请求。
- [ ] A8 profile管理员/成员三关系与四种分组、分页、OwnOnly；访客桌面/手机无工具栏/列表GET；I19(page)通过。
- [ ] A9 本地设置按账号/workspace/目标隔离，同目标三view共享；冷标签边界、project_id保留、损坏/不可用storage、切页owner均有证据。
- [ ] A10 I5/I15/I19与C2/C9在Chromium/Firefox/WebKit实际非零通过；watchPage无意外失败、旧API、无处理拒绝。
- [ ] A11 修改范围dead/oxlint/knip、逐Task行为红绿与至少15项不同性质有效变异、完整F及准确最终分支CI闭环。
- [ ] A12 整分支deep-review、局部问题修复复审、交接子项与文档证据完成；先交负责人安排，不自动合并。

## 5. 不在范围与交给后续

| 接收方 | 保持的边界 |
|---|---|
| P14 | 完整详情/创建/父子/关联/链接页面；标题与创建表单等剩余字段；编辑器分割、撤销重做、EMOJI警告删除及嵌套Escape。P13只核归档只读及已存在命令，不关闭完整详情验收。 |
| P15 | 评论/动态/版本/草稿/命令搜索、C9草稿、旧搜索最后删除和复制C8。 |
| P16/收尾 | 全局dead/no-shadow、最后getTabIndex、C12、设计§14/总体§9.4、输入handoff顶层关闭和M5–M8最终M4-closeout。 |
| M6/M7 | 迭代/模块/视图/收集箱自身接口与页面；完整工作区标签目录，沿P10a A方案和P11范围不扩。 |

不更换后端授权、数据模型、分页协议、并发协调或业务接口，不新增依赖/全局安装，不变更已有Docker和容器。

## 6. 风险

异步选项与defaultOpen的衔接、日期未完成草稿、服务器偏好与本地偏好的错误含义、共享控件兼容、归档命令和只读分离，以及同地址返回时的旧宿主响应是主要风险。对应测试须看实际UI/请求/焦点和DB，不只spy回调；共享权限/canonical的性质沿原root验证，不在组件mock中宣称通过。

## 7. 已知限制、交接和关闭条件

本轮没有产品实现、原型、逐Task重放、编译、Vitest、E2E或变异运行；A1–A12全部开放。准备材料保存在主目录 .superpowers/sdd/P13-preparation-20261031/，未来必须随实施材料归档。

接收 M1-closeout 的 typed profile view；M2-closeout §10四选择器/defaultOpen与§12归档页头实例的页面回归；M3-closeout §6个人主页/访客工具栏/key、§7筛选与日期/归档弹窗、§8文案，以及P5/P10b/P11/P12移交。本期实施结束才逐子项记录处理结果，handoff顶层保持open。M4总验收与最终交接不在本次关闭。

## 附录 A：准备验证记录（2026-10-31）

P12最终e6d1329b以--no-ff合并为main ff89d0a1，准确[CI 41087022719](https://github.com/open-nerve/NerveProject/actions/runs/41087022719)三任务成功。归档1786份实施/运行证据及14份准备证据，复制前后与清理后哈希均通过；详见P12 review§28，不计为P13测试。

本次只读源码、生成合同、既有测试与Plane对照；阅读清单区分全文、指定章节和检索。builder-check、deep-review及静态核验见阶段review与准备目录，产品验收留待实施。
