---
status: closed
from: M1/closeout
to: M3
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1350 个（成员 898、prop 452），属于本 M 领域（工作区、项目、成员、状态、标签、权限、邀请）的有 **210 个**（成员 142、prop 68）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M3`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M3` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 694 个（web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节。

## 处理结果（M3/P11）

- **死成员和死 prop**：P11 删掉它改到的文件和只经机械步骤到达的文件中真正没有读者、没有调用方传入的行（[P11 spec](../specs/P11-web-states-labels-and-cleanup.md) 第 2.8 节）：`--rows M3` 中产品代码的行 80 → 54，删去 26 行，没有新出现的。`--rows M3` 剩下的每一行写进 P11 spec 的附录 A.9：文件、名称、谁在读或传它（测试里 `it.each` 的列和 `vi.hoisted` 的桩、经展开、组件类型的变量或第三方库读取的成员、M4 的搜索类型），以及按路径归到 M3、实际属于 M4、M6、M7 的 41 行（M3 设计第 1 节；收尾抄进它们的交接）。
- **oxlint**：`promise(always-return)` 全仓清零（35 条：web 34、editor 1；每一处的拒绝照旧：23 处有处理，6 处交回调用方，6 处没有处理：`entry.client.tsx` 的 `then` 之前的拒绝已被吞下，五处复制链接列给 M4、M6、M7，P11 spec 附录 A.6、第 5 节）；P8a、P8b 留给 P11 的只经机械步骤到达的文件清零；各包的上限随之调到新的条数（web 348 → 303、editor 65 → 64，其余的包条数没有变）。P11 改到的文件中留着的警告：`@nerve/ui` 的 `alert-modal.tsx` 的一条（`tabIndex={1}`，M3 收尾的 `ModalCore` 重写），以及只按 `always-return` 改到的文件中 5 个的 14 条（M4、M6、M7、M8 的组件；P11 spec 第 3 节第 3、8 条）。

状态保持 `open`：P11 的评审写明剩下的行和上限之后，M3 的收尾关闭。

## 处理结果（M3/收尾）

两节的结论：

1. **死成员和死 prop**：M3/P11 删掉了改到的文件中真正没有读者的行，`--rows M3` 中产品代码的行 80 → 54；剩下的每一行写进 [M3/P11 spec](../specs/P11-web-states-labels-and-cleanup.md) 附录 A.9（文件、名字、谁在读或传它）。收尾在它的最后一个代码 Task 之后重跑（`deadsym.mjs`、`domains.mjs --rows M3`）：218 → 221 行，产品代码的 54 行和按路径误归的 41 行不变；测试中的行 123 → 126，多出的是收尾新的测试中 `it.each` 的四列（`project-dialog-texts.test.tsx`），少的一行是 `delete.test.tsx` 的 tooltip 替身的 prop（收尾把两个测试的替身合成 `fake-controls.ts` 的一个；收尾 spec 附录 A.5）。误归的 41 行写进各自的交接：M4 16 行（[M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 8 节），M6 8 行（[M6 的交接](../../M6-cycles-modules/handoffs/M3-closeout.md)第 2 节），M7 17 行（[M7 的交接](../../M7-collaboration/handoffs/M3-closeout.md)第 2 节）。
2. **oxlint**：M3 改到的文件在各 Phase 结束时没有警告，`promise(always-return)` 由 M3/P11 全仓清零，各包的上限照新的条数（M3/P11：web 348 → 303、editor 65 → 64）。收尾再调低上限：web 303 → 293、`@nerve/ui` 15 → 12、`@nerve/propel` 11 → 10。收尾改到的文件中仍有警告的 6 个照控制者的裁定写进交接：只为 `IconButton` 的名字或 `showTooltip` 改到的 `core/description-versions/modal.tsx`、`issues/issue-detail/issue-activity/activity-filter.tsx`（[M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 8 节）、`modules/module-card-item.tsx`（[M6 的交接](../../M6-cycles-modules/handoffs/M3-closeout.md)第 4 节）、通知的筛选菜单和只为 `CustomMenu` 改到的收藏的文件夹（[M7 的交接](../../M7-collaboration/handoffs/M3-closeout.md)第 4 节），和只改了占位文字的页头搜索 `navigation/top-nav-power-k.tsx`（[M8 的交接](../../M8-open-release/handoffs/M3-closeout.md)第 4 节）；收尾 spec 第 3 节第 2 条和附录 A.5 逐个写明。M4–M8 的五份 M1 收尾交接写明 `always-return` 已清零。

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。
