# M6 / P8 最终验收——最终矩阵、规模重跑、遗留收敛与交接实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：已完成（2026-11-16，noff `c4cd1880`，main CI 43851660778 attempt 1 全绿、没有重跑），T1–T16 已实施（T9 没有提交）：最终候选 `0ef66139` 的七项与两个页面配置、最终 HEAD `352740ac` 的分支 CI 43846387746 attempt 1 全绿，合入、归档与清理由 M6 收尾的第一个 Task 回填（[阶段记录](../reviews/P8-final-acceptance-review.md) §5、§7）。已写（2026-11-16），经预检 [re-78](../../../reviews/re-78-M6P8预检.md) 修正后实施；T1 `54976c83`。

**Goal：**
- 回填 P7c，phase 改 `M6/P8`；
- 网页的遗留：命令面板其余三个菜单的 cmdk value、三处英文字面量与表格的列宽、归档页的迭代与模块词、七处共用控件的焦点轮廓；
- 服务端：冻结任务逐项目继续、权限矩阵的访客一列、删除规模的夹具补上规划的数据与冻结的规模测试；
- 死成员与告警的收敛；
- 矩阵专用的页面核对（`e2e/acceptance/`，只在本机跑）与 G1–G20 的最终矩阵；
- 文档、七份输入交接的最终处理结果与 M7、M8 的 `M6-closeout`；门禁、整审、合入。

**Architecture：** 不加接口、表、依赖；服务端只改 `planning/app/freeze_cycles.go` 与测试。网页沿既有的件：命令面板的 `PowerKMenuBuilder` 与 `powerKFilter`（P7c F2、F5），列表的词表在 `@nerve/constants` 的 `ISSUE_DISPLAY_FILTERS_BY_PAGE`，名字经 `usePlanningNames`，开关关闭的规则同 P7b R8、R17。CI 的 e2e 时长不因本段增加：新的页面核对放在默认配置不跑的 `e2e/acceptance/`。

**Tech Stack：** 既有 Go/River/pgx/testcontainers、React/MobX/SWR/vitest/Playwright/cmdk，不加依赖。

**Spec：** [P8-final-acceptance](../specs/P8-final-acceptance.md)（§2 的合同与 §3 的裁定是本计划的约束；裁定的全文是台账的 `controller-rulings.md` H1–H16）；设计 [M6](../M6-design.md) §2、§10–§13；模板：M5 的 [P8 spec](../../M5-files/specs/P8-final-acceptance.md)、[plan](../../M5-files/plans/P8-final-acceptance.md) 与[阶段记录](../../M5-files/reviews/P8-final-acceptance-review.md)（T7 的死成员、T10 的演练、T12 的矩阵）。

## Global Constraints

**分支与提交**
- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p8`，分支 `codex/m6-p8`，基线 main `ea971606`；主目录永远在 main。
- 提交用详细中文，前缀 `feat/fix/refactor/test/docs(M6/P8)`，结尾 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`；推送由主控用 `/usr/bin/git`；noff 已获连续授权。
- 关键词 phase 在 T1 改 `M6/P8`；不延长任何 `until`。`api/` 不改；`server/` 只在 T6（产品代码只有 `planning/app/freeze_cycles.go` 与 worker `planning/adapter/river/freeze_cycles.go` 的注释）、T7、T8（只改测试）改。

**只读与禁区**
- `plane/`、`refer/` 只读，不在其内跑 git（读主目录的 `/Users/xiaoruan/project/nerve-project/plane/`）。
- 不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个（规模测试与 `make test`、`make e2e` 不同时跑）。

**代码规则**
- 手写文件约 400 行内；新文件的版权头为 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 网页：类型只来自 `@nerve/api-client`；SWR 只经 `useSessionSWR`；新代码不读 router store。Go：ports & adapters，用例只经端口，测试的夹具不进产品包。
- 删掉不用的代码；不加开关、桩或兼容别名（`expectOutlinesWhole` 移动之后原处不留转出；`generateQueryParams` 删去）。
- 不放宽 `tools/keywords.json` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`；`lint-cap` 要求告警数恰好等于上限，清掉告警的提交同时下调。
- 新文案 en 与 zh-CN 在同一提交（`check:sync` 不许一个键是另一个键的前缀）。
- 改了 `@nerve/constants`、`@nerve/utils`、`@nerve/ui`、`@nerve/i18n` 之后先 `pnpm --filter <pkg> build` 再跑应用的类型检查。

**测试纪律**
- 每个有行为变化的 Task 先写会失败的测试，看到它因所测性质失败（不是编译失败），再实现到通过。
- 每个性质至少一个能让测试失败的变异（名字见各 Task）；只计能编译的；e2e 的变异另要能构建，之后以干净的源重建。
- 每个网页 Task 跑 `make lint-web`、聚焦的 vitest 与涉及包的测试；改了导出或删了文件时 `make knip`。每个 Go Task 跑 `make lint-go`（`GOLANGCI_LINT=/Users/xiaoruan/project/nerve-project/bin/golangci-lint`）与聚焦的 `go test`（经 `run-check.py`）。

**回归测试的边界**
- 回归测试指 M2–M5 与 P1–P7c 的网页单测、Go 测试与 e2e。除下列，断言不改：
  - **T2**：`PK/menus/` 与 `PK/ui/modal/` 既有的测试一行不改（新的例加在旁边）。
  - **T3**：`common.actions.restore` 的 en 就是 “Restore”，断言它的既有测试（`roots/archive-page.test.tsx:145`、`roots/archive-detail.test.tsx:144`、`:152`、`:184`、`:246`、`peek-overview/restore-menu.test.tsx:107`）照过、不改；断言两条提示或 “No shortcuts found for” 英文原文的测试（若有）按契约改为翻译后的文案（台账逐条写明）。
  - **T4**：归档页的既有测试（`roots/archive-page.test.tsx`、`@nerve/constants` 的过滤表测试）只加例；断言“归档页没有迭代、模块词”的例（若有）按契约改（台账写明）。
  - **T5**：`e2e/fixtures/module-pages.ts` 的 `expectOutlinesWhole` 的调用处只改导入（`g19-module-pages.spec.ts` 等）；面包屑、行、侧栏的既有单测与默认套件的页面故事不改，是截断与滚动照旧的回归保护。
  - **T6**：`planning/app/freeze_cycles_test.go` 的 `TestFreezeCyclesKeepsTheProjectsBeforeAFailure`（`:288` 断言失败之后不再锁下一个）按契约改（台账写明），其余不改；`bootstrap/planning_freeze_test.go` 的 `TestFreezeJobLockTimeoutPreservesPostgresError`（`:176`）、`TestFreezeJobPartialCommitAndRetry`（`:227`）不改、照过。
  - **T7**：既有的矩阵行与列不改，只加变体行（新文件）。
  - **T8**：`TestTheDeletionsAtTheScaleTarget` 既有的断言不改，只加规划表的断言；`TestPurgeScale` 不改。
- 默认套件（`e2e/stories/`）不加文件、不加例；条数照旧 480（T14 以 `--list` 核对）。

**检查点与证据**
- 检查点：T11 结束时（全部产品变化与页面核对完成）串行完整七项（`gates.sh t11-gate`，本段改 Go，`make test` 必须通过；M6-M8-03 的签名只重跑那一个文件一次，M6-M8-05 的签名记录之后重跑那一个包一次，再失败即停下报告主控），主控推送并核分支 CI；e2e 作业应与 P7c 的同量级（默认套件不变），超过 30 分钟时停下报告负责人；web 作业（期限 15 分钟）与服务端两路（期限 25 分钟）每次记下时长：web 作业再被期限取消时按 P7c 的先例只重跑失败的作业一次、在阶段报告里告诉负责人，服务端任一路超过 20 分钟时报告负责人。检查点之后若改了产品代码（修复轮），T15 的最终候选再串行一次完整门禁。
- 证据在 `.superpowers/sdd/P8-final-acceptance/`：台账 `progress.md`，收据经 `run-check.py <label> <命令…>`（在 worktree 的根跑），变异经 `mutate.py`，七项门禁经 `gates.sh`，最终候选经 `final.sh`（七项、m6-pages 与 m6-acceptance），文档链接经 `doc-links.py`，时长经 `durations.py`，CI 日志摘要经 `ci-e2e-summary.py`，归档经 `archive.py`。实现代理先读同目录的 `implementer-brief.md`。
- 台账的行：`Task N: Step k RED/GREEN — <label> exit <code>（…）`、`Task N: 基线 — …`、`Task N: 变异 — <label>（…）→ 失败（哪个测试），restored=yes`、`Task N: Ruling: <决定> — <理由> — 代价：<代价>`、`Task N: complete (commits <first>..<last>, tests: <summary>)`。

## Review Focus

1. **轮廓（§2.5）**：只改轮廓，不改截断与滚动；面包屑的改动对各页无副作用；三种引擎的核对能被变异杀死。
2. **冻结（§2.6）**：逐项目继续、错误合并、`ctx` 结束即停、已提交的不重做；锁序与“锁后读时钟”照旧。
3. **矩阵与规模（§2.7、§2.8）**：访客一列覆盖 P1–P3a 的每个写；规模夹具的行精确归零、邻域不受影响；规模测试默认跳过。
4. **归档页（§2.4）**：词表与项目页同规则；只读页没有写的入口；开关关闭同 P7b。
5. **默认套件不变**（§3 第 1 条）：`e2e/stories/` 没有新文件；`e2e/acceptance/` 的配置继承默认配置、只换 `testDir` 与引擎。
6. **收敛（§2.9、§2.10）**：删的确实没有读者；有读者的逐行写明；cap 等于实际。
7. **交接（§2.13）**：每条输入条款有结果或具名去处；M7、M8 的新条目各有出处。

## Task 1（T1）：P7c 回填与 phase

- [ ] 核对：`python3 .superpowers/sdd/P8-final-acceptance/archive.py verify /Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P7c-integration-20261116`（收据 `t1-p7c-archive-verify`；主控记的是 4718 个文件、751,178,965 字节、6493 个成员、未知 0、清单 sha256 `cfb5ce03c0f8812c478df8a9074da3b3d5ca21f143d83646a01f5ac8a3419399`、head `cf215555`）；`git rev-parse ea971606^1 ea971606^2 ea971606^{tree} cf215555^{tree}`（`t1-merge-tree`）；台账已有 `t1-ci-jobs`（两次 run 的各 attempt）与 `ci-logs/` 三份 e2e 作业日志，以 `python3 -I .superpowers/sdd/P8-final-acceptance/ci-e2e-summary.py <log> g19-work-item-side-page` 取条数与 G19 的三条（`t1-e2e-summary`）。不符即停下报告。
- [ ] 回填 [P7c 阶段记录](../reviews/P7c-palette-mobile-review.md) §7（体例同 P7b 记录 §7：最终 HEAD `cf215555` 的分支 CI 43759328881 两个 attempt——attempt 1 web 132950361275 在 15 分钟的期限被取消（902 秒，各步成功）、e2e 跳过、server 两路过，attempt 2 只重跑失败的作业、全绿（web 602 秒，e2e 132958271577 24 分 42 秒，480 过、故事 22.3 分钟，G19 三条 13.1 秒）；推送前的估算与上界的对照；noff 的双父与树；main CI 43764306705 attempt 1 的 e2e 132970815083 以 M4 的 C7 失败一条（479 过，trace 的摘要在归档的 `ci-evidence/ci-main-43764306705/`）、attempt 2 只重跑失败的作业（e2e 132979921078 27 分 44 秒 = 1664 秒，480 过、故事 25.6 分钟，G19 16.5 秒）；归档；清理；接续指向本段），P7c spec 的 A14 勾选与状态行、P7c plan 的状态行，设计的文件头与 §14 的 P7c 一行（已完成）、P8 一行（进行中，链接本 spec 与 plan），`frontend-changes.md:205`。
- [ ] [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 记 P7c 的数据点（分支 1482 秒、main 1664 秒；web 作业 902 秒碰到期限；C7 一次）；台账写明“本段不给默认套件加故事，余量不变”。
- [ ] `tools/keywords.json` 的 phase 改 `M6/P8`，`node tools/keywords.mjs`（`t1-keywords`）；文档链接（`t1-doc-links`）。
- [ ] 提交 `docs(M6/P8)`。

## Task 2（T2）：命令面板其余菜单的 cmdk value（spec §2.2）

- [ ] RED：`PK/menus/` 的三个菜单各加例（同名的两项都能选到、选中回的是各自的 id；改名之后按新名找到）；`builder` 的例（`getKeywords` 给出时行的 keywords）。收据 `t2-red`。
- [ ] `PowerKMenuBuilder` 加 `getKeywords`（不给时照旧），迭代（打开）、视图、工作区三个菜单以 id 作 value、名字作 keywords；GREEN `t2-green`。
- [ ] 变异：`t2-mut-value-name`（迭代菜单的 value 换回名字 → 同名的例失败）、`t2-mut-no-keywords`（builder 不传 keywords → 按名搜索的例失败）。
- [ ] `make lint-web`；提交 `fix(M6/P8)`。

## Task 3（T3）：英文字面量与表格的列宽（spec §2.3）

- [ ] RED：`helper.tsx` 的“恢复”与两条提示、`shortcut.tsx` 的空结果以 zh-CN 渲染的例（`t3-red-i18n`）；开关关时表格的 `colSpan` 与加载行的列数等于表头的列数（`t3-red-colspan`）。
- [ ] 改为键（en、zh-CN，新键名在台账写明；`handleOptionalAction` 没有 `t`，调用处传已翻译的提示，不插英文的动作名，spec §2.3）；表格的列数改为与表头同源，`getDisplayPropertiesCount`（`issue-layouts/utils.tsx:27`）随之删去；GREEN `t3-green`。
- [ ] 变异：`t3-mut-literal`（“Restore”换回字面量）、`t3-mut-colspan`（恢复原来的计数）。
- [ ] `make lint-web`、`make knip`；提交 `fix(M6/P8)`。

## Task 4（T4）：归档页的迭代、模块词（spec §2.4）

- [ ] 读 Plane 的归档页（主目录 `plane/packages/constants/src/issue/filter.ts:142-160` 与归档页的筛选、分组的件）与 nerve 的 `archived_issues`（`web/packages/constants/src/issue/filter.ts:158-181`）、归档页的本地偏好与面板，台账写明落点。
- [ ] RED：`@nerve/constants` 的过滤表例（归档页有 `cycle_id`、`module_id` 筛选与 `cycle`、`module` 分组键——网页的分组键是单数的 `module`，`filter.ts:62`、`:217`）；归档页的面板例（开关开时给、关时不给未选的、已选的留着可去掉）；组头与条件的名字经 `usePlanningNames`（已归档的带说明）。收据 `t4-red`。
- [ ] 实现；归档页没有拖放、快速新建、“添加已有”（核对既有的只读守卫，不加写的入口）；`plane-diff.md` 的“归档页的迭代、模块词（M6/P7b）”一行改写。GREEN `t4-green`。
- [ ] 变异：`t4-mut-no-vocab`（从过滤表去掉 `cycle_id` → 例失败）、`t4-mut-switch-ignored`（开关关时照给 → 例失败）。
- [ ] `pnpm --filter @nerve/constants build`、`make lint-web`、`make knip`；提交 `feat(M6/P8)`。

## Task 5（T5）：共用控件的焦点轮廓（spec §2.5）

- [ ] 读 M3 交接 §3 已核对的写法（`Button`、`IconButton` 的 `focus-visible` 类）与 `e2e/fixtures/browser.ts` 的 `focusOutline`、`e2e/fixtures/module-pages.ts:106` 的 `expectOutlinesWhole`；台账记七处各自的改法。
- [ ] 把 `expectOutlinesWhole` 移到 `browser.ts`（原处删去、调用处改导入），默认套件里用到它的故事在 chromium 照过（`t5-move`）。
- [ ] 七处的修改（只改轮廓的类与可聚焦元素的空间，不改截断与滚动）；`@nerve/ui` 改后 `pnpm --filter @nerve/ui build`。页面证明在 T11 的核对（三种引擎）；本 Task 先以一个临时的本机探针（不入库）在三种引擎量七处（(c)、(g) 只在 Firefox、WebKit 出现），改前改后各一次，记 `t5-probe-before`、`t5-probe-after`；面包屑的可聚焦链接由调用处渲染（`@nerve/ui` 的 `breadcrumbs.tsx:46`、`:132` 是容器），修改可能落在两处。
- [ ] `make lint-web`、`make knip`；提交 `fix(M6/P8)`。

## Task 6（T6）：冻结任务逐项目继续（spec §2.6）

- [ ] RED：加例（`planning/app/freeze_cycles_test.go` 将过约 400 行时放新的测试文件）：三个候选项目、中间的一个事务失败——第一、第三个都提交、数目是两者之和、返回的错误 `errors.Is` 中间那个的原错误且带它的 id；`ctx` 在第二个之后取消——不再往下、返回的错误含此前的失败与 `ctx.Err()`；七个项目都失败——错误里恰有前 5 个的原错误与“另有 2 个”。收据 `t6-red`。
- [ ] 实现（`errors.Join`，前 5 个项目的错误加其余的计数；`app/freeze_cycles.go` 与 worker `adapter/river/freeze_cycles.go:41-42` 的注释、`README.md:129`、[P4 spec](../specs/P4-progress-transfer.md) `:104` 那句改写，台账写明）；GREEN `t6-green`。
- [ ] 变异：`t6-mut-stop-at-first`（恢复遇错即返回 → 例失败）、`t6-mut-drop-error`（继续但不交回错误 → 例失败）、`t6-mut-no-cap`（不设上限 → 七个都失败的例失败）、`t6-mut-drop-ctx`（`ctx` 结束时丢掉此前的失败 → 例失败）。
- [ ] `make lint-go`、`go test ./internal/modules/planning/...`（`t6-pkg`）与 `bootstrap` 的冻结测试（`-run 'Freeze'`，`t6-bootstrap`）；提交 `fix(M6/P8)`。

## Task 7（T7）：权限矩阵的访客变体行与归档列表的词（spec §2.7、§2.4）

- [ ] 读 `bootstrap/permission_matrix*_test.go` 与转移的访客变体（`permission_matrix_transfer_test.go:37-39`、`guestsSeeAll` 在 `permission_matrix_counts_test.go:164`）；改前聚焦运行一次，记 `TestPermissionMatrix` 的时长（`t7-matrix-before`）。
- [ ] 新文件为 spec §2.7 列出的 13 个写操作各加一个 `variant: "guests see all"` 的行（已有 prepare 的组合它）；期望：访客 `forbidden`，工作区管理员兼访客照过；框架支持时变体只跑这两列。不加的四类在文件头的注释写明理由（spec §2.7）。聚焦运行 `-run 'TestPermissionMatrix|Coverage|Matrix|EveryColumnCalls'`（`t7-matrix`），时长与改前比较：增加超过 60 秒即停下报告主控。
- [ ] 变异：`t7-mut-cycle-write-as-read`、`t7-mut-module-write-as-read`（各把一个写的动作名换成读 → 变体行失败）。
- [ ] 归档列表的词：在 bootstrap 的列表测试加一例，`archived=true` 与 `cycle_id`、`module_id` 筛选、`group_by=module_id`（接口的分组键；网页的分组键是 `module`）合在一起的回答恰如所选（`t7-archived-vocab`）。
- [ ] `make lint-go`；提交 `test(M6/P8)`。

## Task 8（T8）：规模（spec §2.8）

- [ ] 读 `bootstrap/issue_scale_fixture_test.go:81-102` 与 `TestTheDeletionsAtTheScaleTarget`、`TestPurgeScale` 的开关与证据目录；M5 P8 阶段记录的规模数（一个项目 7.6 秒、五个 14.3 秒、邻域编辑至多 68 毫秒）。
- [ ] 夹具播种规划的数据：每项目 8 个迭代、6 个模块（每个模块 3 个成员、2 个链接），每个工作项在一个迭代、两个模块里；`ANALYZE`（`issue_scale_fixture_test.go:109`）加上规划的表。
- [ ] “没有在跑的删除任务”的基线加上 `planning.delete_rows`；项目一步与工作区一步各找到每个被删项目的一个 `planning.delete_rows` 任务，以 `waitPlanningDeletion`（`bootstrap/planning_deletion_fixture_test.go:93`）等它们完成（完成、没有错误；snooze 与批数记下而不禁止；运行秒数），然后才断言规划的表与 `cycle_issues`、`module_issues` 活的行为 0、总行数不变（软删除），写明这两张关系表由哪一个任务删；邻域不受影响。既有的断言不改；若加进来的任务扰动了既有的 90 秒等待或单次运行的断言，停下报告主控。
- [ ] 冻结的规模测试（新文件，`NERVE_MEASURE_SCALE=1` 才跑）：夹具里绝大多数迭代已冻结或未结束并 `ANALYZE`；`EXPLAIN` 断言候选查询用到 `cycles_freeze_due_idx`（规划器仍选顺序扫描时记下计划、停下报告主控）；量一次运行的总时长与逐项目的锁持有时长，证据写入 `NERVE_SCALE_EVIDENCE`。
- [ ] 默认 `make test` 下两者跳过（`t8-default-skip`）；本 Task 只跑一次小规模的冒烟以证测试能跑（`t8-smoke`，台账写明参数）；正式的规模运行在 T12。
- [ ] `make lint-go`；提交 `test(M6/P8)`。

## Task 9（T9）：死成员（spec §2.9）

- [ ] 照 M1 收尾 plan 附录 A（`docs/v0/M1-frontend-trim/plans/closeout.md:1062`、`:1368`、`:1512`）建脚本于台账（不入库），先 `react-router typegen`，再在 HEAD 跑 `--rows M6`（`t9-before`，tsv 入台账）。
- [ ] 删除真死成员与只为它们存在的代码；复扫（`t9-after`），没有新行。（T9 的结论：没有真死成员，`fake-module-dialog.tsx:32-33` 的两个字段有读写者，见 spec §2.9；T9 不提交，读者表在台账 `t9-readers.md`，T12 抄入阶段记录。）
- [ ] 产品文件里有读者的行逐行写明文件、名字与读者（写进阶段记录的草稿，台账 `t9-readers`）；M1 交接的 84 行与 M3 §2 的 8 行的去向。
- [ ] `make lint-web`、`make knip`；提交 `refactor(M6/P8)`。

## Task 10（T10）：告警的一类（spec §2.10）

- [ ] 基线：全仓 oxlint 按规则计数（`t10-baseline`）。
- [ ] 清 `no-unused-expressions`（4 条；删 `@nerve/utils` 的 `generateQueryParams` 这个死导出，其余等价改写），`.oxlintrc.json` 设为 error（全仓生效，含 e2e），各包 `check:lint` 脚本里的 cap（`web/apps/web/package.json:10`、`web/packages/utils/package.json:18`）随实际下调；`make lint-web` 过（`t10-lint`）。
- [ ] M6 全期改动文件的收据：`git diff --diff-filter=AMR --name-only ba3fd6e0..HEAD` 的可 lint 文件 oxlint 0（`t10-oxlint-m6`）。
- [ ] 提交 `refactor(M6/P8)`。

## Task 11（T11）：矩阵专用的页面核对与检查点（spec §2.11）

- [ ] `e2e/playwright.m6-acceptance.config.ts`（继承默认配置，`testDir: "acceptance"`，`projects` 整个换成三种引擎，同 `playwright.m5-slow.config.ts`，不继承 `s3-fixture` 与 Firefox 的 `grep`）、`e2e/package.json` 的 `test:m6-acceptance`；核对 e2e 的 `tsconfig.json`（`**/*.ts`）与 `check:lint`（`.`）已覆盖 `acceptance/`（不改）；默认配置不改。
- [ ] 五类核对（标题 `G<n> (acceptance): …`）：焦点轮廓（迭代页、模块页、§2.5 的七处与迭代页的四种共用控件、P7a–P7c 的新控件，含 390 × 844 的移动端页头；另量行上 M4 的单日期下拉框 `dropdowns/date.tsx:94`——它同为 `outline-none`，T5 的观察，轮廓不完整时照 T5 的内缩轮廓修）；命令面板的行（`aria-selected`、可见的高亮、整行在滚动区之内，不用 `expectOutlinesWhole`）；新建表单里换项目；归档页的迭代、模块词；模块的开关（G18）。第一次量到的控件轮廓不完整时，在本 Task 于根因处修（RED 即这条核对，台账写明），跨模块的停下报告主控。写法照 P7b 阶段记录 §5 与 P7c 的教训（锚在目标页独有的内容、不 reload、`dismissToasts`、窄屏用 `setViewportSize`）。
- [ ] 三种引擎各过（`t11-acceptance`），逐条时长（`durations.py`）。
- [ ] 变异（以干净的源重建）：`t11-mut-outline-a`、`t11-mut-outline-d`、`t11-mut-outline-e`（恢复修改之前的类 → 轮廓的核对失败）、`t11-mut-outline-c`（Firefox、WebKit）、`t11-mut-outline-g`（WebKit）；`t11-mut-archived-vocab`（归档页的过滤表去掉 `cycle_id` → 核对失败）。
- [ ] 提交 `test(M6/P8)`（本 Task 修的产品代码另提交 `fix(M6/P8)`）。
- [ ] **检查点**：`sh .superpowers/sdd/P8-final-acceptance/gates.sh t11-gate`；主控推送并核分支 CI（e2e、web、服务端两路的时长照“检查点与证据”的规则）。

## Task 12（T12）：最终矩阵的运行（主控，spec §2.12）

- [ ] 串行：`make build build-s3-fixture`；默认套件加 JSON 报告（`t12-default`）；`test:m6-pages`（`t12-m6-pages`）；`test:m6-acceptance`（`t12-acceptance`）；间歇项隔离重复至少 5 次（`t12-repeat-*`）；`GOFLAGS='-p=1 -v' make test`（`t12-go-full`）；矩阵的聚焦运行（`t12-matrix`）；`TestPurgeScale`（`t12-purge-scale`）、扩展后的删除规模（`t12-delete-scale`）、冻结的规模（`t12-freeze-scale`）。
- [ ] [阶段记录](../reviews/P8-final-acceptance-review.md)的 G1–G20 表：每个 G 的接口、数据库与页面证据，指向本次运行的原件与它所在的 HEAD；间歇项的结论；规模的数字。
- [ ] 设计 §13 的 14 条（`M6-design.md:257-270`）逐条对到本次运行的收据或写明“closeout”（spec §2.12 的对照）；H3 的理由写进阶段记录。
- [ ] 提交 `docs(M6/P8)`。

## Task 13（T13）：文档与交接（spec §2.13）

- [ ] `README.md`、`server/configs/config.yaml:120` 的注释、`frontend-changes.md`、`plane-diff.md`、设计的状态；文档链接零断（`t13-doc-links`）。
- [ ] 七份输入交接逐条写“处理结果（M6/P8）”；M7、M8 的 `M6-closeout` 补全（spec §2.13 的各条，各有出处）。
- [ ] 提交 `docs(M6/P8)`。

## Task 14（T14）：整分支审查与修复（主控）

- [ ] 整分支 deep-review（fable，只读）与本机七项（`gates.sh t14-gate`）并行；默认套件的条数核对（`t14-list`：480）。
- [ ] 修复轮（若有）：Critical/Important 一次修复、RED/GREEN；Minor 入台账；限定复审。修复改了产品代码时，最终矩阵受影响的行在 T15 的最终候选上重跑；自 T12 以来只改了文档或无关测试的行沿用 T12 的原件；规模、隔离重复与 `-v` 的 Go 全量不在 `final.sh` 里，受影响时单独重跑。

## Task 15（T15）：最终候选与分支 CI（主控）

- [ ] `sh .superpowers/sdd/P8-final-acceptance/final.sh t15-final`（七项、m6-pages、m6-acceptance）；阶段记录 §5、§6。
- [ ] 推送；分支 CI 全绿（已知的间歇照既有规则至多一次）。

## Task 16（T16）：合入、main CI、归档与清理（主控）

- [ ] noff 合入 main、推送、main CI；归档（`archive.py create`、`verify`）、清理（`cleanup.log`）、记忆。阶段记录 §7 与 spec A14 由 closeout 的第一个 Task 回填。
