# M5 / P4 阶段评审记录

> 预检通过，T1–T6字段/绑定/读取/清理及用户、工作区API聚焦回归完成；T7待实施。分支codex/m5-p4，基线main5807a0d6e369c486e98e4cb00389d8b309322221；P3已全绿归档。三份spec/15任务plan和[re-38预检](../../../reviews/re-38-M5P4预检.md)为当前依据。

## 1. 评审方式

一次fresh builder check无重要未决项；主控检查spec覆盖、类型/路径、五项Review Focus、步骤可执行性和篇幅。实施后T14按deep-review逻辑→集成→测试，另做一次fresh整分支终审；现在未进行产品终审。

## 2. 完成标准

主spec A1–A9均未勾选。生成/lint/既有迁移/架构基线通过，只证明准备环境与已有守卫，不替代P4实际功能、并发、清理、七项门禁或CI。

## 3. 规划裁定与代价

- owner事务内Lock→At→owner写→Apply，asset不接owner授权或任意闭包；代价为重读同一已锁集合，换取新项目一次绑定及统一时刻。
- 全部图片响应锁内CPU签发，账号首锁、目标资料账号不逆序锁；代价为短事务持锁，完整集合容量需T12验证。
- 有效邀请链接只签它固定workspace的logo；已签bearer期限不变，不扩大任意UUID读取。
- 四条反向FK即时NO ACTION且精确目录验证，三个owner先条件解除，再由asset删除；有限页增加少量接口协作，遗漏路径必须失败。
- builtin素材和前端控件P6，organization_size仅收紧已有响应枚举；不引入未经设计点名的依赖。

## 4. 整分支发现与处理

T1–T6已实施，整审及修复待T14。不能把builder check的无重要问题写成代码无缺陷。

## 5. 实现与plan差异

T1按计划实施。T2接口核对发现SaveAsset有意不写scope/bound_at，改为资产内部专用BindImage，保留恢复写的范围和Revision保护；不扩owner接口或放宽架构守卫。新增小端口的代价是多一个固定SQL路径，具名真实数据库反例必须覆盖。

## 6. 移交事项

F5–F7页面、29张预设素材及URL缓存属于P6；issue/comment/draft属于P5；整体故障/规模和输入交接P8。P3-M1双故障诊断、P1焦点Minor仍具名保留到P8。M5总体及输入handoff未关闭。

## 7. 限制与证据

P4预检原件为本worktree `.superpowers/sdd/P4-profile-images`，P3规划探针已入P3冻结归档；完整路径和命令/时间见re-38。当前字段与内部绑定已在分支实现，未声称P4公开API、真实图片或后续页面已经上线。


## 8. T1所属字段、纯规则与SQLC

三owner迁移00035–00037加入四个nullable UUID和cover_builtin_id；即时非deferred NO ACTION、无条件索引、精确cover-01..29及互斥CHECK。纯domain补三态Set/可空值和封面选择规则，sqlc查询/行映射保留所有现有方法签名；project INSERT携带最终封面，仍未接公开绑定API。

真实RED：旧库缺全部五列、合法cover-01失败；领域错误ID/双来源未拒绝、选择一侧未清另一侧；三个仓储遗漏读写字段。GREEN覆盖P3存量行升级不变/默认null、四FK实际23503、全部迁移Up/Down/Up、三态字段/列表/返回映射和同事务回滚。首轮全量迁移因第二次Up预期仍34失败，同步为37后通过，未放宽目录核对。

最终八包无缓存全绿（34.114s）：migrations3.574s、archtest1.696s、identity/workspace/project domain0.233/0.177/0.170s、三个postgres包6.295/13.508/5.810s；具名新测试实际运行。Go lint server/tools均0 issues（28.890s）。make gen已生成且重生成输入不变；首次gen-check因生成文件尚未提交退出2，按Makefile原规则在提交后复核，不放宽守卫。原件task-1-schema-red/cover-red/fields-red/final-green及receipt/inputs、task-1-generation、task-1-gen-check、task-1-lint均在阶段workspace。

P4整体A1–A9暂未关闭。反向清理/完整程序目录T4、公开API T5–T9及全部门禁T13仍待实施，T1不能当作阶段完整交付。

## 9. T2原子图片绑定

新增小端口ImageBindings和专用BindImage SQL，owner持有事务和父锁时一次升序锁齐全部旧新资产，统一取得At，写所属引用后再Apply。新候选先全批隐藏判定，再用途/上传/绑定状态；当前引用幂等不写，管理员可替换旧申请者的已绑定图，旧软删审计保持。new_project仅在最终绑定SQL迁入已插入的项目，SaveAsset仍不改申请者/父引用/bound_at。

应用行为RED 1.541s、真实PG RED 3.344s；SQLC nullable At首轮编译不符单独保留，不算行为证据。GREEN真实触发器在第二件绑定时失败，四件资产与用户两字段全部回滚；无事务调用拒绝、Revision/用途/状态拒绝、项目先插入再绑定、SaveAsset边界均验证。最终11项具名主测试、两包带race全通过（8.087s），完整asset/app race亦通过（16.907s，含既有local/S3合同）。lint静态布尔形式提示修正后server/tools均0 issues（17.024s），不改变规则。

四项实际行为变异（申请者、完整范围、7天、升序锁）均被具名断言检出，每次精确SHA恢复，证据task-2-mutations.json及四份日志。最终4249个源输入hash稳定；生成物提交后按原gen-check复核。证据在阶段workspace的task-2-*，公开owner接线和真实内容仍归T5–T11。

## 10. T3批量图片读取

资产端只消费owner给出的四类引用：先拒绝同ID不同宿主/用途，再去重升序按最多1000个ID取锁；全部块查询完成后才CPU签发。准备/失效/不匹配引用返回缺项，真正数据库/解码/签名错误原样返回。共享P2 contentExpiry，零UPDATE且不依赖业务时钟，不接对象I/O或owner回调。

行为RED 1.228s；8项具名主测试（6个新读取测试及2个既有内容测试）带race GREEN 2.519s。四用途、缺项/原错、1000/1001分块、重复去重、后一块失败零签名和读取审计不变通过。bound/完整范围/同ID冲突三项变异都实际断言失败，恢复SHA一致；4252源输入核对稳定。这里的Signer为编排替身，真实local/S3内容仍由T11验收，未提前宣称实物成功。原plan的ResolveContent测试名修正为实际AssetContent，避免漏跑原期限验证。


## 11. T4反向引用清理、目录及测试资源回收

asset页先有界读取候选，再在同一事务依identity→workspace→project的具名端口按owner主键升序锁行并条件清当前引用，最后有序锁/复核/删除资产。失败保留输入游标，删除0行仍按扫描进度前进。四个即时NO ACTION反向FK、完整索引与三个实际解除步骤精确核验；原六个资产父引用、CASCADE祖先与未知23503/40P01负例保持拒绝。已有递归删除目录仅在额外证明全部成立后排除四条已核对OID的引用边，无表名前缀/通用NO ACTION豁免。

持久化RED4.495s，真实双向FK阻止旧流程；初次SQLC歧义和fatalProbe接口错误分别修正，原失败记录保留。最终具名回归覆盖四列、审计不变、DELETE故障整页回滚、候选后换新图、并发purger删0前进、61/59天与metadata空边界，以及真实PID观察的users→workspaces→projects→file_assets顺序。九项变异均由行为/目录断言检出：漏avatar解除、漏identity步骤、无条件清新引用、放宽metadata、删0终止、asset先锁形成阻塞、SET NULL、owner倒序、asset倒序；每轮原件SHA精确恢复。测试只证明内部存储与锁；API前置与进程恢复仍在T11/T12。

T4首次两轮数据库执行因Docker启动/端口映射/套接字故障未进入业务断言。负责人随后明确要求测试容器用完清理。本轮盘点194个容器，其中174个为已退出或未启动、无复用标记/宿主绑定/命名卷的一次性Testcontainers，全部逐ID核验后正常rm -v；其中仅5个能从本项目日志直接归属，另外169个未证明属于本项目，记录保持这一区别。本次按负责人最新清理指示处理已核闲置资源，不把这次操作扩大为后续可清理其他会话容器的长期授权。20个开发/业务容器原ID保留；没有prune、重启Docker或变更镜像。

真实故障测试进一步复现pgtest在容器已分配但启动失败、以及模板数据库初始化失败时遗留容器。局部修复保留失败容器句柄，在失败返回前关闭连接池、用独立有界context删除自己的容器和匿名卷，清理错误与原因合并保留；成功时仍沿原共享容器/Ryuk生命周期。两条RED均实际泄漏，GREEN均在函数返回前确认清除。此修复响应本轮资源清理要求；不改生产数据库、容器布局或P3的S3故障合同。后续按本轮session/精确ID核对回收，不做全局清理。

证据均在阶段workspace：task-4-purge-red、task-4-final-focused（28个具名主测试，race20.140s）、task-4-final-arch-pgtest（完整架构与pgtest，race22.931s）、task-4-pg-cleanup-red/green、task-4-mutations.json及每项日志；格式修正后的具名race回归42.422s（task-4-green-4；其中archtest因过滤无命中，不计架构证据），随后完整archtest/pgtest再次全绿22.335s（task-4-full-arch-pgtest），最终4270源输入稳定；lint16.299s零问题。提交后生成核验见task-4-gen-check。Docker逐ID盘点、排除项、删除回执及后核验保存为docker-*.json，不将环境失败计为产品RED。


## 12. T5用户资料的原子图片与真实读取

GetMe与UpdateMe使用明确消费依赖，自己账号/凭据先锁，锁下读取当前引用；更新一次收齐两图旧新ID，用资产最后锁后的同一At写用户和资产，响应签发在commit前。省略保留、null解除、当前ID重复不写资产；PublicProfiles只新增头像ID事实，不签发，成员出口由T7继续。HTTP使用生成三态UUID和真实URL/到期对，既有四个资产错误码声明并沿用已有中英文文案；M2设计同步说明读取事务边界。

真实API行为RED4.300s：旧PATCH拒绝新增图片字段。GREEN经准备→真实PUT→complete→双图PATCH→GET字节，核URL的精确签名过期、引用/bound/审计、重复与替换/清空；他人、过期、删除、用途、pending及两字段隐藏优先均拒绝且数据库不变。真实owner、第二资产、Signer、deferred commit四种故障使用户和全部四资产快照整体回滚，故障移除后同请求成功。GET审计触发器证明不写用户/资产；失效引用只回答空URL/到期。app层另验顺序、At、事务上下文和端口原错/零返回，不能代替真实回滚。

最终4279个源输入逐hash稳定：完整identity及archtest带race66.301s；四个完整程序图片测试、七项路由/错误合同及三个既有PAT账户测试带race20.869s；受影响Web五文件64测试1.934s、完整web类型9.470s、Go lint1.934s均通过。生成物按Makefile在提交后再核gen-check。首次测试数据漏新字段、PublicProfile结构转换、HTTP新错误码覆盖和Web依赖未构建的失败均保留；没有把编译或准备失败当产品RED。旧HTTP精确快照与所有User测试数据已按新协议更新，无as any/新增依赖。

阶段workspace保存task-5-api-red、task-5-final-identity/bootstrap/web/types/lint及receipts/inputs、task-5-gen-check；docker-task-5-reaped.json核从日志得到的22个自有容器全部消失，20个已有开发/业务容器保留。完整七项与分支CI仍留T13；本任务不声称整个P4、页面或S3图片故事完成。


## 13. T6工作区图标与目录读取

UpdateWorkspace以CallerLock为第一把锁，原W(N)/权限判定后读当前Logo，经T2统一绑定与T3响应签发。新管理员可替换前任已绑定图，不能挪用前任准备件；当前ID/省略不写资产，null解除。Get/List在候选预读后，以账号N、候选W(S)升序锁、活成员/权限重验、资产批量签发构成短事务；只返回已锁且仍可见集合，保留名称/id排序和整集合，不追新工作区。批量锁复用本模块已有精确SQL，通过新消费小端口调用，无跨模块查询。

OrganizationSize响应改为原枚举或null，生成Go/TS类型直接使用；六值创建、未填null及非法/显式null请求拒绝均实际通过。API真实Logo上传、管理员更替、member/guest403、非成员与不存在同404、重复/替换/清除、Get/List/Update实际字节和签名秒精确验证。审计触发器证明Get/List不写业务行；签名失败令Logo、名称和候选绑定整体回滚，重试成功。app验证候选撤权、新加入、删除、slug复用、排序及全部新增读取错误/commit零返回；三个行为变异（去候选过滤、去升序、去slug身份复核）实际失败且SHA恢复。完整数据库交错仍归T10，未用替身冒充并发证据。

行为RED4.268s，旧API拒绝Logo字段。首次范围键拼错、排序函数拼错与非法请求错误使用会校验请求的call均为夹具/编译失败，原件保留；后者改用已有只校验响应的发送入口，仍断言真实422/400。最终4287源输入hash稳定：完整workspace和archtest带race64.562s，完整程序用户/工作区图片与七项合同带race31.873s，受影响Web五文件57测试1.803s、完整web类型9.665s、Go lint1.516s均通过。Workspace测试工厂补必有null字段，无类型断言绕过。提交后gen-check另核。

证据为阶段workspace的task-6-api-red-2、task-6-final-*、task-6-mutations.json/逐项日志、task-6-gen-check及receipts/inputs；docker-task-6-reaped.json核本任务12个自有容器全回收、20个既有容器保留。成员头像、接受邀请与公开预览仍待T7；全部七项、远端CI、终审和集成未提前关闭。
