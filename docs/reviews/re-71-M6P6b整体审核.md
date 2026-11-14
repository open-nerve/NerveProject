# M6 / P6b 整体审核

> 2026-11-14：Ready after fixes；修复轮 F 已提交，待限定复审（§4）。
> - **目标**：分支 `codex/m6-p6b` 相对 main `06b72d4b65bcba911344a4ed3c358f164a5fdeeb` 的全部修改（`06b72d4b..513d2ff3`，`ca4e62ac` 至 `513d2ff3` 的 18 个提交）；修复轮 F（`5b1046b7` 起的九个提交，F1–F9：F5 并入 F2，F9 的文档另成一个提交）另经一次限定复审（§4）。
> - **依据**：[P6b spec](../v0/M6-cycles-modules/specs/P6b-module-list.md)、[P6b plan](../v0/M6-cycles-modules/plans/P6b-module-list.md)、[预检 re-70](re-70-M6P6b预检.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。

## 1. 方法

一次 fresh 只读审查（`fable`，台账 Task 16），范围 `06b72d4b..513d2ff3`（T1–T15 的全部提交），不改动文件与 git 状态。它对照的依据见上；工作项写入模块的集合对照 nerve 的 `server/internal/modules/issue/app/membership.go`，文档的准确性对照差异清单与回执引用的 Plane 源码（主目录的只读 `plane/`，抽查约 30 处）。

原始报告没有另存文件；结论、核对过的性质与各条发现由主控转述，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **工作项根 store 的拆分**（R1）：照搬不改，写入单元对根没有值的循环导入，合并后没有留下转发。
- **两域共用的件**（R2）：没有按域的分支；迭代测试的断言只改了 `cycleID` → `entityID` 与契约变化处的一例。
- **列表引擎的 MODULE**：store、动作、路由映射、挂载键、本页筛选与空态的接线正确。
- **模块集合的写入**：本页新建、移出与修改的请求体与 nerve 的 `membership.go` 一致（整集合、轮到时算、保留已归档与别的模块）。
- **三页的偏好读取**（R3、R4）与按页收窄偏好（R16）。
- **其余**：写后的后续动作只在 nerve 确认之后；会话分代下的迟到回答不上身；旧件删除完整；守卫（`project-address-route`、swr 的限制、告警上限、knip、关键词、i18n）；测试与 e2e；文档与其中 Plane 的引用。

## 3. 结论与处理

结论 **Ready after fixes**：没有 Critical、Important，六处 Minor（M1–M4 即 spec §5 预先列出的修复轮清单，M5、M6 是新的）与四条观察，没有须负责人决定的事。修复轮 F 处理六处 Minor 与一条观察（主控裁定现在收紧），另三条观察不需动作；修复轮中另见的两处同类边缘由 F9 修正（主控裁定）：

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | 测试帮手的重复：`roots/module-row-dialogs.test.tsx` 复制了 `row-dialog-lifetime.test.tsx` 的 `outlivesTheRow`；`module-create.test.tsx` 与 `cycle-create.test.tsx` 重复替身与帮手；`module-quick-actions.test.tsx` 的弹窗与编辑器替身、`preferences-reread.test.tsx` 与页头测试的装置又各一份 | F1 把帮手移进按域的替身 `roots/fake-planning-page.tsx`（作用于 `planningPageTab` 最后打开的页，调用处不改），弹窗与编辑器的替身移进新的 `issue-modal/fake-dialog.tsx`（`vi.mock` 仍各在本文件，工厂导入它）；迭代的测试只改导入，18 个文件 201 例同名同数（`5b1046b7`） |
| M2 | 列表状态不核对实体属于路由的项目：缺口在 `usePlanningPageHeader.canCreate` 与 `PLANNING_PAGE_DOMAINS` 的 `joinable`（正文与模块页头的面包屑已核对）；store 持有项目 P1 的模块 X 时打开 `/P2/modules/X`，页头仍给“新建工作项”，全局新建仍带上 X；迭代同样 | F2 两个纯函数在项目在、而实体的 `project_id` 不是它时同“页面还没有这个实体”：只读、不可加入；新测试 8 例（RED 6 例，另两例是迭代单测的对照），迭代的测试只增；变异“不看项目”两域各三例失败（`11b9ea74`） |
| M3 | `core/store/issue/cycle/filter.store.test.ts` 旁边已没有同名的源文件 | F3 以 `git mv` 改名为 `cycle/page-preferences.test.ts`，与模块的 `module/page-preferences.test.ts` 相称，内容不改（`9b67b78c`） |
| M4 | `cycle/priority-issues.store.ts` 文件内的 `CyclePage` 与两域共用的 `PlanningPage` 相近 | F4 保留，加一行注释：优先级列表迭代独有（spec §2.2），`entityID` 只会让它说得更含糊（`238a29d2`） |
| M5 | `use-cycle-list-state.ts` 的 `CycleListState` 导出而文件外没有读者 | F5 去掉导出，与 F2 同一文件，并入 F2 的提交（`11b9ea74`） |
| M6 | `module/issue.store.test.ts` 的“移出被拒”以 400 答 `planning.modules_disabled`，nerve 答 409（`membership.go:29-30`） | F6 改为 409，断言不变（`36e23061`） |
| 观察 | `.oxlintrc.json` 的 swr 限制没有列出六个既有的 `useSessionSWR` 读者 | F7 加入五个：`browse/[workItem]/layout.tsx`、`inbox/content/description-versions.tsx`、`power-k/ui/modal/search-menu.tsx`、`hooks/use-project-read.ts`、`hooks/use-current-user-fetch.ts`（都没有从 swr 导入值，只有前者一行类型导入）；第六个 `workspace/settings/use-members-settings-fetch.ts` 已由表中的 `core/components/workspace/settings/**` 覆盖；改前往 `search-menu.tsx` 加 swr 的值导入 oxlint 通过，改后失败；告警上限仍为 44（`d302141f`） |
| 观察 | 列表的动作 hook 由每个调用者各自实例化 | 不需动作 |
| 观察 | 四种布局（列表、看板、日历、表格）各有一个写法相同的 `roots/planning-page-root.tsx` | 有意如此（照各布局既有的按页的根），不需动作 |
| 观察 | 模块的 G15 页面故事 `g15-module-list-page.spec.ts` 恰好 400 行 | 在约 400 行的上限之内，不需动作 |
| 修复轮中发现 | 与 M2 同类、审查没有列出的两处边缘：页头的 `preferencesShown` 与数目（`usePlanningPageHeader`）只看本页偏好是否读到过，不看实体的项目——先在 P1 的模块页读过 X 的偏好，再打开 `/P2/modules/X`，页头仍给布局与显示的开关（临时探针证实；迭代经同一个 hook，同理）；迭代页头的面包屑标题与 `CycleQuickActions` 不核对 `project_id`（模块页头自 P6a 起核对）。都只在手改 URL 时出现，nerve 按实体自己的项目授权，没有越权 | F9 以 `core/hooks/planning-page-entity.ts` 的 `pageEntity` 为“本页持有”的唯一说法（F2 的项目核对移来，两个纯函数改调它）；两个列表状态的 hook 另给 `held`（已归档、已结束的照样算持有），页头的数目与偏好的控件要 `listed && held`，没有按域的分支；迭代页头照模块页头只在迭代属路由的项目时给面包屑标题与快捷操作。两个页头测试各加一例（先显示 P1 的页，再渲染 `/P2/…` 的页头：没有数目、布局、筛选、显示、新建，迭代另核面包屑与快捷操作），能编译的变异五个（两域的 `held` 不看项目、面包屑用别的项目的迭代、快捷操作不核对项目、页头不看 `held`）（`09aba5f8`） |

F8 写本记录，并改 spec §5 的修复轮清单、spec 与 plan 的状态、设计 §14 的 P6b 一行（`45c5e665`）；F9 之后的文档提交把 F9 记进本节与 spec §5。

主控对修复轮的两处偏离的裁定：
- F7 只加五个、不加第六个：接受——`core/components/workspace/settings/**` 已覆盖它，改前往它加 swr 的值导入 oxlint 就失败。
- F1 在迭代测试里多改的导入行（`cycle-create` 的两处 `vi.mock` 工厂改为导入共用的替身，`cycle-quick-actions` 与迭代页头测试改为导入逐字相同的帮手）：接受——都是导入行或换成相同帮手的导入，断言不变，在回归边界之内。

## 4. 修复轮 F 的复审

（待限定复审之后由主控填写。）
