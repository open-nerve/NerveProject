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
| `project_user_properties` | 15 列保留 11 列（M3/P4a，`00012_project_project_user_properties.sql`） | M3 设计 4.8 |
| `project_user_properties` | `workspace_id`、`project_id`、`user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `project_user_properties` | `preferences`：新加默认值 `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}` 和 CHECK（外层恰好一个键 `navigation`，它恰好有字符串 `default_tab` 和数组 `hide_in_more_menu`）；`sort_order`：新加 `DEFAULT 65535`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值，去掉 `pages`（文档页已砍）；二·全局（对象型的 `jsonb` 列） |
| `project_user_properties` | 部分唯一索引照搬，改名为 `project_user_properties_project_id_user_id_key ON (project_id, user_id) WHERE deleted_at IS NULL`（Plane 是 `project_user_property_unique_user_project_when_deleted_at_null ON (user_id, project_id)`）；新加不带条件的 `project_user_properties_workspace_id_idx`、`project_user_properties_project_id_idx` | 物理级联要不带条件的索引（M3 设计 4） |
| `project_user_properties` | 删除 `filters`、`display_filters`、`display_properties`、`rich_filters` | 工作项列表的筛选和显示列由它们的使用者 M4 按自己的格式加回（M3 设计 3.18） |
| `states` | 18 列保留 14 列（M3/P4a，`00013_project_states.sql`） | M3 设计 4.9 |
| `states` | `workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `states` | `name`：新加 `CHECK (name <> '')`；`"group"`：新加 `DEFAULT 'backlog'` 和 `CHECK ("group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled', 'triage'))`；`description`：新加 `DEFAULT ''`；`sequence`：新加 `DEFAULT 65535`；`"default"`：新加 `DEFAULT false`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 `StateGroup` |
| `states` | 部分唯一索引照搬，改名为 `states_project_id_name_key ON (project_id, name) WHERE deleted_at IS NULL`（Plane 是 `state_unique_name_project_when_deleted_at_null ON (name, project_id)`）；新加 `states_project_id_default_key ON (project_id) WHERE "default" AND deleted_at IS NULL`、`states_project_id_triage_key ON (project_id) WHERE "group" = 'triage' AND deleted_at IS NULL`，不带条件的 `states_workspace_id_idx`、`states_project_id_idx` | 见第四节"默认状态、分诊状态"（M3 设计 3.17）；物理级联要不带条件的索引（M3 设计 4） |
| `states` | 删除 `slug`、`is_triage` | `slug` 没有读取者；分诊状态只由 `"group" = 'triage'` 识别（M3 设计 3.17） |
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
| 集合型的列表（工作区、成员、项目等） | 一次返回全部，响应是裸数组 | 同样一次返回全部、不分页，响应是 `{"data": [...]}` 封套，每个列表写明顺序（M3 设计 3.12） |
| 关联字段 | 名字不带 `_id`（`workspace`、`member`、`project_lead`），有的内嵌对象 | 只给 id，名字带 `_id`（`WorkspaceMember.workspace_id`）；唯一的例外是工作区成员内嵌成员的公开资料 `member`（`MemberUser`，v0-design 3.6；M3 设计 5.2） |
| 工作区的显示设置的路径 | `/api/workspaces/{slug}/user-properties/`（与工作项的筛选合在一起） | `GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`，只有项目导航的两项（M3 设计 3.18）；没有 `/sidebar-preferences/` |
| 迭代和模块归属 | 通过单独的接口设置 | 作为工作项字段，用 PATCH 修改 |
| 错误 | `{"error": "..."}`、DRF 字段错误、认证错误码三种格式混用 | 统一使用 RFC 9457 |
| 无权限 | 403 | 看不到的资源返回 404，看得到但没权限返回 403 |
| 错误码 | 接口描述中没有 | 每个操作在接口描述中用 `x-problem-codes` 声明它可能返回的错误码；字段错误带 `code`（M2 设计 3.11） |
| 请求体 | DRF 的序列化器忽略未知字段；重复的键以后一个为准（Python 的 `json`） | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段，也拒绝能有两种读法的请求体（同一个对象里重复的成员名，字符串不是合法的 Unicode）：400 `bad_request`，一次列出全部问题（能有两种读法的请求体只列出这两种问题；每个问题只报一次，最多 16 个，路径最长 256 字节；M2 设计 3.11） |
| 个人访问令牌的路径 | `/api/users/api-tokens/`、`/api/users/api-tokens/{id}/`（查看、修改、撤销） | 列出和创建是 `/api/v0/me/api-tokens`，撤销是 `DELETE /api/v0/api-tokens/{token_id}`：单个资源用短路径（v0-design 3.2）；不能查看单个令牌，也不能修改 |

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
| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；服务器管理员的 `nerve users deactivate` 与自助停用相同，`nerve users activate` 恢复账户，恢复后没有过期的 PAT 重新可用；"唯一管理员"的检查由 M3 在同一个事务里实现（M2 设计决策点 3） |
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
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态，在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。标签由 M3/P7 加入这个事务 |
| 邀请的列出、创建、修改、删除 | 工作区管理员和成员；修改不限制角色，成员能把邀请改成管理员 | 只有工作区管理员（M3 设计决策点 4）；邀请的角色因此不高于邀请人（M3 设计 3.8） |
| 邀请令牌 | JWT，原文存库；公开的查看不要令牌，返回被邀请的邮箱；关闭注册时，有任何一份未删除的邀请的邮箱就能注册 | 由签名密钥派生的 MAC（`nrv_inv_` 加 22 个字符），不存库，管理员列出时重新算出；公开的查看要令牌、不返回邮箱；关闭注册时要有效的令牌、且注册邮箱与邀请的相同（M3 设计 3.8、决策点 1） |
| 邀请的链接 | `/workspace-invitations/?invitation_id=…&slug=…&token=…`；另有系统内的接受：`/invitations` 页和新手引导的"加入工作区"一步按账户的邮箱列出发给他的邀请，批量接受 | 只有链接一条路：`/workspace-invitations?invitation_id=…&token=…`；接受要登录，账户的邮箱须与邀请的相同（M3 设计 3.8、决策点 2）（页面：P9；`/invitations` 页和新手引导的一步由 P8–P11 删除） |
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
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`（M3 设计 3.17） |
| 时区 | 只接受 `pytz.common_timezones`；时区列表中负的非整点偏移多算一小时（例如马克萨斯群岛的 −09:30 写成 −10:30） | 接受 Go 的时区数据认得的任何 IANA 名称（`Local` 除外），程序内嵌时区数据；时区列表接口给的仍是同一份常用列表，偏移按请求时刻计算，写法正确（M2 设计 5.3） |
