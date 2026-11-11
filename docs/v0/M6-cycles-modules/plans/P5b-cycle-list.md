# M6 / P5b 迭代页的工作项列表实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（修正后，[re-66](../../../reviews/re-66-M6P5b预检.md)）：H1 迭代关闭时列表区停用、不取列表与偏好，H2 日历子树的读改经本页筛选（T6），M1 整字段替换不依赖已读的值、先进入后打开预览，M2 base root 的只读 prop（T7），M3 别的项目新建不带 `cycle_id`、页头弹窗锁定项目，M4 优先级页签的 store 每代一个，M5 失权迟到答与归档范围进页面故事，M6 `project-address-route` 落到各 Task，L1–L9 已改入；Task 由 14 个拆为 16 个。

**Goal：**
- 迭代页的工作项列表接上 M4 的列表引擎：`EIssuesStoreType.CYCLE`、本页偏好、四种布局、页头、空态、只读、本页新建与移出迭代；
- 成员判断认得 `cycle_id`（与 `module_id`），写入放行 `cycle_id`；
- 进行中卡片的优先级页签，卡片与侧栏的行点击；
- G15、G19 的迭代页列表部分在三种引擎上有页面故事。

**Architecture：**
- 列表是项目列表加 `cycle_id`（spec R1）：`CycleIssues extends BaseIssuesStore`，绑定的 scope 仍是 `project`；成员判断与 `fields()` 认得迭代与模块（R2）。
- 偏好的筛选 store 抽出按页的基类，项目页与迭代页各给 service 与键（R3），不读 router store。
- 页面读路由参数；写入口按权限与迭代的状态显示；跟进经 `followInSession`，拒绝经 `useRefusalToast`/`useFormRefusal`。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/vitest/Playwright，不加依赖。

**Spec：** [P5b-cycle-list](../specs/P5b-cycle-list.md)（§2 的合同与 §3 的 R1–R11 是本计划的约束）；设计 [M6](../M6-design.md) §7、§8、§12；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：`core/store/issue/project/{issue.store.ts,filter.store.ts}`（及 `filter.store.test.ts`、`issue.quick-create.test.ts`）、`core/store/issue/helpers/base-issues.store.ts`（及 `base-issues.store.test.ts`）、`core/components/issues/issue-layouts/roots/project-layout-root.tsx`、`*/roots/project-root.tsx`、`core/components/issues/header.tsx`、`core/components/issues/filters.tsx`、`core/hooks/use-project-preferences-follow.ts`、`e2e/fixtures/issue-list-page.ts`、`e2e/fixtures/planning-lists.ts`、`e2e/stories/planning/g19-cycle-pages.spec.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p5b`，分支 `codex/m6-p5b`，基线 main `0b8efcd7`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/docs/test/refactor(M6/P5b)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 仍为 `M6/P5`；不延长任何 `until`。本段不改 `server/`、`api/`。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内；新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`；service 不转换数据；没有模块级的带令牌 service 实例；SWR 只经 `useSessionSWR`；新代码不读 router store（页面读路由参数）。
- 删掉不用的代码；不加开关、桩或兼容别名。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 53）；清掉告警的提交同时下调到实际值，新文件不得新增告警。
- **knip**：新文件在第一个读者出现的 Task 里建立，只给测试读的导出不算读者。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync`、`tools/i18n-keys.mjs`）。
- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例，跑 `node tools/keywords.mjs`。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败，再实现到通过。
- 成员判断、权限门（新建、移出只给能写的人）、只读（已结束、已归档）、迟到的应答（换会话、换迭代）、拒绝时说明，每个性质至少一个能让测试失败的变异，只计能编译的变异。
- 每个 Task 跑：`make lint-web`、`make test-web`（或聚焦 `cd web/apps/web && pnpm exec vitest run <files>`），涉及包的 `pnpm --filter <pkg> test`；改了导出时跑 `make knip`。

**回归测试的边界**
- 回归测试指 M2–M5、P1–P5a 的网页单测与 e2e。
- 只在 spec 说契约变化的地方改：`use-issue-layout-store.test.tsx` 中 `cycleId` 映射到 DEFAULT 的用例与 `useIssuesActions("CYCLE")` 被拒的用例；`use-issues.test.ts` 中 `useIssues("CYCLE")` 被拒的用例；`use-issue-layout-store.test.tsx` 中在别的项目新建的用例只改映射到的 store 类型（请求体仍为 `{name}`）；`R/cycles/(detail)/[cycleId]/page.test.tsx` 中正文只有转移横幅的部分；`active-cycle` 统计卡的页签数（两个变三个）；进度侧栏中“行不可点”的断言（迭代页上可点）。项目页的筛选 store 与列表的测试断言不改（R3 的回归保护）。其他断言不改。

**检查点与证据**
- 检查点：T10 结束时（引擎、偏好、页面、页头、快捷操作完成）串行完整七项，推送并核分支 CI（同时核 e2e 作业的实际时长）。T16 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准）。
- 证据放在 worktree 的 `.superpowers/sdd/P5b-cycle-list/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，调研笔记在 `research/`。实现代理先读同目录的 `implementer-brief.md`。

## Review Focus

1. **成员判断（R1、R2）**：迭代页的绑定只留本迭代的工作项；别处移入、移出、建立、删除后按 nerve 的回答增减；`fields()` 让别处的读标记成员可能已变。
2. **偏好（R3）**：基类重构后项目页行为不变；迭代页的读与修改的先后、整字段替换、被拒；不读 router store。
3. **会话与页面的范围（G19）**：挂载键带迭代 id；换迭代、后退前进、换会话时旧的回答不进新页面；优先级页签的绑定不替换本页列表（R7）。
4. **权限与只读（R5、R6）**：新建、移出的显示条件；已结束、已归档的只读；被拒时说明。
5. **一次发出（R4）**：本页新建只发一个带 `cycle_id` 的请求。
6. **守卫**：路由参数、告警上限、按本页分组的守卫。

## Task 1（T1）：P5a 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P5a-web-cycles-review.md`：抬头与 §7。
- `specs/P5a-web-cycles.md`：抬头与 A8；`plans/P5a-web-cycles.md`：抬头。
- `M6-design.md`：§14 的 P5a 行“已完成”、P5b 行“进行中”。
- `docs/v0/M8-open-release/handoffs/M6-closeout.md`：M6-M8-05 补一段。

- [ ] Step 1：按 P5a 的归档（主目录 `.superpowers/sdd/M6-P5a-integration-20261111`）里的 CI 收据与 `cleanup.log` 回填 §7：
  - **分支 CI**：`cbfb0d50322f80aab3f6a262376fb2ac000cf918` 的 42984392888 attempt 1：server (1) 130402002284、server (0) 130402002429、web 130402002583 成功，e2e 130408585582 失败（G18，§5）；修正后 `8175e76e258d50f3235a7e77b7c260eb5f9ec8fa` 的 42990196723 attempt 1 成功（web 130420905071、server (1) 130420905427、server (0) 130420905573、e2e 130427802649）。
  - **合入与 main CI**：noff `0b8efcd701edf7e73a37d5bb82ad176a7795b910`，双父 `a2b50624`、`8175e76e`，树与候选相同（`59fdf4e3`）；第一次推送 SSH 报“Could not read from remote repository”，再推一次成功；main CI 42995581409 attempt 1 成功（server (1) 130438746995、web 130438746998、server (0) 130438747043、e2e 130445895361）。
  - **归档**：`M6-P5a-integration-20261111` 1836 个文件、609,100,902 字节、未知 0、清单 sha256 `b9704d297a1c36b80137885bceb6bc40c2416d66501fcf707b82fc740fc5d9ea`；本段开工时复核 `archive.py verify`（收据 `t1-p5a-archive-verify`）。
  - **清理**：`cleanup.log` 记录分支已含于 origin/main，worktree `m6-p5a`、本地与远端分支 `codex/m6-p5a` 已删除，主目录停在 `0b8efcd7`。
  - 勾选 A8（注明本段的 `make test` 以分支 CI 为准）；阶段记录、spec、plan 的抬头改为已完成。
- [ ] Step 2：设计 §14 的 P5a 行改为“已完成”并附证据，P5b 行改为“进行中”，链接 P5b 的 spec 与 plan。M6-M8-05 补一段：P2 的根因修复（`pgtest/hostport.go`、`s3test/hostport.go` 的 `startPublished`/`hostPortTaken`）只认 Docker 拒绝启动的那一种（“programming external connectivity … address already in use”）；Docker Desktop 在主机一侧转发失败时容器照常启动，只表现为等映射端口超时，这层重试认不出；并记负责人 2026-11-11 把规则延续到 P5b–P7 中不改 Go 的阶段。
- [ ] Step 3：检查文档链接（`doc-links.py`）与关键词，然后提交 `docs(M6/P5b): 回填P5a的集成结果，进入P5b`。

## Task 2（T2）：成员判断与写入的请求体

**Files：** `core/lib/issue-membership.ts`（及测试）、`core/store/issue/root.store.ts`（`fields()`、`createIssue`、`updateIssue` 的白名单，及测试）、`core/store/issue/helpers/base-issues.store.test.ts`（加例）。

- [ ] Step 1（RED）：`issueMembership` 的用例：查询 `cycle_id: [X]` 时 `cycle_id` 为 X 的属于、为 Y 或 `null` 的不属于；`cycle_id: ["none"]` 时只有 `null` 的属于；`[X, "none"]` 两者都属于；`module_id` 同形（`module_ids` 与所选值相交，`none` 是空数组）。`fields()`：只改了 `cycle_id`（或 `module_ids`）的读让覆盖它的列表标为成员可能已变。`createIssue` 的请求体带上调用者给的 `cycle_id`；`updateIssue` 的请求体带上 `cycle_id`（含 `null`）。经 `acceptConfirmed` 的组合例：迭代页的绑定（`cycle_id: [X]`）在一个工作项移到 Y 的确认后不再列它，在别处建立的、`cycle_id` 为 X 的确认后列它。
- [ ] Step 2（GREEN）：实现；变异：去掉 `cycle_id` 的判断、去掉 `module_id` 的判断、`fields()` 去掉 `cycle_id`、白名单去掉 `cycle_id`，各至少一个测试失败。
- [ ] Step 3：`make lint-web`、聚焦与 `make test-web`；提交 `feat(M6/P5b): 列表的成员判断认得迭代与模块，写入放行 cycle_id`。

## Task 3（T3）：偏好的筛选 store 抽出基类

**Files：** `core/store/issue/helpers/page-preferences-filter.store.ts`（新，名字由实现定）、`core/store/issue/project/filter.store.ts`（改为继承）。

- [ ] Step 1：先跑项目页的既有测试（`filter.store.test.ts`、`base-issues.store.test.ts`、`issue.quick-create.test.ts`、`project-layout-root` 与页头的组件测试），记收据为基线。
- [ ] Step 2：把取数的先后、每字段的 `oneAtATime`、整字段替换与版本、`show` 的显示映射、看板折叠的本地值、`updateFilters` 的三种类型抽到基类；基类收 service（`get(id)`、`update(id, body)`，偏好的三个字段）、本地值的 store 类型与键。项目页只给 `ProjectPreferencesService`、`EIssuesStoreType.PROJECT` 与它读 router 的 `issueFilters`（不改）。基类的整字段修改不依赖已读的值：没有已读的值时也发出（只发该字段），以回答（完整的偏好）种下本地的值并升该字段的版本（spec §2.4，re-66 M1）；项目页现在“没有已读的值就抛错”的用例若有，按此契约变化改，并在台账写明。
- [ ] Step 3：既有测试断言不改全过（与基线同数）；基类自己的测试只补项目页测试没覆盖的分支（如有）。`make lint-web`、`make knip`、`make test-web`；提交 `refactor(M6/P5b): 偏好的筛选 store 抽出按页的基类，项目页行为不变`。

## Task 4（T4）：迭代页偏好的筛选 store

**Files：** `core/services/planning/cycle-preferences.service.ts`（新）、`core/store/issue/cycle/filter.store.ts`（新）及测试、`core/store/issue/root.store.ts`（建立并 `retire`）。

- [ ] Step 1（RED）：`CycleIssuesFilter` 按迭代 id：读到默认值（nerve 的默认）、显示映射同项目页；没读过的迭代整字段替换 `filters` 只发一个 `PATCH`，之后的读不盖掉它；修改一个字段只发该字段、确认后写回；被拒时不改、交给调用者；较旧的读不覆盖较新的修改；两个迭代互不影响；换代后旧一代的回答不写（`assertActive`）。没有读 router 的 getter（测试以不同的路由断言不变）。
- [ ] Step 2（GREEN）：实现（继承 T3 的基类，service 调 `getCyclePreferences`/`updateCyclePreferences`）。变异：较旧的读覆盖、整字段改为合并、换代后照写，各失败。
- [ ] Step 3：`make lint-web`、`make test-web`；knip 若报新文件无读者，把建立放到 T5 与 store 一起（第一个读者）。提交 `feat(M6/P5b): 迭代页偏好的筛选 store`。

## Task 5（T5）：列表引擎的 CYCLE

**Files：** `web/packages/types/src/issues/issue.ts`（`EIssuesStoreType.CYCLE`）、`web/packages/constants/src/issue/{filter.ts,common.ts}`（迭代页的显示选项常量，同一 Task 由 `ISSUE_STORE_TO_FILTERS_MAP[CYCLE]` 引用，免 knip 报未用导出；`TCreateModalStoreTypes`）、`core/store/issue/cycle/issue.store.ts`（新）、`core/store/issue/root.store.ts`（建立 `cycleIssues` 与 `cycleIssuesFilter`）、`core/hooks/store/use-issues.ts`、`core/hooks/use-issues-actions.tsx`、`core/hooks/use-issue-layout-store.ts`、`core/hooks/use-issue-list-mount.ts`。

- [ ] Step 1（RED）：`CycleIssues.fetchIssues` 的查询是本页偏好得出的查询加 `cycle_id: [id]`，绑定的 scope 带它；两个迭代是两个绑定；`createIssue` 在目标项目是迭代的项目时带本迭代的 `cycle_id`，别的项目时不带（spec §2.5，re-66 M3）；`useIssues(CYCLE)`、`useIssuesActions(CYCLE)` 给出迭代页的 store 与动作（动作经路由参数的 `cycleId`）；迭代页路由映射到 CYCLE；`useIssueListMount` 的键带迭代 id。
- [ ] Step 2（GREEN）：实现；按回归边界改 `use-issue-layout-store.test.tsx` 中 `cycleId` 的用例（只改映射到的 store 类型；在别的项目新建时请求体仍恰为 `{name}`）与 `useIssuesActions("CYCLE")`、`use-issues.test.ts` 中 `useIssues("CYCLE")` 被拒的用例。变异：挂载键去掉迭代 id、`createIssue` 不带 `cycle_id`、别的项目也带 `cycle_id`、查询不带 `cycle_id`，各失败。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`；提交 `feat(M6/P5b): 列表引擎的迭代页 store`。

## Task 6（T6）：读本页筛选、拖拽与日历

**Files：** 按 store 类型给出本页筛选的 hook（`core/hooks/store/` 下，名字由实现定）；四个 base root（`list/base-list-root.tsx:56`、`kanban/base-kanban-root.tsx:71`、`spreadsheet/base-spreadsheet-root.tsx:118-119`、`calendar/base-calendar-root.tsx:76`）；日历子树（`calendar/issue-block.tsx:132`、`calendar/calendar.tsx`（`viewFlags` 按 store 类型，现固定读 PROJECT）、`calendar/week-days.tsx:69-70`、`calendar/header.tsx:37`、`calendar/day-tile.tsx:73`、`calendar/dropdowns/months-dropdown.tsx:26`、`calendar/dropdowns/options-dropdown.tsx:59-85`）；`core/hooks/use-group-dragndrop.ts`（store 类型的联合与守卫）；`tools/keywords.json`（`project-address-route` 收入改写的日历文件与新 hook，带样例）。

- [ ] Step 1（RED）：组件测试：CYCLE 时四个 base root 与日历子树读的是本页（路由参数的 `cycleId`）的筛选；迭代页的日历按本页偏好显示周视图与周末，周/月与周末的开关写进本页偏好；日历的快速新建读 CYCLE 的 `viewFlags`；PROJECT、PROFILE、ARCHIVED 的读法与行为不变（既有测试断言不改）；拖拽认得 CYCLE。
- [ ] Step 2（GREEN）：实现：`CalendarChart` 不再拿整个筛选 store，改拿已解析的日历显示设置与更新回调（re-66 H2）。变异：CYCLE 时读 router 驱动的 getter、日历的开关写到项目偏好，各失败。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `refactor(M6/P5b): 列表与日历按 store 类型读本页的筛选`。

## Task 7（T7）：base root 的只读

**Files：** 四个 base root（加一个只读的 prop，并进 `disableIssueCreation`、日历的 `readOnly` 与 `canEditProperties`）及其测试。

- [ ] Step 1（RED）：只读时：没有快速新建、分组头的“+”、行内属性修改（下拉框停用）、拖拽；快捷操作只读；日历只读。不只读时行为同现在（项目页的既有测试断言不改）。
- [ ] Step 2（GREEN）：实现；变异：只读不并进 `disableIssueCreation`、日历忽略只读、拖拽忽略只读，各失败。
- [ ] Step 3：`make lint-web`、`make test-web`；提交 `feat(M6/P5b): 列表的四种布局可以整体只读`。

## Task 8（T8）：迭代页的布局根、四种布局与空态

**Files：** `core/components/issues/issue-layouts/roots/cycle-layout-root.tsx`（新）、`{list,kanban,calendar,spreadsheet}/roots/cycle-root.tsx`（新）、`empty-states/cycle.tsx`（新）与 `empty-states/index.tsx`、`R/cycles/(detail)/[cycleId]/page.tsx`（正文加列表与迭代关闭的分支）、`core/hooks/use-project-preferences-follow.ts`（按 store 类型与 id，名字由实现定）、`tools/keywords.json`（`project-address-route` 收入 `roots/cycle-layout-root.tsx`、`calendar/roots/cycle-root.tsx`，带样例）、i18n（如有新文案，en 与 zh-CN 同一提交）。

- [ ] Step 1（RED）：组件测试：
  - 迭代页的正文在转移横幅之下渲染列表根，`IssuesStoreContext` 为 CYCLE；预览的宿主是迭代页自己的（`hostScope` 带迭代 id）；
  - 四种布局按本页偏好切换；本页偏好取数失败时说明并可重试；偏好修改确认后列表按现有分页重取（同项目页的跟进规则）；
  - 迭代关闭（`cycle_view` 为假）时列表区是停用空态，没有 `/me/cycles/{id}/preferences` 与 `/projects/{id}/issues` 的请求（re-66 H1）；
  - 空态：未结束能新建的人有“新建工作项”、有筛选时“清除筛选”、已结束或已归档只有说明；
  - 已结束、已归档时四种布局只读（T7 的 prop）；访客看得到列表、没有新建。
- [ ] Step 2（GREEN）：实现；按回归边界改 `[cycleId]/page.test.tsx` 的正文部分。变异：只读的条件去掉已结束、去掉已归档、空态对已结束仍给操作、迭代关闭时仍取列表，各失败。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；告警上限按实际下调（如有）。提交 `feat(M6/P5b): 迭代页的工作项列表、空态与只读`。

## Task 9（T9）：页头与新建

**Files：** `R/cycles/(detail)/header.tsx`、`core/components/issues/filters.tsx`（按 store 类型与 id）、`core/components/issues/issue-modal/base.tsx`（CYCLE 的动作带 `cycle_id`；迭代页页头打开时锁定项目）、`core/components/issues/issue-layouts/quick-add/root.tsx`（如需）、`modals/work-item-level.tsx`、`tools/keywords.json`（收入 `filters.tsx`、`issue-modal/base.tsx`）。

- [ ] Step 1（RED）：页头的数目、四种布局、筛选开关、显示（显示选项用迭代页的常量，不含迭代，R9 的守卫测试）；迭代关闭时这些都不显示；“新建工作项”只给能建的人且迭代打开、未结束、未归档；新建弹窗（项目锁定）、快速新建、分组头的“+”各只发一个带 `cycle_id` 的创建请求，确认后提示、行出现在本页；被拒（`planning.cycle_completed`、`planning.cycles_disabled` 等）时说明、不提示成功（测试不断言拒绝码之间的次序，re-66 L7）。
- [ ] Step 2（GREEN）：实现；变异：新建的权限门去掉、迭代状态条件去掉、请求体不带 `cycle_id`、弹窗不锁项目，各失败。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P5b): 迭代页的页头与本页新建`。

## Task 10（T10）：迭代页的快捷操作与写后重读进度（检查点）

**Files：** 迭代页行的快捷操作（`quick-action-dropdowns/cycle-issue.tsx` 新，规则已列这个路径；修改与“复制一份”的弹窗以 CYCLE 打开，re-66 L4）、CYCLE 的动作在确认后重取 `/progress`（`core/hooks/use-issues-actions.tsx` 的迭代分支或同等位置）、i18n（“移出迭代”等，en 与 zh-CN 同一提交）。

- [ ] Step 1（RED）：“移出迭代”只给能改工作项的人、列表不只读、迭代打开时显示；经 `updateIssue` 发 `cycle_id: null`，确认后行离开本页列表并提示，被拒时说明、行留着；“复制一份”的请求带本迭代的 `cycle_id`；迭代页列表的动作（新建、修改、移出、删除、归档、拖拽）确认后重取本迭代的 `/progress` 一次，不推算；迟到的确认（已换迭代、换会话）不提示、不重取新页面的进度。
- [ ] Step 2（GREEN）：实现；变异：移出的权限门去掉、只读时仍给、确认后不重取进度、迟到的确认仍重取，各失败。
- [ ] Step 3（检查点）：`sh .superpowers/sdd/P5b-cycle-list/gates.sh t10-gate`（七项串行；`make test` 照负责人的规则），推送，核分支 CI 与 e2e 作业时长；提交 `feat(M6/P5b): 迭代页的快捷操作与写后重读进度`。

## Task 11（T11）：进行中卡片的优先级页签

**Files：** `core/components/cycles/active-cycle/cycle-stats.tsx`（三个页签）、优先级页签的组件、它的 store（`core/store/issue/cycle/priority-issues.store.ts` 或同等）与 `core/store/issue/root.store.ts`（构造时建一个，随根退役，re-66 M4）、i18n（页签名与空态，en 与 zh-CN 同一提交）。

- [ ] Step 1（RED）：页签次序为优先级、负责人、标签，默认负责人；优先级页签的请求是显式的完整查询（`cycle_id`、`priority=urgent,high`、`order_by=-created_at`、`sub_issue=true`、`limit=30`），不含任何偏好；滚到底按游标续页；迭代页的当前绑定不因它改变；卡片挂载两次只登记一个绑定；行显示编号、名字、优先级、只读的状态、目标日期；空态与加载；换会话后旧的回答不写。
- [ ] Step 2（GREEN）：实现；变异：查询并入偏好、用迭代页列表的 store 实例（替换当前绑定）、按组件建实例、不续页，各失败。按回归边界改统计卡的页签数。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`；提交 `feat(M6/P5b): 进行中卡片的优先级页签`。

## Task 12（T12）：卡片与侧栏的行点击

**Files：** `core/components/core/sidebar/single-progress-stats.tsx`（可点、已选）、`core/components/cycles/distribution-stats.tsx`、`active-cycle/{progress.tsx,cycle-stats.tsx}`、`progress-sidebar/progress-stats.tsx`、`core/components/core/sidebar/progress-stats/shared.ts`（切换的帮助函数）、`tools/keywords.json`（收入 `single-progress-stats.tsx`）。

- [ ] Step 1（RED）：
  - 卡片的状态组行（数大于 0）、负责人行、标签行、优先级行点击后，该迭代的页面筛选换成这一个条件（经 `updateFilterExpressionFromConditions`，没有实例时经偏好的整字段替换），nerve 确认后进入迭代页；未读过偏好的迭代只发一个 `PATCH`，进入后的读不盖掉它（re-66 M1）；被拒时提示、不跳转；“无负责人”“无标签”行不可点；
  - 优先级行：进入迭代页后在它的预览宿主下打开该工作项的预览；
  - 迭代页侧栏的行切换该值（加入、去掉、条件空了去掉条件），行说明是否已选；列表页预览里的侧栏行不可点；行是以行命名的按钮；
  - 迟到的确认（已离开列表页、换会话）不跳转。
- [ ] Step 2（GREEN）：实现；变异：卡片改为合并而非替换、确认前就跳转、先读后写、读覆盖了写、预览里可点、侧栏改为替换，各失败。按回归边界改侧栏“行不可点”的断言。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P5b): 进行中卡片与侧栏的行设迭代页的筛选`。

## Task 13（T13）：页面故事一（G15）

**Files：** `e2e/stories/planning/g15-cycle-list-page.spec.ts`（新）、夹具（扩展 `e2e/fixtures/planning-lists.ts` 或新建 `cycle-list-pages.ts`）、`e2e/playwright.m6-pages.config.ts`（如需）。

- [ ] Step 1：故事（先 RED：在未实现的提交上或以变异核对它会失败，留收据）：
  - 迭代页的列表在四种布局里只列本迭代的工作项（另一个迭代的、没有迭代的都不出现），已归档迭代的页面同样列出且只读；迭代关闭时列表区是停用空态、没有列表与偏好的请求；
  - 续页按 M4 的双向分页（`expectPages`、`scrollToLastListRow` 的既有写法：不重复、不丢、空页）；按状态分组时改一行的状态，两组的数随 nerve 的回答变化（跨分组写后计数）；在迭代页归档一行，行离开、数目随 nerve 变化（归档范围）；
  - 本页偏好：改布局、分组、显示属性后重载仍在，项目页的偏好不变；
  - 页头新建与快速新建各只发一个带 `cycle_id` 的请求，行出现；移出迭代后行离开、数据库里没有该成员；已结束迭代的页面没有这些入口；
  - 附件列：两个工作项各有真实附件（1 个、3 个），表格的附件列显示数目，按附件数排序、续页，换会话后照常；增删附件后数目随读取更新（M5-M6-02）；
  - 每种布局以 `expectListPageQuiet` 收尾（整页零非预期失败）。
- [ ] Step 2：chromium 跑通；三种引擎的页面配置跑通；新故事 `--repeat-each=3`；`make lint-web`。提交 `test(M6/P5b): 迭代页列表的页面故事（G15）`。

## Task 14（T14）：页面故事二（G19 与行点击）

**Files：** `e2e/stories/planning/g19-cycle-list-page.spec.ts`（新）或扩展 `g19-cycle-pages.spec.ts`。

- [ ] Step 1：故事（先 RED）：
  - 换迭代（面包屑）、后退前进、换项目、换会话时，旧迭代的列表与迟到的回答不进新页面（以延迟的列表回答核对）；
  - 失权的迟到回答：列表回答被扣住时经接口把调用者移出项目，放行后页面不显示那一页、不报非预期失败；
  - 只看自己的访客、工作区管理员兼访客只看到自己建立的工作项与数目，没有新建与移出；
  - 进行中卡片：优先级页签的请求与续页；点负责人行后迭代页按他筛选（重载仍在），点优先级行后迭代页按紧急与高筛选并打开预览；侧栏的行切换筛选；
  - 键盘：页头的布局、筛选、显示与新建按钮可用 Tab 到达并以名字命名。
- [ ] Step 2：chromium、三种引擎、`--repeat-each=3`；`make lint-web`。提交 `test(M6/P5b): 迭代页列表的页面故事（G19、行点击）`。

## Task 15（T15）：文档与交接回执

- [ ] Step 1：输入交接的“处理结果（M6/P5b）”：M4-closeout（M6-01 的列表部分、M6-02 的迭代部分：列表在 P5b 先开放，“添加已有”、日历 `addIssuesToView`、`existing-issues-list-modal.tsx` 的调用方依设计 §12 的 P7 一行与 §15 第 12 条在 P7）、M5-closeout（M5-M6-02 的迭代部分）、M3-closeout（§1 迭代的 user-properties）；均保持 open（设计 §11）。
- [ ] Step 2：差异清单（迭代页列表的请求、一次新建、已结束与已归档的只读、迭代关闭时的列表区、行点击、优先级页签的查询与续页、写后重读进度、布局数）；前端改动清单（M6 一行、优先级页签一行、新增的列表 store、偏好基类、读本页筛选的 hook）；设计 §8/§12 如需；spec §5 的后续补全。
- [ ] Step 3：`doc-links.py`、关键词；提交 `docs(M6/P5b): 交接回执、差异清单与前端改动清单`。

## Task 16（T16）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t16-gate`，留收据；`make test` 照负责人的规则），另以三种引擎的页面配置跑一次（收据），推送准确的 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账；
  - 写阶段记录 `reviews/P5b-cycle-list-review.md`；
  - 勾选 spec §4；A8 待合入后由 P6 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
