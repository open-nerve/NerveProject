# M5/closeout 完成标记、输入交接关闭与两轮集成 Implementation Plan

> 2026-11-06：预检通过，12项局部问题已补实（[re-53](../../../reviews/re-53-M5收尾预检.md)）；T1–T4完成，首次集成main `1831dabc`（[main CI 42229175694](https://github.com/open-nerve/NerveProject/actions/runs/42229175694)（attempt 2）三项全绿）。分支`codex/m5-closeout`，基线main`d8ec845750e61b4d1970aec58cb57cb72e3685ab`（P8 noff，main CI 42212370678 attempt 2）。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 逐Task内联实施；本阶段只有文档与集成，不派实现者，不做整分支deep-review（P8已做，见re-52）。

**Goal:** 回填P8完成证据、关闭七份输入交接、为§14十八条给出最终证据，经两轮`--no-ff`与准确main CI把M5标为已完成，最后归档清理并向负责人报告。
**Architecture:** 只改文档与关键词phase；最终事实写在仓库外的集成收据。
**Tech Stack:** 工作区`doc-links.py`、`tools/keywords.mjs`、`make lint-web`、GitHub CI与P8的归档工具。
**Spec:** [收尾合同](../specs/closeout.md)。

## Global Constraints

- 不改产品代码、测试、迁移或依赖；发现需要改代码的问题时停下，先提交已完成部分再按纪律处理。
- 每次提交前从仓库根跑`python3 .superpowers/sdd/M5-closeout/doc-links.py`与`node tools/keywords.mjs`；改`tools/keywords.json`后跑`make lint-web`。检查经`run-check.py`留收据。
- 合并一律`--no-ff`，推送用`/usr/bin/git push`；主目录始终在main。
- CI规则见spec §5：依赖安装耗尽期限的取消属环境失败，同输入重跑一次并保留原件；重跑仍失败停下报告；测试失败不当环境处理；不改预算；`cancel-in-progress`取消不计。
- 运行证据放worktree的`.superpowers/sdd/M5-closeout/`（git排除），由T6归档；第二次main CI、归档与清理的收据放主目录`.superpowers/sdd/M5-closeout-integration-20261106/`，在归档payload旁边。

## Task 1（T1）：回填P8完成证据并补完P8自身的勾选与状态

- [x] Step 1：从P8台账（P8归档`payload/worktree/.superpowers/sdd/P8-final-acceptance/progress.md`第119–123行及其后的T16行）、CI原件（`session-scratchpad/ci-42188195625-attempt{1,2,3}*`、`ci-42212370678-attempt{1,2}*`、`p8-t16-merge.log`）与归档收据读数，写P8阶段记录T16段：准确候选`d0513ff9`；分支CI三次运行（attempt 1依赖安装停滞在期限处取消、attempt 2 M4 C12 Escape弹窗偶发、负责人2026-11-06同意的attempt 3全绿及本地重复收据`t16-c12-repeat`）；双父合并`d8ec8457`；main CI两次运行（attempt 1同样取消、按规则重跑一次的attempt 2全绿，e2e作业24.1分钟）；归档文件数/字节/成员/manifest SHA-256与清理收据。
- [x] Step 2：P8 spec A1–A8逐项勾选并指向§7的证据；P8 plan T15 Step 3与T16 Step 1–2勾选；P8 spec、plan与阶段记录的头部及阶段记录§2改写为已集成；M5设计头部的状态与“当前文档分支”一行、§15的P8行（已完成）与closeout行（进行中）改写；`frontend-changes.md`的M5行为“P8 noff main d8ec8457已集成”，M4行按M4已完成的事实更正；`tools/keywords.json`的phase设为`M5/closeout`。
- [x] Step 3：跑`make lint-web`与`doc-links.py`（经`run-check.py`），提交`docs(M5/closeout): 回填P8完成证据`。

## Task 2（T2）：关闭七份输入交接并补M8观察

- [x] Step 1：逐份读原文条款与“处理结果（M5/P8）”，核每条都有结论；转交项核对其M6/M7/M8`M5-closeout`的锚点（`M5-04`等）存在。
- [x] Step 2：七份`status: open`改为`closed`，各加“关闭（M5/closeout）”段：结论与链接，不改原文与处理结果；M2、M3的段落引用`docs/v0/plane-diff.md`中users/workspaces/projects三行；M1的段落写明附件图标目录已删、图标登记条件不再适用；三份输出交接保持open。
- [x] Step 3：M8交接M5-M8-06补两条：M4 C12 (review)故事的Escape弹窗偶发（2026-11-06首次，本地40次未复现，原因未查实）；CI的Playwright浏览器安装停滞两次使e2e作业在25分钟期限处取消，绿色e2e作业已用23–24分钟。M5设计§13“当前所有输入仍为open”一句改写为关闭事实，§13末段“保持open至收尾”后注明已关闭。
- [x] Step 4：跑`doc-links.py`；提交`docs(M5/closeout): 关闭七份输入交接`。

## Task 3（T3）：§14证据映射

- [x] Step 1：新建`docs/v0/M5-files/reviews/closeout-integration.md`：各阶段合并与归档根、§14十八条逐条列证据（阶段记录、整审、P8矩阵行与收据名），此时不勾选；写预检结论（[re-53](../../../reviews/re-53-M5收尾预检.md)，12项局部问题的落点）与集成收据的路径，写明“第二次main CI、归档与清理由它签结”。
- [x] Step 2：跑`doc-links.py`；提交`docs(M5/closeout): 列出完成标准的最终证据`。

## Task 4（T4）：首次集成

- [x] Step 1：串行本地检查`doc-links.py`、`node tools/keywords.mjs`、`make lint-web`；推送分支，核准确HEAD的server/web/e2e全绿（中途推送被取消的运行不计）。
- [x] Step 2：核主目录main干净且等于origin/main、worktree干净、本地与远端分支都是该候选；`--no-ff`合入，核双父与tree一致，推送。
- [x] Step 3：核准确main CI三项全绿，保存run/attempt/jobs原件。

## Task 5（T5）：完成勾选与第二次集成

- [x] Step 1：§14十八条逐条勾选，每条附最终证据并写首次main CI；第17、18条写明P1–P8已成立的事实与本阶段集成收据路径；收尾记录补首次集成事实；总体§9.4的M5改为“已完成”并链接设计与收尾记录，同文件“暂时使用packages/services”一句更正为已删除；M5设计头部与§15 closeout行、`frontend-changes.md`的M5行改为已完成；P8 plan T16 Step 3勾选；收尾spec §4勾选前三项（后两项由收据签结）。
- [ ] Step 2：跑`doc-links.py`与关键词守卫；提交`docs(M5/closeout): 标记M5完成`；推送并核准确分支CI全绿。
- [ ] Step 3：第二次`--no-ff`合入并推送；核准确main CI三项全绿，run/attempt/jobs写入集成收据。

## Task 6（T6）：归档、清理与负责人报告

- [ ] Step 1：用worktree中的`archive.py`（SHA-256先与P8`create-receipt.json`的`tool_sha256`核对）归档收尾全部ignored原件与会话scratchpad，核unknown为0。
- [ ] Step 2：`origin/main..branch`与`origin/main..origin/branch`为0后正常删除远端分支、worktree与本地分支；从归档副本再核全部成员一致；清理收据写在集成收据目录。
- [ ] Step 3：向负责人提交最终报告：M5完成（两次main CI的run/attempt/jobs、归档数字）、跨M修复、re-52的deferred Minor、2026-11-06的两项负责人决定（共用S3服务、最终候选的第二次重跑）、M5-M8-06的开放观察（含新补两条）、CI作业余量。
