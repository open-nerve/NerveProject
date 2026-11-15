# M6 / P7c 工作项一侧（三）——命令面板、移动端页头与遗留实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：实现完成（T1–T14，2026-11-16）；T15 进行中（主控）：`5087b09e` 的第一次门禁 `t15-gate` 与整分支审查 [re-77](../../../reviews/re-77-M6P7c整体审核.md)（Ready to merge after fixes：I1、M1、M2）已完成，修复轮 F（F2 `98d9f985`、F3 `10ca6ae6`、F1 `165b252b`；F4 写记录）已完成、另经限定复审（re-77 §4），待最终候选的门禁、推送、分支 CI 与合入（[阶段记录](../reviews/P7c-palette-mobile-review.md)）。T1–T13 的提交：`b5e1c615`（T1）、`87dbb973`（T2）、`3151dd9b`、`f5ef9548`（T3）、`df04b1bc`（T4）、`74381080`（T5）、`6c41721c`（T6）、`165420ef`、`77ac77a4`（T7，前者是先于 RED 的纯搬移）、`03b85a6e`（T8）、`91272fc2`（T9）、`d96ed7b2`（T10，检查点：本机七项中六项通过、build+e2e 一条以 M6-M8-07 的签名失败而只重跑该文件即过，分支 CI 43742488275 全绿，e2e 作业 19 分 06 秒）、`5e5f8ee8`（T11）、`fb5c345d`（T12）、`3b5b0a78`（T13），T14 是其后的文档提交；另有预检的修正 `78fa4dbc` 与 F8 的事实更正 `3774b98e`；逐步的收据、变异与裁定在台账 `progress.md`。spec 与 plan 已写（2026-11-15），预检通过（修正后，[re-76](../../../reviews/re-76-M6P7c预检.md)：F1–F16 已改入），T1 已完成（`b5e1c615`）。基线 main `d052213f82aa4fba234f5684722335e1a5cbb5be`（P7b 的 noff，双父 `f42f9d64`、`4fe308ce`；P7b 的最终分支 CI 43702923005 全绿，e2e 作业 27 分 50 秒；合入后的 main CI 43706438924 attempt 1 全绿，e2e 作业 23 分 42 秒；归档 `.superpowers/sdd/M6-P7b-integration-20261115/`；T1 核对后回填）。本计划只管 P7c（P7 的第三段，[P7a spec](../specs/P7a-work-item-planning.md) R1）；主控在写 spec 之前的裁定 H1–H16 是 spec 的 R1–R16。

**Goal：**
- 命令面板：先拆导航的命令；规划入口一条规则，打开的菜单不再转圈；菜单的空文案经键；搜索有项目范围的迭代、模块结果；工作项有“改迭代”“改模块”两个命令；
- 快捷操作的“恢复”在途时禁用，归档页的行菜单不再收两个空的设置函数；
- 一个共用的移动端页头（项目页改用它），迭代页、模块页接上，两页的窄屏顶行一条规则；
- 分组时页头的数目、没有读者的键、替身的 `opened`、“添加已有”打开即聚焦搜索框；设计 §8 的四个视图文件；
- G19 的工作项一侧的页面故事在三种引擎，CI 的 e2e 时长在报告线以内（或越过之前停下报告负责人）。

**Architecture：**
- 入口规则与列表读不了的原因是 `core/lib/planning-switch.ts` 里的纯函数；命令、搜索、菜单都调它们，角色以 slug 与项目取（不经路由）。
- 工作项命令的写在命令面板既有的守卫之内，经 P7a 的 `IssueChange` 与 `changeBody`（模块集合在写队列轮到时以所持的算出）。
- 移动端页头与 `HeaderFilters` 同一组按 store 类型的读写；页头的数目按范围读在 store 一侧。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/vitest/Playwright/cmdk，不加依赖。

**Spec：** [P7c-palette-mobile](../specs/P7c-palette-mobile.md)（§2 的合同与 §3 的 R1–R16 是本计划的约束）；设计 [M6](../M6-design.md) §2 G19、§8、§12；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：P7b 的 [spec](../specs/P7b-list-vocabulary.md)、[plan](P7b-list-vocabulary.md) 与[阶段记录](../reviews/P7b-list-vocabulary-review.md) §5 的教训；代码的模板：命令面板工作项的命令与它的表格式单测（`PK/ui/pages/context-based/work-item/mutation-commands.ts`、`commands.test.tsx`）、两个下拉框（`core/components/dropdowns/{cycle,module}/base.tsx`）、`HeaderFilters`（`core/components/issues/filters.tsx`）、`pageTotal`（`core/store/issue/helpers/planning-page-issues.store.ts`）、表单的 `data-autofocus`（`core/components/cycles/form.tsx:133`）、快捷操作的 `followInSession`（`core/components/cycles/quick-actions.tsx:80-93`）；e2e 的 `e2e/fixtures/{power-k,planning-list-pages,settings-pages,cycle-pages,dialogs}.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p7c`，分支 `codex/m6-p7c`，基线 main `d052213f`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/refactor/test/docs(M6/P7c)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送由主控用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 仍为 `M6/P7`（三段共用，T1 核对没有到期的例外）；不延长任何 `until`。本段不改 `server/`、`api/`（R15）。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内（R16）：`navigation/commands.ts`（466）在 T2 先拆再改；`fake-planning-page.tsx`（378）在 T11 只加几行；`e2e/fixtures/planning-list-pages.ts`（359）、`cycle-pages.ts`（388）不加东西（新的帮手放在故事文件或新的夹具里）。新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写或拆出的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（`IssueChange` 由 `IssueUpdate` 推出）；SWR 只经 `useSessionSWR`；新代码不读 router store（页面读路由参数）。
- 删掉不用的代码；不加开关、桩或兼容别名（项目页移动端页头手写的读写删去、文件改写为薄包装件；`CYCLE_STATUS[].i18n_label` 删去；空的设置函数删去）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 44）；清掉告警的提交同时下调，新文件不得新增告警；本段改过的文件以 oxlint 0 告警为目标（M1 交接，T14 收据）。
- **knip**：新文件在第一个产品读者出现的 Task 里建立；只给测试读的导出不算读者（本阶段更严的规则，逐 Task 的审查执行）；删掉某件最后一个外部读者的 Task 同时删去它或收回导出。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀）；本段的键见 spec §2.11。
- **`project-address-route`**：新建与改写的组件、hook 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例），跑 `node tools/keywords.mjs`；被删的路径留在表里阻止复活。
- **`.oxlintrc.json`**：用 `useSessionSWR` 的新文件随它的 Task 加入 swr 限制的文件表。只收紧。
- **共用的假件**：规划 store 的数据用 `core/store/planning/fake-*.ts` 与 `fake-projects.ts`；规划页的帮手从 `issue-layouts/roots/fake-planning-page.tsx` 导入；权限用 `core/store/user/fake-permissions.ts`；命令面板的工作项命令测试用 `issueStoreTab`（fake nerve）——导入，不复制。

**构建与类型**
- 改了 `@nerve/types`、`@nerve/constants`、`@nerve/utils`、`@nerve/i18n` 之后，先构建该包（`pnpm --filter <pkg> build`）再跑应用的类型检查；改了路由文件之后先跑 `react-router typegen`（`make knip` 已含）。
- 变异只计能编译的；类型检查不过的变异不算失败（以类型测试与 `tsc` 的收据记下）；e2e 的变异另要能构建。

**测试纪律**
- 每个 Task 先写会失败的测试，看到它因所测性质失败（不是编译失败），再实现到通过；纯搬移（T2）先以特征测试与既有测试的收据为基线，之后同数。
- 下列性质每个至少一个能让测试失败的变异（名字见各 Task）：入口规则（访客、归档的项目、开关关闭不给）；打开的菜单门为假时不转圈；空文案读键；搜索的组只在项目范围、本地过滤、不给访客；改迭代不给已结束的、选所持的不发、请求体；改模块每次一个增减、在轮到时算、已归档的只可移出；恢复在途时禁用；移动端页头偏好读到之前不给；页头的数目分组时显示、别的项目时不显示；替身关上即清空；“添加已有”打开即聚焦、移动端不聚焦；视图写之后换了会话不跟进。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出、删了文件或改了路由时跑 `make knip`；改了 `tools/keywords.json` 时跑 `node tools/keywords.mjs`。
- 单测的 SWR 缓存：同一文件里读同一份键的组件测试各给自己的 `SWRConfig` 缓存（P7a F1 的教训）。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P7b 的网页单测与 e2e。
- **拆分（T2）**：没有单测直接执行导航的命令（写 plan 时 grep `usePowerKNavigationCommandsRecord|config/navigation` 只有 `config/commands.ts`、`navigation/root.ts`）；T2 先加特征测试（命令的 id、`keySequence`、`page`、`group` 与 `usePowerKNavigationCommands` 的次序，加几种上下文下的显示：工作区页、项目页的 ADMIN、GUEST、开关关闭、**项目已归档**），基线后再搬，之后同数；`PK/` 下既有的测试逐文件同名同数。
- **入口规则（T3）**：`open-entity/project-cycles-menu.test.tsx`、`project-modules-menu.test.tsx` 的既有三例（`:121` 列出与打开、`:134` 共用外壳的读、`:139` 拒绝结束等待）与 `:147` 的 `it.each` 三行（开关关闭、已归档、不是成员：门为假时什么都不读）一行不改，T3 的“说原因、不转圈”加在这三行旁（re-76 F6）；T2 的特征测试中归档项目的那一格按契约改（台账写明）。
- **空文案（T4）**：菜单的既有测试若断言英文的空文案，按契约改为翻译后的文案（台账逐条写明）。
- **搜索（T5）**：`PK/ui/modal/search-menu.test.tsx` 既有五例一行不改。
- **工作项命令（T6、T7）**：`work-item/commands.test.tsx` 的既有表格行与守卫的例一行不改，新命令加行。
- **快捷操作（T8）**：`cycles/actions-host.test.tsx`、`modules/actions-host.changes.test.tsx` 的“只发一次”（`:225`、`:243`、`:251`；`:228`、`:246`、`:254`）与两个 `quick-actions.test.tsx` 一行不改；归档页的 `menus.test.tsx`、`roots/archive-page.test.tsx` 不改。
- **移动端页头（T9、T10）**：`issue-layouts/filters/preferences-controls.test.tsx` 的四个移动端的例（`:228` 的 `it.each` 三行与 `:251` 一例，帮手 `mountMobile` 在 `:218`；re-76 F5）断言不改，只在挂载的帮手要变时改它（项目页的件仍是 `R/issues/(list)/mobile-header.tsx`，改写为包装件），基线同数 4；两页页头的测试（`R/cycles/(detail)/header.test.tsx`、`R/modules/(detail)/header.test.tsx`）既有的例一行不改（迭代页的开关、快捷操作在各宽度都给只加例）。e2e 的 C2（`e2e/stories/issue/filter-archive-profile-interactions.spec.ts:89`）、C12 review（`review-boundaries.spec.ts:158`）一行不改，是项目页移动端页头的回归保护。
- **页头的数目（T11）**：`core/components/issues/header.test.tsx` 既有的例一行不改；项目页 store 的既有测试一行不改。
- **焦点（T11）**：`existing-issues-list-modal.test.tsx` 既有的例一行不改；P7b 的 G4 (page)（`g4-add-existing-page.spec.ts`）先 `field.focus()` 的一步不改（仍过）。
- **视图（T12）**：视图的既有测试（若有）一行不改。
- 其他断言不改。

**检查点与证据**
- 检查点：T10 结束时（命令面板、快捷操作、移动端页头完成）串行完整七项，主控推送并核分支 CI，记录 e2e 作业的实际时长（期限 35 分钟；超过 30 分钟先报告负责人，不改期限或作业结构）；这时还没有本段的页面故事，这个数是 T15 的校准点（spec §2.12）。T13 记下新故事文件在本机三个引擎的时长；T15 推送之前照 spec §2.12 算估算、上界与校准点。T15 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准；M6-M8-03、-07 的签名照各条只重跑那一个文件一次）。
- 证据放在 worktree 的 `.superpowers/sdd/P7c-palette-mobile/`：台账 `progress.md`，收据经 `run-check.py`（`python3 .superpowers/sdd/P7c-palette-mobile/run-check.py <label> <命令…>`，在 worktree 的根跑），变异经 `mutate.py`（`python3 .superpowers/sdd/P7c-palette-mobile/mutate.py <label> <文件> <旧> <新> [<旧> <新> …] -- <命令…>`），七项门禁经 `gates.sh`，最终候选经 `final.sh`，文档链接经 `doc-links.py`，时长经 `durations.py`，归档经 `archive.py`，调研笔记在 `research/`，主控的裁定在 `controller-rulings.md`。实现代理先读同目录的 `implementer-brief.md`。
- 台账的行：`Task N: Step k RED/GREEN — <label> exit <code>（…）`、`Task N: 基线 — <label> exit <code>（文件与数）`、`Task N: 变异 — <label>（…）→ 失败（哪个测试），restored=yes`、`Task N: Ruling: <决定> — <理由> — 代价：<代价>`、`Task N: complete (commits <first>..<last>, tests: <summary>)`。

## Review Focus

1. **入口规则（R2）**：一个纯函数、一个原因函数，角色以 slug 与项目取；四个导航命令与搜索的两组调它；打开的菜单门为假时说原因、不转圈；侧栏不改。
2. **拆分（R3）**：只搬不改，特征测试前后同数；项目一级一件进守卫，工作区一级不进。
3. **工作项命令（R6）**：选项与下拉框同一规则（`cycleOptions`、`moduleOptions`）；写经 `IssueChange` 的增减、在守卫之内、在轮到时算；类型让整集合赋不进去；`shift+c`、`shift+m` 不与既有的冲突。
4. **移动端页头（R8、R9）**：共用的件只读写按 store 类型的那一组；门在每页的 `observer` 包装件里算、偏好读到之前不给；项目页手写的读写删去、既有测试同数（4）；迭代页的开关与快捷操作各宽度都给。
5. **遗留（R10、R11）**：页头的数目按范围读在 store 一侧；键与字段删净；替身只清自己；聚焦只在非移动端。
6. **X1（R12）**：四个文件的地址读不经路由；视图写之后的跟进经 `followInSession`；没有动 M7 的东西。
7. **e2e（R13、R14）**：三条故事合计 ≤ 8 秒；写法照 P7b 记录 §5 的教训；变异落在被证的一步。
8. **质量（R16）**：行数、knip、告警、i18n、守卫；只给测试读的导出。

## Task 1（T1）：P7b 回填与 e2e 余量的重算

已完成（`b5e1c615`，台账 Task 1 的各行）。

- [x] 以归档核对 P7b 的事实：`/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P7b-integration-20261115/`（`manifest.json`、`cleanup.log`、`payload/worktree/.superpowers/sdd/P7b-list-vocabulary/progress.md`）经 `archive.py verify`（收据 `t1-p7b-archive-verify`；主控记的是 3964 个文件、920,634,424 字节、5216 个成员、未知 0、清单 sha256 `7a17a397fa181c56c3d46425d07783d2b689ec482e8f6c0f2cb041eca582799c`、head `4fe308ce`）；两次 CI 以公开的 jobs 接口（只读：`https://api.github.com/repos/open-nerve/NerveProject/actions/runs/<id>/jobs`，43702923005 与 43706438924，收据 `t1-ci-jobs`）核对作业号、时刻、attempt 与结论；e2e 的条数与逐条时长以作业日志核对（可得时，收据 `t1-e2e-job-logs`）；合并的双父与树以 `git rev-parse d052213f^1 d052213f^2 d052213f^{tree} 4fe308ce^{tree}`（`t1-merge-tree`）。原件与台账不一致、归档缺失或 main CI 不绿时停下报告。
- [x] 回填 P7b 阶段记录 §7（体例同 P7a 记录 §7：最终 HEAD 的分支 CI 与估算、上界的对照，合入与 main CI，归档，清理，接续指向本段）、P7b spec A15 勾选与状态行、P7b plan 的状态行、设计 §14 的 P7b 一行改为已完成、P7c 一行改为进行中（链接本 spec 与 plan）、设计 §3 状态行。
- [x] [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 记 P7b 的两个数据点（分支 CI 1670 秒是至今最慢的，main CI 1422 秒），按 spec §2.12 重算余量（台账写算式；关系一支的等待约 0.9 秒算作故事时长，余量 14.6–19.3 秒）；P7b 的 `t1-…` 与本段的收据各写在台账。
- [x] 关键词 phase 仍为 `M6/P7`，核对 `tools/keywords.json` 没有在本段到期的例外（`node tools/keywords.mjs`，收据 `t1-keywords`）；文档链接（`t1-doc-links`）。
- [x] 提交 `docs(M6/P7c)`。

## Task 2（T2）：先拆 `navigation/commands.ts`（R3，spec §2.1）

- [ ] oxlint 基线（re-76 F15）：本段要改的文件（plan 各 Task 点名的，含 T12 的视图文件）逐文件的告警数（收据 `t2-oxlint-baseline`），有告警的文件在它的 Task 里清掉并同时下调上限。
- [ ] 特征测试：`PK/config/navigation/commands.test.ts(x)`（新）以真实的权限 store 替身（`fake-permissions.ts` 的 `callerInWeb(workspaceRole, projectRole, guestViewAll, fields)`，`fields` 可给 `archived_at`、`cycle_view`）、`useUser` 的 mock（回 `{ data, permission }`，`config/creation/command.test.tsx:16` 的写法）与项目 store 替身给出几种上下文（工作区页、项目页的 ADMIN、GUEST、开关关闭、项目已归档），核对 `usePowerKNavigationCommands` 的命令次序与每个的 `id`、`keySequence`、`page`、`group`，与各上下文下 `isVisible`/`isEnabled` 的答（显式的期望表，不用快照）。收据 `t2-baseline-characterize`。
- [ ] 按层级拆为工作区一级与项目一级两件（文件名由本 Task 定，例如 `workspace-commands.ts`、`project-commands.ts`，`commands.ts` 只组合两者与键的类型，或删去而由 `root.ts` 组合），只搬不改（一拆为二 `git diff -M` 显示不出，以 `git diff --color-moved` 或命令块的行多重集比较证明，台账记 `t2-moved-lines`；新的只有胶水：一个 hook 变两个、各自的 `useUser()`/`useProject()`、两级都用的 `baseProjectConditions` 的去处、键的联合类型拆开——台账写明这是唯一不是搬来的代码，re-76 F11），各在约 400 行以内；特征测试与 `PK/` 下的既有测试同数（`t2-after`）。
- [ ] 项目一级一件加入 `project-address-route`（样例），工作区一级一件不加（台账写理由）；`node tools/keywords.mjs`、`make lint-web`、`make knip`。
- [ ] 提交 `refactor(M6/P7c)`。

## Task 3（T3）：规划入口一条规则与菜单的说明（R2，spec §2.2）

- [ ] RED：`core/lib/planning-switch.test.ts` 的入口规则（ADMIN、MEMBER、工作区管理员而项目访客（映射为 ADMIN）给；GUEST、没有角色、项目已归档、开关关闭不给；两域）与原因函数（各一码）；T2 的特征测试中归档项目的那一格改为“四个导航命令不给”；两个打开的菜单在 `:147` 的 `it.each` 旁各加：门为假时说出具体的键（`errors.project_archived`、`errors.planning_cycles_disabled` / `planning_modules_disabled`）、不转圈；store 里持有这一域的列表而开关已关时也说原因、不列（re-76 F10）。
- [ ] GREEN：两个纯函数（`planning-switch.ts`）；四个命令改调入口规则（角色以 `getProjectRoleByWorkspaceSlugAndProjectId(ctx.params.workspaceSlug, ctx.params.projectId)`）；两个菜单门为假时经原因函数与 `problemMessageKey(code)`（`core/lib/error-messages.ts:129`；不是 `errorMessageKey`，re-76 F3）说明，原因优先于持有的列表。
- [ ] 变异：`t3-mut-archived-entry`（入口规则不看归档）、`t3-mut-guest-entry`（不看角色）、`t3-mut-spinner`（门为假时回到转圈）、`t3-mut-held-list`（门为假而持有列表时照列）各让测试失败。
- [ ] 提交 `feat(M6/P7c)`。

## Task 4（T4）：菜单的空文案（R4，spec §2.3）

- [ ] RED：构造件的单测（没有项时显示翻译后的文案；默认读 `power_k.search_menu.no_results`）；类型测试：给字面量编译不过（`@ts-expect-error`）。
- [ ] GREEN：构造件收键；七个菜单各给新键（en 与 zh-CN）；`empty-state.tsx` 的默认读键；§2.2 的说明给已翻译的文字（`PowerKMenuEmptyState` 收文字）。
- [ ] 变异：`t4-mut-literal`（构造件直接显示键名或英文）让测试失败。
- [ ] 提交 `feat(M6/P7c)`。

## Task 5（T5）：搜索的迭代、模块结果（R5，spec §2.4）

- [ ] RED：`search-menu.test.tsx` 加：项目范围下两组按名字本地过滤（大小写、子串）、含已结束的迭代、不含已归档的、行有项目标识、选中即关面板并转到；工作区范围没有两组；访客、归档的项目、开关关闭没有该组；store 没有时以同一个键读（`/browse/:workItem`）、读失败或还没回答时这一组不显示、不挡工作项与项目的结果；只有迭代一组有结果时不给“没有结果”，四组都空时才给（re-76 F9）。
- [ ] GREEN：搜索的件（`search-menu.tsx`、`search-results.tsx`，或新件）；读用 `useSessionSWR` 的文件加入 swr 限制表；守卫的文件表。
- [ ] 变异：`t5-mut-workspace-scope`（工作区范围也给）、`t5-mut-no-filter`（不过滤）、`t5-mut-guest`（不看入口规则）各让测试失败。
- [ ] 提交 `feat(M6/P7c)`。

## Task 6（T6）：工作项的“改迭代”（R6，spec §2.5）

- [ ] 选项的读已由预检定（re-76 F14，spec §2.5）：打开的菜单的门控 `useSessionSWR` 读加纯函数 `cycleOptions`，不用 `useCycleOptions`（它的 `load` 要挂载的 effect，会碰 `exhaustive-deps` 而上限恰为 44）。快捷键声明为 `modifierShortcut: "shift+c"`（带 Shift 的走 `handleModifierShortcut`，re-76 F8）。
- [ ] RED：`work-item/commands.test.tsx` 的表格加“改迭代”一行（选 Sprint → 恰为 `{cycle_id}`）、选“无迭代” → `{cycle_id: null}`、选所持的 → 不发请求；选项（“无迭代”在前、没有已结束与已归档的、所持的标为当前、所持的已结束的以名字显示为当前）；显示（不可改、开关关闭时不给）；守卫（换项目、换人、轮到时不可改不发，拒绝说原因）；`shift+c` 的快捷键（与既有的不冲突）。
- [ ] GREEN：页类型、页、菜单（`PK/menus/cycles.tsx` 若要给当前值就加 prop）、命令（`mutation-commands.ts` 加 `change(change: IssueChange)`）、占位与命令的键（en 与 zh-CN）；`PK/ui/pages/context-based/work-item/root.tsx` 的局部变量 `projectMemberIds` 改名为 `memberIds`，文件加入 `project-address-route`（re-76 F7）。
- [ ] 变异：`t6-mut-completed-option`（选项含已结束的）、`t6-mut-held-sends`（选所持的也发）、`t6-mut-no-guard`（轮到时不再看 `canEdit`）各让测试失败。
- [ ] 提交 `feat(M6/P7c)`。

## Task 7（T7）：工作项的“改模块”（R6，spec §2.5）

- [ ] RED：表格加“改模块”（选 Alpha → `{module_ids: [所持的…, Alpha]}`，再选 → 移出），请求体在轮到时以所持的算出（所持的在队列里先变了，结果照变了的）；所持的已归档模块列出、勾着、只可移出；列表不关；类型测试：给整集合编译不过；`modifierShortcut: "shift+m"`。
- [ ] GREEN：页类型、页、菜单（给当前值的勾选）、命令、键。
- [ ] 变异：`t7-mut-whole-set`（以打开时的集合算）、`t7-mut-archived-hidden`（不列所持的已归档的）、`t7-mut-closes`（选后关上）各让测试失败。
- [ ] 提交 `feat(M6/P7c)`。

## Task 8（T8）：A17、A18（R7，spec §2.6）

- [ ] RED：两个快捷操作的“恢复”在途时 `disabled`（按钮菜单与右键菜单各一例），回答（成功、被拒）之后可用；同一个 `act` 里调两次恢复的处理（或在重绘之前触发两个菜单）只发一个请求（re-76 F4：既有的“只发一次”的第二次 `choose` 点的是已禁用的按钮，碰不到处理里的守卫）；归档页行菜单的类型测试（给设置函数编译不过）。
- [ ] GREEN：同步的 ref 照旧守卫发送，另加状态只为禁用的显示；工厂的恢复项收 `disabled`；归档页的输入类型收窄，两个空的设置函数删去。
- [ ] 变异：`t8-mut-restore-enabled`（不传 `disabled`）、`t8-mut-no-sync-guard`（去掉 ref 的守卫、只看状态）各让测试失败；类型的收据 `t8-type-setters`（放回设置函数即 `tsc` 报错）。
- [ ] 提交 `fix(M6/P7c)`。

## Task 9（T9）：共用的移动端页头与项目页（R8，spec §2.7，X2）

- [ ] 基线：`preferences-controls.test.tsx` 的四个移动端的例（`:228` 的三行与 `:251`，`t9-baseline-mobile`）。
- [ ] RED：共用的件的测试（布局显示当前的、选项是列表、看板、日历；“显示”面板按页的常量；写经 `updateFilters`）；项目页经 layout 挂载、偏好在挂载之后才答回：答回之前不给移动端的控件、答回之后给（门在 `observer` 包装件里算，re-76 F1）。
- [ ] GREEN：`core/components/issues/` 下的共用件（只管布局与显示）；`R/issues/(list)/mobile-header.tsx` 改写为薄的 `observer` 包装件（门同桌面页头 `header.tsx:52`），手写的读写（`:32-66`）删去；四例断言不改，同数（`t9-after-mobile`）。
- [ ] 变异：`t9-mut-no-gate`（不看 `preferencesShown`）让测试失败。
- [ ] 提交 `feat(M6/P7c)`。

## Task 10（T10）：迭代页、模块页的移动端页头与顶行（检查点，R8、R9，spec §2.7）

- [ ] RED：两页经 layout 挂载、偏好在挂载之后才答回：窄屏一行答回之后才出现（re-76 F1）；布局的修改写本页的偏好；“显示”没有本页的分组；迭代页页头：侧栏开关与快捷操作不在 `md:flex` 的包里（各宽度都给）。
- [ ] GREEN：规划页一个按 store 类型的 `observer` 包装件（门取自 `usePlanningPageHeader(storeType, address).preferencesShown`），两页 layout 传它；迭代页页头的开关与快捷操作移出；`plane-diff.md` 的两行（`:382`、`:401`）改写、迭代页顶行的差异。
- [ ] 变异：`t10-mut-own-grouping`（共用的件用项目页的常量）让模块页或迭代页的测试失败；`t10-mut-toggle-hidden`（开关放回 `md:flex`）让页头的测试失败；`t10-mut-gate-outside-observer`（门在 layout 里算、以 prop 传入）让“答回之后才出现”的测试失败。
- [ ] 提交 `feat(M6/P7c)`。
- [ ] **检查点**（在提交之后，re-76 F13）：`sh .superpowers/sdd/P7c-palette-mobile/gates.sh t10-gate`（七项），三种引擎的 m6-pages（`t10-m6-pages`）；主控推送这个提交并核分支 CI，记 e2e 作业的时长（校准点）。

## Task 11（T11）：A19、A20、A21 与“添加已有”的焦点（R10、R11，spec §2.8）

- [ ] A19 RED：项目页 store 与归档页 store 的按页总数的单测（分组时有、别的项目的列表时无）；项目页页头的测试“按状态分组时显示 nerve 的总数”。GREEN：两个 store 的按页的读，两个页头改调。变异 `t11-mut-empty-group`（回到读 `""` 组）、`t11-mut-unscoped`（不比范围）各让测试失败。
- [ ] A20：删去键（en、zh-CN）与 `CYCLE_STATUS` 的 `i18n_label`（类型与五项），`@nerve/constants` 构建后类型检查与 i18n 守卫过。
- [ ] A21 RED：替身一级的测试（关上之后取当前的页报错；先开 A 再开 B、关 A 不清 B）。GREEN：替身的 `close` 只清自己。变异 `t11-mut-opened-kept` 让测试失败。
- [ ] 焦点 RED：`existing-issues-list-modal.test.tsx` 加“打开时焦点在搜索框”“移动端不在”。GREEN：搜索框在非移动端带 `data-autofocus`（或同效的写法，台账写明）。变异 `t11-mut-no-focus` 让测试失败。
- [ ] 提交（可分两次）`fix(M6/P7c)`。

## Task 12（T12）：设计 §8 的四个视图文件（R12，spec §2.9）

- [ ] RED：视图列表页、两个组件以 slug 与项目判断权限的单测（路由 store 里是别的项目时仍按地址的项目答）；视图的新建、修改、删除之后会话换了不跟进（`followInSession`）。
- [ ] GREEN：四个文件的地址读；加入 `project-address-route`（样例）；视图弹窗与删除弹窗的跟进经 `followInSession`。
- [ ] 变异：`t12-mut-router-fallback`（去掉 slug 与项目）让测试或守卫失败（守卫失败以 `keywords.mjs` 的收据记）；`t12-mut-follow-raw`（跟进不经 `followInSession`）让测试失败。
- [ ] 提交 `fix(M6/P7c)`。

## Task 13（T13）：页面故事 G19 (page) 的工作项一侧（R13，spec §2.12）

- [ ] 写 `e2e/stories/planning/g19-work-item-side-page.spec.ts` 的三条故事（spec §2.12 的表）；构建 `make build build-s3-fixture`，chromium 先跑（`t13-chromium`），再三种引擎（`t13-m6-pages-file`），`--repeat-each=3`（`t13-repeat3`），以 `durations.py` 记各引擎各故事的时长；合计 chromium ≤ 8 秒，超过先压缩，仍超停下报告主控。
- [ ] 变异（产品，落在被证的一步）：`t13-mut-search-group`（搜索不给迭代组）、`t13-mut-cycle-body`（改迭代发整个工作项的值）、`t13-mut-own-grouping`（移动端的“显示”用项目页的常量）或 `t13-mut-toggle-hidden`（迭代页的开关放回 `md:flex`）、`t13-mut-focus`（不聚焦）各让对应的故事失败；“偏好答回之前不给”只由 T9 的单测与 `t9-mut-no-gate` 证（故事里偏好在缩窄之前已答回，re-76 F2）；变异之后以干净的源重建。
- [ ] 提交 `test(M6/P7c)`。

## Task 14（T14）：文档与交接回执

- [ ] `docs/v0/plane-diff.md`（搜索、两个命令、迭代页的顶行、移动端页头）与前端改动清单（本段改动的文件与理由）；设计 §8 的“工作项一侧”“命令面板”“交给 M6 的视图文件”收尾的说法、§12 的 P7c 一行、§14 的 P7c 一行（进行中的已提交列表）。
- [ ] 交接回执“处理结果（M6/P7c）”：M3 §1（视图的跟进）、§3（“添加已有”打开即聚焦搜索框）、§4（四个视图文件的地址）；M4 M6-02（分组时页头的数目）；M1（本段改过的文件 oxlint 0 告警，收据 `t14-oxlint-touched`，对照 T2 的基线；死成员的收敛在 P8）；M2 §3（本段新的 SWR 读都经 `useSessionSWR`、键带 `loginId`，re-76 F12）；M8 的 M6-M8-01（T10 检查点的数据点）、M6-M8-08（根与写入单元的行数，`t14-line-counts`）。
- [ ] 文档链接、关键词、`make lint-web`；提交 `docs(M6/P7c)`。

## Task 15（T15）：门禁、整审与合入（主控）

- [ ] 最终候选：`sh .superpowers/sdd/P7c-palette-mobile/final.sh t15-gate`（七项与三种引擎的 m6-pages）；与之并行的整分支审查（fable，只读）。
- [ ] 修复轮（若有）、限定复审；阶段记录 `docs/v0/M6-cycles-modules/reviews/P7c-palette-mobile-review.md`（§7 留给 P8 的 T1）。
- [ ] 推送前照 spec §2.12 算估算、上界与校准点；都在 30 分钟以内才推送；分支 CI 全绿后 noff 合入 main、推送、main CI；归档、清理、记忆。
