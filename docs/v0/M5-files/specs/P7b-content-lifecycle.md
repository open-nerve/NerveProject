# P7b 子合同：内容资源、上传与下载

> 2026-11-05：正式预检通过，已获准从T1实施；阶段结束经deep-review、noff/main CI及归档后继续P8和M5收尾。分支`codex/m5-p7b`，基线main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`；P7a准确main CI41869045846、全量归档及正常清理已核。一次独立builder发现的局部问题已修订合同并自检闭环，产品验收仍待实施。

本页与[主 spec](P7b-content-assets.md)一起接收恢复责任，只描述本阶段新增接口，P6 的资料行为保持。

## 1. 服务接口

AssetService 消费当前 root 注入的 ApiClient，直接返回生成类型，不另建 DTO：

```ts
read(id: string, signal?: AbortSignal): Promise<Asset>;
listDraftPreparations(draftID: string, cursor?: string, signal?: AbortSignal): Promise<DraftAssetPreparationPage>;
remove(id: string, signal?: AbortSignal): Promise<void>;
duplicate(id: string, input: DuplicateAsset, signal?: AbortSignal): Promise<Asset>;
content(id: string, signal?: AbortSignal): Promise<Content>;
// 原 prepare(NewAsset, signal?) / complete(id, signal?) 保持。
```

各自为 GET/DELETE `/api/v0/assets/{id}`、POST `/api/v0/assets/{id}/duplicate`、GET `/api/v0/assets/{id}/content`。listDraftPreparations消费P7a新增 GET `/api/v0/drafts/{draft_id}/asset-preparations`，limit50、最多100，直接用生成类型；只有draft view可用。无下载参数；DuplicateAsset 为 purpose/scope/name?，重发会生成新资产。所有拒绝保持现有 unwrap/Problem，不把 404 当空数组或成功。

P7a实施接收：生成operation为`listDraftAssetPreparations`，返回`DraftAssetPreparationPage`；省略cursor读取首屏，后续只原样传递非null的next_cursor，不传空字符串或跨draft复用。两种草稿用途、pending/uploaded且未绑定均可返回；每页重验，跨页没有冻结快照。实际恢复、权限及交错证据见[P7a阶段记录](../reviews/P7a-draft-preparations-review.md#10-t5-全部门禁与接收核对)，P7a集成后的正式基线预检已完成；当前从T1实施，不以预检替代运行验收。

AttachmentService.list(target, cursor?, signal?): Promise<AttachmentPage>，其中 target 为 `{kind:"issue"|"comment"|"draft"; id:string}`，分别调用 `/api/v0/issues/{issue_id}/attachments`、`/api/v0/issue-comments/{comment_id}/attachments`、`/api/v0/drafts/{draft_id}/attachments`。请求 `limit:50`；服务端最大100，UUID升序。Attachment 六字段直接消费；分页无 URL，next_cursor=null 才到底。

## 2. 宿主视图与唯一失效来源

```ts
type ContentHost =
  | {kind:"issue"|"new_comment"; workspaceID:string; projectID:string; issueID:string}
  | {kind:"comment"; workspaceID:string; projectID:string; issueID:string; commentID:string}
  | {kind:"draft"; workspaceID:string; projectID:string|null; draftID:string};
type ContentUse = "attachment" | "description";
// RootAssetStore 的两个新工厂，不改变原 createUpload(ProfileUploadTarget)。
openContent(host: ContentHost): ContentView;
createContentUpload(view: ContentView, use: ContentUse): UploadSession;
```

ContentView 提供 `readonly host/current/signal`、`canWrite(use):boolean`、`dispose():void`。自身只持 scope 身份、单调生命期和其请求；实体来自当前 owner store，权限变化只经 IssueRootStore.registerResource/PermissionReads。不能新增角色 reaction 或 asset 权限缓存，也不能使用旧全局 Map 中未经当前读取证明的 DTO。

视图在宿主真实详情/评论/草稿成功加载后创建；根/账号/路由/编辑器实例/宿主 ID 变更即结束，取消全部子请求、URL/计时器/预览并拒绝晚到结果。工作项/评论按现有读 token 及工作区+项目范围注册；私有草稿沿工作区 member 范围，不能要求项目成员身份，projectID 可 null。每个 await 和同步 observable 回调之后再核 current；失效永不可逆，A→B→A 必须新建视图。

宿主存在性是 owner 事实，不是另一套权限算法。视图观察规范宿主身份/父范围是否仍匹配，出现缺席、评论删除、草稿项目改变或发布移除即结束；仅观察这些实体身份变化，不重复观察角色。草稿 UI 发出项目变更/发布前先 dispose，避免 React 批处理或请求排队错过中间状态。规范数据未加载不自动猜权限、不发上传；重新进入靠新 host 读取建立视图。若实现证实既有 owner 发布边界无法满足该合同，先保存证据并上报，不能放宽水位。

私有草稿的资源不能把DetailResource的项目角色条件当草稿授权：asset ContentView可复用既有PermissionReads.currentWorkspace和registerResource，但当前规范Draft仍必须由WorkspaceDraftIssues提供；owner数据一旦不可读即结束视图，不私持旧DTO继续授权。草稿附件/准备件页可直接用既有Reconciled与view/current/请求号调和，不改DetailResource通用算法，也不另建角色observer。现有owner是工作区级集合，某些同工作区项目代次变化会使其暂时不可读；保留这项保守失效，重新读草稿后建新view，不宣称同工作区所有项目变化均无影响。

后端为最终权限：普通工作项附件/正文写同 issue.update；新评论同 issue_comment.create；已有评论新增图片/附件只作者，删除普通附件按既有 comment remove 规则（作者或允许的项目管理员），不能把管理员可删当可编辑。可见 guest 可读自己的工作项/评论并按评论规则创建或修改自己的评论；不借项目可见性显示别人的隐藏工作项。归档项目/工作项可读不可写；私有草稿按现有 DraftLocks 作者/工作区/未删除项目合同，归档项目不额外禁止保存，发布重判目标创建权限。

## 3. 上传有限状态机

UploadSession 对 P6 原公开 phase/current/readyAsset/recovery/select/start/retryComplete/restart/cancel/dispose 合同保持；唯一 prepare→PUT→complete 算法不复制。内部 UploadPolicy 由消费方 upload-session 声明，资料和内容工厂分别提供：不可变 target、`current():boolean`、`subscribeEnd(end:()=>void):()=>void`、`maxBytes():number|undefined`、`contentType(file:File):string`、`accepts(file:File):boolean`、`preview:boolean`。移除算法对资料专属范围/类型的硬编码，P6 原策略和 ScopeLifetime 留在资料 adapter。

target 是生成 NewAsset 的 purpose/scope 子集；内容工厂唯一映射：issue→ISSUE_ATTACHMENT/ISSUE_DESCRIPTION+issueID，comment→COMMENT_ATTACHMENT/COMMENT_DESCRIPTION+commentID，new_comment→同评论用途+issueID，draft→DRAFT_ISSUE_ATTACHMENT/DRAFT_ISSUE_DESCRIPTION+draftID。没有 new_issue，调用者不能任意拼 purpose/scope 绕过 view。

普通附件接受 File.type 的合法 canonical MIME；为空使用 application/octet-stream，不读文件头猜类型、不加 file-type 依赖。图片只接受 image/jpeg、image/png、image/webp、image/gif，空 MIME 和 image/jpg/SVG 拒绝。名字1–255 Unicode字符，无路径分隔/控制字符；所有文件 size>0 且≤当前 file_size_limit，配置未知拒绝选择，不回退陈旧常量。原 File、声明字节数与 grant headers 一致，PUT 仍 omit/redirect error/no-referrer/15分钟预算。普通附件不建立图片预览；图片 objectURL 仅选择时分配，替换/取消/卸载释放。

一次选择一个普通附件；编辑器一次最多10张图片，每视图最多2个传输链，超量整批拒绝；队列启动前核同一节点与视图。错误阶段按 P6 恢复：尚未 prepare→重试申请；PUT/complete 不确定→先对同ID confirm；仅明确 object_missing 或用户明确重新申请才新 prepare。不能重放已封存 PUT。结果成功后列表刷新失败只重读，不重新传文件。

已有 issue/comment/draft 的普通附件 complete 已原子绑定，ready 不能再发 owner PATCH；描述/new_comment 的 ready 仍私有，只有宿主正文保存或评论创建绑定。取消/卸载不能撤销已发送的服务端写、不自动 DELETE；就算旧 complete 在服务端成功，旧视图也不得发布，下一次新读取核真实结果。草稿取消/重新申请后可能仍有pending，发布前按§6显式恢复；UploadSession不能暗自DELETE以修补未知ID。

## 4. 附件资源与确认后刷新

`AttachmentResource(view, service)` 提供 `rows:readonly Attachment[]`、`nextCursor:string|null|undefined`、`loading/error`、`load():Promise<void>`、`more():Promise<void>`、`retryRead():Promise<void>`、`remove(id):Promise<void>`、`afterUpload(asset):Promise<void>`、`dispose()`。既有DetailResource或Reconciled按宿主范围管DTO的有效性与读写调和，view 管永久终止；请求号使旧首读/旧分页不能覆盖新列表。同一 cursor 只一在途；响应游标重复报可读错误、无自动循环，按 ID 去重且只按协议顺序展示。刷新从第一页重建，不将局部页标成完整集合。

新评论没有已发表列表端点：表单持有本视图已确认的普通准备件数组（Asset），≤100；移除仅从待发送集合去掉，取消交现有准备件 GC。发表成功后由新 comment view 读取服务端 AttachmentPage。禁止用已有 issue 的附件列表代替 comment 的列表。

删除按目标 ID oneAtATime，204 才确认移除；拒绝保留列表/确认框，未知结果先重读列表，不自动重发。确认成功后 `DetailResource.refreshAfterConfirmation` 承担只读刷新，刷新失败用独立“重新读取”入口而不是把命令称为失败。issue 的普通附件增删还调用 `issue.issues.refreshAuthoritative(issueID,{afterWrite:true})` 和现有 `issueDetail.refreshAfterChange`，使详情、排序和活动重新核准；不手工±attachment_count，不把loaded页长度当总数。评论/草稿普通附件不改变 Issue.attachment_count。

### 4.1 权威重读期间的输入与命令结果

当前`IssueVerification.request`会把Issue标为pending，规范getter随即返回undefined；详情和peek据此卸载正文子树。省略afterWrite同样会进入pending，不能靠参数或旧DTO绕过。T5在现有`IssueDetailHost`作用域内持有一个小型`IssueDetailInputState`（建议文件`components/issues/detail-input-state.ts`及相邻context/hook），放在该条件子树之外。它只管理用户输入与已确认命令收据，不承担授权、网络读取或上传算法。

状态随现有host的root、工作区/项目/issue与PermissionReads单调身份一起创建和退休；详情和peek各有独立实例，内部按表单/字段及commentID区分，不能把两份评论混在一起。只因权威读取pending/失败而暂时缺少规范Issue、但host权限身份未变时，保留未提交标题/正文、编辑基准和本地revision、dirty/已知拒绝、未发送或正在编辑的评论、选中的已确认准备件ID，以及已经确认的complete/DELETE结果。真实换路由/宿主、权限代次变化（含撤回再恢复）、root退休或关闭peek仍清空，不使用localStorage或模块全局Map延长寿命。

这里的读取失败限于仍待核准的暂时故障；owner已确认宿主删除/不可见，或宿主读取最终返回401/403/404时，输入状态也永久退休。直接消费现有owner读取结论，不额外发请求或观察角色。之后即使同ID再次可见也新建状态。已知正文保存422等业务拒绝仍按原合同保留输入；fresh owner若只允许读，保留的未提交输入不得伪装成规范正文或恢复写能力。

旧ContentView、UploadSession、节点请求、读句柄、签名URL、objectURL和定时器仍永久结束，不放进输入状态。未完成传输只保留用户输入和已知资产身份/未确认提示，不复活旧队列，不自动重发prepare/duplicate/评论POST/DELETE，也不因重挂载或“重新读取”重复已确认命令。需要重新选择文件或明确处理准备件时如实提示，不能把取消写成服务端删除。新规范Issue读取成功后才创建新view；保留的资产ID在当前view重新读取、核用途/宿主/状态后才能再次提交。读取失败时保留本地输入与独立只读重试入口，禁止继续写入。

恢复输入先比较新规范正文与原编辑基准：没有并发变化才恢复待编辑内容；基准已变化时保留输入并提示冲突，由用户明确选择后再保存，不能自动覆盖新正文。已有自动保存的debounce、串行与确认语义保持，拒绝过的revision不因重挂载自动重试。该边界也覆盖评论编辑、标题与图片节点的未保存内容；能力URL和陈旧owner DTO不属于可恢复内容。

T3发布确认结果后再触发权威重读；T5提供host输入所有者及标题衔接；T6接入评论表单；T9接入正文/图片恢复；T13以真实RootStore和详情/peek做联合反例，不能用各自孤立单测替代。若实现需要改变IssueVerification的uncertain、PermissionReads、DetailResource或Reconciled通用语义，先提交证据上报，不在此裁定内。

## 5. 内容能力与下载

`ContentReads(view, assetService)` 在当前视图内按 assetID 合并在途和有效期，提供 `read(id,signal):Promise<{asset:Asset; content:Content}>`、`retry(id,signal)`、`invalidate(id)`、`dispose()`，以及 `acquire(id):ContentReadLease` 给T7图片接口适配。ContentReadLease为当前资源稳定快照的getSnapshot/subscribe、loaded/failed/retry/dispose句柄，快照含loading/unavailable、error原值或ready的asset/content；T7只投影为包声明的EditorImageRead，不复制重读状态机。先 readAsset 验证 uploaded/purpose/当前宿主范围，再 content 授权；同视图同ID多节点共享一次读，跨宿主/账号不得共享授权。描述历史允许当前宿主已绑定、当前正文不再引用的图片。签名 URL 与 expires_at 只在临时资源，绝不进入业务 HTML/store DTO。

到期前复用；到期 timer、visibility/focus/online 仅在到期时触发一次重新授权。图片网络错误最多一次自动重新解析，此后显示可读占位和显式重试；只有实际 load 成功恢复自动额度。错误/旧load/旧resolve按节点代次丢弃，不打印 URL。失效立即清图；S3 已发 capability 到期前可能有效的既定限制不变。

`downloadAsset(input:{asset:Asset;content:Content},signal:AbortSignal):Promise<void>` 只在用户主动操作时调用，独立 fetch GET 原 capability，credentials omit、redirect error、referrerPolicy no-referrer，无业务 token；URL/expiry合法性校验复用已有传输规则的可共享纯函数，不能为了复用 PUT 写第二份含业务认证的 API client。单次下载15分钟预算、单视图同ID一个在途，流式读取累计不得超过 asset.size（最大104857600），结束必须精确等于声明字节，失败/取消立即cancel reader并释放监听/timer。Blob ObjectURL 用 asset.name 下载，动作完成/视图结束 revoke；onerror也回收，不用永久签名href。

图片节点拿到的只是消费者声明的 URL/expiry 投影，不直接 import ApiClient/RootStore；下载调用应用端口。下载不在节点 mount 自动请求。内存容量的权衡见主spec，不新增缓存或持久化二进制。

## 6. 草稿准备件恢复

`DraftPreparationsResource`只接受当前draft ContentView，消费P7a生成分页DTO；沿T3同一资源调和/请求号原则，私有草稿不额外套项目角色管理rows/nextCursor/load/more/retryRead/dispose，不能另建授权观察器。它与已绑定AttachmentResource分开呈现，不能混成附件总数；当前页不等于全部准备件。入口由用户主动打开，也可在发布asset.not_uploaded后提示打开，不在每次键入或图片onError无界刷新。

显示文件名、pending/uploaded、用途、大小、创建时间，明确“未绑定不等于可以自动删除，可能是另一标签页正在使用”。确认说明“所选文件可能正在另一标签页上传，或已成为普通附件；丢弃会删除这个文件”，配对应英文，不能承诺只删仍未绑定件。用户按ID逐项选择并确认丢弃，已有AssetService.remove执行；busy防重入、失败保留，confirmed204与后续刷新失败分开。未知DELETE先重读并必要read(id)核验，不以列表缺席独断删除成功，不自动重发；描述图已绑定返回asset.bound时保留并解释，普通附件按既有权限可删，不能删正文或恢复历史。

本地取消/重新选文件只终止本链。丢回prepare回答、failed duplicate或刷新后没有已知ID时，仍从完整有界分页找到；不按文件名猜、不静默全删、不自动丢弃其他标签页。明确删除选中的pending后，重新检查当前正文/附件并让用户发布；新申请可让服务器继续拒绝，提供有限显式重读，不无限自动publish。此处使用P7a协议，不能只保留已知ID作为恢复实现。
