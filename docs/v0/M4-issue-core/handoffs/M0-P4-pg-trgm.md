---
status: open
from: M0/P4
to: M4
created: 2026-09-22
---

# M4 工作项标题的 pg_trgm 索引

`issues.name` 的 pg_trgm 索引（[v0 设计](../../v0-design.md) 6.9 已定）要先 `CREATE EXTENSION pg_trgm`：Plane 没有任何扩展，快照中不含它。由 M4 的迁移创建，并确认 Postgres 18 镜像中有这个扩展。

## 部署要求（转给 M8）

`pg_trgm` 自 PG13 起是"受信任扩展"（trusted extension）：拥有数据库 `CREATE` 权限的迁移角色就能安装它，不需要超级用户权限。M8 编写部署文档时，把这条记为部署要求：迁移账户不需要额外的超级用户权限。

来源：[M0/P4 评审记录](../../M0-foundation/reviews/P4-plane-schema-review.md)。
