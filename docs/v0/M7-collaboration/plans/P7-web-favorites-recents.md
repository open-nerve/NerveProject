# M7 / P7 plan：网页的收藏与最近访问——侧栏收藏、星与命令、连带、记录与首页的“最近”

> spec：[P7 spec](../specs/P7-web-favorites-recents.md)（§2 是合同；§3 的 R1–R29 是裁定，R-n 即台账 `rulings.md` 的 E-n；**L!** 是 R2、R7、R9、R10、R11、R12、R13、R14、R16、R17；没有负责人的裁定；§3 末是向负责人说明的四件、起草时核出的十三条补记与预检 re-98 的处理；§8 已无待定：`writesSeen` 由 R26 定为 (a)）。预检 [re-98](../../../reviews/re-98-M7P7预检.md)（修正后通过）的四条重要、十二条次要与采纳的观察都已折进本 plan（主控的处理，re-98 §6）。基线 main `c9f26bc2`（`c9f26bc2c4595e1a7b28ce0346e04a5e07197bed`，P6b 的 noff）。台账 `L` = `.superpowers/sdd/P7-web-favorites-recents/`。缩写同 spec。

**与调研草案（`research/scope.md` §8）的对照**：照 R24 的 15 个 Task，“收藏”筛选并进 T6、T7。微调四处，都因依赖：① T3 删旧 store，旧侧栏的拖放接线读它的三个移动方法，所以 T3 同一提交删去旧的拖放接线，T4 照 spec §2.4 重接（T3–T4 的窗口，见下）；② 四个 store 的 `answeredAtOf` 在 T3（第一个读者是侧栏行的名字），不在 T2；③ `useFavoritesOffered` 在 T5（第一个读者是星），不在 T3；④ 最近访问的列表的读（service 的 `list`、store 的 `lists`/`fetchRecentVisits`/`getRecentVisits`）在 T9 与首页一起建，T8 只建记录的一半（免得 T8→T9 的“成员先于读者”）；⑤ `SB/favorites.helpers.ts` 与拖动的手柄在 T3 删、T4 新建（它们引 `IFavorite`、只被 T3 删去的拖放接线读，re-98 I4）；连带与改名的口 `TargetFavorites` 在 T6 建（第一个读者是迭代与模块的 changes），收藏 store 的 `renameTarget` 在 T2 建（窗口里，读者在 T6、T7，re-98 I1）。检查点 0 在 T2 之后（网页看不出变化，跨模块改动之前的基线），1 在 T8 之后（请求面最大的变化：记录），2 在 T9 之后（产品代码完成）；故事在检查点 2 之后，终检全跑。

## Global Constraints

**分支与提交**
- 主目录永远在 main。提交用详细中文，前缀 `feat/fix/docs/test/refactor(M7/P7)`，结尾一行 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。提交只在本 Task 的 lint 过了之后，不与它串在一条命令里，用显式路径（`git commit -F <msg> -- <paths>`）。
- 实现代理不推送、不合并；检查点由主控推送。
- 关键词的 phase 在 T1 由 `M7/P6` 改为 `M7/P7`（`K:2`，主控）；不延长任何 `until`。
- 不改 `api/`、`server/`、`deploy/`、`.github/`；设计由主控改（T1、T14）。超出 spec 的契约或架构改动先停下报告。
- 新写的代码注释写 “M7/P7 spec”，不写裸的 “P7 spec”（已有的 “P7a”–“P7c spec” 指 M6），不改旧注释。

**只读与禁区**：`plane/`、`refer/` 只读，不在其内跑 git。不 stash、`git clean`、`reset --hard`、强推；还原变异只经 `mutate.py`。不动别的项目的容器。容器套件一次一个：`make test`、`make test-shard`、`make e2e` 与 Playwright 的配置不并发。

**代码规则**（总体设计 §7.2、§7.7）
- 手写文件约 400 行以内，新代码进新文件，按关注点分（spec §2.14 的表）。新文件的版权头 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（`Favorite`、`FavoriteKind`、`FavoriteCreate`、`FavoriteUpdate`、`RecentVisit`、`RecentVisitCreate`、`RecentVisitKind`）；service 不转换；store 只经自己的根找兄弟；SWR 只经 `useSessionSWR`、只用它的选项（`C/lib/use-session-swr.ts:38-46`）；新代码不读 router store 的地址（不用 `currentWorkspace`、不带 slug 的工作区级 `allowPermissions`）。
- 收藏的写只经收藏 store 的队列，体在轮到时算；`await` 之后的提示、关输入、打开收藏区一律经 `followInSession`；拒绝经 `useRefusalToast`；不 `console.*`。最近访问的记录失败静默（spec §2.8），也不 `console.*`。
- 删掉不用的代码，不加开关、桩、只给测试用的钩子或兼容别名。
- 名字里不得有不分大小写的 “sentry”（`K:1224` 的 `unsplash-telemetry`）：`favoritesOn`、`favorites-gate.ts` 可以；`favoritesEntry…`、`useFavoritesEntry` 一类不行（“favorite**s Entry**”拼起来含 “sentry”）。测试标题与文件名同理。
- 不放宽 `K` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`、`expectQuietConsole`。
- **行数**（`wc -l` 于基线）：测试与故事每份不过 400 行（已超的 `s2-web-app.spec.ts` 581、`collaboration-interactions.spec.ts` 798、`module.store.changes.test.ts` 433 不变长，新格进新文件；`h9-intake-list-page.spec.ts` 398 是写明的例外，上限 410（re-98 m7：T3 的 `FAVORITES` 让 `:25` 的导入到 114 字、T8 的 `VISIT` 再到 121，过 `.oxfmtrc.json` 的 120 而折成约十行；`...VISIT` 另起一行放进 `:85` 的外层数组——它没有占位，不必过 `valued`——`:86` 只加 `...FAVORITES` 到 111 字；T3、T8 的收据各量一次 `wc -l`）；`cycle.store.changes.test.ts` 397 不加格）；写的一层 `C/store/favorites/favorite-changes.ts` 不过 220；`C/store/project/project.store.ts` 608（已过 400：只加 `answeredAtOf`、把 `getProjectById` 的“最新的一份”提成私有函数、`forget` 与归档的确认里各一行，不过 620）；`C/store/planning/module-changes.ts` 235（只加构造参数与三处调用——连带两处、改名一处——不过 245）、`cycle-changes.ts` 180（同样三处，不过 190）、`C/store/views/view-changes.ts` 144（连带、改名各一处，不过 155）；`cycle.store.ts` 163、`module.store.ts` 170、`view.store.ts` 156（各加 `answeredAtOf` 与给 changes 的口，各不过 +12）；`C/components/project/card.tsx` 345（不过 360）；`SB/favorites-menu.tsx` 294、`SB/favorite-folder.tsx` 299（改写之后各不过 250，拆出去的进新文件）；`PROJ/browse/[workItem]/layout.tsx` 106（不过 112）；`PROJ/projects/(detail)/[projectId]/layout.tsx` 64（不过 68）；`E/fixtures/intake-pages.ts` 344（只改 `writesSeen` 的一行条件与一句注释）；`E/fixtures/mounts.ts` 97（三个常量与注释，不过 130）。

**knip 与第一个读者**（新文件、新导出在它第一个读者出现的 Task 里建立；测试数据的导出以第一个读它的测试为准）：
- T2：`FavoritesService`（读者：收藏 store）；`FavoriteStore`、`IFavoriteStore`（读者：`RootStore`）；`favoriteTree`、`placeAmong`（读者：store、写）。T2 的并存窗口里新旧两个类同名（`C/store/root.store.ts:23-24` 导入旧的 `IFavoriteStore`、`FavoriteStore`）：T2 在 `root.store.ts` 里以别名导入旧的（如 `FavoriteStore as PlaneFavoriteStore`），T3 删旧的时去掉别名（re-98 m9 ②）。
- **T2 的“成员先于读者”窗口**（唯一的例外，R24 把数据层排在最前）：收藏 store 与写的一半的成员在 T2 只有单测读，产品读者在 T3–T7：`fetchFavorites`、`getFavorites`、树、`answeredAt`、`removeFavorite`、`createFolder`、`renameFolder`（T3），`move`（T4），`add`、`remove`、`favoriteOf`、`isPending`、`forgetProject`（T5），`forgetTarget`、`renameTarget`（T6、T7）。T2 之后的死成员比对把它们写成“读者在 T<n>”；T7 之后的比对必须清零（T5、T6 之后逐个划掉）。别的 Task 没有这种例外。
- T3：`useFavorites`、`useFavoritesRead`（读者：`AppSidebar`）；`favoritesOn`（读者：`useFavoritesRead`）；四个 store 的 `answeredAtOf`（读者：`use-favorite-item-details.tsx`）；`favoriteLink`（读者：行）；`FavoriteFolderForm`；`E` 的 `FAVORITES`（读者：`s2-web-app`）。
- T5：`FavoriteStar`、`useFavoriteToggle`、`useFavoritesOffered`（读者：项目卡片）。
- T6：`useFavoriteCommand`（读者：迭代、模块的命令）；`TargetFavorites`（读者：迭代、模块的 changes；T7 的视图 changes 是第二个）；模块筛选 store 的 `getFavoritesOnly`/`setFavoritesOnly`、`isFiltered`（读者：菜单、已应用的筛选、列表、模块列表页与页头）。
- T7：`viewFiltersApplied`（读者：项目视图列表页与已应用的一行）。
- T8：`RecentVisitsService`（只有 `record`）、`RecentVisitStore`（只有 `record`）、`useRecentVisits`、`ProjectVisit`、`useRecordVisit`；`E` 的 `VISIT`。
- T9：service 的 `list`、store 的 `lists`/`fetchRecentVisits`/`getRecentVisits`、首页的小部件与预览宿主；`E` 的 `RECENT_VISITS`。
- 删掉某个导出最后一个读者的 Task 同一提交删去它（spec §2.11）。knip 查不出只被测试读的产品导出与包的桶导出：逐 Task 的审查与死成员的整份比对兜底。

**告警上限**：`lint-cap` 要求 oxlint 的告警数恰好等于上限（web `W/package.json:10` 现为 14；e2e、`@nerve/constants`、`@nerve/types` 为 0）。新文件不得新增告警；删掉或改写掉带告警的写法时，同一提交照实改上限并留收据（`t<N>-oxlint-count`；T3 删旧层时要跑：M3 交接说 `SB/favorite-folder.tsx` 有 2 条，调研实跑为 0，照实写）；T15 只核对。

**文案**：新键只有 spec §2.12 列的：`common.json` 七个（T3：`favorite_item.remove`、`favorite_folder.{rename,remove,name_too_long}`、`favorite_list.load_failed`；T5：`favorite_star.{add,remove}`）、`power-k.json` 四个（T6）、`home.json` 八个（T9）；en 与 zh-CN 同一提交；写完整的字面量（`tools/i18n-keys.mjs` 才认得），不许一个键是另一个键的前缀。删的键全在 T3（re-98 m8），共八个：`common.json` 的 `folder_already_exists`、`no_favorites_yet`、`failed_to_move_favorite`、`failed_to_reorder_favorite`（后两个的读者随拖放接线走掉），`accessibility.json` 的 `aria_labels.projects_sidebar.{open,close}_favorites_menu`、`{open,close}_folder`；每个先核读者（`tools/i18n-keys.mjs` 不报没人读的键）。`favorite_updated_successfully` 留着（改名成功的提示）。T4 不删键。别的一律用 spec §2.12 列的已有的键。

**守卫**
- `workspace-address-route`（`K:2847`）：每个新建或改写的、读工作区的文件收入文件名单并带 hit 样例；`project-address-route`（`K:2334`）同理收读项目的；改名或删去的文件同步改两条规则的样例；每次改 `K` 跑 `node tools/keywords.mjs`。
- `old-favorite-urls`（spec §2.12）在 T3 与旧 service 同一提交加。
- `.oxlintrc.json` 的 swr 名单（`:121-223` 那一组）：T3 收 `C/hooks/use-favorites-read.ts`，T9 收 `C/components/home/widgets/recents/use-recents-read.ts`。

**允许改的已有测试与夹具**（改之前在台账逐条写明，只照这里写的改法；其余断言是回归，不改；编译跟随与“替身补一行”只为能跑、不改断言，逐处记台账；任何一条要改断言而这里没写，先停下报告）：
- **渲染宿主的测试的一行替身**（T5、T6、T7 通用）：星、“收藏”筛选的选项与命令都经 `useFavoritesOffered` 与收藏 store；以 `vi.mock` 换掉 hook、没有 store 的既有测试渲染到它们会抛。这类测试补**一行**：`vi.mock("@/hooks/use-favorites-offered", () => ({ useFavoritesOffered: () => undefined }))`（星、选项都不出现，断言不改）。已知的：`C/components/project/card.test.tsx`（136，T5）、`C/components/modules/module-filters.test.tsx`（241，T6）；按需的（跑了才知道，失败的才补，逐个记台账）：`C/components/project/form.test.tsx`（288，T5）、`C/components/modules/inline-changes.test.tsx`（243）与 `module-view-header.test.tsx`（201，T6）、模块列表页的 `PROJ/projects/(detail)/[projectId]/modules/(list)/page.test.tsx`（279，T6，I3 之后页面读 `isFiltered`）、P6a/P6b 渲染 `ViewListItem` 或视图列表的测试（`C/components/views/{actions-host,actions-host-workspace,quick-actions,quick-actions-workspace,modal,modal-workspace}.test.tsx`（经 `fake-view-pages.tsx`）、`PROJ/projects/(detail)/[projectId]/views/(list)/{choices,layout}.test.tsx`、`PROJ/workspace-views/(list)/{page,page-rows}.test.tsx`，T7）。翻转的格用星的替身（下面）。
- T2：`C/store/root.store.test.ts:105` 的 `toHaveLength(19)` 改 20（新 `favorites` 与旧 `favorite` 并存）。
- T3：
  - `root.store.test.ts:105` 改回 19（旧 `favorite` 删去）。
  - `SB/favorite-folder.test.tsx`（55，M3 收尾的）：替身的路径 `:10` 改为新 hook、fixture `:13-25` 改为生成的 `Favorite`（`parent_id`、`entity_id` 等）、组件的 props 跟随改写；三条断言（`:46` 没有嵌套按钮、`:47-49` 名字按钮可聚焦、`:50-54` Space 展开）与标题不改。
  - `C/hooks/use-favorite-item-details.test.tsx`（76，P6a 的）：fixture 改为 `Favorite`；`:52-62` 一格的条件改为“视图 store 的那一份比收藏列表答得晚”（断言的名字、图标、链接不改）；`:64-76` 一格不改意思；新格（收藏列表较新、项目、迭代、模块、工作区视图的链接）进新文件 `use-favorite-item-details-newer.test.tsx`。
  - 实现 `IProjectStore`、`ICycleStore`、`IModuleStore`、`IViewStore` 的测试替身：编译跟随（加 `answeredAtOf`），按 `tsc` 的报错逐处记台账。
  - `E/fixtures/mounts.ts`：加 `FAVORITES`，头注释 `:5-19` 照实改（`:16` 一句）。
  - `E/stories/smoke/s2-web-app.spec.ts`（581，不变长）：导入 `:18-29` 加 `FAVORITES`；`profileVisits`（`:130-137`）、`archivedIssuesVisit`（`:138-146`）、`draftsVisit`（`:147-150`）加 `...FAVORITES`（这三个只给管理员、成员）；管理员、成员、项目非成员的 `"/"`（`:162`、`:185`、`:235`）与 `/{slug}/projects`（`:163`、`:186`、`:236`）加 `...FAVORITES`；项目工作项页的三块相同的期望（管理员 `:169-179`、成员 `:192-202`、访客 `:219-229`）收成一个本地帮手 `projectIssuesVisit(...extra)`，管理员、成员的加 `...FAVORITES`，访客不加；项目非成员的 `:239` 加 `...FAVORITES`；表外的 `:511`（管理员打开已归档项目的工作项页）的 `expectExactly` 加 `...FAVORITES`（re-98 I2 ③；不加 `VISIT`）；设置页的行与 `:497`、`:524`、`:538` 不变。帮手让文件变短，三处断言的意思不变。
  - 九份故事的十处，各只改期望的那一行与导入（spec §2.13）：`h12-project-view-page.spec.ts:64`、`h12-project-views-list-page.spec.ts:76`、`h13-workspace-list-page.spec.ts:59`、`h13-workspace-view-page.spec.ts:103`、`h14-workspace-views-list-page.spec.ts:68`、`h9-intake-list-page.spec.ts:86`、`g13-archive-delete-module-page.spec.ts:77`、`:298`、`g3-change-cycle-page.spec.ts:270`、`g8-archive-cycle-page.spec.ts:89` 加 `...FAVORITES`。
- T5：`card.test.tsx` 一行替身（上面）；`C/store/project/fake-projects.ts` 的 `projectTab`（`:257-277`）：以它的 `api` 建一个收藏 store，追加进项目根的 `fakeRoot`（`:266-269`）并在返回值里给出（spec §2.7）；项目 store 的既有测试不改断言；连带的格进新文件 `project.store.favorites.test.ts`。名单之外的红先看 `C/store/user/fake-permissions.ts:35-38` 的 `permissionsOver`（它自己建的项目根没有收藏 store，re-98 o2）。
- T6：
  - `C/store/fake-project-lists.ts` 的 `listTab`（`:78-85`）：把 `projectTab`（`:79`）给出的**同一个**收藏 store 追加进自己的根（`:83`），并在返回值里给出；迭代、模块的既有 store 测试不改断言；连带与改名的格进新文件（`cycle.store.favorites.test.ts`、`module.store.favorites.test.ts`）。
  - I3 的页面格进新文件（`PROJ/projects/(detail)/[projectId]/modules/(list)/page-favorites.test.tsx`）；`modules/(list)/page.test.tsx` 只按需补一行替身。
  - `C/components/cycles/cycles-view.test.tsx`（281）：加星的替身 `vi.mock("@/components/favorites/favorite-star", () => ({ FavoriteStar: ({ entityId }: { entityId: string }) => <button aria-label={`star of ${entityId}`} /> }))`（写法照它对快捷操作的替身 `:49`）；`:220-230` 一格翻转：标题改为 “…by their avatars, and its star”，`:228-229` 的期望改为 `["project_cycles.more_details", "star of c-WEB-Alpha", "options of c-WEB-Alpha"]`。
  - `C/components/modules/modules-list-view.test.tsx`（294）：同样的替身；`:206-213` 翻转：标题改为 “…its status, its star and its quick actions”，期望 `["project_module.more_details", "days", "project_modules.status.in_progress", "star of m-WEB-Auth", "options of m-WEB-Auth"]`；`:254-259`（卡片）期望 `["project_modules.status.in_progress", "project_module.more_details", "days", "star of m-WEB-Auth", "options of m-WEB-Auth"]`；头注释 `:22-24` 的 “no star to favorite it” 照实改。
  - 命令面板（`use-favorite-command.ts` 的门由它自己的新测试 `use-favorite-command.test.tsx` 以真 store 与真权限钉住）：`…/cycle/commands.test.tsx`（90）与 `…/module/commands.test.tsx`（196）以替身换掉 `../use-favorite-command`（给出一条 id 为 `toggle_<kind>_favorite`、`isVisible`/`isEnabled` 恒真的命令）；`cycle/commands.test.tsx:68-72` 翻转：标题改为 “gives a cycle's page its favorite's command first and its link's copy”，期望 `[["toggle_cycle_favorite", true, true], ["copy_cycle_url", true, true]]`；`module/commands.test.tsx:117-124` 翻转：标题去掉 “: no favorite”，期望最前加 `["toggle_module_favorite", true, true]`；`:126-138` 的 `it.each`：四行的期望最前加同一行，标题改为 “offers no change of the module's members or status to %s: its link's copy, and the favorite's command (its own gate, use-favorite-command.test.tsx)”。
  - `module-filters.test.tsx` 一行替身（上面）；`:139`、`:192` 的 `titles()` 不变（“收藏”是单独的选项，不是一节，spec §3 补记 12）。`C/store/planning/module_filter.store.test.ts`（124）不改，新格进新文件。
- T7：
  - `C/store/views/fake-views.ts` 的 `viewsTab`（`:56-65`）与 `workspaceViewsTab`（`:82-91`）的根、`C/store/views/view.store.workspace.test.ts:66` 的 `fakeRoot`：追加收藏 store；视图 store 的既有测试（`view.store.test.ts`、`view-changes.test.ts`、`view-busy.test.ts`、`view.store.workspace.test.ts`）不改断言；连带与改名的格进新文件 `view.store.favorites.test.ts`。
  - I3 的页面格进新文件（`PROJ/projects/(detail)/[projectId]/views/(list)/page-favorites.test.tsx`，以 `fake-list-page.tsx` 挂页面）；`choices.test.tsx`、`layout.test.tsx` 只按需补一行替身。
  - `C/components/views/view-list-order.test.ts`（73）：`viewsShown` 的两处调用（`:27`、`:71`）加第三个参数 `() => false`（编译跟随，断言不改）；新格进新文件。
  - 渲染视图行或视图列表的既有测试：按需一行替身（上面）。`choices.test.tsx:187-197` 的“没有匹配”计数不变（“收藏”是单独的选项、不随搜索）。
- T8：
  - `root.store.test.ts:105` 改 20（`recentVisits`）。
  - `mounts.ts` 加 `VISIT`、头注释补一句。
  - S2：`projectIssuesVisit` 的管理员、成员、访客三处加 `...VISIT`（项目非成员的 `:239` 不加）。
  - 五处 `VISIT`（期望那一行）：`h12-project-view-page.spec.ts:64`、`h12-project-views-list-page.spec.ts:76`、`h9-intake-list-page.spec.ts:86`（`...VISIT` 另起一行进 `:85` 的外层数组，见“行数”）、`g13-archive-delete-module-page.spec.ts:298`、`g3-change-cycle-page.spec.ts:270`。
  - 八处“之后没有写”（spec §2.13）：`issue/cleanup-interactions.spec.ts:231`、`issue/collaboration-interactions.spec.ts:696`、`issue/detail-interactions.spec.ts:98` 的过滤加 `&& !VISIT.includes(request)`（同一行，注释一句：记录的时刻不随这些步骤，H17 (page) 钉它的次数）；`issue/i11-comments-reactions.spec.ts:312`、`:315` 两处同样；`collaboration/h12-project-view-page.spec.ts:125-127` 的期望改为 `[...VISIT, "POST /api/v0/auth/refresh"].toSorted()`；`collaboration/h20-view-pages.spec.ts:351` 的期望加 `...VISIT`（排序后比）；`collaboration/h12-project-view-saving-page.spec.ts:104` 与 `e2e/acceptance/g19-m3-dialogs-and-filters.spec.ts:146-149` 的过滤同样排除 `VISIT`（re-98 I2 ①②）。
  - 受影响表以整个 `e2e/` 上的 `git grep` 为底（`stories`、`acceptance`、`drills`、`fixtures`，按方法过滤请求、对 `apiRequests` 与 `recordApi` 断言的每一处），不限 `stories`；表上出现本清单之外的一处，先停下报告。
  - `E/fixtures/intake-pages.ts:302` 的 `writesSeen`：条件里排除恰好 `POST /api/v0/me/recent-visits`（R26：方法与路径都对上，一行，注释写 “M7/P7 spec”）；五份收集箱故事不改。
  - 核过不受影响、不改（写进台账）：`h13-workspace-list-page.spec.ts:131`、`:210`，`g15-cycle-list-page.spec.ts:330`、`g15-module-list-page.spec.ts:342`，`h1-subscription-page.spec.ts:52-59`，`h14-workspace-views-list-page.spec.ts:120`，`p2-visibility.spec.ts:129`；单测 `C/components/browse/detail-route.test.tsx`、`intake-redirect.test.tsx`（按路径取调用；`:177` 的全表在被拒的路径上，没有记录）——它们若红，先停下报告。
- T9：`mounts.ts` 加 `RECENT_VISITS`、头注释补一句；S2 四组的 `"/"` 加 `...RECENT_VISITS`。`C/components/home/root.test.tsx`（78）以替身换掉 `HomeBody`（`:27`），不受影响、不改。
- 不改、不受影响（回归）：M4 的预览与详情的单测；P5b 的通知预览；`place-between.ts` 与它的三个读者；`E/fixtures/browser.ts` 的 `watchPage`、`E/fixtures/cycle-pages.ts:342` 的 `recordApi`（R16：不过滤）；`e2e` 的 m4、m5、m6 配置里除上面点名之外的故事。

**红的窗口**：没有红的提交。产品改动让既有的格或故事变的，都在同一提交里照上面的清单改（收藏的读在 T3，记录在 T8，首页的读在 T9）。名单之外的任何红先停下报告。

**T3 与 T4 之间的窗口**：T3 删旧 store，旧侧栏的拖放接线（`SB/favorites-menu.tsx:71-109` 的 `handleDrop` 与两个组件里对旧 store 的移动）随之删去，连同只被它读的 `SB/favorites.helpers.ts` 与手柄 `SB/favorite-items/common/favorite-item-drag-handle.tsx`（T3 的行不画手柄；T4 两个都新建，re-98 I4）；T3 到 T4 之间侧栏的收藏不能拖（基线上收藏区从不出现，没有回归）；T4 照 spec §2.4 重接。

**检查点**（七项用 `sh L/gates.sh <前缀>`，含 `gen-check`、`lint-go`、Go 两路；带页面配置用 `sh L/final.sh <前缀>`：七项之后跑 m7-pages、m6-pages、m6-acceptance、m4-list、m4-filters、m4-detail、m4-collaboration、m5-content 八个配置，每个配置 chromium 与 firefox 一次、WebKit 单独 `--workers=2` 一次，spec R21；主控在 T1 已改好 `final.sh` 的配置与头部注释）：
- **0（T2 之后，跨模块改动之前）**：`final.sh cp0`；网页看不出变化，e2e 应与 main 逐个相同；主控推送、核分支 CI；台账记基线：本机第 1 路 bootstrap（P7 的基线，之后 +30 秒停查）、第 0 路、web 作业的 Lint、Unit tests、Build 三步、e2e 两片，以 `gh` 读 main CI `46146403651` 的同名作业与步作对照；八个页面配置这一次的结果是之后比对的底。
- **1（T8 之后：收藏的星、侧栏、连带与记录完成）**：`final.sh cp1`；主控推送、核分支 CI，记 web 作业三步与 e2e 两片。
- **2（T9 之后：产品代码完成）**：`final.sh cp2`；推送之前照 P4 spec §2.11（`specs/P4-web-notifications.md:270`）估出 e2e 两片（以最近一次分支 CI 的两片为底，加新文件的 chromium 时长 × CI 与本机之比），超过 30 分钟先报告、不推送；主控推送、核分支 CI。
- **终检（T15）**：`final.sh fin`（同八个配置）。
- **另跑的单配置**（动了别的 M 的共用件；先 `make build build-s3-fixture`，再 `pnpm --filter web exec react-router typegen`；每个配置跑完先把 `e2e/test-results` 拷进台账再跑下一个；带 WebKit 的照“本机 WebKit 单独 2 个 worker”）：
  - T3 chromium：`s2-web-app`（`t3-s2`）；九份故事（`t3-mounts`）；有收藏的旧故事 `w2-landing.spec.ts`、`w3-workspace-settings-page.spec.ts`、`p4-archive.spec.ts`（`t3-furnished`：`furnishWorkspace`、`furnishIssueRows` 给管理员建了收藏，`E/fixtures/furnish.ts:116-119`、`:209`，侧栏从此有收藏区）；之后默认配置的整套 e2e 一次（`t3-e2e`：`make e2e`，chromium 全部故事；收藏的读落在管理员、成员每个有侧栏的页上，下一次整套是 T8 之后的检查点 1，re-98 m10）。
  - T5 chromium：项目卡片所在的故事 `p1-create-project-page`、`p2-visibility`、`p4-archive`、`p5-project-members-page`、`f7-project-cover-page`、`s7-chinese-project`（`t5-projects`）。
  - T6 chromium：m6-pages、m6-acceptance（`t6-m6-pages`、`t6-m6-acceptance`；含按中心悬停模块行的 `g19-module-list-page.spec.ts:313`；模块列表页与页头的“有筛选”判断改了，I3）。
  - T7 chromium：m7-pages（`t7-m7-pages`；项目视图列表页的“有筛选”判断改了，I3；含按中心点击视图行的 `h12-project-view-layouts-page.spec.ts:65`、`h12-project-view-page.spec.ts:171`、`h20-view-pages.spec.ts:120`、`:333`、`h20-workspace-view-pages.spec.ts:121`、`:350` 与右键的 `h12-project-views-list-page.spec.ts:329`、`h14-workspace-views-list-page.spec.ts:265`）。
  - T8 chromium：`s2-web-app`（`t8-s2`）；五处 `VISIT` 与八处“之后没有写”在默认配置里的文件（`t8-follow`：H12 两份、`h12-project-view-saving-page`、H9、G13、G3、`cleanup-interactions`、`collaboration-interactions`、`detail-interactions`、`i11-comments-reactions`、`h20-view-pages`）；`e2e/acceptance` 的那一处（`t8-m6-acceptance`：`--config=playwright.m6-acceptance.config.ts --project=chromium`，re-98 I2 ②）；`writesSeen` 的五份收集箱故事（`t8-intake`：`h8-intake-submit-page`、`h11-intake-content-page`、`h11-intake-links-assignees-page`、`h11-intake-attachments-page`、`h11-intake-images-page`）；不受影响的核对（`t8-unaffected`：`h13-workspace-list-page`、`g15-cycle-list-page`、`g15-module-list-page`、`h1-subscription-page`）；m4-detail、m4-collaboration、m5-content（`t8-m4-detail`、`t8-m4-collaboration`、`t8-m5-content`，与检查点 0 的同名配置比对）。
  - T9 chromium：`s2-web-app`、`s7-chinese.spec.ts`、`w8-navigation-preferences.spec.ts`、`w2-landing.spec.ts`（`t9-home`：首页的 `h2` 与侧栏；`w2-landing` 的首页有 `furnishIssueRows` 经接口记过的两行，`E/fixtures/furnish.ts:120-121`，re-98 m10）。
- 第 1 路：本机超过 CP0 基线 +30 秒先停下查；CI 的 bootstrap 过期限的八成只报告。本机 Go 两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 的服务端两路为准并记录（负责人的规则）。
- 本机若被 WindowServer 注销（WebKit 的 corpse），同一 HEAD 以新前缀只补跑没跑到的配置。

**证据**
- 每次检查经 `run-check.py`（`python3 L/run-check.py <label> <command…>`），变异经 `mutate.py`（每个旧串恰出现一次，工具恢复并核 sha256、打印 `restored=yes`）；标签 `t<N>-…`、`cp<k>-…`，不重用。
- **变异包的源码**（`PKG/*`）：在变异的命令里重建那个包的 dist，跑完之后再重建一次并记 dist 的 sha256 前后（P6a T6 的教训）。
- **RED 先于实现**：先写失败的测试，看它因所测性质失败（不是编译失败），留 RED 收据，再实现到绿。
- 下表每个性质至少一个能编译、能让测试失败的变异。两层都在查同一件事时两层一起改，并留只改一层的对照。只有竞态杀得死的变异不算。
- **页面故事先写一个期望错误的版本**（或以 e2e 变异），看到断言在被改处失败，再改正；收据 `t<N>-red-page-…`。
- **死成员**：T1 的基线（`L/dead-members/dead-before.tsv`，先 typegen），T2、T3、T5、T6、T7、T8、T9 之后整份比对（`t<N>-dead-after`），T14 的终表；新出现的每一行写明去处（T2 的窗口见上）；`P6aR:187` 留给 P7 的收藏 7 行在 T3 之后清零。
- **读者核对**：删任何东西之前 `git grep -n` 核读者（含测试与替身、`K` 的样例），写进台账；删之后再核一次只剩预期的。

**测试时间**
- web 作业期限 20 分钟、960 秒报告线；每个检查点记 Lint、Unit tests、Build 三步；越线时三步对 main CI `46146403651` 的同名步各算比值，三个比值相近是慢 runner，照报告、不停；只有 Unit tests 的比值明显高于另两步才是本段的增长，先报告主控再继续。加大量 vitest 的 Task 记新文件的时长。
- e2e 每片 35 分钟、报告线 30 分钟。T10–T13 记每个新文件每个 `test()` 在三种引擎的时长（JSON 报告）；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

**页面故事的教训**（P5b、P6a、P6b 阶段记录 §7、§9，spec R25）
- 每个“在途”都靠扣住回答（`holdAnswer`），不靠时间；不断言墙上的时钟。
- 一律放行或拒绝的路由只在 Chromium 装，而且只在目标页的代码（含编辑器、命令面板的代码块）载入之后装：WebKit 在页面载入或动态 import 时有 `page.route` 生效就不用预载、再取一次。
- 换页、离开、换工作区、改服务端的状态（另一会话经接口删目标、改角色、加收藏）之前，先等在途的读答完：动作之前登记 `waitForResponse`。
- swr 迟到的失败无条件写进键：每次挂载的读在取数函数里兑现、不抛（P6a 整审 m1）；收藏与最近访问的读经 `Reconciled`，被超过的读给 `undefined`、不抛（`C/lib/reconciled.ts:51-78`）。
- 去重窗让读的次数随墙上时钟：收藏的读只在整页载入时数次数，应用内导航之后断言种类（“读过”），不数（P6b 的 I2）；首页的读 `dedupingInterval: 0`，每次挂载一次，可数。
- 删死代码可能暴露靠死代码通过的格（P6b 修复轮 2）：T3 删旧层之后，凡是红了的既有格先查根因，不顺手改定位。
- “不会变红”的前提要算到布局：星让行与卡片的操作区变宽（P6b T5 的 I19 教训），按中心点击这些行的故事已在上面的单配置里点名；窄屏（700/800 宽）的几步都在页头。
- 归档里从没败过的故事变红，先查根因，不重跑了事；本机代理的 TUN 让 C8/C12 失败属环境（P6b 的记录），换网络或以分支 CI 为准并记录。
- 只有图标的按钮与只靠占位的输入框经 `t()` 得名；星以行为名、`aria-pressed`。
- jsdom 只钉落点的目标，真实焦点以三引擎的故事为准；键盘、指针、窄屏（800 宽）分开看。
- 按名字定位的按钮限在它的区域里（侧栏挂两份：限在 “Main sidebar”）；不在重载之后查控制台（M6-M8-09）；不给任何东西起名 “Notifications”。
- 每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去）与 `expectQuietConsole`（不放宽）结束。
- Docker 重启之后 testcontainers 的端口超时：等 ryuk 退出，以新标签重跑（M6-M8-05）。

## 性质与变异

| 性质 | 变异（标签） | Task |
|---|---|---|
| 树按 `sequence` 降序、同值按 `id` 降序 | 升序（`t2-mut-tree-ascending`）；同值不看 `id`（`t2-mut-tree-no-id`） | T2 |
| 文件夹的子项在文件夹下、不在顶层 | 不看 `parent_id`（`t2-mut-tree-flat`） | T2 |
| 按目标查按种类与目标 | 只比 `entity_id`（`t2-mut-favorite-of-id-only`，同 id 不同种类的一格失败） | T2 |
| 加入幂等：轮到时已有不发 | 去掉轮到时的检查（`t2-mut-add-twice`，“两次加入恰一个 `POST`”失败） | T2 |
| 体在轮到时算 | `add` 的检查在调用时做（`t2-mut-add-at-call`，“扣住一次移出、再加入：轮到时没有、发出”失败） | T2 |
| 移出：轮到时没有不发 | 照发（`t2-mut-remove-absent-sends`） | T2 |
| 不乐观 | 发出之前就写进列表（`t2-mut-optimistic`，“扣住回答时列表不变”失败） | T2 |
| 删文件夹连子项 | 只丢文件夹（`t2-mut-folder-children-kept`） | T2 |
| 200 原样不重复、读在外时不重复 | 回答无条件追加（`t2-mut-create-duplicates`，“取数在外时确认、读的回答已含它：只有一份”失败，`fetchedWhileChangeIsOut`；`prepended` 本来就跳过已有的 id，`C/lib/reconciled.ts:201-203`，以它为变异杀不死，re-98 m6） | T2 |
| 一个工作区一个队列 | 所有工作区一个队列（`t2-mut-one-queue`，“扣住 acme 的写，beta 的写照发”失败） | T2 |
| 在排与在途可观察、两种键都记 | `isPending` 恒假（`t2-mut-pending-never`）；`removeFavorite` 只记收藏 id（`t2-mut-pending-one-key`，“行菜单移出时同一目标的星停用”失败） | T2 |
| 404 `navigation.favorite_not_found` 丢掉那一行 | 照别的拒绝处理、列表不变（`t2-mut-not-found-kept`，“移出得 404：行没了、算成功；改名得 404：行没了、照失败”失败） | T2 |
| 放置修 D40 | “之下”恒为 `X − 10000`（`t2-mut-below-minus-step`，D40 的反例失败）；“之上”恒取中值（`t2-mut-above-first-mid`，第一个之上 `+10000` 失败）；以 `||` 判邻居（`t2-mut-zero-missing`，邻居为 0 的一格失败） | T2 |
| 移动一个 `PATCH`、只给变了的 | 体总带两个字段（`t2-mut-move-full-body`，“父没变不给 `parent_id`”失败） | T2 |
| 移动的目标在轮到时不在不发 | 照发（`t2-mut-move-gone-sends`） | T2 |
| 移动的原位与文件夹进文件夹在轮到时判 | 按算出的值判原位（`t2-mut-move-same-place-by-value`，“紧挨着 X 的放回 X 之上不发”失败）；轮到时不判文件夹进文件夹（`t2-mut-move-folder-into-folder-sends`，“X 在排队时被移进文件夹，再把文件夹放到 X 之下：不发”失败） | T2 |
| `forgetTarget` 丢指向它的、取数在外时重放 | 不经 `confirm` 而直接改 `value`（`t2-mut-forget-not-replayed`，“读在外时连带，读答到之后仍没有它”失败）；什么都不做（`t2-mut-forget-noop`） | T2 |
| `forgetProject` 丢项目里的全部、只丢它的 | 只丢项目自己的收藏（`t2-mut-forget-project-only`，迭代的收藏还在）；整个列表都丢（`t2-mut-forget-project-all`，文件夹与别的项目的收藏留着的一格失败） | T2 |
| `renameTarget` 改指向它的、取数在外时重放 | 什么都不做（`t2-mut-rename-target-noop`）；不经 `confirm` 而直接改 `value`（`t2-mut-rename-target-not-replayed`，“读在外时改名，读答到之后仍是新名”失败） | T2 |
| 门：管理员、成员 | `favoritesOn` 收访客（`t3-mut-gate-guest`，表的访客格与侧栏“访客不读”失败） | T3 |
| 读的键带工作区 id 与 slug | 只带 slug（`t3-mut-key-slug-only`） | T3 |
| 列表持有且非空才显示 | 空的也显示（`t3-mut-empty-shown`） | T3 |
| 名字取较新的一份 | 一律取 store 的（`t3-mut-name-store-always`，“收藏列表较新用收藏的”失败）；一律取收藏的（`t3-mut-name-favorite-always`，“改名之后侧栏跟上”失败） | T3 |
| 四个 store 的 `answeredAtOf` 是 getter 那一份的 | 项目给 0（`t3-mut-project-answered-zero`）；迭代（`t3-mut-cycle-answered-zero`）；模块（`t3-mut-module-answered-zero`）；视图（`t3-mut-view-answered-zero`）；各自的“较新用它”一格失败 | T3 |
| 工作区视图的链接 | 一律挂项目地址（`t3-mut-view-link-project`） | T3 |
| 文件夹：等回答再关 | 发出即关（`t3-mut-folder-close-early`，“扣住回答时输入还在、停用”失败） | T3 |
| 文件夹：409 在字段下、输入留着 | 409 经提示、关输入（`t3-mut-folder-409-toast`） | T3 |
| 文件夹：空白与超长不发 | trim 之前判空（`t3-mut-folder-untrimmed`，“只有空格”发出）；不判长度（`t3-mut-folder-no-length`） | T3 |
| 文件夹：在途时不取消 | Escape 不看在途（`t3-mut-folder-escape-in-flight`） | T3 |
| 改名没变不发 | 照发（`t3-mut-rename-same-sends`） | T3 |
| 开合存浏览器 | 只在内存（`t3-mut-open-not-stored`，重挂之后还开着的一格失败） | T3 |
| 没有持有时读失败给重试；持有时照画行 | 不显示失败（`t3-mut-read-failure-silent`）；有 `error` 就收起行（`t3-mut-failure-hides-rows`，“持有之后重读失败，行还在”失败，re-98 m3） | T3 |
| 列表没持有时不给新建文件夹 | 照给（`t3-mut-create-folder-unheld`） | T3 |
| 文件夹名按码点数 | 用 `.length`（`t3-mut-folder-length-units`，“255 个辅助平面字符照发”失败，re-98 o8） | T3 |
| `old-favorite-urls` 拦旧地址 | 在一个 web 文件里放回旧地址（`t3-mut-keyword-old-url`，`node tools/keywords.mjs` 失败） | T3 |
| 文件夹不进文件夹 | `getCanDrop` 放行文件夹进文件夹（`t4-mut-folder-into-folder`） | T4 |
| 进文件夹只改父 | `make-child` 也给 `sequence`（`t4-mut-make-child-reorders`） | T4 |
| 经页头移出 | 页头的落点什么都不做（`t4-mut-header-noop`） | T4 |
| 落回原位不发 | 照发（`t4-mut-same-place-sends`） | T4 |
| 移动失败有提示 | 只处理成功（`t4-mut-move-failure-silent`） | T4 |
| 手柄不是停留点 | 手柄可聚焦（`t4-mut-handle-tabstop`） | T4 |
| 星的门：工作区角色 | `useFavoritesOffered` 不看角色（`t5-mut-offered-guest`） | T5 |
| 星只在列表持有之后 | 不看持有（`t5-mut-star-before-held`） | T5 |
| 项目卡片另要成员、未归档 | 不看成员（`t5-mut-card-star-non-member`）；不看归档（`t5-mut-card-star-archived`） | T5 |
| 星在途停用 | 不看 `isPending`（`t5-mut-star-not-disabled`） | T5 |
| 星的按下不冒泡到行的链接 | 去掉 `preventDefault`、`stopPropagation`（`t5-mut-star-bubbles`，“按星不导航”失败） | T5 |
| 星以行为名、`aria-pressed` | 名字不带行名（`t5-mut-star-name-generic`）；`aria-pressed` 恒假（`t5-mut-star-not-pressed`） | T5 |
| 加入之后收藏区打开 | 不写开合（`t5-mut-open-after-add-missing`） | T5 |
| 星的拒绝有提示 | 只处理成功（`t5-mut-star-failure-silent`） | T5 |
| 项目归档、删除、离开之后连带 | 归档不调（`t5-mut-archive-no-forget`）；`forget` 不调（`t5-mut-forget-no-favorites`，删除与离开两格失败） | T5 |
| 迭代行、模块行与卡片的星不给归档的 | 迭代不看归档（`t6-mut-cycle-star-archived`）；模块卡片不渲染星（`t6-mut-module-card-no-star`） | T6 |
| 命令的门与快捷键 | 命令对访客可见（`t6-mut-command-guest`）；不看归档（`t6-mut-command-archived`）；快捷键不是 `shift+f`（`t6-mut-command-shortcut`）；标题不随状态（`t6-mut-command-title-fixed`） | T6 |
| 迭代、模块的归档与删除之后连带 | 迭代归档不调（`t6-mut-cycle-archive-no-forget`）；迭代删除不调（`t6-mut-cycle-remove-no-forget`）；模块归档不调（`t6-mut-module-archive-no-forget`）；模块删除不调（`t6-mut-module-remove-no-forget`） | T6 |
| 迭代、模块的改名在确认时写进收藏 | 迭代 `update` 不调 `rename`（`t6-mut-cycle-update-no-rename`）；模块 `changed` 不调（`t6-mut-module-update-no-rename`）；各自的“收藏列表较新时在本标签页改名，侧栏跟上”失败（re-98 I1） | T6 |
| 模块“收藏”筛选只留收藏的 | 筛选不看收藏（`t6-mut-module-favorites-ignored`）；归档列表也给（`t6-mut-module-favorites-archive`）；访客也给（`t6-mut-module-favorites-guest`）；清空不清它（`t6-mut-module-clear-keeps-favorites`）；`isFiltered` 不看收藏（`t6-mut-module-favorites-row-hidden`，“只开收藏：页面上已应用的一行在、页头亮”失败，re-98 I3） | T6 |
| 两种视图行都有星 | 工作区容器不给（`t7-mut-view-star-project-only`） | T7 |
| 视图删除之后连带 | 不调（`t7-mut-view-remove-no-forget`） | T7 |
| 视图的改名在确认时写进收藏 | `update` 不调 `rename`（`t7-mut-view-update-no-rename`，两种视图各一格） | T7 |
| 视图列表“收藏”筛选 | `viewsShown` 不看 `favorites`（`t7-mut-view-favorites-ignored`）；已应用的移除不清它（`t7-mut-view-favorites-chip`）；访客也给（`t7-mut-view-favorites-guest`）；`viewFiltersApplied` 不看收藏（`t7-mut-view-favorites-row-hidden`，“只开收藏：页面上已应用的一行在”失败，re-98 I3） | T7 |
| 进入项目记一次、只给成员的子树 | 记录点挂在包装层之外（`t8-mut-record-outside-wrapper`，“非成员不记”失败）；挂在每一页（`t8-mut-record-each-page`，“同一项目换页不再记”失败） | T8 |
| 工作项页在查找答到之后、不在收集箱时记 | 查找之前记（`t8-mut-record-before-lookup`）；收集箱里也记（`t8-mut-record-intake`） | T8 |
| 一次挂载一次；换会话再记 | 依赖里放 `data`（`t8-mut-record-each-answer`，重取之后再记）；依赖里没有最近访问 store（`t8-mut-record-session-fixed`，“换会话再记一次”失败，R29） | T8 |
| 记录失败静默 | 失败经提示（`t8-mut-record-toast`）；`console.error`（`t8-mut-record-console`） | T8 |
| 记录经一个队列 | 不排队（`t8-mut-record-unqueued`，“扣住前一个记录，后一个等它”失败） | T8 |
| 筛选与查询 | 一律不带 `entity_type`（`t9-mut-entity-type-missing`）；换筛选不换键（`t9-mut-filter-key-fixed`） | T9 |
| 每次挂载读 | 去掉 `dedupingInterval: 0`（`t9-mut-read-deduped`，真的 SWR，“卸下立刻再挂上再读一次”失败） | T9 |
| 状态给不出不画 | 给不出时画默认的（`t9-mut-state-fallback`） | T9 |
| 空态按筛选 | 一律默认的（`t9-mut-empty-default-only`） | T9 |
| 没有持有时读失败给重试；持有时照画行 | 不显示失败（`t9-mut-failure-silent`）；有 `error` 就收起行（`t9-mut-failure-hides-rows`） | T9 |
| 小部件不在 `loader` 之后 | 放回 `loader` 之后（`t9-mut-widget-after-loader`，“应用内换到项目列表没读过的工作区：首页的读恰一次”失败，re-98 m5） | T9 |
| 未归档的成员项目开预览、否则走链接 | 以项目角色判（`t9-mut-peek-by-role`，“已归档的项目被持有时仍走链接”失败，re-98 m4）；一律走链接（`t9-mut-link-always`）；Ctrl/⌘ 也开预览（`t9-mut-modifier-peeks`，格只用 Ctrl/⌘） | T9 |
| 首页的宿主读标签 | 去掉读（`t9-mut-host-no-labels`） | T9 |
| 迟到的回答只写自己的键 | 列表取最后答到的那个筛选的（`t9-mut-latest-filter-shown`） | T9 |

## Tasks

### T1 基线与文档（主控）

- **提交**：spec、plan、预检（含处理）。实施者简报（`L/implementer-brief.md`）与台账一样不进 git，随阶段归档。
- **P6b 的回填**（spec R22；数字取自归档 `M7-P6b-integration-20261128` 的台账 `progress.md` 末几行，主控核过）：`reviews/P6b-web-workspace-views-review.md` 的状态行（`:3`）、§2 的 A12 一行（`:49`）与 §9 末句（`:259`）补“终检与集成”：终检 `final.sh fin`（`e83b0881`，WebKit 单独 2 个 worker）一次全过、没有重跑、没有注销（bootstrap 第 0 路 459.6、第 1 路 645.6 秒，build+e2e 574 过，六个配置都过）；最终分支 CI `46143678387` attempt 1 全绿（web 765 秒；第 1 路 bootstrap 957.0 秒 = 79.7%，贴着八成的报告线、未越过，只报告）；合并 `c9f26bc2`（`--no-ff`，父 `6dead17d`、`e83b0881`）；main CI `46146403651` attempt 1 全绿（web 749 秒；第 1 路 871.3 秒 = 72.6%）；归档 `M7-P6b-integration-20261128`（10243 个文件，unknown 0）。P6b spec 的 A12（`specs/P6b-web-workspace-views.md:372`）照实勾选。
- **设计与清单**：设计 §14 的 P6b 一行（`D:355`）改为已完成、加 P7 一行；§12 的 P7 一行（`D:311`）链 spec。`docs/v0/frontend-changes.md` §3.1 的 M7 一行（`:206`）补 P6b 的合并。主控改，实现代理不碰设计。
- **关键词**：`K:2` 的 phase 改 `M7/P7`（R15）；跑 `node tools/keywords.mjs`。
- **台账工具**：`L/final.sh` 的页面配置改为八个（加 m4-collaboration、m5-content），WebKit 单独的循环不变，头部注释写 P7 的理由（R21）。
- **依赖**：根目录 `pnpm install --frozen-lockfile`（收据 `t1-install`）。
- **死成员的基线**：先 `pnpm --filter web exec react-router typegen`，再 `node L/dead-members/deadsym.mjs --tsv > L/dead-members/dead-before.tsv`、`node L/dead-members/domains.mjs L/dead-members/dead-before.tsv --rows M7`（收据 `t1-dead-before`）；与 P6b 的终表逐行比对，`P6aR:187` 留给 P7 的收藏 7 行列出。
- **检查**：文档链接（`doc-links.py`）与关键词，然后提交 `docs(M7/P7): …`。

### T2 收藏的数据层 → **检查点 0**

**Files**：新 `C/services/favorites/favorites.service.ts`；新 `C/store/favorites/{favorite.store,favorite-changes,favorite-tree,favorite-place}.ts` 与测试（`favorite.store.test.ts`、`favorite-changes.test.ts`、`favorite-tree.test.ts`、`favorite-place.test.ts`，各不过 400 行）；需要时新 `C/store/favorites/fake-favorites.ts`（测试的帮手：以 `FakeNerve` 建 store、种下列表、`favoriteOf…` 的工厂）；改 `C/store/root.store.ts`（116，加 `favorites`，与旧 `favorite` 并存；旧的两个名字以别名导入，Global Constraints）与 `root.store.test.ts:105`（允许清单）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t2-red`）：
  - service 四个操作的路径、方法与体（`FakeNerve`）；200 与 201 都回 `Favorite`。
  - store：以 slug 读、以 id 存；两个工作区各存各的；换会话之后旧的读不写；`answeredAt` 随回答增长。
  - 树：降序、同值按 `id`、子项在文件夹下、父不在的不显示；不依赖数组次序。
  - 按目标查：四种目标；视图一种两种容器；同 `entity_id` 不同种类不混。
  - 写（每格以 `C/store/fake-queue.ts` 的 `inTurn`/`sent` 与扣住的回答）：spec §2.1 的七个意图各自的“发什么、不发什么”；不乐观；`upserted` 不重复；删文件夹连子项；拒绝什么都不写；一个工作区一个队列（扣住 acme 的写，beta 的照发）；`isPending` 在排与在途为真、落定为假，两种键都记；404 `navigation.favorite_not_found` 丢掉那一行（移出算成功、改名与移动照失败）；`forgetTarget`、`forgetProject`（含读在外时连带、读答到之后仍没有它；文件夹与别的项目的收藏留着）；`renameTarget`（只改指向它的那一项的名字与给了的 `logo_props`，读在外时重放）；创建的回答在读在外时确认、读的回答已含它：只有一份。
  - 放置（表格式）：spec §2.1 的各行与 D40 的反例；移动的体（只给变了的、目标不在不发）；原位与文件夹进文件夹按轮到时的位置判（spec §2.1 的 `move`）。
- [ ] Step 2：实现到通过（spec §2.1）。`RootStore` 加 `favorites`（构造在 `notification` 之后、旧 `favorite` 旁），`root.store.test.ts:105` 改 20。
- [ ] Step 3：变异照上表 T2 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`；死成员整份比对（`t2-dead-after`：spec 写的窗口，每一行写“读者在 T<n>”）。提交 `feat(M7/P7): 收藏的数据层——按工作区持有本人的收藏，派生侧栏的树与按目标查；一个工作区一个写队列，体在轮到时从 nerve 最后回答的列表算、不乐观、意图幂等，nerve 说不在的收藏从列表里拿掉；降序的放置修 D40；跟随目标的连带与改名的窄口`。
- [ ] Step 4（**检查点 0**）：`final.sh cp0`；主控推送、核分支 CI，台账记基线（Global Constraints）。

### T3 侧栏收藏、读与门、旧层删除

**Files**：
- 改：`PROJ/sidebar.tsx`（47）；`SB/favorites-menu.tsx`（294）、`SB/favorite-folder.tsx`（299）、`SB/favorite-items/root.tsx`（144）与 `common/*`（icon 46、title 38、wrapper 34、quick-action 57、`index.ts` 12）改写（本 Task 不接拖放、不画手柄）；`C/hooks/use-favorite-item-details.tsx`（63）与测试（76）；`C/store/project/project.store.ts`（608）、`C/store/planning/cycle.store.ts`（163）、`module.store.ts`（170）、`C/store/views/view.store.ts`（156）各加 `answeredAtOf`；`C/store/root.store.ts`（去掉 `favorite`）、`root.store.test.ts`；`SB/favorite-folder.test.tsx`（55）；en/zh-CN（`common.json` 五个新键、删四个键，`accessibility.json` 删四个键）；`K`（`old-favorite-urls`、两条地址规则的名单）；`.oxlintrc.json`；`E/fixtures/mounts.ts`、`s2-web-app.spec.ts`、九份故事（允许清单）。
- 新：`C/hooks/store/use-favorites.ts`、`C/hooks/use-favorites-read.ts`、`C/lib/favorites-gate.ts`、`SB/favorite-link.ts`、`SB/favorite-folder-form.tsx` 与各自的测试；`use-favorite-item-details-newer.test.tsx`。
- 删（spec §2.11 的 T3 各行）：`C/services/favorite/*`、`C/store/favorite.store.ts`、`C/hooks/store/use-favorite.ts`、`PKG/types/src/favorite/*` 与桶导出、`PKG/constants/src/sidebar-favorites.ts` 与桶导出、`SB/favorite-items/common/helper.tsx`、`SB/new-fav-folder.tsx`、旧的拖放接线，连同只被它读的 `SB/favorites.helpers.ts`（72）与 `SB/favorite-items/common/favorite-item-drag-handle.tsx`（46，与 `common/index.ts:7` 的导出；文件夹一行自己的手柄也去掉）——T4 新建（re-98 I4）。

**Steps**：
- [ ] Step 0：T3 是最大的 Task（侧栏、读与门、文件夹表单、四个 store、旧层删除、`K`、S2 与九份故事），先在台账写方案（P6a T8、P6b T8 的先例，re-98 o5）：文件的拆法、提交的分界、每个读者的去向。再写读者表与受影响的故事表（`t3-readers-before`：spec §2.11 的 T3 各行逐个 `git grep -n`；`t3-affected`：以整个 `e2e/` 上的 `git grep` 为底，S2 的表与 `:511`、九份故事的十处、有收藏的三份旧故事）。
- [ ] Step 1：先写测试，看到失败（收据 `t3-red`）：
  - 门的表（管理员 T、成员 T、访客 F、没有角色 F）；`useFavoritesRead` 的键 `["FAVORITES", id, slug]`、门关着不读（真的 `useSessionSWR` 一格，“访客挂上没有请求”）。
  - 侧栏：没读、读着、空的都没有区；有收藏时的行与次序；一个开合按钮（名字、`aria-expanded`）、开合存浏览器（重挂之后还开着）；“新建文件夹”在区里有焦点时可见、列表没持有时不给；没有持有时读失败一行与重试，持有之后重读失败行还在（R28、re-98 m3）。
  - 行：名字与图标的较新一份（每种目标两格：store 较新、收藏较新）；五种链接；行菜单以行名为名、移出发 `DELETE` 它的 id、成功与拒绝各有提示。
  - 四个 store 的 `answeredAtOf`：给 getter 给出的那一份的时刻，给不出是 0（各两格）。
  - 文件夹：spec §2.3 的每一条（空白、超长按码点数、409、别的拒绝、等回答、在途停用、在途不取消、成功关与两种提示、改名没变不发、连同子项移出）；M3 的单测照允许清单跟随后照旧过。
- [ ] Step 2：实现到通过（spec §2.2、§2.3）；旧层照 §2.11 删（同一提交加 `old-favorite-urls`，跑 `node tools/keywords.mjs`）；读者表再跑一次（`t3-readers-after`：没有读者）；`root.store.test.ts:105` 改 19；`mounts.ts`、S2 与九份故事照允许清单改（同一提交）；新文件收入地址规则与 swr 名单。
- [ ] Step 3：变异照上表 T3 各行。门禁同 T2；`pnpm --filter @nerve/types test`、`@nerve/constants`（若有）；oxlint 计数（`t3-oxlint-count`）；死成员整份比对（`t3-dead-after`：收藏 7 行清零；T2 窗口的 T3 一半划掉）；`make build build-s3-fixture` 与 typegen 之后单配置 `t3-s2`、`t3-mounts`、`t3-furnished`，再跑一次默认配置的整套（`t3-e2e`）；`h9-intake-list-page.spec.ts` 的 `wc -l`（≤ 410）。提交：先换侧栏时旧的 `use-favorite.ts`、`helper.tsx`、`FAVORITE_ITEM_LINKS` 当场没有读者、knip 不过，所以删除并进第一个提交，或只写一个（照 P6b T11 的分界，re-98 o4）；每个提交都绿。提交说明：`feat(M7/P7): 侧栏收藏——管理员与成员在有侧栏的页读一次，名字与图标取 nerve 最后回答的那一份，链接加工作区视图一支；文件夹行内建与改名等回答、错误在字段下、409 不在前端查重，连同子项移出；开合存浏览器` 与 `refactor(M7/P7): 删除 Plane 的旧收藏层——store、service、hook、手写类型与链接常量；old-favorite-urls 拦收藏与最近访问的旧地址`。

### T4 拖放

**Files**：新 `SB/favorites.helpers.ts`、`SB/favorite-items/common/favorite-item-drag-handle.tsx`（T3 删去的，新建）与 `common/index.ts` 的导出；改 `SB/favorites-menu.tsx`、`SB/favorite-folder.tsx`（文件夹一行的手柄）、`SB/favorite-items/root.tsx`（T3 之后的版本，接拖放）；新 `SB/favorites.helpers.test.ts`、拖放的组件格（新文件）。T4 不删键（两个失败提示的键已在 T3 随拖放接线删去）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t4-red`）：spec §2.4 的五条规则（落点到意图的表：行的上半、下半，文件夹的中间，页头，文件夹拖到文件夹与文件夹里的行上）；每个意图到体（经 T2 的 `move`）；落回原位不发；失败经 `useRefusalToast`；手柄不可聚焦、没有提示。
- [ ] Step 2：实现到通过（spec §2.4）。
- [ ] Step 3：变异照上表 T4 各行。门禁同 T2；死成员（`t4-dead-after`，T2 窗口的 `move` 划掉）。提交 `feat(M7/P7): 侧栏收藏的拖放——排序与进出文件夹合成一个 PATCH，体在轮到时算；文件夹不进文件夹，经页头移出；失败有提示`。

### T5 项目的星与连带

**Files**：新 `C/components/favorites/{favorite-star.tsx,use-favorite-toggle.ts}`、`C/hooks/use-favorites-offered.ts` 与测试；改 `C/components/project/card.tsx`（345）、`C/store/project/project.store.ts`（T3 之后，连带两处）、`C/store/project/fake-projects.ts`（`projectTab` 的根）、`card.test.tsx`（一行替身）；按需 `form.test.tsx`（一行替身）；en/zh-CN（`favorite_star.{add,remove}`）；`K`（地址规则）；新 `card-star.test.tsx`、`project.store.favorites.test.ts`。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t5-red`）：
  - `useFavoritesOffered` 的表（角色 × 列表持有）。
  - 星：没持有不渲染；以行为名、`aria-pressed`；按下加入发 `POST`、移出发 `DELETE`；在途停用；成功提示、拒绝提示；加入之后 `IS_FAVORITE_MENU_OPEN` 为真；按下不触发行的链接。
  - 项目卡片：成员的未归档项目有星、非成员没有、已归档没有、访客没有；星不遮标题（标题行的右边距）。
  - 连带：归档、删除、离开之后收藏 store 里项目的全部收藏没了（迭代、模块、视图的也没了，文件夹留着），没有多的请求。
- [ ] Step 2：实现到通过（spec §2.5、§2.7）；`card.test.tsx` 与 `projectTab` 照允许清单改；按需的替身逐个记。
- [ ] Step 3：变异照上表 T5 各行。门禁同 T2；死成员（`t5-dead-after`）；单配置 `t5-projects`。提交 `feat(M7/P7): 项目的星与项目的连带——卡片的星给工作区管理员与成员、项目成员的未归档项目，等 nerve 回答、在途停用、加入之后打开收藏区；归档、删除、离开项目之后收藏立刻消失`。

### T6 迭代、模块的星、命令、连带与“收藏”筛选

**Files**：改 `C/components/cycles/list/cycle-list-item-action.tsx`（125）、`C/components/modules/module-list-item-action.tsx`（79）、`module-card-item.tsx`（123）；新 `C/components/power-k/ui/pages/context-based/use-favorite-command.ts` 与测试，改 `…/cycle/commands.ts`（47）、`…/module/commands.tsx`（112）；新 `C/store/favorites/target-favorites.ts`（`TargetFavorites`）；改 `C/store/planning/cycle-changes.ts`（180）、`module-changes.ts`（235）、`cycle.store.ts`、`module.store.ts`（T3 之后；连带与改名经同一个口）；改 `C/store/fake-project-lists.ts`（111，`listTab` 的根）；模块的“收藏”筛选：`C/store/planning/module_filter.store.ts`（69）、`C/components/modules/dropdowns/filters/root.tsx`（132）、`applied-filters/root.tsx`（110）、`modules-list-view.tsx`（126）、`PROJ/projects/(detail)/[projectId]/modules/(list)/page.tsx`（111，`:98-102`）、`C/components/modules/module-view-header.tsx`（89，`:63`）（re-98 I3）；翻转与替身：`cycles-view.test.tsx`（281）、`modules-list-view.test.tsx`（294）、`cycle/commands.test.tsx`（90）、`module/commands.test.tsx`（196）、`module-filters.test.tsx`（241），按需 `inline-changes.test.tsx`、`module-view-header.test.tsx`（允许清单）；en/zh-CN（`power-k.json` 四个键）；`K`；新 `cycle.store.favorites.test.ts`、`module.store.favorites.test.ts`、`module_filter.favorites.test.ts`、`modules/(list)/page-favorites.test.tsx`、行与卡片的星的格（新文件）；按需 `modules/(list)/page.test.tsx`（一行替身）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t6-red`）：
  - 迭代行、模块行与卡片的星：给未归档的、不给归档的（归档列表的行）；位置在快捷操作之前。
  - 命令（`use-favorite-command.test.tsx`，真权限与真 store）：门的表（成员 T、访客 F、兼工作区管理员的项目访客 T、归档的 F、列表没持有 F、在途 `isEnabled` F）；标题随状态；`shift+f`；在两组命令表的最前。
  - 连带：迭代、模块的归档与删除之后各自的收藏没了；恢复不加回；没有多的请求。
  - 改名的跟随：收藏列表答得比目标晚（先读目标、后读收藏）时，在本标签页改迭代、模块的名，侧栏立刻是新名，没有多的请求（re-98 I1）；模块的成员改动不改名。
  - “收藏”筛选：只留收藏的；只在会话（筛选 store 不写浏览器）；归档列表没有；访客没有；已应用的可移除；清空一并清掉；与别的筛选、搜索并用；挂模块列表页：只开“收藏”时已应用的一行在、页头的标记亮，移除之后列表回来（re-98 I3）。
- [ ] Step 2：实现到通过（spec §2.5、§2.6、§2.7）；翻转的格与替身照允许清单改（同一提交，逐条记台账）；四处“M7 brings them back / No star”说明照实改（`…/cycle/commands.ts:18`、`…/module/commands.tsx:30`、`module-list-item-action.tsx:37`、`module-card-item.tsx:40`）。
- [ ] Step 3：变异照上表 T6 各行。门禁同 T2；死成员（`t6-dead-after`）；单配置 `t6-m6-pages`、`t6-m6-acceptance`。提交 `feat(M7/P7): 迭代与模块的星、命令面板的收藏命令（shift+f）、连带与改名的跟随——归档与删除之后收藏立刻消失，改名之后侧栏立刻是新名；模块列表的“收藏”筛选只在这一会话，页面与页头按同一个判断显示它`。

### T7 视图的星、连带与“收藏”筛选

**Files**：改 `C/components/views/view-list-item-action.tsx`（43）；`C/store/views/view-changes.ts`（144）、`view.store.ts`（T3 之后）；`C/store/views/fake-views.ts`（91）与 `view.store.workspace.test.ts`（220，`:66` 的根）；项目视图列表的“收藏”筛选：`C/components/views/view-list-order.ts`（66）、`filters/filter-selection.tsx`（99）、`applied-filters/root.tsx`（78，`:38-39`）、`views-list.tsx`（94）、`PROJ/projects/(detail)/[projectId]/views/(list)/page.tsx`（86，`:69`、`:73-80`，re-98 I3）、`view-list-order.test.ts`（73，编译跟随）；按需渲染视图行的测试（允许清单）；`K`；新 `view.store.favorites.test.ts`、`view-list-order-favorites.test.ts`、`views/(list)/page-favorites.test.tsx`、视图行的星的格（新文件）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t7-red`）：两种容器的已存视图行有星（工作区视图的门同样是工作区角色）、默认四行没有；视图删除之后（两种）收藏没了、没有多的请求；收藏列表较新时在本标签页改视图的名与图标（两种），侧栏立刻跟上（re-98 I1）；“收藏”筛选只留收藏的视图、已应用的可移除、清空一并清掉、访客没有、工作区视图列表没有（只有搜索）；挂项目视图列表页：只开“收藏”时已应用的一行在，移除之后列表回来（re-98 I3）。
- [ ] Step 2：实现到通过（spec §2.5、§2.6、§2.7）；允许清单的跟随（同一提交）。
- [ ] Step 3：变异照上表 T7 各行。门禁同 T2；死成员（`t7-dead-after`：T2 的窗口必须清零）；单配置 `t7-m7-pages`。提交 `feat(M7/P7): 视图的星（项目视图与工作区视图，默认四个没有）、视图删除的连带与改名的跟随；项目视图列表的“收藏”筛选，页面与已应用的一行按同一个判断显示它`。

### T8 最近访问的记录 → **检查点 1**

**Files**：新 `C/services/recent-visits/recent-visits.service.ts`（只有 `record`）、`C/store/recent-visits/recent-visit.store.ts`（只有 `record` 与队列）、`C/hooks/store/use-recent-visits.ts`、`C/components/recent-visits/{project-visit.tsx,use-record-visit.ts}` 与测试；改 `PROJ/projects/(detail)/[projectId]/layout.tsx`（64）、`PROJ/browse/[workItem]/layout.tsx`（106）、`C/store/root.store.ts`、`root.store.test.ts`；`K`（地址规则）；e2e 的允许清单 T8 各条（`mounts.ts`、S2、五处 `VISIT`、八处“之后没有写”——其一在 `e2e/acceptance`——、`E/fixtures/intake-pages.ts:302`）。

**Steps**：
- [ ] Step 0：台账写受影响的故事表（`t8-affected`：以整个 `e2e/` 上的 `git grep` 为底——`stories`、`acceptance`、`drills`、`fixtures`，按方法过滤请求、对 `apiRequests`、`recordApi`、`writesSeen` 断言的每一处——列出 spec §2.13 的五处、八处、`writesSeen` 一族与“核过不受影响”的几处，各写理由；表上有清单之外的一处，先停下报告）。
- [ ] Step 1：先写测试，看到失败（收据 `t8-red`）：
  - service 的路径与体；store 的记录经一个队列。
  - 项目的记录点：成员的子树挂上记一次（体恰为 `{entity_type: "project", entity_id}`）；同一项目里换页不再记；换项目再记；包装层不渲染子树（非成员、已归档、读失败）时不记；换会话再记一次。
  - 工作项页：查找答到、不在收集箱记一次（体恰为 `{entity_type: "issue", entity_id}`）；查找在途、失败、在收集箱都不记；重取（`mutate`）之后不再记；以新的 key 重挂再记。
  - 失败静默：拒绝与 `SessionChangedError` 都没有提示、没有 `console.*`。
  - 预览不记（挂 `IssuePeekOverview` 的一格：没有 `POST /api/v0/me/recent-visits`）。
- [ ] Step 2：实现到通过（spec §2.8）；`root.store.test.ts:105` 改 20；e2e 的跟随照允许清单（同一提交）。
- [ ] Step 3：变异照上表 T8 各行。门禁同 T2；死成员（`t8-dead-after`）；`make build build-s3-fixture` 与 typegen 之后单配置 `t8-s2`、`t8-follow`、`t8-m6-acceptance`、`t8-intake`、`t8-unaffected`、`t8-m4-detail`、`t8-m4-collaboration`、`t8-m5-content`（与检查点 0 比对）；`h9-intake-list-page.spec.ts` 的 `wc -l`（≤ 410）。提交 `feat(M7/P7): 最近访问的记录——进入项目（成员的页）与打开工作项页（查找答到、不在收集箱）各记一次，预览、设置、归档、收集箱的选中不记，失败静默；e2e 的请求清单与“之后没有写”跟随`。
- [ ] Step 4（**检查点 1**）：`final.sh cp1`；主控推送、核分支 CI，记 web 作业三步与 e2e 两片。

### T9 首页的“最近”与预览 → **检查点 2**

**Files**：改 `C/components/home/home-body.tsx`（24）、`C/services/recent-visits/recent-visits.service.ts` 与 `C/store/recent-visits/recent-visit.store.ts`（加列表的读）；新 `C/components/home/widgets/recents/{index.tsx,filters.tsx,issue-row.tsx,project-row.tsx,home-peek.tsx,use-recents-read.ts}` 与测试；`home.json`（en、zh-CN，八个键）；`.oxlintrc.json`；`K`；`mounts.ts`（`RECENT_VISITS`）、S2 的 `"/"`（允许清单）。

**Steps**：
- [ ] Step 1：先写测试，看到失败（收据 `t9-red`）：
  - 读：键带工作区 id、slug 与筛选；三种筛选的查询；真的 SWR 一格“卸下立刻再挂上再读一次”；迟到的回答只写自己的键（换筛选、换工作区）。
  - 显示：骨架、行、空态（按筛选）、没有持有时失败与重试；读在途时、持有之后重读失败时照旧显示持有的（re-98 m3）；小部件不在 `loader` 之后：应用内换到项目列表没读过的工作区，首页的读恰一次（re-98 m5）。
  - 工作项行：标识与名字、几分钟前、状态（给得出画、给不出不画）、优先级、负责人；链接；项目是调用者的未归档成员项目（`joinedAndUnarchived`）时开预览，否则走链接——已归档的项目被项目 store 持有时也走链接（re-98 m4）；Ctrl/⌘ 与中键照链接。
  - 项目行：图标、标识、名字、成员（给得出才画）；链接。
  - 预览宿主：`hostScope`、打开时读那个项目的标签。
  - 标题是 `h3`；筛选按钮以标题为前缀得名（re-98 m11）；三种角色都有。
- [ ] Step 2：实现到通过（spec §2.9、§2.10）；S2 与 `mounts.ts` 照允许清单（同一提交）。
- [ ] Step 3：变异照上表 T9 各行。门禁同 T2；死成员（`t9-dead-after`）；单配置 `t9-home`。提交 `feat(M7/P7): 首页的“最近”——全部、工作项、项目三种筛选，两种行，空态、失败与重试，每次进入读一次；工作项行在项目是调用者的未归档成员项目时开预览，首页的预览宿主读项目的标签`。
- [ ] Step 4（**检查点 2**）：`final.sh cp2`；推送之前估出 e2e 两片；主控推送、核分支 CI。

### T10 H15 (page)

**Files**：新 `E/fixtures/favorite-pages.ts`（不过 300 行）、`E/stories/collaboration/h15-favorites-sidebar-page.spec.ts`、`h15-favorite-stars-page.spec.ts`。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（收据 `t10-red-page-sidebar`、`t10-red-page-stars`），看到断言在被改处失败。
- [ ] Step 2：写故事（spec §2.13 的表）：每个写的请求体精确、每步有库里的行（`favoriteRows`）；拖放的每一步扣住 `PATCH` 读体；收藏的读只在整页载入时数；访客的整页请求里没有 `FAVORITES`。
- [ ] Step 3：跑法：`make build build-s3-fixture` 与 typegen；chromium 聚焦；`--config=playwright.m7-pages.config.ts` 先 chromium 与 firefox、再 WebKit 单独 `--workers=2`（各自拷出 test-results）；新文件 chromium `--repeat-each=3`（`t10-repeat`）。记每个 `test()` 三引擎的时长；每份不过 400 行。提交 `test(M7/P7): H15 的页面故事——侧栏收藏的列、文件夹、拖动与访客；五种宿主的星与命令面板`。

### T11 H16 (page)

**Files**：新 `E/stories/collaboration/h16-favorite-cascades-page.spec.ts`（夹具用 T10 的）。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（`t11-red-page`）。
- [ ] Step 2：写故事（spec §2.13 的表）：每种动作之后侧栏立刻没有它，且这一页没有再读收藏（读在动作之后的请求里没有）；另一个账户重载之后没有；恢复不加回。
- [ ] Step 3：同 T10 的跑法（`t11-repeat`）。提交 `test(M7/P7): H16 的页面故事——删除、归档与离开之后收藏在本页立刻消失，别人重载之后也没有`。

### T12 H17 (page)

**Files**：新 `E/fixtures/home-pages.ts`（不过 200 行）、`E/stories/collaboration/h17-recent-visits-page.spec.ts`。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（`t12-red-page`）。
- [ ] Step 2：写故事（spec §2.13 的表）：`VISIT` 的次数与体（同一项目换页、后退、前进、工作项页、预览、收集箱的选中、设置、归档）；首页的行、筛选的查询、次序、同一目标一行、删除的不在、已归档项目的工作项（先去过项目页、读过归档列表之后也走链接）；应用内换工作区时首页的读恰一次；访客；首页的读不写；记录被拒与首页的读失败（只在 Chromium、目标页代码载入之后装路由；Chromium 自己记的 “Failed to load resource” 照 `planning/g3-change-cycle-page.spec.ts:281` 写进 `expectQuietConsole` 的预期，re-98 o9）。
- [ ] Step 3：同 T10 的跑法（`t12-repeat`）。提交 `test(M7/P7): H17 的页面故事——进入项目与打开工作项页各记一次，预览与收集箱不记；首页的“最近”`。

### T13 H20 (page)

**Files**：新 `E/stories/collaboration/h20-favorites-home-pages.spec.ts`。

**Steps**：
- [ ] Step 1：先跑期望错误的版本（`t13-red-page`）。
- [ ] Step 2：写故事（spec §2.13 的表）：键盘与读屏（三引擎记焦点；星按 Enter 之后焦点仍在星上，丢了就照 spec §2.5 改 `aria-disabled`，re-98 m11 ②；首页筛选按钮的名字）；迟到的回答（换工作区、另一个标签页换账号、换筛选）；800 宽一次。
- [ ] Step 3：同 T10 的跑法（`t13-repeat`）。提交 `test(M7/P7): H20 的收藏与首页——键盘与读屏，迟到的回答不写进新页面`。

### T14 文档与交接（主控）

- **plane-diff**：spec §7 的十行进 `docs/v0/plane-diff.md` 行为差异一节；Plane 的行号对照主目录的只读 `plane/` 逐一核过。
- **frontend-changes**：3.1 的 M7 一行写 P7；3.2 加行：收藏的数据层、侧栏、拖放、星与命令、连带、“收藏”筛选、最近访问的记录、首页；`:238`、`:250` 两行写处理结果。
- **交接**：spec §5 的表，每条写“处理结果（M7/P7）”；M8 的候选写进阶段记录 §7。
- **死成员**：重跑脚本（`t14-dead-final`），剩下的行写进阶段记录。
- **告警**：核对本段改到与新建的文件 oxlint 为零、计数等于上限（`t14-oxlint-files`）。
- **设计**：§14 的 P7 一行（主控）。
- **阶段记录** `reviews/P7-web-favorites-recents-review.md`：§1 方法与范围、§2 验收 → Task、§3 实施中的裁定（十条 L! 写明）、§4 向负责人说明（spec §3 的四件）、§5 门禁与 CI、§6 时长、§7 交给 M7 收官与 M8、§8 文档、§9 审查与集成。
- 检查文档链接与关键词，提交 `docs(M7/P7): …`。

### T15 门禁、整审与集成（主控）

1. **整分支审查**：一次 fresh 只读的整分支 deep-review，重点是 Review Focus 的 1–10。Critical/Important 在一次修复轮里各有 RED/GREEN；Minor 记台账，主控裁定改否。之后一次限定复审。
2. **终检**：`final.sh fin`（八个页面配置）。
3. **推送前的估算**：照 P4 spec §2.11 估出 e2e 时长。
4. **最终分支 CI**：推送准确的 HEAD（`git rev-parse`），核分支 CI（`wait-run.sh`；排队 0 个作业约 20 分钟即停下报告）。
5. **合并**：`--no-ff` 合入 main，推送后以 `ls-remote` 核对。
6. **main CI**：核准确的 main CI。
7. **归档与清理**：归档 `.superpowers/sdd/M7-P7-integration-<日期>`（`archive.py`），清理 worktree 与本地分支，远端分支核过已含于 main 之后删。
8. **回填**：spec A13 与设计 §14 由 M7 收官（或下一段）的 T1 回填。

## Review Focus

1. **R16 e2e 跟随（L!）**：`mounts.ts` 三个常量与头注释；S2 不变长、帮手不改意思；九份故事十处只改期望的一行；五处 `VISIT` 与它们为什么在项目布局里（G8、G13 `:77` 为什么不加）；八处“之后没有写”（其一在 `e2e/acceptance`）的每一处是排除还是算进期望、理由对不对；S2 表外的 `:511`；受影响表以整个 `e2e/` 为底；`writesSeen` 只排恰好 `POST /api/v0/me/recent-visits`（R26）；“核过不受影响”的几处理由成立；`watchPage`、`recordApi` 没有过滤。
2. **R11 记录（L!）**：两个记录点的位置（包装层里、查找答到之后）；一次挂载一次；换会话再记；“不记”的每一种；失败静默而控制台安静；M3 的项目布局与 M4/P5b 的工作项页只多了几行。
3. **R12 连带与 R7 的改名跟随（L!）**：七个连带的调用点、三个改名的调用点各在 nerve 确认之后、同一个 `runInAction`；不加请求；恢复不加回；`TargetFavorites` 一个口、调用时才读 `rootStore.favorites`；四份测试的根只追加，`projectTab` 与 `listTab` 给同一个收藏 store。
4. **R9、R10 星与命令（L!）**：门（工作区角色、列表持有、未归档、项目卡片的成员）；访客没有；以行为名与 `aria-pressed`；在途停用；加入之后打开区；命令在最前、`shift+f`；翻转的六格与替身照允许清单，没有别的断言变。
5. **R7 名字与图标（L!）**：四个 store 的 `answeredAtOf` 与 getter 同出一处；比较的方向；本标签页的改名经 `renameTarget`（两种回答次序都跟上，re-98 I1）；P6a 的单测的改法。
6. **R2 读与门（L!）**：只在侧栏读；设置页、通知页不读；侧栏两份一个请求；列表持有之前什么都不出现；名字没有 “sentry”。
7. **R13、R14 首页（L!）**：每次挂载读；小部件不在 `loader` 之后；持有时重读失败照画行；迟到的回答只写自己的键；开预览以 `joinedAndUnarchived` 判、Ctrl/⌘ 照链接；已归档项目的工作项没有状态、走链接；预览宿主读标签、没有改 M4/P5b 的预览；`h3`；筛选按钮的名字。
8. **R17 “收藏”筛选（L!）**：单独的选项、不随搜索；只在会话；访客没有；归档列表与工作区视图列表没有；两页“有没有筛选”各收成一个判断、页面与页头共用，格挂在页面上（re-98 I3）；M6 的筛选 store 与 P6a 的页面状态只多了这一项。
9. **R1、R3、R4、R6 数据层与侧栏**：一个工作区一个队列、体在轮到时算、不乐观、幂等、`isPending` 两种键；404 丢掉那一行；`move` 的原位与文件夹进文件夹在轮到时判；放置的表；文件夹的字段提示、码点数与 409；读失败与持有；拖放的五条规则；旧层删净（含 T3 删、T4 新建的 `favorites.helpers.ts` 与手柄）、`old-favorite-urls` 的样例、八个键；死成员 T2 的窗口在 T7 之后清零。
10. **共用的外壳**：M3 的侧栏与项目卡片、M6 的列表与命令面板、P6a/P6b 的视图行、M4/P5b 的工作项页；八个页面配置与整套默认 e2e。
