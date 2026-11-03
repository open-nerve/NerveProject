# M5/P5 工作项、评论与草稿资产 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 按任务内联实施，沿既定原生方式；不用逐 Task 新派实现者/审者。每步保留 RED/GREEN、输入hash和裁定，阶段结束一次 fresh whole-branch review。

**Goal:** 在现有文件服务上完成工作项/评论/草稿引用、真实附件计数排序、迁移和事件，保持权限与回滚边界。
**Architecture:** issue 声明小消费端口，asset 仅处理自有表，bootstrap 唯一转换与接线；owner 已授权事务持父锁后才锁完整资产集合。列表沿同一快照完整目录传参，无跨模块 JOIN 或持久计数副本。
**Tech Stack:** 仓库已锁 Go/PostgreSQL/sqlc/OpenAPI、TypeScript/Vitest/Playwright；local和已固定SeaweedFS，不新增依赖。
**Spec:** [主spec](../specs/P5-issue-assets.md)、[绑定](../specs/P5-content-binding.md)、[读取](../specs/P5-attachment-reads.md)。预检通过；基线main 0f93100a05304d26b074790633b5659285813156，主控自查及一次builder check闭环，见[预检](../../../reviews/re-40-M5P5预检.md)。

## Global Constraints

- 根目录HEAD始终main；本段codex/m5-p5 worktree，plane/refer只读，不在其中运行git；详细中文提交，系统/usr/bin/git push，集成--no-ff。
- 不操作已有容器或Docker；容器套件串行、Go跨包-p1；本任务已不用容器按精确ID/会话核验回收，不用prune扫其他会话。
- 无新依赖/业务表/跨模块import或JOIN；sqlc生成，不手改生成物；domain纯净，app小接口，bootstrap唯一接线。手写文件约400行，现有大文件只按本次职责拆分。
- 同步绑定/迁移/动态/事件同ctx事务、最后锁后一个At/By；存储I/O仍事务外；不复制Locks.run、不放宽守卫；隐藏404逐值相同。
- 评论非空规则与草稿有界JSON保存规则保留；附件分页50/100、内部SQL1000；图片只规范UUID绑定，有序src身份不去重。
- seven gates在有产品检查点push前串行完成；一次fresh终审，重要问题真实RED/GREEN后主控复核；Minor具名去向，重大风险先提交后上报。
- 不提前标记P6/P7页面、输入handoff或M5整体完成；原始失败/缓存/skip与tested SHA单独记录，最终准确branch/main CI各自证明。

## Review Focus

1. 别人上传的已绑定附件应可由宿主合法成员读取，但同一用户的未绑定/已删/跨用途件绝不能被公共化（T3/T10）。
2. duplicate的源Read与目标Prepare混合、ID重复或交换输入顺序时，不能用目标合法覆盖源不可见（T3/T10）。
3. 草稿已有高UUID资产加新低UUID引用、确认外部I/O途中改项目/发布，完整锁序不能逆行且旧结果不能绑定新宿主（T7/T10）。
4. 附件计数目录跨1000边界、排序最重项原在第二页、guest藏项、分组同Issue多cell，分页和total不能只对当前页成立（T8/T12）。
5. 已绑定历史图片移出正文并越过20版本、父墓碑/物理对象故障/清理重试，不能提前删历史或失去唯一恢复记录（T11/T12）。

## 文件与接口

精确值/签名由绑定§1、读取§1定义；接口在issue/app，asset根新增content.go外观并由bootstrap/asset_content.go转换。asset app按content_bindings.go、draft_assets.go、attachment_reads.go、attachment_events.go分职责；自有postgres对应queries/content.sql、attachments.sql及适配器。image_bindings.go继续仅资料图片。

issue新增app/content_asset_ports.go、attachment_ports.go、attachments.go、attachment_events.go；修改content projection/domain rich_text、comment创建/修改、draft_writes/publish_draft、create_core、seen/Listing、三套列表SQL/cursor。API issue.yaml与生成物、中文英文错误/动态文案、必要TS实体fixture随真实实现同步；P7控件不提前挂载。

## Task 1（T1）：有序图片内容身份

**Files:** issue/domain/rich_text.go/description_test.go；adapter/content/projection.go及projection/plane_tags测试；tools/keywords.json、阶段review。
**Interfaces:** 产出ContentIdentity.Images []string、纯方法ContentIdentity.AssetImageIDs() []uuid.UUID（只取规范UUID，保留出现顺序；锁层去重排序）和同一Sanitizer.Clean结果；保留HTMLSanitizer签名，identity保留原顺序重复。

- [x] Step 1：写TestImageContentIdentity / TestImageIdentitySkipActivity：两图换序、相同图重复、空src、远程安全src、大小写非规范UUID、raw img、仅block ID迁移；换序/换图不得skip，格式不改identity。
- [x] Step 2：`cd server && go test ./internal/modules/issue/domain ./internal/modules/issue/adapter/content -run 'Test(Image|RichText|Description)' -count=1`，记录真实断言RED。
- [x] Step 3：实现有序投影/Equal，phase=M5/P5；不把URL清洗改成UUID-only，不引入domain HTML解析。
- [x] Step 4：相同命令GREEN、两完整包与lint-go；变异删Images比较及sort/去重src均须失败。提交 `feat(M5/P5): 将有序图片身份纳入内容比较`。

## Task 2（T2）：完整集合绑定与迁移能力

**Files:** issue/app两份端口；asset/app/content_bindings.go/draft_assets.go及纯规则；postgres/content_bindings.go/queries/content.sql、asset/content.go/provide.go；bootstrap/asset_content.go仅值转换；对应app/PG测试。
**Interfaces:** 消费T1规范图片投影；产出spec的ContentBinding、DraftAssetTransfer；保存scope使用专用SQL，SaveAsset不可变合同不扩写。

- [ ] Step 1：TestContentBindings / TestDraftAssetTransfer覆盖申请者/用途/完整父链/已上传/已删/既绑定同宿主、1001完整集合、零UUID与末项坏零写、UUID倒序、无事务拒绝、两阶段At相等；草稿全量含pending改项目/发布拒pending。
- [ ] Step 2：`cd server && go test -p 1 ./internal/modules/asset/app ./internal/modules/asset/adapter/postgres -run 'Test(ContentBindings|DraftAssetTransfer)' -count=1`取得行为RED；按子合同实现去重排序集合锁、最后读钟和专用绑定/迁移。
- [ ] Step 3：相同命令GREEN、-race与架构/lint；故障第2个UPDATE整体回滚，SaveAsset仍不能改scope；去掉申请者/范围/全集合锁各有失败变异。
- [ ] Step 4：gen及受影响检查后提交 `feat(M5/P5): 实现内容绑定与草稿资产迁移端口`。

## Task 3（T3）：附件发布、读取权限与精确事件

**Files:** asset/app/access.go/read.go/complete.go/duplicate.go/delete.go及publication事件职责文件；asset/domain纯规则；issue/app/attachment_events.go/domain活动；两模块module_events.go/module.go、bootstrap/event目录/组合根及新测试；API问题码和必要文案。
**Interfaces:** 用具名资产访问请求绑定ID与HostRequest，完整父锁顺序不变；Event attachment_added/deleted严格遵主spec§2.4，asset.EventTypes()声明两类型并加入bootstrap.eventTypes，issue subscriber只消费当前事务与P(N)能力。

- [ ] Step 1：TestAttachmentPublication / TestBoundAssetAccess / TestAttachmentEvents真实API/PG：已有宿主三附件complete绑定、new_comment不绑、旧uploaded未绑重试一次、重复/并发确认、成员读别人活附件、准备/资料仍私有、已绑定描述禁止删、附件删除幂等与首次审计。
- [ ] Step 2：运行`cd server && go test -p 1 ./internal/modules/asset/... ./internal/modules/issue/app ./internal/bootstrap -run 'Test(AttachmentPublication|BoundAssetAccess|AttachmentEvents)' -count=1`记录RED，完成资格纯规则、关联授权与事件/动态接线；不能去掉全局applicant后任意放行。
- [ ] Step 3：相同命令GREEN；subscriber/COMMIT故障全部回滚，重复complete/delete无额外动态，父清理无逐件事件；opaque错误不回显filename/key/私有ID。
- [ ] Step 4：完整asset app、事件合同、旧P2/P4权限与真实存储回归、gen/lint；提交 `feat(M5/P5): 接通附件发布权限与事务事件`。

## Task 4（T4）：分页列表与批量计数读取端口

**Files:** asset/app/attachment_reads.go、postgres/attachments.go/queries/attachments.sql；issue/app/attachments.go/attachment_ports.go、domain attachment_cursor.go、HTTP attachments.go；api/modules/issue.yaml及Go/TS生成；bootstrap接线与tests。
**Interfaces:** spec读取§1的AttachmentReader/AttachmentCounter/AttachmentCountCatalog；三个GetAttachments用例复用当前read snapshot/owner权限，内部请求limit+1。

- [ ] Step 1：TestAttachmentLists / TestAttachmentCounts / TestAttachmentCursor覆盖50/100与101分页、宿主错cursor、unknown字段400、已删/未上传/未绑/描述过滤、完整两页scope、另一作者附件、SQL/端口故障不是空页。
- [ ] Step 2：`cd server && go test -p 1 ./internal/modules/asset/adapter/postgres ./internal/modules/issue/domain ./internal/modules/issue/app ./internal/bootstrap -run 'TestAttachment(Lists|Counts|Cursor)' -count=1`记录RED后实现端口/路由与生成合同。
- [ ] Step 3：GREEN及归档/权限旧回归；GET零DML/零存储I/O，同快照先授权再查资产；1001 IDs无截断且无N+1。
- [ ] Step 4：完整契约/架构、gen/lint与前端类型，提交 `feat(M5/P5): 提供有界附件列表和真实批量计数`。

## Task 5（T5）：工作项正文绑定与全部Issue出口

**Files:** issue/app/create_core.go/create_issue.go/update_issue.go/seen.go/listing.go、domain/issue.go、HTTP DTO、module.go；api/modules/issue.yaml及生成物、web受影响fixture；bootstrap/issue_content_assets_test.go/issue_attachment_projection_test.go。
**Interfaces:** 产出create_core私有prepareLocked/writePreparedAt，保持唯一创建逻辑；Issue.AttachmentCount int64，所有API通过T4 AttachmentCounter填真实值，不留临时0实现；NewLocks在原五参数后增加attachments AttachmentCounter，所有生产/测试构造者同时接线。IssueCreationDeps及ListingDeps新增AttachmentCounter依赖，禁止nil回落假0。

- [ ] Step 1：TestIssueContentAssets / TestIssueAttachmentProjection：同宿主新/历史图、别人的已绑图可保留、跨宿主/用途/未上传404或既有码、skip图片变更真实版本动态；审计每个Issue出口包括关系内嵌/搜索/批量与member列表。
- [ ] Step 2：`cd server && go test -p 1 ./internal/modules/issue/... ./internal/bootstrap -run 'Test(IssueContentAssets|IssueAttachmentProjection|Description)' -count=1`取得RED后接线；直接新建Issue不能引用另一Issue/draft图片；readBack/seen/Listing补同批统计。
- [ ] Step 3：GREEN；正文/资产/版本/subscriber/commit逐点回滚；真实count仅ISSUE_ATTACHMENT，完整Issue合同和旧M4创建/更新/列表回归。
- [ ] Step 4：生成、lint及TS类型/相关fixture测试，提交 `feat(M5/P5): 原子绑定工作项图片并返回真实附件计数`。

## Task 6（T6）：评论准备件与创建原子性

**Files:** issue/app/comments.go/comment_writes.go及域输入；adapter/http/comments.go/comment_writes.go；api/modules/issue.yaml；bootstrap/comment_assets_test.go，现有评论测试/生成类型。
**Interfaces:** CreateIssueComment.Execute消费具名输入{HTML string,AttachmentIDs []uuid.UUID}；HTTP attachment_ids省略空、非null、最多100唯一。Update保持HTML参数，不引入整体替换attachments。

- [ ] Step 1：TestCommentAssets：HTML两图+两个附件完整集合、申请者不同、同issue不同准备件、100/101、重复、最后一个坏、非空规则、重试前故障不消费件，现有comment追加附件权限。
- [ ] Step 2：`cd server && go test -p 1 ./internal/modules/issue/app ./internal/modules/issue/adapter/http ./internal/bootstrap -run 'Test(CommentAssets|CreateIssueComment|UpdateIssueComment)' -count=1`取得RED；预分配ID、先LockContent，再同At INSERT/Bind，注入失败整体回滚。
- [ ] Step 3：GREEN；评论删除立即禁止签发、原DELETE事件同At，独立附件页与react聚合不混用；不新增评论附件的issue attachment动作。
- [ ] Step 4：旧评论/反应/契约回归、gen/lint，提交 `feat(M5/P5): 在评论事务内绑定上传准备件`。

## Task 7（T7）：草稿保存、改项目与原子发布

**Files:** issue/app/draft_writes.go/publish_draft.go/create_core.go、draft_assets端口接线；bootstrap/draft_assets_test.go/draft_assets_races_test.go；既有draft API fixture与测试。
**Interfaces:** 消费T2 DraftAssetTransfer和T5唯一创建核心；保存使用DraftAssetRequest.ImageIDs合并全scope锁；发布最终reply在Move后Count，草稿项目可null。

- [ ] Step 1：TestDraftAssets / TestDraftAssetPublish：保留partial JSON及非法类型待发布拒绝；改旧/新/null项目迁移全部pending/uploaded；双发布仅一Issue；pending阻止、已成功final带回收candidate不阻止；描述历史及附件迁移scope/purpose/At/By准确；从未引用的描述迁移后bound=null、其他成员404、申请者可删、原7天年龄保持。
- [ ] Step 2：`cd server && go test -p 1 ./internal/modules/issue/app ./internal/bootstrap -run 'Test(DraftAssets|DraftAssetPublish|PublishDraft)' -count=1`记录RED后实现，Lock后唯一At，Issue/Move/Draft软删/事件同事务。
- [ ] Step 3：GREEN；序号/Issue/版本/第二资产UPDATE/draft/event/COMMIT故障图整体原样，回滚后重试一次；两项目UUID方向及全集合1001，source晚到不误绑。
- [ ] Step 4：完整draft/创建核心/旧M4交错回归与gen/lint，提交 `feat(M5/P5): 原子迁移草稿文件并保持唯一发布`。

## Task 8（T8）：附件全局排序与精确游标

**Files:** issue/app/listing.go/list_ports.go、domain/listvocab/list_cursor_key/list_order、postgres三套list SQL与适配器；API排序枚举/生成物；web/apps/web/core/lib/issue-order.ts、issue-filters.ts、store/issue/root.store.ts、helpers/issue-filter-helper.store.ts、project/filter.store.ts及对应测试；P7才挂UI。
**Interfaces:** ListDirectories.AttachmentCounts []AttachmentCount；T4目录完整传参；普通页面计数沿T5；±attachment_count整数比较/UUID方向继承现有规则。

- [ ] Step 1：TestListAttachmentCount：0/1/2/同数乱ID、最大bigint游标、1001目录尾最大值、guest隐藏、两方向三页、空页total、group/subgroup重叠cell、成员关系与archived/subissue组合。
- [ ] Step 2：`cd server && go test -p 1 ./internal/modules/issue/domain ./internal/modules/issue/adapter/postgres ./internal/bootstrap -run 'Test(ListAttachmentCount|List.*Cursor)' -count=1`取得RED后实现filtered阶段目录join/精确key；只允许内存目录，无asset表SQL。
- [ ] Step 3：GREEN；一次snapshot内新增/删除时count/order/cursor一致；目录失败/重复/负值显式失败，跨项目分页没有after错位；变异先page后sort、截断1000、float64、吞错误均失败。
- [ ] Step 4：完整列表/设置/成员/架构与TS比较器回归（包括root.store变更指纹识别attachment_count、列表迟到回包不覆盖新计数）、gen/lint，提交 `feat(M5/P5): 支持真实附件计数排序和精确分页`。

## Task 9（T9）：F8–F12真实API及双存储

**Files:** e2e/stories/asset/f8-issue-attachments.spec.ts、f9-comment-assets.spec.ts、f10-draft-assets.spec.ts、f11-content-copy.spec.ts、f12-image-identity.spec.ts；复用fixtures/storage；bootstrap资产双driver测试。
**Interfaces:** 只消费已实现公开API；SQL断言/故障显式区分正向准备；真实File字节与最终身份、非上传者读取、source不可见404。

- [ ] Step 1：先列故事成功及区分失败的持久化断言，再运行当前测试找真实缺口；不为新增故事伪造产品RED。
- [ ] Step 2：local与已固定SeaweedFS逐用途上传/确认/绑定/下载字节、复制源/目标反向拒绝、同大小不同字节、评论失败重试、草稿发布返回计数；rawURL期限/inline/attachment合同不变。
- [ ] Step 3：`make build`后`cd e2e && pnpm exec playwright test stories/asset/f{8,9,10,11,12}-*.spec.ts --project=chromium`与具名Go双driver测试串行；Expected故事/所有cells实际命中通过；保留完整report/trace。
- [ ] Step 4：记录实物和新测试容器归属/退出清理，提交 `test(M5/P5): 证明工作项评论草稿的真实文件链路`。

## Task 10（T10）：权限与真实并发交错

**Files:** bootstrap/asset_content_permissions_test.go、asset_content_races_test.go、draft_assets_races_test.go；复用issueWorld/heldTx/serializeAt/pgtest.RowWaits。
**Interfaces:** 已有gate/pool、caller/W/P/D/A真实锁；无固定sleep、无生产专用测试hook。

- [ ] Step 1：TestContentAssetPermissions / TestContentAssetRaces覆盖Review Focus1/2/3：普通/管理员/guest自身他人/管理员兼guest两种OwnOnly、私有/撤权/归档/删除；Prepare/Complete/Get/Content/Duplicate/Delete/HTML所有入口。
- [ ] Step 2：双项目两UUID次序、两资产逆序、删除/归档/停用先后、草稿双发布/改项目/pending确认外部I/O中断；精确观察PID等待点、零写拒绝和取消后gate/pool复用。
- [ ] Step 3：`cd server && go test -p 1 ./internal/bootstrap -run 'Test(ContentAssetPermissions|ContentAssetRaces|DraftAssetRaces)' -race -count=5`，Expected每具名测试五轮通过；去掉源权限/绑定隔离/取消释放/全集合顺序的变异分别失败。
- [ ] Step 4：归档原始时序和矩阵，清理精确自建容器，提交 `test(M5/P5): 验证附件权限边界与迁移交错`。

## Task 11（T11）：删除图、历史图片与恢复

**Files:** bootstrap/asset_content_deletion_test.go/asset_content_retention_test.go；现有删除目录/asset purge fixtures；e2e/fixtures/assert/issue.ts与I15/I17/I21按新bound状态扩充。
**Interfaces:** 复用P2父保护、P4反向引用目录和有界purger；不加未知FK豁免；parent删除事件原At/By，不冒充用户attachment_deleted。

- [ ] Step 1：TestContentAssetDeletion / TestContentAssetRetention：正文移图超过20版本仍活、直接DELETE bound图409，三附件/三描述/pending/准备件完整删除，多个邻域与混合created/deleted作者。
- [ ] Step 2：真实对象删除故障→父NO ACTION阻挡且后续候选推进→下一进程恢复；61/59天cutoff、原审计、失败页cursor、迁移后旧draft清理不删新Issue字节；未引用描述按原7天年龄回收，已绑定历史图不能同批回收。
- [ ] Step 3：`cd server && go test -p 1 ./internal/bootstrap -run 'Test(ContentAssetDeletion|ContentAssetRetention|AssetProtected|ImageReferenceCatalog)' -race -count=1`及I15/I17/I21聚焦故事；Expected图/字节/任务精确通过。
- [ ] Step 4：失败与恢复报告完整保留、及时清理自建资源，提交 `test(M5/P5): 验证历史图片和迁移资产的删除恢复`。

## Task 12（T12）：容量和性质变异总账

**Files:** bootstrap/asset_attachment_capacity_test.go、issue/PG计数容量测试；.superpowers证据与阶段review。
**Interfaces:** 冷统计1000/1001/10000有附件项与1001项目分批，不改变5秒客户端/15秒API期限；SQL数/内存目录规模完整记录。

- [ ] Step 1：TestAttachmentCapacity冷库，零/重叠/多project/guest/两方向/group/cell；query前不ANALYZE；普通页只查返回ID，附件排序完整目录，取消后资源复用。
- [ ] Step 2：至少15行为变异：申请者隔离、用途、父链、uploaded、bound、deleted过滤、src次序、skip图片比较、source授权、全集合锁、发布原子性、重复事件、1000截断、page后sort、精确cursor；每项真实失败/恢复SHA/最终GREEN，不靠编译错。
- [ ] Step 3：受影响完整asset/issue/arch包race、关键交错五轮及完整gen/lint；Expected全绿，原件/计数与复用输入范围可验证。
- [ ] Step 4：写明局部裁定和代价、未声称的页面/存储品牌，提交 `test(M5/P5): 证明附件容量与安全性质`。

## Task 13（T13）：完整七门禁与产品检查点

- [ ] Step 1：逐项A1–A7和输入handoff处理结果、Plane差异、类型/文案/phase与到期豁免核验，文档逐句对照实际证据。
- [ ] Step 2：串行`make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`，各exit0；缓存/skip/基础设施失败分别记录，不隐藏原失败。
- [ ] Step 3：详细中文检查点commit/system git push；准确HEAD server/web/e2e全绿。W11按已知规则重跑一次，仍失败停报。
- [ ] Step 4：记录本地/远端分开事实，不提前声明阶段main完成。

## Task 14（T14）：整分支deep-review

- [ ] Step 1：固定main基线到候选HEAD，deep-review按逻辑/集成/测试，整份手写文件及caller清单，生成物核源。
- [ ] Step 2：一次fresh reviewer携spec/plan/Review Focus原文及全部Ruling；主控按实际影响分级，报告入docs/reviews下一编号。
- [ ] Step 3：Important/Critical一次局部修复与真实RED/GREEN、影响门禁及主控复核；Minor具名去向；重大风险先提交后上报，不派重复同范围审者。
- [ ] Step 4：裁定/代价/未读范围逐条落盘，最终commit/push与准确最终HEAD CI全绿，才关闭A8。

## Task 15（T15）：集成归档及P6预检

- [ ] Step 1：核干净W/main、祖先和准确green HEAD，连续授权下--no-ff合入main、双父/树一致、system git push。
- [ ] Step 2：准确main CI全绿，归档全部ignored台账/失败/raw/浏览器/审者/变异，hash/bytes/mode/membership/unknown=0；rev-list origin/main..branch=0后正常清远端、本地与worktree，归档再核。
- [ ] Step 3：下一独立codex/m5-p6 worktree回填阶段和A9；准备P6上传传输器/会话store/资料页面spec/plan与一次builder check，M5整体完成和handoff仍未勾。

## 预检自查（已复核）

spec A1→T1/T5；A2→T2–T5/T10；A3→T6；A4→T7/T10；A5→T4/T5/T8/T12；A6→T3/T11；A7→T9–T12；A8→T13/T14；A9→T15。15 Tasks；接口按两份子合同唯一声明，步骤无实现体抄录；主控逐项核类型一致、源码路径、每个测试有区分力、Review Focus映射与篇幅后记预检记录。
