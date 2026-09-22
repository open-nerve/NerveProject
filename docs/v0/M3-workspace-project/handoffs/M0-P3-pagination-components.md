---
status: open
from: M0/P3
to: M3
created: 2026-09-22
---

# 第一个列表接口加入分页的公共组件

M0/P3 的 `api/common.yaml` 只有 `Problem` 和 `FieldError`，分页组件要等到第一个列表接口出现时才加入（P3 spec 第 3 节差异 2）。M3 写工作区或项目的列表接口时：

- 在 `api/common.yaml` 中加入以下组件，写法按总体设计 3.4 和 P3 spec 2.2 的验证：
  - 参数 `Limit`、`Cursor`；
  - schema `NextCursor`（`type: [string, 'null']`）。
- 模块文件通过 `../common.yaml#/components/parameters/…` 和 `…/schemas/…` 引用它们。不要引用 `responses`：跨文件引用 response 会让生成的代码编译失败（P3 spec 2.4 的写法约定）。
- 生成之后，`apitest` 的组件名检查会确认没有出现 Redocly 冲突改名产生的 `Name-2`。

来源：[M0/P3 评审记录](../../M0-foundation/reviews/P3-api-contract-review.md)。
