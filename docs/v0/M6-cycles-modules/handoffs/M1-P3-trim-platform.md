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

## 处理结果（M6/P4）

新的进度接口是 `GET /api/v0/cycles/{cycle_id}/progress`（生成的类型在 `api-client` 的 `schema.gen.ts`，`CycleProgress`），没有结尾斜杠的问题；模块另有 `GET /api/v0/modules/{module_id}/progress`。没有工作区级的进度，新接口没有 `TCycleProgress`、`IWorkspaceProgressResponse` 的对应结构（[P4 spec](../specs/P4-progress-transfer.md) §2.3）。网页改用新接口在 M6/P5，本交接保持 open。证据：提交 `a92a292387db1105e97a7a52b8e4b9ac6d7d70bb`。

## 处理结果（M6/P5a）

迭代的进度改用 `GET /api/v0/cycles/{cycle_id}/progress`（没有结尾斜杠的问题）；读 Plane 地址的旧 `core/services/cycle.service.ts` 与它的测试删除，新规则 `old-cycle-urls`（以 `/api/workspaces/` 前缀锚定，同 `old-issue-urls`）拦住旧的迭代接口地址；新接口没有 `TCycleProgress`、`IWorkspaceProgressResponse` 的对应结构，`@nerve/types` 的 `cycle/` 整目录删除。关闭条件的迭代部分已满足，与 M1/P2 的交接一起在收尾关闭。证据：提交 `b4911394dc2a92d4bd0224e572902e6d45275941`、`f497055055f00e0cdbfa74e08f5186d40a1297d3`；[P5a spec](../specs/P5a-web-cycles.md) §2.7、§2.8。
