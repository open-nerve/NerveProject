# M5/P5：内容绑定、权限与草稿迁移子合同

> 与 P5-issue-assets.md 同属通过预检的子合同。没有新的模块 import、跨模块 SQL 或通用任意闭包端口。

## 1. 职责与值

issue/app 声明消费端口；asset 仅操作 file_assets；bootstrap 转换双方具名值。不得把 asset/domain 类型传入 issue/domain。域内纯规则不解析 HTML、不依赖数据库或文件系统。既有 ImageBinder 留在资料图片用途，不混入内容分支。

issue/app/content_asset_ports.go：

```go
type ContentHost struct { WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID }
type ContentBindRequest struct {
    ActorID uuid.UUID
    Host ContentHost
    ImageIDs, AttachmentIDs []uuid.UUID
    NewComment bool
}
type ContentBinding interface {
    LockContent(context.Context, ContentBindRequest) (time.Time, error)
    BindContent(context.Context, ContentBindRequest, time.Time) error
}
type DraftAssetRequest struct {
    ActorID, WorkspaceID, DraftID uuid.UUID
    OldProjectID, NewProjectID *uuid.UUID
    ImageIDs []uuid.UUID
    Publish bool
}
type DraftAssetTransfer interface {
    LockDraftAssets(context.Context, DraftAssetRequest) (time.Time, error)
    MoveDraftAssets(context.Context, DraftAssetRequest, uuid.UUID, time.Time) error
}
```

两阶段端口不把可变行对象/DB连接交回调用者。Lock在既有ctx里收齐并按全局UUID锁定，检查完整集合，返回最后锁后唯一业务At；Bind/Move同ctx重验锁下条件、保留申请者/created时间、按At写归属/用途/bound与updated，并增加storage_revision。新评论Host.CommentID是预分配最终ID，NewComment使原范围固定为同issue的new_comment，不能由客户端自由指定。

LockContent只核所列当前图片+普通附件集合；旧已绑定历史图片不解除。重复ID按UUID去重锁一次，但正文ContentIdentity仍保留重复。每SQL最多1000 ID，全请求集合不得截断；任一缺失/未完成/用途/父链/申请者失败整体回滚。零UUID虽然是规范UUID拼写，但不是有效资产，绑定端返回asset.not_found，不能将其过滤掉冒充非资产源。Bind在INSERT宿主后执行，不能靠延迟FK提交半成品。

LockDraftAssets在D(U)保护下分页遍历全部活资产、合并ImageIDs后全局UUID顺序加锁，单页1000，直到真实耗尽；保存/改项目包括pending、uploaded，并验证ImageIDs为本草稿已完成的描述图片。Publish存在任一尚未成功发布的活资产即asset.not_uploaded，不能默默丢件；已经成功发布的资产携带待回收竞争候选不阻止发布。MoveDraftAssets的issueID：保存/改项目为Nil，更新父项目并只绑定ImageIDs涉及的描述图片；发布为新IssueID，清draft_id、换scope=issue、用途两种draft→issue；普通附件、当前ImageIDs中的描述才在bound_at为空时设置At，已有bound_at保持，未引用过且不在当前ImageIDs的描述保持null（不自动公开、不重置7天年龄）。已删除资产不复活，原草稿父引用留作回收证据。草稿路径只调用这组完整集合端口，不再额外调用LockContent造成第二次加锁/读钟。

**P7接收补充（2026-11-04，负责人批准A）：** 上述发布合同保持。未知ID的prepare/duplicate失败准备件由[P7a私有查询](P7a-draft-preparations.md)发现，P7b让作者逐项明确丢弃，复用既有DELETE，之后publish仍在锁下核全活集合。查询不绑定/删除、不提供发布许可，已绑定历史图不进入准备列表；不因恢复入口放宽pending拒绝、迁移集合或清理规则。P5完成记录保留，本补充不冒称P7新协议已实施。

app/asset端口不可嵌套开事务，不取得更上层账号/工作区/项目锁，不执行存储I/O。内部SQL专用更新入口，与SaveAsset不可变scope合同分开。

## 2. 锁序与调用图

| 入口 | 已有owner边界 | P5后续 |
|---|---|---|
| issue正文更新 | Locks.run：无连接gate→P(N)→重读/授权/归档/相关检查 | 当前引用集合A(U，UUID)→唯一At→正文/提及/动态/版本/资产绑定→事件→commit |
| 创建/修改评论 | 同一Locks.run与commentCheck，保持guest 404优先 | A(U)→唯一At→comment INSERT/UPDATE及binding→原评论事件→commit |
| 保存/改项目草稿 | W(S)→旧新P(S，去重升序)→D(U)→重验 | 必须先收齐整个draft集合及新增引用，再一次UUID升序A(U)→唯一At→draft+资产→commit |
| 发布草稿 | 无连接gate→W(S)→P(N)→D(U)→锁下decode/原创建校验 | A(U)→唯一At→Issue创建+资产迁移+draft软删+精确事件→commit |
| complete/duplicate/delete/content | asset Access：gate→caller→W(S)→P(N)→D(U，若有)→父权限重验 | A(U)→唯一At；签名仅CPU，外部封存/复制仍在两事务之间 |

普通issue写仍不额外锁W，不复制第二套Locks.run；账号锁仍由已有相应入口负责。资产事件消费issue行必须证明所属P(N)已持有，不重进gate、不逆向取得其他父锁。原有issue相关行插入/更新受同一P(N)串行保护。所有后续读失败原样返回。

改项目若既有草稿绑定图与payload图片集合重叠，不能先锁已有A高ID再锁新A低ID：准备阶段先读全候选，合并/排序后一次逐批锁；D(U)挡住新上传申请和最终确认，锁下再验父链/集合。取消任一阶段释放pool/gate，回包迟到不迁移到新宿主。

## 3. 创建核心与发布原子性

IssueCreation保持唯一创建逻辑。把createLocked内部拆为私有prepareLocked和writePreparedAt：前者做原相关校验、清洗、mentions/assignees/labels、分配序号及最终ID，尚不写业务行；后者消费准备值和唯一At，写同一图并发原issue.created。Sequences.Next属于本事务，任何后续失败回滚序号。

两个私有签名固定为 `prepareLocked(ctx context.Context, actorID, projectID uuid.UUID, h held, in domain.NewIssue) (preparedIssue, error)` 与 `writePreparedAt(ctx context.Context, p preparedIssue, at time.Time) (domain.Issue, error)`；preparedIssue仅本app可见，包含最终IssueRow（At/CompletedAt在write时设置）、checked和RichText。普通createLocked保留薄包装，发布直接复用两步，不写第二套创建图。

普通CreateIssue继续薄组合prepare→必要资产校验→Clock/At→write；新issue没有事前scope，不能引用别处的图片UUID。草稿Publish用同一准备值，验证其图片在草稿资产集合内，LockDraftAssets后得到At，再write、Move、DeleteDraft；不得先createLocked读时钟后才锁资产。创建返回值在资产迁移后再补真实附件计数，不能提前返回0。

asset 的 DraftAssetDeps 与外观显式接收已有 EventPublisher；Move 完成全部资产写后，按 UUID 顺序为迁入的普通工作项附件发布 added，包含原来已绑定的草稿附件。同一个 At/By，不扩张 issue 消费端口、不另开事务。issue.created 保持唯一创建核心内的原位置；附件事件看到的是完整迁移后的集合。

发布任何点（Issue/描述版本INSERT、任一资产UPDATE、draft软删、event subscriber、COMMIT）失败：工作项/序号/文件scope与用途/绑定/审计/动态/事件/river_job全回滚，物理字节未搬动。双发布同D(U)仅一胜者；失败后原draft可重试。先软删的资产或宿主不因重试复活。

## 4. 已绑定读取与删除

基线ownedAsset不能简单去掉applicant条件。以具名纯规则区分：资料四用途与所有未绑定件仍仅申请者；已绑定issue/comment两对用途可按host读；draft两用途仍由owner确认作者。读取先获得资产候选，再宿主可见性；隐藏/不存在/异主未绑定的HTTP码体头一致，不返回私有名称。

Access将资产ID与明确HostRequest关联，而非按两个数组位置猜；duplicate同时有源Read和目标Prepare，若assetID同一重复须归并一致请求，否则拒绝。源允许读不表示目标允许写。全请求父锁仍按UUID排序，语义拒绝按源后目标；最后锁下逐资产重验publication/bound/scope/intent/applicant。Complete始终仅申请者，不能由其他成员替人完成准备件。

删除普通已绑定附件依据host Remove；已绑定描述和资料仍asset.bound，不能借新共用规则放开。绑定状态改变、draft改项目或发布发生于外部I/O期间，最终以currentPlacement返回409并保留可回收候选。旧能力URL在到期前有效的限制不变。

## 5. 必须证明的交错/回滚

- 不同上传者的成员可读/复制已绑定附件；无写权不能删；guest他人/不可见私有项目404逐值相同，管理员兼guest两种OwnOnly。
- 已绑定描述只能同宿主引用；别人准备件、当前scope一样但用途错、最后一个ID坏、重复ID、1001集合不能部分绑定。
- issue/project归档、父删除、成员撤权、账号停用与asset操作两顺序；真实用户首锁/W/P/D/A等待点和取消后复用。
- draft双发布、改项目两UUID方向、确认正在I/O时改项目/发布、发布与删除、失败后下一进程恢复；真实PID与事务故障，不固定sleep。
- 活历史图片移出正文超过20版本仍能授权读取；另准备上传完成但从未引用的图片，发布后bound_at仍null、其他成员读404、申请者可删、原7天年龄不重置。两者不能混为同一状态。不能按当前HTML反向删历史图片。
