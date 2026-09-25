---
status: open
from: M1/P3
to: M6
created: 2026-09-24
---

# 迭代：进度地址没有结尾斜杠，两个进度类型已删除

- 迭代进度仍从 Plane 的 `GET /api/workspaces/<slug>/projects/<id>/cycles/<id>/analytics?type=issues` 读取（`web/apps/web/core/services/cycle.service.ts`）。这个地址没有结尾斜杠，靠 Django 的 `APPEND_SLASH` 重定向。M6 换成新的进度接口时一并处理。关键词守卫为它登记的例外（规则 `analytics`，`until: "M6"`，见 M1/P2 的交接）随之变陈旧，届时从 `tools/keywords.json` 删掉。
- `TCycleProgress`、`IWorkspaceProgressResponse` 两个类型已删除（M1/P2 评审点名的死导出），新接口不需要提供它们。

## 关闭条件

M6 合并时：迭代进度改用新接口，`analytics` 的守卫例外已删掉（与 M1/P2 的交接一起关闭）；新接口没有 `TCycleProgress`、`IWorkspaceProgressResponse` 的对应结构。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
