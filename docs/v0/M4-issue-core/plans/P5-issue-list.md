# M4/P5 工作项列表与显示设置实施计划

> **执行方式：** 按 Task 使用 superpowers:subagent-driven-development 或 superpowers:executing-plans。状态：实施中；独立静态预检及修订复核通过。已执行步骤按各Task勾选，真实证据见阶段评审。P5 起步/局部实现按既有自主授权与独立预检继续；本阶段完成后先交负责人安排整分支终审，再处理意见并合并，除非负责人另作安排。确需架构裁定仍上报。

**目标：** 交付 M4 §12/P5 的列表引擎、两个 API、可见性目录及关联替换、sort_order PATCH、三类项目偏好、性能和接口故事。

**架构：** issue/domain 持有词汇；issue/app 经各模块小端口读目录，在已有 ReadOnlySnapshot 的同一 ctx 内用三条静态 sqlc 查询；project 经其声明的 IssueListSettings 复用纯校验。既有 PageSize、Historical、Locks.run、Issue 列表 DTO 直接消费。

**技术栈：** 仓库锁定 Go 1.27/toolchain go1.27.1、pgx 5.11.0、sqlc 1.31.1、oapi-codegen 2.8.0、PostgreSQL 18.6、Node 24、pnpm 11.10.0、Playwright；不新增依赖。

**规格：** [P5 spec](../specs/P5-issue-list.md)；[P5 review](../reviews/P5-issue-list-review.md)。基线 `e9782b8f7e7571b39a4eb8924b4b72a6e60eac51`；[main CI 40180521025](https://github.com/open-nerve/NerveProject/actions/runs/40180521025) 的 server/web/e2e 已全部成功，仅证明P4基线。16 Task 是原设计15项的局部重切，sort_order单列，全部P5范围保留。

**方法调整：** 依 [02-method §3](../../claude_to_gpt/02-method.md#3-哪些是要求哪些是做法)，采用明确职责、消费/产出签名、文件、测试和命令的紧凑计划；逐Task TDD、实际变异、恢复控制组和独立评审替代完整预制原型/逐字重放。没有跑原型就不声称跑过；不减少纪律、真实文档、性质证伪、交接、终审或合并/main CI完成门。

## Global Constraints

- P4 已合并且 main CI 通过；起步使用 `.codex/worktress/codex/m4-p5` / `codex/m4-p5`，主目录 main 不动。不用 stash/clean/reset --hard；不动 plane/refer，不操作既有容器、不访问凭据。
- 启动即落正式 spec/plan 与 review 记录，`tools/keywords.json` phase 改为 M4/P5；不要到 T16 才写计划。起步核对基线与本计划的局部接缝，处理独立预检意见并记录复审结论；不虚称已通过。
- 每 Task TDD→关键变异→独立评审→局部修复→提交；中文消息，保留 feat(M4/P5) 等前缀；push 用 `/usr/bin/git`。P4 本轮旧 Claude 等待门已被负责人新指示替代；不自动取消 P5 后续完成门。P5 完成后先交负责人安排整分支终审，处理后合并，除非另有安排。
- 写复用 Locks.run 与闸门；不可见404在角色403前，项目归档→工作项归档→无改动→依赖→时钟→写。全局表序沿 P4：issues→assignees→labels→relations→links→subscribers→mentions→comments→issue_reactions→comment_reactions→activities→description_versions。
- 模块不互导、不跨模块 JOIN；SQL 仅 sqlc；三条静态查询，不拼 SQL、不写模块 SET、不加 raw pgx 豁免、不提前索引特化。端口错误原样传播，ctx 不逃逸。新逻辑文件约400行以内，生成物沿既有例外。
- 读必须独立 ReadTx，所有鉴权/目录/SQL/补 total 传事务 ctx；不能把 ReadOnlySnapshot 嵌进已有写事务来升级隔离。PageSize 默认50、1–100唯一来源 shared。
- 无新增依赖；生成物只经 make gen/gen-go，提交后 make gen-check。不能手改生成代码；闭合 body/schema，不加 oneOf/allOf/开放 map schema。
- 新 API 当 Task 加权限矩阵、HTTP 双向契约；只修改已有 PATCH 不创造新 operation/503 枚举，仍扩该写的锁与归档反例。新增错误码才同 Task 改 error-messages/en/zh-CN；复用 workspace.member_not_found 也要在 issue HTTP 覆盖。
- 每个命令须确认确有匹配测试；空匹配不是通过。测试名为拟新增名字，实施者须以最终文件核对。一次只跑一套容器 suite，使用 pgtest 模板库；变异恢复后复跑控制组，不固定 sleep 协调。
- 以下命令用于P5实施，执行状态按各Task勾选；每Task在review登记实际HEAD、命令、匹配测试、红因、绿结果、变异差异/失败/恢复结果、独立评审及修复，不能只记“通过”。

## Review Focus

1. 多值过滤与分组相互覆盖、泳道总数把格相加、分页后数组不完整；T5–7/T9/T15 的 A∩C、双标签×双负责人固定反例。
2. 目录与查询不在同快照，普通目录复用 Historical，活跃成员名单丢历史负责人，管理员绕 OwnOnly；T4/T10–12/T14 的可见性和屏障交错。
3. 微秒/数值经 float64 丢失、SQL CASE 被 sort_order float8 分支隐式提升、null 降序跑前面、同键 id 方向错、最小名字降序变最大；T3/T5/T12 用19排序、整数精度边界、相邻可表示 sort_order、逆 ID、两方向/null/Unicode验证。可统一numeric并显式转换，游标与SQL键须同一全序，未验证前不宣称成立。
4. 原始 query 重复被生成绑定提前变400，日期重复被压扁，错误映射波及其他端点/坏path；T9 按既定“绑定先于认证”验证局部映射和未经改写的原始query。
5. 偏好 PATCH 整行覆盖丢并发修改、CHECK 对非对象抛类型错误、sort_order 没进入 Changes/skip互斥；T8/T13/T14 的并发、直接 SQL 坏形状与 true/false skip 变异。

## 文件结构、职责与依赖

下文 I=`server/internal/modules/issue`，P=`server/internal/modules/project`，W=`server/internal/modules/workspace`，B=`server/internal/bootstrap`。路径均相对 P5 worktree；新路径为计划，最终命名变更须同步本表与测试命令，不放任上下 Task 漂移。

| Task | 主文件/职责 | 依赖 |
|---|---|---|
| 1 | platform/postgres/pool.go、pool_test.go：连接运行参数 | 基线 |
| 2 | I/domain/listvocab.go、list_date.go及测试：词汇、页面能力和单一日期原子解析 | 基线 |
| 3 | I/domain/list_query.go、list_order.go、list_cursor.go、测试；api/testdata/issue-order.json | 2 |
| 4 | I/app/visible_projects.go/list_ports.go；P 的 issue_ports/普通目录；B/ports.go；负责人 ID 查询 | 2、3 |
| 5 | I/app/listing.go；I/adapter/postgres/list_issues.go、queries/list_issues.sql、测试 | 3、4 |
| 6 | I/adapter/postgres/list_issue_groups.go、queries/list_issue_groups.sql、测试 | 5 |
| 7 | I/adapter/postgres/list_issue_subgroups.go、queries/list_issue_subgroups.sql、测试 | 6 |
| 8 | I/domain/patch.go/description.go；app/update_issue.go/rows.go；postgres/update.go/queries/update.sql；HTTP/API | P4 |
| 9 | I/app/list_issues.go；HTTP/list_issues.go/list_query.go/handler.go；module.go；issue.yaml/openapi.yaml | 5–7 |
| 10 | I/app/list_member_issues.go；W目录；access规则；HTTP/module/bootstrap接线 | 4、9 |
| 11 | I/app/list_relations.go/seen.go；普通可见性替换，删除旧未消费端口 | 4、9、10 |
| 12 | B/issue_list_pages_test.go、issue_list_snapshot_test.go、issue_list_scale_test.go | 5–7、9–11 |
| 13 | 00026迁移；P/domain/app/http/postgres preferences；IssueListSettings纯适配与接线；project.yaml | 2、3 |
| 14 | B权限矩阵/运行角色/跨层回归；archtest/contract/fake fixture | 8–13 |
| 15 | I3–I7/I19/I20 API故事、I15增长；e2e fixtures/assert | 14 |
| 16 | 设计/差异/交接/review/最终验证与合并材料 | 15、实际证据 |

### 起步规模估计与共用夹具

估计依据是已落地P3/P4的相近职责，不是不存在的P5完整原型：当前P3关联列表用例90行、其app测试171行、组合列表测试107行；P4动态分页用例76行、postgres适配36行、SQL17行、组合翻页测试216行（当前文件wc实读，属于代表文件，不是整Task差异量）。P3 spec附录A.1记录旧完整代码块任务约660–1560行、过密块经拆分；那些是plan块行数，不冒充P5代码实测。P5复用既有目录、事务、授权、分页与夹具，因此按下表预留测试占比和跨层接线空间。

| Task | 预计新增/修改手写代码与测试行（非实测） |
|---|---|
| T1 / T2 / T3 / T4 | 150–300 / 250–450 / 600–900 / 700–1100 |
| T5 / T6 / T7 / T8 | 1100–1450 / 800–1300 / 900–1400 / 600–1000 |
| T9 / T10 / T11 / T12 | 800–1200 / 800–1200 / 350–650 / 1100–1450 |
| T13 / T14 / T15 / T16 | 1100–1450 / 600–1000 / 900–1400 / 0–100（T16主要文档另计） |

每Task预计小于约1500手写代码/测试行；生成物和文档另计，复用文件的总行数不当新增差异量。T5–7/T12/T13最重，实施前后以真实diff/测试职责核对；若超约1500或Phase约16任务范围，按设计D2已批准机制重切任务或P5a/P5b并同步spec/plan/状态，不削测试、不编造完整原型SHA/逐Task精确行数，也不把局部切分升级成架构审批。

共用helpers：pgtest模板库/RowWaits、现有issueWorld及其API建模、apitest双向契约/错误码检查；沿已有锁屏障及工作项审计断言；e2e/fixtures/issues.ts、furnish.ts、assert/issue.ts及关联assert/catalog等。只为真正缺的场景补窄helper，不重新搭数据库/认证客户端/迁移框架/断言集合。

**共用收口 G（每 Task 最后一项都执行）：** 先跑该 Task 的定位命令，确认红是目标断言失败，再实现并绿。需要真实 Postgres 的包用现有 pgtest，不手造另外一套服务。按改动范围跑 `make lint-go`、`make test`；契约用 `make gen`，仅 SQL 用 `make gen-go`；契约/web/e2e 改动加 `make lint-web`、`make knip`、`make test-web`；迁移/故事跑 `make e2e`。同一依赖链完成后全量结果可按精确 HEAD 复用，后续未影响它时不无意义重跑；新故障/改动必须验证。独立评审后只 add 本 Task 精确文件，提交中文消息；带生成物的提交后 `make gen-check` 必须无差异。

**共用 API 收口 H：** 当 Task 更新 api/modules/issue.yaml（或 project.yaml）、api/openapi.yaml 新路径转引、handler/UseCases/module/bootstrap；HTTP 定位命令整包，覆盖 TestMain 的错误码双向检查。定义 closed DTO，列表只 Issue 不带 description_html/link_count；缺分支省略，[] 与 null 游标明确。矩阵项当场加入 `B/permission_matrix_issue_lists_test.go`（T9建），目标/种子/完整性同步，不等 T14 补授权。生成的 TS 必填 preferences 字段导致 `web/apps/web/core/store/project/fake-projects.ts` 编译变化时同 Task 补默认。

### 跨 Task 类型与签名

以下为设计签名，不是实现全文。uuid 均为本仓库标准库 `"uuid"` 的 uuid.UUID，ctx 为 context.Context。已有 StateRef、LabelRef、VisibleProject、IssueReader、PublicProfiles 复用，不另造同义模型。

```go
// T2–3 issue/domain；不导入HTTP、数据库或其他模块。
type ListScope uint8 // ProjectList, MemberList
type ListOrder string
type ListGroup string
type IDFilter struct { IDs []uuid.UUID; None bool }
type DateFilter struct { None bool; From, To *shared.Date }
type ListFilters struct {
    StateIDs, CreatorIDs, MentionIDs, SubscriberIDs, ProjectIDs []uuid.UUID
    StateGroups []StateGroup; Priorities []Priority
    Labels, Assignees IDFilter
    Start, Target []DateFilter
    SubIssues bool; Archived bool
}
type ListQuery struct {
    Filters ListFilters; Order ListOrder
    GroupBy, SubGroupBy ListGroup; GroupKey, SubGroupKey *string
    Relation string; Limit int; Cursor *ListCursor
}
type ListCursor struct { Order ListOrder; Key json.RawMessage; ID uuid.UUID }
func ParseListQuery(raw map[string][]string, scope ListScope, actorID uuid.UUID) (ListQuery, error)
func NameOrderKey(name string) string
func EncodeListCursor(c ListCursor) (string, error)
func DecodeListCursor(value string, order ListOrder) (ListCursor, error)
// Key保留类型；Decode按order检查并交shared.DecodeCursor canonical检验。

// T4 issue/app；VisibleProjects同时保留历史方法。
type VisibleProjects interface {
    Visible(context.Context, uuid.UUID, uuid.UUID) ([]VisibleProject, error)
    Historical(context.Context, uuid.UUID, uuid.UUID) ([]VisibleProject, error)
}
type ListStateCatalog interface { ListStates(context.Context, []uuid.UUID) ([]StateRef, error) }
type ListLabelCatalog interface { ListLabels(context.Context, []uuid.UUID) ([]LabelRef, error) }
type AssigneeDirectory interface { AssigneeIDs(context.Context, []uuid.UUID) ([]uuid.UUID, error) }
type ListState struct { ID uuid.UUID; Group domain.StateGroup; Key string }
type ListName struct { ID uuid.UUID; Key string }
type ListDirectories struct { Projects []VisibleProject; States []ListState; Labels, Assignees []ListName }

// T5–7 issue/app；三SQL返回同一形状，字段按阶段使用。
type ListingInput struct {
    ActorID uuid.UUID; TargetMemberID *uuid.UUID
    Projects []VisibleProject; Query domain.ListQuery
}
type ListRead struct {
    ActorID uuid.UUID; TargetMemberID *uuid.UUID
    Query domain.ListQuery; Directories ListDirectories
}
type ListRecord struct {
    Issue domain.Issue; Position domain.ListCursor
    GroupKey, SubGroupKey string
    Total, GroupTotal, SubgroupTotal, CellTotal int64
}
type ListRows interface {
    Flat(context.Context, ListRead) ([]ListRecord, error)
    Groups(context.Context, ListRead) ([]ListRecord, error)
    Subgroups(context.Context, ListRead) ([]ListRecord, error)
}
type ListTotal struct { Key string; Total int64 }
type IssueListGroup struct {
    Key string; Total int64; Data []domain.Issue; NextCursor string
    Subgroups []IssueListGroup
}
type IssueListPage struct {
    Total int64; Data []domain.Issue; NextCursor string
    Groups []IssueListGroup; SubgroupTotals []ListTotal
}
// mapper按Query选分支，不用omitempty吞掉应出现的空data；内部空cursor映射null。
type ListingDeps struct {
    Rows ListRows; States ListStateCatalog; Labels ListLabelCatalog
    Assignees AssigneeDirectory; Profiles PublicProfiles; Issues IssueReader
}
func NewListing(d ListingDeps) *Listing
func (l *Listing) Read(ctx context.Context, input ListingInput) (IssueListPage, error)
// Read消费调用者已打开的读快照；取目录→SQL→裁剪/映射→空页补total，不自己开嵌套事务。
// 项目: TargetMemberID=nil且Query.Relation为空；个人: 授权/成员存在检查后传路径目标。
// SQL relation_user_id来自TargetMemberID；ActorID仍用于OwnOnly/可见性，不能互换。
// Listing与仓储校验Relation/target配对，内部不一致返回错误，禁止退回actor或query覆盖。

// T9–10 issue/app；new构造参数以Deps struct收口。
type ListIssuesDeps struct { ReadTx shared.TxManager; Locks Locks; Listing *Listing }
func NewListIssues(d ListIssuesDeps) *ListIssues
func (u *ListIssues) Execute(context.Context, uuid.UUID, map[string][]string) (IssueListPage, error)
type WorkspaceMemberDirectory interface {
    MemberExists(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}
type ListMemberIssuesDeps struct {
    ReadTx shared.TxManager; Authorizer shared.Authorizer; Workspaces WorkspaceDirectory
    Members WorkspaceMemberDirectory; Projects VisibleProjects; Listing *Listing
}
func NewListMemberIssues(d ListMemberIssuesDeps) *ListMemberIssues
func (u *ListMemberIssues) Execute(context.Context, string, uuid.UUID, map[string][]string) (IssueListPage, error)
```

`shared.Authorizer` 复用现有 shared/authorize.go 的 Authorize(ctx,Actor,Action,Target)(Grant,error)，不引入另一份授权接口。ListScope 常量、规则值和错误定义均由所属 Task 同时创建/消费。ListRows 在 T5 仅引入 Flat 的最小接口，T6/T7 随消费者扩方法，不提前留空实现。IssueListGroup 递归仅内部 Go 值；OpenAPI 使用独立 Group/Subgroup 闭合类型，不递归 schema。

project 侧 T4 新提供 `VisibleIssueProjects(ctx,wid,actor) ([]app.VisibleIssueProject,error)`（新普通目录小值为ID uuid.UUID、Role shared.Role、OwnOnly bool；HistoricalIssueProject与历史方法保持不动）、`IssueListStates(ctx,pids) ([]app.IssueState,error)`、`IssueListLabels(ctx,pids) ([]app.IssueLabel,error)`；ID/Name/Group/Default按现有app.IssueState、ID/Name按现有app.IssueLabel复用，bootstrap 转成 issue 的值。W 新提供 `WorkspaceMemberExists(ctx,wid,uid)(bool,error)`，基于现有 MemberOf 的未删除、不要求 active 语义，不 FOR SHARE。

T13 settings 端口的边界值固定为两模块各自的 app 小值，bootstrap 明确转换，拒绝跨模块 domain import：

```go
// project/app；issue/app定义同形的ListDisplayFilters，由issue根包导出alias。
type ListCalendar struct { Layout string; ShowWeekends bool }
type ListDisplayFilters struct {
    Layout string; GroupBy, SubGroupBy *string; OrderBy string
    SubIssue, ShowEmptyGroups bool; Calendar ListCalendar
}
type IssueListSettings interface {
    Validate(filters map[string][]string, displayFilters *ListDisplayFilters,
             displayProperties map[string]bool) []shared.FieldError
}
// nil代表本次未给；非nil空filters代表清空。内部map不影响API闭合schema。
// project/domain仍使用闭合Filters/DisplayFilters/DisplayProperties值和Patch指针，
// app把已通过结构校验的值转成上述端口小值，issue纯适配器复用domain词汇。
```

### HTTP query 接缝：标准扩展点的局部适配

保留 M2 既定绑定在认证/限流之前的顺序，静态预检已核对方案，来源证据登记 [P5 review](../reviews/P5-issue-list-review.md)。Register 的 StdHTTPServerOptions.ErrorHandlerFunc 用errors.As检查gen.InvalidParamFormatError；仅请求为P5两条GET、ParamName在该route已声明非日期query集合、真实Query该key长度>1时写422/not_allowed，其他完全fallback api.Errors.BadRequest。不靠错误文本，不用未实际产生的TooManyValuesForParamError。

新 `I/adapter/http/list_query.go` 提供 `listQueryBindingError(api *httpserver.API) func(http.ResponseWriter,*http.Request,error)` 与 `withListQuery(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc`；后者只对ListIssues/ListMemberIssues深拷贝真实query到私有context，调用next时传派生ctx，handler以 `rawListQuery(ctx) map[string][]string` 消费。注册仍原BaseRouter、原Middlewares；不改request/URL、不以占位绕绑定、不改平台/gen/守卫。纯map不带HTTP对象穿过应用层。

参数schema精确按spec§2.3：9个CSV筛选array/items string、form/explode=false；两日期array/items string、explode=true；sub_issue/archived为string单值而非boolean，由domain校true/false；group/order/relation为枚举string单值（relation required）；group/subgroup key与cursor为string；limit为integer。保留原设计的数组，不能为绕绑定改scalar。未认证+重复在绑定阶段422；坏path先400；其他API重复仍400；绑定成功但无token仍401。Task9专测该矩阵及原query未经改写，不建立新的认证优先规则。

## Task 1：固定池运行参数

文件：改 `server/internal/platform/postgres/pool.go`、`pool_test.go`；保留 UTC AfterConnect。

- [x] 写 `TestPoolForcesListRuntimeParams`：URL明确传 jit=on/plan_cache_mode=auto；同时借出两连接，SELECT current_setting 读到 off/force_custom_plan；另断言UTC timestamptz扫描仍成立。
- [x] 红：`go -C server test ./internal/platform/postgres -run 'TestPoolForcesListRuntimeParams' -count=1`，必须因真实连接值错误失败。
- [x] ParseConfig 后赋 RuntimeParams 两项，不靠模块SQL SET，不破坏其他参数或AfterConnect。
- [x] 绿同命令；实际变异删其中一项/让URL覆盖，测试必须红；恢复重跑。
- [x] G；提交 `feat(M4/P5): 固定列表连接的 JIT 与计划缓存参数`。

## Task 2：列表词汇与页面能力

文件：新 `I/domain/listvocab.go`、`listvocab_test.go`、`list_date.go`、`list_date_test.go`。词汇、页面能力和单一日期原子解析供T3/T13消费；ListScope/ListOrder/ListGroup与DateFilter在T2定义，T3直接复用。

- [x] 写 `TestListVocabulary` 表：19排序码全接受；-sort_order/±link_count拒绝；项目7分组、个人5分组；个人拒state_id/assignee_id/creator/archived/subgroup；UUID/me/none权限按字段区分。
- [x] 红：`go -C server test ./internal/modules/issue/domain -run 'TestListVocabulary' -count=1`。
- [x] 纯值与枚举实现；日期复用shared.Date；不把动态词汇搬shared，不引入HTTP或SQL依赖。
- [x] 绿；变异放行link_count或个人assignee分组，具体表行失败；恢复。
- [x] G；提交 `feat(M4/P5): 定义列表与显示设置共用词汇`。

## Task 3：查询、全序与 canonical 游标

文件：新 `I/domain/list_query.go/list_order.go/list_cursor.go` 及对应测试、`api/testdata/issue-order.json`；shared/page.go/cursor.go只消费。

- [x] 写 `TestParseListQuery`：CSV OR/字段AND、UUID+none、me、重复日期闭区间、none互斥、合法空交集、非法/反向/空token、重复每个非日期参数、未知参数、个人专用边界、group_key/sub_group_key所有组合、limit省略/0/101。
- [x] 写 `TestIssueOrderSamples`、`TestListCursor`：19码roundtrip与错误order/类型/版本/base64/canonical/UUID；Go读取共享JSON，验证生产名字键归一化、键类型/可空性/规范游标，保留日期/null/逆ID、非ASCII大小写及U+FF5E<U+1F600的独立固定预期供T5真实SQL消费。时间固定六位小数UTC，拒绝亚微秒；count保持int64精度；sort有限float64且相邻值可辨、正负零规范为正零。不新增只有测试消费的生产全序比较器；null方向、UUID方向与多值min的实际排序证明在T5/T12完成。
- [x] 红：`go -C server test ./internal/modules/issue/domain -run 'Test(ParseListQuery|IssueOrderSamples|ListCursor)' -count=1`。
- [x] 实现ParseListQuery/NameOrderKey/cursor；精确数字不落float64通用解码；共享JSON内容可读，说明P10b消费义务，不写无消费者TS代码。
- [x] 绿；实际变异只保留最后一个重复日期条件、名字改Unicode lower、时间截毫秒、计数先转float64各至少一次，记录真实业务失败与恢复。日期AND→OR、null降序反转、同键id方向固定及多值min→max是尚未存在的SQL行为，移到T5实际查询变异并在T12收口，不在T3提前标完成。
- [x] G；提交 `feat(M4/P5): 解析列表参数并固定全序游标契约`。

## Task 4：普通项目与完整名字目录

文件：改 `I/app/visible_projects.go`、`I/module.go`、`P/app/issue_ports.go`、`P/module.go`、`P/adapter/postgres/issue_ports.go`、`B/ports.go`；新 `P/adapter/postgres/queries/visible_issue_projects.sql`，列表状态/标签批读SQL，`I/adapter/postgres/list_directories.go/queries/list_directories.sql`；相应测试。

- [ ] 写 `TestVisibleIssueProjects`：有效普通成员、guest OwnOnly、管理员兼guest、管理员非项目成员、退出/软删成员、软删/归档项目；同fixture单独断言Historical历史语义仍不变。
- [ ] 写 `TestListDirectories`：范围内完整状态/标签，triage排除但既有States保留；负责人先曾被指派后退出/降级仍取名，软删负责人行/外项目排除；空输入零目录、不nil，端口故障原样传播/ctx一致。
- [ ] 红：`go -C server test ./internal/modules/project/adapter/postgres ./internal/modules/issue/adapter/postgres ./internal/bootstrap -run 'Test(VisibleIssueProjects|ListDirectories|HistoricalIssueProjects|HistoricalProjects)' -count=1`。
- [ ] 实现上表端口与bootstrap转换，sqlc生成；只取所属模块表，数组配对由ID映射构建，不依赖map随机顺序。
- [ ] 绿；实际变异删除is_active/archived条件、负责人目录误取软删除关系、误改Historical删过滤，每类对应反例红；恢复。退出/降guest后仍取姓名由真实AssigneeIDs→PublicProfiles组合验证，T5实际Listing消费者再覆盖。T4尚无可替换成活跃成员名单的生产调用点，不为强造该变异添加跨模块SQL/端口或修改测试来源来冒充生产变异。
- [ ] G；提交 `feat(M4/P5): 提供普通可读项目与列表名字目录`。

## Task 5：不分组静态查询与公共编排

文件：新 `I/app/list_ports.go/listing.go/listing_test.go`、`I/adapter/postgres/list_issues.go/list_issues_test.go/queries/list_issues.sql`，sqlc生成。

- [ ] 写 `TestFlatIssueList`：所有筛选各有命中/邻居，组合筛选、OwnOnly、triage/软删/归档、重复关联不重复issue；全部19排序与limit+1；label/assignee完整数组；子项计数含隐藏/归档邻居。 真实SQL消费T3的issue-order.json固定预期（适用的名字/日期/null/同键ID），返回ID直接比较独立期望，不复用生产比较器重算；超大整数边界另作精确表达式探针，不伪造海量子项。
- [ ] 写 `TestListArrayParameters`：每个可空数组字段分别nil与[]等价；测试直接sqlc传NULL以证明SQLcoalesce防御；另直接验证生产Store实际调用的参数映射函数产出的各sqlc slice均非nil、并列目录长度一致，或用既有测试tracer捕获真实执行边界；不以app层fake Rows证明adapter规整、不新增生产测试注入。SQL直接NULL的coalesce与adapter规整须分别变异证伪。
- [ ] 写 `TestListingEmptyCursorTotal`：超尾cursor、仍有原筛选、total非0；同ctx去cursor且limit1，不能删group选择；无cursor真空不多查；目录故障/二次查询错误原样返回。
- [ ] 红：`go -C server test ./internal/modules/issue/app ./internal/modules/issue/adapter/postgres -run 'Test(FlatIssueList|ListArrayParameters|ListingEmptyCursorTotal)' -count=1`。
- [ ] ListRead把TargetMemberID映射为独立sqlc参数relation_user_id；关系过滤不用ActorID。项目target=nil/Relation为空，个人二者同时有效；内部错配报错，不静默回落。
- [ ] 实现filtered MATERIALIZED/窄排序键/COLLATE C/window total与一次排序/谓词。k_num分支建议显式统一numeric，时间微秒/日期天数/整数计数不隐式转float8；sort_order和cursor使用一致的明确转换，先用相邻可表示值验证转换未合并两个键。完整数组只为输出行；Row内子项计数遵守项目OwnOnly，父ID按既有hideParents规则裁剪；主查询不读projects/states/users。
- [ ] 绿；变异过滤移到LIMIT后、删coalesce、漏nil规整、total计cursor后、名前取max、漏HiddenFrom计数，以及从T3移来的日期AND→OR、null降序反转、同键id方向固定，均须有真实SQL业务失败并恢复。SQL形状测试只辅助，不替代行为。
- [ ] G；提交 `feat(M4/P5): 实现同快照的不分组列表查询`。

## Task 6：分组查询与组续页

文件：新 `I/adapter/postgres/list_issue_groups.go/list_issue_groups_test.go/queries/list_issue_groups.sql`；扩app接口/响应组织。

- [ ] `TestIssueGroups` fixture：A+C标签项、仅C项、无标签项、多个负责人、各priority/state_group；每组至少limit+1。断言总体去重、组total、完整数组、组序和none末尾。
- [ ] `TestIssueGroupContinuation`：label_id=A+group_key=C只含A∩C，target_date=none组续页；limit=1翻完无重无漏，组total不随cursor变，伪造越尾cursor仍补组total。
- [ ] 红：`go -C server test ./internal/modules/issue/app ./internal/modules/issue/adapter/postgres -run 'TestIssueGroup' -count=1`。
- [ ] 静态Groups SQL，组展开层每issue/key一行；row_number partition group，limit+1；group_key额外谓词，保留原filter；只有非空组。
- [ ] 绿；变异把group_key覆盖labelfilter、全局limit代替pergroup、数组裁成当前组、总体统计展开行；恢复。
- [ ] G；提交 `feat(M4/P5): 支持多值分组与保留原筛选的续页`。

## Task 7：子分组与泳道去重总数

文件：新 `I/adapter/postgres/list_issue_subgroups.go/list_issue_subgroups_test.go/queries/list_issue_subgroups.sql`；扩app/mapper内部模型。

- [ ] `TestIssueSubgroups`：一个工作项两标签×两负责人，另有共享一个标签/负责人项和none项；明确写出每组/每泳道/每格/整体期望集合与计数，不能只断言相互相等。
- [ ] `TestIssueCellContinuation`：原filter+group_key+sub_group_key，limit1翻完；某格为空、超尾cursor补该格total；每格游标独立，行数组完整。
- [ ] 红：`go -C server test ./internal/modules/issue/app ./internal/modules/issue/adapter/postgres -run 'TestIssue(Subgroups|CellContinuation)' -count=1`。
- [ ] 三层展开与独立泳道窗口；不用格和/UNION ALL空行/可空结果塞非空字段；每格limit+1，正确响应枝。
- [ ] 绿；变异泳道=格数之和、组total在双展开后算、全局limit；对应双多值fixture必须杀死；恢复。
- [ ] G；提交 `feat(M4/P5): 实现泳道列表与分层去重计数`。

## Task 8：sort_order PATCH 全链

文件：改 `api/modules/issue.yaml`、`I/domain/patch.go/description.go`、`I/app/rows.go/update_issue.go`、`I/adapter/http/issues.go`、`I/adapter/postgres/update.go/queries/update.sql`；各层既有测试与B/issue_updates_test.go、issue_description_skip_test.go、写锁测试。

- [ ] `TestIssueSortOrderPatch`：省略保持、0写入、同值不写、有限值、纯domain非有限拒绝；单独排序更新审计但无新activity，state+sort设置completed_at且只一条state动态；归档/访客/锁后角色变动沿既有规则。
- [ ] `TestSkipActivityRejectsSortOrder`：true与false各带description+sort都422skip_activity/not_allowed，DB时间戳/描述/排序/动态/版本无变化；HTTP null/字符串400。
- [ ] 红：`go -C server test ./internal/modules/issue/domain ./internal/modules/issue/app ./internal/modules/issue/adapter/postgres ./internal/bootstrap -run 'Test(IssueSortOrderPatch|SkipActivityRejectsSortOrder)' -count=1`；HTTP整包红也要定位到预期新增字段/错误断言。
- [ ] 补IssuePatch/Changes/None、I/app/rows.go中现有IssueFields、合并/UpdateSQL/HTTP与相应映射测试；不把IssueFields复制到ports.go；不误改创建排序和P3子项列表created_at/id，不扩activity枚举。
- [ ] 绿；变异漏Changes.None、零值当省略、漏skip=false组合拒绝、只改state未写sort；恢复。已有PATCH的锁/闸门/归档回归保留。
- [ ] 新 `TestIssueSortOrderInterleaving` 覆盖排序PATCH与归档、成员退出/降级、另一排序/状态写的锁前后两种顺序；使用既有锁屏障，断言锁下重判、拒绝无副作用、成功审计与最终状态一致。强制 `go -C server test ./internal/bootstrap -run 'TestIssueSortOrderInterleaving' -race -count=5`；同步复跑该写复用的现有锁路径交错，以实际测试名列入review命令，不以“复用”免除验证。
- [ ] H/G；提交 `feat(M4/P5): 支持工作项排序更新并保留描述互斥规则`。

## Task 9：项目列表 API 与 HTTP 边界

前置：正式spec/plan及选定query局部适配已独立预检；不得靠占位绕过绑定。文件：新 `I/app/list_issues.go/list_issues_test.go`、`I/adapter/http/list_issues.go/list_issues_test.go/list_query.go`，改handler/module/bootstrap/API，新B/permission_matrix_issue_lists_test.go及issue_lists_test.go。

- [ ] `TestListIssues`：活列表issue.read与归档列表issue.archive分流；已归档项目仍可读；显式项目grant只包含该项目；actor→Parse→ReadTx内授权→Listing，无ctx逃逸；端口失败不吞。
- [ ] `TestIssueListQueryBoundary`：每个非日期参数（含CSV数组）重复422，日期重复AND，未知query422，错误UUID/date/bool语义422，limit非整数400/越界422，cursor坏400；缺token+重复仍422、坏path+重复先400、绑定成功无token401、正常认证后非法词值422；旧comments重复仍400；原始request.URL.RawQuery不变。
- [ ] `TestIssueListEnvelope`：三形状精确字段集合、空data[]、cursor=null、组格续页平面、无description_html/link_count；BodyShapes/路由登记与客户端契约一致。
- [ ] 红：`go -C server test ./internal/modules/issue/app -run 'TestListIssues' -count=1`；`go -C server test ./internal/modules/issue/adapter/http -count=1`；`go -C server test ./internal/bootstrap -run 'Test(IssueListQueryBoundary|IssueListEnvelope|IssueListPermissions)' -count=1`。
- [ ] 按标准ErrorHandlerFunc局部映射和StrictMiddlewareFunc保存原query，不重复认证/限流；GET契约按spec参数表保留CSV string[]/重复日期string[]/布尔词string/limit integer；组结构不用递归schema/oneOf。ReadTx在第一个授权/目录读之前开始。
- [ ] 绿；变异把归档当普通权限、去ReadTx、把重复压扁、错误映射扩到旧comments或坏path；边界测试应各红；恢复。
- [ ] H/G；提交 `feat(M4/P5): 发布项目工作项列表与分组接口`。

## Task 10：个人主页列表与目标成员目录

文件：新 `I/app/list_member_issues.go`/测试与HTTP对应文件；改 `I/domain/actions.go`、`access/domain/rules.go`及规则测试、W/module.go/目录适配/queries/members.sql、B/ports.go/app.go、API/矩阵。

- [ ] `TestListMemberIssues`：assigned/created/subscribed集合互异；目标当前/已结束未删成员可读，未加入/软删404member_not_found；调用者guest403；目标权限不能代替caller权限。
- [ ] `TestMemberIssueRelationTarget` 用调用者A≠目标B，经真实SQL分别验证assigned/created/subscribed；为A、B、第三人布置互不相同关系，断言三次都筛B。另设A为OwnOnly项目guest而B为普通成员：只返回A创建且与B满足关系的项，不能按B权限放行。项目列表target=nil，query注入target字段被拒绝；Listing/仓储错配Relation/target返回错误。
- [ ] `TestMemberIssueVisibility`：caller不同项目角色，管理员兼guest OwnOnly、管理员未入项目、项目归档/软删、退出项目；scope筛选project_id只能求交；workspace不存在/无权先按现有隐藏规则。
- [ ] 红：`go -C server test ./internal/modules/issue/app ./internal/modules/workspace/adapter/postgres ./internal/bootstrap -run 'Test(ListMemberIssues|MemberIssueRelationTarget|MemberIssueVisibility|WorkspaceMemberExists)' -count=1`；access域和issue HTTP整包。
- [ ] 无锁WorkspaceMemberExists，保留inactive；member_issue.list规则复用既有workspace授权模型；同ReadTx获取workspace→授权→目标存在→普通目录；构造ListingInput，ActorID为caller、TargetMemberID为已验证路径user_id，再调用Listing.Read。不得从query覆盖目标。零可读项目正常空页，不跨模块JOIN。
- [ ] 绿；变异relation_user_id误用ActorID、目标要求active、workspace-admin拿全项目、按targetID做OwnOnly、忽略caller项目退出；恢复。
- [ ] H/G；提交 `feat(M4/P5): 支持个人主页工作项的三类关系列表`。

## Task 11：P3 关联列表改用普通可见目录

文件：改 `I/app/list_relations.go/list_relations_test.go/seen.go`、`I/module.go`、B/ports.go；清理确认无消费者的ArchivedProjects端口/查询/测试；扩B/issue_relation_lists_test.go。

- [ ] `TestRelationListUsesVisibleProjects`：跨项目可见/退出/归档/软删/OwnOnly、源项目归档但同项目目标可见、目标issue归档排除；顺序仍沿relation，父ID与subcount不泄漏，目录/seen错误传播。
- [ ] 红：`go -C server test ./internal/modules/issue/app ./internal/bootstrap -run 'Test(RelationListUsesVisibleProjects|IssueRelationLists|HistoricalProjects)' -count=1`，核对已有函数名并整包补旧集合回归。
- [ ] 源授权与读取在ReadTx；普通目录映射grants，加入明确已授权源项目grant；替换seeEach/Archived链，保留seen语义。rg确认无消费者后同步删旧port/provider/fake，Historical完全保留。
- [ ] 绿；变异不补源grant、跨项目目标不用OwnOnly、误用Historical；恢复。矩阵原I9列表行为不变。
- [ ] G；提交 `refactor(M4/P5): 用普通可读目录统一关联列表过滤`。

## Task 12：完整分页、快照与最终 SQL 性能

文件：新 `B/issue_list_pages_test.go/issue_list_snapshot_test.go/issue_list_scale_test.go`；必要窄测试夹具；正式review记录性能表及可复跑命令/seed。

- [ ] `TestIssueListAllOrders`：19码各遍历全部页，ID逆于时间、同键、null、Unicode、多值min、归档/软删/OwnOnly子项计数；加入时间微秒精度边界、相邻可表示sort_order与同键逆UUID，检查SQL键与cursor键无舍入退化；100k计数不能证明超过2^53的整数边界，该边界另用真实SQL数值表达式探针或明确精确类型证明，和领域游标测试分别记证据；对照集合和独立期望序，不复用生产比较器计算期望。
- [ ] `TestIssueListAllGroupingFields` 以真实SQL覆盖项目7种group（state_id/state_group/priority/label_id/assignee_id/created_by_id/target_date）、个人5种group（state_group/priority/label_id/target_date/project_id）；项目全部42种合法group×subgroup组合均验证返回集合、整体/组/泳道/格total与limit=1续页。每种可为空字段含none，特别单列created_by_id、project_id、target_date=none；筛选与分组重叠、空cursor页补total、行完整数组均有反例。期望键和集合独立列出，不调用生产分组键/比较器反算；不能仅在词汇测试中接受枚举就标覆盖。created_by_id/project_id/target_date各至少一次把SQL分支替成错误字段的实际变异，必须被对应断言杀死并恢复。
- [ ] `TestIssueListSnapshot`：目录读后用可控channel/RowWaits暂停，另一连接新增state并移issue或退出成员/更名；完成本页应全旧快照，下一次请求全新；空cursor补total也走旧快照。全程读tx可观测READ ONLY/RR，不靠sleep。
- [ ] 红/绿定位：`go -C server test ./internal/bootstrap -run 'TestIssueList(AllOrders|AllGroupingFields|Snapshot)' -count=1`；去ReadTx/传外ctx变异必须失败，恢复后重跑；完成门强制 `go -C server test ./internal/bootstrap -run 'TestIssueListSnapshot' -race -count=5`，不能以一次运行或普通-race替代。
- [ ] 实现可选显式运行的 `TestIssueListScale`（环境 `NERVE_LIST_SCALE=1`，默认skip记录清楚，不把skip当性能通过）：真实迁移/运行角色，10k和100k种子及数据库核对，最终生产SQL非近似副本；计时外比对正确性。测试pgx tracer捕获真实Store执行的sqlc SQL和参数，在测试连接对同一SQL/参数取EXPLAIN（计时外），不复制近似查询、不新增生产rawSQL出口。
- [ ] 独占执行：`NERVE_LIST_SCALE=1 go -C server test ./internal/bootstrap -run '^TestIssueListScale$' -count=1 -timeout=20m -v`。默认页/label_name/state组/label组/泳道各至少预热1次、记录5次实测及min/median/max，代表性筛选和跨项目可见目录另测；记录目录耗时与总请求耗时，PG/硬件/参数/分布/EXPLAIN BUFFERS/实际索引。
- [ ] 与spec基准逐项比较，显著超界先EXPLAIN局部优化再复测；不能只保留最快值或降低100k完成线。原S1程序未保留，当前结果不能冒称原S1复现。大规模/默认索引特化归M8。
- [ ] G；提交 `test(M4/P5): 验证列表全序快照并记录规模成本`。

## Task 13：三类项目显示设置与纯词汇端口

文件：新 `server/migrations/sql/00026_project_project_user_properties_issue_display.sql`；改 `server/migrations/project_schema_test.go`、P/domain/preferences.go与测试、P/app/update_preferences.go/get_preferences.go/ports、P/adapter/http/preferences.go/测试、P/adapter/postgres/preferences.go/queries/preferences.sql/测试；新I/app/list_settings.go/测试；改双方module、B/ports.go/app.go、api/modules/project.yaml、相关TS fixture。

- [ ] `TestIssueDisplayMigration`：老行获得精确默认；三列NOTNULL；逐列对象/数组/标量/JSONnull，未知/缺键/错误嵌套类型/string[]中非string均check_violation；合法形状非法词值DB允许、应用拒绝；Up→Down→Up且旧navigation/sort保持。
- [ ] `TestIssueListSettings`：same词汇合法/非法对应查询；filters.label_id[0]、日期none混合、filters空数组、单个元素含逗号拒绝；group/subgroup组合、布局/日历/19排序与未来词拒绝；纯adapter无pool。
- [ ] `TestProjectIssuePreferences`：GET缺行默认且不写、PATCH每个字段整体替换/省略保持/{}清空、其他用户不变、归档规则保持；null/未知body400、非法词值422路径准确；成员离开再恢复保留偏好。
- [ ] `TestPreferenceFieldsDoNotOverwrite`：navigation与filters、display_filters与display_properties两个连接交错PATCH，最终两边修改都在；读整行全覆盖的变异要实际杀死。
- [ ] 红：`go -C server test ./migrations ./internal/modules/project/domain ./internal/modules/project/app ./internal/modules/project/adapter/postgres ./internal/modules/issue/app ./internal/bootstrap -run 'Test(IssueDisplayMigration|IssueListSettings|ProjectIssuePreferences|PreferenceFieldsDoNotOverwrite)' -count=1`；project HTTP整包。
- [ ] CHECK用CASE object保护、形状only，filters jsonb_path_exists；domain闭合结构及defaults；IssueListSettings app小值/纯issue提供/bootstrap转换；Upsert每列set标志，INSERT依赖默认保持已有ensure语义；生成closedAPI，不加rich_filters。
- [ ] 绿；变异漏set标志/去CASE/只校形状不校词/误用未来group/改成员恢复覆盖行；恢复。检查现有grants仍覆盖原表，无新表/权限豁免/删除步骤；如显式列权限需要更新，按实际grant验证决定。
- [ ] 偏好并发与新增恢复/快照交错全部强制 `go -C server test ./internal/bootstrap -run 'TestPreferenceFieldsDoNotOverwrite' -race -count=5`；若新增交错在其他测试包，逐包列实际名字同样执行-race -count=5。恢复所有变异后重跑，记录无数据竞争/无死锁/无丢更新。
- [ ] H/G（含make e2e验证迁移与旧偏好故事）；提交 `feat(M4/P5): 保存项目工作项筛选与显示设置`。

## Task 14：矩阵、整程序边界与回归闭环

文件：补 `B/permission_matrix_issue_lists_test.go`、matrix seed/targets/columns、runtime_role_test.go、contract_test.go、issue_list_permissions_test.go、project_preferences_test.go；必要TS fixture与端口假实现；不堆新helper绕过已有断言。

- [ ] 审核listIssues两权限分支、listMemberIssues的所有目标、preferences三字段进入既有矩阵；角色按集合：workspace admin兼project guest、无project成员、deleted/inactive、公开/私有、隐藏先404。
- [ ] 新 `TestP5ReadBoundary` 证实读无FOR锁/写连接/写闸门，源/目录/SQL/补total一个ctx；sort_order仍原写锁/闸门路径。复跑历史activities、description skip、relations、subissues与原preferences，确保没有范围倒退。
- [ ] 定位 `go -C server test ./internal/bootstrap -run 'Test(PermissionMatrix|P5ReadBoundary|IssueListPermissions|ProjectIssuePreferences)' -count=1`；再跑 `go -C server test ./internal/archtest ./internal/bootstrap -count=1`，HTTP issue/project整包覆盖所有码。
- [ ] 对前面漏测处先红再修，不写镜像实现测试；至少额外实际变异去member_issue.list角色限制、readonly目录ctx外移、settings按用户更新范围丢actor条件；恢复后绿。
- [ ] 强制完成门：`go -C server test ./internal/bootstrap -run 'Test(IssueListSnapshot|PreferenceFieldsDoNotOverwrite|IssueSortOrderInterleaving)' -race -count=5`；核对所有新增快照/偏好并发/sort_order相关交错均被匹配，其余包或名称追加对应同参数命令，不允许“需要时”。sort_order复用锁路径的相应既有交错同样五遍race，review列出完整匹配清单。
- [ ] `make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web` 及已提交生成物的 `make gen-check`，记录真实命令/HEAD/结果，失败不包装为通过。
- [ ] G；提交 `test(M4/P5): 补齐列表权限矩阵与跨层回归`。

## Task 15：接口故事与既有归档故事增长

文件：新 `e2e/stories/issue/i3-list-groups.spec.ts`、`i4-list-order-calendar.spec.ts`、`i5-list-preferences.spec.ts`、`i6-board-order.spec.ts`、`i7-quick-add.spec.ts`、`i19-member-issues.spec.ts`、`i20-guest-boundary.spec.ts`；改i15-archive-issue.spec.ts、e2e/fixtures/issues.ts与assert/issue.ts。每故事一个文件，尚无页面版本。

- [ ] I3按state组/priority×label泳道、总数/完整数组/加载更多A∩C；I4每排序码翻页、日期范围及none；I5三类设置按人隔离、同参数GET等价；I6 state+sort同时写、completed_at/单条state动态；I7复用I1创建并验证状态与末尾排序，无新API。
- [ ] I19三关系/跨项目caller可见性/guest403；I20四格（别人的详情404、创建403、列表own、自己项评论201），访客项由member建后API降级；I15新增仅归档列表及恢复后的消失/活列表出现。
- [ ] 数据全部经生成客户端API，断言复用assert helpers；数据库仅验证结果。需要第二页至少limit+1不同值，不能只断言200/数组长度；所有负例检查错误码/字段。
- [ ] 先以当前缺故事断言运行并记录红：`make build` 后 `pnpm --dir e2e exec playwright test --project=chromium 'issue/i(3-|4-|5-|6-|7-|15-|19-|20-)'`；实际runner参数若按path regex须核对匹配8文件；不匹配不能当绿。实现fixtures/断言，再绿同命令。
- [ ] 至少变异删组续页filter、I6排序赋值、preferences用户隔离、I20 own过滤对应一个故事红；恢复重跑。异步栏为空，不固定等待；旧评论功能仅作为P5故事依赖。
- [ ] G并完整 `make e2e`；提交 `test(M4/P5): 验证列表显示设置与个人主页接口故事`。

## Task 16：同步设计、交接与最终证据

文件：正式P5 spec/plan/review；改 `docs/v0/v0-design.md` §3.3/3.4/6.2、`docs/v0/M3-workspace-project/M3-design.md` §3.18、`docs/v0/plane-diff.md`、M4-design §3.23/P5与§5.1子项排序残留；M2/M3/P1–P4交接对应行。

- [ ] 按spec每节→Task→测试/变异/实测记录逐行核对；缺scope不以“后续可做”消失。三列逐列、列表响应/命名/none/重复日期/total/RR/游标/目录词汇全同步。
- [ ] spec性能部分补最终生产SQL实际索引/EXPLAIN与成本，保留20k目标/100k验证、M8跨项目扫描与名字变更跨页漂移；不新增未经批准索引路线。
- [ ] P10b交接共用issue-order.json、真实前端码点比较器/vitest与旧store迁移；P8 link_count/搜索复用词汇；P12–15页面故事；M4/交接开放状态不提前关闭。
- [ ] 自检本计划16任务边界与签名，核对真实基线与最终签名，删除不适用候选，独立审查整分支而非只看diff；按严重度处理局部缺陷，架构/高风险上报。最终review记录命令/HEAD/环境、测试红绿、变异恢复、性能、未解决风险，不用虚构百分比。
- [ ] 按最终变更范围跑所需G、提交后make gen-check；提交 `docs(M4/P5): 同步列表契约性能证据与后续交接`。先交负责人安排整分支终审并处理意见，再按既有流程分支CI→--no-ff合并→main CI→worktree清理；不提前标完成。

## 范围核对与当前证据边界

范围对照：§3.16→T1–7/T9–12；sort_order/I6→T8/T15；普通Visible/Historical/P3→T4/T11；§3.17/4.15→T2/T13；矩阵→各API Task及T14；全部P5故事→T15；§3.23→T16。ReadOnlySnapshot/PageSize直接消费；HTTP接缝已有静态预检结论；起步版本纳入四项Local/Medium及两项Low修订；后续运行验证与实施评审逐任务登记，未完成的任务不能提前声称通过。

局部选择和HTTP标准扩展点方案已记录，未改变架构。基线与CI已核实；起步须处理预检意见并登记复审结论。P5 TDD、变异、全部新增交错race五遍、精度/性能、门禁与终审证据待实施，不能把静态核对当产品验证。
