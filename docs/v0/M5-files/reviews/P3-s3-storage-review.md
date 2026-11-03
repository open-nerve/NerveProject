# M5 / P3 阶段评审记录

> spec/plan预检通过，T1配置/秘密读取与T2客户端/夹具已完成，T3待开始。基线main2d1bcf0c（CI41556463795全绿）；当前codex/m5-p3，见[re-36](../../../reviews/re-36-M5P3预检.md)。本记录不宣称整个P3或阶段整审完成。

## 1. 评审方式

预检已进行一次fresh builder check和主控plan自查。实施按executing-plans逐Task/TDD/聚焦验证；T9完整七项，T10 deep-review三视角与一次独立整分支审查、局部修复后复核。后者尚未开始。

## 2. 验收标准逐条核对

[spec §8](../specs/P3-s3-storage.md#8-完成标准)的A1–A10尚未逐条关闭；T1/T2证据见§8/§9，A1还需T6的活动驱动装配核对。预检固定SDK/真实服务协议和版本化桶反例只为可行性证据，不提前打勾；准确原件和范围见re-36。

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
