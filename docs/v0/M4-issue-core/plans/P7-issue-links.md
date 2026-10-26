# M4/P7 链接、出站与元数据抓取实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **实施状态：** I1 的局部物理约束修正已由最窄 PG/sqlc 实验验证并经独立复审；全部预检发现已关闭，基线 main CI 全绿。负责人已授权启动 P7；T1–T5 独立评审通过，T4 检查点本地 Go 8228/Web 1170/E2E 150 门禁通过；T6 地址策略已实施待独立评审，整阶段验收未完成。

**目标：** 实现 M4 §12/P7 的链接CRUD、首个生产事务订阅者、抓取worker和防SSRF出站，交付I10接口与带links的I15/I17删除目录故事。

**架构：** 链接请求写经既有Locks.run；app消费小端口，domain纯值与规则。outbound是独立标准库平台包，crawler适配器消费它并解析HTML；bootstrap构造出站与已有事务投递桥。抓取只以id+url+未删除守卫更新metadata，不改审计，不取项目锁。

**技术栈：** 仓库锁定Go1.27/toolchain1.27.1、pgx5.11.0、River0.47.0、sqlc1.31.1、PostgreSQL18.6、已存在x/net/html v0.60.0、Node24/pnpm11.10.0/Playwright。不增加依赖。

**规格：** [P7 spec](../specs/P7-issue-links.md)。基线为P6合并 `5005be985e7ec78727edc6f8fc5c05999b45cc65`；主控已核实同一SHA的main CI `40305247546` 最终Success，server/web/e2e全绿；首轮基础设施超时及重跑历史另记review。

**方法与状态：** 准备完成并经独立预检/修订复审；T1–T5 已实施且独立评审通过，T4 检查点本地完整门禁通过；T6 地址策略已实施待独立评审，其余任务与整阶段验收未完成。沿用02§3允许的紧凑方法：确切文件/接口/断言/命令，实施时逐Task真实红绿、生产变异、恢复与独立评审；本次静态预检不冒充完整原型或重放；独立最窄PG/sqlc实验只证明spec附录列明的物理约束能力。已批准M4设计是架构依据，局部细化见spec§3。正式实施按负责人授权逐Task推进，不自动合并。

**本地门禁安排（负责人 2026-10-26 调整）：** 用户要求提前执行完整本地 CI。T4 检查点已由主控完成全套、修复及独立复审；后续有产品变化的开发检查点推送前先跑完整本地门禁。逐 Task 仍保留聚焦与受影响回归，最终 T15 继续汇总验收。本条记录执行方法与 T4 已通过事实，不表示 P7 整阶段验收完成。

## Global Constraints

- 分支`codex/m4-p7`，worktree`.codex/worktress/codex/m4-p7`；主目录HEAD永远main。准备首提交改`tools/keywords.json`为M4/P7，检查到期例外，不随手延后。
- plane/refer只读且不在其中跑git；不操作既有容器/Docker、不全局安装、不新增未点名依赖、不建node_modules符号链接；不用stash/clean/reset --hard/force push，不取钥匙串凭据。
- 提交用中文、多行解释关键变化，前缀feat/fix/docs(M4/P7)；push用`/usr/bin/git`。产品完成后deep-review整分支终审、局部修复复审，不等Claude但不自动合并。集成另经授权--no-ff、精确main CI、证据归档核hash后清理。
- 每Task实现者与要求/质量评审者独立，局部问题根因修；架构回归、需改变既定模块协议、意外高风险先提交完成工作再上报。本文点名的跨模块组合已在M4设计批准，不重复索要许可。
- 请求写只经Locks.run，隐藏404先于403；拒绝不进闸门/事务；锁下重读再授权→项目归档→工作项归档→noop→唯一→时钟→issues→links→activities→events。三写声明server_busy与Retry-After。
- 抓取只更新metadata，单句id/url/deleted_at守卫；不借此引入新锁路径、版本列或审计更新。只改title必须保留数据库当前metadata，UpdateLink返回domain.Link，不退回只error签名。
- domain/app只标准库和本模块内层/shared，不见net/http、River、pgx、x/net/html或outbound；platform/outbound不导入模块/shared或其他platform。保持现有架构与CLI守卫。
- URL保持2048 Unicode字符和原文精确相等；12列不变，以spec§2.2的内置hash exclusion作活行排重，无extension/新依赖/散列字段。仅23P01+issue_links_issue_id_url_key映射link_exists，其他数据库错误原链保留。
- SQL仅sqlc，不跨模块JOIN、模块SQL无SET；迁移/SQL/契约生成物只经make gen。新公开码与首次契约、双语文案同Task进入，PROBLEM_MESSAGES与契约码集合相等。
- 新手写逻辑按职责约400行以内；既有module.go196行、HTTP handler189行、activity170行、issue_write_locks_test322行（准备期测量）只加本期职责，接近上限就拆同职责文件；生成物按既有例外，不手改。Task目标350–1200手写差异行、约10%余量，最高16Task/约1500行plan。
- 复用Locks/issueWorld/marked audit helpers/pgtest.RowWaits/LockNotAvailable/permission matrix/runtime-role/APIfixtures；不新造测试数据库、登录或RPC基础设施。
- 容器套件一次一套，确认退出再下一套。全部等待有条件/期限，无固定sleep。Go测试用-count=1，整仓用GOFLAGS=-p=1；聚焦命令须确认真实RUN数，空匹配不是通过。
- 测试不用真实外网：outbound用httptest、私有resolver/dial/interface依赖；其他link fixtures用会在拨号前blocked的地址或本地fixture。不得把用户邮箱或凭据发送到外部。
- 既有关键词/架构/生成/文案守卫不弱化。IANA完整性fixture不能由生产deny表枚举而来自独立官方快照；重叠CIDR用清单相等守卫和端点行为分别证明。
- 文档中文、代码注释沿仓库英文惯例；handoffs只追加本期处理结果，status保持open，P8/P9/P11/P14/M4总体责任不提前关闭。

## Review Focus

1. URL大小写/省略协议/显式非HTTP scheme与path/query含`://`混淆、nullable title丢省略状态：T1/T2的高熵多字节实存/真实hash碰撞与精确字符串/三态表，T3/T4创建修改201/200/409/422边界。
2. 标题独改覆写并发抓取的新metadata、旧URL任务覆盖新URL、删除后同URL新ID误命中：T4库内条件更新与T13真实交错。
3. 只检查DNS第一个答案、Unmap漏外层拒绝、连接二次DNS、重定向/代理/userinfo泄漏：T6独立地址表与T7固定IP/HTTPS/跨站反例。
4. 图标成为独立不受限出口、五秒预算重置、超量body/SVG落库，或诊断日志泄露URL凭据/query/body：T7长度/期限与T9共享预算、类型/边界及捕获slog哨兵断言。
5. 事件发出但没原事务投递、漏登记worker、registrationRows跳过请求任务、测试认错历史completed：T11真实提交回滚、T12独立目录、T14按linkID+URL定位job。

## 文件职责、接口与规模

路径相对worktree：I=`server/internal/modules/issue`，B=`server/internal/bootstrap`，O=`server/internal/platform/outbound`，C=`server/internal/platform/config`。完整文件表、16任务与设计12项映射见spec§2.1，每一Task下再列精确修改点。

T1–5链接请求、T6–9安全网络、T10–13任务集成、T14故事、T15门禁、T16文档，各自可独立评审。T1至T3的暂时无API填充只用存储测试，不改最终删除目录断言；T6地址策略与T7真正网络执行分开，避免一个Task同时审两个敏感面。P7不含列表link_count。

全局沿用的真实接口：`Locks.run(ctx,actor,write,func(ctx,held)error)` / `Locks.see`；`Clock.Now() time.Time`；`shared.EventPublisher.Publish(ctx,Event) error`；`postgres.Tx(ctx)(pgx.Tx,bool)`；`riveradapter.Inserter.InsertTx(ctx,river.JobArgs,*river.InsertOpts) error`；`jobs.Register[T](river.Worker[T]) *WorkerRegistration`；`issue.EventTypes/EnqueuedKinds() []string`；`issue.Subscribers(Inserter) []shared.EventSubscriber`；`(*Module).Jobs() []jobs.Job`。

新增类型/端口的唯一签名表在spec§2.3–2.7；各Task的Interfaces必须按同一表消费，尤其`UpdateLink(ctx,LinkWrite)(domain.Link,error)`、`LinkCrawler.Crawl(ctx,string) domain.LinkMetadata`、`SetLinkMetadata(ctx,id,url,m)(bool,error)`。不要把本计划的类型名解释成当前已存在实现。

## 通用收尾 G（每 Task 都执行）

1. 先加该Task断言，记录真实红因；编译未定义只证明缺实现，不能冒称性质红。实现后运行表列聚焦命令，记录真实匹配数/退出码/stdout/stderr。
2. 在生产代码施加该Task代表变异，运行最窄能杀死它的测试；恢复原文件字节/hash，再运行恢复绿。禁止用放宽期望/只改测试/环境错误冒充有效变异。错误/无法核实在台账说明，不静默跳过。
3. 生成契约/SQL的Task跑`make gen-go`/`make gen-web`，保留输出文件SHA和行数；提交后gen-check确认无漂移。不可为消除预期未提交生成物差异提前伪报gen-check通过。
4. `git diff --check`与影响包回归，更新ignored `.superpowers/sdd/P7-issue-links/`任务报告/进度/变异恢复证据；只提交本Task精确路径，中文详细message。
5. 独立评审该Task精确BASE..HEAD：要求→质量；局部发现修复/复审；满足本Task门禁后再下一个；逐Task保留聚焦/受影响回归，完整本地门禁按上文负责人调整执行，T15仍汇总验收。

## Task 1：链接表、grants 和两条级联

**Files:** 新`server/migrations/sql/00027_issue_issue_links.sql`、I/adapter/postgres/link_constraint_test.go、testdata/link_url_hash_collision.json；改`server/sqlc.yaml`、`deploy/runtime-grants.sql`、I/app/delete_issue.go、delete_project_rows.go、对应测试/fakes、I/adapter/postgres/delete_issue.go、delete_rows.go、queries/delete_issue.sql、delete_rows.sql及对应存储测试；B/issue_deletion_test.go、project_deletion_test.go、workspace_deletion_catalog_test.go的既有种子。迁移回归另含 `server/migrations/issue_display_test.go`、`schema_test.go`、`issue_schema_test.go`。
**Interfaces:** 消费`IssueDeletion`、`RowBatch`；产生`DeleteIssueLinkRows(ctx,IssueDeletion) error`、`DeleteProjectLinkRows(ctx,RowBatch)([]uuid.UUID,error)`，在两组rowSteps的relations后/subscribers前。

- [x] 写`TestLinkSchema`、`TestLinkDeletionRows`：真实12列迁移中，高熵多字节URL恰2048字符INSERT/UPDATE成功且回读逐字相同；2049字符为23514 CHECK、空URL/metadata数组/未知键/数字值拒绝。长URL以`http://169.254.169.254/`加`rand.New(rand.NewPCG(71,27))`（标准库math/rand/v2）产生的`0x10000 + r.IntN(0x100000)`码点填满2048字符，UTF-8显著超过2704字节，不用重复ASCII掩盖压缩；同issue精确重复INSERT/UPDATE为23P01及issue_links_issue_id_url_key，跨issue/软删重建成功。三层树和creator≠deleter核审计，早删不变。
- [x] 写`TestLinkHashCollision`：使用独立固定实验fixture（issue UUID `0199a2b4-0000-7000-8000-000000000001`，URL `http://collision.example/271150`/`http://collision.example/83166`，hash `-1669361772`），冻结到testdata/link_url_hash_collision.json。先用内置`hashtext(issue_id::text || ':' || url)`重算并断言各hash同值且两原文不同，再经真实约束插两活行并存；fixture只在存储层无任务/无网络，不需要每轮搜索。T2存储复用该固定fixture；T3/T4验证实际API边界，不重复搜索碰撞，也不把固定UUID结果套给任意issue。
- [x] 运行`go test -count=1 -p=1 ./internal/modules/issue/adapter/postgres ./internal/bootstrap -run 'Test(LinkSchema|LinkHashCollision|LinkDeletionRows|TheGrantsFileCoversEveryRelationAndFunction|.*Deletion)'`（server目录），确认新表缺失/遗漏失败，再加12列及精确约束`CONSTRAINT issue_links_issue_id_url_key EXCLUDE USING hash ((issue_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)`，不额外COLLATE，FK索引/grants和两条删除SQL/adapter/port。
- [x] 同一statement先按id取锁再写批次，包含工作项直接和项目/工作区同步过渡；更新目录种子与fakes，不给links永久空行豁免。`make gen-go`后重跑聚焦及grants覆盖为绿。
- [x] 变异M1跳过links级联、M2错审计写者、M3去exclusion或metadata CHECK，M3-btree改回长原文B-tree由高熵URL击穿，M3-hash-only改为只对同一`hashtext(issue_id::text || ':' || url)`建唯一索引，由真实碰撞对击穿：存储/组合各有红，恢复。runtime权限本Task已有grants覆盖，真正HTTP/job运行由T12补。
- [x] **在 T4 检查点补验，非 T1 最初执行：** T1 漏跑完整 `./migrations`；用户要求提前全套后发现历史 display 测试的 FS 边界和全量迁移/目录期望遗漏。限定 display 历史测试仅含至 26，保留 25→26→25→26 的原业务断言；全量独立期望更新至 27、纳入 issue_links 的完整 Up/Down/Up 与 14 项精确约束/索引。`cd server && go test -v -count=1 -p=1 ./migrations` 在 T4 修复轮补跑为绿（14 顶层/167 含子 RUN）；不追记为 T1 已执行。
- [x] G；提交`feat(M4/P7): 新增链接表并接入两条删除链`，说明furnish依赖T3、当前未跑完整E2E。

## Task 2：纯链接规则与读写存储

**Files:** 新I/domain/link.go、link_metadata.go、link_test.go、link_metadata_test.go；I/app/link_ports.go、link_rows.go；I/adapter/postgres/links.go、links_test.go、queries/links.sql；生成sqlc。
**Interfaces:** 产生spec§2.3的Link/NewLink/LinkPatch/LinkMetadata、CheckNewLink/CheckLinkPatch、LinkWrite/LinkDeletion及消费端口；Store.GetLink/Links/LinkURLExists/InsertLink/UpdateLink/DeleteLink实现。`UpdateLink`返回Link，不仅error。

- [x] 写`TestLinkURLNormalization`/`TestLinkTitlePatch`，断言片段：`got.URL == "HTTP://Example.com/P?q=A"`；输入`" example.com/p?next=ftp://x "`得到`"http://example.com/p?next=ftp://x"`；`ftp://example.com`/userinfo/zone/NUL拒绝；bare`127.0.0.1:4321/page`可；T1同构高熵多字节2048通过/2049失败（不是字节数）、255个界通过/256失败，`SetTitle=false`与`true,nil`不同。
- [x] 写`TestLinkStore`：同created_at逆id确保DESC，Get已删false，URLexists排除当前link；标题更新保持数据库已有metadata，改URL清{}；UPDATE RETURNING与SQL结果逐值相同。另用实际issueID的高熵2048字符做InsertLink/UpdateLink完整回读，直接存储2049触发23514，同issue重复INSERT/UPDATE为指定23P01、跨issue/软删复用可、真实hash碰撞不同原文并存。仅插fixture第一条时LinkURLExists对碰撞另一原文为false，对精确原文为true，排除本ID正确；不按摘要判重复。T2排斥冲突原链保留，T3首次公开码时才加domain映射/文案，避免测试要求“契约码=文案”被提前破坏。
- [x] 运行`go test -count=1 -p=1 ./internal/modules/issue/domain ./internal/modules/issue/adapter/postgres -run 'Test(LinkURLNormalization|LinkTitlePatch|LinkMetadata|LinkStore)'`确认红；实现纯域检查、端口与SQL（不对存储URL做额外小写/重序列化），生成后同命令绿。
- [x] 变异M4去协议识别/错误foldURL、M5按字节截长度或丢SetTitle、M6标题改写旧metadata；M6-url把LinkURLExists误改成比较同一hashtext值，固定碰撞fixture必须红；相应域/存储红，恢复。
- [x] G；提交`feat(M4/P7): 固定链接字段语义并实现存储端口`（`5b171386a2117ea8bbbe199ab5acf494e4dd2a4c`，规格与质量独立评审通过）。

## Task 3：创建、集合读取与首次链接事件

**Files:** 改`api/modules/issue.yaml`、`api/openapi.yaml`聚合入口及生成物；新I/app/create_issue_link.go、list_issue_links.go、link_events.go及测试；I/domain/events.go、errors.go、link_activity.go；I/adapter/http/links.go、link_usecases.go、links_test.go、handler.go；module.go/module_events.go；B/events_test.go、issue_links_test.go、permission_matrix_links_test.go、issue_write_locks_test.go/gate_test.go/issue_writes_test.go；新e2e/fixtures/links.ts，改furnish.ts、assert/issue-collaboration.ts、assert/issue.ts、stories/issue/i15-archive-issue.spec.ts、i17-delete-issue.spec.ts；错误文案三文件；I/adapter/postgres/links.go、links_test.go、link_conflict.go、link_conflict_test.go补精确冲突映射。
**Interfaces:** 消费T2 LinkCreator/LinkLister、现有Locks/ReadTx/Clock/Events；产生Create/List的构造/Execute签名（spec§2.4）；HTTP UseCases同签名。声明EventLinkCreated/Updated/Deleted目录（后两发布点T4）；`stamp.linkEvent(kind string, id uuid.UUID, changes []shared.Change) shared.Event`集中封装Entity，业务字段Changes由用例明确构造。

- [x] 写`TestCreateIssueLink`/`TestListIssueLinks`及HTTP双向错误覆盖：201metadata{}、NULL title/creator形状；重复URL409而其他issue同URL成功；列表排序/空data；guest OwnOnly=true时仅自己可见、guest_view_all_features=true/OwnOnly=false时他人可读但普通guest仍不可写。创建活动四槽、事件url/title nil和值、Event.ID≠linkID和同一时刻逐值断言。高熵多字节2048字符创建201/DB原文同值，2049为422 url/not_allowed且无写；精确重复409，跨issue同URL201，T4删除接口可用后再补软删重建201。真实hash碰撞只由T1/T2固定存储fixture证明，API测试不另搜索或创建外网碰撞地址。
- [x] 写`TestLinkConflictMapping`：真实约束23P01映射ErrLinkExists，并以包装PgError表测“正确码+正确名”唯一映射；其他名23P01、同名23505、其他23505与非PG错误均保持原链。`errors.As`解包、不按全23P01/23505粗映射；同步将T2存储重复期望改为ErrLinkExists，原始SQLSTATE仍由T1直接SQL证明；完整API重复测试不能替代这层。
- [x] 迁移B/events_test.go的`TestIssueEventDirectory`：独立期望现有12类加link_created/link_updated/link_deleted共15类及精确顺序/无重复/防外部切片变更；移除future表的link_created，保留project.deleted/workspace.deleted未知拒绝。将目录专用dispatcher改为`newDispatcher(types,nil)`，空payload仅测目录，不调用生产Subscribers；生产订阅注册/真实payload/事务证明归T11。
- [x] 聚焦`go test -count=1 -p=1 ./internal/modules/issue/app ./internal/modules/issue/adapter/http ./internal/modules/issue/adapter/postgres ./internal/bootstrap -run 'Test(CreateIssueLink|ListIssueLinks|LinkConflictMapping|IssueEventDirectory|.*Link.*Matrix|.*IssueWrite|.*Gate)'`先红，实现Locks链及ReadTx；添加`ErrLinkExists`、仅`Code==23P01 && ConstraintName==issue_links_issue_id_url_key`映射、契约x-problem-codes、PROBLEM_MESSAGES与en/zh-CN同Task。
- [x] 新fixture用生成operation类型；`furnishIssueRows`通过API加link，使用`http://169.254.169.254/furnish/<issue-id>`（后来worker会拨号前blocked），不请求真实外网。扩I17现有种子为link，尚不增加页面UI。 **T4 完整本地 E2E 发现并补验：** I15 同样调用严格 `expectIssueDeleted`，先前漏了 link API 种子；修复轮补入 ann 创建/admin 删除的 link 与创建活动逐值期望，保留目录守卫，不记为 T3 最初已覆盖。
- [x] `make gen-go`、`make gen-web`后聚焦绿，`make test-web`与`pnpm --filter @nerve/e2e check:types`绿。现有权限矩阵/写锁/闸门覆盖表随operation立即补行，不等T5。
- [ ] 变异M7去访客隐藏或MarkChanged、M8漏创建title Change/Entity错ID、M9只查未规整URL；另M8-directory删除任一link事件声明，独立15类目录必须红；M9-map去约束名或只认23505，错误映射对照必须红；app和至少一条真实组合红，G；提交`feat(M4/P7): 提供链接创建与集合读取接口`（实现与七项生产变异已完成，独立评审待主控安排）。

## Task 4：链接改删、metadata 原子性与活动事件

**Files:** 新I/app/update_issue_link.go、delete_issue_link.go、link_target.go及测试；I/adapter/http/link_writes.go/test；改domain/link_activity.go、errors.go、api/modules/issue.yaml、module.go/HTTP handler与生成物；B/issue_link_writes_test.go、矩阵/锁/闸门行；三文案文件。
**Interfaces:** 消费T2 LinkUpdater（UpdateLink返回Link）/LinkDeleter；产生Update/Delete Execute及Deps见spec§2.4；`linkTargetCheck(links LinkReader,id uuid.UUID,current *domain.Link) func(context.Context,domain.Issue,shared.Grant) error`仅检查活资源/父项归属，不复制comment作者限制。

- [ ] 写`TestUpdateIssueLink`/`TestDeleteIssueLink`：成员可改删他人link；标题null/空/省略，规整后相同URL noop；项目/issue归档均先拒绝noop；URL换后{}，title-only保持metadata；业务行/动态/事件审计精确，title-only只有updated事件、无url Change/无活动。高熵多字节2048字符更新200且DB/响应原文相同，2049为422无写；把另一个link改为已用长URL为409，整笔审计/metadata/活动/事件回滚；跨issue同URL200，软删后创建/修改为原URL成功。不同原文hash碰撞的写/查性质由T1/T2固定存储fixture证明，本Task复用同一原文查重端口。
- [ ] `go test -count=1 -p=1 ./internal/modules/issue/app ./internal/modules/issue/adapter/http ./internal/modules/issue/adapter/postgres ./internal/bootstrap -run 'Test(UpdateIssueLink|DeleteIssueLink|LinkStore|.*Link.*Matrix|.*IssueWrite|.*Gate)'`先红；实现预检与锁下重读和拒绝顺序，返回Store.UpdateLink的实际行；新增link_not_found契约/文案同Task。
- [ ] domain活动构造单点固定17–19与comment字符串`created a link`/`updated a link`/`deleted the link`；删除活动NewValue指向空串而非nil，事件删除New=nil，避免混用活动与事件规则。
- [ ] make gen-go/gen-web、聚焦与make test-web通过。新公开码与契约同时出现；不修改既有安全守卫。
- [ ] 变异M10取消锁下link重读、M11给title-only添url Change/活动、M12混淆删除空串与nil；必要语义反例红，恢复，G；提交`feat(M4/P7): 完成链接改删及原子审计事件`。

## Task 5：完整请求安全、授权交错与回滚证明

**Files:** 新B/issue_link_security_test.go、issue_link_transactions_test.go；扩现有issue_world_test、permission_matrix_links_test、issue_write_locks_test、gate_test；必要app异常测试。复用已有辅助函数。
**Interfaces:** 只消费T3/T4生产API与真实Locks/事务；不产生第二套授权或锁实现。

- [x] `TestIssueLinkHiddenAnswers`按合法输入逐值比较不存在/OwnOnly=true时别人issue/已删父项，含管理员兼访客；再将guest_view_all_features=true证明OwnOnly=false时他人可见，普通guest写403、workspace admin兼guest可写，拒绝来自ActionUpdate而非复制角色谓词。`TestIssueLinkWritesLockAndRecheck`经RowWaits证明第一锁project N、无workspace冲突锁、预检拒绝不进gate/tx，等锁期间降级/移出后按新授权失败。新增同项目两并发创建/改为同URL，观察第二条等待既有项目N锁再进排斥检查；一条成功、另一条409，无40P01。最窄PG实验没有替代这条串行性证明。
- [x] `TestIssueLinkWriteRollback`以真实PG在MarkChanged后、link写后、activity后、publisher失败逐点失败，`before == after`比较所有业务/活动（T11之后另含任务）；两用户/两issue/两个link防止“只一行”空过。
- [x] 先跑`go test -count=1 -p=1 ./internal/bootstrap -run 'Test(IssueLinkHiddenAnswers|IssueLinkWritesLockAndRecheck|IssueLinkWriteRollback|.*Link.*Matrix)'`；另跑真实矩阵 `go test -count=1 -p=1 ./internal/bootstrap -run 'TestPermissionMatrix/(prepare|listIssueLinks|createIssueLink|updateIssueLink|deleteIssueLink)'`（既有矩阵需要显式选择 prepare；`.*Link.*Matrix` 不匹配该入口）。若现有实现已绿，记录这是补性质覆盖，不编造红，再用对应生产变异验证会红。
- [x] 变异M13绕过precheck/gate、M14在事务前读Clock或吞Publish错误、M15错误作者限制；组合层红→恢复绿；对新增交错用`go test -race -count=5 -p=1 ./internal/bootstrap -run 'TestIssueLinkWritesLockAndRecheck'`，无40P01。
- [x] G；提交`test(M4/P7): 钉住链接写的权限锁序与原子回滚`。

## Task 6：显式地址策略和独立 IANA 清单

**Files:** 新O/policy.go、special_ranges.go、errors.go、policy_test.go、policy_registry_test.go、testdata/iana-ipv4-special-registry-1.csv、testdata/iana-ipv6-special-registry-1.csv、testdata/sources.json（官方URL/访问时刻/SHA）；不得生成第三方新模块。
**Interfaces:** 产生包内`newPolicy(allow []netip.Prefix,local []netip.Addr) (policy,error)`、`(policy).check(addr netip.Addr) error`（拒绝识别ErrBlocked）；ErrBlocked的定义放O/errors.go供T7扩；不提供公网skip检查出口。

**T6 实施事实（待独评）：** 独立 CSV 与 13:59 冻结前重核 manifest 逐字节/SHA 一致，清单 51+3 项与所有首尾分别验证。allow 仅覆盖原 literal 的静态/本机拒绝，不传给已知编码的内嵌 IPv4；固定 NAT64 `/96` 解码，技术无关 local `/48` 只作整段拒绝，不猜协议/位位置（权威来源见 spec §2.5）。纯地址策略与 registry 解析测试分文件；真实 GET/接口读取失败/0 dial 留 T7，未提前完成阶段验收。

- [ ] 写`TestPolicyRegistryCoverage`：独立CSV每CIDR逐个与生产拒绝清单对照（另核额外multicast/fec0），`TestPolicyEndpoints`独立算各prefix两端；任何rawmapped不能先Unmap脱离外层；首/尾都阻止。冻结时再次只读核官方快照，变化写来源/代价而非忽略。
- [ ] 写`TestPolicyEmbeddedAndLocal`：mapped/NAT64/6to4/Teredo含内网/公网载荷、Teredo server/client两端、本机普通公网、非法地址/zone；allow127.0.0.1/32仅放这一literal，127.0.0.2与mapped127不放；验证allow/local切片外部修改不影响已建policy。
- [ ] `go test -count=1 ./internal/platform/outbound -run 'TestPolicy'`先红，按spec§2.5实现显式所有prefix与递归检查，恢复绿；生产表不能在init/global可变slice构造。
- [ ] 变异M16只用IsPrivate、M17删除非重叠prefix/删重叠条目、M18先Unmap/忽略本机；前者行为红，重叠删除由独立清单红，不虚称每行删除都会改变行为。
- [ ] G；提交`feat(M4/P7): 建立可核对的出站地址拒绝策略`。

## Task 7：固定 IP 的安全 GET 与资源上限

**Files:** 新O/client.go、transport.go、client_test.go、transport_test.go、deadline_test.go；扩errors.go。测试私有依赖用于resolver/dial/interface/TLS root，无全局替换/可变测试hook。
**T7 实施事实（待独评）：** 完成固定IP/每跳复检、Host/SNI与证书验证、手动跳转、解压后限额、共享期限与显式连接关闭；M19–M22七个生产变异均红后按原字节恢复绿，T6 mapped端点守卫Minor以值比较和临时helper Unmap自证闭环。仅T7实现与聚焦门禁完成，T8配置/T9日志消费与整阶段验收未完成。

**T7 修复轮1（待定向复审）：** 修复Unicode hostname被自定义DNS/TLS绕过IDNA规范化的问题：复用Transport交给dial callback的规范地址，在同一边界解析/检查全部候选，再固定IP建连；Host/SNI/证书按同一规范名，保留原URL语义与分类。新增受信任punycode SAN本地HTTP/HTTPS/跳转反例与晚到连接关闭覆盖。

**Interfaces:** 产生spec§2.5的Options/Limits/Response/New/Get与四错误、`Reason(error) string`（仅空或redirect_limit，支持标准包装）；New在生产读取InterfaceAddrs，测试私有`newClient(options Options,deps dependencies)`中依赖：resolver的`LookupNetIP(ctx,network,host)([]netip.Addr,error)`、dial函数、interface读取、TLS roots（只测试），不暴露给模块消费方。

- [x] 写`TestOutboundPinsEveryHop`：混合public+blocked答案断言`dialCalls==0`；resolver第二次返回127但同hop只解析一次、dial目标为第一次publicIP；同host重定向重新解析；HTTPS记录Host/SNI，正确证书成功、错证书失败，不能测试依靠InsecureSkipVerify。
- [x] 写`TestOutboundLimitsAndHeaders`：最多5redirect，第6次返回ErrUnreachable且Reason返回redirect_limit，日志与crawler failed映射由T9验证；redirect userinfo/zone/nonhttp无连接且Blocked；代理env的httptest计数0；Set-Cookie后跳转Cookie为空，Authorization/Proxy-Authorization/Referer空；MaxBytes恰好N成功，N+1/chunked/解压展开超量TooLarge，取消/解析/读body也受总期限，中间body关闭。`TestOutboundLocalSnapshot`在outbound同包私有依赖中使InterfaceAddrs返回错误，断言New失败；本机普通公网地址拒绝。
- [x] `go test -count=1 ./internal/platform/outbound -run 'TestOutbound'`初始缺接口编译红单列；连接生命周期两项真实断言红与下列生产变异红分别留证；实现manual redirects、先全候选检查后拨号、Host/SNI保持、Proxy=nil、无CookieJar；同Get共用ctx，连接2s/总5s由Limits传入，用户更短ctx优先，响应资源关闭。
- [x] 变异M19二次DNS/只首IP、M20去redirect重检/允许第6跳、M21ProxyFromEnvironment、M22仅信Content-Length/每跳重开期限；对应本地HTTP/HTTPS测试红→恢复绿。期限测试用阻塞通道与ctx done，不sleep。
- [x] `go test -race -count=1 ./internal/platform/outbound`与archtest通过；G；提交`feat(M4/P7): 实现固定地址与限额的安全出站请求`。

## Task 8：测试专用 CIDR 配置与构造接缝

**Files:** 改C/config.go、load.go（既有listHook中仅收紧`[]string`输入形状）、validate.go、load_test.go、validate_test.go；新C/outbound_test.go；server/configs/config.yaml、config.test.yaml、embed_test.go；新B/outbound.go、outbound_test.go；server/cmd/nerve/main_test.go；server/internal/archtest/composition_test.go（原CLI组合守卫补`outbound.New`）。
**Interfaces:** 产生`config.OutboundConfig{AllowCIDRs []string}`和Config.Outbound；`newOutbound(cfg config.Config,version string)(*outbound.Client,error)`；T10生产调用传`buildinfo.Get().Version`（bootstrap可import两个platform包，outbound自己不互导）。

- [ ] 写`TestOutboundAllowCIDRsConfig`表：test空/127.0.0.1/32/IPv6合法通过；非test非空各env失败；坏CIDR及JSON/env形状失败且不悄悄当空。按现有koanf路径`NERVE_OUTBOUND__ALLOW_CIDRS`验证实际切片解码，不另创env协议。
- [ ] 写`TestNewOutbound`：直接传坏CIDR配置也构造失败，test127能访问本地fixture而未放行127.0.0.2不可，版本传到UA；既有`Config.LogValue`/slog配置日志与配置文档不泄漏凭据，不新增普通fmt渲染协议。接口采样失败由T7同包私有依赖测试，不给bootstrap增加生产注入开关。
- [ ] `go test -count=1 ./internal/platform/config ./configs ./internal/bootstrap -run 'Test(OutboundAllowCIDRsConfig|NewOutbound|.*Config)'`确认红，追加默认空/test127.0.0.1/32、验证与构造；同命令绿。T8只交真实测试消费的构造接缝，T10接server；不在CLI/Provide构造。
- [ ] 变异M23去test-only限制/解析错误忽略、M24把allow扩大127/8或读取接口失败当空；配置/组件红→恢复，G；提交`feat(M4/P7): 限定测试环境的抓取放行配置`。

## Task 9：页面和图标抓取适配器

**Files:** 新I/app/crawl_ports.go；I/adapter/crawler/crawler.go、metadata.go、crawler_test.go、metadata_test.go、logging_test.go；使用已有x/net/html，不改go.mod。
**Interfaces:** 产生`LinkCrawler.Crawl(ctx,string) domain.LinkMetadata`；`crawler.New(*outbound.Client,*slog.Logger)*Crawler`及同签名Crawl。以errors.Is分类，`outbound.Reason(err)`仅返回固定redirect_limit时写reason，禁止解析原error文本。不把outbound.Response或错误放入app/domain。

- [ ] 写`TestCrawlerMetadata`：本地HTML `&amp;amp;`标题只解一次后`&amp;`，前后空白trim，256个Unicode截255；无title为空；按四rel优先级/同类首href/无候选favicon.ico；redirect最终页的相对href，成功metadata四键齐，url仍原任务URL，favicon_url成功最终URL。
- [ ] 写`TestCrawlerLimitsAndFailures`：页面1MiB/1MiB+1，图标64KiB/64KiB+1、六mime/参数/非允许SVG；图标blocked/失败保留title而两icon字段null；页面blocked仅{error:blocked,url}、其余仅failed；整个crawl共享5s而非页面+图标10s。
- [ ] 写`TestCrawlerLogsRedacted`：用开启所有级别的捕获slog handler检查原始JSON日志bytes，query=`QUERY_SENTINEL`、userinfo=`USER_SENTINEL:PASS_SENTINEL`、body=`BODY_SENTINEL`，重定向Location与icon href另用独立哨兵。真实本地fixture分别触发页面连接/超时/超量/状态/类型失败、redirect被拒/超限、图标blocked/超量/坏MIME；每次断言哨兵/原始URL/含URL原始error不在消息或任一属性，同时诊断非空且stage为page/icon，category对应blocked/timeout/unreachable/too_large/http_status/non_html/invalid_icon_type，redirect超限另保留reason=redirect_limit。禁止靠静默日志通过，禁止只检查metadata或HTTP响应。
- [ ] `go test -count=1 ./internal/modules/issue/adapter/crawler`先红，按spec§2.6实现；网络错误仅记受控阶段/细分类，原始URL/query/userinfo/body及未经分类的error字符串均不入日志；图标同outbound，不HEAD、无脚本/refresh/HTMLbase；同命令绿。
- [ ] 变异M25图标绕outbound、M26存SVG/漏64KiB、M27双重反转义/重开5s/细分公开error；另M27-log在生产失败分支临时记录原始URL，及在含URL包装错误的分支临时记录原始error，两次分别运行`go test -count=1 ./internal/modules/issue/adapter/crawler -run TestCrawlerLogsRedacted`确认哨兵红；若该错误本身不含哨兵须更换真实输入，不能伪造失败。逐次恢复原字节/hash并同命令绿。httptest可区分结果、0危险dial；恢复绿，G；提交`feat(M4/P7): 抓取受限的标题与内嵌图标元数据`。

## Task 10：守卫写回、worker 与 server 组合

**Files:** 新I/app/crawl_link.go/test；I/adapter/postgres/link_metadata.go/test、queries/link_metadata.sql；I/adapter/river/crawl_link.go/test；改module.go/module_jobs.go、B/app.go/app_test.go；archtest/composition_test.go扩CLI出站构造禁达（不弱化旧规则）。
**Interfaces:** 产生spec§2.7的LinkMetadataWriter/SetLinkMetadata、CrawlLink/CrawlLinkArgs/CrawlLinkUseCase/CrawlLinkJob。`issue.Deps`新增`Outbound *outbound.Client`、`Logger *slog.Logger`（模块root可见技术），Module新增crawlLink；app仅见Crawler/Links端口；EnqueuedKinds返回crawlkind。

- [ ] 写`TestLinkMetadataGuard`：匹配活id/url写1，旧URL/已删/另ID写0，title/created_at/updated_at/updated_by/deleted_at以及activity/event均不变；数据库错误errors.Is原链；`TestCrawlLinkWorker`核args.Kind/default/MaxAttempts3、网络结果成功、DB失败返回error。
- [ ] `go test -count=1 -p=1 ./internal/modules/issue/app ./internal/modules/issue/adapter/postgres ./internal/modules/issue/adapter/river ./internal/bootstrap -run 'Test(LinkMetadataGuard|CrawlLinkWorker|.*Job|.*NewApp)'`先红，写一条带守卫UPDATE、app编排、worker/Register；接newOutbound→issue.New→Jobs，任一步构造失败close已有pool，CLI仍不构造网络/dispatcher/jobs。
- [ ] MaxAttempts在`CrawlLinkArgs.InsertOpts() river.InsertOpts`固定default/3，T11显式投递opts同值，单一常量`CrawlLinkMaxAttempts=3`避免漂移。新worker不读项目、不取项目锁，无metadata审计事件。
- [ ] 变异M28去URL/deleted守卫或改updated_at、M29吞DB错误/漏Jobs登记；存储/worker/构造层红，恢复；`go test -count=1 ./internal/archtest`绿，G；提交`feat(M4/P7): 接入抓取任务和带条件的元数据写回`。

## Task 11：首个生产订阅者与原事务投递

**Files:** 新I/app/crawl_on_link_change.go/test；I/adapter/river/link_crawls.go/test；改module_events.go；B/events_test.go保持T3的独立目录测试；新B/issue_link_jobs_test.go，复用issue_inserter及现有真实River probe。
**Interfaces:** 产生`LinkCrawls.Enqueue(ctx,linkID,url) error`、`NewCrawlOnLinkChange(LinkCrawls) shared.EventSubscriber`、river.NewLinkCrawls(Inserter)；issue.Subscribers旧签名不变，返回唯一生产subscriber。

- [ ] 写`TestCrawlOnLinkChange`，created和含url的updated各投一次，title-only零且成功；Entity.ID与Event.ID故意不同，抓参数是linkID和Change.New；created缺url、两种订阅类型重复url/url项nil New返回程序错，updated没有url不是错误；不订阅deleted。
- [ ] 新`TestIssueLinkSubscriberRegistration`在B/issue_link_jobs_test.go以`jobs.NewInserter(pool)`产生的client构造生产`issue.Subscribers(issueInserter{client:client})`并组合真实dispatcher，断言唯一生产subscriber只订阅created/updated；使用含独立Event.ID、`Entity:{Kind:"issue_link",ID:linkID}`、url Change的真实payload，在真实事务中经Inserter投递。目录的接受/拒绝仍由`TestIssueEventDirectory`无订阅者验证，不能继续用空`shared.Event{Type:kind}`调用生产subscriber。
- [ ] `TestLinkJobCommitRollback`真实PG/River：另一连接提交前0job，commit后正确args+业务，rollback后两者均无；标题独改确实提交新title/audit且没有新job；created缺url或重复url失败时业务回滚；第一subscriber入队第二失败第三不调用，全部业务/活动/job回滚；不以“没启动worker”替代事务证明。
- [ ] `go test -count=1 -p=1 ./internal/modules/issue/app ./internal/modules/issue/adapter/river ./internal/bootstrap -run 'Test(CrawlOnLinkChange|IssueLinkSubscriberRegistration|LinkJobCommitRollback|IssueEventDirectory)'`先红，接现有Inserter.InsertTx，Queue default/3attempt，无独立事务、无网络、无锁、无异步goroutine。
- [ ] 变异M30InsertTx换独立投递/丢ctx、M31错误urlChange条件/吞enqueue错误；真实组合层红→恢复绿；确认`issue.EventTypes`有15类型且只预期新增3，G；提交`feat(M4/P7): 由链接事件在原事务内投递抓取任务`。

## Task 12：请求任务目录与运行角色真实完成

**Files:** 改B/invitations_test.go（registrationRows的实际定义）、table_rows_test.go、runtime_role_test.go、jobs_test.go；新B/issue_link_runtime_test.go、request_jobs_test.go（按大小拆职责）。
**Interfaces:** 保留`registrationRows(t,*pgxpool.Pool) string`调用方；内部追加声明请求kind的river_job稳定id/kind/args快照，不把state/attempt/finished时间等worker变化列写入。独立预期kind常量指向`issue.crawl_link`，不从实际Jobs反推预期。

- [ ] 写`TestRegistrationRowsSeeRequestJobs`：插入一个crawl任务前后不同，改变args也不同；periodic/leader变化和同请求job状态变化前后相同；请求kind声明遗漏的独立预期检查失败。沿既有被拒注册用例核快照不变。
- [ ] 写`TestIssueLinkRuntimeJob`：恰好grants权限启动真实server/River，以API建本地页面link，DB/HTTP metadata与completed一致；再改标题/URL/删、含清单以外普通表反例，不以单纯SELECT成功冒称worker可写。
- [ ] `go test -count=1 -p=1 ./internal/bootstrap -run 'Test(RegistrationRowsSeeRequestJobs|IssueLinkRuntimeJob|TheGrantsFileCoversEveryRelationAndFunction|TheRuntimeRoleServesWithTheGrantsFile|TheRowChecksSeeEveryTable|RegisteringWithAnInvitation)'`；缺覆盖先红或生产变异红，扩现有目录而非复制第二套注册快照；恢复后全绿。
- [ ] 变异M32去请求job快照、M33快照只count不含args、M34删link UPDATE授权/漏worker；独立反例与运行角色红→恢复，G；提交`test(M4/P7): 覆盖请求任务目录及最小权限抓取`。

## Task 13：抓取与 URL、标题及删除的真实交错

**Files:** 新B/issue_link_interleaving_test.go、issue_link_race_helpers_test.go；必要扩已有pgtest辅助的消费，不造平台同步框架。只修被反例证明的问题。
**Interfaces:** 消费T4 UpdateLink返回Link、T10 SetLinkMetadata、T11真实投递。同步crawler fixture实现已有LinkCrawler，不向生产加屏障API。

- [ ] 写`TestLinkCrawlURLInterleaving`：老A抓取停在网络返回前，API改B提交+清{}+留B任务，放老A令0写，再B完成；反向先A写再改B依然{}。最后断言metadata.url属于B、活动只有URL更新、两job归属精确。
- [ ] 写`TestLinkTitlePreservesConcurrentMetadata`：标题请求已读pending、抓取先更新metadata、标题写返回值与DB都保留新metadata；相反顺序亦成立；真实事务/锁条件证明控制点，不只fake call log。`TestLinkCrawlAfterDeletion`：先删/新ID重建同URL旧任务写0；A→B→A旧A能写的已批准限制单独命名，不能偷加版本条件。
- [ ] `go test -race -count=5 -p=1 ./internal/bootstrap -run 'Test(LinkCrawlURLInterleaving|LinkTitlePreservesConcurrentMetadata|LinkCrawlAfterDeletion)'`实际匹配每轮所有用例，无40P01/超时泄漏；两连接等待用RowWaits/通道与期限。
- [ ] 变异M35去urlguard、M36标题整体写回旧metadata、M37匹配只url不id；真实交错必须红，恢复同一聚焦普通运行绿，之后只受改动影响再race。G；提交`test(M4/P7): 证明抓取与链接修改删除的交错语义`。

## Task 14：I10 接口故事与带链接的删除故事

**Files:** 新e2e/fixtures/link-page.ts、stories/issue/i10-links.spec.ts；扩fixtures/links.ts、furnish.ts；改stories/issue/i15-archive-issue.spec.ts、i17-delete-issue.spec.ts、i17-catalog.spec.ts、必要assert/issue.ts（复用activityRow）；不加页面实现。
**Interfaces:** fixture `startLinkPage(): Promise<LinkPage>`，`LinkPage{url:string; release():void; close():Promise<void>}`提供受控首个页面请求/后续不同标题路由，本测试生命周期拥有server并在finally关闭；API fixture直接使用生成类型。

- [ ] 写I10：API建普通用户/项目/issue→输入127.0.0.1:<port>/page→201URL带http且metadata{}→受控放行后四键metadata/内嵌data图标→DB对应linkID/url/kind的job completed→标题独改无新增任务→URL更新清{}后抓新标题→重复409→删除活动。blocked例169.254.169.254只断言metadata.error，不声称E2E观测0连接。
- [ ] 扩I17两层子树、不同creator/deleter、已归档子项与先删对照；API填每个受测links，目录断言同刻同写者。I15 的归档后删除也必须 API 填 link 并纳入活动期望（T4 检查点全套发现遗漏后补入），项目/工作区的furnish也有link，不能只改I17而漏同步cascade。
- [ ] 先`pnpm --filter @nerve/e2e check:types`，`make build`后`pnpm --filter @nerve/e2e exec playwright test stories/issue/i10-links.spec.ts stories/issue/i15-archive-issue.spec.ts stories/issue/i17-delete-issue.spec.ts stories/issue/i17-catalog.spec.ts --project=chromium`；继承已验证项目内浏览器缓存，移除NO_COLOR/FORCE_COLOR，不全局安装。
- [ ] 故事按API/DB/job三层断言，轮询有期限、任务按本linkID/url匹配；先停server/失败也清理。选M30/M35等已有真实生产变异中最合适一项做故事归属验证，不为凑数重复全部变异；恢复绿，G；提交`test(M4/P7): 加入链接抓取故事并扩展删除目录`。

## Task 15：全链路矩阵、变异归档与最终门禁

**Files:** 新`docs/v0/M4-issue-core/reviews/P7-issue-links-behavior-matrix.md`、`P7-issue-links-mutation-matrix.md`；补真实跨层缺口的既有测试；原始输出仅ignored台账。
**Interfaces:** 无新产品接口；消费T1–T14最终实现和报告。

- [ ] 逐行核四operation/三活动/三事件/订阅条件与无副作用例外；每行标测试、层、精确HEAD，含四槽nil/value、metadata pending/success/error shape、全部拒绝码双向覆盖。检查最终不存在临时emptySubscriber/fakecrawler/绕过目录的过渡。
- [ ] 从M1–M37挑至少15个实际生产变异，每类地址/网络/权限/锁/事务/guard/注册至少一个，复用已录raw证据；每个都必须实际红→原字节恢复→绿。若因安全表重叠由清单守卫杀死，明确层和原因，不说HTTP行为因此变。
- [ ] 串行`GOFLAGS=-p=1 make test`、`make lint-go`、`make lint-web`、`make knip`、`make test-web`、`make e2e`；记录测试包/匹配/退出码、browser数量，不把缓存数或Infra失败当通过；不得操作已有容器来“修环境”。
- [ ] 带生成物提交后`GOFLAGS=-p=1 make gen-check`无差异；库/code/hash与证据一一对应。已知W11不专门调查；若阻塞合并CI且重跑仍不过按用户纪律报告。
- [ ] G；提交`test(M4/P7): 完成链接全链路门禁与性质证据`；后续deep-review不无理由重复完整套件。

## Task 16：deep-review、正式记录和阶段交付

**Files:** 本spec/plan/review三文档、两矩阵；M4-design §3.11/§4.8（hash exclusion局部物理修正）、§3.23/§15、v0-design §6.2/§8、plane-diff的issue_links12列/抓取差异、README安全；M2/M3 handoffs与P6 spec§5/review§6接收结果；整审报告`docs/reviews/re-NN-M4P7整体审核.md`按现存编号。
**Interfaces:** 无新产品接口；每个结论引用真实已提交实现/证据。

- [ ] 全任务独评完成后用deep-review审准确基线..HEAD整支：全文文件清单、逻辑/集成/测试覆盖，包含全部请求写死锁推导、网络边界、事件/任务原事务、文档事实。必要局部修复/测试和聚焦复审后再判通过；结构问题遵守先提交后上报。
- [ ] 写review七节（方式、逐项证据、局部决定、发现处理、最终与plan差异、移交、限制）；spec§4只勾真实完成项，Task实际签名/文件差异同步，不为“表面按plan”掩盖实现修正。
- [ ] 同步总体平台/安全说明、逐列差异与README网络边界/限额/公网任意端口/NAT和网卡变化需网络层覆盖；P7不改CSP，不写已存在页面。输入交接status保持open，追加本期结果与剩余P9/P14责任。
- [ ] M4§14只补P7子项真实证据，不把“事件和任务”整条或M4总体完成；v0§9.4仍进行中，不创建M5–M8最终closeout。
- [ ] 文档链接/代码路径/签名/checkbox范围和`git diff --check`通过，提交`docs(M4/P7): 记录阶段终审与链接交付证据`；推分支准确HEAD并确认其CI，提供分支/基线/spec/plan/review/终审路径，先不合并。
- [ ] 后续得到授权才--no-ff合并、准确main SHA CI全绿、ignored原始证据归档验证、清理分支/worktree；不能用分支CI替代mainCI。

## 准备期自查与交付边界

16 Task覆盖设计12粗项，接口生产/消费与方法名逐项核对，UpdateLink始终返回Link，五项Review Focus均有归属测试；spec所有验收保持未勾。当前准备文件已通过独立安全与集成预检及定向复审，5项Medium全部关闭；尚无P7完整原型/逐Task重放、产品运行或生产变异证据，只有spec附录界定的4列PG约束/sqlc最窄实验。所有产品验收仍未勾，真实基线/mainCI与预检结论见review；等待负责人安排启动T1。
