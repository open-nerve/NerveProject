# M3/P9 工作区的页面：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P9 `web-workspace-pages` |
| 日期 | 2026-10-09 |
| 状态 | 已裁定：第 3 节的八条由控制者裁定（P1–P8，2026-10-09，都接受）；预检（0 高、4 中、10 低）之后修订一次，各条的落点在第 3 节的"预检之后"；控制者的浏览器核对 C1–C5 和评审在 plan 的 Task 之后（M3 设计 9.7） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W1–W9、S2）、3.8、3.10、3.11、3.14、3.18、7.1、7.4、7.5、9.5、9.6、9.7、12（P9）、13.1 节；[总体设计](../../v0-design.md) 7.7 |
| 前置交接 | [P8a spec](P8a-web-workspace-data.md) 第 3 节第 12 条、第 5 节 P9 一行；[P8a review](../reviews/P8a-web-workspace-data-review.md) 第 6 节 P9 一行；[P8b spec](P8b-web-project-data.md) 第 5 节 P9 一行；[P8b review](../reviews/P8b-web-project-data-review.md) 第 6 节 P9 一行（E5）；[M2 收尾交接](../handoffs/M2-closeout.md) 第 1、2、11、13 节；P2、P3、P6 的 spec 第 5 节，P3、P4b、P6 的 review 第 6 节中 P9 的行；[M1-P2 交接](../handoffs/M1-P2-trim-content.md)；[Codex 设计评审](../reviews/M3-design-codex-adversarial-review.md) M-4、4.3 |
| 计划 | [P9 plan](../plans/P9-web-workspace-pages.md) |

本 spec 只写 M3 设计交给 P9 决定的东西：名字、签名、文件的位置、测试名，以及原型证明了什么。规则本身以 M3 设计和总体设计 7.7 为准，这里引用节号，不重述。P9 依赖 P8a、P8b（`4b1334a5` 的 `main`），不改契约、Go 代码和生成的文件。

P9 的每一处页面改动都按安全测试看待（brief）：一个角色不能发的请求在页面上是 403；换账户之后才落地的写以错误的身份写入（W3）；页面不能显示角色读不到的东西。所以附录 A 的变异逐个核对：页面发出修改之后的跳转、提示和界面变化只在发出它的会话里进行（2.13 的每一处由它自己的 vitest 守着，W3 的会话切换）；每一页挂载时只取它的角色能读的（成员页的邀请只为管理员取，S2 的清单，2.12）；每个请求体只有表单编辑的字段，类型来自生成的客户端，角色是数字；每个错误经 `errorMessageKey` 显示；取决于 store 所持的值的修改在轮到它时算（E5）。只有评审才能发现的会话、权限或取数的性质算缺口：没有（附录 A.2）。

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

**文件总览**：plan 的"文件结构"一节逐个列出 Task 1–11 改到的 110 个文件和每个文件所属的 Task：新文件 33 个（web 的 32 个：13 个模块、hook 和组件，2 个测试的共用部分，17 个测试文件；e2e 的 `fixtures/workspace-pages.ts`），删除 2 个（`delete-workspace-form.tsx`、`onboarding/steps/workspace/root.tsx`），修改 75 个。web 的 TS 文件 79 个（其中测试 22 个）都是手改，没有机械步骤；web 的其余 9 个是 `web/apps/web/package.json`（oxlint 的上限）和四个文案文件的两种语言；e2e 15 个（fixture 4 个、故事 11 个）；静态检查 2 个（根目录 `.oxlintrc.json`、`tools/keywords.json`，2.14）；文档 3 个（`README.md`、`docs/v0/frontend-changes.md`、M2 收尾交接）。契约、Go 代码、生成的文件和 `pnpm-lock.yaml` 都不变。

评审时（`dba0d756`）是 114 个：各 Task 的修正轮另改到四个 plan 没有写到的文件，`e2e/fixtures/auth.ts`（`removeRecord`，Task 4）、`core/components/ui/empty-space.tsx`（Task 4）、`core/store/root.store.test.ts` 和 `core/store/user/profile.store.test.ts`（Task 6）；web 的 TS 文件 81 个（其中测试 24 个），e2e 16 个。修复轮（裁定 F-1–F-8，2.15）又改到 40 个此前没有改到的文件：新文件 4 个（`core/hooks/use-refusal-toast.ts`、`core/hooks/use-sign-out.ts` 和它们的测试）；M2、P8b 的页面和测试 27 个（两个 hook 取代的提示和退出登录；P8b 的四个手写的 `ApiError` 改用 `refusal()`，测试 4 个）；P9 的弹窗 2 个（`confirm-workspace-member-remove.tsx`、`invite-modal/actions.tsx`）和 `authentication-wrapper.tsx` 的说明；e2e 5 个（`fixtures/browser.ts`，故事 A2、A6、A9、A11）；文档 1 个（M1-P2 交接）。复审之后的补充（裁定 F-11）又加一个新文件 `e2e/fixtures/deferred.ts`。合计 155 个：web 的 TS 文件 115 个（其中测试 30 个），e2e 22 个。

**依赖**：不加 npm 包或 Go 模块。

**共用的新部分**：
- `core/lib/in-session.ts` 加 `followInSession<T>(change, followers): Promise<void>`，`ChangeFollowers<T> = { done?: Follower<T>; failed: Follower<unknown> }`，跟进收修改的回答或拒绝，交回 `void` 或 `Promise<void>`（`sessionGuard` 旁边，Task 1；`done` 从 Task 2 起可省；可交回 `Promise` 是修复轮的，裁定 F-2）：发出修改时取标签页的会话，修改兑现之后只在标签页仍在那个会话里时调 `done` 或 `failed`，另一个标签页把它换到别的账户之后两者都不调；修改和跟进交回的 `Promise` 都兑现之后它才兑现，所以等它的按钮忙到跟进做完（打开工作区、结束新手引导）；自己从不拒绝：跟进本身出错时交给 `reportError`（控制台和窗口的 `error` 事件），不交给调用方。页面发出工作区一侧的修改之后的跳转、提示和界面变化都经它（2.13）。
- 测试的共用部分：`core/lib/auth/fake-tab.ts`（`api-client.ts` 的替身：`tokenManager`、`signedIn`、`switchAccount`、`heldChange`（M2 的 `fake-browser.ts` 的 `gate()`，不另写一份）、`lateSettlings`、`pageSettled`，修复轮加 `settledYet`（一个 `Promise` 是否已兑现）；测试决定一个修改何时兑现，在换账户之前或之后），`core/lib/fake-refusal.ts`（`refusal(status, code, fields?)`：nerve 的拒绝，`ApiError`）。组件的测试照 P8b 的写法：服务端渲染（`renderToStaticMarkup`），`@nerve/ui` 的控件换成 P8b 的 `fake-controls.ts`，提示、文案用 `fake-toast.ts`、`fake-i18n.ts`。
- 页面的决定是纯函数，返回 `kind` 的联合，用 `it.each` 测：`invitation-view.ts`（Task 4）、`onboarding-place.ts`（Task 6）、`invite-modal/refusal.ts`（Task 3）、`use-create-workspace.ts` 的 `creationRefusal`（Task 5）。
- e2e：`e2e/fixtures/settings-pages.ts` 的 `sentTo(page, method, path, act)`：经路由在请求离开页面时读它的请求体（第 3 节第 12 条），读请求体的路由是它下面的 `bodiesSentTo(page, method, path)`（Task 8 的修正轮）；修复轮加 `sentHeld`（扣住回答的 `sentTo`）、`holdScripts`（扣住页面要的脚本，导航随之等着）、`enabledWithin`、`closedByEscape`（有期限的"一直不能用""关不掉"；补充（裁定 F-11）起它只认模态的对话框，提示也是 `dialog` 但不是模态的，只把一秒用完算作没关，别的失败照常抛出；裁定 F-12 起 `enabledWithin` 同样：先要求定位到的恰是一个按钮（`toHaveRole("button")`，定位到零个、几个或别的元素都失败），再按 `getByRole` 的 `disabled: false` 等它可用，只把一秒用完算作一直不能用），`e2e/fixtures/auth.ts` 的 `anotherTabSignsIn(context, api, email)`（另一个标签页以另一个账户登录，W1、W3、W5、A6、A9、A11 共用）和 `e2e/fixtures/browser.ts` 的 `refuseClipboardWrites(page)`；补充（裁定 F-11）加新文件 `e2e/fixtures/deferred.ts` 的 `deferred<T>()`（测试决定何时兑现的 `Promise` 和兑现它的函数；e2e 的 lib 是 ES2023，没有 `Promise.withResolvers`），`holdAnswer`、`holdScripts` 和 A9、A12、W1 共用，取代六处各写一份的 `let resolve!:`；新文件 `e2e/fixtures/workspace-pages.ts`：工作区页面上一个人的操作（另一个浏览器、删除、重新登录、切换工作区、成员行、改角色、结束成员关系、邀请链接、邀请行、发出邀请、删除邀请）。

### 2.1 页面在会话里跟进修改；general 页和删除工作区；W3 的页面版本（Task 1；7.1、7.5、9.5、9.6；P8a 的 P12、P4b 的 P26）

- `DeleteWorkspaceModal`（整个文件，收 `workspace: Workspace`）：表单并入弹窗（`delete-workspace-form.tsx` 删除），按提交的值核对名称和"delete my workspace"；`deleteWorkspace(workspace)` 经 `followInSession`：成功时关闭、回到 `/`（落点按剩下的工作区决定，3.14）并提示，失败时按 `code` 提示并停在弹窗。删除在路上时弹窗关不掉：取消按钮禁用，Escape 和背景不关（修复轮，裁定 F-1；W3 核对）。`delete-workspace-section.tsx` 传地址的工作区。文案：被拒绝时说 nerve 的原因，`delete_modal.error_message`（"Try again, please."）没有了读者，两种语言都删除；`delete_modal.success_message` 原来说"即将跳转到个人资料页"，改为去另一个工作区或创建工作区（附录 A.8）。
- `WorkspaceDetails`（general 页）：地址的工作区经 `useParams` 和 `getWorkspaceBySlug`（P8b 的 P13：页面只按路由读地址）；修改只发表单编辑的 `name`、`organization_size`（没有时不发）、`timezone`（生成的 `WorkspaceUpdate`）；规模没有时表单的值是 `null`，选择框一开始就受控；结果经 `followInSession`，拒绝按 `code` 提示，按钮结束加载（P12）。复制地址时浏览器不让写剪贴板，提示失败（原来 `.catch` 什么都不说；与邀请链接的复制同一句，M2 交接第 14 节；预检 L7；修复轮起 W4 的页面版本在浏览器拒绝写剪贴板时核对两处复制的提示，`refuseClipboardWrites`）。
- 测试：`in-session.test.ts` 改用 `fake-tab.ts`，加 `followInSession` 的三个（在会话里跟进回答和拒绝；换账户之后兑现的两种都不跟进；换账户之后才发出的修改照常跟进）；修复轮再加四个（跟进交回的 `Promise` 兑现之后才兑现，回答、拒绝各一个；跟进出错时交给 `reportError`、自己不拒绝，两个）。`delete-workspace-modal.test.tsx` 6 个（9.5：删除工作区的组件，修改在 `loginId` 改变之后才兑现时不跳转、不提示）、`workspace-details.test.tsx` 5 个。
- 端到端：W3 的两个页面版本（修改、成员的只读视图、被降级的管理员被拒绝并看到原因、删除之后落到另一个工作区；会话切换：`holdAnswer` 先 `route.fetch()` 把删除发到 nerve，另一个标签页以另一个账户登录之后才 `route.fulfill()`，原标签页不跳转、不提示）。`e2e/fixtures/api.ts` 的 `furnishWorkspace` 给工作区每张删除会写的表一行（W2 的 API 版本原来自己写的一段移到这里，W3 的页面版本也用它）；W3 的页面版本不先删除项目，删除之后的断言传 `["workspace_member_invites"]`（P26：成员接受时那条邀请已先被删除）。`changeRole`（经 API 改成员的角色）给 W3 的被降级一段，W9 也用它。S2 加每个角色的 `/{slug}/settings`（2.12）。

### 2.2 成员页的三个修改；W7、W2 的页面版本（Task 2；W7、W2，3.14，7.1，7.5；P8a 的 P21、P22）

- `useMembershipChanges(workspaceSlug)`（`core/components/workspace/settings/use-membership-changes.ts`）：`changeRole(userId, role: WorkspaceRole)` 发 `{ role }`（生成的 `WorkspaceMemberUpdate`，编号）、`remove(userId)`、`leave()`（成功之后回到 `/`），都经 `followInSession`，拒绝按 `code` 提示。原来三处各自的 `try/catch` 读 `err.error`（P21），换账户之后仍跳转。
- `workspace-roles.ts` 的 `WORKSPACE_ROLES: WorkspaceRole[] = [5, 15, 20]`：成员页的角色列、Task 3 的邀请行和邀请列表共用。`member-columns.tsx` 的 `AccountTypeColumn` 去掉从没有提交过的 `useForm`/`Controller` 和 `value as EUserPermissions`；`members-list-item.tsx`（整个文件）收地址的 `workspaceSlug`（`members-list.tsx` 传入，不读 `currentWorkspace`），自己一行是"离开"，别人的是"移出"；修复轮起它把 `workspaceSlug` 交给 `useMemberColumns(workspaceSlug)`，列不再自己读 `useParams`，一行只读一次地址（裁定 F-5 的 P3）。
- 测试：`use-membership-changes.test.ts` 12 个（角色是编号、发到成员关系的工作区；移出之后停在原页；离开之后落点；三种修改被拒绝的 `it.each`；三种修改各两种换账户之后兑现）。
- 端到端：W7 的页面版本（`PATCH` 的请求体 `{ role: 5 }`，P22；唯一的管理员离开被拒绝，页面说明原因；成员看得到角色、改不了）；W2 的页面版本（落点、经工作区菜单切换写的 `last_workspace_id`、删除和离开之后的落点，没有失败的请求）。S2 加每个角色的 `/{slug}/settings/members`：邀请只为管理员取（2.12）。

### 2.3 邀请的表单和列表；W4 的页面版本（Task 3；W4，7.1，7.4，7.5；P8a 的 P21、P22，P8a spec 第 3 节第 12 条）

- `use-workspace-invitation.tsx`（整个文件）：表单的值就是生成的 `WorkspaceInvitationsCreate`（Plane 的 `InvitationFormValues`、`EmailRole` 和 `role: EUserPermissions` 删除）；提交经 `followInSession`；成功时清空、提示，把 nerve 建的邀请交给 `onSent(invitations)`（表单所在的地方决定之后做什么：弹窗关闭，Task 6 的新手引导列出链接）；`clear()` 清空表单，弹窗自己在离场的过渡之后调它（原来 hook 的 `handleClose` 知道弹窗的 350 毫秒）；被拒绝时 `invitationRefusal` 决定原因放在 nerve 指出的行下（`setError("invitations.<i>.email", …)`）还是提示。原来在 `catch` 里读 `err.error` 再重新抛出（没有处理的拒绝）。
- `invite-modal/refusal.ts`：`invitationRefusal(error, rows)`，`{ kind: "rows"; rows } | { kind: "toast" }`（修复轮起不带文案，2.15）：nerve 的字段错误都指向表单里的行（`invitations[<i>].email`）时落到行下（`not_allowed` 是"已是成员"，`duplicate` 是"已有邀请"，其余按 `FIELD_ERROR_MESSAGES`）；有一条不在行上时整个提示，文案由 `useRefusalToast` 按 `errorMessageKey` 给出。
- `core/lib/invitation-link.ts` 的 `invitationLink(origin, invitation)`：`/workspace-invitations?invitation_id=…&token=…`，经 `URLSearchParams` 编码（原来在列表项里拼字符串，令牌不编码；9.5）。
- `useInvitationChanges()`（`core/components/workspace/settings/use-invitation-changes.ts`）：`changeRole(workspaceSlug, invitationId, role: WorkspaceRole)` 发 `{ role }`（编号）、`remove(workspaceSlug, invitationId)`（成功时提示），都经 `followInSession`，拒绝按 `code` 提示。与 `useMembershipChanges` 同一形状；参数照 store 的修改给出工作区和邀请（列表项在地址没有工作区时不渲染，hook 要先于这个判断调用）。原来列表项自己写这两段，它们的跟进没有一个测试守着（预检 M1）。
- 弹窗收 `invite`（成员页传 `inviteMembersToWorkspace`，它交回 nerve 建的邀请；成员页自己的 `handleWorkspaceInvite` 删除），`onSent` 是它的 `onClose`；行的角色选项是 `WORKSPACE_ROLES`。邀请列表：已忽略的邀请（`responded_at` 不为空）标"已忽略"，没有角色选择和复制链接，只能删除（P8a spec 第 3 节第 12 条）；删除、改角色经 `useInvitationChanges`；复制链接经 `useCopyInvitationLink()`（`core/hooks/use-copy-invitation-link.ts`，成功、失败都提示；Task 6 的邀请一步也用它）。文案 `members.declined`、`modal.errors.already_member`、`already_invited`。web 的 oxlint 上限 360 → 359（之后 Task 4 的修正轮 357，修复轮 356，附录 A.5）。邀请在路上时弹窗的取消按钮禁用（弹窗本来没有 `handleClose`，Escape 和背景不关；修复轮，裁定 F-1；W4 核对）；成员页和邀请列表的确认框（`confirm-workspace-member-remove.tsx`）同样，W7 核对。
- 测试：`refusal.test.ts` 6 行、`invitation-link.test.ts` 5 个（Task 4 加 2 个）、`members/invite-modal.test.tsx` 6 个（请求体是 `WorkspaceInvitationsCreate`、角色是编号；不是地址的不发；行的拒绝不关、不提示（行下的原因由 W4 核对，预检 L1）；整个的拒绝提示原因；两种换账户之后兑现）、`settings/use-invitation-changes.test.ts` 8 个（角色是编号、发到邀请的工作区；删除之后提示；两种修改被拒绝的 `it.each`；两种修改各两种换账户之后兑现）。
- 端到端：W4 的页面版本（成员的页面不显示也不请求邀请；已忽略的标"Declined"；两行一次发出，角色是编号；改角色 `{ role: 15 }`；复制的链接；先经 API 删掉的邀请，删除得到 404 并提示；已是成员、已忽略的邀请的邮箱各在自己的行下被拒绝；删除已忽略的之后可以再邀请）。`assert/workspace.ts` 的 `expectInvitations` 让同一邮箱的两条按创建时间排（第 3 节第 13 条）。

### 2.4 邀请页；打开刚加入的工作区；W5 的页面版本（Task 4；W5，3.8，3.14，7.1，7.4；决策点 1；P8a 的 P27；Codex M-4）

- `invitation-view.ts`：`invitationView({ link, preview, signedIn, mismatched })`：链接缺 id 或令牌、nerve 按链接找不到（400、404）是 `invalid`；nerve 没有回答是 `unavailable`；预览在路上是 `loading`；然后已忽略 `declined`、没有登录 `sign-in`、nerve 对调用者的回答说是别的邮箱的 `mismatch`，其余 `answer`。
- `use-open-workspace.ts`：`useOpenWorkspace()` 给出 `(workspace) => Promise<void>`：写成上次打开的工作区（`updateUserProfile({ last_workspace_id })`，尽力而为），再去 `/{slug}`，经 `followInSession`；修复轮起交回导航的 `Promise`，导航完成之后才兑现（裁定 F-2）。接受邀请、创建工作区（Task 5）之后用它。
- `invitationAuthPath("/" | "/sign-up", invitation)`：从链接去登录或注册的地址，带 `invitation_id`、`token`（页头显示要加入的工作区，Task 7 的注册带上邀请）和回到链接的 `next_path`。
- `useInvitationAnswer({ reread, mismatched })`（`core/components/workspace/use-invitation-answer.ts`）：`accept(invitationId, token)`、`decline(invitationId, token)`，都经 `followInSession`：接受之后 `useOpenWorkspace`，忽略之后 `reread()`；nerve 拒绝时：邮箱不一致（`workspace.invitation_email_mismatch`）调 `mismatched()`，其余按 `code` 提示并 `reread()`。原来这两段写在页面里，它们的跟进没有一个测试守着（预检 M1）。
- 邀请页（整个文件）按 `invitationView` 显示，文案经 `t()`（`workspace_invitation.*`，14 条，两种语言），标题写工作区和角色，不写邮箱；接受、忽略经 `useInvitationAnswer`（`reread` 重读预览；`mismatched` 转到 `mismatch`，只给"退出登录"：Codex M-4，打开时无从知道）。P27 的四条都在这里：缺 id 或令牌不再一直转圈；接受、忽略之后不再显示缓存的预览；没登录的人不再看到"接受"（原来得到 401 进 `console.error`）。
- 测试：`invitation-view.test.ts` 10 行；`use-open-workspace.test.ts` 4 个（修复轮加 1 个：导航的 `Promise` 兑现之后才兑现）；`invitation-link.test.ts` 加 2 个；`use-invitation-answer.test.ts` 10 个（接受之后打开 nerve 答的工作区；忽略之后重读；两种回答被拒绝为别的邮箱、被拒绝说原因并重读的 `it.each`；两种回答各两种换账户之后兑现）。
- 端到端：W5 的两个页面版本（未登录 → 登录再回到链接 → 接受、落在工作区、写成上次打开的；另一条忽略之后说已忽略；链接打开时被删除的邀请，接受被拒绝、说明原因，之后无效；另一个邮箱的邀请，接受被拒绝之后说发给了另一个邮箱、只给退出登录、什么都不变；令牌改一位或没有令牌：无效）。

### 2.5 创建工作区的共用 hook；`/create-workspace`；W1 的页面版本（一）（Task 5；W1，3.10，3.11，3.14，7.1；P8a 的 P9、M7 和第 5 节的 `t5-slug-ignored`）

- `use-create-workspace.ts`：`CreationForm`（`Pick<WorkspaceCreate, "name" | "slug"> & { organization_size: OrganizationSize | null }`）；`slugFrom(text)`（小写，空格换成 `-`：字段显示的就是发出的）；`creationRefusal(error)`，`{ kind: "fields"; fields } | { kind: "toast" }`（修复轮起不带文案，2.15；检查之后被占用的 `workspace.slug_taken` 落到 slug 下；字段错误只指向 `name`、`slug` 时落到字段下，`not_allowed` 是保留、`invalid_format` 是格式不对；否则提示，文案由 `useRefusalToast` 按 `errorMessageKey` 给出）；`useCreateWorkspace()` 给出 `(form, setError) => Promise<Workspace | undefined>`：先问 nerve slug 能不能用（`checkWorkspaceSlug`），不能时不发，原因放在 slug 下；能用时发 `{ name, slug, organization_size? }`；成功时提示并交回工作区；经 `followInSession`，换账户之后既不提示也不交回。`/create-workspace` 和 Task 6 的新手引导共用它。修复轮（裁定 F-5，评审的 M4）在它旁边加 `useCreationForm(onCreated)`：两个创建表单共用的表单（打开时是空的）、字段的规则（名称必填、`validateWorkspaceName`、80 字；slug 必填、48 字；规模必填）、`onNameChange`（名称给出 slug）和提交；提交创建工作区，把它和发出的值交给 `onCreated`，等它兑现之后表单才不忙。两个表单各自排版。
- `CreateWorkspaceForm` 收 `onCreated(workspace)`，表单自己的 slug 检查（`validateSlug`、`invalidSlug`、`slugError` 和两段文字）删除：nerve 的检查是唯一的检查（第 3 节第 8 条）。`/create-workspace` 创建之后 `useOpenWorkspace`，表单忙到工作区打开（Task 5 的修正轮；修复轮起等到导航完成）；表单的值是它自己的，在 `AuthenticationWrapper` 之下，换账户时随表单卸载（修复轮，裁定 F-3：原来页面在包装层之上留着草稿，另一个账户看得到）；创建关闭时说明已关闭、请工作区的管理员给一个邀请链接，与新手引导一致（W1"两个入口都显示'创建工作区已关闭'"），Plane 的"只有实例管理员能创建"和给实例管理员写信的按钮删除（第 3 节第 7 条）。
- 文案：`url_alphanumeric` 改成 nerve 的规则，加 `url_reserved`；`creation_disabled` 的标题和说明改写；`creation_disabled.request_button`、`request_email.subject`、`request_email.body` 删除（附录 A.8）。
- 测试：`use-create-workspace.test.ts` 16 个（检查、创建、提示、交回；没有规模时不发；三种不能用的 slug 不发、原因在 slug 下；字段的拒绝落到字段；没有字段的拒绝提示；两种换账户之后兑现；`creationRefusal` 5 行；`slugFrom` 2 行），Task 5 的修正轮加 3 个（slug 的检查在换账户之后才回答或被拒绝的两行；检查被拒绝时提示，不当作已占用），共 19 个。P8a 第 5 节的 `t5-slug-ignored`（创建不等 nerve 对 slug 的回答）从这里起由 vitest 和端到端两层发现（`T5.1`）。
- 端到端：`/create-workspace` 的 W1 页面版本（三种 slug 都没有 `POST`，原因在字段下；以字段显示的 slug 创建，请求体经 `sentTo`；写上次打开的工作区的回答被扣住、之后工作区页面的脚本被扣住时，按钮一直不能用；打开、写成上次打开的；另一个标签页登录另一个账户之后表单是空的；创建关闭的 nerve 上说明已关闭，没有表单，没有 `mailto:` 链接）。

### 2.6 新手引导；W1 的页面版本（二）（Task 6；W1，7.1，7.4；决策点 2；P8a 的 P7、P21、M4、M7，P8a spec 第 3 节第 12 条；M2 交接第 2、11 节的页面一侧）

- `onboarding-place.ts`：`OnboardingPlace`（`kind` 是 `EOnboardingSteps` 的三步，邀请一步带它邀请到的工作区）；`resumedPlace(profile, workspaces)`：资料、创建都已完成、还没邀请、创建的工作区（写成上次打开的那个）仍在列表中时在邀请一步（邀请到它，不是列表的第一个，M4）；资料已完成、没有创建、没有工作区时在创建一步；其余在资料一步。`afterProfile(workspaces)`：有工作区的人资料一步之后完成，没有的去创建。
- `onboarding/root.tsx`（整个文件）：`OnboardingRoot` 取调用者的工作区列表（M7：原来读一份没人取过的列表），到之前显示加载，取不到时 `SessionUnavailable`；`OnboardingSteps` 按 `onboarding-place.ts` 决定步骤，资料一步之后、创建之后、完成时的修改都经 `followInSession`，拒绝按 `code` 提示；创建的工作区与它的步骤一起写成上次打开的。`profile.store.ts` 的 `finishUserOnboarding` 不再把列表中的第一个工作区写成上次打开的。
- 资料一步（`steps/profile/root.tsx`）：名字的保存经 `followInSession`，只在发出名字的会话里、保存成功时调 `onDone()` 把引导交给根（预检 M2：原来 `await` 之后直接交给根，另一个标签页换了账户之后，根的完成或下一步在新会话里发生，旧 store 的请求被 `SessionChangedError` 截断，"出错了"的提示显示在那个账户的页面上；探查 4 次都复现）。这一步自己的提示仍是 M2 的（第 5 节 P11）。
- 步骤：`steps/root.tsx` 按 `place` 显示，收 `onNamed`、`onCreated(workspace, alone)`、`onDone`；"创建或加入"的包装 `steps/workspace/root.tsx` 删除；创建一步改用 `useCreateWorkspace`（它自己的 slug 检查删除，slug 框的红边原来读 `errors.name`，改为 `errors.slug`）；邀请一步（`steps/team/root.tsx`，整个文件）用成员页的表单（`useWorkspaceInvitationActions`、`InvitationFields`），邀请到这一步的工作区，发出之后 `steps/team/links.tsx` 列出每个邀请的链接和"复制链接"，再"继续"；"以后再说"结束引导（第 3 节第 9 条）。表单的 `onSent` 交来的邀请就是列出的，复制用 Task 3 的 `useCopyInvitationLink()`。
- 删除：`@nerve/utils` 的 `validateSlug`、`SLUG_REGEX`；`@nerve/constants` 的 `ROLE_DETAILS` 的 `i18n_description` 和它指向的 `role_details.{guest,member,admin}.description`（两种语言）：唯一的读者是旧的邀请一步（预检 L2）；文案 `onboarding.invite.role`、`onboarding.invite.not_an_email`、`workspace_creation.toast.error.title`、`message`；改写 `onboarding.workspace.creation_disabled`（P7）；加 `onboarding.invite.links.title`、`description`。
- 测试：`onboarding-place.test.ts` 8 行；`onboarding/root.test.tsx` 10 个（9.5：已有工作区的人资料一步之后完成；一步的资料修改和完成引导各两种换账户之后兑现，预检 M1），Task 6 的修正轮加 3 个（列表取不到；单人工作区的结束在换账户之后兑现的两种），共 13 个；`steps/profile/root.test.tsx` 3 个（`react-hook-form` 的 `useForm` 换成留下提交函数的替身：名字发出之后交给根；被拒绝时不交；换账户之后才保存成功时不交），修复轮加 1 个（这一步等根交回的结束兑现），共 4 个。
- 忙碌的状态（Task 6 的偏离和修正轮，修复轮）：只给自己建工作区（"Just myself"）的人，创建一步等引导的结束；nerve 拒绝结束时说明原因，再进入那个工作区的邀请一步。修复轮（裁定 F-2）起结束引导的每个按钮都忙到 nerve 回答结束：资料一步（已有工作区的人）、邀请一步的"以后再说"和链接之后的"继续"。
- 端到端：W1 的四个新手引导版本（新人：资料、创建、写成上次打开的、邀请到它、复制链接、完成、落在它；新人在邀请一步"以后再说"：落在他建的工作区，没有邀请的请求；只给自己建工作区的人：结束的回答被扣住时创建一步一直不能用，nerve 拒绝结束之后去邀请一步；先接受了邀请的人资料一步之后完成；接着做的人邀请到他建的工作区，不是列表的第一个；创建关闭的 nerve 上资料一步之后说明要邀请链接）。修复轮起，结束的回答被扣住时，链接之后的"继续"、"以后再说"、资料一步的"继续"都一直不能用（`sentHeld`、`enabledWithin`）；第二、三个版本中 Bob 的页面也经 `watchPage` 核对失败的请求和控制台。S2 加新人打开 `/`（2.12；像其余的 S2 测试，安静的控制台之后再读一次请求，预检 L3）。

### 2.7 带邀请的注册（Task 7；W6，3.8，7.4；决策点 1；M2 设计 3.18）

- `auth-root.tsx`：地址带 `invitation_id` 和 `token` 时把 `{ id, token }` 交给密码表单；`password.tsx` 的注册发 `{ email, password, invitation }`（生成的 `RegisterRequest`，注册关闭时 nerve 凭它放行），登录不带。原来注册页的页头显示要加入的工作区，注册本身却不带邀请。
- `auth-screens/header.tsx`：登录页和注册页之间的链接带上当前页面的查询参数（邀请和 `next_path`）。
- 测试：没有新的 vitest（注册的请求体由端到端经 `sentTo` 钉住）。端到端：W6 的两个页面版本（注册关闭：没有链接时注册页说已关闭；凭链接注册，请求体带邀请；回到链接接受，新手引导只有资料一步，落在工作区；注册开放：从链接的注册页去登录页，登录页保留链接、登录之后回到它）。

### 2.8 导航设置的修改在轮到它时算；项目导航对话框；首页的导览；W8 的页面版本（Task 8；W8，3.18，7.1，7.5；总体设计 7.7；P8b 的 E5，P8a 的 P28）

- `WorkspacePreferencesStore.updatePreferences(slug, change: PreferencesChange)`，`PreferencesChange = (held: WorkspacePreferences) => WorkspacePreferencesUpdate`：在队列轮到它时作用于 nerve 最近一次回答的设置再发出（照 P8b 的 `ProjectPreferencesStore`）；还没有设置时不发出，以 `Workspace settings not found` 失败。原来收现成的请求体，页面按调用时侧边栏显示的值算（E5）。
- `core/hooks/navigation-preferences.ts`（整个文件）：`NERVE_DEFAULTS`（`ACCORDION`、10）是 `navigationOf` 的缺省参数（设置到达之前按默认值显示，不再从"全部项目"跳到 10 个；`@nerve/types` 的 `DEFAULT_PROJECT_PREFERENCES` 删除）；`TProjectNavigationChange` 是模式、限制的一次开关（`{ limitToggled: true }`）或数量；`preferencesChangeOf(change): PreferencesChange`：开关由它作用的设置决定打开还是关闭（限制项目数时关闭，是 0；不限制时打开，取 `navigationOf(held)` 的数量），连按两下回到原样（预检 M3：原来对话框发出点击时显示的开或关，在 nerve 回答第一下之前的第二下发出同一个值；P8b 评审 P14 的 `toggleProject` 是同一类）；`countOf(draft)`（数字组成的数，至少 1）。
- `use-navigation-preferences.ts`（整个文件）：`useProjectNavigationPreferences()` 给出 `{ preferences, changeNavigation }`，经 `followInSession`，拒绝按 `code` 提示（原来三个更新函数，被拒绝时是没有处理的拒绝）。`ProjectNavigationDialog`：开关发出 `{ limitToggled: true }`；数量框的文字是草稿（`null` 时显示设置的数量），失去焦点时发出一次；标签经 `htmlFor`/`useId` 关联输入框。`sidebar-wrapper.tsx` 打开对话框的按钮有名字（`aria_labels.projects_sidebar.project_navigation`）。`home/root.tsx` 的导览结束经 `followInSession`，拒绝按 `code` 提示（原来只有 `console.error`）。
- 测试：`preferences.store.test.ts` 的修改都写成函数，加 3 个（作用于前一个修改的回答；连按两下开关，每一下作用于前一下的回答：先关闭再打开；没有设置时不发出）；`navigation-preferences.test.ts` 11 行；`use-navigation-preferences.test.ts` 5 个（开关作用于 nerve 最近的回答，不是提出时显示的设置）；`home/root.test.tsx` 4 个。
- 端到端：W8 的页面版本（跳过导览的请求体；默认值下四个项目都显示；标签页式；连按两下限制的开关，第二下在 nerve 回答第一下之前（`holdAnswer`）：先关闭 `0`，再在那个回答之上打开 `10`，回到原样；输入 30 删一位再离开，只发一次 3；刷新之后仍是 3 个、对话框显示 3；nerve 拒绝 3000000000（422），页面说明，侧边栏仍是 3 个；一共五个 `PATCH`）。

### 2.9 停用弹窗（Task 9；W9，3.7，3.9，7.1，7.5；P6 review 第 6 节）

- `DeactivateAccountModal`：nerve 拒绝停用时（409 `workspace.sole_admin`、`project.sole_admin`），原因（`errorMessageKey`）写在弹窗里（`role="alert"`），弹窗不关；关闭或再次确认时清掉。原来是提示。停用成功时，只在它结束了标签页的会话时提示"账户已停用"并关闭（第 3 节第 3、4 条）。
- `UserStore.deactivateAccount(): Promise<boolean>`（M2 的 store，原来 `Promise<void>`）：发出时读标签页的会话，nerve 停用之后结束它（`tokenManager.endSession(loginId)`），交回 `endSession` 的回答。
- 测试：`deactivate-account-modal.test.tsx` 3 个（被拒绝时不提示、不关；弹窗里的原因由 W9 核对：服务端渲染看不到之后才设的状态，预检 L1）；`user/index.test.ts` 加 1 个 `it.each`（2 行）。端到端：A12 被拒绝的一段改读弹窗里的原因；W9 的页面版本（409 的原因在弹窗里，账户和成员关系不变；取消再打开，原因不在了；把成员改成管理员之后停用成功，回到登录页，成员关系结束）。

### 2.10 工作区列表的唯一取数；`WorkspaceAuthWrapper` 的界面（Task 10；7.1，7.5；P8a review 的 M7；M2 交接第 13 节）

- `core/hooks/use-workspaces-fetch.ts`：`useWorkspacesFetch(wanted = true)`，调用者的工作区列表的唯一取数（`useSessionSWR(wanted ? ["WORKSPACES"] : null, …)`）。落点（`use-landing.ts`）、工作区的页面（`use-workspace-fetch.ts`）、新手引导（Task 6 的取数）改用它；个人设置的侧边栏（`settings/profile/sidebar/workspace-options.tsx`）挂载时经它取列表（M7：直接打开 `/settings/profile/*` 时侧边栏和命令面板的工作区菜单原来是空的）。
- `WorkspaceAuthWrapper`："找不到工作区"的界面经 `t("workspace_not_found.*")`（两种语言各 5 条），退出登录失败的提示经已有的 `auth.sign_out.toast.error.*`（原来写死的英文；修复轮起经 `useSignOut`，2.15）。`workspace-wrapper.test.tsx` 改为核对键。"找不到"的判断不变（7.5、P19）。
- "唯一的取数"由关键词守卫的规则 `workspaces-list-fetch` 看住（2.14；预检 L10）。
- 端到端：S2 加直接打开个人设置（2.12；安静的控制台之后再读一次请求）。

### 2.11 文档（Task 11；3.20 中 P9 的一行；M2 交接第 2 节；裁定 P3）

- `README.md` 的"前端"一节："M2 中看到的页面"改写为"能用的页面（M3/P9 起）"。
- `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态；3.2"所有处理接口错误的地方"写明 P9 改到的页面；新增五行（页面跟进修改的结果、创建工作区、新手引导的邀请成员一步、侧边栏的项目导航设置、停用账户的弹窗），都是"已完成，M3/P9"。
- `docs/v0/M3-workspace-project/handoffs/M2-closeout.md` 加"处理结果（M3/P9）"：第 1、2 节的页面一侧完成；第 11 节的页面一侧中工作区的部分；第 13 节 P9 的部分（浏览器核对是 C4）；M2 的 `UserStore.deactivateAccount` 交回 `Promise<boolean>`（`endSession` 的回答），停用弹窗只在为 `true` 时说已停用（裁定 P3：M2 的 store 的说明随签名改）。

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

页面在修改兑现之后跳转、提示、改界面的每一处，都经 `followInSession`（只在发出它的会话里跟进），或不跟进。每一处都由它自己的 vitest 守着：修改在另一个标签页换了账户之后才兑现（`lateSettlings` 的两种），页面什么都不做；那一处不经 `followInSession` 自己跟进时，这个测试失败（裁定 P2 的条件；附录 A.2 的 W3 一类）：

| 组件 | 修改 | 跟进 | 守着它的 vitest | Task |
|---|---|---|---|---|
| `delete-workspace-modal.tsx` | 删除工作区 | 关闭、回到 `/`、提示；拒绝提示 | `delete-workspace-modal.test.tsx` | 1 |
| `workspace-details.tsx` | 改名、规模、时区 | 提示；拒绝提示 | `workspace-details.test.tsx` | 1 |
| `use-membership-changes.ts` | 改角色、移出、离开 | 离开之后回到 `/`；拒绝提示 | `use-membership-changes.test.ts` | 2 |
| `use-workspace-invitation.tsx`（成员页的弹窗、新手引导的邀请一步） | 批量邀请 | 清空、提示、交给 `onSent`；拒绝落到行下或提示 | `members/invite-modal.test.tsx` | 3、6 |
| `use-invitation-changes.ts`（邀请列表） | 删除邀请、改邀请的角色 | 删除之后提示；拒绝提示 | `use-invitation-changes.test.ts` | 3 |
| `use-invitation-answer.ts`（邀请页） | 接受、忽略 | 打开工作区（`useOpenWorkspace`）、重读预览；邮箱不一致的界面；拒绝提示 | `use-invitation-answer.test.ts` | 4 |
| `use-open-workspace.ts` | 写上次打开的工作区 | 去工作区（成功、失败都去） | `use-open-workspace.test.ts` | 4 |
| `use-create-workspace.ts`（`/create-workspace`、新手引导的创建一步） | slug 的检查和创建 | 原因在字段下、提示、交回工作区 | `use-create-workspace.test.ts` | 5 |
| `onboarding/root.tsx` | 资料一步、创建一步写进资料（`change`），完成引导（`finish`） | 拒绝提示 | `onboarding/root.test.tsx`（两者各两行） | 6 |
| `onboarding/steps/profile/root.tsx` | 资料一步的名字 | 交给根（`onDone`）；这一步自己的提示是 M2 的（第 5 节 P11） | `steps/profile/root.test.tsx` | 6 |
| `use-navigation-preferences.ts`（对话框） | 导航设置 | 拒绝提示 | `use-navigation-preferences.test.ts` | 8 |
| `home/root.tsx` | 导览结束 | 拒绝提示 | `home/root.test.tsx` | 8 |
| `deactivate-account-modal.tsx` | 停用账户 | 只在结束了标签页的会话时提示并关闭；拒绝写在弹窗里（第 3 节第 3、4 条） | `deactivate-account-modal.test.tsx`、`user/index.test.ts` | 9 |

不跟进的：
- 工作区菜单的切换（链接直接跳转，写 `last_workspace_id` 不等回答，P8a）；退出登录失败的提示（退出失败时标签页的会话没有变；修复轮起六处都经 `useSignOut`，2.15）。
- 一次登录或换账户经过包装层的加载，子组件重新挂载；退出登录时 PUBLIC 页面的子组件仍挂载着，所以这样的页面所持的会话的东西只在它登录之后的视图里读；包装层之上的状态比任何一次改变都长，不放会话的东西（修复轮，裁定 F-8 的搁置项 P7）。
- 组件自己的状态：确认框在 `await onSubmit()` 之后关闭（`ConfirmWorkspaceMemberRemove`，成员页和邀请列表）、general 页的按钮结束加载、停用弹窗的按钮结束加载和被拒绝的原因。另一个标签页以另一个账户登录时，`AuthenticationWrapper` 在新账户的资料到达之前显示加载，页面连同这些组件卸载，之后兑现的修改写进的是已卸载的组件，不显示（第 3 节第 4 条，裁定 P4；预检在代码中和探查中核对过，3 次都是这样；预检 L4）。提示和跳转是全局的，不在此列，所以都经 `followInSession`。
- M2 的账户页面（新手引导的资料一步自己的提示、个人设置的三页）在第 5 节 P11。

### 2.14 静态检查的范围（Task 1–6、8–10；总体设计 7.7；预检 M4、L10）

- 根目录 `.oxlintrc.json` 的两条 `overrides`（P8a 起，P8b 加过它的文件；总体设计 7.7："之后的 M 把接上新接口的取数文件加进去"），各在加入文件的 Task：
  - `no-restricted-imports`（会话的取数不从 `swr` 导入值）：工作区列表的三个取数文件 `onboarding/root.tsx`（Task 6）、`core/hooks/use-workspaces-fetch.ts`、`settings/profile/sidebar/workspace-options.tsx`（Task 10）。邀请页在 P8a 起已在范围内；P9 的其余取数经它们。
  - `typescript/no-non-null-assertion`：P9 的新模块和测试中不在已有范围（`workspace/settings/**` 等）里的：Task 1 `fake-tab.ts`、`fake-refusal.ts`、`delete-workspace-modal.test.tsx`；Task 2 `workspace-roles.ts`；Task 3 `invitation-link.ts`、`invite-modal/refusal.ts` 和它们的测试、`members/invite-modal.test.tsx`、`use-copy-invitation-link.ts`；Task 4 `invitation-view.ts`、`use-invitation-answer.ts`、`use-open-workspace.ts` 和它们的测试，修正轮的 `ui/empty-space.tsx`；Task 5 `use-create-workspace.ts` 和它的测试；Task 6 `onboarding-place.ts` 和它的测试、`onboarding/root.test.tsx`、`steps/profile/root.test.tsx`、`steps/team/links.tsx`；Task 8 `use-navigation-preferences.test.ts`、`home/root.test.tsx`；Task 9 `deactivate-account-modal.test.tsx`；Task 10 `use-workspaces-fetch.ts`；修复轮的 `use-refusal-toast.ts`、`use-sign-out.ts` 和它们的测试。它们都没有非空断言。
- `tools/keywords.json` 加规则 `workspaces-list-fetch`（Task 10）：web 应用的源文件中，store 自己（`core/store/workspace/`）、测试和测试的替身（`fake-store-hooks.ts`）之外，只有 `use-workspaces-fetch.ts` 能写 `fetchWorkspaces`：调用者的工作区列表只有一个键、一份取数（第 3 节第 14 条）。预检的 PF11（个人设置的侧边栏以自己的键取列表：S2 看到的仍是一个请求，`revalidateOnMount` 每次挂载都取）由它在静态一层发现（`T10.6`）。修复轮（裁定 F-4）再加两条，形状相同（2.15）：`refusal-toast`（`setToast` 里按 `errorMessageKey` 的失败提示只写在 `core/hooks/use-refusal-toast.ts`；主题切换经 `setPromiseToast` 自己说失败，规则不管，第 5 节收尾）、`sign-out-toast`（`auth.sign_out.toast.error` 只写在 `core/hooks/use-sign-out.ts`），测试除外；规则 65 → 67 条。
- 变异（附录 A.2 的 W4、W12 两类）：`no-restricted-imports` 的三个文件各一个（取数换成 `swr` 的），非空断言的范围每个 Task 加的一组各一个（在其中一个文件里写一个非空断言），`check:lint` 都失败。

### 2.15 修复轮（评审之后；裁定 F-1–F-8）

评审（C0、I1、M6）和执行中搁置的 P1–P12 由裁定 F-1–F-8 交给一次修复轮，在 `dba0d756` 之上（变异和它们发现的检查在 `$M3TMP/p9tools/p9-mutation-record.md`）：

- **请求在路上时弹窗关不掉**（F-1，评审 I1，Task 9 的根因修法）：删除工作区的弹窗、离开或移出的确认框的取消按钮禁用，`ModalCore` 的 `handleClose` 为空（Escape 和背景不关）；邀请弹窗的取消按钮禁用。W3、W4、W7 的页面版本扣住请求核对（`closedByEscape`）；A12 的 Escape 一步改为有期限的"关不掉"。W4 的 Escape 一步是补充（裁定 F-11）加的：修复轮写了 W4 用它，却没有那一步；加上之后它起初在变异 `I1.f`（邀请弹窗有 `handleClose`）下仍通过，因为页面上还有一条提示，也是 `dialog`，定位两个元素的失败被当成"没关"，于是 `closedByEscape` 改为只认模态的对话框、只把超时当成没关（第 2 节的共用部分）；`I1.d`、`I1.e`、`T9.7` 在改过的检查下重跑，照旧被发现。
- **忙碌到跟进做完**（F-2，评审 M1，搁置项 P9）：`followInSession` 等跟进交回的 `Promise`（第 2 节的共用部分）；`useOpenWorkspace` 交回导航的 `Promise`；新手引导的根在结束的一支交回 `finish()`，资料一步交回 `onDone()`，"以后再说"和链接之后的"继续"在结束的回答到达之前显示忙碌（2.4、2.6）。
- **`/create-workspace` 不跨会话留草稿**（F-3，评审 M2）：页面在包装层之上留着的草稿（`defaultValues`、`setDefaultValues` 和卸载时的保存）删除，表单只有自己的值（2.5）。
- **一处失败提示、一处退出登录**（F-4，搁置项 P12、评审 M3、搁置项 P1 退出的一半）：`core/hooks/use-refusal-toast.ts` 的 `useRefusalToast(titleKey = "toast.error")` 给出 `(error) => void`，按 `t(errorMessageKey(error))` 提示，取代 M2、P8b、P9 的 26 处相同的写法和 3 处自己标题的；先在字段和提示之间决定的几处（创建工作区、邀请表单、资料一步、个人设置的 general 表单、安全页、API 令牌的表单）照旧决定，提示的一支调它；停用弹窗里的原因不变。`core/hooks/use-sign-out.ts` 的 `useSignOut()` 给出 `() => Promise<boolean>`：退出失败时提示 `auth.sign_out.toast.error.*`，交回 `false`；取代五处退出登录和邀请页的一处（原来没有失败的一支）。每个 hook 一个 vitest（3 个、2 个），每一行各发现一个有名字的变异（补充，裁定 F-11：`M3.c`、`P12.f`）；关键词守卫的两条规则让 `setToast` 里按 `errorMessageKey` 的失败提示和退出失败的提示各只写在 hook 里（2.14）。主题切换经 `setPromiseToast` 的失败提示仍自己按 `errorMessageKey` 写，不在规则内（第 5 节收尾）。创建工作区和邀请表单的拒绝的 `toast` 一种不再带文案的键（`{ kind: "toast" }`），文案由 hook 按错误给出。
- **小的改动**（F-5）：`useMemberColumns(workspaceSlug)`（2.2）；`useCreationForm`（2.5）；`EmptySpaceItem` 的按钮里不放 `<div>`（`<span className="block …">`，外观不变）。
- **测试**（F-6）：两处复制的失败提示（W4，`refuseClipboardWrites`）；`anotherTabSignsIn`（第 2 节的共用部分）；P8b 的四个手写的 `ApiError` 改用 `refusal()`；W1 中 Bob 的页面经 `watchPage` 核对，单人工作区的结束本身被扣住（不再靠排在它前面的那次写）；A2 的 `showSignIn` 写明它的正向等待为什么够（包装层在有实例的设置之前不渲染页面）。
- **变异的记录**（F-7）和**文档**（F-8，本节和第 2–5 节、附录中标"修复轮"的句子）。

## 3. 与设计的差异、补充和需要裁定的

标"裁定"的八条由控制者裁定（P1–P8，2026-10-09，都接受，`$M3TMP/p9-architect-rulings.md`）；其余是本 Phase 在设计之内做的决定和理由，控制者照写接受。预检之后的修订在本节末尾的"预检之后"。

1. **Task 的切分**（裁定 P1：接受）：设计 12 节 P9 的草稿是 14 个任务，plan 是 11 个（P7、P8 的先例：在上限之内重新切分）。对应：设计 1（落点的页面一侧）分散在写 `last_workspace_id` 和落点的各 Task（1、2、4、5、6）；2 → 6；3 → 4；4 → 7；5 → 8；6 → 1；7 → 2；8 → 3；9 → 9；10 → 10；11（会话核对）→ 1（`followInSession`、删除组件的 vitest、W3 的会话切换）和各 Task 的页面；12、13（端到端）并进各页面的 Task（测试与代码同一个提交，每个 Task 之后 `make e2e` 通过）；14 → 11（只改文档；C1–C5 和评审是控制者的）。11 个 Task 都在约 16 个之内，每个在约 1,500 行之内：最长的 Task 6 是 1,475 行，Task 3 是 1,372 行（附录 A.6、A.9；修订让 Task 6 长到 1,687 行，邀请的两个 hook 随之移到 Task 3，见"预检之后"的最后一行）。
2. **会话核对写在一处**（裁定 P2：接受，条件是每个组件不经它自己跟进时由它自己的 vitest 发现；设计原来写"组件在 `await` 之后先核对 `inSession()`"，M3 设计 7.1、9.6、12 节 P9 的第 11 条和 P10 的第 8 条、Codex 4.3 第 2 条和复核 M6 两行，总体设计 7.7 的规则和代码一句，都在本次修订中改为写明 `followInSession`，规则不变）：每个组件自己取 `sessionGuard()`、`await`、核对、再跟进，十几处写同一段；P9 把它写成 `followInSession(change, { done?, failed })`，组件只给出跟进什么。9.6 的变异"去掉组件里 `inSession()` 的核对"在这里是去掉 `followInSession` 的核对（`T1.1`），由 W3 的会话切换端到端和三个 vitest 文件发现；每个组件"不经 `followInSession` 自己跟进"由各自的 vitest 发现（2.13 的最后一列，附录 A.2 的 W3 一类；预检看到三处没有这样的测试，修订补上，见"预检之后"的 A-M1）。
3. **停用的成功经 `endSession` 的回答跟进，M2 的 `UserStore.deactivateAccount` 交回 `boolean`**（裁定 P3：接受；M2 收尾交接的 P9 处理结果写明新的签名，Task 11）：停用成功时自己结束标签页的会话，`sessionGuard()` 的核对随之为假，`followInSession` 用不上；而另一个标签页在停用发出之后把这个标签页换到别的账户时，原来的弹窗照样提示"账户已停用"、关闭，显示在那个账户的页面上（W3 一类）。`tokenManager.endSession(loginId)` 已经回答"标签页的记录是否仍是那个会话的"，store 交回它，弹窗只在为真时提示、关闭。签名改在 M2 的 store：它唯一的调用方是这个弹窗。
4. **停用被拒绝的原因是弹窗自己的状态，不另加会话核对**（裁定 P4：接受，由预检核对卸载：代码中 `authentication-wrapper.tsx` 在 `!currentUser` 时显示加载，`startSession` 建的新 `RootStore` 没有账户；W3 式的探查 3 次都是弹窗在放行之前已卸载、之后没有提示也没有弹窗）：另一个标签页以另一个账户登录时，`AuthenticationWrapper` 在新账户的资料到达之前显示加载（`!currentUser`），页面连同弹窗卸载，之后兑现的拒绝写进的是已卸载的组件，不显示。所以拒绝一支直接 `setRefusal`（注释写明理由）；`T9.3` 去掉成功一支的核对时由弹窗的 vitest 发现。确切地说（修复轮，裁定 F-8 的搁置项 P7）：一次登录或换账户经过包装层的加载，子组件重新挂载；退出登录时 PUBLIC 页面的子组件仍挂载着，所以这样的页面所持的会话的东西只在它登录之后的视图里读（邀请页如此）；包装层之上的状态比任何一次改变都长，不放会话的东西（`/create-workspace` 原来的草稿是唯一的例外，修复轮删除，2.15）。`AuthenticationWrapper` 的说明写着同一句。
5. **W1 的"含大写"**（裁定 P5：接受；M3 设计第 2 节 W1 一句改为写明实际的样例）：故事写"slug 已被占用、是保留名、含大写：表单在字段下方提示，不提交"。slug 字段照 Plane 的显示把输入转成小写（`slugFrom`），所以"含大写"到不了 nerve，也不会被提示；P9 让字段显示的就是发出的（原来显示转换后的、发出原文）。端到端的"格式不对"用 `café`（nerve 的规则只收小写字母、数字、`-`、`_`）；大写由字段的转换核对：W1 输入大写和空格，字段显示的、发出的都是小写、`-` 连接的 slug（`slugFrom` 的 vitest；变异 `T5.5`、`T5.9`）。
6. **`members-list.tsx` 89 → 91**（裁定 P6：接受）（brief："只为使用方改到的 Plane 文件不变长"）：成员行收地址的工作区（`workspaceSlug` prop），格式化把一行拆成三行。其余变长的 Plane 文件都是本 Phase 的对象（页面行为改在它们里面，附录 A.6）。
7. **`/create-workspace` 创建关闭的文案**：W1 写"两个入口都显示'创建工作区已关闭'"；`/create-workspace` 原来显示 Plane 的"只有实例管理员能创建工作区"和给实例管理员写信的按钮（`mailto:`，信的主题和正文两条文案）。Nerve 没有实例管理员（M2 设计 3.16），创建由部署的配置关闭；P9 改为与新手引导相同的说法（请工作区的管理员给一个邀请链接），删除按钮、页面里拼信的 `getMailtoHref` 和三条文案（按钮、信的主题、正文）。
8. **表单自己的 slug 检查删除，nerve 的检查是唯一的**（W17）：Plane 的 `validateSlug` 允许 Unicode 字母、文案与 nerve 的不同，还可能与 nerve 的原因一起显示两遍。现在不合格的 slug 在点"创建"之后由 nerve 说明（`checkWorkspaceSlug` 的三种原因），不再在输入时提示；`@nerve/utils` 的 `validateSlug`、`SLUG_REGEX` 在新手引导也不用之后删除（Task 6）。
9. **新手引导的邀请一步用成员页的表单**（7.4"邀请成员一步调 `createWorkspaceInvitations`，完成后显示每个邀请的链接供复制"）：Plane 的三行表单（Listbox 的角色、自己的邮箱检查和文案）换成成员页的 `InvitationFields` 和 `useWorkspaceInvitationActions`：一行起，角色是编号，行的错误来自 nerve（`invitationRefusal`）；发出之后列出链接（`links.tsx`）。W1 的故事写"点跳过"，页面上是"以后再说"（Plane 的按钮），它结束引导，W1 的第二个新手引导版本核对它（`T6.9`）。
10. **侧边栏在设置到达之前按 nerve 的默认值显示**（P28 的"侧边栏在显示设置到达前后的切换"）：`navigationOf` 的缺省是 nerve 的默认值（折叠式、10 个），一个从没改过设置的人在设置到达时侧边栏不变；`DEFAULT_PROJECT_PREFERENCES`（Plane 的，显示全部）删除。
11. **数量在失去焦点时发出**（P28 的"失焦或防抖"）：选失焦，不用计时器：数量框是草稿，离开时发一次（按 Escape 关闭对话框时 Headless UI 先移走焦点，同样发出，原型中核对过）。限制的开关不是草稿：每一下是一次开关（`{ limitToggled: true }`），在轮到它时按 nerve 最近的回答决定打开还是关闭，连按两下回到原样（预检 M3；总体设计 7.7 的"轮到它时才算"，P8b 的 `toggleProject` 的先例）。
12. **`sentTo` 的写法**（e2e 的共用部分）：页面的客户端发出的 `fetch`，Playwright 的 `response.request().postDataJSON()` 读不到请求体，所以经路由在请求离开页面时读（这条路由是 `bodiesSentTo`，`sentTo` 和修复轮的 `sentHeld` 都建在它上面，Task 8 的修正轮）；路由按路径的模式匹配，留到页面关闭：在页面发出下一个请求时移除路由，那个请求可能一直挂着（W1 的创建之后紧接着写资料，函数路由加移除在 4 次中失败 1–3 次，模式路由加移除 8 次中失败 1 次，模式路由不移除 12 次都通过）。
13. **`expectInvitations` 的顺序**（P3 的 fixture）：W4 的页面版本中同一邮箱先后有两条邀请（删除已忽略的那条之后再邀请），原来按邮箱排，两条的先后不定，偶尔失败；改为同一邮箱的按创建时间排，期望值也排序。
14. **`useWorkspacesFetch` 是新加的 hook**：P8a 的三处（落点、工作区包装层、Task 6 的新手引导）各写一遍键和取数，个人设置是第四处；四处共用一个，键只写一次。个人设置的取数是新的挂载请求（S2 的行，2.12）。命令面板的工作区菜单读的就是这份列表，不另取（P8a review 的 M7 第三处）。"唯一"由关键词守卫的规则 `workspaces-list-fetch` 看住（2.14；预检 L10：一个以自己的键取列表的组件，S2 看到的仍是一个请求）。
15. **三个 `as const`**（裁定 P7：接受；brief：新的 `as`、`any`、`!` 算发现，除非 spec 裁定）：`in-session.ts` 的两个 `"done" as const`、`"failed" as const`（给联合打上标签），`invitation-link.test.ts` 的 `["/", "/sign-up"] as const`（`it.each` 的路径是字面类型）。都不是类型转换。裁定时还有 `fake-tab.ts` 的 `heldChange` 中一个 `let settle!: …`：修订让 `heldChange` 用 M2 的 `fake-browser.ts` 的 `gate()`（预检 L9），它随之没有了，P9 不写非空断言。源文件中 `as` 7 → 0、`any` 2 → 0、非空断言 0 → 0（附录 A.7）。
16. **M2 的账户页面自己的提示不在 P9 的会话核对中**（裁定 P8：接受）：新手引导的资料一步自己的提示（`steps/profile/root.tsx`）、个人设置的 general、preferences、security 三页在修改兑现之后提示，不核对会话（与 2.13 同一类）。它们是 M2 的页面，修改的是账户，不是工作区一侧，放进 P11 的清理（第 5 节写明四个文件）。资料一步把引导交给根却是 P9 的：它在名字的保存之后调根的 `onDone`，根接着完成引导或进入下一步，这些是工作区一侧的跟进（2.13 的 `onboarding/root.tsx`），所以交接本身经 `followInSession`，只在发出名字的会话里（预检 M2，2.6；裁定 P8 的文字没有分开这两件事）。
17. **删除工作区之后的文案**（W17）：成功的提示原来说"即将跳转到您的个人资料页面"，Nerve 的根路径落到另一个工作区或 `/create-workspace`（3.14），改写；被拒绝时说 nerve 的原因之后，"请重试"一条没有读者，删除（附录 A.8 的检查发现）。
18. **S2 的行**：general 页和成员页的取数 P9 没有改，它们在 Task 1、2 的树上加入 S2，那时的清单就是 `4b1334a5` 的；新人和个人设置两行在 `4b1334a5` 上没有 `GET /api/v0/workspaces`（P8a review 的 M7），Task 6、10 加上。访客的 `/{slug}/settings` 没有时区：工作区设置的布局不给访客 general 页（9.2）。
19. **页面上看得到的不同**（W17）：
    - 会话：工作区一侧的修改（2.13）在另一个标签页换了账户之后兑现时，页面不跳转、不提示、不改界面（原来照样跳转、提示，显示在那个账户的页面上）；停用同样；新手引导的资料一步在换账户之后才保存成功时，不再把引导交给根（原来根在那个账户的会话里完成引导，旧的请求被截断，"出错了"显示在那个账户的页面上）。
    - 错误：general 页的修改、删除工作区、成员页的三个修改、邀请（弹窗和列表）、邀请页的接受和忽略、创建工作区、新手引导的每一步、导航设置、导览的结束被拒绝时，都说 nerve 的原因（原来读 `err.error`、显示通用的文字，或只在控制台记一条，或是没有处理的拒绝）；被拒绝的修改结束加载。
    - general 页：规模没有时第一次选择不再有 React 的"uncontrolled to controlled"控制台错误。删除工作区的成功提示说去另一个工作区或创建（第 17 条）。复制地址失败时提示（原来什么都不说）。删除在路上时弹窗关不掉（修复轮；原来可以取消、按 Escape，回答之后照样跳转）。
    - 成员页：角色是编号（P22；Plane 的选择框值是 `any`）；角色选择勾着那一行现在的角色（`value={rowData.role}`，原来什么都没勾；Task 2，它的评审记下）。离开、移出在路上时确认框关不掉（修复轮）。邀请弹窗：已是成员、已有邀请（含已忽略的）、格式不对的地址的原因在各自的行下，弹窗不关；邀请在路上时取消按钮禁用（修复轮）。邀请列表：已忽略的标"已忽略"，没有角色选择和复制链接；复制失败有提示；链接的查询参数编码。
    - 邀请页：链接缺 id 或令牌时说无效（原来一直转圈）；nerve 没有回答时可以重试；没有登录时给"登录以接受""注册以接受"（原来给"接受"）；接受之后写上次打开的工作区再进入；忽略之后说已忽略（原来缓存的预览仍是未忽略）；邮箱不一致时只给退出登录；被删除或已回答的邀请，接受、忽略被拒绝之后重读预览。文案都经 `t()`。回答在路上时显示加载，连按两下只发一次；选项是真正的按钮（Task 4 的修正轮）。退出登录失败时提示（修复轮；原来什么都不说）。
    - 创建工作区：slug 字段显示的就是发出的；不合格的 slug 在点"创建"之后说明（第 8 条）；创建之后写上次打开的工作区；创建关闭的说法（第 7 条）。表单忙到工作区打开（Task 5 的修正轮；修复轮起到导航完成，原来新页面加载时按钮又显示"创建"）；另一个标签页登录另一个账户之后表单是空的（修复轮；原来留着前一个账户写的）。
    - 新手引导：列表到之前显示加载、取不到时可以重试；先接受了邀请的人资料一步之后完成（原来列表没有取，他也要创建工作区）；接着做时在它离开的那一步（邀请一步邀请到他建的工作区）；邀请一步是成员页的表单，发出之后列出链接（第 9 条）；完成时不再把列表的第一个工作区写成上次打开的；创建一步的 slug 框红边读 slug 的错误。只给自己建工作区的人，nerve 拒绝结束时说明原因，进入那个工作区的邀请一步（Task 6 的修正轮；原来回到创建一步，再点说 slug 已被占用）。结束引导的按钮（资料一步、"以后再说"、链接之后的"继续"）忙到 nerve 回答结束（修复轮）。
    - 注册：凭邀请链接注册时带上邀请，注册关闭时也能注册（原来被拒绝）；登录页和注册页之间的链接保留邀请和回到它的路；注册关闭的 nerve 上注册页仍有去登录页的链接（Task 7 的修正轮）。
    - 侧边栏：设置到达之前按 nerve 的默认值；对话框的数量离开时发一次（原来每个按键一次）、显示设置的数量（原来是挂载时的）；限制的开关每一下按 nerve 最近的回答打开或关闭，连按两下回到原样（原来发出点击时显示的值，快按两下停在关闭）；打开对话框的按钮有名字。
    - 停用弹窗：被拒绝的原因在弹窗里（原来是提示），关闭再打开不在了；停用在路上时弹窗关不掉（Task 9 的修正轮）。
    - 包装层："找不到工作区"的界面和退出失败的提示随语言（原来写死的英文）。个人设置直接打开时侧边栏和命令面板有调用者的工作区。

**预检之后**（预检 `$M3TMP/p9-preflight/preflight.md`：0 高、4 中、10 低；控制者的裁定都接受，`$M3TMP/p9-amend-brief.md`；修订一次，之后从 `4b1334a5` 重新逐 Task 复现，附录 A.10）：

| 编号 | 预检的发现 | 落点 |
|---|---|---|
| A-M1（M1，裁定 P2 的条件） | 2.13 的三行没有"不经 `followInSession` 时失败"的测试：邀请列表、邀请页的接受和忽略、新手引导根的 `change`（PF2、PF2b、PF3c、PF19 在每一层都存活） | 邀请列表的修改移进 `useInvitationChanges`（Task 3，2.3），邀请页的回答移进 `useInvitationAnswer`（Task 4，2.4），两者照 `useMembershipChanges` 的形状，各有 `lateSettlings` 的 vitest；`onboarding/root.test.tsx` 的换账户两行扩到 `change`（Task 6）。PF3c（改角色、删除两处）、PF2、PF2b、PF19 在新代码上是 `T3.9`、`T3.10`、`T4.10`、`T4.11`、`T6.10`，都由 vitest 发现（附录 A.2） |
| A-M2（M2，裁定 P8 的缺口） | 资料一步在 `await` 之后把引导交给根，根的完成在新会话里发生，"出错了"显示在另一个账户的页面上（探查 4 次都复现） | 交接经 `followInSession`（Task 6，2.6，第 3 节第 16 条）；`steps/profile/root.test.tsx` 的换账户一行；探查的情形是变异 `T6.11`，由它发现 |
| A-M3（M3，P8b 的 P14 一类） | 对话框的限制开关发出点击时显示的开或关，快按两下停在关闭 | 开关是 `{ limitToggled: true }`，在轮到它时按 nerve 最近的回答决定（Task 8，2.8，第 3 节第 11 条）；store 的测试连按两下回到原样，W8 的页面版本经 `holdAnswer` 连按两下；变异 `T8.14`（点击时决定，W8 发现）、`T8.15`（每次都打开）；总体设计 7.7 的"轮到它时才算"加上 P9 的一句 |
| A-M4（M4） | P9 的取数文件不在 `swr` 的范围，新模块不在非空断言的范围 | 根目录 `.oxlintrc.json`，各在加入文件的 Task（2.14）；总体设计 7.7 的范围一句加上三个文件；变异 `T6.12`、`T10.4`、`T10.5`（W4）和九个 W12 |
| L1 | 两个 vitest 的标题说的比它查的多 | 改为只说它查的（不关、不提示），弹窗里的原因由 W9、行下的原因由 W4 核对（2.3、2.9，附录 A.2 的 PF7、PF20） |
| L2 | 角色的三条说明随旧的邀请一步没有读者，A.8 的检查只看字面的键 | `ROLE_DETAILS.i18n_description` 和三条文案（两种语言）删除（Task 6）；A.8 的检查加上"经常量的字段到达的键"，它的范围写在 A.8 |
| L3 | S2 的两个新测试缺最后一次"之后没有别的请求"的核对 | 补上（Task 6、10） |
| L4 | 2.13 的"不跟进的"没有列出只因卸载才安全的组件状态 | 列出，理由是裁定 P4 已核对的卸载（2.13） |
| L5（裁定 P2 的文字） | 设计还有五处写"组件先核对 `inSession()`" | 本次修订改 M3 设计 7.1、9.6、12 节 P9 的第 11 条和 P10 的第 8 条、13.x 中 Codex 4.3 第 2 条和复核 M6 两行，总体设计 7.7 的规则一句和代码一句：写明 `followInSession`，规则不变 |
| L6 | 变异表的 `T4.6` 写错 | 改为"`invitation_id` 不是 id 的链接（nerve 答 400）显示重试" |
| L7 | general 页复制地址失败什么都不说 | 提示失败，与邀请链接的复制同一句（Task 1，2.1，第 19 条）；失败的一支没有测试：要浏览器拒绝写剪贴板才走到，组件的 vitest 是服务端渲染，拿不到按钮的点击；邀请链接的复制的失败一支同样没有（两处同一句提示；修复轮由 W4 补上，2.15） |
| L8 | 第 5 节收尾一行只写了"Success!" | 按文件列出 P9 的页面中仍写死的英文（第 5 节） |
| L9 | `heldChange` 是 `fake-browser.ts` 的 `gate()` 的第二份 | 改用 `gate()`，`let settle!` 随之没有（第 15 条） |
| L10 | "`useWorkspacesFetch` 是唯一的取数"只由评审看住 | 关键词守卫的规则 `workspaces-list-fetch`（Task 10，2.14）；预检的 PF11 是变异 `T10.6`，在静态一层（关键词守卫）发现 |
| PF16 | 两种语言都留下、没有读者的键没有常设的检查 | 仍是 P8b 的 PF-L5 留下的缺口（第 5 节收尾）：A.8 加宽的一次性检查在 PF16 的树上会列出它，但常设的检查要给仓库现有的大量无读者的键定一个基线，不便宜，不在 P9 加 |
| 裁定 P3 的文档 | M2 的 store 的说明 | M2 收尾交接的"处理结果（M3/P9）"写明 `deactivateAccount` 的新签名（Task 11，2.11） |
| 裁定 P5 的文档 | W1 的故事 | M3 设计第 2 节 W1 写明大写由字段转成小写、不合格的样例是 `café`（本次修订） |
| 裁定 P8 的文档 | P11 一行 | 第 5 节 P11 一行写明四个页面的五个文件，并写明资料一步的交接是 P9 的（第 16 条） |
| 修订的连带：Task 的大小 | 以上各条让 Task 6 长到 1,687 行（约 1,500 行的上限之上） | 邀请表单的 hook 交出 nerve 建的邀请（`onSent`、`clear`，弹窗自己管离场的过渡）和 `useCopyInvitationLink` 从 Task 6 移到 Task 3：它们的第一个使用者（成员页的弹窗、邀请列表）都在 Task 3，Task 4、5 不碰这三个文件；最终的代码不变（只有 `.oxlintrc.json` 中一行的位置），变异 `T6.8` 随之是 `T3.12`。Task 6 是 1,475 行、Task 3 是 1,372 行（附录 A.6） |

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

- [ ] W1–W9 的页面版本、S1、S2 和此前的全部故事通过：worktree 中 `make e2e` 93 个全部通过（`4b1334a5` 是 75 个；W3 的会话切换在其中）；`make test-web` 通过（web 应用 73 个测试文件、640 个测试；各 Task 的修正轮之后 646 个，修复轮之后 75 个测试文件、657 个测试，附录 A.1）。
- [ ] 9.5 中 P9 的 vitest 通过：删除工作区的组件（`delete-workspace-modal.test.tsx`）、新手引导（`onboarding/root.test.tsx`、`onboarding-place.test.ts`）、邀请链接的拼法（`invitation-link.test.ts`）；每个都有一个发现它的变异，P9 写或改的每个测试都至少被一个变异发现（W18，附录 A.2）。
- [ ] 9.6 的变异核对：去掉 `followInSession` 的核对（`T1.1`），W3 的会话切换失败（附录 A.2；写进评审是控制者的）。
- [ ] W4 的成员视角：成员打开成员页，没有失败的请求，也不请求邀请（W4 的页面版本、S2）。
- [ ] `node tools/keywords.mjs` 通过：67 条规则（P9 加 `workspaces-list-fetch`，修复轮加 `refusal-toast`、`sign-out-toast`）、3 个例外，没有命中（`make lint-web`）；`tsc`、knip 通过。
- [ ] 根目录 `.oxlintrc.json` 的范围（2.14）：P9 的三个取数文件在 `no-restricted-imports`（`swr`）的范围里，P9 新建的模块和改到的测试在 `no-non-null-assertion` 的范围里；每一组各有一个变异让 `check:lint` 失败（附录 A.2）。
- [ ] 改到的文件按 7.9 没有 oxlint 警告：web 的 79 个 TS 文件都是 0 条（修复轮之后 115 个）；web 的上限 360 → 359（修正轮之后 357，修复轮之后 356），其余各包不变；没有新的抑制（附录 A.5）。
- [ ] 没有新的 `as`、`any`、`!`，第 3 节第 15 条裁定的除外（附录 A.7）。
- [ ] 逐 Task 复现：从 `4b1334a5` 照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同（附录 A.10）。
- [ ] 控制者的浏览器核对 C1–C5 写进评审（9.7；不在 plan 中）。

## 5. 不在 P9 范围内

- **P10**：项目的页面；`ProjectAuthWrapper` 中已归档项目的界面；删除项目、离开项目之后的会话核对（7.1，它们照 `followInSession` 写）；P8b spec 第 5 节 P10 一行照录。另外 P9 的清扫看到的（附录 A.4）：项目成员页的 `member-list-item.tsx:60`、`:68` 和 `project/settings/member-columns.tsx:152` 仍读 `err.error`（P8b spec 第 5 节已记）；`navigation/use-tab-preferences.ts` 的拒绝进 `console.error`。
- **P11**：M2 的账户页面在修改兑现之后自己的提示不核对会话（第 3 节第 16 条，裁定 P8），四个页面、五个文件：新手引导的资料一步 `web/apps/web/core/components/onboarding/steps/profile/root.tsx`（它的提示；它把引导交给根是 P9 的，已经经 `followInSession`）、个人设置的 general `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、preferences `web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`、security `web/apps/web/core/components/settings/profile/content/pages/security.tsx`，以及 preferences 一页的另一处 `web/apps/web/core/components/profile/start-of-week-preference.tsx`；个人主页的页面（故事 P9 的页面版本，C10）；P8a、P8b 留给 P11 的照旧（P9 没有只经机械步骤到达的文件，A.8 的清单不变）。
- **M4**：工作项的创建、修改中读 `error?.error` 的两处（`issues/issue-modal/base.tsx:233`、`:323`）；P8a、P8b 交给 M4 的照旧。
- **M6**：迭代、模块的弹窗读 `error?.error`（`cycles/transfer-issues-modal.tsx:74`、`modules/modal.tsx:73`、`:95`）；视图的修改之后的跟进。
- **M7**：已挂载的包装层换 slug 时不重取列表（P19，设计 7.5，改时改 `useWorkspaceFetch` 这一个判断）；收藏的 `favorite-items/common/helper.tsx` 的 `console.error`。
- **M8**：webhooks 的三处读 `error?.error`（`create-webhook-modal.tsx:91`、`form/secret-key.tsx:86`、`webhooks/[webhookId]/page.tsx:74`）。
- **收尾**：P9 改到的文件中仍写死的英文（Plane 原有，P9 没有改它们的文字；预检 L8，`$M3TMP/p9tools/amend/hardcoded.py` 列出，加上多行的弹窗正文）：
  - `workspace/settings/workspace-details.tsx`：修改成功的"Success!"和"Workspace updated successfully"，复制地址成功的"Workspace URL copied to the clipboard."；
  - `workspace/settings/use-invitation-changes.ts`（原在 `invitations-list-item.tsx`）：删除邀请成功的"Success!"和"Invitation removed successfully."；
  - `workspace/delete-workspace-modal.tsx`：正文"You are about to delete the workspace … Tread very carefully."、"Type in this workspace's name to continue."、"For final confirmation, type …"（P9 把表单并进弹窗，文字照旧）；
  - `navigation/project-navigation-dialog.tsx`：两种模式的说明"Feature tabs will appear as nested items under project and acts as accordion."、"Feature tabs will appear as horizontal tabs inside a project."，和"Minimum value is 1"（Task 8 重写这个对话框的修改，文字照旧）；
  - 评审另找到的（评审 M5）：`workspace/settings/member-columns.tsx` 行菜单的"Leave "、"Remove "和"Suspended"，`members/page.tsx` 的页面标题 `` `${name} - Members` ``，`workspace-details.tsx` 和 `create-workspace/page.tsx` 中图片的 `alt` 文字。

  工具的缺口（P8b 的 PF-L5，预检 PF16）：两种语言都留下、已没有读者的文案键没有常设的检查。P9 用一次性的 `i18norphans.py` 查过（附录 A.8）；修订把它加宽到经常量的字段到达的键，它在预检 PF16 的树上列出那个键，但常设的检查要先给仓库中已有的无读者的键定基线，不便宜，P9 不加，仍由收尾决定。它还有一个看不见的地方（评审 M5）：键的字面在别处因为别的原因出现时算作有读者，Task 6 留下的顶层 `"email"` 就是这样漏过的（Task 6 的修正轮删除了它）。

  角色名只有一个来源（搁置项 P6，评审 M5）：P9 的邀请页和 `invite-modal/fields.tsx` 经 `t(ROLE_DETAILS[role].i18n_title)` 随语言显示角色，其余的读者仍读只有英文的 `ROLE[...]`：`workspace/settings/member-columns.tsx`、`workspace/settings/invitations-list-item.tsx`、`add-project-members-modal.tsx`、`project/settings/member-columns.tsx` 和 `editor/embeds/mentions/user.tsx`。邀请页的标题写的角色没有核对：变异 `T4f.M3a`（标题按英文的 `ROLE` 写角色）存活（附录 A.2），W5 读英文。收尾让整个应用只有一个来源。

  失败提示的第二处（复审 M3，裁定 F-11）：`core/components/appearance/theme-switcher.tsx:56-59` 经 `setPromiseToast` 自己按 `errorMessageKey` 写失败的提示，不经 `useRefusalToast`（它只给 `setToast`），关键词规则 `refusal-toast` 也只看 `setToast`。收尾决定 hook 的模块是否另给出文案（`(error) => string`），让 `setPromiseToast` 的这一支也用它。

  P9 让它们不再成立的"将来"句（评审的清单；之前各 Phase 的 spec、plan、评审和 M2 的设计中"P9 将…"的句子是记录，不改）：`M3-design.md` 第 1176 行（`workspace/delete-workspace-form.tsx:67-75` 已删，并进 `delete-workspace-modal.tsx`）、第 1232 行（7.4：新手引导的分支现在是 `onboarding-place.ts` 的 `afterProfile` 和根的 `useWorkspacesFetch`）、第 1276 行（邀请行的复制现在是 `core/hooks/use-copy-invitation-link.ts`，与新手引导共用）；`docs/v0/plane-diff.md` 第 242 行（页面已完成，`/invitations` 页和新手引导的一步在 P8a 删除）。M3 设计 §15 中 P9 的一行由评审的提交改。

## 6. 风险

- **S2 的清单是一份要维护的名单**：P10、P11、M4–M7 每在挂载路径上加一个取数，S2 都会失败，直到清单改过（P8b 的裁定 D5，这正是它守的性质）。
- **会话切换的端到端有一个时间窗**：W3 的会话切换在 `route.fulfill()` 之后等 2 秒，断言页面没有跳转、没有提示。跳转与删除的回答在同一串 Promise 的步骤里，比这个窗口短几个数量级；`T1.1` 在每次运行中都让它失败。退路：失败时先看跟进是否被推迟到了计时器之后。
- **`sentTo` 的路由一直留着**（第 3 节第 12 条）：同一个页面上同一路径的第二次 `sentTo` 再加一层路由，各自读第一个请求体；P9 的故事中同一路径的多次修改都用新的 `sentTo`，读到的是各自的第一个（例如 W8 的五个 `PATCH`）。
- **W8 的连按两下靠 `holdAnswer` 排队**：第一下的回答被扣住时发第二下，断言两次发出的上限是 `[0, 10]`。点击与 `PATCH` 之间只有 Promise 的步骤；`T8.14`（在点击时决定）在每次运行中都让它失败（附录 A.2，反复 6 次通过）。退路：失败时先看记录的路由是否注册在 `holdAnswer` 之后（后注册的先匹配）。
- **不是 git 仓库的副本里 S3 失败**（P4b spec F4）：复现在副本中运行，只有 S3 因为读不到提交而失败，由复现脚本单独认出；worktree 中 93 个都要通过。

## 7. 已知的限制、交接和关闭条件

- **关闭**：M2 收尾第 1 节的页面一侧、第 2 节的页面一侧（交接的这两节至此关闭）；第 13 节在控制者的 C4 写进评审之后关闭；第 11 节的页面一侧在 P9 完成工作区的部分，随 P10 关闭（13.1）。P8a review 第 6 节 P9 一行、P8b review 第 6 节 P9 一行（E5）；P2、P3、P4b（P26）、P6 交给 P9 的行；M1-P2 交接的侧边栏偏好；Codex M-4、4.3 第 2 条的页面一侧。
- **交接**：第 5 节各行照录进 P10、P11 的 spec 和 M4–M8 的交接（13.2）；第 3 节中裁定的结果写进 P9 的评审。
- **放不下的关闭条件**：没有。设计 12 节 P9 的"关闭"三节都有落点；C1–C5 和评审按 brief 是控制者的。
- **限制**：每次挂载都重取，聚焦时不取（P8a 的 F-1）；已挂载的工作区包装层换 slug 时不重取列表（P19）。

## 附录 A：原型验证记录（2026-10-09）

### A.1 方法与门禁

- 原型 `$M3TMP/p9proto` 是 `4b1334a5` 的 `git archive` 副本（`pnpm install --frozen-lockfile` 之后）。每个 Task 由一个脚本从上一个快照写出（`$M3TMP/p9tools/edit/T<n>.py`），照仓库的格式化（oxfmt）排版，`tsc` 通过之后存一份快照（`$M3TMP/p9snap/T1`…`T11`）。plan 的块由快照之间的差异写出（`mkblocks.py`：11 个 Task，0 个问题；它把每个 Task 的块依次应用到上一个快照，与那个 Task 的快照逐字节比较）。代码或测试改过时，从改动的那个 Task 起重新写出之后的每个快照（`chain.sh`），再重新写出块。
- 预检之后的修订（第 3 节"预检之后"）改了 Task 1、3、4、6、8、9、10、11 的脚本，从 Task 1 起重新写出每个快照（修订之前的快照留在 `$M3TMP/p9snap-v1`）；邀请的两个 hook 移到 Task 3 之后，从 Task 3 起再写出一次（移动之前的快照 `p9snap-a1`：两者的最终树只有 `.oxlintrc.json` 中一行的位置不同，`treediff.mjs`）。
- 原型上每个 Task 写出之后 `make lint-web`、`make knip`、`make test-web` 都跑过，有页面或挂载路径改动的 Task 都跑了 `make e2e`；最后一次逐 Task 复现在每个 Task 的树上再跑四个门禁（A.10：它的每个 Task 的树与快照相同，块由 `mkblocks.py` 核对过）。`4b1334a5` 上：`make lint-web`、`make knip`、`make test-web` 通过，`make e2e` 75 个中 74 个通过（S3 在副本中失败，F4）。
- 最终原型上：`make lint-web` 通过（关键词守卫 65 条规则、3 个例外、没有命中）；`make knip` 通过；`make test-web` 通过（web 应用 73 个测试文件、640 个测试；`4b1334a5` 上是 56 个、504 个；修订之前是 70 个、617 个）；`make e2e` 93 个中 92 个通过（S3，F4）。没有 Go、接口描述和生成的文件改动，`make lint-go`、`make test`、`make gen-check` 不需要（`git diff --stat 4b1334a5 -- server/ api/ pnpm-lock.yaml` 没有输出）。
- 端到端的反复：修订之后，修订改到的五个故事 W8、S2、W1、W4、W5 各 6 次（22 个测试，132 次），都通过；修订之前 W1 的新手引导版本 3 次、W8 6 次、W9 和 A12 15 次，都通过（`sentTo` 改为不移除路由之后，第 3 节第 12 条）。

| Task | 改到的文件 | web 的测试文件 / 测试 | 端到端 |
|---|---|---|---|
| 开始 | | 56 / 504 | 75 |
| 1 | 19 | 58 / 519 | 77 |
| 2 | 14 | 59 / 531 | 79 |
| 3 | 21 | 63 / 556 | 80 |
| 4 | 13 | 66 / 582 | 82 |
| 5 | 8 | 67 / 598 | 83 |
| 6 | 23 | 70 / 619 | 88 |
| 7 | 4 | 70 / 619 | 90 |
| 8 | 15 | 72 / 635 | 91 |
| 9 | 7 | 73 / 640 | 92 |
| 10 | 12 | 73 / 640 | 93 |
| 11 | 3 | 73 / 640 | 93 |

（测试数按 vitest 自己的计数，`it.each` 的每一行算一个（`vitestcounts.py`）；一个文件可以由几个 Task 改到，所以改到的文件之和大于 110。）

上表是原型的。分支上各 Task 的修正轮之后，从 Task 5 起的测试数是 601、625、625、641、646、646、646（Task 1–4 与上表相同）；修复轮之后 web 应用 75 个测试文件、657 个测试（加 `use-refusal-toast.test.ts` 3 个、`use-sign-out.test.ts` 2 个，`in-session.test.ts` 4 个，`use-open-workspace.test.ts`、`steps/profile/root.test.tsx` 各 1 个），`make e2e` 仍是 93 个（修复轮的检查都加在已有的故事里）。

### A.2 变异

`mutants_p9.py`（Task 1、2）和 `mutants_T3.py`…`mutants_T10.py`：97 个变异，每个只改一处或几处，`mut.py` 在最终原型上、在它写的每个检查上各跑一次：97 个都被发现，没有存活的（`$M3TMP/p9tools/mut-results.json`，日志 `mut-logs-amend/`）。修订之前是 76 个；修订加 21 个：

- 预检在每一层都存活的五处，在新代码上：`T3.9`、`T3.10`（PF3c：邀请的改角色、删除不经 `followInSession`）、`T4.10`、`T4.11`（PF2、PF2b：接受、忽略）、`T6.10`（PF19：新手引导一步的修改）；预检的 M2、M3、PF11：`T6.11`（资料一步不论会话都交给根）、`T8.14`（开关在点击时决定）、`T8.15`（开关每次都打开）、`T10.6`（个人设置的侧边栏以自己的键取列表）；
- 静态检查的范围（2.14）：W4 三个（`T6.12`、`T10.4`、`T10.5`：取数换成 `swr` 的）、W12 九个（`T1.10`、`T2.9`、`T3.11`、`T4.12`、`T5.10`、`T6.13`、`T8.16`、`T9.5`、`T10.7`：在范围内的一个文件里写一个非空断言），`check:lint` 都因为那条规则失败（日志中核对过规则名）。

另外 `T3.4`、`T4.2`、`T4.3`、`T4.9` 改在移进 hook 的代码上（由 hook 的 vitest 和故事发现），`T8.3`、`T8.4` 照开关的新写法改写，`T4.6` 的说明改正（预检 L6）。Task 6 原来的 `T6.8`（弹窗发出之后关闭）随代码移到 Task 3，是 `T3.12`；Task 6 之后的编号依次前移。

每个变异另在它自己的 Task 的快照上跑过（`mutpertask.py`，`mut-T<n>.json`）：97 个都被发现，每一层都与最终原型上相同（`finalvstask.py`：0 处不同），所以 plan 的变异表中没有"（之后的 Task 起）"的标记；每个变异要改的文字在它的 Task 的快照上和最终原型上都恰好出现一次（`mutcheck.py`：0 个不适用；`T6.1`、`T6.12` 要改的文字在 Task 10 改了，变异给出两个版本的文字）。"静态"是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的范围）、关键词守卫和 `make knip`。按缺陷类别（一个变异可以被几层发现；W4、W12 是 P8 的清扫的编号：SWR 的键、类型）：

| 类别 | 变异 | 静态 | vitest | 端到端 | 存活 |
|---|---|---|---|---|---|
| W17 页面的行为照故事 | 27 | 2 | 15 | 22 | |
| W3 页面的跟进只在发出修改的会话里 | 17 | 1 | 17 | 2 | |
| 错误的读法（P8b 的第二类） | 15 | | 14 | 9 | |
| 封闭的请求体（P8b 的第一类） | 13 | 2 | 9 | 8 | |
| W12 非空断言的范围 | 9 | 9 | | | |
| W2 挂载时的取数和落点 | 7 | 1 | 3 | 5 | |
| 请求体取自所持的值（P8b 的第三类，E5） | 6 | | 5 | 4 | |
| W4 取数经 `useSessionSWR`（`no-restricted-imports` 的范围） | 3 | 3 | | | |

按发现它的层的组合：只有 vitest 33 个、vitest 和端到端 29 个、只有端到端 17 个、只有静态 14 个（W12 九个、W4 三个、`T10.6` 关键词守卫、`T7.2` 登录也带邀请的 `tsc`）、静态和端到端 3 个、静态和 vitest 1 个（`T9.3`）。按 Task：Task 1 10 个、2 9 个、3 12 个、4 12 个、5 10 个、6 13 个、7 3 个、8 16 个、9 5 个、10 7 个；Task 11 只改文档，没有变异。

- **W3 的 17 个**：`T1.1`（`followInSession` 不论会话都跟进，三个 vitest 文件和 W3 的会话切换）；离开、邀请、接受、创建、新手引导、对话框、导览各自不经核对（`T2.3`、`T3.3`、`T4.1`、`T5.2`、`T6.6`、`T8.7`、`T8.13`）；预检补上的五处和资料一步的交接（`T3.9`、`T3.10`、`T4.10`、`T4.11`、`T6.10`、`T6.11`），每个由它自己的 vitest 发现（2.13 的最后一列）；删除之后不回到根路径（`T1.8`）；停用的两个（`T9.3`：弹窗不论 store 的回答都说已停用，oxlint 的上限和弹窗的 vitest；`T9.4`：store 不论 `endSession` 的回答都说结束了，store 的 vitest）。
- **只有端到端的 17 个**：都是页面上的行为，P9 的页面在 vitest 中没有渲染它的那一部分（例如成员行的菜单、邀请列表的"已忽略"、对话框的草稿和开关、注册页之间的链接、W9 取消之后的弹窗）；每个都由它的故事的页面版本发现。`T2.8`（成员页对每个角色都取邀请）由 S2 发现：成员页的取数条件的 vitest 在 P8a（`use-members-settings-fetch.test.ts`），P9 没有改它，S2 的成员、访客两行是页面一层的核对。
- **vitest 只核对一半的两处**（预检 L1）：邀请表单的"行的拒绝"和停用弹窗的"拒绝"，vitest 只核对弹窗不关、不提示（标题照此改写）：服务端渲染看不到之后才设的错误。行下的原因由 W4、弹窗里的原因由 W9 核对：预检的 PF20（表单丢掉行的错误）、PF7（弹窗丢掉拒绝）在 vitest 一层存活，由 W4、W9 发现（预检的变异在最终原型上重跑：`pmut.py`，结果在 `$M3TMP/p9tools/pf-results-amend.json`。PF7、PF20 的 vitest 存活，由 W9、W4（和 oxlint）发现。PF2、PF2b、PF3、PF3b、PF3c、PF5 要改的代码已移进 hook，不适用：在新代码上是 `T3.9`、`T3.10`、`T4.10`、`T4.11`；邀请的改角色只收角色，发不出整个邀请。PF11 在它自己写的各层都存活，与它相同的 `T10.6` 由关键词守卫发现。PF16 存活（第 5 节收尾）。PF19 现在由 vitest 发现。PF1、PF4、PF6、PF8–PF10、PF12–PF15、PF17、PF18、PF21、PF22、PF25 照旧被发现。）。
- **W18（每个新的 vitest 都能失败）**：`mut.py` 只跑一个变异写的检查，所以另由 `w18run.py` 把每个变异对 P9 写或改的全部 22 个 vitest 文件各跑一次，`w18p9.py` 把失败的测试对照最终的 vitest 报告：P9 新加或改写的 140 个测试（`it.each` 的每一行算一个）中，97 个变异让 88 个失败。其余 52 个多是正向的行（例如"在会话里跟进"、"移出之后停在原页"、决定表中每一步的一行），另写 44 个探查变异（`mutants_w18_p9.py`，`W18.1`–`W18.44`，只在最终原型上跑，不在 plan 的表中）：成功的修改之后在核对之外提示（改角色、移出、邀请的改角色、新手引导一步的资料修改、完成引导、导航设置、导览的结束，W3）；邀请链接指向 Plane 的 `/invitations`；邀请被拒绝时的四种错法；邀请的改角色发出字符串；不核对地址的格式；nerve 不保存时停在原页；`invitationView` 每一步的错法（6 个）；`creationRefusal` 和创建的五种错法；`resumedPlace`、`afterProfile` 的六种错法；资料一步不写进资料、名字被拒绝时也交给根；停用弹窗、store 的回答反过来；`followInSession` 不跟进成功、核对模块载入时的会话；数量框、`preferencesChangeOf` 和限制开关的六种错法。44 个都被发现，140 个测试都至少失败一次（修订加 `W18.41`–`W18.44`，给修订的测试中没有变异让它失败的四个；`W18.37`、`W18.38` 照开关的新写法改写）。改到的五个 P8 的测试文件中 P8 原有的 23 个测试不算（其中 2 个也被 P9 的变异发现）。
- **例外**（只由评审或之后的 Task 才能发现的性质）：没有。预检的 PF16（两种语言都留下的无读者的键）仍只由一次性的检查看到（A.8，第 5 节收尾）。
- **修正轮和修复轮**（裁定 F-7，搁置项 P5）：执行中各 Task 的探查和修正轮的变异（`T3.13`–`T3.14`；Task 4 的 `T4f.*` 八个；`T5f.1`–`T5f.5`；`T6f.1`–`T6f.4`、`T6p.1`–`T6p.2`；`T7p.1`、`T7f.1`–`T7f.2`；`T8p.2`–`T8p.3` 和原来的 `T8.4`（`T8.4-old`）；`T9.6`–`T9.7`、`T9p.1`–`T9p.5`；`T10p.1`–`T10p.4`、`T10f.1`–`T10f.3`）收进各 Task 的表，修复轮自己的 27 个和复审之后的补充（裁定 F-11）的 3 个（`M3.c`、`P12.f`：两个 hook 的测试中原来没有发现任何变异的两行；`I1.f`：邀请弹窗的 Escape）在 `mutants_FW.py`；之后的改动挪了位置的，变异给出各版本的文字（改到别的文件时连同文件）。全表 165 个在修复轮的树上各跑一次（`mut-results-fw.json`）：163 个被发现；`T8.4-old` 是等价的（它在轮到修改时才读显示的设置，那时与 store 所持的相同；修正轮改写的 `T8.4` 由 vitest 和 W8 发现）；`T4f.M3a`（邀请页的标题按英文的 `ROLE` 写角色）存活，Task 4 的修正轮已记下：邀请页的标题没有渲染的测试、W5 读英文（第 5 节收尾的角色名一项）。补充的 3 个，和 `closedByEscape` 改过之后（2.15 的 F-1 一项）的 `I1.d`、`I1.e`、`T9.7`，在补充的树上再跑（同一个结果文件）：都被发现，全表 168 个中 166 个被发现，存活和等价的仍是上面两个。每个变异的层、发现它的检查和结果在 `$M3TMP/p9tools/p9-mutation-record.md`。

### A.3 W2：挂载清单（真实运行）

S2 的清单在 Task 1、2、6、10 的树上加入，每次 `make e2e` 都核对（2.12 的表；S2 从 7 个测试到 9 个：新人、个人设置各一个，general 页和成员页的行在每个账户已有的测试中；新的两个像其余的一样，在安静的控制台之后再读一次请求，之后没有别的请求，预检 L3）。之前与之后：

- `4b1334a5` 的 S2：四个账户各打开 `/`（`APP` + `WORKSPACE`）、项目设置；管理员另打开项目页（`ARCHIVED`）。
- P9 加的行：`/{slug}/settings`，管理员、成员、不是项目成员的成员是 `APP` + `WORKSPACE` + `GENERAL`，访客是 `APP` + `WORKSPACE`；`/{slug}/settings/members`，管理员是 `APP` + `WORKSPACE` + `INVITATIONS`，其余三个是 `APP` + `WORKSPACE`。这两页的取数 P9 没有改（第 3 节第 18 条），在 `4b1334a5` 上相同。
- 新人打开 `/`：`4b1334a5` 上落点不为新手引导取列表，新手引导读一份没人取过的列表（P8a review 的 M7）；Task 6 起是 `APP`（列表一次：新手引导和落点同一个键）。
- 直接打开个人设置：`4b1334a5` 上没有 `GET /api/v0/workspaces`，侧边栏没有工作区；Task 10 起是 `APP`，侧边栏列出 Acme（S2 断言）。
- 每张清单没有 M6、M7 的地址，没有 `/api/v0` 之外的请求，没有失败的请求。故事的页面版本（W1–W9）都经 `watchPage`：每个故事断言记下的失败的请求恰好是它要的那几个（没有，或者例如 W3 的 403、W8 的 422、W9 的 409），没有旧地址的请求、没有页面错误，控制台只有这些失败的报告（`expectQuietConsole`）。

### A.4 缺陷类的清扫

- **封闭的请求体**（P8b 的第一类）：P9 的页面发出的每个请求体都由表单编辑的字段构成，类型是生成的 `…Create`、`…Update`，角色是数字，每个由一个测试钉住：general 页的 `{ name, organization_size?, timezone }`（`workspace-details.test.tsx`、W3 的 `sentTo`）；改角色 `{ role }`（`use-membership-changes.test.ts`、W7）；批量邀请 `{ invitations: [{ email, role }] }`（`invite-modal.test.tsx`、W4）；邀请的改角色 `{ role }`（`use-invitation-changes.test.ts`、W4）；接受、忽略（W5）；创建工作区（`use-create-workspace.test.ts`、W1）；写上次打开的工作区（`use-open-workspace.test.ts`、`onboarding/root.test.tsx`、W1、W2、W5）；新手引导的步骤和完成（`root.test.tsx`、W1）；导航设置（`navigation-preferences.test.ts`、W8）；导览的结束（`home/root.test.tsx`、W8）；注册（W6）。13 个变异，都被发现（A.2）。
- **错误的读法**（第二类）：P9 改到的页面的每一条出错的路径都经 `errorMessageKey`（提示、弹窗）或 `FIELD_ERROR_MESSAGES`（字段、行下），没有吞进 `console.error` 的拒绝，被拒绝的修改结束加载。P9 删掉的 `console.error`：`invitations-list-item.tsx`（复制）、`workspace-details.tsx`、`home/root.tsx`、邀请页。仍读 `err.error` 的都不在 P9 的页面中：项目成员页（P10）、webhooks（M8）、迭代和模块（M6）、工作项（M4）（第 5 节）。15 个变异。
- **请求体取自所持的值**（第三类）：P9 的页面中只有导航设置是"按显示的值算请求体"的形状（E5），Task 8 把它交给 store 在轮到它时算；限制的开关也是：每一下在轮到它时按 nerve 最近的回答打开或关闭（预检 M3：原来发出点击时显示的开或关）；general 页、角色、邀请发的都是这次编辑的值，不取 store 所持的；对话框的数量是草稿。6 个变异（`T8.14` 是页面上的这一类，由 W8 的连按两下发现）。
- **会话外的跟进**（W3）：`followsweep.py` 列出 web 应用中提示或跳转的每个文件和它们等待的 store 调用，逐个读过：工作区一侧的都在 2.13 的表中；M2 的账户页面自己的提示在第 3 节第 16 条；退出登录失败的提示五处（工作区菜单、包装层等）不是这一类（失败时会话没有变）。预检之后：2.13 的每一行都有一个"那一处不经 `followInSession` 时失败"的 vitest（表的最后一列；预检看到三处没有，A-M1 补上：`T3.9`、`T3.10`、`T4.10`、`T4.11`、`T6.10`）；资料一步把引导交给根也经 `followInSession`（A-M2，`T6.11`）；只因卸载才安全的组件状态列在 2.13 的"不跟进的"（L4）。
- **19（每个新的 store 或 hook 方法有失败测试）**：`followInSession`、`useMembershipChanges` 的三个修改、`useInvitationChanges` 的两个、`useInvitationAnswer` 的两个、`useCreateWorkspace`、`useOpenWorkspace`、`useProjectNavigationPreferences`、新手引导的根、资料一步、首页、停用弹窗、`updatePreferences` 都有被拒绝的测试；`invitationView`、`creationRefusal`、`invitationRefusal` 的表中各有没有回答的失败一行。
- **36（另一条路径到达同一结果）**：同一个修改的两个入口共用一个实现：创建工作区（`/create-workspace` 和新手引导，`useCreateWorkspace`）、邀请表单（成员页的弹窗和新手引导，`useWorkspaceInvitationActions`）、复制邀请链接（邀请列表和新手引导，`useCopyInvitationLink`）、工作区列表的取数（四处，`useWorkspacesFetch`）、打开刚加入的工作区（接受、创建，`useOpenWorkspace`）。新手引导的邀请一步和成员页由同一组测试守着（`invite-modal.test.tsx` 测共用的 hook；W1、W4 两个故事各走一个入口）。
- **45（没有不能失败的断言）**：W18（A.2）：140 个测试都至少被一个变异或探查变异发现；"换账户之后什么都不做"的 `is answered` 行由"在核对之外跟进"的探查变异发现。
- **46（每个等待有期限）**：vitest 中等待页面跟进的是 `pageSettled()`（一个计时器的轮次）和 `vi.waitFor`（vitest 的期限）；端到端的等待是 Playwright 的 `expect`、`expect.poll` 和 `waitForResponse`，都有期限。
- **50（说明与代码一致）**：每个 Task 结束时，新写或改动的注释、JSDoc 与它描述的代码或测试逐句核对；最终原型上 P9 加或改的注释（`addedcomments.py`，约 1,000 行）再全部读过一遍；README 和前端改动清单的行与最终的代码核对（Task 11）。

### A.5 oxlint（7.9，R3）

- 有手改的 TS 文件：web 应用 76 个（含 22 个测试）、`@nerve/types` 的 `navigation-preferences.ts`、`@nerve/utils` 的 `validation.ts`、`@nerve/constants` 的 `workspace.ts`，都是 0 条警告（`oxfinal.sh`）。P9 没有机械步骤，没有只经机械步骤到达的文件。`4b1334a5` 上这些文件共有 1 条（`use-workspace-invitation.tsx`，Task 3 清零；各包的上限要求警告数等于上限，web 的只降了 1，其余不变）。
- 上限：web 应用 360 → 359（Task 3）→ 357（Task 4 的修正轮：`empty-space.tsx` 清零）→ 356（修复轮：`confirm-workspace-member-remove.tsx` 的正的 `tabIndex`）；`utils` 7、`types` 0、`constants` 1、`propel` 16、`ui` 19 不变。修复轮之后 P9 有手改的 web 的 TS 文件 115 个（其中测试 30 个）都是 0 条警告。
- 抑制：没有新的抑制；删去两处（Task 6：新手引导的根原来以 `useEffect` 决定第一步，抑制 `react-hooks/exhaustive-deps`；Task 8：对话框的 `unicorn/consistent-function-scoping`），188 → 186（`suppressions.py`）。
- 根目录 `.oxlintrc.json` 的两条错误级的范围（2.14）：`no-restricted-imports` 加 3 个文件，`typescript/no-non-null-assertion` 加 P9 的新模块和测试，各在加入它们的 Task；每组一个变异（A.2 的 W4、W12）。
- 关键词守卫：65 条规则、3 个例外，每个 Task 都没有命中；P9 加一条 `workspaces-list-fetch`（Task 10，2.14），没有删掉旧地址（P8a、P8b 已删完）。修复轮加 `refusal-toast`、`sign-out-toast`（2.14、2.15）：67 条规则、3 个例外，没有命中。

### A.6 规模

- **plan** 共 9,105 行：Task 1 之前（约束、文件结构）117 行；各 Task 一节（修订之前共 8,520 行）：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 会话里的跟进；general 页、删除；W3 | 1,277 | 1,136 | 19 |
| 2 成员页；W7、W2 | 941 | 824 | 14 |
| 3 邀请的表单和列表；W4 | 1,372 | 1,223 | 21 |
| 4 邀请页；W5 | 937 | 816 | 13 |
| 5 创建工作区；W1（一） | 780 | 673 | 8 |
| 6 新手引导；W1（二） | 1,475 | 1,317 | 23 |
| 7 带邀请的注册；W6 | 266 | 195 | 4 |
| 8 导航设置、对话框、导览；W8 | 1,045 | 911 | 15 |
| 9 停用弹窗；W9 | 401 | 309 | 7 |
| 10 列表的取数、包装层的界面 | 387 | 279 | 12 |
| 11 文档 | 108 | 49 | 3 |

- **文件的行数**（约 400 行）：P9 改到的文件中最长的是 `e2e/stories/workspace/w3-workspace-settings.spec.ts` 395 行、`e2e/fixtures/assert/workspace.ts` 390 行、S2 379 行（三个都是原有的故事和 fixture，P9 加了页面版本和行）；新文件中最长的是 `use-create-workspace.test.ts` 165 行、`e2e/fixtures/workspace-pages.ts` 162 行。有手改的代码文件没有一个结束时超过 400 行；超过的只有 `tools/keywords.json`（2,209 → 2,241 行，关键词守卫的规则表，P9 加一条，2.14）。修正轮和修复轮之后：W1 403 行（修复轮在它的五个页面版本中加了忙碌、草稿和 Bob 的页面的核对，约 400 行）、W3 397 行、`assert/workspace.ts` 390 行、S2 383 行；新文件中 `use-create-workspace.test.ts` 189 行、`workspace-pages.ts` 184 行；`tools/keywords.json` 2,241 → 2,307 行（修复轮的两条规则和它们的样例）。
- **变长的 Plane 文件**（brief：只为使用方改到的不变长）：17 个变长，16 个是本 Phase 的对象（行为改在它们里面：邀请页 95 → 153、删除弹窗 30 → 179（表单并入）、`auth-root.tsx` 35 → 39、`password.tsx` 284 → 287、停用弹窗 93 → 102、`auth-screens/header.tsx` 81 → 83、`home/root.tsx` 47 → 53、新手引导的资料一步 147 → 156（交接经 `followInSession`，预检 M2）、`onboarding/steps/root.tsx` 57 → 62、`workspace-options.tsx` 49 → 53、`sidebar-wrapper.tsx` 85 → 88、`invite-modal/fields.tsx` 120 → 122、`members/invite-modal.tsx` 57 → 67、`workspace-details.tsx` 251 → 267（复制失败的提示，预检 L7）、`use-workspace-invitation.tsx` 91 → 101、`store/user/index.ts` 199 → 200）；只为使用方改到的是 `members-list.tsx` 89 → 91（第 3 节第 6 条）。其余有手改的 Plane 文件变短或不变（例如新手引导的邀请一步 396 → 80、邀请列表项 196 → 159、`create-workspace-form.tsx` 236 → 193、`validation.ts` 120 → 83）（`planegrow.py`）。修复轮和补充（`dba0d756` 对补充之后的树，`git diff --numstat`）又让 11 个 Plane 文件变长，行为都改在它们里面，没有只为使用方改到的：F-1 的删除弹窗 179 → 182、`confirm-workspace-member-remove.tsx` 91 → 98、`invite-modal/actions.tsx` 71 → 72；F-2 的新手引导邀请一步 80 → 89、`onboarding/steps/root.tsx` 65 → 68；F-4 的邀请页 172 → 174（退出失败的一支）、`use-workspace-invitation.tsx` 101 → 104、个人设置的 general 表单 278 → 279、API 令牌的表单 234 → 235；F-5 的 `empty-space.tsx` 67 → 68；F-8 的 `authentication-wrapper.tsx` 90 → 93（说明）。

### A.7 W12：类型断言

- `w12.cjs`（TypeScript 的语法树，`4b1334a5` 对最终原型）：P9 改到的 59 个源文件（测试除外）中，`as`（不含 `as const`）7 → 0、`any` 2 → 0、非空断言 0 → 0。去掉的：`member-columns.tsx` 2 个、`members-list-item.tsx` 2 个、`invitations-list-item.tsx` 2 个、成员页 1 个（`as`）；新手引导的邀请一步 2 个 `any`。
- 新的 `as const` 三个（第 3 节第 15 条，裁定 P7）：`in-session.ts` 的 `"done" as const`、`"failed" as const`；`invitation-link.test.ts` 的 `["/", "/sign-up"] as const`（`asconst.py`）。裁定时 `fake-tab.ts` 还有一个 `let settle!: …`，修订让 `heldChange` 用 `gate()`，它没有了（预检 L9）。`casts.py` 逐个 Task 核对，最后对整个 Phase 的 91 个 TS 文件（测试和 e2e 在内）再核对一次：没有新的 `as`、`any`、`!` 或 `oxlint-disable`（它列出的都是字符串里的英文 "as"、`!confirmed(…)` 的取反和 `tokenManager as tab` 的导入别名）。非空断言另有 oxlint 看住：P9 的新模块和测试都在 `typescript/no-non-null-assertion` 的范围里（2.14）。修复轮在 e2e 写了三个新的 `let …!:`（`holdScripts` 两个、W1 一个，各是一份先建 `Promise` 再取它的 `resolve` 的写法；复审 M1）；补充（裁定 F-11）把它们和 e2e 中此前同样写的三处（`holdAnswer`、A9、A12）都改为 `deferred()`（第 2 节的共用部分），e2e 的 fixture 和故事中不再有 `let …!:`。
- 没有手写的、重述契约的类型：表单的值是 `WorkspaceInvitationsCreate`、`CreationForm`（`Pick<WorkspaceCreate, …>` 加可空的规模）、`Pick<Workspace, …>`；knip 没有未使用的导出。

### A.8 文案

两种语言相同，从快照逐个量出（`i18nadded.py`，`en`；`check:sync` 核对 `zh-CN` 的键相同）：加 26 条、删 11 条、改写 5 条。

| Task | 文件 | 加 | 删 | 改写 |
|---|---|---|---|---|
| 1 | `workspace-settings.json` | | `delete_modal.error_message` | `delete_modal.success_message` |
| 3 | `workspace-settings.json` | `members.declined`、`members.modal.errors.already_member`、`already_invited` | | |
| 4 | `workspace.json` | `workspace_invitation.*` 14 条 | | |
| 5 | `workspace.json` | `workspace_creation.errors.validation.url_reserved` | `errors.creation_disabled.request_button`、`request_email.subject`、`request_email.body` | `errors.creation_disabled.title`、`description`，`errors.validation.url_alphanumeric` |
| 6 | `common.json`、`workspace.json` | `onboarding.invite.links.title`、`description` | `onboarding.invite.role`、`onboarding.invite.not_an_email`、`role_details.{guest,member,admin}.description`（预检 L2）、`workspace_creation.toast.error.title`、`message` | `onboarding.workspace.creation_disabled` |
| 8 | `accessibility.json` | `aria_labels.projects_sidebar.project_navigation` | | |
| 10 | `workspace.json` | `workspace_not_found.*` 5 条 | | |

- **删除是完整的**（P8b 的 PF-L5：`check:sync` 看不见两种语言都留下的键）：`i18norphans.py` 列出 `4b1334a5` 上有读者、给定的树上没有读者、文案文件中仍在的键。一个键有读者：有源文件写出它的全名；修订加宽（预检 L2）：全名只写在导出的常量的字段里时（例如 `ROLE_DETAILS` 的 `i18n_description: "role_details.admin.description"`），要有写出这个常量的源文件也读这个字段（`.i18n_description`）。结果：
  - 修订之前的最终树（`p9snap-v1/T11`）上列出 `role_details.{admin,guest,member}.description` 三条：唯一读 `i18n_description` 的是旧的邀请一步，Task 6 删了它；修订在 Task 6 删除这三条和那个字段。
  - 最终原型上 0 条（原来只看全名的检查在原型中列出过 `delete_modal.error_message`，Task 1 删除了它，第 3 节第 17 条）。
  - 预检的 PF16（把它放回两种语言）的树上列出它。

  它的范围：经模板字符串拼出键名的读者两边都看不见；常量换了名字传下去、在不写出常量名的文件里读字段时，算作没有读者。它是一次性的检查，不是门禁（第 5 节收尾）。
- 改写的文案都与 nerve 一致：slug 的格式是 nerve 的规则（小写字母、数字、`-`、`_`）；创建关闭说请工作区的管理员给邀请链接（两处相同）；删除工作区之后去另一个工作区或创建。

### A.9 Phase 的大小

写 plan 之前的核对（brief）：设计的 14 个任务中，12、13（端到端）并进各页面的 Task（测试与代码同一个提交），14 只改文档，估出约 10 个 Task、约 9,000–11,000 行，最长的约 1,100 行：在约 16 个、约 1,500 行之内，不拆。原型中实测：11 个 Task（停用弹窗单独成为 Task 9，它的端到端和 store 的改动比估计的多），修订之前 8,520 行、最长的 Task 6 是 1,475 行。修订加了约 590 行，Task 6 长到 1,687 行；邀请的两个 hook 移到 Task 3 之后（第 3 节"预检之后"），共 9,105 行，最长的 Task 6 是 1,475 行，Task 3 是 1,372 行，其余都在 1,277 行之内。

### A.10 逐 Task 复现

`replay.py` 在 `4b1334a5` 的 `git archive` 副本上，换进修订之后的四份文档（M3 设计、总体设计、本 spec 和 plan，`amend/mkbase.py`；`$M3TMP/p9replay`，`pnpm install --frozen-lockfile` 之后）照 plan 逐个 Task 执行：写入块（`planapply.mjs`），核对 Go 的模块文件不变、`pnpm-lock.yaml` 与那个 Task 的快照相同，再执行这个 Task 的每个 `Run:`。副本不是 git 仓库，所以 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（它列文件不用 git，规则和样例相同），`make e2e` 只有 S3 因为读不到提交而失败（F4），由脚本单独认出。

| Task | 写入的文件 | 检查 |
|---|---|---|
| 1 | 19 | `make lint-web`、`make knip`、`make test-web` 通过；`make e2e` 76 个通过（S3 除外） |
| 2–5 | 14、21、13、8 | 同 Task 1，`make e2e` 78、79、81、82 个通过 |
| 6–10 | 23、4、15、7、12 | 同 Task 1，`make e2e` 87、89、90、91、92 个通过 |
| 11 | 3 | 同 Task 10 |

这一次复现用的 plan 就是本 spec 同一个提交中的 plan（9,105 行，SHA-256 `26c1df542acabfec…`），11 个 Task 都通过（`$M3TMP/p9tools/replay-logs/summary.txt`）。最后的副本与原型逐文件相同，只差这个提交的四份文档：`treediff.mjs p9replay p9snap/T11`：3,173 个文件对 3,171 个，4 处差异，都是这四份文档（快照里没有它们的修订）。修订之前的复现（第五次，plan 8,520 行）同样 11 个 Task 都通过，副本与当时的原型 0 处差异（`replay-logs-v5`）；更早的四次在 plan 改过时停下，不作为证据。本 spec 的附录在复现之后补上了复现、W18 和预检的变异的结果，plan 没有再改。

