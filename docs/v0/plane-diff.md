# 与 Plane 的差异清单

基线：Plane v1.4.2，提交 `02c19e1`（`preview` 分支，`package.json` 中的版本是 1.4.2；`v1.4.2` 标签指向 `5f7d927`，两者的迁移文件逐字节相同，见 [`tools/plane-schema/README.md`](../../tools/plane-schema/README.md)）。本清单记录 Nerve 在表结构、接口和行为上与 Plane 的每一处差异，在整个 v0 期间持续更新。

- 设计阶段已确定的差异在下文中列出。
- 列级别的细节（每张表逐列核对），由**建这张表的 M** 在编写迁移时补全，起点是 M0/P4 生成的 Plane 表结构快照 [`tools/plane-schema/plane-v1.4.2-schema.sql`](../../tools/plane-schema/plane-v1.4.2-schema.sql)。之后的变更，也由改动它的 M 负责登记。

---

## 一、未保留的表

Plane 共有 96 张业务表（`db` 应用 92 张，`license` 应用 4 张）。Nerve 保留 44 张，其余 52 张不保留。Django 和 Celery Beat 的 14 张系统表（`django_migrations`、`django_content_type`、`django_session`、`auth_group`、`auth_group_permissions`、`auth_permission`、`users_groups`、`users_user_permissions`，以及 6 张 `django_celery_beat_*`）同样不保留。96 + 14 = 110，正是表结构快照中的全部表（M0/P4 核对）。

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
| `sessions` | 替换模型，不逐列继承：Django 的会话存储由 JWT 访问令牌和 `auth_sessions`（一次登录一行，见二·按表）替代（M2 设计 4.5） |
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
| 默认值、非空约束、枚举检查（优先级、状态组、角色、邀请状态、收集箱状态等）写进数据库 | Plane 只在 Python 代码中处理：快照中没有任何列默认值，CHECK 约束只有 18 个由 Django 正整数字段生成的 `>= 0` |
| 保留 Plane 的部分唯一索引写法（`WHERE deleted_at IS NULL`）；去掉与之配套的 `unique_together (..., deleted_at)` | 在 Postgres 中 NULL 与 NULL 不相等，这种约束管不住未删除的数据，几乎不起作用 |
| ID 改由应用生成 UUIDv7（列类型仍然是 `uuid`） | 让索引更紧凑 |
| 外键写上 `ON DELETE`，照搬每个外键在 Django 模型中的 `on_delete`：`CASCADE` → `ON DELETE CASCADE`，`SET_NULL` → `ON DELETE SET NULL`，`PROTECT` → `ON DELETE RESTRICT`，`DO_NOTHING` 不写；不用 `DEFERRABLE` | Plane 在 Python 中处理级联，快照中 474 个外键都是 `DEFERRABLE INITIALLY DEFERRED`、没有 `ON DELETE`；Nerve 在同一个事务里按先父后子的顺序写（M2 设计 3.13，删除关系图见 M2 设计 4.7） |
| 约束和索引一律改名：主键、外键、唯一约束和一列唯一的 CHECK 用 Postgres 的默认名（`<表>_pkey`、`<表>_<列>_fkey`、`<表>_<列>_key`、`<表>_<列>_check`）；涉及多列的 CHECK 显式命名为 `<表>_<含义>_check`；索引命名为 `<表>_<列>_idx` | 快照中的名字带 Django 的哈希后缀，29 张表的主键名与表名不对应；按约束名映射错误、以后 `DROP CONSTRAINT` 都需要稳定的名字（M2 设计 3.13） |
| 不建 `*_like`（`varchar_pattern_ops`）索引；外键列只在查询或级联用到时建索引，`created_by_id`、`updated_by_id` 不建 | 前者只服务 Django 的 `LIKE 'x%'`；后者是 Django 给每个外键默认建的，多数用不上（M2 设计 3.13） |
| 审计时间列 `created_at`、`updated_at` 由应用按用例的时钟显式写入；`DEFAULT now()` 是新加的兜底，只在应用之外写入时起作用 | 时间只有一个来源，测试用固定时钟；自动归档按 `updated_at` 判断（M2 设计 3.13） |
| 对象型的 `jsonb` 列用 CHECK 保证是对象、键的集合和每个值的类型 | Plane 只在 Python 中保证（M2 设计 3.13） |

### 按表
| 表 | 改动 | 原因 |
|---|---|---|
| `users` | 40 列保留 10 列（M2/P1，`00001_identity_users.sql`）；主键名 `users_pkey`、唯一约束名 `users_email_key`（Plane 是 `user_pkey`、`user_email_key`） | M2 设计 4.2 |
| `users` | `email`：改为 `NOT NULL`（Plane 可为空）；新加 `CHECK (email = lower(email) AND email !~ '[[:space:]]')` | 规范化原来只在 Python 里做 |
| `users` | `password`：列名和类型照搬，内容改为 argon2id 的 PHC 字符串 | 密码哈希改用 argon2id（M2 设计 3.8） |
| `users` | `first_name`、`last_name`：新加 `DEFAULT ''` | 模型的默认值 |
| `users` | `display_name`：新加 `CHECK (display_name <> '')`；注册时取邮箱 @ 之前的部分 | Plane 由 `User.save()` 填上，数据库不约束 |
| `users` | `user_timezone`：新加 `DEFAULT 'UTC'` | 模型的默认值（取值范围的差异由 M2/P3 登记在第四节） |
| `users` | `is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `users` | 删除 Django 与管理后台的列 `last_login`、`is_superuser`、`is_staff`、`username`，以及与 `created_at` 重复的 `date_joined` | Nerve 没有 Django 的管理后台；`username` 前端从不读取 |
| `users` | 删除登录记录 `last_login_time`、`last_logout_time`、`last_login_ip`、`last_logout_ip`、`last_login_medium`、`last_login_uagent`、`last_active`、`token`、`token_updated_at` | 改由 `auth_sessions` 承担 |
| `users` | 删除 `is_password_autoset`、`is_email_verified`、`is_email_valid`、`mobile_number`、`last_location`、`created_location`、`is_managed`、`is_password_expired`、`is_password_reset_required`、`masked_at` | 第三方登录、验证码登录、邮箱验证已砍掉；其余在 Plane 代码中已不使用 |
| `users` | 删除 `is_bot`、`bot_type` | 核心原则：不区分人和机器人 |
| `users` | 删除旧的头像和封面 URL 列（`avatar`、`cover_image`）；`avatar_asset_id`、`cover_image_asset_id` 暂不建，由 M5 随文件存储加入（迁移归 `identity`） | 遗留列；在那之前接口的 `avatar_url`、`cover_image_url` 是 `null` |
| `profiles` | 29 列保留 11 列（M2/P1，`00002_identity_profiles.sql`） | M2 设计 4.3 |
| `profiles` | `user_id`：唯一约束照搬（`OneToOne`），加上 `ON DELETE CASCADE` | 二·全局 |
| `profiles` | `theme`：`jsonb`（默认 `{}`，存自定义调色板）改为 `varchar(20) NOT NULL DEFAULT 'system'`，`CHECK (theme IN ('system','light','dark','light-contrast','dark-contrast'))` | 自定义主题已删除（M1），只剩一个值 |
| `profiles` | `is_tour_completed`、`is_onboarded`：新加 `DEFAULT false` | 模型的默认值 |
| `profiles` | `onboarding_step`：新加默认值（四个键都是 `false`）和 CHECK（恰好四个键，值都是布尔值） | 模型的 `get_default_onboarding()`；Plane 只在 Python 中保证 |
| `profiles` | `language`：新加 `DEFAULT 'en'` 和 `CHECK (language IN ('en','zh-CN'))` | Nerve 只有两种语言；Plane 接受任意字符串 |
| `profiles` | `start_of_the_week`：新加 `DEFAULT 0`；CHECK 由 `>= 0` 改为 `BETWEEN 0 AND 6` | 模型的 choices |
| `profiles` | `created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `profiles` | 删除 `role`、`use_case` | 新手引导的"角色""用途"两步已删除（M2 设计 3.19） |
| `profiles` | 删除账单和移动端字段 `billing_address_country`、`billing_address`、`has_billing_address`、`company_name`、`is_mobile_onboarded`、`mobile_onboarding_step`、`mobile_timezone_auto_set` | Plane 云服务专用或已废弃 |
| `profiles` | 删除 `is_smooth_cursor_enabled`、`is_app_rail_docked`、`background_color`、`goals`、`is_navigation_tour_completed`、`product_tour`、`notification_view_mode`、`has_marketing_email_consent`、`is_subscribed_to_changelog` | 前端不用的 Plane 新功能，以及营销邮件、更新日志 |
| `api_tokens` | `token` 改为 `token_hash`；删除 `user_type` | 不存令牌原文；不区分人和机器人 |
| `auth_sessions` | **新增**（M2/P1，`00003_identity_auth_sessions.sql`，替代 `sessions`，见一 B）：一次登录一行，12 列：`id`（访问令牌中的 `sid`）、`user_id`（`ON DELETE CASCADE`）、`token_hash`（当前一代刷新令牌密文的 SHA-256，32 字节）、`generation`（代数）、`user_agent`、`ip`（`inet`）、`expires_at`（登录时刻加会话期限，之后不变）、`last_refreshed_at`、`revoked_at`、`revoke_reason`（六个取值）、`created_at`、`updated_at`；`auth_sessions_revoked_consistent_check` 要求 `revoked_at` 与 `revoke_reason` 同时为空或同时有值；索引 `auth_sessions_user_id_idx`、`auth_sessions_expires_at_idx`。旧代的刷新令牌不存，由令牌里的 HMAC 标签认出 | JWT 认证；刷新令牌的轮换和重复使用检测（M2 设计 3.4、3.5、4.5） |
| `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
| `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
| `projects` | 删除 `emoji`、`icon_prop`、旧的 `cover_image`、`description_text`、`description_html`（旧的 json 列）、`page_view`、`is_time_tracking_enabled`、`is_issue_type_enabled`、`estimate_id`、`close_in` | 遗留列或对应功能已砍掉（归档保留，`archive_in` 和 `archived_at` 保留） |
| `projects` | **新增**工作项编号计数列（列名在 M3 建表时确定） | 替代 `issue_sequences` |
| `project_members` | 删除 `view_props`、`default_props`、`preferences` | 和 `project_user_properties` 重复 |
| `issues` | 删除 `point`、`is_draft`、`estimate_point_id`、`type_id`、`description_binary` | 遗留列或对应功能已砍掉 |
| `issues` | `archived_at` 由 `date` 改为 `timestamptz` | 和 `cycles`、`modules`、`projects` 的 `archived_at` 保持一致 |
| `issues` | **新增**唯一约束 `(project_id, sequence_id)` | Plane 只靠咨询锁保证编号不重复 |
| `issues` | **新增** `name` 的 pg_trgm 索引（需要 `pg_trgm` 扩展） | 标题模糊搜索（v0-design 6.9）；快照中没有任何扩展，`issues` 上只有外键列的 btree 索引 |
| `issue_labels` | **新增**部分唯一约束 `(issue_id, label_id)` | Plane 在数据库层没有这个约束 |
| `issue_comments` | 删除 `description_id` 及其唯一约束 | 它是指向 `descriptions`（不保留，见一 B）的一对一外键 |
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
| 错误码 | 接口描述中没有 | 每个操作在接口描述中用 `x-problem-codes` 声明它可能返回的错误码；字段错误带 `code`（M2 设计 3.11） |
| 请求体 | DRF 的序列化器忽略未知字段 | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段：400 `bad_request`，一次列出全部问题（M2 设计 3.11） |

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
| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行；修改密码、创建账户和重置密码两个命令随 M2/P3 加入 |
| 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2/P3 加入）（M2 设计决策点 2） |
| 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（M2 设计 3.11） |
| 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`，带 `Retry-After: 1`（M2 设计 3.8） |
| 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401 `identity.invalid_credentials`，耗时也相同：对一个启动时生成的假哈希做一次同样参数的校验（M2 设计 3.9） |
| 会话的期限 | Django 会话，从登录起固定 7 天（`SESSION_COOKIE_AGE`），请求不延长 | 访问令牌 15 分钟；会话从登录起 30 天，续期不延长；刷新令牌每次使用后换新，并检测重复使用；退出只结束当前这一处登录（M2 设计 3.5） |
| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 进程内的令牌桶，每个桶有速率和突发：匿名按 IP、已认证按凭证、登录按 IP 和"IP + 邮箱"、注册按 IP，认证之前另有按 IP 的失败闸门；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*`（M2 设计 3.10）。修改密码按账户的桶随 M2/P3 加入 |
