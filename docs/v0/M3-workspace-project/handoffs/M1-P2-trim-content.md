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

## 处理结果（M3/P1）

- **保留的工作区地址，服务端一侧**（完成）：服务端的保留名单是 `server/internal/modules/workspace/domain/reserved_slugs.txt`，分"应用""服务端""预留"三段（M3 设计 3.10），本文件列出的已去掉的词都不在其中。两个操作都按它回答：建工作区答 422 `validation_failed`，`slug` 字段的码是 `not_allowed`；查 slug 答 200 `{"available": false, "reason": "reserved"}`；"服务端"一段由 `bootstrap` 的 Go 测试核对。

仍未处理，状态保持 `open`：保留名单的前端一侧（`RESTRICTED_URLS` 删除，前后端一份，有测试核对，P8）；项目字段、侧边栏偏好、个人主页，随 M3 设计 13.1 中各自的 Phase。

来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

## 处理结果（M3/P2）

- **侧边栏偏好**（接口一侧完成）：没有 `/sidebar-preferences/`。项目导航偏好是 `GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`，结构 `WorkspacePreferences {navigation_control_preference, navigation_project_limit}`，任何有效成员读写自己的；没有这一行时 `GET` 返回默认值（`ACCORDION`、10），不写库，第一次修改时建行（M3 设计 3.18）。W8 的接口版本核对这两件。`ProjectNavigationDialog` 改调它在 P9（W8 的页面版本）。
- **访客能否查看别人的个人主页**（权限矩阵一侧完成）：能。卡片的数据来自 `listWorkspaceMembers`，它在权限矩阵中有了行：管理员、成员、访客都得到 200，含已结束的成员关系（被移出或已离开，`is_active: false`），卡片据此显示"不是成员"；访客看到的邮箱都是 `null`，他自己的也是（M3 设计 9.2）。个人主页的页面在 P11（故事 P9 的页面版本和 C10）。

仍未处理，状态保持 `open`：保留名单的前端一侧（P8）；项目字段（P4）；个人主页的页面（P11，故事 P9 的页面版本和 C10）。

来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

## 处理结果（M3/P4a）

- **项目字段**（完成）：项目的接口（`api/modules/project.yaml` 的 `Project`、`ProjectCreate`）没有 `close_in`、`default_state`、`page_view`、`estimate_id`，表里也没有这几列（`server/migrations/sql/00010_project_projects.sql`；差异清单二·按表）。新工作项的默认状态来自 `states."default"`，每个项目至多一个（`states_project_id_default_key`）；项目的"自动化"只剩 `archive_in`。

仍未处理，状态保持 `open`：保留名单的前端一侧（P8）；个人主页的页面（P11，故事 P9 的页面版本和 C10）。

来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

## 处理结果（M3/P9）

- **侧边栏偏好**（页面一侧完成）：`ProjectNavigationDialog` 经 `useProjectNavigationPreferences()`（`web/apps/web/core/hooks/use-navigation-preferences.ts`）读写项目导航偏好（`GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`）：每个修改由 `WorkspacePreferencesStore.updatePreferences` 在队列轮到它时作用于 nerve 最近一次回答的设置；设置到达之前侧边栏按 nerve 的默认值（`ACCORDION`、10）显示。W8 的页面版本核对（M3/P9 spec 2.8）。

保留名单的前端一侧在 M3/P8a 完成（`RESTRICTED_URLS` 删除，[P8a spec](../specs/P8a-web-workspace-data.md) 第 1 节）。仍未处理，状态保持 `open`：个人主页的页面（P11，故事 P9 的页面版本和 C10）。

来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。
