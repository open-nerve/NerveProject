# P6 图片消费者清单（预检基线）

基线 `c59cedf337c5c5654bef2f0cbb33407ff04ced26` 的 49 文件、78 处源码搜索清单；尚未声称逐个文件全文审读。正式预检按 self/workspace/project/member/public-preview/非资料资源逐项映射；新增调用者不得遗漏。

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

