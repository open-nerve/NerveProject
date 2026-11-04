# M5/P6：资料图片上传与展示

> 预检通过，T1–T4 完成，继续 T5。分支 `codex/m5-p6`，初始基线c59cedf3，实施基线已同步到 `main@f5f4ce9431e7fb360363bf169da33487cf9b864f`。P5 与两次集成修复已完成准确 main CI、全部证据归档及分支/worktree清理，见[P5 收尾记录](../reviews/P5-issue-assets-review.md#222-最终main全绿与归档清理)。
> PC1 的复现、已批准方案 A 及代价见[决策记录](../reviews/P6-public-project-lifetime-decision.md)；M4/M5设计、spec、plan与联合回归落点已同步，原角色授权和ProjectStore R06保持。
> 依据 M5 设计 §3、§5–6、§9、§11–12、P4 已实现的资料引用合同及 P5 的准备件隐私。执行沿连续授权：主控局部决定并记代价，重大风险先提交再上报。

## 1. 目标、边界与决定

恢复个人头像/封面、引导头像、管理员工作区图标和项目自定义/预设封面的实际交互。用户选文件后能看到本地预览；上传经 prepare→raw PUT→complete，再由宿主自己的更新或一次创建请求绑定。展示地址到期后通过有权读取的宿主 API 刷新，任何旧会话、路由或权限生命期的回包不得重新显示图片或更新新宿主。

只用当前 React/MobX/SWR、浏览器 File/fetch、生成的 OpenAPI DTO、现有 ModalCore 和样式。无服务端新协议、表、依赖或通用框架变更。不恢复 Unsplash、Page、M6/M7 未实现接口，不提前开放 P7 编辑器/附件入口。

主控决定及代价：

1. 独立 fetch 直传，credentials=omit、redirect=error、referrerPolicy=no-referrer；不使用会话 API client 或旧 Axios 上传器。XHR.withCredentials=false 仍会发送同源 Cookie，已用隔离 Chromium/Firefox/WebKit 验证。显示准备/上传/确认阶段与不确定进度，不编造字节百分比；代价是暂无线性字节进度。浏览器实际 File 自动产生 Content-Length，不手写禁用头。
2. 新资料消费者用 AssetService；旧 FileService、EditorAssetStore、services 包随 P7 最后编辑器/评论消费者一起删除。把整套 P7 提前会扩大阶段，复制旧 multipart helper 会留下兼容层；此选择多保留一个阶段旧代码，M5 整包删除责任与关键词守卫不变，设计 §9/§12 和交接同步。
3. 上传会话由当前视图持有，RootAssetStore 只提供当前登录服务、生命周期接入和图片读合并。继续由 IssueRootStore.registerResource 发出权限失效，不新增权限观察器、实体目录或授权缓存。代价是资料资源需要自己的小生命周期对象，不能把强制 workspace 的 DetailResource 当作个人账号资源。
4. UserStore 接入已有 Reconciled<User>，阻止刷新旧读覆盖已确认的换图。WorkspaceRoot/ProjectStore 的规范事实选择保持，尤其不得让旧项目详情复活完整目录已移除的私有项目。此为新增并发读取的必要局部收敛，不重构共享 Reconciled。
5. 资料图片的通用显示组件同时覆盖既有可达头像/封面消费者；成员图片必须依据当前工作区成员接口及角色，不以 MemberRoot.memberMap 的历史头像地址为授权证明。名称等原显示语义不借此改动。

## 2. 上传合同

AssetService(ApiClient) 直接消费 NewAsset / PreparedAsset / Asset，prepare、complete 接受 AbortSignal；只提供实际消费者需要的方法。四种用途及 scope：USER_AVATAR、USER_COVER→{kind:user}；WORKSPACE_LOGO→{kind:workspace,id:workspaceUUID}；已有项目 PROJECT_COVER→{kind:project,id:projectUUID}；新建项目→{kind:new_project,id:workspaceUUID}。不以 slug 充当 UUID，不加入 new_issue 或客户端自造用途。

选择图片允许 JPEG/PNG/WebP/GIF；File.type 为空或不在集合时明确拒绝，不能靠文件扩展名猜测可信 MIME。size 必须为正且不超过当前 InstanceInfo.file_size_limit，超限不截断/拆分。名称、实际解码和服务端限制仍由后端最后裁决。输入错误与服务端 problem 均显示稳定中文/英文文案，不回显签名地址或原始服务端 XML。

raw PUT 使用服务端 upload.method/url/headers 原值：只接受 PUT；URL 支持当前 local 返回的根相对路径（按当前页面 origin 解析）和 S3 返回的绝对 HTTP(S) 地址，拒绝协议相对地址、路径相对地址、其他 scheme、userinfo、fragment 或控制字符，不改写签名 query。headers 只接受当前合同的一个 Content-Type，拒绝重复或额外头，不加 Authorization/Cookie/Content-Length。File 直接作为 body，禁止 FormData、multipart、自造查询参数和透明重定向。HTTP 非 2xx 为失败，网络拒绝、超时、用户取消和生命周期失效可区分。上限 15 分钟主动取消，父 AbortSignal 立即取消；定时器、监听、响应体读完或取消均释放。上传授权 expires_at 在发起前检查，不能把它当作已开始传输的截止点。服务端的独立传输预算仍生效。

UploadSession 状态至少含 idle、selected、preparing、uploading、confirming、ready、failed、cancelled、disposed；busy 只有前三个请求阶段。ready 保存真实 Asset（state=uploaded）和选择文件的预览，不由客户端假造。双击开始只有一条 prepare/PUT/complete 链；每个 await 之后及回调之前检查所属生命周期。取消或 unmount 后不能继续 complete、owner bind、toast、关新弹窗或覆盖新进度。

失败恢复：

- prepare 失败：保留选择文件，明确重试从新 prepare 开始。
- PUT 的网络/响应不确定：保留 prepared ID，允许先重试 complete 判断实物；若得到 409 asset.object_missing，显示需要重新上传，重新上传必须新申请和新 key，不能复用已经可能写过的不可变 PUT。
- complete 失败：保留 prepared ID，明确重试 complete；不默认重新传字节。授权过期但此前 PUT 已成功不能被前端错误阻止 complete。
- owner PATCH/POST 失败：保留已完成准备件，重试绑定不重传；选择替换文件则另起准备件。取消不在 effect cleanup 中发 DELETE，未绑定件按既有 7 天清理。
- 新建项目 POST 结果未知：不自动重发、不声称服务端已回滚；保留表单并给出明确核验/重试说明，重复提交由现有创建流程处理。

## 3. 生命周期和图片读取

上传的不可变目标包含登录所属 root、user/workspace/project UUID。组件须由既有路由 key 隔离，资源在 effect cleanup 释放；Back/Forward、关弹窗、目标切换、退出/换账号、撤权和 root.dispose 均终止旧生命期。失权后恢复权限也不能让旧请求有效（ABA）。个人资料不因无关项目权限变更而取消；工作区图标不因另一个项目变化而取消；项目封面跟随本项目/所在工作区及有效权限。工作区写只允许 role20；新建项目 role15/20；已有项目写按现有项目管理权限与 archived 规则，后端最终裁决。

**PC1 已批准 A：** 现有 PermissionReads 的项目事实签名必须区分“规范 getter 可见、工作区匹配但无有效工作项角色”和“缺席”。失去当前可见性时推进原项目水位并经 IssueRootStore 通知覆盖资源；重新出现不能使旧 token 有效，新活动项须重新读取证明。有效角色、guest 范围和业务写权限继续独立检查；不能由非空签名推出工作项可读。无变化目录/无关项目不触发失效，ProjectStore R06 事实选择保持。实际 root/ProjectStore、图片资源和页面三个层次分别验证，不以孤立伪 store 代替联合回归。

只共享同一个 root 中当前仍被视图引用的图片来源读取：当前用户、工作区目录、单项目或某工作区成员列表。共享项只有 in-flight、失效代次、引用数、错误/加载状态，实体事实仍属于既有 User/Workspace/Project/WorkspaceMember store；最后引用释放后移除活动项。每个视图自身拥有取消/到期计时及 URL 展示证明。IssueRootStore 统一回调失效，禁止第二套权限监听。

取消的物理边界：本阶段新发的 prepare/PUT/complete 支持 AbortSignal；既有宿主 store 的共享读取和已发送的绑定写不为单个图片卸载而改变全模块 API。它们可能完成，但失效的图片生命期不能发布 URL、继续绑定或反馈；服务器已收到的写不承诺回滚。最后引用释放不保留 pending Map 项，旧 promise 结果不得清除/覆盖随后创建的新活动项。代价是已有宿主网络读可能运行到原期限，不能把忽略迟到结果说成传输已中止。

首次获得一个来源的活动读取生命期时，经宿主 API 重新授权；相同来源同时可见的多张头像合并一次读，不能每张头像各取完整成员表。数据以宿主 store 已调和的返回值为准，不把原始旧回包重新投影成图片。收到的新来源与当前 scope/代次不符时不展示；权限恢复需新读证明，不能直接复用历史 URL。

上述新读针对要展示的能力 URL：已知图片字段为 null、静态默认图或合法预设时没有能力地址需要刷新，不为每个空头像/预设卡片额外 GET。未知宿主仍等待原页面加载，不绕过授权自己枚举用户/项目。后续图片字段变为非 null 时再按当前生命期读取；同时可见的同来源图片仍只合并一条请求。

宿主 fetch 返回 undefined 代表被较新读取代，不作为成功读取证明。本次保留占位并允许明确重试，不能从历史缓存补图或自动循环；正常页面初次并发的复验应覆盖这一条。每个来源最多一个正在进行的本阶段读，签名 URL 可复用到原 expires_at，不能因此反复取目录。

图片显示必须同时满足当前宿主可见、URL 非空、expires_at 是有效未来时刻、该来源读取生命期有效。过期立即卸载 img，显示 initials/中性占位；再进行一次有权宿主刷新。已过期/无效新地址或刷新失败留占位与明确重试，不自动无限重试。图片网络 onError 每个来源版本最多一次重新授权，仍错则占位。手动重试只读，不重放已成功上传/绑定。组件、时钟、网络恢复均以当前时间再核 expiry，不能仅依赖后台标签页延迟的 timer。

来源规则：

- 自己头像/封面由 /me DTO 和 UserStore 刷新；换图完成后侧栏/个人设置/引导一致。
- 工作区图标由当前工作区目录及 WorkspaceRoot.fetchWorkspaces 刷新；成员/访客可读其所属目录的图标，只有管理员可写。
- 项目封面由 ProjectStore.fetchProject / 当前目录事实读取；cover_builtin_id 走本地静态映射，不生成资产 URL。当前完整目录判项目缺席时旧详情不能作图片权限证据。
- 成员头像只取当前 workspace memberships[memberID].member 的 avatar_url/expires_at；自己的头像可用自己的 /me。读取其他成员要求当前工作区 role15/20；guest 不因显示占位头像而发送禁止的成员 API。inactive 成员是否显示沿所在页面既有语义，不机械全删。MemberUser 不暴露 asset ID，禁止凭 UUID 拼他人资产读取入口。
- 邀请预览是已有公开邀请能力的例外：用邀请标识及已有预览接口重新读 workspace_logo_url/expires_at，不能强求登录或复用登录会话图片租约；邀请更换/页面卸载取消，拒绝不无限重试。

## 4. 控件与持久化

复用 ModalCore 的焦点约束、Escape、背景和返回触发器规则。原生 file input 支持键盘，拖放只接受一个图片文件；拒绝多个文件、目录或非文件数据，并有可读文案。本地预览 createObjectURL 在选择事件/生命周期中创建，替换/取消/卸载 revoke，不能在 render 内重复分配。没有 react-dropzone，禁止新增依赖。

弹窗保留 Plane 的预览、替换、移除、取消与上传保存布局语义。请求中禁重复提交和隐式 Escape/背景关闭，上传阶段提供明确“取消上传”；owner 绑定请求中不承诺撤销服务端写，等待明确结果或随路由卸载。aria-live 读出阶段与结果，错误 role=alert 并焦点可达；选择/错误/成功的焦点均测试。当前 scope 已消失时不把焦点返回旧页面。

个人资料头像、封面独立保存，UserUpdate 分别传 avatar_asset_id / cover_image_asset_id；清除传 null，省略不得误清其他字段。引导头像沿相同控件和服务，不提前提交整个引导流程。WorkspaceUpdate.logo_asset_id 同理，非管理员没有上传/删除入口；服务端拒绝保留原图与选择。

项目封面三态：未选择（创建不传、更新省略）、预设 cover_builtin_id、上传 cover_image_asset_id；显式移除发送两个 null。自定义和预设不能同时非 null。新建项目先完成准备件，CreateProjectForm 只用一个 ProjectCreate POST 带所选 ID，禁止先建项目再 PATCH。默认无持久封面，界面可以沿现有中性背景。制作本项目自己的 29 个 SVG 静态封面，稳定编号 cover-01…cover-29，SOURCES 标明自制来源；随机只响应用户主动操作，保存稳定 ID，不保存 Vite hash URL。已有项目设置与卡片由 DTO 解析同一映射，未知预设安全占位，不随机替换。

所有新增文案 en/zh-CN 同步，修改文件 oxlint 零警告；不将旧警告上限调高。M5 全仓至少一个旧规则清零仍由 P8 总复核，不重复领取 M4 no-shadow。

## 5. 验收与证据

- [ ] A1 真实 File 直传无业务凭据、无同源 Cookie、无重定向，长度/字节/MIME、取消/超时及逐阶段重试正确。
- [ ] A2 账号/路由/权限 ABA 与迟到成功/失败不污染；自身用户和无关项目互不误失效；资源、timer、objectURL、临时容器全部释放。
- [ ] A3 图片按宿主重新授权和 expiry 显示，多头像合并读取，失权即卸载，旧详情/全局成员缓存不能复活 URL；公开邀请边界正确。
- [ ] A4 F5 个人资料/引导替换和移除、旧 read/new write 交错、拒绝保留、刷新与重登录经真实 API 和字节验收。
- [ ] A5 F6 管理员图标、成员/访客边界、成员列表头像、签名到期恢复；F7 一次项目创建绑定、29 个预设和手动随机、已有项目替换/清除。
- [ ] A6 F16/F17 在三浏览器验证真实控件、raw PUT、拖放、取消、焦点/读屏、Back/Forward 和迟到结果；local/S3 至少各一条真实资料上传页面，API/DB/磁盘或对象 hash 相符。
- [ ] A7 局部测试、必要行为变异、七项本地门禁、一次整分支 deep-review 与修复复核、准确最终分支 CI 全绿。
- [ ] A8 --no-ff 后准确 main CI 全绿，原始证据归档并核 hash/bytes/mode/membership/unknown=0，正常清理，准备 P7 预检。

F5–F7 正向数据经 API，SQL 只断言/明确故障夹具；到期 UI 测试可将宿主响应的 expires_at 限为已过去时刻，但须标明是界面失效注入，不冒称服务端真实签名 TTL 已被修改。P1/P3 已有真实签名期限合同保持。至少覆盖上传后 ready 但 bind 拒绝、完成超时后重试旧 ID、PUT object_missing 后新 prepare、预设替换正在上传文件、同名不同字节、两个工作区同头像、权限撤销恢复后的旧响应和卸载双次清理。

## 6. 参考

- https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequest/withCredentials ：withCredentials 对同源请求无效。
- https://fetch.spec.whatwg.org/#concept-request-credentials-mode ：omit 模式不携带凭据。
- P5 ignored p6-transport-probe-results.json 为三浏览器隔离协议证据，归档时附 SHA/bytes；不是产品实现或 P6 通过证明。
