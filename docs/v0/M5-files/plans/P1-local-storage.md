# M5 / P1 本地存储实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过，实施中；见[预检记录](../../../reviews/re-31-M5设计与P1预检.md)。负责人已授权按既有节奏连续推进，常规预检通过后继续，不另设确认门。

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

## 文件分工与接口

`A=server/internal/modules/asset`，`H=server/internal/platform/httpserver`，`C=server/internal/platform/config`，`B=server/internal/bootstrap`。

| 文件组 | 单一职责 |
|---|---|
| A/domain/storage.go、storage_test.go | key、grant、object info 与值校验；无 I/O |
| A/app/storage_ports.go、storage_contract_test.go | 消费方小接口；共享行为合同入口 |
| A/adapter/local/store.go、object.go、seal.go | root 生命周期、不可变对象布局、检查/读/删、条件封存 |
| A/adapter/local/sign.go、token.go | 按用途 MAC、规范载荷、相对 URL；不查 DB |
| A/adapter/http/transfer.go、read.go、upload.go、budget.go | raw 路由、验签顺序、标准读取、流量预算和并发 |
| A/module.go | 模块外观、资源生命周期和注册；无运行时服务定位 |
| H/transfer.go | 只复用客户端 metadata/限流，保持 JSON 中间件原样 |
| C/files.go、files_test.go，config.go/validate.go | 精确配置和可测试预算约束，日志不含 root |
| B/assets.go、app.go、asset_transfer_test.go | MAC/配置/限流接线与正常/失败/关闭清理 |
| e2e/fixtures/server.ts、clock.ts | 进程/world 的独占文件目录与重启复用 |

domain 构造函数固定为 `ParseObjectKey(string) (ObjectKey, error)`、`NewObjectKey(kind byte, assetID, objectID uuid.UUID) (ObjectKey, error)`、`ValidateUpload(UploadGrant, int64, time.Time) error`、`ValidateRead(ReadGrant, time.Time) error`。两个 UUID 均由调用方显式提供，构造函数不取随机数；ObjectKey 的 Kind/AssetID/ObjectID 通过方法读出，不向调用方暴露正则细节。

local 外观固定为 `New(Config, Deps) (*Store, error)`；`Config{Root string; SizeLimit int64}`；`Deps{UploadMAC, ReadMAC app.MAC; RuntimeNow func() time.Time}`。Store 实现 spec 的六个小接口和 `Close() error`，另提供传输消费者自己的 `Put(ctx, grant, io.Reader) error`；只能由已验签 handler 或测试内调用，内部仍核 key/type/长度/ctx。签名解析用 `VerifyUpload(key string, rawQuery string, method string) (domain.UploadGrant, error)` 和对应 VerifyRead。错误使用具名值与 errors.Is，不用字符串包含判断。

http 外观 `NewTransfer(Deps) (*Transfer, error)`；Deps 包含 store 所实现的精简读/写/验签接口、预算配置、运行时钟与 HTTP 错误写入器。`Register(*httpserver.Router, func(http.Handler) http.Handler)` 注册精确 raw 集合，包装器由 bootstrap 从 API 的 `TransferMiddleware(httpserver.Limiter)` 取得。不把整套 API 或业务 Authorizer 注入存储。

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

**Files：** A/adapter/local/store.go、object.go、upload_test.go、object_test.go。

**Interfaces：** 消费 T1/T2；产生 Put/Inspect/Open/Delete/Close，严格使用 spec 的对象布局。

- [ ] Step 1：`TestLocalUploadPublishesOnlyCompleteObjects` 用阻塞 reader 在写到一半时 Inspect 必须不存在；完成后 bytes/size/type/ETag 一致。少字节、超额、reader 错、ctx 取消、磁盘拒绝均不发布且本次 tmp 清空。
- [ ] Step 2：`TestLocalObjectConfinement` 放外部哨兵、符号链接、非法 key、非普通数据文件；读取/写入/删除不触及哨兵，不挂住。并发相同 key 上传只允许一个完整对象，失败者不能删胜者目录。
- [ ] Step 3：运行聚焦测试确认红，使用 os.Root、独立临时目录、SHA-256、元数据版本与原子 rename 实现；所有 ctx/文件句柄归属明确。Delete 缺失幂等，故障保留错误，不能错误吞成成功。
- [ ] Step 4：`go test -race -count=1 ./internal/modules/asset/adapter/local` 绿；变异提前发布/忽略 reader 错/允许 key 逃逸必须失败，恢复；检查多个实例共享同配置 root 时不复用临时目录名。
- [ ] Step 5：两轮复核后提交 `feat(M5/P1): 实现受根目录约束的完整文件上传`。

## Task 4（T4）：条件封存与共享存储合同（A2/A4）

**Files：** A/adapter/local/seal.go、seal_test.go；A/app/storage_contract_test.go；必要的测试帮助放 A 下测试包，不建无生产读者的通用库。

**Interfaces：** Seal(ctx, source, destination, expectedETag)；source 可 u/o，destination 必须新的 o。共同合同不承诺 staging 可重写，不依赖 local ETag 必为 SHA256 来约束未来 S3 ETag。

- [ ] Step 1：`TestLocalSealKeepsPublishedBytes`：source ETag 错/缺失不产生目标；测试中将 data 改成同长度不同字节但保留旧 meta，Seal 也必须拒绝；两个不同随机候选并发字节一致，旧 u key 重放不能改变任一 o；o key 不能用 UploadGrant 签发。
- [ ] Step 2：`TestStorageContract` 经真实 HTTP PUT 和未来 T5 注册的 handler 验证完整链（T5 前可分离已有 store 子合同，未接 HTTP 的部分不得标完成）；错误缺失与 I/O 故障可区分，Delete/Open/ctx/句柄结束符合消费合同。
- [ ] Step 3：确认新增失败后实现，复制期间不关闭调用方的其他对象、不在读取旧 ETag 之后重新打开可能已换掉的文件。目标已存在为冲突，不覆盖；未知结果的重试由 P2 的候选状态负责。
- [ ] Step 4：`go test -race -count=1 ./internal/modules/asset/...`；变异移除 ETag 比较/让公开 PUT 接受 o，预期合同失败，恢复后绿。
- [ ] Step 5：两轮复核后提交 `feat(M5/P1): 实现条件封存及存储共享合同`。

## Task 5（T5）：真实 raw HTTP 上传与读取（A3/A6/A9）

**Files：** A/adapter/http/transfer.go、upload.go、read.go、transfer_test.go、read_test.go；A/module.go；H/routes.go 只按实际需要加入明确 storage fallback。

**Interfaces：** NewTransfer/Transfer.Register；使用 T2 验签、T3 存储、T4 合同。补齐 T4 延后到本 Task 的真实 HTTP 合同，不以 store 调用替代浏览器协议。

- [ ] Step 1：`TestLocalHTTPUploadContract`：正确 PUT 204，缺/错 length、短体/超额、类型不符、重复 key 按 spec 拒绝。无效签名请求不存在和存在 key 响应一致，不能通过错误内容推断存在性。
- [ ] Step 2：`TestLocalHTTPReadContract`：GET/HEAD 精确 body/header，前/后/越界 Range、If-Range、304；多段 416，过期 token 带 If-None-Match 仍 403；中文/引号文件名正确编码，CRLF 拒绝，active MIME 不能 inline。
- [ ] Step 3：跑真实 httptest.Server 的聚焦测试确认红，再实现；原始 endpoint 不读取 bearer/cookie，不返回内部 metadata/key，错误无路径，写出字节之后失败中止连接。
- [ ] Step 4：合同、模块及 `go test -count=1 ./internal/platform/httpserver` 绿；变异验签晚于条件读取、反射 filename header、放开读取 u key 应失败，恢复。
- [ ] Step 5：两轮复核后提交 `feat(M5/P1): 接入受签名保护的文件传输端点`。

## Task 6（T6）：真实 socket 预算、限流与释放（A5/A7）

**Files：** H/transfer.go、transfer_test.go；A/adapter/http/budget.go、budget_test.go；C/files.go 必要的同职责校正。

**Interfaces：** `(*API).TransferMiddleware(Limiter) func(http.Handler) http.Handler` 只做既有 requestMeta 与传入 limiter；不走 authenticate/publicOperations，不调 JSON deadline/bodyLimit。handler 的并发预算与 idle/total 控制保持在文件传输职责内。

- [ ] Step 1：`TestTransferMiddlewarePreservesClientScope` 覆盖受信/不受信代理、IPv4-mapped/IPv6 prefix、独立 bucket、429 Retry-After；复用现有请求帮手，不复制 clientIPs 算法。
- [ ] Step 2：`TestLocalTransferDeadlines` 使用实际 net.Listener/httpserver.Server 和阻塞 reader/writer：小全局期限但足量进度上传成功，idle/total/cancel 真正终止 socket；名额满 503，全部退出路径释放，之后新上传可成功。真实阶段证据再运行一次 >30 秒、>1 MiB 的默认期限慢流（T9）。
- [ ] Step 3：确认红后实现 spec 的 overflow-safe 预算和 ResponseController；上下文取消要解除阻塞 I/O，Unsupported 在写字节前明确失败。不能调用全局 server 配置来扩大所有请求。
- [ ] Step 4：`go test -race -count=1 ./internal/platform/httpserver ./internal/modules/asset/...`；移除 Unwrap/名额释放/某期限的变异应被限定测试杀死，恢复原实现与原守卫。
- [ ] Step 5：两轮复核后提交 `feat(M5/P1): 约束文件传输的期限与并发资源`。

## Task 7（T7）：应用接线与测试存储隔离（A8）

**Files：** B/assets.go、app.go、app_test.go、asset_lifecycle_test.go；A/module.go；e2e/fixtures/server.ts、clock.ts；.gitignore；README.md。

**Interfaces：** bootstrap 构造 local 后取得关闭函数放入 app 的明确资源所有权；任何后续错误调用同一清理路径。正常停机 HTTP → jobs → storage → migrator/pool，不能先关 jobs 还可能使用的存储。传输限流复用 bootstrap 的单一 limiter，运行时钟 time.Now，不用 businessClock。

- [ ] Step 1：`TestAssetLifetimeFollowsApp` 验证正常 close、后续模块构造失败、重复关闭的定义行为；用可观察 Close 计数/临时目录重新打开验证，不依赖不稳定系统 FD 总数。
- [ ] Step 2：测试世界 A/B 相同 key 不共享文件；同世界重启且相同配置/密钥能读取旧 seal；新密钥不能。E2E clock world 明确复用 root，不能以每个日志路径自动变根覆盖重启性质。
- [ ] Step 3：实现配置映射和 Register；Go testConfig 使用 t.TempDir，e2e process 默认输出目录并提供可覆盖 root。仅忽略 `/data/files/`、`/server/data/files/` 默认生成目录，不扩大整个仓库忽略范围；补运行目录/持久 key/备份说明。
- [ ] Step 4：`go test -count=1 ./internal/bootstrap -run 'TestAssetLifetime|TestAssetMAC|TestApp'` 与既有 clock-lifecycle 页面/夹具测试；无默认目录污染，根 main 不受修改。
- [ ] Step 5：两轮复核后提交 `feat(M5/P1): 接线文件模块并隔离各测试世界的存储`。

## Task 8（T8）：整程序合同与失败边界（A1/A9）

**Files：** B/asset_transfer_test.go、contract_test.go；H 的现有 body/deadline/security 测试；docs/v0/M5-files/reviews/P1-local-storage-review.md。

**Interfaces：** 原 /api/v0 精确 route/publicOperations 合同保持；追加 raw route 精确集合，只允许 spec 列出的 PUT/GET/HEAD 与 storage fallback，不许 broad ignore。

- [ ] Step 1：`TestRawStorageRoutesAreExact` 核对实际 router.Patterns；新增任意遗漏的 storage 路由会让测试红。无签名请求全部拒绝，未知 storage 路径是 404 Problem，未误入 SPA。
- [ ] Step 2：`TestStorageDoesNotWidenJSONAPI` 证明 JSON 仍受原 1 MiB/15 秒控制，认证/结构检查与 no-store 未变；原 health/readiness 和安全头仍成立。
- [ ] Step 3：对所需的真实组合断言补最小代码/测试，先红后绿；不把新 asset 的安全策略注入其他业务模块。
- [ ] Step 4：`go test -count=1 ./internal/bootstrap -run 'TestRawStorage|TestStorageDoesNot|TestAPIRoutes|TestPublicOperations|TestOperationsThatNeed'` 和架构包；精确路由/隔离失败负例归档。
- [ ] Step 5：两轮复核后提交 `test(M5/P1): 验证文件端点与原 API 合同同时成立`。

## Task 9（T9）：实际慢流、七项门禁与检查点

**Files：** B/asset_transfer_slow_test.go；本 worktree `.superpowers/sdd/P1-local-storage` 的日志、manifest、结果汇总；review。

- [ ] Step 1：实际默认 server read=30s/API=15s 下上传 >1 MiB，按进度调节持续 >30s 且小于本次计算预算；成功 204、seal/GET 字节一致。用有界条件/节流器制造慢流，不用 sleep 同步竞争。
- [ ] Step 2：聚焦 race/变异后恢复全部代码，检查 git diff 和工作树；依顺序完成 `make gen && make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`。容器重套件只串行。
- [ ] Step 3：记录命令、退出码、准确 tested diff/HEAD、原件路径/hash/bytes；失败先定根因，环境重跑独立记。仅已知 W11 用既定处理，不专项扩范围。
- [ ] Step 4：提交完整检查点后使用系统 git 推送；按准确 HEAD 查询 branch CI，不以别的提交绿灯代替。

## Task 10（T10）：deep-review、修复与复审

**Files：** 本阶段全部变更与其真实调用/测试；下一编号 docs/reviews/re-NN-M5P1整体审核.md；P1 review。

- [ ] Step 1：按 deep-review 完整读取逻辑、集成、测试三视角，特别复查本 plan 的五条 Review Focus 和 A1–A10；记录局部问题、结构问题和具名测试缺口。
- [ ] Step 2：局部根因自主修复，必要回归/变异并复审；若确有架构级新问题，提交已完成工作和证据再上报，不做旁路补丁。
- [ ] Step 3：修复有产品变化则重跑受影响及完整七项后推最终 HEAD，确认最终分支 CI；补 P1 review 七节：范围、逐条验收、测试、审查问题、plan 差异、交接、集成状态。
- [ ] Step 4：A1–A9 据实勾选，A10 的 main/归档部分留待 T11，不把“实施与整审完成”写成“已合入”。

## Task 11（T11）：noff、main CI、归档与下一段

- [ ] Step 1：核无未知修改、目标 main 与已验收基线关系，按当前授权使用 --no-ff 合并 P1，正常系统 git 推送 main。
- [ ] Step 2：等准确 main SHA 的三个 CI 作业全绿；失败按日志根因处理，W11 重跑仍失败停下报告。分支绿不能替代 main。
- [ ] Step 3：归档所有原件到主目录 ignored 证据区，验证 SHA-256/bytes/mode/membership 与回执引用，未知文件为零后常规删本段 remote/local branch 与 worktree，不 force。
- [ ] Step 4：记录 A10/设计进度与 main 收据，准备 P2 的完整 spec/plan/锁图/预检，再按当前连续授权推进；不关闭尚未完成的 M5 总体标准。
