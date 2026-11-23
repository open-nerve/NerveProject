---
status: open
from: M1/closeout
to: M7
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1350 个（成员 898、prop 452），属于本 M 领域（通知、收集箱、视图、收藏、最近访问、首页）的有 **88 个**（成员 63、prop 25）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M7`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M7` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

**处理结果（M7/P4，死成员）：** P4 是 M7 的第一个页面阶段：T2 跑基线（1618 行：成员 1224、prop 390、导出 4；`--rows M7` 95 行），T5 删除旧代码之后整份比对，T15 再跑一次（1611 行：成员 1214、prop 393、导出 4；`--rows M7` 98 行）。通知的行（旧 store 的 `updateNotification` 两行与 `mutateNotifications`、旧卡片的两个 prop、`@nerve/types` 的 `workspace-notifications.ts` 的 17 行、旧筛选菜单的 `label`）都随旧代码删除，P4 改到的文件里列出的行删去（`sidebar-item.tsx` 的两个 `className`）；新出现、留下的行逐条写进 [P4 阶段记录](../reviews/P4-web-notifications-review.md) §5（收集箱组件上通知的两个 prop 随 P5，`enableSaveView` 等四个随 P6a/P6b，其余是测试自己的替身与字面量）。M7 其余领域的行随 P5–P7，保持 open。

**处理结果（M7/P5a，死成员）：** T2 跑基线（1614 行：成员 1217、prop 393、导出 4；`--rows M7` 98 行，与 P4 的终表逐行相同），T14 再跑一次（`t14-dead-final`，1598 行：成员 1204、prop 390、导出 4；`--rows M7` 88 行）。P5 的 39 行各有去处：33 行随旧代码删除（`intake-description-version.ts` 15、旧收集箱 store 9 与 `fetchIssueActivity` 1、`@nerve/types` 的 `inbox.ts` 3、`inbox-issue-status.tsx` 的 `showDescription`、`create-root.files.test.tsx:148` 的两个字面量），收集箱组件上通知的两个 prop 在 T6a 不再收（R32，P5b 随读者加回）；留下 6 行是两份新建弹窗测试里 `ResizeObserver` 替身的 `observe`、`unobserve`、`disconnect`（读者是 `create-root.tsx` 经全局构造的 `ResizeObserver`，新建弹窗随 P5b）。新出现、留下的 23 行都是 P5a 自己的测试与夹具里的字面量成员，读者是测试本身；逐条写进 [P5a 阶段记录](../reviews/P5a-web-intake-review.md) §5。M7 其余领域的行（视图 30、收藏与首页 14 等）随 P6–P7，保持 open。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 694 个（web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

**处理结果（M7/P4，oxlint）：** P4 改到与新建的文件 oxlint 警告为零（T15 的只读核对 `t15-oxlint-p4-files`：web 应用 85 个、`@nerve/constants` 1 个、`@nerve/i18n` 1 个、`@nerve/types` 2 个、`@nerve/utils` 4 个、e2e 14 个可检查的文件，都是 0）；通知的旧代码在 web 里的 9 条警告全在 T5 删除的三个文件里（暂缓弹窗 5、暂缓根 1、菜单项 3），web 的上限随同一提交 40 → 31，其余各包的上限不变（`@nerve/utils` 2、`@nerve/constants`、`@nerve/i18n`、`@nerve/types` 与 e2e 0）；按规则清一类：`no-empty-pattern` 升为 error，全仓唯一的一条警告（旧暂缓根）随文件删除，产品代码里被行内 disable 压住的一处（`activity-list.tsx` 的 `comment: {}`）同一提交删掉（[P4 spec](../specs/P4-web-notifications.md) §2.10，[P4 阶段记录](../reviews/P4-web-notifications-review.md) §5）。M7 其余页面阶段照此，保持 open。

**处理结果（M7/P5a，oxlint）：** P5a 改到与新建的 124 个 `.ts`/`.tsx` 文件 oxlint 警告为零（T14 的只读核对 `t14-oxlint-files`：web 应用 106 个、`@nerve/constants` 1 个、`@nerve/types` 3 个、`@nerve/utils` 1 个、e2e 13 个）；收集箱的旧代码在 web 里的 13 条警告全在 T6b 删除与改写的旧文件里（旧的已选筛选 10、`filter-selection.tsx` 1、旧侧栏 2），web 的上限随同一提交 31 → 18，其余各包的上限不变（`@nerve/utils` 2，`@nerve/constants`、`@nerve/i18n`、`@nerve/types` 与 e2e 0）。P5a 没有按规则清一类（M7 的“按规则清一类”P4 已做）。M7 其余阶段改到的文件随各自的阶段，保持 open。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节。
