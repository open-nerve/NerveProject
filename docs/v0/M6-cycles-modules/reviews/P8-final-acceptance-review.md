# M6/P8 阶段评审记录

> 进行中（2026-11-16）：T1–T15 的本机部分已完成（T9 没有可删的真死成员，没有提交）。检查点 `d94a0999` 的本机七项（`t11-gate`）与分支 CI 43831815008 全绿；T12 的最终矩阵见 §4.3；整分支审查 re-79 与修复轮、限定复审见 §6；最终候选 `0ef66139` 的 `final.sh t15-final` 七项（e2e 一条以 M6-M8-03 的签名失败、只重跑该文件即过）、m6-pages 与 m6-acceptance 全过（§5）。最终候选的分支 CI、合入、main CI、归档与清理（T16）的数字由 closeout 的第一个 Task 回填 §5 与 §7。基线 main `ea971606b829fb3ade4beb9d204ecbc23d3af674`（P7c 的 noff）。

## 1. 评审方式与范围

P8 是 M6 最后一个编号阶段（[M6 设计](../M6-design.md) §12 的 P8 一行，[spec](../specs/P8-final-acceptance.md) §1）：回填 P7c；网页的遗留（命令面板其余三个菜单的 cmdk value、三处英文字面量与表格的列宽、归档页的迭代与模块词、七处共用控件的焦点轮廓）；服务端（冻结任务逐项目继续、权限矩阵的访客变体行、删除规模的夹具补上规划的数据、冻结的规模测试）；死成员与告警的收敛；矩阵专用的页面核对与 G1–G20 的最终矩阵；文档、七份输入交接的处理结果与给 M7、M8 的交接。

- **调研**：三个只读调研代理在 `ea971606` 上写出笔记（台账 `research/p8-inbound.md`：交给 P8 的全部事项；`research/g-matrix.md`：G1–G20 的现有证据、规模测试与 M5 的最终矩阵的跑法；`research/cleanup-lint-docs.md`：死成员、告警、文档与到期例外），随台账归档。
- **界定**：主控在起草 spec 之前裁定 H1–H16（台账 `controller-rulings.md`，要点见 §3），写出 [spec](../specs/P8-final-acceptance.md)（A1–A14）、16 个 Task 的 [plan](../plans/P8-final-acceptance.md) 与本记录的骨架（`1e85b5d284d5a3aedff269d69b7323f7ca9b95eb`）。
- **预检**：一次 fresh 只读预检 [re-78](../../../reviews/re-78-M6P8预检.md)（`fable`），结论 Ready after fixes：六处 Important——I1 删除规模要等 `planning.delete_rows`、I2 命令面板的行另立一类核对、I3 T5 的探针改三种引擎且检查点移到 T11 的末尾、I4 合并错误的上限与 `ctx` 结束时的返回、I5 矩阵用变体行（13 个写、四类排除、时长的停止线）、I6 设计 §13 的 14 条要有对照——与十一处 Minor，主控除 m8 外全部接受，改入 spec 与 plan（`65f20a7b6427587dabc9b804b522861c820532ea`；设计 §14 的 P8 一行链接 re-78，`a0d82965fa995de9ef68409ca5f96a7572965c77`）。T1 在预检进行时完成（只改文档与关键词的 phase）。
- **实施**：T1–T11、T13 由实现代理完成，主控逐 Task 复核并记台账（`Task N: 主控` 12 行）；T12 由主控以台账的 `t12.sh` 串行运行（一次一个容器套件，失败也接着跑、各留收据）；本记录由写手据台账、收据与日志写出。每次检查经 `run-check.py`（收据记命令、起始时刻、秒数、exit、结束时的 HEAD 与运行前后的输入是否相同），变异经 `mutate.py`（还原后核 sha256，`restored=yes`）。检查点在 T11 的末尾（`gates.sh t11-gate`）。默认套件不加故事（H1），矩阵专用的核对放在默认配置不跑的 `e2e/acceptance/`。

## 2. 验收状态

A1–A13 满足，[spec §4](../specs/P8-final-acceptance.md#4-验收标准)由本提交勾选；A14（七项门禁、整分支审查、分支 CI、main CI、归档与清理）由 closeout 的第一个 Task 回填 §7 时勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | P7c 记录 §7、P7c spec 的 A14 与状态行、P7c plan、设计的抬头与 §14、`frontend-changes.md:205`、M6-M8-01 的三条数据；phase `M6/P8`；原件逐项核对（`t1-*`） | T1 |
| A2 | 三个菜单以 id 作 cmdk 的 value、名字作 keywords；同名两项都能选到、改名之后按新名找到（`PK/menus/search.test.tsx` 8 例）；变异 2 个 | T2 |
| A3 | 三处英文字面量与表格表头的 “Work items” 经键（en、zh-CN）；表格的合并单元格与加载行与表头同源，`getDisplayPropertiesCount` 删去；变异 7 个 | T3 |
| A4 | 归档页的迭代、模块筛选与分组，开关关闭时同项目页；`plane-diff.md` 的一行改写；页面核对 G15 (acceptance) | T4、T11 |
| A5 | 七处 (a)–(g) 与 T11 第一次量到的六个控件的焦点轮廓在三种引擎完整（`t11-acceptance-final`、`t12-acceptance`），`expectOutlinesWhole` 在 `e2e/fixtures/browser.ts` | T5、T11 |
| A6 | 冻结任务逐项目继续、交回有上限的合并错误；单测 3 例与契约变更 1 处；变异 4 个 | T6 |
| A7 | 13 个写的“访客看得到全部”变体行 182 格与守卫；两个变异各杀死对应的格；归档列表的词一例 | T7 |
| A8 | 删除规模的夹具含规划的数据并等 `planning.delete_rows`，新的冻结规模测试；三个规模测试本次的证据（`t12-purge-scale`、`t12-delete-scale`、`t12-freeze-scale`，§4.3） | T8、T12 |
| A9 | `--rows M6` 前后都是 245 行、tsv 逐字节相同；没有真死成员（`fake-module-dialog.tsx:32-33` 有读者，两个探针）；读者表见 §4.2 | T9 |
| A10 | `no-unused-expressions` 清零并设为 error（全仓含 e2e），上限 web 40、utils 2 等于实际；M6 全期改动文件 oxlint 0（`t10-oxlint-m6`） | T10 |
| A11 | `e2e/acceptance/` 与 `playwright.m6-acceptance.config.ts`：五类核对 8 条 × 三种引擎 = 24 例通过（`t11-acceptance-final`，本次 `t12-acceptance`） | T11、T12 |
| A12 | G1–G20 的证据表、设计 §13 的对照、间歇项的重复运行与规模的数字（§4.3），各指向本次运行的原件与 HEAD | T12（本提交） |
| A13 | README、`config.yaml:120` 的注释、`frontend-changes.md`、设计；七份输入交接的处理结果；M7、M8 的 `M6-closeout` | T13（`dac255a9`、`de1e8881`） |
| A14 | 由 closeout 的第一个 Task 回填 §7 时勾选 | T14–T16 |

## 3. 局部裁定

**主控的裁定 H1–H16**（台账 `controller-rulings.md`；spec §3 照录要点）：

- H1 边界：不加接口、表、依赖与权限事实来源；CI 的预算、期限与作业结构不改；默认套件不加故事，矩阵专用的页面核对放在默认配置不跑的文件里；网页单测只加必要的几条。
- H2 归档页给迭代、模块词（Plane 同；归档列表与项目页同一个接口与解析，服务端不改）；H3 开关关闭时存下的词照旧（理由见下）；H4 冻结任务逐项目继续，自动归档的同形交 M8；H5 冻结的规模测试（以环境变量打开）；H6 删除规模的夹具补上规划的数据（只改测试）。
- H7 访客看到的项目设置、视图、收集箱命令不在 P8 改（视图、收集箱交 M7，项目设置交 M8）；H8 M5-M6-01 以未触发关闭、条件转交 M7；H9 焦点轮廓在根因处修，只改轮廓、不改截断与滚动。
- H10 命令面板其余三个菜单以 id 作 value；H11 英文字面量；H12 表格的列宽；H13 权限矩阵的访客；H14 死成员与告警的一类；H15 最终矩阵（G14、G16、G17、G18、G20 没有带编号的故事：G18 补模块开关的页面核对，其余写明证据所在，不为凑编号写重复的故事）；H16 交接。

**H3 的理由**（spec §3 第 3 条、§2.12；预检 re-78 m9 要它有落点）：P7b 留给 P8 的问题是开关关闭之后存下的词——分组的组头 “—”、筛选条件的值“不可用的选项”（[P7b spec](../specs/P7b-list-vocabulary.md) R8 与 §5 的 P8 一条）——要不要更好的说法。裁定照旧：Plane 在开关关着时根本不显示这些词（P7b R8 所引 Plane 的 `display-filters-selection.tsx:53-59` 在开关关着时藏起选项），没有可照的说法；“不可用的选项”是 Plane 对叫不出的值的通用说法，换成领域专门的说法要加 Plane 没有的键，收益小。P8 的归档页沿用同一规则（§2.4）。

**各 Task 的裁定**：台账 68 条 `Ruling:`（T1 4、T2 3、T3 10、T4 5、T5 4、T6 5、T7 7、T8 11、T9 3、T10 4、T11 8、T13 4；T12 没有）与 7 行契约变更，主控复核的 12 行；都是局部取舍，没有架构待决项，也没有新的须负责人决定的事（已知的开放问题照旧：CI 的 web 作业 15 分钟的期限与 e2e 的作业结构，spec §5）。主控在复核时裁定或接受的：

- **T3**：① 断言 “Restore” 的六处改为键，接受（plan 回归边界 T3 一条的前提不成立：这三个文件以 fake-i18n 显示键）；② 不接受实现代理新建的 `core/lib/fake-i18n-locales.ts`（第二套 locale 查找与占位符填充，可能与真 ICU 分叉），两个新测试改回仓库惯例 `fake-i18n-values`；③ 追加表格表头的 “Work items” 改读键，RED、GREEN 与变异照做；④ 真 `@nerve/i18n` 在 vitest 下不做 ICU 插值，只作台账的观察。
- **T4**：归档页不加列表的读，直接打开、店里没有两份列表的归档页“添加筛选”不给这两项（同 Plane 与归档页的状态、标签、成员筛选，M4 P13“冷页面可无候选”），主控知悉；T11 的核对经项目的页进入。
- **T5**：(e)(f) 用仓库的内缩环而不是 `Button` 的外扩环，接受并改 spec §2.5（`478da8679a3036998460a53f02d221420a6315d8`）；`dropdowns/date.tsx:94` 的同形入 T11 的轮廓核对。
- **T6**：接受计数行的措辞 “and N more of the projects failed”、因 `ctx` 失败的项目照记为失败、Go 的新测试文件不加版权头（同 `server/` 的 1942 个 Go 文件）；P4 spec §5 `:199` 的处理结果写进本记录（§4.3 的规模）、自动归档的同形入 M8（M6-M8-18）。
- **T7**：接受多加的守卫 `TestEachPlanningWriteHasAGuestsSeeAllMatrixRow`（既有的覆盖守卫看不出丢掉的变体行）；接口的分组键是 `module_id`、网页的 `module` 映射为它，spec 与 plan 的措辞随后改（T13、`de1e8881`）。
- **T8**：年轻部署（冻结的迭代少）时候选查询的规划器估计偏高、可能选顺序扫描——那时表小、每小时一次，不是性能问题；根治要扩展统计的迁移（产品变化），写入 M8 作观察（M6-M8-20）；正式运行若翻成顺序扫描即停下报告（本次没有，§4.3）。
- **T9**：接受“不删”，spec §2.9、A9 与 plan T9 照实改写（`f46d29d950295aa594c1327bbf290f54ea954a00`）；不给两个偏好服务加 `implements`、不动 propel 的 `TModuleStatus`。
- **T10**：接受删去 `use-reload-confirmation` 的三个死参数（顺带清它的 `exhaustive-deps`，web 上限 40）；`tools/keywords.json:197` 的样例名字已删的 `generateQueryParams`，交 T13 换成仍存在的 `generateWorkItemLink`。
- **T11**：六个第一次量到的控件就地修（只改类、同 T5）接受；命令面板子菜单没有选中的行、富筛选 “Remove filter” 的英文 aria-label 两条观察交 M8（M6-M8-21、-22）。
- **T13**：接受；主控补 README 两处 River 任务的清单与 plan T7 的 `group_by=module_id`（`de1e8881`）；M6-M8-01 的占位行在 T15 补；`frontend-changes.md:228` 属 M5，不改。

## 4. 实施记录

变异数为能编译的产品变异（e2e 的另要能构建，之后以干净的源重建）；探针是 plan 之外钉住事实的变异或临时文件，不入库。

### 4.1 各 Task

| Task | 提交 | 内容 |
|---|---|---|
| spec、plan 与预检 | `1e85b5d284d5a3aedff269d69b7323f7ca9b95eb`、`65f20a7b6427587dabc9b804b522861c820532ea`、`a0d82965fa995de9ef68409ca5f96a7572965c77` | spec 与 plan（16 个 Task；H1–H16、A1–A14）与本记录的骨架（`spec-doc-links-2` 0 断链、`spec-keywords` 无命中）；预检 re-78 的修正与记录；设计 §14 链接 re-78 |
| T1 P7c 的回填与 phase | `54976c83f40b3985ff3a6201d3d6411205ce57f1` | 原件逐项核对：`t1-p7c-archive-verify`（4718 个文件、751,178,965 字节、6493 个成员、未知 0、清单 sha256 `cfb5ce03…`、head `cf215555`）、`t1-merge-tree`（`ea971606` 的双父 `d052213f`、`cf215555`，树同为 `ed7af4ff`）、`t1-ci-jobs` 与 `t1-e2e-summary`（分支 attempt 2 480 过、故事 22.3 分钟；main attempt 1 479 过、C7 一条失败；attempt 2 480 过、25.6 分钟）；P7c 记录的抬头、§2、§7，P7c spec 的 A14 与状态行，P7c plan，设计的抬头与 §14，`frontend-changes.md:205`，M6-M8-01 三条（web 作业碰到期限、余量的数据点、C7 的再现）；phase `M6/P8`（`t1-keywords`：78 条规则、7 个例外、无命中）；`t1-doc-links` 0 断链；只改文档 7 个文件。裁定 4 条：main 与分支只比汇总行的故事时长与作业时长（`ci-e2e-summary.py` 不认分钟的时长与 `[s3-fixture]`）、P7c 记录 §2 的首句随 A14、M6-M8-01 分写三条、设计 §14 拆开 P8 与 closeout 两行；台账记“本段不给默认套件加故事，余量不变” |
| T2 命令面板其余菜单 | `83aaea3fd546c881b804f0d91a104c3270b84973` | `PowerKMenuBuilder` 加 `getKeywords`（不给时照旧），迭代（打开）、视图、工作区三个菜单以 id 作 value、名字作 keywords；新 `PK/menus/search.test.tsx` 8 例（jsdom、真 cmdk、`powerKFilter`、键盘），RED `t2-red`（4 例因所测性质失败：两项同名时 ArrowDown 到不了第二项；builder 的例按 keywords 找不到），GREEN `t2-green`；`PK/menus/` 与 `PK/ui/modal/` 既有测试一行未改；变异 `t2-mut-value-name`、`t2-mut-no-keywords` 都被杀死（各在 tsc 通过的格式化源上再跑一次 `-2`）；web 481 个文件 5461 例（`t2-web-vitest`）。观察：视图菜单的 `IProjectView` 来自 `@nerve/types`（视图属 M7），入 [M6-M7-04](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-04) |
| T3 英文字面量与表格的列宽 | `d90d9a99637f2f4b667e8f34666a29b17d7d56e9`、`d3d36975078c4cb5705eb34ada24961a71a5c2cb` | 行菜单的 “Restore” 读 `common.actions.restore`；两条提示改为新键 `issue.action_unavailable.title`、`issue.action_unavailable.message`（`handleOptionalAction` 收调用处已翻译的提示，`{action}` 插动作已翻译的名字）；快捷键说明的空结果读新键 `power_k.no_shortcuts_found`（经 `Translated`，查询词仍在 span 里）；表格表头的 “Work items” 读 `work_items`（主控追加）；开关关时表格的合并单元格与加载行的列数取自表头同一份列，`getDisplayPropertiesCount` 删去。RED `t3-red-i18n-4`、`t3-red-colspan`（关迭代或模块时 tfoot 14 列、表头 13 列）、`t3-red-work-items`，GREEN `t3-green-r`；变异 `t3-mut-literal-r`、`t3-mut-toast-r`、`t3-mut-action-name-r2`、`t3-mut-shortcut-r`、`t3-mut-work-items`、`t3-mut-colspan`、`t3-mut-colspan-loader` 都被杀死，`t3-mut-action-name-r` 在“文案即键”的替身下与原样不可区分（等价变异），以 `-r2`（插入英文的动作名，即原缺陷）证；web 483 个文件 5466 例（`t3-web-vitest-r`）。**契约变更 6 处**（主控接受）：`roots/archive-page.test.tsx:145`、`roots/archive-detail.test.tsx:144`、`:152`（不改则 `toBeUndefined` 恒真）、`:184`、`:246`、`peek-overview/restore-menu.test.tsx:107` 断言的 “Restore” 改为 `common.actions.restore`（fake-i18n 下菜单项的文案即键）；断言两条提示、“No shortcuts found for” 与表头英文原文的既有测试：无 |
| T4 归档页的迭代、模块词 | `cff437313106ebe17872f851d084aff0d75f0d6a` | `@nerve/constants` 的 `archived_issues` 加 `cycle_id`、`module_id` 筛选与 `cycle`、`module` 分组键；本地偏好的共用 `displayGroups` 收这两个分组（个人页读出之后照旧收窄为自己的四个）；面板、筛选与组名经共用的件与 `usePlanningNames`，开关关时同项目页；只读守卫核对未改；`plane-diff.md` 的“归档页的迭代、模块词（M6/P8）”一行改写。RED `t4-red-2`（6 例），GREEN `t4-green-2`；变异 `t4-mut-no-vocab`、`t4-mut-no-vocab-group`、`t4-mut-switch-ignored`、`t4-mut-switch-ignored-filter`、`t4-mut-local-drops`、`t4-mut-archive-adds` 都被杀死；新 `roots/archive-vocabulary.test.tsx` 4 例，既有测试只加例、无契约变更；web 484 个文件 5472 例（`t4-web-vitest`） |
| T5 共用控件的焦点轮廓 | `f3ed4d2230fc5426be03129a15f8715d62142dd7`、`0fed290d19c6e6ad771c920e92d5a315cb828f5d`；spec `478da8679a3036998460a53f02d221420a6315d8` | `expectOutlinesWhole` 原样移到 `e2e/fixtures/browser.ts`，原处删去、调用处只改导入（`t5-move`：chromium 14 过）；七处统一用仓库的内缩环 `focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong`（`issues/select/base.tsx:197` 同形），(d) 在调用处 `BreadcrumbLink` 的链接加 `h-full`（链接原比面包屑的那一节高，越出 `overflow-hidden` 的容器上 2px、下 1px），(f) 的 `transition-all` 改 `transition-colors`（环的宽与 offset 渐变时会被量在途中）；`@nerve/ui` 未改，truncate 与 overflow 一处未动。临时探针（三种引擎，不入库）：改前 `t5-probe-before-4` exit 1（七处在其引擎全中），改后 `t5-probe-after-2` 22 处完整、裁剪祖先与截断不变；变异 `t5-mut-d-no-hfull`（只加环、不加 `h-full`）三种引擎各 4 处仍被裁，之后以干净的源重建；`t5-e2e-regression` 默认配置 91 过、`t5-m6-pages` 135 过，断言一行未改 |
| T6 冻结任务逐项目继续 | `c1c0fb559c7e853282e30dfd4c1f8f3532c4efba` | `planning/app/freeze_cycles.go` 的 `Execute`：一个项目失败只回滚它自己的事务、接着下一个，跑完以 `errors.Join` 交回（小类型 `failures`：前 5 个项目的错误各包成 `project <id>: …`，其余只计个数），`ctx` 结束即停、交回此前的失败与 `ctx.Err()`；注释、worker 的注释、`README.md:129` 与 P4 spec `:104` 随之改写。新 `freeze_cycles_failures_test.go` 3 例，RED `t6-red`、GREEN `t6-green`；变异 `t6-mut-stop-at-first`、`t6-mut-drop-error`、`t6-mut-no-cap`、`t6-mut-drop-ctx` 都被杀死；`t6-pkg`、`t6-bootstrap`（`-run Freeze` 6 个测试，`TestFreezeJobLockTimeoutPreservesPostgresError`、`TestFreezeJobPartialCommitAndRetry` 未改）通过。**契约变更 1 处**：`TestFreezeCyclesKeepsTheProjectsBeforeAFailure` 的 a_freeze 子例 `:287` 由“失败之后不再锁下一个项目”改为“锁下一个”，同例其余断言未改 |
| T7 权限矩阵的访客变体行 | `c3cb2719d120f56e3bd26386fd4232fdd3df6cc6` | 新 `bootstrap/permission_matrix_guests_see_all_test.go`：13 个写（迭代 5、模块 5、模块链接 3）各一个 `guests see all` 变体行，跑满 `cycleColumns` 的 14 列（框架只收整张表），182 格：项目访客 `forbidden`，工作区管理员兼访客得成功格；文件头写明不加的四类；守卫 `TestEachPlanningWriteHasAGuestsSeeAllMatrixRow`。`TestPermissionMatrix` 3046 → 3228 格，墙钟 26.30 → 29.39 秒（`-json` 的事件时间戳，`t7-matrix-before-json`、`t7-matrix-json`），远低于 60 秒的停止线。变异 `t7-mut-cycle-write-as-read`、`t7-mut-module-write-as-read` 各只杀死对应变体行的项目访客格，跳过全部变体格时照过（`t7-hole-*`，即 P4 spec §5 所说的洞）；守卫的变异 `t7-mut-guard-drop-row`、`t7-mut-guard-row-listed` 被杀死。归档列表的词：新 `TestArchivedIssueListPlanningWords`（`archived=true` 与 `cycle_id`、`module_id`、`group_by=module_id`），RED `t7-archived-vocab-red`、GREEN `t7-archived-vocab`；既有的行、列与断言未改 |
| T8 规模 | `0f2d331fb3ef39a96c93fceacf52734c00f91cbb` | 删除规模：新 `issue_scale_planning_test.go` 的 `seedPlanningAtScale`（每项目 8 个迭代、6 个模块、18 个成员、12 个链接，每个工作项在一个迭代、两个模块里；自己 ANALYZE 六张表；邻域项目同样播种），只由 `TestTheDeletionsAtTheScaleTarget` 调用（`seedAtScale` 也是 `TestPurgeScale` 的夹具，不改）；基线加上 `planning.delete_rows`，每个被删项目恰一个规划任务，以 `waitPlanningDeletion` 等它完成之后才断言规划的表与两张关系表（活 0、总数不变、带删除时刻与删除者的行数等于总数；`cycle_issues`、`module_issues` 由 `issue.delete_rows` 删，迭代、模块、成员、链接由 `planning.delete_rows` 删）；既有断言一行未改。冻结规模：新 `TestTheFreezeAtTheScaleTarget`（`planning_freeze_scale_test.go` 196 行、`planning_freeze_scale_probe_test.go` 236 行；`NERVE_MEASURE_SCALE` 才跑，必须给证据目录）。冒烟 `t8-smoke-freeze-3`、`t8-smoke-deletion`（数见 §4.3 的规模）；`t8-default-skip`、`-2` 两者默认 SKIP；变异 `t8-mut-freeze-no-index`、`t8-mut-planning-skip-links`、`t8-mut-issue-skip-cycle-issues` 都被杀死（最后一个在没有本 Task 的断言时，规模测试看不出）。冻结夹具由 v1（每项目 20 个迭代、16 个冻结，计划的代价余量 3.6%）改为 v2（40、36，余量 26%） |
| T9 死成员 | 没有提交；spec `f46d29d950295aa594c1327bbf290f54ea954a00` | M1 收尾 plan 附录 A 的三个脚本（SHA-256 同 M5 P8 T7），`t9-typegen` 之后 `t9-before`、`t9-after`：prop 390、member 1224、export 5，`--rows M6` 245 行，前后 tsv 逐字节相同；调研以为的唯一真死成员 `fake-module-dialog.tsx:32-33` 经两个探针证明有读者（`t9-probe-drop-fieldof` 回归测试 2 例失败、`t9-probe-drop-decls` tsc exit 2），不删；`t9-lint-web`、`t9-knip`、`t9-test-web` 通过。读者表见 §4.2 |
| T10 告警的一类 | `8342965d2de648d80ab98a3cd5b6c522ceeadf02` | 清 `no-unused-expressions` 4 条：删 `@nerve/utils` 的 `router.ts`（只有没有读者的 `generateQueryParams`）、`button.tsx` 去掉恒真的守卫、`use-intersection-observer.ts` 改 `callback?.()`、`use-reload-confirmation.tsx` 删三个没有人传的参数；`.oxlintrc.json` 设为 error（全仓含 e2e）；上限 web 44 → 40、utils 3 → 2，全仓告警 60 → 55（`t10-baseline-2` → `t10-after`）。变异 `t10-mut-rule`、`t10-mut-rule-e2e` 以 “errors are never allowed” 失败；`t10-mut-m6-receipt-2` 证收据的脚本抓得到告警；M6 全期改动文件的收据 `t10-oxlint-m6`（`git diff --diff-filter=AMR ba3fd6e0..8342965d`，可 lint 的 696 个文件 0 告警 0 错误）；`t10-lint`、`t10-knip`、`t10-test-web` 通过 |
| T11 页面核对与检查点 | `ea7580d6fdf74bf665d6ebae58d9fc532549b762`、`d94a09997682db603d3ff0352381bbb45d50de99` | `e2e/playwright.m6-acceptance.config.ts`（继承默认配置，`testDir: "acceptance"`，`projects` 换成三种引擎）与 `test:m6-acceptance`；`e2e/acceptance/` 五个文件 8 条（焦点轮廓 3、命令面板的行 1、表单里换项目 2、归档页的词 1、模块的开关 1）；默认配置与 `stories/` 未动。第一次量到的六个控件轮廓不完整，就地修、只改类（`ea7580d6`）：`dropdowns/date.tsx` 的单日期按钮、进行中卡片的一节按钮与页签、富筛选的 Remove filter 与条件的值按钮（`transition-all` → `transition-colors`）、窄屏的 `MobileLayoutSelection`——RED `t11-red-date`、`t11-red-active-section`、`t11-red-active-tab`、`t11-red-remove-filter`、`t11-red-value-transition`（与时机有关，6 例中 1 例）、`t11-red-mobile-layout`；GREEN `t11-acceptance-final` 24 过（34.0 秒）、`t11-acceptance-final-repeat3` 72 过。变异 `t11-mut-outline-a`、`-d`、`-e`、`-c`（Firefox、WebKit）、`-g`（Firefox、WebKit）与 `t11-mut-archived-vocab` 都失败，之后以干净的源重建（`t11-build-clean`）；`t11-e2e-regression` 默认配置 138 过、`t11-m6-pages` 135 过。检查点见 §5 |
| T12 最终矩阵 | 本提交 | 运行由主控（§4.3）；本记录的 §1–§4.3 与 §5 的检查点、spec A1–A13 的勾选、[M6-M8-17](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-17) 补 `test:m4-cleanup` 一条 |
| T13 文档与交接 | `dac255a9a09f0c6c788b6548aad69711ed1339bf`、`de1e88817df2419d35f9611513fa10903ce586ee` | README（M6 一条、e2e 一节的“M6 的验收运行”、`planning.delete_rows` 的六步、`planning.crawl_link`；主控补两处 River 任务的清单），`server/configs/config.yaml:120` 的注释，[前端改动清单](../../frontend-changes.md)（`:205`、`:214`、`:329` 与 §3.2 新五行），设计的抬头与 §14，`tools/keywords.json:197` 的样例改为 `generateWorkItemLink`，spec §2.7 与 plan T7 的 `group_by=module_id`；七份输入交接各写“处理结果（M6/P8）”（`status` 不改，由收尾改为 closed）；M7 的 `M6-closeout` 新 M6-M7-02…09（M5-04 与 M5-M6-01 的原文以脚本逐字节取出），M8 的新 M6-M8-16…22 与 -01、-05、-08 的数据点；`t13-doc-links` 174 份、1806 条相对链接、0 断链，`t13-keywords` 无命中（`dac255a9`：15 个文件 +265 −15） |

网页包从 P7c 终点的 480 个文件 5453 例（[P7c 记录](P7c-palette-mobile-review.md) §4）到 484 个文件 5472 例（`t4-web-vitest`；T5 的 `t5-web-vitest` 同数），之后的 `make test-web`（`t10-test-web`、`t11-gate-test-web`）都通过，台账没有再记例数。e2e 的默认套件照旧 480 条，m6-pages 135 例，新的 m6-acceptance 24 例。Go 新增 planning/app 3 例、矩阵的 182 格与一个守卫、`TestArchivedIssueListPlanningWords`、以环境变量打开的 `TestTheFreezeAtTheScaleTarget` 与删除规模的规划部分。

交出的观察：T2 视图菜单的类型（M6-M7-04）；T3 真 i18n 在 vitest 下不插值（只记台账）；T5 单日期按钮（T11 已修）；T6 自动归档遇错即停（M6-M8-18）；T8 冻结候选查询的规划器余量（M6-M8-20）；T10 关键词的样例（T13 已换）；T11 命令面板的子菜单与 “Remove filter”（M6-M8-21、-22）；T12 M4 的三引擎并集在默认 worker 数下的时序失败（M6-M8-17，§4.3）。

### 4.2 T9 的读者表

抄自台账 `t9-readers.md`（HEAD `0f2d331f`）。路径除另注都相对 `web/apps/web/core/`。

#### 方法与计数

- 脚本：M1 收尾 plan 附录 A 原样取出（台账 `deadsym/`，不入库）。SHA-256：`deadsym.mjs` 3d85f090…、`lib/program.mjs` b2c52513…、`domains.mjs` 5f76ea3b…，与 M5 P8 T7、调研记的相同。TypeScript 5.8.3。先 `pnpm --filter web exec react-router typegen`（`t9-typegen` exit 0）。
- `t9-before`（HEAD，exit 0）：prop 390、member 1224、export 5。`--rows M6` 共 245 行（member 172、prop 73）：测试 206、测试替身 15、产品 24。tsv 在 `deadsym/dead-before.tsv`、`deadsym/rows-M6-before.txt`。
- 对照 `ea971606`（`git archive` 到 scratchpad、`node_modules` 以符号链接借用）：prop 390、member 1217、export 5，`--rows M6` 245 行，与调研一致。存为 `deadsym/dead-base-ea971606.tsv`。
- 与 `ea971606` 比，HEAD 多出 7 行，没有消失的行。7 行都属 M4，来自 T2–T5，都有读者（见“T2–T5 新增的 7 行”）。`--rows M6` 不变。
- 没有删除。调研说的唯一真死成员 `fake-module-dialog.tsx:32-33` 经两次探针证明有读者（见“`fake-module-dialog.tsx:32-33` 不是死成员”）。
- `t9-after`（exit 0）：同为 prop 390、member 1224、export 5，`--rows M6` 245 行，`dead-after.tsv` 与 `dead-before.tsv` 逐字节相同。没有新行。

#### 产品文件（24 行）：都是误报，读者如下

| 文件:行 | 名字 | 为什么被报 | 读者 |
|---|---|---|---|
| `components/cycles/applied-filters/root.tsx:33` | `key`、`i18n_title` | `satisfies` 约束里类型字面量的成员 | 筛选表 `FILTERS` 的项：`:46` 的 `FILTERS.filter(({ key })…)`；`:52` 解构出 `key`、`i18n_title`；`:53` 的 `t(i18n_title)`；`:56` 的 `key={key}` |
| `components/modules/applied-filters/root.tsx:36` | `key`、`i18n_title`、`kind` | 同上 | `:49` 的 `FILTERS.filter(({ key })…)`；`:56` 的 `t(filter.i18n_title)`；`:59` 的 `filter.key`；`:64`、`:71` 的 `filter.kind` |
| `components/cycles/peek-on-entry.tsx:10` | `peekIssue` | 导航的 state 是 `unknown`，经 `in` 收窄后读 | `:17-18`（`"peekIssue" in state && state.peekIssue`）；写在 `:13` 的 `peekOnEntry`，由 `cycles/active-cycle/use-enter-filtered.ts:54` 传给 `navigate` |
| `services/planning/cycle-preferences.service.ts:19`、`:26` | `get`、`update` | 类没有写 `implements`；store 经接口读它 | `store/issue/helpers/page-preferences-filter.store.ts:33-35` 的 `PagePreferencesService<F>`，读在 `:180`（`this.service.get`）、`:219`（`this.service.update`）；实例在 `store/issue/root.store.ts:197-198` 建好并传入 |
| `services/planning/module-preferences.service.ts:19`、`:26` | `get`、`update` | 同上（另有测试的 2 次读） | 同上，实例在 `root.store.ts:202-203` |
| `services/planning/cycle-preferences.service.ts:11`、`module-preferences.service.ts:11` | `filters` | `*PreferencesChange` 的字段，由计算键写入 | 写：`page-preferences-filter.store.ts:219` 的 `{ [field]: next() }`；读：两个文件 `:30` 的 `body: data`，交给接口客户端序列化 |
| `store/planning/cycle.store.ts:20` | `answeredAt` | 以字符串键读 | `maxBy(copies, "answeredAt")`，在 `:81`、`:91`；写在 `:79`、`:160` |
| `store/planning/module.store.ts:20` | `answeredAt` | 同上 | `:84`、`:94`；写在 `:82`、`:167` |
| `web/packages/utils/src/work-item-filters/configs/filters/cycle.ts:23` | `additionalOptions` | 经 `...params` 展开传递，在另一个类型下读 | `cycle.ts:42` 的 `...params` 进 `getMultiSelectConfig` 的 config，`rich-filters/factories/configs/core.ts:56` 读 `config.additionalOptions`；写在 `hooks/work-item-filters/use-planning-filter-configs.tsx:114` |
| `web/packages/utils/src/work-item-filters/configs/filters/module.ts:23` | `additionalOptions` | 同上 | `module.ts:42` 的 `...params`，`core.ts:56`；写在 `use-planning-filter-configs.tsx:128` |
| `components/issues/issue-layouts/spreadsheet/columns/cycle-column.tsx:10-13` | `issue`、`onClose`、`onChange`、`disabled` | 组件经 `SPREADSHEET_COLUMNS` 以 `TSpreadsheetColumn` 传递（与 M5 的 `attachment-column` 同类） | `spreadsheet/spreadsheet-columns.tsx:76` 的 `cycle: SpreadsheetCycleColumn`；`spreadsheet/issue-column.tsx:35` 的 `const Column = SPREADSHEET_COLUMNS[property]`，`:53-57` 的 `<Column issue onChange disabled onClose>` |
| `components/issues/issue-layouts/spreadsheet/columns/module-column.tsx:10-13` | `issue`、`onClose`、`onChange`、`disabled` | 同上 | `spreadsheet-columns.tsx:75` 的 `modules: SpreadsheetModuleColumn`；`issue-column.tsx:35`、`:53-57` |

可选的改法：给两个偏好服务加上 `implements PagePreferencesService<…>`，脚本就能连上 `get`、`update` 这 4 行。T9 不改。原因：这要让 `services/` 去导入 `store/` 的类型，依赖方向反了；而这份约束已经在 `root.store.ts:197-203` 的传参处由类型检查器核对。

#### 测试替身（15 行）：都是误报

| 文件:行 | 名字 | 读者 |
|---|---|---|
| `components/cycles/fake-transfer-pages.tsx:147` | `navigate` | `where.navigate` 由 `:150` 的 `useNavigate()` 写，由 `cycles/transfer-issues.test.tsx:303` 读 |
| `components/cycles/progress-sidebar/fake-sidebar.tsx:23` | `change` | `sent` 的项，由 `cycles/progress-sidebar/sidebar-header.test.tsx` 读（r3） |
| `components/modules/progress-sidebar/fake-sidebar.tsx:26`、`:27` | `change`（`sent`、`linkSent`） | 由 `modules/progress-sidebar/sidebar-changes.test.tsx`、`sidebar-links.test.tsx` 读（r3、r2） |
| `components/modules/fake-module-dialog.tsx:38`、`:39` | `project`、`dates` | `pickers` 的成员，写在 `:68`、`:73`，由 `modules/modal.test.tsx` 读（r5、r2） |
| `components/modules/fake-module-dialog.tsx:22-24` | `value`、`onChange`、`renderCondition` | 替身 `ProjectDropdown` 的 props。真的 `modules/form.tsx` 经 `vi.mock` 把它们传进来（脚本看不到这条路，所以 w0），测试经 `pickers.project` 读 |
| `components/modules/fake-module-dialog.tsx:28-31` | `placeholder`、`memberIds`、`value`、`onChange` | 替身 `MemberDropdownBase` 的 props。真的 `dropdowns/member/dropdown.tsx:44-45` 以 `{...props}` 经 `vi.mock` 传进来；`:82` 存进 `pickers.members[placeholder]`，测试读 |
| `components/modules/fake-module-dialog.tsx:32`、`:33` | `invalid`、`describedBy` | 同上的写者（`modules/form.tsx:239-240`、`:265-266` 经真的 `MemberDropdown` 的 `{...props}`）。读者是 `:84` 的 `{...fieldOf(props)}`：`fieldOf` 以自己参数的类型（`:62`）解构，脚本看不到这次读。它渲染出 `aria-invalid`、`aria-describedby`，`modules/modal.test.tsx:339-363` 的 `lead_id`、`member_ids[1]` 两例断言了它们（`:358-359`）。证据见下一节 |

#### `fake-module-dialog.tsx:32-33` 不是死成员：两次探针

- `t9-probe-baseline` exit 0：`modules/modal.test.tsx` 与 `actions-host.test.tsx` 共 33 例全过。
- `t9-probe-drop-fieldof`：把替身 `MemberDropdownBase` 的 `{...fieldOf(props)}` 去掉，exit 1，2 例失败（refusal 的 `lead_id`、`member_ids[1]` 两例：`aria-invalid` 期望 `"true"`、得 `null`），restored=yes。可见运行时有读者，而且断言它的是回归测试。
- `t9-probe-drop-decls`：只删 `:32-33` 的两条声明，`tsc --noEmit` exit 2：`fake-module-dialog.tsx(82,39): error TS2559: Type 'MemberList' has no properties in common with type '{ invalid?: …; describedBy?: … }'`，restored=yes。可见类型上也有读者，就是 `fieldOf(props)`。
- 结论：这两个成员与同文件 `:22-31` 的 props 同类，都是 `vi.mock` 替身收到的 props（P6a 的结论）。它们只在类型上未被直接读，运行时由 `fieldOf` 读，并由回归测试断言。按 Review Focus 6（删的确实没有读者），T9 不删。

#### 测试文件（206 行）：都是误报，测试自己读

这些是测试自己的数据：被 `vi.mock` 的组件收到的 `children`、`as`，路由的 `params`，测试表与夹具的字段。它们只在测试里被读，而脚本不把测试里的读算作读（`readOf = r − rTest`）。性质与 P6a 的 66 行、M5 的测试行相同。各文件的行数：

| 行数 | 文件 | 名字 |
|---|---|---|
| 2 | `app/…/[projectId]/archives/cycles/page.test.tsx` | params, cycleId |
| 3 | `app/…/[projectId]/archives/modules/page.test.tsx` | params, alt, moduleId |
| 15 | `app/…/cycles/(detail)/[cycleId]/page.test.tsx` | cycleId×3, value, query, isOpen, what, of, nerve, params, title, navigationItems, onChange, header, handleClose |
| 3 | `app/…/cycles/(detail)/header.test.tsx` | children, as, groups |
| 4 | `app/…/cycles/(list)/page.test.tsx` | params, header, mobileHeader, projectId |
| 18 | `app/…/modules/(detail)/[moduleId]/page.test.tsx` | moduleId×2, value, query, isOpen, what, of, nerve, params, fields, unlisted, project, title, navigationItems, onChange, header, handleClose, storeType |
| 3 | `app/…/modules/(detail)/header.test.tsx` | children, as, groups |
| 5 | `app/…/modules/(list)/page.test.tsx` | projectId×2, params, header, mobileHeader |
| 4 | `components/cycles/active-cycle/root.test.tsx` | memberID, source, burndown, cycleId |
| 8 | `components/cycles/active-cycle/row-filters.test.tsx` | children×2, as, kind, tab, name, filters, issueId |
| 3 | `components/cycles/cycle-peek-overview.test.tsx` | cycleId, projectId, handleClose |
| 7 | `components/cycles/cycles-view.test.tsx` | cycleId×2, memberID, alt, source, projectId, row |
| 3 | `components/cycles/modal.test.tsx` | value, onChange, renderCondition |
| 5 | `components/cycles/progress-sidebar/root.test.tsx` | memberID, row, progress, source, burndown |
| 3 | `components/cycles/progress-sidebar/row-filters.test.tsx` | children, as, filters |
| 3 | `components/cycles/quick-actions.test.tsx` | isOpen, cycleId, projectId |
| 6 | `components/dropdowns/cycle/dropdown.test.tsx` | value, projectId, disabled, issueName, buttonVariant, answer |
| 6 | `components/dropdowns/module/dropdown.test.tsx` | value, projectId, disabled, issueName, buttonVariant, answer |
| 1 | `components/issues/issue-detail/issue-activity/activity/actions/cycle.test.tsx` | children |
| 5 | `components/issues/issue-layouts/roots/cycle-create.test.tsx` | children, as, layout, send, planningLoading |
| 5 | `…/roots/cycle-layout-access.test.tsx` | children, as, groupCreate, inlineEdit, drag |
| 5 | `…/roots/cycle-layout-root.test.tsx` | hostScope×2, store, children, as |
| 2 | `…/roots/cycle-quick-actions.test.tsx` | children, as |
| 2 | `…/roots/cycle-row-planning.test.tsx` | children, as |
| 5 | `…/roots/module-create.test.tsx` | children, as, layout, rows, send |
| 5 | `…/roots/module-layout-access.test.tsx` | children, as, groupCreate, inlineEdit, drag |
| 5 | `…/roots/module-layout-root.test.tsx` | hostScope×2, store, children, as |
| 2 | `…/roots/module-quick-actions.test.tsx` | children, as |
| 2 | `…/roots/module-row-dialogs.test.tsx` | children, as |
| 4 | `components/modules/links/use-metadata-refresh.test.tsx` | nerve, at, links, after |
| 1 | `components/modules/module-filters.test.tsx` | alt |
| 3 | `components/modules/module-peek-overview.test.tsx` | moduleId, projectId, handleClose |
| 3 | `components/modules/module-view-header.test.tsx` | moduleId×2, id |
| 5 | `components/modules/modules-list-view.test.tsx` | moduleId×2, memberID, alt, source |
| 3 | `components/modules/progress-sidebar/root.test.tsx` | memberID, source, burndown |
| 3 | `components/modules/progress-sidebar/row-filters.test.tsx` | children, as, filters |
| 8 | `components/modules/quick-actions.test.tsx` | archived×2, project×2, isOpen, moduleId, projectId, created |
| 2 | `components/modules/use-module-display.test.tsx` | name, projectId |
| 1 | `hooks/cycle-preferences-follow.integration.test.tsx` | answer |
| 4 | `hooks/cycle-progress-follow.integration.test.tsx` | send, endpoint, confirmed, answer |
| 5 | `hooks/module-page-actions.integration.test.tsx` | send, endpoint, body, confirmed, answer |
| 4 | `store/planning/cycle.store.changes.test.ts` | change, send, request, refusal |
| 8 | `store/planning/cycle.store.test.ts` | value, given, fetch, path, older, newer, shown, shows |
| 4 | `store/planning/module.store.changes.test.ts` | change, send, request, refusal |
| 8 | `store/planning/module.store.test.ts` | value, given, fetch, path, older, newer, shown, shows |

#### T2–T5 新增的 7 行（属 M4，不在 `--rows M6` 之内）：都有读者

| 文件:行 | 名字 | 读者 |
|---|---|---|
| `components/issues/issue-layouts/quick-action-dropdowns/helper.tsx:28` | `UnavailableToast` 的 `title`、`message`（T3） | 经展开读：`:64` 的 `setToast({ type: TOAST_TYPE.ERROR, ...toast })`；写在 `:32-33` 的 `unavailable` |
| `…/quick-action-dropdowns/helper.test.tsx:23` | `archived`、`project` | 测试自己的变量，写在 `:25`、`:31`，由同文件的断言读 |
| `…/roots/archive-vocabulary.test.tsx:47` | `children` | 被模拟的 `render-if-visible-HOC` 收到的 `children`（JSX 子节点，不是属性，所以 w0） |
| `components/power-k/menus/search.test.tsx:27` | `Row` 的 `id`、`name` | 测试表的类型，同文件读了 5 次 |

#### M1 交接的 84 行（`handoffs/M1-closeout.md:14`）的去向

在 M1 收尾树 `559c3c6`（`git archive` 后重新 typegen）上照原样重跑。结果与 M1 记的相同：prop 452、member 898、export 2，M6 member 46、prop 38，共 84 行（`deadsym/rows-M6-m1-559c3c6.txt`）。逐行对照 HEAD（`deadsym/m1-84-fate.tsv`）：

- **78 行已消失。**
  - 57 行：所在文件已删除（`services/cycle.service.ts`、`services/module.service.ts`、`store/cycle_filter.store.ts`、`store/cycle.store.ts`、`store/module_filter.store.ts`、`store/module.store.ts`、`store/issue/{cycle,module}/filter.store.ts`、`@nerve/types` 的 `cycle/cycle.ts` 与 `module/modules.ts`、`dropdowns/cycle/index.tsx`、`dropdowns/module/module-options.tsx`）。
  - 19 行：成员已从仍在的文件里删去。
  - 2 行：名字还在文件里，但已不是 prop。`cycles/active-cycle/root.tsx` 的 `cycleId` 现在是局部变量；`modules/module-layout-icon.tsx` 的 `className` 现在是 JSX 属性。
- **6 行仍被报，都是误报。** 即 `spreadsheet/columns/cycle-column.tsx` 与 `module-column.tsx` 各自的 `issue`、`onClose`、`disabled`（现在在 `:10`、`:11`、`:13`）。读者见上面产品文件一表的表格两列（`SPREADSHEET_COLUMNS`）。

#### M3 交接 §2 的 8 行（`handoffs/M3-closeout.md:27`）

`core/sidebar/progress-stats/` 的 `label.tsx` 5 行、`state_group.tsx` 3 行，随 P6a `c09d4f8d` 删除 `progress-stats/{assignee,label,state_group}.tsx` 而消失。目录里只剩 `shared.ts`。HEAD 的 tsv 里没有任何 `progress-stats` 行。了结的记录在 `M3-closeout.md:106`。

#### 附：调研 §1.5 未追到读者的行（在 M6 新建的文件里，但不匹配 M6 的路径正则）

| 文件:行 | 名字 | 读者 |
|---|---|---|
| `components/core/sidebar/burndown-chart.tsx:17` | `Day` 的 `date`、`remaining`、`ideal` | 以字符串键读：图表的 `areas[].key`（`"remaining"`、`"ideal"`）与 `xAxis.key`（`"date"`），由 `@nerve/propel` 的 `area-chart/root.tsx:60`、`:102` 当作 `dataKey` 读 |
| `components/issues/issue-layouts/planning-page-layouts.ts:45` | `OptionCaption` 的 `issue` | 写在 `add-existing-dialog.tsx:69` 的 `<OptionCaption issue={issue} />`；读者 `current-cycle-caption.tsx:16` 以自己的类型字面量解构（结构类型） |
| `components/issues/issue-layouts/current-cycle-caption.tsx:16` | `issue`（prop） | 同上：写者是 `add-existing-dialog.tsx:69`，它的类型来自 `planning-page-layouts.ts:45` 的 `ComponentType<{ issue: Issue }>`，是另一个声明 |
| `web/packages/types/src/issues/issue-change.ts:17` | `module_ids?: never` | 类型守卫：不让一次改动带上整个模块集合（P7a R3）。由类型检查器读；`store/issue/issue-write-fields.test.ts:169` 的 `@ts-expect-error` 核对它 |

### 4.3 最终矩阵（T12）

#### 运行

主控以台账的 `t12.sh` 串行运行（2026-11-16T08:30:59Z 起，一次一个容器套件），每次一个收据（`<label>-receipt.json`、`<label>.log`、运行前后的输入清单），原件都在台账。产品 HEAD 是检查点 `d94a09997682db603d3ff0352381bbb45d50de99`（T11 的最后一个提交，也是分支 CI 的 head）。T12 运行期间只落下两个提交，都是文档：T13 的 `dac255a9`（文档、`config.yaml:120` 的注释、关键词的一个样例）与主控的 `de1e8881`（README 与 plan），所以收据所记的 HEAD 分两段：

- 页面配置的运行与四次隔离重复在 `d94a0999` 上；全部 e2e 的运行（含最后 M4 的三个隔离重复）都用 `t12-build` 建的同一个 `bin/nerve`。`t12-build`、`t12-default`、`t12-m6-pages`、`t12-acceptance` 的输入与 `d94a0999` 的树相同、运行前后不变。`t12-m4-cleanup` 的收据 `inputs_unchanged` 为假：运行期间 T13 尚未提交的改动动了六个文件（README、设计、P8 spec、`frontend-changes.md`、`config.yaml:120` 的注释、`tools/keywords.json` 的样例），都不是这次运行的输入。四次隔离重复（`t12-repeat-*`）的收据也记 `d94a0999`、输入前后不变，输入里带着同样几处未提交的改动。
- Go 的运行结束于 `de1e8881`。`t12-go-full` 开始时工作树已有 T13 对 `config.yaml:120` 注释的改动，运行期间又落下 11 个文档文件（七份输入交接、M7 与 M8 的交接、plan、README），没有 Go 的源与测试文件，所以收据的 `inputs_unchanged` 为假。之后的运行（矩阵、三个规模、重跑与 M4 的三个隔离重复）在 `de1e8881` 上、输入前后不变。`git diff d94a0999..de1e8881` 在 `server/` 下只有 `config.yaml` 的那处注释。

| 收据 | 运行 | HEAD | exit | 秒 | 结果 |
|---|---|---|---|---|---|
| `t12-build` | `make build build-s3-fixture` | `d94a0999` | 0 | 1.1 | turbo 10 个任务都取缓存 |
| `t12-default` | 默认配置（`make e2e` 的同一命令）加 JSON 报告 | `d94a0999` | 0 | 233.3 | 480 过（3.9 分钟），flaky 0 |
| `t12-m6-pages` | `playwright.m6-pages.config.ts`，三种引擎 | `d94a0999` | 0 | 150.5 | 135 过（2.5 分钟） |
| `t12-acceptance` | `playwright.m6-acceptance.config.ts`，三种引擎 | `d94a0999` | 0 | 36.0 | 24 过（35.3 秒） |
| `t12-m4-cleanup` | `playwright.m4-cleanup.config.ts`，三种引擎、默认 9 个 worker | `d94a0999`（输入有变，见上） | 1 | 313.6 | 510 例中 507 过、3 条失败（见“间歇项”） |
| `t12-repeat-g11-webkit` | G11 (page)，WebKit，`--repeat-each=5` | `d94a0999` | 0 | 19.5 | 5 过 |
| `t12-repeat-g2` | G2 (page)，三种引擎，`--repeat-each=5` | `d94a0999` | 0 | 21.3 | 15 过 |
| `t12-repeat-c7` | M4 的 C7，chromium，`--repeat-each=5` | `d94a0999` | 0 | 20.3 | C7 5 过（连同 s3 夹具的自检 5 例） |
| `t12-repeat-f11` | M5 的 F11 (page)，chromium，`--repeat-each=5` | `d94a0999` | 0 | 47.4 | [local]、[s3] 各 5 过（连同 s3 夹具的自检 5 例） |
| `t12-go-full` | `GOFLAGS='-p=1 -v' make test` | `de1e8881`（输入有变，见上） | 2 | 1415.0 | server 模块 64 个包中 63 个 ok，bootstrap 只有 `TestImageStorageContract/s3` 失败（M6-M8-05） |
| `t12-matrix` | `go test -p=1 -count=1 -v ./internal/bootstrap -run 'TestPermissionMatrix\|Coverage\|Matrix\|EveryColumnCalls'` | `de1e8881` | 0 | 37.5 | 3416 行 `--- PASS`、0 行 FAIL（包 36.1 秒） |
| `t12-purge-scale` | `NERVE_PURGE_SCALE=1`，`TestPurgeScale` | `de1e8881` | 0 | 35.6 | 过（见“规模”） |
| `t12-delete-scale` | `NERVE_MEASURE_SCALE=1`，`TestTheDeletionsAtTheScaleTarget` | `de1e8881` | 0 | 109.8 | 过 |
| `t12-freeze-scale` | `NERVE_MEASURE_SCALE=1`，`TestTheFreezeAtTheScaleTarget` | `de1e8881` | 0 | 142.2 | 过 |
| `t12-go-rerun-image-storage` | `-run '^TestImageStorageContract$'` | `de1e8881` | 0 | 22.1 | local 与 s3 都过 |
| `t12-m4-repeat-i4-firefox` | I4 (page) 表格按 `sub_issues_count` 排序，Firefox，×5 | `de1e8881` | 0 | 32.3 | 5 过 |
| `t12-m4-repeat-i11-webkit` | I11 (page) 两种回应，WebKit，×5 | `de1e8881` | 0 | 18.6 | 5 过 |
| `t12-m4-repeat-c12-webkit` | C12 (review) 子工作项的相对与自定日期，WebKit，×5 | `de1e8881` | 0 | 20.1 | 5 过 |

- **JSON 报告在日志里**：三个页面配置的命令把 `PLAYWRIGHT_JSON_OUTPUT_NAME=…` 写在 `cd e2e` 之前，这个变量只作用于 `cd`，json reporter 于是写到标准输出：报告在 `t12-default.log` 第 489 行、`t12-m6-pages.log` 第 141 行、`t12-acceptance.log` 第 30 行起到文件末尾；运行之后由主控从日志抽出为台账的 `t12-default.json`、`t12-m6-pages.json`、`t12-acceptance.json`（台账 Task 12 的主控一行）。报告的 stats：480 / 135 / 24 例 expected，unexpected 0、flaky 0。逐条的秒数以 `durations.py` 读这段 JSON 取出（台账的工具，单例自身的时长）。
- **逐条的时长**：m6-acceptance 的 8 条在 chromium 合计 26.5 秒、firefox 53.5 秒、webkit 32.9 秒（T11 的 `t11-acceptance-final` 是 22.4、46.7、30.8 秒）；最长的是 G18 (acceptance)（firefox 11.2 秒）与 G15 (acceptance)（firefox 9.1 秒）。m6-pages 的 45 条在 chromium 合计 283.6 秒、firefox 381.5 秒、webkit 362.5 秒，其中 G19 的 17 条是 76.9、105.6、101.3 秒。默认套件里 `stories/planning/` 的 64 条在 chromium 合计 293.4 秒（G19 的 17 条 75.1 秒），9 个 worker 并行，只作对照。
- **Go 的全量**：`make test` 先跑 server 模块，失败之后不再跑 `server/tools` 模块，所以 `t12-go-full` 里没有 `server/tools` 的两个包（`bodyshapegen`、`s3fixture`）；它们在检查点 `t11-gate-test`（`d94a0999`）通过，之后 `server/tools` 没有改动。server 模块的顶层用例 2623 个 `--- PASS`、1 个 FAIL（`TestImageStorageContract`，其 local 子例过、s3 子例失败）、7 个 SKIP：六个以环境变量打开的测量与演练（`TestFilesCleanupDrill`、`TestIssueListScale`、`TestTheDeletionsAtTheScaleTarget`、`TestTheFreezeAtTheScaleTarget`、`TestPurgeScale`、`TestSearch100KPerformance`）与一个子进程入口（`TestS3RecoveryProcess`）。bootstrap 包 1030.7 秒。

#### G1–G20

接口与数据库的证据是 Go 用例（`bootstrap` 的用例经真的 HTTP 与 Postgres，其余注明包）与默认套件的 `G<n> (API)` 故事。下表列出的 Go 用例在 `t12-go-full` 里都是 `--- PASS`（列出的名字逐个对过日志），规模用例在各自的收据里通过。页面的证据：“默认”是 `t12-default`（chromium），“m6-pages”是 `t12-m6-pages`（三种引擎），“验收”是 `t12-acceptance`（三种引擎），本次都通过。HEAD：页面 `d94a0999`，Go `de1e8881`（`server/` 下与 `d94a0999` 只差 `config.yaml:120` 的注释）。

| G | 接口与数据库（本次） | 页面（本次） | HEAD |
|---|---|---|---|
| G1 | `TestCreateCycleThroughTheApp`、`TestCreateCycleRefusals`、`TestCreateCycleInterleaving`；planning/domain 的 `TestCheckNewDates`、`TestOverlapping`；G1 (API) 默认 1 过 | `g1-create-cycle-page` 2 条：默认 2、m6-pages 6 过；G1 (acceptance)（新建表单里以键盘换项目）验收 3 过 | `d94a0999` · `de1e8881` |
| G2 | planning/domain 的 `TestStatusOnTheProjectsDay`；`TestReadCyclesThroughTheApp`；migrations 的 `TestCyclesSchema`（状态不入库）；G2 (API) 1 过 | `g2-cycle-status-page` 1 条：默认 1、m6-pages 3 过；隔离重复 `t12-repeat-g2` 15 过 | 同上 |
| G3 | `TestUpdateCycleThroughTheApp`、`TestUpdateCycleRefusals`、`TestDeleteCycleThroughTheApp`；planning/domain 的 `TestCheckCyclePatch`、`TestOnlySortOrder`；G3 (API) 1 过 | `g3-change-cycle-page` 2 条：默认 2、m6-pages 6 过 | 同上 |
| G4 | `TestWorkItemsJoinMoveAndLeaveCycles`、`TestIssueEventsCarryTheCycle`、`TestOwnOnlyCallersChangeTheCycleOfTheirOwnWorkItems`、`TestJoiningACycleIsRefusedInOrder`、`TestJoiningACycleIsJudgedOnTheWritesDay`、`TestTheEndOfACycleIsJudgedAfterTheLastLock`、`TestTwoChangesOfAWorkItemsCycleSerialize`、`TestDraftsSaveAnyCycleAndModules`、`TestDraftPublishJoinsTheCycleAndTheModules`、`TestDraftPublishIsRefusedAsACreationIs`、`TestIssueMembershipsAcrossRepresentations`；G4 (API) 1 过 | `g4-issue-cycle-page` 1 条与 `g4-add-existing-page` 的 G4 一条：默认 2、m6-pages 6 过 | 同上 |
| G5 | `TestReadCycleProgressThroughTheApp`、`TestPlanningProgressReadsInOneSnapshot`、`TestListCyclesWithCountsThroughTheApp`、`TestPlanningListsReadEachPageOnceInOneSnapshot`；planning/domain 的 `TestProgressCounts`、`TestProgressDistribution`；G5 (API) 1 过 | `g5-cycle-progress-page` 1 条（含只看自己的 walt）：默认 1、m6-pages 3 过 | 同上 |
| G6 | `TestFreezeCyclesOnEachProjectsDay`、`TestFreezeJobPartialCommitAndRetry`、`TestFreezeJobLockTimeoutPreservesPostgresError`、`TestTheFreezeJobAndACompletionSerialize`、`TestTransferKeepsTheJobsFreeze`；planning/adapter/postgres 的 `TestFreezeCycle`、`TestFreezeDueProjects`、`TestCyclesToFreeze`；migrations 的 `TestCyclesProgressSnapshotMigration`；planning/app 的 P8 新例 `TestFreezeCyclesGoesOnPastAFailedProject`、`TestFreezeCyclesStopsAtItsContextsEnd`、`TestFreezeCyclesBoundsItsError` 与按契约改过的 `TestFreezeCyclesKeepsTheProjectsBeforeAFailure`；规模 `TestTheFreezeAtTheScaleTarget`（`t12-freeze-scale`）；G6 (API) 1 过 | `g6-cycle-freeze-page` 1 条：默认 1、m6-pages 3 过 | 同上 |
| G7 | `TestTransferCycleIssuesThroughTheApp`、`TestTransferCycleIssuesRefusalsThroughTheApp`、`TestATransferAndAMoveOfItsWorkItemSerialize`、`TestATransferAndACompletionOfItsWorkItemSerialize`、`TestATransferAndTheDeletionOfItsTargetSerialize`、`TestTheFreezeJobAndATransferSerialize`、`TestATransferPastItsLockTimeoutIsBusy`、`TestATransferReadsItsClockAfterItsLock`；矩阵的转移行与它的 guests see all 变体 14 格（`t12-matrix`）；G7 (API) 1 过 | `g7-cycle-transfer-page` 1 条（walt、gus）：默认 1、m6-pages 3 过 | 同上 |
| G8 | `TestArchiveCycleThroughTheApp`；planning/app 的 `TestArchiveCycleRefusesAndRepeats`；G8 (API) 1 过 | `g8-archive-cycle-page` 1 条：默认 1、m6-pages 3 过 | 同上 |
| G9 | `TestCreateModuleThroughTheApp`、`TestCreateModuleRefusals`、`TestCreateModuleLeadAndMembers`、`TestCreateModuleInterleaving`、`TestUpdateModuleThroughTheApp`、`TestUpdateModuleRefusals`、`TestReadModulesThroughTheApp`；G9 (API) 2 过 | `g9-create-module-page` 2 条、`g9-module-name-page` 1 条：默认 3、m6-pages 9 过；G9 (acceptance)（新建表单里换项目）验收 3 过 | 同上 |
| G10 | `TestModuleLinksThroughTheApp`、`TestModuleLinkRefusals`、`TestCreateModuleLinkInterleaving`、`TestModuleLinkCrawlRuns`、`TestModuleLinkCrawlJobs`；planning/adapter/river 的 `TestCrawlLinkWorker`、planning/adapter/postgres 的 `TestSetModuleLinkMetadata`；G10 (API) 1 过 | `g10-module-links-page` 1 条：默认 1、m6-pages 3 过 | 同上 |
| G11 | `TestWorkItemsJoinAndLeaveModules`、`TestTheCycleAndTheModulesChangeInOneWrite`、`TestOwnOnlyCallersChangeTheModulesOfTheirOwnWorkItems`、`TestJoiningModulesIsRefusedInOrder`，草稿同 G4；G11 (API) 1 过 | `g11-issue-modules-page` 1 条与 `g4-add-existing-page` 的 G11 一条：默认 2、m6-pages 6 过；隔离重复 `t12-repeat-g11-webkit` 5 过 | 同上 |
| G12 | `TestReadModuleProgressThroughTheApp`、`TestListModulesWithCountsThroughTheApp`，planning/domain 同 G5；G12 (API) 1 过 | `g12-module-progress-page` 1 条（walt）：默认 1、m6-pages 3 过 | 同上 |
| G13 | `TestArchiveModuleThroughTheApp`、`TestDeleteModuleThroughTheApp`；planning/app 的 `TestArchiveModuleRefusesAndRepeats`；G13 (API) 1 过 | `g13-archive-delete-module-page` 3 条（wanda、gus）：默认 3、m6-pages 9 过 | 同上 |
| G14 | `TestDeletingACycleTakesItsWorkItemsOut`、`TestDeletingAModuleTakesItsWorkItemsOut`、`TestJoiningAndDeletingATargetInterleave`、`TestPageSettingsAndTheirTargetsDeletionInterleave`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestAFailedProjectsStepRollsTheDeletionBack`、`TestADeletionRefusedAtItsCommitChangesNoRow`、`TestDeletionViolationsCatchesEachGap`、`TestNewAppPlanningJobCatalog`、`TestPurgeCatalogCoversTablesAndIncomingKeys`、`TestPurgeCatalogRejectsEachGap`、`TestPurgeDeletedRunsWithRuntimeRole`、`TestPurgeParentCascadesUnprocessedIssueRows`；planning/adapter/postgres 的 `TestDeleteProjectPlanningRows`、`TestPurgePlanningTables`、`TestModulePurgeChildrenFirst`、`TestPurgeCycles`；规模 `TestPurgeScale`（`t12-purge-scale`）、`TestTheDeletionsAtTheScaleTarget`（`t12-delete-scale`）；G14 (API) 1 过；项目、工作区的删除与清理经 `furnishPlanningRows`、`furnishWorkspace` 带上规划的行的 W2、W3、P4、I15、I17、I21 也在默认套件里过 | 没有带编号的页面故事（spec §2.12）：迭代、模块的删除入口与删除之后的地址在 G3 (page)、G13 (page)（见上）；删除之后工作项上的归属与动态的显示没有页面故事（调研 `research/g-matrix.md` §0），不为凑编号补写 | 同上 |
| G15 | `TestIssueListPlanningFilters`、`TestIssueListPlanningGroups`（bootstrap 与 issue/adapter/postgres 各一个）、`TestListPlanningNamesSnapshot`、`TestListOtherOrdersReadNoPlanningNames`、`TestIssueListAllOrders`、`TestPlanningPreferencesAPI`、`TestPlanningPreferencesAsRuntimeRole`、`TestAPagePreferencesChangeSharesItsProject`；issue/adapter/postgres 的 `TestFlatIssueListOrders`；P8 新例 `TestArchivedIssueListPlanningWords`；G15 (API) 3 过 | `g15-cycle-list-page` 3 条、`g15-module-list-page` 3 条、`g15-work-item-lists-page` 1 条：默认 7、m6-pages 21 过；G15 (acceptance)（归档页按迭代筛选、按模块分组，请求与组恰如所选）验收 3 过 | 同上 |
| G16 | 只有 Go：`TestAutoArchiveKeepsTheWorkOfOpenCyclesAndModules`（bootstrap）、issue/adapter/postgres 的 `TestAutoArchiveKeepsTheOpenTargetsWorkItems`、planning/adapter/postgres 的 `TestOpenCycleIDs`、`TestOpenModuleIDs`、planning/app 的 `TestOpenTargetsOnTheProjectsDay` | 页面不适用：自动归档是后台的周期任务，没有页面；e2e 的 I16 不涉迭代、模块（调研 `research/g-matrix.md` §0） | `de1e8881` |
| G17 | `TestPermissionMatrix`（`t12-matrix`：3228 格，其中 guests see all 变体 196 格 = P8 的 13 个写 × 14 列 182 格 + 转移 14 格）与覆盖守卫 `TestThePermissionMatrixCoversEveryOperation`、`TestEveryColumnCallsAsARegisteredAccount`、`TestEachMatrixTableIsOfOneLevel`、`TestMatrixViolationsCatchesEachGap`、`TestMatrixViolationsCatchesEachColumnGap`、P8 的 `TestEachPlanningWriteHasAGuestsSeeAllMatrixRow`（`t12-matrix` 与 `t12-go-full` 都过）；planning/app 的 `TestGetCycleProgressOfAnOwnOnlyCaller`、`TestListCyclesRowsOfAnOwnOnlyCaller`；G6、G7 (API) 里的 OwnOnly | 没有带编号的故事（spec §2.12）：页面一侧是已有故事里的断言——walt（只看自己的工作项）在 G5、G7、G12 (page)，gus（访客）在 G7、G13 (page) 与 G19 的迭代、模块列表页，wanda（工作区管理员兼项目访客）在 G13 (page) 与 G19 的两个列表页；本次默认与 m6-pages 都过 | `de1e8881` · `d94a0999` |
| G18 | 带 `planning.cycles_disabled`、`modules_disabled` 的拒绝：`TestCreateCycleRefusals`、`TestUpdateCycleRefusals`、`TestArchiveCycleThroughTheApp`、`TestDeleteCycleThroughTheApp`、`TestTransferCycleIssuesRefusalsThroughTheApp`、`TestCreateModuleRefusals`、`TestUpdateModuleRefusals`、`TestArchiveModuleThroughTheApp`、`TestModuleLinkRefusals`、`TestJoiningACycleIsRefusedInOrder`、`TestJoiningModulesIsRefusedInOrder`、`TestDraftPublishIsRefusedAsACreationIs`；planning/app 的 `TestTheLockPathsToggleFollowsTheAction`；G18 (API)（迭代）1 过，模块的开关在 G9、G10、G11 (API) 里 | G18 (acceptance)（关掉模块：侧栏与命令面板没有入口，按地址读模块页照旧，列表没有模块列与筛选；再打开数据完好）验收 3 过；开关关着的页面另见 G1 (page)、G9 (page)，途中被关的拒绝见 G8 (page)、G13 (page) | 同上 |
| G19 | 没有 Go（网页）；网页单测在 `make test-web`（`t11-gate-test-web`，`d94a0999`） | `g19-cycle-list-page` 4 条、`g19-cycle-pages` 3 条、`g19-module-list-page` 4 条、`g19-module-pages` 3 条、`g19-work-item-side-page` 3 条：默认 17、m6-pages 51 过；G19 (acceptance) 4 条（迭代列表、迭代页、模块列表与模块页的焦点轮廓，命令面板的行）验收 12 过 | `d94a0999` |
| G20 | 静态的门：关键词 78 条规则、7 个例外、无命中（phase `M6/P8`，没有到期的例外；`t11-gate-lint-web` 于 `d94a0999`，`t13-head-keywords` 于 `dac255a9`）；`make knip`（`t11-gate-knip`）；lint 的上限 web 40、utils 2 等于实际（`t10-after`、`t11-gate-lint-web`）；死成员 `t9-before`、`t9-after`（`0f2d331f`，§4.2）；`no-unused-expressions` 清零为 error、M6 全期改动文件 oxlint 0（`t10-oxlint-m6` 于 `8342965d`，T11 改过与新建的 12 个文件 `t11-oxlint-files` 0） | 页面不适用；完整的对等验收是 m6-pages 与验收两个配置（上面各行） | 各收据所记 |

#### 间歇项

- **M6-M8-13（G11 (page) 在 WebKit，晚到的内容让列表离开按钮）**：`t12-m6-pages` 的 G11 两条在三种引擎都过，隔离重复 `t12-repeat-g11-webkit` 5 过；没有再现。故事照该条的处理先等上方的内容到齐再打开列表，产品没有为它改动，[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13) 保持 open。
- **G2 (page) 的上海午夜窗口（M6-M8-16）**：`t12-repeat-g2` 三种引擎 ×5 = 15 过，跑在 08:43:34Z–08:43:56Z，即上海 16:43，离下一个上海午夜（16:00Z）约 7 小时 16 分。外因是跑的时刻（故事按上海此刻的小时给浏览器选另一天的时区，并核对读列表期间上海的日子没有变），这次重复只证窗口之外照过，不证窗口本身；[M6-M8-16](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-16) 交 M8。
- **M6-M8-03（e2e 共用的 Postgres 连接用尽）**：P8 的两次完整默认运行（`t11-gate-build-e2e`、`t12-default`，各 480 条）的日志里都没有 `too many clients` 或 SQLSTATE 53300，也没有失败。它不属某个故事，没有隔离重复；外因是全部 Playwright worker 共用一个 Postgres 测试容器、默认的 `max_connections`（100），本机 9 个 worker 的连接池合起来远超它（[M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03) 的核算）。结论：环境的余量问题，本段没有撞上，交 M8。
- **M6-M8-07（F11 的中止读取）**：`t12-default` 的 F11 (page) [local]、[s3] 都过，隔离重复 `t12-repeat-f11` 两个变体各 5 过；没有再现，该条保持 open。
- **M6-M8-01（M4 的 C7）**：`t12-default` 的 C7 过，隔离重复 `t12-repeat-c7` chromium 5 过；没有再现，该条保持 open。
- **M6-M8-05（Docker Desktop 的端口转发与 macOS 的临时端口重叠）**：`t12-go-full` 的唯一失败 `TestImageStorageContract/s3`：s3test 启动容器 `a777a7ebfae7` 时等状态到期（`wait until ready: get state … context deadline exceeded`），Docker Desktop 的 `com.docker.backend.log.20261116-170845.690:4250` 记 08:49:51Z 开端口转发 `127.0.0.1:55236 -> 172.17.0.5:8333` 时 `bind: address already in use`——该条的签名；bootstrap 包里 `--- FAIL` 只此一处。照规则只重跑这一例一次：`t12-go-rerun-image-storage` 过（20.11 秒，local 与 s3 都过）。Go 全量的结论：`t12-go-full` 的其余各例与这次重跑合起来全过。
- **M4 的三引擎并集 `test:m4-cleanup`**：调研（`research/g-matrix.md` §4）建议在 `expectOutlinesWhole` 移到 `browser.ts` 之后也跑它（M4 的 C12 用同文件的 `focusOutline`）。`t12-m4-cleanup` 一次跑三种引擎、默认的 9 个 worker，510 例中 507 过，三条失败：Firefox 的 I4 (page) 表格按 `sub_issues_count` 换序之后 5 秒内表头没回来（快照是表格整表加载的骨架，同 M6-M8-01 所记 I4 换序重渲染的慢）；WebKit 的 I11 (page) 回应的 DELETE 5 秒内没有发出；WebKit 的 C12 (review) 子工作项的 Filters 点开之后下拉没有开。三条在各自的引擎隔离重复 5 次都过（`t12-m4-repeat-i4-firefox`、`t12-m4-repeat-i11-webkit`、`t12-m4-repeat-c12-webkit`）。P8 没有改回应、子工作项的筛选与换序的整表加载（T3 只改表格 tfoot 的 `colSpan`、续页的加载行与表头首列的键），判为三种引擎同时跑、默认 worker 数下的负载时序，不是 P8 的回归。M4 P16 的配方本是每个引擎一次、`--workers=2`（[P16-C12-preparation](../../M4-issue-core/reviews/P16-C12-preparation.md) §1）；发布前的运行照那个配方，写入 [M6-M8-17](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-17)。

#### 规模

证据目录在台账 `t12-scale/{purge,deletion,freeze}/`，数取自日志与其中的 JSON。

- **冻结**（`t12-freeze-scale`，测试 140.36 秒）：5000 个项目（建 29.6 秒）、每项目 40 个两周的迭代与 128 个工作项（播种 29.5 秒）；更早的一次真实运行冻结 180,000 个迭代（72.4 秒，逐项目的锁持有 p50 13.97、p95 15.14、最大 32.5 毫秒），之后 VACUUM (ANALYZE) 0.37 秒。测量日 `2044-06-15` 12:00 UTC 的候选恰为 5000 个项目中的 2500 个；store 实际发出的 `FreezeDueProjects` 在连接池同样的规划设置下 `EXPLAIN (ANALYZE, BUFFERS)`：Sort ← Aggregate ← Bitmap Heap Scan ← **Bitmap Index Scan on `cycles_freeze_due_idx`**（`end_date <= '2044-06-15'`），代价 34980.77，关掉三种索引扫描时 48360.91（约低 28%）；位图堆扫描估 16999 行、实际 2500；执行 0.789 毫秒，读 115 个共享缓冲块（其中索引 3 块）；store 读候选 1.94 毫秒；`cycles` 45467 页、199491 行，索引 158 页、14071 行。本次运行冻结 2500 个迭代共 3.524 秒，逐项目的锁持有 p50 1.084、p95 1.232、最大 3.699 毫秒，锁等待 p50 0.288 毫秒。没有翻成顺序扫描（T8 的停止条件）。对照 T8 的冒烟 `t8-smoke-freeze-3`（同为 v2 夹具）：代价 35745.38 对 48339.92（余量约 26%）、估 17757 行、执行 0.748 毫秒，2500 个冻结 3.227 秒，锁持有 p50 0.976、p95 1.109、最大 6.508 毫秒。
- **删除**（`t12-delete-scale`，测试 108.10 秒）：规划的播种 4.45 秒、ANALYZE 0.055 秒。项目一步：`issue.delete_rows` 一次运行、attempt 1、0 snooze、8.22 秒；`planning.delete_rows` 1 次运行、0 snooze、0.220 秒。工作区一步：五个 `issue.delete_rows` 各 18.74 秒、`workspace_drafts` 0.241 秒；五个 `planning.delete_rows` 各 1 次运行、0 snooze，0.241–0.377 秒；七个 issue 任务都是一次运行、attempt 1、零错误零 snooze。每个被删项目：`cycles` 8、`modules` 6、`module_members` 18、`module_links` 12（`planning.delete_rows`，各 1 批），`cycle_issues` 20000（20 批）、`module_issues` 40000（40 批，二者由 `issue.delete_rows` 删），活的 0、带删除时刻与删除者的行数等于总数、总数不变；项目一步之后另五个项目与邻域全活、扇出照旧（成员 3、链接 2、迭代 1、模块 2），工作区一步之后邻域全活。邻域的编辑延迟：项目一步处理期间 PATCH p50 3.48、p95 4.45、最大 4.50 毫秒，工作区一步 p50 54.3、p95 65.9、最大 66.9 毫秒。对照：T8 的冒烟 `t8-smoke-deletion` 是 7.66 秒、五个各 16.29 秒、规划任务 0.160 秒与 0.171–0.289 秒、邻域 PATCH 最大 3.98 与 66.0 毫秒；M5 的 P8（[记录](../../M5-files/reviews/P8-final-acceptance-review.md)，没有规划的数据）是 7.6 秒、14.3 秒、邻域 PATCH 最大 68 毫秒。
- **清理**（`t12-purge-scale`，测试 33.87 秒）：`cleanup.purge_deleted` 任务 21，attempt 1，1 次运行、0 snooze、运行 5.52 秒，扫描与删除 660,035 行，778 条语句、没有被预算截断的语句，运行角色 `test_1_server`；旧范围由自己的语句删掉，59 天与活的行不变。M5 的 P8 是一次运行 5.0 秒删除 660,026 行。

**P4 spec §5 `:199`“P8（规模）”的处理结果**（T13 交本记录）：冻结任务改为越过失败的项目继续、交回有上限的合并错误（T6 `c1c0fb55`；P4 spec `:104` 已由 T6 改写）；冻结的规模测试 `TestTheFreezeAtTheScaleTarget` 新增（T8 `0f2d331f`），正式运行 `t12-freeze-scale` 的计划用到 `cycles_freeze_due_idx`（上一条）；年轻部署的规划器余量入 [M6-M8-20](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-20)，自动归档的同形入 [M6-M8-18](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-18)。

#### 设计 §13 的对照

[设计 §13](../M6-design.md#13-完成标准) 的 14 条（`M6-design.md:257-270`）只在收尾首次 main CI 成功后勾选；下表把每条对到本次的收据，或写明属后续的哪一步。

| # | §13 的一条 | 本次的收据 | 勾选 |
|---|---|---|---|
| 1 | `:257` G1–G20 每条有与失败路径相区分的接口、数据库与页面证据 | 上面的 G1–G20 表（`t12-go-full` 与 `t12-go-rerun-image-storage`、`t12-matrix`、三个规模、`t12-default`、`t12-m6-pages`、`t12-acceptance`）；G14、G16、G17、G20 的页面一侧照 spec §2.12 写明 | closeout |
| 2 | `:258` 迭代的日期、状态、重叠、可改范围与归档；模块的名字、状态、成员、链接与归档 | G1–G3、G8 与 G9、G10、G13 的行 | closeout |
| 3 | `:259` 工作项归属只经工作项字段、一个迭代、多个模块，动态完整；转移与冻结 | G4、G11、G6、G7 的行；冻结逐项目继续的单测（planning/app，`t12-go-full`）与 `t12-freeze-scale` | closeout |
| 4 | `:260` 进度口径由每域一个函数计算，各处一致；燃尽与分布按项目时区与可见范围 | 网页的进度只有一个定义 `progressPercent`（`web/packages/utils/src/planning.ts:77`），迭代、模块的列表行、进行中卡片、两种侧栏与页签都读它，单测 `planning.test.ts` 在 `make test-web`（`t11-gate-test-web`）；服务端只给计数（G5、G12 的 Go 行）；G5 (page)、G12 (page)（含只看自己的 walt）三种引擎（`t12-m6-pages`） | closeout |
| 5 | `:261` 列表引擎的筛选、分组、排序、显示与迭代、模块页的列表走同一引擎，Go 与前端同 fixture | `api/testdata/issue-order.json` 的四个读者：Go 的 `TestIssueOrderSamples`（issue/domain）、`TestFlatIssueListOrders`（issue/adapter/postgres）、`TestIssueListAllOrders`（bootstrap）在 `t12-go-full` 过；网页的 `core/lib/issue-order.test.ts`（`ordersAllSharedFixtureScenarios`）在 `make test-web`（`t11-gate-test-web`）；迭代页、模块页的列表 G15 (page) 三种引擎 | closeout |
| 6 | `:262` 权限矩阵与开关、访客与 OwnOnly、私有项目、撤权；无 Plane 的默认 PUT 或跨项目漏洞 | `t12-matrix`（3228 格与覆盖守卫）、G17、G18 的行。默认 PUT：`api/` 里没有 `put:`，`api/modules/planning.yaml` 的 25 个操作只有 get 9、post 8、patch 5、delete 3；合同之外的方法答 404 problem（`TestUnknownAPIRequestsStillAnswerProblem404`，`t12-go-full`）；每个操作都有矩阵的行（`TestThePermissionMatrixCoversEveryOperation`，`t12-matrix`）。跨项目：工作项加入别的项目的迭代、模块与转移到别的项目的迭代都被拒（`TestJoiningACycleIsRefusedInOrder`、`TestJoiningModulesIsRefusedInOrder`、`TestTransferCycleIssuesRefusalsThroughTheApp`，`t12-go-full`） | closeout |
| 7 | `:263` 删除图延伸、事件与清理任务、60 天有界清理、`TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget` 在 M6 代码上重跑 | G14 的行；`t12-purge-scale`、`t12-delete-scale`（含规划的数据与 `planning.delete_rows`） | closeout |
| 8 | `:264` 前端只调 `/api/v0`，类型来自 OpenAPI，stores 按会话分代，旧 service/类型/地址删除 | 关键词规则 `old-cycle-urls`、`old-module-urls`（Plane 的迭代、模块地址不得出现在 web 的源文件里，测试也算）连同其余规则无命中（`t11-gate-lint-web`、`t13-head-keywords`）；G3、G8、G13 (page) 核对页面只问 nerve 的接口（故事标题的 “asks nerve's API alone”）。“类型来自 OpenAPI”“stores 按会话分代”本段没有专门的收据，照各段（P5a、P6a）的记录 | closeout |
| 9 | `:265` 交接中的键盘、弹窗、下拉框、地址、复制在浏览器核对；本领域死成员与改到文件的告警为零，清一类规则 | `t12-acceptance` 与 `t12-m6-pages`；死成员 `t9-before`、`t9-after`（没有真死成员，§4.2）；`no-unused-expressions` 清零为 error、M6 全期改动文件 oxlint 0（`t10-after`、`t10-oxlint-m6`、`t11-oxlint-files`）；上限等于实际（`t11-gate-lint-web`） | closeout |
| 10 | `:266` `analytics`、`copy-hook` 的例外删除，无到期例外；差异清单与前端改动清单同步 | `tools/keywords.json` 的 7 条例外里没有 `analytics`；`copy-hook` 余下的 5 条都在 M7、M8 的文件里（until `M7/closeout` 4 条、`M8/closeout` 1 条），另两条 until `M9`；`keywords.mjs` 遇到到期的例外即失败，phase `M6/P8` 下无命中（`t11-gate-lint-web`、`t13-head-keywords`）；差异清单（T4 `cff43731`）、前端改动清单（T13 `dac255a9`），`t13-doc-links` 0 断链 | closeout |
| 11 | `:267` 所有输入交接 closed；M7/M8 收到 M6-closeout，M5-04 原文未丢 | 七份输入交接的“处理结果（M6/P8）”、M7 的 M6-M7-01…09（M6-M7-08 是 M5-04 的原文）、M8 的 M6-M8-01…22（T13 `dac255a9`）；`status: closed` 由收尾改 | closeout |
| 12 | `:268` 最终本地七项、deep-review 与修复复审、精确分支/main CI 通过；历史失败诚实保留 | 检查点 `t11-gate` 与分支 CI 43831815008（§5）；最终候选、整分支审查与 CI 在 T14–T16 | T14–T16、closeout |
| 13 | `:269` 全阶段与收尾 noff 合入，第二次 noff 后 main 全绿，总体 §9.4 标完成 | — | closeout |
| 14 | `:270` 原始证据完整归档并核验，分支与 worktree 正常清理、未知文件为零 | — | T16、closeout |

## 5. 门禁与 CI

- **检查点**（T11 之后，`d94a09997682db603d3ff0352381bbb45d50de99`）：`t11-gate`（2026-11-16T08:08:17Z 起，串行、一次一个容器套件）七项一次全过，没有重跑：`make gen-check` 6.9 秒；`make lint-go` 4.5 秒（server 与 server/tools 各 0 issues）；`make test` 972.6 秒（66 个包 ok，含 `server/tools` 的两个，bootstrap 969.3 秒；没有 M6-M8-05 的签名）；`make lint-web` 3.3 秒（关键词 78 条规则、7 个例外、无命中；i18n-keys 1512；turbo 50 个任务、35 个取缓存）；`make knip` 2.6 秒；`make test-web` 82.3 秒（turbo 16 个任务、15 个取缓存——缓存键是任务的输入，回放的是同一输入上次成功的输出）；`make build && make e2e` 254.4 秒（480 过、4.2 分钟，没有 M6-M8-03、-07 的签名）。全部收据的 `inputs_unchanged` 为真。
- **推送前的估算**：P8 不给默认套件加故事（480 条不变），基线 1664 秒（P7c 的 main CI 43764306705 attempt 2），上界取至今最慢的 1670 秒，都低于 30 分钟的报告线；2026-11-16T08:30:36Z 推送。
- **分支 CI** 43831815008（head `d94a0999`）attempt 1 全绿、没有重跑：server (0) 133178687666（08:30:48Z–08:45:13Z，865 秒）、server (1) 133178688013（08:30:49Z–08:39:42Z，533 秒）、web 133178688146（08:30:48Z–08:42:13Z，685 秒）、e2e 133185272705（08:45:15Z–09:05:51Z，1236 秒 = 20 分 36 秒）。都在期限与报告线之内：e2e 低于 30 分钟（比基线 1664 秒少 428 秒），web 低于 15 分钟的期限，server 两路都低于 20 分钟的报告线。
- 检查点之后的提交——T13 的 `dac255a9`、`de1e8881`（只改文档、`config.yaml:120` 的注释与关键词的一个样例），T14 修复轮的 `c52fcc6487e0ab7a7d9fde20e3e21610ef991bd2`（只改 e2e 的验收核对与 `e2e/fixtures/browser.ts`，产品代码未改，经过由 T14 写入 §6）与本提交（只改文档）——都没有推送：写本记录时远端分支仍在 `d94a0999`，照 plan 由 T15 推送最终候选。
- **最终候选**（T15，`0ef661393be56c1fc0d88f1c65ca6314f430e6c2`，T14 的修复轮与文档之后；与检查点相比产品代码未变，只有 e2e 的核对与文档）：`final.sh t15-final`（2026-11-16T09:44:33Z 起，串行）：`make gen-check` 4.1 秒；`make lint-go` 1.8 秒（两个模块 0 issues）；`make test` 957.6 秒（66 个包 ok，含 `server/tools` 的两个；没有 M6-M8-05 的签名）；`make lint-web` 5.5 秒（关键词 78 条规则、7 个例外、无命中；turbo 50 个任务、32 个取缓存）；`make knip` 2.5 秒；`make test-web` 0.6 秒（turbo 16 个任务全部取缓存：自检查点以来 `web/` 未改，缓存即 `t11-gate-test-web` 的那次运行）；build+e2e 246.8 秒 exit 2——480 条中 479 条通过，P1 (page)（`e2e/stories/project/p1-create-project-page.spec.ts:63`）在 `expectProjectCreated` 的查询处收到 `sorry, too many clients already`，之后的 `pg_dump` 同样连不上：M6-M8-03 的签名，照规则只重跑该文件一次，`t15-final-rerun-p1` exit 0（8 过，含 s3-fixture 的依赖），失败的 `test-results` 留在台账 `t15-final-test-results`；m6-pages 150.5 秒 exit 0（三种引擎 135 过，`t15-final-m6-pages.json`）；m6-acceptance 35.8 秒 exit 0（三种引擎 24 过，`t15-final-m6-acceptance.json`）。本条之后只有本提交（文档）。
- **最终候选的分支 CI**：本提交推送之后运行，结果、时长与合入后的 main CI 由 closeout 的第一个 Task 回填（同 P7c 的 §5 由 P8 的 T1 回填）。

## 6. 整分支审查

一次 fresh 只读审查（`fable`，范围 `ea971606..de1e8881`，20 个提交、90 个文件），与 T12 的阶段记录的写作同时进行；结论 **Ready after fixes**，没有 Critical，两处 Important、六处 Minor，没有须负责人决定的事。逐条与处理见 [re-79](../../../reviews/re-79-M6P8整体审核.md)，要点：

- **I1**（记录是骨架）由本记录的 T12 提交 `9a420c1f` 解决；**I2**（冷打开的归档页“添加筛选”没有迭代、模块）照 T4 的裁定不改（同 Plane 与归档页的状态、标签、成员筛选，`plane-diff.md` 写明）。
- **修复轮** `c52fcc6487e0ab7a7d9fde20e3e21610ef991bd2`（只改 e2e）：`expectOutlinesWhole` 另核轮廓至少 1 像素宽（m1；`t14-red-outline-width` exit 0 → `t14-mut-outline-width` exit 1）；G18 核开关关闭时条件的值——同一会话里是关之前读到的名字、冷打开的浏览器里是“不可用的选项”（m4；`t14-mut-g18-value-2`、`t14-mut-g18-value-held` exit 1）；三种引擎的验收 24 例与 m6-pages 135 例照过（`t14-fix-acceptance-2`、`t14-fix-m6-pages`）。修复轮发现 `plane-diff.md` 把条件的值写成无条件的“不可用的选项”，照代码与 P7b spec §2.6 改写。
- **文档**：P4 spec `:199` 加“处理结果（M6/P8）”（m2）；M6-M8-20 补冻结任务一次运行的时长上限（River 的默认 `JobTimeout` 1 分钟，m5）；m3 由 `9a420c1f` 补入 M6-M8-17；m6 确认无须改动。
- 本机七项未在 T14 另跑（`t14-gate`）：`d94a0999` 之后到修复轮只有文档与 e2e 核对的提交，产品代码与 `t11-gate`、分支 CI 43831815008 相同；T15 的 `final.sh` 在最终候选上重跑七项与两个页面配置。默认套件的条数 `t14-list`：480（m6-pages 135、m6-acceptance 24）。

## 7. 集成、归档与接续

由 closeout 的第一个 Task 回填。
