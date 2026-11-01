# M4 收尾：服务端性质与交错复核

2026-11-01，T3 已完成。执行输入为 `91ec709499e9c670f1f1e81a44e1a18c41fb495e`，生产代码未改；所有临时变异均恢复。原始收据位于 `.superpowers/sdd/M4-closeout/`。本记录支持分支验收；最终 main 与规模结果仍按总台账待验。

## 1. 运行与输入

| 运行 | 命令/范围 | 结果与限制 |
|---|---|---|
| `t3-contract` | `go test -p=1 -count=1 -json ./internal/archtest ./internal/modules/access/... ./internal/platform/jobs/... ./internal/platform/outbound/... ./internal/modules/issue/... ./internal/platform/httpserver/apitest ./internal/platform/gate/...` | exit 0，67.413s；506 顶层、3074 含子例通过 |
| `t3-race` | `go test -p=1 -race -count=5 -json ./internal/bootstrap -run '^(43 个具名测试)$'`，完整正则在 `t3-race-command.json` | exit 0，530.024s；43×5=215 顶层、2950 含子例通过；零 skip、零 race、零 40P01 |
| `t3-registration-restored` | `go test -p=1 -count=1 -json ./internal/platform/jobs -run '^TestJobRegistration$'` | 恢复后 exit 0，1.117s，23 含子例通过 |
| `t2-after` | 前一任务同一生产输入的 bootstrap/identity/project/workspace 全量包 | 1177 顶层、6305 含子例通过；含下述全矩阵、runtime、任务注册、迟到清理测试 |

前三个正常运行逐文件输入总 hash 均为 `442ab91b8939d086b75e10fcf6b7a0fbc6f8c9a617c76c4d21930168aa066f3a`。每个 runner 的 JSON 保存 argv、起止、exit、HEAD、输入不变判定和日志 SHA；`t3-race-summary.json` 核对每个测试正好 5 次，不用日志行数推算。文档后续更新不改变这些代码输入。

专项跳过如实记录：`TestSearch100KPerformance` 明确留 T8 opt-in；`TestFlatIssueListOrders/sub_issues_count` 的天文级计数不用造行，数值表达式由 `TestIssueListNumericExpressions`、`TestListNumericCursor` 检验，真实有界计数由 `TestFlatIssueListCountOrder` 和 HTTP `TestIssueListAllOrders` 覆盖；不能把该 skip 当作规模验收。其余 package-level skip 是无测试包。

## 2. 权限、锁与拒绝

- 规则、operation 目录、矩阵和路由形成双向核对：`TestOperations`、`TestThePermissionMatrixCoversEveryOperation`、`TestMatrixViolationsCatchesEachGap`/`EachColumnGap`、`TestMemberIssueMatrixAccountTargets`，以及 search/drafts 专门矩阵。借别例的 issue/编号不能冒充目标列覆盖，分别有反例守卫。
- 工作区管理员仍遵守项目成员身份；非项目成员无写权，项目访客遵守 OwnOnly。AllowCreator 要求创建者仍有效且至少项目成员。先判断不可见，再返回禁止，工作区管理员身份不豁免项目访客的 OwnOnly 可见范围；有效项目访客同时是工作区管理员时，写角色按既有管理员规则通过（例如删除自己创建的工作项答 204），不借 AllowCreator 放宽。读列表、子项计数、关注、动态、搜索、个人三种关系使用同一可见性条件。
- `TestEachIssueWriteLocksItsProjectFirst` 按 operation 逐项观察真实请求 PID/事务；拒绝先结束、不取 gate/连接/行锁、不写表。普通写先不锁的预检，再 gate，再事务内排序项目锁及重判。持项目 N 时不取冲突的工作区锁，外键 KEY SHARE 兼容。
- 草稿 CRUD 为 W(S)→排序 P(S)→D(U)，不走项目写 gate；发布先 gate，再 W(S)→P(N)→D(U)，锁后项目变更答 409，不临时追拿第二个项目。关联删可在任一端获权，但锁和存在性核对都覆盖两端；预检看不到的项目不入锁集，等待期间新获可见性不能扩张原请求范围。
- 工作区/身份旧邀请锁排序、项目首锁、连续热写和跨项目排序在 T2 与本次 L20/L21 联合核对。池测试实际使用 pool=10、K=2；200 个同项目写中 2 个在库、64 个排队、134 个 503，另项目 20 个读受原阈值约束。T9 再记录分层耗时，当前不把断言通过冒充性能表。

## 3. 二十六交错逐项

完整原条款、文件 SHA 和运行次数保存在 `t3-interleaving-selection.json`；下面索引不删去设计增长要求。L02/L03、L04/L05 共用测试中的不同子例，但每条行为均有断言。

| 编号 | 具名测试（均在 `server/internal/bootstrap`） | 本次结果 |
|---|---|---|
| L01 | `TestTwoCreationsOnAProjectSerialize`（issue_interleaving_test.go）<br>`TestTwoChangesOfAWorkItemSerialize`（issue_interleaving_change_test.go） | 各 5 次通过 |
| L02 | `TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`（issue_interleaving_deletion_test.go） | 各 5 次通过 |
| L03 | `TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`（issue_interleaving_deletion_test.go） | 各 5 次通过 |
| L04 | `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（issue_interleaving_test.go） | 各 5 次通过 |
| L05 | `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（issue_interleaving_test.go） | 各 5 次通过 |
| L06 | `TestDeletingAStateAndPuttingAWorkItemInItSerialize`（issue_interleaving_catalog_test.go） | 各 5 次通过 |
| L07 | `TestDeletingALabelAndLabellingAWorkItemWithItSerialize`（issue_interleaving_catalog_test.go） | 各 5 次通过 |
| L08 | `TestArchivingAProjectAndCreatingAWorkItemSerialize`（issue_interleaving_test.go） | 各 5 次通过 |
| L09 | `TestTwoMovesThatWouldMakeALoopSerialize`（issue_interleaving_hierarchy_test.go）<br>`TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize`（issue_interleaving_hierarchy_test.go） | 各 5 次通过 |
| L10 | `TestOppositeRelationsOfTwoWorkItemsSerialize`（issue_interleaving_relations_test.go） | 各 5 次通过 |
| L11 | `TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize`（issue_interleaving_relations_test.go） | 各 5 次通过 |
| L12 | `TestArchivingAWorkItemAndMovingItSerialize`（issue_interleaving_archive_test.go） | 各 5 次通过 |
| L13 | `TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize`（issue_interleaving_hierarchy_test.go） | 各 5 次通过 |
| L14 | `TestCommentAndIssueDeletionSerialize`（issue_collaboration_interleaving_test.go）<br>`TestCommentAndReactionsSerialize`（issue_collaboration_interleaving_test.go）<br>`TestCommentWritesAndArchivalSerialize`（issue_collaboration_interleaving_test.go） | 各 5 次通过 |
| L15 | `TestDescriptionSavesSerialize`（issue_collaboration_clock_test.go）<br>`TestActivityCreatedAtDoesNotGoBack`（issue_collaboration_clock_test.go）<br>`TestCollaborationReadsClockUnderLock`（issue_collaboration_clock_test.go） | 各 5 次通过 |
| L16 | `TestDraftPublishAndProjectDeletionSerialize`（issue_draft_interleaving_test.go） | 各 5 次通过 |
| L17 | `TestDraftProjectMoveAndPublishSerialize`（issue_draft_interleaving_test.go）<br>`TestDraftPublishUsesPayloadAfterPatchWait`（issue_draft_publish_locks_test.go） | 各 5 次通过 |
| L18 | `TestAutoArchiveAndStateChangeSerialize`（issue_auto_archive_interleaving_test.go）<br>`TestAutoArchiveReadsClockAfterLock`（issue_auto_archive_interleaving_test.go）<br>`TestAutoArchiveUsesLockedPolicy`（issue_auto_archive_policy_interleaving_test.go）<br>`TestAutoArchiveUsesLockedPolicyOneProjectAtATime`（issue_auto_archive_policy_interleaving_test.go） | 各 5 次通过 |
| L19 | `TestLinkCrawlURLInterleaving`（issue_link_interleaving_test.go）<br>`TestLinkTitlePreservesConcurrentMetadata`（issue_link_interleaving_test.go）<br>`TestLinkCrawlAfterDeletion`（issue_link_interleaving_test.go） | 各 5 次通过 |
| L20 | `TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`（issue_pool_test.go）<br>`TestAHotProjectHoldsUpNoReadOfAnother`（issue_pool_test.go） | 各 5 次通过 |
| L21 | `TestEachIssueWriteLocksItsProjectFirst`（issue_write_locks_test.go）<br>`TestARelationLocksNoProjectItDoesNotReach`（issue_relation_locks_test.go）<br>`TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`（issue_relation_locks_test.go）<br>`TestDeleteRelationChecksBothLockedProjects`（issue_relation_locks_test.go） | 各 5 次通过 |
| L22 | `TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize`（issue_interleaving_test.go） | 各 5 次通过 |
| L23 | `TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize`（issue_interleaving_relations_test.go）<br>`TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize`（issue_interleaving_relations_test.go） | 各 5 次通过 |
| L24 | `TestDeletionInterleaving24`（issue_deletion_interleaving_test.go）<br>`TestIssueDeletionWindowAnswersAreStable`（issue_deletion_window_test.go） | 各 5 次通过 |
| L25 | `TestDeletionInterleaving25`（issue_deletion_interleaving_test.go） | 各 5 次通过 |
| L26 | `TestDeletingARelationAndItsOtherEndSerialize`（issue_interleaving_relations_test.go）<br>`TestTwoDeletionsOfARelationSerialize`（issue_interleaving_relations_test.go） | 各 5 次通过 |

## 4. D03–D07 的直接证据

| 标准 | 实际观察与测试 | 仍待部分 |
|---|---|---|
| D03 | apitest 公开错误码与 issue/project 双向登记；access 规则/operation/matrix 覆盖守卫；`TestDeletingAStateAWorkItemIsInIsRefused` 核 `project.state_in_use`，L06 核等待后重判 | T13 最终门禁/main 绑定 |
| D04 | 21 个合法码，`sort_order` 只升序；共享 11 场景 fixture 的升降精确 ID；有界子项/链接计数，NULL/名称/日期/UUID tie，数值精度和游标载荷、可见性、分组续页均由 adapter/domain/HTTP 测试观察 | T4 浏览器/TS 与 T8 10k/20k/100k 分层测量 |
| D05 | `TestInserterCommitRollback` 查实际 `river_job` 提交可见/回滚无行/插入失败全回滚；`TestInserterRequiresTransaction` 拒无事务；`TestRegistrationRowsSeeRequestJobs` 实际入队；两 worker 变异见下一节；`TestBusinessClockWorkerKeepsRiverRealTime`/`TestPurgeBusinessClock` 保持业务 clock 与 River 实时时间边界 | T4 clock 故事与 T13 最终 CI |
| D06 | `TestOutboundPinsEveryHop` 检全部 DNS 后才 dial、固定 literal IP、不二次 DNS、跳转重检；`TestOutboundLocalSnapshot` 拒本机快照，资源关闭/截止期/HTTPS 降级/Host/SNI/禁代理和凭据均有实际 transport 断言；危险分支零 dial | T6 I10 blocked 页面与最终三浏览器 |
| D07 | `TestPurgeCatalogCoversTablesAndIncomingKeys` 检 23 张 deleted_at 业务表和全部入边（含非清理表探针）；`TestPurgeCatalogRejectsEachGap` 拒缺项/不安全 FK；13 issue 表按子先父。`TestPurgeDeletedRunsWithRuntimeRole` 真 runtime 非 owner，模块 issue→project→workspace→identity，同 cutoff；61d 删/59d 留。删除窗口 15 读、22 写在任务暂停与完成后逐 body 等于不存在响应；L24/L25 逐行 At/By 不被覆盖 | T9 规模清理与 T13 I21/最终 CI |

迟到清理的已批准例外单列：`TestLateDeleteRowsAndPurgeRetry` 故意制造两个真实死锁 victim，原 River 任务第二次尝试收敛；`TestLateDeleteRowsAfterPurge` 和 `TestPurgeParentCascadesUnprocessedIssueRows` 检父 CASCADE 及 59d/活行边界。这不是本轮 26 交错出现的新 40P01，不删除其证据、不放宽重试合同，仍作为 M8 已知成本转交。

## 5. 变异候选及本轮已杀死项

T10 至少 15 个有效变异必须逐个具有业务断言失败、源码恢复和原态通过证据。下表保留原候选索引，未抽取的不能计入完成数；当前完整结果为 [15 项关键性质及 9 项补充](closeout-mutations.md)；编译失败、默认 skip、只找字符串不计。

| ID | 暂时破坏的性质 | 预期捕获位置 | 当前状态 |
|---|---|---|---|
| M01 | 工作区成员锁漏 FOR NO KEY UPDATE | `TestLockMemberWorkspaces` 的 acme/gamma FOR SHARE 探针 | 已杀死；`t2-mutant-remove-lock`，恢复 hash 见收据 |
| M02 | 周期任务没有 worker 仍允许构造 | `TestJobRegistration/periodic_worker_removed` | 已杀死；`t3-mutant-periodic-worker`，实际 New() 为 nil、期望缺 worker 错误 |
| M03 | 会投递的 kind 没有 worker 仍允许构造 | `TestJobRegistration/enqueued_worker_removed` | 已杀死；`t3-mutant-enqueued-worker`，实际 New() 为 nil、期望缺 worker 错误 |
| M04 | OwnOnly 放行其他创建者 | access 判定/权限矩阵 | 已杀死；见 closeout-mutations K07 |
| M05 | 创建者离开后仍 AllowCreator | access 判定/矩阵离开身份 | 已杀死；见 closeout-mutations K08 |
| M06 | 跳过锁下调用者重判 | L04/L05 | 候选未抽取，不计完成数；正向/交错证据见 §2–4 |
| M07 | 删除关联仅锁有权限的一端 | L21/L26 | 候选未抽取，不计完成数；正向/交错证据见 §2–4 |
| M08 | 描述活动合并改成新增 | L15/描述事件断言 | 已杀死；见 closeout-mutations K10 |
| M09 | link_count 纳入墓碑 | `TestFlatIssueListCountOrder/link_count_limit2` | 已杀死；见 closeout-mutations K11 |
| M10 | 数值游标丢精度或 UUID tie | `TestListNumericCursor`/fixture 精确 ID | 已杀死；见 closeout-mutations K12 |
| M11 | outbound 接受私网 DNS 答案 | `TestOutboundPinsEveryHop` 零危险 dial | 已杀死；见 closeout-mutations K13 |
| M12 | outbound 跳转不重新检查 | `TestOutboundPinsEveryHop` 每跳拒绝 | 候选未抽取，不计完成数；正向/交错证据见 §2–4 |
| M13 | purge cutoff 含等于边界 | `TestIssuePurgeAllTables` 严格边界 | 候选未抽取，不计完成数；正向/交错证据见 §2–4 |
| M14 | purge 遗漏一个 child 表 | catalog/全表清理反例 | 候选未抽取，不计完成数；正向/交错证据见 §2–4 |
| M15 | issue 列表旧会话迟到写入 | root store 账户切换兑现用例 | 已杀死；见 closeout-mutations K15 |
| M16 | copy await 后忽略账户代际 | T4 即将补的迟到成功/拒绝用例 | T4 真实生命周期反证已修复；不重复计入 K01–K15 |
| M17 | 归档动作保留旧工作区闭包 | T4 真实 store 路由重用用例 | T4 真实生命周期反证已修复；不重复计入 K01–K15 |
| M18 | draft publish 绕过 payload 锁后重读 | `TestDraftPublishUsesPayloadAfterPatchWait` | 候选未抽取，不计完成数；正向/交错证据见 §2–4 |

两 worker 源码都恢复到 `f50129aa848f5bdd4253077c15669874e4d5f3542e87ad7ffe7171049b210a22`；每次 diff、变异输入、失败原文、恢复 hash 分别保存。恢复后完整 registration 23 子例通过，接着 26 交错在同一原态输入跑完；没有把变异留在提交内。
