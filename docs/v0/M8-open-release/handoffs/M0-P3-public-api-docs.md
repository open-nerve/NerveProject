---
status: open
from: M0/P3
to: M8
created: 2026-09-22
---

# 对外接口文档

- **文档来源。** 对外接口文档以 `api/dist/openapi.yaml` 为准。
- **`redocly lint`。** M0/P3 没有使用它：它的推荐规则（`servers`、`security`、`summary` 等）对内部开发接口是噪音。M8 发布前评估启用哪些规则，并补上 `servers` 等对外文档需要的内容。
  - apitest 加载接口描述时已经忽略 `servers`，加入 `servers` 不会影响契约测试。
- **方法不对时返回 405 还是 404。**
  - v0 中，`/api/` 下"路径存在但方法不对"的请求返回 404 problem+json（P2 spec 2.7、P3 spec 2.6）。
  - 对外发布 v1 时决定是否改为 405。改为 405 需要平台的 `/api/` 兜底按方法探测路由表。

来源：[M0/P3 评审记录](../../M0-foundation/reviews/P3-api-contract-review.md)。
