---
status: done
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

## 处理结果（M2/P3a）

M2 的 PAT 列表（`listApiTokens`）是第一个列表接口，分页的公共组件随它加入（M2 设计 3.12）：

- **组件**（完成）：`api/common.yaml` 有参数 `Limit`（1–100，默认 50）、`Cursor` 和 schema `NextCursor`（`type: [string, 'null']`）。`api/modules/identity.yaml` 通过 `../common.yaml#/components/parameters/…` 和 `…/schemas/…` 引用它们，不引用 `responses`；`apitest` 的组件名检查通过，没有 `Name-2`。
- **游标的封套**在 `internal/shared`（`EncodeCursor`、`DecodeCursor`、`InvalidCursor`）；载荷由每个列表按自己的排序定义（M2 设计 3.12）。
- **页大小的规则**（1–100、默认 50，超出是 422 `validation_failed`）暂时在 `identity/domain`（`PageSize`），第二个分页列表出现时移到 `shared`（M2 设计 13.2 的 M3 一行）。

全部处理完，状态改为 `done`。M3 的列表接口直接引用这些组件。

来源：[M2/P3a spec](../../M2-auth/specs/P3a-account-api.md) 2.11、第 3 节第 8 条。
