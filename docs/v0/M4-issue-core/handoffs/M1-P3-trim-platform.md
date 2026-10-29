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

## 处理结果（M4/P1a）

- **工作项的字段**（工作项的第一个操作一侧完成）：`createIssue` 的 `IssueCreate`、`IssueDetail` 没有 `is_epic`、`type_id`，`issues` 表也没有 `type_id`（M4 设计 4.2）。没有批量删除、批量归档的接口。

仍未处理，状态保持 `open`：动态的字段名里没有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`（M4 设计 3.13 的表）、评论没有 `access`（都在 M4 的 P4）；描述历史和按编号查找的地址（M4 的 P4、P2）；富文本筛选的显示层（M4 的 P10b，关键词守卫的终态在 P16）；收集箱的来源（以转交关闭：收尾写进 M7 的交接；M4 删除工作项动态中显示来源的一支在 P15，M4 设计 13.1）。

来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

## 处理结果（M4/P2）

- **地址，按编号查找的一半**（接口一侧完成）：按编号取工作项是 `GET /api/v0/workspaces/{slug}/issues/{identifier}`（`getIssueByIdentifier`，M4 设计 3.7、3.18），与其余工作项的接口同在 `/api/v0` 下；前端的 `/work-items/` 字面量随前端改接新的 service 删除（关键词守卫 `old-issue-urls`，M4 的 P10a 起）。

仍未处理，状态保持 `open`：动态的字段名里没有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`（M4 设计 3.13 的表）、评论没有 `access`（都在 M4 的 P4）；描述历史的地址（M4 的 P4）和前端的两处 `/work-items/` 字面量（两个 service 在 M4 的 P11 换掉，`browse` 的路由在 P14 改经按编号取）；富文本筛选的显示层（M4 的 P10b，关键词守卫的终态在 P16）；收集箱的来源（以转交关闭：收尾写进 M7 的交接；M4 删除工作项动态中显示来源的一支在 P15，M4 设计 13.1）。

来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

## 处理结果（M4/P4）

- 评论DTO/请求无 access、parent_id、comment_json，动态不含 WORKLOG 或 ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY，也不返回 comment 或完整资源。T3/4/12契约、T14矩阵与T15 API故事已验证。
- 描述历史服务端统一为 `GET /api/v0/issues/{issue_id}/description-versions`、`GET /api/v0/issue-description-versions/{version_id}`；还原用普通 PATCH。T10/T15已验证；P2按编号地址保持既定契约。
- 余项：P11 前端两个旧 work-items 地址及service/store换型；P10b/P16筛选显示层与关键词清理；P15删除动态中收集箱来源旧分支，收集箱来源决定经M4收尾转M7。无批量删除/归档接口。

状态保持 `open`；本次仅完成P4接收部分，待M4真正收尾逐项核对关闭条件。证据：[P4 spec §4/§7](../specs/P4-issue-collaboration.md#4-验收标准)、[P4 review §2/§6](../reviews/P4-issue-collaboration-review.md#2-验收标准核对)。


## 处理结果（M4/P10a，2026-10-28）

本期删除四类领域工作项列表、四旧store kind与相关布局/字段/表单读取，直接生成Issue/IssueDetail未恢复is_epic/type_id等旧核心字段；其他动态/评论DTO的完整改接仍属P11。old-issue-urls已按八类完整segment守卫登记精确file+match/count/until，普通按编号/work-items与描述版本的前端旧路径仍由P11按M4/P12到期删除。`WorkspaceService.searchWorkspace`的`/search/`当前无自动guard，按广义手工账由P15删除/until M4/P16，不称已被新旧规则覆盖。准确方法与到期证据见[P10a移交](../reviews/P10a-web-issue-types-review.md#6-移交事项)。

富文本筛选否定显示层留P10b、守卫终态留P16；收集箱只准确化create PartialDetail、response union、metadata所属DTO与独立旧update请求stringflag，未决定新的来源取值或完成来源显示迁移。P15删除工作项动态中旧来源分支，M4收尾向M7转交来源决定及原文细节；status保持open，不以本期类型边界修正关闭原输入。

本期Task级完成至T8提交8081112492a2fc744a43b3e814c6d36ff4ba3959；A-01/A-02代码终审修复已独评并提交5f7085bce4aa5103a0133fa982847d274efa6e47；文稿最后独评已通过，最终交付HEAD分支CI待提交后核验，准确产品5f708 CI已核成功见review§8，不把本段子项写成M4最终交付或集成。


整审A-01新真实测试旧subissues字面量精确count1/untilM4/P12已审，最后73rules/53exceptions、old-issue41hits/29keys；T7/T8的40/28仍是历史检查点，生产旧请求未恢复、rule未放宽。

## P10b 本段处理（2026-10-29）

否定显示层及 hook 已由 T8 删除，negation-filters 守卫及 T13 M43 反例见阶段 review；选择器呈现归 P13，P16 核终态。 证据见 [P10b review](../reviews/P10b-web-list-data-review.md)；最终F、CI与整审待办，整份交接 status 保持 open。
