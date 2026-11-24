# M7 / P6a plan：网页的项目视图——列表、视图页、另存与更新视图、命令面板的视图

> spec：[P6a spec](../specs/P6a-web-project-views.md)（§2 是合同，§3 的 R1–R27 是裁定、逐条对应台账 `rulings.md` 的 C-1–C-27，**L!** 是 R1、R7、R8、R9、R10、R13、R14、R16、R21 与 R29；R28–R30 是主控对起草者 O-1、O-2 与另存默认名的补充裁定，R29 经主控确认为 L!；§8 是补充裁定（O-1 取 (b)、O-2 改为在途停用、O-3–O-5 同意、O-6 两处也停用）；本 plan 已照补充裁定与预检 [re-96](../../../reviews/re-96-M7P6a预检.md) §6 的处理修正）。基线 main `47bd860b`（`47bd860ba104145a37a66687c12d6db7b49658b4`）。worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m7-p6a`，分支 `codex/m7-p6a`，台账 `L` = `.superpowers/sdd/P6a-web-project-views/`。缩写同 spec（`C`、`P`、`V`、`VC`、`PKG`、`E`、`K`、`HO`、`D`）。

**与调研草案（`research/scope.md` §7）的对照**：T1–T14 同草案；检查点 0 从 T3 之后移到 T5 之后（引擎改动之前，spec R25、O-5）；检查点 2 在 T10 之后（命令面板与删除完成，产品代码完成，C-25）；`projectMemberIds` 的删除并进 T4（两个读者都在列表页）；`getTabIndex` 的删除并进 T5（最后的读者是表单）；`create_view` 的门并进 T3（入口规则）。

## Global Constraints

**分支与提交**
- 主目录永远在 main。提交用详细中文，前缀 `feat/fix/docs/test/refactor(M7/P6a)`，结尾一行 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。提交只在本 Task 的 lint 过了之后，不与它串在一条命令里，用显式路径（`git commit -F <msg> -- <paths>`）。
- 实现代理不推送、不合并；检查点由主控推送。
- 关键词的 phase 由 T1 改为 `M7/P6`（P6a、P6b 共用）；不延长任何 `until`。
- 本段不改 `server/`、`api/`、`deploy/`、`.github/`，不改设计文档。超出 spec 的契约或架构改动先停下报告。

**只读与禁区**：`plane/`、`refer/` 只读，不在其内跑 git。不 stash、`git clean`、`reset --hard`、强推；还原变异只经 `mutate.py`。不动别的项目的容器。容器套件一次一个：`make test`、`make test-shard`、`make e2e` 与 Playwright 的配置不并发。

**代码规则**（总体设计 §7.2、§7.7）
- 手写文件约 400 行以内，新代码进新文件，按关注点分（spec §2.12 的表）。新文件的版权头 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（`View`、`ViewCreate`、`ViewUpdate`、`ViewFilters`）；service 不转换；store 只经自己的根找兄弟；SWR 只经 `useSessionSWR`、只用它的三个选项（`C/lib/use-session-swr.ts:38`）；新代码不读 router store 的地址（不用 `projectMemberIds`、`currentProjectDetails`、不带 slug 与项目的 `allowPermissions`）。
- 视图的写只经视图 store 的队列，请求体在轮到时算（`docs/v0/v0-design.md:638`）；`await` 之后的提示、导航、关弹窗一律经 `followInSession`；拒绝经 `useRefusalToast`；不 `console.error`。
- 删掉不用的代码，不加开关、桩、只给测试用的钩子或兼容别名。
- 不放宽 `K` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **行数**（`wc -l` 于基线）：测试与故事每份不过 400 行；`C/store/issue/root.store.ts` 329（T6 只加构造，不超过 340）、`C/hooks/use-issues-actions.tsx` 347（T6 只加一个导入、一行无条件的调用、两个 `case`——`PROJECT_VIEW` 与今天落进 `default` 的 `WORKSPACE_DRAFT`——与 `default` 的穷尽，视图页的命令进新文件）、`C/hooks/use-issue-list-mount.ts` 63（T6 只改 `page` 一段）、`C/store/issue/helpers/page-preferences-filter.store.ts` 292（只在 T8 加 `confirmed`，不超过 300；显示的换算不导出，R28）、`C/components/work-item-filters/filters-hoc/project-level.tsx` 212（T8 之后不超过 220，放不下的进新 `use-project-view-saving.ts`）、`C/components/power-k/ui/modal/search-menu.test.tsx` 365（不碰，搜索组的格进新文件）、`C/store/member/project/project-member.store.test.ts` 381（只删）、`C/components/inbox/filters/menu.test.tsx` 327（只删一个字段）、`C/store/favorite.store.ts` 391（只删）、`E/fixtures/cycle-pages.ts` 388（只导入）、`E/stories/project/p8-project-preferences.spec.ts` 397（只照允许清单加一行等待）。新文件每份不过 400 行；store 的写层（`view-changes.ts`）不过 220 行。

**knip 与第一个读者**（新文件、新导出在它第一个读者出现的 Task 里建立；测试数据的导出以第一个读它的测试为准）：
- T2：`ViewsService`（读者：`ViewStore`）；`ViewStore`、`ViewChanges`（读者：`RootStore`）；`useViews`（读者：T2 自己的 hook 测试不算——第一个产品读者在 T4，所以 `use-views.ts` 在 T4 建）。
- T3：`viewEntryOn`、`viewsCreatableBy`（读者：侧栏、页签栏、命令）。
- T4：`use-views.ts`（列表页）；`VC/page-state.tsx`、`VC/actions-host.tsx`（列表布局；T4 的宿主只有删除，预检 re-96 I5 ①）；列表的排序与筛选的纯函数（读者：`page-state.tsx`）。
- T5：“Add view” 的合同默认显示与属性的常量（读者：新建的弹窗）；宿主的新建与编辑（读者：行的“编辑”、“Add view”、空态的“新建”）。
- T6：`ProjectViewIssues`、`ProjectViewIssuesFilter`、`ViewDraftService`（读者：`IssueRootStore`）；`use-project-view-issue-actions.ts`（读者：`use-issues-actions.tsx`）；`use-project-view-preferences-follow.ts`（读者：上一个）。
- T7：`ProjectViewLayoutRoot`、`VC/use-view-page-read.ts`（读者：视图页）；`ProjectViewMobileHeader`（读者：视图页的布局）。
- T8：`confirmed`（读者：HOC）；`ViewStore.isUpdating`（读者：视图页头；R29 的成员随第一个读者建，不在 T2 建）；页头五个组件与筛选行的可选 `disabled`（读者：视图页头、`ProjectViewLayoutRoot`）；`DraftBusyContext`（读者：`HeaderColumn`、日历的选项下拉、`ProjectEmptyState`，提供者 `ProjectViewLayoutRoot`）；`PLANNING_PAGE_DOMAINS` 的 `viewTerms`（读者：`planning-page-layout-root.tsx`）；`use-project-view-saving.ts`（若有，读者：HOC）。
- T9：`use-view-search.ts`（读者：`search-menu.tsx`）。
- 没有“成员先于读者”的例外。删掉某个导出最后一个读者的 Task 同一提交删去它（T4 的 `projectMemberIds`，T5 的 `getTabIndex` 全套，T10 的旧层）。knip 查不出只被测试读的产品导出与包的桶文件导出：逐 Task 的审查与死成员的整份比对兜底。

**告警上限**：`lint-cap` 要求 oxlint 的告警数恰好等于上限（web `web/apps/web/package.json:10` 现为 18；e2e、`@nerve/constants` 为 0）。新文件不得新增告警；删掉的文件若带告警，同一提交照实改上限并留收据（`t<N>-oxlint-count`）；T14 只核对。

**文案**：新键只有 spec §8 O-3 的清单（主控已同意：`project_views.add_view`、`project_views.not_found.{title,description,action}`、`project_views.action.{create,update,delete}.success`、`common.filter_options.{save_view,update_view,confirming}`）；en 与 zh-CN 同一提交；文案键写完整的字面量（`tools/i18n-keys.mjs` 才认得），不许一个键是另一个键的前缀。别的一律用已有的键（`views`、`common.filters`、`common.clear_all`、`common.item_options`、`common.search.label`、`common.filter_options.custom`、`common.filter_options.unavailable`、`common.display`、`common.cancel`、`common.creating`、`common.updating`、`common.retry`、`copy_link`、`common.link_copied`、`toast.success`、`toast.error`、`view.create.label`、`view.update.label`、`project_views.delete_view.*`、`project_view.sort_by.*`、`project_empty_state.views.*`、`disabled_project.empty_state.view.*`、`power_k.creation_actions.create_view`、`power_k.menus.no_views`、`inbox_issue.filters.created_at`、`errors.navigation_view_not_found`、`errors.navigation_views_disabled`）；实现中觉得要新键先记台账并报主控。

**守卫**
- `project-address-route`（`K:2334`）：每个新建或改写的、读项目或角色的文件收入规则的文件名单并带 hit 样例；删掉或改名的文件同步改样例（hit `K:2631-2637`、miss `K:2797-2803` 里的视图文件）；改后跑 `node tools/keywords.mjs`。
- `.oxlintrc.json` 的 swr 名单（`:121-215`）：每个新的取数文件收入（含 T7 的 `VC/use-view-page-read.ts`），删掉的文件移出。
- `copy-hook` 的视图例外（`K:3350-3356`）在 T4 删；`old-view-urls` 在 T10 加（phase `M7/P6`）。

**允许改的已有测试与夹具**（改之前在台账逐条写明，只照这里写的改法；其余断言是回归，不改；编译跟随只为编译、不改断言，逐处记台账；任何一条要改断言而这里没写，先停下报告）：
- T2 `C/store/root.store.test.ts:106`：`toHaveLength(20)` 改 21（新 store 与旧 store 并存）；T10 改回 20。
- T3 `C/components/power-k/config/navigation/commands.test.tsx`（209）：`:120`、`:138` 两行不变；只追加角色的格（成员 T、访客 T、没有角色 F）。`project-commands.test.tsx`（57）、`C/components/workspace/sidebar/project-navigation.test.tsx`（119）、`C/components/navigation/use-navigation-items.test.tsx`（51）：只追加视图的格。`create_view` 的格加在 `C/components/power-k/config/creation/` 旁的新测试里。
- T4 `V/(list)/page.test.tsx`（171）：测的是旧列表页与旧 store（`:143`、`:155`、`:164` 三组），随页面重写删去，新列表页的格在新测试里；台账写明旧的每一格由哪一格接上。`VC/follow-ups.test.tsx`（162）：`DeleteProjectViewModal` 一组（`:132-162`）随删除进宿主删去（新的格在宿主的测试里）；`CreateUpdateProjectViewModal` 一组（`:87-131`）留到 T5。`C/store/member/project/project-member.store.test.ts:102` 改为只比 `getFilteredProjectMemberIds`，`:113` 删；`C/components/inbox/filters/menu.test.tsx:64` 删 `projectMemberIds` 字段。`C/components/workspace/views/quick-action.tsx:52`（P6b 的文件）：`useViewMenuItems` 的实参去掉 `view`、`workspaceSlug`（编译跟随，行为不变）。
- T5 `VC/follow-ups.test.tsx`：整份删去（弹窗的跟进在新弹窗的测试里，台账写明旧格 → 新格）。`C/components/dropdowns/layout.test.tsx`（110）：随 `LayoutDropDown` 删去（R28：表单不再有布局，它没有别的产品读者；键盘的性质由 H20 (page) 在视图页的布局切换上接上）。
- T6 `C/hooks/use-issue-layout-store.test.tsx:75-114`：`viewId` 一格期望 `PROJECT_VIEW`（“在选定的项目建、不取列表”的断言不变）；`:125-131` 与 `C/hooks/store/use-issues.test.ts:26-31` 的拒绝表去掉 `"PROJECT_VIEW"`。
- T7 `P/domain-pages.test.tsx:114-163`（“project view has one detail reader”，钉旧 store、不列工作项、没有 “Add work item” 与 “Display”）：删去，性质由视图页的新测试接上（恰一个视图的读、列表的查询、矩阵）；文件里别的格不改。
- T7 `E/stories/smoke/s2-issue-list-removal.spec.ts`（171，预检 re-96 I6 ①，与视图页的重写同一提交）：照 spec §2.11 删 `:16` 一行与随之死掉的分支（`:81` 的 `detail`、`:82-85` 的项目一支、`:86` 的 `PROJECT`/`PROJECT_MEMBER`、`:103` 的 `if (detail)`、导入 `:6`），注释 `:11-14` 加一句 P6a。T7 到 T11 之间“视图不存在”只有单测的证据。
- T8 在途停用（R29）：页头五个组件（`HeaderFilters`、`MobileHeaderFilters`、`LayoutSelection`、`MobileLayoutSelection`、筛选开关）与筛选行只加可选的 `disabled`（默认不停用），`HeaderColumn` 与 `ProjectEmptyState` 只读 `DraftBusyContext`（默认 `false`）；读它们的既有测试（`C/components/issues/issue-layouts/filters/preferences-controls.test.tsx`、`properties/row-planning-properties.test.tsx`、`spreadsheet/columns/header-column.test.tsx`、`route-scope.test.tsx` 等）不改；有格因此要改，先停下报告。
- T8 `C/components/issues/issue-layouts/roots/save-view-collapsed.test.tsx`（108）：`git mv` 为 `save-view-offered.test.tsx`，`:99-107` 的 `it.each` 翻为“对管理员给 Save view”，加“访客不给”“开关关着不给”的格；`:106`、`:107` 按字面找的 “Clear all”“Save view” 改为键 `common.clear_all`、`common.filter_options.save_view`（R21，`fake-i18n` 的文字即键，`C/lib/fake-i18n.ts:10-12`）；夹具不改。
- T8 `C/components/issues/issue-layouts/roots/cycle-layout-root.test.tsx:91`（132）、`module-layout-root.test.tsx:92`（133）（预检 re-96 I6 ②）：`buttonOf("Clear all")` 改为 `buttonOf("common.clear_all")`，断言的意思不变；两份别的格不改。
- T9 `C/components/power-k/menus/search.test.tsx`（169）：`:33` 的 `viewOf`、`:64` 的视图一行改用 `View`（编译跟随，断言不改）。
- T10 `C/store/root.store.test.ts:106` 回到 20。收藏的测试若读 `viewStore` 或 `removeFavoriteEntityFromStore`（基线 `git grep` 没有读者），先停下报告。
- T11 `E/fixtures/mounts.ts`：加 `PROJECT_VIEWS`，头注释 `:1-15` 改写。（S2 一条移到 T7。）
- T4 `E/stories/project/p8-project-preferences.spec.ts`：只在 chromium 跑出在途的读的问题时，加一个等视图列表的读答完的 `waitForResponse`（在 `:357` 的 Enter 之前登记、`:358` 的 `toHaveURL` 之后等；这一步是 `:332-358`），断言不改。
- 不改、不受影响（回归）：M4、M6 的列表与筛选测试（`page-preferences-filter` 的、`use-page-filter-expression` 的、四个布局根的、`planning-page-*` 的；`cycle-layout-root.test.tsx:91`、`module-layout-root.test.tsx:92` 两行照上面 T8 的一条改键，别的格不改）；`filters-hoc/base.test.tsx`；P4、P5 的测试；E2E 的 m4、m6 配置里的故事。

**红的窗口**：没有红的提交。产品改动让既有的格或故事变的，都在同一提交里照上面的清单改（S2 在 T7、三处按字面找的单测在 T8）。名单之外的任何红先停下报告。

**T4 与 T5 之间的窗口**（预检 re-96 I5 ①，取 (a)）：T4 的宿主只画删除的弹窗；行菜单的“编辑”、列表页的 “Add view” 与空态的“新建”照矩阵给（T4 的格只测给不给），它们的处理函数在 T5 随新的弹窗接到宿主上。T4 里 “Add view” 与“新建”仍是今天的 `toggleCreateViewModal(true)`（`C/components/modals/project-level.tsx:59-64` 的旧弹窗，基线上就写旧地址），“编辑”今天的行内弹窗收 `IProjectView`、交不进 `View`：T4 删去行内的那对弹窗，“编辑”一项照矩阵出现，`useViewMenuItems` 的 `handleEdit` 在 T4 是空函数、T4 里点了不开弹窗（分支内一个 Task 的窗口，不进 main；T4 的提交说明与台账写明，T5 换成宿主的 `openDialog` 并以格钉住打开；T5 之后 `git grep` 核没有空的处理函数，收据 `t5-edit-wired`）。

**检查点**（七项用 `sh L/gates.sh <前缀>`；带页面配置用 `sh L/final.sh <前缀>`：七项之后跑 m7-pages、m6-pages、m6-acceptance、m4-list、m4-filters、m4-detail 六个配置，三种引擎，spec R22）：
- **0（T5 之后，引擎改动之前）**：`final.sh cp0`；主控推送、核分支 CI；台账记基线：本机第 1 路 bootstrap（P6a 的基线，+30 秒停查）、web 作业的 Lint、Unit tests、Build 三步、e2e 两片、Go 两路，以 `gh` 读 main CI `45375109166` 的同名作业与步作对照；m4-list、m4-filters、m4-detail 这一次是 T6 之前的基线。
- **1（T7 之后：引擎与视图页完成）**：`final.sh cp1`；主控推送、核分支 CI。
- **2（T10 之后：命令面板与删除完成，产品代码完成）**：`final.sh cp2`；推送之前照 P4 spec §2.11（`specs/P4-web-notifications.md:270`）估出 e2e 两片（以最近一次分支 CI 的两片为底，加新文件的 chromium 时长 × CI 与本机之比），超过 30 分钟先报告、不推送；主控推送、核分支 CI，记两片与 web 作业三步。
- **终检（T14）**：`final.sh fin`（同六个配置）。
- **另跑的单配置**（动了别的 M 的共用件，先 `make build`）：T6 之后 chromium 的 m4-list、m4-filters、m4-detail（`t6-m4-list`、`t6-m4-filters`、`t6-m4-detail`）；T8 之后 chromium 的 m4-filters、m4-list 与 m6-pages（`t8-m4-filters`、`t8-m4-list`、`t8-m6-pages`：筛选行与页头的布局、显示是这些页面共用的）；T4 之后 chromium 的 P8 故事（`t4-p8-follow`）；T7 之后 S2 与默认配置的 chromium（`t7-s2`，S2 随视图页改，预检 re-96 I6 ①）；T9 之后 chromium 单跑打开命令面板的三份故事（`t9-palette`：`E/stories/issue/i18-search.spec.ts`、`E/stories/planning/g19-work-item-side-page.spec.ts`、`E/stories/collaboration/h1-subscription-page.spec.ts`；它们多数不在六个页面配置里，预检 re-96 m4 ③）。
- 第 1 路：本机超过 CP0 基线 +30 秒先停下查；CI 的 bootstrap 过期限的八成只报告（P5b 的最终分支 CI 是 82.6%，`.376`）。本机 Go 两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 的服务端两路为准并记录（负责人的规则）。

**证据**
- 每次检查经 `run-check.py`（`python3 L/run-check.py <label> <command…>`），变异经 `mutate.py`（每个旧串恰出现一次，工具恢复并核 sha256、打印 `restored=yes`）；标签 `t<N>-…`、`cp<k>-…`，不重用。变异 `PKG/` 的源码之后核对 dist 已随之还原（重建或核 sha256，P5b 的教训）。
- **RED 先于实现**：先写失败的测试，看它因所测性质失败（不是编译失败），留 RED 收据，再实现到绿。
- 下表每个性质至少一个能编译、能让测试失败的变异。两层都在查同一件事时两层一起改，并留只改一层的对照。只有竞态杀得死的变异不算。
- **页面故事先写一个期望错误的版本**（或以 e2e 变异），看到断言在被改处失败，再改正；收据 `t<N>-red-page-…`。
- **死成员**：T1 的基线（`L/dead-members/dead-before.tsv`），T4、T6、T8、T10 之后整份比对（`t<N>-dead-after`），T13 的终表；新出现的每一行写明去处。

**测试时间**
- Go 不变，不加顶层名。
- web 作业期限 20 分钟、960 秒报告线；每个检查点记 Lint、Unit tests、Build 三步；越线时三步对 main CI `45375109166` 的同名步各算比值，三个比值相近是慢 runner，照报告、不停；只有 Unit tests 的比值明显高于另两步才是本段的增长，先报告主控再继续。加大量 vitest 的 Task 记新文件的时长。
- e2e 每片 35 分钟、报告线 30 分钟。T11、T12 记每个新文件每个 `test()` 在三种引擎的时长（JSON 报告）；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

**页面故事的教训**（P5b 阶段记录 §7、§9，C-27）
- 每个“在途”都靠扣住回答（`holdAnswer`），不靠时间。
- 一律放行或拒绝的路由只在 Chromium 装，而且只在目标页的代码（含编辑器、命令面板的代码块）载入之后装：WebKit 在页面载入或动态 import 时有 `page.route` 生效就不用预载、再取一次，约 3 秒后报 “preloaded … not used”。
- 换页、离开项目、改服务端的状态（另一会话经接口关开关、删视图）之前，先等在途的读答完：动作之前登记 `waitForResponse`。
- 一次 `fetchIssue` 是两个 `GET`（M4 的浮层），数读的时候照此。
- swr 2.4.2 的重连只通知一个键上的第一个 hook：要靠重连的格用真实 SWR、写明是哪个 hook。
- jsdom 只钉落点的目标，真实焦点以三引擎的故事为准；键盘、指针、窄屏（800 宽）分开看。
- 按名字定位的按钮限在它的区域里；不在重载之后查控制台（M6-M8-09）；不给任何东西起名 “Notifications”。
- 每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去）与 `expectQuietConsole`（不放宽）结束。
- `AlertModalCore` 的默认按钮文字是英文（`web/packages/ui/src/modals/alert-modal.tsx:62-66`）：显式传 `t()`。
- Docker 重启之后 testcontainers 的端口超时：等 ryuk 退出，以新标签重跑（M6-M8-05）。

## 性质与变异

| 性质 | 变异（标签） | Task |
|---|---|---|
| 写一个接一个 | 视图的写不经队列（`t2-mut-no-queue`，两次改的第二次在第一次答到之前发出的一格失败） | T2 |
| 体在轮到时算 | 体在调用时算（`t2-mut-body-at-call`，排在前一次之后的改读到旧值的一格失败） | T2 |
| 回答写进列表与详情 | 建不 `prepended`（`t2-mut-create-no-prepend`）；删留着行（`t2-mut-delete-keeps-row`）；改不 `replaced` 详情（`t2-mut-update-detail-stale`） | T2 |
| 没有乐观改 | 发出之前先在本地改（`t2-mut-optimistic`，“答到之前列表不变”的一格失败） | T2 |
| 被拒不改任何值 | 被拒时仍以请求体写进详情（`t2-mut-refusal-writes`，“被拒之后列表与详情不变”的一格失败） | T2 |
| 换会话之后旧回答不写 | 去掉代的检查（`t2-mut-no-generation`） | T2 |
| 项目不在了不显示它的视图 | 不看项目（`t2-mut-project-gone-shows`） | T2 |
| 入口要三种角色之一 | 不看角色（`t3-mut-entry-no-role`，“没有角色 F”的格失败） | T3 |
| 入口看开关 | 不看开关（`t3-mut-entry-no-toggle`） | T3 |
| 不排除已归档 | 排除已归档（`t3-mut-entry-excludes-archived`，表里已归档一列失败） | T3 |
| 新建只给管理员与成员 | 访客也给（`t3-mut-create-guest`，`create_view` 访客 F 的格失败） | T3 |
| 命令读入口规则 | 两条命令回到只看开关（`t3-mut-command-toggle-only`） | T3 |
| 页面状态按项目 | 状态提到 key 之外（`t4-mut-state-global`，换项目之后搜索还在的一格失败） | T4 |
| 搜索不分大小写、排序生效 | 区分大小写（`t4-mut-search-case`）；排序不生效（`t4-mut-order-ignored`） | T4 |
| 矩阵的列表一半 | “Add view” 不看角色（`t4-mut-add-view-any-role`）；空态的“新建”给访客（`t4-mut-empty-new-guest`）；成员能删别人的（`t4-mut-delete-member-others`）；管理员能改别人的（`t4-mut-edit-admin-others`） | T4 |
| 开关关着不读 | 门不看开关（`t4-mut-disabled-reads`，“关着没有 `GET`”的一格失败） | T4 |
| 一行一份快捷操作 | 加回 `quickActionElement`（`t4-mut-two-menus`，“一行一个右键菜单”的一格失败） | T4 |
| 菜单按钮以视图名为名 | 回到 “Options”（`t4-mut-menu-options-name`） | T4 |
| 复制的失败有提示 | 只处理成功（`t4-mut-copy-success-only`） | T4 |
| 删除的跟进只在宿主还挂着时 | 不看 `pageMounted`（`t4-mut-follow-unmounted`，离开页面之后答到仍提示的一格失败） | T4 |
| 所有者的选项含访客 | `getProjectMemberIds(projectId, false)`（`t4-mut-members-no-guests`） | T4 |
| 在途不关（两层） | 在途也给 `handleClose` 且 Cancel 不停用（`t5-mut-close-in-flight`）；只改一层的对照 `t5-ctl-close-in-flight-one-layer` | T5 |
| 不在途时 Escape、点背景、Cancel 关 | 一律不给 `handleClose`（`t5-mut-no-handleclose`，不在途 Escape 的一格失败） | T5 |
| 没有文档级 Escape | 加回 `useKeypress("Escape")`（`t5-mut-document-escape`，图标选择器开着时按 Escape 外层也关的一格失败） | T5 |
| 组字时的 Escape 不关（IME） | 去掉 `ModalCore` 的 `isComposing` 一项（`t5-mut-ime-escape-closes`，`PKG/ui` 的源码，变异之后核 dist 已还原；先例 P5b 的 `t2-mut-ime-escape-closes-r2`） | T5 |
| 弹窗以 `ModalTitle` 为名 | 回到 `h3`（`t5-mut-plain-title`） | T5 |
| 名字按字符、去空白 | 按 `name.length`（`t5-mut-length-units`）；不去空白（`t5-mut-no-trim`） | T5 |
| 被拒焦点给名字、`aria-invalid` | 不经 `useRefusedFocus`（`t5-mut-no-refused-focus`）；恒为假（`t5-mut-invalid-off`） | T5 |
| nerve 点名 `name` 放在字段下 | 一律提示（`t5-mut-name-refusal-toast`） | T5 |
| 编辑只发改了的字段 | 发整份（`t5-mut-send-all-fields`） | T5 |
| 只在 nerve 确认之后关 | 发出即关（`t5-mut-close-before-answer`） | T5 |
| 迟到的成功只在弹窗还挂着时 | 去掉 `mounted`（`t5-mut-follow-unmounted`） | T5 |
| 新建成功转到新视图 | 不跳转（`t5-mut-no-navigate`） | T5 |
| 行的“编辑”、“Add view”、空态的“新建”打开宿主的弹窗（T4 只测给不给） | “编辑”打开新建（`t5-mut-edit-opens-create`，“编辑的名字初值是视图名”的一格失败）；“Add view” 仍走旧弹窗（`t5-mut-add-view-old-modal`，“点 Add view 打开宿主的新建”的一格失败） | T5 |
| 自然 Tab 次序 | 给提交按钮 `tabIndex={1}`（`t5-mut-positive-tabindex`） | T5 |
| 弹窗只有名字、说明、图标（R28） | 加回布局的控件（`t5-mut-modal-layout`，“弹窗里没有布局与显示”的一格失败） | T5 |
| “Add view” 的体带合同默认的显示与属性 | 常量里的 `group_by` 改为 `"state_id"`（`t5-mut-default-grouped`）；不带显示与属性（`t5-mut-add-no-defaults`），请求体的格失败 | T5 |
| 编辑不发显示 | 编辑也带显示（`t5-mut-edit-sends-display`，“只发改了的名字”的一格失败） | T5 |
| 名字的初值为空（R30） | 另存在视图页预填“{视图名} 2”（`t5-mut-saveas-name-default`） | T5 |
| 视图页的列表与项目页的分开 | `use-issues.ts` 的 `PROJECT_VIEW` 回项目的 store（`t6-mut-view-uses-project`） | T6 |
| 视图的路由是 `PROJECT_VIEW` | `viewId` 回到 `DEFAULT`（`t6-mut-viewid-default`） | T6 |
| 草稿只替换给出的字段 | `update` 整份替换（`t6-mut-update-replaces-all`） | T6 |
| 有草稿不再读 | 每次 `get` 都读视图（`t6-mut-get-reads-every-time`，“`adopt` 之后的 `fetchFilters`——“更新视图”成功之后的那一次——不发 `GET`”的一格失败；预检 re-96 I1：重连不再读，格换了读者） | T6 |
| `discard` 丢 `filters[viewId]` | 只丢草稿（`t6-mut-discard-keeps-filters`，“再进来读答到之前不显示旧草稿”的一格失败） | T6 |
| 迟到的播种不建回草稿（I2） | 去掉草稿服务的代（`t6-mut-late-seed-revives`，“扣住播种的读、`discard`、放开、再 `get` 恰再读一次、`filters[viewId]` 不在”的一格失败） | T6 |
| `adopt` 只在草稿还在时（I2） | 没有草稿也建（`t6-mut-adopt-revives`，“`discard` 之后 `adopt`，再 `get` 恰再读一次”的一格失败） | T6 |
| 列表的查询来自草稿 | 查询来自视图存着的（`t6-mut-query-from-saved`，改了草稿之后重取的查询的一格失败） | T6 |
| 空态读视图页的筛选 | 读项目页的（`t6-mut-empty-project-filter`） | T6 |
| 列表挂载的键带 `viewId`（I3 (a)） | `page` 一段对 `PROJECT_VIEW` 给空（`t6-mut-mount-key-no-view`，“扣住 A 的列表读、换到 B，B 发自己的读”的一格失败） | T6 |
| “显示”有选项（I3） | `ISSUE_STORE_TO_FILTERS_MAP` 去掉 `PROJECT_VIEW`（`t6-mut-filters-map-missing`） | T6 |
| 看板与列表的本地筛选（I3） | `use-page-issue-filters.ts` 去掉 `PROJECT_VIEW` 的 `case`（`t6-mut-page-filters-missing`） | T6 |
| 空态的分派（I3） | `empty-states/index.tsx` 去掉 `PROJECT_VIEW` 的 `case`（`t6-mut-empty-state-null`，视图页没有空态的一格失败） | T6 |
| 分组头的“+”（I3） | `DIALOG_STORES` 去掉 `PROJECT_VIEW`（`t6-mut-dialog-store-missing`，点“+”抛错的一格失败） | T6 |
| 行的编辑与复制写进视图页的列表（I3 (c)） | `ProjectIssueQuickActions` 恒传 `PROJECT`（`t6-mut-quick-actions-project`） | T6 |
| 日历某一天的“添加已有”（I3 (b)） | 视图页的命令不给 `addIssuesToView`（`t6-mut-no-add-existing`，宿主找不到命令的一格失败） | T6 |
| 拖放（I3） | `use-group-dragndrop.ts` 的联合与分支去掉 `PROJECT_VIEW`（`t6-mut-dragdrop-missing`） | T6 |
| `updateFilters` 不看给的 id（m11 ①） | 用调用方给的 id（`t6-mut-update-filters-given-id`，表格与日历传项目 id 的两格把草稿写到项目 id 名下而失败） | T6 |
| 以 `viewId` 为 key | 去掉 key（`t7-mut-no-key`，换视图之后 A 的草稿还在的一格失败） | T7 |
| 不存在与失败分开 | 一律算失败（`t7-mut-not-found-as-failure`） | T7 |
| 进页面恰一个视图的读 | 另加一个 `["VIEW", viewId]` 的读（`t7-mut-double-read`） | T7 |
| 每次挂载读自己的（I1） | 回到普通的挂载读（`t7-mut-plain-mount-read`，“真的 hook 与真的 SWR 同一个缓存 provider，A 答到之后立刻卸下再挂上，恰再一个 `GET`、页面离开加载”的一格失败） | T7 |
| 视图页的筛选不推进基线（R10） | 改用 `usePageFilterExpression`（`t7-mut-view-advances-baseline`，“改了筛选之后 `hasChanges` 为真——基线没被推进——、没有 `PATCH`”的一格失败；“`canUpdateView` 为真”在 T8，预检 re-96 I5 ②） | T7 |
| 卸下时丢草稿 | 不调 `discard`（`t7-mut-no-discard`） | T7 |
| “Add work item” 给管理员与成员 | 访客也给（`t7-mut-add-guest`） | T7 |
| 删除之后转到列表（m11 ③） | 删除之后不跳转（`t7-mut-delete-stays`） | T7 |
| 切换器打开时才读列表（m11 ③） | 挂上就读（`t7-mut-switcher-reads-closed`，“切换器关着没有 `GET …/views`”的一格失败） | T7 |
| 页头在四种状态下画面包屑（m8） | 没有视图时什么都不画（`t7-mut-header-blank`） | T7 |
| 窄屏页头（m8） | 布局不给 `mobileHeader`（`t7-mut-no-mobile-header`） | T7 |
| 另存给谁 | 访客也给（`t8-mut-saveas-guest`）；迭代页不传 `enableSaveView`（`t8-mut-saveas-collapsed-cycle`） | T8 |
| 视图页改了筛选之后给“更新视图”（R10 的 T8 一半，I5 ②） | 视图页的根不传 `enableUpdateView`（`t8-mut-root-no-update`） | T8 |
| 迭代、模块另存带词 | 不带（`t8-mut-no-view-terms`） | T8 |
| 另存的显示是已确认的线上值 | 取控件词（`t8-mut-saveas-control-words`） | T8 |
| `confirmed` 返回拷贝 | 返回所持的对象（`t8-mut-confirmed-live`，改了返回值之后所持的值变了的一格失败） | T8 |
| “更新视图”只给所有者（角色管理员或成员） | 不看所有者（`t8-mut-update-non-owner`）；被降为访客的所有者也给（`t8-mut-update-guest-owner`） | T8 |
| `hasAdditionalChanges` 比线上值 | 拿控件词比（`t8-mut-additional-control-words`，没有改动也出现按钮的一格失败） | T8 |
| `hasAdditionalChanges` 跟着显示重算（m9） | memo 的依赖只挂 `confirmed`（`t8-mut-additional-confirmed-only`，“改了显示之后按钮出现”的一格失败） | T8 |
| 失败之后按钮留着、草稿与基线不动 | 失败也 resolve（`t8-mut-update-resolves-on-failure`） | T8 |
| 成功之后基线是回答 | 回答与当前表达式不等价时不 `resetExpression`（`t8-mut-baseline-current`） | T8 |
| 草稿换成回答 | 不 `adopt`（`t8-mut-no-adopt`，成功之后显示仍是草稿的一格失败） | T8 |
| 体的显示在轮到时读 | 在点击时读（`t8-mut-body-at-click`；格在 HOC 加假队列的那一层：扣住前一次写、点“更新视图”、改草稿的显示、放开，体带改后的显示。R29 之后界面上分不出，页面故事里看不到，预检 re-96 O1） | T8 |
| 跟进只在 HOC 还挂着时（I2） | 去掉 `mounted`（`t8-mut-update-follow-unmounted`，“离开之后答到：不提示、没有草稿”的一格失败） | T8 |
| 换会话之后答到不提示（m11 ②） | 成功的提示不经 `followInSession`（`t8-mut-update-other-session-toast`） | T8 |
| 成功之后焦点落在添加筛选（m6） | 不移焦点（`t8-mut-update-focus-lost`，“焦点不在 `body`、在添加筛选”的一格失败） | T8 |
| 筛选行接住拒绝 | 去掉 `catch`（`t8-mut-no-catch`，未处理的拒绝的一格失败） | T8 |
| 在途时筛选行改草稿的控件停用（R29） | 筛选行不看 `disabled`（`t8-mut-row-busy-enabled`，在途时条件仍可改的一格被杀）；`ProjectViewLayoutRoot` 不传（`t8-mut-root-no-busy`） | T8 |
| 在途时表格列头的排序停用（R29，O-6） | `HeaderColumn` 不读 `DraftBusyContext`（`t8-mut-sort-busy-enabled`，在途时列头仍能排序的一格被杀） | T8 |
| 在途时日历的选项下拉停用（R29，I4） | 选项下拉不读 `DraftBusyContext`（`t8-mut-calendar-options-busy-enabled`，在途时仍能换月 / 周的一格被杀；页面故事不另加，单测承担） | T8 |
| 在途时空态的清除筛选停用（R29，O-6） | `ProjectEmptyState` 的清除不读 `DraftBusyContext`（`t8-mut-empty-clear-busy-enabled`，在途时仍能清除的一格被杀） | T8 |
| 在途时页头的布局与显示停用（R29） | 视图页头不传 `isUpdating(viewId)`（`t8-mut-header-busy-enabled`，在途时布局仍可换的一格被杀） | T8 |
| `isUpdating` 在落定时清 | 被拒之后不清（`t8-mut-busy-stuck`，“被拒之后控件恢复”的一格失败） | T8 |
| 项目页、迭代页、模块页不停用 | `HeaderFilters` 的停用默认为真（`t8-mut-header-default-disabled`，项目页的布局切换的既有格失败） | T8 |
| 筛选行与 HOC 的文字经 `t()` | “Update view” 回到字面量（`t8-mut-literal-update`，`fake-i18n` 的键的一格失败） | T8 |
| 菜单与列表页同一个键、门是入口规则 | 换一个键（`t9-mut-menu-other-key`，“面板与列表页同时挂着时共用一次在途的读”的一格失败，先例 `C/components/power-k/ui/modal/search-menu.test.tsx:335-345`）；不看门（`t9-mut-menu-no-gate`） | T9 |
| 搜索在 store 里没有才读（m4 ②） | 有了也读（`t9-mut-search-reads-when-held`，“store 里已有这个项目的视图时开面板没有 `GET`”的一格失败） | T9 |
| 搜索只在项目范围 | 工作区范围也给（`t9-mut-search-workspace-scope`） | T9 |
| 搜索不分大小写 | 区分（`t9-mut-search-case`） | T9 |
| 读被拒不挡别的组 | 被拒时整个结果给“没有结果”（`t9-mut-search-blocks`） | T9 |
| 收藏的名字取新 store 所持的 | 一律取收藏自己的（`t10-mut-favorite-own-name`） | T10 |
| `old-view-urls` 拦住旧地址 | 在一个 web 文件里放回旧地址（`t10-mut-keyword-old-url`，`node tools/keywords.mjs` 失败） | T10 |

## Tasks

### T1 基线与文档（主控）

- **提交**：spec、plan、预检（含处理）。实施者简报（`L/implementer-brief.md`）与台账一样不进 git（`.superpowers/` 在本地的 exclude 里），随阶段归档。
- **P5b 的回填**（spec R26；数字取自台账 `L/progress.md` 第二行，主控核过归档 `M7-P5b-integration-20261124` 的台账）：`reviews/P5b-web-intake-files-embed-review.md` 的状态行（`:3`）与 §9 末句（“终检、最终分支 CI、合并与 main CI 由 P6a 的 T1 补记”）补“终检与集成”：终检 `fin` 全绿（本机第 1 路 657.6 秒）；最终分支 CI `45369650152` 全绿（web 作业 1075 秒、第 1 路 990.9 秒 = 82.6%，都在 `.376`，只报告）；合并 `47bd860b`；main CI `45375109166` 全绿（第 1 路 729.2 秒 = 60.8%）；归档 `M7-P5b-integration-20261124`（10965 个文件，unknown 0，清理之后核过）。
- **设计与清单**：设计 §14 的 P5b 一行（`D:352`）改为已完成、加 P6a 一行；§12 的 P6a 一行（`D:308`）链 spec。`docs/v0/frontend-changes.md` §3.1 的 M7 一行（`:206`）补 P5b 的完成。主控改，实现代理不碰设计。
- **关键词**：`K:2` 的 phase 改为 `M7/P6`；跑 `node tools/keywords.mjs`。
- **台账工具**：`L/final.sh` 的页面配置改为 `m7-pages m6-pages m6-acceptance m4-list m4-filters m4-detail`（spec R22；今天是 P5b 的六个，含 m5-content、不含 m4-filters），文件头的注释照改。
- **依赖**：根目录 `pnpm install --frozen-lockfile`（收据 `t1-install`）。
- **死成员的基线**：先 `pnpm --filter web exec react-router typegen`，再 `node L/dead-members/deadsym.mjs --tsv > L/dead-members/dead-before.tsv`、`node L/dead-members/domains.mjs L/dead-members/dead-before.tsv --rows M7`（收据 `t1-dead-before`）；与 P5b 的终表（`/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M7-P5b-integration-20261124/payload/worktree/.superpowers/sdd/P5b-web-intake-files-embed/dead-members/rows-M7-t11-2.tsv`）逐行比对，视图的行（`views.helper.ts`、`constants/src/views.ts`、`workspace-views.ts`）列出。
- **检查**：文档链接（`doc-links.py`）与关键词，然后提交 `docs(M7/P6a): …`。

### T2 数据层

**Files**：
- 新：`C/services/views/views.service.ts`、`C/store/views/view.store.ts`、`C/store/views/view-changes.ts`（不过 220 行）与它们的测试（`view.store.test.ts`、`view-changes.test.ts`）。
- 改：`C/store/root.store.ts`（120；`views` 字段，在 `issue` 之前构造）、`C/store/root.store.test.ts:106`（允许清单）。

**Steps**：
- [ ] Step 1：先写测试（`@/lib/auth/fake-nerve`、`@/store/fake-queue`），看到它们因所测性质失败（收据 `t2-red`）：读一个项目的视图与一个视图（回答原样进 `lists`、`details`）；建、改、删各经队列、回答写进列表与详情（建插在最前、改替换两处、删移出并把详情设为 `null`）；两次改的第二次在第一次答到之前不发出、它的体在轮到时算；答到之前列表与详情不变；被拒不改任何值、拒绝交给调用方；换会话之后旧的回答不写；项目不在 store 里时 `getProjectViews` 为空；没有改动的 `PATCH` 照样写回答。
- [ ] Step 2：实现到通过（spec §2.1；先例 `C/services/planning/modules.service.ts`、`C/store/planning/module.store.ts:48-58`、`module-changes.ts`）；`RootStore` 加 `views`（`projectView` 照旧），`root.store.test.ts:106` 改 21。
- [ ] Step 3：变异照上表 T2 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`；新测试的时长。提交 `feat(M7/P6a): 视图的 service 与 store——项目的视图列表与单个视图照 v0 7.7（ReconciledByKey、读不排队），建改删一个视图一个队列、请求体轮到时算、只在 nerve 确认时写`。

### T3 入口规则

**Files**：新 `C/lib/views-entry.ts` 与 `views-entry.test.ts`；改 `C/components/workspace/sidebar/project-navigation.tsx`（191，`:103-111`）、`C/components/navigation/use-navigation-items.ts`（113，`:73-82`）、`C/components/power-k/config/navigation/project-commands.ts`（258，`:180-181`、`:192-193`）、`C/components/power-k/config/creation/command.ts`（165，`:110-111`）；测试照允许清单追加。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t3-red`）：`viewEntryOn` 的表（开关开着与关着 × 管理员、成员、访客、没有角色 × 已归档与否；先例 `C/lib/intake-entry.test.ts`）；`viewsCreatableBy`（访客 F）；侧栏、页签栏、两条命令、`create_view` 各一格（没有角色 F、访客：导航 T、`create_view` F）。
- [ ] Step 2：实现到通过（spec §2.2）。角色取 `getProjectRoleByWorkspaceSlugAndProjectId`（路由参数）。新文件若读项目，收入 `project-address-route`。
- [ ] Step 3：变异照上表 T3 各行。门禁同 T2。提交 `feat(M7/P6a): 视图的入口规则 viewEntryOn——开关开着、三种角色，不排除已归档；侧栏、页签栏、两条导航命令共用，新建与 create_view 另要管理员或成员`。

### T4 列表页、行与宿主、`projectMemberIds`

**Files**：
- 改：`V/(list)/{layout,page,header,mobile-header}.tsx`（27、111、62、63）；`VC/{views-list,view-list-item,view-list-item-action,view-list-header,quick-actions,delete-view-modal}.tsx`（91、62、112、127、136、78）；`VC/filters/{filter-selection,order-by}.tsx`（113、84）；`VC/applied-filters/{root,index}.tsx`（98、7）；`C/components/common/filters/created-at.tsx`（82；“Created date”“Custom”、`:76` 的 “No matches found” 改键）、`created-by.tsx`（116；“Created by” `:62`、“You” `:88`、“View less / View all” `:98`、“No matches found” `:103` 改键，`:39-50` 的选项依赖补上成员、已选与当前用户、删去 `:49` 关规则的注释，预检 re-96 m10）；`C/components/common/quick-actions-helper.tsx`（145，`:47-57`、`:130-145`）；`C/components/workspace/views/quick-action.tsx:52`（编译跟随）；`C/store/member/project/project-member.store.ts`（234，`:40`、`:85`、`:106-109`）与测试、`C/components/inbox/filters/menu.test.tsx:64`（允许清单）；`K`（`copy-hook` 的视图例外删、`project-address-route` 的名单与样例）；`.oxlintrc.json`；en、zh-CN 的 `project.json`（`project_views.add_view`）。
- 新：`C/hooks/store/use-views.ts`、`VC/page-state.tsx`、`VC/actions-host.tsx`（T4 只有删除，见 Global Constraints 的“T4 与 T5 之间的窗口”）、列表的排序与筛选的纯函数（`VC/view-list-order.ts`）与测试（按关注点拆，每份不过 400 行）。
- 删：`VC/applied-filters/access.tsx`（54）；`V/(list)/page.test.tsx`、`VC/follow-ups.test.tsx` 的删除一组（允许清单）。

**Steps**：
- [ ] Step 0：读 T1 的死成员基线（收据 `t1-dead-before`）。`git grep -n projectMemberIds -- web tools` 核读者只有两处视图与 store 自己、测试与替身，与 `C/components/issues/issue-detail-widgets/sub-issues/title-actions.tsx:45`、`:94` 的同名局部变量、`K` 的规则文字（`K:2336` 的说明、`:2342` 的正则、`:2348` 的 hit 样例）（收据 `t4-member-readers-before`）。
- [ ] Step 1：先写测试，看到失败（收据 `t4-red`）：列表页的状态（加载、失败与重试、空、有行、没有匹配、开关关着不读）；按项目的页面状态（换项目即丢）；搜索、所有者与创建日期筛选、排序；矩阵的列表一半，只测给不给（“Add view”、空态的“新建”、行菜单的编辑与删除按所有者与角色、访客只有复制与新标签页；打开新建与编辑的格在 T5）；一行一个右键菜单、菜单按钮名为 “{name} options”、不嵌按钮；复制成功与失败的提示（`useCopyLink`）；宿主：删除经 `openDialog`，删除确认之后提示、转到列表、行消失，离开页面之后答到不提示；所有者的选项含访客（`getProjectMemberIds(projectId, true)`），成员晚到时面板跟上（`created-by.tsx` 的依赖）；筛选面板与页面标题的英文经 `t()`。
- [ ] Step 2：实现到通过（spec §2.3）；删 `quickActionElement`、没人打开的那对弹窗、公开私有与收藏星、`applied-filters/access.tsx`；`useViewMenuItems` 的参数照 spec；`copy-hook` 的视图例外删；英文改键。两个 `projectMemberIds` 读者改显式取，重新 `git grep`（收据 `t4-member-readers-after`：只剩 store 的定义、`title-actions.tsx` 的同名局部变量与 `K` 的规则文字），删接口、`computed`、getter、专属断言与替身字段（允许清单）；`K:2336` 的说明照同一句里 `currentProjectDetails` 的写法改为“P6a 删去、模式里的这一式只防它回来”，正则与样例留着。新文件收入 swr 名单与 `project-address-route`；跑 `node tools/keywords.mjs`。
- [ ] Step 3：变异照上表 T4 各行。门禁同 T2；死成员整份比对（`t4-dead-after`）；oxlint 计数（`t4-oxlint-count`）；`make build` 之后 chromium 跑 P8 的故事（`t4-p8-follow`；红了照允许清单加等待，再跑 `t4-p8-follow-2`）。提交分两个：`feat(M7/P6a): 视图列表页——按项目的搜索、筛选与排序，矩阵给控件，一行一个菜单以视图名为名、复制经 useCopyLink，删除的弹窗在页面的宿主（新建与编辑随 T5 的弹窗接上）` 与 `refactor(M7/P6a): 删除成员 store 的 projectMemberIds——最后两个读者（视图列表页）改为显式取项目的成员`。

### T5 视图的弹窗、`LayoutDropDown` 与 `getTabIndex` → **检查点 0**

**Files**：
- 改：`VC/modal.tsx`（94）、`VC/form.tsx`（265；删 `:197-246` 的布局与显示、`:41-47` 里的 `access` 与显示的初值）；`VC/actions-host.tsx`（T4 新建；加新建与编辑两个弹窗）、`VC/quick-actions.tsx`（“编辑”接 `openDialog`）、`V/(list)/header.tsx` 与 `VC/views-list.tsx`（“Add view” 与空态的“新建”接宿主的新建，今天 `header.tsx:54`、`views-list.tsx:82` 的 `toggleCreateViewModal(true)`；窄屏页头没有 “Add view”）；`C/components/modals/project-level.tsx`（67，`create_view` 的新建，只换 props 的形状）；en、zh-CN 的 `project.json`（`project_views.action.*`）。
- 删：`C/components/dropdowns/layout.tsx`（49）与 `layout.test.tsx`（110，允许清单）；`PKG/utils/src/tab-indices.ts`（18）、`PKG/constants/src/tab-indices.ts`（15）、桶导出 `PKG/utils/src/index.ts:23`、`PKG/constants/src/index.ts:29`；`VC/follow-ups.test.tsx`（允许清单）。
- 新：“Add view” 的合同默认显示与属性的常量（弹窗旁，线上词）；弹窗的测试（关闭、名字、字段与体，按关注点拆，每份不过 400 行；要真实 `ModalCore` 的格经弹窗渲染、从 `document` 查，先例 `C/components/cycles/` 的弹窗测试）。
- 不碰：`C/store/issue/helpers/page-preferences-filter.store.ts`（显示的换算不导出，R28）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t5-red`）：
  - 在途不关（Escape、点背景、Cancel 停用，两层）；不在途 Escape、点背景、Cancel 关；没有文档级 Escape（图标选择器开着时按 Escape 只关它）；组字时的 Escape 不关（IME，`HO/M4-closeout.md:71`）；`ModalTitle` 为名；
  - 打开：行的“编辑”打开编辑（名字初值是视图的名字）、“Add view” 与空态的“新建”打开宿主的新建、`create_view` 打开 `project-level.tsx` 的新建；
  - 名字的规则（去空白之后为空、超过 255 个字符、emoji 按字符）、`FieldError`、`aria-invalid`、被拒取得焦点、nerve 点名 `name` 放在字段下；
  - 弹窗里只有图标、名字、说明，没有布局与显示（R28）；“Add view” 的请求体是名字、说明、图标加合同默认的 `display_filters`、`display_properties`、没有 `filters`；编辑只发改了的字段（只会是名字、说明、图标）；新建与另存的名字初值为空、编辑是视图的名字（R30）；
  - 只在 nerve 确认之后关；新建成功转到新视图、编辑成功提示；迟到的成功只在挂着时；自然 Tab（图标、名字、说明、Cancel、提交）。
- [ ] Step 2：实现到通过（spec §2.4）。宿主接上新建与编辑；T4 里空的 `handleEdit` 换成 `openDialog`，`git grep` 核视图的菜单与按钮没有空的处理函数（收据 `t5-edit-wired`）。`LayoutDropDown` 与它的测试删去（`git grep -n "dropdowns/layout\|LayoutDropDown" -- web` 为空，收据 `t5-layout-dropdown-gone`）；`getTabIndex` 全套删去（同一提交），`git grep -n "getTabIndex\|ETabIndices\|TAB_INDEX_MAP\|PROJECT_VIEW_TAB_INDICES" -- web` 为空（收据 `t5-tabindex-gone`）；共用控件非正的 `tabIndex?` 的读者核过（收据 `t5-tabindex-readers`，写进台账）。
- [ ] Step 3：变异照上表 T5 各行。门禁同 T2；`pnpm --filter @nerve/constants test`、`pnpm --filter @nerve/utils test`；oxlint 计数（`t5-oxlint-count`）。提交 `feat(M7/P6a): 视图的弹窗只管名字、说明与图标，照 M6——在途不关、不在途 Escape 点背景 Cancel 都关，ModalTitle、名字的 FieldError 与被拒的焦点、自然 Tab；新建与编辑接到列表页的宿主；删除 LayoutDropDown 与 getTabIndex 全套`。
- [ ] Step 4（**检查点 0**）：`final.sh cp0`；主控推送、核分支 CI，台账记基线（Global Constraints）。

### T6 列表引擎的 `PROJECT_VIEW`

**Files**：
- 改：`PKG/types/src/issues/issue.ts`（29）；`C/store/issue/root.store.ts`（329）；`C/hooks/store/use-issues.ts`（105）、`C/hooks/use-issues-actions.tsx`（347）、`C/hooks/use-issue-layout-store.ts`（40）、`C/hooks/store/use-page-issue-filters.ts`（38）、`C/components/issues/use-page-display.ts`（55）、`PKG/constants/src/issue/filter.ts`（309）、`PKG/constants/src/issue/common.ts`（293）、`C/components/issues/issue-layouts/command-host.tsx`（296）、`add-existing-dialog.tsx`（73，`:25` 的请求类型收 `PROJECT_VIEW`）、四个布局根（195、262、225、145）、`C/hooks/use-group-dragndrop.ts`（95）、`quick-action-dropdowns/project-issue.tsx`（171，`:169-171` 读 `IssuesStoreContext`、只在 `PROJECT_VIEW` 时换；六个使用处不改）、`empty-states/{index,project-issues}.tsx`（34、77）、`C/hooks/use-issue-list-mount.ts`（63，`:29` 的 `page` 对 `PROJECT_VIEW` 取路由的 `viewId`）；两份钉 `DEFAULT` 的测试（允许清单）。
- 新：`C/store/issue/project-view/{issue.store,filter.store,view-draft.service}.ts` 与测试；`C/hooks/use-project-view-issue-actions.ts`、`C/hooks/use-project-view-preferences-follow.ts` 与测试。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t6-red`）：
  - 草稿：没有草稿时 `get` 读视图一次并播种；有草稿不再读（`adopt` 之后的 `fetchFilters` 不发 `GET`）；`update` 只替换给出的字段；`adopt` 只在草稿还在时；`discard` 丢草稿、作废在途的播种（扣住播种的读、`discard`、放开：不建草稿、`filters[viewId]` 不在、回答进 `ViewStore.details`；再 `get` 恰再读一次）；`ProjectViewIssuesFilter.discard` 丢 `filters[viewId]`；
  - `ProjectViewIssues` 的查询来自草稿、与项目页的列表分开（各自的绑定）；`isPreferenceQueryCurrent(projectID, viewID)`；
  - `useIssues(PROJECT_VIEW)`、`useIssuesActions(PROJECT_VIEW)` 给视图页的 store 与命令（`fetchIssues`、`fetchNextIssues`、`createIssue`、`createIssueAsGiven`、`updateIssue`、`removeIssue`、`addIssuesToView`、`archiveIssue`、`dropIssue`、`updateFilters` 都在；建、改、删与项目页同、写进视图页的列表，取数是视图页的）；`updateFilters` 不看给的 id（传项目 id 也写这个视图的草稿）、写草稿、跟进重取；`viewId` 的路由是 `PROJECT_VIEW`；
  - 分派点各一格（预检 re-96 I3）：列表挂载的键带 `viewId`（扣住 A 的列表读、换到 B，B 发自己的读）；“显示”有选项；看板与列表的本地筛选；空态（读视图页的筛选）；分组头的“+”；行的编辑与复制写进视图页的列表（`ProjectIssueQuickActions` 在 `PROJECT_VIEW` 的上下文里、个人页的 `PROFILE` 上下文仍是 `PROJECT`）；日历某一天的“添加已有”经视图页的 `addIssuesToView`；拖放。
- [ ] Step 2：实现到通过（spec §2.5）；`use-issues.ts`、`use-issues-actions.tsx` 的 `switch` 穷尽（`default` 里 `satisfies never`；`use-issues-actions.tsx` 显式列出 `WORKSPACE_DRAFT` 并照旧抛错），故意漏一个值编译失败（收据 `t6-exhaustive-probe`：在临时副本上删一个 `case`、`tsc` 报错，不提交）；两份钉 `DEFAULT` 的测试照允许清单改。`root.store.ts` 不超过 340、`use-issues-actions.tsx` 只多一个导入、一行调用、两个 `case`（`t6-wc`）。
- [ ] Step 3：变异照上表 T6 各行。门禁同 T2；死成员整份比对（`t6-dead-after`）；`make build` 之后 chromium 的 m4-list、m4-filters、m4-detail（`t6-m4-list`、`t6-m4-filters`、`t6-m4-detail`），与检查点 0 的同名配置比对。提交 `feat(M7/P6a): 列表引擎加 PROJECT_VIEW——项目列表页的一种，查询来自视图页的草稿；草稿在内存、只替换给出的字段、卸下即丢、迟到的播种不建回；分派点逐个接上（挂载的键、行的快捷操作、添加已有），两个 switch 穷尽`。

### T7 视图页 → **检查点 1**

**Files**：改 `V/(detail)/layout.tsx`（31；`<Fragment key={viewId}>`、加 `mobileHeader`）、`V/(detail)/[viewId]/{page,header}.tsx`（55、116；页头四种状态下的面包屑）、`P/domain-pages.test.tsx`、`E/stories/smoke/s2-issue-list-removal.spec.ts`（171）（两份照允许清单）；新 `V/(detail)/[viewId]/mobile-header.tsx`（`ProjectViewMobileHeader`，先例 `P/cycles/(detail)/layout.tsx:25`、`:31-38`）、`VC/use-view-page-read.ts`（每次挂载读自己的，先例 `C/components/inbox/content/root.tsx:108-125`）、`C/components/issues/issue-layouts/roots/project-view-layout-root.tsx` 与视图页的测试；en、zh-CN（`project_views.not_found.*`）；`.oxlintrc.json`（swr 名单收 `use-view-page-read.ts`）；`K`（`project-address-route` 收新的页头与读）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t7-red`）：
  - 以 `viewId` 为 key（换视图之后 A 的草稿不在 B 上；后退回到 A 是新读到的）；四种状态（加载、`navigation.view_not_found` 的不存在与回到列表、别的失败与重试、开关关着不读），每种状态下页头画面包屑（项目、视图列表）；窄屏页头在视图在时画 `MobileHeaderFilters`；
  - 读（预检 re-96 I1）：进页面恰一个 `GET /api/v0/views/{id}`、列表的查询来自视图存着的；真的 `useViewPageRead` 与真的 SWR（同一个缓存 provider，不替身）：A 答到之后立刻卸下再挂上，恰再一个 `GET`、页面离开加载；重连不读；
  - 改了筛选之后 `hasChanges` 为真（基线没被推进）、没有 `PATCH`（R10；根不传 `enableSaveView`、`enableUpdateView`，“`canUpdateView` 为真”在 T8，预检 re-96 I5 ②）；卸下时丢草稿、再进来读答到之前不显示旧草稿；
  - “Add work item” 与快速添加给管理员与成员；页头的切换器打开时才读列表；删除之后转到列表；“不可用”的条件原样在查询里。
- [ ] Step 2：实现到通过（spec §2.6）；`domain-pages.test.tsx:114-163` 与 S2 的 `project view` 一行照允许清单删去（同一提交，预检 re-96 I6 ①）。新取数文件收入 swr 名单，读项目的收入 `project-address-route`。
- [ ] Step 3：变异照上表 T7 各行。门禁同 T2；`make build` 之后 S2 与默认配置的 chromium（`t7-s2`）。提交 `feat(M7/P6a): 视图页——以 viewId 为 key，读中、不存在、失败重试、停用四种状态与各自的页头，窄屏页头；每次挂载读自己的视图，列表照视图存着的筛选与显示，改动只进页面里的草稿；S2 去掉项目视图一格`。
- [ ] Step 4（**检查点 1**）：`final.sh cp1`；主控推送、核分支 CI，记 web 作业三步与 e2e 两片。

### T8 另存、更新视图与在途停用

**Files**：
- 改：`C/components/work-item-filters/filters-hoc/{project-level.tsx,shared.ts}`（212、33）、`C/components/rich-filters/filters-row.tsx`（185）、`C/components/issues/issue-layouts/roots/{project-layout-root,planning-page-layout-root}.tsx`（113、128）、`project-view-layout-root.tsx`（T7 新建；传 `enableSaveView`、`enableUpdateView`）、`C/hooks/planning-page-domains.ts`（74）、`C/store/issue/helpers/page-preferences-filter.store.ts`（`confirmed`）、`save-view-collapsed.test.tsx` → `save-view-offered.test.tsx`、`IL/roots/cycle-layout-root.test.tsx:91`（132）、`IL/roots/module-layout-root.test.tsx:92`（133）（三份照允许清单）；en、zh-CN（`common.filter_options.{save_view,update_view,confirming}`）。`IL` = `C/components/issues/issue-layouts`。
- 改（R29 的在途停用，L!）：`C/store/views/{view.store,view-changes}.ts`（`isUpdating(viewId)`）；`C/components/issues/filters.tsx`（67）、`C/components/issues/mobile-header-filters.tsx`（56）、`C/components/issues/issue-layouts/filters/header/layout-selection.tsx`（64）、`mobile-layout-selection.tsx`（56）、`C/components/work-item-filters/filters-toggle.tsx`（28）与 `C/components/rich-filters/filters-toggle.tsx`（93）、`C/components/rich-filters/filters-row.tsx`（185）（各加一个可选的 `disabled`，默认不停用）；`IL/spreadsheet/columns/header-column.tsx`（112，`:42` 的 `CustomMenu` 读 `DraftBusyContext` 停用）、`IL/calendar/dropdowns/options-dropdown.tsx`（146，`:53-69` 的月 / 周与显示周末读它停用，预检 re-96 I4）、`IL/empty-states/project-issues.tsx`（77，`:51-53` 的清除读它）；`C/components/issues/issue-layouts/roots/project-view-layout-root.tsx`（T7 新建；提供 `DraftBusyContext`、给筛选行传 `disabled`）；`V/(detail)/[viewId]/{header,mobile-header}.tsx`（116 与 T7 新建；传 `isUpdating(viewId)`）。新 `IL/draft-busy.ts`（`DraftBusyContext`，约 10 行）。
- 新：可能 `filters-hoc/use-project-view-saving.ts`；HOC、筛选行与页头停用的新测试。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t8-red`）：
  - `confirmed(id)` 返回拷贝；HOC 的 `hasAdditionalChanges` 在改了显示之后重算（依赖挂在 `issuesFilter.filters[id]` 上，`confirmed` 不是响应式的）；
  - 四个页面给“Save view”（管理员、成员；访客与开关关着不给）；迭代页、模块页另存的体带 `cycle_id`、`module_id`；显示与属性是已确认的线上值；名字的初值为空（R30）；
  - “更新视图”：只给所有者（被降为访客的所有者不给）、不看锁；视图页改了筛选之后出现（`canUpdateView` 为真，R10 的 T8 一半）；没有改动时不出现（`hasAdditionalChanges` 比线上值）；扣住回答时按钮是 “Confirming”；成功之后提示、按钮消失、焦点落在筛选行的添加筛选（筛选行不在时落在筛选开关）、基线是回答（回答整理过筛选的一格）、草稿是回答；失败之后提示、按钮留着、焦点不动、草稿与基线不动（`navigation.views_disabled`、`navigation.view_not_found` 各一格）；体的显示在轮到时读（HOC 加假队列的那一层：扣住前一次写、点击、改草稿的显示、放开，预检 re-96 O1）；换会话之后答到不提示；离开页面（HOC 卸下）之后答到：不提示、不 `adopt`、不 `fetchFilters`、没有草稿（预检 re-96 I2）；
  - 在途停用（R29）：扣住回答时筛选行的条件、添加筛选、“Clear all”、“Save view” 停用，视图页头的布局（宽屏与窄屏）、显示、筛选开关里的添加筛选停用，表格列头的排序菜单停用（一格），日历的选项下拉停用（一格，预检 re-96 I4；页面故事不另加，单测承担），有筛选时空态的“清除筛选”停用（一格）；放开之后（成功与被拒各一次）都恢复；项目页的页头、筛选行、列头、日历的选项与空态在同样的操作下不停用；
  - 筛选行的五个字面量与 HOC 的提示经 `t()`（`@/lib/fake-i18n` 的键），en 的文字不变；按字面找 “Clear all”“Save view” 的三份既有单测照允许清单改键。
- [ ] Step 2：实现到通过（spec §2.7）；`save-view-collapsed.test.tsx` 照允许清单 `git mv` 并翻过来，两份 M6 根测试改键。HOC 不超过 220（`t8-wc`）。
- [ ] Step 3：变异照上表 T8 各行（两层的：HOC 的 `isDisabled` 与 shared-state 的 `canUpdateView` 一起改，留只改一层的对照 `t8-ctl-update-one-layer`；shared-state 的变异之后核 dist 已还原）。门禁同 T2；死成员整份比对（`t8-dead-after`）；`make build` 之后 chromium 的 m4-filters、m4-list 与 m6-pages（`t8-m4-filters`、`t8-m4-list`、`t8-m6-pages`）。提交分两个（预检 re-96 O7，整审照此分范围）：
  - **提交 A** `feat(M7/P6a): 四个列表页的另存与视图页的更新视图——另存取页面已确认的线上值、迭代与模块带自己的词；更新视图等 nerve 的回答，失败留着按钮，成功之后基线是回答、焦点落在添加筛选，离开页面之后答到不跟进；筛选行与 HOC 的英文改键`。文件：`filters-hoc/{project-level.tsx,shared.ts,use-project-view-saving.ts}`、`rich-filters/filters-row.tsx`（`catch`、`t()`、添加筛选的 ref）、`roots/{project-layout-root,planning-page-layout-root,project-view-layout-root}.tsx`（`enableSaveView`、`enableUpdateView`）、`planning-page-domains.ts`、`page-preferences-filter.store.ts`（`confirmed`）、`save-view-offered.test.tsx`、两份 M6 根测试、en 与 zh-CN、HOC 与筛选行的新测试。变异：`t8-mut-saveas-guest`、`t8-mut-saveas-collapsed-cycle`、`t8-mut-root-no-update`、`t8-mut-no-view-terms`、`t8-mut-saveas-control-words`、`t8-mut-confirmed-live`、`t8-mut-update-non-owner`、`t8-mut-update-guest-owner`、`t8-mut-additional-control-words`、`t8-mut-additional-confirmed-only`、`t8-mut-update-resolves-on-failure`、`t8-mut-baseline-current`、`t8-mut-no-adopt`、`t8-mut-body-at-click`、`t8-mut-update-follow-unmounted`、`t8-mut-update-other-session-toast`、`t8-mut-update-focus-lost`、`t8-mut-no-catch`、`t8-mut-literal-update`，两层的对照 `t8-ctl-update-one-layer`。
  - **提交 B** `feat(M7/P6a): 更新视图在途时，视图页上改草稿的入口停用——筛选行、页头的布局与显示、表格列头的排序、日历的选项、空态的清除，落定之后恢复`。文件：`C/store/views/{view.store,view-changes}.ts`（`isUpdating`）、`C/components/issues/{filters,mobile-header-filters}.tsx`、`IL/filters/header/{layout-selection,mobile-layout-selection}.tsx`、两份 `filters-toggle.tsx`、`rich-filters/filters-row.tsx`（`disabled`）、`IL/draft-busy.ts`、`IL/spreadsheet/columns/header-column.tsx`、`IL/calendar/dropdowns/options-dropdown.tsx`、`IL/empty-states/project-issues.tsx`、`roots/project-view-layout-root.tsx`（提供 `DraftBusyContext`、传 `disabled`）、`V/(detail)/[viewId]/{header,mobile-header}.tsx`、在途停用的新测试。变异：`t8-mut-row-busy-enabled`、`t8-mut-root-no-busy`、`t8-mut-sort-busy-enabled`、`t8-mut-calendar-options-busy-enabled`、`t8-mut-empty-clear-busy-enabled`、`t8-mut-header-busy-enabled`、`t8-mut-busy-stuck`、`t8-mut-header-default-disabled`。
  - 两个提交各自过门禁（提交 A 之后视图页已有“更新视图”、还没有在途停用：分支内的窗口，不进 main）。

### T9 命令面板

**Files**：改 `C/components/power-k/menus/views.tsx`（33）、`C/components/power-k/config/navigation/project-commands.ts`（258，`:18`、`:171` 的 `IProjectView` 换成 `View`，与 `menus/views.tsx:11` 同一提交，预检 re-96 m2）、`C/components/power-k/ui/pages/open-entity/project-views-menu.tsx`（34）、`C/components/power-k/ui/modal/{search-menu,search-results}.tsx`（107、100）、`C/components/power-k/menus/search.test.tsx`（允许清单）；新 `C/components/power-k/ui/modal/use-view-search.ts` 与测试（搜索组的格不进 365 行的 `search-menu.test.tsx`）；`.oxlintrc.json`；`K`。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t9-red`）：菜单以 `["PROJECT_VIEWS", projectId]` 读（与列表页同时挂着时共用一次在途的读，先例 `search-menu.test.tsx:335-345`）、门是入口规则、在途、被拒、空三种；搜索在 store 里没有这个项目的视图时开面板读一次、有了不读（R20 的代价，spec §2.8）；行的类型是 `View`；搜索只在项目范围给 “Views” 组（标题读 `views`），不分大小写的子串、nerve 的次序，被拒或在途时这组不显示、不挡别的组，“没有结果”在所有组都空时才给；选中关面板并转到视图页。
- [ ] Step 2：实现到通过（spec §2.8）。
- [ ] Step 3：变异照上表 T9 各行。门禁同 T2；`make build` 之后 chromium 单跑 `t9-palette`（Global Constraints）。提交 `feat(M7/P6a): 命令面板的视图——菜单与列表页同一个键，搜索的 Views 组只在项目范围、store 里没有才读、本地过滤；命令面板不再读 IProjectView`。

### T10 删除与 `old-view-urls` → **检查点 2**

**Files**：删 `C/services/view.service.ts`（42）、`C/store/project-view.store.ts`（278）、`C/hooks/store/use-project-view.ts`（17）、`web/apps/web/helpers/views.helper.ts`（26）、`PKG/utils/src/project-views.ts`（97）与桶导出 `:19`；改 `C/store/root.store.ts`（去掉 `projectView`）、`root.store.test.ts:106`（允许清单）、`PKG/types/src/views.ts`（58）、`PKG/constants/src/views.ts`（26）与桶导出 `PKG/constants/src/index.ts:32`（文件空了时）、`PKG/types/src/index.ts:25`（文件空了时）、`C/store/favorite.store.ts`（391）、`C/hooks/use-favorite-item-details.tsx`（62）；`K`（`old-view-urls`）。

**Steps**：
- [ ] Step 1：读者表（收据 `t10-readers-before`）：spec §2.9 的每一项 `git grep`，写进台账（还被 P6b 的文件读的留下并写明）。
- [ ] Step 2：先写收藏的测试，看到失败（收据 `t10-red`）：视图收藏的名字与图标取新 store 所持的视图，没有时取收藏自己的。实现；删 `viewStore`、`removeFavoriteEntityFromStore` 与它的登记和四个调用。
- [ ] Step 3：删旧层与类型、常量、帮手；`RootStore` 只剩 `views`，`root.store.test.ts:106` 回到 20；读者表再跑一次（`t10-readers-after`：只剩 P6b 的读者）。加 `old-view-urls`（hit 与 miss、文件样例写全），跑 `node tools/keywords.mjs`。
- [ ] Step 4：变异照上表 T10 各行。门禁同 T2；`pnpm --filter @nerve/types test`（若有）、`@nerve/utils`、`@nerve/constants`；死成员整份比对（`t10-dead-after`，视图的行逐行写去处）；oxlint 计数（`t10-oxlint-count`）。提交 `refactor(M7/P6a): 删除 Plane 的旧视图层——service、store、hook、项目视图独占的类型、常量与帮手，收藏不再读写旧 store；加 old-view-urls`。
- [ ] Step 5（**检查点 2**）：`final.sh cp2`；推送之前照 P4 spec §2.11 估出 e2e 两片；主控推送、核分支 CI。

### T11 H12 (page)

**Files**：新 `E/fixtures/view-pages.ts`（约 200 行）、`E/stories/collaboration/h12-project-views-list-page.spec.ts`、`h12-project-view-page.spec.ts`、`h12-project-view-layouts-page.spec.ts`（四种布局与翻页，预检 re-96 I3）；`h12-project-view-page` 过 400 行时另存一组另起 `h12-project-view-saving-page.spec.ts`（拆分出口，预检 re-96 m7 ③）；改 `E/fixtures/mounts.ts`（72，允许清单）。S2 已在 T7 改。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（收据 `t11-red-page-list`、`t11-red-page-view`、`t11-red-page-layouts`，拆出另存时加 `t11-red-page-saving`），看到断言在被改处失败。
- [ ] Step 2：写故事（spec §2.11 的表），其中：“Add view” 的请求体钉住合同默认的显示与属性（等于 `viewDefaults`、没有 `filters`，R28）；弹窗里没有布局与显示；新建与另存的名字初值为空（R30）；“更新视图”扣住回答的一步断言筛选行的条件与添加筛选、“Clear all”、“Save view”、页头的布局与显示都停用，放开之后（成功一次、被拒一次）恢复（R29）；读表照实数（回答整理过筛选时成功之后多一次列表的读，预检 re-96 O3）；离开再进来（两秒之内）恰再一个 `GET /api/v0/views/{id}`（I1）；列表的查询包含 `savedQuery` 的各项（`E/fixtures/views.ts:184`，不是相等）；列表页的 “Filters” 与视图页的 “Display”（`FiltersDropdown`）各一步：名字、`aria-expanded`、Escape 之后焦点回到按钮（`HO/M3-closeout.md:56`）；页头在四种状态下的面包屑与窄屏页头；迭代页、模块页的另存先加一个条件（没有条件不给 “Save view”，预检 re-96 O4）；四种布局各存一个视图、各打开一次（请求的查询与列出的行）、翻页一次、分组头的“+”与日历的“添加已有”。前置条件只经接口；视图的内容经接口读回（`viewCalls(api).read`，`E/fixtures/views.ts:119-125`）与请求体、回答比较，数据库只经 `viewRow`（`:158`，软删与所有者）；扣住回答经 `holdAnswer`；读失败的一步只在 Chromium、只在视图页的代码载入之后装路由；换页、关开关、删视图之前先等在途的读答完。
- [ ] Step 3：跑法：`make build`；chromium 聚焦；`--config=playwright.m7-pages.config.ts` 三引擎；新文件 chromium `--repeat-each=3`（`t11-repeat`）。记每个 `test()` 三引擎的时长；每份不过 400 行。提交 `test(M7/P6a): H12 的页面故事——视图列表页的矩阵、弹窗与菜单，视图页的四种状态、另存与更新视图、四种布局与翻页；mounts 加视图的读`。

### T12 H20 (page)

**Files**：新 `E/stories/collaboration/h20-view-pages.spec.ts`（过 400 行时命令面板一步另起 `h20-views-palette-pages.spec.ts`）。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（`t12-red-page`）。
- [ ] Step 2：写故事（spec §2.11 的表）：迟到的回答（列表的读、视图的读、“更新视图”、删除、换账号；视图的读与“更新视图”两步之后各回到原视图，断言又一个 `GET /api/v0/views/{id}`、列出的是新读到的，预检 re-96 I2）；以键盘“更新视图”，成功之后焦点在筛选行的添加筛选（三引擎记焦点，预检 re-96 m6）；键盘（弹窗的 Tab 次序：图标、名字、说明、Cancel、提交；嵌套 Escape：图标选择器开着时只关它；切换器；行菜单）；命令面板（菜单、搜索组、`create_view`、访客）。视图页的布局切换以键盘选（宽屏 `LayoutSelection` 的按钮、窄屏 `MobileLayoutSelection` 的菜单各一次）作 M2 交接 §3 `Dropdown` 一半“不适用”的证据（spec R26、R28）。
- [ ] Step 3：同 T11 的跑法与时长记录（`t12-repeat`）；记录焦点在三引擎各落在哪个控件。提交 `test(M7/P6a): H20 的视图页——迟到的回答不写进新页面，弹窗、布局切换与菜单的键盘，命令面板的视图`。

### T13 文档与交接（主控）

- **plane-diff**：spec §7 的八行进 `docs/v0/plane-diff.md` 行为差异一节（与 M7/P5a、P5b 的行同处）；Plane 的行号对照主目录的只读 `plane/` 逐一核过。
- **frontend-changes**：3.1 的 M7 一行写 P6a；3.2 加行：视图的数据层、`viewEntryOn`、列表页与宿主、弹窗、`PROJECT_VIEW` 与草稿、视图页、另存与更新视图（R7–R10）、命令面板、删除（旧层、`getTabIndex`、`projectMemberIds`）。
- **交接**：spec §5 的表，每条写“处理结果（M7/P6a）”（M7-04 `:71` 照 spec §5 的逐项写，`HO/M3-closeout.md:60` 写项目视图一半）；M2 交接 §3 的 `Dropdown` 一半写“不适用”（`single-select.tsx` 在 M4/P16 删、`LayoutDropDown` 在 T5 删，T12 在视图页的布局切换上的键盘证据）。
- **死成员**：重跑脚本（`t13-dead-final`），剩下的行写进阶段记录。
- **告警**：核对本段改到与新建的文件 oxlint 为零、计数等于上限（`t13-oxlint-files`）。
- **设计**：§12 的 P6a 一行链 spec、§14 的 P6a 一行（主控）。
- **阶段记录** `reviews/P6a-web-project-views-review.md`：§1 方法与范围、§2 验收 → Task、§3 实施中的裁定（十条 L! 写明；R1 多动的列表挂载的键、行的快捷操作、“添加已有”的请求类型，R29 多动的日历选项下拉一并列出）、§4 向负责人说明（P6a 之后 main 的样子；R28 弹窗只管身份、R29 在途停用的代价与被否的另一做法）、§5 门禁与 CI、§6 时长、§7 交给 P6b 与 M8、§8 文档、§9 审查与集成。
- 检查文档链接与关键词，提交 `docs(M7/P6a): …`。

### T14 门禁、整审与集成（主控）

1. **整分支审查**：一次 fresh 只读的整分支 deep-review（fable），重点是 Review Focus 的 1–7。Critical/Important 在一次修复轮里各有 RED/GREEN；Minor 记台账，主控裁定改否。之后一次限定复审。
2. **终检**：`final.sh fin`（六个页面配置）。
3. **推送前的估算**：照 P4 spec §2.11 估出 e2e 时长。
4. **最终分支 CI**：推送准确的 HEAD（`git rev-parse`），核分支 CI（`wait-run.sh`；排队 0 个作业约 20 分钟即停下报告）。
5. **合并**：`--no-ff` 合入 main，推送后以 `ls-remote` 核对。
6. **main CI**：核准确的 main CI。
7. **归档与清理**：归档 `.superpowers/sdd/M7-P6a-integration-<日期>`（`archive.py`），清理 worktree 与本地分支，远端分支核过已含于 main 之后删。
8. **回填**：spec A10 与设计 §14 由下一段（P6b）的 T1 回填。

## Review Focus

1. **R1 引擎（L!）**：`PROJECT_VIEW` 在每个分派点（spec §2.5 的清单：列表挂载的键带 `viewId`、日历的“添加已有”、`ProjectIssueQuickActions` 读上下文、静默的几处各有格与变异）；命令清单写全；两个 `switch` 穷尽、漏一处是编译错；没有进 M6 的两张表；视图页的列表与项目页的分开；项目页、迭代页、模块页不回归（m4-list、m4-filters、m4-detail、m6-pages）；`root.store.ts` 只加构造。
2. **R9、R10、R29 “更新视图”、基线与在途停用（L!）**：视图页不用 `usePageFilterExpression`；`onViewUpdate` 返回 Promise、失败 reject（`followInSession` 不 reject，HOC 自己 throw）；成功时不等价才 `resetExpression`、草稿换成回答；体的显示在轮到时读；`hasAdditionalChanges` 比线上值；不看 `is_locked`；筛选行的 `catch`；在途时视图页上每个改草稿的入口（筛选行、页头的布局与显示、表格列头的排序、日历的选项下拉、空态的清除筛选；以写者的 `git grep` 核全）都停用、落定之后恢复，来源只有 `isUpdating(viewId)`，项目页、迭代页、模块页不受影响；没有复用 `disabledAllOperations`；跟进只在 HOC 挂着时；成功之后的焦点；`confirmed` 的读者跟着 `filters[id]` 重算。
3. **R7、R8、R28、R30 另存与弹窗（L!）**：四个页面给、给谁；迭代、模块的词经表的新列、共用的根不分支；显示取 `confirmed` 的拷贝；P4 的测试翻过来；弹窗只有名字、说明、图标，“Add view” 的默认显示与属性是合同的、编辑不发显示；名字初值为空；M4/M6 的换算没有被导出；`LayoutDropDown` 删干净。
4. **R13、R14 命令面板与入口（L!）**：`viewEntryOn` 一处、五个读者；单测表两行不变、角色的格；菜单与列表页同键；搜索只在项目范围、不挡别的组；`create_view` 的门。
5. **R16、R21（L!）**：`projectMemberIds` 的读者重新核过再删、只删专属断言；筛选行与 HOC 的字面量改键、en 文字不变、新键只在清单里。
6. **会话分代与草稿（v0 §7.7）**：读的键带 `loginId`；写一个视图一个队列、体在轮到时算、只在确认时写；`await` 之后都经 `followInSession`；草稿卸下即丢（含 `filters[viewId]`）、迟到的播种不建回（草稿服务的代）、换会话随根重建；视图页的读每次挂载读自己的、重连不读；以 `viewId`、项目 id 为 key 的布局。
7. **矩阵与可访问**：§2.2 的每一格；弹窗的关闭、名字、`FieldError`、被拒的焦点、自然 Tab、嵌套 Escape；行菜单的名字与右键菜单；复制的失败；每个失败都提示、不 `console.error`。
8. **删除干净**：§2.9 的表逐项；读者表；`old-view-urls` 的样例；`project-address-route` 的名单与样例；swr 名单；死成员整份比对；告警等于上限。
9. **共用的外壳**：M4 的列表与浮层、M6 的命令面板与表驱动的根、P4 收起的 HOC、P5a 的收集箱测试替身；六个页面配置与整套默认 e2e。
