# M6 / P7a 工作项一侧（一）——写入路径与工作项自己的迭代、模块实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（修正后，[re-72](../../../reviews/re-72-M6P7a预检.md)），待实施：F1（Important）变化的类型声明 `module_ids?: never`、命令逐个点名写请求体、类型测试用 `Partial<IssueDetail>` 的值（T3）；F2（Important）新建的初值在弹窗一级并进表单的默认值，迭代页、模块页的新建测试因此仍是 R2 的回归保护（T10）；F3（Important）原 T5 拆为 T5、T6，计划为 16 个 Task；F4（Important）分组头的新建经命令宿主时宿主改三处（T12）；F5 慢 runner 与 27 分 41 秒的数据点、P7b/P7c 的 e2e 预算（T16、spec §2.11、§5）；F6 草稿 `draftPayload` 不改，`module_ids: []` 照发（T13）；F7 初值的过滤等两份未归档的列表答到（T10、T12）；F8 调用者 `data` 的过滤移到 T12；F9 `issue-list-binding.ts` 交 P7b 先拆（spec §5）；F10 表格两列的表头的两处改法（T9）。基线 main `954a1838`（P6b 的 noff，双父 `06b72d4b`、`c0572192`；P6b 的最终分支 CI 43545459485 全绿；合入后的 main CI 43549053092 只在 M4 的 I4 (page) 的表格故事失败，由修复分支 `codex/m6-p6b-ci` 处理，修复的 noff 在 T1 之前并入本分支，T1 回填）。P7 在实施之前拆为 P7a/P7b/P7c（主控裁定 D1，spec R1），本计划只管 P7a。

**Goal：**
- 写入单元的范围必给、去掉 router 退路；模块集合的变化以增减在写队列轮到时作用于所持的集合，迭代是单值字段；
- 按项目的规划目录（选项在打开时读，名字按需读，含已归档的），P7b 的目录复用它；
- 迭代、模块两个下拉框，用在工作项详情的侧栏、预览、列表与看板的行、表格的列、新建与修改的弹窗、草稿的表单与行；显示属性 `cycle`/`modules`；
- 新建时本页的值只是表单的初值（表单优先），修改发出差异，拒绝落到字段；分组头的新建弹窗移到命令宿主；复制带源的可加入的迭代与模块；
- G4、G11 的页面故事在三种引擎上，CI 的 e2e 时长在报告线以内（或越过之前停下报告负责人）。

**Architecture：**
- 写的一侧只在写入单元（P6b R1）：新的“按变化写”的命令把模块的增减在轮到时作用于所持的集合，列表、详情、行、表格与弹窗的修改都经它；变化的类型声明 `module_ids?: never`（spec §2.1、§2.2，R3）。
- 目录的读经规划 store 已有的读法与 SWR 键（P5a、P6a），选项与名字的规则在一处（spec §2.3）；下拉框建在 `@nerve/ui` 的 `CustomSearchSelect` 上（spec §2.4）。
- 页面读路由参数；按域的差别经 `planning-page-domains.ts` 的表，不写 `if (domain === …)`；跟进经 `followInSession`，拒绝经 `useRefusalToast`/`useFormRefusal`。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/react-hook-form/vitest/Playwright，不加依赖。

**Spec：** [P7a-work-item-planning](../specs/P7a-work-item-planning.md)（§2 的合同与 §3 的 R1–R21 是本计划的约束；R1–R10 是主控的 D1–D10）；预检 [re-72](../../../reviews/re-72-M6P7a预检.md)；设计 [M6](../M6-design.md) §3.3、§3.6、§8、§12；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：状态下拉框（`core/components/dropdowns/state/{base,dropdown}.tsx`）、成员下拉框（`dropdowns/member/base.tsx` 的多选）、目录的读（`issue-detail-widgets/directory-read.tsx`）、P6b 的“移出模块”（`core/store/issue/module/issue.store.ts:41-47`）、规划页的表（`core/hooks/planning-page-domains.ts`）、行的对话的宿主（`issue-layouts/command-host.tsx`，P5b T13）与它们的测试；e2e 的 `e2e/fixtures/{cycle-pages,module-pages,planning-list-pages,settings-pages,dialogs}.ts`、`fixtures/assert/issue-planning.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p7a`，分支 `codex/m6-p7a`，基线 main `954a1838`（T1 之前并入 P6b 的 CI 修复的 noff）；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/refactor/test/docs(M6/P7a)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 自 T1 起为 `M6/P7`（P7a/P7b/P7c 共用）；不延长任何 `until`。本段不改 `server/`、`api/`（D10）。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内：`issue-modal/base.tsx`、`form.tsx`、`issue-layouts/utils.tsx` 先在 T4 拆开再加代码（D7）；工作项根 store（320）与写入单元（300）在约 400 行内（M6-M8-08，T16 以 `wc -l` 核对）；写入单元对根只有 `import type` 的窄接口（P6b 的 re-70 F3）。新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（变化的类型由 `IssueUpdate` 推出）；service 不转换数据；没有模块级的带令牌 service 实例；SWR 只经 `useSessionSWR`；新代码不读 router store（页面读路由参数）。
- 删掉不用的代码；不加开关、桩或兼容别名。旧的下拉框写链（`addCycleToIssue`、`changeModulesInIssue`）与乐观的分布更新不恢复（R3、R8）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 44）；清掉告警的提交同时下调，新文件不得新增告警。
- **knip**：新文件在第一个产品读者出现的 Task 里建立（规则模块、选项的读与迭代下拉框随侧栏的迭代一栏在 T5，名字的按需读随它的第一个显示者在 T6，模块下拉框随模块一栏在 T7）；只给测试读的导出不算读者（本阶段更严的规则，逐 Task 的审查执行）；删掉某件最后一个外部读者的 Task 同时删去它或收回导出；类的成员 knip 不查，审查核对。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀）；本段读、加、删的键见 spec §2.4、§2.10。
- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例），跑 `node tools/keywords.mjs`；拆开的文件的新件同样加入，被取代的路径留在表里阻止复活。
- **`.oxlintrc.json`**：用 `useSessionSWR` 的新文件随它的 Task 加入 swr 限制的文件表。只收紧。
- **共用的假件**：组件测试要规划 store 的数据时用 `core/store/planning/fake-*.ts` 与 `fake-projects.ts`；规划页的帮手从 `issue-layouts/roots/fake-planning-page.tsx`、弹窗与编辑器的替身从 `issue-modal/fake-dialog.tsx` 导入，不复制；`fake-planning-page.tsx` 的模块级 `opened` 不在关闭时清空（A21，属 P7c），用它的测试都先打开页。

**构建与类型**
- 改了 `@nerve/types`、`@nerve/constants`、`@nerve/utils` 之后，先构建该包（`pnpm --filter <pkg> build`）再跑应用的类型检查；改了路由文件之后先跑 `react-router typegen`（`make knip` 已含）。
- 变异只计能编译的；类型检查不过的变异不算失败（e2e 的变异另要能构建）。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败（不是编译失败），再实现到通过；纯搬移（T4）与只改类型、接收者的部分（T2、T3）先以既有测试的收据为基线，之后同数。
- 下列性质每个至少一个能让测试失败的变异：增减在轮到时作用于所持的集合、已归档与不认识的 id 保留、迭代是单值、范围必给、选项（不列已结束与已归档）、值的名字（含已归档）、名字一个项目只读一次、迟到的目录回答不写、换项目时列表关上、开关关闭时不显示、只读时不可改、表单优先（初值在弹窗一级）、初值的过滤等列表答到、修改发出差异、拒绝落到字段、分组头的弹窗活过核验、复制只带可加入的、草稿不发空的迭代、草稿换项目清空。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出、删了文件或改了路由时跑 `make knip`。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P6b 的网页单测与 e2e。
- **范围必给（T2）**：`research/web.md` §1.6 的 8 个测试文件 32 处 `updateIssue` 调用只加范围参数（`root.scope(…)`，必要时 `drop` 的 `undefined`），断言不改；T2 第一步记下这 8 个文件逐个的通过数与 `make test-web` 的总数，之后同数。
- **变化命令（T3）**：经 `BaseIssuesStore.updateIssue`、`TIssueOperations.update`、行的 `updateIssue` 与表格的 `onChange` 的测试只随类型改参数的写法（这些测试写 plan 时没有带 `module_ids` 的），断言不改；P6b 的“移出模块”的测试（`core/store/issue/module/issue.store.test.ts`、`issue-layouts/roots/module-quick-actions.test.tsx` 等，以 grep `removeIssueFromModule`、`removeIssueFromView` 为准）一行不改，基线同数。
- **拆分（T4）**：`issue-modal/` 与用到 `issue-layouts/utils.tsx` 的测试只改导入路径，断言不改，基线同数。
- **契约变化处**（只在这里改断言，台账逐条写明）：
  - 钉住 12 个显示属性的测试（`@nerve/constants`、`@nerve/utils` 的包测试，`helpers/issue-filter-helper.store` 的本地显示属性名；写 plan 时 grep 没有找到，有则加 `cycle`、`modules`）（T8）；
  - 钉住表单字段 `ISSUE_FORM_FIELDS` 的测试（写 plan 时 `issue-modal/form-fields.test.tsx` 没有钉住，有则加两个字段）（T10）；
  - `issue-layouts/roots/module-quick-actions.test.tsx:264` 的 “joins the copy of a work item in three modules to the page's module alone” 按 R15 改为带源的可加入的模块并本模块（T12）；
  - 逐字核对**新**草稿请求体（或数据库里新草稿的 payload）的测试与 e2e：新草稿可以多出 `module_ids: []`（re-72 F6 的裁定：`draftPayload` 不改，空集合照发）；只在这样的断言上补上这一键（T13，以 grep 与 T9 检查点的运行为准）；已有草稿的修改只发改过的字段，请求体照旧；
  - 在开着 `cycle_view`/`module_view` 的项目里逐个数行的控件、表格的列或 Tab 次序的组件测试（写 plan 时未见；有则 T8、T9 只按契约补上新的两项）；
  - e2e：写 plan 时没有找到会受影响的断言（既有故事按名字找列，M4、M5 的项目开关关着）；T9 检查点若有页面故事因行上、表格里多出的两项失败，只按契约补上并在台账写明。
- M4、M5 的非草稿新建的请求体照旧：两个值为空或字段不显示时请求体里没有这两个键（spec §2.6）。
- P5b、P6b 的新建测试断言不改，它们是表单优先（R2）之后本页预填的回归保护：`roots/cycle-create.test.tsx`、`module-create.test.tsx`（页头、快速新建、分组头、全局新建只在可加入时带本页；别的项目的请求体恰为 `{name}`）。这两个文件在 `:66-68` 把 `IssueDefaultProperties` 换成 `() => null`，它们的断言（`cycle-create.test.tsx:155-160`、`module-create.test.tsx:162`）今天只靠 store 的注入成立；T10 把初值在弹窗一级并进表单的默认值，字段的组件不在场时请求体照样带本页的值，这些测试一行不改仍然通过（re-72 F2）。P6b 的“修改的请求体不带 `module_ids`”同样不改（模块没改时不带）。
- 其他断言不改。

**检查点与证据**
- 检查点：T9 结束时（写入路径、目录、两个下拉框、详情与预览、行、表格完成）串行完整七项，推送并核分支 CI，记录 e2e 作业的实际时长（期限 35 分钟；超过 30 分钟先报告负责人，不改期限或作业结构）；这时还没有新的页面故事，但迭代页、模块页的故事开着开关、行上多了两个属性，这个数是 T16 的校准点（spec §2.11）。T14 记下每个新故事文件在本机三个引擎的时长；T16 推送之前照 spec §2.11 算估算、上界与校准点。T16 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准）。
- 证据放在 worktree 的 `.superpowers/sdd/P7a-work-item-planning/`：台账 `progress.md`，收据经 `run-check.py`，变异经 `mutate.py`，七项门禁经 `gates.sh`，调研笔记在 `research/`，主控的裁定在 `controller-rulings.md`。实现代理先读同目录的 `implementer-brief.md`。
- 台账的行：`Task N: Step k RED/GREEN — <label> exit <code>（…）`、`Task N: 基线 — <label> exit <code>（文件与数）`、`Task N: 变异 — <label>（…）→ 失败（哪个测试），restored=yes`、`Task N: Ruling: <决定> — <理由> — 代价：<代价>`、`Task N: complete (commits <first>..<last>, tests: <summary>)`。
- e2e 的写法照 spec §2.11 的“写法”一段。

## Review Focus

1. **写入路径（R3、R19）**：写入单元的范围必给、读不到 router；模块的每一处变化都是增减、在队列轮到时作用于所持的集合；变化的类型声明 `module_ids?: never`、`Partial<IssueDetail>` 的值赋不进去，命令逐个点名写请求体；已归档与不认识的 id 保留；迭代是单值；P6b 的移出改经同一命令、测试不改；没有恢复 Plane 的写链。
2. **选项与值（R4、R13、R14）**：选项不列已结束、已归档；值以名字显示，含已结束、已归档的；名字按需读、一个项目一次，经与包装层、归档目录同一份的 SWR 键；迟到的回答不写。
3. **下拉框（R12、R20，M2 §4、§5(a)，M3 §3）**：建在共用件上（popper 在列表上、搜索框取得焦点、`modal={false}`、Escape 不外传）；模块的 `onChange` 是增减；已归档的已选项只能去掉；按钮里没有嵌套的按钮；以行为名；换项目时列表关上。
4. **各处（R11、R21）**：开关关闭时不显示；只读时不可改；显示属性的默认与本地存的页；表头没有排序。
5. **弹窗（R2、R15–R17）**：初值在弹窗一级并进表单的默认值、按选项过滤等两份列表答到；表单优先、本页的注入只给快速新建；修改不预填本页、以表单的初值算差异；拒绝落到字段；分组头的弹窗在命令宿主（宿主持有对话的项目、锁定项目、`DialogStore`）；复制只带可加入的。
6. **草稿（R6）**：不发空的迭代、`module_ids: []` 照发、换项目清空、行的模块增减在草稿的队列里轮到时算、发布的拒绝落到字段。
7. **拆分与守卫（R7）**：T4 只搬不改；`project-address-route`（含 `issue-writes.ts`）、`.oxlintrc.json`、告警上限、knip、i18n、phase `M6/P7`。

## Task 1（T1）：P6b 回填（含 CI 的修复）与阶段

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P6b-module-list-review.md`：抬头与 §7。
- `specs/P6b-module-list.md`：抬头与 A8；`plans/P6b-module-list.md`：抬头。
- `M6-design.md`：抬头的状态句（P6 已完成，P7 进行中、拆为 P7a/P7b/P7c）；§14 的 P6b 行“已完成”并写上合入后 main CI 的失败与修复分支；从“P7、P8、closeout”一行分出 P7a 行“进行中”（链接预检 [re-72](../../../reviews/re-72-M6P7a预检.md)、本段的 spec 与 plan，同 P6b 行链接 re-70 的写法）、P7b 与 P7c 行“未开始”。
- `docs/v0/frontend-changes.md`：M6 一行中 P6b 的“待合入”改为已合入，P7 改为“P7a 进行中”。
- `tools/keywords.json`：`phase` 改为 `M6/P7`（A24）。
- 不改：`docs/v0/M8-open-release/handoffs/M6-closeout.md`——M6-M8-01 关于 43549053092 与 I4 的新段随修复分支合入，本 Task 只链接它。

- [ ] Step 0（前提）：P6b 的 CI 修复（分支 `codex/m6-p6b-ci`，提交 `0cfe405d` 的测试与 `9ae1517f` 的文档）的 noff 已在 main 上，且已并入本分支（主控在 T1 之前做）；以 `git merge-base --is-ancestor <修复的 noff> HEAD` 核对（收据 `t1-fix-merged`）。不在时停下告诉主控。
- [ ] Step 1：以 P6b 的归档为准回填 §7（P6b 的 worktree 已删除；归档在 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P6b-integration-<日期>/`，主控归档之前把修复分支的台账作为 `ci-fix/` 放进 P6b 的台账，一份归档里两者都有）：台账 `payload/worktree/.superpowers/sdd/P6b-module-list/progress.md` 的 Task 16 末尾与 `ci-fix/`，同一归档的 `cleanup.log` 与 `manifest.json`；作业号与时刻以公开的 jobs 接口（`https://api.github.com/repos/open-nerve/NerveProject/actions/runs/<id>/jobs`）核对，归档以 `archive.py verify` 核对（收据 `t1-p6b-archive-verify`）。§7 写下：
  - **最终 HEAD 的分支 CI**：`c0572192e831957dd63939b96ec007c3e1eb875b` 的 43545459485 attempt 1 成功（server (1) 132295623381、web 132295623598、server (0) 132295623771、e2e 132300181032；e2e 作业 02:32:37Z–02:55:46Z，23 分 09 秒，低于 P6b §5 的估算约 26.1 分钟与上界 27.3–27.6 分钟）。
  - **合入与 main CI**：noff `954a1838bff44e98cc44f15df524da27e42fb1d8`，双父 `06b72d4b`、`c0572192`，树与分支相同（以 `git rev-parse <sha>^{tree}` 核对）；main CI 43549053092 attempt 1：e2e 作业 132311928038（03:14:42Z–03:42:23Z，27 分 41 秒，故事本身约 25.4 分钟）只有 M4 的 I4 (page) 的表格故事失败（故事的时间预算），没有重跑；其余作业与结论照原件。
  - **修复**：分支 `codex/m6-p6b-ci`（`0cfe405d` 把那个故事拆成四个，本机 chromium 约多 5 秒；`9ae1517f` 的文档，含 M6-M8-01 的新段），它的分支 CI、noff 与合入后的 main CI（作业号、结论、e2e 作业时长）照 `ci-fix/` 与 jobs 接口写；链接 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的新段。
  - **归档**：文件数、字节、成员数、未知数、清单的 sha256 与清单的 `head`，以 `archive.py verify` 的收据为准。
  - **清理**：照 `cleanup.log`（两个分支与 worktree 的删除、main 与 origin/main 的位置）。
  - 修复的 main CI 没有全绿、归档不在、或原件与台账的数字不一致时，停下告诉主控，不写“已完成”。
  - 勾选 P6b spec 的 A8（注明 `make test` 的规则与上面的证据）；阶段记录、spec、plan 的抬头改为“已完成（日期，noff `954a1838`，修复的 noff …，main CI …）”，attempt 与结论照收据写。
- [ ] Step 2：设计的抬头与 §14；前端改动清单；`tools/keywords.json` 的 `phase` 改为 `M6/P7`，`node tools/keywords.mjs` 通过（收据 `t1-keywords-phase`：没有在 `M6/P7` 到期的例外）。台账记下本段 e2e 的基线（修复合入后的 main CI 的 e2e 作业时长）与最慢的数据点（27 分 41 秒），供 T16。
- [ ] Step 3：`python3 .superpowers/sdd/P7a-work-item-planning/doc-links.py`、`node tools/keywords.mjs`、`make lint-web`（收据 `t1-*`）；提交 `docs(M6/P7a): 回填P6b的集成结果与CI的修复，关键词阶段进入M6/P7`（正文写明 phase 的改动）。

## Task 2（T2）：写入单元的范围必给（A10、A11）

**Files：** `core/store/issue/issue-writes.ts`（`updateIssue` 的 `operationScope` 必给，`WritesRoot` 去掉 `"workspaceSlug"`，删去 `:260-261` 的退路与注释）；`research/web.md` §1.6 的 8 个测试文件（只加范围）；`core/store/issue/root.store.ts`（router 的 getter 只在仍有读者时留着）；`tools/keywords.json`（`issue-writes.ts` 加入 `project-address-route`，带样例）。

- [ ] Step 1：先记基线：8 个测试文件逐个聚焦跑（收据 `t2-baseline-scope`，台账记各自的通过数）与 `make test-web`（`t2-baseline-test-web`，记总数）；把参数改为必给后以 `tsc --noEmit` 列出缺范围的调用处（台账记文件与处数，应为 8 个文件 32 处；不一致时以 tsc 为准并写明）。
- [ ] Step 2（GREEN）：改签名与 `WritesRoot`，32 处只加范围参数（以 `git diff -U0` 过一遍，台账写明只有这一种改动）；以 grep 核对根上 `workspaceSlug`、`projectId` 等 router getter 的读者，没有读者的删去（类的成员 knip 不查）；`issue-writes.ts` 加入 `project-address-route`。变异：守卫——在写入单元里以 `this.root.rootStore.router.workspaceSlug` 补回退路（`t2-mut-router-fallback`，`node tools/keywords.mjs` 须失败）；行为——范围的项目改取工作项自己的 `project_id`（`t2-mut-scope-from-snapshot`，能编译）须让既有测试失败，没有失败的就先补一个钉住“确认写进给出的范围的列表”的测试（RED 由这个变异给出，台账写明）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（总数与基线同）、8 个文件逐个同数；`wc -l` 记根与写入单元的行数（`t2-line-counts`）；提交 `refactor(M6/P7a): 写入单元的范围必给，去掉 router 的退路，加入 project-address-route`。

## Task 3（T3）：模块集合的变化以增减在轮到时算（D3，re-72 F1）

**Files：**
- 改：`core/store/issue/issue-write-fields.ts`（增减作用于所持集合的纯函数）与 `issue-write-fields.test.ts`；`issue-writes.ts`（按变化写的命令，名字由实现定）与它的测试；`helpers/base-issues.store.ts`（`updateIssue` 收变化、经新命令）；`module/issue.store.ts`（移出改为 `{ remove: [本模块] }`，R19）；`core/hooks/use-issues-actions.tsx`、`core/hooks/use-planning-page-issue-actions.ts`、`issue-details/issue.store.ts`、`issue-detail/root.tsx`（`TIssueOperations` 的默认类型）、`issue-detail/use-issue-operations.ts`；行与表格的写的类型（今天是 `Partial<IssueDetail>`：`properties/all-properties.tsx:37`、`spreadsheet/columns/label-column.tsx:20` 与同目录各列，以 tsc 为准）。
- 不改：命令面板的 `mutation-commands.ts`、`description-versions.store.ts`、拖拽的 `dropIssue`（各自已给 `makeBody`）；`updateBody` 的白名单照旧放行 `module_ids`（`issue-write-fields.ts:35-36`），只给新命令自己算出的整集合用。

- [ ] Step 1：先跑回归边界“变化命令”一条所列的测试留基线（`t3-baseline-writes`，含 P6b 的移出与经动作、详情的修改）。
- [ ] Step 2（RED，`FakeNerve`、`fake-queue.ts`）：
  - 纯函数：所持 [A, G]（G 是已归档的、页面不认识的），`{add: [B]}` → [A, G, B]；`{remove: [A]}` → [G]；加已有的、去没有的都不变；次序保留；
  - 写入单元：同一工作项第一个变化在外时排第二个——所持 [A, G]，先 `{add: [B]}` 后 `{remove: [A]}`：请求体依次 [A, G, B] 与 [G, B]（第二个以第一个的回答为底）；第一个被拒时第二个是 [G]；变化带的其余字段照给出的发（`{name, modules}` → 请求体有 `name` 与整集合），变化上多出的、不是 `IssueUpdate` 字段的东西不进请求体（命令逐个点名写，不写 `...change`）；
  - 迭代：两次 `{cycle_id}` 各发给出的值，`null` 移出，不按所持算；
  - 经 `BaseIssuesStore.updateIssue`、详情的 `updateIssue` 与 `TIssueOperations.update` 同上；
  - 类型（类型测试，`@ts-expect-error` 各一行）：变化的类型声明 `module_ids?: never`；对象字面量 `{ module_ids: [A] }` 赋不进去；**一个类型为 `Partial<IssueDetail>` 的值**（今天行与表格传的就是它）赋不进去；
  - P6b 的移出经新命令：请求体与时机与基线同（既有测试不改）。
- [ ] Step 3（GREEN）：实现。变异：在调用时以所持算出整集合（`t3-mut-modules-at-call`）、只保留规划 store 认得的 id（`t3-mut-drop-unknown`，已归档的 G 丢掉）、加入以给出的替换所持（`t3-mut-add-replaces`）、第一个被拒后第二个仍以第一个的请求为底（`t3-mut-base-on-request`）、命令以 `...change` 写请求体（`t3-mut-spread-change`：多出的字段进了请求体，Step 2 的用例失败）；把变化的类型改回 `Partial<IssueDetail>` 这一变异编译不过，不计（类型测试本身以 `tsc` 的收据 `t3-type-guard` 记下）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线的测试同数）；提交 `feat(M6/P7a): 模块集合的变化以增减在写队列轮到时作用于所持的集合，列表与详情的修改经它`。

## Task 4（T4）：到上限的三个文件先拆（D7）

**Files：** `issue-modal/base.tsx`（新建、修改、发布三个命令与描述的读各出一件）、`issue-modal/form.tsx`（提交与页脚各出一件）、`issue-layouts/utils.tsx`（拖放的件、表格的列表与图标各出一件）（spec §2.9；新文件的名字由实现定）；读者与测试的导入；`tools/keywords.json`（被拆文件在 `project-address-route` 里时，新件同样加入）。

- [ ] Step 1：先跑 `issue-modal/` 下的全部测试与导入 `issue-layouts/utils` 的测试留基线（`t4-baseline-modal`、`t4-baseline-layout-utils`，台账记文件与逐个的数）。
- [ ] Step 2（GREEN）：只搬不改（以 `git diff -M` 核对搬移，台账写明），测试只改导入；拆后各约 400 行以内（`t4-line-counts`）；knip 无误（拆出件的导出各有产品读者）。变异（落在搬过去的代码上，既有测试须失败）：新建的请求体去掉 `parent_id`（`t4-mut-create-body`）、拖拽的请求体不设 `sort_order`（`t4-mut-drop-body`）、修改的请求体带上没改过的字段（`t4-mut-update-dirty`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（与基线同数）；提交 `refactor(M6/P7a): 弹窗的 base、form 与布局的 utils 按职责拆开，行为不变`。

## Task 5（T5）：选项的规则与读、迭代下拉框与侧栏的迭代一栏（A1、A3，re-72 F3）

**Files：**
- 新：选项与名字的规则（`core/lib/` 的纯模块）；选项的读（`core/hooks/`，打开时经 `useDirectoryRead`）；`core/components/dropdowns/cycle/{base,dropdown}.tsx`；详情与预览共用的规划属性一件（名字由实现定，本 Task 只有迭代一行、只在侧栏用）；各自的测试。
- 改：`issue-detail/sidebar.tsx`（截止日期之后加这一件，它是规则、读与下拉框的第一个产品读者）；i18n（读 `common.cycle`、`cycle.no_cycle`、`cycle.label`）；`tools/keywords.json`。
- 不在本 Task：名字的按需读、“已归档”的说明、预览的一栏与五个键的删除（T6）。

- [ ] Step 1（RED）：
  - 规则：选项是未归档且状态不是 `completed` 的迭代、nerve 的次序、“无迭代”在首；名字的规则认已结束、已归档的（store 有它时）；
  - 选项的读：打开时读一次项目的列表；读失败时列表底部说明并可重试；换了项目或权限身份之后旧的回答不写；
  - 下拉框：选项；选中给 id 并关上；“无迭代”给 `null`；选当前的值不调用；已结束的现值在按钮上（状态图标），不在列表里；`projectId` 在列表开着时变了，列表关上，再开读新项目；`disabled` 打不开；触发器以“字段：值”为名，给出行时名字另含那一行的工作项；
  - 侧栏：迭代一栏只在 `cycle_view` 打开时显示；不可编辑时只读；选中经 `issueOperations.update` 只发一个 `{cycle_id}` 的 `PATCH`；被拒时提示、显示 nerve 的值；连续两次选择发两个请求、各是所选的值。
- [ ] Step 2（GREEN）：实现。变异：已结束的作为选项（`t5-mut-completed-option`）、没有“无迭代”（`t5-mut-no-none-option`）、换项目时列表留着（`t5-mut-project-change-keeps-list`）、迟到的选项回答写入（`t5-mut-late-options`）、不看开关（`t5-mut-ignore-switch`）、不可编辑时仍可选（`t5-mut-editable-ignored`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 迭代的选项规则与读、迭代下拉框与详情侧栏的迭代一栏`。

## Task 6（T6）：名字的按需读、“已归档”的说明、预览的迭代一栏（R14，re-72 F3）

**Files：** 名字的按需读（`core/hooks/`，经 `useSessionSWR` 的 `PROJECT_CYCLES`、`PROJECT_ARCHIVED_CYCLES`，模块的两个键同一件里给出）；T5 的规划属性一件加名字的读与“已归档”的说明；`peek-overview/properties.tsx`（截止日期之后加这一件）；i18n（加“已归档”的说明；删 `issue.add.cycle.failed`、`issue.remove.cycle.{loading,failed}`、`issue.remove.module.{loading,failed}`，R18，en 与 zh-CN）；`.oxlintrc.json`（用 `useSessionSWR` 的新文件）；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 名字：store 叫不出名字时，调用者在项目里有角色才读，先未归档的列表（还没有时）后已归档的；同一项目的两处只发一个请求；没有角色不读；开关关闭时不显示也不读；读到之后已归档的迭代以名字与“已归档”的说明显示；
  - 预览：迭代一栏同侧栏（开关、只读、一个请求、被拒时说明）；
  - 删去的五个键没有读者（`tools/i18n-keys.mjs` 与 `check:sync` 通过）。
- [ ] Step 2（GREEN）：实现。变异：名字按行各读一次（`t6-mut-names-per-row`，以请求数失败）、没有角色也读（`t6-mut-names-without-role`）、不读已归档的列表（`t6-mut-no-archived-read`，名字缺）、预览不看开关。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 迭代与模块的名字按需读、“已归档”的说明与预览的迭代一栏`。

## Task 7（T7）：模块下拉框与详情、预览的模块一栏（A2、A3）

**Files：** 新 `core/components/dropdowns/module/{base,dropdown}.tsx` 与测试；T5 的规划属性一件加模块一行（Plane 的次序：模块在迭代之前）；i18n（读 `common.modules`、`module.no_module`；多于一个时的个数文案若要新键随本 Task 加入）；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 下拉框：选项是未归档的模块（状态不论）、nerve 的次序、已选的不移到前面；工作项上已归档的模块在选项之后列为已选项、标明已归档，点它给 `{remove}`，不能再选；点一个选项给 `{add}` 或 `{remove}` 一个 id，不给整集合；选了不关；没有模块时按钮写 `module.no_module`，多于一个时显示个数；触发器里没有按钮（以 DOM 核对没有嵌套的 `button`）；以行为名同 T5；
  - 详情与预览（`FakeNerve` 的整合）：工作项在 [A, G]（G 已归档），第一个 `PATCH` 扣住时点 B、再点 A：请求体依次 [A, G, B] 与 [G, B]；去掉 G 发 [A]；`module_view` 关闭时不显示；不可编辑时只读；被拒时提示、显示 nerve 的值。
- [ ] Step 2（GREEN）：实现。变异：下拉框给出整集合、详情照发（`t7-mut-emits-set`，第二个请求体丢掉 B）、已归档的作为可选的选项（`t7-mut-archived-as-option`）、已归档的已选项不列出（`t7-mut-archived-not-listed`，去不掉 G）、已选的移到前面（`t7-mut-selected-first`）、选了就关（`t7-mut-close-on-select`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 模块下拉框与详情、预览的模块一栏，模块的变化是增减`。

## Task 8（T8）：显示属性与列表、看板的行（A1、A2）

**Files：** `web/packages/types/src/view-props.ts`（`IIssueDisplayProperties` 加 `cycle`、`modules`）；`web/packages/constants/src/issue/common.ts`（`ISSUE_DISPLAY_PROPERTIES_KEYS`、`ISSUE_DISPLAY_PROPERTIES`，标题 `common.cycle`、`common.modules`）；`web/packages/utils/src/work-item/base.ts`（`getComputedDisplayProperties` 默认 `true`）；`core/store/issue/helpers/issue-filter-helper.store.ts`（`displayPropertyNames`）；`issue-layouts/filters/header/display-filters/display-properties.tsx`（项目的页上按路由的项目的开关给两项）；`issue-layouts/properties/` 新一件行的属性与 `all-properties.tsx` 渲染它的一处；契约变化的包测试（有则）；i18n（以行为名若要新键，R20）；`tools/keywords.json`。`SPREADSHEET_PROPERTY_DETAILS` 的映射类型是同态的，本 Task 加了两个键也不必给它条目（`columns/header-column.tsx:39` 在没有条目时给 `null`），表格的两列在 T9。

- [ ] Step 1（RED）：
  - 默认：`getComputedDisplayProperties({})` 的 `cycle`、`modules` 为 `true`；本地存的页（个人页、归档页）收下并保留这两个布尔值；
  - 项目页：在显示面板关掉 `cycle`，发出 `PATCH /api/v0/projects/{id}/preferences`（`display_properties.cycle` 为假），行上不再有迭代；面板在路由的项目只开了模块时只给“模块”一项、都关时两项都不给；个人页的面板两项都给；迭代页、模块页照常给两项；
  - 行：显示属性打开且行的项目开着开关时显示；`isReadOnly` 时只读；迭代页上把一行改到别的迭代，只发一个 `PATCH`，确认后行离开本页、本迭代的进度重读一次（既有的跟进）；行的模块变化是增减；个人页上两个项目的行，名字叫不出时每个项目各读一次；行的触发器以那一行为名。
- [ ] Step 2（GREEN）：先 `pnpm --filter @nerve/types build`、`@nerve/constants`、`@nerve/utils` 再实现；按回归边界改契约变化处。变异：默认为假（`t8-mut-default-false`）、面板不看开关（`t8-mut-panel-ignores-switch`）、行不看显示属性（`t8-mut-row-ignores-display`）、行不看行的项目的开关（`t8-mut-row-ignores-switch`）、本地存的页丢掉两个键（`t8-mut-local-drops-keys`）。若本 Task 超出一个实现者的量，显示面板的开关门移到 T9（台账写明），不加 Task。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、包的测试、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 显示属性的迭代与模块，列表与看板的行上的两个属性`。

## Task 9（T9）：表格的两列（检查点，re-72 F10）

**Files：**
- `web/packages/constants/src/issue/common.ts`：`SPREADSHEET_PROPERTY_LIST` 在标签之后加 `modules`、`cycle`；`SPREADSHEET_PROPERTY_DETAILS` 给两项标题与图标，两项的排序键（`ascendingOrderKey`、`descendingOrderKey` 与两个标题）在类型上可省（只这两项可省，写法由实现定；R21）。
- `issue-layouts/spreadsheet/spreadsheet-header-column.tsx` 与 `columns/header-column.tsx`：条目没有排序键时，表头只给标题与图标，不渲染 `CustomMenu`，`<th>` 不带 `aria-sort`（今天 `spreadsheet-header-column.tsx:33` 总给 `aria-sort`，`columns/header-column.tsx:39` 起是排序菜单）；`spreadsheet/spreadsheet-sort.ts` 随之处理没有排序键的条目。
- 新 `issue-layouts/spreadsheet/columns/` 两列；T4 拆出的表格的列表与图标加两项；测试；`tools/keywords.json`。

- [ ] Step 1（RED）：两列在显示属性打开、项目开着开关时出现，次序在标签之后（模块、迭代）；两列的表头只有标题与图标，没有排序菜单、没有 `aria-sort`，别的列照旧（`header-column.test.tsx` 断言不改）；格里改迭代发 `{cycle_id}`、改模块是增减；只读时不可改；已归档的值以名字与说明显示。
- [ ] Step 2（GREEN）：实现。变异：两列的表头给出排序菜单（`t9-mut-header-sorts`）、两列的 `<th>` 带 `aria-sort`（`t9-mut-aria-sort`）、两列的次序颠倒（`t9-mut-column-order`）、格里发整集合（`t9-mut-cell-sends-set`）。
- [ ] Step 3：先构建 `@nerve/constants`；`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 表格的迭代与模块两列，表头暂无排序`。
- [ ] Step 4（检查点）：`sh .superpowers/sdd/P7a-work-item-planning/gates.sh t9-gate` 串行七项（`make test` 照负责人的规则）；既有页面故事若因行上、表格里多出的两项失败，按回归边界只补契约并在台账写明；推送准确的 HEAD，核分支 CI，记录 e2e 作业的时长（校准点；超过 30 分钟先报告负责人，不改期限或作业结构）。

## Task 10（T10）：新建弹窗的字段与表单优先（D2，re-72 F2、F7）

**Files：** `issue-modal/types.ts`（`IssueFormValues`、`ISSUE_FORM_FIELDS` 加两个字段）；`web/packages/constants/src/issue/modal.ts`（默认 `cycle_id: null`、`module_ids: []`）；`issue-modal/components/default-properties.tsx` 与新的规划字段一件（两个下拉框，模块字段把增减作用于表单自己的值，只在表单的项目开着开关时）；T4 拆出的新建命令（请求体加 `cycle_id`、`module_ids`，为空时省略）与弹窗一级的初值（`base.tsx` 或新建命令一件：表单挂上之前并进默认值）；`core/hooks/planning-page-domains.ts`（本页的初值与“确认的工作项在本页”两列，名字由实现定）；`core/hooks/use-planning-page-issue-actions.ts`（给表单的照给出的新建）与 `use-issues-actions.tsx`（其余 store 类型的同名命令就是新建）；`tools/keywords.json`。
- 不在本 Task：调用者给的 `data` 的过滤与它的测试（T12，re-72 F8）。

- [ ] Step 1（RED）：
  - 初值在弹窗一级：可加入的迭代页上新建，即使把 `IssueDefaultProperties` 换成 `() => null`（同 `cycle-create.test.tsx:66-68` 的写法），请求体也带本迭代；表单的项目没开开关时请求体没有这个键；
  - 可加入的迭代页上新建，表单的迭代一栏是本迭代；改成别的迭代，只发一个 `POST`、带那个迭代（表单优先）；选“无迭代”，请求体没有 `cycle_id` 这个键；不可加入（已结束、已归档、开关关闭、实体属别的项目）时没有预填；
  - 可加入的模块页上新建，表单的模块一栏是本模块；去掉它，请求体不带它；再加一个，两个都带；
  - 弹窗带着项目打开时经 `PROJECT_CYCLES`、`PROJECT_MODULES` 读两份未归档的列表；答到之前两个字段显示加载、本页的初值留着（re-72 F7）；
  - 换项目：两个字段清空，请求体都不带；两个值为空或字段不显示时非草稿新建的请求体没有这两个键（M4、M5 的新建请求体照旧）；
  - 快速新建在可加入的规划页上照旧加入本页（注入）；全局新建、页头、分组头在规划页上只在可加入时带本页（P5b、P6b 的测试一行不改照旧通过）；
  - 确认的工作项在本页时本页的进度重读一次，不在本页（选了别的迭代）时不重读。
- [ ] Step 2（GREEN）：实现；按回归边界改契约变化处（有则）。变异（都落在弹窗一级并入初值、发出新建的那一处）：表单的新建仍经本页的注入（`t10-mut-page-overwrites-form`，选别的迭代时请求体是本迭代）、初值改由字段的组件去填（`t10-mut-prefill-in-field`：把字段换成 `() => null` 的用例失败）、预填不看可加入（`t10-mut-prefill-ignores-joinable`）、`cycle_id` 为空时发 `null`（`t10-mut-null-cycle-sent`）、换项目不清空（`t10-mut-reset-keeps-planning`）、快速新建不加入本页（`t10-mut-quick-add-no-join`）、不在本页时也重读进度。
- [ ] Step 3：先构建 `@nerve/constants`；`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 新建弹窗的迭代与模块字段，初值在弹窗一级、表单优先`。

## Task 11（T11）：修改发出差异，拒绝落到字段

**Files：** T4 拆出的修改命令（模块的差异经 T3 的命令）与提交件（以表单挂上时的初值算增减）；`useFormRefusal` 的字段与 `planning.cycle_completed` → `cycle_id`（spec §2.6，R17）；测试。

- [ ] Step 1（RED）：
  - 修改显示工作项真实的值（已归档的模块、已结束的迭代），在别的迭代的页上打开修改，迭代一栏是工作项的而不是本页的；
  - 只改名字，请求体没有模块与迭代（P6b 的用例照旧）；
  - 弹窗打开后它自己读工作项时读到多出的模块 D，表单显示打开时的 [B, G]，去掉 B 保存：请求体是 [G, D]（差异在轮到时作用于所持的集合）；加一个同理；改迭代带 `{cycle_id}`；
  - 422 `module_ids[1]` 落到模块一栏并取得焦点；422 `cycle_id` 落到迭代一栏；409 `planning.cycle_completed` 落到迭代一栏；409 `planning.archived` 照旧提示。
- [ ] Step 2（GREEN）：实现。变异：修改发出表单的整集合（`t11-mut-edit-whole-set`，D 丢掉）、以随读取变化的 `data` 作初值（`t11-mut-diff-from-data`）、修改预填本页（`t11-mut-prefill-on-edit`）、`planning.cycle_completed` 只提示（`t11-mut-cycle-completed-toast`）、`module_ids[i]` 不落到字段（`t11-mut-module-error-toast`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`；提交 `feat(M6/P7a): 修改发出迭代与模块的差异，拒绝落到字段`。

## Task 12（T12）：分组头的新建弹窗、复制与调用者给的初值（A13、A14，re-72 F4、F7、F8）

**Files：** `issue-layouts/command-host.tsx`；`list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx`（经命令宿主请求，不再自己渲染弹窗）；T10 的弹窗一级的初值（调用者给的 `data` 按选项过滤）；测试（照 `roots/row-dialog-lifetime.test.tsx` 与 `fake-planning-page.tsx` 的 `outlivesTheRow`）；`issue-modal/content-copy.test.tsx`；契约变化的 `roots/module-quick-actions.test.tsx:264`；`tools/keywords.json`。

命令宿主改三处（spec §2.7）：
- (a) 加一种“新建”的请求，带分组的 `data` 与 store 类型（`RequestDialog`，`command-host.tsx:27`）。今天宿主只在 `holds(dialog, dialog.data.project_id)` 时渲染对话（`:160-165`、`:196`），分组的 payload 多半没有 `project_id`：新建的对话由路由的项目持有，按项目分组时由分组的 `project_id` 持有，没有路由的项目的页（个人页）由工作区的权限身份持有（项目列表为空）；
- (b) 锁定项目（`isProjectSelectionDisabled`）由宿主按 `isPlanningPageStoreType` 给出，同今天的两个分组头（`list/headers/group-by-card.tsx:99`、`kanban/headers/group-by-card.tsx:63`）；
- (c) `DialogStore`（`:26`）包括分组头会给“+”的每一种 store 类型（以各 store 的 `viewFlags.enableIssueCreation` 为准：项目、迭代、模块、个人页）。

- [ ] Step 1（RED）：
  - 项目页按状态分组（payload `{state_id}`）：分组头的“+”打开新建弹窗，列表核验（布局换成加载）时弹窗留着，提交只发一个请求；看板同；按项目分组（个人页）时对话由分组的项目持有；规划页上照旧锁定项目、项目页上不锁；
  - 调用者给的初值（第一个真正的调用者是复制）：源在 [A, G]（G 已归档）、在已结束的迭代里：复制的表单在两份未归档的列表答到之前字段显示加载、值留着；答到之后带 A、不带 G 与那个迭代（不算改动）；模块页上再并本模块；迭代页上复制本页的行照旧加入本迭代（`cycle-quick-actions.test.tsx:175` 照旧）；
  - 描述带图片的复制经内容复制：草稿的 payload 带表单的迭代与模块，发布后工作项在其中（本页的迭代或模块）。
- [ ] Step 2（GREEN）：实现；按回归边界改 `module-quick-actions.test.tsx:264`（台账写明 R15）。变异：弹窗留在分组头里（`t12-mut-header-modal-in-group`）、宿主仍以 `data.project_id` 持有新建的对话（`t12-mut-hold-by-data-project`：按状态分组时打不开）、宿主不锁规划页的项目（`t12-mut-host-no-lock`）、`copyOf` 不带迭代与模块（`t12-mut-copy-no-planning`：P6b 记下的嵌套变异现在失败）、复制带上已归档的模块（`t12-mut-copy-keeps-archived`）、列表答到之前就过滤（`t12-mut-filter-before-lists`：值被丢掉）、内容复制丢掉两个值（`t12-mut-content-copy-drops-planning`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 分组头的新建弹窗移到命令宿主，复制带源的可加入的迭代与模块`。

## Task 13（T13）：草稿（D6，re-72 F6）

**Files：** `issue-modal/draft-form.ts`（`draftPayloadFields` 加两个）；T4 拆出的提交件（草稿换项目时清空两者）；T4 拆出的发布命令与 `handleMoveToProjects`（发布的拒绝落到字段）；`workspace-draft/draft-issue-properties.tsx`（两个下拉框，模块的增减经草稿 store 的 `input(held)`）；测试（`issue-modal/draft.test.tsx`、`workspace-draft/*.test.tsx`）；契约变化的新草稿请求体（有则）；`tools/keywords.json`。`core/services/issue/draft-input.ts` 不改（主控裁定）。

- [ ] Step 1（RED）：
  - 草稿的表单显示 payload 里的迭代与模块（含已结束、已归档的）；选“无迭代”保存，修改带 `cycle_id: null`、存下的 payload 没有 `cycle_id`（不是 `null`、不是 `""`）；从弹窗“存为草稿”新建的草稿没有 `cycle_id`，可以带 `module_ids: []`；
  - 草稿去掉全部模块：修改带 `module_ids: []`，存下的集合为空（省掉键会留下原来的集合）；
  - 草稿换项目：两者清空（`cycle_id` 去掉键、`module_ids: []`）；
  - 草稿的行：改迭代发草稿的 `PATCH`（payload 带 `cycle_id`）；第一个 `PATCH` 扣住时连点两个模块，第二个 payload 以第一个的回答为底，payload 的其余键照所持的保留；草稿的项目没开开关时不显示；两个项目的草稿各在打开时读自己项目的选项；
  - 在草稿的弹窗里发布被拒：422 `cycle_id` 落到迭代一栏、草稿留着；409 `planning.cycle_completed` 落到迭代一栏。
- [ ] Step 2（GREEN）：实现；按回归边界补新草稿请求体里的 `module_ids: []`（有则，台账逐条写明）。变异：保存发 `cycle_id: null` 进 payload（`t13-mut-null-cycle-draft`）、去掉全部模块时省掉键（`t13-mut-empty-modules-omitted`：原来的集合留着）、换项目不清空（`t13-mut-project-change-keeps`）、行的模块在调用时算（`t13-mut-row-set-at-call`）、发布的拒绝只提示（`t13-mut-publish-toast-only`）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7a): 草稿的迭代与模块：表单、行与发布`。

## Task 14（T14）：页面故事（G4、G11）

**Files：** `e2e/stories/planning/g4-issue-cycle-page.spec.ts`、`g11-issue-modules-page.spec.ts`（新）；世界与帮手若要共用（名字由实现定，例如 `e2e/fixtures/work-item-planning-pages.ts`）；已有的帮手从原处导入（spec §2.11）。

- [ ] Step 1：写两个故事（spec §2.11 的表）。每个故事先以期望错误的版本看到断言失败（收据），再写产品变异：G4——已结束的迭代作为选项（`t14-e2e-mut-completed-option`）、表单的新建仍经本页的注入（`t14-e2e-mut-form-injected`，请求体是本迭代）、名字不读已归档的列表（`t14-e2e-mut-no-archived-names`）；G11——下拉框发整集合（`t14-e2e-mut-dropdown-set`，数据库里没有 Beta）、丢掉已归档的模块（`t14-e2e-mut-drop-archived`）、修改发整集合（`t14-e2e-mut-edit-whole-set`，数据库里没有 Delta）。变异的脚本 `t14-e2e.sh` 依次跑 `react-router typegen`、web 与 e2e 的 `tsc --noEmit`、`make build build-s3-fixture` 与 chromium 的故事，只计能编译、能构建的。有两道守卫时以嵌套变异证明并写明单去一道为何等价。
- [ ] Step 2：`make build build-s3-fixture` 后 chromium 聚焦跑通；以 `--config playwright.m6-pages.config.ts` 在三种引擎上跑一次（收据），新故事另 `--repeat-each=3`；从三种引擎那次运行的报告记下两个新文件在各引擎的时长（台账，供 T16）；两个文件本机 chromium 合计超过 25 秒时先压缩故事再继续（台账写明，D9）；`make lint-web`；提交 `test(M6/P7a): 工作项的迭代与模块的页面故事（G4、G11）`（两个文件可各一个提交）。

## Task 15（T15）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`：新行（spec R2–R6、R11–R18 中改变行为的）：下拉框的写法（增减在轮到时算、一个请求、被拒时说明，plane-contract §8 第 2、3 条）；迭代的选项与已结束、已归档的现值；模块的选项、已归档的已选项只能去掉、没有嵌套的移除按钮、已选的不移到前面；属性随项目的开关；新建弹窗的表单优先与本页的初值（扩充“迭代页上的新建”“模块页上的新建”两行）；修改发出差异（扩充“模块页行的修改与复制”一行）；复制带源的可加入的值；分组头的新建弹窗在命令宿主；草稿的界面（不发空的迭代、`module_ids: []` 照发、换项目清空，扩充“建立与发布草稿时的归属”一行）；跨项目的页按项目读目录；表格两列的表头暂无排序（P7b）。Plane 的行号对照主目录的只读 `plane/` 逐一核过；已登记的行引用不重复；P6b 写着“在 P7”的行（如“模块页与迭代页的空态”）只改到本段做了的部分。
- `docs/v0/frontend-changes.md`：M6 一行（P7a）；“迭代和模块的归属”一行（下拉框的写法、新建弹窗与草稿已完成，“添加已有”在 P7b）；“工作项根 store 的写入单元”一行（范围必给、按变化写的命令）；新行：迭代与模块的下拉框、工作项上的迭代与模块（详情、预览、行、表格、显示属性）、新建与修改的弹窗与草稿、分组头的弹窗宿主、三个文件的拆分。
- 输入交接的“处理结果（M6/P7a）”：[M2-closeout](../handoffs/M2-closeout.md) §4 的三点（第一点：列表开着时换项目关上、再开读新项目；第二、三点照 P5a 的结论）与 §5(a) 两个下拉框（列表在按钮旁、搜索框取得焦点、`Combobox` 一份打开的状态，G4、G11 的页面故事在三种引擎核对）；[M3-closeout](../handoffs/M3-closeout.md) §3 的“弹窗里的列表”中两个下拉框（工作项弹窗里 `modal={false}`，G4、G11 核对）与“每一行的控件”中行与表格的下拉框；其余各项注明在 P7b、P7c；均保持 open（设计 §11）。M8 交接 [M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)：本段之后根与写入单元的行数；[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)：T9 检查点的分支 CI 的 e2e 作业时长。
- 设计：§8 的“工作项一侧”写明本段做了的（下拉框、属性、显示属性、弹窗与草稿、复制，写入单元的变化命令）与留给 P7b、P7c 的；§12 的 P7 一行拆为 P7a、P7b、P7c 三行（各写交付边界与验证，引本 spec 的 §1、R1）；§14 的 P7a 行；spec §5 的补全（含 P7b、P7c 的 e2e 余量与作业结构须负责人决定，re-72 F5）。

- [ ] Step 1：写差异与回执。
- [ ] Step 2：`doc-links.py`、`node tools/keywords.mjs`；提交 `docs(M6/P7a): 交接回执、差异清单、前端改动清单与设计中 P7 的拆分`。

## Task 16（T16）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t16-gate`，留收据；`make test` 照负责人的规则），另以三种引擎的页面配置跑一次（收据）；`wc -l` 记根、写入单元与 T4 拆出的各件的行数（M6-M8-08）。推送之前照 spec §2.11 算出三个数并写进台账：
  - **估算**：基线（T1 记下的修复合入后的 main CI 的 e2e 作业时长）加上 T14 记下的新文件 chromium 时长 × 1.67–2.25，两个 worker 摊开；
  - **上界**：已测到的最慢的数据点（27 分 41 秒、T9 的检查点与 T1 记下的数中最大的；那次运行没有 I4 的拆分时再加它约 5 秒本机 chromium 的份）加上新文件 chromium 时长 × 3.6–4.1；写 spec 时约 29.2–29.4 分钟；
  - **校准点**：T9 检查点的分支 CI 实测的 e2e 作业时长与基线之比，按它换算估算在那种 runner 上的数；慢的 runner（×1.31，M6-M8-01）另算一次。
  - 估算、上界或任一次实测（T9 的检查点、推送后的分支 CI）超过 30 分钟时停下报告负责人、不推送；改 CI 的期限或作业结构（例如分片）须负责人同意（D9）。否则推送准确的 HEAD，核分支 CI，记录 e2e 作业的实际时长（超过 30 分钟同样先报告）。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账或在修复轮做（同 P6b 的修复轮 F）；修复轮另经一次限定复审；
  - 写阶段记录 `reviews/P7a-work-item-planning-review.md`（§7 留给 P7b 的 T1 回填）；
  - 勾选 spec §4 的 A1–A11；A12 待合入后由 P7b 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档并正常清理。
