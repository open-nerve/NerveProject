# P7 子合同：内容资源、上传与下载

> **预检停点：Needs Decision。** 草稿失败准备件无法恢复的问题见[决策记录](../reviews/P7-draft-preparation-recovery-decision.md)。以下为未实施草案；其中“取消后即可发布”与“取消一律交GC”的组合不能作为有效合同，待负责人选择恢复方案后修订。
本页与[主 spec](P7-content-assets.md)一起预检，尚未实施。只描述本阶段新增接口，P6 的资料行为保持。

## 1. 服务接口

AssetService 消费当前 root 注入的 ApiClient，直接返回生成类型，不另建 DTO：

```ts
read(id: string, signal?: AbortSignal): Promise<Asset>;
remove(id: string, signal?: AbortSignal): Promise<void>;
duplicate(id: string, input: DuplicateAsset, signal?: AbortSignal): Promise<Asset>;
content(id: string, signal?: AbortSignal): Promise<Content>;
// 原 prepare(NewAsset, signal?) / complete(id, signal?) 保持。
```

各自为 GET/DELETE `/api/v0/assets/{id}`、POST `/api/v0/assets/{id}/duplicate`、GET `/api/v0/assets/{id}/content`。无下载参数；DuplicateAsset 为 purpose/scope/name?，重发会生成新资产。所有拒绝保持现有 unwrap/Problem，不把 404 当空数组或成功。

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

后端为最终权限：普通工作项附件/正文写同 issue.update；新评论同 issue_comment.create；已有评论新增图片/附件只作者，删除普通附件按既有 comment remove 规则（作者或允许的项目管理员），不能把管理员可删当可编辑。可见 guest 可读自己的工作项/评论并按评论规则创建或修改自己的评论；不借项目可见性显示别人的隐藏工作项。归档项目/工作项可读不可写；私有草稿按现有 DraftLocks 作者/工作区/未删除项目合同，归档项目不额外禁止保存，发布重判目标创建权限。

## 3. 上传有限状态机

UploadSession 对 P6 原公开 phase/current/readyAsset/recovery/select/start/retryComplete/restart/cancel/dispose 合同保持；唯一 prepare→PUT→complete 算法不复制。内部 UploadPolicy 由消费方 upload-session 声明，资料和内容工厂分别提供：不可变 target、`current():boolean`、`subscribeEnd(end:()=>void):()=>void`、`maxBytes():number|undefined`、`contentType(file:File):string`、`accepts(file:File):boolean`、`preview:boolean`。移除算法对资料专属范围/类型的硬编码，P6 原策略和 ScopeLifetime 留在资料 adapter。

target 是生成 NewAsset 的 purpose/scope 子集；内容工厂唯一映射：issue→ISSUE_ATTACHMENT/ISSUE_DESCRIPTION+issueID，comment→COMMENT_ATTACHMENT/COMMENT_DESCRIPTION+commentID，new_comment→同评论用途+issueID，draft→DRAFT_ISSUE_ATTACHMENT/DRAFT_ISSUE_DESCRIPTION+draftID。没有 new_issue，调用者不能任意拼 purpose/scope 绕过 view。

普通附件接受 File.type 的合法 canonical MIME；为空使用 application/octet-stream，不读文件头猜类型、不加 file-type 依赖。图片只接受 image/jpeg、image/png、image/webp、image/gif，空 MIME 和 image/jpg/SVG 拒绝。名字1–255 Unicode字符，无路径分隔/控制字符；所有文件 size>0 且≤当前 file_size_limit，配置未知拒绝选择，不回退陈旧常量。原 File、声明字节数与 grant headers 一致，PUT 仍 omit/redirect error/no-referrer/15分钟预算。普通附件不建立图片预览；图片 objectURL 仅选择时分配，替换/取消/卸载释放。

一次选择一个普通附件；编辑器一次最多10张图片，每视图最多2个传输链，超量整批拒绝；队列启动前核同一节点与视图。错误阶段按 P6 恢复：尚未 prepare→重试申请；PUT/complete 不确定→先对同ID confirm；仅明确 object_missing 或用户明确重新申请才新 prepare。不能重放已封存 PUT。结果成功后列表刷新失败只重读，不重新传文件。

已有 issue/comment/draft 的普通附件 complete 已原子绑定，ready 不能再发 owner PATCH；描述/new_comment 的 ready 仍私有，只有宿主正文保存或评论创建绑定。取消/卸载不能撤销已发送的服务端写、不自动 DELETE；就算旧 complete 在服务端成功，旧视图也不得发布，下一次新读取核真实结果。

## 4. 附件资源与确认后刷新

`AttachmentResource(view, service)` 提供 `rows:readonly Attachment[]`、`nextCursor:string|null|undefined`、`loading/error`、`load():Promise<void>`、`more():Promise<void>`、`retryRead():Promise<void>`、`remove(id):Promise<void>`、`afterUpload(asset):Promise<void>`、`dispose()`。DetailResource/Reconciled 管 DTO 的有效性与读写调和，view 管永久终止；请求号使旧首读/旧分页不能覆盖新列表。同一 cursor 只一在途；响应游标重复报可读错误、无自动循环，按 ID 去重且只按协议顺序展示。刷新从第一页重建，不将局部页标成完整集合。

新评论没有已发表列表端点：表单持有本视图已确认的普通准备件数组（Asset），≤100；移除仅从待发送集合去掉，取消交现有准备件 GC。发表成功后由新 comment view 读取服务端 AttachmentPage。禁止用已有 issue 的附件列表代替 comment 的列表。

删除按目标 ID oneAtATime，204 才确认移除；拒绝保留列表/确认框，未知结果先重读列表，不自动重发。确认成功后 `DetailResource.refreshAfterConfirmation` 承担只读刷新，刷新失败用独立“重新读取”入口而不是把命令称为失败。issue 的普通附件增删还调用 `issue.issues.refreshAuthoritative(issueID,{afterWrite:true})` 和现有 `issueDetail.refreshAfterChange`，使详情、排序和活动重新核准；不手工±attachment_count，不把loaded页长度当总数。评论/草稿普通附件不改变 Issue.attachment_count。

## 5. 内容能力与下载

`ContentReads(view, assetService)` 在当前视图内按 assetID 合并在途和有效期，提供 `read(id,signal):Promise<{asset:Asset; content:Content}>`、`retry(id,signal)`、`invalidate(id)`、`dispose()`，以及 `acquire(id):ContentReadLease` 给T7图片接口适配。ContentReadLease为当前资源稳定快照的getSnapshot/subscribe、loaded/failed/retry/dispose句柄，快照含loading/unavailable、error原值或ready的asset/content；T7只投影为包声明的EditorImageRead，不复制重读状态机。先 readAsset 验证 uploaded/purpose/当前宿主范围，再 content 授权；同视图同ID多节点共享一次读，跨宿主/账号不得共享授权。描述历史允许当前宿主已绑定、当前正文不再引用的图片。签名 URL 与 expires_at 只在临时资源，绝不进入业务 HTML/store DTO。

到期前复用；到期 timer、visibility/focus/online 仅在到期时触发一次重新授权。图片网络错误最多一次自动重新解析，此后显示可读占位和显式重试；只有实际 load 成功恢复自动额度。错误/旧load/旧resolve按节点代次丢弃，不打印 URL。失效立即清图；S3 已发 capability 到期前可能有效的既定限制不变。

`downloadAsset(input:{asset:Asset;content:Content},signal:AbortSignal):Promise<void>` 只在用户主动操作时调用，独立 fetch GET 原 capability，credentials omit、redirect error、referrerPolicy no-referrer，无业务 token；URL/expiry合法性校验复用已有传输规则的可共享纯函数，不能为了复用 PUT 写第二份含业务认证的 API client。单次下载15分钟预算、单视图同ID一个在途，流式读取累计不得超过 asset.size（最大104857600），结束必须精确等于声明字节，失败/取消立即cancel reader并释放监听/timer。Blob ObjectURL 用 asset.name 下载，动作完成/视图结束 revoke；onerror也回收，不用永久签名href。

图片节点拿到的只是消费者声明的 URL/expiry 投影，不直接 import ApiClient/RootStore；下载调用应用端口。下载不在节点 mount 自动请求。内存容量的权衡见主spec，不新增缓存或持久化二进制。
