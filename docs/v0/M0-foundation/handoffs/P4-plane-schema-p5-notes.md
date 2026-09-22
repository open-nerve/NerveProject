---
status: done
from: M0/P4
to: M0/P5
created: 2026-09-22
---

# P5 迁入前端时，与参考源码提交相关的注意事项

1. **前端的来源提交是 `02c19e1`，不是 `v1.4.2` 标签**（`5f7d927`）。M0 设计第 1 节锁定的 React Router 8.3.0、pnpm 11.10.0 都来自 `02c19e1`；标签中是 React Router 7.17/7.18、pnpm 11.3.0。见 [P4 spec](../specs/P4-plane-schema.md) 2.3。
2. **`plane/` 在移动的 `preview` 分支上**，迁入前要先确认取到的正是 `02c19e1341d93141e8ad7b3278298adce208bafc`：
   - 用显式提交复制，例如 `git -C plane archive 02c19e1341d93141e8ad7b3278298adce208bafc | tar -x -C web/apps/web`，不要用 `HEAD`；
   - 或者先执行 `git -C plane rev-parse HEAD`，断言结果等于这个 SHA，再复制。
3. **把实际使用的提交 SHA 记进 [`frontend-changes.md`](../../frontend-changes.md)**（当前只写了 `02c19e1`，没有完整 SHA）。

## 处理结果（M0/P5）

1. 前端从 `02c19e1` 迁入（[P5 spec](../specs/P5-web-import.md) 2.3）。
2. 复制命令用完整的 SHA `02c19e1341d93141e8ad7b3278298adce208bafc`，复制前先用 `git -C "$PLANE" rev-parse <SHA>^{commit}` 确认提交存在；迁入的 13 个目录和 `patches/` 与这个提交中对应的 git 树对象完全相同。
3. [前端改动清单](../../frontend-changes.md)"一、代码来源"记下了完整的 SHA。

来源：[M0/P4 评审记录](../reviews/P4-plane-schema-review.md)。
