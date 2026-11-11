# M6/P5a 阶段评审记录

> 已完成（2026-11-11，noff `0b8efcd7`，main CI 42995581409 首次通过），整分支审查修复后 Ready to merge（[re-65](../../../reviews/re-65-M6P5a整体审核.md)）；分支 CI、合入、main CI 与归档清理见 §7（由 P5b 的 T1 回填）。基线 main `a2b50624fd872c183628a55d04b7961f9a1dbc60`（P4 的 noff `c3ba5a40` 与其 main CI 失败的根因修复 noff）。

## 1. 评审方式与范围

P5a 交付设计 §12 P5a 一行：网页的迭代改调 nerve 的 `/api/v0`——数据层（service、按会话分代的 store、筛选 store）、列表页（进行中卡片、即将开始、已结束）、预览、归档目录、详情页与进度侧栏、新建/修改、快捷操作、删除、归档与恢复、转移，以及迭代文件的地址、复制、收藏入口与项目级弹窗宿主。迭代页的工作项列表与进行中卡片的优先级页签在 P5b。

- **调研**：写 spec 前，两个只读调研代理对照 main 写出现状笔记（网页的迭代领域与约定；Plane 的页面行为与 nerve 的接口契约），随台账归档。
- **界定**：[spec](../specs/P5a-web-cycles.md) 与 [plan](../plans/P5a-web-cycles.md) 界定 16 个 Task；P5 因超出一段的上限拆为 P5a/P5b。
- **预检**：实施前一次 fresh 只读预检（[re-64](../../../reviews/re-64-M6P5a预检.md)），结论为修正后可实施；三处 High（告警上限随清掉告警的提交下调、新工具函数另起模块与名字、优先级页签移到 P5b）与七处 Medium 在实施前改入 spec 与 plan。
- **实施**：逐 Task 由实现代理完成，主控复核并记台账；T1、T15 的文档由主控完成。主控复核另开三个修复轮：F1（只看自己的访客的写入口）、F2（两项交接要求的浏览器核对）、F3（整审的修复）。
- **门禁与审查**：T9 之后一次检查点（七项门禁与分支 CI）；T16 一次 fresh 只读整分支审查（[re-65](../../../reviews/re-65-M6P5a整体审核.md)），修复轮 F3 另经一次限定复审；最终候选的七项门禁与三种引擎的页面故事。

## 2. 验收状态

A1–A7 满足（[spec §4](../specs/P5a-web-cycles.md#4-验收标准)已勾选），A8 待合入后由 P5b 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 数据层：service、按会话分代的 store、包装层按权限与开关取数、筛选 store 没有反应 | T2、T3、T4、T12 |
| A2 | 进度口径：`cycleCounts`、`progressPercent`、冻结只在已结束时读、燃尽与分布 | T4、T5、T8、T10 |
| A3 | G1–G3、G8 的页面：建、改、删、归档、恢复，被拒时留着，权限与状态条件 | T6、T7、T9、F1、F3、T13、F2 |
| A4 | G5–G7 的页面：进行中卡片、侧栏、冻结、转移 | T8、T10、T11、T14 |
| A5 | G19：迟到的回答、key、宿主、键盘与名字、复制失败 | T6、T7、T8、T9、F3、T14 |
| A6 | 旧代码删除与守卫 | T7–T9、T12、F3 |
| A7 | 文档与交接回执 | T1、T15、T16 |

## 3. 局部裁定

台账中约 160 条 `Ruling:`，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **页面级宿主**（T7，偏离 plan）：迭代的修改、归档、删除弹窗与“页面的生命期”由 `CycleActionsHost` 持有，挂在列表、详情、归档目录三个布局上，快捷操作经 `useCyclePage()` 打开弹窗。plan 原写弹窗留在行里；探针证实行在自己的修改确认时先卸载，以行为准会丢掉成功的提示与跳转，只看会话又会让后退之后迟到的回答在新页面提示，以页面为准两者都对（先例 M4 的列表命令宿主）。
- **写的门**（F1、F3）：`canWriteCycles`、`canDeleteCycle` 在 `UserPermissionStore` 一处，照 `access/domain/decide.go` 与 `planning/app/lock.go`：只看自己的访客（按项目角色，含工作区管理员兼访客）与迭代关闭的项目都没有写入口；删除给管理员与按项目角色能写的建立者。新建弹窗的项目下拉也用它（工作项的 `projectsWithCreatePermissions` 不变）。
- **详情页的取数**（T9）：未归档的迭代以包装层同键取 `PROJECT_CYCLES`，已归档的取 `PROJECT_ARCHIVED_CYCLES`（R11），成员与计数从那一行读。
- **侧栏的状态页签**（T8）：只列计入进度的四组，每组占（总数 − 已取消）的份额，已取消另起一行说明（同进行中卡片）；记入差异清单。
- **分布的共用**（T10）：进行中卡片与侧栏共用 `distribution-stats.tsx`；`progress-stats/*` 只剩模块的读者，交 P6。
- **改日期**（T8）：两天都选了才发，与现有相同不发，在途时选择器停用，nerve 确认后提示，被拒回到 nerve 的值；删去 `date-check` 与时区换算（R6）。
- **删除与进度**（T3）：`details` 持 `Cycle | null`，删除后写成 `null`；进度的值留在键下，但只给 `getCycleById` 能给的迭代，所以读不到（spec §2.2 已随改）。
- **交接的关闭时点**（T15）：spec A7 原写 M1-P2、M1-P3 在本段关闭，与设计 §11“保持 open 至收尾关闭”冲突，以设计为准（M1-P2 还有模块的进度）。
- **页面故事**：G2 按上海此刻的小时选浏览器时区，只在上海午夜前后几秒内会以明确的消息失败；firefox 在新页面于上一页之后约半秒内加载时报字体“预载未用”的告警，故事改为应用内跳转。两者记入 spec §5 的 M8（观察）。
- 其余见台账。

## 4. 实施记录

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `3345c975`、`c31def71` | 16 个 Task；预检 re-64 的修正 |
| T1 回填与阶段 | `caaced4f` | P4 阶段记录 §7、spec A8、设计 §14；P4 两个归档复核；phase `M6/P5` |
| T2 service 与读 | `b4911394` | `cycles.service.ts`、`RootStore.planning` 的读、包装层按权限与开关取数；9 个变异失败 |
| T3 写 | `08867c4f` | `cycle-changes.ts`：一个接一个、写进每一份、行保留计数；10 个变异失败 |
| T4 筛选与工具函数 | `a21686c4` | `cycle_filter.store.ts`（没有反应）、`@nerve/utils` 的 `planning.ts`；5 个变异失败 |
| T5 燃尽与分布 | `523c3d92` | `BurndownChart`、`distribution-rows.ts`；8 个变异失败 |
| T6 宿主与表单 | `5cdd61c2` | `ProjectLevelModals` 以项目为 key、卸载时关上开关；新建/修改弹窗只在确认后关闭；14 个变异失败 |
| T7 快捷操作 | `11ef5b52` | 删除、归档、恢复与页面级宿主；16 个变异失败；告警上限 58 → 57 |
| T8 详情页与侧栏 | `a415c86e` | 以迭代为 key、进度与冻结、改日期；18 个变异失败；上限 57 → 55 |
| T9 列表、预览、归档目录 | `b6a0b7df` | 按路由参数与权限取数、失败可重试；17 个变异失败；上限 55 → 53；检查点 |
| 检查点的契约变化 | `b36fe3b1` | M4 冒烟故事 S2 移出迭代的两条（钉的是旧地址 404），覆盖交 G3/G8 |
| T10 进行中卡片 | `51271de5` | 新 store、`/progress` 的同一份进度、统计卡只留负责人与标签；8 组变异失败 |
| T11 转移 | `3104995c` | 转移弹窗与两个入口，按实时未完成数与权限；17 个变异失败 |
| F1 只看自己的访客 | `8472478c` | `canWriteCycles`、`canDeleteCycle`；7 个变异失败 |
| T12 读者与删除 | `f4970550`、`aabd9685`、`7143859c` | 命令面板与收藏改读新 store，删除旧代码，`old-cycle-urls`，死成员；7 个变异失败 |
| T13 页面故事一 | `1523da86` | G1、G2、G3、G8 与三种引擎的页面配置 |
| T14 页面故事二 | `5524a44d` | G5、G6、G7、G19 |
| F2 浏览器核对 | `9831341c` | 右键菜单（G3）与名字的输入法（G1） |
| T15 文档 | `8f2b1248` | 六份交接回执、设计 §8/§12、总体设计 §7.7、两份清单、spec §5 |
| F3 整审修复 | `5b5da6ea` | 预览侧栏以迭代为 key、迭代关闭时没有写入口、恢复只发一次、`old-cycle-urls` 补无尾段；7 个变异失败 |
| T16 记录 | `cbfb0d50` | re-65、本记录、spec §4 与 §2.2/R8/§5、plan 与设计 §14 的状态、两份清单的一句、M8 交接 M6-M8-03 补记与 M6-M8-05 |
| G18 的根因修正 | 本记录所在提交 | 取“之前的读”前等已结束的迭代冻结（见 §5） |

## 5. 门禁与 CI

- **检查点**（T9 之后）：
  - `t9-gate`（`b6a0b7df`）：`make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web` 六项通过；`make build && make e2e` 434 过、2 败，是 M4 冒烟故事 S2 中迭代的两条：钉的是迭代页面请求旧地址得 404、归档目录永远加载，正是本段改掉的。按契约变化把这两条移出（`b36fe3b1`），覆盖交 G3/G8 的页面故事。
  - `t9-gate2`（`b36fe3b1`，差异只有一个 e2e 故事文件与 plan，Go 与接口未变，前三项沿用 `t9-gate` 的收据）：网页四项通过，e2e 434 条全过。
  - 分支 CI 42911333596（`b36fe3b1`）attempt 1 成功：web 130162458387、server (0) 130162458757、server (1) 130162458787、e2e 130169949956（约 19 分 50 秒，期限 35 分钟）。
- **页面故事的三种引擎**：T13、T14、F2 各在 chromium、firefox、webkit 上跑页面配置（`e2e/playwright.m6-pages.config.ts`），新故事另以 `--repeat-each=3` 各跑一次，没有失败或 flaky；最终候选 `t16-three-engines`（`5b5da6ea`）36 条全过（12 条 × 3 种引擎，60 秒）。
- **最终候选**（`5b5da6ea`）：
  - `t16-gate` 在 `8f2b1248` 上运行到 `make test` 时因整审要修复而停止（`gen-check`、`lint-go` 已过），testcontainers 的容器由 ryuk 回收。
  - `t16-gate2`：`gen-check`、`lint-go` 通过；`make test` 63 个包通过，`asset/adapter/s3` 一包失败：testcontainers 查询容器状态时 Docker API 超时（`context deadline exceeded`），容器已启动。分支没有任何 `server/`、`api/` 的改动，判为本机环境失败，按负责人的规则整段重跑一次。
  - `t16-gate3`：`gen-check`、`lint-go` 通过；`make test` 62 个包通过，`platform/postgres` 与 `migrations` 两包失败，签名相同。
  - **根因**（本机环境）：Docker Desktop 的日志（`com.docker.backend.log`）里，失败的容器开端口转发时 “listen tcp 0.0.0.0:56932（56934）: bind: address already in use”，此后 testcontainers 等映射端口约 60 秒到期限。Docker VM 给容器随机分配主机端口的范围是 55000–65535，macOS 自己的临时端口是 49152–65535，两者重叠，分到的端口可能已被本机的连接或监听占用。这条警告此前只在 11-10 的两个时刻出现，与本次的两次失败一一对应；P3b 的 t14-gate1 与 P2 的一次门禁是同一签名。
  - **负责人的决定**（2026-11-11）：本段的服务端测试以分支 CI（Linux、准确的 HEAD、两路）为准，本机两次失败与根因记在这里；端口范围重叠交 M8（[M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05)），不改本机设置，不改测试代码。
  - 网页四项：`make lint-web`、`make knip`、`make test-web`（turbo 缓存重放，输入与修复轮 F3 的 `f3-test-web` 相同）通过；`make build && make e2e` 445 过、1 败：C12 的 LAN HTTP 一例页面上 `GET …/states` 500，该测试 nerve 日志为 `FATAL: sorry, too many clients already`（53300），即 e2e 共用的 Postgres 连接用尽，已登记为 [M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03)（M6-M8-01 要求 C12 再失败时按新问题定位，查实是这一条）；照既有做法同一输入只重跑这一项（`t16-gate3-e2e-c12-rerun`），29 条全过。
  - 各收据的输入前后一致，且等于提交的内容。之后只有本记录、re-65、spec、plan、设计 §14 与两份清单的文档修改，以文档链接与关键词检查覆盖。
- **分支 CI 第一次**（`cbfb0d50`，42984392888）：server (1) 130402002284、server (0) 130402002429、web 130402002583 成功（本段的服务端测试以此为准）；e2e 130408585582 失败 445/1：G18 (API)（`g18-cycles-switch.spec.ts:118`）开关打开后再读，已结束的 Sprint、Old 的列表行带上了冻结的计数，与开关之前的读不等。
  - **根因**（测试的缺陷，P4 起潜伏）：G18 在 P1 写成，比较开关前后对迭代的整份读；P4 加入冻结任务（测试配置每 2 秒），已结束的迭代在其后的第一次运行时被写一次冻结。取“之前的读”之后若恰逢一次冻结，后面的读就不等。产品行为正确（冻结只写一次、不覆盖），P5a 没有改服务端。
  - **证据与修正**：在“之前的读”之后插入 3 秒的等待，本机必然复现（`t16ci-red-g18-freeze-between-reads`）；修正为取“之前的读”前以既有的 `frozenSnapshot` 等两个已结束的迭代冻结，同样插入 3 秒也通过（`t16ci-green-g18-freeze-between-reads`），`--repeat-each=5` 全过，`make lint-web` 通过。G3 的比较只看名字与最后修改的时刻（冻结不写），不受影响。
  - 按负责人 2026-11-08 的规则不重跑，修正后以新提交重跑分支 CI，结果见 §7。
- **最终候选的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-65](../../../reviews/re-65-M6P5a整体审核.md)）逐项核对了：

- 数据层与写入（副本、迟到的回答、行的计数、轮到时找迭代、拒绝）；
- 写的门与服务端的规则、七个写入口；
- 宿主与 key、只在确认后关闭与跟进、燃尽与分布、日期；
- 旧代码的删除与守卫、告警上限与文件大小；
- 测试的 RED 与变异、页面故事的时序；
- 文档与 Plane 源码的引用。

结论 Ready after fixes：一处 Important（预览侧栏没有以迭代为 key），七处 Minor。修复轮 F3 修了 I1 与 M3、M4、M5，M1、M2 改 spec，M6 交 P6，M7 并入 P7；F3 的限定复审结论为四项都在根因上关闭、没有回归，三条观察不改（re-65 §4）。

## 7. 集成、归档与接续

由 P5b 的 T1 按归档里的台账、CI 收据与 `cleanup.log` 回填。

- **最终候选的分支 CI**：
  - 第一次，`cbfb0d50322f80aab3f6a262376fb2ac000cf918` 的 42984392888 attempt 1：server (1) 130402002284、server (0) 130402002429、web 130402002583 成功（本段的服务端测试以此为准，§5）；e2e 130408585582 失败 445/1（G18，根因与修正见 §5）。
  - 修正后，`8175e76e258d50f3235a7e77b7c260eb5f9ec8fa` 的 42990196723 attempt 1 成功（web 130420905071、server (1) 130420905427、server (0) 130420905573、e2e 130427802649，约 21 分钟）。
- **合入与 main CI**：noff `0b8efcd701edf7e73a37d5bb82ad176a7795b910`，双父 `a2b50624`、`8175e76e`，树与候选相同（`59fdf4e3`）。第一次推送 SSH 报 “Could not read from remote repository”（网络），再推一次成功。main CI 42995581409 attempt 1 成功（server (1) 130438746995、web 130438746998、server (0) 130438747043、e2e 130445895361）。
- **归档**（主目录，P5b 开工时以 `archive.py verify` 复核，收据 `t1-p5a-archive-verify`）：`.superpowers/sdd/M6-P5a-integration-20261111`：1836 个文件、609,100,902 字节、未知 0、清单 sha256 `b9704d297a1c36b80137885bceb6bc40c2416d66501fcf707b82fc740fc5d9ea`（台账、调研笔记、收据、变异、CI 原件、scratchpad 的脚本与草稿）。
- **清理**：`cleanup.log` 记录分支已含于 origin/main，worktree `m6-p5a`、本地与远端分支 `codex/m6-p5a` 已删除，主目录停在 `0b8efcd7`。

**接续**：P5b（迭代页的工作项列表，[P5b spec](../specs/P5b-cycle-list.md)）。
