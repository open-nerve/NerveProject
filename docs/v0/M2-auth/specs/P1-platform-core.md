# M2/P1 平台约定与第一个认证：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M2/P1 `platform-core` |
| 日期 | 2026-09-26 |
| 状态 | 已完成（[评审记录](../reviews/P1-platform-core-review.md)） |
| 上级文档 | [M2 设计文档](../M2-design.md) 第 3.3–3.14、3.20、4.1–4.3、4.5–4.7、5.1–5.4、6.1–6.6、8.7、9.1–9.5、12（P1）、13.1、16 节；[v0 总体设计](../../v0-design.md) 3.1、3.5、4、5.5、5.6、6.2–6.4、6.8、8.2 节；[M0 设计](../../M0-foundation/M0-design.md) 3.1、3.3、3.5 节 |
| 前置交接 | [M0-P1-sqlc-cgo](../handoffs/M0-P1-sqlc-cgo.md)、[M0-P2-platform-notes](../handoffs/M0-P2-platform-notes.md)、[M0-P3-api-codegen-notes](../handoffs/M0-P3-api-codegen-notes.md)、[M0-P4-schema-conventions](../handoffs/M0-P4-schema-conventions.md)、[M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md)（本 Phase 关闭或部分关闭的条目见第 7 节） |
| 计划 | [P1 plan](../plans/P1-platform-core.md) |

本 spec 只写 M2 设计交给 P1 决定的东西：名字、签名、SQL、配置项、测试名，以及原型证明了什么。规则本身以 M2 设计为准，这里引用节号，不重述。

## 1. 目标

按 M2 设计 12 节 P1 的目标：任何调用方都能注册，并用注册得到的令牌访问 `GET /me`；不带令牌访问非公开操作一律 401；不合契约的请求体一律 400；后续 M 照做的平台约定全部定下。具体是：

- 平台：共享内核 `internal/shared`、时钟、`TxManager`（`COMMIT` 不受请求期限的取消）、`timestamptz` 按 UTC 读出；`httpserver` 的 `Router`、`API` 值、按路由的中间件链（请求元信息 → 期限 → 请求体上限 → 默认拒绝的认证 → 请求体结构）、`ProblemError` 的"一条路"映射；
- 契约：`FieldError.code`、`security` 与 `x-problem-codes` 的写法和 `apitest` 的核对、`CheckRequest`、oapi-codegen 的模块模板、请求体结构表的生成器 `bodyshapegen`；
- 数据：表结构约定，迁移 `00001`–`00003`，sqlc 接入与 `TestSQLCSchemaScope`；
- `identity`：`register`、`getMe`、认证用例与 `authn` 适配器、argon2id（并发上限和等待上限）、密码规则与常见密码名单、Ed25519 JWT 和刷新令牌的 MAC；
- `bootstrap` 的接线和四个整程序测试；端到端的 fixture、S1 的迁移断言、A1 和 A2 的接口版本；
- 3.20 中标 P1 的文档同步、8.7 中标 P1 的 README 内容、交接的处理记录。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由命令生成并提交，不手改。"Task"是 plan 中负责它的任务；两个 Task 号表示先写过渡版本、后一个 Task 写成最终版本。

| 路径 | 内容 | Task |
|---|---|---|
| `Makefile` | `make test`、`make lint-go` 进入工具模块；`gen-go` 运行 `bodyshapegen` 和 sqlc | 1、6 |
| `server/internal/platform/httpserver/bodyshape/` | 请求体结构检查：`Table`、`Check`、`Middleware`（只用标准库） | 1 |
| `server/tools/bodyshapegen/` | 由模块的接口描述生成结构表；`testdata/` 是黄金文件 | 1 |
| `server/tools/go.mod`、`go.sum` | sqlc v1.31.1 作为 tool；生成器的直接依赖 | 1、6 |
| `server/internal/modules/*/adapter/http/gen/bodyshape.gen.go` | 生成 | 1、10 |
| `server/internal/shared/` | `Error`、`Kind`、字段码、`Actor`、`TxManager` | 2 |
| `server/internal/platform/clock/`、`clock/clocktest/` | `System`；测试用的 `Fixed`（截到微秒） | 2 |
| `server/internal/platform/postgres/tx.go`、`pool.go` | `TxManager`、`DB(ctx, pool)`；连接池按 UTC 读 `timestamptz` | 2 |
| `server/internal/archtest/rules_test.go`、`rules_cases_test.go` | 平台不导入 `shared`；生成代码规则推广到 `adapter/*/gen`；`clocktest` 只给测试 | 2 |
| `server/internal/platform/config/`、`server/configs/` | 6.5 中 P1 的配置项、校验、空值报错、`LogValue` | 3 |
| `server/internal/platform/httpserver/{problem,middleware,routes,server,apierrors,api}.go` | 平台码、`RequestID`、`Router`、`addr_file`、`ProblemError` 映射、`API` | 4 |
| `api/common.yaml` | `FieldError.code`；`Problem.code` 的说明 | 4 |
| `server/internal/platform/httpserver/apigen/components.gen.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts` | 生成 | 4、5、10 |
| `server/internal/platform/httpserver/apitest/{apitest,problems}.go` | `CheckRequest`、问题码核对、`Main` | 5 |
| `server/internal/platform/httpserver/apitest/rules_test.go` 等 | `security`、`x-problem-codes` 的写法检查 | 5 |
| `api/openapi.yaml` | 顶层 `x-problem-codes`、`securitySchemes.bearer`；P1 的路径 | 5、10 |
| `api/modules/instance.yaml`、`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`、`main_test.go` | `security: []`、`x-problem-codes: []`；模块模板；`apitest.Main` | 5 |
| `server/migrations/sql/0000{1,2,3}_*.sql`、`embed.go`、`schema_test.go` | 三张表；约束名和反例测试 | 6 |
| `server/sqlc.yaml`、`server/internal/modules/identity/adapter/postgres/queries/*.sql` | sqlc 配置和查询 | 6 |
| `server/internal/modules/identity/adapter/postgres/gen/` | 生成 | 6 |
| `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go` | `TestSQLCSchemaScope` | 6 |
| `e2e/stories/smoke/s1-server-ready.spec.ts`、`e2e/tsconfig.json` | S1 的迁移断言 | 6 |
| `server/internal/modules/identity/domain/` | 邮箱、密码规则、名单、刷新令牌的布局、`NewAccount`、模块错误 | 7 |
| `server/internal/modules/identity/domain/common_passwords.txt` | 生成（`tools/password-blocklist/build.mjs`） | 7 |
| `tools/password-blocklist/build.mjs`、`knip.jsonc` | 名单的生成脚本；knip 的入口 | 7 |
| `server/internal/modules/identity/app/` | 端口、`Register`、`GetMe`、`Authenticate` | 8 |
| `server/internal/modules/identity/adapter/{argon2,signing,postgres,authn}/` | 适配器 | 9 |
| `server/go.mod`、`go.sum` | jwt v5.3.1、x/crypto v0.57.0、nullable v1.2.0 | 9、10 |
| `api/modules/identity.yaml`、`identity/adapter/http/`（含 `gen/oapi-codegen.yaml`） | `register`、`getMe` | 10 |
| `server/internal/modules/identity/module.go`、`instance/module.go`、`instance/adapter/http/handler.go` | 模块入口：`PublicOperations`、`Register(router, api)` | 11（instance 另有 4 的过渡版本） |
| `server/internal/platform/httpserver/apitest/operations.go` | 整程序测试用的 `Operations`、`Target`、`BodyCases` | 11 |
| `server/internal/bootstrap/` | 接线、签名密钥文件、环境提醒、致命错误的结构化日志；四个整程序测试 | 2、4、6、11 |
| `e2e/fixtures/`、`e2e/global-setup.ts`、`e2e/stories/identity/` | fixture 扩展；A1、A2 的接口版本 | 12 |
| `docs/…`、`README.md` | 3.20 的 P1 各行、8.7 的 P1 内容、交接的处理记录 | 13 |

### 2.2 依赖版本（2026-09-26 在原型中核实，写死）

| 依赖 | 版本 | 模块 | 引入 |
|---|---|---|---|
| `github.com/sqlc-dev/sqlc`（tool） | v1.31.1 | `server/tools` | Task 6：`go -C server/tools get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1`，再 `go mod tidy` |
| `github.com/getkin/kin-openapi`、`github.com/oapi-codegen/oapi-codegen/v2`、`go.yaml.in/yaml/v3` | v0.142.0、v2.8.0、v3.0.4（原有，改为直接依赖） | `server/tools` | Task 1：`go -C server/tools mod tidy` |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | `server` | Task 9 |
| `golang.org/x/crypto` | v0.57.0（间接把 `golang.org/x/text` 从 v0.41.0 抬到 v0.42.0） | `server` | Task 9 |
| `github.com/oapi-codegen/nullable` | v1.2.0 | `server` | Task 10：`identity` 的生成代码第一个用到它（M2 设计 6.6"随第一个用到它的生成代码加入"） |

- 每次 `go get`、`go mod tidy` 之后核对 `server/go.mod` 和 `server/tools/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`（原型中三次核对都未变）。
- sqlc 加入工具模块后，oapi-codegen 的依赖版本没有变化（kin-openapi、yaml、x/text、x/tools、x/mod 都与 M0 相同），`instance` 的 `server.gen.go` 逐字节不变。M2 设计 §16 的这条风险在原型中已排除。
- 不加 `github.com/oapi-codegen/runtime`：P1 没有带参数的操作（M2 设计 6.6，随 P3 加入）。

### 2.3 工具模块进持续集成（M2 设计 3.11 核验 F2）

- `make test` 在 `cd server && go test -count=1 ./...` 之后另跑 `go -C server/tools test -count=1 ./...`；`make lint-go` 另在 `server/tools` 跑 `golangci-lint run --config ../.golangci.yml ./...`。持续集成的 server 任务调用的就是这两个目标，`ci.yml` 不改。
- 原型核对了覆盖面：在 `server/tools/bodyshapegen` 里放一个没用到的函数，`make lint-go` 报 `unused`；删掉后 0 issues。

### 2.4 请求体结构检查：`bodyshape` 与 `bodyshapegen`（M2 设计 3.11）

**`platform/httpserver/bodyshape`**（只用标准库和标准库的 `uuid`）：

- 类型：`Type`（位集合 `Null`、`Boolean`、`Integer`、`Number`、`String`、`Array`、`Object`，`Any` 为 0）；`Format`（`FormatNone`、`FormatTime`、`FormatUUID`）；`Node{Types, Format, Props map[string]int, Required []string, Extra, Items}`，`Extra`、`Items` 是节点下标或 `Closed`（-1）、`Open`（-2）；`Table{Nodes []Node, Roots map[string]int}`，`Roots` 的键是路由模式。
- `(*Table).Check(pattern string, body []byte) error`：对任何请求体都有答复。路由没有结构表、请求体为空或只有 JSON 空白时返回 nil（没有要查的，由生成代码的解码报错）；不是一个合法的 JSON 文档时返回 `ErrNotJSON`；否则一次遍历收集全部问题，按字段、再按码排序，有问题时返回 `*Error`。字段路径形如 `tags[1].name`，请求体本身是空路径。字段码只有 `required`、`invalid_format`、`not_allowed` 三个。
  - 遍历原始字节（`map[string]json.RawMessage` / `[]json.RawMessage`），不用 `UseNumber` 的通用解码：格式检查器拿到的是原样的字节，与生成代码解码时看到的相同。`Integer` 判定为 `strconv.ParseInt`（64 位）能解析，`Number` 判定为 `strconv.ParseFloat`（64 位）能解析；两者都解析不了的数（如 `1e400`）不属于任何类型，对有类型的节点是 `invalid_format`。
  - 格式检查把原始值 `json.Unmarshal` 进 `time.Time` 或 `uuid.UUID`，与生成代码解码用的是同一条路（M2 设计 3.11）。
- `Error{Fields []FieldError}` 满足 `httpserver.ProblemError`：400 `bad_request`，`detail` 固定为 "The request body does not match the API description."；`FieldError` 满足字段接口，`message` 分别是 "is required"、"is not a property of this request"、"has the wrong type or format"。
- `Middleware(t *Table, onError func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler`：路由没有结构表或没有请求体时放行；读出请求体后放回，交给 `Check`：`Check` 返回的错误（`ErrNotJSON`、`*Error`）交给 `onError`，返回 nil（包括空白的请求体）就放行；读取失败（包括超过请求体上限）原样交给 `onError`。`API` 把 `onError` 接到 `APIErrors.BodyError`。

**`server/tools/bodyshapegen`**：`go -C server/tools run ./bodyshapegen -config <模块的 oapi-codegen.yaml> -out <gen>/bodyshape.gen.go <模块的接口描述>`。

- 读模块的 `oapi-codegen.yaml`，与 oapi-codegen 相同地合并 `type-mapping`（`codegen.DefaultTypeMapping.Merge`）；用 `util.LoadSwagger` 读描述。
- 输出 `func BodyShapes() *bodyshape.Table`，文件头 `// Code generated by bodyshapegen from <描述文件名>. DO NOT EDIT.`，gofmt 过。路径、方法、属性都排序后编号，两次生成逐字节相同（`TestGenerateIsDeterministic` 连续生成 10 次）。
- 支持的写法：`type`（含 `[X, 'null']`）、`properties`、`required`、`additionalProperties`（`false`、`true`、schema、省略）、`items`、`enum`（取值由领域检查）、`format`，以及 `anyOf: [X, {type: 'null'}]`。
- **遇到不支持的写法就失败并说明**（M2 设计 §16），错误带路径前缀，例如 `POST /api/v0/x: due.at[]: …`：
  - `x-go-type`、`x-go-type-import`："bypasses the type-mapping, so the generated Go type is unknown"；
  - `allOf`、`oneOf`、`not`："allOf, oneOf and not are not supported"；
  - OpenAPI 3.0 的 `nullable`："nullable is OpenAPI 3.0; write type: [X, 'null']"；
  - 其他形式的 `anyOf`："anyOf is supported only as [X, {type: 'null'}]"；
  - 生成为非 `string` 的 Go 类型、又没有检查器的格式：`format "<f>" is generated as <T>, which has no bodyshape checker`（`TestGenerateFollowsTheTypeMapping` 用默认映射核对 `email` 会因此失败）。
  - `integer` 生成为 `int`、`int64` 以外、`number` 生成为 `float64` 以外的 Go 类型：`<integer|number>[ format "<f>"] is generated as <T>, whose range bodyshape does not check`；`format` 不在映射里：`… is not in the type-mapping; the default Go type need not hold the range it names`（`TestGenerateRejectsTheDefaultNumberMapping` 核对默认映射的 `float32`）。
- 生成器在工具模块；生成的文件只导入 `bodyshape`。

### 2.5 共享内核、时钟与事务（M2 设计 3.3、3.6、3.13）

**`internal/shared`**（只导入标准库和标准库的 `uuid`）：

- `type Kind int`：`KindInvalid`（422）、`KindBadRequest`（400）、`KindUnauthenticated`（401）、`KindForbidden`（403）、`KindNotFound`（404）、`KindConflict`（409）、`KindRateLimited`（429）、`KindUnavailable`（503）；状态码写成字面量（`shared` 不导入 `net/http`）。
- 平台码常量 `CodeValidationFailed`、`CodeUnauthorized`、`CodeServerBusy`；字段码常量是 3.11 的封闭集合全部十个（`FieldRequired` … `FieldContainsURL`），与 `api/common.yaml` 的枚举一致：`FieldCodes()` 列出全部十个（`TestFieldCodesListEveryFieldConstant` 读源码核对没有漏掉的 `Field*` 常量），bootstrap 的 `TestFieldCodesAreTheContractsEnum` 用 `apitest` 的 `Enum` 读 `api/dist/openapi.yaml`，两个方向核对。
- `FieldError{Field, Code, Message}`；`Error{Kind, Code, Detail, Fields, RetryDelay}`，方法 `Error`、`Is`（按 Kind 和 Code 匹配）、`ProblemStatus`、`ProblemCode`、`ProblemFields`、`RetryAfter`。构造函数 `NewError(kind, code, detail)`、`Invalid(fields...)`（detail "The request has invalid values."）、`Unauthenticated()`（"Authentication is required."）、`ServerBusy(retry)`（"The server is busy; retry shortly."）。
- `Actor{UserID, SessionID}`、`WithActor`、`RequireActor`（取不到时是 `Unauthenticated()`）。
- `TxManager` 接口：`WithinTx(ctx, fn func(ctx) error) error`。

**`platform/clock`**：`System{}`（`time.Now()` 转为 UTC、截到微秒）；`clocktest.At(t) *Fixed`（`Advance(d)`，同样是 UTC、微秒），满足同一个时钟契约（`TestSystemFollowsTheClockContract`、`TestFixedFollowsTheClockContract`）。各模块在自己的 `app` 声明 `Clock` 端口（M2 设计 3.3），`clock` 包不声明接口。

**`platform/postgres`**：

- `TxManager`（`NewTxManager(pool, commitTimeout)`）按结构满足 `shared.TxManager`；`bootstrap` 写编译期断言 `var _ shared.TxManager = (*postgres.TxManager)(nil)`。事务放在 `context` 里，嵌套的 `WithinTx` 加入外层事务；`fn` 出错或 panic 都回滚。
- **`COMMIT`、`ROLLBACK` 在 `context.WithoutCancel(ctx)` 下执行，另有 `database.commit_timeout` 的期限**（M2 设计 3.6 控制者复核 R7）。测试 `TestWithinTxCommitsAfterTheContextIsCancelled`（语句做完后取消 `context`，仍然提交）、`TestWithinTxRollsBackAfterAFailedStatementAndCancel`（语句失败后取消，回滚完成，连接可以再用）。`ROLLBACK` 本身失败时，返回的错误包住这次失败，只保留 `fn` 错误的文字、不保留它的身份，由 `Write` 答 500（`TestWithinTxReportsAFailedRollback`）。
- `DB(ctx, pool) Querier`：有事务时返回事务，否则返回连接池；`Querier` 与 sqlc 生成的 `DBTX` 方法相同，仓储把它交给生成的查询。
- 连接池的 `AfterConnect` 把 `timestamptz` 的扫描时区设为 UTC（`TestPoolScansTimestamptzInUTC`）。

### 2.6 架构测试（M2 设计 3.3、3.14；M0-P2 交接 6）

- 规则 4 改为"platform does not import modules, bootstrap or internal/shared"。
- 规则 6 改为"generated code is imported only by its own adapter"：`internal/modules/<m>/adapter/<a>/gen` 只能由 `adapter/<a>` 及其子包导入，覆盖 oapi-codegen 的 `http/gen` 和 sqlc 的 `postgres/gen`。
- 测试工具规则加上 `clock/clocktest`。
- 每条改动都有人造导入边的正反例（`rules_cases_test.go`）。
- `shared` 只导入标准库的规则（`sharedKernelIsPure`）是 M0 加固时加的，不改。

### 2.7 配置（M2 设计 6.5 中 P1 的部分；M0-P2 交接 4、8）

新配置项（`server/configs/config.yaml` 写默认值）：

| 键 | 默认值 | 校验 |
|---|---|---|
| `server.request_timeout` | `15s` | 大于 0，且小于 `server.write_timeout` |
| `server.max_body_bytes` | `1048576` | 至少 1 |
| `server.addr_file` | `""` | — |
| `database.commit_timeout` | `2s` | 大于 0 |
| `auth.signup_enabled` | `false`（`config.dev.yaml`、`config.test.yaml` 覆盖为 `true`） | — |
| `auth.access_token_ttl` | `15m` | 大于 0 |
| `auth.session_ttl` | `720h` | 大于 0，且大于 `access_token_ttl` |
| `auth.jwt.private_key_file` | `""` | prod 必填 |
| `auth.password.argon2_memory_kib` | `19456`（test 为 `64`） | 至少 8 × 并行度（x/crypto 在更低时悄悄抬高） |
| `auth.password.argon2_iterations` | `2`（test 为 `1`） | 至少 1（x/crypto 在 0 时 panic） |
| `auth.password.argon2_parallelism` | `1` | 至少 1 |
| `auth.password.max_concurrent_hashes` | `4` | 至少 1 |
| `auth.password.max_wait` | `2s` | 大于 0 |

- `LogValue` 对 `*_file` 只记是否设置：`addr_file_set`、`private_key_file_set`（`TestLogValueHidesFilePaths`）。
- 环境变量给非字符串的键传空值时报错 "must not be empty"，不再悄悄当作 `false` 或 0（`emptyValueHook`，M0-P2 交接 8 第一项）。
- 私钥文件在 `bootstrap` 启动时读取、在 `identity.New` 中解析；`config.validate` 只检查 prod 是否设置。错误只写配置项的名字，不写路径（`auth.jwt.private_key_file: no such file or directory`）。
- "prod 在没有任何覆盖时默认关闭注册"由 `server/configs/embed_test.go` 的 `TestBuiltInProfiles` 证明（prod 一行 `signup: false`）；`TestProdRequiresDatabaseURLAndSigningKey` 证明 prod 缺 `database.url` 和 `auth.jwt.private_key_file` 时两个都报。

### 2.8 `httpserver`：Router、API 值、错误映射、默认拒绝（M2 设计 3.6、3.11）

**平台码**：`CodeBadRequest`、`CodeUnauthorized`、`CodeNotFound`、`CodePayloadTooLarge`、`CodeInternal`、`CodeNotReady`。`validation_failed`、`server_busy` 只由领域错误经 `ProblemError` 带出。`FieldError` 加上 `Code`（`json:"code"`）。

**`RequestID(ctx)`** 导出（M0-P2 交接 3）。`/healthz`、`/readyz` 的访问日志降为 DEBUG（交接 8 第三项，`TestAccessLogOfProbesIsDebug`）。

**`Router`**：`NewRouter(logger, checks...)` 替代 `NewMux`；方法 `HandleFunc`、`Handle`（都记下模式）、`ServeHTTP`、`Patterns()`（按注册顺序的副本）。它满足生成代码的 `ServeMux` 接口。

**`server.addr_file`**：`ListenAndServe` 监听成功后，先写临时文件再改名，把实际地址写进这个文件；写不了时关闭监听并返回 `server.addr_file: write: …`（不带路径）。

**`ProblemError` 与 `APIErrors`**：

```go
type ProblemError interface {
	error
	ProblemStatus() int
	ProblemCode() string
}
// 可选：ProblemFields() []error（每个元素有 ProblemField()、ProblemCode()），RetryAfter() time.Duration
```

- `NewAPIErrors(logger)`；`BadRequest`（参数绑定，照旧）；`BodyError`（请求体读取、解析、解码）：`ProblemError` 和 `*http.MaxBytesError` 交给 `Write`，其余是 400 `bad_request`，`detail` 固定为 "The request body could not be decoded."，解码器的原话只进 DEBUG 日志（它带 Go 的类型名）。
- `Write` 是"一条路"：`ProblemError` → 它自己的状态、码、`detail`、字段、`Retry-After`（向上取整到秒），401 还带 `WWW-Authenticate: Bearer`（已经设置的挑战保留，M2 设计 3.6）；`*http.MaxBytesError` → 413 `payload_too_large`（"The request body exceeds N bytes."）；请求的 `context` 已取消而错误是 `context.Canceled` → 记 DEBUG "client went away"、写 499，不算 500；其余 → 记 ERROR "API handler failed"、500 `internal_error` 不带 `detail`。响应已开始时记 WARN 并 `panic(http.ErrAbortHandler)`（沿用 M0）。
- `InternalError` 删除，生成代码的 `ResponseErrorHandlerFunc` 接 `Write`，`RequestErrorHandlerFunc` 接 `BodyError`。

**`API`**：

```go
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (context.Context, string, error)
}
type APIConfig struct {
	Logger           *slog.Logger
	Authenticator    Authenticator
	PublicOperations []string      // 各模块公开操作的并集，例如 "POST /api/v0/auth/register"
	MaxBodyBytes     int64         // server.max_body_bytes
	RequestTimeout   time.Duration // server.request_timeout
}
func NewAPI(cfg APIConfig) *API                                                // API.Errors 是 APIErrors
func (a *API) Middlewares(bodies *bodyshape.Table) []func(http.Handler) http.Handler
type RequestMeta struct{ ClientIP netip.Addr; UserAgent string }
func RequestMetaFrom(ctx context.Context) RequestMeta
```

- `Middlewares` 按"请求元信息 → 期限 → 请求体上限 → 认证 → 请求体结构"的顺序排好，再**反转**返回（生成代码把列表最后一个包在最外层）。顺序由 `api_test.go` 的三个测试从外部核对（`TestMetaAndDeadlineRunBeforeAuthentication`、`TestAuthenticationRunsBeforeTheBodyCheck`、`TestBodyLimitRunsBeforeTheBodyCheck`）；去掉反转，这三个和 `TestRequestMetaClientIP`（请求元信息落到认证之后，认证器拿到的 `context` 里还没有它）共四个测试失败（变异核对）；`TestBodyCheckAnswersEveryProblemAs400` 仍然通过，因为结构检查到了最外层，在认证之前答出同样的 400。
- 请求元信息：`ClientIP` 取连接的对端地址，去掉端口和 zone，IPv4 映射的 IPv6 地址转为 IPv4；解析不了时是零值。可信代理在 P2 加入。
- 认证：`r.Pattern` 在公开集合中就放行，不看 `Authorization`；否则要求 `Authorization: Bearer <token>`（方案名不区分大小写，令牌不能有空白）。没有令牌 → 401 `WWW-Authenticate: Bearer`、`detail` "This operation requires a bearer token."；认证器返回 `ProblemStatus()` 为 401 的错误 → 401 `WWW-Authenticate: Bearer error="invalid_token"`、"The bearer token is invalid or has expired."；失败原因只进 DEBUG 日志（"authentication failed"，带 `request_id`、`route`、`error`）；认证器的其他错误走 `Write`。认证器返回的第二个值（限流键）P1 不用，P2 的限流用它。

### 2.9 接口描述约定与 `apitest`（M2 设计 3.11、3.12）

- **`api/common.yaml`**：`FieldError` 的 `required` 为 `[field, code, message]`，`code` 是十个值的枚举，每个值有说明；`Problem.code` 的说明列出平台码。
- **`security`**：每个操作显式写；`securitySchemes.bearer`（`type: http`、`scheme: bearer`）同时写在 `api/openapi.yaml` 和每个模块文件。`instance.yaml` 的 `getInstance` 写 `security: []`。
- **`x-problem-codes`**：顶层（`api/openapi.yaml`）为 `[bad_request, payload_too_large, internal_error]`；`rate_limited` 随 P2 的限流加入（见第 3 节第 2 条）。`getInstance` 写 `[]`。
- **`apitest` 的写法检查**（`rules_test.go` 的 `authoringViolations`，每条都有反例）：
  - 顶层没有 `x-problem-codes`，或其中有不是平台码的码；
  - 操作没有 `security`（"declares no security; write [{bearer: []}], or [] for a public operation"），或引用了 `components.securitySchemes` 中没有的 scheme；
  - 操作没有 `x-problem-codes`（"write [] when it answers only the top-level codes"）、不是列表、码的写法不合 `^([a-z]+\.)?[a-z_]+$`、前缀不是所在模块、不带前缀又不是平台码。
  - 平台码是 `bad_request`、`unauthorized`、`not_found`、`payload_too_large`、`validation_failed`、`rate_limited`、`server_busy`、`internal_error`、`not_ready` 九个。
- **`CheckResponse` 核对问题码**：problem 的码必须在"顶层 ∪ 这个操作的 ∪（需要令牌时）`unauthorized`"之内，否则报 `problem code %q is not declared: %s may answer %q`；每个答过的码记下来。
- **`apitest.Main(m, module)`**：模块的 handler 测试包用它做 `TestMain`。测试全部通过、而且没有用 `-run`、`-skip`、`-short` 缩小范围时，核对这个模块每个操作声明的码都至少被一个经过 `CheckResponse` 的测试答过，否则失败并列出 `operationId: code`。
- **`CheckRequest(t, req)`**：按契约核对测试要发出的请求（路径、方法、参数、请求体；不核对 `security`），请求体读完放回。
- **`TestComponentNamesAreTypeNames`** 跳过 `securitySchemes`：`bearer` 不是类型名，而且每个模块都声明同一个（见第 3 节第 6 条）。
- **oapi-codegen 的模块模板**（M0-P3 交接 1）：`output-options` 的 `name-normalizer: ToCamelCaseWithInitialisms`（沿用）、`nullable-type: true`、`type-mapping.string.formats` 中 `uuid` → `{type: uuid.UUID, import: uuid}`、`email` → `{type: string}`、`type-mapping.number.default` → `{type: float64}`，注释用中文，照抄到每个模块。`instance` 的 `server.gen.go` 用新模板生成后逐字节不变。

### 2.10 迁移与 sqlc（M2 设计 3.13、3.14、4.1–4.3、4.5）

- 迁移文件 `00001_identity_users.sql`、`00002_identity_profiles.sql`、`00003_identity_auth_sessions.sql`，DDL 与 M2 设计 4.2、4.3、4.5 的表逐列一致，文件头用中文写来源和约定。`embed.go` 改为 `//go:embed sql/*.sql`，删除 `sql/.gitkeep`（有了迁移文件，`all:` 的理由不在了）；`embed_test` 要求至少一个文件。
- **约束和索引的名字**（`TestConstraintAndIndexNames`，PG 18 的 `pg_constraint` 也列出 NOT NULL，查询排除 `contype = 'n'`）：`users_pkey`、`users_email_key`、`users_email_check`、`users_display_name_check`；`profiles_pkey`、`profiles_user_id_key`、`profiles_user_id_fkey`（`ON DELETE CASCADE`）、`profiles_theme_check`、`profiles_onboarding_step_check`、`profiles_language_check`、`profiles_start_of_the_week_check`；`auth_sessions_pkey`、`auth_sessions_user_id_fkey`（`CASCADE`）、`auth_sessions_token_hash_check`、`auth_sessions_generation_check`、`auth_sessions_revoke_reason_check`、`auth_sessions_revoked_consistent_check`、`auth_sessions_user_id_idx`、`auth_sessions_expires_at_idx`。
- **CHECK 的反例**（`TestChecksRejectCounterexamples`，25 个，每个核对 `23514` 和约束名）：邮箱的大写 ASCII 和非 ASCII、首尾空格、制表符、换行、中间空格、U+3000、U+2028；空的 `display_name`；`onboarding_step` 为数组、标量、JSON null、缺键、多键、值为字符串、null、数字；未知的 `theme`、`language`；`start_of_the_week = 7`；`token_hash` 不是 32 字节；负的 `generation`；未知的 `revoke_reason`；`revoked_at` 与 `revoke_reason` 只有一个。正例包括 `élodie@exämple.com`（CHECK 接受；领域层按 Django 的规则拒绝非 ASCII 的本地部分，见 2.11）和两次 `||` 合并。
- `TestMigrationsGoUpDownAndUpAgain`：三个迁移 up、逐个 down 到没有表、再 up。
- **`server/sqlc.yaml`**：每个模块一个条目。`identity` 的 `schema` 只列三个迁移，`queries` 是 `internal/modules/identity/adapter/postgres/queries`，输出 `…/postgres/gen`，`package gen`、`sql_package: pgx/v5`、`emit_pointers_for_null_types: true`；类型覆盖：`uuid` → 标准库 `uuid.UUID`（可空时指针），`timestamptz` → `time.Time`（可空时指针；`db_type` 写 `timestamptz`，写 `pg_catalog.timestamptz` 不匹配）。`inet` 生成 `*netip.Addr`，不需要覆盖。
- **查询**（审计列一律取用例的时钟，`sqlc.arg(now)`）：

  ```sql
  -- name: CreateUser :exec
  INSERT INTO users (id, email, password, display_name, created_at, updated_at)
  VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password), sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now));

  -- name: GetUser :one
  SELECT id, email, first_name, last_name, display_name, user_timezone, created_at
  FROM users WHERE id = sqlc.arg(id);

  -- name: CreateProfile :exec
  INSERT INTO profiles (id, user_id, created_at, updated_at)
  VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));

  -- name: CreateSession :exec
  INSERT INTO auth_sessions (id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, updated_at)
  VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), 0, sqlc.arg(user_agent), sqlc.arg(ip),
          sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now));

  -- name: GetSessionCredential :one
  SELECT s.user_id, s.expires_at, s.revoked_at, u.is_active AS user_active
  FROM auth_sessions s JOIN users u ON u.id = s.user_id
  WHERE s.id = sqlc.arg(id);
  ```

  `GetSessionCredential` 读出 `revoked_at` 本身：`revoked_at IS NOT NULL AS revoked` 会被 sqlc 生成为 `interface{}`（原型实测）。
- **`make gen-go`** 最后执行 `cd server && CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc generate`；`GEN_GO_OUT` 加上 `server/internal/modules/*/adapter/postgres/gen`，`gen-go` 先删掉其中的 `*.go`。
- **`TestSQLCSchemaScope`**（`internal/archtest`，用 koanf 的 yaml 解析器读 `server/sqlc.yaml`，纯函数 `sqlcScopeViolations` 另有 12 个人造布局的正反例）：
  - 迁移文件名合 `NNNNN_<module>_<description>.sql`；
  - 每张表只由一个迁移 `CREATE`（"tables: %s is created by both %s and %s"），表的所有者是创建它的迁移的模块；
  - `ALTER TABLE` 只改已有的表（"migration %s alters %s, which no migration creates"），而且迁移文件归被改表的模块（"… which module %s creates: the migration belongs to %s"）；同一文件 Up、Down 两段改同一张表只报一次；
  - 每个 sqlc 条目：`schema` 恰好是这个模块的全部迁移（多一个、少一个都报），`queries` 和 `out` 在这个模块的 `adapter/postgres` 下；
  - 有 `adapter/postgres/queries` 目录的模块必须有条目（"module %s has adapter/postgres/queries but no sqlc entry"）。
- **S1**（`e2e/stories/smoke/s1-server-ready.spec.ts`）：`nerve migrate status` 的表头是 `VERSION  STATE  APPLIED AT  SOURCE`，每一行的状态是 `applied`、来源依次等于 `server/migrations/sql/*.sql`；`goose_db_version` 的最大版本等于最后一个文件的序号（M0-P6 交接）。`e2e/tsconfig.json` 的 `lib` 改为 `ES2023`（用 `toSorted`；oxlint 不许原地 `sort`）。

### 2.11 `identity` 领域（M2 设计 3.4、3.8、4.2、5.4）

- **邮箱**：`NormalizeEmail = strings.ToLower(strings.TrimSpace(s))`；`ValidEmail`：非空、不超过 255 个字符、合法 UTF-8、任何位置都没有空白（`unicode.IsSpace`）和控制字符，并通过 Django 5.2 `EmailValidator` 的规则。
  - Django 的正则逐条移植到 RE2：把 lookaround 改写成等价的显式形式；Django 用 `re.IGNORECASE` 编译，Python 在这个标志下让 `[a-z]` 也匹配 `ı`（U+0131）和 `ſ`（U+017F），本地部分的字符类加上这两个；`localhost` 和 `[IP]` 字面量照 Django 处理。
  - 用 Django 的原始正则（Python `re`）逐条对照 60 个样例：Go 与 Python 结果全部一致；只有 3 个样例两边不同，都是 Nerve 额外拒绝的（转义的制表符、U+3000、256 个字符），与 M2 设计 4.2 的"不允许任何空白和控制字符，长度不超过 255"一致。
  - 非 ASCII 的本地部分（例如 `élodie@…`）按 Django 的规则不合法；Plane 同样拒绝。
- `DisplayNameFromEmail`：第一个 `@` 之前的部分（Plane 的 `User.save()`）。
- **密码**（`PasswordRules`，`NewPasswordRules()` 解析嵌入的名单：文件头、一个空行，然后每行一条）：`Check(field, password, email) *shared.FieldError`：空 → `required`；长度（按 UTF-16 码元，与界面的 `password.length` 相同）不在 8–128、或缺大写、小写、数字、`!@#$%^&*()-_+=[]{}|;:'",.<>?/` 中的特殊字符 → `weak_password`（"must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of …"）；小写的整个密码或主干在名单中、或主干等于邮箱本地部分的主干 → `common_password`（"is too common"）。主干：转小写后从两端去掉所有不是字母（`unicode.IsLetter`）的字符。
- **名单**：`tools/password-blocklist/build.mjs <PwnedPasswordsTop100k.txt>` 核对 SHA-256 `c2e5696882c603b76bb67a47ee970897e5a76fc4c3f5547abe3d0ca340c576e0`，按 M2 设计 3.8 的两个过滤条件留下 33,887 条，转小写、去重、按字节排序，写出 `common_passwords.txt`（278,161 字节：16 行文件头、一个空行和 33,887 行名单）。文件头写来源（NCSC 的原地址和 SecLists 中的路径）、SHA-256、"Contains public sector information licensed under the Open Government Licence v3.0:" 和 OGL 的链接，以及过滤规则。`TestCommonPasswordList` 核对条数、排序去重、每条都满足某个过滤条件、文件头的 SHA 和许可两行。
- **下载地址**：NCSC 的原地址 `https://www.ncsc.gov.uk/static-assets/documents/PwnedPasswordsTop100k.txt` 已失效（2026-09-26 实测 404）。SecLists 的 `Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt`（`https://raw.githubusercontent.com/danielmiessler/SecLists/master/…`）与设计阶段下载的文件 SHA-256 相同，plan 从这里下载；脚本核对 SHA-256，换了内容就拒绝。
- **许可的核对**（M2 设计 3.8 要求 P1 对具体数据文件再核一次）：NCSC 网站的条款页写明网站内容可按 OGL v3.0 再使用，须注明来源并尽可能附上 OGL 的链接；例外只有注明为第三方许可的材料、第三方图片和徽标，这个文件不在其中。名单的原始数据来自 Have I Been Pwned 的 Pwned Passwords，HIBP 的 API 文档写明 Pwned Passwords 没有许可和署名要求（欢迎署名）。所以文件头和 README 写 OGL 的声明和链接，另写"from Have I Been Pwned"。NCSC 当年发布这份名单的博客页已下线，只剩文件本身。
- **刷新令牌的布局**：`RefreshToken{SessionID, Generation uint32, Secret [32]byte, Tag [16]byte}`；`MACMessage()` 是前 52 字节；`SecretHash()` 是密文的 SHA-256；`String()` 是 `nrv_rt_` 加上 68 字节的 base64url（不补 `=`，91 个字符），合 M2 设计 8.6 的扫描模式 `nrv_rt_[A-Za-z0-9_-]{91}`。P1 只签发，不解析：解析和标签的核对随续期在 P2 加入（见第 3 节第 3 条）。
- `SanitizeUserAgent`：非法 UTF-8 换成 U+FFFD，去掉 NUL，截到 512 个字符。
- `NewAccount(rules, email, password) (string, error)`：规范化后检查邮箱（`required`、`too_long`、`invalid_format`）和密码，全部问题一次返回 422 `validation_failed`；成功时返回规范化的邮箱。
- 模块错误：`ErrSignupDisabled`（403 `identity.signup_disabled`，"Sign-up is disabled on this instance."）、`ErrEmailTaken`（409 `identity.email_taken`，"An account with this e-mail address already exists."）。

### 2.12 `identity` 用例与端口（M2 设计 6.2、6.3）

- 端口（`identity/app/ports.go`）：`Clock`；`UserCreator.CreateUser(ctx, NewUser)`（地址已用时返回 `domain.ErrEmailTaken`）；`UserReader.GetUser(ctx, id)`（没有时 `ErrNotFound`）；`ProfileCreator.CreateDefaultProfile(ctx, id, userID, now)`；`SessionCreator.CreateSession(ctx, NewSession)`；`SessionReader.SessionCredential(ctx, id) (SessionCredential{UserID, ExpiresAt, Revoked, UserActive}, error)`；`PasswordHasher.Hash(ctx, password)`；`AccessTokens{Issue(AccessClaims); Verify(token, now)}`，`ErrAccessTokenExpired` 表示签名有效而只是过期；`RefreshTokenMAC.Tag(message) [16]byte`；`SignupPolicy.AllowSignup(ctx)`。
- **`Register.Execute(ctx, RegisterInput{Email, Password, UserAgent, IP}) (Tokens, error)`**：
  1. `SignupPolicy` 关闭时先答 403，不做其他任何检查（M2 设计 3.9）；
  2. `NewAccount` 校验（422）；
  3. 在事务之外哈希（M2 设计 3.5）；
  4. 生成账户 id、会话 id（`uuid.NewV7()`）、随机密文、标签，签出访问令牌——都在事务之前，事务提交之后不会再失败；
  5. 一个事务写入账户、默认资料、第 0 代会话（`expires_at = now + session_ttl`）；地址已用 → 409；
  6. 记 INFO "account registered"（只带 `user_id`，不带邮箱、密码和令牌，`TestRegisterLogsTheAccountButNoSecret`）。
- `GetMe.Execute(ctx)`：`RequireActor` 后按 id 读账户；读不到（账户已不存在）答 401。
- **`Authenticate.Execute(ctx, token) (shared.Actor, error)`**：验签和 `exp`，再按主键查一次会话和账户（`GetSessionCredential`）。会话不存在、属于别的账户、已撤销、已过期（`now >= expires_at`）、账户已停用，都是包着原因的 `shared.Unauthenticated()`（`fmt.Errorf("%w: %w", …)`）；过期的访问令牌同时满足 `errors.Is(err, ErrAccessTokenExpired)`；数据库错误原样返回（不是 401，`TestAuthenticateDatabaseFailureIsNot401`）。`nrv_pat_` 前缀的分支随 PAT 在 P3 加入；在那之前 PAT 按 JWT 解析失败，同样是 401。

### 2.13 `identity` 适配器

- **`adapter/argon2`**（包名 `argon2adapter`）：`New(Params{MemoryKiB, Iterations, Parallelism, MaxConcurrent, MaxWait}, logger)`；`Hash` 输出 `$argon2id$v=19$m=…,t=…,p=…$<盐>$<哈希>`（盐 16 字节、输出 32 字节，`RawStdEncoding`）。名额用带缓冲的通道；拿不到时最多等 `MaxWait`，期间请求结束就返回 `ctx.Err()`，超时返回 `shared.ServerBusy(time.Second)`（503、`Retry-After: 1`），并记 INFO "password hashing is saturated"（带排队数 `queued`）。
- **`adapter/signing`**：
  - `ParseKeys(pem)`：只接受 PKCS#8 的 `PRIVATE KEY` 块里的 Ed25519 私钥；错误不引用输入（`not a PEM "PRIVATE KEY" block (PKCS#8)`、`parse PKCS#8: …`、`the key is %T, want an Ed25519 key`）。`EphemeralKeys()` 生成本进程的临时密钥。
  - MAC 密钥 `hkdf.Key(sha256.New, seed, nil, "nerve refresh-token mac v1", 32)`（标准库 `crypto/hkdf`）；`RefreshTokenMAC.Tag` 是 HMAC-SHA256 的前 16 字节。
  - `AccessTokens.Issue`：头部 `{"alg":"EdDSA","typ":"JWT"}`，载荷只有 `exp`、`sid`、`sub`。`Verify`：`WithValidMethods(["EdDSA"])`、`WithExpirationRequired()`、`WithStrictDecoding()`、`WithTimeFunc(now)`；jwt/v5 先验签后验声明，所以 `jwt.ErrTokenExpired` 意味着签名有效，映射为 `app.ErrAccessTokenExpired`。`sub`、`sid` 都必须解析为 uuid。`TestAccessTokenVerifyRejects` 覆盖 10 种：别的密钥、`HS256`（用公钥字节做 HMAC 密钥）、`none`、篡改载荷、非严格的 base64（带 `=`）、缺 `exp`、缺 `sub`、缺 `sid`、刷新令牌、空串。
- **`adapter/postgres`**（包名 `postgresadapter`）：`Store` 用 `gen.New(postgres.DB(ctx, pool))`，在事务内外都能用。`CreateUser` 把 `users_email_key` 的唯一冲突（`23505`）映射为 `domain.ErrEmailTaken`，其他错误（包括 CHECK 违反）包一层返回，成为 500（见第 3 节第 1 条）。`GetUser`、`SessionCredential` 把 `pgx.ErrNoRows` 映射为 `app.ErrNotFound`。`CreateSession` 的零值 IP 写 `NULL`。集成测试断言审计列等于固定时钟（`TestCreateAndGetUser`）。
- **`adapter/authn`**：`Authenticator.Authenticate` 调用 `Authenticate` 用例，成功时返回 `shared.WithActor(ctx, actor)` 和 `session:<sid>`；编译期断言它满足 `httpserver.Authenticator`（在测试中）。

### 2.14 接口：`register` 与 `getMe`（M2 设计 5.1、5.2、5.4）

`api/modules/identity.yaml`：

| 操作 | `security` | `x-problem-codes` | 成功 |
|---|---|---|---|
| `register`：`POST /api/v0/auth/register` | `[]` | `identity.signup_disabled`、`validation_failed`、`identity.email_taken`、`server_busy` | 201 `AuthTokens` |
| `getMe`：`GET /api/v0/me` | `[{bearer: []}]` | `[]`（`unauthorized` 由 `bearer` 隐含） | 200 `User` |

- `RegisterRequest`：`email`（`format: email`，映射为 `string`，格式由领域校验）、`password`，都必填，`additionalProperties: false`。
- `AuthTokens`：`token_type`（枚举 `Bearer`）、`access_token`、`access_token_expires_in`（秒）、`refresh_token`、`refresh_token_expires_at`（会话的绝对期限）。
- `User`：`id`、`email`、`first_name`、`last_name`、`display_name`、`user_timezone`、`avatar_url` 和 `cover_image_url`（`[string, 'null']`，必填，M5 之前为 `null`）、`created_at`。
- 必填又可为空的字段生成 `nullable.Nullable[string]`，它的零值表示"没传"，会写成 `""`；handler 显式写 `nullable.NewNullNullable[string]()`，`TestGetMe` 逐字核对 JSON 中的 `null`。
- `httpadapter`：`UseCases{Register, GetMe}`（两个小接口）；`PublicOperations()` 是 `["POST /api/v0/auth/register"]`；`Register(router, api, uc)` 把 `api.Middlewares(gen.BodyShapes())` 交给生成代码。`register` 从 `httpserver.RequestMetaFrom(ctx)` 取 UA 和 IP。
- 模块入口 `identity.New(Deps) (*Module, error)`：`Deps{Pool, Tx, Clock, Logger, SignupPolicy, SigningKeyPEM, AccessTokenTTL, SessionTTL, Password PasswordHashing}`；没有密钥时记 WARN "auth.jwt.private_key_file is not set: signing with an ephemeral key; access tokens stop verifying at restart (dev and test only)"；密钥解析失败返回 `auth.jwt.private_key_file: <原因>`。方法 `PublicOperations()`、`Authenticator()`、`Register(router, api)`。`instance` 的入口改成同样的形状。

### 2.15 `bootstrap` 接线与整程序测试（M2 设计 3.6、3.11、6.1）

- `newApp`：先读签名密钥文件（`readSigningKey`，`fs.PathError` 只留原因），再建连接池、迁移器、`identity`（`TxManager` 用 `database.commit_timeout`，`SignupPolicy` 是 `auth.signup_enabled` 的开关类型 `signupSwitch`）、`instance`、`Router`、`API`（公开操作是两个模块的并集，记在 `app.publicOperations`），两个模块 `Register`，最后挂 web 界面。编译期断言 `shared.TxManager` 和 `httpserver.ProblemError` 两个按结构的满足关系。
- `warnIfExposed`（`run` 开始时）：非 prod、又监听在回环地址之外（`localhost`、`127.0.0.0/8`、`::1` 之外，包括 `:8080`、`0.0.0.0`、主机名）时记一条 WARN "not running as prod but listening beyond this machine: …"（M2 设计 6.1）。
- `Serve` 在 logger 建好之后出现致命错误时，另记一条 ERROR "nerve serve failed"（M0-P2 交接 8 第二项；stderr 的一行照旧）。
- 测试的 `testConfig` 在 `127.0.0.1:0` 上监听，经 `server.addr_file` 取得地址（不再先猜端口）。
- **四个整程序测试**（`bootstrap/contract_test.go`；`apitest` 的 `Operations()` 从契约列出全部操作，新加的操作自动进入）：
  1. `TestPublicOperationsAreTheContractsPublicOperations`：`app.publicOperations` 排序后等于契约中 `security: []` 的操作；
  2. `TestAPIRoutesAreTheContractsOperations`：`Router.Patterns()` 中路径以 `/api/v0/` 开头的模式（带方法或不带方法的都算）等于契约的全部操作；
  3. `TestOperationsThatNeedATokenAnswer401WithoutOne`：每个非公开操作不带令牌得到 401 `unauthorized`、`WWW-Authenticate: Bearer`，并过 `CheckResponse`；路径参数和必填的查询参数由 `Operation.Target()` 填上合法值（P1 没有这类参数，`TestTarget` 用人造契约证明它）；
  4. `TestBodiesThatBreakTheStructureAnswer400`：对每个带 JSON 请求体的操作，`Operation.BodyCases()` 由 schema 造出合法的请求体再逐项改坏（M2 设计 3.11 的七种；这个操作没有的种类跳过），发给真实的程序和数据库（非公开操作带注册得到的令牌），断言 400 `bad_request` 且 `errors` 的 `field`、`code` 逐项相等；可为空的字段传 `null` 不得到 400。P1 的 `register` 产生四个情况：未知字段、缺 `email`、缺 `password`、一次返回全部问题。
- 另有：`TestEveryKindBecomesItsProblem`（每个 `Kind` 经 `APIErrors.Write` 得到的状态、码、字段和 `Retry-After`，包括包了一层的错误；M2 设计 3.11"bootstrap 的测试逐个 Kind 核对映射"）、`TestRegisterThenGetMe`（真实数据库、密钥文件；核对访问令牌确实由这把密钥签名）、`TestBadSigningKeyFileStopsTheApp`、`TestEphemeralSigningKeyIsAWarning`、`TestWarnIfExposed`、`TestServeLogsAFatalError`。
- 原型的变异核对：公开集合漏掉 `identity` 的一项 → 测试 1、4 和 `TestRegisterThenGetMe` 失败；多挂一个 `/api/v0/debug` → 测试 2 失败。

### 2.16 端到端（M2 设计 9.5；M0-P6 交接）

- **`server.ts`**：`startNerve(databaseUrl, logFile, env = {})`。nerve 在 `127.0.0.1:0` 上监听，`NERVE_SERVER__ADDR_FILE` 指向日志旁边的 `<名字>.addr`；等待时先读地址文件，再轮询 `/readyz`，都受同一个 30 秒的期限约束，每个请求只用剩余时间。M0 的空闲端口猜测和"就绪后再等一个轮询间隔"的缓解一起删除（端口竞争的根本解决）。
- **`db.ts`**：`createDatabase(name, template?)` 返回 URL；`openDatabase(name)` 返回每个 worker 自己的 `pg` 连接池（`max: 2`，连接 10 秒、查询 30 秒的期限），带 `query`、`dump(file)`、`close()`。全局准备只连接模板库、执行 `migrate up`，不开连接池（模板库上有会话时 `CREATE DATABASE … TEMPLATE` 会失败）。
- **失败时的数据库快照**：`test.ts` 的自动 fixture `databaseSnapshot` 在测试失败时，用 `docker exec <容器> pg_dump --no-owner` 导出本 worker 的库到 `testInfo.outputPath("database.sql")` 并作为附件（60 秒期限，`SIGKILL`）；全局准备把容器 id 放进 `NERVE_E2E_POSTGRES_CONTAINER`。不录像（M2 设计 9.5）。持续集成上传的 `e2e/test-results/` 已经包含它。
- **`nerveWith(env)`**（测试级 fixture）：在本 worker 的库上另起一个 nerve（例如 `NERVE_AUTH__SIGNUP_ENABLED=false`），测试结束时停止；每起一个，就把 nerve fixture 的预算加到这个测试的超时上，fixture 自己的期限先到。
- **`auth.ts`**：`password`（`Tr0ub4dor&3`）、`emailFor(testInfo, label)`（由测试 id、`repeatEachIndex` 和 `retry` 组成，`--repeat-each` 在同一个 worker 里重跑也不冲突）、`register(api, email, headers)`。登录和 `signedInPage` 在 P2、P4 加入。
- **`assert/identity.ts`**：`expectRegistered(db, {email, refreshToken, userAgent, ip})`（A1 的数据库断言：邮箱已转小写、`$argon2id$`、`display_name`、`is_active`；默认资料逐列；会话的 id 等于令牌里的会话 id、`generation = 0`、`token_hash` 等于令牌密文的 SHA-256、UA、IP、`expires_at - created_at` 恰好 30 天、未撤销）；`countIdentity`、`expectNothingAdded`（A2）。
- **故事**：`e2e/stories/identity/a1-sign-up.spec.ts`（"A1 (API): …"，另核对访问令牌能立即读 `/me`）、`a2-sign-up-refused.spec.ts`（409 大小写不同的已有地址；422 `weak_password`、`common_password`（`Password1!`、`Password1!~`）；关闭注册的独立 nerve 对新地址和已有地址都答 403；最后核对三张表的行数不变）。页面版本随 P4 加入。

### 2.17 文档、README 与交接（M2 设计 3.20、8.7、13.1）

plan 的 Task 13 逐行给出文字。要点：

- **总体设计**：3.1（注册返回令牌，不返回账户）；3.5（新平台码、`FieldError.code`、400 与 422 的分界、`x-problem-codes`）；4.1（刷新令牌的格式与 MAC）；4.2（注册默认值、第一个账户）；5.5（审计时间列由用例的时钟写入）；5.6（改别的模块的表的迁移归被改表的模块）；6.2（端口在使用方的 `app`；`shared` 的内容；时钟端口由各模块声明；`module.go` 的入口形状）；6.3（平台不导入 `shared`；sqlc 的 `schema` 按模块限定）；6.4（按路由的中间件及其顺序；参数先绑定；`TxManager` 的提交规则）；8.2（失败时的产物：trace、截图、nerve 日志和数据库快照，不录像；`server.ts` 读地址文件）。
- **M0 设计** 3.1：包职责表中的平台依赖、`module.go` 和 `internal/shared`；3.3：按路由的中间件及其顺序；另把"生成代码的错误出口"一段改为 `BodyError`、`Write`（3.1 和这一段见第 3 节第 10 条）；3.5：`//go:embed sql/*.sql` 与"M0 不包含任何迁移文件"改为现状。
- **差异清单**：一 B（`sessions` 由 `auth_sessions` 替代）、二·全局（3.13 的约定）、二·按表（`users`、`profiles` 逐列，`auth_sessions` 按实际的列改写）、三（错误码逐个操作声明；请求体按契约拒绝）、四（4.6 中标 P1 的五行）。
- **README**：8.7 中 P1 的四行（`NERVE_ENV=prod` 与启动提醒；签名密钥的生成与换钥的后果；常见密码名单的第三方声明；迁移角色要能读 `goose_db_version`）。
- **交接**：M0-P1、M0-P4 标为 `done`；M0-P2、M0-P3、M0-P6 追加"处理结果（M2/P1）"一节，列出已处理的条目和留给 P2、P3 的条目，状态仍为 `open`。

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M2 设计的架构；第 1、2、3 条是按"删除不用的代码"做的取舍，第 10 条是 3.20 表漏掉的同步位置。

1. **CHECK 违反是 500，不映射为领域错误**。P1 的仓储只把 `users_email_key` 的唯一冲突映射为 `identity.email_taken`；其余约束违反说明领域校验和数据库不一致，是缺陷，按 3.11 成为 500 并记日志（`TestCreateUserBreakingACheckIsInternal`）。M2 设计 3.13 的"仓储就不能把它映射为领域错误"说的是 `check_violation` 能否被识别，P1 没有需要识别它的用例。
2. **顶层 `x-problem-codes` 在 P1 只有三个码**：`rate_limited` 随 P2 的限流一起加入。P1 没有任何代码能答出 429，提前声明就是一条假的契约。M2 设计 3.11 的"所有操作都可能返回的四个码"在 P2 成立。
3. **设计中暂无使用者的成员，随使用者在 P2、P3 加入**：`ExpiredCredential()`（P2 的失败闸门据此退回单位；P1 已有 `app.ErrAccessTokenExpired`，`errors.Is` 能区分）、argon2 的 `Verify`（P2 登录）、刷新令牌的解析和标签核对（P2 续期）、`Actor.APITokenID` 和 `nrv_pat_` 分支（P3 的 PAT）。`Authenticator` 的签名照设计返回限流键，`authn` 已返回 `session:<sid>`，平台在 P2 使用。
4. **签名密钥在 `identity.New` 中解析**：`config.validate` 只检查 prod 是否设置（它不读文件）；文件由 `bootstrap` 在启动时读取，内容交给 `identity`，解析失败时 `newApp` 返回错误、进程退出。满足 M2 设计 3.7 的"prod 必须提供，否则启动失败，并指出这个配置项"。
5. **中间件顺序的测试在 `httpserver`**，不在 `bootstrap`：顺序只在 `API.Middlewares` 里写一次，在平台测它最直接；`bootstrap` 的整程序测试再从外部覆盖一遍（未知字段在无令牌时是 401 而不是 400，由测试 3 和 4 共同保证）。
6. **`securitySchemes.bearer` 不受"组件名是 PascalCase 类型名"的约束**：M0 的 `TestComponentNamesAreTypeNames` 会拒绝它；M2 设计 3.12 要求每个模块都写同一个 `bearer`，所以这条测试跳过 `securitySchemes`。
7. **`nullable` 在 P1 加入**：`User` 的 `avatar_url`、`cover_image_url` 必填又可为空，生成代码用 `nullable.Nullable[string]`。M2 设计 6.6 的"随第一个用到它的生成代码加入"指的正是这里。handler 必须显式写 `null`（零值会写成 `""`），测试逐字核对。
8. **`élodie@exämple.com`**：M2 设计 4.2 记录它通过 CHECK，这是对 CHECK 的实测，不是领域的规则。领域层按 Django 的 `EmailValidator` 拒绝非 ASCII 的本地部分（Plane 同样拒绝），所以它不会从应用写入；`schema_test` 把它作为 CHECK 的正例保留。
9. **适配器的包名**：`adapter/postgres`、`adapter/argon2` 的包名是 `postgresadapter`、`argon2adapter`，与 `platform/postgres`、`golang.org/x/crypto/argon2` 不冲突；目录名不变，符合规则 6 的布局。以后的模块照这个写法。
10. **3.20 表漏掉的同步位置**（P1 在 Task 13 一并同步）：M0 设计 3.1 的包职责表仍写 `Register(mux, apiErrors)`、平台只是"不能依赖 `modules`"、`internal/shared`"预计在 M2"建立；M0 设计 3.3 的"生成代码的错误出口"一段仍写 `InternalError`；M0 设计 3.5 仍写 `//go:embed all:sql` 和"M0 不包含任何迁移文件"；总体设计 6.2 的 `module.go` 一行仍是 `Register(mux, apiErrors)`；总体设计 8.2 的 `server.ts` 仍写"随机端口"。
11. **`golang.org/x/text` 被 x/crypto v0.57.0 抬到 v0.42.0**（间接依赖）。
12. **名单的 NCSC 原地址已失效**：改从 SecLists 下载同一个文件（SHA-256 相同，2.11）；文件头同时写两处来源。

## 4. 验收标准（完成线，M2 设计 12 节 P1）

- [ ] A1、A2 的接口版本和 S1–S4 在持续集成中通过；`TestBuiltInProfiles` 证明 prod 默认关闭注册。
- [ ] 四个整程序测试和 `apitest` 的新核对（写法、问题码、`Main` 的覆盖、`CheckRequest`）通过。
- [ ] `make gen-check` 覆盖 sqlc 和请求体结构表的输出；`TestGenerateIsDeterministic` 通过。
- [ ] 架构测试（含 `TestSQLCSchemaScope`）和传递依赖测试通过。
- [ ] 审计列等于固定时钟的集成测试、CHECK 的反例测试通过。
- [ ] `TxManager` 在请求的 `context` 被取消后仍然提交、仍然回滚的集成测试通过。
- [ ] `make test`、`make lint-go` 覆盖工具模块，持续集成的日志里能看到 `ok …/server/tools/bodyshapegen`。
- [ ] `make lint`（含 `make lint-web` 的关键词守卫和 `e2e` 的 oxlint 上限 0）、`make knip` 通过。
- [ ] 3.20 中 P1 的各行、8.7 中 P1 的 README 内容已在同一次合并中写好；交接按第 7 节处理。

## 5. 不在 P1 范围内

- 登录、续期、退出、限流、失败闸门、安全响应头（P2）；PAT、资料和偏好、修改密码、停用、`users` 命令、River、游标、`oapi-codegen/runtime`（P3）；前端（P4、P5）。
- `@nerve/api-client` 生成命令的 `--root-types`（M2 设计 3.12）：使用方是 P4 的 stores（7.5），随 P4 加入。
- `nerve users create`：prod 默认关闭注册，P1 到 P3 之间 prod 只能用 `NERVE_AUTH__SIGNUP_ENABLED=true` 临时打开注册来建第一个账户。v0 在 M2 收尾前不发布，没有实际影响。

## 6. 风险

| 风险 | 应对 |
|---|---|
| Django 邮箱规则的移植走样 | 60 个样例与 Python `re` 下的原始正则逐条对照一致（附录 A）；单元测试保留这些样例 |
| 名单生成脚本与 Go 的主干规则走样 | `TestCommonPasswordList` 核对每一条都满足某个过滤条件；脚本重新生成的文件与提交的逐字节相同 |
| argon2 在持续集成的机器上比本机慢 | 默认参数本机约 14 毫秒一次（附录 A）；测试环境用 m = 64 KiB、t = 1；等待上限 2 秒。持续集成的实测写进 P1 review |
| `pg_dump` 快照依赖 `docker` 命令 | 持续集成的 ubuntu 镜像自带；本地 `make e2e` 本来就需要 Docker。快照失败只让这个失败的测试多一条错误，不掩盖原来的失败 |
| 端到端每个 worker 另起的 nerve 增加耗时 | 只有 A2 用 `nerveWith`，一个 nerve 启动约 0.1 秒 |
| 生成器遇到以后 M 的新写法 | 生成失败并说明原因（2.4），那个 M 带着测试扩展生成器 |

## 7. 交接的处理

| 交接 | P1 处理的条目 | 留下的条目 | 状态 |
|---|---|---|---|
| M0-P1-sqlc-cgo | 1 一律 `CGO_ENABLED=0` 运行 sqlc（wasm 的 libpg_query），不需要 C 编译器，也不需要 Docker 退路；原型核对 cgo 与非 cgo 的输出逐字节相同。2 迁移不写 PG 18 的语法和 `uuidv7()`（3.13） | — | done |
| M0-P2-platform-notes | 1 `TxManager`；2 中认证按路由挂载；3 `RequestID`；4 `*_file` 只记是否设置；5 中请求期限；6 规则 6 推广；7 S1 的迁移断言和 README 的迁移角色；8 三个小问题 | 2 中限流（P2）和接口调用日志（M8）；5 中 River、停机顺序、连接池关闭的时限、River 的迁移（P3） | open |
| M0-P3-api-codegen-notes | 1 生成选项（模块模板）；2 中映射、分层、请求体解码和 handler 两个出口、413、`context.Canceled`、`CheckRequest`；3 `security` 的写法；4 模块入口；5 组织规则（M2 没有 map 型对象） | 1 中 google/uuid 的守卫（P3）；2 中参数绑定的出口（P3） | open |
| M0-P4-schema-conventions | 1 全部（3.13）；2 主键与 ID | — | done |
| M0-P6-e2e-notes | 数据库断言的连接池与写法；失败时的数据库快照；S1 的迁移版本；端口竞争的根本解决；新等待的期限；是否录像（不录像，同步总体设计 8.2） | PAT 对等验收与认证 fixture 的登录、PAT、页面部分（P2、P3、P4）；S3 的新字段（P3）；S2 的断言（P4）；River 停机（P3）；fixture 写法的延伸（M4、M5、M8） | open |

## 附录 A：原型验证记录（2026-09-26）

原型在 `$M2TMP/p1proto`（仓库文件的副本，Go 1.27.1、Node 24.15.0、Docker 29.7.2、Apple M5 Max）。plan 中的代码就是原型中运行过的代码，由脚本从原型文件原样拼入 plan。另在 `$M2TMP/p1stage` 从 `b253681` 的仓库副本出发，**按 plan 的 Task 1–12 逐个应用**（最终文件加上 plan 写明的过渡版本，以及 plan 写明的 `go get`、`go mod tidy`、名单生成命令），改了生成来源的 Task 之后先运行 `make gen`，每个 Task 之后都运行 `make lint-go` 和 `make test`，全部通过；Task 6 和 Task 12 之后另跑了端到端（S3 除外，原因见下）。Task 12 之后 `p1stage` 与原型逐文件相同（含生成物、`go.mod`、`go.sum`、名单）。

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物没有差异 | `make gen` 前后逐目录比较 `apigen`、`modules/*/adapter/http/gen`、`modules/*/adapter/postgres/gen`、`api/dist`、`schema.gen.ts`（原型不是 git 仓库，用脚本代替 `git status`） | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过；`go -C server/tools test` 输出 `ok …/tools/bodyshapegen` |
| 前端检查 | `turbo run check:types check:lint check:format check:sync`；`make knip`；关键词守卫（原型不是 git 仓库，用遍历目录的等价脚本） | 54 个任务通过；knip 通过；没有新的命中 |
| 端到端 | `make build`，再 `playwright test` | S1、S2（两个）、S4、A1、A2 通过；`--repeat-each=5 --workers=1` 30 个全部通过。S3 在原型中失败，原因是副本不是 git 仓库，`go build` 取不到提交号（`commit: "unknown"`）；这次失败同时证明了失败时的 `pg_dump` 附件（249 行，含三张表） |
| sqlc 不用 cgo | `CGO_ENABLED=0` 与 `CGO_ENABLED=1` 各生成一次 | 输出逐字节相同；非 cgo 用 `wasilibs/go-pgquery`，cgo 用 `pg_query_go/v6/parser` |
| 加入 sqlc 不影响 oapi-codegen | 加 tool 前后比较 `tools/go.mod` 中 oapi-codegen 的依赖和 `instance` 的 `server.gen.go` | 版本不变，生成代码逐字节相同 |
| `bodyshapegen` 确定性 | `TestGenerateIsDeterministic`（10 次） | 相同 |
| 格式检查与解码一致 | `TestFormatsMatchTheDecoder`（13 个样例，含 JSON 转义写法的日期和 uuid） | 检查器与 `json.Unmarshal` 的判断逐个相同 |
| argon2 耗时 | `go test -bench BenchmarkHashDefaultParams -count=3 ./internal/modules/identity/adapter/argon2/` | 默认参数（m = 19456 KiB、t = 2、p = 1）每次 13.9–14.2 毫秒，分配约 19 MiB；设计估计 40 毫秒 |
| 中间件顺序的测试有效 | 去掉 `Middlewares` 中的 `slices.Reverse` | 4 个测试失败 |
| 整程序测试有效 | 公开集合漏掉 `identity`；多挂 `/api/v0/debug` | 前者 3 个测试失败，后者测试 2 失败 |
| lint 覆盖工具模块 | 在 `bodyshapegen` 放一个未使用的函数 | `make lint-go` 报 `unused` |
| 邮箱规则 | `$M2TMP/p1tools/django_email_check.py` 用 Django 5.2.15 的原始正则跑 60 个样例 | 与 Go 的 `ValidEmail` 全部一致（Nerve 另外拒绝的 3 个除外，见 2.11） |
| 名单 | 从 SecLists 下载（SHA-256 与设计阶段的文件相同），`node tools/password-blocklist/build.mjs <文件>` | 99,840 行，转小写去重 97,746 条，留下 33,887 条；输出文件 278,161 字节，SHA-256 `74d064a43b9eac19a5e4d58b06ceca3dc740e1df7c0de4a6d95c6efbf5167914`；在 `p1stage` 重跑逐字节相同 |
| 许可 | NCSC 条款页、HIBP API 文档 | 见 2.11 |

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型只在本机运行。风险低：sqlc 不用 cgo，其余与 M0 相同；P1 合并后看持续集成的结果。
- S3 在原型中因为不是 git 仓库而失败（见上），在仓库中照常；P1 的 `make e2e` 在仓库中运行一遍即可确认。
- `pg_dump` 快照在持续集成中的上传：`e2e/test-results/` 已在上传路径中，未在持续集成中实测。
