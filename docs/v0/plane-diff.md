# 与 Plane 的差异清单

基线：Plane v1.4.2，提交 `02c19e1`（`preview` 分支，`package.json` 中的版本是 1.4.2；`v1.4.2` 标签指向 `5f7d927`，两者的迁移文件逐字节相同，见 [`tools/plane-schema/README.md`](../../tools/plane-schema/README.md)）。本清单记录 Nerve 在表结构、接口和行为上与 Plane 的每一处差异，在整个 v0 期间持续更新。

- 设计阶段已确定的差异在下文中列出。
- 列级别的细节（每张表逐列核对），由**建这张表的 M** 在编写迁移时补全，起点是 M0/P4 生成的 Plane 表结构快照 [`tools/plane-schema/plane-v1.4.2-schema.sql`](../../tools/plane-schema/plane-v1.4.2-schema.sql)。之后的变更，也由改动它的 M 负责登记。

---

## 一、未保留的表

Plane 共有 96 张业务表（`db` 应用 92 张，`license` 应用 4 张）。Nerve 保留 43 张，其余 53 张不保留；加上新增的 `auth_sessions`，Nerve 共 44 张业务表。Django 和 Celery Beat 的 14 张系统表（`django_migrations`、`django_content_type`、`django_session`、`auth_group`、`auth_group_permissions`、`auth_permission`、`users_groups`、`users_user_permissions`，以及 6 张 `django_celery_beat_*`）同样不保留。96 + 14 = 110，正是表结构快照中的全部表（M0/P4 核对）。

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

### C. 砍掉的功能（25 张）

| 功能 | 表 |
|---|---|
| 文档页 | `pages`、`project_pages`、`page_labels`、`page_logs`、`page_versions` |
| 估算 | `estimates`、`estimate_points` |
| 工作项快照 | `issue_versions`（M4/P4，决策点 5）；仅保留描述修改历史 |
| 导出 | `exporters` |
| 项目邀请 | `project_member_invites` |
| 邮件通知 | `email_notification_logs`、`user_notification_preferences`（Plane 的通知偏好只用于决定是否发邮件，站内通知不受它影响） |
| 第三方登录 | `accounts` |
| 首页个性化与主题 | `workspace_home_preferences`、`workspace_user_links`、`workspace_user_preferences`、`workspace_themes` |
| 便签 | `stickies` |
| 公开发布 | `deploy_boards`、`project_public_members`、`issue_votes` |
| 草稿的关联表 | `draft_issue_assignees`、`draft_issue_labels`由 `draft_issues.payload` 数组替代；`draft_issue_modules`、`draft_issue_cycles`不保留，P8 payload不含模块/迭代 |

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
| `users` | `user_timezone`：新加 `DEFAULT 'UTC'` | 模型的默认值（取值范围的差异见第四节） |
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
| `api_tokens` | 17 列保留 12 列（M2/P3a，`00004_identity_api_tokens.sql`） | M2 设计 4.4 |
| `api_tokens` | `token`（令牌原文）改为 `token_hash bytea NOT NULL UNIQUE`：整个令牌（`nrv_pat_` 加 43 个字符）的 SHA-256，新加 `CHECK (octet_length(token_hash) = 32)`；令牌原文只在创建时返回一次 | 不存令牌原文 |
| `api_tokens` | `label`：新加 `CHECK (label <> '')`；不传时由应用生成 32 位十六进制（Plane 的 `uuid4().hex`） | Plane 只在 Python 中生成 |
| `api_tokens` | `description`：新加 `DEFAULT ''`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `api_tokens` | `user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `api_tokens` | `expired_at`（空表示永不过期）、`last_used`、`deleted_at` 照搬；撤销就是软删除 | — |
| `api_tokens` | 删除 `user_type`、`workspace_id`、`is_active`、`is_service`、`allowed_rate_limit` | 不区分人和机器人；Plane 自己已把个人令牌的 `workspace_id` 置空；撤销用软删除，`is_active` 没有独立的写入方；社区版不创建服务令牌；没有限流读取 `allowed_rate_limit` |
| `api_tokens` | 索引 `api_tokens_user_id_created_at_idx ON (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` | 列表的游标分页（M2 设计 3.12） |
| `auth_sessions` | **新增**（M2/P1，`00003_identity_auth_sessions.sql`，替代 `sessions`，见一 B）：一次登录一行，12 列：`id`（访问令牌中的 `sid`）、`user_id`（`ON DELETE CASCADE`）、`token_hash`（当前一代刷新令牌密文的 SHA-256，32 字节）、`generation`（代数）、`user_agent`、`ip`（`inet`）、`expires_at`（登录时刻加会话期限，之后不变）、`last_refreshed_at`、`revoked_at`、`revoke_reason`（六个取值）、`created_at`、`updated_at`；`auth_sessions_revoked_consistent_check` 要求 `revoked_at` 与 `revoke_reason` 同时为空或同时有值；索引 `auth_sessions_user_id_idx`、`auth_sessions_expires_at_idx`。旧代的刷新令牌不存，由令牌里的 HMAC 标签认出 | JWT 认证；刷新令牌的轮换和重复使用检测（M2 设计 3.4、3.5、4.5） |
| River 的表 | **新增**的基础设施表（M2/P3b，`00005_river_main_v2_to_v7.sql`）：`river_job`、`river_leader`（`UNLOGGED`）、`river_queue`、`river_notification`，枚举 `river_job_state`，函数 `river_job_state_in_bitmask`。内容是 River v0.47.0 主线第 2–7 版迁移的原样导出（`river migrate-get --line main --all --exclude-version 1`），不建 `river_migration`，版本由 goose 管理；表、约束和索引的名字随 River，不按二·全局的约定改 | 后台任务和定时任务改用 River，与业务数据同库，替代 Plane 的 Celery 和 Celery Beat（v0 总体设计 5.2、6.7；M2 设计 3.15） |
| `workspaces` | 14 列保留 10 列（M3/P1，`00006_workspace_workspaces.sql`） | M3 设计 4.2 |
| `workspaces` | `name`：新加 `CHECK (name <> '')`；"1–80 个字符、至少一个字母或数字、不含网址"在领域层 | Plane 只在序列化器中检查 |
| `workspaces` | `slug`：新加 `CHECK (slug ~ '^[a-z0-9_-]+$')`，只有小写；全表唯一的 `workspace_slug_key` 改为部分唯一索引 `workspaces_slug_key ON (slug) WHERE deleted_at IS NULL`（不同于二·全局去掉的 `(…, deleted_at)` 一类：Plane 这里是全表唯一，删除工作区时把 slug 改名腾出它），删除不再改名 | 页面只收小写；部分唯一索引下删除之后 slug 可以重用（M3 设计 3.10） |
| `workspaces` | `organization_size`：新加 CHECK，取值是前端的六个选项（`Just myself`、`2-10`、`11-50`、`51-200`、`201-500`、`500+`） | Plane 的服务端不检查 |
| `workspaces` | `timezone`：新加 `DEFAULT 'UTC'`；取值在领域层按 IANA 名称校验 | 模型的默认值（M3 设计 3.13） |
| `workspaces` | `created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局 |
| `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
| `workspaces` | 删除 `owner_id`、`background_color`；`logo_asset_id` 暂不建，由 M5 随文件存储加入 | `owner_id` 与 `created_by_id` 重复、没有读取者，它的 `CASCADE` 会随账户删掉工作区（M3 设计 3.15）；`background_color` 前端不读；在 M5 之前接口的 `logo_url` 是 `null` |
| `workspace_members` | 17 列保留 10 列（M3/P1，`00007_workspace_workspace_members.sql`） | M3 设计 4.3 |
| `workspace_members` | `workspace_id`、`member_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `workspace_members` | `role`：`CHECK (role >= 0)` 收紧为 `CHECK (role IN (5, 15, 20))`，新加 `DEFAULT 5` | 只有三种角色（M3 设计 3.4） |
| `workspace_members` | `is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `workspace_members` | 部分唯一索引 `workspace_members_workspace_id_member_id_key ON (workspace_id, member_id) WHERE deleted_at IS NULL` 照搬；新加 `workspace_members_member_id_idx ON (member_id) WHERE deleted_at IS NULL` 和不带条件的 `workspace_members_workspace_id_idx ON (workspace_id)` | 我的工作区按账户查；物理级联要不带条件的索引（M3 设计 4） |
| `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
| `workspace_members` | 删除 `issue_props`、`company_role`、`getting_started_checklist`、`tips`、`explored_features` | 前端不读不写；公司角色随 M2 删掉的新手引导步骤没有了写入方；后三项是 Plane 已砍功能的状态 |
| `workspace_user_properties` | 14 列保留 10 列（M3/P2，`00008_workspace_workspace_user_properties.sql`） | M3 设计 4.5 |
| `workspace_user_properties` | `workspace_id`、`user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `workspace_user_properties` | `navigation_project_limit`：新加 `DEFAULT 10`、`CHECK (navigation_project_limit >= 0)`；`navigation_control_preference`：新加 `DEFAULT 'ACCORDION'`、`CHECK (navigation_control_preference IN ('ACCORDION', 'TABBED'))`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 choices；0 表示显示全部项目 |
| `workspace_user_properties` | 部分唯一索引照搬，改名为 `workspace_user_properties_workspace_id_user_id_key ON (workspace_id, user_id) WHERE deleted_at IS NULL`（Plane 叫 `workspace_user_properties_unique_workspace_user_when_deleted_at`）；新加不带条件的 `workspace_user_properties_workspace_id_idx ON (workspace_id)`；`user_id` 不单独建索引 | 显示设置的写入经这个索引 `ON CONFLICT`（M3 设计 3.18）；物理级联要不带条件的索引（M3 设计 4）；按账户查的都带着工作区 |
| `workspace_user_properties` | 删除 `filters`、`display_filters`、`display_properties`、`rich_filters` | 工作项列表的筛选和显示列由它们的使用者 M4 按自己的格式加回（M3 设计 3.18） |
| `workspace_member_invites` | 13 列保留 11 列（M3/P3，`00009_workspace_workspace_member_invites.sql`） | M3 设计 4.4 |
| `workspace_member_invites` | `workspace_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `workspace_member_invites` | `email`：新加 `CHECK (email <> '' AND email = lower(email) AND email !~ '[[:space:]]')`，存规范化之后的邮箱 | 与 `users.email` 的规则相同（M3 设计 3.13）；规范化原来只在 Python 里做（`views/workspace/invite.py:93` 的 `strip().lower()`） |
| `workspace_member_invites` | `role`：`CHECK (role >= 0)` 收紧为 `CHECK (role IN (5, 15, 20))`，新加 `DEFAULT 5`；`accepted`：新加 `DEFAULT false`；新加 `workspace_member_invites_responded_check CHECK (responded_at IS NOT NULL OR NOT accepted)`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 只有三种角色（M3 设计 3.4）；接受的邀请一定有回应的时刻 |
| `workspace_member_invites` | 部分唯一索引改为 `workspace_member_invites_workspace_id_email_key ON (workspace_id, email) WHERE deleted_at IS NULL`（Plane 是 `workspace_member_invite_unique_email_workspace_when_deleted_at_ ON (email, workspace_id)`）；新加 `workspace_member_invites_email_idx ON (email) WHERE deleted_at IS NULL`；`workspace_id` 的索引改为 `workspace_member_invites_workspace_id_idx` | 已忽略的邀请没有删除，仍占着它的邮箱（M3 设计 3.8）；停用按邮箱删除发给它的邀请（M3/P6；注册按邀请的 id 读，不按邮箱）；物理级联要不带条件的索引（M3 设计 4） |
| `workspace_member_invites` | 删除 `token`、`message` | 不存令牌：链接里的令牌是由签名密钥派生的 MAC 从邀请的 id 算出的，数据库泄露时待接受的链接不泄露（M3 设计 3.8、8.1）；`message` 没有写入方 |
| `projects` | 36 列保留 22 列，另加 `last_issue_sequence`（见下），共 23 列（M3/P4a，`00010_project_projects.sql`） | M3 设计 4.6 |
| `projects` | `workspace_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `projects` | `project_lead_id`、`default_assignee_id`：模型的 `CASCADE` 改为 `ON DELETE SET NULL` | 物理删除一个账户不连带删除他负责的项目（M3 设计 3.15，删除关系图见 M3 设计 4.12） |
| `projects` | `name`：新加 `CHECK (name <> '' AND name !~ '[&+,:;$^}{*=?@#\|''<>.()%!-]')` | Plane 的禁用字符只在序列化器中检查（`serializers/project.py:39-44`，M3 设计 3.19） |
| `projects` | `identifier`：新加 `CHECK (identifier ~ '^[A-Z0-9ÇŞĞİÖÜ]{1,10}$')`，列类型仍是 `varchar(12)` | 见第四节"项目标识"（M3 设计 3.19） |
| `projects` | `network`：`CHECK (network >= 0)` 收紧为 `CHECK (network IN (0, 2))`，新加 `DEFAULT 2`；`description`：新加 `DEFAULT ''`；`cycle_view`、`module_view`、`issue_views_view`、`intake_view`、`guest_view_all_features`：新加 `DEFAULT false`；`archive_in`：新加 `DEFAULT 0`、`CHECK (archive_in BETWEEN 0 AND 12)`；`timezone`：新加 `DEFAULT 'UTC'`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值、choices 和验证器（0 私密、2 公开） |
| `projects` | `logo_props`：新加 `DEFAULT '{}'` 和 `projects_logo_props_check`：是对象，键只能是 `in_use`、`emoji`、`icon`，出现的每个键的值类型对，嵌套的 `emoji`、`icon` 两个对象也查到底；二十个反例都得到 `check_violation`，M3 设计 4.6 的十个在内，CHECK 的每个条件都有反例；五个合法值通过：4.6 的四个，和网页建项目时发的 `in_use` 为 `emoji` 的值（`TestProjectChecksRejectCounterexamples`） | 二·全局（对象型的 `jsonb` 列）；`{}` 表示没有图标（M3 设计 4.6） |
| `projects` | **新增** `last_issue_sequence integer NOT NULL DEFAULT 0 CHECK (last_issue_sequence >= 0)`：工作项编号的计数列，M4 取号；M3 不读写它，它不进入接口 | 替代 `issue_sequences`（一 B；v0-design 5.3） |
| `projects` | 两个部分唯一索引照搬，改名为 `projects_workspace_id_identifier_key`、`projects_workspace_id_name_key`（`ON (workspace_id, …) WHERE deleted_at IS NULL`；Plane 是 `project_unique_identifier_workspace_when_deleted_at_null`、`project_unique_name_workspace_when_deleted_at_null`，列的顺序相反）；新加不带条件的 `projects_workspace_id_idx ON (workspace_id)` | 按工作区列出项目用它们；物理级联要不带条件的索引（M3 设计 4） |
| `projects` | 删除 `emoji`、`icon_prop`、旧的 `cover_image`、`description_text`、`description_html`（旧的 json 列）、`page_view`、`is_time_tracking_enabled`、`is_issue_type_enabled`、`estimate_id`、`close_in` | 遗留列或对应功能已砍掉（归档保留，`archive_in` 和 `archived_at` 保留） |
| `projects` | 删除 `default_state_id`；`cover_image_asset_id` 暂不建，由 M5 随文件存储加入 | 新工作项的默认状态来自 `states."default"`（M1/P2 的交接）；在 M5 之前接口的 `cover_image_url` 是 `null` |
| `project_members` | 16 列保留 11 列（M3/P4a，`00011_project_project_members.sql`） | M3 设计 4.7 |
| `project_members` | `workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`member_id`：改为 `NOT NULL`（Plane 可为空），加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`）；没有成员的成员关系没有意义，Plane 也从不写入空值 |
| `project_members` | `role`：`CHECK (role >= 0)` 收紧为 `CHECK (role IN (5, 15, 20))`，新加 `DEFAULT 5`；`is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 只有三种角色（M3 设计 3.4）；模型的默认值 |
| `project_members` | 部分唯一索引照搬，改名为 `project_members_project_id_member_id_key ON (project_id, member_id) WHERE deleted_at IS NULL`（Plane 是 `project_member_unique_project_member_when_deleted_at_null`）；新加 `project_members_member_id_idx ON (member_id) WHERE deleted_at IS NULL`，不带条件的 `project_members_workspace_id_idx`、`project_members_project_id_idx` | 降为访客、结束成员关系按账户查；物理级联要不带条件的索引（M3 设计 4） |
| `project_members` | 删除 `view_props`、`default_props`、`preferences` | 和 `project_user_properties` 重复 |
| `project_members` | 删除 `sort_order`、`comment` | 侧边栏的顺序在 `project_user_properties.sort_order`，前端不读这一列；`comment` 没有写入方 |
| `project_user_properties` | M3/P4a由15列保留11列（`00012`）；M4/P5经 `00026_project_project_user_properties_issue_display.sql` 加回三列后为14列，仍不建 `rich_filters` | M3 设计 3.18/4.8；M4 设计 4.15 |
| `project_user_properties` | `workspace_id`、`project_id`、`user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `project_user_properties` | `preferences`：新加默认值 `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}` 和 CHECK（外层恰好一个键 `navigation`，它恰好有字符串 `default_tab` 和数组 `hide_in_more_menu`）；`sort_order`：新加 `DEFAULT 65535`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值，去掉 `pages`（文档页已砍）；二·全局（对象型的 `jsonb` 列） |
| `project_user_properties` | 部分唯一索引照搬，改名为 `project_user_properties_project_id_user_id_key ON (project_id, user_id) WHERE deleted_at IS NULL`（Plane 是 `project_user_property_unique_user_project_when_deleted_at_null ON (user_id, project_id)`）；新加不带条件的 `project_user_properties_workspace_id_idx`、`project_user_properties_project_id_idx` | 物理级联要不带条件的索引（M3 设计 4） |
| `project_user_properties` | `filters`：M4/P5加回 `jsonb NOT NULL DEFAULT '{}'`；十可选string[]键为state_id/state_group/priority/label_id/assignee_id/created_by_id/mention_id/subscriber_id/start_date/target_date，无archived/sub_issue/project_id；CHECK只校对象、键及string[]形状 | M4 §3.17/4.15；词值经IssueListSettings，非Plane旧JSON筛选树 |
| `project_user_properties` | `display_filters`：M4/P5加回 `jsonb NOT NULL`；默认 `{"layout":"list","group_by":null,"sub_group_by":null,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}`；CHECK要求七键及calendar两键的准确类型 | 去掉Plane的type/calendar_date_range，加calendar；布局/分组/排序词值由纯端口校验 |
| `project_user_properties` | `display_properties`：M4/P5加回 `jsonb NOT NULL`；assignee/start_date/due_date/labels/key/priority/state/sub_issue_count/link/created_on/updated_on十一boolean，默认全true，CHECK要求准确键和类型 | link仅显示偏好；所有三列CHECK用CASE保护非对象，JSON null为check_violation；不建rich_filters，工作区列范围不变 |
| `states` | 18 列保留 14 列（M3/P4a，`00013_project_states.sql`） | M3 设计 4.9 |
| `states` | `workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `states` | `name`：新加 `CHECK (name <> '')`；`"group"`：新加 `DEFAULT 'backlog'` 和 `CHECK ("group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled', 'triage'))`；`description`：新加 `DEFAULT ''`；`sequence`：新加 `DEFAULT 65535`；`"default"`：新加 `DEFAULT false`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 `StateGroup` |
| `states` | 部分唯一索引照搬，改名为 `states_project_id_name_key ON (project_id, name) WHERE deleted_at IS NULL`（Plane 是 `state_unique_name_project_when_deleted_at_null ON (name, project_id)`）；新加 `states_project_id_default_key ON (project_id) WHERE "default" AND deleted_at IS NULL`、`states_project_id_triage_key ON (project_id) WHERE "group" = 'triage' AND deleted_at IS NULL`，不带条件的 `states_workspace_id_idx`、`states_project_id_idx` | 见第四节"默认状态、分诊状态"（M3 设计 3.17）；物理级联要不带条件的索引（M3 设计 4） |
| `states` | 删除 `slug`、`is_triage` | `slug` 没有读取者；分诊状态只由 `"group" = 'triage'` 识别（M3 设计 3.17） |
| `labels` | 15 列保留 12 列（M3/P7b，`00014_project_labels.sql`） | M3 设计 4.10 |
| `labels` | `workspace_id`、`parent_id`：加上 `ON DELETE CASCADE`；`project_id`：改为 `NOT NULL`（Plane 可为空），加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`）；没有工作区级标签（M3 设计 3.16） |
| `labels` | `name`：新加 `CHECK (name <> '')`；`color`：新加 `DEFAULT ''`；`sort_order`：新加 `DEFAULT 65535`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局）；新加 `labels_not_own_parent_check CHECK (parent_id <> id)` | 模型的默认值；标签不能做自己的父标签（M3 设计 3.16） |
| `labels` | 两个部分唯一索引（`project_id` 为空时按 `name` 的 `unique_name_when_project_null_and_not_deleted`、不为空时按 `(project_id, name)` 的 `unique_project_name_when_not_deleted`）改为一个 `labels_project_id_name_key ON (project_id, lower(name)) WHERE deleted_at IS NULL`；新加不带条件的 `labels_parent_id_idx`、`labels_workspace_id_idx`、`labels_project_id_idx` | 名称在项目内唯一，不分大小写：Plane 的数据库区分大小写，序列化器不区分；没有工作区级标签（M3 设计 3.16）；删除父标签时按 `parent_id` 找子标签；物理级联要不带条件的索引（M3 设计 4） |
| `labels` | 删除 `description`、`external_source`、`external_id` | `description` 没有读取者（M3 设计 3.16）；v0 不做导入 |
| `issues` | 28 列保留 20 列（M4/P1a，`00015_issue_issues.sql`）；主键名 `issues_pkey`（Plane 是 `issue_pkey`） | M4 设计 4.2 |
| `issues` | `workspace_id`、`project_id`、`parent_id`：加上 `ON DELETE CASCADE`；`state_id`：改为 `NOT NULL`（Plane 可为空），加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`）；创建总有状态（项目恰有一个默认状态），单独删除状态因未删除工作项（含归档）在用而拒绝（第四节"删除状态"），该路径后续清理仅连带已软删工作项；父级项目/工作区异步删除后，严格过期的状态或父行清理可经CASCADE回收仍为活标记的issue，迟到交叠允许40P01原错重试（M4设计3.6、3.7、3.21） |
| `issues` | `name`：新加 `CHECK (name <> '')`；`description_html`：新加 `DEFAULT '<p></p>'`；`priority`：新加 `DEFAULT 'none'` 和 `CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none'))`；`sort_order`：新加 `DEFAULT 65535`；`sequence_id`：没有默认值（Plane 的模型默认 1），新加 `CHECK (sequence_id > 0)`；新加 `issues_not_own_parent_check CHECK (parent_id <> id)`、`issues_dates_check CHECK (start_date <= target_date)`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 choices；编号只取自项目的计数列 `last_issue_sequence`；工作项不能做自己的父工作项，开始日期不晚于目标日期（M4 设计 3.7、3.9）。描述和起止日期的列随表建出，进入接口在 M4 的 P4、P2 |
| `issues` | `archived_at` 由 `date` 改为 `timestamptz` | 和 `cycles`、`modules`、`projects` 的 `archived_at` 保持一致 |
| `issues` | 删除 `point`、`estimate_point_id`、`type_id`、`is_draft`、`description_binary`、`description_json` | 估算、工作项类型已砍掉；草稿是另一张表（`draft_issues`）；`description_binary` 是协同编辑的遗留；`description_json` 没有读取者（M4 设计 3.7） |
| `issues` | **新增**唯一约束 `issues_project_id_sequence_id_key UNIQUE (project_id, sequence_id)`（全表唯一，不是部分唯一）；外键列的索引只留不带条件的 `issues_workspace_id_idx`、`issues_project_id_idx`、`issues_parent_id_idx`、`issues_state_id_idx`；新加 `issues_project_id_state_id_idx ON (project_id, state_id) WHERE deleted_at IS NULL` | Plane 只靠咨询锁保证编号不重复；计数列从不回退、编号从不重用，所以全表唯一（M4 设计 3.7）；物理级联、子工作项、在用的状态按外键列找行；同一状态最大的 `sort_order`、按状态筛选（M4 设计 4.2） |
| `issues` | **已新增（M4/P8，00029）** `issues_name_trgm_idx USING gin(name gin_trgm_ops) WHERE deleted_at IS NULL`、`CREATE EXTENSION IF NOT EXISTS pg_trgm` | 标题字面包含候选及similarity排序；计划按选择性/locale决定，不保证三字符必走索引。Plane模型/迁移/搜索helper定向未见同类定义/调用；不据此断言其部署数据库从未安装扩展 |
| `issue_assignees` | 10 列照搬（M4/P1a，`00016_issue_issue_assignees.sql`）；主键名 `issue_assignees_pkey`（Plane 是 `issue_assignee_pkey`） | M4 设计 4.3 |
| `issue_assignees` | `workspace_id`、`project_id`、`issue_id`、`assignee_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局（模型的 `on_delete`）；v0 不物理删除账户，`assignee_id` 的 `CASCADE` 不会触发 |
| `issue_assignees` | 部分唯一索引照搬，改名为 `issue_assignees_issue_id_assignee_id_key ON (issue_id, assignee_id) WHERE deleted_at IS NULL`（Plane 是 `issue_assignee_unique_issue_assignee_when_deleted_at_null`），去掉 `(issue_id, assignee_id, deleted_at)` 的唯一约束（二·全局）；新加 `issue_assignees_assignee_id_idx ON (assignee_id) WHERE deleted_at IS NULL`，外键列的索引只留不带条件的 `issue_assignees_issue_id_idx`、`issue_assignees_workspace_id_idx`、`issue_assignees_project_id_idx` | 个人主页的"已分配"、按负责人筛选；物理级联要不带条件的索引（M4 设计 4.3） |
| `issue_labels` | 10 列照搬（M4/P1a，`00017_issue_issue_labels.sql`）；主键名 `issue_labels_pkey`（Plane 是 `issue_label_pkey`） | M4 设计 4.4 |
| `issue_labels` | `workspace_id`、`project_id`、`issue_id`、`label_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局（模型的 `on_delete`） |
| `issue_labels` | **新增**部分唯一索引 `issue_labels_issue_id_label_id_key ON (issue_id, label_id) WHERE deleted_at IS NULL`；外键列的索引只留不带条件的 `issue_labels_issue_id_idx`、`issue_labels_label_id_idx`、`issue_labels_workspace_id_idx`、`issue_labels_project_id_idx` | Plane 在数据库层没有这个约束，同一个标签可以挂两次（M4 设计 4.4）；物理级联要不带条件的索引，标签的也是 |
| `issue_relations` | 11 列照搬（M4/P3，`00020_issue_issue_relations.sql`） | M4 设计 4.6 |
| `issue_relations` | `workspace_id`、`project_id`、`issue_id`、`related_issue_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局）；`project_id` 是存储的 `issue_id` 那一端的项目 | 二·全局（模型的 `on_delete`）；Plane 写请求路径里的项目（M4 设计 3.10） |
| `issue_relations` | `relation_type`：不设默认值（Plane 的模型默认 `blocked_by`），新加 `CHECK (relation_type IN ('blocked_by', 'duplicate', 'relates_to'))`；新加 `issue_relations_not_self_check CHECK (issue_id <> related_issue_id)` | 只有页面的四种类型，阻塞反过来存为对方的 `blocked_by`；工作项不能关联自己（M4 设计 3.10、4.6） |
| `issue_relations` | 部分唯一索引 `issue_relation_unique_issue_related_issue_when_deleted_at_null ON (issue_id, related_issue_id)` 改为不分方向的 `issue_relations_pair_key ON (LEAST(issue_id, related_issue_id), GREATEST(issue_id, related_issue_id)) WHERE deleted_at IS NULL`，去掉 `(issue_id, related_issue_id, deleted_at)` 的唯一约束（二·全局）；外键列的索引只留不带条件的 `issue_relations_issue_id_idx`、`issue_relations_related_issue_id_idx`、`issue_relations_workspace_id_idx`、`issue_relations_project_id_idx` | 一对工作项之间至多一条关联，不论类型和方向（Plane 的键有方向，A 被 B 阻塞和 B 被 A 阻塞可以并存）；两个方向的读取和物理级联按两列找行（M4 设计 3.10、4.6） |
| `issue_subscribers` | 10 列照搬（M4/P1a，`00018_issue_issue_subscribers.sql`） | M4 设计 4.5 |
| `issue_subscribers` | `workspace_id`、`project_id`、`issue_id`、`subscriber_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局（模型的 `on_delete`）；v0 不物理删除账户，`subscriber_id` 的 `CASCADE` 不会触发 |
| `issue_subscribers` | 部分唯一索引照搬，改名为 `issue_subscribers_issue_id_subscriber_id_key ON (issue_id, subscriber_id) WHERE deleted_at IS NULL`（Plane 是 `issue_subscriber_unique_issue_subscriber_when_deleted_at_null`），去掉 `(issue_id, subscriber_id, deleted_at)` 的唯一约束（二·全局）；新加 `issue_subscribers_subscriber_id_idx ON (subscriber_id) WHERE deleted_at IS NULL`，外键列的索引只留不带条件的 `issue_subscribers_issue_id_idx`、`issue_subscribers_workspace_id_idx`、`issue_subscribers_project_id_idx` | "已关注"、按关注者筛选；物理级联要不带条件的索引（M4 设计 4.5） |
| `issue_activities` | 20 列保留 17 列（M4/P1a，`00019_issue_issue_activities.sql`）；主键名 `issue_activities_pkey`（Plane 是 `issue_activity_pkey`） | M4 设计 4.11 |
| `issue_activities` | `issue_id`：改为 `NOT NULL`（Plane 可为空），`DO_NOTHING` 改为 `ON DELETE CASCADE`；`workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`actor_id`、`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`project_id` 是这一行的工作项的项目 | 动态随工作项物理删除，不挡住清理（M4 设计 4.11、3.21）；`actor_id` 为空表示 nerve 自己（M4 设计决策点 4） |
| `issue_activities` | `verb`：新加 `DEFAULT 'created'` 和 `CHECK (verb IN ('created', 'updated', 'deleted'))`；`comment`：新加 `DEFAULT ''`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局）；`created_by_id`、`updated_by_id` 写执行者，删除时写删除者 | 模型的默认值；动态只有这三种动词（M7 的收集箱随它的迁移放宽）；Plane 在 Celery 里写动态，这两列实际总为空（M4 设计 4.11） |
| `issue_activities` | 删除 `issue_comment_id`、`epoch`、`attachments` | 评论不写动态，这个 `DO_NOTHING` 的外键本会挡住物理清理；`epoch` 只写不读；`attachments` 从不写（M4 设计 4.11、3.13） |
| `issue_activities` | 外键列的索引只留不带条件的 `issue_activities_workspace_id_idx`、`issue_activities_project_id_idx`；新加 `issue_activities_issue_id_created_at_idx ON (issue_id, created_at, id)`（不带条件） | 动态列表的游标、找"最新的一行"、物理级联都按工作项找行（M4 设计 4.11） |
| `issue_comments` | 20 列保留 13 列（M4/P4，`00021_issue_issue_comments.sql`）；`id`、`created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` 保留类型与可空性；主键/外键改标准名，`created_by_id`、`updated_by_id` 加 `ON DELETE SET NULL`；`created_at`、`updated_at` 加 `DEFAULT now()`，应用仍显式写锁下时刻 | 逐列以 Plane v1.4.2 快照与实际迁移核对；审计兜底不替代用例时钟 |
| `issue_comments` | `workspace_id`、`project_id`、`issue_id`、`actor_id`：保留 uuid，全部加 `ON DELETE CASCADE`；前三列保持非空，`actor_id` 由可空改为 `NOT NULL` | 用户引用按模型 CASCADE，其余归属连带与删除图一致（comment_reactions经评论二层连到工作项） |
| `issue_comments` | `comment_html` 保留非空 text，加 `DEFAULT '<p></p>'`；`comment_stripped` 保留非空 text，加 `DEFAULT ''`；`edited_at` 保留可空 timestamptz | 只保存清洗 HTML/文本，编辑后的审计由同一事务写 |
| `issue_comments` | 删除 `attachments`、`comment_json`、`access`、`external_id`、`external_source`、`description_id`、`parent_id` 及后两列的外键/description 唯一约束 | 附件有独立模型；无回复、无可见范围、无 JSON 镜像、无导入或 descriptions 镜像 |
| `issue_comments` | 保留并改名无条件 `issue_comments_{issue_id,workspace_id,project_id}_idx`；新增部分索引 `issue_comments_issue_id_created_at_idx ON (issue_id, created_at, id) WHERE deleted_at IS NULL`；不保留 actor/审计及已删 parent 列索引 | 物理级联与可见评论键集分页；不建其余无消费者索引 |
| `issue_reactions` | 11 列照搬（M4/P4，`00022_issue_issue_reactions.sql`）；`id`、`created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` 保留类型与可空性；主键/外键改标准名，`created_by_id`、`updated_by_id` 加 `ON DELETE SET NULL`；`created_at`、`updated_at` 加 `DEFAULT now()`，应用仍显式写锁下时刻 | 逐列以 Plane v1.4.2 快照与实际迁移核对；审计兜底不替代用例时钟 |
| `issue_reactions` | `workspace_id`、`project_id`、`issue_id`、`actor_id`：保留 uuid/非空，全部加 `ON DELETE CASCADE` | 用户引用按模型 CASCADE，其余归属连带与删除图一致（comment_reactions经评论二层连到工作项） |
| `issue_reactions` | `reaction` 保留非空 text；新增 `CHECK (reaction ~ '^[0-9]{1,7}(-[0-9]{1,7}){0,15}$' AND length(reaction) <= 128)` | 十进制码位格式，不是 Unicode Emoji 名录校验 |
| `issue_reactions` | 保留部分唯一并改名 `issue_reactions_issue_id_actor_id_reaction_key ON (issue_id, actor_id, reaction) WHERE deleted_at IS NULL`，去掉含 `deleted_at` 的全表唯一约束；保留并改名 `issue_reactions_{issue_id,workspace_id,project_id}_idx` 无条件外键索引，不保留 actor/审计索引 | 同一 actor 只能有一个活的同值反应；软删后可重加，物理连带可定位 |
| `comment_reactions` | 11 列照搬（M4/P4，`00023_issue_comment_reactions.sql`）；`id`、`created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` 保留类型与可空性；主键/外键改标准名，`created_by_id`、`updated_by_id` 加 `ON DELETE SET NULL`；`created_at`、`updated_at` 加 `DEFAULT now()`，应用仍显式写锁下时刻 | 逐列以 Plane v1.4.2 快照与实际迁移核对；审计兜底不替代用例时钟 |
| `comment_reactions` | `workspace_id`、`project_id`、`comment_id`、`actor_id`：保留 uuid/非空，全部加 `ON DELETE CASCADE` | 用户引用按模型 CASCADE，其余归属连带与删除图一致（comment_reactions经评论二层连到工作项） |
| `comment_reactions` | `reaction` 保留非空 text；新增 `CHECK (reaction ~ '^[0-9]{1,7}(-[0-9]{1,7}){0,15}$' AND length(reaction) <= 128)` | 十进制码位格式，不是 Unicode Emoji 名录校验 |
| `comment_reactions` | 保留部分唯一并改名 `comment_reactions_comment_id_actor_id_reaction_key ON (comment_id, actor_id, reaction) WHERE deleted_at IS NULL`，去掉含 `deleted_at` 的全表唯一约束；保留并改名 `comment_reactions_{comment_id,workspace_id,project_id}_idx` 无条件外键索引，不保留 actor/审计索引 | 同一 actor 只能有一个活的同值反应；软删后可重加，物理连带可定位 |
| `issue_mentions` | 10 列照搬（M4/P4，`00024_issue_issue_mentions.sql`）；`id`、`created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` 保留类型与可空性；主键/外键改标准名，`created_by_id`、`updated_by_id` 加 `ON DELETE SET NULL`；`created_at`、`updated_at` 加 `DEFAULT now()`，应用仍显式写锁下时刻 | 逐列以 Plane v1.4.2 快照与实际迁移核对；审计兜底不替代用例时钟 |
| `issue_mentions` | `workspace_id`、`project_id`、`issue_id`、`mention_id`：保留 uuid/非空，全部加 `ON DELETE CASCADE` | 用户引用按模型 CASCADE，其余归属连带与删除图一致（comment_reactions经评论二层连到工作项） |
| `issue_mentions` | 保留部分唯一并改名 `issue_mentions_issue_id_mention_id_key ON (issue_id, mention_id) WHERE deleted_at IS NULL`，去掉含 `deleted_at` 的全表唯一约束；`issue_mentions_mention_id_idx` 改为仅活行；保留并改名无条件 `issue_mentions_{issue_id,workspace_id,project_id}_idx`，不保留审计索引 | 从保存后的描述做有效成员差集；隐藏访客可被提及但不自动关注 |
| `issue_description_versions` | 15 列保留 13 列（M4/P4，`00025_issue_issue_description_versions.sql`）；`id`、`created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` 保留类型与可空性；主键/外键改标准名，`created_by_id`、`updated_by_id` 加 `ON DELETE SET NULL`；`created_at`、`updated_at` 加 `DEFAULT now()`，应用仍显式写锁下时刻 | 逐列以 Plane v1.4.2 快照与实际迁移核对；审计兜底不替代用例时钟 |
| `issue_description_versions` | `workspace_id`、`project_id`、`issue_id`、`owned_by_id`：保留 uuid/非空，全部加 `ON DELETE CASCADE` | 用户引用按模型 CASCADE，其余归属连带与删除图一致（comment_reactions经评论二层连到工作项） |
| `issue_description_versions` | `description_html` 保留非空 text，加 `DEFAULT '<p></p>'`；`description_stripped` 保留可空 text；`last_saved_at` 保留非空 timestamptz（无默认值）；删除 `description_binary`、`description_json` | 没有二进制协作/JSON 镜像；最后保存时间由锁下用例提供 |
| `issue_description_versions` | 新增 `issue_description_versions_issue_id_last_saved_at_idx ON (issue_id, last_saved_at, id)`（无条件，替代单列 issue 索引）；保留并改名无条件 `issue_description_versions_{workspace_id,project_id}_idx`，不保留 owned_by/审计索引 | 最新行及稳定降序分页、物理连带；P4请求不截断；P9周期按last_saved_at/id保留活版本最新20，归档仍参与、三层墓碑不碰 |
| `issue_links`（M4/P7） | 12列全部保留，无新增/删除列；实际迁移`00027_issue_issue_links.sql`，以下逐列核对Plane v1.4.2快照 | 表结构/真实长URL与碰撞证据见[P7 review](M4-issue-core/reviews/P7-issue-links-review.md) |
| `issue_links.id` | 保留uuid非空PK、标准名`issue_links_pkey` | 主键名与快照相同 |
| `issue_links.workspace_id` | 保留非空uuid；FK CASCADE、无条件外键索引改标准名 | FK默认立即检查，Plane为DEFERRABLE INITIALLY DEFERRED |
| `issue_links.project_id` | 保留非空uuid；FK CASCADE、无条件外键索引改标准名 | 同上 |
| `issue_links.issue_id` | 保留非空uuid；FK CASCADE、无条件`issue_links_issue_id_idx` | 同上，另保留原文排重索引 |
| `issue_links.title` | 保留nullable varchar(255) | 用户标题省略/null/空串分明，不由抓取覆盖 |
| `issue_links.url` | 保留非空text；新增非空且length≤2048字符CHECK | 不缩成字节限额或短varchar |
| `issue_links.metadata` | 保留非空jsonb；新增默认{}、对象/五键/string或null CHECK | DB通用map不等同crawler三种shape；公开error另限blocked/failed/null |
| `issue_links.created_by_id` | 保留nullable uuid，FK SET NULL，不保留单列审计索引 | 外键立即检查 |
| `issue_links.updated_by_id` | 保留nullable uuid，FK SET NULL，不保留单列审计索引 | 同上 |
| `issue_links.created_at` | 保留非空timestamptz，新增now()默认 | 请求仍显式写锁后同一时刻 |
| `issue_links.updated_at` | 保留非空timestamptz，新增now()默认 | 抓取不推进审计 |
| `issue_links.deleted_at` | 保留nullable timestamptz | 活行排重/任务写回均检查NULL |
| `issue_links`约束/授权 | 新增内置hash exclusion `((issue_id::text \|\| ':' \|\| url) WITH =) WHERE deleted_at IS NULL`，约束名`issue_links_issue_id_url_key`；只有该名+23P01映射409 | 原文相等recheck允许真实hash碰撞共存；不加extension/散列列；sqlc与runtime-grants纳入此表的四DML |
| `cycle_issues` | **新增**部分唯一约束 `(issue_id)`：一个工作项最多属于一个迭代 | Plane 只在代码里检查 |
| `modules` | 删除旧的 `description_text`、`description_html` | 遗留列 |
| `*_user_properties`、`issue_views` | 删除旧的 `filters` 列；筛选条件按 Nerve 自己的格式存储（列名与格式在 M3/M4 确定）；`issue_views` 删除 `query` 列 | 筛选格式改变（见接口差异） |
| `file_assets` | 删除 `page_id`；`draft_issue_id` 改为引用新的 `draft_issues`；`entity_type` 去掉 `PAGE_DESCRIPTION` | 文档页已砍掉；草稿重新设计 |
| `draft_issues` | **已重新设计（M4/P8，00028）**：Plane24列重构为九列，八个身份/审计列加optional创建字段的payload；不保留模块/迭代/估算/type | 本期逐列/FK/三索引及实际源码对照见文末P8附录，v0-design §5.4 |

---

## 三、接口差异

Nerve 不兼容 Plane 的 `/api/`、`/auth/`、`/api/v1/`、`/api/public/`、`/api/instances/`，而是重新设计了一套 `/api/v0`（接口版本号与产品主版本号一致），规范见 [v0-design 第 3 节](v0-design.md#3-接口设计规范)。和 Plane 相比，主要区别如下：

| 方面 | Plane | Nerve |
|---|---|---|
| 版本 | 页面用的内部接口 `/api/` 不带版本；公开接口是 `/api/v1/` | 只有一套接口 `/api/v0`，版本号跟随产品主版本号 |
| 路径 | `/api/workspaces/{slug}/projects/{pid}/issues/{id}/` 这样的多层嵌套，结尾必须带 `/` | 列表挂在父资源下，单个资源用短路径；结尾不带 `/` |
| 认证 | 页面用会话 Cookie 加表单登录，程序用另一套 `/api/v1` 加 `X-Api-Key` | 所有调用方都使用同一套接口和 `Authorization: Bearer` |
| PATCH 的响应 | 204，没有响应体 | 返回完整资源 |
| 筛选 | JSON 筛选树和 25 种旧查询参数并存 | 普通CSV参数，字段间AND、项内OR；none/me按字段能力；日期单日/闭区间/none，可重复且AND；非法词/未知或重复非日期参数422（M4/P5） |
| 分页 | `每页条数:页码:是否上一页` 形式的偏移游标 | canonical不透明键集游标；工作项载荷[order,key,id]，默认-created_at；P5当时十九码，P8保留原19并新增±link_count为当前21。两向null末尾、同键id同方向、名字仅折ASCII后按码点序，非Plane名字键兼容层 |
| 工作项列表路径/名称 | 多层workspace/project及旧results/名字查询键 | `listIssues`：GET `/api/v0/projects/{project_id}/issues`；`listMemberIssues`：GET `/api/v0/workspaces/{slug}/members/{user_id}/issues?relation=assigned\|created\|subscribed`。筛选/分组用state_id/state_group/priority/label_id/assignee_id/created_by_id/target_date，个人另project_id；排序用state_name/label_name/assignee_name/sub_issues_count等，不用state__name/labels__name/assignees__first_name |
| 工作项列表响应 | results及按页裁剪的关系数组，部分serializer已含link_count | `IssueListPage`/`Issue`：平面/组格续页{total,data,next_cursor}，分组{total,groups:[{key,total,data,next_cursor}]}，泳道组内subgroups及顶层subgroup_totals。缺分支省略、空[]、末页null，Issue不含description_html；P8全部Issue/Detail required非负int64 link_count，0显式返回，仅数活links、不看作者/metadata，支持±排序与整数游标。组格续页保留筛选另加group_key/sub_group_key，所有读取同独立RO/RR快照 |
| 项目的工作项显示设置路径 | `/projects/{id}/user-properties/` | 沿GET/PATCH `/api/v0/me/projects/{project_id}/preferences`，三字段闭合typed结构；结构400、词值422，逐顶层字段整体替换，省略保留、{}清filters；前端旧store/service迁移仍属P10b |
| 集合型的列表（工作区、成员、项目等） | 一次返回全部，响应是裸数组 | 同样一次返回全部、不分页，响应是 `{"data": [...]}` 封套，每个列表写明顺序（M3 设计 3.12） |
| 关联字段 | 名字不带 `_id`（`workspace`、`member`、`project_lead`），有的内嵌对象 | 只给 id，名字带 `_id`（`WorkspaceMember.workspace_id`）；例外有两个：工作区成员内嵌成员的公开资料 `member`（`MemberUser`，v0-design 3.6；M3 设计 5.2），关联内嵌对方工作项的列表行 `issue`（`IssueRelation.issue`，M4/P3；M4 设计 3.10） |
| 工作区的显示设置的路径 | `/api/workspaces/{slug}/user-properties/`（与工作项的筛选合在一起） | `GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`，只有项目导航的两项（M3 设计 3.18）；没有 `/sidebar-preferences/` |
| 迭代和模块归属 | 通过单独的接口设置 | 作为工作项字段，用 PATCH 修改 |
| 错误 | `{"error": "..."}`、DRF 字段错误、认证错误码三种格式混用 | 统一使用 RFC 9457 |
| 无权限 | 403 | 看不到的资源返回 404，看得到但没权限返回 403 |
| 错误码 | 接口描述中没有 | 每个操作在接口描述中用 `x-problem-codes` 声明它可能返回的错误码；字段错误带 `code`（M2 设计 3.11） |
| 请求体 | DRF 的序列化器忽略未知字段；重复的键以后一个为准（Python 的 `json`） | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段，也拒绝能有两种读法的请求体（同一个对象里重复的成员名，字符串不是合法的 Unicode）：400 `bad_request`，一次列出全部问题（能有两种读法的请求体只列出这两种问题；每个问题只报一次，最多 16 个，路径最长 256 字节；M2 设计 3.11） |
| 个人访问令牌的路径 | `/api/users/api-tokens/`、`/api/users/api-tokens/{id}/`（查看、修改、撤销） | 列出和创建是 `/api/v0/me/api-tokens`，撤销是 `DELETE /api/v0/api-tokens/{token_id}`：单个资源用短路径（v0-design 3.2）；不能查看单个令牌，也不能修改 |

---

## 四、行为差异

**M4/P14 创建与编辑表单**：沿现有表单和原生命令，描述自动补块 ID 只更新提交基线，不单独标记用户修改；有真实编辑时保留修改，普通工作项纯迁移显式保存才带 boolean skip_activity。字段反馈与拒绝后焦点统一走 FieldError/useRefusedFocus，创建请求中的空状态省略，交给服务端默认值。创建命令补设计要求的 c 快捷键，仍支持既有 ni；确认提示链接展示服务端编号。T5 保留 Plane 的 dirty→保存草稿确认流程，但将按钮、外点和 Escape 交给同一就绪/写锁/dirty 决策，删除描述编辑器的 document Escape，避免一键关闭两层；提示改为 ModalTitle 和双语文案。真实嵌套 Dialog 和 C4 页面验证焦点、背景隔离、退出动画、日期/标签 portal。无新增协议或全局 owner；三浏览器仍待 T16。

**M4/P14 父项（T6）**：Plane parent-issues-list-modal 的旧搜索/简要字段/正 tabIndex/悬停链接改为原生 Issue、已有 useIssueSearch 的权限证明与目录显示；新建无 id 不发 for，已有项由服务端 for=parent 排除所有后代。创建、详情及收集箱共用调用一致迁移，Promise 确认后才关闭；展示编号不再是 disabled button。I8/C4 页面核两已有/一新子项、进度、移动/解除、两层后代排除和内层 Escape，计数/动态均以真实服务端确认。M7 自身业务仍归 M7。

**M4/P14 关联/查重（T7）**：保留 Plane 的关系枚举、单选查重和 M7 ID 通知；查重用原生搜索/权限证明/目录，错误可只读重试，输入与弹窗有名称。最后的旧 ProjectService、barrel、专用搜索类型和例外已删除。关联仍用原生原子批量命令，I9 页面验证同/跨项目、反向显示、仅能写对端项目成员删边及两端审计；部分确认后的重试不再携带已确认项，已有实现经变异保护，无新增协议。

整审局部收口：A-01删除跨项目子项无消费者的cycle/module目录预取，每外部项目少两旧GET，state/member/label与原拒绝边界保持；A-02删除失去最后生产读者的旧列表乐观分布更新helper/barrel/两wrapper/六死测试及两独占常量块。真实领域analytics/progress取数及calculateCycleProgress七测保留；两项独立复审Approved，非将领域数据层提前接通。

**M4/P10a quick-add等待展示（R8，T4已实施并独评Approved）**：创建请求未返回前，输入在quick-add入口显示等待预览并禁用，保留Promise toast；不再将缺服务端编号/审计事实的输入作为普通卡片加入正式排序、分组或计数。成功后真实响应出现于列表，失败保留输入重试。首请求pending、双提交、跨workspace/project与旧结算边界及准确提交身份见[P10a T4/R8证据](M4-issue-core/reviews/P10a-web-issue-types-review.md#t4-r8-evidence)；未提前实现P10b协调引擎。

P10a移除迭代、模块、项目/工作区视图的工作项列表及专属动作，领域自身页头/详情/归档目录取数保留。M6/M7前既缺列表，也可能因自身旧接口未接通而拿不到名称/详情/统计/目录；实际观察与恢复责任见[P10a移交](M4-issue-core/reviews/P10a-web-issue-types-review.md#6-移交事项)。详情reaction/link不再从core嵌套展开读取，改由各自原资源GET入cache，非null父另一次raw retrieve；目标链3 GET/非null父4 GET，其他历史side-load另计，新增资源由所属fetch等待并传拒绝，真实Base提供有限重试。R10写请求wire/次数不变只适用于create/update，不表示所有读请求数量/时序不变。列表关系extractor已删，详情关系资源保留；其余P11数据层责任未完成。

实施与终审完成、集成待办；最终文稿提交后准确HEAD CI以交付回执核验；最后产品5f7085bce4aa5103a0133fa982847d274efa6e47，准确产品CI已核成功，最后交付HEAD CI待核。

M4/P6 已实现的差异以 [P6 review](M4-issue-core/reviews/P6-issue-events-review.md) 的事务、时钟及自动归档证据为准；P7已实现的链接/出站/请求任务见[P7 review](M4-issue-core/reviews/P7-issue-links-review.md)；P9后台删除/保留/清理的当前证据见 [P9 review](M4-issue-core/reviews/P9-purge-review.md)，页面及后续阶段仍开放。

| 行为 | Plane | Nerve |
|---|---|---|
| 自动归档与系统执行者（M4/P6） | Celery 按项目设置异步处理，活动显示旧机器人名 | River `issue.auto_archive` 在 periodic 队列；每项目一个事务，锁下按完成/取消及 `archive_in × 30` 天的 `updated_at` 阈值归档；动态行的 `actor_id` 为 SQL NULL，未持久化的 `issue.archived` 事件的 `ActorID` 为 nil；同戳发布，失败回滚当项目 |
| 领域事件与任务投递（M4/P6） | Celery信号/任务按原项目逻辑分发 | 12种`issue.*`事件在业务写末尾同事务同步分发；只投递客户端将任务写入同一PG事务，提交留/回滚无；P6生产订阅为空，首个链接订阅者在P7 |
| 链接与请求任务（M4/P7） | 代码检查重复，允许ftp/ftps；改URL保留旧预览至抓取结束 | 四API；HTTP(S)/2048字符/原文相等活行409；改URL原子清metadata，title-only保留当前metadata且不重抓；15事件目录，首个生产subscriber用原事务投crawl，default/3attempt；P6当时12/空生产订阅仍保留为历史 |
| 链接预览抓取（M4/P7） | 页面/图标无对应大小上限，默认内嵌SVG | 页面1MiB/图标64KiB、共享5秒、每资源最多5跳；全候选/固定IP防SSRF，六类图标MIME拒SVG，BOM/header/meta解码；网络失败blocked/failed后完成，日志脱敏；只id+URL+未删除守卫写metadata，不改审计/活动/事件 |
| 列表的分组和排序（M4/P5） | 未知排序静默回退；组总数错误、关系数组只当前页、偏移分页 | 非法词422；各层total分别去重、数组完整、原筛选AND组格键续页；19排序与7/5/42字段矩阵、八快照和I3/I4均有真实证据 |
| 已归档项目的工作项列表（M4/P5） | issue_objects返回空 | 显式项目列表照常读，archived=false/true分别经issue.read/issue.archive；项目写仍409。个人跨项目目录排除归档项目 |
| 个人主页工作项（M4/P5） | 访客可读且未完整施加访客规则 | member_issue.list仅工作区管理员/成员；actor可见性/OwnOnly与路径目标三关系分离，inactive目标仍可存在；I19/I20及矩阵已验证 |
| 项目工作项显示设置（M4/P5） | filters/rich_filters/display_filters/display_properties四列旧格式 | 项目仅三列新格式，不建rich_filters；GET无写、按人隔离、恢复保留、不同字段并发不丢值，I5及四方向race验证；工作区范围不变 |
| 写操作动态 | Celery 异步写入 | 与业务写入在同一事务中同步写入 |
| 软删除的连带删除 | Celery 异步执行 | 删除工作项、状态、标签仍原事务同步；删除项目/工作区时M3行同步、同事务投issue分批任务，issue资源提交后即不可见、后台写原请求At/By；默认60天后的父CASCADE与迟到任务交叠允许40P01重试 |
| 最近访问 | 在 GET 请求中顺带写入 | 由前端显式调用接口记录，GET 请求没有副作用 |
| 通知偏好 | 决定是否发送邮件 | v0 没有邮件，不设通知偏好；站内通知的生成规则与 Plane 一致 |
| Webhook 负载 | Plane 格式 | Nerve 格式，使用 HMAC-SHA256 签名，禁止指向内网地址 |
| 接口调用日志 | 只记录 `/api/v1` 中用 API Key 发起的请求，请求头原样存储 | 记录所有写请求，不区分令牌类型；存储前去掉敏感请求头 |
| 工作项编号 | 咨询锁加 `issue_sequences` 表；序列化器允许客户端写 `sequence_id` | 在项目行上用计数列原子自增；客户端不能写它；`(project_id, sequence_id)` 全表唯一，编号从不重用（M4 设计 3.7） |
| 工作项的写的取锁 | 动态在 Celery 里异步写，写之间不串行 | 不加锁地预检，判定拒绝的在那里答（404、403），不取行锁；放行的写以 `FOR NO KEY UPDATE` 锁住项目行，在锁下重读、再判定，动态、关注与工作项在同一个事务里写（M4 设计 3.5，决策点 1） |
| 一个项目上的并发写 | 不限 | 每个进程里一个项目上的工作项的写至多 K 个同时在数据库里（`issue.write_gate.per_project`，默认 2），多的在进程里排队、不持连接；排满、等过 `issue.write_gate.max_wait`、等锁超过 `issue.write_lock_timeout` 时 503 `server_busy`，带 `Retry-After: 1`（M4 设计 3.5，架构问题 11） |
| 修改工作项的响应 | 204 | 200，回答修改之后的 `IssueDetail`；什么都没改的修改不写任何行，回答锁下读到的工作项（总体设计 3.1；M4/P1b spec 第 3 节第 4 条） |
| 工作项的负责人的读取 | 只返回"在某个项目里仍有效"的负责人 | 返回全部未删除的负责人行，离开了项目的负责人照样列出；修改只检查新加的负责人，留下的不查（M4 设计 3.6；M4/P1b spec 第 3 节第 3 条） |
| 工作项的负责人、标签不合格 | 静默去掉 | 负责人不是项目中不是访客的有效成员、标签不是这个项目未删除的标签：422 `not_allowed`（`assignee_ids[i]`、`label_ids[i]`）；同一个 id 给两次：422 `duplicate`（M4 设计 3.7） |
| 默认负责人 | 指派但不关注 | 指派并关注；建工作项时现查他仍是项目中不是访客的有效成员，不是就不指派、不报错（M4 设计 3.7、3.12） |
| 负责人的动态 | 按请求里的 id 写，含被去掉的 | 只为实际指派的写（M4 设计 3.13） |
| 自动关注 | 每个产生通知的动态的执行者；被提及的人都关注 | 创建者、负责人、评论者，以及被提及的有效成员中能读这个工作项的人；按被提及者自身项目角色及 `OwnOnly` 检查，描述修改者不因编辑自动关注（M4/P4，设计 3.12） |
| 删除状态 | 有工作项时 400 | 有工作项在它里面（已归档的也算）时 409 `project.state_in_use`，这一检查在默认状态的检查之前（M4 设计 3.6） |
| 工作项的起止日期 | 两个都在请求里才检查开始不晚于截止 | 按合并之后的值检查：只给一个日期时与存着的另一个比较，开始晚于截止 422 `not_allowed`（在 `target_date`，只给了 `start_date` 时在 `start_date`）；同一天允许；数据库另有 `issues_dates_check`（M4 设计 3.7） |
| 工作项的创建者的权限 | 只看工作区成员关系，已离开项目也能读、改、删 | 创建者只在删除上另有权限（`issue.delete` 的 `AllowCreator`）：他须仍是项目的有效成员、角色不低于成员；离开项目或成了访客之后不能再凭它删（没有别的权限的，看不到项目时 404，看得到时 403）（M4 设计 3.4） |
| 访客只看自己的工作项 | 只在约 6 处执行 | 项目没开 `guest_view_all_features` 时，它的访客（按项目角色算，工作区管理员在项目里是访客时同样）只看得到自己创建的工作项：取、按编号取、修改、删除、归档、恢复别人的都答 404 `issue.not_found`；子工作项的列表和 `sub_issues_count`、关联的列表只含他看得到的工作项，回答里的 `parent_id` 指名他看不到的工作项时为 `null`（M4/P3），修改的 `parent_id` 同他得到的回答比，他看不到的父工作项算没有：送回这个 `null` 什么都不改，指名它像指名不存在的工作项一样 422（M4/P3，负责人裁定 A，2026-10-24）；读在判定之后执行这条规则，写在不加锁的预检和锁下的判定中各执行一次（M4 设计 3.4；其余的列表、子资源在各自的 Phase） |
| 访客删除自己建的工作项 | 服务端允许，页面不显示 | 403 `forbidden`（工作区管理员是项目的有效成员时不论项目角色都通过项目的每条规则，总体设计 6.5 的约定；所以只是项目访客的工作区管理员删得了自己建的，204，不是凭 `AllowCreator`）（M4 设计 3.4） |
| 已归档的工作项 | 只挡修改；评论、链接、表情、关联不挡；能成为父工作项 | 修改 409 `issue.archived`，什么都没改的修改也是；恢复、删除照常；在已归档的项目里先答 409 `project.archived`；往它下面加子工作项、从它加关联、删除它的关联（任一端已归档）409 `issue.archived`；不能被改成别的工作项的父工作项、被加到别的工作项下面、成为关联的对方（422 `not_allowed`）；已归档的子工作项、对方不在子工作项和关联的列表、`sub_issues_count` 里，恢复之后重新出现（M4/P3）（M4 设计 3.8、3.9、3.10；评论/表情P4及链接P7的写已接归档只读，页面仍按各自Phase） |
| 工作项的归档、恢复 | 归档答 `{archived_at}`，恢复 204；再归档、恢复没归档的 404；状态不在"已完成""已取消"组时 400 | 都答 `IssueDetail`；已归档的再归档、未归档的再恢复不写、200；状态不在"已完成""已取消"组时 409 `issue.not_archivable`；各写一条动态（`archived_at` 的 `manual_archive`、`restore`），动态的 `comment` 不同（M4 设计 3.8、3.13） |
| 按编号取工作项 | 格式不对 400，不是成员 403 | `GET /api/v0/workspaces/{slug}/issues/{identifier}`：不是工作区的有效成员 404 `workspace.not_found`；之后格式不对、项目或编号不存在、看不到、访客规则都是同一个 404 `issue.not_found`；看得到项目而不是它的有效成员的（工作区管理员；项目公开时，工作区成员）403 `forbidden`，与取工作项相同；标识经同一个折叠比较（`identifier-fold.json`，`i`、`ı`、`I`、`İ` 一样）（M4 设计 3.7） |
| 删除工作项 | 动态保留，另写一条"已删除"的动态；确认框不提子工作项 | 工作项及各层子工作项连同负责人、标签、双方向关联、链接、关注、提及、动态、评论/两层表情和描述版本按原事务审计软删；已软删行不覆戳，关系先锁后判另一端项目删除归属。不写"已删除"动态；物理清理时活动/评论子链采用CASCADE，不沿用DO_NOTHING阻挡说法。完整issue侧13表目录见P9 spec；子树确认框仍P14 |
| 父工作项 | 不检查自己、不检查成环 | 父工作项要是同一个项目里调用者看得到、未归档的工作项，不是它自己、不是它下面的：创建、修改的 `parent_id` 不是这样的工作项（不存在、已删除的也是）422 `parent_id` `not_allowed`；加子工作项的父工作项是路径里的工作项，调用者看不到它（不存在、已删除的也是）404 `issue.not_found`，它的项目已归档 409 `project.archived`，它已归档 409 `issue.archived`，放进去的每一个不是同一个项目里调用者看得到、未归档的工作项（不存在、已删除的也是），或是父工作项自己、它上面的，各在它的位置 422 `sub_issue_ids[i]` `not_allowed`，一次答全部；同一个项目里改父工作项的写在项目的 N 之下一个接一个，两个并发的移动不会合起来成环（M4 设计 3.9） |
| 加子工作项的动态 | `old_value` 是子工作项自己的编号；`comment` 是 "updated the parent issue to" | 是它原来的父工作项的编号（没有时为空），`old_identifier` 是原来的父工作项（M4 设计 3.13 第 5 行）；`comment` 是 "updated the parent work item to"，修改父工作项的一行同样（Nerve 说 work item，同 M4/P2 的归档、恢复两行） |
| 子工作项数 | 数 `issue_objects` 里的直接子工作项（不含已删除、已归档、草稿、分诊状态的，也不含已归档的项目里的），不论调用者看不看得到 | `sub_issues_count` 数未删除、未归档的直接子工作项中调用者看得到的：项目里"只看自己"的访客只数自己建的（M4 设计 3.9；按它排序在 M4 的 P5） |
| 关联 | 六种类型，不检查类型和自己；一对工作项可有两条；显示看不到的项目里的工作项；删除不存在的 500 | 页面的四种（`relates_to`、`duplicate`、`blocked_by`、`blocking`，阻塞反过来存为对方的 `blocked_by`），别的类型 422 `relation_type` `invalid_format`；一对工作项至多一条，不论类型和方向（422 `issue_ids[i]` `duplicate`）；对方要是同一个工作区里调用者看得到、未归档、所在项目未归档的别的工作项（422 `not_allowed`）；列表只含调用者看得到、未归档的对方，不含在别的已归档的项目里的：已归档的项目里的工作项照样列出它与同一个项目里的工作项的关联；按关联的 `id` 删除，能改两端任一个的即可，不存在、已删除、两端都看不到的 404 `issue.relation_not_found`（M4 设计 3.10） |
| 关联的动态 | 对方 id 在 `old_identifier`；反向行的 `project_id` 是本项目；删除的反向行的 `comment` 用正向的类型（`field` 只把 `blocked_by`、`blocking` 反过来），`old_identifier` 是它自己 | 增、删各两条：正向的写在本工作项上、反向的写在对方上，各用从自己看的类型，各在自己的项目里；加的一条对方 id 在 `new_identifier`，删的一条在 `old_identifier`（M4 设计 3.13 第 20–23 行） |
| 草稿发布 | `create_draft_to_issue`限定owner，使用request.data给IssueCreateSerializer；从草稿拿项目上下文，另排活动/迁移附件再delete | 先PATCH，再零字节body发布锁下存储payload；要求draft.write和issue.create，共享唯一创建核心、同事务/actor/At软删草稿，结构400/业务422同路径，项目改变409；附件移交仍留M5 |
| 归档 | 工作项、迭代、模块、项目的归档与恢复，以及项目级自动归档 | 规则一致（见 v0-design 5.5）；归档和恢复改为 `POST .../archive` 和 `POST .../unarchive` 两个动作接口；列表通过 `?archived=true` 查询已归档的对象 |
| 描述、评论的长度 | 没有同一边界规则 | `description_html`、`comment_html` 各最多 500000 个 Unicode 字符，默认请求体仍为 1 MiB（M4/P4） |
| 描述与清洗 | 异步动态/版本与宽泛输入 | `issue/adapter/content` 唯一清洗策略，app 小接口、domain 纯值；创建默认 `<p></p>`，保存后的描述、提及、关注、动态、版本同事务；旧工作项不伪造初版（M4/P4，A1） |
| `skip_activity` | 沿旧字符串标志跳过动态 | 仅 JSON boolean；出现（含 false）只能与描述一起且不能带其他 patch 字段。true 且纯文本、按顺序保留重复的 user mention UUID、`a[href]`、checkbox checked 均相同时，只写清洗 HTML/stripped 并同步提及资格，不改主审计、不写动态/版本；false 或内容标识变化是普通保存。只改格式可不留审计是已批准代价，图片 src 投影留 M5（M4/P4） |
| 空评论 | 服务端接受 | 清洗后无文字且无 mention/image 节点，422 `comment_html/required`（M4/P4） |
| 改、删别人的评论 | 项目管理员可以改、删 | 仅作者编辑；作者或项目管理员删除，其他人 403 `issue.not_author`；隐藏先统一子资源 404，项目归档先于工作项归档 409，归档先于相同 HTML 无改动（M4/P4） |
| 表情的值与归属 | 任意文本；重复 400 | 十进制格式 `^[0-9]{1,7}(-[0-9]{1,7}){0,15}$` 且 ≤128 字符；自己的重复 409，删除只找自己的活行，无该行 404；评论表情不推进 issue/comment 审计（M4/P4） |
| 评论、表情的动态 | 写，从不显示 | 不写；通知事件留 P6，评论删除的痕迹仅审计，当前无事件发布（M4/P4） |
| 提及的同步 | 与请求体比较，不带描述的修改删掉全部 | 从实际保存后的清洗描述做集合同步；普通相同 HTML 不写，skip=true 相同内容仍同步资格；去掉提及不取消旧关注（M4/P4） |
| 描述的历史 | 访客能读；版本后台生成 | 仅管理员/成员可读，访客规则仍在先；同所有者且距最后保存 0–600 秒（含端点）滑动合并，其他情况新建；普通 PATCH 还原；仅描述历史，无 issue_versions；P9周期仅对未删项目/工作项/版本按last_saved_at DESC,id DESC保留20；归档仍处理、软删版本归purge（P4历史请求规则不变） |
| 动态的列表 | 不分页；不带类型时 500；每行带完整工作项、项目、工作区；泄漏隐藏引用 | `(created_at,id)` 升序游标分页，只给精简 DTO/actor_id、不返回 comment；引用 old/new 两端都按历史有效成员关系和 OwnOnly 过滤，先过滤再 LIMIT。历史目录看 is_active，不看 deleted_at/归档，物理清理引用后略去；目标权限、目录与查询处于同一只读快照（M4/P4） |
| 时间线的增量读取 | `created_at__gt`，只取新增 | 保留产生末页的请求 cursor 重取，再按 id 覆盖/重新排序；描述原地合并推进 created_at 会再出现。别端对旧评论的编辑/删除须整体重读，不能承诺仅末页自动更新（M4/P4） |
| 自动关闭 | 项目设置 `close_in` 后，长期未更新的未完成工作项会被自动关闭 | v0 不做 |
| 忘记密码 | 发邮件重置 | 服务器管理员用命令行重置 |
| 管理员重置密码 | 只改密码（会话随之失效），PAT 不动 | `nerve users reset-password` 结束全部会话、撤销全部 PAT，输出撤销的数量（M2 设计 3.5、3.17） |
| 重置密码命令的邮箱 | `reset_password` 按原样匹配 | 按注册时的规则规范化（去掉首尾空白、转小写） |
| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行，M2/P3a 在修改密码时执行，M2/P3b 在创建账户和重置密码两个命令中执行 |
| 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2 设计决策点 2） |
| 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（M2 设计 3.11） |
| 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`，带 `Retry-After: 1`（M2 设计 3.8） |
| 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401 `identity.invalid_credentials`，耗时也相同：对一个启动时生成的假哈希做一次同样参数的校验（M2 设计 3.9）；这只在存储的哈希都用当前参数时成立，调高 argon2 参数之后，休眠的账户再次登录之前能被耗时区分（M2 设计 §16） |
| 会话的期限 | Django 会话，从登录起固定 7 天（`SESSION_COOKIE_AGE`），请求不延长 | 访问令牌 15 分钟；会话从登录起 30 天，续期不延长；刷新令牌每次使用后换新，并检测重复使用；退出只结束当前这一处登录（M2 设计 3.5） |
| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 进程内的令牌桶，每个桶有速率和突发：匿名按 IP、已认证按凭证、登录按 IP 和"IP + 邮箱"、注册按 IP，认证之前另有按 IP 的失败闸门；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*`（M2 设计 3.10）。修改密码另有按账户的桶 `password_user` |
| 无效的个人访问令牌 | 403（`AuthenticationFailed` 没有 `authenticate_header`） | 401 `unauthorized` |
| 退出、修改密码、停用之后的旧凭证 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（M2 设计 3.5） |
| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；服务器管理员的 `nerve users deactivate` 与自助停用相同，`nerve users activate` 恢复账户，恢复后没有过期的 PAT 重新可用；停用在同一个事务里结束他在全部工作区和项目的成员关系（行保留），软删除发给他邮箱的全部邀请（待接受的和已忽略的，与 Plane 相同），以及他是唯一有效成员的工作区里的待接受邀请（Plane 不删：接受之后工作区有成员而没有管理员）；他是某个工作区或项目唯一的有效管理员、那里还有别的有效成员时拒绝（409 `workspace.sole_admin`、`project.sole_admin`，命令退出码 1），什么都不改；`nerve users activate` 只恢复账户，成员关系按工作区用 `nerve workspaces reactivate-member` 恢复（M2 设计决策点 3，M3 设计 3.9） |
| 修改登录邮箱 | 用户在个人设置中向新邮箱索取验证码后修改 | 只能由服务器管理员用 `nerve users set-email` 修改：新邮箱按注册时的规则规范化，结束该账户的全部会话，PAT 不撤销（M2 设计决策点 1、3.17） |
| 创建账户的命令 | 没有（第一个账户通过实例设置页创建） | `nerve users create`：建账户和资料，不建会话，注册关闭时也能用（M2 设计决策点 2） |
| 个人访问令牌的管理 | 只能用 Cookie 会话管理 | 任何凭证都能管理，包括 PAT 本身（v0-design 0.2 原则 2） |
| 个人访问令牌的 `last_used` | 每个请求都写 | 每分钟最多写一次 |
| 个人访问令牌的名称和过期时间 | 不校验：名称过长时变成 500，过期时间可以是过去 | 名称 1–255 个字符；过期时间必须在未来 |
| 个人访问令牌的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供；令牌原文只在创建时返回一次 |
| 工作区的 slug | 服务端接受大写；删除工作区时把 slug 改成 `slug__<时间戳>` 腾出它；`PATCH` 能改 | 只有小写；部分唯一索引，删除时不改名；建好之后不能改（M3 设计 3.10） |
| 保留的工作区名 | 前后端各一份，服务端 45 个以上，含 Plane 的产品词 | 服务端一份（`workspace/domain/reserved_slugs.txt`）：本站用到的顶层路径段（应用的顶层路由段、`public/` 的顶层目录、服务端的 `api`、`assets`、`healthz`、`readyz`）加 4 个预留段（`admin`、`docs`、`help`、`static`）；前端的副本随 M3 的前端改造删除，改问 `GET /api/v0/workspace-slugs/{slug}`（M3 设计 3.10） |
| 建工作区之后 | 投递 `workspace_seed`：建一个名为 "Plane" 的机器人账户做管理员，再建演示项目、状态、标签和工作项 | 什么都不投递，没有演示数据（M3 设计 3.11） |
| 关闭创建工作区时 | 实例管理员在管理后台为自己建工作区 | 服务器管理员用 `nerve workspaces create --slug --name --admin-email` 建，不受开关限制，`--admin-email` 的账户是它的管理员（M3 设计 3.11） |
| 工作区的显示设置 | `GET` 时 `get_or_create`：读取就建行 | `GET` 不写库，没有行时返回默认值（`ACCORDION`、10）；第一次修改时经部分唯一索引 `INSERT … ON CONFLICT` 建行（M3 设计 3.18） |
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`，不删账户，提交后slug立即可重用；工作区/邀请/成员/显示设置及本次锁住项目的M3行同步软删，同事务投每个本次项目一条issue任务及一条无项目草稿任务。issue窗口立即不可见、后台沿原请求At/By，更早删除项目和旧墓碑保留自己的审计（M4/P9） |
| 邀请的列出、创建、修改、删除 | 工作区管理员和成员；修改不限制角色，成员能把邀请改成管理员 | 只有工作区管理员（M3 设计决策点 4）；邀请的角色因此不高于邀请人（M3 设计 3.8） |
| 邀请令牌 | JWT，原文存库；公开的查看不要令牌，返回被邀请的邮箱；关闭注册时，有任何一份未删除的邀请的邮箱就能注册 | 由签名密钥派生的 MAC（`nrv_inv_` 加 22 个字符），不存库，管理员列出时重新算出；公开的查看要令牌、不返回邮箱；关闭注册时要有效的令牌、且注册邮箱与邀请的相同（M3 设计 3.8、决策点 1） |
| 邀请的链接 | `/workspace-invitations/?invitation_id=…&slug=…&token=…`；另有系统内的接受：`/invitations` 页和新手引导的"加入工作区"一步按账户的邮箱列出发给他的邀请，批量接受 | 只有链接一条路：`/workspace-invitations?invitation_id=…&token=…`；接受要登录，账户的邮箱须与邀请的相同（M3 设计 3.8、决策点 2）（页面在 M3/P9；`/invitations` 页和新手引导的一步在 M3/P8a 删除） |
| 重复的邀请 | 静默忽略 | 422 `duplicate`，整批不插入，`invitations[i]` 是它在请求中的下标；已忽略的邀请仍占着这个邮箱，删除之后才能再邀请（M3 设计 3.8） |
| 接受邀请之后 | 服务端写 `last_workspace_id` | 服务端不写；前端写（M3 设计 3.14；页面：P9） |
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色 | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的；取的是访客时，同一个事务里他在这个工作区的项目角色都改为访客，含已离开的项目（M3 设计 3.8） |
| 移出成员、离开工作区时的唯一管理员检查 | 移出：拿工作区成员的 `id` 去比项目成员的 `member_id`，永远不会命中，查的又是"只有一个成员"的项目；离开：查"只有他一个成员、而他是管理员的项目"，与它的提示语相反 | 他是这个工作区里某个项目唯一的有效管理员、而那个项目还有别的有效成员时，移出和离开都答 409 `project.sole_admin`，什么都不改；那里只有他一人时允许。离开另有一条，与 Plane 相同：工作区唯一的有效管理员不能离开，哪怕只有他一人，409 `workspace.sole_admin`（M3 设计 3.7） |
| 恢复被移出的成员 | 管理命令 `reactivate_workspace_member`（位置参数） | `nerve workspaces reactivate-member --slug <slug> --email <邮箱>`，行为照搬：已结束的成员关系恢复为有效，角色不变，停用的账户照样恢复；项目成员关系仍无效，他加入项目（3.5 允许时）或被有权添加项目成员的人（项目管理员，或同时是工作区管理员的项目成员，M3 设计 3.4）重新添加时恢复（M3 设计 3.5、3.11） |
| 移出成员、离开工作区时发给他的待接受邀请 | 不动：他凭旧链接就能回来 | 同一个事务里软删除这个工作区里发给他邮箱的待接受邀请（已忽略的不动），回来要新的邀请（M3 设计 3.8） |
| 项目负责人、默认负责人 | 外键 `CASCADE`；负责人可以是任何账户 | `ON DELETE SET NULL`；创建项目时，负责人须是工作区的有效管理员或成员，否则 422（`project_lead_id`，`not_allowed`），他与创建者都成为项目管理员（M3 设计 3.15、3.19）。修改项目时，负责人、默认负责人都须是项目中不是访客的有效成员，否则 422（`project_lead_id`、`default_assignee_id`，`not_allowed`），`null` 清空；修改不把谁加为成员（M3 设计 3.19） |
| 看得到而不是成员时取项目 | 409（公开项目）或 403（私密项目） | 200，`member_role`、`sort_order` 为 `null`（M3 设计 3.19） |
| 工作区访客取没加入的公开项目 | 409 | 404 `project.not_found`：看不到（M3 设计 3.19） |
| 已归档的项目 | 取单个 404；列表里与未归档的混在一起 | 取单个照常返回；列表默认不含，`?archived=true` 只列它们（M3 设计 3.19） |
| 项目标识 | 最多 12 个字符，只禁一组符号 | 转成大写后 1–10 个，只能是 `A-Z`、`0-9` 和 `ÇŞĞİÖÜ`（M3 设计 3.19）；修改项目时同一规则 |
| 修改、删除项目 | 不是项目成员的工作区管理员也能 | 项目级规则：项目管理员，或同时是工作区管理员的项目成员；不是成员的工作区管理员先加入（M3 设计 3.4） |
| 归档、恢复项目 | 项目管理员和成员 | 项目管理员，或同时是工作区管理员的项目成员（M3 设计 3.4） |
| 添加已是有效成员的人为项目成员 | 顺手改他的角色 | 422 `duplicate`（`members[i].member_id`），整批都不写（M3 设计 3.5） |
| 加入项目时恢复以前的成员行 | 只改 `is_active`，保留旧的角色：被移出的项目管理员自己加入就拿回管理员 | 角色取原来那一行的角色与他现在的工作区角色中较低的一个：不比新加入给得更多（被移出的项目管理员现在是工作区成员，回来是成员），也不比原来那一行更多（与 Plane 相同，被降为访客的人离开再加入仍是访客）（M3 设计 3.5） |
| 修改状态 | 项目的访客也能 | 新建、修改、删除状态和设为默认：项目管理员，或同时是工作区管理员的项目成员（M3 设计 3.4） |
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`；状态的操作同样按 `group` 认出分诊状态（列表不含它，按 id 修改、删除、设为默认答 404 `project.state_not_found`），设为默认在同一个事务里先清掉原来的默认，项目恰好一个默认状态（M3 设计 3.17） |
| 标签的名称 | 数据库区分大小写，序列化器不区分（"bug" 与 "Bug" 并发时都能写入） | 数据库不分大小写：`labels_project_id_name_key ON (project_id, lower(name))`，同名答 409 `project.label_name_taken`（M3 设计 3.16） |
| 标签的层级 | 服务端不检查：接口能建三层、把有子标签的标签放到别的标签下、以别的项目的标签做父标签 | 服务端执行页面的两层规则：父标签须是同一项目、未删除、没有父标签的标签，有子标签的标签不能有父标签，标签不能做自己的父标签，否则 422（`parent_id`，`not_allowed`）；每个标签的写都先锁项目行，并发下规则仍成立（M3 设计 3.16） |
| 工作区级标签 | 模型允许（`project_id` 可空） | 没有：`project_id` 非空（M3 设计 3.16） |
| 列出标签 | 任何工作区成员都得到 200，已离开的成员仍能列出 | 项目的有效成员，访客也能；已归档的项目照常列出（M3 设计 3.4、3.19） |
| 新建标签的顺序 | 请求可以给 `sort_order`：项目还没有标签时用它，否则忽略，一律是最大值加 10000；没给、没有标签时 65535 | 请求不给 `sort_order`（给了答 400，同"请求中的未知字段"）：一律是项目未删除的标签的最大值加 10000，没有标签时 65535（M3 设计 4.10）；位置用 `updateLabel` 改 |
| 一组中唯一的状态 | 服务端能删除、能改到别的组（页面不让） | 删除它、把它改到别的组：409 `project.state_last_in_group`（它同时是默认状态时，删除先答 409 `project.state_default`，例如新项目的 Backlog），每一组（分诊组除外）至少一个状态（M3 设计 3.17） |
| 时区 | 只接受 `pytz.common_timezones`；时区列表中负的非整点偏移多算一小时（例如马克萨斯群岛的 −09:30 写成 −10:30） | 接受 Go 的时区数据认得的任何 IANA 名称（`Local` 除外），程序内嵌时区数据；时区列表接口给的仍是同一份常用列表，偏移按请求时刻计算，写法正确（M2 设计 5.3） |

## 五、M4/P8实际源码对照（2026-10-27）

本期对照主目录只读 `plane/apps/api/plane/` 的实际源码（下文缩写 `P/`），同时核本清单既有Plane schema快照；未在plane/refer运行Git、测试或容器。Nerve结果对应产品`10f44790`，详见[P8 spec](M4-issue-core/specs/P8-drafts-search.md)与[review](M4-issue-core/reviews/P8-drafts-search-review.md)。源码行号是此次读取位置，不以ignored导航文件替代源码证据。

### 草稿九列、外键与索引

Plane `P/db/models/draft.py:16–82`及 `workspace.py:185–187` 为实体字段与nullable项目；本仓快照 `tools/plane-schema/plane-v1.4.2-schema.sql:705–730`共24列，外键8613–8669均deferred、无数据库ON DELETE，索引4742–4796为单列审计/引用btree。Nerve `00028_issue_draft_issues.sql`实际九列如下；未复制四张关联表（Plane draft.py:139–215）。

| Nerve列 | Plane与本期变化 | Nerve约束/外键 |
|---|---|---|
| id | 保留uuid；应用生成UUIDv7 | PRIMARY KEY |
| workspace_id | 保留非空uuid | REFERENCES workspaces ON DELETE CASCADE，立即检查 |
| project_id | 保留nullable uuid，不藏进payload | REFERENCES projects ON DELETE CASCADE，立即检查 |
| payload | 新增jsonb；承载name/description_html/state_id/priority/assignee_ids/label_ids/start_date/target_date/parent_id，字段均可省略 | NOT NULL DEFAULT '{}'，CHECK jsonb_typeof='object'；封闭键/类型由API检查 |
| created_by_id | Plane nullable改为非空；不在公开Draft返回 | REFERENCES users ON DELETE CASCADE |
| updated_by_id | 保留nullable，不在公开Draft返回 | REFERENCES users ON DELETE SET NULL |
| created_at | 保留非空timestamptz | 新增DEFAULT now()兜底，请求显式写唯一At |
| updated_at | 保留非空timestamptz | 新增DEFAULT now()，JSONB等值noop不推进 |
| deleted_at | 保留nullable timestamptz | 软删沿请求原At/actor；旧删除行审计不重写 |

删除原16列：name、description_json/html/stripped/binary、priority、start_date、target_date、sort_order、completed_at、external_source/id、estimate_point_id、parent_id、state_id、type_id。其中九个payload字段只保留存储值，parent/state不设实体外键；估算/type/外部导入字段不保留，迭代/模块不加入P8 payload。附件表尚未在本期建出或迁移，原 `file_assets` 行是M5接手设计。

保留主键后新增三个索引：活草稿 `draft_issues_author_created_at_idx(workspace_id,created_by_id,created_at DESC,id DESC) WHERE deleted_at IS NULL`；无条件 `draft_issues_workspace_id_idx` / `draft_issues_project_id_idx`支持物理FK和级联。不复制Plane审计者或state/parent/type/estimate单列索引，不加title/sequence/version列。

### 草稿保存、权限与发布

| 核对点 | Plane实际源码 | Nerve已交付 |
|---|---|---|
| 保存校验/副作用 | `P/app/serializers/draft.py:33–140`实体引用及日期/HTML/binary检查、状态/父项/估算所属项目、负责人/标签目录过滤；`P/db/models/draft.py:84–132`默认state、completed_at、stripped和sort_order | 只封闭结构/类型及三个自由字符串NUL可存性；允许空标题、坏priority、无效引用先保存，发布再按创建检查。payload整替换，只有外层project nullable；不发issue事件/活动/号序/版本/关注 |
| 草稿角色/归属 | `P/app/views/workspace/draft.py:98–121` list/create装饰器含guest；list/retrieve/PATCH按created_by；156–203 PATCH成功204，destroy函数按workspace/pk读取，装饰器另有限制 | 六操作唯一issue_draft.write，工作区admin/member自己可用、别人404，guest403；PATCH200。未展开Plane装饰器全实现，不把destroy本体直接推成完整权限漏洞 |
| 无项目与项目窗口 | `draft.py:219–234`无项目400，项目来自草稿 | 无项目422；旧项目删除草稿404，显式新project不可见/已删422；退出/归档项目仍列个人草稿，发布按当前create权限/归档409 |
| 发布载荷 | `draft.py:205–234`限定owner，IssueCreateSerializer(data=request.data)，草稿只供项目/工作区/defaultAssignee上下文 | 先PATCH再无body发布存储payload，任何非零字节含空白400；D锁下重读、项目改变409，不接受正文覆盖 |
| 副作用/事务 | `draft.py:236–318`另行排issue/cycle/module活动、迁移FileAsset，再调用delete；`P/db/mixins.py:72–78`默认软删并排关联任务 | 唯一创建核心与草稿软删同事务/actor/At，三真实失败注入整体回滚；附件留M5。未见该函数显式atomic不等于Plane全局没有请求事务 |

### 全局搜索与项目选择器分路

Plane `P/app/urls/search.py:12–25`的workspace `search/` 与project `search-issues/`是不同入口，不能用一路替代另一条。

| 核对点 | Plane实际源码 | Nerve已交付 |
|---|---|---|
| 全局issue匹配/响应 | `P/app/views/search/base.py:83–114,271–302` name/project identifier icontains OR从query提数字匹配sequence；有效项目成员/未归档项目，最多100简要行，`results.issue`；`P/db/models/issue.py:92–100,172–178`manager排软删/归档/triage/draft，默认-created_at | 统一issue-search七键、完整Issue的data，无page/total；空词updated_at/id降序，非空精确编号/纯数字/标题OR后rank/similarity/created_at/id降序，不搜索stripped |
| 选择器匹配/范围 | `P/utils/issue_search.py:14–24`也用name/identifier contains与数字提取（query长度≤20）；`P/app/views/search/issue.py:99–160`默认路径project、workspace_search可跨项目，最多100简要行裸数组；guest过滤只对路径project查role=5 | 完整编号使用可见目录Unicode折叠，纯ASCII数字允许前导零且有界；逐候选项目OwnOnly含adminGuest，目录/source/SQL/seen同RO/RR；不声称Plane逐候选执行本项目OwnOnly |
| 层级/关系/日期 | `search/issue.py:37–76,123–139`parent排source/当前parent/直接children，sub限root并排当前parent；relation双向，另有cycle/module及target_date=none | parent/sub固定source项目，排全部后代/祖先，递归穿隐藏/删/归档中间节点；relation双向全部活类型。两日期完整词汇、重复非none AND、none互斥；权限/日期/排除在LIMIT前 |
| 错误/字面词 | 上述旧入口使用不同query名及ORM icontains，未提供本项目严格七键契约 | Unicode100/101、非法UTF8/NUL、未知/重复标量、空for/source依赖与scope-before-conflict均明确400/422/404；LIKE特殊字符按字面转义，不做纠错 |

### trigram、活链接数量与排序

本次定向检索 `P/db/models/`、`P/db/migrations/`及 `P/utils/issue_search.py` 的trigram/pg_trgm/GinIndex/similarity无命中；上列两路搜索实际不调用similarity。因此00029扩展/部分GIN与rank后的similarity排序是本项目新增；不推断Plane其他源码/部署从无扩展。Nerve Down只删索引，预存共享扩展OID保留；runtime默认PUBLIC EXECUTE及owner数据库CREATE条件见README。

Plane已有活links计数：`P/db/models/issue.py:371–381`的IssueLink普通metadata，默认manager `P/db/mixins.py:56–67`排deleted_at；`P/app/views/issue/base.py:130–133,234–239`数id，无作者或metadata.error谓词；`P/app/serializers/issue.py:738–749,780–813`部分serializer公开IntegerField link_count。本项目沿用活数语义，要求每个Issue/Detail非负int64、零显式返回。base.py按issue聚合Subquery未Coalesce，而sub_issue.py:52–62明确Coalesce0，只能说明源码形式不统一，未实测断言全部响应为NULL。

Plane `P/utils/order_queryset.py:18–33,153–157`当前排序allowlist无link_count；Nerve保留P5原19码、新增±link_count成当前21，全平铺/分组/子组/成员/偏好共用词汇及整数/numeric游标。不加link_count筛选、不复制派生列到issues；metadata/URL/title更新不改数量。跨请求链接变化仍是弱快照边界，后续生产前端比较器/页面由P10b–P15接手。
