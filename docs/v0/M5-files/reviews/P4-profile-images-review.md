# M5 / P4 阶段评审记录

> 预检通过，T1–T12实现和验收完成，T13七项与准确产品CI已全绿，T14整审通过、最终文档CI待核。分支codex/m5-p4，基线main5807a0d6e369c486e98e4cb00389d8b309322221；P3已全绿归档。三份spec/15任务plan和[re-38预检](../../../reviews/re-38-M5P4预检.md)为当前依据。

## 1. 评审方式

一次fresh builder check无重要未决项；主控检查spec覆盖、类型/路径、五项Review Focus、步骤可执行性和篇幅。实施后T14按deep-review逻辑→集成→测试，另做一次fresh整分支终审；T14的一次fresh整审已完成，主控已逐文件SHA复核，见[re-39](../../../reviews/re-39-M5P4整体审核.md)。

## 2. 完成标准

T13逐项核对主spec A1–A7及T1–T12实际证据后已勾选；A8仍等待完整七项、整审和准确最终分支CI，A9仍等待集成、main CI、归档清理和P5预检。

## 3. 规划裁定与代价

- owner事务内Lock→At→owner写→Apply，asset不接owner授权或任意闭包；代价为重读同一已锁集合，换取新项目一次绑定及统一时刻。
- 全部图片响应锁内CPU签发，账号首锁、目标资料账号不逆序锁；代价为短事务持锁，T12千引用已实测，完整集合仍有短事务持锁成本。
- 有效邀请链接只签它固定workspace的logo；已签bearer期限不变，不扩大任意UUID读取。
- 四条反向FK即时NO ACTION且精确目录验证，三个owner先条件解除，再由asset删除；有限页增加少量接口协作，遗漏路径必须失败。
- builtin素材和前端控件P6，organization_size仅收紧已有响应枚举；不引入未经设计点名的依赖。

## 4. 整分支发现与处理

T14独立整审与主控复核未发现Critical/Important；维护项P4-M1由P8死成员/接口清理承接，完整范围和限制见[re-39](../../../reviews/re-39-M5P4整体审核.md)。

## 5. 实现与plan差异

T1按计划实施。T2接口核对发现SaveAsset有意不写scope/bound_at，改为资产内部专用BindImage，保留恢复写的范围和Revision保护；不扩owner接口或放宽架构守卫。新增小端口的代价是多一个固定SQL路径，具名真实数据库反例必须覆盖。

## 6. 移交事项

F5–F7页面、29张预设素材及URL缓存属于P6；issue/comment/draft属于P5；整体故障/规模和输入交接P8。P3-M1双故障诊断、P1焦点Minor仍具名保留到P8。M5总体及输入handoff未关闭。

## 7. 限制与证据

P4预检原件为本worktree `.superpowers/sdd/P4-profile-images`，P3规划探针已入P3冻结归档；完整路径和命令/时间见re-38。当前T1–T12字段、完整公开API与真实图片/清理规模已在分支验收；阶段尚未集成，P6页面尚未实施。


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


## 14. T7成员头像与邀请Logo

成员列表和改角色回答采用已授权的公开资料事实，批量签当前头像；目标账号不在W后追锁，停用资料与guest邮箱null规则保持。接受邀请在原成员恢复/cascade之后、提交之前签Logo，签名失败则全部回滚。公开预览先校验MAC，错误token实测零数据库查询，只签固定工作区Logo；任意额外asset_id查询仍被既有路由忽略，不能选择别的资产。已忽略链接继续有效，已撤回/接受/工作区删除均404。

锁核对发现原access合同误把撤回邀请视为W(N)：真实是W(S)+邀请U，非成员收件人停用还可不锁W。因此本模块预览补W(S)→固定邀请S→asset，不改写侧协议或引入账户逆序锁；M3/M5设计及spec/plan已同步。代价是操作同一邀请需等短签名事务。真实PID/SQL证明预览先到时撤回与非成员停用等待，撤回先提交时预览404；去掉邀请锁的行为变异失败，恢复后通过。

旧S1(b)测试假设停用持有账号后成员修改还能先持W，与已批准的图片响应CallerLock首锁冲突，原10秒失败保留。更新为观测users等待、停用提交后修改401且不进入W写入，Alice角色保持；不放松生产锁。原项目加入(c)维持到T9处理。旧账户型夹具只验证真实账户锁和原连带规则，不冒充完整凭据验收，完整HTTP凭据双向在T10。

行为RED4.421s；成员/邀请聚焦race27.181s；完整workspace+arch race66.985s；最终bootstrap成员/邀请/旧并发/个人及工作区图片/七项API合同race26.759s，完整arch补验7.399s；Web6文件64测试1.873s、全量types10.745s、最终lint16.313s。签名失败真实500且成员/项目成员/邀请/资产逐行快照不变，恢复签名后成功；忽略接受邀请签名错误的变异也被实际断言抓到。共两项变异hash恢复。

原workspace/Web/types通过后只改bootstrap旧并发测试，产品输入不变；按输入范围复用对应结果，补跑bootstrap/arch/lint，核差异仅该测试文件（task-7-final-input-reuse.json）。不为无关测试注释/夹具重复全量容器运行。证据task-7-*、receipt/input、mutation与docker-task-7-reaped.json保留：本任务22个自有容器均已回收，20个既有容器保留。提交后生成核验3.109s通过；T8–T15、阶段终审与集成当时仍待完成。


## 15. T8项目封面与一次创建

CreateProject在无锁权限预检后，以调用者账号、工作区和所需成员锁确定创建资格；最后一次资产锁后的同一At用于项目、成员、偏好、六个默认状态和绑定审计。INSERT一次携带最终上传引用或稳定builtin，随后迁移本人new_project准备件；签发与commit仍在同一事务。UpdateProject预检后进入issue/asset共用的同一项目gate，再取账号及原W/P锁；权限和归档先于候选资产检查，锁下当前引用参与完整old/new集合。只为返回图片的这两个入口接线，原label/state锁协议不变。

真实API证明cover-01/29、省略/null/非null、互斥及非法builtin、原申请者准备件归属、其他管理员替换既有绑定、当前ID幂等和私有隐藏404。一次POST即产生完整2成员/2偏好/6状态并读回实际文件字节；states、Apply、签名和延迟commit四种故障均500且全部业务/资产快照不变，原准备件仍可重试。Update签名失败也整笔回滚。HTTP声明四项资产错误及update的server_busy，生成Go/TS与旧消费者真实null字段同步。

行为RED4.567s；扩展聚焦race17.310s；完整project+arch race133.605s。旧gatedAuthorizer屏障在新增无锁预检误暂停并再次close，原失败保留；仅将夹具限定真实事务内暂停，保留原锁强度、PID/阻塞和角色结果断言。最终bootstrap新封面/旧项目锁交错/七项API合同race34.792s、完整arch7.425s、lint16.682s通过。Web5文件104测试2.183s及全量types11.515s通过；这两项与完整project只复用唯一bootstrap测试夹具变化之前的结果，逐路径hash差异记录在task-8-final-input-reuse.json，受影响bootstrap/arch/lint均重跑。

证据task-8-*日志/receipt/input与docker-task-8-reaped.json保留，4303输入稳定；12个自有临时容器全回收、20个既有保留。提交后生成核验3.105s通过。T9其余项目响应及T10–T15联合验收、整审、集成尚未完成；A1–A9保持未整体关闭。


## 16. T9全部项目响应的封面授权

Get/List先预读可见项目候选，无连接进入与issue/asset同实例gate；专用于图片读取的ImageReadLocks明确账号N→W(S)→候选P(S)升序，重读原权限/完整集合后批量签发。沿本模块ShareIssueProjects已有排序SQL，不另造查询或跨模块依赖。只返回候选内仍可见的项目，保留展示顺序和归档过滤，新增留到下次，slug改指另一工作区不能沿旧锁签发。Archive/Unarchive/Join只在原写流程前接CallerLock，最终项目签发在commit前；原label/state和其他不返回图片的动作不变。

真实API覆盖Create/Update/Get/List/Archive/Unarchive/Join的实际文件字节和期限、私有无权与不存在同形、guest可见、归档过滤及已删排除。Get/List执行期间项目和资产statement级审计计数0、全图快照不变；Get/List/Archive/Unarchive/Join五项签名故障均500，写动作全图回滚。预检隐藏目标不进gate；候选增删/反序、slug身份、各端口原样失败、提交失败不给部分DTO、gate释放均有独立测试；取消候选过滤/锁排序/slug同一性/退gate四个实际行为变异被抓到并恢复hash。查询的server_busy及Retry-After:1在API和HTTP用例同步声明验证。

原M3三项并发夹具实际失败：S1(c)加入及停用/加入四格不再能越过新的账号首锁先等W；同调用者archive/update也先在账号串行。保留原日志，前两者改验证真实users等待、停用后401且无越过W的写及成员结果，第三项改由两个已有管理员写同一项目，继续验证真实项目锁串行和archive后update409。原W/P强度、双向先后和角色/审计断言保留；误写错误码unauthenticated的夹具修正为现有unauthorized，原失败也保留，不当作产品缺陷。

行为RED10.825s；真实响应/故障聚焦race17.228s；最终完整project+arch race133.472s、bootstrap相关旧M3锁交错/新图片/七项API合同race52.864s、全量lint16.614s通过，4307输入一致。证据task-9-*日志、receipt/input、4项mutation及docker-task-9-reaped.json；自有容器均回收，既有20个保留。提交后生成核验3.126s通过。T10–T15、A1–A9整体关闭、阶段终审和集成仍待完成。

## 17. T10真实授权与锁交错

九个顶层测试各五轮race通过（169.309s）。12格账号reset/deactivate与三类图片、10格W撤权/删除及P私有化/撤权/归档、8格成员头像替换/停用和调用者UUID两侧、6格邀请撤销/接受/非成员收件人停用、2格同准备件双创建，均观察请求自身PID、实际SQL、等待类型与blocker链并验证结果。15个返回图片的认证入口在父/资产表同时被阻塞时首先等待users。失败操作保留准备件和业务图；已失效凭据之后为401，撤权之后不新签发，取消后连接及共用gate可再用。

隐藏件与不存在件的code、完整body及除Date外所有headers一致；双图输入反序UUID仍优先隐藏判定。真实上传验证I/O计数器已接通后，七个只读图片出口正常CPU签名、存储I/O增量0，users/profiles/workspaces/projects/file_assets的statement审计0、完整快照不变。成员头像继续沿M3公共资料事实，不在W之后锁目标账号。

去掉账号首锁、加目标账号逆序锁、去掉锁内owner重新授权三个变异均有真实行为RED并恢复hash；没有因这些验证改变生产协议。共享gate夹具最初错误地用Bob请求Alice已绑定封面的普通content接口，按P2合同应先隐藏404；改用Bob自己的同项目附件，确认与图片/工作项共用gate，三类请求均503且Retry-After:1。原失败保留，不算产品RED。

最终五轮输入4316稳定；lint只要求一个import分组空行，保存前后hash和逐token一致证明后复用行为结果，另跑lint/架构守卫。完整门禁、双驱动实物、清理恢复、整分支审查和集成仍待T11–T15；不提前关闭A1–A9。

最终lint16.016s零问题、完整架构race7.153s通过。本任务18个自有容器均已回收，20个既有容器保留，清单docker-task-10-reaped.json。提交后生成核验3.169s通过。

## 18. T11真实文件故事与双驱动合同

F5–F7三个Playwright API故事全部通过（4.820s，执行3.6s）：个人双图在重新认证后仍按同一申请者绑定；替换/移除及混合隐藏件拒绝不留半笔；管理员更替后可保留原管理员绑定的当前图标，旧管理员失权；成员头像与有效邀请预览/接受返回实际字节，邀请消费后失效；一次项目POST建立最终封面及完整默认图，重复准备件创建失败无新项目，cover-29与自定义图互换、非法预设拒绝不写。每个故事都校验实际GET字节/SHA、DB引用/范围/bound和审计，并在拒绝前后比完整业务/资产快照。

TestImageStorageContract将同一组业务请求跑在local和真实S3：四类图片共8个子组，真实prepare/PUT/complete、绑定/读取/替换/移除和延期提交故障全部运行；额外覆盖新项目INSERT最后提交失败。API回答与持久owner字段/资产范围一致，new_project迁移到最终项目，两个驱动的实际字节SHA均与metadata一致，期限与真实签名一致。两驱动均仅由API创建账号、工作区和项目及成员，不使用旧issue夹具的成员store写入；SQL只作断言或具名提交故障。最终race25.984s通过，不以fake Storage代替实物。

构建12.088s通过（11项build中2项缓存，未把build缓存称测试通过）；E2E类型检查2.234s、最终Go lint15.713s、E2E lint1.321s/format0.453s通过。类型检查最初误用member_id，按生成合同改为member.id；静态检查要求time.Until，均为测试局部修正，原失败保留不算产品RED。最终source4322；build/types之后只修改不被它们消费的Go测试文件，逐路径记录此唯一差异，新的Go契约和lint以最终输入重验。

Playwright report/test-results复制保存在task-11-e2e，所有task-11日志、输入hash/receipt及容器ID保留；后续全量E2E不会覆盖本次原始证据。F5–F7仅关闭API部分，页面/素材/缓存仍在P6，完整P4门禁、清理恢复、整审与集成待T12–T15。

完整架构race6.518s通过。Go和Playwright日志明确创建的11个自有容器均已回收，20个既有容器保留，见docker-task-11-reaped.json。提交后生成核验3.135s通过。


## 19. T12保留期、真实进程恢复和冷统计容量

TestImageReferenceRetention先经真实注册/建工作区/建项目、prepare/PUT/complete及owner PATCH建立四类引用。16件同时覆盖61天empty、59天empty、61天nonempty和活件：精确让四个final key删除失败，保留资产恢复行、四个owner引用和工作区/项目父行，同时无关eligible页继续前进；恢复实际存储后解除→资产→父链收敛。每次物理清理batch1，活件/59天件的完整owner与资产JSON不变，解除不改owner审计；额外独立证明老父不能CASCADE年轻资产。空对象先实际删除字节才清metadata。用户没有保留旧引用的公开删除入口，因此用户墓碑+旧引用为具名持久化故障，年龄/安全戳也是时间夹具，不声称存在用户公开物理删除API。初次具名GREEN7.606s。

TestImagePurgeProcess的两个场景分别在首个真实资产页提交后暂停、第二页真实DELETE触发器报错后退出OS进程。另一PID启动原River worker、用River JobRetry继续同一个job，读取原持久metadata：首个after不回退、不跳页，四引用和资产没有半笔；新进程业务时刻前移30天，仍沿原cutoff，59天和活邻域快照不变，最终父链清完。正常续跑与失败续跑7.182s通过。第一次夹具只接受retryable，但当前River对短重试使用available，导致17.597s失败；依据依赖源码改看持久errors和合法调度状态，生产实现不变，原失败保留。

容量夹具通过API上传并绑定1000件，再增加到1001件。首次5秒超时定位到第一个ListProjects，尚未取资产锁或签名。pg_stat_activity显示CPU执行且无锁等待；诊断EXPLAIN ANALYZE也30秒未完。冷统计plan估计各表只有1行，把全成员/偏好组合反复对每project扫描；手工ANALYZE后的对照执行1.674ms。局部修复只改project所属SQL：偏好连接锚定p.id/current user，并保留m.id存在前提，非成员sort_order仍null；不改权限、排序、接口、架构守卫、依赖或请求期限。未经手工ANALYZE的真实GET随后通过：1000件14SQL/292ms，1001件15SQL/295ms，签名次数分别1000/1001，存储I/O为0，后一块锁未取得前签名为0，取消后连接/gate可重用。诊断失败和执行计划原件保留于task-12-capacity-*；最终race和完整项目回归另核。

本任务新增uploaded、binding-bound、创建业务图后提前commit三项变异，均实际触发原用例的状态/原子性断言并精确恢复源SHA。累计31项变异逐份核原始日志，不计编译/启动失败或空匹配，覆盖全部15项要求：申请者、scope、bound、uploaded、7d、old/new排序、账户首锁、目标账号逆序锁、owner授权、同UUID混合引用、创建事务、忽略签名失败、反向FK/漏列、盲清当前ref、删0结束页。对应具体用例、日志与恢复hash集中在task-12-mutation-matrix.json；缺列另有T4 avatar-detach负例，不仅靠FK改SET NULL的目录测试。

最终4327项源输入逐路径hash相同：完整asset app/postgres、project全部包、workspace app与arch带race通过172.781s；21项bootstrap具名验收及双驱动合同带race150.787s；四项清理/换进程交错各五轮race52.343s；成员/邀请签名失败回滚补核4.618s；Go lint16.278s零问题。最终容量带race为1000件14SQL/611ms、1001件15SQL/560ms，取消后再读558ms。25个日志可明确归属的测试容器全部回收，现存20个与T11保留集逐ID一致；无全局清理。提交59c8e84a后的生成核验3.206s通过，task-done已核所有最终收据与输入一致。完整七项、准确CI与阶段整审仍由T13–T15关闭。


## 20. T13完整门禁与检查点

本地七项均通过，4329项最终源输入在各受影响门禁运行前后保持一致。原始命令、输出、耗时与逐文件SHA位于阶段workspace的对应log/receipt/inputs；verify-task-13逐一拒绝未解释的输入漂移。

| 门禁 | 命令耗时 | 退出码 | 证据前缀 |
|---|---|---|---|
| 生成核验 | 3.083s | 0 | `task-13-green-gen` |
| Go静态检查 | 1.421s | 0 | `task-13-green-lint-go` |
| 完整Go测试 | 923.616s | 0 | `task-13-green-go` |
| 前端静态/类型/格式 | 17.543s | 0 | `task-13-last-lint-web` |
| 依赖检查 | 1.885s | 0 | `task-13-last-knip` |
| 完整前端测试 | 38.273s | 0 | `task-13-final-web` |
| 构建与完整E2E | 144.386s | 0 | `task-13-last-e2e` |

Go以GOFLAGS=-p=1串行容器包、-count=1且非-short执行：61个实际测试包、21个无测试文件包，无测试结果缓存。四项历史opt-in规模测试未设置开关，巨大计数键探针、S3子进程入口和skip传播负例的跳过有具名说明，不能把默认make test算作这些规模验收；P4容量独立由T12真实1000/1001引用证明。前端316文件/3398个Vitest测试全部通过，另有API-client的5个Node原生测试通过，18个Turbo任务无缓存；前端静态54任务也无缓存。完整E2E354例均通过、无跳过/重试掩盖；两次构建各11任务命中构建缓存，不算新增测试证据。

首次全量E2E342通过/6失败：五项旧删除检查将新增图片反向引用当成业务归属边，遍历到users/workspaces/projects；W5的严格预览响应期望漏了新增null期限字段。两项真实PG目录RED证明错误遍历和CASCADE未被拒绝，随后精确校验四条schema/table/column/target/NO ACTION/非延迟/无条件前导索引，再仅按已证明OID从归属图排除；未知关系仍拒绝或继续遍历。W5保留完整toEqual。聚焦17故事通过69.332s，最终参数化目录六例随354全量通过。全部失败报告分别保存于task-13-e2e-red和task-13-catalog-red，不改产品权限/清理实现或放宽守卫。

中间一次聚焦在Ryuk端口映射为空时启动失败13.599s，业务用例尚未运行；明确记录该容器ID/session，准备清理前已自行回收，未执行rm。正常重试通过。前端两次静态失败仅为三条旧测试缩进及新增夹具的map-spread/串行await提示，原件保留；改具名参数化案例和显式属性，断言不减。

完整Go/gen/lint-go之后仅三条Web测试纯缩进及四条E2E路径变化；完整Vitest之后仅四条E2E路径变化。逐路径/纯缩进证明限定复用这些不消费修改文件的结果；受影响lint-web/knip/build/E2E均以最终源重新执行。日志可明确归属的两个本轮容器均已回收，另一次启动失败容器也已核消失，最终现存20个逐ID等于T12保留集；不把日志未打印ID的容器数量冒充已知计数。准确产品HEAD319d4aa21790c51ee1967789dc2559119793451b的[CI41631755023](https://github.com/open-nerve/NerveProject/actions/runs/41631755023)首轮server/web/e2e全绿，远端354例通过16.7m；原始三job日志及run/jobs元数据已保存。W11本轮通过，不声称既有偶发根因已修。T13 task-done通过；一次整审见§21，最终文档提交和main的准确CI仍另核，A8/A9稍后按最终事实回填。


## 21. T14整分支审查

范围5807a0d6..319d4aa2，14提交。一次fresh审查全文读209份手写源/测试及12份文档，另读23份相关caller；5份历史文档只读变动与精确上下文，20生成物核完整diff及生成源，未声称逐行重读全部生成正文。主控重核246路径和全部审阅SHA，无漂移；三份spec、33项当时Ruling、31项变异和最终门禁输入逐项核对。完整报告为[re-39](../../../reviews/re-39-M5P4整体审核.md)。

无Critical/Important需修复；主控维护Minor P4-M1（WorkspacePatch.Logo无消费者）具名留P8，无用户可见影响。原P1焦点和P3-M1双故障cause继续P8，不重复算作本次发现或已修复。准确最后分支与main CI仍按T13/T14/T15分别关闭；当前尚未合并。

最终审查文档为产品候选之后的纯文档提交；所有4329项非文档输入保持一致。本地通过结果按逐文件hash复用，最后文档HEAD的远端CI仍完整执行。阶段checkbox与最终SHA/归档事实随T15/P5预检回填，当前不提前称main完成。
