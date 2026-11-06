# P7b 子合同：编辑器图片与宿主绑定

> 2026-11-05：正式预检通过，已获准从T1实施；阶段结束经deep-review、noff/main CI及归档后继续P8和M5收尾。分支`codex/m5-p7b`，基线main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`；P7a准确main CI41869045846、全量归档及正常清理已核。一次独立builder发现的局部问题已修订合同并自检闭环，产品验收仍待实施。

本页与[主 spec](P7b-content-assets.md)、[内容资源](P7b-content-lifecycle.md)一起接收恢复责任。包声明小接口，应用实现协议与授权；不得把业务服务导入 editor 包。

## 1. 新图片能力替换旧 TFileHandler

在 `web/packages/editor/src/types/config.ts` 声明以下消费者端口，`IEditorProps.images?:EditorImages` 替换必填 fileHandler，删除旧百分比、delete/restore/getAssetDownloadSrc 与上传空函数兼容层：

```ts
type ImageTransferPhase = "queued"|"preparing"|"uploading"|"confirming";
type ImageOperation = { signal:AbortSignal; onPhase:(phase:ImageTransferPhase)=>void };
type EditorImageSnapshot =
  | {phase:"loading"|"unavailable"}
  | {phase:"error"; message:string}
  | {phase:"ready"; url:string; expiresAt:string};
type EditorImageRead = {
  getSnapshot:()=>EditorImageSnapshot;
  subscribe:(changed:()=>void)=>(()=>void);
  loaded:()=>void;
  failed:()=>void;
  retry:()=>void;
  dispose:()=>void;
};
type EditorImages = {
  signal:AbortSignal;
  read:(id:string)=>EditorImageRead;
  download:(id:string, signal:AbortSignal)=>Promise<void>;
  write?: {
    upload:(nodeID:string,file:File,operation:ImageOperation)=>Promise<string>;
    retryUpload:(nodeID:string,operation:ImageOperation)=>Promise<string>;
    duplicate:(nodeID:string,sourceID:string,operation:ImageOperation)=>Promise<string>;
    cancel:(nodeID:string)=>void;
    maxBytes:number;
  };
};
```

这不是新的业务 DTO，ID来自真实资产回答，URL只在节点运行态。只读 editor 可提供 read/download 而不提供 write；没有 images 时不创建文件请求、不启用图片上传/粘贴菜单、现有图片安全占位。Rich/Lite wrapper 同步改为 images，可删除无意义 workspace/旧 entity props，但真正用于 mentions/链接的 props 保留。

应用 `useEditorAssets(view:ContentView|undefined):EditorImages|undefined` 持有每 nodeID 的 UploadSession、当前操作号和最多2并发队列。成功/取消节点不留下活 session；失败保留原 File/prepare身份供显式 retryUpload，选择新 File 先退休旧链。结束视图清空全部，无模块顶层实例。duplicate 不自动重试；读超时/失败不改变业务 HTML，不冒充上传。

## 2. 节点生命期与显示

每个图片节点以 editor 实例+稳定nodeID+src+操作号标识当前操作，卸载/移除/换src/换images.signal立即abort。resolve、upload、duplicate、download 的成功和失败都同时核节点与view；即使 abort 后网络回包已到，旧任务也不能 updateAttributes、提示成功或创建新图片。对象URL与进度只存在节点/会话，不序列化。

HTML只含合法 image-component[src=UUID]，保持原尺寸/对齐和M4稳定ID规则；src不是签名地址。首次打开及expiry重新解析不会触发onChange保存。有效期/自动错误重读预算由ContentReads唯一管理；节点用useSyncExternalStore消费稳定快照，只报告loaded/failed/retry，不能另起timer或无限重试。read句柄按node获得、卸载dispose；同view同assetID共享底层读取但各有句柄生命期，旧句柄的load/error不得影响新代次。最后引用释放时取消请求、timer和事件监听。

图片工具栏保留对齐、调整尺寸、全屏与显式下载；提供 en/zh-CN 可读标签，下载/取消有忙态和失败反馈，拒绝不打开空标签页、不写错误成功提示。view.signal.abort 时所有已解析src立即移除；只靠随后失败的Promise不足以完成失效。

## 3. 文件、剪贴板、复制与撤销

- 有write且editable时才接图；只接四种已批准MIME、大小/名字按内容合同。文件选择和drop/paste共用校验。一次最多10图。粘贴时，被拒的图片批（超数、超限、空MIME、SVG、目录、混合普通文件）整批不插入（连同附带的文字/HTML）、提示一次并拦下事件；纯非图片文件交回ProseMirror的粘贴管线；拖放的被拒与普通文件不拦下。（M5/P8改写：原写“混合批保持事件未处理”时，插件返回已处理却不拦下，浏览器原生粘贴绕过ProseMirror，见[P8 spec§3第1条](P8-final-acceptance.md#3-局部裁定与合同补充)。）目录、无 File、空MIME、SVG拒绝，错误role=alert；没有alert()。
- 拖入已允许的图片时才preventDefault；选择/drop/paste不可触发重复上传。输入法组合键和既有快捷键合同保留。
- 剪贴板 image-component 的src必须是UUID，不接受http/data/blob/javascript作为内部资产身份。自称“同一文档”“已上传”的剪贴板标记不可信；每次跨剪贴板插入的UUID图片均走服务器 duplicate，不能直接把源UUID保存到目标文稿。普通浏览器<img src=外部地址>沿M4清洗保持不自动加载，不借图片接线开放外部网络。
- 复制服务先验证源再验证目标；源不可见与不存在相同404。新副本描述用途未绑定，只有目标宿主保存引用时绑定。失败不改src为原图；未知复制结果保留失败节点并提示可能已生成准备件，不在mount/重开/online自动重发。显式重试可能新增私有副本；草稿宿主通过P7a恢复列表显示并允许明确丢弃，其他未绑定件按原GC。不能告知草稿用户仅取消节点就保证能发布。
- Undo/Redo恢复已完成节点时复用其UUID，不再上传/复制/restore；移除正在上传/复制的节点取消本地操作，晚到回答不能重新插回。删除已绑定图片节点只改文稿，历史和undo仍能读；不调用DELETE。删除旧 TrackFileDeletion/TrackFileRestoration、restorePublicImages 及无使用类型，不能以noop填旧接口。
- `getMarkDown`/`useCopyText`继续处理产品复制；`EditorRefApi.copyMarkdownToClipboard`保持无消费者并交P8核查，不借公开ref声明认为旧监听风险已关闭。

## 4. 宿主保存

工作项详情/peek：图片ready后才将UUID入正文，沿现有自动保存/版本机制，失败保留编辑内容与明确重试；只读历史仅解析。换图/顺序变化不得带skip_activity逃过版本，运行URL变化不触发保存。只补合法ID的既定迁移语义与严格布尔保持。

附件确认触发的权威重读会卸载正文。标题/正文/评论的未提交内容、编辑基准、失败revision和已确认资产身份按[内容资源§4.1](P7b-content-lifecycle.md#41-权威重读期间的输入与命令结果)交给现有详情host持有；包仍只消费图片小接口，不访问该host或owner store。旧图片请求/URL/队列照常结束；新view重新核资产后显式恢复编辑，重挂载不自动重试拒绝保存或未知复制，服务器正文变化时保留输入并提示冲突。

新评论：host=new_comment、scope.id=issueID；普通附件ID和正文一并进入create，成功后清表单，失败保留。上传/复制未完成时禁发送。正文图片不混入attachment_ids。评论结果未知时提示“暂时无法确认评论是否已发送。请先检查评论列表，确认未发送后再重试。当前内容和附件已保留。”以及对应英文，不自动POST重发。已有评论仅作者编辑正文/新增附件，读取及管理员删除按实际权限分离。

草稿：打开已有Draft.id后提供图片能力和附件区，包括project_id=null。新表单按主spec显式保存草稿后再进入。项目变更/发布在任何请求发出前结束旧content view；当前传输进行时要求先等待完成或明确取消，不丢未保存文稿。取消不等于服务端pending已删；发布若报asset.not_uploaded，提供准备件恢复列表，由用户明确处理，回包丢失/复制失败也必须能找回。先以既有draft update保存完整payload及project，再调用publish；后端迁移所有草稿资产，不在前端发逐个duplicate/move。返回后创建新的workitem view；失败新建原draft view重新读取授权，不恢复旧view。编辑草稿按既有字段合并/完整payload语义，不能覆盖并发较新草稿。

实际普通副本入口位于工作项详情/项目quick-action，草稿副本位于workspace-draft：普通副本最终必须返回Issue，草稿副本最终返回Draft。打开弹窗不写数据库；只有用户明确提交含图副本时才建一次真实目标draft，逐图服务器duplicate并按原顺序替换，再保存目标完整payload。普通副本在同一次明确提交流程中调用现有publish，确认Issue后才关闭/toast/回调；草稿副本确认Draft后结束。无图普通副本保留原直接create路径，不能为此无端增加草稿。

部分复制/保存/发布失败保留同一target draftID及已完成映射，重试复用目标；失败复制留下未知pending时从恢复列表明确丢弃。取消持久化后的流程须说明草稿已保留并给打开路径，不声称“没有创建任何内容”。unknown create/publish先核相应草稿/工作项列表与真实响应，不能自动重发创建或猜测成功实体；若不能确定，保留明确未确认状态和文稿。每个阶段仍核当前权限/视图；实际角色/字段/错误语义不变，不增加new_issue范围，不将源UUID直接保存为目标引用。

## 5. 必须可失败的反例

真实 RootStore/DOM：prepare/PUT/complete/duplicate/resolve 各延迟成功及失败；两个节点同ID与换src；同节点重新选不同字节；撤权恢复、草稿A→B→A与发布移除；已读缓存到期后台恢复、旧onError晚到；挂起复制时Undo后Redo；伪造clipboard标记；粘贴普通PDF交回ProseMirror、粘贴混合PNG+PDF整批不插入且拦下、普通/混合与只读drop均不拦下；保存拒绝及未知评论POST；图片顺序/换图/只补ID。删掉对应关键屏障时应AssertionError，编译/环境失败不算RED。真实浏览器同时检查请求路径、原字节、persisted UUID与无旧文件API，不能只测编辑器JSON。
