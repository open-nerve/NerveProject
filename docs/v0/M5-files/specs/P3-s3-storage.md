# M5 / P3 S3 存储合同

> 已完成实施、整审修复、noff集成和归档；最终分支06820e34的CI41588478051全绿，main5807a0d6的CI41591764345全绿（345 E2E，13.9m）。2251文件归档核验并正常清理分支/worktree；P4预检接续见[阶段记录§19](../reviews/P3-s3-storage-review.md#19-集成归档与p4预检2026-11-03)。原阶段内“待CI/待集成”为当时快照。

## 1. 目的、范围和方案

在 P2 消费方拥有的小存储端口后增加 S3，使同一套准备、确认、读取、复制、删除和清理用例能保存到私有对象桶。交付真实兼容协议、配置/部署说明、精确 CORS/CSP 和 F4 浏览器实物故事；local 行为完整回归。

采用官方 AWS SDK v2 的显式客户端和静态凭据，保留 P2 权限、锁序、候选持久化、外部 I/O 后重验及分页恢复协议。没有新的公共 API、数据表、owner 权限实例、后台清理入口或跨模块 SQL。不实现 P4/P5 绑定、P6/P7 页面附件、multipart、版本化桶、Object Lock、访问点/Express 桶、凭据自动发现/刷新或存量后端迁移。

已比较：① 官方 SDK 加现有小端口，使用真实服务证明兼容（采用）；② 手写 SigV4/通用带凭据 HTTP 客户端，增加签名与出站边界维护面（不采用）；③ 为每个品牌写一套 app 或放宽条件复制（破坏 P2 不可变合同，不采用）。单版本桶、专用测试服务、有限配置格式、测试辅助程序属于本阶段的局部工程裁定，代价见 §9。重大新风险仍先提交再上报。

## 2. 结构、依赖和资源所有权

- `asset/adapter/s3` 拥有 SDK、SigV4 期限适配、HTTP、对象元数据和错误转换。Store 实现既有 `UploadSigner`、`ReadSigner`、`ObjectInspector/Reader/Sealer/Deleter/Lister`、`TemporaryCleaner`；app/domain 不增加 SDK、HTTP 或配置依赖。
- bootstrap 只在自身内部聚合这些接口；各用例仍分别接收小端口。`asset.Deps` 显式接收 Files/RuntimeNow，local raw transfer 为可选的 HTTP 依赖，业务配置不再从 raw transfer 反取。组合根私有assetComponents提供runtimeNow给业务构造，生产默认time.Now；原测试时钟覆盖也改走此显式入口，不再借本地HTTP传输依赖。S3 不构造本地 MAC、目录或 raw handler；asset 模块不判断驱动字符串。单一 gate、权限、River 和 DB 实例不变。
- `New(ctx, Config, Deps) (*Store,error)` 有界只读初始化；`Close() error` 由组合根在 HTTP/jobs 停止后调用。构造中途失败释放自建 transport；不持有 DB 连接。取消/超时、关闭和错误日志均须可证明，不把 SDK 请求结束等同服务端已经停止。
- 直接模块精确锁定：`github.com/aws/aws-sdk-go-v2 v1.47.5`、`.../credentials v1.20.13`、`.../service/s3 v1.114.6`。必要间接模块：`aws/protocol/eventstream v1.7.25`、`internal/configsources v1.5.8`、`internal/endpoints/v2 v2.8.8`、`internal/v4a v1.5.8`、`service/internal/accept-encoding v1.13.23`、`checksum v1.11.9`、`presigned-url v1.14.8`、`s3shared v1.20.8`、`github.com/aws/smithy-go v1.28.7`（省略前缀的条目均属 `github.com/aws/aws-sdk-go-v2/`）。不引入 aws/config 或 JS AWS SDK。
- 直接构造与部署配置共用platform/config的纯S3格式验证；该依赖只在adapter边界，app/domain不导入config。现有测试依赖testcontainers-go v0.44.0、moby/api v1.55.0、containerd/errdefs v1.0.0因显式夹具、loopback端口类型和Docker实际NotFound验证转为直接声明，版本不变、未增加其他模块；s3test及子包明确禁止进入生产导入/二进制。
- SDK 第一次实际消费时再改 `server/go.mod/go.sum`。测试专用 `server/tools/s3fixture` 消费同三版本及必要间接模块，按需改 tools 模块；Node 用既有 testcontainers，将现有传递依赖 `testcontainers 12.1.0` 声明为 e2e 同版 devDependency。设计先点名，不能安装未列出的库。

## 3. 配置、秘密和启动

现有 files 限额/预算不扩大：新上传 1..104857600 字节、默认 5242880；upload_ttl 10s..15m、默认15m；operation_timeout 1s..10s、默认10s；其他 P2 清理常数保持。新增 `files.s3`：

| 字段 | 合同 |
|---|---|
| endpoint | 必填主机为DNS或IPv4的http/https origin；IPv6字面量因浏览器CSP不兼容而拒绝（DNS可解析到IPv6），不放宽策略；只接受空 path 或 `/`，拒绝 userinfo/query/fragment/控制字符、非法端口；不提供跳过 TLS 开关 |
| region | 必填，1..64 ASCII 字母/数字/连字符；不发现系统 region |
| bucket | 必填，3..63 小写 DNS 普通桶名；首尾字母/数字，允许内部 `-` 和 `.`，无相邻点、无点连字符相邻、无 IPv4 字面量，拒绝 ARN 和 AWS 特殊桶/别名保留前后缀 |
| prefix | 必填，实例独占；1..128 字节，ASCII 字母/数字/`_`/`-` 段以 `/` 分隔并以 `/` 结尾；无空段、dot、前导 `/`；不自动给所有实例同一个前缀 |
| path_style | bool，默认 true；false 仍由官方 EndpointResolverV2 决定实际寻址方式，不能手工拼 bucket host |
| access_key_id / secret_access_key | 各恰有一个值或同名 `_file` 来源；值最长256/4096字节，拒绝控制字符和空白 |
| session_token | 可选，值或 `_file` 二选一；最长16384字节，同样拒绝控制字符/空白；不自动刷新 |

直接秘密值通过 `NERVE_FILES__S3__...` 注入；YAML 只允许空默认值或 `_file` 路径，非空内联秘密拒绝。文件仅在 S3 组合时读取，允许内容上限外最多一个LF/CRLF；实际读取最多上限+3字节（额外1字节用于判超限），最多移除末尾一个 LF 或 CRLF；空文件、超限、目录和读取失败均报配置键及类别，不带值/路径。值与文件同时声明始终拒绝（包括未启用S3时），不能按优先级默默忽略。local 不要求 S3 完整配置、不读取 S3 文件；活动 S3 配置错误不得 fallback 到 local。所有凭据/文件路径 LogValue 只输出是否已设置，SDK/HTTP 错误也不能泄露签名 query、Authorization、secret 或 token。

配置验证后、SDK解析/签名前，将endpoint主机名转小写、端口转规范十进制并省略HTTP80/HTTPS443。拒绝浏览器会改释为IPv4或拒绝的非标准数字主机（短IPv4、八/十六进制及数字末标签的伪DNS），标准四段IPv4仍接受。官方 `EndpointResolverV2` 再计算实际 bucket endpoint；出站 HTTP 守卫与 CSP 共用其规范化 origin。路径/host 在签名之后不修改，也不放宽精确Host守卫。独立 HTTP Transport 使用系统 TLS、`Proxy=nil`、拒绝所有重定向，限制请求到该唯一 origin；SDK `RetryMaxAttempts=1`。每次操作继承 caller context，并受 operation_timeout 限制；不得沿用 credential-free link preview 的 outbound.Get。Close 释放自有空闲连接。

启动在一个 operation_timeout 总期限内读取 HeadBucket 和 GetBucketVersioning，只接受版本状态为空（从未启用），Enabled/Suspended/未知/权限不足均拒绝。生产启动不创建桶、不改 CORS、不写探针对象；只读启动检查不宣称证明所有厂商的条件复制。运行账号需读桶状态权限；部署保持桶模式、桶私有和 prefix 独占，不用外部生命周期策略删 Nerve 活对象。支持性需通过下述真实合同，不能只凭“S3 compatible”字样。

## 4. 签名和实物合同

### 4.1 签发

先执行既有 domain.ValidateUpload/Read。PUT 绑定合法 u-key、精确 ContentLength、规范 ContentType 和期限；无伪造 Body。公开 Headers 只含浏览器能设置的必要头（Content-Type），Host/Content-Length 仍被签入但由浏览器/HTTP 栈产生；不带 Nerve bearer/cookie。提供方 PUT 的200/204都可为成功，公开业务 API 不变。

使用官方 HTTPPresignerV4 扩展点，在 SDK 传入真实 signingTime 后、调用官方 v4.Signer.PresignHTTP 前，令 X-Amz-Expires 等于 `deadline.Unix()-signingTime.Unix()`；保留 S3 DisableURIPathEscaping。差值必须正且在 domain 上界内，不伪造 SDK 时钟、不改签后 URL。签名实际到期秒必须与已持久化/公开 expires_at 一致；排队跨秒或已过期返回错误。既有上传最多15m、读取最多65m。

GET/HEAD 使用各自 SDK Presign 操作，只签 o-key；绑定方法、key、prefix、桶和完整 query。通过标准响应覆盖设置类型、Content-Disposition（普通附件 attachment，inline 仅四类合法图片）及 `Cache-Control: private, no-store`；中文/引号文件名不能注入头。S3不能按实际GET时刻动态计算剩余缓存秒数，因此不把签发时的相对max-age固化到URL，避免临近到期才取件导致缓存越过期限；代价是S3字节不持久缓存，业务content URL仍可在expires_at内复用，local原动态private缓存不变。业务 content GET 保持零 DML。过期 URL 不能开始新请求；有效期内已开始的传输可能晚到，按 §5 恢复，不能声称到期强制中止服务端。

### 4.2 检查、流读取、复制、删除

HEAD 返回实际 key/Size/ContentType/ETag/LastModified；ETag 是单个强HTTP不透明版本，保留完整引号，不当SHA；拒绝通配符、列表、弱标签和控制字符，不能把预期版本比较放宽为任意对象匹配。结构缺失、负值、无效时间/类型/ETag或超过既有16KiB元数据上界均明确失败；独占S3 transport在SDK解析前限制响应头总量32KiB，给协议头保留余量，不改变全局HTTP服务器预算。只有对象404/NoSuchKey映射 ErrObjectNotFound；403、5xx、TLS/DNS/超时不能伪装为缺失。bucket 不存在是后端故障；HEAD无内容404及兼容提供方含糊的NoSuchKey均在同一操作剩余期限内用HeadBucket确认桶存在后才映射缺对象，不能把运行中删桶当清理成功。SDK 200 内嵌 COPY 错误不能当成功。

Open 用 GetObject IfMatch 读取预期版本，返回流而非 ReadAll；取消后（包括先 EOF 再取消）返回 context 错误，Close 释放连接，不能在返回 reader 前取消其 operation context。完整字节数和 SHA 仍由 app 流式验证。

Seal 把源复制到全新 o-key，同时带 CopySourceIfMatch 和目的 IfNoneMatch=`*`，不退无条件 COPY，不以“先 HEAD 目的再写”替代原子条件；正确处理编码后的 bucket/prefix/key。源变化或目的存在的412/409映射 ErrObjectChanged；缺源404为 ErrObjectNotFound。空 expected ETag 不发无条件 COPY：先识别确实不存在的源，否则拒绝。成功后返回目标 HEAD，app 继续双流完整 SHA；源先后变化、同大小异内容、目的竞争均不能发布错误文件。

Delete 缺失幂等成功，其他失败必须保留；只操作本 prefix 的合法 key。首期不允许 versioned/Suspended 桶，避免“删了 delete marker、旧字节仍留存”而丢恢复记录。外部成功、响应丢失或最后 DB 提交失败，持久 candidate 仍按 P2 恢复，旧候选/旧 PUT 不得替换已发布 final。

## 5. 清单、晚到数据和恢复

ListObjectsV2 每次只访问独占 prefix，limit 1..1000；结果必须严格升序、无重复、每个 key 都能从确切 prefix 还原为合法 u/o key，大小/时间/ETag有效。Next 是不解释的 provider ContinuationToken，最长8192字节；空表示耗尽。截断但无 Next、Next不前进、超页、越界 key、异常布局一律明确失败，不全桶扫描、不无界缓存。清单响应XML在SDK解析前限制为64KiB + limit×16KiB（含未知节点及chunked正文）；其他协议正文（启动、COPY、错误等）上限64KiB。超出明确失败；HEAD没有正文，其Content-Length仍为对象大小；仅显式Open的成功GET流不受元数据正文限制。SDK解析失败不能仅凭HTTP404/409/412转成业务缺失/冲突，必须先得到完整解析的API错误；真实HEAD空正文仍由SDK生成状态错误，并沿同预算核桶规则。P2 的 token+ObjectAfter 负责页内进度，页不是快照。

S3 没有 local tmp：TemporaryCleaner 校验 context、非零 cutoff、limit1..1000、token必须为空，然后返回耗尽（next空/removed0）。app 不能按driver新增分支。driver/prefix变更不保证旧cursor或旧资产迁移，部署不热换数据归属。

runtime LastModified 与业务墓碑时钟分开。对象客户端超时、URL过期或本地取消，不证明服务端请求已停止；候选key永不复用、迟到完成必须过 P2 最终锁内事实重验。删除之后才落地的未知实物仍由 runtime24h+所有DB引用复核清理。真实提供方配受控延迟代理验证迟到 PUT/COPY、响应丢失、清单中断/重启、分页变化，不能用业务快进证明真实上传已终止。

## 6. 组合根、路由和浏览器

驱动选择只在 bootstrap。可选 local raw handler 缺席时，六个资产业务API仍注册，`/static` 保留树继续阻止规范化重定向且未知local传输返回404；合法 storage 工作区路由不受影响。完整路由目录按明确 local/S3 两种模式验证，不把所有 `/static` 或未知操作从守卫豁免。JSON 1MiB/15s及全局连接预算不扩大。

webui.Options 只接 StorageOrigin；Handler 验证后只对页面 img-src/connect-src 增加该确切 origin，无通配。local 的 CSP逐字保持，inline脚本hash、其余指令、API安全头和错误页策略保持；Options不接SDK或整个files配置。实际endpoint、签名URL、CSP的origin须一致，含path_style=false、dotted bucket和显式端口。

生产部署样例包含精确 frontend origin，AllowedMethods PUT/GET/HEAD，AllowedHeaders Content-Type，ExposeHeaders ETag/Content-Length/Content-Type/Content-Disposition；不设置 `*` 或允许credentials。CORS是浏览器读取边界，签名URL本身是bearer capability；错误origin的浏览器应无法读取，不能谎称CORS能阻止拿到URL的非浏览器上传。桶仍私有，匿名URL拒绝。

F4 在真正应用 origin 和浏览器 fetch/File/Blob 中完成 prepare→raw PUT→complete→content→GET→duplicate→DELETE，实物 SHA/大小与DB结果核对。覆盖正确长度/类型、改长度/类型403、匿名拒绝、跨origin错误、签名错误、CSP阻断未列出的origin，三浏览器专门复验。业务content API只有GET能力，F4实际验证GET成功、改HEAD返回403；合法HEAD独立签名的正例由T3存储适配器真实协议覆盖，不新增业务API或测试helper签名动作。仅 APIRequestContext 或Go HTTP通过不代表浏览器/CORS通过。此阶段不创建附件UI。

## 7. 可重复测试服务和辅助程序

真实合同使用 [SeaweedFS 4.52](https://github.com/seaweedfs/seaweedfs/releases/tag/4.52)，源码984eac6a790be89e2a11de175a7413e1e3e32dd3；镜像 `chrislusf/seaweedfs@sha256:57877e32bb90073b37dc109b438ba66e84fd5ed9ced5f5c04f56444ef243ce59`。amd64子摘要a54ba35b44895a449f5cdd415999062201b17b9835ec0ffc19ea605b72ffa3d1；arm64子摘要18a805c81cd574a24571cfbc23ff1cb0bb2b875b1ac028a58c065c15f0c7b3ad。

测试只新建和管理自己的容器、随机凭据/桶/prefix、loopback端口和临时文件；已有Docker/容器不动。Go夹具沿用现有testcontainers-go，Node沿用同框架；套件串行执行，单服务多世界可共用服务但不共用桶/CORS/DB。多桶要预留实际卷容量，不把启动403视为实物可写。启动总时限90s、停止30s，失败路径也释放自有资源；测试提供方的配置/数据可留失败附件，真实凭据不可从外部寻找。

`server/tools/s3fixture` 是测试专用一次性JSON stdin/stdout程序（输入≤1MiB、输出≤1MiB、命令期限10s），构建至ignored `bin/nerve-s3-fixture`，不是nerve CLI命令。动作限于为本测试创建桶、设精确CORS、按key读metadata/SHA和删除测试数据；拒绝不合法输入/未知动作，不输出凭据/签名。Node夹具等Nerve实际baseURL后设置CORS；关闭/启动失败/重复关闭都回收自己的容器、子进程、数据库连接和临时秘密文件。无JS SDK、无自写SigV4、无Seaweed内部管理API依赖，不据此关闭未来River CLI交接。

失败启动的回收证据按本用例实际资源核验：世界在启动Nerve前记录确切容器名与秘密目录到自己的outputPath，只含归属标识、不含凭据；不得用全局容器/临时目录快照推断自己是否泄漏。恢复审计断言比较实际入库前后记录，保留亚微秒输入覆盖平台差异，不以时间容差掩盖审计变化。测试诊断Minor P3-M1的双失败cause保留由P8具名处理，见re-37。

预检已验证上述固定SDK/服务的7组真实协议、精确期限适配及版本化桶旧字节反例；第一次夹具卷容量失败和后续原件全部保留。该证据不证明Nerve适配器、浏览器、AWS云或其他品牌已通过；永久合同是P3实施内容。

## 8. 完成标准

- [x] A1 配置/活动驱动、秘密来源/文件上限/日志遮盖、只读启动和单版本限制有真实负例；依赖锁定和纯层/导入守卫不放宽。证据：[阶段记录](../reviews/P3-s3-storage-review.md)§13，真实S3/PG/API、故障恢复及受影响race。
- [x] A2 local/S3同一实物合同均通过；签名方法/大小/类型/期限/篡改、GET/HEAD和取消后的reader有独立断言。证据：[阶段记录](../reviews/P3-s3-storage-review.md)§10/§11，T4最终race的两Factory共享合同及S3签名/实物用例。
- [x] A3 源/目的双条件COPY、同大小异内容/并发目的/源改变、200内嵌错误和错误分类证明final不可变，DB提交故障不丢candidate。证据：[阶段记录](../reviews/P3-s3-storage-review.md)§13，真实S3/PG/API、故障恢复及受影响race。
- [x] A4 有限prefix清单、opaque cursor、页内/进程恢复、后到实物回收、邻域保护、双时钟和无tmp合同通过。证据：[阶段记录](../reviews/P3-s3-storage-review.md)§12，真实S3/PG与新OS进程River续跑。
- [x] A5 S3完整Nerve API/PG/实物闭环、权限撤销/锁外I/O/零DML和原local回归；路由目录两driver准确。证据：[阶段记录](../reviews/P3-s3-storage-review.md)§13，真实S3/PG/API、故障恢复及受影响race。
- [x] A6 精确CORS/CSP及三浏览器F4通过，实际File长度/类型绑定、匿名/错误origin/未准许CSP均受测；无新附件UI。证据：[阶段记录](../reviews/P3-s3-storage-review.md)§14，三浏览器、独立字节/SHA、CSP实际变异与IPv6反例、资源失败回收。
- [x] A7 1B/默认5MiB/最大100MiB流式容量、真实慢流/取消、连接/资源回收有记录；不以本机吞吐承诺任意公网性能。 证据：[阶段记录](../reviews/P3-s3-storage-review.md)§15，容量/资源race、25项变异、四组各五轮及守卫原件。
- [x] A8 至少15个独立性质变异命中具体断言而非编译失败/0 tests，记录还原hash；并发封存/迟到/恢复交错跑race五轮，无死锁/数据竞争；原始失败、重试和测试输入完整保留。 证据：[阶段记录](../reviews/P3-s3-storage-review.md)§15，容量/资源race、25项变异、四组各五轮及守卫原件。
- [x] A9 七项本地门禁、生成和本阶段独立受影响回归通过；完整三视角deep-review/一次新上下文整分支审查及修复闭环，精确branch CI通过。
- [x] A10 noff main精确CI全绿、完整证据归档校验/常规清理，P4 spec/plan预检；不提前勾选整个M5或关闭绑定/UI交接。

## 9. 取舍、限制和证据入口

单版本限制换取沿用P2可靠实物删除，代价是不能直接接已开过版本的桶；若以后要支持必须联合设计VersionId/版本清单/恢复，不能偷偷降级。明确endpoint/静态凭据和不跟随redirect换取出站边界，代价是部署要配置正确region/origin及自行轮换重启。条件COPY多一次存储请求，并有两次流式hash，这是已批准不可变文件的成本。测试SDK小程序增加测试构建，但复用同版签名实现，避免第二套JS SDK/协议。以上不改变owner或P2持久化协议。

P2全量341 E2E在2workers约15.6m；P3新增F4必须记录真实CI时间，复用自己拥有的服务降低开销，不能删旧断言、无限重试或用本地绿代替main。已知W11按既有一次重跑规则；再失败停报。M0 HTTP预算交接仅记录P3兑现部分，未来CLI不提前关闭；其他输入交接等对应P4–P8实施。

主来源：[SDK S3发布](https://github.com/aws/aws-sdk-go-v2/blob/service/s3/v1.114.6/service/s3/CHANGELOG.md)、[SDK凭据发布](https://github.com/aws/aws-sdk-go-v2/blob/credentials/v1.20.13/credentials/CHANGELOG.md)、[CopyObject条件与内嵌错误](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CopyObject.html)、[桶版本状态](https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketVersioning.html)、[预签URL的请求开始期限](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-presigned-url.html)。准确探针、manifest、失败原件路径由预检记录附上。
