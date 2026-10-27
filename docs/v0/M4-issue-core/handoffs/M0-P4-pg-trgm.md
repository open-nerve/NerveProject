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

## 处理结果（M4/P8，2026-10-27）

服务端子项已承接：T9迁移00029在postgres:18.6镜像内由rolsuper=false且具数据库CREATE的owner安装pg_trgm，建活issues.name部分GIN；预存扩展Down/Up OID保持，Down只删本迁移索引。runtime真实similarity/草稿CRUD及禁止DDL、全关系/函数目录、sqlc原生float32均验证。T15恰好10万中英文Store+RO/RR实测（PG18.6/libc/en_US.utf8），最终SQL/17实参/EXPLAIN/5warm与first-observed有原件；八标题场景顺扫、单项目login bitmap，不保证三字符必走索引。README已说明镜像扩展、非superuser数据库CREATE、runtime PUBLIC EXECUTE依赖及收回后的显式函数授权。最终locale/部署/升级维护仍交M8，顶部status继续open。

准确产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6`，首次整审 `166b9efd..10f44790`；[P8 review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)列A条款/真实RUN/变异恢复/四次七门禁及准确产品checkpoint CI，[整体审核](../../../reviews/re-07-M4P8整体审核.md)列三镜头/186文件覆盖与历史D1。文档增量独评及最终准确HEAD CI由后续提交和ignored收据追踪，未合并、不清理；不以本期子项已处理改写历史关闭条件或M4整体状态。
