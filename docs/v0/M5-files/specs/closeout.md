# M5/closeout：完成标记、输入交接关闭与两轮集成

> 状态：2026-11-06：预检前草案。 基线main`d8ec845750e61b4d1970aec58cb57cb72e3685ab`（P8 noff）；P8准确分支CI 42188195625（attempt 3）、main CI 42212370678（attempt 2）三项全绿，归档与清理见[P8阶段记录T16](../reviews/P8-final-acceptance-review.md)。

## 1. 目标与边界

按[M5设计§12](../M5-design.md#12-分阶段推进)的closeout行与§14全部十八条完成标准收尾：回填P8完成证据，关闭七份输入交接，为§14每条给出可复核的最终证据；首次`--no-ff`合入并等准确main CI全绿后，才逐条勾选§14、把[总体设计§9.4](../../v0-design.md#94-里程碑进度表)的M5改为“已完成”，经第二次`--no-ff`与main CI全绿，最后归档并正常清理。

不改产品代码、API、迁移、依赖或测试；P8已完成最终验收矩阵、演练、deep-review（[re-52](../../../reviews/re-52-M5P8整体审核.md)）与交接。收尾中若CI失败，按根因处理；环境失败同输入重跑一次并保留原件；重跑仍失败则停止并向负责人报告。不改CI预算。

## 2. 交付物与Task

| Task | 交付 | 验证重点 |
|---|---|---|
| T1 | P8阶段记录补T16（合并、main CI、归档清单、清理收据）；M5设计头部与§15的P8行改为已完成；`tools/keywords.json`的phase设为`M5/closeout` | 数字均来自收据与CI；不提前写M5完成 |
| T2 | 七份输入交接`status: closed`，各加“关闭（M5/closeout）”一段：每条在“处理结果（M5/P8）”中有落点，转交项链接M6/M7/M8的`M5-closeout`锚点；三份输出交接保持open | 逐条核对处理结果覆盖原文全部条款；输出不因输入关闭改称已实现 |
| T3 | 收尾记录`reviews/closeout-integration.md`：§14十八条逐条的证据（阶段记录、整审、P8矩阵行、收据），此时不勾选 | 每条指向具体证据；未满足项如实保留 |
| T4 | 首次集成：与改动范围对应的本地检查，推送分支并核准确分支CI，`--no-ff`合入main并核准确main CI三项全绿 | 双父与tree一致；不以分支CI代替main CI |
| T5 | §14十八条逐条勾选并附最终证据（含首次main CI）；总体§9.4的M5改为“已完成”并链接收尾记录；第二次`--no-ff`与main CI三项全绿 | 勾选只在首次main CI成功之后；第二次main CI的结果写在仓库外的收据 |
| T6 | 归档收尾全部ignored原件并核hash/bytes/mode/membership、unknown为0；`origin/main..branch`为0后正常删除远端与本地分支和worktree，归档再核 | 不用force/stash/clean/reset-hard；收据位于冻结payload之外 |

## 3. 局部裁定

- 收尾的最终事实（第二次main CI、归档与清理）无法写进被它验证的提交，按M4先例写在仓库外的集成收据（归档根下），收尾记录只写到首次main CI与勾选为止，并指明收据位置。
- 输入交接关闭以P8 T13写的“处理结果（M5/P8）”为逐条依据，不另建交接索引；关闭段只写结论与链接，原文与处理结果不改。
- 本阶段只有文档变更：本地检查为`doc-links.py`、`node tools/keywords.mjs`与`make lint-web`（关键词phase在其中检查，worktree先`pnpm install --frozen-lockfile`）；Go与E2E由准确分支/main CI覆盖。`tools/keywords.json`没有在`M5/closeout`到期的例外（最早为M6）。

## 4. 验收标准

- [ ] P8完成证据回填，数字与收据一致。
- [ ] 七份输入交接closed，每条有落点；三份输出open且未改称已实现。
- [ ] §14十八条各有证据；首次main CI全绿后才勾选。
- [ ] 总体§9.4的M5为“已完成”；两次`--no-ff`的准确main CI三项全绿。
- [ ] 归档核验unknown为0，分支与worktree正常清理，清理后归档再核一致。

## 5. 风险

文档链接与锚点在关闭段中断：每次提交前跑`doc-links.py`。CI间歇（WebKit不在默认CI；锁测试与启动时周期任务的竞争已在P8修两项、其余暴露写入M8交接）：失败先看原件区分环境与测试，按P8的规则处理，不放宽断言。
