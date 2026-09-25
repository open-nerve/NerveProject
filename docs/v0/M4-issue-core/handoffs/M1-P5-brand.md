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
