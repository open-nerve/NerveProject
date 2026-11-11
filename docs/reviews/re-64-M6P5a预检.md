# M6 / P5a 预检

> 2026-11-11：预检通过（修正后），可以按 P5a 的 T1 开始实施。
> - **目标**：[P5a spec](../v0/M6-cycles-modules/specs/P5a-web-cycles.md)、[P5a plan](../v0/M6-cycles-modules/plans/P5a-web-cycles.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `a2b50624fd872c183628a55d04b7961f9a1dbc60`（P4 的 noff `c3ba5a40` 与其 main CI 失败的根因修复 noff，main CI 42824266166 首次通过）；审查对象为分支 `codex/m6-p5a` 的 `3345c975`。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：两个只读调研代理对照 main 的代码写出现状笔记，随阶段台账归档：
   - 网页：迭代领域的全部文件、读者与旧地址，生成的客户端里迭代的操作与 schema，已重写领域的约定（service、`Reconciled`、`oneAtATime`、`useSessionSWR`、`followInSession`、拒绝与复制的 hook、测试假件），迭代页列表要用的列表引擎，守卫、告警上限与 knip，e2e 的夹具与页面故事的写法；
   - Plane 与契约：Plane 迭代页面的确切行为（列表三段、进行中卡片、行与快捷操作、预览、详情、侧栏、图表、归档、表单、删除、转移、开关），nerve 迭代接口的实际实现（字段、可空、封闭的请求体、每个码与次序、权限），交到 P5 的各条交接与差异清单的现状。
2. **写 spec 与 plan**：据此把 P5 拆为 P5a/P5b，写出 P5a 的 spec 与 16 个 Task 的 plan（`3345c975`）。
3. **审查**：一次 fresh 只读审查（`fable`）对照分支上的实际代码逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在某个 Task 内变红而计划不修（尤其新旧 store 并行时的 knip、告警上限的恰等、复制例外的过期、文案键、store 数）；
   - R1–R12 是否站得住、有没有须负责人决定的事；
   - A1–A8 与 e2e 表对 G1–G3、G5–G8、G19 与各交接的覆盖；
   - spec 与接口契约、v0 §7.7 是否一致。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **网页的迭代**：没有一处调 `/api/v0` 的迭代接口；列表页从不取数（包装层不取、没有调用方调 `fetchAllCycles`），显示“没有迭代”；归档目录在冷的 store 上永远加载；详情页只有侧栏，请求都发往 nerve 不提供的旧地址；筛选 store 跟随 router 的反应没有释放；两个模块级的 `CycleService`；归档、恢复吞掉错误，失败也提示成功；侧栏改日期不等回答就提示成功。
- **生成的类型**已经齐备：`Cycle`（没有计数）、`CycleListItem`（六个实时数、`assignee_ids`、`FrozenCounts`）、`CycleProgress`（`status`、`live`、`FrozenProgress`）、只带 id 的分布、数组形状的燃尽。`CycleCreate` 的日期不可为 `null`，所有请求体封闭。
- **守卫**：`lint-cap` 要求告警数恰好等于上限（58，迭代领域与共用文件 5 条）；`root.store.test.ts` 钉住 24 个 store；`copy-hook` 迭代的两条例外到 `M6/closeout`，不再匹配即判为过期；没有规则拦旧的迭代地址；关键词 phase 仍是 `M6/P4`。
- **列表引擎**：没有 `EIssuesStoreType.CYCLE`，迭代路由映射到 `DEFAULT` 并有测试钉住；`ProjectIssues.fetchIssues` 会并入项目偏好的查询；列表的成员判断不认 `cycle_id`。

## 3. 结论与处理

结论 **Ready after fixes**：三处 High、七处 Medium、七处 Low，没有架构问题，没有须负责人决定的事（H3 若改成扩展 M4 的列表引擎会触及跨模块，选择移到 P5b 后仍是本地决定）。全部在实施前改入 spec 与 plan：

| 编号 | 问题 | 处理 |
|---|---|---|
| H1 | `lint-cap` 要求告警数恰等于上限；T7–T9 清掉告警而上限到 T12 才下调，`make lint-web` 在中间变红 | 清掉告警的提交同时下调上限（T7、T8、T9），T12 只核对；写进 Global Constraints 与各 Task |
| H2 | 新的 `orderCycles`、`shouldFilterCycle` 与 `@nerve/utils` 旧的同名函数冲突，旧 store 读它们到 T12 | 新函数另起模块 `utils/src/planning.ts` 与名字（`orderCycleRows`、`matchesCycleFilters`、`cycleCounts`、`progressPercent`），旧的 `cycle.ts`、`progress.test.ts` 在 T12 删除 |
| H3 | 进行中卡片的优先级页签没有 store 的设计：放在迭代 store 是第二份工作项缓存（v0 §7.7 禁止），经 `ProjectIssues.fetchIssues` 会并入项目偏好 | 移到 P5b：P5b 给列表引擎加迭代页 binding 时，以显式的完整查询为它另取一个 binding；P5a 的统计卡只有负责人、标签（Plane 的默认页签）。spec R10、§1、§5 与 T10 改写 |
| M1 | `old-cycle-urls` 照原写会命中保留的代码（组件目录的导入、路由字符串、网页内的链接） | 照 `old-issue-urls` 以 `/api/workspaces/` 前缀锚定，带命中与不命中的样例 |
| M2 | 没有读者的 hook 让 `make knip` 在 T2/T4 变红 | 新文件在第一个读者出现的 Task 里建立：`use-cycles.ts` 随 T2 的包装层，`use-cycle-filters.ts` 随 T9 的列表页 |
| M3 | M4 交接 M6-04(a) 要求真实表单在三种引擎上核对，计划只跑 chromium | 新 `e2e/playwright.m6-pages.config.ts`（三种引擎），T13、T14、T16 在本机跑并留收据；CI 照常跑默认配置 |
| M4 | R4 去掉了反应，而 v0 §7.7 与设计 §8 写的是“由 `dispose()` 释放” | T15 改写 v0 §7.7 那一句与设计 §8，`store-context.test.ts` 的注释随 T4 改 |
| M5 | 直接打开已归档迭代时没有列表行可读成员与计数 | 详情页打开已归档的迭代时另取归档列表，成员从那一行读；它不在面包屑的切换列表里 |
| M6 | 死成员的三个脚本不在仓库里 | T12 把 M1 收尾计划附录 A 的脚本原样写进台账目录，先跑 `react-router typegen` 再跑 |
| M7 | 回执缺 M2 §4 与 M3 §4(c) 的结论；G19 没有“换工作区” | T15 写明两条结论；`g19-cycle-pages` 加换工作区 |
| L1 | `DateRangeDropdown` 已固定 `modal={false}` | 改为在浏览器核对（M2 交接 §5(d)） |
| L2 | `BreadcrumbNavigationSearchDropdown` 没有 `onOpen`（在内部的 `CustomSearchSelect`） | 改为在浏览器核对 M2 交接 §5(c) |
| L3 | 新 store 会超过约 400 行 | 值、读与选择器一个文件，写另一个文件，切法由 T2/T3 定 |
| L4 | e2e 期限 | 最近的 main CI e2e 作业 17 分 45 秒，期限 35 分钟，够用；T9 的检查点记录基线 |
| L5 | 差异清单要点名的行 | 恢复后进入归档目录照 Plane（spec 原写的“进入列表页”改回）；已结束/已归档的菜单里没有删除而服务端允许；已归档迭代的详情可读 |
| L6 | 组件测试的 hook 假件没有迭代的 | Global Constraints 写明扩展 `fake-store-hooks` 或直接 mock |
| L7 | `progress-stats/*`、`single-progress-stats.tsx` 与模块共用 | T8 点名这些文件，模块的调用方保持可编译、行为不变 |

## 4. 审者核实为正确的事实（摘要）

- **计数与名字**：72 个文件、6,138 行；oxlint 58 条告警与所列 5 条；上限 58；24 个 store；76 条规则、12 条例外；`useSessionSWR` 的选项与 `sessionKey` 的形状；`ReconciledByKey`、`oneAtATime`、`followInSession`、`useFormRefusal`、`useRefusalToast`、`useCopyLink(link, copied)`、`ModalTitle` 的接口；`WorkspaceLevelModals` 的清理先例；`ProjectLevelModals` 挂在项目的 key 之上；e2e 的各夹具都在。
- **契约**：草稿不带日期键、修改时日期成对、409 的各码、转移的 422 `target_cycle_id` 与 409 经 `errorMessageKey`、重复归档 200；`FrozenCounts` 与 `FrozenProgress` 都带六个数，`cycleCounts` 可同时服务列表行与 `/progress`。
- **裁定**：R1 拆段、R2 在本段重写的文件里完成设计列给 P7 的几项、R3 新旧 store 并行（旧地址本来不工作，不是兼容层）、R5–R7、R9、R12 站得住且属本地决定；R4 在实质上满足 M2 §3、M3 §1 的关闭条件（文档见 M4）；R8 合 `planning.yaml` 与 `access/domain/rules.go`；R11 是登记的差异。
- **覆盖**：交接中归 P5 的各行由 P5a 或 P5b 覆盖；缺口只有 M3、M7 两处，已补。

## 5. 负责人须知

没有须负责人决定的事。供知悉（本地决定）：P5 拆为 P5a/P5b；设计 §12 列给 P7 的“地址与复制、弹窗宿主与 key、收藏入口删除”中属于迭代文件的部分在 P5a 完成；进行中卡片的优先级页签随迭代页的列表在 P5b 交付。
