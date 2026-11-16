# M6/closeout 完成标记、输入交接关闭与两轮集成实施计划

> 状态：已实施（2026-11-16）：经预检 [re-80](../../../reviews/re-80-M6收尾预检.md) 修正后实施；T0–T5 Step 1 完成，首次 noff `9440e0fc`（main CI 43871234997 attempt 2 全绿）；第二次集成、归档与清理由集成收据签结（[收尾记录](../reviews/closeout-integration.md) §5.2）。分支 `codex/m6-closeout`，基线 main `c4cd1880dc6dfc7d3f063b397de63a80e284ad28`（P8 的 noff，main CI 43851660778 attempt 1 全绿）。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 逐 Task 实施；本阶段只有文档与集成，不做整分支审查（P8 已做，见 re-79）。

**Goal:** 回填 P8 的完成证据、关闭七份输入交接与 M6-M8-08、为 §13 十四条给出最终证据，经两轮 `--no-ff` 与准确的 main CI 把 M6 标为已完成，最后归档清理并向负责人报告。
**Architecture:** 只改文档与关键词的 phase；最终事实写在仓库外的集成收据。
**Tech Stack:** 台账的 `doc-links.py`、`run-check.py`、`wait-run.sh`，`tools/keywords.mjs`、`make lint-web`、GitHub CI 与 P8 的归档工具。
**Spec:** [收尾合同](../specs/closeout.md)。

## Global Constraints

- 不改产品代码、测试、迁移或依赖；发现需要改代码的问题时停下，先提交已完成的部分再按纪律处理。
- 每次提交前从仓库根跑 `python3 .superpowers/sdd/M6-closeout/doc-links.py` 与 `node tools/keywords.mjs`；改 `tools/keywords.json` 后跑 `make lint-web`。检查经 `run-check.py` 留收据。
- 合并一律 `--no-ff`（`git merge --no-ff -F <file>`），推送用 `/usr/bin/git push`；主目录始终在 main。
- CI 规则见 spec §5；等 CI 用台账的 `wait-run.sh <branch> <完整 SHA>`（有上限）。
- 运行证据放 worktree 的 `.superpowers/sdd/M6-closeout/`（git 排除），由 T6 归档；第二次 main CI、归档与清理的收据放主目录的 `.superpowers/sdd/M6-closeout-integration-20261116/`，在归档 payload 旁边。

## Task 0：预检后的修正

- [ ] 预检 [re-80](../../../reviews/re-80-M6收尾预检.md) 的 I1–I4、m1–m11 改入 spec 与 plan，写预检记录；提交 `docs(M6/closeout): 预检 re-80 与 spec、plan 的修正`。

## Task 1（T1）：回填 P8 的完成证据并补完 P8 自身的勾选与状态

- [ ] Step 1：`python3 .superpowers/sdd/M6-closeout/run-check.py t1-p8-archive-verify python3 .superpowers/sdd/M6-closeout/archive.py verify /Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P8-integration-20261116`（应得 4747 个文件、539,117,657 字节、7271 个成员、unknown 0、manifest SHA-256 `511f280f…ab48fd`）。从 P8 的台账（归档 `payload/worktree/.superpowers/sdd/P8-final-acceptance/progress.md` 的 Task 15、Task 16 各行）、`ci-evidence/`（三个 waiter 的输出、三次运行的 jobs JSON、合入说明）、`manifest.json`、`cleanup.log` 与本收据读数，写 P8 阶段记录的状态行、§2（含 A14 一行）、§5（本机七项在 `0ef66139`，最终候选的分支 CI 43846387746 attempt 1 在 `352740ac`：两者之间只差五个文档文件；各作业的 id、起止与秒数）与 §7（noff `c4cd1880`、双父与 tree、main CI 43851660778 attempt 1、归档的数字、清理的时刻与步骤、`t1-p8-archive-verify` 即清理之后的再核；时刻照台账所记）。
- [ ] Step 2：P8 spec 的 A14 勾选并指向 §7，`:129` 一句改为“七份输入交接改 `status`、M7 与 M8 的交接保持 open”；P8 spec、plan 的状态行改为已集成；M6 设计的头部状态与“当前文档分支”一行、§14 的 P8 行（已完成：noff `c4cd1880`）与 closeout 行（进行中；不链接 T3 才建的收尾记录）；`frontend-changes.md` 的 M6 行写 P8 已集成；M8 交接 M6-M8-01 的 P8 一条补最终候选（e2e 1234 秒、web 595 秒、server 825/630 秒）与 main CI（e2e 1258 秒、web 602 秒、server 829/564 秒），换掉“由 M6 收尾的第一个 Task 补”的说法；`tools/keywords.json` 的 phase 设为 `M6/closeout`。
- [ ] Step 3：跑 `make lint-web`、`doc-links.py` 与 `node tools/keywords.mjs`（经 `run-check.py`），提交 `docs(M6/closeout): 回填 P8 的完成证据`。

## Task 2（T2）：关闭七份输入交接与 M6-M8-08

- [ ] Step 1：逐份读原文的条款与各阶段的“处理结果（M6/P<k>）”，核每条都有结论；转交项核对其 M7、M8 `M6-closeout` 的锚点（`M6-M7-0n`、`M6-M8-nn`）与 M7 自己的 `M3-closeout` §3 存在。
- [ ] Step 2：七份的 `status: open` 改为 `closed`，各加“关闭（M6/closeout）”一段：结论与链接，不改原文与处理结果；M5 的一份写明 M5-M6-01 以“未触发”关闭、条件转 M6-M7-09，M5-04 原文转 M6-M7-08；M3 的一份写明第 3 节的逐项汇总在收尾记录（T3），第 3 节视图的三项（`:32-33` 视图弹窗、`:36` 视图列表的右键菜单、`:37` 视图页头的 `CustomSearchSelect`）M6 走不到、转 M7，M6-M7-04 加一行指向 M7 的 `M3-closeout` §3。两份输出交接保持 open。M6 设计 §11 输入表之后“保持 open 至收尾关闭”一句注明已关闭。
- [ ] Step 3：M6-M8-08：`run-check.py t2-m6-m8-08-lines wc -l` 六个文件（`store/issue/root.store.ts`、`issue-writes.ts`、`issue-write-fields.ts`、`helpers/issue-list-binding.ts`、`issue-list-read.ts`、`issue-list-catalog.ts`，路径照该条），都在该条的上限之内时在该条下加“处理结果（M6/closeout）：已核对，关闭”，文件级的 `status` 仍 open。
- [ ] Step 4：跑 `doc-links.py`；提交 `docs(M6/closeout): 关闭七份输入交接与 M6-M8-08`。

## Task 3（T3）：§13 的证据映射

- [ ] Step 1：新建 `docs/v0/M6-cycles-modules/reviews/closeout-integration.md`：各阶段的合并、main CI 与归档根（P1–P8 与各修复分支）；§13 十四条逐条列证据（阶段记录、整分支审查、P8 的 G1–G20 表与设计 §13 的对照、收据名），此时不勾选；第 8、9、10 条写明“本领域”的范围（spec T3）；M3 第 3 节与 M2 第 4、5 节的逐项表（条款 → 阶段的处理结果 → 故事或收据）；本阶段预检的结论与集成收据的路径，写明“第二次 main CI、归档与清理由它签结”。M6 设计 §14 的 closeout 行此时链接它。
- [ ] Step 2：跑 `doc-links.py`；提交 `docs(M6/closeout): 列出完成标准的最终证据`。

## Task 3a：补核 M3 交接第 3 节的三项（主控在 T3 之后的裁定，spec §3）

- [ ] Step 1：一个实现者照 P8 的 e2e 规则在 `e2e/acceptance/` 加一个文件：迭代、模块的弹窗点外面关上（`:32`）、筛选按钮 Escape 之后焦点回到它（`:38`）、模块一行右键菜单点它的项（`:36`）；三种引擎、`--repeat-each=3`、各一个变异；默认套件仍 480；提交 `test(M6/closeout): …`。在干净的代码上失败即停下评估。
- [ ] Step 2：收尾记录 §2 第 9、11 条与 §3.1、M3 交接的关闭段、README 与 M6-M8-17 的条数随之改写；跑 `doc-links.py` 与关键词；提交 `docs(M6/closeout): …`。

## Task 4（T4）：首次集成

- [ ] Step 1：串行的本地检查 `doc-links.py`、`node tools/keywords.mjs`、`make lint-web`；推送分支，核准确 HEAD 的 server 两路、web、e2e 全绿（中途推送被取消的运行不计）。
- [ ] Step 2：核主目录 main 干净且等于 origin/main、worktree 干净、本地与远端分支都是该候选；`--no-ff` 合入，核双父与 tree 一致，推送。
- [ ] Step 3：核准确的 main CI 全绿，保存 run、attempt 与 jobs 的原件。

## Task 5（T5）：完成的勾选与第二次集成

- [ ] Step 1：分支快进到首次的 noff（`git merge --ff-only main`）。§13 十四条逐条勾选，每条附最终证据并写首次 main CI，“本节现在全未完成”一句改写；最后两条写明 P1–P8 已成立的事实与本阶段集成收据的路径；收尾记录补首次集成的事实；总体 §9.4 的 M6 改为“已完成”并链接设计与收尾记录；M6 设计的头部与 §14 的 closeout 行、`frontend-changes.md` 的 M6 行改为已完成；收尾 spec 与 plan 的状态行，spec §4 勾选前三项（后两项由收据签结）。
- [ ] Step 2：跑 `doc-links.py` 与关键词守卫；提交 `docs(M6/closeout): 标记 M6 完成`；推送并核准确的分支 CI 全绿。
- [ ] Step 3：第二次 `--no-ff` 合入并推送；核准确的 main CI 全绿，run、attempt 与 jobs 写入集成收据。

## Task 6（T6）：归档、清理与负责人报告

- [ ] Step 1：用 worktree 中的 `archive.py` 归档收尾的全部 ignored 原件，核 unknown 为 0。
- [ ] Step 2：`origin/main..branch` 与 `origin/main..origin/branch` 为 0 之后正常删除远端分支、worktree 与本地分支；从归档副本再核全部成员一致；清理收据写在集成收据目录。
- [ ] Step 3：向负责人提交最终报告：M6 完成（两次 main CI 的 run、attempt、jobs，归档的数字）、各阶段的根因修复、两份输出交接的条目、负责人的开放问题（web 作业的期限与 e2e 的作业结构）与 CI 的余量。
