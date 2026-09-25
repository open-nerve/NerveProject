---
status: open
from: M1/P2
to: M3
created: 2026-09-23
---

# 项目、工作区与侧边栏：前端已经不再使用的字段和接口

M1/P2 删掉的功能在前端留下了下面这些约定。M3 设计工作区、项目和成员的接口时，与它们保持一致。

- **项目字段**：`IProject`（`web/packages/types/src/project/projects.ts`）不再有这些字段，新接口不要提供：
  - `close_in`、`default_state`：自动关闭及其目标状态。新工作项的默认状态来自 `State.default`，与 `default_state` 无关；
  - `page_view`：文档页开关；
  - `estimate_id`：估算。

  项目的"自动化"设置只剩自动归档（`archive_in`）。
- **侧边栏偏好**：前端不再调用 `/sidebar-preferences/`。侧边栏是常量里的固定列表（`web/packages/constants/src/workspace.ts`，由 `navigation.test.ts` 守住）。项目导航偏好（手风琴或标签页、显示的项目数）仍然存在，由 `ProjectNavigationDialog` 通过工作区用户属性保存。
- **保留的工作区地址**：`RESTRICTED_URLS`（`web/packages/constants/src/workspace.ts`）已去掉 `import`、`importers`、`integrations`、`integration`、`pages`、`live`，后端的工作区 slug 保留名单要与之一致。`silo` 以及重复的 `config`、`mobile` 由 M1/P3 处理。
- **个人主页**：个人主页只剩用户卡片和工作项分页，`/profile/:userId` 重定向到"分配给他的"。卡片按工作区成员列表区分四种状态：加载中、加载失败、不是成员（包括 `is_active: false`）、是成员。访客能否查看别人的个人主页，由 M3 的权限矩阵决定。

## 关闭条件

M3 合并时：

- `api/` 里的项目接口没有 `close_in`、`default_state`、`page_view`、`estimate_id`，也没有 `/sidebar-preferences/`；
- 工作区 slug 的保留名单与前端的 `RESTRICTED_URLS` 出自同一份来源（与 M1/P3、M1/P4 交接的同一项一起关闭）；
- 访客能否查看别人的个人主页写进 M3 的权限矩阵。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
