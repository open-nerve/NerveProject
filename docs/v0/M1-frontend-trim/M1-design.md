# M1 前端瘦身：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M1 前端瘦身（`docs/v0/M1-frontend-trim`） |
| 日期 | 2026-09-22 |
| 状态 | 已确认（控制者按 2026-09-22 的授权推进）。第 3 节的裁定都在总体设计已定的范围内，都可以在以后改回，M1 不做任何后端工作；其中几项会约束后续 M 的接口，负责的 M 和决策点见 3.11 |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.1、1.2、7、8、9 节；[前端改动清单](../frontend-changes.md)；[差异清单](../plane-diff.md) |
| 前置交接 | [M0-P5-frontend-trim-notes](handoffs/M0-P5-frontend-trim-notes.md)、[M0-P6-knip-notes](handoffs/M0-P6-knip-notes.md)（逐条落到第 9 节的 Phase，见 9.7） |
| 设计评审 | 1. 内部独立评审（opus）：3 条 Critical、6 条 Important、4 条 Minor，已全部吸收。<br>2. Codex 对抗性评审（[报告](reviews/M1-design-codex-adversarial-review.md)）：1 条 Critical、6 条 Important、2 条 Minor、1 条建议，经核实全部成立，已吸收；逐条处理结果见报告第 10 节 |

---

## 0. 目标与范围

### 0.1 目标
把 M0 原样迁入的 Plane 前端删到只剩 v0 要的功能，并清掉迁入时带进来的兼容层和遗留物。M1 结束时：
1. 总体设计 1.2 列出的功能、Plane 企业版的残留和 Plane 自身的死代码，从路由到依赖的每一层都已删除。
2. 总体设计 1.1 保留的功能仍然完整，其中被共享代码牵连的行为按 7.5 的矩阵核对过。
3. Next.js 兼容垫片已经删除，路由全部是 React Router 的原生写法。
4. 界面上看不到 Plane 的名称和 Logo；内部包名改为 `@nerve/*`；只剩 `zh-CN` 和 `en` 两种语言。
5. 门禁：
   - 类型检查通过；
   - knip 为零并成为门禁；
   - oxlint 不超过新基线；
   - 关键词守卫（7.4）没有未登记的命中；
   - 能正常构建。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 按第 2 节删除功能（每个功能删到 2.3 列出的每一层） | 对接新接口：services、stores 改用生成的客户端（M2 起按领域推进） |
| Next.js 兼容垫片及其配套（`process.env` 注入、dotenv、结尾 `/`、Plane 的旧地址重定向） | 认证的传输方式：CSRF 和会话 Cookie 换成 Bearer 令牌（M2，见 3.6） |
| 品牌替换、包名改为 `@nerve/*` | `@nerve/services` 整包删除（M5；M1 只删其中没人用的部分，见 3.5） |
| 多语言只留中英文 | 后端和数据库的任何改动 |
| 部署遗留、工具链遗留（两份 M0 交接中的事项） | 新的端到端用户故事（M1 不新增故事，S1–S4 保持通过，见第 10 节） |
| 门禁：knip 改为门禁；lint 上限自动核对；前端单元测试接入持续集成；关键词守卫 | 清零 oxlint 警告本身（按 7.3 的计划在 M2–M8 进行） |

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
| oxlint | 合计 995 条，每个包都正好等于它的上限（web 779、editor 75、propel 59、utils 34、ui 32、services 6、hooks 4、i18n 3、constants 2、types 1，其余为 0）。M0 设计 5.2 写的"合计 1005"是加错了，已更正 |
| knip | 未使用文件 131、依赖 19、开发依赖 4、未列出的依赖 2、导出 135、类型 83、枚举成员 4、重复导出 1；配置提示的条数随本地生成过哪些文件而变（1–3 条） |
| 耗时 | 不走缓存的 `make lint-web` 约 30 秒；`make knip` 1.3 秒；不走缓存的 `make build-web` 约 15 秒 |
| 构建产物 | 约 1230 个文件、30.5–30.7 MiB（总字节数，不等于首次加载的开销）。P1 按 7.6 记录分类的基线 |

删除量的估计来自三份只读清点（按功能列出了入口、可整体删除的文件、要修改的保留文件、类型、常量、文案和依赖），又经 Codex 评审复核：
- 内容类功能：约删 500 个文件、改约 200 个文件；
- 账户、平台和死代码：约删 170 个文件，外加 `packages/services` 中的 49 个；
- 多语言：532 个文件。

上面的数字都是估计，实施时以各 Phase spec 的清点为准。

---

## 2. 删除清单

逐项对应前端改动清单第二节。"删""改"是文件数的估计。

### 2.1 内容类功能（P2）

| 功能 | 删 / 改 | 要点 |
|---|---|---|
| 文档页（Pages）及协作编辑 | 约 166 / 约 100 | 已核实：工作项描述、评论、收集箱、草稿用的都是非协作的编辑器；没有保留功能消费 Yjs、`description_binary`、工作项嵌入或页面提及。编辑器内核的处理见 3.1。可删除的依赖：`yjs`、`y-prosemirror`、`y-indexeddb`、`y-protocols`、`@hocuspocus/provider`、`@tiptap/extension-collaboration`、`@tiptap/html`、`buffer`、`@react-pdf/renderer`、`react-pdf-html` |
| 估算 | 53 / 约 80 | 迭代和模块的进度改为只按工作项数计算，见 3.4 |
| 甘特图与时间线 | 73 / 约 40 | 先把保留的关联关系用到的 `REVERSE_RELATIONS` 移出 `gantt-chart.ts`；最后删 `EIssueLayoutTypes.GANTT`，由类型检查列出所有剩下的引用 |
| 自动关闭 | 1 / 6 | "自动化"设置页只留自动归档 |
| 数据分析 | 约 77 / 约 27 | 连同它的旧地址重定向（`:workspaceSlug/analytics`）和侧边栏入口；连同 propel 中只给它和个人主页统计用的 `bar-chart`、`pie-chart`。燃尽图用的 `area-chart` 和 `recharts` 保留；迭代和模块的进度代码不在其中，见 3.4 |
| 导出 | 11 / 约 17 | 和"从未创建过的集成服务"一起删：导出记录列表借用了集成服务 |
| 便签 | 37 / 约 24 | 依赖 `react-masonry-component` |
| 首页快捷链接、首页个性化、自定义主题 | 61 / 约 26 | 首页正文按 3.14 固定下来；已经没有入口的旧首页"仪表盘"（`DashboardStore`，它调用的接口 Plane 自己也没有）一起删；亮色、暗色、跟随系统和高对比度主题与自定义主题无关，保留 |
| 侧边栏"自定义导航" | 约 10 个文件改为固定列表 | 见 3.3 |
| 个人主页的统计和动态 | 14 / 约 9 | 个人主页的新形态见 3.2 |
| "活跃迭代"推广页 | 10 / 约 8 | 只删工作区级的升级推广页；项目迭代列表中的"当前迭代"区块保留 |
| 更新日志（"what's new"） | 随文档页一起删 | 它的类型借用了文档页的 `TPage` |
| 新手导览中的文档页一步 | 1 / 1 | 导览本身（`TourRoot`）保留，"视图"一步改为最后一步 |

### 2.2 账户、平台与死代码（P3）

| 功能 | 删 / 改 | 要点 |
|---|---|---|
| 公开发布 | 7 / 约 23 | 评论的"内部 / 外部"可见范围只为公开发布而存在，一起删，见 3.8 |
| 管理后台入口 | 3 / 约 10 | 包括"实例未完成设置"页。实例请求失败时显示的维护页（`MaintenanceView`）是另一个分支，保留。见 3.7 |
| 第三方登录、验证码登录、找回 / 重置 / 设置密码、登录前的"检查邮箱"步骤 | 31 / 约 28 | 登录页和注册页各剩一个"邮箱 + 密码"表单，模式由路由决定；新手引导中的设置密码一步和安全页中的 `is_password_autoset` 分支一起删；修改密码保留；CSRF 留到 M2，见 3.6 |
| 邮件通知偏好、营销邮件同意 | 5 / 约 9 | 站内通知保留 |
| 修改登录邮箱 | 1 / 约 3 | 靠邮件验证码完成，随邮件发送一起删除，见 3.13 |
| AI 助手 | 11 / 约 25 | 编辑器包里的 AI 部分已在 P2 随文档页收敛（3.1），P3 删应用层的 AI 服务、按钮和实例开关 |
| Unsplash 封面图 | 1 / 4 | 封面选择只留上传 |
| 遥测残留 | 少量 | 已核实应用里没有遥测 SDK，只剩配置字段和文案 |
| 项目邀请 | 0 / 约 4 | web 里没有按邮件邀请进项目的流程，"邀请"弹窗本来就是从工作区成员中直接添加，只改名字和文案。工作区邀请保留，见 3.15 |
| 企业版残留：Epic、团队、工作项类型、计费和升级提示、批量操作、`extended` 空壳 | 65 / 约 200 | 在 P2 删完甘特图、数据分析、文档页之后再删；计费和升级提示放在最后删。<br>Epic 贯穿工作项的 store、hooks、services（`issues\|epics\|work-items` 的地址切换）。删除多态时，描述历史和按编号查找确实使用的 `work-items` 地址保留，不能一律改成 `issues`。团队删除后，普通项目仍在用的共享 `ProjectIssues` 类保留。<br>批量操作与多选见 3.9 |
| 其他有引用的企业空壳 | 约 10 | 这些代码有引用，knip 发现不了，P3 spec 逐个列出：<br>- 工作项模板：`workItemTemplateId`、`isApplyingTemplate`、`handleTemplateChange` 等弹窗上下文字段，它们在 CE 中固定为空或空函数；<br>- 工时记录：操作动态中的 `WORKLOG` 分支和 `is_time_tracking_enabled` 分支、propel 的 worklog 插图；<br>- 定时 / 重复工作项的文案（`recurring_work_items.*`）。<br>它们都不在 v0 范围内 |
| Plane 自身的死代码 | 约 40 / 约 15 | 注释掉的 IndexedDB store、knip 报告的未使用文件和导出、调用 Plane 后端里也不存在的接口的方法（清点按 Plane 的 `urls.py` 找出 59 个，P3 逐个复核后再删）。集成和导入器随导出在 P2 删除；`packages/services` 中没人用的 49 个文件在 P1 删除（3.5） |

### 2.3 删除的做法
- **先写清保留边界**：每个功能的 spec 先列出它牵连到的保留行为（7.5 的矩阵），再删。"类型检查和 knip 都不报错"不能证明保留行为还在：删掉某个保留功能的唯一入口之后，剩下的代码同样能编译通过，也同样没有未使用的导出。
- **每个功能删到这些层**：
  1. 路由、导航和菜单入口：侧边栏、标签页、设置导航、命令面板、快捷键、空状态中的链接；
  2. 组件、store（包括它在根 store 中的接线和 `resetOnSignOut`）、services、hooks；
  3. `packages/types` 中的类型和字段；
  4. 常量和枚举；
  5. 多语言文案；
  6. 不再使用的依赖和图片资源。
- **类型检查和 knip 是删除的向导，不是裁判**：每删一层就运行一次，把它们报出的下一层接着删掉。knip 看不出以下两类，由清点和评审逐处处理：
  - 接口里没人用的成员，比如编辑器 ref 的方法；
  - 有引用但整条链路已经不起作用的代码，比如 3.9 的多选、2.2 的企业空壳。
- **文案最后删**：一个命名空间里的键，在所有代码引用都删掉之后再删。键可以不带命名空间调用，所以核对引用要用"整键字面量 + 模板前缀"的方式，并追到传入 `t()` 的常量和变量，不能只搜命名空间。
- **不留过渡代码**：不写空实现、不加开关、不做"兼容旧字段"的转换。被删功能在保留代码里的分支（例如 `switch` 中的 `EPIC` 分支）直接删除。
- **共享文件由一个任务收口**：根 store、命令面板 store、工作项 store 的分支、编辑器包、路由表、类型和常量的导出文件，每个 Phase 内按任务顺序修改，不并行。一个功能的删除任务同时改完它在这些文件里的所有引用：导入、字段、构造、重置、导出、注册表。
- **接口字段交给后续的 M**：前端不再读的 Plane 接口字段、枚举值和接口地址，由各 Phase 从自己的实际改动中导出完整清单，写进对应 M 的 handoff（3.11）。

---

## 3. 设计裁定

清点和两轮评审中发现了几处总体设计没有写到的地方。以下裁定都在总体设计已经定下的范围内（删掉的功能、差异清单已经砍掉的表），M1 不因此做任何后端工作。其中 3.2、3.6、3.7、3.8、3.11、3.13、3.15 会约束后续 M 的接口或产品能力，3.11 写明了每一项的负责 M 和决策点。

### 3.1 编辑器内核：只去掉协作，保留本地能力
- **现状**：`packages/editor` 的公共内核（`editor-ref.ts`、`use-editor.ts`、`extensions.ts`、`unique-id`、`editor-container.tsx`）带着 Yjs 和 Hocuspocus 的接口。富文本和精简编辑器调用时总是传 `provider: undefined`，功能上没有依赖，但依赖包删不掉。
- **裁定**：文档页删完之后，P2 用一个"编辑器收敛"任务整理内核：
  - **删除**：`getDocument().binary`、`setProviderDocument`、`emitRealTimeUpdate`、`listenToRealTimeUpdate`、`provider` 参数；只给文档页用的标题、文档信息成员；编辑器包里的 AI 菜单和 AI 参数（它们只挂在协作文档编辑器上）。
  - **保留**：
    - `UniqueID` 扩展：所有编辑器都安装它，非协作编辑器也用节点的 id 定位节点。它只去掉 `provider`。
    - `copyMarkdownToClipboard`：描述历史在用。
    - 用户 @提及。
    - 图片和附件的节点。
  - 改动只在编辑器包和它的调用方。

### 3.2 个人主页
- **现状**：侧边栏的"我的工作"、快捷键 `g y`、成员列表中的名字、操作动态中的人名、@提及，全部链接到 `/:workspaceSlug/profile/:userId`，而这个地址正是要删的统计页。页头的用户名、用户卡片上的加入时间和时区也取自统计接口。访客只有统计这一个标签。
- **裁定**：
  - 个人主页只剩"分配给他的 / 他创建的 / 他关注的"三个工作项标签，加上右侧的用户卡片；每个项目的统计数字删掉。
  - 主页地址本身重定向到"分配给他的"标签，所有已有链接不用改。
  - 页头和用户卡片的数据改从工作区成员 store 读取，不再调用统计接口。工作区的包装层已经预取了成员。成员数据里没有时区，卡片只显示头像、名字和加入工作区的时间（`joining_date`），时区删掉。
  - 以下几种状态分开显示，不能渲染成空名字：
    - 成员数据加载中；
    - 成员数据加载失败；
    - 目标用户不是（或已不是）这个工作区的成员。
  - 访客沿用现有的权限规则，看到用户卡片和"无权访问"提示。M1 不扩大、也不永久缩小访客的访问范围；最终规则由 M3 的权限矩阵决定（交接给 M3）。

### 3.3 侧边栏"自定义导航"随首页个性化一起删除
- **现状**：侧边栏的"自定义导航"（固定、排序、隐藏菜单项）读写 Plane 的 `/sidebar-preferences/` 接口，数据在 `workspace_user_preferences` 表。差异清单已经把这张表放在"首页个性化与主题"下砍掉了。
- **裁定**：删除。
  - 侧边栏改为固定列表，顺序取现在的默认顺序。
  - 删除自定义导航弹窗、偏好 store 和对应的接口方法。
  - 以下几项不受影响：项目导航的偏好（`workspace_user_properties`，保留的表）、应用栏的本地偏好、侧边栏的折叠 / 宽度 / 分组展开、收藏。
  - 总体设计 1.2 的"首页个性化"和前端改动清单已补上这一项。

### 3.4 迭代和模块的进度
- **估算删掉之后**：进度和燃尽图只按工作项数计算。以下几样一起删除（约 10 个文件）：
  - "按工作项数 / 按点数"的切换；
  - 只为点数存在的燃尽 / 燃起状态；
  - `distribution-update.ts` 中的点数计算。
- **必须保留并核对的行为**：
  - 工作项创建、完成、取消、重新打开之后，数量和进度的更新（乐观更新）；
  - 当前迭代和已结束迭代的快照；
  - 模块的进度；
  - 已归档对象的进度。
  - 数量计算用单元测试覆盖（7.5）。
- **命名**：保留的迭代、模块进度代码目前叫 "analytics"（`analytics-sidebar`、`fetchActiveCycleAnalytics` 等），和要删的数据分析重名，在 P2 中改名为 progress。它调用的 Plane 接口地址本身带 `analytics`（如 `/cycles/${cycleId}/analytics?type=`），这是 Plane 的接口，M1 不改。关键词守卫只对这几个地址字面量开精确的例外，到 M6 换成新接口为止（交接给 M6）。

### 3.5 `packages/services` 在 P1 修剪
- web 只用到其中三样：令牌服务（`APITokenService`）、地址规范化（`normalizeAPIRequestURL`）、上传文件的元数据工具。其余 49 个文件是其他应用或已删功能的服务副本，没有任何调用方；Codex 已在副本中试删并通过类型检查。
- **裁定**：P1 删除这 49 个文件，包本身按总体设计 7.1 留到 M5 删除。
  - 其中几个文件引用了 P2 要删的类型，留到后面反而会让 P2 编译失败。
  - 保留的地址工具已有 13 个单元测试（`url.test.ts`），它们随 7.5 的前端单元测试一起进入持续集成。

### 3.6 认证：M1 只删界面上的功能，传输方式留给 M2
- **现状**：登录、注册、退出、修改密码、新手引导中的设置密码，都通过 Django 会话 + CSRF 令牌提交：原生表单 POST 带 `csrfmiddlewaretoken`（提交到 `/auth/sign-in/`、`/auth/sign-up/`，不在 `/api/` 下），或者带 `X-CSRFTOKEN` 请求头。
- **裁定**：
  - M1 删除第三方登录、验证码登录、找回 / 重置 / 设置密码页面和"检查邮箱"步骤；登录页和注册页各剩一个"邮箱 + 密码"表单，模式由路由决定。
  - CSRF 和会话提交方式是认证传输层的一部分，在 M2 和替代它的令牌管理器一起删除（总体设计 9.2 中 M2 负责"前端令牌管理器和登录页"）。M1 先删掉它，只会让这几处调用停在一种 M2 马上又要重写的中间状态。M2 必须删掉 Cookie 和 CSRF，不能继续延期。
  - P3 的运行时核对要同时拦截 `**/auth/**` 和 `**/api/**`，检查表单提交的字段、CSRF 令牌、`next_path` 和结果跳转（7.5）。
  - 总体设计 7.3 和前端改动清单已同步；交接给 M2。

### 3.7 实例配置与启动
`InstanceWrapper` 在启动时读取 Plane 的实例配置。其中一部分开关决定认证页显示什么，另一部分被保留功能使用。

| 字段或分支 | 处理 |
|---|---|
| 第三方登录、验证码、SMTP、AI、Unsplash、遥测等开关；`is_setup_done` 和"实例未完成设置"页 | 随对应功能删除 |
| `enable_signup`（是否开放注册） | 保留；M2 的实例配置接口决定最终名称 |
| `is_workspace_creation_disabled`（是否禁止普通用户创建工作区；创建工作区页、新手引导、工作区菜单、命令面板在用） | 保留；M2 决定最终名称和策略 |
| `file_size_limit`（上传大小上限；`use-file-size` 在用） | 保留；M2 或 M5 决定最终名称和策略 |
| 实例请求失败时的维护页（`MaintenanceView`） | 保留：这是启动失败时的展示，和"实例未完成设置"是两个分支 |

实例配置接口的最终字段在 M2 定义，前端届时改用生成的类型。P3 从实际改动中导出完整的"删除 / 保留 / 等待重新定义"清单，交接给 M2。

### 3.8 评论的可见范围
评论的"内部 / 外部"（`access`）只在项目公开发布时有意义：Plane 的公开页按 `EXTERNAL` 过滤评论。它不是评论的权限，回复、表情回应、署名都不受影响。**裁定**：界面和类型中删除；`issue_comments.access` 列是否保留，交接给建表的 M4（M0 设计 7.1：列级别的细节由建表的 M 补全）。

### 3.9 批量操作与工作项多选整条删除
- **现状**：批量操作是 Plane 的付费功能，调用的接口 Plane 社区版也没有。社区版中 `useBulkOperationStatus()` 恒为 `false`。列表、表格、甘特图把它变成"禁用"，却仍然把多选的状态和 `selectionHelpers` 一层层往下传。所以这条链路有引用，但不起作用，knip 也不会把它判为未使用。
- **裁定**：P3 用一个任务删除整条链路：
  - 批量操作；
  - 工作项的 `MultipleSelectStore`；
  - `useMultipleSelect` 系列 hooks；
  - 组头和行上的选择框；
  - `selectionHelpers` 的逐层透传；
  - 根 store 中的构造和退出重置。
- **不在其中**：
  - 站内通知的"全部已读"、预览（使用独立的 `WorkspaceNotificationStore`）；
  - 表单中的多选下拉框。

  它们名字相似，但是真实在用的交互，保留。

### 3.10 `@makeplane/propel` 保留为第三方依赖
- 它是 Plane 发布在 npm 上的设计系统（0.3.0，AGPL-3.0-only），提供设计令牌（样式变量）和 523 处图标导入，是 web、editor、propel、tailwind-config、ui、utils 的依赖，不是工作区里的包。
- **裁定**：保留，照常锁定版本。
  - 它的名字只出现在导入路径里，界面上看不到；许可证与本仓库一致。
  - 包名重命名（第 5 节）只针对工作区里的 `@plane/*`；关键词守卫对 `@makeplane/propel` 这个包名开精确的例外。
  - 其中带 Plane 品牌的图标只有 AI 相关的几个，随 AI 助手一起不再使用。
- **来源记录**（P5）：记下锁定版本的 tarball 完整性哈希、许可证，以及能找到的对应源码位置。npm 元数据里没有对应的提交号，所以"以后并入源码"只是一个需要先确认源码版本的备选方案，不是随时可以无成本执行的退路。
- 这一项已向项目负责人报备。

### 3.11 交给后续 M 的接口字段（种子清单）
下表是清点和评审得到的种子清单，不是全部。P2、P3 各自从实际改动中导出完整的清单（字段、枚举值、接口地址、默认行为），逐项对应到负责的 M 和差异清单，写进该 M 的 handoff。

| 接收的 M | 字段与事项 |
|---|---|
| M2 | `last_login_medium`、`is_password_autoset`、`is_email_verified`、`has_marketing_email_consent`、计费和试用字段；主题只剩 `{theme}`；CSRF 和会话（3.6）；实例配置的保留字段（3.7）；是否提供修改登录邮箱，最终能力由项目负责人决定（3.13）；差异清单已删除的 `users.is_bot`、`bot_type`、`api_tokens.user_type` 在前端的读取方（成员筛选、评论 / 通知 / 链接的署名、令牌类型），按领域清除 |
| M3 | 项目的 `page_view`、`close_in`、`default_state`、`estimate_id`、`is_issue_type_enabled`、`is_time_tracking_enabled`（差异清单已删除）；`anchor`；侧边栏偏好接口不再需要（3.3）；访客能否查看个人主页（3.2）；工作区邀请链接是否保留（3.15）；工作区保留地址名单 |
| M4 | `estimate_point`、`type_id`、`is_epic`、`description_binary`（包括描述历史和快照中的）、评论的 `access`（3.8）、`POST /issue-dates/`；搜索结果不再有 `page`；操作动态中估算、类型、Epic 类记录的取值约定；描述历史仍使用 `work-items` 地址（2.2） |
| M5 | 资源类型 `PAGE_DESCRIPTION` 和 `file_assets.page_id`；`file_size_limit` 的策略（3.7）；`@nerve/services` 删除前，核对地址工具的最后一批使用方 |
| M6 | 迭代和模块的进度不带 `points`、`estimate_distribution` 和各类 `*_estimate_points`；快照中的分布；关键词守卫中 `analytics` 地址的例外随进度接口的替换一起消失（3.4） |
| M7 | 收藏和最近访问不再有 `page` 类型；通知实体中已删功能的类型；首页的最近访问（3.14） |

**工作区保留地址名单**：`packages/constants/src/workspace.ts` 的 `RESTRICTED_URLS` 防止工作区地址和应用的一级路径冲突。各 Phase 删掉自己功能对应的条目，例如：
- `god-mode`、`epics`、`upgrade`、`billing`（P3）；
- `integrations`、`importers`、`pages`（P2）；
- `plane-pro`、`plane-ultimate`、`plane-enterprise`（P3，计费残留）。

还在使用的一级路径不能从名单里删。M3 让后端的保留名单和删减后的前端一致。

### 3.12 Plane 的旧地址重定向
- `routes/core.ts` 末尾有 11 条"为兼容旧地址"的重定向（`/sign-in`、`/login`、`/accounts/sign-up` 等），是 Plane 的历史包袱，要删除。
- 但有些现用入口仍在依赖它们：
  - 命令面板的"账户设置"跳到 `/:workspaceSlug/settings/account`，靠重定向才到达 `/settings/profile/general`；
  - 侧边栏的数据分析入口 `/analytics/` 靠重定向才到达 overview；
  - 两处链接指向 `/sign-in`。

  Codex 用真实的路由匹配复现：直接删除之后，这些入口会落到"页面不存在"。
- **裁定**：
  - 数据分析的重定向随数据分析在 P2 删除。
  - 其余 10 条放到 P4，和所有内部跳转目标一起处理：先把每个仍然依赖重定向的入口改为正式地址，再在同一个提交里删除重定向。P4 用路由匹配核对所有内部跳转目标都能命中真实的路由（4.2）。

### 3.13 修改登录邮箱
- **现状**：个人资料中的"修改邮箱"弹窗先向新邮箱发送验证码，输入验证码后才生效。
- **裁定**：随"所有邮件发送"（总体设计 1.2）一起删除旧流程。删除旧流程不等于永久不提供改邮箱。v0 最终提供哪一种，由项目负责人在 M2 设计时决定（交接给 M2，M2 设计中要把这一项作为决策点提出来）：
  - 输入当前密码后自助修改；
  - 只能由管理员用命令行修改；
  - v0 暂不提供。

  这和总体设计第 10 节"没有邮件服务时账户如何找回"是同一类问题。M2 之前前端还连不上认证接口，这段时间里没有用户受影响。

### 3.14 首页的内容
- **现状**：首页正文完全由首页小组件 store（`HomeStore`，它要删）驱动。唯一渲染"最近访问"的地方是小组件列表（`DashboardWidgets`），而最近访问是总体设计 1.1 保留的功能。如果顺着删小组件时的编译报错清理，最近访问会跟着变成"没有引用"的代码，被合法地清掉。
- **裁定**：首页固定显示以下内容，并直接挂载这些组件：
  - 问候语；
  - 没有项目时的空状态；
  - 最近访问（项目和工作项）；
  - 工作项预览需要的 peek。

  以下内容删除：小组件 store、"管理小组件"面板、首页个性化的接口调用。新手导览按 2.1 保留。P2 的运行时核对要覆盖"有最近访问"和"空状态"两种情况；最近访问的接口由 M7 换成新接口。

### 3.15 工作区邀请链接
- **现状**：除了"系统内接受邀请"（总体设计 1.1 保留），现有代码还有一个按令牌接受邀请的页面（`/workspace-invitations`），以及成员设置里"复制邀请链接"的入口。
- **裁定**：M1 不删。它不属于 1.2 的任何一项，也不是只为发邮件存在的代码。v0 是否同时保留"系统内接受"和"链接接受"，由 M3 设计邀请接口时一起决定（交接给 M3），届时删掉不用的那条路径。

---

## 4. 去掉 Next.js 兼容层（P4）

### 4.1 要删的东西
- **垫片**：`app/compat/next/*`、`app/types/next-link.d.ts` 和 `next-navigation.d.ts`、`vite.config.ts` 中的两个别名、包装层 `use-app-router.tsx`。其中没人用的 `image.tsx`、`script.tsx`，以及 `next/script` 的别名和 `next-script.d.ts`，是死代码，P1 先删。
- **强制结尾 `/`**：垫片里的 `ensureTrailingSlash`，以及 `app/layout.tsx` 中返回 308 的 `clientMiddleware`。
- **地址的约定**（替代"统一不带 `/`"的简单说法）：
  - 应用内部生成的界面地址一律不带结尾 `/`。只取消自动补 `/` 并不能做到这一点：代码里仍有很多带 `/` 的跳转目标，例如顶部通知的 `/${workspaceSlug}/notifications/`，要逐个改掉。
  - 判断"当前是哪个菜单项"的逻辑，对带和不带结尾 `/` 的地址给出相同的结果。
  - 这里说的是界面路由。Plane 接口地址的结尾 `/`（`ensureAPITrailingSlash`，为 Django 而加）不在其中，随各领域对接新接口时退出；签名上传的地址不能被加上 `/`。
- **Plane 的旧地址重定向**：除数据分析以外的 10 条（3.12）。
- **`process.env` 的注入**：
  - 删除：`vite.config.ts` 顶部的 dotenv 加载和 `define: { "process.env": … }`，`web/apps/web/.env.example`，`turbo.json` 中的 `VITE_*` 环境变量和 `build.inputs` 中的 `.env*`。
  - 代码中读取的每个 `VITE_*` 变量：
    - 属于已删功能的，已随功能删除；
    - `VITE_API_BASE_URL` 删除，接口一律用相对路径（同源部署，见 M2 的 [M0-P5-frontend-api-notes](../M2-auth/handoffs/M0-P5-frontend-api-notes.md)），`APIService` 不再接收基础地址参数；
    - `VITE_SUPPORT_EMAIL` 删除，文案改为"联系管理员"。
  - i18n 包中的 `process.env.NODE_ENV` 判断整体改写为 `import.meta.env.DEV`，并给这个包加上 Vite 的环境类型：当前的包里没有 `ImportMeta.env` 类型，直接替换会编译失败。
  - 删除顺序受一个约束：各个包用 tsdown 构建出的 `dist/` 原样保留 `process.env.X`，靠 `define` 才不报错。所以先删掉所有读取，再删 `define`。
- **Next.js 的其他遗留**：`app/(all)/layout.preload.tsx`（内容全被注释掉）、`typescript-config/nextjs.json`、各处忽略列表中的 `.next`。

### 4.2 改写方式
- **机械部分用一次性的 ts-morph 脚本**。脚本不进仓库，在提交说明和 review 中说明。它做以下替换：
  - `next/link` 的 `Link href` → React Router 的 `Link to`。代码里没有用到 `prefetch`、`scroll`、`shallow`、`replace`。
  - `usePathname()` → `useLocation().pathname`；`useSearchParams()` → `const [searchParams] = useSearchParams()`；`useParams` 改从 `react-router` 导入。
  - `router.push(x)` → `navigate(x)`，`router.replace(x)` → `navigate(x, { replace: true })`，`router.back()` → `navigate(-1)`。
  - 命令面板上下文中的 `router` 字段改为 `navigate: NavigateFunction`。
- **手工部分**：
  - **路由参数的类型**：垫片的手写类型说参数总是 `string`，React Router 的真实类型是 `string | undefined`。换过去之后有 152 处类型错误（68 个文件，其中一部分在 P2、P3 删掉的功能里，P4 开始时重新统计）。每处都按真实情况处理：
    - 路由组件用 `./+types/*` 的 `params`（参数一定存在）；
    - 共享组件里用守卫。

    不用非空断言或 `as string` 批量压掉错误。
  - **渲染时跳转**：`AuthenticationWrapper` 在组件渲染过程中调用跳转（现在 8 处；P3 删掉设置密码的分支之后是 6 处），只因为垫片用 `setTimeout` 推迟了跳转才能工作；React Router 会丢掉首次渲染中的跳转。改为渲染 `<Navigate replace />`。这个包装层在 M2 随令牌管理器重写，M1 只做让它在原生路由下正确工作的最小改动。
  - **当前菜单项的判断**：逐个语义列出所有依赖地址形式的判断——相等比较、`includes`、`startsWith`、正则——范围包括侧边栏、设置导航、个人主页标签、应用栏、顶部通知。一律改用 React Router 的匹配（`NavLink` 的 `isActive`、`matchPath`）。没有调用方的 `highlight` 函数删除。
- **验证**：
  - 类型检查、knip、构建、关键词守卫（Next.js 垫片的规则）通过。
  - **内部跳转目标的路由匹配**：用真实的路由表（`matchRoutes`）核对所有内部跳转目标都命中真实的路由，而不是"页面不存在"。这项检查只依赖路由表，适合作为单元测试长期保留（7.5）。
  - Vite 构建时会替换 `process.env`，开发服务器对预构建的依赖不会。删掉 `define` 之后，用 `make web-dev` 打开首页，确认控制台没有 `process is not defined`。
  - **运行时核对**（7.5 的 P4 一行）。M1 的后端还不提供 Plane 的接口，所以用拦截接口的临时脚本核对以下内容：
    - 未登录、未完成引导、已登录已引导、没有工作区几种情况的跳转；
    - `next_path`、后退、替换跳转；
    - 个人主页、设置、应用栏、侧边栏、顶部通知的当前项在带和不带结尾 `/` 时都正确；
    - 查询参数和 `#` 片段不丢失；
    - 控制台没有"navigate() 应在 useEffect 中调用"的警告。

    P4 还要重跑 P2、P3 中与路由有关的核对场景。

---

## 5. 品牌与包名（P5）

| 类别 | 处理 |
|---|---|
| 版权声明（2926 个文件） | **保留**（总体设计 2.3） |
| Logo、网站图标、应用清单、加载动画 | 换成 Nerve 的图形；propel 中的 `PlaneLogo`、`PlaneLockup`、`PlaneWordmark` 等改为 Nerve 的对应组件 |
| 页面标题、`SITE_NAME` 等元数据 | 改为 Nerve |
| 中英文文案中的 "Plane"（各约 108 行） | 改为 Nerve；讲的是 Plane 公司或 Plane 服务的句子（如"联系 Plane 支持"）按含义改写或随功能删除 |
| 硬编码的文案（约 70 行） | 同上 |
| Plane 的网址（plane.so、文档、论坛、状态页、GitHub、社交账号） | 帮助菜单、命令面板帮助命令、错误页中的这类链接删除；需要外部链接的地方指向 Nerve 的仓库；测试样例中的 `api.plane.so` 这类地址换成中性的域名 |
| 代码标识符、本地存储键（`plane_tab_prefs` 等）、组件 `displayName`（`plane-ui-*`） | 改为 Nerve 的名字 |
| 注释 `// plane imports` 等（约 1200 行） | 随包名一起改为 `// nerve imports` |
| `@makeplane/propel` | 保留（3.10） |

- **Logo**：Nerve 目前没有设计稿。P5 做一套简单的矢量图形（图标、横版标志、网站图标的各个尺寸）。之后可以直接替换文件，不需要改代码。
- **新文件的版权**：Nerve 新写的文件和新画的图形按实际来源标注 Nerve 的版权和 SPDX 标识。改写过的 Plane 文件保留原有的版权声明。不能只把 Plane 的 Logo 改个名字当作新图形。放不下文件头的资源，在同目录的来源说明里登记。
- **二进制品牌资源**：图片中的 Logo 搜不到关键词，要按使用位置逐个替换，并在实际页面上目视核对。
- **包名 `@plane/*` → `@nerve/*`**：单独一个纯机械的提交，放在删除和 P4 之后（涉及的文件最少），之后再做品牌语义的改动。
  - 用 `sed 's#@plane/#@nerve/#g'` 替换全部出现（不会误伤 `@makeplane/`），再执行 `pnpm install` 重写锁文件。这个提交里不升级任何依赖。
  - 完整性核对：
    - `git grep -n "@plane/"` 在 `docs/` 以外没有结果；
    - 锁文件相对重命名之前的提交只有包名的变化，即 `git show <重命名之前的提交>:pnpm-lock.yaml | sed 's#@plane/#@nerve/#g' | diff - pnpm-lock.yaml` 为空；
    - `pnpm -r ls --depth -1` 只列出根包 `nerve`、应用 `web` 和 `@nerve/*`；
    - `knip.jsonc` 中的工作区路径是目录，不用改（P6 交接）。
- **S2 仍不断言页面文字**：M2 之前页面还是启动错误页（P6 交接）。

---

## 6. 多语言（P1）
- **删除语言**：删除 19 种语言的目录（532 个文件）；`SUPPORTED_LANGUAGES` 只剩两项，`TLanguage` 改为 `"en" | "zh-CN"`。
- **不再支持的语言**：服务端或本地存储里存着不再支持的语言时，回退到 `en`，语言选择框显示英文。
- **翻译键的生成**：`keys.generated.ts` 生成出来之后没有任何地方使用（`t` 的参数类型是 `string`，还有 187 处用变量调用）。删除生成脚本、`generate:types` 和 `knip.jsonc` 中 i18n 的 `ignoreUnresolved`（P6 交接）。
- **中英文键一致**：已有的 `sync-check` 脚本只在"缺键"时失败，改为双向严格相等：
  - `en` 和 `zh-CN` 两个目录都必须存在；
  - 键集合完全相同，多键、缺键都失败；
  - 保留跨命名空间冲突的检查。

  它接进 `make lint-web`，作为门禁，脚本和语言文件都进入任务的输入。它只能保证中英文同步；两边一起删错的情况要靠"整键 + 模板前缀"的引用核对（2.3）。
- **企业版的整个命名空间**：`automation`、`editor`、`workflow`、`template`、`work-item-type`、`tour`、`update`、`wiki` 这几个命名空间（`editor` 由 P1 的核对补上），以及 `workspace_settings.settings.applications.*`（161 个叶子键），代码中一处引用都没有，是死文案，连同命名空间的登记在 P1 删除。已核对以下几点：
  - `applications.*` 与保留的个人访问令牌（`api_tokens.*`）、Webhook（`webhooks.*`）文案互不相干，删除范围不扩大到开发者设置分类或整个 `workspace-settings.json`；
  - `tour` 命名空间是企业版产品导览的文案，与保留的新手导览组件 `TourRoot` 不是同一条链路。

  只在 P2、P3 删完代码之后才失去引用的键，由那两个 Phase 删除；收尾时再核对一遍。
- **不在本 M 范围**：各处写死的 `"en-US"`（日期问候、时区换算）保持不变。

---

## 7. 质量门禁与验收

### 7.1 lint 上限自动核对（P1）
- **现状**：每个包的上限只防止警告变多；警告减少后"同一个提交里调低上限"只靠评审把关（M0 设计 5.2）。M1 会删掉大量代码，几乎每个任务都会让警告变少。
- **裁定**：P1 加一个自动检查。每个包的实际警告数必须**等于**上限，多了或少了都失败：
  - 警告比上限多：提示"修掉新增的警告，上限只能调低"，不给出调高后的数值（M0 设计 5.2：只降不升）。
  - 警告比上限少：提示把上限调低到实测值。
- **做法的约束**（具体实现在 P1 spec 中通过原型确定）：
  - 警告数从 oxlint 的 JSON 输出中按严重程度统计，运行目录、配置和忽略规则与 `check:lint` 完全相同；
  - 先保留"有 error 或进程失败就失败"的语义，再比较数值；
  - 上限只从各包的 `package.json` 一处读取；
  - "运行 oxlint + 比较上限"是同一个 turbo 任务，检查脚本进入任务的输入；
  - 原型至少证明以下几种情况下缓存行为正确：冷缓存 / 热缓存、少一条警告、多一条警告、改上限、改检查脚本、改依赖的包。
- **局限**：数值相等不能证明"没有新增一条、同时删掉另一条"，所以它不能替代评审对新增警告的检查。
- `@nerve/api-client`、`@nerve/e2e` 的上限保持 0（P6 交接）。

### 7.2 knip 改为门禁（P3 结束时）
- P3 删完死代码、knip 报告为零之后，`make knip` 去掉 `--no-exit-code`，成为持续集成的门禁；P4、P5 和之后的所有 M 都在门禁下进行。P1、P2 的 knip 仍是报告，报告中剩下的每一项都能对应到后续 Phase。
- **配置提示**：配置提示的条数随本地生成过哪些文件而变（P6 交接），根因是 knip 有时在生成之前运行、有时在生成之后运行。
  - P1 修好 `tailwind-config` 的入口（按它真实的 `index.css` 入口和 `exports` 写，不为了消除提示造一个空的 JS 文件），删掉 i18n 的生成步骤和它的 `ignoreUnresolved`。
  - P3 让 `make knip` 先运行 web 的 react-router 类型生成（`+types/` 由它生成，几秒以内），这样 knip 总是在同一种状态下运行。随后删掉 web 的 `ignoreUnresolved`，并打开 `--treat-config-hints-as-errors`，配置本身过时也会被门禁发现。Codex 已在副本中验证：类型生成之后去掉这一项，不会出现无法解析的导入。

### 7.3 oxlint 清零计划
- M1 结束时，按包、按规则重新列出警告数（由 7.1 保证就是上限），写进 M1 的收尾 review。
- **计划**（写进收尾 review，由后续每个 M 执行）：
  1. **谁改谁清**：后续 M 修改的文件，在该 M 结束时不再有警告。M2 起前端要逐个领域改写 stores 和组件，大部分警告会随之清掉。
  2. **按规则集中清理**：没有被改到的文件按规则分批清理，每个 M 至少清掉一类，并在该 M 的 review 中记录。优先清可以机械修复的规则，例如 `no-shadow`、`promise/always-return`、`no-unneeded-ternary`。
  3. **目标**：M8 发布前全部清零，之后 oxlint 警告即报错（总体设计 7.6 的原始要求）。

### 7.4 关键词守卫
总体设计 7.4 要求关键词搜索没有结果。M1 把这项验收做成一个进仓库的检查工具，而不是每个 Phase 手工跑 `git grep`。原因有三：
- 手工命令混用了两种正则引擎。内容用 git 的 PCRE，文件名用 `grep -E`，而后者不支持 `(?<!…)`，报错退出（exit 2），很容易被当成"零命中"；
- git 的 PCRE 是可选的编译功能，不同机器不一定一样；
- 按文件豁免太粗：根 store、共享类型、翻译 JSON 同时属于好几个 Phase。

**工具**（P1 实现，即 `tools/keywords.mjs` 和规则文件 `tools/keywords.json`；只用 Node，不加依赖。`make lint-web` 的第一步运行它，所以本地和持续集成的 `web` 任务从 P1 起都经过它）：
- **文件范围**：来自 `git ls-files`，包括跟踪的文件，以及未跟踪、未忽略的文件：本地新加的文件提交之前就受检查。跳过二进制文件。除了 `web/`，还包括根目录的 `pnpm-lock.yaml`、`pnpm-workspace.yaml`、`turbo.json`，用来检查旧依赖和旧环境变量。
- **规则**：内容规则和文件名规则分开定义。
  - 正则用 JavaScript 的 `RegExp`，显式写 flags，在 macOS 和 Linux 上行为一致；
  - 规则文件本身带正例和反例样本，工具先用样本自检。内容规则的两个正则都要有样本：选文件的那个（`files`）和查文本的那个（`content`）。只给后者写样本，等于把"这条规则读哪些文件"留在自检之外，以后有人把范围改窄，样本照样全过；
  - 任何正则或读取错误都以非零退出，绝不当成零命中。
- **例外**：必须精确，包括规则编号、路径、命中的文本或符号、理由和到期的 Phase 或 M。一条例外默认只覆盖一处命中；同一个文件里同一段原文出现多处时，例外里写明处数（`count`），实际处数必须与它相等，多了少了都失败——否则一条例外会把后来新增的同样一处也顺带盖住。同一处只能登记一条例外，重复登记报错。一条例外不再命中任何内容时（例如代码已经删了），工具报错，提醒删掉它。到期也由工具管：规则文件里写着当前的 Phase（`phase`），例外的 `until` 到了或早于它，工具就报错——也就是说，到期的那个 Phase 一开始，这条例外就必须已经删掉，而不是靠人记得。
- **规则随删除加入**：每个 Phase 在删完某个功能的同一个提交里加入它的规则，所以已加入的规则永远是"必须为零"。M1 结束后这些规则继续守在持续集成里，防止删掉的功能重新长回来（总体设计 7.6）。
- **二进制品牌资源**：守卫看不到图片里的 Logo，这部分按第 5 节目视核对。

**种子规则**：下表来自清点和 Codex 的实测命中，各 Phase 的 spec 按实际命中收窄。只能改得更精确，不能为了通过而扩大例外。

| 功能（负责的 Phase） | 规则要点 | 注意 |
|---|---|---|
| 文档页、协作（P2） | `page_view`、`description_binary`、`yjs`、`y-prosemirror`、`y-indexeddb`、`y-protocols`、`hocuspocus`、`Collaborative`、`extension-collaboration`、`pageId`/`page_id`（整词）、`WorkItemEmbed`、`issue-embed`、`LIVE_BASE`、`VITE_LIVE`、`comlink`、`react-pdf`、`@tiptap/html`、`wiki`（不区分大小写） | 不搜单独的 `page`、`pages`；`wiki` 对 `packages/utils/src/tlds.ts` 中的顶级域名数据开例外；导览中的文档页一步和它的图片另外核对 |
| 估算（P2） | `estimat`（不区分大小写）、`估算` | |
| 甘特图、时间线（P2） | `gantt`（不区分大小写）、`time_?line`（排除 `view_` 前缀）、`issue-dates` | 保留的 `useTimeLineRelationOptions` 在 P2 改名 |
| 自动关闭（P2） | `close_in`、`auto[-_ ]?close`（不区分大小写）、`default_state` | |
| 数据分析（P2） | `analytics`（不区分大小写） | 依赖 3.4 的改名；只对 `tlds.ts` 和 3.4 所说的几个 Plane 进度接口地址开例外（到 M6） |
| 导出（P2） | `exporter`（不区分大小写）、`settings/exports`、`export-issues`、`EXPORTERS_LIST`、`IExportData` | 不搜单独的 `export` |
| 便签（P2） | `Sticky`、`STICKY`、`[Ss]tickies`、`STICKIES`、`sticky[-_.:]`（区分大小写） | 不搜 `stick`（不区分大小写时会命中 `AxisTick`），也不搜单独的 `sticky`（CSS） |
| 首页、主题（P2） | `quick_links`、`quick-links`、`QuickLink`、`HomeWidget`、`manage_widgets`、`home-preferences`、`CustomTheme`、`customize_your_theme`、`palette-generator`、`DashboardStore`、`useDashboard` | 不搜单独的 `widget`、`dashboard`、`custom` |
| 侧边栏自定义（P2） | `sidebar-preferences`、`SidebarNavigation`、`customize-navigation`、`navigationPreferences` | 项目导航偏好是保留的，规则按确切的符号写 |
| 个人主页统计（P2） | `user-stats`、`user-profile/`、`user-activity/`、`IUserProfileData`、`profile.stats`、`ProfileSidebar`、`your_work_by_` | `user-activity` 带结尾 `/`（propel 有同名图标） |
| 活跃迭代推广（P2） | `active-cycles`、`active_cycles`、`WorkspaceActiveCycles`、`workspaceActiveCycles`（整词） | 不搜 `active_cycle`、`ActiveCycle`（保留的当前迭代区块） |
| 更新日志（P2） | `changelog`、`product-updates`、`what.s new`（不区分大小写） | |
| 公开发布（P3） | `SPACE_BASE`、`SPACE_APP`、`SITES_URL`、`deploy-boards`、`PublishProject`、`ProjectPublish`、`publish_project`；`anchor` 相关的规则只作用于 `web/apps/web` 和 `web/packages/types` | 不搜单独的 `publish`（草稿"发布为工作项"保留）；编辑器里的 `anchor`（选区、链接）是保留代码 |
| 管理后台（P3） | `god[-_]mode`、`GOD_MODE`、`ADMIN_BASE`、`instance-admin`、`InstanceNotReady`、`is_setup_done` | |
| 认证（P3） | `oauth`（不区分大小写）、`unique[-_]code`、`UniqueCode`、`magic[-_](generate\|code\|login)`、`MAGIC_`、`forgot[-_]password`、`reset[-_]password`、`set[-_]password`、`SET_PASSWORD`、`email-check`、`emailCheck`、`gitea`、`gitlab`、`is_google_enabled` | 不搜单独的 `magic`（propel 图标名 `magic_exchange`）；CSRF 留到 M2（3.6），不在本清单 |
| 邮件（P3） | `email_notification`、`marketing_email`、`EmailNotification`、`change_email_modal`、`ChangeEmailModal` | |
| AI、Unsplash、遥测（P3） | AI 的确切符号（`PlaneAi`、`aiHandler`、`AI_` 常量、`rephrase`、`gpt`、`llm` 等，大小写敏感的部分单独成规则）、`unsplash`、`posthog`、`telemetry`、`sentry`、`intercom`（不区分大小写） | AI 的具体规则在 P3 spec 中按命中结果确定 |
| 项目邀请（P3） | `ProjectInvitation`、`project.*invitation` | 精确例外：`joinProject` 调用的 Plane 地址 `/projects/invitations/`，到 M3 |
| 企业版残留（P3） | Epic：`(?<![dD])epic`、`Epic`、`EPIC`；团队：`teamspace`、`TEAM_VIEW`、`TEAM_PROJECT`、`TEAM_SPACE`，以及 `EIssuesStoreType.TEAM` 这类确切的枚举值；类型：`type_id`、`issueTypeId`、`IssueTypeSwitcher`、`work_item_type`；计费：`upgrade`、`billing`（不区分大小写）、`EProductSubscription`、`ProIcon`、`contact_sales`、`plane-pro`、`LockedComponent`；批量操作：`bulk[-_]?operation`、`MultipleSelectStore`、`useMultipleSelect`、`selectionHelpers`；模板、工时、重复：`workItemTemplateId`、`isApplyingTemplate`、`handleTemplateChange`、`WORKLOG`、`is_time_tracking_enabled`、`worklog`、`recurring_work_items`；空壳：按 EE 空壳的确切符号列出（`useExtendedEditorConfig`、`ExtendedBasePage`、`auth-ee`、`extendedRoutes`、`useAdditionalEditorMention`、`AdditionalSlashCommand`、`CE/EE`、`no-op in CE` 等） | 不用 `TExtended`、`TAdditional` 这类前缀：它们命中保留的侧边栏组件（`TExtendedSidebarItemProps`）和筛选组件（`TAdditionalWorkItemFiltersProps`）；`upgrade` 在依赖名和注释中的命中逐条核对；普通英文单词 `team` 不搜 |
| 死代码（P3） | `integration`、`importer`、`jira`、`slack`、`github-repository`、`app-installation`、`indexeddb`（不区分大小写） | Webhook 保留，不会命中这些词 |
| Next.js 垫片（P4） | `next/(link\|navigation\|script\|image)`、`useAppRouter`、`ensureTrailingSlash`、`compat/next`、`process\.env` | Node 端的工具脚本如果有合法的 `process.env` 读取，按执行环境开精确例外，不做机械替换 |
| 多语言（P1） | 文件名规则：`web/packages/i18n/src/locales/` 下只允许 `en/` 和 `zh-CN/` | |
| 部署遗留、死文件（P1） | 文件名规则：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`sw.js`、`workbox-`、`compat/next/(image\|script)` | |
| 品牌（P5） | `plane`（不区分大小写） | 精确例外：版权声明行（`Copyright (c) 2023-present Plane Software, Inc.`）、`@makeplane/propel` 包名；airplane、planet 这类普通单词按确切的词开例外；包名规则：`@plane/` |

- **按 Phase 验收**：一个 Phase 加入的规则，在它合并时必须没有未登记的命中。如果某处命中位于后续 Phase 才删除的文件或符号（例如 P3 才删的计费页面里也出现了 `estimat`），就登记一条精确到符号、到期为那个 Phase 的例外，不能整个文件豁免。到期的 Phase 合并时，这些例外必须已经消失（过期的例外会让工具报错）。
- **收尾**：只允许剩下明确跨 M 的例外，例如 `analytics` 进度地址到 M6、`joinProject` 的地址到 M3。

### 7.5 保留行为的验收
M1 不以新的后端为完成前提，但"页面停在启动错误页"不能证明保留功能没被删坏。类型检查、knip 和关键词守卫都发现不了以下两类问题：
- 修改共享代码后，保留功能的行为变了；
- 保留功能的唯一入口被一起删掉。

所以每个 Phase 按下表核对，并把核对的手段固定下来。

| Phase | 最少要核对的行为 | 手段 |
|---|---|---|
| P1 | 不支持的语言回退到 `en`；中英文键集合相同；预渲染和首次客户端渲染的结构一致（没有 #418）；修改 `web/packages/*` 的源码能在 `make web-dev` 中生效 | 键比较、构建；浏览器核对；在开发服务器中改一个包里的符号，确认生效后恢复 |
| P2 首页、个人主页、侧边栏 | 最近访问的"有数据"和"空状态"；没有文档页的入口；个人主页地址重定向到"分配给他的"；成员、访客、成员不存在、成员接口失败几种状态分别正确显示；侧边栏为固定列表 | 临时核对脚本 |
| P2 编辑器 | 工作项富文本的输入、保存、撤销 / 重做；评论（精简编辑器）和 @成员；描述历史的查看、还原、复制 Markdown；图片和节点定位 | 编辑器的小型测试或临时核对脚本，不需要模拟协作服务器 |
| P2 进度 | 工作项创建、完成、取消、重新打开之后的数量和进度；当前 / 已结束迭代的快照；模块进度；没有点数切换 | 数量计算的单元测试（进仓库），外加一次渲染核对 |
| P2、P3 动态、通知、工作项列表 | 状态、关联、迭代、模块、评论几类操作动态照常显示；通知的预览和"全部已读"；项目、迭代、模块、个人主页、归档几种上下文取到正确的工作项 store；列表、看板、表格、日历的入口都在 | 用不含估算、文档页、Epic 的代表性数据核对显示和 store 的选择 |
| P3 认证、实例 | 登录和注册表单的邮箱可以编辑；提交的字段、CSRF、`next_path` 正确；成功和失败后的跳转；关闭注册时的表现；实例请求失败时显示维护页；修改密码没有被"重置密码"的清理误伤 | 临时核对脚本，同时拦截 `**/auth/**` 和 `**/api/**`，检查请求体和最终地址；脚本没有列出的请求一律当作失败 |
| P4 路由 | 见 4.2 | 路由匹配的单元测试（进仓库），加临时核对脚本 |
| P5 | 图形、页面标题、错误页、帮助链接中看不到 Plane；个人访问令牌、Webhook 的设置页仍然可达 | 重跑前面的核对场景，加目视核对；S2 继续不断言启动页的文字 |

- **进仓库的测试**：只给以后仍然有效的稳定逻辑写小测试，例如：
  - 地址工具（已有 13 个）；
  - 数量进度的计算；
  - 内部跳转目标的路由匹配；
  - 能脱离接口测试的编辑器行为。

  前端单元测试（vitest）在 P1 接入 Makefile 和持续集成（总体设计 8.1、8.3 早已列出，M0 没有接入）。
- **临时核对脚本不进仓库**：它们模拟的是 M2 起就要替换的 Plane 旧接口。脚本全文、最小的假数据、运行命令和断言写进该 Phase review 的附录（markdown 代码块），可以复制出来重跑。后面的 Phase 改到同一块区域时，要重跑前面的场景；P4 重跑所有与路由有关的场景。
- **不做**：完整的 Plane 假后端、全量截图、为 M1 新造兼容模型。

### 7.6 构建产物的体积
- P1 用同一条命令记录基线：JS、CSS、字体、其他资源各自的字节数，最大的 chunk，语言 chunk 的个数。
- 收尾时用同一条命令再测一次，写进收尾 review 作对比。
- 不设拍脑袋的体积指标；构建产物的文件数只记口径，不作为门禁。

---

## 8. 工具链与部署遗留（P1）
来自两份 M0 交接、清点和评审：
- **部署遗留**：
  - `Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`；
  - `serve` 依赖及 `start`、`preview` 脚本。崩溃的根因是全局覆盖 `path-to-regexp: 0.1.13` 强加给了 `serve-handler`；删掉 `serve` 之后，这条覆盖也不再作用于任何包；
  - `public/` 中从未注册的 `sw.js`、`workbox-*.js` 及 source map；
  - 没有被引用的 `manifest.json`、`public/favicon/site.webmanifest`；
  - 13 个 `.prettierignore`：oxfmt 会读取它们，其中真正起作用的三条忽略规则改写进 `.oxfmtrc.json`（P1 spec 2.5）；
  - `web/apps/web/.gitignore`（只有 Sentry 的条目）。
- **pnpm 工作区**：删掉提到没有迁入内容的注释；删掉已不作用于任何包的覆盖项。Express 系列的覆盖项仍然作用于 `@react-router/dev` 的可选对等依赖 `@react-router/serve`，保留。
- **锁文件核对**（P1、P2、P3 每次删依赖之后都做，收尾再做一次）：
  - 比较删除前后 importers 和存活包的版本、完整性哈希、依赖边。只允许由删除直接引起的差异（对等依赖后缀、孤立的包），以及 pnpm 重新解析时向锁文件中已有版本的去重（不能出现新的包或新的版本），并逐条解释；
  - 同时检查 catalog、overrides、allowBuilds、minimumReleaseAgeExclude、peerDependencyRules 和 patches 中只作用于被删依赖的条目，它们也一并删除（pnpm 不会报告失效的条目）；
  - 方法沿用 M0/P5 spec 2.4 和 P6 计划 Task 2 Step 3。
- **没有调用方的任务和脚本**：`turbo.json` 中的 `start`、`build-storybook`、`clean`、`check`、`fix`、`fix:lint`；`fix:format` 保留，因为 README 写明它是修格式的入口；各包里同样没有调用方的脚本；Storybook（ui 和 propel 各一套，没有 Makefile 或持续集成入口）连同其配置、stories 和依赖。
  - 原则：任务或脚本只有被 Makefile、持续集成、另一个任务调用，或者是 README 写明的开发入口，才保留。
  - `test` 任务和 `@plane/services` 的 `test` 脚本保留：它们由新接入的前端单元测试调用（7.5）。
- **前端单元测试**：新增 `make` 入口（经 turbo 的 `test` 任务运行各包的 vitest），持续集成的 `web` 任务运行它。
- **关键词守卫**：7.4 的工具、P1 负责的规则、持续集成中的运行。
- **门禁覆盖 `tools/`**：`tools/` 下的检查脚本本身就是门禁，但不属于任何工作区包。根目录的 `package.json` 加 `check:lint`、`check:format`，`turbo.json` 注册为根任务，由 `make lint-web` 一起运行（P1 spec 第 3 节第 13 项）。
- **未使用的依赖**：knip 报出的 19 个依赖、4 个开发依赖、2 个未列出的依赖。其中只被未使用文件引用的依赖，连同这些文件一起删除。例外：
  - editor 的 `buffer`：knip 把它当成了 Node 的内置模块，实际上 Yjs 的工具代码还在导入它，随 Yjs 在 P2 删除。
  - `isbot`、`@react-router/node` 保留：应用没有自己的 `entry.server.tsx`，React Router 默认的服务端入口会导入它们。SPA 模式构建时也要用这个入口预渲染 `index.html`；删掉之后，React Router 的类型生成会自己把 `isbot` 加回来。
- **死代码**：Next.js 垫片中没人用的 `image.tsx`、`script.tsx`，`next/script` 的别名和 `next-script.d.ts`（4.1）；`packages/services` 中没人用的 49 个文件（3.5）。
- **构建警告**：`tailwind-config` 的 `package.json` 加 `"type": "module"`；删掉指向不存在文件的 `main`，入口只由 `exports` 声明（`./index.css`、`./postcss.config.js`，7.2）；`vite-tsconfig-paths` 插件换成 Vite 8 的 `resolve.tsconfigPaths`。
- **TypeScript 增量编译文件**：`typescript-config/base.json` 把 `tsBuildInfoFile` 写成了相对它自己的路径，13 个包写同一个文件、互相覆盖。改为每个包写自己的。
- **React #418**：`root.tsx` 的 `HydrateFallback` 在预渲染时输出空的 `<div>`；浏览器首次渲染时已经知道主题，输出加载动画，两者不一致。改为预渲染和首次渲染输出同样的结构。
- **`make web-dev`**：改为同时监视各个包（`turbo run dev --filter=web...`），并发数按持久任务数设置（10 个包的监视加 web，共 11 个持久任务，至少 12）。验证时要确认改了 `web/packages/*` 的源码真的出现在运行中的页面里，而不只是 Vite 能启动。M0 设计 6.2 和 README 中"改了 packages 要重新执行"的说明随之修改。
- **构建体积基线**：7.6。

---

## 9. Phase 划分与实施规划

所有 Phase 依次推进。每个 Phase 按 M0 的节奏：
1. worktree；
2. spec 和 plan（先做原型验证）；
3. 按任务实现，逐个评审；
4. 整分支评审（opus）；
5. 修复和限定范围的复审；
6. review、交接；
7. `--no-ff` 合并，推送；
8. 持续集成通过；
9. 清理 worktree 和分支。

每个 Phase 合并时，持续集成的全部门禁和 S1–S4 都必须通过。

**为什么这样分**：Codex 比较了几种拆法，结论是保留这五个 Phase：
- 把 P2、P3 合成一次大删除，评审、定位失败、回退的成本都会陡增；
- 按包或按层拆，为了能单独合并，很容易被迫造临时的空实现；
- 先迁路由或先改包名，会去改大量马上就要删掉的文件。

重复修改的风险（同一批共享文件被几个 Phase 改到）按 2.3 的"共享文件由一个任务收口"控制。

### P1 `web-hygiene`：工具链、遗留物与多语言
- **交付物**：
  - 第 8 节的全部内容，包括 3.5 的 `packages/services` 修剪、垫片中的死文件、前端单元测试的接入、关键词守卫、构建体积基线；
  - 第 6 节的多语言，包括企业版的整个命名空间；
  - 7.1 的 lint 上限自动核对。

  Plane 的旧地址重定向不在 P1（3.12）。
- **顺序**：lint 上限核对 → 关键词守卫（先于所有删除，每个删除任务才能先加规则、看它命中，再删到零）→ 部署遗留 → 没有调用方的任务、Storybook、前端单元测试的入口 → 未使用的依赖 → `packages/services` → 构建警告 → #418 → 两种语言 → 键一致性和死文案 → 门禁覆盖 `tools/` → 开发监视和文档（P1 plan 的 Task 1–10 和 9A）。
- **验收**：
  - `make lint-web`（含上限核对和中英文键一致性检查）、前端单元测试、关键词守卫、`make build-web`、`make e2e` 通过；
  - knip 报告中未使用的依赖、开发依赖、未列出的依赖三类为零（editor 的 `buffer` 除外，见第 8 节）；配置提示只剩 web 的 `+types/` 一条（是否出现取决于本地是否生成过类型，P3 处理，见 7.2）；
  - 构建不再有 `MODULE_TYPELESS_PACKAGE_JSON` 和 `vite-tsconfig-paths` 的警告；
  - 7.5 中 P1 一行的核对；
  - 锁文件核对（第 8 节）。

### P2 `trim-content`：内容类功能
- **交付物**：
  - 2.1 的全部功能；
  - 3.1–3.4、3.14 的裁定；
  - 属于这些功能的文案、类型、常量、依赖、图片资源和关键词规则；
  - 交给 M3、M4、M6、M7 的 handoff（3.11，从实际改动中导出完整清单）。
- **顺序**（清点和评审得出的约束，标"硬"的必须遵守）：
  1. 侧边栏的自定义导航。先删，后面几个功能都要改侧边栏。
  2. 数据分析和个人主页统计。数据分析先于估算（硬）；个人主页的卡片和重定向先到位，再删统计的数据源（硬）。
  3. 估算，连同数量进度的整条链路。数量计算和旧的估算字段在同一个任务里改（硬）。
  4. 甘特图和模块时间线。先搬走 `REVERSE_RELATIONS`（硬）。
  5. 文档页，连同更新日志、搜索 / 收藏 / 资源类型中的文档页分支、导览中的文档页一步。
  6. 编辑器收敛（3.1），包括编辑器包里的 AI 部分。文档页先于编辑器收敛（硬）。
  7. 便签。
  8. 首页（3.14）。文档页和便签先于首页（硬）。
  9. 导出和集成服务。两者同时删（硬）。
  10. 统一收口：根 store、导出文件、文案、图片资源、关键词规则。个人主页统计和数据分析删完之后才删 propel 的 `bar-chart`、`pie-chart`（硬）；图标和扩展的注册表与调用方同时删。
- **验收**：
  - 类型检查、lint（上限等于实测）、前端单元测试、构建、S1–S4 通过；
  - 关键词守卫：本 Phase 加入的规则没有未登记的命中，指向后续 Phase 的例外精确到符号（7.4）；
  - knip 不再报告这些功能的任何文件、导出和依赖；
  - 锁文件核对；
  - 7.5 中 P2 各行的核对，临时脚本写进 review 附录。

### P3 `trim-platform`：账户、平台与死代码
- **交付物**：
  - 2.2 的全部内容；
  - 3.6–3.9 的裁定；
  - knip 清零并改为门禁（7.2）；
  - 交给 M2、M4、M5 的 handoff（3.6、3.7、3.8、3.11、3.13）。
- **顺序**：
  1. 认证、实例配置、邮件相关的表单，包括新手引导和安全页。实例字段和读它们的所有地方同时改（硬）。
  2. 公开发布，连同评论的可见范围。
  3. Epic、团队、工作项类型，连同批量操作与多选（3.9）和模板等空壳。它们改同一组 store 和 hook 的分支，放在同一个任务（硬）；`work-items` 地址和共享的 `ProjectIssues` 类保留（2.2）。
  4. 其余的 AI、Unsplash、遥测。
  5. 计费和升级提示，最后删。它们引用的、已在 P2 删除的类型，由 P2 的对应任务同步处理，不为等 P3 而保留空类型。
  6. 死代码、接线、图片资源、knip 门禁。
- **验收**：同 P2（运行时核对的对象是 7.5 中 P3 的几行）；另外 `make knip` 先运行类型生成，不带 `--no-exit-code`、带 `--treat-config-hints-as-errors` 通过，持续集成中生效。

### P4 `router-native`：去掉 Next.js 兼容层
- **交付物**：
  - 第 4 节的全部内容，包括 3.12 中除数据分析以外的 10 条旧地址重定向和它们的现用入口；
  - README 中关于 `.env` 的两处说明（P6 交接）。
- **顺序**：
  1. 删掉各个包中所有的环境变量读取（包括 i18n 的 `NODE_ENV` 判断和它的类型）；
  2. 删除 `define` 和 dotenv；
  3. 迁移 `Link`、hooks 和命令面板上下文；处理参数守卫和 `<Navigate>`；
  4. 删除垫片、手写的类型声明、强制结尾 `/` 和旧地址重定向，完成全部导航的匹配核对。
- **验收**：同 P2；4.2 的验证全部通过。

### P5 `brand`：品牌与包名
- **交付物**：
  - 第 5 节的全部内容；
  - 3.10 的来源记录；
  - 前端改动清单第四节。
- **顺序**：
  1. 包名的纯机械提交，加锁文件等价核对；
  2. 品牌资源、文案、外部链接、本地存储键、组件名；
  3. 可见品牌、版权和来源的检查。
- **验收**：
  - 同 P2；
  - 包名核对（第 5 节）；
  - 品牌规则加入关键词守卫后没有未登记的命中；
  - 在实际页面上目视核对。

### 收尾 `closeout`
- 关键词守卫的全部规则：只剩明确跨 M 的例外，其他例外都已到期删除；
- 中英文文案中没有代码不再引用的键（"整键 + 模板前缀"，用变量调用的键按常量逐一核对；不能证明是死键的，先查清所有变量来源，不直接删）；
- 最后一次锁文件核对；构建体积与 P1 基线的对比（7.6）；
- 按包、按规则列出 oxlint 警告数和清零计划（7.3）；
- 前端改动清单、总体设计 9.4 同步；本 M 的 handoff 全部关闭，交给后续 M 的 handoff 都有接收的 M 和关闭条件；
- 总体设计中 M1 的状态改为"已完成"。

### 9.7 两份 M0 交接的落点
| 交接事项 | Phase |
|---|---|
| 部署遗留、`serve`、`sw.js`/workbox、pnpm 工作区的覆盖项和注释、没有调用方的 turbo 任务 | P1 |
| 功能删减之后用锁文件核对 catalog、overrides、allowBuilds | P1、P2、P3 各做一次（每次删依赖之后），收尾再做一次 |
| `.env.example`、dotenv、`define: process.env`；`turbo.json` 中 admin、space、live 的环境变量；README 中关于 `.env` 的说明 | P4 |
| 重新测出警告基线、是否加自动检查 | P1（7.1）、收尾（7.3） |
| React #418、`tailwind-config` 的 `"type": "module"`、`vite-tsconfig-paths`、`make web-dev` | P1 |
| 结尾 `/` 在去掉垫片后是否还需要 | P4（不需要，并统一地址的约定，见 4.1） |
| knip 改为门禁、配置提示 | P3（7.2）；i18n 的 `ignoreUnresolved` 在 P1 删除 |
| `@nerve/api-client`、`@nerve/e2e` 的上限保持 0 | P1（7.1） |
| S2 不断言页面文字；`knip.jsonc` 的工作区路径不随包名改 | P5 |

---

## 10. 验收方式
- **静态验收**：M1 是静态验收（总体设计 9.2），包括类型检查、oxlint、knip、格式检查、构建、前端单元测试和关键词守卫。
- **保留行为**：按 7.5 的矩阵核对。"静态"指不以新的后端为完成前提，不等于只看编译结果。
- **端到端**：M1 不新增用户故事；S1–S4 在每个 Phase 合并时都通过（总体设计 9.3："之前所有 M 的故事也都通过"）。S2 核对首页、深层路径、同源请求，能发现构建产物和内嵌方面的倒退，但不经过保留的业务页面。
- **没有 PAT 对等验收**：M1 不涉及接口。

---

## 11. 完成标准
- [ ] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。
- [ ] 类型检查通过；knip 为零，并且是持续集成的门禁；oxlint 的每个包都等于新的上限，上限由 7.1 自动核对；格式检查通过；前端单元测试在持续集成中运行并通过。
- [ ] 关键词守卫在持续集成中运行，没有未登记的命中，只剩明确跨 M 的例外；中英文文案中没有代码不再引用的键。
- [ ] 7.5 的保留行为矩阵逐行核对过，临时脚本都写在各 Phase review 的附录中。
- [ ] `make build` 能构建；S1–S4 全部通过。
- [ ] 多语言只剩 `zh-CN` 和 `en`；界面上没有 Plane 的名称和 Logo；包名全部是 `@nerve/*`。
- [ ] oxlint 清零计划、构建体积对比写进收尾 review。
- [ ] `handoffs/` 中没有 `open` 状态的事项；交给后续 M 的事项已放进对应 M 的 `handoffs/`。
- [ ] 前端改动清单同步（第二、四节，以及本文件第 3 节带来的增减）；总体设计中 M1 的状态改为"已完成"。

---

## 12. Phase 进度表

| Phase | 名称 | 状态 | spec | plan | review |
|---|---|---|---|---|---|
| P1 | web-hygiene | 进行中 | [spec](specs/P1-web-hygiene.md) | [plan](plans/P1-web-hygiene.md) | — |
| P2 | trim-content | 未开始 | — | — | — |
| P3 | trim-platform | 未开始 | — | — | — |
| P4 | router-native | 未开始 | — | — | — |
| P5 | brand | 未开始 | — | — | — |
| 收尾 | closeout | 未开始 | — | — | — |

---

## 13. 风险

| 风险 | 应对 |
|---|---|
| 删除量大（约 1200 个文件被删、数百个文件被改），P2、P3 的任务之间容易改到同一批文件（根 store、侧边栏、工作项 store 的分支、编辑器包） | 按第 9 节的顺序串行推进；共享文件由一个任务收口（2.3）；每个任务结束时类型检查、lint、构建都要通过，出问题只回退这一个任务 |
| 前端连不上 Plane 的接口，删错了也不会在运行时自己暴露；有引用但不起作用的代码、保留功能的唯一入口，类型检查和 knip 都发现不了 | 先写清保留边界（2.3）；7.5 的保留行为矩阵，稳定逻辑用进仓库的单元测试，其余用可重跑的临时脚本 |
| 去掉垫片后路由参数的类型变严，暴露出 152 处可能为空的参数 | 放在删除之后做（相关文件更少）；逐处按真实情况处理，不批量断言 |
| 地址的形式（带不带结尾 `/`）影响当前菜单项的判断 | 4.1 的地址约定；所有判断改用 React Router 的匹配；内部跳转目标的路由匹配测试 |
| 翻译键可以不带命名空间、也可以用变量调用，删文案时容易删多或删少 | 用"整键字面量 + 模板前缀"并追到常量的方式核对；中英文键一致性由门禁保证；收尾时核对未引用的键 |
| 关键词验收的手工命令会把正则报错当成零命中，或者为了零命中扩大豁免 | 7.4 的关键词守卫：统一的正则引擎、报错即失败、精确到符号的例外和到期检查 |
| lint 上限自动核对让每个减少警告的提交都必须同时改上限 | 这正是 M0 定下的规则；检查会直接给出应改成的数值 |
| Logo 没有设计稿 | P5 做简单的矢量图形，文件可以直接替换 |
| `@makeplane/propel` 是长期的第三方依赖 | 锁定版本，P5 记录来源；需要自己修改设计令牌时，再按 3.10 评估并入源码 |
