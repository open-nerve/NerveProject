---
status: open
from: M4/closeout
to: M7
created: 2026-11-02
---

# M4 交给 M7：收集箱来源、系统身份、通知权限、视图和旧接口的最后读者

这是接收方尚待完成的工作。M4 输入的“转交关闭”只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M4 设计 §13](../../M4-issue-core/M4-design.md#13-交接的落点)、[逐条交接索引](../../M4-issue-core/reviews/closeout-handoffs.md)和该索引中的各 Phase spec §5 / review §6。
M4 的最终 C12、准确分支 CI 和未来 main 集成分别由[验收台账](../../M4-issue-core/reviews/closeout-acceptance.md)记录；不以阶段历史绿灯替代最终输入。

<a id="M7-01"></a>
## M7-01 收集箱来源与系统创建者

此项承接 [M1-P3](../../M4-issue-core/handoffs/M1-P3-trim-platform.md)、[M1-P5](../../M4-issue-core/handoffs/M1-P5-brand.md) 的完整细节，以转交关闭 M4 输入。

- `EInboxIssueSource` 现有 `IN_APP`、`FORMS`、`EMAIL`；前端 `inbox-issue.service.ts` 只创建 IN_APP。来源取值由 M7 定；保留表单或邮件来源时，改写项目功能设置的说明“在将工作项加入项目之前，先进行审议和讨论”，使其准确描述保留用法。
- `inbox/content/issue-root.tsx` 对 FORMS 显示 **“Intake Form user”** 的分支必须随此决定处理。M4 工作项动态里的来源显示已在 P15 删除，不重新加回。
- 系统代建工作项记在谁名下、怎样显示，依 M4 决策点 4 设计；不按邮箱、名字或前缀猜测系统身份。M1/P5 已删除 `inbox/sidebar/inbox-list-item.tsx` 中 `intake@plane.so` 的 “Plane” / “P” 头像，以及 `issues/peek-overview/properties.tsx` 对 `-intake` 名字显示 Plane 的分支（小于 768px 的布局也用到后者）。不要恢复它们。
- M4 自动归档的 actor 为 SQL NULL / Event.ActorID=nil，页面显示 SITE_NAME；人工恢复仍是真实操作者。不造零 UUID、假 bot 或通用系统账户替代这个合同。
- 自动归档加收集箱状态条件；动态动词 CHECK 随真实状态变化扩展。`select-duplicate.tsx` 已改接原生搜索并能编译，M7 接通后还要验证它的真实页面。

**何时验收：** 收集箱 API/页面设计和首次接通阶段。真实三种来源或明确删去的来源、系统代建/人工操作、窄屏、设置文案与查重操作逐项验收。

**处理结果（M7/P2a）：** 接口一侧已定（[P2a spec](../specs/P2a-intake-submit-triage.md) §1、§2.2、§2.4）：只有应用内提交，`intake_issues` 不建来源列与外部来源三列，`FORMS`、`EMAIL` 不保留；提交者是条目与工作项共同的创建者（真实的调用者），没有系统代建；动态动词的 CHECK 只为 `field = 'intake'` 放开五个状态名（`00051`）；自动归档按状态组判定，在收集箱里的工作项在 Triage 状态组，按构造不会被自动归档，不另加条件。**仍待 P5**（收集箱的页面接通）：删除网页的 `FORMS` 分支（“Intake Form user”）与旧的 `inbox-issue` 服务、项目功能设置的说明按“只有应用内提交”改写、`select-duplicate.tsx` 的真实页面、窄屏；本条在 P5 关闭。

**处理结果（M7/P5a）：** 网页的 `FORMS` 分支（“Intake Form user”）随旧详情树 `inbox/content/issue-root.tsx` 在 T6a 删除（`3ea45cd4`），旧的 `inbox-issue` 服务随旧数据层在 T6c 删除（`bd1802fd`）；项目功能设置的说明按设计不改：只有应用内提交，“在将工作项加入项目之前，先进行审议和讨论”照旧准确（M7 设计 `:97`，[P5a spec](../specs/P5a-web-intake.md) R24），该子项关闭。`select-duplicate.tsx` 的真实页面在 H10 (page)（`e2e/stories/collaboration/h10-intake-triage-page.spec.ts:152`：经选择器标重复，选择器不列条目自己与收集箱里别的条目）与 H20 (page) 三种引擎走过；窄屏在 H20 (page)（`h20-intake-pages.spec.ts:308`，800 宽：列表盖在详情上，按一行之后藏起，窄页头的菜单拒绝、接受、删除）。新建弹窗的其余随 P5b，本条在 P5b 关闭；保持 open。

**处理结果（M7/P5b）：** 新建弹窗的其余在 P5b 做完（[P5b spec](../specs/P5b-web-intake-files-embed.md) §2.1；`fa78bb11`、`ec6b91a5`）：关闭政策、名字的提示与被拒、`ModalTitle`、“Create more” 是开关本身、离开 `getTabIndex`、英文改键、死代码删除。H8 (page) 两份故事（`e2e/stories/collaboration/h8-intake-submit-page.spec.ts`、`h8-intake-close-page.spec.ts`）三引擎核：管理员、成员、访客经弹窗提交（请求体、条目行、选中新条目、列表与侧栏的数），名字的三种被拒，收集箱在弹窗开着时被关掉之后提交答 409 的提示，关闭政策与在途的焦点。**本条关闭。**

<a id="M7-02"></a>
## M7-02 通知发送时重新判权

- 实现订阅/取消订阅接口及 `is_subscribed`。M4 服务端自动写的关注行不保证接收者以后仍可读；被移出、降为只看自己的访客、guest_view_all_features 关闭，都可能失权。
- **通知发出的那一刻**重新按 `issue.read` 和 `Issue.HiddenFrom` / OwnOnly 判定；排队时的权限不足以作为发送许可。涵盖工作区管理员兼项目访客，不能按角色大小粗比。
- 消费 M4 §3.19 事件。删工作项对子树逐项发 `issue.deleted`；删项目/工作区走父事件，不逐项发工作项事件。通知内容不得引用接收者看不到的另一端或旧描述。
- 评论、表情有事件，但不因此写成工作项动态；元数据爬取、无变化/合法 ID 迁移不凭空发新事件。摘要使用清洗结果，不能以搜索结果代替正文合同。
- 动态的 `comment` 列是否进入公开接口由 M7 决定并生成协议、补权限和回归。

**何时验收：** 第一个通知消费者合入时。排队→撤权→发送的交错、两种 guest 标志、成员离开、系统 actor、父范围删除和回滚不发通知都要有反例。

**处理结果（M7/P1a）：** 关注、取消关注的接口与 `IssueDetail.is_subscribed` 已实现（[P1a spec](../specs/P1a-subscriptions-events.md) §2.3、§2.4）：按 `issue.read` 与只看自己判定，事务里 `FOR SHARE` 锁工作项行后再判定，与删除串行；描述、评论的保存只让新被提及的人自动关注（R7）。通知要用的事件已补：评论、链接事件带所在工作项（删除类在 Old 一侧），工作项与评论的事件带这一次新提及的账户（§2.2）；“写入但未改变”与正文不变的保存照旧不发布。动态的 `comment` 列不进公开接口（页面按字段与值渲染，设计 §11）。发送时的复核、通知内容不引用看不到的另一端、各反例由 P1b 交付；本条保持 open。

**处理结果（M7/P1b）：** 发送时复核已实现（[P1b spec](../specs/P1b-notification-core.md) §2.4、§2.7、§2.9）：订阅者只在写的事务里投 `notification.fanout`，任务在事务外以 `FOR SHARE` 锁工作项行后，按那一刻的 `issue.read` 与只看自己（发送时快照里的创建者，`HiddenFrom`）批量判定收件人（`DecideMany`，与 `Authorize` 在权限矩阵的种子上逐格等价，含工作区管理员兼项目访客与 `guest_view_all_features` 的两种取值）。反例都在 bootstrap：排队→撤权→发送（移出项目、结束工作区成员、降为只看自己的访客、移出私有项目、管理员兼访客，`notification_recheck_test.go`）；系统执行者（自动归档，执行者为空，`notification_issue_writes_test.go`）；父范围删除（项目删除时通知的删除等工作项删完再删，`notification_interleaving_test.go`）；回滚不发通知（`notification_rollback_test.go`）。通知不引用旧描述：描述的变化不通知，HTML 不进任务参数，评论只存纯文本摘录；父工作项与关联的另一端只存 id，按读的人此刻的可见解析由 P1c 交付。本条保持 open，到 P1c 的读与另一端的解析合入时关闭。

**处理结果（M7/P1c）：** 读时的可见与另一端已实现（[P1c spec](../specs/P1c-notification-api.md) §2.5、§2.6、R4、R7）：通知的四个接口只列读的人此刻读得到的项目（含已归档的）里的，只看自己按存下的创建者（与 `Authorize`、`HiddenFrom` 的等价测试）；父工作项与关联的另一端按读的人此刻的可见给出编号与名字，读不到的连 id 也不给，回答里的 id 按白名单（新加关联类型而通知没跟上时不漏出 id，守卫测试）。排队→撤权→读的各种情形见 `notification_visibility_test.go`、`notification_other_ends_test.go`。**本条关闭。**

<a id="M7-03"></a>
## M7-03 视图与跨项目列表

- P10A-S1 选择 A：项目视图/工作区视图自身页面完整可用、整页零失败由 M7 接通时验收。接收 [P10a review §6–7](../../M4-issue-core/reviews/P10a-web-issue-types-review.md#6-移交事项) 的方法、URL/query、触发、状态、源码归属和归档证据。六入口的 75 请求＝69 父资源 200＋6 领域 404，其中两条领域失败属于视图；不称有效视图通过。
- 恢复 `EIssuesStoreType.PROJECT_VIEW` / `GLOBAL`、两个布局根、`workspace-views/[globalViewId]` 及项目视图详情的列表部分。依据 Plane 在**新引擎**上实现，不能恢复旧列表引擎/空壳或借过渡记录永久放行失败。
- 新增工作区跨项目列表分支，保持只读 REPEATABLE READ 快照、逐项目权限/OwnOnly、统一游标和比较器；个人主页按标签分组需要完整的工作区标签目录。
- 旧 WorkspaceService 留下的真实视图方法为 `createView`、`updateView`、`deleteView`、`getAllViews`，已有 transport 测试；自身 API 接通并替换最后读者时删除旧方法，不能在 M4 提前砍掉有效调用。

**何时验收：** 有效视图首次开放前，替换旧接口/例外并删除对应过渡记录；用真实有效实体测详情、列表、筛选、跨项目访客、无权项计数、失败重试与会话切换，整页零非预期失败。

**处理结果（M7/P3a，跨项目列表一支）：** 工作区跨项目列表已接通 `GET /workspaces/{slug}/issues`：只读 REPEATABLE READ 快照、范围为有效成员的未归档项目（`VisibleProjects.Visible`）、逐项目只看自己、共享游标与比较器（[P3a spec](../specs/P3a-workspace-list.md)，规模证据见 [P3a 阶段记录](../reviews/P3a-workspace-list-review.md) §4）。项目视图与工作区视图随 P3b；页面、完整的工作区标签目录（个人主页按标签分组与工作区列表的标签筛选两个使用者）与旧 `WorkspaceService` 四个方法的删除随 P6b（设计 §12）。保持 open。

**处理结果（M7/P3b，视图的接口一支）：** 项目视图与工作区视图的接口已接通（`listProjectViews`、`createProjectView`、`listWorkspaceViews`、`createWorkspaceView`、`getView`、`updateView`、`deleteView`，[P3b spec](../specs/P3b-views.md) §2.4）；已存的词原样作为查询发给项目列表与工作区跨项目列表都被接受（组合测试 `TestWorkspaceViewsThroughTheApp` 的往返与 e2e `h14-workspace-views.spec.ts`）。页面（两个布局根、`PROJECT_VIEW`/`GLOBAL` 的 store、视图详情的列表部分）与旧 `WorkspaceService` 四个方法、旧 `view.service.ts` 的删除仍随 P6a/P6b。保持 open。

**处理结果（M7/P6a，项目视图一支）：** `EIssuesStoreType.PROJECT_VIEW` 在新引擎上恢复（[P6a spec](../specs/P6a-web-project-views.md) §2.5、§2.6、R1、R6）：它是项目列表页的一种（`core/store/issue/project-view/` 的 `ProjectViewIssues`、`ProjectViewIssuesFilter`，查询来自内存里的草稿 `ViewDraftService`），没有恢复旧列表引擎或空壳；每个分派点都接上，两个 `switch` 以 `satisfies never` 穷尽；视图页的根 `core/components/issues/issue-layouts/roots/project-view-layout-root.tsx` 与视图页 `[projectId]/views/(detail)/[viewId]/page.tsx`（以 `viewId` 为 key，读中、不存在、失败重试、停用四种状态）。旧 `core/services/view.service.ts` 连同旧 store 与 hook 在 T10 删除（`6ea2f0c9`），关键词 `old-view-urls` 拦它的地址。整页以真实有效的视图验收（H12 (page) 四份、H20 (page) 两份，三种引擎）：详情与它的列表、四种布局与翻页、筛选与“更新视图”、读失败与重试（Chromium）、管理员、成员与只看自己的访客、换会话（`e2e/stories/collaboration/h20-view-pages.spec.ts:214`）、迟到的回答；`t12-m7-pages` 200 过 1 跳（跳过的是 P5b H8 的 WebKit 一格），整页没有非预期的失败。项目视图一支关闭；`GLOBAL`、工作区视图的页面与布局根、旧 `WorkspaceService` 的四个方法与完整的工作区标签目录随 P6b。保持 open。

**处理结果（M7/P6b，工作区一支）：** `EIssuesStoreType.GLOBAL` 在新引擎上恢复（[P6b spec](../specs/P6b-web-workspace-views.md) §2.3、§2.8、R1）：新的范围臂 `"workspace"`（`core/store/issue/helpers/issue-list-read.ts`；只读快照、逐项目权限与只看自己在服务端，客户端核对行的范围）与列表 store `GlobalIssues`（`core/store/issue/workspace/`），没有恢复旧列表引擎或空壳；每个分派点都接上，两个 `switch` 穷尽；工作区视图页的根 `core/components/issues/issue-layouts/roots/workspace-view-layout-root.tsx` 与视图页 `workspace-views/(detail)/[globalViewId]/page.tsx`（以 `globalViewId` 为 key）。个人主页按标签分组与工作区列表的标签筛选两个使用者经新的完整工作区标签目录（`GET /api/v0/workspaces/{slug}/labels`，R16、R17）。旧 `WorkspaceService` 的四个方法（`core/services/workspace.service.ts` 连同它的测试）在 T11 删除（`7ddc1d4b`），关键词 `old-view-urls` 拦它的地址。整页以真实有效的视图验收，`:58` 的验收词逐个的去处：详情——H13 (page) 的已存视图页（`e2e/stories/collaboration/h13-workspace-view-page.spec.ts:73`）；列表——四个默认视图（`h13-workspace-list-page.spec.ts:39`）；筛选——含项目与标签的条件（同 `:39`）；跨项目访客——只看自己的访客在只看自己的项目里只见自己的、在另一个项目里见全部（`:250`）；无权项计数——接口故事 `h13-workspace-list.spec.ts:101`（总数按数据库逐条），页面只核行（不是成员的项目、已归档的项目的行不出，`:39`）；失败重试——列表读（`:250`）与视图读（`h13-workspace-view-page.spec.ts:247`），只在 Chromium；会话切换——H20 (page) 另一个标签页换账号（`h20-workspace-view-pages.spec.ts:203`）。新故事三引擎（`t14-m7-pages` 两引擎 162 过、WebKit 单独 80 过 1 跳，跳过的是 P5b H8 的 WebKit 一格），整页没有非预期的失败（每个故事以 `apiFailures`、`oldApiRequests`、`pageErrors` 为空与 `expectQuietConsole` 结束）。两支都已关闭，**这一条关闭。**

<a id="M7-04"></a>
## M7-04 收集箱旧 DTO、版本与命令面板

- P10a 仅准确化 `services/inbox/inbox-issue.service.ts` 的既有输入/响应：create 描述为 `Partial<IssueDetail>`；嵌套响应为 `Issue | IssueDetail`，普通列表无描述；updateIssue 的旧请求输入为 `Partial<IssueDetail> & { skip_activity?: string }`，状态/查重/延后元数据保留自身 DTO。
- string flag 只属该旧 wire 输入，不进入 canonical 实体、分组、回滚；M4 新工作项接口只有 boolean。M7 接自身接口时删除旧地址、到期例外、最后旧 request type 与版本 DTO/service/`inbox/content/description-versions` 的过渡，不能给 M4 新接口恢复兼容层。
- 命令面板迭代、视图、收集箱结果按真实接口恢复；删除工作项时清掉最近访问，由 M7 扩自己的删除图/消费者。
- `frame-src` 仅在确实实现需 iframe 的功能时加具体来源、CSP 说明和测试；M4 原生 emoji 不需要它，不加宽泛放行。
- 待 lazy editor 就绪期间收集箱 discard/文档 Escape 的关闭政策、未等待筛选/视图写后关闭等，仍属 M7 页面验收；需统一 dirty/pending/busy、输入保留、嵌套 Escape/IME、拒绝与换宿主迟到答，不能借 M4 属性兼容宣称页面已修好。

**何时验收：** 相应 API/页面首次接通前；旧接口最后读者清除时同步类型、关键词、测试和本交接。

**处理结果（M7/P3d，删除工作项时清掉最近访问）：** 已实现（[P3d spec](../specs/P3d-recents.md) §2.6、R10）：navigation 的订阅者 `navigation_forget_deleted_issue_visits` 订 `issue.deleted`，在删除的事务里软删**所有人**指向这个工作项的最近访问，时刻与删除者取事件的，`visited_at` 不动；`deleteIssue` 的子树与 `deleteIntakeIssue` 删未接受的条目都经删除核心、每个工作项一个 `issue.deleted`，所以逐个盖住；已接受的条目的删除只删条目行，工作项与它的访问照旧。项目、工作区删除经 `navigation.delete_rows` 的第三步。证据：`TestRecentVisitsFollowDeletedWorkItems`、`TestRecentVisitWritesInterleave` 的记录 × 删除各格、e2e H17 的删除一步。旧 DTO 与版本、命令面板与页面的各项仍随 P5–P7；保持 open。

**处理结果（M7/P5a，旧 DTO、版本与命令面板）：** 收集箱的过渡都已删除，没有给 M4 新接口加兼容层（[P5a spec](../specs/P5a-web-intake.md) §2.10）：旧的 `services/inbox/inbox-issue.service.ts`（旧地址与它的 DTO）、`@nerve/types` 的 `inbox.ts` 与最后的旧请求类型 `TLegacyIssueUpdateRequest` 随旧数据层在 T6c 删除（`bd1802fd`）；版本的 service `services/inbox/intake-work_item_version.service.ts`、DTO `intake-description-version.ts` 与 `inbox/content/description-versions.tsx` 在 T6a 删除（`3ea45cd4`），描述版本改经 M4 的版本接口（收集箱条目在工作项的写队列上恢复，§2.4）；关键词 `copy-hook` 的收集箱例外在 T6a 删除，新规则 `old-intake-urls` 拦住 Plane 的收集箱地址（T6c）。命令面板的收集箱结果：Plane 也没有收集箱的搜索结果，设计保留“转到收集箱”（M7 设计 `:245`），按 R24 关闭该项；“转到收集箱”改经入口规则 `intakeEntryOn`（M6-M7-07）。`frame-src`：收集箱页不嵌 iframe（`git grep -i iframe -- web/apps/web/core web/apps/web/app web/packages/editor/src` 为空），CSP 不改。新建弹窗的关闭政策随 P5b；迭代、视图的命令面板结果随 P6a/P6b。保持 open。

**处理结果（M7/P5b，收集箱的关闭政策）：** 上面第五条（lazy editor 就绪期间的关闭政策等）的收集箱部分逐项（[P5b spec](../specs/P5b-web-intake-files-embed.md) §2.1、§5；`fa78bb11`）：
- discard 与文档 Escape 的关闭政策：两个文档级的 Escape 删去，Escape、点背景与 Discard 走 `requestClose` 一条规则，编辑器没建成可关（单测 `create-root.close.test.tsx`；H8 (page) `h8-intake-close-page.spec.ts:97` 在扣住编辑器代码块时核，Chromium、Firefox）。
- dirty/pending/busy：有改动先确认放弃、没有改动直接关；提交在途时弹窗不给 `handleClose`、`requestClose` 也不关，Discard 与 “Create more” 停用（R28；H8 (page) `:156` 三引擎断言 Escape、点背景与被拒之后焦点都在弹窗里）。
- 输入保留：确认框的取消与 Escape 留住表单与输入（H8 (page) 两个测试）。
- 嵌套 Escape：负责人（Combobox）与截止日（Popover）开着时 Escape 只关它们（R7，H8 (page) `:156` 三引擎）。
- IME：组字时的 Escape 由 `ModalCore` 阻止，不关、不弹确认框（`create-root.close.test.tsx` 一格，变异 `t2-mut-ime-escape-closes-r2`）。
- 拒绝：名字在本地被拒与 nerve 点名 `name` 的拒绝都放到名字下、焦点给它，别的拒绝经提示、表单留着（R5；`create-root.name.test.tsx`；H8 (page) `h8-intake-submit-page.spec.ts` 的超长、空的、只有空格的三次与收集箱关掉之后的 409）。
- 换宿主迟到答：成功与被拒的跟进只在表单还挂着时做（R29；`create-root.form.test.tsx` 两格，变异 `t2-mut-follow-unmounted-r3`）。

收集箱部分关闭。留下：第三条里迭代、视图的命令面板结果与第五条里筛选、视图写后关闭，随 P6a/P6b。保持 open。

**处理结果（M7/P6a，命令面板的视图与视图的关闭政策）：** 上面第三条的视图结果与第五条的项目视图部分逐项（[P6a spec](../specs/P6a-web-project-views.md) §2.4、§2.7、§2.8、§5）：
- 命令面板的视图（第三条）：按真实接口恢复——“打开视图”的菜单以列表页的键读（`power-k/ui/pages/open-entity/project-views-menu.tsx`），在途、被拒、空、门关着各有交代；搜索的 Views 组只在项目范围、store 里没有这个项目的视图才读、本地过滤（`power-k/ui/modal/use-view-search.ts`）；命令面板不再读 `IProjectView`（R13；单测 `project-views-menu.test.tsx` 7 格、`use-view-search.test.tsx` 10 格；H20 (page) `e2e/stories/collaboration/h20-views-palette-pages.spec.ts:32`、`:130` 三引擎）。
- discard 与文档 Escape：视图弹窗的文档级 Escape 删去，Escape、点背景、Cancel 在不在途时一条规则（T5，变异 `t5-mut-document-escape`；H12 (page) `e2e/stories/collaboration/h12-project-views-list-page.spec.ts:196`）。
- dirty/pending/busy：视图弹窗没有编辑器、没有放弃确认（同 M6 的迭代、模块弹窗），在途不关、Cancel 停用（T5，`t5-mut-close-in-flight`）；“更新视图”在途时视图页上改草稿的入口停用（R29，T8；H12 (page) `h12-project-view-page.spec.ts:233`）。
- 输入保留：被拒时表单与输入留着（T5，`views/modal.test.tsx`）。
- 嵌套 Escape：图标选择器开着时 Escape 只关它（T5；H20 (page) `e2e/stories/collaboration/h20-view-pages.spec.ts:282` 三引擎）。
- IME：组字时的 Escape 由 `ModalCore` 阻止（T5 一格，变异 `t5-mut-ime-escape-closes`）；Enter 不另写。
- 拒绝：名字的 `FieldError`、`aria-invalid` 与焦点，nerve 点名 `name` 时放到名字下，别的拒绝经提示（T5，变异 `t5-mut-no-refused-focus`、`t5-mut-invalid-off`、`t5-mut-name-refusal-toast`；H12 (page) `:196`）。
- 换宿主迟到答：弹窗与宿主的跟进只在还挂着时做（T4、T5，`t5-mut-follow-unmounted`）；“更新视图”的跟进只在 HOC 还挂着时做（T8，`t8-mut-update-follow-unmounted`；H20 (page) `:76`、`:214`）。
- 未等待的筛选与视图写后关闭：“更新视图”等 nerve 的回答，失败时按钮留着、草稿与基线不动（R9，T8，`t8-mut-update-resolves-on-failure`；H12 (page) `:233`）。

项目视图的部分关闭。留下：工作区视图的部分（工作区 HOC 的写后关闭、工作区视图的弹窗）随 P6b。保持 open。

**处理结果（M7/P6b，工作区视图的部分）：** 第五条的工作区一半与第三条的工作区视图（[P6b spec](../specs/P6b-web-workspace-views.md) §2.9、§2.11、R10、R14、R19）：
- 工作区 HOC 的写后关闭：`filters-hoc/workspace-level.tsx` 去掉 Plane 的分支（`useGlobalView`、锁、所有者、`rich_filters` 的体与英文提示，208 → 90 行），“更新视图”经共用的 `use-view-update.ts` 等 nerve 的回答，失败时提示、按钮留着、草稿与基线不动，请求体在轮到时从草稿算；跟进只在 HOC 还挂着时做（`t10-mut-update-unmounted`；H20 (page) `e2e/stories/collaboration/h20-workspace-view-pages.spec.ts:203` 的离开之后与换账号之后答到）；在途时改草稿的入口停用（`t10-mut-busy-not-passed`；H13 (page) `h13-workspace-view-page.spec.ts:73` 扣住回答的一步）。
- 工作区视图的弹窗：用 P6a 的弹窗（按容器），discard 与文档 Escape、dirty/pending/busy、输入保留、嵌套 Escape、IME、拒绝、换宿主迟到答都同上面 P6a 的那一组，工作区的格在 `views/modal-workspace.test.tsx`、`actions-host-workspace.test.tsx`；H14 (page) `e2e/stories/collaboration/h14-workspace-views-list-page.spec.ts:129`（空白与超长被拒、焦点）、H20 (page) `:299`（Tab 次序、嵌套 Escape）。
- 第三条的视图结果：工作区视图照 R19 不进命令面板（四条导航命令照旧，不加工作区视图的菜单、搜索与新建命令；Plane 的搜索有工作区范围，见 plane-diff 的“命令面板的视图（网页，M7/P6a）”一行）。

工作区视图的部分关闭。第三条里迭代的命令面板结果不在 P6b 的范围，M7 收官时逐条核对。保持 open。

<a id="M7-05"></a>
## M7-05 最后读者与全应用收口

`projectMemberIds` 现余三个真实读者：

1. `app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/mobile-header.tsx`。
2. `core/components/views/view-list-header.tsx`。
3. `core/components/inbox/inbox-filter/filters/filter-selection.tsx`。

最后一个改成明确项目参数后，同时删除 IProjectMemberStore 声明、computed 注解、class getter 及对应旧测试；保留显式 `getProjectMemberIds`。读者原件见 [P16 inventory](../../M4-issue-core/reviews/P16-web-cleanup-inventory.md)。

`getTabIndex` 共余 7 文件、4 组：M6 cycles/form、modules/form；M7 views/form 与 inbox create-root / issue-properties / issue-title / issue-description。M6/M7 最后消费者改完删除函数、两份 `tab-indices.ts`、barrel/枚举/映射；保留菜单正常 -1。对应责任与日期/项目共用件验收见 [M6-04](../../M6-cycles-modules/handoffs/M4-closeout.md#M6-04)。

**何时验收：** 最后一个消费者的阶段，不等无期限收尾；结果双向回写 M6/M7。新页面同时遵循 ModalTitle、FieldError/useRefusedFocus、嵌套两级 Escape、带行名控件、按钮/菜单角色和翻译文字；M8 只做全应用最后核验。

**处理结果（M7/P4，通知页的部分）：** 通知页照这些规则写（[P4 spec](../specs/P4-web-notifications.md) §2.6、§2.7、R16）：自定义暂缓弹窗的标题经 `ModalTitle`；它的两个字段的提示是 `FieldError`，控件以它为描述并带 `aria-invalid`，第一个被拒的经 `useRefusedFocus` 取得焦点；卡片的动作以那一行命名（“Mark WEB-12 Fix login read”“Archive …”“Snooze …”）；页签是 Headless `Tab`，类型筛选与视图是 `PopoverMenu` 里的 `<button aria-pressed>`，暂缓的预设是菜单项，“加载更多”是按钮；文字全部经 `t()`，en 与 zh-CN 同加。Escape 关上两个菜单、暂缓菜单与弹窗并把焦点还给按钮，在途时弹窗关不掉（H20 (page) 三种引擎、`snooze-modal.test.tsx`、`snooze-menu.test.tsx`）；弹窗里的日历或时间列表开着时的两级 Escape 没有单独的故事。`projectMemberIds` 的三个读者与 `getTabIndex` 的七个文件 P4 都没有改到，随 P5、P6a。保持 open。

**处理结果（M7/P5a，收集箱的部分）：** `projectMemberIds` 的收集箱读者 `inbox/inbox-filter/filters/filter-selection.tsx` 随旧筛选树在 T6b 删除（`2ac1928c`），新的负责人与创建者筛选读路由项目的显式 id（`getProjectMemberIds(projectId, true)`，`inbox/filters/menu.tsx:106`，T7）；现余的两个读者都是视图的（上面 1、2），getter 的删除随 P6a（[P5a spec](../specs/P5a-web-intake.md) R23）。`getTabIndex` 的四个收集箱文件都在新建弹窗里，随 P5b。收集箱页照这些规则写（spec §2.7–§2.9、R35）：暂缓与选择器的弹窗标题经 `ModalTitle`（`inbox/modals/snooze-issue-modal.tsx:82`、`select-duplicate.tsx:98`），拒绝与删除是 `AlertModalCore`；暂缓的日子被拒时提示是 `FieldError`（`:104`），经 `useRefusedFocus` 取得焦点（`:59`）；列表的一行以“编号 名字”为名（`inbox/sidebar/list-item.tsx:59`）；页签是 Headless `Tab`；面包屑、上一条与下一条的名字经 `t()`，en 与 zh-CN 同加。H20 (page) 三种引擎核对：每个分诊弹窗以标题为名、焦点在里面、后面的页面隐藏，Escape 或取消关上之后焦点回打开它的按钮，在途时 Escape、点外面与取消都关不掉。保持 open。

**处理结果（M7/P5b，收集箱的新建弹窗）：** `getTabIndex` 的四个收集箱文件离开它（`fa78bb11`），它只剩视图的 `views/form.tsx`，函数、两份 `tab-indices.ts` 与桶文件的导出随 P6a。新建弹窗照这些规则写（[P5b spec](../specs/P5b-web-intake-files-embed.md) §2.1）：标题经 `ModalTitle`（弹窗以 “Create intake work item” 为名）；名字的提示是 `FieldError`、控件 `aria-invalid`，被拒经 `useRefusedFocus` 取得焦点；确认放弃是 `AlertModalCore`（以标题为名，按钮的文字经 `t()`）；“Create more” 是开关本身，按钮里不再套开关；英文改走已有的键，新键只有确认框的两个（en、zh-CN）；两级 Escape 在 H8 (page) 三引擎核（负责人列表、截止日的日历与确认框开着时 Escape 只关它们）。`projectMemberIds` 的两个读者仍都是视图的。保持 open。

**处理结果（M7/P6a，两项都关闭）：** `projectMemberIds`：视图的两个读者（上面 1、2）改读 `getProjectMemberIds(projectId, true)`（T4 `bd2cfff0`），重新核过读者之后删除 `IProjectMemberStore` 的声明、`computed` 注解、类的 getter 与只为它的 `routerStore`，专属的旧测试断言随之改；显式的 `getProjectMemberIds` 保留（[P6a spec](../specs/P6a-web-project-views.md) §2.9、R16；[M3 交接](M3-closeout.md) §7）。`getTabIndex`：视图表单离开它时是最后的读者，函数、`ETabIndices`、`TAB_INDEX_MAP`、两份 `tab-indices.ts` 与桶导出删去（T5 `5a58db86`，R15；[M3 交接](M3-closeout.md) §6）；菜单正常的 `-1`（`issues/issue-layouts/spreadsheet/columns/header-column.tsx:48` 的 `customButtonTabIndex={-1}`）与共用控件非正的 `tabIndex?` 保留。视图的页面照这些规则写（§2.3、§2.4）：弹窗标题经 `ModalTitle`；名字的提示是 `FieldError`、控件 `aria-invalid`，被拒经 `useRefusedFocus` 取得焦点；图标选择器开着时 Escape 只关它；一行的菜单按钮以视图的名字为名；按钮与菜单是各自的角色；文字经 `t()`。M6 一侧的 M6-04 已在 M6 关闭（M6 的迭代、模块表单在 M6 离开 `getTabIndex`），这里回写 M7 一侧。
