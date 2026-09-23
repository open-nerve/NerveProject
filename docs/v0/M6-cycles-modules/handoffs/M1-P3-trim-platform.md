---
status: open
from: M1/P3
to: M6
created: 2026-09-24
---

# 迭代：进度地址没有结尾斜杠，两个进度类型已删除

- 迭代进度仍从 Plane 的 `GET /api/workspaces/<slug>/projects/<id>/cycles/<id>/analytics?type=issues` 读取（`web/apps/web/core/services/cycle.service.ts`）。这个地址没有结尾斜杠，靠 Django 的 `APPEND_SLASH` 重定向。M6 换成新的进度接口时一并处理。关键词守卫为它登记的例外（规则 `analytics`，`until: "M6"`，见 M1/P2 的交接）随之变陈旧，届时从 `tools/keywords.json` 删掉。
- `TCycleProgress`、`IWorkspaceProgressResponse` 两个类型已删除（M1/P2 评审点名的死导出），新接口不需要提供它们。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
