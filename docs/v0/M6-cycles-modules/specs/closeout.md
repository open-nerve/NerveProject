# M6/closeout：完成标记、输入交接关闭与两轮集成

> 状态：已实施（2026-11-16）：经预检 [re-80](../../../reviews/re-80-M6收尾预检.md) 修正后实施（I1–I4、m1–m11 全部改入）；首次 noff `9440e0fc`（main CI 43871234997 attempt 2 全绿），§4 的前三项已勾选，后两项由集成收据签结（[收尾记录](../reviews/closeout-integration.md) §5.2）。基线 main `c4cd1880dc6dfc7d3f063b397de63a80e284ad28`（P8 的 noff，双父 `ea971606`、`352740ac`；main CI 43851660778 attempt 1 四个作业全绿）。

## 1. 目标与边界

按 [M6 设计 §12](../M6-design.md) 的 closeout 一行与 §13 的十四条完成标准收尾：回填 P8 的完成证据并补完 P8 自身的勾选与状态，关闭七份输入交接与 M6-M8-08，为 §13 每条给出可复核的最终证据；首次 `--no-ff` 合入并等准确的 main CI 全绿之后，才逐条勾选 §13、把[总体设计 §9.4](../../v0-design.md#94-里程碑进度表) 的 M6 改为“已完成”，经第二次 `--no-ff` 与 main CI 全绿，最后归档、正常清理并向负责人报告。先例是 [M5 的收尾](../../M5-files/specs/closeout.md)。

不改产品代码、API、迁移、依赖或测试；只改文档与 `tools/keywords.json` 的 phase。P8 已完成最终矩阵、整分支审查（[re-79](../../../reviews/re-79-M6P8整体审核.md)）与交接，本阶段不重做。

## 2. 交付物与 Task

| Task | 交付 | 验证重点 |
|---|---|---|
| T1 | P8 的回填：先以 `archive.py verify` 再核 P8 的归档（收据 `t1-p8-archive-verify`，即 P8 清理之后的归档再核，P8 的 T1 对 P7c 的同一做法）；[P8 阶段记录](../reviews/P8-final-acceptance-review.md)的状态行、§2（含 A14 一行）、§5（本机七项在 `0ef66139`、最终候选的分支 CI 43846387746 在 `352740ac`，两者之间只差五个文档文件）与 §7（noff `c4cd1880`、双父与 tree、main CI 43851660778、归档的数字与清理，逐项引用 P8 台账、`ci-evidence/`、`manifest.json`、`cleanup.log` 与 `t1-p8-archive-verify`；时刻照台账所记）；P8 spec A14 勾选，`:129` 一句改为“七份输入交接改 `status`、M7 与 M8 的交接保持 open”；P8 spec、plan 的状态行；M6 设计的头部（状态与“当前文档分支”一行）、§14 的 P8 行（已完成）与 closeout 行（进行中，此时不链接 T3 才建的收尾记录）；`docs/v0/frontend-changes.md` 的 M6 行（P8 已集成）；M8 交接 M6-M8-01 的最终候选与 main CI 的数据点；`tools/keywords.json` 的 phase 设为 `M6/closeout` | 数字都来自收据与 CI 原件；历史失败与重跑照实保留；不提前写 M6 完成 |
| T2 | 七份输入交接（`handoffs/` 下 M1-P2、M1-P3、M1、M2、M3、M4、M5 的收尾交接）`status: closed`，各加“关闭（M6/closeout）”一段：结论与链接，不改原文与各阶段的处理结果；转交项核对其在 M7、M8 的 `M6-closeout` 的锚点存在；M5-M6-01 以“未触发”关闭、条件已原文转交 M7（M6-M7-09）。M3 的关闭段另写：第 3 节的逐项浏览器核对汇总在收尾记录（T3）而不在 P8 的 §4.3（P8 的处理结果那一句所指不确）；第 3 节视图的三项——视图弹窗的 Tab、Escape、焦点与名字（`M3-closeout.md:32-33`）、视图列表的右键菜单（`:36`）、视图页头 `CustomSearchSelect` 的键盘（`:37`）——M6 走不到（视图页没有读），转 M7，M6-M7-04 加一行指向 M7 自己的 [M3-closeout §3](../../M7-collaboration/handoffs/M3-closeout.md#3-键盘弹窗和下拉框)。**M6-M8-08**：P7 之后以 `wc -l` 再核根与写入单元的六个文件仍在上限之内（收据 `t2-m6-m8-08-lines`），在该条下加“处理结果（M6/closeout）：已核对，关闭”，文件级的 `status` 仍 open。两份输出交接（M7、M8 的 `M6-closeout`）保持 open、不改称已实现；M6 设计 §11 的“保持 open 至收尾关闭”注明已关闭 | 逐条核对处理结果覆盖原文的全部条款；每个转交的锚点存在 |
| T3 | 收尾记录 `reviews/closeout-integration.md`：各阶段的合并、main CI 与归档根；§13 十四条逐条的证据（阶段记录、整分支审查、P8 最终矩阵的行与收据名），此时不勾选；第 8、9、10 条照 P8 的对照（`P8-final-acceptance-review.md:315-317`）写明“本领域”的范围——别的领域的旧服务（收藏、视图、收件箱、webhook、通知仍调旧地址）、M7 与 M8 的 `copy-hook` 例外、`--rows M6` 的 245 行（都有读者）各归其 M；M3 第 3 节与 M2 第 4、5 节（`M2-closeout.md:37`、`:47` 的条件同为“M6 的评审写明”）的逐项表：条款 → 阶段的处理结果 → 故事或收据；本阶段预检的结论；仓库外集成收据的路径 | 每条指向具体证据；未满足的照实保留 |
| T4 | 首次集成：本地检查，推送分支并核准确的分支 CI，`--no-ff` 合入 main 并核准确的 main CI 四个作业全绿 | 双父与 tree 一致；不以分支 CI 代替 main CI |
| T5 | 先把分支快进到首次的 noff（`git merge --ff-only main`，M5 的先例），再写完成标记：§13 十四条逐条勾选并附最终证据（含首次 main CI），同节“本节现在全未完成”一句改写；最后两条以 P1–P8 已成立的事实加本阶段集成收据的路径勾选；总体 §9.4 的 M6 改为“已完成”并链接设计与收尾记录；M6 设计头部与 §14 的 closeout 行、`frontend-changes.md` 的 M6 行改为已完成；本 spec 与 plan 的状态行，本 spec §4 勾选前三项；第二次 `--no-ff` 与 main CI 四个作业全绿，结果写入集成收据 | 勾选只在首次 main CI 成功之后；第二次 main CI 的结果不写进被它验证的提交 |
| T6 | 归档收尾的全部 ignored 原件并核 hash、bytes、mode、membership，unknown 为 0；`origin/main..branch` 为 0 之后正常删除远端与本地分支和 worktree，清理之后归档再核；向负责人提交最终报告 | 不用 force、stash、clean、reset --hard；收据在冻结的 payload 之外 |

## 3. 局部裁定

- 收尾的最终事实（第二次 main CI、归档与清理）无法写进被它验证的提交，照 M5 的先例写在仓库外的集成收据：主目录的 `.superpowers/sdd/M6-closeout-integration-20261116/`（git 排除），与归档 payload 并列而不在其中；T6 冻结之后产生的文件都放在 payload 旁边。本 spec §4 的最后两项因此在仓库内保持未勾选，由收据签结。
- 输入交接的关闭以各阶段写的“处理结果（M6/P<k>）”为逐条依据（P8 的 T13 已为每份写了最终的处理结果），不另建交接索引；关闭段只写结论与链接，处理结果所指不确之处在关闭段更正（预检 I3），不改原文。
- 本阶段只有文档与关键词 phase 的变更。本地检查：`python3 .superpowers/sdd/M6-closeout/doc-links.py`（取自 P8 的台账，从仓库根运行）、`node tools/keywords.mjs` 与 `make lint-web`（phase 在其中检查，worktree 已 `pnpm install --frozen-lockfile`）；Go 与 e2e 由准确的分支、main CI 覆盖。`tools/keywords.mjs` 接受 `M6/closeout` 并排在全部编号阶段之后，phase 只影响例外的到期；`tools/keywords.json` 的七条例外最早到 `M7/closeout`，§13 的“无到期例外”照此成立。
- 运行证据放 worktree 的 `.superpowers/sdd/M6-closeout/`（台账 `progress.md`、`run-check.py` 的收据、预检的原文、有上限的 CI 等待 `wait-run.sh`：排队 0 个作业约 20 分钟或总计约 70 分钟即停），由 T6 归档；归档工具 `archive.py` 取自 P8 的台账（SHA-256 记入台账）。
- 本阶段不做整分支审查（P8 已做，re-79）；预检一次（`fable`，只读，[re-80](../../../reviews/re-80-M6收尾预检.md)）。
- **补核 M3 交接第 3 节的三项**（主控在 T3 之后的裁定）：T3 写逐项表时发现 `:32` 点外面关上迭代、模块的弹窗、`:38` Escape 之后焦点回到筛选的按钮在 M6 的页面上没有断言，`:36` 模块一侧的右键菜单没有点它的项。这是 M3 写明“对接时逐个核对”交给 M6 的条款，补上便宜，在本阶段补核、不转交：放在默认套件与 CI 不跑的 `e2e/acceptance/`（P8 的 H1，默认套件仍 480 条），由一个实现者照 P8 的 e2e 规则写、各以变异证明；§1 的“不改测试”为此收窄为“不改产品代码、不改默认套件与 CI”。核对在干净的代码上失败即为产品缺陷，须停下评估（实际三条都过，`f27cce7a`）。

## 4. 验收标准

- [x] P8 的完成证据回填（含全部失败与重跑与 P8 归档的再核），P8 自身的勾选与状态补完，数字与收据一致。
- [x] 七份输入交接 closed、M6-M8-08 关闭，每条有落点；两份输出交接 open 且未改称已实现。
- [x] §13 十四条各有证据；首次 main CI 全绿之后才勾选。
- [ ] 总体 §9.4 的 M6 为“已完成”；两次 `--no-ff` 的准确 main CI 四个作业全绿（第二次由集成收据签结）。
- [ ] 归档核验 unknown 为 0，分支与 worktree 正常清理，清理之后归档再核一致（由集成收据签结）。

## 5. 风险与 CI 规则

文档链接与锚点在关闭段中断：每次提交前跑 `doc-links.py`。CI 规则照负责人的规定：已知的间歇（M6-M8-01 的 C7）只重跑失败的作业一次并记录；web 作业在 15 分钟的期限处被取消（P7c 时 902 秒，慢的 runner）照 P7c 的先例只重跑失败的作业一次并在报告里告诉负责人；作业因别的环境原因（依赖安装停滞、runner 故障）失败，同一输入重跑一次并保留原件；重跑仍失败即停下报告；测试本身的失败先看原件、按根因处理或报告负责人；`timeout-minutes`、预算与作业结构不动；e2e 的估算、上界或实测超过 30 分钟时停下报告（本阶段不改测试与产品，e2e 照 P8 的 20–21 分钟）。非 main 分支的 CI 有 `cancel-in-progress`，推送新提交取消旧运行不算失败、也不计为重跑。负责人的开放问题（web 作业 15 分钟的期限与 e2e 的作业结构）写进最终报告。
