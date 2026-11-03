# M5 / P3 阶段评审记录

> T1–T8完成；T9原本地七项通过，检查点6fc79a59的CI服务端暴露测试时间精度错误。T10一次整审和两项局部修复完成红绿/主控复审，修复后完整七项及三浏览器均通过；准确最后分支/main CI与归档待核。基线main2d1bcf0c（CI41556463795全绿）；当前codex/m5-p3，见[整审re-37](../../../reviews/re-37-M5P3整体审核.md)。

## 1. 评审方式

预检已进行一次fresh builder check和主控plan自查。实施按executing-plans逐Task/TDD/聚焦验证；T9完整七项，T10按deep-review顺序核逻辑、集成、测试，并以2d1bcf0c..6fc79a59安排一次fresh整分支审查。主控真实复现和修复后复核，不重复派同范围独立复审；完整范围、限制及裁定见re-37。

## 2. 验收标准逐条核对

[spec §8](../specs/P3-s3-storage.md#8-完成标准)的A1–A8已由T1–T8证据关闭；A9本地门禁和审查已通过，等待准确最终分支CI，A10等待main/归档及P4预检。证据矩阵见§18，不用预检探针代替真实Nerve、浏览器和恢复验收。

## 3. 执行中的决定

预检局部裁定为官方SDK/同版测试helper、never-versioned桶、精确origin/静态凭据、真实签名秒适配、S3读no-store及可选local raw接线；每项原因和代价见spec/re-36。T1局部裁定见§8：同一配置规则验证环境/文件值，未启用驱动也拒绝明确冲突，YAML源限制在合并覆盖前执行。T5将XML流量边界置于SDK解析前，数据库/River恢复测试复用PG夹具及独立进程，理由/代价见§12。T8的协议响应上限与错误分类见§15；T9在签名前统一SDK/浏览器端点来源，保留精确origin守卫，原因及部署代价见§16。

## 4. 整分支评审的发现与处理

独立原审1 Important、1 Minor、0 Critical。I1为失败启动测试使用全局资源快照导致并行误报；主控3失败/6通过实红后改为确切资源归属，9/9通过。远端另发现I2恢复断言将插入前纳秒与PG微秒混比，主控注入789ns复现后改用实际持久记录，五轮race通过。P3-M1启动与清理双失败丢原cause为诊断Minor，具名交M5/P8，未声称已修。详见re-37和§17，最终完整套件通过、CI独立核验，见§17。

## 5. 最终实现与plan差异

实现仍沿通过预检的小端口和唯一组合根。按职责新增YAML源校验、签名前期限适配及协议正文边界文件；私有runtimeNow从可选raw依赖中分离，SDK/HTTP不进入domain/app。真实浏览器证明IPv6字面量CSP不支持后提前拒绝字面量；T9统一签名前端点规范。I1/I2仅修测试，生产审计/回收和架构守卫未变。所有局部裁定、代价及排程调整见re-37附录；没有未解决架构分歧。

## 6. 移交事项

P2已完成并归档；P3交付S3/CORS/CSP/真实合同和容量边界。绑定/资料/附件计数和事件仍P4/P5，页面P6/P7，规模/交接收敛P8；P8同时承担P3-M1双失败诊断反例，P1既有焦点Minor仍保留。输入handoff和未来CLI条款继续开放。

## 7. 已知限制

首期只支持明确endpoint/静态凭据/never-versioned普通私有桶及实例独占prefix；无multipart/版本回收/自动后端迁移。已经开始的S3请求可能晚到，靠P2记录与对账恢复；已发读URL到期前仍可能有效。真实固定提供方合同、容量和三浏览器已验收，不代表AWS云/其他厂商或任意公网性能。阶段整审和两项聚焦修复已执行；精确最终branch/main CI与归档仍独立核验。

## 8. T1配置与秘密读取

- 完整S3配置、活动驱动、endpoint/region/bucket/prefix边界、显式来源冲突和错误日志/LogValue遮盖落地；不引入SDK或默认凭据链，S3构造接线仍T2/T6。local不需要S3完整配置、不读取S3秘密文件，但明确的值+文件冲突不能被inactive状态隐藏。
- YAML每一层在环境覆盖前拒绝非空内联秘密，使用既有koanf parser扩展点，无新增依赖；大小写路径也覆盖。文件只读regular、每种上限+3字节封顶用于溢出探测，只去一个尾LF/CRLF，实际值经同一ValidateCredentials验证。文件错误只含键/类别、不带真实路径或内容；没有读取本机实际凭据。
- setup编译RED只作为接口缺失，真正行为RED在task-1-behavior-red.log。额外发现的混合双换行和inactive来源冲突均先由task-1-edge-red2.log复现后修复；此前选择器选中0测试的task-1-edge-red.log不算证明。
- 最终聚焦RED→GREEN及输入hash见task-1-final-green.log/receipt/inputs；完整config/configs race分别2.405s/1.932s，bootstrap秘密与原资源生命周期race3.381s，archtest1.595s。make lint-go的server/tools均0 issues，关键词76规则/14例外无命中，M5/P3没有到期例外被顺延。
- 3个行为变异均命中具体断言且源hash恢复：去掉双来源拒绝、把secret写进日志、把文件限流截成64字节。没有用编译红、0 tests或同一实现计算预期代替性质证明。
- 历史配置回归首次失败为新path_style默认/合法driver预期未同步，以及旧日志断言搜泛词secret误命中新字段名；改为独立数据库秘密sentinel，保留真实值泄露断言。初次lint的De Morgan建议已按结构简化，全部失败原件保留。
- 生成物和模块锁未变；本Task没有声称完整P3七项/branch CI通过，产品检查点仍在T9。A1尚需T2真实启动/桶模式等证据，A1–A10均未提前勾选。


## 9. T2受限客户端与真实测试服务

- 官方SDK及全部9个必要间接模块精确落锁；仅新增设计点名的12个模块，所有原版本未变，go mod verify通过。现有3个测试传递模块按实际类型使用改为直接声明，见spec §2；未安装aws/config或新增凭据发现机制。
- 同一S3格式验证服务部署和adapter直接构造；只有adapter依赖platform/config，未污染app/domain。客户端只用显式静态凭据，官方resolver决定实际bucket origin；path-style、virtual-host、HTTP带点桶/HTTPS回退及显式端口均核到真实请求。
- 独占HTTP transport禁用环境proxy，拒绝同源/跨源301/307，URL和Host双检查；可信TLS未放宽。HeadBucket/GetBucketVersioning在同一个operation_timeout内完成，拒绝Enabled/Suspended/未知/403，500只发一次失败请求；context错误可识别，外部错误不带SDK原文。
- 固定SeaweedFS4.52服务使用独占loopback端口、随机测试凭据/桶及自有tmpfs；readiness真实PUT/GET核字节再DELETE，不把根路由403当可写。真实反例证明版本化删除后9字节旧版本仍存在，构造拒绝Enabled及Suspended。成功和失败启动回收均由Docker的NotFound证实，重复Close安全；容器2秒优雅退出包含在30秒回收预算内，仅丢弃本测试数据。
- 初次正式夹具失败是root0600测试秘密被镜像降权后的seaweed用户拒读；已核对固定源码entrypoint，在自有容器启动前只转交该文件归属，再执行原entrypoint，保持0600和非root服务。失败容器已回收，原始完整日志task-2-first-green.log保留，名字中的green不代表那次成功。
- 测试helper生产导入和子包负例、二进制直接导入负例先实际RED再GREEN，没有豁免架构守卫。整体受影响race：S3适配器16.600s、s3test19.826s、config2.288s、archtest5.107s；server/tools lint均0 issues。
- 4个性质变异分别放行版本化桶、跟随同源redirect、取消origin/Host守卫、暴露SDK原始错误，都命中具体行为断言，源hash完整恢复；加上T1累计7个。最终恢复后具名测试通过（15.747s/19.280s），4183个非文档输入哈希和成员集合前后不变，见task-2-final-green-receipt.json/inputs；未以编译失败或0 tests计数。
- 原件位于当前worktree的.superpowers/sdd/P3-s3-storage/task-2-*，后续按T11整体归档。T2不宣称预签、对象协议、业务装配、F4或阶段整审已完成，T9前仍不推送未完整验收的产品检查点。


## 10. T3精确预签与真实传输

- 新增deadline.go与presign.go，保持既有UploadSigner/ReadSigner小端口。先验证domain期限上界，再向下取整并拒绝已过期的整秒；使用SDK传入的真实signingTime，在官方V4签名之前把Expires改为绝对秒差，不改签后URL、不伪造生产签名时钟。签名前/后与排队跨过期限均受context和期限检查。
- 固定SDK默认方案再次实际观测到requested02:35:51Z、actual02:35:50Z（提前1秒）；精确适配为0秒偏差。永久测试另以独立字面时间覆盖亚秒、跨秒、最短整秒、上传15m/读取65m、越界和取消，不依赖恰巧落在秒边界的随机情况。
- 真实服务验证3字节PUT：错误长度/类型403，正确200；签名包含content-length/content-type/host，公开头只有Content-Type，不伪造Body。改方法、key、query或期限403；实际HEAD大小/类型与真实字节一致。
- GET/HEAD独立签名、UTF-8及引号文件名用标准Content-Disposition编码，普通附件attachment与合法图片inline均验证；返回类型覆盖、长度、空HEAD正文与GET SHA均有独立断言。临近过期GET仍为private,no-store，到期后新GET403；匿名读取和文件名换行注入被拒绝。F4浏览器/CORS仍属于T7，当前没有宣称浏览器验收完成。
- 3个性质变异真实命中：移除长度签名后错误大小PUT变成200；移除类型签名后错误类型PUT变成200；改相对时长舍入后精确到期断言失败。源hash均恢复，累计10个。完整记录为task-3-mutations.json及对应日志，未以单纯检查源文本证明保护。
- 原始行为RED14.904s、首轮具名GREEN20.151s；完整受影响race为S3 35.547s、local7.071s、domain1.634s。首次lint只发现V4导入需显式别名，两处按goimports修正，server/tools最终均0 issues；格式修正后的具名GREEN20.368s，4187个非文档输入哈希/成员集合前后不变，见task-3-final-formatted-receipt.json/inputs。格式前race原件仍保留于task-3-final-green.*，没有把它声称为格式后重新跑过。
- 无新增依赖、schema或公开DTO；阶段验收A2还需T4共享实物合同，T9七项和T10 deep-review仍待后续完成。


## 11. T4共享实物合同与异常处理

- LocalFactory/S3Factory运行同一个runStorageContract，新增同大小不同内容源不能替换已有final的断言；正确封存后删除staging不影响最终字节，GET/HEAD与ETag仍一致。S3实现既有Inspect/Open/Seal/Delete端口，无生产大接口、owner或清理协议变化。
- COPY同时发送CopySourceIfMatch和目的IfNoneMatch=*，不做HEAD后无条件覆盖；真实旧源ETag失败、已存在的异内容final保留，8组同时写同一新目标均只成功一个，最终字节属于胜者。空/通配符/列表/弱ETag不得变成无条件操作；opaque标签保持完整引号，不把ETag当内容摘要。共享合同保留local既有ErrObjectExists及S3 ErrObjectChanged的冲突类别，两者在既有app均映射对象变化。
- 真实删桶后Inspect/Open/Seal/Delete全部返回后端故障，不冒充缺对象；含糊404在同一剩余预算内核实HeadBucket。403/500即使携带NoSuchKey代码也不得映射缺失。SDK正确识别200响应中的COPY Error，不重试或发布成功，也不回显SDK错误原文。
- 返回reader持有操作context，只有Close/取消/超时结束读取；实际慢流读到首3字节后，取消和Close都观察到服务端连接结束，超时保留DeadlineExceeded。先EOF再取消仍返回Canceled，Close可重复。GetObject携带IfMatch并核对返回ETag，拒绝提供方忽略条件的响应。
- 元数据大小/类型/时间/强ETag逐项负例通过。新增33KiB无关响应头反例先RED（0.586s），随后在独占transport设32KiB解析前上限并GREEN（0.574s）；规范化对象元数据仍16KiB上限。没有放宽全局HTTP预算或架构守卫。
- 行为RED为S3 14.899s/app14.163s；首轮具名GREEN21.809s/14.282s。5个实际性质变异移除源条件、目的条件、EOF后的取消检查、Close释放和删桶防线，全部命中具体断言且hash恢复；累计15个，但A8的后续恢复交错与5轮race仍待T5/T8，不提前关闭。
- 最终完整受影响race通过：S3 57.713s、app15.287s、local6.727s、archtest5.200s；4195个非文档输入哈希与成员集合不变，见task-4-final-green-receipt.json/inputs。server/tools lint均0 issues；本轮自有S3容器全部回收，只读清单为空。全部原始日志/变异存于task-4-*，等待T11归档。
- 本Task关闭A2。A3的DB提交故障和完整业务接线仍T6；清单/迟到对象、浏览器、容量、整审和阶段CI未提前标完成。


## 12. T5有限清单、迟到实物与持久恢复

- ListObjectsV2只访问独占prefix并设置MaxKeys；limit1真实跨页、关闭客户端和删除前页后继续原token。严格校验升序、无重复、u/o key、size/ETag/时间、截断/Next一致性、无CommonPrefixes和页上限；异常页明确失败。S3 TemporaryCleaner验证context/cutoff/limit/空token后耗尽，app没有新增driver分支。
- 局部资源裁定：仅清单XML在SDK解析前设64KiB + limit×16KiB预算，兼容chunked正文，读到溢出哨兵即失败；用同一16KiB对象元数据上界及固定协议余量形成有限预算，不能只在SDK分配后检查对象数量。代价是返回过大冗余XML的提供方明确失败；下载正文不受此限制。去掉守卫后96KiB未知节点反例真实转为错误接受。
- 测试专用FaultProxy只转发本夹具，保持已签Host/path/query；通过通道暂停一个指定请求或最后上传字节，后台请求独立限时15s，缓冲测试输入/响应各不超过1MiB，关闭时释放。真实PUT在客户端取消且URL到期后仍落地；真实COPY在客户端取消、目的删除后仍产生原字节。此证据不意味着到期可开始新上传。
- PG清理先以业务+61天证明活候选runtime lease仍保留，再明确快进注入的清理runtime并确认缺对象/移除候选，最后放行实际COPY。迟到key不被DB重新发布，runtime24h相等不删、+1µs删除；已绑定live final字节始终保留。DELETE实际成功但响应被扣住，取消后reclaiming候选仍持久化；新Store重试幂等删除，审计不变。
- River测试真实持久化非空provider token与页内ObjectAfter；停runner、关Store/连接池后，以当前Go测试二进制启动独立OS进程，重新连接同一自有DB/桶继续原job。原cutoff不变，原5个孤立对象收敛；扫描后落在游标之前的低key留给新周期发现，邻prefix的abc实物不变。子进程只收测试生成的stdin配置，不查外部凭据。
- 测试位置裁定：有数据库/River行为的用例放asset/adapter/postgres测试包，直接复用既有PG夹具；受控代理放已受生产导入/二进制禁止规则约束的s3test。避免把PG/River依赖加进存储实现。代价是聚焦命令必须同时选择s3和postgres；原app/清理状态机代码完全未改。
- 清单setup缺方法编译RED不计行为；真实行为RED14.610s→GREEN14.548s。迟到请求首轮18.436s通过，完整恢复第二轮17.593s通过；首次恢复失败来自测试SDK的bytes.Buffer不可seek，改用bytes.Reader，未改变生产逻辑，原件保留。
- 5个独立性质变异分别去掉prefix、重复token拒绝、XML预算、把runtime年龄换成business、在页内中断时提前推进页token，全部命中具体断言并恢复原hash；累计20个，A8的5轮race仍在T8。最终受影响race通过：s3 88.265s、app 15.413s、postgres 24.385s、s3test 19.891s、archtest 5.152s；4203个非文档输入hash/成员集合不变。server/tools lint均0 issues，关键词76/14无命中，自有S3容器已回收。
- 本Task关闭A4；全部原件在当前worktree .superpowers/sdd/P3-s3-storage/task-5-*，T11归档。业务装配、数据库提交故障、F4浏览器、容量与最终整审/阶段CI仍待后续Task，不提前关闭A1/A3/A5–A10。


## 13. T6单一组合根与完整业务闭环

- 驱动选择仅在bootstrap/newAssets；私有assetStore聚合现有消费方小端口，S3创建受限客户端而不读local MAC/目录。newAssets和factory均接收caller context，newS3Assets先验证格式再按T1边界读秘密文件，失败不fallback。asset.Deps显式Files/RuntimeNow与可选Transfer，domain/app没有SDK/config/HTTP新依赖。
- 私有runtimeNow是业务构造的显式来源，生产仍time.Now；原late lease测试从c.transfer.RuntimeNow迁到c.runtimeNow。理由是S3没有local HTTP依赖，业务时钟不应反向依赖传输处理器；代价是未来测试若只改raw时钟，不会再意外改变业务时钟。没有新增生产快进或旁路配置。
- local/S3复用同一保留树注册函数；S3仅保留/static及/static/的404，不注册PUT/GET/HEAD local对象路由。完整router目录按明确driver枚举，没有泛路径豁免；两种驱动调用同一API/public/401检查，原Problem headers/原raw错误目录与保留slug守卫保留。路径规范化穿越不redirect，合法storage工作区创建及项目/设置页面仍200。
- 实际HTTP prepare→S3 PUT→complete→metadata/content→真实GET→duplicate→DELETE全链路核对PG最终key、完整SHA、ETag和原字节；重复确认不改行/审计。S3可以重放staging PUT，但final保持原字节。PG语句级触发器记录metadata/content反复读取为零DML，原S3响应no-store保持。
- DEFERRABLE提交触发器使已封存后的最终DB事务失败：uploaded/final不发布，持久候选及桶对象仍在；撤除故障重试使用新key，旧候选不丢。完整应用关闭后重新启动，由生产RunOnStart River清理旧候选/staging、保留活final；另一闭环以业务+61天清理已删资产实物和引用。测试显式注入runtime+2h模拟lease结束，不声称真实等待了两小时，S3提供方时钟不变。
- 真实封存后暂停返回，分别撤成员权限/删除工作项，最终锁内重验均404且保留候选实物。在暂停点核app pool零占用，用户/工作区/项目/资产行可FOR UPDATE NOWAIT，项目同一单槽gate可完成PATCH，证明外部I/O未持业务连接/锁/gate。原完整local权限、并发与删除系列同时回归。
- 实际从测试生成的*_file读取S3凭据，nil local signing keys亦可成功；取消构造保留context.Canceled，缺失活动秘密文件明确失败且无local根目录。复用原app生命周期测试，正常关闭/重复close和后续模块构造失败都只关闭一次存储，且在HTTP/jobs停止后、PG池之前释放。
- 首个具名RED15.717s为files.driver仍只允许local；扩展全组RED29.921s。中间一轮新增测试unused import导致编译失败，仅保留不计行为证明。首GREEN37.158s，补齐重启/生命周期/完整目录后GREEN42.816s。最终race：bootstrap 162.810s、http 6.883s、httpserver 2.831s、archtest 6.002s，4209个非文档输入hash与成员集合一致；server/tools lint0、关键词76/14无命中，自有S3容器回收。
- 首轮受影响race曾复现原local超时测试仍通过transfer.Files覆盖业务预算，拆分后不再生效；新增实际reader deadline断言先RED7.239s，再将测试改从显式config.Files入口设置1s，生产预算与客户端时限均未放宽。旧失败及聚焦复验原件保留，最终受影响race在修正后重新运行。
- 本Task关闭A1/A3/A5，原件task-6-*保留到T11整体归档；完整七项、CSP/CORS/三浏览器、容量、5轮race与deep-review及阶段CI仍待T7–T10。


## 14. T7浏览器传输、来源策略与部署

- webui.Handler显式Options，只接收存储适配器最终origin；格式校验拒绝通配、多来源、userinfo/path/query/fragment/非法端口。仅img-src/connect-src增加确切来源，local逐字旧policy、inline脚本hash、其他指令及非HTML/API错误策略不变。初始化失败沿原app.close顺序释放，local/S3共用的生命周期测试加入webui失败分支。
- TestStorageCSP行为RED0.525s，真实Chromium旧CSP使第一笔File上传失败；trace确认connect-src self。原始失败归档task-7-f4-red-artifacts。补确切origin后初轮三浏览器3/3（48.2s）。首次回收检查因Docker返回小写no such object而误报，改为明确NotFound类别的大小写兼容；并无遗留容器。以后也不把其他inspect故障当缺失。
- F4由真正Nerve页面origin执行fetch/File，浏览器自动长度，正确PUT200、改类型/大小403；匿名/签名篡改/改HEAD均403。GET下载字节、可见ETag/长度/类型/中文处置名正确；复制新实物key、完整SHA与PG及独立Gohelper流读一致，DELETE后业务读404。raw请求没有业务Authorization或Cookie。错误origin另开浏览器页，读取被CORS拒绝；同URL非浏览器仍读到原字节，明确CORS不撤销bearer能力。
- 第二个测试HTTP origin真实可达且允许CORS；CSP阻止浏览器fetch，服务端零命中。把生产CSP改成通配后，真实浏览器读到第二来源，独立断言Expected true/Received false实红；task-7-csp-mutation.json记录原hash并已还原源码及正常二进制，累计独立性质变异21项（T8仍需核目录与五轮race）。
- 新发现：IPv6字面量URL在无CSP时三个浏览器均读到ipv6-proof，增加精确字面量CSP后均TypeError并报CSP；原受控探针保留task-7-ipv6-controlled-probe.log。config与Options拒绝字面量的用例先RED0.754/1.104s，随后修正；DNS可解析到IPv6，服务端监听不变。代价是IPv6端点需DNS名称，不用放宽CSP伪装支持。设计/spec/部署限制同步。
- Go helper在独立tools模块消费与server完全同版3SDK+9必要间接；只接受loopback和自生成桶/prefix，动作create-bucket/cors/inspect/delete，输入输出各≤1MiB、命令10s，最多100MiB流式SHA，不输出完整文件/凭据/签名。实际未结束输入管道曾超时不退出，新增管道RED1.565s，context关闭自有I/O后GREEN0.636s；假协议测试覆盖精确CORS、数据摘要、超限、redirect和取消，真实协议由F4/世界启动探针验证。
- Node显式使用既有testcontainers12.1.0，无JS SDK。一次非frozen安装意外重解lru-cache和peer图，恢复原图仅增加3行importer，frozen install通过。服务由隔离Node24子进程拥有，父进程90s启动期限、先join再按随机name+label兜底回收；30s关闭预算、命令取消/并发close共享同一promise，世界200s夹具预算。secret文件0600、每世界独立桶/CORS/DB、实际baseURL后设CORS，已有Docker/容器未改动。Makefile测试helper单独构建，生产build不链接它。
- Node生命周期具名3项覆盖正常并发close、Nerve启动失败和child启动中关闭；实际检查loopback端口、容器NotFound、HTTP停止、DB连接关闭、临时秘密删除、关闭后拒绝操作及重复回收。世界级catch聚合原错误与清理错误；底层startS3双失败的原cause缺口由T10登记P3-M1，不把该聚合外推为全部层级。针对oxlint不识别AggregateError第三个cause参数，仅保留一行说明性抑制，没有放宽架构/关键词或lint上限。
- 最终三浏览器3/3（47.7s），Node生命周期3/3（30.3s）。完整config/webui/archtest race、helper race及bootstrap具名S3生命周期/原local生命周期race通过；每次测试的非文档输入hash/集合稳定。前端54检查/knip通过，server/tools lint均0。全部task-7-*原件留T11归档；T9才跑完整七项并首次产品推送。
- deploy/s3提供配置、最低IAM、精确CORS、never-versioned限制、迁移前提和故障排查；AWS HeadBucket需桶级ListBucket，样例诚实说明名称列表权限，名称隔离需独立桶。真实提供方只证明固定SeaweedFS4.52，AWS云及其他品牌未验收。M2交接仅记录CSP部分兑现，绑定/UI/services/River CLI仍开放。

## 15. T8容量、资源故障与协议边界

- 同一真实私有桶完成1B、5MiB、100MiB的签名PUT→HEAD→流式SHA→双条件COPY→最终SHA→DELETE。生成流与独立Python字面摘要对照，没有整文件ReadAll；100MiB的配置传输预算26m50s，测试本身上限2m，未扩大生产operation_timeout。5MiB在发送首块后真实暂停16秒，上传16.052s成功，证明不受普通JSON API的15秒限制。
- 最终race容量记录：1B上传3.692ms、往返17.000ms；5MiB往返16.087s；100MiB上传473.723ms、往返1.015s、211.09MiB/s。100MiB前Go堆1,500,464B、峰值1,872,880B、增量372,416B，Sys峰值15,468,808B。环境darwin/arm64、Go1.27.1、18逻辑CPU、DockerDesktop29.7.2/7931MB、固定SeaweedFS4.52与2GiB自有tmpfs；指标仅此Go进程、非提供方RSS，不承诺公网吞吐。
- 单连接真实socket测试覆盖调用者取消、1秒操作超时和Store.Close：活reader及时报对应context错误，服务端观察断连；已排队请求取消不发HEAD，正常等待者随后可用，新读取返回abc；关闭后拒绝新操作，活跃及idle socket归零。T6的实际PG连接、行锁与gate释放在本Task聚焦恢复组再验。
- 新负例复现启动、COPY和错误正文无统一上限。局部修正在S3独占transport施加普通64KiB XML界限，清单保留64KiB+limit×16KiB；HEAD的Content-Length仍表示对象大小，只有显式Open成功GET豁免。固定长度和chunked的96KiB未知节点均拒绝，不改变全局HTTP/业务接口/清理合同。代价是提供方的超大冗余诊断响应按后端故障处理。
- 首次补限流仍有chunked404失败。固定SDK源码表明io.Copy已返回DeserializationError，适配器却只看外层HTTP状态：增加“已完整解析API错误”的前提后才映射404/409/412。9个真实截断XML的Open/Seal/Delete反例先RED0.602s，防止DELETE把未知结果确认成功；HEAD正常空错误体仍沿SDK状态错误/同预算核桶规则。资源组raceGREEN2.747s，最终容量/资源/业务恢复race为S3 33.667s、bootstrap23.781s，4228非文档输入hash/集合前后不变。
- T1–T7的21项性质变异逐一核实际断言日志与当时正常源码hash；T8再加整文件缓冲、取消普通XML上限、把错误当文件流、跳过解析后分类4项，共25项。整文件缓冲变异的100MiB堆增量252,060,872B，实际命中50MiB门槛。初次在未全绿基线上运行的变异、一次导致nil panic的分类变异均保留且不计数；最终只统计具体断言失败，恢复hash与独立正常输入核对。原件task-8-mutation-audit.json及task-8-final-mutation-*。
- 四组各五轮race全部通过：s3 193.506s、postgres92.530s、bootstrap119.748s；完整路由/401/原raw与S3生命周期race22.374s，架构race6.083s，生成一致、server/tools lint零问题、关键词无命中、自有容器清单为空。所有最终收据为同一4228非文档输入。A7/A8关闭；T9完整七项、T10阶段整审和精确分支/main CI仍开放。

## 16. T9原检查点全量门禁与远端失败

- T8提交为2d289899。首次全量Go命令为了统计用例数使用全局GOFLAGS=-json，意外传给架构测试内部go list，触发“go list -f cannot be used with -json”；这不是产品守卫失效。仅中断本次Go/test进程，108.200s、退出码2和全部日志/输入保留，自有测试会话容器已由其回收。以后不注入该全局参数，以正常make test和实际包结果记录。
- Review Focus1发现标准端点写法尚有缺口：SDK签名器省略默认端口后，原Host/URL精确比较拒绝启动；大小写主机及前导零端口仍进签名，而浏览器会改写。独立wire探针与真实提供方用例均实红；新增永久TestS3EndpointCanonicalOrigin覆盖HTTP80/HTTPS443、大小写、前导零及非默认端口，真实TestS3SignedRequests子例核最终URL和PUT/HEAD/DELETE。
- 局部修正只在配置和适配器：先按原规则验证，然后在SDK解析/签名前规范主机名/端口；不修改签后地址、不放宽origin或Host守卫。TestS3EndpointBrowserHost另以8个反例拒绝被浏览器另释为IPv4或拒绝的数字主机，6个普通DNS/标准IPv4正例保留。Node原生URL解析证据存task-9-browser-url-origins.log；代价是部署需用标准IPv4或正常DNS，不接受短地址、八/十六进制或数字末标签的伪DNS。
- 初次overlay探针的大小写用例受到旧测试dial helper的大小写判断干扰，改为只连独占测试listener后重跑，前后原件均保留；永久用例再在真实提供方实红14.479s，修复后config race1.613s、S3 race15.541s通过。正常全量Go旧代码运行在390.823s主动中断，不能计入通过；新代码的全部七项另以task-9-green-*收据重新执行。第一次lint仅两处De Morgan建议，改为具名布尔值，未放宽检查。
- server与tools各新增同一12个批准SDK模块，所有旧模块版本不变；pnpm-lock仅3行testcontainers importer，无其他升级。83个改动Go/TS文件均≤400行，实际依赖/规模表存task-9-module-audit.json、task-9-source-size.json。T7三浏览器证据仍独立保留，默认E2E不能替代它。
- 七项本地门禁全部通过：生成3.101s、Go lint31.120s、全量Go775.772s、web lint2.850s、knip2.146s、web test0.359s、build+E2E147.214s（345项、Playwright报告2.4m）。前端lint 54项中38缓存，test 18项均命中共享本地Turbo缓存；关键词76规则/14例外、1312翻译键检查另实际运行。Go由make的-count=1执行，未输出子用例计数，不以包数量冒充用例数。全部7份4230个非文档输入逐项等于6fc79a59源码；准确包数/缓存及哈希收据见task-9-seven-audit.json与task-9-green-*。本轮自有S3容器清单为空，原始E2E报告已独立复制；后续远端失败和修复独立记在下文。

以上为原6fc79a59检查点的实测事实，后续远端情况如下：准确[CI41584967958](https://github.com/open-nerve/NerveProject/actions/runs/41584967958)的web成功，server因I2测试精度失败，E2E skipped。Go实际61测试包/21无测试包/0缓存的本地通过不能替代它；修复后原收据也不能冒充新树完整通过。远端raw run/jobs/完整server日志分别保留task-9-checkpoint-*，没有以重跑掩盖该错误。T9后续门禁与T10最终候选共同闭环。

## 17. T10整审修复与主控复核

- I1原全局快照实际并行探针3失败/6通过；新专属资源记录缺失断言先RED16.490s，再9/9 GREEN（31.6s）。夹具在启动Nerve前记录实际随机容器名和秘密目录，失败故事只查本用例资源；不存秘密、不改生产或并发预算。原件task-10-fixture-parallel-probe/ownership-record-red/fixture-parallel-green及各完整浏览器报告均保留。
- I2远端失败在本地用789ns输入确定性重现（15.624s）；修复保留CreateAsset的数据库RETURNING记录，以该持久审计作严格Equal基线，未放宽时间容差。具名race五轮全过（71.119s）；原件task-10-recovery-audit-red/green，五轮不是五个不同性质变异。
- 修复后实际Chromium、Firefox、WebKit F4均通过（3/3、46.7s），每引擎核真实File字节/长度/类型、跨来源、匿名拒绝及SHA；task-10-three-browsers原件与4230输入稳定收据保存。新树完整七项已通过，见下条；最终CI尚未提前通过。
- 独立审者没有审后续修复，后者由主控全文复核三个变更测试/夹具文件、确定失败机制和运行反例。Minor P3-M1仅限测试诊断，依据确定控制流而非实际双故障执行；M5/P8具名承接，不做隐性关闭。

- 修复后最终七项：gen3.311s、Go lint2.172s、全量Go776.227s、web lint4.214s、knip2.062s、web test0.375s、build+E2E146.067s（345/345、2.4m）。Go61测试包/21无测试包/0缓存；web lint54项/36缓存、test18项/18缓存，关键词76规则/14例外与1312翻译键另实际运行。全部七项和新三浏览器的4230个非文档输入完全相同，前后hash与成员集合未变；task-10-final-audit.json、task-10-green-*及各完整报告为原件。自有S3容器只读清单为空。此收据覆盖最后候选的实际代码，准确branch CI仍待推送后核验。

## 18. A1–A10证据矩阵

| 标准 | 证据 |
|---|---|
| A1 配置/秘密/只读启动/依赖 | §8/§9/§13；T1配置源和有界文件、T2真实桶模式/HTTP/TLS/redirect与架构负例，T9模块图 |
| A2 共享实物合同/签名/reader | §10/§11；两Factory、真实绑定和精确期限、实际socket取消，T9端点来源补验 |
| A3 封存/错误/提交失败 | §11/§13/§15；双条件竞争、200内嵌错误、截断错误体、DB提交失败保留candidate |
| A4 清单/进程恢复/双时钟 | §12；实际limit1和token/ObjectAfter、独立OS进程River、runtime24h边界、邻域与迟到实物，§17修复审计断言 |
| A5 完整接线/权限/local | §13；API/PG/S3字节、撤权/删除交错、零DML、锁/连接/gate释放、两driver路由及原local回归 |
| A6 浏览器 | §14及§17；T7三引擎/CSP性质变异、I1后3/3新复验 |
| A7 容量/资源 | §15；1B/5MiB/100MiB流式hash、真实16秒慢流、单连接取消/复用与堆指标；§17专属资源9/9 |
| A8 变异/race | §15；25个真实断言变异、四组各五轮race、失败/恢复hash保留；I2另五轮race |
| A9 七项/整审/准确branch CI | §16原树本地通过及远端失败分别记；§17修复，re-37完整范围；最后HEAD CI待独立收据 |
| A10 main/归档/P4 | T11待实施；原件保留到归档校验，M5总体及后续handoff不提前关闭 |
