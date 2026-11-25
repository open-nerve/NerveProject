---
status: open
from: M6/closeout
to: M7
created: 2026-11-10
---

# M6 交给 M7：收集箱新建弹窗在编辑器建成之前拒绝关闭、迭代与模块的收藏、工作区级的迭代与模块、视图的 store 与 M3 的视图遗留、焦点索引的最后删除、转移不发布 issue.updated、访客看到的视图与收集箱命令、首个投递任务的 CLI、迭代与模块成为资产宿主的条件

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页问题已经解决。M6 进行中发现的条目随发现写入（M6-M7-01）；其余由 M6/P8 补齐（M6-M7-02 起，[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.13、§3），各条写明出处。

<a id="M6-M7-01"></a>
## M6-M7-01 收集箱新建弹窗在编辑器建成之前拒绝关闭

- M6/P4 查实 M4 新建工作项弹窗的 C12 间歇失败：弹窗刚打开时描述编辑器尚未建成，编辑器的引用却已挂上一套操作空编辑器的 API，`isEditorReadyToDiscard()` 答 false，弹窗把 Escape 当成“编辑器忙”而拒绝关闭。修复分支 `codex/m6-p4-ci` 改为编辑器建成之前引用为 null（`web/packages/editor/src/hooks/use-editor.ts`），“引用存在”即“编辑器存在”；经过见 [P4 阶段记录](../../M6-cycles-modules/reviews/P4-progress-transfer-review.md) §5 与 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)。
- 收集箱的新建弹窗（`web/apps/web/core/components/inbox/modals/create-modal/create-root.tsx`，沿用 Plane）有同类问题，这次修正前后行为相同：Escape（`handleEscKeyDown`）与 Cancel 按钮只在 `descriptionEditorRef.current?.isEditorReadyToDiscard()` 为真时关闭，否则提示 “Editor is still processing changes” 并留在原处。编辑器还在加载或尚未建成时引用为 null，空表单也关不掉。
- 处理方向：与 `issue-modal/draft-issue-layout.tsx` 的 `requestClose` 同一条规则——引用为 null（编辑器未建成）时可以关闭，只有已建成且有未完成的图片时才拒绝；提交仍要求编辑器就绪。M6 不改这个页面，因为收集箱的接通属于 M7。

**何时验收：** 收集箱页面首次接通的阶段。弹窗一打开就按 Escape 或 Cancel 能关闭空表单（可仿 M4/P14 的 C10 编辑器加载故事扣住编辑器代码块），有未完成图片时仍拒绝。

**处理结果（M7/P5b）：** 收集箱新建弹窗的关闭照 `draft-issue-layout.tsx` 的 `requestClose` 去掉草稿改写（[P5b spec](../specs/P5b-web-intake-files-embed.md) §2.1、R4、R28；`fa78bb11`）：Escape、点背景与 Discard 走同一条规则 `requestClose`（`create-modal/create-root.tsx`）——编辑器的引用为 `null`（还没建成）时可以关；已建成而 `isEditorReadyToDiscard()` 为假时提示、不关（这个表单不收图片，按代码建成之后恒为真，规则照 M4 留着，单测以替身的引用造出它）；有改动先确认放弃；提交在途不关；提交仍要求编辑器就绪；文档级的 Escape 删去。单测 `create-root.close.test.tsx` 17 格，变异 `t2-mut-null-ref-refuses-r2`（回到“引用为 `null` 即拒绝”：空表单的 Discard 不关）、`t2-mut-unready-closes-r2` 等都杀。页面照 C10 的次序（H8 (page) `e2e/stories/collaboration/h8-intake-close-page.spec.ts:97`）：页面载入之前扣住编辑器的代码块、收集箱是空的，每一步之前断言扣住仍在、弹窗里没有可编辑的编辑器；Escape 关空表单，再开、Discard 关空表单，提交被拒且没有 `POST`；放开代码块之后点背景关空表单。去掉扣住的红色版本 `t7-red-page-hold` 在“提交被拒、没有 `POST`”一步失败。这个测试在 Chromium 与 Firefox 跑，WebKit 以写明原因的 `test.skip` 跳过（WebKit 在动态 import 时只要有 `page.route` 生效就不用预载、再取一次，约 3 秒后告警，安静控制台过不了，[P5b 阶段记录](../reviews/P5b-web-intake-files-embed-review.md) §3）；WebKit 的关闭政策由编辑器已建成的 `:156` 一格三引擎核。“有未完成图片时仍拒绝”对这个不收图片的表单不适用。**本条关闭。**

<a id="M6-M7-02"></a>
## M6-M7-02 迭代与模块的收藏，归档时移除收藏

- **现状**：收藏的表与接口属 M7（[M6 设计](../../M6-cycles-modules/M6-design.md) §3.6 `:89`、§8 `:201`）。M6 删除了迭代、模块里调用旧收藏接口的入口与 store 逻辑——迭代的在 P5a（[P5a spec](../../M6-cycles-modules/specs/P5a-web-cycles.md) §5），模块的在 P6a（[P6a spec](../../M6-cycles-modules/specs/P6a-web-modules.md) §5）：列表行与卡片的星、命令面板的“加入/移出收藏”都没有了，差异见 [plane-diff](../../plane-diff.md) 的“迭代的收藏与日期检查（M6/P5a）”“模块的收藏与工作区的模块列表（M6/P6a）”两行。代码里指向 M7 的说明：`web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts:18`、`…/context-based/module/commands.tsx:30`、`web/apps/web/core/hooks/use-favorite-item-details.tsx:46`、`:51`（收藏的显示只用收藏自己的名字，不读迭代、模块的 store）。
- **归档时移除收藏**：[总体设计](../../v0-design.md) §5.5（`v0-design.md:338`）照搬 Plane：迭代归档时同时移除指向它的收藏；M6 设计 §3.6 把它交 M7 落实。M6 的归档（`/api/v0/cycles/{cycle_id}/archive`、`/api/v0/modules/{module_id}/archive`）今天不碰收藏。模块归档时 Plane 是否同样移除，由 M7 照 Plane 核对。

**何时验收：** M7 的收藏接通时：迭代、模块的收藏入口随之恢复，只调 `/api/v0`、SWR 键带 `loginId`；归档迭代时指向它的收藏被移除（模块照 M7 核对的 Plane 行为），以接口、数据库与页面的证据核对；看不到的项目、迭代、模块的收藏不显示。

**处理结果（M7/P3c，后端）：** 迭代、模块的收藏接口在 `/api/v0`；迭代、模块的删除与归档都在原事务里移除**所有人**对它的收藏（经 `planning.cycle_deleted`、`planning.module_deleted` 与 P3c 新发布的 `planning.cycle_archived`、`planning.module_archived`），恢复不加回；本条留的一问“模块归档时 Plane 是否同样移除”答是（`apps/api/plane/app/views/module/archive.py:553-558` 按目标删所有人的）；看不到的项目、迭代、模块的收藏读时不列（[P3c spec](../specs/P3c-favorites.md) §2.5、§2.6）。收藏入口的恢复与页面的证据随 P7，保持 open。

<a id="M6-M7-03"></a>
## M6-M7-03 工作区级的迭代与模块：列表、视图的迭代与模块词、跨项目的行、工作区范围的命令面板搜索

- **现状**：nerve 没有工作区级的迭代、模块列表（[M6 设计](../../M6-cycles-modules/M6-design.md) §5 `:166`：Plane 的 `/workspaces/{slug}/cycles/` 只给工作区视图与通知用，属 M7）；M6 的列表都按项目（`/api/v0/projects/{project_id}/cycles`、`…/modules`）。P6a 删去通知页对工作区级模块列表的取数（`use-workspace-issue-properties.ts` 整个删除，[P6a spec](../../M6-cycles-modules/specs/P6a-web-modules.md) §5）。
- **视图的迭代、模块词**：工作区级视图的迭代、模块筛选（[P3b spec](../../M6-cycles-modules/specs/P3b-list-engine.md) §5），视图的筛选与分组的迭代、模块词随视图的页面（[P7b spec](../../M6-cycles-modules/specs/P7b-list-vocabulary.md) §5）。项目页、迭代页、模块页与归档页的词表、开关关闭的规则（P7b 的 R8、R17）在 `@nerve/constants` 的 `ISSUE_DISPLAY_FILTERS_BY_PAGE` 与 `usePlanningNames`、`usePlanningFilterConfigs` 等共用件里，视图照它们给。
- **跨项目的行**：工作区的草稿、个人页等列出多个项目的行，每行按自己工作项的项目读那个项目的迭代、模块目录，没有工作区级的目录（[P7a spec](../../M6-cycles-modules/specs/P7a-work-item-planning.md) §2.3、R5）；工作区级的列表出现之后，跨项目的行、成员列表的目录与跨项目的视图可改读它（P7a spec §5、P7b spec §5）。
- **工作区范围的命令面板搜索**：命令面板的搜索今天只在项目范围、在 store 已取得的该项目未归档的列表里按名字本地过滤，工作区范围不给（nerve 没有跨项目列迭代、模块的接口；[M6 设计](../../M6-cycles-modules/M6-design.md) §8 `:202`、[P7c spec](../../M6-cycles-modules/specs/P7c-palette-mobile.md) §5 与 R15，差异见 [plane-diff](../../plane-diff.md) 的“命令面板的搜索：迭代、模块（M6/P7c）”一行）。

**何时验收：** M7 加工作区级的迭代、模块列表（或搜索接口）时：接口按 planning 的权限（看不到的项目、关着该域的项目不列，访客与 OwnOnly 同 M6 的规则），视图的筛选、分组给迭代、模块词且与项目页同规则；跨项目的行、成员列表的目录改读它；工作区范围的命令面板搜索给迭代、模块两组。各有接口与页面的证据。

**处理结果（M7/P3a）：** 工作区跨项目列表不收迭代、模块的筛选与分组词（设计 §15 D4：工作区视图没有迭代、模块词，422）；次序照收不分范围的 `ListOrderAllowed`（含 `cycle_name`、`module_name`，它也是游标键的校验，成员列表已如此）。本条“视图的迭代、模块词”在工作区一侧由 D4 关闭；工作区级的迭代、模块列表、跨项目的行改读它、工作区范围的命令面板搜索仍未做，M7 收官时定去向。保持 open。

**处理结果（M7/P3b）：** 项目视图存迭代、模块词（`cycle_id`、`module_id` 是项目视图的已存筛选键，经项目列表的 `ListSettings` 校验，与项目页同规则）；工作区视图不收（422，D4）（[P3b spec](../specs/P3b-views.md) §2.3、§2.4）。视图页上的词表随 P6a/P6b。其余照上。保持 open。

**处理结果（M7/P6a，项目视图的词）：** 视图页是项目列表页的一种，词表与项目页相同（`web/packages/constants/src/issue/filter.ts:245` 的 `[EIssuesStoreType.PROJECT_VIEW]: ISSUE_DISPLAY_FILTERS_BY_PAGE.issues`），筛选与分组照项目页给迭代、模块词；迭代页、模块页另存的视图带该迭代、模块的条件（`web/apps/web/core/hooks/planning-page-domains.ts:44` 的新列 `viewTerms`，迭代 `:61` 给 `cycle_id`、模块 `:75` 给 `module_id`），另存出来的与页面上看到的是同一批工作项（[P6a spec](../specs/P6a-web-project-views.md) §2.7、R7；变异 `t8-mut-no-view-terms`；H12 (page) `e2e/stories/collaboration/h12-project-view-saving-page.spec.ts:141`）。项目视图的部分关闭；工作区视图（D4 不收）、工作区级的迭代、模块列表、跨项目的行与工作区范围的命令面板搜索照上，M7 收官时定去向。保持 open。

<a id="M6-M7-04"></a>
## M6-M7-04 视图的 store、读与页面，M3 的视图遗留

- **现状**：视图的 store 与读今天没有东西填（视图页一直在加载）。M3 交给 M6 的视图文件 M6 已做（P7c）：四个视图文件的地址只从路由参数、视图写之后的跟进经 `followInSession`、`isCreateViewModalOpen` 随项目级弹窗宿主（[M3 → M6 交接](../../M6-cycles-modules/handoffs/M3-closeout.md)的“处理结果（M6/P7c）”，[M6 设计](../../M6-cycles-modules/M6-design.md) §8 `:203`）。
- **不改、属 M7**（[P7c spec](../../M6-cycles-modules/specs/P7c-palette-mobile.md) §5、R12；M3 → M6 交接的 P7c 回执）：
  - 视图的 store 与读、页面内容与保存的视图自己的页面；
  - `projectViewIds`：`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/[viewId]/header.tsx:41`、`:47` 经 store 的这个 getter 读路由；
  - `projectMemberIds`：`…/[projectId]/views/(list)/mobile-header.tsx:23`、`:56` 读 router 驱动的成员 getter；
  - 视图的文案：拒绝的提示照旧是英文（视图的 service 仍调 Plane 的旧地址，nerve 的问题码对不上）；
  - 嵌套的按钮：`web/apps/web/core/components/views/quick-actions.tsx` 的菜单按钮里套 `IconButton`（单测渲染视图的行时 React 报出）。
- **命令面板的视图菜单**：`web/apps/web/core/components/power-k/menus/views.tsx` 在 M6/P8 改为以 id 作 cmdk 的 value、名字作 keywords（`83aaea3f`），它的行仍以 `@nerve/types` 的 `IProjectView` 为类型（`views.tsx:11`；P8 T2 的裁定：视图属 M7，不顺带换类型的来源）。
- **M3 第 3 节视图的三项**（M6/closeout 补，预检 [re-80](../../../reviews/re-80-M6收尾预检.md) I4）：[M3 → M6 交接](../../M6-cycles-modules/handoffs/M3-closeout.md)第 3 节有三项落在视图的页面上，M6 走不到（视图页没有读），没有处理结果：视图的弹窗的 Tab、Escape、点外面、关上之后的焦点与弹窗的名字（`:32-33`）；视图列表的右键菜单关着时 `inert`、右键打开它并点它的项（`:36`）；视图页头的 `CustomSearchSelect` 以键盘核对（`:37`）。前两项 M7 自己的 [M3-closeout §3](M3-closeout.md#3-键盘弹窗和下拉框) 已有（弹窗、右键菜单两条）；视图页头的切换不在其中，照本条核对。

**何时验收：** M7 接通视图时：视图的 store 与读只调 `/api/v0`、按会话分代，类型来自 `@nerve/api-client`（命令面板的视图菜单一并换掉 `IProjectView`）；上面的 getter 改为读路由参数，改过的文件加入关键词规则 `project-address-route`；视图的文案经 `t()`；菜单按钮不再嵌按钮；上面 M3 第 3 节视图的三项在浏览器核对（视图的弹窗与右键菜单随 M7 自己的 M3-closeout §3 的关闭条件，视图页头的 `CustomSearchSelect` 以键盘）。

**处理结果（M7/P3b，只到接口）：** 视图的接口在 `/api/v0`（[P3b spec](../specs/P3b-views.md) §2.4），两个新码 `navigation.view_not_found`、`navigation.views_disabled` 有网页的文案（`error-messages.ts` 与 `errors.json`）。store、读、页面、getter、命令面板的视图菜单、嵌套的按钮与 M3 第 3 节视图的三项都还在 P6a/P6b。保持 open。

**处理结果（M7/P6a，项目视图一半）：** 逐项（[P6a spec](../specs/P6a-web-project-views.md) §2.1–§2.8）：
- 视图的 store 与读：`ViewStore`、`ViewChanges`（`web/apps/web/core/store/views/`）只调 `/api/v0`（`core/services/views/views.service.ts`），按会话分代、类型来自 `@nerve/api-client`；读只在用到的页面（R20）；视图页与它的列表、另存与“更新视图”接通（T2、T6–T8）。
- `projectViewIds`：视图页头改写，切换器的选项来自列表页的键的读、第一次打开时才读（`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/[viewId]/header.tsx:100`），不再经 store 的 getter 读路由；旧 store 连同 getter 在 T10 删除。
- `projectMemberIds`：两个读者改读 `getProjectMemberIds(projectId, true)`（`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/mobile-header.tsx:78`、`core/components/views/view-list-header.tsx:127`），getter 删除（R16，T4）。
- 视图的文案：拒绝经 `useRefusalToast` 按 nerve 的问题码说（`navigation.view_not_found`、`navigation.views_disabled` 的文案 P3b 已有），页面、页头、弹窗与筛选行的英文经 `t()`（R21）。
- 嵌套的按钮：`core/components/views/quick-actions.tsx` 的菜单按钮是 `CustomMenu` 自己的按钮，不再套 `IconButton`（T4）。
- 命令面板的视图菜单：`power-k/menus/views.tsx` 改读 `View`（`@nerve/api-client`），命令面板不再读 `IProjectView`（R13，T9）。
- 改过与新写的视图文件读路由参数，收入关键词规则 `project-address-route`（T4、T6、T7、T8）。
- M3 第 3 节视图的三项：弹窗与右键菜单见 [M3-closeout §3](M3-closeout.md#3-键盘弹窗和下拉框) 的 P6a 处理结果；视图页头的 `CustomSearchSelect`（`@nerve/ui` 的 `BreadcrumbNavigationSearchDropdown`）以键盘核过，三种引擎：焦点在切换器上按 Enter 打开，搜索框取得焦点，打字、下箭头选中（`aria-activedescendant`）、Enter 换到另一个视图（`e2e/stories/collaboration/h20-view-pages.spec.ts:52-69` 的 `switchByKeyboard`，在 `:125`、`:146` 用）。

项目视图一半关闭；工作区视图的 store、页面与 M3 交给它的项随 P6b。保持 open。

<a id="M6-M7-05"></a>
## M6-M7-05 `getTabIndex` 与两份 `tab-indices.ts` 的最后删除

- **现状**（P8 的 HEAD 以 `git grep` 核对）：`getTabIndex`（`web/packages/utils/src/tab-indices.ts:11`）的读者只剩 M7 的五个文件：`web/apps/web/core/components/views/form.tsx:75`（`PROJECT_VIEW`）与收集箱新建弹窗 `web/apps/web/core/components/inbox/modals/create-modal/{create-root,issue-description,issue-properties,issue-title}.tsx`（`INTAKE_ISSUE_FORM`）；`ETabIndices` 与它的表在 `web/packages/constants/src/tab-indices.ts`，两份经各自的桶文件导出（`web/packages/utils/src/index.ts:25`、`web/packages/constants/src/index.ts:29`）。以 `getIndex(…)` 把正的 `tabIndex` 传给控件的只在这五个文件里；`@nerve/ui` 共用的 `IDropdownProps.tabIndex`（`web/packages/ui/src/dropdowns/helper.tsx:25`）由 `CustomSelect`、`CustomSearchSelect`、`CustomMenu` 读。M6 的读者（`cycles/form.tsx`、`modules/form.tsx` 的 `PROJECT_CYCLE`、`PROJECT_MODULE`）与两处归档按钮的 `tabIndex={1}` 已在 P5a、P6a 删去。
- **来源**：[M6 设计](../../M6-cycles-modules/M6-design.md) §8 `:199`、§11；[M3 → M6 交接](../../M6-cycles-modules/handoffs/M3-closeout.md)第 5 节与 P6a、P8 的回执；[M4 → M6 交接](../../M6-cycles-modules/handoffs/M4-closeout.md#M6-04)的 M6-04（“同步另一份交接”即本条）；同一条件也在 M7 自己的 [M3-closeout](M3-closeout.md)、[M4-closeout](M4-closeout.md)。

**何时验收：** M7 把视图表单与收集箱新建弹窗迁到自然的 Tab 次序（同 M6 的 `cycles/form.tsx`、`modules/form.tsx`），完成各自弹窗、portal 与回焦的回归之后，删除 `getTabIndex`、两份 `tab-indices.ts`、两个桶文件里的导出与 `ETabIndices`；全仓再没有正的 `tabIndex` 传给共享控件时，`IDropdownProps.tabIndex` 随之收口。菜单的非正 `-1`（如 M4 表格列头的 `customButtonTabIndex={-1}`）独立核实后保留，不以“没有正索引”等同全部焦点属性可删（M3 → M6 交接 §5 的 M4/P16 回执）。

**处理结果（M7/P5b）：** 收集箱新建弹窗的四个文件（`create-root`、`issue-description`、`issue-properties`、`issue-title`）离开 `getTabIndex` 与只为它读的 `usePlatformOS`，走 DOM 的自然次序；`web/packages/constants/src/tab-indices.ts` 删去收集箱一组（`INTAKE_ISSUE_CREATE_FORM_TAB_INDICES`、枚举成员 `INTAKE_ISSUE_FORM` 与映射一行）（`fa78bb11`，[P5b spec](../specs/P5b-web-intake-files-embed.md) §2.1）。描述编辑器改带非正的 `tabIndex={-1}`（`ec6b91a5`：共用编辑器没有 `tabIndex` 时吞掉列表之外的 Tab，[P5b 阶段记录](../reviews/P5b-web-intake-files-embed-review.md) §3）。`git grep getTabIndex -- web` 只剩视图表单 `web/apps/web/core/components/views/form.tsx:26`、`:75` 与函数本身；`getTabIndex`、`ETabIndices`、两份 `tab-indices.ts` 与桶文件的导出随 P6a 删除。保持 open。

**处理结果（M7/P6a）：** 视图表单在 T5 离开 `getTabIndex`、走自然的 Tab 次序（图标 → 名字 → 说明 → Cancel → 提交），弹窗、portal 与回焦的回归在 H12 (page)、H20 (page) 三种引擎（`e2e/stories/collaboration/h20-view-pages.spec.ts:291`、`:255`；`h12-project-views-list-page.spec.ts:196`）；同一提交删除 `getTabIndex`、`ETabIndices`、`TAB_INDEX_MAP`、两份 `tab-indices.ts` 与两个桶文件里的导出（`5a58db86`，[P6a spec](../specs/P6a-web-project-views.md) §2.9、R15）。核过：之后 `git grep -n "getTabIndex\|ETabIndices\|TAB_INDEX_MAP\|PROJECT_VIEW_TAB_INDICES\|tab-indices" -- web` 为空、三个包的 dist 里也没有（`t5-tabindex-gone`）；测试之外全 web 没有正的 `tabIndex`。`IDropdownProps.tabIndex`（`web/packages/ui/src/dropdowns/helper.tsx:25`）与 `dropdowns/types.d.ts:21`、`date-range.tsx:50` 的非正 `tabIndex?` 保留：各下拉框原样转给按钮，应用里传显式值的只有 M4 表格列头的 `customButtonTabIndex={-1}`（`web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/header-column.tsx:48`），独立核过、不是正索引的残留（`t5-tabindex-readers`）。**本条关闭。**

<a id="M6-M7-06"></a>
## M6-M7-06 迭代的转移不发布 `issue.updated`

- **现状**：转移把源迭代里未完成的工作项移到目标迭代：每个工作项的旧归属软删、新归属插入、一条 `CycleMoved` 动态、`MarkChanged`，都在 planning 的写事务里，不发布 `issue.updated`（`server/internal/modules/issue/app/cycle_transfer.go:73`，[P4 spec](../../M6-cycles-modules/specs/P4-progress-transfer.md) R8；先例：删迭代、模块时的 `NewMembershipRemovals` 同样不发布）。
- **来源**：[P4 spec](../../M6-cycles-modules/specs/P4-progress-transfer.md) §5 的“P8（写给 M7 的交接）”一条（预检 re-62 L5）。

**何时验收：** M7 的通知接通时决定通知是否要知道转移：要，就在转移的事务里加事件（由 M7 定由谁发布、几条），订阅者据此通知，以接口与数据库的证据核对一次转移的通知；不要，在此写明理由。

<a id="M6-M7-07"></a>
## M6-M7-07 访客在命令面板看到视图与收集箱的命令

- **现状**：项目一级的导航命令里，“打开视图”`open_project_view` 与“转到视图”`nav_project_views` 只看项目的 `issue_views_view`，“转到收集箱”`nav_project_intake` 只看 `intake_view`（`web/apps/web/core/components/power-k/config/navigation/project-commands.ts:175-176`、`:187-188`、`:200-201`），不看角色：项目的访客也看得到。对照：“转到归档”`nav_project_archives` 以 `hasProjectMemberLevelPermissions` 给（`:213-214`），迭代、模块的四个命令以入口规则 `planningEntryOn` 给。M3、M4 起的行为，P7c 的特征测试（`power-k/config/navigation/commands.test.tsx`）钉住它，P7c 复核时记下（[P7c 阶段记录](../../M6-cycles-modules/reviews/P7c-palette-mobile-review.md) §1），P8 不改（[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §3 第 6 条：视图、收集箱属 M7 的领域）。项目设置的同类命令交 M8（[M6-M8-19](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-19)）。

**何时验收：** M7 接通视图、收集箱的页面时，按 M7 的权限定这三个命令给谁（访客能否读这些页面），命令面板与侧栏同一规则，以单测钉住访客与成员两种角色。

**处理结果（M7/P5a，收集箱的一半）：** 收集箱的页面给项目的管理员、成员与访客（访客能读收集箱），所以“转到收集箱”`nav_project_intake` 给有项目角色、收集箱开着的人：侧栏（`workspace/sidebar/project-navigation.tsx:120`）、页签栏（`navigation/use-navigation-items.ts:90`）与命令面板（`power-k/config/navigation/project-commands.ts:76`）共用一条规则 `intakeEntryOn`（`core/lib/intake-entry.ts:18-24`，不排除已归档的项目）（[P5a spec](../specs/P5a-web-intake.md) §2.5、R21）。单测钉住：规则本身六格（管理员、成员、访客、没有角色、关着、已归档，`core/lib/intake-entry.test.ts`）；命令面板与页签栏各给成员与访客、不给在项目里没有角色的工作区管理员，页签栏另有收集箱关着的一格（`project-commands.test.tsx:49`、`:54`，`use-navigation-items.test.tsx:41`、`:45`、`:49`）；侧栏给访客、不给没有角色的工作区管理员（`project-navigation.test.tsx:107`、`:113`）。收集箱的一半关闭；视图的两个命令随 P6a，保持 open。

**处理结果（M7/P6a，视图的一半）：** 视图的页面给项目的管理员、成员与访客（访客能读视图），所以“打开视图”`open_project_view` 与“转到视图”`nav_project_views` 给有项目角色、开关开着的人：侧栏、页签栏与命令面板共用一条规则 `viewEntryOn`（`web/apps/web/core/lib/views-entry.ts:18`，不排除已归档的项目），命令 `create_view` 读 `viewsCreatableBy`（`:31`，另要管理员或成员）（[P6a spec](../specs/P6a-web-project-views.md) §2.2、R4、R14）。单测钉住：规则本身 34 格（`core/lib/views-entry.test.ts`）、`create_view` 8 格（`power-k/config/creation/create-view.test.tsx`）；命令面板的表 `power-k/config/navigation/commands.test.tsx:120`、`:138` 两行不变，文件末追加两条视图命令的角色格（成员、访客给，没有角色的不给）；变异 `t3-mut-entry-no-role` 等。视图的一半关闭，两半都已关闭。**本条关闭。**

<a id="M6-M7-08"></a>
## M6-M7-08 首个投递任务的 CLI（原文转交）

下面完整引用 M5 交给 M6 的 M5-04（其中又完整引用 M4 交给 M5 的 M5-04 与 M2 原第 1 节）；相对链接按原出处解释，原文不改。可点击来源：[M5 → M6 原交接](../../M6-cycles-modules/handoffs/M5-closeout.md#M5-04)。M7 自己的 [M5-closeout](M5-closeout.md#M5-04) 持有同一条款的 M5 原文。

<!-- M5-M6-04-BEGIN -->
`````text
## M5-04 首个投递任务的 CLI（原文转交）

下面完整引用 M4 交给 M5 的 M5-04，其中又完整引用 M2 原第 1 节；相对链接按原出处解释，原文不改。可点击来源：[M4 → M5 原交接](../../M5-files/handoffs/M4-closeout.md#M5-04)。

<!-- M4-M5-04-BEGIN -->
````text
## M5-04 首个投递任务的 CLI

下面完整引用 M2 原第 1 节；其中相对链接按原出处解释，原文不改。可点击来源：[M2 → M4 原交接](../../M4-issue-core/handoffs/M2-closeout.md)。

<!-- M2-SECTION-1-BEGIN -->
```text
## 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）

- **负责人的要求**：2026-09-26 批准把它推迟到 M4 时说，"做好记录，以后别漏了"。M4 用不到它，也要把这一节原样转进下一个 M 的交接，不能随本交接关闭而丢掉。
- **做什么**：第一个"用例会投递 River 任务、又能从命令行触发"的命令加入时，`platform/jobs` 提供一个只投递的客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`），`bootstrap` 的命令组合把它交给这些用例。
- **为什么**：命令行的组合只有连接池和 `identity` 的管理用例，没有 River 客户端（`server/internal/bootstrap/users.go:22-26`，M2 设计 3.17）。服务用的客户端（`server/internal/platform/jobs/jobs.go:70` 的 `New`）配队列、启动 worker，命令行不能用它。用例在事务里用 `InsertTx` 投递（M2 设计 3.15）；命令行上没有客户端时，任务要么没投递，要么用例在运行中失败。M2 的命令都不投递任务，所以推迟（M2 设计 3.15、3.17、13.2）。
- **可能早于 M4**：M3 给停用加端口（[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 6 节），`nerve users deactivate` 也走这个用例；端口的实现投递任务时，这一项提前到 M3。
- **建议的守法**：投递任务的端口在用例的构造参数里必填，命令的组合漏了它，在构造时就失败，而不是运行到一半；`server/internal/bootstrap/users_test.go` 这样的组合测试覆盖每个命令。
- **关闭条件**：M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。

```
<!-- M2-SECTION-1-END -->

**当前事实与责任：** 只投递的客户端已在 `platform/jobs`（`NewInserter`，M4/P6）；M3、M4 没有投递任务的命令，现有 CLI 组合无需构造它。**第一个投递任务的命令加入时，把它交给命令行的组合**，按 §3.20 显式声明投递的 kinds，启动时验证对应 worker。

**何时验收：** 首个投递任务的 CLI 提交中完成。真实命令经运行角色执行，验证 `river_job` 的 kind/args、提交留任务、回滚不留任务；少登记一个 worker 的负例必须失败。不能把服务端已有 NewInserter 当 CLI 已接通。
````
<!-- M4-M5-04-END -->

**M5 结束时的事实：** M5 仍没有投递任务的命令。CLI 的组合（`bootstrap/users.go`、`workspaces.go`、`commands.go`）只构造连接池与管理用例，不接事件发布者；`jobs.NewInserter` 唯一的非测试调用在 `bootstrap/app.go`（serve），由 `issueInserter` 与 `assetInserter` 包装。M5 新增的投递均在服务中：`asset.delete_scope` 由 `project.deleted`/`workspace.deleted` 的订阅者在发布者事务内投递。

**何时验收：** 首个投递任务的 CLI 提交中完成，验收同上文原文；M6、M7、M8 谁先加入这样的命令，谁完成并在另两份交接中写明。
`````
<!-- M5-M6-04-END -->

**M6 结束时的事实：** M6 仍没有投递任务的命令。CLI 的组合（`server/cmd/nerve` 与 `server/internal/bootstrap/users.go`、`workspaces.go`、`commands.go`）不构造 River 客户端，`jobs.NewInserter` 唯一的非测试调用仍在 `server/internal/bootstrap/app.go`（serve）。M6 新加的投递都在服务里：`planning.delete_rows` 由 `project.deleted`、`workspace.deleted` 的订阅者在发布者的事务里投递，`planning.crawl_link` 在模块链接的写事务里投递，`planning.freeze_cycles` 是周期任务（[M5 → M6 交接](../../M6-cycles-modules/handoffs/M5-closeout.md)的“处理结果（M6/P8）”）。

**何时验收：** 首个投递任务的 CLI 提交中完成，验收同上文原文；M7、M8 谁先加入这样的命令，谁完成并在另一份交接中写明。转交不表示已实现。

<a id="M6-M7-09"></a>
## M6-M7-09 迭代与模块成为资产宿主的条件（M5-M6-01 原文转交）

M5-M6-01 在 M6 以“未触发”结束：M6 结束时迭代、模块的描述是纯文本，模块链接的图标以 `data:` 地址内嵌，`api/modules/asset.yaml` 与 `server/internal/modules/asset` 在 M6 全期没有改动（[M5 → M6 交接](../../M6-cycles-modules/handoffs/M5-closeout.md#M5-M6-01)的“处理结果（M6/P8）”）。M7 若给迭代、模块带图片或附件，照下面的原文接入；相对路径按原出处解释，原文不改。可点击来源：[M5 → M6 原交接](../../M6-cycles-modules/handoffs/M5-closeout.md#M5-M6-01)。

<!-- M5-M6-01-BEGIN -->
```text
## M5-M6-01 迭代与模块现在不是资产的宿主

- **现状**：资产用途由 OpenAPI 的 `AssetPurpose` 给出，共十种（`api/modules/asset.yaml`；Go 为 `asset/domain/scope.go` 的 `Purpose` 与 `ScopeKind`，数据库为 `file_assets_scope_check`）：`USER_AVATAR`、`USER_COVER`（用户本人）、`WORKSPACE_LOGO`（工作区）、`PROJECT_COVER`（项目，或建项目前以工作区为 id 的 `new_project`）、`ISSUE_ATTACHMENT`/`ISSUE_DESCRIPTION`（工作项）、`COMMENT_ATTACHMENT`/`COMMENT_DESCRIPTION`（评论，或发评论前以工作项为 id 的 `new_comment`）、`DRAFT_ISSUE_ATTACHMENT`/`DRAFT_ISSUE_DESCRIPTION`（草稿）。迭代与模块的表单是纯文本（`components/cycles/form.tsx`、`components/modules/form.tsx` 的 `TextArea`），进度侧栏只读，没有图片或附件。
- **若 M6 让迭代/模块带图片或附件**，照现有宿主（工作项、评论、草稿）的接法新增宿主，不另起一套：
  1. 用途与范围加入 `asset/domain/scope.go`、`content.go` 与 YAML 枚举；迁移给 `file_assets` 加 `<host>_id … ON DELETE NO ACTION` 外键与新的范围检查分支（先例 `00034_asset_file_assets.sql`）。
  2. 宿主模块导出 `AssetAccess{Inspect, Check}` 与 `AssetHostNotFound`/`AssetHostChanged`（先例 `issue/module_assets.go`、`issue/app/asset_access.go`），在 `bootstrap/asset_access.go` 加一支；宿主行需要锁时加锁端口（`bootstrap/asset_ports.go`）。
  3. 导出 `Asset<Host>Facts` 返回 `AssetParent{ID, DeletedAt, DeletedByID}`，在 `bootstrap/asset_parent_reader.go` 加一批读取；`asset.cleanup` 的行阶段据此判断父删除。
  4. 删除：宿主删除事件由 `asset/app/deletion_events.go` 订阅（现订 `issue.deleted`、`issue.comment_deleted` 同步软删，`project.deleted`、`workspace.deleted` 在发布者事务内投递 `asset.delete_scope`），或像草稿一样调直接端口；`DeleteScopeRows` 现在只处理 issue/comment/draft，需同步扩展。
  5. 富文本图片：issue 模块从 `image-component[src]` 取规范 UUID，写宿主前 `LockContent`、写后以同一时刻 `BindContent`（`issue/app/create_core.go`）。新宿主须扩展 `asset/app/content_bindings.go` 的宿主检查与 `content_binding_rules.go` 的用途与放置规则；前端 `store/asset/content-view.ts` 的 `ContentHost` 与 EditorImages 能力随之扩展。描述中只存资产 ID，不存签名地址。
  6. 物理清理：外键加入 `ProtectedParentReferences`，宿主的 purge 步骤标 `Protected: true`，模块在 `bootstrap/app.go` 的 `newPurgeDeleted` 中排在 asset 之后、被它引用的父模块之前；`asset_protection_catalog_test.go` 与 `purge_catalog_test.go` 会核对覆盖。
- **不能做的**：跨模块 JOIN `file_assets`；绕开宿主的权限检查读文件；把旧的 Plane 文件地址或 `/api/assets/v2` 带回。

**何时验收：** 迭代/模块第一次带文件之前。上传、拒绝、取消、换宿主与迟到结果、权限矩阵、删除立即不可见、60 天清理与存储重试，均需与 M5 的 F8–F15 同等的 API、字节与页面证据（local 与 S3）。
```
<!-- M5-M6-01-END -->

**何时验收：** 同上文原文：迭代或模块第一次带文件之前，与 M5 的 F8–F15 同等的 API、字节与页面证据（local 与 S3）。M7 不让它们带文件时，本条在 M7 收尾时以“未触发”关闭并照原文转交下一个 M。
