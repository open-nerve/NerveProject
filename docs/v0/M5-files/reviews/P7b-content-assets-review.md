# M5/P7b 阶段评审记录

> 2026-11-06：P7b已完成；预检历史结论保留，实施见§8，T14门禁与接收核对见§9，T15整审修复见§10，集成、归档与接续见§11。原分支`codex/m5-p7b`（已正常清理），基线main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`，noff main`2b2c6f33e3a93536742d5f28b74eb959ba73d6d1`。P7a main CI 41869045846及归档/正常清理已核；

## 1. 评审方式与范围

[主spec](../specs/P7b-content-assets.md)、[内容资源](../specs/P7b-content-lifecycle.md)、[编辑器](../specs/P7b-editor-images.md)与[plan](../plans/P7b-content-assets.md)共同界定16个Task。一次fresh builder对照冻结P7a实际代码逐Task核查，主控处理1项局部编排遗漏；最终main产品树与审查候选一致。正式报告见[re-49](../../../reviews/re-49-M5P7b预检.md)。

这是实施前预检，尚未开始P7b整分支deep-review；原P7、P7a审查与当前前端范围分别记录，不冒充同一轮完成。

## 2. 验收状态

A1–A8全部完成：A1–A6逐项核对见§9；A7（RED/GREEN与变异、local/S3、三浏览器、七门禁、一次整分支deep-review修复与准确最终分支CI）见§9–§10；A8（--no-ff后准确main CI、全量归档与正常清理、P8 spec/plan预检）见§11。P5/P7a的API、P6的上传及权限回归仅为前置；P7b真实页面、三浏览器、local/S3、有效变异和完整七门禁均在本阶段新运行。

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

T11按负责人2026-11-05裁定A偏离计划原句“不新建后端字段”：服务端display_properties原为OpenAPI、project/issue词表与迁移00026 CHECK三层严格的固定11键，前端PATCH完整对象，只加前端键会被拒；改为端到端新增attachment_count显示键（OpenAPI与生成、两处词表、迁移00038替换CHECK并回填true）。这与M4设计表结构“attachment_count（M5）随各自迁移加回”一致；B（并入link/常显）与C（仅本地开关）未采用。

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

T11实现与验证：附件数直接取服务端`Issue.attachment_count`。列表/看板行在显示且计数大于0时给出带朗读名称的信息性计数（点击仍打开工作项），表格新增附件列（数量+单位，0也显示），表头双向排序发送±attachment_count；排序与跨页位置全部由服务器决定，前端只在偏好确认后重读第一页，从不按已取样本自排。按§5的负责人裁定，服务端新增attachment_count显示键：project domain默认true、HTTP双向映射、project/issue两处list_settings词表，迁移00038替换CHECK、为既有行回填true，Down恢复11键；项目范围由服务端偏好保存，成员/归档等本地范围的读回名单同步加入；列表页解码对attachment_count做安全整数检查，与sub_issues/link一致。

测试：13项真实页面（`issue-layouts/attachment-count.test.tsx`：列表/看板/表格的0/1/51、偏好隐藏、表头双向排序与方向性ID tie、增删附件后的权威读使列表重读而行进出未加载页、另一列表带来新计数时重读、403/503不沿用旧值且只有确认重试显示新值、404经重读离开、成员跨项目视图），另有行属性7项、表头4项、显示菜单1项（保存整组并重载）、成员视图偏好1项、页解码1项；全部请求监测无逐行GET或附件列表读取。服务端：迁移00038的回填/默认/CHECK拒绝码/Down、HTTP 12键往返、bootstrap默认与自定义偏好、issue list_settings词表。Step2命令57文件860项、constants 10项通过。8项变异全部以断言失败并SHA恢复：零计数仍显示、忽略显示偏好、本地范围丢失选择、接受不安全计数、表格丢数字、行显示别的计数、排序方向对调；store fields去掉attachment_count的变异两次存活（权威读本身就会重读列表），补“另一列表带来新计数”测试后被杀。完整make test-web 18/18、web直接335文件3792测试、类型、17文件零warning、19文件格式、web lint 69等于上限、1416翻译引用与中英键同步、关键词、golangci-lint 0 issues通过，生成物重跑一致。

局部裁定与原件：bootstrap首跑因命令遗漏项目的`-timeout=20m`，在Go默认10分钟处截断（当时运行中的测试才1s，非挂起），按Makefile的GO_TEST_TIMEOUT重跑623s通过，期间只有web文件变化。knip现报3个未用文件、2个导出、1个类型与3个枚举成员，均为旧文件链与T7/T8遗留、无T11新增，由T12（Step2含make knip）清除。T11原件t11-server-red、t11-server-green（bootstrap超时）、t11-server-bootstrap、t11-web-red、t11-web-green-1~8、t11-build-packages、t11-step2/-constants、t11-mut-1~8与5b/5c、t11-lint（无效路径）/-2/-3、t11-format（无效路径）/-2~4、t11-types/-2、t11-i18n-keys（无效路径）/-root、t11-i18n-sync、t11-keywords、t11-check-lint、t11-lint-go、t11-test-web、t11-web-all、t11-knip、t11-gen-repeat。真实浏览器、local/S3与三浏览器仍由T13验证。

T11补正：T12的`make lint-web`发现e2e包类型检查失败——i5列表偏好与p8项目偏好两个端到端故事仍按11键构造并整体比较偏好，运行时toEqual也会因服务端返回12键失败。T11只跑了web包的类型检查。单独提交`ccc7369c`补键，i5显示菜单逐项切换加入“Attachments”，测试名与证据标签改为twelve；e2e类型检查通过，真实运行由T13完成。

T12消费者清单与删除：重跑全仓rg（原件t12-consumers.txt，95处/34文件）并逐条归类。删除：FileService/FileUploadService、编辑器资产store与`useEditorAsset`（含14处测试死mock）、`use-file-size`与`MAX_FILE_SIZE`/`DANGEROUS_EXTENSIONS`、utils的`getEditorAssetSrc`/`getEditorAssetDownloadSrc`/`getFileURL`/`getAssetIdFromUrl`、`EFileAssetType`与`types/file.ts`、编辑器无人使用的`helpers/assets.ts`，以及整个`web/packages/services`与独有依赖`file-type`（锁文件由pnpm离线重算，只删file-type及其传递依赖，testcontainers条目去掉peer后缀、版本不变）。迁移：`normalizeAPIRequestURL`连同原单元测试移到`core/lib/api-request-url.ts`，APIService只改导入、行为不变，测试示例改为中性相对路径；`use-parse-editor-content`不再给图片造旧地址，复制为Markdown时省略图片（P8具名风险）。未来领域无能力：M7收集箱创建弹窗删去从未被调用的上传回调与bulk状态请求，不启用图片。T1–T10已迁：工作项/评论/草稿附件与正文图片。`TCommentsOperations`已无上传成员，`TFileHandler`在T8已删。

行为测试：真实编辑器的收集箱创建测试粘贴图片后提交，断言旧APIService仅一次intake POST、生成客户端与原始传输零请求、正文无图片；`watchLegacyFileTraffic`接入附件宿主、评论文件、正文图片与草稿文件四个测试文件的afterEach，对三条通道断言无`/api/assets/v2`、file-assets、bulk、旧restore或multipart（50项通过）。两项变异均被杀：收集箱无条件发bulk、评论创建后补旧bulk绑定（报`old service /api/assets/v2/.../bulk/`）。

门禁：`make knip`通过（顺带清掉T7/T8遗留：`DownloadFailure`/`useImageAvailable`/`ImagePending`改为模块内私有，`ECustomImageStatus`删去从未使用的UPLOADING/DUPLICATING/DUPLICATION_FAILED）；`make lint-web` 50/50（删除旧文件少了2条oxlint告警，web上限69→67）；`make test-web` 16/16（services包任务消失），web直接337文件3804测试；37个修改文件零warning且格式正确。root.store的store数量断言25→24。关键词：`services-files`规则由文件白名单改为挡住该目录任何文件，新注释中的品牌词改写；全仓旧名字只剩关键词守卫规则与样本（用来禁止回流），文档历史除外。plane-diff、前端改动清单3.1及M1/M1-P2/M1-P3/M2/M4交接已登记处理结果，各交接仍open。T12原件t12-consumers、t12-inbox-1~3、t12-mut-inbox、t12-legacy-1/-2、t12-mut-legacy、t12-pnpm-install、t12-build-packages/-editor、t12-knip-1/-2、t12-keywords-1/-2、t12-lint-web-1/-2、t12-test-web/-2、t12-web-all、t12-lint、t12-format，及T11补正的t11-fix-e2e-types。


T13真实页面：`e2e/stories/files/`新增f8–f12与f16六个故事文件，`[local]`由`playwright.m5-content.config.ts`（`pnpm --dir e2e run test:m5-content`）在Chromium/Firefox/WebKit运行，`[s3]`变体用基础配置在Chromium运行（沿P6）；共享helper`fixtures/content-files-page.ts`（成员登录、请求与失败监测、传输与持久化断言、下载SHA、粘贴/拖放）。F8：附件全部动作与51项分页、计数与双向排序、下载SHA、上传中取消（不删已收字节、无complete）、未知DELETE经核验由用户再次确认；F9：两评论私有准备与一次绑定、拒绝后保留文字与文件待显式再发、未知响应先核验评论列表不重发、现有评论加文件与作者删除（恰一次DELETE、只重读该评论列表）；F10：无项目草稿附件与图片、移动、未完成文件拒绝发布与评审丢弃、两标签页并发发布、丢失prepare回答、另一标签页上传中、普通副本与草稿副本的最终实体；F11：伪造/无权源与目标拒绝、粘贴恰一次服务器副本且保存副本ID、Undo/Redo、普通文件不吞；F12：换图、调序、历史（他人版本经内容读取解码、无文件输入）、只补ID、上传中替换占位（被替换传输无complete）；F16：详情与peek联合链（422正文、丢失评论编辑、未发送评论→上传→规范重读503→他人正文→Retry→保留与冲突→DELETE→显式保存）、403/404结论（同ID重现不复活）、401续期重发、真实撤销401与另一账号登录、另一实例与换宿主、后退/前进、换工作区、重连、权限ABA（访客只读、他人评论文件不可删）、归档只读、授权到期续期一次。

T13发现并各自单独修复的产品缺陷（均RED→GREEN，按本地缺陷处理、可单独回退）：`370aba81` browse路由在确认命令的规范重读pending时卸载IssueDetailHost及其保留输入/收据，并补确认后焦点归还；`604829a7` 评论未知结果核验被已绑定文件拒绝，用户到不了“I checked”；`86c9f9f6` **M2缺陷**：auth中间件以`options.fetch(copy)`重发续期副本，浏览器拒绝非window接收者（Illegal invocation），M2设计7.1“401续期一次并重发一次”在真实浏览器从未生效，FakeNerve客户端fetch现与浏览器一样检查接收者；`0d093f03` browse路由以SWR瞬时isValidating代替“本次访问的读取已证实”——温缓存首帧挂载交互详情，重连重读已证实路由并卸载宿主（丢失未发送评论、保留输入与收据），改为每次访问自发唯一lookup、应答前不挂载、关闭隐式reconnect（同M5设计§226），useSessionSWR增窄选项revalidateOnMount:false。

Step 2：`make build build-s3-fixture`后m5-content配置[local]三浏览器63/63（t13-content-local-5），[s3]变体Chromium 9/9（t13-content-s3-3），两次运行的输入清单一致（t13-build-8之后）；local与S3各实际上传/显示/下载，接收端核raw字节与凭据；expires_at响应注入只证明UI到期续期一次，不冒称服务端TTL。中间全量t13-content-local（首跑49/51，ABA故事两浏览器因SWR去重与中止读取失败，见Ruling）、t13-content-local-2 54/54、t13-content-local-3 60/60（加入换工作区故事前）、t13-content-s3 9/9、t13-content-s3-2 9/9保留；加入换工作区故事后的t13-content-local-4有两项超时（WebKit F8首次选择无filechooser事件、Firefox换工作区故事的PUT未到路由），其余61项逐项耗时不变，隔离重复8/8与保留产物的全量重跑63/63均未复现，按自动化层间歇记入T14/CI观察（见Ruling）。

Step 3变异（`mutate.py`逐项施改、经run-check运行、恢复并核SHA，全部restored=yes，结束于干净make build）：1漏取消后complete（F8）、2漏节点代次（F12占位）、5未知POST自动再发（F9）、7以loaded数代总数（F8）、9撤销发DELETE（F11）、10到期无限再读（F16到期）、11宿主不保留输入（F16详情联合链，实际内容丢失）、12续期副本不发出（F16续期）、13暂时撤回当结论（F16 peek联合链）均以断言杀死；8伪剪贴板免copy首跑只因超时失败，F11改为显式断言后杀死（-2）；6 confirm后刷新重复命令首跑在F8存活（issue宿主规范重读结束view并中止重复DELETE），F9新增作者删除现有评论附件后杀死（-2）；3项目可见即当issue权限与4草稿ABA复活在页面上等价（3被组件编辑门、4被useContentView丢弃已结束view与资产root登记双层遮蔽），在所改层的单元测试上以断言失败（03-unit、04-unit），同类另加3b他人评论文件删除闸放宽为任何项目角色，F16权限ABA故事杀死；另加14重连重读路由，F16重连故事杀死；15离开后迟到完成（两处卸载处置同时失效），换工作区故事杀死，同一工作区的后退/前进故事另有一层遮蔽。首跑原件全部保留（t13-mut-01…15及-2/-unit/3b）。

T13原件：t13-build~-8、t13-route-red/-green、t13-focus-red/-green~-3、t13-committed-red/-green、t13-resend-red/-green、t13-visit-red/-2、t13-visit-green~-3、t13-web-units/-2、t13-fix-*与t13-fix2-types、t13-content-local~-5、t13-content-s3~-3、t13-mut-*。最终F1–F18矩阵与默认部署演示仍归P8（规格§5）。

## 9. T14 门禁与接收核对

T13记录提交为`e6e2bc15`。验收与证据逐项对应如下（证据标签为本阶段ignored目录中的run-check原件，输入前后一致；页面结论不以单元测试替代）：

| 验收 | 证据与观察边界 |
|---|---|
| A1 F8 | T1服务与有限下载、T2/T3资源与读取、T4附件控件、T5详情/peek宿主、T11计数/排序/显示偏好（含服务端attachment_count键）；T13 F8真实页面（三浏览器local与S3变体）：选择器与拖放上传、51项分页、下载SHA、计数与双向排序、上传中取消、未知DELETE核验后显式再确认；变异1/6/7。页外附件由服务端`attachment_count`计入，不按已加载样本。 |
| A2 F9 | T6评论私有准备与一次绑定、T9评论正文图片；T13 F9（私有准备、拒绝保留、未知响应核验、现有评论文件与作者删除）与F16联合链（未发送评论、失败的正文与评论编辑在规范重读中保留，不重复写）；`604829a7`；作者/管理员/访客边界：F16权限ABA（访客只读、他人评论文件不可删）与content-view单元；变异5/6-2/11/13/3b。 |
| A3 F10 | T10草稿附件/图片、刷新/改项目/发布、未知ID准备件恢复、普通与草稿副本；P7a协议与API为前置；T13 F10三项（无项目草稿与移动、未完成文件拒绝与评审丢弃、两标签页并发发布、丢失prepare回答、另一标签页上传中、副本的最终实体），local与S3；双发布与迁移失败沿P5证据并补页面拒绝；变异4在其所改层的单元测试杀死。 |
| A4 F11/F12 | T7/T8编辑器图片端口与包内迁移、T9正文图片；T13 F11（伪造/无权源与目标拒绝、粘贴恰一次服务器副本、Undo/Redo、普通文件不吞）与F12（换图、调序、历史、只补ID、上传中替换占位），HTML只存UUID；`skip_activity`只在内容标识不变时保留审计（服务端`image_identity_test`，既有），T9保存路径不以它绕过图片核验；变异2/8-2/9。 |
| A5 F16/F17 | T2/T3/T5生命期与宿主；T13 F16 13项：换账号（真实撤销401后另一账号登录）、换工作区、项目权限ABA、工作项与另一实例、后退/前进、重连、迟到成功/失败/进度（扣住PUT后离开，规范重读503/403/404）；F10草稿移动/发布；拒绝可读（role=alert）、焦点（选择后到动作、上传/删除后归还，`370aba81`）、繁忙不可重入（按钮禁用、评论发送禁用）、明确取消（F8上传中取消）；忙碌删除拒绝Escape/backdrop与重复提交在组件层（`list.test`）；`86c9f9f6`、`0d093f03`；变异10/11/12/13/14/15。 |
| A6 | T12删除旧文件链与`@nerve/services`；重扫`t14-consumers`共18处命中，全在守卫或样本（关键词规则、遗留流量监视、断言无旧路由的测试），消费者0；`make knip`通过（`t13-knip`，死成员扫描，Step 2再跑）；无新增依赖（只删`@nerve/services`与`file-type`链，e2e只加脚本）；守卫只收紧（web lint上限69→67，services规则改为挡住整个目录）；本分支新增/修改的218个文件逐包实际检查oxlint 0警告0错误，另3个按共享ignorePatterns排除（两份`*.config.ts`与生成的`schema.gen.ts`）（`t14-oxlint-modified-4`；-1～-3是计数与汇总校验的原件）。 |

跨M的本地修复（向负责人明示）：`86c9f9f6`修复M2认证中间件续期副本的重发（M2设计7.1在真实浏览器从未生效）；`370aba81`与`0d093f03`修改M4的browse详情路由（只在路由读取进行时显示骨架屏；每次访问自发lookup、关闭隐式reconnect），`useSessionSWR`增窄选项`revalidateOnMount: false`。T14门禁又发现一处：`5ff2e4a2`——`370aba81`让browse路由在写后规范重读期间保留IssueDetailHost（与宿主“暂时核准保留owner”的设计及peek一致）后，宿主里的弹窗开关跨过了正文撤回，而弹窗表单随正文卸载、完成回调按自身作用域跳过关闭，重读后正文重挂载即重开空弹窗（I8/I9/I10；peek自M4起同样如此，无故事覆盖）；改为弹窗开关属于绘制它的正文，规范项在/不在每次变化都换新的IssueDetailModals，规格§4.1要求保留的输入仍只随宿主scope。四者都是页面、宿主或单文件层的根因修复，不改M4通用权限/读取算法（DetailResource、IssueVerification、权限观察器），按§7预检约定在此提交证据并上报；均可单独回退。另有编辑器包表格插件的既有缺陷`8d85db5a`（视图销毁不取消首次扫描，P7b按图片能力重建编辑器后在C12故事表现为页面错误），在插件处修复。

原件：本阶段384份run-check收据中，T13/T14的全部收据输入前后一致；只有四份输入有变，均属设计或已记录（`t11-gen`生成、`t12-pnpm-install`改锁文件、`t11-server-bootstrap`/`-green`长Go运行期间只改web文件，见§8 T11）。T13/T14的33份非零退出都是有意保留的原件（RED、变异、`t13-content-local-4`间歇、修正与校验迭代）。文档链接：M5文档、前端改动清单与plane-diff共54份、409个相对链接全部可解析（`t14-doc-links`）。

文档与交接：M5设计§9补browse路由读取的reconnect与访问读取规则；前端改动清单3.1文件行与3.2 M4详情行更新；M2/M4交接追加T14核对，各交接保持open；plane-diff无变化（T13不改服务端，T11的display_properties键已登记）。

Step 2七门禁（串行、`run-check`收据，不改预算/workers/期限，未操作既有容器）：首轮`t14-*`中gen-check、lint-go、lint-web、knip、test-web通过，test因archtest发现T11迁移00038未登记到sqlc的project条目失败，e2e 415中9失败；逐项根因修复并各自单独提交——`dd60ef8a`（sqlc登记，make gen无生成物变化）、`5ff2e4a2`（I8/I9/I10确认后弹窗以空表单重开，见上文跨M修复）、`8d85db5a`（C12表格页面错误）、`55f2b487`（I11评论POST体与editor-loading detail变体按P7b合同更新，creation变体不变）。第二轮`t14-*-2`：e2e 415/415，lint-web因本人新测试的局部变量遮蔽（no-shadow）失败，`ae4e7440`改名；test只在`asset/adapter/postgres`一个包失败——testcontainers启动postgres时Docker API查询容器状态连续超时（context deadline exceeded），包内其余用例因缓存的启动失败即刻失败，该包随后单独运行通过（`t14-asset-postgres-alone`）。第三轮`t14-*-3`七项输入清单同一（digest `986993e2…`，前后不变）：gen-check、lint-go、test、lint-web、knip、test-web全部通过；e2e 414/415，F10页面[s3]变体在S3夹具启动阶段失败（16.3秒，远未到90秒期限，尚未执行业务步骤），原因只见“S3 child startup failed”——夹具子进程吞掉原错误（与P3-M1同类的诊断缺口），Docker事件缓冲已被其他项目容器的健康检查挤出，无法事后取证；与第二轮及P5 T13记录的Docker查询超时同类，按P5先例不改设置，以同一输入完整重跑`make e2e`（`t14-e2e-4`）415/415。最终同一输入（HEAD `ae4e7440`）七门禁：gen-check 4秒；lint-go 2秒；`GOFLAGS=-p=1 make test` 988秒（61个测试包、21个无测试包，-count=1，bootstrap 644秒）；lint-web 50任务/33缓存（web实际执行）；knip通过；test-web 16任务/15缓存（web实际执行337文件3815项，缓存项为相同输入的回放，如编辑器包135项来自第二轮实际执行）；e2e 415/415（205秒）。其他项目的15个容器前后名称与镜像一致，只有testcontainers自身的ryuk收割器短暂存在并自行退出。

P8未完成事项（规格§5、M5设计）：M6/M7/M8领域接线、最终F1–F18矩阵与默认部署/容量演示、P3-M1清理错误诊断、P4-M1死字段、全仓新warning规则清零与死成员、复制为Markdown省略图片的具名风险、全仓旧第三方图标消费者核对、七个输入交接的最终关闭；另有T13记入T14/CI观察的两项间歇（`t13-content-local-4`）；M4既有：写后规范重读期间弹窗表单卸载，链接/子项/关联的成功toast被跳过（M4详情页已如此，写入与列表正确，交M4交接）。S3夹具子进程启动失败时应回传原错误（T14第三轮只见通用信息，扩充P3-M1）。

## 10. T15 整分支审查与修复

一次fresh whole-branch审查：冻结main基线`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`..T14检查点`0716b12f77468159726bd6c8d6e6cf154809de24`（分支CI 42079747995三作业首次全绿），只读子代理（显式指定最强档`fable`）带计划Review Focus原文、全部局部Ruling、跨M修复清单，按逻辑/集成/测试三视角审查25个提交、298个文件，生成物逐一对源；报告见[re-50](../../../reviews/re-50-M5P7b整体审核.md)，审者原文`t15-review-raw.md`在ignored证据目录。没有逐Task派审者或第二个同范围审者。

结论With fixes：五项Review Focus全部PASS；四处跨M修复（`86c9f9f6`、`370aba81`/`0d093f03`、`5ff2e4a2`、`8d85db5a`）判为页面、宿主或单文件层的根因修复，不改DetailResource、IssueVerification（仅新增只读`isMissing`）、PermissionReads或权限观察器。主控按效果复核后的处理：

- F1（Important，维持）：设备休眠或断网超过内容读取期限后，到期计时器那次唯一续期常在网络恢复前发出、未送达，此后的online/focus/visibility唤醒只处理ready条目，页面图片停在“无法加载/重试”。`6baaf7bf`：网络未送达（TypeError）的到期续期由之后每个唤醒再试一次，服务端回答（含503）仍待显式重试；新用例先红后绿，变异“任何失败都可唤醒再试”起初存活，既有用例改为推进时间后断言即被杀死。
- F7b：两个只有自身测试读取的`suspended` getter——knip看不到类成员，T14死成员扫描漏掉此类；按T14合同未完成项删除（`6058fe2b`），并对本阶段新增/修改文件的类成员补做读者扫描，再无无读者成员。
- Minor F2–F6、F7a（传输中拖动图片块、未知发布后的“草稿已保留”提示、被拒文件批的原生粘贴、删除核验404后的死路、保存保留输入时重建编辑器、确认收据常驻）不在本阶段修改，交P8；审者暂不判断的8项逐条裁定维持，见re-50§6。

T15七门禁（修复后HEAD `6058fe2b`，七项输入清单同一，digest `93f855b4…`前后不变）：gen-check、lint-go、lint-web（50任务/33缓存，web实际执行）、knip、test-web（16任务/15缓存，web实际执行337文件3816项）、e2e 415/415通过；`GOFLAGS=-p=1 make test`首跑只在`project/adapter/postgres`一包因testcontainers-go会话收割器容器在就绪检查中消失而失败（未执行业务断言，同M4/P7记录的Ryuk INFRA失败；Docker VM内存与磁盘无压力），按先例不改设置与期限，以同一输入完整重跑通过（`t15-test-2`，61个测试包，bootstrap 654秒）。其他项目的15个容器前后一致，测试容器全部由收割器清理。

P8新增事项：上述Minor；测试设施在本阶段三次出现Docker侧停顿（第二轮Go包查询容器状态超时、第三轮S3夹具子进程启动失败、T15收割器消失），均未到业务步骤——`go test -p=1 ./...`各包共享一个testcontainers会话时收割器退出与复用之间的竞态及S3夹具子进程吞错（P3-M1）一并交P8测试设施故障演练。准确候选分支CI与main集成是T15 Step 3与T16的前置，本节写成时尚未合并，结果由T16回填。

T15候选`f26630b6`的分支CI [42097010743](https://github.com/open-nerve/NerveProject/actions/runs/42097010743)：attempt1中server、web通过，e2e在25分钟作业期限处被取消——`playwright install --with-deps`一步因GitHub托管runner的Ubuntu apt镜像极慢耗时23分钟，故事未及运行；不改预算，原始元数据与日志保存后以`gh run rerun --failed`重跑。attempt2跑完414通过1失败（22.5分钟）后仍超作业期限被取消。失败的是M4故事C12（review）history Markdown：CI trace中故事等到的PATCH回包是编辑器给无块标识修订补data-id的迁移保存（防抖1.5秒），慢runner上它恰与粘贴同刻发出，故事随即重载，粘贴的防抖保存尚未发出。根因是故事把粘贴后第一个PATCH当成粘贴的保存；`6b9d8051`按同族cleanup-interactions的做法先等迁移保存应答再粘贴，产品不变。临时探针扣住迁移保存，“原顺序”确定性复现失败、“先等应答”通过（`t15-race-probe-red-2`，探针用后删除、副本随证据归档），修正后的故事连续3次通过（`t15-history-story-green`）。

最终候选的门禁：受影响的gen-check、lint-go、lint-web、knip与e2e（415/415，205秒）在最终输入上重跑通过（`t15-*-2`）；Go全量（`t15-test-2`）与test-web（`t15-test-web`）复用——输入清单逐文件比对，与最终输入只差T15三份文档与这一个故事文件；e2e包没有`test`脚本，`make test-web`不运行故事，Go不读e2e（P5“精确替换限定复用”先例）。

## 11. T16 集成、归档与接续

准确最终候选`a5cb9d65bb3715ddea713689953114232a466d3e`（只在T15候选之上加C12故事修正`6b9d8051`与本记录§10的补记）的[分支CI 42109712732](https://github.com/open-nerve/NerveProject/actions/runs/42109712732) attempt1三个作业均success：server127533730693、web127533731202、e2e127541590787。E2E 415/415，测试21.2分钟、作业23分钟（浏览器依赖安装22秒、构建41秒）；同一套件在T14检查点CI为16.9分钟，差异来自runner，作业离25分钟期限只余约2分钟，记入P8观察，未改预算。

根main原基线`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`与origin一致且干净，worktree干净，本地与远端分支都是该候选（30个提交，main为其祖先）。`--no-ff`合并生成`2b2c6f33e3a93536742d5f28b74eb959ba73d6d1`，双父依次为原main与最终候选，完整tree与候选相同；系统Git推送成功，主目录未切开发分支。[准确main CI 42115811156](https://github.com/open-nerve/NerveProject/actions/runs/42115811156) attempt1三个作业均success：web127555018192、server127555018588、e2e127563759440；E2E 415/415，测试20.5分钟、作业22.3分钟。无需重跑，没有用分支CI替代main结果，原始run/all-jobs与E2E日志单独保存。

归档根：`/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M5-P7b-integration-20261106`。P7a的`archive.py`先按其manifest记录的成员SHA-256（`6e855b19…`）核对一致，再复制进本阶段工作区；会话scratchpad（门禁与探针脚本、CI原件、浏览器运行、容器快照、合并与归档脚本，4,328个文件）一并放入工作区后归档。包含Codex实施T1–T8与接手后T9–T16的全部台账、run-check收据与原件、失败/变异、审者原文、浏览器报告和全部`.turbo`。共7,397个文件、676,283,877字节、9,850个成员；18个证据根与30个可重建路径（node_modules、dist、build、bin、`.react-router`、webui dist）逐项分类，unknown=0，源在归档期间未改变，manifest head为最终候选。manifest SHA-256为`6ff41dedb451349bef20bfb3d22e2af4092520ff1dfbf9ada71449b32c050818`。

确认本地main=origin/main=`2b2c6f33`、`origin/main..codex/m5-p7b`与`origin/main..origin/codex/m5-p7b`均为0后，正常删除远端分支、worktree及本地分支（was `a5cb9d65`）；未使用force/stash/clean/reset-hard，远端只剩main。清理后从归档副本再次核对全部成员，与清理前一致。[清理收据](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M5-P7b-integration-20261106/cleanup-receipt.json)与[清单](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M5-P7b-integration-20261106/manifest.json)位于冻结payload之外，原件不再改写。

P8 worktree从该main以Git创建于`.codex/worktress/codex/m5-p8`（分支`codex/m5-p8`）；本阶段只写P8 spec/plan并做一次fresh builder预检，不安装依赖、不实施，详见[re-51](../../../reviews/re-51-M5P8预检.md)。T16最后完成收据在P8 worktree的同名P7b接续台账登记，冻结归档保留清理前现场。M5最终矩阵、交接与完成勾选继续由P8/closeout承担。
