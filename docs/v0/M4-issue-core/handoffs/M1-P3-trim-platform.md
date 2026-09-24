---
status: open
from: M1/P3
to: M4
created: 2026-09-24
---

# 工作项：前端不再使用的字段和接口，留给 M4 决定的筛选显示层和收集箱来源

M1/P3 删掉了 Epic、工作项类型、团队、批量操作与多选、工时（Task 3）和评论的可见范围（Task 2）。M4 设计工作项、动态、评论和筛选的接口时，与下面这些保持一致。

## 不再读、不再写的字段

- 工作项：`is_epic`、`type_id`。显示属性和筛选项没有 `issue_type`，分组方式没有 `team_project`。
- 动态：不再有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY` 两类。
- 评论：不再读、不再写 `access`（M1 设计 3.8：评论没有"内部 / 外部"之分）。公开页的工作项、评论、表情、投票类型已删除。

## 地址

- 描述历史和按编号查找仍用 `/work-items/` 地址：
  - `…/projects/<id>/work-items/<id>/description-versions/`（`work_item_version.service.ts`）；
  - `/api/workspaces/<slug>/work-items/<项目标识>-<序号>/`（`IssueService`）。

  其余工作项地址是 `/issues/`。工作项的 service 类型（`EIssueServiceType`）已整个删除，这两处是字面量。
- 删除了 Plane 不路由、也没人调用的方法：`IssueService` 上的 `deleteIssueRelation`（关联仍由 `IssueRelationService` 删除）、`get/updateIssueDisplayProperties`、`bulkSubscribeIssues`，`UserService.userIssues`，`ViewService.getViewIssues`，状态的 `PUT` 更新（store 用的是 `PATCH`）。
- 没有批量操作，前端也不再有多选。`IssueService` 的 `bulkDeleteIssues`、`bulkArchiveIssues` 和各 store 的 `removeBulkIssues`、`archiveBulkIssues` 已删除：批量归档在基点就没有调用方；命令面板的"批量删除工作项"弹窗在基点就打不开（没有任何代码把它的开关设为打开，Plane 上游的社区版也一样）。M4 不需要提供批量删除或批量归档接口。

## 留给 M4 决定

- **富文本筛选的"显示用"一层。** 合并企业版扩展点之后，`TAllAvailable*ForDisplay`（`web/packages/types/src/rich-filters/`）、否定运算符的标签这一层仍在。它不是企业版接缝，而是 Plane 为筛选界面做的映射。`useFiltersOperatorConfigs`（`web/apps/web/core/hooks/rich-filters/use-filters-operator-configs.ts`）目前返回常量：全部运算符、`allowNegative: false`，它的 `workspaceSlug` 参数不读。M4 重做工作项筛选时，决定是否支持否定运算符，并据此保留或删掉这一层和这个 hook。
- **收集箱的来源。** `EInboxIssueSource` 有 `IN_APP`、`FORMS`、`EMAIL` 三个取值。前端只创建 `IN_APP`（`inbox-issue.service.ts`），另外两个只在显示时读到：收集箱详情对 `FORMS` 显示"Intake Form user"（`web/apps/web/core/components/inbox/content/issue-root.tsx`），工作项动态对非 `IN_APP` 的来源显示来源名（`issue-activity/activity/actions/default.tsx`）。前端没有提交表单或邮件的入口，这两个值只能来自 Plane 后端。M4 定义收集箱接口时决定保留哪些来源，前端随之删掉不再出现的取值和这些显示分支。项目功能设置里收集箱的说明已改为只描述保留的用法（"在将工作项加入项目之前，先进行审议和讨论"），原来的"允许非成员提交"说的是被删的收集表单；如果 M4 保留表单或邮件来源，再相应改写。

## 关闭条件

M4 合并时：

- 工作项、动态、评论的接口没有 `is_epic`、`type_id`、`WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`、`access`；
- 描述历史和按编号查找的地址与其余工作项地址统一；没有批量删除、批量归档接口；
- 富文本筛选的显示层和 `useFiltersOperatorConfigs` 有结论：支持否定运算符并用上它们，或者删掉；
- 收集箱来源的取值已定，前端删掉不再出现的取值和它们的显示分支。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
