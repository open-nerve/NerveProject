# M6 收尾：完成标准的证据与集成记录

> 收尾合同：[closeout spec](../specs/closeout.md)、[plan](../plans/closeout.md)，预检 [re-80](../../../reviews/re-80-M6收尾预检.md)。本页列出各阶段的合入、main CI 与归档（§1），[设计 §13](../M6-design.md#13-完成标准) 十四条各自的最终证据（§2），输入交接里条件为“M6 的评审写明”的逐项表（§3）与本阶段的预检（§4）。§13 只在收尾首次 `--no-ff` 的 main CI 全绿之后逐条勾选，本页写成时（T3）一条都没有勾；第二次 main CI、归档与清理的结果写在仓库外的集成收据（§5），不由被它验证的提交预报。
> 本页的数字与 SHA 取自收尾台账 `.superpowers/sdd/M6-closeout/` 的收据（git 排除，随 T6 归档）：`t3-merges`（main 上 M6 的合并与双父）、`t3-main-ci` 与 `t3-main-ci-attempts`（GitHub 的接口，只读）、`t3-archives`（各归档的清单）、`t3-story-greps` 与 `t3-gaps`（下文引用的故事与代码行）、T1 的 `t1-*` 与 T2 的 `t2-*`；各阶段的细节以该阶段记录的 §5、§7 为准。

## 1. 各阶段的合入、main CI 与归档

`ba3fd6e0..c4cd1880`（M5 收尾的 noff 到 P8 的 noff）的 main 上有 19 个合并，都是双父（`t3-merges`）。main CI 一列是合入后的运行在 GitHub 接口上的结论与 attempt（`t3-main-ci`、`t3-main-ci-attempts`）；未过的运行照实列出，随后的修复合并写在同一行。归档根都在主目录的 `.superpowers/sdd/` 下（git 排除），括号里是清单 `manifest.json` 的 `head`（`t3-archives`）；建档、核验与清理的收据见各阶段记录的 §7。

| 阶段 | main 上的 noff | 合入后的 main CI | 阶段记录 | 预检 / 整分支审查 | 归档根（清单的 head） |
|---|---|---|---|---|---|
| 设计与 P1 预检 | `d1ec8c69`；e2e 作业期限 25 → 35 分钟 `910a55cd`（负责人同意） | 42281169459：attempt 1 的 e2e 以 M4 的 C7 失败，attempt 2 的 e2e 在 25 分钟的期限处被取消（[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)）；42332404481：attempt 1 的 e2e 未过，attempt 2 通过 | — | [re-54](../../../reviews/re-54-M6设计与P1预检.md) | `M6-design-integration-20261107`（`a8cdbe2c`） |
| P1 迭代后端 | `0a601f21` | 42343073949 attempt 1 通过 | [P1](P1-cycles-backend-review.md) | [re-55](../../../reviews/re-55-M6P1整体审核.md) | `M6-P1-integration-20261107`（`72b83beb`） |
| P2 模块后端 | `861ea1b5`；根因修复 `ddd79a9f` | 42399566014 两个 attempt 的 server 作业都未过（锁探测被启动时的修剪任务骗过、S3 测试容器的端口冲突，M6-M8-01）；42458454710 attempt 1 通过 | [P2](P2-modules-backend-review.md) | [re-56](../../../reviews/re-56-M6P2预检.md) / [re-57](../../../reviews/re-57-M6P2整体审核.md) | `M6-P2-integration-20261108`（`6eea8e09`） |
| P3a 工作项的归属 | `2648a739` | 42487991301 attempt 1 通过 | [P3a](P3a-issue-membership-review.md) | [re-58](../../../reviews/re-58-M6P3a预检.md) / [re-59](../../../reviews/re-59-M6P3a整体审核.md) | `M6-P3a-integration-20261108`（`384a365f`） |
| P3b 列表引擎与偏好 | `ed8f0df4` | 42621199100 attempt 1 通过 | [P3b](P3b-list-engine-review.md) | [re-60](../../../reviews/re-60-M6P3b预检.md) / [re-61](../../../reviews/re-61-M6P3b整体审核.md) | `M6-P3b-integration-20261109`（`79a540f9`） |
| P4 进度、冻结与转移 | `c3ba5a40`；根因修复 `a2b50624` | 42809569341 未过（e2e 的 C12，编辑器建成之前就挂引用的产品缺陷）；42824266166 attempt 1 通过 | [P4](P4-progress-transfer-review.md) | [re-62](../../../reviews/re-62-M6P4预检.md) / [re-63](../../../reviews/re-63-M6P4整体审核.md) | `M6-P4-integration-20261110`（`292f40fe`）、`M6-P4-ci-20261110`（`d7824b26`） |
| P5a 网页的迭代 | `0b8efcd7` | 42995581409 attempt 1 通过 | [P5a](P5a-web-cycles-review.md) | [re-64](../../../reviews/re-64-M6P5a预检.md) / [re-65](../../../reviews/re-65-M6P5a整体审核.md) | `M6-P5a-integration-20261111`（`8175e76e`） |
| P5b 迭代页的工作项列表 | `9675bb31`；根因修正 `aadaf61a` | 43362963122 未过（e2e 的 G15 故事三以两个页面都有的行为锚，本段故事的缺陷）；43373039561 attempt 1 通过 | [P5b](P5b-cycle-list-review.md) | [re-66](../../../reviews/re-66-M6P5b预检.md) / [re-67](../../../reviews/re-67-M6P5b整体审核.md) | `M6-P5b-integration-20261113`（`4a11baeb`） |
| P6a 网页的模块 | `06b72d4b` | 43469787621：attempt 1 的 e2e 以 M4 的 C7 失败（M6-M8-01），只重跑失败的作业，attempt 2 通过 | [P6a](P6a-web-modules-review.md) | [re-68](../../../reviews/re-68-M6P6a预检.md) / [re-69](../../../reviews/re-69-M6P6a整体审核.md) | `M6-P6a-integration-20261114`（`8188582e`） |
| P6b 模块页的工作项列表 | `954a1838`；修复 `7868a6b4` | 43549053092 未过（e2e 的 M4 I4 表格排序故事超出 90 秒的时限，M6-M8-01）；43557098427 attempt 1 通过 | [P6b](P6b-module-list-review.md) | [re-70](../../../reviews/re-70-M6P6b预检.md) / [re-71](../../../reviews/re-71-M6P6b整体审核.md) | `M6-P6b-integration-20261114`（`c0572192`） |
| P7a 工作项一侧（一） | `f42f9d64` | 43603772626 attempt 1 通过 | [P7a](P7a-work-item-planning-review.md) | [re-72](../../../reviews/re-72-M6P7a预检.md) / [re-73](../../../reviews/re-73-M6P7a整体审核.md) | `M6-P7a-integration-20261114`（`5d6dab04`） |
| P7b 工作项一侧（二） | `d052213f` | 43706438924 attempt 1 通过 | [P7b](P7b-list-vocabulary-review.md) | [re-74](../../../reviews/re-74-M6P7b预检.md) / [re-75](../../../reviews/re-75-M6P7b整体审核.md) | `M6-P7b-integration-20261115`（`4fe308ce`） |
| P7c 工作项一侧（三） | `ea971606` | 43764306705：attempt 1 的 e2e 以 M4 的 C7 失败（M6-M8-01），只重跑失败的作业，attempt 2 通过 | [P7c](P7c-palette-mobile-review.md) | [re-76](../../../reviews/re-76-M6P7c预检.md) / [re-77](../../../reviews/re-77-M6P7c整体审核.md) | `M6-P7c-integration-20261116`（`cf215555`） |
| P8 最终验收 | `c4cd1880` | 43851660778 attempt 1 通过 | [P8](P8-final-acceptance-review.md) | [re-78](../../../reviews/re-78-M6P8预检.md) / [re-79](../../../reviews/re-79-M6P8整体审核.md) | `M6-P8-integration-20261116`（`352740ac`） |

P8 的归档没有建立与清理之后的核验收据，收尾的 T1 以 `archive.py verify` 再核（`t1-p8-archive-verify`：4747 个文件、539,117,657 字节、7271 个成员、未知 0，[P8 阶段记录 §7](P8-final-acceptance-review.md#7-集成归档与接续)）。

## 2. 设计 §13 十四条的证据

P8 阶段记录的 [§4.3](P8-final-acceptance-review.md#43-最终矩阵t12) 有 G1–G20 的证据表与设计 §13 的对照（`P8-final-acceptance-review.md:308-321`）；下表在那份对照之上补齐 P8 合入之后的事实与收尾的再核。“范围”一列照 P8 的对照（`:315-317`）写明第 8、9、10 条只对本领域（迭代、模块）成立的部分。此时不勾选。

| # | §13 的一条（`M6-design.md`） | 最终证据 | 范围 |
|---|---|---|---|
| 1 | `:257` G1–G20 每条有与失败路径相区分的接口、数据库与页面证据 | P8 §4.3 的 G1–G20 表：Go 全量 `t12-go-full`（唯一的失败 `TestImageStorageContract/s3` 是 M6-M8-05 的签名，只重跑该例 `t12-go-rerun-image-storage` 即过）、`t12-matrix`、三个规模 `t12-purge-scale`、`t12-delete-scale`、`t12-freeze-scale`，页面 `t12-default`、`t12-m6-pages`、`t12-acceptance`；各 G 落地时的 RED、GREEN 与变异在 §1 的各阶段记录 | G14、G16、G17、G20 的页面一侧照 P8 spec §2.12 写明证据所在（没有带编号的故事或页面不适用），不为凑编号补故事 |
| 2 | `:258` 迭代的日期、状态、重叠、可改范围与归档；模块的名字、状态、成员、链接与归档 | P8 §4.3 的 G1–G3、G8 与 G9、G10、G13 行；后端在 P1（迭代）、P2（模块），页面在 P5a、P6a | — |
| 3 | `:259` 工作项归属只经工作项字段、一个迭代、多个模块，动态完整；转移与冻结 | G4、G11（P3a 的归属与动态，P7a、P7b 的页面）与 G6、G7（P4）的行；冻结任务越过失败的项目继续（P8 的 T6）与正式的冻结规模 `t12-freeze-scale`（计划用到 `cycles_freeze_due_idx`） | — |
| 4 | `:260` 进度口径由每域一个函数计算，各处一致；燃尽与分布按项目时区与可见范围 | 网页的进度只有 `progressPercent` 一个定义（`web/packages/utils/src/planning.ts:77`），迭代、模块的列表行、进行中卡片、两种侧栏与页签都读它（P5a、P6a；[M1 → M6 交接](../handoffs/M1-closeout.md)的进度一节）；服务端只给计数，燃尽与分布按项目时区、只看自己的访客只算他建立的（P4，G5、G12 的 Go 行）；G5 (page)、G12 (page) 三种引擎（`t12-m6-pages`） | — |
| 5 | `:261` 列表引擎的筛选、分组、排序、显示与迭代、模块页的列表走同一引擎，Go 与前端同 fixture | `api/testdata/issue-order.json` 的四个读者：Go 的 `TestIssueOrderSamples`、`TestFlatIssueListOrders`、`TestIssueListAllOrders`（`t12-go-full`）与网页的 `core/lib/issue-order.test.ts`（`make test-web`）；引擎在 P3b，迭代页、模块页的列表在 P5b、P6b，词表在 P7b；G15 (page) 三种引擎 | — |
| 6 | `:262` 权限矩阵与开关、访客与 OwnOnly、私有项目、撤权；无 Plane 的默认 PUT 或跨项目漏洞 | `t12-matrix`：3228 格与覆盖守卫，迭代、模块表的 14 列含私有项目的成员、被移出与已删除的账户（`callerMemberPrivate`、`callerRemoved`、`callerDeleted`，`server/internal/bootstrap/permission_matrix_cycles_test.go:40-42`），访客变体行 196 格；G17、G18 的行。默认 PUT：`api/` 里没有 `put:`（收尾在 `e1cf6f4d` 上再核，`t2-handoff-claims-2`），合同之外的方法答 404 problem；跨项目的加入与转移都被拒（P8 对照的第 6 行） | — |
| 7 | `:263` 删除图延伸、事件与清理任务、60 天有界清理、`TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget` 在 M6 代码上重跑 | G14 的行（删除图与清理目录在 P1–P3b 随每张表扩展，守卫 `TestPurgeCatalogCoversTablesAndIncomingKeys` 等）；最终代码上的 `t12-purge-scale` 与 `t12-delete-scale`（P8 的 T8 让删除规模播种规划的数据并等 `planning.delete_rows`），数字见 [P8 §4.3 的规模](P8-final-acceptance-review.md#规模) | — |
| 8 | `:264` 前端只调 `/api/v0`，类型来自 OpenAPI，stores 按会话分代，旧 service、类型与地址删除 | 关键词规则 `old-cycle-urls`、`old-module-urls`（Plane 的迭代、模块地址不得出现在 web 的源文件里）在 phase `M6/closeout` 下无命中（`t1-keywords`）；旧的迭代、模块 service 与 `@nerve/types` 的 `cycle/`、`module/` 在 P5a、P6a 删除，类型来自 `@nerve/api-client`；按会话分代见 [M2 → M6 交接](../handoffs/M2-closeout.md)第 3 节（P5a、P6a 的 `store-context.test.ts`，P7c 的 `useSessionSWR`）；G3、G8、G13 (page) 核对页面只问 nerve 的接口 | 本领域成立。别的领域的旧服务——收藏、视图、收集箱、webhook、通知仍调旧地址——各归 M7、M8 |
| 9 | `:265` 交接中的键盘、弹窗、下拉框、地址、复制逐项在浏览器核对；本领域死成员与改到文件的告警为零，清一类规则 | 浏览器核对：本页 §3 的两张逐项表（M3 第 3 节、M2 第 4、5 节）与最终运行 `t12-default`、`t12-m6-pages`、`t12-acceptance`；地址与复制见 M3 交接第 4 节与 M2 交接第 6 节的关闭段。死成员：`t9-before`、`t9-after` 与 P8 §4.2 的[读者表](P8-final-acceptance-review.md#42-t9-的读者表)；告警：M6 全期改动的 696 个可 lint 的文件 0 告警（`t10-oxlint-m6`），T11 之后的 12 个文件 0（`t11-oxlint-files`）；清一类：`no-unused-expressions` 全仓清零并设为 error（P8 的 T10） | 本领域的死成员为零指没有真死成员：`--rows M6` 的 245 行都有读者（逐行写明），不是脚本列出 0 行；M3 第 3 节视图的三项 M6 走不到，转 M7；写本表时发现的两项（点外面关上弹窗、Escape 之后焦点回到筛选的按钮）与模块一行右键菜单点它的项，由收尾补核（`f27cce7a`，§3.1） |
| 10 | `:266` `analytics`、`copy-hook` 的例外删除，无到期例外；差异清单与前端改动清单同步 | `analytics` 的例外在 P1 删除；`copy-hook` 中 M6 的例外在 P5a、P6a 删尽（例外 10 → 7）；phase `M6/closeout` 下 7 条例外没有到期的（`t1-keywords`；`until` 为 M9 ×2、`M7/closeout` ×4、`M8/closeout` ×1，`t2-handoff-claims-2`）；[差异清单](../../plane-diff.md)（P8 的 T4 与修复轮）、[前端改动清单](../../frontend-changes.md)（P8 的 T13、收尾的 T1） | `copy-hook` 余下的 5 条在 M7 的收集箱、视图与 M8 的 webhook 文件，属其 M |
| 11 | `:267` 所有输入交接 closed；M7、M8 收到 M6-closeout，M5-04 原文未丢 | 收尾的 T2（`90f79c19`）：七份输入交接 `status: closed`，各有“关闭（M6/closeout）”一段；[M7 的 M6-closeout](../../M7-collaboration/handoffs/M6-closeout.md) 有 M6-M7-01…09、[M8 的 M6-closeout](../../M8-open-release/handoffs/M6-closeout.md) 有 M6-M8-01…22（`t2-anchors`），两份保持 open；M5-04 的原文在 [M6-M7-08](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-08)，M8 另持 M5 的原文 | M3 交接第 3 节写本表时发现未核对的两项已由收尾补核（`f27cce7a`，§3.1），M3 交接的 `status` 照 T2 为 closed |
| 12 | `:268` 最终本地七项、deep-review 与修复复审、精确分支与 main CI 通过；历史失败诚实保留 | P8 最终候选 `0ef66139` 的七项 `t15-final`（build+e2e 以 M6-M8-03 的签名失败一条，只重跑该文件即过）与 m6-pages、m6-acceptance；整分支审查 re-79、修复轮 `c52fcc64` 与限定复审；最终 HEAD `352740ac` 的分支 CI 43846387746 与 main CI 43851660778 都是 attempt 1 全绿（[P8 §5、§7](P8-final-acceptance-review.md#7-集成归档与接续)）；各段的整分支审查与修复复审见 §1；未过的 main CI 与只重跑失败作业的几次都照实留在 §1 与各阶段记录 | 收尾的首次与第二次 main CI 另记（§5） |
| 13 | `:269` 全阶段与收尾 noff 合入，完成标记第二次 noff 后 main 全绿，总体 §9.4 标完成 | §1 的 19 个 noff（都是双父，`t3-merges`）；收尾的首次与第二次 noff、总体 §9.4 在 T4、T5（§5） | 第二次 main CI 由集成收据签结 |
| 14 | `:270` 原始证据完整归档并核验，分支与 worktree 正常清理、未知文件为零 | §1 各阶段的归档根（`t3-archives`）与各阶段记录 §7 的核验与清理；P8 的归档在清理之后以 `t1-p8-archive-verify` 再核；收尾的归档、清理与再核在 T6 | 由集成收据签结（§5） |

## 3. 输入交接的逐项表

两处关闭条件写的是“M6 的评审写明”：[M3 → M6 交接](../handoffs/M3-closeout.md)第 3 节（`:45`）与 [M2 → M6 交接](../handoffs/M2-closeout.md)第 4、5 节（`:37`、`:47`）。各项的处理散在各阶段的处理结果里；P8 的处理结果说 M3 第 3 节“由 P8 阶段记录的最终矩阵（T12）逐项汇总”，P8 记录里并没有这样的汇总（预检 re-80 I3），逐项汇总在此。页面故事都在 `e2e/stories/planning/`，“三种引擎”指 `playwright.m6-pages.config.ts`；P8 的最终运行 `t12-default`（480 过）与 `t12-m6-pages`（135 过）跑了下表的全部页面故事，`t12-acceptance`（24 过）跑了 `e2e/acceptance/` 的核对。

### 3.1 M3 第 3 节

| 条款（`M3-closeout.md`） | 阶段的处理结果 | 故事或收据 |
|---|---|---|
| `:32` 弹窗：Tab 在弹窗里转、Escape 与点外面、关上之后的焦点、弹窗里经 portal 的日期与下拉框；确认框被拒时留着、只在成功后关上 | 迭代 P5a；模块 P6a；工作项弹窗里的迭代、模块两栏 P7a。P5a、P6a 的 spec 写明 Escape 与点外面都经 `ModalCore` 的 `handleClose`，回执与故事只核 Escape | Tab、Escape、焦点、日期与确认框：G1、G3、G8 (page) 与 G19 的迭代页（迭代）；G9、G13 (page) 与 G19 的模块页（模块）；G4、G11 (page)（工作项的新建与修改弹窗里，列表在按钮旁，Escape 只关列表）。**点外面**：M6 的页面故事原本没有这条断言（`closedByBackdropClick` 只在 A11 的 `a11-api-tokens.spec.ts` 用，`t3-gaps`），由收尾补核：`e2e/acceptance/g19-m3-dialogs-and-filters.spec.ts` 的第一条（三种引擎），新建迭代、迭代的修改、新建模块、模块的修改四个弹窗点外面即关上、不发写、焦点回到打开它的按钮（`cl-acceptance`；变异 `cl-mut-backdrop`） |
| `:32-33` 中视图的弹窗 | M6 走不到（视图页没有读，P7c 的回执） | 转 M7：[M6-M7-04](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-04) |
| `:33` 弹窗的名字经 `ModalTitle`；自定义日期筛选的标题经 `t()` | 迭代的新建、修改、归档、转移与删除 P5a；模块的新建、修改、归档、删除与链接 P6a（模块的日期筛选与迭代共用 `planning/filter-date.tsx`）；“添加已有” M4/P12 已改、P7b 核对 | `toHaveAccessibleName`：“Create cycle”（`g1-create-cycle-page`、`g19-cycle-pages`）、“Update cycle”“Delete cycle”（`g3-change-cycle-page`）、“Archive cycle …”（`g8-archive-cycle-page`）、“Transfer work items”（`g7-cycle-transfer-page`）、“Create Module”“Update Module”（`g9-create-module-page`、`g9-module-name-page`、`g19-module-pages`）、“Add link”“Update link”（`g10-module-links-page`）、“Archive module …”“Delete module”（`g13-archive-delete-module-page`）、“Add existing work item”（`g4-add-existing-page`、`g19-work-item-side-page`）（`t3-story-greps`） |
| `:34` 正的 `tabIndex` | `PROJECT_CYCLE`、`PROJECT_MODULE` 与两处归档按钮的 `tabIndex={1}` 删除（P5a、P6a），`ModuleStatusSelect` 不再收 `tabIndex`；视图表单的 `PROJECT_VIEW` 与 `getTabIndex` 的最后删除转 M7（[M6-M7-05](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-05)） | G1、G3 (page) 与 G9 (page) 的 Tab 在弹窗里按次序转；变异 `t7-mut-archive-positive-tabindex`（P6a）；`getTabIndex` 的读者只剩 M7 的五个文件（`t2-handoff-claims-2`） |
| `:35` 嵌套的按钮 | 迭代、模块的快捷操作的菜单按钮不再嵌按钮、以名字命名（P5a、P6a） | G19 的模块页核对它的名字与 `aria-expanded`；迭代一行的菜单见 P5a 的回执 |
| `:36` 右键菜单 | 迭代 P5a（`9831341c`）；模块 P6a | `g3-change-cycle-page` 关着时 `inert`、右键一行打开、Escape 回到该行、再右键点“Edit”打开修改弹窗；`g19-module-pages` 关着时 `inert`、右键打开、首项取得焦点、Escape 回到行的链接，没有点它的项（`t3-story-greps`、`t3-gaps`）；收尾补核 `e2e/acceptance/g19-m3-dialogs-and-filters.spec.ts` 的第三条：右键 Beta 一行、菜单与按钮的菜单同项、点 “Edit” 打开带 Beta 的值的修改弹窗（`cl-acceptance`；变异 `cl-mut-module-context-item`） |
| `:36` 中视图的列表 | M6 走不到 | 转 M7：M6-M7-04 |
| `:37` `CustomSearchSelect` 的调用方以键盘核对 | 成员下拉框（负责人、成员）P6a；迭代页头的切换 P5a；模块页头的切换 P6a；工作项的迭代、模块下拉框 P7a | G9 (page) 选负责人与成员；G19 的迭代页、模块页经页头换到另一个；G4、G11 (page) 以键盘打开、搜索框取得焦点、Escape 回到按钮 |
| `:37` 中视图的页头 | M6 走不到 | 转 M7：M6-M7-04 的“何时验收” |
| `:37` 共用组件的焦点轮廓不被 `overflow-hidden` 裁掉 | 四种共用控件 P6a 的修复轮 F2；其余十三处 P8 的 T5、T11 在根因处改为内缩环 | `g19-module-pages`（P6a）；`e2e/acceptance/g19-focus-outlines.spec.ts` 三种引擎（`t11-acceptance-final`、`t12-acceptance`）；命令面板的行另在 `g19-palette-rows.spec.ts` |
| `:38` 筛选的按钮以 `title` 为名、`aria-expanded`、Escape 之后焦点回到它 | 迭代 P5a（名字与 `aria-expanded`）；模块 P6a（名字、`aria-expanded`、Escape 关上） | 名字与 `aria-expanded`：`g19-cycle-pages.spec.ts:227`、`g19-module-pages.spec.ts:249` 的 “Filters” 与各节，Escape 之后 `aria-expanded` 为 false。**Escape 之后焦点回到它**：M6 的页面故事原本没有这条断言（同一个共用的 `FiltersDropdown` 只在 M3 的 `p5-project-members-page.spec.ts:77`、`:105` 断言过，`t3-gaps`），由收尾补核：`e2e/acceptance/g19-m3-dialogs-and-filters.spec.ts` 的第二条，迭代列表、模块列表的 “Filters” 与迭代页、模块页的默认按钮 “Display”，焦点进到面板之后按 Escape，`aria-expanded` 为 false、焦点回到按钮（`cl-acceptance`；变异 `cl-mut-filters-focus`） |
| `:39` 每一行的控件以它那一行为名 | 迭代、模块一行的菜单经 `common.item_options`（P5a、P6a），链接一行的改、删、复制（P6a）；行与表格的迭代、模块下拉框（P7a）；分组头的“添加已有”（P7b） | G19 的模块页；G4 (page) 以行名找到那一行的迭代控件；G11 (page) 的 “Add existing work item: Todo” |
| `:41` 字段的提示：名字的 `FieldError`、`aria-invalid`、被拒时取得焦点；模块链接的地址的提示 | 迭代 P5a；模块 P6a；工作项弹窗与草稿的迭代、模块两栏 P7a | G1 (page)（迭代的名字）；G9 (page)（空的名字与只有空格的名字）；G10 (page)（链接的地址为空时提示、不发出） |
| `:42` 弹窗里的列表 `modal={false}` 可用；“添加已有”打开时后面的页面隐藏 | 日期范围 P5a、P6a；迭代、模块下拉框 P7a；“添加已有” P7b | G1、G3、G9 (page) 在弹窗里选日期；G4、G11 (page) 弹窗里的列表在按钮旁；G4 (page) 的 `pageBehindHidden` |
| `:43` “添加已有”选项里只在指针经过时显示的链接、已选的移除按钮没有名字 | M4/P12 已改，P7b 核对；打开即聚焦搜索框 P7c | G4、G11 (page)：选项里 `a` 为 0，移除按钮依次是 “Remove Z”“Remove Y”“Remove X”；`g19-work-item-side-page` 的第三条（变异 `t13-mut-focus`） |
| `:44` 模块的状态选择没过规则时只变红 | 经 `t()` 的提示、`aria-invalid`、被拒时取得焦点（P6a） | 组件测试 `select/status.test.tsx`；界面上走不到（新建默认“积压”，修改带着模块的状态，P6a 的回执） |

**写本表时发现的未核对项与补核**：`:32` 的“点外面”关上迭代、模块的弹窗，与 `:38` 的“Escape 之后焦点回到筛选的按钮”，各阶段的处理结果与 P8 的处理结果都没有提到，M6 的页面故事也没有断言；`:36` 模块一侧的右键菜单没有点它的项。主控裁定在收尾补核，不转交、不以理由接受：这是 M3 写明“对接时逐个核对”交给 M6 的条款，补上便宜；照 P8 的 H1 放在默认套件与 CI 不跑的 `e2e/acceptance/`（默认套件仍 480 条），收尾 spec §3 记此裁定。补核的提交 `f27cce7a3e934d6b57c1d09d4fae04031d720123`（`e2e/acceptance/g19-m3-dialogs-and-filters.spec.ts`，三条，三种引擎）：三条在干净的代码上都过，没有产品缺陷；整个验收配置 33 例（11 条 × 三种引擎）过（`cl-acceptance`），新文件 `--repeat-each=3` 27 例过（`cl-acceptance-repeat3`）；各以只破坏该性质、能编译的变异证明（`cl-mut-backdrop`、`cl-mut-filters-focus`、`cl-mut-module-context-item`，各只让自己那一条在三种引擎失败）。

### 3.2 M2 第 4、5 节

| 条款（`M2-closeout.md`） | 阶段的处理结果 | 故事或收据 |
|---|---|---|
| `:34` §4 列表开着时项目变了 | P7a：外层以项目为 key，列表关上，再打开读新项目；旧项目迟到的回答不进新项目的列表 | `dropdowns/cycle/dropdown.test.tsx`、`dropdowns/module/dropdown.test.tsx` 的 “closes as its project changes …”；变异 `t5-mut-project-change-keeps-list`、`t7-mut-project-change-keeps-list`、`t5-mut-late-options` |
| `:35` §4 并发的两次取数，旧的回答后到 | P5a：规划 store 的 `Reconciled`，较旧的取数不写；P7a：`useDirectoryRead` 的请求号 | 变异 `t2-mut-older-fetch-writes`（P5a） |
| `:36` §4 `status` 是非字符串的假值时 `toLowerCase()` 抛出 | P5a、P7a：`status` 是生成的 `CycleStatus`，网页不再对它 `toLowerCase()`，选项的规则只比较 `status !== "completed"` | `core/lib/planning-directory.ts` 的 `cycleOptions`（P7a 的回执） |
| `:43` §5 (a) 迭代、模块下拉框的列表在按钮旁展开 | P7a：建在共用的 `CustomSearchSelect` 上，Plane 的 `cycle-options.tsx`、`module-options.tsx` 不恢复 | G4、G11 (page) 的 `expectListBesideButton`（`t3-story-greps`），三种引擎（P7a 的 `t14-three-engines-2`、`t14-repeat3-three-engines-2`、`t14-full-pages-2`） |
| `:44` §5 (b) `CustomSelect` 在模块的调用方 | P6a：`module-status-dropdown.tsx`（侧栏的页头、列表行与卡片共用）与 `select/status.tsx`（表单），`progress-sidebar/root.tsx` 不再直接用它 | G9 (page) 在侧栏改状态（等 nerve 确认、在途停用、被拒回到 nerve 的值），在新建弹窗里选 “In Progress”（`g9-create-module-page.spec.ts:173`）；组件测试 `select/status.test.tsx` |
| `:45` §5 (c) `BreadcrumbNavigationSearchDropdown` 在迭代、模块的页头 | P5a（迭代）、P6a（模块） | G19 的迭代页、模块页经它打开另一个 |
| `:46` §5 (d) 迭代、模块表单里的 `date-range.tsx` 与 `project/base.tsx` | 日期范围 P5a、P6a；项目下拉框的组件测试 P5a、P6a（变异 `t6-mut-project-any`、`t6-mut-project-picker-create-permissions`），浏览器核对 P8 | G1、G3、G9 (page) 在弹窗里选日期；`e2e/acceptance/g1-g9-form-project.spec.ts` 的 G1、G9 (acceptance) 三种引擎（`t11-acceptance-final`、`t12-acceptance`） |

## 4. 预检

一次 fresh 只读预检（`fable`），结论 Ready after fixes：四处 Important（I1 M6-M8-08 由收尾关闭；I2 P8 的归档没有核验收据；I3 M3 第 3 节的逐项汇总所指不确；I4 M3 第 3 节视图的三项没有落点）与十一处 Minor 都只涉及文档，全部接受、改入收尾的 spec 与 plan（`74486ef9`），没有须改产品代码或测试的事，也没有须负责人决定的事。落点：I1 → T2 的 Step 3（`t2-m6-m8-08-lines`）；I2 → T1 的 `t1-p8-archive-verify`；I3 → §3 与 [M3 的关闭段](../handoffs/M3-closeout.md)；I4 → M3 的关闭段与 M6-M7-04；Minor 的落点见 [re-80](../../../reviews/re-80-M6收尾预检.md) §2。

## 5. 集成

### 5.1 首次集成

本节在首次 `--no-ff` 与准确的 main CI 四个作业全绿之后由 T5 补写；此前 §2 各条不勾选。

### 5.2 第二次集成、归档与清理

第二次 main CI、收尾的全量归档与正常清理发生在被它们验证的提交之后，结果写在仓库外的集成收据：主目录的 `.superpowers/sdd/M6-closeout-integration-20261116/`（git 排除），与归档的 payload 并列而不在其中。第二次 main CI、归档与清理由它签结：设计 §13 的第 13、14 条与[收尾合同 §4](../specs/closeout.md#4-验收标准) 的最后两项以这份收据为准。
