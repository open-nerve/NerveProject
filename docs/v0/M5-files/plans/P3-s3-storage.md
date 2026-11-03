# M5 / P3 S3 存储实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 预检通过，T1–T8已完成，T9原检查点远端失败已复现并局部修复，T10整审修复后完整七项/三浏览器通过，准确最后分支CI待核，见[re-36](../../../reviews/re-36-M5P3预检.md)。worktree为`/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m5-p3`；基线main2d1bcf0c、CI41556463795三job全绿，P2归档清理完成。负责人连续授权常规 M5 推进；新重大风险/分歧先提交再上报。

**Goal：** 为既有资产用例接入真实 S3 私有存储，完成 F4、部署配置和 local 回归。

**Architecture：** asset/adapter/s3 实现现有小端口；SDK/HTTP/错误语义收敛在adapter，驱动和资源接线只在bootstrap；owner、P2候选/锁序/恢复不变。

**Tech Stack：** Go1.27、既有 PostgreSQL/River/Playwright，AWS v2 root1.47.5/credentials1.20.13/service-s3 1.114.6，固定摘要SeaweedFS4.52；详见spec §2/§7。

**Spec：** [P3-s3-storage](../specs/P3-s3-storage.md)，并复核 P2-access/recovery/protected-purge 合同，不重新发明一套。

## Global Constraints

- 主目录永远main；P3使用codex/m5-p3 worktree，基线2d1bcf0c。文档中文，详细中文提交保留feat/fix/docs(M5/P3)前缀；系统/usr/bin/git推送，无force/stash/clean/reset--hard。
- plane/refer只读，不在其内跑git；不动已有Docker/容器，不全局安装；新隔离测试服务仅创建/清理自己的资源。任何容器套件串行跑。
- 手写文件约400行内，按职责拆，不靠生成文件/守卫豁免隐藏新逻辑。domain/app不进SDK/HTTP/config；新生产接口由消费方声明。
- 固定值来自spec：endpoint唯一origin；region1..64，bucket3..63，prefix1..128字节；ID/secret/token256/4096/16384字节，文件最多一个尾LF或CRLF；SDK重试1次；List1..1000、token≤8192；启动90s/停止30s测试夹具；S3只读启动共用operation_timeout1s..10s；文件上限104857600。
- 禁用默认凭据链、代理/redirect、无条件复制、版本化桶；候选≤3/metadata≤16KiB/ETag≤1024，双时钟、7d/配置墓碑期/runtime24h沿用P2。签名绝对到期秒精确相等；S3读响应private,no-store。
- 不提前加UI/绑定/新schema/owner协议。无生产默认S3容器、无自动建生产桶/CORS，无跨driver迁移或fallback。
- 每Task实际记录RED/GREEN、命令退出码、所选测试非0及hash；性质变异必须断言失败，编译失败不计数。结束给中文详细提交，保留全部失败和原件。T9前不把未完成阶段当完整产品检查点推送。
- 依赖首次实际消费才落锁：T2 server三SDK及必要间接；T7 tools同版SDK与e2e既有testcontainers12.1.0显式化。无其他安装。

## Review Focus

1. 虚拟host/带点bucket/显式端口和配置redirect：真实签名origin、HTTP守卫、CSP必须相同，携带凭据不得走第二host（T2/T7）。
2. URL临近过期才使用、SDK跨秒签名和Read EOF后取消：期限精确且reader释放/缓存不越期（T3/T4）。
3. COPY返回200内嵌错误、请求晚到、桶运行中删除：不得把错误当发布/回收成功，不丢candidate（T4/T5/T6）。
4. limit1页内失败后恢复、邻域key/乱序/旧token、对象在扫描后落地：不漏长期孤立对象、不误删live字节（T5）。
5. 浏览器禁止手设长度、CORS拒绝不代表非浏览器不能用URL、100MiB/低速取消/启动失败：实物断言和资源所有权需真实成立（T7/T8）。

## 文件结构与固定接口

- config新增`S3FilesConfig`与`FilesConfig.S3`，字段对应spec，PathStyle bool；秘密文件读取在bootstrap `loadS3Credentials(config.S3FilesConfig) (s3Credentials,error)`，返回三字符串，仅该helper读文件；config只验证源/格式与遮盖。
- 新`asset/adapter/s3/{store,config,transport,errors,presign,inspect,read,seal,delete,inventory}.go`，按职责拆；Config{Endpoint,Region,Bucket,Prefix,AccessKeyID,SecretAccessKey,SessionToken string; UsePathStyle bool; SizeLimit int64; OperationTimeout time.Duration}，Deps{RuntimeNow func()time.Time}；`New(context.Context,Config,Deps)(*Store,error)`、`Origin() string`、`Close() error`。
- 方法完全沿用app签名：SignUpload(ctx,domain.UploadGrant)(domain.SignedRequest,error)、SignRead(ctx,domain.ReadGrant)(domain.SignedRequest,error)、Inspect(ctx,domain.ObjectKey)(domain.ObjectInfo,error)、Open(ctx,key,etag)(io.ReadCloser,error)、Seal(ctx,source,target,etag)(domain.ObjectInfo,error)、Delete(ctx,key)error、List(ctx,token string,limit int)(app.ObjectPage,error)、SweepTemporary(ctx,cutoff time.Time,limit int,token string)(next string,removed int,err error)。不向app提供大Storage接口。
- 新测试专用`platform/s3test/{service,bucket}.go`：`New(t testing.TB)*Service`自有90s启动/30s回收；`Service.NewBucket(t testing.TB)*Bucket`返回Endpoint/Region/Name/AccessKeyID/SecretAccessKey，prefix由每世界生成；`Bucket.Client()*s3.Client`只用于测试实物断言。测试helper不是产品导入；真实数据面ready使用独立测试key后清理，不污染生产构造的只读约定。
- bootstrap newAssets改为`newAssets(context.Context,config.FilesConfig,*identity.Keys,*slog.Logger)(assetComponents,error)`；factory与调用者同步。assetComponents内部store聚合现有小端口，transfer为`*assethttp.Deps`可空，origin string、runtimeNow func()time.Time；close归原app所有。
- asset.Deps新增`Files config.FilesConfig`和`RuntimeNow func()time.Time`，Transfer改可空`*httpadapter.Deps`；New仅在非空时建raw handler，Register仅在其存在时注册；业务用例统一取Files/RuntimeNow。
- webui `Options{StorageOrigin string}`、`Handler(fs.FS,Options)(http.Handler,error)`；`contentSecurityPolicy(index []byte,origin string) string`只接受已验证origin，调用者处理构造错误。

## Task 1（T1）：活动驱动配置与显式秘密

**Files：** 修改config/files.go、load.go、configs/config.yaml、tools/keywords.json；新增config/files_s3.go、files_s3_yaml.go、files_s3_test.go、bootstrap/assets_secrets.go、assets_secrets_test.go；同步configs/embed_test.go和原配置测试的合法驱动/默认值。

**Interfaces：** 产出上述S3FilesConfig和loadS3Credentials；不读取SDK/外部存储，local默认不读S3秘密文件。

- [x] Step 1：写TestFilesS3Config/TestS3CredentialSources RED：`local+空S3=>valid`、`s3+缺endpoint=>error`、ID/secret/token边界及+1、双来源、YAML非空秘密、未知键、prefix越界、日志不含值/文件路径；文件空/目录/不可读/超限、LF/CRLF成功。
- [x] Step 2：`cd server && go test ./internal/platform/config ./internal/bootstrap -run 'Test(FilesS3Config|S3CredentialSources)$' -count=1`；先确认具名失败；缺类型编译RED后仍补行为RED，不把编译失败记性质变异。
- [x] Step 3：按spec §3实现配置/默认path_style/只读秘密helper，keys phase=M5/P3并核到期例外；保持原loader未知键/null拒绝和日志遮盖。
- [x] Step 4：同命令GREEN，再完整config、bootstrap秘密/生命周期相关测试；分别移除双来源拒绝、遮盖检查做2个实际性质变异并恢复hash；make lint-go。
- [x] Step 5：记录证据与`feat(M5/P3): 增加显式 S3 配置及受限秘密读取`提交。

## Task 2（T2）：SDK客户端、桶能力和自有测试服务

**Files：** 新s3/store.go、config.go、transport.go、errors.go及对应_test；新platform/s3test/service.go、bucket.go、service_test.go；server/go.mod/go.sum；config/files_s3.go导出同一纯格式验证；archtest/rules_test.go、rules_cases_test.go、binary_test.go明确将新s3test纳入测试专用包限制，生产导入负例保留。

**Interfaces：** 产出New/Origin/Close与s3test Service/Bucket；使用T1值映射，生产只读构造。HTTP测试注入只在包内，不给运行配置加跳过TLS/Dial开关。

- [x] Step 1：TestS3Startup、TestS3TransportBoundary、TestS3ServiceLifecycle断言未版本化成功，Enabled/Suspended/403/未知失败、一次总预算超时、取消关闭；设HTTP_PROXY仍只直达准许origin，301/307不能触发第二server；path-style/virtual-host/带点桶/端口解析与实际请求一致。
- [x] Step 2：先运行具名RED；首次消费上述精确SDK版本更新server模块，必要testcontainers-go由已有v0.44.0间接转直接但不升级；保存go.mod diff。
- [x] Step 3：按官方resolver、显式StaticCredentialsProvider、独占transport和RetryMaxAttempts=1实现构造；错误只留操作/类别/status及可识别context，不泄露SDK/url.Error原文；fixture固定digest、随机数据、真实就绪及失败回收。
- [x] Step 4：`cd server && go test ./internal/modules/asset/adapter/s3 ./internal/platform/s3test -run 'TestS3(Startup|TransportBoundary|ServiceLifecycle)$' -count=1` GREEN；版本化旧字节反例、redirect越界和秘密泄露3个性质变异；受影响race、archtest与lint。
- [x] Step 5：提交`feat(M5/P3): 接入受限 S3 客户端与真实存储夹具`，记录镜像与模块校验；不把只读启动检查称为全部协议证明。

## Task 3（T3）：精确期限和浏览器可用的预签请求

**Files：** 新s3/presign.go、presign_test.go、presign_integration_test.go；必要独立deadline.go；不改app端口/domain上界。

**Interfaces：** SignUpload/SignRead返回existing SignedRequest；内部HTTPPresignerV4委托官方v4.Signer，签名前计算Expires秒，不篡改signingTime。

- [x] Step 1：TestS3PresignDeadline/TestS3SignedRequests真实PUT3B：wrong type/size=>403、正确=>200/204；SignedHeaders含length/type/host、公开Headers只有Content-Type；实际X-Amz-Date+Expires等于ExpiresAt。跨秒/极短/已到期/65m边界，改方法/key/query/期限全部拒绝。
- [x] Step 2：运行`cd server && go test ./internal/modules/asset/adapter/s3 -run 'TestS3(PresignDeadline|SignedRequests)$' -count=1`记录RED及默认SDK早1秒反例。
- [x] Step 3：实现签发、mime.FormatMediaType文件名、GET/HEAD响应覆盖类型/处置/`private, no-store`；检测context和排队后到期。
- [x] Step 4：同命令GREEN；近到期GET仍no-store，HEAD无body且长度实际一致，匿名拒绝；移除长度、类型签名和期限适配3个性质变异，按hash恢复；受影响race/lint。
- [x] Step 5：提交`feat(M5/P3): 提供绑定长度类型和绝对期限的 S3 预签请求`。

## Task 4（T4）：实物读写与同一共享合同

**Files：** 新s3/inspect.go、read.go、seal.go、delete.go；测试按object_real_test.go/object_wire_test.go/read_fault_test.go拆分；更新errors.go及独占transport响应头上限；修改app/storage_contract_test.go，新增app/storage_s3_contract_test.go复用runStorageContract，保持原local测试。

**Interfaces：** Inspect/Open/Seal/Delete；Open读者拥有ctx取消函数直到Close，不能defer在Open返回前取消；内部元数据/错误转换用于T5。

- [x] Step 1：TestStorageContract/S3Factory与TestS3ObjectFailures：真实缺对象/缺桶、403/500/timeout、坏metadata、旧ETag、EOF后cancel；源A/目标B同大小不同hash、目的并发已有、源改变均不覆盖final。用受控服务响应注入200内嵌COPY错误，断言没有成功结果。
- [x] Step 2：`cd server && go test ./internal/modules/asset/adapter/s3 ./internal/modules/asset/app -run 'Test(Storage.*Contract|S3ObjectFailures)$' -count=1`先RED。
- [x] Step 3：实现SDK实际HEAD/GetObject IfMatch/CopyObject双条件/Delete；HEAD无内容404用同一剩余期限HeadBucket区分桶；Seal不合法/空etag不发无条件请求；copy后HEAD目标。
- [x] Step 4：同命令GREEN；取消读者关闭连接而非只测ctx字段；移除源条件、目的条件、reader取消/Close及把缺桶当缺对象各1性质变异；完整受影响race及lint。
- [x] Step 5：提交`feat(M5/P3): 实现条件封存和流式读取的共享实物合同`。

## Task 5（T5）：有界清单、晚到文件和恢复

**Files：** 新s3/inventory.go、metadata_body.go、inventory_test.go、inventory_wire_test.go、recovery_integration_test.go；复用inspect元数据验证与transport响应上限；真实数据库恢复放postgres/s3_recovery_test.go与s3_recovery_process_test.go，受控代理在测试专用platform/s3test/fault_proxy.go。P2清理状态机不改。

**Interfaces：** List(ctx,token,limit)、SweepTemporary(ctx,cutoff,limit,token)原签名，调用现有清理用例；原ObjectAfter仍由app拥有。

- [x] Step 1：TestS3Inventory/TestS3LateObjects：limit1真实多页、邻prefix不动、token中断恢复、空/重复/倒序/非法key/超限token失败；temporary空token耗尽，非法参数拒绝。受控延迟真实PUT/COPY在客户端退出后落地，candidate或unknown实物仍可恢复；业务快进不提前跨runtime24h。
- [x] Step 2：`cd server && go test -p 1 ./internal/modules/asset/adapter/s3 ./internal/modules/asset/adapter/postgres -run 'TestS3(Inventory|LateObjects|Recovery)$' -count=1`记录清单行为RED；迟到/恢复为复用既有状态机的集成验证，其区分度由Step4真实变异证明。所有延迟有清晰时间/同步点而非盲sleep。
- [x] Step 3：实现严格finite ListObjectsV2、opaqueNext、元数据验证及S3无tmp适配；清单XML在SDK解析前限制为64KiB + limit×16KiB（含未知节点/chunked正文），不限制下载流；使用既有River/P2恢复装配，不新增provider游标状态表。
- [x] Step 4：GREEN并做prefix移除、Next不前进/页内恢复、runtime改business三类性质变异；真实重启后DB/桶实物均收敛，旧key不重用；race/lint/archtest。
- [x] Step 5：提交`feat(M5/P3): 接入有限对象清单和迟到文件恢复验证`。

## Task 6（T6）：单一组合根与完整业务闭环

**Files：** 修改bootstrap/assets.go、app.go、asset/module.go及现有factory测试；新增bootstrap/assets_s3.go及asset_s3_fixture/lifecycle/recovery/lifetime_test.go；抽出asset/adapter/http/storage_namespace.go共用保留树；修改bootstrap/contract_test.go、asset_contract_test.go与asset_transfer_test.go的明确driver目录。

**Interfaces：** 应用前述context版newAssets、assetStore聚合、Files/RuntimeNow/可选Transfer；既有owner/读写Tx/gate只注入一次。

- [x] Step 1：TestAssetS3Lifecycle/TestAssetS3Recovery：HTTP prepare→raw PUT→complete→metadata/content→实物GET→duplicate→delete→cleanup，实际PG及hash；S3构造不创建local目录/raw route；local完整旧路径。deferred DB提交错误时candidate和对象仍可对账；撤权/宿主删除/锁外阻塞与零DML断言继续有效。
- [x] Step 2：`cd server && go test ./internal/bootstrap -run 'TestAssetS3(Lifecycle|Recovery)$' -count=1`记录RED；完整route/public/Problem目录的两driver实际差异明确列出。
- [x] Step 3：只重组组合根和模块可选raw依赖，ctx传进构造；scope、业务事务、owner、SQL和River状态不变，启动失败及app.close沿原所有权释放。
- [x] Step 4：同命令GREEN，再原Asset prepare/read/complete/duplicate/deletion/cleanup/resources/contract及httpserver/archtest，race验证实际连接/gate释放；不新增放宽守卫。
- [x] Step 5：提交`feat(M5/P3): 以单一组合根接入 S3 资产生命周期`。

## Task 7（T7）：CSP、部署和真正浏览器 F4

**Files：** 修改webui/handler.go、csp.go及测试，新增webui/options.go，修改bootstrap/app.go及生命周期测试、config/files_s3.go及边界测试；新增tools/s3fixture/{main,request,operations}.go及测试、e2e/fixtures/s3.ts、s3-process.ts、s3-container.ts、s3-world.ts与smoke/s3-lifecycle.spec.ts、e2e/stories/files/f4-s3-storage.spec.ts、e2e/playwright.m5-s3.config.ts、deploy/s3/{README.md,cors.example.json,policy.example.json}；修改tools/go.mod/go.sum、e2e/package.json、pnpm-lock.yaml、Makefile。

**Interfaces：** webui Options/Handler如上；Node `startS3():Promise<S3Service>`、`createS3World(info:TestInfo):Promise<S3World>`，S3Service创建测试桶/CORS与close，World拥有Nerve/API/DB/桶/prefix和close。world fixture预算200000ms覆盖S3启动90s/停止30s、既有nerveFixtureTimeoutMs70s和10s余量，不以Playwright默认30s截断其自身回收。Gohelper只JSON stdin/stdout动作create-bucket/cors/inspect/delete测试数据，单次10s、1MiB输入输出；读实物只返回size/hash/headers，不能输出完整100MiB内容。Node使用独立测试子进程执行testcontainers，父进程限定启动并在join后按确切name+label兜底回收；业务content只有GET，浏览器验证GET200/改HEAD403，正向HEAD签名仍归T3。

- [x] Step 1：TestStorageCSP RED：local逐字原policy，S3仅img/connect增加唯一origin，非法source拒绝。F4先真实浏览器RED，记录File/Blob自动长度、错类型/大小403、download字节和CORS/CSP失败；上下文不继承API令牌到raw请求。
- [x] Step 2：实现Options校验与bootstrap实际origin接线；helper同版SDK，testcontainers12.1.0显式化；Makefile新增build-s3-fixture测试目标并作为e2e前置，不把测试helper塞进生产build；Nodefixture自有启动失败/close并发安全，实际baseURL后写本测试CORS；不动生产bucket。
- [x] Step 3：`cd server && go test ./internal/platform/webui ./internal/bootstrap -run 'Test(StorageCSP|AssetS3Lifecycle)$' -count=1`、`cd server/tools && go test ./s3fixture -count=1`、`make lint-web knip` GREEN；所有Handler调用者同步。
- [x] Step 4：`make build build-s3-fixture`后`cd e2e && pnpm exec playwright test --config=playwright.m5-s3.config.ts`三浏览器F4；默认CI主配置纳入Chromium F4，专用配置三浏览器补验；CSP通配变异实际浏览器访问第二origin才红。真实错误origin只能断言浏览器读取被CORS挡，不能断言URL能力失效。
- [x] Step 5：配置/最低IAM/never-versioned/错误排障/迁移前提/真实支持品牌和原始失败写入部署说明及handoff兑现部分；提交`feat(M5/P3): 完成 S3 浏览器传输和精确来源策略`。

## Task 8（T8）：容量、故障和守卫闭环

**Files：** 新s3/capacity_test.go、fault_test.go；局部修正metadata_body.go、transport.go、read.go、errors.go，普通协议正文限流及错误解析后分类；补前述bootstrap/fixture资源测试、阶段review证据索引。

**Interfaces：** 只使用已有接口和fixture；不提供绕过预算/签名的生产测试开关。

- [x] Step 1：TestS3Capacity1B5MiB100MiB/TestS3Resources：流生成/增量hash，100MiB不能在Go进程整体ReadAll；记录基线/峰值内存、吞吐、实际环境。慢流在operation_timeout或caller取消后及时结束，等待者/reader/idle连接/DB/gate可再次使用。
- [x] Step 2：`cd server && go test ./internal/modules/asset/adapter/s3 ./internal/bootstrap -run 'Test(S3Capacity1B5MiB100MiB|S3Resources|AssetS3Recovery)$' -count=1`；若新增性质先RED后实现GREEN，不用调大范围掩盖失败；100MiB需配置原传输最大预算覆盖。
- [x] Step 3：确认T1–T7至少15性质变异已实红（双秘密源、秘密日志、签名长度/类型、期限、源条件、目的条件、prefix、缺桶错误分类、versioning、redirect、reader、恢复cursor、CSP、时钟），原hash全恢复；缺项补有杀伤力测试，不计编译红。
- [x] Step 4：`cd server && go test -race -p 1 -count=5 ./internal/modules/asset/adapter/s3 ./internal/modules/asset/adapter/postgres ./internal/bootstrap -run 'TestS3(ObjectFailures|LateObjects|Recovery)$|TestAssetS3Recovery$'`记录所选每组5轮且无死锁/竞争；架构/生成/完整路由守卫通过。失败含环境中断也保留原件，所有测试容器已正常清理。
- [x] Step 5：更新A1–A8证据、边界和代价，提交`test(M5/P3): 验证容量边界和存储故障恢复`。

## Task 9（T9）：全量门禁和首次产品检查点

**Files：** spec/plan/阶段review、必要输入handoff及M5进度；Review Focus1局部根因修正store.go/transport.go/config/files_s3.go，新增endpoint_origin_test.go/endpoint_host_test.go并扩展真实签名用例、部署说明；不改未到期消费者。

- [x] Step 1：串行`make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`；保存输入hash/命令/退出码/实际test数及缓存情况，失败根因修正后只重验需重验部分，最终证据对应同一代码树。
- [x] Step 2：核生成一致/工作树/依赖实际导入、规模与临时资源、五个Review Focus均有证据；对CSP/三浏览器另附T7不能以默认E2E代替。
- [ ] Step 3：中文提交并系统git push；连接GitHub核准确SHA的server/web/e2e，失败主动取日志根因处理；已知W11仅按既有规则，记录20m job预算中实际新增成本，不盲调时限/无限重跑。
- [x] Step 4：补阶段review七节及边界/原件索引，转整审；不能以这次检查点CI替代最后HEAD。

原检查点6fc79a59的CI41584967958服务端因测试纳秒/微秒预期错误失败；原件保留，T10修复后最终候选同时闭环T9 Step3/4，不能把原检查点写成通过。

## Task 10（T10）：deep-review、一次独立整审与修复

**Files：** docs/reviews下一编号M5P3整体审核、阶段review及真实发现文件。

- [x] Step 1：deep-review逻辑/集成/测试三视角顺序完整读基线至候选；手写实现与测试逐项记录覆盖，依赖/生成物按源和锁验证，明示未读范围。
- [x] Step 2：按executing-plans/requesting-code-review要求一次fresh整分支审查，主控逐项裁定；局部Important/Critical做真实RED/GREEN修复并主控复审，Major新分歧提交后上报；不反复派同范围独立复审。
- [ ] Step 3：新产品变化完成必要受影响/全量门禁；逐条A1–A9证据，最终文档提交push，精确最后HEAD CI全绿。不能仅以代码前一提交绿声称最后HEAD绿。
- [x] Step 4：记录所有裁定、代价、Minor具名去向和准确范围；不提前勾选A10/M5总体完成。

## Task 11（T11）：集成归档及下一阶段

- [ ] Step 1：核全绿HEAD/干净worktree与main关系，已授权`--no-ff`合并main、系统git正常push；主目录保持main。
- [ ] Step 2：准确main SHA的三job全绿；分支绿不替代main。异常按根因，重复已知W11停报。
- [ ] Step 3：归档全部raw/失败/审查/变异/浏览器产物，hash/bytes/mode/membership校验、未知0；正常删除远端/本地分支和worktree，清理后再验归档。
- [ ] Step 4：补P3完成与A10，准备P4绑定字段/一次创建/成员资料的spec和plan及预检，普通推进连续授权有效；M5总体完成项保持未勾选。

## 预检覆盖自检

| spec要求 | 任务 |
|---|---|
| §2结构/依赖/所有权，§3配置/HTTP/单版本 | T1/T2/T6；A1/A5 |
| §4签名/reader/COPY/错误/删除 | T3/T4/T6；A2/A3 |
| §5双时钟/分页/后到数据 | T5/T6；A4 |
| §6路由/CSP/CORS/F4，§7test fixture/helper | T2/T6/T7；A5/A6 |
| §8容量/变异/门禁/整审/集成 | T8/T9/T10/T11；A7–A10 |
| §9代价/部署/交接及精确CI | T7/T9/T10/T11 |

11 Tasks。主控自检需逐项确认spec覆盖、每步可执行、前后类型一致、五项Review Focus归属及篇幅；builder check只审完整spec，不另派plan reviewer。
