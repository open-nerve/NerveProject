# M6/P1 阶段评审记录

> 2026-11-07：已合入 main（noff `0a601f21`，main CI 42343073949 首次通过），归档与清理完成，见 §7。基线 main `d1ec8c696d017f19aac66471f6737b8ea1678d0b`（M6 设计 noff），实施中合入 main `910a55cd`（e2e 作业期限 35 分钟与 M8 交接 M6-M8-01）。

## 1. 评审方式与范围

[spec](../specs/P1-cycles-backend.md)与[plan](../plans/P1-cycles-backend.md)界定 14 个 Task。实施前一次 fresh builder 对照 main 的实际代码核查设计与 P1 计划，报告见[re-54](../../../reviews/re-54-M6设计与P1预检.md)。实施逐 Task 进行（RED → GREEN → 变异，每次运行有收据），台账在本阶段 ignored 目录 `.superpowers/sdd/P1-cycles-backend/progress.md`，随阶段归档。门禁之后一次整分支 deep-review，报告见[re-55](../../../reviews/re-55-M6P1整体审核.md)，处理见 §6。

## 2. 验收状态

A1–A7 满足（[spec §4](../specs/P1-cycles-backend.md#4-验收标准)已勾选），证据见 §4：A1（G1）→ T4、T7、T12；A2（G2）→ T4、T8、T12；A3（G3）→ T9、T11、T12；A4（G8）→ T10、T12；A5（G18 与权限）→ T6–T11 的矩阵与 apitest、T12；A6（删除）→ T2、T3、T7 的删除故事；A7 → T1、T7–T11 的文案、T13 的回执。A8 的本地七项门禁、整审与修复闭环见 §5、§6；准确分支 CI、noff 后 main CI 与归档清理见 §7。

## 3. 局部裁定

台账中 20 条 `Ruling:`（整审逐条同意，见 [re-55](../../../reviews/re-55-M6P1整体审核.md) §6），均为局部取舍，没有架构待决项，也没有需要负责人决定的问题：

- 表与清理：外键索引只建两条 `CASCADE`（`workspace_id`、`project_id`），指向 `users` 的 `SET NULL` 外键照 `projects.project_lead_id` 不建，代价是账户清理置空负责人与审计列时顺序扫 `cycles`；删除任务参数为 `{project_id, at, by}`，工作区删除展开为各项目；任务事务复用 `newIssueJobTx`，锁等待上限与工作项一致；`planning/app/purge.go` 照抄 project、issue 的同名文件（模块互不 import 的先例，重复约 100 行）；删除与清理守卫、运行角色清理与规模夹具以 SQL 种迭代。
- 端口与加锁：项目端口两法（`PlanningFacts` 兼作不加锁的查找，事实里已有 `workspace_id`）；预检也判 409（归档项目、开关关闭），被拒的调用者不占闸门与连接，锁下再判一次为准；`planning.archived` 与 `cycle_completed` 属各用例；planning 的写不进“工作区行先 FOR SHARE”的清单（设计 §6 沿用 M4 的锁序，交错测试证明它在项目行上等待）。
- 读：只读快照事务（同 M4 的 `ReadTx`），授权、项目事实与迭代行在同一快照；`listCycles`、`getCycle` 也声明 `forbidden`（看得到公开项目但不是成员的调用者，同 `listLabels`、`getIssue`）。
- 组合：`bootstrap/app.go` 由 414 行降到 349 行，配置转交的六个函数原样移到 `settings.go`，模块接线留在 `app.go`；e2e 新增 `furnishPlanningRows`（以项目管理员打开 `cycle_view` 并建草稿），不改 `furnishIssueRows`（它的调用者多是项目成员）。
- 其余（sqlc 条目随第一条查询、矩阵按需扩展、README 清理顺序补上自 M5 漏写的 asset 等）见台账。

## 4. 实施记录

每个 Task 的 RED、GREEN、变异与收据名见台账；这里只记结果与偏离。

- **T1 前端与守卫**：关键词的 `phase` 改为 `M6/P1`，`analytics` 对 `cycle.service.ts` 及其测试的两条例外到期并删除，同时删去指向 Nerve 没有的旧地址 `…/cycles/{id}/analytics?type=issues` 的 `cycleDistribution`、`fetchActiveCycleDistribution` 与进行中迭代卡片对它的调用（`b3007ab1`）。
- **T2 表**：`00039_planning_cycles.sql`；变异“删日期 CHECK”“负责人改 `CASCADE`”被杀死。
- **T3 删除与清理**：订阅项目、工作区删除，在原事务投递 `planning.delete_rows`（沿用请求的 At/By，有界续跑）；60 天清理加入 planning 步骤（asset → issue → planning → project → workspace → identity）；运行角色授权；`TestPurgeScale` 一次运行完成。根因修复一处测试配置：`testConfig` 手写配置绕过校验，planning 的预算为 0，任务一直 snooze。
- **T4 领域**：日期、名字、状态（项目时区的日历日）、重叠、错误目录；变异“严格不等号”“按 UTC 取日”被杀死。
- **T5 项目端口**：`PlanningFacts` 与 `LockPlanningProject`（`FOR NO KEY UPDATE`）；变异“改 `FOR KEY SHARE`”被杀死。
- **T6 加锁之路**：预检在事务外、闸门、事务内先锁项目再重读与终判、闸门最后离开；变异“看不见答 403”“事务内沿用预检”被杀死。
- **T7 建立**：第一个接口与组合（`Module.Register`、bootstrap 的 `planningProjects`），交错测试证明两次重叠的建立在项目行上串行、无 `40P01`；权限矩阵首批、错误文案、`furnishPlanningRows`。检查点七项门禁（`e73e5a18`、`583e6415`）全过，其间 I21 漏调 `furnishPlanningRows`（查调用处时输出被截断）已补；三条编辑器故事在整套负载下间歇失败，单跑 9/9、完整重跑 417/417。
- **T8 读与归档目录**：判定拆出 `decide.go` 的 `see`，读与写共用；读在只读快照事务里判定，无闸门无锁。
- **T9 修改**：合并日期、已结束只能改排序、已归档拒任何修改、无改动不写；自查发现日期按指针比较，加 `Dates.Equal`。提交 `1829458e` 时 lint 未过（提交接在 lint 命令之后），下一提交修正。
- **T10 归档与恢复**、**T11 删除**：重复归档与恢复不写；删除只限管理员与建立者；矩阵按 R10 让每列作用于自己的迭代（另种成员建立的 `Mine`）。
- **T12 接口故事**：`e2e/stories/planning/` 的 G1–G3、G8、G18，管理员用 PAT、成员用登录令牌；先跑两个期望错误的版本，各在预期处失败。
- **T13 文档**：plane-diff、五份输入交接的“处理结果（M6/P1）”（均保持 open）、设计 §14；文档链接 0 失效。

## 5. 门禁与 CI

- **整审前**（`eb1b2208`，T13 之后）：七项门禁串行全过，输入前后一致：`make gen-check`、`make lint-go`、`make test`（66 个包，827 秒）、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`（422 条全过，即原 417 条加 planning 五条，192 秒）。
- **检查点的分支 CI**（42334013393，`c528d682`）：attempt 1 的 e2e 417 条中 416 过，C12（`review-interactions.spec.ts:83` 的 list 布局）失败，即 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 所记的间歇失败；只重跑失败的作业，attempt 2 通过。
- **修复轮之后**：在 `a95a3b8e`（修复轮 `6af33742` 加记录）上串行重跑七项门禁，全部 exit 0、输入前后一致：`make gen-check`、`make lint-go`、`make test`（66 个包，840 秒，`-count=1`）、`make lint-web`（50 项任务中 35 项由 turbo 缓存回放）、`make knip`、`make test-web`（16 项任务全部由缓存回放：`web/` 自 `eb1b2208` 的实跑以来未变）、`make build && make e2e`（422 条全过）。

## 6. 整分支审查

冻结候选 `eb1b2208`，一次 fresh 只读审查（`fable`），报告见 [re-55](../../../reviews/re-55-M6P1整体审核.md)：Ready to merge: Yes，没有 Critical 或 Important，Minor 8 条。修复轮 `6af33742` 修 1、2、4、5、7：列表按项目当天一次推算（同时补上此前在用例测试中存活的“列表按 UTC 取日”变异）、三份事务内投递桥合并为 `jobInserter`、`DeleteCycle` 的值类型参数、app 测试引入 `time/tzdata`、交接回执的文件名。3（第三份清理分页器）交 [M8 交接 M6-M8-02](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-02)；6（删除屏障的时间窗）记录不改；8 随本记录改设计 §14。

## 7. 集成、归档与接续

最终候选推送后核准确分支 CI，`--no-ff` 合入 main 并核 main CI，再归档本阶段 ignored 证据（台账、收据、审者原文、CI 原件）并正常清理 worktree 与分支。以下由 P2 的 T1 按归档的台账回填。

- **分支 CI**：最终候选 `72b83bebb7bd0eef89a41a209925b717bee50a37` 的 42340200507 attempt 1 成功（web 128305957226、server 128305957374、e2e 128309777084，e2e 22 分钟）。
- **合入**：main `910a55cd` 干净且与 origin 一致；`--no-ff` 生成 `0a601f212e32057c4e48fe43e8c4f19691254573`，双父依次为 `910a55cd`、`72b83beb`，树与候选相同；系统 Git 推送。
- **main CI**：42343073949 attempt 1 成功（web 128314830261、server 128314830458、e2e 128318456919，e2e 22 分 44 秒）。
- **归档**：`.superpowers/sdd/M6-P1-integration-20261107`（主目录，ignored）：1982 个文件、245,220,964 字节、未知 0，清单 sha256 `4a07a8abe1fc7bb6f5dd8939c639172a942804af0e81f89602807d9a2f73e541`，`archive.py verify` 通过（P2 开工时复核一次）；`cleanup.log` 记有清理前候选已含于 origin/main。
- **清理**：worktree `m6-p1`、本地与远端分支 `codex/m6-p1` 已删除，主目录停在 main `0a601f21`。
- **接续**：P2（模块的后端），预检见 [re-56](../../../reviews/re-56-M6P2预检.md)。
