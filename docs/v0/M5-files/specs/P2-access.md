# M5 / P2：宿主权限与锁序

> P2 预检子合同；与主 spec、protected-purge、recovery 一起评审。这里定义新增接口，不声称已有实现。局部签名和锁粒度由主控裁定，既有角色规则及 HiddenFrom 不变。

## 1. 准备范围与授权矩阵

每次申请都产生 `bound_at = NULL` 的准备件。P2 不提供绑定 API；真实宿主引用由 P4/P5 接入，不能靠 complete 自动公开。即使已有宿主 ID，未绑定件仍仅申请者可读写，且申请者仍须通过宿主当下权限。所有用途的 `user_id` 都是不可变申请者，服务端填 actor.UserID；不是任意可转让 owner。

请求使用一个严格 `scope` 对象，`kind` 为 user/workspace/project/new_project/issue/comment/new_comment/draft，`id` 为该实体 UUID；user 不接受 id，服务端取当前用户。new_project.id 指已有工作区，new_comment.id 指已有工作项。不能同时传重复父 ID；服务端由 owner 查询完整父链。

| 用途 | 允许 scope | 申请/确认/复制目标 | 元数据/内容/复制源 | 删除准备件 |
|---|---|---|---|---|
| USER_AVATAR / USER_COVER | user | 当前活账号 | 仅申请者 | 仅申请者 |
| WORKSPACE_LOGO | workspace | workspace.update | workspace.read | workspace.update |
| PROJECT_COVER | project / new_project | project.update / project.create | project.read / project.create | 与申请相同 |
| ISSUE_ATTACHMENT / ISSUE_DESCRIPTION | issue | issue.update | issue.read | issue.update |
| COMMENT_DESCRIPTION / COMMENT_ATTACHMENT | comment / new_comment | issue_comment.write + commentCheck(allowAdmin=false) / issue_comment.create | issue.read + commentTargetCheck / issue.read | commentCheck(allowAdmin=true) / issue_comment.create |
| DRAFT_ISSUE_DESCRIPTION / DRAFT_ISSUE_ATTACHMENT | draft | DraftLocks.read（作者+工作区） | 同左 | 同左 |

表中动作不是 asset 的新 Action。workspace/project 在自己的 app 复用 decide，issue 在自己的 app 复用 Locks.seeAs/decide、commentCheck/targetCheck、DraftLocks.read。asset 不接收 Authorizer，也不查询成员/项目/issue 表。具名 actionless 登记说明委托，原动作目录等式及负例不变。

优先级：结构 400/字段 422；认证 401；指名准备件不是申请者/资产已删/宿主不可见或不存在，一律 asset.not_found；可见后角色/评论作者不足 403；写权限通过后项目归档 → 工作项归档 → 状态冲突。owner 的“资源不存在”只在 bootstrap 的具名转换映射为 asset.not_found，基础设施错误原样返回，不能把任何 error 都变 404。预检重放同一可见性顺序；归档检查在锁下最后判定。

草稿复用现有作者私有规则，已归档但未删除的项目不额外禁止草稿保存/准备；项目删除则拒绝。发布时再判目标项目创建权限属于 P5。更换草稿项目后，P2 旧准备件的项目快照不再匹配，返回 scope_changed；不在 asset 锁后追新项目。P5 随 UpdateDraft/PublishDraft 的同事务迁移处理仍是后续责任。

P2 公开 API 不能制造 bound 件，因此成员读取已绑定头像/封面、附件列表、历史描述图片读取分别在 P4/P5 验收；不能用测试 SQL 的 bound 行冒充这些 API 已上线。当前矩阵覆盖每种准备范围及在该范围撤权/删除的行为。

## 2. 消费方接口及值

以下值位于 asset/app/access_ports.go；Go 字段为明确值，不把 owner 的 domain 整体传进 asset。bootstrap 为各 owner 做逐字段转换。

```go
type AccessIntent uint8 // ReadAsset, PrepareAsset, RemoveAsset
type HostRequest struct { Scope domain.Scope; Intent AccessIntent }
type Host struct {
    Scope domain.Scope
    WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID
}
type HostAccess interface {
    Inspect(context.Context, shared.Actor, HostRequest) (Host, error)
    Check(context.Context, shared.Actor, HostRequest, Host) error
}
type WorkspaceLocks interface { ShareMany(context.Context, []uuid.UUID) ([]uuid.UUID, error) }
type ProjectLocks interface { LockMany(context.Context, []uuid.UUID) ([]uuid.UUID, error) }
type DraftLocks interface { LockMany(context.Context, []uuid.UUID) ([]uuid.UUID, error) }
type CallerLock interface { LockCaller(context.Context, shared.Actor, time.Time) error }
type WriteGate interface { Enter(context.Context, ...uuid.UUID) (func(), error) }
```

`Inspect` 只读，回答完整父链并执行身份/角色/作者预检，不取得连接之外的长寿命资源。`Check` 在调用者已取得父锁后重新执行同一权限判定，核对全部父 ID 与预读 Host 相同，然后检查写归档限制；它不另开事务、不进写门、不取 W/P/asset 锁。返回不同范围为 scope_changed，已失去可见性仍优先 404。方法参数不能由 HTTP 原样赋成“已授权”标记。

owner 外观在各自 module_assets.go 暴露 `AssetAccess()`；issue/app/asset_access.go 内组合现有 Locks 和 DraftLocks（使用原方法，禁止复制 HiddenFrom/成员角色分支）。workspace/app/asset_access.go、project/app/asset_access.go 使用各自已存在的 decide。各 owner 定义自己的 Request/Host 值，只包含自己拥有的范围；bootstrap/asset_access.go 按 scope kind 路由到这三个具名能力。user 分支只检当前账号，使用已注入 CallerLock，不查询别人的 profile。

workspace.Provide 增加 `ShareAssetWorkspaces(ctx, ids)`，固定 sqlc 语句 `ORDER BY id FOR SHARE`；project 复用 `IssueProjects.LockIssueProjects`（N 锁），不新造第二套项目门；issue.Provide 增加 `LockAssetDrafts(ctx, ids)`（排序 U 锁，包含已删行以供锁下判定），每次一条查询。输入去重、UUID 升序；端口报告实际取得的 ID，缺项由 owner 重验产生同一 404。所有方法使用 ctx 现有事务。

## 3. 一条资产编排路径

asset/app/access.go 内 `Access.Within(ctx, actor, requests, assetIDs, fn)` 是本模块内部方法；fn 不跨模块，asset 自己完成事务内写。其他模块只交小接口，不接收任意事务闭包。

1. 资产预读（若指名资产），检查申请者/删除；按语义顺序 Inspect（复制源先于目标），完整回答 404/403。收集全部 W/P/D/asset ID，去重排序。
2. 在无连接、无锁时一次进入全部项目写门。bootstrap 把**同一实例**同时传 issue 与 asset；不得各 new 一把只保护自己的 gate。
3. 开短写事务。当前账号 N 锁与 credential 检查为第一把锁（认证时刻沿用 identity 注入的业务 Clock，在调用 CallerLock 前读；资产审计时刻另在最终锁后读，签名时刻才用 runtime）。然后排序 W(S) → 一次排序 P(N) → 排序 D(U) → 排序 asset(U)。issue/comment 写由 P(N) 序列化，不额外倒序取 issue 锁。
4. 父锁后、asset 锁前调用 Check。asset 锁后重读资产，核对申请者、用途、绑定/删除、全部父范围、metadata revision；变化则冲突，不再追父。最后一把锁之后读业务时钟，只为真实写生成同一 At/By。
5. 在事务内生成 CPU-only 签名，成功提交后才返回 URL。local HMAC 和后续静态配置 S3 presign 不执行网络 I/O；任何配置会触发网络取凭据的方式均不在此合同中。Inspect/Open/Seal/Delete/List 全在事务外。
6. 任一错误/取消/panic 均结束事务，再放回所有门；等待门时无连接，外部操作时无连接、父锁、资产锁及门。

`GET /assets/{id}` 使用既有只读快照事务：资产+owner 授权在同一快照，不签发 capability。`GET .../content` 使用上述短锁事务，生成 URL 但不 UPDATE 资产、不改使用次数或审计；撤权先提交则不能新签发，已先签发 URL 仍按 M5 bearer 边界有效。

complete 的候选准备和最终提交各走上述完整路径，中间释放全部资源；duplicate 的两段都一次收集源与目标。最终提交不仅检查目标，还重验源可读、source final key/摘要仍相同；源失效不发布目标，保留目标候选待回收。source 与 target 任何 ID 次序都遵守同一物理排序，拒绝顺序仍源先目标后。

公开操作不得在已有任意 owner 事务中调用 Access.Within。P4/P5 的绑定小端口将只锁资产自身，不回取账户/父锁；在那些阶段实现，不提前加无消费者代码。

## 4. 构造与同步删除

bootstrap 明确接线顺序：

1. 创建 local storage 资源并持有唯一 close；此时不创建不完整 asset.Module。
2. Provide identity/workspace/project/issue/asset 自有存储端口，构造 authorizer 和同一 write gate。
3. NewInserter；构造 issue 原订阅者 + asset 自有同步删除/父范围任务订阅者，再一次创建 immutable dispatcher。
4. issue.New（已有 Locks + 新 AssetAccess；DeleteDraft 注入 asset 的同步删除小端口），project.New，workspace.New，identity.New。
5. asset.New 接 storage、CallerLock、owner access、小锁端口、gate、Tx/ReadTx/JobTx/Clock/RuntimeNow；一次完成业务与 raw 端点。无 New 后补注入。
6. 汇集 jobs、注册路由；失败和 close 仍 HTTP → jobs → storage → DB，覆盖后半构造失败。

实际评论事件名是 `issue.comment_deleted`，不是 `comment.deleted`。issue.deleted 与 issue.comment_deleted 的资产订阅者在原事务锁定并软删对应资产，沿用 envelope 的 At/By。DeleteDraft 没有删除事件，在 draft_writes.go 的真正删除路径声明消费方 `DraftAssets.DeleteDraftAssets(ctx, id, by, at) error`；PublishDraft 内部软删不得调用它。project.deleted/workspace.deleted 原事务只投递资产范围任务，保持既有父同步删除不遍历全部资产。

## 5. 必须证明的交错

| 对手 | 防线与可观察结果 |
|---|---|
| password reset / deactivation | 账户首锁；先撤销则 401，先签发则其事务结束后撤销，绝无 W/P→账户等待环 |
| workspace 删除/成员撤权 | W(S) 与 W(N)；等待后重判，不用旧目录事实 |
| project 私有化/成员撤权/删除/归档 | P(N) 与既有全部写同序；先变化则无新授权或明确归档拒绝 |
| issue/comment 删除/归档 | 同一 P(N)；同步 asset 标记或最终提交重验保证无迟到发布 |
| draft 改项目/删除/发布 | W → 排序 P → D；观察范围变化不追锁，双项目正反次序均测 |
| 双 complete / copy-vs-delete | 资产升序锁与 metadata revision；唯一 final，外部残留有记录 |
| 周期回收 | 仅持 asset 锁，不能在其后查询带锁 owner；读父事实在前、锁下核当前 scope，再标记 |
| 物理父 CASCADE | 受保护 FK 保留恢复记录；真实 40P01/连接错必须重试，不能记为 blocked |

组合测试用 pgtest 的真实请求 PID、blocker 与查询身份观察，不靠 sleep。至少三种项目 ID 次序、同一项目去重、一个拒绝来源不进门、持锁时撤权的两个方向、三阶段存储阻塞时池余量、普通 issue 写使用同一 gate 都有区分数据。执行时 -race -count=5，正常写不允许 40P01；物理 purge 与迟到软删的 PostgreSQL 重试边界沿用 M4。
