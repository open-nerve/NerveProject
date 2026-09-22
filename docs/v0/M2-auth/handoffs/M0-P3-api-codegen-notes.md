---
status: open
from: M0/P3
to: M2
created: 2026-09-22
---

# M2 第一次写带参数、带请求体、需要认证的接口时的注意事项

## 1. 生成配置：在第一个可选、可空或 uuid 字段出现之前一次定下，所有模块都用同一套

- `format: uuid`：用 `output-options.type-mapping` 映射到 Go 1.27 标准库的 `uuid.UUID`。
  - 不映射时，生成代码会导入 `oapi-codegen/runtime/types`，从而间接引入 `github.com/google/uuid`。
  - depguard 看不到生成的文件，但 P3 新增的架构测试会拦住生产程序对 google/uuid 的传递依赖。
- `output-options.nullable-type: true`：生成 `nullable.Nullable[T]`，用于 PATCH 的"传 `null` 表示清空"（引入 `github.com/oapi-codegen/nullable`，写死版本）。
- `output-options.prefer-skip-optional-pointer`：决定可选字段生成 `*T` 还是 `T`。与上面两项一起定，一旦定下，以后再改会改动所有模块的类型。
- `github.com/oapi-codegen/runtime`：第一个带路径或查询参数的接口会让生成代码导入它（2026-09-22 的最新版是 v1.7.0），写死版本。加入依赖后检查 `server/go.mod` 仍是 `go 1.27`。

## 2. 错误映射

- **请求取值的校验。** 生成的代码只做类型绑定和 JSON 解码，不校验取值。需要决定：
  - 校验放在哪一层；
  - 参数校验错误的错误码；
  - `errors` 字段怎么填。
- **JSON 解码错误的 `detail` 会暴露 Go 的类型名。** 例如 `json: cannot unmarshal number into Go struct field IssueCreate.name of type string`。有请求体的接口出现时，把它映射为 `errors[]` 或通用的说明。
- **其他需要映射的错误：**
  - `http.MaxBytesError` → 413；
  - `context.Canceled`（客户端断开）不记为 500，也不写 error 日志。
- **领域错误到 problem 的方式。** 二选一，所有模块统一：
  - 在 OpenAPI 中声明带类型的 default 响应，handler 直接返回 `apigen.Problem`；
  - handler 返回 `error`，由 `ResponseErrorHandlerFunc` 按错误码映射为 403、404、409、422 等 problem。
- **错误出口的测试。** 第一个带参数或请求体的模块，要测试它有的每一个错误出口：参数绑定、请求体解码、handler 返回错误，并对 problem 做 `CheckResponse`。instance 没有参数，覆盖不到这些。
- **可选：请求校验。** 总体设计 8.1 也提到请求的契约校验，需要时在 apitest 中加 `CheckRequest`。

## 3. 安全声明的写法（已实际验证）

- **打包时会丢失的内容：**
  - 模块文件顶层的 `security` 和 `components.securitySchemes`，打包进 `api/dist/openapi.yaml` 时会被丢掉。
  - 操作级的 `security` 会保留，但它引用的 scheme 不会被带进来，而 kin-openapi 的 `doc.Validate` 不会报错。
- **写法：**
  - 每个操作单独声明 `security`。
  - `securitySchemes` 同时写在入口文件 `api/openapi.yaml` 和每个模块文件中（后者供 oapi-codegen 使用）。
  - 在 apitest 的写法检查中加一条：每个操作引用的 scheme 都存在。

## 4. 模块入口的演进

- 按路由挂载认证、限流、接口调用日志之后，`Register(mux, apiErrors)` 的参数会增多。
  - 在第二个模块出现之前，把平台提供的 HTTP 依赖（`APIErrors`、认证中间件等）合并成一个值传入。
  - `httpadapter.Register` 接收一个用例集合的结构体，而不是逐个传入指针。
- 生成代码的 `Middlewares` 按相反的顺序包装：列表中最后一个在最外层。
- 其他模块需要使用的能力（`Authorizer`、`Authenticator`）由模块导出访问方法，在 `bootstrap` 中接线。

## 5. 接口描述的组织规则

- **跨模块共用的接口类型放在 `common.yaml`。** 模块文件之间互相 `$ref` 会让一个模块的生成代码导入另一个模块的生成代码，违反架构规则 3 和 6。
- **一个路径只属于一个模块文件。** 入口文件每个路径只能 `$ref` 一个 path item。
- **模块文件名与 Go 模块目录名一致**（`api/modules/<m>.yaml` ↔ `server/internal/modules/<m>/`），Makefile 按文件名找生成配置。
- **沿用 P3 的模块模板**（P3 spec 2.7）：
  - `module.go` 提供 `New(依赖)` 和 `Register(…)`；
  - `adapter/http` 的包名为 `httpadapter`；
  - 三个错误出口都接到 `APIErrors`。
- **写法检查目前没有 map 型对象的例外。** `TestContractFollowsAuthoringRules` 要求每个 object 类型的组件 schema 都设 `additionalProperties: false`；一个 `type: object, additionalProperties: {type: string}` 这样的 map 型 schema 会被判为违规，目前没有例外（P3 final-fix-report C3）。M2 如果需要 map 型对象，扩展 apitest 的检查函数 `closedObject`（`server/internal/platform/httpserver/apitest/rules_test.go`），加一个例外分支和一个反例用例。

来源：[M0/P3 评审记录](../../M0-foundation/reviews/P3-api-contract-review.md)。
