# M7 / P6b plan：网页的工作区视图与跨项目列表——工作区标签目录、视图页、列表页、另存与更新视图

> spec：[P6b spec](../specs/P6b-web-workspace-views.md)（§2 是合同；§3 的 R1–R32 是裁定，R-n 即台账 `rulings.md` 的 D-n；**L!** 是 R1、R2、R5、R7、R9、R10、R14、R15、R17、R18、R20，R10 里 O1 的选法（P6a 项目视图的“更新视图”一并改为从草稿取体）经主控确认为 L!；**R16、R8 是负责人的**（2026-11-27）；§3 末是起草时核出的十二条补记，主控照收；R29–R32 是主控对起草时四件待定的裁定：六个新文案键、“不存在”回列表页、同名状态不带标识（交 M8）、目录的持有形状照起草；spec §8 已标为已裁定；预检 [re-97](../../../reviews/re-97-M7P6b预检.md) 的 I1–I5、m1–m9 主控全部采纳，已照它的修法并入 spec 与本 plan，观察 o1、o3、o4、o6 的建议一并并入）。基线 main `6dead17d`（`6dead17d7aa199f8d124cfdb403eacc64aedb1e5`，P6a 的 noff）。台账 `L` = `.superpowers/sdd/P6b-web-workspace-views/`。缩写同 spec。

**与调研草案（`research/scope.md` §7）的对照**：草案 14 个 Task，负责人批了 Q16 (a) 之后到 16 个（R26）：接口两个（T2、T3）紧接 T1；工作区标签目录拆成目录与草稿页（T4）、个人页（T5）两个，个人页动的是 M4 的故事与页面（R18，L!），单独一轮审查；数据层与入口并为 T6；弹窗、宿主、行菜单与删除按容器泛化提到引擎之前（T7），视图页的页头要用它们；草案的 T9（删除）并进列表页（T11），因为旧组件、旧 store、旧 service 与旧类型最后的读者都在列表页，照“删掉最后一个读者的 Task 同一提交删去”只能在那里删；S2 的移除冒烟随视图页（T9）删（spec R22 的补记）。检查点 0 在 T7 之后（引擎改动之前），1 在 T10 之后（视图页与另存完成），2 在 T11 之后（删除完成，产品代码完成）。

## Global Constraints

**分支与提交**
- 主目录永远在 main。提交用详细中文，前缀 `feat/fix/docs/test/refactor(M7/P6b)`，结尾一行 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。提交只在本 Task 的 lint 过了之后，不与它串在一条命令里，用显式路径（`git commit -F <msg> -- <paths>`）。
- 实现代理不推送、不合并；检查点由主控推送。
- 关键词的 phase 已是 `M7/P6`（`K:2`，P6a、P6b 共用），不改；不延长任何 `until`。
- 只有 T2 改 `api/` 与 `server/`（spec R16 的范围）；不改 `deploy/`、`.github/`；设计由主控改（T1、T15）。超出 spec 的契约或架构改动先停下报告。
- 新写的代码注释写 “M7/P6b spec”，不写裸的 “P6b spec”（已有 76 处指 M6/P6b），不改旧注释。

**只读与禁区**：`plane/`、`refer/` 只读，不在其内跑 git。不 stash、`git clean`、`reset --hard`、强推；还原变异只经 `mutate.py`。不动别的项目的容器。容器套件一次一个：`make test`、`make test-shard`、`make e2e` 与 Playwright 的配置不并发。

**代码规则**（总体设计 §7.2、§7.7）
- 手写文件约 400 行以内，新代码进新文件，按关注点分（spec §2.16 的表）。新文件的版权头 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（`View`、`ViewCreate`、`ViewUpdate`、`ViewFilters`、`Label`、`IssueListPage`、`operations`）；service 不转换；store 只经自己的根找兄弟；SWR 只经 `useSessionSWR`、只用它的选项（`C/lib/use-session-swr.ts:38-46`）；新代码不读 router store 的地址（不用 `currentWorkspace`、`joinedProjectIds`、不带 slug 的工作区级 `allowPermissions`）。
- 视图的写只经视图 store 的队列，请求体在轮到时算（`docs/v0/v0-design.md:638`）；`await` 之后的提示、导航、关弹窗一律经 `followInSession`；拒绝经 `useRefusalToast`；不 `console.error`。
- 删掉不用的代码，不加开关、桩、只给测试用的钩子或兼容别名。
- 名字里不得有不分大小写的 “sentry”（`K:1224` 的 `unsplash-telemetry`，P6a 的教训）：`workspaceViewsOn`、`workspaceViewsCreatableBy` 可以；`…ViewsEntry…`、`viewsEntry…` 一类不行。
- 不放宽 `K` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`、`expectQuietConsole`。
- **行数**（`wc -l` 于基线）：测试与故事每份不过 400 行（已超的 `i19-member-issues.spec.ts` 528、`s2-web-app.spec.ts` 582、`store-context.test.ts` 545、`issue-filters.test.ts` 495 不变长，新格进新文件）；`C/store/issue/root.store.ts` 338（T8 只加两个字段与它们的构造，不过 350，R1）；`C/hooks/use-issues-actions.tsx` 353（T8 只加一个导入、一行无条件的调用、一个 `case`）；`C/hooks/store/use-issues.ts` 118；`C/hooks/use-issue-list-mount.ts` 68（不过 85）；`C/store/issue/helpers/page-preferences-filter.store.ts` 298（只改类型，不过 300）；`C/lib/issue-filters.ts` 349（不过 400）；`C/store/issue/helpers/issue-list-binding.ts` 286（不过 300）；`C/lib/paged-groups.ts` 445（已过 400：T8 只加 `ListContext` 的一臂（+1 行）、改 `:332` 一行，别的不动）；`C/lib/issue-membership.ts` 112（工作区支与共用的小函数，不过 150）；`C/store/views/view-changes.ts` 134（写的一层，不过 220）；`C/store/label.store.ts` 208（不过 260）；`filters-hoc/workspace-level.tsx` 208（改后不过 150）；`filters-hoc/{use-view-update,use-project-view-saving,use-workspace-view-saving}.ts` 各不过 150；`SV/adapter/http/labels_test.go` 290（不长，新格进新文件）；`BS/permission_matrix_labels_test.go` 282；`SV/adapter/postgres/failures_test.go` 308（只加一行与头注释一句）；`C/components/profile/profile-page.test.tsx` 326（T5 改写三格，不过 400）。

**knip 与第一个读者**（新文件、新导出在它第一个读者出现的 Task 里建立；测试数据的导出以第一个读它的测试为准）：
- T2：`ListWorkspaceLabels`、`WorkspaceLabelLister`（读者：`module.go`、处理器）；`ActionWorkspaceLabelList`（读者：用例、`Actions()`）。
- T4：`LabelsService.listInWorkspace`（读者：标签 store）；`fetchWorkspaceLabels`（读者：新 hook）；`useWorkspaceLabelCatalog`（第一个产品读者：草稿页）；`E` 的 `WORKSPACE_LABELS`（读者：`s2-web-app`）。
- T6：`ViewContainer`、`containerKey`（读者：`ViewChanges`、`ViewStore`）；`createWorkspaceView`（读者：写）；`workspaceViewsOn`（读者：`ViewStore.getViewById`）。
- T7：`VC/use-view-rights.ts`（读者：行菜单）；`VC/view-paths.ts`（`viewPath`、`viewsPath`、`pageViewParam`；读者：弹窗、行、删除）；`WORKSPACE_VIEW_CONTENT`（读者：弹窗）；`workspaceViewsCreatableBy`（读者：`use-view-rights.ts`）。
- T8：`GLOBAL`；`GlobalIssues`、`GlobalIssuesFilter`、`DefaultViewPreferences`、`workspacePageId`、`defaultViewOfPage`（读者：`IssueRootStore`、分派点）；`toWorkspaceFilters`、`workspaceListQuery`、`WorkspacePageFilters`、`ListPageFilters`（读者：引擎与 `PagePreferencesFilter`）；`IssueListService.listWorkspace`、`WorkspaceIssueQuery`（读者：`readPage`、`ListContext`）；`useGlobalIssueActions`、`useGlobalPreferencesFollow`（读者：`use-issues-actions.tsx`）；`DefaultWorkspaceViewKey` 与表的 `own`（读者：页面 id、`GlobalIssues`）。
- T9：`WorkspaceViewLayoutRoot`、`WorkspaceSpreadsheetRoot`、`GlobalViewEmptyState`、`useShownWorkspaceView`、`useDefaultViewRead`（`WC/use-default-view-read.ts`）（读者：视图页）；`WC/default-view-menu.tsx`（读者：视图页页头，T11 的默认行是第二个读者；预检 I4）；`listWorkspaceViews`、`ViewStore.fetchWorkspaceViews`、`getWorkspaceViews`（读者：视图页页头的切换器）。
- 没有“成员先于读者”的例外：类的成员 knip 查不出，死成员的整份比对（T4、T8、T9、T11）每一行要有去处。
- T10：`useViewUpdate`、`useWorkspaceViewSaving`（读者：两个 HOC）。
- T11：`WC/page-state.tsx`、`default-view-row.tsx`（读者：列表页）。
- 删掉某个导出最后一个读者的 Task 同一提交删去它（T8 的 `getFilterConditionBasedOnViews`、`my_issues` 的旧名；T9 的 `WC/quick-action.tsx`、`WC/default-view-quick-action.tsx` 与 `TStaticViewTypes`/`STATIC_VIEW_TYPES` 的三行——它们最后的读者 `WV/header.tsx` 在 T9 删，留到 T11 则 T9 的 knip 必红（预检 I4）；T11 的其余旧层）。knip 查不出只被测试读的产品导出与包的桶导出：逐 Task 的审查与死成员的整份比对兜底。

**告警上限**：`lint-cap` 要求 oxlint 的告警数恰好等于上限（web `W/package.json:10` 现为 14；e2e、`@nerve/constants` 为 0）。新文件不得新增告警；删掉或改写掉带告警的写法时，同一提交照实改上限并留收据（`t<N>-oxlint-count`；T9 删五个旧文件、T11 删其余旧层，两处都要）；T16 只核对。

**文案**：新键只有 spec R29 的六个：`workspace_views.action.{create,update,delete}.success`（T7 加，en 同 `project_views.action.*`）与 `workspace_views.not_found.{title,description,action}`（T9 加，en：“View does not exist”、“The view you are looking for does not exist, or you may not see it.”、“Go to views”）；工作区的页面不读 `project_views.*`；en 与 zh-CN 同一提交；写完整的字面量（`tools/i18n-keys.mjs` 才认得），按容器取键用键写全的一张表，不许一个键是另一个键的前缀。别的一律用 spec §2.14 列的已有的键。

**守卫**
- `workspace-address-route`（`K:2844`）：每个新建或改写的、读工作区的文件收入文件名单并带 hit 样例（spec §2.13 的清单）；`WV/(list)/page.tsx` 只在 T11 重写之后收（T9 原样移过去的它还读 `currentWorkspace`，`WV/page.tsx:25`、`:28`），T9 只收 `(list)/{layout,header}.tsx`；`project-address-route`（`K:2334`）里 P6a 的 `VC` 文件留着，改名或删去的文件同步改两条规则的样例；每次改 `K` 跑 `node tools/keywords.mjs`。
- `.oxlintrc.json` 的 swr 名单（`:205-219` 那一组）：每个新的取数文件收入（T4 的 `use-workspace-label-catalog.ts`，T9 的 `(detail)/[globalViewId]/header.tsx`，T11 的 `(list)/page.tsx` 或它读已存视图的组件），删掉的文件移出。
- `copy-hook` 的两个工作区视图例外（`K:3424-3437`）在 T9 随 `WC/quick-action.tsx`、`WC/default-view-quick-action.tsx` 删（文件没了例外即成 stale，`tools/keywords.mjs:182-211` 报错）；`old-view-urls`（`K:3257`）在 T11 扩（与旧 service 同一提交）。
- `project-address-route`（`K:2334`，内容 `K:2342`）把两参、三参的 `allowPermissions(…)` 都算 hit：`VC` 文件按容器取角色一律走 `getWorkspaceRoleByWorkspaceSlug` / `getProjectRoleByWorkspaceSlugAndProjectId`，不换成 `allowPermissions`，不放宽规则（预检 o3）。

**允许改的已有测试与夹具**（改之前在台账逐条写明，只照这里写的改法；其余断言是回归，不改；编译跟随只为编译、不改断言，逐处记台账；任何一条要改断言而这里没写，先停下报告）：
- T2：`server/internal/modules/access/domain/rules_test.go:37` 旁加一行；`SV/adapter/http/handler_test.go` 的替身加一个字段（`:83-84`、`:260-264`、`:280` 旁，编译跟随）；`BS/permission_matrix_labels_test.go` 的 `labelMatrixRows()` 加一行与它的检查函数；`SV/adapter/postgres/labels_test.go` 加一格；`SV/adapter/postgres/failures_test.go` 的 `TestAFailedReadIsAnErrorNotAnAnswer` 加 `ListWorkspaceLabels` 一行（照 `ListWorkspaceStates` 的 `:135-136`），头注释 `:41-42` 的 labels 一句补上 `listWorkspaceLabels`（预检 m2）。别的 Go 测试不改（bootstrap 的“动作并集等于规则表的键”自动覆盖新动作）。
- T4：`E/stories/smoke/s2-web-app.spec.ts:148-151` 的 `draftsVisit` 加 `...WORKSPACE_LABELS`（与导入）；`E/fixtures/mounts.ts` 加常量、头注释 `:5-16` 补一句；`C/components/issues/workspace-draft/root.test.tsx:59-75` 与 `page.test.tsx:98`、`:104-129` 的调用断言（`:104-129` 含等到第三个调用的 `calls[2]` 与 `:129` 的“每个调用都以 `/drafts` 结尾”，预检 I3）：目录的读出现时只取草稿的路径（或先答目录的读），草稿的断言不变，另加“能读草稿时恰一个目录的读、访客没有”的格（在新文件）；`C/hooks/store/fake-store-hooks.ts` 追加 `fetchWorkspaceLabels` 的替身（若有测试经它读标签 store）。
- T5（预检 I3：真正渲染个人页、按调用断言的是下面三份；`profile-route.test.tsx` 把个人页换成替身（`:44`），不受影响、不改）：
  - `s2-web-app.spec.ts:128-137` 的 `profileVisits` 加 `...WORKSPACE_LABELS`、`:128` 的注释照实改。
  - `C/components/profile/profile-page.test.tsx`（经路由的页面挂真的 `ProfileIssuesPage`）的三格照 R18 改写，文件不过 400 行：`:253-263`（“…without prefetching directories”）的 `:262` 只数目录之外的调用，另断言恰一个目录的读；`:264-294`（“offers only already held labels…”）改为“选项是目录里成员的未归档项目的标签”（`:293` 的“没有以 `/labels` 结尾的调用”换成恰一个目录的读、没有项目标签的读）；`:296-326`（“…even while the label catalog is cold”）改为“目录答到之前照旧列出，答到之后组名齐全”（`:313` 同理）。这三格钉的正是 R18 要翻过来的行为。
  - `IL/list-data.test.tsx:248-262` 的个人页一行与 `IL/attachment-count.test.tsx:300-311`：按路径取成员列表（`tab.nerve.to(…)` 或只比对工作项的读），断言的意思不变（前者“恰一个成员列表的读、查询是 `relation: assigned`”，后者“不读行”）；`list-data` 的 `archived` 一行不变。
  - 渲染这两页的另两份按路径取调用，不受影响、不改：`IL/properties/row-planning-properties.test.tsx:269-283`、`C/components/issues/workspace-draft/draft-planning.test.tsx:108-134`。
- T6：P6a 的 `C/store/views/{view.store,view-changes,view-busy}.test.ts`：`create(projectId, …)` 改为 `create({ kind: "project", projectId }, …)`（编译跟随，断言不改）；`fake-views.ts` 追加工作区视图的帮手；`C/lib/views-entry.test.ts` 追加工作区两个函数的表。工作区的新格进新文件。
- T7：`VC/{modal,modal-fields,actions-host,quick-actions}.test.tsx`、`VC/{fake-view-pages,fake-view-dialog}.tsx`、`C/components/modals/project-level.tsx` 与 `P/views` 下读这些组件的页面：`projectId=` 改为 `container={{ kind: "project", projectId }}`（编译跟随，断言不改）；`VC/fake-view-pages.tsx`（100）另追加工作区的 `WorkspaceViewPages`：真实形状的两条路由 `:workspaceSlug/workspace-views`、`:workspaceSlug/workspace-views/:globalViewId`（`W/app/routes/core.ts:75-82`），它的 `OwnPage` 按 `globalViewId` 读，项目的 `ViewPages` 不改（预检 I5：照项目的替身读 `viewId` 挂工作区页，单测全绿而产品错）；工作区的新格进新文件（`*-workspace.test.tsx`）。
- T8：`C/hooks/store/use-issues.test.ts:26` 与 `C/hooks/use-issue-layout-store.test.tsx:127` 的拒绝表去掉 `"GLOBAL"`；`use-issue-layout-store.test.tsx:75-116` 的三元式末支 `EIssuesStoreType.DEFAULT` 改为 `EIssuesStoreType.GLOBAL`（别的断言不改）；`C/store/issue/project-view/view-draft.service.test.ts` 随服务 `git mv` 到 `helpers/`、只改导入与构造的第二个参数；`project-view/filter.store.test.ts` 与 `C/store/issue/root.store.ts` 的导入路径；`WC/form.tsx:137` 的 `my_issues` → `workspace_issues`（按 `git grep` 跟随，不是编译跟随：`ISSUE_DISPLAY_FILTERS_BY_PAGE` 是索引签名（`PKG/constants/src/issue/filter.ts:38-40`），漏改照样编译、运行时旧列表页的 “Edit View” 一点就抛；预检 m4；T11 删）；`C/store/issue/fake-issues.ts`（208）在 `listContext`（`:105-112`）旁追加一个工作区的帮手（只追加，需要时）。`profile/filter.store.test.ts`（313）、`page-preferences-filter*.test.ts`、`issue-filters.test.ts`（495）不改（回归）；新格进新文件。
- T9：`WV/layout.test.tsx`（175）随旧布局删去（台账写明旧的每一格由哪一格接上；它对 `WC/quick-action.tsx`、`WC/default-view-quick-action.tsx` 的两个替身 `:29-30` 随之没了，两个文件同一提交删）；`E/stories/smoke/s2-issue-list-removal.spec.ts`（161）整份删（R22，与视图页的重写同一提交）；`VC/use-view-page-read.test.tsx`（151）与 `P/views/(detail)/[viewId]/{page,header}.tsx` 的 `useViewPageRead(slug, id)` 加第一个参数 `EIssuesStoreType.PROJECT_VIEW`（编译跟随，断言不改）。
- T10：`filters-hoc/{save-as,update-view,update-view-busy,update-view-offered}.test.tsx` 与 `fake-view-saving.tsx`：只改拆分后两个 hook 的导入与 HOC 的 props（容器）；断言不改。若有格的表达式不经草稿给出、请求体因此变（spec R10 的来源换了），先停下报告。`base.test.tsx` 不改。
- T11：`C/store/root.store.test.ts:106` 的 `toHaveLength(20)` 改 19，删 `:113` 与导入 `:14`；`C/lib/store-context.test.ts`：删导入 `:11`，`:313-318` 的例子换成 `x.issue`（`const issue = x.issue;`、`issue.rootStore.user.updateCurrentUser(…)`，注释 `:313-314` 照实改），断言不改；`C/services/workspace.service.test.ts`（53）随旧 service 删。`C/components/power-k/config/navigation/commands.test.tsx` 不改（R19）。
- T12：`E/fixtures/mounts.ts` 加 `WORKSPACE_VIEWS`。
- T14：`E/stories/issue/i19-member-issues.spec.ts`：`:336` 一行、`:460` 的标题、`:485-493` 改写，三处的正则（`:336`、`:485`、`:491`）收成一个常量，文件不超过 528 行（spec §2.12）。
- 不改、不受影响（回归）：M4、M6 的列表与筛选测试（四个布局根的、`planning-page-*` 的、`use-page-filter-expression` 的），**例外**是 T5 一条里点名的 `IL/list-data.test.tsx:248-262`、`IL/attachment-count.test.tsx:300-311` 与 `profile-page.test.tsx` 的三格；P6a 的 H12、H20 页面故事；P4、P5 的测试；`group-columns` 的既有格；`e2e` 的 m4、m6 配置里的故事。

**红的窗口**：没有红的提交。产品改动让既有的格或故事变的，都在同一提交里照上面的清单改（`s2-web-app` 在 T4、T5，S2 的移除冒烟在 T9）。名单之外的任何红先停下报告。

**T9 与 T11 之间的窗口**：T9 把 `WV/page.tsx` 原样移到 `WV/(list)/page.tsx`，页头与布局是新的（“Add view” 经新宿主），正文仍是旧组件（读旧 store、打 Plane 的旧地址，同基线上的样子）；T11 重写正文并删旧层。T9 到 T11 之间列表页的已存行仍读不到（基线上就读不到），视图页与页头的切换器是新的。窗口的三处接缝（预检 m5）：① 原样移过去的 `(list)/page.tsx` 读 `currentWorkspace`（`WV/page.tsx:25`、`:28`），T9 不把它收进 `workspace-address-route`（只收 `(list)/{layout,header}.tsx`），T11 重写之后才收；② T9 的视图页用的还是旧的工作区 HOC：它不给状态的选项（`workspace-level.tsx:44-60` 只有成员、项目、标签）、不给另存与更新视图，所以 T9 到 T10 之间工作区页的 `state_id` 条件没有选项可选、没有 “Save as”，T10 补上；③ 列表页的搜索状态由 `(list)/layout.tsx` 提供，`WC/page-state.tsx` 是 T11 的新文件：T9 的 `(list)/layout.tsx` 只有 key 与宿主，T11 的 Files 含它（加页面状态）。

**检查点**（七项用 `sh L/gates.sh <前缀>`，含 `gen-check`、`lint-go`、Go 两路；带页面配置用 `sh L/final.sh <前缀>`：七项之后跑 m7-pages、m6-pages、m6-acceptance、m4-list、m4-filters、m4-detail 六个配置，三种引擎，spec R23；`final.sh` 的六个配置与 P6a 相同，主控在 T1 只改它头部注释里的理由）：
- **0（T7 之后，引擎改动之前）**：`final.sh cp0`；主控推送、核分支 CI；台账记基线：本机第 1 路 bootstrap（P6b 的基线，之后 +30 秒停查）、第 0 路（含 T2 的两个可见性测试）、web 作业的 Lint、Unit tests、Build 三步、e2e 两片，以 `gh` 读 main CI `45778376995` 的同名作业与步作对照；m4-list、m4-filters、m4-detail 这一次是 T8 之前的基线。
- **1（T10 之后：引擎、视图页与另存完成）**：`final.sh cp1`；主控推送、核分支 CI，记 web 作业三步与 e2e 两片。
- **2（T11 之后：删除完成，产品代码完成）**：`final.sh cp2`；推送之前照 P4 spec §2.11（`specs/P4-web-notifications.md:270`）估出 e2e 两片（以最近一次分支 CI 的两片为底，加新文件的 chromium 时长 × CI 与本机之比），超过 30 分钟先报告、不推送；主控推送、核分支 CI。
- **终检（T16）**：`final.sh fin`（同六个配置）。
- **另跑的单配置**（动了别的 M 的共用件，先 `make build build-s3-fixture`，再 `pnpm --filter web exec react-router typegen`；每个配置跑完先把 `e2e/test-results` 拷进台账再跑下一个）：T3 chromium 的接口故事（`t3-api`）；T4 chromium 的 `s2-web-app` 与草稿页的 `i14-drafts`、`f10-draft-files-page`（`t4-s2`、`t4-drafts`）；T5 chromium 的 `s2-web-app` 与 m4-filters（`t5-s2`、`t5-m4-filters`）；T8 chromium 的 m4-list、m4-filters、m4-detail（`t8-m4-list`、`t8-m4-filters`、`t8-m4-detail`，与检查点 0 的同名配置比对）；T9 chromium 的 P6a 视图页故事（`t9-h12`：`h12-project-view-page.spec.ts`、`h20-view-pages.spec.ts`，读 hook 泛化之后）；T10 chromium 的 P6a 另存故事与 m6-pages（`t10-h12-saving`、`t10-m6-pages`）。
- 第 1 路：本机超过 CP0 基线 +30 秒先停下查；CI 的 bootstrap 过期限的八成只报告。本机 Go 两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 的服务端两路为准并记录（负责人的规则）。
- 本机若被 WindowServer 注销（WebKit 的 corpse），同一 HEAD 以新前缀只补跑没跑到的配置（P6a 的先例，spec R23）。

**证据**
- 每次检查经 `run-check.py`（`python3 L/run-check.py <label> <command…>`），变异经 `mutate.py`（每个旧串恰出现一次，工具恢复并核 sha256、打印 `restored=yes`）；标签 `t<N>-…`、`cp<k>-…`，不重用。
- **变异包的源码**（`PKG/*`）：在变异的命令里重建那个包的 dist，跑完之后再重建一次并记 dist 的 sha256 前后（P6a T6 的教训）。**变异 SQL**：改生成的 `SV/adapter/postgres/gen/labels.sql.go` 里的查询字符串（不重跑 sqlc），由 `mutate.py` 还原。
- **RED 先于实现**：先写失败的测试，看它因所测性质失败（不是编译失败），留 RED 收据，再实现到绿。
- 下表每个性质至少一个能编译、能让测试失败的变异。两层都在查同一件事时两层一起改，并留只改一层的对照。只有竞态杀得死的变异不算。
- **页面故事先写一个期望错误的版本**（或以 e2e 变异），看到断言在被改处失败，再改正；收据 `t<N>-red-page-…`。
- **死成员**：T1 的基线（`L/dead-members/dead-before.tsv`，先 typegen），T4、T8、T9、T11 之后整份比对（`t<N>-dead-after`），T15 的终表；新出现的每一行写明去处；P6aR §7 留给 P6b 的 19 行在 T11 之后清零。
- **读者核对**：删任何东西之前 `git grep -n` 核读者（含测试与替身、`K` 的样例），写进台账；删之后再核一次只剩预期的（P6a T4、T10 的写法）。

**测试时间**
- Go：T2 加两个顶层测试 `TestListingWorkspaceLabelsListsEachProjects`（桶 12）、`TestWorkspaceLabelsKeepToTheirWorkspace`（桶 1），都在第 0 路；改名先 `printf %s <name> | cksum` 核桶号（小于 46），写进台账；不往第 1 路加顶层名。
- web 作业期限 20 分钟、960 秒报告线；每个检查点记 Lint、Unit tests、Build 三步；越线时三步对 main CI `45778376995` 的同名步各算比值，三个比值相近是慢 runner，照报告、不停；只有 Unit tests 的比值明显高于另两步才是本段的增长，先报告主控再继续。加大量 vitest 的 Task 记新文件的时长。
- e2e 每片 35 分钟、报告线 30 分钟。T3、T12–T14 记每个新文件每个 `test()` 在三种引擎的时长（JSON 报告）；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

**页面故事的教训**（P5b、P6a 阶段记录 §7、§9，spec R28）
- 每个“在途”都靠扣住回答（`holdAnswer`），不靠时间；不断言墙上的时钟（P6a 整审 m2）。
- 一律放行或拒绝的路由只在 Chromium 装，而且只在目标页的代码（含编辑器、命令面板的代码块）载入之后装：WebKit 在页面载入或动态 import 时有 `page.route` 生效就不用预载、再取一次，约 3 秒后报 “preloaded … not used”。
- 换页、离开、换工作区、改服务端的状态（另一会话经接口删视图、改角色）之前，先等在途的读答完：动作之前登记 `waitForResponse`。
- 一次 `fetchIssue` 是两个 `GET`（M4 的浮层），数读的时候照此。
- swr 2.4.2 的重连只通知一个键上的第一个 hook；swr 迟到的失败无条件写进键（不核对是不是当前的请求，P6a 整审 m1）：每次挂载读自己的读，取数函数接住 `NoViewDraftError`、照常兑现。
- 视图页只在自己的宿主发出删除时不画（`pageLeaving`，P6a 限定复审 n1）；别的“没有视图”一律“不存在”。删除弹窗判“这一页显示的视图”按容器取路由参数（工作区 `globalViewId`），工作区的单测挂真实的路由形状，不借项目的替身（预检 I5）。
- 卸下要丢的偏好（默认视图）不经 SWR 读：同一个键在去重窗里再挂上，取数函数不被调用，页面空白（预检 I2，re-96 I1 的同一类）。
- 只有图标的按钮与只靠占位的输入框经 `t()` 得名（P6a 整审 O5）。
- jsdom 只钉落点的目标，真实焦点以三引擎的故事为准；键盘、指针、窄屏（800 宽）分开看。
- 按名字定位的按钮限在它的区域里；不在重载之后查控制台（M6-M8-09）；不给任何东西起名 “Notifications”。
- 每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去）与 `expectQuietConsole`（不放宽）结束。
- `AlertModalCore` 的默认按钮文字是英文（`web/packages/ui/src/modals/alert-modal.tsx:62-66`）：显式传 `t()`。
- Docker 重启之后 testcontainers 的端口超时：等 ryuk 退出，以新标签重跑（M6-M8-05）。

## 性质与变异

| 性质 | 变异（标签） | Task |
|---|---|---|
| 目录只列成员的项目 | 去掉 `m.member_id` 的条件（`t2-mut-sql-any-project`，改生成的查询，适配器一格与可见性失败） | T2 |
| 已归档的项目不在 | 去掉 `p.archived_at IS NULL`（`t2-mut-sql-archived`） | T2 |
| 已删除的标签不在 | 去掉 `l.deleted_at IS NULL`（`t2-mut-sql-deleted`） | T2 |
| 按项目、再按 `sort_order`、`id` | `ORDER BY l.sort_order, l.id`（`t2-mut-sql-order`） | T2 |
| 三种角色都可读 | 规则去掉访客（`t2-mut-rule-no-guest`，`rules_test.go` 一行与矩阵的访客格失败；只改规则一层的对照即本变异） | T2 |
| 先判工作区与规则 | 用例跳过 `findWorkspaceAndDecide`（`t2-mut-uc-no-decide`） | T2 |
| 回答原样换算 | 处理器丢 `parent_id`（`t2-mut-http-no-parent`） | T2 |
| 失败的读是错误、不是空列表 | 适配器吞掉错误、答空列表（`t2-mut-adapter-swallow`，`failures_test.go` 的新行失败） | T2 |
| 目录按项目给出 | `getProjectLabels` 不回落到目录（`t4-mut-no-fallback`，“目录在时按项目给出”“目录在时按标签排序不读项目标签”两格失败） | T4 |
| 已归档的项目不回落 | 范围判断去掉“未归档”一半（`t4-mut-fallback-archived`） | T4 |
| 不是成员的项目不回落（预检 m6） | 范围判断去掉“是成员”一半（`t4-mut-fallback-non-member`，“`member_role === null` 的项目给 `undefined`”失败） | T4 |
| `labelMap` 含目录 | 只取项目的列表（`t4-mut-map-projects-only`） | T4 |
| 写入两处都写 | 只写项目的列表（`t4-mut-confirm-project-only`，“建一个标签之后目录里有”失败） | T4 |
| 草稿页的门 | 不看 `canRead`（`t4-mut-drafts-gate`，访客格失败） | T4 |
| 个人页的门 | 不看 `allowed`（`t5-mut-profile-gate`） | T5 |
| 个人页挂载时读目录 | 去掉调用（`t5-mut-profile-no-read`，“按标签分组的组名来自目录”失败） | T5 |
| 工作区视图进工作区的列表 | 改删写回 `?? undefined`（`t6-mut-workspace-undefined`）；建不 `prepended`（`t6-mut-create-no-prepend`） | T6 |
| 工作区视图对访客不给 | `getViewById` 不看角色（`t6-mut-workspace-view-guest`） | T6 |
| 新建的队列按容器 | 队列键不带容器（`t6-mut-one-create-queue`，“项目的新建在途时工作区的新建不等”失败） | T6 |
| 入口规则 | `workspaceViewsOn` 收访客（`t6-mut-entry-guest`） | T6 |
| 弹窗按容器跳转与内容 | 一律走项目的地址（`t7-mut-modal-project-path`）；工作区用 `NEW_VIEW_CONTENT`（`t7-mut-content-list`） | T7 |
| 行菜单按容器的角色 | 工作区容器读项目角色（`t7-mut-menu-project-role`，工作区所有者没有“编辑”失败） | T7 |
| 删除之后回容器的列表 | 一律回项目的列表（`t7-mut-delete-project-list`） | T7 |
| 删除弹窗按容器取页面的视图 id（预检 I5） | 工作区容器仍读 `viewId`（`t7-mut-delete-param-viewid`，挂在真实形状的工作区路由上的“在视图自己的页删除：发出之后页面不画、成功之后到 `/{slug}/workspace-views`”失败） | T7 |
| 提示按容器的键 | 一律项目的键（`t7-mut-toast-project-key`） | T7 |
| 路由判为 `GLOBAL` | `globalViewId` → `DEFAULT`（`t8-mut-store-type-default`） | T8 |
| `useIssues`、`useIssuesActions` 给工作区的 | `GLOBAL` 给个人页的 store（`t8-mut-use-issues-profile`）；给个人页的命令（`t8-mut-actions-profile`） | T8 |
| 本页的筛选 | `usePageIssueFilters(GLOBAL)` 给 `undefined`（`t8-mut-page-filters-undefined`，表格读不到筛选） | T8 |
| Display 一律表格的选项 | 按持有的布局取（`t8-mut-display-held-layout`，`list` 视图没有选项） | T8 |
| Display 不给迭代、模块（预检 m1） | `workspace_issues` 的表格属性仍是 14 个键（`t8-mut-display-planning-offered`，`@nerve/constants`，命令里重建 dist；“工作区页的 Display 不给 Cycle、Modules”失败） | T8 |
| 词表与显示的选项 | `ISSUE_STORE_TO_FILTERS_MAP` 去掉 `GLOBAL`（`t8-mut-filters-map-missing`，`@nerve/constants`，命令里重建 dist） | T8 |
| 挂载的键带页面 | `page` 一段为空（`t8-mut-mount-page-empty`，两个默认视图共用一个键） | T8 |
| 挂载的项目集取这一页筛选的 `project_id`（预检 m8） | 项目集恒为空（`t8-mut-mount-projects-ignored`，“筛到一个项目之后，别的项目的角色变化不让这一页重取”失败） | T8 |
| 三种角色都读 | 挂载的 `allowed` 只给管理员成员（`t8-mut-mount-guest-denied`）；`allowsRole` 对工作区臂只给管理员成员（`t8-mut-allows-role-member`） | T8 |
| 行的范围核对 | 工作区臂不核 `project_id`（`t8-mut-read-no-row-check`） | T8 |
| 写进工作区列表 | `covers` 对工作区臂恒假（`t8-mut-covers-false`，“编辑一行，工作区列表跟着变”失败） | T8 |
| 工作区臂的行在不在列表里按自己的词判（预检 I1） | 工作区支不调与项目支共用的三条（落回成员支的判断）（`t8-mut-membership-no-assignee`，“分配给我上改负责人，行离开、总数减一”与“`state_id` 条件下改状态，行离开”两格失败） | T8 |
| 提及、关注的条件走重读 | 工作区支不把 `mention_id`/`subscriber_id` 记“未知”（`t8-mut-membership-subscriber-known`，“我关注的上的确认走重读、不直接插入”失败） | T8 |
| 别的工作区的确认不动这张表 | `paged-groups.ts:332` 仍只认 `"member"`（`t8-mut-foreign-workspace`，“别的工作区、前快照不可得的确认不让这张表重读”失败） | T8 |
| 默认视图自己的词 | 查询里没有（`t8-mut-own-term-missing`）；词表里还有自己的键（`t8-mut-own-key-offered`） | T8 |
| 筛选只在这次进入 | `discard` 留着筛选（`t8-mut-default-filters-persist`） | T8 |
| 显示存浏览器、按工作区分 | `update` 不写存储（`t8-mut-display-not-stored`）；页面 id 不带 slug（`t8-mut-page-id-plain-key`） | T8 |
| 读回的校验 | 读回留着存的布局（`t8-mut-storage-layout-kept`） | T8 |
| `toFilterExpression` 收 `project_id` | 不收（`t8-mut-expression-no-project`，工作区视图的项目条件与个人页的项目条件两格失败；两层：`toWorkspaceFilters` 一起改，留只改一层的对照 `t8-ctl-expression-one-layer`） | T8 |
| 工作区的查询平铺 | `workspaceListQuery` 带 `group_by`（`t8-mut-workspace-query-groups`） | T8 |
| 11 个词 | `toWorkspaceFilters` 丢 `mention_id`（`t8-mut-workspace-no-mention`，API 建的视图的词经“更新视图”保留的一格失败） | T8 |
| 以 `globalViewId` 为 key | 布局不带 key（`t9-mut-no-key`，换视图之后草稿跟过去） | T9 |
| 访客不读已存视图 | 门不看角色（`t9-mut-guest-reads`，“访客没有 `GET /api/v0/views/{id}`”失败） | T9 |
| 项目视图的 id 算不存在 | 不核 `project_id`（`t9-mut-project-view-shown`） | T9 |
| `forbidden` 算不存在 | 走失败与重试（`t9-mut-forbidden-failed`） | T9 |
| “不存在”回列表页（R30） | 动作回到所有工作项（`t9-mut-not-found-all-issues`，“动作回到 `/{slug}/workspace-views`”一格失败） | T9 |
| 默认视图不读视图 | 默认视图也 `fetchView`（`t9-mut-default-reads-view`） | T9 |
| 默认视图的偏好每次挂载读自己的（预检 I2） | `useDefaultViewRead` 换成普通的 `useSessionSWR` 读（`t9-mut-default-read-shared`，真的 SWR，“默认视图卸下立刻再挂上，照常画出，显示是浏览器里的”失败） | T9 |
| 每次挂载读自己的 | 普通的挂载读（`t9-mut-mount-read-shared`，真的 SWR，“卸下立刻再挂上恰再一个 `GET`”失败） | T9 |
| 迟到的 `NoViewDraftError` 兑现 | 去掉 `.catch`（`t9-mut-no-draft-thrown`） | T9 |
| 只在自己的宿主删除时不画 | 不看 `pageLeaving`（`t9-mut-leaving-any`） | T9 |
| 按行的项目判可编辑 | 工作区根不传 `canEditPropertiesBasedOnProject`（`t9-mut-row-editable-route`，整张表只读） | T9 |
| 不给快速新建 | `enableQuickAdd` 为真（`t9-mut-quick-add`） | T9 |
| 空态的按钮照矩阵 | 给访客（`t9-mut-empty-guest`） | T9 |
| 空态的分派（预检 m8） | `IL/empty-states/index.tsx` 去掉 `case GLOBAL`（`t9-mut-empty-case-missing`，落到 `default: return null`，三种空态的格都失败） | T9 |
| 布局根读目录 | 去掉调用（`t9-mut-catalog-not-read`） | T9 |
| 面包屑链到列表页 | 不链（`t9-mut-breadcrumb-no-link`） | T9 |
| 切换器打开时才读已存视图、访客不读 | 挂上就读（`t9-mut-switcher-reads-on-mount`）；访客也读（`t9-mut-switcher-guest-reads`） | T9 |
| 默认视图的菜单只有复制与打开（预检 I4：菜单在 T9 建） | 加“编辑”（`t9-mut-default-menu-edit`） | T9 |
| 默认视图菜单复制的失败有提示 | 只处理成功（`t9-mut-copy-success-only`） | T9 |
| 请求体整个在轮到时从草稿算（O1） | 筛选取点击时的表达式（`t10-mut-filters-at-click`，“扣住前一笔、点击、改草稿、放开：体是新草稿”失败） | T10 |
| 离开之后才轮到的体是离开前的草稿（预检 m3） | 体取页面显示的筛选（`getIssueFilters`，卸下即删）而不是 `confirmed`（`t10-mut-body-from-shown`，“扣住前一笔、点击、卸下页面、放开：体等于卸下之前的草稿”失败） | T10 |
| 另存带默认视图的词、明写表格 | 不带词（`t10-mut-save-as-no-own`）；不改布局（`t10-mut-save-as-layout-kept`） | T10 |
| 另存与更新只给能写的 | 另存给访客（`t10-mut-save-guest`）；所有者被降为访客也给“更新视图”（`t10-mut-update-guest-owner`） | T10 |
| 在途停用 | 工作区根不提供 `isUpdating`（`t10-mut-busy-not-passed`） | T10 |
| 跟进只在挂着时 | 去掉 `mounted` 的检查（`t10-mut-update-unmounted`） | T10 |
| 访客不读已存视图（列表页） | 列表页对访客也读（`t11-mut-list-guest-reads`） | T11 |
| 搜索不分大小写 | 区分（`t11-mut-search-case`） | T11 |
| 默认行一行一个右键菜单（T9 的菜单） | 默认行不接右键菜单（`t11-mut-default-row-no-context-menu`，“右键打开的菜单只有复制与打开”失败） | T11 |
| `old-view-urls` 拦住工作区旧地址 | 在一个 web 文件里放回旧地址（`t11-mut-keyword-old-url`，`node tools/keywords.mjs` 失败） | T11 |

## Tasks

### T1 基线与文档（主控）

- **提交**：spec、plan、预检（含处理）。实施者简报（`L/implementer-brief.md`）与台账一样不进 git，随阶段归档。
- **P6a 的回填**（spec R27；数字取自归档 `M7-P6a-integration-20261126` 的台账 `progress.md` 末几行，主控核过）：`reviews/P6a-web-project-views-review.md` 的状态行（`:3`）、§2 的 A10 一行（`:51`）与 §9 末句（`:247`）补“终检与集成”：终检 `fin` 被第四次 WindowServer 注销中断，七项、build+e2e、m7-pages、m6-pages、m6-acceptance 已过，m4-list、m4-filters、m4-detail 以 `finr` 在同一 HEAD `d7597112` 补跑全过（bootstrap 第 0 路 467.1、第 1 路 629.1 秒）；最终分支 CI `45773943757` attempt 1 全绿（web 1046 秒，越过 960 秒报告线 87.2%，只报告；第 1 路 760.8 秒 = 63.4%）；合并 `6dead17d`（`--no-ff`，父 `47bd860b`、`d7597112`）；main CI `45778376995` attempt 1 全绿（web 741 秒、第 1 路 740.9 秒 = 61.7%，都没到报告线）；归档 `M7-P6a-integration-20261126`。P6a spec 的 A10（`specs/P6a-web-project-views.md:335`）照实勾选。
- **设计与清单**：设计 §14 的 P6a 一行（`D:353`）改为已完成、加 P6b 一行；§12 的 P6b 一行（`D:309`）链 spec；§5 的接口表加 `GET /workspaces/{slug}/labels` 一行（R16，负责人已批）。`docs/v0/frontend-changes.md` §3.1 的 M7 一行（`:206`）补 P6a 的合并。主控改，实现代理不碰设计。
- **台账工具**：`L/final.sh` 的六个页面配置不变（spec R23），头部注释里 P6a 的理由（`:4-9`）照 P6b 的理由改写（m7-pages 收 H13/H14/H20 与 P6a 的视图故事，视图的 store、弹窗、HOC 是共用的；m4-list 兜引擎与表格；m4-filters 兜 I19 的个人页与 `PagePreferencesFilter`；m4-detail 兜浮层与行的命令；m6-pages、m6-acceptance 兜外壳与迭代页、模块页的另存）。
- **依赖**：根目录 `pnpm install --frozen-lockfile`（收据 `t1-install`）。
- **死成员的基线**：先 `pnpm --filter web exec react-router typegen`，再 `node L/dead-members/deadsym.mjs --tsv > L/dead-members/dead-before.tsv`、`node L/dead-members/domains.mjs L/dead-members/dead-before.tsv --rows M7`（收据 `t1-dead-before`）；与 P6a 的终表逐行比对，P6aR §7 留给 P6b 的 19 行列出。
- **检查**：文档链接（`doc-links.py`）与关键词，然后提交 `docs(M7/P6b): …`。

### T2 接口：契约、服务端与矩阵

**Files**：
- 改：`PRJ`（`:597-622` 之后加一个路径）、`api/openapi.yaml`（`:120-121` 旁）、生成物（`make gen`）；`SV/domain/actions.go`（80）；`server/internal/modules/access/domain/rules.go`（265，`:86` 旁）、`rules_test.go`（309，`:37` 旁）；`SV/app/label_ports.go`；`SV/adapter/postgres/queries/labels.sql`、`SV/adapter/postgres/labels.go`（122）、`labels_test.go`（208）、`failures_test.go`（308，允许清单）；`SV/adapter/http/handler.go`（`:118-124`、`:166-167` 旁）、`handler_test.go`（允许清单）；`SV/module.go`（373，`:351` 旁）；`BS/permission_matrix_labels_test.go`（282）。
- 新：`SV/app/list_workspace_labels.go` 与 `_test.go`；`SV/adapter/http/workspace_labels.go` 与 `_test.go`；`BS/workspace_labels_visibility_test.go`。

**Steps**：
- [ ] Step 1：契约先行：加路径与操作（spec §2.1 的描述），`make gen`，`make gen-check`（收据 `t2-gen`、`t2-gen-check`）；生成的处理器接口让 `server/` 编译不过（预期，不提交）。
- [ ] Step 2：先写测试，看到失败（收据 `t2-red`）：用例四格（没有身份、工作区找不到、规则拒绝、成功）；适配器一格（次序、已归档、不是成员、已删除的标签、不在职的成员身份）；`failures_test.go` 的新行（取消的上下文：错误、不是空列表）；处理器两格（换算、拒绝的问题码）；`rules_test.go` 一行；矩阵一行与检查；两个可见性测试（各未归档项目的 `listLabels` 接起来，答 403 的算没有；已归档的项目不接进来，与 states 的先例不同，spec §2.1）。
- [ ] Step 3：实现到通过（spec §2.1；查询照 `sqlc.arg(…)` 写）：动作与 `Actions()`、规则、端口、查询（`make gen` 重生 sqlc）、store、用例、处理器、接线。
- [ ] Step 4：变异照上表 T2 各行（SQL 的改生成的查询字符串）。`make lint-go`、聚焦 `go test ./internal/modules/project/... ./internal/modules/access/...`、bootstrap 里的矩阵与两个可见性测试（`go test ./internal/bootstrap -run 'TestPermissionMatrix|TestListingWorkspaceLabelsListsEachProjects|TestWorkspaceLabelsKeepToTheirWorkspace'`，收据 `t2-bootstrap`）；两个名字的桶号（`printf %s … | cksum`，收据 `t2-buckets`）。提交 `feat(M7/P6b): 工作区标签目录接口 listWorkspaceLabels——调用者是有效成员的未归档项目的标签，按项目、sort_order、id；project 模块的用例、查询、处理器，权限矩阵一行与可见性两格（第 0 路）`。

### T3 接口故事 H13 (API)

**Files**：新 `E/stories/collaboration/h13-workspace-labels.spec.ts`（不过 200 行）；前置条件只经接口（`E/fixtures/api.ts`、`E/fixtures/views.ts` 的建项目、建标签、归档、加成员的帮手；没有的写在本文件里）。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（收据 `t3-red-api`），看到断言在被改处失败。
- [ ] Step 2：写故事（spec §2.1 的覆盖），标题 “H13 (API): …”，确认 m7-pages 不收它、默认配置收它。
- [ ] Step 3：`make build build-s3-fixture`；chromium 聚焦（`t3-api`）、`--repeat-each=3`（`t3-repeat`）；记时长。提交 `test(M7/P6b): H13 的接口故事——工作区标签目录的范围、次序与拒绝`。

### T4 工作区标签目录：service、store、读与草稿页

**Files**：改 `C/services/project/labels.service.ts`（42）、`C/store/label.store.ts`（208）、`C/components/issues/workspace-draft/root.tsx`（64）、`.oxlintrc.json`（swr 名单）、`K`（`workspace-address-route` 收新 hook）、`E/fixtures/mounts.ts`（80）、`E/stories/smoke/s2-web-app.spec.ts`（`:148-151`）与两份草稿页的单测（允许清单）；新 `C/hooks/use-workspace-label-catalog.ts` 与测试、标签 store 的目录格（新文件 `label.store.catalog.test.ts`，`label.store.test.ts` 已 296 行）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t4-red`）：读目录一次、按项目给出（`getProjectLabels`、`labelMap`）；项目自己的列表在时取它；已归档的项目不回落；不是成员而看得到的项目（`member_role === null`）不回落、给 `undefined`（预检 m6）；建、改、移、删之后目录与项目的列表都是 nerve 的回答；换会话之后旧的回答不写；`prepareListCatalog` 在目录在时按标签排序不读项目标签；hook 的键带工作区 id 与 slug、门关着不读；草稿页能读时恰一个目录的读、访客没有。
- [ ] Step 2：实现到通过（spec §2.2）；`s2-web-app` 的草稿一行、`mounts.ts`、两份草稿页单测照允许清单改（同一提交）。
- [ ] Step 3：变异照上表 T4 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`；死成员整份比对（`t4-dead-after`）；`make build build-s3-fixture` 与 typegen 之后 chromium 的 `s2-web-app`（`t4-s2`），并顺手单跑草稿页的两份故事 `i14-drafts.spec.ts`、`f10-draft-files-page.spec.ts`（`t4-drafts`；它们不按整张读表断言，照理不受影响，但否则要到检查点 0 才第一次跑到；预检 o4）。提交 `feat(M7/P6b): 工作区标签目录——标签 store 按工作区持有、按项目给出，写入两处都写；草稿页挂载时读一次（能读草稿时），不进外壳`。

### T5 个人页读目录：按标签分组与标签筛选

**Files**：改 `C/components/profile/profile-issues.tsx`（88）、`IL/group-columns.tsx`（325，只改 `:277` 的注释）、`s2-web-app.spec.ts`（`:128-137`）；允许清单 T5 一条的三份单测：`C/components/profile/profile-page.test.tsx`（326，三格照 R18 改写）、`IL/list-data.test.tsx`（`:248-262`）、`IL/attachment-count.test.tsx`（`:300-311`）；`profile-route.test.tsx` 不改（个人页是替身，`:44`）；新个人页的测试（分组的组名、筛选的选项、门；进新文件）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t5-red`）：管理员、成员进个人页恰一个目录的读、访客没有；按标签分组时组名来自目录（不是 “—”）、空组齐全，只有目录里没有的 id 是 “—”；标签筛选的选项是目录里成员的未归档项目的全部标签。
- [ ] Step 2：实现到通过（spec §2.12）；`s2-web-app` 的个人页三行与允许清单 T5 一条的三份单测照它写的改法改（同一提交；改之前台账逐条写明）。
- [ ] Step 3：变异照上表 T5 各行。门禁同 T4；`make build build-s3-fixture` 与 typegen 之后 chromium 的 `s2-web-app`（`t5-s2`）与 m4-filters（`t5-m4-filters`：I19 照旧过，它的正则匹配不到目录的读）。提交 `feat(M7/P6b): 个人页挂载时读工作区标签目录——按标签分组的组名与标签筛选的选项齐全，只有读之后被删的 id 是“—”`。

### T6 视图的数据层按容器与入口规则

**Files**：改 `C/services/views/views.service.ts`（54，本 Task 只加 `createWorkspaceView`；`listWorkspaceViews` 随第一个读者在 T9）、`C/store/views/view.store.ts`（110）、`view-changes.ts`（134）、`C/lib/views-entry.ts`（33）与 P6a 的三份 store 测试、`fake-views.ts`（71）、`views-entry.test.ts`（61）（允许清单）；新 `C/store/views/view-container.ts` 与工作区的格（新文件 `view.store.workspace.test.ts`）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t6-red`）：工作区视图的建（回答插在 `lists[workspaceId]` 的最前，测试以 `lists.receive` 种下列表）；改、删写进工作区的列表与详情；`getViewById` 对工作区视图要工作区在、角色是管理员或成员（访客、离开的工作区都不给）；项目视图照旧；新建一个容器一个队列；`workspaceViewsOn` 的表（管理员 T、成员 T、访客 F、没有角色 F）。
- [ ] Step 2：实现到通过（spec §2.6、§2.7 的规则）；`create` 的调用处编译跟随（`VC/modal.tsx:101` 与测试）。列的读（`listWorkspaceViews`、`fetchWorkspaceViews`、`getWorkspaceViews`）在 T9、`workspaceViewsCreatableBy` 在 T7，随各自的第一个读者建（Global Constraints 的 knip）。
- [ ] Step 3：变异照上表 T6 各行。门禁同 T4。提交 `feat(M7/P6b): 视图的数据层按容器——工作区视图的列与建，按项目或工作区持有列表，工作区视图只给工作区的管理员与成员；入口规则 workspaceViewsOn`。

### T7 弹窗、宿主、行菜单与删除按容器 → **检查点 0**

**Files**：改 `VC/modal.tsx`（133）、`actions-host.tsx`（102）、`quick-actions.tsx`（106）、`delete-view-modal.tsx`（92）、`view-list-item.tsx`（59）、`view-list-item-action.tsx`（40）、`view-content.ts`（46）、`C/components/modals/project-level.tsx`（67）、`P/views` 下读这些组件的页面与 P6a 的四份组件测试、两个替身（允许清单，编译跟随；`fake-view-pages.tsx` 另追加工作区的真实形状路由）；en、zh-CN（`workspace_views.action.{create,update,delete}.success`，R29）；`K`（两条地址规则的名单）。新 `VC/use-view-rights.ts`、`VC/view-paths.ts` 与工作区的格（新文件 `modal-workspace.test.tsx`、`actions-host-workspace.test.tsx`、`quick-actions-workspace.test.tsx`）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t7-red`）：工作区容器的新建：请求体是名字、说明、图标加 `WORKSPACE_VIEW_CONTENT`（`layout: "spreadsheet"`）、没有 `filters`，成功之后转到 `/{slug}/workspace-views/{id}`、提示 `workspace_views.action.create.success`；编辑只发改了的字段；行菜单：工作区所有者（管理员或成员）有编辑与删除、工作区管理员删别人的、成员删不了别人的、访客没有菜单的编辑与删除，复制与新标签页打开的链接是工作区视图的地址，复制失败有提示；删除（预检 I5）：工作区的格挂在真实形状的路由（`:workspaceSlug/workspace-views/:globalViewId`，`fake-view-pages.tsx` 追加的 `WorkspaceViewPages`）上，在视图自己的页删除：发出之后页面不画、成功之后到 `/{slug}/workspace-views`、被拒之后照常画；在列表页删除不设 `pageLeaving`、不跳转；`pageViewParam` 的表（项目 `viewId`、工作区 `globalViewId`）；项目容器的既有格不改断言全过。
- [ ] Step 2：实现到通过（spec §2.11）；`projectId` 的 props 改容器（编译跟随，逐处记台账）。
- [ ] Step 3：变异照上表 T7 各行。门禁同 T4；oxlint 计数（`t7-oxlint-count`）。提交 `refactor(M7/P6b): 视图的弹窗、宿主、行菜单与删除按容器——项目视图与工作区视图同一套，工作区新建明写表格的显示，删除之后回容器的列表`。
- [ ] Step 4（**检查点 0**）：`final.sh cp0`；主控推送、核分支 CI，台账记基线（Global Constraints）。

### T8 列表引擎的 `GLOBAL` 与工作区的偏好

**Files**：
- 改：`PKG/types/src/issues/issue.ts`（31）；`C/store/issue/root.store.ts`（338，不过 350）；`C/store/issue/helpers/issue-list-read.ts`（163）、`issue-list-binding.ts`（286）、`C/services/issue/issue-list.service.ts`（28）；`C/lib/paged-groups.ts`（445，只加 `ListContext` 的一臂、改 `:332` 一行与 `:329-330` 的注释）、`C/lib/issue-membership.ts`（112，工作区支，预检 I1）；`C/lib/issue-filters.ts`（349，不过 400）；`C/store/issue/helpers/page-preferences-filter.store.ts`（298，只改类型）；`C/store/issue/profile/filter.store.ts`（206，`:27-32`）；`C/store/issue/helpers/issue-filter-helper.store.ts`（324，删 `getFilterConditionBasedOnViews` 与 `TStaticViewTypes` 的导入）；`C/hooks/store/use-issues.ts`（118）、`C/hooks/use-issues-actions.tsx`（353）、`C/hooks/use-issue-layout-store.ts`（43）、`C/hooks/store/use-page-issue-filters.ts`（42）、`C/components/issues/use-page-display.ts`（62）、`C/hooks/use-issue-list-mount.ts`（68）；`PKG/constants/src/issue/filter.ts`（311）、`common.ts`（294）、`workspace.ts`（110）；`IL/spreadsheet/base-spreadsheet-root.tsx`（149，只改联合）；`C/store/issue/project-view/filter.store.ts`（36，导入）；钉 `GLOBAL` 的两份测试（三处）、`WC/form.tsx:137`（按 `git grep` 跟随）与需要时 `C/store/issue/fake-issues.ts`（只追加）（允许清单）。
- 移：`C/store/issue/project-view/view-draft.service.ts` 与 `.test.ts` → `C/store/issue/helpers/`（`git mv`，参数化，spec §2.4）。
- 新：`C/store/issue/workspace/{issue.store,filter.store,default-view-preferences,page-id}.ts` 与测试；`C/hooks/use-global-issue-actions.ts`、`C/hooks/use-global-preferences-follow.ts` 与测试；`issue-filters-workspace.test.ts`（新格，`issue-filters.test.ts` 已 495 行）。

**Steps**：
- [ ] Step 0：台账写方案（P6a T8 的先例：L! 最集中的一块，先写再做）：分派点的清单与“不接”的理由（spec §2.3），页面 id，两种偏好服务的接口，`ListContext` 与 `issueMembership` 的工作区支；提交的分界（预检 o6：建议引擎的臂、成员判断与换算一份，偏好与分派点一份；分两个提交时每个提交 knip、测试都要绿，不行就一个提交），整审的范围照此分。
- [ ] Step 1：先写测试，看到失败（收据 `t8-red`）：
  - 范围臂：`allowsRole` 三种角色都过、项目的已归档列表与成员列表照旧；`readPage` 调工作区接口、行的范围核对；`covers` 与身份同成员臂；`listWorkspace` 的路径与查询。
  - 行在不在列表里（预检 I1，`issueMembership` 的工作区支与 `applyGroupChange`）：“分配给我”上把一行的负责人改成别人，确认之后行离开、总数减一；`state_id` 条件下改状态同理；在“分配给我”上新建一个没有分给自己的工作项，不插进列表；“我关注的”上的确认走重读（`verifyIds`）、不直接插入；别的工作区、前快照不可得的确认不动这张表、不重读；工作区臂读 `query.group_by` 得 `undefined`（平铺）。
  - 词与换算：`toWorkspaceFilters` 收 11 个词、丢 `cycle_id`/`module_id`；`workspaceListQuery` 平铺；`toFilterExpression` 收 `project_id`；个人页的项目条件照旧（既有格）。
  - 偏好：默认视图 `get` 给浏览器的显示与空筛选；`update` 的显示写浏览器、筛选只在内存；`discard` 之后再 `get` 没有筛选、显示还在；两个工作区的同一个默认视图各存各的；坏的存储值回到默认、布局一律表格；已存视图的草稿照 P6a（`ViewDraftService<WorkspacePageFilters>` 的既有格只改导入）。
  - store 与查询：`GlobalIssues.fetchIssues` 的查询是页面的筛选加自己的词（“分配给我”有 `assignee_id=me`，词表里没有 `assignee_id`）、带 `limit`、没有分组参数；`isPreferenceQueryCurrent`。
  - 分派点各一格（spec §2.3 的 1–8、10、11；第 9 点空态在 T9）：路由判 `GLOBAL`；`useIssues`、`useIssuesActions` 给工作区的；本页的筛选；Display 对 `list` 视图给表格的选项、不给 Cycle 与 Modules（预检 m1）；`ISSUE_STORE_TO_FILTERS_MAP`；`TCreateModalStoreTypes`（类型，编译即证）；表格根的联合；挂载的键带 `globalViewId`、访客也读、项目集取筛选的 `project_id`（筛到一个项目之后，别的项目的角色变化不让这一页重取，预检 m8）；在工作区页上编辑一行（`ProjectIssueQuickActions` → `PROJECT`），工作区列表里那一行跟着变。
  - 命令写全：八个成员都在、`addIssuesToView` 与 `dropIssue` 不在；`updateFilters` 不看给的 id、写路由的页面、跟进重取。
- [ ] Step 2：实现到通过（spec §2.3–§2.5）；两个 `switch` 穷尽，故意漏一个值编译失败（收据 `t8-exhaustive-probe`：在临时副本上删一个 `case`、`tsc` 报错，不提交）；钉 `GLOBAL` 的两份测试（三处）照允许清单改；`root.store.ts` 不过 350、`use-issues-actions.tsx` 只多一个导入、一行调用、一个 `case`、`paged-groups.ts` 只多一行（`t8-wc`）；读者表（`t8-readers`：`getFilterConditionBasedOnViews` 没有调用者；改名之后 `my_issues` 零处，`workspace_issues` 的读者是 `WC/form.tsx:137` 与 `ISSUE_STORE_TO_FILTERS_MAP`——索引签名，编译查不出，只能这样核，预检 m4；`TStaticViewTypes` 只剩 `WC/default-view-quick-action.tsx:14`、`:22`，T9 删）。
- [ ] Step 3：变异照上表 T8 各行（`@nerve/constants`、`@nerve/types` 的在命令里重建 dist，跑完再重建并记 sha256）。门禁同 T4；`pnpm --filter @nerve/constants test`、`@nerve/types`（若有）；死成员整份比对（`t8-dead-after`）；`make build build-s3-fixture` 与 typegen 之后 chromium 的 m4-list、m4-filters、m4-detail（`t8-m4-list`、`t8-m4-filters`、`t8-m4-detail`），与检查点 0 的同名配置比对。提交 `feat(M7/P6b): 列表引擎加 GLOBAL 与工作区范围——三种角色读、平铺、默认视图自己的词固定在查询里；工作区页的偏好：默认视图的显示存浏览器、筛选只在本次进入，已存视图用泛化的草稿；契约的 11 个词，toFilterExpression 收 project_id、去掉个人页的绕行`。

### T9 工作区视图页 → 删去 S2 的移除冒烟

**Files**：
- 改：`W/app/routes/core.ts`（299，`:75-82` 拆成两个布局）；`C/services/views/views.service.ts` 与 `C/store/views/view.store.ts`（`listWorkspaceViews`、`fetchWorkspaceViews`、`getWorkspaceViews`，读者是页头的切换器；格进 `view.store.workspace.test.ts`）；`VC/use-view-page-read.ts`（102，加 store 类型参数与 `useShownWorkspaceView`）与它的测试、P6a 视图页的两个调用处（允许清单）；`IL/empty-states/index.tsx`（38）；en、zh-CN（`workspace_views.not_found.{title,description,action}`，R29）；`.oxlintrc.json`；`K`（两个 `copy-hook` 例外 `K:3424-3437` 删；`workspace-address-route` 收 T9 的新文件，不收 `(list)/page.tsx`）；`PKG/types/src/workspace-views.ts`（43，只删 `:41-43` 的 `STATIC_VIEW_TYPES`/`TStaticViewTypes`）；`W/package.json`（oxlint 上限，若删掉的文件带告警）。
- 移与新：`WV/page.tsx` → `WV/(list)/page.tsx`（原样，`git mv`）；新 `WV/(list)/{layout,header}.tsx`、`WV/(detail)/layout.tsx`、`WV/(detail)/[globalViewId]/{page,header}.tsx`（`WV/[globalViewId]/page.tsx` `git mv` 过来重写）；新 `IL/roots/workspace-view-layout-root.tsx`、`IL/spreadsheet/roots/workspace-root.tsx`、`IL/empty-states/global-view.tsx`、`WC/default-view-menu.tsx`（页头是第一个读者，预检 I4）、`WC/use-default-view-read.ts`（预检 I2）与视图页、页头、空态、菜单、默认视图的读的测试。`(list)/layout.tsx` 这一步只有 key 与宿主（页面状态在 T11）。
- 删：`WV/layout.tsx`（57）、`WV/header.tsx`（113）、`WV/layout.test.tsx`（175）、`E/stories/smoke/s2-issue-list-removal.spec.ts`（161）（允许清单）；`WC/quick-action.tsx`（107）、`WC/default-view-quick-action.tsx`（103）：唯一的导入者是同一提交删掉的 `WV/header.tsx:19`、`:21`（另有 `WV/layout.test.tsx:29-30` 的替身），留到 T11 则 T9 的 knip 报未使用的文件（预检 I4）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t9-red`）：
  - 以 `globalViewId` 为 key（换视图之后上一个的草稿与筛选不跟过去）；默认视图不发 `GET /api/v0/views/{id}`、卸下即丢筛选；默认视图卸下立刻再挂上，照常画出，显示是浏览器里的（预检 I2；这一格挂真的 `useSessionSWR` 与真的 SWR、带 cache provider，`t9-mut-default-read-shared` 才有去重窗可撞）；已存视图的四种状态（加载、`navigation.view_not_found` 与 `forbidden` 的不存在、项目视图的 id 的不存在、别的失败与重试）与各自的页头面包屑；“不存在”的动作以 “Go to views”（`workspace_views.not_found.action`）为名、回到列表页 `/{slug}/workspace-views`（R30）；访客打开已存视图的地址不发请求、说“不存在”；真的 `useViewPageRead` 与真的 SWR：卸下立刻再挂上恰再一个 `GET`；迟到的 `NoViewDraftError` 不让此刻挂着的页面空白；只在自己的宿主删除时不画（挂在真实的路由形状 `:workspaceSlug/workspace-views/:globalViewId` 上、经真的页头删除，预检 I5）。
  - 表格根：按行的项目判可编辑（访客的行只读、成员的行可编辑）、没有快速新建、没有迭代与模块两列；Display 一律表格的选项；`list` 视图照表格画。
  - 页头：面包屑 “Views” 链到列表页；切换器打开时才读已存视图（访客只列默认四个、不读）；“Add view” 照矩阵；默认视图的快捷操作（`WC/default-view-menu.tsx`）只有复制与打开、复制成功与失败各自提示、菜单按钮以 `t("common.item_options", { name })` 为名；页面标题。
  - 空态的三种（筛到空、没有项目、新建工作项）与按钮照矩阵（`IL/empty-states/index.tsx` 的 `case GLOBAL`，spec §2.3 第 9 点）；布局根读目录（三种角色）。
- [ ] Step 2：读者表（`t9-readers-before`：`WC/quick-action.tsx`、`WC/default-view-quick-action.tsx`、`TStaticViewTypes`、`STATIC_VIEW_TYPES`，只剩 `WV/header.tsx`、`WV/layout.test.tsx` 与 `K` 的两个例外）。实现到通过（spec §2.8）；`WV/layout.test.tsx` 与 S2 的移除冒烟照允许清单删去（同一提交；旧格 → 新格写进台账）；两个旧组件、`workspace-views.ts:41-43` 与两个 `copy-hook` 例外同一提交删，跑 `node tools/keywords.mjs`；读者表再跑一次（`t9-readers-after`：没有读者）；新取数文件收入 swr 名单，读工作区的收入 `workspace-address-route`（`(list)/page.tsx` 不收，窗口的 ①）。
- [ ] Step 3：变异照上表 T9 各行。门禁同 T4；`pnpm --filter @nerve/types test`（若有）；oxlint 计数（`t9-oxlint-count`，这一步删五个旧文件）；死成员整份比对（`t9-dead-after`：P6a 留下的 19 行里 `WV/layout.test.tsx:24`、`:26` 与 `WC/default-view-quick-action.tsx:23` 三行随文件走掉，写明去处）；`make build build-s3-fixture` 与 typegen 之后 chromium 的 P6a 视图页故事（`t9-h12`）与默认配置的冒烟（`t9-smoke`：`e2e/stories/smoke/`）。提交 `feat(M7/P6b): 工作区视图页——以 globalViewId 为 key，默认视图不读视图、已存视图四种状态（访客与项目视图算不存在），只渲染表格、按行的项目判可编辑，Plane 的页头与空态，默认视图的偏好每次挂载读自己的；删去 S2 的移除冒烟、两个只被旧页头读的旧快捷操作组件与它们的 copy-hook 例外、TStaticViewTypes`。

### T10 另存与更新视图 → **检查点 1**

**Files**：改 `filters-hoc/use-project-view-saving.ts`（170）、`workspace-level.tsx`（208）、`shared.ts`（33，删无读者的覆盖类型）、`IL/roots/workspace-view-layout-root.tsx`（T9 新建；传 `enableSaveView`、`enableUpdateView`、`isUpdating`）、`WV/(detail)/[globalViewId]/header.tsx`（在途停用）与 HOC 的四份测试、`fake-view-saving.tsx`（允许清单）；新 `filters-hoc/use-view-update.ts`、`use-workspace-view-saving.ts` 与测试（新文件 `update-view-workspace.test.tsx`、`save-as-workspace.test.tsx`、`update-view-body.test.tsx`）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t10-red`）：
  - O1（R10）：扣住同一视图前一笔改（编辑弹窗的 `PATCH`）、点“更新视图”、经 `updateFilterExpression` 改草稿、放开：发出的请求体的筛选与显示是新草稿；成功之后基线与当前表达式都是回答。项目视图与工作区视图各一格。改草稿经 `updateFilterExpression`（不经停用的控件），保留这个写法。
  - O1 的两句边界（预检 m3，spec §2.9）：扣住前一笔、点击、卸下页面（`discard`）、放开：体等于卸下之前的草稿（`confirmed` 仍给：基类的 `values` 不随 `discard` 清掉，这一格同时钉住它在这条路上不为空）、成功不提示（HOC 不挂着）；`confirmed` 为空时体抛错、不发请求、`failed` 照失败处理（走不到的一支，代码照写，说明理由写进注释）。
  - 工作区另存：五种工作区页给（“分配给我”先加一个条件），体带自己的词与 `layout: "spreadsheet"`、名字为空，成功之后转到新视图；访客不给。
  - 工作区“更新视图”：所有者（管理员或成员）给、被降为访客的所有者不给、成员看别人的不给；成功的提示键 `workspace_views.action.update.success`；失败按码提示、按钮留着；离开页面之后答到不提示；在途时筛选行、筛选开关、Display、表格列头、空态的清除都停用，落定之后恢复；成功之后焦点落在添加筛选。
  - 个人页的 HOC 照旧不给保存；P6a 的四份 HOC 测试不改断言全过。
- [ ] Step 2：实现到通过（spec §2.9）；`shared.ts` 的覆盖类型核读者（`t10-shared-readers`），没有读者的删；工作区 HOC 的旧分支删去（`useGlobalView`、`IWorkspaceView`、`EViewAccess` 在 HOC 里的读者随之没了，写进读者表）。
- [ ] Step 3：变异照上表 T10 各行。门禁同 T4；`make build build-s3-fixture` 与 typegen 之后 chromium 的 P6a 另存故事（`t10-h12-saving`：`h12-project-view-saving-page.spec.ts`、`h12-project-view-page.spec.ts`）与 m6-pages（`t10-m6-pages`）。提交 `feat(M7/P6b): 另存与更新视图按容器——更新的一半共用、请求体整个在轮到时从草稿算（O1）；五种工作区页给另存，默认视图另存带它的词、明写表格；工作区 HOC 去掉 Plane 的分支，P4 R17 的工作区一半关闭`。
- [ ] Step 4（**检查点 1**）：`final.sh cp1`；主控推送、核分支 CI，记 web 作业三步与 e2e 两片。

### T11 列表页与删除旧层、`old-view-urls` → **检查点 2**

**Files**：
- 改：`WV/(list)/page.tsx`（T9 移来，59，重写正文；重写之后才收进 `workspace-address-route`）、`WV/(list)/layout.tsx`（T9 新建，加 `WC/page-state.tsx` 的页面状态，预检 m5）、`C/store/root.store.ts`（120，去掉 `globalView`）、`root.store.test.ts`、`C/lib/store-context.test.ts`（允许清单）、`PKG/types/src/views.ts`（14，删 `EViewAccess`）、`view-props.ts`（181，删 `IWorkspaceViewProps`）、`PKG/types/src/index.ts`（桶导出）、`K`（`old-view-urls`、两条地址规则的名单与样例；两个 `copy-hook` 例外已在 T9 删）、`.oxlintrc.json`、`PKG/types/src/workspace-views.ts`（T9 之后剩的 `IWorkspaceView`，整份删）。
- 新：`WC/{page-state,default-view-row}.tsx` 与列表页的测试（默认行用 T9 的 `WC/default-view-menu.tsx`）。
- 删：`WC/{modal,form,delete-view-modal,view-list-item,default-view-list-item,views-list}.tsx`（110、177、65、82、36、44；`quick-action.tsx`、`default-view-quick-action.tsx` 已在 T9 删）；`C/store/global-view.store.ts`（179）、`C/hooks/store/use-global-view.ts`（17）、`C/services/workspace.service.ts`（43）与 `workspace.service.test.ts`（53）；`PKG/types/src/workspace-views.ts`（T9 之后约 40 行）。

**Steps**：
- [ ] Step 1：读者表（收据 `t11-readers-before`）：spec §2.13 的每一项 `git grep`，写进台账；还有 T11 之外的读者先停下报告。
- [ ] Step 2：先写测试，看到失败（收据 `t11-red`）：列表页的状态（默认四行；已存行的加载、失败与重试、没有匹配）；按工作区的搜索（换工作区即丢、不分大小写、默认行按译文）；访客只见默认四行、不读；默认行用 T9 的菜单（只有复制与打开）、菜单按钮以名字为名、一行一个右键菜单；已存行的菜单照矩阵（T7 的行菜单）；“Add view” 照矩阵。
- [ ] Step 3：实现到通过（spec §2.10）；删旧层（§2.13 的表）；`RootStore` 只剩 `views`，`root.store.test.ts` 19、`store-context.test.ts` 换例子；读者表再跑一次（`t11-readers-after`：没有读者）；`old-view-urls` 扩（hit、miss、文件样例写全），跑 `node tools/keywords.mjs`；`git grep -i iframe -- web/apps/web/core web/apps/web/app`（收据 `t11-iframe`，写一句）。
- [ ] Step 4：变异照上表 T11 各行。门禁同 T4；`pnpm --filter @nerve/types test`（若有）、`@nerve/constants`；死成员整份比对（`t11-dead-after`，19 行逐行写去处：三行已在 T9 走掉，其余 16 行在这里）；oxlint 计数（`t11-oxlint-count`）。提交分两个：`feat(M7/P6b): 工作区视图列表页——默认四行与已存视图、按工作区的搜索，访客只见默认四行、不读；行与菜单按容器` 与 `refactor(M7/P6b): 删除 Plane 的旧工作区视图层——store、service、hook、类型与旧组件，RootStore 少一个 store；old-view-urls 扩到工作区旧地址`。
- [ ] Step 5（**检查点 2**）：`final.sh cp2`；推送之前照 P4 spec §2.11 估出 e2e 两片；主控推送、核分支 CI。

### T12 H13 (page)

**Files**：新 `E/fixtures/workspace-view-pages.ts`（约 180 行）、`E/stories/collaboration/h13-workspace-list-page.spec.ts`、`h13-workspace-view-page.spec.ts`；改 `E/fixtures/mounts.ts`（加 `WORKSPACE_VIEWS`，允许清单）。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（收据 `t12-red-page-list`、`t12-red-page-view`），看到断言在被改处失败。
- [ ] Step 2：写故事（spec §2.15 的表）。读表照实数（视图页恰一个 `GET /api/v0/views/{id}`、一个目录的读；回答整理过筛选时“更新视图”成功之后多一次列表的读）；“更新视图”的一步扣住 `PATCH` 断言停用、放开之后恢复（成功一次、被拒一次）；`list` 视图的一步断言请求体的 `layout` 仍是 `list`；读失败的一步只在 Chromium、目标页代码载入之后装路由（视图读与列表读各一步，预检 m7；总数不含不是成员与已归档项目的工作项由接口故事 `h13-workspace-list.spec.ts:101` 承担，页面只核行）；默认视图经切换器从“所有工作项”到“分配给我”再立刻回来，断言列表在、显示是浏览器里的（预检 I2）；“行的属性可编辑”一步放在“分配给我”上：把一行的负责人改成别人，确认之后行离开（预检 I1）；Display 没有 Cycle、Modules（预检 m1）。
- [ ] Step 3：跑法：`make build build-s3-fixture` 与 typegen；chromium 聚焦；`--config=playwright.m7-pages.config.ts` 三引擎（拷出 test-results）；新文件 chromium `--repeat-each=3`（`t12-repeat`）。记每个 `test()` 三引擎的时长；每份不过 400 行。提交 `test(M7/P6b): H13 的页面故事——四个默认视图的跨项目列表、偏好与筛选的寿命、标签目录，已存视图页的更新、另存与不存在`。

### T13 H14 (page)

**Files**：新 `E/stories/collaboration/h14-workspace-views-list-page.spec.ts`。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（`t13-red-page`）。
- [ ] Step 2：写故事（spec §2.15 的表）：“Add view” 的请求体精确（`WORKSPACE_VIEW_CONTENT`、没有 `filters`）、`viewRow` 的 `project_id` 为空；三种角色的列表页；改名只发 `name`；删除照矩阵；在视图自己的页头删除之后转到列表页、不闪“不存在”；复制成功与被拒；右键菜单。
- [ ] Step 3：同 T12 的跑法与时长记录（`t13-repeat`）。提交 `test(M7/P6b): H14 的页面故事——工作区视图列表页的矩阵、新建、改名、删除与复制`。

### T14 H20 (page) 与 I19 的改写

**Files**：新 `E/stories/collaboration/h20-workspace-view-pages.spec.ts`；改 `E/stories/issue/i19-member-issues.spec.ts`（`:336`、`:460`、`:485-493`，允许清单）。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（`t14-red-page`、`t14-red-i19`：I19 先断言“没有目录的读”，看到它在 `:336` 处失败）。
- [ ] Step 2：写故事（spec §2.15 的表）：迟到的回答（已存视图列表的读、视图的读、“更新视图”、删除、换账号；视图的读那一步之后回到原视图又一个 `GET`）；后退前进；键盘（弹窗的 Tab 次序、嵌套 Escape、切换器、行菜单、以键盘“更新视图”之后的焦点，三引擎记焦点）；“Display” 的名字、`aria-expanded`、Escape 回焦；800 宽一次。I19 两处照 spec §2.12 改写，不变长。
- [ ] Step 3：同 T12 的跑法；I19 另以 `--config=playwright.m4-filters.config.ts` 三引擎跑（`t14-m4-filters`）；`t14-repeat`。提交 `test(M7/P6b): H20 的工作区视图页——迟到的回答不写进新页面、后退前进、弹窗切换器与菜单的键盘；I19 改写为个人页读一次工作区标签目录`。

### T15 文档与交接（主控）

- **plane-diff**：spec §7 的十一行进 `docs/v0/plane-diff.md` 行为差异一节（与 M7/P6a 的行同处）；Plane 的行号对照主目录的只读 `plane/` 逐一核过。
- **frontend-changes**：3.1 的 M7 一行写 P6b；3.2 加行：工作区标签目录、`GLOBAL` 与工作区范围、工作区页的两种偏好、视图的数据层按容器、视图页、另存与更新视图（O1）、列表页、删除。
- **交接**：spec §5 的表，每条写“处理结果（M7/P6b）”；M8 的候选写进阶段记录 §7（M7 收官时进 M8 的交接），其中有同名状态的项目标识（R31）与状态 store 回落的归档守护（R32）。
- **死成员**：重跑脚本（`t15-dead-final`），剩下的行写进阶段记录。
- **告警**：核对本段改到与新建的文件 oxlint 为零、计数等于上限（`t15-oxlint-files`）。
- **设计**：§14 的 P6b 一行（主控）；§5 的接口表已在 T1。
- **阶段记录** `reviews/P6b-web-workspace-views-review.md`：§1 方法与范围、§2 验收 → Task、§3 实施中的裁定（十一条 L! 写明，R10 含 O1 改 P6a 项目视图“更新视图”的来源；R16、R8 是负责人的；R29–R32）、§4 向负责人说明（P6b 之后 main 的样子；R8 接口默认与页面的不一致；R10 选“轮到时算”的理由）、§5 门禁与 CI、§6 时长、§7 交给 P7 与 M8、§8 文档、§9 审查与集成。
- 检查文档链接与关键词，提交 `docs(M7/P6b): …`。

### T16 门禁、整审与集成（主控）

1. **整分支审查**：一次 fresh 只读的整分支 deep-review，重点是 Review Focus 的 1–8。Critical/Important 在一次修复轮里各有 RED/GREEN；Minor 记台账，主控裁定改否。之后一次限定复审。
2. **终检**：`final.sh fin`（六个页面配置）。
3. **推送前的估算**：照 P4 spec §2.11 估出 e2e 时长。
4. **最终分支 CI**：推送准确的 HEAD（`git rev-parse`），核分支 CI（`wait-run.sh`；排队 0 个作业约 20 分钟即停下报告）。
5. **合并**：`--no-ff` 合入 main，推送后以 `ls-remote` 核对。
6. **main CI**：核准确的 main CI。
7. **归档与清理**：归档 `.superpowers/sdd/M7-P6b-integration-<日期>`（`archive.py`），清理 worktree 与本地分支，远端分支核过已含于 main 之后删。
8. **回填**：spec A12 与设计 §14 由下一段（P7）的 T1 回填。

## Review Focus

1. **R1 引擎（L!）**：`GLOBAL` 在每个分派点（spec §2.3 的 1–11，静默的几处各有格与变异）；“不接”的五处各有理由、确实没有读者；范围臂的 `allowsRole`（项目的已归档与成员列表不变）、`readPage` 的行核对、`covers`、身份；`ListContext` 的工作区臂（`group_by?: never`）与 `issueMembership` 的工作区支（自己的词、状态、负责人、创建者参与，提及与关注记“未知”，没有 `relation`），`paged-groups.ts:332` 的一行（预检 I1）；命令写全、不给的两个；两个 `switch` 穷尽；`GlobalIssues` 不借 `PagePreferencesIssues`；`root.store.ts` 只加构造、不过 350；项目页、迭代页、模块页、项目视图页、个人页不回归（m4-list、m4-filters、m4-detail、m6-pages、m7-pages）。
2. **R2、R5 偏好与词（L!）**：`PagePreferencesFilter` 只改类型；显示的换算没有被导出、没有第二份；`toFilterExpression` 收 `project_id` 之后个人页不变；11 个词；默认视图的浏览器存储按（账户、页面 id）、读回校验、不复用按控件词校验的旧存储；页面 id 带 slug；默认视图的偏好每次挂载读自己的、不经 SWR（预检 I2）；`workspace_issues` 的 Display 属性去掉 `cycle`、`modules`（m1），`my_issues` 的读者按 `git grep` 跟随（m4）。
3. **R10 另存与更新视图（L!）**：请求体整个在轮到时从草稿算、点击的表达式不用（O1 的选法，主控确认为 L!：P6a 项目视图的“更新视图”一并改为同一来源）；O1 的格与 `t10-mut-filters-at-click`；两句边界（`confirmed` 为空不发、照失败处理；离开之后轮到的体是离开前的草稿，预检 m3）；P6a 的“更新视图”不回归、它的单测不改断言；在途停用照 P6a R29；默认视图另存带它的词、明写表格；工作区 HOC 没有 Plane 的残留；`shared.ts` 的读者核过。
4. **R9、R14、R15 数据层与组件按容器（L!）**：一个实体一个 store；`getViewById` 的可见按种类；改删写进工作区的列表；弹窗、宿主、行菜单、删除对项目一侧不改行为（P6a 的格不改断言）；按容器的文案键写全；`useCopyLink`；删除弹窗按容器取页面的视图 id（`pageViewParam`，工作区 `globalViewId`），工作区的格挂在真实的路由形状上（预检 I5）。
5. **R17、R18、R32 标签目录与个人页（L!）**：照状态 store 的形状（R32）；已归档的项目不回落；写入两处都写；三页的门；外壳不读；S2 与草稿页、个人页单测的改法在允许清单里；I19 两处改写不变长、断言是真的（重载前后各一个目录的读）。
6. **R7、R8 表格与布局（L!、负责人）**：工作区根只传既有的接缝；不给快速新建；`list` 视图照表格画、原样存回、没有写 `layout` 的入口；Display 一律表格的选项。
7. **R20 删除（L!）**：spec §2.13 的表逐项（`WC/quick-action.tsx`、`WC/default-view-quick-action.tsx`、`TStaticViewTypes` 与两个 `copy-hook` 例外在 T9，预检 I4）；读者表；`common/applied-filters` 不删的理由；`old-view-urls` 的样例；`workspace-address-route` 的名单；swr 名单；死成员 19 行清零；告警等于上限；两份会话测试只换例子。
8. **会话分代、读与矩阵（R11、R12、R29、R30）**：“不存在”回列表页、文字 “Go to views”；新键只有六个、写全；读的键带 `loginId`；已存视图的读每次挂载读自己的、迟到的 `NoViewDraftError` 兑现；以 `globalViewId`、`workspaceSlug` 为 key 的布局；访客不发已存视图的读；spec §2.7 的每一格；弹窗与菜单的可访问；每个失败都提示、不 `console.error`；名字里没有 “sentry”。
9. **R16 接口（负责人）**：契约与 `listWorkspaceStates` 同构；查询的范围与次序；规则三种角色；矩阵与可见性；两个顶层测试在第 0 路；接口故事。
10. **共用的外壳**：M4 的列表与浮层、M6 的表驱动的根、P6a 的视图组件与 HOC、M3 的标签 store 与 S2；六个页面配置与整套默认 e2e。
