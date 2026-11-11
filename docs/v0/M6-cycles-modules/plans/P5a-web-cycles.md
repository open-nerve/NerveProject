# M6 / P5a 网页的迭代实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：已完成（2026-11-11，noff `0b8efcd7`，main CI 42995581409 首次通过），T1–T16 已实施，整分支审查 [re-65](../../../reviews/re-65-M6P5a整体审核.md)，阶段记录 [P5a 阶段记录](../reviews/P5a-web-cycles-review.md)。预检通过（修正后，[re-64](../../../reviews/re-64-M6P5a预检.md)）：H1 告警上限随清掉告警的提交下调、H2 新工具函数另起模块与名字、H3 优先级页签移到 P5b、M1 旧地址规则以接口前缀锚定、M2 hook 随第一个读者建立、M3 三种引擎的页面配置、M4–M7 与 L1–L7 已改入。

**Goal：**
- 迭代的 service 与 store 按会话分代、只调 `/api/v0`、类型来自 OpenAPI；包装层按权限取迭代；
- 迭代列表、进行中卡片、预览、归档目录、详情页的页头与进度侧栏、新建/修改弹窗、快捷操作、删除与归档确认框、转移弹窗都接上新接口，被拒时留着并说明；
- 每域一个进度函数，燃尽图按 `Burndown` 画；
- 旧的 service、store、手写类型与旧地址删除，守卫与告警收紧。

**Architecture：**
- 新的 `RootStore.planning`（`cycles`、`cycleFilters`）照 v0 §7.7 的规则写：`ReconciledByKey` 存值、`oneAtATime` 发修改、`useSessionSWR` 取数；与旧的 `cycle`/`cycleFilter` 并存到 T12，逐个页面改用新 store（spec R3）。
- 页面读路由参数，写入口按 nerve 的权限规则显示，跟进经 `followInSession`，拒绝经 `useFormRefusal`/`useRefusalToast`。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/vitest/Playwright，不加依赖。

**Spec：** [P5a-web-cycles](../specs/P5a-web-cycles.md)（§2 的合同与 §3 的 R1–R12 是本计划的约束）；设计 [M6](../M6-design.md) §2、§3、§8；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：`core/services/project/labels.service.ts`、`core/store/label.store.ts`（及 `label.store.test.ts`、`label.store.changes.test.ts`）、`core/store/project/project.store.ts`（未归档、归档、详情三份副本）、`core/store/project/project_filter.store.ts`、`core/components/project/archive-restore-modal.tsx`、`core/components/labels/use-label-changes.ts`、`e2e/stories/project/p6-states-page.spec.ts`、`e2e/stories/issue/i22-session-switch.spec.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p5a`，分支 `codex/m6-p5a`，基线 main `a2b50624`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/docs/test/refactor(M6/P5a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 由 T1 改为 `M6/P5`；不延长任何 `until`。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（worktree 里没有 `plane/`，读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内；新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`；service 不转换数据；store 只经自己的 `RootStore` 找兄弟 store；没有模块级的带令牌 service 实例；SWR 只经 `useSessionSWR`。
- 删掉不用的代码；不加开关、桩或兼容别名（R3 的并行在 T12 结束）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限；清掉告警的提交同时把上限下调到实际值（T7、T8、T9 各自的提交），不留到 T12。
- **knip**：新文件在第一个读者出现的 Task 里建立（`use-cycles.ts` 的第一个读者是 T2 的包装层，`use-cycle-filters.ts` 是 T9 的列表页），只给测试读的导出不算读者。
- **共用的 hook 假件**：组件测试要迭代的 hook 时扩展 `@/hooks/store/fake-store-hooks`（现无迭代的 hook）或直接 mock `@/hooks/store/use-cycles`。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败，再实现到通过。
- 权限门（写入口只给能写的人、删除只给管理员或建立者）、迟到的应答（换会话、换页面）、拒绝时留着、冻结的条件，每个性质至少一个能让测试失败的变异，只计能编译的变异。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出时跑 `make knip`。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P4 的网页单测与 e2e。
- 只在 spec 说契约变化的地方改：`root.store.test.ts` 的 store 数与 `dispose()` 的断言、`removal.test.tsx` 与 `cycle-peek-overview.test.tsx`、`[cycleId]/page.test.tsx`、`form-focus.test.tsx` 中读旧迭代 store 或旧类型的部分、`progress.test.ts` 的旧用例；M4 的冒烟故事 `e2e/stories/smoke/s2-issue-list-removal.spec.ts` 中迭代与已归档迭代两条（它们钉的是“迭代页面还请求旧地址、得 404”的状态，T9 的检查点发现；两条移出，覆盖由 T13 的 G3、G8 页面故事接上）。其他断言不改。

**检查点与证据**
- 检查点：T9 结束时（数据层、弹窗、快捷操作、详情、列表与归档目录完成）串行完整七项，推送并核分支 CI（同时核 e2e 作业的实际时长）。T16 收尾再串行一次。
- 证据放在 worktree 的 `.superpowers/sdd/P5a-web-cycles/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，调研笔记在 `research/`。实现代理先读同目录的 `implementer-brief.md`。

## Review Focus

1. **会话分代（v0 §7.7）**：客户端来自这一代；SWR 键带 `loginId`；换代后旧一代的回答不写进新一代；修改排队、请求体轮到时算；副本取最后回答的一份；写入的回答不把行的计数写坏（R9）。
2. **口径（spec §2.3、R7）**：`cycleCounts`、`progressPercent` 是唯一口径；冻结只在已结束时读；燃尽的 `null` 与单点。
3. **拒绝（R12）**：每个写入口被拒时留着并说明，不提示成功；只在 nerve 确认后关闭、跳转、提示。
4. **权限（R8）**：写入口、删除、转移、侧栏改日期的显示条件；按权限的取数。
5. **页面的范围（G19）**：路由参数而非 router store；详情以 `cycleId` 为 key；宿主以 `projectId` 为 key 并清理开关；迟到的跟进不进新页面。
6. **删除干净**：旧 service、store、类型、地址、收藏入口、时区换算全部删除；守卫与告警的上限。

## Task 1（T1）：P4 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P4-progress-transfer-review.md`：抬头与 §7。
- `specs/P4-progress-transfer.md`：抬头与 A8；`plans/P4-progress-transfer.md`：抬头。
- `M6-design.md`：抬头、§14 的 P4 行“已完成”、P5 行“进行中（P5a）”。
- `tools/keywords.json`：顶层 `phase` 改为 `M6/P5`。

- [ ] Step 1：按 P4 的两个归档（主目录 `.superpowers/sdd/M6-P4-integration-20261110`、`M6-P4-ci-20261110`）里的 CI 收据与 `cleanup.log` 回填 §7：
  - **最终候选的分支 CI**：`18aeb25ae9e43e9c4d724be98a19bc64ab63a2a0` 的 42800976748 attempt 1：web 129778085248、server (1) 129778085665 成功，server (0) 129778085598 失败（`TestTransferKeepsTheJobsFreeze`，§5），e2e 129784645493 跳过；修正后 `292f40fe3a9b1d4899fbd27b4f0fbf8ed2073399` 的 42803535189 attempt 1 成功（server (0) 129786842989、web 129786843414、server (1) 129786843437、e2e 129794166643）。
  - **合入与 main CI**：noff `c3ba5a40b38380e03a03334e1c680763e991b587`，双父 `ed8f0df4`、`292f40fe`，树与候选相同（`5489b877`）；main CI 42809569341 attempt 1：web 129807294464、server (0) 129807294819、server (1) 129807295028 成功，e2e 129815035293 失败 435/1（C12，§5），按规则不重跑。
  - **修复分支与第二次 main CI**：`codex/m6-p4-ci`（自 `c3ba5a40`：`675de55e` 修正与单测、`d7824b26` 复审结论与交接）的分支 CI 42818532689 attempt 1 成功（web 129837762256、server (1) 129837762619、server (0) 129837762738、e2e 129846836535）；noff `a2b50624fd872c183628a55d04b7961f9a1dbc60`，双父 `c3ba5a40`、`d7824b26`，树与修复分支相同（`16eae5f1`）；main CI 42824266166 attempt 1 成功（web 129857313676、server (0) 129857314105、server (1) 129857314265、e2e 129866383345）。
  - **归档**：`M6-P4-integration-20261110` 3823 个文件、532,660,546 字节、未知 0、清单 sha256 `e786eaba5e9d4b9a06d551babd3194799c4d8ea21b65bfdf7c3a3b0e9573e430`；`M6-P4-ci-20261110` 2521 个文件、64,500,851 字节、未知 0、清单 sha256 `dd85429f4bdda14cd53a8eac74a7994f2dde95ea5b8aeaf9c5ceef8d6cd7d50b`；P5a 开工时复核 `archive.py verify`（收据 `t1-p4-archive-verify`、`t1-p4ci-archive-verify`）。
  - **清理**：两份 `cleanup.log` 记录分支已含于 origin/main，worktree `m6-p4`、`m6-p4-ci`，本地与远端分支 `codex/m6-p4`、`codex/m6-p4-ci` 已删除，主目录停在 `a2b50624`。
  - 勾选 A8；阶段记录、spec、plan 的抬头改为已完成。
- [ ] Step 2：设计 §14 的 P4 行改为“已完成”并附证据，P5 行改为“进行中（拆为 P5a/P5b）”，链接 P5a 的 spec 与 plan；抬头的状态句随之改。`keywords.json` 的 phase 改为 `M6/P5`，跑 `node tools/keywords.mjs`（没有到期的例外）。
- [ ] Step 3：检查文档链接（`doc-links.py`），然后提交 `docs(M6/P5a): 回填P4的集成结果，进入P5阶段`。

## Task 2（T2）：service 与 store 的读

**Files：**
- 新：`web/apps/web/core/services/planning/cycles.service.ts`；`core/store/planning/{index.ts, cycle.store.ts}` 与测试（值、读与选择器在此，写在 T3 另一个文件，约 400 行的限制）；`core/hooks/store/use-cycles.ts`（读者：包装层）。
- 改：`core/store/root.store.ts`（`planning` 字段，传 `api`）、`root.store.test.ts`（store 数）；`core/layouts/auth-layout/use-project-fetch.ts` 与测试；`.oxlintrc.json`（`no-restricted-imports` 列入新文件）。

- [ ] Step 1：先写测试（`FakeNerve`、`fakeRoot`、`listTab`、`fake-projects.ts` 的 `cycleOf`）：
  - 列表、归档列表、详情、进度各自经 service 取到并按项目/迭代 id 存；
  - 较旧的取数答到时不写（`Reconciled` 的性质，至少一例）；
  - `getCycleById` 取最后回答的一份，行的计数来自列表行；
  - 项目不在调用者的项目列表里时选择器回答 `undefined`；
  - 进行中的迭代 id 是第一个 `status === "current"` 的；
  - 包装层：成员、项目未归档、`cycle_view` 打开时才取（`PROJECT_CYCLES`，键经 `sessionKey`），三种情形各一例不取。
  看到它们因所测性质失败（不是编译失败）。
- [ ] Step 2：实现 service、store 的读与包装层的取数，到通过。变异：包装层不看 `member_role`（`t2-mut-fetch-non-member`）、不看 `cycle_view`、`getCycleById` 不看 `answeredAt`。
- [ ] Step 3：`make lint-web`、聚焦 vitest、`make knip`；提交。

## Task 3（T3）：store 的写

**Files：** `core/store/planning/` 中写的文件（新，由 `cycle.store.ts` 经组合或方法委托使用；切法记入台账）、新 `cycle.store.changes.test.ts`。

- [ ] Step 1：先写测试（`fake-queue.ts` 的 `inTurn`、`sent`、`fetchedWhileChangeIsOut`）：
  - 建：回答补成计数为 0、`assignee_ids: []`、`progress_snapshot: null` 的行，放在未归档列表最前；
  - 改：字段与 `status` 写进每一份，行的计数不变（spec R9）；
  - 归档/恢复：行在两个列表之间移动、计数保留、放在最前；重复归档照常；
  - 删：每一份删去，进度删去；
  - 转移：回答个数，之后重取源与目标的进度与本项目列表；
  - 修改一个接一个发出，请求体在轮到它时算；取数在外时确认的修改重放到它的回答上；
  - 每个写的拒绝交给调用者，store 的值不变；`SessionChangedError` 不改这一代。
- [ ] Step 2：实现到通过。变异：改把计数写成回答里没有的值（`t3-mut-update-drops-counts`）、归档不移动、拒绝时仍写入、请求体在调用时算。
- [ ] Step 3：门禁同 T2；提交。

## Task 4（T4）：筛选 store 与工具函数

**Files：**
- 新：`core/store/planning/cycle_filter.store.ts` 与测试（hook `use-cycle-filters.ts` 在 T9 随列表页建立）。
- 新：`web/packages/utils/src/planning.ts` 与测试，由 `index.ts` 导出（旧的 `cycle.ts` 有同名的 `orderCycles`、`shouldFilterCycle`，由旧 store 读到 T12，所以新函数另起名字：`orderCycleRows`、`matchesCycleFilters`、`cycleCounts`、`progressPercent`）。
- 改：`core/lib/store-context.test.ts`（退役的一代不再响应 router，spec R4；`:468-470` 的注释随之改）；`core/store/root.store.ts` 的 `dispose()` 说明。

- [ ] Step 1：先写测试：
  - 筛选与搜索词按项目 id 存取，两个项目互不影响；没有 `reaction`，`store-context.test.ts` 断言换代后旧一代的筛选不随 router 变；
  - `orderCycleRows` 的 Plane 次序只用服务端的 `status`；
  - `matchesCycleFilters` 的状态与日期筛选按日历日（在负偏移时区 `TZ=America/Los_Angeles` 下另跑一次，日子不偏）；
  - `cycleCounts`：已结束且有冻结时给冻结，未结束（即使有冻结）给实时，已结束无冻结给实时；
  - `progressPercent`：分母为 0 时为 0，已取消不在分母里，取整。
- [ ] Step 2：实现到通过。变异：`cycleCounts` 不看 `status`（`t4-mut-frozen-any-status`）、分母含已取消、日期经 `new Date()` 解析。
- [ ] Step 3：门禁同 T2（含 `pnpm --filter @nerve/utils test`）；提交。

## Task 5（T5）：燃尽图与分布的行

**Files：** 新 `core/components/core/sidebar/burndown-chart.tsx` 与测试；新 `core/components/cycles/distribution-rows.ts`（或同等位置，把分布行配上名字的 hook/函数）与测试；`@nerve/propel` 的 `AreaChart` 只在需要时改（单点的除零）。

- [ ] Step 1：先写测试：
  - 燃尽图：1 个点、2 个点、末尾若干 `null`、全是 `null`（今天在开始之前）各一例，断言传给图表的数据：`null` 的实际值不是 0、理想线从 `scope` 到 0、单点时理想值为 0、所有坐标有限；
  - 分布的行：成员与标签的名字、头像与颜色；成员已离开、标签已删除时的兜底名字；`null` 行的名字；次序（名字、兜底在后、`null` 最后）。
- [ ] Step 2：实现到通过。变异：`null` 当 0（`t5-mut-null-as-zero`）、单点按 `n-1` 除。
- [ ] Step 3：门禁同 T2；提交。

## Task 6（T6）：项目级宿主与新建/修改弹窗

**Files：** `core/components/modals/project-level.tsx`（与 `power-k/projects-app-provider.tsx` 的挂载处）、`core/components/cycles/{modal.tsx, form.tsx}`、`form-focus.test.tsx`；新文案。

- [ ] Step 1：先写组件测试（jsdom，`fake-session-swr`、`fake-store-hooks`、`fake-refusal`）：
  - 宿主以 `projectId` 为 key，卸载时关上三个开关；
  - 草稿的请求体没有日期键，带日期的有一对；修改只发改了的字段；
  - 被拒（409 重叠、422 名字）时弹窗留着、输入保留，字段错误在字段下并聚焦，其余拒绝经提示；只在确认后关闭；
  - 标题经 `ModalTitle`，没有正 `tabIndex`，Escape 经 `handleClose`、提交中不能关，焦点回到打开的控件（`DateRangeDropdown` 已固定 `modal={false}`，浮层在弹窗内的可用性由 T13 的页面故事在三种引擎上核对）。
- [ ] Step 2：实现到通过（删除 `date-check` 与模块级 `CycleService`、`getTabIndex(PROJECT_CYCLE)`、全局 `mutate`）。变异：失败也关闭（`t6-mut-close-on-refusal`）、草稿带 `null` 日期。
- [ ] Step 3：门禁同 T2；提交。

## Task 7（T7）：快捷操作、删除、归档与恢复

**Files：** `core/components/cycles/{quick-actions.tsx, delete-modal.tsx}`、`archived-cycles/modal.tsx`、`common/quick-actions-helper.tsx` 的迭代菜单；`tools/keywords.json`（`copy-hook` 中 `cycles/quick-actions.tsx` 的例外）。

- [ ] Step 1：先写组件测试：
  - 菜单各项的显示与停用条件（spec §2.5 快捷操作一行、R8）：能写、已结束、已归档、管理员、建立者、其他成员、访客；
  - 复制链接经 `useCopyLink`，失败有提示；新标签页带 `noopener,noreferrer`；
  - 删除、归档、恢复只在确认后关闭/提示/跳转，被拒时留着并说明，不提示成功；
  - 菜单按钮以迭代命名、不嵌套按钮；归档确认框的标题经 `ModalTitle`、没有 `tabIndex={1}`。
- [ ] Step 2：实现到通过；同一提交删除 `copy-hook` 的这条例外，并因 `archived-cycles/modal.tsx` 的正 `tabIndex` 告警消失把 `lint-cap` 的上限减一。恢复成功后进入归档目录（同 Plane）。变异：成员非建立者也能删除（`t7-mut-delete-any-member`）、恢复失败仍提示成功、访客看到归档。
- [ ] Step 3：门禁同 T2；提交。

## Task 8（T8）：详情页与进度侧栏

**Files：** 路由 `cycles/(detail)/{layout.tsx, header.tsx, [cycleId]/page.tsx}` 与 `page.test.tsx`；`core/components/cycles/progress-sidebar/*`；与模块共用的 `core/components/core/sidebar/{progress-stats/{assignee,label,state_group,shared}.tsx, single-progress-stats.tsx}` 中迭代用到的部分（模块的调用方保持可编译、行为不变；改到 `single-progress-stats.tsx` 时它的 2 条告警清掉、上限同提交下调）；`project-address-route` 的文件表。

- [ ] Step 1：先写测试：
  - 子树以 `cycleId` 为 key；404 与其他失败的两种态；已归档的迭代另取归档列表，成员从那一行读；
  - 面包屑切换列出本项目未归档的迭代；
  - 侧栏改日期：PATCH 确认前不提示成功、被拒时回到 nerve 的值；能写、未结束、未归档时才可改；
  - “工作项”与状态页签的数经 `cycleCounts`/`progressPercent`；已结束且有冻结时读冻结（计数、分布、燃尽），否则读实时；草稿“暂无数据”；
  - 负责人与标签页签用 T5 的行；页签记在 localStorage。
- [ ] Step 2：实现到通过（删除 `cycleDateCheck`、模块级 `CycleService`、`useTimeZoneConverter` 在侧栏的使用、`validateCycleSnapshot`）。变异：侧栏不等回答就提示（`t8-mut-toast-before-answer`）、冻结在未结束时也读。
- [ ] Step 3：门禁同 T2；提交。

## Task 9（T9）：列表页、预览与归档目录（检查点）

**Files：** 路由 `cycles/(list)/*`、`archives/cycles/*`；`core/components/cycles/{cycles-view.tsx, cycles-view-header.tsx, list/*, cycle-peek-overview.tsx, archived-cycles/{root,header,view}.tsx, applied-filters/*, dropdowns/filters/*}`；`cycle-peek-overview.test.tsx`、`removal.test.tsx` 中迭代归档根的部分；`project-address-route` 的文件表。

- [ ] Step 1：先写测试：
  - 列表页的五种态（停用、加载、失败可重试、空、有迭代）；三段的成员与次序（草稿在“即将开始”）；
  - 行：圆环的百分比、草稿与即将开始的工作项数、日历日、负责人头像、没有收藏星；已归档的行打开预览；
  - 搜索与筛选按项目、按服务端 `status`；空态区分清除筛选与清除搜索；
  - 归档目录：取数失败可重试（不永远加载）、空态、搜索与筛选；
  - 移动端版式菜单删除；筛选标题经 `t()`；没有 `autoFocus` 告警。
- [ ] Step 2：实现到通过（建立 `use-cycle-filters.ts`；删除 `useTimeZoneConverter` 在列表行的使用、列表行的收藏；`dropdowns/filters/root.tsx` 的 `autoFocus`、`applied-filters/root.tsx` 的 `prefer-set-has` 两条告警清掉，上限同提交下调）。变异：取数失败时仍显示加载（`t9-mut-archive-forever-loader`）、草稿另成一段。
- [ ] Step 3：门禁同 T2；提交。
- [ ] Step 4（检查点）：`sh .superpowers/sdd/P5a-web-cycles/gates.sh t9-gate` 串行七项；推送准确的 HEAD，核分支 CI（记录 e2e 作业的时长）。

## Task 10（T10）：进行中卡片

**Files：** `core/components/cycles/active-cycle/*`（优先级页签在 P5b，spec R10；本 Task 删除现有的优先级页签代码，P5b 重建）。

- [ ] Step 1：先写测试：
  - 没有进行中的迭代时的空态；
  - 进度卡（状态组的行、已取消的说明、百分比）、燃尽卡（待处理数、`BurndownChart`）；
  - 统计卡只有负责人与标签两个页签，用 T5 的行，默认负责人；页签记在 localStorage。
- [ ] Step 2：实现到通过。变异：进度卡读冻结（进行中的迭代不该有冻结，`t10-mut-active-reads-frozen`）、页签用服务端的 id 次序而不是名字。
- [ ] Step 3：门禁同 T2；提交。

## Task 11（T11）：转移

**Files：** 新 `core/components/cycles/{transfer-issues-modal.tsx, transfer-issues.tsx}` 与测试；列表行与详情页的入口；新文案。

- [ ] Step 1：先写测试：
  - 入口的显示条件（能写、已结束、未归档、实时未完成数 > 0；已冻结仍显示）；
  - 目标的范围（未归档、草稿/即将开始/进行中、不是源），搜索；没有目标的说明；
  - 等回答：成功关闭并提示个数（0 也成功）、触发重取；被拒（422、409）时留着并说明；
  - 标题经 `ModalTitle`，目标按钮有名字。
- [ ] Step 2：实现到通过。变异：冻结后隐藏入口（`t11-mut-hide-when-frozen`）、不等回答就关闭、访客看到入口。
- [ ] Step 3：门禁同 T2；提交。

## Task 12（T12）：跨领域读者、旧代码删除与守卫

**Files：** spec §2.7 所列；`tools/keywords.json`（`old-cycle-urls`、`project-address-route`、`copy-hook` 中命令面板迭代的例外）；`web/apps/web/package.json` 的 `lint-cap`；`.oxlintrc.json`。

- [ ] Step 1：先写会失败的检查：`old-cycle-urls` 规则（以 `/api/workspaces/` 前缀锚定，同 `old-issue-urls`，spec §2.8；样例的命中含 `…/cycles/`、`…/archived-cycles/`、`…/cycles/date-check/`、`…/cycle-issues/`、`…/transfer-issues/`，不命中含 `/api/v0/projects/{project_id}/cycles`、`./archived-cycles/modal`、`/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}`）加入后 `node tools/keywords.mjs` 因旧 service 命中而失败（收据）；命令面板的迭代菜单与上下文的测试改用新 store。
- [ ] Step 2：改完 spec §2.7 的读者，删除旧的 service、store、hook、`@nerve/types` 的 `cycle/`、`@nerve/utils` 的 `cycle.ts` 与 `progress.test.ts`、`useTimeZoneConverter`、`ETabIndices.PROJECT_CYCLE`、`WORKSPACE_CYCLES`、不再有读者的文案键；删除命令面板迭代的 `copy-hook` 例外；`RootStore` 只剩 `planning`；`root.store.test.ts` 的 store 数随之改。
- [ ] Step 3：死成员：把 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)附录 A 的 `deadsym.mjs`、`lib/program.mjs`、`domains.mjs` 原样写进台账目录 `dead-members/`（`lib/program.mjs` 写死 `typescript@5.8.3`，与已装的一致），先跑 `react-router typegen` 再跑 `domains.mjs --rows M6`，列出迭代的行与 `progress-stats` 的 8 行，删去或写下结论（台账）；oxlint 告警：核对本段改到的文件为零、`lint-cap` 等于实际值（T7–T9 已随提交下调）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`；提交。

## Task 13（T13）：页面故事一（G1、G2、G3、G8）

**Files：** `e2e/stories/planning/{g1-create-cycle-page, g2-cycle-status-page, g3-change-cycle-page, g8-archive-cycle-page}.spec.ts`；新 `e2e/playwright.m6-pages.config.ts`（chromium、firefox、webkit，`grep` `G\d+ \(page\)`，先例 `playwright.m4-list.config.ts`）；需要的夹具（`fixtures/` 的页面定位与等待）。

- [ ] Step 1：写故事（spec §2.9 的表），数据库断言与同号接口故事共用；先跑一个期望错误的版本，看到断言失败，再改正。另接上 S2 移出的两条：G3 中打开不存在的迭代地址显示“迭代不存在”，接口请求只有 `/api/v0` 的读（迭代与进度 404），没有旧地址；G8 中没有已归档迭代时归档目录显示空态，不是加载占位，没有旧地址。
- [ ] Step 2：`make build build-s3-fixture` 后聚焦跑（`cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test --project=chromium stories/planning/g1-create-cycle-page.spec.ts …`），通过后再以 `--config playwright.m6-pages.config.ts` 在三种引擎上跑一次（M4 交接 M6-04(a)，收据），然后提交。

## Task 14（T14）：页面故事二（G5、G6、G7、G19）

**Files：** `e2e/stories/planning/{g5-cycle-progress-page, g6-cycle-freeze-page, g7-cycle-transfer-page, g19-cycle-pages}.spec.ts`；夹具。

- [ ] Step 1：写故事；冻结任务在测试环境 2s 一次，以轮询数据库等待，不睡固定时间；迟到的回答用 `sentHeld`，换账号用 `anotherTabSignsIn`，换工作区与换项目用 `moveWithinApp`/后退，复制失败用 `refuseClipboardWrites`。先看到错误的期望失败，再改正。
- [ ] Step 2：同 T13 聚焦跑并以三种引擎的配置跑一次，通过后提交。

## Task 15（T15）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`：网页的行为差异（spec R6、R8、R11、R12；没有 `date-check` 的调用、状态只读服务端、冻结只在已结束时读、转移看实时未完成数且可多次、转移与删除的权限门、已结束/已归档的迭代菜单里没有删除而服务端允许、已归档迭代的详情可读、收藏入口删除、燃尽 `null` 不画）。
- `docs/v0/frontend-changes.md`：§3.1 迭代一行、§3.2 结构行、包装层的取数、R4 的搜索词。
- 输入交接的“处理结果（M6/P5a）”：M1-P2、M1-P3（网页改用新接口，`status: closed`）；M1-closeout（进度的口径、死成员与告警的迭代部分）；M2-closeout §3–§6 的迭代各项（§4 写明：`cycle-options.tsx` 已不存在，“较旧的取数不覆盖较新的”由 T2 的 store 测试钉住，下拉框在 P7）；M3-closeout §1–§5 的迭代各项（§4(c) 写明 `no-shadow` 已在全仓为错误、`cycles/modal.tsx` 没有该告警）；M4-closeout M6-01、M6-04（含三种引擎的收据）、M6-02 的迭代部分（列表在 P5b）；其余保持 open。
- 设计：§8（R5 的三段、R2 的落点、R4 的迭代筛选没有反应）、§12（P5 拆为 P5a/P5b，P7 一行去掉已在 P5a 完成的几项）、§14 的 P5a 行；`docs/v0/v0-design.md` §7.7“一代退役时释放它的反应”一句（迭代筛选不跟随 router、没有反应；模块的在 P6）。

- [ ] Step 1：写差异与回执；Plane 的行号对照主目录的只读 `plane/` 逐一核过。
- [ ] Step 2：检查文档链接与关键词，然后提交。

## Task 16（T16）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t16-gate`，留收据），另以三种引擎的页面配置跑一次（收据），推送准确的 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账；
  - 写阶段记录 `reviews/P5a-web-cycles-review.md`；
  - 勾选 spec §4；A8 待合入后由 P5b 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
