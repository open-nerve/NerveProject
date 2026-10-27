# M4/P7 `issue-links`：链接、出站客户端与元数据抓取规格

**当前状态（2026-10-27）：P7 已完成。** 经负责人授权以 `--no-ff` 合并为 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`，[main CI 40451632280](https://github.com/open-nerve/NerveProject/actions/runs/40451632280) 的 server/web/e2e 全绿；5,443 个证据文件归档校验后，本地/远端分支及 worktree 已清理。集成证据见 [P7 review](../reviews/P7-issue-links-review.md) 的集成回执；M4 总体及后续交接仍开放。

**交付前冻结记录（以下状态由文末集成回执更新）：** 日期：2026-10-27。**T1–T15 已实施并独立评审通过；准确原范围 `5005be98..d2a3cf26` 的单次整分支 deep-review 已完成（165 文件/457 chunks）。T16 单次局部修复、正式文档和准确提交的完整本地七门禁已完成；独立定向复审已确认原8项全部 ADDRESSED；T7历史时态文案已由作者纠正，本次纯文档增量确认及最终交付 HEAD CI 以主控最终记录为准。产品检查点`cd0fce7e`的[CI40397086823](https://github.com/open-nerve/NerveProject/actions/runs/40397086823)已由主控核准确HEAD全绿；此后纯文档交付HEAD的CI另核；未合并 main。** 基线：P6 已集成的 `5005be985e7ec78727edc6f8fc5c05999b45cc65`，同一 SHA 的 main CI `40305247546` 重跑全绿。分支 `codex/m4-p7`，worktree `.codex/worktress/codex/m4-p7`。配套：[plan](../plans/P7-issue-links.md)、[review](../reviews/P7-issue-links-review.md)、[整体审核](../../../reviews/re-05-M4P7整体审核.md)。预检、最窄 PG/sqlc 实验及 T4/T8/T12/T15 检查点按各自日期/源码保存，不作为本轮新执行。

权威输入：[M4 设计](../M4-design.md) §2/I10/I17、3.4–3.6、3.11、3.13/17–19、3.19–3.20、3.23/P7、4.8、5.1–5.3、6.8、8.1/8.8、9.1–9.4、12/P7、13.1；[02](../../claude_to_gpt/02-method.md)、[03](../../claude_to_gpt/03-rules-and-pitfalls.md)；P6 spec §5/review §6 与本规格 §7。D3 事务内同步事件、架构问题 11.8 的平台出站与固定 IP、后台写回的单句守卫已批准，不重新裁定。

## 1. 目标

交付链接的创建、集合读取、修改和删除；首个生产事件订阅者在原事务中投递 `issue.crawl_link`；后台通过防 SSRF 的出站客户端抓取标题和小图标，只把结果写回匹配的活链接。所有链接请求写沿既有 `Locks.run`，保持访客 404、项目锁/闸门、锁下时钟、审计与回滚语义。

新增一张 `issue_links` 表、四个 HTTP operation、两个公开错误码、三种 link 事件、一个生产订阅者和一个 default 队列 worker。无新权限动作，无新依赖，不放宽架构守卫，不改变 P6 事件分发/事务平台，不增加 outbox。`link_count` 与排序留 P8，页面/详情数据层留 P11/P14，项目与工作区异步删除留 P9。

## 2. 交付物

### 2.1 文件、任务与依赖

相对项目根；I=`server/internal/modules/issue`，B=`server/internal/bootstrap`，O=`server/internal/platform/outbound`，C=`server/internal/platform/config`。表中为实际文件职责；逐 Task 的提交/执行证据见 review §2，生成物仅由生成命令产出。

| Task | 主要文件 | 交付 / 前置 |
|---|---|---|
| T1 | `server/migrations/sql/00027_issue_issue_links.sql`、`server/sqlc.yaml`、`deploy/runtime-grants.sql`；I/app/delete_issue.go、delete_project_rows.go；I/adapter/postgres/delete_issue.go、delete_rows.go 及 queries | 表、grants、两条删除路径 / 基线 |
| T2 | I/domain/link.go、link_metadata.go；I/app/link_ports.go、link_rows.go；I/adapter/postgres/links.go、queries/links.sql | 纯领域与存储端口 / T1 |
| T3 | `api/modules/issue.yaml`、`api/openapi.yaml` 聚合入口及生成物；I/app/create_issue_link.go、list_issue_links.go、link_events.go；I/domain/events.go、errors.go、link_activity.go；I/adapter/http/links.go、link_usecases.go；I/adapter/postgres/link_conflict.go；I/module.go、module_events.go；B/events_test.go；e2e/fixtures/links.ts、furnish.ts、assert/issue-collaboration.ts 与 stories/issue/i17-delete-issue.spec.ts；错误文案 | 创建、列表、创建事件与 API fixture / T2 |
| T4 | `api/modules/issue.yaml`、`api/openapi.yaml` 聚合入口及生成物；I/app/update_issue_link.go、delete_issue_link.go、link_target.go；I/adapter/http/link_writes.go、link_usecases.go；I/domain/link_activity.go、errors.go；I/module.go；B/issue_link_writes_test.go 与权限/写锁/闸门/单连接登记；错误文案 | 改删、三种事件完整、审计与活动 / T3 |
| T5 | B/issue_link_security_test.go、issue_link_transactions_test.go；权限/闸门/写锁矩阵 | 链接请求完整安全与回滚证明 / T4 |
| T6 | O/policy.go、special_ranges.go、errors.go、testdata/iana-*.csv、testdata/sources.json、policy_test.go、policy_registry_test.go | 地址策略和独立 IANA 快照 / 基线 |
| T7 | O/client.go、transport.go、errors.go 及测试 | 安全 GET、跳转、期限/响应上限 / T6 |
| T8 | C/outbound_test.go、config.go、validate.go；server/configs/config.yaml、config.test.yaml；B/outbound.go、outbound_test.go | test-only 放行配置与组合构造函数 / T7 |
| T9 | I/app/crawl_ports.go；I/adapter/crawler/crawler.go、metadata.go 及测试 | 页面/图标解析，失败收敛 / T7–T8 |
| T10 | I/app/crawl_link.go；I/adapter/postgres/link_metadata.go、queries/link_metadata.sql；I/adapter/river/crawl_link.go；module_jobs.go/module.go；B/app.go | 守卫写回、worker 与运行组合 / T2/T9 |
| T11 | I/app/crawl_on_link_change.go；I/adapter/river/link_crawls.go；module_events.go；B/events_test.go、issue_link_jobs_test.go | 首个订阅者与真实事务投递 / T3–T4/T10 |
| T12 | B/invitations_test.go、table_rows_test.go、runtime_role_test.go、issue_link_runtime_test.go、request_jobs_test.go；复用 app_test.go/issue_crawl_jobs_test.go | 请求任务快照及运行角色真实抓取 / T11 |
| T13 | B/issue_link_interleaving_test.go、issue_link_race_helpers_test.go | 交错 19、标题与抓取、删除相遇 / T11 |
| T14 | e2e/fixtures/link-page.ts、links.ts；stories/issue/i10-links.spec.ts、i17-delete-issue.spec.ts、i17-catalog.spec.ts | I10 API、I17 与删除目录回归 / T11–T13 |
| T15 | 本期测试/发布与变异矩阵、ignored 原始证据 | 跨层核对、代表变异和完整门禁 / T1–T14 |
| T16 | spec/plan/review、两矩阵、M4/v0 设计、plane-diff、README、输入 handoffs、02/03/04；crawler/charset.go及测试、公开metadata.error schema/生成物/HTTP、既有422 fixture等待 | deep-review、文档事实与后续责任 / T15 |

设计 §12/P7 的 12 条映射：1→T1/T3/T14；2→T2/T3；3→T4；4→T6/T7；5→T8；6→T9；7→T10；8→T11；9→T12；10→T13；11→T14；12→T16。T5/T15 是横跨这些条目的性质证明。16 Task 是上限，粗估各 Task 350–1200 手写差异行，留约 10% 修复余量；这不是原型测量。

### 2.2 T1：12 列、grants 与删除顺序

`issue_links`：`id uuid PRIMARY KEY`；`workspace_id`、`project_id`、`issue_id uuid NOT NULL` 分别指向对应表 `ON DELETE CASCADE`；`title varchar(255)`；`url text NOT NULL`，`CHECK (url <> '' AND length(url) <= 2048)`；`metadata jsonb NOT NULL DEFAULT '{}'`，CHECK 是对象、键为 title/favicon/url/favicon_url/error 的子集、值仅 string/null；`created_by_id`、`updated_by_id` 可空，引用 users `ON DELETE SET NULL`；`created_at`、`updated_at timestamptz NOT NULL DEFAULT now()`；`deleted_at timestamptz`。

活行精确排重使用内置 hash exclusion（与独立 PG/sqlc 实验表达式一致，不额外添加 COLLATE）：

```sql
CONSTRAINT issue_links_issue_id_url_key EXCLUDE USING hash ((issue_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)
```

`issue_id::text` 是固定36字符的规范UUID，第37位为固定分隔符，因此表达式相等当且仅当issue_id和原文URL分别相等；URL内的冒号不造成组合歧义。沿数据库既有确定性text等值，不折叠大小写。hash仅用于索引候选定位，引擎必须对堆行重算完整表达式并用原文等号复查；不同原文即使hash相同也能共存。不是digest唯一键，也不是hash unique index；没有新字段、扩展、依赖或触发器。无条件外键索引为 `issue_links_issue_id_idx`、`issue_links_workspace_id_idx`、`issue_links_project_id_idx`，分别覆盖其同名列，沿既有issue附表的索引模式。sqlc 的 issue schema 纳入 00027，运行角色拥有与其他 issue 表相同的 SELECT/INSERT/UPDATE/DELETE。

T1以真实12列迁移证明：高熵多字节2048字符URL完整存取，2049字符由CHECK拒绝；同issue精确重复INSERT/UPDATE只产生该约束的23P01，不同issue/软删重建允许。T1/T2存储碰撞fixture固定issue UUID `0199a2b4-0000-7000-8000-000000000001`、URL `http://collision.example/271150` 与 `http://collision.example/83166`、hash `-1669361772`；它只做数据库读写，不发布任务或请求网络。先独立断言两原文不同、`hashtext(issue_id::text || ':' || url)`相同，再证明两活行共存；不能只模拟相同hash。T2存储复用固定碰撞fixture并验证LinkURLExists仍查原文；T3/T4真实API覆盖高熵长度、精确重复、跨issue与软删复用，不重复搜索物理hash碰撞。不能仅测纯字符计数或高重复ASCII。API长URL使用被出站策略拒绝的保留地址，避免以后worker启动时触发真实外网。

新增 `IssueRowsDeleter.DeleteIssueLinkRows(ctx, IssueDeletion) error` 与 `ProjectRowsDeleter.DeleteProjectLinkRows(ctx, RowBatch) ([]uuid.UUID,error)`；分别接入 `issueRowSteps` 与 `rowSteps`，位置严格为 relations 后、subscribers 前。工作项子树和项目/工作区同步过渡删除都覆盖活 links，同一时刻/删除者，早已删除行保持旧审计。SQL 经 sqlc；项目批次按 id 取锁/键集前进，不绕过目录自查。E2E 的 API 填充随 T3 创建接口可用时加入，不在 T1 调不存在的接口，也不豁免最终目录。

### 2.3 T2：领域、URL 和存储接口

```go
// domain，只有标准库/shared；LinkMetadata 的键和值受下文约束。
type Link struct {
    ID, WorkspaceID, ProjectID, IssueID uuid.UUID
    URL string
    Title *string
    Metadata LinkMetadata
    CreatedByID *uuid.UUID
    CreatedAt, UpdatedAt time.Time
}
type LinkMetadata map[string]*string
type NewLink struct { URL string; Title *string }
type LinkPatch struct { URL *string; SetTitle bool; Title *string }
func CheckNewLink(NewLink) (NewLink,error)
func CheckLinkPatch(LinkPatch) (LinkPatch,error)

// app；读/写端口按消费方组合，不作为一个通用仓储交给所有用例。
type LinkWrite struct { Link domain.Link; By uuid.UUID; At time.Time }
type LinkDeletion struct { ID, By uuid.UUID; At time.Time }
type LinkReader interface { GetLink(context.Context,uuid.UUID) (domain.Link,bool,error) }
type LinkLister interface { Links(context.Context,uuid.UUID) ([]domain.Link,error) }
type LinkURLChecker interface { LinkURLExists(context.Context,uuid.UUID,uuid.UUID,string) (bool,error) }
// LinkURLExists参数依次issueID、excludeLinkID（创建零UUID）、规整URL。
type LinkCreator interface {
    LinkURLChecker
    InsertLink(context.Context,LinkWrite) error
    MarkChanged(context.Context,ChangedRows) error
    InsertActivities(context.Context,[]ActivityRow) error
}
type LinkUpdater interface {
    LinkReader; LinkURLChecker
    UpdateLink(context.Context,LinkWrite) (domain.Link,error)
    MarkChanged(context.Context,ChangedRows) error
    InsertActivities(context.Context,[]ActivityRow) error
}
type LinkDeleter interface {
    LinkReader
    DeleteLink(context.Context,LinkDeletion) error
    MarkChanged(context.Context,ChangedRows) error
    InsertActivities(context.Context,[]ActivityRow) error
}
```

URL 先 TrimSpace；以 `http://`/`https://` 开头的识别不分大小写，**存储仍保留原字符串大小写**；否则补 `http://`。前导显式非 HTTP 的 `scheme://`（按 RFC scheme 字符语法识别）直接拒绝，不补成伪 host；bare `host:port/path` 仍可，路径或 query 内的 `://` 不构成前导 scheme。解析用副本，不能以重新序列化、host 小写、默认端口/fragment 删除等方式改变逐字唯一键。

规整后的 URL 必须是 http(s)、非空 host、合法可拨号端口语法（缺省 80/443；显式十进制 0–65535，不限制业务端口），无 userinfo、无 IPv6 zone、无 NUL/控制字符、长度至多 2048 Unicode 字符；无 DNS/网络访问的域检查。其余不合格 URL 统一422 `url/not_allowed`，不返回解析器细节。标题可 null/空串，原样保存、不 trim，至多255字符；超长 `title/too_long`，NUL `title/invalid_format`。多个字段的问题同一个422。

创建省略 title 与显式 null 都存 NULL；patch 省略 title 保持，null 清除，空串是空串。空 patch、规整后 URL 和 title 均相同属于 no-op，但仍先完成权限与项目/工作项归档判断。URL 不允许 null，缺少创建 URL 或写入 metadata 等未知字段由已有 bodyshape 返回400。

集合固定 `(created_at DESC,id DESC)`，无分页，空集合为 `data:[]`。活行读取含 workspace/project/issue 身份；GetLink 的不存在为 `(zero,false,nil)`，其他错误原样保留链。存储只有在 `PgError.Code == "23P01"` 且 `ConstraintName == "issue_links_issue_id_url_key"` 同时成立时映射为 `ErrLinkExists`；其他23505/23P01、同名不同码、其他约束及非匹配包装错误均保留原链；通过errors.As解包后匹配，不能按错误码一概吞成业务错。`LinkURLExists`仍以原始issue_id/url精确比较、排除当前ID并过滤已删，利用独立issue_id索引；不改成hash值查重。该映射随T3公开码/契约/文案一起引入；T2先保留原PG错误，不制造临时第二种业务错误。

**并发 metadata**：`UpdateLink` 的 SQL 在库内比较旧 url 与参数 url；仅 URL 改变时重置 `{}`，只改 title 时不把 app 曾读到的 metadata 写回。使用 UPDATE RETURNING 返回该写的完整当前 Link，保持响应与该数据库写所得值一致。该性质防止后台抓取在请求读后写入的 metadata 被标题更新覆盖。

### 2.4 T3–T5：API、权限、审计、活动和事件

```go
func NewCreateIssueLink(CreateIssueLinkDeps) *CreateIssueLink
func (*CreateIssueLink) Execute(context.Context,uuid.UUID,domain.NewLink) (domain.Link,error)
func NewListIssueLinks(ListIssueLinksDeps) *ListIssueLinks
func (*ListIssueLinks) Execute(context.Context,uuid.UUID) ([]domain.Link,error)
func NewUpdateIssueLink(UpdateIssueLinkDeps) *UpdateIssueLink
func (*UpdateIssueLink) Execute(context.Context,uuid.UUID,domain.LinkPatch) (domain.Link,error)
func NewDeleteIssueLink(DeleteIssueLinkDeps) *DeleteIssueLink
func (*DeleteIssueLink) Execute(context.Context,uuid.UUID) error
```

Deps：创建 `{Locks, Links LinkCreator, Clock, Events shared.EventPublisher}`；列表 `{Locks, Links LinkLister, ReadTx shared.TxManager}`；修改 `{Locks, Links LinkUpdater, Clock, Events}`；删除 `{Locks, Links LinkDeleter, Clock, Events}`。Locks/Clock 沿既有 app 类型。

| operation | 路径/输入 | 成功 | 声明的业务失败 |
|---|---|---|---|
| listIssueLinks | GET `/api/v0/issues/{issue_id}/links` | 200 IssueLinkCollection | issue.not_found、forbidden |
| createIssueLink | POST 同上；`{url,title?}` | 201 IssueLink | validation_failed、issue.not_found、forbidden、issue.link_exists、project.archived、issue.archived、server_busy |
| updateIssueLink | PATCH `/api/v0/issue-links/{link_id}`；`{url?,title?}` | 200 IssueLink | validation_failed、issue.link_not_found、forbidden、issue.link_exists、project.archived、issue.archived、server_busy |
| deleteIssueLink | DELETE 同上 | 204 | issue.link_not_found、forbidden、project.archived、issue.archived、server_busy |

共通身份401、结构400等沿现有契约。`IssueLink` 输出 `{id,issue_id,url,title,metadata,created_by_id,created_at,updated_at}`；title/created_by_id可null，metadata为下节封闭对象。三个写都声明503及Retry-After。新码 `issue.link_exists`(409)、`issue.link_not_found`(404) 各自首次进入代码的同一 Task 补 `web/apps/web/core/lib/error-messages.ts` 及 `web/packages/i18n/src/locales/{en,zh-CN}/errors.json` 并 test-web；文案表与契约码集合必须相等，不能早一Task先加“未来”的文案。

列表在 ReadTx 中经 `Locks.see(...,ActionRead)`；创建经 `Locks.run` issueID/ActionUpdate/notFound=ErrIssueNotFound；改删先读link，w.issue=它的IssueID，w.action=ActionUpdate，w.notFound=ErrLinkNotFound，w.check预检和锁下都重读link并核所属issue。隐藏规则唯一来源是 `Issue.HiddenFrom(actor,grant.OwnOnly)`：项目guest且guest_view_all_features=false时OwnOnly=true，自己的可见、他人的404（管理员兼项目guest同样）；guest_view_all_features=true时OwnOnly=false，按ActionRead可见的他人项同样可读。写权限仍由ActionUpdate判定，普通guest可见项写403，workspace admin兼项目guest对可见项按管理员规则可写；不可见404始终先于403。项目成员能改删其他成员的link，无作者限制。

顺序：结构/域合法性→资源与权限预检→写闸门→事务项目N→重读/重新判定→项目归档409→工作项归档409→no-op→URL唯一→最后一把锁后Clock.Now一次→MarkChanged(issue)→link写→活动（创建/删除写活动；修改仅URL真实变化写活动，title-only无活动）→事件；创建/改删所有实际业务行同一时刻/写者。`issue` 主行先于 links，符合全局表序。任一写、活动、发布或投递失败全部回滚。拒绝/no-op不发事件、不写活动、不推进审计。

活动精确对应设计17–19：创建 `field=link,verb=created,old_value=nil,new_value=url,old_identifier=nil,new_identifier=linkID`；URL更新 `updated,old_url,new_url,linkID,linkID`；删除 `deleted,old_url,"",nil,nil`。存储comment分别`created a link`、`updated a link`、`deleted the link`（API仍不显示comment）。只改title不写活动，但推进审计和一个updated事件。

三种事件由 domain 常量 `EventLinkCreated/Updated/Deleted` 声明，`issue.EventTypes()` 加入：`issue.link_created`、`issue.link_updated`、`issue.link_deleted`；Entity=`{Kind:"issue_link",ID:linkID}`，顶层workspace/project/actor/time沿当前写，Event.ID每次独立。Changes固定顺序url、title；创建两项，Old/OldID=nil、New为实际值（title null则nil）；修改仅变化字段，Old/New来自锁下旧值与最终值；删除两项，Old为旧值、New=nil。**所有link Change的OldID/NewID均nil**，link身份在Entity，不能把活动identifier拷成事件字段ID。标题独改无url Change。连带删除links只已有issue/project事件，不另逐link发布。

T3同步迁移 `B/events_test.go` 的 `TestIssueEventDirectory`：独立写明现有12类加3种link共15类，核精确序列、无重复、返回切片不可变；link_created不再列入future拒绝表。目录测试构造无订阅者的dispatcher，只证明目录接受/拒绝，不把空payload喂给生产 `issue.Subscribers`。T11另以真实link payload、真实事务/Inserter验证生产订阅目录与投递，不能把目录测试变成无事务的业务调用，也不能削弱生产payload校验来迁就旧测试。

### 2.5 T6–T8：出站安全客户端与配置

```go
// platform/outbound，标准库；平台不引业务或其他平台包。
type Options struct { AllowCIDRs []netip.Prefix; Version string }
type Limits struct { ConnectTimeout, Timeout time.Duration; MaxBytes int64 }
type Response struct { FinalURL string; StatusCode int; ContentType string; Body []byte }
func New(Options) (*Client,error)
func (*Client) Get(context.Context,string,Limits) (Response,error)
// 分类错误经errors.Is识别；包装链供分类，不直接把error字符串写入日志。
var ErrBlocked, ErrTimeout, ErrUnreachable, ErrTooLarge error
// 仅返回空字符串或受控 redirect_limit，支持标准错误包装。
func Reason(error) string
// config；不依赖outbound，启动验证自己解析CIDR。
type OutboundConfig struct { AllowCIDRs []string `koanf:"allow_cidrs"` }
```

构造时读取 `net.InterfaceAddrs` 成功后固定本机地址快照；失败返回错误，不当空集合。无全局可变策略。Options传入的prefix复制持有，Version从bootstrap现有版本来源给出，outbound不为取版本跨platform导入。Get的Limits均须正数，MaxBytes还须小于MaxInt64以保留越界探测字节；错误为调用者程序错误，不能退化为无限制。

拒绝集为两张 IANA special-purpose 登记表的**所有**prefix（含 globally reachable=true、已终止保留记录），加 IPv4/IPv6 multicast 和 `fec0::/10`；各prefix显式保留，不只用IsPrivate/IsGlobalUnicast。2026-10-26官方页面最后更新2025-10-09，CSV精准来源/hash见附录；实施时核对是否更新并记录实际采用快照。mapped IPv6、NAT64、6to4、Teredo外层与内嵌IPv4均验证；外层被拒不能先Unmap绕过。NAT64 local `/48` 整体拒绝；Teredo检查server和反转的client。固定 NAT64 `64:ff9b::/96` 按已知格式检查内嵌 IPv4；local `64:ff9b:1::/48` 是技术无关预留，不推断载荷存在、位置或算法，默认整段拒绝（[RFC 8215 §5](https://www.rfc-editor.org/rfc/rfc8215.html#section-5)、[RFC 6052 §2.2](https://www.rfc-editor.org/rfc/rfc6052.html#section-2.2)）。测试放行仅对该原地址落在显式allow prefix时覆盖静态/本机拒绝；放行IPv4 127.0.0.1/32不放行整段127/8，也不自动放行IPv6嵌入形式。显式放行已知编码外层只覆盖该原 literal 的静态/本机拒绝，递归载荷不继承 allow；公网载荷仍可通过，私网/本机载荷仍拒绝。

每次GET/每跳只许http(s)，拒绝userinfo、IPv6zone、无host/坏端口；合法hostname经标准库Transport规范为IDNA名称后解析全部A/AAAA，任一不合格即在任何实际socket dial前拒绝；每跳在DialContext/DialTLSContext收到的规范地址边界只解析一次。literal不额外DNS。实际拨号只向本轮已验证IP字面值，不把hostname交回Dialer；HTTP Host、TLS SNI与证书检查使用同一规范hostname，原URL语义保持，绝不InsecureSkipVerify。不得读取代理环境变量（Proxy=nil），无CookieJar、不转发Set-Cookie、Authorization/Proxy-Authorization，不接受调用者自带认证header。

手动跟随301/302/303/307/308，最多5次跳转（初始+5次GET），每一跳都重新解析检查、相对Location按当前URL解析；第6次跳转作为不能完成请求的失败返回（包装ErrUnreachable，日志原因为redirect_limit，metadata.failed），非HTTP目标或userinfo/zone为Blocked。不发Referer，跨站/降级跳转也不会带任何凭据。中间响应及时close，不为复用无限drain body；每次Get结束close自己持有的连接/transport资源。只发GET，User-Agent=`Nerve/<version> (+link preview)`。

ConnectTimeout=2s限制建连，Timeout=5s覆盖解析、全部跳转、TLS、header、body；上层context更短时以其为准。正文使用最多MaxBytes+1判越界，只把<=MaxBytes的body返回；不能信任Content-Length，解压后也受限。HTTP非2xx仍交Response由crawler归类，网络/地址/超时/超量按四种分类保留错误链。

`outbound.allow_cidrs` 默认空，只在env=test可非空；其余环境任何非空或格式错误启动失败。`config.test.yaml`固定127.0.0.1/32，使worker nerve与nerveWith一致。boot函数 `newOutbound(cfg config.Config, version string) (*outbound.Client,error)`只负责解析已验证CIDR和New；T10接入真正server组合，CLI不构造它。测试用同包私有依赖注入resolver/dial/interface/TLS roots，不加生产绕过开关。

### 2.6 T9：抓取元数据

app声明 `type LinkCrawler interface { Crawl(context.Context,string) domain.LinkMetadata }`；crawler adapter实现 `New(client *outbound.Client, logger *slog.Logger) *Crawler`，`(*Crawler).Crawl(ctx,url string) domain.LinkMetadata`。app不importhttp、outbound或HTML库。网络诊断由adapter结构化日志记录受控阶段与错误类别，公开元数据不暴露端口探测细分。日志不得记录原始URL、query、userinfo、响应body或未经分类的 `error.Error()`（其包装链可能含URL）；对所有级别及消息/属性同样适用。阶段固定为page或icon，类别区分blocked、timeout、unreachable、too_large、http_status、non_html、invalid_icon_type；adapter先用errors.Is分类，仅在outbound.Reason(err)返回固定redirect_limit时记录该诊断理由；不解析原始错误文本，仍映射公开failed，重定向被拒保留blocked与所在page/icon阶段。不要以完全不记日志通过保密测试。

T9的 `TestCrawlerLogsRedacted` 使用捕获所有级别的 `slog` handler，分别驱动页面失败、重定向被拒/超限、图标失败；在输入query、被拒userinfo、重定向Location、图标href和响应body放互不相同哨兵。断言完整序列化日志中均不出现这些哨兵、原始URL或原始错误文本，同时每条预期诊断保留对应page/icon阶段与受控分类（含redirect_limit）。再临时恢复记录原始URL或含URL的原始error的生产变异，证明该测试变红；恢复字节/hash后转绿。

整个Crawl共用5秒context，页面和图标不会各重获5秒；每次Get的ConnectTimeout2s、Timeout5s受父context约束。页面正文1MiB，成功2xx且Content-Type经mime解析为text/html才解析；接受的页面先用已有 `html/charset` 转为 UTF-8，再用 `golang.org/x/net/html` 取第一个title的文本，实体只解码一次，TrimSpace后按Unicode截到255；缺/空title为null。不运行脚本、不访问其他资源、不跟随HTML refresh/base改变安全判定。解码遵循 BOM→HTTP charset→前1024字节内有效meta；库默认Windows-1252而全文合法UTF-8、无有效声明时保留UTF-8，避免长ASCII前缀后的中文乱码。显式单字节声明即使正文碰巧是合法UTF-8也优先；重复属性、pragma和无效label按库的meta语义识别。1MiB是解压后、转码前的响应字节上限；不是HTML解析的硬CPU期限。

图标按优先级icon→shortcut icon→apple-touch-icon→apple-touch-icon-precomposed，大小写不敏感rel按token语义；同类取首个非空href；相对地址基于页面最终URL（忽略base）；没有候选取同origin `/favicon.ico`。只尝试一个候选，不遍历多URL，GET上限64KiB。MIME允许image/png、image/x-icon、image/vnd.microsoft.icon、image/gif、image/jpeg、image/webp；去参数、规范小写；拒绝SVG，不另存默认图标。成功favicon=`data:<mime>;base64,<bytes>`、favicon_url为图标成功请求最终URL；失败/被拒/过大/非法mime的favicon和favicon_url均null，保留页面标题。

metadata形状：初始/改URL为非nil空map `{}`；页面成功恰好 `{title,favicon,url,favicon_url}`，三个可空项也存在，url保存本次任务的原规整URL；页面被outbound拒绝为 `{error:"blocked",url}`，其他失败（超时/连接/状态/超量/非HTML）为 `{error:"failed",url}`。图标失败遵循上一段，不将页面成功降为失败。公开 `error` 可省略、可null，非null仅 `blocked` / `failed`；canonical用独立enum组件加该字段的nullable `oneOf`/ref，沿既有生成规则，不把整个metadata改成状态联合。域/数据库仍是五键string/null通用map；不用任意外部解析对象。网络失败不重试，只写粗粒度结果。

### 2.7 T10–T11：守卫写回、worker 与订阅者

```go
// app
type LinkMetadataWriter interface {
    SetLinkMetadata(context.Context,uuid.UUID,string,domain.LinkMetadata) (bool,error)
}
type CrawlLinkDeps struct { Crawler LinkCrawler; Links LinkMetadataWriter }
func NewCrawlLink(CrawlLinkDeps) *CrawlLink
func (*CrawlLink) Execute(context.Context,uuid.UUID,string) error
type LinkCrawls interface { Enqueue(context.Context,uuid.UUID,string) error }
func NewCrawlOnLinkChange(LinkCrawls) shared.EventSubscriber
// issue/adapter/river
const CrawlLinkKind = "issue.crawl_link"
const CrawlLinkMaxAttempts = 3
type CrawlLinkArgs struct { LinkID uuid.UUID `json:"link_id"`; URL string `json:"url"` }
func (CrawlLinkArgs) Kind() string
func (CrawlLinkArgs) InsertOpts() river.InsertOpts // default队列、CrawlLinkMaxAttempts
type CrawlLinkUseCase interface { Execute(context.Context,uuid.UUID,string) error }
func CrawlLinkJob(CrawlLinkUseCase) jobs.Job
func NewLinkCrawls(Inserter) *LinkCrawls
func (*LinkCrawls) Enqueue(context.Context,uuid.UUID,string) error
```

写回只一条 `UPDATE issue_links SET metadata=$m WHERE id=$id AND url=$url AND deleted_at IS NULL`；不写审计/活动/事件，不取其他锁/项目闸门，不先查后无守卫地写。0行成功；DB错误返回原链给River，最多3次attempt。网络失败写metadata并完成，不人为返回网络错误制造重试。取消/停止若数据库拒绝当前context，正常返回该错误，不脱离context写库。已删除项目但link尚未删的窗口允许metadata更新（设计§3.5），不另加入project JOIN。

worker在default，无周期；`issue.Module.Jobs()`增加worker，`EnqueuedKinds()`准确返回crawl kind；既有 `jobs.New`校验漏worker。module New构造crawler和CrawlLink，bootstrap把outbound/logger传入新Deps，构造失败关闭已有pool，CLI/Provide不构造客户端。

订阅者名 `crawl_on_link_change`，只订阅created/updated。created取url Change.New投一次；updated仅实际url Change投一次；title-only没有url Change，成功且不投；删除不订阅。created缺url、两种订阅类型出现重复url或url Change.New为nil，是程序错误，原事务失败，不安静漏任务。LinkID取Entity.ID而非Event.ID；订阅者不再读link、不抓网络、不取锁。适配器以既有Inserter.InsertTx入队 `{link_id,url}`，`Queue:"default",MaxAttempts:3`，不去重不同改动，不使用独立pool.Insert；`issue.Subscribers`仍在issue.New之前可构造。

原URL A→B→A 时，旧A任务可能满足已批准的id+url守卫；本期接受同URL元数据结果最后完成者写入，不擅加version列/事件generation。这不允许B结果写入A。

### 2.8 T12–T15：组合与端到端证据

`registrationRows`实现在B/invitations_test.go，table_rows_test.go提供目录和反例。保留普通表全行快照；另按各模块声明的请求kind读取river_job的稳定投递身份/载荷（id、kind、args，按id），忽略state/attempt等worker自行变化字段和periodic/leader噪声。独立反例必须证明：新增/替换请求任务载荷会被看到，周期调度不会误报；被拒注册不增加任何请求任务。

运行角色实际创建/改/读/删link并完成一个crawl job，独立预期kind含crawl，不从实际Jobs反推预期集合。事务测试真实PG/River证明未提交不可见、commit业务与job同时存在、rollback全部无、后续订阅者失败也无；字段比对包括link ID/url/actor/time和无额外任务。

交错19用真实两个连接与受控crawler/HTTP同步点：A抓取暂停→URL改B提交→旧A写回0→B抓取成功；反向顺序先A完成再改B清空；标题请求已读旧metadata时crawler提交新metadata，标题UPDATE RETURNING仍保留该新metadata；先删除/删除重建同URL的新ID不被旧任务影响。观测数据库锁等待或通道条件，有期限、不sleep；`-race -count=5`，无40P01。

I10经API准备用户/工作区/项目/issue/link，本地页面server绑定127.0.0.1临时端口并可受控放行首响应；地址输入不带协议，断言201metadata{}、规范URL、最终标题/图标/dataURL、对应args的job completed、URL更换重抓、标题独改不新增任务、重复409、删除活动。169.254.169.254只断言blocked，“0连接”在outbound Go测试证明。I17扩两层子树link，furnish经API让项目/工作区删除目录同样覆盖。

T15逐列活动/事件矩阵、至少15项代表性真实生产变异（地址/重绑定/跳转/限额/权限/锁/事务/metadataguard），每项有原源码SHA、变异补丁/命令/实际失败与恢复绿；不同任务已有足够证据可直接复用，不能用仅改测试的红替代。完整门禁按02串行一次，只有后续变更/失败/未解风险才扩重跑。

## 3. 与设计的差异及局部裁定

以下是推进者在已批准架构内的细化，未冒称负责人逐条作出的决定；没有未决架构项。

| 裁定 | 理由 | 错了的代价 / 验证 |
|---|---|---|
| 12粗项展开16任务；保留紧凑计划、静态预检，逐Task真实红绿/生产变异与独评替代预制原型/逐字重放 | 02§3明确可调整方法；安全HTTP与地址策略分开审 | 预检不能证明运行，所有验收保持未勾，T15实际门禁 |
| 活 `(issue_id,url)` 排重改为单一原文组合表达式的内置hash exclusion，沿既有text等值；仅该名+23P01映射409 | 最窄PG18.6实验重现2048字符8135字节URL被B-tree拒绝；候选支持原文相等重检、碰撞不同原文并存、sqlc1.31.1生成，不缩短公开长度、不增加12列之外字段或依赖 | 索引类型/SQLSTATE改变，须同步M4§3.11/§4.8及差异；hash查找需要堆行复查，不替代issue_id索引；直接并发重复写可能等待/死锁，生产仍由既有项目N锁串行，T5/T13另取运行证据 |
| URL只TrimSpace/必要时补前缀；识别http(s)不分大小写但存储原文；显式其他scheme://拒绝 | 保持设计逐字唯一，避免http://ftp://host误解析 | 同host大小写URL可能独立存行，按既定逐字规则测试；路径/query含://保留 |
| userinfo/IPv6zone在写入检查和outbound均拒绝；不折叠host/默认端口/fragment | §6.8不带凭据与普通公网地址约束，避免歧义/泄漏 | 合法带凭据链接被拒是预期安全边界，422与blocked分别验证 |
| title不trim，null/空/省略分开；noop先过归档判断 | Plane可空标题，现有patch范式与归档只读 | 比较/清空错误会写多余审计，T2/T4覆盖 |
| 集合按created_at DESC,id DESC | Plane -created_at，增加同刻稳定顺序 | 次序可见，逆id与同刻数据证明 |
| 标题UpdateLink只在库内URL变化时清metadata，并RETURNING当前Link | worker只锁link，不能让旧metadata覆盖其新结果 | 类型/SQL/应答必须贯穿一致，T13真交错 |
| metadata采用纯 `map[string]*string`，仅受控五键；成功四键齐、失败两键齐、pending空对象 | 精确表达缺键/null/值，不引第三方nullable到domain | 任意键可能漂移，域/SQL/生成schema闭合表与返回shape测试 |
| 整次crawl共享5秒；图标失败保留页面，favicon/favicon_url为null | §3.20一次抓取5秒，§3.11否则默认图标 | 图标网络慢可能耗尽剩余预算，但标题保留，T9验证 |
| icon按rel优先级、成功favicon_url记最终URL，metadata.url仍原任务URL；不使用HTML base | 明确Plane选择法和安全解析基准，无多余外发 | 不同图标结果可见，重定向/相对href测试 |
| IANA冻结本次官方CSV并以独立fixtures核完整性；全表拒绝，不只private | 设计已经要求每网段，不能遗漏新分配项 | 表未来会更新，来源/日期/hash和移交维护责任明确 |
| allow只作用原地址，不自动放行嵌入IPv4；接口读取失败阻止构造 | 最窄测试出口、fail closed | 某些测试需显式IPv6allow，不扩大生产权限 |
| A→B→A遵循原id+url守卫，不新增generation | 守卫已批准且metadata为可刷新预览，不改变模型 | 同URL旧抓取可晚覆盖同URL新结果，明确限制，不声称版本严格新鲜 |
| E2E链接填充随T3 API可用到来，T1先做存储目录；link_exists随首次创建补文案 | 解决粗任务内部依赖，不改阶段交付 | T1至T3不能声称全E2E通过，最终全覆盖无豁免 |

## 4. 逐项验收与证据

以下产品项依据 T1–T15 真实执行、逐 Task 独评及两矩阵勾选；T16新增边界另列本轮证据。勾选不等于最终分支CI或main集成；准确提交/命令/计数见 [review §2](../reviews/P7-issue-links-review.md#2-验收标准逐条核对)。

- [x] 12列迁移/CHECK/hash exclusion精确排重/FK索引/grants及两个runtime守卫通过（T1/T12）。
- [x] 子树/项目/工作区两路删除覆盖links，全局顺序、同刻同写者、已删对照不变（T1/T14）。
- [x] URL和标题规整/长度/null/多问题、集合稳定次序与完整封闭契约通过（T2–T4）。
- [x] 2048字符高熵多字节完整INSERT/UPDATE与API201/200、2049CHECK/422、精确重复23P01/409、跨issue/软删复用和真实hash碰撞不同原文共存通过；只映射指定约束名+23P01（T1–T4）。
- [x] 四操作的HTTP码双向覆盖、三写503及双语错误文案通过（T3–T5）。
- [x] 全身份/隐藏404逐值相同、拒绝无锁无闸门、锁下重授权、归档优先/noop与回滚通过（T3–T5）。
- [x] 三种link活动与三事件完整四槽、标题独改无动态/无重抓、审计同刻通过（T3–T5/T11）。
- [x] IANA每prefix首尾/清单完整、全部DNS候选、嵌入IPv4、本机地址、blocked 0 dial通过（T6/T7）。
- [x] 固定IP/Host/SNI/证书、每跳重检、5跳、无代理/cookie/凭据、2s/5s和响应限额通过（T7）。
- [x] test-only allow配置、解析失败拒绝、127.0.0.1/32各nerve一致，CLI不构造出站（T8/T10）。
- [x] HTML标题/6类图标/dataURL/相对地址/1MiB与64KiB/5s总限、失败仅blocked/failed通过（T9）。
- [x] 捕获slog证明页面/重定向/图标失败的阶段分类保留，query/userinfo/body哨兵及原始URL/error无泄漏，原始日志生产变异红且恢复绿（T9）。
- [x] 15类事件目录独立核对，不将空payload交生产订阅者；真实payload/事务投递另有组合测试（T3/T11）。
- [x] worker default/3attempt配置、声明完整、真实completed、DB/context错误原链返回及网络失败完成通过（未人为注入DB故障观察真实River三次attempt）（T10–T12）。
- [x] 订阅者原事务投递：未提交不可见、commit留、rollback/后订阅失败均不留（T11）。
- [x] registrationRows看到请求任务且忽略周期状态变化，反例不空过（T12）。
- [x] 交错19、标题保持新metadata、删除/重建守卫、审计不变，race-count5无40P01（T13）。
- [x] I10 API含169.254.169.254 blocked；I17/项目/工作区删除含link通过（T14）。
- [x] T15发布/活动矩阵、21项历史生产变异/恢复及完整七门禁；原范围deep-review全文覆盖完成。
- [x] T16修复准确提交完整本地七门禁通过（tested `cd0fce7ed4f18739a60fcf3d0c44f811b809b7b8`）。
- [x] 原8项修复经独立定向复审全部 ADDRESSED（`d2a3cf26..7e903ecb`）；T16-DOC-1历史时态文案已纠正，本次纯文档增量确认见主控最终记录。
- [x] §3.23所有P7文档/差异/交接处理结果为真；最终分支 `023c96e8` 的 CI 40400860045 已全绿，T16-DOC-1 增量复审已关闭（T16/交付）。
- [x] 经授权 `--no-ff` 合并，准确 main `166b9efd` 的 CI 40451632280 全绿；5,443 个 ignored 证据文件归档校验后清理本地/远端分支与 worktree（阶段集成）。

## 5. 不在范围内与后续 Phase

| 接收方 | 保留责任 |
|---|---|
| P8 | Issue.link_count、两个排序码与列表SQL；草稿/搜索 |
| P9 | project/workspace删除事件、后台删除/清理/版本保留；本期级联仍同步过渡 |
| P11/P14 | 生成类型接入详情store/service与页面，I10/I17页面版本、链接弹窗表单可达性/卸载清理 |
| M8 | 复用outbound进行Webhook投送；IANA更新/网络出站策略/公网端口探测边界部署说明 |
| M4收尾 | §14整体逐项证据、所有输入handoff关闭、M5–M8正式M4-closeout，v0§9.4完成 |

P7没有新事件持久化、抓取进度接口、重试按钮、生产allowlist、动态抓取配置、favicon缓存、任意HTTP方法/headers或跨模块JOIN，不把被阻止的地址拒绝为HTTP创建失败（语法合法内网URL仍可创建，随后metadata blocked）。

### P8接收回执（2026-10-27）

T13活issue_links计数进入全部Issue/Detail、Get/GetMany/SubIssues/search及三列表，0显式，URL/title/metadata.error或worker更新不改数量；新增±link_count成当前21序及整数游标。T13 M33–36、公开全表示/权限/精度，T14双向0/1/2/2、软删后detail/search一致和真实M33 binary红绿已验证。P7原出站/事件/集成不重判，P9/P11/P14/P15/M8仍开放。 实际命令/计数/变异/检查点见[P8 review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)。产品HEAD `10f44790`、首次整审 `166b9efd..10f44790`；文档独评/最终准确HEAD CI按后续收据追踪。未合并，不改变本页历史验收与集成事实，输入handoff总体仍open。

## 6. 风险与限制

- 包内只能拒绝启动时网卡地址；NAT/LB公网地址、启动后网卡变化由网络层出站策略覆盖。普通公网任意端口仍可被请求，metadata错误粗化不是全面阻止端口探测；README必须如实说明。
- IANA策略快照需维护，不能声称永远与登记表同步；嵌入地址测试不能仅从生产常量生成，重叠prefix要用清单核对。
- 链接集合不分页、数量无上限是既定集合接口；抓取受default固定worker并发控制，不增加每链接速率桶。每次crawl最多两个资源，每个资源初始GET加至多5次跳转，页面/图标共享5秒；authenticated限制到达率，固定worker限制并发，都不是待执行job数量的硬上限。反复修改URL可投多个任务，DB失败重试也可再次抓取。
- hash exclusion的直接并发重复写可能等待并回滚死锁一方；生产同项目写仍先取既有项目N锁，metadata worker不改issue_id/url。T5/T13要证明这条既有串行路径，不能以最窄实验替代并发验收；不新增绕过项目锁的直写入口。
- URL逐字唯一允许同资源不同拼写；A→B→A同URL旧结果可晚写。安全始终按实际请求的URL再次检查，不依赖API校验。
- T1加入表而T3才API填充的短暂状态仅限阶段任务推进；最终目录测试不可豁免。初期仅静态预检不能证明网络/事务/锁性质。
- 局部问题按SOLID根因修复；若实现要求放宽架构守卫、另造锁路径、改事务协议或出现超出已批准边界的高风险，则先提交完成工作再向负责人上报。

## 7. 输入交接与关闭条件

| 输入 | 本期处理 | 仍开放 |
|---|---|---|
| P6 spec§5/review§6 | 3种link事件、首个生产订阅者/LinkCrawls、真实请求任务/worker接线 | P9删除任务与P15页面 |
| M2-closeout§3 | 链接写推进issue.updated_at，锁后单一时刻、真实审计和回滚测试 | 其余原落点 |
| M2-closeout§6 | 第一生产订阅者InsertTx、worker声明与grants/runtime真实证据 | 其他后续任务 |
| M3-closeout§1 | registrationRows另核请求kind的river_job，并有反例 | 邀请同步规则保留；项目/工作区异步迁移P9 |
| M4§3.23/P7 | v0§6.2/§8新增outbound说明；plane-diff链接12列和抓取差异；README安全 | M4整体仍进行中 |

阶段首提交设关键词phase=M4/P7，核到期例外，不能随手延后。每个处理结果附证据并写剩余责任，handoff整体status保持open至M4收尾。阶段终审由Codex使用deep-review，局部发现修复后复审；不再等Claude，但阶段实现完成后不自动合并，等待负责人下一步。

## 附录 A：准备期证据边界与来源

只读盘点基于P6实现HEAD `e8df8485eee76cd83b93bc4a2ad48b6e3d89240f`；P6后续文档已集成为页首的准确main基线。读了M4上述条目、现存Locks/Store/事件/worker/配置/生成链、Plane链接model/serializer/view/task及url_security/ip_address；记录在准备台账 `p7-scope-inventory.md`。作者采用writing-plans自查，独立安全分区对v3的S2修订复审Approved，独立集成分区对v4的I1/I2定向复审Approved；全部5项Medium闭环，正式结论见review。**文档预检不等于产品运行验收。**

独立集成审者另在2026-10-26运行最窄可行性实验，并以固定UUID重复验证：PostgreSQL18.6、8KiB页、两个4列表（id/issue_id/url/deleted_at）。2048字符/8135字节的合法高熵URL在原B-tree上报54000（索引项8168字节>2704上限）；原文组合hash exclusion接受它，精确重复INSERT/UPDATE报指定约束23P01，跨issue/软删复用成功、2049字符CHECK拒绝，实际hash碰撞的不同原文可同时存在。sqlc1.31.1在首轮对该候选DDL生成成功；固定UUID复验未改schema/query/config，沿用该次生成物，没有重复运行sqlc。它验证的是局部约束可行性，**没有运行P7完整12列迁移、FK/grants、真实API、项目锁并发、crawler或完整门禁，也不是完整原型/逐Task重放**。

原始实验归档 `index-probe/manifest.json` SHA-256为 `d24aef42e4bf5ab681f8ffe60d9f193e94ae62aded61f8aed5c396fb50eac3e7`；固定输入复验归档 `index-probe-v2/manifest.json` SHA为 `eb0a6dba9e7bf35278e18712be1a4f8f101f21ab1111a7db834d0788a1407469`，保留前轮而不覆盖。两份manifest逐件固定schema、源码、生成物和raw log。相同 `schema.sql` SHA为 `2061f330589be8f6a74267efc14be34614c6d5b053636d07abb824b45cfb6a5e`；固定复验 `go-test-fixed.log` SHA为 `00465bead76d321ae4c8db0f4afaf2f7a93b662eac049df3bbeb7bc170c8c638`。正式T1/T2直接使用§2.2的固定UUID/URL/hash对照，并先独立验证hash确实相同、原文确实不同；不在每轮测试中搜索500000个候选，也不把碰撞对套给任意issue ID。

IANA官方CSV于2026-10-26获取（页面Last Updated 2025-10-09）：

| 来源 | SHA-256 |
|---|---|
| [IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry-1.csv) | `cbc4b7653c8bc640ba1e8e5d4ebbf11e7c7a1d867e7f1593aad001d3f8632aac` |
| [IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry-1.csv) | `bf56772aef97fbf9b4a27c98b8511f2bb1ba96da4b9ccc12ab203a2496faba66` |

原始CSV、精确UTC获取时刻/内容类型/字节数与hash在ignored `p7-iana-snapshot/manifest.json`；复制至P7证据台账后保留，实施T6采用前再次核对。除上述独立最窄PG/sqlc实验外，未执行P7产品迁移、网络目标抓取或完整生成/测试；本草稿编写者只读实验归档并修订文档，未运行实验或产品。

T6 采用冻结前重核的两份官方 CSV：2026-10-26 13:59:10–13:59:11 UTC，均为原 URL 的 HTTP 200，与本附录快照逐字节相同；产品测试中的 `testdata/sources.json` 记录准确访问时刻、URL、大小和 SHA。独立 CSV 展开 50 条记录、51 个 prefix，加规格额外 3 项；清单对照与逐项首尾行为分开验证。该冻结记录只覆盖T6纯地址策略；T7后续真实GET、全部DNS候选、0 dial与接口读取失败证据见行为矩阵，不倒写为T6已执行。

## 2026-10-27 集成回执

**当前状态（2026-10-27）：P7 已完成。** 经负责人授权以 `--no-ff` 合并为 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`，[main CI 40451632280](https://github.com/open-nerve/NerveProject/actions/runs/40451632280) 的 server/web/e2e 全绿；5,443 个证据文件归档校验后，本地/远端分支及 worktree 已清理。集成证据见 [P7 review](../reviews/P7-issue-links-review.md) 的集成回执；M4 总体及后续交接仍开放。

上述交付前记录保留原审查范围与当时事实；最终 `7e903ecb..023c96e8` 文档增量复审已通过，原8项与T16-DOC-1均已关闭。完整本地门禁的产品源码与最终分支/合并树身份已核对，不冒称另跑了未变套件。原始证据现位于项目根目录 `.superpowers/sdd/P7-integration-20261027/worktree-evidence/.superpowers/sdd/P7-issue-links/`；旧 `E` 路径是阶段执行时的位置。
