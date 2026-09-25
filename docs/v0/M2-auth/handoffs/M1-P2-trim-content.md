---
status: open
from: M1/P2
to: M2
created: 2026-09-23
---

# 用户主题只剩 `{ theme }`

M1/P2 删掉了自定义主题（Task 10）。前端的 `IUserTheme`（`web/packages/types/src/users.ts`）只剩一个字段：

```ts
theme: string | undefined; // a value of THEME_OPTIONS
```

取值是 `THEME_OPTIONS`（`web/packages/constants/src/themes.ts`）各项的 `value`：`system`、`light`、`dark`、`light-contrast`、`dark-contrast`（M1 收尾的 T4 删掉了只列后四个的 `THEMES`，注释随之改写）。M2 设计用户资料接口时：

- 用户主题字段只要这一个值，不要再有 Plane 自定义主题的 `primary`、`background`、`text`、`sidebarBackground`、`sidebarText`、`darkPalette` 等字段；
- 不要再接受或返回 `custom`。前端遇到 Plane 旧数据里的 `custom` 时，页面显示亮色调色板，主题选择框显示占位符，直到用户重新选择。新后端从一开始就不会存这个值。

## 关闭条件

M2 合并时：`api/` 里用户资料或偏好的接口描述中，主题只有 `theme` 一个字段，取值是 `THEME_OPTIONS` 的 5 个 `value`（`system`、`light`、`dark`、`light-contrast`、`dark-contrast`），没有 `custom`，也没有 `primary`、`background`、`darkPalette` 这类调色板字段；前端的 `IUserTheme` 改用生成的类型。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
