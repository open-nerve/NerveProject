# M5 / P1 本地存储实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过，实施中；见[预检记录](../../../reviews/re-31-M5设计与P1预检.md)。负责人已授权按既有节奏连续推进，常规预检通过后继续，不另设确认门。
> T9 修订已授权：按[最终裁定](../reviews/P1-storage-namespace-decision.md#6-最终裁定2026-11-02)使用 `/static/local/{key}`，补路径规范化、开发代理及构建边界。T9、T10、T11 尚未完成；完整门禁通过前不推送产品检查点。

**Goal：** 建立可真实传输、检查、封存和读取的 local 存储能力，给 P2 资产用例提供经过验证的小接口。

**Architecture：** asset/domain 放值与规则，app 声明存储端口，local adapter 做文件与签名，http adapter 做原始传输。bootstrap 复用既有用途 MAC、限流和 HTTP 平台接线，不放宽 /api/v0 守卫。

**Tech Stack：** Go 1.27.1 标准库、当前 identity MAC、platform/httpserver/ratelimit；无新增依赖。

**Spec：** [P1-local-storage](../specs/P1-local-storage.md)，上位 [M5-design](../M5-design.md)。

## Global Constraints

- 主目录永远 main；实施用 `codex/m5-p1` worktree。文档准备分支为 `codex/m5-design`，实施基线为文档 noff 合入后的 main `fb3e788b7be33467018385c9a67d515e51b79dad`。其树与已全绿的设计分支相同；main CI 等待期间可做本地 Task 和聚焦验证提交；P1 首次产品检查点推送前须确认该 main CI 全绿并归档设计分支。
- plane/refer 只读且不运行其中的 Git；不动已有容器/Docker、不全局安装、不取凭据、不 stash/clean/reset --hard；系统 `/usr/bin/git` 推送，中文详细提交。
- 每 Task 聚焦测试和 spec/质量复核；产品检查点先七项完整本地门禁再推；阶段 deep-review、修复复审、分支 CI、noff、main CI、归档清理依序进行。
- T1 实现提交将 keyword phase 从 M4/closeout 改 M5/P1；当前无 until=M5 的例外，但实施时重新验证。不能仅延期例外，不修改已有架构守卫来迎合实现。
- 只实现 P1 范围；不新增数据库表、业务申请 API、S3 依赖或页面。传输端点始终需要签名，P1 没有面向用户的签发入口。
- 计划 ≤16 Task，当前 11；每个 Task 保存输入/结果/问题/修复/复核。若局部调整只改变文件切分、测试帮手或实现细节，同步 plan 后自主继续；架构回归/重大分歧先提交再上报。

## Review Focus

1. 签名 query 重复值、编码变体、HEAD 与 GET、u/o key 或 MAC purpose 混用（T2/T5）。
2. 上传部分写入/断连、封存过程中源 ETag 变化、候选竞争、失败后名额与临时文件残留（T3/T4/T6）。
3. ResponseController 在双层包装 writer 下是否真正生效；普通 API 的限制是否被顺带放宽（T6/T8/T9）。
4. Range/304 是否在验签之前返回，文件名/类型是否能改变浏览器解释或泄露路径（T5）。
5. newApp 后续构造失败、clock world 重启、并行测试配置是否泄露文件句柄或共享目录（T7/T8）。
6. 命名空间与 M3 保留集合、构建资源边界，非规范路径中途进入再离开、开发代理上下游取消与错误日志的签名泄露（T9 修订）。

## 文件分工与接口

`A=server/internal/modules/asset`，`H=server/internal/platform/httpserver`，`C=server/internal/platform/config`，`B=server/internal/bootstrap`。

| 文件组 | 单一职责 |
|---|---|
| A/domain/storage.go、storage_test.go | key、grant、object info 与值校验；无 I/O |
| A/app/storage_ports.go、storage_contract_test.go | 消费方小接口；共享行为合同入口 |
| A/adapter/local/store.go、object.go、seal.go | root 生命周期、不可变对象布局、检查/读/删、条件封存 |
| A/adapter/local/sign.go、token.go | 按用途 MAC、规范载荷、相对 URL；不查 DB |
| A/adapter/http/transfer.go、read.go、upload.go、budget.go | raw 路由、验签顺序、标准读取、流量预算和并发 |
| A/module.go | 模块外观与注册；资源由 bootstrap 明确拥有，无运行时服务定位 |
| H/transfer.go | 只复用客户端 metadata/限流，保持 JSON 中间件原样 |
| C/files.go、files_test.go，config.go/validate.go | 精确配置和可测试预算约束，日志不含 root |
| B/assets.go、app.go、asset_transfer_test.go | MAC/配置/限流接线与正常/失败/关闭清理 |
| e2e/fixtures/server.ts、clock.ts | 进程/world 的独占文件目录与重启复用 |

domain 构造函数固定为 `ParseObjectKey(string) (ObjectKey, error)`、`NewObjectKey(kind byte, assetID, objectID uuid.UUID) (ObjectKey, error)`、`ValidateUpload(UploadGrant, int64, time.Time) error`、`ValidateRead(ReadGrant, time.Time) error`。两个 UUID 均由调用方显式提供，构造函数不取随机数；ObjectKey 的 Kind/AssetID/ObjectID 通过方法读出，不向调用方暴露正则细节。

local 外观固定为 `New(Config, Deps) (*Store, error)`；`Config{Root string; SizeLimit int64}`；`Deps{UploadMAC, ReadMAC app.MAC; RuntimeNow func() time.Time}`。Store 实现 spec 的六个小接口和 `Close() error`，另提供传输消费者自己的 `Put(ctx, grant, io.Reader) error`；只能由已验签 handler 或测试内调用，内部仍核 key/type/长度/ctx。签名解析用 `VerifyUpload(key string, rawQuery string, method string) (domain.UploadGrant, error)` 和对应 VerifyRead。错误使用具名值与 errors.Is，不用字符串包含判断。

http 外观 `NewTransfer(Deps) (*Transfer, error)`；Deps 包含 store 所实现的精简读/写/验签接口、预算配置、运行时钟与 HTTP 错误写入器。`Register(*httpserver.Router, func(http.Handler) http.Handler)` 注册精确 raw 集合，包装器由 bootstrap 从 API 的 `TransferMiddleware(httpserver.Limiter)` 取得。不把整套 API 或业务 Authorizer 注入存储。

HTTP 的 `TransferReader.OpenTransfer(ctx, key) (ObjectInfo, io.ReadSeekCloser, error)` 只服务本地传输，由同一次打开取得元数据/句柄；app.ObjectReader 保持流式。标准库 ServeContent 负责条件读取，adapter 限制为单段 Range、格式化 disposition、将读取前错误转为 no-store Problem，并在正文不足时中止连接。

## Task 1（T1）：值、配置和消费方合同（A1）

**Files：** A/domain/storage.go、storage_test.go；A/app/storage_ports.go；C/files.go、files_test.go、config.go、validate.go；server/configs/config.yaml；tools/keywords.json。

**Interfaces：** 产生 spec §2.2 的所有值/六个存储接口、MAC 和本计划的 domain 构造函数；配置按 spec §2.5 逐值落实。

- [x] Step 1：写 `TestObjectKeyAndGrantBounds`：空/nil/大写/编码斜杠/非法 kind、size=0/104857601、过期边界、GET/HEAD 区分、控制字符文件名；合法 key 往返保持两个 UUID。
- [x] Step 2：写 `TestFilesConfigurationBudgets`：默认 5 MiB/131072/15s 得 55s；负数、小数、超大整数、driver=s3、空 root、最大文件预算超过 max_duration、零并发、非法 rate 全部拒绝。用现有配置加载/日志帮手核未知 key 和 root 遮盖。
- [x] Step 3：跑 `go test -count=1 ./internal/modules/asset/domain ./internal/platform/config`，记录新增性质红；实现值和配置，不改全局 server 默认。
- [x] Step 4：同命令绿，`go test -count=1 ./internal/archtest`；定向移除一个上限/phase 值保护的负例应失败，恢复后复测。更新 keyword phase，并实际跑 `node tools/keywords.mjs`。
- [x] Step 5：spec/质量复核并提交 `feat(M5/P1): 定义存储值与逐请求传输配置`，正文记录无业务 API/迁移、固定预算和验证结果。

## Task 2（T2）：用途隔离签名（A3）

**Files：** A/adapter/local/sign.go、token.go、sign_test.go；A/adapter/local/store.go 的构造参数；A/keys.go 的稳定用途常量；B/assets.go 的 MAC 构造帮手、asset_mac_test.go 的真实 HKDF/重启核验。

**Interfaces：** 消费 T1 的 grant/MAC；产生 SignUpload/SignRead/VerifyUpload/VerifyRead。Store 的文件能力在 T3 接入，签名代码不可通过 nil 文件依赖猜测结果。

- [x] Step 1：`TestSignedTransferBindsEveryField` 对 method/key/size/type/name/inline/expiry/version 各变一项，均拒绝；合法返回同源相对 URL 和仅 Content-Type 必需头。
- [x] Step 2：`TestSignedTransferRejectsAmbiguousQuery` 覆盖重复 payload/tag、未知 query、非规范编码、4097 字节、错误 purpose、GET 改 HEAD、u 改 o、nil/空 MAC 依赖；错误无输入回显。
- [x] Step 3：用测试自己生成的 identity Keys 通过 bootstrap MAC 连接验证用途隔离及相同 PEM 重启；asset 测试可用简单测试 MAC，但不能替代组合层真 HKDF 证明。跑对应包新增测试确认红，再实现固定格式和恒定时间 Verify。
- [x] Step 4：`go test -count=1 ./internal/modules/asset/... ./internal/bootstrap -run 'TestSignedTransfer|TestAssetMAC'` 绿；删除 method/purpose/expiry 校验的定向变异分别被杀，恢复复测。
- [x] Step 5：两轮复核后提交 `feat(M5/P1): 加入用途隔离的本地上传与读取签名`。

## Task 3（T3）：磁盘对象与完整上传（A2/A4）

**Files：** A/adapter/local/store.go、object.go、upload.go、reader.go、upload_test.go、object_test.go；domain/storage.go 的共用 MIME 规则；T2 的两处 Store 测试夹具补 Close。

**Interfaces：** 消费 T1/T2；产生 Put/Inspect/Open/Delete/Close，严格使用 spec 的对象布局。

- [x] Step 1：`TestLocalUploadPublishesOnlyCompleteObjects` 用阻塞 reader 在写到一半时 Inspect 必须不存在；完成后 bytes/size/type/ETag 一致。少字节、超额、reader 错、ctx 取消、磁盘拒绝均不发布且本次 tmp 清空。
- [x] Step 2：`TestLocalObjectConfinement` 放外部哨兵、符号链接、非法 key、非普通数据文件；读取/写入/删除不触及哨兵，不挂住。并发相同 key 上传只允许一个完整对象，失败者不能删胜者目录。
- [x] Step 3：运行聚焦测试确认红，使用 os.Root、独立临时目录、SHA-256、元数据版本与原子 rename 实现；所有 ctx/文件句柄归属明确。Delete 缺失幂等，故障保留错误，不能错误吞成成功。
- [x] Step 4：`go test -race -count=1 ./internal/modules/asset/adapter/local` 绿；变异提前发布/忽略 reader 错/跳过 key 格式拒绝必须失败，恢复；外部哨兵与符号链接另验证根目录约束，不能把格式层的变异声称为实际逃逸。检查多个实例共享同配置 root 时不复用临时目录名。
- [x] Step 5：两轮复核后提交 `feat(M5/P1): 实现受根目录约束的完整文件上传`。

## Task 4（T4）：条件封存与共享存储合同（A2/A4）

**Files：** A/adapter/local/seal.go、seal_test.go；A/app/storage_contract_test.go；A/adapter/storagetest/mac.go 复用测试 MAC，架构守卫禁止生产代码导入。不建无生产读者的通用库。

**Interfaces：** Seal(ctx, source, destination, expectedETag)；source 可 u/o，destination 必须新的 o。共同合同不承诺 staging 可重写，不依赖 local ETag 必为 SHA256 来约束未来 S3 ETag。

- [x] Step 1：`TestLocalSealKeepsPublishedBytes`：source ETag 错/缺失不产生目标；测试中将 data 改成同长度不同字节但保留旧 meta，Seal 也必须拒绝；两个不同随机候选并发字节一致，旧 u key 重放不能改变任一 o；o key 不能用 UploadGrant 签发。
- [x] Step 2（store 子合同）：`TestStorageContract` 已验证缺失、Delete/Open/ctx/句柄结束和 opaque ETag；T3 的故障测试区分 I/O 与缺失。真实 HTTP PUT→Seal→GET/HEAD 完整链已在 T5 补齐，A2 仍须阶段最终门禁和审查。
- [x] Step 3：确认新增失败后实现，复制期间不关闭调用方的其他对象、不在读取旧 ETag 之后重新打开可能已换掉的文件。目标已存在为冲突，不覆盖；未知结果的重试由 P2 的候选状态负责。
- [x] Step 4：`go test -race -count=1 ./internal/modules/asset/...`；变异跳过复制后 ETag 核对/允许签发 o 上传，合同均失败，恢复后绿；公开 HTTP 的对应变异留 T5。
- [x] Step 5：两轮复核后提交 `feat(M5/P1): 实现条件封存及存储共享合同`。

## Task 5（T5）：真实 raw HTTP 上传与读取（A3/A6/A9）

**Files：** A/adapter/http/transfer.go、upload.go、read.go 与传输/读取/故障测试；A/adapter/local/transfer.go、reader.go、sign.go、upload.go 的消费合同校正；A/app/storage_contract_test.go。storage fallback 由 Transfer.Register 显式注册，未修改平台 router；A/module.go 与 bootstrap 接线在 T7 一并完成。

**Interfaces：** NewTransfer/Transfer.Register；使用 T2 验签、T3 存储、T4 合同。补齐 T4 延后到本 Task 的真实 HTTP 合同，不以 store 调用替代浏览器协议。

- [x] Step 1：`TestLocalHTTPUploadContract`：正确 PUT 204，缺/错 length、短体/超额、类型不符、重复 key 按 spec 拒绝。无效签名请求不存在和存在 key 响应一致，不能通过错误内容推断存在性。
- [x] Step 2：`TestLocalHTTPReadContract`：GET/HEAD 精确 body/header，前/后/越界 Range、If-Range、304；多段 416，过期 token 带 If-None-Match 仍 403；中文/引号文件名正确编码，CRLF 拒绝，active MIME 不能 inline。
- [x] Step 3：跑真实 httptest.Server 的聚焦测试确认红，再实现；原始 endpoint 不读取 bearer/cookie，不返回内部 metadata/key，错误无路径，写出字节之后失败中止连接。
- [x] Step 4：合同、模块及 `go test -count=1 ./internal/platform/httpserver` 绿；变异验签晚于条件读取、反射 filename header、放开读取 u key 均由断言拦住，已恢复。
- [x] Step 5：两轮复核后提交 `feat(M5/P1): 接入受签名保护的文件传输端点`。

## Task 6（T6）：真实 socket 预算、限流与释放（A5/A7）

**Files：** H/transfer.go、transfer_test.go；A/adapter/http/budget.go、budget_test.go；C/files.go 必要的同职责校正。

**Interfaces：** `(*API).TransferMiddleware(Limiter) func(http.Handler) http.Handler` 只做既有 requestMeta 与传入 limiter；不走 authenticate/publicOperations，不调 JSON deadline/bodyLimit。handler 的并发预算与 idle/total 控制保持在文件传输职责内。

- [x] Step 1：`TestTransferMiddlewarePreservesClientScope` 覆盖受信/不受信代理、IPv4-mapped/IPv6 prefix、独立 bucket、429 Retry-After；复用现有请求帮手，不复制 clientIPs 算法。
- [x] Step 2：`TestLocalTransferDeadlines` 使用实际 net.Listener/httpserver.Server 和阻塞 reader/writer：小全局期限但足量进度上传成功，idle/total/cancel 真正终止 socket；名额满 503，退出路径释放，之后新上传可成功；实际 8 MiB 健康/停读下载均验证。真实阶段证据再运行一次 >30 秒、>1 MiB 的默认期限慢流（T9，仍待完成）。
- [x] Step 3：确认红后实现 spec 的 overflow-safe 预算和 ResponseController；上下文取消解除阻塞 I/O，Unsupported 在写字节前明确失败。全局配置未放宽；GET/HEAD 后台读只设总期限，新上传上限降低后旧对象仍可在最长时限内读取。
- [x] Step 4：`go test -race -count=1 ./internal/platform/httpserver ./internal/modules/asset/...`；移除 Unwrap、名额释放、读 idle、写 idle、取消中断的五项变异均由定向断言杀死，已恢复原实现与原守卫。
- [x] Step 5：两轮复核后提交 `feat(M5/P1): 约束文件传输的期限与并发资源`。

## Task 7（T7）：应用接线与测试存储隔离（A8）

**Files：** B/assets.go、app.go、app_test.go、asset_fixture_test.go、asset_lifecycle_test.go；A/module.go；e2e/fixtures/server.ts、clock.ts 和 stories/smoke/clock-lifecycle.spec.ts；.gitignore；README.md。T6 的 budget_test.go 按职责拆出 budget_fixture_test.go，保持约 400 行文件边界。

**Interfaces：** bootstrap 构造 local 后取得关闭函数放入 app 的明确资源所有权；任何后续错误调用同一清理路径。正常停机 HTTP → jobs → storage → migrator/pool，不能先关 jobs 还可能使用的存储。传输限流复用 bootstrap 的单一 limiter，运行时钟 time.Now，不用 businessClock。

- [x] Step 1：`TestAssetLifetimeFollowsApp` 验证正常 close、后续模块构造失败、重复关闭的定义行为；用可观察 Close 计数/临时目录重新打开验证，不依赖不稳定系统 FD 总数。
- [x] Step 2：测试世界 A/B 相同 key 不共享文件；同世界重启且相同配置/密钥能读取旧 seal；新密钥不能。E2E clock world 明确复用 root，不能以每个日志路径自动变根覆盖重启性质。
- [x] Step 3：实现配置映射和 Register；Go testConfig 使用 t.TempDir，e2e process 默认输出目录并提供可覆盖 root。仅忽略 `/data/files/`、`/server/data/files/` 默认生成目录，不扩大整个仓库忽略范围；补运行目录/持久 key/备份说明。
- [x] Step 4：`go test -count=1 ./internal/bootstrap -run 'TestAssetLifetime|TestAssetMAC|TestApp'` 与既有 clock-lifecycle 页面/夹具测试；无默认目录污染，根 main 不受修改。
- [x] Step 5：两轮复核后提交 `feat(M5/P1): 接线文件模块并隔离各测试世界的存储`。

## Task 8（T8）：整程序合同与失败边界（A1/A9）

**Files：** B/asset_transfer_test.go、asset_contract_test.go、asset_api_isolation_test.go、asset_fixture_test.go 与既有 contract_test.go；H/routes.go 的具名路径树策略、A/adapter/http/transfer.go 的声明；api/storage-local.json；前端 error-messages.ts/test 与 en/zh-CN errors.json、tools/i18n-keys.mjs、turbo.json 的测试输入；docs/v0/M5-files/reviews/P1-local-storage-review.md。

**Interfaces：** 原 /api/v0 精确 route/publicOperations 合同保持；追加 raw route 精确集合，只允许 spec 列出的 PUT/GET/HEAD 与 storage fallback，不许 broad ignore。

- [x] Step 1：`TestRawStorageRoutesAreExact` 核对实际 router.Patterns；新增任意遗漏的 storage 路由会让测试红。无签名请求全部拒绝，未知 storage 路径是 404 Problem，未误入 SPA。
- [x] Step 2：`TestStorageDoesNotWidenJSONAPI` 证明 JSON 仍受原 1 MiB/15 秒控制，认证/结构检查与 no-store 未变；原 health/readiness 和安全头仍成立。
- [x] Step 3：对所需的真实组合断言补最小代码/测试，先红后绿；不把新 asset 的安全策略注入其他业务模块。
- [x] Step 3a：raw 错误目录与源码精确核对，补齐中英文消息；前端消息守卫准确比较 OpenAPI/raw 并集，运行 `make test-web`。T5/T6 首次引入码的同 Task 文案遗漏在首个产品检查点前补齐，保留偏差记录。
- [x] Step 4：`go test -count=1 ./internal/bootstrap -run 'TestRawStorage|TestStorageDoesNot|TestAPIRoutes|TestPublicOperations|TestOperationsThatNeed'` 和架构包；精确路由/隔离失败负例归档。
- [x] Step 5：两轮复核后提交 `fix(M5/P1): 约束签名路径并核对整程序合同`。

## Task 9（T9）：实际慢流、七项门禁与检查点

**Files：** e2e/stories/issue/review-boundaries.spec.ts（门禁复现的既有焦点时序）；B/asset_transfer_slow_test.go、asset_namespace_test.go、asset_transfer_test.go；A/adapter/localprotocol/path.go 与原签发/handler；H/routes.go、path_redirect_test.go；workspace/domain/reserved_slugs.txt、reserved_test.go；web/apps/web/local-transfer.ts、transfer-proxy.ts 与相邻测试、vite.config.ts；api/storage-local.json；M3/M5 设计、spec/plan/review、README；本 worktree `.superpowers/sdd/P1-local-storage` 原件。

**Interfaces：** localprotocol 仅固定本地传输 Root/ObjectPrefix/Path，domain/app 不感知路径；Router 路径策略保持通用。localTransfer(target) 返回 Vite Plugin；具名传输器用 Node http/https 标准库，完整拥有上游与关闭，日志不给任意错误文本。原 /api 仍走已有 Vite proxy。

- [x] Step 1：实际默认 server read=30s/API=15s 下上传 >1 MiB，按进度调节持续 >30s 且小于本次计算预算；成功 204、seal/GET 字节一致。用有界条件/节流器制造慢流，不用 sleep 同步竞争。
- [x] Step 1a：先写真实 storage 工作区/静态资源回归，签发路径约定改为 `/static/local`，观察旧实现失败；迁移 raw 目录、注册、签发/精确检查及相关合同。static 只从 reserved 移入 server，新增保留名字总集合不变断言，原 M3 根路径守卫不放宽。聚焦测试/架构绿后提交。
- [x] Step 1b：将 re-32 路径反例落为通用 Router 的永久测试，先确认 307 红，再按规范化逐段判断归属；编码、重复斜杠、点段、进入再离开以及相邻树控制同时验证。原始/最终路径二点法变异须被杀，恢复后提交。
- [x] Step 1c：真实 Vite + 上游测试先红，再实现具名本地传输插件；精确路径分派、原始方法/头/字节/query、private/no-store、失败日志、断连/取消/关闭均验收；构建/public 占 static 必须失败。使用 Node 标准库避免默认代理错误输出凭证，120s idle 宽于 Go 最大 idle。无新依赖，不关闭全部错误日志；变异泄露错误文本/吞掉取消须被杀，恢复后提交。
- [x] Step 1d：完成工作区与文件请求的联合回归、静态缓存/缺失 chunk/emojibase、API 隔离、race；同步实际偏差和证据，不把设计复核当阶段整审。
- [ ] Step 1e：完整 E2E 复现 C12 的菜单延迟恢复焦点与 F6 竞争，保留 trace；仅让故事等待菜单关闭后的真实焦点后置条件，再验证通知关闭。旧代码重复 10 次中 3 次同因失败；修复后重复验证与完整 E2E 都须通过，不改产品焦点/动画策略，不当作 W11。
- [ ] Step 2：聚焦 race/变异后恢复全部代码，检查 git diff 和工作树；依顺序完成 `make gen && make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`。容器重套件只串行。
- [ ] Step 3：记录命令、退出码、准确 tested diff/HEAD、原件路径/hash/bytes；失败先定根因，环境重跑独立记。仅已知 W11 用既定处理，不专项扩范围。
- [ ] Step 4：提交完整检查点后使用系统 git 推送；按准确 HEAD 查询 branch CI，不以别的提交绿灯代替。

## Task 10（T10）：deep-review、修复与复审

**Files：** 本阶段全部变更与其真实调用/测试；下一编号 docs/reviews/re-NN-M5P1整体审核.md；P1 review。

- [ ] Step 1：按 deep-review 完整读取逻辑、集成、测试三视角，特别复查本 plan 的六条 Review Focus 和 A1–A10；记录局部问题、结构问题和具名测试缺口。
- [ ] Step 2：局部根因自主修复，必要回归/变异并复审；若确有架构级新问题，提交已完成工作和证据再上报，不做旁路补丁。
- [ ] Step 3：修复有产品变化则重跑受影响及完整七项后推最终 HEAD，确认最终分支 CI；补 P1 review 七节：范围、逐条验收、测试、审查问题、plan 差异、交接、集成状态。
- [ ] Step 4：A1–A9 据实勾选，A10 的 main/归档部分留待 T11，不把“实施与整审完成”写成“已合入”。

## Task 11（T11）：noff、main CI、归档与下一段

- [ ] Step 1：核无未知修改、目标 main 与已验收基线关系，按当前授权使用 --no-ff 合并 P1，正常系统 git 推送 main。
- [ ] Step 2：等准确 main SHA 的三个 CI 作业全绿；失败按日志根因处理，W11 重跑仍失败停下报告。分支绿不能替代 main。
- [ ] Step 3：归档所有原件到主目录 ignored 证据区，验证 SHA-256/bytes/mode/membership 与回执引用，未知文件为零后常规删本段 remote/local branch 与 worktree，不 force。
- [ ] Step 4：记录 A10/设计进度与 main 收据，准备 P2 的完整 spec/plan/锁图/预检，再按当前连续授权推进；不关闭尚未完成的 M5 总体标准。
