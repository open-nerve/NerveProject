# M5/closeout：完成标记、输入交接关闭与两轮集成

> 2026-11-06：预检通过，12项局部问题已补实（[re-53](../../../reviews/re-53-M5收尾预检.md)）。基线main`d8ec845750e61b4d1970aec58cb57cb72e3685ab`（P8 noff）；P8准确分支CI 42188195625（attempt 3）、main CI 42212370678（attempt 2）三项全绿，归档与清理已完成，完整记录由本阶段T1回填到P8阶段记录T16段。

## 1. 目标与边界

按[M5设计§12](../M5-design.md#12-分阶段推进)的closeout行与§14全部十八条完成标准收尾：回填P8完成证据并补完P8自身的勾选与状态，关闭七份输入交接，为§14每条给出可复核的最终证据；首次`--no-ff`合入并等准确main CI全绿后，才逐条勾选§14、把[总体设计§9.4](../../v0-design.md#94-里程碑进度表)的M5改为“已完成”，经第二次`--no-ff`与main CI全绿，最后归档、正常清理并向负责人报告。

不改产品代码、API、迁移、依赖或测试；只改文档与`tools/keywords.json`的phase。P8已完成最终验收矩阵、演练、deep-review（[re-52](../../../reviews/re-52-M5P8整体审核.md)）与交接，本阶段不重做。

## 2. 交付物与Task

| Task | 交付 | 验证重点 |
|---|---|---|
| T1 | P8阶段记录新增T16段：准确候选、分支CI三次运行与负责人2026-11-06同意的第二次重跑、双父合并、main CI两次运行、归档数字与清理收据（逐项引用P8台账与原件）；P8 spec A1–A8与P8 plan T15 Step 3、T16 Step 1–2勾选（T16 Step 3在T5勾选）；P8 spec/plan/阶段记录的头部与阶段记录§2、M5设计头部（分支与授权行）改写；M5设计§15的P8行为已完成、closeout行为进行中；`docs/v0/frontend-changes.md`的M5行为P8已集成，M4行按其已完成的事实更正；`tools/keywords.json`的phase设为`M5/closeout` | 数字均来自收据与CI；历史失败与重跑全部保留；不提前写M5完成 |
| T2 | 七份输入交接`status: closed`，各加“关闭（M5/closeout）”一段（M2、M3引用plane-diff中users/workspaces/projects三行；M1的图标登记条件因目录已删而不再适用；转交链接指向三份输出交接的`M5-04`锚点）；M8交接M5-M8-06补两条观察：M4 C12 (review)故事的Escape弹窗偶发、CI浏览器安装停滞与e2e作业余量；M5设计§13改写关闭事实 | 逐条核对处理结果覆盖原文全部条款；三份输出交接保持open、不改称已实现 |
| T3 | 收尾记录`reviews/closeout-integration.md`：各阶段合并与归档、§14十八条逐条的证据（此时不勾选）、本阶段预检结论、仓库外集成收据的路径 | 每条指向具体证据；未满足项如实保留 |
| T4 | 首次集成：本地检查，推送分支并核准确分支CI，`--no-ff`合入main并核准确main CI三项全绿 | 双父与tree一致；不以分支CI代替main CI |
| T5 | §14十八条逐条勾选并附最终证据（含首次main CI）；第17、18条以P1–P8已成立的事实加本阶段集成收据路径勾选；总体§9.4的M5改为“已完成”，同文件“暂时使用packages/services”一句更正；M5设计头部、§15 closeout行与`frontend-changes.md`的M5行改为已完成；P8 plan T16 Step 3勾选；第二次`--no-ff`与main CI三项全绿，结果写入集成收据 | 勾选只在首次main CI成功之后；第二次main CI的结果不写进被它验证的提交 |
| T6 | 归档收尾全部ignored原件并核hash/bytes/mode/membership、unknown为0；`origin/main..branch`为0后正常删除远端与本地分支和worktree，归档再核；向负责人提交最终报告 | 不用force/stash/clean/reset-hard；收据位于冻结payload之外 |

## 3. 局部裁定

- 收尾的最终事实（第二次main CI、归档与清理）无法写进被它验证的提交，按M4先例写在仓库外的集成收据：主目录`.superpowers/sdd/M5-closeout-integration-20261106/`（git排除），与归档payload并列而不在其中，T6 Step 1冻结之后产生的文件都放在payload旁边。收尾spec §4的最后两项因此在仓库内保持未勾选，由收据签结。
- 输入交接关闭以P8 T13写的“处理结果（M5/P8）”为逐条依据，不另建交接索引；关闭段只写结论与链接，原文与处理结果不改。预检已核七份的每条都有结论。
- 本阶段只有文档与关键词phase的变更。本地检查：`python3 .superpowers/sdd/M5-closeout/doc-links.py`（工作区工具，SHA-256 `166196eb…`，从仓库根运行）、`node tools/keywords.mjs`与`make lint-web`（关键词phase在其中检查，worktree已`pnpm install --frozen-lockfile`）；Go与E2E由准确分支/main CI覆盖。`tools/keywords.json`没有在`M5/closeout`到期的例外（最早为M6）。
- 运行证据与原件放worktree的`.superpowers/sdd/M5-closeout/`（台账`progress.md`、`run-check.py`收据、预检原文），由T6归档；归档工具`archive.py`取自P8归档，其SHA-256 `6e855b19…`与P8`create-receipt.json`的`tool_sha256`一致。

## 4. 验收标准

- [ ] P8完成证据回填（含全部失败与重跑），P8自身的勾选与状态补完，数字与收据一致。
- [ ] 七份输入交接closed，每条有落点；三份输出open且未改称已实现；M8交接补两条观察。
- [ ] §14十八条各有证据；首次main CI全绿后才勾选。
- [ ] 总体§9.4的M5为“已完成”；两次`--no-ff`的准确main CI三项全绿（第二次由集成收据签结）。
- [ ] 归档核验unknown为0，分支与worktree正常清理，清理后归档再核一致（由集成收据签结）。

## 5. 风险与CI规则

文档链接与锚点在关闭段中断：每次提交前跑`doc-links.py`。CI规则按负责人规定执行：作业在25分钟期限处被取消且时间耗在依赖安装（2026-11-06两次如此）属环境失败，同一输入重跑一次并保留原件（`gh run view <id> --attempt N --json …`与`--log`）；重跑仍失败即停下报告。测试本身失败（如P8遇到的M4 C12 Escape弹窗偶发）不是环境失败，先看原件再按根因或报告负责人处理。`timeout-minutes`与预算不动。非main分支的CI有`cancel-in-progress`，推送新提交取消旧运行不算失败、也不计为重跑。绿色e2e作业已用23–24分钟（期限25分钟），这一余量写入M8交接与负责人报告。
