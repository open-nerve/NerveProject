# M5 / P4：图片绑定和读取编排

> P4主spec的必要子合同，预检通过、T1–T11绑定、读取和全部owner出口已接线，T10五轮真实交错已通过，T11实物故事已通过，T12–T15联合验收及整审待完成。仅用于profile/workspace/project图片；不改P2通用准备件API，不提前实现P5。

## 1. 资产侧的精确接口

以下类型位于asset/app，外观只通过asset根包公开别名/能力。owner各自在自己的app定义自己使用的值和接口，bootstrap逐值转换；owner不能import资产模块。purpose由各字段固定映射，不能接收外部“已授权”布尔值。

```go
type ImageBindingHost struct {
    ApplicantID uuid.UUID
    WorkspaceID, ProjectID uuid.UUID // user范围都为零
    NewProject bool // workspace已有，project尚未插入；该ID由owner生成
}
type ImageChange struct {
    Field string // avatar_asset_id / cover_image_asset_id / logo_asset_id
    Purpose domain.Purpose
    CurrentID, NextID *uuid.UUID // 两者均为owner确定的有效请求结果，不含省略态
}
type ImageBindingRequest struct {
    Host ImageBindingHost
    Changes []ImageChange // 单个owner，固定字段顺序；无变化项也可带入以验证真实当前引用
}
type ImageBindings interface {
    LockImageBindings(context.Context, ImageBindingRequest) (time.Time, error)
    ApplyImageBindings(context.Context, ImageBindingRequest, time.Time) error
}
type ImageReference struct {
    AssetID uuid.UUID
    Purpose domain.Purpose
    UserID, WorkspaceID, ProjectID uuid.UUID // user图像验证UserID；workspace/project不限制旧申请者
}
type ImageURL struct { URL string; ExpiresAt time.Time }
type ImageReader interface {
    ReadBoundImages(context.Context, []ImageReference) (map[uuid.UUID]ImageURL, error)
}
```

ImageBindings的app实现接AssetLocker/MetadataWriter/ImageBindingWriter/Clock四个消费小端口；LockAssets已返回完整行，不再注入未消费的AssetReader。ImageReader接AssetLocker/ReadSigner/RuntimeNow。方法必须在调用者已有事务中运行。没有外部对象操作、Access.Within、gate.Enter、账号/owner回调，也不传递SQL/连接/任意闭包。各自构造文件按职责拆，小接口没有暴露完整Storage。

T2代码核对修正：现有SaveAsset刻意不改申请者/scope/bound_at，继续只用于旧件软删/恢复metadata，不放宽它。新增资产内部 `ImageBindingWrite{AssetID,ProjectID,By uuid.UUID; Revision int64; At time.Time}` 与 `ImageBindingWriter.BindImage(context.Context, ImageBindingWrite) (domain.Asset,error)`。固定SQL只将未删、已上传、未绑定的四种图片按Revision设bound_at/审计并递增Revision；仅原new_project行变为project并填ProjectID，其他scope/父列和申请者不变。适配器拒绝无事务调用；未匹配返回原scope_changed。owner两方法及构造边界不变。

LockImageBindings取全部CurrentID/NextID并集UUID升序，验证并返回**最后锁之后**的业务时刻。Apply使用同一完整请求和At、同一ctx事务，再在相同集合取锁并重验（已持锁不追新ID），应用真实变更；不是调用者指定任意bound/deleted状态的入口。校验流程共享一个内部纯规则实现，不能两份逻辑漂移。跨两次调用不允许结束事务/外部I/O。所有需要的旧引用在owner锁下先读到；无变化时不生成资产UPDATE。

owner的小端口命名沿用这两个方法，但请求值按本模块裁剪：
- identity `ProfileImageChanges{UserID; AvatarCurrent,AvatarNext,CoverCurrent,CoverNext *uuid.UUID}`；
- workspace `LogoImageChange{ActorID,WorkspaceID uuid.UUID; Current,Next *uuid.UUID}`；
- project `CoverImageChange{ActorID,WorkspaceID,ProjectID uuid.UUID; NewProject bool; Current,Next *uuid.UUID}`。
对应返回At，Apply同请求+At。Read端口各为 `ReadProfileImages(ctx, []ProfileImageReference)`、`ReadWorkspaceImages(ctx, []WorkspaceImageReference)`、`ReadProjectImages(ctx, []ProjectImageReference)`；workspace成员另用 `ReadMemberAvatars(ctx, []MemberAvatarReference)`，PublicProfile增加AvatarAssetID，身份端口只返回事实，不自行签发。每个reference包含所属实体ID及可空的字段ID；结果为本模块app的ImageURL映射，domain只承载纯值。

## 2. 绑定完整事务

| 操作 | 连接/锁/时刻/写的顺序 |
|---|---|
| UpdateMe | 认证/字段检查 → Tx → 自己账号N及CredentialLock重验 → 读当前User → LockImageBindings(old/new并集) → At → UpdateUser最终字段 → Apply → 签发响应 → commit |
| UpdateWorkspace | 认证/字段检查 → Tx → CallerLock账号N → W(N) → 既有workspace.update判定 → 读真实logo → LockImageBindings → At → UpdateWorkspace → Apply → 签发 → commit |
| UpdateProject | 原预读/权限预检 → 无连接进同实例项目gate → Tx → CallerLock账号N → W(S)/必要目标成员锁 → P(N) → 原权限/归档与业务检查 → 读真实cover → LockImageBindings → At → UpdateProject → Apply → 签发 → commit → 退gate |
| CreateProject | 字段/工作区创建预检 → Tx → CallerLock账号N → W(S)/创建权限 → 成员share排序/lead检查 → 生成的新projectID及候选LockImageBindings → At → INSERT project（一次写最终cover字段）、成员/偏好/默认state → Apply（new_project转project）→ 签发 → commit |

新projectID在同一事务首次插入，不对不存在的新行进入项目gate，也不追任何既有父锁。INSERT的cover FK目标是已经锁住的准备资产；随后资产的project FK指向新行，两边在提交前一致。失败时两侧、成员/偏好/状态全部回滚。旧项目更新进入的gate必须是bootstrap同时给issue/asset的实例；预检仍用project自己的decide，不复制权限谓词。

两个图片字段同时提交先收齐全部ID，At只能读一次。项目builtin切换的最终asset引用为null，其旧件照同一协议软删；builtin没有asset行。名称等普通变更如果需要返回已有图片仍经过签发所需的账户/父/资产边界；不为不相关label/state等操作全局改project.Locks。

所有owner对asset的错误原样返回，公开操作x-problem-codes补真实可能错误。绑定值非法保持原owner字段验证400/422分工，目标owner不可见与不存在仍逐值同形。新候选非当前引用时先检查所属用户及其完整允许范围；范围不符统一asset.not_found，不回取另一个owner锁。当前引用检查不要求申请者==当前管理员。

## 3. 签发端口与批量

ReadBoundImages只消费从owner事实产生的引用。单次先收齐所有非零资产UUID，去重升序。每条SQL最多1000个ID；超出时按同一全局顺序分块，先取得所有块的锁并校验，再签发。不能在签完一部分后追一个更小ID，也不能按输入显示顺序加锁。重复引用结果只签一次。各用途校验准确的宿主：user的UserID，workspace的WorkspaceID，project的WorkspaceID+ProjectID。准备件、未上传、deleted、无active final、scope/purpose不匹配不签发，返回该引用缺项，由owner将URL/期限设null。真正SQL、metadata解码或签名失败原样error，不作为缺项。

同一AssetID对应不同purpose或宿主的reference集合属于端口输入不一致，在签发前整体报错，不能按ID去重后把一个合法引用的结果送给另一个非法引用。SQL查询数为O(ceil(唯一资产ID/1000))，不是每图一次；签名纯CPU，零UPDATE资产/owner/计数/审计。URL/ExpiresAt使用P2同一contentExpiry函数与P3精确签名，期限来自runtimeNow，不依赖业务时钟。结果只含URL和expires，不泄露key、存储配置或申请者。

owner拿到用户资料的头像引用时不锁“目标用户”账号：M3已经允许在W/P锁下无锁批量读取PublicProfiles，防止与目标用户停用的账号→W顺序成环。只锁当前调用者账号第一把。若目标用户换图抢先提交，旧asset已软删，ReadBoundImages不再签它，允许这次回答null；后续重读得到新图。若本次先锁到有效旧asset并签发，则换图等待本次提交，旧URL仍按bearer期限有效。缺项不从客户端猜测另一个ID重试或越序追锁。

## 4. 所有响应入口

不能只处理GET后让其他返回同一DTO的操作写回假null。owner中可共享私有 `withImages` /批量转换函数；它只在已建立下表边界后调用，不在HTTP DTO转换器认证/开事务/访问存储。

| 入口 | 边界 |
|---|---|
| identity GetMe/UpdateMe | 自己账号N+CredentialLock，User读取和两图签发在同一事务 |
| workspace Get/List | 预读范围候选，Tx内当前账号N → 候选W(S)按ID → 活成员事实重读/既有授权 → 批量logo |
| workspace Create | 新工作区没有logo准备范围，回答真实null；原账号/创建规则保持 |
| workspace Update | §2；W(N)内读写logo并签发 |
| workspace AcceptInvitation | 账号CredentialLock先于原responder账号/工作区锁；既有邀请/成员恢复和project cascade结束后签工作区logo；任一失败同事务回滚 |
| workspace ListMembers | 预读workspace → Tx内CallerLock → W(S) → 原member.list判定 → memberships + PublicProfiles无目标账号锁批量读取 → 批量avatar；guest邮箱规则保持 |
| workspace UpdateMember | CallerLock第一把；原W(N)/角色规则/必要project cascade之后、commit之前签返回成员头像 |
| project Get/List | 原可见性预检收齐候选 → 无连接进入相同项目gate → Tx内CallerLock → W(S) → P(S)全部UUID升序 → 原权限/过滤重读 → 批量cover |
| project Create/Update | §2 |
| project Archive/Unarchive/Join | CallerLock第一把；原W/成员/P锁、授权与变更结束后读取最终project并签cover，commit后再返回；不改变原动作权限与归档语义 |
| public GetInvitation | 首先MAC验证（错误token仍零查询）→ 无锁预读邀请的workspace → Tx内W(S) → 邀请行S并重读同一未删/未accepted邀请及有效workspace → 固定logo签发；没有actor时不造账号锁，不接收任意资产ID |

公开邀请预览是M3已有的“有效链接持有者可看工作区名/slug/logo”的窄能力，OpenAPI原字段明确待M5；它不授予workspace.read，也不开放generic资产API。M3的修改/撤回/忽略邀请实际持W(S)+邀请U，接受和workspace删除持W(N)，账户停用还可不锁该工作区而删除发给它的邀请。预览因此必须在W(S)后持固定邀请行S，直到图片签发事务提交；只持W(S)不足以保护邀请有效性。先撤销提交则锁下重读404，先锁定有效邀请并签发则撤销等提交。新ShareInvitationPreview固定查询只锁邀请表，不改任何既有写用例的锁型/顺序；错token仍在全部数据库访问前拒绝。被忽略但未删除的链接保持M3现有预览语义。复制、猜测UUID、错误token、指向另一workspace的伪造请求均不能得到图片。

所有签发读取都使用短写事务（只行锁、无DML），不能把readonly snapshot当成可线性签发的替代。列表先预读候选IDs，在锁下只返回仍可见且属于候选集合的行；期间新加入的workspace/project留到下一次请求，不追新锁。保持现有名称/ID排序、整集合响应及归档过滤，不截前N条；不把已不可见行残留在DTO。单目标不可见仍原owner404。项目ID升序共享锁在owner自有SQL执行；读操作永不升级为N。

普通元数据列表现有容量边界不擅自变化；多锁批次代价明确记入1000引用实测。写门失败统一沿bootstrap映射503 server_busy/Retry-After:1，不在owner新增错误包装。所有读/写事务取消或错误后放回连接与gate。零业务DML指图片与owner业务记录，不改变认证中既有PAT last_used机制；相关断言需明确观察范围。

T12冷统计验收发现既有ListProjects在新建大量项目时会重复扫描成员/偏好组合。所属SQL将个人偏好连接明确锚定当前project与user，并以有效成员行存在为前提；非成员的sort_order仍null，权限、排序、归档和整集合语义不变。不通过预先ANALYZE、延长请求期限或截断列表掩盖此问题；1000/1001引用在未经手工统计更新的真实API夹具上验证。

## 5. 必须有区分力的验证

- source候选非本人/跨scope/已删/过期和不存在逐值同形，owner403不被候选状态改写；两图混合隐藏/未上传，排序反转仍先隐藏。
- 同一current重复不更新资产；管理员B替换A上传的logo可行，B挪用A未绑定件拒绝；同user avatar拿来cover拒绝。
- 7天exact/正1微秒边界；signed upload过期但7天内已完成件仍可绑定，不能按运行时钟误判。
- CreateProject在默认状态/Apply/签名/commit注入失败，所有业务行和候选scope/bound/audit回滚；双请求最终仅一个project成功且没有另一套成员/state。
- 每个表中列出的响应操作实际产生URL，读取真实内容、ExpiresAt匹配，PG触发器/审计证明GET不写。
- 密码reset/账户停用、workspace成员撤权、project私有化/归档/删除、邀请删除/accepted双方先后，以真实PID/阻塞观察决定签发是否成功；正常请求无40P01。
- profile替换与其他用户在member.list读头像并发，证明不在W后锁目标账号；旧件失效不签、已有URL边界诚实。
- 大列表中预读后新添/撤权对象，最终仅锁住且仍可见集合；1000/1001引用分块有序、重复去重、无N+1，取消后池与gate可再次使用。
