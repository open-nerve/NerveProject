# M6 / P3a 整体审核

> 2026-11-08：Ready to merge。目标：分支 `codex/m6-p3a` 相对 main `ddd79a9f5362aedfc33c940acecc2a5bf0e3d9b8` 的全部修改（`ae0ef51f..070762f5`），依据 [P3a spec](../v0/M6-cycles-modules/specs/P3a-issue-membership.md)、[P3a plan](../v0/M6-cycles-modules/plans/P3a-issue-membership.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[硬规则 §4](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。阶段记录见 [P3a 阶段记录](../v0/M6-cycles-modules/reviews/P3a-issue-membership-review.md)。

## 1. 方法

一次 fresh 只读审查（`fable`），在最终候选的七项门禁运行期间只读代码、不运行测试：逐个读差异中全部手写的 Go、TS、SQL，spec、plan、设计与硬规则，实施台账的全部裁定，预检 re-58，差异清单与交接的修改，并抽读承载各项声明的 bootstrap、app、适配器与 e2e 测试。原始报告没有另存文件，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **写入路径**：建立、PATCH、草稿发布三条路的拒绝次序合 R2（项目已归档 → 工作项已归档 → 无改动 → 开关（离开也算变化）→ 新加入的目标已归档 → 合并后的一个 422，未知目标排在其他字段之后、模块按请求下标）；“已结束”只在资产锁与读时钟之后的两处判定，只看加入的迭代，离开已结束或已归档的迭代可以；只检查新加入的，留下的不问、移出的只读名字；无改动判断认得两个字段（集合不计次序）；`skip_activity` 不会吞掉归属变化；动态的字段、动词、标识与次序合 §2.4，两种事件的 `Changes` 带上；访客与只看自己的调用者由工作项的规则判定，读到的值相同。
- **加锁与并发**：目标事实全在工作项写的事务与项目行锁下读取（planning 的读取经上下文中的事务）；项目开关与时区的修改、planning 的一切写都先取同一把项目行锁；订阅者的次序为项目行 → 迭代/模块行 → 归属行（一条语句按 id 锁定）→ 动态 → 工作项行，与工作项写不会交叉等待；两种次序的交错与跨午夜的“锁后判定”都有测试；移除归属在事务外运行即被拒。
- **删除与清理**：删工作项、删除任务（含剩余探查的分支）与清理的两步齐全，清理次序 `module_issues → cycle_issues → issues`、模块次序使其先于 `cycles`/`modules`；架构测试、`TestSQLCSchemaScope`、原始 SQL 与清理目录、运行角色守卫没有放宽，计数只按两张表增加；Go 夹具以 SQL 种归属，e2e 经接口。
- **回答**：七处投影的两列位置一致；`LEFT JOIN` 因部分唯一不会使行变多；`module_ids` 一律有序。
- **自动归档**：次序 `Lock → States → Now → OpenTargets → AutoArchive`，同一时钟、同一事务；排除只读 `issue` 的表；“未结束”合设计 §7 与 Plane。
- **架构**：端口由消费方声明、planning 实现、bootstrap 转换；两个模块互不导入；没有开关、桩或死代码。
- **文档**：差异清单各行引用 Plane 的文件与行并与代码一致；spec R10 与设计端口名已更正；输入交接回执与 M8 的 CI 观察与实现一致。

台账中的局部裁定（读不到被移出的目标答内部错误、`LEFT JOIN`、适配器测试的 `ANALYZE`、假时钟步长、`collaborationState` 加两张表、网页草稿白名单、一条语句的移除、`OpenTargets` 排在 `Now` 之后等）均在实现者的权限内且站得住。

## 3. 结论与处理

没有 Critical 与 Important。五条 Minor：

| 编号 | 问题 | 处理 |
|---|---|---|
| m1 | spec §2.1 的 `web/` 一行仍写“13 个文件”，实际只有 `fake-issues.ts` | 已改（T13） |
| m2 | “已结束/未结束”在 planning 内有 Go 与 SQL 两种写法，由 `TestOpenCycleIDs` 对照，但改 `StatusOf` 不会编译失败 | 设计 §7 写明 SQL 是 `StatusOf` 的取反、由该测试钉住（T13） |
| m3 | `issue/app/fakes_test.go` 422 行、`migrations/issue_schema_test.go` 460 行（名字清单） | 记入阶段记录，下次改到时拆 |
| m4 | `issue/app/membership.go:24` 注释笔误 | 已改（T13） |
| m5 | 一次加入迭代的写读两次项目事实 | 已知、无害，不改 |

Ready to merge: Yes——没有正确性、加锁、删除或架构缺陷。
