# M4/P8 草稿、搜索与链接计数：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P8 `drafts-search` |
| 日期 | 2026-10-27 |
| 状态 | **已完成集成**：最终分支 `cde70d9735994cb6a579e0b800469f32631e9981` 经授权 `--no-ff` 合并为 main `6b9d08d4821f1423c7bb9795557727c206b7880f`；[main CI 40520609987](https://github.com/open-nerve/NerveProject/actions/runs/40520609987) 三任务全绿，归档核验与分支/worktree清理完成，见[评审记录 §9](../reviews/P8-drafts-search-review.md#9-授权合并main-ci-与归档清理2026-10-27) |
| 代码基线 | P7 的 main 合并 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`，文件树 `51982bad72186f17a0b10c31f2f508629bb9cca7`；主控已核同 SHA 的 main CI `40451632280` 全绿；归档/清理证据见 P7 收尾记录 |
| 实施期分支（历史） | `codex/m4-p8`；worktree `.codex/worktress/codex/m4-p8`；集成与归档核验后已清理，主目录保持 main |
| 上级文档 | [M4 设计](../M4-design.md) §3.3–3.5、3.15–3.18、3.23、4.13、5、6、8.6–8.7、9、12/P8、13.1；[方法](../../claude_to_gpt/02-method.md)、[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md) |
| 计划 | [P8 plan](../plans/P8-drafts-search.md) |

本 spec 固定批准设计留给 P8 的接口、参数边界、文件职责和验证方法。设计的 13 个粗任务拆为 16 个可独立评审的 Task，沿 P7 紧凑方法，不重做已批准架构，也不把静态预检记成原型或逐任务复现。以下新增符号均为实施期目标，未声称已存在。

以上及目标中的“阶段完成先不合并”为批准时口径；同日实施已交付下列接口，产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6`。附录保留准备/T8/T15当时边界；§4的A1–A12已按实际证据验收，实施证据见[review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)，最终分支CI与获授权后的集成完成状态见[review §9](../reviews/P8-drafts-search-review.md#9-授权合并main-ci-与归档清理2026-10-27)。

## 1. 目标

1. 六个个人草稿操作：按工作区列/创建，按草稿读/改/删/发布；只属于创建者，保存不执行工作项语义，发布读取存储载荷，经原创建结构检查和唯一创建核心，在同一个事务创建工作项并软删除草稿。
2. 草稿遵守 W(S)→P(S)→draft(U)；发布无数据库资源时先入项目写闸门，再 W(S)→P(N)→draft(U)。旧、新项目升序取锁；等待后项目变化答 409，不增补另一条锁路。
3. 搜索在独立只读快照中按项目可见范围、OwnOnly、日期和选择器排除取前 N 个完整 `Issue`；空词最近更新，非空词精确编号/数字/标题字面包含分级排序。只有 `data`，无分页、总数和 `page`。
4. 迁移建立 `pg_trgm` 与标题部分 GIN 索引，验证 PostgreSQL 18.6 非超级用户迁移角色、runtime 权限、sqlc 数值类型，并在 10 万英文/中文数据上记录实际耗时和计划。
5. `link_count` 进入每一种 `Issue`/`IssueDetail` 表示，以及平铺、分组、子分组、成员列表和偏好的两个排序码；保留 P5 已交付的整数/数值游标和授权性质。
6. 交付 I14、I18 接口故事，扩 I4 的两方向跨页；完成矩阵、交错 16/17、真实生产变异、整分支 deep-review 与完整本地七门禁。阶段完成先不合并。

## 2. 交付物

### 2.1 文件职责和粗任务映射

路径均相对 worktree。I=`server/internal/modules/issue`，P=`server/internal/modules/project`，B=`server/internal/bootstrap`。生成目录只经生成器修改；表列测试名为计划新增者时用“新”标记。

| 精确文件或同职责文件组 | 责任 | Task；设计粗任务 |
|---|---|---|
| `server/migrations/sql/00028_issue_draft_issues.sql`；`server/sqlc.yaml`；`deploy/runtime-grants.sql`；I/app/cascade.go、delete_project_rows.go；I/adapter/postgres/queries/delete_rows.sql、delete_rows.go；I/module.go；P/app/issue_ports.go、cascade.go；对应既有 cascade/deletion tests | 草稿 schema、授权与项目/工作区同步级联，workspaceID 贯穿 | T1；1 |
| I/domain/draft.go、draft_cursor.go 及测试（新）；I/app/draft_ports.go（新）；I/adapter/postgres/queries/drafts.sql、drafts.go、drafts_test.go（新） | 纯草稿值、保存边界、游标和单表存储 | T2；2 |
| I/app/lock_draft.go、draft_reads.go、draft_writes.go 及测试（新）；I/domain/actions.go；access/domain/rules.go；P/adapter/postgres/queries/issue_ports.sql、issue_ports.go；P/module.go；B/ports.go | 草稿唯一锁编排、读快照、CRUD、小目录端口 | T3；2–3 |
| `api/modules/issue.yaml`、`api/openapi.yaml`；I/adapter/http/drafts.go、draft_query.go 及测试（新）；I/adapter/http/handler.go、I/module.go；B/permission_matrix_drafts_test.go、issue_drafts_test.go（新）；双语 errors 与 PROBLEM_MESSAGES；`e2e/fixtures/furnish.ts`、`assert/project.ts`、`assert/workspace.ts` | 五个 CRUD 操作、双语码、权限矩阵、完整删除目录的 API 前置 | T4；1–3、10 |
| I/app/create_core.go（新）、create_issue.go、module.go构造及既有create/lock/event测试；I/adapter/http/create_payload.go、create_payload_test.go（新）、issues.go | 单一已持锁创建核心；app 消费结构解码小端口，HTTP adapter 复用生成 bodyshape/映射 | T5；4 |
| I/app/publish_draft.go、publish_draft_test.go（新）；I/adapter/http/drafts.go；I/module.go；B/issue_draft_publish_test.go（新）；契约/矩阵 | 发布无 body、同事务副作用与拒绝顺序 | T6；4、10 |
| B/issue_draft_interleaving_test.go、issue_draft_lock_shapes_test.go（新）；已有 gate/issue_write_locks tests | 交错 16/17 双向、权限变化、旧新项目锁序/时钟与拒绝零锁 | T7；5 |
| `e2e/stories/issue/i14-drafts.spec.ts`（新）；`e2e/fixtures/issues.ts`；项目/工作区删除故事；B/project_deletion_test.go、workspace_deletion_test.go、workspace_deletion_catalog_test.go | I14、零项目工作区与无项目草稿级联、阶段回归 | T8；1、11 |
| `server/migrations/sql/00029_issue_issues_name_trgm.sql`；`server/sqlc.yaml`；`server/migrations/schema_test.go`、trigram_schema_test.go；B/search_runtime_role_test.go复用runtime_role_test.go角色工具；I/adapter/postgres/search_type_test.go（新） | 扩展/GIN、部署角色与 sqlc float4 | T9；6 |
| I/domain/search.go、search_test.go（新）；共享 `api/testdata/identifier-fold.json` 只消费 | 词汇、重复参数、日期、编号折叠、字面查询和错误边界 | T10；7 |
| I/app/search_ports.go、search_issues.go、search_issues_test.go（新）；I/adapter/postgres/queries/search_issues.sql、search_issues.go、search_issues_test.go（新）；B/issue_search_test.go、issue_search_snapshot_test.go（新） | 同快照授权、静态 SQL 排序、全深度排除和统一 seen | T11；7–8、10 |
| I/adapter/http/search_issues.go、search_issues_test.go（新）；I/adapter/http/handler.go、draft_query.go；I/module.go；契约；B/permission_matrix_search_test.go（新） | 搜索 HTTP 边界和完整矩阵 | T12；8、10 |
| I/domain/issue.go、listvocab.go、list_cursor_key.go；I/adapter/postgres/queries/issues.sql、sub_issues.sql、list_issues.sql、list_issue_groups.sql、list_issue_subgroups.sql、search_issues.sql；对应 Go 映射/游标/测试及list_link_count_test.go共享helper；I/adapter/http/issues.go；契约；`api/testdata/issue-order.json`；B/issue_link_counts_test.go与已有列表/偏好/形状/HTTP list-member测试、受影响I3/I4 fixtures | 所有响应 live link count、19→21 排序码、三 SQL 同形和游标 | T13；9 |
| `e2e/stories/issue/i18-search.spec.ts`（新）、`i4-list-order-calendar.spec.ts`；`e2e/fixtures/issues.ts`、`fixtures/assert/issue.ts` | I18、I4 双向非零计数续页；全表示Go验证已在T13 | T14；11 |
| I/adapter/postgres/search_performance{,_fixture,_evidence}_test.go（新，显式 opt-in 场景/seed/原始取证）；本 spec 附录、ignored 原始证据 | 10 万数据、locale/EXPLAIN/耗时、全门禁与性质审计 | T15；12 |
| 本spec/plan/review、`docs/reviews/re-07-M4P8整体审核.md`；M4-design当前端口/草稿/搜索/§3.23/6.5/11.2/12/15，v0/Plane/README，03/04当前导航；四输入handoff与P1a–P7 spec§5/review§6追加 | 30文档准确范围deep-review/历史D1摘要修正、实际验收与交接；文档a65e8605独评Approved；最终分支CI与集成完成见review§9 | T16；13 |

单 Task 目标约 350–1200 手写差异行，预留局部修复余量；超过约 1500 行先按职责重新切分，不能把安全/锁序测试压缩掉。单个新逻辑文件约 400 行上限；生成物、静态 SQL 和有明确理由的测试表沿既有例外。

### 2.2 草稿数据与纯领域签名

`00028` 建 `draft_issues` 九列：`id uuid PRIMARY KEY`；`workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE`；`project_id uuid NULL REFERENCES projects(id) ON DELETE CASCADE`；`payload jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(payload)='object')`；`created_by_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE`；`updated_by_id uuid NULL REFERENCES users(id) ON DELETE SET NULL`；`created_at/updated_at timestamptz NOT NULL DEFAULT now()`；`deleted_at timestamptz NULL`。活行索引 `(workspace_id,created_by_id,created_at DESC,id DESC) WHERE deleted_at IS NULL`，另外 workspace/project 无条件 FK 索引；不新增冗余 title、sequence 或版本列。

```go
// I/domain/draft.go / draft_cursor.go；json/time/uuid 均为现有标准库依赖。
type Draft struct {
    ID, WorkspaceID, CreatedByID uuid.UUID
    ProjectID, UpdatedByID *uuid.UUID
    Payload json.RawMessage
    CreatedAt, UpdatedAt time.Time
}
type NewDraft struct { ProjectID *uuid.UUID; Payload json.RawMessage }
type DraftPatch struct { SetProject bool; ProjectID *uuid.UUID; SetPayload bool; Payload json.RawMessage }
func CheckDraftPayload(json.RawMessage) error // 只固定3个自由字符串的NUL可存性，不做create语义。
type DraftCursor struct { CreatedAt time.Time; ID uuid.UUID }
func EncodeDraftCursor(DraftCursor) (string,error)
func DecodeDraftCursor(string) (DraftCursor,error)
```

草稿 payload 字段精确为 `name,description_html,state_id,priority,assignee_ids,label_ids,start_date,target_date,parent_id`，全可省略；出现时类型/UUID/date 格式与 IssueCreate 相同，均不可 null。`name:""`、空/重复/超过100项数组、非法优先级、超长业务字符串、无效引用可保存，发布才按创建规则拒绝。禁止 `skip_activity/sort_order/project_id` 塞进 payload。外层创建 `payload` 必需且非 null object；PATCH 中 payload 省略保持，给出则整体替换（`{}` 合法），null400；project 省略保持，null 清空。

`CheckDraftPayload` 仅对 `payload.name`、`payload.description_html`、`payload.priority` 的 NUL 答422 `invalid_format`，原文不 trim、不删字符、不清洗。结构校验由 HTTP bodyshape 负责；不扩展共享 bodyshape 或做通用 JSON 递归框架。此为静态发现的 PostgreSQL JSONB 表示边界，实施期必须真实 API/DB 验证。

草稿游标复用 `shared.EncodeCursor/DecodeCursor`，payload 精确两元素 `[created_at,id]`，复用本模块 collaboration_cursor.go 的 marshal/unmarshal 形状校验（UTC RFC3339Nano 时间、非零 UUID），不新造另一套JSON规范化；固定降序。非法/非规范/错误版本/额外元素400 bad_request/cursor；合法篡改只是排序位置，绝不回查该 id 的 owner。limit 复用 `shared.PageSize(*int)(int,error)`：默认50，1–100，非整数400，越界422。`DraftPage{Data []Draft;NextCursor *string;Total int64}`，空 Data=[]，total 不受 cursor 影响；极末页仍答真实 total。

### 2.3 app 草稿端口、锁与创建核心

```go
// I/app/draft_ports.go：使用者拥有的小接口；不扩全部旧IssueReader假实现。
type WorkspaceSharer interface { ShareWorkspaceByID(context.Context,uuid.UUID)(bool,error) }
type DraftProjects interface {
    UndeletedIDs(context.Context,uuid.UUID)([]uuid.UUID,error)
    ShareMany(context.Context,[]uuid.UUID)(map[uuid.UUID]uuid.UUID,error)
}
type DraftReader interface {
    GetDraft(context.Context,uuid.UUID)(domain.Draft,bool,error)
    LockDraft(context.Context,uuid.UUID)(domain.Draft,bool,error)
}
type DraftListRead struct {
    WorkspaceID, ActorID uuid.UUID
    ProjectIDs []uuid.UUID // 同工作区全部未删项目；不是调用者Visible目录。
    After *domain.DraftCursor
    Limit int // app请求limit+1；total独立同一快照。
}
type DraftLister interface { ListDrafts(context.Context,DraftListRead)([]domain.Draft,int64,error) }
type DraftWrite struct { ID,WorkspaceID,By uuid.UUID; ProjectID *uuid.UUID; Payload json.RawMessage; At time.Time }
type DraftDeletion struct { ID,By uuid.UUID; At time.Time }
type DraftWriter interface {
    InsertDraft(context.Context,DraftWrite)(domain.Draft,error)
    UpdateDraft(context.Context,DraftWrite)(domain.Draft,error)
    DeleteDraft(context.Context,DraftDeletion) error
}
type IssueCreatePayloadDecoder interface { DecodeCreatePayload([]byte)(domain.NewIssue,error) }
```

`GetDraft/LockDraft` 仅返回未删草稿，**不先加 owner 谓词**，让 workspace 访客对活草稿先答403；app 在任何响应前核 owner 和旧项目存在性。`ListDrafts` 在 SQL 内过滤 workspace/owner/deleted/project-ID 集合，并在 LIMIT 前完成全部过滤。Update SQL 仅在 project 或 JSONB payload 不同才写审计，返回真实库中结果；JSONB 对象 key 顺序变化不是修改。对同一workspace多个草稿的读取不因旧项目无成员/归档误过滤。全体读在 `ReadTx`（`ReadOnlySnapshot()`）里完成目录/授权/页和 total。

```go
// I/app/lock_draft.go：私有编排只此一套，组合唯一既有Locks。
func NewDraftLocks(base Locks,w WorkspaceSharer,p DraftProjects,d DraftReader) DraftLocks
func (l DraftLocks) run(ctx context.Context,actor shared.Actor,w draftWrite,fn func(context.Context,draftHeld)error) error
func (l DraftLocks) publish(ctx context.Context,actor shared.Actor,id uuid.UUID,fn func(context.Context,publishHeld)error) error
func (l DraftLocks) seeWorkspace(context.Context,shared.Actor,uuid.UUID) error
func (l DraftLocks) read(context.Context,shared.Actor,uuid.UUID)(domain.Draft,error)
// draftWrite={WorkspaceID uuid.UUID; DraftID *uuid.UUID; SetProject bool; ProjectID *uuid.UUID}。
// draftHeld={Draft *domain.Draft; ProjectID *uuid.UUID}；创建Draft为nil；publishHeld={Draft domain.Draft; Issue held}。

// I/app/create_core.go；构造公开给module，锁内方法只供app。
type IssueCreationDeps struct {
    Events shared.EventPublisher; Issues IssueCreator; Sequences Sequences
    States StateCatalog; Labels LabelCatalog; Members ProjectMembers
    Profiles PublicProfiles; Clock Clock; Sanitizer HTMLSanitizer
}
type IssueCreation struct { d IssueCreationDeps }
func NewIssueCreation(IssueCreationDeps) *IssueCreation
func (c *IssueCreation) createLocked(ctx context.Context,actorID,projectID uuid.UUID,h held,in domain.NewIssue)(domain.Issue,time.Time,error)
```

`DraftLocks` 从 `base` 复用 authorizer/gate/写 tx/错误映射，旧 `NewLocks(ProjectGate,IssueReader,Authorizer,WriteGate,TxManager)` 不变。CRUD/publish 用例不各自再持 authorizer/gate/写 tx；read 用例可持独立 ReadTx。project 在其模块导出 `UndeletedIssueProjectIDs(ctx,wid)([]uuid.UUID,error)`、`ShareIssueProjects(ctx,ids)(map[uuid.UUID]uuid.UUID,error)`；后者一条按 id 升序 `FOR SHARE`，不以 N 代 S。workspace 已有 `ShareWorkspaceByID`；bootstrap 转成 bool 小端口，只在组合层接线。原 `ProjectGate` 继续负责发布 N 锁，不增加 SQL 跨模块 JOIN。

CRUD：无锁读/预检 → 写事务 W SHARE → 无额外锁重读资源存在性（见下一段）→ 工作区重新授权 → 旧/显式新项目去重升序 SHARE → 草稿 UPDATE → 重读 owner/deleted/项目与初读相等 → 检查显式新项目 → 时刻 → 写。create 无旧草稿；nil 项目不是零 UUID 项目锁。删除和省略 project 的 patch 只要求旧项目未删，不重新要求旧项目可见；显式提供 project（包括与旧值相同）须同工作区未删且 现有Authorizer以`shared.Action("project.read")`判定成功（该规则为LevelVisible，不是issue.read的LevelProject；不用并不存在的Grant.Visible字段），允许归档项目，也允许工作区成员可见但未加入的public项目；private不可见项目拒422。

publish：无锁读/预检（含 owner、旧项目存在性、项目非空和 create 权限）→ 无连接/无锁入项目 gate → 事务 W SHARE → 无额外锁重读资源存在性 → draft.write → P NO KEY UPDATE/create → draft UPDATE → 重读（删除404，项目变409）→ 项目归档409 → decoder → `domain.CheckNewIssue` → `IssueCreation.createLocked` → 草稿软删 → commit → 离 gate。核心负责与当前创建完全相同的项目归档/引用/默认负责人/HTML清洗/提及/订阅/号序/排序/活动/描述版本/issue.created/readBack，返回唯一 `At` 给草稿删除；禁止在这里嵌套 `CreateIssue.Execute`、再次入 gate 或打开写 tx。

W SHARE后的资源重读只判断草稿活行及当前关联项目未删（创建只读workspace），不提前比较project变化、判owner或追锁新的项目。若它已见删除，404先于workspace降guest；资源仍活而权限拒绝则403，以这次资源读为判定时点，不为确认后来并发删除而取未授权P/D锁。通过授权后仍按初读项目取锁，P/D等待后再遇删除404、项目变409。CRUD与publish一致；T7覆盖删除+降guest都已提交以及仅降guest两种终态。

module.go只构造一次`NewIssueCreation`，同时给`CreateIssueDeps{Locks Locks;Creation *IssueCreation}`和`PublishDraftDeps{Locks DraftLocks;Decoder IssueCreatePayloadDecoder;Creation *IssueCreation;Drafts DraftWriter}`；既有创建测试构造同步改显式依赖，没有optional setter。普通 CreateIssue 仍先纯领域校验再 `Locks.run`，在 callback 调同核心；publish 校验必须用锁下重读的 payload。decoder 放 I/adapter/http/create_payload.go，精确调用 `gen.BodyShapes().Check("POST /api/v0/projects/{project_id}/issues",raw)` 后解 `gen.IssueCreate`，复用从 issues.go 抽出的 `newIssueInput(gen.IssueCreate) domain.NewIssue`。构造测试核此 root 存在，避免 Check 遇缺根静默放行。app/domain 不 import bodyshape、HTTP 或 gen。

| 用例 | 唯一 Execute 签名；依赖 |
|---|---|
| ListDrafts | `Execute(ctx,slug string,limit *int,cursor *string)(DraftPage,error)`；ReadTx、DraftLocks、WorkspaceDirectory、DraftProjects、DraftLister |
| GetDraft | `Execute(ctx,id uuid.UUID)(domain.Draft,error)`；ReadTx、DraftLocks |
| CreateDraft | `Execute(ctx,slug string,in domain.NewDraft)(domain.Draft,error)`；DraftLocks、WorkspaceDirectory、DraftWriter、Clock |
| UpdateDraft | `Execute(ctx,id uuid.UUID,in domain.DraftPatch)(domain.Draft,error)`；DraftLocks、DraftWriter、Clock |
| DeleteDraft | `Execute(ctx,id uuid.UUID)error`；DraftLocks、DraftWriter、Clock |
| PublishDraft | `Execute(ctx,id uuid.UUID)(domain.Issue,error)`；DraftLocks、IssueCreatePayloadDecoder、*IssueCreation、DraftWriter |

每个用例使用同名 `XDeps`、`NewX(XDeps)*X`；签名中的 ctx 均为 context.Context，Constructor 按该行列出依赖，不引入可选 setter。

### 2.4 草稿公开契约、权限和拒绝顺序

六操作均 `domain.ActionDraftWrite = "issue_draft.write"`，沿工作区 admin/member 允许、guest403、非有效成员404；不新增 draft.read action。他人活草稿：可见工作区的 guest403，admin/member404 `issue.draft_not_found`。旧项目已删除/错工作区的草稿统一不可达，不泄漏 payload/owner/project；所有相关404对未知随机 id 逐值相同。

| operationId / 路由 | 响应 | 模块级错误（公共认证/结构错误沿平台） |
|---|---|---|
| listDrafts `GET /workspaces/{slug}/drafts` | 200 DraftPage | workspace.not_found404、forbidden403、游标400、limit422 |
| createDraft `POST /workspaces/{slug}/drafts` | 201 Draft | workspace.not_found404、forbidden403、validation_failed422、server_busy503 |
| getDraft `GET /drafts/{draft_id}` | 200 Draft | issue.draft_not_found404、forbidden403 |
| updateDraft `PATCH /drafts/{draft_id}` | 200 Draft | issue.draft_not_found404、forbidden403、validation_failed422、issue.draft_changed409、server_busy503 |
| deleteDraft `DELETE /drafts/{draft_id}` | 204 | issue.draft_not_found404、forbidden403、issue.draft_changed409、server_busy503 |
| publishDraft `POST /drafts/{draft_id}/publish`（无 body） | 201 IssueDetail | issue.draft_not_found404、project.not_found404、forbidden403、存储结构bad_request400、validation_failed422、issue.draft_changed409、project.archived409、server_busy503 |

路径均有 `/api/v0` 前缀。Draft 只公开 `{id,workspace_id,project_id,payload,created_at,updated_at}`，不回创建者/更新者/删除时刻。新增两公开码与 `PROBLEM_MESSAGES` 和 en/zh-CN errors.json 同 Task 出现；`issue.draft_not_found` 文案“草稿不存在或你无权访问。”/“The draft does not exist, or you cannot see it.”；`issue.draft_changed` 文案“草稿的项目已更改，请重试。”/“The draft's project changed. Please try again.”。锁超时沿 bootstrap 统一503与 Retry-After，不再造映射。既有非GET issue写完整性守卫在T4拆成普通issue、draft CRUD、draft publish三种显式锁形状目录，其union精确覆盖全部tagged写操作（publish在T6注册时加入），不能以excludeDrafts通用豁免或等T7才调整。

publish-only HTTP adapter守卫在现有认证/限流/BodyLimit之后、进入用例前核请求body：EOF/零字节允许，任意至少一个字节（包括空白、`{}`或chunked）通过现有APIErrors.BodyError答400；读错误沿该函数处理。只读至第一个字节判断，不按Content-Length单独判定；不扩全局bodyshape、不把HTTP概念下沉app/domain。

请求框架的401/结构400先行，存储可表示性的纯422在资源预检前。指定草稿按“活行/旧项目存在 → workspace可见和draft权限 → owner”预检；普通写W后按§2.3资源读时点重判，P/D锁下再次读并比较初读 project，变更409；显式新项目无效422 `project_id/not_allowed`。发布无项目422 `project_id/required`；工作区/项目权限先于项目归档，归档409先于存储 payload 检查，随后原创建400/422。并发删除旧项目即使草稿仍活也404 draft，不能退成422 project；等待中换项目则409，不在同一事务追锁新项目。

保存/删除草稿不发 issue 事件、不产生活动/关注/提及/版本/号序；publish 成功只一次 issue.created，返回的可见父/子计数沿原 readBack；事件、写入、草稿删除任一步失败整体回滚。已发布再请求404。

### 2.5 级联签名与历史边界

`ProjectRowsDeleter` 增 `DeleteProjectDraftRows(ctx context.Context,b RowBatch)([]uuid.UUID,error)`，在 issues 步骤之前按 id 锁/软删草稿。无项目草稿使用 `WorkspaceDraftBatch{WorkspaceID,After,By uuid.UUID;Limit int;At time.Time}` 和 `DeleteWorkspaceDraftRows(ctx context.Context,b WorkspaceDraftBatch)([]uuid.UUID,error)`，只选 `workspace_id` 匹配、`project_id IS NULL`、未删行。

贯穿 I/app.Cascade、I/module.go.IssueCascade、P/app.IssueCascade 和 P/app.Cascade 的精确目标：`DeleteWorkspaceIssues(ctx context.Context,workspaceID uuid.UUID,projectIDs []uuid.UUID,by uuid.UUID,now time.Time) error`。调用者已有 workspaceID，不能从非空 projectIDs 猜。所有新增行沿请求原 actor/At；先前已删除项目上的草稿不改旧审计，零项目工作区也删除无项目草稿。删除工作项不删 payload.parent_id 指向它的草稿，发布再422 parent_id。

`furnishIssueRows` 新增显式 workspaceSlug 参数，用 API 创建带项目草稿，更新所有调用点；工作区断言另创建无项目草稿。不可 DB 查 slug 或 SQL 填充故事前置，不在最终目录里留下 draft 豁免。P9 的异步删除/物理清理未在本期实现。

### 2.6 搜索端口、词汇与同快照 SQL

```go
// I/domain/search.go
type SearchFor string // 内部""仅代表省略；外部只接受parent/sub_issue/relation
type SearchQuery struct {
    Text string; ProjectID,IssueID *uuid.UUID; For SearchFor; Limit int
    StartDate,TargetDate []DateFilter
    Identifier string; IdentifierNumber,SequenceNumber *int
}
func ParseSearchQuery(map[string][]string)(SearchQuery,error)
// 复用 ParseIdentifier、MatchIdentifier、ParseListDateFilter 和 shared.PageSize。
// I/app/search_ports.go
type SearchRead struct {
    ActorID uuid.UUID; Projects []VisibleProject; Query domain.SearchQuery
    ExactProjectID *uuid.UUID // 在可见项目中MatchIdentifier所得，未匹配为nil。
}
type SearchRows interface { Search(context.Context,SearchRead)([]domain.Issue,error) }
// I/app/search_issues.go
type SearchIssuesDeps struct {
    ReadTx shared.TxManager; Authorizer shared.Authorizer; Workspaces WorkspaceDirectory
    Projects VisibleProjects; Identifiers ProjectIdentifiers; Issues IssueReader
    Locks Locks; Rows SearchRows
}
func NewSearchIssues(SearchIssuesDeps)*SearchIssues
func (*SearchIssues) Execute(context.Context,string,map[string][]string)([]domain.Issue,error)
```

查询键精确 `q,project_id,for,issue_id,start_date,target_date,limit`。未知键422 not_allowed；除两个日期键可重复（非none条件AND，none不能与同字段任何另一条件并用）外，任何重复标量422 invalid_format。route 定向保留 raw query，`url.ParseQuery` 百分号错误400，不用会吞错的 URL.Query；生成binding ErrorHandler也先解析raw query，保证重复q加畸形百分号仍400，不被重复参数422抢先；limit 非整数仍400。UUID/日期/for 值错误422，对重复日期按 `start_date[i]`/`target_date[i]` 路径报告。某字段多值且含none时，报告该字段首个none位置的not_allowed（none+date为[0]，date+none为[1]，none+none为[0]），全部422；不改既有列表的报错路径。q `TrimSpace` 后 ≤100 Unicode 字符，101→422 too_long；非法 UTF-8/NUL→422 invalid_format，不做模糊纠错。空白 q 等同空 q。

外部出现`for`时只接受parent/sub_issue/relation，`?for=`或unknown均422 for/not_allowed；`?issue_id=`/`?project_id=`均422各自invalid_format，不能视为省略。合法`for`出现必须有 issue_id（422 required）；只给 issue_id 无 for→422 issue_id/not_allowed；parent/sub_issue 的显式 project_id 与 source 项目不同时，在 scope/source授权通过后422 project_id/not_allowed。纯数字是 ASCII 十进制：`00012` 按 sequence12，0/超2147483647仅无数字分支，仍可标题匹配；完整 `PROJ-012` 不放宽既有 ParseIdentifier。编号项目折叠只在本次可见项目中用既有 MatchIdentifier（含 i/ı/I/İ、精确 uppercase 优先），不让隐藏同名项目改变匹配。

成功结果范围是 VisibleProjects.Visible 的有效项目成员、未删未归档项目；issue 未删未归档，按每项目 OwnOnly 过滤，workspace admin+project guest 也保留 OwnOnly。显式 project_id 不在该目录404 project.not_found；source不在同工作区目录、OwnOnly不可见、已删/归档一律404 issue.not_found。

顺序：parse纯参数 → ReadTx 内 workspace lookup/issue.search授权 → visible目录 → 显式project scope → source读取/OwnOnly → parent/sub项目冲突 → 标识目录/静态SQL → `Locks.seen` → 响应。404/403沿原工作区 Authorizer。整个过程经 `ReadOnlySnapshot()`，包括隐藏 parent_id 和 sub_issues_count；不把搜索结果再次套另一个目录事务，不取写锁。

静态 `search_issues.sql` 只引用 issue 自己表；projectIDs/OwnOnly/标识候选从端口传入。权限、日期、排除都在 LIMIT 前。空 q 按 `updated_at DESC,id DESC`；非空候选是 OR：精确项目+编号、纯数字 sequence、标题 `ILIKE` 字面包含。每行只出现一次，rank 精确0→数字1→标题2，之后 `similarity(name,@query) DESC,created_at DESC,id DESC`；精确或数字行也必须满足全部授权/日期/排除条件。`%`、`_`、`\` 作为用户字面量转义，绝不变成通配词；description_stripped 不搜索。

parent：仅 source 项目，排除 source、自身当前 parent、全部后代；sub_issue：仅 source 项目且 parent IS NULL，排除 source、全部祖先；relation：允许范围内跨项目，排除 source 和双向所有类型活关联。递归用 UNION 终止异常环，穿过隐藏/归档/删除的中间节点，不能因中间节点不可见而漏后代/祖先。返回 candidates 仍受自己的可见性约束。搜索不替代写时最终校验。

### 2.7 pg_trgm 与链接计数

`00029`：`CREATE EXTENSION IF NOT EXISTS pg_trgm`；`issues_name_trgm_idx USING gin(name gin_trgm_ops) WHERE deleted_at IS NULL`；Down 只移除本迁移索引，不删除可能先存在或为其他对象共享的扩展。列入 issue 的 sqlc schema，参数名 `@query`，生成 `SearchIssuesRow.Similarity` 必须可赋给 `float32`，不接受 interface{} 或通过 float64 绕开。最小查询/类型支架可在 T9 建立，T11 替换成最终完整 SQL 并再核类型。迁移用 PostgreSQL18.6 中有数据库 CREATE 的非超级用户 owner；runtime 经 PUBLIC EXECUTE 调用 similarity，保留关系/函数授权完整性检查。schema_test.go的public函数目录通过pg_depend扩展成员身份区分pg_trgm对象与应用自有函数，后者仍保持严格Up/Down目录；不按函数名黑名单放宽守卫。预先存在扩展的Down/Up测试核OID保持。

`domain.Issue.LinkCount int64`，OpenAPI 两个对象都 required `link_count`，integer/int64/minimum0。数量仅 `issue_links.deleted_at IS NULL`，不受链接作者、metadata.error/抓取状态影响；0 必须显式返回。基础 GetIssue/GetIssues、SubIssues、列表三 SQL、搜索和映射全部补齐，继承 create/update/archive/restore/readBack、标识查询、父子和关联内嵌；`IssueDetail` 只比 `Issue` 多 description_html 的守卫继续通过。

三列表 SQL 的最终字段名/顺序/类型同形（Groups/Subgroups→FlatIssuesRow 的现有编译约束）；link 排序只在相应 source_keys 需要时算，最终结果字段复用或 coalesce，不能为每种排序提前扫全量 links。排序码由19变21，加 `link_count/-link_count`；不加 link_count filter，display_properties 已有 link 不重复添加。计数键按非负 int64 编解码，保留 P5 内部 numeric 比较和 id 同向 tie-break。单请求同快照，跨请求链接数量改变仍是既有弱快照限制，不承诺冻结。

## 3. 自主裁定、理由和错判代价

这些是批准设计内的局部细化；若实施证明需改变模块责任、锁图或公开授权规则，按负责人纪律先提交已完成工作，再上报。

| 编号 | 结论 | 理由 | 若判断错误的代价/核验 |
|---|---|---|---|
| D1 | 13粗任务拆16，准备期只静态预检，真实红绿/变异在实施期 | P7方法与用户“暂不实现”一致 | 不得用准备稿勾验收；T16审准确范围 |
| D2 | DraftLocks组合base；consumer小端口；S锁与workspaceID是既有设计兑现 | 不扩大普通Locks构造，不另接auth/gate/tx | T7重推完整锁图与架构守卫，拒绝变第二条写链 |
| D3 | 六操作单draft.write；getDraft补403 | 权限表比§5.1漏项明确，管理员不拥有他人草稿 | 矩阵guest+他人403，member/admin+他人404；授权泄漏为阻断项 |
| D4 | 旧project删除→draft404；显式新project不可见/删除→422 | 前者是资源存在性，后者是输入；交错16明确404 | P9窗口及真实删除两次序测试；同步设计泛句 |
| D5 | 项目退出/归档不抹个人草稿；省略project更新或删除可操作；显式指定必须经project.read/LevelVisible可见 | 读只要求未删，个人草稿应能编辑/清理；发布才需create | 退出项目后的读/改/清空/删/发布表；若未来产品改变需显式契约变更 |
| D6 | payload整体替换，JSONB等值patch为noop；显式null仅外层project可用 | 契约optional与nullable可分别表达，避免隐式merge语义 | T4旧payload键消失/空对象/null/审计时刻精确断言 |
| D7 | NUL只拒3个自由字符串和search q；不调用完整create验证 | JSONB/text存储表示边界，不是草稿业务完整性要求 | T4真实422零行变化；长标题/坏优先级/坏引用仍可保存；静态发现非已复现 |
| D8 | 共用已持锁核心返回At；adapter解码stored payload | 保持domain纯净及一个事务/事件/clock，不放宽守卫 | T5/6语义与结构对照、事件/软删故障完整回滚 |
| D9 | 存储结构400比较状态/码/字段；不承诺任意坏JSON原字节顺序 | JSONB规范化顺序；API合法保存和注入损坏旧数据是两类 | 缺name真实存草稿与同体POST；注入坏payload使用同canonical bytes对照，含>16问题截断边界 |
| D10 | 搜索候选OR，纯数字前导0可匹配，0/溢出仅不走sequence | 不丢标题匹配，避免PG转换500；精确编号仍复用原规则 | 数字/编号/标题混合组逐值顺序；共享fold fixture |
| D11 | 未给for的issue_id422；parent/sub冲突422；source归档404 | 参数不静默丢弃，source必须来自本次可搜索范围 | T10/12优先级用例，scope先于冲突，不泄漏隐藏源 |
| D12 | 未知query与重复标量422；非none日期重复AND、none不可并用；literal LIKE；q按Unicode计数 | 沿列表词汇并关闭隐式参数歧义 | malformed percent400（含重复scalar同现）、100/101中文、NUL、%/_/反斜杠、组合日期/none互斥 |
| D13 | 空词id DESC，非空最后id DESC；所有scope/排除先于limit | 稳定前N，避免后过滤短页 | 多个同时间同similarity候选与隐藏高分项，limit1/2反例 |
| D14 | search从静态SQL完整Issue后唯一调用Locks.seen | 复用父屏蔽/子数定义，避免第二份OwnOnly计算 | adminGuest隐藏父、不同OwnOnly子数及同快照测试 |
| D15 | 00029 Down不drop扩展；不预承诺3字符必走GIN/固定延迟 | IF NOT EXISTS无法认领已有扩展；计划受选择性/locale影响 | owner Up/Down/Up保留共享扩展，100k真实最终SQL计划/耗时 |
| D17 | publish仅零字节body可用，非空含空白400；专属HTTP小守卫 | 无requestBody的生成handler不会自动拒额外body，显式契约边界防误用 | T6认证先行、零字节/空白/chunked/JSON/读错，确保不进用例或写行 |
| D16 | link_count全表示/双排序在一个Task落地；P5历史19不倒改 | 数据字段随数据/API进入，历史验收真实不改写 | 当前词汇/fixture21；历史P5事实保持并追加P8承接 |

## 4. 验收

以下按真实证据回填批准验收；命令/真实RUN、变异、提交与限制逐条列于[review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)，性能原表在附录A。A11文档修复已在准确a65e8605独评通过，最终提交CI仍由后续收据核验，不用产品HEAD通过替代未来文档提交。

- [x] A1：00028 九列/索引/级联/grants、无项目/零项目工作区删除、旧审计保留、furnish所有调用点和完整表目录通过（T1/4/8）。
- [x] A2：保存全 optional/整体替换/nullable/noop/NUL三路径；无额外issue副作用；自有游标/total及未知游标位置不泄漏通过（T2–4）。
- [x] A3：六操作权限矩阵含guest、ownOnly、adminGuest、退出/归档/删除项目窗口，404逐值一致，预检拒绝零锁/零gate（T3/4/6/7）。
- [x] A4：发布使用锁下存储载荷、与直接创建400/422同路径、完整副作用同At/actor、失败整体回滚、重复发布404（T5/6/8）。
- [x] A5：交错16/17双次序、相反项目UUID顺序移动、workspace删除/降级、真实等待者PID证明与恢复变异通过（T7）。
- [x] A6：pg_trgm非superuser owner、runtime函数/关系权限、00029 schema包含与similarity float32编译证据（T9/11）。
- [x] A7：空词、数字/Unicode编号、literal包含、重复参数/日期/冲突、全深度/双向排除、同ReadOnlySnapshot的隐藏父及子数通过（T10–12）。
- [x] A8：link_count每种表示、活/软删/metadata数量、21码双向平铺/分组/子分组/成员/偏好、整数游标和伪造游标不越权通过（T13/14）。
- [x] A9：I14/I18接口、I4两方向跨页及所有受影响目录故事通过（T8/14）。
- [x] A10：10万英文/中文数据的2字和≥3字真实最终查询参数、locale、计划、样本数/耗时写入附录；不关闭seqscan作伪证（T15）。
- [x] A11：四次完整本地七门禁先于产品检查点push，新码文案/架构/关键词不放宽，首次整审完成；10f44790..a65e8605文档独评Approved，历史D1已独立ADDRESSED，证据为task-16-review.md（T16）；最终准确分支HEAD `cde70d9735994cb6a579e0b800469f32631e9981` 的 [CI 40515294403](https://github.com/open-nerve/NerveProject/actions/runs/40515294403) 已通过，集成回执见review§9。
- [x] A12：§3.23上级文档/README和输入handoffs已追加准确结果，BASE/产品HEAD/spec/plan/review路径已交付；已审文档a65e8605/独评task-16-review.md及最终状态复核均有原件（T16）。交付时未合并的历史状态保留在branch-final-receipt.json；其后已授权集成、main CI通过、归档及清理完成，见review§9。

## 5. 不在本期与后续接手

| 后续 | 本期边界和接手输入 |
|---|---|
| P9 | 异步项目/工作区删除、60天物理清理、描述版本保留；接草稿表、workspaceID批量接口、项目删除窗口与固定原请求审计；本期只同步删除 |
| P10b/P11/P12/P14 | 当前生成TS的编译fixture适配可做；生产前端比较器、详情store/service、日历/ExistingIssues、父子关联选择器仍由后续阶段实现 |
| P15 | 草稿页面、创建弹窗“存草稿”、命令面板、旧ProjectService/WorkspaceService搜索移除；必须先PATCH再无body发布，不误以搜索已代替写校验 |
| M5/M6/M7 | 发布附件在同事务移交留M5；迭代/模块/收集箱字段不加入本期payload、搜索或表 |
| M8/M4收尾 | 扩展部署/维护最终交接、总体设计9.4与M5–M8/M4-closeout、所有交接status关闭由M4收尾统一做，本期不提前标完成 |

## 6. 风险与验证责任

草稿加入 W/P/D 锁序，不能沿用“所有issue写最先P”的旧概括；T7按当前全部writer重画gate/行锁资源图并真实观察阻塞。读权限目录三种语义（未删、保存project.read/LevelVisible、搜索成员Visible）不得合并。创建核心抽取不能漏P4清洗/版本、P6事件，也不能让普通创建的拒绝顺序回归。

schema生成/JSONB规范化可能改变错误截断顺序，T5对规范存储体明确比较，不承诺未经证明的任意字节顺序。搜索递归必须穿过隐藏中间节点但不返回它们。`link_count`最终投影多，T13从每处domain.Issue构造向上核对到公开路由。100k性能只说明记录的环境/参数，不保证任意生产分布。

已知W11偶发不专查；若合并main CI被它挡住且重跑仍失败，按用户要求停止报告。其余新失败先根因修复。若架构回归、改变既定跨模块协议或意外高风险，先提交已完成工作再用大白话上报；本spec已批准的端口补齐不是重复审批点。

## 7. 交接承接与关闭口径

| 输入 | 本期处理目标；证据归属 |
|---|---|
| P1a/P1b/P2 | workspaceID贯穿、草稿专属锁形状、闸门/超时和RO/RR、issue.search/draft.write授权；T1/3/4/7/12 |
| P3/P4 | 多层父子/双向关联、隐藏父和可见子数；HTML清洗/提及/版本、副作用、共享PageSize；T5/6/11 |
| P5 | link_count/两码、日期/Visible/游标/同快照；仅追加P8承接结果，不改P5当时19码完成事实；T10–14 |
| P6/P7 | 草稿保存无事件、发布唯一issue.created；links活数/删除和metadata不改数量；T6/13/14 |
| [M0-P4](../handoffs/M0-P4-pg-trgm.md) | 扩展迁移与镜像/owner验证，部署部分留M8；T9/15 |
| [M1-P2](../handoffs/M1-P2-trim-content.md) | 搜索结果无page；T12/14 |
| [M2](../handoffs/M2-closeout.md) §2/4 | 草稿游标篡改、发布存储体结构同400，保留封闭schema与代价守卫；T2/4/5/6 |
| [M3](../handoffs/M3-closeout.md) §2/3/4 | 新写锁序/矩阵声明、级联表目录、无项目草稿；T1/4/7/8 |

各handoff追加“处理结果（M4/P8）”及真实证据；总体status保持open至M4统一收尾。实施完成时review按原要求记录精确BASE..HEAD、审查方法/发现/修复/复审、门禁和CI；随后用户已授权集成，结果见review§9。

## 附录 A：准备与实施证据边界

以下保留各阶段原始时间点。准备原件 `.superpowers/sdd/P8-drafts-search/preparation/` 仍在主项目根目录原处，不属于本次worktree归档；T8/T15等实施证据中的同名相对根则是执行时P8 worktree路径，当前完整原件归档在主项目根目录 `.superpowers/sdd/P8-integration-20261027/worktree-evidence/.superpowers/sdd/P8-drafts-search/`，核验与清理事实见review§9。

2026-10-27：只读核对M4设计、P3样板、P7交付树、现有创建/锁/列表/游标/HTTP bodyshape/删除接缝；独立需求和代码盘点保存在 ignored `.superpowers/sdd/P8-drafts-search/preparation/`。规格作者和独立审者没有跑迁移、生成器、产品测试、SQL原型或性能实验；主控另已运行既有关键词/架构检查，结果见评审记录，不构成P8产品验收；NUL可存性和生成风险为静态推导，均有实施期验证Task。P7基线CI事实来自主控对准确SHA的核验；P8尚无实现或运行证据。

实施后仅追加有原始输出/准确SHA的记录：Task红绿及实际RUN数；生产变异红因/层级/恢复hash；交错PID链；T4/8/12/15门禁；10万性能数据分布、PG版本/locale/LC_CTYPE、最终SQL及参数、EXPLAIN(ANALYZE,BUFFERS)、暖/冷说明与至少5次样本中位数/范围；deep-review和定向复审。未执行项保持未勾。

2026-10-27，T8作者聚焦证据（基线`2eb2ba80eaafa9e3c8e12bb8e5b34efdc8d526b8`，最终提交及源码hash收据见 ignored `.superpowers/sdd/P8-drafts-search/task-8-author-report.md`）：

- I14三条HTTP故事覆盖缺name保存201/发布与直接创建同400、PATCH整替换、description/parent发布201、草稿list消失/get及重复publish404、同At/actor审计、失效state422、他人404/project guest403，以及归档/退出项目仍列草稿且发布按当前权限拒绝。前置均走API，数据库仅作断言。先`make build`后首轮3/3直接绿，反映T1–T7已有实现，不当作初始红。
- M14在生产`publish_draft.go`真实略去消费草稿，`make build`成功后同I14因发布后草稿仍在列表而红（1失败、2通过）；恢复源码原字节及SHA256 `3aea4485bde8a0eddee1cd2bf6815f11b8c09a66845ef0534d705f29101f530f`，重新`make build`后同故事3/3绿。编译来源/二进制hash及原始红绿输出保存在 ignored `task-8-evidence/m14-receipt.json`和`checks/t8-m14-*`。
- P4/W2/W3指定回归6/6通过；W3新增显式nil-project草稿、零项目工作区HTTP删除的动态全表目录断言及已删旧项目草稿原始微秒审计前后快照。P4/W2复用T4的API furnish与动态目录；bootstrap发布/删除/目录/子项28 RUN、存储层草稿/链接删除13 RUN均通过，保留原subItems/links覆盖。

此记录仅为T8作者聚焦/影响验证；完整F、独立评审、准确HEAD push及CI仍由主控完成后补证。A9还包含T14的I18/I4，未在T8提前完成；各阶段验收项保持未勾。


2026-10-27，T15 实际规模测量与最终性质回归（完整 F 和 T16 仍待主控）：

- 测量输入为基线 `88ba09fa87883e11e3f6702998a9a7e53bef8a5b` 加未提交的三个性能测试文件和 plan 变更，**不是干净提交上的运行**。最终提交与这些字节的映射、逐文件 SHA256、完整原始输出见 ignored `task-15-author-report.md`、`checks/t15-measure-restored/{result.json,output.log}`、`task-15-evidence/restored/metadata.json`。生产搜索 SQL/生成 Go/adapter 未修改；本段文档在测量之后追加，不将文档提交本身说成运行时 HEAD。
- 显式 `NERVE_SEARCH_PERF=1 go -C server test -p=1 -count=1 -v ./internal/modules/issue/adapter/postgres -run '^TestSearch100KPerformance$'`：13 个场景加父测试共 **14 RUN/14 PASS**，每场景 1 次 first-observed + 5 次 warm。默认关闭时是明确 skip，不能据此称性能已验。首轮真实尝试在完成 seed 后因把 `lc_ctype` 当运行设置而失败，没有产生生产耗时；按 PG18 `pg_database.datctype/datcollate` 修正取证，原日志/seed 保留。第一次完整成功测量与随后 OwnOnly 恢复测量均保留，下表采用恢复后的完整成功测量。
- 固定 seed `20261027`、生成版本 `search-100k-v1`，**恰好 100000 行 issues**：每个可见项目 50000（英文 25000/中文 25000），共英文 50000/中文 50000；每项目 actor-owned 47056、归档 496、软删 488、rare 日期样本 1352，均包含在 50000 内，类别可重叠。第二项目目录为 OwnOnly；另有两项目均 OwnOnly 和单项目场景。四种标题包含受控 exact-title 领导行及字母后缀变体，rare 条件为 `start_date=2026-06-15`。Go 从固定 fixture 元数据独立枚举过滤前后数量，并按受控 exact-title/created_at/id（空词 updated_at/id，编号 rank）给出 topN；所有返回 ID、name、活 link_count 均比较固定期望，期望不是由 Search 输出生成。候选 count 是保留 captured candidates CTE 原谓词及全部 17 个绑定的单独诊断，不计作生产耗时。
- 真实最终投影还读取 **20000 issue_links（10000 活/10000 软删；活行含 metadata）**，exact-title 行各 1 活链接；issue_assignees、issue_labels、issue_relations 均为空，此成本边界不外推为关联密集生产分布。owner `postgres` 负责 seed/五张相关表 ANALYZE；测量角色 `test_1_search_runtime` 为非 superuser、属于 `nerve_runtime`，按实际 grants 配置。PG **18.6 / Debian / aarch64**，`LC_CTYPE=LC_COLLATE=en_US.utf8`、UTF8、libc provider，`datlocale` 对该 provider 为空；Go/CPU/平台/索引定义和逐表 last_analyze/analyze_count 均有 metadata 原值。
- 计时是生产 pool 配置下 `TxManager(ReadOnlySnapshot)` → 完整 `Store.Search`（含 LinkCount 与 native float32 similarity 的最终生成 SQL）。每次捕获同 snapshot PID、实际 BEGIN RR/RO、完整 SQL、原始 17 typed args（保留 typed nil/空 slice）；`jit=off`、`plan_cache_mode=force_custom_plan`、`enable_seqscan=on`。原 SQL 与后续 EXPLAIN 的参数 OID 完全相同：`25,2951,1000,2950,25,2950,23,23,2950,25,1000,1009,1009,1000,1009,1009,23`。精确值/Go 类型/最终 SQL/参数和 `EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON)` 在逐场景 samples/plans 文件，不以手抄 SQL 代替真实 capture。
- 下表单位 ms；snapshot 包含 BEGIN/Store.Search/COMMIT 的客户端总耗时；pgx 列为 TraceQueryStart→End（含客户端往返、行读取/扫描），**不是纯服务端执行时间**；PG Execution 列是另一次 EXPLAIN 的单次服务端 Execution Time，不是五次中位数。first-observed 列 **不是物理冷缓存**：COPY/ANALYZE 已触页，未驱逐 shared buffers 或 OS cache；诊断/EXPLAIN 在该场景采样之后，亦可暖后续场景。未关闭 seqscan、未声称固定延迟达标。

| 查询/范围 | 独立/实际候选数 | first snapshot | warm snapshot min/median/max | warm pgx min/median/max | 单次 PG Execution |
|---|---:|---:|---:|---:|---:|
| lo | 47786 | 47.936 | 36.827/37.686/39.865 | 36.331/37.180/39.360 | 38.369 |
| lo / rare | 1302 | 38.348 | 37.516/37.708/39.640 | 36.926/37.196/39.024 | 37.502 |
| login | 23791 | 28.631 | 26.553/27.330/27.568 | 26.099/26.887/27.045 | 27.129 |
| login / rare | 643 | 27.273 | 26.850/26.980/27.600 | 26.350/26.455/27.162 | 26.196 |
| 登录 | 47582 | 45.457 | 42.177/42.310/43.584 | 41.579/41.754/43.056 | 42.846 |
| 登录 / rare | 1286 | 43.561 | 42.246/42.488/43.039 | 41.683/41.884/42.427 | 42.724 |
| 修复登录 | 23791 | 32.636 | 30.620/30.903/31.006 | 30.168/30.495/30.562 | 30.647 |
| 修复登录 / rare | 643 | 30.653 | 30.200/31.028/31.348 | 29.795/30.631/30.901 | 30.713 |
| 空词 | 95164 | 47.935 | 42.171/42.493/43.156 | 41.644/42.022/42.776 | 52.368 |
| 31415 | 2 | 1.390 | 1.074/1.135/1.149 | 0.719/0.761/0.789 | 0.289 |
| PERFA-31415 | 1 | 1.536 | 1.306/1.342/1.370 | 0.979/0.994/1.024 | 0.563 |
| lo / 两项目ownOnly | 46340 | 41.812 | 37.180/37.584/38.061 | 36.646/37.123/37.566 | 36.465 |
| login / 单项目 | 12256 | 30.393 | 29.384/29.521/29.717 | 28.903/29.027/29.217 | 30.931 |

- 八个标题高/低选择性场景实际均选择顺扫；rare 日期显著减少候选数，但该 fixture 下没有因此缩短为索引查找。单项目 login 实际使用 trigram bitmap；纯数字/完整编号使用编号与 trigram 的 bitmap OR。结果未暴露需改变生产查询的契约内退化；这里只说明记录的分布/环境，不保证其他生产数据形状。
- 代表生产变异把 adapter 目录 OwnOnly 参数改为 false，真实 100k 用例在 topN[16] 多出他人固定 ID 而红（2 RUN/0 PASS，`checks/t15-own-mutation-red`）；恢复原字节 SHA256 `8d731dcac54e69fa4f064d39ac9622a3efe3829a7e0b6e67f03a187dde03f67b` 后上述 14 RUN 全绿，恢复收据在 `task-15-evidence/mutation/restoration.json`。编译 import 冲突和 locale 取证失败不是产品行为红。
- 消费独立 `mutation-audit-through-t14.md` 的 M1–M36、T14 真实 M26/M33 E2E 变体与恢复链；保留旧生成恢复方式、T7 未跟踪测试 hash 限制、T9 overlap 和 T13 import-only 关闭的真实历史，没有重跑旧 97 份日志来改写历史。本 Task 交错 16/17 `-count=5 -race` **65 RUN/65 PASS**，含真实 PID/holder 边；P5 `TestIssueListReadSnapshot`、`TestMemberIssueReadSnapshot`、`TestIssueListsAsRuntimeRole`、`TestIssueListPermissions` **4 RUN/4 PASS**。这补最终源的规定回归，不替代主控尚待执行的完整七门禁 F、独立 T15 评审及 T16 整分支审查/交接。
