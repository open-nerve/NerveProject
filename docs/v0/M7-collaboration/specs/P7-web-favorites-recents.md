# M7 / P7：网页的收藏与最近访问——侧栏收藏、星与命令、连带、记录与首页的“最近”

> 状态：草稿；预检 [re-98](../../../reviews/re-98-M7P7预检.md) 结论“修正后通过”（严重 0、重要 4、次要 12、观察 10），主控的处理（re-98 §6）与起草之后的裁定 E-26–E-29 已折入（§3 末“预检 re-98 的处理”逐条列出）。基线 main `c9f26bc2`（P6b 的 noff，`c9f26bc2c4595e1a7b28ce0346e04a5e07197bed`）。设计见 [M7 设计](../M7-design.md) §2（H15 `:42`、H16 `:43`、H17 `:44`、H20 `:47`）、§3.8（`:123-128`）、§3.9（`:130-134`）、§7（数据层 `:241`、P7 一行 `:248`）、§12（P7 一行 `:311`）、§14；网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)（请求体轮到时才算）。调研与裁定在台账 `.superpowers/sdd/P7-web-favorites-recents/`：`research/scope.md`、`research/questions.md`（Q1–Q24）、`rulings.md`（E-1–E-29，本 spec 的 R1–R29 逐条对应，**R-n 即 E-n**）。P7 不改契约、服务端、部署与 CI：没有要负责人批的；四件告诉负责人（§3 末）。P3c、P3d、P6a、P6b 的裁定照旧约束 P7。新写的代码注释一律写 “M7/P7 spec”（代码里已有 “P7a”–“P7c spec” 指 M6/P7）。

缩写：`C` = `web/apps/web/core`；`W` = `web/apps/web`；`PROJ` = `W/app/(all)/[workspaceSlug]/(projects)`；`SB` = `C/components/workspace/sidebar/favorites`；`PKG` = `web/packages`；`E` = `e2e`；`K` = `tools/keywords.json`；`HO` = `docs/v0/M7-collaboration/handoffs`；`HO8` = `docs/v0/M8-open-release/handoffs`；`D` = M7 设计；`NAV` = `api/modules/navigation.yaml`；`PW` = 只读的 `plane/apps/web`；`P3cS`/`P3dS`/`P6aS`/`P6bS` = 同目录的各段 spec；`P6aR`/`P6bR` = `../reviews/` 下的阶段记录；`L` = 台账目录。

## 1. 目标与边界

P7 是 M7 的最后一段产品代码（`D:311`）：H15–H17 的页面与 H20 的收藏、首页。后端 P3c、P3d 已全（`NAV:221-450`），网页今天一行都没接：收藏 store 打 Plane 的旧地址、没有列表的读（`C/store/favorite.store.ts:14-36`、`:289-308`），侧栏只在列表非空时挂收藏（`PROJ/sidebar.tsx:36`、`:42`），所以收藏区从不出现；星全删了；最近访问网页里没有任何代码。P7 交付：

- **收藏的数据层**（§2.1，R1、R3、R4）：`FavoritesService`、一个按工作区 id 持有的收藏 store（`ReconciledByKey`）、派生的树与“按目标查”、一个工作区一个写队列（体在轮到时算、不乐观、意图幂等）、降序的放置（修 D40）。
- **读与门**（§2.2，R2，**L!**）：侧栏读，门 `favoritesOn`；访客、设置页、通知页不读。
- **侧栏收藏**（§2.3，R5–R8，**L!** 的是 R7）：区、行、名字与图标（nerve 最后回答的那一份；本标签页的改名经 `renameTarget` 跟上）、链接（加工作区视图一支）、文件夹（行内建、改名，等回答，`FieldError`，409 在字段下，修 D41；删连子项）、行菜单、开合存浏览器。
- **拖放**（§2.4，R4、R5）：排序与进出文件夹合成一个 `PATCH`，一层。
- **星与命令**（§2.5，R9、R10，**L!**）：项目卡片、迭代行、模块行与卡片、两种视图行；迭代页与模块页的命令面板（`shift+f`）。
- **“收藏”筛选**（§2.6，R17，**L!**）：模块列表与项目视图列表，只在这一会话。
- **连带与改名的跟随**（§2.7，R12、R7，**L!**）：本标签页的删除、归档、离开在 nerve 确认之后让收藏立刻消失，迭代、模块、视图的改名在确认之后立刻写进收藏，都不加请求。
- **最近访问的数据层与记录**（§2.8，R11，**L!**）：进入项目、打开工作项页各记一次。
- **首页的“最近”与预览**（§2.9、§2.10，R13、R14，**L!**）。
- **删除与守卫**（§2.11、§2.12，R15）：旧层全删，`old-favorite-urls`，phase `M7/P7`。
- **e2e**（§2.13，R16、R18、R21，**L!** 的是 R16）：`mounts.ts` 的三个新常量，S2、九份故事、八处“之后没有写”（含 `e2e/acceptance` 的一份）与 `writesSeen` 跟随；五份页面故事；八个页面配置。
- **P6b 的收尾回填**（T1，R22）。

**P7 不做**：
- 契约、服务端、CI：不改（R19）；`RecentVisitIssue` 只有 `state_id`，已归档项目里的工作项行没有状态图标（R13、R19）。
- `is_favorite`、建项目之后加入收藏、`favoriteProjectIds`（R20：不加回，交接写“不适用”）。
- 预览、项目设置、归档目录、收集箱的选中都不记最近访问（R11）。
- 键盘移动收藏（R5：只有指针拖放，与 Plane 相同）。
- 首页的快速链接、便签、小部件管理（R13，M1 3.14 已删）。
- 工作区视图列表页的“收藏”筛选（它只有搜索，P6b R13；R17 只管项目视图列表与模块列表）。
- 应用里的 `useLocalStorage` 副本本身（M6-M8-11，交 M8；P7 只把收藏那一处改用 `@nerve/hooks`）。
- “几分钟前”的本地化（`calculateTimeAgo` 是英文，日期本地化是 M8 的全应用决定，`HO8/M3-closeout.md:35`）。

**P7 之前 main 的样子**：见上；另：`FAVORITE_ITEM_LINKS` 把视图都挂在项目地址下，工作区视图没有链接（`PKG/constants/src/sidebar-favorites.ts:10-15`）；新建文件夹前端查重、不 trim、区分大小写、拿全部收藏比、不等回答就关（`SB/new-fav-folder.tsx:54`、`:91`、`:97`、`:130`，D41）；拖动“下方”恒减 10000、以全部收藏为基准、一次放下发两个请求（`C/store/favorite.store.ts:198-233`、`SB/favorites-menu.tsx:71-109`，D40）；`console.*` 九处（store `:148`、`:170`、`:193`、`:230`、`:243`、`:270`、`:305`，`SB/favorites-menu.tsx:176`，`SB/favorite-items/common/helper.tsx:14`）；首页只有问候与“没有项目”（`C/components/home/home-body.tsx:17-24`）。

**P7 之后 main 的样子**：工作区的管理员与成员在有侧栏的页上看到自己的收藏（有收藏时）：顶层与一层文件夹、`sequence` 降序，名字与图标是 nerve 最后回答的那一份，点进去到目标的页；能建、改名、删文件夹（错误在字段下），能拖动排序、进出文件夹；项目卡片、迭代与模块的行和卡片、两种视图的行与迭代页、模块页的命令面板有星（工作区角色管理员或成员、目标未归档、项目卡片另要是项目成员），按下等 nerve 回答、在途停用、加入之后收藏区打开；本标签页删除、归档、离开之后收藏立刻消失；模块列表与项目视图列表能只看收藏；进入项目、打开工作项页各记一次最近访问，失败静默；首页有“最近”（三种角色），筛选、两种行、空态、失败与重试，工作项行点开预览（没有角色时走链接）；Plane 的旧收藏层与它的地址、手写类型全部删除。

## 2. 交付合同

### 2.1 收藏的数据层（R1、R3、R4）

- **service**（新 `C/services/favorites/favorites.service.ts`，照 `C/services/views/views.service.ts` 的形状，类型只来自 `@nerve/api-client`：`Favorite`、`FavoriteCreate`、`FavoriteUpdate`、`FavoriteKind`）：`list(slug)`（`GET /api/v0/workspaces/{slug}/favorites`，回 `data`）、`create(slug, body)`（`POST` 同路径；201 新建、200 原样，两者都回 `Favorite`）、`update(favoriteId, body)`（`PATCH /api/v0/favorites/{favorite_id}`）、`remove(favoriteId)`（`DELETE` 同路径，204）。不转换，经 `unwrap`。
- **store**（新目录 `C/store/favorites/`；`RootStore.favorites`，见 §2.11 的并存窗口）：
  - `lists = new ReconciledByKey<Favorite[]>()`，键是工作区 id；`fetchFavorites(workspace: { id, slug })` 以 slug 读、以 id 存；`getFavorites(workspaceId)` 给持有的列表（没读过是 `undefined`）；`answeredAt(workspaceId)` 给列表的 `ReconciledByKey.answeredAt`（`C/lib/reconciled.ts:149`，全局递增，可与别的 store 的副本比）。
  - **树**（纯函数，新 `favorite-tree.ts`，store 以 `computedFn` 包）：`favoriteTree(list)` → 顶层（`parent_id === null`）与每个文件夹的子项（`parent_id === 文件夹 id`），各自按 `sequence` 降序、同值按 `id` 降序（与 `listFavorites` 的次序相同，`NAV:221-254`），不依赖列表在数组里的次序；父不在列表里的项不显示（父已删、行留着时服务端不会给，防御写一句注释）。不深拷贝（今天 `C/store/favorite.store.ts:90-105` 每次 `JSON.parse`）。
  - **按目标查**：`favoriteOf(workspaceId, kind, entityId)`（`kind` 是 `FavoriteKind` 去掉 `folder`）：列表里 `entity_type === kind && entity_id === entityId` 的那一项，没有是 `undefined`；视图一种同时查两种视图（契约的 `view` 兼指，`NAV:684-690`）。
  - `isPending(workspaceId, key)`（可观察）：这个工作区的队列里有没有指向这个键的写在排或在途。键有两种：目标 `"{kind}:{id}"` 与收藏 id；**入队时两种都记**（`add`/`remove` 记目标键，列表里有它时也记它的收藏 id；`removeFavorite`、`renameFolder`、`move` 记收藏 id，是指向目标的收藏时也记目标键），所以行菜单正在移出时，同一目标的星也停用（re-98 o3）。
- **写**（新 `favorite-changes.ts` 的 `FavoriteChanges`，写的一层不过 220 行）：**一个工作区一个队列**（`oneAtATime()` 按工作区 id 建在 `Map` 里，`C/lib/one-at-a-time.ts:14`）；每个写是**意图**，体在轮到时从 nerve 最后回答的列表（`lists.get(workspaceId)`）算；**不乐观**：nerve 确认之前列表不变，确认之后以 `Reconciled` 的确认写进去（`confirm`，取数在外时照样重放）；拒绝与失败让写失败、什么都不写，只有一种例外：**nerve 答 404 `navigation.favorite_not_found`**（以 `problemCode` 判，`C/lib/api-error.ts:34`；已删的、目标他看不到的收藏都答这个，`NAV:351-371`）时，以 `confirm` 从列表里丢掉这一项（文件夹连子项），不重读——nerve 说它不在了，留着它行菜单与星都关不掉它；`remove`、`removeFavorite` 照成功算（提示“已移出”），`renameFolder`、`move` 照失败提示（re-98 m2）。意图与体：
  - `add(workspace, kind, entityId)`：轮到时 `favoriteOf` 已有 → 不发、给已有的那一项；否则 `POST {entity_type, entity_id}`（不给 `parent_id`、`name`），回答（201 或 200）以 `upserted` 写进列表（200 的那一项若本标签页的列表已有，原地替换，不重复）。
  - `remove(workspace, kind, entityId)`：轮到时 `favoriteOf` 没有 → 不发；否则 `DELETE` 它的 id，确认之后 `dropped`。
  - `removeFavorite(workspace, favoriteId)`：侧栏的行菜单与文件夹菜单。轮到时列表里没有 → 不发；否则 `DELETE`；确认之后丢掉它，是文件夹则连同 `parent_id` 是它的全部（契约：文件夹连它里面的一起删，`NAV:351-371`）。
  - `createFolder(workspace, name)`：`POST {entity_type: "folder", name}`（`name` 是调用方已 trim 的）；回答 `upserted`；409 `navigation.favorite_folder_name_taken` 与别的拒绝都让写失败（`ApiError`，表单读它，§2.3）。
  - `renameFolder(workspace, favoriteId, name)`：轮到时列表里没有或名字已等于 `name` → 不发；否则 `PATCH {name}`，回答 `replaced`。
  - `move(workspace, favoriteId, drop)`：§2.4；轮到时算体。不发的三种，都按轮到时的列表判（re-98 m9）：被拖的或落点的那一行已不在；**原位**——父不变且被拖的本来就紧挨在 X 的那一侧（放到 X 之上时它本来就是 X 上面那一项，之下时是 X 下面那一项），或落进它已在的文件夹、在顶层时经页头“移出”（按位置判，不按算出的值：去掉被拖的之后对同一对邻居算出的是新的中值，不等于原值）；**文件夹进文件夹**——被拖的是文件夹而落点在某个文件夹里（X 的父是文件夹，或落在文件夹一行的中间；落点规则在 §2.4 第 2、4 条已挡，轮到时 X 才被移进文件夹的也在这里挡住，否则发出去的体必答 422）。否则一个 `PATCH`（只给变了的 `parent_id`、`sequence`），回答 `replaced`。
  - 每个写在排与在途时它的键进 `isPending`（星以它停用，§2.5；侧栏的行在途时不可再拖）。
- **放置**（新 `favorite-place.ts` 的 `placeAmong(siblings, droppedOnId, at)`，纯函数，不动 `C/lib/place-between.ts`——它是升序的、三个读者，`:6-11`、`:23-33`）：`siblings` 是目的地按显示次序（降序）的兄弟，**已去掉被拖的那一项**；放到 X 之上：X 是第一个 → `X.sequence + 10000`，否则 X 与它上面那一项的中值；放到 X 之下：X 是最后一个 → `X.sequence − 10000`，否则 X 与它下面那一项的中值。`0` 与负数照常（不以 `||` 判有无，今天 `C/store/favorite.store.ts:208`、`:212` 的错法；`:219` 是恒减 10000 的那一行）；相等的两个邻居得它们的值（之后的次序落到 `id`，写进注释，契约不重排，`P3cS:98`）。表格式单测含 D40 的反例：A 30000、B 29000、C 20000，C 放到 A 之下得 29500（Plane 得 20000，仍在 B 后）。
- **跟随目标的窄口**（都不发请求，§2.7）：`forgetTarget(workspaceId, kind, entityId)`、`forgetProject(workspaceId, projectId)`；`renameTarget(workspaceId, kind, entityId, shown)`（`shown` 是 `{ name }` 或 `{ name, logo_props }`）：以 `confirm` 改列表里指向这个目标的那一项的 `name`，给了 `logo_props` 时连它（取数在外时照样重放；没有指向它的收藏时什么都不改）。
- **会话**：store 随 `RootStore` 一代一个；读的键带 `loginId`（经 `useSessionSWR`）；换会话时在途的读得 `undefined`、不写（`C/lib/reconciled.ts:51-78`）。

### 2.2 读与门（R2，L!）

- **门**（新 `C/lib/favorites-gate.ts`）：`favoritesOn(role: WorkspaceRole | undefined)`：管理员、成员为真，访客与没有角色为假（同 `workspaceViewsOn` 的写法，`C/lib/views-entry.ts:44-46`；服务端 `favorite.read`/`favorite.write` 只给这两种）。名字不得含不分大小写的 “sentry”（`K:1224`）：`favoritesEntry…`、`favoritesEntryOn` 一类都不行。
- **读**（新 `C/hooks/use-favorites-read.ts` 的 `useFavoritesRead(workspaceSlug)`）：工作区按路由的 slug 取（`getWorkspaceBySlug`），角色按 slug 取（`getWorkspaceRoleByWorkspaceSlug`）；`useSessionSWR(favoritesOn(role) && workspace ? ["FAVORITES", workspace.id, workspace.slug] : null, (id, slug) => fetchFavorites({ id, slug }))`，SWR 的选项照默认。给出读的 `error` 与 `mutate`（§2.3 的失败与重试）。
- **谁读**：只有应用侧栏 `AppSidebar`（`PROJ/sidebar.tsx`）。它在每个有应用侧栏的页上（工作区的页里除了设置页与通知页：设置在 `(settings)` 布局下，`W/app/routes/core.ts:214`；通知页的侧栏返回 `null`，`PROJ/_sidebar.tsx:27`、`:32`）。侧栏的子树挂两份（主栏与收起时的浮出栏，`C/components/sidebar/resizable-sidebar.tsx:184`、`:228`，后者 `inert`），两个 hook 同一个键，SWR 合成一个请求；故事按 “Main sidebar” 区域定位。
- **用到列表的地方**（星、命令、筛选）都只读 store，不自己读；它们都在有侧栏的页上。新 `C/hooks/use-favorites-offered.ts` 的 `useFavoritesOffered(workspaceSlug)`：`favoritesOn(按 slug 的角色)` 且这个工作区的列表已持有时给工作区 id，否则 `undefined`。**列表持有之前，星、命令与“收藏”筛选都不出现**（不出现“按下即错”的星）。
- `.oxlintrc.json` 的 swr 名单（`:121-223` 那一组）收入 `use-favorites-read.ts`。

### 2.3 侧栏收藏（R5–R8；R7 是 L!）

- **区**：`AppSidebar` 以门与读挂收藏区；列表持有且非空时显示（照 Plane，`PW/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx:38`）；没读、读着、空的都不显示；**持有着列表时一律照画行**，之后的重读失败（侧栏重挂时的重读）也不收起它（re-98 m3，同 P6b 整审 O7 的一类：`Reconciled.fetch` 最新的一次读失败时值照旧，`C/lib/reconciled.ts:72-74`，而 swr 的键上有 `error`）；**没有持有而读失败**时显示区的页头，展开时一行“没能读到收藏”（新键 `favorite_list.load_failed`）与重试（`common.retry`，`mutate()`）（R28）。区的开合存浏览器：键 `IS_FAVORITE_MENU_OPEN`（`PKG/constants/src/workspace.ts:121`，键不变），读写经 `@nerve/hooks` 的 `useLocalStorage`（M6-M8-11 的那一处，`SB/favorites-menu.tsx:30`、`:51`；它以窗口事件让同一个键的每个 hook 跟上，两份侧栏与星都经它）；缺省关。区的页头是**一个**开合按钮，以 `t("favorites")` 为名、`aria-expanded`（今天两个按钮同名 “Open favorites menu”，`SB/favorites-menu.tsx:207-208`、`:233-234`）；“新建文件夹”的按钮以 `aria_labels.projects_sidebar.create_favorites_folder` 为名，悬停或区里有焦点时可见（`focus-within`，今天 `opacity-0`、焦点看不到，`:213`，`HO/M3-closeout.md:62`），按下即展开区、在最上显示文件夹的输入；列表没持有时不给（建成了也写不进没持有的列表：没读过的键上 `confirm` 是空操作，`C/lib/reconciled.ts:182-184`）。
- **行**：树的顶层与每个文件夹的子项（§2.1），React 的 key 是收藏 id（今天缺 key，`SB/favorites-menu.tsx:266`）。每行：图标、名字、链接（整行是链接）、行菜单（以 `t("common.item_options", { name })` 为名，菜单项“移出收藏”`favorite_item.remove`）；拖动的手柄是 T4 的（§2.4）：T3 的行与文件夹不画手柄；T4 新建的手柄不是停留点（`tabIndex=-1`、`aria-hidden`），没有提示（今天按不存在的 `sort_order` 写死英文的提示，`SB/favorite-items/common/favorite-item-drag-handle.tsx:29`、`SB/favorite-folder.tsx:187`，随 T3 删去）。
- **名字与图标**（R7，L!）：`C/hooks/use-favorite-item-details.tsx` 改写：目标的 store 给得出它、且那一份比收藏列表答得晚（`answeredAtOf(id) > favorites.answeredAt(workspaceId)`）就用它的名字与图标，否则用收藏的 `name`、`logo_props`（v0 7.7“显示 nerve 最后回答的那一份”）：项目经项目 store（名字、`logo_props`），迭代、模块经各自的 store（名字；图标是种类的图标），视图经视图 store（名字、`logo_props`），文件夹用自己的。为此四个 store 各给一个 `answeredAtOf(id)`：它的 `get…ById` 给出的那一份的 `answeredAt`，给不出是 0；两者从同一个私有的“最新的一份”取（`C/store/views/view.store.ts:101-110`、`C/store/planning/cycle.store.ts:74-83`、`C/store/planning/module.store.ts:77-86`、`C/store/project/project.store.ts:211-235` 今天都在 getter 里私下算），不复制判断。比较只管“目标读得比收藏晚”：迭代、模块、视图的改名在本标签页是 `confirm`，不推进 `answeredAt`（`C/lib/reconciled.ts:117-123`；只有取数、分段取数与 `receive` 推进，如 `:69`），所以本标签页的改名另经 `renameTarget` 在 nerve 确认时写进收藏列表（§2.7，re-98 I1 (a)）；项目的改动经 `details.receive`（`C/store/project/project.store.ts:584`），推进时刻，比较即可。于是：本标签页改名之后侧栏立刻跟上（两种回答次序都是）；别的标签页改名在下一次读收藏或读目标时跟上；本页 store 的旧副本盖不过更新的收藏列表。
- **链接**（新 `SB/favorite-link.ts` 的纯函数，删 `@nerve/constants` 的 `FAVORITE_ITEM_LINKS` 与 `SB/favorite-items/common/helper.tsx`）：`project` → `/{slug}/projects/{entity_id}/issues`；`cycle` → `/{slug}/projects/{project_id}/cycles/{entity_id}`；`module` → `…/modules/{entity_id}`；`view` 有 `project_id` → `…/projects/{project_id}/views/{entity_id}`，没有（工作区视图）→ `/{slug}/workspace-views/{entity_id}`；`folder` 没有链接。迭代、模块、视图的开关关着时照列（契约照列，`NAV:221-254`），点进去是各自的停用页（R7）。
- **文件夹**（R6，修 D41）：文件夹一行是一个开合按钮（以文件夹名为名、`aria-expanded`、Space 与 Enter 开合，没有嵌套按钮——M3 的单测 `SB/favorite-folder.test.tsx:31-55` 照旧钉它）与文件夹菜单（以 `t("common.item_options", { name })` 为名：“改名”`favorite_folder.rename`、“连同里面的收藏移出”`favorite_folder.remove`，后者不加确认，照 Plane）。新建与改名是行内输入（新 `SB/favorite-folder-form.tsx` 代替 `SB/new-fav-folder.tsx`），以 `aria_labels.projects_sidebar.enter_folder_name` 为名：
  - 提交（Enter）时 trim；空的 → 字段下 `FieldError` `folder_name_cannot_be_empty`、`aria-invalid`、不发请求；超过 255 个字符（按码点数，`[...name].length`，与服务端的 `utf8.RuneCountInString` 一致，`server/internal/modules/navigation/domain/content.go:138`；不用 `.length`，re-98 o8）→ `FieldError` `favorite_folder.name_too_long`（新键）、不发；**不在前端查重**（删 `folder_already_exists` 与 `existingFolders`）。
  - 发出之后输入停用、不关；nerve 答 409 `navigation.favorite_folder_name_taken` → 字段下 `FieldError`（`errors.navigation_favorite_folder_name_taken`，`C/lib/error-messages.ts:108`）、输入留着、焦点回输入（`useRefusedFocus`，M3/M4 的写法）；别的拒绝经 `useRefusalToast`、输入留着；成功 → 关输入（建的提示 `favorite_created_successfully`，改名的提示 `favorite_updated_successfully`——今天改名成功的那一个，`SB/new-fav-folder.tsx:119`）。改名时名字没变（trim 之后、区分大小写）→ 不发、关输入。
  - Escape 与点外面取消，只在没有在途时（今天点外面即取消、不看在途，`SB/new-fav-folder.tsx:138-140`）。
  - 只一层：拖放规则不让文件夹进文件夹（§2.4），服务端兜底（文件夹不能有父）。
- **移出**：行菜单的“移出收藏”与文件夹菜单的“连同里面的收藏移出”经 `removeFavorite`；成功提示 `favorite_removed_successfully`，拒绝经 `useRefusalToast`；不 `console.error`。
- **地址**：所有新写、改写的文件从路由参数取地址（`useParams()` 的 `workspaceSlug`），收入 `workspace-address-route`（§2.12）；旧 store 读 `workspaceRoot.currentWorkspace`（`C/store/favorite.store.ts:72-84`）随删除了结（`HO/M3-closeout.md:95` 点名的最后一个）。

### 2.4 拖放（R4、R5）

- 照 Plane 的 pragmatic-dnd（`@atlaskit/pragmatic-drag-and-drop*`，同 `SB/favorite-folder.tsx`、`SB/favorite-items/root.tsx` 今天的写法），只有指针；键盘能做的是链接、行菜单、文件夹的开合与菜单，不另加键盘移动（R5）。`SB/favorites.helpers.ts`（72）**T3 删、T4 新建**（re-98 I4）：它引 `IFavorite`（`:8`、`:60`，T3 删），三个导出只被 T3 删去的拖放接线读（`SB/favorites-menu.tsx:34-35`、`SB/favorite-folder.tsx:44`、`SB/favorite-items/root.tsx:27`），留到 T4 不是编译不过就是 knip 报未使用；T4 新建的 `TargetData` 用 `Favorite` 的字段（`parent_id`，今天是 `parent`）。手柄 `SB/favorite-items/common/favorite-item-drag-handle.tsx`（46，读者只有 `SB/favorite-items/root.tsx:28`、`:131` 与 `common/index.ts:7`）与文件夹一行自己的手柄同样 T3 删、T4 新建：T3 的行不画手柄（拖不动的手柄是死的界面）；“手柄不是停留点”的格与 `t4-mut-handle-tabstop` 在 T4。
- **落点与请求**（每次放下最多一个 `PATCH`，体在轮到时从 nerve 最后回答的列表算，§2.1 的 `move`）：
  1. 落在行 X 的上半或下半（X 在顶层或某个文件夹里）：被拖的放进 X 的兄弟、在 X 之上或之下；体 = `sequence`（`placeAmong`）加上 `parent_id`（只在父变了时，进那个文件夹或回顶层）。
  2. 落在文件夹 F 一行的中间：放进 F（`parent_id: F`），`sequence` 不动（照 Plane）；被拖的是文件夹 → 不允许（不显示落点、不发）；已在 F 里 → 不发。
  3. 落在收藏区的页头：移出文件夹（`parent_id: null`），`sequence` 不动；已在顶层 → 不发。
  4. 被拖的是文件夹：只能在顶层的行之上、之下；落在文件夹里的行上 → 不允许（D42 在网页上由规则守住，服务端兜底）；轮到时才变成这样的（X 已被移进文件夹）也不发（§2.1 的 `move`）。
  5. 落回原位（同一个父、被拖的本来就紧挨在落点的那一侧；按位置判，不按算出的值，§2.1 的 `move`）→ 不发。
- 失败（拒绝或没有回答）经 `useRefusalToast`；列表不变（不乐观）；不 `console.log`（今天页头的落点 `SB/favorites-menu.tsx:176`）。

### 2.5 星与命令（R9、R10，L!）

- **星**（新 `C/components/favorites/favorite-star.tsx` 的 `FavoriteStar`，不回 `@nerve/ui`：只有应用读，名字要经 `t()`）：`<button type="button" aria-pressed>`，以行为名（“Add {name} to favorites”`favorite_star.add` / “Remove {name} from favorites”`favorite_star.remove`，M3 的行名规则 `HO/M3-closeout.md:66`）；只在 `useFavoritesOffered(slug)` 给出工作区 id、且宿主说可收藏（`offered`）时渲染；按下的是加入还是移出取按下那一刻的 `favoriteOf`，意图经 `add`/`remove`（§2.1，幂等）；在途（`isPending`）停用（键盘按 Enter 之后焦点仍在星上：H20 (page) 三引擎各记一次；有引擎在停用时丢焦点，就改为 `aria-disabled` 并忽略在途时的按下，re-98 m11 ②）；成功提示（`favorite_created_successfully` / `favorite_removed_successfully`），拒绝经 `useRefusalToast`，都经 `followInSession`；加入成功之后把收藏区打开（`IS_FAVORITE_MENU_OPEN` 写真，Plane `PW/core/components/cycles/list/cycle-list-item-action.tsx:119`）。按下不冒泡到行的链接（`preventDefault`、`stopPropagation`）。新 `C/components/favorites/use-favorite-toggle.ts` 是星与命令面板共用的一半（意图、提示、打开区）。
- **门**（R9）：`favoritesOn(工作区角色)` 且列表已持有（`useFavoritesOffered`）且目标未归档；项目卡片另要调用者是项目成员（`joinedAndUnarchived`，`C/lib/project-scope.ts:10`；不是成员的项目收藏会 422）；访客没有星。门不看项目里的角色（项目访客兼工作区成员也有星，契约允许，`P3cS:86-94`）。
- **宿主与位置**：
  - 项目卡片（`C/components/project/card.tsx`）：复制链接按钮旁（`:216-233`），同一个绝对定位的容器；标题行的右边距由 `pr-[52px]`（`:185`）放宽到容得下两个按钮。
  - 迭代行（`C/components/cycles/list/cycle-list-item-action.tsx:122` 的快捷操作之前）：门加 `row.archived_at === null`。
  - 模块行（`C/components/modules/module-list-item-action.tsx:76` 之前）与模块卡片（`module-card-item.tsx:113` 之前）：门加未归档；两处“No star to favorite it”的说明（`:37`、`:40`）照实改。
  - 视图行（R10）：共用的 `ViewListItemAction`（`C/components/views/view-list-item-action.tsx:31-43`，P6b R15）里快捷操作之前，两种容器都给；默认四个工作区视图的行（`C/components/workspace/views/default-view-row.tsx`）不经它，没有星。
  - 侧栏项目菜单：不加（Plane 注释掉了）。
- **命令面板**（新 `C/components/power-k/ui/pages/context-based/use-favorite-command.ts`）：迭代页（`…/cycle/commands.ts:28-46`）与模块页（`…/module/commands.tsx:62` 起）的命令表**最前**加一条：id `toggle_cycle_favorite` / `toggle_module_favorite`，标题按此刻的状态 `power_k.contextual_actions.{cycle,module}.{add_to_favorites,remove_from_favorites}`，`modifierShortcut: "shift+f"`（Plane `PW/core/components/power-k/ui/pages/context-based/cycle/commands.ts:72-86`；在这两组里没有撞上的），`isVisible`/`isEnabled` = 星的门（路由的迭代或模块、它未归档），在途时 `isEnabled` 为假，动作经 `use-favorite-toggle.ts`。两处“favorites are gone until M7”的说明（`…/cycle/commands.ts:18`、`…/module/commands.tsx:30`）照实改。
- **会翻转的既有格**（R9；调研说四格，实为四份文件六格，§3 补记 2）：`C/components/cycles/cycles-view.test.tsx:220-230`、`C/components/modules/modules-list-view.test.tsx:206-213` 与 `:246-262`（卡片的按钮 `:254-259`），`…/cycle/commands.test.tsx:68-72`，`…/module/commands.test.tsx:117-124` 与 `:126-138`；改法见 plan 的允许清单。

### 2.6 “收藏”筛选（R17，L!）

- **照 Plane 是一个单独的选项**（不是带标题的一节，不随菜单的搜索过滤：`PW/core/components/modules/dropdowns/filters/root.tsx:65-73`、`PW/core/components/views/filters/filter-selection.tsx:80-89`），文字 `t("favorites")`，`aria-pressed`；只在 `useFavoritesOffered(slug)` 给出时出现（访客没有：不给什么都不做的选项）。
- **模块列表**：`C/store/planning/module_filter.store.ts` 每个列表的状态加 `favorites: boolean`（会话，不进浏览器，理由同成员筛选，`:33-40` 的说明），`getFavoritesOnly`/`setFavoritesOnly`，`clearAllFilters` 一并清掉；只给 `default` 列表（归档的不可收藏）；菜单 `C/components/modules/dropdowns/filters/root.tsx` 最上加选项；已应用的筛选（`C/components/modules/applied-filters/root.tsx`）加一项“Favorites”可移除；列表（`C/components/modules/modules-list-view.tsx:32` 的 `filterModuleRows`）只留 `favoriteOf(workspaceId, "module", row.id)` 给得出的。`@nerve/utils` 的 `ModuleFilters`（`PKG/utils/src/planning.ts:89-95`）不动：它只管模块自己的字段。
- **项目视图列表**：`C/components/views/view-list-order.ts` 的 `ViewListFilters` 加 `favorites: boolean`（`NO_VIEW_FILTERS` 为假；页面状态本来只在这次进入，`C/components/views/page-state.tsx`）；`viewsShown(views, choices, favored)` 多一个参数（`(viewId) => boolean`），`favorites` 为真时只留 `favored` 的；`filters/filter-selection.tsx` 最上加选项，`applied-filters/root.tsx` 加一项；两处按数组取键的帮手（`filter-selection.tsx:43`、`applied-filters/root.tsx:42`）的键类型收窄为数组键。
- **页面上“有没有筛选”的判断**（re-98 I3）：今天两页各在页面上判断，只看数组筛选。模块列表页以 `calculateTotalFilters(getFilters(projectId, "default")) !== 0` 决定画不画已应用的一行（`PROJ/projects/(detail)/[projectId]/modules/(list)/page.tsx:98-102`），页头的“已筛选”标记是同一个式子（`C/components/modules/module-view-header.tsx:63`）；项目视图列表页以 `owned_by.length > 0 || created_at.length > 0` 决定（`PROJ/projects/(detail)/[projectId]/views/(list)/page.tsx:69`、`:73-80`），已应用的一行自己在没有数组筛选时也返回空（`C/components/views/applied-filters/root.tsx:38-39`）。“收藏”在 `filters` 之外，照这些式子只开它时没有可移除的一项，模块页头也不亮。所以各收成**一个**判断，页面、页头与已应用的一行共用：模块是筛选 store 的 `isFiltered(projectId, list)`（数组筛选的数不为 0，或 `default` 列表的 `favorites`），视图是 `view-list-order.ts` 的 `viewFiltersApplied(filters)`（两个数组之一非空，或 `favorites`）。格挂在页面上：只开“收藏”时已应用的一行在、模块页头的标记亮，移除之后列表回来。
- 都在 T6、T7，与各自的星同一个 Task（R17、R24）。

### 2.7 连带与改名的跟随（R12、R7，L!）

- **窄口**：`forgetTarget(workspaceId, kind, entityId)`：以 `confirm` 丢掉这个工作区列表里指向它的收藏（取数在外时照样重放）；`forgetProject(workspaceId, projectId)`：丢掉 `project_id === projectId` 的全部（项目、它的迭代、模块、项目视图的收藏；文件夹的 `project_id` 为空，留着）。都不发请求（服务端在原事务里软删，`P3cS` §2.6；离开项目时行留着、读时过滤）。`renameTarget`（§2.1）同样不发请求，只把 nerve 确认的名字写进收藏列表（§2.3 的名字规则缺它不成立，re-98 I1）。
- **调用方**（nerve 确认之后，与各自的确认同一个 `runInAction`）：
  - `C/store/planning/cycle-changes.ts`：`archive`（`:115`）、`remove`（`:126-135`）；`unarchive` 不加回（服务端也不加回）。
  - `C/store/planning/module-changes.ts`：`archive`（`:140`）、`remove`（`:151-160`）。
  - `C/store/views/view-changes.ts`：`remove`（`:99-108`，两种视图，`kind: "view"`）。
  - `C/store/project/project.store.ts`：`archiveProject` 的确认（`:328-338`）、`deleteProject` 与 `leaveProject` 经私有的 `forget`（`:603-607`，`:321-325`、`:375-382` 都走它）。
  - 改名（`renameTarget`）：迭代的 `update`（`cycle-changes.ts:100-109`）；模块的 `changed`（`module-changes.ts:216-223`，`update` `:124-125` 与 `changeMembers` `:131` 都经它：写进去的是 nerve 回答里的名字，成员的改动写同一个名字，无害）；视图的 `update`（`view-changes.ts:84-92`，两种视图，带 `logo_props`）。项目不必：它的改动经 `details.receive`，推进时刻（§2.3）。
- **怎样找到收藏 store**（v0 7.7“store 只经自己的 RootStore 找兄弟”）：三个 changes 今天只收 `(held, service)`（`cycle-changes.ts:83-86`、`module-changes.ts:104-107`、`view-changes.ts:61-64`），各加第三个构造参数——一个窄口 `TargetFavorites<E> = { forget(entity: E): void; rename(entity: E): void }`（类型在新 `C/store/favorites/target-favorites.ts`，T6 建，第一个读者是迭代与模块的 changes；连带与改名收成一个口），由各自的 store 在构造时给一个对象，它的两个方法**调用时才读** `rootStore.favorites`（迭代、模块的 `rename` 给 `{ name }`——这两种类型没有 `logo_props`；视图给 `{ name, logo_props }`；先例 `view.store.ts:71-72` 的 `workspaceOf`；`RootStore` 先建项目、规划、视图，后建收藏，`C/store/root.store.ts:90-99`）；工作区 id 取被确认的实体自己的 `workspace_id`（`Cycle`、`Module`、`View`、`Project` 都有）。项目 store 已持 `this.rootStore`（`project.store.ts:117`、`:144`），在 `forget` 与归档的确认里直接调。
- **测试的根**（§3 补记 4）：store 的单测以 `fakeRoot` 给兄弟，读了没给的兄弟即抛（`C/store/fake-root.ts:11-19`），或以普通对象冒充根（`C/store/views/fake-views.ts:63`、`:85`）：调用方的这些测试走到删除、归档、离开就会读 `root.favorites`。`projectTab`（`C/store/project/fake-projects.ts:257-277`）以它的 `api` 建一个真收藏 store，给进项目根的 `fakeRoot`（`:266-269`）并返回它；`listTab`（`C/store/fake-project-lists.ts:78-85`，建在 `projectTab` 之上，`:79`）把**同一个**给进自己的根（`:83`）——项目的连带读 `projectTab` 的根，迭代与模块的读 `listTab` 的根，两处给同一个，连带的格才看得到同一份列表（re-98 o2）；`viewsTab`/`workspaceViewsTab`（`fake-views.ts:56-91`）与 `C/store/views/view.store.workspace.test.ts:66` 的根各追加一个以同一个 `FakeNerve` 建的。只追加；连带与改名的格读它。另一个项目根：`C/store/user/fake-permissions.ts:35-38` 的 `permissionsOver` 自己建 `ProjectRootStore`，根里没有收藏 store；今天没有测试经它调离开、归档、删除，T5 若有名单之外的红，先看这里（re-98 o2）。
- 别的标签页、别人、agent 的删除与归档：下一次读收藏时消失（`useSessionSWR` 不在聚焦时重取，`C/lib/use-session-swr.ts:26`）。
- **加入项目是反方向**：离开时服务端留着行、读时过滤；本标签页再加入（`project.store.ts:357`）之后它们在服务端又可见，本页到下一次读收藏才出现，其间项目卡片的星显示“未收藏”，按下得 200、`upserted` 接得住（re-98 o1）。不另加读。

### 2.8 最近访问的数据层与记录（R11，L!）

- **service**（新 `C/services/recent-visits/recent-visits.service.ts`）：`record(body: RecentVisitCreate)`（`POST /api/v0/me/recent-visits`，204）、`list(slug, kind)`（`GET /api/v0/workspaces/{slug}/recent-visits`，`kind` 不是 `all` 时带 `?entity_type=`）。
- **store**（新 `C/store/recent-visits/recent-visit.store.ts`；`RootStore.recentVisits`）：`lists = new ReconciledByKey<RecentVisit[]>()`，键 `JSON.stringify([workspaceId, kind])`（`kind` ∈ `all`、`issue`、`project`）；`fetchRecentVisits(workspace, kind)`；`getRecentVisits(workspaceId, kind)`；`record(body)` 经一个 `oneAtATime()` 队列（服务端收到的次序即进入的次序），失败照常拒绝（store 不吞）；不把记录写进列表（首页每次挂载都读，§2.9）。
- **记录点**（两个，各一次；名字里没有 “sentry”）：
  1. **进入项目**：新 `C/components/recent-visits/project-visit.tsx` 的 `ProjectVisit({ projectId })`，挂在 `PROJ/projects/(detail)/[projectId]/layout.tsx` 的 `<ProjectAuthWrapper>` 里、`<Outlet />` 之前（`:57-59`）：包装层只在调用者是未归档项目的有效成员时渲染子树（`C/layouts/auth-layout/project-wrapper.tsx:45-64`；三种角色都算，服务端三种角色都可记），所以非成员、已归档、读失败都不记；布局以 `projectId` 为 key（`:36`），同一个项目里换页（工作项、迭代、模块、视图、收集箱五类页，`W/app/routes/core.ts:106-175`）不重挂、不再记，进别的项目（含后退、前进）重挂、记一次。挂载时（`useEffect`）记 `{entity_type: "project", entity_id}`；依赖是最近访问 store 与 `projectId`：换会话（新的 `RootStore`）算新的一次进入，再记一次，新账号的访问记给新账号（R29，§3 补记 9）。
  2. **打开工作项页**：`PROJ/browse/[workItem]/layout.tsx` 的 `IssueRouteVisit`（`:54-104`）在自己的查找答到、有工作项、不在收集箱时（`!pending && !error && data && !inIntake(data)`，`:70-85`）记一次 `{entity_type: "issue", entity_id: data.id}`；一次 `IssueRouteVisit` 的挂载一次（它以 `[slug, workItem, identity, loginId]` 为 key，`:35-43`：后退、前进、换会话即新的一次；权限水位变了重挂也再记一次，服务端同一目标一行、只更新时间）。收集箱里的条目转去收集箱页（`:82-89`），不记。
- **不记**：预览（M4/P5b 的共用件，九处宿主）、通知页的预览、首页的预览、项目设置（`(settings)` 布局）、归档目录（在项目布局之外，`W/app/routes/core.ts:177-208`）、收集箱页里选中条目、按 id 的工作项地址的工作项一半（`PROJ/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page.tsx:31-44` 读到之后 `replace` 到工作项页，由那一页记；它在项目布局里，所以照第 1 条记项目一次，§3 补记 10）、工作区的页。
- **失败静默**：记录点经新 `C/components/recent-visits/use-record-visit.ts` 发出，接住一切失败（含 `SessionChangedError`）：不提示、不 `console.*`（故事以 `expectQuietConsole` 核）；前端不节流（服务端一个目标一行、重访只改时间，`NAV:372-410`）。

### 2.9 首页的“最近”（R13，L!）

- **位置**（re-98 m5）：`C/components/home/home-body.tsx` 照旧按 `loader` 画 `HomeLoader` 或 `NoProjectsEmptyState`；新的 `RecentsWidget`（`C/components/home/widgets/recents/`）画在它们之后，**不看 `loader`**。`loader` 出自项目 store 的“当前工作区”（router store 的，`C/store/project/project.store.ts:151-171`），比路由晚一次提交：放在它之后的小部件，在应用内换到一个项目列表还没读过的工作区时会挂、卸、再挂，读两次（去重窗为 0），也间接读了 router store 的地址。小部件只要路由的工作区与自己的读；行要的角色与成员在点击、渲染时从 store 取（可观察）。三种角色都有。
- **读**（新 `widgets/recents/use-recents-read.ts`）：`useSessionSWR(workspace ? ["RECENT_VISITS", workspace.id, workspace.slug, kind] : null, (id, slug, k) => fetchRecentVisits({ id, slug }, k), { dedupingInterval: 0 })`：每次挂载都读（重访之后回首页要见到它；默认的 2 秒去重会让读的次数随墙上时钟，P6b 的教训）；筛选的状态只在这次进入（`useState`，Plane 同）。swr 名单收入它。
- **显示**（按这次的筛选 `kind`）：持有这个键的列表 → 行（读在途时、之后的重读失败时都照旧显示持有的，re-98 m3）；没有持有、读在途 → 骨架；没有持有而读失败 → 一行 `home.recents.failed` 与重试（`common.retry`，`mutate()`）；列表为空 → 按筛选的空态（`home.recents.empty.issue`、`.project`、`.default`，Plane 的文字）。迟到的回答只写它自己的键：换了筛选、换了工作区、换了会话之后答到的，不出现在当前的列表里。
- **标题与筛选**：标题 `home.recents.title`（“Recents”）是 `h3`：首页唯一的 `h2` 是问候（`E/stories/smoke/s7-chinese.spec.ts` 按 `heading level 2` 断言它，§3 补记 7）；筛选是一个菜单按钮（`CustomMenu`，照 Plane `PW/core/components/home/widgets/recents/filters.tsx`），按钮以标题为前缀得名（`aria-labelledby` 指向标题与当前选项，读作 “Recents All”）——只叫 “All”“Projects” 不说它筛的是什么，选到 “Projects” 时还与侧栏的 “Projects” 按钮同名（`C/components/workspace/sidebar/projects-list.tsx:136-142`，re-98 m11 ①）；选项“全部”`home.recents.filters.all`、“工作项”`.issues`、“项目”`.projects`，键盘可达。
- **工作项行**（照 Plane `recents/issue.tsx`，用 `ListItem`）：标识 `{project.identifier}-{issue.sequence_id}` 与名字；`calculateTimeAgo(visited_at)`；状态图标与提示（状态 store 给得出 `state_id` 时；给不出——已归档项目里的工作项，工作区的状态目录只有未归档的成员项目，`C/layouts/auth-layout/use-workspace-fetch.ts:49-51`——不画，§3 补记 6）；优先级图标；负责人头像（只读）。链接 `/{slug}/browse/{identifier}-{sequence_id}`。点击（`ListItem` 的链接是 `ControlLink`：Ctrl/⌘ 加左键照浏览器的默认，其余左键先 `preventDefault` 再调行的回调，`PKG/ui/src/control-link/control-link.tsx:26-32`）：项目 store 给得出这个项目、且它对调用者 `joinedAndUnarchived`（`C/lib/project-scope.ts:10`）→ 打开预览（§2.10）；否则（已归档、不是成员、给不出）→ 回调里照链接导航。不以“有没有项目角色”判：角色的 getter 不看归档（`C/store/user/permissions.store.ts:68-75`），项目页读过归档列表（`E/fixtures/mounts.ts:86`）或本标签页刚归档之后，项目 store 给得出已归档的项目（`project.store.ts:211-235`），那时会开预览（re-98 m4 ①）。Ctrl/⌘ 与中键照链接（新标签页）；不另说 Shift（它到不了默认动作，re-98 m4 ②）。
- **项目行**（照 Plane `recents/project.tsx`）：图标、标识、名字、几分钟前；成员头像只给项目 store 给得出的项目（`member_ids`；已归档的不在列表里，不显示，P3d PE-4）。链接 `/{slug}/projects/{id}/issues`。
- 首页的读是 GET，不写；首页不记最近访问。

### 2.10 首页的预览（R14，L!）

- 新 `C/components/home/widgets/recents/home-peek.tsx`：挂在 `RecentsWidget` 里，`<IssuePeekOverview hostScope={JSON.stringify(["home", workspaceSlug])} />`（`hostScope` 是共用件要的，`C/components/issues/peek-overview/root.tsx:32-35`）；预览打开、是这个工作区的时，以 `useProjectLabelsRead(peekIssue.projectId)` 读那个项目的标签（首页不在项目的页里，照通知页的预览 `C/components/workspace-notifications/root.tsx:161`、`C/hooks/use-project-labels-read.ts:13-16`）。共用件只在调用者在项目里有角色时读（`peek-overview/root.tsx:38-44`）；工作项行开预览的条件（§2.9，`joinedAndUnarchived`）蕴含它，所以开了的预览都读得到。不改 M4/P5b 的预览与 P5b 的读法。预览不记最近访问（R11）。

### 2.11 删除与读者（R15）

每一项在删之前 `git grep -n` 核读者（测试、替身、`K` 的样例都算），写进台账；删掉最后一个读者的 Task 同一提交删去它。

| 删除 | 最后的读者（基线） | 何时 |
|---|---|---|
| `C/services/favorite/{favorite.service.ts,index.ts}`（43、7；旧地址 `:13`、`:21`、`:29`、`:37`） | 旧收藏 store | T3（同一提交加 `old-favorite-urls`） |
| `C/store/favorite.store.ts`（309）、`C/hooks/store/use-favorite.ts`（16）、`RootStore.favorite`（`C/store/root.store.ts:23-24`、`:67`、`:99`） | `PROJ/sidebar.tsx`、`SB/*`、`use-favorite-item-details.tsx`、`SB/favorite-folder.test.tsx:10` 的替身 | T3 |
| `IFavorite`、`TFavoriteEntityType`（`PKG/types/src/favorite/{favorite.ts,index.ts}`，`PKG/types/src/index.ts:12`） | 旧 store、旧 service、`FAVORITE_ITEM_LINKS`、`SB/*`、`use-favorite-item-details.tsx` 与它的测试、`SB/favorite-folder.test.tsx` | T3（改用生成的 `Favorite`、`FavoriteKind`，`HO/M1-P3-trim-platform.md:15`） |
| `FAVORITE_ITEM_LINKS`（`PKG/constants/src/sidebar-favorites.ts`，桶导出 `PKG/constants/src/index.ts:25`）、`SB/favorite-items/common/helper.tsx`（19） | `helper.tsx` | T3（链接进 `SB/favorite-link.ts`） |
| `SB/new-fav-folder.tsx`（163）、`folder_already_exists` 键（en、zh-CN） | `SB/favorites-menu.tsx`、`SB/favorite-folder.tsx` | T3（代以 `SB/favorite-folder-form.tsx`） |
| 两个同名开合按钮的键 `aria_labels.projects_sidebar.{open,close}_favorites_menu`，文件夹的 `{open,close}_folder` | `SB/favorites-menu.tsx`、`SB/favorite-folder.tsx:256` | T3（读者核对之后；开合按钮改以名字为名） |
| 旧的拖放接线（`SB/favorites-menu.tsx:71-109` 的 `handleDrop`，`SB/favorite-folder.tsx`、`SB/favorite-items/root.tsx` 里对旧 store 的移动） | 旧 store 的 `moveFavoriteToFolder`、`removeFromFavoriteFolder`、`reOrderFavorite` | T3 删、T4 照 §2.4 重接（T3–T4 的窗口：侧栏不能拖，plan） |
| `SB/favorites.helpers.ts`（72；引 `IFavorite`，`:8`、`:60`） | 旧的拖放接线（`SB/favorites-menu.tsx:34-35`、`SB/favorite-folder.tsx:44`、`SB/favorite-items/root.tsx:27`） | T3 删、T4 新建（re-98 I4） |
| `SB/favorite-items/common/favorite-item-drag-handle.tsx`（46）与桶导出 `common/index.ts:7`，文件夹一行的手柄 | `SB/favorite-items/root.tsx:28`、`:131`；`SB/favorite-folder.tsx:187` | T3 删、T4 新建（T3 的行不画手柄） |
| `failed_to_move_favorite`、`failed_to_reorder_favorite` 键（en `common.json:234`、`:244`） | `SB/favorites-menu.tsx:66`、`:137`、`:149`，随拖放接线走掉 | T3（re-98 m8 ③；T4 的失败经 `useRefusalToast`） |
| `no_favorites_yet` 键（en `common.json:236`） | `SB/favorites-menu.tsx:260`（空的区不显示之后没有读者） | T3（re-98 m8 ①） |
| 死成员 7 行（`P6aR:187`：`favorite.store.ts:16-17`、`new-fav-folder.tsx:24-27`、`favorite.ts:17`） | — | T3（随文件） |

- `IS_FAVORITE_MENU_OPEN` 留在 `PKG/constants/src/workspace.ts:121`（键不变，R8）。
- `favorite_updated_successfully`（en `common.json:239`）留着：改名成功的提示（§2.3，re-98 m8 ②）。
- `tools/i18n-keys.mjs` 只查“读了没定义的”（`:1-14`），没人读的键它不报：键的删除只靠读者核对。
- `PKG/propel/src/icons/favorite-folder-icon.tsx` 留着（新侧栏照用）。
- pragmatic 的类型（`InstructionType`、`IPragmaticPayloadLocation`、`TDropTarget`，`PKG/types/src/pragmatic.ts`）有标签的读者（`C/components/labels/*`），留着。

### 2.12 守卫与告警（R15、R25）

- **`old-favorite-urls`**（新规则，紧接 `old-view-urls` `K:3285` 之后，phase `M7/P7`）：文件 `^web/.*\.(?:[cm]?[jt]sx?|json)$`；内容以 `/api/workspaces/` 锚定（照 `old-view-urls`、`old-cycle-urls` 的写法，`K:3072`、`K:3285`）：`/api/workspaces/(?:[^/"'`\s]+/)+(?:user-favorites|user-favorite-cycles|recent-visits)(?![\w-])[^"'`\s]*`；`why` 写明一条规则管收藏与最近访问两个旧地址（都是 navigation 的个人导航，R15）。hit 样例：`/api/workspaces/${workspaceSlug}/user-favorites/`、`…/user-favorites/${favoriteId}/`、`…/user-favorites/${favoriteId}/group/`、`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-favorite-cycles/`、`/api/workspaces/${workspaceSlug}/recent-visits/`、`` `/api/workspaces/${workspaceSlug}/recent-visits/?entity_name=issue` ``；miss：`/api/v0/workspaces/{slug}/favorites`、`/api/v0/favorites/{favorite_id}`、`/api/v0/me/recent-visits`、`/api/v0/workspaces/{slug}/recent-visits`、`/api/workspaces/${workspaceSlug}/user-favorite-projects/`（`plane-workspace-urls` 的，`K:2053`）、`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-favorite-modules/`（`old-module-urls` 的，`K:3131`）；文件 hit 写 `web/apps/web/core/services/favorite/favorite.service.ts`（只是路径）。与旧 service 同一提交（T3）。今天没有规则拦这三种（以 Python 正则对照过）。
- **phase**：`K:2` 由 `M7/P6` 改为 `M7/P7`（T1）；不延长任何 `until`。
- **地址规则**：P7 新写、改写的读工作区的文件收入 `workspace-address-route`（`K:2847`）的名单并带 hit 样例（`PROJ/sidebar.tsx` 与首页 `PROJ/page.tsx` 已在）；读项目的收入 `project-address-route`（`K:2334`；迭代与模块的行、命令、视图的行已在）。工作区级的角色一律经 `getWorkspaceRoleByWorkspaceSlug`（`C/store/user/permissions.store.ts:22`、`:56`），不经不带 slug 的 `allowPermissions`。
- **swr 名单**（`.oxlintrc.json:121-223`）：收入 `C/hooks/use-favorites-read.ts`、`C/components/home/widgets/recents/use-recents-read.ts`；删掉的文件移出（基线上收藏的文件都不在名单里）。
- **oxlint**：web 的上限 14（`W/package.json:10`）是等式；新文件零告警；M3 交接说 `SB/favorite-folder.tsx` 有 2 条 `no-shadow`（`HO/M3-closeout.md:109`），调研实跑基线为 0：T3 照实跑、照实写（改写之后仍为 0）。
- **文案**：P7 动到或新建的文件里的字面量都经 `t()`（M3 交接 `HO/M3-closeout.md:104`：今天 `SB/favorite-folder.tsx:239`、`:245`、`SB/favorite-items/common/favorite-item-quick-action.tsx:52`、`favorite-item-drag-handle.tsx:29` 写死英文）。新键（en 与 zh-CN 同一提交，写完整的字面量，不许一个键是另一个键的前缀）：
  - `common.json`：`favorite_star.add`（“Add {name} to favorites”）、`favorite_star.remove`（“Remove {name} from favorites”）、`favorite_item.remove`（“Remove from favorites”）、`favorite_folder.rename`（“Rename folder”）、`favorite_folder.remove`（“Remove folder and its favorites”）、`favorite_folder.name_too_long`（“Folder name can have at most 255 characters”）、`favorite_list.load_failed`（“Favorites could not be loaded”）。
  - `power-k.json`：`power_k.contextual_actions.{cycle,module}.{add_to_favorites,remove_from_favorites}`（“Add to favorites”、“Remove from favorites”，Plane 的文字）。
  - `home.json`：`home.recents.title`（“Recents”）、`home.recents.filters.{all,issues,projects}`（“All”、“Work items”、“Projects”）、`home.recents.empty.{issue,project,default}`（Plane 的三句）、`home.recents.failed`（“Recents could not be loaded”）。
  - 删的键见 §2.11。别的一律用已有的键（`favorites`、`create_folder`、`new_folder`、`folder_name_cannot_be_empty`、`favorite_{created,removed,updated}_successfully`、`common.item_options`、`common.retry`、`aria_labels.projects_sidebar.{create_favorites_folder,enter_folder_name}`、`errors.navigation_favorite_*`）。
- **iframe**：收藏与首页不嵌 iframe（终检时 `git grep -i iframe` 再核一次）。

### 2.13 e2e（R16、R18、R21，L! 的是 R16）

- **挂载清单**（`E/fixtures/mounts.ts`，97 行）：加三个常量，头注释 `:5-19` 照实改写（`:16` 的“favourites, recents”不读一句改为各自的门与页）：
  - `FAVORITES = ["GET /api/v0/workspaces/{slug}/favorites"]`：有应用侧栏的页（工作区的页里除了设置与通知），工作区的管理员与成员，挂载时一次（键 `["FAVORITES", workspace]`）；访客从不。T3 加。
  - `VISIT = ["POST /api/v0/me/recent-visits"]`：进入项目（项目布局里成员的页，三种角色）与打开工作项页（查找答到、不在收集箱）各一次。T8 加。
  - `RECENT_VISITS = ["GET /api/v0/workspaces/{slug}/recent-visits"]`：首页，三种角色，每次挂载一次（筛选“全部”不带查询）。T9 加。
- **S2**（`E/stories/smoke/s2-web-app.spec.ts`，581 行，不变长）经常量跟随：管理员、成员、项目非成员的有侧栏的页（`"/"`、`/{slug}/projects`、`archivedIssuesVisit`、`profileVisits`、`draftsVisit`、项目的工作项页）加 `...FAVORITES`（T3）；管理员、成员、访客的项目工作项页加 `...VISIT`（T8；项目非成员停在加入页，不记）；四组的 `"/"` 加 `...RECENT_VISITS`（T9）；设置页与访客的有侧栏的页不变。表之外的四个 `expectExactly`：`:511`（管理员打开已归档项目的工作项页，有侧栏的页）加 `...FAVORITES`（T3，re-98 I2 ③；不加 `VISIT`：已归档时包装层不渲染子树，`project-wrapper.tsx:61-62`）；`:497`（项目的设置页）、`:524`（引导页）、`:538`（个人设置）不变。三组相同的项目工作项页一行收成一个本地帮手，文件不变长（plan）。
- **九份按 `WORKSPACE` 断言整页请求的故事**经常量跟随（签入者都是工作区管理员或成员）：加 `...FAVORITES`（T3）的十处——`collaboration/h12-project-view-page.spec.ts:64`、`h12-project-views-list-page.spec.ts:76`、`h13-workspace-list-page.spec.ts:59`、`h13-workspace-view-page.spec.ts:103`、`h14-workspace-views-list-page.spec.ts:68`、`h9-intake-list-page.spec.ts:86`、`planning/g13-archive-delete-module-page.spec.ts:77`、`:298`、`g3-change-cycle-page.spec.ts:270`、`g8-archive-cycle-page.spec.ts:89`；其中在项目布局里的再加 `...VISIT`（T8）——H12 两份、H9、G13 `:298`、G3（G8 `:89`、G13 `:77` 是归档目录，在项目布局之外；H13、H14 是工作区的页）。`h14-workspace-views-list-page.spec.ts:120` 是“包含”的核对、`h2-h4-notification-cards-page.spec.ts:240` 与 `h5-notification-list-page.spec.ts:96` 在通知页（没有侧栏）、`project/p2-visibility.spec.ts:129` 只用 `PROJECT_MEMBER`：都不改。
- **“某一步之后没有写”的八处**（R16，逐个核过；re-98 I2 补了后两处，其中一处在 `e2e/acceptance`，只由 m6-acceptance 跑——默认配置的 `testDir` 是 `stories`，`E/playwright.config.ts:9`；受影响表以整个 `e2e/` 上的 `git grep` 为底，`stories`、`acceptance`、`drills`、`fixtures` 都算）：
  - `issue/cleanup-interactions.spec.ts:231`：后退回到项目的列表（`:228-229`），项目布局重挂、记一次，记录必在切片里 → 过滤式排除 `VISIT`（一行、写理由）。
  - `issue/collaboration-interactions.spec.ts:696`、`issue/detail-interactions.spec.ts:98`：切片在页面打开之后开始（`:615`、`:47`），而工作项页的记录在查找答到的那一次提交的 effect 里发出，标题可见并不排在它之后 → 排除 `VISIT`，免得竞态。
  - `issue/i11-comments-reactions.spec.ts:312`、`:315`：`:305` 的重载让工作项页再记一次，快照 `:312` 可能早于它 → 两处的过滤都排除 `VISIT`。
  - `collaboration/h12-project-view-page.spec.ts:125`：整个会话的写，项目布局记过一次（之后同一项目里换页不再记）→ 期望 `[...VISIT, "POST /api/v0/auth/refresh"].toSorted()`。
  - `collaboration/h20-view-pages.spec.ts:351`：`recorded` 自 `:294` 进入项目之前开始 → 期望加 `...VISIT`。
  - `collaboration/h12-project-view-saving-page.spec.ts:104`：整段会话里合 `/^(PATCH|POST|DELETE) \/api\/v0\/(views|me)\//` 的写须为空，`watch` 在进视图页（项目布局里）之前开始（`:89-90`），`POST /api/v0/me/recent-visits` 合这个正则 → 过滤里排除 `VISIT`（同 `cleanup-interactions` 的写法）。
  - `e2e/acceptance/g19-m3-dialogs-and-filters.spec.ts:146-149`：除刷新之外的非 GET 须为空，`:107` 进迭代列表页、项目布局记一次 → 过滤里排除 `VISIT`。
  - 核过不受影响、不改：`h13-workspace-list-page.spec.ts:131`、`:210`（工作区的页，不进项目布局）；`planning/g15-cycle-list-page.spec.ts:330`、`g15-module-list-page.spec.ts:342`（切片从列出行之后开始，项目的记录在列表的读答到之前已发出）；`collaboration/h1-subscription-page.spec.ts:52-59`（切片从按钮显示关注状态之后开始，那要另一个读答到）。
- **`writesSeen` 一族**（调研没列到，§3 补记 3；定法 R26）：`E/fixtures/intake-pages.ts:293-305` 的 `writesSeen` 在进入项目之前开始记（每个调用都在 `page.goto` 收集箱或项目页之前），十余处断言写的全表或按位置读（`h8-intake-submit-page.spec.ts:198`、`:242`、`:295`，`h11-intake-content-page.spec.ts:145`、`:251`、`:321`、`:377`，`h11-intake-links-assignees-page.spec.ts:141`、`:197`、`:215`、`:219`，`h11-intake-attachments-page.spec.ts:62`、`:91`、`:122`（`.slice(2)`）、`:208`，`h11-intake-images-page.spec.ts:69`、`:94`、`:125`（`.slice(3)`）、`:161`、`:183`，`wrote` 的 `writes[at]`，`intake-pages.ts:319-321`）。定法（R26）：`writesSeen` 像排除会话的刷新（`/api/v0/auth/`）那样排除记录访问的写，只排恰好 `POST /api/v0/me/recent-visits` 一种（方法与路径都对上；`intake-pages.ts:302` 的条件一行，注释写 “M7/P7 spec”）；五份收集箱故事的二十来处断言不动；`watchPage` 与 `recordApi` 不过滤（R16：页面发了什么的记录仍是全的）。记录访问本身由 H17 (page) 断言（`rulings.md` 的 E-26 末句写作 H16，主控在 re-98 §6 改为 H17 (page)）。
- **页面故事**（R18，T10–T13；`E/stories/collaboration/`，名字合 m7-pages 的 `testMatch`（`-page.spec.ts`/`-pages.spec.ts`）与 `grep`（标题带 `H15 (page)`、`H16 (page)`、`H17 (page)` 或 `H20 (page)`），`E/playwright.m7-pages.config.ts:10-11`；三引擎；CI 以默认配置在 chromium 跑一次；与已有的接口故事 `h15-favorites.spec.ts`、`h16-favorite-cascades.spec.ts`、`h17-recent-visits.spec.ts` 不同名）：

| 文件 | 标题 | 覆盖 |
|---|---|---|
| `h15-favorites-sidebar-page.spec.ts` | `H15 (page)` | 管理员：没有收藏时没有收藏区；经接口建的收藏重载之后按 `sequence` 降序列出、开合存浏览器（重载之后还开着）；名字是 nerve 的（另一会话改了视图名、重载之后跟上）；工作区视图的链接到 `/{slug}/workspace-views/{id}`；文件夹：空白与超长在字段下、不发请求，与已有的只差大小写的名字 409 在字段下、输入留着，成功之后在最上、库里一行；改名只发 `{name}`、没变不发；连同里面的收藏移出（一个 `DELETE`、子项一起没了、库里软删）；行菜单移出；拖动（`dragOnto`）：D40 的反例（C 放到 A 之下，体恰为 `{sequence: 29500}`）、进文件夹（体恰为 `{parent_id}`）、经页头移出（体恰为 `{parent_id: null}`）、文件夹拖进文件夹不发；成员同样；访客没有收藏区、整页没有 `FAVORITES` 的读 |
| `h15-favorite-stars-page.spec.ts` | `H15 (page)` | 星：项目卡片（成员的未归档项目有，公开项目的非成员与已归档项目没有）、迭代行、模块行与卡片、项目视图行、工作区视图行（默认四行没有）；以行为名、`aria-pressed`；扣住回答时停用、放开之后翻转；请求体恰为 `{entity_type, entity_id}`；加入之后收藏区打开、行出现；另一会话已收藏同一目标时按下得 200、侧栏不重复；命令面板：迭代页、模块页以 `shift+f` 加入与移出；项目访客兼工作区成员有星；访客到处没有星与命令 |
| `h16-favorite-cascades-page.spec.ts` | `H16 (page)` | 本标签页删迭代、归档模块、删项目视图、删工作区视图、归档项目、离开项目之后，侧栏立刻没有它（这一页没有再读收藏）；本标签页改迭代、模块、视图的名之后侧栏立刻是新名（同样没有再读，re-98 I1）；另一个账户（收藏过同一目标）重载之后也没有；恢复不加回；被移出项目的看不到 |
| `h17-recent-visits-page.spec.ts` | `H17 (page)` | 进入项目恰一个 `VISIT`，体恰为 `{entity_type: "project", entity_id}`；同一项目里换页不记；进别的项目、后退、前进各一个；工作项页一个 `{entity_type: "issue"}`；预览、收集箱的选中、项目设置、归档目录不记；首页：两种行、三种筛选（查询恰为 `entity_type=issue` 等）、按时间、同一目标一行（重访移到最上）、删除的工作项不在、已归档项目里的工作项没有状态图标且点它走链接（先去过项目页、读过归档列表之后也是，re-98 m4）；应用内换到另一个工作区时首页的读恰一次（m5）；访客也有；首页的读不写；记录被拒（只在 Chromium、目标页代码载入之后装路由）时没有提示、控制台安静（Chromium 自己记的 “Failed to load resource” 照 `planning/g3-change-cycle-page.spec.ts:281` 写进 `expectQuietConsole` 的预期，不算放宽，re-98 o9；应用自己不出声由单测钉）；首页的读失败与重试（同一条件） |
| `h20-favorites-home-pages.spec.ts` | `H20 (page)` | 键盘与读屏：收藏区的开合（`aria-expanded`）、“新建文件夹”有焦点时可见、文件夹输入的 Escape（在途时不取消）、行菜单以行名为名（Enter 打开、Escape 之后焦点回按钮）、文件夹以 Space 开合、星可以 Tab 到、Enter 翻转（之后焦点仍在星上，三引擎各记，re-98 m11 ②）；首页筛选以键盘选，筛选按钮以标题为前缀得名；迟到的回答：换工作区之后答到的收藏读不写进新工作区的侧栏，另一个标签页换了账号之后答到的星不在新会话里提示，换了筛选之后答到的“最近”不出现在当前筛选下；800 宽一次（星与菜单可达） |

- **故事的写法**：每个“在途”都靠扣住回答（`holdAnswer`），不靠时间，不断言墙上的时钟；一律放行或拒绝的路由只在 Chromium 装，而且只在目标页的代码载入之后装；换页、离开、换工作区、改服务端的状态之前先等在途的读答完；数收藏的读只在整页载入时数（侧栏在应用内导航时不重挂，去重窗让应用内的次数随时钟，断言种类）；拖放用 `dragOnto`（`E/fixtures/project-pages.ts:83-99`），三引擎都跑；侧栏在主栏与浮出栏各有一份，定位限在 “Main sidebar”；每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去）与 `expectQuietConsole`（不放宽）结束。
- **夹具**：新 `E/fixtures/favorite-pages.ts`（侧栏的区、行、文件夹、菜单、星、拖放的落点，收藏的读与写的路径）与 `E/fixtures/home-pages.ts`（“最近”的行、筛选、预览）；前置条件只经接口（`E/fixtures/favorites.ts` 的 `createFavorite` `:17`、`favorViewInFolder` `:35`、`favoriteRows` `:116`；`E/fixtures/recent-visits.ts` 的 `recordVisit` `:18`、`recentVisitRows` `:75`）。`cycle-pages.ts`、`view-pages.ts`、`intake-pages.ts`（除 `writesSeen` 的一行）不长。
- **检查点与终检的页面配置**（R21）：P6b 的六个（m7-pages、m6-pages、m6-acceptance、m4-list、m4-filters、m4-detail）加 m4-collaboration（I11、I13、C6–C9 打开工作项页与预览）与 m5-content（F 系列打开工作项页），共八个；本机 chromium 与 firefox 一起、WebKit 单独 `--workers=2`（负责人 2026-11-28）。第 1 路以 P7 检查点 0 的本机值为基线、+30 秒停查；CI 的八成、web 960 秒、e2e 30 分钟只报告（R23）。
- **时长**：每份不过 400 行（既有的 `h9-intake-list-page.spec.ts` 398 是写明的例外，上限 410：T3、T8 的两个常量让导入过 oxfmt 的 120 字而折行，re-98 m7）；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

### 2.14 文件与职责

| 文件 | 职责 | Task |
|---|---|---|
| `C/services/favorites/favorites.service.ts`、`C/store/favorites/{favorite.store,favorite-changes,favorite-tree,favorite-place}.ts`（新）与测试；`C/store/root.store.ts`（加 `favorites`）、`root.store.test.ts` | 收藏的数据层 | T2 |
| `PROJ/sidebar.tsx`；`SB/*` 改写（`favorites-menu.tsx`、`favorite-folder.tsx`、`favorite-items/*`、新 `favorite-folder-form.tsx`、`favorite-link.ts`）；`C/hooks/use-favorite-item-details.tsx` 与测试；`C/hooks/store/use-favorites.ts`、`C/hooks/use-favorites-read.ts`、`C/lib/favorites-gate.ts`（新）；四个 store 的 `answeredAtOf`；§2.11 的 T3 删除（含 `SB/favorites.helpers.ts` 与手柄）；`K`、`.oxlintrc.json`；en/zh-CN；`E/fixtures/mounts.ts`（`FAVORITES`）、S2、九份故事 | 侧栏、读与门、旧层删除 | T3 |
| `SB/favorites.helpers.ts`、`SB/favorite-items/common/favorite-item-drag-handle.tsx`（新建）；`SB/favorites-menu.tsx`、`SB/favorite-folder.tsx`、`SB/favorite-items/root.tsx`（拖放） | 拖放 | T4 |
| `C/components/favorites/{favorite-star.tsx,use-favorite-toggle.ts}`、`C/hooks/use-favorites-offered.ts`（新）；`C/components/project/card.tsx`；`C/store/project/project.store.ts`（连带）；`C/store/project/fake-projects.ts`（根）；`card.test.tsx`（替身） | 项目的星与连带 | T5 |
| `cycle-list-item-action.tsx`、`module-list-item-action.tsx`、`module-card-item.tsx`；`…/context-based/use-favorite-command.ts`（新）、`cycle/commands.ts`、`module/commands.tsx`；`cycle-changes.ts`、`module-changes.ts`、`cycle.store.ts`、`module.store.ts`、`C/store/favorites/target-favorites.ts`（新：连带与改名的口，第一个读者是这两个 changes）；`C/store/fake-project-lists.ts`（根）；模块的“收藏”筛选（`module_filter.store.ts`、`dropdowns/filters/root.tsx`、`applied-filters/root.tsx`、`modules-list-view.tsx`、`PROJ/projects/(detail)/[projectId]/modules/(list)/page.tsx`、`module-view-header.tsx`）；翻转的格与替身 | 迭代、模块的星、命令、连带、改名的跟随与筛选 | T6 |
| `view-list-item-action.tsx`；`view-changes.ts`、`view.store.ts`；`fake-views.ts` 与 `view.store.workspace.test.ts`（根）；项目视图列表的“收藏”筛选（`view-list-order.ts`、`filters/filter-selection.tsx`、`applied-filters/root.tsx`、`views-list.tsx`、`PROJ/projects/(detail)/[projectId]/views/(list)/page.tsx`） | 视图的星、连带、改名的跟随与筛选 | T7 |
| `C/services/recent-visits/recent-visits.service.ts`、`C/store/recent-visits/recent-visit.store.ts`、`C/hooks/store/use-recent-visits.ts`、`C/components/recent-visits/{project-visit.tsx,use-record-visit.ts}`（新）；`[projectId]/layout.tsx`、`browse/[workItem]/layout.tsx`；`root.store.ts`、`root.store.test.ts`；`mounts.ts`（`VISIT`）、S2、五处 `VISIT`、八处“之后没有写”、`writesSeen` | 最近访问与记录 | T8 |
| `C/components/home/home-body.tsx`、`C/components/home/widgets/recents/*`（新）；`home.json`；`mounts.ts`（`RECENT_VISITS`）、S2 的 `"/"` | 首页的“最近”与预览 | T9 |
| `E/fixtures/favorite-pages.ts`、`h15-favorites-sidebar-page.spec.ts`、`h15-favorite-stars-page.spec.ts` | H15 (page) | T10 |
| `h16-favorite-cascades-page.spec.ts` | H16 (page) | T11 |
| `E/fixtures/home-pages.ts`、`h17-recent-visits-page.spec.ts` | H17 (page) | T12 |
| `h20-favorites-home-pages.spec.ts` | H20 (page) | T13 |

## 3. 局部裁定

主控的裁定在台账 `rulings.md`（E-1–E-29；E-26–E-29 是起草之后的裁定），本节逐条写成 R1–R29：**R-n 即 E-n**。预检 re-98 的处理（主控，re-98 §6）折进各条，本节末逐条列出。十条是**高风险的局部裁定**（**L!**，动到 M3、M4、M6 或 M7 前段（P4–P6b）已合进 main 的共用代码、共用测试与故事），阶段记录写明、收官报告列出：R2、R7、R9、R10、R11、R12、R13、R14、R16、R17。没有负责人的裁定（契约、服务端、部署与 CI 结构都不改）。

| E | R | E | R | E | R | E | R | E | R |
|---|---|---|---|---|---|---|---|---|---|
| E-1 | R1 | E-6 | R6 | E-11 | R11（L!） | E-16 | R16（L!） | E-21 | R21 |
| E-2 | R2（L!） | E-7 | R7（L!） | E-12 | R12（L!） | E-17 | R17（L!） | E-22 | R22 |
| E-3 | R3 | E-8 | R8 | E-13 | R13（L!） | E-18 | R18 | E-23 | R23 |
| E-4 | R4 | E-9 | R9（L!） | E-14 | R14（L!） | E-19 | R19 | E-24 | R24 |
| E-5 | R5 | E-10 | R10（L!） | E-15 | R15 | E-20 | R20 | E-25 | R25 |
| E-26 | R26 | E-27 | R27 | E-28 | R28 | E-29 | R29 | | |

- **R1 一个收藏 store，侧栏的树与星的“按目标查”都由它派生；新 service；旧 store、service、手写类型全换**（E-1）：§2.1。理由：一份列表、一个确认点，侧栏与五处星天然一致（资源不带 `is_favorite`，`D:127`）。代价：数据层约 250 行与 service 约 60 行、单测与变异；`RootStore` 换字段，`root.store.test.ts:105` 的数 19 → 20 → 19 → 20（T2 加、T3 删旧、T8 加最近访问）。
- **R2 侧栏读，门 `favoritesOn`，键 `["FAVORITES", 工作区 id, slug]`；访客、设置页、通知页不读**（E-2，**L!**）：§2.2。理由：只在用得上的页读，与 P5a 侧栏的待处理数同一形状。代价：S2 与九份故事的管理员、成员的有侧栏的页多一个读（§2.13）；M3 的侧栏文件。名字不含 “sentry”。
- **R3 一个工作区一个队列，体在轮到时算，不乐观，在途停用，意图幂等**（E-3）：§2.1。理由：移动的体依赖兄弟，各自一个队列时 D40 那类乱序会回来；v0 7.7。代价：一个工作区的写串行（人手操作，量小）；星按下之后等一个来回才变。
- **R4 降序放置、合成一个 `PATCH`、目标不在不发、不改 `place-between.ts`**（E-4）：§2.1、§2.4。理由：修 D40（`P3cS:161`）；`place-between.ts` 是升序的共用件（标签、项目、状态）。代价：一个约 40 行的纯函数与表格式单测。
- **R5 照 Plane 的拖放，行菜单键盘可达，不加键盘移动，故事用 `dragOnto`**（E-5）：§2.4。代价：键盘用户不能改次序与进出文件夹（同 Plane）。
- **R6 文件夹行内建与改名、等回答、`FieldError`、不在前端查重、409 在字段下、删连子项不确认、一层**（E-6）：§2.3。理由：修 D41；M3 交接的字段提示（`HO/M3-closeout.md:80`）。代价：表单改写约 120 行；删 `folder_already_exists`。
- **R7 名字与图标取 nerve 最后回答的那一份；链接移进应用、加工作区视图一支；开关关着的照列**（E-7，**L!**）：§2.3。代价：四个 store（M3 的项目、M6 的迭代与模块、P6a 的视图）各加一个 `answeredAtOf`（从 getter 的同一个“最新的一份”取，§3 补记 5），实现这些接口的替身编译跟随；P6a 的 `use-favorite-item-details.test.tsx` 改写。迭代、模块、视图的改名不推进 `answeredAt`，所以收藏 store 另给 `renameTarget`，三个 changes 在改名的确认里经与连带同一个构造参数的口调它（re-98 I1 (a)，§2.7）；不取“store 有一份就用 store 的”（旧的实体副本会盖过更新的收藏列表，违反 v0 7.7），也不改 `Reconciled.confirm` 去推进 `answeredAt`（M4、M6 共用的语义）。
- **R8 开合存浏览器、加入之后打开，改用 `@nerve/hooks` 的 `useLocalStorage`**（E-8）：§2.3。代价：一行导入；M8 交接写处理结果（副本本身仍归 M8）。
- **R9 星的位置与门：工作区管理员、成员且目标未归档，项目卡片另要项目成员；访客没有；翻转的格写进允许清单**（E-9，**L!**）：§2.5。理由：一条规则（工作区角色）同时管侧栏、星与命令；项目卡片按契约避开必败的写。代价：M3 的卡片、M6 的行、卡片与命令、P6a/P6b 的视图行；翻转的既有格（四份文件六格，§3 补记 2）；渲染这些宿主的既有测试加星的替身（plan）。
- **R10 两种视图都给星，在共用的 `ViewListItemAction`；默认四个没有**（E-10，**L!**）：§2.5。代价：P6a/P6b 的共用行组件；渲染它的视图测试按需跟随（plan）。
- **R11 进入项目与打开工作项页各记一次；预览、设置、归档、收集箱的选中不记；失败静默、不节流**（E-11，**L!**）：§2.8。理由：与设计的字面（“工作项详情”“进入项目”，`D:248`）一致，e2e 的面最小。代价：M3 的项目布局、M4/P5b 的工作项页各一处；只开预览的工作项进不了“最近”（与 Plane 不同）。
- **R12 `forgetTarget`、`forgetProject`，在 nerve 确认之后调，不加 GET**（E-12，**L!**）：§2.7。代价：三个 changes 各多一个构造参数（连带与 R7 的改名同一个口 `TargetFavorites`）、项目 store 两处；四份测试的根追加真收藏 store（§3 补记 4；`projectTab` 与 `listTab` 给同一个）；各一格单测与变异。
- **R13 首页只有“最近”：三种筛选、两种行、空态、失败与重试；已归档项目里的工作项没有状态图标**（E-13，**L!**）：§2.9。代价：小部件约 250 行；S2 的 `"/"` 四组各一个读；首页的标题不能是 `h2`（§3 补记 7）。
- **R14 首页的预览宿主读项目的标签，没有角色时退回链接**（E-14，**L!**）：§2.10。代价：宿主约 30 行，照用 M4/P5b 的预览与 P5b 的读法，不改它们。
- **R15 旧层全删，`old-favorite-urls` 锚在 `/api/workspaces/`、拦三种旧地址，新文件进两条地址规则，phase `M7/P7`，删最后读者的 Task 同一提交删它**（E-15）：§2.11、§2.12。
- **R16 `mounts.ts` 加三个常量，S2 与九份故事经常量跟随，八处“之后没有写”逐个核、按需排除，不在 `watchPage` 里过滤**（E-16，**L!**）：§2.13。代价：S2（表与表外的 `:511`）与 M3–P6b 的十余份故事，其中一份在 `e2e/acceptance`（只由 m6-acceptance 跑）；另有调研没列到的 `writesSeen` 一族（§3 补记 3、R26）。re-98 I2 补了三处（`h12-project-view-saving-page.spec.ts:104`、`acceptance/g19-m3-dialogs-and-filters.spec.ts:146-149`、`s2-web-app.spec.ts:511`）；受影响表以整个 `e2e/` 的 `git grep` 为底。
- **R17 恢复模块列表与视图列表的“收藏”筛选，并进 T6、T7，只在这一会话**（E-17，**L!**）：§2.6。代价：M6 的模块筛选 store 与菜单、P6a 的视图列表页面状态；两页“有没有筛选”的判断各收成一个、页面与页头共用（re-98 I3，§2.6），所以 M6 的模块列表页与页头、P6a 的视图列表页也动；工作区视图列表没有筛选，不加（§3 补记 8）。
- **R18 五份页面故事**（E-18）：§2.13。
- **R19 契约够**（E-19）：最近访问的行只有 `state_id`，不改契约。
- **R20 `is_favorite`、建后收藏、`favoriteProjectIds` 不加回**（E-20）：资源不带 `is_favorite`（`D:127`）；Plane 的“建后收藏”所有调用处都传 `false` 或缺省，`favoriteProjectIds` 没有读者。交接写“不适用”与证据。
- **R21 八个页面配置，WebKit 单独 `--workers=2`；第 1 路以检查点 0 为基线 +30 秒停查**（E-21）：§2.13。
- **R22 T1 回填 P6b**（E-22）：plan T1。
- **R23 照常设的报告线，越线只报告**（E-23）。
- **R24 15 个 Task，检查点在 T2、T8、T9 之后**（E-24）：plan。
- **R25 P6a、P6b 的教训进实施者简报**（E-25）：plan 的 Global Constraints 与简报。
- **R26 `writesSeen` 取 (a)**（E-26）：只排恰好 `POST /api/v0/me/recent-visits`（方法与路径都对上），一行；收集箱故事的二十来处位置断言不动；`watchPage` 不过滤；记录访问本身由 H17 (page) 断言（§2.13）。
- **R27 T2 的“成员先于读者”窗口**（E-27）：接受；读者在 T3–T7 陆续来，T7 的死成员整份比对必须为零（plan）。
- **R28 读失败的侧栏**（E-28）：没有持有而读失败时区头照显示、展开一行失败与重试（§2.3，§3 补记 13）；持有着列表时照画行（re-98 m3）。
- **R29 换账号再记一次**（E-29）：新账号的访问记给新账号（§2.8，§3 补记 9）。

**向负责人说明**（都是与 Plane 的差异，照做并登记 plane-diff，收官报告列出）：
1. **预览不记最近访问**（R11）：Plane 的预览也记（它在 GET 里记），所以只用预览的人的“最近”里没有那些工作项。
2. **星的门**（R9）：按工作区角色给（项目访客兼工作区成员也有星），项目卡片只给项目成员（Plane 给所有未归档项目，契约会 422）。
3. **工作区视图的星**（R10）：Plane 的网页收藏不了工作区视图，契约允许，P7 给。
4. **e2e 跟随的面**（R16）：比调研数的大——S2（表与表外的已归档项目一处）、九份故事的十处、五处 `VISIT`、八处“之后没有写”（其中一处在 `e2e/acceptance`，只由 m6-acceptance 跑），加上 `writesSeen` 的五份收集箱故事（R26：一行排除）与六格翻转的单测。另：P6b 最终分支 CI 的第 1 路 bootstrap 79.7%，贴着报告线；P7 不加 Go 测试，web 作业的 Unit 一步会再涨，越线只报告。

**起草时的补记**（起草者核出的与调研、裁定的出入，各自的定法）：
1. **`HO/M3-closeout.md:35-37` 不是 `helper.tsx` 的 `console.error`**（调研 §2 的表）：那几行是 `ProjectViewStore.loader` 与通知页的收集箱项。M3 交接里没有点名 `helper.tsx` 的条；`console.error` 改提示的依据是设计 `D:248`（“失败经提示而不是 `console.error`”）。
2. **“四个没有星的格”实为四份文件六格**（R9）：另有 `modules-list-view.test.tsx:246-262` 的卡片一格（`:254-259`）与 `module/commands.test.tsx:126-138` 的 `it.each`（它的标题“its link's copy alone”对兼工作区管理员的项目访客与模块关着的管理员不再成立）；`modules-list-view.test.tsx:22-24` 的说明随之改。
3. **`writesSeen` 一族**（R16）：调研只列了六处“之后没有写”；`writesSeen`（`E/fixtures/intake-pages.ts:293-305`）在进入项目之前开始记，五份收集箱故事约 20 处断言写的全表或按位置读（§2.13）。定法见 R26。
4. **测试的根会抛**（R12）：调研说连带是“各加一两行”；`fakeRoot` 读了没给的兄弟即抛（`C/store/fake-root.ts:11-19`），普通对象的根读 `favorites` 得 `undefined`、调用时抛：`listTab`、`projectTab`、`viewsTab`/`workspaceViewsTab` 与 `view.store.workspace.test.ts:66` 的根各追加一个真收藏 store（§2.7）。
5. **四个 store 都没有对外的 `answeredAt`**（R7）：调研说项目与视图“已有 `answeredAt` 的取法”，那是 getter 里私下的；四个都加 `answeredAtOf`，从同一个“最新的一份”取（§2.3）。实现 `IProjectStore`、`ICycleStore`、`IModuleStore`、`IViewStore` 的测试替身编译跟随。
6. **状态目录不含已归档项目**（R13）：P3d §5 “状态经工作区的状态目录解析”（`P3dS:191`）对已归档项目里的工作项不成立：不画状态图标。
7. **首页的标题不是 `h2`**：`E/stories/smoke/s7-chinese.spec.ts` 以 `getByRole("heading", { level: 2 })` 断言问候是唯一的 `h2`；“最近”的标题用 `h3`。
8. **工作区视图列表没有筛选**（R17）：它只有搜索（P6b R13），“收藏”筛选只在项目视图列表与模块列表。
9. **换会话再记一次**：工作项页的 `IssueRouteVisit` 本来以 `loginId` 为 key（`:35-43`），项目的记录点照同一个意思以会话的 store 为依赖：新账户此刻在这一页上，算一次进入（R29 接受）。
10. **按 id 的工作项地址在项目布局里**：它照第 1 条记项目一次，工作项由它转去的工作项页记（调研的“按 id 的地址不记”只对工作项成立）。
11. **侧栏挂两份**（R2）：主栏与浮出栏（`resizable-sidebar.tsx:184`、`:228`）；读是一个请求（同一个键），DOM 是两份，故事限在 “Main sidebar”。
12. **“收藏”筛选是单独的选项**（R17，照 Plane）：不是带标题的一节，所以既有的筛选菜单的单测（`C/components/modules/module-filters.test.tsx:139` 的 `titles()`、`views/(list)/choices.test.tsx:187-197` 的“没有匹配”的计数）不变；只为它们的 hook 替身补一行（plan）。
13. **读失败的侧栏**：调研与裁定没说；起草者定为区的页头照显示，展开时一行失败与重试（§2.3），与“失败经提示”（`D:248`）一致，不静默（R28 接受；re-98 m3 补了“持有着列表时照画行”）。

**预检 re-98 的处理**（主控全部采纳，re-98 §6；各条已改进上文）：
- **I1**（名字的较新一份给不出“本标签页改名立刻跟上”）：取 (a)，`renameTarget`，§2.1、§2.3、§2.7、R7；T2、T6、T7 的格与变异。
- **I2**（e2e 漏三处）：`h12-project-view-saving-page.spec.ts:104`、`acceptance/g19-m3-dialogs-and-filters.spec.ts:146-149`、`s2-web-app.spec.ts:511`；“六处”改“八处”；§2.13、R16。
- **I3**（只开“收藏”时已应用的一行不出现）：两页的判断各收成一个，§2.6、R17。
- **I4**（`favorites.helpers.ts` 在 T3 过不去）：T3 删、T4 新建，手柄同样，§2.4、§2.11。
- **m1** E-26–E-29 折进（R26–R29、§2.13、§8）；**m2** 404 的写丢掉那一行（§2.1）；**m3** 持有着列表时照画行、没持有不给新建文件夹（§2.3、§2.9 的“显示”照旧以持有为先）；**m4** 开预览以 `joinedAndUnarchived` 判、不说 Shift（§2.9、§2.10）；**m5** 小部件不在 `loader` 之后（§2.9）；**m6** 变异表（plan）；**m7** h9 的例外上限 410（§2.13、plan）；**m8** 四个键的去处（§2.11、§2.3）；**m9** `use-favorites-offered.ts` 在 T5（§2.14）、T2 并存窗口的同名（plan）、`move` 的原位与文件夹进文件夹在轮到时判（§2.1、§2.4）；**m10** T3 加一次默认配置的整套 e2e、`t9-home` 加 `w2-landing`（plan）；**m11** 筛选按钮以标题为前缀得名、星在途后的焦点（§2.5、§2.9、§2.13）；**m12** 三处行号与说法（§1、§2.1、§5）。
- **观察**：o1（§2.7 一句）、o2（§2.7 的测试的根、简报）、o3（§2.1 的 `isPending` 两种键）、o4、o5（plan T3 的提交与 Step 0）、o8（§2.3 按码点数）、o9（§2.13 H17 的控制台预期）照做；o6、o10 不需要改；o7（每个星一个 `useLocalStorage` 的窗口监听）不做：监听数以渲染出的行为界、与 Plane 相同，为它给 `@nerve/hooks` 加一个导出不值。

## 4. 验收标准

- [ ] **A1 数据层**：service 四个操作、生成的类型；store 按工作区 id 持有、树与“按目标查”的派生、`answeredAt`；一个工作区一个队列、体在轮到时算、不乐观、意图幂等、`isPending`（两种键都记）；404 `navigation.favorite_not_found` 丢掉那一行；`move` 的原位与文件夹进文件夹在轮到时判；`renameTarget`；放置的表（D40 的反例、首尾 ±10000、0、负数、相等的邻居）；每个性质一个变异。
- [ ] **A2 读与门**：管理员、成员读一次、访客不读；设置页、通知页不读；键带工作区 id 与 slug；列表持有之前星、命令、筛选都不出现；swr 名单；名字不含 “sentry”。
- [ ] **A3 侧栏**：区的显示条件、开合存浏览器、一个开合按钮；行的次序、名字与图标按 `answeredAt` 取较新的、链接五种（含工作区视图）；文件夹的建、改名、删（字段提示、按码点数的长度、409、等回答、在途不取消、改名的提示）；行菜单；没有持有时读失败与重试、持有时照画行、没持有不给新建文件夹；本标签页改名立刻跟上；可访问的名字；M3 的文件夹单测照旧过。
- [ ] **A4 拖放**：§2.4 的五条规则各一格；一个 `PATCH`、只给变了的字段；失败提示；手柄（T4 新建）不是停留点；`favorites.helpers.ts` 在 T3 删、T4 新建。
- [ ] **A5 星与命令**：五个宿主、门（工作区角色、未归档、项目卡片的成员）、访客没有；`aria-pressed` 与以行为名；在途停用；加入之后区打开；命令面板两组、`shift+f`；翻转的六格照允许清单。
- [ ] **A6 “收藏”筛选**：模块列表与项目视图列表；只在会话；访客没有；只开“收藏”时页面上已应用的一行在、模块页头亮（页面一级的格）；已应用的可移除、清空一并清掉；归档列表没有。
- [ ] **A7 连带与改名的跟随**：七个连带的调用点、三个改名的调用点各一格一变异；不加请求；恢复不加回；`projectTab` 与 `listTab` 给同一个收藏 store。
- [ ] **A8 记录**：两处各一次、门与时机；§2.8 的“不记”每种有格（单测或故事）；失败静默、控制台安静。
- [ ] **A9 首页**：三种筛选与查询、两种行、空态、持有时照画行、没持有时失败与重试、每次挂载读、迟到的回答只写自己的键；小部件不在 `loader` 之后（换工作区读一次）；开预览以 `joinedAndUnarchived` 判（已归档的被持有时也走链接）、Ctrl/⌘ 照链接；已归档项目的工作项没有状态；预览宿主读标签；标题是 `h3`；筛选按钮以标题为前缀得名。
- [ ] **A10 删除与守卫**：§2.11 的表逐项删去、读者写进阶段记录；`old-favorite-urls` 有样例；phase `M7/P7`；两条地址规则的名单；oxlint 等于上限、新文件零告警；knip；死成员 7 行清零；文案经 `t()`。
- [ ] **A11 e2e 跟随**：`mounts.ts` 三个常量与头注释；S2 不变长（含 `:511`）；九份故事、五处 `VISIT`、八处“之后没有写”、`writesSeen`（R26）照 §2.13；受影响表以整个 `e2e/` 为底；不受影响的几处核过写进台账。
- [ ] **A12 页面故事**：五份三引擎各一次、chromium `--repeat-each=3`；每份不过 400 行。
- [ ] **A13 收尾**：八个页面配置与整套默认 e2e 不回归；plane-diff；前端改动清单；交接的处理结果（§5）；终检、最终分支 CI、合并、main CI 与归档（由下一段的 T1 补记）。

## 5. 交接与后续责任

**本段处理的输入交接**：都写“处理结果（M7/P7）”；写关闭的照写关闭，其余照设计 §11 保持 open 至收尾。T14 写。

| 交接 | 条款 | 本段 | Task |
|---|---|---|---|
| `D:241`（§7 数据层） | 照 v0 7.7 重写、删手写类型、旧地址一条 `old-…-urls` | 收藏与最近访问两半关闭：§2.1、§2.8、§2.11、§2.12 | T2、T3、T8 |
| `D:248`（§7 P7） | 侧栏收藏（文件夹、拖动、移出，失败经提示）、星与命令、首页的最近访问、显式记录 | 关闭 | T3–T9 |
| `D:311`（§12 P7） | H15–H17 的页面、H20 的收藏与首页 | §2.13 | T10–T13 |
| `HO/M3-closeout.md:14` | 侧栏的收藏、首页的“最近”与它的预览，按权限取数 | 关闭：§2.2、§2.9、§2.10 | T3、T9 |
| `HO/M3-closeout.md:23-24` | 项目的收藏 | 卡片的星关闭（§2.5）；`is_favorite`、建后收藏、`favoriteProjectIds` 不适用（R20） | T5 |
| `HO/M3-closeout.md:29` | 挂载时的取数按权限启用、S2 随之改 | 收藏与最近访问的一半关闭 | T3、T9 |
| `HO/M3-closeout.md:40` | 界面状态在以 id 为 key 的布局之下 | 收藏的输入在侧栏里，侧栏在以 slug 为 key 的工作区内容之下；首页的筛选在首页 | T3、T9 |
| `HO/M3-closeout.md:62` | 收藏菜单的 `opacity-0`、文件夹手柄是停留点 | 关闭：§2.3 | T3、T4 |
| `HO/M3-closeout.md:66` | 每一行的控件以它那一行为名 | 收藏的行与菜单、星关闭 | T3、T5–T7 |
| `HO/M3-closeout.md:80` | 收藏夹新建的规则没有提示 | 关闭：§2.3 | T3 |
| `HO/M3-closeout.md:95` | 地址只从路由（`favorite.store.ts`） | 关闭（这一条点名的最后一个） | T3 |
| `HO/M3-closeout.md:104` | 收藏的写死英文 | 关闭：§2.12 | T3、T4 |
| `HO/M3-closeout.md:109` | `favorite-folder.tsx` 的 2 条 oxlint | 照实跑：基线已为 0，改写之后仍为 0 | T3 |
| `HO/M1-P2-trim-content.md:18` | 首页的最近访问改用新接口 | 关闭 | T9 |
| `HO/M1-P3-trim-platform.md:15` | `TFavoriteEntityType` 改用生成的类型 | 关闭（删去，用 `FavoriteKind`） | T3 |
| `HO/M1-closeout.md:14` | 本领域的死成员 | 收藏、最近访问、首页的一半：7 行清零、新代码零 | T3、T14 |
| `HO/M2-closeout.md:29-31` | 新的 SWR 键带 `loginId` | 收藏与最近访问的读都经 `useSessionSWR` | T3、T9 |
| `HO/M6-closeout.md:24-31`（M6-M7-02） | 迭代与模块的收藏入口、归档时移除 | 关闭：§2.5、§2.7 | T6 |
| `HO8/M6-closeout.md:188-192`（M6-M8-11） | 应用里的 `useLocalStorage` 副本 | 收藏那一处改用 `@nerve/hooks`；副本与其余读者仍归 M8 | T3 |
| `P6aR:187`（P6a §7） | 收藏的写与视图的星；`entityMap`；死成员 7 行 | 关闭 | T3、T7 |
| `P6bR:192`（P6b §7） | 视图的星（工作区视图给不给）；M3 交接里随 P7 的各项；T1 回填 | 关闭（R10、上面各行、T1） | T1、T7 |
| `docs/v0/frontend-changes.md:238`、`:250` | 迭代、模块的收藏“M7 随收藏的接口加回” | 关闭 | T14 |

**后续**：
- **M8**：应用的 `useLocalStorage` 副本（M6-M8-11）；“几分钟前”的本地化；最近访问的工作项行对已归档项目没有状态（若要显示，是契约的改动，`RecentVisitIssue` 加状态的名与组，负责人）。
- **M7 收官报告**：十条 L!（§3）；向负责人说明的四件。

## 6. 风险

- **e2e 的面**（R16）：收藏的读与记录的 `POST` 碰到十余份 M2–P6b 的故事。缓解：经 `mounts.ts` 的常量；T3、T8 之前在台账写读者表与受影响的故事表（以整个 `e2e/` 的 `git grep` 为底，`acceptance` 在内）；T3 之后跑一次默认配置的整套 e2e；单配置在 chromium 先跑一遍受影响的文件（plan）；不受影响的几处写明理由。
- **记录的时机**：记录在 effect 里发出，与页面内容的出现不同步。缓解：切片里可能有它的一律排除、整会话的一律算进期望（§2.13），不靠“大概已经发了”。
- **连带漏一处**（R12）：漏了，那一个动作之后收藏还在（到下次读）。缓解：七个调用点各一格一变异；H16 (page) 每种动作一步。
- **名字的较新一份**（R7）：比较的两边来自不同的 store。缓解：`answeredAt` 是全局递增的写入序号（`C/lib/reconciled.ts:18-19`、`:35`），每个 store 一格“它较新用它、收藏较新用收藏”，一个变异；本标签页的改名不推进它，由 `renameTarget` 补上（R7），T6、T7 各一格“收藏列表较新时改名，侧栏跟上”。
- **拖放在 WebKit**：pragmatic-dnd 与 Playwright 的拖动在三引擎上靠 `dragOnto`（I6、G15 的先例三引擎都跑过）；若 WebKit 不稳，先报主控，不自行把格限在 Chromium。
- **布局变宽**：星让行与卡片的操作区变宽（P6b T5 的 I19 教训：行变宽之后按中心点击落到别处）。按中心点击这些行的故事：视图行的链接（`h12-project-view-layouts-page.spec.ts:65`、`h12-project-view-page.spec.ts:171`、`h20-view-pages.spec.ts:120`、`:333`、`h20-workspace-view-pages.spec.ts:121`、`:350`；右键 `h12-project-views-list-page.spec.ts:329`、`h14-workspace-views-list-page.spec.ts:265`）与模块行的悬停（`planning/g19-module-list-page.spec.ts:313`）；链接在操作区之左、随之变窄，中心仍在链接上；窄屏（700 宽）的几步都在视图页的页头，不在列表行上。T6、T7 的单配置照跑这些文件。
- **既有测试**：渲染宿主的测试替身多、真 store 少（`card.test.tsx` 等以 `vi.mock` 换 hook）：星在没有 store 的测试里会读不到收藏 store。缓解：允许清单列出每个文件与“加星的替身、断言不改”的改法；翻转的六格之外任何断言变了先停下报告。
- **时长与 CI**：单测估 2,000–3,000 行新增（Unit 一步 +30–60 秒）；五份页面故事 chromium 估 50–70 秒（两片各 +25–35 秒，估算，未测）；Go 不变。

## 7. 与 Plane 的差异（T14 登记进 `docs/v0/plane-diff.md` 的行为差异一节）

1. 预览不记最近访问；只在进入项目与打开工作项页时记（Plane 在 GET 里记，详情、预览、按编号、项目的工作项列表与详情、迭代、模块都记）（R11）。
2. 星的门按工作区角色，项目卡片只给项目成员（Plane 按项目角色，卡片给所有未归档项目）（R9）。
3. 工作区视图可以收藏（Plane 的网页不能）（R10）。
4. 星、侧栏的写不乐观，等 nerve 回答、在途停用（Plane 乐观写、失败改回）（R3）。
5. 拖动“之下”取与下一项的中值、以目的地的兄弟为基准、改父与次序一个请求（Plane 恒减 10000、以全部收藏为基准、两个请求，D40）（R4）。
6. 文件夹名不在前端查重，409 在字段下；等回答再关；在途时不取消（Plane 前端查重、不等回答，D41）（R6）。
7. 侧栏行的名字取 nerve 最后回答的那一份（R7）。
8. 首页只有“最近”，筛选没有“页”（R13）；已归档项目里的工作项没有状态图标。
9. 首页的工作项行在调用者没有项目角色时走链接，不开预览（R14）。
10. 侧栏收藏的读失败时给失败与重试（Plane 静默）。

## 8. 起草时的待定

无。原来的一件（`writesSeen` 的定法）由 E-26 定为 (a)（R26）：`writesSeen` 是“收集箱的写”的记录，按位置读（`wrote` 的 `writes[at]`、`.slice(2)`、`.slice(3)`）的几处对一个早于动作、时刻不定的后台写会变得不确定，而 R16 要的“页面发了什么”的全记录（`watchPage`、`recordApi`）不受影响。E-27–E-29 接受了起草者的 T2 窗口、读失败的界面与换账号再记（R27–R29）。
