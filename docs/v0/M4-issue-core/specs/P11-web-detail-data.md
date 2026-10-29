# M4/P11 详情数据层规格

> 状态：负责人已裁定P11-S1采用A，**文档预检通过，已获实施授权，T1–T5已实现，继续T6**。分支 `codex/m4-p11`，基线 `df5d3bbc1887486bd105fe6fc663f57f5b371992`。P10b 已合并且准确 main CI 全绿、证据归档和清理完成，见 [P10b review §17](../reviews/P10b-web-list-data-review.md#17-授权合并main-ci与归档清理2026-10-30)。准备阶段已结束；当前按计划实施，产品验收全部开放；编号列宽按已加载、当前可见数据计算，不新增项目API字段，见[已裁定记录](../reviews/P11-project-number-width-decision.md)。

权威输入：[M4 设计](../M4-design.md) §3.9–3.15、§7.1–7.4、§9.5、§12 P11、§13；[推进方法](../../claude_to_gpt/02-method.md)、[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md)；[P10b spec](P10b-web-list-data.md) §2.6/§7；输入 handoffs 的 M1-closeout、M2-closeout §12、M3-closeout §5/6/8。配套 [plan](../plans/P11-web-detail-data.md)、[阶段预检](../reviews/P11-web-detail-data-review.md)。

## 1. 目标

详情、子项、关联、链接、评论、两类表情、动态、描述版本和个人草稿使用生成客户端；同一会话的工作项仍只存于 P10b 的 canonical。详情资源复用 IssueRootStore 的权限读取失效，消除旧地址、模块实例、乐观回滚和未等待的拒绝。补齐按需目录读取、标签创建和弹窗宿主的责任。

这是数据层接通及其调用方的编译/基本行为闭合。P12–P15 的完整页面故事、键盘/焦点、后退与可访问性验收保持原阶段；M5–M7 自身接口和领域页面不在本段验收。不得把当前可编译的旧服务视为已接通，也不得用生成类型标注仍来自 Plane 的应答。

## 2. 交付物

### 2.1 目录、任务及边界

下文 `C/` 为 `web/apps/web/core/`，`P/` 为 `web/packages/`；标“新”的路径仅为实施目标。按资源整组迁移类型、服务、store、直接调用方和测试，每个Task末可编译。原十四个Task在T4后按规模预留拆为十五个：T5评论及其内嵌表情、T6工作项表情，原T6–T14顺延为T7–T15。合同与阶段范围不变，历史证据保留当时编号。

| Task | 主要文件 / 职责 | 目标测试（“新”在实施时创建） |
|---|---|---|
| T1 | C/services/issue/issue-api.service.ts 增按编号读；新 issue-detail.service.ts 提供子项/关联/链接；其它薄服务随所属资源 Task 迁移；tools/keywords.json phase | 既有 issue-api.service.test.ts；新 issue-detail.service.test.ts |
| T2 | C/store/issue/root.store.ts、permission-reads.ts、issue-details/issue.store.ts；新 issue-details/resource.ts；归档页头及 issueId 跳转的最小数据接线 | 既有 issue.fetch.test.ts；新 resource.test.ts、detail-entry.test.tsx |
| T3 | issue-details/sub_issues.store.ts、relation.store.ts、root.store.ts；父子/关联控件与旧类型 | 既有 sub_issues.fetch.test.ts；新 relation.store.test.ts、detail-changes.test.ts |
| T4 | issue-details/link.store.ts；links 控件/弹窗；删除旧 IssueService 最后方法及文件 | 新 link.store.test.ts、links/data.test.tsx |
| T5 | issue_comment.service.ts；comment/comment_reaction stores；comments控件、对应旧类型 | 新collaboration.service.test.ts、comment.store.test.ts、comments/data.test.tsx |
| T6 | issue_reaction.service.ts、reaction.store.ts；工作项表情实际控件、最后旧表情类型 | 扩collaboration.service.test.ts；新reaction.store.test.ts及UI测试 |
| T7 | issue_activity.service.ts、activity.store.ts；新 timeline.ts；issue-activity 展示，删旧动态细节类型 | 新 timeline.test.ts、activity.store.test.ts；T5 评论分页回归 |
| T8 | work_item_version.service.ts；新 issue-details/description-versions.store.ts；core/description-versions 与两个 M4 调用方；M7 版本 DTO 归属 | 新 description-versions.store.test.ts、core/description-versions/root.test.tsx |
| T9 | workspace_draft.service.ts、workspace-draft/issue.store.ts；draft 页面/表单直接消费者、root 创建确认入口 | 新 workspace-draft/issue.store.test.ts、issue-modal/draft.test.tsx |
| T10 | hooks/editor/use-editor-mention.tsx、editor/lite-text、rich-text/editor与description-input、issue-modal/components/description-editor、inbox/modals/create-modal/issue-description；WorkspaceService.searchEntity 及旧 JSON 类型 | 新 hooks/editor/use-editor-mention.test.tsx；既有 editor 包测试 |
| T11 | 详情及按需属性目录读取、详情一侧 SessionSWR；同 Task 删除 .oxlintrc.json 对应旧 SWR 豁免 | 新 detail-refusals.test.tsx；既有 use-session-swr、issue.fetch/sub_issues.fetch 测试 |
| T12 | modals/work-item-level.tsx、issue-details/root.store.ts、详情/peek/列表实际弹窗宿主 | 新 modals/work-item-level.test.tsx、issues/detail-host.test.tsx |
| T13 | issue-modal/base.tsx、五处标签创建调用；use-label-changes；工作区标签边界/列表统一编号列宽 | 既有 base.loading/base.clone 测试；新 issue-properties.test.tsx |
| T14 | 关键词、barrel、死成员、修改矩阵、跨资源竞态、S2 请求边界 | 新 detail-mutations.test.ts；既有列表权限/确认回归，S2 |
| T15 | 设计、frontend-changes、各输入 handoff 的局部处理结果、验收、阶段整分支 deep-review 和修复 | 最终完整 F、准确分支 CI、原始审查与修复证据 |

### 2.2 生成合同与薄服务

真实合同为 `api/modules/issue.yaml`，类型来自 `@nerve/api-client` 的 `components`/已导出类型及 `operations`。构造器接本代 `ApiClient`；每个 service 方法只调用一次生成 GET/POST/PATCH/DELETE 并 unwrap。service 不排序、不改形状、不写缓存，不吞 Problem 或 SessionChangedError。查询类型从相应 operation 提取；cursor 原样传递，不解码、不自造。

| 资源 | 操作与原生应答 | 关键约束 |
|---|---|---|
| 核心 | getIssue、getIssueByIdentifier → IssueDetail | 后者路径 `/api/v0/workspaces/{slug}/issues/{identifier}`；按 id 用既有 IssueApiService.get |
| 子项 | listSubIssues/addSubIssues → IssueCollection | POST `{sub_issue_ids}`；应答只有 data，无 state_distribution；1–100，同项目 |
| 关联 | listIssueRelations/createIssueRelations → IssueRelationCollection；deleteIssueRelation → 204 | 创建 `{relation_type,issue_ids}`；删除 `/api/v0/issue-relations/{relation_id}`，不是对方工作项 ID |
| 链接 | listIssueLinks → IssueLinkCollection；create/update → IssueLink；delete → 204 | update/delete `/api/v0/issue-links/{link_id}`；title/metadata nullable；这不是 M5 文件 |
| 评论 | listIssueComments → IssueCommentPage；create/update → IssueComment；delete → 204 | comment_html，actor_id、edited_at 和内嵌 reactions；无 comment_json/actor_detail |
| 工作项表情 | listIssueReactions → IssueReactionCollection；add → IssueReaction；remove → 204 | actor_id；按 reaction 删除调用者自己的记录，不能按 UI 传来的另一 userId 清别人 |
| 评论表情 | addCommentReaction → CommentReaction；remove → 204 | **没有 GET 评论表情集合接口**；读取来自 IssueComment.reactions |
| 动态 | listIssueActivities → IssueActivityPage | actor_id 可空；field/old_value/new_value/identifiers；无 source_data、epoch、is_bot |
| 描述版本 | listDescriptionVersions → DescriptionVersionPage；getDescriptionVersion → DescriptionVersionDetail | owned_by_id；没有专用 restore endpoint；只允许管理员/成员读 |
| 草稿 | listDrafts → DraftPage；create/get/update → Draft；delete → 204；publish → IssueDetail | Draft.payload 是 IssueDraftPayload；PATCH 是整份 payload 替换；publish **不带 body** |

具体接口目标：IssueApiService 增 `getByIdentifier(slug, identifier): Promise<IssueDetail>`；新 IssueDetailService 的 `listSubIssues/addSubIssues/listRelations/createRelations/deleteRelation/listLinks/createLink/updateLink/deleteLink` 按表中的生成请求/响应类型。旧 collaboration/version/draft 服务原地换构造和方法，所有调用方在对应 Task 同步改，不维持旧参数/结果适配层。每个公开方法均有真实生成客户端请求/拒绝测试。IssueSearchService 在 P12 首个生产使用方接通时创建，理由和边界见§3 D2。

### 2.3 一份工作项与权限范围读取

P10b 已交付 `IssueRootStore.captureRead/isReadCurrent`、`PermissionReads`、`issues.acceptRead`、`verification.refresh`、per-id 写队列和列表确认协调。P11 **沿用已批准的方案 A**：权限事实仍只有 root 的一处观察，详情资源不得另订阅一套角色或复制项目/工作项实体。

新局部 resource 容器基于现有 Reconciled，每份值带 root 的读取令牌与 workspace/project scope；root 在权限改变、成功删除和退役时使相关读取失效。读取值时再次核令牌，失效值立即不可见；当前代的普通读取失败保留原值并公开错误，不能等下次成功 GET 才遮住失权值。未知后代删除沿 P10b 项目读取代际处理。注册有 dispose；root.retire 幂等释放；引用者不得通过裸 map 读绕开判定。

`fetch(read)`、`confirm(change)`、`invalidate()` 复用 Reconciled 的最新轮次与确认重放；collection 的历史 GET 晚到不能撤销确认的新增/编辑/删除。跨项目关联资源保存涉及项目的权限证明（包括实际内嵌项目），失效只处理受影响范围。权限降级再恢复也不能使旧响应或 SWR 缓存复活；SessionSWR key 除 loginId、业务参数，还带 root 的 permission identity。

按 id 读取先校验已知 scope；应答的 id/project/workspace 必须与请求匹配才可进 canonical。按编号入口只把完整编号交给服务端解析，不复制服务端大小写/Unicode 规则，也不用本地 identifier 严格相等抢先报错。先确认当前 workspace 并捕获 root token，GET 后确认该 workspace、读取仍有效，再通过同代项目目录取得返回 project 的当前权限，之后才 acceptRead/登记别名及启动子资源。目录请求被拒绝则原错交回；不可见项目不写任何实体。编号未知项目不能取消后置权限证明；已知明确无权时不预取其附属资源。其它按 id/项目入口保留原有 preflight，不能借编号入口放宽它们。

核心 Detail 只有 acceptRead/既有写协调入口可落地；描述只由真实 Detail 写入，保留纳秒版本与 verification freshness floor。不得手改 updated_at、计数或复活已删除实体。详情装配的父项、links、reactions、activities、comments、sub-issues、relations、必要目录均有 await 或明确的由 UI 持有的受控请求；每步在下一请求前核当前 scope。核心成功、附属失败时，保留已确认核心并让调用方显示错误/重试，不伪造空集合成功。

### 2.4 修改串行与确认

核心属性/描述/父项仍调用 root.updateIssue 等既有入口。子项批量、关联、链接、评论和表情操作通过 root 的同一工作项队列；草稿另按 draftId 串行。批量影响多个工作项时对去重后排序的 ID 进入同一组队列，队列轮到发出时再核会话/权限；不能在占有该队列的回调里再调用会取得同队列的 updateIssue，避免自等。

只消费服务端成功答案；拒绝期间不预改 canonical、计数、关系/评论集合。子项、关联应答中的 Issue 经 canonical 接收，资源本身只存 ID/原生资源 DTO；不得维护第二套完整 Issue。涉及父项计数、link_count 或其它列表投影变化时，通过 root 既有确认/权威回查路径刷新相关 Issue，再向全部持有列表传播；未知旧父或第三方版本按 P10b 标 stale/补取，不能手加减猜计数。

POST/PATCH/DELETE 成功后补读失败，与修改本身被拒绝分开：保留成功资源确认，canonical 标待核/列表 stale，报告补读错误并提供读重试；不要回滚服务端成功，也不要重发写入。退役或权限失效后不写本地/不 toast/不导航，不自动重试命令。每个异步 UI 跟进同时核 followInSession 和宿主 scope，避免同账号切项目后关掉新页面的弹窗。

### 2.5 子项、关联与链接

子项按原生 data 的 id 关联 canonical；状态分布由子项 state_id 和目录的 State.group 计算，不把 sub_issues_count 当所有状态分布。新接口只接同项目子项；现有测试中人为跨项目子项改为真实合同样本，跨项目权限测试放到关联资源，不削弱权限性质。跨项目状态使用已有 workspace 状态目录；成员/标签有实际展示需求时等待既有目录，拒绝交回。

关联保留 relation.id/type/created_at 及关联目标 ID，展示从唯一 canonical 读；批量返回和重复确认不重复插入。删除参数从目标关系行取得 relation.id。链接存原生 IssueLink，metadata 的 title/favicon/url/favicon_url/error 按可空类型消费；创建/更新后 crawler 的 blocked/failed 是元数据状态，不等于 HTTP 写失败。调用方不再读取 created_by/detail 等旧字段。

### 2.6 评论、表情和时间线

评论与动态各自首次读取全部页（默认50，允许1–100）；页链在同一读取轮次中，任一页失败不得把已有部分当成“完整成功”或丢失上次成功 cursor。首次完整读取替换集合，删除的旧评论因此消失。每份集合记录**最后一页请求使用的 cursor**（首/唯一页为 undefined），增量从该 cursor 再读，沿 next_cursor 到 null；不能从末页返回的 null 重置为第一页或停止。

合并按 ID 覆盖，再按服务端 `(created_at,id)` 升序排序；时间比较复用保留纳秒的 issue-order，不能 Date.parse 截毫秒。描述合并动态会移动旧 ID 的 created_at，必须重排同一 ID，不能假定 append-only。评论修改按 updated_at/确认重放保护；动态没有 updated_at，使用读取轮次及原生 created_at，不伪造版本字段。合并展示带 resource kind，避免评论/动态 key 混淆。系统 actor_id=null 使用系统文案；actor/member 信息来自目录，缺失与停用有已有 fallback。

评论 reactions 是权威读取源。评论表情 store 改为评论记录上的派生/确认操作；删除独立 fetchCommentReactions 假接口及双份数据。工作项表情保留独立 collection；add 幂等答复按 id 去重，remove 只移除当前登录者同 reaction。编辑/删除评论与其表情共享父 issue 队列，旧 GET 晚到不能撤销确认。其他浏览器编辑/删除最后增量窗口之外的评论不保证立刻出现，关闭重开全量读可见（设计§3.13限制保留）。

### 2.7 描述版本与 M7 编译边界

M4 新版本 store 使用 DescriptionVersion/Page/Detail，按 last_saved_at/id 倒序分页，列表能继续加载且可选历史版本；只在项目管理员/成员时调用。详情与列表分别按资源 scope 缓存，选项快速切换时旧详情不得盖新项。恢复调用普通 description_html PATCH，等待确认后关闭/提示，不另造 restore API，不承诺每次恢复必生成新版本。

共享 DescriptionVersionsRoot 当前同时服务 M4 与 M7。M4 入口使用新 store；M7 的 IntakeWorkItemVersionService 仍属未接通领域，保留它自己的旧 URL、body 和精确旧 DTO，旧 DTO 移到 `C/services/inbox/` 的专属文件，不继续从共享 types 输出通用旧版本类型。共享下拉/模态显示只接明确的展示 props（id、ownerId、lastSavedAt、HTML 等）或泛型 accessor；M4 原生 DTO 不转换成 Plane 形状，M7 DTO 不冒充生成 DTO。共同组件不拥有两个传输协议，不代替 M7 接 API。移出 DTO 的可选性/空值保持原样；M7 既有调用行为和错误仍保留，单测明确此边界。

### 2.8 个人草稿

草稿是 Draft，不是字段摊平的 Issue。store 按 workspace/本人持有分页、total、记录与 loader；只有 workspace 管理员/成员可读。scope 包含 member 权限，退出、降级、workspace 消失立即遮住旧内容；不同 workspace 的 cursor/loader 不互相覆盖。输入 builder 显式白名单 IssueDraftPayload，允许暂不满足发布条件的数据；省略 project_id 与显式 null 区分。PATCH payload 是整份替换，表单保存要保留它拥有但未编辑的合法字段，不能以局部 payload 覆盖其它字段。

表单清空 parent_id/start_date/target_date 等可空输入时，从待保存的完整 payload 删除该键，不发送生成合同不接受的 null，也不把旧值合并回来；数组清空使用 []，合法空字符串按原值保存。顶层 project_id=null 才表示清空草稿目标项目，与 payload 内字段不同。无项目草稿仍可保存；现有普通工作项创建/更新的必选项目保护不得误用于草稿分支，发布所需项目和完整语义由服务端校验。

直接消费者同步改为Draft.payload，缺项目不隐藏整行/编辑器，不因没有项目目录而覆盖已有草稿。读取本人的草稿由服务端限定，删除入口使用本代工作区成员权限和资源scope，不从不存在的created_by或项目管理员角色推断草稿所有权。保存/删除拒绝保留输入及弹窗；确认结果回调只做本scope的界面跟进，不能再用返回实体补发第二次修改。这里是原生草稿合同的最小调用闭合，完整草稿页面故事仍归P15。

发布一个 draft 的保存和发布是同一串行操作：先 PATCH 最新完整载荷/目标项目，成功再 POST publish（无 body）；PATCH 失败不得发 POST；publish 409 draft_changed/422/403 保留保存成功的 Draft、表单和错误，不能假装已经发布。publish 201 的 IssueDetail 经过 root 的**创建确认协调入口**（冻结 before/pending、canonical、所有列表/计数同 createIssue），不能仅 acceptRead。root 将既有 createIssue 的协调流程抽成小范围复用入口，实际请求由调用者给定；不新增另一套创建缓存。成功删除该 draft 的确认要能重放到晚到列表；total 由服务端回答/确认来源处理，不按一次局部页长度猜全量。

**P11-S1裁定A：**创建/发布成功后的编号宽度由实际进入当前列表的确认工作项派生，不为列宽额外刷新项目，不添加或伪造 `Project.next_work_item_sequence`。若将来其它功能真实修改项目公开字段，仍按该功能自己的确认流程刷新，不能把此裁定扩大为禁止一切项目刷新。跟进导航/提示依 followInSession 与宿主 scope；其它窗口的草稿并发由 draft_changed 表达，不新增客户端锁协议。

### 2.9 提及、按需读取与宿主

编辑器 @ 提及只从成员目录过滤：有 projectId 时使用有效项目成员与 MemberUser；无 projectId 的草稿使用当前工作区有效成员（显式排除 inactive）。按 display_name 匹配，保留既有最多五条；不读 email，不借当前筛选器限制成员，不调用 entity-search。每个输入轮次只提交最新 query/current scope 结果，拒绝由编辑器受控状态显示或交回，不能吞成“无人”；只读编辑器不发建议请求。四处 searchEntity 调用、方法和独占请求类型同 Task 删除；description_json/comment_json/共享 JSONContent 没有合法读者后删除，不误删 editor 内部自己的 JSON 类型。

按需 states/members/labels 的全部入口建立清单：拒绝显示错误、结束 loader、可重试，fire-and-forget 改 await/受控 UI；不把 catch 空数组当成功。详情 SWR 全部 useSessionSWR，key 带 loginId、scope 和 permission identity，禁直接 swr 的路径范围同步扩，不新增豁免。

WorkItemLevelModals 留在现有 workspace provider；`projectId ?? workItemIdentifier` 改变及卸载用 useLayoutEffect cleanup 结束它拥有的开关。详情/peek/列表的实际宿主结束自己设置的开关、peekIssue 及 issueCrudOperationState/issueLinkData/relationKey，重设为原初始值。并存的详情与 peek 必须有所有权标记/租约或等效局部机制，旧宿主的 cleanup 不能关新宿主的弹窗；旧请求完成也不能改新 scope 的 UI。isArchiveIssueModalOpen、附件宿主已由 P10a 删除，只核零残留，不重建。

### 2.10 错误、标签与守卫

issue-modal/base.tsx 的旧 error?.error 经 errorMessageKey/现有文案表；只有新增公开码才同步 en/zh-CN 及码测试。五个工作项新建标签入口走 useLabelChanges().create(projectId, LabelCreate, underField, done)，重名 project.label_name_taken 落 name 字段；成功后跟进按会话。没有 projectId 时只使用已有、当前仍可见项目的 labelMap；未取目录的标签不显示，不声称有 workspace 标签接口，不传空项目 ID、不逐项目扩取数。工作区完整标签恢复仍归 M7（M3 handoff§6）；本段核限制和拒绝边界。编号宽度按P11-S1裁定A实现（下节）；projectIdentifier 为可选 string 的现状保留，不恢复项目对象参数。

### 2.11 编号列宽（P11-S1裁定A）

一个列表布局的所有行共用同一编号列宽，包括展开的已加载子项；项目/归档/profile三个列表入口复用这份显示规则。输入仅该布局实际纳入显示、当前有权读取的工作项及各自Project.identifier，不扫描全局canonical，不读取未加载/已失权编号；“可见”指通过权限、筛选与展开条件，不等于仅视口内已渲染的DOM。跨项目profile按每个实际完整编号计算所需宽度，再取最大，不拼接不同记录的字段冒充一个编号。

复用现有calculateIdentifierWidth的像素估算（第二参数是实际sequence_id数字，不是位数），它只是显示函数，不代表项目最大序号。Project尚未到时不渲染undefined-编号或猜标识；等目录到达再参加计算。空集合初值0，关闭key显示时输出0；目录和行的HTTP读取仍由原所属store负责，宽度逻辑零网络请求。

局部布局状态只记数值宽度与scope/行注册身份，不缓存另一份Issue/Project。建议在List拥有局部宽度状态，在IssueBlockRoot（RenderIfVisible外）注册当前合法完整编号的所需宽度，覆盖视口外已加载行及递归展开子项；用已有props或局部context传给行，禁止各行自己算不同列宽。注册须有宿主所有权和清理，不让卸载的旧行写入新布局；实现细节可以等价替换，行为约束不变。

scope由当前loginId、workspace、列表种类、project或profile user/view，以及root的权限identity组成；改变时同步重置已记宽度、从新scope当前合法行重算，不能先画一帧旧范围。普通分页/确认新行/展开较长子项只增宽；同scope删除、折叠或筛选缩小允许保持已达到宽度以免抖动。权限改变必须重算，不沿用这种保持策略。卸载时释放；重新挂载从当前数据开始，不存localStorage。独立详情编号按自身内容展示，不新增列表宽度状态。

必要样本：WEB-9/WEB-10/WEB-100、不同长度项目标识的profile列、跨分组统一值、分页和展开子项增长、视口卸载不缩列、过滤/删除保持、权限收回/ABA/切scope重算、目录未到/空集合/key隐藏、两个布局不互改。代价是新页首次出现更长编号时列可变宽；无需提前为未来编号留位。无需新增项目公开字段、后端任务或架构守卫例外。

phase 在**第一条实施提交**改 M4/P11，本次准备不动。当前无 until=M4/P11 例外；P11 应删除的旧地址共有21个 exact key、期限 M4/P12，随真正消费者消失逐项减少，不能延长。新增 label-create-direct 限制实际 M4 创建入口；旧 IssueService 在 T4 最后消费者迁移后删除。旧搜索调用仍由 P12/P14/P15 负责，M5–M7 旧接口只按各自登记保留，不给新的 M4 旧调用开洞。

## 3. 补充裁定

D1–D7 是既定设计内的执行细化；D8曾因上游合同缺口提交后上报，负责人2026-10-30已选择A，现按裁定完成文档修订，不扩大项目公开协议。

| ID | 裁定、理由、错误代价 |
|---|---|
| D1 | 按资源同步换型，T4后盘点原T5的服务、三store及真实消费者和测试将超过约1500行，按§6预留拆为T5评论及内嵌表情、T6工作项表情，总十五Task。无新协议或范围，代价是旧工作项表情服务比评论多保留一个任务，T6末必须删除。 |
| D2 | 搜索生成合同本段登记，薄 service 延到 P12 首个生产调用方同任务落地；P14/P15 接其它消费者，最后删旧搜索类型。当前没有 P11 独立生产读者，提前建最终无读者导出会绕 knip。页面时间点不后移，代价是不能声称 P11 已接通搜索。 |
| D3 | 归档 header 和 issueId redirect 的两个 IssueService 实例的数据接线前移 P11 T2；P13/P14 仍做完整页面验收。旧 IssueService 本就必须 P11 删，延后会延长有界旧接口。七处原要求加这两处，共九处 M4 实例在 P11 消失，13处总门禁仍 P15。 |
| D4 | M7 版本旧 DTO 收归 M7，只有共享展示 props 分离，真实旧请求保持；符合设计要求的 M7 编译闭合，不迁 M7 业务。错误代价是偷换 DTO 导致未接通领域再添隐形假合同，故设专门测试。 |
| D5 | 沿用 root 权限令牌/队列/Reconciled，资源只增加必要 scope 元数据；不建立第二个权限观察者、实体副本或通用同步框架。副作用成功但补读失败按确认与待核两阶段处理，不能回滚已成功写。 |
| D6 | 编号解析交给已实现服务端，前后核 workspace/返回项目权限；不是把本地大小写匹配当权限。代价是未知编号最多多一次服务端拒绝及必要目录读取，安全由服务器与落库前 scope 证明共同保证。 |
| D7 | 当前只准备，取消本轮完整产品原型与逐 Task 代码重放；采用合同/调用链/现有测试静态预检及独立 builder-check。所有行为证明留实施，未跑命令不写绿；方法调整由02§3允许，且遵从负责人“暂不实现”。 |
| D8 / P11-S1 | **负责人已裁定A。**编号列宽来自当前布局已加载可见编号，按§2.11统一/增宽/重置；不新增Project序号字段，不为此刷新项目。代价是更长编号首次出现时列可增宽；原公开合同保持，历史B方案留决策记录。 |

## 4. 验收标准

- [ ] A1：所有 M4 详情服务为本代生成客户端；请求/响应和拒绝逐方法测试；旧 IssueService 与过期关键词条目删除，守卫不放宽。
- [ ] A2：唯一 canonical、纳秒版本、删除/退役/权限 ABA、两工作区、多个并存详情资源全通过；无旧响应/缓存复活。
- [ ] A3：子项状态分布/计数、关联关系 ID、链接元数据及计数由原生合同/权威确认驱动；所有修改串行、拒绝不改副本。
- [ ] A4：评论/表情不乐观；无不存在的评论表情 GET；全页和增量 cursor/ID 重排/纳秒排序、并发确认和二页失败覆盖。
- [ ] A5：版本读取权限、分页/快速选择、恢复等待确认；M7 独立旧 DTO 不冒充生成类型，原消费合同保留。
- [ ] A6：草稿分页/权限/完整 payload、PATCH→无 body publish、拒绝保留、发布接创建协调通过；确认行按A方案影响当前列表列宽，无列宽专用项目请求。
- [ ] A7：原七处模块实例与两个前移实例消失；提及四处改成员目录，JSONContent/comment_json/description_json/is_bot 的 M4 旧读写删除。
- [ ] A8：每处按需拒绝可观察、结束加载/重试；详情 SessionSWR 含会话和权限身份；宿主换 scope/卸载/晚答不串弹窗。
- [ ] A9：错误按码、标签统一创建、无项目标签限制准确；编号统一列宽、分页/子项增长和scope/权限重置按§2.11通过；dead/oxlint/knip 与新守卫通过。
- [ ] A10：逐 Task 证据、至少15项独立性质变异、完整 F、准确最终分支 CI、deep-review 与修复复审闭环，移交记录真实。阶段先不合并，另待负责人安排。

## 5. 不在范围与后续

| 接收阶段 | 明确剩余责任 |
|---|---|
| P12 | IssueSearchService 与 ExistingIssuesListModal 首次生产接通；列表四布局/拖放/快捷操作/焦点，I3/I4/I6/I7；新搜索不能依赖 P11 已有 wrapper。 |
| P13 | 归档/profile/筛选完整页面、按 id key 与后退；P10b R36 DateDropdown 嵌套按钮和 R38 筛选 render 警告仍按原移交。 |
| P14 | 详情/创建/父子/关联/链接的完整交互与故事，parent 搜索与 M7 查重调用的编译接通；issueId/header 的数据接线已安排 P11，不能再依赖旧 IssueService。 |
| P15 | 评论/动态/版本/草稿真实页面及搜索命令面板、所有剩余搜索旧方法/类型；13处实例最终零命中。 |
| P16 / M4 收尾 | 全仓守卫、no-shadow、dead、输入 handoff 关闭条件；M4 §14/总体 §9.4；统一 M5–M8 M4-closeout。 |
| M5–M7 | 文件/迭代模块/视图收集箱自身业务接口与有效实体整页。M7 旧版本 DTO 的最后删除由其接口切换负责；不把本段编译闭合当领域验收。 |

## 6. 风险

资源 ID 与工作项 ID 混用会删错关联；多 ID 队列的重入会自等；成功写后的补取失败会被误当写失败；permission ABA 会让 SWR 命中旧值；整份 payload 替换会丢字段；老动态 ID 的时间移动会漏显示。这些均设独立断言和生产变异，不靠编译证明。计划单 Task 估算含测试上限约1500行，T5/T9 超额时先按资源局部拆分（总数约16），同步 spec/plan，再实施，不堆补丁。

## 7. 已知限制、交接与关闭条件

当前真实 main 仍包含未迁移详情旧地址和 DTO，这是 P11 的实施输入。准备稿没有解决它们、没有跑产品测试。P10b 归档中历史变异原件/覆盖限制继续原样保留，不用本次静态盘点补记历史通过。M4 总体验收与所有输入 handoff 的顶层 open 保持；本段执行完成时只写对应子项处理结果，不建未来最终交接文件。

阶段完成需A1–A10实证、整分支deep-review后局部问题全闭环；本段仍先不合并，集成授权另行取得。当前已经获得实施授权，T1–T5完成，后续按修订计划继续；准备阶段的“暂不实现”不是当前阻塞。

## 附录 A：准备证据（2026-10-30）

P10b main `df5d3bbc` 的 [CI 40908365144](https://github.com/open-nerve/NerveProject/actions/runs/40908365144) attempt1 三任务全绿；产品树等于阶段交付 `2aaef85c`。归档根为主项目 `.superpowers/sdd/P10b-integration-20261030/`，两 manifest 和 cleanup-receipt.json 记录真实校验/清理。

本轮准备根为主项目 `.superpowers/sdd/P11-preparation-20261030/`，在 worktree 外，未来归档必须一并保留。api-inventory.json 从本地 YAML 提取30个相关 operation；consumer-inventory.json 是77条关键词路径/哈希清单，不等于77文件全文审查。预检覆盖、独立问题及修订见 review；没有原型、逐 Task 产品重放、编译、Vitest 或 E2E 执行证据。静态预检不能代替实施的 F 与准确分支 CI。
