---
status: open
from: M1/closeout
to: M6
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint、进度的口径

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1350 个（成员 898、prop 452），属于本 M 领域（迭代、模块）的有 **84 个**（成员 46、prop 38）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M6`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M6` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 694 个（web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

## 进度的口径

M1 收尾 T19 让活动迭代卡片的进度经 `@nerve/utils` 的 `calculateCycleProgress` 计算（完成 / (总数 − 取消)，M1 设计 3.4），与迭代列表相同（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 2.9 的 Important 2、3.17）。收尾只改了 Codex 点名的卡片，下面三件事留给本 M（迭代和模块归本 M）：

1. **迭代侧边栏把取消的算在总数里。** `web/apps/web/core/components/cycles/progress-sidebar/sidebar-details.tsx:39,42` 显示完成 / 总数（39 行是已完成迭代的快照，42 行是实时的计数），活动迭代卡片和迭代列表（`calculateCycleProgress`）不算取消的。共 10、完成 8、取消 2 时：卡片写 8/8，列表是 100%，侧边栏写 8/10。
2. **模块的进度是另一个口径，而且写了不止一处。** (完成 + 取消) / 总数 写了两遍：列表项 `web/apps/web/core/components/modules/module-list-item.tsx:45` 和按进度排序的 `web/packages/utils/src/module.ts:38`；模块卡片 `web/apps/web/core/components/modules/module-card-item.tsx:164` 又是完成 / 总数（取消的算在总数里）。两种都与迭代卡片和列表的口径不同。
3. **`calculateCycleProgress` 只要 `progress_snapshot` 不为空就读快照，不看迭代的状态；卡片的文字读的是实时的计数。** 侧边栏进度一节的 `validateCycleSnapshot`（`progress-sidebar/issue-progress.tsx:29`）也只看快照是否为空；侧边栏的数字（`sidebar-details.tsx:36`）却只在迭代已完成时读快照。Nerve 的服务还没有迭代接口，今天没有东西写快照，所以这一条现在不起作用；本 M 定下快照何时写入，读取的条件随之定。

- **怎样处理**：本 M 定进度的口径（取消的算不算），迭代、模块各写一个计算进度的函数，每处显示（卡片、列表、排序、侧边栏）都调用它；快照的读取条件在这个函数里定一次。
- **关闭条件**：每种进度（迭代、模块）由一个函数计算，每处显示都用它；口径（取消的算不算）由本 M 定，写进本 M 的 review。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节；进度一节：2.9 的 Important 2、3.17、8.1。

## 处理结果（M6/P4）：进度的口径

口径定下，写进 M6 设计 §3.5 与 [P4 spec](../specs/P4-progress-transfer.md) §2.1：服务端只给计数，进度 = 已完成 ÷（总数 − 已取消），分母为 0 时为 0，由网页每域一个函数计算（取消的不算在分母里，与活动迭代卡片和迭代列表相同；侧边栏与模块的三种写法在 P5/P6 改为调用它）。快照何时写入也定下：迭代结束后第一次被看到时由周期任务 `planning.freeze_cycles` 或转移写一次（先写者为准），之后不覆盖；未冻结时 `progress_snapshot` 为 `null`（不是 `{}`），只看自己的访客恒为 `null`；读取的条件是迭代的 `status` 为已结束时用冻结，否则用实时（设计 §3.5），由 P5 的那个函数一次定下。第 1–3 条的网页改动在 P5（迭代）、P6（模块），本交接保持 open。证据：提交 `534d9cf092803a870b7d1ce9d64b23a04bc3d51a`（冻结列）、`a92a292387db1105e97a7a52b8e4b9ac6d7d70bb`（进度接口）、`e664868e4d4abea0c3d3e433dc864d533d9043e9`（冻结任务）。

## 处理结果（M6/P5a）：死成员、oxlint 与进度的口径（迭代）

- **死成员和死 prop**：按[M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)附录 A 的三个脚本原样运行（先 `react-router typegen`）。M6 的行：开工时成员 143、prop 39，P5a 删除旧的迭代代码之后 51、38，删去迭代组件里没有调用方传的 prop（`CycleGroupIcon` 的 `color`、`height`、`width`，`BurndownChart` 的 `className`）与侧栏替身中无人读的字段之后 49、35。余下 84 行逐条有结论（P5a 台账“Task 12: 死成员”，随阶段归档）：63 行是测试与夹具的类型（误报）；`applied-filters` 的 `key`、`i18n_title` 与 `cycle.store` 的 `answeredAt` 是误报；18 行属模块，交 P6。
- **oxlint**：P5a 改到的 117 个源文件告警为零；清掉告警的提交同时下调上限：web 58 → 53（迭代归档弹窗的正 `tabIndex`、`single-progress-stats.tsx` 两条、筛选面板的 `autoFocus`、`applied-filters` 的 `prefer-set-has`），utils 4 → 3（随旧 `cycle.ts` 删除）。“按规则清一类”留给 P8。
- **进度的口径**：第 1 条（侧栏把取消的算在总数里）已改：侧栏的“工作项”与状态页签经 `cycleCounts` 与 `progressPercent`，完成 ÷（总数 − 取消）；第 3 条已改：冻结只在 `status` 为已结束且有冻结时读，条件只写在 `cycleCounts` 一处，`validateCycleSnapshot` 删除；列表行的圆环、进行中卡片、侧栏都调同一组函数。第 2 条（模块）在 P6。本交接保持 open。
- 证据：提交 `a21686c4a9827c1fb98d28eafa67c35a5b1150f7`、`a415c86eb74295cd132d449c0ad49b512f93e91f`、`b6a0b7dfe45aeb2fce97abda9b3e6ed14f1e4a10`、`51271de5b98e3d42013086558c310f9b4fbe2881`、`f497055055f00e0cdbfa74e08f5186d40a1297d3`、`aabd9685fdcbcb099e15967069d61ffe6fff7f5d`；[P5a spec](../specs/P5a-web-cycles.md) §2.3、§2.8。
