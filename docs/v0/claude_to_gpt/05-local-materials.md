# 05 本机的资料和工具

这些都在本机，不在仓库里。工具不进仓库（负责人的规则）。

## 1. 持久目录（重启不丢）
`/Users/xiaoruan/.claude/projects/-Users-xiaoruan-project-nerve-project/carry-tools/`

**根目录**：
- `wait-page.sh`、`ci-labels.sh`、`ci-run-jobs.sh`、`ci-job.sh`、`ci-annotations.sh`：看持续集成，用法见 [03](03-rules-and-pitfalls.md) 第 7 节。
- `mut.py`、`e2ekills.py`：M3 时的变异工具，P3 的版本在下面的 `tools/`。
- `m4-kickoff-prompt.md`：M4 开工时的交接提示词，有节奏、硬规则和完成条件。

**`m4-handoff/briefs/`**：P3 用过的简报原文，英文，可以当模板改写：
- `p3-architect-brief.md`：架构（写 spec 和 plan、原型、重放）的简报，有输入、最高风险、此前各段的教训、大小的检查、原型和输出的要求。
- `p3-architect-rulings.md`、`p3-preflight-rulings.md`：裁定的写法样例。
- `p3-preflight-brief.md`：预检评审的简报。
- `implementer-standing.md`、`task-review-standing.md`：每个 Task 的实现者、评审者的长期简报，有自查清单和评审的标准。
- `p3-final-review-brief.md`：整分支终审的简报，有跨 Task 要看的七类和分诊。
- `final-fix-brief.md`：终审之后的修复轮。
- `review-doc-brief.md`：写评审记录的简报。
- `p3-merge-msg.txt`：合并提交说明的样例。

**`m4-handoff/records/`**：P2、P3 的过程记录：
- `p2-sdd-ledger.md`、`p3-sdd-ledger.md`：逐 Task 的台账，有每个裁定和修复轮；
- `p3-controller-notes.md`：控制者的笔记，也有第 340 次运行的排查；
- `p3-preflight.md`：预检评审，有死锁推导、逐操作的访客规则；
- `p3-final-review.md`、`p3-final-rereview.md`：终审和复审。

**`m4-handoff/tools/`**：P3 的工具。里面写死了 Claude 临时目录的绝对路径，用之前先读文件头、改路径：

| 工具 | 用途 |
|---|---|
| `planapply.mjs` | 把 plan 里一个 Task 的代码块应用到一棵树：`apply <plan> <tree> <N>`，先检查每个块，有一个不匹配就什么都不写；`check` 只检查 |
| `taskblocks.py`、`compose_plan.py` | 修复轮之后，从前后两棵树重新生成这个 Task 的代码块并合回 plan（"照执行的样子"） |
| `treediff.mjs`、`snapcmp.py` | 比较两棵树（工作树和某个 Task 的快照） |
| `mut.py`、`mutcheck.py` | 跑变异：每个变异在一份新副本里改，在每一层跑测试。`MUT_ONLY`、`MUT_LATER`，结果和日志的路径由环境变量给出；pgtest 的容器起不来记为 `INFRA`，不算发现 |
| `e2ekills.py` | 核对端到端一层的"发现"是断言失败，不是崩溃或超时 |
| `race.sh`、`raceall.sh`、`racenames.py` | 用 `-race -count=5` 跑交错测试 |
| `doclinks.py` | 检查文档里的相对链接 |
| `spectable.py`、`filetablecheck.py`、`countfiles.py` | 核对 spec 的文件表、数目与树相符 |

P3 的变异定义（`mutants_p3*.py`）只对 P3 的树有效，没有复制。

**`m4-handoff/w11-probes/`**：排查 W11 偶发失败时用的脚本和探针，说明见 [06](06-ci-w11-flake.md) 第 7 节。

## 2. 会话的临时目录（重启就没了）
`/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/271ea4af-9c2b-4e77-aad3-44f0c397be7d/scratchpad/nerve-m4/`

系统还会定期清掉 `/private/tmp` 里放了几天的文件：P1a、P1b 的台账存档已经空了。还在的有：
- 各段的原型 `p*proto/`、快照 `p*snap/`、重放 `p*replay/`、工具 `p*tools/`，其中部分已被清掉；
- 台账的存档 `p2-sdd-archive/`、`p3-sdd-archive/`；
- 架构、预检、终审的报告和裁定 `p*-architect-*.md`、`p*-preflight*`、`p*final/`；
- 第 340 次运行的排查 `p3ci/`。

值得留的已复制到上面的持久目录。这里的其余文件是过程的副产品，仓库里的 spec、plan、评审记录已经记下了结论。
