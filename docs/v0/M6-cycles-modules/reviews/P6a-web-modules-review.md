# M6/P6a 阶段评审记录

> 已完成（2026-11-14，noff `06b72d4b`，main CI 43469787621 attempt 2 通过：attempt 1 的 e2e 在 M4 的 C7 间歇失败，照 M6-M8-01 只重跑 e2e），整分支审查 Ready after fixes（[re-69](../../../reviews/re-69-M6P6a整体审核.md)：没有 Critical、Important，三处 Minor），修复轮 F 处理这三处与主控留下的两项，经限定复审 Ready to merge（一处 Minor 措辞由本记录所在的提交改，re-69 §4）；最终候选的代码 `1f41c8ca`，其后只有本记录所在的提交（只改文档），即最终 HEAD `8188582e`。最终 HEAD 的分支 CI、合入、main CI 与归档清理见 §7（由 P6b 的 T1 回填）。基线 main `aadaf61a5929b57d8aa53b93e5ae04b03cc046c6`（P5b 的 noff `9675bb31` 与其 main CI 失败的根因修正 noff，main CI 43373039561 首次通过）。

## 1. 评审方式与范围

P6a 交付设计 §12 P6 一行的前一段（P6 因超出一段的上限拆为 P6a/P6b，[spec](../specs/P6a-web-modules.md) §1、R1）：模块的 service 与 store 按会话分代、只调 `/api/v0`、类型来自 OpenAPI；项目包装层按权限与开关取模块；迭代与模块共用的取数门与件（读失败、停用空态、分布、进度页签）；模块的写与删的门照 nerve 的规则；模块列表（列表与卡片、排序、筛选、搜索）与预览、归档目录、详情页的页头与侧栏（状态、负责人、成员、日期、链接、进度）、新建/修改弹窗、快捷操作、删除与归档确认框；跨领域读者（命令面板、收藏、通知页）改用新 store，旧 service、store、手写类型与 Plane 的地址删除；G9、G10、G12、G13 的页面与 G19 的模块页面。模块页的工作项列表（G15、G19 的模块页列表部分）在 P6b（R15、R16）。

- **调研**：两个只读调研代理在 main `aadaf61a` 上写出现状笔记（网页的模块现状与 P5a/P5b 的约定；Plane 的模块页面与 nerve 的模块契约），随台账归档。
- **界定**：[spec](../specs/P6a-web-modules.md)（R1–R16、A1–A8）与 [plan](../plans/P6a-web-modules.md)（16 个 Task，T10 检查点）。
- **预检**：实施前一次 fresh 只读预检（[re-68](../../../reviews/re-68-M6P6a预检.md)），结论为修正后可实施：两处 Important（筛选的成员读按地址的 slug，I1；版式与排序存在浏览器、筛选与搜索存在会话，I2）与六处 Medium 在实施前改入（`3b73b19a`）。
- **实施**：T1、T15 的文档与各 Task 由实现代理完成，主控逐 Task 复核并记台账；T7 开头的第 0 步是主控裁定的一次 refactor（§3）。
- **门禁与审查**：T10 一次检查点（七项门禁与分支 CI）；T16 一次 fresh 只读整分支审查（[re-69](../../../reviews/re-69-M6P6a整体审核.md)），修复轮 F 另经一次限定复审；最终候选的七项门禁、三种引擎的页面配置，与推送前的 CI 时长估算（spec §2.11）。

## 2. 验收状态

A1–A7 满足（[spec §4](../specs/P6a-web-modules.md#4-验收标准)已勾选），A8 待合入后由 P6b 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 数据层：service、store 的读与写、包装层的取数、取数门、会话的筛选 store、版式与排序存在浏览器、筛选的成员不读 router | T2–T5、T10、T11 |
| A2 | 进度：`progressPercent` 一个口径（行、卡片、侧栏、排序），燃尽与分布用共用件，`ProgressChart`、`TModuleCompletionChartDistribution`、`progress-stats` 与 `comparisonLine` 删除 | T4、T5、T8、T10 |
| A3 | G9、G10、G13 的页面：建、改、删、归档、恢复、链接的加改删，被拒时留着并说明，下拉没有访客，元数据 | T6–T11、T13、F2 |
| A4 | G12 的页面：行、卡片、侧栏的计数与百分比、燃尽、分布、只看自己的访客的数 | T8、T10、T14 |
| A5 | G19 的模块页面：迟到的回答、以模块为 key、键盘与名字、焦点轮廓、两处复制失败 | T8、T10、T14、F2 |
| A6 | 删除与守卫：旧代码删除、`old-module-urls`、`project-address-route` 与嵌套解构一支、`copy-hook` 三条例外删除、告警上限 53 → 44、死成员、21 个 store | T7、T9–T12、F3 |
| A7 | 文档与交接回执、P5b 的回填 | T1、T15、F6、T16 |

## 3. 局部裁定

台账中 157 条 `Ruling:`（T1–T15 149 条、修复轮 F 8 条）与主控的 4 条裁定，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **版式与排序存在浏览器，筛选与搜索存在会话**（预检 re-68 I2 的主控裁定，spec R4；T4、T10、T11）：版式与排序经 `useModuleDisplay(projectId)` 以 `@nerve/hooks` 的 `useLocalStorage` 按项目存两个键，值对照选项校验，越界回到默认 list/name；`ModuleFilterStore` 只按项目存两组筛选与两个搜索词，没有反应、不写 `localStorage`（筛选里有成员的 id），重新载入后回到空（登记差异）。负责人与成员的筛选读 `getWorkspaceMemberIds(地址的 slug)`，不读 router（re-68 I1，`project-address-route` 加嵌套解构一支）。
- **取数门与开关的谓词**（T5；T7 第 0 步，主控裁定）：`usePlanningReads` 回答“可读”与“列表可取”，迭代各处按 spec §2.3 的表改用，迭代的 41 个测试文件 504 例断言不改；谓词 `planningOn` 不依赖 store、单独成文件（包装层的回归测试在 node 环境里载入它）。T6 的写的门从 `@/hooks` 导入谓词，是产品代码里第一处 store → hooks 的依赖，方向反了；T7 开头以单独的 refactor 移到 `core/lib`（`e113199e`），读者与 `vi.mock` 路径随之改。
- **写入在轮到时算**（T3，R6）：成员的增减是单独的 `changeMembers(moduleId, { add, remove })`，轮到时以 nerve 最后回答的 `member_ids` 算出整集合；删除与链接的改删在轮到时找对象，找不到就在本地失败、不问 nerve；模块与链接的修改共用一个 `oneAtATime` 队列；删除确认后四个读都为空，在外的取数答到之后仍不显示。
- **排序**（T4，R14）：`orderModuleRows` 按每个选项的取值排，降序是比较器反过来（相等的两向都保持传入次序），没有目标日的两向都在最后，建立时刻经 `Date.parse`（nerve 写出的秒的小数位数不定，按文本排不对），进度按取整的百分比比。
- **侧栏、行与卡片的修改**（T8、T10）：每个控件一份 `useModuleChange`（在途停用、只在确认后提示、被拒回到 nerve 的值、控件卸载后只进 store）；日期只发完整的一对或清除、与现值相同不发，`ModuleDates` 由侧栏、行与卡片共用；卡片改为伸展的链接（清掉 5 条告警），行只挂一处快捷操作，负责人查不到时写“已离开的成员”。`module_sidebar_collapsed` 改按布尔存，同一浏览器里旧的折叠状态读回为展开一次。
- **链接与元数据**（T9，R10）：链接一节人人可见，写只给能写且模块未归档的人；元数据重读经页面的 `["MODULE_LINKS", id]` 键重新校验，按（一代、链接的 id 与地址的集合）开一轮，每秒一次、至多 30 次、30 秒后停，遇拒绝、卸载、换会话即停；地址照输入发出，由 nerve 规整；删除的跟进归链接一节（行在确认后随 store 卸下，提示须由活得更长的一节给）。
- **跨领域读者与删除**（T12）：命令面板的模块命令以 `canWriteModules` 且模块未归档为门，被拒时说原因；收藏的模块分支删去，收藏项显示收藏自带的名字（M7 加回）；通知页不再请求工作区的模块；`@nerve/types` 的 `module/` 与 `ILinkDetails`、`@nerve/utils` 的 `module.ts`、`router.moduleId` 删去；没有读者的文案键与基线 `aadaf61a` 相同（93 个），基线就没有读者的九个模块文案键分给 P6b、P7（spec §5）。
- **e2e 的世界与帮手**（T13、T14）：模块的世界在迭代夹具的世界上打开模块（同五个人、同项目），与迭代共用的帮手从 `cycle-pages.ts` 导入；每页只载入一次，之后应用内移动，以目标页独有的内容为锚（P5b §5 的教训，[M6-M8-09](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-09)）；G12 用接口 G12 与迭代页 G5 的 `progressWorld`；卡片经 `moduleRow` 读，不另建同体的定位。`dismissToasts` 的修正与 T14 的等价变异见 §5。
- **修复轮 F 的取舍**（F2–F4，T16 的主控裁定）：模块名字的输入法另立文件 `g9-module-name-page.spec.ts`（G9 已 393 行）；焦点轮廓的量法从 M4 抽出、不抄第二份；F2 量出的四种共用控件以外的焦点问题都在两域或全应用共用的组件里，照 spec §5 交 P8，不在本段扩大范围（与合入前的 main 相同，不是本段引入），P5a 没有为迭代页做的核对一并交 P8。`toggled` 进 `@nerve/utils` 的 `planning.ts`（`toggledOptions`，与 `matchesModuleFilters` 同文件）；搜索框在迭代页头的测试不改断言即通过之后才抽出（T11 曾裁定两域各留一份）。
- **模块表单名字的输入法**（T15 之后，主控裁定）：T15 写回执时发现 M4 交接 M6-04 的这一项没有核对，不交 P8，照 P5a 修复轮 F2 的先例在本段的修复轮做（F2）。
- 其余见台账。

## 4. 实施记录

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `45cb4482`、`3b73b19a` | 16 个 Task；预检 re-68 的修正（I1、I2、M1–M6） |
| T1 回填与阶段 | `0a7313f1` | P5b 阶段记录 §7 与抬头、P5b spec A8、设计 §14、keywords 的 phase `M6/P6` |
| T2 service、store 的读与包装层 | `d71ab782` | `ModulesService`（13 个操作）、`ModuleStore` 的五个 `ReconciledByKey` 与读、包装层按 `module_view` 取模块；12 个变异失败 |
| T3 store 的写 | `f54b3760` | `ModuleChanges` 的九个写一个接一个、成员的增减在轮到时算、删除后四个读为空；7 个变异失败 |
| T4 筛选 store、界面类型与工具函数 | `6a088615` | 会话的 `ModuleFilterStore`、`ModuleLayout`/`ModuleOrderBy`、`matchesModuleFilters`/`orderModuleRows`；9 个变异失败 |
| T5 取数门与共用件 | `3f46edea` | `usePlanningReads`、`planningOn`；读失败、停用空态、分布、进度页签迁出迭代目录，`comparisonLine` 删去；迭代的既有测试不改；9 个变异失败 |
| T6 写的门、宿主与弹窗 | `cd366811` | `canWriteModules`/`canDeleteModule` 与两域共用的 `planningWritesRefused`、`ModuleActionsHost`（修改）、新建/修改弹窗与字段错误；22 个变异失败 |
| T7 快捷操作、删除、归档与恢复 | `e113199e`、`e9fc041b` | 第 0 步把开关的谓词移到 `core/lib`（主控裁定）；三个确认框归宿主、只在确认后跟进、被拒留着；`copy-hook` 例外 10 → 9、告警上限 53 → 52；14 个变异失败 |
| T8 详情页与侧栏 | `c09d4f8d` | 以 `moduleId` 为 key 的详情布局、侧栏四个控件经 `useModuleChange`、`ProgressChart` 与旧 progress-stats 删去、嵌套解构的守卫一支；17 个变异失败 |
| T9 链接 | `d8c87d1a` | 链接一节在新 store 上、人人可见、只在确认后跟进，元数据有界重读；`copy-hook` 例外 9 → 8；21 个变异失败 |
| T10 列表页、两种版式、页头与预览 | `82dac1ef` | 列表页五种态、显示设置 hook、列表与卡片、行内的状态与日期、预览；告警上限 52 → 47；26 个变异失败；检查点 |
| T11 搜索、筛选与归档目录 | `e2d967a1` | 页头的搜索与筛选在会话的 store、成员读按 slug、已选筛选条、归档目录；日期的筛选件两域共用；告警上限 47 → 44；13 个变异失败 |
| T12 跨领域读者、旧代码删除与守卫 | `dc9172b0` | 命令面板、收藏、通知页改用新 store，旧 service、store、hook、类型删去，`old-module-urls`，store 数 23 → 21，`copy-hook` 例外 8 → 7；11 个源码变异与 2 个守卫变异失败 |
| T13 页面故事一 | `b1d2578e` | G9、G10、G13（6 例）与夹具 `module-pages.ts`，S2 移出的两条由 G13 接上；共用的 `dismissToasts` 按 Base UI 的根因修正；期望错误的 RED 4 个、e2e 变异 7 个 |
| T14 页面故事二 | `b8ed29df` | G12、G19（4 例）；期望错误的 RED 8 个、e2e 产品变异 8 个（一个是嵌套的），单去会话守卫的一个等价存活已说明 |
| T15 文档 | `7d355551` | 差异清单 11 行、前端改动清单 5 行、M1–M5 交接的处理结果、设计 §8/§12/§14、总体设计 §7.7、spec §5、M6-M8-10 与 M6-M8-01/03 的补记 |
| 修复轮 F | `7b347f58`–`1f41c8ca` | F1 G9 用夹具的 `listedOn`；F2 模块名字的输入法与模块页共用控件的焦点轮廓（RED 4 个、产品变异 4 个）；F3 删去桶文件（re-69 M1）；F4 `toggledOptions` 与 `PlanningListSearch` 两域共用（re-69 M2，RED 1 个、变异 2 个）；F5 `cycleMenu` 改名 `itemMenu`（re-69 M3）；F6 re-69 的记录与 M6-M8-11 |
| T16 记录 | 本记录所在提交 | re-69 §4（修复轮的限定复审）、本记录、spec §4 的勾选与 §5、spec 与 plan 的状态、设计 §14、前端改动清单 M6 一行、M6-M8-03 的再现、M6-M8-11 的措辞 |

## 5. 门禁与 CI

- **检查点**（T10 之后，`82dac1ef`）：
  - `t10-gate`：`make gen-check`、`make lint-go`、`make test`（本机 922 秒通过）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 451 过、1 败：W6 (API)（`w6-sign-up-by-invitation.spec.ts:13`）测试一侧的 pg 连接池报 “too many clients already”（`e2e/fixtures/db.ts:88`），即 [M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03)，只重跑这一个文件一次 8 条通过（`t10-gate-rerun-w6`）。
  - 分支 CI 43418051413 attempt 1 全绿：web 131860810563、server (0) 131860810671、server (1) 131860810717、e2e 131867770453（e2e 作业 11:49:39Z–12:14:57Z，25 分 18 秒；E2E 一步 23 分 09 秒，452 过，2 个 worker）。比基线 main CI 43373039561 的 e2e 作业 19 分 14 秒（E2E 一步 17 分 37 秒，453 过）多约 6 分钟。逐条比对：页面故事整体慢 36%、API 故事慢 11%，与模块无关的故事（工作项列表、文件、项目设置）同样变慢；本机同一套（`t10-gate-build-e2e`）与 P5b 的最终门禁逐条比，页面 1.00、API 0.99、其他 1.02，总时长都是 3.4–3.5 分钟；没有改 Go 的 server 两路也一快一慢（10:38 → 9:02、13:49 → 15:51）。判为 runner 的速度差异，不是产品的回归；低于 30 分钟的报告线，记入 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的余量复核。
- **三种引擎的页面配置**（`playwright.m6-pages.config.ts`）：T13 新故事三引擎 18 例、`--repeat-each=3` 54 例、整套 75 例；T14 新故事 12 例、`--repeat-each=3` 36 例、整套 87 例；F2 的两个文件 12 例、`--repeat-each=3` 36 例，M4 cleanup 配置 18 例复跑；都通过。修复轮的终树（`1f41c8ca`）：`f-final-m6-pages` 90 过（T14 时的 87 例加 G9 输入法 3 例，0 flaky），`f-final-repeat3`（G9 两个文件与 F5 改名触及的 g3、g8、g19-cycle-pages、g13、g19-module-pages）`--repeat-each=3` 三引擎 135 过；同一棵树上 `f-final-lint-web`、`f-final-knip`、`f-final-test-web`、`f-final-build` 通过。
- **最终候选**：`f-final-gate`（`1f41c8ca78a3607174ba6bb3095a58a5dce87b2a`）：`make gen-check`、`make lint-go`、`make test`（本机 938 秒通过）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 461 过、1 败：W9（`e2e/stories/workspace/w9-deactivation.spec.ts:30`）第一次 `refusedBoth`（故事第 96 行，经第 91 行与 `e2e/fixtures/users.ts:44` 的 `nerveUsersFails`）的 `nerve users deactivate` 命令连不上数据库，报 `FATAL: sorry, too many clients already`（SQLSTATE 53300），即 M6-M8-03 的签名（P2 那次 W12 的命令一侧同类）。P6a 没有改这个故事、`server/` 与命令；只重跑这一个文件一次 7 条通过（`f-final-gate-rerun-w9`）。
  - 修复轮的限定复审与这次门禁并行，结论见 §6。
  - 本记录所在提交只改文档，另跑文档链接、关键词与 `make lint-web`（收据 `t16-record-*`），其余以准确 HEAD 的分支 CI 为准。
- **CI 的 e2e 时长估算**（推送前，spec §2.11）：迭代页面故事（19 个）在 CI 与本机的时长之比，main CI 43373039561 为 1.67，检查点 43418051413（慢的 runner）为 2.25；本段新的页面故事在 `f-final-gate` 本机 chromium 合计 83.5 秒（g9 24.8、g9-name 3.8、g10 9.9、g12 6.3、g13 19.4、g19 19.3）；在 CI 上约 139–188 秒的串行测试时间，2 个 worker 摊开后墙钟约加 1.2–3.1 分钟：正常的 runner 约 20.5–22.5 分钟，慢的 runner 约 26.5–28.5 分钟，都低于 30 分钟的报告线（期限 35 分钟），所以推送。实际的 e2e 作业时长见 §7。
- **e2e 的教训**（交 P6b）：
  - **关通知**（T13）：Base UI 1.0.0-beta.3 的 `ToastRoot` 在挂载时与它的 ResizeObserver 第一次回报（约一帧之后）各量一次高度，都把该通知的 `transitionStatus` 写回 `undefined`；关闭把它置为退场并清掉计时器，所以两次测量之间按下的关闭被第二次测量撤销，通知永远留着。P5a、P5b 的故事在通知出现后总有别的等待，碰不到这一帧；G9 故事二在被拒的通知出现约 7 ms 后关它而失败。共用的 `dismissToasts`（`e2e/fixtures/cycle-pages.ts`）改为按下尚未退场的通知的关闭、2 秒内都消失为止，未消失就再按，至多 10 秒（`expect.toPass`）；退回只按一次的变异 `t13-mut-dismiss-once` 让 G9 故事二三次全败。P6b 照用它，不在故事里另写关法（[M6-M8-10](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-10)）。
  - **等价存活的变异**（T14）：G19 故事一“另一个标签页换账号后，迟到的回答不在新页面提示”有两道守卫：`followInSession` 的会话守卫与弹窗自己的 `mounted` 守卫。单去会话守卫（`t14-e2e-mut-follow-any-session`）存活：换会话换了 `RootStore`，命令面板 store 的新建开关为假，旧的弹窗随之卸下，`mounted` 已挡住跟进，是等价变异；两道同时去掉的嵌套变异（外层 `t14-e2e-mut-follow-any-session-unmounted`、内层 `-inner`）让故事在第 88 行失败。有两道守卫时以嵌套变异证明故事守着它们，并在台账写明单去一道为何等价，不为让变异失败而削弱另一道。同理，“经面包屑换模块后旧模块迟到的进度不写入”没有能编译又让故事失败的产品变异，以期望错误的 RED（`t14-red-g19-late-progress-shown`）证实断言起作用，key 由组件测试的变异钉住（`t8-mut-subtree-unkeyed`、`t10-mut-peek-unkeyed`）。
  - **焦点轮廓的量法**（F2）：M4 C12 的 `visibleFocus` 中量轮廓与裁切的部分原样抽成 `e2e/fixtures/browser.ts` 的 `focusOutline`，M4 的故事改调它（m4-cleanup 配置三引擎 18 例照过）；模块页的 `expectOutlinesWhole`（`e2e/fixtures/module-pages.ts`）逐个以键盘到达控件，只核“画出了、没被视口或隐藏溢出的容器裁掉”，不核宽度：Chromium 的 `auto` 轮廓算出 1px，Firefox、WebKit 为 3px，M4 的“宽至少 2”不能跨引擎用。量法先以 RED 证明看得见真的裁切（`f2-red-g19-row-link-outline`：把行的链接列进去，即报被 `DIV` 裁），两个产品变异（菜单放进 `overflow-hidden` 的容器、菜单按钮加 `focus-visible:outline-none`）都在轮廓的断言处失败。P8 核迭代页时把 `expectOutlinesWhole` 移到 `browser.ts` 的 `focusOutline` 旁（re-69 §4 的观察，spec §5 的 P8）。
- **最终 HEAD 的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-69](../../../reviews/re-69-M6P6a整体审核.md)，`fable`，范围 `aadaf61a..7d355551`）逐项核对了：

- 数据层（模块 store 的写入与重放，后续动作只在 nerve 确认之后）；
- 门（每一处取数门与写的门都与 nerve 的 `decide.go`、`lock.go` 一致）；
- 会话与路由（新代码不读 router store，SWR 只经 `useSessionSWR`）；
- 删除与守卫（旧的模块代码删除完整；`old-module-urls`、`project-address-route` 与嵌套解构一支、`copy-hook` 的例外降到 7、文案键成对）；
- 回归边界、e2e 与文档的准确性。

结论 Ready after fixes：没有 Critical、Important，三处 Minor 与一条观察，没有须负责人决定的事。修复轮 F 处理三处 Minor（F3–F5），观察记入 [M6-M8-11](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-11)（F6），另做主控在 T14、T15 复核时留下的两项（F1、F2）。修复轮的限定复审结论 Ready to merge：每一处修复只做它所说的，删除完整，两个共用件各有两个读者，F2 的核对是真的，回归边界照守；一处 Minor 措辞与两条观察由本记录所在的提交处理，见 re-69 §4。

## 7. 集成、归档与接续

由 P6b 的 T1 按归档里的台账、`manifest.json` 与 `cleanup.log` 回填；作业号、时刻与结论以 GitHub 公开的 jobs 接口（只读）核对，e2e 的条数与失败处以作业日志核对。

- **最终 HEAD 的分支 CI**：`8188582e184c14a4ae9f29baf512467dbef62e6f` 的 43463416098 attempt 1 成功（server (1) 132016072027、web 132016072634、server (0) 132016072832、e2e 132025064734；e2e 作业 16:45:49Z–17:07:27Z，21 分 38 秒，462 过，在 §5 估算的 20.5–22.5 分钟内）。
- **合入与 main CI**：noff `06b72d4b65bcba911344a4ed3c358f164a5fdeeb`，双父 `aadaf61a`、`8188582e`，树与分支相同（`c8da8856`）。main CI 43469787621 attempt 1：server (0) 132038343932、web 132038344421、server (1) 132038344536 成功；e2e 132048379310（e2e 作业 17:25:42Z–17:49:59Z）461 过、1 败，失败的是 M4 的 C7（`e2e/stories/issue/collaboration-interactions.spec.ts:422`，键盘选主题：第二次 `reopenDuringExit("Meta+k")`（`:474`）期待面板停在 `data-leave`（`:464`），面板已 `hidden`、`data-leave` 为 null），即 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 记下的间歇失败；P6a 在命令面板只改了模块的命令与菜单（迭代的菜单改用共用的取数门），没有动主题选择与这个故事；照 M6-M8-01 的验收语只重跑一次。attempt 2 只重跑 e2e：132062963902（e2e 作业 17:51:22Z–18:16:22Z，25 分 00 秒，462 过）成功；web、server (0)、server (1) 沿用 attempt 1 的成功（attempt 2 中列为 132063047173、132062966245、132063053589，起止时刻与 attempt 1 的相同）。这次的 25 分 00 秒是 P6b 的 e2e 时长基线（[P6b spec](../specs/P6b-module-list.md) §2.10）。
- **归档**（主目录，P6b 开工时以 `archive.py verify` 复核，收据 `t1-p6a-archive-verify`）：`.superpowers/sdd/M6-P6a-integration-20261114`：2910 个文件、1,011,232,897 字节、2938 个成员、未知 0、清单 sha256 `9c627e2ba4a8eaefe0cc1ed547a9cfd1885e3718f86e6b306dcaf7288ca157c4`，清单的 `head` 为 `8188582e`（台账、调研笔记、收据、变异、e2e 的 `test-results` 与 `playwright-report`、各包的 `.turbo`）。
- **清理**：`cleanup.log`（2026-11-13T18:18:06Z）记录 `codex/m6-p6a`（`8188582e`）本地与远端一致、已含于 origin/main，main 与 origin/main 都在 `06b72d4b`；worktree `m6-p6`、本地分支与远端分支已删除。

**接续**：P6b（模块页的工作项列表，[P6b spec](../specs/P6b-module-list.md)、[P6b plan](../plans/P6b-module-list.md)，预检 [re-70](../../../reviews/re-70-M6P6b预检.md)），基线 main `06b72d4b`。
