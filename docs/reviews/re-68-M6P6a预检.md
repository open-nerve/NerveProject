# M6 / P6a 预检

> 2026-11-13：预检通过（修正后），可以按 P6a 的 T1 开始实施。
> - **目标**：[P6a spec](../v0/M6-cycles-modules/specs/P6a-web-modules.md)、[P6a plan](../v0/M6-cycles-modules/plans/P6a-web-modules.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `aadaf61a5929b57d8aa53b93e5ae04b03cc046c6`（P5b 的 noff `9675bb31` 与其 main CI 失败的根因修正 noff，main CI 43373039561 首次通过）；审查对象为分支 `codex/m6-p6a` 的 `45cb4482`。
> - **本轮**：只有文档，没有产品修改或测试运行；I1 的守卫分支在草稿脚本里对 `aadaf61a` 的树试跑过（§3）。

## 1. 范围与方法

1. **调研**：两个只读调研代理在 main `aadaf61a` 上写出现状笔记（台账 `research/web.md`、`research/plane-contract.md`，随阶段归档）：
   - 网页：模块领域的文件与行数、旧 service 与 store、读 router 的地方、P5a/P5b 的约定（会话分代、取数门、宿主、共用件）、生成的客户端、守卫的现值与交接的模块各项；
   - Plane 与契约：Plane 的模块列表、预览、归档目录、详情与侧栏、弹窗与快捷操作、链接与元数据，nerve 的模块接口（P2、P4）与两者的差异。
2. **写 spec 与 plan**：据此把设计 §12 的 P6 拆为 P6a/P6b，写出 P6a 的 spec（R1–R16、A1–A8）与 16 个 Task 的 plan（`45cb4482`）。
3. **审查**：一次 fresh 只读审查对照 HEAD 的代码与只读的 `plane/` 逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在某个 Task 内变红而计划不修（knip、告警上限、关键词规则、i18n、store 数）；
   - R1–R16 是否站得住、有没有须负责人决定的事；
   - A1–A8 与 e2e 对 G9–G13、G19 与各交接验收条件的覆盖，以及 e2e 作业的时长；
   - 与接口契约、总体设计 §7.7 是否一致，对 Plane 行为的描述是否准确。
4. **裁定**：主控逐条核过审查的结论并裁定（I2 由主控按局部决定改定），再由写 spec 的代理改入。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **模块仍是 Plane 的形状**：模块领域约 73 个文件、6,770 行，调 Plane 的旧地址（`core/services/module.service.ts`、`module_archive.service.ts`）；`RootStore.module`、`moduleFilter` 不按会话分代，类型来自 `@nerve/types` 手写的 `module/`，`@nerve/constants` 的 `src/module.ts` 从那里导入版式、排序与状态的类型。
- **旧的模块筛选 store**：以 `reaction` 跟随 `router.projectId`，并把显示设置与筛选写进 `localStorage`（Plane 的 `module_display_filters`、`module_filters`，按浏览器、不按账户）；M3 交接 §1 记下退役的一代在每次换路由时仍写它。Plane 的迭代筛选 store 从不写 `localStorage`。
- **取数的条件分散**：迭代一侧有七处各写自己的“角色、未归档、`cycle_view`”组合（列表页、命令面板的迭代菜单、详情页、归档目录、进行中卡片的优先级页签、迭代页的列表根、迭代详情页头），包装层另以“有效成员 + `cycle_view`”取迭代的列表。
- **页头读 router**：模块的两个页头以 `const { workspace: { workspaceMemberIds } } = useMember()` 读 router 驱动的 getter；`project-address-route` 的内容模式只认一层解构。
- **守卫现值**：oxlint 告警 53（上限同值），其中模块文件 9 条；`root.store.test.ts` 23 个 store；关键词 77 条规则、10 条例外、phase `M6/P5`；knip 6.37.0。
- **钉住旧状态的冒烟故事**：M4 的 `s2-issue-list-removal.spec.ts` 的 `module`、`archived modules` 两条钉的是“请求旧地址得 404、显示英文的 ‘Module does not exist’、归档目录不取数”。

## 3. 结论与处理

结论 **Ready after fixes**：两处 Important、六处 Medium，没有架构问题；I2 经主控的局部裁定后没有须负责人决定的事。全部在实施前改入 spec 与 plan，Task 数不变（16 个）：

| 编号 | 问题 | 处理 |
|---|---|---|
| I1 | spec 写负责人与成员的筛选列 `workspaceMemberIds`，它读 router store 的 `workspaceSlug`（`workspace-member.store.ts:156-161`）；两个模块页头今天以嵌套解构读它（`module-view-header.tsx:40-41,167`、`archived-modules/header.tsx:39-40,152`），`project-address-route` 认不出这种写法，守卫会一直绿着 | spec R14、§2.4：读按 slug 的 `getWorkspaceMemberIds(workspaceSlug)`（`:169`），slug 取自路由参数。§2.10：守卫的内容模式加嵌套一层的分支（四个 getter 不变），`samples.hit` 加两例嵌套解构、`samples.miss` 加 `workspace: { getWorkspaceMemberIds }`。先试跑：规则文件表现有的 121 个文件命中 0 个，可以加；表外命中 9 个（模块的两个页头、成员下拉框、项目与工作区成员页等），不在规则范围内。plan：T8（第一个把模块页头加进文件表的 Task）加分支，先看到 hit 样例失败（变异 `t8-mut-guard-flat-only`）；T10、T11 的两个页头加入文件表的同一提交改读 `getWorkspaceMemberIds`；T11 以“router 的 slug 指向别的工作区时仍列路由那个的”测试钉住，变异 `t11-mut-members-from-router` |
| I2 | R4 把版式与排序随筛选一起从 `localStorage` 拿掉，以 P5a R4 为先例，但 Plane 的迭代筛选 store 从不持久化，先例不覆盖“去掉持久化”；不写 `localStorage` 的理由（“不按账户”）同样适用于 spec 保留的 `module_sidebar_collapsed` 与 `module-progress-tab-{id}`，前后不一致 | 主控裁定（局部决定）：版式与排序是纯展示的设置，在组件里经 `useLocalStorage` 按项目、按浏览器存，默认 `list`、`name`，同 Plane；筛选与搜索按项目 id 留在会话的 `ModuleFilterStore`，没有反应、不写 `localStorage`，理由是筛选里有成员的 id，不能在同一浏览器里带到另一个账户；删去不一致的理由。登记差异：重新载入后筛选与搜索回到空，版式与排序照 Plane 保留。spec R4、D1、§2.4（store 的形状去掉版式与排序，另立显示设置的 hook）、§2.6、§2.12、A1；plan T4（store 的测试与变异 `t4-mut-filters-persist`）、T10（hook 的默认、按项目、读回、非法值回默认，变异 `t10-mut-display-key-no-project`）、T11（归档目录读同一个排序）、T15 的差异措辞 |
| M1 | T2 同时做 service、store 的读、包装层与取数门的抽取，偏重；“开关的条件只写一处”漏了三处：`priority-issues.tsx:59-63`、`cycle-layout-root.tsx:60-74`（P5b H1 另看 `cycle_view`）、`R/cycles/(detail)/header.tsx:67-84` | T2 只留 service、store 的读、假件、hook 与包装层的取数，模块的条件写在包装层迭代的条件旁。T5（以迭代既有测试为基线的重构 Task，在第一个模块页面 T8 之前）抽出取数门与开关的谓词、逐处改用（变异 `t5-mut-gate-archived`）。spec §2.3 逐处写定：优先级页签、迭代详情页头用“列表可取”；迭代页的列表根以“可读”决定渲染、以“列表可取”决定停用空态（它只在包装层之下渲染，已归档的项目到不了它，多出的“未归档”一条不改行为）；列表页与两个命令面板菜单用“列表可取”；详情页、预览与两个归档目录用“可读”；包装层用开关的谓词、成员判断照旧。导航与侧栏的标签、`archive-tabs-list`、命令的可见性（`create_module` 改用写的门）、`useCycleListState` 的“可加入”与写的门回答“显示哪个入口、能不能写”，不是取数门，留着自己的条件 |
| M2 | T6 的宿主要持有归档与删除的确认框，但旧的两个收 `IModule` 与旧 store 的参数（`delete-module-modal.tsx:20-24`、`archived-modules/modal.tsx:18-30`），T6 接不进收 `Module` 的宿主 | spec §2.7；plan T6 的宿主只接修改弹窗与 `pageMounted`，T7 改写两个确认框时接入宿主并测“行卸载后确认照常提示、宿主卸载后迟到的回答只进 store” |
| M3 | 只给界面用的版式、排序、状态类型放在哪里没有写定，T12 删除 `@nerve/types` 的 `module/` 之后没有落脚处 | spec §2.4、§2.9：`ModuleLayout`、`ModuleOrderBy` 在 `@nerve/constants` 的 `src/module.ts` 由选项数组推出（先例 `constants/src/cycle.ts:7-14` 的内联联合），状态用 `@nerve/api-client` 的 `ModuleStatus`，`MODULE_STATUS` 随之改；`ModuleFilters` 在 `@nerve/utils` 的 `planning.ts`。核过 `@nerve/utils` 已依赖 `@nerve/constants`（`utils/package.json:27`，`src/tab-indices.ts` 已从它导入），`orderModuleRows` 可以从它取类型。plan T4 建立，T10 换掉两个旧读者，T12 删除 `types/src/module/*` |
| M4 | plan 写“只给测试读的导出不算读者”，像是 knip 会查；knip 6.37 把 vitest 的测试文件当入口，只被测试读的导出在它看来仍有读者 | Global Constraints 改写：这是本段更严的规则，由逐 Task 的审查执行，不是 knip 能查出的 |
| M5 | 新增 5 个页面故事，e2e 作业的时长只在 T10 检查点核一次，那时还没有新故事 | spec §2.11、§6；plan 的检查点一段、T13、T14、T16：T13、T14 从本机三种引擎的运行记下每个新文件的时长；T16 推送之前估出 CI 的 e2e 时长（main CI 43373039561 的 e2e 19 分 14 秒，加上新文件 chromium 的时长乘以迭代页面故事 CI 与本机之比），期限 35 分钟，估出或实测超过 30 分钟先报告负责人；写明 T10 检查点量不到新故事 |
| M6 | spec §6 写“58 个 web 文件”，审者数出 57 | 重数一遍是 58，其中含 3 个测试与 propel 自己的 `TModuleStatus`（同名不同物），按读者算是 57；改为不写死的“五十余个（含几个测试与它们自己的定义）”。归档的字节数仍以 T1 的 `archive.py verify` 收据为准 |

## 4. 审者核实为正确的事实（摘要）

- oxlint 告警 53，其中模块文件 9 条：`module-card-item.tsx` 5 条、`applied-filters/root.tsx` 2 条 `prefer-set-has`、`dropdowns/filters/root.tsx` 的 `autoFocus`、`archived-modules/modal.tsx` 的正 `tabIndex`；plan 的下调 53 → 52（T7）→ 47（T10）→ 44（T11）与之对应。
- `root.store.test.ts` 的 store 数 23 → 21（T12 删除 `module`、`moduleFilter`）。
- S2 冒烟故事的 `module`、`archived modules` 两条确实钉着本段要改掉的旧状态，plan 在 T8、T11 移出、由 T13 的 G13 接上。
- `MemberDropdown` 带 `projectId` 时只列不是访客的项目成员（`dropdowns/member/dropdown.tsx:34` 的 `getProjectMemberIds(projectId, false)`），R6 的“已经不给访客”成立。
- T1 回填的事实（分支 CI 与两次 main CI 的运行号与作业号、两个 noff 的 SHA、双父与树、归档的文件数与清单 sha256、清理记录）与 P5b 的归档一致。

## 5. 负责人须知

没有须负责人决定的事。I2 由主控按局部决定改定：版式与排序同 Plane 存在浏览器，筛选与搜索只在会话里，差异登记在差异清单；R3（取数门）、R5（已归档菜单里没有删除）、R6（换项目清空负责人与成员）、R15、R16（G15 的模块部分与工作项根 store 的拆分在 P6b）都在已批准的设计之内，是本地决定。
