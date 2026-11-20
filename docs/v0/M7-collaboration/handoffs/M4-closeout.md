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

<a id="M7-04"></a>
## M7-04 收集箱旧 DTO、版本与命令面板

- P10a 仅准确化 `services/inbox/inbox-issue.service.ts` 的既有输入/响应：create 描述为 `Partial<IssueDetail>`；嵌套响应为 `Issue | IssueDetail`，普通列表无描述；updateIssue 的旧请求输入为 `Partial<IssueDetail> & { skip_activity?: string }`，状态/查重/延后元数据保留自身 DTO。
- string flag 只属该旧 wire 输入，不进入 canonical 实体、分组、回滚；M4 新工作项接口只有 boolean。M7 接自身接口时删除旧地址、到期例外、最后旧 request type 与版本 DTO/service/`inbox/content/description-versions` 的过渡，不能给 M4 新接口恢复兼容层。
- 命令面板迭代、视图、收集箱结果按真实接口恢复；删除工作项时清掉最近访问，由 M7 扩自己的删除图/消费者。
- `frame-src` 仅在确实实现需 iframe 的功能时加具体来源、CSP 说明和测试；M4 原生 emoji 不需要它，不加宽泛放行。
- 待 lazy editor 就绪期间收集箱 discard/文档 Escape 的关闭政策、未等待筛选/视图写后关闭等，仍属 M7 页面验收；需统一 dirty/pending/busy、输入保留、嵌套 Escape/IME、拒绝与换宿主迟到答，不能借 M4 属性兼容宣称页面已修好。

**何时验收：** 相应 API/页面首次接通前；旧接口最后读者清除时同步类型、关键词、测试和本交接。

<a id="M7-05"></a>
## M7-05 最后读者与全应用收口

`projectMemberIds` 现余三个真实读者：

1. `app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/mobile-header.tsx`。
2. `core/components/views/view-list-header.tsx`。
3. `core/components/inbox/inbox-filter/filters/filter-selection.tsx`。

最后一个改成明确项目参数后，同时删除 IProjectMemberStore 声明、computed 注解、class getter 及对应旧测试；保留显式 `getProjectMemberIds`。读者原件见 [P16 inventory](../../M4-issue-core/reviews/P16-web-cleanup-inventory.md)。

`getTabIndex` 共余 7 文件、4 组：M6 cycles/form、modules/form；M7 views/form 与 inbox create-root / issue-properties / issue-title / issue-description。M6/M7 最后消费者改完删除函数、两份 `tab-indices.ts`、barrel/枚举/映射；保留菜单正常 -1。对应责任与日期/项目共用件验收见 [M6-04](../../M6-cycles-modules/handoffs/M4-closeout.md#M6-04)。

**何时验收：** 最后一个消费者的阶段，不等无期限收尾；结果双向回写 M6/M7。新页面同时遵循 ModalTitle、FieldError/useRefusedFocus、嵌套两级 Escape、带行名控件、按钮/菜单角色和翻译文字；M8 只做全应用最后核验。
