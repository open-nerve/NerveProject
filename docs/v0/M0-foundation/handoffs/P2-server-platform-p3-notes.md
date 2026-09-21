---
status: open
from: M0/P2
to: M0/P3
created: 2026-09-22
---

# P3 接入接口契约和试点模块时的注意事项

1. **生成的路由挂在 `httpserver.NewMux` 返回的根路由上。**
   - 例如 `GET /api/v0/instance` 比平台的兜底模式 `/api/` 更具体，两者不冲突。
   - **不要**为 `/api/v0/` 单独建一个子路由再整体挂上去。那样会遮住平台的 `/api/` 兜底，不存在的接口路径就会得到 ServeMux 的纯文本 404，而不是 problem+json。
2. **oapi-codegen 的错误处理函数都要改成输出 problem+json。**
   - 需要改的是 strict server 的 `RequestErrorHandlerFunc`、`ResponseErrorHandlerFunc`，以及 `StdHTTPServerOptions.ErrorHandlerFunc`。它们默认用 `http.Error` 输出纯文本。
   - 因此需要新增平台错误码，至少包括请求格式错误（400）和参数校验错误（带 `errors` 字段）。
3. **`api/common.yaml` 的 `Problem` 与 `httpserver.Problem` 保持一致。**
   - 字段为 `status`、`code`、`title`、`detail`（可选）、`errors`（可选）。
   - `title` 固定为 HTTP 状态短语，具体说明放在 `detail`，程序判断用 `code`（总体设计 3.5）。
   - 用 kin-openapi 对平台写出的 404、500、503 响应体做契约校验，防止两边的定义以后不一致。
4. **`/api/` 下"路径存在但方法不对"的请求返回 404 problem+json，不是 405**（P2 spec 2.7）。如果 P3 认为需要 405，在 P3 的 spec 中决定。
5. **试点模块的 `app` 层只能依赖本模块的 `domain` 和 `internal/shared`**，由 archtest 检查（P2 修复后的规则 2）。
6. P1 的 [持续集成拆分](P1-repo-toolchain-ci-split.md) 也在 P3 处理。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
