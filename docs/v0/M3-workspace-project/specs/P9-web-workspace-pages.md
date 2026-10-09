# M3/P9 工作区的页面：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P9 `web-workspace-pages` |
| 日期 | 2026-10-09 |
| 状态 | 待裁定：第 3 节标"请裁定"的各条由控制者裁定之后执行；控制者的浏览器核对 C1–C5 和评审在 plan 的 Task 之后（M3 设计 9.7） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W1–W9、S2）、3.8、3.10、3.11、3.14、3.18、7.1、7.4、7.5、9.5、9.6、9.7、12（P9）、13.1 节；[总体设计](../../v0-design.md) 7.7 |
| 前置交接 | [P8a spec](P8a-web-workspace-data.md) 第 3 节第 12 条、第 5 节 P9 一行；[P8a review](../reviews/P8a-web-workspace-data-review.md) 第 6 节 P9 一行；[P8b spec](P8b-web-project-data.md) 第 5 节 P9 一行；[P8b review](../reviews/P8b-web-project-data-review.md) 第 6 节 P9 一行（E5）；[M2 收尾交接](../handoffs/M2-closeout.md) 第 1、2、11、13 节；P2、P3、P6 的 spec 第 5 节，P3、P4b、P6 的 review 第 6 节中 P9 的行；[M1-P2 交接](../handoffs/M1-P2-trim-content.md)；[Codex 设计评审](../reviews/M3-design-codex-adversarial-review.md) M-4、4.3 |
| 计划 | [P9 plan](../plans/P9-web-workspace-pages.md) |

本 spec 只写 M3 设计交给 P9 决定的东西：名字、签名、文件的位置、测试名，以及原型证明了什么。规则本身以 M3 设计和总体设计 7.7 为准，这里引用节号，不重述。P9 依赖 P8a、P8b（`4b1334a5` 的 `main`），不改契约、Go 代码和生成的文件。

P9 的每一处页面改动都按安全测试看待（brief）：一个角色不能发的请求在页面上是 403；换账户之后才落地的写以错误的身份写入（W3）；页面不能显示角色读不到的东西。所以附录 A 的变异逐个核对：页面发出修改之后的跳转、提示和界面变化只在发出它的会话里进行（2.13 的每一处，W3 的会话切换）；每一页挂载时只取它的角色能读的（成员页的邀请只为管理员取，S2 的清单，2.12）；每个请求体只有表单编辑的字段，类型来自生成的客户端，角色是数字；每个错误经 `errorMessageKey` 显示；取决于 store 所持的值的修改在轮到它时算（E5）。只有评审才能发现的会话、权限或取数的性质算缺口：没有（附录 A.2）。

## 1. 目标

按 M3 设计 12 节 P9 和第 2 节的故事：

- 落点的页面一侧：切换工作区、创建工作区、接受邀请之后写 `last_workspace_id`，删除、离开之后经根路径落点（3.14、7.4）。
- 新手引导按调用者的工作区列表决定步骤：已有工作区的人资料一步之后完成；没有的人创建工作区、邀请成员（复制每个邀请的链接，v0 不发邮件）；创建关闭时说明（7.4，决策点 2）。
- 邀请页、带邀请的注册（7.4，决策点 1）；工作区首页和侧边栏的工作区部分、项目导航对话框（7.5，W8）；工作区设置 general、members（成员、邀请）（7.5）；停用弹窗显示 409 的说明（W9）；`WorkspaceAuthWrapper` 的界面（M2 交接第 13 节）。
- 页面级副作用的会话核对（7.1、9.6）：经 `core/lib/in-session.ts` 的 `followInSession`；删除工作区组件的 vitest；W3 的会话切换端到端，去掉核对时它失败。
- W1–W9 的页面版本（W4 含成员打开成员页没有失败的请求），S2 的挂载清单加上 P9 的页面。
- P8 留给 P9 的三类缺陷在 P9 的页面上清零：封闭的请求体、错误的读法、请求体取自所持的值（附录 A.4）。
- 3.20 中 P9 的一行：README 的"前端"一节；前端改动清单。控制者的浏览器核对 C1–C5 和评审不在 plan 中（Task 11 只改文档）。

## 2. 交付物

**文件总览**：plan 的"文件结构"一节逐个列出 Task 1–11 改到的 101 个文件和每个文件所属的 Task：新文件 28 个（web 的 27 个：11 个模块、hook 和组件，2 个测试的共用部分，14 个测试文件；e2e 的 `fixtures/workspace-pages.ts`），删除 2 个（`delete-workspace-form.tsx`、`onboarding/steps/workspace/root.tsx`），修改 71 个。web 的 TS 文件 73 个（其中测试 19 个）都是手改，没有机械步骤；web 的其余 9 个是 `web/apps/web/package.json`（oxlint 的上限）和四个文案文件的两种语言；e2e 15 个（fixture 4 个、故事 11 个）；文档 2 个（`README.md`、`docs/v0/frontend-changes.md`）。契约、Go 代码、生成的文件、`tools/keywords.json`、`.oxlintrc.json` 和 `pnpm-lock.yaml` 都不变。

**依赖**：不加 npm 包或 Go 模块。

**共用的新部分**：
- `core/lib/in-session.ts` 加 `followInSession<T>(change, followers): Promise<void>`，`ChangeFollowers<T> = { done?: (answer: T) => void; failed: (error: unknown) => void }`（`sessionGuard` 旁边，Task 1；`done` 从 Task 2 起可省）：发出修改时取标签页的会话，修改兑现之后只在标签页仍在那个会话里时调 `done` 或 `failed`，另一个标签页把它换到别的账户之后两者都不调；自己从不拒绝。页面发出工作区一侧的修改之后的跳转、提示和界面变化都经它（2.13）。
- 测试的共用部分：`core/lib/auth/fake-tab.ts`（`api-client.ts` 的替身：`tokenManager`、`signedIn`、`switchAccount`、`heldChange`、`lateSettlings`、`pageSettled`；测试决定一个修改何时兑现，在换账户之前或之后），`core/lib/fake-refusal.ts`（`refusal(status, code, fields?)`：nerve 的拒绝，`ApiError`）。组件的测试照 P8b 的写法：服务端渲染（`renderToStaticMarkup`），`@nerve/ui` 的控件换成 P8b 的 `fake-controls.ts`，提示、文案用 `fake-toast.ts`、`fake-i18n.ts`。
- 页面的决定是纯函数，返回 `kind` 的联合，用 `it.each` 测：`invitation-view.ts`（Task 4）、`onboarding-place.ts`（Task 6）、`invite-modal/refusal.ts`（Task 3）、`use-create-workspace.ts` 的 `creationRefusal`（Task 5）。
- e2e：`e2e/fixtures/settings-pages.ts` 的 `sentTo(page, method, path, act)`：经路由在请求离开页面时读它的请求体（第 3 节第 12 条）；新文件 `e2e/fixtures/workspace-pages.ts`：工作区页面上一个人的操作（另一个浏览器、删除、重新登录、切换工作区、成员行、改角色、结束成员关系、邀请链接、邀请行、发出邀请、删除邀请）。

### 2.1 页面在会话里跟进修改；general 页和删除工作区；W3 的页面版本（Task 1；7.1、7.5、9.5、9.6；P8a 的 P12、P4b 的 P26）

- `DeleteWorkspaceModal`（整个文件，收 `workspace: Workspace`）：表单并入弹窗（`delete-workspace-form.tsx` 删除），按提交的值核对名称和"delete my workspace"；`deleteWorkspace(workspace)` 经 `followInSession`：成功时关闭、回到 `/`（落点按剩下的工作区决定，3.14）并提示，失败时按 `code` 提示并停在弹窗。`delete-workspace-section.tsx` 传地址的工作区。文案：被拒绝时说 nerve 的原因，`delete_modal.error_message`（"Try again, please."）没有了读者，两种语言都删除；`delete_modal.success_message` 原来说"即将跳转到个人资料页"，改为去另一个工作区或创建工作区（附录 A.8）。
- `WorkspaceDetails`（general 页）：地址的工作区经 `useParams` 和 `getWorkspaceBySlug`（P8b 的 P13：页面只按路由读地址）；修改只发表单编辑的 `name`、`organization_size`（没有时不发）、`timezone`（生成的 `WorkspaceUpdate`）；规模没有时表单的值是 `null`，选择框一开始就受控；结果经 `followInSession`，拒绝按 `code` 提示，按钮结束加载（P12）。
- 测试：`in-session.test.ts` 改用 `fake-tab.ts`，加 `followInSession` 的三个（在会话里跟进回答和拒绝；换账户之后兑现的两种都不跟进；换账户之后才发出的修改照常跟进）。`delete-workspace-modal.test.tsx` 6 个（9.5：删除工作区的组件，修改在 `loginId` 改变之后才兑现时不跳转、不提示）、`workspace-details.test.tsx` 5 个。
- 端到端：W3 的两个页面版本（修改、成员的只读视图、被降级的管理员被拒绝并看到原因、删除之后落到另一个工作区；会话切换：`holdAnswer` 先 `route.fetch()` 把删除发到 nerve，另一个标签页以另一个账户登录之后才 `route.fulfill()`，原标签页不跳转、不提示）。`e2e/fixtures/api.ts` 的 `furnishWorkspace` 给工作区每张删除会写的表一行（W2 的 API 版本原来自己写的一段移到这里，W3 的页面版本也用它）；W3 的页面版本不先删除项目，删除之后的断言传 `["workspace_member_invites"]`（P26：成员接受时那条邀请已先被删除）。`changeRole`（经 API 改成员的角色）给 W3 的被降级一段，W9 也用它。S2 加每个角色的 `/{slug}/settings`（2.12）。

### 2.2 成员页的三个修改；W7、W2 的页面版本（Task 2；W7、W2，3.14，7.1，7.5；P8a 的 P21、P22）

- `useMembershipChanges(workspaceSlug)`（`core/components/workspace/settings/use-membership-changes.ts`）：`changeRole(userId, role: WorkspaceRole)` 发 `{ role }`（生成的 `WorkspaceMemberUpdate`，编号）、`remove(userId)`、`leave()`（成功之后回到 `/`），都经 `followInSession`，拒绝按 `code` 提示。原来三处各自的 `try/catch` 读 `err.error`（P21），换账户之后仍跳转。
- `workspace-roles.ts` 的 `WORKSPACE_ROLES: WorkspaceRole[] = [5, 15, 20]`：成员页的角色列、Task 3 的邀请行和邀请列表共用。`member-columns.tsx` 的 `AccountTypeColumn` 去掉从没有提交过的 `useForm`/`Controller` 和 `value as EUserPermissions`；`members-list-item.tsx`（整个文件）收地址的 `workspaceSlug`（`members-list.tsx` 传入，不读 `currentWorkspace`），自己一行是"离开"，别人的是"移出"。
- 测试：`use-membership-changes.test.ts` 12 个（角色是编号、发到成员关系的工作区；移出之后停在原页；离开之后落点；三种修改被拒绝的 `it.each`；三种修改各两种换账户之后兑现）。
- 端到端：W7 的页面版本（`PATCH` 的请求体 `{ role: 5 }`，P22；唯一的管理员离开被拒绝，页面说明原因；成员看得到角色、改不了）；W2 的页面版本（落点、经工作区菜单切换写的 `last_workspace_id`、删除和离开之后的落点，没有失败的请求）。S2 加每个角色的 `/{slug}/settings/members`：邀请只为管理员取（2.12）。

### 2.3 邀请的表单和列表；W4 的页面版本（Task 3；W4，7.1，7.4，7.5；P8a 的 P21、P22，P8a spec 第 3 节第 12 条）

- `use-workspace-invitation.tsx`（整个文件）：表单的值就是生成的 `WorkspaceInvitationsCreate`（Plane 的 `InvitationFormValues`、`EmailRole` 和 `role: EUserPermissions` 删除）；提交经 `followInSession`；成功时清空、提示；被拒绝时 `invitationRefusal` 决定原因放在 nerve 指出的行下（`setError("invitations.<i>.email", …)`）还是提示。原来在 `catch` 里读 `err.error` 再重新抛出（没有处理的拒绝）。
- `invite-modal/refusal.ts`：`invitationRefusal(error, rows)`，`{ kind: "rows"; rows } | { kind: "toast"; message }`：nerve 的字段错误都指向表单里的行（`invitations[<i>].email`）时落到行下（`not_allowed` 是"已是成员"，`duplicate` 是"已有邀请"，其余按 `FIELD_ERROR_MESSAGES`）；有一条不在行上时整个按 `errorMessageKey` 提示。
- `core/lib/invitation-link.ts` 的 `invitationLink(origin, invitation)`：`/workspace-invitations?invitation_id=…&token=…`，经 `URLSearchParams` 编码（原来在列表项里拼字符串，令牌不编码；9.5）。
- 弹窗收 `invite`（成员页传 `inviteMembersToWorkspace`，它自己的 `handleWorkspaceInvite` 删除）；行的角色选项是 `WORKSPACE_ROLES`。邀请列表：已忽略的邀请（`responded_at` 不为空）标"已忽略"，没有角色选择和复制链接，只能删除（P8a spec 第 3 节第 12 条）；删除、改角色经 `followInSession`，拒绝按 `code` 提示；复制失败有提示。文案 `members.declined`、`modal.errors.already_member`、`already_invited`。web 的 oxlint 上限 360 → 359。
- 测试：`refusal.test.ts` 6 行、`invitation-link.test.ts` 5 个（Task 4 加 2 个）、`members/invite-modal.test.tsx` 6 个（请求体是 `WorkspaceInvitationsCreate`、角色是编号；不是地址的不发；行的拒绝不关；整个的拒绝提示原因；两种换账户之后兑现）。
- 端到端：W4 的页面版本（成员的页面不显示也不请求邀请；已忽略的标"Declined"；两行一次发出，角色是编号；改角色 `{ role: 15 }`；复制的链接；先经 API 删掉的邀请，删除得到 404 并提示；已是成员、已忽略的邀请的邮箱各在自己的行下被拒绝；删除已忽略的之后可以再邀请）。`assert/workspace.ts` 的 `expectInvitations` 让同一邮箱的两条按创建时间排（第 3 节第 13 条）。

### 2.4 邀请页；打开刚加入的工作区；W5 的页面版本（Task 4；W5，3.8，3.14，7.1，7.4；决策点 1；P8a 的 P27；Codex M-4）

- `invitation-view.ts`：`invitationView({ link, preview, signedIn, mismatched })`：链接缺 id 或令牌、nerve 按链接找不到（400、404）是 `invalid`；nerve 没有回答是 `unavailable`；预览在路上是 `loading`；然后已忽略 `declined`、没有登录 `sign-in`、nerve 对调用者的回答说是别的邮箱的 `mismatch`，其余 `answer`。
- `use-open-workspace.ts`：`useOpenWorkspace()` 给出 `(workspace) => Promise<void>`：写成上次打开的工作区（`updateUserProfile({ last_workspace_id })`，尽力而为），再去 `/{slug}`，经 `followInSession`。接受邀请、创建工作区（Task 5）之后用它。
- `invitationAuthPath("/" | "/sign-up", invitation)`：从链接去登录或注册的地址，带 `invitation_id`、`token`（页头显示要加入的工作区，Task 7 的注册带上邀请）和回到链接的 `next_path`。
- 邀请页（整个文件）按 `invitationView` 显示，文案经 `t()`（`workspace_invitation.*`，14 条，两种语言），标题写工作区和角色，不写邮箱；接受经 `followInSession`，成功之后 `useOpenWorkspace`；忽略之后重读预览；nerve 拒绝时：邮箱不一致（`workspace.invitation_email_mismatch`）转到 `mismatch`，只给"退出登录"（Codex M-4：打开时无从知道）；其余按 `code` 提示并重读预览。P27 的四条都在这里：缺 id 或令牌不再一直转圈；接受、忽略之后不再显示缓存的预览；没登录的人不再看到"接受"（原来得到 401 进 `console.error`）。
- 测试：`invitation-view.test.ts` 10 行；`use-open-workspace.test.ts` 4 个；`invitation-link.test.ts` 加 2 个。
- 端到端：W5 的两个页面版本（未登录 → 登录再回到链接 → 接受、落在工作区、写成上次打开的；另一条忽略之后说已忽略；链接打开时被删除的邀请，接受被拒绝、说明原因，之后无效；另一个邮箱的邀请，接受被拒绝之后说发给了另一个邮箱、只给退出登录、什么都不变；令牌改一位或没有令牌：无效）。

### 2.5 创建工作区的共用 hook；`/create-workspace`；W1 的页面版本（一）（Task 5；W1，3.10，3.11，3.14，7.1；P8a 的 P9、M7 和第 5 节的 `t5-slug-ignored`）

- `use-create-workspace.ts`：`CreationForm`（`Pick<WorkspaceCreate, "name" | "slug"> & { organization_size: OrganizationSize | null }`）；`slugFrom(text)`（小写，空格换成 `-`：字段显示的就是发出的）；`creationRefusal(error)`，`{ kind: "fields"; fields } | { kind: "toast"; message }`（检查之后被占用的 `workspace.slug_taken` 落到 slug 下；字段错误只指向 `name`、`slug` 时落到字段下，`not_allowed` 是保留、`invalid_format` 是格式不对；否则按 `errorMessageKey` 提示）；`useCreateWorkspace()` 给出 `(form, setError) => Promise<Workspace | undefined>`：先问 nerve slug 能不能用（`checkWorkspaceSlug`），不能时不发，原因放在 slug 下；能用时发 `{ name, slug, organization_size? }`；成功时提示并交回工作区；经 `followInSession`，换账户之后既不提示也不交回。`/create-workspace` 和 Task 6 的新手引导共用它。
- `CreateWorkspaceForm` 收 `onCreated(workspace)`，表单自己的 slug 检查（`validateSlug`、`invalidSlug`、`slugError` 和两段文字）删除：nerve 的检查是唯一的检查（第 3 节第 8 条）。`/create-workspace` 创建之后 `useOpenWorkspace`；创建关闭时说明已关闭、请工作区的管理员给一个邀请链接，与新手引导一致（W1"两个入口都显示'创建工作区已关闭'"），Plane 的"只有实例管理员能创建"和给实例管理员写信的按钮删除（第 3 节第 7 条）。
- 文案：`url_alphanumeric` 改成 nerve 的规则，加 `url_reserved`；`creation_disabled` 的标题和说明改写；`creation_disabled.request_button`、`request_email.subject`、`request_email.body` 删除（附录 A.8）。
- 测试：`use-create-workspace.test.ts` 16 个（检查、创建、提示、交回；没有规模时不发；三种不能用的 slug 不发、原因在 slug 下；字段的拒绝落到字段；没有字段的拒绝提示；两种换账户之后兑现；`creationRefusal` 5 行；`slugFrom` 2 行）。P8a 第 5 节的 `t5-slug-ignored`（创建不等 nerve 对 slug 的回答）从这里起由 vitest 和端到端两层发现（`T5.1`）。
- 端到端：`/create-workspace` 的 W1 页面版本（三种 slug 都没有 `POST`，原因在字段下；以字段显示的 slug 创建，请求体经 `sentTo`；打开、写成上次打开的；创建关闭的 nerve 上说明已关闭，没有表单，没有 `mailto:` 链接）。

### 2.6 新手引导；W1 的页面版本（二）（Task 6；W1，7.1，7.4；决策点 2；P8a 的 P7、P21、M4、M7，P8a spec 第 3 节第 12 条；M2 交接第 2、11 节的页面一侧）

- `onboarding-place.ts`：`OnboardingPlace`（`kind` 是 `EOnboardingSteps` 的三步，邀请一步带它邀请到的工作区）；`resumedPlace(profile, workspaces)`：资料、创建都已完成、还没邀请、创建的工作区（写成上次打开的那个）仍在列表中时在邀请一步（邀请到它，不是列表的第一个，M4）；资料已完成、没有创建、没有工作区时在创建一步；其余在资料一步。`afterProfile(workspaces)`：有工作区的人资料一步之后完成，没有的去创建。
- `onboarding/root.tsx`（整个文件）：`OnboardingRoot` 取调用者的工作区列表（M7：原来读一份没人取过的列表），到之前显示加载，取不到时 `SessionUnavailable`；`OnboardingSteps` 按 `onboarding-place.ts` 决定步骤，资料一步之后、创建之后、完成时的修改都经 `followInSession`，拒绝按 `code` 提示；创建的工作区与它的步骤一起写成上次打开的。`profile.store.ts` 的 `finishUserOnboarding` 不再把列表中的第一个工作区写成上次打开的。
- 步骤：`steps/root.tsx` 按 `place` 显示，收 `onNamed`、`onCreated(workspace, alone)`、`onDone`；"创建或加入"的包装 `steps/workspace/root.tsx` 删除；创建一步改用 `useCreateWorkspace`（它自己的 slug 检查删除，slug 框的红边原来读 `errors.name`，改为 `errors.slug`）；邀请一步（`steps/team/root.tsx`，整个文件）用成员页的表单（`useWorkspaceInvitationActions`、`InvitationFields`），邀请到这一步的工作区，发出之后 `steps/team/links.tsx` 列出每个邀请的链接和"复制链接"，再"继续"；"以后再说"结束引导（第 3 节第 9 条）。`useCopyInvitationLink()`（`core/hooks/use-copy-invitation-link.ts`）由邀请列表和这一步共用。`use-workspace-invitation.tsx` 的 `invite` 交回 nerve 建的邀请，`onSent(invitations)` 取代 `onClose`，`clear()` 取代 `handleClose`。
- 删除：`@nerve/utils` 的 `validateSlug`、`SLUG_REGEX`；文案 `onboarding.invite.role`、`onboarding.invite.not_an_email`、`workspace_creation.toast.error.title`、`message`；改写 `onboarding.workspace.creation_disabled`（P7）；加 `onboarding.invite.links.title`、`description`。
- 测试：`onboarding-place.test.ts` 8 行；`onboarding/root.test.tsx` 8 个（9.5：已有工作区的人资料一步之后完成）。
- 端到端：W1 的四个新手引导版本（新人：资料、创建、写成上次打开的、邀请到它、复制链接、完成、落在它；新人在邀请一步"以后再说"：落在他建的工作区，没有邀请的请求；先接受了邀请的人资料一步之后完成；接着做的人邀请到他建的工作区，不是列表的第一个；创建关闭的 nerve 上资料一步之后说明要邀请链接）。S2 加新人打开 `/`（2.12）。

### 2.7 带邀请的注册（Task 7；W6，3.8，7.4；决策点 1；M2 设计 3.18）

- `auth-root.tsx`：地址带 `invitation_id` 和 `token` 时把 `{ id, token }` 交给密码表单；`password.tsx` 的注册发 `{ email, password, invitation }`（生成的 `RegisterRequest`，注册关闭时 nerve 凭它放行），登录不带。原来注册页的页头显示要加入的工作区，注册本身却不带邀请。
- `auth-screens/header.tsx`：登录页和注册页之间的链接带上当前页面的查询参数（邀请和 `next_path`）。
- 测试：没有新的 vitest（注册的请求体由端到端经 `sentTo` 钉住）。端到端：W6 的两个页面版本（注册关闭：没有链接时注册页说已关闭；凭链接注册，请求体带邀请；回到链接接受，新手引导只有资料一步，落在工作区；注册开放：从链接的注册页去登录页，登录页保留链接、登录之后回到它）。

### 2.8 导航设置的修改在轮到它时算；项目导航对话框；首页的导览；W8 的页面版本（Task 8；W8，3.18，7.1，7.5；总体设计 7.7；P8b 的 E5，P8a 的 P28）

- `WorkspacePreferencesStore.updatePreferences(slug, change: PreferencesChange)`，`PreferencesChange = (held: WorkspacePreferences) => WorkspacePreferencesUpdate`：在队列轮到它时作用于 nerve 最近一次回答的设置再发出（照 P8b 的 `ProjectPreferencesStore`）；还没有设置时不发出，以 `Workspace settings not found` 失败。原来收现成的请求体，页面按调用时侧边栏显示的值算（E5）。
- `core/hooks/navigation-preferences.ts`（整个文件）：`NERVE_DEFAULTS`（`ACCORDION`、10）是 `navigationOf` 的缺省参数（设置到达之前按默认值显示，不再从"全部项目"跳到 10 个；`@nerve/types` 的 `DEFAULT_PROJECT_PREFERENCES` 删除）；`preferencesChangeOf(change): PreferencesChange`（打开限制时取 `navigationOf(held)` 的数量，关闭是 0）；`countOf(draft)`（数字组成的数，至少 1）。
- `use-navigation-preferences.ts`（整个文件）：`useProjectNavigationPreferences()` 给出 `{ preferences, changeNavigation }`，经 `followInSession`，拒绝按 `code` 提示（原来三个更新函数，被拒绝时是没有处理的拒绝）。`ProjectNavigationDialog`：数量框的文字是草稿（`null` 时显示设置的数量），失去焦点时发出一次；标签经 `htmlFor`/`useId` 关联输入框。`sidebar-wrapper.tsx` 打开对话框的按钮有名字（`aria_labels.projects_sidebar.project_navigation`）。`home/root.tsx` 的导览结束经 `followInSession`，拒绝按 `code` 提示（原来只有 `console.error`）。
- 测试：`preferences.store.test.ts` 的修改都写成函数，加 2 个（作用于前一个修改的回答；没有设置时不发出）；`navigation-preferences.test.ts` 12 行；`use-navigation-preferences.test.ts` 5 个；`home/root.test.tsx` 4 个。
- 端到端：W8 的页面版本（跳过导览的请求体；默认值下四个项目都显示；标签页式；关闭、打开限制；输入 30 删一位再离开，只发一次 3；刷新之后仍是 3 个、对话框显示 3；nerve 拒绝 3000000000（422），页面说明，侧边栏仍是 3 个；一共五个 `PATCH`）。

### 2.9 停用弹窗（Task 9；W9，3.7，3.9，7.1，7.5；P6 review 第 6 节）

- `DeactivateAccountModal`：nerve 拒绝停用时（409 `workspace.sole_admin`、`project.sole_admin`），原因（`errorMessageKey`）写在弹窗里（`role="alert"`），弹窗不关；关闭或再次确认时清掉。原来是提示。停用成功时，只在它结束了标签页的会话时提示"账户已停用"并关闭（第 3 节第 3、4 条）。
- `UserStore.deactivateAccount(): Promise<boolean>`（M2 的 store，原来 `Promise<void>`）：发出时读标签页的会话，nerve 停用之后结束它（`tokenManager.endSession(loginId)`），交回 `endSession` 的回答。
- 测试：`deactivate-account-modal.test.tsx` 3 个；`user/index.test.ts` 加 1 个 `it.each`（2 行）。端到端：A12 被拒绝的一段改读弹窗里的原因；W9 的页面版本（409 的原因在弹窗里，账户和成员关系不变；取消再打开，原因不在了；把成员改成管理员之后停用成功，回到登录页，成员关系结束）。

### 2.10 工作区列表的唯一取数；`WorkspaceAuthWrapper` 的界面（Task 10；7.1，7.5；P8a review 的 M7；M2 交接第 13 节）

- `core/hooks/use-workspaces-fetch.ts`：`useWorkspacesFetch(wanted = true)`，调用者的工作区列表的唯一取数（`useSessionSWR(wanted ? ["WORKSPACES"] : null, …)`）。落点（`use-landing.ts`）、工作区的页面（`use-workspace-fetch.ts`）、新手引导（Task 6 的取数）改用它；个人设置的侧边栏（`settings/profile/sidebar/workspace-options.tsx`）挂载时经它取列表（M7：直接打开 `/settings/profile/*` 时侧边栏和命令面板的工作区菜单原来是空的）。
- `WorkspaceAuthWrapper`："找不到工作区"的界面经 `t("workspace_not_found.*")`（两种语言各 5 条），退出登录失败的提示经已有的 `auth.sign_out.toast.error.*`（原来写死的英文）。`workspace-wrapper.test.tsx` 改为核对键。"找不到"的判断不变（7.5、P19）。
- 端到端：S2 加直接打开个人设置（2.12）。

### 2.11 文档（Task 11；3.20 中 P9 的一行；M2 交接第 2 节）

- `README.md` 的"前端"一节："M2 中看到的页面"改写为"能用的页面（M3/P9 起）"。
- `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态；3.2"所有处理接口错误的地方"写明 P9 改到的页面；新增五行（页面跟进修改的结果、创建工作区、新手引导的邀请成员一步、侧边栏的项目导航设置、停用账户的弹窗），都是"已完成，M3/P9"。

### 2.12 挂载时的取数（W2、W4；7.1；S2）

S2 的 `REQUESTS` 是挂载清单唯一的地方（P8b 的裁定 D5），P9 加的行（`e2e/stories/smoke/s2-web-app.spec.ts`；附录 A.3 是这时的记录）：

| 页面 | 管理员 | 成员 | 访客 | 不是项目成员的成员 | 条件 |
|---|---|---|---|---|---|
| `/{slug}/settings`（Task 1） | `APP`、`WORKSPACE`、`GENERAL` | 同管理员 | `APP`、`WORKSPACE` | 同管理员 | 时区由 general 页取；工作区设置不给访客 general 页（9.2） |
| `/{slug}/settings/members`（Task 2） | `APP`、`WORKSPACE`、`INVITATIONS` | `APP`、`WORKSPACE` | 同成员 | 同成员 | 邀请只在调用者是管理员时取（P8a 的 `use-members-settings-fetch.ts`） |
| 新人打开 `/`，到新手引导（Task 6） | | | | | `APP`：新手引导的列表与落点的同一个键，一次 |
| 直接打开 `/settings/profile/general`（Task 10） | | | | | `APP`：个人设置的侧边栏取列表 |

`APP` 是 `POST /api/v0/auth/refresh`、`GET /api/v0/instance`、`GET /api/v0/me`、`GET /api/v0/me/profile`、`GET /api/v0/workspaces`；`WORKSPACE` 是工作区包装层的四个（显示设置、成员、未归档的项目、工作区的状态）；`GENERAL` 是 `GET /api/v0/timezones`；`INVITATIONS` 是 `GET /api/v0/workspaces/{slug}/invitations`。每张清单是恰好的：多一个、少一个、重复的请求，`/api/v0` 之外的请求和失败的请求都让 S2 失败（`watchPage`）。邀请页的预览经 `publicClient`（`useInvitationPreview`，P8a），不是会话的取数。

### 2.13 页面级副作用的会话核对（7.1：组件逐个列出）

页面在修改兑现之后跳转、提示、改界面的每一处，都经 `followInSession`（只在发出它的会话里跟进），或不跟进：

| 组件 | 修改 | 跟进 | Task |
|---|---|---|---|
| `delete-workspace-modal.tsx` | 删除工作区 | 关闭、回到 `/`、提示；拒绝提示 | 1 |
| `workspace-details.tsx` | 改名、规模、时区 | 提示；拒绝提示 | 1 |
| `use-membership-changes.ts` | 改角色、移出、离开 | 离开之后回到 `/`；拒绝提示 | 2 |
| `use-workspace-invitation.tsx`（成员页的弹窗、新手引导的邀请一步） | 批量邀请 | 清空、提示、交给 `onSent`；拒绝落到行下或提示 | 3、6 |
| `invitations-list-item.tsx` | 删除邀请、改邀请的角色 | 拒绝提示 | 3 |
| `workspace-invitations/page.tsx` | 接受、忽略 | 打开工作区（`useOpenWorkspace`）、重读预览；邮箱不一致的界面；拒绝提示 | 4 |
| `use-open-workspace.ts` | 写上次打开的工作区 | 去工作区（成功、失败都去） | 4 |
| `use-create-workspace.ts`（`/create-workspace`、新手引导的创建一步） | slug 的检查和创建 | 原因在字段下、提示、交回工作区 | 5 |
| `onboarding/root.tsx` | 资料一步、创建一步写进资料，完成引导 | 拒绝提示 | 6 |
| `use-navigation-preferences.ts`（对话框） | 导航设置 | 拒绝提示 | 8 |
| `home/root.tsx` | 导览结束 | 拒绝提示 | 8 |
| `deactivate-account-modal.tsx` | 停用账户 | 只在结束了标签页的会话时提示并关闭；拒绝写在弹窗里（第 3 节第 3、4 条） | 9 |

不跟进的：工作区菜单的切换（链接直接跳转，写 `last_workspace_id` 不等回答，P8a）；退出登录失败的提示（退出失败时标签页的会话没有变）。M2 的账户页面（新手引导的资料一步自己的提示、个人设置的三页）在第 5 节 P11。

## 3. 与设计的差异、补充和需要裁定的

标"**请裁定**"的各条需要控制者裁定；其余是本 Phase 在设计之内做的决定和理由。

1. **请裁定：Task 的切分**：设计 12 节 P9 的草稿是 14 个任务，plan 是 11 个（P7、P8 的先例：在上限之内重新切分）。对应：设计 1（落点的页面一侧）分散在写 `last_workspace_id` 和落点的各 Task（1、2、4、5、6）；2 → 6；3 → 4；4 → 7；5 → 8；6 → 1；7 → 2；8 → 3；9 → 9；10 → 10；11（会话核对）→ 1（`followInSession`、删除组件的 vitest、W3 的会话切换）和各 Task 的页面；12、13（端到端）并进各页面的 Task（测试与代码同一个提交，每个 Task 之后 `make e2e` 通过）；14 → 11（只改文档；C1–C5 和评审是控制者的）。11 个 Task 都在约 16 个之内，最长的 Task 6 是 1,475 行（附录 A.9）。
2. **请裁定：会话核对写在一处**（7.1、9.6 写的是"组件在 `await` 之后先核对 `inSession()`"）：每个组件自己取 `sessionGuard()`、`await`、核对、再跟进，十几处写同一段；P9 把它写成 `followInSession(change, { done?, failed })`，组件只给出跟进什么。9.6 的变异"去掉组件里 `inSession()` 的核对"在这里是去掉 `followInSession` 的核对（`T1.1`），由 W3 的会话切换端到端和三个 vitest 文件发现；每个组件"不经 `followInSession` 自己跟进"由各自的 vitest 发现（附录 A.2 的 W3 一类）。
3. **请裁定：停用的成功经 `endSession` 的回答跟进，M2 的 `UserStore.deactivateAccount` 交回 `boolean`**：停用成功时自己结束标签页的会话，`sessionGuard()` 的核对随之为假，`followInSession` 用不上；而另一个标签页在停用发出之后把这个标签页换到别的账户时，原来的弹窗照样提示"账户已停用"、关闭，显示在那个账户的页面上（W3 一类）。`tokenManager.endSession(loginId)` 已经回答"标签页的记录是否仍是那个会话的"，store 交回它，弹窗只在为真时提示、关闭。签名改在 M2 的 store：它唯一的调用方是这个弹窗。
4. **请裁定：停用被拒绝的原因是弹窗自己的状态，不另加会话核对**：另一个标签页以另一个账户登录时，`AuthenticationWrapper` 在新账户的资料到达之前显示加载（`!currentUser`），页面连同弹窗卸载，之后兑现的拒绝写进的是已卸载的组件，不显示。所以拒绝一支直接 `setRefusal`（注释写明理由）；`T9.3` 去掉成功一支的核对时由弹窗的 vitest 发现。
5. **请裁定：W1 的"含大写"**：故事写"slug 已被占用、是保留名、含大写：表单在字段下方提示，不提交"。slug 字段照 Plane 的显示把输入转成小写（`slugFrom`），所以"含大写"到不了 nerve，也不会被提示；P9 让字段显示的就是发出的（原来显示转换后的、发出原文）。端到端的"格式不对"用 `café`（nerve 的规则只收小写字母、数字、`-`、`_`）；大写由字段的转换核对：W1 输入大写和空格，字段显示的、发出的都是小写、`-` 连接的 slug（`slugFrom` 的 vitest；变异 `T5.5`、`T5.9`）。
6. **请裁定：`members-list.tsx` 89 → 91**（brief："只为使用方改到的 Plane 文件不变长"）：成员行收地址的工作区（`workspaceSlug` prop），格式化把一行拆成三行。其余变长的 Plane 文件都是本 Phase 的对象（页面行为改在它们里面，附录 A.6）。
7. **`/create-workspace` 创建关闭的文案**：W1 写"两个入口都显示'创建工作区已关闭'"；`/create-workspace` 原来显示 Plane 的"只有实例管理员能创建工作区"和给实例管理员写信的按钮（`mailto:`，信的主题和正文两条文案）。Nerve 没有实例管理员（M2 设计 3.16），创建由部署的配置关闭；P9 改为与新手引导相同的说法（请工作区的管理员给一个邀请链接），删除按钮、页面里拼信的 `getMailtoHref` 和三条文案（按钮、信的主题、正文）。
8. **表单自己的 slug 检查删除，nerve 的检查是唯一的**（W17）：Plane 的 `validateSlug` 允许 Unicode 字母、文案与 nerve 的不同，还可能与 nerve 的原因一起显示两遍。现在不合格的 slug 在点"创建"之后由 nerve 说明（`checkWorkspaceSlug` 的三种原因），不再在输入时提示；`@nerve/utils` 的 `validateSlug`、`SLUG_REGEX` 在新手引导也不用之后删除（Task 6）。
9. **新手引导的邀请一步用成员页的表单**（7.4"邀请成员一步调 `createWorkspaceInvitations`，完成后显示每个邀请的链接供复制"）：Plane 的三行表单（Listbox 的角色、自己的邮箱检查和文案）换成成员页的 `InvitationFields` 和 `useWorkspaceInvitationActions`：一行起，角色是编号，行的错误来自 nerve（`invitationRefusal`）；发出之后列出链接（`links.tsx`）。W1 的故事写"点跳过"，页面上是"以后再说"（Plane 的按钮），它结束引导，W1 的第二个新手引导版本核对它（`T6.10`）。
10. **侧边栏在设置到达之前按 nerve 的默认值显示**（P28 的"侧边栏在显示设置到达前后的切换"）：`navigationOf` 的缺省是 nerve 的默认值（折叠式、10 个），一个从没改过设置的人在设置到达时侧边栏不变；`DEFAULT_PROJECT_PREFERENCES`（Plane 的，显示全部）删除。
11. **数量在失去焦点时发出**（P28 的"失焦或防抖"）：选失焦，不用计时器：数量框是草稿，离开时发一次（按 Escape 关闭对话框时 Headless UI 先移走焦点，同样发出，原型中核对过）。
12. **`sentTo` 的写法**（e2e 的共用部分）：页面的客户端发出的 `fetch`，Playwright 的 `response.request().postDataJSON()` 读不到请求体，所以经路由在请求离开页面时读；路由按路径的模式匹配，留到页面关闭：在页面发出下一个请求时移除路由，那个请求可能一直挂着（W1 的创建之后紧接着写资料，函数路由加移除在 4 次中失败 1–3 次，模式路由加移除 8 次中失败 1 次，模式路由不移除 12 次都通过）。
13. **`expectInvitations` 的顺序**（P3 的 fixture）：W4 的页面版本中同一邮箱先后有两条邀请（删除已忽略的那条之后再邀请），原来按邮箱排，两条的先后不定，偶尔失败；改为同一邮箱的按创建时间排，期望值也排序。
14. **`useWorkspacesFetch` 是新加的 hook**：P8a 的三处（落点、工作区包装层、Task 6 的新手引导）各写一遍键和取数，个人设置是第四处；四处共用一个，键只写一次。个人设置的取数是新的挂载请求（S2 的行，2.12）。命令面板的工作区菜单读的就是这份列表，不另取（P8a review 的 M7 第三处）。
15. **请裁定：测试的共用部分中的一个 `!` 和三个 `as const`**（brief：新的 `as`、`any`、`!` 算发现，除非 spec 裁定）：`fake-tab.ts` 的 `heldChange` 有 `let settle!: …`（确定赋值的断言：Promise 的构造函数同步调用执行器，`settle` 在下一行之前已赋值；M2 的 `core/lib/auth/fake-browser.ts:101` 同样写法）；`in-session.ts` 的两个 `"done" as const`、`"failed" as const`（给联合打上标签），`invitation-link.test.ts` 的 `["/", "/sign-up"] as const`（`it.each` 的路径是字面类型）。都不是类型转换；源文件中 `as` 7 → 0、`any` 2 → 0、非空断言 0 → 0（附录 A.7）。
16. **请裁定：M2 的账户页面不在 P9 的会话核对中**：新手引导的资料一步自己的提示（`steps/profile/root.tsx`）、个人设置的 general、preferences、security 三页在修改兑现之后提示，不核对会话（与 2.13 同一类）。它们是 M2 的页面，修改的是账户，不是工作区一侧；P9 只让资料一步的完成交给根（根的跟进经 `followInSession`）。建议放进 P11 的清理（第 5 节）；改它们会让 P9 多改 M2 的四个页面和它们的测试。
17. **删除工作区之后的文案**（W17）：成功的提示原来说"即将跳转到您的个人资料页面"，Nerve 的根路径落到另一个工作区或 `/create-workspace`（3.14），改写；被拒绝时说 nerve 的原因之后，"请重试"一条没有读者，删除（附录 A.8 的检查发现）。
18. **S2 的行**：general 页和成员页的取数 P9 没有改，它们在 Task 1、2 的树上加入 S2，那时的清单就是 `4b1334a5` 的；新人和个人设置两行在 `4b1334a5` 上没有 `GET /api/v0/workspaces`（P8a review 的 M7），Task 6、10 加上。访客的 `/{slug}/settings` 没有时区：工作区设置的布局不给访客 general 页（9.2）。
19. **页面上看得到的不同**（W17）：
    - 会话：工作区一侧的修改（2.13）在另一个标签页换了账户之后兑现时，页面不跳转、不提示、不改界面（原来照样跳转、提示，显示在那个账户的页面上）；停用同样。
    - 错误：general 页的修改、删除工作区、成员页的三个修改、邀请（弹窗和列表）、邀请页的接受和忽略、创建工作区、新手引导的每一步、导航设置、导览的结束被拒绝时，都说 nerve 的原因（原来读 `err.error`、显示通用的文字，或只在控制台记一条，或是没有处理的拒绝）；被拒绝的修改结束加载。
    - general 页：规模没有时第一次选择不再有 React 的"uncontrolled to controlled"控制台错误。删除工作区的成功提示说去另一个工作区或创建（第 17 条）。
    - 成员页：角色是编号（P22；Plane 的选择框值是 `any`）。邀请弹窗：已是成员、已有邀请（含已忽略的）、格式不对的地址的原因在各自的行下，弹窗不关。邀请列表：已忽略的标"已忽略"，没有角色选择和复制链接；复制失败有提示；链接的查询参数编码。
    - 邀请页：链接缺 id 或令牌时说无效（原来一直转圈）；nerve 没有回答时可以重试；没有登录时给"登录以接受""注册以接受"（原来给"接受"）；接受之后写上次打开的工作区再进入；忽略之后说已忽略（原来缓存的预览仍是未忽略）；邮箱不一致时只给退出登录；被删除或已回答的邀请，接受、忽略被拒绝之后重读预览。文案都经 `t()`。
    - 创建工作区：slug 字段显示的就是发出的；不合格的 slug 在点"创建"之后说明（第 8 条）；创建之后写上次打开的工作区；创建关闭的说法（第 7 条）。
    - 新手引导：列表到之前显示加载、取不到时可以重试；先接受了邀请的人资料一步之后完成（原来列表没有取，他也要创建工作区）；接着做时在它离开的那一步（邀请一步邀请到他建的工作区）；邀请一步是成员页的表单，发出之后列出链接（第 9 条）；完成时不再把列表的第一个工作区写成上次打开的；创建一步的 slug 框红边读 slug 的错误。
    - 注册：凭邀请链接注册时带上邀请，注册关闭时也能注册（原来被拒绝）；登录页和注册页之间的链接保留邀请和回到它的路。
    - 侧边栏：设置到达之前按 nerve 的默认值；对话框的数量离开时发一次（原来每个按键一次）、显示设置的数量（原来是挂载时的）、打开限制用 nerve 最近回答的数量；打开对话框的按钮有名字。
    - 停用弹窗：被拒绝的原因在弹窗里（原来是提示），关闭再打开不在了。
    - 包装层："找不到工作区"的界面和退出失败的提示随语言（原来写死的英文）。个人设置直接打开时侧边栏和命令面板有调用者的工作区。

**交接到 P9 的事项的落点**（brief 的"Carried into P9"）：

| 来源 | 事项 | 落点 |
|---|---|---|
| P8a spec 第 5 节 P9 一行 | 落点、新手引导、邀请页、注册页、首页和侧边栏、工作区设置的页面行为；W2 的页面版本 | Task 1–10（第 2 节）；W2：Task 2 |
| | 创建表单照 nerve 对 slug 的回答决定能否提交的直接检查（`t5-slug-ignored`） | Task 5（`T5.1`，vitest 和 W1 两层） |
| | 删除、离开、创建、接受之后的 `sessionGuard()` 和它们的 vitest | `followInSession`：Task 1（删除）、2（离开）、5（创建）、4（接受）；2.13 |
| | 切换、创建、接受之后写 `last_workspace_id` | 切换：P8a 已写，W2 的页面版本核对（Task 2）；创建：Task 5、6；接受：Task 4 |
| | 第 3 节第 12 条中 P9 的两个 Plane 缺陷 | 已忽略的邀请显示为待接受：Task 3；新手引导邀请一步失败的文案：Task 6（成员页的表单） |
| | M2 收尾第 13 节 | Task 10（文案）；浏览器核对 C4 是控制者的 |
| | 读列表而没有取数的三处 | 新手引导：Task 6；个人设置：Task 10；命令面板：读同一份列表（第 3 节第 14 条） |
| | 被拒绝的列表重取被当作创建失败 | Task 5、6：创建之后不再重取列表：`createWorkspace` 把回答写进已取的列表（P8a），新手引导从 Task 6 起先取列表；`useCreateWorkspace` 只看创建的回答 |
| | slug 变化时列表是旧的 | 不改（设计 7.5："P9 也不加第二种判断"）；第 5 节 M7 |
| | 新手引导的邀请发到刚创建的工作区 | Task 6（`resumedPlace` 和邀请一步的 `workspace`） |
| | 成员页角色是数字的页面检查 | Task 2（W7 的 `{ role: 5 }`）、Task 3（W4） |
| P8a review 第 6 节 P9 一行 | P7 `creation_disabled` 的文案 | Task 6（新手引导）、Task 5（`/create-workspace`） |
| | P9 W1、W3、W10 的页面版本 | W1：Task 5、6；W3：Task 1；W10 只有命令和接口，没有页面 |
| | P12 改名被拒绝只有 `console.error` | Task 1 |
| | P19 "找不到"由调用者的列表决定 | 不改（同上） |
| | P21 读 `err.error` 的提示：`member-columns.tsx`、`members-list-item.tsx` 两处、`invitations-list-item.tsx` 两处、`members/page.tsx`、`onboarding/steps/team/root.tsx` | Task 2（前三处）、Task 3（`invitations-list-item.tsx`、`members/page.tsx`）、Task 6（新手引导） |
| | P22 W7 断言 `PATCH` 的角色是数字 | Task 2 |
| | P27 缺令牌或 id 一直转圈；缓存的预览；未登录的 401；W5、W6 的页面版本 | Task 4（`invitationView`、重读预览、`sign-in`）；W5：Task 4；W6：Task 7 |
| | P28 对话框：拒绝走文案表；失焦或防抖；挂载时捕获的旧上限；设置到达前后的切换 | Task 8（第 3 节第 10、11 条） |
| | M4 新手引导的邀请发往刚建的工作区 | Task 6 |
| | M7 三处读列表；创建之后的重取 | Task 6、10；Task 5、6 |
| P8b spec、review 第 6 节 P9 一行 | E5：`preferencesChangeOf` 在调用时读显示的数量 | Task 8（store 的修改是函数，`T8.1`–`T8.4`） |
| M2 收尾第 1 节 | 页面一侧：邀请页、注册页带着邀请回到邀请页，W5、W6 的页面版本 | Task 4、7：关闭 |
| M2 收尾第 2 节 | 页面一侧：W2 的页面版本、新手引导的两步、README 的说明 | Task 2、6、11：关闭 |
| M2 收尾第 11 节 | 页面一侧：新手引导的创建、邀请两步；工作区 general 页的时区选择 | Task 6；Task 1（时区随 general 页的修改发出，W3 核对；S2 的时区请求）。本节随 P10 关闭 |
| M2 收尾第 13 节 | `WorkspaceAuthWrapper` 的退出按钮、`isLoading`；新手引导邀请一步和导览在中文下的文案；"加入工作区"一步删除 | Task 10（文案经 `t()`）、Task 6（邀请一步的文案）；逐条的浏览器核对是 C4，控制者的。"加入工作区"一步在 P8a 删除 |
| P2 spec 第 5 节 | W8 的页面版本、成员页 | Task 8、2 |
| | 个人主页的页面 | P11（故事 P9 的页面版本和 C10，M1-P2 交接已改） |
| M1-P2 交接 | `ProjectNavigationDialog` 改调新接口（W8 的页面版本） | Task 8 |
| P3 spec、review 第 6 节 | W4–W6 的页面版本 | Task 3、4、7 |
| P4b review 第 6 节 | P26 W3 的页面版本不先删除项目时 `deletedAlone` 要作参数 | Task 1（`furnishWorkspace` 给每张表一行，W3 的页面版本传 `["workspace_member_invites"]`） |
| P6 spec、review 第 6 节 | W9 的页面版本 | Task 9 |
| Codex 设计评审 M-4 | W5 点"接受"得到 403 之后说明 | Task 4（`mismatch`） |
| Codex 设计评审 4.3 第 1 条 | 成员页给成员取邀请 | P8a 的 hook；页面版本：W4（成员的页面）、S2（每个角色的成员页） |
| Codex 设计评审 4.3 第 2 条 | 已发出的修改迟到地成功，旧页面仍跳转、提示 | `followInSession`（Task 1 起）、W3 的会话切换 |
| M1 收尾（7.9） | 有手改的文件 0 条 oxlint 警告；上限调低 | 每个 Task；附录 A.5 |
| 3.20 | README"前端"一节 | Task 11 |

## 4. 验收标准（完成线，M3 设计 12 节 P9）

- [ ] W1–W9 的页面版本、S1、S2 和此前的全部故事通过：worktree 中 `make e2e` 93 个全部通过（`4b1334a5` 是 75 个；W3 的会话切换在其中）。
- [ ] 9.5 中 P9 的 vitest 通过：删除工作区的组件（`delete-workspace-modal.test.tsx`）、新手引导（`onboarding/root.test.tsx`、`onboarding-place.test.ts`）、邀请链接的拼法（`invitation-link.test.ts`）；每个都有一个发现它的变异，P9 写或改的每个测试都至少被一个变异发现（W18，附录 A.2）。
- [ ] 9.6 的变异核对：去掉 `followInSession` 的核对（`T1.1`），W3 的会话切换失败（附录 A.2；写进评审是控制者的）。
- [ ] W4 的成员视角：成员打开成员页，没有失败的请求，也不请求邀请（W4 的页面版本、S2）。
- [ ] `node tools/keywords.mjs` 通过：64 条规则、3 个例外，没有命中（`make lint-web`）；`tsc`、knip 通过。
- [ ] 改到的文件按 7.9 没有 oxlint 警告：web 的 73 个 TS 文件都是 0 条；web 的上限 360 → 359，其余各包不变；没有新的抑制（附录 A.5）。
- [ ] 没有新的 `as`、`any`、`!`，第 3 节第 15 条裁定的除外（附录 A.7）。
- [ ] 逐 Task 复现：从 `4b1334a5` 照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同（附录 A.10）。
- [ ] 控制者的浏览器核对 C1–C5 写进评审（9.7；不在 plan 中）。

## 5. 不在 P9 范围内

- **P10**：项目的页面；`ProjectAuthWrapper` 中已归档项目的界面；删除项目、离开项目之后的会话核对（7.1，它们照 `followInSession` 写）；P8b spec 第 5 节 P10 一行照录。另外 P9 的清扫看到的（附录 A.4）：项目成员页的 `member-list-item.tsx:60`、`:68` 和 `project/settings/member-columns.tsx:152` 仍读 `err.error`（P8b spec 第 5 节已记）；`navigation/use-tab-preferences.ts` 的拒绝进 `console.error`。
- **P11**：M2 的账户页面在修改兑现之后的提示不核对会话（第 3 节第 16 条，请裁定）：新手引导的资料一步（`onboarding/steps/profile/root.tsx`）、个人设置的 general（`settings/profile/content/pages/general/form.tsx`）、preferences（`pages/preferences/language-and-timezone-list.tsx`、`profile/start-of-week-preference.tsx`）、security（`pages/security.tsx`）；个人主页的页面（故事 P9 的页面版本，C10）；P8a、P8b 留给 P11 的照旧（P9 没有只经机械步骤到达的文件，A.8 的清单不变）。
- **M4**：工作项的创建、修改中读 `error?.error` 的两处（`issues/issue-modal/base.tsx:233`、`:323`）；P8a、P8b 交给 M4 的照旧。
- **M6**：迭代、模块的弹窗读 `error?.error`（`cycles/transfer-issues-modal.tsx:74`、`modules/modal.tsx:73`、`:95`）；视图的修改之后的跟进。
- **M7**：已挂载的包装层换 slug 时不重取列表（P19，设计 7.5，改时改 `useWorkspaceFetch` 这一个判断）；收藏的 `favorite-items/common/helper.tsx` 的 `console.error`。
- **M8**：webhooks 的三处读 `error?.error`（`create-webhook-modal.tsx:91`、`form/secret-key.tsx:86`、`webhooks/[webhookId]/page.tsx:74`）。
- **收尾**：P9 改到的页面中仍有写死的英文标题"Success!"（`workspace-details.tsx` 和 `invitations-list-item.tsx` 的成功提示，Plane 原有，P9 没有改它们的文字）；工具的缺口（P8b 的 PF-L5）：两种语言都留下、已没有读者的文案键没有常设的检查，P9 用一次性的 `i18norphans.py` 查过（附录 A.8），收尾决定是否加一个检查。

## 6. 风险

- **S2 的清单是一份要维护的名单**：P10、P11、M4–M7 每在挂载路径上加一个取数，S2 都会失败，直到清单改过（P8b 的裁定 D5，这正是它守的性质）。
- **会话切换的端到端有一个时间窗**：W3 的会话切换在 `route.fulfill()` 之后等 2 秒，断言页面没有跳转、没有提示。跳转与删除的回答在同一串 Promise 的步骤里，比这个窗口短几个数量级；`T1.1` 在每次运行中都让它失败。退路：失败时先看跟进是否被推迟到了计时器之后。
- **`sentTo` 的路由一直留着**（第 3 节第 12 条）：同一个页面上同一路径的第二次 `sentTo` 再加一层路由，各自读第一个请求体；P9 的故事中同一路径的多次修改都用新的 `sentTo`，读到的是各自的第一个（例如 W8 的五个 `PATCH`）。
- **不是 git 仓库的副本里 S3 失败**（P4b spec F4）：复现在副本中运行，只有 S3 因为读不到提交而失败，由复现脚本单独认出；worktree 中 93 个都要通过。

## 7. 已知的限制、交接和关闭条件

- **关闭**：M2 收尾第 1 节的页面一侧、第 2 节的页面一侧（交接的这两节至此关闭）；第 13 节在控制者的 C4 写进评审之后关闭；第 11 节的页面一侧在 P9 完成工作区的部分，随 P10 关闭（13.1）。P8a review 第 6 节 P9 一行、P8b review 第 6 节 P9 一行（E5）；P2、P3、P4b（P26）、P6 交给 P9 的行；M1-P2 交接的侧边栏偏好；Codex M-4、4.3 第 2 条的页面一侧。
- **交接**：第 5 节各行照录进 P10、P11 的 spec 和 M4–M8 的交接（13.2）；第 3 节中裁定的结果写进 P9 的评审。
- **放不下的关闭条件**：没有。设计 12 节 P9 的"关闭"三节都有落点；C1–C5 和评审按 brief 是控制者的。
- **限制**：每次挂载都重取，聚焦时不取（P8a 的 F-1）；已挂载的工作区包装层换 slug 时不重取列表（P19）。

## 附录 A：原型验证记录（2026-10-09）

### A.1 方法与门禁

- 原型 `$M3TMP/p9proto` 是 `4b1334a5` 的 `git archive` 副本（`pnpm install --frozen-lockfile` 之后）。每个 Task 由一个脚本从上一个快照写出（`$M3TMP/p9tools/edit/T<n>.py`），照仓库的格式化（oxfmt）排版，`tsc` 通过之后存一份快照（`$M3TMP/p9snap/T1`…`T11`）。plan 的块由快照之间的差异写出（`mkblocks.py`：11 个 Task，0 个问题；它把每个 Task 的块依次应用到上一个快照，与那个 Task 的快照逐字节比较）。代码或测试改过时，从改动的那个 Task 起重新写出之后的每个快照（`chain.sh`），再重新写出块。
- 原型上每个 Task 写出之后 `make lint-web`、`make knip`、`make test-web` 都跑过，有页面或挂载路径改动的 Task 都跑了 `make e2e`；快照重新写出之后，最后一次逐 Task 复现在每个 Task 的树上再跑四个门禁（A.10：它的每个 Task 的树与快照相同，块由 `mkblocks.py` 核对过）。`4b1334a5` 上：`make lint-web`、`make knip`、`make test-web` 通过，`make e2e` 75 个中 74 个通过（S3 在副本中失败，F4）。
- 最终原型上：`make lint-web` 通过（关键词守卫 64 条规则、3 个例外、没有命中）；`make knip` 通过；`make test-web` 通过（web 应用 70 个测试文件、617 个测试；`4b1334a5` 上是 56 个、504 个）；`make e2e` 93 个中 92 个通过（S3，F4）。没有 Go、接口描述和生成的文件改动，`make lint-go`、`make test`、`make gen-check` 不需要（`git diff --stat 4b1334a5 -- server/ api/ pnpm-lock.yaml` 没有输出）。
- 端到端的反复：W1 的新手引导版本 3 次、W8 6 次、W9 和 A12 15 次，都通过（`sentTo` 改为不移除路由之后，第 3 节第 12 条）。

| Task | 改到的文件 | web 的测试文件 / 测试 | 端到端 |
|---|---|---|---|
| 开始 | | 56 / 504 | 75 |
| 1 | 18 | 58 / 519 | 77 |
| 2 | 13 | 59 / 531 | 79 |
| 3 | 17 | 62 / 548 | 80 |
| 4 | 10 | 64 / 564 | 82 |
| 5 | 7 | 65 / 580 | 83 |
| 6 | 24 | 67 / 596 | 88 |
| 7 | 4 | 67 / 596 | 90 |
| 8 | 14 | 69 / 612 | 91 |
| 9 | 6 | 70 / 617 | 92 |
| 10 | 10 | 70 / 617 | 93 |
| 11 | 2 | 70 / 617 | 93 |

（测试数按 vitest 自己的计数，`it.each` 的每一行算一个；一个文件可以由几个 Task 改到，所以改到的文件之和大于 101。）

### A.2 变异

`mutants_p9.py`（Task 1、2）和 `mutants_T3.py`…`mutants_T10.py`：76 个变异，每个只改一处或几处，`mut.py` 在最终原型上、在它写的每个检查上各跑一次：76 个都被发现，没有存活的。每个变异另在它自己的 Task 的快照上跑过（`mut-T<n>.json`；最后一次跑变异之后，快照只在两种语言的 `workspace-settings.json` 上改过，第 3 节第 17 条，没有变异改它们或由它们决定），所用的层与最终原型上相同（`finalvstask.py`：0 处不同），所以 plan 的变异表中没有"（之后的 Task 起）"的标记；每个变异要改的文字在它的 Task 的快照上和最终原型上都恰好出现一次（`mutcheck.py`：0 个不适用；`T6.1` 要改的文字在 Task 10 改了，变异给出两个版本的文字）。"静态"是 `make lint-web` 的 `tsc`、oxlint（上限）和 `make knip`。按缺陷类别（一个变异可以被几层发现）：

| 类别 | 变异 | 静态 | vitest | 端到端 | 存活 |
|---|---|---|---|---|---|
| W17 页面的行为照故事 | 27 | 2 | 13 | 22 | |
| 错误的读法（P8b 的第二类） | 15 | | 12 | 9 | |
| 封闭的请求体（P8b 的第一类） | 13 | 2 | 9 | 8 | |
| W3 页面的跟进只在发出修改的会话里 | 11 | 1 | 11 | 2 | |
| W2 挂载时的取数和落点 | 6 | | 3 | 5 | |
| 请求体取自所持的值（P8b 的第三类，E5） | 4 | | 4 | | |

按发现它的层的组合：只有 vitest 28 个、vitest 和端到端 23 个、只有端到端 20 个、静态和端到端 3 个、只有静态 1 个（`T7.2`：登录也带邀请，`tsc`）、静态和 vitest 1 个（`T9.3`）。按 Task：Task 1 9 个、2 8 个、3 8 个、4 9 个、5 9 个、6 10 个、7 3 个、8 13 个、9 4 个、10 3 个；Task 11 只改文档，没有变异。

- **W3 的 11 个**：`T1.1`（`followInSession` 不论会话都跟进，三个 vitest 文件和 W3 的会话切换）；离开、邀请、接受、创建、新手引导、对话框、导览各自不经核对（`T2.3`、`T3.3`、`T4.1`、`T5.2`、`T6.6`、`T8.7`、`T8.13`）；删除之后不回到根路径（`T1.8`）；停用的两个（`T9.3`：弹窗不论 store 的回答都说已停用，oxlint 的上限和弹窗的 vitest；`T9.4`：store 不论 `endSession` 的回答都说结束了，store 的 vitest）。
- **只有端到端的 20 个**：都是页面上的行为，P9 的页面在 vitest 中没有渲染它的那一部分（例如成员行的菜单、邀请列表的"已忽略"、对话框的草稿、注册页之间的链接、W9 取消之后的弹窗）；每个都由它的故事的页面版本发现。`T2.8`（成员页对每个角色都取邀请）由 S2 发现：成员页的取数条件的 vitest 在 P8a（`use-members-settings-fetch.test.ts`），P9 没有改它，S2 的成员、访客两行是页面一层的核对。
- **W18（每个新的 vitest 都能失败）**：`mut.py` 只跑一个变异写的检查，所以另由 `w18run.py` 把每个变异对 P9 写或改的全部 19 个 vitest 文件各跑一次，`w18p9.py` 把失败的测试对照最终的 vitest 报告：P9 新加或改写的 118 个测试（`it.each` 的每一行算一个）中，76 个变异让 71 个失败。其余 47 个是正向的行（例如"在会话里跟进"、"移出之后停在原页"、决定表中每一步的一行），另写 40 个探查变异（`mutants_w18_p9.py`，`W18.1`–`W18.40`，只在最终原型上跑，不在 plan 的表中）：成功的修改之后在核对之外提示（改角色、移出、完成引导、导航设置、导览的结束，W3）；邀请链接指向 Plane 的 `/invitations`；邀请被拒绝时的四种错法；不核对地址的格式；nerve 不保存时停在原页；`invitationView` 每一步的错法（6 个）；`creationRefusal` 和创建的五种错法；`resumedPlace`、`afterProfile` 的六种错法；资料一步不写进资料；停用弹窗、store 的回答反过来；`followInSession` 不跟进成功、核对模块载入时的会话；数量框和 `preferencesChangeOf` 的六种错法。40 个都被发现，118 个测试都至少失败一次。改到的五个 P8 的测试文件中 P8 原有的 23 个测试不算（其中 2 个也被 P9 的变异发现）。
- **例外**（只由评审或之后的 Task 才能发现的性质）：没有。

### A.3 W2：挂载清单（真实运行）

S2 的清单在 Task 1、2、6、10 的树上加入，每次 `make e2e` 都核对（2.12 的表；S2 从 7 个测试到 9 个：新人、个人设置各一个，general 页和成员页的行在每个账户已有的测试中）。之前与之后：

- `4b1334a5` 的 S2：四个账户各打开 `/`（`APP` + `WORKSPACE`）、项目设置；管理员另打开项目页（`ARCHIVED`）。
- P9 加的行：`/{slug}/settings`，管理员、成员、不是项目成员的成员是 `APP` + `WORKSPACE` + `GENERAL`，访客是 `APP` + `WORKSPACE`；`/{slug}/settings/members`，管理员是 `APP` + `WORKSPACE` + `INVITATIONS`，其余三个是 `APP` + `WORKSPACE`。这两页的取数 P9 没有改（第 3 节第 18 条），在 `4b1334a5` 上相同。
- 新人打开 `/`：`4b1334a5` 上落点不为新手引导取列表，新手引导读一份没人取过的列表（P8a review 的 M7）；Task 6 起是 `APP`（列表一次：新手引导和落点同一个键）。
- 直接打开个人设置：`4b1334a5` 上没有 `GET /api/v0/workspaces`，侧边栏没有工作区；Task 10 起是 `APP`，侧边栏列出 Acme（S2 断言）。
- 每张清单没有 M6、M7 的地址，没有 `/api/v0` 之外的请求，没有失败的请求。故事的页面版本（W1–W9）都经 `watchPage`：每个故事断言记下的失败的请求恰好是它要的那几个（没有，或者例如 W3 的 403、W8 的 422、W9 的 409），没有旧地址的请求、没有页面错误，控制台只有这些失败的报告（`expectQuietConsole`）。

### A.4 缺陷类的清扫

- **封闭的请求体**（P8b 的第一类）：P9 的页面发出的每个请求体都由表单编辑的字段构成，类型是生成的 `…Create`、`…Update`，角色是数字，每个由一个测试钉住：general 页的 `{ name, organization_size?, timezone }`（`workspace-details.test.tsx`、W3 的 `sentTo`）；改角色 `{ role }`（`use-membership-changes.test.ts`、W7）；批量邀请 `{ invitations: [{ email, role }] }`（`invite-modal.test.tsx`、W4）；邀请的改角色 `{ role }`（W4）；接受、忽略（W5）；创建工作区（`use-create-workspace.test.ts`、W1）；写上次打开的工作区（`use-open-workspace.test.ts`、`onboarding/root.test.tsx`、W1、W2、W5）；新手引导的步骤和完成（`root.test.tsx`、W1）；导航设置（`navigation-preferences.test.ts`、W8）；导览的结束（`home/root.test.tsx`、W8）；注册（W6）。13 个变异，都被发现（A.2）。
- **错误的读法**（第二类）：P9 改到的页面的每一条出错的路径都经 `errorMessageKey`（提示、弹窗）或 `FIELD_ERROR_MESSAGES`（字段、行下），没有吞进 `console.error` 的拒绝，被拒绝的修改结束加载。P9 删掉的 `console.error`：`invitations-list-item.tsx`（复制）、`workspace-details.tsx`、`home/root.tsx`、邀请页。仍读 `err.error` 的都不在 P9 的页面中：项目成员页（P10）、webhooks（M8）、迭代和模块（M6）、工作项（M4）（第 5 节）。15 个变异。
- **请求体取自所持的值**（第三类）：P9 的页面中只有导航设置是"按显示的值算请求体"的形状（E5），Task 8 把它交给 store 在轮到它时算；general 页、角色、邀请发的都是这次编辑的值，不取 store 所持的；对话框的数量是草稿。4 个变异。
- **会话外的跟进**（W3）：`followsweep.py` 列出 web 应用中提示或跳转的每个文件和它们等待的 store 调用，逐个读过：工作区一侧的都在 2.13 的表中；M2 的账户页面在第 3 节第 16 条；退出登录失败的提示五处（工作区菜单、包装层等）不是这一类（失败时会话没有变）。
- **19（每个新的 store 或 hook 方法有失败测试）**：`followInSession`、`useMembershipChanges` 的三个修改、`useCreateWorkspace`、`useOpenWorkspace`、`useProjectNavigationPreferences`、新手引导的根、首页、停用弹窗、`updatePreferences` 都有被拒绝的测试；`invitationView`、`creationRefusal`、`invitationRefusal` 的表中各有没有回答的失败一行。
- **36（另一条路径到达同一结果）**：同一个修改的两个入口共用一个实现：创建工作区（`/create-workspace` 和新手引导，`useCreateWorkspace`）、邀请表单（成员页的弹窗和新手引导，`useWorkspaceInvitationActions`）、复制邀请链接（邀请列表和新手引导，`useCopyInvitationLink`）、工作区列表的取数（四处，`useWorkspacesFetch`）、打开刚加入的工作区（接受、创建，`useOpenWorkspace`）。新手引导的邀请一步和成员页由同一组测试守着（`invite-modal.test.tsx` 测共用的 hook；W1、W4 两个故事各走一个入口）。
- **45（没有不能失败的断言）**：W18（A.2）：118 个测试都至少被一个变异发现；"换账户之后什么都不做"的 `is answered` 行由"在核对之外跟进"的探查变异发现。
- **46（每个等待有期限）**：vitest 中等待页面跟进的是 `pageSettled()`（一个计时器的轮次）和 `vi.waitFor`（vitest 的期限）；端到端的等待是 Playwright 的 `expect`、`expect.poll` 和 `waitForResponse`，都有期限。
- **50（说明与代码一致）**：每个 Task 结束时，新写或改动的注释、JSDoc 与它描述的代码或测试逐句核对；最终原型上 P9 加或改的注释（`addedcomments.py`，约 1,000 行）再全部读过一遍；README 和前端改动清单的行与最终的代码核对（Task 11）。

### A.5 oxlint（7.9，R3）

- 有手改的 TS 文件：web 应用 71 个（含测试）、`@nerve/types` 的 `navigation-preferences.ts`、`@nerve/utils` 的 `validation.ts`，都是 0 条警告。P9 没有机械步骤，没有只经机械步骤到达的文件。`4b1334a5` 上这些文件共有 1 条（`use-workspace-invitation.tsx`，Task 3 清零；各包的上限要求警告数等于上限，web 的只降了 1，其余不变）。
- 上限：web 应用 360 → 359（Task 3）；`utils` 7、`types` 0、`constants` 1、`propel` 16、`ui` 19 不变。
- 抑制：没有新的抑制；删去两处（Task 6：新手引导的根原来以 `useEffect` 决定第一步，抑制 `react-hooks/exhaustive-deps`；Task 8：对话框的 `unicorn/consistent-function-scoping`），188 → 186。
- 关键词守卫：64 条规则、3 个例外，每个 Task 都没有命中；P9 没有删掉旧地址（P8a、P8b 已删完），不改 `tools/keywords.json`。

### A.6 规模

- **plan** 共 8,520 行：Task 1 之前（约束、文件结构）111 行；各 Task 一节：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 会话里的跟进；general 页、删除；W3 | 1,244 | 1,131 | 18 |
| 2 成员页；W7、W2 | 927 | 833 | 13 |
| 3 邀请的表单和列表；W4 | 1,174 | 1,064 | 17 |
| 4 邀请页；W5 | 775 | 688 | 10 |
| 5 创建工作区；W1（一） | 764 | 683 | 7 |
| 6 新手引导；W1（二） | 1,475 | 1,340 | 24 |
| 7 带邀请的注册；W6 | 266 | 209 | 4 |
| 8 导航设置、对话框、导览；W8 | 1,006 | 901 | 14 |
| 9 停用弹窗；W9 | 385 | 317 | 6 |
| 10 列表的取数、包装层的界面 | 312 | 233 | 10 |
| 11 文档 | 81 | 42 | 2 |

- **文件的行数**（约 400 行）：P9 改到的文件中最长的是 `e2e/stories/workspace/w3-workspace-settings.spec.ts` 395 行、`e2e/fixtures/assert/workspace.ts` 390 行、S2 375 行（三个都是原有的故事和 fixture，P9 加了页面版本和行）；新文件中最长的是 `use-create-workspace.test.ts` 165 行、`e2e/fixtures/workspace-pages.ts` 162 行。有手改的文件没有一个结束时超过 400 行。
- **变长的 Plane 文件**（brief：只为使用方改到的不变长）：16 个变长，15 个是本 Phase 的对象（行为改在它们里面：邀请页 95 → 179、删除弹窗 30 → 179（表单并入）、`auth-root.tsx` 35 → 39、`password.tsx` 284 → 287、停用弹窗 93 → 102、`auth-screens/header.tsx` 81 → 83、`home/root.tsx` 47 → 53、`onboarding/steps/root.tsx` 57 → 62、`workspace-options.tsx` 49 → 53、`sidebar-wrapper.tsx` 85 → 88、`invite-modal/fields.tsx` 120 → 122、`members/invite-modal.tsx` 57 → 67、`workspace-details.tsx` 251 → 262、`use-workspace-invitation.tsx` 91 → 101、`store/user/index.ts` 199 → 200）；只为使用方改到的是 `members-list.tsx` 89 → 91（第 3 节第 6 条）。其余有手改的 Plane 文件变短或不变（例如新手引导的邀请一步 396 → 80、`create-workspace-form.tsx` 236 → 193、`validation.ts` 120 → 83）。

### A.7 W12：类型断言

- `w12.cjs`（TypeScript 的语法树，`4b1334a5` 对最终原型）：P9 改到的 56 个源文件（测试除外）中，`as`（不含 `as const`）7 → 0、`any` 2 → 0、非空断言 0 → 0。去掉的：`member-columns.tsx` 2 个、`members-list-item.tsx` 2 个、`invitations-list-item.tsx` 2 个、成员页 1 个（`as`）；新手引导的邀请一步 2 个 `any`。
- 新的 `as const` 三个、确定赋值的断言一个（第 3 节第 15 条，请裁定）：`in-session.ts` 的 `"done" as const`、`"failed" as const`；`invitation-link.test.ts` 的 `["/", "/sign-up"] as const`；`fake-tab.ts` 的 `let settle!: …`。`casts.py` 逐个 Task 核对：没有别的新的 `as`、`any`、`!` 或 `oxlint-disable`。
- 没有手写的、重述契约的类型：表单的值是 `WorkspaceInvitationsCreate`、`CreationForm`（`Pick<WorkspaceCreate, …>` 加可空的规模）、`Pick<Workspace, …>`；knip 没有未使用的导出。

### A.8 文案

两种语言相同，从快照逐个量出（`i18nadded.py`，`en`；`check:sync` 核对 `zh-CN` 的键相同）：加 26 条、删 8 条、改写 5 条。

| Task | 文件 | 加 | 删 | 改写 |
|---|---|---|---|---|
| 1 | `workspace-settings.json` | | `delete_modal.error_message` | `delete_modal.success_message` |
| 3 | `workspace-settings.json` | `members.declined`、`members.modal.errors.already_member`、`already_invited` | | |
| 4 | `workspace.json` | `workspace_invitation.*` 14 条 | | |
| 5 | `workspace.json` | `workspace_creation.errors.validation.url_reserved` | `errors.creation_disabled.request_button`、`request_email.subject`、`request_email.body` | `errors.creation_disabled.title`、`description`，`errors.validation.url_alphanumeric` |
| 6 | `common.json`、`workspace.json` | `onboarding.invite.links.title`、`description` | `onboarding.invite.role`、`onboarding.invite.not_an_email`、`workspace_creation.toast.error.title`、`message` | `onboarding.workspace.creation_disabled` |
| 8 | `accessibility.json` | `aria_labels.projects_sidebar.project_navigation` | | |
| 10 | `workspace.json` | `workspace_not_found.*` 5 条 | | |

- **删除是完整的**（P8b 的 PF-L5：`check:sync` 看不见两种语言都留下的键）：`i18norphans.py` 列出 `4b1334a5` 上有源文件写出全名、最终原型上没有任何源文件写出、文案文件中仍在的键：只有 `delete_modal.error_message` 一条，Task 1 删除它（第 3 节第 17 条）。之后再查：0 条。只经模板拼出键名的读者两边都看不见，这一类 P9 没有改。
- 改写的文案都与 nerve 一致：slug 的格式是 nerve 的规则（小写字母、数字、`-`、`_`）；创建关闭说请工作区的管理员给邀请链接（两处相同）；删除工作区之后去另一个工作区或创建。

### A.9 Phase 的大小

写 plan 之前的核对（brief）：设计的 14 个任务中，12、13（端到端）并进各页面的 Task（测试与代码同一个提交），14 只改文档，估出约 10 个 Task、约 9,000–11,000 行，最长的约 1,100 行：在约 16 个、约 1,500 行之内，不拆。原型中实测：11 个 Task（停用弹窗单独成为 Task 9，它的端到端和 store 的改动比估计的多），8,520 行，最长的 Task 6 是 1,475 行（新手引导的根、位置、三个步骤和四个端到端），在约 1,500 行之内；其余都在 1,244 行之内。

### A.10 逐 Task 复现

`replay.py` 在 `4b1334a5` 的 `git archive` 副本（`$M3TMP/p9replay`，`pnpm install --frozen-lockfile` 之后）上照 plan 逐个 Task 执行：写入块（`planapply.mjs`），核对 Go 的模块文件不变、`pnpm-lock.yaml` 与那个 Task 的快照相同，再执行这个 Task 的每个 `Run:`。副本不是 git 仓库，所以 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（它列文件不用 git，规则和样例相同），`make e2e` 只有 S3 因为读不到提交而失败（F4），由脚本单独认出。

| Task | 写入的文件 | 检查 |
|---|---|---|
| 1 | 18 | `make lint-web`、`make knip`、`make test-web` 通过；`make e2e` 76 个通过（S3 除外） |
| 2–5 | 13、17、10、7 | 同 Task 1，`make e2e` 78、79、81、82 个通过 |
| 6–10 | 24、4、14、6、10 | 同 Task 1，`make e2e` 87、89、90、91、92 个通过 |
| 11 | 2 | 同 Task 10 |

这一次复现用的 plan 就是本 spec 同一个提交中的 plan（8,520 行，SHA-256 `99775dc2d9a3239e…`），从 `4b1334a5` 的一个新副本从头复现，11 个 Task 都通过（`$M3TMP/p9tools/replay-logs/summary.txt`）。最后的副本与原型逐文件相同：`treediff.mjs p9replay p9snap/T11`：3,166 个文件，0 处差异。此前四次复现（plan 在其间改过：停用弹窗的成功一支、W1 的"以后再说"、`/create-workspace` 创建关闭的文案、删除工作区的文案）各自通过到它们被停下或结束的 Task，不作为证据。
