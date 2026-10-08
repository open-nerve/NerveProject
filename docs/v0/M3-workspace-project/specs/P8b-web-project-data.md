# M3/P8b 项目一侧的数据层：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P8b `web-project-data` |
| 日期 | 2026-10-08 |
| 状态 | 设计完成，待控制者裁定第 3 节标"裁定"的各条；未执行。拆分照控制者的裁定 R1–R4（负责人 2026-10-07 确认） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（S2）、3.1、3.2（`cover_image_url`、`is_favorite`、`intake_count`）、3.4、3.16–3.19、3.20（P8b 的两行）、5.1、7.1–7.3、7.6、7.8–7.11、8.3、9.2、9.5、12（P8b 与约束 2、4）、13.1 节；[总体设计](../../v0-design.md) 7.7 |
| 前置交接 | [P8a spec](P8a-web-workspace-data.md) 第 5 节 P8b 一行；[P8a review](../reviews/P8a-web-workspace-data-review.md) 第 6 节 P8b 一行（P14、P18、S2 的改写、A6 的范围、F-1、F-4、F-5、F-6）；[M2 收尾交接](../handoffs/M2-closeout.md) 第 3、11 节；[M1-P3](../handoffs/M1-P3-trim-platform.md) 的项目成员；[M1 收尾交接](../handoffs/M1-closeout.md)（oxlint）；P4b、P7a、P7b 的 spec 第 5 节和 review 第 6 节中 P8 的行；[Codex 设计评审](../reviews/M3-design-codex-adversarial-review.md) M-3、4.3 |
| 计划 | [P8b plan](../plans/P8b-web-project-data.md) |

本 spec 只写 M3 设计交给 P8b 决定的东西：名字、签名、文件的位置、测试名，以及原型证明了什么。规则本身以 M3 设计和总体设计 7.7 为准，这里引用节号，不重述。P8b 依赖 P8a（`7390e012` 的 `main`）。

P8b 的每一处取数都按安全测试看待（brief）：一个角色不能发的请求在页面上是 403；换账户之后才落地的写以错误的身份写入；不是项目成员的人（`member_role` 为 `null`）不能取项目的任何子资源。所以附录 A 的变异逐个核对：项目一侧的每个 store 只给出项目 store 此刻仍给出的项目的东西（离开、删除的项目和不再是调用者的工作区里的项目什么都不给，P14）；权限 store 的项目一半与 9.2 同一组身份一致（含 PM+WA、WA-，P18）；项目包装层只在 nerve 说调用者是成员之后才取子资源；每个修改一个接一个（拖动排序也是）；每个会话的取数经 `useSessionSWR`（键带 `loginId`）。只有评审才能发现的会话、权限或取数的性质算缺口（第 3 节第 11 条是唯一一个，在本 Phase 的最后一个 Task 关闭）。

## 1. 目标

按 M3 设计 12 节 P8b：

- 项目一侧的类型、service、store 迁到 `/api/v0`：项目（未归档、已归档两份列表，详情，创建、修改、删除、归档、恢复、加入、离开、侧边栏的位置、标识检查）、调用者在项目里的标签栏、项目成员、状态（顺序和组中的位置）、标签（两层、nerve 的顺序），按会话分代，修改经 `oneAtATime()` 一个接一个、写入 nerve 的回答，取数与修改经 `core/lib/reconciled.ts` 调和，每个项目、每个工作区的状态按 id 存（总体设计 7.7；P8a 的裁定 F-1、F-4、F-5、F-6）。
- 权限 store 的项目一半取自 `Project.member_role`（3.4、7.3）；`project-roles` 和 `project-members/me` 的取数删除（7.8）。
- 两个包装层项目一侧的挂载时取数改经 `useSessionSWR`、按权限启用（7.1）：工作区包装层在调用者的列表有这个工作区之后取项目列表和工作区的状态；项目包装层先取项目，nerve 说调用者是它的有效成员之后才取标签栏、标签、成员和状态，包装层的全部判断在一个可以脱离 React 测试的 hook 里（`useProjectFetch`）。
- 生成的 `Project` 没有的字段从页面删除：封面的上传和预设、项目的收藏（M7）、`intake_count`、`next_work_item_sequence`（3.2、7.8）；M7 的收集箱分诊状态和工作区级的标签从 store 和页面删除（3.1、3.16、3.17）。
- 使用方改到能编译、行为不变；不同的逐条写在第 3 节第 21 条（W17）。
- 旧 `ProjectService` 只留三个方法，其余旧的项目一侧 service 和模块级实例删除（7.3；M2 交接第 3 节的 10 处至此全部消失）。
- 关键词规则补全（7.10）；S2 改写为四个账户登录之后的挂载清单（第 2 节 S2）；3.20 中 P8b 的两行。

## 2. 交付物

**文件总览**：plan 的"文件结构"一节逐个列出 Task 1–11 改到的 262 个文件和每个文件所属的 Task：新文件 15 个（5 个 service、1 个 store、2 个 hook、6 个测试、1 个测试的共用部分 `core/store/project/fake-projects.ts`）、删除 67 个（Task 1 的 56 张预设封面和封面选择器，Task 3、6、8、9 的四个旧 service，Task 7 的分诊状态的 6 个文件）、修改 180 个（其中 38 个只经 Task 2、8、9 的机械步骤改到）。web 之外只有 `tools/keywords.json`、根目录 `.oxlintrc.json`、`pnpm-lock.yaml`、`e2e/stories/smoke/s2-web-app.spec.ts` 和两份上级文档（总体设计、前端改动清单）。契约、Go 代码和生成的文件都不变。

**依赖**：不加 npm 包或 Go 模块。`@nerve/propel` 加工作区内的依赖 `@nerve/api-client`（`workspace:*`）：项目的图标是生成的 `LogoProps`，`Logo` 组件收它（7.2：类型只来自生成的客户端；P8a 的 `@nerve/types`、`@nerve/utils` 同样）。锁文件随之一处，`pnpm install --frozen-lockfile` 通过。

### 2.1 生成的 `Project` 没有的（Task 1；3.2 的规则 2、7.8）

- 封面的上传和选择删除：`core/components/core/image-picker-popover.tsx`；`helpers/cover-image.helper.ts` 只留 `DEFAULT_COVER_IMAGE_URL`（`image_1.webp`）和 `getCoverImageDisplayURL`；其余 28 张预设封面（`image_2`–`image_29` 的 `.svg`、`.webp`，56 个文件）和 `SOURCES.md` 中它们的行；`EFileAssetType.PROJECT_COVER`、`ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE`、`TAB_INDEX` 的 `cover_image`、文案 `change_cover`（两种语言）。卡片、设置页、创建弹窗只显示封面，没有自己的封面时显示默认的那一张（`CoverImage` 的 `showDefaultWhenEmpty`）；创建时不再随机给封面、不再上传（第 3 节第 3 条）。
- 项目的收藏（M7）：卡片的星标、创建弹窗的 `setToFavorite`、侧边栏菜单中注释掉的收藏一段、`FavoriteStore` 的项目一支（写 `projectMap[…].is_favorite`）。工作区的收藏列表本身不动（M7）。
- `intake_count`：侧边栏"收集箱"一项的数字，收集箱 store 在状态变化时改写它的两段。
- `next_work_item_sequence`：列表视图中工作项的键宽按一位数字算（`currentProjectNextSequenceId` 不再读，M4 加回）。
- 这些文件的 oxlint 警告清零（7.9）。没有新的 vitest：只删除，`tsc` 和 knip 核对没有留下读者（附录 A.7 的 W13）。

### 2.2 生成的 `Project`、`ProjectCreate`（Task 2；7.2）

- `Project` 取代 `TProject`、`IProject`、`TPartialProject`、`IPartialProject`；创建表单的值是 `ProjectCreate`。字段照生成的类型：`workspace_id`、`member_ids`、`project_lead_id`、`default_assignee_id`、`intake_view`（Plane 的 `inbox_view` 是它的只读别名，7.2）；`logo_props` 是 `LogoProps`；`cover_image_url` 可空，读作 `null`（W20）。38 个文件的改名是机械步骤（`rename_types.py` 三次），其中 16 个另有手改。
- `packages/types/src/project/projects.ts` 只留 M3 之外的：`EUserProjectRoles`、项目成员的过渡类型（Task 6 删除）、`IProjectMemberNavigationPreferences`（Task 5 删除）、M4 的工作项搜索参数和回答。
- `ProjectStore` 中读者在 Task 1 已删除的 `favoriteProjectIds`、`currentProjectNextSequenceId`、两个收藏方法，和 `getPartialProjectById`（使用方改调 `getProjectById`，两者一直给出同一个对象）删除。项目设置的成员默认值的表单只有它改的两个字段（`Pick<Project, "project_lead_id" | "default_assignee_id">`），Plane 读嵌套对象的 `as MemberUser`、`as Workspace` 随之删除。
- `@nerve/utils` 的 `project.ts` 有手改，两处 `sortBy(…).reverse()` 改为 `toReversed()`（`sortBy` 交回新的数组，顺序不变），清零；`utils` 的上限 12 → 10。

### 2.3 项目的 service 和 store（Task 3；3.19、7.1、7.3）

- `ProjectsService(api)`：`list(slug, archived)`（交回 `ProjectList.data`）、`create`、`get`、`update`、`delete`、`archive`、`unarchive`（Task 3），`join`、`leave`（Task 4），`checkIdentifier`（Task 5）。每个方法一次调用生成的客户端。`ProjectPreferencesService(api)`：`update`（侧边栏的位置，Task 3）、`get`（Task 5）。旧的 `project-archive.service.ts` 删除。
- `ProjectStore(rootStore, filters, api)`（`core/store/project/project.store.ts`，重写）：每个工作区的两份列表（未归档、已归档）按工作区的 id，每个项目自己的读按项目的 id，三者都是 `ReconciledByKey`。`getProjectById(id)` 先给项目自己的读，再给列表中的；项目所在的工作区已不在调用者的列表中时什么都不给（P14）。`workspaceProjectIds`、`totalProjectIds`、`filteredProjectIds`（项目页的筛选和顺序，两份列表都到了才给）、`joinedProjectIds`（调用者是成员的未归档项目，按他侧边栏的 `sort_order`）、`loader`。取数：`fetchProjects(workspace)`、`fetchArchivedProjects(workspace)`（收 `Pick<Workspace, "id" | "slug">`）、`fetchProject(projectId)`。修改经 `changes = oneAtATime()`、写入回答、被拒绝时不改：`createProject`（排在已取的列表最前，`prepended`）、`updateProject`、`deleteProject(project)`、`archiveProject`、`restoreProject`（在两份列表之间移动）、`updateProjectSortOrder(project, sortOrder)`（发到调用者在项目里的设置，写回答的 `sort_order`），`joinProject`、`leaveProject`（Task 4），`checkProjectIdentifier`（Task 5，不改 store），`confirmProject(project, change)`（Task 6）。Plane 的 `projectMap`、`fetchStatus`、`fetchPartialProjects`、`fetchProjectDetails` 和按 slug 的签名删除。
- `ProjectRootStore(root, api)` 先建筛选 store 再交给 `ProjectStore`；`RootStore` 传这一代的客户端。
- 挂载时：`useWorkspaceFetch` 在列表中有这个工作区之后取 `["PROJECTS", id, slug]`；项目页（`components/projects/page.tsx`）取 `["ARCHIVED_PROJECTS", id, slug]`。工作区包装层的 `fetchPartialProjects` 的 `useSWR` 和键删除。
- 测试的共用部分：`fake-projects.ts` 的 `projectOf(identifier, workspaceId, fields?)`、`loadProjects`、`loadArchivedProjects`（经 P8a 的 `answered`，核对方法和路径）；`fake-store-hooks.ts` 加 `useProject()`。
- 测试：`project.store.test.ts` 最终 26 个（Task 3 写，Task 4、5 加）：列表和读 6 个（两份列表、项目自己的读和它被拒绝、列表被拒绝、离开的工作区什么都不给、会话已换：先加载，再把同一个客户端换成会话已换、重取、状态不变）和标识检查 1 个（回答和拒绝，Task 5）；修改 6 个（创建、修改、归档和恢复、删除和离开（Task 4）、加入（Task 4）、侧边栏的位置）、拒绝的 `it.each` 8 行、排队 1 个（含两次连续的移动）；取数与修改的三种交错 4 个（取数不等修改，回答的列表与加载的不同；重取中确认的创建、修改、删除各保留一次；较旧的取数后到时答到列表、答到失败两行都不写）。`use-workspace-fetch.test.ts` 的取数测试加上项目列表（键和 fetcher 的参数）。

### 2.4 加入、离开；权限 store 的项目一半（Task 4；3.4、7.3；M2 交接第 11 节；P14、P18）

- `ProjectStore.joinProject(projectId)`：回答是调用者此刻看到的项目（带他的 `member_role`），写进它所在的列表（原位，没有时在最后）和它自己的读；`leaveProject(project)`：之后 store 不再给出它（两份列表和它自己的读都去掉；公开项目在下一次取数时显示为不是成员）。两者与其余修改同一个队列（第 3 节第 4 条）。
- `UserPermissionStore(rootStore)`（`core/store/user/permissions.store.ts`，重写）只读两个 store：工作区角色是列表给出的 `Workspace.role`（P8a），项目角色是项目 store 给出的 `Project.member_role`。`getProjectRoleByWorkspaceSlugAndProjectId(slug, projectId)`：项目属于这个工作区、调用者是成员时给出角色，工作区管理员给出管理员（PM+WA 是管理员；WA- 什么都没有，P18）；项目、工作区不在 store 中时什么都不给。权限 store 自己什么都不存，所以离开或删除工作区之后它的项目没有角色，同一个 slug 重建的工作区也没有旧项目的角色（P14）。`projectUserInfo`、`workspaceProjectsPermissions`、`fetchUserProjectInfo`、`fetchUserProjectPermissions`、`getProjectRolesByWorkspaceSlug`、`joinProject`、`leaveProject` 和它的 `WorkspaceService` 字段删除（P8a spec 第 3 节第 4 条关闭）。`UserStore.projectsWithCreatePermissions` 从 `joinedProjectIds` 和项目角色算出。
- 删除：`UserService.joinProject`、`leaveProject`；`WorkspaceService.getWorkspaceUserProjectsRole`；`ProjectMemberService.projectMemberMe` 和模块级的 `projectMemberService`；`IUserProjectsRole`；键 `WORKSPACE_PROJECTS_ROLES_INFORMATION`、`PROJECT_ME_INFORMATION`；两个包装层取项目角色和 `project-members/me` 的 `useSWR`。关键词规则 `project-invitations` 的例外删除（M1-P3 的项目成员一条关闭）。
- 测试：`permissions.store.test.ts` 9 → 20 个：`UserPermissionStore, in a project` 的 `allows $who what nerve allows in it, in the project named or in the address's`（9.2 的项目一列：PA、PM、PG、PM+WA、WA-、WM-公、WM-私、WG-、P-前，9 行；每行核对管理员、成员、任何人、只有访客四组，在给出的项目和地址的项目中各一次）；经别的工作区不给角色；离开的工作区和同一个 slug 重建的工作区都不给（P14）。`project.store.test.ts` 加入（先读了它自己的读，加入之后两处都是回答）、离开、拒绝的两行、排队。

### 2.5 调用者在项目里的标签栏；标识检查；旧 `ProjectService`（Task 5；3.18、7.3；P4b 的交接；Codex M-3）

- `ProjectPreferencesStore(projectOf, api)`（`core/store/project/preferences.store.ts`，`ProjectRootStore.preferences`）：每个项目的标签栏按项目的 id（`ReconciledByKey`，一份文档）；`getNavigation(projectId)`（项目 store 不再给出这个项目时什么都不给）、`fetchNavigation`、`updateNavigation`（经 `changes`，nerve 整份替换，写入回答）。设置的另一个字段 `sort_order` 由项目 store 持有、经 `updateProjectSortOrder` 修改（`Project.sort_order` 也带着它，第 3 节第 13 条）。`useProjectPreferences()`；`useTabPreferences(projectId)` 读写新 store，键是生成的 `ProjectTab`，Plane 先改再回滚的写法删除。
- `ProjectStore.checkProjectIdentifier(slug, identifier)`；项目设置页的标识检查改调它，`project/form.tsx` 的模块级 `ProjectService` 删除（M2 交接第 3 节的一处）。
- `core/services/project/project.service.ts` 只留 M4、M6、M7 的三个方法：`getProjectUserProperties`、`updateProjectUserProperties`（M4 的工作项筛选）、`projectIssuesSearch`（M4、M6、M7 的工作项搜索；Codex M-3）。成员 store 的 `fetchProjectUserProperties`、`updateProjectUserProperties`、`projectUserPropertiesMap`，`IProjectMemberNavigationPreferences`、`IProjectUserPropertiesResponse` 删除。
- 测试：`preferences.store.test.ts` 10 个（取到；拒绝，没有时不留、已有时保留；会话已换，先加载再重取；写入回答，回答与请求不同；拒绝的修改；两个项目分开；排队；取数不等修改；两种交错，一份文档没有"已列出的又被创建"，文件里说明）；`project.store.test.ts` 的标识检查（回答和拒绝，`asks nerve whether an identifier is free, and fails when nerve refuses`）。

### 2.6 项目成员（Task 6；7.2、7.3；M2 交接第 3 节）

- `ProjectMembersService(api)`：`list`、`add`（`ProjectMembersAdd`，回答是加入的成员关系）、`update`、`remove`；旧的 `project-member.service.ts` 删除。
- `ProjectMemberStore(memberRoot: Pick<IMemberRootStore, "memberMap">, rootStore, api)`（重写）：每个项目的成员关系按成员的账户 id 存，项目按它的 id 存（`ReconciledByKey`）；项目 store 不再给出这个项目时什么都不给。`projectMemberIds`、`getProjectMemberDetails(userId, projectId)`（`IProjectMemberDetails = ProjectMember & { member: MemberUser }`，公开资料来自工作区成员）、`getProjectMemberIds(projectId, includeGuestUsers)`（调用者在最前，其余按显示名）、`getFilteredProjectMemberDetails`；`fetchProjectMembers(projectId)`；修改经 `changes`、写入回答：`bulkAddMembersToProject`、`updateMemberRole`、`removeMemberFromProject`；store 没有列出这个人时不问 nerve、直接失败。修改改到的项目一侧（`member_ids`、调用者自己的 `member_role`）经 `ProjectStore.confirmProject` 写到项目 store 显示它的每一处，也重放到取数在外的回答上。Plane 的乐观修改和回滚删除。
- `TProjectMembership`、`IProjectBulkAddFormData`、`RowData` 删除；成员行直接用 `IProjectMemberDetails`，角色是 `ProjectMember.role`；加成员的表单值是 `ProjectMembersAdd`。成员下拉框没有调用方传的 `memberIds` 删除（7.9）。
- 测试：`project-member.store.test.ts` 16 个：成员 4 个（列出，连同公开资料；离开的项目不给；拒绝；会话已换）；修改：加入、改角色（连同调用者自己的角色；回答的 `created_at` 与列表的不同）、移出，拒绝的 `it.each` 3 行，没有列出的成员的 `it.each` 2 行，排队，取数不等修改；两种交错（成员关系按成员存，"已列出的又被加入"不会列两次，文件里说明）。

### 2.7 M7 的部分先走；标签的页面清零（Task 7；3.1、3.16、3.17、7.3）

- 收集箱的分诊状态删除：`dropdowns/intake-state/`、`StateStore` 的六个分诊成员、旧状态 service 的 `getIntakeState`、`packages/types/src/intake/`、`@nerve/propel` 的 `IntakeStateGroupIcon`、`TriageGroupIcon` 和它们的类型、颜色。收集箱的工作项：已接受的显示状态下拉框，未接受的不显示状态；创建弹窗没有状态一项（第 3 节第 6 条）。
- 工作区级的标签删除：`LabelStore` 的四个工作区级成员、旧标签 service 的 `getWorkspaceIssueLabels`、键 `WORKSPACE_LABELS`；`useWorkspaceIssueProperties` 不再取它；工作区级的工作项筛选没有标签一项，按标签分组只有"无"（M7 加回）。
- 标签的设置页和工作项的标签选择器在 Task 9 有手改，它们的 oxlint 警告在这里先清零（`then` 的回调不再返回值、遮蔽的变量改名、点击的 `p`/`li` 改为 `button`、只收集按键的根加 `role="presentation"`，`@nerve/ui` 的 `ComboBox` 加 `role` 这个 prop、`useCallback` 的依赖补齐），行为不变；web 的上限 406 → 370。没有新的 vitest。

### 2.8 状态（Task 8；3.17、7.1、7.2、7.3；P7a 的交接）

- 生成的 `State`、`StateGroup` 取代 `IState`、`TStateGroups`（机械步骤 28 个文件，其中 9 个另有手改）；`packages/types/src/state.ts` 只留 `TStateOperationsCallbacks`。
- `StatesService(api)`：`list`、`listInWorkspace`、`create`、`update`、`delete`、`markDefault`；旧的 `project-state.service.ts` 删除。
- `StateStore(rootStore, api)`（重写）：每个项目的状态按项目的 id，每个工作区的（调用者是成员的项目的状态）按工作区的 id，两者都是 `ReconciledByKey`；项目 store 不再给出的项目的状态什么都不给。顺序是 `sortStates`（组在 `STATE_GROUPS` 中的顺序，再按 `sequence`）；nerve 不给 `order`，组中的位置由顺序算出：`getStatePercentageInGroup(stateId)`（组中最后一个是 100）。`stateMap`、`workspaceStates`、`projectStates`、`groupedProjectStates`、`getStateById`、`getProjectStates`（先给项目自己的列表，没有时给工作区列表中这个项目的）、`getProjectStateIds`；`fetchProjectStates`、`fetchWorkspaceStates(workspace)`；修改经 `changes`、写入回答、在两份列表中都改：`createState`、`updateState`（移动就是改 `group`、`sequence`）、`deleteState`、`markStateAsDefault`（原来的默认不再是默认）；store 没有这个状态时删除和设默认不问 nerve、直接失败。
- `ReconciledByKey.values()`（`reconciled.ts`，扩展、不复制）：已取的每个键的值，`stateMap` 用它。
- `sortStates`（`@nerve/utils`）用 `toSorted`，不改给它的数组；`utils` 的上限 10 → 9。
- 页面：状态设置页（`project-states/root.tsx`）经 `useSessionSWR(["PROJECT_STATES", projectId])` 取，拖动排序发出 `updateState(stateId, { group, sequence })`、在回答之后显示（W17）；状态图标的百分比读 `getStatePercentageInGroup`。
- 挂载时：`useWorkspaceFetch` 在列表中有这个工作区之后取 `["WORKSPACE_STATES", id, slug]`；工作区包装层原来按 slug 取（不是成员的工作区也发出）的 `useSWR` 和键删除，包装层从此不从 `swr` 导入值，加进 `.oxlintrc.json` 的 `no-restricted-imports` 范围（P8a review 第 6 节）。
- 测试：`state.store.test.ts` 17 个：状态 4 个（按组再按 `sequence` 排、组中的位置，nerve 列出的次序打乱，完成组的 `sequence` 小于进行中的两个；工作区列表的后备和离开的项目；拒绝，项目的和工作区的两份列表都有；会话已换）；修改 3 个（创建排在项目的最后；移动在两份列表中都改；删除和设默认）、拒绝的 `it.each` 4 行、没有的状态的 `it.each` 2 行、排队（含两次连续的移动）、取数不等修改；两种交错（重取已列出的创建只列一次，确认的修改保留；较旧的取数后到时不写）。`use-workspace-fetch.test.ts` 的取数测试加上工作区的状态。

### 2.9 标签（Task 9；3.16、7.2、7.3；P7b 的交接）

- 生成的 `Label` 取代 `IIssueLabel`（机械步骤 22 个文件，其中 11 个另有手改）；`IIssueLabelTree` 和 `@nerve/utils` 的 `buildTree` 删除。
- `LabelsService(api)`：`list`、`create`、`update`、`delete`（归档的项目也列得出，P7b）；旧的 `issue_label.service.ts` 删除。
- `LabelStore(rootStore, api)`（重写）：每个项目的标签按项目的 id（`ReconciledByKey`），项目 store 不再给出的项目的标签什么都不给；只按项目取。顺序是 nerve 的：`sort_order`，再按 id；两层由 `parent_id` 算出：`projectLabelsTree` 是顶层的标签，各带它下面的标签。`labelMap`、`projectLabels`、`getLabelById`、`getProjectLabels`、`getProjectLabelIds`；`fetchProjectLabels`；修改经 `changes`、写入回答：`createLabel(projectId, data)`（不发 `sort_order`，顶层不发 `parent_id: null`，P7b）、`updateLabel`、`updateLabelPosition(labelId, parentId, droppedOnId, dropAtEndOfList)`（位置在轮到它发出时，从 nerve 最近一次回答的标签算出，第 3 节第 15 条）、`deleteLabel`（它下面的标签一并去掉）；store 没有这个标签时移动和删除不问 nerve、直接失败。
- 使用方：设置页和选择器改调新签名；被拒绝的移动和"移出分组"提示失败（原来什么都不提示，W17）。`@nerve/utils` 的 `array.ts` 两处 `[...options].sort` 改为 `toSorted`；`utils` 的上限 9 → 7。
- 测试：`label.store.test.ts` 23 个：标签 4 个（顺序和两层；每个项目分开、离开的项目不给；拒绝；会话已换）；修改：创建（顶层排在最后，或在页面给的父标签下）、修改和移到顶层、移动的 `it.each` 5 行（父标签下的第一个之前、顶层的两个之间、顶层标签下的最后、列表的末尾、没有同级的父标签下）、放回原父标签下不发、删除连带子标签、拒绝的 `it.each` 4 行、没有的标签的 `it.each` 2 行、排队（移动的位置从前一个回答算）、取数不等修改；两种交错。

### 2.10 项目包装层（Task 10；3.19、7.1、7.6、8.3；Codex 4.3 第 1 条的后一半）

- `useProjectFetch(projectId): ProjectAccess`（`core/layouts/auth-layout/use-project-fetch.ts`）：项目一侧挂载时的取数和"这个项目对调用者是什么"这唯一的判断，`kind` 标记的联合：`unavailable`（带重取的 `retry`）、`loading`、`not-found`、`not-member`（带项目）、`member`（带项目）。先取 nerve 对项目的读（`["PROJECT", projectId]`）；判断按这个顺序：读被拒绝（码是 `project.not_found` → `not-found`，其余 → `unavailable`；nerve 先前答过也一样）、还没有回答（`loading`）、项目 store 此刻不再给出它（删除、离开 → `not-found`）、`member_role` 为 `null`（`not-member`）、其余（`member`）。只在 `member` 时再取标签栏、标签、成员、状态（`["PROJECT_PREFERENCES", id]`、`["PROJECT_LABELS", id]`、`["PROJECT_MEMBERS", id]`、`["PROJECT_STATES", id]`），条件由 hook 从 nerve 的读和项目 store 算出，调用方传不进来（W5）。
- `ProjectAuthWrapper({ projectId, children })`（重写）只照它渲染：`loading` 什么都不显示；`unavailable` 显示 `SessionUnavailable`；`not-member` 显示加入的界面（`joinProject` 经项目 store 的队列）；`not-found` 显示"找不到项目"；`member` 渲染页面。Plane 按 HTTP 状态码（403、409）和工作区管理员判断的分支、"无权访问"的界面和它的文案 `project_empty_state.no_access.restricted_description`（两种语言）删除；`ProjectAccessRestriction` 收 `canJoin`。三个布局不再传 `workspaceSlug`；项目设置的成员默认值不再自己取项目。
- `fake-store-hooks.ts` 加项目一侧的 hook；`.oxlintrc.json` 把项目包装层、`use-project-fetch.ts` 加进两条规则的范围（P8a review 第 6 节：两个包装层都在 A6 的范围内）。
- 测试：`use-project-fetch.test.ts` 14 个：成员的五个取数（键和 fetcher 的参数）；只取项目的 `it.each` 4 行（看得到不是成员、nerve 还没有读而工作区的列表有它、找不到、对成员重取时取不到）；包装层的判断 `it.each` 8 行（第一次就取不到、重取时取不到而 store 仍有它、找不到、还没有回答、store 不再给出、看得到不是成员、是访客、是成员）；重取。"第一次就取不到"和"重取时取不到"两行守着判断的顺序（P8a 的教训）。

### 2.11 关键词、S2、项目筛选的死行、文档（Task 11；7.10、第 2 节 S2、3.20）

- `plane-workspace-urls` 的模式补上项目一侧的旧地址（`/project-roles/`、项目的列出和创建、`projects/details/`、一个项目和它的 `archive/`、`members/`、`project-members/me/`、`states/`、`issue-labels/`、`user-properties/`，工作区的 `project-identifiers`、`states/`、`labels/`）；精确例外一条：`project.service.ts` 中项目的 `user-properties/` 两处，`until: "M4"`（7.10）。`plane-user-urls` 收紧为整个 `/api/users/`（M2 交接第 11 节）。规则 64 条不变；例外 3 → 2（Task 4）→ 3（附录 A.11）。
- S2（`e2e/stories/smoke/s2-web-app.spec.ts`）：未登录的两个测试不变；加四个账户各一个：工作区的管理员、成员、访客（三人都是项目的成员，访客以访客的角色），和工作区的成员而不是项目成员的人。每个账户登录、落点到工作区的首页，再打开项目的设置页；每一页加载时的 API 请求（去重、排序，slug 和项目的 id 换成名字）必须恰好等于给它的清单：应用启动的五个、工作区包装层的四个、项目的读，和只有项目成员才有的项目自己的四个（加上成员看到的 general 页的时区）。两张清单都齐了之后再没有请求；没有失败的请求、没有 `/api/v0` 之外的请求、没有 CSP 违规、没有页面错误，控制台只有两次加载各一条已知的警告（第 3 节第 5 条）。
- `project_filter.store.ts` 的死行（P8a spec 第 5 节）：`getDisplayFiltersByWorkspaceSlug`、`getFiltersByWorkspaceSlug` 和接口上的 `displayFilters`、`filters` 删除；`store-context.test.ts` 改读 `currentWorkspaceDisplayFilters`。
- 文档：总体设计 7.7 的三处（`.oxlintrc.json` 的范围加上 P8b 的文件；"页面按权限决定取数"加项目包装层的例子；`reconciled.ts` 的使用者加上 P8b 的 store 和 `values()`）；前端改动清单 3.1 的 M3 一行、错误格式一行，3.2 加五行（项目封面的上传、项目的收藏、收集箱的分诊状态、工作区级的标签、项目一侧的挂载时取数）。

### 2.12 挂载时的取数和它们的条件（W5）

| 取数 | 键 | 何时取 | 在哪里 |
|---|---|---|---|
| 工作区的项目（未归档） | `["PROJECTS", loginId, id, slug]` | 调用者的列表中有这个工作区 | `useWorkspaceFetch`（工作区的每一页） |
| 工作区的状态 | `["WORKSPACE_STATES", loginId, id, slug]` | 同上 | `useWorkspaceFetch` |
| 工作区的已归档项目 | `["ARCHIVED_PROJECTS", loginId, id, slug]` | 列表中有地址的工作区（`currentWorkspace`） | 项目页 `components/projects/page.tsx`（在工作区包装层之内） |
| 项目的读 | `["PROJECT", loginId, projectId]` | 已登录，地址有项目 | `useProjectFetch`（项目的每一页） |
| 调用者在项目里的标签栏 | `["PROJECT_PREFERENCES", loginId, projectId]` | nerve 的读说调用者是成员（`member_role` 不为 `null`），且项目 store 仍给出它 | `useProjectFetch` |
| 项目的标签 | `["PROJECT_LABELS", loginId, projectId]` | 同上 | `useProjectFetch` |
| 项目的成员 | `["PROJECT_MEMBERS", loginId, projectId]` | 同上 | `useProjectFetch` |
| 项目的状态 | `["PROJECT_STATES", loginId, projectId]` | 同上 | `useProjectFetch`；状态设置页 `project-states/root.tsx` 用同一个键（它只在包装层渲染页面时挂载，即只对成员） |

每个取数都经 `useSessionSWR`，SWR 的配置只有它的一份（P8a 2.1）；绕过它的会话取数由根目录 `.oxlintrc.json` 的 `no-restricted-imports` 发现（附录 A.6；W4 的四个变异）。"何时取"的条件都由取数的 hook 自己从 store 和 nerve 的回答算出，调用方只传地址。工作区一级的键带工作区的 id（P8a 的 F-6），项目一级的带项目的 id；请求的地址照契约（工作区的按 slug，项目的按 id）。

不在挂载路径上、仍按需取的（M4 的工作项页面，第 5 节）：状态、成员、标签的下拉框打开时取（`dropdowns/state/dropdown.tsx`、`dropdowns/member/dropdown.tsx`、`issues/select/dropdown.tsx` 等），子工作项的 store 取别的项目的状态、成员、标签；通知页的收集箱项经原来的 `useSWR` 读项目（M7）。

### 2.13 修改的排队（W6）

| store | 经 `oneAtATime()` 的修改（每个 store 一个队列） | 不排队的读 |
|---|---|---|
| `ProjectStore` | `createProject`、`updateProject`、`deleteProject`、`archiveProject`、`restoreProject`、`updateProjectSortOrder`、`joinProject`、`leaveProject` | `fetchProjects`、`fetchArchivedProjects`、`fetchProject`、`checkProjectIdentifier` |
| `ProjectPreferencesStore` | `updateNavigation` | `fetchNavigation` |
| `ProjectMemberStore` | `bulkAddMembersToProject`、`updateMemberRole`、`removeMemberFromProject` | `fetchProjectMembers` |
| `StateStore` | `createState`、`updateState`（拖动排序）、`deleteState`、`markStateAsDefault` | `fetchProjectStates`、`fetchWorkspaceStates` |
| `LabelStore` | `createLabel`、`updateLabel`、`updateLabelPosition`（拖动排序）、`deleteLabel` | `fetchProjectLabels` |

每个队列有"上一个有了回答才发下一个，被拒绝也一样"的 vitest（P8a 的 `inTurn`），表中的每个修改都在它的队列的这个测试里；三处拖动排序（侧边栏的项目、状态、标签）的测试是两次连续的移动（9.5）。标签的移动在轮到它发出时才算位置，所以第二次移动从第一次的回答算起（第 3 节第 15 条）。"取数不等修改"的 vitest 五个 store 各有一个，共用 `fetchedWhileChangeIsOut`，回答的列表与加载的不同。W6 的 10 个变异（每个 store 绕过队列、取数进队列）都由 vitest 发现（附录 A.2）。`ProjectStore.confirmProject` 不是修改：它把项目成员 store 的修改的回答写到项目上（2.6），由成员的队列排着。

## 3. 与设计的差异、补充和需要裁定的

标"**裁定**"的各条请控制者裁定；其余是本 Phase 在设计之内做的决定和它们的理由。

1. **裁定：Task 的切分**：设计 12 节 P8b 的草稿是 9 个任务，plan 是 11 个（P7、P8a 的先例："在上限之内重新切分任务"）。对应：设计 1（类型和删除）→ Task 1（生成的 `Project` 没有的先删，之后的 Task 不必再改它们的使用方）、Task 2（类型的替换，机械步骤）；2 → 3；3 → 4（另加加入、离开，第 4 条）；4 → 6（加入、离开已在 Task 4）；5 → 8（分诊状态的删除提前到 Task 7）；6 → 9（工作区级的标签的删除提前到 Task 7，标签页面的清零也在 Task 7，否则 Task 9 超过上限）；7 → 5（标签栏）、10（项目包装层：它读 Task 5、6、8、9 的四个 store，放在它们之后）；8（oxlint）不单独成为任务：每个 Task 把自己有手改的文件清零、调低上限（R3，附录 A.8）；9 → 5（旧 `ProjectService` 的三个方法随标签栏和标识检查）、11（关键词、S2、3.20）。11 个 Task 都在约 16 个之内。
2. **裁定：两个 Task 在约 1,500 行上下**：Task 3 是 1,559 行（其中代码块 1,374 行：项目 store 和它的 26 个测试的大部分在这里）、Task 8 是 1,504 行（附录 A.10）；其余都在 1,310 行之内。Task 3 比约 1,500 行多 4%、Task 8 多 4 行，与 P8a 的 Task 4（1,567 行，裁定 A1 认可）相当。拆它要把项目 store 的修改和列表分到两个 Task，中间的树上有一半的项目 store 是 Plane 的写法、一半是新的，过渡版本更多，故没有拆。
3. **裁定：项目封面的上传、预设和随机封面在 P8b 删除**（Task 1；设计 7.8 和 12 节 P10 的任务 3 原写在 P10，P8a spec 第 5 节 P10 一行照录）：生成的 `ProjectCreate`、`ProjectUpdate` 没有封面（3.2 的规则 2），Task 2、3 把创建弹窗和设置页换成生成的类型和新 store 时，上传、随机给封面的步骤无法再编译，与 P8a 的工作区图标（裁定 A2）同理。删掉它是 P10 一项的提前，P10 少这一项；28 张预设封面只为选择器和随机封面存在，一并删除（brief："删除完整，含孤立的资源"），默认的 `image_1.webp` 留给"没有封面时显示默认图"（3.2）。裁定之后，设计 7.8 的"项目封面的上传控件和随机封面（3.2，P10）"和 12 节 P10 的任务 3 照此改（由修订提交或收尾改）。
4. **裁定（小）：加入、离开放在项目 store**：设计 12 节 P8b 的任务 4 把它们写在项目成员的 store，`user.service.ts` 的两个方法"改用生成的客户端"。P8b 把它们放进 `ProjectsService.join`、`leave` 和 `ProjectStore.joinProject`、`leaveProject`（经项目的队列）：加入的回答是 `Project`（带调用者的 `member_role`），改的是调用者看到的项目列表和它自己的读，与 P8a 把接受邀请、离开工作区放进 `WorkspaceRootStore`（P8a spec 第 3 节第 10、23 条）同理；成员 store 是项目管理员的。`UserService.joinProject`、`leaveProject` 删除。
5. **裁定：S2 断言确切的挂载清单，不从关键词规则推出旧地址的模式**（W19 的写法）：brief 的 W19 要 S2"从关键词规则推出旧接口的模式，不另写一份"。关键词规则的模式写的是源码（`` `/api/workspaces/${…}/` `` 这样的模板字符串），不是请求的地址，推出来要另写一层转换，而且只能发现规则点名的 M3 的旧地址；M6、M7 的取数回到挂载路径（P8a review 第 6 节点名的一类）不在规则里。P8b 的 S2 让每个账户的每一页的请求恰好等于给它的清单（2.11），加上 `watchPage` 的"没有 `/api/v0` 之外的请求"：一个旧地址、一个 M6 或 M7 的地址、一个多出来的 `/api/v0` 请求（例如给不是成员的人取了子资源）、一个少了的取数，都让它失败；没有第二份旧地址的模式。变异：`t11-cycles-back`、`t11-unread-back`（P8a 存活的两个）、`t11-old-members-back`（旧的项目成员地址回到挂载路径，关键词守卫也发现）、`t10-pf-non-member`、`t10-pw-children-non-member` 都由 S2 发现（附录 A.2）。代价：清单是 S2 自己的一份，挂载路径以后每加一个取数（P9–P11、M4–M7）都要改这张清单；这正是它要守住的。
6. **裁定：M7 的分诊状态和工作区级的标签从页面删除的范围**（Task 7；3.1、3.16、3.17 只写"从 state store 删除""取数删除"）：store 的成员一删，读它们的组件不能编译。P8b 删到没有读者为止：分诊状态的下拉框、它的类型和 `@nerve/propel` 的两个图标（只有它们用）；收集箱的工作项上，已接受的显示原来的状态下拉框，未接受的（原来显示分诊状态）不显示状态，创建弹窗没有状态一项；工作区级的工作项筛选没有标签一项，按标签分组只有"无"。这些都是 M7 的页面，M7 随各自的接口加回（第 5 节）。看得到的不同列在第 21 条。
7. **关键词规则在 Task 11 一次补全**：P8a 在每个 Task 加上那个 Task 删掉的地址；P8b 的项目一侧的地址在 Task 3–9 逐个消失，模式在 Task 11 一次写全（`plane-workspace-urls` 的一个正则，2.11），`project-invitations` 的例外在 Task 4 删除。Task 3–10 之间一个旧地址若被加回，由 `tsc`、knip（旧 service 的方法已删除）和各 Task 的 vitest 发现，关键词守卫从 Task 11 起发现（`t11-old-members-back`）。分几次写这个正则只多 plan 的行数，不多一种检查。
8. **裁定：三个 Plane 文件变长**（brief："只为使用方改到的 Plane 文件不变长"）：`issues/issue-layouts/utils.tsx` 808 → 812（状态图标读组中的位置，`getStatePercentageInGroup(state.id)` 比 `state.order` 长，格式化把一行拆成五行）；`power-k/config/navigation/commands.ts` 465 → 466（两个不读闭包的条件移出 hook，清掉 `consistent-function-scoping`，多一个空行）；只经机械步骤改到的 `power-k/ui/pages/context-based/work-item/commands.ts` 414 → 415（`rename_types.py` 把 `Label` 另起一行导入）。都是格式，不是加逻辑；拆这几个 Plane 文件不在 P8b 的范围（附录 A.10）。
9. **裁定：`project.store.test.ts` 是 422 行**（brief："文件在约 400 行以内，测试也一样"）：26 个测试覆盖两份列表、项目自己的读、八种修改和三种交错；拆成两个文件要把加载和 `fakeRoot` 的准备写两份，或再加一个测试的共用部分。按"约 400 行"计，没有拆。其余新文件都在 362 行以内。
10. **过渡：Task 3 到 Task 5 之间，项目成员的修改不写项目的 `member_ids`**：Plane 的成员 store 改成员时写 `projectMap[…].members`；Task 3 的项目 store 没有 `projectMap` 了，成员 store 在 Task 6 重写之前没有写回的地方。这三个 Task 的树上，加入、移出成员之后项目卡片的成员数要到下一次取数才变；Task 6 的 `confirmProject` 把成员修改的回答写到项目 store 显示这个项目的每一处，也重放到取数在外的回答上（`t6-pm-add-project`、`t6-pm-remove-project`、`t6-pm-own-role`）。这期间没有页面的故事加入。
11. **裁定：一个性质只从下一个 Task 起由检查发现**：`t10-pw-children-non-member`（项目包装层给不是成员的人渲染页面）。包装层的判断在 `useProjectFetch`，由 vitest 守着（`t10-pf-all-members` 等）；但包装层把 `not-member` 照判断渲染成加入的界面这一步只有渲染它的检查能发现：web 的 vitest 在 node 环境里运行，渲染不了组件（P8a spec 第 3 节第 6 条），能渲染它的是 S2，S2 在 Task 11 改写。所以 Task 10 的提交到 Task 11 的提交之间，这一步只有评审能发现；合并时（Task 11 之后）由 S2 守着（不是项目成员的人打开项目设置，清单上只有项目的读，没有子资源）。另一种做法：把 S2 的改写并进 Task 10（约 604 + 260 行，仍在上限之内），代价是 Task 10 既改包装层又改故事。请裁定是否照另一种做法改。
12. **修改写入回答，不乐观**（与 P8a 第 7 条同）：Plane 改项目、标签栏、成员角色、状态和标签的位置时先改 store、失败再回滚；P8b 的 store 等 nerve 的回答、写入回答。页面上的差别是改动在回答之后才显示（第 21 条）。
13. **store 的形状**：项目一侧的每个 store 按 id 存（`ReconciledByKey`：项目的两份列表和工作区的状态按工作区的 id，项目自己的读、标签栏、成员、状态、标签按项目的 id），每次取数整份换掉，带上取数在外时 nerve 确认的修改；项目一侧的子 store 读之前先问项目 store 是否仍给出这个项目（`getProjectById`），项目 store 再问工作区 store 是否仍有它的工作区。P14 由此成立：离开、删除的项目，和调用者已离开或删除的工作区（包括之后同一个 slug 新建的）里的项目，它们的状态留到这一代结束但读不到（与 P8a 的 F-6 同）。侧边栏的 `sort_order` 是调用者在项目里的设置（`ProjectPreferences.sort_order`），nerve 也在 `Project.sort_order` 中给出它，所以由项目 store 持有，修改发到设置的地址、写回答的 `sort_order`；标签栏是设置的另一个字段，由 `ProjectPreferencesStore` 持有（2.5）。
14. **`reconciled.ts` 加 `values()`**：状态 store 的 `stateMap` 要在项目和工作区两份列表中按 id 找状态；`ReconciledByKey.values()` 给出已取的每个键的值。按总体设计 7.7"扩展，不复制"（`t8-rc-upserted-twice` 守着 `upserted` 在重放时不重复）。
15. **标签的移动在轮到它发出时才算位置**：拖动给出"放在哪个标签上、在它之前还是之后、新的父标签"，`sort_order` 要从同级的标签算出。若在调用时算，连续两次拖动时第二次用的是第一次回答之前的顺序；P8b 在队列轮到它时，从 nerve 最近一次回答的标签算（`t9-lb-no-queue`）。规则：在新的同级之间取两邻的中点；放在第一个之前取它的一半；放到最后取最后一个加 10000（与服务端 `SortOrderAfter` 的步长相同，3.16）；新父标签下没有同级时只发 `parent_id`；放回原父标签下、没有放在某个标签上时什么都不发。
16. **状态在组中的位置由顺序算出**（P7a 的交接）：nerve 不给 `order`；`sortStates` 按组在 `STATE_GROUPS` 中的顺序、再按 `sequence` 排，`getStatePercentageInGroup` 是它在组中的位置（组中最后一个是 100），状态图标读它。拖动排序发出 `{ group, sequence }`，由 nerve 决定（P7a）。
17. **`useProjectFetch` 是新加的 hook，判断的顺序是固定的**（P8a 的 `useWorkspaceFetch` 同样，P8a spec 第 3 节第 13 条）：读被拒绝先于"还没有回答"（否则 nerve 连不上时包装层一直空白，没有重取；`t10-pf-no-retry` 和"第一次就取不到"一行），nerve 的读先于项目 store 的副本（否则显示 store 中旧的成员身份：`t10-pf-no-loading`、`t10-pf-stale-member`），项目 store 是否仍给出先于 `member_role`（删除、离开之后显示"找不到项目"：`t10-pf-store-gone-waits`）。码只认 `project.not_found`（`t10-pf-code`、`t10-pf-any-refusal-not-found`）：其余的拒绝（包括 nerve 连不上）是 `unavailable`，可以重取。
18. **"找不到项目"包括私密项目的非成员**（8.3）：nerve 对看不到的项目答 404 `project.not_found`，与不存在的不分；页面照它显示"找不到项目"。附录 A.5 的清单中，成员、访客、不是项目成员的人打开私密项目 `ops` 时那一个 404 就是它，不是旧接口。
19. **旧 `ProjectService` 只留三个方法**（7.3、Codex M-3）：`getProjectUserProperties`、`updateProjectUserProperties`（M4 的工作项筛选，关键词规则的 `until: "M4"` 例外）、`projectIssuesSearch`（M4、M6、M7 的工作项搜索，它的三个调用方各有一个模块级实例：M2 交接第 3 节的计数中它们属于 M4–M7）。
20. **`@nerve/propel` 依赖 `@nerve/api-client`**：项目的图标是生成的 `LogoProps`，`Logo` 收它（Plane 的 `TLogoProps` 是 M7 的，形状与它相同）。工作区内的依赖，不是新的 npm 包（与 P8a 的 `@nerve/types`、`@nerve/utils` 同）。
21. **裁定：页面上看得到的不同**（W17；P9–P11 照第 2 节的故事改页面）：
    - 项目一侧现在能显示：`7390e012` 上项目的列表、读和子资源都是 404（附录 A.5），项目页和侧边栏是空的、项目的页面打不开。
    - 封面只显示，没有上传和预设的选择，没有封面的显示默认图；新建项目不再随机给封面（第 3 条）。卡片没有收藏的星标，创建项目没有"加入收藏"；侧边栏的"收集箱"没有数字；列表视图中工作项的键宽按一位数字算。
    - 项目、标签栏、成员角色、状态和标签的拖动在 nerve 回答之后才显示（第 12 条）；侧边栏项目的顺序是 nerve 回答的位置。项目页在已归档的列表也到了之后才显示。
    - 加入项目之后显示 nerve 回答的项目；离开之后它立即从列表中消失（公开项目在下一次取数时显示为不是成员）。
    - 项目包装层：看得到、不是成员（公开项目）→ 加入的界面；找不到（不存在、看不到的私密项目、删除、离开）→"找不到项目"；nerve 连不上 → `SessionUnavailable`，带重取；Plane 的"无权访问"界面没有了。不是成员的人不发任何子资源的请求（原来发出四个、都是 403）。
    - 收集箱的工作项：未接受的不显示状态，创建弹窗没有状态一项；工作区级的工作项筛选没有标签一项，按标签分组只有"无"（第 6 条，M7）。
    - 被拒绝的标签移动和"移出分组"提示失败（原来什么都不提示）。对 store 没有列出的成员、没有的状态和标签，修改不发请求、直接失败。
    - 工作区的状态只在调用者的列表有这个工作区时取（原来对不是成员的工作区也按 slug 发出）；项目设置的成员默认值不再自己取项目（少一个重复的请求）。项目一侧的取数每次挂载都重取，聚焦时不取（P8a 的 F-1）。

**交接到 P8b 的事项的落点**（brief"Carried into P8b"）：

| 来源 | 事项 | 落点 |
|---|---|---|
| P8a spec 第 5 节 | 权限 store 的项目一半（`project-roles`、`project-members/me`）、`joinProject`、`leaveProject`、`workspaceService` 字段；`pf-project-admin-any` 必须被发现 | Task 4；`t4-pf-admin-any` 由 `permissions.store.test.ts` 发现 |
| | `plane-user-urls` 收紧；`project-invitations` 的例外删除 | Task 11；Task 4 |
| | 项目 store 的旧地址；`project/form.tsx` 的模块级 `ProjectService`；`is_favorite`、收藏 store 的项目一支、`intake_count`；`cover_image_url` | Task 3；Task 5；Task 1；Task 2（W20，`t2-cover-null`） |
| | 状态 store：工作区的状态（此刻只按 slug 启用）、项目的状态 | Task 8（按调用者的列表启用） |
| | 项目成员 store：`user-properties`、`members`；`ProjectMemberService` 的模块级实例 | Task 5；Task 6；Task 4 |
| | 标签 store：`issue-labels` | Task 9 |
| | 项目包装层按 `member_role` 启用和它的 vitest；关键词规则补全；两个包装层进 A6 的范围 | Task 10；Task 11；Task 8（工作区包装层）、Task 10（项目包装层） |
| | S2 的改写，发现 M6、M7 的取数回到挂载路径这一类 | Task 11（第 5 条） |
| | P8a 改到的项目一侧文件中 M3 的死行：`add-project-members-modal.tsx` 4 行、`project-member.store.ts` 4 行、`project_filter.store.ts` 5 行、`useProjectColumns.tsx` 1 行 | Task 6、6、11、6（附录 A.9：都已不在） |
| P8a review 第 6 节 | P14：权限 store 的项目一半不能活过离开和删除工作区；每个项目一级的状态按项目的 id | Task 3–9 的按 id 的 store 和 `getProjectById` 的门（第 13 条）；Task 4 的权限 store 自己不存状态，P14 的测试和 `t4-pf-old-workspace`、`t3-ps-left-workspace` |
| | `workspace-wrapper.tsx` 在 `currentWorkspaceInfo` 有缓存时直接渲染子组件 | `7390e012` 上已不在：P8a 的修复轮（F-2）之后包装层只在 `useWorkspaceFetch` 给出 `ready` 时渲染，判断只读调用者此刻的列表；项目一半见上一行 |
| | P18：PM+WA、WA- 的行；杀掉 `pf-project-admin-any` | Task 4（9 行的 `it.each`，`t4-pf-wa-admin`、`t4-pf-wa-no-member`、`t4-pf-admin-any`） |
| | S2 断言每张挂载清单上没有 M6、M7 的地址 | Task 11（确切的清单，第 5 条） |
| | 两个包装层进 A6 的范围 | Task 8、Task 10（`t8-raw-swr-workspace-wrapper`、`t10-raw-swr-wrapper`） |
| | F-1、F-4、F-5、F-6 | Task 3、5、6、8、9、10（附录 A.2 的 W3、W6） |
| M2 收尾第 3 节 | 10 处模块级实例全部消失 | P8a 8 处；Task 4（`ProjectMemberService`）、Task 5（`project/form.tsx` 的 `ProjectService`）：关闭（附录 A.6） |
| M2 收尾第 11 节 | 项目一侧：`joinProject`、`leaveProject`；`plane-user-urls` 收紧 | Task 4；Task 11：关闭 |
| M1-P3 项目成员 | `project-invitations` 的例外删除 | Task 4：关闭 |
| P4b | `ProjectPreferences.navigation`；侧边栏的 `sort_order`；`joinProject` | Task 5；Task 3；Task 4 |
| P7a | 状态的顺序和组中的位置取代 `order` | Task 8（第 16 条） |
| P7b | 标签的类型；只按项目取；建时不发 `sort_order` 和 `null` 的父标签；`listLabels` 在已归档的项目上照常列出，`listStates` 不 | Task 9（`t9-lb-create-sort-order`、`t9-lb-create-null-parent`）；已归档项目的界面在 P10（第 5 节） |
| Codex 设计评审 M-3 | 旧 `ProjectService` 的三个方法留给后续的 M | Task 5（第 19 条） |
| Codex 设计评审 4.3 第 1 条后一半 | 项目包装层等确认是有效的项目成员再取子资源；不是成员的人直达项目没有多余的失败请求 | Task 10；Task 11 的 S2（不是项目成员的账户） |
| Codex 设计评审 4.3 第 2 条 | 删除项目之后的跳转核对会话 | P10（`sessionGuard()` 在 P8a） |
| M1 收尾（7.9） | 改到的文件 0 条 oxlint 警告；上限调低；只经机械步骤到达的列给 P11 | 每个 Task；附录 A.8 |
| 3.20 | 总体设计 7.7；前端改动清单 | Task 11 |
| P8a spec 第 5 节 P10 | 项目封面的上传和随机封面的删除 | Task 1（第 3 条） |

## 4. 验收标准（完成线，M3 设计 12 节 P8b）

- [ ] S1、改写的 S2 和此前的全部故事通过：worktree 中 `make e2e` 74 个全部通过（S2 6 个）。
- [ ] M2 交接第 3 节的 `git grep` 中 M3 的 10 处都已消失（P8a 8 处、P8b 2 处）。
- [ ] `node tools/keywords.mjs` 通过：64 条规则、3 个例外（M3 加的只有 `plane-workspace-urls` 的 `until: "M4"` 一条），没有命中（`make lint-web`）。
- [ ] 9.5 中 P8b 的 vitest 通过：权限 store 的项目一半（9.2 同一组身份，含 PM+WA、WA-）、状态的顺序和组中的位置、拖动排序一个接一个（项目、状态、标签）、`ProjectAuthWrapper` 在 `member_role` 为 `null` 时不取子资源；五个 store 的取数、修改、会话、排队、取数不等修改、取数与修改的交错、按 id 存；每个都有一个发现它的变异（W18，附录 A.2）。
- [ ] 根目录 `.oxlintrc.json` 的两条 `overrides` 覆盖 P8b 的文件：范围内的会话取数从 `swr` 导入值、范围内的文件用非空断言，`check:lint` 都失败（附录 A.2 的 W4、W12 变异）；两个包装层都在范围内。
- [ ] `tsc`、knip 通过。
- [ ] 改到的文件按 7.9 没有 oxlint 警告：有手改的源文件 146 个都是 0 条，没有新的抑制；只经机械步骤到达的 38 个文件的 14 条列在附录 A.8，留给 P11 第 4 个任务；web 的上限 435 → 367，`utils` 12 → 7，其余各包不变。
- [ ] 逐 Task 复现：从 `7390e012` 加上三份文档的树照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同（附录 A.12）。

## 5. 不在 P8b 范围内

- **P9**：工作区的页面（P8a spec 第 5 节 P9 一行照录）。P8b 没有新加的。
- **P10**：项目的页面（第 2 节的故事）；`ProjectAuthWrapper` 中已归档项目的界面（7.6：`archived_at` 有值；成员打开已归档的项目时包装层照常取四个子资源，nerve 对状态答空列表、对标签照常列出，P7b）；删除项目之后的 `sessionGuard()`（Codex 4.3 第 2 条）；项目成员设置的角色下拉框发出字符串的角色（P8a spec 第 3 节第 12 条，`member-columns.tsx` 仍以 `Object.entries(…)` 的键作值）；下拉框、复制到剪贴板、表情选择器（7.7）；项目创建弹窗的页面版本（封面一步已在 Task 1 删除，第 3 节第 3 条）。
- **P11**：只经机械步骤到达的 38 个文件中 5 个的 14 条警告（附录 A.8；其中 `use-work-item-filters-config.tsx` 的 3 条已在 P8a 的清单中）；`--rows M3` 剩下的行（附录 A.9）；核对 M3 加的关键词例外只剩 `until: "M4"` 的一条。
- **M4**：项目的 `user-properties`（工作项筛选，旧 `ProjectService` 的两个方法和关键词的例外）；`projectIssuesSearch` 和它的三个模块级实例（与 M6、M7 共用）；工作项页面中按需的取数照 7.1 按权限启用：状态、成员、标签的下拉框打开时取（`dropdowns/state/dropdown.tsx`、`dropdowns/member/dropdown.tsx`、`issues/select/dropdown.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`issue-detail/label/select/label-select.tsx`），子工作项的 store 取别的项目的状态、成员、标签（`sub_issues.store.ts`，调用者可能不是那些项目的成员）；工作项的键宽读回 `next_work_item_sequence`（Task 1）；`TIssueIdentifierProps.projectIdentifier` 可以是 `undefined`（Task 3：项目未到时）；P8a 交给 M4 的各项照旧。
- **M5**：项目封面的上传（`cover_image_asset_id`，3.2）连同预设封面（Task 1 删除的 28 张）和"新建项目时的封面值"（M5 的 M1-closeout）。
- **M6**：迭代的 `user-properties` 不在关键词规则的模式里（不命中样例）；P8a 交给 M6 的各项照旧。
- **M7**：项目的收藏（卡片的星标、创建时加入收藏、`FavoriteStore` 的项目一支，Task 1）；`intake_count`（侧边栏的数字和收集箱 store 写它的两段）；收集箱的分诊状态（store、下拉框、类型、`@nerve/propel` 的两个图标，Task 7）；工作区级的标签（store 的四个成员、筛选的标签一项、按标签分组，Task 7）；通知页的收集箱项经原来的 `useSWR` 读项目（`workspace-notifications/root.tsx`，Task 4 改调 `fetchProject`）；P8a 交给 M7 的各项照旧。
- **收尾**：核对 3.20 的 P8b 两行（总体设计 7.7、前端改动清单）；设计 7.8、12 节 P10 照第 3 节第 3 条的裁定改。

## 6. 风险

- **S2 的清单是一份要维护的名单**（第 3 节第 5 条）：P9–P11、M4–M7 每在挂载路径上加一个取数，S2 都会失败，直到清单改过。退路：这正是它守的性质；改清单的提交说明为什么多了这个取数，评审照 7.1 核对它的条件。
- **Task 10 到 Task 11 之间，包装层的渲染只有评审守着**（第 3 节第 11 条）。退路：这期间没有页面的故事加入；照裁定可以把 S2 并进 Task 10。
- **M4 的工作项页面仍按需取别的项目的状态、成员、标签**（第 5 节）：调用者不是那些项目的成员时 nerve 答 403，页面上是一次失败的请求。它们不在 M3 的挂载路径上（附录 A.5 的清单中没有）。退路：M4 重写工作项页面时照 7.1 启用。
- **机械步骤依赖 oxfmt 的版本**：Task 9 的散列表在 oxfmt 0.35.0 上量出（Task 2、8 的机械步骤不排版）；锁文件固定它。对不上时先核对 `pnpm exec oxfmt --version`。
- **不是 git 仓库的副本里 S3 失败**（P4b spec F4）：复现在副本中运行，只有 S3 因为读不到提交而失败，由复现脚本单独认出；worktree 中 74 个都要通过。

## 7. 已知的限制、交接和关闭条件

- **关闭**：M2 收尾第 3 节（M3 的 10 处模块级实例全部消失）、第 11 节中项目一侧的部分（`joinProject`、`leaveProject`、`plane-user-urls` 收紧）；M1-P3 的项目成员（`project-invitations` 的例外删除）；P8a review 第 6 节 P8b 一行（P14、P18、S2 的改写、A6 的范围）。
- **交接**：第 5 节各行照录进 P9–P11 的 spec 和 M4–M7 的交接（13.2）；P8b 的 review 第 6 节列出只经机械步骤到达的文件和它们的警告数（R3），以及第 3 节中裁定的结果。
- **限制**：P8b 不改任何页面的新行为（第 2 节的故事在 P9–P11）；项目一侧的状态留到这一代结束但读不到（与 P8a 的 F-6 同）；每次挂载都重取，聚焦时不取（F-1）。

## 附录 A：原型验证记录（2026-10-08）

### A.1 方法与门禁

- 原型 `$M3TMP/p8bproto` 是 `7390e012` 的 `git archive` 副本（`pnpm install --frozen-lockfile` 之后）。每个 Task 由一个脚本从上一个快照写出（`$M3TMP/p8btools/edit/T<n>.py`；Task 2、8、9 先执行 plan 的机械步骤），照仓库的格式化（oxfmt）排版，`tsc` 通过之后存一份快照（`$M3TMP/p8bsnap/T1`…`T11`；Task 2、8、9 另有机械步骤之后的 `T2m`、`T8m`、`T9m`）。plan 的块由快照之间的差异写出（`mkblocks.py`：11 个 Task，0 个问题），机械步骤的文件不写成块。测试或代码改过时，从改动的那个 Task 起重新写出之后的每个快照（`chain.sh`），再重新写出块。
- 最终原型上：`make gen-check`（副本中以 `make gen` 之后逐字节比较代替）0 处差异；`make lint-web` 通过（关键词守卫 64 条规则、3 个例外、没有命中）；`make knip` 通过；`make test-web` 通过（web 应用 39 个测试文件、411 个测试；`7390e012` 上是 33 个、294 个）；`make build` 通过；`make e2e` 74 个中 73 个通过，S3 在副本中失败（F4）。没有 Go 文件改动，`make lint-go`、`make test` 不需要。
- 每个快照上 `make lint-web`、`make knip`、`make test-web` 都跑过；`make e2e` 在逐 Task 复现中每个 Task 都跑（A.12）。
- 逐 Task 复现（`replay.py`）：从 `7390e012` 的副本起，照 plan 先执行机械步骤（核对散列表），再应用块，再执行每个 `Run:`；结果见 A.12。

### A.2 变异

`mutants_p8b.py`：132 个变异，每个只改一处或几处，`mut.py` 在最终原型上、在它写的每个检查上各跑一次；132 个都被发现，没有存活的。"静态"是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，和 `make knip`。按缺陷类别（一个变异可以被几层发现）：

| 类别 | 变异 | 静态 | vitest | 端到端 | 存活 |
|---|---|---|---|---|---|
| W1 旧地址回到挂载路径、关键词规则的宽度 | 4 | 4 | | 1 | |
| W2 挂载时取数的清单 | 3 | | 3 | | |
| W3 分代、按 id 的状态、写入回答、调和 | 58 | | 58 | | |
| W4 会话的取数经 `useSessionSWR`（键带 `loginId`） | 4 | 4 | | | |
| W5 按权限取数 | 12 | | 11 | 2 | |
| W6 一个接一个 | 10 | | 10 | | |
| W9 权限 store 的项目一半 | 10 | | 10 | | |
| W10 状态的顺序和组中的位置 | 3 | | 3 | | |
| W11 标签：两层、nerve 的顺序、按项目、建时的请求体、移动的位置 | 19 | 2 | 17 | | |
| W12 非空断言 | 3 | 3 | | | |
| W13 删除的代码 | 2 | 2 | | | |
| W16 取数失败（重取） | 1 | | 1 | | |
| W19 M6、M7 的取数回到挂载路径 | 2 | | | 2 | |
| W20 可空的字段 | 1 | 1 | | | |

- W3 的 58 个：项目一侧的五个 store 和 `reconciled.ts` 的每一项性质，分为：写入回答而不是请求、在回答之前不改（`t5-pp-request`、`t5-pp-before-answer`、`t6-pm-role-request`、`t6-ps-sort-request` 等）；修改写到每一处（列表、项目自己的读、工作区的列表、项目的 `member_ids` 和调用者的 `member_role`）；按 id、项目 store 不再给出的什么都不给（P14：`t3-ps-left-workspace`、`t5-pp-left`、`t6-pm-left`、`t8-st-left`）；每个取数、修改被拒绝时失败（第 19 类清扫的 13 个，A.3）；调和（`t3-rc-*` 5 个、`t5-rc-one-entry`、`t8-rc-upserted-twice`：改的是 P8a 的 `reconciled.ts`，由 P8b 的五个 store 测试发现）；`t11-filter-dispose`（`RootStore.dispose()` 释放项目筛选的反应，`store-context.test.ts`）。
- W5 的 12 个：`useWorkspaceFetch` 在列表没有这个工作区时也取项目（`t3-wf-projects-unlisted`）；`useProjectFetch` 的每一项判断和条件（`t10-pf-*` 10 个：不是成员也取四个子资源、读到之前就取、显示旧的成员身份、任何拒绝都算找不到、码不是 nerve 的、store 不再给出时一直等、访客算不是成员、看得到的都算成员、成员都算不是成员、读到之前用 store 的副本）；`t10-pw-children-non-member`（包装层给不是成员的人渲染页面，只由 S2 发现，第 3 节第 11 条）。`t10-pf-non-member` 由 vitest 和 S2 两层发现。
- W9 的 10 个：每个角色一个（PA、PM、PG、PM+WA、WA-、WM-公）；经别的工作区；没有列出的项目按 Plane 的兜底给访客；调用者离开的工作区的项目在同一个 slug 重建的工作区中给出角色（P14）；任何工作区角色都算项目管理员（`t4-pf-admin-any`，P8a 的 `pf-project-admin-any`，P18）。都由 `permissions.store.test.ts` 发现。
- W4、W12 的 7 个由 `.oxlintrc.json` 的两条 `overrides` 发现（`check:lint` 报"`swr` import is restricted"或"Forbidden non-null assertion"）：项目的 hook 经 `useSWR` 取标签、项目包装层自己经 `useSWR` 取项目、状态设置页经 `useSWR` 取状态、工作区包装层又经 `useSWR` 取工作区的状态（P8a review 第 6 节：两个包装层都在范围内）；项目 store、状态 store、项目的 hook 各一处 `!`。
- W19 的 2 个是 P8a 存活的 `t11-cycles-back`（项目包装层又取迭代，M6）、`t11-unread-back`（顶部导航又取未读通知数，M7），都由改写的 S2 发现；W1 的 `t11-old-members-back`（旧的项目成员地址回到挂载路径）由关键词守卫和 S2 两层发现。
- W18（每个新的 vitest 都能失败）：`w18tests.py` 把 vitest 的 JSON 报告中 P8b 写或改的 9 个测试文件的 142 个测试（`it.each` 的每一行算一个），对照每个变异的 vitest 日志中失败的测试：P8b 新加或改写的 120 个都至少失败一次；没有失败过的 22 个都是 P8a 原有的测试（`store-context.test.ts` 的 6 个、`use-workspace-fetch.test.ts` 包装层判断的 7 个、`permissions.store.test.ts` 工作区一半的 9 个），不是 P8b 写的。
- 一个变异列在它守的性质出现的 Task（`mutcheck.py`：每个变异要改的文字在它的 Task 的快照上都恰好出现一次；其中 17 个要改的文字在更早的快照上就已出现：它们改的是 P8b 之前就有的代码，例如 `reconciled.ts`、`sortStates`、项目卡片、工作区包装层，列在第一个依赖这个性质的 Task）；`(之后的 Task 起)` 的标记由发现它的测试文件第一次出现的 Task 和 S2 的改写（Task 11）算出：只有 `t10-pw-children-non-member` 一个（`t10-pf-non-member` 的 S2 一层也从 Task 11 起，它在 Task 10 由 vitest 发现）。每个 Task 的变异另在那个 Task 自己的快照上跑过（`mutpertask.py`：把原型的源码换成那个快照、从它构建工作区内的包，先在没有变异的树上跑一遍变异要用的每个检查，都通过，再跑变异）：132 个中 131 个在它自己的 Task 的树上就被发现，所用的层与最终原型上相同（`t4-pf-old-workspace` 在 Task 4 的树上另由 `tsc` 发现）；唯一的例外是上面的 `t10-pw-children-non-member`，Task 10 的树上还是旧的 S2，它只从 Task 11 起被发现。
- **例外**（只由评审或之后的 Task 才能发现的性质）：`t10-pw-children-non-member` 从 Task 11 起由 S2 发现（第 3 节第 11 条）。没有只由之后的 Phase 才能发现的。

### A.3 清扫

- **19（每个新的 store 或 service 方法有失败测试）**：五个 store 的每个取数、修改各有一个拒绝的测试（`it.each` 按修改列出）；标识检查、项目的读、工作区的状态各有一个。13 个变异让取数或修改在被拒绝时给出空值而不失败（每个 store 的修改、取数各一个，加已归档的列表、标识检查、工作区的状态），都由 vitest 发现。原型的第一稿中标识检查和工作区的状态没有拒绝的测试，清扫 19 找出之后补上（`t5-ps-identifier-swallows`、`t8-st-workspace-fetch-swallows`）；`useProjectFetch` 的读被拒绝有 3 个（`t10-pf-any-refusal-not-found`、`t10-pf-code`、`t10-pf-no-retry`）。
- **36（另一条路径到达同一结果）**：16 个变异：项目自己的读绕过工作区的门（`t3-ps-left-workspace`）、包装层不看列表按地址取项目（`t3-wf-projects-unlisted`）、权限经别的工作区、Plane 的公开项目兜底、同一个 slug 重建的工作区（`t4-pf-other-workspace`、`t4-pf-unlisted-guest`、`t4-pf-old-workspace`）、子 store 绕过项目 store 的门（`t5-pp-left`、`t6-pm-left`、`t8-st-left`、`t9-lb-left`、`t9-lb-left-list`）、对 store 没有的成员、状态、标签照样发出修改（`t6-pm-unlisted`、`t8-st-unheld`、`t9-lb-unheld`）、旧地址或 M6、M7 的取数回到挂载路径（`t11-old-members-back`、`t11-cycles-back`、`t11-unread-back`）。13 个由 vitest 发现，3 个由 S2 发现（其中一个也由关键词守卫发现）。
- **45（没有不能失败的断言）**：每个新的测试至少由一个变异发现（W18，A.2）；拒绝的 `it.each` 的每一行由它自己的变异发现；"取数不等修改"的测试回答的列表与加载的不同，回答确实被写入；会话和拒绝的测试都从加载好的 store 开始（P8a 的教训一）；每个修改的回答带一个请求没有的字段（教训二）；每个 store 的测试里有两个以上的工作区或项目（教训三）。
- **46（每个等待有期限）**：store 的测试只用 `until(cond, label)`、`settle(promise, label)`（`fake-time.ts`，有期限、超时时说出在等什么）和假计时器上的 `inTurn`；没有不设期限的 `waitFor`；hook 的测试同步运行，不等待。S2 的等待是 `expect.poll`（Playwright 的期限）和 `waitForResponse` 的 `timeout`。
- **50（说明与代码一致）**：每个 Task 结束时，新写或改动的注释、JSDoc、关键词规则的 `why` 与它描述的代码或测试逐句核对；文档的两处（总体设计 7.7、前端改动清单）与最终的代码核对（Task 11）。

### A.4 W1：旧地址（源码和构建产物）

`w1bundle.py` 按 7.10 的地址组数 `web/apps/web` 的源码（`app/`、`core/`，测试除外）和构建产物（`build/client/assets`，最终原型的 `make build`）中的命中：

| 地址组 | `7390e012` 源码 | 最终 源码 | 最终 构建产物 |
|---|---|---|---|
| 项目（列出、创建） | 2 | 0 | 0 |
| 项目的 `details/` | 1 | 0 | 0 |
| 一个项目（查看、修改、删除） | 3 | 0 | 0 |
| 归档、恢复 | 2 | 0 | 0 |
| 标识检查 | 1 | 0 | 0 |
| 项目成员、离开 | 6 | 0 | 0 |
| `project-members/me` | 1 | 0 | 0 |
| 项目角色 | 1 | 0 | 0 |
| 加入（Plane 的 `projects/invitations`） | 1 | 0 | 0 |
| 项目的状态 | 6 | 0 | 0 |
| 工作区的状态 | 1 | 0 | 0 |
| 收集箱的分诊状态（M7） | 1 | 0 | 0 |
| 项目的标签 | 4 | 0 | 0 |
| 工作区的标签（M7） | 1 | 0 | 0 |
| 项目的 `user-properties`（M4） | 2 | 2 | 2 |
| 项目的收藏（M7） | 3 | 0 | 0 |
| `/api/users/` 之下的任何地址 | 2 | 0 | 0 |

项目的 `user-properties` 留下的 2 处是旧 `ProjectService` 留给 M4 的两个方法（关键词规则的 `until: "M4"` 例外，第 3 节第 19 条）；工作项页面读它们（A.5 的清单中只在工作项页上，答 404）。关键词规则对每个地址组有命中样例（A.11）。

### A.5 W2：每一页挂载时的请求（真实运行）

探针 `w2-inventory.spec.ts` 复制进副本的 `e2e/stories/` 运行之后删除；它打开每一页，记下 2 秒内没有新请求之前的每个 `/api/` 请求（路径中的 slug、项目的 id 换成占位）。工作区 `acme` 有一个公开项目 `web`（管理员、成员、访客都是它的成员，访客以访客的角色）、一个私密项目 `ops`（只有管理员）和一个已归档的公开项目 `old`；五个账户：工作区的管理员、成员、访客，工作区的成员而不是 `web` 的成员的人（"不是项目成员"），和只有另一个工作区的人（"不是工作区成员"）。每个登录的页面都有的五个请求（`auth/refresh`、`instance`、`me`、`me/profile`、`workspaces`，都是 200）不再列出。

**`7390e012`（之前）**，管理员、成员、访客、不是项目成员的人都一样：

| 页面 | 200 | 404（旧接口） |
|---|---|---|
| `/`、`/{slug}` | 工作区的显示设置、工作区的成员 | 项目角色 `/api/users/me/workspaces/{slug}/project-roles/`、项目 `/api/workspaces/{slug}/projects/`、工作区的状态 `/api/workspaces/{slug}/states/` |
| `/{slug}/projects`、`/{slug}/projects/archives` | 同上 | 同上，加 `/projects/details/` |
| `/{slug}/settings`、`/{slug}/settings/members` | 同上；general 页加时区（访客没有：页面不给他看），成员页对管理员加邀请 | 同第一行 |
| 项目设置的六页、`/{slug}/projects/{web}/issues`、`/{slug}/settings/projects/{ops}` | 工作区的显示设置、工作区的成员 | 同第一行，加项目 `/projects/{id}/`、`project-members/me/`、`members/`、`user-properties/`、`issue-labels/`、`states/` |

不是工作区成员的人：`/` 落到他自己的工作区，与上面的第一行相同；`/{slug}`（不是他的）和那里的项目设置只多一个 404：`/api/workspaces/{slug}/states/`（只按 slug 启用）。项目一侧的页面都打不开。

**最终（之后）**：

| 账户 | 页面 | 请求（都是 `/api/v0`，200，除非写明） |
|---|---|---|
| 管理员 | `/`、`/{slug}`、`/{slug}/projects`、`/{slug}/projects/archives` | 工作区的显示设置、成员、项目、状态 |
| | `/{slug}/settings`、`/{slug}/settings/members` | 同上；general 页加时区，成员页加邀请（只给管理员，P8a） |
| | `/{slug}/settings/projects/{web}`（general，加时区）、它的成员、迭代、状态、标签、自动化五页；`/{slug}/settings/projects/{ops}` | 工作区的四个，加项目的读和成员的四个：`projects/{id}`、`me/projects/{id}/preferences`、`projects/{id}/labels`、`projects/{id}/members`、`projects/{id}/states` |
| | `/{slug}/projects/{web}/issues` | 同上一行，另有 404 `/api/workspaces/{slug}/projects/{web}/user-properties/`（M4 的工作项筛选，第 5 节） |
| 成员 | 工作区的页面、`{web}` 的项目设置和工作项页 | 与管理员相同（成员页没有邀请） |
| | `/{slug}/settings/projects/{ops}` | 工作区的四个，加 404 `projects/{ops}`（`project.not_found`：私密项目对不是成员的人，8.3）；显示"找不到项目"（第 3 节第 18 条） |
| 访客 | 与成员相同，除了工作区设置的两页没有时区 | 工作区设置的两页和 `{web}` 的设置子页显示 Plane 的"无权查看"（页面自己的判断，P9、P10），请求与成员相同 |
| 不是项目成员 | 工作区的页面 | 与成员相同 |
| | `{web}` 的项目设置六页和工作项页 | 工作区的四个，加项目的读 `projects/{web}`（200，`member_role: null`）；没有任何子资源的请求；显示加入的界面 |
| | `/{slug}/settings/projects/{ops}` | 同成员 |
| 不是工作区成员 | `/`（落到他自己的工作区） | 那个工作区的四个 |
| | `/{slug}`、`/{slug}/settings/projects/{web}` | 只有每页都有的五个；显示"找不到工作区" |

之后的清单中没有 `/api/v0` 之外的请求，除了 M4 的工作项筛选那一个；没有收藏、通知、"最近"、迭代、模块、视图、分诊状态的请求；不是成员的人没有子资源的请求。M3 的页面（工作区的页面、项目的设置）上没有失败的请求，只有 nerve 对看不到的私密项目设计好的 404。

### A.6 W3、W4：分代和键

- M2 交接第 3 节的 `git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web`：27 处 → 22 处。M3 的 2 处（`project/form.tsx` 的 `ProjectService`、`project-member.service.ts` 的 `ProjectMemberService`）消失，M3 的 10 处至此全部消失；另 3 处是封面上传的 `FileService`（`image-picker-popover.tsx`、`create-project-modal.tsx`、`cover-image.helper.ts`，M5 的，随 Task 1 删除）。剩下的 22 处都是 M4–M7 的（其中 `ProjectService` 的 3 处是工作项搜索的调用方，第 3 节第 19 条）。
- 项目一侧的每个 store 和 service 由 `RootStore` 用这一代的 `ApiClient` 建：`ProjectRootStore(root, api)` 建 `ProjectStore`、`ProjectPreferencesStore`；`MemberRootStore` 建 `ProjectMemberStore`；`StateStore(root, api)`、`LabelStore(root, api)`；五个 service 由各自的 store 建。权限 store 不建 service（它只读 store）。
- 键：项目一侧的每个会话的取数经 `useSessionSWR`（2.12 的表）。P8b 改到的文件中仍从 `swr` 导入值的只有三个，取的都是 M4–M7 的东西：工作项的浏览页 `browse/[workItem]/page.tsx`（工作项，M4；Task 10 只改了包装层的 prop）、`use-workspace-issue-properties.ts`（迭代、模块，M6；Task 7 删去其中的工作区标签）、通知页的 `workspace-notifications/root.tsx`（收集箱项读项目，M7，第 5 节）。
- 静态检查：根目录 `.oxlintrc.json` 的 `no-restricted-imports` 的范围加上 `web/apps/web/core/store/member/project/**`、`core/store/project/**`、`core/store/state.store.ts`、`core/store/label.store.ts`、`core/components/projects/page.tsx`、`core/components/project-states/**`、`core/components/labels/**`、`core/components/project/project-settings-member-defaults.tsx`、`core/layouts/auth-layout/workspace-wrapper.tsx`、`project-wrapper.tsx`、`use-project-fetch.ts`（P8a 的已知限制在这里解除：两个包装层都在范围内）；`typescript/no-non-null-assertion` 的范围加上五个 service、`core/store/project/**`、`core/store/member/project/**`、状态和标签的 store 和它们的测试、`use-project-fetch.ts` 和它的测试、`permissions.store.test.ts`、`use-project-preferences.ts`、`use-tab-preferences.ts`、`projects/page.tsx`。W4、W12 的 7 个变异都让 `check:lint` 失败（A.2）。

### A.7 W12、W13

- **W12**（`w12.cjs`，TypeScript 的语法树，`7390e012` 对最终原型）：P8b 改到的 182 个源文件（测试除外）中，`as` 92 → 78、`any` 49 → 25、`!` 2 → 2（两处都在 M7 的 `project-inbox.store.ts`，原有，P8b 只删了它改写 `intake_count` 的两段）；没有一个文件的 `as`、`any`、`!` 变多，没有新的 `as`。连同测试是 192 个文件，测试中的 `as` 13 → 13（都是原有的）。没有手写的、重述契约的类型；knip 没有未使用的导出。
- **W13**（`w13.py`，`7390e012` 对最终原型，每一项一个 grep）：项目角色的取数 27 → 0、调用者的项目成员关系 24 → 0、权限 store 的项目 map 14 → 0、`intake_count` 22 → 0、项目的收藏 28 → 0、封面的上传和随机封面 43 → 0、收集箱的分诊状态 54 → 0、工作区的标签 18 → 0、Plane 的项目类型 197 → 0、Plane 的状态类型和 `order` 156 → 0、Plane 的标签类型和树 122 → 0、Plane 的项目成员类型和 service 44 → 0、旧的归档 service 4 → 0、旧的取数键 11 → 0、"无权访问"的界面 9 → 0；模块级的 `ProjectService` 4 → 3（工作项搜索，第 3 节第 19 条）。删除的路径都不存在（`image-picker-popover.tsx`、56 张预设封面、`project-archive.service.ts`、`project-member.service.ts`、`project-state.service.ts`、`issue_label.service.ts`、`dropdowns/intake-state/`、`packages/types/src/intake/`、`@nerve/propel` 的两个分诊图标）。
- **删除的文案键**（评审核对：`check:sync` 只核对两种语言的键相同，两种语言都留下的键它看不见；每个 Task 的"完成时"写明这些键在 `en`、`zh-CN` 中都已不在）。两种语言相同，从快照逐个量出（`i18nkeys.py`）：

| Task | 文件 | 键 |
|---|---|---|
| 1 | `common.json` | `change_cover`（1 条） |
| 10 | `empty-state.json` | `project_empty_state.no_access.restricted_description`（1 条） |

  其余 Task 没有删除或移动文案键（分诊状态、工作区的标签和收藏用的文案仍有别的读者）。

### A.8 oxlint（7.9，R3）

- 有手改的源文件（测试除外）146 个，0 条警告；P8b 写或改的测试文件也都是 0 条。修订前 `@nerve/utils` 的 `project.ts`（Task 2 手改）有两条 `no-array-reverse`，已改为 `toReversed()`（2.2）。
- 只经机械步骤到达的文件 38 个，其中 5 个共 14 条，留给 P11 第 4 个任务（P8b review 第 6 节照录）：

| 文件 | 条数 | 规则 |
|---|---|---|
| `core/components/core/sidebar/progress-stats/state_group.tsx` | 1 | `react/no-array-index-key` 1 |
| `core/components/dropdowns/project/base.tsx` | 5 | `no-shadow` 3、`unicorn/consistent-function-scoping` 1、`jsx-a11y/no-static-element-interactions` 1 |
| `core/components/power-k/ui/pages/context-based/work-item/commands.ts` | 3 | `promise/always-return` 3 |
| `core/components/project-states/group-item.tsx` | 2 | `jsx-a11y/no-static-element-interactions` 1、`jsx-a11y/click-events-have-key-events` 1 |
| `core/hooks/work-item-filters/use-work-item-filters-config.tsx` | 3 | `no-shadow` 3（P8a 的清单中已有） |

  其余 33 个只经机械步骤到达的文件没有警告。
- 上限：web 应用 435 → 421（Task 1）→ 414（Task 2）→ 411（Task 3）→ 409（Task 4）→ 408（Task 5）→ 406（Task 6）→ 370（Task 7）→ 367（Task 8）；`utils` 12 → 10（Task 2）→ 9（Task 8）→ 7（Task 9）；`types` 0、`constants` 1、`propel` 16、`ui` 19 不变。
- `promise/always-return`：P8b 有手改的文件中都已清零；只经机械步骤到达的 3 条在上表。
- 抑制：没有新的抑制；删去四处（`dropdowns/intake-state/base.tsx` 的一处随文件删除，`project-member.store.ts` 一处、`state.store.ts` 两处随重写删除）。

### A.9 死成员和死 prop（7.9）

M1 收尾的 `deadsym.mjs`、`domains.mjs`（`$M3TMP/p8btools/dead/`，`dead.sh`、`deadcmp.py`）：

- `--rows M3`：`7390e012` 上 163 行 → 最终 126 行；去掉 49 行，新加 12 行。
- 新加的 12 行都是测试里 `it.each` 的表的列（四个 store 测试的 `change`、`refusal`、`send`，在标题和行中读）。
- 去掉的 49 行包括 P8a 交给 P8b 的全部死行（`add-project-members-modal.tsx` 的 `value`、`query`、`content`、`onSuccess`，`project-member.store.ts` 的 4 行，`project_filter.store.ts` 的 4 个成员（`displayFilters`、`filters` 和两个按 slug 的 getter；第 5 行 `projectFilter` 在 `project/index.ts`），`useProjectColumns.tsx` 的 `member`），以及 P8b 重写的 store 中原有的死成员（`fetchedMap`、`getWorkspaceLabels`、`intakeStateMap`、`processProjectAfterCreation` 等）和没有调用方传的 prop（`memberIds`、`stateIds`、`handleOnClick` 等）。
- P8b 改到的文件中仍有的 M3 行和它们的去处：只经机械步骤到达的六个文件的 prop（`state_group.tsx` 的 `completed`、`state`、`total`，`label-item-block.tsx` 的 `draggable`，`group-list.tsx` 的三个 `className`，`state-item-title.tsx` 的 `shouldShowDescription`，`common-attributes.tsx` 的 `handleFormOnChange`，`project-create-buttons.tsx` 的 `isMobile`：P11 第 4 个任务）；`workspace.service.ts` 的搜索参数和 `types/src/project/projects.ts` 的工作项搜索类型（M4 的搜索，P8a 已记）。

### A.10 规模

plan 共 12,491 行：Task 1 之前（约束、一次性脚本、文件结构）183 行；各 Task 一节（从它的标题到下一个 Task 的标题；代码块的行数含围栏和提交命令的块）：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 生成的 `Project` 没有的 | 1,308 | 1,028 | 79（57 个删除，其中 56 张图片） |
| 2 生成的 `Project` | 1,299 | 1,082 | 57（38 个经机械步骤，其中 16 个另有手改） |
| 3 项目的 service 和 store | 1,559 | 1,374 | 35 |
| 4 加入、离开；权限 store | 958 | 833 | 21 |
| 5 标签栏；标识检查；旧 `ProjectService` | 1,116 | 982 | 24 |
| 6 项目成员 | 1,237 | 1,115 | 20 |
| 7 M7 的部分；标签页面清零 | 949 | 819 | 30 |
| 8 状态 | 1,504 | 1,286 | 50（28 个经机械步骤，其中 9 个另有手改） |
| 9 标签 | 1,289 | 1,104 | 34（22 个经机械步骤，其中 11 个另有手改） |
| 10 项目包装层 | 604 | 501 | 13 |
| 11 关键词、S2、死行、文档 | 485 | 415 | 6 |

- **Phase 的大小**（brief 的检查，写 plan 之前做）：11 个 Task，在约 16 个之内；Task 3 的 1,559 行比约 1,500 行多 4%，Task 8 是 1,504 行（第 3 节第 2 条）。没有停下来。
- **文件的行数**（约 400 行）：P8b 写或重写的文件中最长的是 `project.store.test.ts` 422 行（第 3 节第 9 条）、`label.store.test.ts` 362 行、`project-member.store.test.ts` 322 行、`state.store.test.ts` 323 行、`project.store.ts` 312 行、S2 260 行。只为使用方改到的 Plane 文件，原来超过 400 行的都没有变长，除了第 3 节第 8 条的三个（808 → 812、465 → 466、414 → 415）；`tools/keywords.json` 2,189 → 2,200（规则的样例）。其余手改的 Plane 文件都变短了（例如 `project/form.tsx` 462 → 411、`projects-list-item.tsx` 476 → 460、`project-inbox.store.ts` 524 → 511）。

### A.11 关键词规则

`tools/keywords.json` 仍是 64 条规则；例外 3 → 2（Task 4：`project-invitations` 的 `until: "M3"`）→ 3（Task 11：`plane-workspace-urls` 的 `until: "M4"`，`project.service.ts` 中项目的 `user-properties/` 两处，`count: 2`）。每条规则的命中、不命中样例和 `files` 样例都由 `node tools/keywords.mjs` 核对。

| 规则 | 模式的一段（Task 11 加的） | 命中样例 | 不命中样例（模式不能宽到它们） |
|---|---|---|---|
| `plane-workspace-urls` | `/project-roles/` | `/api/users/me/workspaces/${workspaceSlug}/project-roles/` | |
| | `` /api/workspaces/${…}/projects/` ``、`projects/details/` | 列出、创建、`details/` | `…/projects/${projectId}/search-issues/`（M4） |
| | `` projects/${…}/` ``、`archive/`、`members/`、`project-members/me/` | 一个项目、归档、成员、离开、改角色、`project-members/me/` | `…/projects/${projectId}/invitations/`（M1-P3 删除的项目邀请）、`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/` |
| | `projects/${…}/states/`、`issue-labels/` | 设默认、删除标签 | `…/issues/${issueId}/labels/`（工作项的标签，M4）、`/api/v0/projects/{project_id}/labels` |
| | `projects/${…}/user-properties/`（M4 的例外） | 项目的 `user-properties/` | `…/cycles/${cycleId}/user-properties/`（M6）、`…/archived-cycles/`（M6） |
| | 工作区的 `project-identifiers`、`states/`、`labels/` | 标识检查、工作区的状态、工作区的标签 | `/api/workspaces/${workspaceSlug}/views/`（M7） |
| `plane-user-urls` | 整个 `/api/users/` 和 `/api/instances/` | 原来的 7 个，加原来的三个不命中样例（加入项目的 `projects/invitations/`、用户设置、工作区列表）和 `project-roles` | `/api/v0/me/profile`、`/api/v0/instance`、`/api/v0/projects/{project_id}/join` |

`plane-workspace-urls` 共 31 个命中样例、15 个不命中样例；`plane-user-urls` 11 个、3 个。最终原型上：64 条规则，3 个例外，没有命中。

### A.12 逐 Task 复现

`replay.py` 在 `7390e012` 的 `git archive` 副本（`$M3TMP/p8breplay`，`pnpm install --frozen-lockfile` 之后）上照 plan 逐个 Task 执行：先是 `Run（机械步骤）:` 的命令（`$P8BTMP` 中的一次性脚本从 plan 的"一次性脚本"一节原样取出），核对散列表；再写入块；核对 Go 的模块文件不变、`pnpm-lock.yaml` 与那个 Task 的快照相同；再执行这个 Task 的每个 `Run:`。副本不是 git 仓库，所以 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（它列文件不用 git，规则和样例相同），`make e2e` 只有 S3 因为读不到提交而失败（F4），由脚本单独认出。

| Task | 机械步骤 | 写入的文件 | 检查 |
|---|---|---|---|
| 1 | | 79 | `make lint-web`、`make knip`、`make test-web` 通过；`make e2e` 69 个通过（S3 除外） |
| 2 | `rename_types.py` 三次（30、1、7 个文件），38 个文件的散列和行数与表相同 | 35 | `pnpm install --frozen-lockfile` 通过；其余同 Task 1 |
| 3–7 | | 35、21、24、20、30 | 同 Task 1 |
| 8 | `rename_types.py` 三次（21、11、2 个文件），28 个文件的散列和行数与表相同 | 31 | 同 Task 1 |
| 9 | `rename_types.py`（22 个文件）和 oxfmt 两次，22 个文件的散列和行数与表相同 | 23 | 同 Task 1 |
| 10 | | 13 | 同 Task 1 |
| 11 | | 6 | 同 Task 1，`make e2e` 73 个通过（S3 除外） |

第一次复现在清扫 19 补上两个拒绝的测试之前（只有 Task 5、8 的两个测试文件的块不同），11 个 Task 都通过；补上之后从一个新的副本从头重新复现，11 个 Task 都通过，上表是这一次。最后的副本与原型逐文件相同：`treediff.mjs p8breplay p8bproto`：3,113 个文件，0 处差异（三份文档：M3 设计、本 spec、plan 在比较之前放进两边）。副本上 web 应用的 vitest 是 39 个测试文件、411 个测试。
