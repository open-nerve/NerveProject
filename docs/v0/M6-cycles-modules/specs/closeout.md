# M6/closeout：完成标记、输入交接关闭与两轮集成

> 状态：spec 与 plan 已写（2026-11-16），待预检。基线 main `c4cd1880dc6dfc7d3f063b397de63a80e284ad28`（P8 的 noff，双父 `ea971606`、`352740ac`；main CI 43851660778 attempt 1 三项全绿）。

## 1. 目标与边界

按 [M6 设计 §12](../M6-design.md) 的 closeout 一行与 §13 的十四条完成标准收尾：回填 P8 的完成证据并补完 P8 自身的勾选与状态，关闭七份输入交接，为 §13 每条给出可复核的最终证据；首次 `--no-ff` 合入并等准确的 main CI 全绿之后，才逐条勾选 §13、把[总体设计 §9.4](../../v0-design.md#94-里程碑进度表) 的 M6 改为“已完成”，经第二次 `--no-ff` 与 main CI 全绿，最后归档、正常清理并向负责人报告。先例是 [M5 的收尾](../../M5-files/specs/closeout.md)。

不改产品代码、API、迁移、依赖或测试；只改文档与 `tools/keywords.json` 的 phase。P8 已完成最终矩阵、整分支审查（[re-79](../../../reviews/re-79-M6P8整体审核.md)）与交接，本阶段不重做。

## 2. 交付物与 Task

| Task | 交付 | 验证重点 |
|---|---|---|
| T1 | P8 的回填：[P8 阶段记录](../reviews/P8-final-acceptance-review.md) §5 的最终候选分支 CI（43846387746）与 §7（noff `c4cd1880`、main CI 43851660778、归档数字、清理收据，逐项引用 P8 台账与归档里的原件）；P8 spec A14 勾选；P8 spec、plan 与阶段记录的状态行改为已集成；M6 设计的头部（状态与“当前文档分支”一行）、§14 的 P8 行（已完成）与 closeout 行（进行中）；`docs/v0/frontend-changes.md` 的 M6 行（P8 已集成）；M8 交接 M6-M8-01 的最终候选与 main CI 的数据点（P8 的 T15 写明由收尾补）；`tools/keywords.json` 的 phase 设为 `M6/closeout` | 数字都来自收据与 CI 原件；历史失败与重跑照实保留；不提前写 M6 完成 |
| T2 | 七份输入交接（`handoffs/` 下 M1-P2、M1-P3、M1、M2、M3、M4、M5 的收尾交接）`status: closed`，各加“关闭（M6/closeout）”一段：结论与链接，不改原文与各阶段的处理结果；转交项核对其在 M7、M8 的 `M6-closeout` 的锚点存在；M5-M6-01 以“未触发”关闭、条件已原文转交 M7（M6-M7-09）；两份输出交接（M7、M8 的 `M6-closeout`）保持 open、不改称已实现；M6 设计 §11 的输入表注明已关闭 | 逐条核对处理结果覆盖原文的全部条款；每个转交的锚点存在 |
| T3 | 收尾记录 `reviews/closeout-integration.md`：各阶段的合并、main CI 与归档根；§13 十四条逐条的证据（阶段记录、整分支审查、P8 最终矩阵的行与收据名），此时不勾选；本阶段预检的结论；仓库外集成收据的路径 | 每条指向具体证据；未满足的照实保留 |
| T4 | 首次集成：本地检查，推送分支并核准确的分支 CI，`--no-ff` 合入 main 并核准确的 main CI 三项全绿 | 双父与 tree 一致；不以分支 CI 代替 main CI |
| T5 | §13 十四条逐条勾选并附最终证据（含首次 main CI）；最后两条以 P1–P8 已成立的事实加本阶段集成收据的路径勾选；总体 §9.4 的 M6 改为“已完成”并链接设计与收尾记录；M6 设计头部与 §14 的 closeout 行、`frontend-changes.md` 的 M6 行改为已完成；本 spec §4 勾选前三项；第二次 `--no-ff` 与 main CI 三项全绿，结果写入集成收据 | 勾选只在首次 main CI 成功之后；第二次 main CI 的结果不写进被它验证的提交 |
| T6 | 归档收尾的全部 ignored 原件并核 hash、bytes、mode、membership，unknown 为 0；`origin/main..branch` 为 0 之后正常删除远端与本地分支和 worktree，清理之后归档再核；向负责人提交最终报告 | 不用 force、stash、clean、reset --hard；收据在冻结的 payload 之外 |

## 3. 局部裁定

- 收尾的最终事实（第二次 main CI、归档与清理）无法写进被它验证的提交，照 M5 的先例写在仓库外的集成收据：主目录的 `.superpowers/sdd/M6-closeout-integration-20261116/`（git 排除），与归档 payload 并列而不在其中；T6 冻结之后产生的文件都放在 payload 旁边。本 spec §4 的最后两项因此在仓库内保持未勾选，由收据签结。
- 输入交接的关闭以各阶段写的“处理结果（M6/P<k>）”为逐条依据（P8 的 T13 已为每份写了最终的处理结果），不另建交接索引；关闭段只写结论与链接。
- 本阶段只有文档与关键词 phase 的变更。本地检查：`python3 .superpowers/sdd/M6-closeout/doc-links.py`（取自 P8 的台账，从仓库根运行）、`node tools/keywords.mjs` 与 `make lint-web`（phase 在其中检查，worktree 已 `pnpm install --frozen-lockfile`）；Go 与 e2e 由准确的分支、main CI 覆盖。`tools/keywords.json` 没有在 `M6/closeout` 或更早到期的例外（最早为 `M7/closeout`），§13 的“无到期例外”照此成立。
- 运行证据放 worktree 的 `.superpowers/sdd/M6-closeout/`（台账 `progress.md`、`run-check.py` 的收据、预检的原文），由 T6 归档；归档工具 `archive.py` 取自 P8 的台账（SHA-256 记入台账）。
- 本阶段不派实现者、不做整分支审查（P8 已做，re-79）；预检一次（`fable`，只读）。

## 4. 验收标准

- [ ] P8 的完成证据回填（含全部失败与重跑），P8 自身的勾选与状态补完，数字与收据一致。
- [ ] 七份输入交接 closed，每条有落点；两份输出交接 open 且未改称已实现。
- [ ] §13 十四条各有证据；首次 main CI 全绿之后才勾选。
- [ ] 总体 §9.4 的 M6 为“已完成”；两次 `--no-ff` 的准确 main CI 三项全绿（第二次由集成收据签结）。
- [ ] 归档核验 unknown 为 0，分支与 worktree 正常清理，清理之后归档再核一致（由集成收据签结）。

## 5. 风险与 CI 规则

文档链接与锚点在关闭段中断：每次提交前跑 `doc-links.py`。CI 规则照负责人的规定：已知的间歇（M6-M8-01 的 C7）只重跑失败的作业一次并记录；作业因环境（依赖安装停滞、runner 故障）失败，同一输入重跑一次并保留原件；重跑仍失败即停下报告；测试本身的失败先看原件、按根因处理或报告负责人；`timeout-minutes`、预算与作业结构不动；e2e 的估算、上界或实测超过 30 分钟时停下报告（本阶段不改测试与产品，e2e 照 P8 的 20–21 分钟）。非 main 分支的 CI 有 `cancel-in-progress`，推送新提交取消旧运行不算失败、也不计为重跑。负责人的开放问题（web 作业 15 分钟的期限与 e2e 的作业结构）写进最终报告。
