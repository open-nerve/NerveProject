---
status: open
from: M3/closeout
to: M4
created: 2026-10-13
---

# M3 交给 M4：事件和连带、加锁约定、权限规则、状态和标签的数据、工作项页面上留下的前端

M3（工作区与项目）做完了工作区、成员、邀请、项目、项目成员、状态、标签、个人显示设置、权限框架和这些页面（[M3 设计](../../M3-workspace-project/M3-design.md)）。下面是 M3 定下的约定第一次落到工作项上的地方，和 M3 改到、要由 M4 接上的代码。每一节来自 M3 设计 13.2 的 M4 一行，以及各 Phase 的 spec 第 5 节、评审第 6 节交给 M4 的各条和 M3 收尾找到的（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明每一条的来处）；第 9 节是 M2 收尾交接第 12 节的原文。为什么这样定，看链接的节；这里写做什么、代码在哪、M3 做到哪一步、怎样算完。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. 事件与删除工作区的连带

- **做什么**：M3 没有领域事件，删除工作区、降为访客、结束成员关系的连带经 `ProjectCascade` 同步完成（M3 设计 3.3）。M4 随第一个异步订阅者引入事件（M2 设计 3.15、M4 的 [M2 收尾交接](M2-closeout.md)第 6 节不变），届时可以把删除工作区的连带改挂到 `WorkspaceDeleted` 上，并为工作项加上删除项目、删除工作区的连带。
- **不能改成异步的一步**：邀请的软删除留在删除工作区的事务里。注册时的邀请检查（`SignupInvitations`）只看邀请行、不看工作区，连带改为异步之后，从删除到任务运行之间，已删除工作区的邀请链接会让被邀请的邮箱在注册关闭时注册；要么邀请这一步保持同步，要么让这个检查也读工作区（M3/P3 评审第 6 节，F-M7）。
- **由请求入队的任务**：`server/internal/bootstrap/table_rows_test.go` 的 `registrationRows` 排除 River 的表（选主、定时任务在测试期间写它们）。有了由请求入队的任务之后，"被拒的注册不写任何表"要另外核对 `river_job`，例如按任务种类或参数（M3/P3 评审第 6 节，R2-M3）。
- **`platform/jobs` 的测试的余量**（M3 收尾 Task 14 的评审）：M3 收尾把这个包的测试中时间的余量改成了期限（K15，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.13；M3/P7a 评审第 6 节的 FW-a：固定的余量在机器忙时偶尔不够），同一类的两处留着：`server/internal/platform/jobs/jobs_test.go:265`（`TestStartWithoutADatabase`，`Stop` 在 500 毫秒之内返回）和 `runner_test.go:136`（`TestStopEndsTheAttemptsToStart`，100 毫秒）。River 归 M4（[M2 收尾交接](M2-closeout.md)第 1 节），M4 改 `platform/jobs` 时照 K15 把它们改成期限：分得开"立即返回"和"等过了重试的间隔"（`firstRetry`，1 秒）就够。
- **关闭条件**：M4 设计写明连带同步还是异步；改成异步时，邀请一步的选择和它的测试；`registrationRows` 的核对覆盖新的任务；M4 改到 `platform/jobs` 时，上面两处是期限。

## 2. 加锁的约定和方案 E

- **约定**（M3 设计 3.6、11.2，总体设计 4.2）：写事务先锁父行再判定；全局加锁顺序；成员关系集合的增长与收缩（约定六，以后加入新的成员关系时照它写，恢复"不比新授予给得更多，也不比原来那一行更多"）；共享锁之下的批量插入按唯一键排序（约定五）和它的例外：一个批量写的行若有别的写不持父行的锁也能写到，先按 `id` 锁住、再只写锁住的那些（M3/P6 的裁定 G-1；今天只有停用按地址删除的邀请）；账户行只作第一把锁（约定一）。M4 的批量写（工作项、评论等的连带）逐个照约定五核对。
- **方案 E 的范围**（M3/P4b 整分支评审的 I2，负责人 2026-10-02）：工作项的写是否最先取工作区行的 `FOR SHARE` 由 M4 的设计决定，不默认沿用约定二。取的话，新的 `FOR SHARE` 越过等待中的 `FOR NO KEY UPDATE`，高频的工作项写入可以让工作区一级的写（修改、删除工作区，改角色，接受邀请，移出、离开、恢复成员，停用）一直等到请求期限，M4 要说明怎样不被饿死，或证明在它的负载下不会；不取的话，要说明工作项一侧的行与工作区一级的连带怎样不出现 F-M3 那样的时刻倒退（M3 设计 3.3）。另算进去：先锁后判定让任何已认证的账户在得到 404 之前短暂取得一个工作区的 `FOR NO KEY UPDATE`（离开只要 slug，移出要一个成员关系的 id），方案 E 之下每一次都短暂挡住这个工作区的项目级的写，限速约束它的频率（M3/P5a、P5b 评审第 6 节）。
- **新的成员关系增长**：工作项若加入新的增长（例如指派时自动加入项目），照约定六与停用串行（项目一侧的增长取工作区的 S），并加一条与停用的交错测试（M3/P6 评审第 6 节）。
- **物理删除账户**：`workspace_members.member_id`、`project_members.member_id`、`project_user_properties.user_id` 的 `ON DELETE CASCADE` 会不经工作区的锁删除成员关系和显示设置。M4 的清理若物理删除账户，先锁工作区，或写明为什么不需要（M3/P2 评审 T11 C2、P4a 评审 F-M4）。
- **关闭条件**：M4 设计写明方案 E 的选择和代价；每个新的批量写按约定五核对；每条新的增长路径有与停用的交错测试。

## 3. 权限规则和错误码

- 规则表在 `server/internal/modules/access/domain/rules.go`，操作名常量在各模块的 `domain/actions.go`，`bootstrap` 的完整性测试核对两者一一对应（M3 设计 3.4）；每个操作在权限矩阵里有一行，写法照 M3 设计 9.2（逐格，含已不是成员的身份）。
- 错误码前缀的规则（M3 设计 11.7）：带前缀的码，前缀是 nerve 的一个模块；平台码 `forbidden` 用于"看得到而角色不允许"。声明在一个模块上的跨模块码由这个模块自己的 HTTP 测试返回过（`apitest.Main` 两个方向，M3 设计 9.4）。
- 规则表的 `AllowCreator`（"创建者本人可以修改或删除自己创建的对象"）随第一条用到它的规则加入；`guest_view_all_features` 对工作项的约束（M3 设计 3.4）。
- 决策点 3 的规则行：访客看不到别人的个人主页上的工作项列表（M3 设计 9.2 的那一格）；页面一侧（个人主页的工作项一栏和访客看到的筛选工具栏，`user-issues` 的地址今天答 404）见第 6 节。
- **关闭条件**：工作项的每个操作在规则表、操作名、矩阵中各有一行；完整性测试和 `apitest.Main` 通过。

## 4. 状态、标签和项目的数据

- **状态**：删除状态之前检查它下面的工作项（"状态下还有工作项时不能删"，M3 设计 3.17）；新工作项取项目的默认状态（`states."default"`，每个项目至多一个），那时 `State.default` 的说明补上这一句（M3/P7a spec 第 3 节第 7 条）。
- **标签**：工作项带标签（`issue_labels`，部分唯一 `(issue_id, label_id)`）；定下删除标签时工作项上的标签怎么办；`issue_labels` 的写如果不持有项目或工作区的 `FOR NO KEY UPDATE`，就落在约定五的例外里，`DeleteLabel` 的一条语句改为按 `id` 先锁；任何新的设 `parent_id` 的路径再核对两层和已删除的父标签的规则，"给已有父标签的标签换父标签"要有自己的拒绝行（M3/P7b 评审第 6 节）。
- **项目**：工作项编号取 `projects.last_issue_sequence`（`UPDATE … SET last_issue_sequence = last_issue_sequence + 1 … RETURNING`，M3 设计 4.6）。`Project` 的 `identifier`、`default_assignee_id`、`guest_view_all_features`、`archive_in` 四个字段的说明现在只说 nerve 存了什么（M3 收尾，控制者的裁定 A1）：工作项读它们之后，说明改成它们那时的作用（编号的前缀、新工作项的默认负责人、访客看到的工作项、自动归档）。
- **默认负责人照设定保留**（M3 收尾 Task 14 的评审）：`projects.default_assignee_id` 设定时必须是项目的有效成员、不是它的访客（`updateProject` 的 `not_allowed`），之后照设定保留：那个账户离开项目、成为项目的访客或被停用时，没有什么清掉它（只有物理删除账户时，外键的 `ON DELETE SET NULL`），所以契约的说明是"kept as set"（`api/modules/project.yaml` 的 `Project.default_assignee_id`）。M4 在创建工作项时决定怎么用它，例如只在它仍是项目当前的、不是访客的成员时才指派，并写进 M4 的契约（创建工作项的说明）。
- **表和删除**：工作项引用状态、标签、项目的外键在 M3 设计 4.12 的图上延伸；60 天清理包括 M3 的表，指向 `workspaces`、`projects`、`labels.parent_id` 的外键已有不带条件的 `*_idx`，工作项的表照做。新表带 `workspace_id`，否则按目录的删除守卫拒绝它（M3/P4a 评审 F-M1）；端到端的 `workspaceTables`（`e2e/fixtures/assert/workspace.ts`）加上工作项的表，并考虑像 `expectProjectDeleted` 那样的目录自查；`expectProjectDeleted` 对有 `project_id` 而没有指向 `projects` 的外键的新表失败（有意的），两个列表保持 `COLLATE "C"`（M3/P7b 评审第 6 节）。
- **关闭条件**：M4 设计写明删除状态、标签时工作项的去向；4.12 的图延伸到工作项的表；端到端的删除断言含工作项的表；创建工作项的契约写明默认负责人已不是项目当前的、不是访客的成员时怎么办。

## 5. 显示设置与旧的 service

- 工作项列表的筛选和显示列加在 `workspace_user_properties`、`project_user_properties`（M3 设计 3.18），格式由 M4 定；旧 `ProjectService`（`web/apps/web/core/services/project/project.service.ts`）的 `/user-properties/` 两个方法和关键词守卫的例外（`until: "M4"`）随之删除；今天工作项页面上 `user-properties` 的请求答 404（M3/P8b spec 第 5 节和附录 A.5），打开工作项页面的故事（P2、P8、S7）经 `e2e/fixtures/mounts.ts` 的 `workItemFilters` 允许这个请求、它的 404 和浏览器的报错，M4 改筛选时删去它和故事中的允许。
- 旧 `ProjectService` 的 `projectIssuesSearch` 和它的三个调用方（M4 的选父工作项、M6 的添加已有工作项、M7 的收集箱查重）随 M4 的工作项搜索接口替换，之后删除旧 service（M3 设计 7.3）。旧 `WorkspaceService`（`web/apps/web/core/services/workspace.service.ts`）的视图、草稿、搜索方法同样随各自的接口改接或删除。
- 工作区搜索的类型和参数：`@nerve/types` 的 `workspace.ts` 24 行、`project/projects.ts` 9 行，`workspace.service.ts` 3 行（`domains.mjs --rows M3` 列出的 36 行，属于 M4，M3/P11 spec 附录 A.9）。
- 活动类型中的 `is_bot`（`@nerve/types` 的 `issues/activity/base.ts` 的 `TIssueActivityUserDetail`、`comments/card/display.tsx`）：Nerve 不区分人和机器人，删除（M2 收尾交接第 11 节留下的两处）。
- **关闭条件**：`/user-properties/` 的例外从 `tools/keywords.json` 删掉，`workItemFilters` 随之删掉；旧 `ProjectService`、`WorkspaceService` 中工作项的方法都已改接或删除；`is_bot` 不再出现在 `web/` 里。

## 6. stores、会话和页面的取数（总体设计 7.7）

- `issue/root.store.ts` 的 `autorun` 不在 action 之外写 `router` 的副本：把 `workspaceSlug`、`projectId` 等做成 `rootStore.router` 上的计算属性，或改成 `reaction`，由 `RootStore.dispose()` 调用它的释放函数（在那之前 `store-context.test.ts` 有两条 MobX 严格模式的警告，M3/P8a 评审第 6 节）。工作项的分页、分组列表扩展 `core/lib/reconciled.ts`，不另写一份。
- **按需的取数按权限启用**（M3/P8b spec 第 5 节）：状态、成员、标签的下拉框打开时取（`dropdowns/state/dropdown.tsx`、`dropdowns/member/dropdown.tsx`、`issues/select/dropdown.tsx`、`issues/issue-layouts/properties/label-dropdown.tsx`、`issues/issue-detail/label/select/label-select.tsx`），子工作项的 store 取别的项目的状态、成员、标签（`store/issue/issue-details/sub_issues.store.ts`，调用者可能不是那些项目的成员）。被拒绝时：状态下拉框的加载一直不结束、是未处理的拒绝，成员下拉框是未处理的拒绝，工作项 store 不等结果的取数（`issue.store.ts` 取状态的两处、`sub_issues.store.ts` 取状态、成员、标签）是未处理的拒绝，三个标签选择器的取数被拒绝时一直"加载中"或被吞掉（Plane 原有，M3/P8b 评审的 P9）。
- 错误经文案表（`errorMessageKey`）：工作项的创建、修改读 `error?.error` 的两处（`issues/issue-modal/base.tsx`，M3/P9 spec 第 5 节）。
- 工作项页面上的标签新建（`issues/` 下的 `issue-detail/label/root.tsx`、`issue-detail/label/select/root.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`select/dropdown.tsx`、`select/base.tsx` 直接调 store 的 `createLabel`）改经 `useLabelChanges().create`（会话的核对、名称已被占用在名称下），之后可以加关键词规则（M3/P11 spec 第 3 节第 13 条）。
- 工作区一级显示的工作项不显示没有取过标签的项目的标签（M3/P8b 删了为每个项目取标签的工作区取数，M7 加回）；`fetchParentStats` 不再重取项目，工作项的修改若动了项目的字段，加一个不等的重取；工作项的键宽读回 `next_work_item_sequence`，`TIssueIdentifierProps.projectIdentifier` 在项目未到时可以是 `undefined`（M3/P8b spec 第 5 节）。
- 个人主页的工作项一栏（成员和管理员；`/api/workspaces/{slug}/user-issues/…` 今天答 404）和访客看到的筛选工具栏（布局、筛选、Display 的控件在"没有权限"之上，M3/P11 评审 C10）。
- M3 页面上指向工作项页面的链接接上之后，这些页面进入 `watchPage` 的范围（M3 设计 3.1）；S2 的 `REQUESTS` 是一份要维护的名单，在挂载路径上加取数时同时改它（M3/P11 spec 第 6 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，组件状态里的弹窗和状态不带过去。store 持有的弹窗开关由渲染它的宿主在卸载时关上（`useLayoutEffect` 的清理，像 `WorkspaceLevelModals`）；宿主在 key 之上时还要在范围结束时关上。工作项的弹窗、选中、草稿等属于一个工作区或项目的界面状态放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）；要在切换之间留着的，写明为什么。
- **工作项级弹窗的宿主**（M3 收尾；控制者的裁定：store 持有的弹窗开关属于渲染它的宿主，由宿主结束）：`modals/work-item-level.tsx` 的 `WorkItemLevelModals`（创建工作项 `isCreateIssueModalOpen`、删除工作项 `isDeleteIssueModalOpen`）。开关在命令面板的 store（`web/apps/web/core/store/command-palette.store.ts`）里，宿主由 `power-k/projects-app-provider.tsx` 渲染，在 `(projects)/layout.tsx` 和 `(settings)/layout.tsx` 的 `ProjectsAppPowerKProvider` 里，在项目的 key 之上：换项目（后退、前进也是）时开着的弹窗留着，作用于路由新指向的项目。M4 接上工作项页面时两件都做：把宿主移到项目的 key 之下，或在项目的范围结束时关上它的开关；并像 `modals/workspace-level.tsx` 的 `WorkspaceLevelModals` 那样在卸载时关上它渲染的每个开关（`useLayoutEffect` 的清理，在下一个页面绘制之前）。加一个像 P3、P8 那样的后退检查（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）。
- **工作项详情 store 的弹窗开关和浮出的工作项**（M3 收尾的修复轮，最终评审的 m1；同一条裁定）：`web/apps/web/core/store/issue/issue-details/root.store.ts` 的 `peekIssue`（:126）和 8 个弹窗开关（:143–150：`isCreateIssueModalOpen`、`isIssueLinkModalOpen`、`isParentIssueModalOpen`、`isDeleteIssueModalOpen`、`isArchiveIssueModalOpen`、`isRelationModalOpen`、`isSubIssuesModalOpen`、`attachmentDeleteModalId`）在根 store 上，不随工作区、项目、工作项的 key 重来：换工作区、项目或工作项（后退、前进也是）时开着的留着，作用于路由新指向的那一个。渲染、结束它们的宿主有四个：`issues/issue-detail-widgets/issue-detail-widget-modals.tsx`（新建子工作项、链接、关系、子工作项的四个开关）、`issues/issue-detail/parent-select.tsx`（`isParentIssueModalOpen`）、`issues/peek-overview/view.tsx`（`peekIssue`：浮出层由 `peek-overview/root.tsx` 照它渲染，`view.tsx` 以 `setPeekIssue(undefined)` 关上它，并读 `isAnyModalOpen`）、`issues/attachment/attachment-item-list.tsx`（`attachmentDeleteModalId`）。`isDeleteIssueModalOpen`、`isArchiveIssueModalOpen` 没有照它们渲染的弹窗：子工作项和关系的一行（`issue-detail-widgets/sub-issues/content.tsx`、`sub-issues/issues-list/list-item.tsx`、`issue-detail-widgets/relations/content.tsx`、`issues/relations/issue-list-item.tsx`）设 `isDeleteIssueModalOpen`，弹窗是那些组件自己的状态，store 的 `isAnyModalOpen` 读它让浮出层在弹窗开着时不关；`isArchiveIssueModalOpen` 没有谁设它（M4 删除它和 `isAnyModalOpen` 中读它的一处）；浮出层的删除、归档是 `view.tsx` 自己的状态，同名而不是这两个开关。同一个根 store 还存着这些弹窗的数据，也由同一个宿主结束（M3 收尾的再评审的 m4）：`issueCrudOperationState`（:129–140，新建子工作项、添加已有工作项的弹窗各自的 `toggle` 和 `parentIssueId`，`issue-detail-widget-modals.tsx` 照它和 `isCreateIssueModalOpen`、`isSubIssuesModalOpen` 渲染两个弹窗）、`issueLinkData`（:128，正在编辑的链接：`issue-detail/links/link-item.tsx` 设它，`create-update-link-modal.tsx` 打开时预填它，只有这个弹窗自己关上时清掉）和 `relationKey`（:127，关系的弹窗选的种类）。宿主只关上开关、不重设这三个，下一次在别的工作项上"添加链接"就预填旧的链接，留下的 `create.toggle` 带着旧的父工作项随下一次 `isCreateIssueModalOpen` 又打开。M4 照上一条做：每个宿主在卸载时关上它渲染、设的开关，并重设 `issueCrudOperationState`、`issueLinkData`、`relationKey`（`useLayoutEffect` 的清理，在下一个页面绘制之前），`peekIssue` 由浮出层的宿主在卸载时清空；加一个后退的检查（开着一个弹窗或浮出的工作项，后退到另一个项目或工作项之后它不再开着）。
- **嵌套的 id 没有 key**（M3 收尾）：工作项的页面 `projects/(detail)/[projectId]/issues/(detail)/[issueId]`、`archives/issues/(detail)/[archivedIssueId]`、`browse/[workItem]` 和个人主页 `profile/[userId]`（连同 `[profileViewId]`）只在工作区、项目的 key 之下，换一个工作项或一个人（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M4 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：上面每一处的拒绝有提示或交回调用方，没有未处理的拒绝（vitest 或故事）；`autorun` 随退役的一代释放，`store-context.test.ts` 没有严格模式的警告。

## 7. 工作项页面的键盘、下拉框和弹窗

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：弹窗打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉、取消不能用。工作项的弹窗在 M3 的故事里走不到：M4 对接时逐个核对 Tab 在弹窗里转、Escape 和点外面关上、关上之后焦点回到打开它的控件，以及弹窗里经 portal 渲染的东西（日期选择、下拉框、表情）能用（收尾 spec 附录 A.3 的门户清单只证明了 M3 页面上的几种）。`AlertModalCore` 的确认框照 A-M2：被拒绝时留着，只在成功的跟进中关上；按钮的文字经 `t()` 传入（`@nerve/ui` 的默认值是英文，见 M8 的交接）。
- **弹窗的名字**：Headless UI v2 的 `Dialog` 只以它的 `DialogTitle` 为名（`aria-labelledby`），普通的标题不给弹窗名字，读屏读出一个没有名字的对话框；M3 收尾的修复轮在 `@nerve/ui` 的 `ModalCore` 旁边导出 `ModalTitle`（它就是 `DialogTitle`，`h3`），`AlertModalCore` 的标题经它，M2、M3 的弹窗的标题都经它（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）。M4 的确认框（`AlertModalCore`）因此都有名字；别的弹窗的标题改经 `ModalTitle`，没有看得见的标题的给一个 `sr-only` 的（不加第二个看得见的），标题的文字经 `t()`。以普通标题的：`issues/archive-issue-modal.tsx:73`、`issues/confirm-issue-discard.tsx:40`、`issues/issue-detail/links/create-update-link-modal.tsx:82`、`issues/issue-modal/form.tsx:267`（创建、编辑工作项的弹窗 `issue-modal/base.tsx`）；没有标题的：`core/description-versions/modal.tsx`、`issues/parent-issues-list-modal.tsx`（搜索的列表）。共用的自定义日期的弹窗（`core/filters/date-filter-modal.tsx`，9 个调用方，M3 的项目列表的筛选的"自定义"在内）M3 收尾已在根上改（修复轮的后续一轮，再评审的裁定 4b）：以调用方给的 `title` 为名（`sr-only` 的 `ModalTitle`），下拉框的三种、范围的说明和两个按钮经 `t()`（`date_filter.*`、`cancel`）。M4 的两个调用方（`issues/issue-layouts/filters/header/filters/start-date.tsx`、`due-date.tsx`）传的 `title` 是写死的英文（"Start date""Due date"），它就是弹窗的名字和下拉框的那句里的日期：改经 `t()`。M4 的故事打开它们时断言 `expect(modalDialog(page)).toHaveAccessibleName(…)`。
- **正的 `tabIndex`**：工作项的表单（`getTabIndex(ETabIndices.ISSUE_FORM)`）：浏览器先走完正的、再走焦点陷阱的哨兵，Tab 在最后一个之后离开弹窗，M3/P10 的创建表单因此删去了它的，M4 照样删去。同一个机制的另几处（M3/P10 的裁定 F-17、F-18）：`core/modals/existing-issues-list-modal.tsx`、`issues/parent-issues-list-modal.tsx` 的搜索框和 `issues/issue-detail/label/select/label-select.tsx` 收的 `baseTabIndex`；`issues/issue-modal/components/default-properties.tsx` 给父工作项的 `CustomMenu` 正的 `tabIndex`（M3 收尾起落在菜单的按钮上，原来在 Menu 的根上）；`issues/archive-issue-modal.tsx`、`core/description-versions/modal.tsx` 写死的 `tabIndex={1}`。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个，同时删除 `getTabIndex`（`@nerve/utils` 的 `tab-indices.ts`）和 `@nerve/constants` 的 `tab-indices.ts`，`CustomMenu` 不再读 `tabIndex`（M3/P10 评审第 6 节）。
- **嵌套的按钮**：`dropdowns/` 的 `DropdownButton` 放在 Headless UI 的按钮里（M3/P10 的裁定 R3），面包屑的下拉框之前一节的标题按钮也在选择的按钮里，M4 随工作项的属性下拉框和页头一起改。菜单自己的按钮里又放了 `IconButton` 的三处（M3 收尾找到，`customButton` 或 `PopoverMenu` 的 `button` 中的 `IconButton`，两个 Tab 停留点、按钮里有按钮）：`comments/quick-actions.tsx`、`issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx`、`issues/issue-detail/issue-activity/activity-filter.tsx`（外面的按钮 `outline-none`，焦点看不到）；改法照 M3 收尾的项目标签栏的菜单（`navigation/project-actions-menu.tsx`：`CustomMenu` 的 `customButtonClassName` 给菜单自己的按钮样式，内容是图标，不再嵌一个按钮）。
- **`CustomMenu` 的键盘**（M3 收尾 Task 8 的修复轮 1 在根上改：键盘打开、选择，Escape 关上，焦点回到按钮；[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9、附录 A.4 的调用方清单）：M4 的页面上留下的三处。工作项快捷操作的 `SubMenu`（`issues/issue-layouts/quick-action-dropdowns/` 的 `all-issue.tsx`、`cycle-issue.tsx`、`module-issue.tsx`、`project-issue.tsx`、`issue-detail.tsx`）的触发项是包着按钮的 `Menu.Item as="div"`，菜单上的 Enter 点的是 `div`，键盘打不开子菜单（原来也是）；看板、日历的卡片给快捷操作的 `customActionButton` 的"激活"底色只随指针的 click 切换，键盘打开菜单时不显示；`issues/issue-modal/components/default-properties.tsx` 给父工作项的菜单的 `customButton` 是一个 `<button>`，放在菜单的按钮里（按钮里有按钮）。在 M4 的页面上用键盘核对。
- **下拉框**：成员下拉框以 `CustomSearchSelect` 为底之后 M4 的调用方看得到的不同（按键由 Headless UI 处理，`WhileOpen` 记下打开时的回调，Escape 之后焦点在选择自己的按钮上，列表的新样子），在 M4 的页面上核对，C7 没有核对 M4 页面上的调用方（成员下拉框的 8 个文件、4 个筛选）。`issues/select/base.tsx` 的根与 M3 收尾改过的项目下拉框是同一种写法：调用方给根 `tabIndex`，根是一个没有名字的停留点；照收尾把 `tabIndex` 给里面的按钮。`getButtonStyling` 的两处（`rich-filters/add-filters/button.tsx`、`dropdowns/layout.tsx`）用键盘核对轮廓。
- **筛选的按钮**（M3 收尾 Task 12，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：`FiltersDropdown`（`issues/issue-layouts/filters/header/helpers/dropdown.tsx`）默认的按钮现在就是弹层的 `Popover.Button`：以 `title` 为名，说出是否展开（`aria-expanded`），Escape 之后焦点回到它，指针点开之后焦点留在它上面；M4 的 7 个调用方（3 个用默认的按钮）在收尾 spec 附录 A.4 逐个核对过，没有故事走到，M4 的筛选的故事落地时断言按钮的名字、`aria-expanded` 和 Escape 之后焦点回到它。
- **组字的判断只有一个**（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.11）：关键词规则 `composing-check` 看住 M4 的键盘处理：一个按键是不是输入法的，只经 `@nerve/utils` 的 `isComposing`（`web/packages/utils/src/keyboard.ts`：`isComposing` 或 `keyCode` 229，Safari 只交出后者），不直接读 `.isComposing`，不拿 `keyCode` 与 229 比较。M4 今天的 4 处守卫（`issues/issue-layouts/properties/label-dropdown.tsx`、`issues/issue-detail/label/select/label-select.tsx`、`issues/select/base.tsx`、`web/apps/web/core/hooks/use-dropdown-key-down.tsx`）已经经它，没有故事走到它们，由规则看住；M4 新加的守卫照样。
- **右键菜单**：工作项和草稿的列表的右键菜单关着时 `inert`（M3/P10 的裁定 F-14），在 M4 的页面上核对右键打开它、点它的项。
- **层叠顺序**：页面上的元素停在弹窗层（`ModalCore` 的 `z-30`）之下，`z-30` 及以上留给弹窗和从弹窗打开、经 portal 渲染到 `body` 的东西（下拉框、提示、表情）；M3 收尾因此把设置页的手机菜单从 `z-50` 降到 `z-20`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.6）。工作项页面上画在任何弹窗之上的元素（行号是 M3 收尾时的）：`issues/issue-layouts/roots/project-layout-root.tsx:80`（`fixed … z-50`）；日历的两个 `Popover.Panel`（`issues/issue-layouts/calendar/dropdowns/options-dropdown.tsx:128`、`months-dropdown.tsx:102`，`fixed z-50`，不经 portal）；`issues/issue-layouts/quick-add/form/calendar.tsx:22`（`z-50`）；更新中的遮罩 `issues/issue-layouts/calendar/issue-block.tsx:110`、`issues/issue-layouts/list/block.tsx:217`（`z-[99999]`）；拖动中的 `issues/issue-layouts/kanban/block.tsx:261`（`z-[100]`）。与弹窗同层或之上的还有 `issues/attachment/attachment-item-list.tsx:123`（`z-30`）和拖动中的 `issues/issue-layouts/kanban/kanban-delete-drop-zone.tsx:52`（`z-40`）。M4 改这些页面时把它们降到 `z-30` 之下，或开着一个弹窗核对它们不画在弹窗上。
- 共用组件的焦点轮廓（`Button`、`IconButton`、`getButtonStyling`、`CustomMenu`）是否被 `overflow-hidden` 裁掉，在 M4 的页面上用键盘核对（M3/P11 评审第 6 节）。`IconButton` 的 `aria-label` 和 `EmojiPicker` 的 `ariaLabel` 自 M3 收尾必传，新加的每一个都有名字（编辑器标注块的选图标按钮是英文的"Callout icon"，编辑器的文字见 M8 的交接第 3 节）。
- **每一行的控件以它那一行为名**（M3 收尾的规则，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9：一行一个、每行都一样的控件以那一行的东西为名，同 F6 的项目一行、"{name} settings"、"Remove X"）：评论的菜单按钮 `comments/quick-actions.tsx:72`（"Options"）和每条评论的"Add reaction"（`comments/card/display.tsx:138`、`comments/comment-reaction.tsx:87`）每条评论都一样。以评论的作者为名：`display.tsx` 中已有的 `displayName`，另两处的 `comment.actor_detail?.display_name`；各一个带 `{name}` 的键（如 "Options for {name}'s comment"、"Add a reaction to {name}'s comment"），en 与 zh-CN 同时加。
- **字段的提示和被拒绝时的焦点**（M3 收尾的浏览器核对之后的修复，C11-1，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2 条）。
  - **M3 的写法**：M2、M3 的表单中字段的提示是 propel 的 `FieldError`（`@makeplane/propel/components/field`，`match`，字号由 `size` 给），放在字段的 propel `Field`（`invalid`）里，控件是 propel 的 `Input`、`TextArea` 或 `FieldControl`：Base UI 把提示的 id 放进控件的 `aria-describedby`，控件 `aria-invalid`。不是 Base UI 的控件由提示的 id 描述：`@nerve/ui` 的 `CustomSelect`、`CustomSearchSelect` 收 `describedBy`、`invalid`（按钮 `aria-invalid`，`fieldButtonProps`）。nerve 的拒绝经 `core/lib/form-refusal.ts` 的 `useFormRefusal`，显示在字段下（`setError`，不带 `shouldFocus`）或提示里。
  - **焦点只有一种写法**：`core/lib/refused-focus.ts` 的 `useRefusedFocus(attempt)`。表单的元素收它给的 ref（`<form>`，没有 `<form>` 的给包着字段的元素）；`attempt` 是 react-hook-form 的 `formState.submitCount`（自己记拒绝的表单给最后一次的拒绝，像登录表单）。每次尝试之后，元素里文档顺序第一个 `aria-invalid` 的控件取得焦点，两种原因（提交时没过规则、nerve 拒绝在字段下）都是；打字不移动焦点。`useForm` 写 `shouldFocusError: false`，`setError` 不带 `shouldFocus`：react-hook-form 自己的焦点在提示渲染之前。`Controller` 的 render 照旧把 `field.ref` 给输入框（`setFocus` 读它）。
  - **M4 的表单照做**：工作项的标题（`issues/issue-modal/components/title-input.tsx:85` 的一段）、链接的弹窗的地址（`issues/issue-detail/links/create-update-link-modal.tsx:112` 的一段在 `Field` 之外，没有链到输入框）；项目的选择（`issue-modal/components/project-select.tsx`）的规则没有提示，下拉框也不 `aria-invalid`，没过规则时焦点不到它上面；快速添加的标题（`issues/issue-layouts/quick-add/root.tsx`）没过规则时只有红框（`hasError`），没有提示可读，M4 照同一个写法给它提示或说明为什么不给。M4 的故事在被拒绝之后断言字段取得焦点、`toHaveAccessibleDescription(提示)`，像 P1、P6；并像 P1、A7 用 `recordFocuses`（`e2e/fixtures/browser.ts`）断言取得焦点的那一刻它已经 `aria-invalid`、以提示为描述。
  - **时机**（控制者 2026-10-15 的裁定，收尾 spec 2.17 第 2 条的"时机"一项）：焦点在 React 提交提示之后、还在那次提交引起的渲染之后给。propel 的 `FieldError` 在它自己的布局副作用里把 id 交给 `Field`，控件在随后的一次渲染才有 `aria-describedby`；`useRefusedFocus` 因此在布局副作用里排一个微任务，在微任务里给焦点。M4 不在 `setError` 里、也不在副作用里同步地给焦点（那一刻控件还没有描述），只用 `useRefusedFocus`。
- **弹窗里的列表**（C11-4，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 5 条）：Headless UI 的 `Listbox`、`Combobox`、`Menu` 的列表默认 `modal`，在弹窗里打开时后面的页面不再 `inert`、`aria-hidden`。`ModalCore` 提供 `useInsideModal()`（`@nerve/ui`），`CustomSelect`、`CustomMenu` 和 `dropdowns/date.tsx` 的 `DateDropdown` 在弹窗里 `modal={false}`（`DateDropdown` 的一行 M3 收尾已改）。在工作项的弹窗里打开的别的列表照样读它，它们的 `Combobox.Options` 没有写 `modal`：`dropdowns/state/base.tsx`、`dropdowns/priority.tsx`、`dropdowns/project/base.tsx`、`issues/issue-layouts/properties/label-dropdown.tsx`、`issues/select/base.tsx`、`issues/issue-detail/label/select/label-select.tsx`；`issues/parent-issues-list-modal.tsx` 的列表本身就是弹窗的内容，核对它打开时后面的页面隐藏着。M4 的故事在弹窗里打开它们时断言 `pageBehindHidden`（`e2e/fixtures/dialogs.ts`），像 A11、W4。
- **列表里不是选项的控件**（C11-2，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 3 条）：方向键只在选项、菜单项之间移动，列表里别的控件键盘到不了。M4 的：`issues/select/base.tsx:299` 和 `issues/issue-detail/label/select/label-select.tsx:201` 的"新建标签"按钮，`issues/parent-issues-list-modal.tsx:173` 选项里只在指针经过时显示的链接；另有 `issues/peek-overview/header.tsx:188`，`CustomSelect` 的 `customButton` 是一个 `<button>`（按钮里有按钮）。改法照 M3 收尾：自动归档的"Customize time range"是一个哨兵值的选项，项目页头的"更多"菜单每个动作是一个以它的标签页为名的菜单项。
- **两处字段的提示不是 M2、M3 的写法**（M3 收尾，浏览器核对之后）：工作项详情的标题 `issues/title-input.tsx:180` 的必填提示是一个红字的 `span`，在 `Field`（`:147`）之外，没有链到输入框，超长时只有 `:173` 的字数变红；筛选的日期范围 `rich-filters/filter-value-input/date/range.tsx:56` 不完整时只让 `DateRangeDropdown` 的按钮变红，没有提示。它们在 M4 的工作项和筛选的页面上（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2 条的第三次扫描找到）。照上面"字段的提示和被拒绝时的焦点"的写法改：提示是 `Field` 里的 `FieldError`，控件恰在提示显示时 `aria-invalid`；不是 Base UI 的按钮像 `CustomSelect` 那样由提示的 id 描述（`describedBy`、`invalid`）。
- **嵌在工作项弹窗里的两个弹窗只由 vitest 看住**（M3 收尾，浏览器核对之后）：工作项的弹窗 `issues/issue-modal/base.tsx:378` 的 `ModalCore` 没有 `handleClose`，里面打开"Save this draft?"（`issues/confirm-issue-discard.tsx:36`，从 `issue-modal/draft-issue-layout.tsx:113`）和选父工作项（`issues/parent-issues-list-modal.tsx:82`，从 `issue-modal/components/default-properties.tsx:300`），两个都有 `handleClose`。它们的 Escape 能关上，靠的是最里层的 `ModalCore` 认领一个 Escape（`@nerve/ui` 的 `modals/modal-core.tsx` 的 `claimedEscapes`）；M2、M3 没有这样的嵌套，只有 `modal-core.test.tsx` 的嵌套一行看住它（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17）。M4 接上工作项的弹窗时加一个故事：在里层的弹窗按 Escape，里层关上，工作项的弹窗留着。
- **描述编辑器自己的 Escape 监听也收到选父工作项的 Escape**（M3 收尾，浏览器核对之后）：`issues/issue-modal/components/description-editor.tsx:90` 的 `useKeypress("Escape", handleKeyDown)` 监听整个 `document`，调用表单的 `onClose`（`:79`）。所以关上选父工作项的同一个 Escape 也关上工作项的弹窗（没有改动时），或打开"Save this draft?"（有改动时）。这是 M4 的代码，与 Plane 相同，`ModalCore` 管不到（[收尾评审](../../M3-workspace-project/reviews/closeout-review.md)第 4 节，最后一轮的窄复审列出的嵌套的调用方第 5 行）。修法在这个监听里，由 M4 决定：例如不理会目标不在它自己的弹窗里的 Escape（Headless UI 把选父工作项的弹窗经 portal 渲染到工作项的弹窗的 DOM 之外），加一个故事核对。
- **关闭条件**：M4 的评审写明上面每一项的浏览器核对；`getTabIndex` 在最后一个删去正的 `tabIndex` 的 M 删除；上面层叠的每个元素降到 `z-30` 之下，或有开着弹窗的核对；M4 的表单的提示是字段的描述，被拒绝的字段在提示显示之后取得焦点（`useRefusedFocus`）；弹窗里的列表打开时后面的页面隐藏着。

## 8. 地址、复制、文案和留下的行

- **地址只从路由参数**：工作项的页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节 M4 一行列出：6 个，和裁定 F-12 补上的 16 个），M4 改成读路由参数，并把它们加进关键词规则 `project-address-route` 的范围。只读工作区的 4 个（`app/(all)/[workspaceSlug]/(projects)/drafts/header.tsx`、`issues/workspace-draft/empty-state.tsx`、`issues/workspace-draft/root.tsx`、`power-k/config/creation/command.ts`）改用 `useRouteWorkspace()`（`core/hooks/use-route-workspace.ts`）或路由的 slug，并加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）。
- **复制只经 hook**：工作项页面的 10 个文件（`issues/` 下的 `create-issue-toast-action-items.tsx`、`issue-detail-widgets/relations/helper.tsx`、`issue-detail-widgets/sub-issues/helper.ts`、`issue-detail/identifier-text.tsx`、`issue-detail/issue-activity/helper.tsx`、`issue-detail/issue-detail-quick-actions.tsx`、`issue-detail/links/link-item.tsx`、`issue-layouts/quick-action-dropdowns/helper.tsx`、`peek-overview/header.tsx`，和 `power-k/ui/pages/context-based/work-item/commands.ts`）改经 `useCopyLink` 或 `useCopyText`（`core/hooks/use-copy-link.ts`），失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 10 个例外（`until: "M4/closeout"`）随之删除。评论链接和工作项链接的复制今天没有处理拒绝（M3/P11 spec 附录 A.6）。
- **oxlint**：`issues/issue-detail/issue-activity/helper.tsx` 的 2 条（`preserve-caught-error`、`exhaustive-deps`），M4 改它时清零。收尾只为 `IconButton` 的名字改到、仍有警告的两个（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条）：`core/description-versions/modal.tsx` 1 条（`tabindex-no-positive`，第 7 节写死的 `tabIndex={1}`）、`issues/issue-detail/issue-activity/activity-filter.tsx` 2 条（`click-events-have-key-events`、`no-static-element-interactions`，第 7 节嵌套的按钮），M4 改它们时清零。`power-k/ui/pages/context-based/work-item/commands.ts` 418 行，M4 改写时拆开。两个工作项的删除确认框的 `onSubmit` 必传，改写时保持。
- **死成员**：`@nerve/types` 的 `workspace-draft-issues/base.ts` 的 16 行（`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **共用件**：`@nerve/ui` 下拉框的 `any`（`dropdowns/helper.tsx` 7 处、`dropdowns/custom-search-select.tsx` 2 处），改成泛型要改 13 个调用方，多数在 M4–M7 的页面上（M3/P10 spec 第 5 节）：M4 改它们时一起改。成员 store 的 `projectMemberIds` getter 在 M4 的读者是 `issues/issue-layouts/utils.tsx` 的两处，M7 的是视图、收集箱的筛选；M4、M7 中最后改完读者的那一个删除它（M3/P10 spec 第 3 节，裁定 F-2）。
- **搜索下拉框的英文默认值**（M3 收尾 Task 15 的评审）：`@nerve/ui` 的 `CustomSearchSelect` 的搜索框占位、没有结果、加载中三个文字的默认值是英文（"Search""No matches found""Loading..."），`rich-filters/` 下的 4 个调用方（`add-filters/dropdown.tsx`、`filter-item/root.tsx`、`filter-value-input/select/multi.tsx`、`filter-value-input/select/single.tsx`，后两个的共用 props 在 `select/shared.tsx`）都不传 `searchPlaceholder`、`noResultsMessage`、`loadingMessage`：M4 改它们时传经 `t()` 的文字，像成员下拉框（`dropdowns/member/base.tsx`）；最后一个调用方改完之后，M8 把这三个 prop 改为必传（M8 的交接第 3 节）。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M4/closeout`，`copy-hook` 的 10 个例外（`until: "M4/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M4/P<k>`。第 5 节 `/user-properties/` 的例外写的是 `until: "M4"`，`phase` 改为 `M4/P1` 时就到期：M4 的第一个 Phase 改筛选，或由 M4 的设计把它的 `until` 定到改筛选的那个 Phase。
- **关闭条件**：`project-address-route`、`workspace-address-route` 的范围含上面的文件，`copy-hook` 没有 `M4/closeout` 的例外，`--rows M3` 中 `workspace-draft-issues/base.ts` 的行在 M4 的评审中有结论。

## 9. 页大小的规则移到 `shared`（M2 收尾交接第 12 节，原样）

- **做什么**：`limit` 的 1–100、默认 50（`api/common.yaml` 的 `Limit`）现在在 `identity/domain`（`server/internal/modules/identity/domain/api_token.go:128-140` 的 `PageSize`），因为 M2 只有 PAT 列表一个使用者（M2/P3a spec 第 3 节第 8 条）。第二个分页列表出现时移到 `internal/shared`，两个列表共用。
- **同一处的提醒**：游标不签名（总体设计 3.4），改成另一个合格的位置照样可用。列表只能让游标决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：第二个分页列表合并时，页大小的规则在 `shared`，两个列表都用它；M3 没有分页列表时，M3 的 review 写明，把本节写进下一个 M 的交接。

M3 的列表都是集合型的，不分页（M3 设计 3.12、11.3），所以本节原样交给 M4（M3/P1、P2 评审第 6 节）。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M4 一行；M3 各 Phase 的 spec 第 5 节和评审第 6 节中交给 M4 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。

## 处理结果（M4/P1a）

- **第 2 节，先锁后判定的副作用**（P1a 的写完成；加关联的另一端的预检在 M4 的 P3，M4 设计 13.1）：工作项一侧的写在第一把锁之前做一次不加锁的预检，判定拒绝的（看不到答 404，看得到而不能写答 403）都在那里答，不取行锁；放行的写才锁项目行（M4 设计 3.5）。`TestEachIssueWriteLocksItsProjectFirst` 核对：另一个事务以 `EXCLUSIVE` 模式持有这个写读写的每张表（`users` 除外）时，看不到项目的 olga 照样得到 404，看得到而不能写的 walt 和 wanda 照样得到 403：不取行锁、不写（不加锁的读只取表的 `ACCESS SHARE`）。方案 E 的选择（不取工作区的 S，最先锁项目行，草稿的写和发布草稿除外；决策点 1 的 B）写在 M4 设计 3.5、M3 设计 17.4 和总体设计 4.2。
- **第 2 节，新的成员关系增长**（P1a 没有）：指派、关注不让谁成为项目成员（M4 设计 3.5），所以不加与停用的交错测试。
- **第 4 节，状态**（完成）：删除有工作项（已归档的也算）的状态 409 `project.state_in_use`，在项目 N 之下经 `IssueCascade.StateInUse` 数，在默认状态的检查之前（M4 设计 3.6，M3 设计 3.17）；新工作项取项目的默认状态，`State.default` 的说明补上这一句。
- **第 4 节，标签**（完成）：删除标签在同一个事务里、同一时刻从工作项上摘下它和它的子标签（`IssueCascade.DetachLabel`）；`issue_labels` 的写都持项目的 N，不落在约定五的例外里（M4 设计 3.5）。
- **第 4 节，项目**（完成一部分）：编号取 `projects.last_issue_sequence`（项目 N 之下加一）；`identifier`、`default_assignee_id` 的说明改成它们的作用（编号的前缀；新工作项的默认负责人，他仍是项目中不是访客的有效成员时才指派，否则不指派、不报错，`createIssue` 的说明写明）。`guest_view_all_features`（M4 的 P2）、`archive_in`（M4 的 P6）的说明随它们的读者改。
- **第 4 节，表和删除**（P1a 的五张表完成；以后每个加表的 Phase 照做，M4 设计 13.1）：五张表都带 `workspace_id`、`project_id`，进入删除项目、删除工作区的连带，同一时刻、同一个删除者（`IssueCascade.DeleteProjectIssues`、`DeleteWorkspaceIssues`，P9 换成任务）；按目录的删除测试给每张表准备一行，不开豁免；端到端的 `workspaceTables` 从目录读出，并像 `expectProjectDeleted` 一样核对它就是有 `workspace_id` 列的每张表，两个列表保持 `COLLATE "C"`；M3 的 P4、W2、W3 经 `furnishIssueRows` 带着工作项一侧的行。

仍未处理，状态保持 `open`：第 2 节其余各项：加关联的另一端的预检（M4 的 P3）、连接池的代价和写闸门（M4 的 P1b）、以后每个新的批量写按约定五核对（P1a 之后各 Phase 各自的 review，到 M4 的 P9）、物理删除账户（M4 不物理删除账户，M4 设计 3.21，在 M4 的 P9 关闭）；第 4 节的 `guest_view_all_features`（M4 的 P2）、`archive_in`（M4 的 P6）的说明，`expectIssueDeleted`（照 `expectProjectDeleted` 从目录自查、沿外键图递归，M4 的 P2）和删除断言对工作项一侧的表的限时轮询（M4 的 P9）；第 1、3、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

## 处理结果（M4/P1b）

- **第 2 节，连接池的代价和写闸门**（完成）：工作项一侧的写在预检之后、开始事务之前进入它的项目的写闸门（`platform/gate`；`issue/app` 声明的 `WriteGate`，`bootstrap/gate.go` 接线），不持连接、不持锁地等：每个进程里一个项目至多 `issue.write_gate.per_project`（默认 2）个写在数据库里，排队至多 `issue.write_gate.max_queue`（64）个、每个至多 `issue.write_gate.max_wait`（2 秒）；写的事务带 `issue.write_lock_timeout`（5 秒）的 `lock_timeout`。排满、等不到、在闸门上请求到期、等锁到期都答 503 `server_busy`，带 `Retry-After: 1`，只在 `bootstrap/gate.go` 一处换码（M4 设计 3.5、11.11）。交错 20 经真实的连接池（10 个连接）核对（`TestAHotProjectHoldsUpNoReadOfAnother`、`TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`）：200 个写压在一个项目上、它的行被别的事务持着时，另一个项目的读最慢不到 50 毫秒（没有闸门时一秒以上），超出闸门的 134 个写立即 503；首尾相连的工作项写之下，改工作区、改成员的角色、删除工作区各在 1 秒之内（M4/P1b spec 第 3 节）。闸门限的是每个项目的份额：一个进程里同时热的项目达到池的大小除以 K（默认 5 个）时池仍会被等锁的写占满；连接池的大小与 K、同时热的项目数、进程数交给 M8（M4 设计 13.2，收尾时写进 M8 的交接）。
- **第 2 节，约定五**（P1b 的批量写完成）：修改的两条批量写（`RemoveAssignees`、`RemoveLabels`：按谓词软删除一个工作项被去掉的负责人、标签）都在项目 N 之下，那些行只有持项目 N 的写会写到，不落在约定五的例外里；新加的负责人、标签、关注按 `id` 的顺序插入（M4/P1b spec 第 3 节）。

仍未处理，状态保持 `open`：第 2 节其余各项：加关联的另一端的预检（M4 的 P3）、P1b 之后各 Phase 的批量写按约定五核对（到 M4 的 P9）、物理删除账户（M4 的 P9）；第 4 节的 `guest_view_all_features`（M4 的 P2）、`archive_in`（M4 的 P6）的说明，`expectIssueDeleted`（M4 的 P2）和删除断言的限时轮询（M4 的 P9）；第 1、3、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

## 处理结果（M4/P2）

- **第 2 节，约定五**（P2 的批量写完成）：删除工作项的五条语句（`DeleteIssueRow` 按 `id`；`DeleteIssueAssigneeRows`、`DeleteIssueLabelRows`、`DeleteIssueSubscriberRows`、`DeleteIssueActivityRows` 按谓词软删除这个工作项未删除的负责人、标签、关注、动态）不分批，按全局的表序（工作项 → 负责人 → 标签 → 关注 → 动态）各执行一次，都写同一个时刻、同一个删除者；归档、恢复按 `id` 写工作项的一行（`SetArchivedAt`）。这些行只有持项目 N 的写会写到，不落在约定五的例外里，不按 `id` 先锁（M4/P2 spec 第 3 节第 7 条）。钉住它的测试：单元的 `TestDeleteIssueDeletesTheWorkItemAndItsRows`（表序、时刻、删除者）；存储的 `TestTheStatementsOfADeletedWorkItemDeleteItsRows`、`TestArchivingAWorkItemWritesItsRow`（每条语句只写它的工作项的行，写的是给定的时刻和账户）；组合的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、`TestArchivingAWorkItemStampsItsRows`（请求之内的一个时刻、做的人）；取锁形状的 `TestEachIssueWriteLocksItsProjectFirst`（删除、归档、恢复都最先取项目 N）；交错的 `TestDeletingAWorkItemAndChangingItSerialize`、`TestArchivingAWorkItemAndMovingItSerialize`（在项目 N 上串行）。
- **第 3 节，`AllowCreator`**（完成）：随第一条用到它的规则 `issue.delete`（管理员 + `AllowCreator`）加入规则表；判定读目标的创建者（`shared.Target.CreatorID`），调用者是它、在项目里的有效角色不低于成员时允许，已离开项目、成了访客的创建者不能再凭它删（M4 设计 3.4）；判定表测试逐格核对。
- **第 3 节，`guest_view_all_features`**（工作项的取、改、删、按编号取、归档、恢复完成；列表在各自的 Phase）：`access` 读项目的 `guest_view_all_features`，判定交回 `Grant.OwnOnly`（项目的访客、项目没开它时为真，按项目角色算：工作区管理员在项目里是访客时同样）；一条访客规则在读的判定中、在写的不加锁的预检和锁下的判定中各执行一次，别人建的工作项答 404 `issue.not_found`，不是 403（M4 设计 3.4、3.5）；交错 5 加上访客规则下 404 的一支。
- **第 3 节，矩阵和码**（工作项的操作到 P2 为止完成）：P2 的 `getIssueByIdentifier`、`deleteIssue`、`archiveIssue`、`unarchiveIssue` 在规则表、操作名、矩阵中各有一行；矩阵有了工作项表（M4 设计 9.2），P1a、P1b 的 `createIssue`、`getIssue`、`updateIssue` 的行移到它上面：项目级的各列之外，建了它的项目成员，项目访客的三种情形（看得到全部的；只看自己的，对自己建的、对别人建的），只是项目访客的工作区管理员（对别人建的、对自己建的）；设计 9.2 另列的"建了它、后来被降为访客"（403）和"建了它、已离开项目"（404）是"访客·自己"（种子记在他名下的）和"已离开项目"（他离开之前建的）两列上 `deleteIssue` 的格子。新码 `issue.archived`、`issue.not_archivable` 由 `issue` 的 HTTP 测试返回过（`apitest.Main` 两个方向），带前端文案。决策点 3 的个人主页一行在 M4 的 P5。
- **第 4 节，`guest_view_all_features` 的说明**（完成）：契约的 `Project.guest_view_all_features` 改写为它的作用（项目的访客是否看得到项目的每个工作项；不看时别人建的答 `issue.not_found`）。
- **第 4 节，`expectIssueDeleted`**（完成）：`e2e/fixtures/assert/issue.ts` 照 `expectProjectDeleted` 从目录读表，但沿外键图从 `issues` 往下走（图由 `assert/catalog.ts` 的 `keysBelow` 读出，与 `tablesUnder` 同在一处；走到的外键多于一列时失败）：一行的外键指向工作项或它下面的一行，它就在工作项下面（评论的表情指向评论，P4 起同样断言到），它下面的行都与它同一时刻、同一个删除者删除，每个外键下都有这样的一行，删除之前已删除的不算（它保留原来的删除由按目录的删除测试核对）；子工作项的外键（`issues.parent_id`）在 P3 之前没有行，豁免到 P3。第一个使用者是 I15 的接口版本（删除已归档的工作项）。

仍未处理，状态保持 `open`：第 2 节其余各项：加关联的另一端的预检（M4 的 P3）、P2 之后各 Phase 的批量写按约定五核对（到 M4 的 P9）、物理删除账户（M4 的 P9）；第 3 节其余各项：P2 之后各 Phase 的操作在规则表、操作名、矩阵中的行（完整性测试和 `apitest.Main` 随之通过；每个后端的 Phase，到 M4 的 P9）、决策点 3 的个人主页一行（M4 的 P5）和工作项列表的访客规则（各列表的 Phase）；第 4 节的 `archive_in`（M4 的 P6）的说明、删除断言的限时轮询（M4 的 P9）和 `expectIssueDeleted` 中子工作项的豁免（M4 的 P3）；第 1、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

## 处理结果（M4/P3）

- **第 2 节，加关联的另一端的预检**（完成）：`createIssueRelations` 在第一把锁之前对每个另一端做不加锁的预检：读它，按它的项目判定 `issue.read`，再照访客规则；看不到、不能读的另一端在那里记为它的位置上的 `issue_ids[i]` `not_allowed`（与不存在的工作项同一个答复），与锁下查到的其余问题在一个 422 里答，它的项目不进写闸门、不上锁；源工作项的项目和其余另一端的项目按 `id` 升序进入各自的写闸门，在一条语句里锁上（`LockMany`，总体设计 4.2 约定七），锁下的判定是最终的；预检没到达的另一端锁下不再读、不再判定，所以调用者等锁时变得看得到它，也仍在它的位置上得到 422 `not_allowed`（M4 设计 3.5、3.10）。`deleteIssueRelation` 在预检里读两端，先判定存为 `issue_id` 的一端，它拒绝时才判定另一端，调用者能改任一端即可；它写两端的行，两端的项目都进闸门、都锁上。单元的 `TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches`、`TestAWriteOnEitherWorkItemTakesBothProjects` 核对闸门和锁只取这些项目；组合的 `TestARelationLocksNoProjectItDoesNotReach`（交错 21 加的一条）：另一个事务持着 Lab（私有，bob 看不到）和 Pub（公开，bob 不能读它的工作项）的项目行时，bob 指向它们的关联立即得到 422，看得到 Lab 的 alice 等它；`TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused`：alice 打开 Web 的 `guest_view_all_features` 的修改持着 Web 的项目行时，walt（工作区管理员，Web 的访客）把他的工作项关联到访客规则对他藏起的 alice 的一个工作项，他的写过了预检、等这一行；她的修改提交之后，他照样在那个位置上得到 422，两个工作项的行都不变；`TestEachIssueWriteLocksItsProjectFirst` 的 `createIssueRelations`、`deleteIssueRelation` 两行核对看不到、不能写的账户不取行锁、不写。
- **第 2 节，约定五**（P3 的批量写完成）：持项目 N、不落在例外里的：`SetParents`（`addSubIssues` 把子工作项放到父工作项下，一条语句）、删除子树的语句（工作项和它们的负责人、标签、关注、动态，按 `id`、`issue_id` 属于子树，一个项目之内）、`MarkChanged`（关联两端的工作项，持两端项目的 N）；`InsertRelations` 按另一端的 `id` 的顺序插入（约定五的后半句）；`DeleteRelation` 按主键写一行。例外：关联行的 `project_id` 是 `issue_id` 一端的项目，另一个项目的写（删除另一端的工作项、删除它的项目）不持这个项目的锁也写得到这些行，所以 `DeleteIssueRelationRows`（删除子树时两个方向的关联）和 `DeleteProjectRelationRows`（删除项目、工作区时一批的关联）先按 `id` 锁住（`ORDER BY id FOR NO KEY UPDATE`），再只写锁住的那些（总体设计 4.2 约定五的延伸，M4 设计 3.5）。钉住它的测试：存储的 `TestTheRelationsOfADeletedSubtreeAreLockedInIDOrder`（别的事务持着中间的一行时，语句持着前面的、还没有取后面的），`TestTheRelationsOfADeletedSubtreeAreDeletedBothWays`、`TestARelationsBatchTakesTheRowsOfTheProjectsWorkItemsBothWays`、`TestSetParents`、`TestInsertRelations`、`TestMarkChanged`；交错 11（`TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize`）、23（`TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize`）、26（`TestDeletingARelationAndItsOtherEndSerialize`：删除关联与删除它另一端的工作项）和删除工作区与加关联的两种顺序（`TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize`），连同交错 9（两次移动；加子工作项与移动）、10、13 以 `-race -count=5` 跑过，没有 40P01。
- **第 3 节，规则表、操作名、矩阵和码**（P3 的操作完成）：P3 的五个操作不加规则、不加操作名：`listSubIssues`、`listIssueRelations` 是对工作项的 `issue.read`，`addSubIssues`、`createIssueRelations` 是对工作项的 `issue.update`，`deleteIssueRelation` 是对任一端的 `issue.update`，完整性测试不变。矩阵加 19 行（`permission_matrix_sub_issues_test.go` 7 行，`permission_matrix_relations_test.go` 12 行）：每个操作在工作项表上一行、在已归档的项目上一行；写的另加已归档的项目里已归档的工作项一行（项目的 409 在先）和已归档的工作项一行，加子工作项、加关联另加另一个项目的工作项一行，加关联再加已归档的项目的工作项一行。新码 `issue.relation_not_found` 由 `issue` 的 HTTP 测试返回过（`apitest.Main` 两个方向），带前端文案。
- **第 3 节，工作项列表的访客规则**（P3 的列表和数目完成）：`listSubIssues` 只列调用者照访客规则看得到的子工作项；每个答复的 `sub_issues_count` 照调用者的规则数（只看自己的访客只数他自己建的）；`listIssueRelations` 只列另一端调用者看得到的关联（另一端的项目逐个判定 `issue.read`，再照访客规则）。看不到的工作项不在列表、数目和关联内嵌的 `Issue` 里，回答里的 `parent_id` 指名看不到的工作项时为 `null`：取、按编号取、修改、归档、恢复、加子工作项、子工作项的列表的回答和加关联、关联的列表内嵌的 `Issue` 都经同一处 `Locks.seen`；创建的回答不经它，是事务里读回的那一行：它的父工作项是调用者看得到的（创建的检查），它的 `sub_issues_count` 是 0（新的工作项没有子工作项，不读）；修改的 `parent_id` 同调用者得到的回答比，他看不到的父工作项算没有：送回他得到的 `null` 什么都不改，指名它像指名不存在的工作项一样被拒绝，422 `parent_id`（负责人裁定 A，2026-10-24）；作为父工作项、加子工作项、加关联的目标，它的答复与不存在的工作项的逐字节相同，看不到的关联的 404 与不存在的关联的逐字节相同。组合的测试：`TestTheGuestRuleHoldsForSubItems`、`TestEachAnswerCountsTheSubItemsTheCallerSees`、`TestAHiddenParentIsNotInTheAnswer`、`TestAHiddenParentCountsAsNoneInAChange`、`TestARelationListShowsWhatTheCallerSees`、`TestTheGuestRuleHoldsForRelations`、`TestTheGuestRuleHoldsForDeletingARelation`、`TestWhoMayNotDeleteARelation`。
- **第 4 节，表和删除**（P3 的 `issue_relations` 完成）：新表带 `workspace_id`、`project_id`（`issue_id` 一端的项目），进入删除工作项、删除项目、删除工作区的连带，同一时刻、同一个删除者；按目录的删除测试给它准备一行（`seedIssues` 的关联），端到端的 `furnishIssueRows` 带着一个关联。
- **第 4 节，`expectIssueDeleted` 的豁免**（完成）：`e2e/fixtures/assert/issue.ts` 不再豁免子工作项的外键：沿外键图走到的每个外键下都要有与工作项同一时刻、同一个删除者删除的一行，`issues.parent_id` 和关联的两个外键在内。I15 删除已归档的工作项时它带着一个子工作项和两个方向的关联，I17 删除一个带子树和跨项目关联的工作项；组合的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 同样去掉了子工作项的豁免。

仍未处理，状态保持 `open`：第 2 节其余各项：P3 之后各 Phase 的批量写按约定五核对（到 M4 的 P9）、物理删除账户（M4 的 P9）；第 3 节其余各项：P3 之后各 Phase 的操作在规则表、操作名、矩阵中的行（每个后端的 Phase，到 M4 的 P9）、决策点 3 的个人主页一行（M4 的 P5）和其余工作项列表的访客规则（各列表的 Phase）；第 4 节的 `archive_in`（M4 的 P6）的说明和删除断言的限时轮询（M4 的 P9）；第 1、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P3 spec](../specs/P3-issue-hierarchy.md) 第 7 节。

## 处理结果（M4/P4）

- §2：P4新写和五表删除遵全局表序（issues→…→mentions→comments→issue_reactions→comment_reactions→activities→description_versions），项目N先行，批删除仍用现有键集。T13复核P3既有双端关系先relations后MarkChanged沿原批准项目N证明，不扩成新逆序许可；物理清理账户与后续批写仍归P9及拥有Phase。
- §3：12新操作同Task扩规则/操作/矩阵、HTTP双向契约/中英文错误；T14完整隐藏404与管理员兼访客OwnOnly、作者规则及归档顺序，T15 API故事通过；其余列表与个人主页仍归P5及后续Phase。
- §4：五表均随首个API带workspace/project外键、grants/sqlc/删除与furnish，评论表情二层实际FK两端由目录核验；T6两个错误目标列反例、T15旧删除分支/邻居快照和父子孙同戳验证通过；P9异步删除后的限时轮询尚未实现，archive_in说明归P6。
- §9：T2已将PageSize移至shared，默认50/范围1–100，PAT/评论/动态/版本共用；T2 ReadOnlySnapshot已落地并与LockTimeout正确组合，T3/10/12/14消费首读、历史目录、SQL、表情批读同一只读快照。P5复用该能力和Historical目录，新增普通VisibleProjects方法、替换P3 seeEach/ArchivedProjects及force_custom_plan，勿重复实现。
- 余项：§1全M4锁/规模、§5–8筛选/service/store/浏览器/复制/文案按原分配继续，P11新DTO/地址与comment_json、P14保存和弹窗、P15时间线页面是后续工作；本Phase不关闭整份交接。

状态保持 `open`；本次仅完成P4接收部分，待M4真正收尾逐项核对关闭条件。证据：[P4 spec §4/§7](../specs/P4-issue-collaboration.md#4-验收标准)、[P4 review §2/§6](../reviews/P4-issue-collaboration-review.md#2-验收标准核对)。

## 处理结果（M4/P5）

- §2/§3：T8新排序写沿原Locks.run/全局表序；T9/T10扩项目列表40cells、个人三关系18cells，T13b扩既有偏好27cells完整三字段；T14实际复跑矩阵/runtime/双向契约/历史45项及所有新增和六既有交错race五遍。caller/target分离、管理员兼guest仍OwnOnly、个人目标inactive存在和隐藏先404均按原规则。
- §3个人主页：member_issue.list只允许工作区管理员/成员；T10/T12实际三关系与跨项目caller目录，T15 I19每目标集合互异并保留自动订阅，I20补访客四格。T4普通Visible和完整名字目录、T11替换P3 seeEach/ArchivedProjects，保留归档源同项目及Historical，T14实际回归无范围倒退。
- §5服务端：project_user_properties经00026从11→14列，只加filters/display_filters/display_properties，不建rich_filters；workspace_user_properties不变。T13a真实旧库/默认/形状CHECK/Down-Up，T13b闭合API、纯端口接线、GET无写、整字段替换、恢复保留完整行/审计及已有行四方向并发不失更；T14他人行写范围变异、T15 I5保存/查询等价已验证。
- §9：直接复用P4的shared.PageSize（50/1–100）、独立ReadOnlySnapshot与Historical，不重复实现；T1真实双连接参数覆盖，T12八提交快照与最终32轮性能保留全排序/temp、成员相关扫描/跨项目成本，M8继续测量。T14真实满写gate及EXCLUSIVE表锁/单连接下列表均读通。
- 仍开放：§1整体规模/锁与后续批写，§5前端旧ProjectService/user-properties/workItemFilters迁移P10b；api/testdata/issue-order.json由P10b生产前端码点比较器和vitest同文件消费。相关until:M4/P11例外到实际迁移才删；P11详情store/service/JSONContent/comment_json/is_bot、§6–8及P12–15页面、P6 archive_in/P9限时删除等保留原分配。

仅登记本输入的P5承接，`status: open`保持；不以八接口故事关闭页面、M4总体或T16整项，不生成M5–M8的真正收尾文件。证据：[P5 spec §2/§7](../specs/P5-issue-list.md#2-交付物)、[P5 review §2](../reviews/P5-issue-list-review.md#2-验收标准逐条核对)。

## 处理结果（M4/P6）

§1工作项事件事务发布已实现；`jobs_test.go`/`runner_test.go` 两处固定余量已改为状态与期限，`registrationRows` 对请求入队的 `river_job` 仍属P7，项目/工作区删除任务及邀请同步边界仍属P9。§4 `Project`/`ProjectPatch.archive_in` 的0关闭、1–12个月×30天、终态和 `updated_at` 说明已通过生成链；锁下策略、每项目事务、NULL actor和运行角色真实验证由T9–T12完成。§2/3既有锁与权限回归由T15全套覆盖，P9物理清理和删除轮询、§5–9页面仍按原分配。整体 `status: open`；见 [P6 review §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)/[§6](../reviews/P6-issue-events-review.md#6-移交事项)。

## 处理结果（M4/P7，2026-10-27）

§1的registrationRows实际在bootstrap/invitations_test.go，保留普通表完整行，另按id稳定附加各模块声明请求kind的river_job id/kind/args；新增任务或同ID改args可见，worker state/attempt与periodic/leader噪声不污染快照。request_jobs_test.go的独立kind和两个测试守卫反例、issue_link_runtime_test.go的真实受限角色CRUD+crawl completed/metadata及额外表42501已验证。

§2–4链接请求遵守项目锁、前置隐藏404/权限、同刻审计与完整回滚；新issue_links纳入删除目录及API furnish，旧tombstone和邻居完整行保留。邀请删除仍同步；project/workspace异步删除、清理与限时轮询仍归P9，§5–9数据层/页面仍按原分配。

本回执只关闭P7对应子责任，顶部`status: open`保持；M4整体、P8/P9/P11/P14/P15及M5–M8最终closeout不提前关闭。证据：[P7 review §2](../reviews/P7-issue-links-review.md#2-验收标准逐条核对)、[行为矩阵](../reviews/P7-issue-links-behavior-matrix.md)、[整体审核](../../../reviews/re-05-M4P7整体审核.md)。本波独立定向复审与最终分支CI状态在P7 review单列，未合并main。

## 处理结果（M4/P8，2026-10-27）

§2/§3/§4的P8输入已承接：T1将workspaceID贯穿同步级联，T4 API furnish全部调用点及动态全表目录纳draft，T8 W3显式无项目/零项目workspace删除及旧删除草稿审计保持。T3/T6唯一DraftLocks组合base；CRUD W(S)→升序P(S)→D(U)，publish无资源gate→W(S)→P(N)→D(U)，普通issue路径保持。T7请求PID/holder、交错16/17两次序/UUID两方向、W/P/D终态/唯一At/零gate锁及T15 race五遍有实证；六draft操作与search进入规则/错误/矩阵，guest/owner/OwnOnly和404逐值对照通过。历史M4§11.2摘要D1仅修成版本保留按§3.14项目N、P9待实现，不是本期实现P9。异步删除/物理清理/账户组合/限时轮询与页面等余项仍开放，顶部status继续open。

准确产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6`，首次整审 `166b9efd..10f44790`；[P8 review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)列A条款/真实RUN/变异恢复/四次七门禁及准确产品checkpoint CI，[整体审核](../../../reviews/re-07-M4P8整体审核.md)列三镜头/186文件覆盖与历史D1。文档增量独评及最终准确HEAD CI由后续提交和ignored收据追踪，未合并、不清理；不以本期子项已处理改写历史关闭条件或M4整体状态。

## 处理结果（M4/P9，2026-10-28）

| 原输入 | 本期处理结果 | 证据/仍开放责任 |
|---|---|---|
| §1事件/删除连带 | project/workspace原事务发布，Changes只含本次锁住活项目；每项目一job+workspace_drafts（零项目仍投），Old保原审计；邀请/M3同步，registrationRows识别请求kind/args和同id替换 | T8–12原子提交/回滚与目录；非全M3异步，不新增CLI任务 |
| §2约定五/账户 | 不物理删账户；purge为候选id键集+pool单DELETE，trim为项目N+事务，短批/relation显式先锁后写；原错/取消/提交边界逐函数核对，NULL creator独立组合 | T7/11/12/15与最终fixture-M1真实第二请求等待修复；不把候选排序泛化为物理DELETE锁序 |
| §4清理/目录/删除故事 | M3表纳23清理者/索引/FK；13表issue侧有限poll，M3即时断言；W3 Old与零项目直接目录及runtime草稿准确At/By、非空/旧行/邻居保留 | T2–4/T10/T13；目录不新增豁免 |

§3规则/矩阵、§5旧service/显示设置、§6会话/页面取数、§7键盘/弹窗、§8地址/复制/文案/死行、§9共享页大小按原归属继续。只有本期后端子项承接，前端与整体M4收尾不关闭。

证据：[P9 review](../reviews/P9-purge-review.md)及[spec §7/附录D](../specs/P9-purge.md)；最终产品 `e003e96625ba51264d3aeb77729729d53a770103`。最终F/准确分支CI、T16文档独评在review分别列明。本文件 `status: open` 保持到M4收尾，历史原文/先前回执保留；不因P9后端承接关闭整份交接。


## 处理结果（M4/P10a，2026-10-28）

第5/6节本期子项：迭代、模块、项目/工作区视图工作项列表/筛选和独占动作已删除，领域自身详情/目录装载仍保留；T5进一步删除最后普通cycle/module写链，T6删除从core不存在字段提取关系并覆空cache的列表extractor，详情所属资源读取保持。直接生成核心、描述/null/局部输入合同与T7逐方法旧URL最早到期账见[P10a实施记录](../reviews/P10a-web-issue-types-review.md)。

R15新增reaction/link/非null父GET由所属fetch等待并传拒绝，T5 I1/I3及r2 N1已独立复审闭合，真实Base可提示/重试、旧来源/关闭后晚结算不覆盖当前描述或失败状态。此结论限新增资源与当前表单快照路径，不覆盖历史activity/comment/subissue/relation/state detached、所有下拉框按需拒绝或全部模态宿主生命周期。

preferences/workItemFilters与S2 REQUESTS增长仍由P10b；搜索/旧ProjectService依P12/P14读者改接及P15清理，`/search/`无自动guard且until M4/P16；详情资源/其余按需拒绝、模块service和宿主收尾留P11，各页面key/后退与交互按P13–P15。M6/M7自身完整可用/零失败和实际逐请求过渡清单由[P10a review §6](../reviews/P10a-web-issue-types-review.md#6-移交事项)交接；不把列表拆除通过视为其自身页成功，顶层status保持open。

本期Task级完成至T8提交8081112492a2fc744a43b3e814c6d36ff4ba3959；A-01/A-02代码终审修复已独评并提交5f7085bce4aa5103a0133fa982847d274efa6e47；文稿最后独评已通过，最终交付HEAD分支CI待提交后核验，准确产品5f708 CI已核成功见review§8，不把本段子项写成M4最终交付或集成。


整审A-01无消费者子项两域预取与A-02退休列表独占乐观分布链已根因清理并独评；未改会话/资源/领域协议，真实目录/统计/进度保留，不扩大本输入其余关闭范围。

## P10b 本段处理（2026-10-29）

§5 user-properties 两方法、例外和 workItemFilters 已由 T7 删除，T13 四故事复核；§6 router autorun、Reconciled 扩展与 S2 请求集由 T2/T10/T13 验证。详情取数与页面、旧搜索及 is_bot 分别仍属 P11–P15。 证据见 [P10b review](../reviews/P10b-web-list-data-review.md)；最终F、CI与整审待办，整份交接 status 保持 open。
