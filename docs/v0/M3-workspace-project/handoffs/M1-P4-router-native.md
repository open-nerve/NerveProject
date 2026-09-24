---
status: open
from: M1/P4
to: M3
created: 2026-09-24
---

# 保留的工作区名与路由；离开项目的顺序

## 保留的工作区名要与应用的顶层路由段一致

M1/P4 删掉了 Plane 的 10 条旧地址重定向（M1 设计 3.12）。其中 5 个只有一段的旧地址（`/sign-in`、`/signin`、`/login`、`/register`、`/profile`）现在匹配 `/:workspaceSlug`：已登录时显示工作区包装的"Workspace not found"，未登录时先去 `/?next_path=…`。这是路由表对任何未知单段地址的如实回答。

问题在保留名单：`RESTRICTED_URLS`（`web/packages/constants/src/workspace.ts`）里有 `sign-in`、`signin`、`profile`，没有 `login`、`register`；另有一些词已经不是路由。创建工作区的表单和引导的创建步骤在服务端答复"可用"之后再查这份名单，所以前端不会拦下 `login`、`register`。M3 定工作区名的服务端校验时，让保留名单正好是应用的顶层路由段（`web/apps/web/app/routes/core.ts`），前后端用同一份。

## 离开项目：先跳转，后调用接口

`leave-project-modal.tsx` 先跳到项目列表，再调用离开项目的接口（Plane 原有的顺序，P4 的替换保留了它）。接口失败时用户已经离开了项目页，只看到一条错误提示。M3 重写项目成员时决定顺序：先等接口成功再跳转更符合预期。

## 关闭条件

M3 合并时：

- 工作区名的保留名单正好是 `web/apps/web/app/routes/core.ts` 的顶层路由段，前后端用同一份，有测试核对两者一致；
- 离开项目的顺序已定：先等接口成功再跳转，或在 M3 的 review 里写明保留原顺序的理由。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。
