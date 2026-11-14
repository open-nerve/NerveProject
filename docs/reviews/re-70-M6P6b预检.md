# M6 / P6b 预检

> 2026-11-14：预检通过（修正后），可以按 P6b 的 T1 开始实施。
> - **目标**：[P6b spec](../v0/M6-cycles-modules/specs/P6b-module-list.md)、[P6b plan](../v0/M6-cycles-modules/plans/P6b-module-list.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md) 与[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)。
> - **基线**：main `06b72d4b65bcba911344a4ed3c358f164a5fdeeb`（P6a 的 noff，main CI 43469787621 attempt 2 通过）；审查对象为分支 `codex/m6-p6b` 的 `ca4e62ac`。
> - **本轮**：只有文档，没有产品修改或测试运行。

## 1. 范围与方法

1. **调研**：两个只读调研代理在 main `06b72d4b` 上写出现状笔记（台账 `research/web.md`、`research/plane-contract.md`，随阶段归档）：
   - 网页：P5b 怎样接上迭代页的列表（每一处 `EIssuesStoreType.CYCLE` 的分支与联合、偏好的基类、布局根、页头、移出迭代、写后重读进度、侧栏的行切换）、模块一侧已有的件、工作项根 store 的职责与拆法、交给 P6b 的交接各项、e2e 的夹具与时长、守卫的现值；
   - Plane 与契约：Plane 的模块页列表（布局根、页头、MODULE 的 store 与筛选、新建、移出、修改、空态、只读、侧栏的行切换、“添加已有”），nerve 的模块页偏好、列表的 `module_id`、工作项的 `module_ids` 整体替换，两者的差异与交接的验收语。
2. **写 spec 与 plan**：主控先裁定四件事——工作项根 store 取“调用方改经写入单元”的拆法（选项 B，不留转发）、P5b 的迭代页列表件中只差领域的改为两域共用、第 8、9 条用于模块页、迭代页与项目页、按别的模块筛选只登记差异——再由起草代理写出 P6b 的 spec（R1–R16、A1–A8）与 16 个 Task 的 plan（`ca4e62ac`）。
3. **审查**：一次 fresh 只读审查（`fable`）对照 HEAD 的代码与只读的 `plane/` 逐条核查：
   - 事实与计数；
   - 按次序执行时，有没有守卫在某个 Task 内变红而计划不修（knip、告警上限、关键词规则、i18n、store 数、`.oxlintrc.json` 的 swr 限制）；
   - R1–R16 是否站得住、有没有须负责人决定的事；
   - A1–A8 与 e2e 对 G15、G19 的模块页列表部分与各交接验收条件的覆盖，以及 e2e 作业的时长；
   - 与接口契约、总体设计 §7.7 是否一致，对 Plane 行为的描述是否准确。
4. **裁定**：主控逐条核过审查的结论并裁定，再由起草代理改入。

原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- **工作项根 store**：`web/apps/web/core/store/issue/root.store.ts` 615 行（M6-M8-08），写的一侧是每个工作项一条的写队列（`:233-276`）、确认的流水线 `write()`（`:383-509`）、六个命令与两份请求体的白名单（`:510-614`）；`enqueueIssueChanges`、`confirmCreation`、`createIssue`、`updateIssue`、`deleteIssue`、`archiveIssue`、`unarchiveIssue`、`pendingIDs` 八个名字在产品代码里有 12 个文件 18 处调用，测试约 22 个文件 84 处（审者数）。
- **列表引擎没有模块**：`EIssuesStoreType` 没有 MODULE，模块页的路由映射到 DEFAULT（`use-issue-layout-store.ts`），`use-issues.test.ts` 与 `use-issue-layout-store.test.tsx` 钉住 MODULE 被拒与映射到 DEFAULT；创建与更新的白名单不放行 `module_ids`。
- **契约**：`/me/modules/{id}/preferences` 与生成的类型已有（`ModuleIssueFilters` 有 `cycle_id`、没有 `module_id`，本页的键以 400 拒绝），网页没有调用方；工作项的 `module_ids` 是整体替换，只检查新加入的，留下与离开的不检查，已归档的模块也在集合里（`server/internal/modules/issue/app/membership.go:60-99`）。
- **模块页**：正文只有侧栏；页头没有数目与列表的控件；侧栏的分布行不可点。P5b 的迭代页列表件（本页列表、偏好的子类、列表动作与两个跟进 hook、布局根、四个迭代根、空态、移出迭代、侧栏的行切换）都只为迭代写。
- **第 8、9 条**：项目页页头无条件给出布局、筛选与显示（`core/components/issues/header.tsx:93`）；迭代页与项目页的列表根在偏好的任一次读失败时换掉正文。
- **守卫现值**：告警上限 44；`root.store.test.ts` 21 个 store；`.oxlintrc.json` 的 swr 限制不含 `issue-layouts/roots/cycle-layout-root.tsx` 与 `cycles/active-cycle/priority-issues.tsx`（两者都经 `useSessionSWR` 取数）；关键词 78 条规则、7 条例外，phase `M6/P6`。
- **e2e 的时长**：P5b 合入后的 main CI 43373039561 的 e2e 作业 19 分 14 秒；P6a 的最终分支 CI 43463416098 的 e2e 作业 21 分 38 秒；P6a 的 main CI 43469787621 attempt 1 的 e2e 作业 24 分 17 秒（C7 失败），attempt 2 的 132062963902 为 25 分 00 秒（通过）；期限 35 分钟，报告线 30 分钟。

## 3. 结论与处理

结论 **Ready after fixes**：一处 Important、六处 Minor，没有架构问题；没有守卫会在 Task 的次序里变红；T2 只在本分支上暂留、T3 删去的转发可以接受（P5a R3、P6a R2 的形状）；R1–R16 都成立。全部在实施前改入 spec 与 plan（本记录所在的提交），Task 数不变（16 个）：

| 编号 | 问题 | 处理 |
|---|---|---|
| F1（Important） | spec §2.10 的“时长”与 plan T16 Step 1 以 21 分 38 秒（分支 CI）估算，而规则自己的基线是最后一次绿的 main CI 的 e2e 作业，即 attempt 2 的 25 分 00 秒。重算：正常的 runner 约 25.5 分钟，慢的 runner（×1.31，M6-M8-01 记下的比值）约 29–33 分钟，跨过 30 分钟的报告线。这个方法也低估了 P6a：它估加 1.2–3.1 分钟，main 到 main 实际由 19 分 14 秒到 24 分 17 秒、25 分 00 秒（含 runner 的差异） | spec §2.10 改以 25 分 00 秒为基线，直说慢的 runner 上可能超过 30 分钟；T16 推送之前算三个数并写进台账：估算（原方法）、上界（新文件本机 chromium 的时长乘以 P6a 实际的增量比，约 3.6–4.1 倍）、校准点（T10 检查点的分支 CI 实测的 e2e 时长与基线之比，那时跑的故事与 main 相同）；估算或任一次实测超过 30 分钟时 T16 停下报告负责人、不推送，改 CI 的期限或作业结构须负责人同意。spec §6 的风险一行、plan 的“检查点与证据”、T1 Step 2（attempt 2 的 25 分 00 秒作为基线）随之改 |
| F2 | plan T1 Step 1 指向已删除的 P6a worktree 里的台账 | 改为以 P6a 的归档为准：`/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M6-P6a-integration-20261114/payload/worktree/.superpowers/sdd/P6a-web-modules/progress.md`，同一归档的 `cleanup.log` 与 `manifest.json`；attempt 2 的结论、归档的数字与清理已记在本段台账，T1 以它们为线索、以原件核对，归档经 `archive.py verify`（收据）；原件与台账不一致时以原件为准并停下告诉主控。删去“归档与清理还没做就停下”一句（已做） |
| F3 | spec §2.1 没有写写入单元怎样依赖根，可能出现 `root.store.ts` → 写入单元 → `root.store.ts` 的值循环导入 | spec §2.1：写入单元对根只有 `import type`，或只依赖构造时传入的窄接口；根以值导入写入单元来建立它。plan 的代码规则、T2 Step 2（以导入行核对）、Review Focus 第 1 条与 spec A1 随之写入 |
| F4 | 共用的布局根若仍导入 `core/components/cycles/peek-on-entry.tsx`，共用件就依赖迭代独有的件；按域的差别若写成 `if (domain === …)` 的分支，会随域的增加而长 | spec §2.2 的表：共用的布局根不导入 `PeekOnEntry`，迭代页把它作为子节点或插槽交给共用的布局根（或在它旁边渲染）；四种布局的根与空态的按域差别（只读的来源、行的快捷操作、空态的文案与列表状态）经 props 或一张按域的小表给出。plan T7 的 Files 与 Step 2（以导入行核对）、T8 的 Files、Review Focus 第 2 条与 spec A2 随之写入 |
| F5 | 没有写写入单元是否加入 `project-address-route` | spec §2.1、§2.9 写明不加入：它带着 M4 的 router 退路（`this.workspaceSlug`，自 `:552` 原样搬来），规则的内容模式也认不出这种读法，列进去反而误导；spec §5 的 P7 在“去掉没有范围时退回 router 工作区的一支”旁写明去掉之后再加入。plan T2 Step 2 随之写入 |
| F6 | `module_issues` 是 `issues` 的第三份副本 | 照 P5b R9 的先例保留；spec §5 的 P7 加一句：P7 加入让三页的显示选项常量不再相同的词时，`cycle_issues`、`module_issues` 改为由 `issues` 去掉本页的键推出，不让副本在理由消失之后还留着 |
| F7 | T3 的收据 `t3-root-has-no-writes` 写成“根上没有这八个名字”；`observeRead` 留在根上并调写入单元，提到其中的名字会让这个核对误报 | plan T3 Step 2 改为只查根上这八个方法的定义（给出 `rg` 的写法），`observeRead` 调写入单元的那一处不算 |

## 4. 审者核实为正确的事实（摘要）

- 工作项根 store 615 行与 spec §2.1 的各段行号；八个名字；产品代码 12 个文件 18 处，测试约 22 个文件 84 处（spec 原写的“约 20 个文件约 80 处”是约数，随修正改为这个数）。
- 钉住 MODULE 被拒与 `moduleId` 映射到 DEFAULT 的用例；两份白名单不放行 `module_ids`；`ModuleIssueFilters` 有 `cycle_id`、没有 `module_id`。
- 服务端的整体替换只检查新加入的模块，留下与离开的不检查（`membership.go:60-99`），R5 的“请求体由所持的整集合算出、保留已归档的模块”与之相符；spec 列的 i18n 键（`module_work_items` 的四个、`issue.remove.module` 的三个、缺 `label`）与现状一致。
- 告警上限 44、`root.store.test.ts` 21 个 store、`.oxlintrc.json` 的 swr 限制缺两个迭代文件。
- T1 回填的事实（P6a 的最终分支 CI 与作业号、noff 的 SHA、双父与树、main CI attempt 1 的作业号与 C7）。
- 按 Task 的次序没有守卫会变红；T2 的转发只在本分支上、由 T3 删去，与 P5a R3、P6a R2 的并存同形。

## 5. 负责人须知

开工不须负责人决定任何事。R1（工作项根 store 的拆法）、R2（迭代与模块共用列表件）、R8（按别的模块筛选只登记差异）是主控在已批准的设计之内的局部裁定，第 8、9 条按交接用于三页；R3–R7、R9–R16 都在设计与 P5b、P6a 的先例之内。

需要负责人知道的一件事：本段新增两个页面故事之后，CI 的 e2e 作业在慢的 runner 上可能超过 30 分钟的报告线（正常的 runner 约 25.5 分钟，慢的约 29–33 分钟，期限 35 分钟）。主控已预先告知负责人。T10 的检查点先实测一次（校准点），T16 推送之前再算估算与上界；估算或任一次实测超过 30 分钟时，T16 停下报告负责人、不推送。改 CI 的期限或作业结构须负责人同意（M6-M8-06 的教训）。
