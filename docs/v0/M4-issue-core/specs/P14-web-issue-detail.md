# M4/P14 详情、创建弹窗、父子、关联与链接

日期：2026-10-31。状态：**已授权实施，逐任务推进中**。基线为 P13 合并 main `8bb4bbcb52a224ace4cb2f9d383bca9b97a022bf`；分支 `codex/m4-p14`，worktree `.codex/worktress/codex/m4-p14`。遵循 [M4 设计 §12/P14](../M4-design.md)、[推进方法](../../claude_to_gpt/02-method.md)和[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md)。负责人已授权从 T1 实施，阶段末 deep-review、局部修复复审后保留分支，不自动合并。

## 1. 目标

把已经接通的原生工作项数据链落实到详情页、浮层、创建/编辑弹窗、父子关系、关联、链接与删除的真实页面体验。完成 I1/I2/I8/I9/I10/I17/I20/I22 页面、S2 按身份详情，以及 C3 详情侧、C4/C5/C10/C11。浏览器核对包含 Chromium、Firefox、WebKit。

复用 P10b/P11 的 IssueRootStore、权限读取失效、确认写入、描述保存和宿主生命周期；P12 的 ExistingIssuesListModal、下拉框与命令反馈；P13 的归档只读及同宿主晚答规则。已经满足的合同补验收，不重复实现或重写服务端。

## 2. 交付范围与合同

### 2.1 文件地图与分段

缩写：`C=web/apps/web/core`，`P=web/packages`，`R=web/apps/web/app/(all)/[workspaceSlug]/(projects)`，`E=e2e`。下表是实现范围，新增目标在 plan 标明；文件实际变动按读者盘点收敛，不因列入地图就必须改。

| Task | 现有文件/目录 | 交付 |
|---|---|---|
| T1 | `R/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page.tsx`；`R/browse/[workItem]/{page,layout,header,work-item-header}.tsx`；`C/components/issues/detail-host.tsx` | 原生跳转回归，页头与正文共同身份 |
| T2 | `C/components/issues/{title-input.tsx,issue-detail/sidebar.tsx,issue-detail/label/select/label-select.tsx}`；`C/components/dropdowns/` | 详情字段反馈、属性键盘与 I2 属性 |
| T3 | `C/components/editor/{use-confirmed-autosave.tsx,rich-text/description-input/root.tsx}`；`C/components/issues/issue-detail/main-content.tsx` | 描述撤销/重做、迁移与 I2 描述 |
| T4 | `C/components/issues/issue-modal/{base,form,provider}.tsx`、`issue-modal/types.ts`；`components/{title-input,project-select,description-editor,default-properties,parent-tag,date-fields}.tsx`；现有 project/date dropdown 与 `power-k/config/creation/command.ts` | 创建/编辑字段、非用户迁移、I1；日期校验、默认状态省略、既有命令补 c 快捷键 |
| T5 | `C/components/issues/{confirm-issue-discard.tsx,issue-modal/draft-issue-layout.tsx,issue-modal/components/description-editor.tsx}` 与 T4 表单 | 嵌套关闭、名字、忙碌与焦点 |
| T6 | `C/components/issues/{parent-issues-list-modal.tsx,parent-select-root.tsx,issue-detail/parent-select.tsx,issue-modal/provider.tsx,issue-modal/context/issue-modal-context.tsx}`；T4 父项消费者；`issue-detail-widgets/sub-issues/`；`C/components/inbox/modals/create-modal/issue-properties.tsx` | 原生父项搜索、子项与 I8；M7 调用只适配类型 |
| T7 | `C/components/issues/issue-detail-widgets/relations/`；`C/components/core/modals/existing-issues-list-modal.tsx`；`C/components/inbox/modals/select-duplicate.tsx`；`C/services/project/project.service.ts` | 关联与 I9；收集箱查重只改读取 |
| T8 | `C/components/issues/issue-detail/links/`、`issue-detail-widgets/links/`；`C/store/issue/issue-details/link.store.ts` | 链接表单、元数据回读、I10 |
| T9 | `C/components/issues/delete-issue-modal.tsx` 及现有调用 | 子树删除说明与 I17 |
| T10 | `C/components/editor/{rich-text,lite-text}/`；`C/components/power-k/projects-app-provider.tsx`；`P/editor/src/extensions/emoji/emoji.ts`；`E/fixtures/{browser,settings-pages,issue-list-page}.ts` 与警告读者 | 编辑器按需加载、个人设置不挂载工作项弹窗、警告根因清理 |
| T11 | `P/editor/src/extensions/{callout,custom-image,emoji,mentions}/`、`custom-color.ts`；`P/editor/src/sanitization-roundtrip.test.ts`；`api/testdata/issue-richtext.json` | 原生标注表情、删除五处死序列化、打包 CSP 复核 |
| T12 | T3/T4 的 M4 编辑器入口、浮层/详情与现有图片扩展配置 | M4 图片入口关闭；附件和关注按钮不挂载 |
| T13 | `C/components/issues/issue-layouts/spreadsheet/columns/sub-issue-column.tsx`、`properties/all-properties.tsx` | 表格已完成的 browse 回归，清理另一子项计数入口的无效片段 |
| T14 | `C/components/modals/work-item-level.tsx`、详情/浮层/路由；`E/stories/smoke/s2-web-app.spec.ts` | I20/I22、S2、Back 与权限失效 |
| T15 | `C/components/issues/issue-detail/{identifier-text,issue-detail-quick-actions}.tsx`、`links/link-item.tsx`；`issue-detail-widgets/{relations/helper.tsx,sub-issues/helper.ts}`；`peek-overview/header.tsx`；`tools/keywords.json` | 六处复制真实反馈及定点例外删除 |
| T16 | 本段故事、三浏览器配置、review、handoffs | 完整门禁、有效变异、dead/oxlint、deep-review 与复审 |

### 2.2 地址、身份与已有基础

项目 id 入口的真实路径是 `/[workspaceSlug]/projects/[projectId]/issues/[issueId]`，不是不存在的工作区平级 issues 路由。P11 已删除此入口的 clientLoader、模块级 IssueService 和旧收集箱跳转，并经 useSessionSWR 读取原生 IssueCore 后 replace 到 browse。P11 的 browse 已按编号读取，缓存键带 workspace/原编号/权限身份，不在前端大小写猜测。T1 验证这些事实，并给包含页头命令和正文的路由子树统一 key；加载期间不可复用上一项的页头、弹窗或正文。

key/owner 继承 workspace、解析后的 project/issue 与现有权限身份；路由 A→B→A、同 URL 关闭重开、权限降再升结束旧宿主。所有异步跟进沿 useDetailScope/followInSession 和既有命令所有者。不能新增全局弹窗协调器或第二份权限缓存。

详情/浮层显示原生 Issue/IssueDetail；归档、访客只读保持 P13 规则。有权删除/恢复的命令单独判断，不能把字段只读扩大为禁止这些动作。新建、修改、删除都沿既有确认写入；确认前、当前拒绝、确认后读取失败、旧宿主晚答分别处理。读取失败重试只 GET，不能重复 POST/PATCH/DELETE。

### 2.3 标题、属性与表单反馈

详情标题、创建/编辑标题与项目选择、链接 URL 的错误用 propel Field 内的 FieldError，控件带 aria-invalid 且由提示描述。复用 useFormRefusal/错误码文案和 useRefusedFocus(attempt)，RHF 的 shouldFocusError=false；不能 setError 同步抢焦点。详情自动保存的本地校验/服务端拒绝也按一次拒绝尝试推进 attempt，正常输入不搬焦点。空白标题、超长标题、倒置日期及项目必选分别证明。

状态、优先级、负责人、标签、起止日期、父项使用已有生成字段；空值/清除和日期原子更新保持 P11。详情 label-select 去 baseTabIndex，M4 表单去所有正 tabIndex，Controller ref 保留。已有共享 dropdown 的 modal=false 按 useInsideModal 使用，不能全局禁用模态或另写弹层库。菜单/选项内部不放键盘到不了的按钮，不嵌套按钮。所有新/修改的产品文字同步 en/zh-CN。

### 2.4 描述与弹窗迁移

详情已有 useConfirmedAutosave 的 1.5 秒窗口、prepareUpdate 捕获和 boolean skip_activity；保留它们。真实编辑器在窗口内撤销再重做，最后保存必须等于界面内容；相等于确认值时也要撤销过时的待发保存。普通编辑被拒绝保留输入与重试；切实体/会话结束后旧确认不回写当前编辑器。不发送 description_json 或字符串 skip_activity。

弹窗 description-editor 的 onChange 第三个参数必须读取 isMigrationUpdate。**编辑已有普通工作项、打开已有草稿**两种分别测试：补块 id 可以更新待提交 HTML，但不得单独标记用户 dirty、触发保存提示或制造一次业务写；有真实编辑时保留 dirty 和最新 HTML，不让后到迁移清掉用户修改。新建空编辑器不能替代这两种测试。若用户随后显式保存迁移后的 HTML，沿对应原生更新语义处理；已有工作项只在纯迁移时允许 boolean skip_activity，草稿不扩展不存在的字段。

### 2.5 弹窗生命周期

创建/编辑、Save this draft、父项搜索、链接弹窗都有 ModalTitle（需要时 sr-only），名字经 t()。Tab 留在最内层、背景 inert/aria-hidden，portal 日期/选择器/表情可操作；Escape 和外点关闭当前可关闭层，焦点回到仍连接的触发器。

只保留一个表单关闭决策：同步 busy 检查、编辑器 readiness、已有 draft-discard 逻辑。外层 ModalCore、Discard 按钮和编辑器关闭入口复用它；描述编辑器不再全 document 处理别的 portal Escape，移除冗余监听或限在自己的 Dialog 内，两者不能同时触发关闭。内层 Escape 只关闭内层，不关闭创建弹窗、不额外弹 Save this draft。请求在路上不能关闭/二次提交，失败留窗，成功跟进仅限原 owner。

执行顺序：T5 完成关闭策略/真实 ModalCore 嵌套单测及 Save this draft、日期/标签的页面验收；父项搜索尚未原生接通，父项 C4 页面验收随 T6 迁移一起闭合，不在 T5 放行旧 API。

本段只验证 Save this draft 嵌套与迁移兼容；保存/发布草稿完整 I14/C9 留 P15，不扩展草稿协议或重写既有生命周期。

### 2.6 父子与关联

ParentIssuesListModal 直接收/回传生成 Issue；value 为 Issue|null，onChange 为 (issue:Issue)=>void|Promise<void>。context/provider/form/default-properties/parent-tag、详情选择器与收集箱创建调用同 Task 迁移，不经过 convertWorkItemDataToSearchResponse 还原旧类型。父项展示的项目标识/状态颜色取现有目录；无目录时中性显示，不编造字段。关闭/切项目清空当前选择证明，旧结果不可提交。

已有工作项选父项用 `IssueSearchQuery {project_id, for:"parent", issue_id, q?}`；新建尚无 issue_id 时用 `{project_id,q?}` 普通可见搜索，因为尚无自身/后代可排除，不能发送缺 issue_id 的 for 或伪 id。通过现有 useIssueSearch 消费权限范围与搜索证明；前端不重写服务端层级排除。移除 hover-only 链接和正 tabIndex，输入有名字，错误/空/加载/只读重试明确，非选项按钮在 Options 外。

子项添加已有沿 ExistingIssuesListModal 的 for=sub_issue；关联沿 for=relation，并保留跨项目切换和 SelectionSubmitResult 的逐项 confirmedIds/failures。服务器确认前不伪造行/计数，部分成功后不重发已确认项；当前拒绝仍可重试，确认后补读失败不回滚。I8 准备两层以上与两个已有子项、一个新建子项，移动/解除父项及状态分布都可见。I9 验证同项目 blocks、跨项目 relates_to、对端 blocked_by 及任一有权端删除；隐藏端不能泄露。

M7 的 select-duplicate 仅换为原生 useIssueSearch/Issue 和读取错误/currentness；其 onSubmit(id):void 及收集箱业务写不改。收集箱创建的父项选择只同步类型/展示，M7 自身页面零失败请求与业务验收仍归 M7。最后真实读者消失时删除 projectIssuesSearch 方法及仅它使用的类型/例外，不为等 P15 保留死方法；准备盘点确认 ProjectService 只有该方法且只有上述两个生产调用；T7 同步删服务文件、barrel 导出及专用旧搜索类型，P15 只验零残留，不重复记删除。若实施时出现新的真实读者，先按其职责核对而非保留空类。

### 2.7 链接与元数据

URL 去首尾空白；无 http(s) scheme 时按既有服务端规则归一化，不扩展 scheme。字段拒绝可读且正确聚焦；409 issue.link_exists 留窗且只报一次。pending 同步锁住提交/关闭，确认后才关闭；GET 失败不伪称保存失败。

链接行展示用户 title（无则 URL）和独立 metadata.title。图标只消费服务端 metadata.favicon 的 data image；favicon_url 是源地址记录，**不得用它发浏览器外部请求**。无图标/blocked/failed 保留链接和正常 fallback，不能把元数据失败当链接保存失败。

现有 store 在写后只回读一次，可能先于 worker。在详情链接资源的 UI owner 内加入有期限的元数据回读，仍调用 link.fetchLinks；不建第二缓存、不新增 API/worker。以当前 link id+URL 为待补全身份，空 metadata 才待补；有字段的成功结果或 error 为终态。单 owner 至多一条在途 GET，首次立即核当前数据，之后每 1 秒最多 30 次/30 秒，离页/换项/权限变化/链接删除/URL 替换立即结束旧循环，迟到结果仍由现有资源 token 拒绝。当前 GET 拒绝转现有读取错误并停止自动循环；用户 Retry 只 GET 并可开启新窗口。到期保留地址和一次明确的读取重试入口，不伪报网络失败或自动重复写。可使用新局部 hook，不把此 UI 调度迁入全局 store 的常驻轮询。

I10 使用已有本机 HTTP fixture，控制返回时机证明“先地址，后元数据”，DB 和 river_job 证明真实抓取。元数据刷新不是固定 sleep 的测试同步，测试按 UI/DB 条件在期限内等待。

### 2.8 删除与地址

DeleteIssueModal 保持必需异步 onSubmit 和同步 pending；文案明确会删除该工作项及其全部子工作项（含归档后代），不新增无权枚举子树数量的请求。拒绝留窗，确认后才提示/导航。I17 通过 API 准备两层后代、归档后代、跨项目关系及相关记录，页面删除后复用 expectIssueDeleted/目录断言核整棵子树和同一时刻。

表格 sub-issue-column 在 P12（ceb154a9）已直接用可见项目 identifier 和 sequence_id 生成 `/browse/...`，无 `#sub-issues`，不用普通/归档旧中转路由；目录未就绪不拼 undefined 地址。T13 对此补消费者回归，不假称新实现；当前 properties/all-properties.tsx 的子项计数入口仍附加 #sub-issues，同 Task 去掉无读者片段并使用原生 browse，保留行内键盘/事件传播语义。

### 2.9 编辑器载入、死代码与边界

RichTextEditor/LiteTextEditor 的公开 props/ref 保持兼容；实现通过动态 import 和 Suspense 按实际挂载加载，类型导出不引回实现，避免 barrel 造成 eager runtime import。当前 lite-text/index.ts 还静态导出 toolbar，而 toolbar 经 @nerve/editor 读取 TOOLBAR_ITEMS；它必须一同进入延迟边界，不能只 lazy 两个 editor.tsx。个人设置的 ProjectsAppPowerKProvider 不挂载 WorkItemLevelModals；保留个人设置合法命令。未登录直达 browse/旧 id 入口不能加载 TipTap/editor 实现 chunk；登录进入可编辑详情后能正常加载、使用、保存。

emoji.ts 当前 supportMap/isSupported 没有读者，renderFallbackImage 已固定 false。删除无读取方的 Canvas 支持检测和死 fallback；is-emoji-supported 若全仓无其他直接读者，同步删 package/catalog/lock 中相应依赖项，不能留下无消费者依赖或屏蔽 console。删除 EMOJI_CHECK_WARNING 定义、当前全部引用及过时注释；33 个命中文件是准备快照，不把历史“29故事/96行”当验收门。expectQuietConsole 保持严格，不能过滤或改 watchPage 让警告静默。

标注块默认 unicode 为 128161，URL 不再指向 jsdelivr；保存/读取已有合法本地原生 emoji/icon，默认与存储 fallback 同步。五处扩展 markdown.serialize 在生产无 getMarkdown 调用、transformCopiedText=false；删除 custom-color/custom-image/emoji/mentions/callout 的死序列化及专用类型/转义 helper，保留 HTML parse/render、Markdown **粘贴解析**、编辑器 ref 的 convertHTMLToMarkdown。P4 曾保留并修复序列化，这次删除按既定 P14 设计收口：共享 issue-richtext fixture 只删不再消费的 expected_markdown/markdown_element，保留所有恶意属性、HTML 清洗、真实编辑器往返断言，不能整组删安全用例。打包后查 worker 创建/URL，并在真实页面观察 CSP；没有 blob worker 就不扩 worker-src，更不为 jsdelivr 放宽 CSP。

M4 详情/浮层、创建/编辑/草稿弹窗的描述编辑器明确 disabledExtensions 包含 image；CoreEditorExtensions 已同时控制 image 和 imageComponent。测试 slash、粘贴、拖拽无上传请求、无附件/关注按钮挂载。不要把关闭图片变成所有 M6/M7 编辑器全局默认；M5 接图片重新开启及身份清洗规则。本期不移除服务端已批准的 HTML 图片清洗能力。

### 2.10 页面、会话与复制

I20 按 API 准备“先成员创建，再降为访客”的 OwnOnly 数据；页面只列自己的、别人地址与不存在相同、无创建/修改入口。管理员兼访客按 Grant/OwnOnly，不能简化为角色等号。S2 公开项目分别验证有权成员可见、OwnOnly 访客隐藏项 404、工作区成员但非项目成员 403，watchPage 仅允许对应那一个拒绝。

I22 的延迟响应必须 route.fetch() 先到达真实 nerve、DB 已确认后，在另一个标签页退出/登录不同账号，再 fulfill 旧响应：无旧 toast、导航、行/详情投影。Back 跨工作区/项目/工作项结束详情、peek、创建/删除弹窗和 WorkItemLevelModals；同地址重开不受旧 finally 影响。

T15 六处复制使用现有 useCopyLink/useCopyText（均收值与成功文案两个参数，返回 Promise<void>，hook 内负责成功/拒绝提示）；消费者不再另报成功，helper 的 copyLink 同步改 Promise<void> 并更新调用。await 真实剪贴板 Promise 后成功，拒绝经 t() 提示，无未处理拒绝/假成功。链接按当前 origin（含局域网 HTTP）生成；编号只复制文本。增加真实消费者按钮/菜单的拒绝测试，不能只测试 hook。仅删除这六处 copy-hook 例外；创建 toast、时间线、布局快捷 helper、命令面板的剩余例外归 P15。

## 3. 准备期决定

| 编号 | 结论、理由 | 代价/边界 |
|---|---|---|
| D1 | 保留设计 16 Task；T1/T3 对 P11 已有能力只做真实消费者回归 | 不重复重写缓存/自动保存；T13 表格链接也已由 P12 完成，补验并修实际剩余入口 |
| D2 | 无源 id 的新建父项搜索不用 for；已有项用 for=parent | 服务端要求 for 与 issue_id 配对；新建没有后代，不改 API |
| D3 | 父项直接用 Issue，同步 M7 共用调用类型 | 多文件适配属设计明确范围；M7 业务验收不扩大 |
| D4 | 链接元数据有期限地沿既有资源回读 | 最多 30 次/窗口；慢任务可手动读重试，不保证无限等待 |
| D5 | 删除死 emoji 检测及五处序列化，保留 HTML 安全测试 | 不再提供无人使用的定制 Markdown 序列化；Markdown 粘贴不变 |
| D6 | 延迟加载保留 wrappers/ref，图片只在 M4 入口禁用 | M6/M7 不被全局默认改变；C10 用真实 build Network 证明 |
| D7 | 创建表单只有一个关闭决策并复用既有 ModalCore | 不新增全局状态/协调器；完整草稿业务仍 P15 |
| D8 | 只关闭交接对应子项，顶层保持 open | M4 总关闭、总体设计 9.4 和 M5–M8 closeout 留真正收尾 |

以上是既定设计内的局部准备选择。若实施发现权限所有权、共享协议或跨模块架构必须改变，先提交已完成工作再上报。

## 4. 验收标准（实施前全部未勾选）

- [ ] A1：原生 id/编号入口正确，页头和正文共同身份；同参数换项、权限 ABA、Back 无旧弹层/结果（T1/T14）。
- [ ] A2：详情标题/七类属性、字段错误描述与拒绝后焦点符合 §2.3；I2 属性完成（T2）。
- [ ] A3：真实编辑器撤销重做与迁移保存正确；boolean skip_activity、不发 JSON；已有 issue 与 draft 的迁移不标 dirty（T3/T4）。
- [ ] A4：I1 创建字段/归一化/码/数据库正确，表单有名字、无正 tabIndex、项目必选可读；拒绝与 pending 保留当前窗（T4）。
- [ ] A5：父项/Save this draft 内层 Escape 不影响外层，portal 背景隔离、循环 Tab、焦点回还（T5/T6/T16）。
- [ ] A6：父项原生查询与所有消费者一致，两个已有/一个新子项、移动/解除/状态分布、排除后代，I8 完成（T6）。
- [ ] A7：跨项目关系/反向/任一端删除/部分确认保留，I9 完成；M7 两个消费者适配且无旧 projectIssuesSearch 读者（T6/T7）。
- [ ] A8：I10 真 worker 元数据、bounded GET 重试、data favicon、无外连，字段拒绝与复制分别验证（T8/T15）。
- [ ] A9：I17 删除整棵子树且文案明确、onSubmit 必需、拒绝留窗/成功跟进；数据库目录断言通过（T9）。
- [ ] A10：未登录无编辑器 chunk、设置不挂载工作项弹窗，EMOJI_CHECK_WARNING 全零且 console 无豁免（T10/T16）。
- [ ] A11：原生 callout、五处死序列化删除而安全/粘贴回归保留，打包 worker/CSP 复核通过（T11）。
- [ ] A12：M4 无图片上传入口/请求，无附件及关注按钮；表格子项地址直接 browse 无片段（T12/T13）。
- [ ] A13：I20、S2 身份、I22 两标签页及 Back 通过；旧会话响应无跟进（T14）。
- [ ] A14：六处真实复制消费者成功/拒绝正确，六个 copy-hook 例外删除，P15 剩余例外具名（T15）。
- [ ] A15：C3 详情/C4/C5/C10/C11 三浏览器均实际非零通过；完整七项门禁、至少 16 种有效生产性质变异及恢复证据、实际改动文件 dead/oxlint 核清（T16）。
- [ ] A16：基线至最终分支 HEAD deep-review、局部问题修复复审闭环、准确分支 CI 全绿及证据归档位置明确；先交负责人安排，不自动合并（T16）。

## 5. 后续范围

P15：时间线、评论、表情、描述历史、草稿完整流、搜索/命令面板、剩余复制和旧 service/类型最终清理；P16/收尾：全局 dead/no-shadow/守卫终态、C12、设计 §14 逐条证据与总体设计 9.4、全部 M4 输入关闭和 M5–M8 handoffs/M4-closeout。M5 重新开启图片/文件；M6/M7 自身接口与业务页仍按 P10a 已裁定 A 验收归属。P14 不提前宣称这些完成。

## 6. 风险与验证

重点不是“能编译”，而是 editor/portal 的真实焦点、确认与读取失败的边界、身份 ABA、跨项目搜索及元数据晚答。单元使用既有 FakeNerve、heldChange、真实组件与资源；E2E 前置全部经 API、数据库只断言。禁止固定 sleep 同步、空匹配算成功、隐藏 console 或扩大失败白名单。测试与运行台账写明确 cwd/argv/输入 SHA 或 dirty 哈希、实际 executed/skip、raw/退出码、变异恢复证明。

有产品变化的检查点按 plan 完整本地 F 后推分支。W11 已知偶发不专门排查；若阻塞合并且重跑仍失败按负责人要求停下。浏览器安装/环境超时与性质红分开记录。main 集成及准确 main CI 只在后续明确授权后进行。

## 7. 交接输入与关闭证据落点

| 输入 | P14 接收部分 | 证据落点/保留 |
|---|---|---|
| M1-P2-trim-content | 描述撤销再重做、boolean 迁移页面 | T3/I2；服务端 P4 原证据保留 |
| M1-P4-router-native | 设置不挂载、子项列 browse 无片段 | T10/T13/C10 |
| M1-closeout | issue/draft isMigrationUpdate、五处死序列化 | T4/T11；P4 历史修复与本期删除分别记 |
| M2-closeout §7/8/11/12 | 延迟编辑器、callout/worker、详情复制、三处旧实例 | T1/T6–T7/T10–T11/T15；跳转实例 P11 已删，仅回归 |
| M3-closeout §6/7/8 | key/会话、详情表单/弹窗/嵌套/键盘、删除说明 | T1–T9/T14–T16；评论等仍 P15 |
| P10b/P11/P12/P13 review §6 | root 统一失效、确认写、原生搜索、归档只读 | 各 Task 与 I20/I22；不另设所有者 |

在实施 review 对应 A 条款写结果，handoff 顶层状态仍 open；M4 真正收尾才按原关闭条件统一关闭。

## 附录 A：准备证据

本次只做源码/合同/测试静态预检，替代原型和逐 Task 产品重放，未安装依赖、运行产品测试或生成新产物。准备原件在主项目 `.superpowers/sdd/P14-preparation-20261031/`，实施原件将位于 worktree `.superpowers/sdd/P14-web-issue-detail/`；清理时两处都须归档。

来源与审查方式、独立 builder-check、静态验收见 [P14 review](../reviews/P14-web-issue-detail-review.md) 与 [预检报告](../../../reviews/re-20-M4P14预检审核.md)。预检通过不等于上面产品 A 条款通过。
