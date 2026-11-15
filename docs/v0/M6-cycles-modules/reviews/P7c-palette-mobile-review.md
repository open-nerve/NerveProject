# M6/P7c 阶段评审记录

> 实现、整分支审查与修复轮完成（2026-11-16），待最终候选的门禁、分支 CI 与合入（T15，主控）。整分支审查 Ready to merge after fixes（[re-77](../../../reviews/re-77-M6P7c整体审核.md)：一处 Important、两处 Minor，没有须负责人决定的事），主控三处都接受，以修复轮 F 处理（F2 `98d9f985`、F3 `10ca6ae6`、F1 `165b252b`；F4 `396bd5f7` 写本记录，只改文档），修复轮另经一次限定复审（re-77 §4：Ready to merge，一处 Minor m1 以 F5 `6b79fee4` 修）；修复轮另发现的两处同一类的问题交 P8 与 M8（§6）。最终候选的门禁、推送前的估算与推送后的分支 CI 见 §5；合入、main CI、归档与清理见 §7（由 P8 的 T1 回填）。基线 main `d052213f82aa4fba234f5684722335e1a5cbb5be`（P7b 的 noff，双父 `f42f9d64`、`4fe308ce`，树与 `4fe308ce` 相同）。

## 1. 评审方式与范围

P7c 交付设计 §12 的 P7c 一行（P7 在实施之前拆为 P7a/P7b/P7c，[P7a spec](../specs/P7a-work-item-planning.md) §1、R1）：命令面板（先按层级拆导航的命令；规划入口一条规则，打开的菜单门为假时说原因、不再转圈；七个菜单的空文案经 i18n 的键；搜索在项目范围给迭代、模块两组；工作项的“改迭代”“改模块”两个命令）；快捷操作的“恢复”在途时禁用与归档页的行菜单的类型（A17、A18）；项目页、迭代页、模块页共用的移动端页头（X2、A15）与迭代页的窄屏顶行；分组时页头的数目（A19）、没有读者的键（A20）、规划页替身的 `opened`（A21）与“添加已有”打开即聚焦搜索框；设计 §8 的四个视图文件（X1）；G19 的工作项一侧的页面故事。归档页的词表与其余的遗留在 P8（[spec](../specs/P7c-palette-mobile.md) §1、§5）。

- **调研**：两个只读调研代理在 `d052213f` 上写出本段的现状笔记（台账 `research/palette-views.md`：命令面板、快捷操作、四个视图文件、G19 的工作项一侧与“添加已有”的焦点；`research/mobile-carryovers.md`：移动端页头、A19–A21、交接回执、e2e 的余量与放置、估算），随台账归档；spec 引用的行号都在 `d052213f` 的树上。
- **界定**：主控在起草 spec 之前裁定十六件事（H1–H16，台账 `controller-rulings.md`）：15 个 Task；规划入口一条规则（ADMIN/MEMBER、项目没有归档、开关打开），侧栏的两张声明表不改；先按层级拆 `navigation/commands.ts`（466 行）；七个菜单的空文案经构造件的键；搜索在项目范围给两组（本地过滤、未归档、含已结束的迭代）；工作项的“改迭代”（`shift+c`）、“改模块”（`shift+m`），选项同下拉框、写经 `IssueChange` 的增减；A17、A18；一个共用的移动端页头（项目页改用，X2）；两页的窄屏顶行一条规则；A19 按范围读；A20、A21 与打开即聚焦；X1 只做地址的读与写之后的跟进；e2e 一个新文件 ≤ 8 秒；余量取保守的 14.6–19.3 秒、检查点在 T10；只改网页；行数与质量。主控据此写出 [spec](../specs/P7c-palette-mobile.md)（R1–R16 照录 H1–H16；A1–A14）与 15 个 Task 的 [plan](../plans/P7c-palette-mobile.md)（`bee18397`）。
- **预检**：实施前一次 fresh 只读预检（[re-76](../../../reviews/re-76-M6P7c预检.md)），结论为修正后可实施：四处 Important（F1 三个 layout 不是观察者，门要在每页一个 `observer` 包装件里算；F2 移动端的故事证不了“偏好答回之前不给”，改由单测与 `t9-mut-no-gate` 证；F3 原因的码经 `problemMessageKey` 说明，`errorMessageKey` 收的是错误对象；F4 “恢复”的同步 ref 照旧守卫发送，状态只管禁用的显示）与十二处 Minor（F5–F16，F16 是过程：审查时 worktree 里有与预检并行的 T1 的未提交改动），主控全部接受，在实施前改入 spec 与 plan（`78fa4dbc`）。主控在预检之后另有两条裁定：F1 取“每页一个薄的包装件算门”，不取“共用的件自己按 store 类型算门”；F14 的读定为打开的菜单的门控 `useSessionSWR` 加纯函数，不留给 T6 选。F8 的一处事实在 T6 更正：审者说 Plane 的两个工作项命令没有快捷键，实为 `modifierShortcut: "shift+c"`、`"shift+m"`（`plane/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts:331`、`:363`；调研笔记是对的），实现照 spec 写 `modifierShortcut`、不受影响，re-76 与 spec 的说法由 `3774b98e` 更正，差异清单写“与 Plane 相同”。
- **实施**：T1–T14 由实现代理完成，主控逐 Task 复核并记台账。拆分与纯搬移（T2，T7 的重构）先以既有测试留基线收据，结束时逐文件同名同数。主控复核时的裁定与跟进：
  - **T3 的 `project.not_found`**：原因函数在没有角色或项目不在 store 时回 `project.not_found`（nerve 读这一域的列表 `listCycles`、`listModules` 时的回答，`api/modules/planning.yaml:29`、`:331`），不再用读单个迭代、模块的 `planning.cycle_not_found` / `planning.module_not_found`；spec §2.2、A2 与 H2 随之改（`f5ef9548`）。
  - **T7 的重构提交**：在 RED 之前另立纯搬移的 `165420ef`——“门为假时说原因”在第四处出现时合为一件 `PowerKPlanningListRefusal`，命令面板守卫之内的写原样抽为 `useWorkItemWrites`（power-k 173 例逐例同名同状态），主控接受。
  - **T10 的两行 mock**：迭代页、模块页的 `page.test.tsx` 各加一行新包装件的 `vi.mock`（新的导入链经 `store-context` 在导入时建 `RootStore`，这两个测试的替身没有 `apiFor`；它们本已 mock `AppHeader`，窄屏一行从不渲染；断言与例数不变），是回归边界的一处环境改动，主控接受。
  - **F8 的事实更正** `3774b98e`（见上）。
  - 复核时另记两条交 P8 的观察：项目页上访客照旧看到项目设置与视图的命令（T2，M3、M4 的行为）；归档页行菜单的 “Restore” 字面量（`quick-action-dropdowns/helper.tsx:219`，T8），与快捷键说明的 “No shortcuts found for” 同一类。
- **门禁与审查**：T10 一次检查点（七项门禁与分支 CI，CI 的实测作 T15 估算的校准点）；T15 在 `5087b09e` 上的第一次最终门禁（`t15-gate`）与一次 fresh 只读整分支审查（[re-77](../../../reviews/re-77-M6P7c整体审核.md)）并行，修复轮 F 另经一次限定复审；最终候选的七项门禁、三种引擎的页面配置，与推送前的 CI 时长估算（spec §2.12）。

## 2. 验收状态

A1–A13 满足（[spec §4](../specs/P7c-palette-mobile.md#4-验收标准)已勾选），A14 合入后由 P8 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 拆分：`navigation/commands.ts`（466）按层级拆为 `workspace-commands.ts`（259）与 `project-commands.ts`（252，T3 之后 253），`commands.ts` 删去、由 `root.ts` 组合；逐段对齐的搬移收据（`t2-moved-lines`，454 行非空行搬入，唯一新写的是胶水）；特征测试 6 例拆前拆后同名全过，power-k 既有 12 个文件 122 例同名同数；项目一级一件在 `project-address-route` 里 | T2 |
| A2 | 入口规则：`planningEntryOn`、`planningListRefusal`（`core/lib/planning-switch.ts`，+19 例，含工作区管理员而项目访客映射为 ADMIN 的一行）；四个导航命令改调入口规则，特征测试“项目已归档”一列的四格由给改为不给（契约变化）；两个打开的菜单门为假时以 `problemMessageKey` 说出具体的键、不转圈、持有的列表也不列；没有角色时是 `project.not_found`（T3 主控裁定）；变异 `t3-mut-archived-entry`、`-guest-entry`、`-spinner`、`-held-list`（两个菜单各一次）、`t3-mut-not-found-code` 都失败 | T3 |
| A3 | 空文案：构造件只收键（`emptyTextKey`，类型 `power_k.${string}`），七个新键 `power_k.menus.no_*`（en 与 zh-CN），默认 `power_k.search_menu.no_results` 只写在 `empty-state` 一处；`menus/builder.test.tsx` 10 例以真的 i18n 测两种语言；变异 `t4-mut-literal` 等 3 个与两个类型收据 | T4 |
| A4 | 搜索：`usePlanningSearch`（项目范围、入口规则、未归档、含已结束的迭代、本地过滤）；store 没有时以项目页的键读；读失败或没回答时这一组为空、不挡别的；“没有结果”四组都空时才给；`search-menu.test.tsx` 既有 12 例一行不改、+8 例；变异 `t5-mut-workspace-scope`、`-no-filter`、`-guest`、`-no-results` 与四个探针 | T5 |
| A5 | 改迭代：`add_work_item_to_cycle`（`modifierShortcut: "shift+c"`）、`work-item/cycles-menu.tsx`（“无迭代”在前、不给已结束与已归档的、所持的已结束或已归档的以名字显示为当前且禁用、选所持的不发、请求体 `{cycle_id}`、守卫之内）；`cycle-command.test.tsx` 18 例与表格一行；变异 `t6-mut-completed-option`、`-held-sends`、`-no-guard` 与六个探针。修复轮 F2：每行以 id 作 cmdk 的 value、名字作 keywords，两个宿主共用 `powerKFilter`；+3 例与 `filter.test.ts` 6 例，变异 `f2-mut-name-value` 与四个探针 | T6、F2 |
| A6 | 改模块：`add_work_item_to_modules`（`shift+m`）、`work-item/modules-menu.tsx`（复用 `menus/modules.tsx`，加回勾选的 `value` 与已归档的说明）；每选一项一个增或减（以 nerve 确认过的勾判断），请求体在写队列轮到时经 `changeBody` 以所持的集合算，列表不关；类型测试让整集合赋不进去（`t7-type-whole-set` tsc exit 2）；`module-command.test.tsx` 19 例；变异 `t7-mut-whole-set`、`-archived-hidden`、`-closes` 与八个探针 | T7 |
| A7 | A17、A18：“恢复”在途时两个菜单都禁用（状态管显示、同步的 ref 守卫发送），回答之后可用；两个 `actions-host` 测试各 +2 例；归档页的行菜单只收 `ArchivedIssueMenuProps`（类型测试 `archived-issue.types.test.ts`，`t8-type-setters`、`-widened` tsc exit 2）；变异 `t8-mut-restore-enabled`、`t8-mut-no-sync-guard`（两域）与一个探针；键盘没有单列的例（T8 的裁定，spec A7 注明） | T8 |
| A8 | 移动端页头：`usePageDisplay` 由宽屏的 `HeaderFilters` 与新件 `MobileHeaderFilters` 共用；项目页的包装件（98 → 33 行，手写的读写删去）与规划页的 `PlanningPageMobileHeader` 都是 `observer`，经 layout 挂载、偏好后答回的测试；`preferences-controls.test.tsx` 移动端 4 例同数、断言不改；两页写本页的偏好、没有本页的分组；迭代页的侧栏开关与快捷操作各宽度都给；变异 `t9-mut-no-gate`、`t9-mut-not-observer`、`t10-mut-own-grouping`、`t10-mut-toggle-hidden`、`t10-mut-gate-outside-observer`（两页）。修复轮 F3：`usePageDisplay` 按地址读本页的偏好（`f3-mut-router-read`） | T9、T10、F3 |
| A9 | A19：两个 store 的 `pageTotal(projectID)`（范围是 `project`、项目是本页的），分组时照样显示 nerve 的总数、别的项目的列表在时不显示；`project-page-total.test.ts` 2 例、`issues/header.test.tsx` +1 例；变异 `t11-mut-empty-group`、`t11-mut-unscoped`（两个 store 各一次） | T11 |
| A10 | A20、A21、焦点：删 `project_cycles.status.days_left`（en、zh-CN）与 `CYCLE_STATUS[].i18n_label`（类型一行与四项，`t11-type-no-label` tsc exit 2）；替身关上即清 `opened`（`fake-planning-page.test.tsx` 2 例，`t11-mut-opened-kept`）；搜索框 `data-autofocus={isMobile ? undefined : true}`（`t11-mut-no-focus` 与两个探针：手机上也聚焦、写成 `{!isMobile}`） | T11 |
| A11 | X1：视图列表页、列表与行的权限按地址的项目判断（地址经 props，必为字符串），七个视图文件加入 `project-address-route`；视图的新建、修改、删除之后的跟进经 `followInSession`；新测试 `R/views/(list)/page.test.tsx` 6 例、`views/follow-ups.test.tsx` 12 例；变异 `t12-mut-router-fallback`（三处，同时让守卫失败）、`t12-mut-follow-raw`（两处） | T12 |
| A12 | 页面故事：`g19-work-item-side-page.spec.ts` 三条故事在三种引擎通过；T13 本机 chromium 4.0 秒，e2e 变异 5 个各只让对应的故事失败；修复轮 F1 换了移动端一条的锚之后 chromium 3.8 秒，三种引擎里 chromium 4.3、firefox 6.9、webkit 6.0 秒，`--repeat-each=3` 无 flaky（各故事取最大合 4.6 秒） | T13、F1 |
| A13 | 守卫与文档：`project-address-route`（各 Task 的新件与改写的件，各带样例）、swr 限制（新加 `use-planning-search.ts`、`work-item/cycles-menu.tsx`、`modules-menu.tsx`）、告警上限 44 不变（`t14-oxlint-touched`：改过与新建的 83 个文件 0 告警）、knip、关键词（phase `M6/P7`，78 条规则、7 个例外）、i18n（新键 11 个：七个菜单的空文案、两个命令与两个占位；删 `days_left`）；P7b 的回填与余量的重算；交接回执（M1、M2 §3、M3 §1/§3/§4、M4 M6-02）；差异清单、前端改动清单、设计 §8/§12/§14、M8 交接（M6-M8-01、-07、-08，新 M6-M8-15）；re-77 的记录与本记录 | T1、T14、F4 |

## 3. 局部裁定

台账中 97 条 `Ruling:`（T1–T14 的 90 条，含 T3 的主控裁定一条；修复轮 F 4 条；主控 3 条：起草 spec 之前的 H1–H16 与预检之后对 re-76 F1、F14 的两条）与主控在复核时的裁定，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **回填**（T1）：P7b 的归档以 `archive.py verify` 复核，作业号、时刻与结论以公开的 jobs 接口核对；分支 CI 的作业日志经接口五次只下全过一次且随后被截，以归档里 P7b 当时下的那份为准；M6-M8-01 另记“分支 CI 是校准点的 1.355–1.357 倍”的观察；前端改动清单 M6 一行把 P7b 改为已合入（主控的清单之外的一行）。
- **拆分**（R3；T2）：`commands.ts` 删去，由 `root.ts` 组合两件（只合并两份记录的一层转手不留）；`baseProjectConditions` 属项目一级、导出给工作区一级（依赖单向）；特征测试以 `permissionsOver` 加 `loadProject` 造调用者与归档的项目（归档的 web 不在 nerve 的未归档列表里，`callerInWeb` 造不出“项目已归档”的页）；工作区一级一件不加入守卫（守卫的探针证它三个参数的 `allowPermissions` 会命中）。
- **入口规则与原因**（R2；T3）：入口规则写为“角色是 ADMIN 或 MEMBER 且原因为无”，归档与开关只在原因函数里写一次，入口因此蕴含读门的 `listed`；`usePlanningReads` 不改，同一次序在 hook 与纯函数两处写（re-77 核为正确）；菜单只在门为假时求原因；`planning-switch.ts` 是纯函数，不加入守卫；没有角色时说 `project.not_found`（主控裁定）。
- **空文案**（R4；T4）：键的类型是不导出的 `power_k.${string}`，挡住英文的字面量与别的命名空间，键是否定义由 `i18n-keys` 守卫读字面量核对；属性改名为 `emptyTextKey`（`PowerKMenuEmptyState` 收已翻译的文字，同名而契约不同正是要分开的混淆）；默认只写在 `empty-state` 一处；zh-CN 照 locale 的惯例译为“未找到…”，cycle 是“周期”；测试用真的 i18n（假的会让“直接显示键名”也过）。
- **搜索**（R5；T5）：新 hook 收会话的 `RootStore`，参数名取 `rootStore` 让守卫管得到它（守卫的内容式按变量名认 router store，这一缺口交 M8 的 [M6-M8-15](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-15)）；只在 store 不持有这一域的列表时读、以面板开着为门（`/browse` 上第一次打开读一次，之后持有）；行带它的项目，组的次序是工作项、项目、迭代、模块。
- **工作项的两个命令**（R6；T6、T7、F2）：命令的 id 取 Plane 的 `add_work_item_to_cycle`、`add_work_item_to_modules`；改迭代页自己列行（选项首项是 `null`，所持的非选项一行要禁用，构造件没有），所持的已结束、已归档的迭代在最前、打勾、禁用；“选所持的不发”与模块的增或减都在选中时以 nerve 确认过的值判断（同下拉框）；en “Change cycle”“Change modules”，zh-CN “更改周期”“更改模块”；命令经 store 的 `changeBody` 算请求体（面板的守卫包在外面，用 `changeIssue` 会绕过轮到时的检查）；“门为假时说原因”在第四处出现时合为一个组件（只在门为假时渲染，角色只在那时求）；类型测试读命令真正用的写 `useWorkItemWrites`；改模块页复用 `PowerKModulesMenu`；`mutation-commands.ts` 不加入守卫（只加了命令、不读地址）。F2：每行的 value 取 id（“无迭代”取常量 `no-cycle`），名字作 keywords，`powerKFilter` 在行给了 keywords 时只比 keywords——不取 `${name} ${id}`（子串匹配会以 id 的十六进制字符找到迭代）；模块页不改（名字在项目内唯一）。
- **A17、A18**（R7；T8）：在途的状态在两个快捷操作里各写一个，不抽共用的 hook；“同一拍”的例开两个菜单、在一个 `act` 里先后点两处；只测被拒的一支；键盘不另立例（两个菜单都渲染 `<button disabled>`，与同一拍的触发是同一个守卫）；A18 也删去只被编辑、复制两项读的 `isEditingAllowed: false`，类型分为 `Editing` 与 `ArchivedIssueMenuProps`，项目页、详情页收两者的合（re-77 记为观察：名字比用途窄）。
- **移动端页头**（R8、R9；T9、T10、F3）：共用的读写抽为 hook `usePageDisplay`，宽屏与窄屏的件都经它；共用件与 hook 各在自己的文件（两个回归测试把 `@/components/issues/filters` 整个 mock 掉）；包装件只收 layout 从路由参数给的地址；拆出的文件保留 Plane 的版权头；两个 `page.test.tsx` 各加一行 mock（主控接受）；“显示”的分组以显式的期望表比；迭代页页头的开关与快捷操作移出 `hidden md:flex`、DOM 次序不变。F3：`usePageDisplay` 以 `getIssueFilters(entityId)` 读，`usePageIssueFilters` 还有 5 个产品读者、留着（它们的同一种读交 M8，§6）。
- **遗留**（R10、R11；T11）：两个 store 各一个 `pageTotal` 与私有的 `pageList`（同一个范围比较不在一个类里写三遍），不往基类加共用件（M6-M8-08 的行数）；归档页页头加入守卫；搜索框写 `data-autofocus={isMobile ? undefined : true}`（`{!isMobile}` 会被 React 写成字符串 `"false"`，仍命中）。
- **X1**（R12；T12）：两个组件的地址经 props（必为字符串，退回 router store 在类型上到不了，行动作里的判空随之删去），`view-list-item.tsx` 因此也改、加入守卫；视图弹窗的跟进只看会话；拒绝的提示照旧是英文（视图的文案属 M7）。
- **e2e**（T13、F1）：分组断言整张表；“改模块”第二次选之前先等页上的 “Modules: Alpha”（命令面板之后页面被 `ModalCore` 设为 `aria-hidden`，查询要 `includeHidden`）；搜索结果的锚是迭代页独有的空态；组合框打开时 Escape 先关它的列表，故事改为点 Cancel 关弹窗。T13 让移动端一条的迭代页与模块页共用一个世界（Plan 同在两页），这正是 re-77 的 I1；F1 给模块页一个只在 Alpha 的工作项 Review 作锚。
- **文档**（T14）：`project-address-route` 只按变量名认 router store 的缺口交 M8（新 M6-M8-15，不交 P8：规则是 M3 立下、几个 M 共用的守卫）；plan 的勾选框不逐个勾、以状态行列出提交；spec A7 注明键盘没有单列的例；M3 §3 的回执不说 P7b 的 G4 (page) 在 T11 之后通过（写为“T15 的门禁里照跑”）。
- 其余见台账。

## 4. 实施记录

变异数为能编译的产品变异（e2e 的为能编译、能构建的），守卫（关键词、swr、i18n）的变异另记；探针是 plan 之外为钉住裁定而做的变异；例数为网页包（`web`）的文件数/用例数之差。

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `bee18397`、`78fa4dbc`、`3774b98e` | spec 与 plan（15 个 Task；R1–R16、A1–A14）；预检 re-76 的修正（F1–F4 Important、F5–F16 Minor）与 re-76 的记录；T6 时更正 F8 的事实 |
| T1 P7b 的回填与余量 | `b5e1c615` | P7b 阶段记录的抬头与 §7、P7b spec 的 A15、P7b plan、设计抬头与 §14、前端改动清单、M6-M8-01 新一条、M6-M8-13 一句；原件逐项核对（收据 `t1-*`）；P7c 的余量 14.6–19.3 秒；只改文档 6 个文件 |
| T2 拆导航的命令 | `87dbb973` | `workspace-commands.ts` 259、`project-commands.ts` 252、`root.ts` 51；特征测试拆前拆后同名全过，power-k 既有 12 个文件 122 例同名同数；+1/6；变异 3 个、守卫的 2 个 |
| T3 入口规则与打开的菜单 | `3151dd9b`、`f5ef9548` | `planningEntryOn`、`planningListRefusal`，四个导航命令，两个菜单说原因；特征测试归档页四格改为不给（契约变化），两个菜单的既有例一行不改；没有角色时 `project.not_found`（主控裁定）；+0/27；变异 4 个（两个菜单各跑一次）与 `t3-mut-not-found-code`、探针 2 个 |
| T4 菜单的空文案 | `df04b1bc` | `emptyTextKey`、七个新键、`empty-state` 的默认；power-k 既有 13 个文件 136 例同名同数，没有契约变化；+1/10；变异 3 个、类型收据 2 个、i18n 守卫的探针 1 个 |
| T5 搜索的迭代、模块 | `74381080` | `use-planning-search.ts`（61 行）、`search-results.tsx` 两组、“没有结果”四组都空；`search-menu.test.tsx` 既有 12 例一行不改；+0/8；变异 4 个、探针 4 个、守卫的 4 个 |
| T6 改迭代 | `6c41721c` | 命令（`shift+c`）、页类型与占位、`work-item/cycles-menu.tsx`、两个新键；`commands.test.tsx` 表格 +1 行（既有的行不改）；+1/19；变异 3 个、探针 6 个、守卫的 2 个、类型 1 个 |
| T7 改模块 | `165420ef`、`77ac77a4` | 先纯搬移（`PowerKPlanningListRefusal`、`useWorkItemWrites`，power-k 173 例同名同状态；探针 1 个）；命令（`shift+m`）、`work-item/modules-menu.tsx`、`menus/modules.tsx` 的勾与说明、两个新键、类型测试；+1/20；变异 3 个、探针 8 个、类型 2 个、守卫的 4 个 |
| T8 A17、A18 | `03b85a6e` | 在途的状态、工厂的 `disabled`；`Editing` 与 `ArchivedIssueMenuProps`；基线 147 个文件 1619 例同名同状态，两个 `actions-host` 测试只改导入一行；+1/5；变异 3 个、探针 1 个、类型 2 个 |
| T9 共用的移动端页头 | `91272fc2` | `usePageDisplay`、`MobileHeaderFilters`、项目页的包装件、`filters.tsx` 86 → 67（纯搬移）；基线 61 例同名同状态，`preferences-controls.test.tsx` 只改挂载一行；+2/5；变异 2 个、探针 3 个、守卫的 3 个 |
| T10 两页的移动端页头与顶行（检查点） | `d96ed7b2` | `PlanningPageMobileHeader`、两页的 layout、迭代页的顶行；基线 132 例同名同状态，两个 `page.test.tsx` 各加一行 mock（主控接受）；+1/8；变异 4 个、探针 3 个、守卫的 1 个 |
| T11 A19–A21 与焦点 | `5e5f8ee8` | 两个 store 的 `pageTotal`，删 `days_left` 与 `i18n_label`，替身的 `opened`，`data-autofocus`；基线 41 个文件 390 例同名同状态；+2/7；变异与探针共 10 个、类型 1 个、守卫的 1 个 |
| T12 四个视图文件 | `fb5c345d` | 地址的读、七个文件加入守卫、`followInSession`；视图的既有 8 个文件 63 例同名同状态；+2/18；变异 5 个（`router-fallback` 三处同时让守卫失败）、探针 1 个、守卫的 3 个 |
| T13 页面故事 | `3b5b0a78` | `g19-work-item-side-page.spec.ts`（195 行，三条故事）；e2e 变异 5 个各只让对应的故事失败，之后以干净的源重建；产品代码不动 |
| T14 文档 | `5087b09e` | 差异清单新四行与一行补句；前端改动清单的总览、两行补句与新十行；设计抬头、§8、§12、§14；M1–M4 交接的回执；M8 交接的 M6-M8-01、-07、-08 与新 M6-M8-15；spec 的状态、A1–A13 的勾选与 §5，plan 的状态；只改文档 10 个文件 |
| F2 | `98d9f985` | 改迭代页的行以 id 作 value、名字作 keywords，两个宿主共用 `powerKFilter`（re-77 M1）；power-k 与 top-nav 17 个文件 198 例同名同状态；+1/9；变异 1 个、探针 4 个 |
| F3 | `10ca6ae6` | `usePageDisplay` 按地址读本页的偏好（re-77 M2）；页头与窄屏一行的 17 个文件 153 例同名同状态；+0/1；变异 1 个 |
| F1 | `165b252b` | 移动端一条的模块页以只在 Alpha 的 Review 为锚（re-77 I1），只改故事（195 → 197 行） |
| F4 记录 | `396bd5f7` | re-77 的记录、本记录、spec §5 的修复轮与交出的两项、spec 与 plan 的状态、设计抬头与 §14、前端改动清单（总览与两行）、M8 交接的 M6-M8-03 与 M6-M8-15 |
| F5 | `6b79fee4` | `PowerKModalCommandItem` 以 keywords 作 cmdk 的项的 key，改名之后搜索认新名字（限定复审 m1）；power-k 与 top-nav 18 个文件 207 例同名同状态；+1/1；变异 1 个 |

网页包从基线的 466 个文件 5309 例（P7b 的终点）到 480 个文件 5453 例（F5 之后的全量收据 `f5-web-count`）；常量包 2 个文件 22 例不变（T11 删了字段，例数不变），其余包同基线；e2e 由 477 条到 480 条（G19 (page) 的工作项一侧三条）。

## 5. 门禁与 CI

- **检查点**（T10 之后，`d96ed7b2`）：
  - `t10-gate`（2026-11-15T17:16:38Z–17:36:01Z，19 分 23 秒，串行、一次一个容器套件）：`make gen-check`、`make lint-go`、`make test`（本机 910 秒通过，没有 M6-M8-05 的签名）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 477 条中 476 过、1 条失败（3.7 分钟）：F11 (page) [s3] 以 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 的签名（`net::ERR_ABORTED GET /api/v0/assets/<id>/content`；同一运行里 F11 [local] 通过），`e2e/test-results` 先存为 `t10-gate-test-results`，照该条只重跑这一个文件一次 7 条通过（`t10-gate-e2e-f11-rerun`，30.3 秒）。本机 chromium 的 `stories/planning/` 61 条合计 264.0 秒（9 个工作进程，只作对照）。三种引擎的页面配置 `t10-m6-pages` 126 例通过（2.3 分钟，flaky 0）。
  - 分支 CI 43742488275 attempt 1 全绿、没有重跑：server (1) 132898482119、web 132898482301、server (0) 132898482401、e2e 132902462743（e2e 作业 17:55:04Z–18:14:10Z，19 分 06 秒 = 1146 秒）。这时还没有本段的页面故事，比 P7b 合入后的 main CI 1422 秒少 276 秒（约 0.81 倍），作 T15 估算的校准点（T14 记入 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)）。检查点推送之后的提交留在本地，直到 T15。
- **三种引擎的页面配置**（`playwright.m6-pages.config.ts`，新文件照它的 `testMatch` 自然进入，配置不改）：
  - T13：本文件三种引擎 9 例（chromium 4.6、firefox 7.0、webkit 6.4 秒），chromium 单独 4.0 秒（命令面板 2.2、移动端 1.2、焦点 0.6），`--repeat-each=3` 9 例无 flaky（各故事取最大 2.3 + 1.3 + 1.0 = 4.6 秒）。
  - F1：本文件 chromium 单独 3.8 秒（2.0、1.3、0.6），三种引擎 9 例（chromium 4.3、firefox 6.9、webkit 6.0 秒），`--repeat-each=3` 9 例无 flaky、没有重试（各故事取最大 2.2 + 1.4 + 1.0 = 4.6 秒）。
  - 预算 8 秒（R14），没有间歇失败、没有重跑。
- **T15 的第一次门禁**（`t15-gate`，`5087b09e`，2026-11-15T19:02:34Z 起，串行，与整分支审查 re-77 并行）：`make gen-check`、`make lint-go`、`make test`（本机 897 秒通过，没有 M6-M8-05 的签名）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 480 条中 479 过、1 条失败（3.7 分钟）：S1 server-ready（`e2e/stories/smoke/s1-server-ready.spec.ts:12`）的 `nerve migrate status` 连数据库时与之后的 `pg_dump` 都收到 “FATAL: sorry, too many clients already (SQLSTATE 53300)”，即 [M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03) 的签名，`e2e/test-results` 存为 `t15-gate-test-results`；照该条只重跑这一个文件一次 6 条通过（`t15-gate-e2e-s1-rerun`，16.9 秒）。三种引擎的页面配置 `t15-gate-m6-pages` 135 例通过（126 加本段的 9 例，2.2 分钟，flaky 0）。全部收据的 `inputs_unchanged` 为真。修复轮在这次门禁之后开始。
- **修复轮各自的检查**：F2、F3、F5 各跑焦点测试、改过的文件 oxlint（0 告警，上限 44 不变）、关键词、`make lint-web`、`make knip`、`make test-web`；F1 跑 e2e 的检查、上面的三组 e2e 与 `make lint-web`、`make knip`、关键词（台账的 `Fix F<N>:` 行）。
- **最终候选**：（待主控填写：最终候选的七项门禁、三种引擎的页面配置，推送与分支 CI。）
- **CI 的 e2e 时长估算**（推送前，R14、spec §2.12；台账“T15 估算（主控）”，在 T13 之后算，修复轮的 F1 只换锚，X 仍是 4.6 秒）：
  - **X**：本段新文件本机 chromium 4.6 秒（各故事取较慢的一次）。
  - **估算**：基线 1422 秒（P7b 合入后的 main CI 43706438924）加 4.6 秒 × 1.67–2.25、两个工作进程摊开，1425.84–1427.17 秒（23 分 45.8 秒至 23 分 47.2 秒）。
  - **上界**：S + X × f，S = 1730.48–1740.13 秒（T1 的算法，大于 1670、1422、1146 三次实测），1747.04–1758.99 秒（29 分 07.0 秒至 29 分 19.0 秒）。
  - **校准点**：T10 检查点的 1146 秒加同一增量，1149.84–1151.17 秒（19 分 09.8 秒至 19 分 11.2 秒）。
  - **慢的 runner**（观察，不是停下的条件）：× 1.31 约 1870 秒（31 分 09.6 秒），× 1.357 约 1937 秒（32 分 16.7 秒）。
  - **余量**：用去 4.6 秒，剩 10.0–14.7 秒（预算用满 8 秒时 6.6–11.3 秒）。
  - 估算与上界都低于 30 分钟的报告线，可以推送；实测超过 30 分钟仍先停下报告负责人。实际的 e2e 作业时长见 §7。
- **测试与 e2e 的教训**（交 P8）：
  - **应用内导航之后以目标页独有的内容为锚**（F1）：`moveWithinApp` 什么都不等，前一页还挂着时它的行可能仍在 DOM 里（看板留着行的快捷操作按钮 “Options: <名字>”）；两页共用一个世界时，给目标页一个只在它那里的工作项作锚（同 G15）。
  - **cmdk 以 value 认选中的行**（F2）：行的 value 要唯一（取 id），名字作 keywords；面板的 `powerKFilter` 在行给了 keywords 时只比它们。cmdk 又只在 value 变了时重新登记 keywords（F5）：`PowerKModalCommandItem` 以 keywords 作项的 key，改名之后项重新挂上。
  - **命令面板之后页面是 `aria-hidden`**（T13）：`ModalCore` 把面板背后的页设为 `aria-hidden`，在面板开着时查页上的确认要 `includeHidden`；“改模块”的第二次选先等第一次的确认，免得与它的回答赛跑。
  - **组合框打开时 Escape 先关它的列表**（T13）：Headless UI 的标准行为；关弹窗点 Cancel。
  - **门禁的 e2e 失败时先留下 `test-results`**（T10、T15 照做）：签名因此都核实了（M6-M8-07、M6-M8-03）。
- **最终 HEAD 的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-77](../../../reviews/re-77-M6P7c整体审核.md)，`fable`，范围 `d052213f..5087b09e`，19 个提交、103 个文件，与 `5087b09e` 的第一次最终门禁并行）逐项核对了：入口规则（`planningListRefusal` 恰在 `listed` 为真时为无，同一次序两处写、测试钉住）；搜索（项目范围、入口规则、同一份 SWR 键、“没有结果”的规则、`/browse` 上面板打开即读、搜索词不去空白）；工作项的两个命令（选项、所持的已结束迭代禁用、已归档的模块只可移出、写在守卫之内轮到时算、`null` 经宿主交给命令、增或减以确认过的集合判断，开关在轮到时不再核对、由 nerve 拒绝）；A17、A18（类型拆分不改别的菜单）；移动端页头（`observer` 包装件、路由的参数、迭代页页头的 DOM 次序，T10 的两行 mock）；A19–A21、打开即聚焦、X1；守卫与回归边界；e2e 的故事 1、3 与文档。

结论 Ready to merge after fixes：没有 Critical；一处 Important（I1 移动端的故事在应用内导航之后的锚不唯一，Plan 两页都有，Sprint 的看板留着 “Options: Plan”），两处 Minor（M1 改迭代以名字作 cmdk 的 value，同名的迭代或名为 No cycle 的迭代键盘选错；M2 `usePageDisplay` 对项目页经 router store 读偏好、门按地址，与基线相同）与五条观察，没有须负责人决定的事。主控三处都接受，以修复轮 F 处理：F1（`165b252b`）给模块页一个只在 Alpha 的工作项 Review 作锚；F2（`98d9f985`）改迭代页的行以 id 作 value、名字作 keywords，两个宿主共用 `powerKFilter`；F3（`10ca6ae6`）`usePageDisplay` 按地址读；F4 写记录（`396bd5f7`）。

修复轮另发现两处同一类的问题，主控裁定交出（re-77 §3）：
- **(a)** “打开迭代”（`PK/menus/cycles.tsx`）、视图、工作区三个菜单也以名字作 cmdk 的 value（名字不唯一，只是导航）——交 P8，照 F2 的 keywords 写法、给 `PowerKMenuBuilder` 加 `getKeywords`（[spec](../specs/P7c-palette-mobile.md) §5）。
- **(b)** `usePageIssueFilters` 余下的 5 个读者（四种布局的根与日历的块）在项目页上经 router store 读本页的偏好（早于本段）——并入 M8 的 [M6-M8-15](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-15)，以 F3 的 `usePageDisplay` 为先例。

修复轮的限定复审（[re-77](../../../reviews/re-77-M6P7c整体审核.md) §4，`fable`，只读，范围 `5087b09e..165b252b`）：结论 Ready to merge，没有 Critical、没有 Important。它核对了：F2 解决 M1（各行的 value 唯一；没给 keywords 的行行为不变——cmdk 交来 `[]`，`powerKFilter` 退回比 value；给 keywords 的只有 `command-item.tsx` 与 `cycles-menu.tsx`；`powerKFilter` 的签名合 cmdk 的 `CommandFilter`；id 永远搜不到）；F3 解决 M2（三种 store 按 `entityId` 读，调用者传地址的 id，`usePageIssueFilters` 余下五个产品读者）；F1 解决 I1；回归边界与负责人的规则守住；它自己跑的焦点 vitest 6 个文件 63 例通过。一处 Minor m1，F2 带来的回退：cmdk 1.1.1 只在项的 value 变了时重新登记 keywords，改迭代页开着时迭代被改名，标签变了、搜索仍只认旧名字（选中仍发对的 id）。主控定为修复 F5（`6b79fee4`）：`PowerKModalCommandItem` 以 keywords 作 `Command.Item` 的 key，keywords 变了项重新挂上（修在把 keywords 交给 cmdk 的唯一一处，凡给 keywords 的行都管住，不在改迭代页各写组合的 key，实现者的裁定）；RED `f5-red`、GREEN `f5-green`，变异 `f5-mut-id-key`（去掉这个 key）失败、`restored=yes`，`f5-after-pk` 19 个文件 208 例，oxlint 0 告警（上限 44），关键词、`make lint-web`、`make knip`、`make test-web` 通过（台账 `Fix F5:` 行）。

## 7. 集成、归档与接续

由 P8 的 T1 按归档里的台账、`manifest.json`、`cleanup.log` 与 CI 证据回填：最终 HEAD 的分支 CI、合入与 main CI（作业号、时刻、attempt 与结论以公开的 jobs 接口核对，e2e 的条数与逐条的时长以作业日志核对）、归档与清理；spec 的 A14 随之勾选。

**接续**：P8（[spec](../specs/P7c-palette-mobile.md) §5）——归档页的迭代、模块词表（P7b R10）；[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)；新控件（菜单、移动端页头、顶行）的焦点轮廓随最终矩阵核对；M1 交接的死成员收敛；i18n 的两处英文字面量（快捷键说明的 “No shortcuts found for”、归档页行菜单的 “Restore”）；命令面板其余菜单以名字作 cmdk 的 value（修复轮发现 (a)）；访客在项目页上看到项目设置与视图的命令（最终矩阵的观察）；它的 T1 回填本记录的 §7 与 spec 的 A14，以本段的实测重算 e2e 的余量。之后是 M6 的收尾。M8 另接 [M6-M8-15](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-15)（守卫按变量名认 router store；列表的正文经 router store 读项目页的偏好）。
