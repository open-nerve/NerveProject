---
status: open
from: M1/P5
to: M4
created: 2026-09-24
---

# 系统代为创建和操作的工作项怎样显示

## 收集箱的创建者

Plane 为自己的收集箱机器人写了两处特殊显示：收集箱列表里，创建者邮箱含 `intake@plane.so` 的工作项显示一个名为 "Plane"、字母 P 的头像（`inbox/sidebar/inbox-list-item.tsx`）；工作项属性里的"创建者"，名字带 `-intake` 的用户显示为 "Plane"、不显示头像（`issues/peek-overview/properties.tsx`，完整页面在窗口窄于 768 px 时也用它）。

M1/P5 删掉了这两处：Nerve 不区分人和智能体，也没有 Plane 的收集箱机器人，创建者一律按普通用户显示。M4 定收集箱和工作项的接口时，一并决定由系统代为创建的工作项（例如表单或邮件进来的）记在谁名下、界面怎样显示；不要恢复按邮箱或名字猜测的写法。

## 动态里的系统操作者

工作项动态里，自动归档的那一条的操作者是写死的站点名（`issue-activity/activity/actions/archived-at.tsx` 把 `SITE_NAME` 作为 `customUserName`，基线写死 "Plane"），恢复时显示真实的用户。系统操作的动态由谁发出、怎样显示，与上一节一起定。

## 关闭条件

M4 合并时：由系统代为创建或操作的工作项记在谁名下、界面怎样显示，写进 M4 的设计；前端没有按邮箱或名字猜测系统用户的代码；自动归档那一条动态的操作者按这个结论显示。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P5 评审记录](../../M1-frontend-trim/reviews/P5-brand-review.md)第 7 节。

## 处理结果（M4/P6）

T10 真实 SQL/组合读回确认自动归档动态行的 `actor_id` 为 SQL NULL；`issue.archived` 事件不持久化，其 `shared.Event.ActorID` 为 nil，由事件发布组合断言核对。两者都没有零UUID、项目作者或按邮箱/名字伪造的系统账户；T15全门禁及变异复核见 [P6 review §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)。前端站点名显示及系统代建收集箱工作项仍按P15/M7原落点办理，整体 `status: open`。


## P15 协作子项接收（2026-11-01）

T12 的真实 clockTest 页面在新进程重新登录，以 attempted_at 同时晚于新进程和业务写水位确认新归档；页面显示 SITE_NAME（Nerve），SQL 读回 actor_id=NULL，手动恢复显示真实操作者链接。M17 将系统 actor 当停用用户的变异确实失败；T14 三引擎均实际执行 I16。没有按邮箱或名字猜测的实现。系统代建收集箱仍归 M7，顶部 status 保持 open。

证据：[P15 review](../reviews/P15-web-collaboration-drafts-search-review.md#9-逐任务实施记录)、[三浏览器矩阵](../reviews/P15-browser-matrix.md)。


## P15 T16 整审复核（2026-11-01）

本期子项已完成实现、整审及后置 CI 的局部修复与完整本地门禁，最终候选 CI 后置核准，集成待办；顶部/弹窗宿主、偏好晚答和退场选择保护已复验，92 个 TS/TSX 零告警。准确最终分支 CI 与交付状态见 [阶段记录 §10](../reviews/P15-web-collaboration-drafts-search-review.md#10-最终验收与交付) 和 [整审报告](../../../reviews/re-24-M4P15整体审核.md)。顶部 status 保持 open，P16、M7 与 M4 总关闭责任不改变；尚未合并。
