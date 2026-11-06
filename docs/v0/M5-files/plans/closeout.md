# M5/closeout 完成标记、输入交接关闭与两轮集成 Implementation Plan

> 2026-11-06：预检前草案。 分支`codex/m5-closeout`，基线main`d8ec845750e61b4d1970aec58cb57cb72e3685ab`（P8 noff，main CI 42212370678（attempt 2））。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 逐Task内联实施；本阶段只有文档与集成，不派实现者，不做整分支deep-review（P8已做，见re-52）。

**Goal:** 回填P8完成证据、关闭七份输入交接、为§14十八条给出最终证据，经两轮`--no-ff`与准确main CI把M5标为已完成，最后归档清理。
**Architecture:** 只改文档；最终事实写在仓库外的集成收据。
**Tech Stack:** 既有`doc-links.py`、`tools/keywords.mjs`、`make lint-web`、GitHub CI与P8的归档工具。
**Spec:** [收尾合同](../specs/closeout.md)。

## Global Constraints

- 不改产品代码、测试、迁移或依赖；发现需要改代码的问题时停下，先提交已完成部分再按纪律处理。
- 每次提交前跑`doc-links.py`与`node tools/keywords.mjs`；改`tools/keywords.json`后跑`make lint-web`。
- 合并一律`--no-ff`，推送用`/usr/bin/git push`；主目录始终在main。
- CI失败按根因；环境失败同输入重跑一次并保留原件；重跑仍失败停止并报告负责人。不改预算。
- 运行证据放`.superpowers/sdd/M5-closeout/`（git排除），由T6归档。

## Task 1（T1）：回填P8完成证据

- [ ] Step 1：从P8的合并、main CI、归档与清理收据读数，写P8阶段记录T16段（准确候选、双父、main CI的run/attempt/jobs、归档文件数/字节/成员/manifest SHA-256、清理收据）。
- [ ] Step 2：M5设计头部状态与§15 P8行改为已完成并链接T16段；`tools/keywords.json`的phase设为`M5/closeout`；跑`make lint-web`与`doc-links.py`。
- [ ] Step 3：提交`docs(M5/closeout): 回填P8完成证据`。

## Task 2（T2）：关闭七份输入交接

- [ ] Step 1：逐份读原文条款与“处理结果（M5/P8）”，核每条都有结论；转交项核对其M6/M7/M8`M5-closeout`锚点存在。
- [ ] Step 2：七份`status: open`改为`closed`，各加“关闭（M5/closeout）”段：结论与链接，不改原文与处理结果；三份输出交接保持open。
- [ ] Step 3：M5设计§13的“当前所有输入仍为open”一句改写为关闭事实；跑`doc-links.py`；提交`docs(M5/closeout): 关闭七份输入交接`。

## Task 3（T3）：§14证据映射

- [ ] Step 1：新建`docs/v0/M5-files/reviews/closeout-integration.md`：§14十八条逐条列证据（阶段记录节、整审、P8矩阵行与收据名），此时不勾选；写明最终收据的位置。
- [ ] Step 2：跑`doc-links.py`；提交`docs(M5/closeout): 列出完成标准的最终证据`。

## Task 4（T4）：首次集成

- [ ] Step 1：串行本地检查`doc-links.py`、`node tools/keywords.mjs`、`make lint-web`；推送分支，核准确HEAD的server/web/e2e全绿。
- [ ] Step 2：核主目录main干净且等于origin/main、worktree干净、本地与远端分支都是该候选；`--no-ff`合入，核双父与tree一致，推送。
- [ ] Step 3：核准确main CI三项全绿，保存run/attempt/jobs原件。

## Task 5（T5）：完成勾选与第二次集成

- [ ] Step 1：§14十八条逐条勾选，每条附最终证据并写首次main CI；收尾记录补首次集成事实；总体§9.4的M5改为“已完成”并链接设计与收尾记录；收尾spec§4勾选到首次集成为止。
- [ ] Step 2：跑`doc-links.py`与关键词守卫；提交`docs(M5/closeout): 标记M5完成`；推送并核准确分支CI全绿。
- [ ] Step 3：第二次`--no-ff`合入并推送；核准确main CI三项全绿，结果写入仓库外的集成收据。

## Task 6（T6）：归档与清理

- [ ] Step 1：用P8归档工具（先按P8 manifest核其SHA-256）归档收尾全部ignored原件与会话scratchpad，核unknown为0。
- [ ] Step 2：`origin/main..branch`与`origin/main..origin/branch`为0后正常删除远端分支、worktree与本地分支；从归档副本再核全部成员一致；写清理收据。
