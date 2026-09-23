---
status: open
from: M1/P2
to: M4
created: 2026-09-23
---

# 工作项：前端不再使用的字段和接口，以及一个继承的描述保存问题

## 不再使用的字段和接口

- `estimate_point`：估算已删除（M1/P2 Task 4）。
- `POST …/issue-dates/`：只有拖动甘特图的条会调用，甘特图已删除（Task 5）。
- 搜索结果里的 `page` 类型：文档页已删除（Task 7）。
- `description_binary`：协作编辑的 Yjs 二进制，前端不再读取也不再写入（Task 7）。工作项描述只用 `description_html`。

## 只补 id 的迁移更新

编辑器给每个块节点一个 `data-id`（`UniqueID`）。打开一个缺 id 的旧描述时，编辑器会补上 id 并发出

```
PATCH …/issues/<id>/  {"description_html": "…", "skip_activity": "true"}
```

M4 的描述更新接口要照此处理：`skip_activity` 为真时不记动态。M1/P2 修复了一处回归（`79ef2b5`）：一旦出现只读编辑器，之后的编辑器都不再补 id。编辑器的交互测试（`web/packages/editor/src/editor-interaction.test.ts`）和 P2 评审附录 6.3 的浏览器探测都覆盖这一点。

## 继承自 Plane 的描述保存问题（不是 P2 引入的）

`web/apps/web/core/components/editor/rich-text/description-input/root.tsx`（自导入 `48e1a63` 以来未改）：

- **现象**：如果在 1.5 秒的保存延迟内先撤销、再重做，最终发出的保存带的是撤销后的内容，而编辑器里显示的是重做后的内容。服务端的这份描述要到下一次编辑才会被纠正。
- **原因**：撤销时，安排了一次带撤销后 HTML 的保存；重做时，HTML 等于上次保存的内容，于是提前返回，待发的那次保存没有被替换。

M4 重写描述保存时一并修复。复现步骤和输出见 [M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)附录 6.3 第 7 节。

来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
