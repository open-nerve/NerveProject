# M1/P3 `trim-platform`：账户、平台与死代码 Spec

- 上级设计：[M1 设计](../M1-design.md)（2.2、2.3、3.6–3.9、3.11、3.13、3.15、7.2、7.4、7.5、9 节 P3）
- 前一 Phase：[P2 spec](P2-trim-content.md)、[P2 plan](../plans/P2-trim-content.md)、[P2 review](../reviews/P2-trim-content-review.md)（第 4 节裁定、第 7 节"交给 P3"）
- 实施计划：[P3 plan](../plans/P3-trim-platform.md)
- 分支：`worktree-m1-p3-trim-platform`，基线 `6d9692b`（P2 合并后的 `main`）

---

## 1. 目标

删掉 M1 设计 2.2 列出的全部"账户、平台与死代码"，一次删到底：路由和入口 → 组件、store、service、hook →
类型和字段 → 常量和枚举 → 中英文文案 → 图片资源。不留开关、不留空实现、不留兼容转换。

具体是：公开发布（连同评论的可见范围）、管理后台入口、第三方登录和验证码登录、找回 / 重置 / 设置密码、
登录前的"检查邮箱"步骤、邮件通知偏好和营销邮件同意、修改登录邮箱、AI 助手、Unsplash 封面、遥测残留、
项目邀请的名字，企业版残留（Epic、团队、工作项类型、计费和升级提示、批量操作与工作项多选、模板 / 工时 /
重复工作项 / 工作流 / 项目更新的空壳、企业版扩展点），以及 Plane 自身的死代码。

同时把这几处定下来：

- 登录页和注册页各剩一个"邮箱 + 密码"表单，模式由路由决定（M1 设计 3.6）；
- 实例配置只剩 4 个字段，从实际改动导出"删除 / 保留 / 等待重新定义"清单交给 M2（3.7）；
- knip 清零，`make knip` 改为门禁（7.2）；
- P2 评审第 7 节交给 P3 的全部事项。

P3 不碰 CSRF 和会话（M2，3.6）、Next.js 兼容层和另外 10 条旧地址重定向（P4）、品牌（P5）、
基线就已无引用的通用文案和图片（M1 收尾）。

---

## 2. 交付物

### 2.1 文件总览

7 个 Task，每个一个提交。原型（`6d9692b..98025c6`，在 `$P3TMP/proto`）实测总计
`741 files changed, 2366 insertions(+), 25501 deletions(-)`：按文件去重后删 281 个、新增 2 个、修改 452 个、
重命名 6 个。下表是每个提交各自的数字（同一个文件会被几个 Task 修改，所以"改"一列加起来多于 452）。

| Task | 标题 | 删 | 增 | 改 | 重命名 | 原型提交 |
|---|---|---:|---:|---:|---:|---|
| 1 | 认证、实例配置、邮件：登录和注册各剩一个"邮箱 + 密码"表单 | 53 | — | 51 | — | `0d85d9d` |
| 2 | 公开发布，连同评论的可见范围 | 9 | — | 40 | — | `381a34c` |
| 3 | Epic、团队、工作项类型、批量操作与多选，以及模板、工时、重复工作项、工作流、项目更新的空壳 | 40 | — | 225 | 1 | `7efb8ff` |
| 4 | AI 助手（含 Plane AI）、Unsplash 封面、遥测 | 8 | — | 26 | — | `3d08bf5` |
| 5 | 计费和升级提示；守卫的 `phase` 改为 `M1/P3` | 31 | — | 29 | — | `301369a` |
| 6 | 企业版扩展点；项目"邀请"改为添加成员 | 27 | 1 | 89 | 5 | `c08cc32` |
| 7 | Plane 自身的死代码和 P2 评审留下的事项；knip 改为门禁 | 113 | 1 | 114 | — | `98025c6` |

Task 7 的"增 1"是 `issue-detail/links/types.ts`：原来的 `links/root.tsx` 删到只剩一个类型，`git mv` 改名，
内容变化太大，git 记作删 1 增 1。

Task 1–5 的顺序就是 M1 设计 9 节 P3 的第 1–5 步；第 6 步"死代码、接线、图片资源、knip 门禁"拆成 Task 6 和 7
（差异见第 3 节第 1 条）。

### 2.2 原型验证：结论与证据

在 `$P3TMP/proto`（从本分支的基线克隆）中逐个 Task 做过一遍，每个 Task 结束时类型检查、`make lint-web`、
`make test-web`、`make build-web` 都通过；Task 7 另外用脚本从 Task 6 重放了一遍，得到的树与提交完全相同。
下面是基线与终态的实测值，plan 的每一步都写了当步的预期输出和它来自哪个原型提交。

| 项 | 基线（`6d9692b`） | P3 结束（`98025c6`） |
|---|---|---|
| `make lint-web` | `keywords: 24 rules, 14 exceptions, no hits.` + 52 个 turbo 任务 | `keywords: 42 rules, 4 exceptions, no hits.` + 52 |
| `pnpm exec turbo run check:types` | 23 | 23 |
| `make test-web` | 15 个任务（constants 5 个测试、editor 14 个） | 15 个任务（constants 11 个、editor 16 个） |
| `make build-web` | 11 | 11 |
| `make knip` | 只出报告：文件 85、导出 108、类型 61、枚举成员 4 | 门禁，零；先生成路由类型、带 `--treat-config-hints-as-errors` |
| lint 上限合计 | 809 | 713 |
| — web | 658 | 568 |
| — editor | 67 | 65 |
| — utils | 27 | 25 |
| — constants / types | 2 / 1 | 1 / 0 |
| — ui / propel / hooks / i18n | 28 / 21 / 4 / 1 | 不变 |
| 每种语言的文案键 | 2179 | 1687 |
| 无引用的键（严格方法，2.12） | 969 | 570 |
| 图片（无引用 / 全部） | 158 / 286 | 142 / 257 |
| web 的 service 调用 / Plane 不路由的 | 262 / 51（多数是 `${serviceType}` 多态地址造成的误报） | 232 / 4（都只差结尾斜杠，2.9） |
| 构建体积 js | 439 个 / 7,319,739 字节 | 422 个 / 7,009,402 字节 |
| 构建体积 css | 3 / 305,244 | 3 / 297,032 |
| 构建体积 fonts | 25 / 3,755,608 | 不变 |
| 构建体积 other | 122 / 8,043,211 | 112 / 7,910,515 |
| 最大 chunk | `use-editor-flagging` 1,381,850 | `use-parse-editor-content` 1,380,566 |
| 语言 chunk | 34 | 34 |

每个 Task 结束时的中间值：

| 原型提交 | 键 | 无引用的键 | 图片（无引用 / 全部） | knip 文件 / 导出 / 类型 | lint 上限合计 | 守卫规则 / 例外 |
|---|---:|---:|---|---|---:|---|
| 基线 `6d9692b` | 2179 | 969 | 158 / 286 | 85 / 108 / 61（+4 枚举成员） | 809 | 24 / 14 |
| T1 `0d85d9d` | 1984 | 831 | 158 / 280 | 80 / 107 / 60 | 781 | 28 / 22 |
| T2 `381a34c` | 1959 | 809 | 158 / 280 | 78 / 107 / 60 | 781 | 30 / 21 |
| T3 `7efb8ff` | 1804 | 670 | 151 / 273 | 72 / 105 / 52 | 767 | 36 / 24 |
| T4 `3d08bf5` | 1795 | 666 | 151 / 273 | 69 / 105 / 51 | 759 | 38 / 24 |
| T5 `301369a` | 1759 | 637 | 151 / 271 | 67 / 104 / 51 | 754 | 39 / 3 |
| T6 `c08cc32` | 1758 | 635 | 151 / 271 | 64 / 96 / 44 | 748 | 41 / 4 |
| T7 `98025c6` | 1687 | 570 | 142 / 257 | 0 | 713 | 42 / 4 |

结论：

1. **类型检查仍是删除的向导，P3 多了"恒为假"的化简。** Epic 删掉之后 `isEpic` 恒为 `false`，它贯穿
   59 个文件；工作项的 service 类型在 Epic 走后只剩一个取值。做法是删掉参数，再把每个分支按常量化简，
   写出被选中的那一支。原型第一次化简时留下了 8 处 `${"sub-issues"}` 这样的模板字符串，已改成普通文本；
   plan 要求每个 Task 提交前 `git diff HEAD | grep -F '${"'` 为空。
2. **knip 看不到包的导出。** 工作区包的导出是入口，knip 默认不报；`--include-entry-exports` 在包构建过之后
   把 `@plane/*` 解析到 `dist/`，报出一片误报，不能用。原型写了 `symref.mjs orphaned <基点>`：列出基点时有别的
   文件用、现在没有文件用的导出。它找出了 knip 找不出的孤儿：Task 1 的 `PasswordInput`、Task 3 的
   `MARKETING_PLANE_ONE_PAGE_LINK`、Task 4 的 `IProjectLite` 和 `IWorkspaceLite`，都在各自的 Task 里删掉。
3. **文案用严格方法核对。** `keyref.mjs` 只把"整个键名的字符串字面量"和"模板或拼接的字面量前缀"算作引用
   （P2 评审裁定 12，也是 M1 收尾要用的方法）。P2 的 `keyuse.mjs` 按子串判断，会把常见词的键误判为有引用；
   基线上按严格方法无引用的键是 969 个。
4. **基点就已死、但带着本 Task 词汇的文件，由本 Task 删。** `deadvocab.mjs` 把 knip 报告的未使用文件和本 Phase
   的守卫规则对一遍，找出 8 处：评论的 `comments.tsx`（Task 2）、`common/activity/`、`layout-quick-actions.tsx`、
   power-k 的工作项选择页（Task 3）、侧边栏的 `user-menu.tsx`（Task 4，Plane AI 的入口）、`locked-component.tsx`
   （Task 5）、新手引导的 `profile-setup.tsx` 和 `current-user/profile.ts`（Task 1）。连同只有它们在用的代码一起删。
5. **lint 上限要在 knip 的孤儿删完之后再定。** 原型的 Task 3 先定了上限、后删 knip 报出的孤儿，
   上限把删除留下的一个未使用 import 算了进去；重做之后上限低 1。plan 的顺序是：删完 knip 孤儿 → `lintdiff.sh`
   → 清理新增的 `no-unused-vars` → 定上限。
6. **删除语句的工具不能带走许可证头。** 原型的 `unexport.mjs`、`deadlocals.mjs` 起初从语句的"完整起点"删，
   删掉文件的第一条语句时连文件头的许可证注释一起删了（`links/root.tsx`）。两个工具已改为从许可证头之后删；
   plan 在每个 Task 提交前跑 `headers.sh`，输出必须为空。
7. **守卫例外的到期。** 工具把 `until` 早于或等于顶层 `phase` 的例外当作过期。P2 留下 11 条 `until: M1/P3` 的例外，
   在它们全部消失之前，顶层 `phase` 不能改成 `M1/P3`；所以 Task 1–4 期间顶层仍是 `M1/P2`，本 Phase 内会被删掉的
   例外写 `until: M1/P4`，Task 5 删掉计费文案时这些例外随之消失，同一个提交把 `phase` 改成 `M1/P3`。
8. **Plane 的地址逐个核对过。** `endpoints.py` 读 Plane 的 Django 路由（只读），把 web 的每个 service 调用按方法和
   路径去匹配。死代码 Task 删掉了没有路由、也没有调用方的方法；剩下 4 处都只差结尾斜杠（Django 的
   `APPEND_SLASH` 会重定向），分别交给 M2、M3、M6（第 7 节）。
9. **每个实例自己的东西没有变成共享的（P2 的 `UniqueID` 回归）。** 原型逐个检查了新增的模块级常量和改动过的
   扩展创建：图片选择器的标签页、通知内容映射、收藏图标映射、筛选的 `OPERATORS` 都是只读数据；
   service 仍由每个 store 自己 `new`；`SlashCommands()` 每次调用都 `configure` 出新的扩展。没有把每个编辑器、
   每个 store 或每个表单自己的状态搬到模块级。

### 2.3 认证、实例配置、邮件（Task 1，M1 设计 3.6、3.7、3.13）

- **登录和注册**：删除 `accounts/{forgot-password,reset-password,set-password}` 三个路由（同一步删
  `routes/core.ts` 的条目）、`auth-forms/` 中的邮箱步骤、验证码、找回 / 重置 / 设置密码表单和 `form-root.tsx`；
  `AuthRoot` 只渲染错误提示、页头、`AuthPasswordForm` 和条款，模式直接取路由给的 `authMode`，邮箱可以编辑。
  删掉的 knip 孤儿：`auth-forms/common/{container,header}.tsx`、`hooks/use-timer.tsx`。
- **第三方登录**：`hooks/oauth/`、`ui` 包的 `oauth/`、`google.d.ts`、4 个登录方式的 Logo 和 2 张渐变背景图；
  propel 的 `GithubIcon`、`GitlabIcon`（基点就已无引用，`gitlab` 命中本 Task 的规则）；
  `user/account.store.ts`（第三方账户）和用户 store 的 `accounts`。
- **错误码**：`helpers/authentication.helper.tsx` 删掉已删登录方式、管理后台、实例设置、SMTP、机器人用户的错误码，
  只留"邮箱 + 密码"登录、注册和修改密码还可能返回的；`EAuthSteps` 删除。`@plane/utils` 的 `auth.ts` 里那份重复的
  错误码和 `authErrorHandler`（web 用的是自己的一份）删除，只剩密码强度的两个函数。
- **管理后台**：`InstanceNotReady` 页、`IInstance` 及管理后台的配置类型、`GOD_MODE_URL` / `ADMIN_BASE_*` 常量、
  `turbo.json` 和 `.env.example` 的 `VITE_ADMIN_*`、用户菜单的"进入管理后台"、`RESTRICTED_URLS` 的 `god-mode`。
- **实例配置**：`InstanceStore` 只剩 `config` 和 `isLoading`；`error` 字段删除（它唯一的读取方是一个无论如何都渲染
  子元素的分支）。`InstanceWrapper` 的顺序不变：加载中 → 请求失败显示 `MaintenanceView` → 渲染子元素。
  字段清单见 2.11。
- **邮件**：个人设置的"通知"标签页（它只有邮件通知偏好）、修改登录邮箱弹窗、新手引导资料步骤里的"设置密码"
  和营销邮件同意；`UserService` 的对应方法。
- **安全页**：`is_password_autoset` 分支删除，只剩"输入旧密码修改密码"。修改密码仍带 CSRF 令牌提交（3.6）。
- **成员表**：只显示登录方式的"认证"一列删除（`last_login_medium` 不再读）。
- **`AuthenticationWrapper`**：删掉 `SET_PASSWORD` 页面类型之后剩 6 处渲染时跳转（M1 设计 4.2 的数字）。
- **新增仓库内单元测试**：`navigation.test.ts` 加 2 个，断言个人设置只剩
  `general`、`security`、`preferences`、`api-tokens` 四个标签页，每个都在侧边栏的一个分组里。

### 2.4 公开发布（Task 2，M1 设计 3.8）

- 删除项目发布弹窗、视图发布、`ProjectPublishStore`、`project-publish.service.ts`、`use-project-publish.ts`、
  `types/src/publish.ts`，以及菜单、标签页导航、侧边栏项目列表里的"发布"项和"已发布"标记；只喂发布菜单项的
  `isAdmin` 随之删除。
- 删除 space 应用的地址：`SPACE_BASE_URL`、`SPACE_BASE_PATH`、`SITES_URL`、`metadata.ts` 的 `SPACE_SITE_*`、
  `getPublishViewLink`、`turbo.json` 和 `.env.example` 的 `VITE_SPACE_*`、`RESTRICTED_URLS` 的 `spaces`。
- 删除公开页的类型：公开的工作项、评论、表情、投票，`IPublishedProjectView`、`TPublishViewSettings`，
  收集箱的 `TInboxForm`、`TAnchors`，项目和视图的 `anchor`。
- **评论的可见范围**（3.8）：`EIssueCommentAccessSpecifier`、评论的 `access` 字段、编辑器工具栏的"内部 / 外部"开关、
  评论卡片上的标记。基点就已无引用、带着 `showAccessSpecifier` 的 `comments/comments.tsx` 和它的 `index.ts` 一起删。
- 草稿的"发布为工作项"不受影响（守卫不搜单独的 `publish`）。

### 2.5 Epic、团队、工作项类型、批量操作与其他空壳（Task 3，M1 设计 2.2、3.9）

M1 设计 9 节把这些放在同一个 Task（硬）：它们改的是同一组 store、hook 和布局组件的分支。

- **Epic**：Epic 弹窗、空状态、插图和图标、`types/src/epics.ts`；`isEpic` 参数（59 个文件）和 `is_epic` 字段；
  `EIssuesStoreType.EPIC`、`epicDetail`、`epicService`、`projectEpics`。
- **工作项的 service 类型整个删除**：Epic 走后，每个工作项 service 和详情 store 都用 `ISSUES` 构造，描述历史的
  service 用 `WORK_ITEMS`。`EIssueServiceType` 连同它在约 75 个文件里的传递一起删掉；描述历史和按编号查找本来用的
  `/work-items/` 地址写成字面量保留（M1 设计 2.2："不能一律改成 `issues`"），其余地址是 `/issues/`。
  `useIssueDetail()` 不再带参数。
- **团队**：`EIssuesStoreType` 的 `TEAM`、`TEAM_VIEW`、`TEAM_PROJECT_WORK_ITEMS` 及其 store 实例、路由参数
  `teamspaceId`、分组方式 `team_project`、团队的文件资源类型、插图。共享的 `ProjectIssues` 类保留。
- **工作项类型**：`type_id`、`issueTypeId`、`IssueTypeSwitcher`（它只是渲染 `IssueIdentifier`，调用方直接用
  `IssueIdentifier`）、显示属性和筛选项 `issue_type`、`useWorkItemProperties`（CE 中为空）、自定义属性的值和动态。
- **模板**：工作项弹窗上下文只留社区版会设置的 `allowedProjectIds`、`selectedParentIssue`；模板的字段和空函数删除。
  草稿的"移动到项目"保留，只删企业版的属性值调用。
- **批量操作与多选**（3.9）：批量操作条、`multiple_select.store.ts`（根 store 的两处构造）、`use-multiple-select*`、
  `use-bulk-operation-status`、service 的批量接口和 8 个 store 的 `bulkUpdateProperties`、`TBulkOperationsPayload`、
  组头和行上的选择框、逐层透传的 `selectionHelpers`、`constants/src/spreadsheet.ts`、选中行的样式。
  因此不再被读的 `canEditProperties` prop 删除。命令面板的 `BulkDeleteIssuesModal`（社区版接口）和表单的多选下拉框保留。
- **工时、重复工作项**：动态的 `WORKLOG` 分支、通知卡片的 `estimate_time`、`utils/src/datetime.ts` 里三个"分钟"工具函数、
  worklog 插图、`recurring_work_items` 文案。
- **工作流和项目更新**（企业版项目功能）：列表和看板中为空操作的 `useWorkFlowFDragNDrop`；`StateOption` 从
  `components/workflow/` 搬到 `dropdowns/state/state-option.tsx`，只留 `option`、`className`；状态下拉框只为工作流
  存在的三个 prop 删除；项目动态中 `is_{project_updates,epic,workflow,time_tracking,issue_type}_enabled` 的分支；
  项目更新的图片。
- 基点就已死、带着本 Task 词汇的 `common/activity/`（4 个）、`issues/layout-quick-actions.tsx`、power-k 的
  `work-item-selection-page.tsx`，连同只有它们在用的 `useLayoutMenuItems`、两个菜单项工厂、`types/src/activity.ts`
  和 `types/src/project/activity.ts` 一起删。
- `RESTRICTED_URLS` 去掉 `epics`、`epic`。常量包的重复枚举值警告随之消失，上限 2 → 1。

### 2.6 AI 助手、Unsplash、遥测（Task 4）

- **AI 助手**：工作项表单描述编辑器的 AI 按钮行、"I'm feeling lucky"、`gpt-assistant-popover.tsx`、
  `ai.service.ts`、`types/src/ai.ts`、propel 的 `AiIcon`、实例字段 `has_llm_configured`、标签页序号 `feeling_lucky`；
  表单里只为 AI 存在的 prop 和状态。编辑器包里的 AI 已在 P2 删除。
- **Plane AI（pi-chat）**：第一次按"AI"扫描漏掉了它。它只剩一个入口：侧边栏的 `user-menu.tsx`（knip 报告的未使用
  文件）。这条死链（`user-menu.tsx` → `user-menu-item.tsx` → `notification-app-sidebar-option.tsx`）、propel 的
  `PiChatLogo`、`isAiPath` 和文案一起删。
- **Unsplash**：图片选择器的 Unsplash 标签页，以及它**每次打开都发出的** `/api/unsplash/` 请求；标签页变成常量
  `[images, upload]`；只喂 Unsplash 搜索的 `control` prop 连同组件的泛型删除（3 个调用方）；封面帮助函数的
  `"unsplash"` 类型；`FileService.getUnsplashImages`；实例字段 `has_unsplash_configured`。
- **遥测**：应用里已经没有遥测 SDK，只剩实例的 `is_telemetry_enabled`，已随 Task 1 的 `IInstance` 删除。
  本 Task 只加守卫规则。
- `symref.mjs` 找出的孤儿：`IProjectLite`、`IWorkspaceLite`（只有 `IGptResponse` 用）。

### 2.7 计费和升级提示（Task 5）

- 删除计费设置页（路由、页头、页面）、`components/license/`（10 个）、`workspace/billing/`（9 个）、
  `edition-badge.tsx`、`upgrade-badge.tsx`、基点就已死的 `icons/locked-component.tsx`、
  `constants/src/{payment,subscription}.ts`、`types/src/payment.ts`、`utils/src/subscription.ts`、
  propel 的 `upgrade-icon`、2 张 scribble 图片。
- 工作区设置去掉 `billing-and-plans`（常量、分组、`TWorkspaceSettingsTabs`、图标）；侧边栏底部只放版本徽标
  （外加一段注释掉的帮助菜单）的那一栏删除；帮助菜单的"联系销售"；项目功能列表的 `isPro`（全部为 `false`）、
  升级徽标和提示；用户资料的 `billing_address*` 三个字段；基点就已死的 `MARKETING_CONTACT_US`；
  `RESTRICTED_URLS` 的 `plane-pro`、`plane-ultimate`、`enterprise`、`plane-enterprise`、`upgrade`、`billing`。
- 注销账户的说明不再说"会为工作区计费"（中英文同改）；守卫的 `billing` 规则因此也搜 `billed`。
- 计费文案消失后，登记在它上面的 21 条例外全部过期、删除（P2 留下的 8 条 `until: M1/P3`，P3 自己的 13 条
  `until: M1/P4`）；顶层 `phase` 改为 `M1/P3`（2.2 第 7 条）。
- **新增仓库内单元测试**：`navigation.test.ts` 加 2 个，断言工作区设置只剩 `general`、`members`、`webhooks`，
  每个都在侧边栏的一个分组里。

### 2.8 企业版扩展点；项目"邀请"改为添加成员（Task 6）

社区版代码里有很多给企业版插入用的接缝：在 CE 中为空、恒定或只做转发的 hook、prop、类型和注册表。
它们删掉，原来的作用直接写在用到的地方：

- **路由**：`routes/extended.ts`、`routes/helper.ts`（`mergeRoutes`），以及 `routes/redirects/` 下三个没人用的
  `index.ts`（`core.ts` 自己列出了重定向条目）；`routes.ts` 直接用 `coreRoutes`。
- **编辑器包**：`flaggedExtensions`（从来不读）和 `extendedEditorProps`（类型是 `unknown`）从每个 prop 列表、
  `Pick` / `Omit` 和测试里删除；`editor-extended.ts`、`types/utils.ts`、`extensions/core/`、
  `rich-text-extensions.tsx`（注册表只注册斜杠命令）、`additional-slash-command-options.tsx` 删除。
  富文本编辑器写成"没禁用就装斜杠命令"；斜杠菜单里的图片项直接写在代码块之后、没禁用图片时才出现。
  `CoreEditorRefApi` → `EditorRefApi`，`TEditorImageAsset` → `TEditorAsset`，
  `TCoreCustomComponentsMetaData` → `TCustomComponentsMetaData`，`TExtendedFileHandler` 并入。
  **新增测试** `slash-commands/command-items-list.test.ts`（2 个）：图片项紧跟代码块；禁用图片时没有图片项。
- **web 的编辑器 hook**：`use-editor-flagging.ts`、`use-extended-editor-config.ts`、`use-additional-editor-mention.tsx`
  （@提及只搜用户：`query_type: ["user_mention"]`）。
- **菜单和 store**：菜单 hook 直接返回 `TContextMenuItem[]`（`MenuResult` 和恒为 `null` 的 `modals`、
  `useIntakeHeaderMenuItems` 删除）；只为企业版子类存在的 `Base*` 类加别名改回本名，文件 `git mv`：
  `command-palette.store.ts`、`power-k.store.ts`、`member/project/project-member.store.ts`、
  `user/permissions.store.ts`；`CoreRootStore` → `RootStore`，`BaseWorkspaceRootStore` → `WorkspaceRootStore`。
- **"additional"的另一半**：通知内容的兜底映射并进通知卡片（映射和类型不再导出）、`use-additional-favorite-item-details`、
  `AdditionalFilterValueInput`（并入）、`_getAdditionalOperatorOptions`、`additionalNavigationItems`、
  `additionalRender`、`mutateWorkspaceMembersActivity`、`use-workspace-issue-properties-extended`、
  `GlobalModals` 的 prop（工作区布局不再把路由 props 传给它）。
- **富文本筛选**：`operators`、`derived`、`field-types`、`operator-configs` 四组类型的 `core.ts` 与空的 `extended.ts`
  合并进各自的 `index.ts`，用最终的名字；常量的 `operator-labels` 同样合并；`CORE_OPERATORS` → `OPERATORS`。
  "显示用"的否定运算符那一层（`TAllAvailable*ForDisplay` 等）不在本 Task（交给 M4，第 7 节）。
- **收藏**（P2 评审交给 P3 的产品判断）：`TFavoriteEntityType = "project" | "view" | "cycle" | "module" | "folder"`；
  图标映射按它定类型，去掉兜底的 `PagesOutline`；`switch` 覆盖全部取值；新建文件夹写 `entity_type: "folder"`。
- **项目"邀请"**（M1 设计 2.2）：web 里没有按邮件邀请进项目的流程，弹窗本来就是从工作区成员中直接添加。
  `send-project-invitation-modal.tsx` → `add-project-members-modal.tsx`（`AddProjectMembersModal`），
  键 `project_settings.members.invite_members` → `add_members`（标题"Add members / 添加成员"），
  工作区下拉菜单改用新的 `workspace_settings.settings.members.invite_members`（工作区邀请保留，3.15）；
  离开项目、私有项目的文案不再说"受邀"；无引用的 `common.accessible_only_by_invite` 删除。

### 2.9 Plane 自身的死代码；knip 改为门禁（Task 7，M1 设计 2.2、7.2）

- **knip 报告的 64 个未使用文件全部删除。** `upstream.py` 在 Plane 上游逐个查过：每个文件在上游也只被别的死文件
  导入。连同只有它们在用的：`types/src/base-layouts/`、6 个键、5 张图片。
- **96 个未使用的导出、44 个未使用的类型**：文件内还在用的去掉 `export`，没人用的连声明删除；删到不再导出任何东西的
  文件连同桶文件里的那一行删除；删除在文件内留下的未使用局部代码随之删除。反复到 knip 为零（原型 4 轮）。
  `issue-detail/links/root.tsx` 只剩详情小组件在用的 `TLinkOperations` 类型，改名为 `links/types.ts`。
- **类型包的 React 类型**：`@plane/types` 一直靠全局命名空间写 React 类型、不带 DOM lib 写 DOM 名字，只因为被删的
  `base-layouts` 导入过 `react` 才能编译。现在每个文件导入自己用到的 React 类型，包改用 React 库的 tsconfig。
- **P2 评审第 7 节的清单**：
  - 从未注册的 `NodeHighlightPlugin`：编辑器容器不再给它发事务，`plugins/highlight.ts` 删除，只剩一个子元素的
    fragment 收掉；
  - 表格菜单的"适应宽度"读的是一个没人定义的 CSS 变量，对保留的编辑器没有作用，删除；
  - 链接扩展不再把 `http`、`https` 注册为 linkify 的自定义协议（linkify 本来就支持，"already initialized"的噪声消失）；
  - 工作区设置里空的 FEATURES 分组；`RESTRICTED_URLS` 中重复的 `config`、`mobile`、`monitor`（`silo` 随集成删除）；
  - 没人用的包导出 `ISSUE_DISPLAY_FILTERS_BY_LAYOUT`、`generateDateArray`、`TCycleProgress`、`IWorkspaceProgressResponse`、
    `UserActivityIcon`；`findTotalDaysInRange` 改为内部函数；
  - 集成、导入、从 CSV 导入成员、自动提醒的文案，`jira.svg` 和 8 张集成空状态图。
- **Plane 不路由、也没人调用的 service 方法**：`IssueService` 的 `deleteIssueRelation`、
  `get/updateIssueDisplayProperties`、`bulkSubscribeIssues`，`ProjectStateService.updateState`（PUT；store 用的是
  `patchState`），`UserService.userIssues`，`ViewService.getViewIssues`，以及 `app_config.service.ts`（knip 文件）。
  评论框总有项目，`CommentCreate` 的 `projectId` 改为必填，工作区级的上传分支（地址 Plane 不路由）和
  `FileService.updateBulkWorkspaceAssetsUploadStatus` 删除。
- **新增仓库内单元测试**：`navigation.test.ts` 加 2 个：工作区设置没有空分组；保留地址名单没有重复。
- **knip 门禁**：`make knip` 先运行 `pnpm --filter web exec react-router typegen`，再运行
  `pnpm exec knip --treat-config-hints-as-errors`（去掉 `--no-exit-code`）；`knip.jsonc` 删掉 web 的
  `ignoreUnresolved`；持续集成的步骤名去掉"report only"；README 同步。
- `docs/v0/frontend-changes.md` 第二节：P3 的 10 行改为"已完成 / M1/P3"。

### 2.10 关键词守卫

P2 的 24 条规则之上新增 18 条，全部 `"phase": "M1/P3"`，在删完对应功能的同一个提交里加入（M1 设计 7.4）。
每条规则的每个顶层分支（以及分支里 `(?:a|b)` 的每一支）都有命中样本（`alts.mjs M1/P3` 输出为空）；
在本 Phase 开始时已经没有命中的设计词汇（`PlaneAi`、`aiHandler`、`AI_`、`rephrase`、`ProIcon`、`ExtendedBasePage`）
不设规则，命中样本取不到真实代码的（`github-repository`、应用安装）取自 P2 删除之前的代码。

| 规则 | 要点 | 大小写 | Task |
|---|---|---|---|
| `admin` | `god[-_ ]?mode`、`admin[-_]base`、`instance-admin`、`InstanceNotReady`、`is_setup_done` | 不敏感 | 1 |
| `third-party-login` | `oauth`、整词的 `sso`、`ldap`、`saml`、`oidc` | 不敏感 | 1 |
| `auth-methods` | 验证码、`magic_*`、找回 / 重置密码、`set-password`、`SET_PASSWORD`、`setPassword`、`email-check`、`gitea`、`gitlab`、登录方式和 SMTP 的实例开关、`is_password_autoset`、`EAuthSteps`、`last_login_medium`、`LOGIN_MEDIUM`（不搜 `set_password`：注册表单的"设置密码"标签保留） | 敏感 | 1 |
| `email-sending` | `email_notification`、`marketing_email`、`EmailNotification`、`change_email`、`ChangeEmail`、`notification-preferences`、`email/generate-code` | 敏感 | 1 |
| `publish` | space 的地址常量、`deploy-boards`、发布的组件和 store、公开页的类型、`TInboxForm`、`TAnchors`、`AccessSpecifier`、`comments.switch` | 敏感 | 2 |
| `publish-anchor` | `.anchor`、`anchor:`（只作用于 `web/apps/web` 和 `web/packages/types`） | 敏感 | 2 |
| `epics` | `(?<![dD])epic`、`Epic`、`EPIC` | 敏感 | 3 |
| `teamspaces` | `teamspace`、`team_view`、`team_project`、`team_space`、`EIssuesStoreType.TEAM`（不搜单词 team） | 不敏感 | 3 |
| `work-item-types` | `type_id`、`issueTypeId`、`IssueTypeSwitcher`、`work_item_type`、`issue_type`、`useWorkItemProperties` 等 | 敏感 | 3 |
| `bulk-operations` | `bulk[-_]?operation`、`multipleselect`、`selectionhelper`、`selected-issue-row`、`spreadsheet_select_group` | 不敏感 | 3 |
| `templates-worklogs` | 模板字段、`WORKLOG`、`worklog`、`time_tracking`、`recurring_work_items` 等 | 敏感 | 3 |
| `workflows-project-updates` | `WorkFlow`、`isWorkflow`、`components/workflow`、`is_workflow_enabled`、`project_updates`、`EUpdateStatus` 等（不搜单独的 workflow：保留地址名单里的两个词留给 M3） | 敏感 | 3 |
| `ai-assistant` | `gpt`、`llm`、`feeling lucky`、`AIService`、`AiIcon`、`pi-chat`、`PiChat`、`plane-intelligence`、`isAiPath` 等 | 敏感（`llm` 不区分大小写会命中 `scrollMode`、`allMembers`） | 4 |
| `unsplash-telemetry` | `unsplash`、`posthog`、`telemetry`、`sentry`、`intercom` | 不敏感 | 4 |
| `billing` | `upgrade`、`bill(?:ing\|ed)`、`EProductSubscription`、`contact_sales`、付费套餐的地址、`LockedComponent`、版本徽标、`payment`、`trial`、`add_seats`、`talk_to_sales`、`marketing_(?:contact\|plane)`（保留的职业选项 `marketing_or_growth` 不算） | 不敏感 | 5 |
| `enterprise-shells` | 企业版扩展点的 39 个确切符号，以及整词的 `CE`、`EE`（不用 `TExtended`、`TAdditional` 前缀：它们命中保留的扩展侧边栏和筛选组件） | 敏感 | 6 |
| `project-invitations` | `ProjectInvitation`、`project.*invit(?:e\|ation)` | 不敏感 | 6 |
| `integrations` | `integration`、`importer`、`jira`、`slack`、`github-repository`、`app[-_]?installation`、`indexeddb`、`members_import`、`"silo"`（不搜单独的 github：保留的"在 GitHub 上加星"） | 不敏感 | 7 |

例外从 14 条变为 4 条：

| 规则 | 文件 | match | count | until | 来历 |
|---|---|---|---|---|---|
| `analytics` | `web/packages/utils/src/tlds.ts` | `analytics` | — | M9 | P2，顶级域名数据 |
| `analytics` | `web/apps/web/core/services/cycle.service.ts` | `analytics` | — | M6 | P2，Plane 的进度接口地址 |
| `pages-collaboration` | `web/packages/utils/src/tlds.ts` | `wiki` | — | M9 | P2，顶级域名数据 |
| `project-invitations` | `web/apps/web/core/services/user.service.ts` | `projects/invitation` | 1 | M3 | Task 6，`joinProject` 调用的 Plane 地址 `/projects/invitations/`（M1 设计 7.4） |

本 Phase 登记过、又在本 Phase 消失的例外：Task 1 在计费文案上登记 8 条、Task 3 登记 5 条（都是 `until: M1/P4`），
Task 5 删掉计费文案时连同 P2 的 8 条一起删除；P2 的另外 3 条 `until: M1/P3`（`publish.ts` 的 `gantt`、`epics.ts`
的 `Analytics`、通知卡片的 `estimat`）分别由 Task 2、Task 3 删掉文件或字段时删除。

### 2.11 实例配置字段：删除 / 保留 / 等待重新定义（M1 设计 3.7，交给 M2）

`IInstanceInfo` 只剩 `config`；`IInstanceConfig` 从 16 个字段变为 4 个。

| 字段 | 处理 | 读取方 | Task |
|---|---|---|---|
| `instance`（整个 `IInstance`：版本、许可证、`is_telemetry_enabled`、`is_activated`、`is_setup_done`、`workspaces_exist` 等 21 个字段） | 删除 | "实例未完成设置"页、实例 store | 1 |
| `is_google_enabled`、`is_github_enabled`、`is_gitlab_enabled`、`is_gitea_enabled` | 删除 | 第三方登录 | 1 |
| `is_magic_login_enabled`、`is_email_password_enabled`、`is_smtp_configured` | 删除 | 登录方式的选择、找回密码 | 1 |
| `app_base_url`、`admin_base_url` | 删除 | 无（管理后台的地址） | 1 |
| `space_base_url` | 删除 | 公开发布 | 2 |
| `has_llm_configured` | 删除 | AI 按钮 | 4 |
| `has_unsplash_configured` | 删除 | Unsplash 标签页 | 4 |
| `enable_signup` | 保留 | 登录页头部的"注册"链接 | — |
| `is_workspace_creation_disabled` | 保留 | 创建工作区页、新手引导、工作区菜单、命令面板 | — |
| `file_size_limit` | 保留，等待重新定义（名称和策略由 M2 或 M5 定） | `use-file-size` | — |
| `is_self_managed` | 保留，**等待重新定义** | 新手引导：为 `true` 时跳过"角色""用途"两步 | — |

`is_self_managed` 的去留由 M2 决定：Nerve 只有自托管一种形态，这个字段恒为真时，应当连同新手引导的两个步骤一起
定下来，而不是在 M1 按恒真化简（那是产品决定，不是死代码）。管理后台的配置类型（`IInstanceAdmin`、
`IInstanceConfiguration`、`TInstance*ConfigurationKeys`）一并删除：实例配置改由配置文件和命令行管理。

### 2.12 文案、图片与包导出的口径

- **文案**：用 `keyref.mjs`（严格方法）。每个 Task 删 `orphaned <Task 的基点>` 列出的键，再删以被删功能命名、
  基点就已无引用的键（P2 评审裁定 3）。中英文成对删除，`check:sync` 是门禁；整个命名空间删除时同时删
  `namespaces.ts` 的条目（P3 没有整删的命名空间）。基线就无引用、不指向任何被删功能的键（结束时 570 个）留给收尾。
- **图片**：`assets.mjs` 列出文件名不再被任何源码提到的图片，`assets-orphaned.py` 对比 Task 的基点找出本 Task
  造成的；以被删功能命名的随功能删。P3 删 29 张图片；结束时仍有 142 张无引用，留给收尾。
- **包导出**：knip 不报工作区包的导出（2.2 第 2 条）。每个 Task 用 `symref.mjs orphaned` 删掉自己造成的包级孤儿。
  基线就没人用的包导出不在 knip 的门禁范围内，P3 只删 P2 评审点名的 5 个；`symref.mjs unused web/packages/`
  结束时还列出 355 个（utils 97、types 95、propel 73、constants 55、ui 17、editor 13、api-client 3、
  shared-state 1、hooks 1），交给收尾逐个复核（第 3 节第 9 条）。

### 2.13 保留行为的核对（M1 设计 7.5）

| 7.5 的行 | 手段 | 落点 |
|---|---|---|
| P3 认证、实例 | 临时核对脚本（控制者在 Task 之后写），同时拦截 `**/auth/**` 和 `**/api/**`，脚本没有列出的请求一律算失败。核对：登录、注册表单的邮箱可以编辑；提交的字段、`csrfmiddlewaretoken`、`next_path`；成功后的跳转；失败后回到 `/` 并显示错误（第 3 节第 2 条）；关闭注册时页头没有"注册"链接；实例请求失败时显示维护页；修改密码带 CSRF 令牌提交、没有被"设置密码"的清理误伤；个人设置只剩 4 个标签页 | review 附录 |
| P2、P3 动态、通知、工作项列表 | 重跑 P2 的临时核对脚本 3，数据里去掉 Epic、工时、类型类的动态；另外核对：列表、看板、表格没有选择框；工作项详情的描述历史用 `/work-items/` 地址、按编号查找用 `/work-items/`；@提及只请求用户；收藏的各类实体有图标；图片选择器打开时不再请求 `/api/unsplash/` | review 附录 |
| P3 保留的入口 | **进仓库**的 `navigation.test.ts`（个人设置 4 个标签页、工作区设置 3 个标签页、没有空分组、保留地址不重复）和 `command-items-list.test.ts`（图片斜杠命令的位置） | Task 1、5、6、7 |

临时核对脚本不进仓库（M1 设计 7.5）：全文、假数据、运行命令和断言写进 P3 review 的附录。

---

## 3. 与上级设计的差异和补充

每一条都已按 M1 设计"能自己定的就自己定"的原则决定，这里列出决定和理由，供控制者复核。

1. **第 6 步拆成两个 Task，并多放了三件事。** 9 节 P3 的第 6 步是"死代码、接线、图片资源、knip 门禁"。
   决定：
   - Task 6 做"接线"，即企业版扩展点（2.2 表里"`extended` 空壳"一行），外加项目"邀请"（2.2 的表里有，9 节的
     顺序里没有）。它们改名、合并类型，改的是编辑器包、根 store、菜单这些共享文件，和死代码分开，评审时能看清
     每一处"原来的作用写到了哪里"。
   - Task 7 做死代码、P2 评审的清单、图片、knip 门禁。knip 门禁必须在最后：在它之前的任何 Task 都可能造出新的孤儿。
   - 工作流和项目更新（企业版项目功能）放进 Task 3：它们改的是同一批布局文件，`work_item_type` 的文案也在
     `project_settings.workflows` 下面。

2. **注册失败后显示在登录页（Task 1，行为变化）。** Plane 的认证接口把所有错误都重定向到 `/`（基地址加
   `error_code`）。基点的 `AuthRoot` 收到 `AUTHENTICATION_FAILED_SIGN_UP` 时把 `/` 切成注册模式。模式改由路由决定后
   （3.6），注册失败显示为登录表单上方的错误提示，用户要再点"注册"。决定：不为此加"按错误码切换模式"的代码，
   因为 M2 用令牌管理器重写登录时错误就地显示、不再重定向（交接 M2）。临时核对脚本要断言这一行为。

3. **工作项的 service 类型整个删除（Task 3）。** 设计 2.2 说"删除多态时，描述历史和按编号查找使用的 `work-items`
   地址保留"。实测 Epic 走后 `EIssueServiceType` 只剩两个取值、每处构造都是常量：工作项 service 一律 `ISSUES`，
   描述历史 service 一律 `WORK_ITEMS`。留下一个只有常量实参的参数就是空壳，所以连枚举一起删，`/work-items/`
   写成字面量。`endpoints.py` 核对过删除之后每个地址 Plane 都路由。

4. **基点就已死、但带着本 Task 词汇的代码，由本 Task 删（Task 1–5）。** 设计把"knip 报告的未使用文件"整体放在
   第 6 步。决定：这类文件如果带着某个 Task 的词汇（2.2 第 4 条的 8 处），就在那个 Task 删，因为不删它，
   那个 Task 加的守卫规则就会命中它；为了躲命中而登记例外，等于给要删的东西开豁免。这是 P2 评审裁定 3
   （以被删功能命名的文案和图片随功能删）在代码上的对应。

5. **`useEditorFlagging` 之外的扩展点也在 P3 删（Task 6）。** P2 评审只点了 `useEditorFlagging`。实测同类的扩展点有
   几十个（2.8），`enterprise-shells` 规则按确切符号列出 39 个，全部删完才能加规则。

6. **收藏的实体类型改为联合类型（Task 6，P2 评审交给 P3 的产品判断）。** 取值是 v0 保留的五种：项目、视图、
   迭代、模块、文件夹。去掉兜底图标后，编译器保证每种都有图标；新类型的收藏必须先改这个联合类型。
   后端的取值由 M7 对齐。

7. **表格菜单的"适应宽度"删除（Task 7，P2 评审交给 P3 的产品判断）。** 它读的 CSS 变量在任何地方都没有定义，
   点了没有效果；保留一个无效的菜单项不是保留功能。

8. **`is_self_managed` 保留到 M2（Task 1）。** 见 2.11。它控制新手引导的步骤，是产品决定。

9. **包级的未使用导出只删点名的和本 Phase 造成的。** knip 的门禁不覆盖工作区包的导出（2.2 第 2 条），
   M1 设计 7.2 的"knip 清零"在这个口径下已经做到。基线就没人用的 355 个包导出，每个都要判断是"API 的一部分"
   还是死代码，工作量和收尾的死文案、死图片同类。决定：交给 M1 收尾，与 570 个死键、142 张死图片一起处理，
   方法是 `symref.mjs unused`。**如果控制者认为它们属于 7.2 的"清零"，需要在 P3 加一个 Task**（原型没有做）。

10. **类型包改用 React 库的 tsconfig（Task 7）。** 见 2.9。这不是新依赖：`react-library.json` 是仓库里已有的共享配置，
    ui、propel 已在用。

11. **富文本筛选的"显示用"那一层留给 M4（Task 6）。** `core` / `extended` 合并之后，`TAllAvailable*ForDisplay`、
    否定运算符的标签等仍是 Plane 为 UI 做的一层映射。它不是企业版接缝，而是筛选的设计本身；M4 重做工作项筛选时
    一起决定。

12. **四个 store 文件改名（Task 6）。** `Base*` 加别名的写法只为企业版子类存在。改回本名时文件一起 `git mv`，
    文件名和类名一致；导入这些文件的地方同一个提交改完。

13. **`links/root.tsx` 改名为 `links/types.ts`（Task 7）。** 删掉没人用的 `IssueLinkRoot` 组件后，文件只剩一个类型，
    叫 `root.tsx` 会误导。

14. **不删依赖。** 原型的每个 Task 结束时 knip 都没有报告未使用的依赖或开发依赖，所以 P3 没有 `lock-diff.mjs` 这一步。
    实施时如果 knip 报出未使用的依赖，按 P2 的做法删（`pkg.mjs`、catalog、`lock-diff.mjs`），并在报告中说明。

15. **CSRF 不动。** 登录、注册、修改密码、退出仍带 CSRF 令牌（3.6）。`InstanceService.requestCSRFToken`
    删除，因为它是 `AuthService.requestCSRFToken` 的一份没人调用的副本，不是 CSRF 本身。

---

## 4. 验收标准

- [ ] 7 个 Task 各一个提交，每个提交结束时 `pnpm exec turbo run check:types` 23 个任务通过。
- [ ] `make lint-web` 通过：`keywords: 42 rules, 4 exceptions, no hits.` + 52 个 turbo 任务；
      每个包的 oxlint 警告数**等于**上限（合计 713）；`node $P3TMP/alts.mjs M1/P3` 没有输出。
- [ ] `make test-web` 15 个任务通过（constants 11 个测试、editor 16 个）；`make build-web` 11 个任务通过。
- [ ] `make knip` 以门禁形式通过：先生成路由类型，不带 `--no-exit-code`，带 `--treat-config-hints-as-errors`；
      `knip.jsonc` 里没有 web 的 `ignoreUnresolved`；持续集成的 `web` 任务执行它。
- [ ] 关键词守卫：本 Phase 的 18 条规则没有未登记的命中；4 条例外全部精确到符号，`until` 指向 M3、M6 或 M9，
      M9 的理由写明为什么在 v0 内不可能消失。
- [ ] 中英文键集合相同（`check:sync`）；`keyref.mjs orphaned 6d9692b` 列出的键都已删除；以被删功能命名的键没有剩下。
- [ ] `keyref.mjs orphaned 6d9692b` 输出 0 个键；`assets-orphaned.py` 对 `6d9692b` 没有输出；
      `symref.mjs orphaned 6d9692b` 只列出 `TIssueSearchResponse`（它唯一的外部使用方随 Task 3 删除，但它是导出的
      `TSearchResponse` 的组成部分，与同级的 `TProjectSearchResponse` 等一样保留导出）；
      `headers.sh 6d9692b` 没有输出；`git diff 6d9692b | grep -F '${"'` 为空。
- [ ] `endpoints.py` 只剩第 7 节列出的 4 处。
- [ ] M1 设计 7.5 中 P3 的几行核对过：进仓库的测试通过，临时脚本写进 review 附录。
- [ ] S1–S4 通过；持续集成的 `server`、`web`、`e2e` 三个任务通过。
- [ ] `docs/v0/frontend-changes.md` 第二节同步；M1 设计 12 节 P3 一行更新。

---

## 5. 不在 P3 范围内

- CSRF 和会话提交方式（M2，3.6）；认证错误就地显示（M2）；修改登录邮箱的最终形态（M2，3.13）。
- Next.js 兼容层、`routes/core.ts` 末尾的 10 条旧地址重定向、`AuthenticationWrapper` 的渲染时跳转（P4）。
- 品牌和包名（P5），包括 63 个文件里 `// plane web …` 这类导入分组注释（它们是 CE / EE 拆分留下的名字）。
- 基线就无引用的 570 个键、142 张图片、355 个包导出（M1 收尾）。
- oxlint 警告清零（M2–M8，M1 设计 7.3）。
- 任何后端改动：被删功能对应的 Plane 接口字段、地址和取值按第 7 节交接。

---

## 6. 风险

| 风险 | 应对 |
|---|---|
| 删认证页面时误伤保留的修改密码、退出、注册开关 | 7.5 的临时核对脚本同时拦截 `**/auth/**` 和 `**/api/**`，未列出的请求算失败；个人设置标签页有仓库内测试 |
| Task 3 一次改 266 个文件，工作项 store 的分支、地址可能选错 | service 类型按"每处构造都是常量"化简后，`endpoints.py` 核对地址；描述历史和按编号查找的 `/work-items/` 在 7.5 的脚本里断言；每个 Task 结束时与原型提交对比 |
| 恒为假的化简留下 `${"x"}` 之类的残渣或错的分支 | 提交前 `git diff HEAD \| grep -F '${"'` 为空；lintdiff 列出新增的未使用变量 |
| 删除工具误删文件头、注释或别处的同名代码 | 工具只对 knip、lintdiff 点名的文件和符号动手；`headers.sh` 核对许可证头；每轮之后跑类型检查 |
| knip 改为门禁后，后续 Phase 因为生成文件的状态不同而时好时坏 | `make knip` 先运行 typegen；`--treat-config-hints-as-errors` 让过时的配置也失败 |
| 把每个实例自己的状态变成共享的（P2 的 `UniqueID` 回归） | 2.2 第 9 条逐个检查过；plan 要求评审对每个新增的模块级常量和改动过的扩展创建说明它是否只读 |

---

## 7. 移交事项

### 7.1 交给后续 M（写进对应 M 的 `handoffs/`）

| M | 事项 |
|---|---|
| M2 | - 认证错误就地显示：现在注册失败回到 `/` 显示在登录表单上方（第 3 节第 2 条）；<br>- CSRF 和会话：登录、注册、修改密码、退出仍用 `csrfmiddlewaretoken` / `X-CSRFTOKEN`，M2 必须删掉（3.6）；<br>- 实例配置字段清单（2.11），其中 `is_self_managed` 和新手引导的两个步骤、`enable_signup` 的名称要 M2 定；<br>- 用户字段不再读：`is_password_autoset`、`last_login_medium`、`has_marketing_email_consent`、`billing_address_country`、`billing_address`、`has_billing_address`、邮件通知偏好的 5 个开关、第三方账户（`provider`、`provider_account_id`）；<br>- 不再调用的地址：`/auth/email-check/`、`/auth/magic-*`、`/auth/forgot-password/`、`/auth/reset-password/`、`/auth/set-password/`、第三方登录、`/api/users/me/notification-preferences/`、`/api/users/me/email/generate-code/` 和修改邮箱、`/api/users/me/instance-admin/`（`is_instance_admin`）；<br>- 修改登录邮箱的最终形态（3.13，决策点）；<br>- `packages/services` 的 API 令牌 `retrieve`、`destroy` 地址没有结尾斜杠（靠 `APPEND_SLASH` 重定向），M2 重新定义令牌接口时一并处理 |
| M3 | - `IProject.anchor`、视图的 `anchor` 和发布设置、收集箱的 `anchors` / `is_form_enabled` 不再读；<br>- 项目动态不再显示 `is_project_updates_enabled`、`is_epic_enabled`、`is_workflow_enabled`、`is_time_tracking_enabled`、`is_issue_type_enabled` 的变化；<br>- 项目成员只有"从工作区成员中添加"一种方式；`joinProject` 仍调用 `/projects/invitations/`（守卫例外 `until: M3`），M3 定义项目成员接口时替换；<br>- `checkProjectIdentifierAvailability` 的地址没有结尾斜杠；<br>- `RESTRICTED_URLS` 本 Phase 去掉了 `god-mode`、`spaces`、`epics`、`epic`、`plane-pro`、`plane-ultimate`、`enterprise`、`plane-enterprise`、`upgrade`、`billing`、`silo` 和重复的 `config`、`mobile`、`monitor`；剩下的产品词（`one`、`business`、`pro`、`license(s)`、`initiatives`、`workflow(s)` 等）由 M3 与后端的保留名单一起定；<br>- 工作区邀请的两条路径（3.15）不变 |
| M4 | - 工作项不再读 `is_epic`、`type_id`，显示属性和筛选项没有 `issue_type`，分组方式没有 `team_project`；<br>- 动态不再有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`；<br>- 评论不再读、不再写 `access`（3.8）；公开页的工作项、评论、表情、投票类型已删除；<br>- 描述历史和按编号查找仍用 `/work-items/` 地址，其余工作项地址是 `/issues/`；<br>- 删除了 Plane 不路由的 `deleteIssueRelation`（`IssueService` 上的那一个；关联仍由 `IssueRelationService` 删除）、`get/updateIssueDisplayProperties`、`bulkSubscribeIssues`、`userIssues`、`getViewIssues`、状态的 `PUT` 更新；<br>- 没有批量操作接口，命令面板的批量删除仍调用 Plane 社区版的接口；<br>- 富文本筛选的"显示用"一层（`TAllAvailable*ForDisplay`、否定运算符标签）由 M4 重做筛选时决定 |
| M5 | - 评论的附件总是项目级上传，工作区级的批量上传状态地址不再调用；<br>- 团队等企业版的文件资源类型已删除；<br>- 封面图不再有 `unsplash` 类型，不再请求 `/api/unsplash/`；<br>- `file_size_limit` 的名称和策略（3.7） |
| M6 | - 迭代进度的地址 `/cycles/{id}/analytics?type=` 没有结尾斜杠，守卫例外 `until: M6` 随进度接口的替换一起消失；<br>- `TCycleProgress`、`IWorkspaceProgressResponse` 已删除，新接口不需要提供它们 |
| M7 | - 收藏的 `entity_type` 只有 `project`、`view`、`cycle`、`module`、`folder`，后端的取值与之一致；<br>- 通知卡片不再显示工时（`estimate_time`）；通知内容的兜底映射已并入卡片 |

### 7.2 交给 P4

- `AuthenticationWrapper` 的 6 处渲染时跳转（Task 1 之后的数字，M1 设计 4.2）。
- 删除 `routes/redirects/` 下没人用的三个 `index.ts` 之后，`routes/core.ts` 末尾的 10 条重定向不变，仍由 P4 处理。

### 7.3 交给 P5

- 63 个文件里的 `// plane web imports`、`// plane web components` 等导入分组注释：它们是 Plane 拆分 CE / EE 时的
  目录名，现在指向不存在的层；P5 处理 Plane 名称时一并改写或删除。

### 7.4 交给 M1 收尾

- 死资源：570 个无引用的键（严格方法）、142 张无引用的图片、355 个没人用的包导出（`symref.mjs unused web/packages/`）。
- 重新核对 2 条 `M9` 例外（`tlds.ts` 的 `analytics`、`wiki`）。
- 构建体积对比（2.2 的表）写进收尾 review。
