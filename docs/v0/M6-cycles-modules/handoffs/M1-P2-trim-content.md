---
status: open
from: M1/P2
to: M6
created: 2026-09-23
---

# 迭代与模块的进度只按工作项数计算

M1/P2 删掉了估算（Task 4），迭代和模块的进度改为只按工作项数计算。计算逻辑在 `web/packages/utils/src/cycle.ts`（`calculateCycleProgress`）和 `distribution-update.ts`（乐观更新），有 13 个单元测试（`web/packages/utils/src/progress.test.ts`）。M6 设计进度接口时：

- **不要提供**以下字段：`points`、`estimate_distribution`、各类 `*_estimate_points`（例如 `completed_estimate_points`、`total_estimate_points`），以及快照里的点数分布；
- **只提供**按状态组计数的数据（已完成、已开始、未开始、积压、已取消），当前迭代和已结束迭代的快照也一样。

另外，关键词守卫对 `web/apps/web/core/services/cycle.service.ts` 登记了一条例外（规则 `analytics`，`until: "M6"`）。前端的迭代进度目前仍从 Plane 的 `…/cycles/<id>/analytics?type=issues` 地址读取；M6 用新的进度接口替换它之后，这条例外会变陈旧、守卫失败，届时把例外从 `tools/keywords.json` 删掉。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
