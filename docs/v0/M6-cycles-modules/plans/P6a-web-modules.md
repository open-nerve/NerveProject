# M6 / P6a 网页的模块实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（修正后，[re-68](../../../reviews/re-68-M6P6a预检.md)）：I1 负责人与成员的筛选读按 slug 的 `getWorkspaceMemberIds`、`project-address-route` 加嵌套解构的一支（T8），I2 版式与排序经 `useLocalStorage` 按项目存在浏览器、筛选与搜索存在会话，M1 取数门的抽取移到 T5 并逐处定用哪一项，M2 宿主在 T6 只接修改、T7 接归档与删除，M3 版式与排序的类型放 `@nerve/constants`，M4 knip 把测试当入口、“测试不算读者”是本段的规则由审查执行，M5 每个新故事的时长与 T16 推送前的估算，M6 文件数不写死，已改入。整分支审查 Ready after fixes（[re-69](../../../reviews/re-69-M6P6a整体审核.md)），修复轮 F 已提交，待限定复审。基线 main `aadaf61a`（P5b 的 noff `9675bb31` 与其 main CI 失败的根因修正 noff，main CI 43373039561 首次通过）。

**Goal：**
- 模块的 service 与 store 按会话分代、只调 `/api/v0`、类型来自 OpenAPI；包装层按权限与开关取模块；迭代与模块共用一个取数门，模块的写与删照 nerve 的规则；
- 模块列表（列表与卡片、排序、筛选、搜索）、预览、归档目录、详情页的页头与侧栏（状态、负责人、成员、日期、链接、进度）、新建/修改弹窗、快捷操作、删除与归档确认框都接上新接口，被拒时留着并说明；
- 模块的进度只经 `progressPercent`，燃尽图与分布用 P5a 的件；
- 旧的 service、store、手写类型、Plane 的地址与只剩模块读者的旧进度件删除，守卫与告警收紧。

**Architecture：**
- `PlanningStore` 加 `modules`、`moduleFilters`，照 v0 §7.7 的规则写：`ReconciledByKey` 存值、`oneAtATime` 发修改、`useSessionSWR` 取数；与旧的 `module`/`moduleFilter` 并存到 T12，逐个页面改用新 store（spec R2）。
- 页面读路由参数，写入口只经 `canWriteModules`/`canDeleteModule`，跟进经 `followInSession`，拒绝经 `useFormRefusal`/`useRefusalToast`；弹窗归页面级的 `ModuleActionsHost`。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/vitest/Playwright，不加依赖。

**Spec：** [P6a-web-modules](../specs/P6a-web-modules.md)（§2 的合同与 §3 的 R1–R16 是本计划的约束）；设计 [M6](../M6-design.md) §2、§3、§8；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：P5a 的迭代（`core/services/planning/cycles.service.ts`、`core/store/planning/{cycle.store.ts,cycle-changes.ts,cycle_filter.store.ts}` 及其测试与 `fake-cycles.ts`、`core/components/cycles/{actions-host,quick-actions,modal,form,delete-modal,cycle-peek-overview}.tsx`、`archived-cycles/*`、`progress-sidebar/*`、`R/cycles/**`）、M4 的链接元数据轮询（`core/components/issues/issue-detail-widgets/links/use-link-metadata-refresh.ts`）、`e2e/stories/planning/{g1-create-cycle-page,g3-change-cycle-page,g8-archive-cycle-page,g5-cycle-progress-page,g19-cycle-pages}.spec.ts` 与 `e2e/fixtures/cycle-pages.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p6`，分支 `codex/m6-p6a`，基线 main `aadaf61a`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/docs/test/refactor(M6/P6a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 由 T1 改为 `M6/P6`（P6b 沿用）；不延长任何 `until`。本段不改 `server/`、`api/`。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内；新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（只给界面用的形状除外：版式与排序的 `ModuleLayout`、`ModuleOrderBy` 在 `@nerve/constants` 的 `src/module.ts`、由选项数组导出，先例 `constants/src/cycle.ts`；筛选的 `ModuleFilters` 在 utils 的读者旁）；service 不转换数据；store 只经自己的 `RootStore` 找兄弟 store；没有模块级的带令牌 service 实例；SWR 只经 `useSessionSWR`；新代码不读 router store。
- 删掉不用的代码；不加开关、桩或兼容别名（R2 的并行在 T12 结束）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 53，其中 9 条在模块文件）；清掉告警的提交同时把上限下调到实际值（T7 53 → 52、T10 52 → 47、T11 47 → 44），新文件不得新增告警，T12 只核对。
- **knip**：新文件在第一个读者出现的 Task 里建立（`use-modules.ts` 的第一个读者是 T2 的包装层，`use-module-filters.ts` 与版式/排序的显示设置 hook 是 T10 的列表页，`actions-host.tsx` 是 T6 的三个布局之一）。knip 6.37 把 vitest 的测试文件当入口，只被测试读的导出在 knip 看来仍有读者；“只给测试读的导出不算读者”是本段更严的规则，由逐 Task 的审查执行，不是 knip 能查出的。knip 也报只在本文件内用到的导出：删掉某个文件、组件或类型的最后一个外部读者的 Task，同时删去它或收回它的导出（例如 T8 换掉旧侧栏后的 `ProgressChart`、`progress-stats/{assignee,label,state_group}.tsx`、`TModuleDistribution`，T10/T11 换掉的旧列表组件）。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀），删除不再有读者的键。
- **`project-address-route`**：改写的模块路由页面、组件与 hook 随各自的 Task 加入规则的文件表并带样例，跑 `node tools/keywords.mjs`。
- **共用的 hook 假件**：组件测试要模块的 hook 时扩展 `@/hooks/store/fake-store-hooks`（现只有 `useCycles` 的一个）或直接 mock `@/hooks/store/use-modules`；模块的测试数据用 `fake-projects.ts` 的 `moduleOf` 与新的 `core/store/planning/fake-modules.ts`（先例 `fake-cycles.ts`）。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败，再实现到通过；纯搬移与抽取的 Task 先以既有测试的收据为基线（先例 P5b T3）。
- 取数门、写与删的门、迟到的应答（换会话、换页面、换模块）、拒绝时留着、成员的增减在轮到时算、元数据轮询的上限，每个性质至少一个能让测试失败的变异，只计能编译的变异。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出或删了文件时跑 `make knip`。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P5b 的网页单测与 e2e。迭代的单测是 T5（取数门与共用件）的回归保护，包装层的迭代用例是 T2 的，断言不改；随文件移动改导入路径不算改断言。
- 只在 spec 说契约变化的地方改：
  - `core/store/root.store.test.ts` 的 store 数（T12，23 → 21）、`core/lib/store-context.test.ts` 的注释与模块筛选的新断言（T4、T12）；
  - `R/domain-pages.test.tsx` 的 “module details and its own loader”（T8：钉的是旧 `ModulesStore.moduleService` 与英文的 “Module does not exist”）；
  - `core/components/issues/issue-layouts/removal.test.tsx` 的 “archived module retains real directory items and search”（T11：读旧 `moduleFilter`）；
  - `core/components/cycles/form-focus.test.tsx` 的模块分支（T6：旧的模块弹窗与 `PROJECT_MODULE`）；
  - `core/components/power-k/hooks/use-context-indicator.test.tsx` 对 `use-module` 的 mock（T12）；
  - M4 冒烟故事 `e2e/stories/smoke/s2-issue-list-removal.spec.ts` 的 `module`、`archived modules` 两条：它们钉的是“模块页面请求旧地址得 404、显示英文的 ‘Module does not exist’、归档目录不取数”的状态，正是本段改掉的（P5a 在 T9 的检查点才发现迭代的两条，本段预先列出）；T8 移出 `module`，T11 移出 `archived modules`，覆盖由 T13 的 G13 页面故事接上；
  - `e2e/fixtures/mounts.ts` 加 `PROJECT_MODULES`（T2）；读精确请求表的故事里有模块打开的项目时随之加上（T2 核对，写 plan 时没有）。
- 其他断言不改。

**检查点与证据**
- 检查点：T10 结束时（数据层、共用件、弹窗、快捷操作、详情与侧栏、链接、列表页完成）串行完整七项，推送并核分支 CI，同时记录 e2e 作业的实际时长（期限 35 分钟；超过 30 分钟先报告负责人，不改期限或作业结构）；这时还没有新的页面故事，量不到它们。T13、T14 记下每个新故事文件在本机三个引擎的时长，T16 推送之前照 spec §2.11 估出 CI 的 e2e 时长（基线 main CI 43373039561 的 19 分 14 秒）。T16 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准）。
- 证据放在 worktree 的 `.superpowers/sdd/P6a-web-modules/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，调研笔记在 `research/`。实现代理先读同目录的 `implementer-brief.md`。
- e2e 的写法照 P5b 阶段记录 §5 的教训：应用内跳转之后以目标页独有的内容为锚；不在重新载入之后核对控制台（M6-M8-09），“载入后仍在”由同一人另开的浏览器核对；`dismissToasts` 已容忍通知的自动关闭。

## Review Focus

1. **会话分代（v0 §7.7）**：客户端来自这一代；SWR 键带 `loginId`；换代后旧一代的回答不写进新一代；修改排队、请求体轮到时算（成员的增减）；副本取最后回答的一份；写入的回答不把行的计数写坏（R9）。
2. **取数门与写的门（R3、R5）**：一个 hook（T5 抽取）、spec §2.3 表中的各处都用它、迭代的行为不变；列表看开关、详情与归档不看；所有写入口只经 `canWriteModules`/`canDeleteModule`；已归档的菜单里没有删除。
3. **拒绝（R11）**：每个写入口（弹窗、确认框、侧栏、行内、链接）被拒时留着并说明，不提示成功；只在 nerve 确认后关闭、跳转、提示。
4. **口径与共用件（R8）**：`progressPercent` 是唯一口径；分布与页签与迭代共用、迭代行为不变；旧进度件删干净。
5. **页面的范围（G19）**：路由参数而非 router store（含负责人与成员的筛选读 `getWorkspaceMemberIds(workspaceSlug)`）；版式与排序按项目存在浏览器，筛选与搜索只在会话里、不写 `localStorage`（R4）；详情以 `moduleId` 为 key、预览侧栏以所预览的模块为 key；宿主在 key 之内；迟到的跟进不进新页面；元数据轮询有界且随卸载停止。
6. **删除干净**：旧 service、store、类型、地址、收藏入口、通知页的工作区级取数全部删除；守卫与告警的上限。

## Task 1（T1）：P5b 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P5b-cycle-list-review.md`：抬头与 §7。
- `specs/P5b-cycle-list.md`：抬头与 A8；`plans/P5b-cycle-list.md`：抬头。
- `M6-design.md`：抬头、§14 的 P5b 行“已完成”、P6 行“进行中（拆为 P6a/P6b）”。
- `tools/keywords.json`：顶层 `phase` 改为 `M6/P6`。

- [ ] Step 1：按 P5b 的归档（主目录 `.superpowers/sdd/M6-P5b-integration-20261113`：`payload/worktree/.superpowers/sdd/P5b-cycle-list/progress.md` 的 Task 16 末尾几行、`scratchpad/ci-*`、`cleanup.log`）回填 §7：
  - **最终候选的分支 CI**：`ded24135d4dd74aaec9ab73eb243f8efd9db47cb` 的 43358881107 attempt 1 成功（web 131667377966、server (1) 131667377944、server (0) 131667377578、e2e 131672956203，03:41:19Z–04:02:16Z，约 21 分钟）；之前的两次（`496d905d` 的 43098103752 一直排队、`190a2bb6` 的 43354077443 的 I21）已在 §5。
  - **合入与第一次 main CI**：noff `9675bb313564722afd48b088624812074e6fcfb3`，双父 `0b8efcd7`、`ded24135`，树与候选相同（`eae78883`）；main CI 43362963122：web、server (0)、server (1) 成功（作业号以公开的 jobs 接口 `https://api.github.com/repos/open-nerve/NerveProject/actions/runs/43362963122/jobs` 补全），e2e 131686841602 失败 452/1（G15 故事三，根因与修正已在 §5），按规则不重跑。
  - **修复分支与第二次 main CI**：`codex/m6-p5b-ci`（自 origin/main）的 `4a11baebd19772038339b5cd9f5074ea0da2afca` 的分支 CI 43368561953 成功（web 131698682381、server (1) 131698682592、server (0) 131698682764、e2e 131704307721，05:11:03Z–05:34:11Z）；noff `aadaf61a5929b57d8aa53b93e5ae04b03cc046c6`，双父 `9675bb31`、`4a11baeb`，树与修复分支相同（`b8fa511c`）；main CI 43373039561 attempt 1 成功（server (1) 131713034462、web 131713034518、server (0) 131713034573、e2e 131717802511，05:49:34Z–06:08:48Z）。
  - **归档**：`M6-P5b-integration-20261113` 4761 个文件、857,799,986 字节、未知 0、清单 sha256 `5a1452e603b1a60f084dc8b7c01eab5b8a09c0c56764121957678b50c4ed445c`（清单的 `head` 为 `4a11baeb`）；本段开工时复核 `archive.py verify`（收据 `t1-p5b-archive-verify`），数字以收据为准。
  - **清理**：`cleanup.log`（2026-11-13T06:09:58Z）记录 `codex/m6-p5b`（`ded24135`）与 `codex/m6-p5b-ci`（`4a11baeb`）本地与远端一致、都已含于 origin/main，main 与 origin/main 都在 `aadaf61a`；worktree `m6-p5b`、两个本地分支与两个远端分支已删除。
  - 勾选 P5b spec 的 A8（注明 `make test` 的规则与上面的证据）；阶段记录、spec、plan 的抬头改为“已完成（2026-11-13，noff `9675bb31`，修复 noff `aadaf61a`，main CI 43373039561 首次通过）”。
- [ ] Step 2：设计 §14 的 P5b 行改为“已完成”并附证据（noff `9675bb31`、修复 noff `aadaf61a`、main CI 43373039561、归档名），P6 行改为“进行中（拆为 P6a/P6b）”，链接 P6a 的 spec 与 plan；抬头的状态句随之改（P1–P5 已完成，P6 实施中）。`keywords.json` 的 phase 改为 `M6/P6`，跑 `node tools/keywords.mjs`（没有到期的例外：M6 的例外都是 `until: "M6/closeout"`）。
- [ ] Step 3：检查文档链接（`doc-links.py`）与关键词，然后提交 `docs(M6/P6a): 回填P5b的集成结果，进入P6阶段`。

## Task 2（T2）：service、store 的读与包装层的取数

**Files：**
- 新：`core/services/planning/modules.service.ts`；`core/store/planning/module.store.ts` 与测试（值、读与选择器，写在 T3 另一个文件）；`core/store/planning/fake-modules.ts`；`core/hooks/store/use-modules.ts`（读者：包装层）。
- 改：`core/store/planning/index.ts`（`modules`）；`core/layouts/auth-layout/use-project-fetch.ts` 与测试（模块的条件写在迭代的 `:65` 旁，取数门与开关的谓词在 T5 抽出）；`core/hooks/store/fake-store-hooks.ts`（`useModules`）；`.oxlintrc.json`（`no-restricted-imports` 列入新文件）；`e2e/fixtures/mounts.ts`（`PROJECT_MODULES`）。

- [ ] Step 1：先写测试（`FakeNerve`、`fakeRoot`、`fake-projects.ts` 的 `moduleOf`）：
  - 列表、归档列表、详情、进度、链接各自经 service 取到并按项目/模块 id 存；较旧的取数答到时不写（`Reconciled` 的性质，至少一例）；
  - `getModuleById` 取最后回答的一份（行与自己的读交错答到），`getModuleRow` 的计数来自列表行、字段是最新的；项目不在调用者的项目列表里时选择器与进度、链接都回答 `undefined`；
  - 包装层：有效成员且 `module_view` 打开时才取 `PROJECT_MODULES`（键经 `sessionKey`），不是成员、`module_view` 关闭、nerve 对项目的读未到三种情形各一例不取；迭代的既有用例不改。
  看到它们因所测性质失败（不是编译失败）。包装层的既有测试先跑一次留基线收据。
- [ ] Step 2：实现 service、store 的读与包装层的取数到通过；包装层的迭代用例断言不改全过。变异：包装层不看 `module_view`（`t2-mut-wrapper-module-view`）、`getModuleById` 不看 `answeredAt`、选择器给出不在项目列表里的项目。核对读精确请求表的 e2e 故事里没有模块打开的项目（`fixtures/mounts.ts` 的说明随之改）。
- [ ] Step 3：`make lint-web`、聚焦 vitest、`make knip`；提交 `feat(M6/P6a): 模块的 service 与 store 的读，包装层取模块的列表`。

## Task 3（T3）：store 的写

**Files：** 新 `core/store/planning/module-changes.ts`（由 `module.store.ts` 经委托使用，先例 `cycle-changes.ts`）、新 `module.store.changes.test.ts`。

- [ ] Step 1：先写测试（`fake-queue.ts` 的 `inTurn`、`sent`、`fetchedWhileChangeIsOut`）：
  - 建：回答补成计数为 0 的行，放在未归档列表最前；
  - 改：字段写进每一份（两个列表的行、自己的读），行的计数不变（spec R9）；
  - 成员的增减：两次在途的增减（加 A、再去 B）都在轮到时以 nerve 最近一次回答的集合算出整集合，第二次的请求体含第一次的结果；已离开项目的成员留在集合里；
  - 归档/恢复：行在两个列表之间移动、计数保留、放在最前；重复归档照常；没有行时不造行；
  - 删：两个列表删去，读到的写成 `null`，进度与链接随之读不到；
  - 链接：加的回答放在最前；改与删在轮到时按 `module_id` 找列表；改了地址的回答（元数据为空）照写；
  - 修改一个接一个发出；取数在外时确认的修改重放到它的回答上；
  - 每个写的拒绝交给调用者，store 的值不变；`SessionChangedError` 不改这一代。
- [ ] Step 2：实现到通过。变异：改把计数写成回答里没有的值（`t3-mut-update-drops-counts`）、成员的整集合在调用时算（`t3-mut-members-at-call`）、归档不移动、拒绝时仍写入、链接的改按调用时的列表找。
- [ ] Step 3：门禁同 T2；提交 `feat(M6/P6a): 模块 store 的写——一个接一个、写进每一份、成员的增减在轮到时算`。

## Task 4（T4）：筛选 store、界面类型与工具函数

**Files：**
- 新：`core/store/planning/module_filter.store.ts` 与测试，接进 `PlanningStore`（hook `use-module-filters.ts` 在 T10 随列表页建立）。
- 改：`web/packages/constants/src/module.ts`（`ModuleLayout`、`ModuleOrderBy` 由 `MODULE_VIEW_LAYOUTS`、`MODULE_ORDER_BY_OPTIONS` 推出，选项的类型照 `constants/src/cycle.ts` 写成内联联合；`MODULE_STATUS` 的值改用 `@nerve/api-client` 的 `ModuleStatus`；不再从 `@nerve/types` 导入模块的类型）；`web/packages/utils/src/planning.ts` 与测试（`ModuleFilters`、`orderModuleRows`、`matchesModuleFilters`，旧的 `module.ts` 由旧 store 读到 T12）；`core/lib/store-context.test.ts`（退役的一代的模块筛选不随 router 变，spec R4；`:469-472` 的注释随之改）；`core/store/root.store.ts` 的 `dispose()` 说明与 `core/lib/store-context.tsx` 的说明（`planning` 的模块筛选没有反应；旧 `moduleFilter` 的反应到 T12 删除）。
- 版式与排序不进这个 store：它们由 T10 的显示设置 hook 经 `useLocalStorage` 存（spec §2.4、R4）。

- [ ] Step 1：先写测试：
  - 筛选（`{default, archived}` 两组）与两个搜索词按项目 id 存取，两个项目互不影响；store 上没有版式与排序；没有 `reaction`、不写 `localStorage`（测试核对 `localStorage` 未被写）；`store-context.test.ts` 断言换代后旧一代 `planning.moduleFilters` 不随 router 变；
  - `orderModuleRows`：名字的自然排序；进度经 `progressPercent`（已取消不在分母里，与 Plane 的（完成 + 取消）÷ 总数排出不同次序的数据）；工作项数、目标日、建立时刻的降序按值；没有目标日的两向都在最后；“手动”按 `sort_order`；相等的保持传入次序；遍历 `MODULE_ORDER_BY_OPTIONS` 的每个键都有排法；
  - `matchesModuleFilters`：状态、负责人、成员（相交）、开始日与目标日按日历日（在负偏移时区 `TZ=America/Los_Angeles` 下另跑一次，日子不偏）。
- [ ] Step 2：实现到通过（`TModuleLayoutOptions`、`TModuleOrderByOptions` 的两个旧读者 `module-layout-icon.tsx`、`dropdowns/order-by.tsx` 随 T10 改用新类型，`@nerve/types` 的 `module/` 在 T12 删除）。变异：降序按布尔值排（Plane 的写法，`t4-mut-desc-by-boolean`）、进度按（完成 + 取消）÷ 总数、日期经 `new Date()` 解析、筛选 store 加跟随 router 的反应、筛选 store 写进 `localStorage`（`t4-mut-filters-persist`）。
- [ ] Step 3：门禁同 T2（含 `pnpm --filter @nerve/utils test`、`pnpm --filter @nerve/constants test`）；提交 `feat(M6/P6a): 模块的筛选 store（只在会话里、不跟随 router）、版式与排序的类型、排序与筛选函数`。

## Task 5（T5）：取数门与迭代、模块共用的件

**Files：**
- 取数门（spec §2.3、R3）：新 `core/hooks/use-planning-reads.ts`（名字由实现定，含开关的谓词，如 `planningOn`）与测试；`core/layouts/auth-layout/use-project-fetch.ts`（迭代与模块的开关改用谓词，成员判断照旧）；§2.3 表中迭代的各处：`R/cycles/(list)/page.tsx`、`power-k/ui/pages/open-entity/project-cycles-menu.tsx`、`R/cycles/(detail)/[cycleId]/page.tsx`、`cycles/archived-cycles/root.tsx`、`cycles/active-cycle/priority-issues.tsx`、`issues/issue-layouts/roots/cycle-layout-root.tsx`、`R/cycles/(detail)/header.tsx`，各用表中写定的一项。
- 共用件：`core/components/cycles/{read-failure,disabled}.tsx` → `core/components/planning/`（停用空态按域取文案、图与设置页地址）；`cycles/{distribution-rows.ts,distribution-rows.test.ts,distribution-stats.tsx}` → `core/components/core/sidebar/`；从 `cycles/progress-sidebar/progress-stats.tsx` 抽出共用的进度页签（同处，名字由实现定）；迭代的读者改导入（`R/cycles/**`、`cycles/{cycle-peek-overview,archived-cycles/root,active-cycle/*,progress-sidebar/*}`、`issues/issue-layouts/roots/cycle-layout-root.tsx`）；`@nerve/propel` 的 `charts/area-chart/root.tsx` 与 `@nerve/types` 的 `charts` 删去 `comparisonLine`；`tools/keywords.json`（`project-address-route` 的文件表已列 `cycles/{read-failure,disabled,distribution-stats}` 等旧路径：加入新路径与样例，旧路径照 M4/P12 的做法留着、阻止复活）。

- [ ] Step 1：先跑迭代的既有测试留基线收据（`cycles/**/*.test.tsx`、`distribution-rows.test.ts`、`burndown-chart.test.tsx`、迭代的页面测试，以及取数门各处的测试：`R/cycles/(list)/{page,priority-issues,root}.test.tsx`、`R/cycles/(detail)/header.test.tsx`、`project-cycles-menu.test.tsx`、`issues/issue-layouts/roots/{cycle-layout-root,cycle-layout-access}.test.tsx`、`archived-cycles` 与迭代详情页的测试、`use-project-fetch.test.ts`）。再写新测试：
  - 取数门：成员与访客“可读”；非成员、项目已归档、别的工作区的项目都不可读；“列表可取”在可读之上看该域的开关，迭代与模块各一例（模块看 `module_view`、不看 `cycle_view`）；谓词与包装层用的是同一个；
  - 共用的页签：不带行切换时行不可点（没有按钮）；页签与面板一致（选“状态”看到状态组的行，选“负责人”看到负责人的行）；页签记在调用者给的键下；带行切换时同迭代现在的行为；
  - 停用空态：模块的文案、图与设置页地址；只给项目管理员按钮。
- [ ] Step 2：抽取取数门、各处改用它，搬移与抽取共用件，到通过；迭代的既有测试断言不改全过（与基线同数）。变异：取数门不看项目已归档（`t5-mut-gate-archived`）、模块的“列表可取”读 `cycle_view`、面板次序与页签不一致（Plane 的次序，`t5-mut-panels-plane-order`）、不带行切换时仍渲染按钮、停用空态对成员给按钮。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`pnpm --filter @nerve/propel test`（如有）、`node tools/keywords.mjs`；提交 `refactor(M6/P6a): 迭代与模块共用的取数门、读失败、停用空态、分布与进度页签，删去 AreaChart 无人用的对比线`。

## Task 6（T6）：写的门、宿主与新建/修改弹窗

**Files：** `core/store/user/permissions.store.ts`（`canWriteModules`、`canDeleteModule`，与迭代共用“一律拒绝”的条件）与测试；新 `core/components/modules/actions-host.tsx`（`ModuleActionsHost`、`useModulePage`，挂在 `R/modules/(list)/layout.tsx`、`R/archives/modules/layout.tsx`、`R/modules/(detail)/layout.tsx`；本 Task 只接修改弹窗与 `pageMounted`，归档与删除确认框在 T7 改写时接入，spec §2.7）；`core/components/modules/{modal.tsx,form.tsx,select/status.tsx}`；`core/components/modals/project-level.tsx`；`cycles/form-focus.test.tsx` 的模块分支；`web/packages/constants/src/tab-indices.ts`（删除 `PROJECT_MODULE`）；新文案；`tools/keywords.json`（`project-address-route`）。

- [ ] Step 1：先写测试（jsdom，`fake-session-swr`、`fake-store-hooks`、`fake-refusal`）：
  - 写的门：管理员、成员能写；访客、只看自己的访客（含工作区管理员兼访客、项目不开“访客看全部”）、`module_view` 关闭、项目不在都不能；能删：管理员与工作区管理员、建立者且按项目角色是管理员或成员，其他成员、访客建立者不能；迭代的两个门的既有测试不改；
  - 新建：只给名字时请求体为 `{name, status: "backlog"}`（没有日期键、没有 `project_id`）；带日期、负责人、成员时各有；只选一天时只有那一天；修改只发改了的字段（日期一对、清空为 `null`、`member_ids` 整集合）；
  - 项目下拉只列 `canWriteModules` 的项目；换项目时负责人与成员清空；负责人与成员的下拉没有访客（以项目成员里的一个访客核对）；
  - 被拒（409 `planning.module_name_taken`、422 `name`、422 `lead_id`）时弹窗留着、输入保留，字段错误在字段下并聚焦，其余拒绝经提示；只在确认后关闭；失败后表单不重置；
  - 标题经 `ModalTitle`，没有正 `tabIndex`，Escape 经 `handleClose`、提交中不能关，焦点回到打开的控件；状态选择没过规则时有提示、`aria-invalid`；
  - 宿主：经 `useModulePage()` 打开修改弹窗；行卸载后确认照常提示，宿主卸载后迟到的回答只进 store；宿主此时没有归档与删除的入口（旧的两个确认框收 `IModule`，T7 接入）。
- [ ] Step 2：实现到通过（删除 `getTabIndex(PROJECT_MODULE)` 与 `PROJECT_MODULE` 的表、`error?.detail ?? error?.error`、表单的 `reset` 失败分支）。变异：失败也关闭（`t6-mut-close-on-refusal`）、项目下拉用 `projectsWithCreatePermissions`、换项目不清空成员、只看自己的访客能写、成员非建立者能删。
- [ ] Step 3：门禁同 T2（`make knip`）；提交 `feat(M6/P6a): 模块的写与删的门、页面级宿主与新建/修改弹窗`。

## Task 7（T7）：快捷操作、删除、归档与恢复

**Files：** `core/components/modules/{quick-actions.tsx,delete-module-modal.tsx}`、`archived-modules/modal.tsx`（改写为收 `Module`，`:20-24`、`:18-30` 的旧参数删去）、`core/components/modules/actions-host.tsx`（接入归档与删除两个确认框）、`common/quick-actions-helper.tsx` 的模块菜单；`tools/keywords.json`（`copy-hook` 中 `modules/quick-actions.tsx` 的例外、`project-address-route`）；`web/apps/web/package.json` 的 `lint-cap`。

- [ ] Step 1：先写组件测试：
  - 菜单各项的显示与停用条件（spec §2.6 快捷操作一行、R5）：能写、各状态、已归档、管理员、建立者、其他成员、访客；归档对未完成、未取消的停用并说明（经 `t()`）；已归档的菜单里没有删除；
  - 复制链接经 `useCopyLink`，失败有提示；新标签页带 `noopener,noreferrer`；
  - 删除、归档经宿主的确认框（`useModulePage()` 打开），只在确认后关闭/提示/跳转，被拒时留着并说明，不提示成功；行卸载后确认照常提示，宿主卸载后迟到的回答只进 store；恢复同样只在确认后提示；在详情页或预览里删除所显示的模块后回到列表；恢复在途时再触发不发；
  - 菜单按钮以模块命名、不嵌套按钮；右键菜单关着时不可交互；归档确认框的标题经 `ModalTitle`、没有 `tabIndex={1}`。
- [ ] Step 2：实现到通过；同一提交删除 `copy-hook` 的这条例外，并因 `archived-modules/modal.tsx` 的正 `tabIndex` 告警消失把 `lint-cap` 的上限 53 → 52。恢复成功后进入归档目录（同 Plane）。变异：成员非建立者也能删除（`t7-mut-delete-any-member`）、恢复失败仍提示成功、已归档仍给删除、访客看到归档、恢复在途可再发。
- [ ] Step 3：门禁同 T2；提交 `feat(M6/P6a): 模块的快捷操作、删除、归档与恢复——只在 nerve 确认后跟进`。

## Task 8（T8）：详情页与侧栏

**Files：** 路由 `R/modules/(detail)/{layout.tsx,header.tsx,[moduleId]/page.tsx}` 与新 `page.test.tsx`；`R/domain-pages.test.tsx` 的模块用例；`core/components/modules/progress-sidebar/*`、`module-status-dropdown.tsx`；删除 `core/components/core/sidebar/progress-chart.tsx`、`progress-stats/{assignee,label,state_group}.tsx`；`e2e/stories/smoke/s2-issue-list-removal.spec.ts`（移出 `module` 一条）；`.oxlintrc.json`；`tools/keywords.json`（`project-address-route`：文件表加入详情页与页头，内容模式加嵌套一层解构的分支，spec §2.10）；新文案。

- [ ] Step 1：先写测试：
  - `project-address-route` 的嵌套分支：先在 `samples.hit` 加 `const { workspace: { workspaceMemberIds } } = useMember()` 与另一个嵌套的 getter 两例、`samples.miss` 加 `const { workspace: { getWorkspaceMemberIds } } = useMember()`，看到 `node tools/keywords.mjs` 因两个 hit 样例没有命中而失败（收据）；再加分支（候选写法与试跑在 spec §2.10），样例全过、规则现有的文件不变红；
  - 子树以 `moduleId` 为 key、宿主在 key 之内；`planning.module_not_found` 与其他失败的两种态（文案经 `t()`）；开关关闭时照常读、只读；已归档的模块照常显示、只读、不在切换列表里；
  - 面包屑切换列出本项目未归档的模块（服务端次序）；
  - 侧栏改状态、负责人、成员、日期：PATCH 确认前不提示成功、在途时停用、被拒时回到 nerve 的值；能写且未归档时才可改；成员多选的一次点击是一次增减；日期两天都选了或清除时才发、与现值相同不发；
  - “工作项”与状态页签的数经 `progressPercent`；有燃尽时 `BurndownChart`，没有时“暂无数据”、分布照常；页签用 T5 的件、行不可点；页签记在 `module-progress-tab-{id}`。
- [ ] Step 2：实现到通过（删除 `ProgressChart`、`progress-stats/` 的三个文件与随之没有外部读者的导出；`domain-pages.test.tsx` 的模块用例改为新详情页的读法；S2 移出 `module` 一条，说明覆盖交 G13）。变异：侧栏不等回答就提示（`t8-mut-toast-before-answer`）、已归档时仍可改、只选一天就发、子树不以 `moduleId` 为 key；守卫去掉嵌套分支后 hit 样例失败（`t8-mut-guard-flat-only`）。
- [ ] Step 3：门禁同 T2（`make knip`）；提交 `feat(M6/P6a): 模块的详情页与侧栏——以模块为 key，修改等 nerve 确认，进度用共用件`。

## Task 9（T9）：链接

**Files：** `core/components/modules/links/{list.tsx,list-item.tsx,create-update-modal.tsx}`、新的元数据重读 hook（`links/` 下，名字由实现定）与测试；侧栏的链接一节；`tools/keywords.json`（`copy-hook` 中 `modules/links/list-item.tsx` 的例外、`project-address-route`）；新文案。

- [ ] Step 1：先写测试：
  - 链接对访客也显示；加、改、删只给能写且模块未归档的人；
  - 一行：内嵌的 `data:image/` 图标显示、`favicon_url` 不用；文字依次取链接的标题、元数据的标题、地址；“由谁添加”在账户已不在时只写时刻；复制经 `useCopyText`，失败有提示；
  - 弹窗：标题经 `ModalTitle`；地址为空时显示提示并 `aria-invalid`；422 `url` 在字段下；409 `planning.module_link_exists` 经提示、弹窗留着；只在确认后关闭；删除等确认后行消失、被拒时提示且行留着；
  - 元数据的重读（假时钟）：有元数据为空的链接时每秒重读一次 `MODULE_LINKS`；都有了（含带 `error` 的）时停止；至多 30 次、30 秒；卸载、读被拒时停止；没有空的时不读。
- [ ] Step 2：实现到通过；同一提交删除 `copy-hook` 的这条例外。变异：重读不设上限（`t9-mut-refresh-unbounded`）、卸载后仍重读、用 `favicon_url`、访客看不到链接、已归档仍可加。
- [ ] Step 3：门禁同 T2；提交 `feat(M6/P6a): 模块的链接——人人可见、等 nerve 确认、元数据有界重读`。

## Task 10（T10）：列表页、两种版式、页头与预览（检查点）

**Files：** 路由 `R/modules/(list)/{page,header,mobile-header,layout}.tsx`；`core/components/modules/{modules-list-view,module-list-item,module-list-item-action,module-card-item,module-view-header,module-layout-icon,module-peek-overview}.tsx`、`dropdowns/order-by.tsx` 与测试（两者改用 `@nerve/constants` 的 `ModuleLayout`、`ModuleOrderBy`）；新 `core/hooks/store/use-module-filters.ts`；新的显示设置 hook（`core/components/modules/` 下，名字由实现定，经 `useLocalStorage`，spec §2.4、R4）与测试；`.oxlintrc.json`；`tools/keywords.json`（`project-address-route`）；`web/apps/web/package.json` 的 `lint-cap`；新文案。

- [ ] Step 1：先写测试：
  - 列表页的五种态（停用、加载、失败可重试、空、有模块）；“新建模块”只给能写的人，经项目级宿主；
  - 行：圆环的百分比经 `progressPercent`、已完成不到 100% 时的“!”、负责人或“没有负责人”、日期按日历日、没有收藏星；卡片：“已完成/（总数 − 已取消）”与进度条同一份；行内改状态与日期只给能写且未归档的、等确认、在途停用、被拒回原值；
  - 显示设置 hook：默认版式 `list`、排序 `name`；键带项目 id，两个项目互不影响；卸下再挂上（同一浏览器重新载入）时读回存过的值；存的值不在选项里时回到默认；
  - 版式切换（页头与移动端的菜单）与排序（键盘可用、方向切换、“手动”没有方向）经这个 hook 写，列表随之；筛选 store 上没有版式与排序（T4）；
  - 页头的成员读按 slug：`module-view-header.tsx` 加入 `project-address-route` 的文件表，同一提交把 `workspace: { workspaceMemberIds }` 改为 `getWorkspaceMemberIds(workspaceSlug)`（T8 的嵌套分支认得旧写法；筛选的行为测试与变异在 T11）；
  - 预览：`?peekModule=` 打开，侧栏以所预览的模块为 key（从 A 换到 B 时 A 的选到一半的日期、在途标记、页签不带到 B）；失败可重试；关闭去掉查询参数。
- [ ] Step 2：实现到通过（建立 `use-module-filters.ts`；删除列表行与卡片的收藏、换掉的旧组件；`module-card-item.tsx` 的 5 条告警清掉，上限同提交 52 → 47）。变异：圆环按（完成 + 取消）÷ 总数（`t10-mut-ring-plane-formula`）、访客可行内修改、预览侧栏不以模块为 key、取数失败时仍显示加载、显示设置的键不带项目 id（`t10-mut-display-key-no-project`）、存的值不在选项里时照用。
- [ ] Step 3：门禁同 T2；提交 `feat(M6/P6a): 模块列表页、列表与卡片两种版式、排序与预览`。
- [ ] Step 4（检查点）：`sh .superpowers/sdd/P6a-web-modules/gates.sh t10-gate` 串行七项；推送准确的 HEAD，核分支 CI，记录 e2e 作业的时长（这时还没有新的页面故事，这个数不是它们的时长，spec §2.11）。

## Task 11（T11）：搜索、筛选与归档目录

**Files：** `core/components/modules/{module-view-header.tsx 的搜索,applied-filters/*,dropdowns/filters/*}`；路由 `R/archives/modules/{layout,page}.tsx`、`core/components/modules/archived-modules/{root,header,view}.tsx`（排序读 T10 的显示设置 hook、与列表同一个；页头的成员读同 T10 改为 `getWorkspaceMemberIds(workspaceSlug)`）；`removal.test.tsx` 的已归档模块用例；`e2e/stories/smoke/s2-issue-list-removal.spec.ts`（移出 `archived modules` 一条）；`.oxlintrc.json`；`tools/keywords.json`（`project-address-route`）；`web/apps/web/package.json` 的 `lint-cap`；新文案。

- [ ] Step 1：先写测试：
  - 搜索按项目存、Escape 先清空再收起（同 Plane）；筛选的状态、负责人、成员（路由 slug 的工作区成员：router store 的 `workspaceSlug` 指向别的工作区时仍列路由那个的，列表页与归档目录各一例）、开始日与目标日（标题经 `t()`），已选筛选条可清除；空态区分清除筛选与清除搜索；筛选按钮的名字与 `aria-expanded`；没有 `autoFocus`；
  - 归档目录：取数失败可重试（不永远加载）、空态、搜索、排序（与列表同一个存值）与筛选、行打开预览、恢复经快捷操作；
  - 列表页与归档目录的筛选互不影响。
- [ ] Step 2：实现到通过（`applied-filters/root.tsx` 的 2 条 `prefer-set-has`、`dropdowns/filters/root.tsx` 的 `autoFocus` 清掉，上限同提交 47 → 44；`removal.test.tsx` 的已归档模块用例改为新 store 的读法；S2 移出 `archived modules` 一条，说明覆盖交 G13）。变异：归档目录失败时仍显示加载（`t11-mut-archive-forever-loader`）、筛选把列表与归档混用、日期筛选经 `new Date()`、筛选的成员读 `workspaceMemberIds`（`t11-mut-members-from-router`）。
- [ ] Step 3：门禁同 T2；提交 `feat(M6/P6a): 模块的搜索、筛选与归档目录`。

## Task 12（T12）：跨领域读者、旧代码删除与守卫

**Files：** spec §2.9 所列；`tools/keywords.json`（`old-module-urls`、`old-cycle-urls` 的说明一句、`project-address-route`、`copy-hook` 中命令面板模块的例外）；`.oxlintrc.json`；`core/store/root.store.ts` 与 `root.store.test.ts`、`core/lib/store-context.{tsx,test.ts}`；`core/store/router.store.ts`（`moduleId`，若仍没有读者）。

- [ ] Step 1：先写会失败的检查：`old-module-urls` 规则（spec §2.10 的模式与样例）加入后 `node tools/keywords.mjs` 因旧 service 命中而失败（收据）；命令面板的模块菜单、上下文命令与上下文指示的测试改用新 store（成员的增减经 store、复制经 hook、`create_module` 经 `canWriteModules`），先看它们失败。
- [ ] Step 2：改完 spec §2.9 的读者，删除旧的 service、store、hook、`@nerve/types` 的 `module/` 与 `ILinkDetails`、`@nerve/utils` 的 `module.ts`、`WORKSPACE_MODULES`、`use-workspace-issue-properties.ts` 与通知页对它的调用、收藏 store 与收藏项的模块分支、不再有读者的文案键；删除命令面板模块的 `copy-hook` 例外；`RootStore` 不再有 `module`、`moduleFilter`，`root.store.test.ts` 的 store 数 23 → 21，`dispose()` 与 `store-context` 的说明改为“模块的筛选也没有反应”。变异：`create_module` 用成员级的检查、命令面板的成员增减在调用时算。
- [ ] Step 3：死成员：把 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)附录 A 的 `deadsym.mjs`、`lib/program.mjs`、`domains.mjs` 原样写进台账目录 `dead-members/`，先跑 `react-router typegen` 再跑 `domains.mjs --rows M6`，列出模块的行（P5a 台账 Task 12 的 18 行与本段新出现的），删去或写下结论（台账）；oxlint 告警：核对本段改到的文件为零、`lint-cap` 等于 44。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`；提交 `refactor(M6/P6a): 命令面板、收藏与通知页改用新模块 store，删除旧模块代码，加 old-module-urls`。

## Task 13（T13）：页面故事一（G9、G10、G13）

**Files：** `e2e/stories/planning/{g9-create-module-page,g10-module-links-page,g13-archive-delete-module-page}.spec.ts`；新 `e2e/fixtures/module-pages.ts`（`modulePagesWorld`、地址、行与卡片的定位；与迭代共用的帮手从 `cycle-pages.ts` 导入）。

- [ ] Step 1：写故事（spec §2.11 的表），数据库断言与同号接口故事共用；先跑一个期望错误的版本（或以 e2e 变异），看到断言失败，再改正。接上 S2 移出的两条：G13 中打开不存在的模块地址显示“模块不存在”，接口请求只有 `/api/v0` 的读（模块 404），没有旧地址；没有已归档模块时归档目录显示空态，不是加载占位，没有旧地址。每个故事以整页安静的断言收尾（`expectQuietConsole`）。
- [ ] Step 2：`make build build-s3-fixture` 后聚焦跑（`cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test --project=chromium stories/planning/g9-create-module-page.spec.ts …`），通过后以 `--config playwright.m6-pages.config.ts` 在三种引擎上跑一次（M4 交接 M6-04(a)，收据），新故事另 `--repeat-each=3`；从三种引擎那次运行的报告记下每个新文件在各引擎的时长（台账，供 T16 估算，spec §2.11）；`make lint-web`；提交 `test(M6/P6a): 模块页面的故事（G9、G10、G13）`。

## Task 14（T14）：页面故事二（G12、G19）

**Files：** `e2e/stories/planning/{g12-module-progress-page,g19-module-pages}.spec.ts`；夹具。

- [ ] Step 1：写故事；迟到的回答用 `sentHeld`，换账号用 `anotherTabSignsIn`，换工作区与换项目用 `moveWithinApp`/后退，复制失败用拒绝剪贴板的写法（P5a G19 的先例）。只看自己的访客的数与接口故事 G12 同一份数据库断言。先看到错误的期望失败，再改正。
- [ ] Step 2：同 T13 聚焦跑、三种引擎、`--repeat-each=3`，记下每个新文件在各引擎的时长；`make lint-web`；提交 `test(M6/P6a): 模块页面的故事（G12、G19）`。

## Task 15（T15）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`：网页的模块行为差异（spec R3–R6、R8、R10–R14：开关关闭时的详情、筛选与搜索不跨重新载入（版式与排序照 Plane 保留）、写与删的门、已归档菜单里没有删除而服务端允许、换项目清空负责人与成员、一个进度口径、进度一节总显示分布、链接人人可见与元数据有界重读、不吞拒绝、已归档模块的详情可读、侧栏与行内的日期选完一段才发、降序与空目标日的次序、收藏入口删除）。
- `docs/v0/frontend-changes.md`：模块的数据层一行、筛选 store 与显示设置一行（R4）、写入口一行、收藏一行、包装层的取数（“挂载时的取数”一行补模块）、取数门与共用件一行。
- 输入交接的“处理结果（M6/P6a）”：M1-P2（模块的进度改读新接口；照设计 §11 保持 open 至收尾）；M1-closeout（死成员、oxlint 与进度口径第 2 条的模块部分）；M2-closeout §3（模块的反应）、§5(b)(c)(d)、§6（模块的两处复制）；M3-closeout §1（包装层、反应、`modal.tsx` 的两处、`moduleId` 的 key 与后退核对）、§2（8 行）、§3（模块各项的浏览器核对）、§4、§5（`PROJECT_MODULE`）；M4-closeout M6-01（模块的详情、统计、归档目录、空态、拒绝与重试）、M6-04（模块表单三种引擎、页签与面板的次序、图表 0/1/多个点）、M6-02 的模块部分写明在 P6b；M5-closeout 的 M5-M6-02 模块部分写明在 P6b。均保持 open（设计 §11）。
- 设计：§8（R4 的模块筛选、§2.5 的共用件）、§12（P6 拆为 P6a/P6b 两行，P6b 一行含 G15、G19 的模块页列表部分，R15；P7 一行去掉已在 P6a 完成的模块文件的地址、复制、收藏入口）、§14 的 P6a 行；`docs/v0/v0-design.md` §7.7“一代退役时释放它的反应”一句（模块的筛选在 M6/P6a 改为按项目 id 存取、不跟随 `router`，没有要释放的反应）；spec §5 的后续补全。

- [ ] Step 1：写差异与回执；Plane 的行号对照主目录的只读 `plane/` 逐一核过。
- [ ] Step 2：检查文档链接与关键词，然后提交 `docs(M6/P6a): 交接回执、差异清单与前端改动清单`。

## Task 16（T16）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t16-gate`，留收据；`make test` 照负责人的规则），另以三种引擎的页面配置跑一次（收据）；推送之前照 spec §2.11 估出 CI 的 e2e 作业时长（main CI 43373039561 的 19 分 14 秒，加上 T13、T14 记下的新文件 chromium 时长乘以迭代页面故事 CI 与本机之比），估出超过 30 分钟先报告负责人、不推送；推送准确的 HEAD，核分支 CI，记录 e2e 作业的实际时长（超过 30 分钟同样先报告）。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账；
  - 写阶段记录 `reviews/P6a-web-modules-review.md`；
  - 勾选 spec §4；A8 待合入后由 P6b 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
