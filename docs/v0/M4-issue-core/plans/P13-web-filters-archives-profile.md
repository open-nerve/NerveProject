# M4/P13 筛选、归档与个人主页实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
> 负责人已授权实施，按既有 inline/executing-plans 方式从T1推进；不因上面的通用推荐自行开启逐Task代理。阶段末使用deep-review、修复复审，先不合并。

**Goal：** 完成富筛选/显示设置、归档与个人主页页面，以及I5/I15/I19、C2/C9对应部分。

**Architecture：** 复用IssueRootStore的canonical、PermissionReads、IssueListBinding和确认写入队列。UI只补当前宿主的控件状态、合法配置与反馈；项目设置由服务端确认，归档/profile设置仍在既有本地helper中。不新增后端/API/schema或全局状态所有者。

**Tech Stack：** 仓库锁定的React、MobX、Headless UI、Propel、react-popper、Vitest、Playwright；无新增依赖。

**Spec：** [P13 spec](../specs/P13-web-filters-archives-profile.md)。基线main ff89d0a1219a8d09ff924f5bd5257c486cd7e2b8；分支codex/m4-p13；工作树 .codex/worktress/codex/m4-p13。C/L/R/P缩写及文件结构见spec§2.1；新增目标尚不存在，不视作路径缺失。

## Global Constraints

- 主目录永远main，独立分支只用约定worktree；系统/usr/bin/git负责push，详细中文提交保留feat/fix/test/docs(M4/P13)前缀。
- plane/和refer/只读；不动已有容器和Docker；不全局安装，不加设计未点名依赖；不用stash/clean/reset --hard/force，不提取钥匙串或浏览器凭据。
- 准备快照的phase为M4/P12；实施T1改tools/keywords.json为M4/P13，并核实际到期例外，不统一延期。
- IssueRootStore统一权限失效；不复制canonical或新增权限观察者，不放宽守卫；公开类型来自生成API。无any、双重断言或宽Partial假实体。
- 新生产文件约400行内；超出按职责拆分。新增helper须有真实复用，不复制整个FakeNerve/server fixture。测试长表必要时在review记理由。
- 新增/改动文案同步P/i18n/src/locales/en、zh-CN及键；错误按现有errorMessageKey/useRefusalToast，不拼造成功。
- 当前修改拒绝不改确认数据；修改成功后读取失败不回滚、不重发修改。SessionChanged或旧宿主的错误不显示旧提示。
- 使用已有FakeNerve、fake-issues/issueStoreTab、fake-projects、heldChange/fake-time；E2E复用fixtures/{test,api,issues,browser,issue-list-page}、assert/issue和watchPage；不靠sleep碰时序。
- 局部问题自行根因修复；意外架构回归、跨模块协议/权限归属调整或高风险问题先提交已完成工作再上报。
- 原型/逐Task重放按02§3调整为静态预检；不预填生成hash、红绿、执行数量或未来CI。所有本计划checkbox均属未来实施。

## Review Focus

1. 首次defaultOpen时选项尚未到、随后换条件/卸载：加载与当前失败可见，旧成功/finally不写新状态。T2真实DOM与deferred options覆盖。
2. 先选起日、未选终日就重渲染/换项，以及UTC±/闰日：不丢草稿、不清确认条件、不偏一天。T3范围及日期测试覆盖。
3. 同字段连改、不同字段并发、PATCH成功后GET失败：保留最后确认设置、嵌套calendar/false不丢、重试只GET。T4/T10覆盖。
4. 从有权变访客再变回、目标A→B→A：无旧工具栏/结果/弹窗/旧提示，已确认命令不重复。T6–T9/T11–T12覆盖。
5. storage损坏/不可用、合法选中ID不在已加载目录、隐藏project限制：安全回显和恢复，不静默丢条件、不越权预取目录。T5/T9/T12覆盖。

## 验证与提交协议

逐Task：必要行为测试→确认行为红→最小实现→绿→有效变异红→精确恢复源码/hash→恢复绿→受影响回归/类型/diff-check→要求和质量自查→精确文件提交。缺文件/编译错误不是行为红。已由前段满足的性质只补真实消费者回归和必要变异，不故意破坏代码伪造首次红。

台账放worktree的.superpowers/sdd/P13-web-filters-archives-profile/：每次保留cwd、argv、tested SHA/dirty输入及哈希、raw、退出码、执行数/skip、变异diff和恢复核验。本轮准备目录在主项目.superpowers/sdd/P13-preparation-20261031/，未来归档必须包含两处。

聚焦命令从worktree根执行（此处为T1主测试；T1–T9完整argv见附录B）：

~~~sh
pnpm --filter web exec vitest run core/components/work-item-filters/filter-config.test.tsx
pnpm --filter web check:types
git diff --check
~~~

包源码有改动时先执行pnpm exec turbo run build --filter=web^... --force；包自身测试用该包已有脚本，不借旧dist。只有非零实际执行/退出0才算绿。各Task可把多个目标放同一个vitest argv，不并发运行数据库或完整门禁。

有产品变化的检查点T4、T9、T13推送前，串行完整F：

~~~sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
~~~

T14有产品修复时补受影响验证，最终F覆盖全部最终产品输入。纯文稿可核路径集合/输入hash复用未变证据，不冒称重跑；准确分支CI独立核，main CI等未来授权集成。默认E2E的Firefox只跑A4，三浏览器矩阵按T13另跑。

## Task 1：原生筛选配置与关注人条件（约450–800行）

**Files：** spec§2.1 T1；新C/components/work-item-filters/filter-config.test.tsx；tools/keywords.json；必要的P/utils导出和i18n键。
**Interfaces：** 消费现有toProjectFilters(expression):ProjectIssueFilters、toMemberFilters(expression):ProjectIssueFilters & Pick<MemberIssueQuery,"project_id">、toFilterExpression(filters):TWorkItemFilterExpression；新增getSubscriberFilterConfig<P extends TFilterProperty>(key:P):TCreateFilterConfig<P,TCreateUserFilterParams>，由现有配置hook组合，不新增服务。

- [x] Step 1：核基线/路径/phase与exceptions，列出三页当前allowedFilters；写rendersSupportedNativeFilters和keepsSubscriberAndProjectRestriction，真实配置/HOC下断言合法subscriber正常chip、保存后仍subscriber_id；profile改priority不丢project_id。
- [x] Step 2：运行新filter-config及core/lib/issue-filters.test.ts、filters-hoc/base.test.tsx；预期现有缺subscriber配置的行为断言失败，不能把无匹配算红。
- [x] Step 3：补subscriber配置、三页允许条件及翻译；me/none精确按spec，profile已有project_id范围可编辑。共享HOC仍由P12 layout-effect注册，冷目录不清合法值；不新增created/updated过滤。
- [x] Step 4：同组测试绿、类型绿；M01去掉subscriber生产配置使真实chip/请求断言红，恢复绿；检查无旧URL或第二转换器。
- [x] Step 5：精确提交：feat(M4/P13): 补齐富筛选的原生条件配置。台账记录A1与原生转换回归。

## Task 2：四选择器的加载、翻译和当前性（约550–1000行）

**Files：** spec T2；新C/components/rich-filters/filter-value-input/select/use-filter-select-options.ts及.test.tsx；新C/components/rich-filters/select-inputs.test.tsx；selected-options-display.tsx。
**Interfaces：** 新useFilterSelectOptions(config:TSingleSelectFilterFieldConfig<string>|TMultiSelectFilterFieldConfig<string>):{options:IFilterOption<string>[];loading:boolean;error:unknown;retry:()=>void}，仅single/multi两处消费；删除没有读者的loadOptions。getFormattedOptions保持既有选项形状。

- [ ] Step 1：真实CustomSearchSelect下写opensAnEmptyConditionAfterOptionsArrive、ignoresPreviousConfigAnswer、showsOnlyCurrentReadFailure、keepsUnknownSelectedIds；覆盖初开、配置切换、旧finally、卸载、无候选与合法未知ID、en/zh-CN三文案和每次onOpen一次。
- [ ] Step 2：运行select-inputs与新hook测试；预期旧loadOptions会接受旧答/只console.error等对应断言红。
- [ ] Step 3：实现局部请求epoch/cleanup与重试；四消费者传三翻译props；loading与defaultOpen衔接不丢首次打开，选项到达后不再点一次。占位/更多数量可翻译，保留未知ID条件；禁止新增领域缓存。
- [ ] Step 4：聚焦绿并回归现有C/components/dropdowns/member/base.test.tsx真实CustomSearchSelect消费者测试；M02让旧答覆盖新配置、M03仅console.error、M04候选到达前消耗defaultOpen分别行为红、恢复绿。
- [ ] Step 5：提交fix(M4/P13): 让富筛选选择器跟随当前条件加载。记录A2/Review Focus 1。

## Task 3：范围草稿、字段错误与日期标题（约450–850行）

**Files：** spec T3；新range.test.tsx、date-filter-modal.test.tsx；既有C/components/dropdowns/basic-dropdowns.test.tsx；Field/FieldError复用@makeplane/propel/components/field。
**Interfaces：** range仍onChange(value:string[]):void；DateRangeDropdown的现有Props只增加invalid?:boolean、describedBy?:string并落到按钮，不改变value/onSelect。旧DateFilterModal(title,handleClose,isOpen,onSelect)的callback分号语义保持。

- [ ] Step 1：写keepsTheFirstDayUntilRangeCompletes、describesIncompleteAndReversedRanges、keepsLocalCalendarDays、comparesTheActualSecondDate；真实日历连续点击起终日，检查onChange仅完整范围调用、错误先可见且按钮aria关联。用当前condition换id确保旧草稿结束。另写preservesLoadedOpenRanges：已保存“2026-10-01,”与“,2026-10-31”不标invalid/不触发onChange，编辑起日后的半成品才提示，完成双端后一次替换。
- [ ] Step 2：聚焦range、basic-dropdowns、date-filter-modal；预期第一次选起日后丢草稿/错误仅颜色、date2误读date1的断言红。
- [ ] Step 3：局部草稿按condition归属；本地日期解析/格式化沿现有工具，覆盖闰日和年界；Field错误与按钮关联。start/due标题经t()；修正legacy modal的第二日期比较，保留调用格式。
- [ ] Step 4：测试绿；TZ=Pacific/Kiritimati与TZ=America/Los_Angeles分别聚焦range.test.tsx；回归P11双日期确认与真实project/cycle/inbox现存调用的类型。M05不保留起日、M06用UTC解析移日、M07去aria关联各行为红并恢复。
- [ ] Step 5：提交fix(M4/P13): 保留日期范围草稿并关联字段提示。记录A3；不将共享控件回归写成M6/M7业务验收。

## Task 4：项目显示设置与按钮合同（约600–1100行）

**Files：** C/components/issues/filters.tsx；L/filters/header/helpers/dropdown.tsx及.test.tsx；L/filters/header/display-filters/真实控件；C/hooks/use-project-preferences-follow.ts及已有.test.tsx；新L/filters/preferences-controls.test.tsx。
**Interfaces：** 原useProjectPreferencesFollow()返回(slug:string,id:string,change:()=>Promise<void>)=>Promise<void>保持；当前性复用useDetailScope和root.permissionReads，不新增root观察者。ProjectIssuesFilter.updateFilters/updateFilterExpression签名保持。

- [ ] Step 1：用真实RootStore/FakeNerve挂载控件，写waitsForConfirmedPreferences、keepsOtherFieldsAndCalendar、doesNotFollowAfterPermissionABA、retriesOnlyTheReadAfterConfirmedSave。按钮默认/自定义两支断言name、expanded、disabled、Escape回焦；盘点七个M4生产调用并给图标分支title（项目/profile两手机头、项目filters、归档header、profile-filter、子项filters/display-filters）；全仓其余消费者只兼容复核。
- [ ] Step 2：运行新控件、扩展后的既有hook测试及project/filter.store.test.ts；预期自定义按钮无语义或旧宿主跟进的对应失败；已成立队列性质只回归。保留原hook测试的确认、拒绝、读取失败和导航性质，并补真实root权限/会话ABA断言。
- [ ] Step 3：所有Promise有当前scope反馈，确认后才查询新列表；连续同字段/不同字段不覆盖、false/null/calendar未改字段保留。补FiltersDropdown实际按钮语义与实际Panel定位；不重写P12单开关。
- [ ] Step 4：聚焦绿、共享受影响消费者绿；M08乐观显示未确认偏好、M09丢嵌套calendar字段、M10权限ABA仍跟进各红、恢复绿。检查点F串行全绿后push准确提交。
- [ ] Step 5：提交fix(M4/P13): 完成显示设置的确认反馈和键盘入口。记录A2/A4及F输入。

## Task 5：归档列表与本地设置（约500–950行）

**Files：** spec T5；新L/roots/archive-page.test.tsx；新C/hooks/use-local-issue-filter-update.ts及.test.tsx（T9的profile桌面/手机共用）。
**Interfaces：** useLocalIssueFilterUpdate(workspaceSlug:string,projectId:string|undefined,targetId:string,member:boolean=false):(change:()=>Promise<unknown>)=>Promise<void>；只消费Promise/当前宿主拒绝，数据和GET重试仍归原filter store/list binding。archived原fetchFilters/updateFilters/updateFilterExpression签名不变。

- [ ] Step 1：写restoresArchivedLocalFilters、doesNotPatchProjectPreferences、doesNotFetchForGuest、preservesSavedSettingsWhenListReadFails；覆盖richFilters迁移、损坏/storage抛错、跨账号/workspace/project以及遗留kanban/subgroup设置。
- [ ] Step 2：运行archive-page、hook、archived/filter.store.test.ts和list-mount测试；预期无处理Promise/不支持布局等实际缺口的断言红。
- [ ] Step 3：归档只list、查询恒archived=true；页面能力规范化不能影响其它helper读者。现有归档单测接受kanban/孤儿子分组的预期随本期能力迁移，同时保留合法兄弟字段、false、账号隔离和读取拒绝性质；改到的旧测试使用已有真实fixture替代双重断言假root。接当前本地更新callback、可见首读/续页错误与GET retry，禁字段编辑/快建/拖动；恢复/删除权限由T6/T7独立保留。项目有效角色为访客时不拉归档列表/筛选目录。
- [ ] Step 4：测试绿；M11丢actor存储scope、M12读写rich键再不一致、M13归档开inline写各红、恢复绿。回归已通过P12project-lifecycle/row-expansion。
- [ ] Step 5：提交fix(M4/P13): 接通归档页的本地设置与只读列表。记录A5/A9。

## Task 6：归档确认与恢复的一次反馈（约450–850行）

**Files：** ArchiveIssueModal及两个直接调用；实际detail/peek命令；新archive-issue-modal.test.tsx；既有detail-commands.test.tsx、peek-overview/restore-menu.test.tsx。
**Interfaces：** ArchiveIssueModal.onSubmit:()=>Promise<void>必传，data/dataId仍用于已知快照；scope含workspace/project/item/本次打开。根archiveIssue/unarchiveIssue及确认投影不改。

- [ ] Step 1：写holdsOneArchiveUntilConfirmation、keepsDialogOnRefusal、doesNotCompleteANewerOpen、doesNotReportRestoreSuccessOnRefusal；真实modal+raw命令走一次，缺item不能成功；同一tick双击只一次请求，pending不能dismiss。
- [ ] Step 2：运行modal、detail-commands、restore-menu；预期旧h3命名、正tabIndex、缺pending锁/旧宿主成功等真实失败。
- [ ] Step 3：ModalTitle、无正tabIndex、同步pending锁和稳定快照；按当前useDetailScope结束成功/错误/finally。归档modal单独负责反馈，两个callback只返回原命令；恢复保留P12反馈归属，不再catch后假成功。
- [ ] Step 4：绿；M14去pending锁、M15旧宿主仍关闭新窗、M16恢复吞拒绝后报成功分别红并恢复。断言补读失败一次POST、后续仅GET；项目成员/guest/终态矩阵按原API不扩大。
- [ ] Step 5：提交fix(M4/P13): 完成归档确认的提交锁和宿主反馈。记录A6。

## Task 7：归档详情只读与路由身份（约400–850行）

**Files：** spec T7；新L/roots/archive-detail.test.tsx；按实际需要修改详情只读props、归档快捷菜单与header/body owner。
**Interfaces：** 归档详情key=JSON.stringify([workspaceSlug,projectId,archivedIssueId])，覆盖页头和正文；保留IssueDetailHost/Peek的原owner。可编辑属性与restore/delete命令分别使用既有权限。

- [ ] Step 1：写keepsArchivedContentReadOnlyWithAuthorizedCommands、endsArchivedOverlaysOnBack、doesNotCloseTheNextOwnerFromOldCompletion；archive source A打开菜单/弹窗，Back到B再返回A，释放旧请求；核无编辑POST/PATCH但有权restore/delete可达。
- [ ] Step 2：运行archive-detail与detail-host/host-actions现存测试；预期旧route子树复用或只读传播漏项对应断言红。
- [ ] Step 3：在现存layout边界key化实际owner，页头与正文统一生命周期；详情issue.read仍可对有权guest只读，归档列表AM门禁保持。无权restore/delete不显示；只读不使合法命令失效。
- [ ] Step 4：聚焦绿；M17移除归档key后Back测试红并恢复；模块级IssueService检索保持零，已有resource的拒绝/GET重试不回归。
- [ ] Step 5：提交fix(M4/P13): 按归档工作项身份结束详情浮层。记录A5/A6，不关闭P14完整详情。

## Task 8：个人主页路由类型、工具栏与身份（约500–950行）

**Files：** spec T8；新C/lib/profile-view.ts/.test.ts；新C/components/profile/profile-route.test.tsx；profile page/layout/header/mobile-header及真实viewId传递链。
**Interfaces：** parseProfileView(value:string|undefined):TProfileViews|undefined仅显式三个值；ProfileIssuesPage.type:TProfileViews；两个Profile布局props.viewId:TProfileViews；BaseList/BaseKanban/useIssueListMount/IssueActions.fetchIssues的可选viewId收窄为TProfileViews，非profile调用省略即可。下游row-expansion的字符串scope不必改成领域view。

- [ ] Step 1：写rejectsPrototypeNamesAsViews、neverMountsGuestToolbarsOnDesktopOrMobile、endsAllProfileOverlaysWhenTargetOrViewChanges；输入constructor、toString、未知、undefined无成员列表GET；合法三值原样向下，无断言转型。
- [ ] Step 2：运行profile-view/profile-route及list-mount/route-scope；预期原in对象判断/访客头部/缺key行为红。
- [ ] Step 3：route一次收窄，严格类型传递并同步全部调用/测试；key含workspace/user/view且涵盖页头/手机/列表/peek；effective workspace role只AM挂载工作项工具栏和子树，M3成员摘要保留。
- [ ] Step 4：聚焦与types绿；git grep -n 'as TProfileViews' -- web预期无输出（grep退出1是零命中）；M18放回in对象判断、M19让guest挂工具栏各红并恢复。
- [ ] Step 5：提交fix(M4/P13): 在个人主页路由收窄视图并隔离页面身份。记录A7/A8。

## Task 9：个人主页三关系、目录与本地偏好（约550–1100行）

**Files：** spec T9；profile mobile-header复用T5本地更新hook；新C/components/profile/profile-page.test.tsx；WorkspaceLevelWorkItemFiltersHOC只补当前已持有目录输入。
**Interfaces：** profile.fetchIssues(slug,userID,loader,options,view:TProfileViews)沿原签名；fetchNextIssues(slug,userID,group?)沿原binding；本地key保持actor/workspace/target三维，不增加view维度。新callback消费T5接口。

- [ ] Step 1：写keepsRelationAndRestrictionOnNextPage、sharesPreferencesOnlyAcrossViewsOfOneTarget、keepsUnknownLabelAndDoesNotPrefetchDirectories、doesNotPublishOldProfilePage；真实root、三种关系有交集、group二页、换view即时设置、A→B→A、storage拒绝、目录仅可见已持有。
- [ ] Step 2：运行profile-page/filter.store/use-profile-member和两布局相关测试；预期floating Promise/labelIds未接/旧view等现有缺口行为红。
- [ ] Step 3：列表和看板按profile允许能力工作；项目/状态组/优先级/标签分组及真实total/cursor；current view从T8 typed边界传，不靠晚effect猜relation。桌面/手机/富筛选都消费Promise；列表读错误可见GET retry，不再次本地“保存失败”。
- [ ] Step 4：未知合法标签ID保留条件，不显示失权目录，禁止逐项目预取或workspace-labels接口；project_id限制修改别项仍保留；M20把view丢回assigned、M21改条件丢project_id、M22冷候选清掉合法label各红、恢复绿。检查点F通过后push。
- [ ] Step 5：提交fix(M4/P13): 完成个人主页列表与本地筛选隔离。记录A8/A9/Review Focus 5。

## Task 10：I5页面偏好与富筛选故事（约450–850行）

**Files：** e2e/stories/issue/i5-list-preferences.spec.ts；复用issue-list-page、watchPage、现有API/DB断言；仅两处以上真实复用时抽新helper。
**Interfaces：** 保留已有I5(API)，新增标题I5 (page): ...；API只造前置数据，UI操作生成被断言的preferences PATCH。

- [ ] Step 1：按原API故事搭建A/B、mine/yours/outside/backlog/labelled，页面设置started、assignee=me、label=none与两个target范围2026-10-01..31及15..20；核结果为交集。布局/分组/子分组/排序/空组/子项/11属性逐类可见验证，保留calendar未改键。
- [ ] Step 2：当前make build后，在e2e运行pnpm exec playwright test stories/issue/i5-list-preferences.spec.ts --project=chromium --grep '(page)'；预期非零执行，页面缺口失败原样记；不能靠测试主动发PATCH代替操作。
- [ ] Step 3：实现故事并只修发现的局部产品根因；刷新/另账号/另项目隔离。故意拒绝PATCH时原UI/DB不变；确认PATCH后只让列表GET失败，偏好保留且重试零新PATCH。
- [ ] Step 4：精确UI/请求/DB三层断言绿，watchPage无意外失败；页面生产变异至少使一个偏好保存性质红并恢复。预期拒绝仅精确该请求，不能忽略全部4xx。
- [ ] Step 5：提交test(M4/P13): 验证页面筛选与显示偏好的保存。记录A1–A4。

## Task 11：I15归档页面故事（约400–850行）

**Files：** e2e/stories/issue/i15-archive-issue.spec.ts；fixtures/assert/issue既有删除图断言。
**Interfaces：** I15(API)保留；新增I15 (page): ...，沿原archive/unarchive/delete接口，不新增后端夹具写入。

- [ ] Step 1：真实行菜单归档终态项，pending双击一次POST，确认DB/活动/活动列表消失；归档列表刷新保持筛选，详情只读；restore回活动列表且一次成功提示。对非终态/403拒绝留原事实。
- [ ] Step 2：build后执行pnpm exec playwright test stories/issue/i15-archive-issue.spec.ts --project=chromium --grep '(page)'；原样记录失败，不把API故事算页面通过。
- [ ] Step 3：加有权删除归档项和expectIssueDeleted；只读详情的字段/评论/关系/链接无修改请求；有权guest详情与AM列表不同权限分别核，不把HTTP404改403。Back后旧弹窗及旧回答不能动新页。
- [ ] Step 4：故事绿、请求次数与DB审计精确；拒绝和确认后补读失败均验证不重复写；可复用T6有效变异但记录真实当前hash，不能称新运行。
- [ ] Step 5：提交test(M4/P13): 验证归档只读与恢复删除流程。记录A5/A6。

## Task 12：I19个人主页页面故事（约500–1000行）

**Files：** e2e/stories/issue/i19-member-issues.spec.ts；复用既有newAccount/member/issue fixtures。
**Interfaces：** I19(API)保留；新增I19 (page): ...；三relation只由页签行为选择，预期集合从明确种子推导。

- [ ] Step 1：以原故事active/second/hidden/exited/archived/own项目范围搭页面场景，assigned/created/subscribed有真实交集；三页签、两布局、四组型，至少一组二页保持relation/原筛选，total不按加载长度代替。
- [ ] Step 2：build后执行pnpm exec playwright test stories/issue/i19-member-issues.spec.ts --project=chromium --grep '(page)'；必须实际页面与非零执行。
- [ ] Step 3：workspace guest桌面/手机无工具栏、列表GET为0；workspace admin兼project guest在OwnOnly项目只看到自己创建项；未加入/退出/归档项目与计数不泄露。换用户/view及Back清旧弹层，延迟旧回答不能闪现。
- [ ] Step 4：筛选与显示刷新恢复、跨账号/目标隔离，同目标三view共享；project_id限制不丢，冷标签限制如实验收。故事绿、watchPage断言无意外错误，M19/M20至少一种页面变异红并恢复。
- [ ] Step 5：提交test(M4/P13): 验证个人主页三关系及访客边界。记录A7–A9。

## Task 13：C2/C9三浏览器与完整门禁（约450–950行）

**Files：** spec T13；新增e2e脚本test:m4-filters调用playwright.m4-filters.config.ts；修改到的关键词样本、翻译和dead成员报告。
**Interfaces：** testMatch只匹配i5-list-preferences、i15-archive-issue、i19-member-issues、filter-archive-profile-interactions；grep只(page)|C[29]:；projects显式chromium/firefox/webkit，不能继承base.projects的Firefox grep。

- [ ] Step 1：写C2: .../C9: ...真实交互：四选择器逐一的键盘/加载/defaultOpen/搜索清空、FiltersDropdown默认与自定义名字/expanded/Escape回焦、两日期范围/错误、归档modal焦点、两页面Back/手机权限；定位断言实际panel与按钮邻接且可点击，不只DOM存在。
- [ ] Step 2：新配置--list分别核三个浏览器都有预期页故事且无API故事；先Chromium聚焦，记录实际失败；配置列表检查不是产品通过。
- [ ] Step 3：修发现的局部根因，保持modal外层隔离/portal/P12共享消费者；按实际入口逐浏览器枚举EMOJI旧警告，不能新增宽泛console忽略。精确审改到文件dead/props/exports，oxlint上限不抬高，禁止新的宽泛keywords豁免。
- [ ] Step 4：当前make build后从e2e串行运行下列三条，预期每条非零、0失败/0skip；再完整F并核真实输入，分支push/CI准确SHA另取证。
  ~~~sh
  pnpm exec playwright test --config=playwright.m4-filters.config.ts --project=chromium
  pnpm exec playwright test --config=playwright.m4-filters.config.ts --project=firefox
  pnpm exec playwright test --config=playwright.m4-filters.config.ts --project=webkit
  ~~~
- [ ] Step 5：提交test(M4/P13): 补齐筛选归档和个人主页的跨浏览器验收。记录A10/A11。

## Task 14：整分支deep-review与分支交付

**Files：** 本spec/plan/review；M4-design§15；M1/M2/M3-closeout中本期子项；docs/reviews下一号整体审核；忽略目录回执。
**Interfaces：** 固定审查范围ff89d0a1..最终待交付HEAD；源码输入集合、原始raw、变异生产diff/恢复hash、准确CI组成证据，不从文稿自引用自身SHA。

- [ ] Step 1：按deep-review全文审改动文件及相关测试，顺序跑逻辑/集成/覆盖三视角；列每个写入口确认、拒绝、旧scope、确认后读失败四格。跨Task类型/调用方、页面权限/队列/当前性、模块边界、无测试空过逐一核。
- [ ] Step 2：至少15个不同生产性质有效变异：优先M01–M22，记录所属层与当前hash；同一性质换输入不多算，编译错误/无匹配不算红。没有服务端锁路径变化，不捏造新锁/Go-race实验；若出现服务端改动先按架构纪律处理。
- [ ] Step 3：修所有局部重要问题并必要回归、复审；发现架构级问题先提交再上报。更新真实plan差异/限制，不能把未来交接关闭或已有P12成果重复计数。
- [ ] Step 4：核最终F和三浏览器输入覆盖、dead/守卫、准确最终分支CI三任务success；若产品有变补相应证据。纯文稿CI用后置final-delivery.json绑定HEAD，避免“记录自身SHA→新提交→新CI”循环。W11按06重跑，合并CI重跑仍不过则上报。
- [ ] Step 5：spec A1–A12按证据打勾、设计写“实施与终审完成，集成待办”、交接只记子项；中文提交docs(M4/P13): 记录筛选归档和个人主页的验收与整审。报告分支/基线/spec/plan/review，保持未合并，等待负责人安排。

## 自查与当前边界

本计划14Task，spec§2.1各交付均有任务，A1–A12映射见阶段review；Review Focus五项各有真实测试落点。接口新增仅局部options状态、本地更新反馈和route parser，以及现有日期按钮可选字段；不改变后端契约或权限所有权。估计任务规模不超16项及约1500行/项，实际超出时按02重新拆分并记原因。

准备阶段未执行Task或变异；负责人已授权实施，进度按各Task勾选与阶段review记录。未实际运行的候选变异仍不能当证据。


## 附录 B：逐Task聚焦命令

以下命令中的新增测试由对应Task先创建；本次实施按对应Task执行。Step 2记录真实行为失败，Step 4同命令恢复绿，退出0且非零执行才成立；每Task另执行全局types/diff-check和明确受影响回归。

~~~sh
# T1
pnpm --filter web exec vitest run core/components/work-item-filters/filter-config.test.tsx core/lib/issue-filters.test.ts core/components/work-item-filters/filters-hoc/base.test.tsx
# T2
pnpm --filter web exec vitest run core/components/rich-filters/select-inputs.test.tsx core/components/rich-filters/filter-value-input/select/use-filter-select-options.test.tsx core/components/dropdowns/member/base.test.tsx
# T3
pnpm --filter web exec vitest run core/components/rich-filters/filter-value-input/date/range.test.tsx core/components/core/filters/date-filter-modal.test.tsx core/components/dropdowns/basic-dropdowns.test.tsx core/components/issues/date-range-write.test.tsx
TZ=Pacific/Kiritimati pnpm --filter web exec vitest run core/components/rich-filters/filter-value-input/date/range.test.tsx
TZ=America/Los_Angeles pnpm --filter web exec vitest run core/components/rich-filters/filter-value-input/date/range.test.tsx
# T4
pnpm --filter web exec vitest run core/components/issues/issue-layouts/filters/preferences-controls.test.tsx core/hooks/use-project-preferences-follow.test.tsx core/store/issue/project/filter.store.test.ts core/components/issues/issue-layouts/filters/header/helpers/dropdown.test.tsx
# T5
pnpm --filter web exec vitest run core/components/issues/issue-layouts/roots/archive-page.test.tsx core/hooks/use-local-issue-filter-update.test.tsx core/store/issue/archived/filter.store.test.ts core/hooks/use-issue-list-mount.integration.test.tsx
# T6
pnpm --filter web exec vitest run core/components/issues/archive-issue-modal.test.tsx core/components/issues/detail-commands.test.tsx core/components/issues/peek-overview/restore-menu.test.tsx
# T7
pnpm --filter web exec vitest run core/components/issues/issue-layouts/roots/archive-detail.test.tsx core/components/issues/detail-host.test.tsx core/components/issues/peek-overview/host-actions.test.tsx
# T8
pnpm --filter web exec vitest run core/lib/profile-view.test.ts core/components/profile/profile-route.test.tsx core/hooks/use-issue-list-mount.test.ts core/hooks/use-issue-list-mount.integration.test.tsx core/components/issues/issue-layouts/route-scope.test.tsx
# T9
pnpm --filter web exec vitest run core/components/profile/profile-page.test.tsx core/store/issue/profile/filter.store.test.ts core/components/profile/use-profile-member.test.ts core/components/issues/issue-layouts/group-consumers.test.tsx
~~~

T10–T12的页面命令与T13三浏览器命令已各自列出。T14依最终改动选必要回归并核完整F，不凭计划文字生成通过数字。
