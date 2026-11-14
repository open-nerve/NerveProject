# M6 / P7a 工作项一侧（一）——写入路径与工作项自己的迭代、模块实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：待预检。基线 main `954a1838`（P6b 的 noff，双父 `06b72d4b`、`c0572192`；P6b 的最终分支 CI 43545459485 全绿，main CI 43549053092 的结论由 T1 回填）。P7 在实施之前拆为 P7a/P7b/P7c（主控裁定 D1，spec R1），本计划只管 P7a。

**Goal：**
- 写入单元的范围必给、去掉 router 退路；模块集合的变化以增减在写队列轮到时作用于所持的集合，迭代是单值字段；
- 按项目的规划目录（选项在打开时读，名字按需读，含已归档的），P7b 的目录复用它；
- 迭代、模块两个下拉框，用在工作项详情的侧栏、预览、列表与看板的行、表格的列、新建与修改的弹窗、草稿的表单与行；显示属性 `cycle`/`modules`；
- 新建时本页的值只是表单的初值（表单优先），修改发出差异，拒绝落到字段；分组头的新建弹窗移到命令宿主；复制带源的可加入的迭代与模块；
- G4、G11 的页面故事在三种引擎上，CI 的 e2e 时长在报告线以内。

**Architecture：**
- 写的一侧只在写入单元（P6b R1）：新的“按变化写”的命令把模块的增减在轮到时作用于所持的集合，列表、详情、行、表格与弹窗的修改都经它；变化的类型上没有 `module_ids`（spec §2.1、§2.2，R3）。
- 目录的读经规划 store 已有的读法与 SWR 键（P5a、P6a），选项与名字的规则在一处（spec §2.3）；下拉框建在 `@nerve/ui` 的 `CustomSearchSelect` 上（spec §2.4）。
- 页面读路由参数；按域的差别经 `planning-page-domains.ts` 的表，不写 `if (domain === …)`；跟进经 `followInSession`，拒绝经 `useRefusalToast`/`useFormRefusal`。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/react-hook-form/vitest/Playwright，不加依赖。

**Spec：** [P7a-work-item-planning](../specs/P7a-work-item-planning.md)（§2 的合同与 §3 的 R1–R21 是本计划的约束；R1–R10 是主控的 D1–D10）；设计 [M6](../M6-design.md) §3.3、§3.6、§8、§12；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：状态下拉框（`core/components/dropdowns/state/{base,dropdown}.tsx`）、成员下拉框（`dropdowns/member/base.tsx` 的多选）、目录的读（`issue-detail-widgets/directory-read.tsx`）、P6b 的“移出模块”（`core/store/issue/module/issue.store.ts:41-47`）、规划页的表（`core/hooks/planning-page-domains.ts`）、行的对话的宿主（`issue-layouts/command-host.tsx`，P5b T13）与它们的测试；e2e 的 `e2e/fixtures/{cycle-pages,module-pages,planning-list-pages,settings-pages,dialogs}.ts`、`fixtures/assert/issue-planning.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p7a`，分支 `codex/m6-p7a`，基线 main `954a1838`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/refactor/test/docs(M6/P7a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 自 T1 起为 `M6/P7`（P7a/P7b/P7c 共用）；不延长任何 `until`。本段不改 `server/`、`api/`（D10）。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内：`issue-modal/base.tsx`、`form.tsx`、`issue-layouts/utils.tsx` 先在 T4 拆开再加代码（D7）；工作项根 store（320）与写入单元（300）在约 400 行内（M6-M8-08，T15 以 `wc -l` 核对）；写入单元对根只有 `import type` 的窄接口（P6b 的 re-70 F3）。新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（变化的类型由 `IssueUpdate` 推出）；service 不转换数据；没有模块级的带令牌 service 实例；SWR 只经 `useSessionSWR`；新代码不读 router store（页面读路由参数）。
- 删掉不用的代码；不加开关、桩或兼容别名。旧的下拉框写链（`addCycleToIssue`、`changeModulesInIssue`）与乐观的分布更新不恢复（R3、R8）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 44）；清掉告警的提交同时下调，新文件不得新增告警。
- **knip**：新文件在第一个产品读者出现的 Task 里建立（所以目录与迭代下拉框随详情的迭代一栏在 T5，模块下拉框随模块一栏在 T6）；只给测试读的导出不算读者（本阶段更严的规则，逐 Task 的审查执行）；删掉某件最后一个外部读者的 Task 同时删去它或收回导出；类的成员 knip 不查，审查核对。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀）；本段读、加、删的键见 spec §2.4、§2.10。
- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例），跑 `node tools/keywords.mjs`；拆开的文件的新件同样加入，被取代的路径留在表里阻止复活。
- **`.oxlintrc.json`**：用 `useSessionSWR` 的新文件随它的 Task 加入 swr 限制的文件表。只收紧。
- **共用的假件**：组件测试要规划 store 的数据时用 `core/store/planning/fake-*.ts` 与 `fake-projects.ts`；规划页的帮手从 `issue-layouts/roots/fake-planning-page.tsx`、弹窗与编辑器的替身从 `issue-modal/fake-dialog.tsx` 导入，不复制；`fake-planning-page.tsx` 的模块级 `opened` 不在关闭时清空（A21，属 P7c），用它的测试都先打开页。

**构建与类型**
- 改了 `@nerve/types`、`@nerve/constants`、`@nerve/utils` 之后，先构建该包（`pnpm --filter <pkg> build`）再跑应用的类型检查；改了路由文件之后先跑 `react-router typegen`（`make knip` 已含）。
- 变异只计能编译的；类型检查不过的变异不算失败（e2e 的变异另要能构建）。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败（不是编译失败），再实现到通过；纯搬移（T4）与只改类型、接收者的部分（T2、T3）先以既有测试的收据为基线，之后同数。
- 下列性质每个至少一个能让测试失败的变异：增减在轮到时作用于所持的集合、已归档与不认识的 id 保留、迭代是单值、范围必给、选项（不列已结束与已归档）、值的名字（含已归档）、名字一个项目只读一次、迟到的目录回答不写、换项目时列表关上、开关关闭时不显示、只读时不可改、表单优先、修改发出差异、拒绝落到字段、分组头的弹窗活过核验、复制只带可加入的、草稿不发空的迭代、草稿换项目清空。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出、删了文件或改了路由时跑 `make knip`。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P6b 的网页单测与 e2e。
- **范围必给（T2）**：`research/web.md` §1.6 的 8 个测试文件 32 处 `updateIssue` 调用只加范围参数（`root.scope(…)`，必要时 `drop` 的 `undefined`），断言不改；T2 第一步记下这 8 个文件逐个的通过数与 `make test-web` 的总数，之后同数。
- **变化命令（T3）**：经 `BaseIssuesStore.updateIssue`、`TIssueOperations.update`、行的 `updateIssue` 与表格的 `onChange` 的测试只随类型改参数的写法（这些测试写 plan 时没有带 `module_ids` 的），断言不改；P6b 的“移出模块”的测试（`core/store/issue/module/issue.store.test.ts`、`issue-layouts/roots/module-quick-actions.test.tsx` 等，以 grep `removeIssueFromModule`、`removeIssueFromView` 为准）一行不改，基线同数。
- **拆分（T4）**：`issue-modal/` 与用到 `issue-layouts/utils.tsx` 的测试只改导入路径，断言不改，基线同数。
- **契约变化处**（只在这里改断言，台账逐条写明）：
  - 钉住 12 个显示属性的测试（`@nerve/constants`、`@nerve/utils` 的包测试，`helpers/issue-filter-helper.store` 的本地显示属性名，以 grep 为准）加 `cycle`、`modules`（T7）；
  - 钉住表单字段 `ISSUE_FORM_FIELDS` 的 `issue-modal/form-fields.test.tsx`（若钉住）加两个字段（T9）；
  - `issue-layouts/roots/module-quick-actions.test.tsx:264` 的 “joins the copy of a work item in three modules to the page's module alone” 按 R15 改为带源的可加入的模块并本模块（T11）；
  - 钉住 `draftPayloadFields` 九个字段的草稿测试（`issue-modal/draft.test.tsx` 等，以 grep 为准）加两个（T12）；
  - 在开着 `cycle_view`/`module_view` 的项目里逐个数行的控件、表格的列或 Tab 次序的组件测试（写 plan 时未见；有则 T7、T8 只按契约补上新的两项）；
  - e2e：写 plan 时没有找到会受影响的断言（既有故事按名字找列，M4、M5 的项目开关关着）；T8 检查点若有页面故事因行上、表格里多出的两项失败，只按契约补上并在台账写明。
- M4、M5 的新建与草稿的测试与 e2e（含核对数据库里草稿 payload 的）断言不改：两个值为空或字段不显示时请求里没有这两个键（spec §2.6、§2.8）。
- P5b、P6b 的新建测试（`roots/cycle-create.test.tsx`、`module-create.test.tsx`、全局新建只在可加入时带本页、别的项目的请求体恰为 `{name}`）断言不改：它们是表单优先（R2）之后本页预填的回归保护。P6b 的“修改的请求体不带 `module_ids`”同样不改（模块没改时不带）。
- 其他断言不改。

**检查点与证据**
- 检查点：T8 结束时（写入路径、目录、两个下拉框、详情与预览、行、表格完成）串行完整七项，推送并核分支 CI，记录 e2e 作业的实际时长（期限 35 分钟；超过 30 分钟先报告负责人，不改期限或作业结构）；这时还没有新的页面故事，但迭代页、模块页的故事开着开关、行上多了两个属性，这个数是 T15 的校准点（spec §2.11）。T13 记下每个新故事文件在本机三个引擎的时长；T15 推送之前照 spec §2.11 算估算、上界与校准点。T15 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准）。
- 证据放在 worktree 的 `.superpowers/sdd/P7a-work-item-planning/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，七项门禁经 `gates.sh`，调研笔记在 `research/`，主控的裁定在 `controller-rulings.md`。实现代理先读同目录的 `implementer-brief.md`（主控在 T1 之前按本计划改写）。
- 台账的行：`Task N: Step k RED/GREEN — <label> exit <code>（…）`、`Task N: 基线 — <label> exit <code>（文件与数）`、`Task N: 变异 — <label>（…）→ 失败（哪个测试），restored=yes`、`Task N: Ruling: <决定> — <理由> — 代价：<代价>`、`Task N: complete (commits <first>..<last>, tests: <summary>)`。
- e2e 的写法照 spec §2.11 的“写法”一段。

## Review Focus

1. **写入路径（R3、R19）**：写入单元的范围必给、读不到 router；模块的每一处变化都是增减、在队列轮到时作用于所持的集合；变化的类型上没有 `module_ids`；已归档与不认识的 id 保留；迭代是单值；P6b 的移出改经同一命令、测试不改；没有恢复 Plane 的写链。
2. **选项与值（R4、R13、R14）**：选项不列已结束、已归档；值以名字显示，含已结束、已归档的；名字按需读、一个项目一次，经与包装层、归档目录同一份的 SWR 键；迟到的回答不写。
3. **下拉框（R12、R20，M2 §4、§5(a)，M3 §3）**：建在共用件上（popper 在列表上、搜索框取得焦点、`modal={false}`、Escape 不外传）；模块的 `onChange` 是增减；已归档的已选项只能去掉；按钮里没有嵌套的按钮；以行为名；换项目时列表关上。
4. **各处（R11、R21）**：开关关闭时不显示；只读时不可改；显示属性的默认与本地存的页；表头没有排序。
5. **弹窗（R2、R15–R17）**：表单优先、本页的注入只给快速新建；修改不预填本页、以表单的初值算差异；拒绝落到字段；分组头的弹窗在命令宿主；复制只带可加入的。
6. **草稿（R6）**：不发空的迭代、换项目清空、行的模块增减在草稿的队列里轮到时算、发布的拒绝落到字段。
7. **拆分与守卫（R7）**：T4 只搬不改；`project-address-route`（含 `issue-writes.ts`）、`.oxlintrc.json`、告警上限、knip、i18n、phase `M6/P7`。

## Task 1（T1）：P6b 回填与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P6b-module-list-review.md`：抬头与 §7。
- `specs/P6b-module-list.md`：抬头与 A8；`plans/P6b-module-list.md`：抬头。
- `M6-design.md`：抬头的状态句（P6 已完成，P7 进行中、拆为 P7a/P7b/P7c）；§14 的 P6b 行“已完成”，从“P7、P8、closeout”一行分出 P7a 行“进行中”（链接本段的 spec 与 plan）、P7b 与 P7c 行“未开始”。
- `docs/v0/frontend-changes.md`：M6 一行中 P6b 的“待合入”改为已合入，P7 改为“P7a 进行中”。
- `docs/v0/M8-open-release/handoffs/M6-closeout.md`：M6-M8-01 补 P6b 的最终分支 CI 与合入后 main CI 的 e2e 作业时长。
- `tools/keywords.json`：`phase` 改为 `M6/P7`（A24）。

- [ ] Step 1：以 P6b 的归档为准回填 §7（P6b 的 worktree 已删除；归档在 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P6b-integration-<日期>/`，主控合入后建立）：台账 `payload/worktree/.superpowers/sdd/P6b-module-list/progress.md` 的 Task 16 末尾，同一归档的 `cleanup.log` 与 `manifest.json`；作业号与时刻以公开的 jobs 接口（`https://api.github.com/repos/open-nerve/NerveProject/actions/runs/<id>/jobs`）核对，归档以 `archive.py verify` 核对（收据 `t1-p6b-archive-verify`）：
  - **最终 HEAD 的分支 CI**：`c0572192e831957dd63939b96ec007c3e1eb875b` 的 43545459485 attempt 1 成功（server (1) 132295623381、web 132295623598、server (0) 132295623771、e2e 132300181032；e2e 作业 02:32:37Z–02:55:46Z，23 分 09 秒，低于 P6b §5 的估算约 26.1 分钟与上界 27.3–27.6 分钟）。
  - **合入与 main CI**：noff `954a1838bff44e98cc44f15df524da27e42fb1d8`，双父 `06b72d4b`、`c0572192`，树与分支相同（以 `git rev-parse <sha>^{tree}` 核对）；main CI 43549053092 的各次运行、作业号、结论与 e2e 作业时长照原件写（本 Task 读，不在 plan 里预设）。
  - **归档**：文件数、字节、成员数、未知数、清单的 sha256 与清单的 `head`，以 `archive.py verify` 的收据为准。
  - **清理**：照 `cleanup.log`（分支与 worktree 的删除、main 与 origin/main 的位置）。
  - main CI 没有全绿、归档不在或原件与台账的数字不一致时，停下告诉主控，不写“已完成”。
  - 勾选 P6b spec 的 A8（注明 `make test` 的规则与上面的证据）；阶段记录、spec、plan 的抬头改为“已完成（日期，noff `954a1838`，main CI 43549053092 …）”，attempt 与结论照收据写。
- [ ] Step 2：设计的抬头与 §14；前端改动清单；M6-M8-01 补一段：P6b 的最终分支 CI 的 e2e 作业 23 分 09 秒（同一批故事比检查点的 25 分 38 秒快约 2.5 分钟，又一个 runner 快慢的数据），合入后 main CI 的 e2e 作业时长（Step 1 的原件），作为本段的 e2e 时长基线（spec §2.11）。`tools/keywords.json` 的 `phase` 改为 `M6/P7`，`node tools/keywords.mjs` 通过（收据 `t1-keywords-phase`：没有在 `M6/P7` 到期的例外）。
- [ ] Step 3：`python3 .superpowers/sdd/P7a-work-item-planning/doc-links.py`、`node tools/keywords.mjs`、`make lint-web`（收据 `t1-*`）；提交 `docs(M6/P7a): 回填P6b的集成结果，关键词阶段进入M6/P7`（正文写明 phase 的改动）。

## Task 2（T2）：写入单元的范围必给（A10、A11）

**Files：** `core/store/issue/issue-writes.ts`（`updateIssue` 的 `operationScope` 必给，`WritesRoot` 去掉 `"workspaceSlug"`，删去 `:260-261` 的退路与注释）；`research/web.md` §1.6 的 8 个测试文件（只加范围）；`core/store/issue/root.store.ts`（router 的 getter 只在仍有读者时留着）；`tools/keywords.json`（`issue-writes.ts` 加入 `project-address-route`，带样例）。

- [ ] Step 1：先记基线：8 个测试文件逐个聚焦跑（收据 `t2-baseline-scope`，台账记各自的通过数）与 `make test-web`（`t2-baseline-test-web`，记总数）；把参数改为必给后以 `tsc --noEmit` 列出缺范围的调用处（台账记文件与处数，应为 8 个文件 32 处；不一致时以 tsc 为准并写明）。
- [ ] Step 2（GREEN）：改签名与 `WritesRoot`，32 处只加范围参数（以 `git diff -U0` 过一遍，台账写明只有这一种改动）；以 grep 核对根上 `workspaceSlug`、`projectId` 等 router getter 的读者，没有读者的删去（类的成员 knip 不查）；`issue-writes.ts` 加入 `project-address-route`。变异：守卫——在写入单元里以 `this.root.rootStore.router.workspaceSlug` 补回退路（`t2-mut-router-fallback`，`node tools/keywords.mjs` 须失败）；行为——范围的项目改取工作项自己的 `project_id`（`t2-mut-scope-from-snapshot`，能编译）须让既有测试失败，没有失败的就先补一个钉住“确认写进给出的范围的列表”的测试（RED 由这个变异给出，台账写明）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（总数与基线同）、8 个文件逐个同数；`wc -l` 记根与写入单元的行数（`t2-line-counts`）；提交 `refactor(M6/P7a): 写入单元的范围必给，去掉 router 的退路，加入 project-address-route`。

## Task 3（T3）：模块集合的变化以增减在轮到时算（D3）

**Files：**
- 改：`core/store/issue/issue-write-fields.ts`（增减作用于所持集合的纯函数）与 `issue-write-fields.test.ts`；`issue-writes.ts`（按变化写的命令，名字由实现定）与它的测试；`helpers/base-issues.store.ts`（`updateIssue` 收变化、经新命令）；`module/issue.store.ts`（移出改为 `{ remove: [本模块] }`，R19）；`core/hooks/use-issues-actions.tsx`、`core/hooks/use-planning-page-issue-actions.ts`、`issue-details/issue.store.ts`、`issue-detail/root.tsx`（`TIssueOperations` 的默认类型）、`issue-detail/use-issue-operations.ts`；行与表格的 `updateIssue`/`onChange` 的类型（以 tsc 为准：`properties/all-properties.tsx`、`spreadsheet/` 的列与行等）。
- 不改：命令面板的 `mutation-commands.ts`、`description-versions.store.ts`、拖拽的 `dropIssue`（各自已给 `makeBody`）。

- [ ] Step 1：先跑回归边界“变化命令”一条所列的测试留基线（`t3-baseline-writes`，含 P6b 的移出与经动作、详情的修改）。
- [ ] Step 2（RED，`FakeNerve`、`fake-queue.ts`）：
  - 纯函数：所持 [A, G]（G 是已归档的、页面不认识的），`{add: [B]}` → [A, G, B]；`{remove: [A]}` → [G]；加已有的、去没有的都不变；次序保留；
  - 写入单元：同一工作项第一个变化在外时排第二个——所持 [A, G]，先 `{add: [B]}` 后 `{remove: [A]}`：请求体依次 [A, G, B] 与 [G, B]（第二个以第一个的回答为底）；第一个被拒时第二个是 [G]；变化带的其余字段照给出的发（`{name, modules}` → 请求体有 `name` 与整集合）；
  - 迭代：两次 `{cycle_id}` 各发给出的值，`null` 移出，不按所持算；
  - 经 `BaseIssuesStore.updateIssue`、详情的 `updateIssue` 与 `TIssueOperations.update` 同上；变化的类型上没有 `module_ids`（类型测试里一行 `@ts-expect-error`）；
  - P6b 的移出经新命令：请求体与时机与基线同（既有测试不改）。
- [ ] Step 3（GREEN）：实现。变异：在调用时以所持算出整集合（`t3-mut-modules-at-call`）、只保留规划 store 认得的 id（`t3-mut-drop-unknown`，已归档的 G 丢掉）、加入以给出的替换所持（`t3-mut-add-replaces`）、第一个被拒后第二个仍以第一个的请求为底（`t3-mut-base-on-request`）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线的测试同数）；提交 `feat(M6/P7a): 模块集合的变化以增减在写队列轮到时作用于所持的集合，列表与详情的修改经它`。

## Task 4（T4）：到上限的三个文件先拆（D7）

**Files：** `issue-modal/base.tsx`（新建、修改、发布三个命令与描述的读各出一件）、`issue-modal/form.tsx`（提交与页脚各出一件）、`issue-layouts/utils.tsx`（拖放的件、表格的列表与图标各出一件）（spec §2.9；新文件的名字由实现定）；读者与测试的导入；`tools/keywords.json`（被拆文件在 `project-address-route` 里时，新件同样加入）。

- [ ] Step 1：先跑 `issue-modal/` 下的全部测试与导入 `issue-layouts/utils` 的测试留基线（`t4-baseline-modal`、`t4-baseline-layout-utils`，台账记文件与逐个的数）。
- [ ] Step 2（GREEN）：只搬不改（以 `git diff -M` 核对搬移，台账写明），测试只改导入；拆后各约 400 行以内（`t4-line-counts`）；knip 无误（拆出件的导出各有产品读者）。变异（落在搬过去的代码上，既有测试须失败）：新建的请求体去掉 `parent_id`（`t4-mut-create-body`）、拖拽的请求体不设 `sort_order`（`t4-mut-drop-body`）、修改的请求体带上没改过的字段（`t4-mut-update-dirty`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（与基线同数）；提交 `refactor(M6/P7a): 弹窗的 base、form 与布局的 utils 按职责拆开，行为不变`。

## Task 5（T5）：规划目录、迭代下拉框与详情、预览的迭代一栏（A1、A3）

**Files：**
- 新：选项与名字的规则（`core/lib/` 的纯模块）与读的 hook（`core/hooks/`，选项在打开时经 `useDirectoryRead`，名字按需经 `useSessionSWR`）；`core/components/dropdowns/cycle/{base,dropdown}.tsx`；详情与预览共用的规划属性一件（名字由实现定，本 Task 只有迭代一行）；各自的测试。
- 改：`issue-detail/sidebar.tsx`、`peek-overview/properties.tsx`（截止日期之后加这一件）；i18n（读 `common.cycle`、`cycle.no_cycle`、`cycle.label`；加“已归档”的说明；删 `issue.add.cycle.failed`、`issue.remove.cycle.{loading,failed}`，en 与 zh-CN）；`.oxlintrc.json`（用 `useSessionSWR` 的新文件）；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 规则：选项是未归档且状态不是 `completed` 的迭代、nerve 的次序、“无迭代”在首；名字含已结束、已归档的；
  - 读：打开时读一次项目的列表；读失败时列表底部说明并可重试；换了项目或权限身份之后旧的回答不写；名字叫不出时有角色才读，先未归档的（还没有时）后已归档的，同一项目的两处只发一个请求；没有角色不读；开关关闭时不显示也不读；
  - 下拉框：选项；选中给 id 并关上；“无迭代”给 `null`；选当前的值不调用；已结束、已归档的现值在按钮上（状态图标、已归档的说明），不在列表里；`projectId` 在列表开着时变了，列表关上，再开读新项目；`disabled` 打不开；触发器以“字段：值”为名，给出行时名字另含那一行的工作项；
  - 详情与预览：迭代一栏只在 `cycle_view` 打开时显示；不可编辑时只读；选中经 `issueOperations.update` 只发一个 `{cycle_id}` 的 `PATCH`；被拒时提示、显示 nerve 的值；连续两次选择发两个请求、各是所选的值。
- [ ] Step 2（GREEN）：实现。变异：已结束的作为选项（`t5-mut-completed-option`）、没有“无迭代”、名字按行各读一次（`t5-mut-names-per-row`，以请求数失败）、换项目时列表留着（`t5-mut-project-change-keeps-list`）、迟到的选项回答写入、不看开关（`t5-mut-ignore-switch`）、不可编辑时仍可选。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 按项目的规划目录、迭代下拉框与详情、预览的迭代一栏`。

## Task 6（T6）：模块下拉框与详情、预览的模块一栏（A2、A3）

**Files：** 新 `core/components/dropdowns/module/{base,dropdown}.tsx` 与测试；T5 的规划属性一件加模块一行（Plane 的次序：模块在迭代之前）；i18n（读 `common.modules`、`module.no_module`；多于一个时的个数文案若要新键随本 Task 加入；删 `issue.remove.module.{loading,failed}`）；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 下拉框：选项是未归档的模块（状态不论）、nerve 的次序、已选的不移到前面；工作项上已归档的模块在选项之后列为已选项、标明已归档，点它给 `{remove}`，不能再选；点一个选项给 `{add}` 或 `{remove}` 一个 id，不给整集合；选了不关；没有模块时按钮写 `module.no_module`，多于一个时显示个数；触发器里没有按钮（以 DOM 核对没有嵌套的 `button`）；以行为名同 T5；
  - 详情与预览（`FakeNerve` 的整合）：工作项在 [A, G]（G 已归档），第一个 `PATCH` 扣住时点 B、再点 A：请求体依次 [A, G, B] 与 [G, B]；去掉 G 发 [A]；`module_view` 关闭时不显示；不可编辑时只读；被拒时提示、显示 nerve 的值。
- [ ] Step 2（GREEN）：实现。变异：下拉框给出整集合、详情照发（`t6-mut-emits-set`，第二个请求体丢掉 B）、已归档的作为可选的选项（`t6-mut-archived-as-option`）、已归档的已选项不列出（`t6-mut-archived-not-listed`，去不掉 G）、已选的移到前面（`t6-mut-selected-first`）、选了就关。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 模块下拉框与详情、预览的模块一栏，模块的变化是增减`。

## Task 7（T7）：显示属性与列表、看板的行（A1、A2）

**Files：** `web/packages/types/src/view-props.ts`（`IIssueDisplayProperties` 加 `cycle`、`modules`）；`web/packages/constants/src/issue/common.ts`（`ISSUE_DISPLAY_PROPERTIES_KEYS`、`ISSUE_DISPLAY_PROPERTIES`，标题 `common.cycle`、`common.modules`）；`web/packages/utils/src/work-item/base.ts`（`getComputedDisplayProperties` 默认 `true`）；`core/store/issue/helpers/issue-filter-helper.store.ts`（`displayPropertyNames`）；`issue-layouts/filters/header/display-filters/display-properties.tsx`（项目的页上按路由的项目的开关给两项）；`issue-layouts/properties/` 新一件行的属性与 `all-properties.tsx` 渲染它的一处；契约变化的包测试；i18n（以行为名若要新键，R20）；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 默认：`getComputedDisplayProperties({})` 的 `cycle`、`modules` 为 `true`；本地存的页（个人页、归档页）收下并保留这两个布尔值；
  - 项目页：在显示面板关掉 `cycle`，发出 `PATCH /api/v0/projects/{id}/preferences`（`display_properties.cycle` 为假），行上不再有迭代；面板在路由的项目只开了模块时只给“模块”一项、都关时两项都不给；个人页的面板两项都给；迭代页、模块页照常给两项；
  - 行：显示属性打开且行的项目开着开关时显示；`isReadOnly` 时只读；迭代页上把一行改到别的迭代，只发一个 `PATCH`，确认后行离开本页、本迭代的进度重读一次（既有的跟进）；行的模块变化是增减；个人页上两个项目的行，名字叫不出时每个项目各读一次；行的触发器以那一行为名。
- [ ] Step 2（GREEN）：先 `pnpm --filter @nerve/types build`、`@nerve/constants`、`@nerve/utils` 再实现；按回归边界改契约变化处。变异：默认为假（`t7-mut-default-false`）、面板不看开关（`t7-mut-panel-ignores-switch`）、行不看显示属性、行不看行的项目的开关（`t7-mut-row-ignores-switch`）、本地存的页丢掉两个键（`t7-mut-local-drops-keys`）。若本 Task 超出一个实现者的量，显示面板的开关门移到 T8（台账写明），不加 Task。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、包的测试、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 显示属性的迭代与模块，列表与看板的行上的两个属性`。

## Task 8（T8）：表格的两列（检查点）

**Files：** `web/packages/constants/src/issue/common.ts`（`SPREADSHEET_PROPERTY_LIST` 在标签之后加 `modules`、`cycle`；`SPREADSHEET_PROPERTY_DETAILS` 给标题与图标，两项没有排序的键：类型上排序的键对它们可省，名字与写法由实现定，R21）；`issue-layouts/spreadsheet/columns/` 新两列；T4 拆出的表格的列表与图标加两项；`spreadsheet/columns/header-column.tsx`、`spreadsheet/spreadsheet-sort.ts`（没有排序键的列没有排序菜单、没有 `aria-sort`）；测试；`tools/keywords.json`。

- [ ] Step 1（RED）：两列在显示属性打开、项目开着开关时出现，次序在标签之后（模块、迭代）；表头没有排序菜单；格里改迭代发 `{cycle_id}`、改模块是增减；只读时不可改；已归档的值以名字与说明显示。
- [ ] Step 2（GREEN）：实现。变异：两列的表头给出排序（`t8-mut-header-sorts`，以菜单项与 `aria-sort` 失败）、两列的次序颠倒、格里发整集合（`t8-mut-cell-sends-set`）。
- [ ] Step 3：先构建 `@nerve/constants`；`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 表格的迭代与模块两列`。
- [ ] Step 4（检查点）：`sh .superpowers/sdd/P7a-work-item-planning/gates.sh t8-gate` 串行七项（`make test` 照负责人的规则）；既有页面故事若因行上、表格里多出的两项失败，按回归边界只补契约并在台账写明；推送准确的 HEAD，核分支 CI，记录 e2e 作业的时长（校准点；超过 30 分钟先报告负责人，不改期限或作业结构）。

## Task 9（T9）：新建弹窗的字段与表单优先（D2）

**Files：** `issue-modal/types.ts`（`IssueFormValues`、`ISSUE_FORM_FIELDS` 加两个字段）；`web/packages/constants/src/issue/modal.ts`（默认 `cycle_id: null`、`module_ids: []`）；`issue-modal/components/default-properties.tsx` 与新的规划字段一件（两个下拉框，模块字段把增减作用于表单自己的值，只在表单的项目开着开关时）；T4 拆出的新建命令（请求体加 `cycle_id`、`module_ids`，为空时省略）；`core/hooks/planning-page-domains.ts`（本页的初值与“确认的工作项在本页”两列，名字由实现定）；`core/hooks/use-planning-page-issue-actions.ts`（给表单的照给出的新建）与 `use-issues-actions.tsx`（其余 store 类型的同名命令就是新建）；新建的初值（spec §2.6，名字由实现定）；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 可加入的迭代页上新建，表单的迭代一栏是本迭代；改成别的迭代，只发一个 `POST`、带那个迭代（表单优先）；选“无迭代”，请求体没有 `cycle_id` 这个键；不可加入（已结束、已归档、开关关闭、实体属别的项目）时没有预填；
  - 可加入的模块页上新建，表单的模块一栏是本模块；去掉它，请求体不带它；再加一个，两个都带；
  - 换项目：两个字段清空，请求体都不带；表单的项目没开开关时字段不显示；
  - 两个值为空（`null`、`[]`）或字段不显示时，新建与“存为草稿”的请求里都没有这两个键（M4、M5 的新建与草稿的请求体照旧，spec §2.6）；
  - 调用者给的 `data`（分组的 payload）中是选项的留着、不是的去掉，本页的补上；
  - 快速新建在可加入的规划页上照旧加入本页（注入）；全局新建、页头、分组头在规划页上只在可加入时带本页（P5b、P6b 的测试照旧）；
  - 确认的工作项在本页时本页的进度重读一次，不在本页（选了别的迭代）时不重读。
- [ ] Step 2（GREEN）：实现；按回归边界改契约变化处（`form-fields.test.tsx` 若钉住字段）。变异：表单的新建仍经本页的注入（`t9-mut-page-overwrites-form`，选别的迭代时请求体是本迭代）、预填不看可加入（`t9-mut-prefill-ignores-joinable`）、`cycle_id` 为空时发 `null`（`t9-mut-null-cycle-sent`）、空的 `module_ids` 进“存为草稿”的请求（`t9-mut-empty-modules-in-draft`）、换项目不清空（`t9-mut-reset-keeps-planning`）、快速新建不加入本页（`t9-mut-quick-add-no-join`）、进度在不在本页时也重读。
- [ ] Step 3：先构建 `@nerve/constants`；`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 新建弹窗的迭代与模块字段，本页预填、表单优先`。

## Task 10（T10）：修改发出差异，拒绝落到字段

**Files：** T4 拆出的修改命令（模块的差异经 T3 的命令）与提交件（以表单挂上时的初值算增减）；`useFormRefusal` 的字段与 `planning.cycle_completed` → `cycle_id`（spec §2.6，R17）；测试。

- [ ] Step 1（RED）：
  - 修改显示工作项真实的值（已归档的模块、已结束的迭代），在别的迭代的页上打开修改，迭代一栏是工作项的而不是本页的；
  - 只改名字，请求体没有模块与迭代（P6b 的用例照旧）；
  - 弹窗打开后它自己读工作项时读到多出的模块 D，表单显示打开时的 [B, G]，去掉 B 保存：请求体是 [G, D]（差异在轮到时作用于所持的集合）；加一个同理；改迭代带 `{cycle_id}`；
  - 422 `module_ids[1]` 落到模块一栏并取得焦点；422 `cycle_id` 落到迭代一栏；409 `planning.cycle_completed` 落到迭代一栏；409 `planning.archived` 照旧提示。
- [ ] Step 2（GREEN）：实现。变异：修改发出表单的整集合（`t10-mut-edit-whole-set`，D 丢掉）、以随读取变化的 `data` 作初值（`t10-mut-diff-from-data`）、修改预填本页（`t10-mut-prefill-on-edit`）、`planning.cycle_completed` 只提示（`t10-mut-cycle-completed-toast`）、`module_ids[i]` 不落到字段。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`；提交 `feat(M6/P7a): 修改发出迭代与模块的差异，拒绝落到字段`。

## Task 11（T11）：分组头的新建弹窗与复制（A13、A14）

**Files：** `issue-layouts/command-host.tsx`（对话加一种“新建”，带分组的 `data` 与 store 类型；`copyOf` 带源的 `cycle_id`、`module_ids`）；`list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx`（经命令宿主请求，不再自己渲染弹窗）；测试（照 `roots/row-dialog-lifetime.test.tsx` 与 `fake-planning-page.tsx` 的 `outlivesTheRow`）；`issue-modal/content-copy.test.tsx`；契约变化的 `roots/module-quick-actions.test.tsx:264`；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 分组头的“+”打开的新建弹窗在列表核验（布局换成加载）时留着，提交只发一个请求；规划页上照旧锁定项目；
  - 复制：源在 [A, G]（G 已归档）、在已结束的迭代里：复制的表单带 A、不带 G 与那个迭代；模块页上再并本模块；迭代页上复制本页的行照旧加入本迭代（`cycle-quick-actions.test.tsx:175` 照旧）；
  - 描述带图片的复制经内容复制：草稿的 payload 带表单的迭代与模块，发布后工作项在其中（本页的迭代或模块）。
- [ ] Step 2（GREEN）：实现；按回归边界改 `module-quick-actions.test.tsx:264`（台账写明 R15）。变异：弹窗留在分组头里（`t11-mut-header-modal-in-group`）、`copyOf` 不带迭代与模块（`t11-mut-copy-no-planning`：P6b 记下的嵌套变异现在失败）、复制带上已归档的模块（`t11-mut-copy-keeps-archived`）、内容复制丢掉两个值（`t11-mut-content-copy-drops-planning`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 分组头的新建弹窗移到命令宿主，复制带源的可加入的迭代与模块`。

## Task 12（T12）：草稿（D6）

**Files：** `issue-modal/draft-form.ts`（`draftPayloadFields` 加两个）；T4 拆出的提交件（草稿换项目时清空两者）；T4 拆出的发布命令与 `handleMoveToProjects`（发布的拒绝落到字段）；`workspace-draft/draft-issue-properties.tsx`（两个下拉框，模块的增减经草稿 store 的 `input(held)`）；测试（`issue-modal/draft.test.tsx`、`workspace-draft/*.test.tsx`）；契约变化的草稿字段测试；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 草稿的表单显示 payload 里的迭代与模块（含已结束、已归档的）；选“无迭代”保存，payload 没有 `cycle_id`（不是 `null`、不是 `""`）；从弹窗“存为草稿”同样；
  - 草稿换项目：两者清空，保存的 payload 没有它们；
  - 草稿的行：改迭代发草稿的 `PATCH`（payload 带 `cycle_id`）；第一个 `PATCH` 扣住时连点两个模块，第二个 payload 以第一个的回答为底，payload 的其余键照所持的保留；草稿的项目没开开关时不显示；两个项目的草稿各在打开时读自己项目的选项；
  - 在草稿的弹窗里发布被拒：422 `cycle_id` 落到迭代一栏、草稿留着；409 `planning.cycle_completed` 落到迭代一栏。
- [ ] Step 2（GREEN）：实现；按回归边界改草稿字段的契约测试。变异：保存发 `cycle_id: null`（`t12-mut-null-cycle-draft`）、换项目不清空（`t12-mut-project-change-keeps`）、行的模块在调用时算（`t12-mut-row-set-at-call`）、发布的拒绝只提示（`t12-mut-publish-toast-only`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 草稿的迭代与模块：表单、行与发布`。

## Task 13（T13）：页面故事（G4、G11）

**Files：** `e2e/stories/planning/g4-issue-cycle-page.spec.ts`、`g11-issue-modules-page.spec.ts`（新）；世界与帮手若要共用（名字由实现定，例如 `e2e/fixtures/work-item-planning-pages.ts`）；已有的帮手从原处导入（spec §2.11）。

- [ ] Step 1：写两个故事（spec §2.11 的表）。每个故事先以期望错误的版本看到断言失败（收据），再写产品变异：G4——已结束的迭代作为选项（`t13-e2e-mut-completed-option`）、表单的新建仍经本页的注入（`t13-e2e-mut-form-injected`，请求体是本迭代）、名字不读已归档的列表（`t13-e2e-mut-no-archived-names`）；G11——下拉框发整集合（`t13-e2e-mut-dropdown-set`，数据库里没有 Beta）、丢掉已归档的模块（`t13-e2e-mut-drop-archived`）、修改发整集合（`t13-e2e-mut-edit-whole-set`，数据库里没有 Delta）。变异的脚本 `t13-e2e.sh` 依次跑 `react-router typegen`、web 与 e2e 的 `tsc --noEmit`、`make build build-s3-fixture` 与 chromium 的故事，只计能编译、能构建的。有两道守卫时以嵌套变异证明并写明单去一道为何等价。
- [ ] Step 2：`make build build-s3-fixture` 后 chromium 聚焦跑通；以 `--config playwright.m6-pages.config.ts` 在三种引擎上跑一次（收据），新故事另 `--repeat-each=3`；从三种引擎那次运行的报告记下两个新文件在各引擎的时长（台账，供 T15）；两个文件本机 chromium 合计超过 25 秒时先压缩故事再继续（台账写明，D9）；`make lint-web`；提交 `test(M6/P7a): 工作项的迭代与模块的页面故事（G4、G11）`（两个文件可各一个提交）。

## Task 14（T14）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`：新行（spec R2–R6、R11–R18 中改变行为的）：下拉框的写法（增减在轮到时算、一个请求、被拒时说明，plane-contract §8 第 2、3 条）；迭代的选项与已结束、已归档的现值；模块的选项、已归档的已选项只能去掉、没有嵌套的移除按钮、已选的不移到前面；属性随项目的开关；新建弹窗的表单优先与本页的初值（扩充“迭代页上的新建”“模块页上的新建”两行）；修改发出差异（扩充“模块页行的修改与复制”一行）；复制带源的可加入的值；分组头的新建弹窗在命令宿主；草稿的界面（不发空的迭代、换项目清空，扩充“建立与发布草稿时的归属”一行）；跨项目的页按项目读目录；表格两列的表头暂无排序（P7b）。Plane 的行号对照主目录的只读 `plane/` 逐一核过；已登记的行引用不重复；P6b 写着“在 P7”的行（如“模块页与迭代页的空态”）只改到本段做了的部分。
- `docs/v0/frontend-changes.md`：M6 一行（P7a）；“迭代和模块的归属”一行（下拉框的写法、新建弹窗与草稿已完成，“添加已有”在 P7b）；“工作项根 store 的写入单元”一行（范围必给、按变化写的命令）；新行：迭代与模块的下拉框、工作项上的迭代与模块（详情、预览、行、表格、显示属性）、新建与修改的弹窗与草稿、分组头的弹窗宿主、三个文件的拆分。
- 输入交接的“处理结果（M6/P7a）”：[M2-closeout](../handoffs/M2-closeout.md) §4 的三点（第一点：列表开着时换项目关上、再开读新项目；第二、三点照 P5a 的结论）与 §5(a) 两个下拉框（列表在按钮旁、搜索框取得焦点、`Combobox` 一份打开的状态，G4、G11 的页面故事在三种引擎核对）；[M3-closeout](../handoffs/M3-closeout.md) §3 的“弹窗里的列表”中两个下拉框（工作项弹窗里 `modal={false}`，G4、G11 核对）与“每一行的控件”中行与表格的下拉框；其余各项注明在 P7b、P7c；均保持 open（设计 §11）。M8 交接 [M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)：本段之后根与写入单元的行数；[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)：T8 检查点的分支 CI 的 e2e 作业时长。
- 设计：§8 的“工作项一侧”写明本段做了的（下拉框、属性、显示属性、弹窗与草稿、复制，写入单元的变化命令）与留给 P7b、P7c 的；§12 的 P7 一行拆为 P7a、P7b、P7c 三行（各写交付边界与验证，引本 spec 的 §1、R1）；§14 的 P7a 行；spec §5 的补全。

- [ ] Step 1：写差异与回执。
- [ ] Step 2：`doc-links.py`、`node tools/keywords.mjs`；提交 `docs(M6/P7a): 交接回执、差异清单、前端改动清单与设计中 P7 的拆分`。

## Task 15（T15）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t15-gate`，留收据；`make test` 照负责人的规则），另以三种引擎的页面配置跑一次（收据）；`wc -l` 记根、写入单元与 T4 拆出的各件的行数（M6-M8-08）。推送之前照 spec §2.11 算出三个数并写进台账：
  - **估算**：基线（T1 回填的 P6b 的 main CI 的 e2e 作业时长）加上 T13 记下的新文件 chromium 时长 × 1.67–2.25，两个 worker 摊开；
  - **上界**：已测到的最慢的数据点（25 分 38 秒、T8 的检查点与 T1 回填的数中最大的）加上新文件 chromium 时长 × 3.6–4.1；
  - **校准点**：T8 检查点的分支 CI 实测的 e2e 作业时长与基线之比，按它换算估算在那种 runner 上的数。
  - 估算、上界或任一次实测（T8 的检查点、推送后的分支 CI）超过 30 分钟时停下报告负责人、不推送；改 CI 的期限或作业结构须负责人同意（D9）。否则推送准确的 HEAD，核分支 CI，记录 e2e 作业的实际时长（超过 30 分钟同样先报告）。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账或在修复轮做（同 P6b 的修复轮 F）；修复轮另经一次限定复审；
  - 写阶段记录 `reviews/P7a-work-item-planning-review.md`（§7 留给 P7b 的 T1 回填）；
  - 勾选 spec §4 的 A1–A11；A12 待合入后由 P7b 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
