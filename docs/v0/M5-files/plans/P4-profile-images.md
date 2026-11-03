# M5 / P4 资料图片与引用实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 预检通过，T1–T12实现及验收完成，T13七项与准确产品CI已全绿，T14整审通过、最终文档CI待核。一次fresh builder check及主控plan自检完成；基线main5807a0d6，CI41591764345全绿，P3已归档清理。分支codex/m5-p4，worktree为`/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m5-p4`。连续推进授权有效；按T13接续，重大新风险先提交后上报。

**Goal:** 用户/工作区/项目的图片引用、一次项目创建绑定及授权图片URL真实工作，完成F5–F7 API。

**Architecture:** owner拥有当前引用、授权/父锁和事务；asset只批量处理自己的行并CPU签发。bootstrap唯一接线，反向引用由owner解除后资产才物理删除；不动P2父保护和M4权限守卫。

**Tech Stack:** 既有Go1.27/PostgreSQL18/sqlc/OpenAPI/River/Playwright；P3已有local和S3兼容服务。无新依赖。

**Spec:** [主spec](../specs/P4-profile-images.md)、[绑定/读取](../specs/P4-image-access.md)、[清理](../specs/P4-reference-purge.md)，三份同为实施合同。

## Global Constraints

- 主目录永远main；独立codex/m5-p4 worktree。详细中文提交feat/fix/docs/test(M5/P4)，系统/usr/bin/git正常push；无force/stash/clean/reset--hard。
- plane/refer只读；不动已有Docker或容器、不全局安装；容器套件串行，跨Go包容器命令用-p 1。依赖锁不改。
- 约400行以内的手写职责文件；app/domain不引入SDK/HTTP/配置/跨模块import，所有SQL经自身sqlc；无跨模块JOIN、任意事务闭包、Setter或运行时服务发现。
- 用户两图、workspace logo、project cover的旧/新资产并集UUID升序；账号第一把，W→P→asset，资产之后不追既有owner锁；签发只CPU，任何对象I/O在事务外。
- 7天准备件严格Before才过期；60天物理cutoff与metadata空沿P2；URL runtime期限5..65分钟、过期秒精确；业务时刻在最后锁后读，同一次写共用At/actor。
- cover-01..cover-29精确枚举、互斥；不接受构建URL，不实现P6素材/上传UI。Workspace.organization_size六个现有值生成类型，可null。
- 资产读取每SQL块最多1000ID、先全局有序锁后签；资产引用purge页最多min(batch,1000)，输入config范围不改；不截断已有整集合列表。
- 每Task实际读brief，记录RED/GREEN、退出码/命中测试、变异断言与恢复hash；无测试/编译失败不当行为证据。首次完整产品检查点T13，任务中间提交不声称全阶段可交付。
- T13完整七项；T14一次deep-review三视角+fresh整分支，Important/Critical一次修复RED/GREEN及必要全量复核，Minor具名移交。精确branch/main CI与归档分别核验。

## Review Focus

1. 同一asset UUID在批量输入里对应不同purpose/宿主，不能按ID去重后把合法能力发给非法引用（T3具名反例）。
2. 成员头像目标账户正停用/换图时，W后不能锁目标账号；旧引用先失效不得再签，先签发只承诺bearer期限（T7/T10）。
3. 新project INSERT、asset scope迁移及最终签名/commit任意失败，不留下project或被抢占的准备件；请求失去响应后的重试不扩大为新幂等协议（T8/T10）。
4. 公开邀请在MAC验证后刚被撤销/接受、workspace被删时，固定logo不能绕过原预览权限；错误token仍零数据库查询（T7/T10）。
5. 删除页扫描后owner已换图/页只删0行/进程中断，不能清新引用或提前结束；反向FK遗漏必须被目录拒绝（T4/T12）。

## 文件与接口

精确资产类型/签名见access §1；owner分别声明ProfileImageChanges、LogoImageChange、CoverImageChange及自己的引用值/URL，bootstrap逐字段转换。新asset根外观NewImageBindings(Provided,Clock)、NewImageReader(Provided,ReadSigner,RuntimeNow)、NewPurger(Provided,ImageReferenceDetacher,Tx)一次构造。

新文件按app/image_bindings.go、image_reads.go、image_ports.go，postgres/image_references.go与queries/images.sql分职责；绑定/读取可拆rules文件保持约400行。domain字段加入既有User/Workspace/Project及patch值，图像只用纯ID/URL/ExpiresAt值。HTTP保持严格生成接口，不能读取Storage或鉴权。

迁移00035/00036/00037按主spec §2；FK约束名/索引/用途取值见purge §1。bootstrap增加asset_images.go、asset_references.go，catalog/retention测试在原目录扩展；不复制新测试框架。

## Task 1（T1）：所属迁移、真实字段与纯规则

**Files:** 新三份迁移；修改server/sqlc.yaml、三owner domain与postgres/queries及gen、docs/v0/plane-diff.md；测试migrations、archtest/sqlc和owner domain/postgres。

**Interfaces:** 产出nullable asset IDs、Project.CoverBuiltinID、三态patch（Set标志+可空UUID/ID），ProjectRow新增最终cover字段；保留原公开use case签名的领域值，不先接签名。

- [x] Step 1：写TestProfileImageColumns/TestProjectCoverChoice/Table-driven patch：既有P3用户/工作区/项目升级新列null；有效cover-01/29、两个值同非null拒绝、cover-00/30/空串拒绝，省略/null/值各不同；FK真实23503、即时/索引/所有权元数据。
- [x] Step 2：运行具名RED，确认缺列/规则；仅编译缺字段不算行为红。
- [x] Step 3：添加三owner迁移、sqlc scope和查询/domain纯值；更新Plane差异；所有源变化经make gen。已有P3数据升级保持既有字段；空库Up/Down往返，不宣称带绑定数据降级无损。
- [x] Step 4：`cd server && go test -p 1 ./migrations ./internal/archtest ./internal/modules/identity/domain ./internal/modules/workspace/domain ./internal/modules/project/domain -count=1`；Expected全部通过且具名检查运行。反向清理全程序目录在T4闭环，不提前豁免。
- [x] Step 5：记录并提交 `feat(M5/P4): 增加所属模块的图片引用字段和封面规则`。

## Task 2（T2）：资产批量绑定协议

**Files:** asset/app/image_bindings.go、image_binding_rules.go、image_ports.go及对应测试；asset/adapter/postgres/image_bindings.go、queries/images.sql及image_bindings_test.go；asset/provide.go及新images.go根外观。

**Interfaces:** 消费T1字段规则和既有AssetLocker/MetadataWriter，新增access §1的ImageBindingWriter/BindImage专用写；实现ImageBindingRequest、LockImageBindings/ApplyImageBindings及NewImageBindings。SaveAsset继续不改归属/绑定，未使用的AssetReader不注入。

- [x] Step 1：TestImageBindings检验两图逆序旧新ID一次排序锁、隐藏优先409、跨申请者/完整scope/过期404、用途422、未uploaded409、bound拒绝；7天exact允许/+1微秒拒绝；当前ID幂等、管理员更替移除旧件、旧删除审计保持。
- [x] Step 2：具名RED后实现共同校验器；Lock返回最终锁后Clock时刻，Apply重读相同集合且同At，新件经BindImage、旧软删经SaveAsset；new_project变更仅在Apply调用的固定绑定SQL设置scope和ProjectID，不回锁owner。旧SaveAsset不扩写范围，BindImage无事务/Revision冲突/非图片或已绑定负例真实拒绝。
- [x] Step 3：`cd server && go test -p 1 ./internal/modules/asset/app ./internal/modules/asset/adapter/postgres -run '^TestImageBindings' -count=1`；Expected具名全GREEN。实际事务测试证明无事务LockAssets报错、Apply失败无部分绑定；变异去掉申请者/范围/7天/排序各应断言失败。
- [x] Step 4：受影响app race和lint-go；保存返回时刻/审计证据后提交 `feat(M5/P4): 实现同事务图片绑定和旧引用回收标记`。

## Task 3（T3）：资产批量图片签发

**Files:** asset/app/image_reads.go及测试，asset/images.go；复用read.go的contentExpiry，必要时按纯职责移到独立文件，所有原调用同时改。

**Interfaces:** 实现ImageReference/ImageURL/ReadBoundImages；NewImageReader只依赖已Provide存储、ReadSigner、RuntimeNow。

- [x] Step 1：TestReadBoundImages：四用途/各宿主准确、准备/已删/未uploaded/错误scope不签；数据库/解码/Signer错误原样；同UUID不一致reference整体error且sign次数0；1000/1001+重复引用有序分块、不N+1、不改审计。
- [x] Step 2：具名RED后实现一次收齐ID、全部按序锁后验证/签名；过期规则直接共用P2，不能Sign前后改URL。
- [x] Step 3：`cd server && go test ./internal/modules/asset/app -run '^Test(ReadBoundImages|AssetContent)' -count=1`；Expected所有具名通过；假Signer只证明编排，真实local/S3在T11。去掉bound校验、scope比对、同ID冲突拒绝各有行为变异。
- [x] Step 4：affected race/lint，提交 `feat(M5/P4): 为已授权业务引用批量签发图片地址`。

## Task 4（T4）：反向引用清理与目录证明

**Files:** 三owner的postgres/image_references.go/queries/images.sql、provide.go；asset app/purge.go、postgres/purge.go/queries/purge.sql、provide.go；bootstrap/asset_references.go、app.go、asset_protection_catalog_test.go、purge_catalog_test.go、asset_purge_test.go及retention调用者。

**Interfaces:** 精确采用purge子合同§2/§3，AssetPurgeCandidates/DeletePurgeAssets和ImageReferenceDetacher；去掉原不完整Provided.Purger，所有调用方改NewPurger一次构造。

- [x] Step 1：TestImageReferencePurge/TestImageReferenceCatalog真实PG：双向FK保护，四列逐一解除，identity→workspace→project→asset锁序，旧候选期间换图不误清，解除后失败回滚，batch1最小有key/中间eligible/最大59天，删0也推进。
- [x] Step 2：运行具名RED后实现固定owner SQL与有界asset页；相同事务、无对象I/O、失败cursor不前移。守卫精确集合、索引、非deferred及每owner步骤，不增加任意NO ACTION豁免。
- [x] Step 3：`cd server && go test -p 1 ./internal/modules/asset/app ./internal/modules/asset/adapter/postgres ./internal/bootstrap -run '^Test(ImageReference|AssetProtected|PurgeCatalog|AssetPurge)' -count=1`；Expected真实具名全GREEN；旧P2祖先/未知23503负例继续失败。
- [x] Step 4：移除一个解除列/步骤、改SET NULL、放宽metadata空、删0终止各做真实反例；affected race和gen-check/lint-go；提交 `feat(M5/P4): 接入所属引用解除并延伸物理清理守卫`。

## Task 5（T5）：用户资料端到端接线

**Files:** identity/domain/user.go、app/update_me.go/get_me.go、app/image_ports.go/image_reads.go、module.go、provide.go及postgres/accounts.go/queries；HTTP/me.go、api/modules/identity.yaml与生成物；bootstrap/asset_images.go/app.go；对应测试。

**Interfaces:** NewUpdateMe改用明确Deps（Users、Bindings、Images、CredentialLock、Tx、Clock）；GetMe接同样的读取/凭据/事务依赖。PublicProfile增加AvatarAssetID但不签发，既有DisplayNames调用者映射保持。

- [x] Step 1：TestProfileImages完整认证API：准备→真实PUT→complete→PATCH两图→GET实物；省略保留/null清除、当前ID重复不写资产、另一用户/另一purpose/7天过期拒绝；注入资产/owner/Signer/commit失败保持旧值、candidate未绑定。
- [x] Step 2：RED后接T2/T3，CredentialLock为第一把、当前User在锁下读，At同一；HTTP加入三态ID和URL到期对，x-problem-codes及错误中英文映射随实际码，不加手工DTO。
- [x] Step 3：`make gen`；`cd server && go test -p 1 ./internal/modules/identity/... ./internal/bootstrap -run '^Test(ProfileImages|UpdateMe|GetMe|Provided|PublicOperations|APIRoutes|OperationsThatNeed|BodiesThat|TheAnswerToABroken|ParametersThat|EveryProblem)' -count=1`；Expected全部命中GREEN、完整旧identity回归无新失败。
- [x] Step 4：更新生成类型引起的既有TS测试数据为真实null字段，不用as any；lint-go、受影响test-web；提交 `feat(M5/P4): 打通个人头像与封面的原子更新和读取`。

## Task 6（T6）：工作区图标与目录响应

**Files:** workspace app/update_workspace.go/get_workspace.go/list_workspaces.go、image_ports.go/image_reads.go、module.go、domain/workspace.go、postgres查询；HTTP/workspaces.go、api/modules/workspace.yaml；bootstrap/asset_images.go/app.go。

**Interfaces:** 所属LogoImageChange端口映射T2，ReadWorkspaceImages映射T3；CallerLock显式第一把。ListWorkspaces保持完整集合，候选W升序锁/锁下重读，仅输出本次已锁且仍活成员的集合。

- [x] Step 1：TestWorkspaceImages：管理员B替换A已绑定logo成功/不能绑A准备件；member/guest写403、非成员404；Get/List/Update URL与期限一致、Create无logo真实null；目录预读后撤权/新加入不残留或追锁。
- [x] Step 2：RED后实现logo字段与读取事务；OrganizationSize回答oneOf生成枚举或null，six有效/非法值拒绝、现有请求不改；保持名称/id排序和邮箱/角色规则。
- [x] Step 3：`make gen`；`cd server && go test -p 1 ./internal/modules/workspace/... ./internal/bootstrap -run '^Test(WorkspaceImages|CreateWorkspace|UpdateWorkspace|GetWorkspace|ListWorkspaces|PublicOperations|APIRoutes|OperationsThatNeed|BodiesThat|TheAnswerToABroken|ParametersThat|EveryProblem)' -count=1`；Expected具名GREEN。生成TS消费者用新类型，lint-go/gen-check。
- [x] Step 4：提交 `feat(M5/P4): 接通工作区图标并收紧组织规模响应类型`。

## Task 7（T7）：成员头像与既有邀请预览

**Files:** workspace app/list_members.go/update_member.go/get_invitation.go/accept_invitation.go及image_reads.go；identity PublicProfiles查询；workspace HTTP/members.go/invitations.go/API；bootstrap/ports.go（workspaceProfiles）和asset_images.go。

**Interfaces:** MemberAvatarReference{UserID,AssetID}批量转换；只拿调用者CredentialLock，PublicProfiles不锁目标用户。公开预览仍MAC链接能力，W(S)→邀请行S内重读有效邀请/固定workspace后签logo（ShareInvitationPreview，修正原先对撤回锁型的误判）。

- [x] Step 1：TestMemberAvatarReplies/TestInvitationImagePreview：成员列表/改角色/接受邀请分别返回真实图片；guest邮箱仍null；已停用目标用户资料沿M3规则可见；预览wrong token零查询，指名另一资产无接口，撤销/accepted/workspace删除404，被忽略未删仍可预览。
- [x] Step 2：RED后接现有owner授权，不复制角色谓词；Accept/UpdateMember在已有cascade之后、commit之前签；加CallerLock只能在原W/P锁前。所有原邀请/成员失败仍原样回滚。
- [x] Step 3：`cd server && go test -p 1 ./internal/modules/workspace/... ./internal/bootstrap -run '^Test(MemberAvatar|InvitationImage|AcceptWorkspaceInvitation|GetWorkspaceInvitation|UpdateWorkspaceMember|ListWorkspaceMembers)' -count=1`；Expected命中GREEN；实际W后取目标账号锁的变异须在T10复现，不能只按字符串查代码。
- [x] Step 4：生成/受影响race/lint；提交 `feat(M5/P4): 为成员资料和邀请预览接入授权图片`。

## Task 8（T8）：项目封面与一次创建

**Files:** project domain/project.go及patch规则、app/create_project.go/update_project.go/image_ports.go；postgres/projects.go/queries/projects.sql、module.go；HTTP/projects.go、api/modules/project.yaml与生成物；bootstrap/asset_images.go/app.go。

**Interfaces:** CoverImageChange消费T2；ProjectRow在INSERT携带最终AssetID或BuiltinID，Apply才迁移new_project准备件；现有账号/成员/项目锁和同实例gate按access §2。

- [x] Step 1：TestProjectCoverCreate/TestProjectCoverUpdate：cover-01/29稳定，互斥拒绝，省略/null/非null自动清另一侧；new_project scope与actor必须匹配；一次POST即绑定并有完整members/preferences/6 states。分别在states、Apply、Signer和commit注入失败，断言所有新业务行0、asset仍原scope/unbound/原审计。
- [x] Step 2：RED后实现Lock→At→INSERT最终cover→Apply→签发；Update读真实旧字段并全量收齐ID，权限/归档先于资产状态；不用前端二次PATCH或自动随机。
- [x] Step 3：`make gen`；`cd server && go test -p 1 ./internal/modules/project/... ./internal/bootstrap -run '^Test(ProjectCover|CreateProject|UpdateProject|Contract)' -count=1`；Expected全部具名GREEN，默认无图老路径保持。
- [x] Step 4：受影响race、生成TS fixture/consumer适配和lint；提交 `feat(M5/P4): 原子创建并更新项目的上传或预设封面`。

## Task 9（T9）：项目所有响应中的授权图片

**Files:** project app/get_project.go/list_projects.go/archive_project.go/join_project.go/image_reads.go、模块接线及postgres查询；bootstrap/asset_images.go；对应HTTP/集成测试。

**Interfaces:** 精确access §4；Get/List先预检候选，进同一gate后Tx Caller→W(S)→P(S)排序，重验/过滤；Archive/Unarchive/Join用原写锁返回同DTO的图，不改全模块Locks或label/state写行为。

- [x] Step 1：TestProjectImageReplies：Get/List/Create/Update/Archive/Unarchive/Join每个响应URL/expiry及字节真实；私有无权/guest/已归档/已删，目录期间新增/撤权的候选边界，响应顺序与原集合合同保持。
- [x] Step 2：RED后接集中只属于图像响应的编排，批量project share SQL；预检隐藏目标不进gate，不能边签发边追新项目；签发错误不把已确认写当成功返回。
- [x] Step 3：`cd server && go test -p 1 ./internal/modules/project/... ./internal/bootstrap -run '^Test(ProjectImageReplies|GetProject|ListProjects|ArchiveProject|JoinProject)' -count=1`；Expected具名GREEN；普通旧M3读写测试全跑并记录。
- [x] Step 4：gen-check/lint与受影响race；提交 `feat(M5/P4): 统一项目各响应入口的封面授权和签发`。

## Task 10（T10）：权限、事务与真实锁交错

**Files:** 新bootstrap/asset_image_concurrency_test.go、asset_image_binding_races_test.go、asset_image_auth_test.go及具名fixture文件；复用pgtest实际PID/SQL/blocker辅助。

**Interfaces:** 消费T5–T9完整HTTP路由及P2既有prepared API；不通过测试SQL制造已绑定功能成功。

- [x] Step 1：TestImageCredentialRace/TestImageParentRace/TestMemberAvatarLockOrder/TestInvitationImageRace/TestProjectCoverCreateRace：各自设置两方向先后、反序UUID，观察实际等待锁；reset/deactivate、workspace撤权/删除、project私有化/归档/撤权、target profile替换/停用、invitation撤销/accepted/收件账户停用删邀请（含收件人尚非该工作区成员）、双创建抢同准备件。
- [x] Step 2：让对应性质缺失时实际RED，修复局部接线/断言根因；新增架构分歧先提交。owner隐藏/不存在的code/body/headers逐值比较，双图片隐藏优先不因输入排序改变；当前引用不受旧申请者限制。
- [x] Step 3：`cd server && go test ./internal/bootstrap -run '^Test(ImageCredentialRace|ImageParentRace|MemberAvatarLockOrder|InvitationImageRace|ProjectCoverCreateRace|ImageAuth)' -race -count=5 -v`；Expected所有具名每个五轮通过、无40P01、没有skip/0 tests；PID/连接/gate证据有实际输出。
- [x] Step 4：断言存储I/O计数0、取消后连接与gate可再用，读取审计不变；提交 `test(M5/P4): 覆盖图片授权撤销和原子绑定的真实交错`。

## Task 11（T11）：F5–F7 API与双存储实物

**Files:** 新e2e/stories/files/f5-profile-images.spec.ts、f6-workspace-images.spec.ts、f7-project-covers.spec.ts；复用fixtures/storage.ts及s3-world.ts；新bootstrap/asset_image_storage_test.go。

**Interfaces:** 所有前置经API，local与P3真实S3共用图片字节和断言；浏览器上传行为P3已证明，P4 API故事不冒充P6页面。

- [x] Step 1：F5验证双图/替换/移除/重新认证旧申请归属；F6验证管理员更替/非管理员拒绝/成员资料/邀请预览；F7验证一次POST最终引用、失败无project、稳定builtin和自定义互换。每条核实际文件SHA、DB引用与bound/scope/审计；拒绝前后都读DB证明没有旁路写。
- [x] Step 2：`make build`；`cd e2e && pnpm exec playwright test stories/files/f5-profile-images.spec.ts stories/files/f6-workspace-images.spec.ts stories/files/f7-project-covers.spec.ts --project=chromium`；Expected所有具名GREEN且真实raw字节断言运行。
- [x] Step 3：`cd server && go test ./internal/bootstrap -run '^TestImageStorageContract$' -count=1 -v`；Expectedlocal和真实S3两个子组的绑定/读取/拒绝/提交回滚均实际运行；PG/S3同一套件串行，不用fake Storage成功冒充。
- [x] Step 4：关键local→S3业务代码一致性与资源inventory核验，提交 `test(M5/P4): 验收个人和组织图片的真实文件故事`。

## Task 12（T12）：清理恢复、规模与性质变异

**Files:** bootstrap/asset_image_retention_test.go、asset_image_purge_process_test.go、asset_image_capacity_test.go；原asset_purge/retention/catalog fixture；证据ignored workspace。

**Interfaces:** 消费T4最终Purger与完整API；复用P2/P3实际进程/数据库和故障控制，不加生产测试开关。

- [x] Step 1：TestImageReferenceRetention/TestImagePurgeProcess/TestImageReadCapacity：四类61/59天、empty/nonempty/活邻域、全部四引用及父链；页提交/页失败后真实新进程继续，cutoff/after不漂移；至少1000/1001图引用，记录SQL次数、耗时与取消后资源。
- [x] Step 2：`cd server && go test ./internal/bootstrap -run '^Test(ImageReferenceRetention|ImagePurgeProcess|ImageReadCapacity)$' -count=1 -v`；Expected每个具名GREEN、数据收敛且活邻域不变；精确存储失败保留资产/父行。
- [x] Step 3：累计至少15个行为变异并逐一恢复hash：申请者、scope、bound、uploaded、7d、old/new排序、账户首锁、目标账号逆序锁、owner授权、同UUID混合引用、创建事务、忽略签名失败、反向FK/漏列、盲清当前ref、删0结束页。只计具体断言RED，编译/启动失败及0 tests剔除并保留原件。
- [x] Step 4：受变异影响的最终测试全部GREEN、关键数据库交错五轮race、gen-check/lint；提交 `test(M5/P4): 证明引用清理恢复和批量图片边界`。

## Task 13（T13）：完整门禁与产品检查点

- [x] Step 1：逐项核A1–A7及handoff处理结果，所有文档逐句对照实际证据；新字段/错误生成物与消费者齐全，关键词phase=M5/P4且无到期例外续期。
- [x] Step 2：串行 `make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`；Expected全0、无未解释skip/缓存混算，记录每项输入hash/输出/时间。保留失败后按根因修复，变动后复验受影响门禁。
- [x] Step 3：提交中文产品检查点，系统git push；核对准确HEAD的server/web/e2e三job全绿，日志通过连接GitHub自行读取。已知W11正常重跑一次；再次失败停报，不能取消测试或改断言绕过。
- [x] Step 4：阶段review记录本地/远端各自证据，不提前称集成完成。

## Task 14（T14）：整分支deep-review与修复

- [x] Step 1：固定基线main5807a0d6到最终候选；按deep-review顺序核逻辑、集成、测试，完整手写文件和caller链有清单，生成物核源/锁。存docs/reviews下一编号M5P4整体审核。
- [x] Step 2：executing-plans要求的一次fresh整分支review，交spec/plan、Review Focus原文、Ruling台账；按实际用户影响重新分级。
- [x] Step 3：Important/Critical一次局部修复，真实RED/GREEN、受影响/必要全量门禁和主控复核，不反复派同范围review；Minor具名去向，重大新风险提交后上报。
- [ ] Step 4：所有裁定/代价和未读范围写报告；提交最终文档与修复、推送并核精确最后HEAD CI；A8只有此时可关闭，A9继续等待。

## Task 15（T15）：集成、归档与P5预检

- [ ] Step 1：核准确green HEAD、干净worktree、main祖先关系；按连续授权系统git --no-ff合入main，检查两父与树一致，再正常push。
- [ ] Step 2：准确main SHA CI全绿；归档全部ignored台账/raw/失败/浏览器/审者/变异材料，hash/bytes/mode/membership/unknown=0；正常删除远端分支（rev-list为0）、worktree和本地分支，清理后再核归档。
- [ ] Step 3：回填阶段完成、A9与下一段P5工作项/评论/草稿绑定及附件的spec/plan预检；输入handoff仍open，M5整体完成项保持未勾选。

## 预检自查

| spec要求 | Task与证据 |
|---|---|
| 迁移所有权、三态字段、稳定builtin、organization_size | T1/T5/T6/T8；A1/A3/A4 |
| 两图集合锁、同At、准备件/旧件区别、失败原子性 | T2/T5/T6/T8/T10；A2/A4 |
| 全部DTO出口、MAC预览、账号首锁、无目标用户逆锁 | T3/T5–T10；A5 |
| 反向NO ACTION/owner解除/有限页/恢复/守卫 | T4/T12；A6 |
| F5–F7、local/S3真实实物、规模/15变异/五轮race | T10–T12；A7 |
| 七项/整审/精确CI/main/归档/P5预检 | T13–T15；A8/A9 |

15 Tasks，未拆分为并行项目；绑定、读取和清理共享同一引用合同，按顺序实施。主控自检需在阶段review写明覆盖、类型/路径一致性、每步可执行、五项Review Focus映射与篇幅；builder check只审三份spec，不额外派plan reviewer。


