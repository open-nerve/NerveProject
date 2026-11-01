# M4 收尾：死成员、静态检查与关键词终态

2026-11-02；静态检查点基于 `d66dc91d` 加侧栏死链清理与 R26–R28。运行期间 tracked/unignored 输入不变，最近完整编辑器/构建/消费者/lint/AST/knip 共同输入为 `4fd32f78330f312c8fec474d25ba63eccf43036942952c856614742b64985738`。T11 文档同步仍在进行；最终 F、浏览器及 main 绑定另由 T13/T14 执行。

## 1. 原版扫描与本次清理

原 M1 附录 `deadsym.mjs`、`domains.mjs`、`lib/program.mjs` 原字复制，无分类/测试过滤改动。先 `react-router typegen`，再完整程序扫描；脚本 SHA、输入、typegen hash、raw/global/domains/M4 行及逐行源码 SHA 均在 `.superpowers/sdd/M4-closeout/dead-audit/`。`t11-dead-ast` 为清理前；`t11-dead-final-r28-2` 为本次最近源码检查点，当前源码位置和 SHA 已更新。

- 全仓 member **797**、prop **328**、export **2**；M4 member **217** + prop **137** = **354**。
- P16 最终 353 个 identity 中 352 个保留，1 个是 BlockMenu 虚拟定位对象的 getBoundingClientRect：R28 改用真实句柄同时定位与回焦后，该适配对象连同属性删除，不删除第三方协议。新增两项为 Markdown hook 的 `current.editor`、工具栏定位替身的 `children`，读者按下表实际分组定位。行号只定位；identity 使用 kind/file/symbol/同名次序。
- 两个 export 仍为 api-client 编译负例 `describeInstance`、`rejectsUndocumentedCalls`，由 tsc 检验预期拒绝，不进入生产 barrel。
- `SidebarQuickActions` 的旧草稿弹窗只存在 false 初值和关闭赋值，没有打开者；悬停状态无人读。删除弹窗、localStorage 订阅、定时器、状态与独占 import。真实 AppSidebar → SidebarQuickActions → SidebarAddButton → CommandPalette 创建入口/权限保持。共享 localStorage hook 仍有迭代/模块/菜单读者，未删除。此清理不改变 AST 候选数，不把未计数的死链充作候选减少。
- 现有创建/草稿表单 6 文件 58 项通过；未为不可达代码添加镜像测试。实际侧栏创建由最终 C12/I1 验证。

## 2. 当前 354 行

每行通过 G 编号连接下节具名读者，并在 current-ledger.json 保存当前完整源文件 SHA。所有 102 个当前候选来源文件已全文审查；“保留”不等于其后续 M5/M7 领域页面已通过业务验收。

| ID | 当前声明 | kind / symbol #次序 | 读者 | P16 行 |
|---|---|---|---|---|
| C001 | `web/apps/web/core/components/comments/data.test.tsx:54` | member / `attributes` #1 | G001 | FJ001 |
| C002 | `web/apps/web/core/components/comments/data.test.tsx:56` | member / `clearEditor` #1 | G001 | FJ002 |
| C003 | `web/apps/web/core/components/comments/data.test.tsx:57` | member / `setEditorValue` #1 | G001 | FJ003 |
| C004 | `web/apps/web/core/components/comments/data.test.tsx:58` | member / `isEditorReadyToDiscard` #1 | G001 | FJ004 |
| C005 | `web/apps/web/core/components/comments/permissions.test.tsx:22` | member / `name` #1 | G002 | FJ005 |
| C006 | `web/apps/web/core/components/comments/reaction.test.tsx:22` | member / `name` #1 | G003 | FJ006 |
| C007 | `web/apps/web/core/components/common/layout-error-boundary.tsx:41` | member / `getDerivedStateFromError` #1 | G004 | FJ007 |
| C008 | `web/apps/web/core/components/common/layout-error-boundary.tsx:45` | member / `componentDidCatch` #1 | G004 | FJ008 |
| C009 | `web/apps/web/core/components/common/layout-error-boundary.tsx:54` | member / `render` #1 | G004 | FJ009 |
| C010 | `web/apps/web/core/components/common/layout/sidebar/property-list-item.tsx:11` | member / `className` #1 | G005 | FJ010 |
| C011 | `web/apps/web/core/components/core/description-versions/root.test.tsx:38` | member / `id` #1 | G006 | FJ011 |
| C012 | `web/apps/web/core/components/core/description-versions/root.test.tsx:65` | member / `version` #1 | G006 | FJ012 |
| C013 | `web/apps/web/core/components/core/description-versions/root.test.tsx:65` | member / `resolve` #1 | G006 | FJ013 |
| C014 | `web/apps/web/core/components/core/description-versions/root.test.tsx:65` | member / `reject` #1 | G006 | FJ014 |
| C015 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx:23` | member / `identifier` #1 | G007 | FJ015 |
| C016 | `web/apps/web/core/components/issues/detail-copy.test.tsx:40` | member / `children` #1 | G008 | FJ016 |
| C017 | `web/apps/web/core/components/issues/detail-entry.test.tsx:26` | member / `text` #1 | G009 | FJ017 |
| C018 | `web/apps/web/core/components/issues/detail-entry.test.tsx:26` | member / `onClick` #1 | G009 | FJ018 |
| C019 | `web/apps/web/core/components/issues/detail-entry.test.tsx:40` | member / `children` #1 | G009 | FJ019 |
| C020 | `web/apps/web/core/components/issues/detail-fields.test.tsx:24` | member / `invalid` #1 | G010 | FJ020 |
| C021 | `web/apps/web/core/components/issues/detail-fields.test.tsx:24` | member / `description` #1 | G010 | FJ021 |
| C022 | `web/apps/web/core/components/issues/issue-detail/links/form.test.tsx:68` | member / `invalid` #1 | G011 | FJ022 |
| C023 | `web/apps/web/core/components/issues/issue-detail/links/form.test.tsx:68` | member / `message` #1 | G011 | FJ023 |
| C024 | `web/apps/web/core/components/issues/issue-layouts/calendar/quick-add-issue-actions.test.tsx:24` | member / `customQuickAddButton` #1 | G012 | FJ024 |
| C025 | `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx:31` | member / `sub_issue` #1 | G013 | FJ025 |
| C026 | `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx:32` | member / `show_empty_groups` #1 | G013 | FJ026 |
| C027 | `web/apps/web/core/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx:18` | member / `className` #1 | G014 | FJ027 |
| C028 | `web/apps/web/core/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx:18` | member / `strokeWidth` #1 | G014 | FJ028 |
| C029 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:49` | member / `children` #1 | G015 | FJ029 |
| C030 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:65` | member / `observe` #1 | G015 | FJ030 |
| C031 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:68` | member / `unobserve` #1 | G015 | FJ031 |
| C032 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:71` | member / `disconnect` #1 | G015 | FJ032 |
| C033 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:202` | member / `field` #1 | G015 | FJ033 |
| C034 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:202` | member / `key` #1 | G015 | FJ034 |
| C035 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:202` | member / `expected` #1 | G015 | FJ035 |
| C036 | `web/apps/web/core/components/issues/issue-layouts/list-data.test.tsx:33` | member / `observe` #1 | G016 | FJ036 |
| C037 | `web/apps/web/core/components/issues/issue-layouts/list-data.test.tsx:39` | member / `unobserve` #1 | G016 | FJ037 |
| C038 | `web/apps/web/core/components/issues/issue-layouts/list-data.test.tsx:40` | member / `disconnect` #1 | G016 | FJ038 |
| C039 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:45` | member / `children` #1 | G017 | FJ039 |
| C040 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:58` | member / `children` #2 | G017 | FJ040 |
| C041 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:58` | member / `issueId` #1 | G017 | FJ041 |
| C042 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx:30` | member / `observe` #1 | G018 | FJ042 |
| C043 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx:31` | member / `unobserve` #1 | G018 | FJ043 |
| C044 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx:32` | member / `disconnect` #1 | G018 | FJ044 |
| C045 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx:90` | member / `invalid` #1 | G018 | FJ045 |
| C046 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx:90` | member / `description` #1 | G018 | FJ046 |
| C047 | `web/apps/web/core/components/issues/issue-layouts/roots/archive-page.test.tsx:36` | member / `children` #1 | G019 | FJ047 |
| C048 | `web/apps/web/core/components/issues/issue-layouts/roots/project-lifecycle.test.tsx:43` | member / `element` #1 | G020 | FJ048 |
| C049 | `web/apps/web/core/components/issues/issue-layouts/route-scope.test.tsx:43` | member / `children` #1 | G021 | FJ049 |
| C050 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:83` | member / `observe` #1 | G022 | FJ050 |
| C051 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:84` | member / `unobserve` #1 | G022 | FJ051 |
| C052 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:85` | member / `disconnect` #1 | G022 | FJ052 |
| C053 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:123` | member / `observe` #1 | G023 | FJ053 |
| C054 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:124` | member / `unobserve` #1 | G023 | FJ054 |
| C055 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:125` | member / `disconnect` #1 | G023 | FJ055 |
| C056 | `web/apps/web/core/components/issues/issue-modal/fake-form.tsx:280` | member / `tag` #1 | G024 | FJ056 |
| C057 | `web/apps/web/core/components/issues/issue-modal/fake-form.tsx:280` | member / `invalid` #1 | G024 | FJ057 |
| C058 | `web/apps/web/core/components/issues/issue-modal/fake-form.tsx:280` | member / `description` #1 | G024 | FJ058 |
| C059 | `web/apps/web/core/components/issues/issue-modal/modal.tsx:30` | member / `default` #1 | G025 | FJ059 |
| C060 | `web/apps/web/core/components/issues/issue-modal/modal.tsx:31` | member / `loading` #1 | G025 | FJ060 |
| C061 | `web/apps/web/core/components/issues/peek-overview/header.tsx:38` | member / `className` #1 | G026 | FJ061 |
| C062 | `web/apps/web/core/components/issues/peek-overview/host-actions.test.tsx:29` | member / `children` #1 | G027 | FJ062 |
| C063 | `web/apps/web/core/components/issues/title-input.tsx:26` | member / `name` #1 | G028 | FJ063 |
| C064 | `web/apps/web/core/components/rich-filters/add-filters/button.tsx:27` | member / `defaultOpen` #1 | G029 | FJ064 |
| C065 | `web/apps/web/core/components/work-item-filters/filter-config.test.tsx:30` | member / `filter` #1 | G030 | FJ065 |
| C066 | `web/apps/web/core/components/work-item-filters/filters-hoc/base.tsx:46` | member / `filter` #1 | G031 | FJ066 |
| C067 | `web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx:39` | member / `label` #1 | G032 | FJ067 |
| C068 | `web/apps/web/core/hooks/editor/use-editor-mention.test.tsx:25` | member / `options` #1 | G033 | FJ068 |
| C069 | `web/apps/web/core/lib/issue-order.test.ts:19` | member / `order` #1 | G034 | FJ069 |
| C070 | `web/apps/web/core/lib/issue-order.test.ts:20` | member / `rows` #1 | G034 | FJ070 |
| C071 | `web/apps/web/core/lib/issue-order.test.ts:20` | member / `id` #1 | G034 | FJ071 |
| C072 | `web/apps/web/core/lib/issue-order.test.ts:20` | member / `names` #1 | G034 | FJ072 |
| C073 | `web/apps/web/core/lib/issue-order.test.ts:20` | member / `key` #1 | G034 | FJ073 |
| C074 | `web/apps/web/core/lib/issue-order.test.ts:21` | member / `ascending_ids` #1 | G034 | FJ074 |
| C075 | `web/apps/web/core/lib/issue-order.test.ts:22` | member / `descending_ids` #1 | G034 | FJ075 |
| C076 | `web/apps/web/core/lib/issue-order.test.ts:26` | member / `name_keys` #1 | G034 | FJ076 |
| C077 | `web/apps/web/core/lib/issue-order.test.ts:26` | member / `name` #1 | G034 | FJ077 |
| C078 | `web/apps/web/core/lib/issue-order.test.ts:26` | member / `key` #2 | G034 | FJ078 |
| C079 | `web/apps/web/core/lib/issue-order.test.ts:26` | member / `scenarios` #1 | G034 | FJ079 |
| C080 | `web/apps/web/core/lib/issue-order.test.ts:28` | member / `source` #1 | G034 | FJ080 |
| C081 | `web/apps/web/core/lib/issue-order.test.ts:144` | member / `id` #2 | G034 | FJ081 |
| C082 | `web/apps/web/core/lib/issue-order.test.ts:144` | member / `priority` #1 | G034 | FJ082 |
| C083 | `web/apps/web/core/lib/issue-order.test.ts:144` | member / `rank` #1 | G034 | FJ083 |
| C084 | `web/apps/web/core/services/issue/collaboration.service.test.ts:24` | member / `name` #1 | G035 | FJ084 |
| C085 | `web/apps/web/core/services/issue/collaboration.service.test.ts:25` | member / `run` #1 | G035 | FJ085 |
| C086 | `web/apps/web/core/services/issue/collaboration.service.test.ts:26` | member / `endpoint` #1 | G035 | FJ086 |
| C087 | `web/apps/web/core/services/issue/collaboration.service.test.ts:27` | member / `body` #1 | G035 | FJ087 |
| C088 | `web/apps/web/core/services/issue/collaboration.service.test.ts:28` | member / `answer` #1 | G035 | FJ088 |
| C089 | `web/apps/web/core/services/issue/collaboration.service.test.ts:29` | member / `status` #1 | G035 | FJ089 |
| C090 | `web/apps/web/core/services/issue/collaboration.service.test.ts:30` | member / `query` #1 | G035 | FJ090 |
| C091 | `web/apps/web/core/services/issue/issue-detail.service.test.ts:42` | member / `name` #1 | G036 | FJ091 |
| C092 | `web/apps/web/core/services/issue/issue-detail.service.test.ts:43` | member / `endpoint` #1 | G036 | FJ092 |
| C093 | `web/apps/web/core/services/issue/issue-detail.service.test.ts:44` | member / `run` #1 | G036 | FJ093 |
| C094 | `web/apps/web/core/services/issue/issue-detail.service.test.ts:45` | member / `body` #1 | G036 | FJ094 |
| C095 | `web/apps/web/core/services/issue/issue-detail.service.test.ts:46` | member / `answer` #1 | G036 | FJ095 |
| C096 | `web/apps/web/core/services/issue/issue-detail.service.test.ts:47` | member / `status` #1 | G036 | FJ096 |
| C097 | `web/apps/web/core/services/issue/issue-list.service.test.ts:146` | member / `name` #1 | G037 | FJ097 |
| C098 | `web/apps/web/core/services/issue/issue-list.service.test.ts:146` | member / `page` #1 | G037 | FJ098 |
| C099 | `web/apps/web/core/services/issue/issue-search.service.test.ts:11` | member / `query` #1 | G038 | FJ099 |
| C100 | `web/apps/web/core/services/issue/issue-search.service.test.ts:11` | member / `entries` #1 | G038 | FJ100 |
| C101 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:8` | member / `name` #1 | G039 | FJ101 |
| C102 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:9` | member / `endpoint` #1 | G039 | FJ102 |
| C103 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:10` | member / `run` #1 | G039 | FJ103 |
| C104 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:11` | member / `answer` #1 | G039 | FJ104 |
| C105 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:12` | member / `body` #1 | G039 | FJ105 |
| C106 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:13` | member / `query` #1 | G039 | FJ106 |
| C107 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts:14` | member / `status` #1 | G039 | FJ107 |
| C108 | `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts:118` | member / `filters` #1 | G040 | FJ108 |
| C109 | `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts:124` | member / `computedIssueFilters` #1 | G040 | FJ109 |
| C110 | `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts:129` | member / `computedDisplayFilters` #1 | G040 | FJ110 |
| C111 | `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts:133` | member / `computedDisplayProperties` #1 | G040 | FJ111 |
| C112 | `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts:260` | member / `kanban_filters` #1 | G040 | FJ112 |
| C113 | `web/apps/web/core/store/issue/issue_calendar_view.store.ts:27` | member / `activeMonthDate` #1 | G041 | FJ113 |
| C114 | `web/apps/web/core/store/issue/issue_calendar_view.store.ts:27` | member / `activeWeekDate` #1 | G041 | FJ114 |
| C115 | `web/apps/web/core/store/issue/issue_calendar_view.store.ts:29` | member / `regenerateCalendar` #1 | G041 | FJ115 |
| C116 | `web/apps/web/core/store/issue/issue-details/activity.store.ts:24` | member / `getActivitiesByIssueId` #1 | G042 | FJ116 |
| C117 | `web/apps/web/core/store/issue/issue-details/comment.store.ts:40` | member / `isLoading` #1 | G043 | FJ117 |
| C118 | `web/apps/web/core/store/issue/issue-details/description-versions.store.ts:30` | member / `getVersion` #1 | G044 | FJ118 |
| C119 | `web/packages/editor/src/components/menus/bubble-menu/root.tsx:43` | member / `bold` #1 | G045 | FJ120 |
| C120 | `web/packages/editor/src/components/menus/bubble-menu/root.tsx:44` | member / `italic` #1 | G045 | FJ121 |
| C121 | `web/packages/editor/src/components/menus/bubble-menu/root.tsx:45` | member / `underline` #1 | G045 | FJ122 |
| C122 | `web/packages/editor/src/components/menus/bubble-menu/root.tsx:46` | member / `strikethrough` #1 | G045 | FJ123 |
| C123 | `web/packages/editor/src/editor-interaction.test.ts:55` | member / `type` #1 | G046 | FJ124 |
| C124 | `web/packages/editor/src/editor-interaction.test.ts:55` | member / `id` #1 | G046 | FJ125 |
| C125 | `web/packages/editor/src/editor-markdown-roundtrip.test.ts:95` | member / `editor` #1 | G047 | 新增测试读者 |
| C126 | `web/packages/editor/src/extensions/callout/extension-config.ts:43` | member / `default` #1 | G048 | FJ126 |
| C127 | `web/packages/editor/src/extensions/callout/extension-config.ts:44` | member / `parseHTML` #1 | G048 | FJ127 |
| C128 | `web/packages/editor/src/extensions/code-inline/index.tsx:21` | member / `setCode` #1 | G049 | FJ128 |
| C129 | `web/packages/editor/src/extensions/code-inline/index.tsx:25` | member / `toggleCode` #1 | G049 | FJ129 |
| C130 | `web/packages/editor/src/extensions/code-inline/index.tsx:29` | member / `unsetCode` #1 | G049 | FJ130 |
| C131 | `web/packages/editor/src/extensions/code/code-block.ts:40` | member / `setCodeBlock` #1 | G050 | FJ131 |
| C132 | `web/packages/editor/src/extensions/code/code-block.ts:40` | member / `language` #1 | G050 | FJ132 |
| C133 | `web/packages/editor/src/extensions/code/code-block.ts:44` | member / `toggleCodeBlock` #1 | G050 | FJ133 |
| C134 | `web/packages/editor/src/extensions/code/code-block.ts:44` | member / `language` #2 | G050 | FJ134 |
| C135 | `web/packages/editor/src/extensions/custom-image/extension-config.ts:51` | member / `default` #1 | G051 | FJ135 |
| C136 | `web/packages/editor/src/extensions/custom-image/types.ts:51` | member / `event` #1 | G052 | FJ136 |
| C137 | `web/packages/editor/src/extensions/custom-image/types.ts:51` | member / `event` #2 | G052 | FJ137 |
| C138 | `web/packages/editor/src/extensions/custom-image/types.ts:51` | member / `file` #1 | G052 | FJ138 |
| C139 | `web/packages/editor/src/extensions/custom-image/types.ts:51` | member / `hasOpenedFileInputOnce` #1 | G052 | FJ139 |
| C140 | `web/packages/editor/src/extensions/custom-image/types.ts:68` | member / `fileMap` #1 | G052 | FJ140 |
| C141 | `web/packages/editor/src/extensions/custom-link/extension.tsx:57` | member / `href` #1 | G053 | FJ141 |
| C142 | `web/packages/editor/src/extensions/custom-link/extension.tsx:58` | member / `target` #1 | G053 | FJ142 |
| C143 | `web/packages/editor/src/extensions/custom-link/extension.tsx:59` | member / `rel` #1 | G053 | FJ143 |
| C144 | `web/packages/editor/src/extensions/custom-link/extension.tsx:60` | member / `class` #1 | G053 | FJ144 |
| C145 | `web/packages/editor/src/extensions/custom-link/extension.tsx:74` | member / `isBubbleMenuOpen` #1 | G053 | FJ145 |
| C146 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx:22` | member / `onKeyDown` #1 | G054 | FJ146 |
| C147 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx:25` | member / `name` #1 | G054 | FJ147 |
| C148 | `web/packages/editor/src/extensions/emoji/emoji.ts:31` | member / `emoji` #1 | G055 | FJ148 |
| C149 | `web/packages/editor/src/extensions/emoji/emoji.ts:31` | member / `setEmoji` #1 | G055 | FJ149 |
| C150 | `web/packages/editor/src/extensions/extensions.test.ts:30` | member / `editable` #1 | G056 | FJ150 |
| C151 | `web/packages/editor/src/extensions/extensions.test.ts:30` | member / `enableHistory` #1 | G056 | FJ151 |
| C152 | `web/packages/editor/src/extensions/extensions.test.ts:30` | member / `disabledExtensions` #1 | G056 | FJ152 |
| C153 | `web/packages/editor/src/extensions/horizontal-rule.ts:22` | member / `setHorizontalRule` #1 | G057 | FJ153 |
| C154 | `web/packages/editor/src/extensions/keymap.ts:17` | member / `customKeymap` #1 | G058 | FJ154 |
| C155 | `web/packages/editor/src/extensions/table/plugins/selection-outline/plugin.ts:17` | member / `decorations` #1 | G059 | FJ155 |
| C156 | `web/packages/editor/src/extensions/table/table/table-view.tsx:21` | member / `dom` #1 | G060 | FJ156 |
| C157 | `web/packages/editor/src/extensions/table/table/table-view.tsx:25` | member / `contentDOM` #1 | G060 | FJ157 |
| C158 | `web/packages/editor/src/extensions/table/table/table-view.tsx:53` | member / `update` #1 | G060 | FJ158 |
| C159 | `web/packages/editor/src/extensions/table/table/table-view.tsx:75` | member / `ignoreMutation` #1 | G060 | FJ159 |
| C160 | `web/packages/editor/src/extensions/table/table/table.ts:70` | member / `name` #1 | G061 | FJ160 |
| C161 | `web/packages/editor/src/extensions/table/table/table.ts:71` | member / `options` #1 | G061 | FJ161 |
| C162 | `web/packages/editor/src/extensions/table/table/table.ts:72` | member / `storage` #1 | G061 | FJ162 |
| C163 | `web/packages/editor/src/extensions/table/table/table.ts:73` | member / `parent` #1 | G061 | FJ163 |
| C164 | `web/packages/editor/src/extensions/unique-id/extension.ts:17` | member / `node` #1 | G062 | FJ164 |
| C165 | `web/packages/editor/src/extensions/unique-id/extension.ts:18` | member / `pos` #1 | G062 | FJ165 |
| C166 | `web/packages/editor/src/extensions/utility.ts:50` | member / `assetsUploadStatus` #1 | G063 | FJ166 |
| C167 | `web/packages/editor/src/extensions/utility.ts:51` | member / `uploadInProgress` #1 | G063 | FJ167 |
| C168 | `web/packages/editor/src/extensions/utility.ts:53` | member / `isTouchDevice` #1 | G063 | FJ168 |
| C169 | `web/packages/editor/src/helpers/tippy.ts:8` | member / `event` #1 | G064 | FJ169 |
| C170 | `web/packages/editor/src/helpers/tippy.ts:28` | member / `section` #1 | G064 | FJ170 |
| C171 | `web/packages/editor/src/helpers/tippy.ts:29` | member / `item` #1 | G064 | FJ171 |
| C172 | `web/packages/editor/src/sanitization-roundtrip.test.ts:45` | member / `node` #1 | G065 | FJ172 |
| C173 | `web/packages/editor/src/sanitization-roundtrip.test.ts:45` | member / `type` #1 | G065 | FJ173 |
| C174 | `web/packages/editor/src/sanitization-roundtrip.test.ts:45` | member / `name` #1 | G065 | FJ174 |
| C175 | `web/packages/editor/src/sanitization-roundtrip.test.ts:45` | member / `pos` #1 | G065 | FJ175 |
| C176 | `web/packages/editor/src/sanitization-roundtrip.test.ts:58` | member / `name` #2 | G065 | FJ176 |
| C177 | `web/packages/editor/src/sanitization-roundtrip.test.ts:59` | member / `input` #1 | G065 | FJ177 |
| C178 | `web/packages/editor/src/sanitization-roundtrip.test.ts:60` | member / `sanitized` #1 | G065 | FJ178 |
| C179 | `web/packages/editor/src/sanitization-roundtrip.test.ts:61` | member / `reserialized` #1 | G065 | FJ179 |
| C180 | `web/packages/editor/src/sanitization-roundtrip.test.ts:62` | member / `editor_document` #1 | G065 | FJ180 |
| C181 | `web/packages/editor/src/sanitization-roundtrip.test.ts:63` | member / `preserved_elements` #1 | G065 | FJ181 |
| C182 | `web/packages/editor/src/sanitization-roundtrip.test.ts:64` | member / `selector` #1 | G065 | FJ182 |
| C183 | `web/packages/editor/src/sanitization-roundtrip.test.ts:65` | member / `attributes` #1 | G065 | FJ183 |
| C184 | `web/packages/editor/src/sanitization-roundtrip.test.ts:66` | member / `styles` #1 | G065 | FJ184 |
| C185 | `web/packages/editor/src/sanitization-roundtrip.test.ts:67` | member / `text` #1 | G065 | FJ185 |
| C186 | `web/packages/editor/src/sanitization-roundtrip.test.ts:69` | member / `editable` #1 | G065 | FJ186 |
| C187 | `web/packages/editor/src/sanitization-roundtrip.test.ts:71` | member / `cases` #1 | G065 | FJ187 |
| C188 | `web/packages/editor/src/types/editor.ts:47` | member / `image` #1 | G066 | FJ188 |
| C189 | `web/packages/editor/src/types/editor.ts:48` | member / `savedSelection` #1 | G066 | FJ189 |
| C190 | `web/packages/editor/src/types/editor.ts:50` | member / `text-color` #1 | G066 | FJ190 |
| C191 | `web/packages/editor/src/types/editor.ts:53` | member / `link` #1 | G066 | FJ191 |
| C192 | `web/packages/editor/src/types/editor.ts:57` | member / `background-color` #1 | G066 | FJ192 |
| C193 | `web/packages/editor/src/types/editor.ts:60` | member / `text-align` #1 | G066 | FJ193 |
| C194 | `web/packages/editor/src/types/editor.ts:75` | member / `blur` #1 | G066 | FJ194 |
| C195 | `web/packages/editor/src/types/editor.ts:77` | member / `createSelectionAtCursorPosition` #1 | G066 | FJ195 |
| C196 | `web/packages/editor/src/types/editor.ts:79` | member / `focus` #1 | G066 | FJ196 |
| C197 | `web/packages/editor/src/types/editor.ts:80` | member / `getAttributesWithExtendedMark` #1 | G066 | FJ197 |
| C198 | `web/packages/editor/src/types/editor.ts:84` | member / `getCoordsFromPos` #1 | G066 | FJ198 |
| C199 | `web/packages/editor/src/types/editor.ts:85` | member / `getCurrentCursorPosition` #1 | G066 | FJ199 |
| C200 | `web/packages/editor/src/types/editor.ts:87` | member / `copyMarkdownToClipboard` #1 | G066 | FJ200 |
| C201 | `web/packages/editor/src/types/editor.ts:88` | member / `getSelectedText` #1 | G066 | FJ201 |
| C202 | `web/packages/editor/src/types/editor.ts:89` | member / `insertText` #1 | G066 | FJ202 |
| C203 | `web/packages/editor/src/types/editor.ts:94` | member / `redo` #1 | G066 | FJ203 |
| C204 | `web/packages/editor/src/types/editor.ts:95` | member / `scrollToNodeViaDOMCoordinates` #1 | G066 | FJ204 |
| C205 | `web/packages/editor/src/types/editor.ts:97` | member / `setFocusAtPosition` #1 | G066 | FJ205 |
| C206 | `web/packages/editor/src/types/editor.ts:98` | member / `undo` #1 | G066 | FJ206 |
| C207 | `web/packages/shared-state/src/store/rich-filters/filter.ts:68` | member / `initialFilterExpression` #1 | G067 | FJ207 |
| C208 | `web/packages/shared-state/src/store/rich-filters/filter.ts:72` | member / `onExpressionChange` #1 | G067 | FJ208 |
| C209 | `web/packages/types/src/issues/issue_sub_issues.ts:10` | member / `backlog` #1 | G068 | FJ209 |
| C210 | `web/packages/types/src/issues/issue_sub_issues.ts:11` | member / `unstarted` #1 | G068 | FJ210 |
| C211 | `web/packages/types/src/issues/issue_sub_issues.ts:12` | member / `started` #1 | G068 | FJ211 |
| C212 | `web/packages/types/src/issues/issue_sub_issues.ts:14` | member / `cancelled` #1 | G068 | FJ212 |
| C213 | `web/packages/utils/src/rich-filters/factories/configs/core.ts:32` | member / `singleValueOperator` #1 | G069 | FJ213 |
| C214 | `web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts:24` | member / `additionalOptions` #1 | G070 | FJ214 |
| C215 | `web/packages/utils/src/rich-filters/factories/configs/shared.ts:35` | member / `isEnabled` #1 | G071 | FJ215 |
| C216 | `web/packages/utils/src/rich-filters/factories/configs/shared.ts:36` | member / `label` #1 | G071 | FJ216 |
| C217 | `web/packages/utils/src/work-item-filters/configs/filters/label.ts:21` | member / `additionalOptions` #1 | G072 | FJ217 |
| C218 | `web/apps/web/core/components/comments/data.test.tsx:49` | prop / `id` #1 | G001 | FJ218 |
| C219 | `web/apps/web/core/components/comments/data.test.tsx:50` | prop / `initialValue` #1 | G001 | FJ219 |
| C220 | `web/apps/web/core/components/comments/data.test.tsx:51` | prop / `onChange` #1 | G001 | FJ220 |
| C221 | `web/apps/web/core/components/comments/data.test.tsx:52` | prop / `isSubmitting` #1 | G001 | FJ221 |
| C222 | `web/apps/web/core/components/comments/data.test.tsx:53` | prop / `onEnterKeyPress` #1 | G001 | FJ222 |
| C223 | `web/apps/web/core/components/comments/data.test.tsx:54` | prop / `editorProps` #1 | G001 | FJ223 |
| C224 | `web/apps/web/core/components/comments/data.test.tsx:55` | prop / `ref` #1 | G001 | FJ224 |
| C225 | `web/apps/web/core/components/core/description-versions/modal.test.tsx:22` | prop / `initialValue` #1 | G073 | FJ225 |
| C226 | `web/apps/web/core/components/core/description-versions/modal.test.tsx:23` | prop / `ref` #1 | G073 | FJ226 |
| C227 | `web/apps/web/core/components/core/description-versions/root.test.tsx:16` | prop / `onVersionClick` #1 | G006 | FJ227 |
| C228 | `web/apps/web/core/components/core/description-versions/root.test.tsx:17` | prop / `versions` #1 | G006 | FJ228 |
| C229 | `web/apps/web/core/components/core/description-versions/root.test.tsx:17` | prop / `id` #1 | G006 | FJ229 |
| C230 | `web/apps/web/core/components/core/description-versions/root.test.tsx:37` | prop / `activeVersionDescription` #1 | G006 | FJ230 |
| C231 | `web/apps/web/core/components/core/description-versions/root.test.tsx:38` | prop / `activeVersionDetails` #1 | G006 | FJ231 |
| C232 | `web/apps/web/core/components/core/description-versions/root.test.tsx:39` | prop / `handleClose` #1 | G006 | FJ232 |
| C233 | `web/apps/web/core/components/core/description-versions/root.test.tsx:40` | prop / `handleNavigation` #1 | G006 | FJ233 |
| C234 | `web/apps/web/core/components/core/description-versions/root.test.tsx:41` | prop / `error` #1 | G006 | FJ234 |
| C235 | `web/apps/web/core/components/core/description-versions/root.test.tsx:42` | prop / `retry` #1 | G006 | FJ235 |
| C236 | `web/apps/web/core/components/core/filters/date-filter-modal.test.tsx:16` | prop / `onSelect` #1 | G074 | FJ236 |
| C237 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx:54` | prop / `handleSubmit` #1 | G007 | FJ237 |
| C238 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx:55` | prop / `handleClose` #1 | G007 | FJ238 |
| C239 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx:56` | prop / `isSubmitting` #1 | G007 | FJ239 |
| C240 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx:57` | prop / `isOpen` #1 | G007 | FJ240 |
| C241 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx:58` | prop / `content` #1 | G007 | FJ241 |
| C242 | `web/apps/web/core/components/issues/detail-cache.test.tsx:28` | prop / `issueId` #1 | G075 | FJ242 |
| C243 | `web/apps/web/core/components/issues/detail-entry.test.tsx:26` | prop / `primaryButton` #1 | G009 | FJ243 |
| C244 | `web/apps/web/core/components/issues/detail-entry.test.tsx:32` | prop / `label` #1 | G009 | FJ244 |
| C245 | `web/apps/web/core/components/issues/detail-host.test.tsx:35` | prop / `issueId` #1 | G076 | FJ245 |
| C246 | `web/apps/web/core/components/issues/detail-host.test.tsx:39` | prop / `issueId` #2 | G076 | FJ246 |
| C247 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:29` | prop / `label` #1 | G077 | FJ247 |
| C248 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:30` | prop / `onChange` #1 | G077 | FJ248 |
| C249 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:31` | prop / `disabled` #1 | G077 | FJ249 |
| C250 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:44` | prop / `reactions` #1 | G077 | FJ250 |
| C251 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:44` | prop / `emoji` #1 | G077 | FJ251 |
| C252 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:44` | prop / `count` #1 | G077 | FJ252 |
| C253 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:44` | prop / `users` #1 | G077 | FJ253 |
| C254 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx:45` | prop / `onReactionClick` #1 | G077 | FJ254 |
| C255 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx:53` | prop / `issueId` #1 | G015 | FJ255 |
| C256 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:42` | prop / `identifier` #1 | G017 | FJ256 |
| C257 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:48` | prop / `children` #1 | G017 | FJ257 |
| C258 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:48` | prop / `className` #1 | G017 | FJ258 |
| C259 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:48` | prop / `ref` #1 | G017 | FJ259 |
| C260 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx:53` | prop / `children` #2 | G017 | FJ260 |
| C261 | `web/apps/web/core/components/issues/issue-layouts/roots/archive-detail.test.tsx:34` | prop / `header` #1 | G078 | FJ261 |
| C262 | `web/apps/web/core/components/issues/issue-layouts/roots/archive-detail.test.tsx:39` | prop / `disabled` #1 | G078 | FJ262 |
| C263 | `web/apps/web/core/components/issues/issue-layouts/roots/project-lifecycle.test.tsx:40` | prop / `children` #1 | G020 | FJ263 |
| C264 | `web/apps/web/core/components/issues/issue-layouts/roots/project-lifecycle.test.tsx:40` | prop / `as` #1 | G020 | FJ264 |
| C265 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx:17` | prop / `issue` #1 | G079 | FJ265 |
| C266 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx:18` | prop / `onClose` #1 | G079 | FJ266 |
| C267 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx:19` | prop / `onChange` #1 | G079 | FJ267 |
| C268 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx:20` | prop / `disabled` #1 | G079 | FJ268 |
| C269 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/created-on-column.tsx:18` | prop / `issue` #1 | G080 | FJ269 |
| C270 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx:21` | prop / `issue` #1 | G081 | FJ270 |
| C271 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx:22` | prop / `onClose` #1 | G081 | FJ271 |
| C272 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx:23` | prop / `onChange` #1 | G081 | FJ272 |
| C273 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx:24` | prop / `disabled` #1 | G081 | FJ273 |
| C274 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx:18` | prop / `issue` #1 | G082 | FJ274 |
| C275 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx:19` | prop / `onClose` #1 | G082 | FJ275 |
| C276 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx:20` | prop / `onChange` #1 | G082 | FJ276 |
| C277 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx:21` | prop / `disabled` #1 | G082 | FJ277 |
| C278 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/link-column.tsx:16` | prop / `issue` #1 | G083 | FJ278 |
| C279 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx:17` | prop / `issue` #1 | G084 | FJ279 |
| C280 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx:18` | prop / `onClose` #1 | G084 | FJ280 |
| C281 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx:19` | prop / `onChange` #1 | G084 | FJ281 |
| C282 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx:20` | prop / `disabled` #1 | G084 | FJ282 |
| C283 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx:19` | prop / `issue` #1 | G085 | FJ283 |
| C284 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx:20` | prop / `onClose` #1 | G085 | FJ284 |
| C285 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx:21` | prop / `onChange` #1 | G085 | FJ285 |
| C286 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx:22` | prop / `disabled` #1 | G085 | FJ286 |
| C287 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx:17` | prop / `issue` #1 | G086 | FJ287 |
| C288 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx:18` | prop / `onClose` #1 | G086 | FJ288 |
| C289 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx:19` | prop / `onChange` #1 | G086 | FJ289 |
| C290 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx:20` | prop / `disabled` #1 | G086 | FJ290 |
| C291 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/updated-on-column.tsx:18` | prop / `issue` #1 | G087 | FJ291 |
| C292 | `web/apps/web/core/components/issues/issue-layouts/utils.tsx:389` | prop / `key` #1 | G088 | FJ292 |
| C293 | `web/apps/web/core/components/issues/issue-layouts/utils.tsx:390` | prop / `titleTranslationKey` #1 | G088 | FJ293 |
| C294 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:52` | prop / `disabled` #1 | G022 | FJ294 |
| C295 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:53` | prop / `value` #1 | G022 | FJ295 |
| C296 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:54` | prop / `onChange` #1 | G022 | FJ296 |
| C297 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx:80` | prop / `children` #1 | G022 | FJ297 |
| C298 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:97` | prop / `disabled` #1 | G023 | FJ298 |
| C299 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:97` | prop / `value` #1 | G023 | FJ299 |
| C300 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:119` | prop / `children` #1 | G023 | FJ300 |
| C301 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx:119` | prop / `isOpen` #1 | G023 | FJ301 |
| C302 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:59` | prop / `disabled` #1 | G089 | FJ302 |
| C303 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:60` | prop / `value` #1 | G089 | FJ303 |
| C304 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:61` | prop / `onChange` #1 | G089 | FJ304 |
| C305 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:89` | prop / `MENU_ITEMS` #1 | G089 | FJ305 |
| C306 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:89` | prop / `key` #1 | G089 | FJ306 |
| C307 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:89` | prop / `action` #1 | G089 | FJ307 |
| C308 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx:117` | prop / `isOpen` #1 | G089 | FJ308 |
| C309 | `web/apps/web/core/components/issues/links/data.test.tsx:39` | prop / `size` #1 | G090 | FJ309 |
| C310 | `web/apps/web/core/components/issues/links/data.test.tsx:48` | prop / `loading` #1 | G090 | FJ310 |
| C311 | `web/apps/web/core/components/issues/links/data.test.tsx:48` | prop / `variant` #1 | G090 | FJ311 |
| C312 | `web/apps/web/core/components/issues/links/data.test.tsx:48` | prop / `size` #2 | G090 | FJ312 |
| C313 | `web/apps/web/core/components/issues/links/data.test.tsx:54` | prop / `isOpen` #1 | G090 | FJ313 |
| C314 | `web/apps/web/core/components/issues/links/data.test.tsx:57` | prop / `onClick` #1 | G090 | FJ314 |
| C315 | `web/apps/web/core/components/issues/parent-selection.test.tsx:32` | prop / `isOpen` #1 | G091 | FJ315 |
| C316 | `web/apps/web/core/components/issues/peek-overview/host-actions.test.tsx:19` | prop / `children` #1 | G027 | FJ316 |
| C317 | `web/apps/web/core/components/issues/peek-overview/host-actions.test.tsx:19` | prop / `to` #1 | G027 | FJ317 |
| C318 | `web/apps/web/core/components/issues/relations/data.test.tsx:49` | prop / `children` #1 | G092 | FJ318 |
| C319 | `web/apps/web/core/components/issues/relations/data.test.tsx:52` | prop / `children` #2 | G092 | FJ319 |
| C320 | `web/apps/web/core/components/issues/relations/data.test.tsx:53` | prop / `children` #3 | G092 | FJ320 |
| C321 | `web/apps/web/core/components/issues/relations/data.test.tsx:58` | prop / `children` #4 | G092 | FJ321 |
| C322 | `web/apps/web/core/components/issues/relations/data.test.tsx:59` | prop / `onClick` #1 | G092 | FJ322 |
| C323 | `web/apps/web/core/components/issues/relations/data.test.tsx:64` | prop / `children` #5 | G092 | FJ323 |
| C324 | `web/apps/web/core/components/issues/relations/data.test.tsx:74` | prop / `handleIssueCrudState` #1 | G092 | FJ324 |
| C325 | `web/apps/web/core/components/issues/workspace-draft/page.test.tsx:33` | prop / `issueId` #1 | G093 | FJ325 |
| C326 | `web/apps/web/core/components/issues/workspace-draft/page.test.tsx:33` | prop / `workspaceSlug` #1 | G093 | FJ326 |
| C327 | `web/apps/web/core/components/issues/workspace-draft/root.test.tsx:29` | prop / `issueId` #1 | G094 | FJ327 |
| C328 | `web/apps/web/core/components/power-k/ui/modal/search-results.tsx:11` | prop / `issue` #1 | G095 | FJ328 |
| C329 | `web/apps/web/core/components/power-k/ui/modal/search-results.tsx:11` | prop / `project` #1 | G095 | FJ329 |
| C330 | `web/apps/web/core/components/work-item-filters/filters-hoc/shared.ts:27` | prop / `saveViewOptions` #1 | G096 | FJ330 |
| C331 | `web/apps/web/core/components/work-item-filters/filters-hoc/shared.ts:31` | prop / `enableUpdateView` #1 | G096 | FJ331 |
| C332 | `web/apps/web/core/components/work-item-filters/filters-hoc/shared.ts:32` | prop / `updateViewOptions` #1 | G096 | FJ332 |
| C333 | `web/packages/constants/src/issue/filter.ts:241` | prop / `key` #1 | G097 | FJ333 |
| C334 | `web/packages/constants/src/issue/filter.ts:242` | prop / `labelTranslationKey` #1 | G097 | FJ334 |
| C335 | `web/packages/constants/src/issue/filter.ts:243` | prop / `isSelected` #1 | G097 | FJ335 |
| C336 | `web/packages/constants/src/issue/filter.ts:244` | prop / `onClick` #1 | G097 | FJ336 |
| C337 | `web/packages/constants/src/issue/layout.ts:42` | prop / `key` #1 | G098 | FJ337 |
| C338 | `web/packages/constants/src/issue/layout.ts:43` | prop / `i18n_title` #1 | G098 | FJ338 |
| C339 | `web/packages/constants/src/issue/layout.ts:44` | prop / `i18n_label` #1 | G098 | FJ339 |
| C340 | `web/packages/editor/src/editor-menu-accessibility.test.tsx:13` | prop / `children` #1 | G099 | 新增测试读者 |
| C341 | `web/packages/editor/src/extensions/code/code-block-node-view.tsx:26` | prop / `node` #1 | G100 | FJ340 |
| C342 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx:15` | prop / `name` #1 | G054 | FJ341 |
| C343 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx:16` | prop / `emoji` #1 | G054 | FJ342 |
| C344 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx:17` | prop / `shortcodes` #1 | G054 | FJ343 |
| C345 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx:18` | prop / `tags` #1 | G054 | FJ344 |
| C346 | `web/packages/shared-state/src/store/rich-filters/config.ts:23` | prop / `allEnabledSupportedOperators` #1 | G101 | FJ345 |
| C347 | `web/packages/shared-state/src/store/rich-filters/config.ts:24` | prop / `firstOperator` #1 | G101 | FJ346 |
| C348 | `web/packages/shared-state/src/store/rich-filters/config.ts:26` | prop / `getOperatorConfig` #1 | G101 | FJ347 |
| C349 | `web/packages/shared-state/src/store/rich-filters/config.ts:29` | prop / `getDisplayOperatorByValue` #1 | G101 | FJ348 |
| C350 | `web/packages/shared-state/src/store/rich-filters/config.ts:31` | prop / `mutate` #1 | G101 | FJ349 |
| C351 | `web/packages/types/src/rich-filters/field-types/shared.ts:33` | prop / `icon` #1 | G102 | FJ350 |
| C352 | `web/packages/types/src/rich-filters/field-types/shared.ts:34` | prop / `iconClassName` #1 | G102 | FJ351 |
| C353 | `web/packages/types/src/rich-filters/field-types/shared.ts:35` | prop / `disabled` #1 | G102 | FJ352 |
| C354 | `web/packages/types/src/rich-filters/field-types/shared.ts:36` | prop / `description` #1 | G102 | FJ353 |

## 3. 具名读者

| 编号 | 文件 | 实际消费链 |
|---|---|---|
| G001 | `web/apps/web/core/components/comments/data.test.tsx` | RichTextEditor 替身把 editorProps.attributes/id/initialValue 送 textarea，onChange/onEnterKeyPress/ref 由 CommentCreate/EditForm 使用；useImperativeHandle 的三方法由真实表单调用。 |
| G002 | `web/apps/web/core/components/comments/permissions.test.tsx` | 翻译替身插值参数 name；权限拒绝后按钮/错误提示的文本断言。 |
| G003 | `web/apps/web/core/components/comments/reaction.test.tsx` | 翻译替身插值 name；EmojiReactionGroup 的真实点击反馈断言。 |
| G004 | `web/apps/web/core/components/common/layout-error-boundary.tsx` | React class boundary 调用 getDerivedStateFromError/componentDidCatch/render；不是业务属性读取。 |
| G005 | `web/apps/web/core/components/common/layout/sidebar/property-list-item.tsx` | SidebarPropertyListItem 把 className 传给 Icon FC；调用方传入图标组件。 |
| G006 | `web/apps/web/core/components/core/description-versions/root.test.tsx` | Dropdown 替身 map 版本 id 并触发 onVersionClick；Modal 替身读取描述/active.id/error/retry/导航；deferred 的 version/resolve/reject 由并发测试逐次完成。 |
| G007 | `web/apps/web/core/components/issues/delete-issue-modal.test.tsx` | 翻译器消费 identifier；AlertModal 替身读取内容/开关/提交和关闭回调，真实删除拥有者驱动。 |
| G008 | `web/apps/web/core/components/issues/detail-copy.test.tsx` | Tooltip 替身返回 children，真实编号复制按钮与反馈可见性断言。 |
| G009 | `web/apps/web/core/components/issues/detail-entry.test.tsx` | EmptyState 替身消费 primaryButton.text/onClick；Breadcrumb/ProjectAuthWrapper 消费 label/children。 |
| G010 | `web/apps/web/core/components/issues/detail-fields.test.tsx` | focus 记录的 invalid/description 由 activeElement 采集，再逐对象比较错误可达性。 |
| G011 | `web/apps/web/core/components/issues/issue-detail/links/form.test.tsx` | focus 记录 invalid/message 再 toEqual 验证字段关联提示。 |
| G012 | `web/apps/web/core/components/issues/issue-layouts/calendar/quick-add-issue-actions.test.tsx` | 替身实际渲染 customQuickAddButton，触发真实日历快速添加拥有者。 |
| G013 | `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx` | selectedExtraOptions[option.key] 读取 sub_issue/show_empty_groups。 |
| G014 | `web/apps/web/core/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx` | FilterDropdownHeader 的 cloneElement 为传入 SVG 写 className/strokeWidth，React 图标读取。 |
| G015 | `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx` | VisibleObserver 的 observe/unobserve/disconnect 被真实可见性 hook 调用；children/issueId 由行渲染；it.each 解构 field/key/expected 比较 HTTP 体。 |
| G016 | `web/apps/web/core/components/issues/issue-layouts/list-data.test.tsx` | IntersectionObserver 替身 observe/unobserve/disconnect 由列表分页观察 hook 调用。 |
| G017 | `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx` | Tooltip/Row/ControlLink 替身透传 children/className/ref；viewport 读 children.props.issueId，宽度断言读 identifier。 |
| G018 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx` | ResizeObserver 由 TextInput 调用；focus 记录 invalid/description 供错误反馈断言。 |
| G019 | `web/apps/web/core/components/issues/issue-layouts/roots/archive-page.test.tsx` | viewport wrapper 透传 children，真实归档拥有者进行请求和权限验证。 |
| G020 | `web/apps/web/core/components/issues/issue-layouts/roots/project-lifecycle.test.tsx` | viewport 读取 as/children；autoScroll 替身按 element 绑定并回收监听集合。 |
| G021 | `web/apps/web/core/components/issues/issue-layouts/route-scope.test.tsx` | 布局替身渲染 children，真实路由拥有者切换并销毁浮层。 |
| G022 | `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx` | ProjectSelect 的 disabled/value/onChange 由真实克隆表单传入；Modal 读 children；ResizeObserver 支持真实 TextInput。 |
| G023 | `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx` | ProjectSelect 读 disabled/value；Modal 读 isOpen/children；ResizeObserver 支持真实 TextInput 加载态。 |
| G024 | `web/apps/web/core/components/issues/issue-modal/fake-form.tsx` | focusInvalid/activeField 将 tag/invalid/description 写入记录；form-fields.test.tsx 比较完整对象。 |
| G025 | `web/apps/web/core/components/issues/issue-modal/modal.tsx` | primaryButtonText 交给 base/form；form 选择 default/loading 按钮文案。 |
| G026 | `web/apps/web/core/components/issues/peek-overview/header.tsx` | 菜单项 icon:FC 的 className 由渲染图标读取。 |
| G027 | `web/apps/web/core/components/issues/peek-overview/host-actions.test.tsx` | RouterLink 替身读 children/to；Tooltip 返回 children，真实 Header 驱动复制/打开/关闭。 |
| G028 | `web/apps/web/core/components/issues/title-input.tsx` | onSubmit 调 issueOperations.update，name 是原生更新体字段。 |
| G029 | `web/apps/web/core/components/rich-filters/add-filters/button.tsx` | buttonConfig 展开给 AddFilterDropdown，后者解构 defaultOpen 交 CustomSearchSelect。 |
| G030 | `web/apps/web/core/components/work-item-filters/filter-config.test.tsx` | Chips 读取 filter.allConditions 并传给真实 FilterItem。 |
| G031 | `web/apps/web/core/components/work-item-filters/filters-hoc/base.tsx` | render props children({filter}) 向 project/workspace-level 消费者提供过滤实例。 |
| G032 | `web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx` | translatedFilterTypeOptions → PopoverMenu.render(item) → NotificationFilterOptionItem {...item}；menu-option-item.tsx 解构 label 并显示按钮名称。keyExtractor 只读 value，显式标签类型仍随同一个数据对象传递。 |
| G033 | `web/apps/web/core/hooks/editor/use-editor-mention.test.tsx` | Probe 展开 options 给真实 useEditorMention，rerender 改 scope/查询并检查结果。 |
| G034 | `web/apps/web/core/lib/issue-order.test.ts` | JSON fixture 的 name_keys/scenarios 经 map/it.each 转行排序；reviver.context.source 保留大整数；samples 的 id/priority/rank 直接比较排序结果。 |
| G035 | `web/apps/web/core/services/issue/collaboration.service.test.ts` | Case 表逐条执行 run，按 endpoint/body/query/status/answer 校验 HTTP 与解析；name 为 it.each 标题。 |
| G036 | `web/apps/web/core/services/issue/issue-detail.service.test.ts` | Case 表逐条执行 run，按 endpoint/body/status/answer 校验 HTTP 与返回；name 为用例标题。 |
| G037 | `web/apps/web/core/services/issue/issue-list.service.test.ts` | it.each 解构 name/page，校验各分页数据拒绝或接受。 |
| G038 | `web/apps/web/core/services/issue/issue-search.service.test.ts` | 查询输入表按 query 发请求并比较 entries，验证编码与作用域。 |
| G039 | `web/apps/web/core/services/issue/workspace_draft.service.test.ts` | Case 表逐条执行 run，比较 endpoint/body/query/status/answer；name 为 it.each 标题。 |
| G040 | `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts` | Project/Archived/Profile filter.store 实现并读取 filters/computed*；loadFromStorage 读 kanban_filters。 |
| G041 | `web/apps/web/core/store/issue/issue_calendar_view.store.ts` | CalendarGridHeader 读 filters.activeMonthDate/activeWeekDate；updateCalendarPayload/用户偏好 reaction 调 regenerateCalendar。 |
| G042 | `web/apps/web/core/store/issue/issue-details/activity.store.ts` | 同目录 activity.store.test.ts 调 getActivitiesByIssueId 核分页顺序和权限失效隐藏；是可观察查询接口。 |
| G043 | `web/apps/web/core/store/issue/issue-details/comment.store.ts` | 同目录 comment.store.test.ts:70 读 isLoading，验证拒绝后解除加载态。 |
| G044 | `web/apps/web/core/store/issue/issue-details/description-versions.store.ts` | 同目录 description-versions.store.test.ts 读 getVersion，核版本隔离/空HTML/权限失效。 |
| G045 | `web/packages/editor/src/components/menus/bubble-menu/root.tsx` | 格式按钮 editorState[item.key] 读取 bold/italic/underline/strikethrough；颜色样式读取 textColor/backgroundColor。 |
| G046 | `web/packages/editor/src/editor-interaction.test.ts` | createEditor 解构 editable/content/disabledExtensions/attributes 传真实 Editor；blockIds 收集 type/id 后逐条断言。 |
| G047 | `web/packages/editor/src/editor-markdown-roundtrip.test.ts` | Host 用真实 useEditor 将 Editor 放入 current.editor；waitFor、原生 paste 分发、文本/格式/任务断言及 finally destroy 读取该成员。 |
| G048 | `web/packages/editor/src/extensions/callout/extension-config.ts` | TipTap getSchemaByResolvedExtensions 读属性 default/parseHTML；callout HTML/JSON 双向转换。 |
| G049 | `web/packages/editor/src/extensions/code-inline/index.tsx` | 与已锁定 @tiptap/extension-code Commands 合并；内置 toggle/input/paste 命令及自定义链同型。 |
| G050 | `web/packages/editor/src/extensions/code/code-block.ts` | 与 @tiptap/extension-code-block Commands 合并；set/toggle 的 language 传 setNode/toggleNode。 |
| G051 | `web/packages/editor/src/extensions/custom-image/extension-config.ts` | TipTap schema 读 CustomImageAttributes 的 default；已有图像扩展需保持序列化。 |
| G052 | `web/packages/editor/src/extensions/custom-image/types.ts` | uploader/utils 按 uploadEntity.event 分派、读 file/hasOpenedFileInputOnce，extension.utils 用 fileMap 跟踪在途上传。 |
| G053 | `web/packages/editor/src/extensions/custom-link/extension.tsx` | setLink 将 href/target/rel/class 传 setMark；bubble-menu/link-view-container 通过 storage.link 读 isBubbleMenuOpen。 |
| G054 | `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx` | SuggestionProps.items 经 items.map 读 name/emoji/shortcodes；suggestion.ts tags 匹配；ReactRenderer.ref.onKeyDown 接受插件按键。 |
| G055 | `web/packages/editor/src/extensions/emoji/emoji.ts` | helper/editor-commands 与 suggestion 命令链 setEmoji({emoji})，节点 attrs 渲染。 |
| G056 | `web/packages/editor/src/extensions/extensions.test.ts` | editor fixture 解构 editable/enableHistory/disabledExtensions 生成真实扩展，检查只读/禁用/撤销边界。 |
| G057 | `web/packages/editor/src/extensions/horizontal-rule.ts` | helper/editor-commands 的 divider 分支调 setHorizontalRule；TipTap 动态 Commands 合同。 |
| G058 | `web/packages/editor/src/extensions/keymap.ts` | TipTap Commands 分组 customKeymap 映射 selectTextWithinNodeBoundaries；本扩展 Mod-a 调用。 |
| G059 | `web/packages/editor/src/extensions/table/plugins/selection-outline/plugin.ts` | PluginKey.getState(state).decorations 被 props.decorations 返回到 ProseMirror。 |
| G060 | `web/packages/editor/src/extensions/table/table/table-view.tsx` | ProseMirror NodeView 协议读取 dom/contentDOM、调用 update/ignoreMutation；DOM 编辑进入真实表格。 |
| G061 | `web/packages/editor/src/extensions/table/table/table.ts` | TipTap getExtensionField 绑定 name/options/storage/parent 的 this 上下文，扩展节点解析/插入合同。 |
| G062 | `web/packages/editor/src/extensions/unique-id/extension.ts` | sanitization-roundtrip.test.ts 的 generateUniqueID({node,pos}) 读取上下文；plugin/utils 调配置函数。 |
| G063 | `web/packages/editor/src/extensions/utility.ts` | use-editor-storage、custom-image upload-status 与工具栏通过 editor.storage.utility 读取 assetsUploadStatus/uploadInProgress/isTouchDevice。 |
| G064 | `web/packages/editor/src/helpers/tippy.ts` | slash-commands 命令列表用返回 section/item；Suggestion ReactRenderer.ref.onKeyDown({event})。 |
| G065 | `web/packages/editor/src/sanitization-roundtrip.test.ts` | generateUniqueID 解构 node.type.name/pos；JSON cases 遍历 name/input/sanitized/reserialized/editor_document/preserved_elements/selector/attributes/styles/text/editable 并断言往返。 |
| G066 | `web/packages/editor/src/types/editor.ts` | TCommandWithProps 映射后由 helpers/editor-commands 消费；EditorRefApi 由 helpers/editor-ref 实现并经 useImperativeHandle 暴露，P14 公共合同。 |
| G067 | `web/packages/shared-state/src/store/rich-filters/filter.ts` | hasChanges 比较 initialFilterExpression；mutate/条件更新调 onExpressionChange，work-item-filters/filter.store 注册回调。 |
| G068 | `web/packages/types/src/issues/issue_sub_issues.ts` | sub_issues.store 按 state.group 动态写读分布；sub-issues/progress 读 completed 和总数。 |
| G069 | `web/packages/utils/src/rich-filters/factories/configs/core.ts` | createMultiSelectFilterFieldConfig 写 singleValueOperator；FilterConfig.getDisplayOperatorByValue 读取。 |
| G070 | `web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts` | additionalOptions 展开给 properties/member 的选项工厂，再进入 getOptions 列表。 |
| G071 | `web/packages/utils/src/rich-filters/factories/configs/shared.ts` | createFilterConfig 参数展开到实际配置；FilterConfig 构造读取 isEnabled/label。 |
| G072 | `web/packages/utils/src/work-item-filters/configs/filters/label.ts` | additionalOptions 经 createLabelFilterConfig 展开，core.getOptions 并入结果。 |
| G073 | `web/apps/web/core/components/core/description-versions/modal.test.tsx` | RichTextEditor 替身渲染 initialValue 并向 ref 暴露复制行为，真实 Modal 驱动等待与恢复。 |
| G074 | `web/apps/web/core/components/core/filters/date-filter-modal.test.tsx` | Calendar 替身调用 onSelect 提供反向区间，真实表单验证错误边界。 |
| G075 | `web/apps/web/core/components/issues/detail-cache.test.tsx` | SiblingItem 替身渲染 issueId，实际 SWR/store 处理权限缓存。 |
| G076 | `web/apps/web/core/components/issues/detail-host.test.tsx` | 真实 Detail/Peek 根向替身传 issueId，Probe 保存并测试弹窗所有者。 |
| G077 | `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx` | Picker 替身读 label/onChange/disabled；Group map 读取 emoji/count/users 并调用 onReactionClick，真实 reactions store 接收确认。 |
| G078 | `web/apps/web/core/components/issues/issue-layouts/roots/archive-detail.test.tsx` | AppHeader 透传 header；DescriptionInput 用 disabled 显示只读，真实归档路由与控件断言。 |
| G079 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，assignee 列读取字段并交真实控件；回焦返回 td。 |
| G080 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/created-on-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，created-on 列读取字段并交真实控件；回焦返回 td。 |
| G081 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，due-date 列读取字段并交真实控件；回焦返回 td。 |
| G082 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，label 列读取字段并交真实控件；回焦返回 td。 |
| G083 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/link-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，link 列读取字段并交真实控件；回焦返回 td。 |
| G084 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，priority 列读取字段并交真实控件；回焦返回 td。 |
| G085 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，start-date 列读取字段并交真实控件；回焦返回 td。 |
| G086 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，state 列读取字段并交真实控件；回焦返回 td。 |
| G087 | `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/updated-on-column.tsx` | SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，updated-on 列读取字段并交真实控件；回焦返回 td。 |
| G088 | `web/apps/web/core/components/issues/issue-layouts/utils.tsx` | ISSUE_GROUP_BY_OPTIONS 经 useGroupByOptions 返回；group-by/sub-group-by 菜单读取 key/titleTranslationKey。 |
| G089 | `web/apps/web/core/components/issues/issue-modal/draft.test.tsx` | ProjectSelect 读 disabled/value/onChange；QuickActions map 读取 MENU_ITEMS.key/action；Modal 读 isOpen。 |
| G090 | `web/apps/web/core/components/issues/links/data.test.tsx` | Input/Button 替身剥除 size/variant 防止传错 DOM，loading 驱动 disabled；Modal/Menu 读 isOpen/onClick；多余 ref 声明已删，原生 rest props 仍透传。 |
| G091 | `web/apps/web/core/components/issues/parent-selection.test.tsx` | Modal 替身读 isOpen，真实父项选择拥有者驱动打开关闭。 |
| G092 | `web/apps/web/core/components/issues/relations/data.test.tsx` | Tooltip/Menu/Collapsible 替身透传 children，MenuItem 调 onClick；RelationList 调 handleIssueCrudState，真实拥有者验证确认事件不重复写。 |
| G093 | `web/apps/web/core/components/issues/workspace-draft/page.test.tsx` | DraftIssueBlock 替身显示 workspaceSlug/issueId，真实草稿路由验证作用域。 |
| G094 | `web/apps/web/core/components/issues/workspace-draft/root.test.tsx` | DraftIssueBlock 替身读 issueId；真实无项目草稿列表验证无需偏好读取。 |
| G095 | `web/apps/web/core/components/power-k/ui/modal/search-results.tsx` | issues.map 解构 issue/project，显示编号名称并构造原生路由；root 传入搜索结果。 |
| G096 | `web/apps/web/core/components/work-item-filters/filters-hoc/shared.ts` | project-level/workspace-level 的 hasSaveOption/handleSaveView/handleUpdateView 仍读保存/更新配置；M7 view 合同。 |
| G097 | `web/packages/constants/src/issue/filter.ts` | ACTIVITY_FILTER_OPTIONS 通过 issue-activity/activity-filter.tsx 的 map 读取 key/labelTranslationKey/isSelected/onClick。 |
| G098 | `web/packages/constants/src/issue/layout.ts` | ISSUE_LAYOUTS 经布局菜单 map 读取 key/i18n_title/i18n_label；图标函数被 React 调用。 |
| G099 | `web/packages/editor/src/editor-menu-accessibility.test.tsx` | 真实 EditorBubbleMenu 的 JSX children 由仅替换外部定位壳的 BubbleMenu 替身渲染，工具栏命令与 editor 状态均为真实对象；测试实际点击并断言。 |
| G100 | `web/packages/editor/src/extensions/code/code-block-node-view.tsx` | ReactNodeViewRenderer → CodeBlockComponent → node.attrs.id、node.textContent 与 NodeViewContent；真实节点内容和命名复制按钮均有回归。 |
| G101 | `web/packages/shared-state/src/store/rich-filters/config.ts` | IFilterConfig 被 filter-item/root 与 add-filters/dropdown 读取操作符/显示操作符，配置管理器调用 mutate。 |
| G102 | `web/packages/types/src/rich-filters/field-types/shared.ts` | filter-value-input/dropdown/search-options 渲染选项 icon/iconClassName/disabled/description。 |

EditorRefApi 保留 P14 公开 ref 协议：实现由 useImperativeHandle 提供，真实消费者按命令分发；其中图片相关能力在 M4 禁用、没有当前产品调用的方法不宣称业务通过。`copyMarkdownToClipboard` 旧方法的潜在监听/拒绝处理风险随 M5 编辑器复用条件转交；当前 M4 走 `getMarkDown` + `useCopyText`。

## 4. 历史 52 行

原声明及删除提交逐行对账，commit diff hash 与当前文件 hash 见 history52。旧草稿文件整体消失；另外三个文件仍保留其他合同，原 owner/method 已不存在，不按文件不存在计算整批删除。

| ID | 原 owner.symbol | 删除提交 | 当前核验 |
|---|---|---|---|
| H001 | `TWorkspaceDraftIssue.sort_order` | `87d1bcd7` | 原文件已删除 |
| H002 | `TWorkspaceDraftIssue.parent_id` | `87d1bcd7` | 原文件已删除 |
| H003 | `TWorkspaceDraftIssue.completed_at` | `87d1bcd7` | 原文件已删除 |
| H004 | `TWorkspaceDraftIssue.updated_at` | `87d1bcd7` | 原文件已删除 |
| H005 | `TWorkspaceDraftIssue.updated_by` | `87d1bcd7` | 原文件已删除 |
| H006 | `TWorkspaceDraftIssue.is_draft` | `87d1bcd7` | 原文件已删除 |
| H007 | `TWorkspaceDraftPaginationInfo.prev_cursor` | `87d1bcd7` | 原文件已删除 |
| H008 | `TWorkspaceDraftPaginationInfo.prev_page_results` | `87d1bcd7` | 原文件已删除 |
| H009 | `TWorkspaceDraftPaginationInfo.total_pages` | `87d1bcd7` | 原文件已删除 |
| H010 | `TWorkspaceDraftPaginationInfo.count` | `87d1bcd7` | 原文件已删除 |
| H011 | `TWorkspaceDraftPaginationInfo.total_results` | `87d1bcd7` | 原文件已删除 |
| H012 | `TWorkspaceDraftPaginationInfo.extra_stats` | `87d1bcd7` | 原文件已删除 |
| H013 | `TWorkspaceDraftPaginationInfo.grouped_by` | `87d1bcd7` | 原文件已删除 |
| H014 | `TWorkspaceDraftPaginationInfo.sub_grouped_by` | `87d1bcd7` | 原文件已删除 |
| H015 | `TWorkspaceDraftQueryParams.per_page` | `87d1bcd7` | 原文件已删除 |
| H016 | `TWorkspaceDraftQueryParams.cursor` | `87d1bcd7` | 原文件已删除 |
| H017 | `WorkspaceService.searchWorkspace.params.project_id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H018 | `WorkspaceService.searchWorkspace.params.search` | `5f0961c1` | 文件仍在、原声明已删除 |
| H019 | `WorkspaceService.searchWorkspace.params.workspace_search` | `5f0961c1` | 文件仍在、原声明已删除 |
| H020 | `TProjectIssuesSearchParams.search` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H021 | `TProjectIssuesSearchParams.parent` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H022 | `TProjectIssuesSearchParams.issue_relation` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H023 | `TProjectIssuesSearchParams.cycle` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H024 | `TProjectIssuesSearchParams.module` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H025 | `TProjectIssuesSearchParams.sub_issue` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H026 | `TProjectIssuesSearchParams.issue_id` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H027 | `TProjectIssuesSearchParams.workspace_search` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H028 | `TProjectIssuesSearchParams.target_date` | `4c0ed5ea` | 文件仍在、原声明已删除 |
| H029 | `IWorkspaceSearchResults.results.workspace` | `5f0961c1` | 文件仍在、原声明已删除 |
| H030 | `IWorkspaceSearchResults.results.project` | `5f0961c1` | 文件仍在、原声明已删除 |
| H031 | `IWorkspaceSearchResults.results.issue` | `5f0961c1` | 文件仍在、原声明已删除 |
| H032 | `IWorkspaceSearchResults.results.cycle` | `5f0961c1` | 文件仍在、原声明已删除 |
| H033 | `IWorkspaceSearchResults.results.module` | `5f0961c1` | 文件仍在、原声明已删除 |
| H034 | `IWorkspaceSearchResults.results.issue_view` | `5f0961c1` | 文件仍在、原声明已删除 |
| H035 | `IWorkspaceDefaultSearchResult.id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H036 | `IWorkspaceDefaultSearchResult.name` | `5f0961c1` | 文件仍在、原声明已删除 |
| H037 | `IWorkspaceDefaultSearchResult.project_id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H038 | `IWorkspaceDefaultSearchResult.project__identifier` | `5f0961c1` | 文件仍在、原声明已删除 |
| H039 | `IWorkspaceDefaultSearchResult.workspace__slug` | `5f0961c1` | 文件仍在、原声明已删除 |
| H040 | `IWorkspaceSearchResult.id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H041 | `IWorkspaceSearchResult.name` | `5f0961c1` | 文件仍在、原声明已删除 |
| H042 | `IWorkspaceSearchResult.slug` | `5f0961c1` | 文件仍在、原声明已删除 |
| H043 | `IWorkspaceIssueSearchResult.id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H044 | `IWorkspaceIssueSearchResult.name` | `5f0961c1` | 文件仍在、原声明已删除 |
| H045 | `IWorkspaceIssueSearchResult.project__identifier` | `5f0961c1` | 文件仍在、原声明已删除 |
| H046 | `IWorkspaceIssueSearchResult.project_id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H047 | `IWorkspaceIssueSearchResult.sequence_id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H048 | `IWorkspaceIssueSearchResult.workspace__slug` | `5f0961c1` | 文件仍在、原声明已删除 |
| H049 | `IWorkspaceProjectSearchResult.id` | `5f0961c1` | 文件仍在、原声明已删除 |
| H050 | `IWorkspaceProjectSearchResult.identifier` | `5f0961c1` | 文件仍在、原声明已删除 |
| H051 | `IWorkspaceProjectSearchResult.name` | `5f0961c1` | 文件仍在、原声明已删除 |
| H052 | `IWorkspaceProjectSearchResult.workspace__slug` | `5f0961c1` | 文件仍在、原声明已删除 |

## 5. 14 范围 lint 与守卫

`make lint-web` 54/54，`make knip` 通过。为避免 turbo 缓存代替实际计数，另对原 14 范围逐个执行 oxlint JSON：M4 域、全 M4 当前修改路径、全仓 no-shadow 均 **0**；根规则仍为 error，无 no-shadow 屏蔽或降级。cap 总数 **86**，逐包与实际相等，无新增豁免；规则/config SHA 与逐包 raw 见 `t11-lint-audit-results.json`。

| 范围 | 实际 warning / cap | error |
|---|---|---|
| `.` | 0 / 0 | 0 |
| `e2e` | 0 / 0 | 0 |
| `web/apps/web` | 69 / 69 | 0 |
| `web/packages/api-client` | 0 / 0 | 0 |
| `web/packages/constants` | 0 / 0 | 0 |
| `web/packages/editor` | 0 / 0 | 0 |
| `web/packages/hooks` | 3 / 3 | 0 |
| `web/packages/i18n` | 0 / 0 | 0 |
| `web/packages/propel` | 6 / 6 | 0 |
| `web/packages/services` | 0 / 0 | 0 |
| `web/packages/shared-state` | 0 / 0 | 0 |
| `web/packages/types` | 0 / 0 | 0 |
| `web/packages/ui` | 4 / 4 | 0 |
| `web/packages/utils` | 4 / 4 | 0 |

## 6. 关键词逐实例

阶段 `M4/closeout`，76 规则、13 条精确例外（共 15 处，来源注释占 3 处）；没有未覆盖、过期、失效或数量不符命中。每条源文件 hash、命中行/文本、理由及到期阶段见 `t11-keyword-ledger.json`，规则与样例未放宽。

| ID | 规则 / 精确实例 | 当前行 | 到期 |
|---|---|---|---|
| E01 | analytics / `web/apps/web/core/services/cycle.service.ts` / `analytics` | 13 | M6 |
| E02 | analytics / `web/apps/web/core/services/cycle.service.test.ts` / `analytics` | 8 | M6 |
| E03 | brand / `pnpm-workspace.yaml` / `Plane` | 2,3,164 | M9 |
| E04 | copy-hook / `web/apps/web/core/components/cycles/quick-actions.tsx` / `copyUrlToClipboard(` | 59 | M6/closeout |
| E05 | copy-hook / `web/apps/web/core/components/modules/links/list-item.tsx` / `copyTextToClipboard(` | 39 | M6/closeout |
| E06 | copy-hook / `web/apps/web/core/components/modules/quick-actions.tsx` / `copyUrlToClipboard(` | 59 | M6/closeout |
| E07 | copy-hook / `web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts` / `copyTextToClipboard(` | 55 | M6/closeout |
| E08 | copy-hook / `web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx` / `copyTextToClipboard(` | 85 | M6/closeout |
| E09 | copy-hook / `web/apps/web/core/components/inbox/content/inbox-issue-header.tsx` / `copyUrlToClipboard(` | 179 | M7/closeout |
| E10 | copy-hook / `web/apps/web/core/components/views/quick-actions.tsx` / `copyUrlToClipboard(` | 51 | M7/closeout |
| E11 | copy-hook / `web/apps/web/core/components/workspace/views/default-view-quick-action.tsx` / `copyUrlToClipboard(` | 34 | M7/closeout |
| E12 | copy-hook / `web/apps/web/core/components/workspace/views/quick-action.tsx` / `copyUrlToClipboard(` | 42 | M7/closeout |
| E13 | copy-hook / `web/apps/web/core/components/web-hooks/form/secret-key.tsx` / `copyTextToClipboard(` | 47 | M8/closeout |

迭代 analytics 的服务与传输测试同时归 M6；M6/M7 的复制入口和 M8 Webhook 密钥复制必须迁移现有安全 hook，来源注释保留到 M9。后续交接保留这些精确读者与期限，不把例外数量写成任意豁免。
