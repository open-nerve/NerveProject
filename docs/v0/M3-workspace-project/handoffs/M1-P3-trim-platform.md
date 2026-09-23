---
status: open
from: M1/P3
to: M3
created: 2026-09-24
---

# 工作区与项目：前端不再使用的字段、成员的添加方式、保留地址

M1/P3 删掉了公开发布（Task 2）、企业版项目功能（Task 3）、计费（Task 5），并把项目"邀请"改为从工作区成员中添加（Task 6）。M3 设计工作区、项目和成员的接口时，与下面这些保持一致。

## 不再读的字段

- `IProject.anchor`、视图的 `anchor` 和发布设置、收集箱的 `anchors` 和 `is_form_enabled`：公开发布已删除，前端不再调用任何发布地址。
- 项目动态不再显示 `is_project_updates_enabled`、`is_epic_enabled`、`is_workflow_enabled`、`is_time_tracking_enabled`、`is_issue_type_enabled` 的变化。新的项目动态不需要产生这几类记录。
- 项目功能列表不再有 `isPro`、升级徽标。

## 项目成员

- 只有一种方式：从工作区成员中直接添加（`AddProjectMembersModal`，标题"添加成员"）。web 里本来就没有按邮件邀请进项目的流程。
- 加入公开项目的 `joinProject` 仍调用 `POST /api/users/me/workspaces/<slug>/projects/invitations/`（`web/apps/web/core/services/user.service.ts`）。关键词守卫为它登记了一条例外（规则 `project-invitations`，`until: "M3"`）。M3 定义项目成员接口、替换这个地址之后，例外会变陈旧、守卫失败，届时从 `tools/keywords.json` 删掉它。
- 工作区邀请的两条路径不变（M1 设计 3.15）。

## 保留的工作区地址

`RESTRICTED_URLS`（`web/packages/constants/src/workspace.ts`）本 Phase 去掉了 `god-mode`、`spaces`、`epics`、`epic`、`plane-pro`、`plane-ultimate`、`enterprise`、`plane-enterprise`、`upgrade`、`billing`、`silo`，以及重复的 `config`、`mobile`、`monitor`；名单现在没有重复，由 `navigation.test.ts` 守住。剩下的 45 个里还有 Plane 的产品词（`one`、`business`、`pro`、`license`、`licenses`、`initiatives`、`initiative`、`workflow`、`workflows`、`story`、`disco`、`drive`、`channels` 等），M3 与后端的工作区 slug 保留名单一起决定留哪些。

## 地址

- `ProjectService.checkProjectIdentifierAvailability` 的地址没有结尾斜杠（`/api/workspaces/<slug>/project-identifiers`，靠 Django 的 `APPEND_SLASH` 重定向）。M3 重新定义这个接口时一并处理。
- 工作区设置只剩 `general`、`members`、`webhooks` 三个标签页，没有空分组（`navigation.test.ts`）。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
