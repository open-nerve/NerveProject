# M5 / P1：本地存储与签名传输

> 状态：已完成集成、main CI 与归档清理；见[预检记录](../../../reviews/re-31-M5设计与P1预检.md)。上位设计：[M5-design](../M5-design.md)。
> T9 修订已获授权：采用 `/static/local/{key}`，按[独立复核](../../../reviews/re-32-M5文件路由方案A复核.md)补路径规范化与开发代理日志边界；见[最终裁定](../reviews/P1-storage-namespace-decision.md#6-最终裁定2026-11-02)。T1–T9 实现和七项本地门禁已通过；整审及局部修复复审完成，停读下载夹具已在 Linux 通过；整包预算修订后的准确最终分支 CI 与 main CI 全绿，归档清理完成，见阶段 review §11。
> 文档基线：`3db22f06ad9e82a483c05c8078801af00a5cae02`；实施基线为 `fb3e788b7be33467018385c9a67d515e51b79dad`；[准确 main CI](https://github.com/open-nerve/NerveProject/actions/runs/41414200682) attempt 1 三作业全绿，设计证据已归档并核验，设计分支/worktree 已正常清理。

## 1. 交付边界

建立 asset 的存储值/小接口和 local 适配器，将受签名保护的原始 PUT/GET/HEAD 接入完整程序，验证真正的文件字节、MAC、路径、期限与限流。P2 的业务资产申请/确认将消费这些能力。

本段不建 `file_assets`，不发放面向用户的上传授权、不开放附件/图片页面、不加 S3 SDK，不动既有 Docker/容器和其他模块的业务权限。原始端点没有有效签名只能拒绝；没有业务申请端点不等于可用任意 bearer token 上传。

## 2. 确定合同

### 2.1 路径、对象与发布

- HTTP：`PUT /static/local/{key}`、`GET /static/local/{key}`、`HEAD /static/local/{key}`；另有 `/static` 和 `/static/` 的明确 404 Problem fallback，未知文件路径不能落入 SPA。涉及该路径树的 `..`、重复斜杠规范化在进入 ServeMux 前直接拒绝为 404 no-store Problem，不把签名 query 随重定向带到其他端点。
- `/static` 独占原始文件传输；M3 保留名单中的 static 从 reserved 移到 server，保留名字总集合不变，storage 仍可创建。构建/public 不得占 `/static`；普通 `/assets` 的 hash 文件、emojibase 与缺失 chunk 语义保持。旧 `/storage` 不保留文件别名或重定向。
- 非规范路径的归属按规范化过程逐段判断：中途进入再离开 `/static` 也拒绝；平台机制不写 asset 特判，普通页面/API 保持既有规范化。
- key：`u-<assetUUID的32位小写hex>-<objectUUID的32位小写hex>` 或 `o-...`。`u` 是上传暂存，`o` 是封存/复制结果；两段均非 nil UUID。文件名、客户端路径、编码斜杠和 `..` 不能成为 key。
- local 默认根目录 `./data/files`，仅内部包含 `objects/<key>/{data,meta.json}` 和 `tmp/<随机目录>`。目录 0700，文件 0600；已存在根目录必须是可用目录，不改变用户已有权限。配置/日志不得泄露绝对目录，测试用独占临时目录。
- 使用当前 Go 1.27.1 的 `os.Root` 约束所有对象访问；只接受普通文件/目录，拒绝符号链接逃逸和非普通字节文件。部署目录是服务进程独占的受信任目录，不把 os.Root 宣称为可抵御拥有该目录写权的系统管理员。
- 上传把字节和具名元数据写入独立临时目录，长度/类型参数校验通过后原子 rename。完整对象不覆盖；local 同 key 的第二次 PUT 为 409。取消、超限、写入失败清理本次临时目录。进程崩溃遗留 tmp 不可读取或冒充完成对象，P2 接入有宽限的回收。
- `Put` 不取得调用方输入流的所有权：ctx 取消后停止后续处理，阻塞 I/O 由输入流拥有者解除；HTTP 场景由 T6 的 socket 期限承担，不能用脱离请求的读 goroutine 伪造取消成功。每次操作持有自己的根句柄，失败清理不借用已经关闭的 Store 根句柄；Open 的输出 reader 由调用方关闭，并随其 ctx/Store 生命周期结束读取。
- `meta.json` 有版本、key、size、content_type、ETag、stored_at；ETag 是真实字节的 SHA-256（带 HTTP 强 ETag 引号）。不是用户可写 JSON，不能仅信任请求头声称已经完整。
- `Seal` 在打开的源对象上验证 ETag，复制到调用方已登记、仅供这一次操作的全新 `o` key；复制时重新计算字节 SHA-256，和预期 ETag 不同或字节数不符就不发布。不能只比较旧 meta.json 后盲拷贝已变化的 data。候选不可覆盖，返回的 size/type 与源一致。公开 PUT 永远不能写 `o`。S3 以后可以覆写 staging，共同合同不承诺重复 PUT 成功。

### 2.2 Go 消费接口

值在 `asset/domain/storage.go`，接口在 `asset/app/storage_ports.go`，不暴露 HTTP/pgx/SDK 类型。对象 key 的解析/生成和输入校验用具名构造函数，不在各 handler 复制 regex。

```go
type ObjectKey string
type UploadGrant struct {
    Key ObjectKey
    Size int64
    ContentType string
    ExpiresAt time.Time
}
type ReadGrant struct {
    Key ObjectKey
    Method string // GET 或 HEAD，分别签名
    Name, ContentType string
    Inline bool
    ExpiresAt time.Time
}
type SignedRequest struct {
    Method, URL string
    Headers map[string]string
    ExpiresAt time.Time
}
type ObjectInfo struct {
    Key ObjectKey
    Size int64
    ContentType, ETag string
    ModifiedAt time.Time
}
```

app 的小接口分别为：

```go
type UploadSigner interface { SignUpload(context.Context, domain.UploadGrant) (domain.SignedRequest, error) }
type ReadSigner interface { SignRead(context.Context, domain.ReadGrant) (domain.SignedRequest, error) }
type ObjectInspector interface { Inspect(context.Context, domain.ObjectKey) (domain.ObjectInfo, error) }
type ObjectReader interface { Open(context.Context, domain.ObjectKey, string) (io.ReadCloser, error) }
type ObjectSealer interface { Seal(context.Context, domain.ObjectKey, domain.ObjectKey, string) (domain.ObjectInfo, error) }
type ObjectDeleter interface { Delete(context.Context, domain.ObjectKey) error }
type MAC interface { Tag([]byte) [16]byte; Verify([]byte, [16]byte) bool }
```

`Open` 的第三参和 `Seal` 的第四参是预期 ETag；失配返回具名 `ErrObjectChanged`，缺失 `ErrObjectNotFound`，目标已存在 `ErrObjectExists`，其他 I/O 错误保留为后端故障但不泄露路径。调用者必须关闭 Open 的 reader；其 ctx 取消后读取结束。清单/对账接口在 P2 由真实清理消费者定义，本段不制造闲置万能 Storage 接口。

本地 HTTP 消费者另外声明 `OpenTransfer(context.Context, ObjectKey) (ObjectInfo, io.ReadSeekCloser, error)`，从同一次打开取得元数据和可定位句柄，以支持标准条件读取和单段 Range；该能力不加入 app 的流式 ObjectReader，未来 S3 也无需实现本地定位。句柄遵守相同的取消、Close 和独占所有权合同。

### 2.3 签名

- bootstrap 通过已加载的 `identity.Keys.MAC` 派生 `asset-upload`、`asset-read`，asset 只持有 app.MAC；不重复读密钥、不加密钥文件、不将 identity import 到 asset。
- 签名格式版本 1，规范化载荷固定字段顺序；URL query 只接受 `payload`、`tag` 各一次，拒绝未知项、重复项、非规范 base64url 和大于 4096 字节的 query。载荷绑定版本、用途、method、key、精确大小/类型或读取处置/名称、到期秒；URL path key 与签名 key 必须相等。
- v1 JSON 顺序固定为 `v/purpose/method/key/size/content_type/name/inline/expires_at`，不省略零值；purpose 为 upload/read。上传的 name/inline 必须为空/false，读取的 size 为 0。验 MAC 后重新编码比对原字节，拒绝重复字段、未知字段和非规范 JSON；query 的两项可调换顺序，但值不得用百分号或 padding 编码变体。
- 请求过期条件 `runtimeNow >= expires_at`；上传签发最多 15 分钟，读取最多 65 分钟，过去或超过窗口不能签发。签名用运行时钟，不受测试用业务 clock.offset 影响。
- 签发先按原始精度校验窗口，再向下取整到 UTC 秒，并确认取整后仍未到期；返回的 expires_at 为实际签名时刻，不能用取整接纳略超期限的输入。
- PUT 必须是 `u` key，GET/HEAD 必须是 `o` key；GET 签名不能当 HEAD 签名，上传/读取 MAC 不能互用。客户端不能通过改 query 选择 inline、下载名或类型。
- local 返回同源相对 URL，不读取 Host/X-Forwarded-Host 来组装目的地址；后续生成合同用 uri-reference。SignedRequest.Headers 只有客户端须提供的头，PUT 包含 Content-Type，Content-Length 由浏览器根据 File 自动提供但仍在签名中约束。
- 使用持久密钥时重启可验证旧签名；不同密钥或不同 purpose 必须失败。测试密钥在测试内创建，不能读取本机凭据或用户密钥。

### 2.4 原始 HTTP 合同

- 验证签名前不查询对象是否存在。匹配具体传输端点后的签名/key/路径编码/用途/时效错误统一 403 `asset.invalid_signature`，响应不回显 URL/key；进入端点前的非规范路径按 §2.1 拒绝。有效签名对应对象缺失才是 404 `asset.object_not_found`。
- PUT 要求已声明的非负 Content-Length 等于签名 size；没有或不等为 400 `asset.invalid_upload`，超过 files.size_limit 为 413 `payload_too_large`；实际读出超额体同样拒绝，少字节不得发布。Content-Type 必须匹配签名的规范化 media type，不允许客户端另附 charset 等参数。
- 验签按全局 100 MiB 格式边界检查，当前 size_limit 由传输层和 Put 单独执行；降低配置上限后，旧签名的超限请求仍明确返回 413。短体同时触发请求取消时保留实际读取错误，返回 400；读取后端故障不能误归为上传错误。
- 成功 PUT 为 204；重复 key 为 409 `asset.object_exists`。原始存储 HTTP 与普通 JSON API 的鉴权模型不同，不能把它加入 API 的 publicOperations 以绕过 bearer 守卫。
- 读取为 200/206，HEAD 无 body；ETag、Last-Modified、Content-Length、Content-Type、nosniff、Content-Disposition 完整。任意附件 attachment，inline 只接受 JPEG/PNG/WebP/GIF 类型；真正图片字节的业务验证在 P2 确认阶段做。
- 支持单段 Range、If-Range、If-None-Match；无效/多段 Range 为 416。条件读取也先验证签名，过期签名不能得到 304。缓存 private，max-age 不超过剩余有效期，错误 no-store；JSON API 继续 no-store。
- 文件名最多 255 个 Unicode 字符，不能为空，不含控制字符、CR/LF 或路径分隔符；Content-Disposition 由标准库格式化，不能手拼 header。错误使用固定 detail，不包含本地路径或签名值。

### 2.5 期限、并发和配置

保留 `files.size_limit=5242880`。本段新增以下严格配置；不支持的 driver 启动失败，不回退。

| 配置 | 默认 | 有效范围/解释 |
|---|---|---|
| `files.driver` | `local` | P1 仅 local，P3 实现 s3 时扩充 |
| `files.local.root` | `./data/files` | 非空目录；日志不输出路径 |
| `files.min_transfer_bytes_per_second` | 131072 | 1024–1073741824，预算计算采用溢出安全的整数向上取整 |
| `files.transfer_grace` | 15s | 大于 0，至多 60s |
| `files.transfer_idle_timeout` | 30s | 大于 0，至多 60s |
| `files.max_transfer_duration` | 15m | 大于 0，至多 1h，须容纳配置最大文件的完整预算 |
| `files.max_concurrent_transfers` | 16 | 1–256；无等待队列，满额 503 `server_busy` |
| `files.transfer_rate` | 60/min，burst 20 | 复用 BucketConfig 及其合法性规则，独立于 JSON anonymous 桶 |

`size_limit` 在 1–104857600（100 MiB）之间；默认未变。单请求总预算 = grace + ceil(size/min_rate) 秒，默认 5 MiB 为 55 秒。读取从可信对象大小计算；降低新上传上限不使旧对象失读，读取预算最多取 max_transfer_duration。有效配置必须容纳最大新上传的完整预算。所有乘加/ceil 拒绝溢出，不被负数、float 或超大值绕过。

原始请求仍经过平台 request ID/recover/access log/security headers。复用 httpserver 已有 trusted proxies/IPv6 处理与 ratelimit，不复制客户端 IP 解析；新增仅做 request meta 和给定 bucket 限流的传输包装，不改普通 API Middlewares。

取得传输名额后，设置 context 总期限及 ResponseController 读写期限；上传读侧、下载写侧按实际进度和 min(剩余总预算, idle_timeout) 续期，零进度不延长空闲期限。GET/HEAD 的后台读只设总期限，不能把没有新请求数据当作下载空闲。上传响应写期限覆盖整个传输预算及固定 5 秒响应余量。不能清零全局 deadline，不能仅靠 context 取消而留下阻塞的 socket 读写。

socket 期限使用系统真实单调时钟，不使用业务 clock 或用于签名测试的可注入时刻；进度续期与取消中断串行保护，防止续期覆盖已生效的取消。正常结束先注销取消回调再结束预算 ctx，避免回调抢在 HTTP 最后刷新头部时令连接过期。上传空闲/总时限在响应尚未开始时返回 408 `asset.transfer_timeout`，主动取消仍沿用平台 499；写出响应后失败截断连接，不在文件尾追加 JSON。所有退出路径释放名额、reader 和本次临时文件。

### 2.6 组合与隔离

asset 的 module.Register 只注册本段 raw 路由；P1 没有业务动作，按既有动作目录明确列入 actionlessModules，不改目录核对机制。P2 接业务用例时须按真实授权入口核对登记；P2 预检已确认权限委托宿主使用既有 Actions，保留具名 actionless 登记并更新理由，见 P2 spec §2.2。bootstrap 创建 local 资源并在任一后续构造错误和 close 时释放；先停止 HTTP，再停止 jobs，最后关闭存储和 migrator/pool，避免 P2 的清理任务仍在使用已关闭存储。不能让每次 newApp 测试共享默认磁盘目录。

Go 的 testConfig 为每个世界提供独立 t.TempDir。E2E 的 startNerve 默认把文件目录放进该进程日志的输出目录；clock world 重启显式复用该世界的文件根目录，而不同 world 不共用。只保留测试自身生成物，原件按项目既有归档规则处理，不执行全局清理。

完整程序增加精确 raw route 集合与无签名拒绝测试；既有 /api/v0 注册集合、公开操作、安全头、401 与请求体上限/期限守卫继续原样成立。不靠宽泛忽略 /static 下所有路由使新端点脱离目录检查。

`api/storage-local.json` 是 raw 路由和 asset 专用 `x-problem-codes` 的机器目录。组合测试比较全路由与“既有 API + 平台 + raw 目录”的精确并集，源码错误码与目录逐项一致；前端消息表精确等于业务 OpenAPI 与 raw 错误目录的并集，两份翻译同时验证。它不生成业务 SDK，也不把 raw 端点加入 publicOperations。

### 2.7 开发传输与构建边界

Vite 使用具名 local-transfer 插件将 `/static` 的请求原样流式送到本地 Go；方法、字节、必要头和 query 不改写，相邻 `/static-more` 仍是普通页面。代理只做传输与生命周期管理，验签/权限仍由 Go 负责；用于分派的路径归属检查不替代 Go 的拒绝策略。

该插件用 Node 标准库管理自己的上游请求和失败出口，不引入依赖，不注册 Vite 默认的 raw proxy，避免默认错误/debug 日志输出 capability query。日志只含固定上下文和安全的错误类别，不含原 URL、query、body、任意错误 message/stack；上游不可达返回固定 502 no-store，正文开始后的失败截断，不追加错误尾。取消/断连/开发服务器关闭必须释放上游；代理空闲预算为 120 秒，宽于 Go 允许的最大 60 秒 idle，不改变 Go 的业务预算。

构建时拒绝 public 或 bundle 占用 `/static`，保留 Vite 构建资源的原缓存行为。真实开发服务器测试须证明签名标记不会出现在上游失败日志中；不能仅检查配置文本。

## 3. 验收

A1–A10 已满足；局部验证见 [review §2/§9/§10](../reviews/P1-local-storage-review.md)，准确分支/main CI 和归档清理收据见 review §11。前述实现过程中的待办是历史时点。

- [x] A1：模块、小端口与架构守卫。
- [x] A2：真实字节、完整发布与删除合同。
- [x] A3：字段/用途/时效拒绝与时钟隔离。
- [x] A4：不可变发布与条件封存竞争。
- [x] A5：默认配置实际慢上传及期限释放。
- [x] A6：HEAD/Range/缓存和安全响应。
- [x] A7：限流、并发与失败资源恢复。
- [x] A8：生命周期、密钥与目录隔离。
- [x] A9：路由、命名空间、开发传输与守卫无回归。
- [x] A10：准确分支 CI 41466462231、noff main 85dc4087 的 CI 41471671630 全绿；792 文件归档核验及分支/worktree清理后复验完成，见 review §11。

| 编号 | 可观察条件 |
|---|---|
| A1 | domain/app 内无 HTTP、文件系统、SDK、其他 module import；存储方法拆成实际消费的小端口 |
| A2 | 正确签名 PUT→Inspect→Seal→GET/HEAD 字节与摘要相等，半文件不可见，删除幂等 |
| A3 | key/path/query/header/method/expiry/purpose 的变异拒绝，错误无磁盘路径/秘密，runtime 时钟与业务时钟分离 |
| A4 | 二次 PUT 不能覆盖，源 ETag 失配不 seal，两个独立候选并发不相互覆盖，公开 PUT/GET 不能写/读错误 key 类别 |
| A5 | 超 1 MiB 且真实超过 30 秒的有效上传成功；普通 JSON 仍受 1 MiB/15 秒约束，停滞/总期限/取消释放资源 |
| A6 | Range/HEAD/条件缓存/文件名注入/类型处置和过期后请求行为满足 §2.4 |
| A7 | 单 IP 限流、可信代理、IPv6 及并发满额拒绝有效；任一错误后额度与并发名额恢复符合规则 |
| A8 | bootstrap 正常/失败/关闭释放根句柄；固定密钥跨重启签名可用，新密钥不可用；不同测试世界目录隔离 |
| A9 | 全程序 raw 路由精确匹配；合法 storage 工作区页面、保留名字总集合、构建资源与开发传输/失败日志边界均验证；原 API 认证/结构/缓存/安全头无回归；目录内恶意符号链接/设备类路径不被当文件提供 |
| A10 | 有意义的负例和定向变异、七项本地门禁、deep-review/修复复审、准确分支与 main CI，以及证据归档完成 |

F2/F3 的 P1 证据仅限 local 存储合同，不能提前勾选完整 M5 故事或关闭业务资产生命周期交接。

## 4. 实施与验证方法

按 [P1 plan](../plans/P1-local-storage.md) 逐 Task；每项记录性质红/实现绿、spec 和质量复核。已有性质不伪造首次红，关键变异要指向原本正确的保护条件。真实慢流是特定时间性质试验，其他并发用 barrier/条件同步，不用 sleep 碰运气。

新增依赖：无。只用标准库、已有 MAC/ratelimit/HTTP 平台和当前测试工具。图片解码依赖在 P2 首次使用前点名并锁版本，SDK 与 S3 容器留 P3 的具名版本预检。

## 5. 本段之后

P2 承接业务资产行、准备件、候选 metadata、有界清理/清单、权限与事件；P3 承接 S3 与浏览器 CORS；P6/P7 承接传输 UI。不把 P1 的原始签名服务当作用户已能上传附件。

M0/M2 的慢传输子项可在 review 记为已验收，但交接整体等真实业务链和 S3 证明完整后再关闭。所有 M4 完成状态保持；没有新增 CLI，也没有复用旧 copy 入口。
