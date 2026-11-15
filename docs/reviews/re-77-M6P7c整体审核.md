# M6 / P7c 整体审核

> 2026-11-16：Ready to merge after fixes（一处 Important、两处 Minor，没有须负责人决定的事）；主控三处都接受，以修复轮 F 处理（F1–F3 三个提交，F4 写记录），修复轮另经一次限定复审（§4）。
> - **目标**：分支 `codex/m6-p7c` 相对 main `d052213f82aa4fba234f5684722335e1a5cbb5be`（P7b 的 noff）的全部修改（`d052213f..5087b09e`，`bee18397` 至 `5087b09e` 的 19 个提交，含 spec 与 plan、按预检的修正、F8 的事实更正与 T1–T14，103 个文件）；修复轮 F（`98d9f985`、`10ca6ae6`、`165b252b` 三个提交，即 F2、F3、F1；F4 只改文档）另经一次限定复审（§4）。
> - **依据**：[P7c spec](../v0/M6-cycles-modules/specs/P7c-palette-mobile.md)、[P7c plan](../v0/M6-cycles-modules/plans/P7c-palette-mobile.md)、[预检 re-76](re-76-M6P7c预检.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。
> - **阶段记录**：[P7c 阶段记录](../v0/M6-cycles-modules/reviews/P7c-palette-mobile-review.md)。

## 1. 方法

一次 fresh 只读审查（`fable`，台账“整分支审查 re-77”一行），范围 `d052213f..5087b09e`（spec 与 plan、按预检的修正、F8 的事实更正与 T1–T14 的全部提交），不改动文件与 git 状态。它与 `5087b09e` 的第一次最终门禁（`t15-gate`）并行，没有跑构建与测试，引用的收据是台账的。它对照 spec 的合同与 R1–R16、预检 re-76 的修正与台账的裁定，逐项核对入口规则、搜索、工作项的两个命令、A17/A18、移动端页头、A19–A21 与打开即聚焦、X1、守卫与回归边界，读了 e2e 的三条故事与文档。

原始报告没有另存文件；结论、核对过的性质与各条发现照主控的转述，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **入口规则**：`planningListRefusal`（`core/lib/planning-switch.ts`）恰在读门 `usePlanningReads` 的 `listed` 为真时为无；同一次序在 hook 与纯函数里各写一次（T3 的裁定：`usePlanningReads` 不改，它的形状由既有测试钉住），由 `planning-switch.test.ts` 的次序行与两个菜单门为假的三行钉住；入口规则 `planningEntryOn` 是“角色是 ADMIN 或 MEMBER 且原因为无”。
- **搜索**：只在项目范围、照入口规则；读的键与项目的外壳、打开的菜单相同（`PROJECT_CYCLES`、`PROJECT_MODULES`）；“没有结果”在四组都空时才给；`/browse/:workItem` 上面板一打开就读（T5 的裁定：以面板开着为门，不以有搜索词为门）；搜索词不去首尾空白，同项目一组的写法。
- **工作项的两个命令**：选项与下拉框同一规则（“无迭代”在前、不给已结束与已归档的迭代）；所持的已结束迭代以名字显示为当前、禁用；所持的已归档模块列出、只可移出；写在面板的守卫之内、写队列轮到时算；“无迭代”的 `null` 经宿主交给命令（`PK/ui/modal/wrapper.tsx:57-62`）；模块的增或减以 nerve 确认过的集合判断——故事在第二次选之前先等页上的勾。开关在写轮到时不再核对：开关在途中被关时由 nerve 拒绝、以提示说出，可以接受。
- **A17、A18**：正确；`quick-action-dropdowns/helper.tsx` 的类型拆分没有改变别的菜单。命名的小瑕疵：`ArchivedIssueMenuProps` 也是项目页、详情页的菜单所收的一份（`MenuItemFactoryProps = Editing & ArchivedIssueMenuProps`），名字比它的用途窄。
- **移动端页头**：每页一个 `observer` 包装件算门，layout 只传路由的参数，迭代页页头的 DOM 次序不变；T10 在迭代页、模块页的 `page.test.tsx` 各加的一行 `vi.mock` 可以接受。
- **A19、A20、A21、“添加已有”打开即聚焦、X1**：正确。
- **守卫**：没有问题；`work-item/mutation-commands.ts` 照 T6、T7 的裁定不在 `project-address-route` 里（只加了命令、不读地址），加入它不会命中，是一处可以免费得到的收紧。
- **回归边界**：守住了。
- **e2e**：故事 1（命令面板）与故事 3（“添加已有”的焦点）站得住；故事 1 证不了 `/browse/:workItem` 上面板自己读列表，那由单测证。故事 2 见 I1。
- **文档**：抽查之处属实。
- **负责人**：没有要决定的事。

## 3. 结论与处理

结论 **Ready to merge after fixes**：没有 Critical；一处 Important（I1），两处 Minor（M1、M2，由主控酌定），若干观察，没有须负责人决定的事。主控三处都接受，以修复轮 F 处理（Minor 也在修复轮里修，同 P6b、P7a、P7b 的先例）；修复轮另发现两处同一类的问题，主控裁定交出：

| 编号 | 问题 | 处理 |
|---|---|---|
| I1（Important） | 移动端的故事在应用内导航之后的锚不唯一：`e2e/stories/planning/g19-work-item-side-page.spec.ts:154-156`（世界在 `:112`）在 `moveWithinApp`（pushState 加 popstate，什么都不等，`e2e/fixtures/settings-pages.ts:296-301`）之后轮询 `shownItems` 为 `["Plan"]`；Plan 同在 Sprint 与 Alpha，Sprint 的看板在 DOM 里留着 “Options: Plan”（行的快捷操作，`issue-layouts/kanban/block.tsx:93`），轮询可能在模块页挂上之前就返回，`groupings` 随之可能打开 Sprint 的“显示”，而它随即卸下 | F1（`165b252b`，只改故事）：Plan 只在 Sprint，另建 Review 只在 Alpha，模块页轮询 `["Review"]`（同 `g15-cycle-list-page.spec.ts:124-125` 的写法）；故事 3 用自己的 `moduleListWorld`，不动。竞态没法以“应当失败”的 RED 证，台账照简报记下锚点前后的差异；本文件 chromium 3 例 3.8 秒（`f1-chromium`）、三种引擎 9 例过（chromium 4.3、firefox 6.9、webkit 6.0 秒，`f1-m6-pages-file`）、chromium `--repeat-each=3` 9 例过无 flaky（各故事取最大合 4.6 秒，`f1-repeat3`），仍在 8 秒之内 |
| M1（Minor） | “改迭代”页以迭代的名字作 cmdk 的 `value`（`work-item/cycles-menu.tsx:77`、`:89`、`:98`）；nerve 的迭代名字不唯一，cmdk 1.1.1 以 value 认键盘选中的行：两个同名的可加入迭代，方向键走不到第二个、Enter 选第一个；所持的已结束或已归档的一行与可加入的同名时，选中落在禁用的那一行、Enter 无事；名为 “No cycle” 的迭代与“无迭代”一行同 value，选它发 `null`。模块没有这个问题（名字在项目内唯一） | F2（`98d9f985`）：每行的 value 是迭代的 id（“无迭代”是常量 `no-cycle`），名字作 cmdk 的 `keywords`；`PowerKModalCommandItem` 加 `keywords`；两个面板宿主（`PK/ui/modal/wrapper.tsx`、`core/components/navigation/top-nav-power-k.tsx`）的过滤先原样抽为共用的 `PK/ui/modal/filter.ts` 的 `powerKFilter`，再改为行给了 keywords 时只比 keywords、没给时照旧比 value——id 永远搜不到，别的行行为不变。模块页核过不改：`modules_project_id_name_key` 是 `(project_id, lower(name)) WHERE deleted_at IS NULL`（`server/migrations/sql/00040_planning_modules.sql:29`，含已归档的），名字经 `CleanName` 去首尾空白；迭代没有名字的唯一约束。RED `f2-red` 5 例失败在所测性质（`filter.test.ts` 两行、`cycle-command.test.tsx` 新三行）；变异 `f2-mut-name-value`（value 改回名字）与四个探针都失败 |
| M2（Minor） | `usePageDisplay`（`core/components/issues/use-page-display.ts:37`）经 `usePageIssueFilters`（`core/hooks/store/use-page-issue-filters.ts:27-28`）→ `projectIssuesFilter.issueFilters`（`core/store/issue/project/filter.store.ts:29-34`）以 router store 的项目读项目页的偏好，门却按地址读：地址由 A 换到 B、B 的偏好已在时，页头的控件在 router 落后的那一次提交里显示 A 的布局与显示；写本来就对（按 `entityId`）；与基线相同，不是本段引入 | F3（`10ca6ae6`）：`usePageDisplay` 对三种 store 一律读 `useIssues(storeType).issuesFilter.getIssueFilters(entityId)`（同 `issue-layouts/use-planning-page-header.ts:44`），不再经 `usePageIssueFilters`；后者还有 5 个产品读者，留着。RED `f3-red`（router store 的项目是 A、地址的是 B，窄屏一行与 `HeaderFilters` 显示 A 的布局）；变异 `f3-mut-router-read`（改回 `usePageIssueFilters`）失败 |
| 修复轮发现（a） | M1 的同一类问题也在命令面板别的菜单（台账 F2 的观察）：“打开迭代”（`PK/menus/cycles.tsx:26` 以 `cycle.name` 作 value）、视图（`menus/views.tsx:26`）、工作区（`menus/workspaces.tsx:32`）——都只是导航，名字都不唯一（项目、状态、标签、模块的名字在各自的范围内唯一） | 主控裁定交 P8：照 F2 的 keywords 写法，给 `PowerKMenuBuilder` 加一个 `getKeywords`（[P7c spec](../v0/M6-cycles-modules/specs/P7c-palette-mobile.md) §5） |
| 修复轮发现（b） | M2 的同一种 router 落后的读也在列表的正文（台账 F3 的观察）：`usePageIssueFilters` 余下的 5 个读者——四种布局的根（`list/base-list-root.tsx:72`、`kanban/base-kanban-root.tsx:80`、`spreadsheet/base-spreadsheet-root.tsx:49`、`calendar/base-calendar-root.tsx:85`）与日历的块（`calendar/issue-block.tsx:56`）——在项目页上经 router store 的项目读本页的偏好；早于本段（P5b 的读法） | 主控裁定并入 [M6-M8-15](../v0/M8-open-release/handoffs/M6-closeout.md#M6-M8-15)（同一类：经 router store 读项目页的地址），验收语随之扩充 |
| 观察 | 同一次序在 `usePlanningReads` 与 `planningListRefusal` 两处写 | 不需动作（T3 的裁定，测试钉住） |
| 观察 | 开关在写轮到时不再核对，途中被关时 nerve 拒绝、提示说出 | 不需动作 |
| 观察 | `ArchivedIssueMenuProps` 的名字比它的用途窄 | 不需动作 |
| 观察 | `mutation-commands.ts` 可以免费加入 `project-address-route` | 不需动作（T6、T7 的裁定照旧） |
| 观察 | 故事 1 证不了 `/browse/:workItem` 上面板自己读列表 | 不需动作（单测证，`search-menu.test.tsx` 的 `readsTheListsTheStoreLacksByThePagesKeys`） |

F4 写本记录与 [P7c 阶段记录](../v0/M6-cycles-modules/reviews/P7c-palette-mobile-review.md)，并在 spec §5 加修复轮的清单与交出的两项、改 spec 与 plan 的状态、设计 §14 的 P7c 一行、前端改动清单的两行与 M8 交接的 M6-M8-03、M6-M8-15（`docs` 提交，即本记录所在的提交）。

修复轮中实现者的局部裁定（台账修复轮 F 的 Ruling，交限定复审核对）：
- F2 的 value 取迭代的 id，不取简报举例的 `${name} ${id}`：宿主的过滤是 value 的子串匹配，id 进了 value，只由十六进制的字母与数字组成的搜索词（“a”“1”“de”）就会以 id 找到名字里没有它的迭代；照 cmdk 自己的语义（keywords 加在 value 之外）id 也仍被搜索，所以 `powerKFilter` 在行给了 keywords 时只比 keywords。代价：动了两个宿主（Plane 的同一段过滤合为一个函数）与 `PowerKModalCommandItem` 的一个 prop；宿主的接线（各一行 `filter={powerKFilter}`）没有在页上搜索的单测（`wrapper.test` 的页头是替身、`top-nav-power-k.test` 把页 mock 掉），由类型与审查管；e2e 的“改迭代”以方向键选、不搜索。
- F2 的新例加在既有的 `cycle-command.test.tsx`（359 → 387 行）的一个 `it.each` 三行（同名的两个迭代、名为 No cycle 的迭代、所持的已结束迭代与可加入的同名），迭代的 id 取 `c-1`、`c-2`（不含名字），“以名字找到”只能经 keywords；测试宿主加一个受控的搜索框（`shouldFilter={search !== ""}`，搜索为空时与改前相同），既有 18 例的断言不改；过滤自己的性质另在新的 `filter.test.ts`（6 行的表）。
- F3 的新例放在 `mobile-header-filters.test.tsx`（3 → 4 例，真的 store 与 HTTP），同一次渲染挂窄屏一行与 `HeaderFilters`，断言窄屏的触发器、`HeaderFilters` 的窄触发器与宽屏按下的布局三处；不在 `preferences-controls.test.tsx` 加（它的准备只取 A 的偏好、挂 A 的页，改准备会动既有的例）。
- F1 的新工作项叫 Review、只在 Alpha，不把 Plan 移出 Sprint、也不改等别的东西（例如 Alpha 页独有的标题）：行是 `shownItems` 已有的锚，同 G15 一页一个自己的工作项；多一次接口建项，三种引擎的时长没有可见的增加。

## 4. 修复轮 F 的限定复审

修复轮 F（`5087b09e..165b252b`，3 个提交：F2、F3、F1）另经一次 fresh 限定复审（`fable`，只读）。

（待主控填写：限定复审的结论、核对过的性质与其后的处理。）

## 5. 负责人须知

本段没有须负责人决定的事：re-77 的三处与修复轮交出的两项都是局部的，不改 `server/`、`api/`，没有改 CI 的结构。

需要负责人知道的一件事：

- **e2e 的余量**：本段的新文件本机 chromium 3.8–4.6 秒（单独跑 3.8 秒，三种引擎的页面配置里 4.3 秒，`--repeat-each=3` 各故事取最大合 4.6 秒；修复轮的 F1 只换了锚，没有加时长），预算 8 秒。推送前的估算以 X = 4.6 秒算：约 23 分 46–47 秒，上界约 29 分 07–19 秒，都低于 30 分钟的报告线；余量至此约剩 10.0–14.7 秒（若用满 8 秒的预算是 6.6–11.3 秒）。慢的 runner（× 1.31 至 × 1.357）乘在估算上约 31 分 10 秒至 32 分 17 秒，是观察、不是停下的条件。M6 之后的段再加 e2e 故事之前，作业的结构（例如分片）要负责人决定（同 [re-76](re-76-M6P7c预检.md) §5）；估算、上界或任一次实测越过 30 分钟时仍先停下报告、不推送。
