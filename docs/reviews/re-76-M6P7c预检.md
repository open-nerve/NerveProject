# M6 / P7c 预检

> 2026-11-15：预检通过（修正后），可以按 P7c 的 T2 开始实施（T1 已在预检进行时完成，只改文档，`b5e1c615`）。
> - **目标**：[P7c spec](../v0/M6-cycles-modules/specs/P7c-palette-mobile.md)、[P7c plan](../v0/M6-cycles-modules/plans/P7c-palette-mobile.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `d052213f82aa4fba234f5684722335e1a5cbb5be`（P7b 的 noff，双父 `f42f9d64`、`4fe308ce`）；审查对象为分支 `codex/m6-p7c` 的 `bee18397`。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：两个只读调研代理在 `d052213f` 上写出 P7c 的笔记（台账 `research/palette-views.md`：命令面板、快捷操作、四个视图文件、G19 的工作项一侧与“添加已有”的焦点；`research/mobile-carryovers.md`：移动端页头、A19–A21、交接回执、e2e 的余量与放置、估算；随阶段归档）。
2. **裁定与起草**：主控在写 spec 之前裁定 H1–H16（台账 `controller-rulings.md`）——15 个 Task；规划入口一条规则（ADMIN/MEMBER、项目没有归档、开关打开）与打开的菜单说原因；先按层级拆 `navigation/commands.ts`；空文案经构造件的键；搜索的项目范围的两组；工作项的“改迭代”“改模块”；A17/A18；一个共用的移动端页头；迭代页的窄屏顶行；A19 按范围读；A20、A21 与“添加已有”的焦点；X1 的范围；e2e 一个新文件 ≤ 8 秒；余量取保守的 14.6–19.3 秒、检查点在 T10；只改网页；行数与质量——再由主控写出 spec（R1–R16 照录 H1–H16；A1–A14）与 15 个 Task 的 plan（`bee18397`）。
3. **审查**：一次 fresh 只读审查（`fable`）对照 HEAD 的代码与只读的 `plane/` 逐条核查：事实与计数；按次序执行时有没有守卫在某个 Task 内变红而计划不修；R1–R16 是否站得住（入口规则与侧栏、工作项命令与下拉框和写的路径、命令面板的页里的读、快捷键的解析、`data-autofocus` 与 `ModalCore`、归档页的范围、共用的移动端页头、X1）；A1–A14 与 e2e 对 G19 的工作项一侧、P7a/P7b §5 的遗留与交接回执的覆盖；e2e 在三种引擎是否可行、余量的算式；有没有须负责人决定的事。
4. **裁定**：主控逐条核过审查的结论，全部接受，改入 spec 与 plan（本记录所在的提交）。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **命令面板**：“打开一个迭代/模块”在门为假时永远转圈（`PK/ui/pages/open-entity/project-cycles-menu.tsx:40-45`，模块同），今天只在归档的项目里走得到（四个导航命令只看角色与开关，`PK/config/navigation/commands.ts:310-367`）；`navigation/commands.ts` 466 行；七个菜单的空文案是英文字面量；搜索没有迭代、模块的结果；nerve 没有给工作项改迭代、模块的命令。
- **移动端页头**：迭代页、模块页的 layout 不传 `mobileHeader`；项目页的移动端页头在偏好读到之前就给控件（X2），它的读写重复了 `HeaderFilters` 已按 store 类型做好的那一组。
- **遗留**：分组时项目页与归档页的页头没有数目（读 `""` 组）；没有读者的键只剩 `project_cycles.status.days_left`；替身的 `opened` 关上不清；“添加已有”的搜索框打开时没有焦点；四个视图文件中三个经路由的退路读地址。
- **e2e 的时长**：P7b 的最终分支 CI 43702923005 的 e2e 作业 27 分 50 秒（1670 秒，至今最慢）、合入后的 main CI 43706438924 的 23 分 42 秒（1422 秒）；期限 35 分钟，报告线 30 分钟。

## 3. 结论与处理

结论 **Ready after fixes**：四处 Important、十二处 Minor，没有 Critical，没有架构问题，没有须负责人决定的事。全部在实施前改入 spec 与 plan；Task 数不变（15）。

| 编号 | 问题 | 处理 |
|---|---|---|
| F1（Important） | spec §2.7 让三个 layout 把 `preferencesShown` 传给共用的件，但三个 layout 都是普通的函数组件、不是 `observer`（`R/issues/(list)/layout.tsx:14`、`R/cycles/(detail)/layout.tsx:16`、模块同），在那里读的 MobX 状态（`use-planning-page-header.ts:44`）答回之后不翻转，窄屏一行永远不出现 | spec §2.7、A8、R8、§6：每页一个薄的 `observer` 包装件算本页的门再渲染共用的件——项目页的 `R/issues/(list)/mobile-header.tsx` 改写为包装件（门同桌面页头），迭代页与模块页共用一个按 store 类型的（门取自 `usePlanningPageHeader`）；layout 只传路由的参数。plan T9、T10：测试经 layout 挂载、偏好在挂载之后才答回；变异 `t10-mut-gate-outside-observer` |
| F2（Important） | 移动端的故事在宽屏打开迭代页再缩窄，偏好早已答回，“答回之后才出现”对变异同样成立，`t13-mut-mobile-gate` 杀不死；证“之前不给”要扣住回答，而 WebKit、Firefox 不装路由（P7b 记录 §5） | spec §2.12 的故事改为只核窄屏一行给出布局与“显示”，“之前不给”由 T9 的单测与 `t9-mut-no-gate` 证；plan T13 去掉 `t13-mut-mobile-gate`，改用故事杀得死的 `t13-mut-own-grouping` 或 `t13-mut-toggle-hidden` |
| F3（Important） | spec §2.2 与 plan T3 说原因的码“经 `errorMessageKey`”说明，它收的是错误对象（`core/lib/error-messages.ts:120`），给码只得 `errors.unknown`；给码的是 `problemMessageKey(code)`（`:129`，`use-refusal-toast.ts:44` 的先例） | spec §2.2、A2 与 plan T3 改为 `problemMessageKey`；测试断言具体的键（`errors.project_archived`、`errors.planning_cycles_disabled`），不只“不转圈” |
| F4（Important） | spec §2.6 把“恢复”的在途从 ref 改为状态，会重开同一拍里的两次发送（按钮菜单与右键菜单、连按两次 Enter 都看到“不在途”）；既有的“只发一次”（`cycles/actions-host.test.tsx:225`、`modules/actions-host.changes.test.tsx:228`）的第二次 `choose` 点的是已禁用的按钮（`fake-quick-actions.tsx:40-45`），碰不到处理里的守卫 | spec §2.6、§6：同步的 ref 照旧守卫发送，另加状态只为禁用的显示；plan T8：一个 `act` 里调两次处理只发一个请求，变异 `t8-mut-no-sync-guard` |
| F5 | 项目页移动端测试的引用错：`preferences-controls.test.tsx:218` 是帮手 `mountMobile`，`:228` 是 `it.each` 三行，`:251` 一例，共四例 | spec §2.7 与 plan 的回归边界、T9 改为四例、基线同数 4 |
| F6 | 两个打开的菜单的测试除 `:121`、`:134`、`:139` 三例外还有 `:147` 的 `it.each` 三行（开关关闭、已归档、不是成员：门为假时什么都不读） | plan 回归边界“入口规则”写全，T3 的 RED 加在这三行旁 |
| F7 | T6、T7 改写的 `PK/ui/pages/context-based/work-item/root.tsx` 不在守卫表里，它的局部变量 `projectMemberIds`（`:39`、`:57`）不读路由却命中 `\bprojectMemberIds\b` | spec §2.11 与 plan T6：局部变量改名为 `memberIds` 再加入守卫（审者另核过 `mutation-commands.ts`、`context.ts`、两个搜索件、七个菜单、两个视图弹窗加入时都不命中） |
| F8 | `shift+c`、`shift+m` 带 Shift，按键走 `handleModifierShortcut`（`PK/core/shortcut-handler.ts:71`、`:83-86`），要声明为 `modifierShortcut`；审者说 Plane 的两个命令没有快捷键（**更正**：T6 核实 Plane 有，`commands.ts:331` 的 `"shift+c"`、`:363` 的 `"shift+m"`，调研笔记是对的；实现不受影响，差异清单写“与 Plane 相同”） | spec §2.5、§6 与 plan T6、T7 写明 `modifierShortcut`；两个快捷键与 Plane 相同（T6 更正） |
| F9 | “没有结果”今天在工作项与项目都空时给出（`search-menu.tsx:85`），只有迭代一组有结果时会出现在组的上方；两组的读还没回答时 spec 没有说 | spec §2.4：四组都空时才给；还没回答时这一组不显示、不挡别的；plan T5 的 RED 两例 |
| F10 | 门为假而 store 持有列表时（开关在已读过它的页上被关），菜单今天照列（`project-cycles-menu.tsx:44-48`） | spec §2.2：门为假时原因优先、不列；plan T3 的 RED 一行与变异 `t3-mut-held-list` |
| F11 | 一拆为二 `git diff -M` 显示不出；新的胶水（一个 hook 变两个、各自的 `useUser()`/`useProject()`、两级都用的 `baseProjectConditions`、键的联合）要写明；T3 改的“归档项目的那一格”在 T2 的上下文表里没有 | plan T2：以 `--color-moved` 或行多重集比较证明，台账写明唯一的胶水；特征测试的上下文加“项目已归档”；替身 `callerInWeb(…, fields)` 与 `useUser` 的 mock（`config/creation/command.test.tsx:16`）可用 |
| F12 | 设计 §12 的 P7c 一行与 P7a spec §5 写“M2、M3、M1 其余的回执”，spec 漏了 M2；T5 加了 `useSessionSWR` 的读，M2 §3（键带 `loginId`）适用；A13 没列 M6-M8-08 | spec A13 与 plan T14 加 M2 §3 的回执与 M6-M8-08 的行数 |
| F13 | plan T10 的检查点列在“提交”之前，门禁的树要是推送的那个提交 | plan T10：先提交，再检查点 |
| F14 | `useCycleOptions` 在 `/browse/:workItem` 上可用，但它的 `load` 是命令式、每次渲染新建（`directory-read.tsx:22`），命令面板的页要在挂载的 effect 里调它，会碰 `react-hooks/exhaustive-deps`，而告警上限恰为 44 | spec §2.5 与 plan T6 定为打开的菜单的门控 `useSessionSWR` 读加纯函数 `cycleOptions`、`moduleOptions`，不再留给 T6 选 |
| F15 | 共用的件带 `activeLayout`，项目页的布局按钮从此显示当前布局的图标，§6 说“唯一的变化是门”不确；T12 改到 M7 的视图文件，落进 M1 的“改过的文件 oxlint 0” | spec §2.7 与 §6 改写（两处可见的变化，C2、C12 不受影响）；plan T2 先取本段要改的文件的 oxlint 基线（`t2-oxlint-baseline`），T14 对照 |
| F16（过程） | 审查时 worktree 有四个未提交的文档改动（P7b 的记录、spec、plan 与设计）——那是与预检并行的 T1 | T1 已提交（`b5e1c615`，收据齐全）；本记录与 spec、plan 的修正在其后 |

## 4. 审者核实为正确的事实（摘要）

- 引用与行数：`commands.ts` 466 行与 `:310-367`；七处空文案、`empty-state.tsx:13`、`builder.tsx:38`；`context.ts:24-30`、`mutation-commands.ts:28-48`、`issue-write-fields.ts:58-70`；`cycles/quick-actions.tsx:68-102`；`helper.tsx:64-74`、`:272-283`；`filters.tsx:45-86`、`filter.ts:236-240`；两页页头 `:143`、`:145-164`；项目页移动端页头 98 行；`paged-groups.ts:169`、`planning-page-issues.store.ts:29-51`、`project/issue.store.ts:10-16`；`cycle.ts:9/17/25/33/41`；`fake-planning-page.tsx` 378 行与 `:204`、`:209`；`form.tsx:133`；`in-session.ts:40-53`；两个夹具 359、388 行。
- 守卫：拆分后项目一级一件不命中（`allowPermissions` 都给四个参数），工作区一级一件命中 `:74-79`（应留在外）；`R/issues/(list)/mobile-header.tsx`、两个打开的菜单、`menus/cycles.tsx`、`menus/modules.tsx`、两页 layout 与页头、`issues/header.tsx`、`filters.tsx`、`existing-issues-list-modal.tsx` 已在表里；`R/views/(detail)/[viewId]/header.tsx` 不命中；`search-menu.tsx` 与两个打开的菜单已在 swr 限制表里。
- i18n：`common.cycles`、`common.modules`、`cycle.no_cycle`、`power_k.search_menu.no_results` 与 `errors.*` 的键都在；四个新键都不在、不是别的键的前缀；`days_left` 只经无人读的 `i18n_label`。
- 入口规则与侧栏一致；工作区管理员而项目访客在两处都映射为 ADMIN（`permissions.store.ts:68-74`，A2 加了这一行）。
- 写的路径：`update(held => changeBody(change, held.module_ids))` 恰是 `changeIssue`（`issue-writes.ts:273-274`），选迭代的请求体只有 `{cycle_id}`。
- 焦点：Headless UI 2.2.10 的焦点陷阱先聚焦 `[data-autofocus]`，否则聚焦对话本身，`ModalCore` 适用，“移动端不聚焦”成立（测试的先例 `cycles/modal.test.tsx:310`）。
- A19：归档页的 store 以 `scope: "project"`、`archived: "true"` 在自己的 store 里选列表（`archived/issue.store.ts:18`），“范围是本页的项目”的读在那里也成立。
- X1：两个组件有 `useParams`，列表页有路由参数，两个弹窗已收 `workspaceSlug`、`projectId`；视图没有既有的测试。
- e2e：文件名合 m6-pages 的 `testMatch`；`fixtures/power-k.ts` 有 `ControlOrMeta+k`；`openListPage` 要“Toggle details”按钮，只用于迭代页、模块页（spec 正是如此）；分组的名字是 “Cycle”“Modules”；8 秒在本机 chromium 可行但偏紧（审者没有量）。
- 余量的算式：S = 1730.5–1740.1 秒；X = 8 秒的上界 29 分 19–33 秒；余量 14.6–19.3 秒（不计关系一支的等待 15.5–20.2 秒）；估算 23 分 49–51 秒；用满 8 秒后剩 6.6–11.3 秒。

## 5. 负责人须知

开工不须负责人决定任何事。R1–R16 是主控在已批准的设计之内、写 spec 之前的裁定（H1–H16）；本段只改网页，不改 `server/`、`api/`。

需要负责人知道的一件事：

- **e2e 的余量在缩小**：P7b 的最终分支 CI 的 e2e 作业 27 分 50 秒是至今最慢的；本段 ≤ 8 秒之后上界约 29 分 19–33 秒，余量只剩约 6.6–11.3 秒。T1 另记了一个观察：那次分支 CI 是检查点校准点的 1.355–1.357 倍，乘在 1422 秒上约 32 分 07–10 秒（M6-M8-01 原记的慢 runner 是 ×1.31，约 31 分 03 秒）——这不是停下的条件（估算、上界与实测都在 30 分钟以内），但 M6 之后的段再加 e2e 故事之前，作业的结构（例如分片）要负责人决定。估算、上界或任一次实测越过 30 分钟时仍先停下报告、不推送。
