# M6 / P5b 预检

> 2026-11-11：预检通过（修正后），可以按 P5b 的 T2 开始实施（T1 的回填已在 `5d9b1e2e` 完成）。
> - **目标**：[P5b spec](../v0/M6-cycles-modules/specs/P5b-cycle-list.md)、[P5b plan](../v0/M6-cycles-modules/plans/P5b-cycle-list.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `0b8efcd701edf7e73a37d5bb82ad176a7795b910`（P5a 的 noff，main CI 42995581409 首次通过）；审查对象为分支 `codex/m6-p5b` 的 `a23270ea`。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：P5a 合入前，两个只读调研代理在 P5a 的树（`cbfb0d50`，网页代码与 main `0b8efcd7` 相同）上写出现状笔记，随台账归档：
   - 网页：M4 列表引擎的全程（store 类型与路由映射、工作项的根、绑定与续页、偏好的筛选 store、成员判断与写的确认）、四种布局与页头、快速新建与新建弹窗、生成的客户端与 P3b 的决定、P5a 留给列表的位置、附件列、已结束迭代的只读、守卫与 e2e 夹具；
   - Plane 与契约：Plane 迭代页的列表（布局根、页头、CYCLE 的 store 与筛选、只读、空态、拖拽）、迭代的用户偏好、进行中卡片的优先级页签、卡片与侧栏的行点击、“添加已有”，nerve 的列表与偏好接口、工作项的 `cycle_id`，P5b 要回执的交接。
2. **写 spec 与 plan**：据此写出 P5b 的 spec（R1–R11、A1–A8）与 14 个 Task 的 plan（`a23270ea`）。
3. **审查**：一次 fresh 只读审查（`fable`）对照 HEAD 的代码逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在某个 Task 内变红而计划不修（knip、告警上限、关键词规则、i18n、store 数）；
   - R1–R11 是否站得住、有没有须负责人决定的事；
   - A1–A8 与 e2e 对 G15、G19 与各交接验收条件的覆盖；
   - 与接口契约、总体设计 §7.7 是否一致，对 Plane 行为的描述是否准确。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **列表引擎没有迭代**：`EIssuesStoreType` 没有 CYCLE，迭代的路由映射到 DEFAULT 并有测试钉住；`useIssues`、`useIssuesActions`、空态、`ISSUE_STORE_TO_FILTERS_MAP`、`TCreateModalStoreTypes`、拖拽的联合都没有它。
- **成员判断不认迭代**：`issueMembership` 不读查询的 `cycle_id`、`module_id`；`IssueRootStore.fields()` 没有 `cycle_id`、`module_ids`；创建与更新的白名单去掉 `cycle_id`。
- **读筛选靠 router**：四个 base root 与日历子树经筛选 store 读 router 驱动的 getter；项目页的偏好跟进固定是 PROJECT。
- **偏好**：`GET/PATCH /api/v0/me/cycles/{id}/preferences` 已有，生成的客户端已有操作，网页还没有调用方；`CyclePreferences` 只有三个字段，`filters` 没有 `cycle_id`。
- **P5a 留的位置**：详情页的正文只有转移横幅，页头没有布局、筛选、显示与新建；统计卡只有负责人、标签两个页签；卡片与侧栏的行不可点。

## 3. 结论与处理

结论 **Ready after fixes**：两处 High、六处 Medium、九处 Low，没有架构问题，没有须负责人决定的事。全部在实施前改入 spec 与 plan，Task 由 14 个拆为 16 个：

| 编号 | 问题 | 处理 |
|---|---|---|
| H1 | spec 写“迭代关闭时页面已显示停用的空态（P5a）”，实际只有列表页有；详情页在迭代关闭时照样会挂列表、取偏好，而 nerve 在迭代关闭时拒绝工作项离开迭代 | spec §2.1：详情页在迭代关闭时，列表区改为与列表页相同的停用空态，不取偏好与列表，页头不显示列表的控件；T8 加 RED（没有这两种请求）与变异 |
| H2 | “读筛选的五处”漏了日历子树的十处读（经整个筛选 store 读 router 驱动的 getter），迭代页的日历会读到 `undefined` | spec §2.5 写明；拆出 T6：读本页筛选的 hook 覆盖四个 base root 与日历子树，`CalendarChart` 改拿已解析的日历显示设置与更新回调 |
| M1 | 卡片点击在迭代偏好从未读过时走 `change()`，而它没有已读的值就抛错；打开预览的次序未定（预览是详情 store 的状态加宿主归属） | spec §2.4：整字段替换不依赖已读的值，以回答种下本地值并升版本；§2.7：先进入迭代页、再在它的宿主下打开预览；T3、T4、T12 加 RED 与变异 |
| M2 | 已结束、已归档的只读没有杠杆：快速新建、分组头“+”与日历的只读只看路由权限 | spec §2.6 写明机制；拆出 T7：四个 base root 加一个只读的 prop，并进 `disableIssueCreation`、日历的 `readOnly` 与 `canEditProperties` |
| M3 | 新建弹窗可以换项目，`createIssue` 无条件带本迭代的 `cycle_id` 会让别项目的新建以 422 失败 | spec §2.5：目标项目是迭代的项目时才带；迭代页页头的弹窗锁定项目；T5 的回归用例只改 store 类型 |
| M4 | 绑定在构造时登记进根、根的列表只增不删，优先级页签的 store 若按组件建会无限增长 | spec §2.7：`IssueRootStore` 构造时建一个、随根退役，每个迭代一个绑定；T11 加“挂载两次只登记一个绑定” |
| M5 | M4 交接 M6-02 的“失权迟到答”与“归档范围”没有进页面故事 | spec §2.10、A5 写入；T13 加归档范围，T14 加失权的迟到回答 |
| M6 | `project-address-route` 的收入没有落到任何 Task | Global Constraints 写明；T6、T8、T9、T12 的文件表加 `tools/keywords.json` 与样例 |
| L1 | 新文案（移出迭代、优先级页签与空态）没有现成的键 | Global Constraints：en 与 zh-CN 同一提交加入 |
| L2 | “双向续页”在 spec 与 plan 的措辞不同 | 统一为 M4 的双向分页（`expectPages`、`scrollToLastListRow` 的既有写法） |
| L3 | M4 交接 M6-01 的“整页零非预期失败” | T13 每种布局以 `expectListPageQuiet` 收尾，A5 写入 |
| L4 | 快捷操作的修改与“复制一份”写死 PROJECT，复制会不带 `cycle_id`；预览宿主与项目页共用 | spec §2.6：以 CYCLE 打开，预览宿主带迭代 id；T8、T10 |
| L5 | 写后重读进度的触发点不确定 | spec §2.6：迭代页列表的动作确认后经 `followInSession` 与页面范围的捕获重取，预览里的写不算 |
| L6 | 迭代页的显示选项常量若无读者，knip 变红 | T5：同一 Task 由 `ISSUE_STORE_TO_FILTERS_MAP[CYCLE]` 引用 |
| L7 | spec 罗列的拒绝码与服务端的次序不同 | T9：测试不断言拒绝码之间的次序 |
| L8 | R11 的回执措辞 | spec R11、T15 写明：列表在 P5b 先开放，“添加已有”依设计 §12 的 P7 一行与 §15 第 12 条在 P7 |
| L9 | T5、T6 各逼近一个实现者的上限 | 拆为 T5（引擎）、T6（读本页筛选与日历）、T7（只读）、T8（布局根与页面），共 16 个 Task |

## 4. 审者核实为正确的事实（摘要）

- 钉住 CYCLE 的三处测试、`root.store.ts` 的两个白名单与 `fields()`、`issueMembership` 不读迭代与模块、`useIssueListMount` 的键与 `allowed`、`ISSUE_STORE_TO_FILTERS_MAP`、`TCreateModalStoreTypes`、各处 store 类型的分支、日历固定读 PROJECT 的 `viewFlags`、偏好跟进只认 PROJECT。
- `BaseIssuesStore.select` 总是改写当前列表，R7 另用实例的理由成立；`IssueListScope` 的查询已含 `cycle_id`，R1 复用 `scope: "project"` 不改 `read()`、`covers()`。
- 偏好接口与 `listIssues` 的 `cycle_id`、`IssueCreate`/`IssueUpdate` 的 `cycle_id` 与 P3a、P3b 一致；守卫的现值（告警上限 53、`root.store.test.ts` 23 个 store、77 条规则 10 条例外、phase `M6/P5`）；三种引擎的页面配置能匹配新故事；T1 的归档数字。
- Plane 的行为（卡片三个页签与默认页签、优先级页的查询与续页重排、卡片替换与侧栏切换、预览里不可点、已结束整页只读而服务端只拒绝加入、两步新建、按迭代分组只在界面隐藏）与 spec 登记差异的说法正确。

## 5. 负责人须知

没有须负责人决定的事。R1（复用项目列表加 `cycle_id`）、R3（偏好的筛选 store 抽基类，项目页的测试断言不改）、R5（已结束、已归档只读照 Plane）、R6（移出迭代）、R11（“添加已有”在 P7）都在已批准的设计之内，是本地决定。
