# M5/P6 接口与文件边界

与[主 spec](P6-profile-upload.md)同属已通过预检的合同；PC1 已批准 A，现有 PermissionReads 的扩展边界按[决策记录](../reviews/P6-public-project-lifetime-decision.md)落实。实体类型一律 import @nerve/api-client；以下类型仅为前端生命周期/呈现状态，不镜像服务端实体或角色枚举。

## 1. 服务与传输

新增 `web/apps/web/core/services/asset.service.ts`：

```ts
class AssetService {
  constructor(api: ApiClient);
  prepare(input: NewAsset, signal?: AbortSignal): Promise<PreparedAsset>;
  complete(id: string, signal?: AbortSignal): Promise<Asset>;
}
```

沿 unwrap 传递 API problem；上传预览沿被选择的本地 File，绑定后由 owner DTO 显示。prepare 为 POST /api/v0/assets，complete 为 POST /api/v0/assets/{id}/complete 且 body={}，具体结构直接取当前 OpenAPI。后续 P7 才加入其实际用到的 content/delete/duplicate/list 方法，P6 不堆未使用方法，也不拿他人图片 ID 作授权。

新增 `core/lib/asset-upload.ts`：

```ts
type UploadFailureKind = "network" | "http" | "timeout" | "cancelled" | "expired" | "invalid-grant";
class UploadFailure extends Error { readonly kind: UploadFailureKind; readonly status?: number; }
function putAsset(file: File, grant: PreparedAsset["upload"], signal: AbortSignal): Promise<void>;
```

不构造新的 API problem；无 Authorization/Cookie/Referer，headers 仅复制当前合同支持的一个 Content-Type。直接接受 local 的根相对 URL 与 S3 的绝对 HTTP(S) URL；URL 只校验，不重新序列化签名 query，拒绝协议相对/路径相对地址、其他 scheme、userinfo、fragment、控制字符及重复/额外头。非 2xx 只保留 status 和稳定分类，错误对象/日志不含完整 URL、签名 query、响应 XML 或私有文件名。父取消与 15 分钟 timeout 首个终态决定错误分类，finally 释放，任何一次 resolve/reject 后不再更新状态。

## 2. 上传资源

新增 `core/store/asset/root.store.ts`、`upload-session.ts`、`scope-lifetime.ts`，单文件尽量低于 400 行。

```ts
type ProfileUploadTarget =
  | { purpose: "USER_AVATAR" | "USER_COVER"; scope: { kind: "user" } }
  | { purpose: "WORKSPACE_LOGO"; scope: { kind: "workspace"; id: string } }
  | { purpose: "PROJECT_COVER"; scope: { kind: "project" | "new_project"; id: string } };

type UploadPhase = "idle" | "selected" | "preparing" | "uploading" | "confirming" | "ready" | "failed" | "cancelled" | "disposed";
class UploadSession {
  readonly target: ProfileUploadTarget;
  readonly phase: UploadPhase;
  readonly previewURL: string | undefined;
  readonly readyAsset: Asset | undefined;
  readonly error: unknown;
  readonly busy: boolean;
  readonly current: boolean;
  readonly recovery: "prepare" | "confirm" | "restart" | undefined;
  select(file: File): void;
  start(): Promise<Asset | undefined>;
  retryComplete(): Promise<Asset | undefined>;
  restart(): Promise<Asset | undefined>;
  cancel(): void;
  dispose(): void;
}
class RootAssetStore {
  constructor(root: RootStore, api: ApiClient);
  createUpload(target: ProfileUploadTarget): UploadSession;
  dispose(): void;
}
```

组件负责宿主写，UploadSession 不知道 UserUpdate/ProjectCreate，避免文件模块吞并宿主职责。start 只可在 selected/prepare-failed；retryComplete 只可在持有 prepared 的 PUT 不确定/confirm-failed；restart 明确申请新 ID；错误不吞成成功，undefined 只代表调用所属生命期已结束。重复 start 共享当前 promise，不发双链。选择新文件先结束旧链，准备件不自动删除。 current仅代表资源生命期仍有效；页面绑定后还须核对当前readyAsset是否仍是所绑定准备件，避免换选后旧回调覆盖新选择。recovery只表达显式重试入口，object_missing只能restart，不用错误文案推断阶段。非法选择结束旧选择并给出type/size/unavailable分类，任何预览分配均在选择事件中；未知实例限额时拒绝上传。

scope-lifetime 是实现细节，唯一权限输入为 root 的规范 store getter，唯一失效订阅为 IssueRootStore.registerResource。注册的 affects 对 user 恒 false；workspace/new_project 对自身 workspace 和 memberOnly 角色变化有效、对别的 project 无效；project 对本 project/所在 workspace 有效。dispose 同时注销；用户取消与权限失效区分。任何启动/await 后再次核当前有效角色及 archive 写限制。权限 ABA 由统一失效订阅的单调生命期阻止；按 PC1 在现有 PermissionReads.snapshot 的项目签名中加入规范可见但无角色的状态，ProjectStore 返回项目必须与当前工作区匹配，缺席仍是 undefined。保留原有效角色、member_role、guest_view_all_features 的语义变化，不复制实体或目录；不得将新签名用于授权。公开非成员封面读取不调用 assertProjectReadable，但必须核规范可见性；工作项接口和上传写仍执行原权限判定。

RootStore.asset 在 issue 构造后创建，dispose 释放图片/上传活动项；issue.retire 的资源 dispose 与 asset.dispose 可重复调用而不抛错。旧 editorAssetStore 留给 P7，不让它引用新资料会话。

### 2.1 宿主写发布（PC2 已批准 A）

WorkspaceRootStore构造时接收其自身声明的`WorkspaceWriteScope = (workspaceID: string) => () => boolean`小接口，RootStore用IssueRootStore提供实现。回调只查询当前root及既有工作区身份/成员权限水位，不创建资源观察器，也不把项目/工作项变化算作工作区写失效。请求调用时捕获原workspace UUID和scope，不能在排到请求时换成新代次。

`updateWorkspace`返回`Promise<Workspace | undefined>`；已知身份且scope有效才发请求，scope过时的成功/失败与未知身份给undefined；有效范围内的普通拒绝仍原样抛出。绑定控件同时保留原生命期检查；工作区资料表单的完成反馈仅在实际返回Workspace时执行。请求已经到服务端不保证可撤回，也不自动重发。

确认发布与重放由该store控制。重放还核原始完整目录里的相同UUID和role，原回答已变为另一角色则保留原回答，不以旧整行遮掉这次权限事实。缺席不插入、同slug新UUID不替换；其他工作区/项目变化不影响。公共Reconciled不增加权限逻辑，不改现有角色授权。完整依据见[PC2](../reviews/P6-workspace-write-lifetime-decision.md)。

## 3. 图片显示资源

新增 `core/store/asset/profile-images.ts`、`core/hooks/use-profile-image.ts`、`core/components/assets/profile-image.tsx`。

```ts
type ProfileImageSource =
  | { kind: "self"; image: "avatar" | "cover" }
  | { kind: "workspace"; workspaceID: string }
  | { kind: "project"; workspaceID: string; projectID: string }
  | { kind: "member"; workspaceID: string; memberID: string };
type ProfileImageView = {
  url: string | undefined;
  loading: boolean;
  error: unknown;
  retry(): Promise<void>;
  onError(): void;
  onLoad(): void;
};
function useProfileImage(source: ProfileImageSource): ProfileImageView;
```

RootAssetStore 提供所需的图片来源活动项；精确内部 lease API 由实施者按该职责选择，不公开泛型事务/策略框架。来源分组为 self、workspace-directory、project ID、workspace-members ID；memberID/图片字段仅决定当前 DTO 的投影，不能导致每头像一次目录请求。一个 source 的最初视图获得时创建图片读取生命期；按PC3复用含loginId的既有宿主SWR key，权限代次仅留在图片证明中。fetcher只经当前owner store，到期/错误/手动刷新仍通过同一hook。最后视图释放或权限失效移除图片活动项，宿主在途观察于真实读取完成时移除，root退休清空；不保留无消费者的长期缓存。

ProfileImageView.url 只能来自当前有权 DTO 的签名字段或项目稳定预设映射。source/role/expiry 每次渲染再核；状态改变须触发观察者重渲染，不能只是定时器里改非 observable 标记。onError 对同一来源响应版本仅一次刷新，旧 img 的迟到 error 不作用于新版本。url 无效时卸载 img；不能只删 src 而仍保留旧像素。公开邀请 useInvitationPreview 保留 link key，用独立薄 expiry 显示入口，不接登录 resource。

UserStore.data 改为现有 Reconciled<User> 的投影；fetchCurrentUser 仍等 tokenManager 并获取 profile，新的 user 写用 receive/confirm 的原语保持并发读调和及已确认写顺序，不能修改公共 Reconciled。不得破坏现有退出/注册/停用行为或假造空 User。

## 4. 控件

新增 `core/components/assets/image-picker.tsx`、`image-upload-modal.tsx`：ImagePicker 是无业务 owner 写的内容区，输入 UploadSession、busyBinding、选中结果与取消/提交回调；modal wrapper 只负责现有 ModalCore、标题/描述/焦点及关闭约束。新建项目在已有创建弹窗内嵌同一内容区/封面选择，不再叠第二个上传弹窗，从源头避免父级 Escape 抢关闭；创建表单提交/关闭须同时考虑上传和 owner POST 状态。清除旧 300ms reset 回调，按现有弹窗实例生命期重置，不改 shared ModalCore/useKeypress。

T4的useProfileUpload是视图拥有资源的薄hook：effect创建，关闭/目标更换/unmount时dispose，StrictMode重放重新获得资源。ImagePicker接收该资源，不替父视图处理生命周期；它只编排传输和调用onSubmit(Asset|null)，具体User/Workspace/Project写归页面。onSubmit仅返回宿主写的Promise；成功关闭等UI反馈走受当前资源、操作代次与mounted检查保护的onSaved。onBusyChange把整条操作的忙状态交给外层弹窗/创建表单，busyBinding接收外部宿主写状态；T8同时接这两个输入，不能只看upload.busy。取消上传保持窗口并聚焦重新启用的文件输入；失效范围不再主动聚焦旧控件。

ProfileImage / ProfileAvatar / ProjectCover 是呈现组件，接受明确 source 与原 alt/fallback/size/className；不接受没有 scope/expiry 的任意用户能力 URL。WorkspaceLogo 的已登录调用方传 workspace 身份，公开邀请调用方使用明确 public-preview 分支；避免 optional prop 使老裸 URL 默默绕过新合同。

新增 `web/apps/web/helpers/project-covers.ts` 与 `app/assets/project-covers/cover-01.svg`…`cover-29.svg` / SOURCES.md。导出 `PROJECT_COVERS` 稳定 ID+静态 URL 集合和 `projectCoverURL(id: string | null): string | undefined`；不重复后端枚举校验逻辑、不把 URI 当 DTO。预设选择类型只 Pick<ProjectCreate/ProjectUpdate> 的现有字段，省略/清除/单一选中状态由表单显式表达。

## PC3：共享宿主读取与完成证明（负责人批准A）

图片不创建独立的profile-image网络查询键。工作区复用useWorkspacesFetch/WORKSPACES；成员复用useWorkspaceMembersFetch/WORKSPACE_MEMBERS；个人和项目分别把既有CURRENT_USER、PROJECT读取提取为无额外业务取数的薄hook，由原页面和图片共同使用。原页面参数、DTO、权限决定和错误传播保持；使用0去重窗口允许同键恢复权限后重读，在途请求仍由SWR合并。

图片消费侧声明ObserveProfileRead小接口：接收self/workspace/project/member宿主身份与实际读取函数，返回该函数原结果/错误。RootStore统一接线；宿主store在规范读取的边界发布开始与完成，接口只观察完成，不合并/排队宿主HTTP、不改变Reconciled。这样非页面的既有合法读取也不能让图片永远停在旧请求的错误里，宿主store仍拥有数据事实，SWR仍拥有页面请求。

ProfileImages仅保留在途宿主请求和当前视图的证明；同宿主较新的读取取代旧完成，旧成功/失败不能撤销新证明。开始时捕获现有PermissionReads代次，完成时再次核对root、scope和规范源；新视图可加入同一有效代次的在途请求，但已经完成的SWR缓存不是新授权。最后引用释放移除图片活动项，在途宿主读取可继续服务其他页面；root退休清空观察状态。

工作区图标的读取证明只核身份/目录存在性水位，不把其他项目变化或不影响可见性的成员角色变化当作图标撤权；成员头像仍核成员授权水位，项目仍核PC1规范可见性和现有权限水位。使用既有PermissionReads的窄查询，不加第二个权限观察器。view卸载、失权又恢复后旧view始终无效；新view须有当前有效的读取完成。

到期/错误/手动重试均请求当前宿主hook重新验证，图片通知只合并同宿主的同时刷新意图，不自建取数函数。一次错误自动刷新预算和明确重试保持；已有在途读取可复用时不重复mutate。新视图/恢复权限不能因SWR同键缓存而一直没有证明；当前失败不得自动循环。

为保持既定的失败占位与明确重试，四个共享宿主hook关闭SWR隐式reconnect重读；图片仍由expiry/focus/online检查决定一次到期刷新，页面挂载、身份变化及显式mutate仍读取。useSessionSWR仅新增可关闭reconnect的窄选项，其他会话请求策略保持。代价：仅网络恢复本身不再独立触发这四种宿主读取，等待上述实际读取条件；避免网络事件绕过图片重试预算。
