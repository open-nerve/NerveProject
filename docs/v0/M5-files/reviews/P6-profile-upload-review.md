# M5/P6 资料图片上传与展示阶段记录

> 分支 `codex/m5-p6`，实施基线 `main@f5f4ce9431e7fb360363bf169da33487cf9b864f`。P5 已完整收尾，P6 预检提交 ab9f86b8；当前 T1–T12 完成，PC2按A完成基础与页面回归，A1/A2/A4–A6已挂证据；T13发现PC3后A3重新开放，当前暂停待负责人决定，A7/A8仍开放。按连续授权推进，一次整分支 deep-review 后集成；重大风险先提交上报。

## 1. 输入与范围

[主 spec](../specs/P6-profile-upload.md)、[生命周期合同](../specs/P6-image-lifecycle.md)、[49直接/7透传来源清单](../specs/P6-image-consumers.md)、[15任务 plan](../plans/P6-profile-upload.md)、[预检](../../../reviews/re-42-M5P6预检.md#7-最终基线与执行前预检通过)。PC1方案A已批准；保留原授权角色、ProjectStore R06及唯一IssueRootStore观察所有者。

## 2. 执行与证据

原件在本工作树 `.superpowers/sdd/P6-profile-upload/`；progress.md记录逐Task起点与完成、precheck/保留预检/首次失败及P5收据，run-verified.py保存每次命令全日志/exit/耗时及全量非文档输入hash。产品变化不以预检旧绿色收据代替。

T1：JSON资产prepare/complete用生成DTO及当前ApiClient；raw PUT保持原File和签名URL，拒绝协议相对/带userinfo/fragment/控制字符及重复或额外头。首个终态决定取消/15分钟超时，释放监听、定时器和响应体，不把存储错误内容带入应用错误。初始骨架46 failed/1 passed，最小实现及两个测试数据形状修正后47/47；类型检查中的表驱动union推断错误已修正，原日志保留。修改四文件oxlint零warning；完整前端第一次回归发现路由扫描把协议相对检测字面量当导航地址，改为逐字符判定并保留守卫。最终make test-web 18/18通过（17缓存、web3214项实际运行），类型与四文件格式复核通过；收据接续台账，不将首次失败删除。真实浏览器Cookie隔离和页面上传留T11/T12。

## 3. 局部裁定与代价

- T13重新实测死成员，只移除无调用getter；原生DOM完整标识符的遥测误报按精确词形修正，未放宽目录架构守卫、未新增exception。理由及正反样本见§18–19；代价是关键词守卫仍需对未来原生标识符保留真实反例，不以改名隐藏命中。PC3属于跨模块读取职责冲突，单独提交待负责人决定。

- 跨任务接口按预检§7的三项决定执行：T3能力/T7预设/T9组合分段交付；T6共享logo props当段同步调用方；工作区能力刷新不重置未保存资料。各项正反行为由对应Task验收。
- raw PUT用有限时长独立fetch，阶段进度不伪造字节百分比；代价是无法显示线性上传百分比。前端终止不能撤回服务端已接收写，宿主绑定仍归页面。
- 原件保留至准确main全绿和归档核验，覆盖通用skill的删除scratch建议；代价是临时证据空间。仅一次fresh整审，全部局部发现按负责人要求修复，不默认延期Minor。

- T9保留Plane设置表单先选择、一次更新的顺序；准备件只在表单局部，卡片只读规范事实。切离上传页签结束其传输，避免隐藏busy；代价是回到上传时需要重新选择。卡片以覆盖链接与独立操作控件修正嵌套anchor，不改共享导航。

## 4. 尚未完成

T10–T15及A1–A8继续；没有声称逐Task单元测试等于真实页面、完整P6或M5完成。

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


## 11. T6 工作区图标与联合页面验收

工作区设置管理员通过原图标按钮/文字入口更换或移除，复用ImageUploadModal并只PATCH logo_asset_id；成员/访客保持目录允许的只读图标，没有文件控件。目标按workspace UUID，弹窗随目标变化清理；PC2作用于实际页面绑定，失权再恢复/同slug新UUID的迟到成功不恢复旧图或旧弹窗，跨工作区迟到绑定不影响当前页面。拒绝保留原图与已完成文件，重试仅PATCH。

表单原先按整个workspace对象reset，新增真实DOM测试复现仅logo URL刷新就将“Local draft”清为“ws”。现改为只依workspace ID及name/organization_size/timezone重置，图标上传/移除/能力刷新保持脏字段，服务器资料变化及真实切换工作区仍同步。该修正遵守预检已记录的局部决定，不调整共享表单库。

WorkspaceLogo收紧为明确workspace身份或public-preview，全部六个直接调用点与手绘下拉图标同时迁移；未知身份仅占位，不接裸URL兼容分支。三个图标/两个工作区真实读取共享一轮目录请求及一轮到期刷新。公开邀请分支不依赖登录store，URL需有效expiry，过期/图片失败即卸载；AuthHeader当段适配接口与link key。邀请的完整有界重读/切换/取消仍由T10完成，不能把此处薄显示测试当成A3全部验收。ProfileImageRetry抽成明确的独立按钮，个人和工作区复用，避免把按钮塞进可能嵌在菜单按钮中的图片。

新增页面首次6failed/2passed（5个缺失入口及真实脏表单重置）；图标两项行为RED后全部通过。22项聚焦、37文件315项受影响回归、最终完整前端18/18通过（17缓存，web实际3346项），4468输入hash一致。类型、13文件格式/零warning通过，原数组索引key警告改为三处明确测试消费者，不放宽规则。最终阶段门禁、三浏览器和deep-review仍按T11–T15执行。

## 12. T7 稳定预设与主动选择

29张SVG为OpenNerve自制，稳定cover-01…cover-29，SOURCES逐一登记图形与来源；无外部图片、脚本、字体或品牌。PROJECT_COVERS只把编号解析为显示URL，未知值不给随机fallback。选择器的受控值直接取生成ProjectCreate的两个封面字段，初始省略/上传UUID/预设/两个null分别保留；只有主动随机点击才抽取编号。预设或清除先取消旧上传，原PUT迟到不再complete；绑定中或范围退休无操作。

行为RED为9failed/13passed，含无图、无入口、选中态和旧链取消；后续42项组件/映射测试通过。首次GREEN仅Vitest内联本地SVG的测试假设失败，修正测试对本地内联形式的识别，不改变应用assetsInlineLimit:0。真实应用build通过11/11（9缓存）；因T8/T9尚未消费新模块，另用隔离Vite入口构建29个SVG，逐字节与原件相同，总32601字节，详见t7-cover-build.json。此证据不替代接线后的页面构建。

1440宽截图目视全部29张16:9与4:1裁剪；390窄屏检查前8张，文字对比清楚，无横向溢出，58个img全部解码为1600×900。原件t7-covers-desktop.png/mobile.png及t7-playwright-cli保留；CLI默认Chromium版本未缓存，使用已有Chrome，无安装。预览页唯一404是favicon.ico；旧CLI文档network命令不被本机版本支持，未把它算作页面网络验收。自有临时预览HTTP进程与浏览器已关闭。

翻译全引用守卫发现T6遗漏两条图标按钮文案，补齐en/zh-CN且保留RED。1332引用守卫、类型、格式、4文件零warning及完整前端18/18通过（15缓存）。当前原始web测试与收据保存于t7-web-tests.log/t7-*-receipt.json；图片选择、创建和保存页面仍由T8/T9完成，A1–A8保持开放。

## 13. T8 一次创建绑定与表单生命期

ProjectCreationForm直接包含生成DTO的两个封面字段，默认省略；prepare/PUT/complete完成只选入表单，创建唯一POST带资产ID或稳定预设编号，不补PATCH。CreateProjectCover是原ModalCore内的内容，复用T4/T7；上传busy和创建busy分开，真正创建前同步取得提交权，覆盖react-hook-form异步校验前双事件。保存封面不创建项目，宿主拒绝后仍持有ready准备件，再提交不重复传输。

原表单300ms reset删除；创建表单、已创建后的feature-selection均按工作区UUID和打开状态拥有，关弹窗/Back即卸载上传。useCreateProject只在调用方范围仍有效时toast/字段错误/交付，未触碰服务端协议或ProjectStore的目录事实算法。预设/上传完成/取消返回选择入口焦点；撤权或换页不抢新页面焦点。取消替换会同时清掉旧自定义准备件ID，避免界面无图却静默绑定旧件；显式两个null仍保留。utils默认值原本就省略封面，不作无谓修改；默认emoji只在表单实例初始化时抽取。

首轮六个hook行为RED；DOM初轮缺JSDOM ResizeObserver（14例环境失败，不算行为），补齐浏览器API替身后取得19failed/15passed，包括真实双POST。Modal新增3failed/1passed，证明关闭时旧PUT未abort、换工作区后旧feature步骤仍在；修复后通过。补充取消旧UUID2项、焦点3项、显式清除Escape1项均取得AssertionError再修复，原件保留。首轮types的vi.fn泛型与hook依赖lint已按实际类型/稳定回调修正，不加豁免。

最终45聚焦项、364项受影响回归及完整前端18/18（web307文件3400项、17任务缓存）通过，类型/格式/修改14文件零warning与1333翻译引用守卫保留收据。真实应用bundle含29张SVG；详细读取/磁盘字节和三浏览器仍由T11/T12闭环，不将FakeNerve的页面DOM测试当真实网络验收。原始完整web输出见t8-web-tests.log；A1–A8仍开放。


## 14. T9 项目设置与规范封面

项目设置复用上传与预设选择内容，草稿单独由表单持有；选择后随原“更新项目”一次PATCH携单一非null字段或两个null。拒绝仍显示原已保存封面，保留准备件与局部预览，重试不再PUT；成功后清掉待提交字段，下一次仅改名称不再带封面。普通成员/guest只读、归档项目全部控件禁写。同步提交权覆盖异步校验前的双事件；标识查重后、回包反馈前均核当前页面、会话及既有PermissionReads水位，不新增权限观察器。

ProjectCover只收明确project source，从当前ProjectStore选择事实；完整目录缺席覆盖旧详情和迟到详情。预设映射本地、未知编号占位、公开项目非成员预设零能力GET，自定义走T3授权图片读取。真实卡片测试还复现外层链接包住设置链接及按钮，改为独立原生主链接与操作控件，键盘入口保留。共享ProjectCoverEditor从创建控件提取，两处保持相同语义；切离上传页签取消隐藏传输及旧自定义准备件，保留预设/显式清除。

初轮行为8failed/133passed；补充提交与交互守卫16failed/15passed后194项聚焦通过。17项真实DOM设置测试含双提交、拒绝重试、权限收窄/恢复、迟到成功/失败、取消换选、旧目录与链接结构；切页签单独1failed/16passed后GREEN。受影响455项/42文件通过。完整前端18/18（17缓存，web308文件3424项）通过；4512输入前后hash一致，原始输出见t9-web-tests.log。首次完整运行因测试夹具无后缀图片被导航守卫视作页面而失败，改为明确.png夹具，未改守卫；原失败、类型可空检查与no-shadow错误均保留。13文件格式/零warning、1333翻译引用、最终类型检查通过。真实浏览器和文件字节仍由T11/T12验收，A1–A8保持开放。


## 15. T10 成员图片消费者和公开邀请

全部既有头像消费者改用明确工作区UUID/memberID；当前路由消费使用useRouteWorkspace，传入slug/项目的宿主从规范store解析，历史作者沿root→dropdown→item透传，评论和编辑器保留各自宿主身份。进度统计仅投影成员ID，SwitcherLabel删去无使用者的URL属性/分支。跨工作区成员全局缓存不再充当图片事实，guest/未知成员占位，inactive沿原呈现规则；未新增M6/M7请求。56文件逐项SHA及零旧URL命中见E6 t10-consumer-audit.json；新增InvitationLogo是明确公开能力例外，ProjectCover只据规范字段决定走哪个显示入口。

邀请预览保持原公开link key且不接登录resource。preview服务接收AbortSignal，hook按已提交的链接生命期取消旧请求并校验每次返回，StrictMode重放和链接ABA须重新取当前预览。WorkspaceLogo继续薄expiry显示，InvitationLogo只组合当前预览、有限错误/到期重读与显式按钮：一次自动尝试后须成功图片load才恢复额度，刷新仍给无效expiry或拒绝不会循环；相同URL允许显式/成功重读的新尝试，旧节点事件不能破坏新图。

成员5项真实RED后通过；邀请初轮8failed/1passed，其中1项选择器误将应用加载标志当工作区图标，已改为精确alt，不计行为RED；其余到期重读/取消等7项真实失败。最终邀请10项、历史和mention宿主16项、58文件474项受影响验证通过。完整前端18/18（17缓存，web310文件3439项）通过；4516输入前后hash一致，原始输出t10-web-tests-final.log。首次全量7项失败均为两个旧组件测试未提供新增workspace图片边界，补实际workspace hook和仅呈现的image替身；评论测试保留实际RootStore provider。原输入/输出均保留。

Knip发现T2测试夹具preparedAsset只有内部使用，去除多余export后通过；没有删除仍使用的函数。类型、62文件格式/零warning、1333翻译引用通过。首次Map的SWR Cache类型、非空断言、hook依赖及测试mock lint问题按既有规则修正，无豁免或配置变化。图片GET的真实Cookie、字节/绑定链与三浏览器继续由T11/T12验收，A1–A8尚不提前关闭。


## 16. T11 真实页面、双存储与字节

F5–F7 Chromium七项全部通过：个人头像/封面分别上传、同名不同字节替换、刷新与新登录、清除；引导头像不提前提交profile；管理员工作区图标保持未保存名称，成员/访客只读，成员列表解码他人真实头像；项目准备件只随一次POST绑定，设置更换、29张预设全部解码、主动随机及清除。三个宿主各自都在local和S3完整走浏览器File→原grant PUT→complete→owner，local原URL是最终合同/static/local，未由fixture提前绝对化。

每条链核原请求JSON、URL和query不变、正文大小/原字节/MIME、无Authorization/Cookie/Referer；同源控制fetch确实携测试HttpOnly Cookie。已绑定资产实际下载字节、DB最终key/SHA/标记与local磁盘或S3对象检查一致；重复complete后整行不变。项目双击只创建一行，资料字段各自独立，旧API/pageError/CSP/意外API失败为0。原件见E6 t11-sixth-report、t11-sixth-results和t11-pages-sixth收据，4520输入不变，七项70.4秒（含独立S3启动清理）。容器集合前后相同，自有PG/reaper/S3均无遗留，未操作既有容器。

首次至第五次失败保留：个人路径、引导双入口、JSON流式正文观察、旧建议namespace、浏览器未暴露Content-Length、访客原布局成员GET和访客设置拒绝页，均已按实际合同修正测试。正文核requestBodySize及原bytes，不依赖浏览器调试协议必然暴露自动头；Cookie对照保持。访客M3脱敏目录GET本来允许，图片资源不得额外枚举，原spec措辞已校正；不改M3布局或授权。新增负例等待实际只读/拒绝页完成，避免早期零元素假阳性。首次lint两处no-shadow命名已改，无规则豁免；最终类型、格式和六文件零warning收据保留。三浏览器生命周期和阶段A1–A8继续T13–T15，不以此七项替代最终门禁。

## 17. T12 三浏览器生命周期、可访问性和变异

Chromium/Firefox/WebKit各14项、合计42/42通过（112.7秒）；本阶段独立配置含F5–F7 local、F16和F17，未扩大默认CI浏览器范围。真实prepare/PUT/complete三边界暂停后取消，旧链不再绑定，新选择成功；Back/Forward跨工作区和项目、另页换账号均终止旧资源。工作区管理员实际20→15→20，真实旧PATCH成功和403分别迟到，新的目录和图片不被覆写；恢复后再次打开弹窗核当前图片。个人旧读取跨新写返回后仍为新头像，DB和实物红色PNG一致。

真实File输入和原生DataTransfer拖放、双文件/类型错误、alert及焦点、aria-live阶段、busy禁止Escape、明确取消、重新选择、关闭后焦点归还在三浏览器一致。到期仅注入浏览器时间及宿主DTO期限，服务端签名TTL未改变：旧img卸载、一次自动读取、再次无效停止、手动只读恢复。无重放上传或owner写。T11双存储原件继续有效，T12更严格地用真实接收端观察local Cookie对照、Content-Length/原正文/MIME与无Authorization/Cookie/Referer；S3仍用原URL/独立fixture，不改传输。

首轮三浏览器34/42：旧读正在重新授权时测试错误地要求立即显示新图，修正为响应交付后核最新绑定；WebKit调试协议缺Cookie和File正文，新增只用于P6的原样HTTP转发观察fixture，从实际接收端核验，未跳过任何浏览器。额外端口及在途连接finally释放。此前背景modal aria-hidden定位失败、取消中断原件及所有trace/report均保留。原件见E6 t12-three-second-report/results、t12-three-browsers-second、t12-container-cleanup.json；既有容器集合前后完全相同。

九个定向变异全部取得行为AssertionError：携业务凭据、忽略取消、卸载不释放生命期、删除PC1非成员可见事实、忽略expiry、取全局成员缓存、旧用户读覆盖新写、创建双POST、允许连续错误自动刷新。每项准确源码SHA恢复、退出码和选中用例见t12-mutations.json及原始日志。恢复后87文件856项受影响回归通过；类型、九文件格式、八个实际lint输入零warning/error通过（配置按仓库oxlint默认忽略）。4525输入前后hash一致。完整七门禁、终审和集成继续T13–T15，未提前关闭A7/A8。

## 18. T13 验收与清理核对

| 验收 | 已有证据及边界 |
| --- | --- |
| A1 | T1原grant/File/取消/超时/重定向单元，T2阶段恢复；T11 local/S3七页面的API/DB/实物，T12三浏览器接收端Cookie对照和原正文。 |
| A2 | T2/T4实际RootStore和DOM生命周期；T6 PC2的293项联合与三项变异；T12三边界取消、账号切换、跨工作区项目Back/Forward、真实角色ABA旧成功/403、objectURL及监听清理；容器前后集合一致。 |
| A3 | 上述T3/T10/T12证据仍保留；T13补验发现与页面目录读的PC3冲突，本项重新开放，见§19。 |
| A4 | T5个人和引导控件/拒绝保留，T11 F5 local/S3独立字段、同名异字节、重载重登录和移除；T12旧读新写最终新图及磁盘字节。 |
| A5 | T6工作区图标、T7自制29素材、T8一次创建和T9设置/卡片；T11 F6/F7 local/S3及T12三浏览器，成员列表真实头像、guest只读/脱敏目录原行为保持。 |
| A6 | T12独立配置三浏览器各14项真实控件，原生File/DataTransfer、alert/aria-live/焦点、取消和迟到回包；T11三宿主各local/S3字节。 |
| A7 | 本节七门禁与准确分支CI待完成，T14整分支deep-review待完成，不提前标完成。 |
| A8 | T15准确main CI、归档及分支/worktree清理和P7预检待完成。 |

旧成员扫描实际执行M1附录A三脚本，仅将TypeScript加载路径换为仓库当前安装；全量/领域/本段交集原件在E6 t13-deadsym。全仓814 member/335 prop/2 export命中，M5域44条；不是44个必须删掉的产品成员。P6改动生产文件的18条中，两条为同一UserStore.canPerformAnyCreateAction的接口和实现，只有旧测试读取，已连computed登记和专用断言移除，原角色/工作区矩阵仍保留且9项通过。仍使用的projectsWithCreatePermissions按明确MEMBER/ADMIN集合判断，功能集合不变。

剩余16条逐项归属：

- project-cover.tsx的source.kind透传ProfileImage→useProfileImage→ImageSource判别来源，不能删。
- dropdowns/member/base.tsx两支selection的multiple/value/onChange共六条，通过`{...selection}`传CustomSelect，分别提供单/多选契约。
- upload-session.ts三支target.purpose随`...this.target`进入AssetService.prepare和生成NewAsset，服务端用途必需。
- progress-stats/assignee.tsx的id/title/completed/total由cycles/modules的progress-stats构造distribution数组并消费，数组元素被脚本误判为组件直接prop。
- workspace-menu-root.tsx的open/close由Headless UI Menu的render callback提供，属于库调用参数。

M5域其余测试记录参数/fake-assets只为实际测试使用，不假称生产死代码；未改动的旧file-upload/FileService、editor资产类型与types/file，以及旧CoverImage/propel empty-state透传项留P7/P8逐调用链清理，未机械删除历史48条。M1/M2/M3交接均已记录本段完成子项与剩余责任，整体保持open；M4附件/编辑器与首CLI投递责任未越阶段提前关闭。消费者列表原始命中保留作基线，新增实施核验不抹历史。

## 19. T13 完整门禁与PC3停点

以下是d028709f停点时的历史记录；已批准修复的当前结果见§20。

串行完整门禁实际结果：gen-check exit0/5.634秒；lint-go两模块0 issues、exit0/14.759秒；GOFLAGS=-p=1 make test exit0/991.595秒，61个实际测试包、21个无测试包，-count=1无结果缓存，bootstrap641.209秒。4525个非文档输入前后hash一致。常规Go命令不打印逐用例skip；三个既有规模入口仍由NERVE_LIST_SCALE/NERVE_MEASURE_SCALE/NERVE_PURGE_SCALE显式开启，本次不冒称完成规模验收。

lint-web在关键词步骤exit2：四处命中。素材说明已删除不必要的旧品牌名；遥测规则精确排除原生DOM完整标识符webkitGetAsEntry中的sEntry子串，增加正反样本，SDK及带额外前后缀的用法仍命中，无新增exception。原生API在当前TypeScript lib.dom.d.ts中声明，目录拒绝行为不改。23项选择控件/UserStore回归通过，四文件格式通过；首次SOURCES格式未通过及后续修正均记录。初次Go门禁后仅tools/keywords.json和SOURCES.md字节变化，准确前后hash见t13-post-gates-source-delta.json，不把旧4525输入说成当前完全相同。

剩余workspaces-list-fetch命中为真实架构冲突，原规则逐值保持，未加豁免。实际RootStore/SWR/DOM的两条探针复现重复GET，以及两条成功响应后图片仍陷入错误占位。问题来自P6的独立图片请求键与M3目录读取并行，详见[PC3事实、方案及代价](P6-directory-read-decision.md)。A3重新开放；T13尚未完成、knip/test-web/完整e2e尚未由本轮runner执行，T14整分支deep-review及T15未开始。

原件包括t13-gates.json/runner、每项完整日志和输入hash、t13-pc3-probe-second及源码、t13-keywords-after-local、t13-local-cleanup-final、t13-sources-format、t13-post-gates-source-delta。首轮探针漏开fake timers是夹具错误，不算产品RED；原件保留。测试结束后20个既有容器逐ID均在，新增遗留0，见t13-container-cleanup.json。按03§1先提交已完成工作再上报；七门禁未齐，不推送产品检查点、不合并，也不宣称PC3已修复。

## 20. PC3批准A后的联合修复（进行中）

负责人已批准复用既有读取入口。设计/spec/plan先提交5c370f01；四类图片分别共享CURRENT_USER、WORKSPACES、PROJECT、WORKSPACE_MEMBERS既有会话键。图片侧声明小的ObserveProfileRead接口，由RootStore接到真实宿主读取边界，只观察开始/完成；HTTP与数据事实仍归既有hook/store，Reconciled和目录架构守卫不改。较新真实读取能授权当前图片，旧成功/失败/undefined不能覆盖新证明；完成缓存不能代替新视图授权，同一有效权限代次的在途读取可共享。工作区图标仅核身份水位，成员/项目仍核既有权限水位。

正式四类宿主12项行为先RED后GREEN，补齐新旧成功/失败交错、完成缓存、新成员scope、无关角色/项目变化共26项。根store新增接线字段导致旧计数断言失败，按实际25字段并保留逐字段会话隔离检查。53项边界通过；第一次类型检查仅测试Map和联合Promise推导问题，已修正。三类变异分别破坏同键、旧完成屏障、权限代次，全部行为失败并逐SHA恢复；首轮权限变异停在等待超时，后改为可见请求数断言再次证明，不冒称首轮为AssertionError。

完整门禁还发现测试注释触发既有关键词、P6独立browser配置未登记脚本、内部类型多余export，均局部修正，不加豁免。完整web首轮3447通过/18失败：观察器额外microtask改变原请求启动顺序，已取消额外调度；既有评论/筛选mock补足member入口；项目角色夹具按自己端点答复并允许图片并发，避免“全局只多一个请求”的假设。80项受影响回归全绿，完整门禁与三浏览器复验继续，A3/T13尚待最终收据。

原件在ignored阶段目录的pc3-formal-red-second、pc3-formal-green-second、pc3-boundaries、pc3-types、pc3-full-failures-first/second、pc3-mutations系列和pc3-gates系列。每次失败与源码快照/输入hash均保留。Go/API/Go测试输入逐hash与T13完整Go成功时不变，沿用991.595秒完整Go及两模块零issue结果，范围与差异见pc3-go-evidence-reuse.json；不声称PC3后重新执行Go全套。

三浏览器PC3首轮39通过/3失败，三次均为F16过期恢复：SWR在online事件隐式重取共享宿主，绕过明确重试并使按钮消失。trace和数据库/截图全保留。四种真实页面hook新增行为RED（每种实际GET2次，预期1），修复后30项共享读取回归及共58项联合回归通过。为保持既定的失败占位与明确重试，四个共享宿主hook关闭SWR隐式reconnect重读；图片仍由expiry/focus/online检查决定一次到期刷新，页面挂载、身份变化及显式mutate仍读取。useSessionSWR仅新增可关闭reconnect的窄选项，其他会话请求策略保持。代价：仅网络恢复本身不再独立触发这四种宿主读取，等待上述实际读取条件；避免网络事件绕过图片重试预算。原件pc3-three-browsers、pc3-reconnect-red-page-hooks、pc3-reconnect-green；未改F16断言/期限。

PC3最终三浏览器42/42通过（113.584秒），4529输入不变；过期恢复原F16断言未改。最新完整web311文件3469项通过，其他包结果经18个turbo任务通过（17缓存、web实际重跑）；lint-web54任务通过（36缓存），knip与构建通过，38个本次TS文件零warning/error且格式通过。三类变异重新在当前实现下行为失败并逐SHA恢复。原件pc3-final-seventh-*、pc3-web-test-logs-seventh、pc3-scoped-lint-final、pc3-mutations-reconnect、pc3-three-second-browsers；它们逐项匹配当前4529输入。Go复用最终差异见pc3-go-evidence-reuse-accepted.json。A3关闭，完整E2E/远端CI/终审/集成仍待办。
