# M7 / P6a：网页的项目视图——列表、视图页、另存与更新视图、命令面板的视图

> 状态：草稿，已照主控对起草者 O-1–O-6 的裁定（§8）与预检 [re-96](../../../reviews/re-96-M7P6a预检.md) §6 的处理修正；已实现（T1–T12，T13 写文档），实施中的改动就地标“实施中改（T…）”，裁定的补记在 §3 末的“实施中的补记”，经过见[阶段记录](../reviews/P6a-web-project-views-review.md)；整分支审查、修复轮与限定复审在 T14，修复轮的改动就地标“实施中改（整审修复轮）”；合并前的终检与合并由 P6b 的 T1 补记。基线 main `47bd860b`（P5b 的 noff，`47bd860ba104145a37a66687c12d6db7b49658b4`）。设计见 [M7 设计](../M7-design.md) §2（H12 `:39`、H20 `:47`、H21 `:48`）、§3.7（`:118-120`）、§3.11（`:159`）、§7（`:240`、`:242-243`、`:246`、`:248`）、§12（P6a 一行 `:308`）、§14；网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)（请求体轮到时才算 `:638`）。调研与裁定在台账 `.superpowers/sdd/P6a-web-project-views/`：`research/scope.md`、`research/questions.md`（Q1–Q24）、`rulings.md`（C-1–C-27，本 spec 的 R1–R27 逐条对应）。合同 `api/modules/navigation.yaml`（下称 `NAV`）P3b 已合入，本段不改。

缩写：`C` = `web/apps/web/core`；`P` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]`；`V` = `P/views`；`VC` = `C/components/views`；`PKG` = `web/packages`；`E` = `e2e`；`K` = `tools/keywords.json`；`HO` = `docs/v0/M7-collaboration/handoffs`；`D` = M7 设计；`PW` = 只读的 `plane/apps/web`。

## 1. 目标与边界

P6 在设计 §12 分两段：P6a 项目视图、P6b 工作区视图（`D:308`）。P6a 交付：

- **数据层**（§2.1）：视图的 service 与 store 照 v0 §7.7 重写（`useSessionSWR`、`ReconciledByKey`、`oneAtATime`、`followInSession`），类型只来自 `@nerve/api-client` 的 `View`、`ViewCreate`、`ViewUpdate`；视图只在用到的地方读，不进项目的外壳（R20）。
- **入口与控件**（§2.2）：`viewEntryOn` 一条规则给侧栏、页签栏、两条导航命令；每个控件给谁照一张表，每格对到服务端的一条规则（R4、R14）。
- **列表页**（§2.3）：Plane 的列表去掉访问、锁、收藏与发布；页面状态按项目；行的菜单、右键菜单与复制照 M6；弹窗在页面的宿主（R5、R17）。
- **视图的弹窗**（§2.4）：只管视图的身份——名字、说明、图标；筛选、布局与显示都在视图页改、以“更新视图”存；关闭政策同 M6 的迭代、模块弹窗；`ModalTitle`、`FieldError`、自然 Tab（R11、R12、R28）。
- **列表引擎的 `PROJECT_VIEW`**（§2.5）：新枚举值，作为“项目列表页的一种”接进各分派点，穷尽的分派让漏一处是编译错（R1，**L!**）。
- **视图页**（§2.6）：以 `viewId` 为 key 的布局、四种状态、内存里的草稿（R2、R6、R10）。
- **另存与更新视图**（§2.7）：项目页、迭代页、模块页、视图页给“另存”，视图页给“更新视图”；“更新视图”等 nerve 的回答，在途时视图页改草稿的控件停用，失败留着按钮（R7、R8、R9、R29，**L!**）。
- **命令面板**（§2.8）：视图菜单与列表页同一个读、搜索的 “Views” 组（只在项目范围）、`create_view` 照入口规则（R13，**L!**）。
- **删除**（§2.9）：Plane 的旧视图层（service、store、hook、项目视图独占的类型、常量与帮手）、`getTabIndex` 全套、成员 store 的 `projectMemberIds`、收藏里对旧 store 的读写、弹窗不再用的 `LayoutDropDown`；加 `old-view-urls`（R15、R16、R19、R24、R28）。
- **e2e**（§2.11）：H12 (page) 两份、H20 (page) 一份；S2 去掉项目视图一格、`mounts.ts` 加视图的读（R22、R23）。
- **P5b 的收尾回填**（T1，R26）。

**P6a 不做：**
- 工作区视图（`workspace-views/`、`globalView` store、`workspace-level.tsx` 的 HOC、`workspace/views/*`）：P6b。只有 `useViewMenuItems` 的签名变了，P6b 的 `C/components/workspace/views/quick-action.tsx:52` 编译跟随一行（§2.3）。与工作区视图共用的类型（`EViewAccess`、`IWorkspaceView`）留给 P6b（R19）。
- 收藏的写与侧栏的收藏（P7）；视图的收藏星（R5）。
- 服务端、合同、CI 配置：不改；CI 余量只报告（R22）。
- 视图存着已删除的标签、成员、迭代、模块时的清理（R18 只照项目页显示“不可用”）。
- 富筛选的 `invalid.tsx` 的英文（`C/components/rich-filters/filter-item/invalid.tsx`）：交 M8 的候选（R21）。

**P6a 之前 main 的样子**：项目视图的页面仍读 Plane 的旧地址（`C/services/view.service.ts:12`、`:20`、`:28`、`:36`），列表永远是加载中或 404；视图页只有 `PageHead` 与英文的“View does not exist”（`V/(detail)/[viewId]/page.tsx:32`、`:36-45`、`:50`）；视图页的路由判为 `DEFAULT`（`C/hooks/use-issue-layout-store.ts:27`），不列工作项；项目页、迭代页、模块页的“Save view”被 P4 收起（P4 spec R17，`specs/P4-web-notifications.md:379-381`）；命令面板的视图菜单永远转圈（`C/components/power-k/ui/pages/open-entity/project-views-menu.tsx:25-31`）；`getTabIndex` 只剩视图表单一个读者（`VC/form.tsx:26`、`:75`）；`projectMemberIds` 只剩视图的两个读者（`V/(list)/mobile-header.tsx:22-24`、`VC/view-list-header.tsx:31-33`）。

**P6a 之后 main 的样子**：项目的管理员、成员、访客从侧栏、页签栏或命令面板进视图列表（开关开着），按名字搜索、按所有者与创建日期筛选、按名字或时间排序；管理员与成员新建视图、所有者改自己的、所有者与管理员删；视图页按视图存着的筛选与显示列出项目的工作项，改筛选、显示与布局只改这个页面里的草稿，所有者“更新视图”之后才存（在途时这些控件停用），失败时按钮留着；视图的弹窗只改名字、说明与图标；四个列表页都能“另存为新视图”，迭代页、模块页另存的视图带这个迭代、模块的条件；命令面板列出项目的视图、在项目范围搜得到视图；旧的视图层、`getTabIndex`、`projectMemberIds` 全部删除。

## 2. 交付合同

### 2.1 数据层（R3、R20）

- **service**：新 `C/services/views/views.service.ts` 的 `ViewsService`（每一代以自己的客户端构造，先例 `C/services/planning/modules.service.ts`）：`listProjectViews(projectId)`、`createProjectView(projectId, body: ViewCreate)`、`getView(id)`、`updateView(id, body: ViewUpdate)`、`deleteView(id)`（`NAV:19`、`:44`、`:138`、`:163`、`:199`），回答生成的类型、不转换。
- **store**：新 `C/store/views/view.store.ts` 的 `ViewStore`，`RootStore.views`（构造在 `issue` 之前，`C/store/root.store.ts:96` 今天的 `projectView` 一行旁；旧 store 并存到 T10，§2.9）：
  - `lists`：`ReconciledByKey<View[]>`，按项目 id，nerve 的次序（`created_at` 倒序、同值按 `id`，`D:119`）；`details`：`ReconciledByKey<View | null>`，按视图 id，删了之后是 `null`（先例 `C/store/planning/module.store.ts:48-58`）。
  - 读：`fetchProjectViews(projectId)`、`fetchView(id)`；取值：`getProjectViews(projectId)`、`getViewById(id)`（详情没有时取列表里的那一行）。读不排队；只经 `RootStore` 找兄弟（项目经 `projectRoot.project.getProjectById`，项目不在了不显示它的视图，同模块 store）。
- **写**：新 `C/store/views/view-changes.ts`，每个视图一个 `oneAtATime`（`C/lib/one-at-a-time.ts:14`），新建一个项目一个；请求体在轮到时算（“更新视图”的体在轮到时读草稿，§2.7）。回答经 `Reconciled` 写进详情与项目的列表：建 → `prepended`、改 → `replaced`、删 → `dropped` 且详情为 `null`（`C/lib/reconciled.ts:201`、`:206`、`:218`）。canonical 只在 nerve 确认时变，页面不先在本地改（旧 store 的乐观改 `C/store/project-view.store.ts:196-211` 不保留）。nerve 对没有改动的 `PATCH` 答 200 不写（`D:118`），store 照样写回答。`isUpdating(viewId)`：可观察，这个视图有改在队列里或在途时为真（视图页的在途停用读它，R29；随第一个读者在 T8 建）。
- **hook**：`C/hooks/store/use-views.ts` 的 `useViews()`；页面跟进（提示、跳转、关弹窗）一律经 `followInSession`（`C/lib/in-session.ts:40`）。
- **读的位置**（R20）：键 `["PROJECT_VIEWS", projectId]`，门是 `viewEntryOn`（§2.2）；读者：列表页、视图页页头的切换器（打开时）、命令面板的视图菜单与搜索（打开时）。视图页自己的读是草稿的播种（§2.6）。项目的外壳（`C/layouts/auth-layout/use-project-fetch.ts:60-72`）不加读。`.oxlintrc.json` 的 swr 名单（`:121-215`）收入每个新的取数文件。

### 2.2 入口规则与控件矩阵（R4、R14）

- **`viewEntryOn(project, role)`**：新 `C/lib/views-entry.ts`，照 `intakeEntryOn`（`C/lib/intake-entry.ts:18-24`）：`project !== undefined && project.issue_views_view && role ∈ {ADMIN, MEMBER, GUEST}`，不看归档（已归档项目的页面由外壳挡住，`use-project-fetch.ts:83-92` 的 `archived`）。同文件 `viewsCreatableBy(project, role)` = `viewEntryOn` 且 `role ∈ {ADMIN, MEMBER}`。角色一律 `getProjectRoleByWorkspaceSlugAndProjectId(slug, projectId)`（路由参数；工作区管理员即项目管理员，`C/store/user/permissions.store.ts:68-75`）。
- **读者**：侧栏（`C/components/workspace/sidebar/project-navigation.tsx:103-111`）、页签栏（`C/components/navigation/use-navigation-items.ts:73-82`）、“打开视图”“转到视图”两条命令（`C/components/power-k/config/navigation/project-commands.ts:180-181`、`:192-193`）读 `viewEntryOn`；`create_view`（`C/components/power-k/config/creation/command.ts:110-111`）、列表页的 “Add view” 与空态的“新建”、四个页面的“另存”读 `viewsCreatableBy`。命令的单测表两行（`commands.test.tsx:120`、`:138`，工作区、管理员、访客、关着、已归档 `[F, T, T, F, T]`）不变，另加角色的格：成员 T、访客 T、没有角色 F；`create_view` 访客 F。关闭 M6-M7-07 的视图一半。
- **控件矩阵**（“—”＝不给；每个控件都不给会被 nerve 拒的人）：

| 控件 | 管理员 | 成员 | 访客 | 服务端的规则 |
|---|---|---|---|---|
| 侧栏、页签栏、“打开视图”“转到视图” | 给 | 给 | 给 | `view.read` 三种角色（`server/internal/modules/access/domain/rules.go:223`）；开关只挡写（`D:159`），入口照设计不给 |
| 列表 | 全部 | 全部 | 全部；只看自己的访客只有自己的 | `view.read`；`View.HiddenFrom`（`server/internal/modules/navigation/domain/view.go:52-54`） |
| “Add view”、空态的“新建”、`create_view`、“Save view”（另存） | 给 | 给 | — | `view.create`（`rules.go:228`）；开关关着 409 `navigation.views_disabled` |
| 行与页头的“编辑” | 自己的 | 自己的 | — | `View.UpdatableBy` 只给所有者（`view.go:58`），且角色要管理员或成员（`rules.go:229`） |
| “Update view” | 自己的，有改动时 | 自己的，有改动时 | — | 同上 |
| 行与页头的“删除” | 全部 | 自己的 | — | `View.DeletableBy`：所有者或项目管理员（`view.go:65-73`），角色要管理员或成员（`rules.go:230`） |
| 复制链接、新标签页打开 | 给 | 给 | 给 | `view.read` |
| 视图页的筛选、显示、布局（只改草稿；“更新视图”在途时停用，R29） | 给 | 给 | 给 | 只读；草稿在页面里（`D:120`） |
| 视图页的 “Add work item”、快速添加、行内编辑 | 给 | 给 | — | M4 的工作项规则，同项目页 |
| 开关关着 | 入口都不给；列表页、视图页是停用页、不读 | 同左 | 同左 | 写 409，读照常（`D:159`） |

  被降为访客的所有者不给“编辑”“更新视图”（服务端要管理员或成员）；所有者离开项目之后视图照常列出，没有人能改，管理员能删（P3b R2 之后的通则）。今天的三处偏差改正：空态的“新建”给访客（`VC/views-list.tsx:43-48` 含 `GUEST`）、“Add view” 不看角色（`V/(list)/header.tsx:54-56`）、`canUpdateView` 看 `is_locked`（`C/components/work-item-filters/filters-hoc/project-level.tsx:62`、`:77-91`）。

### 2.3 列表页（R5、R16、R17、R18 的列表一半）

- **布局**：`V/(list)/layout.tsx` 以项目 id 为 key（`HO/M3-closeout.md:35`），提供页面状态（新 `VC/page-state.tsx`，先例 `C/components/inbox/page-state.tsx`）：搜索（名字，不分大小写的子串）、所有者筛选（`owned_by_id`，选项是 `getProjectMemberIds(projectId, true)`）、创建日期筛选（`C/components/common/filters/created-at.tsx`）、排序（名字、创建、更新，升降，本地排）；离开页面、换项目即丢。宿主 `ViewActionsHost`（新 `VC/actions-host.tsx`，先例 `C/components/modules/actions-host.tsx`）在 key 之内，持有新建、编辑、删除三个弹窗与页面的寿命（`pageMounted`）。T4 的宿主只持有删除；新建与编辑的弹窗到 T5 改成 `View` 的形状，行的“编辑”、“Add view” 与空态的“新建”随之在 T5 接上（预检 re-96 I5 ①，T4 的矩阵只测给不给）。
- **页面**：开关关着 → 停用页（`disabled_project.empty_state.view.*`，今天 `V/(list)/page.tsx:75-91` 的那一份，不读）；读在途 → 加载；读失败 → 失败与重试（`PlanningReadFailure` 的形状）；空 → 空态（“新建”只给 `viewsCreatableBy`）；有行 → 本地搜索、筛选、排序之后的行，筛掉了所有行时给“没有匹配”。
- **页头**：面包屑（项目、`t("views")`）、搜索、筛选（`FiltersDropdown`）、排序、“Add view”（`t("project_views.add_view")`，只给 `viewsCreatableBy`）；窄屏页头（`V/(list)/mobile-header.tsx`）同一份状态。英文字面量都经 `t()`（`HO/M3-closeout.md:78`）：“Views”、“Search”、“Filters”、“Add view”；页面标题的 “- Views”（`V/(list)/page.tsx:46`）；筛选面板里的——所有者一组 `C/components/common/filters/created-by.tsx` 的 “Created by”（`:62`，`common.created_by`）、“You”（`:88`，`you`）、“View less / View all”（`:98`，`common.view_less`、`common.view_all`）、“No matches found”（`:103`，`common.search.no_matches_found`），日期一组的 “Created date”“Custom”（`created-at.tsx`，`inbox_issue.filters.created_at`、`common.filter_options.custom`）与 “No matches found”（`created-at.tsx:76`），面板搜索框的占位 “Search”（`VC/filters/filter-selection.tsx:65`，`common.search.label`）。三份都只有视图的读者。只有图标的按钮与只靠占位的搜索框也经 `t()` 得名（实施中改（整审修复轮），整审 O5；都是已有的键）：页头与面板的搜索框 `common.search.label`、两处的清除 `common.search.clear`（`VC/view-list-header.tsx:97`、`:106`，`VC/filters/filter-selection.tsx:59`、`:69`），已应用筛选整组与每个值的移除 `common.remove_item`（`VC/applied-filters/root.tsx:61`；`C/components/common/applied-filters/members.tsx:52`、`date.tsx:53`，名为共用、只有视图列表页读，同上一并改），图标 `aria-hidden`。`created-by.tsx:39-50` 的选项只随搜索词重算（依赖只有 `searchQuery`，`:49` 关了规则），成员晚到时面板是空的：依赖补上成员、已选与当前用户，删去那条关规则的注释。
- **行**：名字（链到视图页）、说明、所有者头像、快捷操作。一行只挂一份快捷操作：删 `VC/view-list-item.tsx:53-57` 的 `quickActionElement`（窄屏由 `ListItem` 显示，同模块行 `C/components/modules/module-list-item-action.tsx:76`）；删 `VC/view-list-item-action.tsx:79-86` 没人打开的那对弹窗、`:88-90` 的公开私有、`:96-106` 的收藏星。
- **快捷操作**（`VC/quick-actions.tsx`）：右键菜单一行一个（关着时 `inert`，`PKG/ui/src/dropdowns/context-menu/root.tsx:99`）；菜单按钮是 `CustomMenu` 的按钮本身，以 `t("common.item_options", { name })` 为名（“{name} options”，同 `C/components/modules/quick-actions.tsx:116-118`），不嵌 `IconButton`（`HO/M3-closeout.md:50`、`:52`）；项：编辑、删除（照矩阵）、复制链接、新标签页打开（`noopener,noreferrer`）。复制经 `useCopyLink`（`C/hooks/use-copy-link.ts:42`），成功与失败各有提示；删 `K` 里 `copy-hook` 的视图例外（`K:3350-3356`）。编辑与删除只调宿主的 `openDialog`。
- **`useViewMenuItems`**（`C/components/common/quick-actions-helper.tsx:130-145`，M6）：参数去掉 `view`、`workspaceSlug`、`projectId`（`:47-57`；`view` 今天只用于 `if (!view) return []`），只收 `isOwner`（所有者且角色是管理员或成员）、`isAdmin` 与四个处理函数；P6b 的 `workspace/views/quick-action.tsx:52` 编译跟随（只删实参）。
- **删除**：`AlertModalCore`（标题经 `ModalTitle`，按钮文字显式传 `t()`），在途不关；nerve 确认之后关、提示，行随 `Reconciled` 消失，页面转到项目的视图列表（今天的行为，`VC/follow-ups.test.tsx:133`；在列表页上即留在原处）；视图页在跳转之前什么都不画，别处删掉它时说“不存在”（实施中改（整审修复轮），§2.6“四种状态”的末句）。被拒提示，弹窗留着。跟进只在宿主还挂着、同一会话时（`pageMounted` 与 `followInSession`）。
- **成员**：两个读者改为 `getProjectMemberIds(projectId, true)`（R16，§2.9）。

### 2.4 视图的弹窗（R11、R12、R28、R30）

- **只管视图的身份**（R28）：名字、说明、图标（`logo_props`）。筛选、布局与显示都不进弹窗，在视图页改、以“更新视图”存（同 R12 的筛选）：编辑弹窗改布局而视图页另有未存的草稿时，同一个视图就有两个真相来源。删 `VC/form.tsx:197-246` 的布局（`LayoutDropDown`）与显示（`FiltersDropdown` + `DisplayFiltersSelection`）两个控件与 `:41-47` 里的 `access`、显示的初值；`LayoutDropDown`（`C/components/dropdowns/layout.tsx`，唯一的产品读者是 `VC/form.tsx:32`、`:204`）与它的测试 `C/components/dropdowns/layout.test.tsx` 随之删（§2.9）。M4/M6 的显示换算不动、不导出。
- **形状**：`VC/modal.tsx`、`VC/form.tsx` 照 M6 的迭代、模块弹窗（`C/components/cycles/modal.tsx:126`、`C/components/modules/modal.tsx:144`、`C/components/cycles/form.tsx`）：
  - 在途不给 `ModalCore` 的 `handleClose`、Cancel 停用；不在途时 Escape、点背景、Cancel 直接关；删文档级 Escape（`VC/modal.tsx:80-82`）。没有“确认放弃”（表单只有名字、说明、图标，丢了不贵）。
  - 标题是 `ModalTitle`（今天普通 `h3`，`VC/form.tsx:96`），文字 `view.create.label`、`view.update.label`，另存时同新建。
  - 名字：react-hook-form 的 `required` 与 `maxLength 255`（`:144-150`）改为同 M6 的规则：去空白之后为空或超过 255 个字符被拒，提示是 `FieldError`（今天 `:170` 的 `span`）、`aria-invalid`，被拒取得焦点（`useRefusedFocus`，`C/lib/refused-focus.ts`）；nerve 的 `validation_failed` 点名 `name` 放在字段下（`useFormRefusal`，`C/lib/form-refusal.ts`）。
  - 自然 Tab 次序（删 `getTabIndex`，§2.9）：图标、名字、说明、Cancel、提交（以 DOM 为准，今天图标在名字之前，`VC/form.tsx:99`、`:141-143`）；`autoFocus` 改为 `data-autofocus`（同 M6）。说明是普通的 `TextArea`（`VC/form.tsx:179-192`；合同是纯文本），没有编辑器的 Tab 问题；它以 `t("description")` 得名，不只靠占位（实施中改（整审修复轮），整审 O5，`VC/form.tsx:163`）。
  - IME（`HO/M4-closeout.md:71`）：组字时的 Escape 由 `ModalCore` 阻止（`PKG/ui/src/modals/modal-core.tsx:99`），不关弹窗，T5 一格（先例 P5b 的 `create-root.close.test.tsx` 与变异 `t2-mut-ime-escape-closes-r2`）；Enter 不另写：表单没有自己的按键处理，提交只经 `<form>` 的 `onSubmit`（`VC/form.tsx:94`），同 M6 的迭代、模块弹窗。
  - 只在 nerve 确认之后关，被拒留着表单与输入；新建（含另存）成功：关、提示、转到新视图的页面（今天的行为，`VC/follow-ups.test.tsx:88`）；编辑成功：关、提示。嵌套 Escape：图标选择器（`EmojiPicker`）开着时 Escape 只关它。迟到的成功只在弹窗还挂着时提示与关（`mounted` 的 ref，同 M6）。
  - 提示经 `t()`：建 `project_views.action.create.success`、改 `project_views.action.update.success`，标题 `toast.success`；被拒经 `useRefusalToast`（`navigation.views_disabled`、`navigation.view_not_found` 已在 `C/lib/error-messages.ts:105-106`）。
- **字段与体**：编辑只发改了的字段（`changes(initial, values)`，同 M6；只会是 `name`、`description`、`logo_props`）；`ViewUpdate` 的每个字段整体替换（`NAV:163`）。不放筛选（R12；Plane 的 `PW/core/components/views/form.tsx:264-289`）。
- **“Add view” 的体**（R28）：`name`、`description`、`logo_props`，加上合同默认值的常量给齐的 `display_filters` 与 `display_properties`（`NAV:635-662`：列表、不分组、最新在前、显示子工作项与空组、日历按月不含周末；14 个属性全开；与 `E/fixtures/views.ts:52-80` 的 `viewDefaults` 相同），常量写在弹窗旁（线上词，不经换算）；不带 `filters`（新建的视图没有筛选）。H12 (page) 以请求体钉住它等于 `viewDefaults`。`create_view` 打开的是同一个“新建”。
- **三种打开**：新建（列表页的 “Add view”、空态、`create_view`，宿主分别是 `ViewActionsHost` 与 `C/components/modals/project-level.tsx:59-64`）；编辑（宿主）；另存（HOC，§2.7）。三种的表单都只有名字、说明、图标；另存的体另带页面的筛选与已确认的显示、属性（§2.7）。名字的初值（R30）：新建与另存都为空（名字必填，用户自己写；不编造 “{视图名} 2” 一类的默认，也不用 Plane 的英文 “Untitled”）；编辑是视图的名字。

### 2.5 列表引擎的 `PROJECT_VIEW`（R1，L!；R10）

- **枚举**：`EIssuesStoreType` 加 `PROJECT_VIEW`（`PKG/types/src/issues/issue.ts:14-22`）。
- **store**：新 `C/store/issue/project-view/`：
  - `issue.store.ts` 的 `ProjectViewIssues extends PagePreferencesIssues`（先例 `C/store/issue/project/issue.store.ts:9-38`）：`fetchIssues(slug, projectID, viewID, loader, options)` → `fetchPage(slug, projectID, viewID, options)`（`C/store/issue/helpers/page-preferences-issues.store.ts:29-43`，项目的列表，查询来自视图页的偏好即草稿）；`isPreferenceQueryCurrent(projectID, viewID)`（项目页的按项目 id 找当前列表，`C/store/issue/project/issue.store.ts:11-23`；视图页的要项目与视图两个量：当前列表是这个项目的、且上次读用的是这个视图的草稿）、`fetchNextIssues`、`fetchIssuesWithExistingPagination` 同项目页的形状。
  - `filter.store.ts` 的 `ProjectViewIssuesFilter extends PagePreferencesFilter<ProjectIssueFilters>`，服务是草稿（§2.6），转换器 `toProjectFilters`，`storeType = PROJECT_VIEW`；只多 `discard(viewID)`（丢草稿、作废在途的播种、丢 `filters[viewID]`，§2.6）与 `adopt(viewID, view)`（实施中改，T8：转给草稿服务的 `adopt`，第一个读者是 HOC 的“更新视图”；草稿服务是这个类私有的字段，根上不另留名字，`C/store/issue/project-view/filter.store.ts:33-34`，主控裁定）。
  - 在 `C/store/issue/root.store.ts:202-212` 旁构造（`ViewDraftService` 收根上的 `views.fetchView`）。
- **分派点**（每处一行或扩联合；漏了多数是静默的，每处在 T6 有一格一变异）：
  - `C/hooks/store/use-issues.ts`：`TStoreIssues` 加一行；`switch`（`:66-103`）加 `case`。
  - `C/hooks/use-issues-actions.tsx`：一个导入、一行无条件的调用（`useProjectViewIssueActions()`，同 `:68-72` 的五行）、两个 `case`（`PROJECT_VIEW`，与今天落进 `default` 抛错的 `WORKSPACE_DRAFT`，显式列出、照旧抛错）与 `default` 的穷尽（`:74-90`）。
  - `C/hooks/use-issue-layout-store.ts:27`：`viewId` → `PROJECT_VIEW`，`globalViewId` 仍 `DEFAULT`（P6b）。
  - `C/hooks/store/use-page-issue-filters.ts:9-14` 的联合与 `:21-38` 的 `switch`（没有 `default`，漏了是 `undefined`：看板与列表的本地筛选、分组的收起读不到）。
  - `C/components/issues/use-page-display.ts:19`（`PageDisplayStoreType`，页头的布局与显示）；`PKG/constants/src/issue/filter.ts:240-244`（`ISSUE_STORE_TO_FILTERS_MAP` 指向项目页的 `issues`；`use-page-display.ts:48` 以可选链读，漏了是“显示”里没有选项）。
  - `PKG/constants/src/issue/common.ts:55-59`（`TCreateModalStoreTypes`）、`C/components/issues/issue-layouts/command-host.tsx:30-35`（`DIALOG_STORES`；分组头的“+”在点击时才查，`C/components/issues/issue-layouts/list/headers/group-by-card.tsx:95-96`）。
  - 四个布局根的联合（`list/base-list-root.tsx:34-39`、`kanban/base-kanban-root.tsx:37-41`、`calendar/base-calendar-root.tsx:35`、`spreadsheet/base-spreadsheet-root.tsx:29`，都在 `as` 之后，不扩也编译）、`C/hooks/use-group-dragndrop.ts:21-27`（拖放）、`empty-states/index.tsx:20-33`（`default` 是 `null`）。
  - **列表挂载的键**（`C/hooks/use-issue-list-mount.ts:27-29`、`:48-62`，M6）：`page` 一段对 `PROJECT_VIEW` 取路由的 `viewId`（今天只给迭代、模块页）。否则同一项目里布局选项相同的两个视图共用一个键（筛选不在键里，`fetchPage` 自己从偏好读，`C/store/issue/helpers/page-preferences-issues.store.ts:15-18`、`:36-42`）：A 的列表读在途时换到 B，B 的挂载并进 A 的那次读，B 的筛选行下面列着 A 的结果。
  - **日历某一天的“添加已有”**：同项目页（只给那一天）。入口把页面的 store 类型交给命令宿主（`C/components/issues/issue-layouts/calendar/quick-add-issue-actions.tsx:43`、`:57-66`），请求的类型（`C/components/issues/issue-layouts/add-existing-dialog.tsx:25`，今天只认项目页与两种计划页）收 `PROJECT_VIEW`，宿主以这个类型的 `addIssuesToView` 发（`command-host.tsx:139`）。
  - **行的快捷操作**：`ProjectIssueQuickActions`（`C/components/issues/issue-layouts/quick-action-dropdowns/project-issue.tsx:169-171`，今天把类型写死为 `PROJECT`）读 `IssuesStoreContext`，只在 `PROJECT_VIEW` 时传 `PROJECT_VIEW`，否则照旧 `PROJECT`（个人页两个根的上下文是 `PROFILE`，仍是 `PROJECT`）；`:35` 的联合收 `PROJECT_VIEW`。六个使用处（四个 `project-root.tsx`、两个 `profile-issues-root.tsx`）不改。
- **穷尽**：`use-issues.ts`、`use-issues-actions.tsx` 的 `switch` 对 `EIssuesStoreType` 穷尽，`default` 里 `satisfies never`（或 `satisfies Record<EIssuesStoreType, …>` 的表），漏一个值是编译错；运行时的未知值仍抛 “Unsupported issue store”。不进 M6 的两张表（`C/hooks/planning-page-domains.ts:43-70`、`C/components/issues/issue-layouts/planning-page-layouts.ts`），也不加两行的 `PROJECT_LIST_PAGES` 表。
- **命令**：新 `C/hooks/use-project-view-issue-actions.ts`，`IssueActions` 的成员写全（它们多是可选的，`C/hooks/use-issues-actions.tsx:37-57`，少给一个不报错）：`fetchIssues`、`fetchNextIssues`、`createIssue`、`createIssueAsGiven`、`updateIssue`、`removeIssue`、`addIssuesToView`、`archiveIssue`、`dropIssue`、`updateFilters`。建、改、删、归档、拖放、“添加已有”同项目页，写进视图页自己的列表（项目页的命令是文件内写死 `PROJECT` 的私有常量，`:93-97`，不复用）；取数、下一页与 `updateFilters` 是视图页的（`updateFilters` 不看调用方给的 id、用路由的 `viewId`：表格与日历传的是项目 id，`C/components/issues/issue-layouts/spreadsheet/base-spreadsheet-root.tsx:95`、`calendar/base-calendar-root.tsx:171`；先例 `C/hooks/use-planning-page-issue-actions.ts:148-156`）；跟进经新 `C/hooks/use-project-view-preferences-follow.ts`（`usePagePreferencesFollow` 的 `capture`、`listed`、`relist`，`C/hooks/use-page-preferences-follow.ts:21`）。
- **筛选不推进基线**（R10，L!）：视图页的富筛选回调只调 `ProjectViewIssuesFilter.updateFilterExpression`（写草稿）再经跟进重取列表，不用 `usePageFilterExpression`（`C/hooks/use-page-filter-expression.ts:38-58` 会把 `initialFilterExpression` 推进为确认的值，“更新视图”就永远不出现）；它的联合（`:20`）不加 `PROJECT_VIEW`。
- **空态**：`ProjectEmptyState`（`empty-states/project-issues.tsx:19`）以 prop 收筛选实例的类型与 id（今天写死 `PROJECT` 与 `projectId`，`:28`），`index.tsx` 加 `PROJECT_VIEW` 的 `case`；有筛选 → 搜索的空态与“清除”，没有 → 新建工作项给管理员与成员（新建弹窗以 `PROJECT_VIEW` 为 store 类型）。不另写一份。
- **两份钉 `DEFAULT` 的测试**：`C/hooks/use-issue-layout-store.test.tsx:75-114` 的 `viewId` 一格期望 `PROJECT_VIEW`（“在选定的项目建、不取列表”不变），`:125-131` 与 `C/hooks/store/use-issues.test.ts:26-31` 的拒绝表去掉 `"PROJECT_VIEW"`（`"GLOBAL"`、`"unknown"` 照旧被拒）。

### 2.6 视图页（R2、R6、R18 的视图页一半）

- **布局**：`V/(detail)/layout.tsx` 以 `viewId` 为 key（`<Fragment key={viewId}>`，先例 `P/cycles/(detail)/layout.tsx:26-27`；`HO/M3-closeout.md:38`），`ViewActionsHost` 在 key 之内；后退、前进换视图时整棵重挂。
- **门**：`viewEntryOn`；开关关着 → 同列表页的停用页（不读；实施中改，T7 跟进：两页渲染同一处 `VC/views-disabled.tsx` 的 `ViewsDisabled`，照 `PlanningDisabled`）。
- **读与草稿**（R2）：新 `C/store/issue/project-view/view-draft.service.ts` 的 `ViewDraftService implements PagePreferencesService<ProjectIssueFilters>`（端口 `page-preferences-filter.store.ts:33-36`），线上词：
  - 草稿按视图带一个代（“页面还在不在”，草稿服务自己的；基类 `PagePreferencesFilter` 不碰，它的读序号与 `values` 是私有的，`page-preferences-filter.store.ts:127-130`、`:175-190`）。
  - `get(viewId)`：有草稿给草稿；没有就记下代、`views.fetchView(viewId)`（一个 `GET /api/v0/views/{id}`，回答进 `ViewStore.details`），回答到时代没变才以它的 `filters`、`display_filters`、`display_properties` 播种；代变了（期间 `discard` 过）就以草稿服务自己的错误拒绝：迟到的回答只进 `ViewStore.details`，不建草稿，基类也不写 `filters[viewId]`（它只在 `get` 答到值时写）。
  - `update(viewId, body)`：只替换给出的字段，返回整份草稿；不发请求。
  - `adopt(viewId, view)`：“更新视图”成功之后草稿换成 nerve 的回答（§2.7），只在草稿还在时；不在就什么都不做。
  - `discard(viewId)`：丢草稿，代加一（作废在途的播种）。
  - 页面的读是“每次挂载读自己的”（P5a 整审 I1、P5b R27 的写法，`C/lib/use-session-swr.ts:19-25`；先例 `C/components/inbox/content/root.tsx:108-125`）：放在自己的文件（新 `C/components/views/use-view-page-read.ts`，收入 swr 名单），`useSessionSWR(entry ? ["PROJECT_VIEW_PAGE", workspaceSlug, viewId] : null, (slug, id) => issuesFilter.fetchFilters(slug, id).catch(noDraftIsNoFailure), { revalidateOnMount: false, revalidateOnReconnect: false })`（`.catch` 是实施中改（整审修复轮），见下面“四种状态”的末句），挂载时 `mutate()` 一次（不带数据的 `mutate()` 先删在途标记、必发请求），页面的四种状态只看这一次挂载自己的读答没答（同先例的 `answered`）。理由：普通的挂载读在一次读答到之后的去重窗口（默认 2000 毫秒）里再挂上不验证（应用的 `revalidateOnMount: true`、没有 `dedupingInterval`，`PKG/constants/src/swr.ts:7-13`），而 `discard` 已丢了 `filters[viewId]`，“离开再在两秒内回来”会一直停在加载。进页面恰一个 `GET /api/v0/views/{id}`，然后是项目列表的读（查询是草稿的）；重连不读。
- **草稿的寿命**：视图页卸下（离开、换视图、换项目）时 `ProjectViewIssuesFilter.discard(viewId)`：丢草稿与 `filters[viewId]`（否则再进来时在读答到之前显示旧草稿，富筛选的基线也会是旧草稿）；再进来以新读到的视图播种。`discard` 之后基类的 `values` 仍留着旧草稿（子类删不到）：再进来、读答到之前 `confirmed(viewId)` 给的是旧值；这时页面在加载，没有读者。换会话时 `IssueRootStore` 连同草稿重建（`RootStore` 跨会话只留 `instance`、`router`、`theme`，`C/store/root.store.test.ts:57`、`:101-115`），不另调。看板的收起照 `PagePreferencesFilter` 的本地存储按 `PROJECT_VIEW` 存，不另写。
- **四种状态**：读在途 → 加载；`problemCode(error) === "navigation.view_not_found"` → “视图不存在”（新 `project_views.not_found.{title,description,action}`，动作回到列表页）；别的失败 → 失败与重试（`PlanningReadFailure` 的形状）；开关关着 → 停用页。视图被别人删掉之后，再进来时读答 404 → 不存在；页面挂着时没有下一次读，这时的“更新视图”得 404，提示 `navigation.view_not_found`。只看自己的访客打开别人的视图同样 404（`view.go:52-54`）。别的项目的视图（地址里换了项目）同“不存在”：nerve 的 `GET /api/v0/views/{id}` 不看项目，页面与页头各比一次回答的 `project_id`，不把它的草稿套在这个项目的列表上（实施中改，T7：`V/(detail)/[viewId]/page.tsx:86`、`VC/use-view-page-read.ts:99`，同迭代页）。读的组件 `ViewVisit` 以会话为 key（`page.tsx:54-55`，先例 `IssueRouteVisit`；实施中改，T7 跟进：换会话时 `StoreProvider` 只重渲染，不分开就多一个 `GET` 并先闪一下“不存在”）。草稿服务的拒绝 `NoViewDraftError` 导出（`C/store/issue/project-view/view-draft.service.ts:18`；实施中改，T7）。实施中改（整审修复轮）：读的 hook 的取数函数接住 `NoViewDraftError`、照常兑现（`VC/use-view-page-read.ts:41`、`:73-75` 的 `noDraftIsNoFailure`，别的失败照旧抛），所以它从不是共用键上的错，hook 只有四态（`reading`、`not-found`、`failed`、`read`，`:22-26`；停用由页面在读之前判）。理由：读的键由同一视图、同一会话的各次挂载共用，swr 的失败一支不核对是不是当前的请求，T7 起初的“挂着的页面拿到它即页面已不在，什么都不画、不提示”会让前一次挂载迟到的拒绝把此刻挂着的页面变成空白（整审 m1）。删除之后：发出删除的是这一页自己的宿主（删的是这一页显示的视图）时，页面在跳转之前什么都不画（宿主上的 `ViewsPage.pageLeaving`，`VC/actions-host.tsx:29`；弹窗在发出之前置真、被拒时清掉，`VC/delete-view-modal.tsx:58-59`、`:72`；`page.tsx:98`）；别的挂载（删除不经这一页，或删除在途时后退、前进进来的新挂载），以及再进来读答 404 的，一律说“视图不存在”并给回列表（整审 O3 与限定复审 n1，经过见阶段记录 §3 的 T14）。
- **页头的四种状态**：今天没有视图时什么都不画（`V/(detail)/[viewId]/header.tsx:45`）；改为加载、不存在、失败、停用时只画面包屑（项目、`t("views")` 链到列表），视图在、开关开着时画全。
- **窄屏页头**：视图页的布局今天只有 `header`（`V/(detail)/layout.tsx:17-25`）；加新 `V/(detail)/[viewId]/mobile-header.tsx`（`ProjectViewMobileHeader`，自己是 observer 的包装，先例 `P/cycles/(detail)/layout.tsx:25`、`:31-38` 的 `PlanningPageMobileHeader`：布局不是 observer），视图在、开关开着时画 `MobileHeaderFilters`（`PROJECT_VIEW`、`viewId`，传 `isUpdating(viewId)`），收入 `project-address-route`。
- **页头**（`V/(detail)/[viewId]/header.tsx`）：面包屑（项目、`t("views")` 链到列表、切换器 `BreadcrumbNavigationSearchDropdown`，打开时以 `["PROJECT_VIEWS", projectId]` 读，`:75`；实施中改，T7：这个 `@nerve/ui` 组件不告诉调用方它打开了，加一个可选的 `onOpen`，在已有的打开回调里调，`PKG/ui/src/breadcrumbs/navigation-search-dropdown.tsx:25`、`:67-69`，页头以“打开过”为读的门，`V/(detail)/[viewId]/header.tsx:100`）、布局与显示（`C/components/issues/filters.tsx` 的 `HeaderFilters`、窄屏的 `C/components/issues/mobile-header-filters.tsx`，`PageDisplayStoreType` 收 `PROJECT_VIEW`；“更新视图”在途时停用，R29、§2.7）、“Add work item”（矩阵）、快捷操作（同行的菜单，删除之后转到列表页）。删 `:93-96` 的私有锁、`:41`、`:47` 的 `projectViewIds`。
- **根**：新 `C/components/issues/issue-layouts/roots/project-view-layout-root.tsx` 的 `ProjectViewLayoutRoot`（先例 `project-layout-root.tsx`、`planning-page-layout-root.tsx:70-128`）：`IssuesStoreContext` = `PROJECT_VIEW`；`ProjectLevelWorkItemFiltersHOC`（`entityType` = `PROJECT_VIEW`、`entityId` = `viewId`，`updateFilters` = R10 的回调；`enableSaveView`、`enableUpdateView` 在 T8 随 HOC 的新读者一起传，T7 不传）；`WorkItemFiltersRow`；四种布局；`IssuePeekOverview` 的 `hostScope` = `["project-view", slug, projectId, viewId]`。
- **筛选引用已删除的东西**（R18）：照项目页，显示“不可用”（`common.filter_options.unavailable`，`C/components/rich-filters/filter-value-input/select/selected-options-display.tsx:26-31`），原样发查询、原样存回，“更新视图”不因此出现。

### 2.7 另存与更新视图（R7、R8、R9、R21、R29，L!）

- **另存在哪些页面给**（R7）：项目页（`C/components/issues/issue-layouts/roots/project-layout-root.tsx`）、迭代页与模块页（`planning-page-layout-root.tsx`）、视图页都传 `enableSaveView`；给谁照 `viewsCreatableBy`；按钮在富筛选有条件时出现（shared-state 的 `canSaveView`，`PKG/shared-state/src/store/rich-filters/filter.ts:271-273`）。P4 的 `roots/save-view-collapsed.test.tsx:99-107`（“offers no Save view”）翻过来，文件改名 `save-view-offered.test.tsx`。
- **另存存什么**：`filters` = 页面的富筛选当前表达式经 `toProjectFilters`，迭代页另加 `cycle_id: [该迭代]`、模块页 `module_id: [该模块]`（M6 的表 `PLANNING_PAGE_DOMAINS` 加一列 `viewTerms`，共用的根不分支；迭代页、模块页的筛选类型本来就排除自己的词，`C/lib/issue-filters.ts:198-209`）；`display_filters`、`display_properties` = 页面已确认的线上值（R8）。弹窗只给名字、说明、图标（§2.4），在 HOC 里（`project-level.tsx:189-198`）。（实施中改，T5 的接缝：HOC 也画视图弹窗，T5 给弹窗最终的接口时 HOC 只改另存的预填——状态 `ViewContent | null`，删 `getDefaultViewDetailPayload`，`handleViewSave` 只给 `{ filters }`，212 → 199 行；显示取 `confirmed` 与迭代、模块的词在 T8。T5 到 T8 之间另存在产品里打不开，分支内的窗口，主控取 (A)。）
- **已确认的线上值**（R8，L!）：`PagePreferencesFilter` 加只读的 `confirmed(id): PagePreferences<F> | undefined`（私有 `values` `:127` 的拷贝），`IPagePreferencesFilter` 随之加；四个页面一条路（视图页的就是草稿）。它读的是不可观察的私有 `Map`，不是响应式的：读者跟着同时变的、可观察的 `issuesFilter.filters[id]` 重算（HOC 的 `hasAdditionalChanges` 的依赖挂在 `filters[id]` 的显示与属性上，不只挂 `confirmed`）。
- **“更新视图”**（R9，L!）：
  - 给谁：HOC 的 `isDisabled` = 不是（`enableUpdateView`、所有者 `owned_by_id === 当前用户`、角色管理员或成员、`viewEntryOn`）；不看 `is_locked`。出现照 shared-state：`canUpdateView = updateViewOptions && (hasChanges || hasAdditionalChanges) && !isDisabled`（`filter.ts:279-285`，不改）。
  - `hasAdditionalChanges` 比线上值：`confirmed(viewId)` 的 `display_filters`、`display_properties` 与视图存着的（今天拿控件词比线上词，`project-level.tsx:94-102`）。
  - `onViewUpdate(expression)` 返回 Promise：经视图的队列发 `PATCH /api/v0/views/{id}`，体在轮到时算 = `{ filters: toProjectFilters(expression), display_filters, display_properties }`，后两项读那一刻的草稿；成功 → 提示（`project_views.action.update.success`）、`adopt` 回答再 `fetchFilters`（草稿与显示换成回答）、富筛选的当前表达式不等于回答的筛选时 `resetExpression(toFilterExpression(回答的筛选))`（等价时不动，保住焦点；同 `use-page-filter-expression.ts:47-52` 的做法），然后 resolve——shared-state 的 `updateView` 随之把基线设为当前表达式（`filter.ts:499-506`），即回答；失败 → 按码提示（`navigation.views_disabled`、`navigation.view_not_found` 与别的拒绝经 `useRefusalToast`），然后 reject，草稿与基线都不动，按钮留着。
  - `followInSession` 从不 reject：HOC 在 `failed` 里记下错误、在它之后自己 throw；换了会话两个跟进都不调、Promise resolve（页面已不在，基线的重置无害）。
  - **跟进只在 HOC 还挂着时**（同弹窗与宿主的 `mounted`，先例 `C/components/modules/actions-host.tsx:54-58`）：离开页面之后答到的成功不提示、不 `adopt`、不 `fetchFilters`（草稿已随 `discard` 丢了，`adopt` 本也什么都不做，§2.6），回答只经 `Reconciled` 进 `ViewStore`；失败也不提示。
  - **成功之后的焦点**：按钮随 `canUpdateView` 变假卸下，焦点不能落到 `body`。点它时焦点在它上面的，落到筛选行的“添加筛选”按钮（`AddFilterButton`，筛选行自己经 ref 给，在 `handleUpdate` 落定之后一个微任务里）；筛选行不在了落到页头的筛选开关。失败时按钮留着，焦点不动。（实施中改，T8：落点等筛选行恢复可用——`disabled` 为假——才落，排在后面的同一视图的改答完之前不落，`t8-mut-land-while-busy`；页头筛选开关的落点经 `rich-filters/filters-toggle.tsx` 的一个标记。）
  - **在途时停用**（R29，L!）：“更新视图”在途时，视图页改草稿的控件停用（busy，同 M6、P5b 在途等待的规则），不让在途期间的改动被回答覆盖、静默丢失：
    - 在途的来源只有一个：视图 store 的 `isUpdating(viewId)`（可观察；这个视图的改从入队到落定为真，§2.1）。视图页头与 `ProjectViewLayoutRoot` 读它，往下传。
    - 筛选行（`C/components/rich-filters/filters-row.tsx`，经 `C/components/work-item-filters/filters-row.tsx:19-20` 原样传入）收一个可选的 `disabled`（默认不停用），`ProjectViewLayoutRoot` 传 `isUpdating(viewId)`：停用时条件（`FilterItem` 的 `isDisabled`，`:60`）、添加筛选（`AddFilterButton` 的 `isDisabled`，`:73`；实施中改，T8：它为真时 `AddFilterButton` 不渲染，`C/components/rich-filters/add-filters/button.tsx:63`，所以在途时添加筛选**隐去**、不是停用，不碰这个共用件，主控裁定接受）、“Clear all”（`:82`）与 “Save view”（`:87`）都停用，“Update view” 照旧以它自己的在途状态（`:33`、`:48-55`）loading、文字 “Confirming”。不复用 `disabledAllOperations`：它把右侧连 “Confirming” 一起藏起来（`:79`）。这是 M4/M6 共用的筛选行的改动，别的页面不传、行为不变。
    - 页头：`HeaderFilters`（`C/components/issues/filters.tsx:52`、`:55`、`:57`、`:58`）与 `MobileHeaderFilters` 收一个可选的 `disabled`，传给布局（`LayoutSelection` 的按钮 `C/components/issues/issue-layouts/filters/header/layout-selection.tsx:40-50`、`MobileLayoutSelection`，两者今天没有 `disabled`，各加一个可选的）、显示（`FiltersDropdown` 已有 `disabled`，`C/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx:22`）与筛选开关里的添加筛选（同上，隐去）；视图页头传 `isUpdating(viewId)`，项目页、迭代页、模块页不传（默认不停用）。
    - 成功之后草稿与基线是回答，控件恢复；失败之后控件恢复，草稿与基线不动。
    - 列表里另三条改草稿的路（补充裁定 O-6 与预检 re-96 I4）同样停用：日历布局自己的选项下拉（月 / 周、显示周末，`C/components/issues/issue-layouts/calendar/dropdowns/options-dropdown.tsx:53-69`，经 `calendar/base-calendar-root.tsx:169-173` 写 `display_filters.calendar`，合并在 `page-preferences-filter.store.ts:253-258`）、表格列头的排序菜单（`C/components/issues/issue-layouts/spreadsheet/columns/header-column.tsx:42` 的 `CustomMenu`，经 `base-spreadsheet-root.tsx:93-99` 写 `DISPLAY_FILTERS`；`CustomMenu` 已有 `disabled`，`web/packages/ui/src/dropdowns/custom-menu.tsx:49`）与有筛选时空态的“清除筛选”（`empty-states/project-issues.tsx:51-53`，动作已有 `disabled`）。三处在共用的列表根之下好几层（`BaseSpreadsheetRoot` → `SpreadsheetView` → `SpreadsheetTable` → `SpreadsheetHeader` → `SpreadsheetHeaderColumn` → `HeaderColumn`），不一层层传 prop：新 `C/components/issues/issue-layouts/draft-busy.ts` 的 `DraftBusyContext`（默认 `false`），`ProjectViewLayoutRoot` 以 `isUpdating(viewId)` 提供，日历的选项下拉、`HeaderColumn` 与 `ProjectEmptyState` 读它；项目页、迭代页、模块页不提供、行为不变。
    - “任何入口”以写者的 `git grep` 为准：显示的写者三处——页头（`C/components/issues/use-page-display.ts:46-52`）、表格列头（`base-spreadsheet-root.tsx:95`）、日历的选项（`base-calendar-root.tsx:171`）；筛选表达式的写者是富筛选实例，入口是筛选行、页头筛选开关里的添加筛选与空态的清除。以上在途时都停用，日历一处由单测承担（页面故事不另加）。
    - `isUpdating(viewId)` 在同一个视图的编辑弹窗的 `PATCH` 与删除在途时也为真（同一个队列），视图页的控件这时也停用；弹窗开着时本来就点不到它们，无害。
  - 筛选行（`C/components/rich-filters/filters-row.tsx:48-55`）的 `handleUpdate` 加 `catch`（提示已给，吞掉）。
- **文案**（R21，L!）：`filters-row.tsx` 的 “Filters”（`:45`）、“Clear all”（`:83`）、“Save view”（`:88`）、“Confirming”“Update view”（`:99`）与 HOC 的三条英文提示（`project-level.tsx:139`、`:152`、`:160`，后两条随 R9 重写）都经 `t()`，en 的文字不变（页面故事按名字找的不受影响；`fake-i18n` 下按字面找的三处改键：`C/components/issues/issue-layouts/roots/cycle-layout-root.test.tsx:91`、`module-layout-root.test.tsx:92` 的 “Clear all” 与 `save-view-offered.test.tsx` 的 “Clear all”“Save view”，`C/lib/fake-i18n.ts:10-12` 的文字即键），en 与 zh-CN 同一提交。`invalid.tsx` 交 M8。
- HOC 不超过 220 行；“另存”“更新”的逻辑若放不下进新 `C/components/work-item-filters/filters-hoc/use-project-view-saving.ts`。（实施中：HOC 73 行，逻辑在 `use-project-view-saving.ts`，170 行；HOC 不再收 `TEnableSaveViewProps`、`TEnableUpdateViewProps` 的 `label` 与 `isDisabled` 覆盖——项目的调用方没有传的，标签由筛选行以 `t()` 的默认说出；工作区的 `workspace-level.tsx` 仍读，`shared.ts` 不改。）`project-level.tsx:41-42` 的弹窗状态与 `IProjectView` 的预载换成 `ViewCreate` 的形状。

### 2.8 命令面板（R13，L!）

- **菜单**：`C/components/power-k/ui/pages/open-entity/project-views-menu.tsx` 以 `useSessionSWR(viewEntryOn ? ["PROJECT_VIEWS", projectId] : null, …)` 读（与列表页同一个键），在途给加载、被拒给 `PowerKMenuEmptyState` 的拒绝文字、空给 `power_k.menus.no_views`（实施中改，T9：门在面板开着时被关——开关关着、没有角色——照 M6/P7c R2 立刻说为什么、不读、不列 store 里还留着的，开关关着用 `disabled_project.empty_state.view.title`，没有角色用 `errors.project_not_found`，`project-views-menu.tsx:60-61`）；`C/components/power-k/menus/views.tsx:11` 与 `C/components/power-k/config/navigation/project-commands.ts:18`、`:171`（`data as IProjectView`）的 `IProjectView` 换成 `View`（同一提交）；删 `fetchedMap` 的永远转圈（`:25-31`）。菜单开着就读（每次挂上都再验证，这是对的）；它与列表页同时挂着时共用一次在途的读（同一个键）。
- **搜索**：只在项目范围（路由的项目，或 `/browse/:workItem` 所在工作项的项目），加 “Views” 组（标题读已有的键 `views`，`PKG/i18n/src/locales/en/common.json:231`；没有 `common.views`）；新 `C/components/power-k/ui/modal/use-view-search.ts`（不推广 `use-planning-search.ts`：入口规则不同，视图给访客），读的时机：面板开着、在项目范围、store 里还没有这个项目的视图列表时，以列表页的键读一次（先例 `C/components/power-k/ui/modal/use-planning-search.ts:45-46` 的“store 里没有才读”）；有了就用 store 的（本会话的写经 `Reconciled` 已在里面）。代价（R20 的）：没去过列表页时，每开一次面板一个 `GET /api/v0/projects/{id}/views`，不搜索也发。在读到的列表里按名字不分大小写的子串本地过滤，组内照 nerve 的次序；读被拒或在途时这一组不显示、不挡别的组；“没有结果”在所有组都空时才给；行是项目标识加名字，选中关面板并转到视图页（`ui/modal/search-menu.tsx:43-48`、`:97-102`，`search-results.tsx:72-96` 的形状）。工作区范围不给（R13 (b) 要新接口，负责人的事）。
- **`create_view`**：`viewsCreatableBy`（§2.2）；打开 `ProjectLevelModals` 的新建弹窗（`C/components/modals/project-level.tsx:59-64`，卸下时关，`:36-43`）。

### 2.9 删除与读者（R15、R16、R19、R24）

每一项在删之前 `git grep` 核读者，阶段记录写下结果；删掉最后一个读者的 Task 同一提交删去它。

| 删除 | 最后的读者（基线） | 何时 |
|---|---|---|
| `PKG/utils/src/tab-indices.ts`（`getTabIndex` `:11`）、`PKG/constants/src/tab-indices.ts`（`PROJECT_VIEW_TAB_INDICES` `:7`、`ETabIndices` `:9-11`、`TAB_INDEX_MAP` `:13-15`）、桶导出 `PKG/utils/src/index.ts:23`、`PKG/constants/src/index.ts:29` | `VC/form.tsx:26`、`:75` | T5（表单走自然次序的同一提交）。共用控件非正的 `tabIndex?`（`C/components/dropdowns/types.d.ts:21`、`date-range.tsx:50`、`PKG/ui/src/dropdowns/helper.tsx:25`）保留，阶段记录写明核过的读者（`HO/M3-closeout.md:107`） |
| `LayoutDropDown`（`C/components/dropdowns/layout.tsx`，49 行）与它的测试 `layout.test.tsx`（110 行，M4/P16 `a3af031b8` 写的键盘与译文两格） | `VC/form.tsx:32`、`:204` | T5（弹窗不再有布局，R28；键盘的性质由 H20 (page) 在视图页的布局切换上接上） |
| 成员 store 的 `projectMemberIds`：接口 `C/store/member/project/project-member.store.ts:40`、`computed` `:85`、getter `:106-109`；专属断言 `project-member.store.test.ts:102`（改为只比 `getFilteredProjectMemberIds`）、`:113`（删）；收集箱测试的替身字段 `C/components/inbox/filters/menu.test.tsx:64` | `V/(list)/mobile-header.tsx:22-24`、`:56`；`VC/view-list-header.tsx:31-33`、`:121`（全仓另有同名的局部变量 `C/components/issues/issue-detail-widgets/sub-issues/title-actions.tsx:45`、`:94`，不是读者；`K` 的 `project-address-route` 有三处写着这个名字：`:2336` 的说明、`:2342` 的正则、`:2348` 的 hit 样例） | T4（两个读者改为 `getProjectMemberIds(projectId, true)` 之后重新核全仓，`HO/M3-closeout.md:121`）。删去 getter 之后 `K:2336` 的说明照同一句里 `currentProjectDetails` 的写法改（“P6a 删去、模式里的这一式只防它回来”），正则与样例留着 |
| 旧 service `C/services/view.service.ts`、旧 store `C/store/project-view.store.ts`、hook `C/hooks/store/use-project-view.ts`、`RootStore.projectView`（`C/store/root.store.ts:58`、`:96`） | 视图的页面与组件、HOC、命令面板、收藏 | T10 |
| `PKG/types/src/views.ts` 的 `IProjectView`（`:20-40`）、`TViewFilterProps`（`:46-51`）、`TViewFilters`（`:53-58`）；`TViewFiltersSortKey`、`TViewFiltersSortBy`（`:42`、`:44`）若新的排序不再读 | 旧 store、旧组件、HOC；命令面板的 `C/components/power-k/menus/views.tsx:11` 与 `C/components/power-k/config/navigation/project-commands.ts:18`、`:171`（T9 换成 `View`） | T10；`EViewAccess`（`:15-18`）留给 P6b（工作区视图的读者）。实施中：两个排序类型仍被 T4 的新排序读（`VC/view-list-order.ts`、`VC/filters/order-by.tsx`），照条件留下 |
| `PKG/utils/src/project-views.ts`（`orderViews` `:21`、`shouldFilterView` `:50`、`getViewName` `:80`、`getValidatedViewFilters` `:91`）与桶导出 `PKG/utils/src/index.ts:19` | 旧 store | T10（新列表页的排序与筛选是自己的纯函数） |
| `PKG/constants/src/views.ts` 的 `VIEW_ACCESS_SPECIFIERS`（`:9-15`）；`VIEW_SORTING_KEY_OPTIONS`、`VIEW_SORT_BY_OPTIONS`（`:17-26`）若新的排序不再读；桶导出 `PKG/constants/src/index.ts:32`（文件空了时） | `web/apps/web/helpers/views.helper.ts`；`VC/filters/order-by.tsx` | `VIEW_ACCESS_SPECIFIERS`：T4（实施中改：它最后的读者 `views.helper.ts` 在 T4 删，同一提交删去）；两个排序常量仍被新排序读，照条件留下，文件与桶导出不改 |
| `web/apps/web/helpers/views.helper.ts`（整份） | `VC/applied-filters/access.tsx`（随列表页删） | T4（实施中改：最后的读者 `access.tsx` 随列表页在 T4 删，knip 报它未用；照“删掉最后一个读者的 Task 同一提交删去”） |
| 列表页的旧筛选树里只属于访问、收藏的部分：`VC/applied-filters/access.tsx`、`VC/filters/filter-selection.tsx:80-89` 的收藏 | 旧列表页 | T4 |
| 收藏里对旧 store 的读写：`C/store/favorite.store.ts` 的 `viewStore`（`:55`、`:78`）、`removeFavoriteEntityFromStore`（`:257-266`，只有视图一支，删了视图一支就是空的）与它的 `action` 登记（`:73`）和四个调用（`:292`、`:297`、`:347`、`:355`）；`C/hooks/use-favorite-item-details.tsx:16`、`:26` 改读 `useViews().getViewById`（有就用新名字与图标，没有就用收藏自己的） | — | T10（P7 再接收藏的写） |
| `K` 的 `copy-hook` 视图例外（`:3350-3356`） | `VC/quick-actions.tsx` 的 `copyUrlToClipboard(` | T4 |

实施中补（T4、T6 跟进、T10 跟进）：表外随最后的读者删去的——`@nerve/ui` 的 `FavoriteStar`（`PKG/ui/src/favorite-star.tsx` 与桶导出，最后的读者是视图行的收藏星，T4）；`ListItem.quickActionElement`（`C/components/core/list/list-item.tsx`，T4 去掉了它唯一的写者，T4 跟进）；`replaceUnderscoreIfSnakeCase`（`PKG/utils/src/string.ts`，已应用筛选的名字改经 `t()` 之后没有读者，T4 跟进）；`ISSUE_LAYOUT_MAP` 的 `export`（`PKG/constants/src/issue/layout.ts`，`LayoutDropDown` 删后只在本文件用，T6 跟进）；收藏 store 随旧 store 失去最后读者的 `removeFavoriteEntity`、`removeFavoriteFromStore` 与接口上的 `entityMap`（T10 跟进；类的 `entityMap` 字段留着，`addFavorite` 的去重读它，`C/store/favorite.store.ts:116`）。

- **新规则 `old-view-urls`**（R24，`phase: "M7/P6"`）：只匹配项目视图的旧地址 `/api/workspaces/…/projects/…/views…`（照 `old-module-urls`、`old-intake-urls` 的写法，`K:3078`、`:3185`），web 的源文件（测试也算）不得出现；hit 与 miss 的样例写全（`/api/v0/projects/{project_id}/views`、`/api/v0/views/{view_id}`、页面链接 `/${workspaceSlug}/projects/${projectId}/views`、组件目录的导入都不碰；工作区视图的旧地址在 `workspace.service` 里，P6b 扩）。T10 加。
- **`project-address-route`**（`K:2334`）：P6a 新建或改写的、读项目或角色的文件收入规则的文件名单并带 hit 样例（`VC/actions-host.tsx`、`VC/page-state.tsx`、`V/(list)/header.tsx`、`mobile-header.tsx`、`[viewId]/page.tsx`、`VC/quick-actions.tsx`、`VC/form.tsx` 若读项目、`project-view-layout-root.tsx`、`use-view-search.ts` 等）；文件名单里删了的文件（如 `VC/view-list-item-action.tsx` 若删、`VC/delete-view-modal.tsx` 若并进宿主）连同 hit 样例（`K:2631-2637`）一起改；miss 样例里变成读项目的（`K:2797-2803`）移出。每次改 `K` 跑 `node tools/keywords.mjs`。

### 2.10 守卫与告警

- **oxlint**：web 的上限 18（`web/apps/web/package.json:10`）是等式（实施中改，T4：18 → 14，去掉的四条是改写的 `VC/applied-filters/root.tsx` 旧的三个筛选键数组的 `unicorn(prefer-set-has)` 与删去的 `views.helper.ts` 的 `oxc(no-map-spread)`，`package.json:10` 现为 14）；新文件零告警；删掉的写法若带告警，同一提交照实改上限并留计数的收据。e2e 的上限 0。
- **knip 与第一个读者**：新文件、新导出在第一个读者出现的 Task 里建立（plan 列出）；包的桶文件导出与只被测试读的产品导出 knip 查不出，死成员的整份比对兜底。
- **swr 名单**（`.oxlintrc.json:121-215`）：每个新的取数文件收入（`V/(list)/page.tsx` 或它的状态组件、视图页的读 `VC/use-view-page-read.ts`（`[viewId]/page.tsx` 不再自己取数）、`[viewId]/header.tsx` 的切换器、`project-views-menu.tsx`、`use-view-search.ts`）；删掉的文件从名单删。
- **文案**：新键只有 §8 O-3 的清单（en、zh-CN 同一提交；写完整的字面量，`tools/i18n-keys.mjs` 才认得；不许一个键是另一个键的前缀）；别的一律用已有的键。
- **死成员**：T1 的基线、T4、T6、T8、T10 之后整份比对，T13 的终表；P5b 终表里视图的行（`views.helper.ts`、`constants/src/views.ts`、`workspace-views.ts`）逐行写去处。

### 2.11 e2e（R22、R23）

- **配置**：新文件都在 `E/stories/collaboration/`，名字合 `E/playwright.m7-pages.config.ts` 的 `testMatch`（`-page.spec.ts` 或 `-pages.spec.ts`）与 `grep`（每个 `test()` 的标题带 `H12 (page)` 或 `H20 (page)`），三引擎。CI 以默认配置在 chromium 跑一次。
- **检查点与终检跑六个页面配置**（R22）：m7-pages、m6-pages、m6-acceptance、m4-list、m4-filters（I5 列表偏好、I15、I19 与筛选的交互，正是 `PagePreferencesFilter` 与筛选行）、m4-detail；去掉 P5b 的 m5-content（P6a 不动文件、编辑器）。检查点 0 那一次（引擎改动之前）是 m4 三个配置的基线。
- **夹具**：新 `E/fixtures/view-pages.ts`（约 200 行：列表页与视图页的定位、行、弹窗、扣住视图的回答、视图页的读表）；前置条件只经接口（`E/fixtures/views.ts` 的 `createProjectView` `:17`、`viewDefaults` `:52`、`viewRow` `:158`、`savedQuery` `:184`）；`E/fixtures/mounts.ts` 加 `PROJECT_VIEWS`（`GET /api/v0/projects/{project}/views`），头注释 `:1-15` 里“views … are not read”（`:11-13`）改写为“视图只在用到的页面读”。帮手复用 `holdAnswer`、`expectPopupBesideButton`、`moveWithinApp`（`E/fixtures/settings-pages.ts:74`、`:240`、`:296`），`dismissToasts`、`failedLoads`、`tabStops`、`recordApi`（`E/fixtures/cycle-pages.ts:275`、`:297`、`:315`、`:342`），`refuseClipboardWrites`、`expectQuietConsole`（`E/fixtures/browser.ts:238`、`:254`），`modalDialog`（`E/fixtures/dialogs.ts:113`）。
- **故事**：

| 文件 | 标题 | 覆盖 |
|---|---|---|
| `h12-project-views-list-page.spec.ts` | `H12 (page)` | 管理员、成员、访客的列表页照 §2.2 的矩阵（“Add view”、空态的“新建”、行菜单的编辑与删除按所有者与管理员，访客只有复制与新标签页）；只看自己的访客只见自己的；搜索、所有者与创建日期筛选、排序、“没有匹配”；筛选的按钮（`FiltersDropdown`）以 “Filters” 为名、`aria-expanded`、Escape 之后焦点回到它（`HO/M3-closeout.md:56`）；“Add view” 新建：弹窗以 “Create View” 为名，名字空白与超长（`FieldError`、`aria-invalid`、焦点在名字上、不发请求），弹窗里只有图标、名字、说明（没有布局与显示），请求体精确（`display_filters`、`display_properties` 等于 `viewDefaults`、没有 `filters`，§2.4），经接口读回视图（`viewCalls(api).read`，`E/fixtures/views.ts:119-125`）与请求体、回答相同，行出现、提示；扣住回答时 Cancel 停用、Escape 不关、焦点在弹窗里，放开之后关；不在途时 Escape、点背景、Cancel 各关一次；编辑只发改了的字段、名字的初值是视图的名字；删除（`AlertModalCore`，在途不关，确认之后行没了，`viewRow` 的 `deleted`）；复制链接成功与被拒（`refuseClipboardWrites`）各有提示；菜单按钮名为 “{name} options”、右键打开右键菜单（`HO/M3-closeout.md:53`）；开关关着是停用页、没有视图的读 |
| `h12-project-view-page.spec.ts` | `H12 (page)` | 视图页的读表（恰一个 `GET /api/v0/views/{id}`；列表的查询包含 `savedQuery` 的各项：`E/fixtures/views.ts:184` 只给筛选、排序与 `sub_issue`、对分组的视图抛，请求另带 `limit` 与布局的选项）；离开再进来（两秒之内）草稿没了、恰好再一个 `GET /api/v0/views/{id}`（R2，§2.6 的每次挂载读自己的）；不存在的视图（`navigation.view_not_found` 404 → “视图不存在”，回到列表）；读失败 → 重试（只在 Chromium 以路由拒一次，目标页的代码载入之后才装路由）；开关关着是停用页；页头在四种状态下的面包屑；窄屏页头；“Display” 的按钮（`FiltersDropdown`）以它为名、`aria-expanded`、Escape 之后焦点回到它（`HO/M3-closeout.md:56`）；改筛选不发 `PATCH`、“Update view” 出现；所有者“更新视图”：扣住回答时按钮是 “Confirming”，筛选行的条件与添加筛选、“Clear all”、“Save view”、页头的布局与显示都停用（R29），放开之后提示、按钮消失、控件恢复、经接口读回的视图（`viewCalls(api).read`）与回答相同；读表照实数（回答整理过筛选时成功之后多一次列表的读：`resetExpression` 总会通知，`PKG/shared-state/src/store/rich-filters/filter.ts:301-309`）；另一会话经接口关掉开关之后“更新视图”被拒（409 `navigation.views_disabled` 写进期望）、提示、按钮留着、控件恢复、草稿不变；成员看别人的视图没有“更新视图”与编辑；筛选里已删除的标签显示 “Unavailable option”、原样存回（R18） |
| `h12-project-view-saving-page.spec.ts`（`h12-project-view-page` 过 400 行时的拆分出口） | `H12 (page)` | “另存”：视图页、项目页、迭代页（请求体带 `cycle_id`）、模块页（带 `module_id`）各一次，名字的初值为空，显示等于页面已确认的，成功之后转到新视图；没有条件时不给 “Save view”（`canSaveView` 要 `hasActiveFilters`，`filter.ts:271-273`），迭代页、模块页两步先加一个条件；在视图页改布局（宽屏 `LayoutSelection`、窄屏 `MobileLayoutSelection`）只改草稿、不发 `PATCH` |
| `h12-project-view-layouts-page.spec.ts` | `H12 (page)` | 四种布局各存一个视图、各打开一次（列表请求的查询与列出的行）；翻页一次；视图页上行的编辑写进视图页的列表；分组头的“+”与日历某一天的“添加已有”在视图页上可用（R1 的分派点，预检 re-96 I3） |
| `h20-view-pages.spec.ts` | `H20 (page)` | 迟到的回答：列表的读在换项目之后答到不写进新项目的列表；视图的读在经切换器换视图之后答到不写进新视图，回到原视图又一个 `GET /api/v0/views/{id}`（re-96 I2 ①）；“更新视图”与删除在离开页面之后答到，没有提示与跳转，“更新视图”那一步之后回到原视图又一个 `GET`、列出的是新读到的（I2 ②）；另一个标签页换了账号之后答到的“更新视图”不在新会话里提示；键盘：以键盘“更新视图”，成功之后焦点落在筛选行的“添加筛选”按钮（三引擎记焦点）；弹窗的 Tab 次序（图标 → 名字 → 说明 → Cancel → 提交，以 DOM 为准，`tabStops`）、嵌套 Escape（图标选择器开着时只关它）、视图页的布局切换以键盘选（宽屏的按钮与窄屏的菜单各一次，作 M2 交接 §3 `Dropdown` 一半“不适用”的证据，R26）、切换器的 `CustomSearchSelect` 以键盘选（`HO/M6-closeout.md` M6-M7-04）、行菜单以键盘打开与关上之后焦点回到按钮；命令面板：“打开视图”列出、选中跳转，搜索的 “Views” 组，`create_view` 打开新建弹窗，访客没有 `create_view`（若这份过 400 行，命令面板的一步另起 `h20-views-palette-pages.spec.ts`） |

实施中改（T11、T12）：另存一组拆进 `h12-project-view-saving-page.spec.ts`（视图页三个 `test()` 318 行、另存三个 230 行）；命令面板一步拆进 `h20-views-palette-pages.spec.ts`（主文件三个 `test()` 344 行、面板两个 169 行）。“更新视图”在途时 H12 (page) 断言页面上任何地方都没有 “Add filter”（隐去，§2.7），筛选开关本身（显示或隐藏筛选行）不改草稿、产品不停用它，故事不断言它停用；成功之后焦点“筛选行不在时落到页头的筛选开关”一支造不出真实路径，只在 T8 的单测。看板的 “Load more ↓” 被同一列吸底的快速新建条盖住、指针点不到（M4/M6 共用的看板，交 M8），翻页一步以键盘按它。

- **S2**（R23）：`E/stories/smoke/s2-issue-list-removal.spec.ts:16` 的 `project view` 一行拿掉，注释 `:11-14` 加一句 P6a（同 M6 的两次先例）；随之死掉的分支一并删（`:81` 的 `detail`、`:82-85` 的项目一支、`:86` 的 `PROJECT`/`PROJECT_MEMBER`、`:103` 的 `if (detail)`，导入 `:6` 随之），`entries` 只剩工作区视图一行（P6b 再定整份的去留）。这一行钉的是视图页今天的样子（读旧地址得 404、显示 “View does not exist”），T7 重写视图页之后必红：改动在 T7 的同一提交（预检 re-96 I6）。视图不存在的页面进 H12 (page)；T7 到 T11 之间“视图不存在”只有单测的证据。
- **既有故事的风险**：`E/stories/project/p8-project-preferences.spec.ts:332-358` 把 alice 带到视图列表之后不再看她的页面，P6a 加的列表读可能在测试结束时还在途；T4 之后 chromium 跑这份（plan 的允许清单只许加一个等读答完的 `waitForResponse`，断言不改）。
- **故事的写法**：每个“在途”都靠扣住回答（`holdAnswer`），不靠时间；一律放行或拒绝的路由只在 Chromium 装，而且只在目标页的代码（含编辑器）载入之后装（WebKit 在有路由时重取预载，P5b 阶段记录 §7）；换页、离开项目、改服务端的状态之前先等在途的读答完（动作之前登记 `waitForResponse`）；每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去）与 `expectQuietConsole`（不放宽）结束。
- **时长**：每份不过 400 行；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

### 2.12 文件与职责

| 文件 | 职责 | Task |
|---|---|---|
| `C/services/views/views.service.ts`（新）、`C/store/views/{view.store,view-changes}.ts`（新）、`C/store/root.store.ts` | 数据层（`use-views.ts` 随第一个产品读者在 T4 建） | T2 |
| `C/lib/views-entry.ts`（新）、侧栏、页签栏、`project-commands.ts`、`creation/command.ts` 与各自的测试 | 入口 | T3 |
| `V/(list)/*`、`C/hooks/store/use-views.ts`（新）、`VC/{page-state,actions-host,views-list,view-list-item,view-list-item-action,view-list-header,quick-actions,delete-view-modal}.tsx`、`VC/filters/*`、`VC/applied-filters/*`、`C/components/common/filters/{created-at,created-by}.tsx`、`quick-actions-helper.tsx`、成员 store | 列表页、行与菜单、宿主（T4 只持有删除，预检 re-96 I5）、`projectMemberIds` | T4 |
| `VC/{modal,form}.tsx`、`VC/actions-host.tsx`（新建、编辑两个弹窗）、列表页的 “Add view” 与空态的“新建”、行的“编辑”；删 `C/components/dropdowns/layout{,.test}.tsx`、`PKG/{utils,constants}/src/tab-indices.ts` 与桶导出 | 弹窗只管身份并接上新建与编辑、`LayoutDropDown` 与 `getTabIndex` 的删除 | T5 |
| `PKG/types/src/issues/issue.ts`、`C/store/issue/project-view/*`（新）、`C/store/issue/root.store.ts`、§2.5 的分派点、`use-project-view-issue-actions.ts`（新）、`use-project-view-preferences-follow.ts`（新）、`empty-states/*`、`C/hooks/use-issue-list-mount.ts`（键）、`add-existing-dialog.tsx`（请求类型）、`quick-action-dropdowns/project-issue.tsx`（行的快捷操作） | 引擎 | T6 |
| `V/(detail)/layout.tsx`、`[viewId]/{page,header}.tsx`、`[viewId]/mobile-header.tsx`（新）、`VC/use-view-page-read.ts`（新）、`roots/project-view-layout-root.tsx`（新）、S2 的一行（预检 re-96 I6） | 视图页 | T7 |
| `filters-hoc/{project-level.tsx,shared.ts,use-project-view-saving.ts}`、`rich-filters/filters-row.tsx`、`project-layout-root.tsx`、`planning-page-layout-root.tsx`、`planning-page-domains.ts`、`page-preferences-filter.store.ts`（`confirmed`）；在途停用：`C/components/issues/{filters,mobile-header-filters}.tsx`、`issue-layouts/filters/header/{layout-selection,mobile-layout-selection}.tsx`、筛选开关、`issue-layouts/draft-busy.ts`（新）、`issue-layouts/spreadsheet/columns/header-column.tsx`、`issue-layouts/calendar/dropdowns/options-dropdown.tsx`、`empty-states/project-issues.tsx`、`roots/project-view-layout-root.tsx`、`V/(detail)/[viewId]/header.tsx` | 另存与更新视图、在途停用 | T8 |
| `power-k/menus/views.tsx`、`config/navigation/project-commands.ts`（`IProjectView` → `View`）、`open-entity/project-views-menu.tsx`、`ui/modal/{search-menu,search-results,use-view-search}.ts(x)` | 命令面板 | T9 |
| 旧视图层、`PKG/{types,utils,constants}` 的视图文件、`helpers/views.helper.ts`、收藏两处、`K` | 删除与 `old-view-urls` | T10 |
| `E/fixtures/{view-pages,mounts}.ts`、H12 的列表页、视图页、布局（与需要时的另存）故事 | H12 (page) | T11 |
| `h20-view-pages.spec.ts` | H20 (page) | T12 |

实施中补（T4–T12）：表外的文件——`VC/view-content.ts`（T5，`ViewContent` 与 `NEW_VIEW_CONTENT`）、`VC/views-disabled.tsx`（T7 跟进，两页共用的停用页）、`PKG/ui/src/breadcrumbs/navigation-search-dropdown.tsx`（T7，可选的 `onOpen`）、`C/components/rich-filters/filters-toggle.tsx`（T8，落点的标记）、`C/components/core/list/list-item.tsx` 与 `PKG/utils/src/string.ts`（T4 跟进）、`PKG/constants/src/issue/layout.ts`（T6 跟进）、`C/hooks/use-favorite-item-details.tsx` 与 `C/store/favorite.store.ts`（T10、T10 跟进）；测试的夹具 `C/store/views/fake-views.ts`、`VC/fake-view-pages.tsx`、`VC/fake-view-dialog.tsx`、`VC/fake-view-page.tsx`、`V/(list)/fake-list-page.tsx`、`C/components/work-item-filters/filters-hoc/fake-view-saving.tsx`；故事 `h12-project-view-saving-page.spec.ts`（T11）与 `h20-views-palette-pages.spec.ts`（T12）。

## 3. 局部裁定

主控的裁定在台账 `rulings.md`（C-1–C-27 与“起草时的补充裁定”），本节逐条写成 R1–R30：R1–R27 即 C-1–C-27，R28–R30 是主控对起草者 O-1、O-2 与 O-6、另存默认名的裁定（§8）；都是局部裁定。十条是**高风险的局部裁定**（**L!**，动到别的 M 或 M7 前段的网页共用代码），阶段记录写明、收官报告列出：R1、R7、R8、R9、R10、R13、R14、R16、R21（主控点名的九条）与 R29（主控确认：它改 M4/M6 共用的筛选行、页头控件、表格列头与空态）。起草者核出、主控同意照写的各点列在本节末的“起草时的补记”。

| C | R | C | R | C | R |
|---|---|---|---|---|---|
| C-1 | R1（L!） | C-10 | R10（L!） | C-19 | R19 |
| C-2 | R2 | C-11 | R11 | C-20 | R20 |
| C-3 | R3 | C-12 | R12 | C-21 | R21（L!） |
| C-4 | R4 | C-13 | R13（L!） | C-22 | R22 |
| C-5 | R5 | C-14 | R14（L!） | C-23 | R23 |
| C-6 | R6 | C-15 | R15 | C-24 | R24 |
| C-7 | R7（L!） | C-16 | R16（L!） | C-25 | R25 |
| C-8 | R8（L!） | C-17 | R17 | C-26 | R26 |
| C-9 | R9（L!） | C-18 | R18 | C-27 | R27 |
| 补充裁定 O-1 (b) | R28 | 补充裁定 O-2、O-6 | R29（L!） | 补充裁定（另存的默认名） | R30 |

- **R1 `PROJECT_VIEW` 是项目列表页的一种**（C-1 的 (a)，**L!**）：§2.5。理由：视图页要的是“项目的列表、查询来自视图、项目页的命令”，`PagePreferencesIssues` 与 `PagePreferencesFilter` 正是它；塞进 M6 的两张表违反里氏替换（`holds`、`join`、`remove`、`addExisting` 在视图上都是假的）；借 `PROJECT` 会把草稿写进项目页的偏好（违反 `D:120`）。代价：约 15 个分派点各一两行、两份钉 `DEFAULT` 的测试改写；穷尽的 `switch` 让漏一处是编译错；M4 的 `use-issues.ts`、`use-issues-actions.tsx`、`use-issue-layout-store.ts`、四个布局根的联合，M6 的 `use-page-display.ts`、`use-page-issue-filters.ts`、`C/store/issue/root.store.ts`（329 行，P5b 时不碰，本段加构造的几行、不超过 340）；T6 之后跑 m4-list、m4-filters、m4-detail。
- **R2 草稿在内存**（C-2 的 (a)）：§2.6。理由：Plane 相同（`PW/core/store/issue/project-views/filter.store.ts` 播种、本地改、卸下时回到存着的），复用整套换算、不写第二份映射。代价：约 60 行与单测；草稿不跨刷新（Plane 相同）。起草时补：换会话不另调 `discard`（根随会话重建）；`discard` 同时丢 `filters[viewId]`（理由见 §2.6）。预检 re-96 补：视图页的读是每次挂载读自己的（I1）；草稿服务按视图带代，`discard` 作废在途的播种，`adopt` 只在草稿还在时（I2）。
- **R3 视图的写走视图 store 自己的队列**（C-3 的 (a)）：§2.1。理由：连点两次“更新视图”不并发、后到者不覆盖；canonical 只在 nerve 确认时变（v0 §7.7）。代价：约 100 行（`C/store/planning/module-changes.ts` 的形状，简单得多）。
- **R4 控件矩阵**（C-4）：§2.2 的表，每格对到服务端的一条规则，改正今天的三处偏差。理由：页面不给会被拒的控件（M3 交接的原则）。代价：H12 (page) 三种角色各走一遍。
- **R5 列表页**（C-5 的 (a)）：§2.3。去掉访问、锁、收藏、发布；页面状态在以项目为 key 的列表布局的 context 里（P5a 的先例，`HO/M3-closeout.md:35`）；收藏留给 P7（M6 删迭代、模块的星的先例）。代价：删 `VC/filters/filter-selection.tsx:80-89` 的收藏与 `VC/applied-filters/access.tsx`；页面状态约 60 行；登记 plane-diff。
- **R6 视图页**（C-6 的 (a)）：§2.6。以 `viewId` 为 key、宿主在 key 之内；读中、不存在（`navigation.view_not_found`，回到列表）、失败重试、停用四种状态；“Add work item” 给管理员与成员。代价：页面与页头约 350 行的重写，`ProjectViewLayoutRoot` 约 120 行。
- **R7 另存在四个页面给**（C-7 的 (a)，**L!**）：§2.7。迭代页、模块页另存带该迭代、模块的词，经 `PLANNING_PAGE_DOMAINS` 的新列 `viewTerms`。理由：另存出来的视图与用户在页面上看到的是同一批工作项（Plane 不带，存出来的不再是“这个迭代的工作项”，`PW/core/components/issues/issue-layouts/roots/cycle-layout-root.tsx:89-97`）。代价：动 M4 的 `project-layout-root.tsx`、M6 的 `planning-page-layout-root.tsx` 与 `planning-page-domains.ts`、P4 的 `save-view-collapsed.test.tsx`（翻过来并改名）；登记 plane-diff。
- **R8 已确认的线上值**（C-8 的 (a)，**L!**）：§2.7。`PagePreferencesFilter.confirmed(id)` 只读、返回拷贝。代价：一个 getter 与它的单测，三个页面的行为不变。起草时补：它只给另存与“更新视图”；新建、编辑的弹窗不碰显示（R28），不需要换算，M4/M6 的换算不导出。
- **R9 “更新视图”等回答**（C-9 的 (a)，**L!**）：§2.7。理由：今天 `handleViewUpdate` 不返回 Promise（`project-level.tsx:134-166`），失败也重置基线，按钮消失、用户以为存上了（`HO/M4-closeout.md:71` 的“未等待的视图写后关闭”）。代价：HOC 约 40 行、筛选行一行；shared-state 不改。起草时补：`followInSession` 从不 reject，HOC 记下失败自己 throw；成功时回答与当前表达式不等价才 `resetExpression`；在途时改草稿的控件停用（R29）。单测：失败之后按钮还在、成功之后基线是回答。
- **R10 视图页的筛选不推进基线**（C-10 的 (a)，**L!**）：§2.5。代价：视图页的根多约 10 行，不碰 M6 的共用 hook；T7 的单测钉“视图页改了筛选之后 `hasChanges` 为真（基线没被推进）、没有 `PATCH`”，“`canUpdateView` 为真”在 T8 随 HOC 的新读者钉（预检 re-96 I5 ②）。
- **R11 弹窗的关闭政策同 M6**（C-11 的 (a)）：§2.4。理由：同类的短表单（迭代、模块）已经是这样，P6b 的工作区视图弹窗也照它；没有确认放弃（表单丢了不贵）。代价：约 60 行；嵌套 Escape 随 R28 只剩图标选择器，在页面上核（H20 (page)）。
- **R12 弹窗里没有筛选**（C-12 的 (a)）：§2.4。理由：在视图页改筛选是主路径（列表就在下面）；弹窗里的临时筛选实例的寿命、嵌套弹层与两级 Escape 代价高。代价：登记 plane-diff（Plane 有，M4/P10a 删了）。
- **R13 命令面板**（C-13 的 (a)，**L!**）：§2.8。菜单与列表页同一个键、门是 `viewEntryOn`；搜索只在项目范围、本地过滤、新 `use-view-search.ts`；`create_view` 要 `viewsCreatableBy`。代价：动 M6 的命令面板文件；登记 plane-diff（同 M6/P7c 的迭代、模块）。工作区范围的搜索要新接口，是负责人的事，不做。
- **R14 `viewEntryOn`**（C-14 的 (a)，**L!**）：§2.2。名字（实施中改，T3）：调研、裁定与设计原写 `viewsEntryOn`，其中的 “sEntry” 撞上关键词 `unsplash-telemetry` 不分大小写的 “sentry”（该规则只为 `webkitGetAsEntry` 开了例外，规则的说明要求其余前后缀都命中）；不放宽规则，改名为 `viewEntryOn`（`viewsCreatableBy` 与文件名 `views-entry.ts` 不命中，不改）。照 `intakeEntryOn`：开关加三种角色、不排除已归档；侧栏、页签栏、两条导航命令共用；单测表加角色的格。理由：两条入口规则一致更好读，已归档项目的页面由外壳挡住（P5a R27 已接受）。代价：新文件约 25 行、五个调用方；关闭 M6-M7-07 的视图一半。
- **R15 `getTabIndex` 全套删除**（C-15 的 (a)）：§2.9。共用控件非正的 `tabIndex?` 保留、核过的读者写进阶段记录。代价：删约 35 行；H20 (page) 核弹窗的自然 Tab 次序；关闭 M6-M7-05。
- **R16 `projectMemberIds` 删除**（C-16 的 (a)，**L!**）：§2.9。理由：P6a 是最后的迁移者（`HO/M3-closeout.md:121`）。代价：动 M3/M4 的成员 store 与测试、P5a 的收集箱测试替身（只删一个字段）；关闭 M3 交接 §7。
- **R17 行的菜单与宿主**（C-17 的 (a)）：§2.3。理由：删除成功的提示与跳转不能靠已卸下的行（M6 换宿主的原因，`C/components/modules/actions-host.tsx:40-48` 的注释）。代价：宿主约 90 行，行与菜单约 60 行；`useViewMenuItems` 收布尔值不收整个视图（P6b 照用）；右键菜单在页面上以右键打开核（`HO/M3-closeout.md:53`）。
- **R18 筛选引用已删除的东西**（C-18 的 (a)）：照项目页显示“不可用”、原样存回与发出（§2.6）。代价：无新代码；H12 (page) 一步。服务端清理是负责人的事，不做。
- **R19 旧类型、常量、帮手的分工**（C-19 的 (a)）：§2.9 的表；与工作区视图共用的留给 P6b，阶段记录逐个写明。起草时补：`helpers/views.helper.ts` 与 `VIEW_ACCESS_SPECIFIERS` 只有项目视图的读者，P6a 删。
- **R20 视图只在用到的地方读**（C-20 的 (a)）：§2.1；`mounts.ts` 加 `PROJECT_VIEWS`、头注释改写。代价：切换器第一次打开时可能等一次读；命令面板在项目范围、store 里没有这个项目的视图时，每开一次面板一个 `GET /api/v0/projects/{id}/views`，不搜索也发（§2.8）。
- **R21 筛选行与 HOC 的英文**（C-21 的 (a)，**L!**）：§2.7。en 的文字不变；`invalid.tsx` 交 M8。代价：约 10 个键（§8 O-3 的清单，主控已同意）。
- **R22 页面配置与时长**（C-22 的 (a)）：§2.11。`L/final.sh` 改成六个配置（主控在 T1 改）；第 1 路以 P6a 检查点 0 的本机值为基线、+30 秒停查；CI 的八成、web 作业 960 秒、e2e 每片 30 分钟只报告；推送之前照 P4 spec §2.11（`specs/P4-web-notifications.md:270`）估出 e2e 两片。
- **R23 S2 去掉项目视图一格**（C-23 的 (a)）：§2.11。随之死掉的分支一并删。
- **R24 `old-view-urls` 与收藏**（C-24 的 (a)）：§2.9。起草时补：`removeFavoriteEntityFromStore` 只有视图一支，整个方法连同 `viewStore` 与四个调用删去。
- **R25 Task 与检查点**（C-25）：14 个 Task；检查点 0 在引擎改动之前（T5 之后，m4 三个配置的基线）、检查点 1 在视图页之后（T7）、检查点 2 在命令面板与删除之后（T10，产品代码完成）；plan 列出允许清单、性质与变异表。
- **R26 P5b 的回填**（C-26）：T1 回填 P5b 阶段记录 §9 的终检之后、设计 §14 与前端改动清单 §3.1 的 P5b 一行；M2 交接 §3 的 `Dropdown` 一半（`single-select.tsx` 已在 M4/P16 `a3af031b8` 删除，`LayoutDropDown` 随 R28 删）在 H20 (page) 以键盘核过视图页的布局切换（`LayoutSelection`、`MobileLayoutSelection`）之后写“不适用”（T13）。
- **R27 实施者简报的教训**（C-27）：写进 `implementer-brief.md`：WebKit 在路由下重取预载（目标页的代码载入之后才装路由）；一次 `fetchIssue` 是两个 `GET`；swr 2.4.2 的重连只通知一个键上的第一个 hook；变异包的源码之后核 dist 已还原；Docker 重启之后 testcontainers 的端口超时（等 ryuk 退出，以新标签重跑）；故事在换页或改服务端状态之前等在途的读；`expectQuietConsole` 不放宽。
- **R28 弹窗只管视图的身份**（补充裁定 O-1 的 (b)）：§2.4。新建与编辑的弹窗只有名字、说明、图标；布局与显示在视图页改、以“更新视图”存（同 R12 的筛选）。“Add view” 的体以合同默认值的常量给齐 `display_filters` 与 `display_properties`，H12 (page) 的请求体断言钉住。另存（R7、R8）照旧存页面已确认的显示与属性。理由：编辑弹窗改布局而视图页另有未存的草稿时，同一个视图有两个真相来源、谁赢说不清；不动 M4/M6 的换算。代价：`LayoutDropDown` 与它的测试失去唯一的读者、删去（§2.9）；R11 的嵌套 Escape 只剩图标选择器；R26 的 `Dropdown` 核对落到视图页的布局切换；登记 plane-diff（§7 第 4 条）。
- **R29 “更新视图”在途时停用改草稿的入口**（补充裁定 O-2、O-6，**L!**）：§2.7。筛选行的条件、添加筛选、“Clear all”、“Save view”，页头的布局、显示与筛选开关里的添加筛选，表格列头的排序，日历的选项下拉，有筛选时空态的“清除筛选”，在途时都停用（“任何入口”以显示的三个写者与富筛选实例的入口为准，§2.7）；成功之后草稿与基线是回答，失败之后都恢复、草稿与基线不动。理由：不让在途期间的改动被回答覆盖、静默丢失（同 M6、P5b 在途等待的规则）。代价：M4/M6 共用的筛选行（`filters-row.tsx`）与页头的五个组件（`HeaderFilters`、`MobileHeaderFilters`、`LayoutSelection`、`MobileLayoutSelection`、筛选开关）各收一个可选的 `disabled`（默认不停用，不复用 `disabledAllOperations`）；M4 的表格列头、日历的选项下拉与空态读一个新的 `DraftBusyContext`（默认不停用）；项目页、迭代页、模块页不传、不提供，行为不变；视图 store 多一个可观察的 `isUpdating(viewId)`；单测每个入口一格、变异每个入口一行、H12 (page) 扣住回答的一步（日历的选项只在单测）。被否的另一做法：只在草稿未变时把草稿换成回答（在途时照常可改）——shared-state 的 `updateView` 在回调 resolve 之后把基线设为当前表达式（`PKG/shared-state/src/store/rich-filters/filter.ts:499-506`），在途时改的筛选会被当成已存、“更新视图”不再出现，而 R9（C-9）不改 shared-state。
- **R30 另存与新建的名字初值为空**（补充裁定）：§2.4。名字必填，用户自己写；不编造 “{视图名} 2” 一类的默认，也不用 Plane 的英文 “Untitled”。登记 plane-diff（§7 第 4 条）。

**预检 re-96 之后的修正**（主控的处理在 `docs/reviews/re-96-M7P6a预检.md` §6，各条照“须做的修正”写进上文）：
- I1（R2）：视图页的读是每次挂载读自己的（§2.6），重连不读。
- I2（R2、R9）：草稿服务按视图带代，`discard` 作废在途的播种、`adopt` 只在草稿还在时；HOC 的跟进只在它还挂着时（§2.6、§2.7）。
- I3（R1）：分派点补四处——列表挂载的键、日历的“添加已有”、行的快捷操作、`use-issues-actions.tsx` 的行数约束（§2.5）；R1 因此多动 M4/M6 的 `use-issue-list-mount.ts`、`add-existing-dialog.tsx`、`project-issue.tsx`，收官报告列出时写上。
- I4（R29）：日历的选项下拉是第三个显示的写者，也停用（§2.7）；R29 多动 M4 的 `options-dropdown.tsx`。
- I5：T4 的宿主只持有删除，新建与编辑随 T5 的弹窗接上；T7 的根不传 `enableSaveView`、`enableUpdateView`，T8 传（plan）。
- I6：S2 的一行在 T7 的同一提交删；`fake-i18n` 下按字面找 “Clear all”“Save view” 的三处测试在 T8 改键（§2.7、§2.11）。
- m1–m12 与观察 O1、O2、O4、O7：见各节与 plan。

**起草时的补记**（起草者核出，主控 2026-11-24 同意照写，台账 `rulings.md` 末节）：
- R9：`followInSession` 从不 reject（`C/lib/in-session.ts:40`），HOC 在 `failed` 里记下错误、之后自己 throw；成功时把回答 `adopt` 进草稿，回答的筛选与当前表达式不同时先 `resetExpression`，再 resolve（shared-state 随后把基线设为当前表达式，`filter.ts:499-506`）。
- R2：卸下时 `discard` 同时丢 `filters[viewId]`；换会话不另调（`IssueRootStore` 随会话重建）。
- R13：搜索组的标题用已有的键 `views`（没有 `common.views`）。
- R17：`useViewMenuItems` 去掉 `view`、`workspaceSlug`、`projectId` 三个参数（它本来就收 `isOwner`、`isAdmin`），P6b 的 `C/components/workspace/views/quick-action.tsx:52` 一行编译跟随。
- R24：收藏 store 的整个 `removeFavoriteEntityFromStore`（只有视图一支）连同 `viewStore`、它的登记与四个调用删去。
- R19：`helpers/views.helper.ts` 与 `VIEW_ACCESS_SPECIFIERS` 没有 P6b 的读者，P6a 删。
- R23：S2 去掉项目视图一行之后死掉的分支一并删。
- R28 之外：新建（含另存）之后转到新视图、删除之后转到列表，今天的行为保留（`VC/follow-ups.test.tsx:88`、`:133`）。
- 守卫：`project-address-route` 的视图文件样例（`K:2631-2637`、`:2797-2803`）随删改更新。
- R1：`C/store/issue/root.store.ts`（329，P5b 时不碰）本段只加构造，上限 340。
- R22：P8 的故事（`E/stories/project/p8-project-preferences.spec.ts:332-358`）只许加一个等视图列表的读答完的 `waitForResponse`。
- 起草者核出的行号漂移（照今天的行号引）：`.oxlintrc.json` 的 swr 名单 `:121-215`；M6-M7-05 引的桶文件 `:25` 今天是 `:23`；`HO/M3-closeout.md:60` 引的 `views/form.tsx:186` 今天是 `:170`；调研的 `use-project-fetch.ts:82-91` 今天是 `:83-92`、`empty-states/index.tsx:21-31` 今天是 `:20-33`。

**实施中的补记**（2026-11-25，T13 写；来自实施中的主控裁定，经过见[阶段记录](../reviews/P6a-web-project-views-review.md) §3；上面的裁定本身不改）：
- **R1 穷尽的写法**（T6）：`use-issues.ts` 在泛型参数上收窄不了，`switch` 改在非泛型的局部变量上；探针删一个 `case` 恰两处 TS1360。照 plan 字面的两个变异（`t6-mut-dialog-store-missing`、`t6-mut-dragdrop-missing`）编译不过，以守卫与分支上的等价变异 `-2` 杀。空态拆出收类型与 id 的 `ProjectListEmptyState`，`ProjectEmptyState` 留作项目页的包装。
- **R2 草稿的拒绝与会话**（T7、T7 跟进；实施中改（整审修复轮））：草稿服务的 `NoViewDraftError` 导出，读的 hook 以 `instanceof` 认；T7 起初认作“页面已不在”、什么都不画，整审 m1 指出它经共用的键落到另一次挂载上，改为取数函数接住它、照常兑现，从不是共用键上的错，hook 四态（§2.6）；视图页的读的组件 `ViewVisit` 以 `loginId` 为 key（§2.6）。
- **R6 别的项目的视图与停用页**（T7、T7 跟进）：别的项目的视图同“不存在”（页面与页头各比 `project_id`）；列表页与视图页共用 `ViewsDisabled`（§2.6）。删除之后的“不存在”（实施中改（整审修复轮））：只有发出删除的这一页在跳转之前不画，记在它的宿主上（`ViewsPage.pageLeaving`，宿主在 `viewId` 的 key 之内、随页面卸下，同 P5a 收集箱的 `leaving`）；别处的删除与再进来的 404 照常说“不存在”、给回列表。修复轮起初以 `ViewStore.isDeleted`（本会话删过）判，限定复审 n1 指出删除在途时后退、再前进进来的新挂载会因此停在空白、没有出路，改为记在宿主上，`isDeleted` 删去（§2.6）。
- **R9 `adopt` 的转交与 HOC 的形状**（T8）：`ProjectViewIssuesFilter.adopt(viewID, view)` 转给草稿服务（§2.5）；HOC 212 → 73 行，逻辑进 `use-project-view-saving.ts`；“更新视图”给谁复用 `viewsCreatableBy` 再加所有者；拒绝提示经 ref 取最新的 `useRefusalToast`。T8 分两个提交（A 给按钮、B 加在途停用），B 在推送之前 amend 过一次，只为死成员扫描报出的 `filters-row.tsx` 的一个无人读的字段（`be18d601` → `88b15b6e`）。
- **R14 的名字**：`viewEntryOn` 的改名在 T3 之前已改进设计、本 spec 与 plan（`0209625e`，R14 本条）。
- **R17 编辑的空处理函数**（T4）：T4 的宿主只画删除，行菜单的“编辑”照矩阵给、处理函数在 T4 是空的，T5 接上并以 `t5-edit-wired` 核没有空处理函数；分支内一个 Task 的窗口（主控在起草时接受）。
- **R19 哪一项在哪个 Task 删**：`views.helper.ts`、`VIEW_ACCESS_SPECIFIERS` 与 `FavoriteStar` 随最后的读者在 T4 删（不是 T10）；`TViewFiltersSortKey`、`TViewFiltersSortBy`、`VIEW_SORTING_KEY_OPTIONS`、`VIEW_SORT_BY_OPTIONS` 仍被新排序读，留下（§2.9 的表与表后一段）。
- **R20 切换器的门**（T7）：`@nerve/ui` 的 `BreadcrumbNavigationSearchDropdown` 加可选的 `onOpen`，视图页头以“打开过”为读列表的门；不取挂上就读（违背本条）、以点击与按键猜打开、在选项标签里藏读（§2.6）。
- **R21 已应用筛选的名字**（T4 跟进，主控推翻 T4 起初留英文的做法）：`VC/applied-filters/root.tsx` 的名字用筛选面板同一组的键（所有者 `common.created_by`，创建日期 `inbox_issue.filters.created_at`，没有新键），`replaceUnderscoreIfSnakeCase` 没有读者，删去。
- **R24 收藏与 `old-view-urls`**（T10、T10 跟进）：规则也拦项目视图收藏的旧地址 `user-favorite-views`（照 `old-module-urls` 拦 `user-favorite-modules` 的先例）；收藏 store 另删随旧 store 失去最后读者的三个成员，类的 `entityMap` 字段留给 `addFavorite` 的去重（§2.9 的表后一段）。
- **R28 弹窗的接口与 HOC 的接缝**（T5）：弹窗收 `isOpen`、`handleClose`、`workspaceSlug`、`projectId`、`viewId?`、`content?: ViewContent`（`VC/view-content.ts`：`Pick<ViewCreate, "filters" | "display_filters" | "display_properties">`，缺省 `NEW_VIEW_CONTENT`）；HOC 在 T5 只改另存的预填（§2.7），不取要桩的 (B)、要碰 R28 不让 T5 碰的文件的 (C)。嵌套 Escape 的探针表明图标选择器开着时第一个 Escape 被 base-ui 的 Popover 接住，不动 `PKG/propel`；`t5-mut-document-escape` 因此由“Escape 关两次”“在途也关”两格杀。名字的提示用 M6 的 `title_is_required`、`title_should_be_less_than_255_characters`。
- **R29 添加筛选隐去**（T8）：在途时添加筛选经 `AddFilterButton` 的 `isDisabled` 隐去、不是停用（§2.7）；成功之后的焦点等控件恢复才落。日历的选项下拉、表格列头与空态的清除经 `DraftBusyContext`（`C/components/issues/issue-layouts/draft-busy.ts:14`，默认 `false`，只有视图页的根提供）。
- **R30 由类型保证**（T5）：弹窗没有名字的入参，调用方传不了名字；`t5-mut-saveas-name-default` 改写在弹窗一层（给了 `content` 时以 “Untitled” 打开），杀“另存打开时名字为空”一格（plan 原文写在视图页预填 “{视图名} 2”，那时 HOC 还没有新的读者）。
- **R22、R25 本机的门禁**：P8 与页面故事要 `make build build-s3-fixture`（全局准备的 S3 探测经 `bin/nerve-s3-fixture`，`E/fixtures/s3-process.ts:85`；`make build` 不建它）；`make build` 清掉 `.react-router` 的路由类型，之后 `tsc` 与变异的命令先跑 `pnpm --filter web exec react-router typegen`。三个检查点的三引擎页面配置在本机各被一次会话注销打断（WindowServer 因 Playwright WebKit 的 GPU 进程的崩溃快照被杀，不是重启），在同一 HEAD 补跑（阶段记录 §4、§5）。

## 4. 验收标准

- [ ] **A1 数据层**：类型只来自 `@nerve/api-client`；service 不转换；读不排队、写一个视图一个队列、体在轮到时算；回答经 `Reconciled` 写进列表与详情（建、改、删各有格）；没有乐观改；换会话之后旧的回答不写进新的 store；swr 名单收入新文件。
- [ ] **A2 入口与矩阵**：`viewEntryOn` 一处、五个读者；单测表两行不变、加角色的格；§2.2 矩阵逐格有单测，H12 (page) 三种角色核过；三处偏差改正。
- [ ] **A3 列表页**：按项目的页面状态（换项目即丢）；搜索、筛选、排序本地；一行一个快捷操作、菜单名为 “{name} options”、右键菜单；复制经 `useCopyLink`、`copy-hook` 的视图例外删去；弹窗在宿主；删除之后提示与行消失；停用页不读；英文改键。
- [ ] **A4 弹窗**：在途不关、Cancel 停用；不在途 Escape、点背景、Cancel 关；没有文档级 Escape；`ModalTitle`；名字的 `FieldError`、`aria-invalid`、被拒取得焦点；自然 Tab；只有名字、说明、图标（没有布局与显示），编辑只发改了的字段；“Add view” 的体带合同默认的显示与属性（H12 (page) 钉住）；新建与另存的名字初值为空；图标选择器开着时 Escape 只关它（三引擎）；`LayoutDropDown` 与它的测试删去。
- [ ] **A5 引擎**：`PROJECT_VIEW` 在每个分派点（§2.5 的清单，含列表挂载的键、日历的“添加已有”、行的快捷操作，每处单测一格、变异一行；H12 (page) 的四种布局与翻页）；命令清单写全；穷尽的 `switch`；两份钉 `DEFAULT` 的测试照 §2.5 改；视图页的筛选不推进基线、不发 `PATCH`；项目页、迭代页、模块页的行为不变（m4-list、m4-filters、m4-detail、m6-pages 不回归）。
- [ ] **A6 视图页**：以 `viewId` 为 key；四种状态与各自的页头；窄屏页头；每次进页面恰一个视图的读（两秒内离开再进来也读）、列表的查询包含视图存着的；草稿卸下即丢、迟到的播种不建回草稿、再进来以新读到的播种、重连不读；“不可用”的条件原样存回。
- [ ] **A7 另存与更新视图**：四个页面给另存（迭代、模块带词，显示取已确认的线上值）；“更新视图”只给所有者、不看 `is_locked`、比线上值；在途时视图页上每个改草稿的入口都停用（筛选行、页头的布局与显示、表格列头的排序、日历的选项下拉、空态的清除筛选；每处单测一格，H12 (page) 扣住回答的一步）；成功之后提示、按钮消失、焦点落在添加筛选、控件恢复、草稿与基线是回答；离开页面之后答到不提示、不建回草稿；失败之后提示、按钮留着、控件恢复、草稿与基线不动；筛选行与 HOC 的英文改键、en 文字不变（`fake-i18n` 下按字面找的三处改键）。
- [ ] **A8 命令面板**：菜单与列表页同时挂着时共用一次在途的读、不再转圈；搜索的 “Views” 组只在项目范围、本地过滤、不挡别的组；`create_view` 照 `viewsCreatableBy`；`IProjectView` 不在命令面板里（含 `project-commands.ts`）；T9 之后 `t9-palette` 单跑不回归。
- [ ] **A9 删除与守卫**：§2.9 的表逐项删去、读者写进阶段记录；`old-view-urls` 有样例；`project-address-route` 的名单与样例跟上；oxlint 等于上限、新文件零告警；knip；死成员整份比对；plane-diff 的行；前端改动清单；交接的处理结果（§5）；P5b 的回填（T1）。
- [x] **A10 e2e 与收尾**：§2.11 的新故事三引擎各一次、chromium `--repeat-each=3`；六个页面配置与整套默认 e2e 不回归；S2（T7）与 `mounts.ts`；终检、最终分支 CI、合并、main CI 与归档，在 P6b 的 T1 补记（阶段记录 §9：终检 fin 与 finr 全过，最终分支 CI 45773943757、合并 `6dead17d`、main CI 45778376995 都第一次全绿，归档 `M7-P6a-integration-20261126`）。

## 5. 交接与后续责任

**本段处理的输入交接**：都写“处理结果（M7/P6a）”；写关闭的照写关闭，其余照设计 §11 保持 open 至收尾。T13 写。

| 交接 | 条款 | 本段 | Task |
|---|---|---|---|
| `D:240`（§7 数据层） | 五个领域的 store 与 service 照 v0 7.7 重写、删手写类型 | 视图一半：§2.1、§2.9；工作区视图随 P6b | T2、T10 |
| `D:242-243`（§7 先收起会 404 的入口） | P4 收起“Save view”，P6a 接回 | 关闭：§2.7（R7） | T8 |
| `D:246`（§7 视图） | 列表引擎加 `PROJECT_VIEW`、另存、更新视图、命令面板的视图 | 项目视图一半关闭；工作区视图随 P6b | T6–T9 |
| `D:248`（§7 交接的页面项） | M3 交接 §3、§4 的可访问性与地址规则 | 视图的部分逐项见下 | T4、T5、T7 |
| `D:308`（§12 P6a） | H12 的页面、H20 的视图页 | §2.11 | T11、T12 |
| `HO/M3-closeout.md:31` | `ProjectViewStore.loader`、`fetchedMap` 没有写者 | 关闭：旧 store 删除 | T10 |
| `HO/M3-closeout.md:38` | 嵌套的 id 没有 key（视图页） | 项目视图一半关闭（`<Fragment key={viewId}>`）；`workspace-views/[globalViewId]` 随 P6b | T7 |
| `HO/M3-closeout.md:43-44` | 弹窗的 Tab、Escape、点外面、焦点、portal；弹窗的名字 | 视图的弹窗关闭（§2.4，H12、H20 (page) 三引擎）；工作区视图的随 P6b | T5、T11、T12 |
| `HO/M3-closeout.md:50`、`:52` | 嵌套的按钮；每一行的控件以它那一行为名 | 视图行关闭（§2.3） | T4 |
| `HO/M3-closeout.md:53` | 右键菜单在 M7 的页面上以右键打开核对；焦点轮廓 | 右键菜单关闭（H12 (page)）；焦点轮廓照旧交 M8 | T4、T11 |
| `HO/M3-closeout.md:56` | `FiltersDropdown` 的 M7 调用方随页面的筛选故事：断言按钮的名字、`aria-expanded`、Escape 之后焦点回到它 | 视图列表页的 “Filters” 与视图页的 “Display” 各一步（H12 (page)，§2.11）；其余保持 open | T11 |
| `HO/M3-closeout.md:60` | 两个视图表单的名字的提示（点名 `views/form.tsx:186`，今天 `:170`；与 `workspace/views/form.tsx:110`） | 项目视图一半关闭（`FieldError`）；工作区视图的随 P6b | T5 |
| `HO/M3-closeout.md:73`、`:76`、`:78` | 地址只从路由参数；复制只经 hook（`views/quick-actions.tsx`）；视图页面的英文 | 项目视图的部分关闭；工作区视图的随 P6b | T4、T7、T8 |
| `HO/M3-closeout.md` §5（`:91-99`） | M4/P14 共用编辑器的接续：统一 Escape、dirty、busy | 写处理结果：视图弹窗照 M6 的规则（没有编辑器时只有 busy）；顶层保持 open | T5 |
| `HO/M3-closeout.md` §6（`:101-109`） | `getTabIndex` 的最后删除条件 | 关闭（R15），共用控件的非正 `tabIndex?` 核过的读者写进 | T5 |
| `HO/M3-closeout.md` §7（`:111-125`） | `projectMemberIds` 的最后读者 | 关闭（R16） | T4 |
| `HO/M4-closeout.md` M7-03（`:51-63`） | 依据 Plane 在新引擎上恢复 `PROJECT_VIEW`（`:54`）；旧 `view.service` 随 P6a 删（`:62`） | 项目视图一半关闭；工作区的跨项目列表随 P6b | T6、T10 |
| `HO/M4-closeout.md` M7-04（`:65-89`） | 命令面板的视图结果（`:69`）；未等待的筛选与视图写后关闭，统一 dirty/pending/busy、输入保留、嵌套 Escape/IME、拒绝与换宿主迟到答（`:71`） | 照 P5b 的写法逐项（下表）；收集箱的部分 P5a、P5b 已写；本 handoff 顶层保持 open 至收尾 | T5、T8、T9 |
| `HO/M4-closeout.md` M7-05（`:91-103`） | 最后读者与全应用收口：`projectMemberIds`（`:93-99`）、`getTabIndex`（`:101`） | 两项都关闭：`projectMemberIds` 的视图两个读者（收集箱的一个 P5a 已改）与 getter（R16）；`getTabIndex` 与两份 `tab-indices.ts`（R15） | T4、T5 |
| `HO/M6-closeout.md` M6-M7-03（`:34`） | 视图的迭代与模块词 | 项目视图的另存带迭代、模块的词（R7）；工作区的随 P6b | T8 |
| `HO/M6-closeout.md` M6-M7-04（`:48`） | 视图的 store、读与页面，`projectViewIds`、`projectMemberIds`、命令面板的 `IProjectView`、M3 的三项视图遗留（含页头的 `CustomSearchSelect` 键盘） | 项目视图一半关闭 | T2、T4、T7、T9、T12 |
| `HO/M6-closeout.md` M6-M7-05（`:65`） | `getTabIndex` 与两份 `tab-indices.ts` 的最后删除（引用的桶文件行号 `:25` 今天是 `:23`） | 关闭 | T5 |
| `HO/M6-closeout.md` M6-M7-07（`:83`） | 访客在命令面板看到视图与收集箱的命令 | 视图一半关闭（R14）；收集箱一半 P5a 已写 | T3 |
| `HO/M2-closeout.md` §3（`:39-50`） | 下拉框（M7 的部分）：视图表单的 `Dropdown`（`single-select.tsx`）、视图页头的 `BreadcrumbNavigationSearchDropdown` | `Dropdown` 一半写“不适用”（`single-select.tsx` 已在 M4/P16 `a3af031b8` 删；视图表单不再有布局，`LayoutDropDown` 随 R28 删；视图页的布局切换在 H20 (page) 以键盘核过）；页头的切换器以键盘核过 | T5、T12、T13 |
| `specs/P4-web-notifications.md:379-381`（P4 R17） | `enableSaveView`、`enableUpdateView` 与工作区 HOC 的分支从 P4 到 P6a/P6b 没有传入者，由 P6a/P6b 关闭 | 项目一侧关闭（R7、R9）；工作区 HOC 随 P6b | T8 |
| `reviews/P5b-web-intake-files-embed-review.md` §7、§9 末句 | P6a 的各项；终检、最终分支 CI、合并与 main CI 由 P6a 的 T1 补记 | 回填 | T1 |

**M7-04 `:71` 的逐项**（本段做的 / 交给 P6b 的）：
- discard 与文档 Escape：视图弹窗的文档级 Escape 删去，Escape、点背景、Cancel 在不在途时一条规则（§2.4，T5；H12 (page)）。工作区视图的弹窗随 P6b。
- dirty/pending/busy：视图弹窗没有编辑器、没有放弃确认（同 M6 的迭代、模块弹窗），在途不关、Cancel 停用（T5）；“更新视图”在途时视图页改草稿的入口停用（R29，T8）。
- 输入保留：被拒留着表单与输入（T5）。
- 嵌套 Escape：图标选择器开着时 Escape 只关它（T5；H20 (page) 三引擎）。
- IME：组字时的 Escape 由 `ModalCore` 阻止，T5 一格一变异；Enter 不另写（§2.4）。
- 拒绝：名字的 `FieldError`、`aria-invalid` 与焦点，别的拒绝经提示（T5；H12 (page)）。
- 换宿主迟到答：弹窗与宿主的跟进只在还挂着时（T4、T5）；“更新视图”的跟进只在 HOC 还挂着时（T8，`t8-mut-update-follow-unmounted`）。
- 未等待的筛选与视图写后关闭：“更新视图”等 nerve 的回答（R9，T8）；工作区 HOC 随 P6b。

**后续**：
- **P6b**：工作区视图的页面、`globalView` store、`workspace-level.tsx`、`EViewAccess`、`IWorkspaceView`、`workspace-views.ts`、`useViewMenuItems` 只剩布尔参数之后的工作区读者、`old-view-urls` 扩到工作区视图、S2 的最后一行、`HO/M3-closeout.md:38` 的 `workspace-views/[globalViewId]`、M6-M7-03 的工作区一半。
- **P7**：收藏的写与视图的收藏星。
- **M7 收官报告**：R1、R7、R8、R9、R10、R13、R14、R16、R21、R29 是动了别的 M 或 M7 前段的网页共用代码的局部裁定（R1 含列表挂载的键、行的快捷操作、“添加已有”的请求类型；R29 含日历的选项下拉）。
- **M8 的候选**：`C/components/rich-filters/filter-item/invalid.tsx` 的英文（R21）；共用组件的焦点轮廓（`HO/M3-closeout.md:53`）。

## 6. 风险

- **引擎的分派**（R1）：漏一个分派点，两个 `switch` 今天是运行时错（`default` 抛错），其余多是静默的（联合在 `as` 之后、可选链、没有 `default`、点击时才查，§2.5）。缓解：两个 `switch` 穷尽、编译器抓；分派点逐个列、每处单测一格一变异；H12 (page) 四种布局与翻页；T6 之后 chromium 的 m4-list、m4-filters、m4-detail；检查点 0 那一次是它们的基线。
- **“更新视图”的基线**（R9、R10）：shared-state 在 `onViewUpdate` resolve 之后把基线设为当前表达式；先写错了顺序就会把在途时的改动当成已存。缓解：在途时改草稿的控件停用（R29）；成功时先替换当前表达式（不等价时）再 resolve；单测钉“失败之后按钮还在”“成功之后基线是回答”“在途时每个改草稿的入口都停用”，各有变异；H12 (page) 扣住回答走一遍。
- **草稿的寿命**：`discard` 漏掉 `filters[viewId]`，再进来时富筛选的基线会是旧草稿；卸下之后才答到的播种或“更新视图”把草稿建回来，再进来就不读。缓解：草稿服务按视图带代、HOC 的跟进只在挂着时（§2.6、§2.7）；单测“离开再进来，读答到之前不显示旧草稿、基线是新读到的”“扣住播种、`discard`、放开、再 `get` 恰再读一次”“离开之后答到的更新不提示、没有草稿”，各有变异；H12 (page) 一步，H20 (page) 两步之后各回到原视图。
- **M4/M6 共用的件**（R8、R29）：`confirmed` 只读、返回拷贝；筛选行与页头五个组件的停用是可选的 prop 或筛选行自己的在途状态，项目页、迭代页、模块页不传、行为不变；m4-filters、m4-list 与 m6-pages 不回归。
- **命令面板的读**：菜单与搜索在面板打开时才读，键与列表页相同；读被拒不挡别的组；搜索在 store 没有这个项目的视图时每开一次面板读一次（R20 的代价）。缓解：单测与 H20 (page)；T9 之后 chromium 单跑 `t9-palette`（打开命令面板的故事多数不在六个页面配置里）。
- **既有故事**：P8 的视图列表一步（§2.11）；S2 的项目视图一行（T7 的同一提交删，§2.11）；`fake-i18n` 下按字面找 “Clear all”“Save view” 的三处单测（T8 改键，§2.7）；`E/fixtures/mounts.ts` 的读表若有故事精确比对视图页，随之。
- **WebKit 的预载**：任何 `page.route` 在目标页代码载入之前装会让 WebKit 重取预载、报控制台警告；读失败的一步只在 Chromium、只在代码载入之后装路由。
- **swr 的去重与重连**：普通的挂载读在去重窗口（2000 毫秒）里再挂上不验证，`discard` 之后会停在加载（预检 re-96 I1）；视图页的读因此每次挂载以 `mutate()` 读自己的，关掉挂载与重连的再验证（§2.6），重连不读。T7 以真的 hook 与真的 SWR 钉“卸下立刻再挂上恰再一个 `GET`”。swr 2.4.2 的重连只通知一个键上的第一个 hook；视图页的读与切换器不共用键（切换器用 `PROJECT_VIEWS`）。
- **合同默认值的第三份**（预检 re-96 O5）：“Add view” 的默认常量是合同 `default`、服务端 `DefaultListDisplayFilters`（`navigation/domain/content.go` 的 `CheckNewView`）之外的第三份；合同的默认一变，三处都要改，H12 (page) 的请求体断言先红。
- **时长与 CI**：单测约 1,500–2,000 行新增、约 600 行改写；web 作业在最终分支 CI 已 1075 秒（P5b，`.376`），越过 960 秒几乎是确定的，照 R22 只报告、三步对比判别。e2e 加约 3 个文件、约 15–20 个 chromium 测试，估约 +150–250 秒（估算，未测）；P5b 检查点 2 的两片 1471、1554 秒（多在装浏览器的下载，P5b 阶段记录 §5），main CI `45375109166` 的第 1 路 729.2 秒（60.8%）。CP2 之前照 P4 spec §2.11 估出，超过 30 分钟先报告、不推送。

## 7. 与 Plane 的差异（T13 登记进 `docs/v0/plane-diff.md` 的行为差异一节，与 M7/P5a、P5b 的行同处）

1. 视图列表页没有公开私有、锁、收藏星与发布；筛选只有所有者与创建日期；搜索、筛选、排序按项目、离开即丢；Plane 的筛选与搜索是全局一份（`PW/core/store/project-view.store.ts:58`）（R5）。
2. 视图页的“更新视图”等 nerve 的回答：在途时视图页改草稿的控件停用；失败时提示、按钮留着；成功之后基线是 nerve 的回答；Plane 的 HOC 不等回答、在途不停用，失败也重置（R9、R29）。只给所有者，不看锁（社区版的锁只读，`D` §3.7）。
3. 迭代页、模块页的“另存”带该迭代、模块的条件；Plane 只存页面的筛选（`PW/core/components/issues/issue-layouts/roots/cycle-layout-root.tsx:89-97`）（R7）。
4. 视图的弹窗没有筛选（Plane `PW/core/components/views/form.tsx:264-289`；M4/P10a 已删）；Escape、点背景、Cancel 不在途时直接关，在途不关；名字的提示在字段下、被拒取得焦点；Plane 是文档级 Escape、点背景不关（R11、R12）。弹窗只有名字、说明、图标：Plane 的表单另有布局与显示（`PW/core/components/views/form.tsx:219`、`:233-257`），nerve 的布局与显示在视图页改、以“更新视图”存；“Add view” 的体带合同默认的显示与属性（R28）：新建的视图不分组，Plane 的表单默认按状态分组（`PW/core/components/views/form.tsx:52`；fork 里同在 `VC/form.tsx:46`，T5 随布局与显示删去）。新建与另存的名字初值为空；Plane 另存时是“{视图名} 2”或 “Untitled”（`PW/core/components/work-item-filters/filters-hoc/project-level.tsx:110`）（R30）。
5. 命令面板搜索视图只在项目范围、在已读到的列表里本地过滤；Plane 在服务端、工作区与项目两种范围（`plane/apps/api/plane/app/views/search/base.py:210-232`）（R13）。
6. 入口照 `viewEntryOn`：开关开着、三种角色，不排除已归档的项目（外壳挡住）；“新建”“另存”与 `create_view` 只给管理员与成员（Plane 的空态给访客）（R4、R14）。
7. 一行一个右键菜单、菜单按钮以视图的名字为名、复制失败有提示（R17）。
8. 视图只在用到的页面读，不在项目的外壳读（Plane `PW/core/layouts/auth-layout/project-wrapper.tsx:134`）（R20）。

（第 1、2、4、5 条里 Plane 的行号由 T13 对照主目录的只读 `plane/` 逐一核过。）

## 8. 起草时的补充裁定（2026-11-24，主控对起草者的 O-1–O-6 与其余；台账 `rulings.md` 末节）

- **O-1 弹窗的布局与显示**：取 (b)。新建与编辑的弹窗只管视图的身份——名字、说明、图标；布局与显示在视图页改、以“更新视图”存（同 C-12 的筛选）。“Add view” 的体以合同默认值的常量给齐 `display_filters` 与 `display_properties`（H12 的请求体断言钉住）。不导出 M4/M6 的换算；另存（C-7、C-8）照旧存页面已确认的显示与属性；C-11 的嵌套 Escape 名单随之缩短；C-26 的 `Dropdown` 核对落到视图页的布局切换；登记 plane-diff。理由：编辑弹窗改布局而视图页另有未存的草稿时，同一个视图有两个真相来源。→ R28。
- **O-2 “更新视图”在途时**：改为在途时停用视图页的筛选行、显示与布局的控件（busy），不让在途期间的改动被回答覆盖、静默丢失；成功之后草稿与基线是回答，失败之后控件恢复、草稿与基线不动。单测一格、变异一行（在途时控件仍可用 → 被杀）、H12 (page) 扣住回答的一步。→ R29。
- **O-3 新文案键**：同意：`project_views.add_view`；`project_views.not_found.{title,description,action}`；`project_views.action.{create,update,delete}.success`（标题用已有的 `toast.success`）；`common.filter_options.{save_view,update_view,confirming}`（筛选行；“Filters”“Clear all” 用已有的 `common.filters`、`common.clear_all`）。en 与 zh-CN 同一提交；其余用已有的键（`created-at.tsx` 的 “Created date” 用 `inbox_issue.filters.created_at`、“Custom” 用 `common.filter_options.custom`；HOC 的 “We couldn't find the view” 一条随 R9 删去）。超出这张单子的先记台账报主控。
- **O-4 `RootStore` 的字段名**：同意：`RootStore.views`（`ViewStore`）在 T2 加、与旧的 `projectView` 并存到 T10，`C/store/root.store.test.ts:106` 的计数 20 → 21 → 20。
- **O-5 检查点 0 的位置**：同意：T5 之后、引擎改动之前，兼作 m4-list、m4-filters、m4-detail 的基线。
- **另存的默认名**：各页都空，不编造 “{视图名} 2”，也不用 “Untitled”。→ R30。
- **起草者核出的其余各点**：都同意照写，见 §3 末的“起草时的补记”。
- **O-6 R29 之外改草稿的路**（起草后新发现）：两处都停用——表格列头的排序与有筛选时空态的“清除筛选”，从 `isUpdating(viewId)` 经 `DraftBusyContext`；每处单测一格、变异一行；（预检 re-96 I4 又找出第三处：日历的选项下拉，同样停用；“任何入口”以写者的 `git grep` 为准，§2.7。）R29 确认为 **L!**；它的代价里写明被否的另一做法（只在草稿未变时换成回答）为何不行。→ R29。
- **`LayoutDropDown` 在 T5 删**：同意。
- **预检 re-96 的处理**（`docs/reviews/re-96-M7P6a预检.md` §6，2026-11-24）：结论修正后通过。I1–I6、m1–m12 照“须做的修正”改进本 spec 与 plan（选择：I2 的机制是草稿服务自己的代；I3 (c) `ProjectIssueQuickActions` 读 `IssuesStoreContext`、四个根不改；I5 ① 取 (a)；m6 的落点是添加筛选；m7 取经接口读回与“包含”；m4 ③ 加 `t9-palette`）。观察：O1 `t8-mut-body-at-click` 的格写在 HOC 加假队列的那一层（plan T8）；O2 写在 §2.7；O3 读表照实数（§2.11）；O4 迭代、模块的另存先加条件（§2.11）；O5 记在 §6；O6 照 R22 只报告；O7 plan 写明 T8 两个提交的分界。
