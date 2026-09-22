# M1 前端瘦身：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M1 前端瘦身（`docs/v0/M1-frontend-trim`） |
| 日期 | 2026-09-22 |
| 状态 | 已确认（控制者按 2026-09-22 的授权自行推进）。第 3 节的裁定都在总体设计已定的范围内，都可以在以后改回；其中 3.6（CSRF 移到 M2）、3.10（保留 `@makeplane/propel`）、3.13（改邮箱）三项向项目负责人报备 |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.2、7、8、9 节；[前端改动清单](../frontend-changes.md)；[差异清单](../plane-diff.md) |
| 前置交接 | [M0-P5-frontend-trim-notes](handoffs/M0-P5-frontend-trim-notes.md)、[M0-P6-knip-notes](handoffs/M0-P6-knip-notes.md)（逐条落到第 9 节的 Phase，见 9.7） |
| 设计评审 | 写完后做了一次独立评审（opus），发现 3 条 Critical、6 条 Important、4 条 Minor，已全部改进本文件：死代码的删除提前到 P1，保证每个 Phase 都能单独通过；按 Phase 划分关键词验收；补上遗漏的交接事项和裁定 |

---

## 0. 目标与范围

### 0.1 目标
把 M0 原样迁入的 Plane 前端删到只剩 v0 要的功能，并清掉迁入时带进来的兼容层和遗留物。M1 结束时：
1. 总体设计 1.2 列出的功能，以及 Plane 企业版的残留和 Plane 自身的死代码，从路由到依赖的每一层都已删除。
2. Next.js 兼容垫片已经删除，路由全部是 React Router 的原生写法。
3. 界面上看不到 Plane 的名称和 Logo；内部包名改为 `@nerve/*`；只剩 `zh-CN` 和 `en` 两种语言。
4. 类型检查通过，knip 为零并成为门禁，oxlint 不超过新基线，关键词搜索没有结果，能正常构建。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 按第 2 节删除功能（每个功能删到 2.3 列出的每一层） | 对接新接口：services、stores 改用生成的客户端（M2 起按领域推进） |
| Next.js 兼容垫片及其配套（`process.env` 注入、dotenv、结尾 `/`） | 认证的传输方式：CSRF 和会话 Cookie 换成 Bearer 令牌（M2，见 3.6） |
| 品牌替换、包名改为 `@nerve/*` | `@nerve/services` 整包删除（M5；M1 只删其中没人用的部分，见 3.5） |
| 多语言只留中英文 | 后端和数据库的任何改动 |
| 部署遗留、工具链遗留（两份 M0 交接中的事项） | 新的端到端故事（M1 不新增故事，S1–S4 保持通过，见第 10 节） |
| knip 改为门禁；lint 上限自动核对；oxlint 清零计划 | 清零 oxlint 警告本身（按 7.3 的计划在 M2–M8 进行） |

---

## 1. 现状基线（2026-09-22 实测，`main` 为 `0e43771`）

| 项 | 数值 |
|---|---|
| `web/` 中的跟踪文件 | 4065 个；`apps/web` 2336 个 |
| Next.js 垫片的引用 | 313 个文件（`next/link` 54 处、`next/navigation` 282 处导入），加上包装层 `useAppRouter` 共 356 个文件；`packages/` 中没有引用 |
| 多语言 | 21 种语言 × 28 个命名空间 = 588 个 JSON；要删 19 种语言的 532 个 |
| `@plane/` | 1575 个文件中出现 4128 次 |
| 版权声明 `Copyright (c) 2023-present Plane Software, Inc.` | 2926 个文件，**保留**（总体设计 2.3） |
| 类型检查 | 13 个工作区全部通过 |
| oxlint | 合计 995 条，每个包都正好等于它的上限（web 779、editor 75、propel 59、utils 34、ui 32、services 6、hooks 4、i18n 3、constants 2、types 1，其余为 0）。M0 设计 5.2 写的"合计 1005"是加错了，本 M 一并更正 |
| knip | 未使用文件 131、依赖 19、开发依赖 4、未列出的依赖 2、导出 135、类型 83、枚举成员 4、重复导出 1；配置提示 3 条（干净克隆上 1 条） |
| 耗时 | 不走缓存的 `make lint-web` 约 30 秒；`make knip` 1.3 秒；不走缓存的 `make build-web` 14 秒（1226 个产物，30.5 MiB） |

删除量的估计来自三份只读清点（按功能列出了入口、可整体删除的文件、要修改的保留文件、类型、常量、文案和依赖）：内容类功能约删 500 个文件、改约 200 个文件；账户、平台和死代码约删 170 个文件，外加 `packages/services` 中的 49 个；多语言 532 个。

---

## 2. 删除清单

逐项对应前端改动清单第二节。"删""改"是文件数的估计，实施时以 spec 的清点为准。

### 2.1 内容类功能（P2）

| 功能 | 删 / 改 | 要点 |
|---|---|---|
| 文档页（Pages）及协作编辑 | 约 166 / 约 100 | 已核实：工作项描述、评论、收集箱、草稿用的都是非协作的编辑器，没有任何保留功能用到 Yjs、`description_binary`、工作项嵌入或页面提及。编辑器内核要改接口，见 3.1。可删除的依赖：`yjs`、`y-prosemirror`、`y-indexeddb`、`y-protocols`、`@hocuspocus/provider`、`@tiptap/extension-collaboration`、`@tiptap/html`、`buffer`、`@react-pdf/renderer`、`react-pdf-html` |
| 估算 | 53 / 约 80 | 迭代和模块的进度改为只按工作项数计算，见 3.4 |
| 甘特图与时间线 | 73 / 约 40 | 先把保留的关联关系用到的 `REVERSE_RELATIONS` 移出 `gantt-chart.ts`；最后删 `EIssueLayoutTypes.GANTT`，由类型检查列出所有剩下的引用 |
| 自动关闭 | 1 / 6 | "自动化"设置页只留自动归档 |
| 数据分析 | 约 77 / 约 27 | 连同 propel 中只给它和个人主页统计用的 `bar-chart`、`pie-chart`；燃尽图用的 `area-chart` 和 `recharts` 保留。迭代和模块的进度代码不在其中，见 3.4 |
| 导出 | 11 / 约 17 | 和"从未创建过的集成服务"一起删：导出记录列表借用了集成服务 |
| 便签 | 37 / 约 24 | 依赖 `react-masonry-component` |
| 首页快捷链接、首页个性化、自定义主题 | 61 / 约 26 | 首页正文在文档页、便签删完之后一次改好；已经没有入口的旧首页"仪表盘"（`DashboardStore`，调用的接口 Plane 自己也没有）一起删；亮色、暗色、跟随系统和高对比度主题与自定义主题无关，保留 |
| 侧边栏"自定义导航" | 约 10 个文件改为固定列表 | 见 3.3 |
| 个人主页的统计和动态 | 14 / 约 9 | 个人主页的新形态见 3.2 |
| "活跃迭代"推广页 | 10 / 约 8 | 只删工作区级的升级推广页；项目迭代列表中的"当前迭代"区块保留 |
| 更新日志（"what's new"） | 随文档页一起删 | 它的类型借用了文档页的 `TPage` |

### 2.2 账户、平台与死代码（P3）

| 功能 | 删 / 改 | 要点 |
|---|---|---|
| 公开发布 | 7 / 约 23 | 评论的"内部 / 外部"可见范围只为公开发布而存在，一起删，见 3.8 |
| 管理后台入口 | 3 / 约 10 | 包括"实例未完成设置"页（Nerve 的实例由配置文件和命令行管理），见 3.7 |
| 第三方登录、验证码登录、找回 / 重置 / 设置密码、登录前的"检查邮箱"步骤 | 31 / 约 28 | 登录页和注册页各剩一个"邮箱 + 密码"表单，模式由路由决定；CSRF 留到 M2，见 3.6 |
| 邮件通知偏好、营销邮件同意 | 5 / 约 9 | 站内通知保留 |
| 修改登录邮箱 | 1 / 约 3 | 靠邮件验证码完成，随邮件发送一起删除，见 3.13 |
| AI 助手 | 11 / 约 25 | 编辑器里的 AI 菜单在文档页删完之后再删，编辑器包只改一次 |
| Unsplash 封面图 | 1 / 4 | 封面选择只留上传 |
| 遥测残留 | 少量 | 已核实应用里没有遥测 SDK，只剩配置字段和文案 |
| 项目邀请 | 0 / 约 4 | web 里没有按邮件邀请进项目的流程，"邀请"弹窗本来就是从工作区成员中直接添加，只改名字和文案 |
| 企业版残留：Epic、团队、工作项类型、计费和升级提示、批量操作、`extended` 空壳 | 65 / 约 200 | Epic 贯穿工作项的 store、hooks、services（`issues\|epics\|work-items` 的地址切换），在 P2 删完甘特图、数据分析、文档页之后再删；计费和升级提示放在最后删。批量操作删掉之后的多选，见 3.9 |
| Plane 自身的死代码 | 约 40 / 约 15 | 注释掉的 IndexedDB store、knip 报告的未使用文件和导出、调用 Plane 后端里也不存在的接口的方法（已按 Plane 的 `urls.py` 逐条核对，59 个）。集成和导入器随导出在 P2 删除；`packages/services` 中没人用的 49 个文件在 P1 删除（3.5） |

### 2.3 删除的做法
- **每个功能删到这些层**：路由、导航和菜单入口（侧边栏、标签页、设置导航、命令面板、快捷键、空状态中的链接）→ 组件、store（及其在根 store 中的接线和 `resetOnSignOut`）、services、hooks → `packages/types` 中的类型和字段 → 常量和枚举 → 多语言文案 → 不再使用的依赖和图片资源。
- **类型检查和 knip 是删除的向导**：每删一层就运行一次，把它们报出的下一层接着删掉，直到两者都不再报告这个功能。knip 看不出接口里没人用的成员（比如编辑器 ref 的方法），这类地方按清点结果手工删。
- **文案最后删**：一个命名空间里的键，在所有代码引用都删掉之后再删；键可以不带命名空间调用，所以用"整键字面量 + 模板前缀"的方式核对引用，不能只搜命名空间。
- **不留过渡代码**：不写空实现、不加开关、不做"兼容旧字段"的转换。被删功能在保留代码里的分支（例如 `switch` 中的 `EPIC` 分支）直接删除。
- **根 store 和命令面板 store 串行修改**：6 个功能都要改 `root.store.ts`，同一个 Phase 里按任务顺序修改，不并行。
- **接口字段交给后续的 M**：前端不再读的 Plane 接口字段，列入对应 M 的 handoff，保证新接口不再带回这些字段（见 3.11）。

---

## 3. 设计裁定

清点中发现了几处总体设计没有写到的地方。以下裁定都在总体设计已经定下的范围内（删掉的功能、差异清单已经砍掉的表），不改变架构，也不涉及后端。

### 3.1 编辑器内核去掉协作接口
- **现状**：`packages/editor` 的公共内核（`editor-ref.ts`、`use-editor.ts`、`extensions.ts`、`unique-id`、`editor-container.tsx`）带着 Yjs 和 Hocuspocus 的接口。富文本和精简编辑器调用时总是传 `provider: undefined`，功能上没有依赖，但依赖包删不掉。
- **裁定**：文档页删完之后，重新整理内核的 ref 接口和 hook 参数：删掉 `getDocument().binary`、`setProviderDocument`、`emitRealTimeUpdate`、`listenToRealTimeUpdate`、`provider` 参数，以及只给文档页用的标题、文档信息成员。`UniqueID` 扩展是否还有保留功能在用，在 P2 spec 中核实后决定去留。改动只在编辑器包内。

### 3.2 个人主页
- **现状**：侧边栏的"我的工作"、快捷键 `g y`、成员列表中的名字、操作动态中的人名、@提及，全部链接到 `/:workspaceSlug/profile/:userId`，而这个地址正是要删的统计页。页头的用户名、用户卡片上的加入时间和时区也取自统计接口。访客只有统计这一个标签。
- **裁定**：
  - 个人主页只剩"分配给他的 / 他创建的 / 他关注的"三个工作项标签，加上右侧的用户卡片；每个项目的统计数字删掉。
  - 主页地址本身重定向到"分配给他的"标签，所有已有链接不用改。
  - 页头和用户卡片的数据改从工作区成员 store 读取，不再调用统计接口。成员数据里没有时区，卡片只显示头像、名字和加入工作区的时间（`joining_date`），时区删掉。
  - 访客沿用现有的权限规则，看到用户卡片和"无权访问"提示。访客能看到什么，由 M3 的权限矩阵最终决定（交接给 M3）。

### 3.3 侧边栏"自定义导航"随首页个性化一起删除
- **现状**：侧边栏的"自定义导航"（固定、排序、隐藏菜单项）读写 Plane 的 `/sidebar-preferences/` 接口，数据在 `workspace_user_preferences` 表。差异清单已经把这张表放在"首页个性化与主题"下砍掉了，但前端改动清单没有列出这个功能。
- **裁定**：删除。侧边栏改为固定列表（顺序取现在的默认顺序），删除自定义导航弹窗、偏好 store 和对应接口方法。项目导航的偏好（`workspace_user_properties`，保留的表）和应用栏的本地偏好不受影响。总体设计 1.2 的"首页个性化"和前端改动清单都补上这一项。

### 3.4 迭代和模块的进度
- **估算删掉之后**：进度、燃尽图只按工作项数计算。"按工作项数 / 按点数"的切换、只为点数存在的燃尽 / 燃起状态和 `distribution-update.ts` 中的点数计算一起删除（约 10 个文件）。
- **命名**：保留的迭代、模块进度代码目前叫 "analytics"（`analytics-sidebar`、`fetchActiveCycleAnalytics` 等），和要删的数据分析重名。在 P2 中改名为 progress。它调用的 Plane 接口地址本身带 `analytics`（如 `/cycles/${cycleId}/analytics?type=`），这是 Plane 的接口，M1 不改；关键词验收对这几个地址字面量开例外，M6 换成新接口后例外消失（交接给 M6）。

### 3.5 `packages/services` 在 M1 修剪
- web 只用到其中三样：令牌服务（`APITokenService`）、地址规范化（`normalizeAPIRequestURL`）、上传文件的元数据工具。其余 49 个文件是其他应用或已删功能的服务副本，没有任何调用方。
- **裁定**：P1 就删除这 49 个文件（它们没有调用方，删除不依赖任何功能的删除；而且其中几个文件引用了 P2 要删的类型，留到后面反而会让 P2 编译失败），包本身按总体设计 7.1 留到 M5 删除。这样关键词搜索不需要为这个包开例外。

### 3.6 认证：M1 只删界面上的功能，传输方式留给 M2
- **现状**：登录、注册、退出、修改密码、新手引导中的设置密码，都通过 Django 会话 + CSRF 令牌提交（原生表单 POST 带 `csrfmiddlewaretoken`，或 `X-CSRFTOKEN` 请求头）。
- **裁定**：
  - M1 删除第三方登录、验证码登录、找回 / 重置 / 设置密码页面和"检查邮箱"步骤；登录页和注册页各剩一个"邮箱 + 密码"表单，模式由路由决定。
  - CSRF 和会话提交方式是认证传输层的一部分，和替代它的令牌管理器一起在 M2 删除（总体设计 9.2 中 M2 负责"前端令牌管理器和登录页"）。M1 先删掉它，只会让这几处调用停在一种 M2 马上又要重写的中间状态。
  - 前端改动清单中"CSRF 相关代码"一行从 M1 移到 M2；交接给 M2。

### 3.7 实例配置与启动
- `InstanceWrapper` 在启动时读取 Plane 的实例配置，其中的开关决定认证页显示什么（第三方登录、验证码、SMTP、AI、Unsplash 等）。
- **裁定**：随着这些功能删除，前端不再读取对应的开关，类型中的字段一起删除；"实例未完成设置"页（`is_setup_done`）删除。`enable_signup`（是否开放注册）保留，由 M2 的实例配置接口决定。实例配置接口的最终字段在 M2 定义，前端届时改用生成的类型。

### 3.8 评论的可见范围
评论的"内部 / 外部"（`access`）只在项目公开发布时有意义。**裁定**：界面和类型中删除；`issue_comments.access` 列是否保留，交接给建表的 M4（M0 设计 7.1：列级别的细节由建表的 M 补全）。

### 3.9 批量操作删除之后的多选
批量操作是 Plane 的付费功能，调用的接口 Plane 社区版也没有。**裁定**：删除批量操作；多选的 store、组件和 hooks 如果因此没有别的使用者，一并删除；如果还有保留功能在用，就保留。由 P3 用类型检查和 knip 核实。

### 3.10 `@makeplane/propel` 保留为第三方依赖
- 它是 Plane 发布在 npm 上的设计系统（0.3.0，AGPL-3.0-only），提供设计令牌（样式变量）和 523 处图标导入，是 web、editor、propel、tailwind-config、ui、utils 的依赖，不是工作区里的包。
- **裁定**：保留，照常锁定版本。它的名字只出现在导入路径里，界面上看不到；许可证与本仓库一致。包名重命名（第 5 节）只针对工作区里的 `@plane/*`，品牌关键词搜索排除 `@makeplane/`。其中带 Plane 品牌的图标只有 AI 相关的几个，随 AI 助手一起不再使用。
- **代价与退路**：界面的设计令牌和大部分图标长期依赖 Plane 发布的这个包。锁定的版本不会自己变化；如果以后需要自己修改设计令牌，或者上游改变许可证、停止发布，可以随时按 AGPL 把它的源码并入工作区的 propel（它在 npm 上只有构建产物，源码在 Plane 的 propel 仓库）。M1 不做这件事：它不影响删减，并入要引入一个新的上游仓库，工作量和风险都不属于 M1。这一项向项目负责人报备。

### 3.11 交给后续 M 的接口字段
前端不再读取下列 Plane 接口字段，后续 M 设计接口时不应带回它们。P2、P3 结束时按领域写进对应 M 的 handoff：
- M2：`last_login_medium`、`is_password_autoset`、`is_email_verified`、`has_marketing_email_consent`、计费和试用字段；主题只剩 `{theme}`；CSRF 和会话（3.6）；是否提供修改登录邮箱（3.13）。
- M3：`page_view`、`close_in`、`default_state`、`anchor`；侧边栏偏好接口不再需要（3.3）；访客能否查看个人主页（3.2）；工作区保留地址名单（`packages/constants/src/workspace.ts` 的 `RESTRICTED_URLS`）：它防止工作区地址和应用的一级路径冲突，各 Phase 删掉自己的功能对应的条目（`god-mode`、`epics`、`upgrade`、`billing`、`integrations`、`importers`、`pages`、`plane-pro` 等），M3 让后端的保留名单与删减后的前端一致。
- M4：`estimate_point`、`type_id`、`is_epic`、`description_binary`、评论的 `access`（3.8）、`POST /issue-dates/`；搜索结果不再有 `page`。
- M5：资源类型 `PAGE_DESCRIPTION`。
- M6：迭代和模块的进度不带 `points` 和 `estimate_distribution`（3.4）；进度接口换掉之后，关键词验收中 `analytics` 的地址例外随之消失（3.4）。
- M7：收藏和最近访问不再有 `page` 类型。

### 3.12 Plane 的旧地址重定向
`routes/core.ts` 末尾的 13 条"为兼容旧地址"的重定向（`/sign-in`、`/login`、`/accounts/sign-up` 等）是 Plane 的历史包袱。**裁定**：删除，两处还指向 `/sign-in` 的内部链接改为 `/`。

### 3.13 修改登录邮箱
- **现状**：个人资料中的"修改邮箱"弹窗先向新邮箱发送验证码，输入验证码后才生效。
- **裁定**：随"所有邮件发送"（总体设计 1.2）一起删除。v0 是否提供不经邮件的改邮箱方式（例如输入当前密码确认，或者只能由管理员用命令行修改），由 M2 设计个人资料接口和管理命令时决定（交接给 M2；与总体设计第 10 节"没有邮件服务时账户如何找回"是同一类问题）。M2 之前前端还连不上认证接口，这段时间里没有用户受影响。这一项向项目负责人报备。

---

## 4. 去掉 Next.js 兼容层（P4）

### 4.1 要删的东西
- 垫片：`app/compat/next/*`、`app/types/next-link.d.ts` 和 `next-navigation.d.ts`、`vite.config.ts` 中的两个别名、包装层 `use-app-router.tsx`。其中没人用的 `image.tsx`、`script.tsx`，以及 `next/script` 的别名和 `next-script.d.ts`，是死代码，P1 先删（knip 在 P3 结束时要清零）。
- 强制结尾 `/` 的两处：垫片里的 `ensureTrailingSlash`，以及 `app/layout.tsx` 中返回 308 的 `clientMiddleware`。删掉后地址统一不带结尾 `/`。
- `process.env` 的注入：`vite.config.ts` 顶部的 dotenv 加载和 `define: { "process.env": … }`，`web/apps/web/.env.example`，`turbo.json` 中的 `VITE_*` 环境变量和 `build.inputs` 中的 `.env*`。
  - 代码中读取的每个 `VITE_*` 变量：属于已删功能的，已随功能删除；`VITE_API_BASE_URL` 删除，接口一律用相对路径（同源部署，见 M2 的 [M0-P5-frontend-api-notes](../M2-auth/handoffs/M0-P5-frontend-api-notes.md)），`APIService` 不再接收基础地址参数；`VITE_SUPPORT_EMAIL` 删除，文案改为"联系管理员"。
  - 删除顺序受一个约束：各个包用 tsdown 构建出的 `dist/` 原样保留 `process.env.X`，靠 `define` 才不报错。所以先删掉所有读取，再删 `define`。
- Next.js 的其他遗留：`app/(all)/layout.preload.tsx`（内容全被注释掉）、`typescript-config/nextjs.json`、各处忽略列表中的 `.next`。

### 4.2 改写方式
- **机械部分用一次性的 ts-morph 脚本**（脚本不进仓库，放在提交说明和 review 中说明）：
  - `next/link` 的 `Link href` → React Router 的 `Link to`（没有用到 `prefetch`、`scroll`、`shallow`、`replace`）；
  - `usePathname()` → `useLocation().pathname`；`useSearchParams()` → `const [searchParams] = useSearchParams()`；`useParams` 改从 `react-router` 导入；
  - `router.push(x)` → `navigate(x)`，`router.replace(x)` → `navigate(x, { replace: true })`，`router.back()` → `navigate(-1)`；命令面板上下文中的 `router` 字段改为 `navigate: NavigateFunction`。
- **手工部分**：
  - **路由参数的类型**：垫片的手写类型说参数总是 `string`，React Router 的真实类型是 `string | undefined`，换过去之后有 152 处类型错误（68 个文件，其中一部分在 P2、P3 删掉的功能里）。每处都按真实情况处理：路由组件用 `./+types/*` 的 `params`（参数一定存在），组件里用守卫。不用非空断言批量压掉错误。
  - **渲染时跳转**：`AuthenticationWrapper` 在组件渲染过程中调用了 8 次跳转，只因为垫片用 `setTimeout` 推迟了跳转才能工作；React Router 会丢掉首次渲染中的跳转。改为渲染 `<Navigate replace />`。这个包装层在 M2 随令牌管理器重写，M1 只做让它在原生路由下正确工作的最小改动。
  - **依赖结尾 `/` 的比较**：约 12 处当前菜单项的判断拿 `pathname` 和带结尾 `/` 的地址直接比较，改用 React Router 的匹配（`NavLink` 的 `isActive`、`matchPath`）；没有调用方的 `highlight` 函数删除。
- **验证**：
  - 类型检查、knip、构建通过；`git grep -nE "next/(link|navigation|script|image)|useAppRouter|ensureTrailingSlash|compat/next|process\.env" -- web` 没有结果（`process.env.NODE_ENV` 改为 `import.meta.env.DEV`）。
  - Vite 构建时会替换 `process.env`，开发服务器对预构建的依赖不会。删掉 `define` 之后，用 `make web-dev` 打开首页，确认控制台没有 `process is not defined`。
  - M1 的后端还不提供 Plane 的接口，页面停在启动错误页，没法端到端走一遍路由。P4 用一次性的 Playwright 脚本拦截 `/api/**`、返回最小的 Plane 响应，核实：未登录跳到 `/?next_path=…`、未完成引导跳到 `/onboarding`、侧边栏和设置导航的当前项在带不带结尾 `/` 时都正确、控制台没有"navigate() 应在 useEffect 中调用"的警告。脚本不进仓库（它模拟的是 M2 就要替换掉的旧接口），结果写进 review。

---

## 5. 品牌与包名（P5）

| 类别 | 处理 |
|---|---|
| 版权声明（2926 个文件） | **保留**（总体设计 2.3） |
| Logo、网站图标、应用清单、加载动画 | 换成 Nerve 的图形；propel 中的 `PlaneLogo`、`PlaneLockup`、`PlaneWordmark` 等改为 Nerve 的对应组件 |
| 页面标题、`SITE_NAME` 等元数据 | 改为 Nerve |
| 中英文文案中的 "Plane"（各约 108 行） | 改为 Nerve；讲的是 Plane 公司或 Plane 服务的句子（如"联系 Plane 支持"）按含义改写或随功能删除 |
| 硬编码的文案（约 70 行） | 同上 |
| Plane 的网址（plane.so、文档、论坛、状态页、GitHub、社交账号） | 帮助菜单、错误页中的这类链接删除；需要外部链接的地方指向 Nerve 的仓库 |
| 代码标识符、本地存储键（`plane_tab_prefs` 等）、组件 `displayName`（`plane-ui-*`） | 改为 Nerve 的名字 |
| 注释 `// plane imports` 等（约 1200 行） | 随包名一起改为 `// nerve imports` |
| `@makeplane/propel` | 保留（3.10） |

工作区保留名单中的 `plane-pro`、`plane-ultimate`、`plane-enterprise` 是为 Plane 付费版本保留的地址，属于计费残留，在 P3 删除。

- **Logo**：Nerve 目前没有设计稿。P5 做一套简单的矢量图形（图标、横版标志、网站图标的各个尺寸），之后可以直接替换文件，不需要改代码。
- **包名 `@plane/*` → `@nerve/*`**：单独一个纯机械的提交，放在删除和 P4 之后（涉及的文件最少）。
  - 用 `sed 's#@plane/#@nerve/#g'` 替换 `web/` 中的全部出现（不会误伤 `@makeplane/`），再执行 `pnpm install` 重写锁文件。
  - 完整性核对：`git grep -n "@plane/"` 在 `docs/` 以外没有结果；锁文件相对上一个提交只有包名的变化（`git show HEAD:pnpm-lock.yaml | sed 's#@plane/#@nerve/#g' | diff - pnpm-lock.yaml` 为空）；`pnpm -r ls --depth -1` 只列出 `@nerve/*` 和 `web`；`knip.jsonc` 中的工作区路径是目录，不用改（P6 交接）。
- **S2 仍不断言页面文字**：M2 之前页面还是启动错误页（P6 交接）。

---

## 6. 多语言（P1）
- 删除 19 种语言的目录（532 个文件）；`SUPPORTED_LANGUAGES` 只剩两项，`TLanguage` 改为 `"en" | "zh-CN"`。
- 服务端或本地存储里的语言不在支持范围内时，回退到 `en`，语言选择框显示英文。
- **翻译键的生成**：`keys.generated.ts` 生成出来之后没有任何地方使用（`t` 的参数类型是 `string`，还有 187 处用变量调用）。删除生成脚本、`generate:types` 和 `knip.jsonc` 中 i18n 的 `ignoreUnresolved`（P6 交接）。
- **中英文键一致**：已有的 `sync-check` 脚本改为只核对 `zh-CN` 与 `en` 的键集合，接进 `make lint-web`，作为门禁。M1 删文案时，中英文必须同时删。
- **企业版的整个命名空间**：`automation`、`workflow`、`template`、`work-item-type`、`tour`、`update`、`wiki` 等命名空间，以及 `workspace_settings.settings.applications.*`（176 个键），代码中一处引用都没有（清点时按"整键字面量 + 模板前缀"核对），是死文案，连同命名空间的登记在 P1 删除。只在 P2、P3 删完代码之后才失去引用的键，由那两个 Phase 删除。
- 各处写死的 `"en-US"`（日期问候、时区换算）保持不变，不在本 M 范围。

---

## 7. 质量门禁

### 7.1 lint 上限自动核对（P1）
- **现状**：每个包的上限只防止警告变多；警告减少后"同一个提交里调低上限"只靠评审把关（M0 设计 5.2）。M1 会删掉大量代码，几乎每个任务都会让警告变少。
- **裁定**：P1 加一个自动检查，接进 `make lint-web`：每个包的实际警告数必须**等于**上限，多了或少了都失败。这样上限永远等于实测值，"重新测出基线"不再需要单独做。
  - 警告比上限多：提示"修掉新增的警告，上限只能调低"，不给出调高后的数值（M0 设计 5.2：只降不升）。
  - 警告比上限少：提示把上限调低到实测值。
  - 上限仍然只有一处来源，检查要能正确利用 turbo 的缓存；具体做法在 P1 spec 中通过原型确定。
- `@nerve/api-client`、`@nerve/e2e` 的上限保持 0（P6 交接）。

### 7.2 knip 改为门禁（P3 结束时）
- P3 删完死代码、knip 报告为零之后，`make knip` 去掉 `--no-exit-code`，成为持续集成的门禁；P4、P5 和之后的所有 M 都在门禁下进行。
- **配置提示**：配置提示的条数随本地生成过哪些文件而变（P6 交接），根因是 knip 有时在生成之前运行、有时在生成之后运行。
  - P1 修好 `tailwind-config` 的 `main`，删掉 i18n 的生成步骤和它的 `ignoreUnresolved`。
  - P3 让 `make knip` 先运行 web 的 react-router 类型生成（`+types/` 由它生成，几秒以内），这样 knip 总是在同一种状态下运行；随后删掉 web 的 `ignoreUnresolved`，并打开 `--treat-config-hints-as-errors`，配置本身过时也会被门禁发现。`knip.jsonc` 中"不要删"的注释随之删除。

### 7.3 oxlint 清零计划
- M1 结束时，按包、按规则重新列出警告数（由 7.1 保证就是上限），写进 M1 的收尾 review。
- **计划**（写进收尾 review，由后续每个 M 执行）：
  1. 谁改谁清：后续 M 修改的文件，在该 M 结束时不再有警告。M2 起前端要逐个领域改写 stores 和组件，大部分警告会随之清掉。
  2. 按规则集中清理：没有被改到的文件，按规则分批清理（例如 `no-shadow`、`promise/always-return`、`no-unneeded-ternary` 这类可以机械修复的规则），每个 M 至少清掉一类，并在该 M 的 review 中记录。
  3. 目标：M8 发布前全部清零，之后 oxlint 警告即报错（总体设计 7.6 的原始要求）。

### 7.4 关键词验收清单
总体设计 7.4 要求关键词搜索没有结果，前端改动清单约定清单在本文件确定。统一用 `git grep -I -n -P '<模式>' -- web` 搜索（只搜跟踪的文件；`-I` 跳过字体等二进制文件；macOS 自带的 git 的 `-E` 不支持 `\b`，所以用 `-P`），再加一遍文件名检查 `git ls-files web | grep -iE '<模式>'`。

| 功能 | 模式 | 注意 |
|---|---|---|
| 文档页、协作 | `page_view`、`description_binary`、`(?i)\byjs\b\|y-prosemirror\|y-indexeddb\|y-protocols\|hocuspocus`、`Collaborative\|extension-collaboration`、`\bpageId\b\|\bpage_id\b`、`WorkItemEmbed\|issue-embed`、`LIVE_BASE\|VITE_LIVE`、`\bcomlink\b\|react-pdf\|@tiptap/html`、`(?i)wiki` | 不搜单独的 `page`、`pages`；`wiki` 排除 `packages/utils/src/tlds.ts` |
| 估算 | `(?i)estimat`、`估算` | |
| 甘特图、时间线 | `(?i)gantt`、`(?i)(?<!view_)time_?line`、`issue-dates` | 保留的 `useTimeLineRelationOptions` 在 P2 改名 |
| 自动关闭 | `\bclose_in\b`、`(?i)auto[-_ ]?close`、`\bdefault_state\b` | |
| 数据分析 | `(?i)analytics` | 依赖 3.4 的改名；排除 `tlds.ts`，以及保留的迭代、模块进度 service 中调用 Plane 进度接口的地址字面量（到 M6 为止，3.4） |
| 导出 | `(?i)exporter`、`settings/exports\|export-issues\|EXPORTERS_LIST\|IExportData` | 不搜单独的 `export` |
| 便签 | `Sticky\|STICKY\|[Ss]tickies\|STICKIES\|sticky[-_.:]`（区分大小写） | 不搜 `(?i)stick`（会命中 `AxisTick`）或单独的 `sticky`（CSS） |
| 首页、主题 | `quick_links\|quick-links\|QuickLink\|HomeWidget\|manage_widgets\|home-preferences`、`CustomTheme\|customize_your_theme\|palette-generator`、`DashboardStore\|useDashboard` | 不搜单独的 `widget`、`dashboard`、`custom` |
| 侧边栏自定义 | `sidebar-preferences\|SidebarNavigation\|customize-navigation\|navigationPreferences` | |
| 个人主页统计 | `user-stats\|user-profile/\|user-activity/\|IUserProfileData\|profile\.stats\|ProfileSidebar\|your_work_by_` | `user-activity` 带结尾 `/`（propel 有同名图标） |
| 活跃迭代推广 | `active-cycles\|active_cycles\|WorkspaceActiveCycles\|\bworkspaceActiveCycles\b` | 不搜 `active_cycle`、`ActiveCycle`（保留的当前迭代区块） |
| 公开发布 | `SPACE_BASE\|SPACE_APP\|SITES_URL\|deploy-boards\|PublishProject\|ProjectPublish\|publish_project` 搜 `web/`；`\.anchor\b\|anchor\??:` 只搜 `web/apps/web` 和 `web/packages/types` | 不搜单独的 `publish`（草稿"发布为工作项"保留）；编辑器里的 `anchor`（选区、链接）是保留代码 |
| 管理后台 | `god[-_]mode\|GOD_MODE\|ADMIN_BASE\|instance-admin\|InstanceNotReady\|is_setup_done` | |
| 认证 | `(?i)oauth`、`unique[-_]code\|UNIQUE_CODE\|UniqueCode\|magic[-_](generate\|code\|login)\|MAGIC_`、`forgot[-_]password\|reset[-_]password\|set[-_]password\|SET_PASSWORD`、`email-check\|emailCheck`、`(?i)gitea\|gitlab\|is_google_enabled` | 不搜单独的 `magic`（propel 图标名 `magic_exchange`）。CSRF 留到 M2（3.6），不在本清单 |
| 邮件 | `email_notification\|marketing_email\|EmailNotification` | |
| AI | `\bAI[A-Z_]\|\bai[-_]\|PlaneAi\|(?i)\bgpt\|\bllm\|rephrase` | 区分大小写的部分放在 `(?i)` 之前（`(?i)` 对它之后的整段生效）；具体模式在 P3 spec 中按命中结果收窄 |
| Unsplash、遥测、更新日志 | `(?i)unsplash`、`(?i)posthog\|telemetry\|sentry\|intercom`、`(?i)changelog\|product-updates\|what.s new` | `what's` 的单引号会打断命令行的单引号，用 `.` 代替 |
| 项目邀请 | `ProjectInvitation\|project.*invitation` | 已知例外：`joinProject` 调用的地址 `/projects/invitations/`（`core/services/user.service.ts`）是 Plane 的接口，M3 换成新接口 |
| 企业版残留 | `(?<![dD])epic\|Epic\|EPIC`、`teamspace\|TEAM_VIEW\|TEAM_PROJECT\|TEAM_SPACE`、`type_id\|issueTypeId\|IssueTypeSwitcher\|work_item_type`、`(?i)upgrade\|(?i)billing\|EProductSubscription\|ProIcon\|contact_sales\|plane-pro\|LockedComponent`、`(?i)bulk[-_]?operation`、`TExtended\|useExtended(Editor\|OAuth)\|ExtendedBasePage\|auth-ee\|extendedRoutes\|useAdditional\|AdditionalSlash\|TAdditional\|CE/EE\|no-op in CE` | `upgrade` 在依赖名和注释中的命中逐条核对；侧边栏的 `extended-sidebar*` 是保留的界面，不在模式里 |
| 死代码 | `(?i)integration\|importer\|jira\|slack\|github-repository\|app-installation\|indexeddb` | Webhook 保留，不会命中这些词 |
| Next.js 垫片 | `next/(link\|navigation\|script\|image)\|useAppRouter\|ensureTrailingSlash\|compat/next\|process\.env` | |
| 多语言 | `git ls-files web/packages/i18n/src/locales \| grep -vE '/locales/(en\|zh-CN)/'` 没有输出 | `grep -E` 不支持 `(?!…)`，所以用反向匹配 |
| 品牌 | `(?i)plane`，排除版权声明行（`Copyright (c) 2023-present Plane Software, Inc.`）、`@makeplane/`，以及 airplane、planet 这类普通单词 | 包名：`@plane/` |

- **按 Phase 验收**：P2、P3 各自负责清单中自己的功能。一个 Phase 结束时，它负责的关键词只允许命中"后续 Phase 才删除的文件"（例如 P3 才删的计费页面里也出现了 `estimat`、`gantt`），每一处都在该 Phase 的 review 中列出文件和负责删除它的 Phase；收尾时整张清单必须没有结果（上表写明的已知例外除外）。
- 模式在各 Phase 的 spec 中可以按实际命中进一步收窄（只能更精确，不能为了通过而扩大排除范围）；每条排除都要在 review 中说明理由。
- 收尾时对整张清单跑一遍，命令和结果写进收尾 review。

---

## 8. 工具链与部署遗留（P1）
来自两份 M0 交接和清点：
- **部署遗留**：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`；`serve` 依赖及 `start`、`preview` 脚本（崩溃的根因是全局覆盖 `path-to-regexp: 0.1.13` 强加给了 `serve-handler`，删掉 `serve` 后这条覆盖也不再作用于任何包）；`public/` 中从未注册的 `sw.js`、`workbox-*.js` 及 source map；没有被引用的 `manifest.json`、`public/favicon/site.webmanifest`；13 个 `.prettierignore`（格式化工具是 oxfmt）；`web/apps/web/.gitignore`（只有 Sentry 的条目）。
- **pnpm 工作区**：删掉提到没有迁入内容的注释；删掉已不作用于任何包的覆盖项。Express 系列的覆盖项仍然作用于 `@react-router/dev` 的可选对等依赖 `@react-router/serve`，保留。用锁文件核对修剪前后的解析结果（P5 spec 2.4 的方法）。
- **没有调用方的任务和脚本**：`turbo.json` 中的 `start`、`build-storybook`、`clean`、`check`、`fix*`、`test`，以及各包里同样没有调用方的脚本；Storybook（ui 和 propel 各一套，没有 Makefile 或持续集成入口）连同其配置、stories 和依赖。原则：任务或脚本只有被 Makefile、持续集成、另一个任务调用，或者是 README 写明的开发入口，才保留。
- **未使用的依赖**：knip 报出的 19 个依赖、4 个开发依赖、2 个未列出的依赖（其中只被未使用文件引用的依赖，连同这些文件一起删除）。例外：
  - editor 的 `buffer`：knip 把它当成了 Node 的内置模块，实际上 Yjs 的工具代码还在导入它，随 Yjs 在 P2 删除。
  - `isbot`、`@react-router/node` 保留：应用没有自己的 `entry.server.tsx`，React Router 的默认服务端入口导入它们（SPA 模式构建时也要用它预渲染 `index.html`），删掉后 React Router 会自己把 `isbot` 加回来。
- **死代码**：Next.js 垫片中没人用的 `image.tsx`、`script.tsx`，`next/script` 的别名和 `next-script.d.ts`（4.1）；`packages/services` 中没人用的 49 个文件（3.5）。
- **构建警告**：`tailwind-config` 的 `package.json` 加 `"type": "module"`，`main` 指向真实存在的入口；`vite-tsconfig-paths` 插件换成 Vite 8 的 `resolve.tsconfigPaths`。
- **TypeScript 增量编译文件**：`typescript-config/base.json` 把 `tsBuildInfoFile` 写成了相对它自己的路径，13 个包写同一个文件、互相覆盖。改为每个包写自己的。
- **React #418**：`root.tsx` 的 `HydrateFallback` 在预渲染时输出空的 `<div>`，在浏览器首次渲染时已经知道主题、输出加载动画，两者不一致。改为预渲染和首次渲染输出同样的结构。
- **`make web-dev`**：改为同时监视各个包（`turbo run dev --filter=web...`），并发数按持久任务数设置（11 个持久任务，至少 12）；M0 设计 6.2 和 README 中"改了 packages 要重新执行"的说明随之修改。

---

## 9. Phase 划分与实施规划

所有 Phase 依次推进。每个 Phase 按 M0 的节奏：worktree → spec 和 plan（先做原型验证）→ 按任务实现并逐个评审 → 整分支评审（opus）→ 修复和限定范围的复审 → review、交接 → `--no-ff` 合并 → 推送 → 持续集成通过 → 清理 worktree 和分支。每个 Phase 合并时，持续集成的全部门禁和 S1–S4 都必须通过。

### P1 `web-hygiene`：工具链、遗留物与多语言
- **交付物**：第 8 节的全部内容（含 3.5 的 `packages/services` 修剪和垫片中的死文件）；第 6 节的多语言（含企业版的整个命名空间）；3.12 的旧地址重定向；7.1 的 lint 上限自动核对。
- **验收**：
  - `make lint-web`（含上限核对和中英文键一致性检查）、`make build-web`、`make e2e` 通过；
  - knip 报告中未使用的依赖、开发依赖、未列出的依赖三类为零（editor 的 `buffer` 除外，见第 8 节）；配置提示只剩 web 的 `+types/` 一条（是否出现取决于本地是否生成过类型，P3 处理，见 7.2）；
  - 构建不再有 `MODULE_TYPELESS_PACKAGE_JSON` 和 `vite-tsconfig-paths` 的警告；浏览器控制台没有 #418；
  - 锁文件核对：除删掉的包以外，解析结果不变。

### P2 `trim-content`：内容类功能
- **交付物**：2.1 的全部功能；3.1–3.4 的裁定；属于这些功能的文案、类型、常量、依赖和图片资源；交给 M3、M4、M6、M7 的接口字段 handoff（3.11）。
- **顺序**（清点得出的约束）：侧边栏自定义导航先删（后面几个功能都要改侧边栏）；数据分析在估算之前；文档页在编辑器内核之前；文档页、便签在首页正文改写之前；导出和集成服务一起删；propel 的图表组件在数据分析和个人主页统计之后删。
- **验收**：
  - 类型检查、lint（上限等于实测）、构建、S1–S4 通过；
  - 7.4 中属于本 Phase 的关键词只命中后续 Phase 才删除的文件，每一处都在 review 中列出（7.4 的"按 Phase 验收"）；
  - knip 不再报告这些功能的任何文件、导出和依赖；
  - 锁文件核对：删掉依赖之后，`pnpm-workspace.yaml` 中只作用于它们的 catalog、overrides、allowBuilds 条目一并删除（pnpm 不会报告失效的条目），除删掉的包以外解析结果不变；
  - 改变运行时行为的地方（个人主页的重定向、固定的侧边栏）用拦截 `/api/**` 的一次性 Playwright 脚本核对，结果写进 review（见第 10 节）。

### P3 `trim-platform`：账户、平台与死代码
- **交付物**：2.2 的全部内容；3.5–3.9 的裁定；knip 清零并改为门禁（7.2）；交给 M2、M4、M5 的 handoff（3.6、3.8、3.11）。
- **顺序**：认证和实例配置在一个任务里改（它们改同一个接口类型）；Epic、团队、工作项类型在同一个任务里删（它们改同一组 store 和 hook 的分支）；计费和升级提示最后删。
- **验收**：同 P2（运行时核对的对象是登录页、注册页和启动流程）；另外 `make knip` 不带 `--no-exit-code`、带 `--treat-config-hints-as-errors` 通过，持续集成中生效。

### P4 `router-native`：去掉 Next.js 兼容层
- **交付物**：第 4 节的全部内容；README 中关于 `.env` 的两处说明（P6 交接）。
- **验收**：同 P2；4.2 的搜索没有结果；一次性 Playwright 核对的结果写进 review。

### P5 `brand`：品牌与包名
- **交付物**：第 5 节的全部内容；前端改动清单第四节。
- **验收**：同 P2；包名核对（第 5 节）；品牌关键词没有结果。

### 收尾 `closeout`
- 对 7.4 的整张清单跑一遍；核对中英文文案中没有代码引用的键（"整键字面量 + 模板前缀"，用变量调用的键按常量逐一核对）并删除；最后一次锁文件核对；按包、按规则列出 oxlint 警告数和清零计划（7.3）；前端改动清单、总体设计 9.4 同步；本 M 的 handoff 全部关闭；总体设计中 M1 的状态改为"已完成"。

### 9.7 两份 M0 交接的落点
| 交接事项 | Phase |
|---|---|
| 部署遗留、`serve`、`sw.js`/workbox、pnpm 工作区的覆盖项和注释、没有调用方的 turbo 任务 | P1 |
| 功能删减之后用锁文件核对 catalog、overrides、allowBuilds | P1、P2、P3 各做一次（每次删依赖之后），收尾再做一次 |
| `.env.example`、dotenv、`define: process.env`；README 中关于 `.env` 的说明 | P4 |
| 重新测出警告基线、是否加自动检查 | P1（7.1）、收尾（7.3） |
| React #418、`tailwind-config` 的 `"type": "module"`、`vite-tsconfig-paths`、`make web-dev` | P1 |
| 结尾 `/` 在去掉垫片后是否还需要 | P4（不需要，4.1） |
| knip 改为门禁、配置提示 | P3（7.2）；i18n 的 `ignoreUnresolved` 在 P1 删除 |
| `@nerve/api-client`、`@nerve/e2e` 的上限保持 0 | P1（7.1） |
| S2 不断言页面文字；`knip.jsonc` 的工作区路径不随包名改 | P5 |

---

## 10. 验收方式
- M1 是静态验收（总体设计 9.2）：类型检查、oxlint、knip、格式检查、构建、关键词清单。
- **端到端**：M1 不新增用户故事；S1–S4 在每个 Phase 合并时都通过（总体设计 9.3："之前所有 M 的故事也都通过"）。S2 核对首页、深层路径、同源请求，能发现构建产物和内嵌方面的倒退。
- **运行时核对**：M1 的后端还不提供 Plane 的接口，页面停在启动错误页。改变运行时行为的 Phase 用一次性的 Playwright 脚本拦截 `/api/**`、返回最小的 Plane 响应来核对，脚本不进仓库（它模拟的是 M2 起就要替换的旧接口），结果写进各自的 review：
  - P1：#418 修复（浏览器控制台）、旧地址重定向删除之后的内部链接；
  - P2：个人主页的重定向、固定的侧边栏；
  - P3：登录页、注册页和启动流程；
  - P4：4.2 列出的各项。
- **没有 PAT 对等验收**：M1 不涉及接口。

---

## 11. 完成标准
- [ ] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。
- [ ] 类型检查通过；knip 为零，并且是持续集成的门禁；oxlint 的每个包都等于新的上限，上限由 7.1 自动核对；格式检查通过。
- [ ] 7.4 的关键词清单全部没有结果（排除项都在 review 中说明了理由）；中英文文案中没有代码不再引用的键。
- [ ] `make build` 能构建；S1–S4 全部通过。
- [ ] 多语言只剩 `zh-CN` 和 `en`；界面上没有 Plane 的名称和 Logo；包名全部是 `@nerve/*`。
- [ ] oxlint 清零计划写进收尾 review。
- [ ] `handoffs/` 中没有 `open` 状态的事项；交给后续 M 的事项已放进对应 M 的 `handoffs/`。
- [ ] 前端改动清单同步（第二、四节，以及本文件第 3 节带来的增减）；M0 设计 5.2 的合计更正为 995；总体设计中 M1 的状态改为"已完成"。

---

## 12. Phase 进度表

| Phase | 名称 | 状态 | spec | plan | review |
|---|---|---|---|---|---|
| P1 | web-hygiene | 未开始 | — | — | — |
| P2 | trim-content | 未开始 | — | — | — |
| P3 | trim-platform | 未开始 | — | — | — |
| P4 | router-native | 未开始 | — | — | — |
| P5 | brand | 未开始 | — | — | — |
| 收尾 | closeout | 未开始 | — | — | — |

---

## 13. 风险

| 风险 | 应对 |
|---|---|
| 删除量大（约 1200 个文件被删、数百个文件被改），P2、P3 的任务之间容易改到同一批文件（根 store、侧边栏、工作项 store 的分支） | 按第 9 节的顺序串行推进；每个任务结束时类型检查、lint、构建都要通过，出问题只回退这一个任务 |
| 前端连不上 Plane 的接口，删错了也不会在运行时暴露 | 以类型检查和 knip 为主；删除只针对清点列出的功能，保留代码中的改动在任务评审中逐处核对；P4 用拦截接口的脚本核对路由 |
| 去掉垫片后路由参数的类型变严，暴露出 152 处可能为空的参数 | 放在删除之后做（相关文件更少）；逐处按真实情况处理，不批量断言 |
| 翻译键可以不带命名空间、也可以用变量调用，删文案时容易删多或删少 | 用"整键字面量 + 模板前缀"的方式核对引用；中英文键一致性由门禁保证；收尾时跑一遍未引用键的核对 |
| lint 上限自动核对让每个减少警告的提交都必须同时改上限 | 这正是 M0 定下的规则；检查会直接给出应改成的数值 |
| Logo 没有设计稿 | P5 做简单的矢量图形，文件可以直接替换 |
