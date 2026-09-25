---
status: open
from: M1/closeout
to: M7
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1347 个（成员 895、prop 452），属于本 M 领域（通知、收集箱、视图、收藏、最近访问、首页）的有 **88 个**（成员 63、prop 25）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M7`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M7` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 695 个（web 566、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节。
