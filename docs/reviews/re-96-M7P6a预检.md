# M7 / P6a 预检

> 2026-11-24：结论 **修正后通过**（Ready after fixes）。严重 0、重要 6、次要 12、观察 7。合同在 `47bd860b` 上都撑得住：视图的五个操作、`ViewCreate`、`ViewUpdate`、`ViewFilters` 与显示的两个 schema 同 spec 写的（`NAV:19`、`:44`、`:138`、`:163`、`:199`、`:566-662`，`api/common.yaml:167-221`），“Add view” 的默认常量与服务端缺省时填的值、与 `E/fixtures/views.ts:52-80` 的 `viewDefaults` 三者相同，控件矩阵每一格都对得上 `rules.go:223`、`:228-230` 与 `view.go:52-73`，两个码的文案已在。spec、plan 与简报引的行号核了约 170 处（Task 依赖的全部，plan 里约 95 个文件的行数逐个量过，都对），偏差只有五处、各差一两行（m12 ①–④、m3 ⑤）。十条高风险裁定方向都对，没有发现裁定与代码或合同相悖；R10、R14、R16 照写就成立，R1、R9、R21、R29 各有漏的读者或漏的入口。要先改的六件事：① 视图页的读照 spec 的写法，离开之后两秒内再进来不发读、页面一直停在“加载”（P5b R27 的写法没有用上）；② 卸下之后才答到的播种读与“更新视图”的回答会把草稿建回来，下次进这个视图就不读了；③ R1 的缓解只盖住两个会抛错的 `switch`，其余分派点漏了是静默的，另有四处没有列，视图页的“四种布局、分页”在页面故事里没有证据；④ R29 漏了日历布局自己的选项，“任何改草稿的入口”不成立；⑤ T4 与 T5、T7 与 T8 的接缝做不出（宿主要挂的弹窗、视图页接上的 HOC 都在下一个 Task 才离开旧 store）；⑥ “没有红的提交”守不住两处：S2 的项目视图一格从 T7 起必红而 plan 到 T11 才改（检查点 1、2 过不了），R21 让 M6 的两份页面根测试各一格必红而它们在“不改”的名单里。
>
> - **目标**：[P6a spec](../v0/M7-collaboration/specs/P6a-web-project-views.md)、[P6a plan](../v0/M7-collaboration/plans/P6a-web-project-views.md)（未提交的草稿），次要目标台账的 `implementer-brief.md` 与 `final.sh`。依据台账 `rulings.md`（C-1–C-27 与 O-1–O-6 的裁定）、`research/scope.md`、`research/questions.md`，[M7 设计](../v0/M7-collaboration/M7-design.md) §2、§3.7、§3.11、§7、§12，[总体设计](../v0/v0-design.md) §7.7，合同 `api/modules/navigation.yaml`，先例 [re-95](re-95-M7P5b预检.md)（含 §6）与 [P5b 阶段记录](../v0/M7-collaboration/reviews/P5b-web-intake-files-embed-review.md) §3、§7。
> - **基线**：main `47bd860b`（P5b 的 noff，`47bd860ba104145a37a66687c12d6db7b49658b4`），worktree `codex/m7-p6a`。

下文 `C/` = `web/apps/web/core/`，`P/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/`，`V/` = `P/views/`，`VC/` = `C/components/views/`，`IL/` = `C/components/issues/issue-layouts/`，`PKG/` = `web/packages/`，`E/` = `e2e/`，`K` = `tools/keywords.json`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`D` = M7 设计，`NAV` = `api/modules/navigation.yaml`，`PW/` = 只读的 `plane/apps/web/`，`L/` = 台账 `.superpowers/sdd/P6a-web-project-views/`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git 状态（只用了 `git status`、`git grep`），不起容器，不跑测试、构建与安装。跑过的只有 `grep`、`sed`、`wc`、`ls`、`cat` 与台账的 `doc-links.py`（写本文件之前 239 个文档、2416 个链接、0 断，两份草稿在内；写完之后 240 个、2422 个、0 断）；管道里另用 `cut`、`head`、`tr`（一次 `awk`）裁剪输出，都只读。`node tools/keywords.mjs`、oxlint、knip、deadsym、vitest、Playwright 都没有运行。审查前后 `git status` 只有两份草稿是未跟踪的（之后多本文件）。

worktree 没有装依赖。`swr@2.4.2` 的行为是读主目录 `node_modules` 里装着的源码核的（只读：`dist/index/index.mjs` 的去重标记、答到之后的清理、挂载时是否再验证）。`plane/` 只用 `sed`、`grep` 读了十来处，没有在里面跑 git。

对照代码核了：

- 列表引擎：`PKG/types/src/issues/issue.ts`、`C/hooks/store/use-issues.ts`、`C/hooks/use-issues-actions.tsx` 全文、`use-issue-layout-store.ts`、`use-issue-list-mount.ts`、`use-page-issue-filters.ts`、`use-group-dragndrop.ts`、`use-planning-page-issue-actions.ts`、三份 `*-preferences-follow.ts`、`use-page-filter-expression.ts`、`planning-page-domains.ts`；`C/store/issue/helpers/{page-preferences-filter,page-preferences-issues,planning-page-issues,base-issues}.store.ts`、`project/{issue,filter}.store.ts`、`root.store.ts:196-214`；`IL/` 的四个布局根、`command-host.tsx`、`add-existing-dialog.tsx`、`quick-action-dropdowns/project-issue.tsx`、`empty-states/`、两个分组头、日历的快速添加与选项下拉、表格的列头、`roots/{project,planning-page}-layout-root.tsx`；`PKG/constants/src/issue/{filter,common}.ts`。`EIssuesStoreType.PROJECT` 的全部产品读者与按类型的映射表用 `git grep` 列过。
- 富筛选与 HOC：`PKG/shared-state/src/store/rich-filters/filter.ts`（`hasChanges`、`canSaveView`、`canUpdateView`、`resetExpression`、`updateView`）、`work-item-filters/filter.store.ts:60-125`、`C/components/work-item-filters/filters-hoc/{base,project-level,shared}.ts(x)` 全文、`C/components/rich-filters/{filters-row,filters-toggle}.tsx`、`C/components/issues/{filters,mobile-header-filters,use-page-display}.ts(x)`、页头的布局与显示组件。页面显示的全部写者（`EIssueFilterType.DISPLAY_*`）用 `git grep` 列过。
- 视图的旧层与页面：`VC/` 的全部文件、`V/(list)/*`、`V/(detail)/*`、`C/store/project-view.store.ts`、`C/services/view.service.ts`、`PKG/{types,constants,utils}` 的视图文件与桶导出、`helpers/views.helper.ts`、`C/components/common/filters/{created-at,created-by}.tsx`、`quick-actions-helper.tsx`、`C/components/modals/project-level.tsx` 与它的测试、`C/store/favorite.store.ts`、`use-favorite-item-details.tsx`；`useProjectView`、`IProjectView`、`projectMemberIds`、`getTabIndex`、`LayoutDropDown`、`useViewMenuItems` 与旧类型常量帮手的读者（`git grep`）。
- 入口与命令面板：侧栏、页签栏、`project-commands.ts`、`creation/command.ts`、`commands.test.tsx:60-145`、`C/lib/intake-entry.ts`、`use-project-fetch.ts:56-94`、`power-k/ui/modal/{search-menu,search-results,use-planning-search}.ts(x)`、`search-menu.test.tsx:1-130`、`:244-345`、打开命令面板的 12 份故事（有没有精确的读表）。
- 会话与取数：`C/lib/{use-session-swr,in-session,one-at-a-time,reconciled,fake-session-swr}.ts`、`PKG/constants/src/swr.ts`、`C/store/root.store.ts:84-106`、`root.store.test.ts:98-116`。
- 合同与服务端：`NAV` 的五个操作与四个 schema、`api/common.yaml:167-221`、`access/domain/rules.go:218-232`、`navigation/domain/{view,content}.go`、`internal/shared/list_settings.go:87-95`、`C/lib/error-messages.ts:105-106`、`permissions.store.ts:68-75`。
- 门禁与 e2e：`L/{gates,final}.sh`、`Makefile:165-166`、`E/playwright.config.ts` 与六个页面配置的 `testMatch`、`grep`、`E/fixtures/{views,mounts,power-k}.ts`、简报点名的 12 个帮手的行号、`E/stories/smoke/s2-issue-list-removal.spec.ts:1-130`、`p8-project-preferences.spec.ts:284-397`、`g19-work-item-side-page.spec.ts:112-175`、`K` 的 `project-address-route`、`copy-hook`、phase、`.oxlintrc.json` 的 swr 名单、`web/apps/web/package.json:10`、`IL/roots/{save-view-collapsed,cycle-layout-root,module-layout-root}.test.tsx`、`fake-planning-page.tsx`、`C/lib/fake-i18n.ts`、en 的 `common.json`、`project.json`。
- 交接与文档：`HO/M3-closeout.md`、`M4-closeout.md`、`M6-closeout.md`、`M2-closeout.md` 被引的每一行，设计 `:39`、`:47-48`、`:118-120`、`:246`、`:308`，P5b 阶段记录 §3、§7，`L/progress.md`。

没有核实的：

- oxlint 的 18 条、死成员、knip 在各 Task 之后的结果、任何 vitest 与页面故事的时长（都没有运行）。
- I1、I3 (a) 的后果是按 swr 的源码与 store 的代码推出的，没有在浏览器里看。
- `use-issues.ts` 的 `switch` 开在泛型参数上，`satisfies never` 在泛型上能不能收窄没有核（留给 plan 的 `t6-exhaustive-probe`；不成时用 spec 给的另一写法，`satisfies Record<EIssuesStoreType, …>` 的表）。
- `View.logo_props` 与收藏图标组件的类型是否相容（T10 的 `use-favorite-item-details.tsx`）。
- `questions.md` 的 Q18–Q24 只读了汇总表。
- 没有逐个核的行号：`use-copy-link.ts:42`、`context-menu/root.tsx:99`、`module-list-item-action.tsx:76`、`VC/views-list.tsx:43-48`、`selected-options-display.tsx:26-31`、右键的两个故事先例、设计 `:159`、`:240-243`、`:248`、`:352`、`frontend-changes.md:206`、zh-CN 的文案键。
- spec §7 的八条差异里，Plane 的行号核了十来处（都对，§3.7）；第 7 条没有对 Plane 的源码。

## 2. 发现

### 严重

无。

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §2.6“读与草稿”“草稿的寿命”、§6“swr 的重连”；plan T7 Step 1、性质表的 `t6-mut-get-reads-every-time` | **视图页的读照 spec 的写法，离开之后两秒内再进来不发读，页面一直停在“加载”。** 页面以普通的 `useSessionSWR(["PROJECT_VIEW_PAGE", slug, viewId], fetchFilters)` 读，卸下时丢草稿与 `filters[viewId]`。SWR 在一次读答到之后把这个键的在途标记再留一个去重窗口（默认 2000 毫秒，视图页没有缩短）；窗口里新挂上的 hook 看到标记就不再验证，取数函数不被调用。于是“打开 A → 回列表或换到 B → 两秒内回到 A”：没有 `GET`、没有错误，`filters[viewId]` 已被丢掉，没有任何东西会让页面离开“加载”（会话取数不随聚焦重取，也不重试）。后退前进、切换器来回都是这个形状；H12 (page) 的“离开再进来草稿没了”与 T7 的“后退回到 A 是新读到的”正落在窗口里，结果取决于时间。项目页没有这个问题，是因为它的偏好不随卸下丢掉。这是 P5a 整审 I1、P5b R27 已经定过写法的那一类读（每次挂载读自己的），`use-session-swr.ts` 的注释写的就是它，视图页没有用上。另：照那个写法关掉重连重读之后，“重连再验证读到草稿、不发请求”没有触发者了，`t6-mut-get-reads-every-time` 的格要换读者（“更新视图”成功之后的那次 `fetchFilters`）。 | `C/lib/use-session-swr.ts:19-25`、`:38-46`；`PKG/constants/src/swr.ts:7-13`（`revalidateOnMount: true`，没有 `dedupingInterval`）；`swr@2.4.2` 的 `index.mjs:347`（有标记且去重时不发新请求）、`:415`（答到之后才定时清标记）、`:577-590`（挂载时有标记就不再验证）；先例 `C/hooks/use-issue-list-mount.ts:62`；re-95 I4 与 §6 的 R27 | spec §2.6 写明视图页的读是“每次挂载读自己的”：放进自己的文件（收入 swr 名单），`{ revalidateOnMount: false, revalidateOnReconnect: false }`，挂载时 `mutate()` 一次（不带数据的 `mutate()` 先删标记、必发请求）；“重连不丢草稿”改写为“重连不读”，§6“swr 的重连”一条随之。T7 加一格：真的 hook 与真的 SWR（同一个缓存 provider），A 答到之后立刻卸下再挂上，恰好再发一个 `GET`、页面离开加载；变异 `t7-mut-plain-mount-read`（回到普通的挂载读）被它杀死。H12 (page)“离开再进来”一步断言第二个 `GET /api/v0/views/{id}`。 |
| I2 | spec §2.6 的 `ViewDraftService` 与“草稿的寿命”、§2.7“更新视图”成功的跟进、§2.11 的 H20 一行；plan T6、T7、T8 的 Step 1 与性质表 | **卸下只丢“已有的”草稿；之后才答到的两种回答会把草稿建回来，下次进这个视图就不读了。** ① 播种的读在途时离开（H20 的“经切换器换视图之后答到”正是这一步）：`discard` 这时没有东西可丢；回答到了，`get` 照常播种，基类的 `fetchFilters` 认它仍是这个 id 最新的一次读，把值写回 `filters[viewId]`。② “更新视图”在途时离开：跟进只核会话，同一会话里照样提示、`adopt`、`fetchFilters`——H20 写的是“没有提示与跳转”，spec 没有给它机制，T8 只有“换会话之后答到不提示”一格。两种情形之后草稿都在，这一会话里再进这个视图：`get` 有草稿就不读，0 个 `GET`，列出的是那次旧回答（别人此后改了视图也看不到），违反 A6 的“再进来以新读到的播种”与“进页面恰一个视图的读”。子类够不着基类的读序号与 `values`（都是 `private`），T6 又规定不碰基类，所以这不是实现时顺手能补的。扣住回答就能确定地重现，不靠时序。 | `C/store/issue/helpers/page-preferences-filter.store.ts:127-130`、`:175-190`；`C/lib/in-session.ts:40`；spec §2.6“`get(viewId)`：有草稿给草稿”、§2.7“换了会话两个跟进都不调”；plan T8 Step 1；先例 `C/components/modules/actions-host.tsx:54-58` 的 `mounted`、re-95 §6 的 R29 | spec 写明草稿按视图有“页面还在不在”的代：`discard` 作废在途的播种（迟到的回答只进 `ViewStore.details`，不建草稿、不写 `filters`），`adopt` 只在草稿还在时替换；HOC 的两个跟进只在它还挂着时做（挂载的范围，同弹窗与宿主）。机制放在哪一层由起草者定（草稿服务自己的代，或 `ProjectViewIssuesFilter` 重写 `fetchFilters`），不改基类成员的可见性。T6 加格与变异 `t6-mut-late-seed-revives`（扣住读、`discard`、放开、再 `get` 恰再读一次）；T8 加格与变异 `t8-mut-update-follow-unmounted`（离开之后答到：不提示、没有草稿）；H20 在两步之后各回到原视图、断言又一个 `GET`。 |
| I3 | spec §2.5“分派点”“穷尽”“命令”、§6 第一条；plan T6 的 Files、行数约束、性质表；spec §2.11 的 `h12-project-view-page` 一行 | **R1 的缓解只盖住两个会抛错的 `switch`；其余分派点漏了是静默的，另有四处没有列。** 没列的：(a) 列表挂载的键——M6 为迭代、模块加了“哪一页”一段，视图页不加，同一项目里布局选项相同的两个视图共用一个键（筛选不在键里，`fetchPage` 自己从偏好读）：A 的列表读在途时换到 B，B 的挂载并进 A 的那次读，B 自己的读不发，B 的筛选行下面列着 A 的查询的结果。(b) 日历某一天的“添加已有工作项”：入口把页面的 store 类型交给命令宿主，请求的类型只认项目页与两种计划页，宿主以这个类型的 `addIssuesToView` 发；spec 的命令清单没有 `addIssuesToView` 与 `createIssueAsGiven`，而 `IssueActions` 里 `addIssuesToView`、`createIssueAsGiven`、`updateIssue`、`dropIssue`、`archiveIssue` 都是可选成员，新 hook 少给一个不报错。(c) 行的快捷操作：spec 列了 `project-issue.tsx:35` 的联合，但四个 `project-root.tsx` 传的是 `ProjectIssueQuickActions`，它把类型写死为 `PROJECT`，spec 又写明四个根不改——扩了的联合没有人传，视图页上行的“编辑”“复制”经项目页的 store 写，不进视图页自己的列表。(d) `use-issues-actions.tsx` 今天没有 `WORKSPACE_DRAFT` 的 `case`（落进 `default`），加了 `satisfies never` 就得补；新 hook 还要一行无条件的调用——“只加一个 `case` 与一个导入”做不到。静默的：四个布局根与行的联合都在 `as` 之后（不扩也编译）；`ISSUE_STORE_TO_FILTERS_MAP` 以可选链读，漏了是“显示”里没有选项；`use-page-issue-filters.ts` 的 `switch` 没有 `default`，漏了是 `undefined`；空态的 `default` 是 `null`；分组头的“+”在点击时才抛。plan 的 T6 性质表只有 7 行，没有一行对着这些；H12 (page) 的视图页故事只换一次布局，设计写的“四种布局、分页”在页面上没有证据。 | `C/hooks/use-issue-list-mount.ts:27-29`、`:48-62`；`C/store/issue/helpers/page-preferences-issues.store.ts:15-18`、`:36-42`；`IL/calendar/quick-add-issue-actions.tsx:43`、`:57-66`；`IL/add-existing-dialog.tsx:25`；`IL/command-host.tsx:139`；`C/hooks/use-issues-actions.tsx:37-57`、`:68-72`、`:74-90`、`:93-97`（项目页的命令是文件内的私有常量、写死 `PROJECT`）；`IL/quick-action-dropdowns/project-issue.tsx:35`、`:169-171` 与它的六个使用处；`C/components/issues/use-page-display.ts:48`；`C/hooks/store/use-page-issue-filters.ts:21-38`；`IL/empty-states/index.tsx:20-33`；`IL/list/headers/group-by-card.tsx:95-96`；`D:39` | spec §2.5 的分派点补四处并各写一句怎么做：(a) 键带路由的 `viewId`；(b) 视图页日历的“添加已有”同项目页（只给那一天），请求的类型收 `PROJECT_VIEW`，命令清单写全 `addIssuesToView`、`createIssueAsGiven`；(c) 定下 `ProjectIssueQuickActions` 怎么得到类型（读 `IssuesStoreContext`、只在 `PROJECT_VIEW` 时换，个人页的两个根仍是 `PROJECT`；或另给视图页一个，四个根就要改），“四个根不改”随之改写；(d) 行数约束写成“一个导入、一行调用、两个 `case` 与 `default` 的穷尽”。T6 的性质表按分派点各一格一变异（至少：键里的 `viewId`——扣住 A 的列表读、换到 B、B 发自己的读；“显示”有选项；看板与列表的本地筛选；空态；分组头的“+”；行的编辑与复制写进视图页的列表；日历的“添加已有”；拖放）。H12 (page) 加：四种布局各存一个视图、各打开一次（请求的查询与行），翻页一次；放不下 400 行就另起一份（同 H20 的写法）。 |
| I4 | spec §2.7 R29 的末句“至此视图页上任何改草稿的入口在途时都用不了”、§3 R29、§8 O-6、A7；plan T8 的 Files、Step 1、性质表、Review Focus 2 | **R29 不完整：日历布局自己的选项（月 / 周、显示周末）也改草稿，在途时照常可用。** 页面的显示在列表引擎里一共三个写者：页头（R29 管了）、表格列头的排序（O-6 管了）、日历的选项下拉（没有管）。后者写进草稿的 `display_filters.calendar`（视图页的 `updateFilters` 不看给的 id，写的就是这个视图的草稿）。“更新视图”在途时改它：请求体轮到时已经算过，回答里是旧值，成功之后 `adopt` 把草稿换成回答，这次改动静默丢失——正是 R29 要挡的事，也是 O-6 找出另两处时用的同一条理由。 | `IL/calendar/dropdowns/options-dropdown.tsx:53-69`；`IL/calendar/base-calendar-root.tsx:169-173`；全部写者的 `git grep`：`C/components/issues/use-page-display.ts:46-52`、`IL/spreadsheet/base-spreadsheet-root.tsx:95`、`IL/calendar/base-calendar-root.tsx:171`；`C/store/issue/helpers/page-preferences-filter.store.ts:253-258`（合并 `calendar`） | 日历的选项下拉作 `DraftBusyContext` 的第三个读者（在途时停用它的按钮）；spec §2.7、R29、A7 与 plan T8 的 Files（`options-dropdown.tsx`）、Step 1（一格）、性质表（`t8-mut-calendar-options-busy-enabled`）补上；H12 (page) 扣住回答的那一步在日历布局下再看一次，或写明由单测承担。R29 的“任何入口”以上面 `git grep` 的三处为准，写进 spec。 |
| I5 | plan T4（Files、Step 1–2、允许清单的 `follow-ups.test.tsx` 一条）与 T5；plan T7（Step 1 的“`canUpdateView` 为真”、`t7-mut-view-advances-baseline`）与 T8；spec §2.3“宿主”、§2.6“根” | **两处 Task 的接缝做不出。** ① T4 建宿主、让它“持有新建、编辑、删除三个弹窗”，行的菜单只调 `openDialog`；而新建 / 编辑弹窗到 T5 才改：T4 结束时它仍收 `IProjectView`、经旧 store 与旧地址写，它的测试（替身是旧 hook）按清单留到 T5。新列表页拿到的是 `View`，交不进去（不是 `IProjectView` 的子类型）；硬接上，T4 的提交就是“列表读新接口、编辑与新建写旧地址（404）”，而 T4 的 Step 1 又要测“编辑经 `openDialog`”。② T7 的视图页给 HOC 传 `enableSaveView`、`enableUpdateView`，而 HOC 到 T8 才认新 store：T7 结束时它仍从旧 store 取视图（永远没有），所有者为假，`updateViewOptions.isDisabled` 为真，筛选实例的 `canUpdateView` 恒为假——T7 Step 1 的“改了筛选之后 `canUpdateView` 为真”与它的变异在 T7 绿不了；“Save view” 这时在视图页上给了，打开的是写旧地址的弹窗。检查点 1 正落在 T7 之后。 | `VC/quick-actions.tsx:83-90`；`VC/modal.tsx:10`、`:23`、`:36`、`:45`、`:64`；`PKG/types/src/views.ts:20-40`；`VC/follow-ups.test.tsx:14`、`:25`、`:87-131`；`C/components/work-item-filters/filters-hoc/project-level.tsx:45`、`:61-63`、`:64-91`；`PKG/shared-state/src/store/rich-filters/filter.ts:279-285` | 各定一种并改 Files、允许清单与 Step：① (a) T4 的宿主只持有删除，行的“编辑”、列表页的 “Add view” 与空态的“新建”在 T5 随弹窗一起接上（T4 的矩阵格只测给不给）；或 (b) 弹窗的数据来源与类型在 T4 就换（`follow-ups.test.tsx` 的弹窗一组移到 T4），T5 只做形状与关闭政策。② T7 的根不传 `enableSaveView`、`enableUpdateView`（T8 传，第一个读者的规矩正好），R10 的格与变异改为钉 `hasChanges`（基线没有被推进）与“没有 `PATCH`”；“`canUpdateView` 为真”移到 T8。 |
| I6 | plan“红的窗口”、允许清单的 T11 S2 一条与“不改、不受影响”一条、检查点 1 与 2；spec §2.11 的 S2、§2.7“文案” | **“没有红的提交”在两处守不住。** ① S2 的 `project view` 一格钉的是视图页今天的样子：读旧地址得 404、显示 “View does not exist”、旧地址的请求恰是那一个。T7 重写视图页之后它必红；plan 到 T11 才删这一行。中间的检查点 1（T7 之后）与检查点 2（T10 之后）各跑一次 `make e2e`（默认配置，整个 `stories/`），两次分支 CI 也跑——都过不了。② R21 把筛选行的 “Clear all” 改成 `t()`；M6 的两份页面根测试在 `fake-i18n`（文字即键）下按英文找这个按钮，各有一格随之必红，而 plan 把它们列在“不改、不受影响”，spec 写“单测按名字找的不受影响”。照规矩（清单没写的先停下报告），T8 一跑就停。 | `E/stories/smoke/s2-issue-list-removal.spec.ts:16`、`:83-89`、`:103`、`:115-123`；`L/gates.sh:25`、`Makefile:165-166`、`E/playwright.config.ts:9-10`；`IL/roots/cycle-layout-root.test.tsx:33`、`:91`；`IL/roots/module-layout-root.test.tsx:33`、`:92`；`C/lib/fake-i18n.ts:10-12`；`IL/roots/fake-planning-page.tsx:224-225`；`C/components/rich-filters/filters-row.tsx:82-83` | ① S2 的改动（spec §2.11 那一条）从 T11 移到 T7 的同一提交与允许清单；T7 到 T11 之间“视图不存在”只有单测的证据，写明。② 允许清单的 T8 加两条：`cycle-layout-root.test.tsx:91`、`module-layout-root.test.tsx:92` 的 `"Clear all"` 改为键 `common.clear_all`（断言的意思不变）；`save-view-offered.test.tsx` 的两处名字同改；spec 的那句改为“页面故事按名字找的不受影响；`fake-i18n` 下按字面找的三处改键”。 |

### 次要

| # | 位置 | 发现 | 须做的修正 |
|---|---|---|---|
| m1 | plan T4 Step 0、Step 2；spec §2.9 的 `projectMemberIds` 一行 | 读者核对的两句收据与事实不符：全仓还有一个同名的局部变量（`C/components/issues/issue-detail-widgets/sub-issues/title-actions.tsx:45`、`:94`，M3 交接 §7 说的就是它不算读者），`K` 的 `project-address-route` 也有三处写着这个名字（`K:2336` 的说明、`:2342` 的正则、`:2348` 的 hit 样例）。“只有两处视图与 store 自己、测试与替身”“只剩 store 的定义”照字面核不过。 | 两句改为“……与 `title-actions.tsx` 的同名局部变量、`K` 的规则文字”；删去 getter 之后 `K:2336` 的说明照同一句里 `currentProjectDetails` 的写法改（“P6a 删去、模式里的这一式只防它回来”），正则与样例留着。 |
| m2 | spec §2.9“`IProjectView`”一行、§2.8、A8；plan T9、T10 的 Files | 命令面板还有一个 `IProjectView` 的读者没有列：`C/components/power-k/config/navigation/project-commands.ts:18`、`:171`（`data as IProjectView`）。T3 只为入口规则改这份文件；T10 删类型时它编译不过，A8 的“`IProjectView` 不在命令面板里”也指着它。 | 列进 T9 的 Files（与 `menus/views.tsx:11` 同一提交换成 `View`）与 §2.9 的读者栏。 |
| m3 | spec §5 的交接表 | ① `HO/M3-closeout.md:60` 点了两个表单（`views/form.tsx:186` 与 `workspace/views/form.tsx:110`），表里写“关闭”，应是“项目视图一半关闭，工作区视图的随 P6b”。② `HO/M4-closeout.md` M7-05（`:91-103`）两项都在：`projectMemberIds`（`:92-98`）与 `getTabIndex`（`:100`）；表里只写了后者与 T5，前者由 T4 关闭、没有记。③ `HO/M3-closeout.md:56` 要“筛选的故事落地时断言按钮的名字、`aria-expanded`、Escape 之后焦点回到它”，表里指向 H12、H20 (page)，而 §2.11 的三行里没有这一步（列表页的 “Filters”、视图页的 “Display” 都是 `FiltersDropdown`）。④ M7-04（`HO/M4-closeout.md:71`）点名的各项没有逐项写：视图弹窗没有 IME 的格（在名字里以输入法确认时的 Enter / Escape）；“拒绝”“换宿主迟到答”有（T5、I2）。⑤ M3 §5 从 `:91` 起（表里写 `:90-99`）；“M7 收官报告”一条列了九条 L!，少 R29。 | 照改五处；③ 在 H12 (page) 的列表页与视图页各加一步；④ 照 P5b 的写法逐项列“本段做的 / 交给 P6b 的”，T5 加 IME 一格（或写明不适用的理由）。 |
| m4 | spec §2.8；plan T9 Step 1、性质表的 `t9-mut-menu-other-key`、“另跑的单配置” | ① “列表页读过之后面板不再读”只在去重窗口（两秒）里成立：每个 hook 挂上都会再验证，隔久了面板会再读一次（这也是对的）。照字面写出的格要么靠时间，要么写成错的性质。先例钉的是“同时挂着时只问一次”（`search-menu.test.tsx:335-345`），而且先例的读有“store 里没有才读”的门（`use-planning-search.ts:45-46`）。② 搜索的读什么时候发没有写死：外壳不读视图（R20），所以照“打开时读”，项目范围里每开一次面板就是一个 `GET …/views`，不搜索也发；这是 R20 的代价，写明。③ T9 之后没有单配置的回归：打开命令面板的故事有 12 份（`i18-search`、三份 `g19-*`、`h1-subscription-page`、`s6-dialogs`、`w11-guest-bounds` 等，都没有精确的读表，核过），多数不在六个页面配置里，要等到检查点 2 的 `make e2e` 才跑到。 | ① 性质改为“面板与列表页同时挂着时共用一次在途的读（同一个键）”，变异不变；② spec 写明读的时机（面板开着就读，还是有搜索词才读）与代价；③ T9 之后加 chromium 的单跑（`t9-palette`：`i18-search`、`g19-work-item-side-page`、`h1-subscription-page`）。 |
| m5 | spec §7 第 1、4 条 | ① Plane 新建视图的默认是按状态分组（`PW/core/components/views/form.tsx:46`，fork 里同在 `VC/form.tsx:46`），R28 的默认是不分组（合同的默认）；这条差异没有登记。② 第 1 条引的是 fork 里的旧 store（`C/store/project-view.store.ts:56`，T10 就删），plane-diff 该引 Plane 的 `PW/core/store/project-view.store.ts:58`。 | 第 4 条加一句“新建的视图不分组，Plane 按状态分组”；第 1 条换引用。 |
| m6 | spec §2.7“更新视图”成功之后；§2.11 的 H20 | “更新视图”成功之后按钮消失（`canUpdateView` 变假），点它时焦点在它上面，卸下之后焦点落到 `body`；在途时按钮是 `disabled` 的（`C/components/rich-filters/filters-row.tsx:92-100`）。P5b 为同类的事定过落点（R26）。键盘用户按了 Enter 之后丢了位置。 | spec 写一个落点（筛选行的添加筛选，或筛选开关）与机制；H20 (page) 加一步键盘的“更新视图”，三引擎记焦点。不做则写明并记给 M8。 |
| m7 | spec §2.11 的两份 H12 (page)；plan T11 Step 2“读数据库只经 `viewRow`” | ① `viewRow`（`E/fixtures/views.ts:158`）只给项目、所有者、创建与修改者、软删与时间戳，没有筛选与显示；“数据库的行与回答相同”照它做不出。② `savedQuery`（`:184`）只给筛选、排序与 `sub_issue`，对分组的视图直接抛；页面的列表请求另带 `limit` 与布局的选项，“列表的查询等于 `savedQuery`”只能是子集比较。③ `h12-project-view-page.spec.ts` 一行装了约 15 组步骤，加上 I3 的四种布局与翻页，过 400 行是大概率的；plan 只给 H20 留了拆分的出口。 | ① 改为经接口读回视图（`getView`）与请求体、回答比较，或给 `viewRow` 加列并把 `E/fixtures/views.ts` 列进 T11 的 Files；② 写成“包含”；③ 给 H12 同样的拆分出口（如另起 `h12-project-view-saving-page.spec.ts`）。 |
| m8 | spec §2.6“布局”“页头”、§2.12 的 T7 一行；plan T7 的 Files | ① 视图页的布局今天没有窄屏页头（`V/(detail)/layout.tsx:17-25` 只有 `header`）；迭代页的是一个自己是 observer 的包装（`P/cycles/(detail)/layout.tsx:25`、`:31-38`，理由在注释里：布局不是 observer）。spec 要窄屏的 `MobileHeaderFilters`，H12、H20 要窄屏的布局菜单，R29 要给它传在途，但 Files 里没有这份新文件。② 页头今天在没有视图时什么都不画（`V/(detail)/[viewId]/header.tsx:45`）；读中、不存在、失败、停用四种状态下页头画什么（面包屑到列表、切换器）没有写。 | T7 的 Files 与 §2.12 加窄屏页头的文件（收入 `project-address-route`）；§2.6 写四种状态下的页头。 |
| m9 | spec §2.7“已确认的线上值”、R8；plan T8 | ① `confirmed(id)` 读的是不可观察的私有 `Map`（`page-preferences-filter.store.ts:127`）；HOC 的 `hasAdditionalChanges` 是 `useMemo`，依赖里若只有它，显示改了不会重算。它与 `filters[id]`（可观察）同时变，依赖要挂在后者上。② `discard` 之后基类的 `values` 仍留着旧草稿（子类删不到）：再进来、读答到之前 `confirmed(viewId)` 给的是旧值。页面这时在“加载”，没有读者，但要写明。 | spec 写明 `confirmed` 不是响应式的、读者怎么跟上（依赖 `issuesFilter.filters[id]`）；T8 加“改了显示之后按钮出现”的格已有，补它的变异（依赖只挂 `confirmed`）。② 一句话记在 §2.6。 |
| m10 | spec §2.3“页头”“布局”、§8 O-3；plan T4 的 Files、“文案” | 列表页筛选面板里的英文没有列全：所有者一组在 `C/components/common/filters/created-by.tsx`（“Created by” `:62`、“You” `:88`、“View less / View all” `:98`、“No matches found” `:103`），日期一组还有 “No matches found”（`created-at.tsx:76`），面板的搜索框占位 “Search”（`VC/filters/filter-selection.tsx:65`）。这三份都只有视图的读者；`created-by.tsx` 不在 T4 的 Files 里。`HO/M3-closeout.md:78` 要关的是“视图页面的英文”。已有的键够用（`common.created_by`、`you`、`common.view_all`、`common.view_less`、`common.search.no_matches_found`、`common.search.label`）。另：`created-by.tsx:39-50` 的选项只随搜索词重算，成员晚到时面板是空的。 | T4 的 Files 加 `created-by.tsx`，“文案”的已有键清单补上；选项的依赖加成员（或写明成员由外壳先读到）。 |
| m11 | plan 性质表 | Step 1 写了格、表里没有行的（照“每个性质至少一个变异”）：① 视图页的 `updateFilters` 不看调用方给的 id（表格与日历传的是项目 id，`base-spreadsheet-root.tsx:95`、`base-calendar-root.tsx:171`）——变异“用给的 id”会把草稿写到项目 id 名下；② T8 的“换会话之后答到不提示”；③ T7 的“删除之后转到列表”“切换器打开时才读列表”；④ T2 的“被拒不改任何值”。 | 各补一行，或写明由哪一行兼管。 |
| m12 | spec、plan 的行号与措辞 | ① spec §2.8 的 `project-level.tsx:35-42` 是 `:36-43`；② spec §2.2 的 `use-navigation-items.ts:75-83`，视图一项在 `:73-82`；③ plan 允许清单的 P8：Enter 在 `:357`、`toHaveURL` 在 `:358`（写的是 `:356`、`:357`），spec 与简报的范围两种写法（`:332-358`、`:332-357`）统一；④ spec §2.11 的 S2 `:102` 的 `if (detail)` 在 `:103`；⑤ spec §2.4“说明若是富文本，编辑器带 `tabIndex={-1}`”：说明是普通的 `TextArea`（`VC/form.tsx:179-192`），合同是纯文本，删去；⑥ spec §2.3 列的四个字面量之外，页面标题的 “- Views”（`V/(list)/page.tsx:46`）也是；⑦ spec §2.5“`isPreferenceQueryCurrent(viewID)` 同项目页的形状”：项目页的按项目 id 找当前列表（`C/store/issue/project/issue.store.ts:11-23`），视图页的要项目与视图两个量，签名写明；⑧ spec §2.6“视图被别人删掉之后，下一次读答 404”：页面挂着时没有“下一次读”，写成“再进来时”，挂着时的“更新视图”得 404 的提示。 | 照改。 |

### 观察

- **O1 R29 之后，“请求体轮到时算”与“点击时算”从界面上分不出。** 在途时控件停用，点击之前的改动都在同一轮事件里落进草稿（草稿的 `update` 不发请求，一个微任务就落定）。`t8-mut-body-at-click` 只能在 HOC 加假队列的那一层杀死（扣住前一次写、改草稿、放开），这是确定的，不算竞态；格要写在那一层，页面故事里看不到。
- **O2 `isUpdating(viewId)` 在编辑弹窗的 `PATCH` 与删除在途时也为真**（同一个视图的队列）。视图页的控件这时也停用；弹窗开着时本来就点不到它们，无害，写一句免得整审时当成缺陷。
- **O3 成功之后多一次列表的读。** 回答整理过筛选时 `resetExpression` 总会通知（`filter.ts:301-309`），视图页的回调把同一份筛选再写一次草稿、再取一次列表；不会成环（写草稿不再通知实例）。读表的断言照此数。
- **O4 没有条件时不给 “Save view”**（`canSaveView` 要 `hasActiveFilters`，`filter.ts:271-273`）：迭代页、模块页不加任何条件时存不出“这个迭代的工作项”的视图，要先加一个条件。与 R7 的文字一致，Plane 相同；H12 (page) 的迭代、模块两步要先加条件。
- **O5 “Add view” 的默认常量是合同默认值的第三份**（合同的 `default`、服务端的 `DefaultListDisplayFilters`、`viewDefaults`）。不带这两个字段时服务端填的就是它（`navigation/domain/content.go` 的 `CheckNewView`）。O-1 已裁定带上，H12 钉住；只提醒：合同的默认值一变，要改三处，故事会先红。
- **O6 web 作业的报告线。** P5b 的最终分支 CI 已是 1075 秒；本段净增单测约 1,500–2,000 行，加上 I1–I4 要补的格。照 R22 只报告、三步对比判别。
- **O7 T8 是本段最大的 Task**：HOC、筛选行、页头五个组件、`DraftBusyContext` 与它的读者（照 I4 是三个）、视图 store 的 `isUpdating`、三张表的文案、二十多个变异，分两个提交。按第一个读者的规矩它拆不成两个 Task；plan 写明两个提交的分界（哪些文件、哪些变异属于哪一个），整审的范围也照此分。

## 3. 逐项回答

### 3.1 行号与事实

spec、plan 与简报引的代码、合同、交接与文档的行号核了约 170 处，Task 依赖的都核了，全部存在、说的是所称的内容；偏差是 m12 的前四处与 m3 ⑤，各差一两行。plan 写的约 95 个文件的行数（`wc -l`）逐个量过，都对。四处“事实”与所称不符：“单测按名字找的不受影响”（I6 ②）、“只剩 store 的定义”（m1）、“四个 `project-root.tsx` 按 `IssuesStoreContext` 取类型，不改”（I3 (c)：它们传的组件把类型写死）、“至此任何改草稿的入口都用不了”（I4）。

### 3.2 合同

- **新建、另存、更新的体**：`ViewCreate` 只要 `name`，其余缺省时由服务端填默认（`NAV:635-662`，`content.go` 的 `CheckNewView`）；`ViewUpdate` 每个字段整体替换（`NAV:663` 起）；`ViewFilters` 有 `cycle_id`、`module_id`（`NAV:566-634`），R7 的另存带词放得进。显示的 7 个键都必填（`api/common.yaml:167-185`），属性 14 个（`:186-221`），另存取“已确认的线上值”整份带上是对的。
- **默认常量**：列表、不分组、`-created_at`、显示子工作项与空组、日历按月不含周末、14 个属性全开，与 `server/internal/shared/list_settings.go:87-95`、`E/fixtures/views.ts:52-80` 相同。
- **名字**：服务端去空白之后 1–255 个码点（`content.go` 的 `cleanName`），spec 的规则同。
- **码**：`navigation.view_not_found`（404）、`navigation.views_disabled`（409）在 `C/lib/error-messages.ts:105-106`；`PATCH` 的检查次序是不存在 → 无权 → 校验 → 开关，“关掉开关之后更新视图得 409”成立（所有者、存在、体合法）；删除在开关关着时也是 409，列表页的删除一步若在关开关之后做，要把它写进期望。
- **矩阵**：`view.read` 三种角色（`rules.go:223`），建、改、删要管理员或成员（`:228-230`），改只给所有者（`view.go:58`），删给所有者或项目管理员（`:65-73`），只看自己的访客看不到别人的（`:52-54`）。每一格都对；“被降为访客的所有者不给编辑与更新”与服务端一致。工作区管理员即项目管理员（`permissions.store.ts:68-75`）。

### 3.3 十条高风险裁定

- **R1（引擎）**：方向成立。`PagePreferencesIssues.fetchPage(slug, projectID, preferencesID, options)` 与 `PagePreferencesFilter` 的端口（`get`、`update`）正是“项目的列表、查询来自这一页的偏好”，基类不读路由，本地存储按类型分键。问题在缓解的范围与漏列的分派点（I3）。`use-issues.ts` 的 `switch` 在泛型参数上，`satisfies never` 能否收窄留给 `t6-exhaustive-probe`。`root.store.ts` 只加构造、不超过 340 够用（`:202-212` 旁三四行）。
- **R7（另存带词）**：`PLANNING_PAGE_DOMAINS`（`planning-page-domains.ts:43-70`）加一列放得下，共用的根不必分支；迭代页、模块页的筛选类型本来就排除自己的词（`C/lib/issue-filters.ts:198-209`），不会重复。翻过来的 P4 测试连同 M6 的两份见 I6 ②；没有条件时不给按钮见 O4。
- **R8（`confirmed`）**：基类没有乐观显示（`replace` 在服务答到之后才写 `values` 与 `filters`，`page-preferences-filter.store.ts:206-227`），所以“已确认的线上值”就是页面显示的那一份的线上词，三个页面的行为不变。只读、返回拷贝是对的；不可观察见 m9。
- **R9（等回答）**：次序对得上 shared-state（`filter.ts:499-506`：回调 resolve 之后才重置基线，reject 则不动）与 `followInSession`（从不 reject，`in-session.ts:40`）；今天的 HOC 不返回 Promise、失败也重置（`project-level.tsx:134-166`）。缺的是离开页面之后的跟进（I2）、成功之后的焦点（m6）、与 T7 的接缝（I5 ②）。
- **R10（不推进基线）**：成立，而且不需要别的改动：已有的筛选实例再登记时只换选项与回调、不换表达式（`work-item-filters/filter.store.ts:81-105`）；HOC 卸下时删掉自己的实例（`filters-hoc/base.tsx:104-109`），再进来的实例从新播种的草稿起，不会留着上一次没存的条件。T7 的格钉什么见 I5 ②。
- **R13（命令面板）**：菜单与搜索照 M6/P7c 的形状做得出；既有的 `search-menu.test.tsx` 不用动（搜索菜单经 `useIssueDetail().rootIssueStore.rootStore` 取根，`search-menu.tsx:24`；多出的那个视图的读没有人答、留在途，不碰任何断言）。要改的是 m4（性质的写法、读的时机、T9 之后的回归）与 m2（漏列的读者）。
- **R14（`viewsEntryOn`）**：与 `intakeEntryOn`（`C/lib/intake-entry.ts:18-24`）同形。侧栏与页签栏今天就是“开关加三种角色”（`project-navigation.tsx:103-112`、`use-navigation-items.ts:73-82`），两条命令的表两行 `[F, T, T, F, T]`（`commands.test.tsx:120`、`:138`）照新规则仍为真，`create_view` 今天是“开关加管理员或成员”（`creation/command.ts:110-114`），与 `viewsCreatableBy` 相同。没有发现漏的读者。
- **R16（`projectMemberIds`）**：产品读者确是两个（`V/(list)/mobile-header.tsx:23`、`:56`，`VC/view-list-header.tsx:32`、`:121`），store 的三处、测试的两处与收集箱替身的一个字段同 spec；显式的 `getProjectMemberIds(projectId, true)` 已在用。收据的措辞见 m1。
- **R21（文案）**：五个字面量与三条提示的行号都对；`common.filters`、`common.clear_all` 已有，en 的文字不变，页面故事不受影响。单测受影响的三处见 I6 ②。
- **R29（在途停用）**：来源只有一个（`isUpdating`）是对的；`FiltersDropdown`、`CustomMenu`、空态的动作都已有 `disabled`，`disabledAllOperations` 确实会把 “Confirming” 一起藏掉（`filters-row.tsx:79`）。漏了日历的选项（I4）。窄屏的页头没有筛选开关（`mobile-header-filters.tsx`），“五个控件”在窄屏是两个；视图页的窄屏页头本身还没有文件（m8）。另见 O1、O2。

### 3.4 可测性

- **性质与变异**：表里的行都能写出确定地失败的格，没有只能靠竞态杀死的行；要改写的是 `t9-mut-menu-other-key`（m4 ①，照字面要靠时间）与 `t6-mut-get-reads-every-time`（I1，触发者没了）。缺行：每次挂载读自己的（I1）、迟到的播种与离开之后的跟进（I2）、每个分派点（I3）、日历的选项（I4）、m9、m11。
- **页面故事**：plan 的写法（扣住回答、路由只在 Chromium 且在目标页代码载入之后装、动作之前等在途的读、`expectQuietConsole` 不放宽）都在。读表：视图页是挂载的那几个读、一个 `GET /api/v0/views/{id}`、一个列表的读；成功的“更新视图”之后可能多一个列表的读（O3）。一次 `fetchIssue` 两个 `GET` 的教训只在故事打开浮层时用得上，计划里的三份都不打开。400 行见 m7 ③。
- **新文件名**：三份都合 m7-pages 的 `testMatch`（`h\d+(?:-h\d+)?-[\w-]+-pages?\.spec\.ts`）与 `grep`（`H\d+(?:–H\d+)? \(page\)`）；m7 ③ 与 plan 给的拆分名也合。

### 3.5 plan

- **次序**：数据层 → 入口 → 列表页 → 弹窗 → 引擎 → 视图页 → 另存与更新 → 命令面板 → 删除，方向对；两处接缝要改（I5），S2 要前移（I6 ①）。
- **检查点**：0 在 T5 之后、兼作 m4 三个配置的基线，1 在 T7 之后，2 在 T10 之后；照 I5 ②、I6 ① 改完，检查点 1 才是“引擎与视图页完成”的干净状态。
- **六个配置**：spec §2.11、plan 的“检查点”、简报 `:158-159` 一致（m7-pages、m6-pages、m6-acceptance、m4-list、m4-filters、m4-detail）。`L/final.sh` 今天的循环是 `m7-pages m6-pages m6-acceptance m4-detail m5-content m4-list`，文件头是 P5b 的说明；plan 的 T1 写了把循环改成这六个、文件头照改——对。`i18-search` 与命令面板的多数故事不在六个里（m4 ③）。
- **不许变长的文件**：十项的行数都对；`use-issues-actions.tsx` 的约束写不成“一个 `case` 与一个导入”（I3 (d)）。
- **knip 的第一个读者**：成立（`use-views.ts` 在 T4、`isUpdating` 与 `DraftBusyContext` 在 T8、`confirmed` 的读者是 HOC）。I5 ② 改完之后 `enableSaveView`、`enableUpdateView` 的传入者也正好在 T8。
- **允许清单**：T2、T3、T6、T9、T10 的是全的（`use-issue-layout-store.test.tsx`、`use-issues.test.ts`、`search.test.tsx`、`root.store.test.ts:106` 核过）。不全的：T4（I5 ①）、T7（S2，I6 ①）、T8（I6 ②）、T10 或 T9（m2）。
- **死成员**：T1 的基线与 T4、T6、T8、T10、T13 的比对点够；P5b 终表里视图的三行有去处。

### 3.6 交接

spec §5 的表逐行核过，行号都是今天的（M3 `:31`、`:35`、`:38`、`:43-44`、`:50`、`:52-53`、`:56`、`:60`、`:73`、`:76`、`:78`、§6 `:101-109`、§7 `:111-125`；M4 `:51-63`、`:65-89`、`:91`；M6 `:34`、`:48`、`:65`、`:83`；M2 `:39-50`；P4 spec `:379-381`）。

- `HO/M3-closeout.md:43-44`（弹窗）、`:53`（右键）：视图弹窗照 M6 的规则、右键在 H12 (page) 以右键打开——覆盖了。
- M3 §5：视图弹窗没有编辑器，只有 busy；“写处理结果、顶层保持 open”是对的。
- M4 M7-04：视图的结果（R13）与“视图写后关闭”（R9）关闭；逐项的写法见 m3 ④。
- M6-M7-05（`getTabIndex`）、M3 §7（`projectMemberIds`）：删除的条件满足，读者核过（m1 的措辞）。
- M6-M7-07：视图一半随 R14 关闭——对。
- M2 §3：`single-select.tsx` 已删、`LayoutDropDown` 随 R28 删，“不适用”加视图页布局切换的键盘证据——说得通；H20 的那一步窄屏一半要先有窄屏页头（m8）。
- P4 R17：项目一侧关闭——对；T7 不要先传这两个开关（I5 ②）。
- 要改措辞的四处见 m3 ①②③⑤。

### 3.7 与 Plane 的差异

spec §7 的八条里，Plane 的引用核了：表单的布局与显示（`PW/core/components/views/form.tsx:219`、`:233-257`）、筛选一行（`:264-289`）、另存的默认名（`filters-hoc/project-level.tsx:110`）、迭代页不带词（`roots/cycle-layout-root.tsx:89-97`）、外壳读视图（`layouts/auth-layout/project-wrapper.tsx:134`）、服务端搜索（`plane/apps/api/plane/app/views/search/base.py:210-232`）、空态给访客（`views-list.tsx:35-37`）、文档级 Escape（`modal.tsx:86-87`）；R2 说的“Plane 相同”也对（视图页的根读一次、卸下时重置，`roots/project-view-layout-root.tsx:67-83`）。都对。漏登记一条、一条引用要换（m5）。

### 3.8 P5b 的教训

- **用上了的**：WebKit 在路由下重取预载；一次 `fetchIssue` 两个 `GET`；动作之前等在途的读（P8 的一步也照此）；`expectQuietConsole` 不放宽；重连只通知一个键上的第一个 hook；变异包的源码之后核 dist；`AlertModalCore` 的按钮文字显式传；弹窗与宿主的跟进只在挂着时做。
- **没有用上的**：每次挂载读自己的那一类读（P5b R27 → I1）；跟进的挂载范围在 HOC 这一处（P5b R29 → I2 ②）；控件消失之后的焦点落点（P5b R26 → m6）；允许清单要对着替身下的字面量核一遍（P5b T4 的偏差 → I6 ②）；“必然发生的事写成计划内的 Task”（re-95 I5 → I6 ① 的 S2）。

## 4. 核实为正确的事实（摘要）

- 枚举今天没有 `PROJECT_VIEW`（`PKG/types/src/issues/issue.ts:14-22`）；视图的路由判为 `DEFAULT`（`use-issue-layout-store.ts:27`）；两个 `switch`（`use-issues.ts:66-103`、`use-issues-actions.tsx:74-90`）的 `default` 抛 “Unsupported issue store”；`use-planning-page-issue-actions.ts:148-156` 的 `updateFilters` 不看给的 id。
- `PagePreferencesFilter`：端口 `:33-36`、`values` `:127`、`fetchFilters` 的读序号 `:175-190`、整字段替换 `:206-227`、显示的合并 `:252-270`；`use-page-filter-expression.ts:38-58` 会推进基线、`:47-52` 等价时不动；视图页的联合不加是对的。
- 富筛选：`hasChanges` `:208`、`canSaveView` `:271-273`、`canUpdateView` `:279-285`、`resetExpression` `:301-309`、`updateView` `:499-506`；筛选行的在途状态 `:33`、`:48-55`、五个字面量 `:45`、`:83`、`:88`、`:99`、右侧在 `disabledAllOperations` 时不画 `:79`。
- HOC：弹窗状态 `:41-42`、旧 store `:45`、锁 `:62`、`canUpdateView` `:77-91`、拿控件词比 `:94-102`、默认名 `:106`、三条英文提示 `:139`、`:152`、`:160`、弹窗 `:189-198`。
- 页头与列表：`HeaderFilters` 的四处（`C/components/issues/filters.tsx:52`、`:55`、`:57`、`:58`）、`LayoutSelection` 的按钮 `:40-50`、`FiltersDropdown` 的 `disabled`（`helpers/dropdown.tsx:22`）、`CustomMenu` 的 `disabled`（`PKG/ui/src/dropdowns/custom-menu.tsx:49`）、列头的菜单（`header-column.tsx:42`）、表格的排序写者（`base-spreadsheet-root.tsx:93-99`）、空态写死的 `PROJECT`（`project-issues.tsx:28`）与清除（`:51-53`）。
- 视图的旧层：`VC/form.tsx` 的 `getTabIndex`（`:26`、`:75`）、`LayoutDropDown`（`:32`、`:204`，唯一的产品读者）、初值 `:41-47`、普通 `h3` `:96`、名字的规则 `:141-150`、`span` 的提示 `:170`、布局与显示 `:197-246`；`VC/modal.tsx` 的文档级 Escape `:80-82`；`VC/quick-actions.tsx` 的复制 `:49-57`、嵌着的 `IconButton` `:93-95`；`useViewMenuItems`（`quick-actions-helper.tsx:47-58`、`:130-145`）与 P6b 的读者（`workspace/views/quick-action.tsx:52`）；列表页的停用页 `:75-91`、“Add view” 不看角色（`V/(list)/header.tsx:54-56`）；视图页 `page.tsx:32`、`:36-45`、`:50`，页头 `:41`、`:47`、`:75`、`:93-101`。
- 桶导出与包：`PKG/utils/src/index.ts:19`、`:23`，`PKG/constants/src/index.ts:29`、`:32`，`PKG/types/src/index.ts:25`；`tab-indices.ts` 两份的行号；`PKG/types/src/views.ts`、`PKG/constants/src/views.ts`、`helpers/views.helper.ts` 的读者同 spec §2.9（`EViewAccess` 有 P6b 的读者）。
- 收藏：`favorite.store.ts:55`、`:73`、`:78`、`:257-266`（只有视图一支）与四个调用 `:292`、`:297`、`:347`、`:355`；`use-favorite-item-details.tsx:16`、`:26`。
- 会话：`followInSession`（`in-session.ts:40`）、`oneAtATime`（`one-at-a-time.ts:14`）、`Reconciled` 的 `prepended`、`replaced`、`dropped`（`reconciled.ts:201`、`:206`、`:218`）、`RootStore` 的构造次序（`projectView` 在 `issue` 之前，`root.store.ts:96-98`）、计数 20（`root.store.test.ts:106`）；外壳不读视图（`use-project-fetch.ts:64-71`）。
- 命令面板：`use-planning-search.ts:45-46` 的门、`search-menu.tsx:43-48`、`search-results.tsx:72-96`、`project-views-menu.tsx:25` 的旧 store、`menus/views.tsx:11`；`ProjectLevelModals` 卸下时关（`project-level.tsx:36-43`）。
- 门禁：web 的上限 18；swr 名单 `.oxlintrc.json:121-215`；`K:2` 的 phase 是 `M7/P5`；`project-address-route` `K:2334`、样例 `:2631-2637`、`:2797-2803`；`copy-hook` 的视图例外 `:3350-3356`；`old-module-urls` `:3078`、`old-intake-urls` `:3185`。
- e2e：`createProjectView` `:17`、`viewDefaults` `:52`、`viewRow` `:158`、`savedQuery` `:184`；`holdAnswer` `:74`、`expectPopupBesideButton` `:240`、`moveWithinApp` `:296`；`dismissToasts` `:275`、`failedLoads` `:297`、`tabStops` `:315`、`recordApi` `:342`；`refuseClipboardWrites` `:238`、`expectQuietConsole` `:254`；`modalDialog` `:113`；`mounts.ts` 的头注释 `:1-15`；P8 在视图列表一步之后不再看 alice 的页面（她的断言在 `:328-330`）。
- 文案：要用的已有键都在（`views`、`common.filters`、`common.clear_all`、`common.item_options`、`view.create.label`、`view.update.label`、`project_views.delete_view.*`、`project_view.sort_by.*`、`disabled_project.empty_state.view.*`、`common.filter_options.custom`、`.unavailable`、`inbox_issue.filters.created_at`）；`project_views.add_view` 是新的（已有的 “Add view” 在工作区一侧，`workspace.json:136`）。
- T1 的回填：`L/progress.md` 第二行的数字与 plan T1 写的一致（终检第 1 路 657.6 秒、分支 CI 45369650152 的 1075 秒与 82.6%、noff `47bd860b`、main CI 45375109166 的 60.8%、归档 10965 个文件）。

## 5. 需要负责人的事

无新的。六条重要发现都是网页一侧的写法与计划，属主控的局部裁定；没有一条要改合同、服务端或 CI 的预算与结构。照已有的约定，R1、R7、R8、R9、R10、R13、R14、R16、R21、R29 十条是动了别的 M 或 M7 前段网页共用代码的高风险局部裁定，阶段记录与 M7 收官报告里向负责人列出（spec §5 的那一句现在少 R29，m3 ⑤）；I3、I4 改完之后 R1、R29 各多动几份 M4/M6 的共用文件（列表挂载的键、行的快捷操作、“添加已有”的请求类型、日历的选项下拉），列出时写上。O6 的 web 作业报告线照 R22 只报告；要动期限或作业结构是负责人的事，本段不需要。

## 6. 处理（主控）

（留给主控。）

## 6. 处理（主控，2026-11-24）

结论照收：修正后通过。没有需要负责人的事。重要六条与次要十二条都照“须做的修正”改进 spec 与 plan（交起草者），观察七条按下面处理。各条的选择：

- **I1**：照改。视图页的读放进自己的文件（收入 swr 名单），关挂载与重连的再验证，每次挂载 `mutate()` 一次（P5a 整审 I1、P5b R27 的写法）；T7 的格与 `t7-mut-plain-mount-read`；H12 断言再进来的第二个 `GET`。
- **I2**：照改。草稿按视图带“页面还在不在”的代：`discard` 作废在途的播种，迟到的回答只进 `ViewStore` 的详情、不建草稿、不写 `filters`；`adopt` 只在草稿还在时替换；HOC 的两个跟进只在它还挂着时做。机制取草稿服务自己的代（不改基类成员的可见性，基类不碰）。T6、T8 的格与两个变异，H20 两步之后各回到原视图断言又一个 `GET`。
- **I3**：照改，四处都补并各写做法：(a) 列表挂载的键带 `viewId`；(b) 日历的“添加已有”同项目页，请求类型收 `PROJECT_VIEW`，命令清单写全；(c) `ProjectIssueQuickActions` 读 `IssuesStoreContext`、只在 `PROJECT_VIEW` 时换，四个根照旧不改；(d) `use-issues-actions.tsx` 的行数约束照实写。T6 按分派点各一格一变异；H12 的四种布局与翻页另起一份文件（`h12-project-view-layouts-page.spec.ts` 一类），每份不过 400 行。
- **I4**：照改，日历的选项下拉作 `DraftBusyContext` 的第三个读者；R29 的“任何入口”以 `git grep` 的三个写者为准写进 spec；单测与变异各一，页面不另加（单测承担，写明）。
- **I5**：① 取 (a)：T4 的宿主只持有删除，行的“编辑”、列表页的 “Add view” 与空态的“新建”在 T5 随弹窗接上，T4 的矩阵格只测给不给；② 照改：T7 的根不传 `enableSaveView`、`enableUpdateView`（T8 随第一个读者传），R10 的格与变异在 T7 钉 `hasChanges` 与“没有 `PATCH`”，“`canUpdateView` 为真”移到 T8。
- **I6**：照改。S2 的改动移到 T7 的同一提交；T8 的允许清单加两份 M6 根测试与 `save-view-offered.test.tsx` 按字面找 “Clear all” 等的地方改为键（断言的意思不变），spec 的那句照改。
- **m1–m12**：都照“须做的修正”改。其中 m4 ③ 加 T9 之后的 chromium 单跑 `t9-palette`（`i18-search`、`g19-work-item-side-page`、`h1-subscription-page`）；m6 的落点定为筛选行的“添加筛选”按钮（不在则筛选开关），spec 写机制、H20 加键盘一步三引擎记焦点；m7 取“经接口读回视图比较”与“包含”，H12 给拆分出口；m8 加视图页的窄屏页头文件与四种状态下的页头；m10 的 `created-by.tsx` 进 T4 并补成员的依赖；m12 ⑤ 删去“说明若是富文本”一句。
- **O1**：照写，`t8-mut-body-at-click` 的格写在 HOC 加假队列的那一层。**O2**：照写一句。**O3**：读表照此数。**O4**：H12 的迭代、模块两步先加条件。**O5**：记下（合同默认值三份）。**O6**：照 R22 只报告。**O7**：plan 写明 T8 两个提交的分界（文件与变异各属哪一个）。
