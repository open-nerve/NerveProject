---
status: open
from: M0/P4
to: M2
created: 2026-09-22
---

# M2 第一次按快照建表时的约定

## 1. 第一次按快照建表时一次定下的约定

第一次按快照建表时，一次定下以下约定，以后各个 M 照做：

1. Plane 的 474 个外键全部是 `DEFERRABLE INITIALLY DEFERRED`，都没有 `ON DELETE`（Django 在 Python 中处理级联）：Nerve 的外键是否保留 `DEFERRABLE`，每类关系用什么 `ON DELETE`，要与"连带删除在同一事务中同步完成"（[v0 设计](../../v0-design.md) 5.5）一起考虑。
2. 约束和索引名带 Django 的哈希后缀（例如 `label_workspace_id_c4c9ae5a`、`accounts_user_id_7f1e1f1e_fk_users_id`、`…_uniq`）：沿用，还是改为可读的命名规则。
   - **命名不对应的证据**：29 张表的主键名字和表名不对应（例如 `labels` 的主键叫 `label_pkey`，`issues` 叫 `issue_pkey`，`project_members` 叫 `project_member_pkey`），因为 Django 改 `db_table` 时不会跟着改约束名。这进一步支持改为可读命名规则的方案。
3. 快照中没有列默认值，也没有枚举 CHECK：默认值和取值范围要从 Plane 源码的 `apps/api/plane/db/models/` 中读取，再写进数据库（差异清单二·全局）。
4. 快照中的 14 张系统表不照搬。
5. **Django 的索引产物**，一次定下怎么处理：
   - 19 个 `*_like` 的 `varchar_pattern_ops` 索引，其中 6 个在保留的表上（`users` ×2、`api_tokens`、`workspaces`、`projects`、`states`）；
   - 每个外键列都有一个 btree 索引（包括 `created_by_id`、`updated_by_id`），部分与唯一索引重叠（例如 `cycle_issues_issue_id_2d5ac97f` 与差异清单要删除的 `(issue_id, cycle_id, deleted_at)` 唯一索引重叠，需要重新核对覆盖关系）；
   - `varchar(n)` 还是 `text`：114 个 `varchar(255)` 列。

## 2. 主键与 ID

94 张业务表的主键是没有默认值的 `id uuid`，与"由应用生成 UUIDv7"（[v0 设计](../../v0-design.md) 5.5）一致；`sessions`（主键 `session_key`）、`project_identifiers`（整数 identity 主键）是例外，两张都不保留。

来源：[M0/P4 评审记录](../../M0-foundation/reviews/P4-plane-schema-review.md)。
