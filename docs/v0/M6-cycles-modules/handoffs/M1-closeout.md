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

## 处理结果（M6/P6a）：死成员、oxlint 与进度的口径（模块）

- **死成员和死 prop**：照[M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)附录 A 的三个脚本原样运行（写进 P6a 台账的 `dead-members/`，先 `react-router typegen`）。M6 的行：P6a 删除旧模块代码之前（`e2d967a1`）成员 184、prop 92，之后成员 122、prop 66，共 188 行（迭代 105、模块 83）。P5a 交来的模块 18 行全部了结：旧 `ModulesStore`、`ModuleFilterStore` 的 7 行随旧 store 删除；`@nerve/types` 的 `module/modules.ts` 5 行（`pending_issues`、`avatar_url`、`first_name`、`last_name` 随 T8 删去的分布类型，`filters` 随目录）删除；`archived-modules/modal.tsx` 的 `onSubmit` 随改写不再存在；`module-layout-icon.tsx` 的 `className`、`containerClassName`、`size`、`withContainer` 与 `power-k/menus/modules.tsx` 的 `value` 删去。两个测试替身各自重复声明的 `MembersChange`（4 行）删去。模块余下的 83 行逐条有结论（P6a 台账“Task 12: 死成员”，随阶段归档）：产品代码的 4 行是误报（`applied-filters/root.tsx` 的 `key`、`i18n_title`、`kind` 是 `satisfies` 约束的成员、经筛选表读出；`module.store.ts` 的 `answeredAt` 以字符串键读）；测试替身的 13 行与测试的 66 行是替身收到的 props 与测试表的类型（脚本看不到 `vi.mock` 替身收到的 props，误报）。
- **oxlint**：P6a 改到的 167 个源文件（web 156、constants 4、types 3、utils 3、propel 1）告警为零；模块的 9 条随改写清掉，清掉的提交同时下调上限：web 53 → 52（`e9fc041b`，归档按钮的正 `tabIndex`）→ 47（`82dac1ef`，卡片的 5 条）→ 44（`e2d967a1`，`applied-filters` 的 2 条 `prefer-set-has` 与筛选面板的 `autoFocus`）；utils 3、propel 6、constants、types、i18n 0 不变。“按规则清一类”留给 P8。
- **进度的口径**：第 2 条（模块）已改：列表行的圆环、按进度排序、卡片的数字与进度条、侧栏的“工作项”与状态页签都经 `progressPercent`，完成 ÷（总数 − 取消），与迭代同一口径；（完成 + 取消）÷ 总数（`module-list-item.tsx`、`@nerve/utils` 的 `module.ts`）与完成 ÷ 总数（`module-card-item.tsx`、侧栏）随改写与删除不再存在。变异 `t4-mut-progress-with-cancelled`、`t8-mut-work-items-total`、`t10-mut-ring-plane-formula`、`t10-mut-card-count-total-2`、`t10-mut-bar-plane-formula-2` 各让测试失败；G12 页面故事核对列表行、卡片、侧栏同为 40%（5 个未取消的完成 2 个，已取消的不在分母里）。第 1、3 条在 P5a 已改，三条至此都已改。本交接保持 open。
- 证据：提交 `6a088615629ba2c7d127e53cea25cc0376c1e2b5`（排序与筛选的函数）、`e9fc041b4a1cae06173fa29ec9b03f9ad0c109ca`、`c09d4f8d9d2544724887ad1ad34a5c6cf0b44ffc`、`82dac1efd9dac3fe7b7d99bdcac18410cd12bb0a`、`e2d967a18fc3ba0f02210e51f9e407d4f1893c5f`、`dc9172b0fecf86c716bf50394b03c2a4e4a6fd6d`（旧代码删除与死成员）、`b8ed29df59bcb59fc30adfbafb8c79748af992b9`（G12 页面故事）；[P6a spec](../specs/P6a-web-modules.md) §2.4、§2.10。

## 处理结果（M6/P7c）：oxlint（本段改过的文件）

- **oxlint，谁改谁清**：本段改过与新建的、受告警上限管的源文件共 83 个（`git diff --name-status -M d052213f..3b5b0a78` 中的 `.ts`、`.tsx`：web 81、`@nerve/constants` 1、e2e 1；没有删去的文件，`power-k/config/navigation/commands.ts` 拆为两件，算作新建），逐个 0 告警、0 错误。计数照 `tools/lint-cap.mjs` 的算法：在各文件所在包的目录跑 `oxlint --format=json .`，再按文件分开（本阶段台账的 `oxlint-files.py`，收据 `t14-oxlint-touched`，清单 `t14-oxlint-files.txt`：T2 开工前基线清单上的 53 个、本段新建的 23 个、不在基线清单上而改过的 7 个）。T2 开工前的基线（收据 `t2-oxlint-baseline`：plan 点名要改的 58 个文件）同样全是 0，所以本段没有清掉的告警、上限不变：web 44（= 上限 44）、`@nerve/constants` 0、e2e 0；新文件没有加告警（中途出现过的几条——测试里的 `jsx-key`、`consistent-function-scoping`、`no-shadow`、`jsx-no-constructed-context-values`——都在各自的 Task 里提交之前改掉，台账的 `t<N>-oxlint-after`）。基线清单上有 5 个本段最终没改：`navigation/commands.ts`（拆去）、`work-item/commands.ts`、两个 `quick-actions.test.tsx`（回归的边界）与视图详情页头（只加入守卫）。
- **死成员和死 prop**、“按规则清一类”：本段不跑，收敛在 P8（[M6 设计](../M6-design.md) §12 的 P8 一行；上文 P5a、P6a 的回执）。
- 证据：收据 `t14-oxlint-touched`（HEAD `3b5b0a78d5c9da7202645f80427bcf5c7f2bb136`）、`t2-oxlint-baseline` 与各 Task 的 `t<N>-oxlint-after`，随本阶段的台账归档；[P7c spec](../specs/P7c-palette-mobile.md) §2.11、R16。本交接保持 open。

## 处理结果（M6/P8）：死成员、按规则清一类、M6 全期改动的文件

- **死成员和死 prop**（T9，[P8 spec](../specs/P8-final-acceptance.md) §2.9）：照附录 A 的三个脚本原样（SHA-256 前缀 `deadsym.mjs` `3d85f090`、`lib/program.mjs` `b2c52513`、`domains.mjs` `5f76ea3b`，先 `react-router typegen`）在 `0f2d331f` 上跑 `--rows M6`：245 行（member 172、prop 73；测试 206、替身 15、产品 24），与 `ea971606` 上同数。没有真死成员：调研以为死的 `fake-module-dialog.tsx:32-33` 的 `invalid`、`describedBy` 有写者（`modules/form.tsx` 经真的 `MemberDropdown` 的 `{...props}`）与读者（替身的 `{...fieldOf(props)}`，脚本认不出经参数类型的解构），删声明则类型检查不过、删读者则 `modal.test.tsx` 失败（探针 `t9-probe-drop-decls`、`t9-probe-drop-fieldof`），故不删、T9 不提交（spec 照实改写，`f46d29d9`）；复扫 `t9-after` 的 tsv 与 `t9-before` 逐字节相同。产品文件的 24 行（`satisfies` 的筛选表、经 `in` 收窄、经接口读的服务方法与计算键写的 `filters`、按字符串键读的 `answeredAt`、展开传递的 `additionalOptions`、经 `SPREADSHEET_COLUMNS` 传递的表格两列）、替身的 15 行逐行写明文件、名字与读者，测试的 206 行按文件计数，读者表在台账 `t9-readers.md`，由 T12 抄入 [P8 阶段记录](../reviews/P8-final-acceptance-review.md)。M1 交来的 84 行（在 `559c3c6` 上重跑得同数，member 46、prop 38）：78 行已消失（57 行的文件已删、19 行的成员已删、2 行已不是 prop），余下 6 行是 `cycle-column.tsx`、`module-column.tsx` 的 `issue`、`onClose`、`disabled`，经 `spreadsheet-columns.tsx` 传给 `issue-column.tsx` 的误报。
- **按规则清一类**（T10，§2.10，`8342965d`）：`eslint(no-unused-expressions)` 全仓 4 条清零——web 3 条（通知卡片的 `options/button.tsx`、`use-intersection-observer.ts`、`use-reload-confirmation.tsx`）、`@nerve/utils` 1 条（没有读者的 `router.ts` 连同 `generateQueryParams` 删去）——并在 `.oxlintrc.json` 设为 error（全仓，含 e2e；恢复一条的变异 `t10-mut-rule`、`t10-mut-rule-e2e` 以 “errors are never allowed” 失败）；`use-reload-confirmation.tsx` 删去三个没有调用方传的参数，同文件的一条 `react-hooks(exhaustive-deps)` 随之消失。各包上限：web 44 → 40、`@nerve/utils` 3 → 2，其余不变（`@nerve/ui` 4、`@nerve/hooks` 3、`@nerve/propel` 6，其余 0）；全仓告警 60 → 55（收据 `t10-baseline-2`、`t10-after`）。
- **谁改谁清（M6 全期）**：`git diff --diff-filter=AMR --name-only ba3fd6e0..HEAD`（`ba3fd6e0` 是 M5 收尾的 noff）的 1327 个文件中有 698 个 JS/TS，`.oxlintrc.json` 的 ignorePatterns 跳过 2 个（`e2e/playwright.m6-pages.config.ts`、`api-client` 的 `schema.gen.ts`），可 lint 的 696 个（web 584、e2e 77、`@nerve/constants` 10、`@nerve/types` 8、`@nerve/utils` 8、`@nerve/editor` 4、`@nerve/propel` 3、`@nerve/api-client` 2）全部 0 告警、0 错误，没有落在受上限管的包之外的（收据 `t10-oxlint-m6`，HEAD `8342965d`；向一个 M6 的文件加一条告警的变异 `t10-mut-m6-receipt-2` 让收据失败）。P5b、P6b、P7a、P7b 没有单独的收据，由这份全期的收据覆盖；之后 T11 改过与新建的 12 个文件同样 0 告警（`t11-oxlint-files`）。
- **进度的口径**：三条在 P4、P5a、P6a 已改（上文的回执），P8 没有改动。
- 关闭条件：死成员一节由阶段记录的读者表满足（每一行已消失或写明读者）；oxlint 一节写明了改到的文件的告警数（0）、清掉的规则（`no-unused-expressions`）与各包上限的变化；进度一节在 P6a 已满足。证据：提交 `8342965d2de648d80ab98a3cd5b6c522ceeadf02`、`f46d29d950295aa594c1327bbf290f54ea954a00`；收据 `t9-before`、`t9-after`、`t9-probe-drop-fieldof`、`t9-probe-drop-decls`、`t10-baseline-2`、`t10-after`、`t10-mut-rule`、`t10-mut-rule-e2e`、`t10-oxlint-m6`、`t10-mut-m6-receipt-2`、`t11-oxlint-files`；本阶段台账 Task 9、10。本交接由收尾改为 `closed`。
