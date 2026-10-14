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

- 工作项列表的筛选和显示列加在 `workspace_user_properties`、`project_user_properties`（M3 设计 3.18），格式由 M4 定；旧 `ProjectService`（`web/apps/web/core/services/project/project.service.ts`）的 `/user-properties/` 两个方法和关键词守卫的例外（`until: "M4"`）随之删除；今天工作项页面上 `user-properties` 的请求答 404（M3/P8b spec 2.12）。
- 旧 `ProjectService` 的 `projectIssuesSearch` 和它的三个调用方（M4 的选父工作项、M6 的添加已有工作项、M7 的收集箱查重）随 M4 的工作项搜索接口替换，之后删除旧 service（M3 设计 7.3）。旧 `WorkspaceService`（`web/apps/web/core/services/workspace.service.ts`）的视图、草稿、搜索方法同样随各自的接口改接或删除。
- 工作区搜索的类型和参数：`@nerve/types` 的 `workspace.ts` 24 行、`project/projects.ts` 9 行，`workspace.service.ts` 3 行（`domains.mjs --rows M3` 列出的 36 行，属于 M4，M3/P11 spec 附录 A.9）。
- 活动类型中的 `is_bot`（`@nerve/types` 的 `issues/activity/base.ts` 的 `TIssueActivityUserDetail`、`comments/card/display.tsx`）：Nerve 不区分人和机器人，删除（M2 收尾交接第 11 节留下的两处）。
- **关闭条件**：`/user-properties/` 的例外从 `tools/keywords.json` 删掉；旧 `ProjectService`、`WorkspaceService` 中工作项的方法都已改接或删除；`is_bot` 不再出现在 `web/` 里。

## 6. stores、会话和页面的取数（总体设计 7.7）

- `issue/root.store.ts` 的 `autorun` 不在 action 之外写 `router` 的副本：把 `workspaceSlug`、`projectId` 等做成 `rootStore.router` 上的计算属性，或改成 `reaction`，由 `RootStore.dispose()` 调用它的释放函数（在那之前 `store-context.test.ts` 有两条 MobX 严格模式的警告，M3/P8a 评审第 6 节）。工作项的分页、分组列表扩展 `core/lib/reconciled.ts`，不另写一份。
- **按需的取数按权限启用**（M3/P8b spec 第 5 节）：状态、成员、标签的下拉框打开时取（`dropdowns/state/dropdown.tsx`、`dropdowns/member/dropdown.tsx`、`issues/select/dropdown.tsx`、`issues/issue-layouts/properties/label-dropdown.tsx`、`issues/issue-detail/label/select/label-select.tsx`），子工作项的 store 取别的项目的状态、成员、标签（`store/issue/issue-details/sub_issues.store.ts`，调用者可能不是那些项目的成员）。被拒绝时：状态下拉框的加载一直不结束、是未处理的拒绝，成员下拉框是未处理的拒绝，工作项 store 不等结果的取数（`issue.store.ts` 取状态的两处、`sub_issues.store.ts` 取状态、成员、标签）是未处理的拒绝，三个标签选择器的取数被拒绝时一直"加载中"或被吞掉（Plane 原有，M3/P8b 评审的 P9）。
- 错误经文案表（`errorMessageKey`）：工作项的创建、修改读 `error?.error` 的两处（`issues/issue-modal/base.tsx`，M3/P9 spec 第 5 节）。
- 工作项页面上的标签新建（`issues/` 下的 `issue-detail/label/root.tsx`、`issue-detail/label/select/root.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`select/dropdown.tsx`、`select/base.tsx` 直接调 store 的 `createLabel`）改经 `useLabelChanges().create`（会话的核对、名称已被占用在名称下），之后可以加关键词规则（M3/P11 spec 第 3 节第 13 条）。
- 工作区一级显示的工作项不显示没有取过标签的项目的标签（M3/P8b 删了为每个项目取标签的工作区取数，M7 加回）；`fetchParentStats` 不再重取项目，工作项的修改若动了项目的字段，加一个不等的重取；工作项的键宽读回 `next_work_item_sequence`，`TIssueIdentifierProps.projectIdentifier` 在项目未到时可以是 `undefined`（M3/P8b spec 第 5 节）。
- 个人主页的工作项一栏（成员和管理员；`/api/workspaces/{slug}/user-issues/…` 今天答 404）和访客看到的筛选工具栏（布局、筛选、Display 的控件在"没有权限"之上，M3/P11 评审 C10）。
- M3 页面上指向工作项页面的链接接上之后，这些页面进入 `watchPage` 的范围（M3 设计 3.1）；S2 的 `REQUESTS` 是一份要维护的名单，在挂载路径上加取数时同时改它（M3/P11 spec 第 6 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，开着的弹窗和组件的状态不带过去。工作项的弹窗、选中、草稿等属于一个工作区或项目的界面状态放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）；要在切换之间留着的，写明为什么。
- **工作项级弹窗的宿主**（M3 收尾；控制者的裁定：store 持有的弹窗开关属于渲染它的宿主，由宿主结束）：`modals/work-item-level.tsx` 的 `WorkItemLevelModals`（创建工作项 `isCreateIssueModalOpen`、删除工作项 `isDeleteIssueModalOpen`）。开关在命令面板的 store（`web/apps/web/core/store/command-palette.store.ts`）里，宿主由 `power-k/projects-app-provider.tsx` 渲染，在 `(projects)/layout.tsx` 和 `(settings)/layout.tsx` 的 `ProjectsAppPowerKProvider` 里，在项目的 key 之上：换项目（后退、前进也是）时开着的弹窗留着，作用于路由新指向的项目。M4 接上工作项页面时两件都做：把宿主移到项目的 key 之下，或在项目的范围结束时关上它的开关；并像 `modals/workspace-level.tsx` 的 `WorkspaceLevelModals` 那样在卸载时关上它渲染的每个开关（`useLayoutEffect` 的清理，在下一个页面绘制之前）。加一个像 P3、P8 那样的后退检查（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）。
- **嵌套的 id 没有 key**（M3 收尾）：工作项的页面 `projects/(detail)/[projectId]/issues/(detail)/[issueId]`、`archives/issues/(detail)/[archivedIssueId]`、`browse/[workItem]` 和个人主页 `profile/[userId]`（连同 `[profileViewId]`）只在工作区、项目的 key 之下，换一个工作项或一个人（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M4 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：上面每一处的拒绝有提示或交回调用方，没有未处理的拒绝（vitest 或故事）；`autorun` 随退役的一代释放，`store-context.test.ts` 没有严格模式的警告。

## 7. 工作项页面的键盘、下拉框和弹窗

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：弹窗打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉、取消不能用。工作项的弹窗在 M3 的故事里走不到：M4 对接时逐个核对 Tab 在弹窗里转、Escape 和点外面关上、关上之后焦点回到打开它的控件，以及弹窗里经 portal 渲染的东西（日期选择、下拉框、表情）能用（收尾 spec 附录 A.3 的门户清单只证明了 M3 页面上的几种）。`AlertModalCore` 的确认框照 A-M2：被拒绝时留着，只在成功的跟进中关上；按钮的文字经 `t()` 传入（`@nerve/ui` 的默认值是英文，见 M8 的交接）。
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
- **关闭条件**：M4 的评审写明上面每一项的浏览器核对；`getTabIndex` 在最后一个删去正的 `tabIndex` 的 M 删除；上面层叠的每个元素降到 `z-30` 之下，或有开着弹窗的核对。

## 8. 地址、复制、文案和留下的行

- **地址只从路由参数**：工作项的页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节 M4 一行列出：6 个，和裁定 F-12 补上的 16 个），M4 改成读路由参数，并把它们加进关键词规则 `project-address-route` 的范围。只读工作区的 4 个（`app/(all)/[workspaceSlug]/(projects)/drafts/header.tsx`、`issues/workspace-draft/empty-state.tsx`、`issues/workspace-draft/root.tsx`、`power-k/config/creation/command.ts`）改用 `useRouteWorkspace()`（`core/hooks/use-route-workspace.ts`）或路由的 slug，并加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）。
- **复制只经 hook**：工作项页面的 10 个文件（`issues/` 下的 `create-issue-toast-action-items.tsx`、`issue-detail-widgets/relations/helper.tsx`、`issue-detail-widgets/sub-issues/helper.ts`、`issue-detail/identifier-text.tsx`、`issue-detail/issue-activity/helper.tsx`、`issue-detail/issue-detail-quick-actions.tsx`、`issue-detail/links/link-item.tsx`、`issue-layouts/quick-action-dropdowns/helper.tsx`、`peek-overview/header.tsx`，和 `power-k/ui/pages/context-based/work-item/commands.ts`）改经 `useCopyLink` 或 `useCopyText`（`core/hooks/use-copy-link.ts`），失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 10 个例外（`until: "M4/closeout"`）随之删除。评论链接和工作项链接的复制今天没有处理拒绝（M3/P11 spec 附录 A.6）。
- **oxlint**：`issues/issue-detail/issue-activity/helper.tsx` 的 2 条（`preserve-caught-error`、`exhaustive-deps`），M4 改它时清零。收尾只为 `IconButton` 的名字改到、仍有警告的两个（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条）：`core/description-versions/modal.tsx` 1 条（`tabindex-no-positive`，第 7 节写死的 `tabIndex={1}`）、`issues/issue-detail/issue-activity/activity-filter.tsx` 2 条（`click-events-have-key-events`、`no-static-element-interactions`，第 7 节嵌套的按钮），M4 改它们时清零。`power-k/ui/pages/context-based/work-item/commands.ts` 418 行，M4 改写时拆开。两个工作项的删除确认框的 `onSubmit` 必传，改写时保持。
- **死成员**：`@nerve/types` 的 `workspace-draft-issues/base.ts` 的 16 行（`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **共用件**：`@nerve/ui` 下拉框的 `any`（`dropdowns/helper.tsx` 7 处、`dropdowns/custom-search-select.tsx` 2 处），改成泛型要改 13 个调用方，多数在 M4–M7 的页面上（M3/P10 spec 第 5 节）：M4 改它们时一起改。成员 store 的 `projectMemberIds` getter 在 M4 的读者是 `issues/issue-layouts/utils.tsx` 的两处，M7 的是视图、收集箱的筛选；M4、M7 中最后改完读者的那一个删除它（M3/P10 spec 第 3 节，裁定 F-2）。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M4/closeout`，`copy-hook` 的 10 个例外（`until: "M4/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M4/P<k>`。第 5 节 `/user-properties/` 的例外写的是 `until: "M4"`，`phase` 改为 `M4/P1` 时就到期：M4 的第一个 Phase 改筛选，或由 M4 的设计把它的 `until` 定到改筛选的那个 Phase。
- **关闭条件**：`project-address-route`、`workspace-address-route` 的范围含上面的文件，`copy-hook` 没有 `M4/closeout` 的例外，`--rows M3` 中 `workspace-draft-issues/base.ts` 的行在 M4 的评审中有结论。

## 9. 页大小的规则移到 `shared`（M2 收尾交接第 12 节，原样）

- **做什么**：`limit` 的 1–100、默认 50（`api/common.yaml` 的 `Limit`）现在在 `identity/domain`（`server/internal/modules/identity/domain/api_token.go:128-140` 的 `PageSize`），因为 M2 只有 PAT 列表一个使用者（M2/P3a spec 第 3 节第 8 条）。第二个分页列表出现时移到 `internal/shared`，两个列表共用。
- **同一处的提醒**：游标不签名（总体设计 3.4），改成另一个合格的位置照样可用。列表只能让游标决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：第二个分页列表合并时，页大小的规则在 `shared`，两个列表都用它；M3 没有分页列表时，M3 的 review 写明，把本节写进下一个 M 的交接。

M3 的列表都是集合型的，不分页（M3 设计 3.12、11.3），所以本节原样交给 M4（M3/P1、P2 评审第 6 节）。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M4 一行；M3 各 Phase 的 spec 第 5 节和评审第 6 节中交给 M4 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
