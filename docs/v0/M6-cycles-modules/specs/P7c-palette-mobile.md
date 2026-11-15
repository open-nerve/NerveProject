# M6 / P7c：工作项一侧（三）——命令面板、移动端页头与遗留

> 状态：spec 与 plan 已写（2026-11-15），待预检。基线 main `d052213f82aa4fba234f5684722335e1a5cbb5be`（P7b 的 noff，双父 `f42f9d64`、`4fe308ce`，树与 `4fe308ce` 相同；P7b 的最终分支 CI 43702923005 attempt 1 全绿，e2e 作业 132781366495 为 10:26:46Z–10:54:36Z，27 分 50 秒，至今最慢；合入后的 main CI 43706438924 attempt 1 全绿，e2e 作业 132792042944 为 11:10:31Z–11:34:13Z，23 分 42 秒；P7b 的归档在主目录的 `.superpowers/sdd/M6-P7b-integration-20261115/`，含 `cleanup.log`；这些由本段 T1 以归档与公开的 jobs 接口核对后回填进 P7b 的记录）。设计见 [M6 设计](../M6-design.md) §2 的 G19，§8 的“工作项一侧”“命令面板”与“交给 M6 的视图文件”，§12 的 P7c 一行，§13–§15；P7 的拆分见 [P7a spec](P7a-work-item-planning.md) §1、R1，交给本段的事见 P7a spec §5、[P7b spec](P7b-list-vocabulary.md) §5 与 P7b 的[阶段记录](../reviews/P7b-list-vocabulary-review.md) §5；主控在写 spec 之前的裁定 H1–H16 见本阶段台账 `controller-rulings.md`（本段 R1–R16 照录）。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。现状调研见本阶段台账 `research/palette-views.md`（命令面板、快捷操作、四个视图文件、G19 的工作项一侧）与 `research/mobile-carryovers.md`（移动端页头、A19–A21、交接回执、e2e 的余量与放置、估算）（ignored，随阶段归档；本 spec 引用的行号都在 `d052213f` 的树上）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认；本段不改 `server/`、`api/`（R15），本机 `make test` 若因 [M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05) 的签名失败，以分支 CI 的服务端两路为准（负责人 2026-11-11）。

文中的路径：`core/…` 在 `web/apps/web/` 下；`PK/…` 是 `core/components/power-k/…`；`R/…` 是 `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/…`；`issue-layouts/…` 是 `core/components/issues/issue-layouts/…`；`@nerve/…` 的包在 `web/packages/` 下。

## 1. 目标与边界

P7a、P7b 交付了工作项自己的迭代与模块和列表的词表；P7 余下的是命令面板、移动端页头与几处遗留（P7a spec §5、P7b spec §5）。现状（`research/*.md` §0）：命令面板“打开一个迭代/模块”在归档的项目里永远转圈（`PK/ui/pages/open-entity/project-cycles-menu.tsx:40-45`，模块同），命令不看项目的归档（`PK/config/navigation/commands.ts:310-367`）；七个菜单的空文案是英文字面量（`PK/menus/*.tsx`）；搜索没有迭代、模块的结果；没有给工作项改迭代、模块的命令；迭代页没有移动端页头、侧栏在窄屏上关上就打不开，模块页也没有；项目页的移动端页头在偏好读到之前就给控件（X2）；分组时项目页与归档页的页头没有数目（A19）；“添加已有”的弹窗打开时焦点不在搜索框；设计 §8 的四个视图文件还经路由的退路读地址（X1）。本段（R1）：

- **回填与余量**：P7b 的回填（阶段记录 §7、spec A15、设计 §14、[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的数据点），以 P7b 实测的 CI 重算 e2e 的余量（R14，§2.12）；关键词 phase 仍是 `M6/P7`。
- **命令面板**（A16）：先按层级拆 `navigation/commands.ts`（466 行，§2.1）；规划入口一条规则、菜单不再转圈（§2.2）；菜单的空文案经构造件的键（§2.3）；搜索的迭代、模块结果（§2.4）；工作项的“改迭代”“改模块”两个命令（§2.5）。
- **快捷操作与归档页的行菜单**（A17、A18，§2.6）。
- **移动端页头**（A15、X2）：一个共用的页的移动端页头，项目页改用它并删去手写的那份，迭代页、模块页接上；两页的窄屏顶行一条规则（§2.7）。
- **遗留**：分组时页头的数目（A19）、没有读者的键（A20）、规划页替身的 `opened`（A21）、“添加已有”打开即聚焦搜索框（§2.8）；设计 §8 的四个视图文件（X1，§2.9）。
- **验收**：G19 的工作项一侧的页面（§2.12）；交接回执：[M3 交接](../handoffs/M3-closeout.md) §1、§3、§4，[M4 交接 M6-02](../handoffs/M4-closeout.md#M6-02) 的“分组时页头的数目”，[M1 交接](../handoffs/M1-closeout.md) 本段改过的文件的 oxlint（§2.11）。

**不做**（交给后来的，详见 §5）：
- **P8**：归档页的迭代、模块词表（P7b R10）；[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)；新控件的焦点轮廓随最终矩阵核对；M1 交接的死成员收敛（设计 §12 的 P8 一行）；命令面板快捷键说明里的英文字面量（`PK/ui/renderer/shortcut.tsx:105`，R4）。
- **M7**：工作区范围的迭代、模块搜索（设计 §8 `:202`）；视图的 store 与读、store 的 getter `projectViewIds`、`projectMemberIds` 的读者、视图页的内容与文案（R12）。
- 任何接口或服务端的变化（R15）。

## 2. 交付合同

### 2.1 先拆 `navigation/commands.ts`（R3）

`PK/config/navigation/commands.ts`（466 行）在加代码之前按层级拆为工作区一级与项目一级两件，只搬不改（`git diff -M`），`usePowerKNavigationCommandsRecord` 的输出逐项相同（基线收据：引用这些命令的单测逐文件同名同数，见 plan）。项目一级的一件（含 `open_project_cycle`、`nav_project_cycles`、`open_project_module`、`nav_project_modules`、视图的两个）加入 `project-address-route`（它的 `allowPermissions` 都给 slug 与项目）；工作区一级的一件不加入（`:74-79` 三个参数的工作区一级调用是对的，会命中规则的正则）。

### 2.2 规划入口一条规则，打开的菜单不再转圈（R2）

- **规则**：`core/lib/planning-switch.ts` 加一个纯函数（名字由 T3 定，下称“入口规则”）：给出项目、调用者在其中的角色与域，答“这一域的规划入口给不给”——角色是 ADMIN 或 MEMBER、项目没有归档、域的开关打开（`planningOn`）。另一个纯函数给“这一域的列表读不了的原因”（问题码）：没有角色 → `planning.cycle_not_found` / `planning.module_not_found`，项目已归档 → `project.archived`，开关关闭 → `planning.cycles_disabled` / `planning.modules_disabled`，读得了 → 无；`usePlanningReads` 的 `listed`（`core/hooks/use-planning-reads.ts:37-39`）与它同一组事实，调用者的角色一律以 `getProjectRoleByWorkspaceSlugAndProjectId(slug, project)` 取（不经路由）。
- **命令**：`open_project_cycle`、`nav_project_cycles`、`open_project_module`、`nav_project_modules` 的显示改由入口规则判断（今天只看角色与开关，`:310-313`、`:325-328`、`:349-352`、`:364-367`），归档的项目里不再给出。侧栏的两张声明表（`core/components/workspace/sidebar/project-navigation.tsx:78-97`、`core/components/navigation/use-navigation-items.ts:45-66`）是同一条规则的声明写法（`access: [ADMIN, MEMBER]` 加 `shouldRender: <开关>`，归档的项目不在侧栏），不改。
- **打开的菜单**：`project-cycles-menu.tsx`、`project-modules-menu.tsx` 在门为假时不再渲染 `<Spinner />`：以上面的原因函数给出的问题码经 `errorMessageKey` 说明（`PowerKMenuEmptyState`），读的失败照旧以 `refusalMessage(read.error)` 说明，只有门为真而还没回答时才转圈。
- 访客照旧可按地址读（读门不变），没有入口（Plane 同：侧栏的迭代、模块对访客不给）。

### 2.3 菜单的空文案（R4）

`PK/menus/builder.tsx` 的 `emptyText`（今天是字面量的字符串，`:38` 在没有项时显示）改为 i18n 的键（类型只收 `power_k` 下的键或 `power_k.search_menu.no_results` 的默认），七个菜单（`cycles.tsx:29`、`modules.tsx:29`、`labels.tsx:34`、`views.tsx:29`、`projects.tsx:33`、`settings.tsx:33`、`workspaces.tsx:35`）各给一个新键（en 与 zh-CN，同一提交），`menus/empty-state.tsx:13` 的默认改读 `power_k.search_menu.no_results`。§2.2 的说明直接给文字（已翻译的拒绝），`PowerKMenuEmptyState` 收翻译后的文字、构造件收键，两者分开。

### 2.4 搜索的迭代、模块结果（R5）

- 只在项目范围（路由的项目，或 `/browse/:workItem` 所在工作项的项目，`PK/projects-app-provider.tsx:51-53`）；工作区范围不给（M7）。
- 两组“Cycles”“Modules”（标题读 `common.cycles`、`common.modules`），只在入口规则给出时；项来自 store 里该项目未归档的列表（含已结束的迭代：这是导航、不是加入的选项），按名字不分大小写的子串本地过滤（今天项目一组的写法，`PK/ui/modal/search-menu.tsx:42-52`；搜索时 cmdk 的根过滤关着，`PK/ui/modal/wrapper.tsx:140`，新组自己过滤）；store 里没有时以 `["PROJECT_CYCLES", projectId]`、`["PROJECT_MODULES", projectId]` 读（与项目的外壳、打开的菜单同一个键，`/browse/:workItem` 没有项目的外壳），门同 §2.2；读失败时这一组不显示、不挡工作项与项目的结果。
- 一行是项目的标识与名字（Plane 的行，`plane/apps/web/core/components/power-k/ui/modal/search-results-map.tsx:36-46`），选中即关面板并转到 `/{slug}/projects/{pid}/cycles/{id}`、`/modules/{id}`（同“打开一个迭代”的导航）。
- 与 Plane 的差异（本地过滤、只有未归档的、只在项目范围）登记在 `plane-diff.md`。

### 2.5 工作项的“改迭代”“改模块”（R6）

- **位置**：工作项上下文的命令（`PK/ui/pages/context-based/work-item/`），两个新页类型（`PK/core/types.ts`），占位文案（`PK/ui/modal/constants.ts`）。显示：工作项可改（上下文的 `canEdit`：ADMIN/MEMBER、工作项与项目都没有归档，`context.ts:24-30`）且该域开关打开（`planningOn`）。
- **改迭代**：`shift+c`，打开一个列表：“无迭代”（读 `cycle.no_cycle`）在前，然后是可加入的（`cycleOptions`：未归档、未结束，nerve 的次序，`core/lib/planning-directory.ts:37-46`）；所持的值标为当前；所持的是已结束、已归档的迭代时它不是选项，以名字（经 `usePlanningNames`，含说明）显示为当前；选中即关，选所持的值不发请求，选“无迭代”发 `{cycle_id: null}`。
- **改模块**：`shift+m`，列表是可加入的模块（`moduleOptions`）加所持的已归档模块（勾着，只为移出，同下拉框 `core/components/dropdowns/module/base.tsx:137-149`）；每选一项发一个增或减（`{modules: {add: [id]}}` 或 `{modules: {remove: [id]}}`），列表不关、勾选随 nerve 的确认。
- **读**：页打开时读该项目的选项（T6 定用 `useCycleOptions`/`useModuleOptions`（`core/hooks/use-planning-options.ts`）还是打开的菜单的读加选项的纯函数，台账写明理由）；读的失败说原因并可重试（同下拉框的 `DirectoryReadError`），换项目、换人之后旧的回答不进新页（G19）。
- **写**：命令的写在 `mutation-commands.ts` 的守卫（`:28-48`：捕获的范围、`sessionGuard`、写队列轮到时再看 `canEdit`、拒绝经 `useRefusalToast`）之内，经 `IssueChange` 的增减：一个 `change(change: IssueChange)` 即 `update(held => changeBody(change, held.module_ids))`（`core/store/issue/issue-write-fields.ts:58-70`），模块集合在轮到时以所持的算出（P7a R3），菜单不给整集合。命令的类型让整集合赋不进去（`IssueChange` 的 `module_ids?: never`）。
- **文案**：Plane 的键 `power_k.contextual_actions.work_item.add_to_cycle`、`add_to_modules`，`power_k.page_placeholders.update_work_item_cycle`、`update_work_item_module`（en 与 zh-CN；en 的说法由 T6 照 nerve 的语义定，例如“Change cycle”“Change modules”）。
- 与 Plane 的差异（有“无迭代”、不给已结束的、模块的已归档值只可移出）登记在 `plane-diff.md`。

### 2.6 快捷操作的“恢复”与归档页的行菜单（R7）

- **A17**：迭代、模块的快捷操作（`core/components/cycles/quick-actions.tsx:68-102`、`modules/quick-actions.tsx:68-106`）的“恢复”在途时禁用：在途改为组件的状态（今天是 ref，项不重绘），工厂的恢复项收 `disabled`（同归档项，`core/components/common/quick-actions-factory.tsx:48-69`），按钮菜单与右键菜单都照它渲染（`disabled={item.disabled}`），键盘选中禁用的项不发请求；回答之后（成功或被拒）恢复可用。
- **A18**：归档页的行菜单（`issue-layouts/quick-action-dropdowns/archived-issue.tsx:58-71`）不再给工厂两个空的设置函数（`:64-65`）：它用的项（恢复、新标签页、复制链接、删除，`helper.tsx:272-283`）不打开对话，它的输入类型收窄为不含 `DialogOpening`（`helper.tsx:64-74`）的那一份，给了就编译不过。

### 2.7 移动端页头（R8、R9）

- **共用的件**：`core/components/issues/` 下一个按 store 类型的页的移动端页头（与 `HeaderFilters`（`filters.tsx:45-86`）同一组读与写：`usePageIssueFilters(storeType)`、`useIssuesActions(storeType).updateFilters`、`ISSUE_STORE_TO_FILTERS_MAP[storeType]`（`@nerve/constants` 的 `issue/filter.ts:236-240`，已去掉本页的分组））：`md:hidden` 的一行，布局（列表、看板、日历，`MobileLayoutSelection` 带 `activeLayout`）与“显示”面板（`DisplayFiltersSelection`）；只在本页的偏好读到之后给出（调用者给 `preferencesShown`，P6b R3），读失败时不给控件（同桌面的页头）。没有筛选的开关与分析（Plane 也没有筛选；nerve 没有分析）。
- **项目页**（X2）：`R/issues/(list)/layout.tsx` 改传共用的件（`preferencesShown` 同桌面页头的 `core/components/issues/header.tsx:52`），手写的 `R/issues/(list)/mobile-header.tsx`（98 行）删去；`issue-layouts/filters/preferences-controls.test.tsx` 里挂它的三例（`:218`、`:229`、`:251`）改挂共用的件，断言不改（基线同数），另加“偏好读到之前不给控件”。
- **迭代页、模块页**（A15）：两页的 layout（`R/cycles/(detail)/layout.tsx:22-33`、`R/modules/(detail)/layout.tsx:23-34`）传共用的件，`preferencesShown` 取自 `usePlanningPageHeader`（`issue-layouts/use-planning-page-header.ts:44`）；“显示”面板因按页的常量没有本页的分组；布局的改动写本页的偏好（`/me/cycles|modules/{id}/preferences`）。
- **顶行**（R9）：迭代页的侧栏开关与快捷操作移出 `hidden … md:flex`（`R/cycles/(detail)/header.tsx:143`），与模块页（`R/modules/(detail)/header.tsx:145-164`）一样在各宽度都给；列表的控件与“新建工作项”两页照旧从 `md` 起。窄屏上迭代的侧栏关上之后可再打开。与 Plane（迭代页的开关与快捷操作在 `md` 以下藏起）的差异登记在 `plane-diff.md`（`:382`、`:401` 两行改写）。

### 2.8 遗留（R10、R11）

- **A19 分组时页头的数目**：项目页（`core/components/issues/header.tsx:49`）与归档页（`R/archives/header.tsx:61`）的数目不再读 `""` 组（只在不分组时有，`core/lib/paged-groups.ts:169`），改读当前列表的总数，且只在当前列表是本页的时（范围是 `project`、项目是本页的项目，同 `pageTotal` 的规则，`core/store/issue/helpers/planning-page-issues.store.ts:29-51`；项目页的 store 今天以 `isPreferenceQueryCurrent` 核范围，`core/store/issue/project/issue.store.ts:10-16`）：分组时照样显示 nerve 的总数，别的项目的列表还在时不显示。读法放在 store 一侧（各自的 store 一个按页的读，不在组件里比范围）。
- **A20**：删去 `project_cycles.status.days_left`（en 与 zh-CN 的 `project.json:49`）与从不被读的 `CYCLE_STATUS[].i18n_label`（`@nerve/constants` 的 `cycle.ts:9`、`:17`、`:25`、`:33`、`:41`）。
- **A21**：`issue-layouts/roots/fake-planning-page.tsx` 的模块级 `opened`（`:204`、`:209`）在这一页的标签关上时清空（只清它自己：关上的是当前打开的那一页时）；加一个替身一级的测试（关上之后取当前的页即报“没有打开的规划页”），文件仍在 400 行以内。
- **“添加已有”打开即聚焦搜索框**：`core/components/core/modals/existing-issues-list-modal.tsx:190-196` 的搜索框在非移动端（`isMobile` 为假，同两个下拉框的 `focusSearchOnOpen`，`core/components/dropdowns/cycle/base.tsx:166`）打开时取得焦点，写法照两个表单的 `data-autofocus`（`core/components/cycles/form.tsx:133`）；组件测试核打开时 `document.activeElement` 是搜索框、移动端不是。M3 交接 §3 的回执。

### 2.9 设计 §8 的四个视图文件（R12）

- `R/views/(list)/page.tsx` 的 `currentProjectDetails`（`:41`、`:70`）改为 `getProjectById(projectId)`（`:45` 已有），`allowPermissions([ADMIN], PROJECT)`（`:47`）补上 slug 与项目；`core/components/views/view-list-item-action.tsx:53-56`、`views-list.tsx:35-38` 的 `allowPermissions` 补上 slug 与项目（组件从路由参数或 props 取，今天已有 `useParams()`）；`R/views/(detail)/[viewId]/header.tsx` 没有命中（它经 store 的 getter `projectViewIds` 读路由，属 M7），只加入守卫。
- 四个文件加入 `project-address-route` 的文件表并带样例；视图的新建、修改（`core/components/views/modal.tsx:39-70`）与删除（`delete-view-modal.tsx:44-58`）之后的跟进改经 `followInSession`（`core/lib/in-session.ts:40-53`，写法照 `cycles/quick-actions.tsx:80-93`）。
- 不改：视图的 store 与读（今天没有东西填它，视图页一直在加载，M7）、`projectViewIds`、`R/views/(list)/mobile-header.tsx` 的 `projectMemberIds`、视图的文案。

### 2.10 读者与写的一览

| 处 | 读 | 门 | 写 |
|---|---|---|---|
| 打开一个迭代/模块（§2.2） | `PROJECT_CYCLES`/`PROJECT_MODULES` | 读门 `listed`；为假时说原因 | — |
| 搜索的两组（§2.4） | 同上 | 入口规则 | — |
| 改迭代、改模块（§2.5） | 选项的读（页打开时）、名字（`usePlanningNames`） | `canEdit` 与开关 | `IssueChange` 的增减，守卫之内 |
| 快捷操作的恢复（§2.6） | — | 既有 | 既有，在途时禁用 |
| 移动端页头（§2.7） | 本页的偏好 | `preferencesShown` | 本页偏好的修改（既有的 `updateFilters`） |
| 页头的数目（§2.8） | 当前列表的总数 | 当前列表是本页的 | — |

### 2.11 守卫、文案与告警

- `project-address-route`：§2.1 的项目一级一件、§2.2 改的两个菜单（已在表里的照旧）、§2.4 改的搜索两件、§2.5 的新件、§2.7 的共用件与两个 layout、§2.9 的四个视图文件，都在表里并带样例（命中与不命中各至少一例）；删去的 `R/issues/(list)/mobile-header.tsx` 留在表里阻止复活。用 `useSessionSWR` 的新文件加入 `.oxlintrc.json` 的 swr 限制。
- 新键（en 与 zh-CN 同一提交，没有一个键是另一个的前缀）：七个菜单的空文案（§2.3）、两个命令与两个占位（§2.5）；删去的键：`project_cycles.status.days_left`（§2.8）。读到的已有键：`errors.project_archived`、`errors.planning_*`、`common.cycles`、`common.modules`、`cycle.no_cycle`、`power_k.search_menu.no_results`。
- 告警上限 44（`web/apps/web/package.json`）：新文件不加告警，清掉告警的提交同时下调；本段改过的文件 oxlint 0 告警（M1 交接的回执，T14 以收据核对）。
- knip：新文件在第一个产品读者出现的 Task 里建立；只给测试读的导出不算读者（逐 Task 审查执行）。

### 2.12 e2e（R13、R14）

一个新文件 `e2e/stories/planning/g19-work-item-side-page.spec.ts`（标题 `G19 (page): …`，合 `e2e/playwright.m6-pages.config.ts` 的 `testMatch` 与 `grep`，三种引擎在本机跑；CI 照常只跑默认配置）。世界经接口建立（新项目的两个开关默认关闭，故事先打开）；帮手从原处导入不复制（`fixtures/power-k.ts` 的 `openCommands`、`selectCommand`，`planning-list-pages.ts` 的 `cycleListWorld`、`moduleListWorld`、`openListPage`、`bodiesSent`，`settings-pages.ts` 的 `answerTo`，`cycle-pages.ts` 的 `recordApi`、`dismissToasts`，`dialogs.ts` 的 `modalDialog`、`focusInDialog`）。

| 故事 | 覆盖 | 本机 chromium（指引） |
|---|---|---|
| 命令面板 | 项目里迭代 Sprint（进行中）、Ended（已结束）、模块 Alpha、Beta；在项目页打开命令面板搜 “spr”：“Cycles”一组恰有 Sprint（行有项目标识），选中即到 Sprint 页（以本页独有的内容为锚）；工作项 W 的页上打开命令面板 → “改迭代”：列表以“无迭代”开头、没有 Ended，选 Sprint：只一个 `PATCH`，请求体恰为 `{cycle_id: Sprint}`，页上显示 Sprint；“改模块”：选 Alpha 发 `{module_ids: [Alpha]}`、列表不关，再选 Alpha 发 `{module_ids: []}`（请求体在轮到时算） | ≤ 3.5 秒 |
| 移动端页头 | 390 × 844：迭代页（打开之后再缩窄，`openListPage` 在宽屏关侧栏）的布局与“显示”在本页偏好的回答之后出现；选“看板”只发本迭代偏好的一个 `PATCH`；“显示”的分组没有 “Cycle”；侧栏开关可见、可再打开侧栏；模块页的“显示”分组没有 “Modules” | ≤ 3 秒 |
| “添加已有”的焦点 | 模块页的空态点“添加已有”：弹窗打开时焦点就在搜索框（`toBeFocused`），不先点它即可以方向键选到一项 | ≤ 1.5 秒 |

**写法**照 P7b 阶段记录 §5 的教训（打开列表之前先等上方的内容与过渡，M6-M8-13；扣住回答、读请求体的路由在页面载入之后才装，WebKit、Firefox 不装路由而以 `request` 事件记；弹窗自己的读以 `answerTo` 等它；e2e 的变异落在被证的一步；变异之后以干净的源重建；不用 `page.reload()`；关通知只用 `dismissToasts`；e2e 的 lint 上限是 0，按键序列不在循环里 `await`）。窄屏以 `page.setViewportSize({ width: 390, height: 844 })`（C2、C12 的先例），不用 `devices`（会改 UA 与 `isMobile`）；命令面板的快捷键经 `fixtures/power-k.ts`（各引擎的修饰键已在其中）。

**时长**（R14）：本文件本机 chromium **≤ 8 秒**（P7b R13；三故事的分配只是指引，约束的是合计）；超过时先压缩（把已有单测覆盖的一步移出故事），仍超就停下报告主控。数据点：

| 数据点 | 值 | 出处 |
|---|---|---|
| 记下的最慢的 runner 之前的锚：main CI 43549053092 的 e2e 作业 | 27 分 41 秒 = 1661 秒 | P7b spec §2.14 |
| 其后加的故事（本机 chromium）：I4 的拆分、P7a、P7b 的页面故事、P7b 的 C4/C8 关系一支的等待 | 5 + 6.5 + 6.9 + 约 0.9 = 19.3 秒 | P7b spec §2.14、阶段记录 §5、[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13) 的回执 |
| P7b 的最终分支 CI 43702923005 的 e2e 作业（已含全部） | 27 分 50 秒 = 1670 秒，至今最慢 | 台账 setup，T1 以收据核对 |
| P7b 合入后的 main CI 43706438924 attempt 1 的 e2e 作业 | 23 分 42 秒 = 1422 秒 | 同上 |

T1 照 P7b spec §2.14 的公式重算并写进台账；T10 的检查点与 T15 推送之前再各算一次：
- **估算**：基线 1422 秒（最后一次绿的 main CI）加新文件 chromium 时长 X × 1.67–2.25、两个 worker 摊开（8 秒约加 7–9 秒），约 23 分 49–51 秒。
- **上界**：S + X × f，f = 3.6–4.1；S 取“1661 + 19.3 × f”（约 1730.5–1740.1 秒）与已含这些故事的各次实测（1670 秒、1422 秒、T10 检查点的分支 CI）中的最大者；X = 8 秒时上界约 29 分 19–33 秒。余量 X ≤ (1800 − S) / f，约 **14.6–19.3 秒**（关系一支的等待算作加进来的故事时长，取保守的一端；不计它时约 15.5–20.2 秒，P7b 阶段记录 §5）。
- **校准点**：T10 检查点的分支 CI 实测的 e2e 作业时长（那时还没有本段的页面故事），慢的 runner（×1.31，M6-M8-01）另算一次作观察（不是停下的条件，负责人已知 e2e 的作业结构迟早要他决定）。
- 估算、上界或任一次实测（T10、推送后的分支 CI）超过 30 分钟时，**停下报告负责人、不推送**；e2e 的分片、期限与作业结构不经负责人不改。

## 3. 局部裁定

- **R1 范围（主控裁定 H1）**：P7c 是 P7 的第三段（P7a spec R1）：P7b 的回填与余量、先拆导航的命令、规划入口的规则与菜单的说明、菜单的空文案、搜索的结果、两个工作项命令、A17/A18、共用的移动端页头（项目页）、两页的移动端页头与顶行（检查点）、A19–A21 与“添加已有”的焦点、四个视图文件、页面故事、文档与回执、门禁与合入，15 个 Task。理由：两份调研估命令面板约 5 个 Task、移动端与遗留约 4 个、X1 一个（`research/*.md` §6、§8），在设计 §12 的“约 16”之内；依赖的次序是先拆再改命令、入口规则在搜索之前、共用的件在两页之前、产品在故事之前。代价：T6、T7 各一个命令，单个不大，分开是为各自的变异与收据清楚。
- **R2 规划入口一条规则（主控裁定 H2）**：命令与搜索的两组以入口规则（ADMIN/MEMBER、项目没有归档、开关打开）判断，侧栏两张声明表是同一规则不改；打开的菜单门为假时说原因，不转圈。理由：今天的命令不看归档，归档的项目里打开的菜单永远转圈（re-65 M7、P5a §5、P6a §5）；读门让访客读（P6a R3），但 Plane 与 nerve 的侧栏都不给访客迭代、模块的入口，入口与读门是两件事。不取“命令照读门”（会给访客入口，与侧栏不一）。代价：一个纯函数、两处命令改调、两个菜单的说明；侧栏的声明表与入口规则两种写法（理由：侧栏是声明表，一个谓词塞不进它的 `access`）。
- **R3 先按层级拆 `navigation/commands.ts`（主控裁定 H3）**：理由：466 行已过约 400 的上限，本段要改它（E15 的先例：要加代码的到上限的文件先拆）；按层级拆正好让项目一级的一件进守卫。不取只改不拆。代价：一个只搬不改的 Task。
- **R4 空文案经构造件的键（主控裁定 H4）**：七个菜单都改。理由：构造件的契约从字面量改为键，所有调用者随之改，一个契约；英文字面量违背 i18n 的规则（P5a §5 点名迭代、模块两个，其余五个同一个件）。不取只改两个（构造件的契约要两种）。快捷键说明的字面量不在构造件里，交 P8。代价：七个新键。
- **R5 搜索的结果（主控裁定 H5）**：只在项目范围、本地过滤、未归档的、入口规则。理由：设计 §8 `:202` 与 M4 设计 `:40`、`:671`（“在已取得的项目列表里本地过滤”，工作区范围属 M7）；nerve 没有跨项目列迭代的接口。不取 Plane 的服务端搜索（要新接口，R15）。代价：只在项目范围；登记差异。
- **R6 两个工作项命令（主控裁定 H6）**：给，以 nerve 的选项规则（P7a 的下拉框）与 `IssueChange` 的增减。理由：设计 §8 `:202` 的“切换”与 §12 的问句；写的路径（P7a 的 `changeBody`）与选项的规则都已在，命令只是另一种入口；命令面板是键盘优先的入口，与下拉框同一规则才不会两处说两种话。不取 Plane 的选项（列已结束的、没有“无迭代”：nerve 拒绝加入已结束的迭代）。代价：两个页类型、四个键、两组单测；登记差异。
- **R7 A17、A18（主控裁定 H7）**：理由：P5a §5（re-65 F3 n1）与 P5b §5；在途的 ref 不让项重绘，禁用要状态；空的设置函数是死参数，类型收窄让它给不进来。代价：两个快捷操作各一个状态。
- **R8 一个共用的移动端页头（主控裁定 H8）**：理由：项目页的那份手写了 `HeaderFilters` 已按 store 类型做好的读与写（`research/mobile-carryovers.md` §1.2），三页一份；P6b 的共用规则（只差领域的件两域共用）。不取 Plane 的每页一份（三份几乎相同）。代价：项目页的移动端测试改挂共用的件。
- **R9 两页的窄屏顶行一条规则（主控裁定 H9）**：理由：迭代页在 `md` 以下藏起侧栏开关，侧栏默认打开、关上之后再也打不开（`research/mobile-carryovers.md` §1.3）；模块页的开关与快捷操作各宽度都给。不取照 Plane（同一缺陷）。代价：与 Plane 的差异登记。
- **R10 页头的数目按范围读（主控裁定 H10）**：理由：同 `pageTotal` 的规则（P5b R10 的 G19：换项目时不显示别的项目的数）；Plane 分组时也显示总数。不取直接读 `total`（换项目的瞬间显示上一个项目的数）。代价：两个 store 各一个按页的读。
- **R11 A20、A21 与焦点（主控裁定 H11）**：理由：删掉没有读者的键与字段（项目的规则）；替身的 `opened` 是 re-71 §4 M1 的遗留；“打开即聚焦搜索”是下拉框的规则（M2/M3 交接），移动端不聚焦免得弹出键盘。代价：无。
- **R12 X1 的范围（主控裁定 H12）**：地址的读与守卫、视图写之后的跟进；视图的 store、读与页面内容属 M7。理由：设计 §8 `:203` 与 M3 交接 §1、§4 的分派；视图页今天读不到数据（store 没人填，`research/palette-views.md` §3），改它的内容要视图的接口（M7）。代价：改的是今天读不到数据的页的地址读，等 M7 接上时已合规则。
- **R13 e2e 一个新文件（主控裁定 H13）**：理由：P7a、P7b 都以新文件单独量时长；既有的 G19 文件已近 320–345 行。代价：一个文件三条故事，其余由单测承担（A17、A18、A19、A21、X1、迟到的回答）。
- **R14 e2e 的余量与检查点（主控裁定 H14）**：余量取保守的 14.6–19.3 秒，本段 ≤ 8 秒；检查点在 T10 之后；越过 30 分钟停下报告负责人（§2.12）。
- **R15 只改网页（主控裁定 H15）**：不改 `server/`、`api/`；写 spec 时没有找到网页上处理不了的契约缺口。
- **R16 行数与质量（主控裁定 H16）**：手写文件约 400 行以内（先拆、只搬不改、留基线）；没有开关、桩或兼容；类型来自 OpenAPI 生成的类型；守卫只收紧（§2.11）；M1 交接本段只给改过的文件 oxlint 0 的回执，死成员的收敛在 P8。

本段的裁定都是局部决定：R1–R16 是主控在写 spec 之前的裁定（H1–H16），都在已批准的设计与 P5a–P7b 的先例之内，不改接口、不改服务端、不跨后端模块；**没有须负责人决定的事**（e2e 的作业结构只在余量不够时才提出，R14）。

## 4. 验收标准

- [ ] A1 拆分：`navigation/commands.ts` 按层级拆为两件，只搬不改，各在约 400 行以内，命令的输出逐项相同（基线同名同数），项目一级一件在 `project-address-route` 里。
- [ ] A2 入口规则：纯函数与原因函数各有单测（ADMIN、MEMBER 给，GUEST、没有角色、项目已归档、开关关闭不给；原因各一码）；四个导航命令改调它（归档的项目里不给，单测与变异）；两个打开的菜单门为假时说原因、不转圈（单测：归档的项目、开关关闭；变异回到转圈即失败）。
- [ ] A3 空文案：构造件只收键，七个菜单各有新键，默认 `power_k.search_menu.no_results`；en 与 zh-CN 同步。
- [ ] A4 搜索：项目范围的两组（本地过滤、未归档、含已结束的迭代、入口规则、行有项目标识、选中即转到）；工作区范围没有；`/browse/:workItem` 上自己读；读失败不挡别的结果（单测与变异）。
- [ ] A5 改迭代：选项（“无迭代”在前、不给已结束与已归档的、所持的标为当前、所持的已结束的以名字显示为当前）、选所持的不发请求、请求体 `{cycle_id}`、守卫（换项目、换人、不可改时不发、拒绝说原因）；`shift+c`；单测表与变异。
- [ ] A6 改模块：选项（可加入的加所持的已归档的、只可移出）、每选一项一个增减、请求体在轮到时以所持的算出、列表不关；`shift+m`；类型让整集合赋不进去（类型测试）；单测与变异。
- [ ] A7 A17、A18：恢复在途时禁用（按钮菜单与右键菜单、键盘），回答之后恢复；归档页的行菜单不给设置函数（类型测试）。
- [ ] A8 移动端页头：共用的件（布局带当前的、显示面板按页的常量、偏好读到之前不给、读失败不给）；项目页改用它、手写的删去、既有的移动端测试同数；迭代页、模块页接上（写本页的偏好、没有本页的分组）；迭代页的侧栏开关与快捷操作各宽度都给（两页页头的测试）。
- [ ] A9 A19：项目页与归档页的数目分组时显示 nerve 的总数，别的项目的列表在时不显示（store 的单测与页头的测试，变异读 `""` 组即失败）。
- [ ] A10 A20、A21、焦点：键与字段删去（i18n 守卫过）；替身关上即清空（替身一级的测试）；“添加已有”打开即聚焦搜索框、移动端不聚焦（组件测试与变异）。
- [ ] A11 X1：四个文件的地址读与守卫（`node tools/keywords.mjs` 过、样例）；视图写之后的跟进经 `followInSession`（单测：会话换了之后不跟进）。
- [ ] A12 页面故事：`g19-work-item-side-page.spec.ts` 三条故事在三种引擎通过，本机 chromium 合计 ≤ 8 秒（台账以 `durations.py` 记各引擎的时长）。
- [ ] A13 守卫与文档：`project-address-route`、swr 限制、告警上限、knip、关键词（phase `M6/P7`）、i18n；P7b 的回填（阶段记录 §7、spec A15、设计 §14、M6-M8-01 的数据点）与余量的重算；交接回执（M3 §1/§3/§4、M4 M6-02 的“分组时页头的数目”、M1 的改过的文件 oxlint 0）；差异清单、前端改动清单、设计 §8/§12/§14、M8 交接的相关条目。
- [ ] A14 七项本地门禁（`make test` 照负责人 2026-11-11 的规则）、整分支审查的修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P8 的 T1 勾选）

## 5. 后续责任

**P8**：归档页的迭代、模块词表（P7b R10）；[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)；新的控件（菜单、移动端页头、顶行）的焦点轮廓随最终矩阵核对；M1 交接的死成员收敛；命令面板快捷键说明的英文字面量（R4）；P8 的 T1 回填本段的阶段记录 §7 与 A14。

**M7**：工作区范围的迭代、模块搜索；视图的 store、读与页面（R12）。

**M8**：[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 多 P7b 的分支 CI、main CI（T1 记）与本段 T10 检查点的数据点；e2e 的余量至此约剩 6.6–11.3 秒（本段用满 8 秒时），之后的段再加故事前要负责人对作业结构的决定（R14）。

**负责人**：e2e 的作业结构（分片）、期限与超时。

## 6. 风险

| 风险 | 处理 |
|---|---|
| 拆分碰坏命令 | T2 只搬不改、基线逐文件同数；命令的 id 与快捷键不变 |
| 两个新快捷键与既有的冲突 | `shift+c`、`shift+m` 只在工作项上下文；既有的是 `c`、`m`、`cmd+shift+c`（`research/palette-views.md` 核对）；T6、T7 以快捷键的单测核对 |
| 命令面板的选项读在 `/browse/:workItem` 上没有项目的外壳 | 页打开时自己读（§2.5），与下拉框同一份 store；换项目、换人之后的回答不进新页 |
| 共用的移动端页头改了项目页的行为 | 项目页的既有移动端测试断言不改（基线同数）；唯一的变化是偏好读到之前不给控件（X2，同桌面） |
| 页头的数目在换项目的瞬间显示旧数 | 按范围读（R10），单测覆盖 |
| 视图页今天读不到数据 | 只改地址读与跟进（R12），内容属 M7；单测以 store 的替身核 |
| e2e 时长 | 本段 ≤ 8 秒；T1 重算、T10 校准、T15 估算与上界；越过 30 分钟先报告负责人、不推送 |
