# M6/P4 阶段评审记录

> 已完成（2026-11-10，noff `c3ba5a40`；合入后 main CI 的 e2e 失败经根因修复第二次 noff `a2b50624`，main CI 42824266166 首次通过），整分支审查 Ready to merge（[re-63](../../../reviews/re-63-M6P4整体审核.md)）；分支 CI、合入、main CI 与归档清理见 §7（由 P5a 的 T1 回填）。基线 main `ed8f0df4e8982dc50a4560a31fdf9b2964d84ac7`（P3b 的 noff）。

## 1. 评审方式与范围

P4 交付设计 §12 P4 一行：迭代与模块的进度、列表行的计数、迭代的冻结与转移。

- **调研**：写 spec 前，两个只读调研代理对照 main 写出现状笔记，随台账归档：一个查后端（planning 的读写之路、端口、迁移、可见范围的谓词、周期任务与交错测试的工具），一个查 Plane 的进度、燃尽、快照与转移的确切行为和网页现读的字段。
- **界定**：[spec](../specs/P4-progress-transfer.md) 与 [plan](../plans/P4-progress-transfer.md) 界定 14 个 Task。
- **预检**：实施前一次 fresh 只读预检（[re-62](../../../reviews/re-62-M6P4预检.md)），结论为修正后可实施。两处 High 在实施前改入计划：
  - 权限矩阵的覆盖守卫要求每个操作有行，矩阵行随操作在 T5、T9 加入；
  - 转移格子的目标迭代与归属由该行的 `prepare` 按格子建立，不改共用种子。

  另删去“改日期即清掉冻结”（不充分，且给改迭代加了隐藏的副作用），冻结写下后不清。
- **实施**：逐 Task 由实现代理完成（T3 先于 T2，因为存储用 T3 的领域类型），主控复核并记台账；T1、T13 的文档由主控完成。
- **门禁与审查**：T6 与 T14 各一次串行七项门禁；T14 一次 fresh 只读整分支审查（[re-63](../../../reviews/re-63-M6P4整体审核.md)）。

## 2. 验收状态

A1–A7 满足（[spec §4](../specs/P4-progress-transfer.md#4-验收标准)已勾选），证据见 §4：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | G5：迭代的计数、分布、燃尽 | T3、T4、T5、T12 |
| A2 | G6：冻结 | T2、T7、T10、T12 |
| A3 | G7：转移 | T8、T9、T10、T12 |
| A4 | G12：模块的进度 | T3、T5、T12 |
| A5 | 列表行 | T6、T11 |
| A6 | G17 与加锁 | T5、T9、T10、T11 |
| A7 | 契约与文档 | T1、T5、T6、T9、T13 |

A8（CI、合入与归档清理）由 P5a 的 T1 勾选，证据见 §7。

## 3. 局部裁定

台账中约 110 条 `Ruling:`，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **计算的分工**：issue 按状态 id 回答工作项事实与批量计数（语句不连 `states`、`cycles`、`modules`），planning 经项目的 `IssueStates` 得状态组，在 `planning/domain` 的纯函数里算出全部数字；列表行与进度共用同一条组规则（`Counts.add`）。
- **冻结的格式**：`cycles.progress_snapshot` 存接口的 `FrozenProgress`；解码严格（解码再编码比对），以后改格式要先回填旧行。列表行只取六个数与时刻（语句去掉分布与燃尽）。
- **冻结任务**：候选不加锁、只为界定超集读一次时钟；每个项目一个任务事务，先锁后读时钟；已归档的项目、关闭开关的项目、已归档的迭代照样冻结；不改迭代的最后修改。权限矩阵的种子迭代已结束，每次启动都被冻结，所以矩阵的 `prepare` 先经用例冻结它们，读进度的格子按冻结断言。
- **转移**：先冻结（已冻结的不重算）再移动；只锁归属行（`FOR NO KEY UPDATE OF r`），工作项行最后由 `MarkChanged` 写，锁序同 spec §2.6；两条语句（移出、插入），因为行 id 由应用生成；不发布 `issue.updated`。
- **两条文案**：`planning_cycle_completed`、`planning_cycle_not_completed` 改为在各个回答它们的场合都成立的说法（en 与 zh-CN），服务端按场合各有消息，不新增码。
- **测试的时钟**：组合测试锚定在上海正午的业务时钟（`newNoonWorld`），e2e 的项目建在此刻离正午最近的 `Etc/GMT±N` 时区，避免跨午夜的误判。
- **两处回归夹具与辅助函数**：`seedCycles` 的 Sprint 改为围绕数据库当天（固定日期 11-16 起成为已结束，启动的冻结运行会在比较整行的删除测试中途写入）；P1 的 `newPlanningRouteObserved` 拆出 `newPlanningRouteWith`。两者都不改断言。
- **聚焦回归的正则**：T2 发现计划写的 bootstrap 正则只匹配到 9 个测试（测试名多以动词开头），改为按子串匹配。
- 其余见台账。

## 4. 实施记录

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `fafcc1dd`、`c832903c` | 14 个 Task；预检 re-62 的修正 |
| T1 回填与阶段 | `908f4e23` | P3b 阶段记录 §7、spec A8、设计 §14；P3b 归档复核；phase `M6/P4` |
| T3 进度的纯计算 | `735cac67` | 计数、分布、燃尽、冻结的编解码；13 个变异失败 |
| T2 冻结列与存储 | `534d9cf0` | `00048`、五条语句；13 个变异失败 |
| 计划修正 | `16a00281`、`3a7dddd0` | 子串正则；参数名 `sub_issue` |
| T4 统计端口 | `a4b56a30`、`52d3400e` | issue 的五条语句、planning 的 `IssueStats`、项目的状态组；30 个变异失败 |
| T5 进度接口 | `a92a2923`、`90a6224c` | 两个 `/progress`、一个快照、OwnOnly 不读冻结、矩阵两行；与列表引擎的总数对照；9 个变异失败；测试锚定上海正午 |
| T6 列表行与检查点 | `e6175248` | `CycleListItem`、`ModuleListItem`、每页每端口一次读；11 个变异失败；检查点 |
| T7 冻结任务 | `e664868e` | `planning.freeze_cycles`、配置、六处周期种类清单；7 个变异失败 |
| W11 的根因 | `38c74554` | 测试的等待条件分不清两个工作区（见 §5） |
| T8 issue 一侧的转移 | `c76eab7d`、`d08ee622` | 两条语句、`CycleMoved`、`MarkChanged`、planning 的 `IssueTransfer`；18 个变异失败 |
| T9 转移的接口 | `4404ba28` | 拒绝次序、先冻结后移动、两条文案、矩阵行；14 个变异失败 |
| T10 交错与锁 | `84af0b02` | 五组交错两种次序、锁超时 503、端到端的锁后读时钟；4 个变异失败 |
| T11 矩阵的计数 | `e706371c` | 154 格计数可见性；转移的“访客看全部”变体；8 个变异失败 |
| T13 文档 | `6b5a322b` | 差异清单、设计 §3.5/§5/§6/§9、五份交接回执、spec §5 的 P8 跟进 |
| T12 接口故事 | `9cf2348d` | G5、G6、G7、G12，各先 RED 后 GREEN |
| T14 整审修复 | `18aeb25a` | re-63 的 m1–m3、本记录 |
| 分支 CI 的根因修正 | 本记录所在提交 | `newNoonWorld` 交出截到微秒的时刻（见 §5） |

## 5. 门禁与 CI

- **检查点**（`e6175248`，T6 之后）：七项门禁串行全过，输入前后一致：`make gen-check`、`make lint-go`、`make test`（66 个包，903 秒）、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`（432 条全过）。
  - 第一次推送时，经 443 端口的 SSH 被网络代理关闭连接（`ssh -T` 同样失败，GitHub 的 HTTPS 正常），等待恢复后按提交号推送，推送方式不变。
  - 分支 CI 42775845319 attempt 1：server (0) 129694033698、server (1) 129694033912、web 129694033946 成功；e2e 129699354565 失败 431/1，W11 第 190 行（Ctrl+K 后命令面板 0 个），签名与[记录 06](../../claude_to_gpt/06-ci-w11-flake.md) 相同。按已知偶发只重跑失败作业一次：attempt 2 的 e2e 129706674516 成功。
  - **W11 的根因**：从产物 `playwright-report` 的 trace 还原帧快照：`moveWithinApp` 返回后地址已是 acme 的，内容仍是 Beta 的；等待条件 “Add Project” 两边都有，立即通过；面板在 Beta 的内容里打开，随 Beta 的宿主卸载而关闭。修正测试的等待条件（`38c74554`），应用不改；记录 06 §8 写证据。
- **最终候选**（`9cf2348d`，`t14-gate1`）：七项门禁从头串行一次全过，输入前后一致：`make gen-check`、`make lint-go`、`make test`（66 个包，955 秒）、`make lint-web`、`make knip`、`make test-web`（turbo 缓存重放，输入与上次相同）、`make build && make e2e`（436 条全过，含四个新故事）。之后只有本节与整审 Minor 的文档修改和 `e2e/fixtures/assert/progress.ts` 的一段注释（re-63 m3），以 `make lint-web` 与文档链接检查覆盖。
- **分支 CI 第一次**（`18aeb25a`，42800976748）：web 129778085248、server (1) 129778085665 成功；server (0) 129778085598 失败，e2e 因此跳过。失败的只有 `TestTransferKeepsTheJobsFreeze`（T9）：冻结的时刻与测试取的“此刻”不相等。
  - **根因**：`newNoonWorld` 交出的“此刻”是 `time.Now()` 加偏移，Linux 上带纳秒；业务时钟与 `clocktest.At` 都截到微秒，存下的冻结时刻因而差几百纳秒。macOS 的时钟只到微秒，所以本机门禁通过。测试的缺陷，产品不改。
  - **证据**：本机给 `newNoonWorld` 的返回值加 123 纳秒，复现与 CI 一字不差的失败（收据 `ci-red-ns-now`）；修正为 `.UTC().Truncate(time.Microsecond)`（同业务时钟的读法）后，同样加 123 纳秒也通过（`ci-green-ns-now`），用到它的组合测试全部通过（`ci-green-noon-tests`），`make lint-go` 通过。本阶段新测试里其余直接取墙上时钟的地方都已截到微秒或不与存下的时刻比较。
  - 按负责人 2026-11-08 的规则不重跑，修正后以新提交重跑分支 CI，结果见 §7。
- **合入后的 main CI**（noff `c3ba5a40`，42809569341）：web 129807294464、server (0) 129807294819、server (1) 129807295028 成功；e2e 129815035293 失败 435/1，是 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 所记 M4 的 C12 (review) list 布局（`review-interactions.spec.ts:117`：按 Escape 后“Create new work item”弹窗 5 秒内未关）。同一 tree 的分支 CI 42803535189 的 e2e 是通过的。
  - **根因**（产品缺陷，不是测试的）：从产物 `playwright-report` 的 trace 还原帧快照：弹窗在 Enter 之后约 0.66 秒收到 Escape，此时描述编辑器的 ProseMirror 元素还不存在（Escape 之后的快照才有），按键之后出现错误提示 “Editor is not ready to discard changes”，弹窗拒绝关闭。TipTap 以 `immediatelyRender: false` 在首次提交之后的 effect 里才建编辑器，而 `useEditor` 的 `useImperativeHandle` 在首次提交就把一套操作空编辑器的 API 挂上引用，其 `isEditorReadyToDiscard()` 因编辑器不存在答 false；Headless UI 挂在 window 上的 Escape 监听可以在这次提交与 effect 之间运行（CI 机器慢时窗口更长），`requestClose` 于是把“编辑器还在创建”当成“编辑器忙（有未完成的图片）”。M4/P14（`09525b31`）已让引用为空（延迟加载的代码还没到）时可以退出，漏了这个窗口。
  - **修正**（修复分支 `codex/m6-p4-ci`）：编辑器包的 `useEditor` 只在编辑器存在时挂引用，之前为 null，“引用存在”即“编辑器存在”，弹窗的既有规则因此覆盖这个窗口；其他调用方都经可选链调用，或先判 `isEditorReadyToDiscard()` 为真，行为不变。新单测 `web/packages/editor/src/hooks/use-editor.test.tsx` 在首次提交读引用：修正前得到整套 API（收据 `red-ref-before-editor-3`），修正后为 null，编辑器建成后可以放弃（`green-ref-before-editor`）。只读复审 Ready to merge，无 Critical 或 Important；它指出的收集箱新建弹窗同类问题（沿用 Plane，这次修正前后行为相同）交 M7（[M6-M7-01](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-01)）。门禁 `ci2`：文档链接、`make gen-check`、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`（436 条全过）；diff 不含 Go，`make lint-go` 与 `make test` 交给分支 CI。
  - 按负责人 2026-11-08 的规则不重跑这次 main CI；修复分支合入后由新的 main CI 验证，结果见 §7。
- **本机环境**：T8 的一次 bootstrap 回归在 testcontainers 等容器就绪时 Docker API 超时，未到数据库，重跑通过（P3b 的 t14-gate1 有一次同类），记入 M8 的观察。

## 6. 整分支审查

fresh 只读审查（[re-63](../../../reviews/re-63-M6P4整体审核.md)）逐项核对了：

- 数字的口径（计数、分布、燃尽、冻结的编解码）与 HTTP/OpenAPI 的映射；
- 每条计数之路的可见范围与 issue 的五条统计语句；
- 读的快照、冻结任务与转移的锁、事务与时钟，以及与其余写者的锁序；
- 端口的方向、`issue.Provide`、守卫没有放宽、文件大小；
- 测试的 RED 与变异、回归的边界、矩阵的增加与 e2e 的时区与轮询；
- 配置与周期任务；W11 的修正；差异清单的 Plane 引用、设计与五份交接回执。

没有 Critical 与 Important，Ready to merge。三条 Minor：

| 编号 | 问题 | 处理 |
|---|---|---|
| m1 | spec §2.6 仍写“一条语句”，实现是两条 | spec 已改，写明理由 |
| m2 | 冻结任务遇失败项目即停（按 spec，同自动归档） | 记入 spec §5 的 P8 |
| m3 | G6 以测试机的时钟作冻结时刻的上界 | 夹具加注释写明前提 |

## 7. 集成、归档与接续

由 P5a 的 T1 按两个归档里的台账、CI 收据与 `cleanup.log` 回填。

- **最终候选的分支 CI**：
  - 第一次，`18aeb25ae9e43e9c4d724be98a19bc64ab63a2a0` 的 42800976748 attempt 1：web 129778085248、server (1) 129778085665 成功；server (0) 129778085598 失败（`TestTransferKeepsTheJobsFreeze`，根因与修正见 §5），e2e 129784645493 因此跳过。
  - 修正后，`292f40fe3a9b1d4899fbd27b4f0fbf8ed2073399` 的 42803535189 attempt 1 成功（server (0) 129786842989、web 129786843414、server (1) 129786843437、e2e 129794166643）。
- **合入与 main CI**：noff `c3ba5a40b38380e03a03334e1c680763e991b587`，双父 `ed8f0df4`、`292f40fe`，树与候选相同（`5489b877`）。main CI 42809569341 attempt 1：web 129807294464、server (0) 129807294819、server (1) 129807295028 成功，e2e 129815035293 失败 435/1（C12，根因与修正见 §5）；按负责人的规则不重跑。
- **修复分支与第二次 main CI**：`codex/m6-p4-ci`（自 `c3ba5a40`：`675de55e` 修正与单测，`d7824b26` 复审结论与 M7 的交接）的分支 CI 42818532689 attempt 1 成功（web 129837762256、server (1) 129837762619、server (0) 129837762738、e2e 129846836535）。noff `a2b50624fd872c183628a55d04b7961f9a1dbc60`，双父 `c3ba5a40`、`d7824b26`，树与修复分支相同（`16eae5f1`）；main CI 42824266166 attempt 1 成功（web 129857313676、server (0) 129857314105、server (1) 129857314265、e2e 129866383345）。
- **归档**（主目录，P5a 开工时以 `archive.py verify` 复核，收据 `t1-p4-archive-verify`、`t1-p4ci-archive-verify`）：
  - `.superpowers/sdd/M6-P4-integration-20261110`：3823 个文件、532,660,546 字节、未知 0、清单 sha256 `e786eaba5e9d4b9a06d551babd3194799c4d8ea21b65bfdf7c3a3b0e9573e430`（台账、调研笔记、收据、CI 原件、W11 的产物与解析工具）；
  - `.superpowers/sdd/M6-P4-ci-20261110`：2521 个文件、64,500,851 字节、未知 0、清单 sha256 `dd85429f4bdda14cd53a8eac74a7994f2dde95ea5b8aeaf9c5ceef8d6cd7d50b`（修复的台账、复审原文、C12 的 trace 解析工具与 CI 原件）。
- **清理**：两份 `cleanup.log` 记录分支已含于 origin/main，worktree `m6-p4`、`m6-p4-ci`，本地与远端分支 `codex/m6-p4`、`codex/m6-p4-ci` 已删除，主目录停在 `a2b50624`。

**接续**：P5 拆为 P5a（网页迭代的数据层与页面，[P5a spec](../specs/P5a-web-cycles.md)）与 P5b（迭代页的工作项列表）。
