# M5/P6 资料图片上传与展示阶段记录

> 分支 `codex/m5-p6`，实施基线 `main@f5f4ce9431e7fb360363bf169da33487cf9b864f`。P5 已完整收尾，P6 预检提交 ab9f86b8；当前 T1–T5 完成，PC2已批准A，T6修复接续，未关闭 A1–A8。按连续授权推进，一次整分支 deep-review 后集成；重大风险先提交上报。

## 1. 输入与范围

[主 spec](../specs/P6-profile-upload.md)、[生命周期合同](../specs/P6-image-lifecycle.md)、[49直接/7透传来源清单](../specs/P6-image-consumers.md)、[15任务 plan](../plans/P6-profile-upload.md)、[预检](../../../reviews/re-42-M5P6预检.md#7-最终基线与执行前预检通过)。PC1方案A已批准；保留原授权角色、ProjectStore R06及唯一IssueRootStore观察所有者。

## 2. 执行与证据

原件在本工作树 `.superpowers/sdd/P6-profile-upload/`；progress.md记录逐Task起点与完成、precheck/保留预检/首次失败及P5收据，run-verified.py保存每次命令全日志/exit/耗时及全量非文档输入hash。产品变化不以预检旧绿色收据代替。

T1：JSON资产prepare/complete用生成DTO及当前ApiClient；raw PUT保持原File和签名URL，拒绝协议相对/带userinfo/fragment/控制字符及重复或额外头。首个终态决定取消/15分钟超时，释放监听、定时器和响应体，不把存储错误内容带入应用错误。初始骨架46 failed/1 passed，最小实现及两个测试数据形状修正后47/47；类型检查中的表驱动union推断错误已修正，原日志保留。修改四文件oxlint零warning；完整前端第一次回归发现路由扫描把协议相对检测字面量当导航地址，改为逐字符判定并保留守卫。最终make test-web 18/18通过（17缓存、web3214项实际运行），类型与四文件格式复核通过；收据接续台账，不将首次失败删除。真实浏览器Cookie隔离和页面上传留T11/T12。

## 3. 局部裁定与代价

- 跨任务接口按预检§7的三项决定执行：T3能力/T7预设/T9组合分段交付；T6共享logo props当段同步调用方；工作区能力刷新不重置未保存资料。各项正反行为由对应Task验收。
- raw PUT用有限时长独立fetch，阶段进度不伪造字节百分比；代价是无法显示线性上传百分比。前端终止不能撤回服务端已接收写，宿主绑定仍归页面。
- 原件保留至准确main全绿和归档核验，覆盖通用skill的删除scratch建议；代价是临时证据空间。仅一次fresh整审，全部局部发现按负责人要求修复，不默认延期Minor。

## 4. 尚未完成

T6–T15及A1–A8继续；没有声称逐Task单元测试等于真实页面、完整P6或M5完成。

## 5. T2 上传会话与统一权限失效

实际RootStore驱动PC1的公开非成员缺席/恢复，RED的旧token仍true和invalidate0修正为永久false/一次通知；工作项角色仍undefined，受限读取仍拒绝，成员/guest和R06保留。新增RootStore.asset经IssueRootStore注册范围生命期，用户不受工作区/项目变动影响，工作区不受其他项目影响，普通工作项删除也不取消项目封面。项目写仍核管理角色与归档状态；没有增加平行权限观察器。

上传按选择代次和目标生命期检查所有await与可观察状态反馈，取消/替换/卸载/root退休释放objectURL、控制器和注册项，无cleanup DELETE。prepare失败重新准备，PUT不确定或complete失败只确认旧ID，object_missing明确新申请；非法文件/未知限额拒绝。真实ApiClient的onResponse延迟覆盖响应已经到达但交付晚于取消/新选择，补充状态观察者重入及归档后busy收尾的RED/GREEN。

两项行为变异分别取消显式cancel和删除选择代次检查，均由真实断言拦截；源码逐SHA恢复。完整make test-web 18/18通过（17缓存，web实际运行），163项受影响回归接task-done收据；类型、格式和11个修改文件oxlint零warning。第一次types发现项目ES版本没有Promise.withResolvers、联合目标拷贝丢失关联，分别使用普通Promise停车和保留联合类型的分步副本修正；不改目标平台或放宽类型。原失败和变异日志全部保留。

新增只读current/recovery明确UI重试入口，绑定后同时核readyAsset身份；选错新文件时旧选择被释放，需重新选择。路由的真正effect cleanup与宿主绑定反馈在后续页面任务联合验收，T2不冒称页面已接通。

## 6. T3 宿主图片读取与到期展示

UserStore.data改为只读Reconciled投影，旧/me回包不得覆盖已确认的头像/封面写；首次未加载也覆盖此交错。原有测试的直接data赋值通过真实PATCH应答初始化，生产代码不加测试setter。最初3项并发性质RED，调和后既有账号/退出与47文件804项调用方回归通过。

活动图片读取按self、工作区目录、单项目、工作区成员列表合并；每个活动权限范围独立持有读取证明，URL始终投影当前规范DTO，不使用memberMap，不增权限观察器。公开非成员项目经实际ProjectStore缺席/恢复，在store和真实React/SWR两层均使旧请求失效，原工作项角色保持undefined。最后引用移除、StrictMode清理重放、源切换、失败有界重试和到期/visibility/focus/online重新核时间均有实测。个人头像/封面共用一次/me加既有profile请求；成员头像同工作区共享一次成员列表，跨工作区分别读取；空图/未知宿主/预设不自行取目录。

最初图片骨架8 failed/1 passed。补充RED揭示重新渲染已隐藏过期图但漏刷新、后来挂载者过早完成等待、新范围被共享请求失败挡住，均由对应行为断言修复。呈现组件移除完整img，保持头像原有尺寸、initials和tooltip；不以去掉src替代卸载。网络error只重新授权一次，旧回调不影响新图，实际onLoad后恢复额度；返回无效expiry则留占位和可手动重试状态。

三项单点变异分别绕过未来expiry、改用全局memberMap和恢复旧read直接覆盖write，均产生AssertionError而非编译失败，三文件逐SHA还原；见t3-mutations.json与原日志。受影响15文件210项通过；完整make test-web 18/18通过（17缓存，web296文件3293项实际运行）。最终异步read分为入口与执行方法，明确先发布pending再通知loading，保持原门禁且聚焦复验接task-done；类型、格式及修改文件零warning收据保留。

失败原件不覆写：跨工作区测试最初respond误答成员DTO给工作区接口，限制为members端点后修正；呈现单测mock不是observable，使用同值新source触发真实props渲染后验证卸载；types中的Avatar不允许className和Map泛型错误分别由外层span承载样式与明确cache类型修正。SWR2.4.2在JSDOM中将visibility事件当作其setTimeout.bind的delay产生Node TimeoutNaNWarning，trace指向锁定依赖的initFocus，不是产品计时器；原warn保留，不更换依赖或压制诊断。

局部裁定：内部拆为宿主投影、读取组与视图lease，不复制实体；工作区请求共享，但授权范围证明分别计代。代价是新范围在已有请求发出后挂载时需再读一次，仍同时最多一条该来源请求。hook增加onLoad用于成功图片恢复错误额度；可观察生命期与计时器分别由store和view拥有。hook的DOM测试扩展名按实际JSX改为.test.tsx，计划命令同步。T3只提供读取与retry接口，页面控件、公开邀请和完整消费者迁移继续由后续Task交付，A1–A8不提前关闭。

## 7. T4 原生图片选择与弹窗

先读Plane个人/工作区图片弹窗，保留预览、替换、移除、取消与上传保存的布局语义；按已批准设计改用原生file/drop和当前上传会话，不引入dropzone、旧FileService、render内objectURL或延迟reset。ImagePicker处理单文件输入、拖放的多文件/目录/文本拒绝、稳定错误文案和阶段提示；prepare失败重试、PUT不确定后先确认、object_missing另申请、ready绑定失败不重传均走原UploadSession。宿主写仍经页面提供的小回调，UI成功反馈另经当前生命期检查。

useProfileUpload把资源分配与释放限定到视图effect；ImageUploadModal仅组合此hook、现有ModalCore及内容组件。上传/绑定均禁重复提交、Escape与背景关闭，明确取消上传能中止链并保留窗口；正常关闭/成功恢复触发器焦点，unmount释放预览和中止传输，迟到绑定成功不关闭新视图。取消聚焦曾早于输入重新启用、资源退休曾聚焦旧错误提示，两项均有真实DOM RED/GREEN修复。

图片内容骨架13项RED；最初弹窗RED是测试夹具尚未载入HeadlessUI的JSDOM动画替身，不算行为证据，保留原件。之后真实ModalCore四项通过；隐去弹窗和删除unmount资源释放两个单点变异分别产生AssertionError，完整还原SHA，见t4-mutations.json。初次关闭等待把所有动画帧包在同一个act中阻止React提交，改为条件/期限约束下逐帧提交，不改产品动画或固定sleep。最新受影响7文件80项通过；类型、格式及修改文件零warning通过。首次types的vi.fn构造/调用联合推断已改为明确函数签名。

新增en/zh-CN各33个键逐叶一致且非空；现有i18n-keys守卫1324处引用通过。局部裁定把资源创建放入可复用useProfileUpload，内容组件只消费资源；代价是嵌入创建页的调用者须同样使用hook并连接onBusyChange/busyBinding。T5起页面接线与T11/T12真实浏览器仍待完成，A1–A8保持开放。


## 8. T5 个人资料、引导和自身图片消费者

个人设置与引导复用UserImageUpload薄宿主接线：仅PATCH头像或封面资产字段，移除显式null；名字与引导完成保持独立。点击头像、封面右下角动作和引导图片旁按钮保留原Plane入口位置，图片绑定成功后才关闭当前弹窗；失败保留已完成资产，重试不再prepare/PUT/complete。原始裸URL展示已从本Task六个页面/组件移除；侧栏、菜单、引导和首页占位均消费self来源，到期立即隐藏，手动重读仅GET。

页面新增最初5项行为RED（没有上传入口）；菜单证明RED为/me读取0次而期望1次。实际RootStore、ApiClient、SWR及ModalCore接通后，新页面6项与菜单1项通过；覆盖独立上传/移除、不保存未提交名字、引导不跳步、失败仅重试PATCH、换账号后的成功/失败无toast/跳步/焦点迁移、手动图片重读不重传。现有名字表单测试仍逐值检查没有图片字段；UserStore旧read/newwrite测试同时回归。菜单测试只替换菜单开合外壳使内容常驻，图片和账号读取使用真实实现；完整菜单键盘及实际浏览器由T11/T12联合验收。

T5最终完整make test-web 18/18通过（17缓存，web实际3318项）；源输入4465个前后hash一致。最终类型检查、15文件格式、13个修改TS文件零warning通过；i18n-keys核1327个引用。首次误用了不存在的prettier/check-types命令，未安装依赖，改用仓库oxfmt/check:types；第一次lint的顺序UI循环改为显式调用两个相同小操作，未增加豁免。初始失败、后续两次完整回归与最终收据全部保留。

局部裁定：同一UserImageUpload只拥有账号字段写与弹窗状态；通用内容仍不认识宿主。现有名字表单SSR测试在图片边界使用替身，新增DOM页面测试验证实际图片链，避免把测试兼容逻辑写入产品。代价是通用图片组件与页面集成均需保留覆盖，不能以SSR字段测试替代真实上传验收。阶段A1–A8仍开放。


## 9. T6 PC2 暂停：工作区写回的权限事实

实际RootStore四场景探针2 failed/2 passed：权限20→15后迟到PATCH把角色恢复20并令新上传可写；20→15→20后迟到PATCH覆盖新logo。原上传仍正确失效；移除与同slug新UUID控制组通过。产品实现尚未开始，按跨模块纪律保存当前工作并上报；推荐在WorkspaceRootStore发布边界接入现有IssueRootStore的权限代次，小接口由组合根接线，不修改通用Reconciled或增加观察器。完整范围、代价、原始证据与待决定事项见[PC2](P6-workspace-write-lifetime-decision.md)。T6–T15未完成，未运行本轮七门禁或推送/合并产品检查点。


## 10. T6 PC2 基础修复与联合回归

负责人批准A后，先以6da66e0d同步M3/M4/M5和P6合同。WorkspaceRootStore声明WorkspaceWriteScope小接口，由RootStore接到IssueRootStore的工作区身份/成员权限代次；原观察器和权限getter不变，没有依赖asset控件或修改公共Reconciled。更新在进入队列前捕获UUID/role和代次，发送前、成功/失败交付、发布后均核对；过时给undefined，当前普通拒绝原样抛出，原工作区资料表单不对undefined发成功toast。

重放检查未经修改的目录答复里的UUID/slug/role，保留尚未发布的新成员事实；否则先用旧整行盖掉新角色，观察器便永远看不到这次变化。工作区范围只含本工作区身份和成员水位，其他工作区、项目或工作项变化不会误失效。已有移除/同slug新UUID不插入的行为保持。

实际RootStore新增17项：原四场景、guest降级/恢复、成员关系移除恢复、排队、root退休/迟到拒绝、未知scope、无关变化、正常已确认写与目录重放，以及guest→admin的直接事实校验。首轮有4项测试把受控时间的每次新值误当固定应答，修正固定expiry后取得13failed/3passed的行为RED；补充guest→admin也取得1failed/16passed后修复。最终20文件293项联合回归通过，完整前端18/18（17缓存，web实际3336项），类型/格式与13个修改文件零warning通过，输入4466个前后一致。首次lint因workspace目录的非空断言规则拒绝8处测试断言，改用明确不存在就抛错的夹具，不新增豁免。

三项变异分别取消有效期检查、取消原目录role检查、恢复undefined结果成功toast，均产生AssertionError，源码逐SHA还原；原件见t6-pc2-mutations.json与t6-pc2-*.log。初始PC2实验原件仍保留。此处完成基础发布修复，T6真实工作区图标页面接线和联合UI回归继续；PC2整体页面验收及阶段A1–A8不提前关闭。
