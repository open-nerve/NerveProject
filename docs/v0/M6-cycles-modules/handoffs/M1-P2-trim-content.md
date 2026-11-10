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

## 关闭条件

M6 合并时：进度接口只提供按状态组的计数，没有点数字段；前端的迭代进度不再读 `…/analytics?type=`，守卫的 `analytics` 例外已从 `tools/keywords.json` 删掉（与 M1/P3 的交接一起关闭）。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

## 处理结果（M6/P1）

进入 M6/P1 时关键词守卫的 `phase` 改为 `M6/P1`，`analytics` 规则对 `core/services/cycle.service.ts` 及其测试的两条例外（`until: "M6"`）随之到期，已删除；同一提交删去指向 Nerve 没有的旧地址 `…/cycles/{id}/analytics?type=issues` 的 `CycleService.cycleDistribution`、`CycleStore.fetchActiveCycleDistribution` 和进行中迭代卡片对它的调用。`calculateCycleProgress` 与它的测试保留。只按状态组计数、没有点数字段的进度接口在 M6/P4，进行中迭代卡片的分布与燃尽由 M6/P5 依新接口重建，本交接保持 open。证据：[P1 spec](../specs/P1-cycles-backend.md) A7。

## 处理结果（M6/P4）

进度的接口只按状态组计数，没有点数：迭代的 `GET /api/v0/cycles/{cycle_id}/progress`（`CycleProgress`：`live` 与 `progress_snapshot`）、模块的 `GET /api/v0/modules/{module_id}/progress`（`Progress`），以及迭代、模块列表行的 `CycleListItem`、`ModuleListItem`，都只有六个计数（总数与五个状态组）、负责人与标签的分布（每行总数、已完成、未完成）和燃尽，没有 `points`、`estimate_distribution`、`*_estimate_points`；存下的冻结 `cycles.progress_snapshot` 就是 `FrozenProgress`，同样没有点数（`api/modules/planning.yaml`，[P4 spec](../specs/P4-progress-transfer.md) §2.3、§2.4）。网页改读新接口、重建进行中迭代卡片的分布与燃尽在 M6/P5，本交接保持 open。证据：提交 `735cac67f04519adee7d6a046051cc9dcefabe05`（计算与冻结的格式）、`a92a292387db1105e97a7a52b8e4b9ac6d7d70bb`（进度接口）、`e61752481bdec426020f989d743119d72230b903`（列表行）。
