# M0/P3 接口契约流水线与试点模块：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M0/P3 `api-contract` |
| 日期 | 2026-09-22 |
| 状态 | 已批准 |
| 上级文档 | [M0 设计文档](../M0-design.md) 第 1、2、3.1–3.3、3.7、3.8、4、6.1、6.3、8、12 节；[v0 总体设计](../../v0-design.md) 3.1、3.4、3.5、6.1–6.4、7.2、8 节 |
| 前置交接 | [P1-repo-toolchain-ci-split](../handoffs/P1-repo-toolchain-ci-split.md)、[P2-server-platform-p3-notes](../handoffs/P2-server-platform-p3-notes.md)（都在本 Phase 处理，见 2.11） |

## 1. 目标
把"接口描述 → 生成代码 → 模块 → 接线 → HTTP → TS 客户端"这条链路走通，并定下以后每个模块照着做的写法：
- 先验证 OpenAPI 3.1 在整条工具链上可用，定下版本（结论：**用 3.1**，见 2.2）；
- `api/` 的目录、Redocly 打包、打包结果 `api/dist/openapi.yaml`；
- 每个模块单独运行 oapi-codegen，`common.yaml` 的公共组件通过 import-mapping 生成到共享包 `platform/httpserver/apigen`；
- 生成代码的错误改为输出 problem+json；
- 试点模块 `instance`（`GET /api/v0/instance`），在 `bootstrap` 中接线；
- 用 kin-openapi 做契约测试：模块的响应、平台写出的全部 problem；
- `web/packages/api-client`：生成的 TS 类型和基于 openapi-fetch 的 `createClient`；
- `make gen` / `make gen-check`，以及按区域拆分的命令和持续集成（P1 的交接）。

## 2. 交付物

### 2.1 文件总览
路径都相对于仓库根目录。"生成"表示由命令生成并提交，不手改。

| 路径 | 内容 |
|---|---|
| `api/openapi.yaml` | 打包入口：`info`、`tags`，每个路径用 `$ref` 指向模块文件（2.4） |
| `api/common.yaml` | 公共组件：`Problem`、`FieldError` |
| `api/modules/instance.yaml` | instance 模块的接口 |
| `api/redocly.yaml` | Redocly 配置：入口、输出、关闭遥测 |
| `api/dist/openapi.yaml` | 打包结果（生成） |
| `package.json`、`pnpm-lock.yaml` | 根目录加入开发依赖 `@redocly/cli` |
| `server/internal/platform/httpserver/problem.go` | 新增平台错误码 `bad_request` |
| `server/internal/platform/httpserver/apierrors.go` | `APIErrors`：生成代码的错误处理函数（2.6） |
| `server/internal/platform/httpserver/apigen/` | `oapi-codegen.yaml`；`components.gen.go`（生成） |
| `server/internal/platform/httpserver/apitest/` | 契约校验工具，只被测试导入（2.8） |
| `server/internal/platform/httpserver/contract_test.go` | 平台写出的 problem 的契约测试 |
| `server/internal/modules/instance/` | 试点模块（2.7）；`adapter/http/gen/` 下是 `oapi-codegen.yaml` 和 `server.gen.go`（生成） |
| `server/internal/bootstrap/app.go`、`api_test.go` | 挂上 instance 模块；接线测试 |
| `server/internal/archtest/` | 规则 8 扩展到 `apitest`；`nerve` 程序的传递依赖测试（2.8） |
| `server/go.mod`、`server/go.sum` | 加入 kin-openapi |
| `web/packages/api-client/` | TS 客户端：`package.json`、`tsconfig.json`、`src/index.ts`、`src/schema.gen.ts`（生成）、`test/client.typecheck.ts` |
| `Makefile` | `gen`、`gen-check`、`lint` 及按区域拆分的命令（2.10） |
| `.github/workflows/ci.yml` | 每个任务只跑自己区域的检查；同仓 PR 不重复运行 |
| `README.md` | 新增"接口与代码生成"一节；第一次启动加上 `pnpm install` |
| `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-ci-split.md`、`P2-server-platform-p3-notes.md` | 改为 `done`，写明处理结果 |

新增的包依赖（箭头表示"导入"，均为非测试代码）：
```
bootstrap ──→ modules/instance（另有原来的 platform 包）
modules/instance ──→ instance/app, instance/adapter/buildinfo, instance/adapter/http, platform/httpserver
instance/adapter/http ──→ instance/app, instance/adapter/http/gen, platform/httpserver
instance/adapter/http/gen ──→ platform/httpserver/apigen
instance/adapter/buildinfo ──→ instance/domain, platform/buildinfo
instance/app ──→ instance/domain
platform/httpserver/apitest ──→ kin-openapi        （只被 _test.go 导入，archtest 规则 8）
```

### 2.2 OpenAPI 3.1 验证：结论与证据

**结论：使用 OpenAPI 3.1（`openapi: 3.1.0`）。** 整条链路（Redocly 打包 → oapi-codegen strict-server/std-http-server → kin-openapi 校验 → openapi-typescript/openapi-fetch）在 M0 设计第 4 节列出的六类写法上全部通过。验证中发现的三处 oapi-codegen 的不足都有等价的写法绕开，定为写法约定（2.4），不影响结论。P3 的 review 会复述这个结论。

**验证方法**：在仓库外的临时目录里写了一个假的 `sample` 模块，结构与 P3 的正式布局相同：`common.yaml`（`Problem`、`FieldError`、分页参数 `Limit` / `Cursor`、`NextCursor`）+ `modules/sample.yaml`（4 个操作：列表、创建、查询、PATCH）+ 入口文件，按正式流程打包、生成 Go 和 TS 代码，写一个 strict server 实现，再用 kin-openapi 校验真实的请求和响应。样例的关键写法见附录 A。

| 写法 | Redocly 2.53.3 打包 | oapi-codegen v2.8.0 生成 | kin-openapi v0.149.0 校验 | openapi-typescript 7.13.0 / openapi-fetch 0.17.0 |
|---|---|---|---|---|
| 可为空 `type: [string, 'null']`（含必填可为空、可为空的 `date` / `date-time`） | 原样保留 | `*string`、`*openapi_types.Date`、`*time.Time`；必填字段不带 `omitempty`，空值输出 `null` | 通过；`null` 放进不可为空的字段、数字放进 `string\|null` 都被拒绝 | `string \| null` |
| 枚举（字段、查询参数） | 保留 | 字符串类型 + 常量 + `Valid()`；查询参数是枚举类型 | 通过；枚举外的值被拒绝 | 字面量联合 `"bug" \| "feature" \| "chore"`；传入枚举外的值编译失败 |
| 可为空的枚举 | 保留 | 写成 `oneOf: [$ref, {type: 'null'}]` → `*Priority`，正确；写成 `enum: [..., null]` → 多出一个常量 `LessThannil = "<nil>"`，`Valid()` 也接受它 | 两种写法都通过，都拒绝枚举外的值 | 两种写法都得到 `... \| null` |
| `oneOf` + `discriminator` | 保留 | 联合类型，带 `As…` / `From…` / `Merge…`、`Discriminator()`、`ValueByDiscriminator()` | 通过；两个分支都不匹配时被拒绝 | 可区分联合，按 `type` 收窄 |
| 可为空的对象 `oneOf: [$ref, {type: 'null'}]` | 保留 | `*UserTarget` | 通过 | `UserTarget \| null` |
| 固定值 `const: user` | 保留 | 字段类型是 `interface{}`：不经过 `From…` 时输出 `null`，校验失败 | 正确拒绝了这个 `null` | `"user"` |
| 固定值写成 `type: string, enum: [user]` | 保留 | 枚举类型 `UserTargetType` | 通过 | `"user"` |
| `date` / `date-time` | 保留 | `openapi_types.Date` / `time.Time` | 通过；非法日期、用 `date-time` 充当 `date`、非法时间都被拒绝 | `string`（JSDoc 注明格式） |
| problem+json 错误响应（400、404、422、`default`） | 各模块相同的 `components.responses.Problem` 合并为一份 | 用 import-mapping 引用共享包的 `externalRef0.Problem`；`Content-Type: application/problem+json` | 默认就能解码 `application/problem+json`；`Content-Type` 不对、缺 `code` 都被拒绝 | 错误响应体的类型是 `Problem`，`error.code` 可用 |
| 游标分页（跨文件的参数 `Limit` / `Cursor`，`next_cursor: NextCursor`） | 参数和 schema 都收进 `components` | `*externalRef0.Limit`、`*externalRef0.Cursor`、`NextCursor *externalRef0.NextCursor` | 第一页 `next_cursor` 为字符串、最后一页为 `null`，都通过 | 查询参数有类型；`next_cursor: string \| null`，翻页循环能通过类型检查 |
| 跨文件引用 `components/responses` | 正常 | **编译失败**：`undefined: externalRef0.ProblemApplicationProblemPlusJSONResponse`（oapi-codegen 的已知限制：被引用的一方也要生成 strict server） | — | — |

其他证据：
- **生成代码的错误出口都能换成 problem+json**：路径参数不是 UUID、`limit=abc`（std-http 的 `ErrorHandlerFunc`）、请求体不是 JSON（strict 的 `RequestErrorHandlerFunc`）、handler 返回错误（`ResponseErrorHandlerFunc`），四种情况都输出 problem+json，并通过 kin-openapi 校验。strict server 先把响应编码到缓冲区再写响应头，所以编码失败时还能改写成 500。
- **kin-openapi 的校验是真的在起作用**：10 组正确的请求和响应全部通过；14 个故意改坏的响应全部被拒绝，并且每个都是因为预期的原因（上表列出的各项，以及缺少必填的可为空字段、多出未声明的字段）。
- **kin-openapi 的一个细节**：对 3.1 文档，`openapi3filter` 自动启用 JSON Schema 2020-12 校验器；但 schema 中含有 `$ref` 时，这个校验器无法单独编译，kin-openapi 会不报错地退回到自带的校验器。自带的校验器同样支持类型数组、`const`、`oneOf`，上面的反例大部分就是在这种 schema 上跑的。
- **TS 类型检查确实生效**：样例中四处错误的用法（枚举外的值、不存在的路径、给不可为空的字段传 `null`、把可为空的字段赋给 `string`）都用 `@ts-expect-error` 标出，`tsc` 通过就说明四处都确实报错（否则会报"未使用的 `@ts-expect-error`"）；去掉标注后四处都报错（三处 TS2322，不存在的路径为 TS2554）。
- **Redocly 的合并规则**：不同模块文件中同名、内容相同的组件合并为一份；同名但内容不同时，改名为 `Name-2`，**只给警告，退出码仍为 0**。这会悄悄改掉 TS 的类型名（Go 代码按模块文件生成，不受影响），所以由 `apitest` 的测试把关（2.8）。

**附带验证（为后续阶段准备，不影响结论）**：
- `format: uuid` 默认生成 `openapi_types.UUID`，它就是 `github.com/google/uuid`；用 `output-options.type-mapping` 可以映射到 Go 1.27 标准库的 `uuid.UUID`（它实现了 `TextMarshaler` / `TextUnmarshaler`），已验证能生成、能编译。
- PATCH 的"不传不改、传 `null` 清空"：`output-options.nullable-type: true` 生成 `nullable.Nullable[T]`（`github.com/oapi-codegen/nullable`），能区分"没传"和"传了 `null`"，已验证能编译。
- 生成的代码不按 schema 校验请求的取值（枚举、长度、范围）：查询参数的枚举只做类型转换，枚举外的值照样进入 handler。

**为什么验证样例不进仓库**：
- 样例是一个假接口。放进持续集成，就要在仓库里为它维护第二份描述、第二套生成的 Go 包和 TS 类型，它们都不属于产品，违背"不写用不上的代码"。
- 结论只在工具升级时才可能变化。四个工具的版本都已写死（2.3）；以后的真实接口会逐步用到这些写法，每个模块的契约测试（2.8）和 TS 类型检查会在持续集成中持续验证它们。
- 升级 oapi-codegen、kin-openapi、openapi-typescript 或 Redocly 时，按附录 A 的样例重跑一遍，并更新本节。

### 2.3 依赖版本（2026-09-22 通过 npm 和 Go 模块代理核实，写死）

| 工具 / 库 | 版本 | 核实 | 用途 | 位置 |
|---|---|---|---|---|
| `@redocly/cli` | 2.53.3 | `npm view @redocly/cli version`：最新稳定版，2026-09-17 发布，之后只有 snapshot 版本；包本身没有依赖 | 打包 `api/dist/openapi.yaml` | 根 `package.json` 的 `devDependencies` |
| `openapi-typescript` | 7.13.0 | npm `latest`；peer 依赖 `typescript ^5.x` | 生成 TS 类型 | `web/packages/api-client` 的 `devDependencies` |
| `openapi-fetch` | 0.17.0 | npm `latest` | TS 客户端 | `web/packages/api-client` 的 `dependencies` |
| `typescript` | 5.8.3 | M0 设计第 1 节 | 类型检查 | `web/packages/api-client` 的 `devDependencies` |
| `github.com/getkin/kin-openapi` | v0.149.0 | Go 代理 `@latest`（2026-08-28） | 契约测试 | `server/go.mod`；只有 `apitest` 导入 |
| oapi-codegen | v2.8.0 | P1 已写死在 `server/tools/go.mod` | 生成 Go 代码 | 不变 |

- Node 工具都写成精确版本；除了 Docker、Go、Node 之外不需要全局安装任何东西。
- `server/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`（加入 kin-openapi 并 `go mod tidy` 之后已核实）。
- instance 生成的代码只导入标准库和 `apigen`，所以 P3 不需要 `github.com/oapi-codegen/runtime`；第一个带参数的接口会引入它（最新版 v1.7.0，见第 7 节）。
- kin-openapi、`jsonschema` 等只出现在测试二进制中，生产的 `nerve` 不链接它们（已用 `go version -m` 核实）。oapi-codegen 自己依赖的 kin-openapi v0.142.0 在 `tools` 模块里，与主模块互不影响。

### 2.4 接口描述：`api/`

```
api/
  openapi.yaml           打包入口：info、tags、paths（每个路径指向模块文件）
  common.yaml            公共组件（只放 schemas 和 parameters）
  modules/instance.yaml  一个模块一个文件，本身是完整的 OpenAPI 3.1 文档
  redocly.yaml           Redocly 配置
  dist/openapi.yaml      打包结果（make gen-web 生成，提交）
```

**入口文件 `api/openapi.yaml`**：`info`（`title: Nerve API`、`version: v0`、AGPL-3.0-only 许可证，用 3.1 的 `license.identifier`）、`tags`（每个模块一个，带说明），`paths` 中每个路径写成 `$ref: 'modules/<模块>.yaml#/paths/~1api~1v0~1…'`。新增路径时在这里加一行，它也是整个接口的目录；漏加时 `TestRootListsEveryModulePath` 失败（2.8）。

**模块文件**（`api/modules/<模块>.yaml`）：
- 本身是完整的 OpenAPI 3.1 文档（`openapi`、`info`、`paths`、`components`），oapi-codegen 直接从它生成本模块的代码。
- 路径写全：`/api/v0/…`。`operationId` 用首字母小写的 camelCase（`^[a-z][A-Za-z0-9]*$`），`tags: [<模块>]`，tag 在入口文件的 `tags` 中声明。
- 每个操作都声明 `default` 响应：`$ref: '#/components/responses/Problem'`；这个 response 在本模块文件里定义，schema 引用 `../common.yaml#/components/schemas/Problem`。
- 以上几条由 `apitest` 的 `TestContractFollowsAuthoringRules` 在 `dist` 上检查（2.8）。

**写法约定**（来自 2.2 的验证）：

| 场景 | 这样写 | 不要这样写 | 原因 | 由谁检查 |
|---|---|---|---|---|
| 可为空的标量 | `type: [string, 'null']` | `nullable: true`（3.0 写法） | — | `TestContractFollowsAuthoringRules` |
| 可为空的枚举或对象 | `oneOf: [{$ref: '#/components/schemas/X'}, {type: 'null'}]` | `enum: [a, b, null]` | 后者让 Go 多出一个 `"<nil>"` 常量 | `TestContractFollowsAuthoringRules` |
| 固定值、`discriminator` 的属性 | `type: string` + 单值 `enum` | `const` | `const` 在 Go 中生成 `interface{}` | `TestContractFollowsAuthoringRules` |
| 引用公共组件 | `../common.yaml#/components/schemas/…`、`…/parameters/…` | `../common.yaml#/components/responses/…` | 跨文件引用 response 会编译失败，除非共享包也生成 strict server | 生成的 Go 代码编译失败 |
| 组件名 | PascalCase，在所有模块文件中唯一 | 两个模块用同一个名字表示不同的东西 | 打包时会被改名为 `Name-2`，`apitest` 的测试会失败（2.8） | `TestComponentNamesAreTypeNames` |
| 响应对象 | `additionalProperties: false` | — | 契约测试能发现多出来的字段 | `TestContractFollowsAuthoringRules`：所有 object 类型的组件 schema（请求 schema 同样适用），只由 `allOf`/`oneOf`/`anyOf` 组合、没有自己 `properties` 的除外 |

**`api/common.yaml`**：`Problem`（`status`、`code`、`title` 必填，`detail`、`errors` 可选，`additionalProperties: false`，与 `httpserver.Problem` 一致）和 `FieldError`（`field`、`message`）。分页的公共组件在第一个列表接口出现时加入（第 3 节差异 2）。

**`api/modules/instance.yaml`**：`GET /api/v0/instance`（`operationId: getInstance`，不需要登录）→ 200 `InstanceInfo`，`default` → `Problem`。`InstanceInfo` 四个字段都必填：`product`、`version`、`commit`（没有 VCS 信息时为 `unknown`）、`api_version`（`enum: [v0]`）。

**`api/redocly.yaml`**：`apis.nerve.root: openapi.yaml`、`output: dist/openapi.yaml`（相对于配置文件），`telemetry: off`（Redocly CLI 默认会上报匿名使用数据）。Makefile 另设 `REDOCLY_SUPPRESS_UPDATE_NOTICE=true`，不检查新版本，输出稳定。不使用 `redocly lint`：它的推荐规则要求 `servers`、`security`、`summary` 等，对 M0 的内部开发接口是噪音；文档结构由 kin-openapi 在契约测试中校验（`doc.Validate`）。对外文档的检查留到 M8。

**`api/dist/openapi.yaml`**：Redocly 打包结果，外部引用全部收进 `components`，组件按引用的先后排列。生成结果是确定的，提交到仓库。服务端的契约测试和 TS 类型都以它为准。

### 2.5 代码生成

**Go**（`make gen-go`，在 `server/` 下运行 `go tool -modfile=tools/go.mod oapi-codegen -config <配置> <描述>`）：

| 描述 | 配置 | 输出 |
|---|---|---|
| `api/common.yaml` | `server/internal/platform/httpserver/apigen/oapi-codegen.yaml` | 包 `apigen`：`components.gen.go`（只有类型） |
| `api/modules/<模块>.yaml` | `server/internal/modules/<模块>/adapter/http/gen/oapi-codegen.yaml` | 包 `gen`：`server.gen.go` |

- **一个模块一份配置，放在生成目录里**：配置是生成器的输入，和输出放在一起，模块自己的东西自己管。Makefile 遍历 `api/modules/*.yaml`，按文件名找到对应模块的配置；新模块忘了写配置时，生成直接失败。配置中的路径相对于 `server/`。
- **模块的配置**：`generate: models + std-http-server + strict-server`；`import-mapping: {../common.yaml: …/platform/httpserver/apigen}`（键必须和 `$ref` 中的写法完全一致）；`compatibility.always-prefix-enum-values: true`；`output-options.name-normalizer: ToCamelCaseWithInitialisms`。
- **`apigen` 的配置**：只生成 `models`，并设 `skip-prune: true`（`common.yaml` 没有 `paths`，默认的裁剪会删掉全部组件）；`always-prefix-enum-values`、`name-normalizer` 同上（`common.yaml` 目前没有枚举，加上这个选项生成结果不变）。
- **两个一开始就定下的选项**（以后再改会改掉已生成的名字，调用方的代码都要跟着改）：
  - `always-prefix-enum-values`：枚举常量总是带类型名前缀（`InstanceInfoAPIVersionV0`）。不开时，只有值冲突才加前缀，以后别的枚举出现同名的值，已有常量会被悄悄改名。
  - `ToCamelCaseWithInitialisms`：`APIVersion`、`ID`，符合 Go 的命名习惯。
- 生成前先删除旧的 `*.gen.go`，这样删掉某个模块的描述后，残留的生成文件会在 `make gen-check` 中暴露出来。
- **import-mapping 的表现符合预期**（M0 设计第 12 节的风险）：模块代码中 `Problem` 是 `externalRef0.Problem` 的别名，没有重复生成；不需要退回到"单个描述文件 + include-tags"的备选方案。
- 生成的文件头有 `Code generated … DO NOT EDIT.`，golangci-lint 自动跳过它们。

**TypeScript**（`make gen-web`）：先 `redocly bundle --config api/redocly.yaml` 生成 `api/dist/openapi.yaml`，再在 `web/packages/api-client` 中执行 `openapi-typescript ../../../api/dist/openapi.yaml --output src/schema.gen.ts`（包里的 `gen` 脚本）。不加任何选项：不生成 TS `enum`（那是运行时代码），类型只在编译期存在。

### 2.6 平台：生成代码的错误改为 problem+json（`platform/httpserver`）

oapi-codegen 生成的代码在三个地方处理错误，默认都用 `http.Error` 输出纯文本。平台提供统一的处理函数，每个模块在注册路由时接上：

```go
const CodeBadRequest = "bad_request" // 与 not_found、internal_error、not_ready 并列

// APIErrors answers the errors that generated code reports, as problem+json.
type APIErrors struct{ /* logger */ }

func NewAPIErrors(logger *slog.Logger) APIErrors

// 400 bad_request；detail 是 err.Error()（绑定或解码失败的原因）
func (APIErrors) BadRequest(w http.ResponseWriter, r *http.Request, err error)

// 记录 error 日志（request_id、method、path、error），返回不带 detail 的 500 internal_error；
// 响应已经开始时改为记录 warn 日志并中断连接（见下文）
func (e APIErrors) InternalError(w http.ResponseWriter, r *http.Request, err error)
```

| 生成代码中的出口 | 何时调用 | 接到 |
|---|---|---|
| `StdHTTPServerOptions.ErrorHandlerFunc` | 路径、查询、头部参数绑定失败（例如 UUID 格式不对、`limit=abc`） | `BadRequest` |
| `StrictHTTPServerOptions.RequestErrorHandlerFunc` | 请求体不是合法的 JSON | `BadRequest` |
| `StrictHTTPServerOptions.ResponseErrorHandlerFunc` | handler 返回错误，或响应编码、写出失败 | `InternalError` |

- `BadRequest` 的 `detail` 直接用生成代码给出的错误信息，例如 `Invalid format for parameter limit: …`，说明的是调用方自己传入的内容，不含服务端内部信息。
- `InternalError` 与异常恢复一致：错误只写进日志，响应中不带 `detail`，避免泄露内部主机名等信息。
- **响应已经开始时中断，不追加 problem**：strict server 先 `WriteHeader(200)` 再写出缓冲区，写出失败时同样调用 `ResponseErrorHandlerFunc`。这时 `InternalError` 按异常恢复的规则处理：平台的 `statusRecorder`（沿 `Unwrap` 找到，模块的中间件可能包了一层）显示状态码已经写出，就记录 warn 日志 `response failed after it started`（request_id、method、path、error），然后 `panic(http.ErrAbortHandler)` 中断连接。客户端看到的是不完整的响应，而不是 200 的响应体后面接着一个 problem。用 warn 而不是 error：最常见的原因是客户端已经断开，不是服务端的故障。
- **只新增 `bad_request`**：生成的代码只做类型绑定和 JSON 解码，不按 schema 校验取值，P3 中没有"参数校验错误（带 `errors`）"的来源。校验放在哪一层、错误码叫什么，随 M2 的错误码体系一起定（第 3 节差异 7、第 7 节）。

**`/api/` 下方法不对：仍然返回 404，P3 不改**（P2 交接第 4 条）：
- M0 设计 3.3 已经写明这个行为。
- ServeMux 只在"没有任何模式匹配"时才返回 405，而平台的 `/api/` 兜底不带方法，总能匹配。要返回 405，兜底就得逐个方法探测路由表，属于为罕见情况增加的机制；按生成的客户端调用时不会遇到方法不对。
- v0 允许不兼容的修改。发布 v1、M8 写对外文档时如需 405 再议。

### 2.7 试点模块 `instance`

目录（即以后所有模块的模板，M0 设计 3.2）：
```
modules/instance/
  domain/info.go                   Product、APIVersion 常量；Build、Info 值对象
  app/ports.go                     InfoSource 端口（由使用方声明）
  app/get_info.go                  GetInfo 用例
  adapter/buildinfo/source.go      InfoSource 的实现：读取 platform/buildinfo
  adapter/http/handler.go          实现生成的 StrictServerInterface；Register 挂载路由
  adapter/http/gen/                oapi-codegen.yaml、server.gen.go（生成）
  module.go                        New() *Module；(*Module).Register(mux, apiErrors)
```

```go
package domain
const Product = "Nerve"
const APIVersion = "v0" // 跟随产品主版本号（总体设计 3.1），也是每个接口路径的前缀
type Build struct{ Version, Commit string }
type Info struct{ Product, Version, Commit, APIVersion string }

package app
type InfoSource interface{ Build() domain.Build }
type GetInfo struct{ /* source */ }
func NewGetInfo(source InfoSource) *GetInfo
func (uc *GetInfo) Execute() domain.Info

package buildinfo // adapter/buildinfo
type Source struct{}
func (Source) Build() domain.Build // 来自 platform/buildinfo.Get()

package httpadapter // adapter/http
func Register(mux *http.ServeMux, getInfo *app.GetInfo, apiErrors httpserver.APIErrors)

package instance
type Module struct{ /* getInfo */ }
func New() *Module
func (m *Module) Register(mux *http.ServeMux, apiErrors httpserver.APIErrors)
```

- **用例不带 `ctx` 和 `error`**：`GetInfo` 不做 I/O，也不会失败；按"不写用不上的代码"，签名只保留实际需要的部分。有 I/O 的用例（M2 起）再带上它们。
- **`adapter/http` 的包名是 `httpadapter`**：包名 `http` 会遮住标准库 `net/http`，而 handler 两者都要用。导入时写明别名 `httpadapter`。
- **handler 只做类型转换**：`Execute()` 的结果转成 `gen.GetInstance200JSONResponse`，`api_version` 转成生成的枚举类型 `gen.InstanceInfoAPIVersion`。
- **`Register` 挂在根路由上**（P2 交接第 1 条）：`gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{BaseRouter: mux, ErrorHandlerFunc: …})` 把 `GET /api/v0/instance` 直接注册到 `httpserver.NewMux` 返回的路由上；它比平台的 `/api/` 更具体，其他 API 路径仍由兜底返回 404 problem+json。
- **接线**：`bootstrap.newApp` 在创建路由之后执行 `instance.New().Register(mux, httpserver.NewAPIErrors(logger))`。
- **`version` 与 `commit`**：来自 `platform/buildinfo`；`go run` 和测试中 `commit` 是 `unknown`，`go build` 出的程序带真实的提交号。

### 2.8 契约测试

**`platform/httpserver/apitest`**（只被测试导入）：
```go
type Contract struct{ /* 文档和路由 */ }
func Load(t testing.TB) *Contract                                           // 读取并校验 api/dist/openapi.yaml
func (c *Contract) CheckResponse(t testing.TB, req *http.Request, res *http.Response) // 按 req 找到操作，校验状态码、Content-Type、响应体
func (c *Contract) CheckSchema(t testing.TB, name string, body []byte)      // 按 components.schemas[name] 校验 JSON
```
- 文件位置由 `runtime.Caller` 得到（本包在仓库根目录下五层；测试不使用 `-trimpath`）。`go test` 的缓存会记录这次文件读取，`dist` 改了测试就会重跑。
- 用 kin-openapi 的 `routers/legacy` 找操作（不引入 gorilla/mux）；`openapi3filter.ValidateResponse` 设 `IncludeResponseStatus` 和 `MultiError`。
- **只按路径和方法找操作**：`Load` 在建路由之前清空文档、路径和操作上的 `servers`。文档有 `servers` 时，legacy 路由还要匹配请求的 scheme 和主机，测试用的主机都对不上，每个 `CheckResponse` 都会失败。测试：给 `dist` 的副本加上 `servers` 后，正确的 instance 响应仍然通过。
- `CheckResponse` 读完响应体后放回一个新的 reader，调用方还能再读。
- 它自己的测试：对真实的 `dist` 校验正确的响应通过，缺字段、枚举外的值、多出的字段、错误的 Content-Type、缺 `code` 的 problem、未声明的路径和方法都失败；`CheckSchema` 同理。
- **组件名检查**：`dist` 中所有组件名（`schemas`、`responses`、`parameters`、`requestBodies`、`headers`、`securitySchemes`、`examples`、`links`、`callbacks`）必须是 PascalCase 的类型名。它拦住 Redocly 在名字冲突时悄悄改出的 `Name-2`（2.2）；另有一个测试在手写的文档上确认九类组件都被列出。
- **写法约定检查**（`rules_test.go`）：`TestContractFollowsAuthoringRules` 用 2.4 的约定检查 `dist`：
  - 每个路径以 `/api/v0/` 开头；
  - 每个操作有首字母小写的 camelCase `operationId`、至少一个 tag，用到的 tag 都在顶层 `tags` 中声明；
  - 每个操作都有 `default` 响应，其 `application/problem+json` 的 schema 就是 `Problem` 组件（按 schema 的同一性比较：打包后是本地 `$ref`，加载器把它解析成组件本身）；
  - 所有 schema（组件和操作中内联的参数、请求体、响应、响应头，递归到 `properties`、`items`、`additionalProperties`、`not`、`allOf`/`oneOf`/`anyOf`）都不用 `nullable`、`const`，`enum` 中没有 `null`。`nullable: false`、`const: null` 解码后与没写一样，所以 `Load` 开启 kin-openapi 的 `IncludeOrigin`，同时检查每个 schema 在源文件中写出的键；
  - 所有 object 类型的组件 schema 都设 `additionalProperties: false`，只由 `allOf`/`oneOf`/`anyOf` 组合、没有自己 `properties` 的除外（`additionalProperties` 看不到子 schema 的属性，关上会拒绝所有实例）。

  检查函数接收文档作为参数：`rules_cases_test.go` 在一份手写的小文档上逐项改坏，证明每项检查都会报错，不在仓库里复制一份 `dist`。
- **入口文件列出所有模块路径**：`TestRootListsEveryModulePath` 读取 `api/modules/*.yaml`（位置的求法与 `dist` 相同），其中每个路径都必须出现在 `dist` 中。入口文件漏写的路径 Go 照样提供，TS 客户端和契约测试却没有。

**使用它的测试**：

| 测试 | 校验什么 |
|---|---|
| `platform/httpserver/contract_test.go` | 平台写出的每一种 problem：`/api/` 兜底 404、`/readyz` 503、panic 500、`BadRequest` 400、`InternalError` 500、带 `errors` 的 422，都符合 `Problem` schema（P2 交接第 3 条）。故意改掉 `Problem` 的一个 JSON 字段名，测试失败 |
| `modules/instance/adapter/http/handler_test.go` | `GET /api/v0/instance` 的响应符合契约，内容正确（M0 设计 3.8） |
| `bootstrap/api_test.go` | 整个程序（含三个中间件）：instance 返回 200、`version` 等于 `buildinfo.Get().Version`、带 `X-Request-Id`；`GET /api/v0/nope`、`POST /api/v0/instance` 返回 404 problem+json。不需要数据库 |

**archtest 规则 8** 改为"测试工具（`pgtest`、`apitest`）只能被测试导入"；规则表格补上相应的违规例子，以及 `gen` → `apigen` 这条合法的边。规则 8 只保证测试工具包本身不进生产程序，管不到别的途径：生成的代码（`embedded-spec: true`，或没有映射的 `format: uuid` 经 `oapi-codegen/runtime/types` 引入 `github.com/google/uuid`）或其他导入，而 depguard 不检查生成的文件。所以 archtest 另有 **`TestNerveBinaryLinksNoBannedModule`**：用 `packages.Load`（`NeedName | NeedImports | NeedDeps`，不含测试）读取 `./cmd/nerve` 的全部传递依赖，出现以 `github.com/getkin/kin-openapi`、`github.com/testcontainers/`、`github.com/google/uuid`、`github.com/docker/` 开头的包就失败，并给出一条导入链，例如 `cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3`。它与规则测试共用遍历源码目录的函数，新的导入会让缓存的结果失效。两者合起来保证：测试工具只在测试中使用，测试专用和禁用的模块不进 `nerve` 程序。

### 2.9 TS 客户端：`web/packages/api-client`

- **包名 `@nerve/api-client`**：它是新写的包，不来自 Plane。M1 会把 Plane 的包改名为 `@nerve/*`（总体设计 9.2），新包直接用最终的名字，免得 M1 再改一次，也不把 Plane 的名字用在我们自己的代码上。
- `package.json`：`private`，`type: module`，`exports: {".": "./src/index.ts"}`（M0 没有前端构建步骤，使用方直接引用 TS 源码：P6 的端到端测试、M2 起的前端）；脚本 `gen`、`check:types`（`tsc --noEmit`；与 Plane 的包和 turbo 的脚本名一致）。
- `tsconfig.json`：`strict`、`noUncheckedIndexedAccess`、`moduleResolution: bundler`、`verbatimModuleSyntax`、`noEmit`，`lib` 含 `DOM`（openapi-fetch 用到 `fetch`、`Request`、`Response` 的类型）。
- `src/index.ts`：导出 `createClient(options?: ClientOptions)`（`openapi-fetch` 的 `createClient<paths>`），以及类型 `paths`、`components`。不写转换层（总体设计 7.2）。
- `test/client.typecheck.ts`：只参与类型检查、不执行。用 `createClient` 调用 `GET /api/v0/instance`，断言 `data.api_version` 的类型是 `"v0"`、`error.code` 可用；用 `@ts-expect-error` 断言不存在的路径和 `POST /api/v0/instance` 无法编译。
- 这是第一个真正的 pnpm 工作区包；`pnpm-lock.yaml` 随之更新，持续集成用 `--frozen-lockfile` 安装。

### 2.10 Makefile 与持续集成

**命令**（`*-go` 只需要 Go，`*-web` 只需要 Node 并且先 `pnpm install`；不带后缀的依次执行两个区域，供本地使用）：

| 命令 | 作用 |
|---|---|
| `make gen` | `gen-go` + `gen-web` |
| `make gen-go` | 删除旧的 `*.gen.go`，生成 `apigen` 和每个模块的 `gen` |
| `make gen-web` | Redocly 打包 `api/dist/openapi.yaml`，生成 `src/schema.gen.ts` |
| `make gen-check` | `gen-check-go` + `gen-check-web` |
| `make gen-check-go` / `gen-check-web` | 先重新生成，再检查本区域的生成物已提交且没有差异：`git status --porcelain -- <生成物路径>` 必须为空，否则打印差异和提示"执行 make gen，并提交生成的文件"，退出码非 0 |
| `make lint` | `lint-go` + `lint-web` |
| `make lint-go` | golangci-lint（原来的 `make lint`） |
| `make lint-web` | `pnpm -r run check:types`（P5 加入 oxlint） |

- 生成物路径：Go 为 `server/internal/platform/httpserver/apigen`、`server/internal/modules/*/adapter/http/gen`；web 为 `api/dist`、`web/packages/api-client/src/schema.gen.ts`。用 `git status` 而不是 `git diff`，新模块没有提交的生成文件（未跟踪）也能发现。
- 本地执行 `make gen-check` 时，生成物有未提交的修改也算失败（M0 设计 6.1 的"检查生成物是否已提交"）。
- 兼容 GNU Make 3.81（`$(call …)`、`$(wildcard …)`，不用 4.x 的语法）。

**持续集成**：

| 任务 | 步骤 |
|---|---|
| `server` | checkout → setup-go → `make gen-check-go` → `make lint-go` → `make test` |
| `web` | checkout → setup-node → `corepack enable` → `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web` |

- 改了接口描述却没有重新生成：Go 生成物不一致时 `server` 任务失败，`dist` 或 TS 类型不一致时 `web` 任务失败（改动一般两边都有）。计划中在本地演示两者都会失败，并由控制者推送一个临时分支在持续集成上确认。
- **同仓 PR 不再重复运行**（P1 交接的"顺带决定"）：两个任务都加 `if: github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository`。同仓库分支的每个提交都由 push 事件跑过，PR 页面显示的就是这些结果；`pull_request` 事件只为来自 fork 的 PR 运行，因为 fork 的推送不会在本仓库触发 push 事件。代价是不再单独测试"合并到 main 之后"的结果；本项目的分支在本地合并、推送 main 时 push 事件会再跑一次。
- 已在本机模拟：只有 Go、没有 Node 的环境中 `server` 任务的三步都通过；只有 Node、没有 Go 的环境中 `web` 任务的三步都通过。

### 2.11 处理 P1、P2 的交接
- [P1-repo-toolchain-ci-split](../handoffs/P1-repo-toolchain-ci-split.md)：命令按区域拆分、每个持续集成任务只调用自己区域的命令、同仓 PR 不重复运行（2.10）。
- [P2-server-platform-p3-notes](../handoffs/P2-server-platform-p3-notes.md)：
  1. 生成的路由挂在根路由上（2.7），`bootstrap` 的测试验证兜底仍然有效（2.8）；
  2. 三个错误出口都输出 problem+json，新增 `bad_request`（2.6）；
  3. `Problem` 与 `api/common.yaml` 一致，由契约测试守住（2.8）；
  4. 方法不对仍然返回 404（2.6）；
  5. instance 的 `app` 只依赖本模块的 `domain`（archtest）。

两个 handoff 在计划的最后一个 Task 中改为 `done`，并写明处理结果。

## 3. 与上级设计的差异和补充（请控制者裁定）

| # | 上级设计 | P3 的做法 | 理由 |
|---|---|---|---|
| 1 | M0 2：`api/` 下有 `common.yaml`、`modules/`、`redocly.yaml`、`dist/` | 另有入口文件 `api/openapi.yaml`，列出每个路径并指向模块文件 | Redocly 的 `bundle` 需要一个根文档；合并多个文件的 `join` 命令仍标着实验性，而且会加入 `x-tagGroups`。入口文件同时是整个接口的目录 |
| 2 | M0 2：`common.yaml` 放"Problem、分页游标等" | P3 只放 `Problem`、`FieldError` | P3 没有列表接口，分页参数和 `next_cursor` 放进去只会生成没人用的 Go 和 TS 类型。第一个列表接口（M3）加入，写法已在 2.2 验证 |
| 3 | M0 3.2：`module.go` 提供 `New(deps) → *Module{Handler}` | `New() *Module` 加 `(*Module).Register(mux, apiErrors)` | 生成的路由必须注册在根路由上（P2 交接第 1 条），模块交出一个 Handler 就只能挂成子路由。instance 没有需要从外面传入的依赖；HTTP 相关的 `APIErrors` 在注册时传入 |
| 4 | M0 3.2：`adapter/http/handler.go` | 目录不变，包名为 `httpadapter` | 包名 `http` 会遮住标准库 `net/http`，handler 两者都要用 |
| 5 | M0 3.2：`adapter/buildinfo` "从 buildinfo 和配置中读取" | 只读 `platform/buildinfo`；`product`、`api_version` 是 `domain` 中的常量 | P3 返回的字段都不来自配置（P2 已决定不加 `app.name`）。M2 加入 `signup_enabled` 时再读配置 |
| 6 | M0 3.7：架构规则第 8 条只管 `pgtest` | 规则 8 改为"测试工具（`pgtest`、`apitest`）只能被测试导入"；新增 `platform/httpserver/apitest` | 契约校验被 httpserver、instance、bootstrap 三处测试共用；这条规则保证测试工具不进生产程序，kin-openapi 等模块另由 archtest 的传递依赖测试把关（2.8） |
| 7 | P2 交接第 2 条：至少新增 400 和"参数校验错误（带 `errors`）"两个平台错误码 | 只新增 `bad_request` | 生成的代码不按 schema 校验取值，P3 中没有校验错误的来源。校验放在哪一层、错误码叫什么，随 M2 的错误码体系一起定 |
| 8 | M0 6.1：`make gen`、`make gen-check`、`make lint` | 另有按区域拆分的 `gen-go` / `gen-web`、`gen-check-go` / `gen-check-web`、`lint-go` / `lint-web`；原来的三个命令保留，依次执行两个区域 | P1 交接：持续集成的 `server` 任务没有 Node |
| 9 | M0 6.3：`server` 任务 `make gen-check` → `make lint` → `make test` | `server`：`gen-check-go` → `lint-go` → `test`；`web`：`pnpm install` → `gen-check-web` → `lint-web` | 同上 |
| 10 | M0 6.3：触发条件"每次推送代码和每个 PR" | 触发条件不变；同仓库分支的 PR 在 `pull_request` 事件上跳过两个任务 | push 事件已经在同一个提交上跑过；来自 fork 的 PR 仍然运行（2.10） |
| 11 | M0 5.1：包名改为 `@nerve/*` 留到 M1 | 新包直接叫 `@nerve/api-client`，Plane 的包在 M1 之前仍是 `@plane/*` | 它不是来自 Plane 的包（2.9） |
| 12 | M0 8 P3：3.1 验证的结论写进 review | spec 2.2 先记录结论和证据，review 再复述；验证样例不进仓库 | 写法约定要以结论为前提；样例不进仓库的理由见 2.2 |
| 13 | M0 3.8：试点模块的 handler 测试做契约校验 | 另外校验平台写出的全部 problem，`bootstrap` 再从整个程序校验一次 | P2 交接第 3 条；也验证挂上模块之后兜底仍然有效 |
| 14 | — | oapi-codegen 选项 `always-prefix-enum-values`、`name-normalizer: ToCamelCaseWithInitialisms` | 以后再改会改掉已生成的类型和常量名，一开始就定下（2.5） |
| 15 | — | 接口描述的写法约定（2.4） | 绕开 2.2 发现的 oapi-codegen 的三处不足；组件名冲突由测试拦住 |

控制者裁定后，评审阶段把 M0 设计第 1 节（Redocly 版本）、第 2 节（`api/openapi.yaml`）、第 3.2 节（模块入口）、第 3.7 节（规则 8）、第 4 节（3.1 的结论、写法约定的链接）、第 6.1 节、第 6.3 节，以及总体设计 3.5 节的平台错误码（加入 `bad_request`）同步更新。

## 4. 验收标准
1. **OpenAPI 3.1 验证**：结论和证据记录在 2.2，P3 的 review 复述。
2. **契约与生成**：
   - `make gen` 之后 `git status` 没有变化；`make gen-check` 退出码为 0。
   - 故意修改 `api/modules/instance.yaml` 而不重新生成：`make gen-check-go` 和 `make gen-check-web` 都失败，并打印差异；恢复后通过。
   - 新增一个模块的描述和配置、生成但不提交：`make gen-check-go` 发现未跟踪的生成文件并失败。
3. **单元测试和契约测试**（`go test ./...` 中）：
   - `APIErrors`：400 和 500 的响应体、`Content-Type`；500 记录日志（错误、请求 ID、方法、路径）且响应中不带错误信息。
   - `apitest`：对真实的 `dist`，正确的响应通过，2.8 列出的各类错误响应失败；组件名都是 PascalCase。
   - 平台写出的 404、503、500（panic、`InternalError`）、400、422 都符合 `Problem` schema；故意改掉 `Problem` 的一个 JSON 字段名后测试失败。
   - instance：用例、适配器、handler 的测试通过，响应通过契约校验。
   - `bootstrap`：`GET /api/v0/instance` 返回 200 并通过契约校验；挂上模块后，`GET /api/v0/nope`、`POST /api/v0/instance` 仍然返回 404 problem+json（M0 设计第 8 节 P3 的验收）。
4. **架构测试通过**；故意在非测试代码中导入 `apitest`，规则 8 报错，删除后恢复。
5. **`make lint` 通过**：golangci-lint `0 issues.`，TS 类型检查通过；临时加入一个错误的调用（`POST /api/v0/instance`）后类型检查失败。
6. **`pnpm install --frozen-lockfile`** 在干净的克隆上成功。
7. **手工验证**（开发库已由 `make dev-db` 启动）：`make run` 后
   - `curl -i localhost:8080/api/v0/instance` → 200，`{"api_version":"v0","commit":"unknown","product":"Nerve","version":"0.1.0-dev"}`；
   - `curl -i localhost:8080/api/v0/nope` → 404 problem+json；
   - `curl -i -X POST localhost:8080/api/v0/instance` → 404 problem+json，`detail` 为 `no API endpoint for POST /api/v0/instance`。
8. `server/go.mod` 仍是 `go 1.27` 和 `toolchain go1.27.1`；生产的 `nerve` 不链接 kin-openapi（`go version -m`）。
9. **持续集成通过**；推送一个"修改描述但不重新生成"的临时提交，`server` 和 `web` 任务都在生成物检查这一步失败。

## 5. 不在 P3 范围内
- 前端应用迁入、turbo、oxlint（P5）；端到端测试（P6）。
- 认证、限流（M2）。
- 请求的取值校验和对应的错误码、领域错误到 problem 的映射（M2 错误码体系）。
- 分页的公共组件（第一个列表接口，M3）。
- 对外接口文档和 `redocly lint`（M8）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| oapi-codegen 对 3.1 的支持仍是"初步支持"，以后的写法可能碰到新的不足 | 2.4 的写法约定绕开已知问题；新写法第一次使用时，模块的契约测试和 TS 类型检查会暴露问题；升级工具时按附录 A 重跑验证 |
| 两个模块文件定义了同名但内容不同的组件，Redocly 只警告并改名 | `apitest` 的组件名测试让 `make test` 失败（2.8） |
| kin-openapi 对含 `$ref` 的 schema 会不报错地换用自带的校验器 | 2.2 已验证自带的校验器对我们用到的写法同样有效；`apitest` 的反例测试在真实文档上持续检查 |
| 本地没有执行 `pnpm install` 时，`make gen-web`、`make lint-web` 报"找不到命令" | README 的第一次启动步骤加入 `pnpm install`；Makefile 的说明写明需要 Node |
| `pnpm-lock.yaml` 中间接依赖的版本取决于安装时的 npm 状态 | 直接依赖都写死；持续集成用 `--frozen-lockfile`，以提交的锁文件为准 |
| 生成的 TS 文件用 4 个空格缩进，与 `.editorconfig` 的 2 个空格不同 | 生成文件不手改；P5 加入 oxfmt、oxlint 时排除 `*.gen.ts` 和 `api/dist`（第 7 节） |
| Redocly CLI 默认上报匿名使用数据、检查新版本 | `api/redocly.yaml` 设 `telemetry: off`；Makefile 设 `REDOCLY_SUPPRESS_UPDATE_NOTICE=true` |

## 7. 移交给后续阶段的事项（评审时建立 handoff）

| 交给 | 事项 |
|---|---|
| M2 | 请求取值的校验放在哪一层（生成的代码不校验枚举、长度等），以及"参数校验错误"的错误码和 `errors` 字段；领域错误在 `ResponseErrorHandlerFunc` 中映射为 problem |
| M2 | 错误出口的细节：请求体 JSON 解码失败时，`BadRequest` 的 `detail` 会带出 Go 的类型名（`json: cannot unmarshal … Go struct field IssueCreate.name …`），改为 `errors[]` 或通用的说明；`http.MaxBytesError` → 413；`context.Canceled` 不产生 500，也不记 ERROR 日志 |
| M2 | problem 只走一条路：用 `apigen.Problem` 的 typed `default` 响应，或者 handler 返回错误再统一映射，二选一 |
| M2 | `format: uuid`：用 `output-options.type-mapping` 映射到标准库 `uuid.UUID`，否则生成代码会用 `github.com/google/uuid`（2.2 已验证可行；漏掉时 archtest 的传递依赖测试失败） |
| M2 | PATCH 的"传 `null` 清空"：`output-options.nullable-type: true`，生成 `nullable.Nullable[T]`（引入 `github.com/oapi-codegen/nullable`） |
| M2 | 模块配置的模板选项一次定下、每个模块照抄：`nullable-type`、`prefer-skip-optional-pointer`、uuid 的 `type-mapping`（与上两行一起决定） |
| M2 | 第一个带参数的接口会让生成代码导入 `github.com/oapi-codegen/runtime`（最新版 v1.7.0），写死版本 |
| M2 | 认证的写法（已试过）：模块文件顶层的 `security`、`securitySchemes` 打包后被丢掉；操作上的 `security` 保留下来，但没有对应的 scheme，`doc.Validate` 也接受这个悬空的名字。约定：每个操作单独声明 `security`；`securitySchemes` 写在入口文件和每个模块文件里；在 `apitest` 中加一项检查 |
| M2 | 模块入口的扩展：第二个模块出现前，把平台的 HTTP 依赖合成一个值传入；`httpadapter.Register` 接收一个用例结构体；生成的 `Middlewares` 按相反的顺序包装（最后一个在最外层）；其他模块要用的能力由模块导出访问方法 |
| M2 | 接口描述的布局：多个模块共用的接口类型放在 `common.yaml`（模块文件之间互相 `$ref` 会违反 archtest 规则 3 和 6）；一个路径只属于一个模块文件；模块文件名与 Go 的模块目录名相同 |
| M2 | 第一个带参数或请求体的模块，为每个错误出口写测试；可选：`apitest` 增加 `CheckRequest` |
| M3 | 第一个列表接口把 `Limit`、`Cursor`（parameters）和 `NextCursor`（schema，`type: [string, 'null']`）加入 `api/common.yaml` |
| P5 | oxfmt、oxlint 排除 `web/packages/api-client/src/schema.gen.ts` 和 `api/dist/`；`make lint-web` 改为 turbo 驱动，保留 api-client 的类型检查（脚本已按 Plane 的习惯叫 `check:types`）；`typescript` 改用 `catalog:`；合并 Plane 根目录的 `package.json` 时保留 `@redocly/cli` |
| P6 | 端到端测试通过 `@nerve/api-client` 的 `createClient({ baseUrl })` 调用接口 |
| P6 | `e2e` 任务的 `if` 和 `needs`：同仓 PR 跳过的任务也报告为成功，`e2e` 经 `needs` 继承这一点；将来把检查设为必需之前，去掉跳过条件或加一个 `if: always()` 的汇总任务 |
| P6 | knip 忽略 `schema.gen.ts` 和 `test/**`；确认 Playwright 能转译真实路径在 `node_modules` 之外的工作区 TS 源码 |
| M8 | 对外接口文档以 `api/dist/openapi.yaml` 为准；届时评估 `redocly lint` 和 405；加入 `servers` 不影响契约测试（`apitest` 已忽略 `servers`，2.8） |

## 附录 A：3.1 验证样例的关键写法

验证样例（`common.yaml` + `modules/sample.yaml` + 入口文件）按 2.4 的布局组织。下面是其中决定结论的部分：

```yaml
# common.yaml（节选）
components:
  schemas:
    NextCursor:
      type: [string, 'null']
  parameters:
    Limit: {name: limit, in: query, schema: {type: integer, minimum: 1, maximum: 200, default: 50}}
    Cursor: {name: cursor, in: query, schema: {type: string}}

# modules/sample.yaml（节选）
paths:
  /api/v0/samples:
    get:        # 列表：limit、cursor（跨文件参数）、kind（枚举）→ 200 SampleList；400、default → Problem
    post:       # 创建：SampleCreate（oneOf 请求体）→ 201 Sample；422、default → Problem
  /api/v0/samples/{sample_id}:   # sample_id: {type: string, format: uuid}
    get:        # → 200 Sample；404、default → Problem
    patch:      # SampleUpdate（可为空字段，null 表示清空）→ 200 Sample；default → Problem
components:
  responses:
    Problem:
      description: Error
      content:
        application/problem+json:
          schema: {$ref: '../common.yaml#/components/schemas/Problem'}
  schemas:
    SampleKind: {type: string, enum: [bug, feature, chore]}
    Priority: {type: string, enum: [urgent, high, medium, low, none]}
    Severity: {type: [string, 'null'], enum: [critical, minor, null]}   # 内联写法，对照用
    UserTarget:
      type: object
      additionalProperties: false
      required: [type, user_id]
      properties:
        type: {type: string, enum: [user]}      # 对照组中写成 const: user
        user_id: {type: string, format: uuid}
    TeamTarget:
      type: object
      additionalProperties: false
      required: [type, team]
      properties:
        type: {type: string, enum: [team]}
        team: {type: string}
    Target:
      oneOf: [{$ref: '#/components/schemas/UserTarget'}, {$ref: '#/components/schemas/TeamTarget'}]
      discriminator:
        propertyName: type
        mapping: {user: '#/components/schemas/UserTarget', team: '#/components/schemas/TeamTarget'}
    Sample:
      type: object
      additionalProperties: false
      required: [id, name, description, kind, priority, severity, due_date, created_at, archived_at, target, owner]
      properties:
        id: {type: string, format: uuid}
        name: {type: string}
        description: {type: [string, 'null']}
        kind: {$ref: '#/components/schemas/SampleKind'}
        priority: {oneOf: [{$ref: '#/components/schemas/Priority'}, {type: 'null'}]}
        severity: {$ref: '#/components/schemas/Severity'}
        due_date: {type: [string, 'null'], format: date}
        created_at: {type: string, format: date-time}
        archived_at: {type: [string, 'null'], format: date-time}
        target: {$ref: '#/components/schemas/Target'}
        owner: {oneOf: [{$ref: '#/components/schemas/UserTarget'}, {type: 'null'}]}
    SampleList:
      type: object
      additionalProperties: false
      required: [data, next_cursor]
      properties:
        data: {type: array, items: {$ref: '#/components/schemas/Sample'}}
        next_cursor: {$ref: '../common.yaml#/components/schemas/NextCursor'}
```

重跑的步骤：
1. `redocly bundle` 打包；
2. oapi-codegen 分别生成 `common.yaml`（`models`、`skip-prune`）和 `sample.yaml`（`models`、`std-http-server`、`strict-server`、import-mapping）；
3. 写一个 strict server 实现，错误出口接 problem+json；用 kin-openapi 校验 2.2 中列出的正例和反例；
4. openapi-typescript 生成类型，用 openapi-fetch 写翻页、创建、PATCH 的调用，再加上 2.2 中的四处 `@ts-expect-error`，执行 `tsc`。
