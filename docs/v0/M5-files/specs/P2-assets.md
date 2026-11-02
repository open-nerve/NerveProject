# M5 / P2：资产用例与生命周期

> 状态：预检通过，T1尚未开始；见[完整预检记录](../../../reviews/re-34-M5P2预检.md)。负责人已批准清理方案 A，见[裁定](../reviews/P2-purge-contract-decision.md)。P2 基线是 P1 noff main `85dc4087a28570f59a62825f5dc861152e41c833`，准确 main CI 41471671630 三 job 全绿，P1 已归档清理。
> 上位：[M5 design](../M5-design.md) §3–§7、§10–§13。必读子合同：[权限与锁序](P2-access.md)、[对象恢复](P2-recovery.md)、[父行物理清理](P2-protected-purge.md)。它们是本 spec 的组成部分。

## 1. 目标与边界

建立 file_assets 与恢复状态，通过真实 local 字节完成申请、确认、读取签发、删除和复制。宿主模块掌握自己的权限和数据，asset 只持自己的 SQL；存储 I/O 在业务事务/连接/写门之外。首张表合入即接删除/清理和目录守卫，数据库不级联抹掉唯一对象恢复记录。

本段全部用途都产生仅申请者可见的准备件；complete 只完成实物验证，不自动绑定。用户/工作区/项目字段引用在 P4，工作项/评论/草稿内容引用、列表/计数/动态与发布迁移在 P5。P2 不以测试 SQL 的 bound 行冒称成员附件页面可用。

预检方式沿用 Codex 已采用的原型边界：真实 PostgreSQL 隔离探针验证重大清理语义，独立 builder check 验证完整合同，逐 Task TDD/变异与最终整分支审查验证实现；不宣称已在副本实现整段或逐 Task 完整重放。

## 2. 交付合同

### 2.1 文件与职责

| 所属 | 创建/修改位置（目录内按职责拆文件，每个手写文件约400行内） | 职责 |
|---|---|---|
| asset/domain | asset.go、scope.go、metadata.go、errors.go、既有 storage.go | 用途、准备件、状态与有界纯规则，不 import 解码器/数据库/SDK |
| asset/app | access_ports.go/access.go、asset_ports.go、prepare.go、complete.go、read.go、duplicate.go、delete.go、cleanup*.go、purge.go | 消费方小端口与唯一授权/事务编排、对象恢复 |
| asset/adapter | postgres/{queries,gen}、content/validate.go、local/list.go、local/temporary.go、http/业务 handler 与 gen、river/清理任务 | 自有 sqlc、有限图片校验、真实清单、DTO/任务转换 |
| owner | issue/project/workspace 的 app/asset_access.go 与 module_assets.go；identity 复用 CredentialLock；各 owner 的 purge adapter/query | 复用原权限、父范围与批量墓碑事实，不跨模块 SQL |
| bootstrap | assets.go、asset_access.go、asset_ports.go、purge.go、app.go、事件/路由/目录测试 | 单一构造根，原件可追踪的完整接线 |
| 契约/迁移 | api/modules/asset.yaml、server/migrations 下下一顺序迁移、sqlc scope/config、生成物 | 六个业务操作、具名约束、精确 API 与 SQL 所有权 |
| 验证 | bootstrap/assets_*_test.go、asset 各层测试、e2e/fixtures/storage.ts、stories/files/f1/f3/f13/f14/f15 | 真实字节、权限、回滚、并发、清理与边界 |

现有 gen/大目录文件按仓库既定例外，不借此添加手写上帝文件。当前迁移最大00033；T5新增00034_asset_file_assets.sql。执行前若出现新的main迁移须先核冲突，不覆盖已占编号。

### 2.2 表与 Plane 差异

以 tools/plane-schema/plane-v1.4.2-schema.sql: file_assets 为来源；读只读 Plane model/view，再逐项记录 plane-diff。保留 id/created_at/updated_at/created_by_id/updated_by_id、attributes、entity_type、workspace_id/project_id/issue_id/comment_id/draft_issue_id/user_id、size、is_uploaded/is_deleted/deleted_at/storage_metadata 的业务作用。

- size 为 bigint，1..104857600；name 是1..255 Unicode字符、不含路径分隔符/控制字符；声明 MIME 用 P1 canonical media type（无参数、无通配符）。API 超过实例 size_limit 不截小，返回 413。
- user_id 非空，为所有用途的不可变申请者（相对 Plane 可选 user 字段的明确用途调整）。审计字段可空；新增 deleted_by_id 保存系统 NULL/原删除者，bound_at、upload_expires_at、storage_revision。业务状态由 is_uploaded/deleted_at 推导；is_deleted 与 deleted_at CHECK 完全一致，不能双事实漂移。
- attributes 只由服务端构造，固定 name/content_type、已验证图片可带 width/height、已完成对象带 sha256；不接任意客户端 attributes。元数据 DTO 不暴露 storage_metadata、key、内部 ETag 或 runtime lease。
- 移除 page_id、external_id/external_source、entity_identifier、Django asset 路径字段与无消费方的 is_archived。最终 key 只在 metadata，不能另存一份可漂移路径。
- 保留全部10种用途。scope 与父列 CHECK：user=仅user_id；workspace=workspace_id；new_project=workspace_id且project_id空；project=workspace_id+project_id；issue=workspace_id+project_id+issue_id；comment=再加comment_id；new_comment=issue范围且comment_id空；draft=workspace_id+draft_issue_id且project_id可空。未列出的父列为 NULL。用独立 scope_kind 保存 new_project/new_comment，不能仅靠用途猜准备宿主；绑定转换由 P4/P5 原子完成。
- 业务父 FK `file_assets_{workspace,project,issue,comment,draft_issue,user}_fk` 即时、非 deferred NO ACTION；审计 FK 为 SET NULL。每个父列有无条件索引，不能只给活行建索引。deleted_at/created_at/ID 分页索引与 metadata 有界 CHECK、revision 非负一起验。
- SQL CHECK 必须真正拒绝缺字段/JSON null/未知 schema_version/非数组/第四候选，不能让 SQL NULL 的 CHECK 三值逻辑放过。Go 同时验证 key 所属资产、角色与唯一性，故障记录不对外开放。

存储状态的精确 shape、30s lease/30s宽限、两时钟及恢复见 recovery；它覆盖对 object metadata 的所有写。

### 2.3 API、DTO 与错误目录

所有业务操作都用 bearer、deny-by-default、bodyshape、Problem/no-store，普通 JSON 1MiB/15s 不改。raw /static/local 仍只在 api/storage-local.json，不混入 bearer OpenAPI。

```text
NewAsset = {purpose, scope:{kind,id?}, name, size, content_type}
Asset = {id,purpose,scope,name,size,content_type,state:pending|uploaded,
         is_bound:boolean,created_at,updated_at,width:integer|null,height:integer|null}
Upload = {method:"PUT",url,headers:map<string,string>,expires_at}
Content = {url,expires_at}
DuplicateAsset = {purpose,scope:{kind,id?},name?}
```

服务端从 owner 解析的完整父链不接受客户端重复声明。DTO scope 采用当下资产保存的 kind/id，不包含未授权其他父信息。sha256 用于服务端验证与证据，不在 API 强制公开；width/height 仅实际图片校验成功时有值。

| operationId / 路径 | 请求 | 成功 | 业务行为 |
|---|---|---|---|
| createAsset / POST /api/v0/assets | NewAsset | 201 {asset,upload} | 事务记录 staging 后返回签名；不会返回截小 size |
| completeAsset / POST /api/v0/assets/{id}/complete | 必须为空的 JSON对象 `{}` | 200 Asset | 准备候选→外部检查/Seal→重验晋升；重复返回同一资产 |
| getAsset / GET /api/v0/assets/{id} | 无 | 200 Asset | 同一只读快照授权；不返回上传 URL |
| resolveAssetContent / GET /api/v0/assets/{id}/content | 无 | 200 Content | 必须 uploaded/final active；UTC整点读期限；不写资产 |
| duplicateAsset / POST /api/v0/assets/{id}/duplicate | DuplicateAsset | 201 Asset | 源读取先于目标准备；新对象/新ID，成功后uploaded且未绑定 |
| deleteAsset / DELETE /api/v0/assets/{id} | 无 | 204 | 准备件软删，重复404；未来 bound 由owner规则，不能通用删头像/历史图片 |

duplicate 可跨用途，但目标重新执行大小/图片验证；非图片附件不能借复制变成头像。name 缺省沿用已授权源名。必须检查目标 scope 与 purpose 合法；不得先向可写目标复制后才验证源。没有公开任意 owner 更换/恢复接口。

具名错误一次加入 domain、x-problem-codes、HTTP映射、PROBLEM_MESSAGES、en/zh-CN errors.json，运行 make test-web。预期新码如下（实际共享状态码复用既有，不另造镜像）：

| 状态/码 | 精确条件 |
|---|---|
| 404 asset.not_found | 资产不存在/已删、准备件非申请者、宿主不存在/不可见；逐值相同 |
| 403 forbidden / 403 issue.not_author | 已可见但角色/评论作者不能写；代码核定于 issue/domain/errors.go |
| 409 asset.not_uploaded | content/duplicate 源尚未完成 |
| 409 asset.object_missing | 数据库准备件仍有效、staging 实物不存在（不是存储连接失败） |
| 409 asset.object_changed | ETag/字节在验证或封存间变化 |
| 409 asset.scope_changed | 预读后父范围/候选代次变化或候选 lease 已过 |
| 409 asset.bound | 通用删除不允许删除已绑定图片/字段引用 |
| 413 asset.too_large | 声明或实物超过本次允许大小 |
| 415 asset.type_unsupported | 图片用途声明不是允许的4种栅格 MIME |
| 422 validation_failed | 非法用途/范围/name/size格式等；沿用 shared.Invalid 字段顺序 |
| 422 asset.content_invalid | 实物大小/MIME不符、图片头/尺寸非法、超过头部上限 |
| 503 asset.busy | 未回收候选已3个；Retry-After:1 |
| 503 asset.storage_unavailable | 存储 I/O/校验读取取消之外的明确后端故障，不能映404 |
| 503 server_busy | 既有 gate/锁等待转换只在 bootstrap，Retry-After:1 |

既有归档码 409 project.archived / 409 issue.archived 由 owner 保持并在相应用途操作声明；一般401、bodyshape400、404路径/405/413/429等公共错误保持真实全程序目录。准确复用 forbidden / issue.not_author / project.archived / issue.archived，不能在 adapter 变更它们的状态或判定顺序。

### 2.4 图片与依赖

`asset/app.ContentValidator.Validate(ctx, reader, expectedSize, contentType, imageRequired) (ContentProof,error)` 返回完整 SHA256 与可选 Width/Height；adapter/content 实现。读取最多 expectedSize+1 字节、64KiB缓冲，直到EOF；同大小不同内容必须由摘要区分。只有七种非attachment用途要求图片；ISSUE_ATTACHMENT、COMMENT_ATTACHMENT、DRAFT_ISSUE_ATTACHMENT允许任意非空字节，三者始终强制attachment，不因声明image/png就inline。

图片只支持 JPEG/PNG/WebP/GIF，校验头部格式与声明一致，DecodeConfig 最多1MiB输入，每边1..16384、像素乘积≤40,000,000（防溢出先检查后乘）。探测允许从有界头缓存重读，然后完整字节流算摘要；不做全图像解码，不把头检查称为验证所有像素/帧。损坏后段可能经头检查但浏览器无法显示，这是明确限制；nosniff和仅栅格inline保持。新增依赖仅设计点名的 `golang.org/x/image v0.47.0` WebP；必要间接 x/sys v0.49.0、x/text v0.43.0 在 go.mod/sum 中解释并回归。无其他新增库、无全局安装。

已核[官方 WebP](https://pkg.go.dev/golang.org/x/image@v0.47.0/webp)及[该版本 go.mod](https://github.com/golang/image/blob/v0.47.0/go.mod)；仓库Go1.27满足其Go1.26底线。依赖尚未安装，所有测试均待实现。

### 2.5 删除、清理与观察

严格执行三个子合同；特别是实际评论事件名 issue.comment_deleted、DeleteDraft 与 PublishDraft 分离、共享write gate、父全部传递祖先、Scanned/Deleted分开、metadata 未清空不得物理删行。既有 cleanup.purge_deleted 改为单页 owner + River output 继续，保留60天条件/子先父顺序与真实 PostgreSQL 重试；不增业务表或通用 outbox。

日志仅 request_id/asset_id/阶段/字节/耗时/错误类别/blocked 数；不记签名query、私有文件名、正文或底层磁盘路径。配置集中files、未知键拒绝、启动边界/样例/遮盖同步，精确默认/范围见recovery。

## 3. 已批准选择与局部裁定

- **负责人A：**即时NO ACTION兜底、联合M4/M5清理。原CASCADE可丢外部恢复记录；代价是对象故障延后相关父墓碑。C（纯owner锁协议）可行但无数据库最后防线，未采用；原证据/比较保留。
- **主控：**保留asset具名actionless登记，因为所有角色判定委托owner既有Actions。没有增加一套asset.*角色规则，没有放宽目录谓词；代价是必须用完整API→owner矩阵与绕过授权变异证明委托。
- **主控：**P2全部是准备件，绑定入口保持P4/P5；可单独证明真实文件/权限/恢复，不抢跑宿主字段/动态。代价是本段不能宣称完整附件功能已交付。
- **主控：**metadata有界、30s候选lease、10s外部段、24h未知残留宽限；恢复与局限见recovery§7。父整批失败后逐候选，代价见protected-purge§7。
- **主控：**新增权限门面复用原内部判定，排序一次锁范围；不跨模块传闭包。账户首锁及同一gate让撤权/签发可线性判定；代价为资产能力签发短暂序列化同账号/父范围，容量测试记录。

## 4. 验收标准

以下全部尚未实现/验收，不能提前勾选。

- [ ] A1 六个API、10种用途、严格DTO/错误/两语/生成物与真实路由目录一致。
- [ ] A2 真实 PUT→complete→content→GET 字节与摘要一致，重复complete只有一个final；旧PUT不能改已发布字节。
- [ ] A3 权限矩阵、非本人准备件、guest自己/他人及管理员兼guest、私有/撤权/归档/删除；不可见与不存在逐值相同。
- [ ] A4 双阶段授权、真实PID锁序、同一gate、正反项目次序、范围改变不追锁；正常交错race五遍无40P01。
- [ ] A5 阻塞存储时不占连接/锁/gate；候选先commit、外部成功DB失败、lease迟到与并发败选都可恢复。
- [ ] A6 复制源/目标双向拒绝，独立对象、目标内容再验证、删除先后及重复请求边界。
- [ ] A7 同步issue/comment/真正DeleteDraft、父范围任务原事务投递与回滚、原At/By、邻域实物保留；首表删除图和furnishIssueRows随阶段延伸。
- [ ] A8 7天/严格60天与runtime安全期限分开；调小配置/迟到PUT/候选/未知对象/tmp可回收，无误删live final或其他世界。
- [ ] A9 受保护父全集/直接与传递FK、精确错误身份、阻塞页前进、恢复后收敛、未知错误重试；原目录负例保留。
- [ ] A10 有限批次/时间/内存、真实River游标snooze/重启/重试；记录10,000目录成本，不冒称任意规模。
- [ ] A11 图片4格式/头部和尺寸上限、完整流式hash与普通附件download安全，依赖变化和资源取消有证据。
- [ ] A12 七项本地门禁、deep-review三视角/修复复审、准确分支/main CI、noff/证据归档清理；后续P3只在前置事实满足后准备。

具体测试/变异分配见plan；原规划探针不替代永久模块/完整程序验收。

## 5. 后续责任

P3：S3真实兼容协议/分页token/后到对象/CORS/CSP/容量；P4：真实反向引用与成员资料读取/稳定封面；P5：附件/图片绑定、计数/事件、草稿换项目和发布原子迁移；P6/P7：会话store、UI、浏览器上传和三浏览器；P8：规模/失败演练、交接与warning收敛。M5输入handoff保持open，只追加已验收处理结果。

P1保留的C12菜单/F6焦点Minor未修，后续UI三浏览器验收继续跟踪，不能混为W11或在本段暗改共享基础库。

## 6. 风险与完成门

1. NO ACTION保护正确性不等于回收必然成功；存储持续失败会保留父行，需要错误计数与可恢复游标。
2. 既有单页purge改造是已批准A的工程范围，所有owner旧回归、真实目录和任务重启必须联合验证，不把接口编译通过当并发证明。
3. 图片仅受限头部验证；local清单O(N)/页；URL是短期bearer，已发出地址不保证即时撤回。这些限制不可在最终描述省略。
4. P2完整spec的builder check及plan自检已通过，当前尚未启动T1、未改迁移/产品/依赖；按连续授权推进。只有新的重大风险/关键分歧才提交上报，不能把文档预检当实现验证。

## 附录A：预检探针

2026-11-02 真实PostgreSQL隔离探针：原CASCADE guard允许丢记录留磁盘；NO ACTION保留恢复记录并被旧guard拒绝；SET NULL违反宿主CHECK。原M4两回归+探针3.207s通过；第一轮错误SQLSTATE假定和shell失败原件保留。

2026-11-03 补充探针：传递CASCADE/自引用保留叶子23503身份；整批失败后Scanned3/Deleted1/Blocked2/After3，移除恢复记录后收敛（2.062s）；真实schema/table/constraint精确核对（3.315s）。时间按本机Asia/Shanghai记录；原件时间戳保留。源执行后移出产品目录。完整列表在 `.superpowers/sdd/P2-assets`，来源见裁定与protected-purge；不是P2产品通过报告。
