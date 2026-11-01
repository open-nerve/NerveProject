# M4 / P16 死成员逐项审计

状态：T4 原始脚本扫描与逐行结论；动态协议变异和后续 T11 终态须按 review 的真实收据核对。

## 1. 输入与判读

- 复制 M1 附录原脚本（deadsym/domains/lib/program），三文件 SHA 与 P15 归档完全相同；未改分类、未过滤测试。原始输入/报告/脚本哈希在 `.superpowers/sdd/P16-web-cleanup/dead-audit/`。
- 准备基线 453；T3 后首次扫描 449；T4 扫描 358（member/prop 合计）。449 行全部列出，91 行删除、358 行保留，无新增 identity。全仓 member 875→801、prop 365→334、export 21→2；剩余 export 均为未调用的编译负例入口。
- 四行准备差异：constants/issue/layout 的 i18n_title 从候选变为 T3 LayoutDropDown 的实际读取；旧 LayoutDropDown 的 disabledLayouts/value/selected 三个 prop 槽随旧 Dropdown 链删除，公开 value/disabledLayouts 仍由新 CustomSelect 包装实现。
- identity 为 kind + 文件 + symbol + 同名出现次序，行号仅定位。同名成员只删除列出的声明，不表示所有同名字段都已消失。每行链接其具名读者和验证；测试字段按实际替身、fixture 或断言读取核对，没有统一豁免。
- EditorRefApi、Commands 合并声明、NodeView/Plugin/React 回调保持实际合同；脚本未解析的动态读取不能当 dead。M5/M7 未实现领域只保留当前合同，不声明其页面已验收。

## 2. 本期整链清理

- `web/apps/web/core/components/core/modals/issue-search-modal-empty-state.tsx`：仅需要数量，issues:{id}[] 改 issueCount:number；三条调用链改传 length。
- `web/apps/web/core/components/issues/issue-layouts/utils.tsx`：columnId 无读者；移除返回字段、两种 DnD 写入与测试多余输入，group/subgroup 身份仍保留。
- `web/apps/web/core/store/issue/helpers/base-issues.store.ts`：clear 空实现和只为其存在的 any 参数整段移除。
- `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts`：getShouldClearIssues 无调用，整个函数移除。
- `web/apps/web/core/store/issue/issue_calendar_view.store.ts`：无消费者的 loader/error/activeWeekNumber，连同 interface、observable/computed 注册和专用 import 移除。
- `web/apps/web/core/store/issue/issue_kanban_view.store.ts`：旧开关/最小化/可拖拽 getter 链无读者，连同 rootStore 注入/注册移除；实际 getCanUserDragDrop 保留。
- `web/apps/web/core/store/issue/workspace-draft/issue.store.ts`：旧 getIssueIds 返回空数组的伪列表接口无读者，整方法移除；原生 draft 列表保留。
- `web/packages/editor/src/components/menus/bubble-menu/root.tsx`：仅删除颜色对象中不读的 key/label 和相反颜色字段，实际格式态/颜色值保留。
- `web/packages/editor/src/extensions/custom-image/components/toolbar/full-screen/root.tsx`：height 未读取，删除 props 与 toolbar/block 独占传递。
- `web/packages/editor/src/extensions/custom-image/types.ts`：deletedImageSet 仅声明/初始化，无读者，删除整条存储字段。
- `web/packages/editor/src/extensions/custom-link/extension.tsx`：此行属于无调用 toggleLink 或只写 isPreviewOpen/posToInsert；删命令/声明/写入；setLink 属性和 isBubbleMenuOpen 保留。
- `web/packages/editor/src/extensions/custom-list-keymap/list-helpers.ts`：遍历只消费 pos，删除 {node,pos} 中的 node，把内部列表改 number[] 并保留合法0。
- `web/packages/editor/src/extensions/table/table/table-view.tsx`：未读装饰/编辑器/位置/控制状态与构造参数移除；dom/contentDOM/update/ignoreMutation 留存。
- `web/packages/editor/src/extensions/table/table/table.ts`：删除无调用的命令声明、实现和独占导入；底层 deleteTable 仍由行列删除操作直接调用。
- `web/packages/editor/src/extensions/table/table/utilities/delete-key-shortcut.ts`：TableInfo.node 未读，只保留实际消费的位置/类型字段。
- `web/packages/editor/src/extensions/unique-id/plugin.ts`：allNodesInDoc 只读取 node，删除 pos 包装字段。
- `web/packages/editor/src/types/config.ts`：cancel/checkIfAssetExists 无消费者，删 use-editor-config 包装、FileService 专属方法和测试桩。
- `web/packages/shared-state/src/store/rich-filters/filter.ts`：FilterInstance.id 无消费者，删 interface/赋值/observable 及唯一 uuid 包依赖；registry entity 身份不变。
- `web/packages/types/src/issues.ts`：无写入的列禁投/错误提示链与旧 display_filters.type 整体删除；权限和实际过滤查询保留。
- `web/packages/types/src/issues/issue_attachment.ts`：四个类型组成无外部读者的孤岛，删除整文件及 barrel；不启用 M5 附件。
- `web/packages/types/src/rich-filters/field-types/index.ts`：defaultValue 无读者，删除四处声明以及 date/dateRange 幽灵泛型；派生日期操作符独占类型链一起删除。
- `web/packages/utils/src/rich-filters/factories/configs/shared.ts`：rightContent/tooltipContent 无任何工厂写入，删除两处声明与恒空 UI 读取链；实际筛选标签保持。
- `web/apps/web/core/components/common/layout/sidebar/property-list-item.tsx`：appendElement 从未传入，移除 prop 与恒空分支。
- `web/apps/web/core/components/editor/rich-text/description-input/loader.tsx`：className 未传入，删除专用 Props/cn 合并，保持原骨架样式。
- `web/apps/web/core/components/issues/issue-layouts/calendar/quick-add-issue-actions.tsx`：onOpen 从未传入，删除空通知链；真实菜单打开不变。
- `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-filters-selection.tsx`：ignoreGroupedFilters 未传入，删除 group/subgroup 传递及恒空排除分支。
- `web/apps/web/core/components/issues/links/data.test.tsx`：测试多余 ref 声明移除；Input 原生 rest props 保留并透传。
- `web/apps/web/core/components/project/filters.tsx`：四个可选展示参数无调用写入，删除声明和恒定样式分支。
- `web/apps/web/core/components/rich-filters/filter-item/property.tsx`：扩展 className 无传入，删专属类型与 merge；属性按钮样式不变。
- `web/apps/web/core/components/ui/loader/layouts/kanban-layout-loader.tsx`：cardsInEachColumn 无调用写入，固定原五列数据并使用稳定列名 key。
- `web/apps/web/core/layouts/default-layout/index.tsx`：gradient 无调用写入，移除条件样式参数，保留默认背景。
- `web/packages/editor/src/components/editors/editor-content.tsx`：className/children 无调用写入，删除两可选参数及独占渲染。
- `web/packages/editor/src/components/menus/floating-menu/root.tsx`：onClick 从未传入，删除空通知；实际上下文 toggle 留存。
- 其它 export 孤岛见 §5；表格六个列组件 onChange 第三个 updates 参数实际被 IssueColumn 忽略，连同 TSpreadsheetColumn 和多余对象构造删除。
- T4 所碰 custom-image/block 的两条既有交互 warning 交 T5 同模块键盘/语义测试收口；不是新增屏蔽，也不以全包 cap 合格声称此文件已零告警。

## 3. 当前候选与删除行

“旧→新”是首次 T4 行号到终态行号；— 表示该声明被删。每个文件下的读者说明适用于该行列出的具体成员/prop；删除行另列原因。


### F001 `web/apps/web/core/components/comments/data.test.tsx`

实际读取：RichTextEditor 替身把 editorProps.attributes/id/initialValue 送 textarea，onChange/onEnterKeyPress/ref 由 CommentCreate/EditForm 使用；useImperativeHandle 的三方法由真实表单调用。

验证：components/comments/data.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D001 | member | `attributes` #1 | 54→54 | 保留 |
| D002 | member | `clearEditor` #1 | 56→56 | 保留 |
| D003 | member | `setEditorValue` #1 | 57→57 | 保留 |
| D004 | member | `isEditorReadyToDiscard` #1 | 58→58 | 保留 |
| D295 | prop | `id` #1 | 49→49 | 保留 |
| D296 | prop | `initialValue` #1 | 50→50 | 保留 |
| D297 | prop | `onChange` #1 | 51→51 | 保留 |
| D298 | prop | `isSubmitting` #1 | 52→52 | 保留 |
| D299 | prop | `onEnterKeyPress` #1 | 53→53 | 保留 |
| D300 | prop | `editorProps` #1 | 54→54 | 保留 |
| D301 | prop | `ref` #1 | 55→55 | 保留 |

### F002 `web/apps/web/core/components/comments/permissions.test.tsx`

实际读取：翻译替身插值参数 name；权限拒绝后按钮/错误提示的文本断言。

验证：components/comments/permissions.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D005 | member | `name` #1 | 22→22 | 保留 |

### F003 `web/apps/web/core/components/comments/reaction.test.tsx`

实际读取：翻译替身插值 name；EmojiReactionGroup 的真实点击反馈断言。

验证：components/comments/reaction.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D006 | member | `name` #1 | 22→22 | 保留 |

### F004 `web/apps/web/core/components/common/layout-error-boundary.tsx`

实际读取：React class boundary 调用 getDerivedStateFromError/componentDidCatch/render；不是业务属性读取。

验证：React 类型检查；S2 正常布局，异常分支未另加测试；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D007 | member | `getDerivedStateFromError` #1 | 41→41 | 保留 |
| D008 | member | `componentDidCatch` #1 | 45→45 | 保留 |
| D009 | member | `render` #1 | 54→54 | 保留 |

### F005 `web/apps/web/core/components/common/layout/sidebar/property-list-item.tsx`

实际读取：SidebarPropertyListItem 把 className 传给 Icon FC；调用方传入图标组件。

验证：web 类型检查；执行收据见 review T4 的全类型/全前端/knip。

删除依据：appendElement 从未传入，移除 prop 与恒空分支。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D010 | member | `className` #1 | 11→11 | 保留 |
| D302 | prop | `appendElement` #1 | 14→— | 删除此声明 |

### F006 `web/apps/web/core/components/core/description-versions/root.test.tsx`

实际读取：Dropdown 替身 map 版本 id 并触发 onVersionClick；Modal 替身读取描述/active.id/error/retry/导航；deferred 的 version/resolve/reject 由并发测试逐次完成。

验证：components/core/description-versions/root.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D011 | member | `id` #1 | 38→38 | 保留 |
| D012 | member | `version` #1 | 65→65 | 保留 |
| D013 | member | `resolve` #1 | 65→65 | 保留 |
| D014 | member | `reject` #1 | 65→65 | 保留 |
| D305 | prop | `onVersionClick` #1 | 16→16 | 保留 |
| D306 | prop | `versions` #1 | 17→17 | 保留 |
| D307 | prop | `id` #1 | 17→17 | 保留 |
| D308 | prop | `activeVersionDescription` #1 | 37→37 | 保留 |
| D309 | prop | `activeVersionDetails` #1 | 38→38 | 保留 |
| D310 | prop | `handleClose` #1 | 39→39 | 保留 |
| D311 | prop | `handleNavigation` #1 | 40→40 | 保留 |
| D312 | prop | `error` #1 | 41→41 | 保留 |
| D313 | prop | `retry` #1 | 42→42 | 保留 |

### F007 `web/apps/web/core/components/core/modals/issue-search-modal-empty-state.tsx`

删除依据：仅需要数量，issues:{id}[] 改 issueCount:number；三条调用链改传 length。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D015 | member | `id` #1 | 20→— | 删除此声明 |

### F008 `web/apps/web/core/components/issues/delete-issue-modal.test.tsx`

实际读取：翻译器消费 identifier；AlertModal 替身读取内容/开关/提交和关闭回调，真实删除拥有者驱动。

验证：components/issues/delete-issue-modal.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D016 | member | `identifier` #1 | 23→23 | 保留 |
| D316 | prop | `handleSubmit` #1 | 54→54 | 保留 |
| D317 | prop | `handleClose` #1 | 55→55 | 保留 |
| D318 | prop | `isSubmitting` #1 | 56→56 | 保留 |
| D319 | prop | `isOpen` #1 | 57→57 | 保留 |
| D320 | prop | `content` #1 | 58→58 | 保留 |

### F009 `web/apps/web/core/components/issues/detail-copy.test.tsx`

实际读取：Tooltip 替身返回 children，真实编号复制按钮与反馈可见性断言。

验证：components/issues/detail-copy.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D017 | member | `children` #1 | 39→39 | 保留 |

### F010 `web/apps/web/core/components/issues/detail-entry.test.tsx`

实际读取：EmptyState 替身消费 primaryButton.text/onClick；Breadcrumb/ProjectAuthWrapper 消费 label/children。

验证：components/issues/detail-entry.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D018 | member | `text` #1 | 26→26 | 保留 |
| D019 | member | `onClick` #1 | 26→26 | 保留 |
| D020 | member | `children` #1 | 40→40 | 保留 |
| D322 | prop | `primaryButton` #1 | 26→26 | 保留 |
| D323 | prop | `label` #1 | 32→32 | 保留 |

### F011 `web/apps/web/core/components/issues/detail-fields.test.tsx`

实际读取：focus 记录的 invalid/description 由 activeElement 采集，再逐对象比较错误可达性。

验证：components/issues/detail-fields.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D021 | member | `invalid` #1 | 24→24 | 保留 |
| D022 | member | `description` #1 | 24→24 | 保留 |

### F012 `web/apps/web/core/components/issues/issue-detail/links/form.test.tsx`

实际读取：focus 记录 invalid/message 再 toEqual 验证字段关联提示。

验证：components/issues/issue-detail/links/form.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D023 | member | `invalid` #1 | 68→68 | 保留 |
| D024 | member | `message` #1 | 68→68 | 保留 |

### F013 `web/apps/web/core/components/issues/issue-layouts/calendar/quick-add-issue-actions.test.tsx`

实际读取：替身实际渲染 customQuickAddButton，触发真实日历快速添加拥有者。

验证：components/issues/issue-layouts/calendar/quick-add-issue-actions.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D025 | member | `customQuickAddButton` #1 | 24→24 | 保留 |

### F014 `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx`

实际读取：selectedExtraOptions[option.key] 读取 sub_issue/show_empty_groups。

验证：issue-layouts/filters/preferences-controls.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D026 | member | `sub_issue` #1 | 31→31 | 保留 |
| D027 | member | `show_empty_groups` #1 | 32→32 | 保留 |

### F015 `web/apps/web/core/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx`

实际读取：FilterDropdownHeader 的 cloneElement 为传入 SVG 写 className/strokeWidth，React 图标读取。

验证：dropdowns/basic-dropdowns.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D028 | member | `className` #1 | 18→18 | 保留 |
| D029 | member | `strokeWidth` #1 | 18→18 | 保留 |

### F016 `web/apps/web/core/components/issues/issue-layouts/group-consumers.test.tsx`

实际读取：VisibleObserver 的 observe/unobserve/disconnect 被真实可见性 hook 调用；children/issueId 由行渲染；it.each 解构 field/key/expected 比较 HTTP 体。

验证：components/issues/issue-layouts/group-consumers.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D030 | member | `children` #1 | 49→49 | 保留 |
| D031 | member | `observe` #1 | 65→65 | 保留 |
| D032 | member | `unobserve` #1 | 68→68 | 保留 |
| D033 | member | `disconnect` #1 | 71→71 | 保留 |
| D034 | member | `field` #1 | 202→202 | 保留 |
| D035 | member | `key` #1 | 202→202 | 保留 |
| D036 | member | `expected` #1 | 202→202 | 保留 |
| D336 | prop | `issueId` #1 | 53→53 | 保留 |

### F017 `web/apps/web/core/components/issues/issue-layouts/list-data.test.tsx`

实际读取：IntersectionObserver 替身 observe/unobserve/disconnect 由列表分页观察 hook 调用。

验证：components/issues/issue-layouts/list-data.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D037 | member | `observe` #1 | 33→33 | 保留 |
| D038 | member | `unobserve` #1 | 39→39 | 保留 |
| D039 | member | `disconnect` #1 | 40→40 | 保留 |

### F018 `web/apps/web/core/components/issues/issue-layouts/list/identifier-width.test.tsx`

实际读取：Tooltip/Row/ControlLink 替身透传 children/className/ref；viewport 读 children.props.issueId，宽度断言读 identifier。

验证：components/issues/issue-layouts/list/identifier-width.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D040 | member | `children` #1 | 45→45 | 保留 |
| D041 | member | `children` #2 | 58→58 | 保留 |
| D042 | member | `issueId` #1 | 58→58 | 保留 |
| D337 | prop | `identifier` #1 | 42→42 | 保留 |
| D338 | prop | `children` #1 | 48→48 | 保留 |
| D339 | prop | `className` #1 | 48→48 | 保留 |
| D340 | prop | `ref` #1 | 48→48 | 保留 |
| D341 | prop | `children` #2 | 53→53 | 保留 |

### F019 `web/apps/web/core/components/issues/issue-layouts/quick-add/root.errors.test.tsx`

实际读取：ResizeObserver 由 TextInput 调用；focus 记录 invalid/description 供错误反馈断言。

验证：components/issues/issue-layouts/quick-add/root.errors.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D043 | member | `observe` #1 | 30→30 | 保留 |
| D044 | member | `unobserve` #1 | 31→31 | 保留 |
| D045 | member | `disconnect` #1 | 32→32 | 保留 |
| D046 | member | `invalid` #1 | 90→90 | 保留 |
| D047 | member | `description` #1 | 90→90 | 保留 |

### F020 `web/apps/web/core/components/issues/issue-layouts/roots/archive-page.test.tsx`

实际读取：viewport wrapper 透传 children，真实归档拥有者进行请求和权限验证。

验证：components/issues/issue-layouts/roots/archive-page.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D048 | member | `children` #1 | 36→36 | 保留 |

### F021 `web/apps/web/core/components/issues/issue-layouts/roots/project-lifecycle.test.tsx`

实际读取：viewport 读取 as/children；autoScroll 替身按 element 绑定并回收监听集合。

验证：components/issues/issue-layouts/roots/project-lifecycle.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D049 | member | `element` #1 | 43→43 | 保留 |
| D344 | prop | `children` #1 | 40→40 | 保留 |
| D345 | prop | `as` #1 | 40→40 | 保留 |

### F022 `web/apps/web/core/components/issues/issue-layouts/route-scope.test.tsx`

实际读取：布局替身渲染 children，真实路由拥有者切换并销毁浮层。

验证：components/issues/issue-layouts/route-scope.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D050 | member | `children` #1 | 43→43 | 保留 |

### F023 `web/apps/web/core/components/issues/issue-layouts/utils.tsx`

实际读取：ISSUE_GROUP_BY_OPTIONS 经 useGroupByOptions 返回；group-by/sub-group-by 菜单读取 key/titleTranslationKey。

验证：issue-layouts/filters/preferences-controls.test.tsx；kanban/drop.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

删除依据：columnId 无读者；移除返回字段、两种 DnD 写入与测试多余输入，group/subgroup 身份仍保留。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D051 | member | `columnId` #1 | 58→— | 删除此声明 |
| D373 | prop | `key` #1 | 392→389 | 保留 |
| D374 | prop | `titleTranslationKey` #1 | 393→390 | 保留 |

### F024 `web/apps/web/core/components/issues/issue-modal/base.clone.test.tsx`

实际读取：ProjectSelect 的 disabled/value/onChange 由真实克隆表单传入；Modal 读 children；ResizeObserver 支持真实 TextInput。

验证：components/issues/issue-modal/base.clone.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D052 | member | `observe` #1 | 83→83 | 保留 |
| D053 | member | `unobserve` #1 | 84→84 | 保留 |
| D054 | member | `disconnect` #1 | 85→85 | 保留 |
| D375 | prop | `disabled` #1 | 52→52 | 保留 |
| D376 | prop | `value` #1 | 53→53 | 保留 |
| D377 | prop | `onChange` #1 | 54→54 | 保留 |
| D378 | prop | `children` #1 | 80→80 | 保留 |

### F025 `web/apps/web/core/components/issues/issue-modal/base.loading.test.tsx`

实际读取：ProjectSelect 读 disabled/value；Modal 读 isOpen/children；ResizeObserver 支持真实 TextInput 加载态。

验证：components/issues/issue-modal/base.loading.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D055 | member | `observe` #1 | 123→123 | 保留 |
| D056 | member | `unobserve` #1 | 124→124 | 保留 |
| D057 | member | `disconnect` #1 | 125→125 | 保留 |
| D379 | prop | `disabled` #1 | 97→97 | 保留 |
| D380 | prop | `value` #1 | 97→97 | 保留 |
| D381 | prop | `children` #1 | 119→119 | 保留 |
| D382 | prop | `isOpen` #1 | 119→119 | 保留 |

### F026 `web/apps/web/core/components/issues/issue-modal/fake-form.tsx`

实际读取：focusInvalid/activeField 将 tag/invalid/description 写入记录；form-fields.test.tsx 比较完整对象。

验证：issue-modal/form-fields.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D058 | member | `tag` #1 | 280→280 | 保留 |
| D059 | member | `invalid` #1 | 280→280 | 保留 |
| D060 | member | `description` #1 | 280→280 | 保留 |

### F027 `web/apps/web/core/components/issues/issue-modal/modal.tsx`

实际读取：primaryButtonText 交给 base/form；form 选择 default/loading 按钮文案。

验证：issue-modal/base.loading.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D061 | member | `default` #1 | 30→30 | 保留 |
| D062 | member | `loading` #1 | 31→31 | 保留 |

### F028 `web/apps/web/core/components/issues/peek-overview/header.tsx`

实际读取：菜单项 icon:FC 的 className 由渲染图标读取。

验证：peek-overview/host-actions.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D063 | member | `className` #1 | 38→38 | 保留 |

### F029 `web/apps/web/core/components/issues/peek-overview/host-actions.test.tsx`

实际读取：RouterLink 替身读 children/to；Tooltip 返回 children，真实 Header 驱动复制/打开/关闭。

验证：components/issues/peek-overview/host-actions.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D064 | member | `children` #1 | 29→29 | 保留 |
| D398 | prop | `children` #1 | 19→19 | 保留 |
| D399 | prop | `to` #1 | 19→19 | 保留 |

### F030 `web/apps/web/core/components/issues/title-input.tsx`

实际读取：onSubmit 调 issueOperations.update，name 是原生更新体字段。

验证：issues/detail-fields.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D065 | member | `name` #1 | 26→26 | 保留 |

### F031 `web/apps/web/core/components/rich-filters/add-filters/button.tsx`

实际读取：buttonConfig 展开给 AddFilterDropdown，后者解构 defaultOpen 交 CustomSearchSelect。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D066 | member | `defaultOpen` #1 | 27→27 | 保留 |

### F032 `web/apps/web/core/components/work-item-filters/filter-config.test.tsx`

实际读取：Chips 读取 filter.allConditions 并传给真实 FilterItem。

验证：components/work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D067 | member | `filter` #1 | 30→30 | 保留 |

### F033 `web/apps/web/core/components/work-item-filters/filters-hoc/base.tsx`

实际读取：render props children({filter}) 向 project/workspace-level 消费者提供过滤实例。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D068 | member | `filter` #1 | 46→46 | 保留 |

### F034 `web/apps/web/core/hooks/editor/use-editor-mention.test.tsx`

实际读取：Probe 展开 options 给真实 useEditorMention，rerender 改 scope/查询并检查结果。

验证：hooks/editor/use-editor-mention.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D069 | member | `options` #1 | 25→25 | 保留 |

### F035 `web/apps/web/core/lib/issue-order.test.ts`

实际读取：JSON fixture 的 name_keys/scenarios 经 map/it.each 转行排序；reviver.context.source 保留大整数；samples 的 id/priority/rank 直接比较排序结果。

验证：lib/issue-order.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D070 | member | `order` #1 | 19→19 | 保留 |
| D071 | member | `rows` #1 | 20→20 | 保留 |
| D072 | member | `id` #1 | 20→20 | 保留 |
| D073 | member | `names` #1 | 20→20 | 保留 |
| D074 | member | `key` #1 | 20→20 | 保留 |
| D075 | member | `ascending_ids` #1 | 21→21 | 保留 |
| D076 | member | `descending_ids` #1 | 22→22 | 保留 |
| D077 | member | `name_keys` #1 | 26→26 | 保留 |
| D078 | member | `name` #1 | 26→26 | 保留 |
| D079 | member | `key` #2 | 26→26 | 保留 |
| D080 | member | `scenarios` #1 | 26→26 | 保留 |
| D081 | member | `source` #1 | 28→28 | 保留 |
| D082 | member | `id` #2 | 144→144 | 保留 |
| D083 | member | `priority` #1 | 144→144 | 保留 |
| D084 | member | `rank` #1 | 144→144 | 保留 |

### F036 `web/apps/web/core/services/issue/collaboration.service.test.ts`

实际读取：Case 表逐条执行 run，按 endpoint/body/query/status/answer 校验 HTTP 与解析；name 为 it.each 标题。

验证：services/issue/collaboration.service.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D085 | member | `name` #1 | 24→24 | 保留 |
| D086 | member | `run` #1 | 25→25 | 保留 |
| D087 | member | `endpoint` #1 | 26→26 | 保留 |
| D088 | member | `body` #1 | 27→27 | 保留 |
| D089 | member | `answer` #1 | 28→28 | 保留 |
| D090 | member | `status` #1 | 29→29 | 保留 |
| D091 | member | `query` #1 | 30→30 | 保留 |

### F037 `web/apps/web/core/services/issue/issue-detail.service.test.ts`

实际读取：Case 表逐条执行 run，按 endpoint/body/status/answer 校验 HTTP 与返回；name 为用例标题。

验证：services/issue/issue-detail.service.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D092 | member | `name` #1 | 42→42 | 保留 |
| D093 | member | `endpoint` #1 | 43→43 | 保留 |
| D094 | member | `run` #1 | 44→44 | 保留 |
| D095 | member | `body` #1 | 45→45 | 保留 |
| D096 | member | `answer` #1 | 46→46 | 保留 |
| D097 | member | `status` #1 | 47→47 | 保留 |

### F038 `web/apps/web/core/services/issue/issue-list.service.test.ts`

实际读取：it.each 解构 name/page，校验各分页数据拒绝或接受。

验证：services/issue/issue-list.service.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D098 | member | `name` #1 | 146→146 | 保留 |
| D099 | member | `page` #1 | 146→146 | 保留 |

### F039 `web/apps/web/core/services/issue/issue-search.service.test.ts`

实际读取：查询输入表按 query 发请求并比较 entries，验证编码与作用域。

验证：services/issue/issue-search.service.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D100 | member | `query` #1 | 11→11 | 保留 |
| D101 | member | `entries` #1 | 11→11 | 保留 |

### F040 `web/apps/web/core/services/issue/workspace_draft.service.test.ts`

实际读取：Case 表逐条执行 run，比较 endpoint/body/query/status/answer；name 为 it.each 标题。

验证：services/issue/workspace_draft.service.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D102 | member | `name` #1 | 8→8 | 保留 |
| D103 | member | `endpoint` #1 | 9→9 | 保留 |
| D104 | member | `run` #1 | 10→10 | 保留 |
| D105 | member | `answer` #1 | 11→11 | 保留 |
| D106 | member | `body` #1 | 12→12 | 保留 |
| D107 | member | `query` #1 | 13→13 | 保留 |
| D108 | member | `status` #1 | 14→14 | 保留 |

### F041 `web/apps/web/core/store/issue/helpers/base-issues.store.ts`

删除依据：clear 空实现和只为其存在的 any 参数整段移除。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D109 | member | `clear` #1 | 58→— | 删除此声明 |

### F042 `web/apps/web/core/store/issue/helpers/issue-filter-helper.store.ts`

实际读取：Project/Archived/Profile filter.store 实现并读取 filters/computed*；loadFromStorage 读 kanban_filters。

验证：store/issue/{project,archived,profile}/filter.store.test.ts；执行收据见 review T4 的全类型/全前端/knip。

删除依据：getShouldClearIssues 无调用，整个函数移除。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D110 | member | `filters` #1 | 118→118 | 保留 |
| D111 | member | `computedIssueFilters` #1 | 124→124 | 保留 |
| D112 | member | `computedDisplayFilters` #1 | 129→129 | 保留 |
| D113 | member | `computedDisplayProperties` #1 | 133→133 | 保留 |
| D114 | member | `kanban_filters` #1 | 260→260 | 保留 |
| D115 | member | `getShouldClearIssues` #1 | 316→— | 删除此声明 |

### F043 `web/apps/web/core/store/issue/issue_calendar_view.store.ts`

实际读取：CalendarGridHeader 读 filters.activeMonthDate/activeWeekDate；updateCalendarPayload/用户偏好 reaction 调 regenerateCalendar。

验证：components/issues/issue-layouts/calendar/{view,selection,quick-add-issue-actions}.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

删除依据：无消费者的 loader/error/activeWeekNumber，连同 interface、observable/computed 注册和专用 import 移除。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D116 | member | `activeMonthDate` #1 | 25→25 | 保留 |
| D117 | member | `activeWeekDate` #1 | 25→25 | 保留 |
| D118 | member | `regenerateCalendar` #1 | 27→27 | 保留 |
| D119 | member | `activeWeekNumber` #1 | 35→— | 删除此声明 |
| D120 | member | `loader` #1 | 41→— | 删除此声明 |
| D121 | member | `error` #1 | 43→— | 删除此声明 |
| D122 | member | `activeWeekNumber` #2 | 117→— | 删除此声明 |

### F044 `web/apps/web/core/store/issue/issue_kanban_view.store.ts`

删除依据：旧开关/最小化/可拖拽 getter 链无读者，连同 rootStore 注入/注册移除；实际 getCanUserDragDrop 保留。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D123 | member | `kanBanToggle` #1 | 16→— | 删除此声明 |
| D124 | member | `groupByHeaderMinMax` #1 | 17→— | 删除此声明 |
| D125 | member | `subgroupByIssuesVisibility` #1 | 18→— | 删除此声明 |
| D126 | member | `canUserDragDropVertically` #1 | 26→— | 删除此声明 |
| D127 | member | `canUserDragDropHorizontally` #1 | 27→— | 删除此声明 |
| D128 | member | `handleKanBanToggle` #1 | 29→— | 删除此声明 |
| D129 | member | `groupByHeaderMinMax` #2 | 35→— | 删除此声明 |
| D130 | member | `subgroupByIssuesVisibility` #2 | 36→— | 删除此声明 |
| D131 | member | `rootStore` #1 | 41→— | 删除此声明 |
| D132 | member | `canUserDragDropVertically` #2 | 80→— | 删除此声明 |
| D133 | member | `canUserDragDropHorizontally` #2 | 84→— | 删除此声明 |
| D134 | member | `handleKanBanToggle` #2 | 88→— | 删除此声明 |

### F045 `web/apps/web/core/store/issue/issue-details/activity.store.ts`

实际读取：同目录 activity.store.test.ts 调 getActivitiesByIssueId 核分页顺序和权限失效隐藏；是可观察查询接口。

验证：activity.store.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D135 | member | `getActivitiesByIssueId` #1 | 24→24 | 保留 |

### F046 `web/apps/web/core/store/issue/issue-details/comment.store.ts`

实际读取：同目录 comment.store.test.ts:70 读 isLoading，验证拒绝后解除加载态。

验证：comment.store.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D136 | member | `isLoading` #1 | 40→40 | 保留 |

### F047 `web/apps/web/core/store/issue/issue-details/description-versions.store.ts`

实际读取：同目录 description-versions.store.test.ts 读 getVersion，核版本隔离/空HTML/权限失效。

验证：description-versions.store.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D137 | member | `getVersion` #1 | 30→30 | 保留 |

### F048 `web/apps/web/core/store/issue/workspace-draft/issue.store.ts`

删除依据：旧 getIssueIds 返回空数组的伪列表接口无读者，整方法移除；原生 draft 列表保留。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D138 | member | `getIssueIds` #1 | 187→— | 删除此声明 |

### F049 `web/packages/editor/src/components/menus/block-menu.tsx`

实际读取：Tippy getReferenceClientRect 取虚拟元素 getBoundingClientRect，定位块菜单。

验证：editor 类型；真实 editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D139 | member | `getBoundingClientRect` #1 | 42→42 | 保留 |

### F050 `web/packages/editor/src/components/menus/bubble-menu/root.tsx`

实际读取：格式按钮 editorState[item.key] 读取 bold/italic/underline/strikethrough；颜色样式读取 textColor/backgroundColor。

验证：editor-interaction.test.ts；editor 类型；执行收据见 review T4 的全类型/全前端/knip。

删除依据：仅删除颜色对象中不读的 key/label 和相反颜色字段，实际格式态/颜色值保留。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D140 | member | `bold` #1 | 43→43 | 保留 |
| D141 | member | `italic` #1 | 44→44 | 保留 |
| D142 | member | `underline` #1 | 45→45 | 保留 |
| D143 | member | `strikethrough` #1 | 46→46 | 保留 |
| D144 | member | `key` #1 | 52→— | 删除此声明 |
| D145 | member | `label` #1 | 53→— | 删除此声明 |
| D146 | member | `backgroundColor` #1 | 55→— | 删除此声明 |
| D147 | member | `key` #2 | 60→— | 删除此声明 |
| D148 | member | `label` #2 | 61→— | 删除此声明 |
| D149 | member | `textColor` #1 | 62→— | 删除此声明 |

### F051 `web/packages/editor/src/editor-interaction.test.ts`

实际读取：createEditor 解构 editable/content/disabledExtensions/attributes 传真实 Editor；blockIds 收集 type/id 后逐条断言。

验证：本文件全部真实编辑器测试；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D150 | member | `editable` #1 | 66→64 | 保留 |
| D151 | member | `content` #1 | 67→65 | 保留 |
| D152 | member | `disabledExtensions` #1 | 68→66 | 保留 |
| D153 | member | `attributes` #1 | 69→67 | 保留 |
| D154 | member | `type` #1 | 122→120 | 保留 |
| D155 | member | `id` #1 | 122→120 | 保留 |

### F052 `web/packages/editor/src/extensions/callout/extension-config.ts`

实际读取：TipTap getSchemaByResolvedExtensions 读属性 default/parseHTML；callout HTML/JSON 双向转换。

验证：extensions/callout/extension-config.test.ts；sanitization-roundtrip.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D156 | member | `default` #1 | 43→43 | 保留 |
| D157 | member | `parseHTML` #1 | 44→44 | 保留 |

### F053 `web/packages/editor/src/extensions/code-inline/index.tsx`

实际读取：与已锁定 @tiptap/extension-code Commands 合并；内置 toggle/input/paste 命令及自定义链同型。

验证：TS2717 删除反证；editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D158 | member | `setCode` #1 | 21→21 | 保留 |
| D159 | member | `toggleCode` #1 | 25→25 | 保留 |
| D160 | member | `unsetCode` #1 | 29→29 | 保留 |

### F054 `web/packages/editor/src/extensions/code/code-block.ts`

实际读取：与 @tiptap/extension-code-block Commands 合并；set/toggle 的 language 传 setNode/toggleNode。

验证：TS2717 删除反证；editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D161 | member | `setCodeBlock` #1 | 40→40 | 保留 |
| D162 | member | `language` #1 | 40→40 | 保留 |
| D163 | member | `toggleCodeBlock` #1 | 44→44 | 保留 |
| D164 | member | `language` #2 | 44→44 | 保留 |

### F055 `web/packages/editor/src/extensions/custom-image/components/toolbar/full-screen/root.tsx`

删除依据：height 未读取，删除 props 与 toolbar/block 独占传递。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D165 | member | `height` #1 | 18→— | 删除此声明 |

### F056 `web/packages/editor/src/extensions/custom-image/extension-config.ts`

实际读取：TipTap schema 读 CustomImageAttributes 的 default；已有图像扩展需保持序列化。

验证：sanitization-roundtrip.test.ts；M4 描述继续禁用 image；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D166 | member | `default` #1 | 51→51 | 保留 |

### F057 `web/packages/editor/src/extensions/custom-image/types.ts`

实际读取：uploader/utils 按 uploadEntity.event 分派、读 file/hasOpenedFileInputOnce，extension.utils 用 fileMap 跟踪在途上传。

验证：editor 类型；sanitization-roundtrip.test.ts；未启用 M4 图片上传；执行收据见 review T4 的全类型/全前端/knip。

删除依据：deletedImageSet 仅声明/初始化，无读者，删除整条存储字段。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D167 | member | `event` #1 | 51→51 | 保留 |
| D168 | member | `event` #2 | 51→51 | 保留 |
| D169 | member | `file` #1 | 51→51 | 保留 |
| D170 | member | `hasOpenedFileInputOnce` #1 | 51→51 | 保留 |
| D171 | member | `fileMap` #1 | 68→68 | 保留 |
| D172 | member | `deletedImageSet` #1 | 69→— | 删除此声明 |

### F058 `web/packages/editor/src/extensions/custom-link/extension.tsx`

实际读取：setLink 将 href/target/rel/class 传 setMark；bubble-menu/link-view-container 通过 storage.link 读 isBubbleMenuOpen。

验证：editor-interaction.test.ts；sanitization-roundtrip.test.ts；执行收据见 review T4 的全类型/全前端/knip。

删除依据：此行属于无调用 toggleLink 或只写 isPreviewOpen/posToInsert；删命令/声明/写入；setLink 属性和 isBubbleMenuOpen 保留。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D173 | member | `href` #1 | 57→57 | 保留 |
| D174 | member | `target` #1 | 58→58 | 保留 |
| D175 | member | `rel` #1 | 59→59 | 保留 |
| D176 | member | `class` #1 | 60→60 | 保留 |
| D177 | member | `toggleLink` #1 | 65→— | 删除此声明 |
| D178 | member | `href` #2 | 66→— | 删除此声明 |
| D179 | member | `target` #2 | 67→— | 删除此声明 |
| D180 | member | `rel` #2 | 68→— | 删除此声明 |
| D181 | member | `class` #2 | 69→— | 删除此声明 |
| D182 | member | `isPreviewOpen` #1 | 83→— | 删除此声明 |
| D183 | member | `posToInsert` #1 | 84→— | 删除此声明 |
| D184 | member | `from` #1 | 84→— | 删除此声明 |
| D185 | member | `to` #1 | 84→— | 删除此声明 |
| D186 | member | `isBubbleMenuOpen` #1 | 85→74 | 保留 |

### F059 `web/packages/editor/src/extensions/custom-list-keymap/list-helpers.ts`

删除依据：遍历只消费 pos，删除 {node,pos} 中的 node，把内部列表改 number[] 并保留合法0。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D187 | member | `node` #1 | 125→— | 删除此声明 |

### F060 `web/packages/editor/src/extensions/emoji/components/emojis-list.tsx`

实际读取：SuggestionProps.items 经 items.map 读 name/emoji/shortcodes；suggestion.ts tags 匹配；ReactRenderer.ref.onKeyDown 接受插件按键。

验证：editor-interaction.test.ts；editor 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D188 | member | `onKeyDown` #1 | 22→22 | 保留 |
| D189 | member | `name` #1 | 25→25 | 保留 |
| D435 | prop | `name` #1 | 15→15 | 保留 |
| D436 | prop | `emoji` #1 | 16→16 | 保留 |
| D437 | prop | `shortcodes` #1 | 17→17 | 保留 |
| D438 | prop | `tags` #1 | 18→18 | 保留 |

### F061 `web/packages/editor/src/extensions/emoji/emoji.ts`

实际读取：helper/editor-commands 与 suggestion 命令链 setEmoji({emoji})，节点 attrs 渲染。

验证：editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D190 | member | `emoji` #1 | 31→31 | 保留 |
| D191 | member | `setEmoji` #1 | 31→31 | 保留 |

### F062 `web/packages/editor/src/extensions/extensions.test.ts`

实际读取：editor fixture 解构 editable/enableHistory/disabledExtensions 生成真实扩展，检查只读/禁用/撤销边界。

验证：本文件全部测试；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D192 | member | `editable` #1 | 32→30 | 保留 |
| D193 | member | `enableHistory` #1 | 32→30 | 保留 |
| D194 | member | `disabledExtensions` #1 | 32→30 | 保留 |

### F063 `web/packages/editor/src/extensions/horizontal-rule.ts`

实际读取：helper/editor-commands 的 divider 分支调 setHorizontalRule；TipTap 动态 Commands 合同。

验证：editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D195 | member | `setHorizontalRule` #1 | 22→22 | 保留 |

### F064 `web/packages/editor/src/extensions/keymap.ts`

实际读取：TipTap Commands 分组 customKeymap 映射 selectTextWithinNodeBoundaries；本扩展 Mod-a 调用。

验证：editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D196 | member | `customKeymap` #1 | 17→17 | 保留 |

### F065 `web/packages/editor/src/extensions/table/plugins/selection-outline/plugin.ts`

实际读取：PluginKey.getState(state).decorations 被 props.decorations 返回到 ProseMirror。

验证：editor-interaction.test.ts；editor 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D197 | member | `decorations` #1 | 17→17 | 保留 |

### F066 `web/packages/editor/src/extensions/table/table/table-view.tsx`

实际读取：ProseMirror NodeView 协议读取 dom/contentDOM、调用 update/ignoreMutation；DOM 编辑进入真实表格。

验证：editor-interaction.test.ts 的 browser edit inside a table；M12；执行收据见 review T4 的全类型/全前端/knip。

删除依据：未读装饰/编辑器/位置/控制状态与构造参数移除；dom/contentDOM/update/ignoreMutation 留存。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D198 | member | `decorations` #1 | 16→— | 删除此声明 |
| D199 | member | `editor` #1 | 17→— | 删除此声明 |
| D200 | member | `getPos` #1 | 18→— | 删除此声明 |
| D201 | member | `hoveredCell` #1 | 19→— | 删除此声明 |
| D202 | member | `controls` #1 | 25→— | 删除此声明 |
| D203 | member | `dom` #1 | 27→21 | 保留 |
| D204 | member | `contentDOM` #1 | 31→25 | 保留 |
| D205 | member | `update` #1 | 69→53 | 保留 |
| D206 | member | `ignoreMutation` #1 | 92→75 | 保留 |

### F067 `web/packages/editor/src/extensions/table/table/table.ts`

实际读取：TipTap getExtensionField 绑定 name/options/storage/parent 的 this 上下文，扩展节点解析/插入合同。

验证：editor-interaction.test.ts；sanitization-roundtrip.test.ts；执行收据见 review T4 的全类型/全前端/knip。

删除依据：删除无调用的命令声明、实现和独占导入；底层 deleteTable 仍由行列删除操作直接调用。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D207 | member | `deleteTable` #1 | 63→— | 删除此声明 |
| D208 | member | `mergeCells` #1 | 64→— | 删除此声明 |
| D209 | member | `splitCell` #1 | 65→— | 删除此声明 |
| D210 | member | `toggleHeaderCell` #1 | 68→— | 删除此声明 |
| D211 | member | `mergeOrSplit` #1 | 70→— | 删除此声明 |
| D212 | member | `setCellAttribute` #1 | 71→— | 删除此声明 |
| D213 | member | `fixTables` #1 | 74→— | 删除此声明 |
| D214 | member | `name` #1 | 83→70 | 保留 |
| D215 | member | `options` #1 | 84→71 | 保留 |
| D216 | member | `storage` #1 | 85→72 | 保留 |
| D217 | member | `parent` #1 | 86→73 | 保留 |

### F068 `web/packages/editor/src/extensions/table/table/utilities/delete-key-shortcut.ts`

删除依据：TableInfo.node 未读，只保留实际消费的位置/类型字段。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D218 | member | `node` #1 | 23→— | 删除此声明 |

### F069 `web/packages/editor/src/extensions/unique-id/extension.ts`

实际读取：sanitization-roundtrip.test.ts 的 generateUniqueID({node,pos}) 读取上下文；plugin/utils 调配置函数。

验证：sanitization-roundtrip.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D219 | member | `node` #1 | 17→17 | 保留 |
| D220 | member | `pos` #1 | 18→18 | 保留 |

### F070 `web/packages/editor/src/extensions/unique-id/plugin.ts`

删除依据：allNodesInDoc 只读取 node，删除 pos 包装字段。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D221 | member | `pos` #1 | 42→— | 删除此声明 |

### F071 `web/packages/editor/src/extensions/utility.ts`

实际读取：use-editor-storage、custom-image upload-status 与工具栏通过 editor.storage.utility 读取 assetsUploadStatus/uploadInProgress/isTouchDevice。

验证：editor-interaction.test.ts；editor 类型；M4 image 保持禁用；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D222 | member | `assetsUploadStatus` #1 | 50→50 | 保留 |
| D223 | member | `uploadInProgress` #1 | 51→51 | 保留 |
| D224 | member | `isTouchDevice` #1 | 53→53 | 保留 |

### F072 `web/packages/editor/src/helpers/tippy.ts`

实际读取：slash-commands 命令列表用返回 section/item；Suggestion ReactRenderer.ref.onKeyDown({event})。

验证：extensions/slash-commands/command-items-list.test.ts；editor-interaction.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D225 | member | `event` #1 | 8→8 | 保留 |
| D226 | member | `section` #1 | 28→28 | 保留 |
| D227 | member | `item` #1 | 29→29 | 保留 |

### F073 `web/packages/editor/src/sanitization-roundtrip.test.ts`

实际读取：generateUniqueID 解构 node.type.name/pos；JSON cases 遍历 name/input/sanitized/reserialized/editor_document/preserved_elements/selector/attributes/styles/text/editable 并断言往返。

验证：本文件服务端黄金样例的编辑器往返测试；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D228 | member | `node` #1 | 47→45 | 保留 |
| D229 | member | `type` #1 | 47→45 | 保留 |
| D230 | member | `name` #1 | 47→45 | 保留 |
| D231 | member | `pos` #1 | 47→45 | 保留 |
| D232 | member | `name` #2 | 60→58 | 保留 |
| D233 | member | `input` #1 | 61→59 | 保留 |
| D234 | member | `sanitized` #1 | 62→60 | 保留 |
| D235 | member | `reserialized` #1 | 63→61 | 保留 |
| D236 | member | `editor_document` #1 | 64→62 | 保留 |
| D237 | member | `preserved_elements` #1 | 65→63 | 保留 |
| D238 | member | `selector` #1 | 66→64 | 保留 |
| D239 | member | `attributes` #1 | 67→65 | 保留 |
| D240 | member | `styles` #1 | 68→66 | 保留 |
| D241 | member | `text` #1 | 69→67 | 保留 |
| D242 | member | `editable` #1 | 71→69 | 保留 |
| D243 | member | `cases` #1 | 73→71 | 保留 |

### F074 `web/packages/editor/src/types/config.ts`

删除依据：cancel/checkIfAssetExists 无消费者，删 use-editor-config 包装、FileService 专属方法和测试桩。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D244 | member | `cancel` #1 | 9→— | 删除此声明 |
| D245 | member | `checkIfAssetExists` #1 | 10→— | 删除此声明 |

### F075 `web/packages/editor/src/types/editor.ts`

实际读取：TCommandWithProps 映射后由 helpers/editor-commands 消费；EditorRefApi 由 helpers/editor-ref 实现并经 useImperativeHandle 暴露，P14 公共合同。

验证：editor-interaction.test.ts ref API/commands 真实编辑器断言；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D246 | member | `image` #1 | 46→46 | 保留 |
| D247 | member | `savedSelection` #1 | 47→47 | 保留 |
| D248 | member | `text-color` #1 | 49→49 | 保留 |
| D249 | member | `link` #1 | 52→52 | 保留 |
| D250 | member | `background-color` #1 | 56→56 | 保留 |
| D251 | member | `text-align` #1 | 59→59 | 保留 |
| D252 | member | `blur` #1 | 74→74 | 保留 |
| D253 | member | `createSelectionAtCursorPosition` #1 | 76→76 | 保留 |
| D254 | member | `focus` #1 | 78→78 | 保留 |
| D255 | member | `getAttributesWithExtendedMark` #1 | 79→79 | 保留 |
| D256 | member | `getCoordsFromPos` #1 | 83→83 | 保留 |
| D257 | member | `getCurrentCursorPosition` #1 | 84→84 | 保留 |
| D258 | member | `copyMarkdownToClipboard` #1 | 86→86 | 保留 |
| D259 | member | `getSelectedText` #1 | 87→87 | 保留 |
| D260 | member | `insertText` #1 | 88→88 | 保留 |
| D261 | member | `redo` #1 | 93→93 | 保留 |
| D262 | member | `scrollToNodeViaDOMCoordinates` #1 | 94→94 | 保留 |
| D263 | member | `setFocusAtPosition` #1 | 96→96 | 保留 |
| D264 | member | `undo` #1 | 97→97 | 保留 |

### F076 `web/packages/shared-state/src/store/rich-filters/filter.ts`

实际读取：hasChanges 比较 initialFilterExpression；mutate/条件更新调 onExpressionChange，work-item-filters/filter.store 注册回调。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

删除依据：FilterInstance.id 无消费者，删 interface/赋值/observable 及唯一 uuid 包依赖；registry entity 身份不变。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D265 | member | `id` #1 | 70→— | 删除此声明 |
| D266 | member | `initialFilterExpression` #1 | 71→68 | 保留 |
| D267 | member | `onExpressionChange` #1 | 75→72 | 保留 |
| D268 | member | `id` #2 | 131→— | 删除此声明 |

### F077 `web/packages/types/src/issues.ts`

删除依据：无写入的列禁投/错误提示链与旧 display_filters.type 整体删除；权限和实际过滤查询保留。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D269 | member | `type` #1 | 54→— | 删除此声明 |
| D444 | prop | `isDropDisabled` #1 | 44→— | 删除此声明 |
| D445 | prop | `dropErrorMessage` #1 | 45→— | 删除此声明 |

### F078 `web/packages/types/src/issues/issue_attachment.ts`

删除依据：四个类型组成无外部读者的孤岛，删除整文件及 barrel；不启用 M5 附件。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D270 | member | `id` #1 | 10→— | 删除此声明 |
| D271 | member | `attributes` #1 | 11→— | 删除此声明 |
| D272 | member | `name` #1 | 12→— | 删除此声明 |
| D273 | member | `size` #1 | 13→— | 删除此声明 |
| D274 | member | `asset_url` #1 | 15→— | 删除此声明 |
| D275 | member | `issue_id` #1 | 16→— | 删除此声明 |
| D276 | member | `updated_at` #1 | 18→— | 删除此声明 |
| D277 | member | `updated_by` #1 | 19→— | 删除此声明 |
| D278 | member | `created_by` #1 | 20→— | 删除此声明 |
| D279 | member | `attachment` #1 | 24→— | 删除此声明 |

### F079 `web/packages/types/src/issues/issue_sub_issues.ts`

实际读取：sub_issues.store 按 state.group 动态写读分布；sub-issues/progress 读 completed 和总数。

验证：store/issue/issue-details/sub_issues.store.test.ts；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D280 | member | `backlog` #1 | 10→10 | 保留 |
| D281 | member | `unstarted` #1 | 11→11 | 保留 |
| D282 | member | `started` #1 | 12→12 | 保留 |
| D283 | member | `cancelled` #1 | 14→14 | 保留 |

### F080 `web/packages/types/src/rich-filters/field-types/index.ts`

删除依据：defaultValue 无读者，删除四处声明以及 date/dateRange 幽灵泛型；派生日期操作符独占类型链一起删除。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D284 | member | `defaultValue` #1 | 36→— | 删除此声明 |
| D285 | member | `defaultValue` #2 | 47→— | 删除此声明 |
| D286 | member | `defaultValue` #3 | 59→— | 删除此声明 |
| D287 | member | `defaultValue` #4 | 71→— | 删除此声明 |

### F081 `web/packages/utils/src/rich-filters/factories/configs/core.ts`

实际读取：createMultiSelectFilterFieldConfig 写 singleValueOperator；FilterConfig.getDisplayOperatorByValue 读取。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D288 | member | `singleValueOperator` #1 | 32→32 | 保留 |

### F082 `web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts`

实际读取：additionalOptions 展开给 properties/member 的选项工厂，再进入 getOptions 列表。

验证：work-item-filters/filter-config.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D289 | member | `additionalOptions` #1 | 24→24 | 保留 |

### F083 `web/packages/utils/src/rich-filters/factories/configs/shared.ts`

实际读取：createFilterConfig 参数展开到实际配置；FilterConfig 构造读取 isEnabled/label。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

删除依据：rightContent/tooltipContent 无任何工厂写入，删除两处声明与恒空 UI 读取链；实际筛选标签保持。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D290 | member | `isEnabled` #1 | 35→35 | 保留 |
| D291 | member | `label` #1 | 36→36 | 保留 |
| D292 | member | `rightContent` #1 | 37→— | 删除此声明 |
| D293 | member | `tooltipContent` #1 | 38→— | 删除此声明 |

### F084 `web/packages/utils/src/work-item-filters/configs/filters/label.ts`

实际读取：additionalOptions 经 createLabelFilterConfig 展开，core.getOptions 并入结果。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D294 | member | `additionalOptions` #1 | 21→21 | 保留 |

### F085 `web/apps/web/core/components/core/description-versions/modal.test.tsx`

实际读取：RichTextEditor 替身渲染 initialValue 并向 ref 暴露复制行为，真实 Modal 驱动等待与恢复。

验证：components/core/description-versions/modal.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D303 | prop | `initialValue` #1 | 22→22 | 保留 |
| D304 | prop | `ref` #1 | 23→23 | 保留 |

### F086 `web/apps/web/core/components/core/filters/date-filter-modal.test.tsx`

实际读取：Calendar 替身调用 onSelect 提供反向区间，真实表单验证错误边界。

验证：components/core/filters/date-filter-modal.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D314 | prop | `onSelect` #1 | 16→16 | 保留 |

### F087 `web/apps/web/core/components/editor/rich-text/description-input/loader.tsx`

删除依据：className 未传入，删除专用 Props/cn 合并，保持原骨架样式。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D315 | prop | `className` #1 | 12→— | 删除此声明 |

### F088 `web/apps/web/core/components/issues/detail-cache.test.tsx`

实际读取：SiblingItem 替身渲染 issueId，实际 SWR/store 处理权限缓存。

验证：components/issues/detail-cache.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D321 | prop | `issueId` #1 | 28→28 | 保留 |

### F089 `web/apps/web/core/components/issues/detail-host.test.tsx`

实际读取：真实 Detail/Peek 根向替身传 issueId，Probe 保存并测试弹窗所有者。

验证：components/issues/detail-host.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D324 | prop | `issueId` #1 | 35→35 | 保留 |
| D325 | prop | `issueId` #2 | 39→39 | 保留 |

### F090 `web/apps/web/core/components/issues/issue-detail/reactions/issue.test.tsx`

实际读取：Picker 替身读 label/onChange/disabled；Group map 读取 emoji/count/users 并调用 onReactionClick，真实 reactions store 接收确认。

验证：components/issues/issue-detail/reactions/issue.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D326 | prop | `label` #1 | 29→29 | 保留 |
| D327 | prop | `onChange` #1 | 30→30 | 保留 |
| D328 | prop | `disabled` #1 | 31→31 | 保留 |
| D329 | prop | `reactions` #1 | 44→44 | 保留 |
| D330 | prop | `emoji` #1 | 44→44 | 保留 |
| D331 | prop | `count` #1 | 44→44 | 保留 |
| D332 | prop | `users` #1 | 44→44 | 保留 |
| D333 | prop | `onReactionClick` #1 | 45→45 | 保留 |

### F091 `web/apps/web/core/components/issues/issue-layouts/calendar/quick-add-issue-actions.tsx`

删除依据：onOpen 从未传入，删除空通知链；真实菜单打开不变。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D334 | prop | `onOpen` #1 | 33→— | 删除此声明 |

### F092 `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-filters-selection.tsx`

删除依据：ignoreGroupedFilters 未传入，删除 group/subgroup 传递及恒空排除分支。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D335 | prop | `ignoreGroupedFilters` #1 | 31→— | 删除此声明 |

### F093 `web/apps/web/core/components/issues/issue-layouts/roots/archive-detail.test.tsx`

实际读取：AppHeader 透传 header；DescriptionInput 用 disabled 显示只读，真实归档路由与控件断言。

验证：components/issues/issue-layouts/roots/archive-detail.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D342 | prop | `header` #1 | 34→34 | 保留 |
| D343 | prop | `disabled` #1 | 39→39 | 保留 |

### F094 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，assignee 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D346 | prop | `issue` #1 | 17→17 | 保留 |
| D347 | prop | `onClose` #1 | 18→18 | 保留 |
| D348 | prop | `onChange` #1 | 19→19 | 保留 |
| D349 | prop | `disabled` #1 | 20→20 | 保留 |

### F095 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/created-on-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，created-on 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D350 | prop | `issue` #1 | 18→18 | 保留 |

### F096 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，due-date 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D351 | prop | `issue` #1 | 21→21 | 保留 |
| D352 | prop | `onClose` #1 | 22→22 | 保留 |
| D353 | prop | `onChange` #1 | 23→23 | 保留 |
| D354 | prop | `disabled` #1 | 24→24 | 保留 |

### F097 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，label 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D355 | prop | `issue` #1 | 18→18 | 保留 |
| D356 | prop | `onClose` #1 | 19→19 | 保留 |
| D357 | prop | `onChange` #1 | 20→20 | 保留 |
| D358 | prop | `disabled` #1 | 21→21 | 保留 |

### F098 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/link-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，link 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D359 | prop | `issue` #1 | 16→16 | 保留 |

### F099 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，priority 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D360 | prop | `issue` #1 | 17→17 | 保留 |
| D361 | prop | `onClose` #1 | 18→18 | 保留 |
| D362 | prop | `onChange` #1 | 19→19 | 保留 |
| D363 | prop | `disabled` #1 | 20→20 | 保留 |

### F100 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，start-date 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D364 | prop | `issue` #1 | 19→19 | 保留 |
| D365 | prop | `onClose` #1 | 20→20 | 保留 |
| D366 | prop | `onChange` #1 | 21→21 | 保留 |
| D367 | prop | `disabled` #1 | 22→22 | 保留 |

### F101 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，state 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D368 | prop | `issue` #1 | 17→17 | 保留 |
| D369 | prop | `onClose` #1 | 18→18 | 保留 |
| D370 | prop | `onChange` #1 | 19→19 | 保留 |
| D371 | prop | `disabled` #1 | 20→20 | 保留 |

### F102 `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/updated-on-column.tsx`

实际读取：SPREADSHEET_COLUMNS 动态选列；IssueColumn 显式传 issue/onClose/onChange/disabled，updated-on 列读取字段并交真实控件；回焦返回 td。

验证：spreadsheet/view.test.tsx；dropdowns/basic-dropdowns.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D372 | prop | `issue` #1 | 18→18 | 保留 |

### F103 `web/apps/web/core/components/issues/issue-modal/draft.test.tsx`

实际读取：ProjectSelect 读 disabled/value/onChange；QuickActions map 读取 MENU_ITEMS.key/action；Modal 读 isOpen。

验证：components/issues/issue-modal/draft.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D383 | prop | `disabled` #1 | 59→59 | 保留 |
| D384 | prop | `value` #1 | 60→60 | 保留 |
| D385 | prop | `onChange` #1 | 61→61 | 保留 |
| D386 | prop | `MENU_ITEMS` #1 | 89→89 | 保留 |
| D387 | prop | `key` #1 | 89→89 | 保留 |
| D388 | prop | `action` #1 | 89→89 | 保留 |
| D389 | prop | `isOpen` #1 | 117→117 | 保留 |

### F104 `web/apps/web/core/components/issues/links/data.test.tsx`

实际读取：Input/Button 替身剥除 size/variant 防止传错 DOM，loading 驱动 disabled；Modal/Menu 读 isOpen/onClick；多余 ref 声明已删，原生 rest props 仍透传。

验证：components/issues/links/data.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

删除依据：测试多余 ref 声明移除；Input 原生 rest props 保留并透传。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D390 | prop | `size` #1 | 39→39 | 保留 |
| D391 | prop | `ref` #1 | 40→— | 删除此声明 |
| D392 | prop | `loading` #1 | 49→48 | 保留 |
| D393 | prop | `variant` #1 | 49→48 | 保留 |
| D394 | prop | `size` #2 | 49→48 | 保留 |
| D395 | prop | `isOpen` #1 | 55→54 | 保留 |
| D396 | prop | `onClick` #1 | 58→57 | 保留 |

### F105 `web/apps/web/core/components/issues/parent-selection.test.tsx`

实际读取：Modal 替身读 isOpen，真实父项选择拥有者驱动打开关闭。

验证：components/issues/parent-selection.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D397 | prop | `isOpen` #1 | 32→32 | 保留 |

### F106 `web/apps/web/core/components/issues/relations/data.test.tsx`

实际读取：Tooltip/Menu/Collapsible 替身透传 children，MenuItem 调 onClick；RelationList 调 handleIssueCrudState，真实拥有者验证确认事件不重复写。

验证：components/issues/relations/data.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D400 | prop | `children` #1 | 49→49 | 保留 |
| D401 | prop | `children` #2 | 52→52 | 保留 |
| D402 | prop | `children` #3 | 53→53 | 保留 |
| D403 | prop | `children` #4 | 58→58 | 保留 |
| D404 | prop | `onClick` #1 | 59→59 | 保留 |
| D405 | prop | `children` #5 | 64→64 | 保留 |
| D406 | prop | `handleIssueCrudState` #1 | 74→74 | 保留 |

### F107 `web/apps/web/core/components/issues/workspace-draft/page.test.tsx`

实际读取：DraftIssueBlock 替身显示 workspaceSlug/issueId，真实草稿路由验证作用域。

验证：components/issues/workspace-draft/page.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D407 | prop | `issueId` #1 | 33→33 | 保留 |
| D408 | prop | `workspaceSlug` #1 | 33→33 | 保留 |

### F108 `web/apps/web/core/components/issues/workspace-draft/root.test.tsx`

实际读取：DraftIssueBlock 替身读 issueId；真实无项目草稿列表验证无需偏好读取。

验证：components/issues/workspace-draft/root.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D409 | prop | `issueId` #1 | 29→29 | 保留 |

### F109 `web/apps/web/core/components/power-k/ui/modal/search-results.tsx`

实际读取：issues.map 解构 issue/project，显示编号名称并构造原生路由；root 传入搜索结果。

验证：components/power-k/ui/modal/search-menu.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D410 | prop | `issue` #1 | 11→11 | 保留 |
| D411 | prop | `project` #1 | 11→11 | 保留 |

### F110 `web/apps/web/core/components/project/filters.tsx`

删除依据：四个可选展示参数无调用写入，删除声明和恒定样式分支。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D412 | prop | `filterMenuButton` #1 | 25→— | 删除此声明 |
| D413 | prop | `classname` #1 | 26→— | 删除此声明 |
| D414 | prop | `filterClassname` #1 | 27→— | 删除此声明 |
| D415 | prop | `isMobile` #1 | 28→— | 删除此声明 |

### F111 `web/apps/web/core/components/rich-filters/filter-item/property.tsx`

删除依据：扩展 className 无传入，删专属类型与 merge；属性按钮样式不变。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D416 | prop | `className` #1 | 50→— | 删除此声明 |

### F112 `web/apps/web/core/components/ui/loader/layouts/kanban-layout-loader.tsx`

删除依据：cardsInEachColumn 无调用写入，固定原五列数据并使用稳定列名 key。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D417 | prop | `cardsInEachColumn` #1 | 57→— | 删除此声明 |

### F113 `web/apps/web/core/components/work-item-filters/filters-hoc/shared.ts`

实际读取：project-level/workspace-level 的 hasSaveOption/handleSaveView/handleUpdateView 仍读保存/更新配置；M7 view 合同。

验证：services/workspace.service.test.ts；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D418 | prop | `saveViewOptions` #1 | 27→27 | 保留 |
| D419 | prop | `enableUpdateView` #1 | 31→31 | 保留 |
| D420 | prop | `updateViewOptions` #1 | 32→32 | 保留 |

### F114 `web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx`

实际读取：filterOptions.map 解构 label/value，checkbox 读选中值；M7 通知展示合同。

验证：web 类型；本期不声称通知接口已实现；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D421 | prop | `label` #1 | 39→39 | 保留 |
| D422 | prop | `value` #1 | 39→39 | 保留 |

### F115 `web/apps/web/core/layouts/default-layout/index.tsx`

删除依据：gradient 无调用写入，移除条件样式参数，保留默认背景。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D423 | prop | `gradient` #1 | 12→— | 删除此声明 |

### F116 `web/packages/constants/src/issue/filter.ts`

实际读取：ACTIVITY_FILTER_OPTIONS 通过 issue-activity/activity-filter.tsx 的 map 读取 key/labelTranslationKey/isSelected/onClick。

验证：issue-detail/issue-activity/data.test.tsx；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D424 | prop | `key` #1 | 252→241 | 保留 |
| D425 | prop | `labelTranslationKey` #1 | 253→242 | 保留 |
| D426 | prop | `isSelected` #1 | 254→243 | 保留 |
| D427 | prop | `onClick` #1 | 255→244 | 保留 |

### F117 `web/packages/constants/src/issue/layout.ts`

实际读取：ISSUE_LAYOUTS 经布局菜单 map 读取 key/i18n_title/i18n_label；图标函数被 React 调用。

验证：dropdowns/layout.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D428 | prop | `key` #1 | 42→42 | 保留 |
| D429 | prop | `i18n_title` #1 | 43→43 | 保留 |
| D430 | prop | `i18n_label` #1 | 44→44 | 保留 |

### F118 `web/packages/editor/src/components/editors/editor-content.tsx`

删除依据：className/children 无调用写入，删除两可选参数及独占渲染。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D431 | prop | `className` #1 | 12→— | 删除此声明 |
| D432 | prop | `children` #1 | 13→— | 删除此声明 |

### F119 `web/packages/editor/src/components/menus/floating-menu/root.tsx`

删除依据：onClick 从未传入，删除空通知；实际上下文 toggle 留存。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D433 | prop | `onClick` #1 | 19→— | 删除此声明 |

### F120 `web/packages/editor/src/extensions/code/code-block-node-view.tsx`

实际读取：ReactNodeViewRenderer 向 CodeBlockNodeView 注入 node；组件读 attrs.language 并更新。

验证：editor-interaction.test.ts；editor 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D434 | prop | `node` #1 | 26→26 | 保留 |

### F121 `web/packages/shared-state/src/store/rich-filters/config.ts`

实际读取：IFilterConfig 被 filter-item/root 与 add-filters/dropdown 读取操作符/显示操作符，配置管理器调用 mutate。

验证：work-item-filters/filter-config.test.tsx；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D439 | prop | `allEnabledSupportedOperators` #1 | 23→23 | 保留 |
| D440 | prop | `firstOperator` #1 | 24→24 | 保留 |
| D441 | prop | `getOperatorConfig` #1 | 26→26 | 保留 |
| D442 | prop | `getDisplayOperatorByValue` #1 | 29→29 | 保留 |
| D443 | prop | `mutate` #1 | 31→31 | 保留 |

### F122 `web/packages/types/src/rich-filters/field-types/shared.ts`

实际读取：filter-value-input/dropdown/search-options 渲染选项 icon/iconClassName/disabled/description。

验证：rich-filters 的选择器测试；web 类型；执行收据见 review T4 的全类型/全前端/knip。

| 行 ID | kind | symbol（同名次序） | 旧→新 | 结论 |
|---|---|---|---|---|
| D446 | prop | `icon` #1 | 33→33 | 保留 |
| D447 | prop | `iconClassName` #1 | 34→34 | 保留 |
| D448 | prop | `disabled` #1 | 35→35 | 保留 |
| D449 | prop | `description` #1 | 36→36 | 保留 |

## 4. 历史 52 行逐名追溯

原报告为 P10a 的 task-7-r1-current-final.tsv（草稿16行）与 P11 的 t15-dead-before.raw（另36行），原始 kind 保留；源路径和 SHA 见 historical-52.json。删除提交的实际 diff 已逐行核对。

| ID | 原文件及行 | kind | owner.symbol | 删除提交 |
|---|---|---|---|---|
| H001 | `web/packages/types/src/workspace-draft-issues/base.ts:12` | member | `TWorkspaceDraftIssue.sort_order` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H002 | `web/packages/types/src/workspace-draft-issues/base.ts:20` | member | `TWorkspaceDraftIssue.parent_id` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H003 | `web/packages/types/src/workspace-draft-issues/base.ts:26` | member | `TWorkspaceDraftIssue.completed_at` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H004 | `web/packages/types/src/workspace-draft-issues/base.ts:29` | member | `TWorkspaceDraftIssue.updated_at` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H005 | `web/packages/types/src/workspace-draft-issues/base.ts:31` | member | `TWorkspaceDraftIssue.updated_by` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H006 | `web/packages/types/src/workspace-draft-issues/base.ts:33` | member | `TWorkspaceDraftIssue.is_draft` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H007 | `web/packages/types/src/workspace-draft-issues/base.ts:38` | member | `TWorkspaceDraftPaginationInfo.prev_cursor` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H008 | `web/packages/types/src/workspace-draft-issues/base.ts:40` | member | `TWorkspaceDraftPaginationInfo.prev_page_results` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H009 | `web/packages/types/src/workspace-draft-issues/base.ts:41` | member | `TWorkspaceDraftPaginationInfo.total_pages` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H010 | `web/packages/types/src/workspace-draft-issues/base.ts:42` | member | `TWorkspaceDraftPaginationInfo.count` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H011 | `web/packages/types/src/workspace-draft-issues/base.ts:44` | member | `TWorkspaceDraftPaginationInfo.total_results` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H012 | `web/packages/types/src/workspace-draft-issues/base.ts:46` | member | `TWorkspaceDraftPaginationInfo.extra_stats` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H013 | `web/packages/types/src/workspace-draft-issues/base.ts:47` | member | `TWorkspaceDraftPaginationInfo.grouped_by` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H014 | `web/packages/types/src/workspace-draft-issues/base.ts:48` | member | `TWorkspaceDraftPaginationInfo.sub_grouped_by` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H015 | `web/packages/types/src/workspace-draft-issues/base.ts:52` | member | `TWorkspaceDraftQueryParams.per_page` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H016 | `web/packages/types/src/workspace-draft-issues/base.ts:53` | member | `TWorkspaceDraftQueryParams.cursor` | `87d1bcd72d959a036db9dc3ce5d3a6fce70a8b20` |
| H017 | `web/apps/web/core/services/workspace.service.ts:15` | member | `WorkspaceService.searchWorkspace.params.project_id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H018 | `web/apps/web/core/services/workspace.service.ts:16` | member | `WorkspaceService.searchWorkspace.params.search` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H019 | `web/apps/web/core/services/workspace.service.ts:17` | member | `WorkspaceService.searchWorkspace.params.workspace_search` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H020 | `web/packages/types/src/project/projects.ts:16` | member | `TProjectIssuesSearchParams.search` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H021 | `web/packages/types/src/project/projects.ts:17` | member | `TProjectIssuesSearchParams.parent` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H022 | `web/packages/types/src/project/projects.ts:18` | member | `TProjectIssuesSearchParams.issue_relation` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H023 | `web/packages/types/src/project/projects.ts:19` | member | `TProjectIssuesSearchParams.cycle` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H024 | `web/packages/types/src/project/projects.ts:20` | member | `TProjectIssuesSearchParams.module` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H025 | `web/packages/types/src/project/projects.ts:21` | member | `TProjectIssuesSearchParams.sub_issue` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H026 | `web/packages/types/src/project/projects.ts:22` | member | `TProjectIssuesSearchParams.issue_id` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H027 | `web/packages/types/src/project/projects.ts:23` | member | `TProjectIssuesSearchParams.workspace_search` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H028 | `web/packages/types/src/project/projects.ts:24` | member | `TProjectIssuesSearchParams.target_date` | `4c0ed5ea369d42e46e942e5a9cd9323f02829175` |
| H029 | `web/packages/types/src/workspace.ts:44` | member | `IWorkspaceSearchResults.results.workspace` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H030 | `web/packages/types/src/workspace.ts:45` | member | `IWorkspaceSearchResults.results.project` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H031 | `web/packages/types/src/workspace.ts:46` | member | `IWorkspaceSearchResults.results.issue` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H032 | `web/packages/types/src/workspace.ts:47` | member | `IWorkspaceSearchResults.results.cycle` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H033 | `web/packages/types/src/workspace.ts:48` | member | `IWorkspaceSearchResults.results.module` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H034 | `web/packages/types/src/workspace.ts:49` | member | `IWorkspaceSearchResults.results.issue_view` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H035 | `web/packages/types/src/workspace.ts:14` | prop | `IWorkspaceDefaultSearchResult.id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H036 | `web/packages/types/src/workspace.ts:15` | prop | `IWorkspaceDefaultSearchResult.name` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H037 | `web/packages/types/src/workspace.ts:16` | prop | `IWorkspaceDefaultSearchResult.project_id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H038 | `web/packages/types/src/workspace.ts:17` | prop | `IWorkspaceDefaultSearchResult.project__identifier` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H039 | `web/packages/types/src/workspace.ts:18` | prop | `IWorkspaceDefaultSearchResult.workspace__slug` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H040 | `web/packages/types/src/workspace.ts:21` | prop | `IWorkspaceSearchResult.id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H041 | `web/packages/types/src/workspace.ts:22` | prop | `IWorkspaceSearchResult.name` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H042 | `web/packages/types/src/workspace.ts:23` | prop | `IWorkspaceSearchResult.slug` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H043 | `web/packages/types/src/workspace.ts:27` | prop | `IWorkspaceIssueSearchResult.id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H044 | `web/packages/types/src/workspace.ts:28` | prop | `IWorkspaceIssueSearchResult.name` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H045 | `web/packages/types/src/workspace.ts:29` | prop | `IWorkspaceIssueSearchResult.project__identifier` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H046 | `web/packages/types/src/workspace.ts:30` | prop | `IWorkspaceIssueSearchResult.project_id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H047 | `web/packages/types/src/workspace.ts:31` | prop | `IWorkspaceIssueSearchResult.sequence_id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H048 | `web/packages/types/src/workspace.ts:32` | prop | `IWorkspaceIssueSearchResult.workspace__slug` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H049 | `web/packages/types/src/workspace.ts:36` | prop | `IWorkspaceProjectSearchResult.id` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H050 | `web/packages/types/src/workspace.ts:37` | prop | `IWorkspaceProjectSearchResult.identifier` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H051 | `web/packages/types/src/workspace.ts:38` | prop | `IWorkspaceProjectSearchResult.name` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |
| H052 | `web/packages/types/src/workspace.ts:39` | prop | `IWorkspaceProjectSearchResult.workspace__slug` | `5f0961c1b3b88d92aa20df760d31e1b7969505eb` |

草稿旧类型文件已整体删除，由原生 Draft/Payload 接管；project/projects.ts 和 workspace.ts 文件仍在，其角色/引导枚举仍有读者。WorkspaceService 只删 searchWorkspace：`createView/updateView/deleteView/getAllViews` 仍分别由 `store/global-view.store.ts` 调用，`services/workspace.service.test.ts` 验证四条 M7 路径及拒绝传递；不能把“旧52行均删除”写成“四文件均删除”。

## 5. 全仓 export 21 项

| 原声明 | 处置与真实读者 |
|---|---|
| `web/packages/api-client/test/client.typecheck.ts:7 describeInstance` | 保留编译专用导出；client.typecheck.ts 的 @ts-expect-error 由 tsc 实际编译，不运行、不从生产 barrel 暴露 |
| `web/packages/api-client/test/client.typecheck.ts:22 rejectsUndocumentedCalls` | 保留编译专用导出；client.typecheck.ts 的 @ts-expect-error 由 tsc 实际编译，不运行、不从生产 barrel 暴露 |
| `web/packages/constants/src/fetch-keys.ts:59 CYCLE_ISSUES_WITH_PARAMS` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/constants/src/fetch-keys.ts:68 ISSUE_DETAILS` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/propel/src/emoji-reaction/emoji-reaction.tsx:109 EmojiReactionButton` | 改为私有；保留本文件真正的使用点（反应组/操作符类型/视图枚举） |
| `web/packages/types/src/issues/issue_attachment.ts:9 TIssueAttachment` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/types/src/issues/issue_attachment.ts:23 TIssueAttachmentUploadResponse` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/types/src/issues/issue_attachment.ts:27 TIssueAttachmentMap` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/types/src/issues/issue_attachment.ts:31 TIssueAttachmentIdMap` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/types/src/rich-filters/derived/index.ts:35 TSupportedDateFilterOperators` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/types/src/rich-filters/operators/index.ts:46 OPERATORS` | 改为私有；保留本文件真正的使用点（反应组/操作符类型/视图枚举） |
| `web/packages/types/src/view-props.ts:14 TIssueLayouts` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/types/src/workspace-views.ts:41 STATIC_VIEW_TYPES` | 改为私有；保留本文件真正的使用点（反应组/操作符类型/视图枚举） |
| `web/packages/ui/src/dropdowns/combo-box.tsx:30 ComboDropDown` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/attachment.ts:7 getFileExtension` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/attachment.ts:9 getFileName` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/attachment.ts:17 convertBytesToSize` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/datetime.ts:238 getCurrentDateTimeInISO` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/datetime.ts:374 isValidDate` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/work-item/base.ts:18 handleIssueQueryParamsByLayout` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |
| `web/packages/utils/src/work-item/base.ts:66 formatTextList` | 删除无外部读者声明及独占实现/barrel；Attachment、derived、ComboDropDown、attachment 工具孤岛整文件删除 |

## 6. 证据与局限

- AST：`t4-initial-ast.json`、`t4-final-ast.json`、`dead-audit/*-source-manifest.json`；终态报告 SHA `70710c06dcc6fa61088e6b9e6ae5182bf61420071bc5734a823c126a5d174df3`。
- 类型：`t4-static-final-types-r2.json`；knip：`t4-knip-r2.json`；全前端强制执行：`t4-web-tests.json`（18 tasks，0 cached）。这些原始输出与源码输入 hash 成套保留。
- M11 检验逐行账本覆盖，不证明业务行为；M12 对真实 TableView.contentDOM 进行合法类型变异，由真实编辑器表格用例检出，恢复文件 SHA 后再绿。最终执行结果见阶段 review。
- React 错误边界、未启用图片/未接通 M7 的某些协议只有类型和存活调用依据，不把未执行的业务场景记为通过。T11 将新 AST 与此逐行结论对账，任何新增行需要新读者记录。
