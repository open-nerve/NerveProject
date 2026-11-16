# M6/closeout 完成标记、输入交接关闭与两轮集成实施计划

> 状态：已写（2026-11-16），待预检。分支 `codex/m6-closeout`，基线 main `c4cd1880dc6dfc7d3f063b397de63a80e284ad28`（P8 的 noff，main CI 43851660778 attempt 1 全绿）。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 逐 Task 实施；本阶段只有文档与集成，不做整分支审查（P8 已做，见 re-79）。

**Goal:** 回填 P8 的完成证据、关闭七份输入交接、为 §13 十四条给出最终证据，经两轮 `--no-ff` 与准确的 main CI 把 M6 标为已完成，最后归档清理并向负责人报告。
**Architecture:** 只改文档与关键词的 phase；最终事实写在仓库外的集成收据。
**Tech Stack:** 台账的 `doc-links.py`、`tools/keywords.mjs`、`make lint-web`、GitHub CI 与 P8 的归档工具。
**Spec:** [收尾合同](../specs/closeout.md)。

## Global Constraints

- 不改产品代码、测试、迁移或依赖；发现需要改代码的问题时停下，先提交已完成的部分再按纪律处理。
- 每次提交前从仓库根跑 `python3 .superpowers/sdd/M6-closeout/doc-links.py` 与 `node tools/keywords.mjs`；改 `tools/keywords.json` 后跑 `make lint-web`。检查经 `run-check.py` 留收据。
- 合并一律 `--no-ff`（`git merge --no-ff -F <file>`），推送用 `/usr/bin/git push`；主目录始终在 main。
- CI 规则见 spec §5。
- 运行证据放 worktree 的 `.superpowers/sdd/M6-closeout/`（git 排除），由 T6 归档；第二次 main CI、归档与清理的收据放主目录的 `.superpowers/sdd/M6-closeout-integration-20261116/`，在归档 payload 旁边。

## Task 1（T1）：回填 P8 的完成证据并补完 P8 自身的勾选与状态

- [ ] Step 1：从 P8 的台账（归档 `.superpowers/sdd/M6-P8-integration-20261116/payload/worktree/.superpowers/sdd/P8-final-acceptance/progress.md` 的 Task 15、Task 16 各行）、`ci-evidence/`（三个 waiter 的输出、三次运行的 jobs JSON、合入说明）、归档的 `manifest.json` 与 `cleanup.log` 读数，写 P8 阶段记录的 §5（最终候选的分支 CI 43846387746 attempt 1：各作业的 id、起止与秒数）与 §7（noff `c4cd1880`、双父、main CI 43851660778 attempt 1、归档的文件数、字节、成员、unknown、manifest 的 SHA-256 与 head、清理的时刻与步骤）。
- [ ] Step 2：P8 spec 的 A14 勾选并指向 §7；P8 spec、plan 与阶段记录的状态行改为已集成；M6 设计的头部状态与“当前文档分支”一行、§14 的 P8 行（已完成：noff `c4cd1880`）与 closeout 行（进行中）；`frontend-changes.md` 的 M6 行写 P8 已集成；M8 交接 M6-M8-01 的 P8 一条补最终候选（e2e 1234 秒、web 595 秒、server 825/630 秒）与 main CI（e2e 1258 秒、web 602 秒、server 829/564 秒），换掉“由 M6 收尾的第一个 Task 补”的说法；`tools/keywords.json` 的 phase 设为 `M6/closeout`。
- [ ] Step 3：跑 `make lint-web`、`doc-links.py` 与 `node tools/keywords.mjs`（经 `run-check.py`），提交 `docs(M6/closeout): 回填 P8 的完成证据`。

## Task 2（T2）：关闭七份输入交接

- [ ] Step 1：逐份读原文的条款与各阶段的“处理结果（M6/P<k>）”，核每条都有结论；转交项核对其 M7、M8 `M6-closeout` 的锚点（`M6-M7-0n`、`M6-M8-nn`）存在。
- [ ] Step 2：七份的 `status: open` 改为 `closed`，各加“关闭（M6/closeout）”一段：结论与链接，不改原文与处理结果；M5 的一份写明 M5-M6-01 以“未触发”关闭、条件转 M6-M7-09，M5-04 原文转 M6-M7-08；两份输出交接保持 open。M6 设计 §11 输入表之后“保持 open 至收尾关闭”一句注明已关闭。
- [ ] Step 3：跑 `doc-links.py`；提交 `docs(M6/closeout): 关闭七份输入交接`。

## Task 3（T3）：§13 的证据映射

- [ ] Step 1：新建 `docs/v0/M6-cycles-modules/reviews/closeout-integration.md`：各阶段的合并、main CI 与归档根（P1–P8 与各修复分支）；§13 十四条逐条列证据（阶段记录、整分支审查、P8 的 G1–G20 表与设计 §13 的对照、收据名），此时不勾选；写本阶段预检的结论与集成收据的路径，写明“第二次 main CI、归档与清理由它签结”。
- [ ] Step 2：跑 `doc-links.py`；提交 `docs(M6/closeout): 列出完成标准的最终证据`。

## Task 4（T4）：首次集成

- [ ] Step 1：串行的本地检查 `doc-links.py`、`node tools/keywords.mjs`、`make lint-web`；推送分支，核准确 HEAD 的 server 两路、web、e2e 全绿（中途推送被取消的运行不计）。
- [ ] Step 2：核主目录 main 干净且等于 origin/main、worktree 干净、本地与远端分支都是该候选；`--no-ff` 合入，核双父与 tree 一致，推送。
- [ ] Step 3：核准确的 main CI 全绿，保存 run、attempt 与 jobs 的原件。

## Task 5（T5）：完成的勾选与第二次集成

- [ ] Step 1：§13 十四条逐条勾选，每条附最终证据并写首次 main CI；最后两条写明 P1–P8 已成立的事实与本阶段集成收据的路径；收尾记录补首次集成的事实；总体 §9.4 的 M6 改为“已完成”并链接设计与收尾记录；M6 设计的头部与 §14 的 closeout 行、`frontend-changes.md` 的 M6 行改为已完成；收尾 spec §4 勾选前三项（后两项由收据签结）。
- [ ] Step 2：跑 `doc-links.py` 与关键词守卫；提交 `docs(M6/closeout): 标记 M6 完成`；推送并核准确的分支 CI 全绿。
- [ ] Step 3：第二次 `--no-ff` 合入并推送；核准确的 main CI 全绿，run、attempt 与 jobs 写入集成收据。

## Task 6（T6）：归档、清理与负责人报告

- [ ] Step 1：用 worktree 中的 `archive.py` 归档收尾的全部 ignored 原件，核 unknown 为 0。
- [ ] Step 2：`origin/main..branch` 与 `origin/main..origin/branch` 为 0 之后正常删除远端分支、worktree 与本地分支；从归档副本再核全部成员一致；清理收据写在集成收据目录。
- [ ] Step 3：向负责人提交最终报告：M6 完成（两次 main CI 的 run、attempt、jobs，归档的数字）、各阶段的根因修复、两份输出交接的条目、负责人的开放问题（web 作业的期限与 e2e 的作业结构）与 CI 的余量。
