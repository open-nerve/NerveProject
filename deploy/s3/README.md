# S3 私有存储部署

M5/P3 使既有资产 API 使用 S3 字节存储。默认仍为 local；选择 S3 不创建本地文件根目录，也不开放本地 PUT/GET/HEAD 路由。业务权限、上传确认、候选恢复、软删除及 River 清理保持原合同。浏览器凭短期签名地址直传，Nerve API 不代理整文件。

## 配置与运行身份

先由部署方准备普通私有桶、运行身份和 CORS，再启动 Nerve。以下是配置片段，需替换桶名、区域、来源和秘密文件路径：

```yaml
files:
  driver: s3
  size_limit: 104857600
  operation_timeout: 10s
  s3:
    endpoint: https://s3.us-east-1.amazonaws.com
    region: us-east-1
    bucket: nerve-private-example
    prefix: instance-a/
    path_style: false
    access_key_id_file: /run/secrets/nerve-s3-access
    secret_access_key_file: /run/secrets/nerve-s3-secret
```

等价环境变量为 `NERVE_FILES__DRIVER=s3` 和 `NERVE_FILES__S3__ENDPOINT` 等双下划线键。秘密仅用显式环境值或 `_file`，同一字段不能两者并存；YAML 不接受非空明文秘密。可显式配置 `session_token` 或 `session_token_file`，程序不自动刷新临时凭据；部署方应确保其有效期覆盖待使用的签名和在途操作。

Access ID、secret、session token 分别最多 256、4096、16384 字节；文件允许一个尾部 LF 或 CRLF，不能有其他空白。秘密文件应只允许运行账户读取。程序不读取 AWS profile、共享配置、IMDS 或默认凭据链，错误及配置日志不输出秘密值或秘密文件路径。

Endpoint 必须是唯一 HTTP(S) origin，不带用户名、密码、路径（尾 `/` 除外）、query 或 fragment；生产使用系统根证书验证的 HTTPS。来源主机使用 DNS 或 IPv4；CSP 无法精确表达 IPv6 字面量，因此配置提前拒绝这种写法，可用解析到 IPv6 的 DNS 域名，服务器本身的 IPv6 监听不受影响。path_style=false 时，SDK 解析出的桶主机名才是实际存储 origin，CSP 使用同一个来源，含端口；带点桶名可能由 SDK 选择 path-style，以最终签名 URL 为准。区域不对而要求重定向的 endpoint 会失败，应修正配置，不能放开重定向。程序不使用环境 HTTP 代理、跳过 TLS 或重试写请求。

Prefix 必填、实例独占、1–128 字节，使用字母、数字、`_`、`-` 组成的路径段并以 `/` 结尾。禁止其他实例或外部生命周期规则修改、归档、清除这段数据；程序只按该前缀有限分页，不扫描全桶。

## 私有桶与最低权限样例

[policy.example.json](policy.example.json) 是无外部标签、无额外 KMS 要求的普通桶运行身份样例，替换桶及 prefix 后由部署方设置。生产进程只需要读桶状态，以及在独占前缀读、写、复制和删除对象；没有建桶、改版本化、设 CORS、改 ACL 或 bucket policy 的权限。COPY 使用源读取和目标写入权限。[AWS 操作权限说明](https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-with-s3-policy-actions.html)

注意 `HeadBucket` 使用桶级 `s3:ListBucket`，请求没有 prefix；样例因此允许读取该桶的名称清单，不把它描述为 IAM 层面的前缀列表隔离。若对象名称也需隔离，部署独立桶。仅把 `s3:ListBucket` 加上必须匹配 prefix 的条件会使只读启动检查失败，不能靠删启动检查绕开。[AWS HeadBucket 权限](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadBucket.html)

必须使用**从未启用版本控制**的普通桶。启动在单个 operation_timeout 内只读 HeadBucket/GetBucketVersioning；Enabled、Suspended、未知状态和权限不足均明确失败。生产构造不会写探针。曾启用后暂停版本化也不支持，因为普通 Delete 不能证明旧版本字节已被删除。保持桶私有；Object Lock、Express、access point、自动跨区域和跨后端迁移不在本期支持范围。

## 浏览器来源

端点在签名前统一主机名大小写和端口写法，省略 HTTP 80 / HTTPS 443；不会改写已签名 URL。IPv4 使用标准四段十进制写法。拒绝 `127.1`、八进制/十六进制地址及数字结尾的伪 DNS，因为浏览器会将它们解释为另一地址或直接拒绝。

[cors.example.json](cors.example.json) 仅允许 `https://nerve.example.com` 的 PUT/GET/HEAD 和 Content-Type，暴露 ETag、Content-Length、Content-Type、Content-Disposition。换成实际前端 origin（协议、主机、端口），不要加入路径、`*` 或凭据允许。CORS 是桶管理设置，由部署方一次配置；Nerve 启动不会改它。

页面 CSP 只在 img-src/connect-src 增加存储的确切 origin；脚本来源、内联脚本 hash、其他指令和 API 安全头不变。上传时按 API 返回的 method、URL 和 headers 传 File/Blob，由浏览器自动发送 Content-Length，不能手设该头；改长度、类型或签名返回 403。

CORS 只限制浏览器跨源读取，签名地址本身是 bearer capability。错误来源不能读不代表该地址对非浏览器客户端失效；不要记录或外传完整签名 URL。业务 content API 签发 GET 地址，改为 HEAD 会返回 403；适配器自身独立 HEAD 签名另有真实协议覆盖。S3 读取返回 private, no-store，避免临近 URL 到期才下载时缓存越过期限；local 保持原动态缓存策略。

## 诊断与恢复

- 启动失败：核对实际 endpoint、区域、桶状态和运行身份的桶读取权限；日志只给安全的操作/错误类别，不含 SDK 原始请求或凭据。
- 浏览器 PUT/GET 失败：先区分 CSP、CORS 和 HTTP 403，核对页面来源、实际签名 origin、method、Content-Type、文件字节长度及期限。不要扩大来源或取消签名字段来消除报错。
- 确认失败：目标原子不存在条件和源 ETag 条件都必须受提供方支持；HTTP 200 内嵌 COPY 错误也算失败。数据库已记候选时不得手工删恢复记录。
- 运行中桶被删、403、5xx、超时均是后端故障，不当作“文件本来不存在”。修复服务或权限后让持久 River 清理恢复，避免丢失对象回收依据。
- URL 到期、客户端取消、请求超时都不保证服务端已经停止；迟到对象按既有候选和 runtime 24 小时孤立对象规则回收。清理保留数据库所有引用保护与业务/运行时双时钟。

修改 driver、bucket 或 prefix 不会迁移历史文件或清理 cursor。已有数据必须先另行设计、评审并完成迁移和验证，不能直接改配置、热切换或依靠 local fallback。P3 没有迁移命令。

## 验证范围与复现

已运行的真实提供方为 SeaweedFS **4.52**，镜像固定为 `chrislusf/seaweedfs@sha256:57877e32bb90073b37dc109b438ba66e84fd5ed9ced5f5c04f56444ef243ce59`；源码和平台摘要见 [P3 spec §7](../../docs/v0/M5-files/specs/P3-s3-storage.md)。这不代表 AWS 云、MinIO 或其他标称 S3 compatible 的品牌已完成合同验收；新提供方需跑相同条件复制、错误分类、分页、取消和浏览器合同。

```sh
make build build-s3-fixture
cd e2e
pnpm exec playwright test --config=playwright.m5-s3.config.ts
```

专用配置串行验证 Chromium、Firefox、WebKit，默认 CI 包含 Chromium F4。Go 单元/集成测试使用同版 SDK 与固定真实服务；Go helper 只为本测试的 loopback 服务、随机桶和前缀工作，JSON 输入输出最多 1 MiB、每命令 10 秒、实物读取只返回流式 SHA/大小/头。它不是 nerve 的部署管理命令，不操作生产桶。

预检和实施出现过卷容量不足、秘密文件归属、旧 CSP 阻断上传及测试夹具错误，原失败、修正和复验全部保留。1B、5MiB、100MiB流式容量及真实16秒慢流已验证；这是固定本地环境的数据，不承诺公网吞吐。容量、整审、精确CI及集成归档的当前证据见 [P3 阶段记录](../../docs/v0/M5-files/reviews/P3-s3-storage-review.md)。
