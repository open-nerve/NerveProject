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

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
