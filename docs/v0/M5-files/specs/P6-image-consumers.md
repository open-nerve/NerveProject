# P6 图片消费者清单（预检基线）

基线 `c59cedf337c5c5654bef2f0cbb33407ff04ced26` 的 49 文件、78 处源码搜索清单。来源映射已逐项补齐，另核 7 处透传调用链；源码 SHA 与原盘点一致。核对方式是相关 props/hooks、调用者搜索及关键透传组件全文阅读，没有声称全部 49 文件全文审读。实施各 Task 仍须完整阅读实际编辑文件；新增调用者不得遗漏。


## 目标来源与任务映射

下表描述本阶段要实现的来源，不表示既有裸 URL 已具备这些权限和生命周期保证。路径相对仓库根。

| 路径 | 目标来源 | 身份/范围 | Task |
| --- | --- | --- | --- |
| `web/apps/web/core/components/account/auth-forms/auth-header.tsx` | public-preview | 邀请 link key 与该预览响应的 workspace_logo_expires_at | T10 |
| `web/apps/web/core/components/comments/card/display.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/common/applied-filters/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/common/filters/created-by.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/common/switcher-label.tsx` | dead-url-branch | 无资料来源；移除未使用的 logo_url prop/分支 | T10 |
| `web/apps/web/core/components/core/description-versions/dropdown-item.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/core/description-versions/modal.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/core/sidebar/progress-stats/assignee.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/cycles/active-cycle/cycle-stats.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/cycles/progress-sidebar/progress-stats.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/cycles/progress-sidebar/sidebar-details.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/dropdowns/member/avatar.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/dropdowns/member/base.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/editor/embeds/mentions/user.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx` | self | 当前登录 RootStore 的个人头像/封面 | T5 |
| `web/apps/web/core/components/inbox/inbox-filter/applied-filters/member.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/inbox/inbox-filter/filters/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/issues/issue-layouts/filters/header/filters/assignee.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/issues/issue-layouts/group-columns.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/modules/applied-filters/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/modules/dropdowns/filters/lead.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/modules/dropdowns/filters/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/modules/progress-sidebar/progress-stats.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/onboarding/steps/profile/root.tsx` | self | 当前登录 RootStore 的个人头像/封面 | T5 |
| `web/apps/web/core/components/onboarding/switch-account-dropdown.tsx` | self | 当前登录 RootStore 的个人头像/封面 | T5 |
| `web/apps/web/core/components/power-k/menus/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/power-k/menus/workspaces.tsx` | workspace | 传入工作区 DTO 的 UUID，或由明确 slug 解析当前规范工作区 UUID | T6 |
| `web/apps/web/core/components/profile/sidebar.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/project/applied-filters/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/project/card.tsx` | project | project.workspace_id + project.id | T9 |
| `web/apps/web/core/components/project/dropdowns/filters/members.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/project/form.tsx` | project | project.workspace_id + project.id | T9 |
| `web/apps/web/core/components/project/member-select.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/project/settings/member-columns.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx` | self | 当前登录 RootStore 的个人头像/封面 | T5 |
| `web/apps/web/core/components/settings/profile/sidebar/header.tsx` | self | 当前登录 RootStore 的个人头像/封面 | T5 |
| `web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx` | workspace | 传入工作区 DTO 的 UUID，或由明确 slug 解析当前规范工作区 UUID | T6 |
| `web/apps/web/core/components/settings/workspace/sidebar/header.tsx` | workspace | 传入工作区 DTO 的 UUID，或由明确 slug 解析当前规范工作区 UUID | T6 |
| `web/apps/web/core/components/workspace/settings/member-columns.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/components/workspace/settings/workspace-details.tsx` | workspace | 传入工作区 DTO 的 UUID，或由明确 slug 解析当前规范工作区 UUID | T6 |
| `web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx` | workspace | 传入工作区 DTO 的 UUID，或由明确 slug 解析当前规范工作区 UUID | T6 |
| `web/apps/web/core/components/workspace/sidebar/user-menu-root.tsx` | self | 当前登录 RootStore 的个人头像/封面 | T5 |
| `web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx` | workspace | 传入工作区 DTO 的 UUID，或由明确 slug 解析当前规范工作区 UUID | T6 |
| `web/apps/web/core/components/workspace-notifications/sidebar/notification-card/item.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/hooks/editor/use-editor-mention.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx` | member | 明确宿主 workspace UUID + member/user/author ID | T10 |
| `web/apps/web/helpers/cover-image.helper.ts` | static-helper | 静态缺省资源与旧呈现辅助函数 | T9 |

### 透传调用链补充

| 路径 | Task | 处理边界 |
| --- | --- | --- |
| `web/apps/web/core/components/common/cover-image.tsx` | T5/T8/T9 | 五处实际调用都是个人或项目封面；迁移时核最后消费者，再删无用裸URL辅助通路。 |
| `web/apps/web/core/components/workspace/logo.tsx` | T6/T10 | 登录 workspace source 与公开邀请预览使用明确分支，禁止可选裸URL兜底绕过合同。 |
| `web/apps/web/core/components/project/create/header.tsx` | T8 | 新建项目本地选择/准备件预览，scope=new_project；不是已有 project source。 |
| `web/apps/web/core/components/core/description-versions/dropdown.tsx` | T10 | 把根组件的明确 workspace UUID 传给每个作者头像条目。 |
| `web/apps/web/core/components/core/description-versions/root.tsx` | T10 | 从既有 workspaceSlug 解析规范工作区 UUID，沿下拉链路传递。 |
| `web/apps/web/core/components/breadcrumbs/project.tsx` | T10 | SwitcherLabel 的图标通路调用者；核无 logo_url 输入。 |
| `web/apps/web/core/components/navigation/project-header.tsx` | T10 | SwitcherLabel 的另一个图标通路调用者；核无 logo_url 输入。 |

公共邀请保持独立预览生命周期；成员图不从跨工作区 memberMap 的裸URL授权。M6/M7只换既有头像叶子和投影；未接通领域页面验收归属保持。所有目标读取和上传失效仍沿已批准的PC1和IssueRootStore，不新增权限源。

## 基线原始命中

## web/apps/web/core/components/account/auth-forms/auth-header.tsx

- L50: `logo={current.workspace_logo_url}`

## web/apps/web/core/components/comments/card/display.tsx

- L68: `const avatarUrl = userDetails?.avatar_url ?? null;`

## web/apps/web/core/components/common/applied-filters/members.tsx

- L41: `src={getFileURL(memberDetails.avatar_url)}`

## web/apps/web/core/components/common/filters/created-by.tsx

- L83: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/common/switcher-label.tsx

主控核对：除组件内部透传外，全 core 无 `logo_url=` 调用方；正式T10应删除无消费者的可选prop和裸URL分支，不为死分支造图片resource。图标logo_props路径保留。

- L15: `logo_url?: string;`
- L21: `export function SwitcherIcon({ logo_props, logo_url, LabelIcon, size = 12, type = "lucide" }: TSwitcherIconProps) {`
- L26: `if (logo_url) {`
- L29: `src={getFileURL(logo_url)}`
- L41: `logo_url?: string;`
- L48: `const { logo_props, name, LabelIcon, logo_url, type = "lucide" } = props;`
- L51: `<SwitcherIcon logo_props={logo_props} logo_url={logo_url} LabelIcon={LabelIcon} type={type} />`

## web/apps/web/core/components/core/description-versions/dropdown-item.tsx

- L38: `src={getFileURL(versionCreator?.avatar_url ?? "")}`

## web/apps/web/core/components/core/description-versions/modal.tsx

- L145: `src={getFileURL(versionCreator?.avatar_url ?? "")}`

## web/apps/web/core/components/core/sidebar/progress-stats/assignee.tsx

- L21: `avatar_url: string | undefined;`
- L46: `src={getFileURL(assignee?.avatar_url ?? "")}`

## web/apps/web/core/components/cycles/active-cycle/cycle-stats.tsx

- L129: `src={getFileURL(assignee?.avatar_url ?? "")}`

## web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx

- L246: `src={getFileURL(member?.avatar_url ?? "")}`

## web/apps/web/core/components/cycles/progress-sidebar/progress-stats.tsx

- L58: `avatar_url: assignee?.avatar_url || undefined,`

## web/apps/web/core/components/cycles/progress-sidebar/sidebar-details.tsx

- L62: `src={getFileURL(cycleOwnerDetails?.avatar_url ?? "")}`
- L87: `src={getFileURL(memberDetails?.avatar_url ?? "")}`

## web/apps/web/core/components/dropdowns/member/avatar.tsx

- L37: `src={getFileURL(userDetails.avatar_url)}`
- L50: `src={getFileURL(userDetails?.avatar_url ?? "")}`

## web/apps/web/core/components/dropdowns/member/base.tsx

- L136: `src={getFileURL(userDetails?.avatar_url ?? "")}`

## web/apps/web/core/components/editor/embeds/mentions/user.tsx

- L69: `src={getFileURL(userDetails?.avatar_url ?? "")}`

## web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx

- L91: `currentUser?.avatar_url && currentUser?.avatar_url.trim() !== "" ? (`
- L95: `src={getFileURL(currentUser?.avatar_url)}`

## web/apps/web/core/components/inbox/inbox-filter/applied-filters/member.tsx

- L55: `src={getFileURL(optionDetail.avatar_url)}`

## web/apps/web/core/components/inbox/inbox-filter/filters/members.tsx

- L91: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/issues/issue-layouts/filters/header/filters/assignee.tsx

- L83: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/issues/issue-layouts/group-columns.tsx

- L156: `src={getFileURL(member.avatar_url ?? "")}`

## web/apps/web/core/components/modules/applied-filters/members.tsx

- L40: `src={getFileURL(memberDetails.avatar_url)}`

## web/apps/web/core/components/modules/dropdowns/filters/lead.tsx

- L83: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/modules/dropdowns/filters/members.tsx

- L83: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/modules/progress-sidebar/progress-stats.tsx

- L57: `avatar_url: assignee?.avatar_url || undefined,`

## web/apps/web/core/components/onboarding/steps/profile/root.tsx

- L91: `{user?.avatar_url ? (`
- L93: `src={getFileURL(user.avatar_url)}`

## web/apps/web/core/components/onboarding/switch-account-dropdown.tsx

- L44: `{user?.avatar_url ? (`
- L46: `src={getFileURL(user?.avatar_url)}`

## web/apps/web/core/components/power-k/menus/members.tsx

- L41: `src={getFileURL(memberDetails?.avatar_url ?? "")}`

## web/apps/web/core/components/power-k/menus/workspaces.tsx

- L26: `<WorkspaceLogo logo={workspace.logo_url} name={workspace.name} classNames="shrink-0" />`

## web/apps/web/core/components/profile/sidebar.tsx

- L84: `{userData.avatar_url && userData.avatar_url !== "" ? (`
- L86: `src={getFileURL(userData.avatar_url)}`

## web/apps/web/core/components/project/add-project-members-modal.tsx

- L127: `src={getFileURL(memberDetails?.member.avatar_url)}`
- L181: `src={getFileURL(selectedMember?.member.avatar_url ?? "")}`

## web/apps/web/core/components/project/applied-filters/members.tsx

- L42: `src={getFileURL(memberDetails.avatar_url)}`

## web/apps/web/core/components/project/card.tsx

- L176: `src={project.cover_image_url ?? undefined}`
- L247: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/project/dropdowns/filters/members.tsx

- L88: `src={getFileURL(member.avatar_url)}`

## web/apps/web/core/components/project/form.tsx

- L73: `src={project.cover_image_url ?? undefined}`

## web/apps/web/core/components/project/member-select.tsx

- L54: `src={getFileURL(memberDetails?.member.avatar_url)}`
- L74: `src={getFileURL(selectedOption.member?.avatar_url)}`

## web/apps/web/core/components/project/settings/member-columns.tsx

- L43: `const { avatar_url, display_name, email, first_name, id, last_name } = rowData.member;`
- L51: `{avatar_url && avatar_url.trim() !== "" ? (`
- L55: `src={getFileURL(avatar_url)}`

## web/apps/web/core/components/settings/profile/content/pages/general/form.tsx

- L110: `src={user.cover_image_url ?? undefined}`
- L117: `{user.avatar_url ? (`
- L120: `src={getFileURL(user.avatar_url)}`

## web/apps/web/core/components/settings/profile/sidebar/header.tsx

- L24: `src={getFileURL(currentUser?.avatar_url ?? "")}`

## web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx

- L36: `iconNode={<WorkspaceLogo logo={workspace.logo_url} name={workspace.name} classNames="shrink-0" />}`

## web/apps/web/core/components/settings/workspace/sidebar/header.tsx

- L48: `logo={workspace?.logo_url}`

## web/apps/web/core/components/workspace/settings/member-columns.tsx

- L44: `const { avatar_url, display_name, email, first_name, id, last_name } = rowData.member;`
- L57: `) : avatar_url && avatar_url.trim() !== "" ? (`
- L61: `src={getFileURL(avatar_url)}`

## web/apps/web/core/components/workspace/settings/workspace-details.tsx

- L125: `{workspace.logo_url ? (`
- L128: `src={getFileURL(workspace.logo_url)}`

## web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx

- L55: `!workspace?.logo_url && "rounded-md bg-[#026292] text-on-color"`
- L58: `{workspace?.logo_url && workspace.logo_url !== "" ? (`
- L60: `src={getFileURL(workspace.logo_url)}`

## web/apps/web/core/components/workspace/sidebar/user-menu-root.tsx

- L54: `src={getFileURL(currentUser?.avatar_url ?? "")}`
- L71: `src={currentUser?.cover_image_url ?? undefined}`
- L83: `src={getFileURL(currentUser?.avatar_url ?? "")}`

## web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx

- L96: `logo={activeWorkspace?.logo_url}`

## web/apps/web/core/components/workspace-notifications/sidebar/notification-card/item.tsx

- L94: `src={getFileURL(notificationTriggeredBy.avatar_url)}`

## web/apps/web/core/hooks/editor/use-editor-mention.tsx

- L85: `src={getFileURL(user.avatar_url)}`

## web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx

- L201: `src={getFileURL(memberDetails.avatar_url)}`
- L222: `src={getFileURL(memberDetails.avatar_url)}`
- L243: `src={getFileURL(memberDetails.avatar_url)}`

## web/apps/web/helpers/cover-image.helper.ts

- L11: `* cover_image_url as null until uploads arrive (M5; M3 design 3.2).`

