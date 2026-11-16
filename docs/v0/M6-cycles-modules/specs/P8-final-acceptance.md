# M6 / P8：最终验收——最终矩阵、规模重跑、遗留收敛与交接

> 状态：spec 与 plan 已写（2026-11-16），待预检。基线 main `ea971606b829fb3ade4beb9d204ecbc23d3af674`（P7c 的 noff；main CI 43764306705 attempt 2 全绿）。

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

- `core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx:219` 的 “Restore” 改读已有的 `common.actions.restore`；同文件 `:53-54` 的两条提示（“Action not available”等）同类，改为键（en、zh-CN）。
- `PK/ui/renderer/shortcut.tsx:105` 的 “No shortcuts found for” 改为新键（en、zh-CN），插值照旧。
- 项目关着迭代（模块）时，表格的合并单元格与加载行仍按迭代、模块两列计数（`issue-layouts/spreadsheet/spreadsheet-table.tsx:110` 对 `spreadsheet-view.tsx:66`），宽出一两列：改为与表头同一个列的来源。单测：开关关时 `colSpan` 等于表头的列数。

### 2.4 归档页的迭代、模块词

给。Plane 的归档页给 `cycle_id`、`module_id` 的筛选与按 `cycle`、`modules` 分组（plane `packages/constants/src/issue/filter.ts:142-160`）；nerve 的归档列表是同一个 `GET /api/v0/projects/{project_id}/issues?archived=true`，筛选与分组的词同项目页（`api/modules/issue.yaml:381` 起），服务端不改。`@nerve/constants` 的 `archived_issues`（`web/packages/constants/src/issue/filter.ts:158-181`）加两个筛选与两个分组；归档页的本地偏好照收这四个词。归档页只读：没有拖放、快速新建与“添加已有”，拒收的列不适用；名字经 `usePlanningNames`（已归档的带说明）；开关关闭时同项目页（P7b R8、R17：不给未选的分组与筛选，已选的留着可去掉，组头 “—”，条件的值“不可用的选项”）。`plane-diff.md` 的“归档页的迭代、模块词（M6/P7b）”一行改写为本段。单测：归档页的词表与开关关闭时的面板；页面核对见 §2.11。

### 2.5 共用控件的焦点轮廓

P6a 交下的七处在根因处修（[P6a spec](P6a-web-modules.md) §5 的 P8 一条，F2 的探针）：(a) 行的链接被 `ListItem` 的 `truncate` 容器裁掉（`core/components/core/list/list-item.tsx:69-71`，迭代与模块的行共用）；(b) 侧栏链接一行的链接（`core/components/modules/links/list-item.tsx:56`、`:65`）；(c) “Add link”在 Firefox、WebKit 被链接区的 `overflow-y-auto` 裁掉（`core/components/modules/progress-sidebar/sidebar-links.tsx:83-90`、`:112`）；(d) 面包屑的链接被 `overflow-hidden` 裁掉（`@nerve/ui` `src/breadcrumbs/breadcrumbs.tsx:46`、`:132`，M3 的共用件，各页都有）；(e) `DateRangeDropdown` 的 `outline-none`（`core/components/dropdowns/date-range.tsx:123`）；(f) `ProgressTabs` 的 `outline-none`（`core/components/core/sidebar/progress-tabs.tsx:114`）；(g) 迭代列表的分节按钮在 WebKit 被列表的滚动容器裁掉（`core/components/cycles/list/root.tsx:58`、`core/components/core/list/list-root.tsx:19`）。做法：被裁的是轮廓不是内容——可聚焦元素以内缩的轮廓（`outline-offset` 为负）或在裁剪容器内留出轮廓的空间，不删 `truncate`、`overflow` 的布局作用（截断与滚动照旧）；`outline-none` 换成仓库的 `focus-visible` 轮廓写法（同 M3 交接 §3 已核对的 `Button`、`IconButton`）。`e2e/fixtures/module-pages.ts:106` 的 `expectOutlinesWhole` 移到 `e2e/fixtures/browser.ts` 的 `focusOutline` 旁，两域共用，原处的导入改指新处。证明在三种引擎（§2.11）。

### 2.6 冻结任务逐项目继续

`planning/app/freeze_cycles.go` 的 `Execute` 今天在第一个失败的项目处返回：一个违反不变量的项目（时区载不进、状态在五组之外）让 id 在它之后的项目每次重试都轮不到，是活性缺陷（P4 spec §5、re-63 m2）。改为：每个项目照旧一个事务；一个项目失败时记下以项目 id 包裹的错误、接着下一个项目；`ctx` 结束时立即停并返回；跑完以 `errors.Join` 交回全部错误与已冻结的数目（River 照旧重试并记录，已提交的项目不重做：冻结过的迭代不再是候选）。注释与 P4 spec §2.5 的那句随之改写。单测：三个项目中间的一个失败，第一个与第三个都已提交，返回的错误含中间那个的原错误与 id；`ctx` 取消时不再往下。变异：恢复“遇错即返回”被单测杀死。M4 的自动归档（`issue/app/auto_archive.go`）同形，属 issue 模块，不在本段改，写入 M8 交接（§2.13）。

### 2.7 权限矩阵的访客一列

迭代、模块、模块链接与模块成员的全部写操作（P1–P3a）在权限矩阵补“看得到全部工作项的访客”一列（今天只有转移有，`bootstrap/permission_matrix_transfer_test.go:37`）：这样的访客读得到、写不了，把任一写操作的 `*.write` 换成 `*.read` 时矩阵失败。聚焦运行连同覆盖守卫（`-run 'TestPermissionMatrix|Coverage|Matrix|EveryColumnCalls'`）；变异：一个迭代的写、一个模块的写各把动作名换成读，都被杀死。

### 2.8 规模

- **删除规模的夹具**：`bootstrap/issue_scale_fixture_test.go`（`:81-102`）的规模夹具今天不播种规划的数据，`TestTheDeletionsAtTheScaleTarget` 因此走不到 M6 的删除步骤。补上：每个项目的迭代、模块、模块成员、模块链接与工作项的 `cycle_issues`、`module_issues` 行（数目在 plan 里定，与工作项的规模同一量级的比例），断言项目、工作区删除与 `planning.delete_rows` 之后这些表的行精确归零、邻域的项目不受影响，证据（时长、批数、邻域的编辑延迟）写入 `NERVE_SCALE_EVIDENCE` 目录。
- **冻结的规模**：新增以环境变量打开、默认跳过的规模测试（同 `NERVE_MEASURE_SCALE` 的做法）：多项目、每项目若干已结束的迭代与工作项，量候选查询 `FreezeDueProjects` 的计划（用到部分索引）与一次运行的总时长、逐项目的锁持有时长。
- **重跑**：`TestPurgeScale`（`NERVE_PURGE_SCALE=1`）、扩展后的删除规模测试（`NERVE_MEASURE_SCALE=1`）与冻结的规模测试在最终代码上各跑一次（§2.12），一次一个容器套件。只改测试，不改产品代码；若规模测试暴露产品的缺陷，按纪律先提交证据再判断（§6）。

### 2.9 死成员

M1 收尾 plan 附录 A 的三个脚本（`docs/v0/M1-frontend-trim/plans/closeout.md:1062`、`:1368`、`:1512`）在最终代码上跑 `--rows M6`，照 M5 P8 T7：前后的 tsv 入台账；删除唯一的真死成员（`fake-module-dialog.tsx:32-33` 的 `invalid`、`describedBy`，没有读写者），删除之后复扫没有新行；产品文件里有读者的行（调研时 24 行：`satisfies` 的筛选表、经 `in` 读、结构类型的服务方法、按字符串键读、展开传递、经 `SPREADSHEET_COLUMNS` 传递的两列）在阶段记录逐行写明文件、名字与读者。M1 交接的 84 行：78 行已消失，6 行是表格两列的假阳性；M3 交接 §2 的 8 行随 P6a `c09d4f8d` 删去。

### 2.10 告警

清 `no-unused-expressions` 一类：全仓 4 条（其一在没有读者的 `@nerve/utils` 的 `router.ts` `generateQueryParams`，连同这个死导出删除），其余以等价的语句改写、行为不变；`.oxlintrc.json` 把这条规则设为 error，`tools/lint-cap.mjs` 的 cap 随实际下调（调研时 web 44→41、utils 3→2，以实测为准）。M6 全期改动的文件（`git diff --diff-filter=AMR ba3fd6e0..HEAD` 的可 lint 文件，`ba3fd6e0` 是 M5 收尾的 noff）oxlint 0 告警的收据（M1 交接的“改到的文件 oxlint 为零”）。

### 2.11 矩阵专用的页面核对

新目录 `e2e/acceptance/`（默认配置的 `testDir` 是 `stories`，不跑它）与新配置 `e2e/playwright.m6-acceptance.config.ts`（继承默认配置，`testDir: "acceptance"`，chromium、firefox、webkit 三种引擎，`package.json` 加 `test:m6-acceptance`），e2e 的 lint 与类型检查覆盖新目录。帮手从原处导入不复制。核对（标题 `G<n> (acceptance): …`）：

| 核对 | 覆盖 |
|---|---|
| 焦点轮廓（迭代页、模块页） | 逐个 Tab 量 §2.5 的七处与迭代页的 M3 交接 §3 四种共用控件、P7a 的两个下拉框、P7b 的筛选与分组控件和“添加已有”的两个入口、P7c 的命令面板菜单行、移动端页头（390 × 844）与迭代页的顶行；`expectOutlinesWhole` 断言轮廓完整；以变异脚本恢复修改之前的类名时至少 (a)、(d)、(e) 三处失败 |
| 表单里的项目选择 | 迭代、模块的新建表单在工作区级入口打开时的项目选择（`dropdowns/project/base.tsx`，M2 交接 §5(d)）：键盘打开、搜索、选中，表单随之换项目 |
| 归档页的迭代、模块词 | 归档页按迭代筛选、按模块分组，请求的查询与组恰如所选；开关关闭时未选的不给 |
| 模块的开关（G18） | 关掉模块：侧栏与命令面板没有入口，按地址读模块页照旧可读，工作项的模块列与筛选按 P7b R8；再打开，数据完好 |

本机三种引擎跑，CI 不跑；时长不设上限，但每条写明。

### 2.12 最终矩阵

在最终产品候选上运行（收据经 `run-check.py`，原件入台账）：`make build`；默认套件加 JSON 报告；`test:m6-pages`（45 条 × 三种引擎）；`test:m6-acceptance`；间歇项隔离重复至少 5 次（M6-M8-13 的 G11 页面在 WebKit、G2 页面、M6-M8-03、M6-M8-07、M6-M8-01 的 C7；其中会撞上外因的写明外因）；Go 全量 `GOFLAGS='-p=1 -v' make test`；矩阵的聚焦运行（§2.7）；三个规模测试（§2.8）。[阶段记录](../reviews/P8-final-acceptance-review.md)每个 G 一行：接口与数据库的证据（Go 用例名与本次的通过数）、页面的证据（故事、引擎与本次的结果），每条指向本次运行的原件。G14、G16、G17、G20 没有带编号的页面故事：G16（自动归档保留开着的迭代、模块的工作项）与 G17（权限与 OwnOnly）以 Go 为主，页面一侧指向已有故事里的断言（G5、G7、G12、G13、G19 的访客与 OwnOnly），G14、G20 写明它们的证据所在；不为凑编号写重复的故事。

### 2.13 文档与交接

- **文档**：`README.md`（`:69` 的“迭代与模块待接”改为 M6 一条、e2e 一节加 `test:m6-pages` 与 `test:m6-acceptance`、`:133` 的 `planning.delete_rows` 改为迭代与模块、`:191` 一节加模块链接的抓取 `planning.crawl_link`），`server/configs/config.yaml:120` 的注释，`frontend-changes.md`（`:205`、`:214`、`:329`），`plane-diff.md`（§2.4 的一行），设计的文件头与 §14。文档链接零断（`doc-links.py`）。
- **七份输入交接**（M1-P2、M1-P3、M1、M2、M3、M4、M5 的收尾交接）逐条写“处理结果（M6/P8）”：完成的指向证据，转交的写明去处；`status` 不改（closeout）。M5-M6-01 以“未触发”结束（M6 没有成为资产宿主），条件原文转交 M7。
- **M7 的 `M6-closeout`**（今天只有 M6-M7-01）补：收藏（含归档时移除）；工作区级的迭代、模块列表与视图的筛选、跨项目的行与工作区范围的命令面板搜索；视图的 store、读与页面及 M3 的视图遗留（`projectViewIds`、`views/(list)/mobile-header.tsx` 的 `projectMemberIds`、视图的文案、`views/quick-actions.tsx` 嵌套的按钮）；`getTabIndex` 与两份 `tab-indices.ts`、桶文件、`IDropdownProps.tabIndex` 的最后删除（读者只剩 M7 的文件）；转移不发布 `issue.updated`（`issue/app/cycle_transfer.go:73`）；访客看得到视图与收件箱的命令面板命令；M5-04 的原文与 M6 的结论（M6 没有加入队的命令行）；M5-M6-01 的条件。
- **M8 的 `M6-closeout`**（M6-M8-01…15，-08 已满足）补：G2 页面的上海午夜窗口（P5a §5）；默认 CI 不跑的对等与发布时的运行（m6-pages、m6-acceptance、三个规模测试）；自动归档遇到失败的项目即停（§2.6 的同形）；访客在命令面板看到项目设置的命令（M3 的代码）；各条补本段的数据点。

### 2.14 门禁与 CI

本机七项（`gates.sh`）加 `test:m6-pages` 与 `test:m6-acceptance`；有产品变化的检查点先过七项再推送；默认套件的条数照旧 480（§3 第 1 条，台账核对）；分支 CI 与 noff 之后的 main CI 分开记录（含环境失败与测试失败的区分）。已知的间歇（M6-M8-01、-03、-07）照既有规则至多一次只重跑失败的文件或作业并记录；估算、上界或实测的 e2e 超过 30 分钟时停下报告负责人。

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
- [ ] A11 `e2e/acceptance/` 与 `m6-acceptance` 配置的四类核对在三种引擎通过（§2.11）。
- [ ] A12 G1–G20 的最终证据表，每条指向本次运行的原件；间歇项有重复运行的结论（§2.12）。
- [ ] A13 文档同步、七份输入交接的最终处理结果、M7 与 M8 的 `M6-closeout` 补全（§2.13）。
- [ ] A14 七项门禁、页面配置与验收配置、一次整分支 deep-review 与修复闭环、分支 CI、noff 之后的 main CI、归档与清理（§2.14）；由 closeout 的第一个 Task 回填阶段记录 §7 时勾选。

## 5. 后续责任

**closeout**（设计 §12 的最后一行）：首次 noff/main 绿 → 设计 §13 逐条勾选与总体 §9.4 → 第二次 noff/main 绿 → 归档清理；七份输入交接与 M7、M8 的交接改 `status`；它的第一个 Task 回填本段阶段记录的 §7 与 A14。**M7**、**M8**：见 §2.13。**负责人**：CI 的 web 作业 15 分钟的期限（慢的 runner 上约 3% 的余量）与 e2e 的作业结构。

## 6. 风险

- **焦点轮廓的改动面**（§2.5）：面包屑是各页共用的件，改坏会影响所有页；只改轮廓的类，以三种引擎的核对与已有的页面故事（默认套件）同时证明截断与滚动照旧。
- **冻结的错误语义**（§2.6）：交回的错误变成多个的合并；River 的重试与日志照旧，测试钉住“已提交的不重做”。
- **规模测试的耗时与环境**（§2.8）：一次一个容器套件；环境停顿按同输入重跑并保留原件；若暴露产品的缺陷，先提交证据，属本地的就在根因处修，跨模块或高风险的停下报告负责人。
- **矩阵不进 CI**：`e2e/acceptance/` 与 m6-pages 只在本机跑，回归要等发布时的运行（交 M8）；这是负责人决定 e2e 的作业结构之前的权宜，阶段记录写明。
