---
status: open
from: M1/P3
to: M7
created: 2026-09-24
---

# 收藏与通知：实体类型收窄为联合类型，通知卡片不再显示工时

- **收藏的实体类型。** `TFavoriteEntityType`（`web/packages/types/src/favorite/favorite.ts`）是 `"project" | "view" | "cycle" | "module" | "folder"`，收藏的 `entity_type` 按它定类型。图标映射和链接映射都以它为键，去掉了兜底图标，编译器保证每种都有图标；新建文件夹写 `entity_type: "folder"`。M7 设计收藏接口时，后端的取值与这五种一致；以后新增一种可收藏的实体，要先改这个联合类型。
- **通知。** 通知卡片不再显示工时（`estimate_time`，工时已删除）。原来单独导出的"通知内容兜底映射"已并入通知卡片，不再导出。

## 关闭条件

M7 合并时：收藏的 `entity_type` 取值正好是 `project`、`view`、`cycle`、`module`、`folder`，前端的 `TFavoriteEntityType` 改用生成的类型；通知没有工时字段。

**处理结果（M7/P3c，契约的部分）：** 收藏的 `entity_type` 在契约里是 `FavoriteKind`，恰为 `project`、`view`、`cycle`、`module`、`folder`（`view` 兼指项目视图与工作区视图），生成的类型在 `@nerve/api-client`（[P3c spec](../specs/P3c-favorites.md) §2.4）。前端的 `TFavoriteEntityType` 改用生成的类型随 P7，保持 open。

**处理结果（M7/P4，通知的部分）：** 通知没有工时：契约的 `NotificationChange` 没有这个字段，新卡片的文案表也没有（`describeChange` 的单测以 `sort_order` 作“没有文字的字段”的一格，[P4 spec](../specs/P4-web-notifications.md) §2.6）；旧卡片（`workspace-notifications/sidebar/notification-card/content.tsx`）连同并进它的内容映射随 T5 删除。收藏的部分随 P7，保持 open。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
