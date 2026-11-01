# P16 准备盘点（仅静态输入）

日期：2026-11-01；基线 `313728cbc61003d7ed072c1e1e191e1cc2a3d75a`。本页不是实施完成证明。原始证据在主目录 `.superpowers/sdd/P16-preparation-20261101/`；web/ 源码扫描复用 P15 最终相同内容；集成修复只改变 I13 E2E 与记录，最终等价核验见 integration-baseline-equivalence.json。

## 1. 共享选择器当前消费者

下列是生产 JSX 直接调用；排除 test/spec/typecheck 和 fake-controls。`.Option`、类型引用与包装组件另核，不能仅靠此表推断全部影响。

| 文件 | 组件 | 调用行（准备定位） |
|---|---|---|
| `web/apps/web/core/components/api-token/modal/form.tsx` | CustomSelect | 170 |
| `web/apps/web/core/components/automation/auto-archive-automation.tsx` | CustomSelect | 109 |
| `web/apps/web/core/components/core/filters/date-filter-select.tsx` | CustomSelect | 49 |
| `web/apps/web/core/components/core/theme/theme-switch.tsx` | CustomSelect | 25 |
| `web/apps/web/core/components/dropdowns/member/base.tsx` | CustomSearchSelect | 159 |
| `web/apps/web/core/components/dropdowns/priority.tsx` | CustomSearchSelect | 344 |
| `web/apps/web/core/components/dropdowns/project/base.tsx` | CustomSearchSelect | 159, 161 |
| `web/apps/web/core/components/dropdowns/state/base.tsx` | CustomSearchSelect | 121 |
| `web/apps/web/core/components/global/timezone-select.tsx` | CustomSearchSelect | 32 |
| `web/apps/web/core/components/issues/peek-overview/header.tsx` | CustomSelect | 152 |
| `web/apps/web/core/components/modules/module-status-dropdown.tsx` | CustomSelect | 30 |
| `web/apps/web/core/components/modules/progress-sidebar/root.tsx` | CustomSelect | 193 |
| `web/apps/web/core/components/modules/select/status.tsx` | CustomSelect | 36 |
| `web/apps/web/core/components/navigation/project-header.tsx` | CustomSearchSelect | 102 |
| `web/apps/web/core/components/profile/start-of-week-preference.tsx` | CustomSelect | 53 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx` | CustomSelect | 222 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx` | CustomSearchSelect | 170 |
| `web/apps/web/core/components/project/form.tsx` | CustomSelect | 239 |
| `web/apps/web/core/components/project/member-select.tsx` | CustomSearchSelect | 72 |
| `web/apps/web/core/components/project/settings/member-columns.tsx` | CustomSelect | 103 |
| `web/apps/web/core/components/projects/create/attributes.tsx` | CustomSelect | 39 |
| `web/apps/web/core/components/rich-filters/add-filters/dropdown.tsx` | CustomSearchSelect | 80 |
| `web/apps/web/core/components/rich-filters/filter-item/root.tsx` | CustomSearchSelect | 106 |
| `web/apps/web/core/components/rich-filters/filter-value-input/select/multi.tsx` | CustomSearchSelect | 41 |
| `web/apps/web/core/components/rich-filters/filter-value-input/select/single.tsx` | CustomSearchSelect | 45 |
| `web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx` | CustomSelect | 74 |
| `web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx` | CustomSelect | 158 |
| `web/apps/web/core/components/workspace/create-workspace-form.tsx` | CustomSelect | 131 |
| `web/apps/web/core/components/workspace/invite-modal/fields.tsx` | CustomSelect | 98 |
| `web/apps/web/core/components/workspace/settings/invitations-list-item.tsx` | CustomSelect | 102 |
| `web/apps/web/core/components/workspace/settings/member-columns.tsx` | CustomSelect | 129 |
| `web/apps/web/core/components/workspace/settings/workspace-details.tsx` | CustomSelect | 188 |
| `web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx` | CustomSearchSelect | 63 |

共32个文件、34个调用点；20个CustomSelect点，14个CustomSearchSelect点。共同泛型源为 `web/packages/types/src/common.ts` 的 ICustomSearchSelectOption；全部类型引用在 select-option-type-readers.txt。

## 2. 当前 lint 和 no-shadow

此次是只读 oxlint JSON 扫描，未改规则/代码。各包 cap 与实际值一致。

| 包 | warning/cap | no-shadow |
|---|---:|---:|
| web | 108 | 14 |
| @nerve/api-client | 0 | 0 |
| @nerve/constants | 0 | 0 |
| @nerve/editor | 48 | 16 |
| @nerve/hooks | 3 | 0 |
| @nerve/i18n | 0 | 0 |
| @nerve/propel | 10 | 4 |
| @nerve/services | 0 | 0 |
| @nerve/shared-state | 0 | 0 |
| @nerve/types | 0 | 0 |
| @nerve/ui | 10 | 3 |
| @nerve/utils | 6 | 1 |
| @nerve/e2e | 0 | 0 |
| nerve | 0 | 0 |

总185 warnings、0 errors，no-shadow38条/27文件；源码清单：

- `web/apps/web/core/components/cycles/active-cycle/cycle-stats.tsx`
- `web/apps/web/core/components/cycles/form.tsx`
- `web/apps/web/core/components/cycles/modal.tsx`
- `web/apps/web/core/components/dropdowns/layout.tsx`
- `web/apps/web/core/components/inbox/inbox-filter/applied-filters/date.tsx`
- `web/apps/web/core/components/inbox/inbox-filter/applied-filters/member.tsx`
- `web/apps/web/core/components/inbox/modals/snooze-issue-modal.tsx`
- `web/apps/web/core/components/modules/form.tsx`
- `web/apps/web/core/components/modules/progress-sidebar/root.tsx`
- `web/apps/web/core/components/ui/loader/layouts/calendar-layout-loader.tsx`
- `web/apps/web/core/components/workspace/sidebar/favorites/favorite-folder.tsx`
- `web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/root.tsx`
- `web/packages/editor/src/components/editors/link-view-container.tsx`
- `web/packages/editor/src/components/menus/bubble-menu/root.tsx`
- `web/packages/editor/src/extensions/custom-image/components/upload-status.tsx`
- `web/packages/editor/src/extensions/image/extension.tsx`
- `web/packages/editor/src/extensions/table/plugins/drag-handles/column/drag-handle.tsx`
- `web/packages/editor/src/extensions/table/plugins/drag-handles/row/drag-handle.tsx`
- `web/packages/editor/src/extensions/utility.ts`
- `web/packages/editor/src/helpers/editor-commands.ts`
- `web/packages/editor/src/helpers/editor-ref.ts`
- `web/packages/editor/src/plugins/drag-handle.ts`
- `web/packages/propel/src/calendar/root.tsx`
- `web/packages/propel/src/charts/area-chart/root.tsx`
- `web/packages/ui/src/dropdown/common/options.tsx`
- `web/packages/ui/src/dropdown/single-select.tsx`
- `web/packages/utils/src/editor/markdown-parser/root.ts`

上述38条未包含被屏蔽的诊断。八个文件各一处既有屏蔽，另有8条隐藏诊断；合计46条/33个诊断文件，连同两处无效屏蔽共35个待核文件：

| 路径（均在 web/apps/web/core/components/） | 隐藏诊断 | 定位/结论 |
|---|---:|---|
| breadcrumbs/project.tsx | 1 | 第39行 projectId |
| inbox/content/issue-root.tsx | 2 | 第104行 workspaceSlug/projectId |
| issues/issue-detail/main-content.tsx | 0 | 文件级屏蔽已无效，仍须删除 |
| issues/issue-modal/form.tsx | 1 | 第261行 data |
| issues/peek-overview/issue-detail.tsx | 0 | 文件级屏蔽已无效，仍须删除 |
| navigation/use-navigation-items.ts | 2 | 第36行 workspaceSlug/projectId |
| views/form.tsx | 1 | 第120行 logoValue |
| workspace/sidebar/projects-list-item.tsx | 1 | 第174行 element |

`shadow-suppression-scan.json` 保留逐文件原hash、原样/仅用空格替换该注释的副本hash、同一锁定oxlint与同规则配置的成对raw。临时配置仅去除ignorePatterns，无no-shadow路径override；原样副本均0、去屏蔽后新增8。原文件/规则未改，这不是P16产品测试。全规则disable检索未发现其它屏蔽；T3须删除八处注释并复核专项/全规则绕过为0，不能将隐藏告警留到升error以后。

domains 原M4匹配60条。M4全期修改以M3 closeout `53a6a05b` 为起点，当前显示6条/3文件（两处sidebar/progress-stats及cycle-stats），需按“谁改谁清”复核，不能只检查本期。M4领域、显式no-shadow文件、选择器消费者和M4全期改动的并集当前103条/55文件；详见planned-lint-scope.json。纳入八个屏蔽文件后，待核范围为63文件（含两处无效屏蔽），已知诊断103显示+8隐藏；不能把原103当完整上限或上抬cap。每次新增修改需刷新集合。

## 3. Dead候选基线

P15 `t16-ci-final-inventory.log` SHA-256：`a914901a68b51b7305fdd39d458b009d9b9374c74f3382ce7598f604f9a7ea68`。本基线的web/扫描输入与P15最终de9a36fc相同，I13集成修复仅改E2E测试与文稿；这是复用原AST产物，不是重新执行P16 AST。全仓member875/prop369/export21；domains M4为member295/prop158，合453候选/123文件。T4须当前typegen后重跑原脚本，逐行归因，不能据计数批删。

| 历史文件组 | 设计历史数 | 当前member/prop候选 | 后续核对 |
|---|---:|---:|---|
| `web/packages/types/src/workspace-draft-issues/base.ts` | 16 | 0 | 逐名核历史删除提交或当前读者；不以0候选宣称全文件已删 |
| `web/packages/types/src/workspace.ts` | 24 | 0 | 逐名核历史删除提交或当前读者；不以0候选宣称全文件已删 |
| `web/packages/types/src/project/projects.ts` | 9 | 0 | 逐名核历史删除提交或当前读者；不以0候选宣称全文件已删 |
| `web/apps/web/core/services/workspace.service.ts` | 3 | 0 | 逐名核历史删除提交或当前读者；不以0候选宣称全文件已删 |

T4 ledger逐行以kind+path+symbol追踪，范围包括测试形状与原脚本误归项；M7 WorkspaceService四个view方法及公开编辑器ref尤其不能误删。

## 4. 最后读者和守卫

`getTabIndex` 的七个生产调用文件：

- `web/apps/web/core/components/views/form.tsx`
- `web/apps/web/core/components/cycles/form.tsx`
- `web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx`
- `web/apps/web/core/components/inbox/modals/create-modal/issue-title.tsx`
- `web/apps/web/core/components/inbox/modals/create-modal/issue-description.tsx`
- `web/apps/web/core/components/inbox/modals/create-modal/create-root.tsx`
- `web/apps/web/core/components/modules/form.tsx`

生产M7隐式projectMemberIds getter读者：

- `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/mobile-header.tsx`
- `web/apps/web/core/components/views/view-list-header.tsx`
- `web/apps/web/core/components/inbox/inbox-filter/filters/filter-selection.tsx`（两处）

M4同名局部变量使用显式getProjectMemberIds(projectId,false)，不是getter；保留M7接口和M3测试直到最后读者迁移。

当前phase=M4/P15。old-issue-urls/negation-filters例外均0；copy-hook例外仅M6五、M7四、M8一。准备期不改phase/守卫；T1推进M4/P16、T6核终态。不匹配八类旧工作项地址的领域服务不自动归零。

## 5. 测试与页面准备入口

现有可复用证据/测试：

- `web/apps/web/core/components/dropdowns/basic-dropdowns.test.tsx`：真实控件、IME/229、portal隔离、单多选；T2补空态/0/对象及类型负例。
- `web/packages/ui/src/modals/modal-core.test.tsx`、`headlessui-transition.test.tsx`：嵌套Escape与快速关闭。
- `web/apps/web/core/components/rich-filters/` 的既有选择器测试：状态/文案/持有目录。
- `e2e/stories/smoke/s2-web-app.spec.ts`：四身份、项目列表/详情、精确请求；T9补最终入口表和缺口。
- `e2e/playwright.m4-list.config.ts`、`playwright.m4-filters.config.ts`、`playwright.m4-detail.config.ts`、`playwright.m4-collaboration.config.ts`：三个引擎的真实项目；T10去重合成矩阵。

尚未新增任何P16测试或运行产品矩阵。本次只准备实现入口，逐项验证留授权实施。

## T2 实施消费者核验（2026-11-01）


依据实际 TypeScript program/checker 的 instantiated JSX props，32文件/34点 value/onChange 无 any；SearchSelect 显示的联合是判别联合投影，不是公开 boolean/value 脱钩；正负 tsc 验证相关性。独立 JSX children 的泛型无法证明 Select.Option 与父组件 T 相等，仍靠真实消费者和运行选项核对。

| 路径 | 调用 | value 类型 | 适配/兼容结论 |
|---|---|---|---|
| `web/apps/web/core/components/navigation/project-header.tsx` | CustomSearchSelect:102 | `string  /  string[]  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/core/theme/theme-switch.tsx` | CustomSelect:25 | `I_THEME_OPTION  /  null` | I_THEME_OPTION 对象原样传递，真实选择验证引用身份。 |
| `web/apps/web/core/components/global/timezone-select.tsx` | CustomSearchSelect:32 | `string  /  string[]  /  null` | 将 undefined 当前空值在拥有者处归一为 null；onChange 仍只发时区字符串。 |
| `web/apps/web/core/components/profile/start-of-week-preference.tsx` | CustomSelect:53 | `0  /  1  /  2  /  3  /  4  /  5  /  6  /  null` | 将尚未加载的当前空值归一为 null，数字 0–6 不变。 |
| `web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx` | CustomSelect:74 | `"en"  /  "zh-CN"  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/api-token/modal/form.tsx` | CustomSelect:170 | `"1_week"  /  "1_month"  /  "3_months"  /  "1_year"  /  "custom"  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/dropdowns/member/base.tsx` | CustomSearchSelect:159 | `string  /  string[]  /  null` | 保留 selection 判别联合，成员 option 默认 string，暂停成员 disabled 保持。 |
| `web/apps/web/core/components/projects/create/attributes.tsx` | CustomSelect:39 | `0  /  2  /  null  /  undefined` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/dropdowns/priority.tsx` | CustomSearchSelect:344 | `TIssuePriorities  /  TIssuePriorities[]  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/dropdowns/state/base.tsx` | CustomSearchSelect:121 | `string  /  string[]  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/dropdowns/project/base.tsx` | CustomSearchSelect:159 | `string  /  string[]  /  null` | 保留两条按 multiple 判别的实际 JSX，string/string[] 回调独立。 |
| `web/apps/web/core/components/dropdowns/project/base.tsx` | CustomSearchSelect:161 | `string  /  string[]  /  null` | 保留两条按 multiple 判别的实际 JSX，string/string[] 回调独立。 |
| `web/apps/web/core/components/core/filters/date-filter-select.tsx` | CustomSelect:49 | `string  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/issues/peek-overview/header.tsx` | CustomSelect:152 | `"side-peek"  /  "modal"  /  "full-screen"  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/modules/select/status.tsx` | CustomSelect:36 | `"backlog"  /  "completed"  /  "cancelled"  /  "planned"  /  "in-progress"  /  "paused"  /  null  /  undefined` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/modules/module-status-dropdown.tsx` | CustomSelect:30 | `"backlog"  /  "completed"  /  "cancelled"  /  "planned"  /  "in-progress"  /  "paused"  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/modules/progress-sidebar/root.tsx` | CustomSelect:193 | `"backlog"  /  "completed"  /  "cancelled"  /  "planned"  /  "in-progress"  /  "paused"  /  null` | 移除 status 回调 any 与 value 遮蔽，字段当前空值归一 null；wire key 仍为 status。 |
| `web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx` | CustomSelect:158 | `string  /  null  /  undefined` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/rich-filters/add-filters/dropdown.tsx` | CustomSearchSelect:83 | `P  /  "all_filters_applied"  /  (P  /  "all_filters_applied")[]  /  null` | 明确 P/disabled 空列表哨兵，当前无选择用 null，哨兵不发业务回调。 |
| `web/apps/web/core/components/rich-filters/filter-value-input/select/multi.tsx` | CustomSearchSelect:41 | `string  /  string[]  /  null` | 按已有 string condition 显式 toFilterArray<string>，数组选择保持。 |
| `web/apps/web/core/components/rich-filters/filter-value-input/select/single.tsx` | CustomSearchSelect:45 | `string  /  string[]  /  null` | 按已有 string condition 显式 toFilterArray<string>，相同值取消的 null 仍由拥有者处理。 |
| `web/apps/web/core/components/rich-filters/filter-item/root.tsx` | CustomSearchSelect:106 | `TSupportedOperators  /  TSupportedOperators[]  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/workspace/settings/workspace-details.tsx` | CustomSelect:188 | `"Just myself"  /  "2-10"  /  "11-50"  /  "51-200"  /  "201-500"  /  "500+"  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/workspace/settings/invitations-list-item.tsx` | CustomSelect:102 | `5  /  15  /  20  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/workspace/settings/member-columns.tsx` | CustomSelect:129 | `5  /  15  /  20  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/workspace/invite-modal/fields.tsx` | CustomSelect:98 | `5  /  15  /  20  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/automation/auto-archive-automation.tsx` | CustomSelect:109 | `number  /  null` | 数字与 custom 哨兵由真实 callback 区分；独立 Option 泛型无法证明父子相等，不冒称此保证。 |
| `web/apps/web/core/components/project/form.tsx` | CustomSelect:239 | `0  /  2  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/project/settings/member-columns.tsx` | CustomSelect:103 | `5  /  15  /  20  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx` | CustomSearchSelect:170 | `string  /  string[]  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx` | CustomSelect:222 | `5  /  15  /  20  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/apps/web/core/components/project/member-select.tsx` | CustomSearchSelect:66 | `string  /  string[]  /  null` | 移除 value:any 与 filter 类型断言；明确 nullable string，none 哨兵及调用者转 null 不变。 |
| `web/apps/web/core/components/workspace/create-workspace-form.tsx` | CustomSelect:131 | `"Just myself"  /  "2-10"  /  "11-50"  /  "51-200"  /  "201-500"  /  "500+"  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |
| `web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx` | CustomSearchSelect:63 | `string  /  string[]  /  null` | 现有值/回调与选项对应；编译保持合同，无需产品改动。 |

## T3 遮蔽与屏蔽逐文件处理（2026-11-01）

准备35文件中，module/progress-sidebar 的一条已在T2删除。T3解除屏蔽后45条/32文件，加两处无效屏蔽共34文件；完整原始诊断和词法rename位置见 t3-unsuppressed-rows.json、t3-renames.json。

| 路径 | 原绑定 | 处理 |
|---|---|---|
| `web/apps/web/core/components/breadcrumbs/project.tsx` | projectId | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/cycles/active-cycle/cycle-stats.tsx` | tab | 移除多余选项映射函数；分组空项使用稳定key |
| `web/apps/web/core/components/cycles/form.tsx` | projectId | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/cycles/modal.tsx` | projectId | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/dropdowns/layout.tsx` | props | 现有 CustomSelect 接管；删旧闭包/断言，语言与键盘回归 |
| `web/apps/web/core/components/inbox/content/issue-root.tsx` | workspaceSlug, projectId | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/inbox/inbox-filter/applied-filters/date.tsx` | InboxIssueAppliedFiltersDate | 私有Props类型改名；可移除项改具名原生按钮，实际值回归 |
| `web/apps/web/core/components/inbox/inbox-filter/applied-filters/member.tsx` | InboxIssueAppliedFiltersMember | 私有Props类型改名；可移除项改具名原生按钮，实际值回归 |
| `web/apps/web/core/components/inbox/modals/snooze-issue-modal.tsx` | date | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/issues/issue-modal/form.tsx` | data | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/modules/form.tsx` | projectId | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/navigation/use-navigation-items.ts` | workspaceSlug, projectId | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/ui/loader/layouts/calendar-layout-loader.tsx` | index | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/views/form.tsx` | logoValue | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/workspace/sidebar/favorites/favorite-folder.tsx` | element, instruction | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/root.tsx` | element, instruction | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx` | element | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/components/editors/link-view-container.tsx` | editor | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/components/menus/bubble-menu/root.tsx` | editor, editor | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/extensions/custom-image/components/upload-status.tsx` | editor | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/extensions/image/extension.tsx` | props | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/extensions/table/plugins/drag-handles/column/drag-handle.tsx` | table | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/extensions/table/plugins/drag-handles/row/drag-handle.tsx` | table | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/extensions/utility.ts` | asset | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/helpers/editor-commands.ts` | range | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/helpers/editor-ref.ts` | itemKey, args, itemKey | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/editor/src/plugins/drag-handle.ts` | view, view, view, view | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/propel/src/calendar/root.tsx` | className, props | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/propel/src/charts/area-chart/root.tsx` | props, props | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/packages/ui/src/dropdown/common/options.tsx` | query | 迁移最后消费者后删除整个旧链及独占依赖 |
| `web/packages/ui/src/dropdown/single-select.tsx` | options, query | 迁移最后消费者后删除整个旧链及独占依赖 |
| `web/packages/utils/src/editor/markdown-parser/root.ts` | child | 按实际词法符号改局部名；对象key和公共回调合同保持 |
| `web/apps/web/core/components/issues/issue-detail/main-content.tsx` | 无有效诊断 | 删除失效 no-shadow 屏蔽 |
| `web/apps/web/core/components/issues/peek-overview/issue-detail.tsx` | 无有效诊断 | 删除失效 no-shadow 屏蔽 |
