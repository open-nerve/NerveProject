# M6 / P5a 整体审核

> 2026-11-11：修复后 Ready to merge。
> - **目标**：分支 `codex/m6-p5a` 相对 main `a2b50624fd872c183628a55d04b7961f9a1dbc60` 的全部修改（`3345c975..8f2b1248`，22 个提交）；修复轮 F3 `5b5da6ea` 另经一次限定复审。
> - **依据**：[P5a spec](../v0/M6-cycles-modules/specs/P5a-web-cycles.md)、[P5a plan](../v0/M6-cycles-modules/plans/P5a-web-cycles.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)、[预检 re-64](re-64-M6P5a预检.md)。
> - **阶段记录**：[P5a 阶段记录](../v0/M6-cycles-modules/reviews/P5a-web-cycles-review.md)。

## 1. 方法

一次 fresh 只读审查（`fable`），在最终候选的七项门禁运行期间进行，只读代码，没有运行测试、构建或 lint。读过的内容：

- 差异中全部手写的 TS、TSX、JSON 与文档；
- spec、plan、设计、预检 re-64 与硬规则；
- 实施台账的 16 个 Task 与修复轮 F1、F2（收据、变异、裁定与主控复核）；
- 服务端决定写权限与开关的代码（`access/domain/decide.go`、`rules.go`、`planning/app/lock.go`）；
- 差异清单新行背后的 Plane 源码（逐一对照主目录的只读 `plane/`）。

审者没有逐行读的：各组件测试的断言体（只核用例名与台账的收据）、i18n 两份 json 的逐键差异（由 `check:sync` 门禁覆盖）、e2e 的断言辅助与部分替身、planning 的各用例。

原始报告没有另存文件，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **数据层与写入**：
  - service 只调 `/api/v0`，构造时拿这一代的 `ApiClient`；`RootStore.planning` 以这一代的客户端建立。
  - 四个 `ReconciledByKey`；较旧的取数答到什么都不写，`SessionChangedError` 给出 `undefined`；页面取数全经 `useSessionSWR`。
  - 读：迭代取 nerve 最后回答的副本，且项目须在调用者此刻的项目列表里；列表行叠最新字段；进行中的迭代取第一个 `status === "current"`。
  - 写：同一条 `oneAtATime` 队列；建补 0 计数、置首；改以 `{...row, ...cycle}` 保留行的计数（`Cycle` 的字段是 `CycleListItem` 的子集，R9 成立）；归档、恢复带行移动，重复时就地写，没有行时不造行；删除从两个列表删去、详情写成 `null`；转移在回答后、队列之外重取两份进度与本项目列表；删除与转移在轮到时找迭代；store 不吞错误。
  - 筛选 store 按项目 id 存取，没有反应；`store-context.test.ts` 断言退役的一代不随 router 变。
  - 取数门：包装层要求有效成员、项目未归档、`cycle_view` 打开；详情页按 `archived_at` 取未归档或已归档的列表（R11）。
- **权限**：`canWriteCycles`、`canDeleteCycle` 与 `decide.go:76-103`、`lock.go:140-153` 逐条一致（OwnOnly 只按项目角色且项目未开“访客看全部”；工作区管理员兼项目成员即管理员；建立者按项目角色）；七个写入口都经它们；菜单的状态条件同 Plane `quick-actions-helper.tsx:78-87`。
- **界面**：
  - `ProjectLevelModals` 以 `projectId` 为 key，卸载时关上开关；`CycleActionsHost` 挂在三个布局上，详情页在 `key={cycleId}` 之内。
  - 四个弹窗、侧栏改日期与恢复都经 `followInSession`，只在 nerve 确认后关闭、提示、跳转，被拒时留着并说明；在途时 Escape、点外面、取消都不关。删除只在页面显示的就是该迭代时回列表；归档回列表、恢复进归档目录，同 Plane。
  - 燃尽：`null` 不画，理想线从范围到 0，单点为 0，坐标有限；`cycleCounts` 是冻结条件的唯一一处，`progressPercent` 是唯一的百分比；分布按名字、兜底在后、`null` 行最后。
  - 日期一律按项目时区的日历日；`matchesCycleFilters` 与 Plane `utils/cycle.ts:50-71` 的语义一致。
- **删除与守卫**：spec §2.7 所列的旧代码都已删除，全仓没有旧符号残留；`copy-hook` 迭代的两条例外删除；`old-cycle-urls` 带命中与不命中的样例；`project-address-route` 收入全部改写的文件；告警上限 web 58 → 53、utils 4 → 3 与各提交的下调一致；改到的手写文件都不超过 400 行。
- **测试**：单测与台账的 RED、GREEN、变异对得上；页面故事以请求体、数据库与“一定时间内不提示”钉住行为（草稿没有日期键、修改只发改了的字段且日期成对）；G19 覆盖换账号、后退、换项目、换工作区、换迭代与复制失败；G2 的时区窗口、G6 以数据库轮询等冻结、G3/G8 的耗时裕量都已核对。
- **文档**：`8f2b1248` 的设计 §8/§12、总体设计 §7.7 一句、前端改动清单、差异清单与 spec §5 与代码一致；差异清单引用的 Plane 行号逐一核过。

## 3. 结论与处理

一处 Important，七处 Minor；没有 Critical，没有架构问题，没有须负责人决定的事。

| 编号 | 问题 | 处理 |
|---|---|---|
| I1 | 预览面板的进度侧栏没有以所预览的迭代为 key：预览从 A 换到 B 时侧栏的状态留着，A 上只选了第一天的日期范围显示在 B 上，再选一天就把 A 的半个范围写到 B；A 的改日期在途时 B 的选择器停用、A 的成功提示出现在 B 上；进度的页签停在 A 的 | 修复轮 F3：`cycle-peek-overview.tsx` 给侧栏 `key={peekCycle}`（同详情页的规则）；新测试以真的侧栏核对三种情形，去掉 key 的变异使三例失败 |
| M1 | spec §2.2 写 `details: ReconciledByKey<Cycle>` 与“删：进度一并删去”，实现是 `Cycle \| null`、进度留在键下而读不到（T3 的裁定） | spec §2.2 改到与代码一致 |
| M2 | spec A7 与 plan T15 写 M1-P2、M1-P3 在网页改用新接口后关闭，设计 §11 写输入交接保持 open 至收尾关闭 | 以设计为准（M1-P2 还有模块的进度）；spec A7 注明，回执已写明 M1-P3 的关闭条件已满足 |
| M3 | 项目的迭代关闭（`cycle_view` 为假）时 nerve 对迭代的每个写 409 `planning.cycles_disabled`，而列表页头、详情页、归档目录的写入口仍显示 | 修复轮 F3：`canWriteCycles`、`canDeleteCycle` 共用一处条件（OwnOnly 或迭代关闭或项目不在），命令面板的“新建迭代”去掉重复的检查；spec R8 补写 |
| M4 | 快捷操作的“恢复”没有在途守卫，快速触发两次会发两次请求、两次提示、两次跳转 | 修复轮 F3：在途时再触发不发送，结束（含被拒）后放开 |
| M5 | `old-cycle-urls` 要求 `cycles` 后跟 `/`，`…/cycles"`、`…/cycles?…` 会漏 | 修复轮 F3：以负向前瞻定边界，补 5 个命中、4 个不命中的样例 |
| M6 | 列表页、详情页、归档目录、命令面板的迭代菜单各写一遍“有效成员且项目未归档（且开关打开）”的取数门 | 记入 spec §5 的 P6：模块再加之前抽成一个 hook |
| M7 | 命令面板“打开迭代”在取数门为假时一直转圈，不只已归档的项目，调用者不是项目成员时也是 | 并入 spec §5 已有的 P7 一条 |

**限定复审**（F3 `5b5da6ea`，另一次 fresh 只读审查）：见 §4。

## 4. 修复轮 F3 的复审

另一次 fresh 只读审查（`fable`）只看 `5b5da6ea`，结论：四项修复都在根因上关闭，没有回归，没有 Critical 与 Important。核对过的：

- **I1**：key 在整个侧栏（页头、详情、进度）上，三处留下的状态（选了一半的日期、在途的标记、只在挂载时读一次的页签）一次归零；A 的改日期在途时换到 B，A 的侧栏卸载后不提示、不改状态，回答只进 store；新测试以真的侧栏钉住“把 A 的半个范围写到 B”。
- **M3**：服务端建、改、归档、恢复、转移、删除六个写都经 `Locks.run` 到 `turnedOff`，迭代关闭时 409；两个门的 8 个非测试调用者逐个核对，没有藏起 nerve 允许的东西（读的门、关闭时的空态、导航与归档的页签不变；详情页照 R11 可读）；命令面板去掉重复检查后，没有项目时仍不显示。
- **M4**：`followInSession` 从不拒绝，`unarchive` 不同步抛出，所以守卫在成功、被拒、换会话时都放开。
- **M5**：新正则在段首锚定，`cycles-overview` 之类不命中；`web/` 下没有任何旧的迭代地址；样例由 `tools/keywords.mjs` 在 `make lint-web` 中校验。
- 没有死代码、兼容别名，既有回归断言未改，文件都不超过 400 行。

三条观察，都不改：

| 编号 | 观察 | 处理 |
|---|---|---|
| n1 | 恢复在途时再打开菜单，“恢复”仍显示为可用，点了没有反应（菜单选后即关，通常看不到） | 不改：只发一次是要钉住的性质；与弹窗一致地禁用菜单项留给 P7 的键盘与菜单逐项时一并看 |
| n2 | `cycleWritesRefused` 的说明列了 OwnOnly 与迭代关闭两道，`decideOn` 还有“项目已归档” | 不改：已归档项目的页面整体由项目包装层换成受限页，没有写入口；该函数只判这两道 |
| n3 | A 的改日期在途时 A → B → A，A 的侧栏重挂后选择器又可用，可以再发一次 | 已知行为：同详情页后退、前进的既有规则（总体设计 §7.7），写入一个接一个发出，迟到的回答只进 store |

**Ready to merge：Yes**。
