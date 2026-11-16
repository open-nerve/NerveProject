# M6 / P8：最终验收——最终矩阵、规模重跑、遗留收敛与交接

> 状态：spec 与 plan 已写（2026-11-16），经预检 [re-78](../../../reviews/re-78-M6P8预检.md) 修正后可实施（I1–I6、m1–m11 改入，m8 不改）。基线 main `ea971606b829fb3ade4beb9d204ecbc23d3af674`（P7c 的 noff；main CI 43764306705 attempt 2 全绿）。

文中的路径：`core/…` 在 `web/apps/web/` 下；`PK/…` 是 `core/components/power-k/…`；`issue-layouts/…` 是 `core/components/issues/issue-layouts/…`；`@nerve/…` 的包在 `web/packages/` 下；`planning/…` 是 `server/internal/modules/planning/…`；`bootstrap/…` 是 `server/internal/bootstrap/…`。台账 `.superpowers/sdd/P8-final-acceptance/`（不入库）的 `research/p8-inbound.md`、`research/g-matrix.md`、`research/cleanup-lint-docs.md` 是本 spec 的事实来源（于 `ea971606`）。

## 1. 目标与边界

P8 是 M6 最后一个编号阶段（[M6 设计](../M6-design.md) §12 的 P8 一行）：在最终代码上为 G1–G20 各给一条与失败路径相区分的最终证据，重跑清理与删除的规模测试并补上 M6 的数据，修掉 P1–P7c 交下的遗留，收敛死成员与告警，同步文档，写出七份输入交接的最终处理结果与给 M7、M8 的 `M6-closeout` 交接。

- **回填**：P7c 阶段记录 §7、spec A14、设计 §14 与文件头、`frontend-changes.md`、[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的数据点；关键词的 phase 改 `M6/P8`（§2.1）。
- **网页的遗留**：命令面板其余三个菜单的 cmdk value（§2.2）；三处英文字面量与表格的列宽（§2.3）；归档页的迭代、模块词（§2.4）；共用控件的焦点轮廓（§2.5）。
- **服务端**：冻结任务逐项目继续（§2.6）；权限矩阵的“看得到全部工作项的访客”一列（§2.7）；删除规模的夹具补上规划的数据、冻结的规模测试（§2.8）。
- **收敛**：死成员（§2.9）；oxlint 清一类规则、M6 全期改动文件 0 告警（§2.10）。
- **验收**：矩阵专用的页面核对与一个只在本机跑的配置（§2.11）；最终矩阵的运行与 G1–G20 的证据表（§2.12）；文档与交接（§2.13）；门禁与 CI（§2.14）。

**不做**：新的接口、表、依赖与权限事实来源；CI 的预算、期限与作业结构（负责人的开放问题，§3 第 1 条）；给默认套件加故事；M7、M8 领域的功能（§5）。两轮 noff/main、设计 §13 的勾选、交接的 `status: closed` 与总体 §9.4 属 closeout，不在 P8 提前写。

## 2. 交付合同

### 2.1 回填与 phase

按 P7c 的归档 `.superpowers/sdd/M6-P7c-integration-20261116`（`manifest.json`、`cleanup.log`、`ci-evidence/`）与台账的 `t1-ci-jobs` 收据、`ci-logs/` 的作业日志（经 `ci-e2e-summary.py` 以 `python3 -I` 读）回填 [P7c 阶段记录](../reviews/P7c-palette-mobile-review.md) §7（照 P7b 记录 §7 的写法：最终 HEAD `cf215555` 的分支 CI 43759328881 两个 attempt——attempt 1 的 web 作业在 15 分钟的期限被取消、e2e 跳过，attempt 2 全绿；noff `ea971606` 的双父与树；main CI 43764306705 两个 attempt——attempt 1 的 e2e 以 M4 的 C7 失败一条，attempt 2 只重跑失败的作业、全绿，e2e 27 分 44 秒 = 1664 秒；归档的计数与 sha256、清理），勾选 P7c spec 的 A14。设计的文件头与 §14 的 P7c、P8 两行、`frontend-changes.md:205` 照实改写；M6-M8-01 补 P7c 的数据点（分支 1482 秒、main 1664 秒的 e2e，web 作业 902 秒碰到期限，C7 一次）。`tools/keywords.json` 的 phase 改 `M6/P8`。P8 不给默认套件加故事，e2e 的余量不变，只在台账记下这一判断与默认套件的条数（480）。

### 2.2 命令面板其余菜单的 cmdk value

迭代（打开）、视图、工作区三个菜单（`PK/menus/cycles.tsx:26`、`views.tsx:26`、`workspaces.tsx:32`）今天以名字作 cmdk 的 `value`：同名的两项互相遮蔽、改名之后搜索仍认旧名。照 P7c 的 F2、F5：`PowerKMenuBuilder`（`PK/menus/builder.tsx:28`、`:57`）加 `getKeywords`，三个菜单以 id 作 value、名字作 keywords，搜索照旧按名字过滤（共用 `powerKFilter`）。单测：同名两项都能选到、改名之后按新名找到。

### 2.3 英文字面量与表格的列宽

- `core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx:219` 的 “Restore” 改读已有的 `common.actions.restore`（en 的文案就是 “Restore”，断言它的既有测试照过）；同文件 `:53-54` 的两条提示（“Action not available”等）同类，改为键（en、zh-CN）：`handleOptionalAction`（`:39-57`）是模块级的函数、没有 `t`，调用处传的是英文的动作名（`:140` 的 “Restore”、`:213` 的 “Archive”），改为调用处传已翻译的提示（或动作的键），不在提示里插英文的动作名。
- `PK/ui/renderer/shortcut.tsx:105` 的 “No shortcuts found for” 改为新键（en、zh-CN），插值照旧。
- 项目关着迭代（模块）时，表格的合并单元格与加载行仍按迭代、模块两列计数（`issue-layouts/spreadsheet/spreadsheet-table.tsx:110` 对 `spreadsheet-view.tsx:66`），宽出一两列：改为与表头同一个列的来源；`getDisplayPropertiesCount`（`issue-layouts/utils.tsx:27`）只有这一个读者，换掉之后删去。单测：开关关时 `colSpan` 等于表头的列数。

### 2.4 归档页的迭代、模块词

给。Plane 的归档页给 `cycle_id`、`module_id` 的筛选与按 `cycle`、`module` 分组（plane `packages/constants/src/issue/filter.ts:142-160`）；nerve 的归档列表是同一个 `GET /api/v0/projects/{project_id}/issues?archived=true`，与项目页经同一个解析（`issue/app/list_issues.go:26-41` 的 `ParseListQuery(raw, domain.ProjectList, …)`），服务端不改；今天没有服务端测试把 `archived=true` 与 `cycle_id`/`module_id`/`group_by` 合在一起，T7 补一例。`@nerve/constants` 的 `archived_issues`（`web/packages/constants/src/issue/filter.ts:158-181`）加 `cycle_id`、`module_id` 两个筛选与 `cycle`、`module` 两个分组键（网页的分组键是单数的 `module`，`filter.ts:62`、`:217`；`modules` 是显示属性）；归档页的本地偏好照收这四个词。归档页只读：没有拖放、快速新建与“添加已有”，拒收的列不适用；名字经 `usePlanningNames`（已归档的带说明）；开关关闭时同项目页（P7b R8、R17：不给未选的分组与筛选，已选的留着可去掉，组头 “—”，条件的值“不可用的选项”）。`plane-diff.md` 的“归档页的迭代、模块词（M6/P7b）”一行改写为本段。单测：归档页的词表与开关关闭时的面板；页面核对见 §2.11。

### 2.5 共用控件的焦点轮廓

P6a 交下的七处在根因处修（[P6a spec](P6a-web-modules.md) §5 的 P8 一条，F2 的探针）：(a) 行的链接被 `ListItem` 的 `truncate` 容器裁掉（`core/components/core/list/list-item.tsx:69-71`，迭代与模块的行共用）；(b) 侧栏链接一行的链接（`core/components/modules/links/list-item.tsx:56`、`:65`）；(c) “Add link”在 Firefox、WebKit 被链接区的 `overflow-y-auto` 裁掉（`core/components/modules/progress-sidebar/sidebar-links.tsx:83-90`、`:112`）；(d) 面包屑的链接被 `overflow-hidden` 裁掉（`@nerve/ui` `src/breadcrumbs/breadcrumbs.tsx:46`、`:132`，M3 的共用件，各页都有）；(e) `DateRangeDropdown` 的 `outline-none`（`core/components/dropdowns/date-range.tsx:123`）；(f) `ProgressTabs` 的 `outline-none`（`core/components/core/sidebar/progress-tabs.tsx:114`）；(g) 迭代列表的分节按钮在 WebKit 被列表的滚动容器裁掉（`core/components/cycles/list/root.tsx:58`、`core/components/core/list/list-root.tsx:19`）。做法：被裁的是轮廓不是内容——可聚焦元素以内缩的轮廓（`outline-offset` 为负）或在裁剪容器内留出轮廓的空间，不删 `truncate`、`overflow` 的布局作用（截断与滚动照旧）；`outline-none` 换成仓库的 `focus-visible` 轮廓写法——七处统一用仓库已有的内缩轮廓 `focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong`（`issues/select/base.tsx:197`，与 `@nerve/ui` 的 `CustomSearchSelect` 同形），不用 `Button` 的外扩 2px 的轮廓：日期范围按钮有 7 个调用处、任何一处的容器都可能裁掉外扩的轮廓，且它与同页已用内缩轮廓的选择框并排（T5 的裁定）。`e2e/fixtures/module-pages.ts:106` 的 `expectOutlinesWhole` 移到 `e2e/fixtures/browser.ts` 的 `focusOutline` 旁，两域共用，原处的导入改指新处。证明在三种引擎（§2.11）。

### 2.6 冻结任务逐项目继续

`planning/app/freeze_cycles.go` 的 `Execute` 今天在第一个失败的项目处返回：一个违反不变量的项目（时区载不进、状态在五组之外）让 id 在它之后的项目每次重试都轮不到，是活性缺陷（P4 spec §5、re-63 m2）。改为：每个项目照旧一个事务；一个项目失败时记下以项目 id 包裹的错误、接着下一个项目；`ctx` 结束时立即停，返回 `errors.Join(已记下的…, ctx.Err())`，此前的失败不丢；跑完以 `errors.Join` 交回错误与已冻结的数目（River 照旧重试，把每次的错误文字存在任务行上；已提交的项目不重做：冻结过的迭代不再是候选，`progress.sql:31-34` 的 `progress_snapshot IS NULL`）。**交回的错误有上限**：保留前 5 个项目的错误（`errors.Is`/`errors.As` 照样穿过，`LockNotAvailable` 与 `isBusy` 都经 `errors.As`），其余只计数（“另有 N 个项目失败”），失败的项目数精确——连接池中途断了这类会让每个候选都失败的错误不会把成百条错误写进任务行。锁序与“锁后读时钟”不变。注释（`app/freeze_cycles.go` 与 worker 的 `adapter/river/freeze_cycles.go:41-42`）、`README.md:129` 与 [P4 spec](P4-progress-transfer.md) `:104` 的那句随之改写。单测：三个项目中间的一个失败，第一个与第三个都已提交，返回的错误含中间那个的原错误与 id；`ctx` 在第二个之后取消时不再往下、返回的错误含此前的失败与 `ctx.Err()`；七个项目都失败时错误里恰有 5 个原错误与“另有 2 个”。变异：恢复“遇错即返回”、去掉上限，各被单测杀死。既有的 `TestFreezeCyclesKeepsTheProjectsBeforeAFailure`（`planning/app/freeze_cycles_test.go:288` 断言失败之后不再锁下一个）按契约改；`bootstrap/planning_freeze_test.go` 的 `TestFreezeJobLockTimeoutPreservesPostgresError`（`:176`）、`TestFreezeJobPartialCommitAndRetry`（`:227`）不改、照过。M4 的自动归档（`issue/app/auto_archive.go`）同形，属 issue 模块，不在本段改，写入 M8 交接（§2.13）。

### 2.7 权限矩阵的访客一列

照转移的先例（`bootstrap/permission_matrix_transfer_test.go:37-39`：`{op, variant: "guests see all", prepare: preparing(guestsSeeAll, …)}`，`guestsSeeAll` 在 `permission_matrix_counts_test.go:164`）为 P1–P2 的写操作各加一个**变体行**（不是给全部行加一列）：`createCycle`、`updateCycle`、`deleteCycle`、`archiveCycle`、`unarchiveCycle`、`createModule`、`updateModule`、`deleteModule`、`archiveModule`、`unarchiveModule`、`createModuleLink`、`updateModuleLink`、`deleteModuleLink`，共 13 个（`api/modules/planning.yaml` 的写路径）。不加的：`updateCyclePreferences`/`updateModulePreferences` 是 `*Preferences` 的动作、访客本可写（`permission_matrix_cycles_test.go:212-213`）；转移已有；模块成员没有自己的接口，是建立、修改模块的 `member_ids`；P3a 的归属写经 issue 模块的动作，不是 `*.write`。期望同转移的变体：访客 `forbidden`，工作区管理员兼访客照过。已有 prepare 的行组合它（如 `unarchiveCycle` 的 `preparing(guestsSeeAll, cycleArchived)`、`archiveModule` 的 `moduleCompleted`）。变体行放新文件（`permission_matrix_cycles_test.go` 394 行、`permission_matrix_modules_test.go` 351 行）。这样的访客读得到、写不了，把任一写操作的 `*.write` 换成 `*.read` 时矩阵失败。聚焦运行连同覆盖守卫（`-run 'TestPermissionMatrix|Coverage|Matrix|EveryColumnCalls'`）；变异：一个迭代的写、一个模块的写各把动作名换成读，都被杀死。**代价**：写的行每格一份数据库副本（`permission_matrix_test.go:161`），13 行若跑满 14 列约 182 份；变体只需访客与工作区管理员兼访客两列时只跑这两列（矩阵的框架若支持；不支持就跑满）。记下 `TestPermissionMatrix` 改前改后的时长；增加超过 60 秒时停下报告主控（服务端分片的余量属负责人的开放问题）。另补一例归档列表的词：`archived=true` 与 `cycle_id`、`module_id` 筛选、`group_by=module` 合在一起的回答（§2.4）。

### 2.8 规模

- **删除规模的夹具**：`bootstrap/issue_scale_fixture_test.go`（`:77-103`）的规模夹具今天不播种规划的数据，`TestTheDeletionsAtTheScaleTarget` 只看 `issue.delete_rows`（基线 `issue_scale_test.go:62`、`issue_project_deletion_jobs_test.go:55` 的 `kind='issue.delete_rows'`、`issue_scale_test.go:93-102` 的 90 秒等待），走不到 M6 的删除步骤。M6 的删除是另一个 River 任务：`planning/app/delete_rows_on_deletion.go:24` 订阅 `project.deleted`、`workspace.deleted`，入队 `planning.delete_rows`（`{project_id, at, by}`，`planning/adapter/river/delete_rows.go:17-25`），超出预算时 snooze（`:60-62`）。补上：(1) 夹具每个项目播种 8 个迭代、6 个模块（每个模块 3 个成员、2 个链接），每个工作项在一个迭代、两个模块里，`ANALYZE`（`issue_scale_fixture_test.go:109`）加上规划的表；(2) “没有在跑的删除任务”的基线加上 `planning.delete_rows`；(3) 项目一步与工作区一步各找到每个被删项目的一个 `planning.delete_rows` 任务，以 `waitPlanningDeletion`（`bootstrap/planning_deletion_fixture_test.go:93`）等它们完成：断言完成、没有错误，snooze 与批数记下而不禁止（`scaleSingleRun` 的“一次运行、attempt 1、零 snooze”不套在它上面），记下运行秒数；(4) 然后才断言规划的表与 `cycle_issues`、`module_issues`：活的行为 0、总行数不变（软删除，同 `checkScaleRows`；物理删除属 `TestPurgeScale`），并写明 `cycle_issues`、`module_issues` 由哪一个任务删；邻域的项目不受影响。证据（时长、批数、邻域的编辑延迟）写入 `NERVE_SCALE_EVIDENCE` 目录。既有的断言不改；加进来的规划任务若扰动了既有的 90 秒等待或单次运行的断言，停下报告主控，不放宽。
- **冻结的规模**：新增以环境变量打开、默认跳过的规模测试（同 `NERVE_MEASURE_SCALE` 的做法）：多项目、每项目若干已结束的迭代与工作项，量候选查询 `FreezeDueProjects` 的计划与一次运行的总时长、逐项目的锁持有时长。夹具里绝大多数迭代是已冻结或未结束的并 `ANALYZE`，使 `cycles_freeze_due_idx` 有选择性；断言计划用到它——规划器仍选顺序扫描时记下计划、停下报告主控，不悄悄放宽断言。
- **重跑**：`TestPurgeScale`（`NERVE_PURGE_SCALE=1`）、扩展后的删除规模测试（`NERVE_MEASURE_SCALE=1`）与冻结的规模测试在最终代码上各跑一次（§2.12），一次一个容器套件。只改测试，不改产品代码；若规模测试暴露产品的缺陷，按纪律先提交证据再判断（§6）。

### 2.9 死成员

M1 收尾 plan 附录 A 的三个脚本（`docs/v0/M1-frontend-trim/plans/closeout.md:1062`、`:1368`、`:1512`）在最终代码上跑 `--rows M6`，照 M5 P8 T7：前后的 tsv 入台账；删除唯一的真死成员（`fake-module-dialog.tsx:32-33` 的 `invalid`、`describedBy`，没有读写者），删除之后复扫没有新行；产品文件里有读者的行（调研时 24 行：`satisfies` 的筛选表、经 `in` 读、结构类型的服务方法、按字符串键读、展开传递、经 `SPREADSHEET_COLUMNS` 传递的两列）在阶段记录逐行写明文件、名字与读者。M1 交接的 84 行：78 行已消失，6 行是表格两列的假阳性；M3 交接 §2 的 8 行随 P6a `c09d4f8d` 删去。

### 2.10 告警

清 `no-unused-expressions` 一类：全仓 4 条（其一在没有读者的 `@nerve/utils` 的 `router.ts` `generateQueryParams`，连同这个死导出删除），其余以等价的语句改写、行为不变；`.oxlintrc.json` 把这条规则设为 error（全仓生效，含 e2e），各包 `check:lint` 脚本里的 cap 随实际下调（`web/apps/web/package.json:10` 的 44、`web/packages/utils/package.json:18` 的 3；`tools/lint-cap.mjs` 只收 cap 作参数；调研时 web 44→41、utils 3→2，以实测为准）。M6 全期改动的文件（`git diff --diff-filter=AMR ba3fd6e0..HEAD` 的可 lint 文件，`ba3fd6e0` 是 M5 收尾的 noff）oxlint 0 告警的收据（M1 交接的“改到的文件 oxlint 为零”）。

### 2.11 矩阵专用的页面核对

新目录 `e2e/acceptance/`（默认配置的 `testDir` 是 `stories`，`make e2e` 与 CI 跑默认配置，不跑它；`e2e/drills` 与 `playwright.m5-slow.config.ts` 是先例）与新配置 `e2e/playwright.m6-acceptance.config.ts`（继承默认配置，`testDir: "acceptance"`，`projects` 整个换成 chromium、firefox、webkit 三种引擎——不继承默认配置的 `s3-fixture` 项目与 Firefox 的 `grep`；`package.json` 加 `test:m6-acceptance`）。e2e 的 `tsconfig.json` 已含 `**/*.ts`、`check:lint` 在 `.` 上跑 oxlint，新目录自然被覆盖（核对，不改）。帮手从原处导入不复制。核对（标题 `G<n> (acceptance): …`）：

| 核对 | 覆盖 |
|---|---|
| 焦点轮廓（迭代页、模块页） | 逐个 Tab 量 §2.5 的七处与迭代页的 M3 交接 §3 四种共用控件、P7a 的两个下拉框、P7b 的筛选与分组控件和“添加已有”的两个入口、移动端页头（390 × 844）与迭代页的顶行；`expectOutlinesWhole` 断言轮廓完整；以变异脚本恢复修改之前的类名时 (a)、(d)、(e) 与只在 Firefox、WebKit 出现的 (c)、(g) 各在其引擎失败。第一次量到的控件（迭代页的四种、P7a–P7c 的新控件）若轮廓不完整，在本 Task 于根因处修（RED 即这条核对），跨模块的停下报告 |
| 命令面板的行 | cmdk 的行不是 Tab 的停靠点（`PK/ui/modal/command-item.tsx:50-56` 的 `focus:outline-none`，选中是 `aria-selected` 的样式，`styles/power-k.css:14`；焦点留在 `Command.Input`，它也是 `outline-none`，`PK/ui/modal/header.tsx:49`）：以方向键选到的行 `aria-selected`、有可见的高亮、整行在列表的滚动区之内；输入框的 `outline-none` 照 Plane 保留（面板本身就是焦点所在的明示），不用 `expectOutlinesWhole` 量行 |
| 表单里的项目选择 | 迭代、模块的新建表单里换项目（`cycles/form.tsx:91-105`、`modules/form.tsx:118` 的 `dropdowns/project/base.tsx`，M2 交接 §5(d)）：键盘打开、搜索、选中，表单随之换项目 |
| 归档页的迭代、模块词 | 归档页按迭代筛选、按模块分组，请求的查询与组恰如所选；开关关闭时未选的不给 |
| 模块的开关（G18） | 关掉模块：侧栏与命令面板没有入口，按地址读模块页照旧可读，工作项的模块列与筛选按 P7b R8；再打开，数据完好 |

本机三种引擎跑，CI 不跑；时长不设上限，但每条写明。

### 2.12 最终矩阵

在最终产品候选上运行（收据经 `run-check.py`，原件入台账）：`make build`；默认套件加 JSON 报告；`test:m6-pages`（45 条 × 三种引擎）；`test:m6-acceptance`；间歇项隔离重复至少 5 次（M6-M8-13 的 G11 页面在 WebKit、G2 页面、M6-M8-03、M6-M8-07、M6-M8-01 的 C7；其中会撞上外因的写明外因）；Go 全量 `GOFLAGS='-p=1 -v' make test`；矩阵的聚焦运行（§2.7）；三个规模测试（§2.8）。[阶段记录](../reviews/P8-final-acceptance-review.md)每个 G 一行：接口与数据库的证据（Go 用例名与本次的通过数）、页面的证据（故事、引擎与本次的结果），每条指向本次运行的原件。G14、G16、G17、G20 没有带编号的页面故事：G16（自动归档保留开着的迭代、模块的工作项）与 G17（权限与 OwnOnly）以 Go 为主，页面一侧指向已有故事里的断言（G5、G7、G12、G13、G19 的访客与 OwnOnly），G14、G20 写明它们的证据所在；不为凑编号写重复的故事。每行记下证据所在的 HEAD；修复轮之后，自 T12 以来只改了文档或无关测试的行沿用 T12 的原件，其余的行在最终候选上重跑（规模、隔离重复与 `-v` 的 Go 全量不在 `final.sh` 里，要单独重跑）。

**设计 §13 的对照**：§13 的 14 条（`M6-design.md:257-270`）逐条对到本次运行的收据或写明“closeout”，closeout 只勾选：除 G1–G20 外，“Go 与前端同 fixture”（列表引擎的顺序测试）、“无 Plane 的默认 PUT 或跨项目漏洞”、“前端只调 `/api/v0`、旧 service/类型/地址删除”（`old-cycle-urls`、`old-module-urls` 等关键词规则）、“无到期例外”、“进度口径每域一个函数”各有本次的收据。H3（开关关闭时存下的词保留“—”与“不可用的选项”）的理由写进阶段记录。

### 2.13 文档与交接

- **文档**：`README.md`（`:69` 的“迭代与模块待接”改为 M6 一条、e2e 一节加 `test:m6-pages` 与 `test:m6-acceptance`、`:133` 的 `planning.delete_rows` 改为迭代与模块、`:191` 一节加模块链接的抓取 `planning.crawl_link`），`server/configs/config.yaml:120` 的注释，`frontend-changes.md`（`:205`、`:214`、`:329`），`plane-diff.md`（§2.4 的一行），设计的文件头与 §14。文档链接零断（`doc-links.py`）。
- **七份输入交接**（M1-P2、M1-P3、M1、M2、M3、M4、M5 的收尾交接）逐条写“处理结果（M6/P8）”：完成的指向证据，转交的写明去处；`status` 不改（closeout）。M5-M6-01 以“未触发”结束（M6 没有成为资产宿主），条件原文转交 M7。
- **M7 的 `M6-closeout`**（今天只有 M6-M7-01）补：收藏（含归档时移除）；工作区级的迭代、模块列表与视图的筛选、跨项目的行与工作区范围的命令面板搜索；视图的 store、读与页面及 M3 的视图遗留（`projectViewIds`、`views/(list)/mobile-header.tsx` 的 `projectMemberIds`、视图的文案、`views/quick-actions.tsx` 嵌套的按钮）；`getTabIndex` 与两份 `tab-indices.ts`、桶文件、`IDropdownProps.tabIndex` 的最后删除（读者只剩 M7 的文件）；转移不发布 `issue.updated`（`issue/app/cycle_transfer.go:73`）；访客看得到视图与收件箱的命令面板命令；M5-04 的原文与 M6 的结论（M6 没有加入队的命令行）；M5-M6-01 的条件。
- **M8 的 `M6-closeout`**（M6-M8-01…15，-08 已满足）补：G2 页面的上海午夜窗口（P5a §5）；默认 CI 不跑的对等与发布时的运行（m6-pages、m6-acceptance、三个规模测试）；自动归档遇到失败的项目即停（§2.6 的同形）；访客在命令面板看到项目设置的命令（M3 的代码）；各条补本段的数据点。

### 2.14 门禁与 CI

本机七项（`gates.sh`）加 `test:m6-pages` 与 `test:m6-acceptance`；有产品变化的检查点先过七项再推送；默认套件的条数照旧 480（§3 第 1 条，台账核对）；分支 CI 与 noff 之后的 main CI 分开记录（含环境失败与测试失败的区分）。已知的间歇（M6-M8-01、-03、-07）照既有规则至多一次只重跑失败的文件或作业并记录；估算、上界或实测的 e2e 超过 30 分钟时停下报告负责人。web 作业（期限 15 分钟，P7c 时 902 秒碰到过一次）与服务端两路（期限 25 分钟）每次 CI 记下时长：web 作业再被期限取消时按 P7c 的先例（已知的环境原因）只重跑失败的作业一次并在阶段报告里告诉负责人；服务端任一路超过 20 分钟时报告负责人。

## 3. 局部裁定

主控的裁定 H1–H16 见台账 `controller-rulings.md`，要点：

1. **默认套件不加故事**（H1）：矩阵专用的核对放 `e2e/acceptance/`，CI 的 e2e 时长不因 P8 增加；网页单测只加必要的几条（web 作业同样紧）。
2. **归档页给迭代、模块词**（H2）：Plane 同；服务端已支持。
3. **开关关闭时存下的词**保留“—”与“不可用的选项”（H3）：Plane 在开关关闭时不显示这些词，“不可用的选项”是它对叫不出的值的通用说法；领域专门的说法要加 Plane 没有的键，收益小。
4. **冻结任务逐项目继续**（H4）；自动归档的同形交 M8（跨模块）。
5. **规模只改测试**（H5、H6）。
6. **访客看到的项目设置、视图、收件箱命令不在 P8 改**（H7）：不是 M6 的领域；视图、收件箱交 M7，项目设置交 M8。
7. **M5-M6-01 以未触发关闭、条件转交 M7**（H8）。
8. **焦点轮廓在根因处修，含 `@nerve/ui` 的面包屑**（H9）：只改轮廓，不改截断与滚动。
9. **其余**照 H10–H16。

## 4. 验收标准

- [ ] A1 P7c 阶段记录 §7 回填、spec A14 勾选；设计与 `frontend-changes.md` 的状态照实；M6-M8-01 的数据点；phase `M6/P8`（§2.1）。
- [ ] A2 三个菜单以 id 作 value、名字作 keywords，同名与改名的单测（§2.2）。
- [ ] A3 三处英文字面量经键（en、zh-CN），表格的 `colSpan` 与表头同源（§2.3）。
- [ ] A4 归档页的迭代、模块筛选与分组，开关关闭时同项目页；`plane-diff.md` 改写（§2.4）。
- [ ] A5 七处焦点轮廓在三种引擎完整，`expectOutlinesWhole` 在 `browser.ts`（§2.5、§2.11）。
- [ ] A6 冻结任务逐项目继续，单测与变异（§2.6）。
- [ ] A7 权限矩阵的访客一列与变异（§2.7）。
- [ ] A8 删除规模的夹具含规划的数据、冻结的规模测试；三个规模测试的本次证据（§2.8）。
- [ ] A9 死成员 `--rows M6` 的前后与读者表，真死成员删除（§2.9）。
- [ ] A10 `no-unused-expressions` 为零且为 error，cap 等于实际；M6 全期改动文件 oxlint 0 的收据（§2.10）。
- [ ] A11 `e2e/acceptance/` 与 `m6-acceptance` 配置的五类核对在三种引擎通过（§2.11）。
- [ ] A12 G1–G20 的最终证据表与设计 §13 的对照，每条指向本次运行的原件；间歇项有重复运行的结论（§2.12）。
- [ ] A13 文档同步、七份输入交接的最终处理结果、M7 与 M8 的 `M6-closeout` 补全（§2.13）。
- [ ] A14 七项门禁、页面配置与验收配置、一次整分支 deep-review 与修复闭环、分支 CI、noff 之后的 main CI、归档与清理（§2.14）；由 closeout 的第一个 Task 回填阶段记录 §7 时勾选。

## 5. 后续责任

**closeout**（设计 §12 的最后一行）：首次 noff/main 绿 → 设计 §13 逐条勾选与总体 §9.4 → 第二次 noff/main 绿 → 归档清理；七份输入交接与 M7、M8 的交接改 `status`；它的第一个 Task 回填本段阶段记录的 §7 与 A14。**M7**、**M8**：见 §2.13。**负责人**：CI 的 web 作业 15 分钟的期限（慢的 runner 上约 3% 的余量）与 e2e 的作业结构。

## 6. 风险

- **焦点轮廓的改动面**（§2.5）：面包屑是各页共用的件，改坏会影响所有页；只改轮廓的类，以三种引擎的核对与已有的页面故事（默认套件）同时证明截断与滚动照旧。
- **冻结的错误语义**（§2.6）：交回的错误变成多个的合并；River 的重试与日志照旧，测试钉住“已提交的不重做”。
- **规模测试的耗时与环境**（§2.8）：一次一个容器套件；环境停顿按同输入重跑并保留原件；若暴露产品的缺陷，先提交证据，属本地的就在根因处修，跨模块或高风险的停下报告负责人。
- **矩阵不进 CI**：`e2e/acceptance/` 与 m6-pages 只在本机跑，回归要等发布时的运行（交 M8）；这是负责人决定 e2e 的作业结构之前的权宜，阶段记录写明。
