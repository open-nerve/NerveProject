# M4/P6 领域事件与自动归档评审记录

日期：2026-10-26。状态：**准备、独立静态预检及修订复审通过，尚未实施 T1**。分支 `codex/m4-p6`；基线 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`。配套：[spec](../specs/P6-issue-events.md)、[plan](../plans/P6-issue-events.md)。

## 1. 评审方式

按 [02-method](../../claude_to_gpt/02-method.md) 第3节允许的方法调整，延续 P5 的紧凑计划：文件/签名/测试/变异责任明确，以实施时真实红绿、变异恢复和逐 Task 独立评审代替本次预制完整产品原型。当前授权是准备 spec/plan 并完成预检；不实现产品。

控制者汇总事件、任务/自动归档、交接三个只读调研，按设计11个粗项细化16 Task。spec作者与plan作者交叉核对后，另由未参与本版文档编写的独立评审者检查事件与事务、任务/锁/时钟、全范围/可执行性；结果与局部裁定在第4节登记。预检采用静态源码对照，不声称逐任务代码重放或原型结果。

## 2. 验收标准逐条核对

| 项 | 真实证据与状态 |
|---|---|
| P5基线 | `--no-ff` 合并 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`，树与已审分支一致；[main CI 40257678959](https://github.com/open-nerve/NerveProject/actions/runs/40257678959) server/web/e2e全部success |
| P5清理 | 先确认远端分支相对main独有提交0；6,517证据文件SHA-256归档一致；P5本地/远端分支与worktree已清理。完整记录见 [P5 review](P5-issue-list-review.md) |
| P6工作树 | 按用户固定路径 `.codex/worktress/codex/m4-p6` 新建，主目录保持main；应用原生worktree工具不支持指定该路径，依用户优先规则使用系统git worktree |
| P6起步范围 | 仅spec/plan/review、设计必要澄清/进度、P5最终集成记录与keywords phase=M4/P6；没有产品实现、迁移或依赖变化 |
| P6静态预检 | 三个独立视角完成首审：High 0、Medium 3、Low 4，均局部；已统一修订；复审另补2 Low也已修订，三个视角全部静态Approved，未关闭发现0 |
| P6产品验收 | spec§4 T1–T16全部未勾；真实InsertTx、worker漏登记、自动归档、交错/race和I16均待实施 |

## 3. 执行中的决定

局部细化以 spec§3 为准：事件payload白名单及删除具体表优先；关系存储方向/真实表情实体；顺序分发与值隔离；强绑定worker登记和独立周期声明；后台项目N、锁下策略与nullable actor；业务Clock统一而River调度真实时间。均为已批准设计的落实，没有重新裁定D3/D4/D7或放宽守卫。

16 Task 是设计约16项上限内的职责切分；规模表为起步估计，不是原型实测。实施前需核真实差异及修复余量。S4只是设计历史输入，T1仍须重做当前代码的真实提交/回滚证明。

## 4. 预检发现与处理

独立首审来源：事件/事务评审 `preflight-events.md`（E1–E3），平台/任务评审 `preflight-jobs.md`（J1/J2），范围/证据评审 `preflight-scope.md`（S1/S2）。报告保存在主目录 ignored `.superpowers/sdd/P6-issue-events/`；作者未担任这三份预检的评审者。首审合计 High 0、Medium 3、Low 4，全部局部，均接受；未发现需要重新裁定的架构问题。

| 来源 / 级别 | 问题与裁定理由 | 修订 / 错判代价 | 复审 |
|---|---|---|---|
| E1 / Medium | 请求busyTx会把55P03转ServerBusy，违背后台保留错误链；接缝分开 | issue.Deps.JobTx接同LockTimeout的raw Tx，bootstrap共用底层构造；请求映射不变。T10真实PG错误/请求503双对照。代价仅额外消费方端口，不加锁或SQL旁路 | ADDRESSED，独立复审通过 |
| E2 / Low | relation/reaction的文本与ID槽不明确，逐值测试可各自自证 | spec四槽表：容器只有ID，文本nil；T5/T6错槽变异。代价为后续消费者必须遵循固定shape | ADDRESSED，独立复审通过 |
| E3 / Low | Event.ID与Entity.ID、批量/连续事件身份未区分 | app生成非零独立ID；跨订阅者不变，同实体连续写/批量互异；置零/复用Entity.ID变异。代价是每事件一次UUID生成 | ADDRESSED，独立复审通过 |
| J1 / Medium | River AddWorkerSafely不会拒绝nil/typed-nil worker，登记集合可能假完整 | New在真实登记/启动前拒绝无效registration，两种nil反例要错误接受变异而非运行panic；不反射River私有字段。代价为平台少量输入校验 | ADDRESSED，独立复审通过 |
| J2 / Low | T7改New四参与worker配置，却把旧bootstrap调用留到T11 | T7同时最小迁移全部调用/配置/测试，T11只最终汇总；代价为T7包含必要接线，保证独立可编译验收 | ADDRESSED，独立复审通过 |
| S1 / Medium | recent对照可能未经过任何任务扫描，RunOnStart的completed可提前满足 | 最后recent写后取本库微秒真实水位，要求attempted_at晚于它且completed；保留spawn门/业务断言；M4 I16与时钟说明同步。代价为最多等下一周期，测试不用固定sleep | ADDRESSED，独立复审通过 |
| S2 / Low | P5当前表/计划仍说保留已删除分支 | 标明实施位置/交付门为历史，最终授权记录优先；不改旧过程事实。代价仅时态澄清 | ADDRESSED，独立复审通过 |

聚焦复审确认上述7项的实质修订闭合，另发现两处 Low 文档接缝并已修订：

- E4/J3（同一处）：T10错误链验证改用 `postgres.LockNotAvailable`，遵守03对锁等待测试的规则；定位命令显式加入既有 `TestABusyTransactionIsServerBusy` 和 `TestAProjectTooBusyForAWriteAnswersServerBusy`，确保用户503对照确实运行。
- S3：M4设计I16的数据库对照列与spec/plan统一：旧非终态在偏移前创建，recent终态在偏移后经API创建/更新，分别证明状态和时间过滤；不让同一新行同时承担两个性质。

这两项同属局部文档修订，未改变产品范围；最终聚焦复核已确认两项闭合。

独立范围评审另核 P5 双亲与tree相同、公开CI三任务全绿、6,517文件/184,269,398字节SHA均符以及远端P5不存在。当前本地关键词守卫72规则/23例外无命中、架构测试通过；这些仅证明起步状态，不证明P6运行功能。

### 最终预检结论与本地准备检查

三个独立聚焦复审 `preflight-events-rereview.md`、`preflight-jobs-rereview.md`、`preflight-scope-rereview.md` 均为静态 APPROVED，High/Medium/Low 未关闭数均0。原始报告与文件SHA留在同一 ignored 台账；正式记录汇总9项局部修订（首审7、复审新增2），无架构待决点。

- `node tools/keywords.mjs`：退出0，72规则、23例外、无命中/过期例外；phase仅由M4/P5改为M4/P6。
- `go test -count=1 ./internal/archtest`：退出0，包测试1.607s；完整命令用时2.299s，stderr空。证据为台账 `verification/archtest.json` / stdout / stderr。它只验证基线架构守卫，P6运行行为未实现。
- 文档自检确认16个Task顺序完整、16条产品验收全部未勾，新增本地链接有效，`git diff --check` 无错误；无产品/迁移/依赖文件变动。M4原有M5–M8未来closeout链接不提前创建。
- 仅文档与关键词标记不重跑已经精确通过的P5完整门禁；main基线CI及内容哈希证明与上表一致。准备提交/推送的精确SHA由交付答复和Git记录给出，不循环提交文档自证自身CI。

## 5. 最终实现与 plan 的差异

当前没有 P6 实现，不存在可报告的实现偏差。起步方法与历史Claude原型/逐字重放不同，已在第1节/spec附录A说明；签名、任务和证据边界不得混称最终实现。

## 6. 移交事项

- P7：链接生产订阅者、请求投递与registrationRows核river_job；P9：删除订阅者/任务、版本保留、物理清理。
- P15：I16页面重新登录、NULL actor展示；M6/M7：迭代/模块/收集箱的自动归档条件。
- M2§1命令只投递交接原文在M4收尾转M5，P6不会让CLI构造jobs；仅统一业务Clock。
- 输入handoff总体status与M4§14/总体§9.4保持开放。本期完成时只追加本期处理证据，具体落点见spec§7。

## 7. 已知限制

本轮仅静态预检，没有P6产品原型、生成物SHA、运行红绿/变异/race/I16或产品CI通过。后续开始T1须逐任务实现、验证和独立评审；阶段完成先不合并，交负责人安排下一步。
