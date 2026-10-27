# M4/P9 物理清理、描述版本保留与异步删除实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **状态：** 准备稿，独立静态预检待进行。当前授权仅 spec/plan 与预检，暂不实施；下列命令、测试、变异、性能与提交均为未来实施步骤，全部未完成。

**Goal：** 交付23表60天物理清理、活描述版本最新20保留、项目/工作区删除后的issue侧分批任务，并证明窗口、锁序、审计、重试和目标规模。

**Architecture：** 四模块提供具体Purger值，bootstrap消费小接口并按issue→project→workspace→identity组成单任务。项目/工作区原事务发布事件，issue订阅者InsertTx投递；app复用既有删除编排/本模块SQL，River adapter只负责任务形状、超时、snooze及终态日志。保留任务按项目N串行，关联删除先锁候选后查另一端DeletedAt。

**Tech Stack：** 仓库锁定Go1.27/toolchain1.27.1、pgx5.11.0、River0.47.0、sqlc1.31.1、PostgreSQL18.6、Node24/pnpm11.10.0/Playwright；不加新依赖。

**Spec：** [P9 spec](../specs/P9-purge.md)，§2为目标接口/值的唯一说明。基线main `6b9d08d4821f1423c7bb9795557727c206b7880f`，CI `40520609987`全绿、P8归档清理已由主控核实；分支`codex/m4-p9`、worktree`.codex/worktress/codex/m4-p9`。

## Global Constraints

- 主目录HEAD永远main；开发仅P9 worktree。准备首提交phase=M4/P9由主控处理，执行T1先核对而不重复切换/延期例外。
- `plane/`、`refer/`只读；不动既有容器/Docker、不全局安装、不加未批准依赖、不建node_modules符号链接；不用stash/clean/reset --hard/force，不取钥匙串或其他凭据。
- 端口归消费方，模块不互相import、不跨模块JOIN；SQL仅sqlc、模块SQL无SET；app/domain不import pgx/platform/River。bootstrap不写清理业务SQL、不动态拼表名。
- Purger每表id键集、严格deleted_at<cutoff、每批pool上单语句隐式独立事务；只累加已成功批次。模块顺序issue→project→workspace→identity；不删账户、不重复会话清理。
- cleanup默认1440h/24h/1000，trim interval24h，delete batch1000/budget45s；interval≥1s，batch1..2147483647，budget>0且+5s不溢出。test interval2s。新配置必须有默认/环境/日志/拒绝证明。
- trim按last_saved_at DESC,id DESC留20；project/issue/version均活，归档仍处理；每项目JobTx+项目N，无活动/事件。软删版本仅purge。
- RowDeletions参数ctx/scope/id/at/by；project和workspace_drafts两scope。原请求At/By贯穿，不用worker时钟盖审计；workspace Changes只本次锁住项目，旧Old不归新请求。
- 关联id升序先锁后查DeletedAt，普通deleteIssue跳过已删另一端；job跳过更早删除端。Candidates/After推进与Written分离，Written=0不能结束扫描。
- delete预算用bootstrap传入time.Now的monotonic运行耗时，绝不使用业务Clock；app Complete=false由adapter映射JobSnooze(0)，Timeout=budget+5s，默认25次真实失败重试。
- 所有已知拒绝在gate/写锁前；普通issue写仍P(N)，draft CRUD为W(S)→P(S)→D(U)，publish为W(S)→P(N)→D(U)。同一锁编排补预检，不另造近似路径。
- 新公开码若实际必要，首次契约/PROBLEM_MESSAGES/en/zh-CN同Task并强制test-web；本计划预期无新公开码。生成物只make gen，不手改。
- 新逻辑文件约400行；每Task约350–1300手写行，约16Task/计划1500行以内，留修复余量。静态SQL、生成物及合理测试表沿既有例外；不以压缩安全测试满足尺寸。
- 容器套件严格串行、上一套退出再下一套。Go -p=1/-count=1，条件等待有期限，不sleep同步；零匹配/编译失败/skip/缓存不是行为绿。复用issueWorld、heldTx、serializeAt、inBothOrdersAt、pgtest.RowWaits、registrationRows与目录fixture。
- 每Task独立要求/质量审查；局部根因修复自行处理，改变既定架构/跨模块协议或意外高风险才按纪律提交已完成工作后上报。
- 详细中文多行提交，前缀feat/fix/docs(M4/P9)，只add精确路径；push使用`/usr/bin/git`。本轮仅准备不实现，准备提交由主控完成；以下Task提交指未来获授权实施。未来集成另按授权，--no-ff、准确main CI、归档哈希后清理。
- F在T4/T8/T12/T15以及最终产品修复后、任何产品检查点push之前执行；分支CI/mainCI各自绑定准确SHA，不能互代。handoff仅追加本期子项结果，status保持open至M4收尾。

## Review Focus

1. 最后批恰好完成、业务Clock偏移61天、前一关系批全属早删端：T7/T8/T12分别证明候选游标、不空snooze、运行时预算和多轮真实attempt不增。
2. 项目已删但issue/relationship仍活，活端调用者只能读无update：T7/T11两存储方向逐值404，已占gate/锁仍立即拒绝，不能窗口前403后404。
3. Old项目任务未完再删workspace，零项目workspace无项目draft：T10/T12/W3保留两个请求的精确审计，既有直接查库点也有限轮询。
4. last_saved_at同刻、归档project/issue、删除后仍留活版本：T5/T6保留20全序、不把归档当已删、不触软删候选，锁等待后重读。
5. 父清理CASCADE与>60天迟到job相遇、scale任务虽attempt=1但曾snooze：T6/T12接受原40P01重试边界，T14直接记录每次running与终态，并在正确存活邻居测读写。

## 文件职责、依赖与通用验证

I/P/W/A/B缩写、各文件职责/尺寸见spec§2.1；下文新增测试均明确标“新”，现有名已从基线源码核对。若实施更名先同步本计划/命令，不能空regex当绿。

依赖：T1→T2→T3→T4；T5→T6；T7→T8→T9→T10→T11→T12；T13依赖T4/T10；T14依赖T12；T15→T16。T5和T7读取T1配置。实际按编号推进，每个中间提交可运行；只有只读独评可以并行，禁止两套数据库测试并跑。

**G（每Task通用收尾）：** 写列出的测试与精确断言→聚焦真实红→最小实现→同命令绿→该Task生产变异红→恢复字节/hash后再绿→影响包回归、`git diff --check`→独立要求/质量评审及修复复审→精确路径提交。新增接口的编译红仅证明缺接口；对应性质必须另有运行红。记录准确HEAD/dirty源码hash、argv/cwd、raw、RUN/skip/exit、变异和恢复hash。每Task提交标题见末步，正文写原因/边界/验证，不`git add .`。

**F（完整本地检查点）：** 生成物先按Task生成并提交，再严格串行：

```sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
```

七项退出0，web证明非缓存，E2E含本次环境要求的浏览器；记录各区域计数与skip理由、准确tested SHA/工作树和原件。基础设施失败、真实断言失败、缺浏览器分别记录；只有新改动/真实失败/未解风险才扩大或重跑。F是实施动作，准备期不执行。

## Task 1：索引、配置与PAT独立批清理

**Files：** 新`server/migrations/sql/00030_identity_purge_indexes.sql`、`00031_workspace_purge_indexes.sql`、`00032_project_purge_indexes.sql`、`00033_issue_purge_indexes.sql`；改`server/sqlc.yaml`、`server/internal/platform/config/config.go`、validate.go、load_test.go、validate_test.go；新config/purge_test.go；改`server/configs/config.yaml`、config.test.yaml；新A/app/purge.go、purge_test.go、adapter/postgres/purge.go、purge_test.go、queries/purge.sql；改A/provide.go；生成四模块gen。预计500–900行。

**Interfaces：** 产出spec§2.2本模块PurgeBatch/PurgeStep/Purger/NewPurger/Tables；adapter `PurgeAPITokens(context.Context,app.PurgeBatch)([]uuid.UUID,error)`；A.Provided.Purger具体值，Provide(pool)不变。产出§2.3六字段与校验。

- [ ] **测试：** 新`TestPurgeConfiguration`钉默认1440h/24h/1000/24h/1000/45s、六环境变量、日志；interval=999ms、batch=0/2147483648、budget=0/MaxInt64拒绝，1s/1/2147483647可接受。新app`TestPurgerAdvancesOnlyCommittedBatches`记录步骤顺序/After、先成功2行后原错返回`count=2`且后续不调用；乱序返回ids仍取max，空结束、取消不新批。
- [ ] **PG测试：** 新`TestAPITokenPurgeBatches`准备61d/59d/精确cutoff/活token及同账户其他行；batch=1只删61d。新`TestPurgeDoesNotJoinCallerTransaction`在外层Tx ctx调用后回滚，另连接仍见前批已删除；故障批不计、users/profiles/auth_sessions不变。断言核心：`deleted==oldCount && atCutoffExists && recentExists && liveExists && accountsUnchanged`。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/platform/config -run '^TestPurgeConfiguration$'`；随后app/PG分别`-run '^Test(PurgerAdvancesOnlyCommittedBatches|APITokenPurgeBatches|PurgeDoesNotJoinCallerTransaction)$'`，新增类型/SQL未有先红；记非零RUN。
- [ ] **实现与绿：** 四迁移逐表索引与各自sqlc schema；PAT静态DELETE严格<、id键集，adapter直接gen.New(pool)单语句隐式事务。`make gen-go`，重复聚焦并`go -C server test -p=1 -count=1 ./migrations ./internal/archtest ./internal/platform/config ./internal/modules/identity/...`绿。
- [ ] **变异/G：** M1 `<`→`<=`在PG精确cutoff红；M2改用s.queries(ctx)在外层回滚反例红；M3计入失败批/after取首id在app红。恢复绿，提交`feat(M4/P9): 建立清理索引配置并按独立批次回收令牌`。

## Task 2：project与workspace清理者

**Files：** 新P/W的app/purge.go、purge_test.go、adapter/postgres/purge.go、purge_test.go、queries/purge.sql；改P/W/module.go；生成对应gen。预计450–850行。

**Interfaces：** 每模块独立的spec§2.2Purge类型；P.Provided.Purger/W.Provided.Purger。adapter新`PurgeProjectUserProperties/PurgeProjectMembers/PurgeLabels/PurgeStates/PurgeProjects`及`PurgeWorkspaceUserProperties/PurgeWorkspaceMemberInvites/PurgeWorkspaceMembers/PurgeWorkspaces`，均`(context.Context,app.PurgeBatch)([]uuid.UUID,error)`。

- [ ] **测试：** 两模块新`TestModulePurgeOrderAndCutoff`，每表61d/59d/边界/活对照，id与时间反序，batch=1核多批；labels两层自引用、workspace邀请和成员偏好有非空行。父链不具备清理资格的recent必须保留；旧父CASCADE另用例允许带走近期子行，返回数只算直接RETURNING。
- [ ] **红：** 依次`go -C server test -p=1 -count=1 ./internal/modules/project/adapter/postgres -run '^TestModulePurgeOrderAndCutoff$'`和workspace同命令；app两包跑`'^TestPurger'`记录缺清理者红。
- [ ] **实现：** 严格按spec表序绑定本模块静态sqlc函数，错误/取消/独立批规则与T1相同，无新共享清理包；Provide仍只pool。
- [ ] **绿：** `make gen-go`后重复聚焦；`go -C server test -p=1 -count=1 ./internal/modules/project/... ./internal/modules/workspace/... ./internal/archtest`通过。
- [ ] **变异/G：** M4父表放到子表前，用直接删除计数/逐步观察抓住CASCADE抢删；M5遗漏workspace_member_invites或把所有表绑定同一查询，真实每表非空fixture红；恢复后提交`feat(M4/P9): 按模块回收项目与工作区的过期软删行`。

## Task 3：issue清理与完整目录守卫

**Files：** 新I/app/purge.go、purge_test.go、adapter/postgres/purge.go、purge_test.go、queries/purge.sql；改I/module.go；新B/purge_catalog_test.go、`server/migrations/purge_indexes_test.go`；消费既有workspace_deletion_catalog_test.go；生成I/gen。预计700–1100行。

**Interfaces：** I.Provided.Purger；13个`Purge<Table>(context.Context,app.PurgeBatch)([]uuid.UUID,error)`后缀/顺序见spec§2.2。B测试消费四值的Tables()，生产B.Purger接口仍留T4定义。

- [ ] **测试：** 新`TestIssuePurgeAllTables`准备带活动/comment/comment_reaction、多层issue、双向关系、links、mentions、versions、两类draft的61d与未到期对照；13表逐个非空再删。`TestPurgeCatalogCoversTablesAndIncomingKeys`从真实catalog取23表与全部入边，含comment_reactions二级边、非purge源表入边，动作仅c/n；`TestPurgeCatalogRejectsEachGap`对删掉清理者、漏FK、NO ACTION逐一断言非空违规。`TestPurgeIndexes`核23精确索引定义及模块schema。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^TestIssuePurgeAllTables$'`；随后bootstrap `-run '^TestPurgeCatalog'`、migrations `-run '^TestPurgeIndexes$'`。目录不能以两个手写列表相等自证。
- [ ] **实现：** 13静态sqlc单条DELETE、局部步骤与Tables副本；复用目录查询/foreignKey结构，遍历全部被清理目标的入边，不把existing comment_reactions例外扩大。直接count与CASCADE计数分开。
- [ ] **绿：** `make gen-go`；重复三聚焦，`go -C server test -p=1 -count=1 ./migrations ./internal/modules/issue/adapter/postgres ./internal/archtest`通过。
- [ ] **变异/G：** M6去deleted谓词，M7去id>after，M8丢comment_reactions步骤/入边之一；分别在PG行为、记录keyset参数、catalog反例红，恢复绿。提交`feat(M4/P9): 回收完整工作项表并守住清理与外键目录`。

## Task 4：单一cleanup周期任务与运行角色（F）

**Files：** 新B/purge.go、purge_test.go、purge_integration_test.go；改B/app.go、all_periodic_jobs_test.go、business_clock_jobs_test.go、runtime_role_test.go；消费platform/jobs既有登记接口。预计600–1000行。

**Interfaces：** B.Purger、newPurgeDeleted、Execute、purgeJob按spec§2.3；固定kind cleanup.purge_deleted/periodic/RunOnStart，使用四Provided.Purger值和业务Clock。仅purge的InsertOpts加`river.UniqueOpts{ByState: []rivertype.JobState{rivertype.JobStateAvailable,rivertype.JobStatePending,rivertype.JobStateRunning,rivertype.JobStateRetryable,rivertype.JobStateScheduled}}`；不设ByArgs/ByPeriod、不含Completed、不改其他任务。

- [ ] **测试：** 新`TestPurgeDeletedUsesOneCutoffAndModuleOrder`假Clock调用恰1，顺序issue/project/workspace/identity，第二模块失败后total仅已提交数且第三不调用；`TestPurgeDeletedRunsWithRuntimeRole`用真实runtime处理PAT和带活动/comment issue、标签/project/workspace，61d删/59d留。扩`TestAllPeriodicJobsRun`为此时3种周期各startup/重复/队列；新`TestPurgeBusinessClock`偏移61d改变cutoff而调度仍真实时间。
- [ ] **重叠测试：** 新`TestPurgePeriodicRunsDoNotOverlap`用真实River运行purgeJob，阻住首轮跨至少两个周期（以独立控制周期任务的完成轮数为条件证据），断言`activeJobs==1 && maxConcurrent==1`；释放completed后出现新job id。另一分支令首轮真实失败进入retryable，再在隔离库把该任务scheduled_at设到两个周期之后，跨周期仍只有该active任务；到期重试仍按同job运行。去UniqueOpts必须真实观察第二active/并发而红，不能只断言配置结构。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(PurgeDeletedUsesOneCutoffAndModuleOrder|PurgeDeletedRunsWithRuntimeRole|PurgeBusinessClock|PurgePeriodicRunsDoNotOverlap|AllPeriodicJobsRun)$'`，无job/无删除/跨周期重叠红；runtime不是owner连接。
- [ ] **实现：** 单个bootstrap聚合worker，只委托Execute；append到jobs.New，不增加四并发purge任务，无SQL/模块表名派发；活动状态唯一性阻止同kind不同轮次及旧重试交叠。失败保留原错误链，默认River期限不改。
- [ ] **绿：** 重复聚焦，`go -C server test -p=1 -count=1 ./internal/platform/jobs ./internal/bootstrap ./internal/archtest`。M9交换模块顺序、M10用time.Now替业务Clock、M11漏periodic worker、M11b去purge UniqueOpts分别红→恢复。
- [ ] **G/F：** 完整七门禁，保存准确提交/原件再push检查点；提交`feat(M4/P9): 组合单一清理任务并验证运行角色与业务时钟`。

## Task 5：最新20描述版本及周期接线

**Files：** 新I/app/trim_versions.go、trim_versions_test.go、adapter/postgres/trim_versions.go、trim_versions_test.go、queries/trim_versions.sql、adapter/river/trim_versions.go、trim_versions_test.go；改I/module.go、module_jobs.go；P/module.go、adapter/postgres/issue_ports.go、queries/issue_ports.sql；B/ports.go、app.go、all_periodic_jobs_test.go；新B/issue_trim_jobs_test.go；生成对应gen。预计650–1100行。

**Interfaces：** spec§2.4 RetentionProjects.UndeletedProjectIDs、P.AllUndeletedIssueProjectIDs、DescriptionVersionTrimmer.TrimDescriptionVersions、TrimVersionsDeps/New/Execute、TrimVersionsJob。JobTx是既有原始PG错误事务，不能套busyTx。

- [ ] **测试：** 新`TestTrimDescriptionVersionsKeepsLatestTwenty`为同issue建23活版+软删版，last_saved_at与created_at反序、21/20边界相同时用反序id；断言剩余精确id集合20，soft-deleted原样。归档project/issue也删到20；已删project、已删issue、软删version不碰。新app`TestTrimVersionsCommitsPerProject`第一项目成功、第二失败，只返回第一计数；等待Lock后found=false跳过；Events/Clock没有依赖。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^TestTrimDescriptionVersionsKeepsLatestTwenty$'`；app `'^TestTrimVersions'`；bootstrap新`'^TestTrimVersionsJobRuns$'`。
- [ ] **实现：** 全量未删项目目录含归档/无策略/无当前成员；按id遍历，N锁下每项目单SQL row_number>20，无分批/活动/事件。module/river/周期同时接上，扩AllPeriodicJobsRun与runtime测试为全部4种。
- [ ] **绿/变异：** `make gen-go`；重复聚焦及`go -C server test -p=1 -count=1 ./internal/modules/issue/... ./internal/modules/project/adapter/postgres ./internal/bootstrap -run 'Test(Trim|AllPeriodicJobsRun)'`；M12改按created_at/漏id，M13排除archived或复用ArchivePolicies，M14漏live issue/version谓词，真实PG红→恢复。
- [ ] **G：** 提交`feat(M4/P9): 按项目锁保留最新二十个活描述版本`。

## Task 6：保留、请求与物理清理的边界

**Files：** 新B/issue_retention_interleaving_test.go、purge_late_deletion_test.go；必要根因修改回T4/T5文件。预计400–750行。

**Interfaces：** 消费T4 purgeDeleted.Execute、T5 TrimVersions.Execute和既有issue update/auto-archive/project delete；只新增测试场景，不建第二套生产编排。晚delete_rows实际worker在T12补齐。

- [ ] **测试：** 新`TestTrimVersionsSerializesWithDescriptionWrites`、`TestTrimVersionsSerializesWithProjectDeletion`用真实holder/PID，两顺序、相同timestamp tie；更新与trim争同P(N)，删除先提交则trim跳过。新`TestTrimAndPurgeUseDisjointVersionSets`同时安排活版本超20/软删61d/软删59d，核精确集合及审计不变；归档不能让保留消失。
- [ ] **测试：** 新`TestPurgeParentCascadesUnprocessedIssueRows`先删project或workspace后保留活issue侧行（隔离测试存储状态），purge旧父的CASCADE消除所有FK子链；用户账户及存活邻居不变。此Task不宣称迟到worker已验证。
- [ ] **红/绿：** `go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(TrimVersionsSerializesWithDescriptionWrites|TrimVersionsSerializesWithProjectDeletion|TrimAndPurgeUseDisjointVersionSets|PurgeParentCascadesUnprocessedIssueRows)$'`；若既有实现直接绿，先记录，再用M15去项目N/M16软删版纳入trim获得性质红，不把无缺陷初绿伪造为初红。
- [ ] **实现：** 仅修上述实际暴露缺陷；恢复变异后同命令绿，相关interleaving `-race -count=5`绿；不要求>60d父CASCADE交叠永无40P01，T12单独验证允许重试。
- [ ] **G：** 提交`test(M4/P9): 证明版本保留与请求清理的集合和锁边界`。

## Task 7：关系先删端归属与集中预检

**Files：** 新I/app/relation_deletions.go、relation_deletions_test.go；改I/app/delete_issue.go、delete_project_rows.go、ports.go、lock_either.go、delete_issue_test.go、lock_test.go及对应fakes；新I/adapter/postgres/relation_deletions.go、relation_deletions_test.go、queries/relation_deletions.sql；改delete_issue.go、queries/delete_issue.sql；P/module.go、adapter/postgres/issue_ports.go、queries/issue_ports.sql；B/ports.go；新B/issue_relation_ownership_test.go；改B/issue_relation_locks_test.go。预计800–1200行。

**Interfaces：** spec§2.5 DeletedAt/DeletedIssueProjects、RelationDeletionRow/Rows、BatchProgress/NewRelationDeletions/ForProject/ForIssues；DeleteIssueDeps.Relations。`Locks.eitherOf(ctx,actor,w)(scope,error)`签名不变，新增两端去重项目Find预检；eitherUnder已有map存在性检查保持。

- [ ] **测试：** 新`TestRelationDeletionOwnership`用跨项目两方向、两种id次序、At=T0/T1及不同删除者：live deleteIssue跳过已删另一端，job只跳过更早另一端，同At可写。新`TestRelationDeletionCursorUsesCandidates`首批2个低id全跳过，后续高id1个归本端：`Candidates=2,Written=0,After=lowMax`仍继续，最后全skip/混合/零候选也正确；DeletedAt错误必须原样传播不当空。
- [ ] **预检测试：** 新`TestDeleteRelationDeletedEndpointPrecheck`在P软删但关系/issue活的窗口，Q调用者分别可update、只可read无update；两存储方向都与任务后/未知relation逐值404。持满两个gate并锁项目表，拒绝仍无gate/无写锁；Find返回原错则原错，不降成404。新`TestDeleteRelationChecksBothLockedProjects`让预检后P删除、LockMany缺P，授权调用前404；既有`TestLockIssueProjectsLeavesOutAProjectDeletedWhileItWaited`继续跑。
- [ ] **红：** app/PG分别`go -C server test -p=1 -count=1 ./internal/modules/issue/app ./internal/modules/issue/adapter/postgres -run '^Test(RelationDeletionOwnership|RelationDeletionCursorUsesCandidates)'`；bootstrap `'^Test(DeleteRelationDeletedEndpointPrecheck|DeleteRelationChecksBothLockedProjects)$'`。预检旧实现的403/入gate是行为红。
- [ ] **实现/绿：** 同事务relation.id升序锁→DeletedAt→仅锁住ids更新，模块SQL不JOIN projects；eitherOf在decideEither之前确认两端项目仍存在，不绕过统一Locks。`make gen-go`，重复聚焦及issue/app/PG/projectPG/arch回归。过渡同步Cascade仍使用原关系批SQL直到T10；不能为Provide(pool)回填ProjectGate。
- [ ] **变异/G：** M17 Written=0即结束，M18先更新后读DeletedAt/忽略早删，M19去eitherOf一端Find，M20去eitherUnder存在性判断；各真实边界红→恢复。提交`fix(M4/P9): 固定关系删除归属并提前拒绝已删除端点`。

## Task 8：分批任务、预算与完整声明接线（F）

**Files：** 改I/app/delete_project_rows.go；新I/app/delete_rows.go、delete_rows_test.go、delete_rows_on_deletion.go、delete_rows_on_deletion_test.go；新I/adapter/postgres/deletion_remaining.go、deletion_remaining_test.go、queries/deletion_remaining.sql；新I/adapter/river/delete_rows.go、delete_rows_test.go、row_deletions.go、row_deletions_test.go；改I/module.go、module_jobs.go、module_events.go；新P/W/module_events.go；改B/app.go、events_test.go、issue_link_jobs_test.go、request_jobs_test.go；新B/issue_delete_jobs_test.go；对应生成。预计900–1300行。

**Interfaces：** spec§2.6/2.7 DeleteRowsDeps/Execute/Result、RemainingRows/ProjectRemainder/RowStep、RowDeletions、DeleteRowsArgs/UseCase/Worker/Job。I.Deps新增DeleteBatch/DeleteBudget/RuntimeNow，module构造时接JobTx与Relations；P/W.EventTypes在本Task声明，发布用例仍T9/T10。

- [ ] **测试：** 新app`TestDeleteRowsCommitsEachBatch`核13表顺序、每批一个Tx、提交后推进、部分成功原错、重做不改旧戳；`TestDeleteRowsBudgetUsesRuntimeElapsed`业务At偏移±61d不影响45s预算；fake RuntimeNow在批边界越线且有候选Complete=false，无候选Complete=true；pending错误原样返回。新`TestDeletionRemainingUsesCurrentCursor`当前After过滤、后续表从头、完成表不重扫、关系只看候选、workspace只project_id NULL。
- [ ] **测试：** 新`TestDeleteRowsWorkerMapsBudgetAndTimeout`断言`Timeout==50*time.Second`、Complete=false为JobSnoozeError、普通错误errors.Is保持、最后失败error日志且snooze不记耗尽。新`TestDeleteRowsOnDeletion`核project1项、workspace按Changes n+1、零项目1、旧Old不在args、At/By精确、nil actor/坏shape不静默；`TestRowDeletionsEnqueue`断言scope/id/at/by JSON、default、实际max_attempts25。扩`TestRegistrationRowsSeeRequestJobs`识别此kind/替换payload/拒绝不留任务。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/modules/issue/app ./internal/modules/issue/adapter/river ./internal/modules/issue/adapter/postgres -run '^Test(DeleteRows|DeletionRemaining|RowDeletions)'`；bootstrap新`TestIssueDeleteSubscriberRegistration`与既有`TestIssueLinkSubscriberRegistration`、`TestRegistrationRowsSeeRequestJobs`。缺订阅/worker红而非空regex。
- [ ] **实现/绿：** 复用rowSteps统一表序和扫描编排，关系步骤以函数注入，async传ForProject；同步Cascade过渡函数仍用旧批SQL。耗尽时才静态pending探测；B传time.Now，不传businessClock。**同时**P/W EventTypes、dispatcher类型并集、两个subscriber、Jobs/EnqueuedKinds；把events_test.go旧future-event预期改为已声明并另用真正unknown类型测试。重复聚焦，`make gen-go`，issue/...、project/workspace app、bootstrap、jobs/arch回归绿。
- [ ] **变异/G/F：** M21业务Clock预算、M22Complete=false返回nil或普通error、M23漏EnqueuedKinds/worker、M24把workspace当一个project scope或漏无项目草稿；红→恢复。完整F，提交`feat(M4/P9): 接入有预算的删除任务与事务订阅声明`。本Task结尾即可启动，不等发布点才修登记。

## Task 9：单项目请求切换与旧断言迁移

**Files：** 改P/app/delete_project.go、delete_project_test.go、module.go及对应fake；新B/issue_project_deletion_events_test.go；改B/project_deletion_test.go、issue_interleaving_relations_test.go、issue_interleaving_deletion_test.go、issue_draft_interleaving_test.go及实际跨workspace关系测试调用点；改`api/modules/project.yaml`、`e2e/fixtures/assert/project.ts`；生成api/dist、Go HTTP与TS。预计400–800行。

**Interfaces：** P.NewDeleteProject第二参数改shared.EventPublisher，P.Deps.Events；project.deleted精确形状见spec§2.8。expectProjectDeleted签名保持，只issue目录行bounded polling。

- [ ] **测试：** 新`TestProjectDeletionPublishesAndEnqueuesAtomically`挂起job，204时M3行已同戳、issue侧仍活且API404；唯一project任务At/By来自project行、Changes空。订阅/InsertTx故障逐表snapshot及registrationRows完全回滚；未授权/已删404零额外任务。既有成功/拒绝catalog全部保留。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(ProjectDeletionPublishesAndEnqueuesAtomically|DeletingARelatedWorkItemAndTheOtherEndsProjectSerialize|RelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize|AWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize|DraftPublishAndProjectDeletionSerialize)$'`；先新测试见“请求仍同步删issue”的红。
- [ ] **实现：** M3事务后发布、不再调用IssueCascade.DeleteProjectIssues；事件错误回滚。**同Task迁移交错11/23及跨workspace关系的旧同步stamp断言**：请求锁顺序仍观察原P，后台关系stamp另用worker barrier/限时条件后核原请求戳，不能继续等待project请求持relation锁；不等T12补救。P4 helper仅issue侧轮询，M3行立即核。
- [ ] **绿/变异：** `make gen`，重复聚焦，project/...、bootstrap回归；build后项目P4故事。M25Publish移出Tx/M26用worker时间或漏project事件在事务/审计反例红→恢复；现有交错11/23新异步断言必须通过。
- [ ] **G：** 提交`feat(M4/P9): 将项目删除的工作项行交给原事务投递任务`。

## Task 10：工作区切换、CLI清理与全部W3异步点

**Files：** 改W/app/delete_workspace.go、delete_workspace_test.go、ports.go、module.go；P/app/cascade.go、cascade_test.go、issue_ports.go、module.go及fake；I/app/cascade.go、cascade_test.go、delete_project_rows.go、adapter/postgres/delete_rows.go、queries/delete_rows.sql、module.go；B/users.go、workspace_deletion_test.go、workspace_deletion_catalog_test.go、issue_interleaving_relations_test.go；新B/issue_workspace_deletion_events_test.go；改`server/internal/archtest/composition_test.go`、`api/modules/workspace.yaml`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`；生成相关产物。预计750–1200行。

**Interfaces：** W.NewDeleteWorkspace增加Events（Clock之后、Logger之前）；P/W.Deps.Events；P.NewCascade去issues，CascadeDeps仅Pool；DeleteWorkspaceProjects(ctx,workspaceID,by,now)；I.NewCascade(StateLabelUses)。删除两个旧IssueCascade方法和旧直写project relation接口/SQL，最终关系只T7编排。

- [ ] **测试：** 新`TestWorkspaceDeletionPublishesOnlyLockedProjects`造Old已删任务挂起、当前P/Q、无项目draft、另workspace；workspace事件Changes恰P/Q升序，任务P/Q各1+workspace_drafts1，Old既有task不替换，不逐P再发事件。零项目workspace仍1draft task；邀请同步撤销，失败subscriber整事务零变化。扩composition守卫Users不再到issue.Provide且不可达issue.New/InsertTx/jobs/dispatcher。
- [ ] **红：** `go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(WorkspaceDeletionPublishesOnlyLockedProjects|RelatingAcrossProjectsAndDeletingTheWorkspaceSerialize)$'`；`go -C server test -p=1 -count=1 ./internal/archtest -run '^TestCommandsComposeNoServerAndNoJobs$'`。
- [ ] **实现：** 锁住项目集合保留在workspace用例，唯一clock值给cascade和event；原同步M3邀请/成员/偏好/项目仍在。删除过渡接口、SQL、批常量和CLI注入/import，StateInUse/DetachLabel保持；迁移workspace关系同步断言。所有调用点编译不留过渡空桩。
- [ ] **E2E/绿：** 两helper只issue侧有限轮询；W3两处另改：Old DELETE后立即oldDraftAudit改取同步Old项目戳作为预期、最终轮询草稿；零项目workspace故事直接`Promise.all(tables.map(db.query))`的issue侧改有限轮询，M3侧仍立即断言。不能等Old job全完再删workspace。`make gen`后Go相关包回归，`make build VERSION=0.1.0-dev`，`NERVE_VERSION=0.1.0-dev pnpm -C e2e exec playwright test stories/project/p4-archive.spec.ts stories/workspace/w2-landing.spec.ts stories/workspace/w3-workspace-settings.spec.ts`；与Makefile实际默认值一致，不复用错误binary。
- [ ] **变异/G：** M27Changes事后全workspace扫描包含Old，M28漏workspace_drafts/project_id NULL，M29漏Old戳或零项目直接断言迁移；真实组合/W3红→恢复，提交`feat(M4/P9): 按本次项目投递工作区删除并移除同步过渡依赖`。

## Task 11：37操作完整窗口与空创建者

**Files：** 新B/issue_deletion_window_test.go、issue_deletion_window_cases_test.go、issue_null_creator_test.go；改既有permission_matrix_coverage_test.go、issue_write_locks_test.go、issue_draft_lock_shapes_test.go及受影响relations/list/search tests；仅实际失败才改原用例。预计700–1200行。

**Interfaces：** 测试case目录operationId集合与真实issue tag契约双向相等；复用T8 worker控制/既有call和response验证，不新增生产测试API。生产接口不扩。

- [ ] **测试：** 新`TestDeletionWindowCoversEveryIssueOperation`断言`len(reads)==15;len(writes)==22`及精确37集合，删任case失败。新`TestIssueDeletionWindowAnswersAreStable`依spec§2.9逐项project/workspace两窗口：挂起→采集→执行完成→重问，码/体/稳定头/数量/顺序/422位置相等；404再比未知资源。反向关联、活端read-only403陷阱、createDraft显式project422、旧draft404、活workspace无projectdraft正常、workspace删后404不能互混。
- [ ] **测试：** 分页平铺/组/子组、listMember/search、description/comment/link/reactions各入口；前页游标在项目删除后续页重新过滤；活源活动引用按Historical规则不因目标deleted_at变化消失。新`TestNullCreatorRemainsHiddenAndUnowned`API建issue后隔离存储把creator=NULL；OwnOnly(含adminGuest)读写404，与未知逐值相同，普通member不能以AllowCreator删除（403），admin允许；不删除账户。
- [ ] **红/实现：** `go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(DeletionWindowCoversEveryIssueOperation|IssueDeletionWindowAnswersAreStable|NullCreatorRemainsHiddenAndUnowned)$'`；若某路径直接绿，先记录，再对应生产变异得性质红。修复仅进统一Locks/目录/原查询，不能靠test过滤响应抹差异。
- [ ] **绿/变异：** 同聚焦、完整permission matrix/写锁目录回归；M30漏一个case、M31删除项目仍入Visible/草稿目录、M32活动错误加deleted过滤、M33NULL creator视caller所有，在组合层分别红→恢复。已有18+3+1=22写完整性、预检零锁及P/W/D形状保持。
- [ ] **G：** 提交`test(M4/P9): 覆盖全部操作的删除窗口及空创建者权限`。

## Task 12：交错、真实snooze、失败耗尽与迟到任务（F）

**Files：** 新B/issue_deletion_interleaving_test.go、issue_delete_job_lifecycle_test.go、job_state_observer_test.go；改B/purge_late_deletion_test.go、issue_interleaving_deletion_test.go、issue_interleaving_relations_test.go、issue_draft_interleaving_test.go；必要根因修复回原文件。预计800–1300行。

**Interfaces：** 新测试帮手`observeJobStates(t testing.TB,pool *pgxpool.Pool)`安装隔离test schema的river_job迁移记录；`jobTransitions(t testing.TB,pool *pgxpool.Pool,id int64) []jobTransition`，jobTransition={JobID int64;From,To string;Attempt int;At time.Time}。只_test.go，T14复用；不扩生产Runner/client接口。

- [ ] **测试：** 新`TestDeletionInterleaving24`、`TestDeletionInterleaving25`覆盖spec§2.9两次序/UUID方向/不同actor/低id跳过高id后续/同At两个job；真PID关系锁等待，逐行deleted_at/updated_at/updated_by_id。旧2/3/11/16/23全部运行；新`TestDeleteRelationChecksBothLockedProjects`与既有LockMany等待删项目强化P3交接。
- [ ] **测试：** 新`TestDeleteRowsRiverSnoozeDoesNotSpendAttempts`小预算+可控RuntimeNow强制≥2轮，observer证明多次running→available/scheduled后completed、attempt保持1、`metadata.snoozes>0`、旧行戳不变，业务clock±61d同结果。`TestDeleteRowsRiverRetriesAndExhausts`局部成功后原DB错误重试；末次任务预置24再真实25 discarded，日志一次exhausted且窗口仍隐藏。`TestLateDeleteRowsAfterPurge`先purge后job零行完成；`TestLateDeleteRowsAndPurgeRetry`制造真实父CASCADE/批写交叠，允许40P01一方回滚，真实River重试后目录空，无吞错/改HTTP503。
- [ ] **红/绿：** `go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(DeletionInterleaving24|DeletionInterleaving25|DeleteRowsRiverSnoozeDoesNotSpendAttempts|DeleteRowsRiverRetriesAndExhausts|LateDeleteRowsAfterPurge|LateDeleteRowsAndPurgeRetry)$'`。等待用observer/pgtest.RowWaits与期限；只返回snooze的adapter单测不替代真实调度。
- [ ] **变异/race：** M34去关系id锁序、M35重写Old审计、M36snooze改普通error、M37吞40P01，逐一运行红→恢复。普通交错组（包括24/25）`go -C server test -p=1 -race -count=5 ./internal/bootstrap -run '^Test(DeletionInterleaving24|DeletionInterleaving25|AWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize|DeletingARelatedWorkItemAndTheOtherEndsProjectSerialize|RelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize|RelatingAcrossProjectsAndDeletingTheWorkspaceSerialize|DraftPublishAndProjectDeletionSerialize|DeleteRelationChecksBothLockedProjects)$'`通过且无40P01；late-purge允许40P01测试单列，不把预期重试混入“无死锁”。
- [ ] **G/F：** 完整F，提交`test(M4/P9): 证明删除交错与真实任务预算重试生命周期`。

## Task 13：I21与删除故事的最终验证

**Files：** 新`e2e/stories/issue/i21-purge.spec.ts`；改`e2e/fixtures/auto-archive.ts`、clock.ts、assert/project.ts、assert/workspace.ts和i16-auto-archive.spec.ts的必要helper调用；P4/W2/W3保持API furnish。预计400–800行。

**Interfaces：** 复用`createClockWorld({databaseName,logFile,now?}):Promise<ClockWorld>`；将既有完成查询抽为`completedPeriodicJobs(db,kind,watermark,startedAt,id?)`，返回形状沿completedAutoArchives，不造新fixture；archiveWatermark或同一局部watermark helper继续返回微秒text。

- [ ] **测试：** I21通过正常进程API造旧issue全部协作行/PAT/label/project/workspace并删除；+61d单进程替换后API造recent和活对照，最近写后取clock_timestamp水位。断言旧父子全物理消失、recent父链与活链仍在、用户不删、登录重新建立；有限等待fresh attempted_at>watermark且completed/本代finalized门。增加crossing-attempt反例，早开始晚结束不算新轮次。
- [ ] **红：** 先build本Task二进制，聚焦I21应在缺故事/断言或M38错误完成门时红；不把无浏览器/未build当性质红。执行`pnpm -C e2e exec playwright test stories/issue/i21-purge.spec.ts stories/issue/i16-auto-archive.spec.ts`时传入Makefile build相同NERVE_VERSION。
- [ ] **实现：** 只抽既有helper参数、增加故事，前置全API，DB只读断言/时钟水位；同库任一时刻仅一nerve，不用其他worker db/nerveWith混库。不用SQL回填业务年龄。
- [ ] **绿/变异：** I21/I16与P4/W2/W3全部过；M38只看finalized_at、M39漏PAT或活动清理，生成/build对应真实binary后故事红→恢复源码/生成/build再绿；目录非空/原戳/邻居仍核。
- [ ] **G：** 提交`test(M4/P9): 以时钟偏移验证物理清理和删除故事`。

## Task 14：现有scale升级为SP2并记录真实执行次数

**Files：** 改B/issue_scale_test.go；新B/issue_scale_fixture_test.go、issue_scale_evidence_test.go；复用job_state_observer_test.go；本spec附录与ignored raw。预计650–1100行。

**Interfaces：** 保留现有`TestTheDeletionsAtTheScaleTarget`与环境开关`NERVE_MEASURE_SCALE`；保留issueWorld/seedAtScale职责并拆文件，新增证据值只_test.go。不新增生产hook/队列/观测API。

- [ ] **fixture测试/断言：** 六个大项目各`20000+400000+40000+40000+60000+40000+60000==660000`，40万动态1/5约1KB，id跨项目交错；五项目workspace为3300000。原50万fixture没有comments/versions，必须真实补齐，不改期望数蒙混。独立计数context、seed/ANALYZE时长分开记录；API仍造workspace/project/成员/标签框架，大规模行允许沿既有SQL seed。
- [ ] **测试红：** 保留普通15s request timeout及正常write/commit配置，仅延长seed所需token；请求前装observer。单项目及五项目workspace分别DELETE，要求204；每任务唯一id/准确scope/id/at/by，`runningCount==1 && attempt==1 && state==completed`，`errors`空且`coalesce((metadata->>'snoozes')::int,0)=0`，无running→available/scheduled/failed轨迹。workspace_drafts任务也核。旧实现/旧seed先被精确数量或运行断言拒绝。
- [ ] **探针实现：** 单项目用同workspace未删邻项目；删workspace用另存活workspace项目。无删除基线+job确实running且已有批提交时的读/write样本，采样后仍有处理证据；不通过长持锁制造负载。保存全部原样本、p50/p95/max、job时长与请求时长、环境/参数/准确源码身份，断言读200/写成功而不是自然404。
- [ ] **运行/变异：** `NERVE_MEASURE_SCALE=1 go -C server test -p=1 -count=1 ./internal/bootstrap -run '^TestTheDeletionsAtTheScaleTarget$' -v -timeout 60m`必须明确非skip。M40人为小预算使snooze但attempt=1，observer断言红；M41省comments/versions或payload，fixture断言红。恢复后完整规模绿，不用普通make test的skip充当证据。
- [ ] **G：** 把全部数据填spec附录并保留原件，提交`test(M4/P9): 在五项目目标规模验证删除任务单次完成`。若任何任务snooze/重试即未达通过线，先根因修复并受影响复审，不能靠调大45s或降低seed通过。

## Task 15：最终性质审计、race与完整F

**Files：** 本spec附录、ignored evidence；若发现缺陷只改归属的现有生产/测试文件并独评。预计100–350行，修复另计。

**Interfaces：** 不新增接口；核spec A1–A13与T1–T14结果一一映射。性能事实只能引用T14准确源码身份，不把后续文档HEAD冒充测量运行HEAD。

- [ ] **审计：** 每条验收列实际测试名/命令/原件/hash/结果，核至少15个代表生产变异横跨模块SQL、cutoff、FK、注册、窗口、预算、关系归属、保留与规模。可核验未受修改的已有证据，不伪造重复运行；失效或未覆盖才重跑。
- [ ] **红/绿修复：** 对新发现先真实性质红、最小根因修复、恢复变异字节/hash后绿并独评；审核每个新批写的约定五、错误链、事务/取消，以及37操作与22写目录未放宽。
- [ ] **race：** 用T12精确普通交错命令再`-race -count=5`，检查实际RUN与零失败；LateDeleteRowsAndPurgeRetry单独运行，明确允许的40P01及最终收敛。必要时issue/app/PG/jobs受影响回归，不为了数量重复所有已证检查。
- [ ] **F：** 按前述七条完整串行执行，准确提交/源码hash、raw/exit/计数完整；如改性能相关生产路径，T14目标规模重测。最终产品HEAD分支CI另核head_sha和三个job，不拿CI代替本地F。
- [ ] **G：** 提交`test(M4/P9): 完成删除清理的最终性质与本地检查点核验`；无文件变化则不造空提交，仅保留证据回执。此时尚不声称集成完成。

## Task 16：整分支deep-review、文档与后端交接

**Files：** 本spec/plan；改`docs/v0/M4-issue-core/reviews/P9-purge-review.md`准备占位；目标`docs/reviews/re-09-M4P9整体审核.md`（实际编号冲突时先核对）；M4-design.md、`docs/v0/v0-design.md`、`docs/v0/M3-workspace-project/M3-design.md`、`docs/v0/plane-diff.md`、README.md；M4/handoffs/M2-closeout.md、M3-closeout.md；P1a–P8各自spec§5/review§6追加回执；03/04当前接续。预计350–750行。

**Interfaces：** 无新产品接口；README精确六配置与失败/窗口/60d回收，契约删除文字与实际行为一致；M4§3.23四行和spec§7输入关闭表逐条承接。

- [ ] **独立整审：** 按准确基线`6b9d08d4821f1423c7bb9795557727c206b7880f..最终产品HEAD`执行deep-review，报告逻辑正确性/集成安全/测试覆盖；覆盖HEAD全部写的锁图、37操作访客/窗口、至少15变异证据及完整F，不以作者自查代替。
- [ ] **修复/复审：** Findings逐条裁定根因修复，必要红绿/影响回归和独立复审；若产品修复影响最终检查点，先F后push。报告真实失败/环境限制、超60天允许40P01，不写比代码宽的无死锁承诺。
- [ ] **同步文档：** 只有A1–A13真实证据齐才勾实施；M4状态写“实施与终审完成、集成待办”。追加P2 NULL creator历史矛盾的准确处理；handoffs只关后端子项、status仍open；M5命令行首投递、M8部署/性能与后续订阅责任保留。
- [ ] **验证：** `git diff --check`；逐条静态链接/实际接口/配置/任务种类核对，独立文档复审。只文档修改不谎称新行为运行，不重跑未受影响规模；正式review清楚区分prepared/static/runtime/branch CI/main CI身份。
- [ ] **G/交付：** 提交`docs(M4/P9): 汇总后台删除清理验收与后端交接`，详细正文说明审计范围/结果/未关闭项；按当前授权交付，合并必须已有或另获授权，`--no-ff`且main准确SHA CI全绿后才归档清理。本准备轮停在静态预检，不执行本Task。

## 作者自查（准备稿范围）

spec覆盖映射：A1→T1/T3，A2→T1–4，A3→T4/5/8，A4→T5/6，A5→T8–10，A6→T7/8/12，A7→T7/11，A8→T9/10/12，A9→T8/12，A10→T10/13，A11→T14，A12→T15/16，A13→T16。新接口集中于spec§2，任务不复制实现体；Review Focus五项均有归属测试。任务预计尺寸与依赖已列；所有执行勾选为空。此自查不是独立预检、原型、测试通过或产品授权。
