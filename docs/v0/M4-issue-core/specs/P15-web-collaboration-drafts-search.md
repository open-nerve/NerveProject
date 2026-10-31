# M4/P15 协作、草稿与命令搜索

状态：文档预检通过，尚未实施。基线 main `7a81ea18d8e9d80f079e4e30ae0e90b11be23ef9`；分支 `codex/m4-p15`。本次只完成 spec、plan 与预检；产品验收和实现步骤全部保留未勾。

## 1. 目标

完成 [M4 设计 §12 P15](../M4-design.md) 的时间线、评论/表情、描述历史、草稿、命令与搜索页面；承接 P4/P6/P8 的原生协议、P11 的资源/草稿存储以及 P14 的宿主、编辑器与复制交接。完成线是 I11–I14、I16、I18 的页面版、I20 的评论和 C6–C8、C9 草稿，既有 API 故事不能代替页面验收。

实现复用一代 IssueRootStore 及 PermissionReads、原生 service、确认写队列、useDetailScope/useSessionSWR；不重建数据层或变更 API、权限归属、数据库、清洗边界。M4 总体收尾仍在 P16 之后。

## 2. 交付范围与合同

### 2.1 文件地图与任务

以下路径相对 worktree 根：C=`web/apps/web/core`，P=`web/packages`，A=`web/apps/web/app/(all)/[workspaceSlug]/(projects)`，E=`e2e`。表中的目录/花括号为范围缩写；plan 指定实际测试路径及新增文件。

| 任务 | 主要范围 | 交付 |
|---|---|---|
| T1 | C/components/issues/issue-detail/issue-activity/{activity-comment-root,activity/actions/helpers/issue-user,activity/actions/archived-at,activity/actions/default}.tsx | 时间线、系统 actor、I12 |
| T2 | C/components/issues/issue-detail/issue-activity/{root,activity-filter,filter-root,sort-root,helper}.tsx | 过滤/排序键盘与命名、点名 lint |
| T3 | C/components/comments/{comment-create,quick-actions,card/root,card/display,card/edit-form}.tsx；时间线 helper | 评论权限、提交、命名、I11/I20 |
| T4 | C/components/comments/comment-reaction.tsx、card/display.tsx；C/components/issues/issue-detail/reactions/issue.tsx | 两类表情和 CSP |
| T5 | C/components/core/description-versions/{root,modal,dropdown}.tsx；C/components/issues/issue-detail/description-versions.tsx | 历史、还原、Markdown 复制、I13 |
| T6 | A/drafts/{page,header}.tsx；C/components/issues/workspace-draft/；C/components/issues/issue-modal/base.tsx、draft-form.ts | 个人草稿整流、I14 |
| T7 | C/components/issues/issue-modal/{form,base,draft-issue-layout}.tsx；C/components/issues/confirm-issue-discard.tsx | 创建弹窗存草稿与 C9 |
| T8 | C/components/power-k/ui/pages/context-based/work-item/commands.ts 及按职责新增子文件 | 工作项命令拆分与原生写合同 |
| T9 | C/components/power-k/config/{preferences-commands,creation/command}.ts；tools/keywords.json | 主题确认、创建路由与四文件守卫 |
| T10 | C/components/power-k/ui/modal/{search-menu,search-results,search-results-map}.tsx；C/hooks/use-issue-search.ts | 原生工作项搜索、项目本地过滤、I18 |
| T11 | C/services/workspace.service.ts；C/components/power-k/core/types.ts；P/{types,constants}/src/workspace.ts | 旧搜索方法/类型/常量与最后模块实例清零 |
| T12 | E/stories/issue/i16-auto-archive.spec.ts；E/fixtures/{clock,auto-archive,auth}.ts | I16 真实时钟页面与系统操作者 |
| T13 | C/components/issues/{create-issue-toast-action-items,issue-detail/issue-activity/helper,issue-layouts/quick-action-dropdowns/helper}.tsx；T8 拆分后的复制命令 | 余下四处复制 hook 与 M4 例外清零 |
| T14 | E/stories/issue/collaboration-interactions.spec.ts（新）、E/playwright.m4-collaboration.config.ts（新） | C6/C7/C8/C9 三浏览器和会话/Back |
| T15 | 本阶段实际改动文件与 tools/keywords.json | 死成员、逐文件 lint、守卫、差异清单 |
| T16 | 本 spec/plan/review、交接子回执、docs/reviews/ | 完整门禁、整分支 deep-review、修复复审与交付 |

原设计 15 个粗任务细化为 16 个：命令拆分与主题/路由分为 T8/T9，其余依序对应；未减少完成线。已有 IssueUser 合并、source/archived actor 分支清理、原生草稿读写、无正 tabIndex 等以回归确认，不反向恢复旧实现制造失败。

### 2.2 身份、权限、确认与错误

- 页面地址用 route slug/project/issue；store 的同步 getter 不承担导航事实。登录代、工作区/项目/工作项、权限水位、宿主卸载/重开均复用现有 owner。发出前和应答后校验；晚答不更新新表单、菜单、导航或提示；不新增权限观察器。
- 项目有效成员可评论/回应可见工作项（含访客），但不把能评论等同于能修改工作项。OwnOnly 按项目角色判断，工作区管理员兼项目访客仍看不到他人工作项；对于其可见项，管理员豁免沿现有 `allowPermissions`，不能自行重新解释。
- 评论仅作者可改；作者或通过现有项目 ADMIN 授权者可删，管理员不能改他人评论。项目标识/作者名称由 CommentCard 的明确上下文传给菜单，服务端仍最终授权；归档只读，保留已有评论/回应展示但不发修改。
- 失败按原生 code/fields，经 useRefusalToast/useFormRefusal/useRefusedFocus；字段错误在焦点到达时已有 aria-invalid/描述。拒绝保留输入；已确认写后的补读失败只重试 GET，不重复 POST/PATCH/DELETE。所有 Promise 由现有命令或组件显式处理。
- en 与 zh-CN 同步。评论各行菜单/加回应以作者 display_name 为名；缺用户用停用用户 fallback，不读 email。新键盘处理复用 isComposing，IME Enter 不提交。

### 2.3 时间线与评论/表情

`IssueActivity.actor_id === null` 统一 SITE_NAME；非空但目录缺失是停用用户。不得通过邮箱、名字、archived 标志猜系统用户；M7 收集箱自己的来源尚未确定，不在本期删其领域取值。

时间线沿 P11：初次读完评论、活动全部 cursor 页，按 kind+id 合并，以保留纳秒的 (created_at,id) 排序。增量从**生成最后一页的请求 cursor**重读到末页；描述合并会移动旧活动 id 的 created_at，须替换重排而非只追加。远端编辑/删除落在增量窗口之外的评论，关闭重开全量读后体现；不新增实时同步承诺。

CommentCreate/Edit 发送 `IssueCommentCreate { comment_html: string }`，等待确认才清空/退出；空 HTML 不能发，清洗后空的服务端 422 必须可见；保留提及和既定 HTML 合同，不恢复 access/回复/JSON。重击和 Enter/按钮并发最多一次写；编辑器未 ready 不误提交。已存在的 disabled/read 权限分离，I20 用先成员创建、后降级的真实账号前置。

表情沿十进制码点串，组合字符不拆成 UTF-16 单元；工作项回应由独立集合，评论回应只来自评论内嵌 reactions。增加 409 reaction_exists 时不假加计数，删除只删本人该码点；确认后回读/拒绝/重复点击与权限失效受现有队列和 owner 管理。实际 picker 使用本站资源/原生表情，无 CDN/CSP 违规，不扩大 CSP。

### 2.4 描述历史和复制

沿原生 `DescriptionVersion/Page/Detail`，列表 newest-first、继续加载、选中后单独 GET。快速换版本/换工作项的旧详情不能覆盖新项；列表和详情错误分别可重试。

还原只走普通 `description_html` PATCH，复用详情描述的 restoreDescription 协调，等待确认才关闭；遵守项目成员/管理员、归档及 OwnOnly 的现有权限。600 秒合并和 noop 保留，不承诺一次还原必新增版本/活动。ModalTitle 命名、无正 tabIndex、Escape/焦点/后层 inert 必须真实浏览器核验。

Markdown 复制调用已有 `EditorRefApi.getMarkDown(): string` 后交 `useCopyText(text, successMessage): Promise<void>`；未 ready 禁用或不执行。公开 `copyMarkdownToClipboard(): void` 保持给其它消费者，不能 await void 后假成功。hook 内负责成功/拒绝，消费者不二次报成功。共享版本组件保留 M7 既有展示 props/原协议适配，不把 M7 页面验收提前关闭。

### 2.5 草稿与创建弹窗

原生 Draft 为 `{id, workspace_id, project_id: string|null, payload, created_at, updated_at}`，不是 Issue。payload 仅 name、description_html、state_id、priority、assignee_ids、label_ids、start_date、target_date、parent_id 九字段；全部可省略，无 null，保留暂不满足发布语义的内容。workspace ADMIN/MEMBER 才读写个人草稿；他人草稿 404、访客 403，不能靠不存在的 created_by_id 判断所有权。

- 无项目/空标题/空 payload 可存；分页 total 来自响应，不从当前页长度推断。无项目不隐藏整行/编辑器，不读工作项 filter/display preferences。
- PATCH payload 是整份替换；表单局部编辑通过 `draftInput(edits, held.payload)` 保留所有未编辑合法字段（包括暂不能显示的 priority/引用）。payload 字段清空即删除键，数组 []、合法空串按合同保留；顶层 project_id=null 才清空项目，省略保持。
- `updateIssue(ws,id,input): Promise<Draft>` 与 `moveIssue(ws,id,input): Promise<IssueDetail>` 沿既有 WorkspaceDraftIssues。发布先 PATCH 最新完整表单和项目，确认后 POST publish **无 body**；保存失败不得 POST。publish 403/409 draft_changed/422 保留已确认 Draft、输入、错误，不能假删除。
- 发布成功进入 root.confirmCreation 的 canonical/列表/计数确认，草稿消失；后续 GET 失败显示读错误并 GET-only retry。成功不补发第二次创建或项目序号读取。删除拒绝留弹窗/行；发布/删除双击一次写，命令晚答不影响新工作区。
- “存为草稿”及嵌套 Save this draft 复用 P14 单关闭决策，内层 Escape 只关内层并回焦；描述加载失败不保存缺失内容，重试后可继续。迁移只补块 id 不算用户修改；混合真实编辑保留。
- 三文件 header、empty-state、root 均按 route slug；root 已接 route prop 的能力保持，header/empty-state 的 allowPermissions 显式传 slug。Back/换账号清宿主状态。草稿的右键菜单关闭时 inert，键盘和指针都能打开编辑/删除/移到项目；UI 的“移到项目”执行上述 publish，不新增另一种移动 API。

### 2.6 命令与搜索

工作项 commands.ts 拆成复制、属性更新/指派、删除等小 hook/配置，聚合入口保留原返回 `TPowerKCommandConfig[]`；新文件随消费者一起落地。写入使用生成 `IssueUpdate` 白名单及现有详情命令，不能用 Partial<IssueDetail> 当请求。判权显式 route slug+项目，关闭/换项/同账号换代均清旧命令上下文，错误按 code；删仍走既有确认弹窗。编号/标题/链接复制统一 T13，T8 不制造临时例外。

任务顺序保持 T8/T9 先补动作、T10 再迁根页搜索。前两项页面验证从空根查询进入子页；中文输入只在 activePage 已选中的属性子页验证。根页非空查询会经过尚未迁移的搜索消费者，故中文/自由查询的端到端验收放 T10/T14；不为提前验证放宽旧接口守卫。

主题：沿 appearance/theme-switcher 的 sessionGuard，等 updateUserTheme 确认且仍同代才 setTheme/reload；失败保留原主题、按 code 提示，旧会话成功也不重载新页。其它本次改到的 preference 回调不得新添未处理拒绝。创建命令不用依赖 router store 的 workspaceProjectIds/canPerformAnyCreateAction 做范围推断；使用 route workspace 对应目录和现有授权，保留非 M4 创建命令原业务边界。

四文件加入 workspace-address-route：A/drafts/header.tsx、C/components/issues/workspace-draft/{empty-state,root}.tsx、C/components/power-k/config/creation/command.ts；规则后缀 `\.tsx?$`，补四个正例及其它领域/测试负例，不放宽 content。

工作项搜索复用 `useIssueSearch(root, slug, query, open)`，服务 `IssueSearchService.search(slug, IssueSearchQuery): Promise<IssueCollection>`，只读 `/api/v0/workspaces/{slug}/issue-search`。不翻成旧 WorkspaceSearch DTO，展示直接解引用 canonical Issue；本地结果只为 Issue/Project 的明确展示类型。

- 命令面板根页有非空 q 才开启工作项搜索；空输入继续原命令列表，不要求在命令首页列近期工作项。API 的空 q 近期语义由既有选择器保留。项目级搜索传 project_id；切工作区级必须省略 project_id。q 变化立即隐藏旧结果，由 useIssueSearch 负责单次 500ms debounce，禁叠加第二个 debounce。
- 匹配项目在已加载 M3 目录本地按 name/identifier 不区分大小写过滤，显式 workspace.id、排除 archived；用 knownProjectIds(workspace.id)+getProjectById 等既有显式范围读取，不借 currentLists/当前显示筛选。项目作用域仅显示当前项目；工作区作用域显示该工作区已加载可见项目。目录加载复用 useSessionSWR 的既有 ["PROJECTS", workspace.id, workspace.slug] 键和 ProjectStore.fetchProjects({id,slug})，仅面板打开时启用、不随 q 重取；只新增当前面板的 loading/error/GET 重试展示，不改 useWorkspaceFetch 或 ProjectStore 所有权。目录未就绪不伪称无项目，不加新接口/服务实例。
- 工作项保持服务端顺序/可见范围，不在 cmdk 二次模糊过滤或重排服务端返回的命中；结果点击只导航当次路由工作区的原生 browse 编号。未知/失效目录不拼 undefined 地址，显示受控读错误/重试。
- 404/403/500 读取错误不显示成无结果；当前查询可重试，权限或会话变化立即隐藏旧项。中文 q、编号、纯序号、字面 %/_/\、排序/limit、OwnOnly 在 I18 保留；q 超长和字段错误按服务端合同处理，不截断用户输入。
- 删除 WorkspaceService.searchWorkspace、其模块实例以及专用 IWorkspace*SearchResult、WORKSPACE_DEFAULT_SEARCH_RESULT、旧搜索参数/六组结果映射；WorkspaceService 的 create/update/delete/getAllViews 属 M7，保留。ProjectService 已 P14 删除，只复核零残留。M6/M7 搜索结果不伪造或接旧接口，按原设计暂不提供。

### 2.7 自动归档、复制与最终质量

I16 页面使用 clockTest/ClockWorld 独立数据库；推进业务时钟后取得新 nerve.baseURL 并重新登录（旧 cookie/access 不能跨 +744h 假有效），等新进程中在业务写后启动并 completed 的真实 auto_archive attempt，再从归档列表打开详情。只给 SQL 查询作断言，不 SQL 写前置或用 sleep 代替时钟；NULL actor 按 SITE_NAME，手动恢复真实用户显示。页面断言旧完成项已归档、非终态/新更新控制项未归档，保留原 API 水位/跨越 attempt 测试。

余下四个复制位置及历史 Markdown 共用现有 useCopyLink/useCopyText 两参数 Promise<void>；hook 处理成功/失败，外层不据 resolved Promise 设置“Copied”或额外成功（hook 在失败也 resolve）。创建 toast 移除自行推测成功的 copied 状态或只用 hook 反馈；评论链接含当前 origin 与 #comment-id。消费者真实拒绝/等待测试和 LAN HTTP 浏览器验证都有，不只测 hook。C8 同时复验 P14 已迁移的工作项链接/编号、子项及关联链接，不把 P14 的受控拒绝冒充本次 LAN HTTP 验收。

本期仅删除四个 M4 copy-hook 例外，M6–M8 例外保持各自归属；13 个历史 M4 模块级实例逐路径清点，允许其它领域仍有独立旧实例。改动 TS/TSX 逐文件 lint 清零，按 02 分类删除本期死参数/成员；P16 全局 no-shadow/守卫终态不提前宣称。

## 3. 准备期决定

| 决定 | 选择与理由 | 代价/验证 |
|---|---|---|
| D1 | 保持已批准 IssueRootStore 权限所有权；只消费原生读/写/owner | 每条新 UI 必须有晚答/ABA 证据，T3–T10/T14 |
| D2 | 将粗项 8 拆为工作项命令与主题/创建路由，共 16 Task | 每 Task 含测试约 1500 行内；超出先局部细化文件职责，不能堆一个大表单 |
| D3 | 已满足 actor/source、草稿数据层等核回归，不再实现 | 不造首次红；用有效变异证明关键断言 |
| D4 | 版本复制使用现有 getMarkDown+useCopyText，保留公共 ref | 不新增返回协议，测试真实 Markdown 与拒绝 |
| D5 | 根搜索页非空 q 才读，项目在显式 workspace 目录本地过滤 | 不承诺空搜索首页近期项或未加载领域全局搜索；T10 覆盖空词/范围切换 |
| D6 | 删除旧搜索专用成员，保留 M7 view service | 不把 M7 未接通页面归为 P15；T11 验实际所有读者 |
| D7 | 页面/组件矩阵在三引擎显式匹配，clock 页独立重登录 | 不沿默认 Firefox A4 grep，不把合成剪贴板等同系统权限 |
| D8 | 本次静态预检取代产品原型/实现重放 | 只写文档，不跑 P15 产品测试；授权实施后逐 Task 留原始证据 |

本表为既定设计内的局部落地决定，无新增服务端协议或跨模块责任改判。若实际实现需要这些变化，先提交已完成工作，再报负责人决定。

## 4. 验收标准

下列均是未来实施验收，本次预检不勾选。

- [ ] A1：T1 的 actor/null/fallback 与无猜测扫描，I12 全页/增量/重排/关闭重开页面通过。
- [ ] A2：T2 动态过滤/排序有名字、键盘与回焦正确、无嵌套按钮，至少留一个过滤项；三点名文件 lint 由各 owning Task 收口。
- [ ] A3：T3 作者编辑/作者及管理员删除、访客自己项评论、空/IME/重击/拒绝/晚答有组件和 I11/I20 页面证据。
- [ ] A4：T4 两类表情确认/仅删自己/重复拒绝/组合码点与归档只读、实际 picker CSP/本站资源通过。
- [ ] A5：T5 历史分页/切换/还原与 I13、ModalTitle/焦点、Markdown 等待/失败反馈通过，M7 展示合同不变。
- [ ] A6：T6 个人草稿列表/分页/无项目/字段保留/删除/发布及 I14 通过；PATCH 失败无 POST、POST 无 body、成功同步列表计数。
- [ ] A7：T7 创建弹窗两种存草稿路径、描述未 ready/拒绝保留、嵌套 Escape/Back/纯迁移由真实编辑器和 C9 验证。
- [ ] A8：T8 工作项命令职责拆分，IssueUpdate、权限/归档/确认/晚答/删除宿主正确。
- [ ] A9：T9 主题仅确认同代后生效，失败不改主题；四 route 文件与 .ts/.tsx 守卫正反例通过。
- [ ] A10：T10 命令原生搜索与项目本地过滤、服务端排序、中文/字面/编号/范围/错误/晚答及 I18 页面通过。
- [ ] A11：T11 旧搜索方法/类型/常量无真实读者；历史 13 模块实例逐路径为零，M7 视图服务完整。
- [ ] A12：T12 I16 页面真实新时钟进程/新登录/新 attempt、自动归档和站点名/手动用户显示通过。
- [ ] A13：T13 四处 hook 与 T5 Markdown 的消费者等待/拒绝均正确，四 M4 copy-hook 例外删除，C8 LAN HTTP 有实际证据。
- [ ] A14：T14 C6/C7/C8/C9 草稿及本期 I 页面在 Chromium、Firefox、WebKit 各有实际非零结果；页面无未说明失败请求、CSP/console/unhandled 错误。
- [ ] A15：T15 改动文件 dead/oxlint 清零、字段/旧 URL/路由/复制守卫通过，差异与边界文档准确。
- [ ] A16：T16 完整七项门禁、整分支 deep-review/局部修复复审、准确最终分支 HEAD CI、全要求证据闭环；阶段完成后先不合并。

## 5. 后续范围

P16 负责全局死成员/no-shadow/守卫终态及 C12 准备；M4 真正收尾才逐条完成设计 §14、总体设计 9.4、全部输入 handoff closed 和 M5–M8 的 M4-closeout。M5 文件/图片，M6/M7 自身协议/领域页/收集箱来源/版本旧 DTO、M7 lazy pending discard 仍各自负责；P10a 裁定 A 不变。项目/工作区搜索之外的领域搜索不预造新 API。

## 6. 风险与验证

关键风险是能评论与能改项混淆、管理员不能删别人评论、整替换 payload 丢字段、搜索 query/权限变更保留旧结果、主题/复制先报成功以及自动归档故事认错进程或 attempt。每项分别落在 T3/T6/T10/T9/T13/T12 的生产消费者测试、持有请求测试及浏览器证据；复用 FakeNerve、issueStoreTab、原 API/assert/clock fixture，不复制服务世界。C6 portal/真实编辑器和 C7 cmdk 二次过滤不能靠 jsdom 单独证明。

## 7. 交接输入与关闭落点

| 输入 | 本期落点 | 仍保留 |
|---|---|---|
| M1-P3-trim-platform 来源显示 | T1 确认工作项动态 source 分支已删除 | M7 来源取值及 M4 最终转交 |
| M1-P5-brand | T1/T12 NULL actor 站点名、无邮箱/名字猜测 | 系统代建收集箱归 M7 |
| M2-closeout §8/9/11/12 | T4/T9/T13/T11 表情 CSP、主题、复制、13实例 | 其它节和最终交接关闭 |
| M3-closeout §5/7/8 | T3/T5/T6/T8–T11/T13–T15 服务/控件/route/复制/lint | 其它里程碑自己的页面/共享类型最后读者 |
| P4/P6/P8/P11 | T1/T3–T7/T10/T12 原生协议、时钟与 owner/队列 | 不重写已实现后端或数据层 |
| P14 review §14 | T5 Markdown、T13 四复制、T7 草稿完整流 | M7 pending discard；公共 EditorRefApi 不破坏 |

所有输入顶部 status 保持 open。本期实施完成只追加对应子项证据，不提前建立未来最终 handoff 或关闭 M4。

## 附录 A：准备与实施证据界线

准备目录（主项目根）：`.superpowers/sdd/P15-preparation-20261031/`。记录实际源码/测试/设计阅读范围、路径/合同/要求映射、builder-check 与 deep-review 文档预检；未来执行证据放 P15 worktree 的 `.superpowers/sdd/P15-web-collaboration-drafts-search/`。P14 main CI 仅是本期基线，不能当作上述 A1–A16 的 P15 验收。准确准备提交和后置收据由预检完成时补记。
