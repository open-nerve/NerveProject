# M3/P8a 工作区一侧的数据层：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P8a `web-workspace-data` |
| 日期 | 2026-10-07 |
| 状态 | 进行中：spec、plan 和原型已写成；修订（2026-10-07）写入了控制者对第 3 节的裁定（A1–A6）和预检的发现（H1、M1、M2、L1–L9） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（S1、W2 的页面清单）、3.1、3.2（`logo_url`、`avatar_url`）、3.8、3.10、3.14、3.18、3.20（P8a 的两行）、7.1–7.5、7.8–7.11、8.3、9.5、12（P8a 与约束 2、4）、13.1 节；[总体设计](../../v0-design.md) 7.7 |
| 前置交接 | [M2 收尾交接](../handoffs/M2-closeout.md) 第 2、3、7、11、13 节；[M1 收尾交接](../handoffs/M1-closeout.md)（死成员、oxlint）；[M1-P2](../handoffs/M1-P2-trim-content.md)、[M1-P3](../handoffs/M1-P3-trim-platform.md)、[M1-P4](../handoffs/M1-P4-router-native.md) 的保留名单；P1–P7b 各 spec 第 5 节和 review 第 6 节中 P8 的行；[Codex 设计评审](../reviews/M3-design-codex-adversarial-review.md) 4.3；拆分的裁定 R1–R4（控制者 2026-10-07） |
| 计划 | [P8a plan](../plans/P8a-web-workspace-data.md) |

本 spec 只写 M3 设计交给 P8a 决定的东西：名字、签名、文件的位置、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P8a 依赖 P7b（`2474d32e` 的 `main`）和拆分的设计提交 `7cf3a286`；P8b 在 P8a 合并之后开始（12 节 P8a 的"拆分"）。

P8a 的每一处取数都按安全测试看待（brief）：一个角色不能发的请求在页面上是 403，换账户之后才落地的写以错误的身份写入。所以附录 A 的变异逐个核对：换代之后旧的一代不写、它的反应已释放；页面只取调用者有权读的；每个键带 `loginId`；每个修改一个接一个；落点只认此刻的列表。只有评审才能发现的会话、权限或取数的性质算缺口。

## 1. 目标

按 M3 设计 12 节 P8a：

- 会话分代的基础（总体设计 7.7、M3 设计 7.1）：`RootStore.dispose()` 释放 `project_filter` 的反应，由 `store-context.tsx` 在换代时调用；`sessionGuard()` 在 `core/lib/in-session.ts`；会话的取数只有一种写法：`sessionKey`（键带 `loginId`）和 `useSessionSWR`（fetcher 交回 store 的 Promise，没有权限时取数为 `null`）；绕过它的会话取数由 oxlint 静态地发现（附录 A.6）。
- problem 码的文案表 `PROBLEM_MESSAGES` 和它与契约一致的 vitest 移到 `core/lib/error-messages.ts`，文案移到 `errors` 命名空间（约束 4）。
- 系统内接受删除（决策点 2，7.8）。
- 工作区一侧的类型（`Workspace`、`WorkspaceMember`、`MemberUser`、`WorkspaceInvitation*`、`InvitationPreview`、`WorkspacePreferences`）、service、store 迁到 `/api/v0`，按会话分代，修改经 `oneAtATime()`、写入 nerve 的回答；权限 store 的工作区一半取自 `Workspace.role`。
- 落点（3.14、7.4）：`landingPath` 算出地址，`useLanding` 做出全部判断，`AuthenticationWrapper` 照它渲染；设置 store 删除；`RESTRICTED_URLS` 删除，保留名单前后端一份，"应用"一段有 vitest（3.10）。
- 两个包装层和顶部导航中 M6、M7 的挂载时取数、首页的"最近"小部件删除（3.1）；工作区一侧的挂载时取数改调新 store，按权限启用（7.1）。
- 使用方改到能编译、行为不变；不同的逐条写在第 3 节和第 6 节（W17）。
- 工作区一侧的关键词规则（7.10）；3.20 中 P8a 的两行。

## 2. 交付物

**文件总览**：plan 的"文件结构"一节逐个列出。合计（`7cf3a286` → 最终原型，本 spec 和 plan 除外）：新文件 36 个、删除 15 个、修改 151 个（其中 29 个只经机械步骤改到）；web 之外只有 `server/internal/modules/workspace/domain/reserved_slugs.txt`、`tools/keywords.json`、`.oxlintrc.json`（Task 8 的 `overrides`）、`pnpm-lock.yaml`、`e2e/stories/identity/a3-sign-in.spec.ts`（只改注释：没有工作区的账户经落点去 `/create-workspace`）和两份文档。新文件中 14 个是测试，5 个是测试的共用部分（`fake-*.ts`）。

**依赖**：不加 npm 包或 Go 模块。`@nerve/types`、`@nerve/utils` 加工作区内的依赖 `@nerve/api-client`（`workspace:*`）：`packages/types` 的 M4 搜索类型和 `packages/utils` 的 `orderWorkspacesList` 要引用生成的 `Workspace`、`MemberUser`（7.2：类型只来自生成的客户端）。锁文件随之两处，`pnpm install --frozen-lockfile` 通过。

### 2.1 会话分代的基础（Task 1；7.1，总体设计 7.7）

- `core/lib/session-key.ts`：`sessionKey(session, name, ...args): SessionKey | null`，`SessionKey = readonly [name, loginId, ...args]`；只有 `signed-in` 且每个参数已知时给出键。键的形状是 `[名称, loginId, 参数…]`，不是 7.1 举例的 `["workspaces", loginId]`：同一个名称在不同的参数下（不同的工作区）是不同的取数。
- `core/lib/use-session-swr.ts`：`useSessionSWR(fetch, fetcher, config)`，`fetch = [name, ...args] | null`；fetcher 收到键中的参数，交回 store 的 Promise。这是会话的取数的唯一写法（W4：一个有 vitest 的键构造，不是只有 grep 看得见的约定）：工作区一侧的取数都经它，M2 已有的两处手写的会话键（令牌列表、当前用户）改用它，项目一侧的取数在 P8b 随各自的 store 改用它。唯一的例外是公开的邀请查看（第 3 节第 9 条）。绕过它由静态检查发现：Task 8 在根目录 `.oxlintrc.json` 加 `no-restricted-imports`，范围内的文件不能从 `swr` 导入值（2.8，附录 A.6）。
- `core/lib/in-session.ts`：`sessionGuard(): () => boolean`。`theme-switcher.tsx` 改用它；页面的删除、离开、创建、接受之后的核对在 P9、P10（7.1）。
- `RootStore.dispose()` 调用 `projectRoot.projectFilter.dispose()`（`reaction` 的返回值）；`store-context.tsx` 的 `follow` 在建好新的一代之后对退役的一代调用它。要释放的只有这一个反应（7.1）；`cycle_filter`、`module_filter`、`issue_calendar_view` 和 `issue/root.store.ts` 的 `autorun` 由 M4、M6 照同一写法接上（13.2）。
- 测试：`session-key.test.ts`（4 个）、`use-session-swr.test.ts`（2 个）、`in-session.test.ts`（2 个）、`store-context.test.ts` 加 1 个（换代之后改路由，旧一代的筛选 store 不再运行，9.5）。

### 2.2 problem 码的文案表（Task 2；约束 4）

- `core/lib/error-messages.ts`：`PROBLEM_MESSAGES`、`FIELD_ERROR_MESSAGES`、`errorMessageKey`、`fieldErrorKeys`、`needsErrorBanner`，从 `helpers/authentication.helper.ts` 原样移来，文案的键从 `auth.errors.*` 改为 `errors.*`。`helpers/authentication.helper.ts` 只留 `EPageTypes`、`EAuthModes`。
- `errors` 命名空间：两份 `errors.json` 是原来 `auth.json` 的 `errors` 对象，原样移来；P6 的两条 `sole_admin` 文案不变（P6 spec 第 3 节第 17 条）。
- 读文案的 11 个组件改从新位置导入，是 plan 的机械步骤（`t2_imports.py`）。
- 测试：`error-messages.test.ts` 即原来的测试随表移来，16 个标题不变；表与契约的 `x-problem-codes`、`FieldError.code` 在两个方向上核对（W14 的两个变异：少一个码、多一个码）。

### 2.3 系统内接受的删除（Task 3；决策点 2，7.4、7.8）

- 删除：`/invitations` 页和它的路由；它的入口（工作区菜单、个人设置侧栏的工作区一栏、命令面板，以及"不是成员"界面的链接和说明）；新手引导的"加入工作区"一步（`join-invites.tsx`、`ECreateOrJoinWorkspaceViews`、`invitations` 这个 prop）；旧 `WorkspaceService` 的 `joinWorkspaces`、`userWorkspaceInvitations`；取数键 `USER_WORKSPACES_LIST`；只为它们存在的文案（两种语言各 14 条）。
- 服务端保留名单的"应用"一段删去 `invitations`（P1 spec 第 3 节第 3 条："删掉这一页的改动同时把它移出名单"），`make test` 照旧通过。
- `/invitations` 之后是一个工作区的地址（3.10），由工作区的页面回答"找不到工作区"，与未知地址相同（W13；`navigation.test.ts` 的叶子路由核对，附录 A 的页面核对）。
- 关键词规则 `in-system-invitations`。

### 2.4 工作区的类型、service、store（Task 4；7.2、7.3）

- `WorkspacesService(api)`：`list`、`create`、`update`、`delete`（Task 4），`leave`、`checkSlug`（Task 5），`accept`、`decline`（Task 9）。每个方法一次调用生成的客户端。
- `WorkspaceRootStore(rootStore, api)`（重写，`core/store/workspace/index.ts`）：`workspaces: Workspace[] | undefined`（nerve 的顺序；没有取过时 `undefined`，与"取到了、一个都没有"区分）、`currentWorkspace`、`getWorkspaceBySlug`、`fetchWorkspaces`、`checkWorkspaceSlug`、`createWorkspace`、`updateWorkspace`、`deleteWorkspace`、`leaveWorkspace`、`acceptInvitation`、`declineInvitation`，子 store `preferences`（Task 10）、`webhook`（不变）。修改经同一个 `changes = oneAtATime()`；`getWorkspaceRedirectionUrl` 删除（7.3），删除和离开之后跳到 `/`，由落点决定。
- `Workspace` 取代 `IWorkspace`：`owner` 没有了；`logo_url` 读作 `null`；`organization_size` 是 `OrganizationSize | null`，`ORGANIZATION_SIZE` 随之改为 `OrganizationSize[]`；调用者的角色是 `Workspace.role`。11 个只换类型名的使用方是机械步骤（`rename_type.py`）。
- 工作区图标的上传控件在这里删除，连同它的两条文案（`edit_logo`、`upload_logo`，两种语言）（第 3 节第 2 条）。
- 测试的共用部分：`fake-queue.ts`（`inTurn`，和"取数不等修改"的 `fetchedWhileChangeIsOut`）、`fake-root.ts`（`fakeRoot`）、`fake-workspaces.ts`（`workspaceOf`、`loadWorkspaces`）、`FakeNerve.replacedSessionClient()`；之后每个 store 的测试都用它们，排队、取数不等修改、会话已换、拒绝的写法各只有一处（brief："不重复的逻辑，测试也一样"）。
- 测试：`index.test.ts` 的列表 4 个、修改 5 个（拒绝的 `it.each` 在 Task 5、9 加到六行）。
- 关键词规则 `plane-workspace-urls`，模式只写到 P8a 换掉的地址为止（第 3 节第 21 条）。

### 2.5 落点、设置 store、保留名单、离开（Task 5；3.10、3.14、7.4、7.8，R4）

- `landingPath(workspaces, lastWorkspaceId)`（`core/lib/landing.ts`）：上次的工作区仍在列表中 → 它；否则最早创建的（按时刻比较，同一时刻取列表靠前的：nerve 按名称、id 排序）；没有 → `/create-workspace`。
- `useLanding(pageType, validNextPath): Landing`（`core/lib/use-landing.ts`，第 3 节第 6 条）做出落点的全部判断：从 store 读资料和工作区列表；资料已到、已完成引导、没有有效的 `next_path`、停在登录页或引导页时需要落点，取列表（`useSessionSWR(lands ? ["WORKSPACES"] : null, …)`）；给出 `none`、`loading`、`unavailable`（带重取的 `retry`）或 `go`（`landingPath(workspaces, profile.last_workspace_id)`）。`AuthenticationWrapper` 只照它渲染：`unavailable` 显示 `SessionUnavailable`，不跳；`go` 跳过去。M2 为"完成引导的用户直接去 `/create-workspace`"写的两个分支改为这条规则（M2 交接第 2 节）。`core/lib/fake-session-swr.ts`（测试里代替 `useSessionSWR`：记下交给它的取数，交回测试给的回答）在这里写成，Task 6、8 的 hook 测试照用。
- 删除：`IUserSettings`、`settings.store.ts`、`useUserSettings`、`currentUserSettings`、四处 `await fetchCurrentUserSettings()`（Task 3、4、5 各自随所在的代码删除）、`RESTRICTED_URLS` 和它的测试。
- 两处创建工作区的表单改问 `checkWorkspaceSlug`；`create-workspace-form.tsx` 没有调用方传的 `secondaryButton`、`primaryButtonText` 删除（7.9）。
- 离开工作区：`WorkspaceRootStore.leaveWorkspace`（经 `changes`，成功之后移出列表）；`UserPermissionStore.leaveWorkspace`、`UserService.leaveWorkspace` 删除（M2 交接第 11 节；它的新家与设计第 12 节的写法不同，第 3 节第 23 条）。
- 设置页窄屏导航的开合放进 `ThemeStore`：`settingsSidebarCollapsed`（初值 `true`）、`toggleSettingsSidebar(collapsed?)`（R4，第 3 节第 18 条）。
- 保留名单只有服务端的一份；`navigation.test.ts` 读它的"应用"一段，等于路由的第一段静态段与 `public/` 的顶层目录的并集，两个方向都核对（W15 的三个变异：少一个路由的名字、多一个没有路由的名字、`public/` 多一个目录）。
- 测试：`landing.test.ts` 5 个（W8：三种情形、上次的工作区不在列表中、空列表、同一时刻、按时刻不按文字）；`use-landing.test.ts` 3 个测试、1 组 5 行的 `it.each`（W8：登录页和引导页的落点、上次的工作区；不落的五种账户和页面，不取列表；列表未到、取不到和重取）；`index.test.ts` 加 slug 检查、离开；`navigation.test.ts` 的保留名单 1 个。
- 关键词规则 `restricted-urls`、`user-settings`。

### 2.6 权限 store 的工作区一半；包装层取列表（Task 6；7.1、7.2、7.3、8.3）

- `UserPermissionStore.getWorkspaceRoleByWorkspaceSlug(slug): WorkspaceRole | undefined` 取自 `workspaceRoot.getWorkspaceBySlug(slug)?.role`；`allowPermissions` 中工作区一级的判断和"工作区管理员"的判断读它。`workspace-members/me` 的取数、`loader`、`workspaceUserInfo`、`workspaceInfoBySlug`、`fetchUserWorkspaceInfo`、`IWorkspaceMemberMe` 删除；`getProjectRole` 的规则不变。
- 模块级的 `workspaceService` 改为 store 的字段：它剩下的唯一调用是 `project-roles`，属于 P8b（第 3 节第 4 条）。
- `useWorkspaceFetch`（`core/layouts/auth-layout/use-workspace-fetch.ts`，第 3 节第 13 条）：工作区包装层挂载时的工作区一侧的取数。Task 6 只取列表；`WorkspaceAuthWrapper` 在列表取不到时显示 `SessionUnavailable`，路由的工作区不在列表中时显示"找不到工作区"，不分不存在和不是成员（8.3）。Plane 的"Not Authorized"分支随 `workspaceInfoBySlug` 删除。
- 读调用者角色的使用方：邀请弹窗的 `fields.tsx` 和 `invitations-list-item.tsx` 改用 `getWorkspaceRoleByWorkspaceSlug`，原来的 `as` 随之消失；12 个页面（工作区设置的布局、10 个设置页、个人主页的页头）原来以 `workspaceUserInfo` 守着无权限的界面，它是一个对象、恒为真，这一层删去，行为不变；草稿 store 往 `workspaceUserInfo` 写草稿数的一段（没有读者）删除。
- 测试：`permissions.store.test.ts`（1 个测试、2 组 `it.each`），与 9.2 同一组身份（管理员、成员、访客、不是成员的人）的角色和允许，列表未取时什么都不允许（W9）；`use-workspace-fetch.test.ts` 1 个（Task 5 的 `core/lib/fake-session-swr.ts` 代替 `useSessionSWR`）。

### 2.7 `MemberUser` 和工作区成员（Task 7；5.2、7.2、7.3）

- `MemberUser` 取代 `IUserLite`（`is_bot`、`joining_date` 随它删除；M2 交接第 7、11 节）；18 个文件的改名是机械步骤，其中 6 个另有手改。`avatar_url`、`email` 读作 `null`（W20）；成员加入的时刻是 `WorkspaceMember.created_at`。
- `WorkspaceMembersService(api)`：`list`、`update`、`remove`。
- `WorkspaceMemberStore(memberRoot: Pick<IMemberRootStore, "memberMap">, rootStore, api)`：每个工作区的成员关系按账户 id 存（已结束的也在），每次取数整份换掉；成员的公开资料写进 `memberMap`。改角色、移出经 `changes`，写入回答；store 没有这个人的成员关系时不问 nerve、直接失败。
- 成员按角色和加入时间排序用 `toSorted`，不改 store 交出的数组：web 应用的 TypeScript `lib` 改为 ES2023（`web/packages/typescript-config/react-router.json`，只有 `web/apps/web/tsconfig.json` 继承它），导航原有的两处 `no-array-sort` 抑制随之改为 `toSorted`、删除（第 3 节第 16 条）。
- 测试：`workspace-member.store.test.ts`，8 个测试、2 组 `it.each`（`fake-members.ts` 的 `membershipOf`、`memberStore`；"取数不等修改"用 Task 4 的 `fetchedWhileChangeIsOut`）。

### 2.8 邀请；成员页按角色取数；工作区的页面取成员（Task 8；3.8、7.1，决策点 4）

- `WorkspaceInvitationsService(api)`：`list`、`create`、`update`、`delete`。
- `WorkspaceMemberStore` 的邀请：每次取数整份换掉；创建、改角色、删除与成员的修改经同一个 `changes`，写入回答，被拒绝时不改。
- `useMembersSettingsFetch(slug)`：成员给每个看得到这一页的人，邀请只给工作区管理员（Codex 4.3 第 1 条的前一半；`members-list.tsx:46-55` 原来不分角色先取邀请）。是不是管理员由 hook 自己从工作区列表读（`getWorkspaceBySlug(slug)?.role === EUserWorkspaceRoles.ADMIN`），不由调用方传入（第 3 节第 13 条）；`members-list.tsx` 调用它，它的 `isAdmin` prop 只决定显示。9.5 的成员页取数条件的 vitest：`use-members-settings-fetch.test.ts` 1 个测试（管理员）、1 组 3 行的 `it.each`（成员、访客、列表中没有这个工作区的调用者）。
- `useWorkspaceFetch(slug)`：列表说明这是调用者的工作区时才取它的成员，hook 自己算出这个条件（`workspaceSlug !== undefined && getWorkspaceBySlug(workspaceSlug) !== null`）；键与成员页相同（`["WORKSPACE_MEMBERS", slug]`），一次取数供两处。
- 根目录 `.oxlintrc.json` 的 `overrides`，两条规则各有自己的文件列表（附录 A.6；放在这个 Task，因为这里出现最后一个在范围内的文件）：会话的取数不能从 `swr` 导入值（`no-restricted-imports`，`allowTypeImports`）；P8a 的 M3 路径不能用非空断言（`typescript/no-non-null-assertion` 为 `error`，W12、W20）。
- 测试：`workspace-invitations.test.ts`，6 个测试、1 组 `it.each`；`use-workspace-fetch.test.ts` 改为 2 个（列表中有的 `acme` 和没有的 `elsewhere`）。

### 2.9 邀请链接（Task 9；3.8、7.1、7.4，决策点 1、2）

- `previewInvitation(client, id, token)`：公开的查看，调用方传 `publicClient`；`useInvitationPreview(id, token)` 的键是 `["INVITATION_PREVIEW", id, token]`，不带 `loginId`（第 3 节第 9 条）。邀请页、登录和注册页的标题读它。
- 接受、忽略在 `WorkspaceRootStore`（第 3 节第 10 条）：接受的工作区加入已取的列表，已是成员时换掉它的那一项；忽略不改列表。
- 邀请页（`workspace-invitations/page.tsx`）只改到用新的查看、接受、忽略：Plane 按 `slug`、`email` 查看和"已接受"的一支删除；7.4 的行为（未登录、已登录、邮箱不符、已忽略、无效）在 P9。
- 测试：`invitation-preview.service.test.ts` 1 个（请求不带 `Authorization`；无效链接失败）；`index.test.ts` 加接受、忽略，排队的测试加上这两个修改。
- 这个 Task 之后，M2 交接第 3 节 `git grep` 中 M3 的 `WorkspaceService` 8 处都已消失（附录 A）。

### 2.10 工作区的显示设置（Task 10；3.18、7.1、7.3）

- `WorkspacePreferencesService(api)`：`get`、`update`。
- `WorkspacePreferencesStore(api)`：`WorkspaceRootStore.preferences`；`getPreferences`、`fetchPreferences`（会话已换给出 `undefined`）、`updatePreferences`（经 `changes`，写入回答；Plane 先改再回滚，第 3 节第 7 条）。
- `useWorkspaceFetch`：列表中有这个工作区时（hook 算出的 `isMember`）再取 `["WORKSPACE_PREFERENCES", slug]`。
- `use-navigation-preferences.ts` 读写新 store；视图模型 `TProjectNavigationPreferences` 留在 `packages/types`，模式用生成的 `NavigationControlPreference`（第 3 节第 5 条）。
- 删除：`fetchWorkspaceFilters`、`patchWorkspaceFilters`、`IWorkspaceUserPropertiesResponse`、键 `WORKSPACE_PROJECT_NAVIGATION_PREFERENCES`、`issue_filter.service.ts` 的注释块。
- 测试：`preferences.store.test.ts` 7 个（"取数不等修改"用 `fetchedWhileChangeIsOut`）；`use-workspace-fetch.test.ts` 的成员一个改为成员和显示设置。

### 2.11 M6、M7 的挂载时取数（Task 11；3.1，R1）

- 删除：工作区包装层取收藏、顶部导航取未读通知数、项目包装层取迭代、两次模块、视图、分诊状态的 SWR，它们的 hook 和键；首页的"最近"小部件和只为它存在的组件、类型、文案、旧 service 方法。
- 这些删除留下的没有调用方的代码一并删除（第 3 节第 15 条）：`FavoriteStore.fetchFavorite`、`FavoriteService.getFavorites`、`ModulesStore.fetchModulesSlim`、`ProjectViewStore.fetchViews`、`ViewService.getViews`、`IWorkspaceNotificationStore.getUnreadNotificationsCount`（接口上的；类里的方法由它自己调用）、`ListItem` 只有"最近"小部件传的四个 prop、旧 `WorkspaceService.updateWorkspaceView`。这些文件因此有了手改，按 7.9 清零：7 条警告（`module.store.ts` 5、`project-view.store.ts` 1、`list-item.tsx` 1），改法不改行为（`then` 回调改为 `await`，`.catch` 改为 `try`，只把错误原样抛出的 `try` 去掉，`x && x` 即 `x`）。
- 项目包装层中 M3 自己的取数不动（P8b）。

### 2.12 文档（Task 12；3.20）

总体设计 7.7 的五处（会话的取数都经 `useSessionSWR`：键的写法、公开操作这个唯一的例外和 `.oxlintrc.json` 的静态检查；"页面按权限决定取数"；释放；`sessionGuard()`；problem 码的文案表在 `core/lib/error-messages.ts`）；前端改动清单 3.1 的 M3 一行、错误格式一行，3.2 加五行（系统内接受、slug 检查、用户设置和落点、挂载时的取数、工作区图标的上传）。

### 2.13 挂载时的取数和它们的条件（W5）

| 取数 | 键 | 何时取 | 在哪里 |
|---|---|---|---|
| 调用者的工作区 | `["WORKSPACES", loginId]` | 已登录 | `useWorkspaceFetch`（工作区的每一页）；`useLanding`（只在需要落点时，`AuthenticationWrapper` 调用） |
| 工作区的成员 | `["WORKSPACE_MEMBERS", loginId, slug]` | 列表中有这个工作区 | `useWorkspaceFetch`；成员页 `useMembersSettingsFetch`（成员页在包装层之内）；个人主页 `use-profile-member.ts` |
| 调用者在工作区的显示设置 | `["WORKSPACE_PREFERENCES", loginId, slug]` | 列表中有这个工作区 | `useWorkspaceFetch` |
| 工作区的邀请 | `["WORKSPACE_INVITATIONS", loginId, slug]` | 列表给出的调用者在这个工作区的角色是管理员 | `useMembersSettingsFetch` |
| 邀请链接的查看 | `["INVITATION_PREVIEW", id, token]` | 链接带两者；公开，不看会话 | `useInvitationPreview`：唯一不经 `useSessionSWR` 的取数，不在 `.oxlintrc.json` 限制 `swr` 的范围内 |
| 当前用户、令牌列表（M2） | `["CURRENT_USER", loginId]`、`["API_TOKENS", loginId]` | 已登录 | 改用 `useSessionSWR`，条件不变 |

"何时取"的条件都由取数的 hook 自己从 store 算出（`useLanding`、`useWorkspaceFetch`、`useMembersSettingsFetch`；第 3 节第 13 条），调用方只传地址，写错的调用方绕不过条件。除了邀请链接的查看，表中的每个取数都经 `useSessionSWR`；绕过它（直接 `useSWR`）的会话取数由根目录 `.oxlintrc.json` 的 `no-restricted-imports` 发现（Task 8，附录 A.6 的范围和限制）。

项目一侧的挂载时取数（项目角色、项目列表、工作区的状态、项目详情和它的子资源）不在本表，它们随各自的 store 在 P8b 改接、按 `member_role` 启用（附录 A 的 W2 清单列出它们此刻的请求）。

### 2.14 修改的排队（W6）

| store | 经 `oneAtATime()` 的修改（每个 store 一个队列） | 不排队的读 |
|---|---|---|
| `WorkspaceRootStore` | `createWorkspace`、`updateWorkspace`、`deleteWorkspace`、`leaveWorkspace`、`acceptInvitation`、`declineInvitation` | `fetchWorkspaces`、`checkWorkspaceSlug` |
| `WorkspaceMemberStore` | `updateMember`、`removeMemberFromWorkspace`、`inviteMembersToWorkspace`、`updateMemberInvitation`、`deleteMemberInvitation` | `fetchWorkspaceMembers`、`fetchWorkspaceMemberInvitations` |
| `WorkspacePreferencesStore` | `updatePreferences` | `fetchPreferences` |

`UserPermissionStore` 的 `joinProject`、`leaveProject` 是项目一侧的，随项目成员的 store 在 P8b 排队。每个队列有"上一个有了回答才发下一个，被拒绝也一样"的 vitest，表中的每个修改都在它的队列的这个测试里（`WorkspaceRootStore` 的六个在 Task 9 齐）；"取数不等修改"的 vitest 三个 store 各有一个，共用 `fake-queue.ts` 的 `fetchedWhileChangeIsOut`（`fetches the list / the members / the settings while a change is out`，变异 `t4-store-fetch-queued`、`t7-members-fetch-queued`、`t10-prefs-fetch-queued`）。W6 的 14 个变异（绕过队列、取数进队列、在回答之前改 store）都由 vitest 发现。

## 3. 与设计的差异、补充和需要裁定的

1. **Task 的切分**：设计 12 节 P8a 的草稿是 11 个任务，plan 是 12 个（12 节 P7 一段的先例："在上限之内重新切分任务"）。控制者已裁定照这 12 个 Task（A1）。对应：设计 1 → Task 1；2 → 2；8 中系统内接受的删除提前为 Task 3（之后的 Task 不必再改它的使用方）；3 → 4（另加图标上传的删除，第 2 条）；4 → 5（另加离开工作区和 R4）；3 中权限 store 的一半和包装层取列表单独为 Task 6（权限 store 读的是 Task 4 的列表）；5、6 合为 Task 7（`MemberUser` 是成员 store 的类型，分开时成员 store 要先写一个过渡的类型）；7 → 8；8 中邀请链接的查看和接受为 Task 9（系统内接受已在 Task 3）；9 → 10；10 → 11；11 的关键词规则分到各 Task（规则随它禁止的地址的删除加入），3.20 为 Task 12。12 个 Task 都在约 1,500 行之内，最长的 Task 4 是 1,567 行（其中代码块 1,361 行，附录 A.10）。
2. **工作区图标的上传控件在 P8a 删除**（裁定 A2；设计 7.8 原写在 P9、P10，已在 P8a 的修订中改：同一个提交改了设计的 7.5、7.8，第 12 节 P8a 的任务 3 和目标，P9 的任务 6）：生成的 `WorkspaceUpdate` 没有 `logo_url`（3.2 的规则 2），Task 4 把 general 页换成 `Workspace` 和新 store 时，上传弹窗设置图标的那一步无法再编译。删掉它是 P9 第 6 个任务的删除提前，P9 少这一项；项目封面的上传仍在 P10。没有调用方的 `deleteWorkspaceAsset` 和只为它存在的 `ACCEPTED_AVATAR_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE` 一并删除。
3. **服务端的保留名单改在 P8a**（`reserved_slugs.txt`，Task 3、5）：删去 `invitations`（P1 spec 第 3 节第 3 条）和两句说明。名单是 Go 嵌入的文本，`make lint-go`、`make test` 在这两个 Task 运行。
4. **`permissions.store.ts` 的 `WorkspaceService`**：M2 交接第 3 节的 `git grep` 数的模块级实例在 Task 6 消失，改为 store 的字段；它剩下的唯一调用 `project-roles` 是项目一侧的，P8b 换掉时整个字段删除。这个字段仍是旧的 axios service，不按会话分代，P8a 到 P8b 之间它答 404（附录 A 的 W2 清单）。
5. **`TProjectNavigationPreferences` 留在 `packages/types`**：7.2 把它列在 `ProjectPreferences` 一行。它实际是侧边栏的视图模型（导航方式、显示几个、是否限制），由 `WorkspacePreferences` 算出，不重述契约；P8a 保留它，导航方式的类型改用生成的 `NavigationControlPreference`，Plane 的 `TProjectNavigationMode` 删除。项目一侧的 `IProjectMemberNavigationPreferences` 在 P8b。
6. **落点的全部判断在 `useLanding`，由 vitest 守着**（裁定 A4 的第二种做法）：原来 `AuthenticationWrapper` 自己判断谁要落点、取列表、处理取不到、把上次的工作区交给 `landingPath`，这些接线只有 P9 的 W2 页面版本能发现（web 的 vitest 在 node 环境里运行，没有 `@testing-library/react`、`jsdom`，渲染不了包装层；P8a 也不能加这个故事：落点到工作区页面时，工作区页面在 P8b 之前仍有项目一侧的 404）。现在 hook 做出全部判断，包装层只照它渲染；`use-landing.test.ts` 用 Task 6、8 已有的写法（`vi.mock` 换掉 `use-session-swr` 和 store 的 hook）测它，判断的每一项各有一个变异（M2 的 `/create-workspace`、不看上次的工作区、引导页不落、取不到时不报、有 `next_path` 时也落、资料未到就落、仍在引导也落、任何页面都落），都由它发现。落点的规则本身由 `landing.test.ts` 的 5 个测试守着。**两个变异由之后的 Phase 发现**（附录 A.2）：`t11-cycles-back`、`t11-unread-back`（M6、M7 的取数回到挂载路径），它们是一类的样例：迭代、模块、视图、分诊状态、收藏、未读通知数、"最近"的取数方法多数仍在 store 上（例如 `fetchModules`），任何一个加回挂载路径，P8a 的检查都看不见；P8b 改写的 S2 断言每个账户的每张挂载清单上都没有 M6、M7 的地址（第 5 节）。
7. **修改写入回答，不乐观**：Plane 改成员角色和显示设置时先改 store、失败再回滚；P8a 的 store 等 nerve 的回答，写入回答（v0 的写法，`ApiTokenStore`）。页面上的差别是改动在回答之后才显示（W17）。
8. **store 的形状**：成员关系、邀请每次取数整份换掉（Plane 是合并），已结束的成员关系留在 map 里由 `is_active` 区分（nerve 列出它们）；成员 store 只依赖 `Pick<IMemberRootStore, "memberMap">`；加入的时刻取 `created_at`（`joining_date` 不在契约里）。
9. **邀请的查看不带 `loginId`**：它是公开的操作，回答只由链接决定，对任何会话和没有会话都一样；键带 `loginId` 会让登录前后各取一次而没有好处。它是 P8a 写的唯一不带 `loginId` 的键，经 `useSWR` 而不经 `useSessionSWR`（附录 A.6）。控制者裁定（A6）：原则是会话的取数带 `loginId`、经 `useSessionSWR`，经 `publicClient` 的公开操作只取决于它的输入，键就是它的输入；这是唯一的例外。设计 7.1 的 SWR 一条已在修订中照此改写（连同键的形状 `[名称, loginId, ...参数]`）。绕过的会话取数由静态检查发现：根目录 `.oxlintrc.json` 的 `no-restricted-imports` 限制范围内的文件从 `swr` 导入值（只能导入类型），`use-invitation-preview.ts` 不在范围内（Task 8，附录 A.6）。
10. **接受、忽略放在 `WorkspaceRootStore`**：接受的回答是工作区，改的是调用者的工作区列表；邀请的 store（成员 store）是管理员的。邀请页"已接受"的一支随 Plane 的查看删除，P9 照 7.4 重写这一页。
11. **`fake-root.ts` 的一处 `as`**：`fakeRoot(siblings)` 用 `Proxy` 给测试一个只有被测 store 读的兄弟 store 的 `RootStore`，读了别的就失败；`Partial<RootStore>` 到 `RootStore` 要一处断言。这是 P8a 唯一新的 `as`，只在测试的共用部分（W12）。
12. **留给 P9、P10 的 Plane 缺陷**（P8a 只改类型，不改这些页面的行为）：项目成员设置的角色下拉框（`core/components/project/settings/member-columns.tsx`）以 `Object.entries(…)` 的键作值，发出的角色是字符串（P10）；已忽略的邀请在工作区成员页的列表中显示为待接受（P9）；新手引导的邀请一步失败时提示的文案未定义（P9）。
13. **`useWorkspaceFetch` 是新加的 hook，取数的条件由 hook 自己算出**：工作区包装层的三个工作区一侧的取数放进一个可以脱离 React 测试的 hook（`fake-session-swr.ts`），键只在一处（W4）。设计没有点名它。它和成员页的 `useMembersSettingsFetch` 只收地址的 slug，条件从工作区 store 读（预检 H1）：`useWorkspaceFetch` 的"列表中有这个工作区"是 `getWorkspaceBySlug(slug) !== null`，`useMembersSettingsFetch` 的"是管理员"是列表给出的 `role === EUserWorkspaceRoles.ADMIN`。条件若由调用方传入（原来的 `isMember`、`isAdmin`），把错的值传进来的调用方（例如把成员的权限当作管理员传）过得了 `tsc`、vitest 和 `make e2e`；现在条件和它的 vitest 在同一处，每一种错法各有一个变异（附录 A.2 的 W5）。
14. **页面上看得到的不同**（W17；P9–P11 照第 2 节的故事改页面）：工作区的页面现在能显示（`7cf3a286` 上没有取列表，都显示"找不到工作区"）；不是成员时显示"找不到工作区"而不是 Plane 的"Not Authorized"（8.3）；侧边栏的项目导航在显示设置取到之后按它（默认显示 10 个）；general 页没有图标上传；首页没有"最近"、侧边栏没有收藏、通知按钮没有数字（3.1）；登录之后按 3.14 落点；删除或离开工作区之后经 `/` 落点；成员页只为管理员取邀请；改角色、显示设置在回答之后才显示（第 7 条）；`/invitations` 是"找不到工作区"。另外六处（预检 L7）：邀请页 `/workspace-invitations` 和注册页带邀请时的标题读 nerve 的公开查看，邀请的链接现在能打开（之前什么都不显示）；邀请页"已经是成员"的一支没有了（第 10 条）；接受之后总是跳转；两处创建工作区的表单和新手引导的邀请一步现在真的发到 nerve；设置页侧边栏的开合在 `ThemeStore`，换账户之后仍保留（R4，第 18 条）；邀请列表的"复制链接"总是显示。
15. **P8a 的删除留下的、没有调用方的代码一并删除**：设计 3.2 对 `favoriteProjectIds` 的写法（"没有读者，成为死代码，在 P8b 删除，M7 加回"）和 brief 的"删除的代码删干净"。M1 收尾的 `deadsym.mjs` 在最终原型上量过：P8a 留下的新行只有测试里的字面量和测试的假实现（附录 A.9）。删除碰到 M6、M7 的四个 store（收藏、模块、项目视图、通知）、两个 service（收藏、视图）和一个共用组件（`ListItem`），它们因此有了手改，按 7.9 清零（2.11）；这些改动不改行为，M6、M7 加回取数时照它们的新接口写。
16. **没有新的 oxlint 抑制：web 应用的 TypeScript `lib` 提到 ES2023**（控制者裁定 A3）：web 应用的 `lib`（ES2022）落后于仓库自己的基础配置（`typescript-config/base.json`、`react-library.json`、`node-library.json` 已是 `es2023`），`packages/utils/src/module.ts` 已在应用的包里调用 `toSorted`，运行时的支持早已假定。Task 7 把 `web/packages/typescript-config/react-router.json` 的 `lib` 改为 ES2023（只有 `web/apps/web/tsconfig.json` 继承它，`target` 不变），`core/store/member/utils.ts` 用 `members.toSorted(…)`，没有抑制、没有注释。原有的两处 `unicorn/no-array-sort` 抑制（`core/components/navigation/use-navigation-items.ts`、`tab-navigation-root.tsx`）排的都是 `filter` 新给的数组，同一步改为 `toSorted`、删去抑制，两个文件之后没有警告（R3：手改约束文件）。改回 `members.sort(` 的变异（`t7-sort-shared`，就地排 store 交出的数组）由 oxlint 的上限发现。
17. **`/invitations` 落到"找不到工作区"**：删掉的页的地址与未知地址相同，由工作区的页面回答（附录 A.5 的页面核对）。没有加重定向：它是一个合法的工作区地址（3.10 的名单不再保留它）。
18. **R4：`sidebarCollapsed` 放进 `ThemeStore`**：它已经有侧边栏、工作项详情侧栏等布局的开合（`sidebarCollapsed`、`issueDetailSidebarCollapsed`），设置页的导航开合是同一类状态；名字改为 `settingsSidebarCollapsed`、`toggleSettingsSidebar`，与主侧边栏的区分。它不存进 `localStorage`（设置 store 原来也不存）。没有新建 store。
19. **删除、离开工作区之后跳到 `/`**：`getWorkspaceRedirectionUrl` 删除（7.3），落点由 `AuthenticationWrapper` 按此刻的列表算（3.14）。Task 4 和 Task 5 之间，`/` 照 M2 去 `/create-workspace`（过渡，Task 5 结束）。
20. **组件在 `await` 之后的核对**（7.1、Codex 4.3 第 2 条）：P8a 提供 `sessionGuard()`；删除工作区、离开、创建、接受之后的跳转改用它在 P9（7.1 写明），P8a 不改这些组件的跳转逻辑。
21. **关键词规则只写到 P8a 换掉的地址**：`plane-workspace-urls` 的模式在每个 Task 加上那个 Task 删掉的地址；不命中样例覆盖项目一侧的每一种地址（项目、项目成员、项目角色、项目的 `user-properties`、项目的离开和邀请），守住模式不会宽到 P8b 还在用的地址（附录 A.11）；P8b 把项目一侧补进模式，连同 `/user-properties/` 的 `until: "M4"` 例外（7.10）。`/search-issues/` 的不命中样例在 Task 4 就在。
22. **成员页取数的 vitest 在 P8a，页面版本在 P9**（7.1："两处的取数条件各有一个 vitest（成员页的在 P8a）"）：`use-members-settings-fetch.test.ts`，它核对 hook 自己从工作区列表读出的角色（管理员取邀请；成员、访客、列表中没有这个工作区的调用者不取，第 13 条），调用方传不进条件；W4 的页面版本（成员打开成员页，`watchPage` 没有失败的请求）在 P9。
23. **`leaveWorkspace` 的新家**（预检 L9）：设计第 12 节 P8a 的任务 6 写"`user.service.ts` 的 `leaveWorkspace` 改用生成的客户端"；P8a 把离开放进 `WorkspacesService.leave` 和 `WorkspaceRootStore.leaveWorkspace`（经工作区的队列，成功之后移出列表，2.5），`UserService.leaveWorkspace` 和 `UserPermissionStore.leaveWorkspace` 删除：离开改的是调用者的工作区列表，与接受、忽略同一个 store（第 10 条）。

**交接到 P8a 的事项的落点**（brief"Carried into P8"；P1–P7b 各 spec 第 5 节、review 第 6 节）：

| 来源 | 事项 | 落点 |
|---|---|---|
| M2 收尾第 3 节 | 10 处模块级实例；`project_filter` 的释放和 vitest；键带 `loginId` | 工作区一侧的 8 处：Task 3、5、6、8、9；释放：Task 1；键：Task 1 起每个取数。`ProjectService`、`ProjectMemberService` 2 处：P8b |
| M2 收尾第 2 节 | 落点函数和 vitest；`currentUserSettings` 和四处 `await`；取数失败没有未处理的拒绝 | Task 5；Task 3、4、5；Task 1（`useSessionSWR` 交回 store 的 Promise，W16 的 8 个变异） |
| M2 收尾第 7 节 | `IUserLite` → `MemberUser`；可空的 `logo_url`、`avatar_url` 读作 `null` | Task 7；Task 4、7（W20）。`cover_image_url`：P8b |
| M2 收尾第 11 节 | `leaveWorkspace`、`joinProject`、`leaveProject`；`is_bot`、`owner`；`plane-user-urls` 收紧；新手引导的加入一步 | `leaveWorkspace`：Task 5；`is_bot`：Task 7（M4 的活动类型中的两处见第 5 节）；`owner`：Task 4；加入一步：Task 3；其余：P8b |
| M2 收尾第 13 节 | "找不到工作区"界面的退出按钮、`isLoading` | P9（9.7 的浏览器核对） |
| M1-P3 项目成员 | `project-invitations` 例外删除 | P8b |
| M1-P2、P3、P4 保留名单 | 一份名单，"应用"一段与路由、`public/` 两个方向一致 | Task 3、5 |
| `PROBLEM_MESSAGES`（P1、P2、P3、P5b、P6、P7a、P7b） | 移出登录的 helper 和 `auth`；vitest 两个方向；P6 的两条文案不变 | Task 2 |
| P4b | `ProjectPreferences.navigation`、侧边栏的 `sort_order`、`joinProject` | P8b |
| P7a | 状态的顺序和组中的位置取代 `order` | P8b |
| P7b | 标签的类型、只按项目取、建时不发 `sort_order` 和 `null` 的父标签 | P8b |
| M1 收尾（7.9） | 改到的文件 0 条 oxlint 警告；上限调低；`always-return` | Task 4–11，附录 A.8；`always-return`：P8a 有手改的文件中的 7 条都已清零，只经机械步骤到达的 3 条和全仓其余的在 P11 第 5 个任务 |
| 3.20 | 总体设计 7.7；前端改动清单 | Task 12 |
| Codex 设计评审 4.3 | 成员页的邀请只为管理员取；项目包装层按 `member_role`；迟到的成功回答核对会话 | Task 8；P8b；`sessionGuard()` 在 Task 1，组件的核对在 P9、P10 |
| Codex 设计评审 M-3 | 旧 `ProjectService` 的三个方法留给后续的 M | P8b |

## 4. 验收标准（完成线，M3 设计 12 节 P8a）

- [ ] S1 和此前的全部故事通过（S2 在 P8b 改写）：worktree 中 `make e2e` 70 个全部通过。
- [ ] M2 交接第 3 节的 `git grep` 中 `WorkspaceService` 的 8 处消失（附录 A.6）。
- [ ] `node tools/keywords.mjs` 通过：64 条规则，没有命中（`make lint-web`）。
- [ ] 9.5 中 P8a 的 vitest 通过：`sessionGuard`、换代释放、`sessionKey`、`useSessionSWR`、文案表与契约、落点（`landingPath` 和 `useLanding` 的全部判断）、保留名单、成员页的取数条件、工作区包装层的取数条件（两者都由 hook 自己从工作区列表算出）、权限 store 的工作区一半、四个 store 的取数、修改、会话、排队、取数不等修改；每个都有一个发现它的变异（W18，附录 A.2）。
- [ ] 根目录 `.oxlintrc.json` 的 `overrides` 在 `make lint-web` 中生效：范围内的会话取数从 `swr` 导入值、P8a 的 M3 路径用非空断言，`check:lint` 都失败（`t8-raw-swr-hook`、`t8-raw-swr-profile`、`t8-nonnull-email`，附录 A.6）。
- [ ] `tsc`、knip 通过。
- [ ] 改到的文件按 7.9 没有 oxlint 警告：有手改的 133 个源文件 0 条，没有新的抑制；只经机械步骤到达的 29 个文件 15 条列在附录 A.8，留给 P11 第 4 个任务；web 的上限 452 → 435，其余各包不变。
- [ ] 逐 Task 复现：从修订提交的树（它只改文档，plan 的改动都还没有做）照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同（附录 A.12）。

## 5. 不在 P8a 范围内

- **P8b**（按 store 和请求；附录 A.5 的 W2 清单此刻仍有的 404 都在这里）：
  - 权限 store 的项目一半：`/api/users/me/workspaces/{slug}/project-roles/`、`/projects/{id}/project-members/me/`；`joinProject`、`leaveProject`；`plane-user-urls` 收紧；`project-invitations` 的例外删除；`permissions.store.ts` 的 `workspaceService` 字段（第 3 节第 4 条）。`getProjectRole` 中"工作区管理员是项目管理员"一条此刻读 P8a 的工作区列表，它的变异 `pf-project-admin-any`（任何工作区角色都算项目管理员，预检 L3）在 P8a 的检查中存活：P8b 的权限测试必须发现它。
  - 项目 store：`/api/workspaces/{slug}/projects/`、`/projects/details/`、`/projects/{id}/`；`project/form.tsx` 的模块级 `ProjectService`；`is_favorite`、`favorite.store.ts` 的项目分支、`intake_count`；`cover_image_url`。
  - 状态 store：`/api/workspaces/{slug}/states/`（此刻只按 slug 启用，不是成员的工作区也发出）、`/projects/{id}/states/`。
  - 项目成员 store：`/projects/{id}/user-properties/`、`/projects/{id}/members/`；`ProjectMemberService` 的模块级实例。
  - 标签 store：`/projects/{id}/issue-labels/`。
  - 项目包装层按 `member_role` 启用子资源和它的 vitest；关键词规则补全；两个包装层（`workspace-wrapper.tsx`、`project-wrapper.tsx`）的取数都改经 `useSessionSWR` 之后，把它们加进 `.oxlintrc.json` 中 `no-restricted-imports` 的范围（附录 A.6）。
  - S2 的改写：断言每个账户的每张挂载清单上都没有 M6、M7 的地址，不只是没有发往旧接口的请求。P8a 存活的两个变异（`t11-cycles-back`、`t11-unread-back`）是这一类的样例，这一类是迭代、模块、视图、分诊状态、收藏、未读通知数、"最近"的取数回到挂载路径（取它们的方法多数仍在 store 上，例如 `fetchModules`）。
  - P8a 改到的项目一侧文件里 M3 的死行：`add-project-members-modal.tsx`（`value`、`query`、`content`、`onSuccess`）、`project-member.store.ts`（4 行）、`project_filter.store.ts`（5 行）、`useProjectColumns.tsx`（`member`）。
- **P9**：落点、新手引导、邀请页、注册页、工作区首页和侧边栏、工作区设置的页面行为（第 2 节的故事），W2 的页面版本（落点的页面一侧；落点的判断在 P8a 已由 `use-landing.test.ts` 守着）；创建工作区的表单照 nerve 对 slug 的回答决定能否提交的直接检查（`t5-slug-ignored` 此刻只因为留下一个没用的变量被 oxlint 的上限发现，附录 A.2）；删除、离开、创建、接受之后的 `sessionGuard()` 和它们的 vitest；切换、创建工作区、接受邀请之后写 `last_workspace_id`；第 3 节第 12 条中 P9 的两个 Plane 缺陷；M2 收尾第 13 节。
- **P10**：项目的页面；项目封面的上传和随机封面的删除；下拉框、复制到剪贴板、表情选择器；第 3 节第 12 条中项目成员的角色下拉框。
- **P11**：只经机械步骤到达的 29 个文件的 15 条警告（附录 A.8）；`always-return` 全仓清零；`--rows M3` 剩下的行（附录 A.9）。
- **M4**：活动类型中的 `is_bot`（`types/src/issues/activity/base.ts` 的 `TIssueActivityUserDetail`、`comments/card/display.tsx`）；`types/src/workspace.ts` 中工作区搜索的类型和旧 `WorkspaceService.searchWorkspace` 的参数；旧 `WorkspaceService` 的视图、草稿、搜索方法；`issue/root.store.ts` 的 `autorun` 的释放（13.2）。
- **M5**：工作区图标的上传（`logo_asset_id`，3.2）。
- **M6**：项目包装层取迭代、模块；`ModulesStore.fetchModulesSlim` 照新接口加回；`cycle_filter`、`module_filter` 的反应的释放。
- **M7**：收藏（`fetchFavorite`、`getFavorites`）、未读通知数（取数和接口上的方法）、首页的"最近"（连同 `ListItem` 的四个 prop）、视图（`fetchViews`、`getViews`）、分诊状态的挂载时取数，随各自的接口加回（13.2）。

## 6. 风险

- **P8a 与 P8b 之间工作区的页面仍有项目一侧的 404**（附录 A.5）：这是拆分的已知过渡（R1），没有页面的故事在这期间加入（P9 在 P8b 之后）。退路：无须；P8b 的 S2 改写之后核对清单为空。
- **M6、M7 的取数回到挂载路径，P8a 的检查看不见**（第 3 节第 6 条）：迭代、模块、视图、分诊状态、收藏、未读通知数、"最近"的取数方法多数仍在 store 上，任何一个加回包装层或顶部导航，`tsc`、关键词规则、vitest 和此刻的 `make e2e` 都通过（`t11-cycles-back`、`t11-unread-back` 是样例）。退路：P8a 与 P8b 之间没有故事加入，P8b 改写的 S2 断言每张挂载清单上都没有 M6、M7 的地址（第 5 节）；评审时照 Task 11 的"变异"表核对。
- **M6、M7 的 store 有了手改**（第 3 节第 15 条）：清零的 7 条警告改的是这些 store 的写法（`await` 代替 `then`、去掉无用的 `try`），没有页面能走到它们（W17）。退路：改法是逐行等价的变换，`tsc` 核对类型；M6、M7 改接新接口时重写这些方法。
- **机械步骤依赖 oxfmt 的版本**：Task 2、4、7 的散列表在 oxfmt 0.35.0 上量出；锁文件固定它。对不上时先核对 `pnpm exec oxfmt --version`。
- **不是 git 仓库的副本里 S3 失败**（P4b spec F4）：复现在副本中运行，只有 S3 因为读不到提交而失败，由复现脚本单独认出；worktree 中 70 个都要通过。

## 7. 已知的限制、交接和关闭条件

- **关闭**（在 P8a 的部分）：M2 收尾第 2 节中 P8a 的部分（落点、`currentUserSettings` 和四处 `await`、取数失败没有未处理的拒绝）；第 7 节（`IUserLite` → `MemberUser`）；第 3、11 节中工作区一侧的部分（其余在 P8b）；M1-P2、M1-P3、M1-P4 的保留名单（前后端一份，有测试）。
- **交接**：第 5 节各行照录进 P8b、P9–P11 的 spec 和 M4–M7 的交接（13.2）；P8a 的 review 第 6 节列出只经机械步骤到达的文件和它们按规则的警告数（R3），以及第 3 节中裁定的结果。
- **限制**：P8a 不改任何页面的新行为（第 2 节的故事在 P9–P11）；工作区一侧的 store 都已按会话分代，项目一侧的仍是 Plane 的写法，直到 P8b。

## 附录 A：原型验证记录（2026-10-07）

### A.1 方法与门禁

- 原型 `$M3TMP/p8proto` 是 `2474d32e` 的 `git archive` 副本，之上是设计提交 `7cf3a286`（只改文档）。每个 Task 结束时存一份快照（`$M3TMP/p8snap/T1`…`T12`；Task 7 另有机械步骤之后的 `T7m`），plan 的块由快照之间的差异写出（`mkblocks.py`：12 个 Task，0 个问题），机械步骤的文件不写成块。
- 最终原型上：`make gen-check`（副本中以 `make gen` 之后逐字节比较代替）0 处差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个包 `ok`；`make lint-web` 通过（关键词守卫 64 条规则、3 个例外、没有命中）；`make knip` 通过；`make test-web` 通过（web 应用 29 个测试文件、246 个测试）；`make e2e` 70 个中 69 个通过，S3 在副本中失败（F4）。
- 每个 Task 的快照上 `make lint-web`、`make knip`、`make test-web`、`make e2e` 都跑过；Task 3、5 另跑 `make lint-go`、`make test`。修订（控制者的裁定和预检的发现）逐个 Task 改原型：改那个 Task 的快照、在它上面跑这四个门禁，再把改动三方合并进之后的快照（`amend.py`、`propagate.py`；移到 Task 5 的 `fake-session-swr.ts`，和 Task 10 又改过的 `use-workspace-fetch.ts`、它的测试，手工合并），最后从快照重新写出 plan 的块。
- 逐 Task 复现（`replay.py`）：从修订提交的树（`7cf3a286` 加上三份文档：M3 设计、本 spec、plan）的副本起，照 plan 先执行机械步骤（核对散列表），再应用块，再执行每个 `Run:`；结果见 A.12。

### A.2 变异

`mutants_p8a.py`：125 个变异，每个只改一处或几处，`mut.py` 在最终原型上、在它写的每个检查上各跑一次；123 个被发现，2 个由之后的 Phase 发现（第 3 节第 6 条）。"静态"是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、i18n 的 `check:sync`，和 `make knip`。按缺陷类别（一个变异可以被几层发现）：

| 类别 | 变异 | 静态 | vitest | 端到端 | 存活 |
|---|---|---|---|---|---|
| W1 旧地址回到挂载路径 | 2 | | | | 2（P8b 的 S2） |
| W3 分代 | 5 | 1 | 5 | | |
| W4 键带 `loginId` | 8 | 2 | 7 | | |
| W5 按权限取数 | 8 | 3 | 8 | | |
| W6 一个接一个 | 14 | | 14 | | |
| W7 `sessionGuard()` | 3 | | 2 | 1 | |
| W8 落点 | 14 | | 14 | 1 | |
| W9 权限 store | 5 | | 5 | | |
| W12 `as`、`any`、`!` | 1 | 1 | | | |
| W13 删除的代码 | 5 | 5 | 1 | | |
| W14 文案表 | 5 | 2 | 3 | | |
| W15 保留名单 | 3 | | 3 | | |
| W16 取数失败 | 8 | 8 | 1 | | |
| W17 行为不变 | 1 | 1 | | | |
| W18 其余的新测试 | 2 | | 2 | | |
| W20 可空的字段 | 1 | 1 | | | |
| 19 每个新方法的失败 | 29 | 1 | 29 | | |
| 36 别的路径到达同一结果 | 10 | 8 | 2 | | |
| R3 手改的文件没有抑制 | 1 | 1 | | | |

修订加的 17 个变异：W8 的 7 个（`useLanding` 的每一项判断：`t5-auth-no-hint`、`t5-auth-onboarding-page`、`t5-auth-list-error`、`t5-auth-next-path`，和为 W18 加的 `t5-auth-no-profile`、`t5-auth-still-onboarding`、`t5-auth-any-page`；`t5-auth-m2-landing`、`t5-auth-drops` 改到 `use-landing.ts`，`t5-auth-m2-landing` 现在由 vitest 发现）；W5 的 3 个（取数条件由 hook 自己算出之后的三种错法：`t8-settings-any-role`、`t8-settings-member-up`、`t8-wrapper-any-slug`；预检把最后一个记作 `t6-wrapper-any-slug`，它改的 `isMember` 在 Task 8 才出现，所以列在 Task 8）；W4 的 2 个（`t8-raw-swr-hook`、`t8-raw-swr-profile`：绕过 `useSessionSWR`，`.oxlintrc.json` 的 `no-restricted-imports` 发现）；W6 的 3 个（`t9-decline-unqueued`，`t7-members-fetch-queued`、`t10-prefs-fetch-queued`）；W12 的 `t8-nonnull-email`（`typescript/no-non-null-assertion`）；R3 的 `t7-sort-shared`（就地排 store 交出的数组，oxlint 的上限发现）。`t2-zh-missing` 由 i18n 的 `check:sync` 发现，算静态（预检 L1，W14 一行的计数本来就是这样）。修订加的和改到的每一行（以及测试文件在修订中改过的 Task 4、5、7、9 的全部 store 行）另在各自 Task 的树上跑过，都在所写的层被发现。

- W18（每个新的 vitest 都能失败）：P8a 新加的每个测试（`it.each` 的每一行算一个）都至少被一个变异发现。`w18tests.py` 把 vitest 的 JSON 报告中这 16 个测试文件的 108 个测试，对照每个变异的 vitest 日志中失败的测试：P8a 新加的都至少失败一次；没有失败过的 21 个都是原有的测试（移来的 `error-messages.test.ts` 的 10 个、`navigation.test.ts` 原有的 5 个、`store-context.test.ts` 原有的 6 个），不是 P8a 写的。第一轮的 94 个变异之后有 14 个新测试没有变异发现，加了 14 个变异（11 个归 19、1 个归 W5、2 个归 W18），都由 vitest 发现；修订的变异跑完之后，`use-landing.test.ts` 的 `it.each` 中四行（资料未到、仍在引导、需要登录的页面、公开页面）没有变异发现，加了 3 个（W8），都由它发现。
- 一个变异列在它改的代码第一次出现的 Task；它在那个 Task 的树上就被发现，除了 plan 中标"（之后的 Task 起）"的检查。第一轮之后为 W18 加的 14 个中，Task 4、5、7 的 5 个改的文件在之后的 Task 还有变化，另在各自 Task 的快照上跑过 vitest，都被发现；其余 9 个改的文件和发现它的测试文件在它的 Task 与最终相同。
- W16 的变异（fetcher 丢掉 store 的 Promise）都由 `tsc` 发现：`useSessionSWR` 的 fetcher 类型要求 `Promise<T>`；`use-session-swr.test.ts` 另核对交给 SWR 的正是 store 的那个 Promise。
- **例外**（只由评审或之后的 Phase 才能发现的性质，brief 的缺陷类别）：
  - `t11-cycles-back`、`t11-unread-back`（第 3 节第 6 条）：M6、M7 的取数回到挂载路径，这一类由 P8b 改写的 S2 发现（第 5 节）。
  - `t5-slug-ignored`（W17，预检 L6）：创建表单不管 nerve 对 slug 的回答都创建。它在表中被发现，只是因为这个变异留下一个没用的变量，警告数超过上限；不留变量的同样的错法通过全部检查。表单在 vitest 中不渲染，这个性质在 P9 之前没有直接的检查（第 5 节 P9）。
  - `workspace-wrapper.tsx` 在 P8b 之前不在 `no-restricted-imports` 的范围内（A.6 的已知限制）。

### A.3 清扫

- **19（每个新的 store 或 service 方法有失败测试）**：四个 store 的每个取数、修改各有一个拒绝的测试（`it.each` 按修改列出），`checkWorkspaceSlug` 和邀请链接的查看各有一个；29 个变异把失败答成成功、吞掉失败、在回答之前改 store 或改错回答（其中 8 个是 service 不读 nerve 的 problem），都由 vitest 发现。
- **36（另一条路径到达同一结果）**：10 个变异在旧 service 或别的组件里加回一个旧方法或旧地址，8 个由关键词守卫或 `tsc` 发现，2 个由 vitest 发现。
- **45（没有不能失败的断言）**：每个新的测试至少由一个变异发现（W18，A.2）；拒绝的 `it.each` 的每一行由它自己的变异发现（各改一个 store 或 service 方法）。
- **46（每个等待有期限）**：store 的测试只用 `until(cond, label)`（`fake-time.ts`，有期限、超时时说出在等什么）、`settle(promise, label)` 和假计时器上的 `inTurn`；没有不设期限的 `waitFor`；hook 的测试同步运行，不等待。
- **50（说明与代码一致）**：每个 Task 结束时，新写或改动的注释、JSDoc、关键词规则的 `why` 与它描述的代码或测试逐句核对；文档的两处（总体设计 7.7、前端改动清单）与最终的代码核对（Task 12）。

### A.4 W1：旧地址（源码和构建产物）

`w1bundle.py` 按 7.10 的地址组数 `web/apps/web` 的源码（`app/`、`core/`，测试除外）和构建产物（`build/client/assets`）中的命中。最终的两列在原型和逐 Task 复现的副本上相同（构建产物是各自最后一次 `make e2e` 的构建）：

| 地址组 | `7cf3a286` 源码 | 最终 源码 | 最终 构建产物 |
|---|---|---|---|
| 工作区（列出、创建） | 2 | 0 | 0 |
| 一个工作区（查看、修改、删除） | 3 | 0 | 0 |
| 上次访问的工作区 | 1 | 0 | 0 |
| slug 检查 | 1 | 0 | 0 |
| 成员、离开 | 4 | 0 | 0 |
| `workspace-members/me` | 1 | 0 | 0 |
| 邀请（管理员、加入） | 6 | 0 | 0 |
| 系统内接受 | 2 | 0 | 0 |
| 用户设置 | 2 | 0 | 0 |
| 工作区的 `user-properties` | 4 | 0 | 0 |
| 收藏（M7） | 5 | 4 | 4 |
| 未读通知数（M7） | 1 | 1 | 1 |
| 最近访问（M7） | 1 | 0 | 0 |

收藏留下的 4 处是 `FavoriteService` 的加收藏、改、删除和分组（M7 的操作，P8a 删掉的只是挂载时的列出）；未读通知数留下的 1 处是通知 store 取通知列表时（通知页，M7）自己调用的方法。它们都不在 P8a 到达的页面的挂载路径上（A.5 的清单中没有它们）。关键词规则对每个 M3 的地址组有命中样例，`/search-issues/` 是不命中样例（A.11）。

### A.5 W2：每一页挂载时的请求（真实运行）

探针 `w2-inventory.spec.ts` 复制进副本的 `e2e/stories/` 运行之后删除；它打开每一页，记下 2 秒内没有新请求之前的每个 `/api/` 请求（路径中的 slug、id 换成占位）。三个账户：工作区 `acme` 的管理员（有一个项目）、没有工作区的人、新来的人。

**`7cf3a286`（之前）**：工作区的页面都只请求 `/api/v0` 的认证、实例、`me`、资料，外加两个 404：`GET /api/workspaces/{slug}/states/`、`GET /api/workspaces/{slug}/user-properties/`；工作区列表从不取，所以这些页面都显示"找不到工作区"。没有工作区的人和未登录的页面只请求 `/api/v0`。

**最终（之后）**，管理员：

| 页面 | `/api/v0` 的请求（200） | 仍是 404 的（都属于 P8b，按 store） |
|---|---|---|
| `/`（落点到 `/{slug}`） | auth/refresh、instance、me、me/profile、workspaces、me/workspaces/{slug}/preferences、workspaces/{slug}/members | 权限：`/api/users/me/workspaces/{slug}/project-roles/`；项目：`/api/workspaces/{slug}/projects/`；状态：`/api/workspaces/{slug}/states/` |
| `/{slug}` | 同上 | 同上 |
| `/{slug}/projects` | 同上 | 同上，加项目：`/projects/details/` |
| `/{slug}/settings` | 同上，加 timezones | 同上（权限、项目、状态） |
| `/{slug}/settings/members` | 同上，加 workspaces/{slug}/invitations（管理员） | 同上（权限、项目、状态） |
| `/{slug}/settings/projects/{id}` | 同上 | 权限、项目、状态，加项目 `/projects/{id}/`、权限 `/projects/{id}/project-members/me/`、项目成员 `/projects/{id}/user-properties/`、`/projects/{id}/members/`、标签 `/projects/{id}/issue-labels/`、状态 `/projects/{id}/states/` |
| `/{slug}/projects/{id}/issues` | 同上 | 同上一行 |
| `/{slug}-x`（不是成员） | auth/refresh、instance、me、me/profile、workspaces | 状态：`/api/workspaces/{slug}-x/states/`（只按 slug 启用，P8b 按列表启用） |
| `/workspace-invitations`（未登录） | instance、workspace-invitations/{id} | |
| `/workspace-invitations`（受邀的人） | auth/refresh、instance、workspace-invitations/{id}、me、me/profile | |

没有工作区的人：`/` 多一个 `GET /api/v0/workspaces`（落点），`/create-workspace` 和个人设置的四页与之前相同。未登录的 `/`、`/sign-up`、项目的工作项页只请求 `/api/v0/instance`；未完成引导的 `/onboarding` 与之前相同。之后的清单中没有收藏、通知、"最近"、迭代、模块、视图、分诊状态的请求。`/invitations` 和 `/no-such-page` 显示同一个"找不到工作区"的界面。

### A.6 W3、W4：分代和键

- M2 交接第 3 节的 `git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web`：37 处 → 27 处；M3 的 `WorkspaceService` 8 处都已消失（Task 9 之后）；M3 剩下的 2 处是 P8b 的（`project/form.tsx` 的 `ProjectService`、`project-member.service.ts` 的 `ProjectMemberService`）；其余 25 处是 M4–M7 的。
- M3 的每个 store 和 service 由 `RootStore` 用这一代的 `ApiClient` 建（`WorkspaceRootStore`、`WorkspacePreferencesStore`、`MemberRootStore`、`WorkspaceMemberStore`）；`previewInvitation` 由调用方传 `publicClient`。
- 键：工作区一侧的每个会话的取数经 `useSessionSWR`（2.13 的表）；工作区一侧直接用 `useSWR` 的只有 `useInvitationPreview`（第 3 节第 9 条）。P8a 改到的两个包装层里仍用 `useSWR` 的是项目一侧的取数（项目角色、项目列表、工作区的状态、项目详情和它的子资源），它们的键在 P8b 随各自的 store 改为 `useSessionSWR`。
- 静态检查（裁定 A6 的第二个条件，Task 8）：根目录 `.oxlintrc.json` 的第一条 `overrides` 对范围内的文件开 `no-restricted-imports`，`paths` 只有 `swr`，`allowTypeImports: true`（`use-workspace-fetch.ts` 的 `import type { SWRResponse }` 照旧）。范围（路径从仓库根写起：根配置的 `files` 按它所在的目录匹配，从 `web/apps/web` 写起的路径什么都不匹配，原型上量过）：`web/apps/web/core/store/workspace/**`、`web/apps/web/core/store/member/workspace/**`、`web/apps/web/core/components/workspace/settings/**`、`web/apps/web/core/components/profile/use-profile-member.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/lib/use-landing.ts`、`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`（`core/lib/wrappers/` 按文件列：`instance-wrapper.tsx` 用 `useSWR` 取公开的实例信息）。不在范围内的：`use-session-swr.ts` 本身、`fake-session-swr.ts`、唯一的例外 `core/hooks/use-invitation-preview.ts`。最终原型上 `check:lint` 照旧在上限；把 `use-workspace-fetch.ts` 或 `use-profile-member.ts` 的成员取数改回 `useSWR` 的两个变异（`t8-raw-swr-hook`、`t8-raw-swr-profile`）都让它失败，变异证明范围确实生效。第二条 `overrides` 对 P8a 的 M3 路径开 `typescript/no-non-null-assertion`（`error`；预检 L4）：上面的 store 和组件，加上 `core/store/user/permissions.store.ts`、`core/services/workspace/**`、`use-workspace-fetch.ts` 和它的测试、`core/lib` 中 P8a 新写的 13 个文件（逐个列出；`core/lib/**` 不行：M2 的 `token-manager.tabs.test.ts` 有 25 处）；变异 `t8-nonnull-email` 让它失败。两条规则在同一个 `overrides` 数组里，各有自己的文件列表（`use-session-swr.ts` 只在第二条的范围内）。`.oxlintrc.json` 由根目录的 `check:format`（oxfmt）排版，plan 的块已照它排好。
- **已知的限制**：`workspace-wrapper.tsx` 在 P8b 之前不在第一条的范围内：它仍用 `useSWR` 取 P8b 的项目角色、项目列表、工作区的状态，放进范围 `check:lint` 就失败（原型上量过）。它在工作区一侧的取数都已移进 `useWorkspaceFetch`（在范围内），包装层自己只调用这个 hook；这期间它若再直接 `useSWR` 一个会话的取数，只有评审能发现。P8b 把 `workspace-wrapper.tsx`、`project-wrapper.tsx` 加进范围（第 5 节）。
- `permissions.store.ts` 的 `WorkspaceService` 不再是模块级的实例，但仍是旧的 axios service（第 3 节第 4 条），在 P8b 删除；M2 交接第 3 节的关闭条件中这一处在 P8b 关闭。

### A.7 W12、W13

- **W12**（`w12.cjs`，TypeScript 的语法树）：P8a 改到的 161 个源文件（测试除外）中，`as` 61 → 53、`any` 38 → 25、`!` 0 → 0；新的 `as` 只有 `core/store/fake-root.ts` 一处（第 3 节第 11 条）。没有新的 `!` 由 `.oxlintrc.json` 的 `typescript/no-non-null-assertion` 守着（A.6，变异 `t8-nonnull-email`）。没有手写的、重述契约的类型；knip 没有未使用的导出。
- **W13**：7.8 的每一项都有一个 grep，最终原型上：系统内接受的路由和入口、方法、新手引导的一步，`RESTRICTED_URLS`，用户设置，`workspace-members/me`，"最近"小部件，M6、M7 的挂载时取数，Plane 的工作区类型，工作区图标的上传：都是 0 个文件；`IUserLite`、`is_bot`：2 个文件，都是 M4 的（第 5 节）；两种语言中删除的文案键都是 0 条（Task 3 删 14 条、Task 4 删 2 条、Task 11 删 9 条，每种语言）。六个删除的路径都不存在（`app/(all)/invitations`、`join-invites.tsx`、`settings.store.ts`、`user-user-settings.ts`、`workspace-image-upload-modal.tsx`、`home/widgets/recents/`）。`/invitations` 落到工作区的页面（`navigation.test.ts` 的 `gives /invitations, the in-app accept's page that is gone, to the workspace of that name`），在页面上与 `/no-such-page` 一样显示"找不到工作区"（A.5）。
- **删除的文案键**（预检 L5；评审核对：`check:sync` 只核对两种语言的键相同，两种语言都留下的键它看不见，所以每个 Task 的"完成时"写明这些键在 `en`、`zh-CN` 中都已不在，评审照这张表核对；没有为它加关键词规则）。两种语言相同，从快照逐个量出（`i18nkeys.py`）：

| Task | 文件 | 键 |
|---|---|---|
| 2 | `auth.json` → `errors.json` | `auth.errors.*` 的 50 条移到 `errors` 命名空间（不是删除，键名去掉 `auth.` 前缀） |
| 3 | `common.json` | `accept_and_join`、`back_to_home`、`go_home`、`join_a_workspace`、`no_pending_invites`、`onboarding.workspace.create_new`、`onboarding.workspace.join_existing`、`onboarding.workspace.join_title`、`onboarding.workspace.no_invitations`、`please_select_at_least_one_invitation`、`we_see_that_someone_has_invited_you_to_join_a_workspace`、`workspace_invites`、`you_can_see_here_if_someone_invites_you_to_a_workspace`（13 条） |
| 3 | `power-k.json` | `power_k.account_actions.workspace_invites`（1 条） |
| 4 | `workspace-settings.json` | `workspace_settings.settings.general.edit_logo`、`workspace_settings.settings.general.upload_logo`（2 条） |
| 11 | `common.json` | `show_all`、`show_less`（2 条） |
| 11 | `home.json` | `home.recents.title`、`home.recents.filters.all`、`home.recents.filters.issues`、`home.recents.filters.projects`、`home.recents.empty.default`、`home.recents.empty.issue`、`home.recents.empty.project`（7 条） |

### A.8 oxlint（7.9，R3）

- 有手改的源文件（测试除外）133 个，0 条警告（修订加了 `use-landing.ts` 和导航的两个文件，裁定 A3）。
- 只经机械步骤到达的文件 29 个，共 15 条，留给 P11 第 4 个任务（P8a review 第 6 节照录）：

| 文件 | 条数 | 规则 |
|---|---|---|
| `core/components/dropdowns/member/base.tsx` | 4 | `no-shadow` 3、`jsx-a11y/no-static-element-interactions` 1 |
| `core/components/dropdowns/member/member-options.tsx` | 2 | `no-unused-expressions` 1、`react-hooks/exhaustive-deps` 1 |
| `core/components/power-k/config/navigation/commands.ts` | 2 | `unicorn/consistent-function-scoping` 2 |
| `core/components/project/confirm-project-member-remove.tsx` | 1 | `jsx-a11y/tabindex-no-positive` 1 |
| `core/components/project/project-settings-member-defaults.tsx` | 2 | `promise/always-return` 2 |
| `core/components/web-hooks/create-webhook-modal.tsx` | 1 | `promise/always-return` 1 |
| `core/hooks/work-item-filters/use-work-item-filters-config.tsx` | 3 | `no-shadow` 3 |

  其余 22 个只经机械步骤到达的文件没有警告。
- 上限：web 应用 452 → 451（Task 4）→ 448（Task 6）→ 445（Task 7）→ 444（Task 8）→ 442（Task 9）→ 435（Task 11）；`types` 0、`constants` 1、`utils` 12 不变。
- M3 领域（`domains.mjs`）：71 条、38 个文件 → 63 条、32 个文件。
- `promise/always-return`：P8a 有手改的文件中的 7 条都已清零（邀请页 2 条、`add-project-members-modal.tsx` 1 条、`module.store.ts` 3 条、`project-view.store.ts` 1 条），工作区成员 store 原来的两处 `oxlint-disable-next-line promise/always-return` 随重写删除；只经机械步骤到达的 3 条在上表。
- 抑制：没有新的抑制；删去四处（两处 `always-return`，见上一条；两处 `no-array-sort`，`use-navigation-items.ts`、`tab-navigation-root.tsx` 改为 `toSorted`，第 3 节第 16 条）。

### A.9 死成员和死 prop（7.9）

M1 收尾的 `deadsym.mjs`、`domains.mjs`（`$M3TMP/p8tools/dead/`，`deadrun.py`）：

- `--rows M3`：`7cf3a286` 上成员 140、prop 68 → 最终成员 118、prop 62（修订加的行都是测试里 `vi.hoisted` 的桩和 `it.each` 的列，例如 `role`、`profile`、`workspaces`）。
- P8a 加的行（`7cf3a286` 上没有的）只有测试里的字面量（`it.each` 的表的列，例如 `change`、`who`，在标题的 `$change` 中读）和测试的假实现（`FakeNerve.replacedSessionClient`，只由测试读）。P8a 的删除留下的没有调用方的代码都已删除（第 3 节第 15 条）；P8a 重写的工作区一侧文件中 M3 的死行（`getMemberIds`、`sortWorkspaceMembers` 约束中的 `is_active`、两个 store 接口上没有读者的 map、`CreateWorkspaceForm` 和 `InvitationFields` 没有调用方传的 prop、`updateWorkspaceView`）在它们的 Task 删除。
- P8a 改到的文件中仍有的 M3 行和它们的去处：项目一侧的文件（P8b，第 5 节）；`types/src/workspace.ts` 的工作区搜索类型和 `workspace.service.ts` 的搜索参数（M4 的搜索，它们被读，没有写者是因为来自接口的回答）；`workspace-menu-root.tsx` 的 `open`、`close`（Headless UI 的渲染参数，被读，由库传入）；只经机械步骤到达的 `types/src/project/projects.ts`（M4 的工作项搜索参数）、`workspace-notifications.ts`（M7，设计第 1 节的 41 行之一）、`member-options.tsx` 的 `className`（P10 重写这个下拉框）。

### A.10 规模

plan 共 10,994 行：Task 1 之前（约束、一次性脚本、文件结构）224 行；各 Task 一节（从它的标题到下一个 Task 的标题；代码块的行数含围栏和提交命令的块）：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 会话分代的基础 | 484 | 380 | 13 |
| 2 文案表 | 648 | 543 | 20（11 个经机械步骤） |
| 3 系统内接受的删除 | 630 | 497 | 24 |
| 4 工作区的类型、service、store | 1,567 | 1,361 | 44（11 个经机械步骤） |
| 5 落点、设置 store、保留名单 | 1,338 | 1,160 | 28 |
| 6 权限 store、包装层取列表 | 954 | 820 | 25 |
| 7 `MemberUser`、成员 | 1,488 | 1,289 | 40（18 个经机械步骤，其中 6 个另有手改） |
| 8 邀请、成员页的取数 | 1,213 | 1,071 | 20 |
| 9 邀请链接 | 706 | 604 | 14 |
| 10 显示设置 | 786 | 685 | 14 |
| 11 M6、M7 的挂载时取数 | 856 | 716 | 30 |
| 12 文档 | 100 | 63 | 2 |

修订之后 Task 4 是 1,567 行（`fake-queue.ts` 的 `fetchedWhileChangeIsOut` 和完成时的文案键一句），比裁定 A1 认可的 1,542 行多 25 行，仍按"约 1,500 行"计；其余都在 1,500 行之内。

### A.11 关键词规则

`tools/keywords.json` 从 60 条到 64 条：`in-system-invitations`（Task 3）、`plane-workspace-urls`（Task 4，Task 5–10 逐步扩大）、`restricted-urls`、`user-settings`（Task 5）。每条规则的命中、不命中样例和 `files` 样例都由 `node tools/keywords.mjs` 核对，不命中样例还有每个新的 `/api/v0` 地址。模式中有项目一侧同类地址的，每一段都有一个项目一侧的不命中样例（P8b 还在用这些地址，模式不能宽到它们）：

| 规则 | 模式的一段（命中） | 项目一侧的不命中样例 |
|---|---|---|
| `plane-workspace-urls` | `"/api/users/me/workspaces/"`（列出） | `/api/users/me/workspaces/${workspaceSlug}/project-roles/`、`…/projects/invitations/` |
| | `"/api/workspaces/"`（创建） | `/api/workspaces/${workspaceSlug}/projects/` |
| | `` `/api/workspaces/${…}/` ``（查看、修改、删除） | `/api/workspaces/${workspaceSlug}/projects/${projectId}/`、`…/states/`、`…/views/` |
| | `/api/workspaces/${…}/members/`（成员、离开） | `…/projects/${projectId}/members/`、`…/members/${memberId}/`、`…/members/leave/` |
| | `/workspace-members/me/` | `…/projects/${projectId}/project-members/me/` |
| | `/api/workspaces/${…}/invitations/` | `…/projects/${projectId}/invitations/` |
| | `/api/workspaces/${…}/user-properties/` | `…/projects/${projectId}/user-properties/`、`…/cycles/${cycleId}/user-properties/`（M6） |
| | `/api/users/last-visited-workspace/`、`/api/workspace-slug-check/` | 没有项目一侧的同类地址 |
| `in-system-invitations` | `/api/users/me/workspaces/invitations/` | `/api/users/me/workspaces/${workspaceSlug}/projects/invitations/` |

另外 `plane-workspace-urls` 有 `/search-issues/`（M4）的不命中样例。`restricted-urls`、`user-settings` 的名称没有项目一侧的同类。最终原型上：64 条规则，3 个例外（M2 留下的），没有命中。

### A.12 逐 Task 复现

`replay.py` 在修订提交的树的副本（`7cf3a286` 的 `git archive` 加上修订之后的三份文档：M3 设计、本 spec、plan；`$M3TMP/p8replay`，`pnpm install --frozen-lockfile` 之后）上照 plan 逐个 Task 执行：先是 `Run（机械步骤）:` 的命令（`$P8ATMP` 中的两个一次性脚本从 plan 的"一次性脚本"一节原样取出），核对散列表；再写入块；核对 `go.mod`、`go.sum` 不变、`pnpm-lock.yaml` 与那个 Task 的快照相同；再执行这个 Task 的每个 `Run:`。副本不是 git 仓库，所以 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（它列文件不用 git，规则和样例相同），`make e2e` 只有 S3 因为读不到提交而失败（F4），由脚本单独认出。

| Task | 机械步骤 | 写入的文件 | 检查 |
|---|---|---|---|
| 1 | | 13 | `make lint-web`、`make knip`、`make test-web` 通过；`make e2e` 69 个通过（S3 除外） |
| 2 | `t2_imports.py` 11 个文件，散列和行数与表相同 | 9 | 同上 |
| 3 | | 24 | `make lint-go` 两段 `0 issues.`、`make test` 42 个包 `ok`；其余同上 |
| 4 | `rename_type.py` 三次（9、1、1 个文件）和 oxfmt，散列和行数与表相同 | 33 | `pnpm install --frozen-lockfile` 通过；其余同 Task 1 |
| 5 | | 28 | `make lint-go`、`make test` 同 Task 3；其余同 Task 1 |
| 6 | | 25 | 同 Task 1 |
| 7 | `rename_type.py` 四次（14、3、1、1 个文件）和 oxfmt，18 个文件的散列和行数与表相同 | 28 | 同 Task 1 |
| 8–11 | | 20、14、14、30 | 同 Task 1 |
| 12 | | 2 | 同 Task 1 |

修订之后重新复现：12 个 Task 都通过。最后的副本与原型逐文件相同：`treediff.mjs p8replay p8proto`：3,156 个文件，0 处差异（原型里也放了同样的三份文档）。副本上 web 应用的 vitest 是 29 个测试文件、246 个测试。
