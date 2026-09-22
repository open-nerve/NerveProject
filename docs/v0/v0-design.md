# Nerve v0 总体设计

| 项 | 内容 |
|---|---|
| 版本 | v0（启动首版，覆盖 M0–M8） |
| 日期 | 2026-09-22 |
| 状态 | 已批准 |
| 代码基线 | Plane v1.4.2，提交 `02c19e1`（仅作参考，放在仓库外的 `plane/`；`02c19e1` 是 `preview` 分支，`v1.4.2` 标签指向 `5f7d927`，两者的迁移文件逐字节相同，见 [`tools/plane-schema/README.md`](../../tools/plane-schema/README.md)） |
| 协议 | AGPL-3.0 |
| 仓库 | https://github.com/open-nerve/NerveProject |

相关文档：[文档约定](../README.md) · [与 Plane 的差异清单](plane-diff.md) · [前端改动清单](frontend-changes.md)

---

## 0. 背景与目标

### 0.1 产品定位
Nerve 是一个轻量的多人协作项目管理系统，面向企业和互联网团队，对标飞书项目、Plane、Jira，但更轻。v0 的目标是做出一个能用的启动版本：一个小团队可以用它真实地管理项目，并且可以按 AGPL 开源发布。

### 0.2 核心原则
1. **账户不区分使用者。** 一个账户就是一个用户，权限由管理员分配。它背后是真人在页面上操作，还是 Agent 在调接口，产品不知道，也不关心。数据模型中不存在"机器人账户""Agent 账户"之类的区分。
2. **页面能做的，接口都能做。** 前端只是公开接口的一个使用方，没有只给前端用的隐藏接口。所有规则都在服务端执行。
3. **站在 Plane 的肩膀上。** 前端复用 Plane 的代码（裁剪后）；后端用 Go 重写；保留下来的功能，其表结构和业务规则照搬 Plane。我们要做的是决定"要哪些、不要哪些"，而不是从零设计需求。
4. **轻量。** 默认部署只有一个 Go 可执行文件（前端内嵌其中）加一个 Postgres。不需要 Redis、消息队列或对象存储。
5. **代码干净。**
   - 后端严格遵循 SOLID，职责清楚，没有循环依赖，没有上帝文件，但也不过度设计（见第 6 节）。
   - 前端把不要的功能删干净，不保留兼容代码，前后端的数据结构统一（见第 7 节）。

### 0.3 做法概要
- **前端**：分叉 Plane 的 `apps/web` 及其依赖包，删掉不要的功能，然后直接对接新接口。前端的数据类型由 OpenAPI 生成。
- **后端**：用 Go 实现一套**全新设计的 REST 接口 `/api/v0`**。先写 OpenAPI 描述，再生成代码。
- **数据库**：保留的功能照搬 Plane 的表结构，每一处改动都登记到[差异清单](plane-diff.md)。

---

## 1. 范围

### 1.1 v0 保留的功能

| 领域 | 功能 |
|---|---|
| 账户与认证 | 邮箱密码注册、登录、退出、修改密码；个人资料和偏好；新手引导；个人访问令牌（PAT） |
| 工作区 | 工作区、成员与角色（管理员 / 成员 / 访客）、成员邀请（在系统内接受，不发邮件） |
| 项目 | 项目、项目成员与角色、项目标识（如 `PROJ`）、项目功能开关（迭代 / 模块 / 视图 / 收集箱）、每个成员的显示设置 |
| 状态与标签 | 按组划分的自定义状态（待规划 / 未开始 / 进行中 / 已完成 / 已取消 / 待分诊）、项目标签（支持层级） |
| 工作项 | 标题、富文本描述、状态、优先级、负责人、标签、起止日期、父子任务、关联关系（阻塞 / 关联 / 重复等）、链接、附件 |
| 列表视图 | 列表、看板、表格、日历四种布局；筛选、分组、子分组、排序 |
| 评论与动态 | 评论（支持回复）、表情回应（工作项和评论都支持）、操作动态 |
| 历史版本 | 工作项快照、描述修改历史（可查看、可还原） |
| 草稿 | 个人草稿，可发布为工作项 |
| **归档** | 工作项归档与恢复（仅限已完成或已取消的）；项目级自动归档（已完成或已取消、且 N 个月未更新的工作项自动归档）；迭代归档（仅限已结束的）；模块归档（仅限已完成或已取消的）；项目归档。已归档的对象默认不出现在列表、看板和搜索中，可以在"归档"页查看和恢复 |
| 迭代 | 迭代（Sprint）、进度与燃尽图、把未完成的工作项转到其他迭代 |
| 模块 | 模块、模块负责人和成员、模块链接 |
| 视图 | 保存的项目视图；跨项目的工作区视图（分配给我的、我创建的、我关注的等） |
| 需求收集箱 | 提交到收集箱、分诊（接受 / 拒绝 / 暂缓 / 标记重复） |
| 通知 | 站内通知、关注工作项、@提及 |
| 收藏与访问 | 侧边栏收藏、最近访问 |
| 文件 | 附件、描述中的图片、头像、工作区图标、项目封面 |
| 搜索 | 命令面板搜索、选择父任务或关联任务时的搜索、@成员搜索、按编号（`PROJ-12`）搜索 |
| 开放能力 | Webhook、接口调用日志、对外接口文档（OpenAPI） |

### 1.2 v0 不做的功能（从代码中彻底删除）
- 文档页（Pages）及其实时协作服务（apps/live）
- 估算（Estimates）、甘特图和时间线
- 自动关闭（项目自动化中的"长期未更新的工作项自动关闭"）
- 数据分析、导出、便签、首页快捷链接、首页个性化（包括侧边栏的自定义导航）、自定义主题
- 公开发布看板（apps/space）、实例管理后台（apps/admin；实例配置改由配置文件和命令行管理）
- 第三方登录（OAuth）、验证码登录、找回密码、所有邮件发送（包括通知邮件和邮件通知偏好）
- AI 助手、Unsplash 封面图、遥测、更新日志
- 项目邀请（改为直接从工作区成员中添加）
- Plane 企业版残留（Epic、团队、工作项类型、升级和计费提示）以及 Plane 自身的死代码

### 1.3 关键决策记录

| 决策 | 选择 | 放弃的方案及原因 |
|---|---|---|
| 架构形态 | 传统的客户端 + 服务端 + Postgres | Supabase 无服务方案：最初想尝试，经权衡后放弃，改用成熟的架构 |
| 后端语言 | Go | Node TS：能和前端共用类型，但内存占用更高。接口一致性改由"先写接口描述、再生成代码"来保证 |
| 后端架构 | 模块化单体 + 端口与适配器，只在一处接线 | 按技术分层堆放（容易长出上帝文件）；微服务、CQRS、依赖注入框架（对当前规模属于过度设计） |
| 接口策略 | 全新设计的干净接口 | 兼容 Plane 的接口：前端改动小，但要背上 Plane 接口的全部历史包袱，也不适合作为公开接口 |
| 接口版本 | `/api/v0`，与产品主版本号一致 | `/api/v1`：和文档中的 v0 不一致，容易混淆 |
| 前端数据类型 | 以 OpenAPI 生成的类型为准，store 直接使用新接口的数据结构 | 用转换层把新接口的数据翻译成 Plane 旧类型：属于兼容代码，会让前后端长期不统一 |
| 认证方式 | 所有调用方统一用 `Authorization: Bearer`（JWT 访问令牌 + 刷新令牌 + PAT） | 会话 Cookie：程序调用不方便，还需要处理 CSRF |
| 配置 | 分环境的配置文件（dev / test / prod）+ 环境变量覆盖 | 只用环境变量：不同环境的配置不直观，也不便于统一管理 |
| 开源协议 | AGPL-3.0，并换掉 Plane 品牌 | 闭源商业化：和复用 Plane 前端代码相冲突 |
| 文件存储 | 默认存本地磁盘；可切换到任意 S3 兼容存储 | MinIO：社区版已于 2026-04 归档，停止维护。RustFS：2026-09 才发布 1.0，过去一年有多个严重安全漏洞，内存占用也高 |
| 草稿 | 一张表，`payload` 字段存"创建工作项"的请求体 | Plane 的 5 张表：结构冗余。在工作项表上加 `is_draft` 标记：会占用编号，也会污染所有查询 |
| 分组 | 在服务端完成（一条 SQL 返回每组的前 N 条和总数） | 由前端拼装：看板加泳道时请求数会成倍增长 |
| 去掉的功能 | 从代码中彻底删除 | 用开关隐藏或返回空数据：留下的死代码会误导后续维护 |

---

## 2. 总体架构

```
            浏览器（Nerve 前端，基于 Plane 分叉）          Agent / 脚本
                         │ Bearer（JWT）                      │ Bearer（JWT 或 PAT）
                         └──────────────┬─────────────────────┘
                                        ▼
            ┌──────────────── nerve（Go，单个进程）───────────────────┐
            │  /api/v0 接口层（由 OpenAPI 生成）                       │
            │  业务模块（用例 + 领域规则），每个写操作一个事务             │
            │  River 后台任务：通知分发、Webhook 投递等                  │
            │  River 定时任务：自动归档、物理删除、各类清理               │
            │  托管前端静态文件（go:embed）                             │
            └───────────────┬──────────────────────────┬─────────────┘
                            ▼                          ▼
                      PostgreSQL 18            文件存储：本地磁盘（默认）
                                               或任意 S3 兼容存储
```

### 2.1 运行形态
- **默认部署**：一个 `nerve` 可执行文件加一个 Postgres。需要 HTTPS 时，在前面加一层 Caddy 等反向代理。
- **配置**：分环境的配置文件加环境变量覆盖，详见 6.8。
- **不需要的组件**：Redis、RabbitMQ、Celery、实时协作服务、独立的前端服务器。

### 2.2 仓库结构
```
NerveProject/
  api/          接口契约，是唯一的依据：按模块拆分的描述文件，打包为 api/dist/openapi.yaml
  server/       Go 后端
  web/          前端（pnpm monorepo：apps/web + 所需的 packages）
  e2e/          Playwright 端到端测试
  deploy/       docker compose 等部署配置
  tools/        不属于运行时的工具（如 Plane 表结构快照）
  docs/         设计与过程文档
  LICENSE       AGPL-3.0
```
仓库根目录同时是 pnpm 工作区的根（包含 `web/` 下的应用和包，以及 `e2e/`），这样端到端测试可以直接使用生成的 TS 客户端。`plane/` 和 `refer/` 只是本地的参考代码，已加入 `.gitignore`。完整布局见 [M0 设计文档](M0-foundation/M0-design.md#2-仓库布局m0-完成时)。

### 2.3 版权与品牌
- 仓库使用 AGPL-3.0。`web/` 中来自 Plane 的文件保留原有的版权声明。
- AGPL 授予的是代码的使用权，**不包括商标**。产品名为 Nerve，Plane 的名称和 Logo 必须全部替换。

---

## 3. 接口设计规范

### 3.1 基本约定
- **路径前缀**：`/api/v0`。**接口版本号与产品主版本号保持一致**：
  - v0 期间允许不兼容的修改，修改时同步更新 `openapi.yaml` 和前端。
  - 发布 v1 时，接口升级为 `/api/v1`，并从那时起承诺向后兼容。
- **字段命名**：JSON 字段用 snake_case，和数据库列名、前端类型保持一致。
- **数据格式**：ID 一律用 UUID 字符串；时间用 RFC 3339 格式（UTC）；日期用 `YYYY-MM-DD`。
- **HTTP 方法**：
  - GET 查询，POST 创建，DELETE 删除（软删除）。
  - PATCH 部分更新：只改传入的字段，传 `null` 表示清空。
  - **POST 和 PATCH 都返回改完之后的完整资源。**
- **接口描述**：放在 `api/`，按模块拆分（`api/modules/<模块>.yaml`，公共组件放在 `api/common.yaml`），打包为 `api/dist/openapi.yaml`。先改接口描述，再写实现。
- **OpenAPI 版本：已定为 3.1**（`openapi: 3.1.0`）。M0/P3 验证了整条工具链，结论和写法约定见 [P3 spec](M0-foundation/specs/P3-api-contract.md) 2.2、2.4。

### 3.2 路由：列表挂在父资源下，单个资源用短路径
```
GET   /api/v0/workspaces
GET   /api/v0/workspaces/{slug}/projects
GET   /api/v0/projects/{project_id}/issues
POST  /api/v0/projects/{project_id}/issues
GET   /api/v0/issues/{issue_id}
PATCH /api/v0/issues/{issue_id}
GET   /api/v0/workspaces/{slug}/issues/{PROJ-12}     按编号取
GET   /api/v0/workspaces/{slug}/issues               跨项目列表（供工作区视图使用）
GET   /api/v0/issues/{issue_id}/comments
```
- **为什么不需要层级 ID**：项目、工作项等资源的 UUID 全局唯一，服务端能自己查出归属并校验权限，调用方不需要凑齐多层 ID。
- **关联字段放在工作项本身**：负责人、标签、迭代、模块都是工作项自己的字段（`assignee_ids`、`label_ids`、`cycle_id`、`module_ids`），一次 PATCH 就能改完。
- **少数业务动作**：用 `POST /资源/{id}/动作名`，例如：
  - `POST /issues/{id}/archive`、`POST /issues/{id}/unarchive`（迭代、模块、项目同理）
  - `POST /cycles/{id}/transfer-issues`
  - `POST /drafts/{id}/publish`
  - `POST /notifications/mark-all-read`
- **最近访问**：由前端显式调用 `POST /api/v0/me/recent-visits` 记录。所有 GET 请求都没有副作用。

### 3.3 筛选
- 用普通查询参数：`?state_id=a,b&assignee_id=me&priority=urgent,high&target_date=2026-09-01..2026-09-30`。
- 不同参数之间是 AND 关系；逗号表示"属于其中之一"；`..` 表示范围；`me` 表示当前账户。
- **归档**：列表默认不包含已归档的对象；传 `?archived=true` 只返回已归档的对象。
- 依据：Plane 社区版前端的筛选器只会生成"多个条件全部满足"的组合，每个条件只有"等于 / 属于 / 范围"三种判断，上面的格式能完整表达。

### 3.4 分页与分组
- **分页**：用游标。请求带 `?limit=50&cursor=...`，响应格式为 `{ "data": [...], "next_cursor": "..." }`。游标是不透明的字符串。
- **分组**：作为列表接口的可选参数，由服务端完成：
  ```
  GET /api/v0/projects/{id}/issues?group_by=state_id&sub_group_by=priority&limit=50
  → { "groups": [ { "key": "...", "total": 37, "data": [...], "next_cursor": "..." } ] }
  ```
  - 实现：一条 SQL（借助窗口函数）算出每组的前 N 条和总数。
  - "某一组加载更多"：用普通列表加上这一组的筛选条件。
  - 按标签、负责人分组时，同一个工作项会出现在多个组里，由服务端处理。

### 3.5 错误
- 统一使用 RFC 9457（`application/problem+json`），`code` 是给程序判断用的固定字符串：
  ```json
  { "status": 422, "code": "issue.state_not_in_project", "title": "Unprocessable Entity",
    "detail": "状态不属于该项目",
    "errors": [{ "field": "state_id", "message": "..." }] }
  ```
- 看不到的资源返回 **404**，不泄露它是否存在；能看到但没权限执行操作，返回 **403**。
- `title` 固定为 HTTP 状态短语，即 Go 的 `http.StatusText(status)`（不带 `type` 时符合 RFC 9457 的语义），具体说明放在 `detail`，程序按 `code` 分支。
- 平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；模块的错误码带模块前缀。
- 请求 ID 只出现在 `X-Request-Id` 响应头中，不放进响应体。

### 3.6 其他
- **限流**：按令牌计数，在进程内实现。登录接口单独按"IP + 邮箱"限流。具体数值在 M2 确定，默认参考 Plane。
- **关联对象只返回 ID**：以后按需增加 `?expand=`。
- **v0 不做乐观锁**：PATCH 只改传入的字段，同一字段并发修改时以后写入的为准。

---

## 4. 认证

### 4.1 令牌
所有请求都使用 `Authorization: Bearer <token>`，服务端根据令牌格式区分类型。

| 令牌 | 获取方式 | 形式 | 有效期 |
|---|---|---|---|
| 访问令牌 | 登录，或用刷新令牌换取 | JWT（Ed25519 签名），只包含用户 id、会话 id 和过期时间，**不含任何权限信息** | 15 分钟 |
| 刷新令牌 | 登录时一并下发 | 随机字符串，数据库只存哈希（`auth_sessions`） | 30 天；每次使用后换新，并检测旧令牌是否被重复使用 |
| 个人访问令牌（PAT） | 在设置页生成 | `nrv_pat_` 前缀的随机字符串，数据库只存哈希（`api_tokens`） | 由用户设定；可随时撤销 |

### 4.2 规则
- **权限不放进令牌**：每个请求都从数据库读取成员关系和角色。所以移出项目、调整角色是立即生效的。
- **会话撤销**：退出登录、修改密码、账户停用时，吊销该账户的刷新令牌；账户停用时，同时让该账户的所有会话失效。
- **重复使用检测**：一旦发现某个已经换过新的刷新令牌又被使用，就作废这次登录派生出的所有令牌。
- **密码哈希**：用 argon2id。
- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置控制；被邀请的邮箱始终可以注册。
- **忘记密码**：没有邮件服务，由服务器管理员通过命令行重置：`nerve users reset-password --email <email>`。

### 4.3 浏览器端
- **令牌存放**：访问令牌存在内存；刷新令牌存在 localStorage。
- **防范 XSS**：HTML 在服务端清洗；配置严格的内容安全策略（CSP）；刷新令牌每次使用后换新，并做重复使用检测。以后如需加强，只需把刷新令牌移进一个仅限续期接口使用的 HttpOnly Cookie，其他接口不受影响。
- **多标签页续期**：用 `navigator.locks` 保证同一时间只有一个标签页在续期。
- **401 处理**：先续期，再重试一次；续期失败就跳转到登录页。
- **`<img src>` 没法带令牌**：
  - 编辑器图片：通过接口换取短期签名地址（Plane 编辑器本来就是异步获取图片地址的）。
  - 头像、图标、封面、附件：接口直接返回带签名的地址。

---

## 5. 数据库

### 5.1 原则
保留的功能照搬 Plane 的表结构。只做以下四类改动，每一处都登记到[差异清单](plane-diff.md)：
1. 删除被砍功能的表和外键列。
2. 删除已确认废弃的遗留列。
3. 把原本只写在应用代码里的默认值、非空约束和枚举检查挪进数据库。
4. 修复明显的缺陷和不一致。

### 5.2 保留的表（共 45 张：44 张来自 Plane，1 张新增）

| 领域 | 表 |
|---|---|
| 账户 | `users`、`profiles`、`api_tokens` |
| 认证（新增） | `auth_sessions` |
| 工作区 | `workspaces`、`workspace_members`、`workspace_member_invites`、`workspace_user_properties` |
| 项目 | `projects`、`project_members`、`project_user_properties` |
| 状态与标签 | `states`、`labels` |
| 工作项 | `issues`、`issue_assignees`、`issue_labels`、`issue_relations`、`issue_links`、`issue_subscribers`、`issue_mentions` |
| 评论与动态 | `issue_comments`、`issue_reactions`、`comment_reactions`、`issue_activities` |
| 历史版本 | `issue_versions`、`issue_description_versions` |
| 草稿 | `draft_issues`（重新设计，见 5.4） |
| 迭代 | `cycles`、`cycle_issues`、`cycle_user_properties` |
| 模块 | `modules`、`module_issues`、`module_members`、`module_links`、`module_user_properties` |
| 视图 | `issue_views` |
| 需求收集箱 | `intakes`、`intake_issues` |
| 通知 | `notifications` |
| 收藏与访问 | `user_favorites`、`user_recent_visits` |
| Webhook | `webhooks`、`webhook_logs` |
| 接口日志 | `api_activity_logs` |
| 文件 | `file_assets` |

归档不需要单独的表，用 `issues`、`cycles`、`modules`、`projects` 上的 `archived_at` 列表示，项目的自动归档周期存在 `projects.archive_in`。另有 River 任务队列自带的表：用 River 为锁定版本导出的迁移 SQL（`river migrate-get`）写成 goose 迁移，和业务表在同一条迁移链上——一个版本表、一次 `migrate up`、一个就绪检查，e2e 的模板库也只迁移一次；不使用 River 自带的迁移命令（见 M2 的 [M0-P2-platform-notes](M2-auth/handoffs/M0-P2-platform-notes.md)）。Plane 原有 96 张表，未保留的 52 张及原因见[差异清单](plane-diff.md#一未保留的表)。

### 5.3 主要改动

| 改动 | 原因 |
|---|---|
| 删除 `users.is_bot`、`users.bot_type`、`api_tokens.user_type` | 在数据层区分人和机器人，违背核心原则 |
| `api_tokens` 只存令牌的哈希 | Plane 存令牌原文，数据库一旦泄露，令牌就一起泄露 |
| 工作项编号改由 `projects` 上的计数列生成（`UPDATE … RETURNING`），删除 `issue_sequences` | 同一项目内取号串行、编号不会复用，效果和原来一样，但省掉一张表和咨询锁 |
| 删除 `project_identifiers` | 和 `projects.identifier` 存的是重复数据 |
| 删除被砍功能的列：`estimate_point_id`、`type_id`、`projects.close_in`、`description_binary`、`page_view` 等 | 对应功能已经砍掉 |
| `issues.archived_at` 由 `date` 改为 `timestamptz` | 和 `cycles`、`modules`、`projects` 的 `archived_at` 保持一致 |
| 删除遗留列：`issues.point`、`issues.is_draft`、`projects.emoji` 和 `icon_prop`、被新字段取代的旧 URL 列和旧筛选列、账单和移动端字段、`external_source` 和 `external_id` 等 | Plane 代码中已不再使用；v0 也不做导入 |
| 默认值、非空约束、枚举检查（优先级、状态组、角色等）都放进数据库 | Plane 的默认值全在 Python 代码里 |
| 补充约束：项目内编号唯一、同一工作项同一标签唯一、**一个工作项最多属于一个迭代**、标签名在工作区内唯一 | Plane 只在代码里检查，或者本身就有缺陷 |

### 5.4 草稿：一张表
```
draft_issues(id, workspace_id, project_id NULL, payload jsonb,
             created_by, created_at, updated_at, deleted_at, …)
```
- **payload**：和"创建工作项"接口的请求体完全一致，在 OpenAPI 中复用同一个结构定义。
- **保存草稿**：只校验字段类型，不校验引用的对象是否还存在。
- **发布**：`POST /api/v0/drafts/{id}/publish`，在同一个事务里完成三件事：
  1. 走和"创建工作项"完全相同的代码路径，所有校验都在这里做。
  2. 把草稿的附件改挂到新工作项上。
  3. 删除草稿。
  引用失效时，返回明确的错误码和出错字段。
- **可见范围**：草稿只有创建人自己能看到，列表按创建时间倒序。

### 5.5 照搬的机制
- **软删除**：删除只是把 `deleted_at` 设为当前时间；唯一约束只对未删除的数据生效（部分唯一索引）。连带删除在同一个事务里同步完成；60 天后由定时任务物理删除。
- **归档**（规则照搬 Plane）：
  - `archived_at` 不为空，就表示已归档。已归档的对象不参与默认查询，但仍然可以查看、可以恢复。
  - 工作项只有处于"已完成"或"已取消"状态组时才能归档。
  - 迭代只有已结束的才能归档；模块只有状态为"已完成"或"已取消"的才能归档。
  - 项目由项目管理员归档。
  - **自动归档**：项目设置了 `archive_in`（1–12 个月）后，定时任务每天把已完成或已取消、并且超过 `archive_in × 30` 天未更新的工作项自动归档，同时记录一条操作动态。
  - 迭代归档时，同时移除指向它的收藏（照搬 Plane）。
- **审计字段**：`created_by` / `updated_by` 在业务代码中显式赋值为当前账户。
- **排序**：`sort_order` 是浮点数，拖拽时由前端计算新值；初始值的算法和 Plane 一致。
- **ID**：UUID 类型，由 Go 1.27 标准库的 `uuid.NewV7()` 生成；数据库不设默认值。
- **默认状态**：新建项目时自动生成 6 个状态：Backlog（默认）、Todo、In Progress、Done、Cancelled、Triage（待分诊，供收集箱使用）。
- **完成时间**：状态进入"已完成"组时填上 `completed_at`，离开时清空。

### 5.6 工具
- **数据库**：PostgreSQL 18。
- **迁移**：goose（纯 SQL），迁移文件内嵌进程序。
- **建表方式**：**每个 M 为自己模块的表编写迁移**，不在 M0 一次性建全部表。起点是 M0 生成的 Plane 表结构快照 [`tools/plane-schema/plane-v1.4.2-schema.sql`](../../tools/plane-schema/plane-v1.4.2-schema.sql)（在临时库上跑完 Plane 自带的 Django 迁移，再用 `pg_dump --schema-only` 导出），然后按 5.3 和[差异清单](plane-diff.md)修改。**不手抄。**
- **外键方向**：每个模块的迁移只建自己的表，以及指向更早建立的模块的外键；指向更晚建立的模块的外键，由后建的模块用 `ALTER TABLE` 补上。
- **数据访问**：pgx + sqlc，手写 SQL，生成类型安全的 Go 代码。

---

## 6. 后端架构

### 6.1 架构原则
后端是一个**模块化单体**：一个进程，内部按业务划分成边界清楚的模块，每个模块内部采用**端口与适配器**（六边形架构）。目标是职责清楚、接线清楚、方便以后扩展，同时不引入这个规模用不上的复杂度。

| SOLID 原则 | 在本项目中的落实方式 |
|---|---|
| 单一职责 | 每个模块只负责一块业务；应用层一个用例一个文件；handler 只做协议转换，仓储只做持久化 |
| 开闭原则 | 新增一种副作用（比如一种新的集成）= 新增一个事件订阅者，工作项模块的代码不用改；新增一种存储 = 新增一个存储适配器 |
| 里氏替换 | 同一个端口的不同实现（`local` 和 `s3` 存储、真实时钟和测试时钟）可以互换；每个端口都配一套共享的契约测试，所有实现都必须通过 |
| 接口隔离 | 接口要小，由**使用方**定义，只包含使用方真正需要的方法，不做"万能仓储" |
| 依赖倒置 | 应用层和领域层只依赖端口（接口），不依赖 pgx、River、HTTP 等具体技术；具体实现只在组合根里接上 |

### 6.2 目录结构
```
server/
  cmd/nerve/main.go         只负责解析命令行，然后交给 bootstrap
  configs/                  分环境的配置文件（见 6.8）
  migrations/               goose 迁移（纯 SQL）
  internal/
    bootstrap/              组合根：读取配置 → 创建适配器 → 接到各模块的端口上 → 启动 HTTP 和 River。
                            全项目只有这里负责"接线"
    platform/               与业务无关的技术基础件：config、postgres（连接池、事务管理器）、
                            logging、httpserver（中间件、problem+json）、ratelimit、clock、idgen
    shared/                 共享内核，尽量小：ID 类型、Actor（当前账户）、领域错误与错误码、
                            分页游标、领域事件接口、Authorizer 端口
    modules/
      identity/             账户、会话、PAT、密码
      access/               成员角色查询与权限规则表（实现 Authorizer 端口）
      workspace/            工作区、成员、邀请
      project/              项目、项目成员、状态、标签、显示设置
      issue/                工作项、评论、关联、链接、表情回应、操作动态、历史版本、草稿、归档、列表查询
      planning/             迭代、模块
      intake/               需求收集箱
      navigation/           保存的视图、收藏、最近访问
      notification/         站内通知（订阅领域事件）
      webhook/              Webhook 配置与投递（订阅领域事件）
      asset/                文件（存储端口，以及 local 和 s3 两种适配器）
      audit/                接口调用日志
```

**每个模块内部的结构（以 issue 为例）：**
```
modules/issue/
  domain/            实体、值对象、业务规则、领域事件、领域错误，以及本模块需要的端口（仓储接口等）
  app/               用例，一个用例一个文件（create_issue.go、archive_issue.go……）：
                     负责事务边界、权限检查和流程编排
  adapter/
    postgres/        仓储实现（基于 sqlc）；列表引擎这类复杂读取写成专门的查询
    http/            handler：把 oapi-codegen 生成的请求类型转成用例的输入，再把结果转成响应
  module.go          模块入口：New(依赖) *Module；(*Module).Register(mux, apiErrors) 把模块生成的路由挂到 bootstrap 的根路由上（M0/P3）
```

### 6.3 依赖规则（在持续集成中强制检查）
1. **依赖方向只能向内**：`adapter → app → domain`。`domain` 只能依赖标准库和 `shared`。
2. **模块之间不能直接 import**：
   - 需要其他模块的能力时，在自己的 `app` 层声明一个小接口，由 `bootstrap` 用对方模块的实现接上。
   - 例外：所有模块都可以依赖 `shared` 和 `platform` 中的技术基础件。
3. **跨模块的副作用通过领域事件**：
   - 比如工作项模块只发布 `IssueUpdated` 这样的事件，不知道谁在监听。
   - 通知、Webhook 等模块在 `bootstrap` 中注册为订阅者。
   - 订阅者在同一个事务内被调用，通常只做一件事：往 River 里投递任务。所以事务是原子的，同时各模块之间解耦。
4. **只在组合根接线**：禁止全局可变状态，禁止 `init()` 副作用，禁止服务定位器。所有依赖都通过构造函数显式传入。
5. **不写大文件、不建大杂烩包**：
   - 一个文件只做一件事；超过约 400 行的文件在评审时必须说明理由或者拆分。
   - 禁止 `utils`、`common`、`helpers` 这类大杂烩包。
6. **强制手段**：
   - Go 编译器本身禁止包之间的循环依赖。
   - **架构测试**（写法类似 Java 的 ArchUnit，随 `go test` 一起运行）检查：依赖只能向内；`domain` 和 `app` 只能依赖标准库（不含 `net/http`、`database/sql`）、本模块的内层包和 `shared`；模块内的包只能放在 `domain`、`app`、`adapter` 和模块根；`shared` 本身也只能依赖标准库（不含 `net/http`、`database/sql`）；这几层连间接依赖也不能碰到 `net/http`、`database/sql` 和第三方库；模块之间不能互相导入；`platform` 不依赖模块，`platform` 的各个包之间也不互相依赖；只有组合根能导入各个模块；生成的代码只能被本模块的适配器导入；测试工具只能被测试代码导入。
   - golangci-lint 的 depguard 只负责禁止使用某些库（比如第三方 uuid 库、viper、标准库 `log`）。
   - 以上都作为持续集成的门禁。

**明确不做的事（避免过度设计）**：
- 不做 CQRS 和事件溯源，不引入依赖注入框架（接线是手写的构造函数调用）。
- 仓储只是 sqlc 之上的薄封装。
- 复杂的读取（比如列表引擎）允许在适配器中写专门的查询，不强制经过领域实体。

### 6.4 请求处理流程
```
请求 → 请求 ID → 异常恢复 → 访问日志 → 认证（识别 JWT 或 PAT，得到 Actor）
     → 限流 → 接口调用日志（只记写操作，异步批量写入）
     → handler（生成的代码只做参数绑定和 JSON 解码，不校验取值；校验放在哪一层由 M2 决定，见 [P3 spec](M0-foundation/specs/P3-api-contract.md) 7）
     → 用例：TxManager.WithinTx { 权限 → 业务规则 → 写数据 → 发布领域事件 } 提交
     → 响应 / problem+json
```
- **请求 ID → 异常恢复 → 访问日志**这三个平台中间件固定在 `httpserver.NewServer` 内部，不可漏掉或调换。
- **认证、限流、接口调用日志**挂在访问日志之后，按路由分别接到各个 API 路由上，不作用于健康检查和前端页面。
- **每个写操作对应一个事务。** 业务数据、操作动态、历史版本、投递给 River 的任务，要么一起成功，要么一起回滚。
- **事务的传递**：`TxManager` 端口声明在 `internal/shared`（由使用方定义接口），`platform/postgres` 提供实现，`bootstrap` 负责接线；仓储从 `ctx` 中取出当前事务。用例代码不接触任何数据库类型。

### 6.5 权限
- **规则照搬 Plane**：
  - 三种角色：管理员 20、成员 15、访客 5。
  - 项目级操作：需要是项目成员且角色符合要求；或者是项目成员（角色不限），同时是工作区管理员。
  - 创建者本人可以修改或删除自己创建的对象。
  - 访客在项目没开 `guest_view_all_features` 时，只能看到自己创建的工作项。
  - 项目分公开和私密两种可见性。
- **集中定义**：规则表放在 `modules/access` 中，各模块通过 `shared` 里的 `Authorizer` 端口调用：
  ```go
  "issue.update":  {Project: [Admin, Member], AllowCreator: true}
  "issue.archive": {Project: [Admin, Member]}
  "issue.delete":  {Project: [Admin],         AllowCreator: true}
  ```
- **归属查询**：Plane 在每张属于项目的表上都存了 `workspace_id` 和 `project_id`，查一次资源就能知道它的归属。
- **测试**：用一张完整的"角色 × 操作"权限矩阵逐格测试。

### 6.6 操作动态、历史版本、通知、Webhook
```
事务内（同步）：
  issue 用例：字段变更比对 → issue_activities（格式照搬 Plane）
            → 描述变更时写 issue_description_versions；同时写 issue_versions
            → 自动关注：创建人、负责人、评论人
            → 解析描述中的 @提及，同步 issue_mentions
            → 发布领域事件
  事件订阅者（在同一个事务里）：notification 投递 notify 任务；webhook 投递 deliver 任务
事务提交后（River 执行）：
  notify   → 为关注者和被提及的人生成站内通知
  deliver  → 向订阅了该事件的 Webhook 投递
```
- **Webhook 负载**：`{id, type: "issue.updated", occurred_at, workspace_id, actor_id, data: <和接口返回一致的资源>, changes: [...]}`。
- **Webhook 投递**：
  - 签名：HMAC-SHA256，放在 `X-Nerve-Signature` 请求头中。
  - 超时 10 秒；失败后间隔逐次加长，最多重试 5 次。
  - 每次投递都写入 `webhook_logs`；连续失败达到上限时自动停用。
  - **禁止指向内网地址（防 SSRF）。**
- **接口调用日志**（`audit` 模块）：
  - 记录所有写请求，不区分令牌类型；读请求不记录。
  - **存储前去掉令牌等敏感请求头**。
  - 由进程内协程每秒批量写入一次，保留 30 天。

### 6.7 后台任务与文件存储
**River 任务：**

| 类型 | 任务 |
|---|---|
| 事件触发 | 通知分发、Webhook 投递、抓取链接的标题和图标（防 SSRF） |
| 定时 | **自动归档**（每天）；物理删除软删除超过 60 天的数据；清理 7 天内未完成上传的文件；按保留期清理接口日志、Webhook 日志、历史版本；清理过期会话 |

River 的具体版本和成熟度在 M0 核实。

**文件存储**（`asset` 模块的存储端口）：
- **两种实现**：
  - `local`（默认）：文件存在服务器磁盘上；由 Go 进程自己用 HMAC 签发带过期时间的地址，并处理上传和下载请求。
  - `s3`：任意 S3 兼容存储，包括 AWS S3、Cloudflare R2、阿里云 OSS、腾讯云 COS，以及自建的 Garage、SeaweedFS。
- **上传流程**：
  1. `POST /api/v0/assets`：检查权限、类型和大小，写入一条未完成的记录，返回 `{method, url, headers}`（预签名 PUT，文件大小和类型都写在签名里）。
  2. 前端按返回的信息直接上传文件。
  3. `POST /api/v0/assets/{id}/complete`：服务端核实文件确实存在、大小相符，然后标记为已上传。
- **读取**：返回短期有效的签名地址，过期时间按小时取整，以便浏览器缓存。
- 两种实现共用一套存储契约测试。

### 6.8 配置与环境
按环境划分配置文件，类似 Spring Boot 的 `application-{profile}.yml`。

```
server/configs/
  config.yaml          基础配置：列出所有配置项及其默认值
  config.dev.yaml      开发环境覆盖项
  config.test.yaml     测试环境覆盖项（集成测试和端到端测试使用）
  config.prod.yaml     生产环境覆盖项
  config.local.yaml    个人本地覆盖项（不进仓库，已加入 .gitignore）
```
- **选择环境**：用环境变量 `NERVE_ENV=dev|test|prod`，默认是 `dev`。
- **加载顺序**（后加载的覆盖先加载的）：
  1. 内置的 `config.yaml`（列出全部配置项及默认值，已编进程序；不在 Go 代码中重复写默认值）
  2. `config.{env}.yaml`
  3. `config.local.yaml`（只在 dev 环境加载）
  4. 环境变量 `NERVE_*`
- **环境变量命名**：`NERVE_` 加上配置路径，层级之间用双下划线分隔。例如 `database.url` 对应 `NERVE_DATABASE__URL`，`auth.jwt.private_key_file` 对应 `NERVE_AUTH__JWT__PRIVATE_KEY_FILE`；单个 `_` 保留在键名内（`NERVE_DATABASE__MAX_CONNS` → `database.max_conns`）。不含 `__` 的 `NERVE_*` 变量不是配置键。
- **密钥不进仓库**：数据库密码、JWT 私钥、S3 密钥等，只通过环境变量或 `*_file` 形式的配置项（指向挂载进来的密钥文件）提供。仓库里的配置文件只放非敏感的值。
- **强类型、启动即校验**：配置被加载到一个强类型的结构体中，启动时逐项校验；有错误就立即退出，并明确指出是哪个配置项出了问题。启动日志会打印生效的配置，密钥打码。未知的配置键直接报错；时长类配置项只接受字符串。
- **单文件部署**：配置文件通过 `go:embed` 编进程序，所以只带一个可执行文件也能运行；也可以用 `NERVE_CONFIG_DIR` 指向外部目录，作为内置文件之上的一层，按键覆盖，目录里缺失的文件会被跳过。
- **实现**：用 koanf 分层加载（它比 viper 轻，而且没有全局状态），放在 `platform/config`。
- **test 环境的典型设置**：降低 argon2 的计算强度、缩短定时任务的间隔、文件存到临时目录、日志输出为便于阅读的文本格式。
- **前端**：不使用 Vite 的模式文件（`.env.development` 等）。前端和后端同源部署，接口地址为空即相对路径，不设置任何前端环境变量（M0/P5 已验证，见 [P5 spec](M0-foundation/specs/P5-web-import.md) 2.6）。
- 详见 [M0/P2 spec](M0-foundation/specs/P2-server-platform.md) 2.3 节。

### 6.9 其他
- **富文本安全**：描述和评论的 HTML 用 bluemonday 清洗，只放行 Plane 编辑器用到的标签；同时提取纯文本存入 `description_stripped`，供搜索使用。
- **搜索**：工作项标题建 pg_trgm 索引，支持模糊匹配；支持按编号搜索。已归档的工作项默认不出现在搜索结果中。
- **运维**：日志用 slog；prod 输出 JSON，dev 和 test 输出便于阅读的文本；提供 `/healthz` 和 `/readyz`；停机时先处理完正在进行的请求和任务。

---

## 7. 前端

### 7.1 代码来源
- **来源提交**：Plane `02c19e1341d93141e8ad7b3278298adce208bafc`（`preview` 分支）。M0/P5 已从这个提交原样迁入，之后的每一处改动登记在[前端改动清单](frontend-changes.md)。
- **使用**：Plane 的 `apps/web`，以及 packages 中的 types、constants、ui、propel、editor、i18n、hooks、utils、shared-state、tailwind-config、typescript-config。
- **暂时使用**：packages/services。web 中的令牌设置页和文件工具函数依赖它；M2（PAT）和 M5（文件）对接新接口时，将它删除。
- **不使用**：apps/admin、apps/space、apps/live、apps/api、apps/proxy、packages/logger、packages/decorators、packages/codemods（已核实 web 及其依赖的包都不引用它们）。
- **工具链**：沿用 pnpm + turbo + Vite。开发时由 Vite 把 `/api` 转发给本地 Go 服务；发布时，打包好的静态文件通过 `go:embed` 编进 Go 程序。

### 7.2 对接新接口：不设转换层，前后端数据结构统一
```
页面组件（components）  → 按需调整，跟随新的数据类型
状态管理（stores）      → 直接使用新接口的数据结构（分页、分组、错误等）
接口调用（services）    → 薄封装：只调用生成的客户端，不做任何数据转换
packages/api-client    ← 由 api/dist/openapi.yaml 生成（openapi-typescript + openapi-fetch）
packages/types         ← 实体类型（Issue、Project、State……）直接使用生成的类型；
                          只有纯界面用的类型（显示设置、布局参数等）才手写
```
- **唯一的依据是 `api/` 中的接口描述**：接口一改，重新生成类型，TypeScript 编译器会指出所有受影响的 store 和组件。
- **不写转换层**：不把新接口的数据翻译成 Plane 的旧结构，也不保留任何为兼容 Plane 旧接口、旧字段而存在的代码。
- **改动量**：因为表结构和字段名基本照搬 Plane，大部分实体类型的字段本来就一致。改动主要集中在几层外壳上：分页和分组的响应结构、错误格式、认证、文件上传、迭代和模块的归属方式。
- **职责划分**：
  - services 只负责调用接口。
  - stores 只负责管理状态。
  - 组件不直接调用接口。

### 7.3 认证
- **保留**：注册、登录、退出、修改密码。登录改为调用 `POST /api/v0/auth/login`，登录页保留原有界面，只留下邮箱和密码。
- **删除**：第三方登录、验证码登录、找回 / 重置 / 设置密码页面、登录前的"检查邮箱"步骤、CSRF 相关代码。CSRF 属于认证的传输方式，和令牌管理器一起在 M2 删除（[M1 设计](M1-frontend-trim/M1-design.md) 3.6），其余在 M1 删除。
- **新增令牌管理器**：实现 4.3 中的规则。

### 7.4 删除不需要的功能
- **删除范围**：
  - 1.2 中列出的所有功能。
  - Plane 企业版的残留，包括写着 `extended` 的空壳扩展文件和空函数。
  - Plane 自身的死代码，包括没有被使用的 IndexedDB 和同步代码、从未被创建过的集成服务，以及调用后端并不存在的接口的方法。
  - **Next.js 兼容垫片**：Plane 从 Next.js 迁到 React Router 时留下的过渡层（`app/compat/next/*`）。约 330 处引用全部改为 React Router 原生写法，然后删除垫片。
- **每个功能都要删到的层面**：路由、导航和菜单入口 → 组件、store、services、hooks → `packages/types` 中的类型和字段 → 常量和枚举 → 多语言文案 → 不再使用的依赖（如 yjs、hocuspocus、y-prosemirror、y-indexeddb、comlink，删除后逐个核实）。
- **"自动化"设置页**：只保留自动归档，删除自动关闭。
- **怎么保证删干净**：反复运行 TypeScript 类型检查和 knip，直到结果为零。
- **验收标准**：
  - 类型检查通过，knip 报告为零。
  - 用关键词清单（如 `estimate`、`page_view`、`epic`、`oauth`、`magic`、`close_in`）全文搜索，没有结果。
  - 能正常构建。

### 7.5 品牌与多语言
- **品牌**：替换 Logo、页面标题和文案中所有的"Plane"。
- **多语言**：只保留 `zh-CN` 和 `en`，删除其他 19 种语言。

### 7.6 前端代码质量要求（长期有效）
- **不保留**死代码、兼容代码、没有用处的开关，也不保留"以后可能会用"的代码。
- **TypeScript 类型检查、oxlint、knip 长期作为持续集成的门禁**，不只在 M1 执行，防止死代码重新长回来。oxfmt 的格式检查和前端构建也是持续集成的门禁（M0/P5 加入）。
- **oxlint 的警告采用"只降不升"的基线**：Plane 现有代码带着上万条警告（它自己也是按每个包的警告上限来管理的）。警告数超过基线，持续集成就失败；警告减少后，同一个提交里就把基线调低。M1 会重新测出基线，并制定逐步清零的计划。
- 新写的代码遵循 7.2 的职责划分，和 Plane 现有的写法保持一致（MobX store、`observer` 组件）。

---

## 8. 测试策略

### 8.1 各层测试

| 层次 | 内容 | 工具与做法 |
|---|---|---|
| 后端单元测试 | 领域规则（归档条件、状态流转、完成时间等）、字段变更比对、筛选解析、游标、HTML 清洗、Webhook 签名 | Go testing，表格驱动 |
| 权限矩阵测试 | "角色 × 操作"的每一格 | 表格驱动 |
| 端口契约测试 | 同一个端口的每种实现（如 `local` 和 `s3` 存储）都跑同一套测试 | Go testing |
| 后端集成测试 | 用例和 SQL，**连接真实的 Postgres** | testcontainers；每个测试从模板库复制一个独立的数据库 |
| 接口契约测试 | 请求和响应是否与 `api/dist/openapi.yaml` 一致 | `platform/httpserver/apitest`（只被测试导入）用 kin-openapi 校验；每个测试对自己的响应调用 `CheckResponse`（M0/P3） |
| 架构测试 | 依赖方向、模块边界 | 仓库内的架构测试（随 `go test` 运行）；depguard 检查禁用的库 |
| 前端单元测试 | store 的状态逻辑、令牌管理器、工具函数 | vitest |
| 前端静态检查 | 类型错误、lint 警告、未使用的代码 | TypeScript 类型检查和 knip 必须为零；oxlint 警告不得超过基线（只降不升） |
| **端到端测试** | 每个领域的主线用户故事 | 见 8.2 |

### 8.2 端到端测试（Playwright + 真实数据库断言）
- **以用户故事为单位**：每个 M 的设计文档都要列出本领域的主线用户故事清单，每个故事对应一个 Playwright 测试文件。这份清单就是这个 M 的验收范围。
- **每个故事在三个层面断言**：
  1. **页面**：用户看到的结果正确。
  2. **数据库**：直接查表，核对业务数据，以及所有附带产生的数据：操作动态、历史版本、自动关注、审计字段、软删除和归档标记、文件是否真的写入了存储。
  3. **异步结果**：通知、Webhook 投递、自动归档。用"限时轮询数据库"等待结果，不用固定的等待时间。
- **对等验收**：每个故事都有一个接口版本，用 PAT 调接口走完同样的流程，**调用同一组数据库断言函数**，要求落库结果完全一样。
- **示例**（M4）：成员在看板上把工作项从"待办"拖到"已完成"。

  | 层面 | 断言 |
  |---|---|
  | 页面 | 卡片出现在"已完成"列，列头计数更新 |
  | `issues` | `state_id` 正确，`completed_at` 已填，`sort_order` 在相邻卡片之间，`updated_by` 是当前账户 |
  | `issue_activities` | 新增一条 `field=state` 的记录，新旧状态 ID 正确，`actor_id` 是当前账户 |
  | 异步 | 关注者收到通知，订阅了事件的 Webhook 收到投递 |
  | 接口版本 | 用 PAT 调用 `PATCH /api/v0/issues/{id}` 做同样的事，跑同一组数据库断言 |

- **测试环境**：
  ```
  e2e/
    fixtures/
      server.ts    每个 Playwright 并行进程启动一个 nerve（NERVE_ENV=test，随机端口）
      db.ts        每个并行进程一个独立数据库（从模板库复制），以及数据库断言函数
      api.ts       用生成的 TS 客户端通过接口准备数据
      test.ts      把以上几个接成 Playwright 的 fixture；故事从这里导入 test、expect
      webhook.ts   本地 Webhook 接收端，记录投递并验证签名
      storage.ts   检查文件是否真的写入了本地存储
    global-setup.ts  每次运行启动一次数据库容器，迁移出模板库（M0/P6 加入）
    stories/<领域>/*.spec.ts   一个故事一个测试文件，个别故事可以有多个测试；M0 没有业务领域，故事都放在 stories/smoke/ 下（M0/P6）
  ```
  - 被测对象是**真正要发布的产物**：打包好的 `nerve` 程序（内嵌前端）、Postgres 18、本地文件存储。
  - 前置数据通过接口准备，只有被测的那一步走页面。
  - 需要"时间流逝"的故事（比如自动归档），在测试环境中通过可注入的时钟来推进，不真的等待。
  - 失败时保存操作记录（trace）和截图；M0 另外保存 nerve 的日志。录像和数据库快照从有业务表的 M 起再加入（M0/P6：trace 已含每一步的截屏，录像还要多下载 ffmpeg；M0 没有业务表）。
- **运行时机**：
  - 本地：`make e2e` 一条命令完成编译、启动数据库和运行全部故事（M0/P6 裁定：命令入口是 Makefile，不是根 `package.json` 的脚本）。
  - **每次提交 PR 都运行完整的端到端测试。**
  - 完成一个 M，要求本 M 的所有故事通过，**并且之前所有 M 的故事也都通过**。

### 8.3 持续集成
生成物一致性检查、golangci-lint（含 depguard）、Go 测试（含 Postgres 和架构测试）、TypeScript 类型检查、oxlint（按基线）、oxfmt 的格式检查、前端构建（M0/P5 加入）、knip（M0 只出报告，M1 起作为门禁）、vitest、Playwright 端到端测试（M0/P6 加入骨架和冒烟故事）。

---

## 9. 分期路线

### 9.1 依赖关系
```
M0 基础骨架 ─┬─→ M2 账户认证 → M3 工作区与项目 → M4 工作项核心 → M5 文件 ─→ ★ Demo 可用
M1 前端瘦身 ─┘                                                      │
                                                  M6 迭代与模块 ←────┘
                                                  M7 协作功能
                                                  M8 开放与发布
```
M0 和 M1 可以同时进行。M2 之后按顺序推进。

### 9.2 各里程碑

| M | 目录 | 内容 | 完成标志 |
|---|---|---|---|
| M0 | `M0-foundation` | 按第 6 节搭好 Go 项目骨架（bootstrap、platform、模块模板、分环境配置、goose 迁移机制、problem+json、oapi-codegen 生成流程、架构测试）；试点模块 `instance`；Plane 表结构快照；前端代码原样迁入 `web/`、能构建并内嵌进程序；docker compose 开发环境；持续集成；端到端测试骨架。**不建业务表，不接入 River 和 sqlc** | 启动 `nerve` 后健康检查正常，内嵌的前端页面能打开，`GET /api/v0/instance` 可用（冒烟测试）；各项门禁在持续集成中生效。详见 [M0 设计文档](M0-foundation/M0-design.md) |
| M1 | `M1-frontend-trim` | 按第 7.4 节彻底删除不要的功能；Next.js 兼容垫片改为 React Router 原生写法；替换品牌（包括包名改为 `@nerve/*`）；只保留中英文；重新测出 oxlint 警告基线并制定清零计划 | 类型检查和 knip 为零，oxlint 不超过新基线；关键词搜索没有结果；能正常构建（不依赖后端，只做静态验收） |
| M2 | `M2-auth` | 注册、登录、续期、退出、修改密码、PAT、个人资料与偏好、实例配置（在 M0 的 `instance` 上扩展）、管理命令；前端令牌管理器和登录页；**首次接入 River（第一个定时任务：清理过期会话）和 sqlc（第一批查询）** | 本领域的用户故事全部通过端到端测试（含 PAT 对等验收） |
| M3 | `M3-workspace-project` | 工作区、成员、邀请；项目、项目成员、**项目归档**；状态、标签、个人显示设置；**权限框架和权限矩阵** | 同上 |
| M4 | `M4-issue-core` | 工作项增删改查；**列表引擎**（筛选、分组、子分组、排序、游标）；子任务、关联、链接、评论、表情回应、操作动态、搜索、历史版本、草稿；**工作项归档、恢复和自动归档** | 同上（工作量最大，实施时可能再拆分） |
| M5 | `M5-files` | 存储端口（local 和 s3）；附件、编辑器图片、头像、图标、封面 | 同上。**到此 Demo 可用** |
| M6 | `M6-cycles-modules` | 迭代（进度、燃尽图、转移工作项）、模块（链接、成员）；**迭代和模块的归档** | 同上 |
| M7 | `M7-collaboration` | 站内通知（关注、@提及）、需求收集箱、保存的视图和跨项目列表、收藏、最近访问 | 同上 |
| M8 | `M8-open-release` | Webhook、接口调用日志、对外接口文档页；完整的对等验收；Docker 镜像和单个可执行文件的发布；备份文档；内存实测（目标：Go 进程空闲时低于 50 MB） | 同上，并发布 v0 |

### 9.3 所有 M 通用的完成标准
- 本 M 的后端测试、前端单元测试和端到端用户故事全部通过；之前所有 M 的端到端测试也都通过。
- 本 M 范围内的所有功能都能用 PAT 通过接口完整操作。
- 本 M 涉及的表由本 M 的迁移创建，以 Plane 表结构快照为起点；与 Plane 的差异已登记。
- `api/` 中的接口描述随本 M 一起扩展，先写接口描述，再写代码。
- 架构测试、depguard 和前端静态检查（类型检查、knip 为零，oxlint 不超过基线）全部通过。
- 本 M 不再留有 `open` 的 handoff。
- [差异清单](plane-diff.md)和[前端改动清单](frontend-changes.md)已同步更新。

### 9.4 里程碑进度表

| M | 名称 | 状态 | 设计文档 |
|---|---|---|---|
| M0 | 基础骨架 | 已完成 | [M0-design.md](M0-foundation/M0-design.md) |
| M1 | 前端瘦身 | 进行中 | [M1-design.md](M1-frontend-trim/M1-design.md) |
| M2 | 账户认证 | 未开始 | — |
| M3 | 工作区与项目 | 未开始 | — |
| M4 | 工作项核心 | 未开始 | — |
| M5 | 文件 | 未开始 | — |
| M6 | 迭代与模块 | 未开始 | — |
| M7 | 协作功能 | 未开始 | — |
| M8 | 开放与发布 | 未开始 | — |

---

## 10. 风险与留待后续确定的事项

| 事项 | 在哪里确定 |
|---|---|
| 列表引擎的复杂度（分组、子分组、多值分组、游标和多种排序的组合）可能超出预期 | M4 设计文档；必要时把 M4 拆成两个 M |
| 操作动态的事件类型和格式：Plane 有 27 种，前端的渲染依赖这些格式 | M4 设计文档，逐一对照 Plane 的实现 |
| 前端对接新接口时，store 和组件的实际改动量 | 各 M 的设计文档中评估；M2 是第一个对接的领域，用它来校准后续的估算 |
| 通知的生成规则：谁在什么情况下会收到通知 | M7 设计文档，对照 Plane 的 `notification_task` |
| 依赖的版本和成熟度（2026-09-22 已核实，见 M0 设计文档第 1 节）；OpenAPI 3.1 工具链是否可用 | M0/P3 验证；River 在 M2 接入时锁定版本 |
| 各接口的限流数值、各类数据的保留期 | 相关 M 的设计文档 |
| 没有邮件服务时账户如何找回（目前只能由管理员用命令行重置） | 以后接入邮件服务时再议（v0 之后） |
