# M5/P7b 阶段评审记录

> 2026-11-05：正式预检通过后已获准实施；预检历史结论保留，当前执行见§8。分支`codex/m5-p7b`，基线main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`。P7a main CI 41869045846及归档/正常清理已核；

## 1. 评审方式与范围

[主spec](../specs/P7b-content-assets.md)、[内容资源](../specs/P7b-content-lifecycle.md)、[编辑器](../specs/P7b-editor-images.md)与[plan](../plans/P7b-content-assets.md)共同界定16个Task。一次fresh builder对照冻结P7a实际代码逐Task核查，主控处理1项局部编排遗漏；最终main产品树与审查候选一致。正式报告见[re-49](../../../reviews/re-49-M5P7b预检.md)。

这是实施前预检，尚未开始P7b整分支deep-review；原P7、P7a审查与当前前端范围分别记录，不冒充同一轮完成。

## 2. 验收状态

A1–A8全部开放，T1–T8已实现，尚待页面与阶段联合验收。P5/P7a的API、P6的上传及权限回归仅为前置；P7b真实页面、三浏览器、local/S3、有效变异和完整七门禁必须在本阶段新运行。

## 3. 局部裁定

主spec§3的11项为绑定合同：唯一UploadSession/消费方小策略、无new_issue、私有null项目草稿、编辑器只处理图片、评论普通附件与正文分离、节点撤销不删资产、有限原始下载、可选图片能力/未来领域不启用、普通副本Issue与草稿副本Draft、遵守现有草稿owner寿命、权威重读保留本地输入。

两个预检补充的理由与代价：

- 草稿owner现为工作区集合；保留它在同工作区项目变化下的保守失效。资产不得私存DTO绕过，代价是可能需重读草稿；不顺带改M4权限寿命。
- IssueDetailHost在条件子树外持有本地输入/基准/已确认命令结果，解决普通附件权威重读引起的表单卸载。旧view/URL/请求照常结束；新owner及资产核验后才编辑，冲突提示、不重发未知写，真实权限/宿主消失永久清空。代价是增加小型受控状态与联合回归，传输可能需显式恢复。权限观察器、owner和通用读取算法不改。

原已批准P7恢复协议保持：仅作者分页查询、每页重新授权；明确选择丢弃，普通附件与绑定描述图删除行为不同；列表缺席不等于删除成功。没有新的架构待决项。

## 4. 发现与处理

唯一builder发现：IssueVerification.pending使规范Issue getter返回undefined，详情/peek移除正文，组件本地未发送评论及失败autosave可能丢失。按§3同步资源spec§4.1、主spec A2、编辑器保存、T3/T5/T6/T9/T13及联合变异要求。

主控额外核实际title-input与host嵌套，明确标题也纳入；宿主读取401/403/404与临时故障分开，前者退休状态；写422仍保留。审者没有看修订后的产品（本轮无产品修改），因此不冒称独立复审通过；主控合同自检与main输入一致性检查闭环。

## 5. 与原草案的差异

正式以P7a最终main为基线；补准备件生成DTO/端点/cursor规则，T13使用实际stories/files与fixtures/file-story.ts；原恢复/副本接收义务保留。明确草稿owner寿命与权威刷新输入所有者，不放宽架构守卫或新增依赖。phase继续M5/P7，不改工具支持的数字格式。

## 6. 移交事项

P7b T1–T13实现与证明F8–F12/F16/F17页面、准备件恢复、旧消费者删除；T14完整门禁、T15一次整审与修复、T16 noff/准确mainCI/全量归档及P8仅预检。P8/closeout承接最终故事矩阵、默认部署/容量、维护项、warning/死成员及全部handoffs关闭。

## 7. 限制与证据

本轮没有实施、安装依赖或运行P7b产品测试，48个Step/8个产品验收仍空。预检只证明合同与实际入口可衔接，页面交错必须在实施中验证；若需要更改M4通用权限/读取语义，先提交证据上报。P7a完整归档/清理与准确CI见[P7a§13](P7a-draft-preparations-review.md#13-t7-集成归档与接续)。P7b原始builder与准备资料、当前preflight输入清单/receipt保存在本阶段ignored目录，实施后随阶段归档。

## 8. 实施记录

2026-11-05负责人授权从T1开始，P7b完成整审后按原节奏集成并继续P8/M5最终收尾。根目录仍main，现有codex/m5-p7b worktree执行；此前“暂不实施”是预检交付时点。阶段验收仍逐条等待实际证据。

T1实现与验证：已有ApiClient消费生成Asset/Content/AttachmentPage/DraftAssetPreparationPage，原始字节下载独立fetch且有取消/期限/大小上限。首轮有效接口/下载断言RED，随后115/115聚焦测试通过；下载成功测试曾因jsdom FileReader任务与fake timers等待方式不符超时，修正测试为既有有期限settle，原失败完整保留。类型检查通过；修改8文件oxlint零warning；完整make test-web为18/18成功（17缓存，web实际重跑）。T1门禁收据在t1-types/t1-lint-2/t1-test-web及task-1-tests，产品输入前后不变。

原件位于本worktree `.superpowers/sdd/P7b-content-assets/`：run-check.py记录每次命令、真实exit、耗时、准确HEAD及前后输入SHA清单。安装使用锁定pnpm且输入未改；基线make test-web为18/18全部缓存命中，不冒充新运行。既有phase=M5/P7按P7a/P7b共同数字阶段合同保持。

T2实现与验证：UploadSession只消费小策略，资料策略保留P6权限；内容view使用唯一PermissionReads与规范owner。六用途、私有null项目草稿、guest评论、作者限制和永久撤回共187项通过；三项变异（取消后发布、误套封面管理员角色、ID返回复活）均触发AssertionError并核SHA恢复。替换选择时clear的同步observer可销毁会话，补重入后current检查根因修复，专用RED保留。初版四个评论用例失败源于fixture作者actor与实际账号u-actor不一致，修正fixture，不放宽权限。类型、14个修改源码文件零warning通过；全前端18/18（15缓存）通过。原件t2-selection-red、t2-green-2、t2-mutation-receipt、t2-types、t2-lint-2、t2-test-web。

T3实现与验证：406项聚焦回归通过，覆盖51文件/UUID协议顺序、首读与分页交错、重复cursor、DELETE确认/刷新分离、真实Issue计数权威重读、私有草稿准备件恢复、同资产多节点/跨view隔离/到期重读/错误额度/同步observer。普通宿主使用DetailResource、draft使用Reconciled；未改通用权限或调和语义。初版非法准备件误用SessionChangedError而被调和层静默处理，改为真实契约错误；确认后评论读取的fixture补全。同步ready→onError的真实重试遗漏有专用RED与修复。

四项最终变异均为AssertionError并逐SHA恢复：晚到分页污染新cursor集合、短页被误当结束、复用过期URL、无限错误重试。第一版短页变异只造成等待期限失败，未冒称有效断言证明；补直接检查非null cursor后重跑，原件保留。类型与修改文件零warning通过；全前端18/18（17缓存，web新跑319文件3638测试）通过。原件t3-red/green-4/reentry-red/mutation2-receipt/types-2/lint-2/test-web及完整web输出；确认收据与本地输入跨卸载的页面联合证明仍由T5/T6/T9/T13完成。

T4实现与验证：原生单文件选择/拖入、普通附件列表与中性图标、主动下载、薄ModalCore删除确认、逐ID草稿准备件恢复和effect拥有的view/upload hooks已接通；组件与资源208项通过，新增hook/控件21项通过（两组重叠，不相加）。同tick双击、防晚到结果、取消、上传不确定先confirm、删除拒绝不关框、确认后刷新失败只GET、未知删除先只读核验、焦点返回稳定重读按钮、StrictMode及宿主切换均有实际交互证明。

只读对照根目录Plane附件root/list-item：保留附件区、文件名/大小与独立删除操作；按已批准合同使用中性图标和按钮，不保留长期签名href、不嵌套button、不按扩展名假装预览。准备件展示状态/用途/大小/创建时间，并明确另一标签页和已成附件风险。复用原fake-image-picker的DOM帮手，无第二套浏览器事件实现。

初次列表/恢复RED因空行为组件尚未引入UI包，getAnimations测试spy未初始化，是fixture失败而非产品RED；修正初始化后重跑真实缺控件/请求断言，原件保留。最终修改12个源码文件零warning、类型、1364翻译引用与中英键同步通过；完整前端18/18（15缓存）通过。T4原件为picker-red/green、lists-red-2/green、components、hooks、types-2、lint-2、i18n(-sync)、test-web；后续T5/T6/T10再连接实际业务页面。


T5实现与验证：真实详情/peek接入附件区，首屏50/总数51、guest/归档只读、complete一次、只读刷新失败恢复、终态401/403/404、关闭peek晚回、删除busy防误关均有实际RootStore/页面交互。输入所有者跨pending保留标题和确认收据，StrictMode、422拒绝、基准冲突与明确选择、只读新owner显示规范文字均验证；临时读取失败不恢复旧上传能力。原链接/关系/子项/活动与owner回归764项通过，最终完整前端18/18（15缓存，web新跑326文件3692测试）通过。

局部根因修复与反例：新增只读isMissing查询供消费已有404结论，不改变IssueVerification状态机；详情/peek原仅按路由标记归档，改为同时消费规范owner归档状态；新增弹窗占用最初使用普通对象，被MobX集合增强后无法按原身份释放，独立两份占用与真实取消→Escape均RED，改为Symbol保持身份；已知保存拒绝与并发基准冲突同时出现时，通用Retry不再绕过明确选择。上述失败原件t5-archive-red、t5-dialog-release/identity-red、t5-conflict-action-red均保留。

首次宿主测试中，自动响应器提前答复旧Issue和空项目页；修正测试seed独立回答与guest的workspace角色（workspace admin有有效管理员权限），没有放宽产品授权。首次lint只有一个未用导入，已删；最终18个修改源码零warning、类型、1373翻译引用与中英键同步通过。证据t5-host-green4、t5-components、t5-types-2、t5-lint-2、t5-i18n(-sync)、t5-test-web和完整web输出。沿T4的中性图标/独立按钮视觉差异，详情和peek共用附件控件，不复制网络算法。

T6实现与验证：新评论的普通准备件仅保留资产ID，fresh view重新读元数据；HTML和attachment_ids一次POST，没有bulk绑定。未知网络/非Problem502先GET评论与资产，再由用户明确允许重发；已知422保留文稿、字段错误与IDs。已有评论独立附件区区分作者新增、管理员删除和非作者读取，不影响工作项附件数。真实RootStore/IssueDetailHost交错覆盖未发送评论与失败编辑同时经过普通附件complete、owner pending/503、只读重试后恢复，两host隔离，已确认回调先于原owner刷新。成功后清表单和重新挂载后的焦点有回归。

局部接口为现有评论命令链的可选同步onConfirmed，不改变通用读取/权限或增加第二命令；准备件资源仅持有当前view的元数据，本地移除选择不DELETE。修复恢复字段ARIA和双host重复创建器DOM ID；两个既有E2E选择器同步为带实例后缀，实际E2E由T13/T14执行。新增lite wrapper submitDisabled只控制按钮可用状态，不冒充传输忙态。

首轮回调RED、缺附件控件RED及输入恢复RED原件完整；最终评论/owner286项、前一轮相关471项通过（有重叠，不相加），9项准备件资源边界通过。类型首次失败因共享types的dist尚未重建，重建后通过；两个测试未用导入已删除，34文件零warning。1382翻译引用/中英键同步通过；完整make test-web18/18（2缓存）通过，web原输出另存。证据t6-confirm-*、files-*、lifetimes-*、input-restore-*、preparations、comments-final、types-final、lint-final、i18n、sync、web。

T7实现与验证：包声明EditorImages小接口，应用hook按editor/view持有节点操作，至多两条传输、排队取消、同节点换File、失败保留同prepare供明确确认、复制拒绝/未知不重发。读句柄稳定投影现有ContentReads，下载沿有限原始字节路径；只读宿主无write，普通附件不冒充正文图。root/view/editor退休终止上传、读取及下载，成功后释放session/preview。16项实际RootStore/hook测试及资源/P6联合238项通过，受影响附件/评论/详情101项通过（与前组有重叠）；编辑器构建、前端类型、8文件零warning通过。

局部修复：最后一个图片读租约释放后，既有ContentReads的wake监听仍常驻；新增实际监听移除断言RED后，改由首/末租约安装/卸载，未增加timer或改变错误预算。翻译函数引用变化不再重建能力；一期类型声明漏括号和fixture翻译函数不稳定导致的失败原件保留。证据t7-red、green/green2、regression、release-red/green、consumers、editor-build、types-final、lint-final。T7没有启用页面旧wrapper，包与wrapper原子迁移在T8。

T8实现与验证：editor及Rich/Lite wrapper原子替换为可选EditorImages，删除TFileHandler、自动删除/恢复监听、假百分比及旧上传hook；旧use-editor-config随类型链提前删除，T12再删除其上游未使用服务。保留既有HTML往返、尺寸/对齐、全屏缩放/拖动/键盘与显式有限下载；外部img和无能力宿主仅占位，不加载URL。包内134项及应用相关97项通过，最终完整前端18/18（15缓存，web实际331文件3732测试）通过，54个修改源码零warning，类型与1401翻译引用/中英键同步通过。

局部根因修复：拒绝文件批次在ProseMirror空文本粘贴前退出，保留未preventDefault的原生事件；禁图消费者仍保留随剪贴板携带的文字。延迟确认不产生独立Undo步骤，已确认节点通过当前editor内UUID记录恢复，内部真实移动与不可信剪贴板复制分别处理。权限abort立即终止节点操作，运行中转只读不写回；旧onLoad/onError核当前快照。StrictMode、换文件、删除节点晚回、未知复制真实卸载重挂不重试均有DOM/transaction证明。四项变异（伪造标记免复制、无限错误重读、旧图片事件、旧上传覆盖新文件）全部AssertionError并SHA恢复。

T8原件为t8-red/first-green/green2/green3/lifetime-red/move-red、editor-all系列、final-editor、web-consumers、types-final、lint-final2、test-web2、i18n/sync与两份mutation receipt；早期错误路径、测试夹具签名、ref类型、lint及失败输出完整保留，不冒称产品断言RED。完整web日志另存。页面尚未启用，T9/T10接续；真实浏览器全屏、local/S3和阶段七门禁仍待T13/T14。

T9实现与验证：工作项详情/peek正文、新建与编辑评论及描述历史统一经useIssueImages/useEditorAssets从已读取规范Issue取得view，历史只给只读images；删除这组旧fileAssetType与uploadCommentAsset/duplicateCommentAsset。正文只保存ready后的UUID，授权续期不PATCH；换图、调序为普通带版本编辑，不删除旧资产也不复制，只补块ID保持skip_activity迁移。评论图片不进attachment_ids，已发表评论按commentID范围编辑，保留评论须重新授权图片后才明确重发。

拒绝或未完成正文由T5输入所有者跨卸载保留；含图片身份的保留稿继续编辑时保持暂停，只有明确保存在当前view重验图片后才resume，纯文字保留稿仍按T5随编辑恢复debounce。联合测试在详情与peek各自经过拒绝正文+未保存标题+图片节点、读取失败、并发新正文后恢复，再经真实项目权限ABA清空；peek失读按既有合同关闭，重开为新owner后核验输入已清。18项真实RootStore/Tiptap页面测试与7项保留端口测试通过，扩展后的Step2命令14文件106项通过。

六项变异（描述不恢复、hook不恢复、无正文保留、无标题保留、全部当迁移、丢迁移标记）均AssertionError并SHA恢复；首个“编辑永不恢复”变异存活，补两项恢复钉住测试后再被杀死，存活原件保留。类型（types构建/包/web）、24文件零warning、25文件格式、web lint 69等于上限、1402翻译引用/中英键同步通过；完整make test-web18/18（17缓存，web实际重跑）通过，web直接运行332文件3751测试通过。

局部根因修复：T9初版测试配置的全局resolve.mainFields让每个无exports依赖改走需vitest转换的ESM构建，整套transform由27s升至60s，满载下把use-issue-list-mount首例（测试体内动态导入root store）推过5s上限，两次完整运行复现、单独运行通过。四组变体实测确认唯一原因是mainFields，tiptap内联对该例无影响；改为仅把prosemirror-codemark别名到ESM入口（否则真实编辑器报“Adding different instances of a keyed plugin”）、仅内联@tiptap（其ESM默认导入CommonJS tippy.js，否则“tippy is not a function”），transform回到29.8s。另修三处完整回归根因：导航守卫拒绝fixture的/assets/storage字面量，改用fake-assets既有/static路径；editor-lifecycle的store-context mock缺StoreContext；editor-status部分fake缺rootIssueStore，useIssueImages改由useWorkspace解析workspace。

执行者变更：Codex在T9未提交WIP时额度用尽（最后原件t9-preserved-edit-red，2026-11-04T20:19Z）；2026-11-05起Claude接手同一worktree、ledger与run-check.py原件规则，T15整审由fresh-context reviewer按deep-review执行。T9原件：Codex期t9-capability-red、first-green、targeted-green2、real-body系列、comment-reauth系列、comments-body-green、history-and-unmount、preserved-edit-red；接手后preserved-edit-green、after-resume-fix、mutation-never-resume（存活）、resume-pins-green、mutation-m1/m2/mj1/mj2/ms/mm、new-joint-swap-order-migration（RED）/new-tests-green、step2-green、types-build/web/pkg/web-3、lint-5、format-5、keywords、i18n(-sync)、web-lint-cap-3、regression-fixes、test-web/-2/-3（失败原件）、list-mount-alone、test-web-4、web-full。t9-lint-4/format-4为zsh未分词导致0文件的无效运行，由-5取代。真实浏览器、local/S3及阶段七门禁仍由T13/T14完成。

T10实现与验证：已有草稿（含无项目）在弹窗内由`issue-modal/draft-files.tsx`取得draft view，提供正文图片、普通附件区与P7a未完成文件恢复列表；新普通表单与新草稿表单零asset请求、零暗建draft，按是否可存草稿提示先保存草稿或先创建工作项。弹窗不再禁用图片节点类型（T8过渡期的disabledExtensions会让已存草稿与副本丢图），无草稿寿命的表单不给images，已存图片保持惰性且不发起上传。改项目或发布前先核传输：附件沿picker既有onBusyChange、图片沿编辑器就绪；繁忙时给出明确提示，用户完成或明确取消（图片取消后移除空占位）才继续。发送前先retire当前view，失败restore为新view重新读取授权，成功后旧view不再发任何请求；发布沿既有moveIssue先保存完整payload再单次publish，asset.not_uploaded后提示打开恢复列表。

副本：工作项quick-action普通副本与草稿行副本在正文含图时由`ContentCopy`处理——明确提交才建一次去图的真实目标draft（服务端不允许新宿主引用别处UUID），按文档顺序逐图服务器duplicate，再一次保存完整payload；普通副本经moveIssue发布并以Issue关闭、草稿副本以Draft结束，无图普通副本保持原直接创建。拒绝保留同一目标与已完成映射，重试不另建；关闭时toast说明草稿已保留并给草稿页路径。未知创建/发布进入明确未确认状态，先重读草稿列表再由用户允许重试；未知发布后目标已不在草稿中则只提示核对工作项，不猜实体也不重发。草稿种子表单只提交改动字段，含图判断按合并后的草稿输入。

15项新页面测试（draft-files 7、content-copy 8，真实RootStore/Tiptap/服务）与扩展Step2范围40文件427项通过；草稿原保存/完整payload/项目清除/双发布/拒绝回归随draft.test保持通过。7项变异全部AssertionError并SHA恢复：切项目不退休旧view、发布后才复制（前端publish后搬asset）、去掉传输闸、重试另建目标、未知结果不拦截、源UUID写入目标；初版“从不复制”变异同样被杀，由发布后复制版本取代。类型、16文件零warning、18文件格式、web lint 69等于上限、1415翻译引用/中英键同步、关键词通过；完整make test-web18/18（15缓存，web实际重跑），web直接运行334文件3766测试通过，transform 27.3s。

局部根因与回归修正：弹窗读取StoreContext后，既有模态测试的store-context mock须导出StoreContext并以真实store作Provider（fake-form、draft.test、base.clone、base.loading）；editor-capabilities三项过渡期“弹窗禁用图片”断言按本Task合同改为“不禁节点、不给images”。草稿种子副本最初漏判图片（只提交改动字段），改以合并后的草稿输入判断。T10原件t10-draft-files-red/-1/-2、t10-copy-red/-1/-2、t10-step2-1~4、t10-clone-fix、t10-green、t10-mutation-m1/m2/m2b/m3/m4/m5/m6、t10-lint~-4、t10-format/-2、t10-types-web~-3、t10-i18n(-sync)、t10-keywords、t10-web-lint-cap、t10-web-full、t10-test-web。真实浏览器、local/S3与另一标签页仍由T13验证。
