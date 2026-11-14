# M6 / P6b 模块页的工作项列表实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（修正后，[re-70](../../../reviews/re-70-M6P6b预检.md)）：F1（Important）e2e 的时长以 25 分 00 秒为基线、T16 另算上界与 T10 的校准点，慢的 runner 上可能超过报告线（T16）；F2 T1 的路径改为 P6a 的归档（T1）；F3 写入单元对根没有值的循环导入（T2）；F4 共用的布局根不导入 `PeekOnEntry`，按域的差别经 props 或按域的小表（T7、T8）；F5 写入单元不加入 `project-address-route`（T2，spec §5 的 P7）；F6 `module_issues` 的副本由 P7 改为推出（spec §5）；F7 T3 的收据只查方法的定义（T3）。T1–T15 已实施，整分支审查 Ready after fixes（[re-71](../../../reviews/re-71-M6P6b整体审核.md)），修复轮 F 已提交，待限定复审。基线 main `06b72d4b`（P6a 的 noff；main CI 43469787621 attempt 2 通过，由 T1 回填）。

**Goal：**
- 往工作项根 store 加模块之前，按职责拆出写入单元，调用方改经它（M6-M8-08）；
- P5b 只为迭代页写的列表件中只差领域的，改为迭代与模块共用；
- 模块页的工作项列表接上 M4 的列表引擎：`EIssuesStoreType.MODULE`、本页偏好、四种布局、页头、空态、已归档只读、本页新建与“移出模块”（`module_ids` 整集合轮到时算）、写后重读进度、附件列、侧栏的行切换本页筛选；
- 页头的列表控件在本页偏好读到之后才出现、重读失败时正文沿用已读的偏好，用于模块页、迭代页与项目页；
- G15、G19 的模块页列表部分在三种引擎上有页面故事。

**Architecture：**
- 工作项根 store 只留组成、读的凭据与可见性、列表与资源的登记；写队列、确认的流水线、命令与请求体在写入单元（spec R1）。
- 列表是项目列表加 `module_id`（P5b R1）：模块页的列表 store 继承两域共用的本页列表，偏好以 `PagePreferencesFilter` 直接构造（spec R2、§2.2）。
- 页面读路由参数；写入口按权限与模块是否可加入显示；移出的请求体在写入单元的队列轮到时由所持的整集合算出；跟进经 `followInSession`，拒绝经 `useRefusalToast`/命令宿主。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/vitest/Playwright，不加依赖。

**Spec：** [P6b-module-list](../specs/P6b-module-list.md)（§2 的合同与 §3 的 R1–R16 是本计划的约束）；设计 [M6](../M6-design.md) §3.3、§7、§8、§12；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：P5b 的迭代页列表（`core/store/issue/cycle/*`、`core/hooks/use-cycle-*.ts`、`core/components/issues/issue-layouts/roots/cycle-layout-root.tsx`、`*/roots/cycle-root.tsx`、`empty-states/cycle.tsx`、`quick-action-dropdowns/cycle-issue.tsx`、`command-host.tsx`、`R/cycles/(detail)/header.tsx` 及其测试与 `fake-cycle-page.tsx`）、`core/components/cycles/progress-sidebar/progress-stats.tsx`、`e2e/fixtures/cycle-list-pages.ts`（T13 起为按域的 `e2e/fixtures/planning-list-pages.ts`）、`e2e/stories/planning/g15-cycle-list-page.spec.ts`、`g19-cycle-list-page.spec.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p6b`，分支 `codex/m6-p6b`，基线 main `06b72d4b`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/docs/test/refactor(M6/P6b)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 仍为 `M6/P6`；不延长任何 `until`。本段不改 `server/`、`api/`。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内（工作项根 store 与写入单元拆后各在其内，T3 以 `wc -l` 核对）；没有循环依赖：写入单元对根只有 `import type` 或构造时传入的窄接口（spec §2.1，re-70 F3）；新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`；service 不转换数据；没有模块级的带令牌 service 实例；SWR 只经 `useSessionSWR`；新代码不读 router store（页面读路由参数）。
- 删掉不用的代码；不加开关、桩或兼容别名。唯一的过渡：T2 为让提交能编译与通过测试，在根上暂留测试所用写入方法的转发，T3 删去（spec R1）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 44）；清掉告警的提交同时把上限下调到实际值，新文件不得新增告警。
- **knip**：新文件在第一个读者出现的 Task 里建立。knip 把 vitest 的测试文件当入口，只被测试读的导出在它看来仍有读者；“只给测试读的导出不算读者”是本阶段更严的规则，由逐 Task 的审查执行。knip 也报只在本文件内用到的导出：删掉某件最后一个外部读者的 Task，同时删去它或收回导出（例如 T4 之后的 `CycleIssuesFilter`、T7 之后被共用件取代的迭代根）。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀）；本段要加的键见 spec §2.9。
- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例），跑 `node tools/keywords.mjs`；被共用件取代的迭代路径留在表里阻止复活。
- **`.oxlintrc.json`**：用 `useSessionSWR` 的新文件（共用的布局根等）随它的 Task 加入 swr 限制的文件表；`cycles/active-cycle/priority-issues.tsx` 由 T1 加入。只收紧。
- **共用的假件**：组件测试要模块的 hook 时扩展 `@/hooks/store/fake-store-hooks` 或直接 mock；模块的测试数据用 `fake-projects.ts` 的 `moduleOf` 与 `core/store/planning/fake-modules.ts`；迭代页的替身 `fake-cycle-page.tsx` 在 T7 改为按域。

**构建与类型**
- 改了 `@nerve/types`、`@nerve/constants` 等包之后，先构建该包（`pnpm --filter <pkg> build`）再跑应用的类型检查，否则应用读到的是旧的声明；改了路由文件之后先跑 `react-router typegen`（`make knip` 已含）。
- 变异只计能编译的；类型检查不过的变异不算失败。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败（不是编译失败），再实现到通过；纯搬移与共用的重构（T2、T3、T4、T7 与 T9–T11 中共用的部分）先以既有测试的收据为基线，之后同数。
- 整集合在轮到时算、在途的写、白名单、只读（已归档）、权限门（新建、移出）、迟到的应答（换会话、换模块）、拒绝时说明、偏好读到之前没有控件、重读失败时沿用，每个性质至少一个能让测试失败的变异。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出、删了文件或改了路由时跑 `make knip`。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P6a 的网页单测与 e2e。
- **拆分（T2、T3）**：调用 `enqueueIssueChanges`、`confirmCreation`、`createIssue`、`updateIssue`、`deleteIssue`、`archiveIssue`、`unarchiveIssue`、`pendingIDs` 的测试（含 `vi.spyOn` 这些方法的与替身 `core/store/issue/fake-issues.ts`；预检 re-70 数到约 22 个文件 84 处）只改接收者（根 → 写入单元）与名字，断言不改。T2 第一步记下这些文件逐个的通过数与 `make test-web` 的总数，T2、T3 之后同数。`core/store/root.store.test.ts` 的 store 数（21）不变。
- **共用（T4、T7，T9–T11 中共用的部分）**：迭代页列表的测试（`core/store/issue/cycle/{issue,filter}.store.test.ts`、`core/hooks/use-issue-layout-store.test.tsx` 的迭代部分、`cycle-progress-follow.integration.test.tsx`、`cycle-preferences-follow.integration.test.tsx`、`issue-layouts/roots/{cycle-layout-root,cycle-layout-access,cycle-create,cycle-quick-actions,row-dialog-lifetime}.test.tsx`、`quick-action-dropdowns/cycle-issue.test.tsx`、`R/cycles/(detail)/header.test.tsx`、`R/cycles/(detail)/[cycleId]/page.test.tsx`、`cycles/progress-sidebar/row-filters.test.tsx`）只改导入路径、构造与调用处的名字（如本页的形状的字段、`CycleIssuesFilter` 改为直接构造、替身改为按域），断言不改；各 Task 第一步记基线收据。
- **契约变化处**（只在这里改断言）：
  - `core/hooks/store/use-issues.test.ts` 中 `useIssues("MODULE")` 被拒的一例；`core/hooks/use-issue-layout-store.test.tsx` 中 `useIssuesActions("MODULE")` 被拒的一例与 `moduleId` 映射到 DEFAULT 的一例（只改映射到的 store 类型；在别的项目新建时请求体仍恰为 `{name}`）（T6）；
  - `R/modules/(detail)/[moduleId]/page.test.tsx` 中正文只有侧栏的部分，`R/domain-pages.test.tsx` 的 “module details and its own loader”（钉的是“模块页没有工作项列表”：不取列表、没有 “Display”、没有 “Add work item”）（T8、T9）；
  - 项目页页头在偏好读到之前给布局、筛选、显示的用例（第 8 条，T9；写 plan 时没有找到这样的断言，`route-scope.test.tsx` 把 `HeaderFilters` 换成空的替身）；
  - 迭代页与项目页的列表根“偏好读到过之后的读失败替换正文”的用例（第 9 条，T12；写 plan 时 `cycle-layout-root.test.tsx` 只测第一次读失败、`project-lifecycle.test.tsx` 测的是列表的第一次读失败，都不受影响；有则按契约改并在台账写明）；
  - `core/components/modules/progress-sidebar/root.test.tsx` 的 “plain rows that choose nothing” 在没有本页筛选实例时仍成立，只在它的替身需要读筛选实例的 hook 时补替身（T11），断言不改。
- P6a 的模块页面故事（G9–G13、`g19-module-pages`）断言不改：模块详情页多了偏好与列表的读；T8 以 chromium 聚焦跑它们，若有故事记录模块详情页的精确请求表，只按契约补上这两项并在台账写明（写 plan 时它们只核 `apiFailures`、`oldApiRequests`、`pageErrors`）。
- 其他断言不改。

**检查点与证据**
- 检查点：T10 结束时（拆分、共用、store、引擎、正文、页头、快捷操作完成）串行完整七项，推送并核分支 CI，记录 e2e 作业的实际时长（期限 35 分钟；超过 30 分钟先报告负责人，不改期限或作业结构）；这时还没有新的页面故事，跑的与 main 相同，这个数是 T16 的校准点（spec §2.10）。T13、T14 记下每个新故事文件在本机三个引擎的时长，T16 推送之前照 spec §2.10 算估算、上界与校准点（基线为 P6a 的 main CI 43469787621 attempt 2 的 e2e 作业 25 分 00 秒）；慢的 runner 上可能超过 30 分钟的报告线。T16 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准）。
- 证据放在 worktree 的 `.superpowers/sdd/P6b-module-list/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，调研笔记在 `research/`。实现代理先读同目录的 `implementer-brief.md`。
- e2e 的写法照 spec §2.10 的“写法”一段（P5b、P6a 阶段记录 §5 的教训）。

## Review Focus

1. **拆分只搬不改（R1）**：写入单元持有写队列、确认的流水线、命令与请求体；根留下的读与登记不变；`observeRead` 与在途的写的配合不变；测试只改接收者；合入的状态里根上没有转发；两文件各约 400 行以内，写入单元对根没有值的导入，不加入 `project-address-route`（它带着 M4 的 router 退路，P7 去掉后加入）。
2. **共用后迭代不变（R2）**：§2.2 的表逐个落实，迭代的测试断言不改；按域的差别经 props 或按域的小表，不写 `if (domain === …)` 的分支，共用件不导入 `PeekOnEntry` 等迭代独有的件；行为不同的（列表状态、加入与移出的请求体、偏好 service、显示选项常量）分开。
3. **整集合（R5–R7）**：移出的请求体在队列轮到时由所持的整集合算出、保留已归档与别的模块；本页新建只在可加入时带本模块、与给定的取并、只发一个请求；修改与复制不覆写集合；只在 nerve 确认后跟进（离开、提示、重读进度），被拒时说明。
4. **会话与页面的范围（G19）**：挂载键与偏好的键带模块 id；换模块、后退前进、换项目、换会话时旧的回答不进新页面；预览宿主是模块页自己的。
5. **权限与只读（R9、R15）**：新建与移出给项目的 ADMIN、MEMBER 且模块可加入；已归档只读、状态不看；模块关闭时不取偏好与列表；访客看得到列表、没有写。
6. **三页的偏好读取（R3、R4）**：页头的列表控件与正文同一个条件；重读失败时正文沿用、提示可重试，第一次读失败照旧。
7. **守卫**：路由参数、`project-address-route`、`.oxlintrc.json`、告警上限、knip、i18n、按本页分组的守卫。

## Task 1（T1）：P6a 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P6a-web-modules-review.md`：抬头与 §7。
- `specs/P6a-web-modules.md`：抬头与 A8；`plans/P6a-web-modules.md`：抬头。
- `M6-design.md`：抬头的状态句、§14 的 P6a 行“已完成”，从“P6b、P7、P8、closeout”一行分出 P6b 行“进行中”（链接本段的 spec 与 plan）。
- `docs/v0/frontend-changes.md`：M6 一行中 P6a 的“待合入”改为已合入；模块的五行（数据层、筛选 store 与列表的显示设置、写入口、收藏、迭代与模块共用的取数门与件）的“已实现，待整审”改为“已完成”（P6a T15 的裁定留给本 Task）。
- `docs/v0/M8-open-release/handoffs/M6-closeout.md`：M6-M8-01 补 C7 的再现。
- `.oxlintrc.json`：swr 限制的文件表加入 `web/apps/web/core/components/cycles/active-cycle/priority-issues.tsx`（它经 `useSessionSWR` 取数而不在表里，本段没有别的 Task 改它）。

- [ ] Step 1：以 P6a 的归档为准回填 §7（P6a 的 worktree 已删除）：台账 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P6a-integration-20261114/payload/worktree/.superpowers/sdd/P6a-web-modules/progress.md` 的 Task 16 末尾，同一归档的 `cleanup.log` 与 `manifest.json`；attempt 2 的结论、归档的数字与清理已记在本段台账的“P6a 收尾”一行，T1 以它们为线索、以原件核对；作业号与时刻以公开的 jobs 接口（`https://api.github.com/repos/open-nerve/NerveProject/actions/runs/<id>/jobs`）核对，归档以 `archive.py verify` 核对（收据 `t1-p6a-archive-verify`）：
  - **最终 HEAD 的分支 CI**：`8188582e184c14a4ae9f29baf512467dbef62e6f` 的 43463416098 attempt 1 成功（server (1) 132016072027、web 132016072634、server (0) 132016072832、e2e 132025064734；e2e 作业 16:45:49Z–17:07:27Z，21 分 38 秒，在 §5 估算的 20.5–22.5 分钟内）。
  - **合入与 main CI**：noff `06b72d4b65bcba911344a4ed3c358f164a5fdeeb`，双父 `aadaf61a`、`8188582e`，树 `c8da8856` 与分支相同。main CI 43469787621 attempt 1：server (0) 132038343932、web 132038344421、server (1) 132038344536 成功；e2e 132048379310（17:25:42Z–17:49:59Z）461 过 1 败，M4 的 C7（`e2e/stories/issue/collaboration-interactions.spec.ts:422`），即 M6-M8-01 记下的间歇失败；照其验收语只重跑一次。attempt 2 只重跑 e2e：132062963902（17:51:22Z–18:16:22Z，25 分 00 秒）成功，其余三个作业沿用 attempt 1 的成功（以 jobs 接口核对）。
  - **归档**：`M6-P6a-integration-20261114` 的文件数、字节、成员数、未知数、清单 sha256 与清单的 `head`（本段台账记为 2910 个文件、1,011,232,897 字节、2938 个成员、未知 0、`head` 为 `8188582e`），以 `archive.py verify` 的收据为准。
  - **清理**：照 `cleanup.log`（本段台账记为 2026-11-13T18:18:06Z：`codex/m6-p6a` 本地与远端一致且含于 origin/main，main 与 origin/main 都在 `06b72d4b`，worktree `m6-p6`、本地与远端分支已删除）。
  - 原件与台账的数字不一致时，以原件为准并停下告诉主控。
  - 勾选 P6a spec 的 A8（注明 `make test` 的规则与上面的证据）；阶段记录、spec、plan 的抬头改为“已完成（2026-11-14，noff `06b72d4b`，main CI 43469787621 attempt 2 …）”，attempt 与结论照收据写。
- [ ] Step 2：设计的抬头与 §14；前端改动清单；M6-M8-01 补一段：P6a 合入的 main CI 43469787621 attempt 1 的 e2e 又在 C7 的同一处失败（P6a 只改了命令面板的模块文件，与主题选择无关），照本条重跑一次，attempt 2 通过（e2e 作业 25 分 00 秒，作为本段的 e2e 时长基线）；`.oxlintrc.json` 加入 `priority-issues.tsx`（它只经 `useSessionSWR` 取数，不从 `swr` 导入值）。
- [ ] Step 3：`python3 .superpowers/sdd/P6b-module-list/doc-links.py`、`node tools/keywords.mjs`、`make lint-web`（收据 `t1-*`）；提交 `docs(M6/P6b): 回填P6a的集成结果，进入P6b`（正文写明 `.oxlintrc.json` 的收紧）。

## Task 2（T2）：拆分一——写入单元与产品代码的调用方

**Files：**
- 新：写入单元（`core/store/issue/` 下，名字由实现定，如 `issue-writes.ts`）；请求体与成员快照的纯函数（同文件或旁边的文件，各约 400 行以内）。
- 改：`core/store/issue/root.store.ts`（写的一侧移出；根在构造时建写入单元；`observeRead` 经它记在途的写的读；以只读的方式给出登记的列表）；产品代码的调用方（spec §2.1 所列 12 个文件 18 处，以 grep 为准）与类型的导入。
- 不改：任何测试文件（T3）。

- [ ] Step 1：先记基线：`make test-web`（收据 `t2-baseline-test-web`，记总数）；受影响的测试文件逐个聚焦跑（以 grep 找出提到这八个名字的测试与替身作候选，逐个核对接收者是工作项的根——页面 store 的同名方法不算——收据 `t2-baseline-affected`，台账记文件与各自的通过数）。
- [ ] Step 2（GREEN）：按 spec §2.1 的表搬移，只搬不改（含 `updateIssue` 没有范围时退回 router 工作区的一支）；产品代码的调用方改经写入单元；写入单元对根只有 `import type` 或构造时传入的窄接口，没有 `root.store.ts` → 写入单元 → `root.store.ts` 的值导入（re-70 F3，以导入行核对并写进台账）；写入单元不加入 `project-address-route`（它带着搬来的 router 退路，spec §2.1，re-70 F5）；根上暂留测试所用方法的转发（注释写明只为 T3 之前的测试，T3 删去），产品代码不经它们（grep 核对）。变异（落在搬过去的代码上，既有测试须失败）：更新的请求体去掉 `cycle_id`（`t2-mut-update-drops-cycle`）、创建的请求体去掉 `cycle_id`、`observeRead` 不看在途的写（`t2-mut-read-ignores-pending`）、`pendingIDs` 恒为空、写队列不等前一个（`t2-mut-queue-no-wait`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（总数与基线同）、受影响的测试逐个同数；`wc -l` 记根与写入单元的行数；提交 `refactor(M6/P6b): 工作项根 store 拆出写入单元，产品代码改经它`。

## Task 3（T3）：拆分二——测试的调用处，删去过渡的转发

**Files：** T2 台账列出的测试文件与替身（只改接收者、spy 的对象与名字）；`core/store/issue/root.store.ts`（删去转发）。

- [ ] Step 1：列出测试里的调用处（同 T2 的 grep，台账记文件与处数）。
- [ ] Step 2（GREEN）：改接收者，删去根上的转发；核对根上没有这八个方法的定义（只查定义，例如 `rg -n '^\s+(enqueueIssueChanges|confirmCreation|createIssue|updateIssue|deleteIssue|archiveIssue|unarchiveIssue)(<[^>]*>)?\(|get pendingIDs\(' web/apps/web/core/store/issue/root.store.ts` 没有结果，收据 `t3-root-has-no-writes`；留在根上的 `observeRead` 调写入单元的那一处不算，re-70 F7），测试的差异只有接收者、spy 的对象、导入与名字（以 `git diff -U0` 过一遍，台账写明）。在终树上重跑 T2 的两个变异（`t3-mut-update-drops-cycle`、`t3-mut-read-ignores-pending`），既有测试仍失败。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（总数与 T2 的基线同）、受影响的测试逐个同数；`wc -l` 核对根与写入单元各约 400 行以内（收据 `t3-line-counts`）；提交 `refactor(M6/P6b): 测试改经写入单元，根上不留转发`。

## Task 4（T4）：共用一——迭代页的列表 store 与 hook 改为按域

**Files：**
- 新：两域共用的本页列表（`core/store/issue/helpers/` 下，名字由实现定，按本页的项取）；按域的列表动作、偏好跟进、进度跟进三个 hook（`core/hooks/` 下，名字由实现定，取代 `use-cycle-issue-actions.ts`、`use-cycle-preferences-follow.ts`、`use-cycle-progress-follow.ts`）。
- 改：`core/store/issue/cycle/issue.store.ts`（只留加入本迭代的 `createIssue`）；删除 `core/store/issue/cycle/filter.store.ts`，`root.store.ts` 以 `PagePreferencesFilter`、`CyclePreferencesService` 与 `EIssuesStoreType.CYCLE` 直接构造；读者（`use-issues-actions.tsx`、`cycle-layout-root.tsx`、`R/cycles/(detail)/header.tsx`、`use-cycle-list-state.ts` 的读者等）与迭代测试的导入、构造；`tools/keywords.json`（新文件与样例）。
- 不改：`use-cycle-list-state.ts` 的规则（spec §2.2 分开的一行）。按域的参数本 Task 只有迭代一个实参，模块在 T6 加入。

- [ ] Step 1：先跑回归边界“共用”一条所列的迭代测试，留基线收据（`t4-baseline-cycle`，台账记各文件的通过数）。
- [ ] Step 2（GREEN）：按 spec §2.2 的前八行改为共用；迭代的测试只改导入、构造与调用处的名字，断言不改全过（与基线同数）。变异（迭代的测试须失败）：共用的本页列表不看本页的项（`t4-mut-term-ignored`）、`pageTotal` 给出别的页的总数、进度跟进在被拒的写之后也重读、偏好跟进在查询变了时不重取。
- [ ] Step 3：`make lint-web`、`make knip`（`CycleIssuesFilter` 与三个旧 hook 没有残留）、`make test-web`、`node tools/keywords.mjs`；提交 `refactor(M6/P6b): 迭代页的列表 store 与 hook 改为按域共用，迭代的行为不变`。

## Task 5（T5）：模块页的偏好、列表 store 与 `module_ids` 的写入

**Files：**
- 新：`core/services/planning/module-preferences.service.ts`；`core/store/issue/module/issue.store.ts`（继承 T4 的本页列表，加模块的新建与移出）与测试。
- 改：`web/packages/types/src/issues/issue.ts`（`EIssuesStoreType.MODULE`，模块页偏好的看板折叠按它存）；`root.store.ts`（建立 `moduleIssuesFilter`、`moduleIssues`，读者即根，knip 无误）；写入单元的两份请求体放行 `module_ids` 与其测试（`root.store.planning.test.ts` 旁或写入单元自己的测试）；`core/lib/store-context.test.ts`（换代后旧一代的模块页偏好不写，如需）。

- [ ] Step 1（RED，`FakeNerve`、`fake-queue.ts`）：
  - service：`get`、`update` 的路径与请求体（`/api/v0/me/modules/{module_id}/preferences`）；
  - 模块页偏好：读到 nerve 的默认值、显示映射同项目页；没读过时整字段替换 `filters` 只发一个 `PATCH`，之后的读不盖掉它；被拒时不改、交给调用者；较旧的读不覆盖较新的修改；两个模块互不影响；换代后旧一代的回答不写；没有读 router 的 getter；
  - 列表：`fetchIssues` 的查询是本页偏好得出的查询加 `module_id: [id]`；两个模块是两个绑定；`pageTotal` 只给本页的绑定；续页只给本页的列表；
  - 新建：给出本页且目标项目是模块的项目时请求体的 `module_ids` 含本模块，调用者给了集合时取并、去重；别的项目或没有本页时请求体不变；恰好一个 `POST`；
  - 移出：工作项在 [本模块 A、模块 B、已归档的模块 C] 中，移出 A 的请求体是 `module_ids: [B, C]`，在队列轮到时算（前一个写在外时，请求体用前一个的回答：再从 B 页移出 B，请求体是 `[C]`）；确认后本页绑定不再列它；被拒时所持的集合不变、拒绝交给调用者；
  - 白名单：创建与更新的请求体带调用者给的 `module_ids`；更新的“不送整个实体”（`issue-drag.test.ts:203`）照旧。
- [ ] Step 2（GREEN）：实现。变异：移出的请求体在调用时算（`t5-mut-removal-at-call`）、移出时去掉已归档的模块（`t5-mut-removal-drops-archived`）、新建在别的项目也带本模块、新建以本模块替换给定的集合（`t5-mut-create-replaces-set`）、创建或更新的白名单去掉 `module_ids`（各一个）、查询不带 `module_id`、较旧的读覆盖较新的修改。
- [ ] Step 3：`pnpm --filter @nerve/types build` 后 `make lint-web`、`make knip`、`make test-web`；移出命令是类的成员（knip 不查），它的第一个产品读者是 T6 的按域动作，审查照此核对；提交 `feat(M6/P6b): 模块页的偏好与列表 store，module_ids 的新建与移出在轮到时算`。

## Task 6（T6）：列表引擎的 MODULE

**Files：** `web/packages/constants/src/issue/filter.ts`（`module_issues`，同一 Task 由 `ISSUE_STORE_TO_FILTERS_MAP[MODULE]` 引用）；`core/hooks/store/use-issues.ts`；`core/hooks/use-issues-actions.tsx`（MODULE 经 T4 的按域动作）；T4 的三个按域 hook 加入模块；新 `core/hooks/use-module-list-state.ts`（读者：按域的动作）；`core/hooks/use-issue-layout-store.ts`；`core/hooks/use-issue-list-mount.ts`；`core/hooks/store/use-page-issue-filters.ts`；契约变化的三个用例；`tools/keywords.json`。

- [ ] Step 1（RED）：`useIssues(MODULE)` 给出模块页的 store 与筛选；`useIssuesActions(MODULE)` 的动作经路由参数的 `moduleId`、不读 router store；模块页的路由映射到 MODULE；`useIssueListMount` 的键带模块 id；`usePageIssueFilters(MODULE)` 按路由的 `moduleId` 读；列表状态：没有模块或已归档只读，已完成、已取消的不只读，可加入还要 `module_view`；模块的动作新建只在可加入且同项目时给出本页（已归档、模块关闭时照项目页新建，请求体不带 `module_ids`）；写（新建、修改、移出、删除、归档、拖拽）经进度跟进；`module_issues` 的分组、子分组选项不含模块（R12 的守卫，同 `cycle_issues` 的测试）。
- [ ] Step 2（GREEN）：实现；按回归边界改三个契约用例。变异：挂载键去掉模块 id（`t6-mut-mount-key-no-module`）、只读按状态判断（`t6-mut-readonly-by-status`）、可加入不看 `module_view`、已归档时全局新建仍带本模块、路由仍映射到 DEFAULT。
- [ ] Step 3：`pnpm --filter @nerve/constants build` 后 `make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P6b): 列表引擎的模块页 store 类型`。

## Task 7（T7）：共用二——布局根、四种布局与空态改为按域

**Files：** 共用的布局根（取代 `issue-layouts/roots/cycle-layout-root.tsx`；不导入 `core/components/cycles/peek-on-entry.tsx`，re-70 F4）；四种布局的共用根（取代 `{list,kanban,calendar,spreadsheet}/roots/cycle-root.tsx`）；共用的空态（取代 `empty-states/cycle.tsx`）与 `empty-states/index.tsx`；按域的小表（只读的来源、行的快捷操作、空态的文案与列表状态，名字由实现定）；按域的测试替身（取代 `roots/fake-cycle-page.tsx`）；`R/cycles/(detail)/[cycleId]/page.tsx`（把 `PeekOnEntry` 作为子节点或插槽交给共用的布局根，或在它旁边渲染）；迭代测试的导入；`.oxlintrc.json`（共用的布局根）；`tools/keywords.json`（新路径与样例，旧路径留着）。

- [ ] Step 1：先跑迭代的页面与布局测试留基线收据（`t7-baseline-cycle`）。
- [ ] Step 2（GREEN）：改为按域，域只有迭代；按域的差别经 props 或按域的小表给出，共用件里不写 `if (domain === …)` 的分支，共用的布局根不导入迭代的件（以导入行核对）；迭代的测试断言不改全过（与基线同数）。变异（迭代的测试须失败）：共用的布局根在开关关闭时仍读偏好（`t7-mut-off-still-reads`）、共用的根忽略只读、空态在只读时仍给新建、预览宿主的前缀不带域。若本 Task 超出一个实现者的量，空态的共用移到 T8（台账写明），不加 Task。
- [ ] Step 3：`make lint-web`、`make knip`（被取代的迭代件没有残留）、`make test-web`、`node tools/keywords.mjs`；提交 `refactor(M6/P6b): 迭代页的布局根、四种布局与空态改为按域共用，迭代的行为不变`。

## Task 8（T8）：模块页的正文——布局根、四种布局、空态、只读与停用

**Files：** `R/modules/(detail)/[moduleId]/page.tsx`（正文在侧栏左边放列表，同迭代页）与 `page.test.tsx`；共用的布局根、四种布局的根与空态加入模块（按域的小表加模块的一行：SWR 键 `MODULE_ISSUE_FILTERS`、`module_issues.filters`、预览宿主 `["module", …]`、只读的来源、空态；模块页不给 `PeekOnEntry`）；四个 base root、`use-group-dragndrop.ts`、`use-page-filter-expression.ts` 的 store 类型加 MODULE；`command-host.tsx` 的 `DialogStore`、`quick-action-dropdowns/project-issue.tsx` 的 `storeType` 加 MODULE（模块页的行先用项目页的快捷操作，修改与复制以 MODULE 打开，移出在 T10）；`R/domain-pages.test.tsx` 的模块用例；i18n（已归档模块的空态，en 与 zh-CN）；`tools/keywords.json`。

- [ ] Step 1（RED）：组件测试（模块的替身用 T7 的按域替身）：
  - 模块页的正文渲染列表根，`IssuesStoreContext` 为 MODULE；预览的宿主是模块页自己的；四种布局按本页偏好切换；偏好第一次读失败时说明并可重试；偏好修改确认后列表按现有分页重取；
  - 模块关闭（`module_view` 为假）时列表区是停用空态，没有 `/me/modules/{id}/preferences` 与 `/projects/{id}/issues` 的请求；
  - 空态：可加入且能新建的人有“新建工作项”（`module_work_items` 的键）、有筛选时“清除筛选”（访客也有）、已归档只有说明（新键）；
  - 已归档时四种布局只读（没有快速新建、分组头的“+”、行内修改、拖拽，日历只读，快捷操作只有新标签页打开与复制链接）；已完成、已取消的模块不只读；访客看得到列表、没有新建；
  - 行的修改、复制弹窗经命令宿主以 MODULE 打开，行在核验中离开时弹窗留着（同 `row-dialog-lifetime.test.tsx` 的写法）。
- [ ] Step 2（GREEN）：实现；按回归边界改 `[moduleId]/page.test.tsx` 的正文部分与 `domain-pages.test.tsx` 的模块用例。变异：只读不看已归档（`t8-mut-archived-writable`）、模块关闭时仍读列表（`t8-mut-off-reads-list`）、空态对已归档仍给新建、预览宿主用项目页的。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；`make build build-s3-fixture` 后以 chromium 聚焦跑 P6a 的模块页面故事（`g9-*-module*`、`g10-module-links-page`、`g12-module-progress-page`、`g13-archive-delete-module-page`、`g19-module-pages`，收据 `t8-p6a-module-stories`），照回归边界核对；提交 `feat(M6/P6b): 模块页的工作项列表、空态、只读与停用`。

## Task 9（T9）：页头与新建（第 8 条：模块页与项目页）

**Files：** `R/modules/(detail)/header.tsx`（数目、布局、筛选、显示、新建）与新测试；页头三个派生值的按域 hook（从 `R/cycles/(detail)/header.tsx` 抽出，迭代页头改用，`header.test.tsx` 断言不改）；`core/components/issues/filters.tsx`（`HeaderFilters` 的 store 类型加 MODULE）；`core/components/issues/header.tsx`（项目页的 `HeaderFilters` 在本页偏好读到之后）；`web/packages/constants/src/issue/common.ts`（`TCreateModalStoreTypes`）；`list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx`、`modals/work-item-level.tsx`（锁定项目的条件为迭代页或模块页）；i18n（模块页数目的提示，en 与 zh-CN）；`tools/keywords.json`。

- [ ] Step 1（RED）：先跑迭代页头的测试留基线（`t9-baseline-cycle-header`）。模块页头：数目只在当前列表是本页的时给出（别的模块的列表答到时不显示）；四种布局、筛选开关、显示在本页偏好读到之后才出现，读到之前、第一次读失败时没有；显示选项不含模块；模块关闭时这些与数目都没有；“新建工作项”只给项目的 ADMIN、MEMBER 且模块可加入（访客、已归档、模块关闭时没有），弹窗锁定项目；新建弹窗、快速新建、分组头的“+”各只发一个带本模块 `module_ids` 的创建请求，确认后提示、行出现在本页；全局的“新建工作项”在模块页上只在可加入时带本模块；被拒（`planning.modules_disabled`、`planning.archived` 等）时说明、不提示成功（不断言拒绝码之间的次序）。项目页头：布局、筛选、显示在本页偏好读到之后才出现。
- [ ] Step 2（GREEN）：实现；迭代页头的测试断言不改全过。变异：控件不等偏好读到（`t9-mut-controls-before-read`，模块页与项目页各一）、新建的权限门去掉、可加入的条件去掉、请求体不带本模块、弹窗不锁项目、数目给出别的模块的。
- [ ] Step 3：`pnpm --filter @nerve/constants build` 后 `make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P6b): 模块页的页头与本页新建，三页的列表控件在偏好读到之后`。

## Task 10（T10）：移出模块、修改与复制、写后重读进度（检查点）

**Files：** 共用的快捷操作（取代 `quick-action-dropdowns/cycle-issue.tsx`，按域的移出命令与菜单文案）、`command-host.tsx`（“移出本页”，按域的成功提示）、`quick-action-dropdowns/helper.tsx`（菜单项按域的标签）与读者；迭代与模块的列表根改用它；i18n（`issue.remove.module.label`，en 与 zh-CN）；`tools/keywords.json`。

- [ ] Step 1（RED）：先跑迭代快捷操作的测试留基线（`t10-baseline-cycle-actions`）。模块页：“移出模块”只给工作项项目的 ADMIN、MEMBER、列表不只读、模块可加入时显示；点下后经模块 store 的移出命令，请求体是所持的集合去掉本模块（含已归档的模块），确认后行离开、提示 `issue.remove.module.success`、重读本模块的 `/progress` 一次；被拒时说明原因、行留着、不重读；同一工作项在途时再点不再发；修改的请求体不带 `module_ids`（以带着三个模块的工作项打开修改、改名字核对）；“复制一份”的请求带本模块的 `module_ids`、不带源的别的模块；迟到的确认（已换模块、换会话）不提示、不重读新页面的进度。
- [ ] Step 2（GREEN）：实现；迭代的快捷操作测试断言不改全过。变异：移出经行的 `handleUpdate`（固定的请求体，`t10-mut-removal-via-handle-update`）、移出的权限门去掉、只读时仍给、被拒后仍重读进度、迟到的确认仍重读（`t10-mut-late-follow`）、修改的请求体带上弹窗初值的 `module_ids`。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P6b): 移出模块、修改与复制，以及模块页的写后重读进度`。
- [ ] Step 4（检查点）：`sh .superpowers/sdd/P6b-module-list/gates.sh t10-gate` 串行七项（`make test` 照负责人的规则）；推送准确的 HEAD，核分支 CI，记录 e2e 作业的时长（这时还没有新的页面故事，这个数不是它们的时长）。

## Task 11（T11）：侧栏的行切换本页筛选

**Files：** 行切换的共用件（`core/components/core/sidebar/` 下，名字由实现定：按 store 类型与 id 给出 `ProgressTabs` 的 `choices`）；`core/components/cycles/progress-sidebar/progress-stats.tsx` 改用它；`core/components/modules/progress-sidebar/issue-progress.tsx` 带上 MODULE 的行切换；`modules/progress-sidebar/root.test.tsx` 的替身（如需）；`tools/keywords.json`。

- [ ] Step 1（RED）：先跑迭代侧栏的 `row-filters.test.tsx` 留基线（`t11-baseline-cycle-rows`）。模块页上状态组、负责人、标签行切换本页筛选（加入、去掉、条件空了去掉条件），经本页偏好的整字段 `filters` 保存，行以名字命名、`aria-pressed` 说明是否已选；本页的筛选实例还没有时（偏好未读到）与模块列表页的预览（`?peekModule`）里行不可点；“无负责人”“无标签”行不可点；不论模块的日期如何行都可点；已归档模块的页面上照常可点。
- [ ] Step 2（GREEN）：实现；迭代侧栏的测试断言不改全过。变异：预览里可点（`t11-mut-peek-clickable`）、切换改为替换、条件空了仍留着、用 CYCLE 的实例。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P6b): 模块页侧栏的行切换本页筛选，与迭代页共用`。

## Task 12（T12）：重读失败时沿用已读的偏好（第 9 条：三页）

**Files：** 重读失败的共用提示（名字由实现定，原因经 `useRefusalMessage`，加“重试”）；共用的布局根（迭代、模块）与 `issue-layouts/roots/project-layout-root.tsx`；i18n（提示的一句，en 与 zh-CN）。

- [ ] Step 1（RED）：三页各一组：偏好第一次读到、列表显示之后，重读（SWR 的重新校验，如 `mutate()`）失败：列表照常显示、列表上方有一行提示与“重试”，页头的布局、筛选、显示仍在；按“重试”读到后提示消失；第一次读失败照旧（模块页、迭代页是可重试的读失败，项目页是 M4 的提示文字）。
- [ ] Step 2（GREEN）：实现；按回归边界核对迭代页与项目页的既有用例（写 plan 时没有受影响的）。变异：重读失败仍替换正文（`t12-mut-replace-body`，三页各跑一次）、提示没有重试、第一次读失败也只显示提示。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`；提交 `feat(M6/P6b): 三页的偏好重读失败时正文沿用已读的偏好，提示并可重试`。

## Task 13（T13）：页面故事一（G15）

**Files：** `e2e/stories/planning/g15-module-list-page.spec.ts`（新）；列表页的夹具改为按域（`e2e/fixtures/cycle-list-pages.ts` 的世界与帮手，名字与位置由实现定；实现为 `git mv` 到 `e2e/fixtures/planning-list-pages.ts`），迭代页的 G15、G19 故事只改调用处的名字与导入。

- [ ] Step 1：夹具的共用先单独提交（迭代页的 G15、G19 以 chromium 跑通、断言不改，收据 `t13-cycle-stories-after-fixture`），再写故事（spec §2.10 的表中 G15 一行，三个故事照迭代页的三个）。每个故事先以期望错误的版本或 e2e 变异看到断言失败（收据），产品变异至少：移出时去掉已归档的模块（`t13-e2e-mut-removal-drops-archived`，故事三在数据库的归属处失败）、新建不带本模块、四种布局的根错映射（看板当作列表）。每种布局以 `expectListPageQuiet` 收尾。
- [ ] Step 2：`make build build-s3-fixture` 后 chromium 聚焦跑通；以 `--config playwright.m6-pages.config.ts` 在三种引擎上跑一次（收据），新故事另 `--repeat-each=3`；从三种引擎那次运行的报告记下新文件在各引擎的时长（台账，供 T16）；`make lint-web`；提交 `test(M6/P6b): 模块页列表的页面故事（G15）`（夹具的共用为同一 Task 的前一个提交 `test(M6/P6b): 列表页的夹具改为按域，迭代与模块共用`）。

## Task 14（T14）：页面故事二（G19）

**Files：** `e2e/stories/planning/g19-module-list-page.spec.ts`（新）；夹具（如需）。

- [ ] Step 1：写故事（spec §2.10 的表中 G19 一行）：迟到的回答用 `sentHeld`，换会话用 `anotherTabSignsIn`，换项目与后退前进用应用内移动；只看自己的访客的数与 nerve 一致。先看到期望错误的版本失败，再改正。产品变异至少：挂载键去掉模块 id、页头控件不等偏好读到、行切换用错实例；有两道守卫时照 spec §2.10 以嵌套变异证明并写明单去一道为何等价。
- [ ] Step 2：同 T13：chromium、三种引擎、`--repeat-each=3`，记下新文件在各引擎的时长；`make lint-web`；提交 `test(M6/P6b): 模块页列表的页面故事（G19）`。

## Task 15（T15）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`：更正两行（“按迭代、模块分组”一行的“结果只有一组”只对迭代页成立，模块页按本模块与本页工作项的其他模块分组；“迭代、模块页的显示设置”一行 Plane 一栏的“本页的筛选键也收下，由网页发请求时去掉”改为 Plane 把整个富筛选与本页的键一起发出、由服务端取交，因此 Plane 的模块页可按别的模块筛选，nerve 以 400 拒绝，R8）；新行：模块页列表的请求、本页新建一次发出、移出模块、修改与复制不覆写集合、已归档模块的列表只读、模块关闭时的模块页、页头控件在偏好读到之后与重读失败（三页）、侧栏的行切换、写后重读进度、空态（“清除筛选”人人有、没有“添加已有”）、按模块分组、访客看得到模块页的列表（spec R3–R9、R11–R15）。Plane 的行号对照主目录的只读 `plane/` 逐一核过；已登记的行引用不重复。
- `docs/v0/frontend-changes.md`：M6 一行（P6b）、“迭代和模块的归属”一行（模块页的新建与移出已完成）、“偏好的按页基类”一行（模块页照用、`CycleIssuesFilter` 删除）、新行：工作项根 store 的写入单元、列表件的两域共用、模块页的工作项列表、三页的偏好读取。
- 输入交接的“处理结果（M6/P6b）”：M4-closeout（M6-02 的模块部分：列表在 P6b 开放，“添加已有”、日历 `addIssuesToView`、`existing-issues-list-modal.tsx` 的模块调用方依设计 §12 的 P7 一行与 §15 第 12 条在 P7，R10；M6-01 的模块页列表部分）、M5-closeout（M5-M6-02 的模块部分）、M3-closeout（§1 模块的 user-properties）；均保持 open（设计 §11）。M8 交接 M6-M8-08：P6b 的拆法与行数，验收的结论。
- 设计：§8（模块页的列表、R2 的共用、拆出的写入单元）、§12（P6b 一行如需）、§14 的 P6b 行；spec §5 的补全。

- [ ] Step 1：写差异与回执。
- [ ] Step 2：`doc-links.py`、`node tools/keywords.mjs`；提交 `docs(M6/P6b): 交接回执、差异清单与前端改动清单`。

## Task 16（T16）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t16-gate`，留收据；`make test` 照负责人的规则），另以三种引擎的页面配置跑一次（收据）。推送之前照 spec §2.10 算出三个数并写进台账（re-70 F1）：
  - **估算**：基线 25 分 00 秒（P6a 的 main CI 43469787621 attempt 2 的 e2e 作业）加上 T13、T14 记下的新文件 chromium 时长乘以页面故事 CI 与本机之比 1.67–2.25，两个 worker 摊开；
  - **上界**：基线加上新文件 chromium 时长乘以 P6a 实际的增量比 3.6–4.1；
  - **校准点**：T10 检查点的分支 CI 实测的 e2e 作业时长与基线之比（那时跑的与 main 相同），按它换算估算在那种 runner 上的数。
  - 估算或任一次实测（T10 的检查点、推送后的分支 CI）超过 30 分钟时停下报告负责人、不推送；改 CI 的期限或作业结构须负责人同意。否则推送准确的 HEAD，核分支 CI，记录 e2e 作业的实际时长（超过 30 分钟同样先报告）。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账；
  - 写阶段记录 `reviews/P6b-module-list-review.md`；
  - 勾选 spec §4 的 A1–A7；A8 待合入后由下一段的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
