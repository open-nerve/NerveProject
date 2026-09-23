# M1/P2 `trim-content`：内容类功能 Spec

- 上级设计：[M1 设计](../M1-design.md)（2.1、3.1–3.4、3.11、3.14、7.4、7.5、9 节 P2）
- 前一 Phase：[P1 spec](P1-web-hygiene.md)、[P1 plan](../plans/P1-web-hygiene.md)、[P1 review](../reviews/P1-web-hygiene-review.md)
- 实施计划：[P2 plan](../plans/P2-trim-content.md)
- 分支：`worktree-m1-p2-trim-content`，基线 `57fccb6`（P1 合并后的 `main`）

---

## 1. 目标

删掉 M1 设计 2.1 列出的全部"内容类功能"，一次删到底：路由和入口 → 组件、store、service、hook →
类型和字段 → 常量和枚举 → 中英文文案 → 依赖和图片资源。不留开关、不留空实现、不留兼容转换。

同时把三处保留功能的形态固定下来（M1 设计 3.1、3.2、3.3、3.14），把迭代和模块的进度改成只按工作项数
计算（3.4），并给每个删掉的功能加一条关键词守卫规则，防止它重新长回来。

P2 不碰账户与平台（P3）、不碰 Next.js 兼容层（P4）、不碰品牌（P5）。

---

## 2. 交付物

### 2.1 文件总览

12 个 Task，每个一个提交。原型（`57fccb6..e0d831b`）实测总计
`862 files changed, 1673 insertions(+), 46198 deletions(-)`：删 575 个文件、新增 9 个、修改 267 个、
重命名 11 个。

| Task | 标题 | 删 | 增 | 改 | 重命名 |
|---|---|---:|---:|---:|---:|
| 1 | 侧边栏改为固定列表 | 3 | 1 | 19 | — |
| 2 | 数据分析、活跃迭代推广页；进度代码改名 | 88 | — | 55 | 11 |
| 3 | 个人主页只剩工作项标签和用户卡片 | 20 | 2 | 20 | — |
| 4 | 估算；进度改为只按工作项数计算 | 53 | 1 | 87 | — |
| 5 | 甘特图与模块时间线 | 73 | — | 60 | 1 |
| 6 | 自动关闭 | 1 | — | 8 | — |
| 7 | 文档页、协作编辑、更新日志 | 203 | — | 110 | — |
| 8 | 编辑器内核收敛 | 3 | — | 6 | — |
| 9 | 便签 | 37 | — | 33 | — |
| 10 | 首页固定内容、自定义主题、旧仪表盘 | 59 | 2 | 33 | — |
| 11 | 导出与从未创建过的集成服务 | 28 | — | 20 | — |
| 12 | 收尾：遗留依赖、编辑器测试、文档 | — | 2 | 5 | — |

Task 的顺序就是 M1 设计 9 节 P2 的顺序（差异见第 3 节）。

### 2.2 原型验证：结论与证据

在 `$P2TMP/proto`（从本分支克隆）中逐个 Task 做过一遍，每个 Task 结束时类型检查、`make lint-web`、
`make test-web`、`make build-web` 都通过。下面是基线与终态的实测值，plan 的每一步都写了当步的预期输出。

| 项 | 基线（`57fccb6`） | P2 结束 |
|---|---|---|
| `make lint-web` | `keywords: 12 rules, 0 exceptions, no hits.` + 52 个 turbo 任务 | `keywords: 24 rules, 14 exceptions, no hits.` + 52 |
| `make test-web` | 12 | 15 |
| `make build-web` | 11 | 11 |
| lint 上限合计 | 954 | 815 |
| — web | 777 | 661 |
| — editor | 75 | 67 |
| — utils | 34 | 27 |
| — propel | 29 | 21 |
| — ui / hooks / constants / types / i18n | 31 / 4 / 2 / 1 / 1 | 不变 |
| knip：未使用文件 | 123 | 94 |
| knip：未使用依赖 / 开发依赖 | 1 / 0 | 0 / 0 |
| knip：未使用导出 / 类型 / 枚举成员 | 133 / 81 / 4 | 111 / 61 / 4 |
| knip：重复导出 / 配置提示 | 1 / 1 | 1 / 1 |
| 构建体积 js | 505 个 / 10,270,419 字节 | 439 个 / 7,369,397 字节 |
| 构建体积 css | 3 / 322,306 | 3 / 307,728 |
| 构建体积 fonts | 34 / 6,849,620 | 25 / 3,755,608 |
| 构建体积 other | 142 / 9,719,426 | 122 / 8,043,211 |
| 最大 chunk | `toolbar` 1,821,937 | `use-editor-flagging` 1,382,279 |
| 语言 chunk | 40 | 34 |

结论：

1. **类型检查是删除的向导。** 每个 Task 都是"删文件 → 跑 `check:types` → 按报错删下一层"，
   直到 22 个任务全部通过。最后才删枚举值（例如 `EIssueLayoutTypes.GANTT`），好让类型检查把所有
   菜单、图标、筛选分支列出来。
2. **删除会让 oxlint 警告先变多。** 删掉一段代码留下的未使用 import 和局部变量会让
   `lint-cap.mjs` 报 "more warnings than the cap"。每个 Task 结束前要按它列出的 `no-unused-vars`
   逐个清理，再调低上限。
3. **增量编译文件会造假报错。** 删完文件后 `check:types` 偶尔在没动过的行上报 TS2339；
   删掉 `.turbo/tsconfig.tsbuildinfo` 后消失。plan 的 Global Constraints 里写了这条。
4. **守卫例外的 `match` 必须是正则实际匹配到的那段文本**，不是整词：`estimat` 而不是 `estimates`，
   `Gantt` 而不是 `GanttChart`。写错时工具报 "stale exception … matches nothing now"。
5. **删路由文件必须同一步删 `app/routes/core.ts` 的条目**，否则 `react-router typegen` 报 ENOENT，
   `check:types` 全线失败。
6. **锁文件只由 `pnpm install` 改写**，每次删依赖后用 `lock-diff.mjs` 核对，只允许删除直接引起的差异。

### 2.3 侧边栏改为固定列表（Task 1，M1 设计 3.3）

- 删除自定义导航弹窗中的"个人""工作区"两节、`/sidebar-preferences/` 的读写、偏好 store 和
  `ExtendedAppSidebar`（第二层侧边栏，只用来放被隐藏的菜单项）。
- **裁定**：弹窗不整删。它的"项目"一节配置的是**保留**的项目导航偏好
  （`workspace_user_properties`，M1 设计 3.3 明写保留），而且是它唯一的写入入口。整删会让保留功能
  失去入口。所以按功能把弹窗拆开：保留项目一节、改名为 `ProjectNavigationDialog`，删掉另外两节。
- **裁定**：固定列表按常量里的声明顺序**全部**显示（Home / 便签 / 我的工作 / 草稿；工作区组：
  项目 / 视图 / 数据分析 / 归档）。设计里的"顺序取现在的默认顺序"若按 `is_pinned` 默认 `false`
  理解，视图和归档会一个都不显示，等于删掉保留功能的唯一侧边栏入口。
  （便签在 Task 9 移出，数据分析在 Task 2 移出。）
- 顺带删除：`hasPageAccess`（零调用方，是动态导航常量在侧边栏之外的唯一读取方）、
  常量记录里的 `inbox` 条目（零引用，通知入口在顶部导航）。

### 2.4 数据分析、活跃迭代推广页；进度代码改名（Task 2）

- 删除工作区数据分析的路由、`:workspaceSlug/analytics` 旧地址重定向（M1 设计 3.12 的 11 条中，
  只有这一条属于 P2）、侧边栏入口、power-k 命令、store、service、hook、常量、类型、
  `helpers/graph.helper.ts` 和 13 个图片。
- 删除工作区级的"活跃迭代"升级推广页（3 个路由 + 推广组件 + 6 个图片）。**项目迭代列表中的
  "当前迭代"区块保留**，所以守卫规则不搜单数的 `active_cycle` / `ActiveCycle`。
- 删除 propel 中只给数据分析用的图表与表格：`line-chart`、`radar-chart`、`scatter-chart`、
  `tree-map`、`table/`，以及 `icons/workspace/analytics-icon.tsx`。燃尽图用的 `area-chart` 和
  `recharts` 保留。`bar-chart`、`pie-chart` 由个人主页统计（Task 3）最后一个用完再删（M1 设计 9 节"硬"约束）。
- **改名（M1 设计 3.4）**：保留的迭代、模块进度代码从 analytics 改名为 progress：
  - `cycles/analytics-sidebar/` → `cycles/progress-sidebar/`，`modules/analytics-sidebar/` → `modules/progress-sidebar/`；
  - `CycleAnalyticsProgress` → `CycleProgress`，`ModuleAnalyticsSidebar` → `ModuleProgressSidebar`，
    `ModuleAnalyticsProgress` → `ModuleProgress`；
  - `fetchActiveCycleAnalytics` → `fetchActiveCycleDistribution`，
    `workspaceActiveCyclesAnalytics` → `cycleDistribution`，`workspaceActiveCyclesProgress` → `cycleProgress`；
  - localStorage 键 `cycle-analytics-tab-*`、`module-analytics-tab-*` → `cycle-progress-tab-*`、`module-progress-tab-*`。
- Plane 进度接口地址本身带 `analytics`（`/cycles/${cycleId}/analytics?type=`），M1 不改，
  守卫按地址字面量开精确例外，到 M6（交接）。
- 删依赖 `@tanstack/react-table`、`export-to-csv`（只有数据分析的表格在用）。

### 2.5 个人主页（Task 3，M1 设计 3.2）

- 个人主页只剩"分配给他的 / 他创建的 / 他关注的"三个工作项标签和右侧用户卡片；
  统计页、动态页和每个项目的统计数字删除。
- **裁定**：`profile/[userId]/page.tsx` 换成 `app/routes/redirects/core/profile-index.tsx`
  （带查询参数透传的 `clientLoader` 重定向到 `/assigned`），侧边栏"我的工作"、`g y`、成员链接、
  @提及都不用改。
- `profile/sidebar.tsx` 重写为用户卡片，数据取自工作区成员 store，四种状态分开显示：
  加载中 / 加载失败 / 不是本工作区成员 / 正常。卡片只显示头像、名字和加入时间（`joining_date`），
  时区删掉（成员数据里没有）。页头的名字同样改从 `getWorkspaceMemberDetails` 取。
- `PROFILE_VIEWER_TAB` + `PROFILE_ADMINS_TAB` 合并为一个 `PROFILE_TABS`；
  `UserService` 删除 4 个统计方法；`IUserProfileData` 等 6 个类型删除。
- propel 的 `bar-chart`、`pie-chart` 在这里删（数据分析已先删，满足 9 节的"硬"约束）。
- **新增仓库内单元测试** `packages/constants/src/navigation.test.ts`：侧边栏固定列表的键与顺序、
  `PROFILE_TABS` 的内容。`@plane/constants` 因此获得 `test` 脚本，`make test-web` 12 → 13。

### 2.6 估算；进度改为只按工作项数计算（Task 4，M1 设计 3.4）

- 删除估算设置页、估算下拉框、估算属性、表格里的估算列、动态里的估算记录、
  power-k 的估算菜单、`estimate.service.ts`、三个 store、四个 hook、类型、常量、工具函数和 8 个图片。
- **数量计算和旧的估算字段在同一个 Task 改（M1 设计 9 节"硬"约束）**：
  - `utils/src/distribution-update.ts` 去掉 `estimatePointById` 参数、三个 `*_estimates` 字段、
    `total_estimate_points` 路径、`estimate_distribution.completion_chart` 和两段按点数的
    负责人 / 标签计数；
  - `utils/src/cycle.ts` 的 `calculateCycleProgress` 去掉 `estimateType` 参数，并删掉零调用方的
    `scope`、`ideal`、`formatV1Data`、`formatV2Data`、`formatActiveCycle`；
  - `constants/src/state.ts` 的 `STATE_DISTRIBUTION` 去掉每行的 `points`；
  - 类型里去掉 `TCycleEstimateDistribution`、六个 `*_estimate_points`、`TCycleEstimateType`、
    `TCyclePlotType`、`TModulePlotType`；
  - store 去掉 `plotType` / `estimatedType` 及其读写方法、`getIsPointsDataAvailable`，
    `fetchActiveCycleDistribution` 去掉 `analytic_type` 参数；
  - 表格布局里 `isEstimateEnabled` 的整条 prop 链删除，`helpers/issue-filter.helper.ts`
    （只判断估算这一列）随之删除。
- **新增仓库内单元测试** `packages/utils/src/progress.test.ts`（13 个），覆盖 M1 设计 7.5 "P2 进度"
  那一行要求的行为：`calculateCycleProgress` 7 个（含空值、取消项、含进行中、快照优先）；
  乐观更新 6 个（新建 / 完成 / 重新打开 / 删除 / 负责人与标签计数 / 燃尽图按完成日扣减）。
  `@plane/utils` 获得 `test` 脚本，`make test-web` 13 → 14。

### 2.7 甘特图与模块时间线（Task 5）

- **第一步**把保留的 `REVERSE_RELATIONS` 从 `constants/src/gantt-chart.ts` `git mv` 到
  `constants/src/issue/relation.ts`（M1 设计 9 节"硬"约束）。
- 删除甘特图组件树、工作项的甘特布局、模块的时间线布局、时间线 store、`issue_gantt_view.store.ts`、
  甘特的快速新建表单与按钮、加载态、`use-timeline-chart.ts`、类型和 10 个图片。
- `IssueService.updateIssueDates` 与 store 中的 `updateIssueDates`（`POST /issue-dates/`）删除
  （交接 M4）；`ENABLE_ISSUE_DEPENDENCIES` 删除。
- `useTimeLineRelationOptions` 删除（M1 设计 7.4 说"在 P2 改名"，实测它只是
  `() => ISSUE_RELATION_OPTIONS`，7 个调用方直接引用常量更干净）。
- `EIssueLayoutTypes.GANTT` **最后**删，由类型检查列出所有剩下的引用。

### 2.8 自动关闭（Task 6）

- 删除 `auto-close-automation.tsx`；"自动化"设置页只剩自动归档。
- `select-month-modal.tsx` 去掉 `type` prop 和 `close_in` 分支（只剩归档月份）。
- `IProject.close_in`、`IProject.default_state` 删除（交接 M3）。

### 2.9 文档页、协作编辑、更新日志（Task 7）

- 删除文档页的路由、组件、store、service、hook、类型、常量、工具函数、i18n 命名空间和图片；
  搜索 / 收藏 / 资源类型 / power-k / 最近访问中的文档页分支；导览中的文档页一步（导览本身保留，
  "视图"一步成为最后一步）。
- 删除更新日志（帮助菜单里的"新功能"弹窗、`IProductUpdateResponse`、`getProductUpdates`、
  `instance_changelog_url`）——它的类型借用文档页的 `TPage`。
- 编辑器包删除协作部分：协作文档编辑器、`contexts/collaboration-context.tsx`、
  `use-yjs-setup.ts`、`use-collaborative-editor.ts`、`parser.ts`、`title-extension.ts`、
  `work-item-embed/`、`document-collaborative-events`、`headings-list.ts` 等 32 个文件。
- 删依赖：`@hocuspocus/provider`、`yjs`、`y-indexeddb`、`y-prosemirror`、`y-protocols`、
  `@tiptap/extension-collaboration`、`@tiptap/extension-character-count`、`@tiptap/html`、`buffer`、
  `@react-pdf/renderer`、`react-pdf-html`；`turbo.json` 的 `globalEnv` 去掉 `VITE_LIVE_*`，
  `.env.example` 同步；`packages/services/src/live.service.ts` 删除。

### 2.10 编辑器内核收敛（Task 8，M1 设计 3.1）

- 删除：`plugins/ai-handle.ts`、`components/menus/ai-menu.tsx`、`types/ai.ts`、
  `SideMenuExtension` 的 `aiEnabled`（AI 只挂在协作文档编辑器上），以及
  `styles/variables.css` 中只给文档页用的 `/* layout config */` 块和两个内容边距变量。
- 保留并在测试里断言：`UniqueID`（所有编辑器都装，非协作编辑器靠节点 id 定位）、
  `copyMarkdownToClipboard`（描述历史在用）、用户 @提及、图片与附件节点。
- `EditorRefApi` 去掉协作成员这一步在 Task 7 完成（它们和协作文件在同一条编译链上）。

### 2.11 便签（Task 9）

- 删除便签页、首页便签组件、便签编辑器、store、service、hook、常量、类型、i18n 命名空间、
  propel 的 3 个便签图标和 `note` 插图、4 个图片；依赖 `react-masonry-component` 及其 catalog
  条目和 `react-masonry-component>react` 覆盖项。
- **工具栏收敛**：文档页编辑器和便签编辑器都没了，只剩一种编辑器，所以
  `packages/editor/src/constants/common.ts` 删掉 `TEditorTypes`、`ToolbarMenuItem.editors`、
  `TYPOGRAPHY_ITEMS`（只有文档页显示）和 `table` 项；`TOOLBAR_ITEMS` 从"按编辑器类型分组"
  改为一组 `{ basic, alignment, list, userAction, complex }`。

### 2.12 首页固定内容、自定义主题、旧仪表盘（Task 10，M1 设计 3.14）

- 首页正文按 3.14 固定下来。**新增** `core/components/home/home-body.tsx`：无项目空状态 +
  最近访问；问候语、导览和工作项 peek 仍由 `home/root.tsx` 挂载。
  `home-dashboard-widgets.tsx`（小组件驱动的正文）删除。
- 删除快捷链接（组件、空状态、加载态、store、`workspace.service.ts` 的四个方法）、
  "管理小组件"面板、首页个性化的两个接口方法、`HomeStore` 和 `useHome`。
- 删除已经没有入口的旧首页仪表盘：`DashboardStore`、`dashboard.service.ts`、`useDashboard`、
  `helpers/dashboard.helper.ts`、`types/src/dashboard.ts`、`constants/src/dashboard.ts`、
  `EDurationFilters` 和 20 个图片。
- 删除自定义主题：`core/theme/` 的 5 个组件、`packages/utils/src/theme/` 7 个文件、
  `ui/src/form-fields/input-color-picker.tsx`、`THEMES` 与 `THEME_OPTIONS` 中的 `custom`、
  `app/root.tsx` 的 `ThemeProvider themes`。**亮色、暗色、跟随系统和两个高对比度主题保留。**
- `IUserTheme` 只剩 `{ theme }`（交接 M2）。`packages/utils/src/theme-legacy.ts` 只剩
  `resolveGeneralTheme`，其中的"legacy"已无所指，`git mv` 为 `theme.ts`。
- `types/src/users.ts` 末尾注释掉的 `ICurrentUser`、`ICustomTheme`、`ICurrentUserSettings`
  三段死注释一起删（守卫会命中 `ICustomTheme`）。

### 2.13 导出与从未创建过的集成服务（Task 11）

- **两者同时删（M1 设计 9 节"硬"约束）**：导出记录列表借用了集成服务。
- 删除导出设置页及其路由、导出弹窗与表单、`project-export.service.ts`；
  `WORKSPACE_SETTINGS`、`GROUPED_WORKSPACE_SETTINGS`、`WORKSPACE_SETTINGS_ICONS` 和
  `TWorkspaceSettingsTabs` 同步。
- 删除集成服务：`core/services/integrations/` 4 个文件、GitHub / Jira / Slack 选择器、
  `single-integration-card.tsx`、已无调用方的 `project/integration-card.tsx`、
  `app_installation.service.ts`、`use-integration-popup.tsx`、
  `ui/loader/settings/{import-and-export,integration}.tsx`、`types/src/importer/`、
  `types/src/integration.ts`、3 个图片。集成设置页（`settings/(workspace)/integrations/page.tsx`）
  **本来就没有路由条目**，从未可达。
- i18n `integration` 命名空间整个删除：291 个键全部没有代码引用。
- `RESTRICTED_URLS` 去掉 `import`、`importers`、`integrations`、`integration`（M1 设计 3.11）。

### 2.14 收尾（Task 12）

- 删掉被删功能遗留、knip 报告为"未使用依赖"的 8 个依赖项：`@plane/editor` 的 `@plane/constants`、
  `@plane/ui`、`@tiptap/extension-document`、`@tiptap/extension-heading`、`@tiptap/extension-text`、
  `tippy.js`；`@plane/utils` 的 `chroma-js`、`@types/chroma-js`。catalog 同步删 6 条。
- **新增仓库内单元测试**（控制者裁定后，见第 3 节第 13 条）：`packages/editor/vitest.config.ts`
  （`environment: "jsdom"`、`resolve.mainFields: ["module", "main"]`）、真实 TipTap `Editor` 的交互测试，
  以及 `packages/editor/src/extensions/extensions.test.ts`（7 个）：断言不装协作扩展、撤销重做来自 starter kit 自己的 history、
  只读时 history 关闭、描述 / 评论 / 历史版本需要的节点仍在、禁用图片时图片节点不装、
  工具栏只有一组且没有只属于文档页的项。`make test-web` 14 → 15。
- `docs/v0/frontend-changes.md` 第二节：11 行改为"已完成 / M1/P2"，
  并把"活跃迭代推广页"从企业版残留那一行拆出来、把"从未创建过的集成服务"从死代码那一行去掉。

### 2.15 关键词守卫

P1 的 12 条规则之上新增 12 条，全部 `"phase": "M1/P2"`，在删完对应功能的同一个提交里加入
（M1 设计 7.4）。顶层 `"phase"` 在 Task 1 从 `M1/P1` 改成 `M1/P2`。

| 规则 | 正则 | 大小写 | Task |
|---|---|---|---|
| `sidebar-customization` | `sidebar-preferences`、`CustomizeNavigation`、`WORKSPACE_SIDEBAR_(DYNAMIC\|STATIC\|PREFERENCES)`、`ExtendedSidebarItem` 等确切符号 | 敏感 | 1 |
| `analytics` | `analytics` | 不敏感 | 2 |
| `active-cycles-promo` | `active-cycles`、`active_cycles`、`WorkspaceActiveCycles`、`\bworkspaceActiveCycles\b` 等 | 敏感 | 2 |
| `profile-stats` | `user-stats`、`user-profile/`、`user-activity/`、`IUserProfileData`、`profile\.stats`、`your_work_by_` 等 | 敏感 | 3 |
| `estimates` | `estimat\|估算` | 不敏感 | 4 |
| `gantt-timeline` | `gantt\|(?<!view_)time_?line\|issue-dates` | 不敏感 | 5 |
| `auto-close` | `close_in\|auto[-_ ]?close\|default_state` | 不敏感 | 6 |
| `pages-collaboration` | M1 设计 7.4 的确切清单（`\byjs\b` 以免命中 `linkifyjs`） | 不敏感 | 7 |
| `changelog` | `changelog\|product-updates\|ProductUpdates\|what.s new` | 不敏感 | 7 |
| `stickies` | `Sticky\|STICKY\|[Ss]tickies\|STICKIES\|sticky[-_.:]` | **敏感** | 9 |
| `home-theme` | `quick_links\|quick-links\|QuickLink\|HomeWidget\|manage_widgets\|home-preferences\|CustomTheme\|customize_your_theme\|palette-generator\|DashboardStore\|useDashboard` | 不敏感 | 10 |
| `export` | `exporter\|settings/exports\|export-issues\|EXPORTERS_LIST\|IExportData` | 不敏感 | 11 |

14 条例外，每一条都精确到符号，`match` 写正则实际匹配到的那段文本：

| 规则 | 文件 | match | count | until |
|---|---|---|---|---|
| `analytics` | `web/packages/utils/src/tlds.ts` | `analytics` | — | M9 |
| `analytics` | `web/apps/web/core/services/cycle.service.ts` | `analytics` | — | M6 |
| `analytics` | `…/billing/comparison/plans.tsx` | `analytics` / `Analytics` | — / 3 | M1/P3 |
| `analytics` | `web/packages/types/src/epics.ts` | `Analytics` | 2 | M1/P3 |
| `estimates` | `…/notification-card/content.tsx` | `estimat` | — | M1/P3 |
| `estimates` | `…/billing/comparison/plans.tsx` | `Estimat` | — | M1/P3 |
| `gantt-timeline` | `…/billing/comparison/plans.tsx` | `Gantt` / `timeline` | 3 / 2 | M1/P3 |
| `gantt-timeline` | `web/packages/types/src/publish.ts` | `gantt` | 2 | M1/P3 |
| `pages-collaboration` | `…/billing/comparison/plans.tsx` | `Wiki` / `wiki` | — | M1/P3 |
| `pages-collaboration` | `web/packages/constants/src/subscription.ts` | `Wiki` | — | M1/P3 |
| `pages-collaboration` | `web/packages/utils/src/tlds.ts` | `wiki` | — | M9 |

`M9` 表示"v0 内不到期"（v0 到 M8 结束）。指向 `M1/P3` 的 11 条都在计费文案、企业版类型和公开发布
类型里，P3 删掉它们时例外随之消失；`cycle.service.ts` 的一条是 Plane 的进度接口地址，M6 换接口时消失。

### 2.16 文案与图片资源的口径

- **只删本 Phase 弄成没人引用的**：用"整键字面量 + 模板前缀"的引用报告，对比 Task 前后的差集；
  守卫命中的键一并删除。P1 结束时就已经没人引用的 564 个键、254 个图片属于"Plane 自带的死资源"，
  由 M1 收尾统一处理（M1 设计 11 节），不在 P2。
- 中英文成对删除，`make lint-web` 的键一致性检查是门禁。
- i18n 命名空间整删时（`page`、`stickies`、`integration`）同时删掉
  `packages/i18n/src/constants/namespaces.ts` 里的条目。

### 2.17 保留行为的核对（M1 设计 7.5）

| 7.5 的行 | 手段 | 落点 |
|---|---|---|
| P2 进度 | **进仓库**的 `packages/utils/src/progress.test.ts`（13 个） | Task 4 |
| P2 编辑器 | **进仓库**的交互测试（真实 TipTap `Editor`：输入、Markdown、撤销重做、只读、@成员、图片、节点 id）+ 组成测试 `extensions.test.ts`（7 个）；描述历史的查看和还原（web 应用层）用 review 附录的临时核对脚本 | Task 12 + review |
| P2 首页、个人主页、侧边栏 | **进仓库**的 `packages/constants/src/navigation.test.ts` + review 附录的临时核对脚本 | Task 3 + review |
| P2、P3 动态、通知、工作项列表 | review 附录的临时核对脚本 | review |

临时核对脚本不进仓库（M1 设计 7.5）：全文、假数据、运行命令和断言写进 P2 review 的附录。

---

## 3. 与上级设计的差异和补充

每一条都已按 M1 设计"能自己定的就自己定"的原则决定，这里列出决定和理由，供控制者复核。

1. **Task 划分比 9 节的顺序多两个。** 9 节 P2 列了 10 条顺序，其中没有"自动关闭"和
   "活跃迭代推广页"（它们在 2.1 的表里）。决定：活跃迭代推广页并进数据分析那一个 Task
   （同属工作区级入口，改同一批侧边栏和迭代 store 文件）；自动关闭独立成 Task 6，放在甘特图和
   文档页之间（它只改项目设置，和谁都不冲突）。其余 10 条顺序原样遵守，包括 5 条"硬"约束。

2. **自定义导航弹窗拆开而不是整删（Task 1）。** 见 2.3。整删会让保留的项目导航偏好失去唯一入口。

3. **固定侧边栏显示常量里的全部条目（Task 1）。** 见 2.3。按 `is_pinned` 默认值理解会删掉
   "视图"和"归档"的唯一入口。

4. **propel 里删掉的图表比设计列的多（Task 2）。** 设计只点了 `bar-chart`、`pie-chart`。实测
   `line-chart`、`radar-chart`、`scatter-chart`、`tree-map` 和 `table/` 的唯一消费者也是数据分析，
   一并删除；`area-chart` 和 `recharts` 按设计保留。

5. **数据分析的文案里有 Epic 的键（Task 2）。** `common.epics` 等是数据分析页面的标签，随页面删除，
   虽然 Epic 功能本身属于 P3。守卫规则不受影响（P3 的 Epic 规则按确切符号写）。

6. **`useTimeLineRelationOptions` 删除而不是改名（Task 5）。** 设计 7.4 备注写"在 P2 改名"。
   实测它的实现就是 `() => ISSUE_RELATION_OPTIONS`，7 个调用方直接引用常量，少一层间接。

7. **工具栏收敛落在便签那个 Task（Task 9），不在编辑器收敛（Task 8）。**
   `TOOLBAR_ITEMS` 按编辑器类型分组，只有文档页**和便签**都删掉之后才能塌成一组。

8. **页面专用的 CSS 变量落在 Task 8，不在 Task 7。** 它们在 `packages/editor/src/styles/variables.css`，
   和 AI 的删除在同一个文件层，一起做只有一个提交动这个文件。

9. **首页新增了一个文件（Task 10）。** 删除型 Phase 里新增 `home-body.tsx` 是 M1 设计 3.14 的直接要求：
   唯一渲染"最近访问"的地方是要删的小组件列表，顺着编译报错删会把保留功能一起删掉。

10. **`theme-legacy.ts` 改名为 `theme.ts`（Task 10）。** 删掉新的 OKLCH 主题目录之后，
    文件里只剩 `resolveGeneralTheme`，"legacy"已无所指，文件头的注释也全部指向已删除的文件。

11. **i18n 的 `integration` 命名空间在 P2 整删（Task 11）**，虽然 `integration` / `jira` / `slack`
    的守卫规则按设计 7.4 归 P3。理由：291 个键在删掉集成服务后一个都没有引用，设计 2.3 要求
    "文案最后删、删到没有引用为止"。P3 的规则届时只会看到更少的命中。

12. **遗留依赖集中在收尾删（Task 12）。** 造成它们的是 Task 7、8、10。决定集中删：锁文件只动一次，
    `lock-diff.mjs` 的输出可复现；每个 Task 内分别删会产生三次互相叠加的锁文件差异，难以逐条解释。

13. **编辑器的交互行为有仓库内测试（Task 12；控制者裁定，推翻原型时的决定）。** 原型用 jsdom + 真实 TipTap
    `Editor` 写交互测试时，每个进程的第一个 `new Editor` 必定抛
    `RangeError: Adding different instances of a keyed plugin (plugin$)`，原型因此改为只断言组成。
    控制者另派调查查到根因：锁文件里 prosemirror-state 只有一份，问题是 CJS/ESM 双包——
    `prosemirror-codemark` 没有 `exports`，vitest 按 CJS 解析它，它 `require` 到 prosemirror-state 的
    `.cjs`，而 `@tiptap/pm/state` 经 ESM 拿到 `.js`，同一个包两个实例。编辑器包的 `vitest.config.ts` 写
    `resolve.mainFields: ["module", "main"]` 即可，这与生产构建的解析方式一致，不是测试特例。
    **决定**：Task 12 加 `jsdom`（P2 唯一的新增依赖），交互测试进仓库；原定的 7 个组成测试保留；
    临时去掉那一行配置，证明它是必需的。描述历史的查看和还原属于 web 应用层，仍用 review 附录的临时脚本。
    交给 P4 的"重新评估"一项取消。

14. **基线就已经死掉的文案和图片不在 P2 删。** 见 2.16。

15. **`.superpowers/` 的忽略（控制者裁定）。** 不改仓库的 `.gitignore`：控制者的 SDD 工作区由它自己的脚本
    写入 `.superpowers/sdd/.gitignore`（内容 `*`），`git status --short` 看不到它。原型时显示 `?? .superpowers/`，
    是因为目录是手工建的。这一点要紧：关键词守卫扫描未跟踪、未忽略的文件，工作区里的任务说明会引用被删的关键词。

---

## 4. 验收标准

- [ ] 12 个 Task 各一个提交，每个提交结束时 `pnpm exec turbo run check:types` 22 个任务通过。
- [ ] `make lint-web` 通过：`keywords: 24 rules, 14 exceptions, no hits.` + 52 个 turbo 任务；
      每个包的 oxlint 警告数**等于**上限（合计 815）。
- [ ] `make test-web` 15 个任务通过；`make build-web` 11 个任务通过。
- [ ] 关键词守卫：本 Phase 的 12 条规则没有未登记的命中；14 条例外全部精确到符号，
      `until` 指向 M1/P3、M6 或 M9。
- [ ] knip 不再报告这些功能的任何文件、导出和依赖；未使用依赖和开发依赖为零。
- [ ] 锁文件核对：每次删依赖后 `lock-diff.mjs` 的输出与 plan 一致，只有删除直接引起的差异。
- [ ] 中英文键集合相同（`check:sync`），没有本 Phase 弄成无引用的键。
- [ ] M1 设计 7.5 中 P2 的四行核对过：三处进仓库的测试通过，临时脚本写进 review 附录。
- [ ] S1–S4 通过；持续集成的 `server`、`web`、`e2e` 三个任务通过。
- [ ] `docs/v0/frontend-changes.md` 第二节同步；M1 设计 12 节 P2 一行更新。

---

## 5. 不在 P2 范围内

- 账户、平台与死代码（P3）：公开发布、管理后台、认证、邮件、AI、Unsplash、遥测、企业版残留、
  Plane 自身的死文件与死导出、`integration` / `importer` / `jira` / `slack` 的守卫规则。
- Next.js 兼容层和另外 10 条旧地址重定向（P4）。
- 品牌（P5）。
- P1 结束时就已经无引用的 564 个文案键和 254 个图片（M1 收尾）。
- oxlint 警告清零（M2–M8，M1 设计 7.3）。
- 任何后端改动：被删功能对应的 Plane 接口字段按第 7 节交接。

---

## 6. 风险

| 风险 | 应对 |
|---|---|
| 一次删 575 个文件，保留功能的唯一入口可能被一起删掉 | 三处已知的入口陷阱（项目导航偏好、固定侧边栏、个人主页地址、首页最近访问）在 spec 里逐条裁定；7.5 的核对覆盖首页、个人主页、侧边栏、编辑器、进度 |
| 共享文件（根 store、命令面板 store、工作项 store、编辑器包、路由表、导出文件）被多个 Task 改到 | 按 9 节顺序串行；每个 Task 一次改完它在共享文件里的全部引用；每个 Task 结束时三个门禁都过 |
| 删代码之后 oxlint 警告变多，上限核对失败 | plan 的 Global Constraints 写明流程：先按 `no-unused-vars` 清理，再调低上限 |
| 增量编译文件造成假报错 | plan 的 Global Constraints：`check:types` 报不该有的错时先删 `.turbo/tsconfig.tsbuildinfo` |
| 文案删多或删少 | 引用报告的差集 + 守卫命中 + 键一致性门禁三重核对 |
| 编辑器的交互行为没有仓库内测试覆盖 | 组成用仓库内测试守住；交互用 review 附录里可重跑的临时脚本；P4 重新评估测试环境的模块解析 |

---

## 7. 移交事项

### 7.1 交给后续 M（写进对应 M 的 `handoffs/`）

| M | 事项 |
|---|---|
| M2 | `IUserTheme` 只剩 `{ theme }`：新接口的用户主题字段不要再有 `primary`、`background`、`darkPalette` |
| M3 | `IProject` 不再读 `close_in`、`default_state`、`page_view`、`estimate_id`；`/sidebar-preferences/` 接口不再需要；`RESTRICTED_URLS` 已去掉 `import`、`importers`、`integrations`、`integration`、`pages`，M3 让后端的保留名单与之一致；访客能否查看个人主页由 M3 的权限矩阵决定 |
| M4 | 工作项不再读 `estimate_point`；`POST /issue-dates/` 不再调用；搜索结果不再有 `page` 类型；`description_binary` 不再读取 |
| M5 | 资源类型 `PAGE_DESCRIPTION` 和 `file_assets.page_id` 不再使用 |
| M6 | 迭代和模块的进度不带 `points`、`estimate_distribution`、各类 `*_estimate_points` 和快照中的点数分布；守卫里 `cycle.service.ts` 的 `analytics` 地址例外随进度接口的替换一起消失 |
| M7 | 收藏和最近访问不再有 `page` 类型；通知实体中已删功能的类型；首页的最近访问接口由 M7 替换 |

### 7.2 交给 P3

- `integration`、`importer`、`jira`、`slack`、`github-repository`、`app-installation`、`indexeddb`
  的守卫规则（M1 设计 7.4 的死代码一行）：P2 已经删掉集成服务本身，P3 加规则时命中会更少。
- 11 条 `until: M1/P3` 的守卫例外：删掉计费文案（`plans.tsx`、`subscription.ts`）、
  企业版类型（`epics.ts`）和公开发布类型（`publish.ts`）时，例外随之删除。
- knip 仍报告的 94 个未使用文件、111 个未使用导出、61 个未使用类型：P3 逐个复核后删除，
  之后 `make knip` 去掉 `--no-exit-code` 成为门禁。

### 7.3 交给 P4

- `routes/core.ts` 末尾剩下的 10 条旧地址重定向（数据分析那一条已在 P2 删除）。

### 7.4 交给 M1 收尾

- P1 结束时就已经无引用的 564 个文案键和 254 个图片。
- 构建体积对比（本 spec 2.2 的表）写进收尾 review。
