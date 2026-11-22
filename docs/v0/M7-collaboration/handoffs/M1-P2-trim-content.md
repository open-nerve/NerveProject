---
status: open
from: M1/P2
to: M7
created: 2026-09-23
---

# 收藏、最近访问与通知：已删除功能的实体类型

M1/P2 删掉了文档页、便签、仪表盘等功能。M7 设计收藏、最近访问和通知的接口时，按下面几点处理：

- **收藏和最近访问**：不再有 `page` 类型。首页的最近访问只显示工作项和项目（P2 评审附录 6.2 的浏览器探测验证过"有数据"和"空状态"）。
- **首页接口**：首页的最近访问目前读取 Plane 的 `…/recent-visits/`，由 M7 替换。首页其余部分已经固定，不再有组件开关和快捷链接接口，M7 不需要提供它们。
- **通知**：通知里的实体不再出现已删除功能的类型，例如文档页、便签、估算相关的字段变更。前端的预览和"全部已读"已由附录 6.4 的探测验证：它发送 `POST …/mark-all-read/`，请求体为 `{"snoozed": false, "archived": false}`。

## 关闭条件

M7 合并时：收藏和最近访问的接口没有 `page` 类型；首页的最近访问改用新接口；通知的实体没有已删除功能的类型。

**处理结果（M7/P3c，收藏的部分）：** 收藏的 `entity_type` 只有 `project`、`cycle`、`module`、`view`、`folder`，没有 `page`（[P3c spec](../specs/P3c-favorites.md) §2.3、§2.4）。最近访问的接口随 P3d、首页随 P7，保持 open。

**处理结果（M7/P3d，最近访问的接口）：** 最近访问的 `entity_type`（`RecentVisitKind`）只有 `issue`、`project`，没有 `page`（[P3d spec](../specs/P3d-recents.md) §2.4）。“首页的最近访问目前读取 Plane 的 `…/recent-visits/`”已过时：首页小部件随 M3/P8a 的 `5c128973` 删除，首页今天只有空状态（`web/apps/web/core/components/home/home-body.tsx`），没有最近访问的读者；首页的“最近”随 P7 读新接口 `GET /workspaces/{slug}/recent-visits`。保持 open。

**处理结果（M7/P4，通知的部分）：** 通知卡片只渲染契约的字段：`describeChange` 按 `NotificationChange` 的 `(field, verb)` 给文案，契约的 `field` 里没有文档页、便签、估算等已删除功能的字段，以后多出的字段只说“更新了工作项”（[P4 spec](../specs/P4-web-notifications.md) §2.6、R11）。全部已读发生成的 `NotificationMarkAllRead`（`POST /api/v0/workspaces/{slug}/notifications/mark-all-read`，`tab`、`archived`、`snoozed`，“全部”页签选了类型时加 `type` 数组），不再是 Plane 的 `POST …/mark-all-read/` 与 `{"snoozed": false, "archived": false}`（§2.2、R9）；旧的预览 hook 与旧数据层随 T5 删除。首页的最近访问随 P7，保持 open。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
