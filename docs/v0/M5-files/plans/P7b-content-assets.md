# M5/P7b 工作项附件与编辑器图片 Implementation Plan

> 2026-11-05：正式预检通过，已获准从T1实施；阶段结束经deep-review、noff/main CI及归档后继续P8和M5收尾。分支`codex/m5-p7b`，基线main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`；P7a准确main CI41869045846、全量归档及正常清理已核。一次独立builder发现的局部问题已修订合同并自检闭环，产品验收仍待实施。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 逐Task内联实施；不逐Task新派实现者/审者，阶段末一次fresh whole-branch review。正式分支`codex/m5-p7b`已从main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`创建；一次fresh builder的局部问题已按现有owner和输入寿命补实，main CI、归档清理和文档自检前置已闭环，实施已从T1开始。

**Goal:** 工作项/评论/草稿附件与编辑器图片真实可用，列表计数/排序接通，旧文件调用链删净。
**Architecture:** 当前会话 AssetService 与唯一 UploadSession 处理协议；ContentView 经 IssueRootStore 注册生命期，DetailResource/ContentReads 管已授权的当前资源；editor 包声明可选图片端口，应用适配，业务 HTML 只保存 UUID。宿主保存及草稿迁移沿现有原子 API。
**Tech Stack:** 已锁定 React/MobX/SWR/Tiptap、Vitest/Playwright、原生 File/fetch/AbortController 与既有 Go local/S3；不加依赖。
**Spec:** [主合同](../specs/P7b-content-assets.md)、[资源与传输](../specs/P7b-content-lifecycle.md)、[编辑器](../specs/P7b-editor-images.md)。

## Global Constraints

- 根目录永远 main，实施仅本 worktree；plane/refer只读、不在其中跑git。详细中文提交；系统 `/usr/bin/git push`；合并 `--no-ff`，精确 branch/main CI 分开核验。
- 无新表/服务端协议/依赖、无新全局实例/平行权限观察器；不放宽 M3/M4 架构守卫，不重开 PC1/PC2/PC3 已批准合同。所有实体直接用生成 DTO。
- 新文件约400行内；既有大文件只拆本次职责，生成物例外。最多16Task，计划不抄整段实现，不把未实现事项写成已验证。
- 图片 JPEG/PNG/WebP/GIF；附件空MIME→application/octet-stream，其他非法MIME拒绝；文件正字节且≤当前limit，名字1–255 Unicode字符、无分隔/控制字符。无配置拒绝，不用旧上限常量补授权。
- 普通附件一次1个；editor一次最多10图、每view最多2传输，整批超量/混合拒绝。PUT原File、omit、redirect error、no-referrer、15分钟；下载有限15分钟且实收字节等于Asset.size≤104857600。
- 真正的已有宿主才开放文件：无new_issue；new_comment的scope.id为issueID；draft可project=null。评论普通附件最多100，正文图片不得混入attachment_ids。
- URL/expiry仅运行态；内容是UUID，UI失效不声称收回已发S3能力。未知非幂等结果不得自动重发；删除正文节点不DELETE资产；草稿发布不前端搬文件。
- 新文案同步en/zh-CN，错误沿 `core/lib/error-messages.ts`/现有拒绝处理；改文件oxlint零warning；不改shared ModalCore/动画。附件控件当场用现有中性图标。
- 代码/生成物稳定才串行七门禁，构建/typegen不与源码编辑或测试清dist竞争。容器套件一次一个、Go跨包-p 1；只清理本阶段确切拥有的临时容器，不操作既有容器/Docker。
- Task测试先实际断言RED再GREEN；缺模块/类型/环境失败不算行为RED。输入hash/原输出/缓存/skip/变异恢复SHA均保留。已绿后仅因改动/失败/未决风险扩大或重跑。
- 实现沿P7a保持phase=M5/P7，拆分后共用现有数字阶段值，不改PHASE_RE；到期例外在最后消费者迁移时删除，不延期掩盖。M5最终warning/交接/部署验收归P8/closeout。

## Review Focus

1. complete/DELETE已执行但回复丢失，或确认成功后列表刷新失败：不能重复上传/重复删除，必须分离命令结果与只读核验；权威重读卸载正文时保留未提交输入/已确认收据，旧view照常结束（T1–T6/T9/T13）。
2. 工作项可见性、评论作者、私有无项目草稿、项目移动及权限A→B→A：不能用P6项目封面权限替代内容权限，旧能力不能恢复（T2/T3/T6/T10/T13）。
3. 多节点同源、后台到期/重连、晚到onError、复制中Undo再Redo：请求有界，旧节点不能改新节点，不能自动重试非幂等复制（T3/T7/T8/T13）。
4. 新评论准备/未知POST、草稿未保存/复制/发布失败：文稿和已完成准备件保留，不误绑、暗建宿主或按半完成状态提示成功（T6/T9/T10/T13）。
5. 纯非图片/混合文件、伪造剪贴板/外部src、旧复制监听、未来领域wrapper：不吞输入、不越权复制、不发旧API、不留下无使用兼容层（T7/T8/T12/T13）。

## 文件与接口约定

`core/` 和 `helpers/` 相对 `web/apps/web/`，其他路径相对仓库。新文件配同目录 `.test.ts(x)`。公开签名集中在两个子spec，本计划的Interfaces按名称引用；T7定义包消费接口，T8同一Task完成包和wrapper形状迁移，不能先交一个不通过类型的版本。局部实现细节由执行者决定，不改合同；每次新局部裁定记ledger和阶段review。

| 组 | 文件责任 |
|---|---|
| JSON/字节 | asset.service.ts 扩展；新attachment.service.ts、lib/asset-download.ts；原lib/asset-upload.ts只有确有共用的纯URL校验可提取 |
| 内容状态 | store/asset/{content-view,upload-policy,attachment-resource,content-reads}.ts；upload-session仅协议，root.store工厂和释放 |
| 通用附件 | components/assets/attachments/{picker,list,item,delete-dialog}.tsx；use-content-assets hook只组装当前view，不获取目录或新建API |
| 编辑器适配 | hooks/editor/use-editor-assets.ts及内部按节点上传管理；editor/types/config.ts声明消费端口，节点/toolbar/drop/paste消费 |
| 宿主 | issue-detail/peek/widgets、comments、issue-modal/draft-form与已有draft布局；父组件拥有保存/发送/切换/发布 |
| 列表 | constants/issue/common.ts、all-properties、display-properties/order-by、spreadsheet attachment-column与原布局偏好验证 |
| 删除链 | 旧FileService/FileUploadService/EditorAssetStore/useEditorAsset/useEditorConfig/TFileHandler/EFileAssetType/旧file helpers/services包及独有依赖；URL规范化搬入web/lib保持原测试 |

## Task 1（T1）：补齐原生资产服务和有限下载

**Files:** 修改core/services/asset.service.ts；新attachment.service.ts/.test.ts、lib/asset-download.ts/.test.ts；回归lib/asset-upload.test.ts；tools/keywords.json；阶段review。
**Interfaces:** AssetService.read/remove/duplicate/content/listDraftPreparations、AttachmentService.list、downloadAsset，签名见资源spec§1/5/6；准备件查询消费P7a生成DTO；原prepare/complete/putAsset不变。

- [x] Step 1：新增服务请求表驱动测试，断言四asset端点、三附件端点及准备件查询、limit50/cursor/path/signal、204、原Problem；source copy不转换为前端先取目标权限。下载用受控ReadableStream验证原URL/query、omit/redirect/no-referrer、精确字节、过短/过长、已过期、用户取消/超时、失败body回收、objectURL释放；不能用jsdom冒称真实Cookie验证。
- [x] Step 2：`pnpm --filter web exec vitest run core/services/asset.service.test.ts core/services/attachment.service.test.ts core/lib/asset-download.test.ts core/lib/asset-upload.test.ts`。先记录有效断言RED，再实现；phase=M5/P7。内容schema的正确类型名为生成的Content，不能写假AssetContent。
- [x] Step 3：同命令GREEN；API客户端及实际传输受影响测试、类型/格式/零warning；提交 `feat(M5/P7b): 补齐内容资产接口和有限下载`。

## Task 2（T2）：共享上传状态机与内容视图

**Files:** 修改store/asset/{upload-session,root.store}.ts；新upload-policy.ts、content-view.ts及测试；profile策略从原实现提取，ScopeLifetime仍服务P6；回归store/root.store.test.ts和issue/permission-reads.test.ts。
**Interfaces:** 原createUpload(ProfileUploadTarget)保持；openContent(ContentHost)→ContentView、createContentUpload(view,use)→UploadSession；消费侧UploadPolicy见资源spec§3。

- [x] Step 1：实际RootStore/FakeNerve覆盖六用途映射、名字/大小/MIME、草稿null项目、未知host零请求、普通附件不建预览、描述图片建并释放预览；P6四用途与ready/failed同步observer重入保持。分阶段暂停后失权/恢复/换issue/评论删除/草稿A→B→A，旧start/成功/失败/retry永久无效，其他工作区独立，issue/comment的无关项目不误杀；私有draft按既有owner集合范围失效，规范Draft撤回时结束view，重新读取后新建，不能私持旧DTO继续。
- [x] Step 2：`pnpm --filter web exec vitest run core/store/asset core/store/root.store.test.ts core/store/issue/permission-reads.test.ts core/store/issue/root.store.test.ts core/store/workspace`。RED后提取一份UploadSession算法；ContentView只经IssueRootStore注册，host观察仅身份/归属，无新角色观察器；每await及同步状态通知后核current。
- [x] Step 3：GREEN；单改取消后继续complete、内容使用项目封面权限、永久代次改成ID相等三变异分别AssertionError并恢复SHA；当前P6上传、PC1/PC2/PC3联合回归、类型/lint；提交 `feat(M5/P7b): 统一内容上传和宿主生命周期`。

## Task 3（T3）：分页附件与临时内容能力

**Files:** 新store/asset/{attachment-resource,content-reads,draft-preparations-resource}.ts及测试；root.store持有/释放子资源。
**Interfaces:** AttachmentResource接口、ContentReads.read/retry/acquire及ContentReadLease按资源spec§4/5，DraftPreparationsResource按§6；DetailResource/Reconciled只复用，不改变通用算法；issue/comment使用明确项目范围，私有draft资源由Reconciled与当前view调和，禁止再套项目成员角色，仍服从规范Draft被owner撤回的现有寿命。准备件独立于已绑定附件，未知DELETE先只读核验，不能以列表缺席冒称删除成功。

- [x] Step 1：51+附件、非名称/时间顺序UUID、首读与分页交错、重复cursor、同cursor双击、删除回包与旧首读、确认成功后刷新拒绝；读重试不再DELETE/PUT。多节点同asset、两个view同asset、expiry、后台恢复、重复error/旧load、实际load恢复预算、最后lease退出全部取消。
- [x] Step 2：`pnpm --filter web exec vitest run core/store/asset core/store/issue/issue-details` 记录RED后实现；issue附件后用现有refreshAuthoritative(afterWrite)和refreshAfterChange更新计数/活动，comment/draft不动issue count。readAsset必须验证真实purpose/host后才接content。
- [x] Step 3：GREEN；变异忽略晚到请求号、把页长度当总数、去掉expiry预算均失败；核current resource不能保留跨host capability，类型/格式；提交 `feat(M5/P7b): 管理附件分页与短期内容授权`。

T3与T5衔接：命令确认收据先发布给稳定host再触发权威重读；pending造成规范Issue缺席时旧ContentView照常结束，不能让其错误覆盖已确认结果。读取重试只GET，完整输入所有权见资源spec§4.1，由T5实现；T3不私存旧Issue或改变通用读取算法。

## Task 4（T4）：附件选择、列表和删除确认

**Files:** 新components/assets/attachments/{picker,list,item,delete-dialog}.tsx及草稿准备件恢复控件与测试、hooks/use-content-assets.ts；en/zh-CN文案；只读对照Plane对应附件文件。
**Interfaces:** Attachments({view:ContentView,resource:AttachmentResource})组合；单个附件项消费生成Attachment与download动作；删除ModalCore薄包装。准备评论的Asset队列可由同一无网络列表呈现组件消费明确投影，不伪造成服务器已绑定列表。

- [x] Step 1：真实input/drop、单文件/目录/多个拒绝、阶段进度无假百分比、双击同链、明确取消、确认失败恢复、读刷新失败分离；删除拒绝保留dialog/文件，成功返回有效触发器，busy时Escape/背景不关闭；状态aria-live、错误role=alert且焦点可达。
- 恢复控件同Task验证pending/uploaded/时间/大小/用途，逐ID显式丢弃、另一标签页在途/已成附件的提示；已绑定描述图拒绝保留，普通附件沿原删除合同，不按文件名猜或后台全删。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/assets/attachments core/store/asset` RED→GREEN。原生选择器、无新包；文件名和大小按原DTO，中性图标，无button嵌套button或下载链接持久能力；删除不使用finally无条件关闭。
- [x] Step 3：原ModalCore相关既有回归、类型/翻译/格式/零warning；提交 `feat(M5/P7b): 提供可访问的附件操作控件`。

## Task 5（T5）：工作项详情与 peek 附件

**Files:** 新components/issues/issue-detail-widgets/attachments/{root,content}.tsx及测试；修改issues/issue-detail/main-content.tsx、peek-overview/issue-detail.tsx；issue-details/root.store只补已需要的widget入口。
**Interfaces:** 已读取Issue的workspace/project/issue身份→ContentView(issue)，T4控件；沿现有IssueDetailHost与useDetailScope。

**输入连续性补充（builder局部裁定）：** 增补`components/issues/detail-host.tsx`、新`detail-input-state.ts`及相邻context/hook、`title-input.tsx`与测试。IssueDetailHost在规范Issue条件子树外持有IssueDetailInputState，只保存资源spec§4.1列出的本地输入/基准/已确认收据；沿原host权限身份清理，不改PermissionReads。为T6/T9提供按表单/字段/commentID区分的状态槽，不提供owner DTO或网络能力。详情/peek两个实例不能串状态。T5加入真实RootStore的pending→成功/失败、标题dirty、确认收据先于卸载与只读重试测试，T6/T9再扩评论和正文。

- [x] Step 1：详情和peek真实宿主、只读/归档/访客拒绝入口、成功complete一次绑定、刷新失败只读重试；关闭peek/切工作项/Back后迟到回答不建列表、不toast；有51条时只首屏50但DTO count完整。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/issues/issue-detail core/components/issues/peek-overview core/components/issues/issue-detail-widgets core/store/issue` RED→GREEN；未加载/不确定宿主不创建上传能力，已观察删除立即退出。
- [x] Step 3：原链接/子项/关系/活动与权限错误回归，记录Plane视觉行为差异；提交 `feat(M5/P7b): 接通工作项详情附件`。

## Task 6（T6）：评论普通附件与一次原子发表

**Files:** components/comments/{comment-create,card/edit-form,card/display}.tsx、issues/issue-detail/issue-activity/helper.tsx及评论测试；只删本Task实际替换的bulk绑定链，图片端口整体迁移由T8/T9。
**Interfaces:** new_comment ContentView→COMMENT_ATTACHMENT准备件；IssueCommentCreate.attachment_ids≤100和comment_html一次POST；已有comment AttachmentResource独立列表，CommentUpdate不加不存在的附件字段。

**跨卸载责任：** 评论创建及编辑的未提交HTML、已确认准备件ID/选择、已知拒绝或未知结果提示接T5 host状态；暂时权威重读失败仍保留，两表单独立。旧view结束，fresh owner及资产核验成功后才允许发送；重挂载不自动POST。Step1必须包含“未发送评论/失败编辑→另一个普通附件complete或DELETE→pending/读取拒绝→只读重试→内容仍在且命令次数不变”。

- [ ] Step 1：新评论未发送私有件、两个表单不混用、普通附件与正文图片集合不混、101拒绝、未完成不能发送、同tick双发一次；已知拒绝保留文稿/IDs，网络与无Problem502未知结果提示先核评论列表，刷新只能GET。已有作者新增/删除，管理员可删不代表能新增，非作者无控件且服务端仍拒绝。
- [ ] Step 2：`pnpm --filter web exec vitest run core/components/comments core/store/issue/issue-details` RED→GREEN；移除成功后FileService bulk status。comment-only附件不影响issue attachment_count，空正文仍遵守原非空合同。
- [ ] Step 3：评论成功后的清表单/焦点和活动刷新、删除评论中旧附件回答、换issue/账号联合回归；提交 `feat(M5/P7b): 原子绑定评论附件并保留拒绝状态`。

## Task 7（T7）：应用侧编辑器图片端口

**Files:** 新hooks/editor/use-editor-assets.ts及按节点队列小文件/测试；editor/src/types/config.ts声明EditorImages/EditorImageRead等端口并导出；新类型暂与旧TFileHandler并存至T8原子迁移，不做旧协议适配器。
**Interfaces:** useEditorAssets(view|undefined)→EditorImages|undefined，完整签名见editor spec§1；ContentReadLease投影稳定快照；UploadSession负责恢复，包不访问RootStore。

- [ ] Step 1：两个节点同ID共享读取且各自dispose，旧句柄error不影响新view；同node换File、最多2并发/第3排队、排队前取消、10图界限由入口验证；retry保留同prepareID；source copy拒绝/目标拒绝/网络未知不自动重试；迟到id不回传给新节点。
- [ ] Step 2：`pnpm --filter web exec vitest run core/hooks/editor/use-editor-assets.test.tsx core/store/asset` RED→GREEN。T7先实现独立新端口及测试，不更换真实页面、不启用旧wrapper图片；普通错误翻译投影不包含签名URL。
- [ ] Step 3：切root/route/editor释放全部session/queue/lease，P6回归；类型/格式/零warning，提交 `feat(M5/P7b): 提供编辑器内容图片适配接口`。

## Task 8（T8）：编辑器包和全部 wrapper 迁移

**Files:** editor/src/types/{config,editor}.ts、extensions/{utility,custom-image/extension,custom-image/types}.ts(x)、custom-image/components/{node-view,block,uploader,upload-status}.tsx与toolbar；plugins/drop.ts、helpers/paste-asset.ts、constants/config.ts；删除plugins/file/与restorePublicImages无使用链。同步web Rich/Lite wrapper/types/use-editor-config替换及全部直接调用形状，既有editor测试夹具。
**Interfaces:** IEditorProps.images?:EditorImages；同Task移除TFileHandler及必填旧props，unsupported consumer不给images；T9/T10再逐宿主启用。

- [ ] Step 1：editor DOM/transaction测试：有效选择/拖入/粘贴、只读/无能力、纯PDF/混合PNG+PDF/目录/空MIME、伪造同文档已上传标记/非法src；删除节点后晚到upload/duplicate不插回；Undo/Redo已完成节点零新copy/restore/DELETE；期限变化不onChange。
- [ ] Step 2：`pnpm --filter @nerve/editor exec vitest run` 与 `pnpm --filter web exec vitest run core/components/editor core/components/issues/editor-capabilities.test.tsx core/components/issues/autosave.test.tsx`。RED后同步迁移包+wrapper，read句柄useSyncExternalStore稳定快照，写操作号+abort双屏障；删除自动mount复制重试、旧假百分比、alert及旧file listener。
- [ ] Step 3：GREEN；包扩展/清洗/稳定ID/Markdown/输入法原回归、所有wrapper类型；变异伪造标记免copy、onError无限循环、旧节点updateAttributes分别失败恢复。提交 `feat(M5/P7b): 替换编辑器图片协议并约束异步节点`。

## Task 9（T9）：工作项、评论与历史正文图片

**Files:** components/editor/rich-text/description-input/root.tsx；issues/issue-detail/main-content.tsx、peek-overview/issue-detail.tsx、comments相关表单/展示、core/description-versions消费者；autosave/editor-capabilities及评论测试。
**Interfaces:** 宿主已读取→useEditorAssets；原工作项自动保存及评论create/update，历史只给只读images。移除本组旧EFileAssetType/uploadCommentAsset/duplicateCommentAsset props与方法。

**跨卸载责任：** `use-confirmed-autosave.tsx`仅增加必要的受控输入衔接，状态所有者为T5 host，debounce/串行/确认算法不复制。失败revision、未保存正文及图片UUID在暂时owner缺席后可恢复；基准变化给冲突、拒绝revision不自动重试。传输中节点旧session/队列彻底结束，只保留输入与已知资产身份/未确认提示，新view重验后由用户明确继续。Step1加入失败正文、未保存标题、图片节点、并发新正文、读取失败后恢复及真实权限ABA清空的联合测试。

- [ ] Step 1：ready前不保存UUID，成功后真实description_html/comment_html只含UUID；换图/调序产生新内容、URL刷新不触发PATCH、只补ID保持原迁移；正文拒绝保留，历史已移除图片仍可解析；评论图片不进普通attachment_ids，已发表再次编辑按commentID范围。
- [ ] Step 2：`pnpm --filter web exec vitest run core/components/issues/autosave.test.tsx core/components/issues/editor-capabilities.test.tsx core/components/comments core/components/core/description-versions` RED→GREEN。保留原debounce取消/串行保存/dirty处理，不复用未知copyMarkdownToClipboard。
- [ ] Step 3：原工作项/评论完整受影响回归、类型/格式；提交 `feat(M5/P7b): 接通正文图片与只读历史`。

## Task 10（T10）：已有草稿附件、换项目与发布

**Files:** issues/issue-modal/{base,form,draft-form,draft-issue-layout,components/description-editor}.ts(x)、issues/workspace-draft/draft-issue-block.tsx；store/issue/workspace-draft/issue.store.ts只有必要宿主状态协作，不改通用权限或服务端事务；相应测试。
**Interfaces:** Draft真实id+project_id可null→ContentView(draft)；WorkspaceDraftIssues.updateIssue/moveIssue；新普通表单无images，明确保存草稿后重开。

- [ ] Step 1：新普通表单零asset请求/零暗建draft，已有无项目草稿能上传保存/重开；传输忙时切项目/发布需完成或明确取消；A→B→A旧read/complete/copy永失效。发布先保存完整payload再单次publish，成功移除旧draftview，失败保留并新建view重读。实际普通工作项/草稿副本入口分别最终Issue/Draft；明确提交才建一次target→按顺序dup→save，普通副本再publish；部分失败/取消保留同一目标和打开路径，重试不另建，unknown创建/发布先核列表。prepare回包丢失/failed duplicate/重新选文件/刷新后，通过P7a列表找到未知pending并显式丢弃才恢复发布，不把本地取消当服务端删除。
- [ ] Step 2：`pnpm --filter web exec vitest run core/components/issues/issue-modal core/components/issues/workspace-draft core/store/issue/workspace-draft core/store/asset`。RED→GREEN，所有服务器迁移由既有API；不拿项目角色禁止私有draft。
- [ ] Step 3：草稿原保存/全payload/项目清除/双发布/拒绝回归；变异切项目不退休旧view、前端publish后搬asset分别应失败；提交 `feat(M5/P7b): 接通草稿文件与原子发布页面`。

## Task 11（T11）：真实附件计数、排序和显示偏好

**Files:** web/packages/constants/src/issue/common.ts；issues/issue-layouts/properties/all-properties.tsx；filters/header/display-filters/{display-properties,order-by}.tsx；新spreadsheet/columns/attachment-column.tsx与列注册；现有偏好验证/测试、en/zh-CN。
**Interfaces:** 直接用生成Issue.attachment_count及已实现±attachment_count排序；显示属性沿现有link_count范式，不新建后端字段。

- [ ] Step 1：0/1/51、tie及ID反序、项目列表/看板/表格/成员跨项目视图、显示隐藏后刷新偏好；增删附件刷新后计数/排序位置正确，即使当前列表未加载那一页；错误/权限收窄不得沿用旧值；请求监测无每行GET。
- [ ] Step 2：`pnpm --filter web exec vitest run core/components/issues/issue-layouts core/store/issue` 并运行constants受影响测试，RED→GREEN；完整页/跨页排序由服务器，前端不只排当前已取样本。
- [ ] Step 3：原列表布局/拖拽/虚拟化/筛选及保存偏好回归，类型/翻译/格式；提交 `feat(M5/P7b): 展示真实附件数量与排序`。

## Task 12（T12）：删除最后旧文件链与 services 包

**Files:** 删除core/services/{file,file-upload}.service.ts、store/editor/asset.store.ts、hooks/store/use-editor-asset.ts和旧config链；清root.store、TCommentsOperations、types/enums/file、utils旧file URL；inbox未开放消费者移除死上传回调但不启用；URL规范化移到core/lib/api-request-url.ts及原测试；删web/packages/services、package/catalog/pnpm-lock引用，knip.jsonc/tools/keywords.json过期例外。
**Interfaces:** 旧APIService仅改导入位置，normalizeAPIRequestURL行为不变；所有其余文件调用者必须使用T1/T2/T7，未来领域没有images能力。原服务包独有file-type及其不再需要的锁依赖由pnpm移除，不手写伪锁文件。

- [ ] Step 1：重跑完整rg消费者清单，逐条标迁移/删除/未来领域无能力；原URL规范化测试在新位置运行。行为测试断言未开放inbox不发文件请求，已开放页面没有/api/assets/v2、multipart、bulk-status、旧restore；不只靠无字符串通过。
- [ ] Step 2：删除真实无消费者链并用仓库pnpm更新锁；`make knip`、`make lint-web`、`make test-web`。关键词旧例外删除，不推迟until；检查editor可选能力无假空返回，TFileHandler/EFileAssetType/旧service包命中0（文档历史除外）。
- [ ] Step 3：实际依赖/import全仓复核与修改文件零warning，登记plane-diff及M1/M2/M4 handoff处理结果但status仍open；提交 `refactor(M5/P7b): 删除旧文件调用链和服务包`。

## Task 13（T13）：真实页面与交错反例

**Files:** 新e2e/stories/files/{f8-issue-attachments-page,f9-comment-files-page,f10-draft-files-page,f11-editor-images-page,f12-image-history-page,f16-content-lifetime-page}.spec.ts，复用fixtures/file-story.ts及P6接收端观察；新playwright.m5-content.config.ts和package脚本/knip登记，必要纯断言helper。
**Interfaces:** API准备宿主/角色，真实页面File选择/粘贴/拖放→本地或既有S3 fixture；SQL只断言或明确故障注入，不直接造业务前置。

- [ ] Step 1：F8全部动作与51分页、下载SHA和count排序；F9两评论私有准备/失败再发/未知响应；F10 null项目/换项目/发布与并发双发布、原P5事务失败证据逐输入核对并新增页面拒绝；F11伪造源/无权源/目标拒绝、Undo/Redo和普通文件不吞；F12换图/调序/历史/只补ID，真实HTML/API/DB断言。
- F10另验证unknown prepare/failed duplicate后的恢复列表、明确丢弃/取消/另一标签页、普通副本最终Issue与草稿副本最终Draft，不用P7a API测试代替真实页面。
- 用真实RootStore与详情/peek挂载链验证：未发送评论、失败正文/评论编辑及图片节点→同宿主普通附件complete/DELETE→规范Issue被pending撤回→读取成功/失败/显式重试。输入和确认结果保留，旧view/URL/请求终止，已确认命令和未知写均无自动重发；fresh正文与编辑基准冲突不覆盖；换宿主/权限ABA/root退休清空，另一详情实例不串数据。至少一个有效变异移除host输入保留后，断言必须以实际内容丢失失败；不得只用资源和编辑器孤立测试证明联合行为。
- 同一联合链还须区分暂时读取故障与owner已确认删除/不可见（宿主读取401/403/404）：前者保留只读重试，后者退休输入状态，同ID重新可见也不复活；fresh owner只读时，未提交内容不得伪装成已保存正文。正文保存422仍保留原输入，不混淆读结论与写拒绝。
- [ ] Step 2：`make build build-s3-fixture` 后 `pnpm --dir e2e exec playwright test --config=playwright.m5-content.config.ts --workers=1`，Chromium/Firefox/WebKit覆盖真实选择、键盘/读屏/焦点和主要交错；local与S3各实际上传/显示/下载，接收端核raw字节/凭据。expires_at响应注入仅证明UI到期，不冒称服务端真实TTL改变。
- [ ] Step 3：至少10个有效变异：漏取消后complete、漏节点代次、项目可见即当issue权限、草稿ABA复活、未知POST自动再发、confirm后刷新重复命令、以loaded数代总数、伪剪贴板免copy、撤销发DELETE、到期无限再读。每项独立断言失败与恢复SHA，原失败报告全留；清本阶段确切拥有的临时资源，提交 `test(M5/P7b): 验证文件页面与宿主切换边界`。

## Task 14（T14）：七门禁与准确检查点 CI

- [ ] Step 1：逐项A1–A6、完整消费者清单、死成员扫描、设计/plane-diff/handoff当前事实与文档链接。标明P8未完成事项，原始证据输入hash稳定。
- [ ] Step 2：串行 `make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e`；真实exit、完整raw、cache/skip、前后输入hash、容器快照。不得用聚焦测试替代全门禁，不改既定预算掩盖失败。
- [ ] Step 3：中文检查点提交、system push，精确HEAD的server/web/e2e全绿。原生GitHub插件可取日志/重跑；W11按既有一次重跑规则，其他失败根因处理；资料P6回归不得失守。

## Task 15（T15）：一次整分支 deep-review 与修复

- [ ] Step 1：冻结正式main基线..候选HEAD，一次fresh reviewer读完整手写文件与调用链、Review Focus/所有局部Ruling、全部消费者与证据；三视角报告保存docs/reviews下一编号，生成物核源。
- [ ] Step 2：主控逐条裁定、局部根因修复及有区分度RED/GREEN/受影响回归；真实架构问题先提交上报，不用第二同范围审者替代修复判断；DeclinedToJudge必须具名解决。
- [ ] Step 3：最终本地七门禁证据与修改范围对应，中文提交/push准确HEAD CI全绿，A7闭环；阶段记录写明事实、限制和P8责任，不提前写main完成。

## Task 16（T16）：集成、归档与 P8 预检

- [ ] Step 1：连续授权下核main与worktree干净、最终准确green候选及祖先，根main用--no-ff，核双父/树一致，system push；不在主目录切开发分支。
- [ ] Step 2：准确main CI三项全绿后归档全部ignored原件（含失败、突变、审者、浏览器、准备记录），核hash/bytes/mode/membership/unknown0；origin/main..branch=0后普通清理远端/本地/worktree，归档再核。
- [ ] Step 3：下一P8 worktree回填P7完成证据；仅P8 spec/plan与一次builder预检，不实施。P8承接最终F1–F18、P3/P4维护项、warning新规则/死成员/默认部署和handoffs；M5完成勾选仍等closeout两轮main证据。

## 预检自查

- [x] spec覆盖逐Task，所有A1–A8都有实现和证据落点。
- [x] 类型与生产/消费名字一致，所有实际路径/后端授权/无项目草稿边界核过，无悬空前向接口。
- [x] 五Review Focus均有明确反例，RED不是缺模块或编译失败，未知结果不自动重发。
- [x] 无新增依赖/协议/权限观察器，旧链删除和未来领域不启用可同时满足；全部跨任务边界可内联实施。
- [x] 正式基于P7a最终main复核，builder问题按既有事实/局部裁定/需负责人决策分类闭环；预检报告、真实新API与P7a归档前置齐全。
