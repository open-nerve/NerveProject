# M5/P7a：草稿准备件恢复协议

> 2026-11-05：P7a T1–T7与A1–A6全部完成。正式开发基线`963636757fe5c5a8c10d5781b43d1481ba6f32a2`，最终候选`756b90314009f00e487483729b5d19f76dbeb110`经一次整分支deep-review及局部修复后，分支CI41859931318全绿；noff main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`的CI41869045846全绿。原件归档/正常清理和P7b正式预检完成，P7b暂不实施。证据见[阶段记录§13](../reviews/P7a-draft-preparations-review.md#13-t7-集成归档与接续)。

## 1. 目标、范围与选择

草稿申请上传成功但回答丢失，或 duplicate 提交 pending 目标后存储失败，会留下作者不知道 ID 的准备件。P5 要求全部活资产完成才发布，现有附件列表只返回已绑定件，作者无法当场发现并明确丢弃这些记录。[原预检 re-46](../../../reviews/re-46-M5P7预检.md)保留发现时证据和两项问题；本合同落实已批准的恢复方向。

新增一个只读、作者私有、有界分页的准备件入口；复用现有资产 DELETE、草稿发布、存储恢复及清理协议。不添加表、依赖、批量删除、自动放弃上传或新的 owner 授权端口。P7a 交付 API 和真实 HTTP 恢复证明；P7b 才交付发现/选择/丢弃的页面、编辑器及旧文件调用链替换，P7a 通过不等于 F10 页面已完成。

方案 A 的收益是同时覆盖未知 ID、刷新、多标签及明确失败复制，不放宽数据完整性。代价是增加一个查询和后续恢复 UI，分页中并发新增时仍需重新读取。备选 B 在发布时放弃 pending 会改变 P5 的事务/数据语义，未获选择；仅保存已知 ID 或等七天 GC 不能覆盖根因。

## 2. API 与可见性

`GET /api/v0/drafts/{draft_id}/asset-preparations`，operationId `listDraftAssetPreparations`，归属 `api/modules/asset.yaml` 及 asset HTTP adapter。业务路径在 draft 下不转移表的所有权，不改 issue 模块路由。参数复用既有 `limit`、`cursor`；新 schema `DraftAssetPreparationPage`：`{data: Asset[], next_cursor: string|null}`，空页为 `[]`，只复用已存在的公开 Asset DTO，不公开内部 metadata、key、候选对象、upload/content URL 或令牌。

返回当前 actor 在该活草稿、当前工作区/项目范围内，`scope_kind=draft`、两种 `DRAFT_ISSUE_*` 用途、`user_id=actor` 且 `created_by_id=actor`、`bound_at IS NULL`、`deleted_at IS NULL` 的资产。pending 与 uploaded 都包括；uploaded 表示已完成但尚未绑定的描述图片等，不能误称都是废文件。已绑定附件、当前或历史正文已绑定图片、其他草稿/作者、迁出为 issue 的资产均不返回。`is_bound` 恒为 false，状态仍由现有 Asset DTO 转换，不能手工构造另一套状态。

作者须仍满足现有 DraftLocks 的工作区成员/活宿主条件；管理员没有查看他人准备件的特权。无项目草稿合法；草稿关联项目的存在性沿 owner 现有规则，不额外要求作者加入该项目，不将项目可见性当工作项权限。已删除草稿/工作区/项目、失去工作区资格、非作者与随机不存在 UUID 均为 `404 asset.not_found`，不泄露文件名或草稿存在性。无有效会话为 401。

**局部裁定 R1：** 既有 DraftLocks 先判断工作区角色再判断作者，live guest 可先得到 403。本新增私有查询将 **Host.Inspect 这一次调用** 返回的 `shared.KindForbidden` 映为 `asset.not_found`，保留 401、取消及意外/数据库错误；不修改全局 `assetHostError`、旧 draft API 或其他资产写的 403。原因是本列表没有可公开的部分，拒绝时不应提供存在性探针；代价是此端点不区分角色不足与不存在，须明确做同形回归。不得泛化为“所有 error 都当 404”。

非法 UUID/非整数 limit/错误 cursor 为既有 400；limit 不在 1–100 为既有 422。参数失败在任何资源查询前处理且不包含宿主数据；默认 limit 50，最大 100。查询只使用标准认证/Problem/no-store 中间件，不增加存储网络访问、写锁、事件、任务、审计更新或使用计数。

全零UUID可以解析但不能代表真实草稿；认证及分页/cursor校验后直接返回`404 asset.not_found`，不打开事务或进入owner。无会话仍优先401，错误分页参数仍400/422；不修改任何旧入口或把真实数据库错误改为404。

## 3. 排序、游标和快照

UUID 升序，SQL `id > after_id ORDER BY id LIMIT limit+1`，多取一行仅判断还有下一页。游标为 `shared.EncodeCursor` 的规范 envelope，payload 固定 `{kind:"draft_asset_preparations",draft_id:<UUID>,after:<UUID>}`；解析同时核 kind、请求 draft_id、非零 after，拒绝其他列表、跨草稿、非规范编码、重复/未知字段。没有游标时 after 为零 UUID。游标不是授权凭据，每一页重验作者和宿主；修改合法 payload 只能改变起点，不能扩大范围。

每页在既有 `ReadTx` 的 repeatable-read/read-only 快照内，先 Host.Inspect，再 asset SQL。授权/宿主范围与数据必须来自同一 ctx/事务，不以两次独立查询拼出混合时点，也不通过跨模块 JOIN 查作者。只读列表与 concurrent complete/delete/move/publish 不互锁；在快照之后提交的变化允许到下一次读取才可见，任何后续 DELETE/publish 均按现有写锁重新授权/验状态。不得宣称列表为删除许可或发布预留锁。

跨页不共享数据库快照，游标之前的新记录可能要从首屏刷新才能看到；UI 不能把一次分页耗尽当未来永无 pending。分页耗尽与新申请交错仍由发布拒绝兜底。列表看见旧unbound、操作时已绑定时，**描述图片**仍受既有DELETE的 `asset.bound` 保护；**普通附件**按既有授权仍可DELETE，选中的pending附件在另一标签页完成后也可能被明确丢弃。这是同一ID原删除合同，不新增条件删除/状态预留，不能承诺所有已绑定项均拒删；不得改为删除正文或宿主。

## 4. 职责、接口与文件责任

| 层 | 责任与预计文件 |
|---|---|
| domain | `asset/domain/draft_preparations.go`：纯游标编码/校验，复用 shared cursor，不 import issue/HTTP/SQL |
| app | `asset/app/draft_preparations.go`：`ListDraftPreparations.Execute(ctx,draftID,limit,cursor)`；声明只读宿主与行查询小接口，RequireActor→参数→ReadTx→Inspect→SQL→page |
| postgres | `queries/draft_preparations.sql`、`draft_preparations.go`：只查 file_assets，空 project 用 IS NOT DISTINCT FROM；复用 Store.queries(ctx)、生成模型与 decodeAsset |
| HTTP | `asset/adapter/http/draft_preparations.go`、handler.go：DTO/参数/Problem，复用 assetDTO，不重写授权 |
| composition | `asset/module.go`：沿已有 d.ReadTx、d.Access.Hosts、d.Ports.store 注入；bootstrap 原 HostAccess 已实现只读接口，无需第二份宿主适配 |
| 契约/生成物 | `api/modules/asset.yaml`、api/dist、asset/adapter/http/gen、web/packages/api-client/src/schema.gen.ts：通过make gen生成，禁止手改；api-client/src/index.ts已有export type *，不新增手写类型别名 |

app 消费接口固定如下，名字可在实施时与邻域一致地调整，但含义和边界不变：

```go
type PreparationHostReader interface {
    Inspect(context.Context, shared.Actor, HostRequest) (Host, error)
}
type DraftPreparationQuery struct {
    ActorID uuid.UUID
    Host Host
    After uuid.UUID
    Limit int // 已验证的 limit+1，最多101
}
type DraftPreparationReader interface {
    ListDraftPreparations(context.Context, DraftPreparationQuery) ([]domain.Asset, error)
}
type DraftPreparationPage struct {
    Data []domain.Asset
    NextCursor *string
}
// Execute(ctx context.Context, draftID uuid.UUID, limit *int, cursor *string)
//     (DraftPreparationPage, error)
```

SQL 同时约束 actor、workspace、null-safe project、draft ID、scope/purpose、unbound/live；issue/comment 外键应为空，避免把不一致父链变成可见结果。现有 draft 外键索引足以作为起点，不先增 schema/index；实际查询若出现不可接受成本，先保留 EXPLAIN/行数证据再评估。读出来的 domain.Asset 仍走既有 codec，损坏 metadata/数据库错误不能伪造空列表。这个用例依赖只读能力，不依赖完整写锁/存储接口，符合消费方小接口和唯一授权实现的原则。

## 5. 恢复协议及 P7b 接收边界

作者显式打开恢复列表，逐项辨认文件名、状态、创建时间/大小/用途，明确选择并确认丢弃后调用已有 `DELETE /assets/{id}`。不根据文件名推断是否安全、不按 pending 自动全删、不在取消/卸载/刷新时 DELETE；同一作者另一标签页的在途件也不能擅自删。已知 ID 取消和未知 ID 失败共用这个恢复路径。

DELETE 204 后仅刷新列表/正文状态，再由用户原发布动作重验；未知 DELETE 结果先只读核验，不自动重发写。列表里消失可能已绑定、迁移或被删，不能单凭缺席宣称“删除成功”。已绑定描述图拒删，普通附件仍沿原DELETE；P7b确认须说明所选ID可能正在另一标签页使用或已完成，不能用“只删未绑定”作保证。新pending仍可能出现，发布409后提示重读，不能无限自动清理/重试。已有上传能力的迟到PUT可能写入存储但不能恢复被软删资产；保留授权期限、candidate、NO ACTION与原清理依据。

P7a 的真实 HTTP 测试必须按客户端不知道新 ID 的视角：准备/复制写已经发生，但调用侧丢弃回答或收到存储错误；仅从新查询发现目标，指名删除后再发布成功。不得用 SQL 找出 ID 后 DELETE 冒称恢复路径已打通。测试内部可以用数据库只读断言确认无泄漏/原子性，不能绕过业务 API 创建前置。

## 6. 验收与任务映射

- [x] A1 API/生成类型/严格参数及无敏感字段正确；页大小和规范游标含跨草稿拒绝（T1/T2）。
- [x] A2 作者、管理员非作者、member/guest/退出/父删除、不存在同形、null 项目、邻域隔离及现有 draft/asset 错误顺序保持（T2/T4）。
- [x] A3 真实 prepare 失去回答、duplicate 目标已建后失败，通过列表发现未知 ID、明确 DELETE 后发布；重开读取、同名文件不混同、已绑定历史图受保护（T3）。
- [x] A4 分页51/101+完整性、只读快照、分页间新增/删除、complete/delete/move/publish 两顺序及取消释放，无锁序/事件/计数回归（T1/T4）。
- [x] A5 必要行为 RED/GREEN/变异与恢复、全部七门禁、一次整分支 deep-review 修复和准确最终分支 CI（T5/T6）。
- [x] A6 --no-ff、准确 main CI、证据全量归档/正常清理；P7b 正式基线 spec/plan 预检，保留全部页面与旧链删除责任（T7）。

证据入口：[阶段记录](../reviews/P7a-draft-preparations-review.md)。预检记录保持为历史证据；产品验证按各 Task 新增，不以既有结果替代。P6 历史通过结果不能替代本端点测试，P7a 不领取 P7b 页面/浏览器交互完成。

## 7. 风险及未完成责任

新接口增加私有文件名的读取面，因此拒绝同形、每页重新授权和同快照是本阶段硬验收。准备件可能仍被另一个标签页使用，必须显式丢弃；不引入“孤立文件”自动判定。只读查询不能保证下一次 publish 必定成功，这是并发语义，不以更宽松发布合同掩盖。S3 能力到期前的限制、原七天/六十天清理、默认部署与全部 F1–F18 仍由既定阶段接续；handoffs 保持 open。
