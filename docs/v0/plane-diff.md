# 与 Plane 的差异清单

基线：Plane v1.4.2，提交 `02c19e1`。本清单记录 Nerve 在表结构、接口和行为上与 Plane 的每一处差异，在整个 v0 期间持续更新。

- 设计阶段已确定的差异在下文中列出。
- 列级别的细节（每张表逐列核对）在 **M0** 生成 `0001_init.sql` 时补全；之后各个 M 的变更，由该 M 负责登记。

---

## 一、未保留的表

Plane 共有 96 张业务表（`db` 应用 92 张，`license` 应用 4 张）。Nerve 保留 44 张，其余 52 张不保留。Django 和 Celery 的系统表（`django_migrations`、`django_content_type`、`auth_group`、`auth_permission`、`users_groups`、`users_user_permissions`、`django_celery_beat_*`）同样不保留。

### A. Plane 社区版本来就用不上（21 张）

| 表 | 情况 |
|---|---|
| `integrations`、`workspace_integrations`、`github_repositories`、`github_repository_syncs`、`github_issue_syncs`、`github_comment_syncs`、`slack_project_syncs` | 集成功能只有表，没有接口 |
| `importers` | 导入器只有表，没有接口 |
| `teams`、`devices`、`device_sessions`、`social_login_connections` | 死表，代码中没有引用 |
| `issue_blockers`、`issue_attachments`、`project_deploy_boards`、`project_webhooks`、`description_versions` | 旧版遗留，早已被新表取代 |
| `analytic_views` | 有接口，但前端从不调用 |
| `issue_types`、`project_issue_types` | 企业版功能的占位表 |
| `changelogs` | 更新日志 |

### B. 被新架构替代（7 张）

| 表 | 替代方式 |
|---|---|
| `sessions` | 改用 JWT 和 `auth_sessions` |
| `instances`、`instance_admins`、`instance_configurations` | 改用分环境的配置文件（可用环境变量覆盖）；服务器管理员通过命令行管理 |
| `issue_sequences` | 改为 `projects` 上的计数列 |
| `project_identifiers` | 和 `projects.identifier` 重复 |
| `descriptions` | 只是评论内容的镜像副本 |

### C. 砍掉的功能（24 张）

| 功能 | 表 |
|---|---|
| 文档页 | `pages`、`project_pages`、`page_labels`、`page_logs`、`page_versions` |
| 估算 | `estimates`、`estimate_points` |
| 导出 | `exporters` |
| 项目邀请 | `project_member_invites` |
| 邮件通知 | `email_notification_logs`、`user_notification_preferences`（Plane 的通知偏好只用于决定是否发邮件，站内通知不受它影响） |
| 第三方登录 | `accounts` |
| 首页个性化与主题 | `workspace_home_preferences`、`workspace_user_links`、`workspace_user_preferences`、`workspace_themes` |
| 便签 | `stickies` |
| 公开发布 | `deploy_boards`、`project_public_members`、`issue_votes` |
| 草稿的关联表 | `draft_issue_assignees`、`draft_issue_labels`、`draft_issue_modules`、`draft_issue_cycles`（并入 `draft_issues.payload`） |

---

## 二、保留表的改动

### 全局
| 改动 | 原因 |
|---|---|
| 删除所有表的 `external_source`、`external_id` | 只有导入器使用这两列，v0 不做导入 |
| 默认值、非空约束、枚举检查（优先级、状态组、角色、邀请状态、收集箱状态等）写进数据库 | Plane 只在 Python 代码中处理 |
| 保留 Plane 的部分唯一索引写法（`WHERE deleted_at IS NULL`）；去掉与之配套的 `unique_together (..., deleted_at)` | 在 Postgres 中 NULL 与 NULL 不相等，这种约束管不住未删除的数据，几乎不起作用 |
| ID 改由应用生成 UUIDv7（列类型仍然是 `uuid`） | 让索引更紧凑 |

### 按表
| 表 | 改动 | 原因 |
|---|---|---|
| `users` | 删除 `is_bot`、`bot_type` | 核心原则：不区分人和机器人 |
| `users` | 删除旧的头像和封面 URL 列（`avatar`、`cover_image`），改用 `*_asset_id` | 遗留列 |
| `profiles` | 删除账单字段和移动端字段 | Plane 云服务专用或已废弃 |
| `api_tokens` | `token` 改为 `token_hash`；删除 `user_type` | 不存令牌原文；不区分人和机器人 |
| `auth_sessions` | **新增**：刷新令牌的哈希、令牌轮换链（用于重复使用检测）、UA、IP、过期和撤销时间 | JWT 认证 |
| `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
| `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
| `projects` | 删除 `emoji`、`icon_prop`、旧的 `cover_image`、`description_text`、`description_html`（旧的 json 列）、`page_view`、`is_time_tracking_enabled`、`is_issue_type_enabled`、`estimate_id`、`close_in` | 遗留列或对应功能已砍掉（归档保留，`archive_in` 和 `archived_at` 保留） |
| `projects` | **新增**工作项编号计数列（列名在 M0 确定） | 替代 `issue_sequences` |
| `project_members` | 删除 `view_props`、`default_props`、`preferences` | 和 `project_user_properties` 重复 |
| `issues` | 删除 `point`、`is_draft`、`estimate_point_id`、`type_id`、`description_binary` | 遗留列或对应功能已砍掉 |
| `issues` | `archived_at` 由 `date` 改为 `timestamptz` | 和 `cycles`、`modules`、`projects` 的 `archived_at` 保持一致 |
| `issues` | **新增**唯一约束 `(project_id, sequence_id)` | Plane 只靠咨询锁保证编号不重复 |
| `issue_labels` | **新增**部分唯一约束 `(issue_id, label_id)` | Plane 在数据库层没有这个约束 |
| `labels` | 工作区级标签的名称唯一范围改为 `(workspace_id, name)` | Plane 的约束没有限定在工作区内，是个缺陷 |
| `cycle_issues` | **新增**部分唯一约束 `(issue_id)`：一个工作项最多属于一个迭代 | Plane 只在代码里检查 |
| `modules` | 删除旧的 `description_text`、`description_html` | 遗留列 |
| `*_user_properties`、`issue_views` | 删除旧的 `filters` 列；筛选条件按 Nerve 自己的格式存储（列名与格式在 M3/M4 确定）；`issue_views` 删除 `query` 列 | 筛选格式改变（见接口差异） |
| `file_assets` | 删除 `page_id`；`draft_issue_id` 改为引用新的 `draft_issues`；`entity_type` 去掉 `PAGE_DESCRIPTION` | 文档页已砍掉；草稿重新设计 |
| `draft_issues` | **重新设计**：`payload jsonb` 存"创建工作项"的请求体，替代 Plane 中复制工作项结构的字段和 4 张关联表 | 详见 v0-design 5.4 |

---

## 三、接口差异

Nerve 不兼容 Plane 的 `/api/`、`/auth/`、`/api/v1/`、`/api/public/`、`/api/instances/`，而是重新设计了一套 `/api/v0`（接口版本号与产品主版本号一致），规范见 [v0-design 第 3 节](v0-design.md#3-接口设计规范)。和 Plane 相比，主要区别如下：

| 方面 | Plane | Nerve |
|---|---|---|
| 版本 | 页面用的内部接口 `/api/` 不带版本；公开接口是 `/api/v1/` | 只有一套接口 `/api/v0`，版本号跟随产品主版本号 |
| 路径 | `/api/workspaces/{slug}/projects/{pid}/issues/{id}/` 这样的多层嵌套，结尾必须带 `/` | 列表挂在父资源下，单个资源用短路径；结尾不带 `/` |
| 认证 | 页面用会话 Cookie 加表单登录，程序用另一套 `/api/v1` 加 `X-Api-Key` | 所有调用方都使用同一套接口和 `Authorization: Bearer` |
| PATCH 的响应 | 204，没有响应体 | 返回完整资源 |
| 筛选 | JSON 筛选树和 25 种旧查询参数并存 | 普通查询参数 |
| 分页 | `每页条数:页码:是否上一页` 形式的偏移游标 | 不透明游标 |
| 迭代和模块归属 | 通过单独的接口设置 | 作为工作项字段，用 PATCH 修改 |
| 错误 | `{"error": "..."}`、DRF 字段错误、认证错误码三种格式混用 | 统一使用 RFC 9457 |
| 无权限 | 403 | 看不到的资源返回 404，看得到但没权限返回 403 |

---

## 四、行为差异

| 行为 | Plane | Nerve |
|---|---|---|
| 写操作动态 | Celery 异步写入 | 与业务写入在同一事务中同步写入 |
| 软删除的连带删除 | Celery 异步执行 | 在同一事务中同步执行 |
| 最近访问 | 在 GET 请求中顺带写入 | 由前端显式调用接口记录，GET 请求没有副作用 |
| 通知偏好 | 决定是否发送邮件 | v0 没有邮件，不设通知偏好；站内通知的生成规则与 Plane 一致 |
| Webhook 负载 | Plane 格式 | Nerve 格式，使用 HMAC-SHA256 签名，禁止指向内网地址 |
| 接口调用日志 | 只记录 `/api/v1` 中用 API Key 发起的请求，请求头原样存储 | 记录所有写请求，不区分令牌类型；存储前去掉敏感请求头 |
| 工作项编号 | 咨询锁加 `issue_sequences` 表 | 在项目行上用计数列原子自增 |
| 草稿发布 | 专门的转换逻辑 | 与"创建工作项"走同一代码路径 |
| 归档 | 工作项、迭代、模块、项目的归档与恢复，以及项目级自动归档 | 规则一致（见 v0-design 5.5）；归档和恢复改为 `POST .../archive` 和 `POST .../unarchive` 两个动作接口；列表通过 `?archived=true` 查询已归档的对象 |
| 自动关闭 | 项目设置 `close_in` 后，长期未更新的未完成工作项会被自动关闭 | v0 不做 |
| 忘记密码 | 发邮件重置 | 服务器管理员用命令行重置 |
