# M6 / P7b 工作项一侧（二）——列表词表、目录与“添加已有”实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：spec 与 plan 已写（2026-11-14），待预检。基线 main `f42f9d6457ccc9540a687019f3530e253fe0f7b5`（P7a 的 noff，双父 `7868a6b4`、`5d6dab04`；P7a 的最终分支 CI 43600392398 全绿，e2e 作业 23 分 48 秒；合入后的 main CI 43603772626 写计划时进行中，T1 回填）。P7 在实施之前拆为 P7a/P7b/P7c（[P7a spec](../specs/P7a-work-item-planning.md) R1），本计划只管 P7b；主控在写 spec 之前的裁定 E1–E15 是 spec 的 R1–R15。

**Goal：**
- 先拆 `issue-list-binding.ts`；列表的目录从 P7a 的规划 store 叫出迭代、模块的名字（含已归档的），按名字排序不再失败；
- 列表的词表：按迭代、模块筛选（含“无”）、分组与子分组、按名字排序，偏好的筛选按页收窄，迭代页、模块页的显示选项由项目页的推出；
- 界面：分组的列与泳道（已归档的组、“无”在最后、拒收的列）、开关关闭时的选项、分组里的新建、筛选、拖拽、表格两列表头的排序；
- “添加已有”：空态、分组头、日历三个入口，弹窗在命令宿主，逐个 `PATCH` 经 `changeIssue`、至多 2 个在途、一批一次进度重读；
- G15 的工作项一侧与 G4、G11 的“添加已有”的页面故事在三种引擎，CI 的 e2e 时长在报告线以内（或越过之前停下报告负责人）。

**Architecture：**
- 列表引擎的目录读规划 store 的同一组读法（P7a §2.3），以列表自己的读为门（spec R17）；界面上的读者经 P7a 的 `usePlanningNames`、`usePlanningListReads`，以开关为门（spec §2.12）。
- 写的一侧只在写入单元（P6b R1）：拖拽与“添加已有”都经 P7a 的 `IssueChange` 与 `changeBody`，模块集合在写队列轮到时以所持的算出（spec §2.11）。
- 页面读路由参数；按域的差别经 `planning-page-domains.ts`、`planning-page-layouts.ts` 的表，不写 `if (domain === …)`；须活过核验的对话归 `IssueListCommandHost`（P7a R16）。

**Tech Stack：** 既有 React/MobX/SWR/openapi-fetch/vitest/Playwright，不加依赖。

**Spec：** [P7b-list-vocabulary](../specs/P7b-list-vocabulary.md)（§2 的合同与 §3 的 R1–R22 是本计划的约束；R1–R15 是主控的 E1–E15）；设计 [M6](../M6-design.md) §3.3、§3.6、§7、§8、§12；网页规则 [v0 §7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。模板：P7a 的 [spec](../specs/P7a-work-item-planning.md)、[plan](P7a-work-item-planning.md) 与[阶段记录](../reviews/P7a-work-item-planning-review.md) §5 的教训；代码的模板：标签的列（`issue-layouts/group-columns.tsx` 的 `labelColumns`）、标签的筛选工厂（`web/packages/utils/src/work-item-filters/configs/filters/label.ts`）与 Plane 的 `cycle.ts`、`module.ts`、命令宿主的“新建”请求（`issue-layouts/command-host.tsx`，P7a T12）、规划页的表（`core/hooks/planning-page-domains.ts`）、P7a 的名字与列表的读（`core/hooks/use-planning-names.ts`）与它们的测试；e2e 的 `e2e/fixtures/{planning-list-pages,planning-lists,settings-pages,cycle-pages,dialogs}.ts`、`fixtures/assert/issue-planning.ts`。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p7b`，分支 `codex/m6-p7b`，基线 main `f42f9d64`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/refactor/test/docs(M6/P7b)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送由主控用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 仍为 `M6/P7`（三段共用，T1 核对）；不延长任何 `until`。本段不改 `server/`、`api/`（E14）。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。

**代码规则**
- 手写文件约 400 行内（E15）：`issue-list-binding.ts` 先在 T2 拆开再加代码；`use-work-item-filters-config.tsx`（353）的两个新配置另出一件（T7）；工作项根 store（320）与写入单元（302）不加（M6-M8-08，T16 以 `wc -l` 核对）。新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写或拆出的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（`IssueChange` 由 `IssueUpdate` 推出，P7a）；service 不转换数据；SWR 只经 `useSessionSWR`；新代码不读 router store（页面读路由参数）。
- 删掉不用的代码；不加开关、桩或兼容别名（`displayPropertyOn` 由新的谓词取代时两个读者改调它，不留别名；T9 删去死分支）。Plane 的旧写链（`addIssueToCycle`、`addIssuesToModule`、`changeModulesInIssue`）与乐观的分布更新不恢复。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json` 的上限（现为 44）；清掉告警的提交同时下调，新文件不得新增告警。
- **knip**：新文件在第一个产品读者出现的 Task 里建立（共用的“下一读”纯函数在 T3 与目录同建；有界并发的小函数与“添加已有”的动作在 T10 与空态的入口同建；筛选的两个工厂与配置的 hook 在 T7 同建）；只给测试读的导出不算读者（本阶段更严的规则，逐 Task 的审查执行）；删掉某件最后一个外部读者的 Task 同时删去它或收回导出；类的成员与对象的成员 knip 不查，审查核对。
- **新文案**：en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀）；本段读、加的键见 spec §2.13、R19。
- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例），跑 `node tools/keywords.mjs`；拆开的文件的新件同样加入，被取代的路径留在表里阻止复活。
- **`.oxlintrc.json`**：用 `useSessionSWR` 的新文件随它的 Task 加入 swr 限制的文件表。只收紧。
- **共用的假件**：组件测试要规划 store 的数据时用 `core/store/planning/fake-*.ts` 与 `fake-projects.ts`；规划页的帮手从 `issue-layouts/roots/fake-planning-page.tsx`（含 `outlivesTheRow`）、弹窗的替身从 `issue-modal/fake-dialog.tsx` 导入，不复制；`fake-planning-page.tsx` 的模块级 `opened`（A21，属 P7c）不在关闭时清空，用它的测试都先打开页。

**构建与类型**
- 改了 `@nerve/types`、`@nerve/constants`、`@nerve/utils`、`@nerve/i18n` 之后，先构建该包（`pnpm --filter <pkg> build`）再跑应用的类型检查；改了路由文件之后先跑 `react-router typegen`（`make knip` 已含）。
- 变异只计能编译的；类型检查不过的变异不算失败（以类型测试与 `tsc` 的收据记下）；e2e 的变异另要能构建。

**测试纪律**
- 每个 Task 先写会失败的测试（vitest 的 store/组件测试，或 e2e），看到它因所测性质失败（不是编译失败），再实现到通过；纯搬移（T2）与只改类型或调用写法的部分先以既有测试的收据为基线，之后同数。
- 下列性质每个至少一个能让测试失败的变异（名字见各 Task）：目录只在按名字排序时读、一个项目各至多一次、仍叫不出时大声失败、开关关闭时照读；本页的键不进本页的偏好；列的次序与已归档的组、“无”最后、泳道并入回答的组；组头与泳道头的名字经 `usePlanningNames`；拒收的列不发请求且不给新建；本页的词不在本页的选项；开关关闭时不给未选的、已选的留着；快速新建带组的值；筛选含“无”、已持的已归档值有名字；拖拽的请求体在轮到时算、拖到“无”只移出来源、已归档的留着；表头两向的名字排序；“添加已有”的弹窗活过空态的卸下与核验、排除本页的、显示当前的迭代、至多 2 个在途、一批一次进度重读、失败的留着、分组头只加入本页；日历在规划页一个请求、项目页不变。
- 每个 Task 跑：`make lint-web`、`make test-web`（或 `cd web/apps/web && pnpm exec vitest run <files>` 聚焦），涉及包的 `pnpm --filter <pkg> test`；改了导出、删了文件或改了路由时跑 `make knip`；改了 `tools/keywords.json` 时跑 `node tools/keywords.mjs`。
- 单测的 SWR 缓存：同一文件里读同一份键的组件测试各给自己的 `SWRConfig` 缓存（P7a F1 的教训，`issue-modal/fake-form.tsx` 的写法）。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P7a 的网页单测与 e2e。
- **拆分（T2）**：执行或导入 `IssueListBinding`、`prepareCatalog` 的测试（以 grep `issue-list-binding|IssueListBinding|prepareCatalog` 为准，至少 `helpers/base-issues.store.test.ts`、`issue-writes.test.ts`、`issue.store.test.ts`、`issue-changes.test.ts`、`issue-drag.test.ts`、`permission-reads.test.ts`、`cycle/priority-issues.store.test.ts`）只改导入路径，断言不改，基线逐文件同名同数。
- **目录（T3）**：`planningDirectoryIsUnavailableUntilP7`（`helpers/base-issues.store.test.ts:453-474`）按本段的合同改写（契约变化，台账写明）；`workItemInNoCycleOrModuleNeedsNoDirectory`（`:476`）、`hooks/use-planning-names.test.tsx`、`lib/issue-order.test.ts` 一行不改。
- **按页收窄（T4）**：偏好的测试（`helpers/page-preferences-filter.store.test.ts`、`cycle/page-preferences.test.ts`、`module/page-preferences.test.ts`、`project/filter.store.test.ts`）只改构造处（给出每页的转换），断言不改，基线同数；`lib/issue-filters.test.ts` 既有的断言不改。
- **分组（T5）**：`group-consumers.test.tsx`、`kanban/swimlanes.test.tsx`、`list/view.test.tsx` 的既有断言不改（分组的词与列只加不改）。
- **快速新建与常量（T6）**：快速新建的既有测试（以 grep `quick add|QuickAdd|prePopulate` 于 `issue-layouts/` 为准，至少 `quick-add/root.*.test.tsx`、`roots/group-create.test.tsx`、`roots/cycle-create.test.tsx`、`roots/module-create.test.tsx`）一行不改，基线同数；两页页头的“不按本页分组”（`[projectId]/cycles/(detail)/header.test.tsx:273`、`modules/(detail)/header.test.tsx:265`）一行不改；`display-properties.test.tsx` 与表格的列的测试只随谓词改名改导入。
- **拖拽（T8）**：`store/issue/issue-drag.test.ts`、`kanban/drop.test.tsx`、`hooks/use-group-dragndrop.test.tsx` 的断言不改（拖拽函数改给 `IssueChange` 后，标签、负责人、状态、优先级的请求体与今天相同），基线同数。
- **表头（T9）**：`spreadsheet/planning-columns.test.tsx:130` 的 “each header its title and its icon with no sort”（`:147-151`）按契约改为两列的排序（spec R9，台账写明）；`spreadsheet/columns/header-column.test.tsx`、`spreadsheet/view.test.tsx` 不改。
- **“添加已有”（T10–T12）**：`core/modals/existing-issues-list-modal.test.tsx` 不改（新的 prop 可省）；`roots/row-dialog-lifetime.test.tsx`、`roots/group-create.test.tsx` 不改；`calendar/quick-add-issue-actions.test.tsx` 若直接在日子的件里找弹窗，按契约改为在命令宿主之下渲染（台账逐条写明），搜索参数与请求体的断言不改。M4 的 I4（`e2e/stories/issue/i4-list-order-calendar.spec.ts:497-530`）一行不改，是项目页日历的回归保护。
- e2e：写 plan 时没有找到会受影响的断言（既有故事按名字点控件，分组与筛选的选项只增）；T8 检查点若有页面故事因新的选项、项目页的新词失败，只按契约补上并在台账写明。
- 其他断言不改。

**检查点与证据**
- 检查点：T8 结束时（目录、词表、按页收窄、分组、常量、筛选、拖拽完成）串行完整七项，主控推送并核分支 CI，记录 e2e 作业的实际时长（期限 35 分钟；超过 30 分钟先报告负责人，不改期限或作业结构）；这时还没有新的页面故事，这个数是 T16 的校准点（spec §2.14）。T13、T14 各记下新故事文件在本机三个引擎的时长；T16 推送之前照 spec §2.14 算估算、上界与校准点。T16 收尾再串行一次。`make test` 照负责人 2026-11-11 的规则（本段不改 Go；因 M6-M8-05 的签名失败时记录，以分支 CI 的服务端两路为准）。
- 证据放在 worktree 的 `.superpowers/sdd/P7b-list-vocabulary/`：台账 `progress.md`，收据经 `run-check.py`（`python3 .superpowers/sdd/P7b-list-vocabulary/run-check.py <label> <命令…>`，在 worktree 的根跑），变异经 `mutate.py`（`python3 .superpowers/sdd/P7b-list-vocabulary/mutate.py <label> <文件> <旧> <新> [<旧> <新> …] -- <命令…>`，每个旧串在文件里恰好一次，跑完还原并核对 sha256），七项门禁经 `gates.sh`（`sh .superpowers/sdd/P7b-list-vocabulary/gates.sh <前缀>`），文档链接经 `doc-links.py`，归档经 `archive.py`，调研笔记在 `research/`，主控的裁定在 `controller-rulings.md`。实现代理先读同目录的 `implementer-brief.md`（主控在实施前放入）。
- 台账的行：`Task N: Step k RED/GREEN — <label> exit <code>（…）`、`Task N: 基线 — <label> exit <code>（文件与数）`、`Task N: 变异 — <label>（…）→ 失败（哪个测试），restored=yes`、`Task N: Ruling: <决定> — <理由> — 代价：<代价>`、`Task N: complete (commits <first>..<last>, tests: <summary>)`。
- e2e 的写法照 spec §2.14 的“写法”一段。

## Review Focus

1. **目录（R17）**：查找取自规划 store、`IssueCatalog` 收窄；只在按名字排序时读、一个项目各至多一次、与 `usePlanningNames` 同一条纯函数；仍叫不出时大声失败；以列表自己的读为门；根不加 getter。
2. **按页收窄（R18）**：本页的键在类型上赋不进本页的偏好（属性写法的 `update`）；迭代页、模块页的修改不发本页的键。
3. **分组（R4、R19）**：词与列同一个 Task；次序、已归档的组的名字与说明、“无”最后、泳道并入回答的组；名字经 `usePlanningNames`；拒收一个字段、三处同读；文案不加 `common.module`。
4. **常量与开关（R8、R12）**：推出的两页常量不含本页的词（页头的测试真能失败）；一张“词 → 域”的表、一个谓词；开关关闭时未选的不给、已选的留着；快速新建带组的值、一个共用函数、既有测试同数。
5. **筛选（R7）**：工厂与配置的 hook、“无”、已持的已归档值有名字、开关门与已应用的条件。
6. **拖拽（R3）**：拖拽函数给 `IssueChange`；请求体在轮到时算；拖到“无”只移出来源；拒收的列不发请求；既有拖拽测试不改。
7. **表头（R9）**：两向的名字排序、存下的名字排序照原样；死分支删干净。
8. **“添加已有”（R2、R5、R6、R11、R20、R21）**：三个入口都经命令宿主、活过空态卸下与核验；排除本页的、显示当前的迭代；至多 2 个在途、一批一次进度重读、逐项结果；分组头只加入本页、以组为名；日历在规划页一个请求、项目页不变（I4）。
9. **拆分与守卫（R15、R16）**：T2 只搬不改；`project-address-route`、`.oxlintrc.json`、告警上限、knip、i18n、phase `M6/P7`；行数。

## Task 1（T1）：P7a 回填与 e2e 余量的重算

**Files：**
- `docs/v0/M6-cycles-modules/reviews/P7a-work-item-planning-review.md`：抬头与 §7。
- `specs/P7a-work-item-planning.md`：抬头与 A12；`plans/P7a-work-item-planning.md`：抬头。
- `M6-design.md`：抬头的状态句（P7a 已完成，P7b 进行中、P7c 未开始）；§14 的 P7a 行“已完成”（照 P6b 行的写法写分支 CI、合入、main CI 与归档），P7b 行“进行中”（链接本段的 spec 与 plan；预检的链接在预检记录存在之后由主控加）。
- `docs/v0/frontend-changes.md`：M6 一行中 P7a 的“待合入”改为已合入，P7b 改为“进行中”。
- `docs/v0/M8-open-release/handoffs/M6-closeout.md`：[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 加 P7a 的最终分支 CI 与 main CI 的 e2e 作业时长（与 P7a 推送前的估算、上界对照）。
- 不改：`tools/keywords.json`（phase 已是 `M6/P7`）。

- [ ] Step 0（前提）：P7a 的归档在主目录的 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P7a-integration-<日期>/`（主控在 main CI 43603772626 全绿之后建立）；以 `archive.py verify` 核对（收据 `t1-p7a-archive-verify`）；main CI 43603772626 的结论以公开的 jobs 接口（`https://api.github.com/repos/open-nerve/NerveProject/actions/runs/43603772626/jobs`）核对（收据 `t1-main-ci-jobs`）。归档不在、main CI 不是全绿、或原件与台账的数字不一致时，停下告诉主控，不写“已完成”。本 Task 只改文档，归档未到时可先做 T2、T3（P7a T2 先于 T1 的先例，次序不影响守卫：phase 不变）。
- [ ] Step 1：以归档为准回填 P7a 阶段记录 §7：
  - **最终 HEAD 的分支 CI**：`5d6dab04` 的 43600392398（以 jobs 接口的 `head_sha` 核对是这个提交；attempt、四个作业号与结论；e2e 作业的起止时刻与时长 23 分 48 秒，与 P7a §5 的估算约 20 分 25–27 秒、上界约 28 分 22–28 秒对照，写明高于估算、低于上界与比值）。
  - **合入与 main CI**：noff `f42f9d6457ccc9540a687019f3530e253fe0f7b5`，双父 `7868a6b4`、`5d6dab04`，树与分支相同（`git rev-parse f42f9d64^{tree} 5d6dab04^{tree}`，收据 `t1-merge-tree`）；main CI 43603772626（attempt、作业号、结论、e2e 作业时长）。
  - **归档**：文件数、字节、成员数、未知数、清单的 sha256 与清单的 `head`，以 `t1-p7a-archive-verify` 为准。
  - **清理**：照归档的 `cleanup.log`。
  - 勾选 P7a spec 的 A12（注明 `make test` 的规则与上面的证据）；阶段记录、spec、plan 的抬头改为“已完成（日期，noff `f42f9d64`，main CI 43603772626）”，attempt 与结论照收据写。
- [ ] Step 2：设计的抬头与 §14、前端改动清单、M6-M8-01 的数据点；`node tools/keywords.mjs` 通过（收据 `t1-keywords`：phase `M6/P7`，没有到期的例外）。
- [ ] Step 3（E13）：照 spec §2.14 重算余量并写进台账：S（1661 + (5 + 6.5) × f 与 1428 秒、main CI 43603772626 的 e2e 作业时长中的最大者）、P7b+P7c 的余量 X ≤ (1800 − S) / f（f = 3.6、4.1 两端）、X = 14 秒时的上界、以 43603772626 为基线的估算。余量小于 22 秒（P7b 14 + P7c 8）时停下报告主控（负责人的决定），不先写故事。
- [ ] Step 4：`python3 .superpowers/sdd/P7b-list-vocabulary/doc-links.py`、`node tools/keywords.mjs`、`make lint-web`（收据 `t1-*`）；提交 `docs(M6/P7b): 回填P7a的集成结果，重算P7b与P7c的e2e余量`（正文写明余量的数）。

## Task 2（T2）：先拆 `issue-list-binding.ts`（R16）

**Files：** `core/store/issue/helpers/issue-list-binding.ts`（487 行）拆为页的读与范围的核对、目录（`context.catalog` 的目录部分与 `prepareCatalog`）、绑定（生命周期、取数、确认与核验）三件（spec §2.1；新文件的名字由实现定，例如 `issue-list-read.ts`、`issue-list-catalog.ts`）；读者与测试的导入；`tools/keywords.json`（三件加入 `project-address-route`，带样例）。

- [ ] Step 1：先跑回归边界“拆分”一条所列的测试留基线（`t2-baseline-binding`，台账记文件与逐个的数）与 `make test-web`（`t2-baseline-test-web`，记总数）。
- [ ] Step 2（GREEN）：只搬不改（以 `git diff -M --color-moved` 核对搬移，台账写明），绑定对外的成员与写入单元的两处调用不变，测试只改导入；拆后各约 400 行以内（`t2-line-counts`）；knip 无误（拆出件的导出各有产品读者）。变异（落在搬过去的代码上，既有测试须失败）：去掉页与范围不符的核对（`t2-mut-scope-check`）、`prepareCatalog` 不读状态（`t2-mut-catalog-states`）、目录之后不再核对读的凭据（`t2-mut-read-current`）。有存活的就先补一个钉住它的测试（RED 由这个变异给出，台账写明；P7a T2、T4 的先例）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`（与基线同数）、`node tools/keywords.mjs`；提交 `refactor(M6/P7b): 列表的绑定按职责拆为读、目录与绑定，行为不变`。

## Task 3（T3）：目录（A7，R17）

**Files：**
- 改：T2 拆出的目录一件（两项取自规划 store 的查找；`prepareCatalog` 按名字排序时读两份列表）；`core/lib/issue-order.ts`（`IssueCatalog.cycles`、`modules` 收窄为 `Pick<ReadonlyMap<…>, "get" | "has">`）；`core/lib/planning-directory.ts`（“下一读”的纯函数：叫不出时先未归档的、仍叫不出再已归档的，名字由实现定）与 `planning-directory.test.ts`；`core/hooks/use-planning-names.ts`（改调这个纯函数，行为不变）。
- 改写：`helpers/base-issues.store.test.ts` 的 `planningDirectoryIsUnavailableUntilP7`（契约变化）。
- 不改：`core/store/issue/root.store.ts`（不加 getter）、写入单元。

- [ ] Step 1：先跑 `hooks/use-planning-names.test.tsx`、`lib/issue-order.test.ts`、`lib/paged-groups.test.ts`、`helpers/base-issues.store.test.ts` 留基线（`t3-baseline`）。
- [ ] Step 2（RED，`FakeNerve`）：
  - 纯函数：叫不出、store 没有未归档的列表 → 读它；有未归档的、仍叫不出、没有已归档的 → 读已归档的；两份都有 → 不读；都叫得出 → 不读；
  - 项目列表按 `cycle_name` 排序，行在未归档的 C1 与已归档的 C9：store 什么都没有时先发 `GET /api/v0/projects/A/cycles`、再发 `…?archived=true`，各一次，列表照名字的次序；store 已有未归档的列表时只读已归档的；行都在未归档的迭代里时不读已归档的；`-module_name` 同（模块的两份列表）；
  - 两份列表都读到、仍有叫不出的（其间被删除）→ “Required issue directory is unavailable”；
  - 不按名字排序的列表不发任何规划的读；
  - 项目的 `cycle_view` 关闭时按 `cycle_name` 排序的列表照读、照名字排（R17）；
  - 个人页（成员列表）按 `module_name` 排序、行来自两个项目：每个项目各读自己的；
  - 写的确认里出现新的叫不出的迭代时读（经 `issue-writes.ts` 的两处确认）；
  - 类型：`new Map()` 照旧赋得进收窄后的两项（既有字面量不改，以 `tsc` 收据 `t3-type-catalog` 记下）。
- [ ] Step 3（GREEN）：实现。变异：只读未归档的列表（`t3-mut-no-archive-read`，已归档的行叫不出而失败）、每次都读（`t3-mut-read-always`，以请求数失败）、以开关为门（`t3-mut-switch-gated`，开关关闭的用例失败）、叫不出时不失败（`t3-mut-silent-unnamed`）、hook 不调纯函数而自先读已归档的（`t3-mut-hook-own-rule`，hook 的测试以请求的次序失败）、查找不认项目（`t3-mut-lookup-any-project`：改为读 `details`/行而不经 `getCycleById`，以“项目 store 不再给的项目”用例失败）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线同数，改写的 R9 测试台账写明）；`wc -l` 记根与目录一件（`t3-line-counts`）；提交 `feat(M6/P7b): 列表的目录从规划store叫出迭代与模块的名字，按名字排序不再失败`。

## Task 4（T4）：筛选的词与偏好的按页收窄（A6、A9，R7、R18、R22）

**Files：** `core/lib/issue-filters.ts`（`fields` 加 `cycle_id`、`module_id`，`validValue` 收 `none`，`memberListQuery` 的分组排除加两个；每页的转换：迭代页去掉 `cycle_id`、模块页去掉 `module_id`）；`web/packages/types/src/view-props.ts`（`WORK_ITEM_FILTER_PROPERTY_KEYS` 加两个键）；`core/store/issue/helpers/page-preferences-filter.store.ts`（以筛选的类型为参数，service 的 `update` 写成属性，每页一个转换）；`core/store/issue/root.store.ts` 与 `project/filter.store.ts`（构造处给出转换，行数不增或只增一两行）；三个 service 的类型（如须）；测试与类型测试；`tools/keywords.json`（改写的件已在规则里的照旧）。

- [ ] Step 1：先跑回归边界“按页收窄”一条所列的测试与 `lib/issue-filters.test.ts` 留基线（`t4-baseline-preferences`）。
- [ ] Step 2（RED）：
  - 词：`toProjectFilters` 收下 `cycle_id__in: "none,<id>"` 与 `module_id__in`（“None”写成 `none`）；`toFilterExpression` 照原样写回；`cleanFilters` 丢掉不是 id、不是 `none` 的值；`memberListQuery` 不带两词的筛选与分组（`group_by` 为 `cycle_id`、`module_id` 时不发）；
  - 按页收窄（先只加词，看到它 RED）：迭代页的筛选修改，表达式里有 `cycle_id__in` 与 `module_id__in`，发出的 `PATCH /api/v0/me/cycles/{id}/preferences` 只有 `module_id`；模块页反之；项目页两者都发；
  - 类型（`@ts-expect-error` 各一行）：迭代页的 service 配项目页的转换；带 `cycle_id` 的迭代页筛选字面量赋给迭代页的偏好；
- [ ] Step 3（GREEN）：实现；改构造处（断言不改）。变异：迭代页的转换留着 `cycle_id`（`t4-mut-keeps-own-key`）、模块页的转换去掉 `cycle_id`（`t4-mut-module-drops-cycle`）、`none` 走 `isId`（`t4-mut-none-refused`）、成员列表保留两词的分组（`t4-mut-member-groups`）；把 `update` 改回方法写法这一变异让类型测试失效、`tsc` 不再报错，以收据 `t4-type-guard`（`tsc` 在 `@ts-expect-error` 处报“未用的指令”）记下。
- [ ] Step 4：先构建 `@nerve/types`；`make lint-web`、`make knip`、`make test-web`（基线同数）；提交 `feat(M6/P7b): 筛选的迭代与模块（含无），偏好的筛选按页收窄`。

## Task 5（T5）：分组的词、列与泳道（A6，R4、R19）

**Files：**
- `web/packages/types/src/view-props.ts`（`TIssueGroupByOptions` 加 `cycle`、`module`）、`issues.ts`（`GroupByColumnTypes` 加两个；`IGroupByColumn` 加拒收的字段，取值是 nerve 的问题码之一）；`web/packages/constants/src/issue/common.ts`（`ISSUE_GROUP_BY_OPTIONS` 加两项，标题 `common.cycle`、`common.modules`）。
- `core/lib/issue-filters.ts`（`groups` 加 `cycle → cycle_id`、`module → module_id`）；`page-preferences-filter.store.ts`（`groupWire`、`groupValues` 同）。
- `issue-layouts/group-columns.tsx`（迭代、模块的列：spec §2.4 的次序、图标、payload、已归档的说明、拒收；项目一级也并入回答的组）；宿主 `list/default.tsx`、`kanban/default.tsx`、`kanban/swimlanes.tsx`（以组的 id 调 `usePlanningNames`；泳道两层都并入回答的组）；测试；`tools/keywords.json`。
- 不在本 Task：分组面板的开关门与常量里的选项（T6）、“+”与快速新建的隐藏（T6）、放下的拒收（T8）。

- [ ] Step 1：先跑回归边界“分组”一条所列的测试留基线（`t5-baseline-groups`）。
- [ ] Step 2（RED）：
  - 存下的 `group_by: cycle_id`、`sub_group_by: module_id` 被显示词认出（`show()`）、照原样写回（`merge`）；`projectListQuery` 把 `cycle`、`module` 映到 `cycle_id`、`module_id`；
  - 列：项目的迭代 Sprint、Next、Ended（已结束）、已归档的 Old、回答里有而两份列表都没有的 X：列依次 Sprint、Next、Ended、Old（名字与“已归档”）、X（“—”）、“无”；Old 叫出之前名字为“—”、次序照回答；Ended、Old 的列带拒收（`planning.cycle_completed`、`planning.archived`），Sprint、Next、“无”不带；payload `{cycle_id}`、`{}`；模块同（`{module_ids: [id]}`，已归档的拒收 `planning.archived`）；项目关着该域时有值的列拒收（`planning.cycles_disabled`）；
  - 宿主：按迭代分组、回答里有 Old 的组：组头经 `usePlanningNames` 读已归档的列表一次、以名字与说明显示；两个宿主（列表、看板）同一项目只读一次；开关关着时不读、组头“—”；
  - 泳道：按迭代分组、按模块子分组，回答里有已归档模块 Gamma 的子分组：泳道有 Gamma（名字与说明）；
  - 多模块的工作项在两个模块的组里各一次。
- [ ] Step 3（GREEN）：先构建 `@nerve/types`、`@nerve/constants`；实现。变异：列不并入回答的组（`t5-mut-no-answer-groups`，Old 的组没有列）、“无”不在最后（`t5-mut-none-first`）、已归档的按回答的次序排在未归档的之前（`t5-mut-archived-first`）、已结束的列不拒收（`t5-mut-completed-joinable`）、宿主不调 `usePlanningNames`（`t5-mut-host-no-names`，Old 的名字缺）、泳道不给回答的组（`t5-mut-lanes-without-ids`）。若本 Task 超出一个实现者的量，泳道一步移到 T6（台账写明），不加 Task。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线同数）、`node tools/keywords.mjs`；提交 `feat(M6/P7b): 按迭代与模块分组的列与泳道，已归档的组以名字显示，“无”在最后`。

## Task 6（T6）：按页的显示选项常量、开关的词表与分组里的新建（A8，R8、R12）

**Files：**
- `web/packages/constants/src/issue/filter.ts`（`issues` 的筛选加 `cycle_id`、`module_id`，分组、子分组加 `cycle`、`module`；`cycle_issues`、`module_issues` 由 `issues` 去掉本页的词推出，两份副本删去）与 `filter.test.ts`（扩到迭代页）。
- `core/lib/planning-switch.ts`（“规划的词 → 域”一张表、一个谓词，取代 `displayPropertyOn`）；它的两个读者 `filters/header/display-filters/display-properties.tsx`、`spreadsheet/spreadsheet-view.tsx` 改调它；`filters/header/display-filters/group-by.tsx`、`sub-group-by.tsx`（路由的项目关着该域时不给未选的，已选的留着）。
- 快速新建：一个共用的预填函数（项目的默认状态并上列与泳道的 payload），`list/list-group.tsx`、`kanban/kanban-group.tsx` 的两份 `prePopulateQuickAddData` 删去；拒收的列不给快速新建；分组头 `list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx` 在列拒收时不给“+”。
- 测试；`tools/keywords.json`。

- [ ] Step 1：先跑回归边界“快速新建与常量”一条所列的测试留基线（`t6-baseline-quick-add`、`t6-baseline-headers`）。
- [ ] Step 2（RED）：
  - 常量：`issues` 的列表、看板给 `cycle`、`module` 的分组与子分组和两个筛选；`cycle_issues` 恰为 `issues` 去掉 `cycle_id`、`cycle`，`module_issues` 去掉 `module_id`、`module`（其余逐项相同）；
  - 面板：路由的项目两域都开时“Cycle”“Modules”都在；只开模块时只有“Modules”；都关时都不在；都关而存下的分组是迭代时“Cycle”在且选中，点别的分组照常改；子分组同；个人页照给出的选项；
  - 快速新建：项目页按迭代分组，在 Sprint 列快速新建，请求体带 `cycle_id: Sprint`；按模块分组在 Alpha 列带 `module_ids: [Alpha]`；“无”列什么都不带；看板的格子（迭代 × 模块）两者都带；迭代页按模块分组时请求体有 Alpha 与本迭代（本页的注入）；按状态、优先级、标签、负责人分组时请求体与基线相同；
  - 拒收的列（Ended、Old）没有“+”与快速新建，别的列有。
- [ ] Step 3（GREEN）：先构建 `@nerve/constants`；实现；两页页头的测试不改即通过。变异：推出迭代页时不去掉 `cycle`（`t6-mut-cycle-page-keeps-cycle`，`cycles/(detail)/header.test.tsx:273` 须失败）、推出模块页时不去掉 `module_id`（`t6-mut-module-page-keeps-filter`，`filter.test.ts` 须失败）、面板不看开关（`t6-mut-panel-ignores-switch`）、已选的也藏（`t6-mut-selected-hidden`）、预填不带列的 payload（`t6-mut-quick-add-key`：退回 `{[groupByKey]: value}`，迭代列的请求体没有 `cycle_id`）、拒收的列仍给“+”（`t6-mut-refusing-creates`）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线同数）、`pnpm --filter @nerve/constants test`、`node tools/keywords.mjs`；提交 `feat(M6/P7b): 迭代页与模块页的显示选项由项目页推出，开关关闭时不给未选的，分组里的快速新建带上组的值`。

## Task 7（T7）：筛选的界面（A6，R7、R8）

**Files：**
- 新：`web/packages/utils/src/work-item-filters/configs/filters/` 的迭代、模块两个工厂（照 Plane 的 `cycle.ts`、`module.ts` 与本仓的 `label.ts`，类型来自 `@nerve/api-client`）与 `index.ts` 的导出；`core/hooks/work-item-filters/` 的一个新 hook（两个配置：选项、“无”、已持的已归档值、开关门；读经 `usePlanningListReads` 与 `usePlanningNames`）与测试。
- 改：`use-work-item-filters-config.tsx`（组合新 hook 的两个配置，仍在约 400 行以内）；项目一级的 HOC（给出项目，如须）；i18n（`common.filter_options.fields.cycle_id`、`module_id`，en 与 zh-CN）；`tools/keywords.json`；`.oxlintrc.json`（若新 hook 直接用 `useSessionSWR`）。

- [ ] Step 1（RED，`filter-config.test.tsx` 的写法）：
  - 项目页两域都开：“加筛选”列出 Cycle 与 Module；迭代的选项恰为“No cycle”、Sprint、Next、Ended（没有已归档的 Old），图标按状态；模块的选项为“No module”与未归档的模块；
  - 选“No cycle”：页面的偏好的 `filters.cycle_id` 为 `["none"]`，列表的查询带 `cycle_id=none`；
  - 存下的条件 `cycle_id__in: <Old>`：已应用的条件显示 “Old” 与“已归档”的说明（经 `usePlanningNames`，读已归档的列表一次），可去掉；
  - 迭代页不列 Cycle、模块页不列 Module（常量）；
  - 路由的项目关着迭代：“加筛选”不列 Cycle；存下的 Cycle 条件仍在、可去掉（值叫不出时是“Unavailable option”），列表照 nerve 的回答；两份列表都不读；
  - 两份未归档的列表与包装层同键（项目页的包装层已读时不再读）。
- [ ] Step 2（GREEN）：先构建 `@nerve/utils`、`@nerve/i18n`；实现。变异：迭代的选项去掉已结束的（`t7-mut-completed-dropped`）、没有“无”（`t7-mut-no-none`）、已持的已归档值不经 `usePlanningNames`（`t7-mut-held-unnamed`，显示“Unavailable option”）、配置不看开关（`t7-mut-filter-ignores-switch`）、迭代页列出 Cycle（`t7-mut-own-key-offered`：配置不看本页的允许，`filter-config.test.tsx` 的迭代页用例失败）。
- [ ] Step 3：`make lint-web`、`make knip`、`make test-web`、`pnpm --filter @nerve/utils test`、`node tools/keywords.mjs`；`wc -l` 记 `use-work-item-filters-config.tsx`（`t7-line-counts`）；提交 `feat(M6/P7b): 按迭代与模块筛选的界面，含无，已归档的已持值以名字显示`。

## Task 8（T8）：拖拽（检查点，A6，R3、R4）

**Files：** `web/packages/constants/src/issue/common.ts`（`DRAG_ALLOWED_GROUPS` 加 `cycle`、`module`）；`issue-layouts/group-drop.ts`（`handleGroupDragDrop` 给 `IssueChange`：迭代单值、模块增减、拖到“无”只移出来源、从“无”来只加入）；`core/store/issue/helpers/base-issues.store.ts`（`dropIssue` 收给 `IssueChange` 的函数，轮到时 `changeBody(change, held.module_ids)` 再设 `sort_order`）；动作的类型（`core/hooks/use-issues-actions.tsx`、`use-planning-page-issue-actions.ts`）；`core/hooks/use-group-dragndrop.ts` 或两个分组件（拒收的列放下不发请求、以 `TOAST_TYPE.WARNING` 说列上的原因，经 `errorMessageKey`）；类型测试；测试；`tools/keywords.json`。

- [ ] Step 1：先跑回归边界“拖拽”一条所列的测试留基线（`t8-baseline-drag`）。
- [ ] Step 2（RED，`FakeNerve`、`fake-queue.ts`）：
  - 迭代：从“无”拖到 Sprint，一个 `PATCH`，请求体 `{cycle_id: Sprint, sort_order}`；从 Sprint 拖到“无”`{cycle_id: null, …}`；
  - 模块：工作项在 Alpha 与已归档的 Gamma，从 Alpha 拖到 Beta：轮到时所持为 [Alpha, Gamma] → `module_ids` [Gamma, Beta]；扣住它之前的另一个写（加入 Delta）回答之后才轮到：[Gamma, Delta, Beta]（以前一个的回答为底）；从 Alpha 拖到“无”：[Gamma]（不清空）；从“无”拖到 Beta：所持并上 Beta；
  - 看板的格子（迭代 × 模块）同时改两者，一个请求；
  - 拒收：拖到 Ended（已结束）或 Old（已归档）的列、或开关关着时有值的列：不发请求，警告的提示是 nerve 拒绝的那一句；nerve 以 409 拒绝时提示 nerve 的原因，行留在 nerve 所在的组；确认之后两组的数目照 nerve；
  - 规划页的拖拽照旧跟进进度一次；
  - 类型（`@ts-expect-error`）：给出 `{ module_ids: [...] }` 的拖拽函数赋不进 `dropIssue`；
  - 状态、优先级、标签、负责人的拖拽请求体与基线相同（既有测试不改）。
- [ ] Step 3（GREEN）：先构建 `@nerve/constants`；实现。变异：拖到“无”清空模块（`t8-mut-none-empties`，Gamma 丢掉）、调用时就算整集合（`t8-mut-set-at-call`，Delta 丢掉）、模块的拖拽只加不减（`t8-mut-no-remove`）、拒收的列照发（`t8-mut-refusal-sends`）、拒收时不提示（`t8-mut-refusal-silent`）、`DRAG_ALLOWED_GROUPS` 不含 `module`（`t8-mut-module-not-draggable`）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线同数）、`node tools/keywords.mjs`；提交 `feat(M6/P7b): 拖到迭代与模块的组，请求体在写队列轮到时算，拒收的列不发请求`。
- [ ] Step 5（检查点）：`sh .superpowers/sdd/P7b-list-vocabulary/gates.sh t8-gate` 串行七项（`make test` 照负责人的规则；M6-M8-03、-05、-07 的签名照各条记下、只重跑那一项一次）；既有页面故事若因新的选项、项目页的新词失败，按回归边界只补契约并在台账写明；主控推送准确的 HEAD，核分支 CI，记录 e2e 作业的时长（校准点；超过 30 分钟先报告负责人，不改期限或作业结构），并照 spec §2.14 再算一次余量。

## Task 9（T9）：表格两列表头的排序与死分支（P7a R21，R9）

**Files：** `core/lib/issue-filters.ts`（`orders` 加 `±cycle_name`、`±module_name`）；`page-preferences-filter.store.ts`（`orderValues` 同）；`web/packages/constants/src/issue/common.ts`（`SPREADSHEET_PROPERTY_DETAILS.modules`、`cycle` 给两向的键与“A”“Z”；删去 `TUnsortedSpreadsheetProperty` 与映射类型的条件分支及注释）；`issue-layouts/spreadsheet/spreadsheet-sort.ts`（删去 `undefined` 一支与注释）；`spreadsheet/columns/header-column.tsx`（删去“只有标题”一支与注释）；`spreadsheet/spreadsheet-header-column.tsx`（`aria-sort` 恒有）；契约变化的 `spreadsheet/planning-columns.test.tsx:130`；测试。

- [ ] Step 1：先跑 `spreadsheet/` 下的测试留基线（`t9-baseline-spreadsheet`）。
- [ ] Step 2（RED）：
  - 表格里迭代一列的表头打开排序菜单，选 A→Z：`PATCH` 偏好的 `display_filters.order_by` 为 `cycle_name`，确认后列表的查询带 `order_by=cycle_name`（经 T3 的目录），行照名字的次序，`aria-sort` 为 `ascending`；Z→A 为 `-cycle_name`、`descending`；模块一列 `module_name` 同；
  - 存下的 `order_by: -module_name` 照原样显示（不映到 `-created_at`）、照原样发出；
  - 别的列的表头与 `header-column.test.tsx` 照旧。
- [ ] Step 3（GREEN）：先构建 `@nerve/constants`；实现；按契约改 `planning-columns.test.tsx:130`（台账写明 R9）；以 grep 核对死分支没有读者、删干净（`TUnsortedSpreadsheetProperty`、`spreadsheetSort` 的 `undefined`、“只有标题”）。变异：两列不给键（`t9-mut-no-keys`，编译不过则以 `tsc` 收据记下，另取能编译的“`orders` 不含 `cycle_name`”`t9-mut-order-unknown`：存下的名字排序映到 `-created_at`）、`aria-sort` 方向反了（`t9-mut-aria-reversed`）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`、`pnpm --filter @nerve/constants test`；提交 `feat(M6/P7b): 表格的迭代与模块两列按名字两向排序，删去没有排序键的分支`。

## Task 10（T10）：“添加已有”：写、命令宿主与空态（A4、A5，R2、R6、R11）

**Files：**
- `core/hooks/planning-page-domains.ts`（每域一列“加入本页的变化”，类型 `IssueChange`）；`core/lib/` 新一个有界并发的小函数（至多 n 个在途，逐项结果按给出的次序）与测试；`core/hooks/use-planning-page-issue-actions.ts`（“添加已有”的动作：每项 `changeIssue(id, {...extra, ...加入本页})`、至多 2 个在途、整批经 `followProgress` 包一次、回答 `SelectionSubmitResult`）与 `use-issues-actions.tsx`（`IssueActions` 加它；项目页的只发 `extra`）。
- `issue-layouts/command-host.tsx`（“添加已有”的请求：store 类型、`extra`、搜索参数、隐藏规则；由路由的项目与权限身份持有，`holds`；以请求的 store 类型的动作提交；规划页的搜索 `limit: 100`、隐藏已在本页的）；`issue-layouts/empty-states/planning-page.tsx`（`cta_secondary`，条件同“新建”；注释改为本段）；`issue-layouts/planning-page-layouts.ts`（`empty.none` 加这个键）；测试（照 `roots/row-dialog-lifetime.test.tsx` 与 `fake-planning-page.tsx` 的 `outlivesTheRow`）；`tools/keywords.json`。
- 不在本 Task：选项的当前迭代、全被排除的说明、分组头（T11）、日历（T12）。

- [ ] Step 1：先跑 `core/modals/existing-issues-list-modal.test.tsx`、`roots/row-dialog-lifetime.test.tsx`、`roots/group-create.test.tsx`、`roots/cycle-layout-root.test.tsx`、`roots/module-layout-root.test.tsx` 留基线（`t10-baseline-host`）。
- [ ] Step 2（RED，`FakeNerve`）：
  - 有界并发：5 项、上限 2：任一时刻至多 2 个在途，逐项的结果按给出的次序，某项被拒不影响别的；
  - 动作（迭代页）：3 项，第二项被拒（409）：恰发 3 个 `PATCH`，请求体各为 `{cycle_id: 本迭代}`，同时在途的至多 2 个；回答 `confirmedIds` 两项、`failures` 一项（带 nerve 的错误）；本迭代的进度只重读一次（全被拒时不重读）；模块页：所持 [Beta, Gamma]（Gamma 已归档）→ `module_ids` [Beta, Gamma, 本模块]，轮到时以所持算（之前的写回答之后再轮到时以它为底）；503 `server_busy` 的一项在 `failures` 里；
  - 宿主：迭代页的空态点“添加已有”打开弹窗（标题“Add existing work item”），搜索带 `project_id`、`limit=100`，已在本迭代的不列；选两项提交，第一项确认之后列表不再为空、空态卸下，弹窗仍开着，第二项被拒时留在选择里、提示原因；换路由或权限身份变了，弹窗结束；
  - 空态：可加入且是管理员或成员时有“添加已有”；访客、不可加入（已结束、已归档、开关关闭）、只读的页、有筛选时都没有。
- [ ] Step 3（GREEN）：实现。变异：弹窗渲染在空态里（`t10-mut-modal-in-empty-state`，空态卸下后结果不报）、不设上限（`t10-mut-unbounded`，在途数超过 2）、每项各跟进进度（`t10-mut-follow-each`，进度读 3 次）、不排除本页的（`t10-mut-no-exclusion`）、模块的加入在调用时以页面认得的模块算（`t10-mut-join-at-call`，Gamma 丢掉）、空态不看可加入（`t10-mut-empty-ignores-joinable`）。若本 Task 超出一个实现者的量，宿主对模块页的覆盖（同一张表的另一行）可移到 T11（台账写明），不加 Task。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`（基线同数）、`node tools/keywords.mjs`；`wc -l` 记 `command-host.tsx`、`use-issues-actions.tsx`（`t10-line-counts`）；提交 `feat(M6/P7b): 迭代页与模块页空态的“添加已有”，弹窗在命令宿主，逐个PATCH至多两个在途`。

## Task 11（T11）：“添加已有”：当前的迭代、全被排除的说明与分组头（A4，R2、R5、R20、R21）

**Files：** `core/components/core/modals/existing-issues-list-modal.tsx`（可省的每个选项的说明、可省的“全被排除”的说明；子工作项、关联的调用者不给）；`command-host.tsx`（迭代页给说明件：选项的工作项当前的迭代，经 `usePlanningNames`，已归档的带说明；给按域的“全被排除”的一句）；`planning-page-layouts.ts`（按域的键）；`list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx`（规划页、“+”给出、可加入、管理员与成员、列不拒收时，“+”旁的“添加已有”按钮，以“{添加已有}: {组}”为名，经宿主请求、不带组的 payload）；i18n（“全被排除”两句；按钮名若要新键）；测试；`tools/keywords.json`。

- [ ] Step 1（RED）：
  - 当前的迭代：迭代页的弹窗里，在未开始的 Next 里的 Y 的选项另显示 “Next”，在已归档的 Old 里的 V 显示 “Old” 与“已归档”（读已归档的列表一次，经 `usePlanningNames`），不在迭代里的 X 没有说明；模块页不显示；子工作项与关联的弹窗照旧（既有测试不改）；
  - 全被排除：搜索答了 2 项、都在本迭代：弹窗说它们已在本迭代，不是“没有找到”；答了 0 项时照旧；
  - 分组头：迭代页按状态分组，Todo 组头有“Add existing work item: Todo”（名字以组为名），提交选中的一项只发 `{cycle_id: 本迭代}`（没有 `state_id`）；项目页的组头没有；只读的页、访客、不可加入、拒收的列（迭代页按模块分组时已归档的 Gamma）没有；看板同；弹窗活过列表的核验（`outlivesTheRow` 的写法）。
- [ ] Step 2（GREEN）：实现。变异：说明不经 `usePlanningNames`（`t11-mut-caption-unnamed`，Old 的名字缺）、模块页也显示说明（`t11-mut-caption-on-module`）、没有“全被排除”的说明（`t11-mut-no-hint`）、分组头带上组的 payload（`t11-mut-header-joins-group`，请求体有 `state_id`）、项目页也给（`t11-mut-header-on-project`）、拒收的列也给（`t11-mut-header-in-refusing`）、分组头的弹窗不经宿主（`t11-mut-header-modal-in-group`，核验时卸下）。
- [ ] Step 3：先构建 `@nerve/i18n`；`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7b): “添加已有”的选项显示当前的迭代，全被排除时说明，分组头的入口只加入本页`。

## Task 12（T12）：日历的“添加已有”（A4，R6）

**Files：** `issue-layouts/calendar/quick-add-issue-actions.tsx`（日子的“添加已有”经命令宿主请求，`extra` 为 `{target_date: 那天}`，搜索 `{target_date: ["none"]}`，规划页另加 `limit: 100`；隐藏的规则照旧并上本页的排除；不再经 `useIssueDetail().updateIssue`、不再全部同时发）；契约变化的 `calendar/quick-add-issue-actions.test.tsx`（若须）；测试；`tools/keywords.json`。

- [ ] Step 1：先跑 `calendar/` 下的测试留基线（`t12-baseline-calendar`）。
- [ ] Step 2（RED）：
  - 迭代页的日历，某天的“添加已有”：搜索为 `{project_id, target_date: "none", limit: 100}`；已在本迭代的、有日子的、开始晚于那天的都不列；选 U 提交：只一个 `PATCH`，请求体恰为 `{target_date: 那天, cycle_id: 本迭代}`；被拒（例如 409）时 U 的日子也不变、留在选择里；本迭代的进度一批重读一次；模块页的日历：`{target_date, module_ids: 所持 ∪ {本模块}}`；
  - 项目页的日历：搜索恰为 `{project_id, target_date: "none"}`，请求体恰为 `{target_date}`（同 I4），至多 2 个在途；
  - 弹窗在宿主：列表核验（日历换成加载）时弹窗留着。
- [ ] Step 3（GREEN）：实现；按回归边界改日历的单测（有则，台账写明）。变异：规划页只发日子（`t12-mut-date-only`）、先发日子再加入（`t12-mut-two-requests`）、项目页也加 `limit`（`t12-mut-project-limit`，项目页的搜索参数断言失败）、弹窗留在日子的格子里（`t12-mut-day-modal-in-cell`）、规划页不排除本页的（`t12-mut-calendar-no-exclusion`）。
- [ ] Step 4：`make lint-web`、`make knip`、`make test-web`、`node tools/keywords.mjs`；提交 `feat(M6/P7b): 规划页日历的“添加已有”以一个请求带日子与本页，项目页照旧`。

## Task 13（T13）：页面故事一：G15 (page) 的工作项一侧

**Files：** `e2e/stories/planning/g15-work-item-lists-page.spec.ts`（新）；共用的世界与帮手若要新件（名字由实现定）；已有的帮手从原处导入（spec §2.14）。

- [ ] Step 1：写故事（spec §2.14 的表的第一行）。先以期望错误的版本看到断言失败（收据），再写产品变异：筛选的“无”被拒（`t13-e2e-mut-none-refused`，查询没有 `cycle_id`）、列不并入回答的组（`t13-e2e-mut-archived-column-missing`，Old 的组不见）、拒收的列照发（`t13-e2e-mut-drop-on-completed`，故事断言没有请求）、目录的查找总是叫不出（`t13-e2e-mut-catalog-empty`，按名字排序时列表失败）、表头的 `aria-sort` 反了（`t13-e2e-mut-aria-reversed`）。变异的脚本 `t13-e2e.sh` 依次跑 `react-router typegen`、web 与 e2e 的 `tsc --noEmit`、`make build build-s3-fixture` 与 chromium 的故事，只计能编译、能构建的；变异之后以干净的源重建。有两道守卫时以嵌套变异证明并写明单去一道为何等价。
- [ ] Step 2：`make build build-s3-fixture` 后 chromium 聚焦跑通；以 `--config playwright.m6-pages.config.ts` 在三种引擎上跑一次（收据 `t13-three-engines`），新故事另 `--repeat-each=3`（`t13-repeat3`）；从那次运行的报告记下本文件在各引擎的时长（台账，供 T16）；本机 chromium 超过 8.5 秒时先压缩（泳道一步已有单测，可移出故事；台账写明），两文件合计仍超 14 秒的余量时停下报告主控；`make lint-web`；提交 `test(M6/P7b): 项目页按迭代与模块筛选、分组、拖拽与表头排序的页面故事（G15）`。

## Task 14（T14）：页面故事二：G4、G11 (page) 的“添加已有”与弹窗的核对（A5）

**Files：** `e2e/stories/planning/g4-add-existing-page.spec.ts`（新，标题 `G4 (page): …` 与 `G11 (page): …`）；共用的帮手若要新件；已有的帮手从原处导入。

- [ ] Step 1：写两个故事（spec §2.14 的表的第二行），含 M3 交接 §3 的弹窗各项：`expect(modalDialog(page)).toHaveAccessibleName(…)`、后面的页面 inert、方向键到达选项、已选的移除按钮以“Remove {名字}”为名、选项里没有只在指针经过时显示的链接。每个故事先以期望错误的版本看到断言失败，再写产品变异：G4——弹窗渲染在空态里（`t14-e2e-mut-modal-in-empty-state`，第一项确认后弹窗没了）、排除任一迭代里的（`t14-e2e-mut-exclude-any-cycle`，Y 不列）、选项不显示当前的迭代（`t14-e2e-mut-no-caption`）、日历先发日子再加入（`t14-e2e-mut-calendar-two-requests`）；G11——分组头带组的 payload（`t14-e2e-mut-header-joins-group`，请求体有 `state_id`）、模块的加入在调用时以页面认得的模块算（`t14-e2e-mut-join-drops-archived`，数据库里没有 Gamma）。变异的脚本 `t14-e2e.sh` 同 T13。
- [ ] Step 2：同 T13 的 Step 2（收据 `t14-three-engines`、`t14-repeat3`；本文件本机 chromium 超过 5.5 秒先压缩，两文件合计超过 14 秒时停下报告主控）；再以页面配置跑整套三种引擎一次（`t14-full-pages`）；`make lint-web`；提交 `test(M6/P7b): 迭代页与模块页“添加已有”的页面故事（G4、G11）与弹窗的核对`。

## Task 15（T15）：文档与交接回执

**Files：**
- `docs/v0/plane-diff.md`：新行与扩充（plane-contract §8 第 10、12–15、17–20 条；Plane 的行号对照主目录的只读 `plane/` 逐一核过）：按迭代、模块筛选的界面（含“无”，本页的键不给，开关关闭时）；按迭代、模块分组的列（已归档的组、次序、“无”最后、拒收的列）；拖到迭代、模块的组（一个 `PATCH`、轮到时算、拖到“无”只移出来源）；分组里的新建（拒收的列不给、快速新建带组的值）；表格两列的表头（扩充 P7a 的“表格的迭代与模块两列”一行：两向按名字排序）；“添加已有”（排除本迭代的而非任一迭代、当前的迭代、上限 100 与客户端的排除、全被排除的说明、逐个 `PATCH` 至多 2 个在途、逐项报告、模块页说模块、分组头是“+”旁的按钮、只加入本页；扩充“模块页与迭代页的空态（M6/P6b）”一行写着“在 P7”的部分）；日历的“添加已有”（一个请求、原子）；开关关闭时的列表词；归档页没有两词（P8）。已登记的行引用不重复。
- `docs/v0/frontend-changes.md`：M6 一行（P7b）；“迭代和模块的归属”一行（“添加已有”已完成）；新行：列表的目录、列表词表与按页收窄、分组的列与拖拽、“添加已有”与命令宿主、绑定的拆分、表头排序与删去的分支。
- 输入交接的“处理结果（M6/P7b）”：[M4-closeout](../handoffs/M4-closeout.md#M6-02) M6-02（筛选、分组、排序、拖拽、目录、分组头与空态与日历的“添加已有”、`existing-issues-list-modal.tsx` 的两域调用方；“何时验收”逐项：同 fixture 与目录、跨分组写后计数（G15 的拖拽）、已归档的组（G15）、按名字排序（G15），双向分页、OwnOnly 与管理员兼访客、失权迟到答、归档范围、换会话引 P5b、P6b 的引擎证据并写明本段的词表走同一个绑定）；[M3-closeout](../handoffs/M3-closeout.md) §1（`existing-issues-list-modal.tsx` 经 M4 的搜索接口，新调用方保留它的边界）与 §3（弹窗的名字、后面的页面、选项里的链接、移除按钮的名字、列表 `modal={false}`，G4 (page) 在三种引擎核对）；均保持 open（设计 §11）。
- M8 交接：[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)（T8 检查点的数据）、[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)（本段之后根与写入单元的行数）、[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)（新故事照它的写法先等上方的内容）；M6-M8-03、-05、-07 再现时照各条的验收语补记。
- 设计：§8 的“工作项一侧”写明本段做了的与留给 P7c 的；§12 的 P7b 一行的验证照本段（spec §1、§2.14）；§14 的 P7b 行；spec §5 的补全（P7c 的 e2e 余量以 T16 的实测为准）。

- [ ] Step 1：写差异与回执。
- [ ] Step 2：`doc-links.py`、`node tools/keywords.mjs`；提交 `docs(M6/P7b): 交接回执、差异清单、前端改动清单与设计中P7b的进度`。

## Task 16（T16）：门禁、整审与合入（主控）

- [ ] Step 1：串行完整七项本地门禁（`gates.sh t16-gate`，留收据；`make test` 照负责人的规则），另以三种引擎的页面配置跑一次（收据）；`wc -l` 记根、写入单元、T2 拆出的三件、`command-host.tsx`、`use-work-item-filters-config.tsx` 与新 hook 的行数（M6-M8-08、E15）。推送之前照 spec §2.14 算出三个数并写进台账：
  - **估算**：基线（T1 记下的 main CI 43603772626 的 e2e 作业时长）加上 T13、T14 记下的新文件 chromium 时长 × 1.67–2.25，两个 worker 摊开；
  - **上界**：S（T1 的算法，并入 T8 检查点的实测）加上新文件 chromium 时长 × 3.6–4.1；写 spec 时 X = 14 秒约 29 分 13–26 秒；
  - **校准点**：T8 检查点的分支 CI 实测与基线之比，按它换算估算在那种 runner 上的数；慢的 runner（×1.31，M6-M8-01）另算一次。
  - 估算、上界或任一次实测（T8 的检查点、推送后的分支 CI）超过 30 分钟时停下报告负责人、不推送；改 CI 的期限或作业结构须负责人同意（E13）。否则推送准确的 HEAD，核分支 CI，记录 e2e 作业的实际时长（超过 30 分钟同样先报告）。
- [ ] Step 2：一次整分支 deep-review，由新的只读审者（fable）做：
  - Critical/Important 在一次修复轮里各有 RED/GREEN，Minor 记入台账或在修复轮做（P6b、P7a 的修复轮 F 的先例）；修复轮另经一次限定复审；
  - 写阶段记录 `reviews/P7b-list-vocabulary-review.md`（§7 留给 P7c 的 T1 回填）；
  - 勾选 spec §4 的 A1–A14；A15 待合入后由 P7c 的 T1 回填。
- [ ] Step 3：noff 合入，核准确的 main CI，归档（`archive.py create`）并正常清理。
