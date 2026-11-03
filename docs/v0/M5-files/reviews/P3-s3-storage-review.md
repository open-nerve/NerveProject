# M5 / P3 阶段评审记录

> spec/plan预检通过，T1–T4配置、客户端、预签与实物合同已完成，T5待开始。基线main2d1bcf0c（CI41556463795全绿）；当前codex/m5-p3，见[re-36](../../../reviews/re-36-M5P3预检.md)。本记录不宣称整个P3或阶段整审完成。

## 1. 评审方式

预检已进行一次fresh builder check和主控plan自查。实施按executing-plans逐Task/TDD/聚焦验证；T9完整七项，T10 deep-review三视角与一次独立整分支审查、局部修复后复核。后者尚未开始。

## 2. 验收标准逐条核对

[spec §8](../specs/P3-s3-storage.md#8-完成标准)的A2已由T3/T4证据关闭，其余尚未逐条关闭；T1–T4证据见§8–§11，A1还需T6的活动驱动装配核对。预检固定SDK/真实服务协议和版本化桶反例只为可行性证据，不提前打勾；准确原件和范围见re-36。

## 3. 执行中的决定

预检局部裁定为官方SDK/同版测试helper、never-versioned桶、精确origin/静态凭据、真实签名秒适配、S3读no-store及可选local raw接线；每项原因和代价见spec/re-36。T1局部裁定见§8：同一配置规则验证环境/文件值，未启用驱动也拒绝明确冲突，YAML源限制在合并覆盖前执行。

## 4. 整分支评审的发现与处理

尚未进行产品整分支deep-review，不把builder check的nothing important left当产品“零问题”。

## 5. 最终实现与plan差异

T1按职责新增files_s3_yaml.go，秘密文件读入后复用config.ValidateCredentials与同一上限常量；同步原配置默认值/驱动测试。阶段仍在实施，无最终实现差异结论，未留待裁定项。

## 6. 移交事项

P2已完成并归档；P3接续S3/CORS/CSP/真实合同，M0超时handoff只记录本阶段实际兑现部分。绑定/资料/附件计数和事件仍P4/P5，页面P6/P7，规模/交接收敛P8；不提前关闭输入handoff和未来CLI条款。

## 7. 已知限制

首期只支持明确endpoint/静态凭据/never-versioned普通私有桶及实例独占prefix；无multipart/版本回收/自动后端迁移。已经开始的S3请求可能晚到，靠P2记录与对账恢复；精确提供方的Go探针不代表全部厂商或浏览器已验收。永久测试/容量、三浏览器和精确最终branch/main CI仍待实施。

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
