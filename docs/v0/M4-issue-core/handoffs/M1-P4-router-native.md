---
status: open
from: M1/P4
to: M4
created: 2026-09-24
---

# 工作项弹窗的挂载与工作项地址的片段

## 个人设置页上的工作项弹窗

命令面板在个人设置 `/settings/profile/*` 上也挂载了工作项弹窗，但那里没有工作区。M1/P4 让路由参数有了真实的类型之后，弹窗在没有工作区时不渲染（`issue-modal/base.tsx`），表单和草稿布局改从弹窗取 `workspaceSlug`。

实际上那里本来就没有打开它的入口：命令面板的"新建工作项"要求有当前工作区，直接打开个人设置和从工作区进入都不提供它，快捷键 `n` `i` 也打不开；基线的提交在没有工作区时也只是提前返回。M4 决定是否把弹窗从个人设置页的挂载中去掉。

## `#sub-issues` 片段

表格的"子工作项"列（`spreadsheet/columns/sub-issue-column.tsx`）跳到 `/:workspaceSlug/projects/:projectId/issues/:issueId#sub-issues`（归档的工作项多一段 `archives/`）。这个地址由加载器重定向到 `/browse/…`，React Router 给加载器的请求不含片段，所以 `#sub-issues` 丢失。没有元素读取这个片段，它本来就不起作用。M4 重做工作项地址时一并决定：直接生成 `/browse/…`（像评论链接的 `#comment-…` 那样），或者去掉片段。

来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。
