# M6 / P5b 整体审核

> 2026-11-12：Ready to merge。
> - **目标**：分支 `codex/m6-p5b` 相对 main `0b8efcd701edf7e73a37d5bb82ad176a7795b910` 的全部修改（`a23270ea..8b11bc63`，21 个提交）；修复轮 F（`6512df33..ebeb2cab`，8 个提交）另经一次限定复审（§4）。
> - **依据**：[P5b spec](../v0/M6-cycles-modules/specs/P5b-cycle-list.md)、[P5b plan](../v0/M6-cycles-modules/plans/P5b-cycle-list.md)、[预检 re-66](re-66-M6P5b预检.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。
> - **阶段记录**：[P5b 阶段记录](../v0/M6-cycles-modules/reviews/P5b-cycle-list-review.md)。

## 1. 方法

一次 fresh 只读审查（`fable`），在最终候选的七项门禁运行期间进行，只读代码与文档，没有运行测试、构建或 lint，也没有改动 git 状态。审者读过：

- 全部产品代码的差异，M4 共用文件（`base-issues.store.ts`、`root.store.ts`、偏好的筛选基类、`usePageIssueFilters`、四个 base root、`command-host.tsx`、`project-issue.tsx`）的完整差异，以及每个被改动的既有测试的差异；
- spec、plan、预检 re-66、设计、总体设计 §7.2/§7.7、硬规则与实施台账（16 个 Task 的收据、变异、裁定与主控复核）；
- nerve 的 `access/domain/decide.go`；差异清单新行背后的 Plane 源码（逐一对照主目录的只读 `plane/`）。

测试的诚实性与文档的准确性另由三个 fresh 只读审者分头核对，审者亲自复核了其中分量最重的两条（M2、M4）。没有逐行读的：e2e 辅助函数中未被引用的部分、`fake-cycle-page.tsx`/`fake-cycle-routes.tsx` 的内部、`base-issues.store.test.ts`、差异之外的 i18n 与 `tools/keywords.json`、`IssueListBinding` 与 shared-state 的筛选实例中未涉及的部分、`server/`（未改）。

原始报告没有另存文件，逐条内容与处理见 §2、§3；Minor 的原文随台账归档（`research/re-67-findings.md`）。

## 2. 核对过的性质

- **数据层与写入**：
  - 成员判断：迭代的绑定只留 `cycle_id` 在查询里的工作项（null 对 “none”），模块按 `module_ids` 的交集；只在项目分支。`fields()` 带 `cycle_id`、`module_ids`，别处的读让成员可能已变；创建与更新的白名单带 `cycle_id`。
  - 偏好的基类 `PagePreferencesFilter`：读按次序、以最新的读为准，读开始时记下各字段的版本，读开始后才确认的字段保留；`replace` 不需要先读、只发一个该字段的 PATCH、以回答种下并升版本；`merge` 没有读过就拒绝、不发请求；每页每字段一条 `oneAtATime`；每次请求前后 `assertActive`。项目页的 `ProjectIssuesFilter` 是 34 行的子类，项目页的测试未改。
  - `CycleIssues`：`scope: "project"` 加 `cycle_id: [id]`（R1，没有新的 scope）；只认本页的列表；`createIssue` 只在目标项目是迭代的项目时带 `cycle_id`，钩子只在 `joinable`（未结束、未归档、迭代打开）时传本页，否则照项目页新建、不重读进度；恰好一个 POST。
  - 优先级页签：显式完整查询、不含偏好；每代每个 [工作区、项目、迭代] 一个绑定，首次取数才建并登记进根；不碰 `cycleIssues.currentList`；游标续页；只在页签打开且卡片的门成立时取数。
  - 移出迭代：`PATCH {cycle_id: null}`，反馈由 `IssueListCommandHost.runRowCommand` 持有；写后经 `useCycleProgressFollow` 重读 `/progress` 一次（`followInSession` 加页面范围）。
- **会话与页面的范围（G19）**：挂载键带 `cycleId`；偏好键 `["CYCLE_ISSUE_FILTERS", slug, cycleId]` 只在成员且迭代打开时非空；优先级的键带权限身份。`select()` 在发出时，迟到的列表回答只写自己的绑定；详情页以 `key={cycleId}` 重挂。没有退役后仍写的路径；`useEnterFiltered` 只在 `current()` 成立时跳转、一次一个；`PeekOnEntry` 在宿主就位后打开并以 `replace` 去掉导航状态。新代码不读 router store，`project-address-route` 为每个新文件加了样例。
- **权限与只读**：新建与移出给 `joinable && allowPermissions([ADMIN, MEMBER])`（工作区管理员在网页上是项目 ADMIN，e2e 的 wanda 钉住），纯访客都没有；OwnOnly 由 nerve 按项目角色给出。四个 base root 的只读 prop 进 `disableIssueCreation`、`canEditProperties` 与日历；已结束、已归档时偏好仍可写（spec §2.6）；迭代关闭时列表区是停用空态、没有取数。行点击：卡片替换整个 `filters` 字段、确认后跳转、被拒说明；侧栏经本页实例切换、`aria-pressed` 说明已选、没有实例时不可点。
- **M4 共用代码与回归边界**：`BaseIssuesStore<TFilter>` 只放宽约束；`PagePreferencesIssues` 的请求与项目页逐字节相同；`usePageIssueFilters` 对 PROJECT、PROFILE、ARCHIVED 返回原来的 getter；`readOnly` 默认 false。命令宿主的弹窗在项目页与个人页以 PROJECT 打开，与原来行内的弹窗相同；删去的行内 `onSubmit` 是死代码。被改动的既有测试都在 plan 的回归边界之内。
- **测试**：单测、组件测试与 e2e 的各项性质都至少有一个在台账变异下失败的测试；e2e 的请求体与查询断言不为空、扣住的回答都放行并等待、假时钟只在 G19 故事二且用法正确。
- **文档**：差异清单第 1、2、4–8 行与 Plane 的引用正确（第 3 行见 M2）；前端改动清单的文件与组件都存在；M3/M4/M5/M8 交接回执引用的 15 个 SHA 都在范围内；i18n 的 en 与 zh-CN 成对加入；`cycle_issues` 常量里没有按迭代、模块分组（R9）。

## 3. 结论与处理

结论 **Ready to merge**：没有 Critical、Important，十一处 Minor，没有架构问题，没有须负责人决定的事。修复轮 F 处理 M1–M9，M10 记为观察，M11 不需动作：

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | spec §2.6 三句与代码不符：页头数目是 `pageTotal` 不是 `getGroupIssueCount`；修改与复制弹窗已由命令宿主持有，“复制的一份加入本迭代”缺内容复制的例外；优先级页签失败时说明并可重试 | F7 改 spec §2.6、§2.7（`d4c93732`） |
| M2 | 差异清单“已结束、已归档迭代的列表”一行说 Plane 只看浏览器时钟；Plane 的列表根是“时钟或 `status`”，页头的新建只看 `status` | F7 对照 Plane 源码改写（`d4c93732`） |
| M3 | spec §2.10、A5 说 G15 覆盖偏好被拒与空页，故事没有做 | F7 改为故事实际覆盖的内容，偏好被拒由单测钉住（`d4c93732`） |
| M4 | G15 故事一的四种布局没有布局本身的锚点，看板错映到列表也能过 | F5 每步先等该布局的结构（列表组、看板列、日历日格、表格）；变异 `f5-mut-kanban-as-list` 让故事失败（`a94e4d8b`） |
| M5 | G19 故事一的后退前进一步只以 1 秒内的否定钉住迟到的回答 | F5 放行后加 Next 只列 “Ship”、数目 1 的正向锚点（`a94e4d8b`） |
| M6 | 单测强度：一处不会失败的断言、两处接受任何拒绝、一处恒真、几处只有否定、一处零行也成立、一处没推进假时间、两处时序没在成功例里断言；三处空缺（优先级的行被移出迭代、`pageTotal`、新建在已归档与迭代关闭时的分支） | F5 逐条修正并补三处测试，各有能让它失败的变异（`a94e4d8b`） |
| M7 | `project-issue.tsx` 以 `useRef` 适配 helper 的两步接口，依赖调用处看不见的同步次序 | F4 菜单工厂的参数改为联合：列表行传 `openDialog(kind)`，详情与归档保留两个 setter；删去 ref（`1f923b23`） |
| M8 | 迭代页的偏好未读到或读失败时，页头的布局、筛选与显示仍可用，点了只得一个笼统的错误提示 | F3 页头在 `issuesFilter.getIssueFilters(cycleId)` 存在后才显示这些控件，与正文同一条件；RED/GREEN 与变异（`8b9fa1c7`，P5a 页测的替身随之补读取 `35215d46`）。项目页同因，M4 起如此，记入 spec §5 的 P6 |
| M9 | `use-enter-filtered.ts` 一行注释约 190 字符 | F6 折行（`1f923b23`） |
| M10 | `root.store.ts` 615 行（M4 的文件，基线 593 行，P5b 加 22 行） | 观察：spec §5 与 [M6-M8-08](../v0/M8-open-release/handoffs/M6-closeout.md#M6-M8-08) |
| M11 | plan 第 199 行已按裁定改正 | 不需动作 |

## 4. 修复轮 F 的复审

修复轮 F（`6512df33..ebeb2cab`，8 个提交）经一次 fresh 只读限定复审（`fable`，在最终候选的门禁运行期间，没有运行或改动任何东西），逐项核对：

- **F1**：`dismissToasts` 一次按下所有未在退场（`[data-ending-style]`）的通知的关闭，再以 `toHaveCount(0)` 等全部消失，没有等待、没有放宽期限（e2e 没有设 `expect.timeout`，默认 5 秒；通知 5000 ms 自动关闭、500 ms 退场）；之前的通知关不掉就仍被数到，不会空过；焦点与指针不动，调用处未改。
- **F2**：`loadedAnew` 以新的浏览器上下文与新的登录载入，“再次载入后仍在”是真的：先 `expect.poll` 证明 nerve 存着这些值，共享的缓存只会让新页面显示默认值而失败；故事一、二原有的断言都保留，故事二另核 nerve 存下了表头排序的 `order_by`。
- **F3**：`getIssueFilters` 未读时为 `undefined`，与正文的 `!workItemFilters` 等价；数目、新建、侧栏开关与快捷操作不变，迭代关闭与非成员的路径不变；P5a 页测只补替身。
- **F4、F6**：菜单工厂的参数是带 `?: never` 的联合，列表行（项目、迭代、个人页）传 `openDialog`，详情与归档传两个 setter；复制仍带 `sourceIssueId`，存储类型迭代页为 CYCLE、其他为 PROJECT；`editing`、`useRef` 已删，没有遗留的读者。
- **F5**：新增的断言都是正向、能失败的，台账所记的变异正对着它们。
- **F7、F8**：spec 的改写、差异清单一行（对照 Plane 的 `cycle.store.ts:177-191` 与页头 `header.tsx:107,239`）、spec §5 与 M8 交接 M6-M8-07/08/09 的锚点与事实。

结论 **Ready to merge**：没有 Critical、Important，五处 Minor，T16 的记录提交处理：

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | “与布局根同一条件”只对第一次读成立：读到过之后的重读失败时（例如重连时的重新校验），正文显示读失败、页头仍给控件 | 改页头注释与 spec §2.6 的措辞；正文是否沿用已读到的偏好交 spec §5 的 P6（模块页一并决定） |
| M2 | M6-M8-07 把第二遍重读归给粘贴，trace 是两次撤销之后 6–9 ms | 主控以 trace 核对后改正 |
| M3 | 归档页行的菜单仍传两个空的 setter（继承自 Plane 的工厂形状） | spec §5 的 P7：动快捷操作时以较窄的参数类型去掉 |
| M4 | plan 第 185 行仍写“重载仍在” | 改为“同一人另开的浏览器重新载入仍在” |
| M5 | 新的 `dismissToasts` 遇到按下之后才来的通知会在 5 秒内失败（比旧版更严） | 不改：五个调用处之后都没有还会弹出通知的动作 |

