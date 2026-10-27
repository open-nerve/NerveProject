# M4/P8 草稿、搜索与链接计数实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **状态：** spec、plan 的独立静态预检已通过（[评审记录](../reviews/P8-drafts-search-review.md)）；负责人已授权从 T1 实施，阶段结束 deep-review 后先不合并。任务、测试、变异、性能和门禁按实际完成证据勾选，不据静态阅读勾选。

**目标：** 交付个人草稿与原子发布、可见范围内的搜索和选择器排除、pg_trgm部署验证，以及每一种Issue表示和列表的link_count。

**架构：** 草稿专属DraftLocks组合唯一既有Locks；发布在W/P/draft锁下调用共用创建核心，不重入gate/事务。app声明consumer小接口，结构解码在HTTP adapter、S锁目录在project模块、组合只在bootstrap。搜索在一个ReadOnlySnapshot中经目录和静态issue SQL读取，再复用Locks.seen。

**技术栈：** 仓库锁定Go1.27/toolchain1.27.1、pgx5.11.0、River0.47.0、sqlc1.31.1、PostgreSQL18.6、Node24/pnpm11.10.0/Playwright；不加新依赖。

**规格：** [P8 spec](../specs/P8-drafts-search.md)，新增接口与边界以spec§2为唯一表。基线main `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`，主控已核该SHA的CI `40451632280`全绿；P7归档/清理另有收尾证据。P8分支`codex/m4-p8`、worktree`.codex/worktress/codex/m4-p8`。

## Global Constraints

- 主目录HEAD永远main，所有开发只在P8 worktree；准备首提交`tools/keywords.json` phase=M4/P8，核到期例外不随手延期。
- plane/refer只读且不在其中跑git；不操作既有容器/Docker、不全局安装、不加设计未点名依赖、不建node_modules符号链接；不用stash/clean/reset --hard/force，不提取钥匙串或其他凭据。
- 提交中文详细多行，前缀feat/fix/docs(M4/P8)；push用`/usr/bin/git`。产品完成后自己deep-review整分支、局部修复复审，先不合并；集成需另有授权、--no-ff、准确main CI和证据归档后才清理。
- 只按已批准设计补consumer小端口：DraftLocks组合base，普通NewLocks五参不变；用例不另持authorizer/gate/写tx；发布不调用CreateIssue.Execute。W(S)→P(S升序)→D(U)，publish预检→无资源gate→W(S)→P(N)→D(U)。事务/55P03沿原平台/组合映射。
- 草稿仅一个action issue_draft.write；工作区admin/member允许、guest403；他人草稿member/admin404。旧project已删为draft404，显式新project无效422；项目退出/归档不把仍未删的个人草稿从列表抹掉。
- payload整替换；仅project nullable；保存只bodyshape结构与name/description_html/priority三个NUL路径422，不执行create业务验证、不发issue事件。publish同一创建核心、同At/actor/事务，结构错误同400。
- 搜索q trim后≤100 Unicode字符、拒NUL；limit默认50范围1–100；未知参数/重复标量422，非none日期可重复AND，none不可与同字段其他条件并用；空词updated_at DESC/id DESC，非空rank→similarity DESC→created_at DESC/id DESC；字面contains，不搜索description_stripped。
- app/domain不导入HTTP/gen/pgx/River；SQL只sqlc、只模块自有表、无模块SET。目录/授权/source/SQL/seen全同ReadOnlySnapshot。不放宽任何架构/关键词/契约封闭性守卫。
- link_count活links计数、不持久化在issues、不按作者或metadata过滤；全Issue/IssueDetail投影同时补，排序19→21，保留numeric/整数游标和旧19码。
- 新公开码与首次契约、PROBLEM_MESSAGES、en/zh-CN文案同Task进；生成物只make gen。历史P5“当时19码”不倒改，只追加P8承接。
- 新逻辑文件约400行以内；单Task目标350–1200手写行，保留修复余量，最高约16Task/1500行计划。生成物/静态SQL/合理大测试表按现有规则注明例外，不能省安全反例换行数。
- 所有有容器的套件串行，确认退出再下一套；测试复用issueWorld/newSplitRoles/pgtest.RowWaits/目录assert，不另造数据库/登录/fixture框架。Go -count=1与-p=1；零匹配不是绿；等待有条件/期限，无固定sleep。
- 每Task作者与要求/质量评审者独立；局部根因修复自行推进。架构回归、改变既定跨模块协议或意外高风险，先提交已完成工作再上报。已批准的端口补齐不重复请示。
- T4/T8/T12/T15有产品变化的检查点push前先完整本地七门禁；首次新增公开码立即跑强制test-web。已知W11不专查；合并main CI被它挡住且重跑仍失败时停止报告。
- 文档中文，代码注释沿仓库英文；handoffs只追加本期结果，status保持open至M4收尾。P9/P10b/P11/P12/P14/P15/M5–M8责任不提前实施或关闭。

## Review Focus

1. 项目已软删但draft仍活、guest访问他人草稿、合法篡改游标带别人的id：T3/4的真实404/403逐值对照和total，必须不泄漏payload/计数。
2. 等锁时草稿换项目、旧新项目UUID反序、已持连接重入gate或取早时钟：T7双方向真实PID阻塞、两种提交次序、拒绝零锁、同At和生产变异。
3. payload整体替换/null三态、合法但不完整内容、NUL、发布漏清洗/事件/回滚：T2/4保存结构矩阵，T5/6同bodyshape与共享创建核心，真实故障零变化。
4. 隐藏中间节点截断多层排除、反向关系漏排、adminGuest范围变宽、源不可见先暴露冲突、%/_被当通配符：T10/11/12用明确树/关系/范围fixture和同快照交错。
5. link_count只补列表而漏detail/嵌套、float游标损精度、扩展只在superuser下能用、中文短词实际性能未测：T9/13/14全投影/21码/运行角色，T15最终SQL的10万真实样本。

## 文件职责、依赖与通用收尾

I=`server/internal/modules/issue`，P=`server/internal/modules/project`，B=`server/internal/bootstrap`。每Task的文件表采用此相对根目录缩写；精确新增接口全部见spec§2.2–2.7，不保留备选签名。所有新增测试名在下面标“新增”；已有测试名来自基线盘点。实施时若重命名测试，先改本计划与命令，不用空regex输出充当验证。

Task依赖：T1→T2→T3→T4；T5依赖T4（保持可运行CRUD检查点），T6→T7→T8；T9→T10→T11→T12；T13依赖搜索投影已落地，T14→T15→T16。同一代码区域单作者，审查可并行只读；不同时运行两套数据库测试。

**通用收尾G（每Task执行）：** 先写精确断言并记录真实红；编译缺符号红只说明接口未实现，性质红另由生产变异证明。实现后运行列出的聚焦命令、记录真实RUN数/退出码/输出；对本Task代表生产变异记录红因/测试层，恢复文件字节与hash后再聚焦绿。生成Task跑make gen-go/gen-web，提交精确路径后gen-check核无漂移；git diff --check、影响包回归、独立要求与质量评审/修复复审、ignored任务报告完成后提交。不要git add .。提交示例给标题；正文解释原因、边界和验证。

**完整本地F（T4/8/12/15和最终有产品修复后）：** 顺序运行`make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`TURBO_FORCE=true make test-web`、`make e2e`。强制web输出必须证明非缓存。生成物先提交再gen-check；完整测试不因有聚焦绿省略。每次保存准确工作树/提交SHA、原始日志、各区域计数和exit0；发生问题先修复/复审再重跑受影响部分，未解释失败不能push产品检查点。F是实施期动作，准备期不执行。

## Task 1：草稿表、grants与同步级联

**Files：** 新`server/migrations/sql/00028_issue_draft_issues.sql`；改`server/sqlc.yaml`、`deploy/runtime-grants.sql`；I/app/delete_project_rows.go、cascade.go、cascade_test.go；I/adapter/postgres/queries/delete_rows.sql、delete_rows.go、delete_rows_test.go、link_deletion_test.go；新I/adapter/postgres/draft_deletion_test.go；I/module.go；P/app/issue_ports.go、cascade.go、cascade_test.go及现有fake；`server/migrations/issue_schema_test.go`、draft_schema_test.go、schema_test.go；B/runtime_role_test.go、project_deletion_test.go、workspace_deletion_catalog_test.go、issue_lifecycle_events_test.go、issue_scale_test.go。生成I/adapter/postgres/gen。

**Interfaces：** 在既有ProjectRowsDeleter增加`DeleteProjectDraftRows(ctx context.Context,b RowBatch)([]uuid.UUID,error)`；`WorkspaceDraftBatch`及`DeleteWorkspaceDraftRows(ctx,b)([]uuid.UUID,error)`；所有消费端`DeleteWorkspaceIssues(ctx,workspaceID,projectIDs,by,now)error`见spec§2.5。仍由原Cascade负责批量循环，不新造异步job。

- [ ] 新`TestDraftDeletionRows`：同workspace各建project草稿2、无project草稿2、已删旧project草稿1（旧deleted_at=T0）、另一workspace草稿1；删当前project只动2，删workspace动2个无project，另一workspace与旧T0不变，所有新deleted/updated为请求T1/actor；batch limit=1证循环、后批故障整体回滚。零项目workspace单独重复无project证明。
- [ ] 扩schema精确9列/3个索引/FK动作/JSONobject检查和授权目录；新`TestWorkspaceDraftCascadeCarriesWorkspaceID`（app或bootstrap按职责）令projectIDs=[]仍调用workspace批量，核全局draft步骤先于issues。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^TestDraftDeletionRows$'`，缺表/方法红；另跑`go -C server test -p=1 -count=1 ./internal/modules/issue/app ./internal/modules/project/app`记录签名消费者缺口，不算行为证明。
- [ ] 实现迁移/grants、sqlc静态批量锁/id键集和workspaceID传递；新rows只用原by/now，恢复旧行不覆盖；`make gen-go`后上述测试绿，`go -C server test -p=1 -count=1 ./migrations ./internal/archtest`通过。T1暂时只存储种子，T4前补API furnish，不改最终目录豁免。
- [ ] 变异M1漏无project batch或去`project_id IS NULL`，M2漏`deleted_at IS NULL`；真实存储反例红→恢复hash→绿；G，提交`feat(M4/P8): 增加个人草稿表并补齐删除级联`。

## Task 2：草稿纯值、游标与单表存储

**Files：** 新I/domain/draft.go、draft_test.go、draft_cursor.go、draft_cursor_test.go；新I/app/draft_ports.go；新I/adapter/postgres/queries/drafts.sql、drafts.go、drafts_test.go；消费既有domain/collaboration_cursor.go、shared/cursor.go/page.go。生成I/adapter/postgres/gen。

**Interfaces：** 产生spec§2.2的Draft/NewDraft/DraftPatch/CheckDraftPayload/DraftCursor及编解码；spec§2.3的DraftReader/Lister/Writer、DraftListRead/DraftWrite/DraftDeletion。Get/Lock不先过滤owner；list在SQL内owner/workspace/project有效集合过滤；更新库内JSONB等值不写。

- [ ] 新`TestDraftPayloadStorageBoundary`：`{}`、`{"name":""}`、256字标题、priority=`later`、101项重复labels、不存在state允许；三个自由字符串分别`\u0000`只给精确payload路径/invalid_format；正常Unicode/HTML原样。结构错误不是此domain函数承诺，HTTP在T4证。
- [ ] 新`TestDraftCursor`：`2026-01-02T03:04:05.123456Z`+非零id往返；零id、null、1/3元素、额外envelope键、错版本、非canonical拼写拒400；合法未知id接受。复用collaboration cursor helper，不新复制base64/JSON规范检查。
- [ ] 新`TestDraftRowsAndCursorScope`：A3行/B2行，同created_at用确定UUID`...001/002/003`；A limit2返回003/002、total3，下一页001、空尾页total3；把B的id/time及随机id填合法cursor仍只A/total3。旧归档项目纳入、旧deleted项目不在有效ID集合；nil项目纳入。JSONB键顺序换位更新不动At，payload整体换`{}`清掉旧键。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/modules/issue/domain -run '^TestDraft(PayloadStorageBoundary|Cursor)$'`；`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^TestDraftRowsAndCursorScope$'`。实现纯边界/共享游标/静态SQL后`make gen-go`，同命令绿。
- [ ] 变异M3去owner谓词，M4去任一NUL守卫，M5total错误套cursor或payloadJSONmerge；匹配domain/真实PG反例红→恢复；G，提交`feat(M4/P8): 实现草稿存储边界与个人游标分页`。

## Task 3：集中草稿锁编排与五个CRUD用例

**Files：** 新I/app/lock_draft.go、lock_draft_test.go、draft_reads.go、draft_reads_test.go、draft_writes.go、draft_writes_test.go、fakes_drafts_test.go；改I/app/draft_ports.go、I/domain/actions.go、errors.go；`server/internal/modules/access/domain/rules.go`、rules_test.go；P/adapter/postgres/queries/issue_ports.sql、issue_ports.go；新P/adapter/postgres/share_issue_projects_test.go；P/module.go；B/ports.go、ports_test.go；新B/issue_draft_app_test.go。对应生成与既有action完整性测试。

**Interfaces：** 产生`NewDraftLocks(base Locks,w WorkspaceSharer,p DraftProjects,d DraftReader)DraftLocks`、run/read/seeWorkspace私有签名（publish/publishHeld随首个生产消费者在T6补齐）；CRUD Execute和依赖见spec§2.3。project公开`UndeletedIssueProjectIDs(ctx,wid)([]uuid.UUID,error)`/`ShareIssueProjects(ctx,ids)(map[uuid.UUID]uuid.UUID,error)`；bootstrap消费workspace现有ShareWorkspaceByID。ActionDraftWrite单一工作区权限。

- [ ] 新`TestDraftLocks`和`TestDraftCRUD`：顺序记录无锁预检→W SHARE/auth→去重升序P SHARE→D UPDATE→重读→clock→write；create nil不取P/D，delete不入gate；owner/权限拒绝不进tx。旧project丢失draft404，显式新project跨workspace/不可见422；同workspace归档仍可保存；工作区成员未加入但project.read/LevelVisible可见的public项目也能保存，private不可见拒422，不能调用issue.read或VisibleProjects.Visible；退出项目后省略project patch/clear/delete可用，显式重给不可见project拒422。
- [ ] 新`TestDraftReadVisibility`：ReadTx包括workspace/目录/total；A能读自己的归档/退出项目草稿，admin不能读B，workspaceguest访问B403；已删project窗口404与随机id逐值相同。授权变化锁下重判，project变化409不追锁新project。
- [ ] 新project端口测试`TestShareIssueProjects`以反序输入两UUID证明单SQL升序SHARE、同项目去重、nil/empty无假锁；普通issue N锁路径未改。红：`go -C server test -p=1 -count=1 ./internal/modules/issue/app -run '^TestDraft'`及`go -C server test -p=1 -count=1 ./internal/modules/project/adapter/postgres -run '^TestShareIssueProjects$'`。
- [ ] 实现唯一DraftLocks组合base、读快照/五用例/端口接线，pure NUL在预检前；no-op仍经过权限/存在性/锁后比较，不写审计。加入draft action声明但新404/409公开注册与首次HTTP契约在T4同提交核集合。`make gen-go`，上述聚焦及`go -C server test -p=1 -count=1 ./internal/modules/access/... ./internal/archtest`绿。
- [ ] 变异M6把P SHARE换N、M7把旧project目录误用Visible或先owner过滤导致guest404；单元+project真实锁反例红→恢复；T7再真实跨模块证明。G，提交`feat(M4/P8): 集中编排草稿锁序与个人读写权限`。

## Task 4：CRUD HTTP、权限矩阵与删除目录（检查点F）

**Files：** 改`api/modules/issue.yaml`、`api/openapi.yaml`；新I/adapter/http/drafts.go、draft_query.go及测试；I/adapter/http/handler.go、I/module.go；新B/issue_drafts_test.go、permission_matrix_drafts_test.go；已有permission_matrix_coverage_test.go、issue_write_locks_test.go；`web/apps/web/core/lib/error-messages.ts`及测试、`web/packages/i18n/src/locales/{en,zh-CN}/errors.json`；`e2e/fixtures/furnish.ts`、issues.ts、assert/project.ts、assert/workspace.ts；furnish所有调用点（p4-archive、w3-workspace-settings及fixture自身）。生成api/dist、Go HTTP、TS client。

**Interfaces：** 公开spec§2.4前五CRUD和封闭Draft/IssueDraftPayload；HTTP映射保留SetProject/SetPayload三态，raw query只允许limit/cursor；`furnishIssueRows(api:Api,token:string,workspaceSlug:string,projectId:string,assigneeId:string,labelId:string):Promise<void>`，新增显式slug不查DB。

- [ ] 新`TestDraftHTTPShapes`：name缺失/空保存201；unknownpayload/project-in-payload、nullpayload/内层null、坏UUID/date、重复JSON键400；PATCH`{"payload":{}}`确实清空旧内容；project省略保持/null清除。合法业务坏值可保存，三NUL字段API422且snapshot零行变化；空patch/等值patch updated_at不变。limit默认50、1/100可用、0/101422、x400、未知/重复scalar422、非法cursor400。
- [ ] 新`TestDraftCursorNeverReadsOtherOwner`和权限矩阵五行：admin/member自己200、别人404，workspaceguest自己/别人403，removed/outsider/已删workspace404；旧project删除窗口不漏；把B游标或随机id代入A返回仅A且相同total，不泄漏Draft创建者/更新者字段。所有404比较完整problem而非只status。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(DraftHTTPShapes|DraftCursorNeverReadsOtherOwner|PermissionMatrixDrafts)$'`（矩阵新增test名固定如此）；先记录未注册路由红，再补五路由/生成和bodyshape。两码/双语同Task落地，逐值核PROBLEM_MESSAGES集合。同步把现有TestEachIssueWriteLocksItsProjectFirst的完整性声明拆为普通issue、draft CRUD、draft publish三类明确形状表（publish行T6注册时加入），三表union精确等于全部tagged非GET issue writes；没有通用excludeDrafts豁免。T4以W/P-S/D形状验证CRUD，F不能等T7再补守卫。
- [ ] 通过API在furnish建项目草稿，furnishWorkspace另建无项目草稿；更改所有slug调用点，删除目录断言包含新表且不留豁免。`make gen`；聚焦绿，`TURBO_FORCE=true make test-web`绿；变异M8把payload合并/M9漏旧project存在性/M10保存调用完整CheckNewIssue，真实HTTP反例红→恢复。
- [ ] G与完整F；记录准确HEAD的本地七门禁后才`/usr/bin/git push`产品检查点；提交`feat(M4/P8): 开放草稿接口并覆盖权限与删除目录`。继续下Task前处理检查点新失败。

## Task 5：抽取创建核心与存储载荷结构解码

**Files：** 新I/app/create_core.go、create_core_test.go；改create_issue.go、create_issue_test.go、lock_test.go、issue_events_test.go及其已有fakes；改I/module.go一次核心构造；新I/adapter/http/create_payload.go、create_payload_test.go；改issues.go；新B/issue_create_core_test.go（必要时复用既有issue_writes_test.go/event测试）。不改platform/bodyshape。

**Interfaces：** `NewIssueCreation(IssueCreationDeps)*IssueCreation`，私有`(*IssueCreation).createLocked(ctx,actorID,projectID,h held,in domain.NewIssue)(domain.Issue,time.Time,error)`；`IssueCreatePayloadDecoder.DecodeCreatePayload([]byte)(domain.NewIssue,error)`；adapter共用`newIssueInput(gen.IssueCreate)domain.NewIssue`。普通Execute签名/校验先后不变；CreateIssueDeps改为显式Locks+Creation，PublishDraftDeps显式DraftLocks+Decoder+Creation+DraftWriter；module只构造一次core，两用例共用，所有既有构造tests同步。核心不持gate/tx/authorizer。

- [ ] 新`TestCreatePayloadDecoder`核实际BodyShapes root存在，缺name `{}`与同体POST都是400 name/required；非法date、unknown键、内层null/数组类型，状态/码/fields对照。注入损坏旧JSON使用同canonical bytes直接创建比对；包含>16字段错误截断反例，不宣称原始输入顺序完全等价。空name、坏priority过结构后仍由domain给422。
- [ ] 新`TestCreateCoreKeepsEffects`断言清洗HTML、defaultAssignee、mentions/subscriptions、activity、description version、sequence和issue.created全部保留；唯一时钟取值由返回At与所有实际写行一致，普通Create仍先纯422后gate。复用已有`TestCreatingAWorkItemStampsItsRows`、`TestIssueCreatedEventComposition`、`TestIssueEventRefusalsAreUnobservable`。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/http -run '^TestCreatePayloadDecoder$'`；`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(CreateCoreKeepsEffects|CreatingAWorkItemStampsItsRows|IssueCreatedEventComposition|IssueEventRefusalsAreUnobservable)$'`。先固定行为再抽取，重构初始既有绿如实记录，不能伪造红。
- [ ] 抽取单核心和同一mapper，decoder先Check再decode，不在domain/app导HTTP/gen；同命令绿，`go -C server test -p=1 -count=1 ./internal/modules/issue/app ./internal/archtest`绿。变异M11绕过Check、M12漏description version或event；真实HTTP/组合红→恢复。
- [ ] G，提交`refactor(M4/P8): 共享已持锁创建核心与载荷结构解码`。

## Task 6：无请求体发布、原子副作用与拒绝顺序

**Files：** 新I/app/publish_draft.go、publish_draft_test.go、lock_draft_publish.go（复用lock_draft.go的资源预检）；I/adapter/http/drafts.go、handler.go；新I/adapter/http/draft_publish_body.go、draft_publish_body_test.go；I/module.go；契约；新B/issue_draft_publish_test.go、issue_draft_publish_effects_test.go、issue_draft_publish_locks_test.go（按行为/副作用/锁观察拆分）；扩permission_matrix_drafts_test.go、permission_matrix_test.go及三类writer目录/完整性等式和gate_test.go。生成HTTP/TS。

**Interfaces：** `NewPublishDraft(PublishDraftDeps)*PublishDraft` / `Execute(ctx,id)(domain.Issue,error)`；消费T5 decoder/core，并在T3既有DraftLocks上补齐spec§2.3的publish/publishHeld私有签名，成功201 IssueDetail。core返回At给DraftDeletion；无独立clock、写tx、gate。

- [ ] 新`TestDraftPublishUsesStoredPayload`：先保存“旧”再PATCH“锁下新”，空body发布只能新；附body经publish专属HTTP守卫拒400不能覆盖存储；成功草稿删、唯一新issue/sequence/event/活动/版本/订阅等原创建副作用同At/actor，重复发布404。无project422 required；归档项目即使payload缺name也先409；projectguest403；workspaceadmin+projectguest沿原create授权成功且response仍OwnOnly父/子显示规则。
- [ ] 新`TestDraftPublishMatchesCreateBodyErrors`：真实保存`{}`后发布与直接POST`{}`同400；保存不存在state/label/parent和非法assignee，再发布按创建聚合422路径；业务非法priority/256字name/101数组也只在发布422。旧引用后来删除的状态/父项分别测试。
- [ ] 新`TestDraftPublishRollsBackWithCreation`：事件故障、soft-delete故障、sequence后写失败三种注入；compare原snapshot、counter、草稿未删、jobs/活动/提及/版本均无新增。失败不是只核HTTP500。已有事件组合帮手复用，不开第二事务。
- [ ] 新`TestDraftPublishBody`用真实handler链核零字节/EOF成功，空白、JSON `{}`、chunked任一字节均400且fake usecase未调用，匿名仍先401；包括ContentLength=-1的chunked与乱码body，读错误沿APIErrors.BodyError。红：`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/http -run '^TestDraftPublishBody$'`及`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^TestDraftPublish(UsesStoredPayload|MatchesCreateBodyErrors|RollsBackWithCreation)$'`。实现publish-only `draftPublishBody(api *httpserver.API) func(http.Handler) http.Handler`，仅匹配POST该路由，接在既有认证/限流/BodyLimit之后、generated handler之前；最多读第一个字节，零字节EOF放行，其他用api.Errors.BodyError，不能仅信Content-Length或扩全局bodyshape。实现D锁重读删除/changed后先拒归档，再decoder→CheckNewIssue→core→删除、完整权限矩阵第六行、归档/错误优先级；T4显式三类写形状目录补publish的gate/W-S/P-N/D-U行及覆盖等式；`make gen`，同命令及issue app回归绿。
- [ ] 变异M13从请求/锁前payload创建、M14省草稿删除或独立提交、M15二次Clock.Now；HTTP/组合必须红，恢复hash后绿；G，提交`feat(M4/P8): 原子发布存储草稿并复用创建副作用`。

## Task 7：交错16/17、锁形状与授权变化

**Files：** 新B/issue_draft_interleaving_test.go、issue_draft_lock_shapes_test.go；将请求PID/有界等待、写资源边界、W/P/D终态小帮助分别拆入issue_draft_wait_test.go、issue_draft_resources_test.go、issue_draft_terminal_test.go（每文件约400行内）；扩B/issue_draft_write_locks_test.go、issue_draft_publish_locks_test.go的请求归属探针及issue_writes_test.go的既有route wiring小帮助。复用issue_write_locks_test.go、gate_test.go、project_write_locks_test.go的完整目录/锁序/gate守卫而不改普通issue锁序；I/app/lock_draft.go仅修已证根因。ignored锁图/阻塞证据。

**Interfaces：** 不新增产品接口。用现`pgtest.RowWaits/LockNotAvailable/heldTx/serializeOnTable`，记录请求连接PID、holderPID和pg_blocking_pids链；不用观察到worker等待就当作请求阻塞。

- [ ] 新`TestDraftPublishAndProjectDeletionSerialize`：publish先拿P→删除等→issue与draft同At后项目删除；删除先→publish等待后draft404，即使直接构造P9窗口draft仍活也404。反转发起顺序各一次，核零副作用/没有死锁。
- [ ] 新`TestDraftProjectMoveAndPublishSerialize`：publish预读P1后阻塞，update移到P2先提交→publish409，绝不能旧P1建issue；publish先D→move等，完成后move404；初始P1 UUID小/大两组，双向move各锁新旧升序，无40P01，不持D后再求P2。
- [ ] 新`TestDraftWritesLockShape`：普通CRUD拒绝/guest/他人不进gate/取锁；普通W SHARE、P SHARE与publishP N不同；workspace删除/角色降级等W后重判；W后先不取额外锁重读资源存在性，再auth，不把新project追锁；删除+降guest均已提交且请求W等待后读终态答draft404，只有降guest而resource仍活答403。P/D等待后删除仍404、项目变409；nilproject只W/D。记录Clock.Now在最后D锁后，阻塞期间推进受控clock，提交行必须新时刻；gate等待时池连接使用数不增长。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^TestDraft(PublishAndProjectDeletionSerialize|ProjectMoveAndPublishSerialize|WritesLockShape)$'`。已有实现可能先绿，真实性质红由M16去project锁后比较、M17倒置P排序/先D、M18提前Clock、M19持tx入gate分别证明；每次仅一个变异，有界ctx，不把timeout/编译错算预期断言红。
- [ ] 恢复全部生产hash后同命令绿；再`go -C server test -p=1 -count=5 -race ./internal/bootstrap -run '^TestDraft(PublishAndProjectDeletionSerialize|ProjectMoveAndPublishSerialize)$'`；更新完整当前writer gate/锁图，确认普通issue锁序未被改写。G，提交`test(M4/P8): 验证草稿发布与删除移动的真实交错`。

## Task 8：I14接口故事与草稿全链路回归（检查点F）

**Files：** 新增`e2e/stories/issue/i14-drafts.spec.ts`；扩展`e2e/fixtures/issues.ts`五个草稿操作及`e2e/stories/workspace/w3-workspace-settings.spec.ts`无项目/零项目/旧草稿审计证据；spec验收证据。复用T4已扩展的`e2e/fixtures/furnish.ts`、`e2e/fixtures/assert/{catalog,project,workspace}.ts`和既有`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`，不制造空改动。复用B/workspace_deletion_test.go、workspace_deletion_catalog_test.go、project_deletion_test.go的全表目录与回滚回归，保留原subItems/links断言；零项目HTTP组合证据放在W3。

**Interfaces：** fixture只包此时已落地的六个草稿和既有issue接口，不用SQL前置。I14 save→patch→publish按两个API，不能publish body拼表单。删除断言按全表目录。

- [ ] 新I14：member保存缺name草稿201→发布400，与同体create问题相同；补name与description/parent后201，草稿list消失/get404；另草稿state被删发布422；他人404/projectguest403；归档/退出项目草稿仍列出、发布按授权拒绝。校验数据库只用于断言同事务/同审计，不用于前置数据。
- [ ] 给工作区删除故事显式无project草稿；新增零项目workspace删草稿组合用例（T1已存储验证，现走HTTP），保留已删旧project草稿原时刻；项目删除目录含draft且原subItems/links断言继续通过。
- [ ] 红：`pnpm --dir e2e exec playwright test stories/issue/i14-drafts.spec.ts --project=chromium`（先`make build`确保当前二进制；用例新增后的失败必须是真实行为或明确无实现，不是旧binary）；实现仅fixture/已证局部修复，同命令绿。
- [ ] 运行`pnpm --dir e2e exec playwright test stories/project/p4-archive.spec.ts stories/workspace/w2-landing.spec.ts stories/workspace/w3-workspace-settings.spec.ts --project=chromium`；完整F保证全目录覆盖。重做M14漏草稿删除，I14必须红→恢复。
- [ ] G、F、准确HEAD检查点push；提交`test(M4/P8): 完成草稿发布故事与完整删除回归`。

## Task 9：pg_trgm迁移、owner/runtime与生成类型

**Files：** 新`server/migrations/sql/00029_issue_issues_name_trgm.sql`；改server/sqlc.yaml、migrations/schema_test.go、issue_schema_test.go；将往返测试移到新migrations/trigram_schema_test.go并扩展；新I/adapter/postgres/queries/search_issues.sql的最小SearchIssues查询及search_type_test.go；新B/search_runtime_role_test.go复用runtime_role_test.go的newSplitRoles并保留其全目录授权守卫；生成I/adapter/postgres/gen。不改镜像或依赖版本。

**Interfaces：** 产生扩展和partial GIN，以及最终`gen.SearchIssuesRow.Similarity`必须float32的编译约束；外部q，SQL参数@query。T11在同一查询名扩完整投影，不留单独unused实验查询。

- [ ] 新`TestSearchTrigramRuntimeRole`复用newSplitRoles：owner`rolsuper=false`、数据库CREATE，以owner Up成功；runtime可执行similarity并查询issue，不能迁移DDL；草稿grants覆盖。新`TestSearchSimilarityType`含`var _ float32 = gen.SearchIssuesRow{}.Similarity`并运行实际数值断言，拒interface{}。
- [ ] 扩`TestMigrationsGoUpDownAndUpAgain`检查GIN名/partial谓词/pg_trgm；Down删除索引但保留扩展，再Up成功；预先存在扩展同样成立且Down/Up前后扩展OID不变。schema_test.go当前functionsQuery枚举public全部函数，必须通过pg_depend对extension成员关系区分pg_trgm对象与应用自有函数；自有函数目录仍精确匹配River/Down空集，不删函数守卫或仅宽泛排除函数名。runtime grants完整性保持原样，不拿schema分类替代权限验证。红：`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^TestSearchTrigramRuntimeRole$'`和`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^TestSearchSimilarityType$'`。
- [ ] 实施00029/schema/最小SQL，make gen-go，记录sqlc准确生成字段类型；上述绿，串行跑`go -C server test -p=1 -count=1 ./migrations -run '^TestMigrationsGoUpDownAndUpAgain$'`与B已有`TestTheGrantsFileCoversEveryRelationAndFunction`。不以superuser成功替代owner证据。
- [ ] 变异M20从sqlc schema漏00029、M21漏draft grants；生成/类型检查、runtime完整性分别红→恢复；G，提交`feat(M4/P8): 建立搜索扩展并验证最小运行权限`。

## Task 10：搜索词汇、编号折叠与参数错误

**Files：** 新I/domain/search.go、search_test.go、search_identifier_test.go；复用domain/identifier.go、list_date.go、shared/page.go及api/testdata/identifier-fold.json。必要的有限字面pattern helper放同职责search文件，不复制日期/identifier算法。

**Interfaces：** `SearchFor/SearchQuery/ParseSearchQuery(map[string][]string)(SearchQuery,error)`见spec§2.6；只解析不访问目录/DB。完整identifier数字遵既有1..2147483647无前导0，纯数字分支允许00012→12。

- [ ] 新`TestSearchQuery`表：空/q空格默认50；100个“中”通过/101失败q too_long；NUL/非法UTF8 q invalid_format；limit0/101422、x400；未知键422、重复q/for/project/issue/limit422；重复start_date非none范围AND；none单值通过，none+date/date+none/none+none均422，首个none下标路径分别[0]/[1]/[0]且not_allowed；for缺source required、无for仅source not_allowed；for空串/unknown均422 for/not_allowed，issue_id空/project_id空均422各自invalid_format，省略与出现空值分别测试；badUUID/date逐路径。
- [ ] 新`TestSearchIdentifierAndNumeric`消费共享fold fixture，`WEB-12`/`İ/ı/i/I`冲突按MatchIdentifier，纯`00012`→12、`0`/2147483648/超长数字不建sequence分支但保留Text；`WEB-012`非精确编号。literal `%`、`_`、`\`保留为字面文本交SQL参数编码，不变regex/trigram阈值。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/modules/issue/domain -run '^TestSearch(Query|IdentifierAndNumeric)$'`；实现trim/UTF8/runes/边界、复用现有date/number/fold规则，聚合纯422按现有FieldError规范，绿。
- [ ] 变异M22使用byte长度、M23溢出数字转int/放宽完整编号、M24重复scalar最后值取胜；domain反例红→恢复；G，提交`feat(M4/P8): 定义严格搜索参数与编号匹配输入`。

## Task 11：搜索静态SQL、完整排除与同快照授权

**Files：** 新I/app/search_ports.go、search_issues.go、search_issues_test.go、fakes_search_test.go；扩I/adapter/postgres/queries/search_issues.sql；新search_issues.go、search_issues_test.go；新B/issue_search_test.go、issue_search_snapshot_test.go、issue_search_snapshot_ports_test.go（复用既有快照probe及通道帮手）；适配I/adapter/postgres/search_type_test.go的完整查询参数，保留原生float32编译与运行断言；扩B/search_runtime_role_test.go验证最终静态查询的runtime角色执行；消费现有visible_projects.go/seen.go；生成sqlc。

**Interfaces：** `SearchRead`、`SearchRows.Search(ctx,in)([]domain.Issue,error)`、`NewSearchIssues/Execute`见spec§2.6。SQL自身仅issue表；projectIDs/OwnOnly/精确编号项目通过参数。每候选先全部过滤后LIMIT，最终只一套Locks.seen补可见parent/subcount。

- [ ] 新`TestSearchLiteralTermsAndOrder`：P1/WEB编号12的标题“different”、P2编号12标题“12 report”、P1编号13标题“WEB-12 notes”，q WEB-12保持精确第一且标题候选仍在；q12保留两项目sequence和标题。另存“100% done/100X done”、“under_score/underXscore”、“a\\b/ab”、中文“修复登录/登录问题”，只字面contains；空q按显式updated_at+id，非空同similarity/created_at靠id DESC，limit1/2先scope再取。
- [ ] 新`TestSearchIssueScopeAndExclusions`：A→B→C→D四层树，B对ownOnly隐藏；parent(source A)排A/B/C/D及当前parent，sub(source D)排A/B/C/D且只无parent；source X另有6个候选，对stored blocked_by/duplicate/relates_to三类型各建正向/反向活边，6个全部排；第7个候选只一条软删边允许返回（正反分别子例）。隐藏/归档/删除中间节点仍穿透；候选自身隐藏/归档/删除/别workspace绝不出；adminGuest与ownOnly同范围。日期none/闭区间/重复AND及source项目冲突顺序固定。
- [ ] 新`TestSearchReadSnapshot`：读目录后阻塞第二连接改项目成员/OwnOnly、issue父/子或关系，当前响应全部旧快照，下一次全部新；结果hidden parent=null、subcounts只可见，scope与seen不跨tx。用现snapshot帮手，不靠sleep。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^TestSearchLiteralTermsAndOrder$'`；`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^TestSearch(IssueScopeAndExclusions|ReadSnapshot)$'`。实现静态CTE/UNION排除、literal ILIKE、rank排序与唯一seen；make gen-go，同命令及T9type/runtime绿。
- [ ] 变异M25ownOnly=false/M26只一层后代或单向relation/M27ILIKE不escape/M28授权后过滤LIMIT/M29不用ReadTx；真实PG或组合红→恢复；G，提交`feat(M4/P8): 在统一读快照内执行可见范围搜索`。

## Task 12：搜索HTTP、source错误顺序与矩阵（检查点F）

**Files：** 新I/adapter/http/search_issues.go、search_issues_test.go；改draft_query.go/handler.go、I/module.go、api/modules/issue.yaml/api/openapi.yaml；新B/permission_matrix_search_test.go；扩permission_matrix_coverage_test.go、issue_search_test.go；生成HTTP/TS。

**Interfaces：** `GET /api/v0/workspaces/{slug}/issue-search`→searchIssues→200 IssueCollection `{data:[]}`，无next_cursor/total/page。定向raw query middleware与generated binding ErrorHandler用于draft/search：ErrorHandler先ParseQuery再判重复scalar；不能套错现有listSingleQuery的词汇或全局改binding错误。

- [ ] 新`TestSearchHTTPContract`：缺q空词成功、data数组非null；未识别cursor/order_by422，非整数limit400，畸形百分号400（含q=a&q=b&bad=%GG；binding ErrorHandler先ParseQuery），未知query/重复标量422；none+date/date+none/none+none按首个none下标422；for=parent+hidden source+不同project先404 source而非冲突；可见source+不在scope的显式project先404 project；都可见但冲突422。source归档/删除/其他workspace/ownOnly-hidden与随机id完全同404。
- [ ] 新`TestPermissionMatrixSearch`覆盖workspace admin/member/guest可搜索各自project目录；outsider/removed404，projectguest OwnOnly和adminGuest不放宽；无visible项目返回data=[]。所有查询类别（空/数字/标识/标题）和for类别走同scope；验完整响应schema不出现page/total/隐藏parent。
- [ ] 红：`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(SearchHTTPContract|PermissionMatrixSearch)$'`，补路由/构造/契约/定向raw middleware，make gen，同命令及issue-search相关app/HTTP包绿。
- [ ] 变异M30无for时忽略issue_id、M31scope前返回source项目冲突、M32source归档放行；HTTP逐值断言红→恢复；G、F、准确HEAD检查点push，提交`feat(M4/P8): 开放搜索接口并固定范围错误语义`。

## Task 13：全表示link_count与21种排序

**Files：** I/domain/issue.go、listvocab.go、list_cursor_key.go及tests；I/adapter/postgres/queries/{issues,sub_issues,list_issues,list_issue_groups,list_issue_subgroups,search_issues}.sql；对应issues.go、sub_issues.go、list_issues.go、list_cursor.go、search_issues.go和list_order/list_numeric_key/list_issue_groups/list_issue_subgroups/list_precision tests；I/adapter/http/issues.go及tests；api/modules/issue.yaml、api/testdata/issue-order.json；已有B/issue_list_orders_test.go、issue_lists_test.go、project_issue_preferences_test.go；I/app/list_settings_test.go；新B/issue_link_counts_test.go；`server/internal/platform/httpserver/apitest/issue_shapes_test.go`；生成API/sqlc/TS及受影响fixtures。

**Interfaces：** `domain.Issue.LinkCount int64`，所有公开Issue/IssueDetail required link_count；ListOrderAllowed增加±link_count成21，不增filter。三个sqlc列表row同形，listRecord/listPosition唯一转换；计数游标非负int64、SQL内部numeric不变。

- [ ] 新`TestIssueLinkCountsAcrossRepresentations`：issue A活links2+deleted1、B0、C1；A一个metadata blocked一个pending仍2，改title/url/worker metadata仍2；删一活link变1。核get/byIdentifier/create=0/update/archive/restore/subIssues/relations嵌套/search/flat/group/subgroup/member均准确，含ownOnly可读issue的全部活links；IssueDetail只比Issue多description_html。
- [ ] 扩已有`TestFlatIssueListOrders`、`TestFlatIssueListCountOrder`、`TestIssueGroups`、`TestIssueSubgroups`、`TestIssueCellContinuation`、`TestListNumericCursor`：0/1/2/2的确定UUID双向limit2续页完整无重漏；计数等值id方向随order；bigint边界、负数/null/非整数cursor400，合法篡改只能起点变化不越权；flat/groups/subgroups最终row编译同形。list_precision_test.go的checkListNumericExpression synthetic source必须新增link_count列；child_count与link_count分别覆盖2^53-1、2^53、2^53+1、MaxInt64，不能用同一列别名掩盖混用。
- [ ] 扩`TestIssueListAllOrders`/偏好共享词汇/issue-order.json为当前21，domain/list_order_test.go fixture scenarios10→11、bootstrap seedListOrder补links，I/app/list_settings_test.go把旧order_by=link_count拒绝样例改真正非法词；display_properties.link_count仍拒绝（现有合法键是link），保留原19每个断言。红：`go -C server test -p=1 -count=1 ./internal/bootstrap -run '^Test(IssueLinkCountsAcrossRepresentations|IssueListAllOrders)$'`及`go -C server test -p=1 -count=1 ./internal/modules/issue/adapter/postgres -run '^Test(FlatIssueListOrders|FlatIssueListCountOrder|IssueGroups|IssueSubgroups|IssueCellContinuation|ListNumericCursor)$'`。
- [ ] 实现各基础与列表/search投影、映射、21词和fixture；make gen，聚焦绿；`go -C server test -p=1 -count=1 ./internal/modules/issue/... ./internal/platform/httpserver/apitest`及强制test-web。不能把旧P5文档当当前assert修成21，历史只在T16追加承接。
- [ ] 变异M33去links软删谓词/M34只改list漏GetIssue或SubIssues/M35link cursor取0/M36删一列表排序分支；各对应真实PG/组合续页红→恢复；G，提交`feat(M4/P8): 补齐链接计数表示与双向列表排序`。

## Task 14：I18搜索与I4双向链接计数故事

**Files：** 新`e2e/stories/issue/i18-search.spec.ts`；改i4-list-order-calendar.spec.ts、fixtures/issues.ts、fixtures/assert/issue.ts；必要的当前生成类型fixture，不改生产前端store/service/比较器。

**Interfaces：** 只消费已生成searchIssues/draft APIs及现links API。I18接口层，页面/命令面板留P15；I4真实order_by/cursor接口，不以测试数组sort替代服务器排序。

- [ ] I18：两个可见项目同sequence、隐藏项目高分、ownOnly别人同词、Unicode编号fold、中文2/3字、空词日期none；for三种排除用三层以上树及反向relation；断言结果只有Issue.data、不含page/total，隐藏父null。全部前置由API建，确定期望id顺序，不复制生产rank算法计算expected。
- [ ] I4 links=0/1/2/2，四个独立id，±link_count各limit2至少两页，合并无重复/遗漏且tie方向对；软删一link后新请求count减少。校验同个issue的detail和搜索count一致。
- [ ] 红：先`make build`，再`pnpm --dir e2e exec playwright test stories/issue/i18-search.spec.ts stories/issue/i4-list-order-calendar.spec.ts --project=chromium`；若已有实现先绿如实记录，M26单向排除/M33软删漏谓词至少各被故事杀死，不借旧binary或测试修改伪红。
- [ ] 恢复hash→重新build→同命令绿；跑I14确保links字段不破发布形状；G，提交`test(M4/P8): 覆盖搜索选择器与链接计数续页故事`。

## Task 15：10万搜索实测、性质审计与完整门禁（检查点F）

**Files：** 新I/adapter/postgres/search_performance_test.go（场景、独立oracle、只显式环境启用的runner）、search_performance_fixture_test.go（精确seed与最小runtime pool）、search_performance_evidence_test.go（实际SQL/实参/OID、元数据与计划）；spec附录A和ignored性能/门禁/变异报告。三文件各约400行以内、合计<1200手写目标；产品修复只根据实际失败定位后按原Task评审。

**Interfaces：** 新`TestSearch100KPerformance`以`NERVE_SEARCH_PERF=1`启用；默认明确Skip，不计为性能已跑。复用现pgtest数据库和最终SearchRows/SQL，禁止单独简化查询冒充生产耗时。

- [ ] 建100000 issues（50000英文/50000中文，均匀分两个可见项目；额外明确ownOnly/归档/删除样本使总数仍100000），固定seed和分布；英文查询`lo`/`login`，中文`登录`/`修复登录`，再空词/纯数字/完整编号正确性样本。含高/低选择性两类，记录期望candidate及topN而非只毫秒。
- [ ] 运行`NERVE_SEARCH_PERF=1 go -C server test -p=1 -count=1 -v ./internal/modules/issue/adapter/postgres -run '^TestSearch100KPerformance$'`；记录准确HEAD、PG版本/locale/LC_CTYPE、数据生成参数、最终SQL/实参、ANALYZE状态、连接设置jit/plan_cache_mode、EXPLAIN(ANALYZE,BUFFERS)、至少5次样本/中位数/范围、暖冷边界。2字可能顺扫，3+也按实际选择性记录；不SET enable_seqscan=off，不臆造固定延迟达标。
- [ ] 若结果暴露局部退化，在不改搜索契约/模块边界条件下根因优化，同数据重新测量并保留前后证据；架构级范围/高风险按纪律先提交上报。性能文字回填spec，准备稿未执行声明改为准确历史+实施结果。
- [ ] 审计M1–M36台账：每一安全/授权/锁/事务性质有非单元证明；不能自然生成的边界注明真实存储注入理由，未被杀死的变异补反例后重试；全恢复SHA确认。再跑交错16/17 -count=5 -race及已有`TestIssueListReadSnapshot`、`TestMemberIssueReadSnapshot`、`TestIssueListsAsRuntimeRole`、`TestIssueListPermissions`，确认21码没有打坏P5。
- [ ] G与完整F，所有A1–A10有准确证据后提交`test(M4/P8): 记录搜索规模实测与阶段完整验证`并推准确HEAD检查点；分支CI等待和失败处理如实留证，不以本地绿代替远端结果。

## Task 16：整分支deep-review、修复复审与文档交付

**Files：** `docs/v0/M4-issue-core/{specs,plans}/P8-drafts-search.md`、`docs/v0/M4-issue-core/reviews/P8-drafts-search-review.md`；M4-design.md §3.23/12/15；docs/v0/v0-design.md §5.4/6.9；docs/v0/plane-diff.md草稿/trigram/search差异；README部署；输入handoffs/M0-P4-pg-trgm.md、M1-P2-trim-content.md、M2-closeout.md、M3-closeout.md；P1a–P7相关spec§5/review§6仅追加承接结果。不得全局将open改closed。

**Interfaces：** 不加产品接口。输出精确BASE..HEAD、spec/plan/review路径、门禁/CI/性能/变异/审查证据索引，阶段先不合并。

- [ ] 冻结准确BASE=`166b9efd...`至产品HEAD的整分支范围，使用deep-review技能做一次完整审查；文档必须区分首次完整审查与修复范围复审。检查全部diff含迁移/契约/生成/fixtures/既有回归，不只审最新提交。
- [ ] 按发现根因修复；每项有复现或论证、定向测试、生产恢复与独立复审。新产品改动跑受影响回归与F，纯文档改动只静态核对/格式/链接，不重复伪造整轮红绿；最新准确HEAD的分支CI另核。
- [ ] 更新spec全部A条款证据，plan实际勾选与未跑项，review含真实计数和未解决风险；总体§5.4写project列/存储发布/权限/409/整替换，§6.9写OR匹配/空词/范围排除/日期/Unicode/stripped不搜索；plane-diff对比真实Plane读取出处；README写镜像内pg_trgm及非superuser数据库CREATE与runtime PUBLIC EXECUTE部署条件。
- [ ] 输入handoff仅追加M4/P8处理结果/证据，M0部署后续留M8；P5历史19码事实保留，在当前说明追加21码；P9/P15等后续承接按spec§5。M4全部完成/9.4/M5–M8最终交接均不提前勾。
- [ ] 自检任务/验收/文件/接口/测试名一致，git diff --check、文档相对链接、关键词/生成无漂移；提交`docs(M4/P8): 记录阶段验收与整分支审查证据`。告知用户分支、基线、spec/plan/review路径与状态，先不合并、不清理P8。

## 预检自检与覆盖表

| 设计P8粗任务 | Task | spec验收 |
|---|---|---|
| 1 schema/grants/级联/furnish | 1、4、8 | A1 |
| 2 草稿领域/读写/游标 | 2–4 | A2–A3 |
| 3 update/delete/码/项目S | 3–4、7 | A2–A3、A5 |
| 4 发布/创建core/bodyshape | 5–6 | A4 |
| 5 交错16/17 | 7 | A5 |
| 6 pg_trgm/sqlc/roles | 9、11 | A6 |
| 7 搜索词汇/编号/顺序 | 10–11 | A7 |
| 8 scope/for/date | 11–12 | A7 |
| 9 link_count/两码 | 13 | A8 |
| 10 矩阵/整程序 | 4、6–7、11–13 | A3–A8 |
| 11 I14/I18/I4 | 8、14 | A9 |
| 12 10万性能 | 15 | A10 |
| 13 文档/review | 16 | A11–A12 |

准备期作者自检只确认文件职责、单一接口、拒绝顺序、锁图计划和验收覆盖；独立预检记录另存review。所有测试命令均为未来实施步骤，新增测试须实际存在/匹配；不存在目录、空匹配、缓存或仅编译红不能写成行为验证。
