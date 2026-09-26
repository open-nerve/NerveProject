# M2/P3a 账户接口：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M2/P3a `account-api` |
| 日期 | 2026-09-26 |
| 状态 | 进行中 |
| 上级文档 | [M2 设计文档](../M2-design.md) 第 2（A7–A11）、3.5、3.10–3.12、3.14、3.19、3.20（P3a 各行）、4.1、4.2–4.4、4.6、5、6.2–6.5、8.2、8.6、8.7、9、12（P3a）、13.1 节；[v0 总体设计](../../v0-design.md) 3.1、3.4、3.6、4.2、6.2 节；[M0 设计](../../M0-foundation/M0-design.md) 3.2、3.7 节；[M0/P3 spec](../../M0-foundation/specs/P3-api-contract.md) 2.8、第 7 节 |
| 前置交接 | [M0-P3-api-codegen-notes](../handoffs/M0-P3-api-codegen-notes.md)、[M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md)、[M1-P2-trim-content](../handoffs/M1-P2-trim-content.md)、[M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md)；[P2 评审记录](../reviews/P2-sessions-review.md) 第 6 节交给 P3 的两项（本 Phase 的处理见第 7 节） |
| 计划 | [P3a plan](../plans/P3a-account-api.md) |

本 spec 只写 M2 设计交给 P3a 决定的东西：名字、签名、SQL、配置项、测试名，以及原型证明了什么。规则本身以 M2 设计为准，这里引用节号，不重述。P3 按负责人的裁定（2026-09-26）拆成 P3a、P3b 两段，P3b 依赖 P3a（M2 设计 12 节）。

## 1. 目标

按 M2 设计 12 节 P3a 的目标：账户的其余接口都可用，而且都能用 PAT 完成；凭证的签发与变更在并发下仍然正确。具体是：

- `identity`：PAT 的创建、分页列表、撤销和认证（`last_used`）；`updateMe`、`getProfile`、`updateProfile`（`onboarding_step` 在 SQL 中合并）；`changePassword`（账户行锁、3.5 的撤销规则、`password_user` 桶）；`deactivateMe`（账户行锁）；共用的 `CredentialLock`；
- `instance`：实例配置的三个字段和 `listTimezones`；`cmd/nerve` 嵌入 `time/tzdata`；
- 平台：`shared` 的页游标封套；参数绑定的 400 写出参数名；`common.yaml` 的分页组件；`apitest` 的参数用例和"每一种问题一起"的请求体用例；uuid 守卫的改写（3.12）；
- 配置：`password_user` 桶、`workspace.creation_enabled`、`files.size_limit`；
- 测试：第五个整程序测试（参数绑定）；每个配置的桶都有整程序测试；3.5 的交错测试 4、5 在真实数据库上；A7–A11 的接口版本和 S3；
- 3.20 中 P3a 的各行、8.7 中 P3a 的两行、交接的处理记录。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由命令生成并提交，不手改。"Task"是 plan 中负责它的任务；几个 Task 号表示先写过渡版本、最后一个 Task 写成最终版本。

| 路径 | 内容 | Task |
|---|---|---|
| `server/internal/platform/config/`、`server/configs/` | `password_user`、`workspace.creation_enabled`、`files.size_limit` | 1 |
| `server/internal/platform/httpserver/apierrors.go`、`apierrors_test.go`、`contract_test.go` | 参数绑定的 400 写出参数名 | 2 |
| `server/internal/platform/httpserver/apitest/` | 请求体用例"每一种问题一起"；参数用例；两条写法规则 | 2、7 |
| `server/tools/bodyshapegen/` | 3.1 的 `contentMediaType`、`contentEncoding` | 2 |
| `server/internal/archtest/` | 按边判断的禁用；生成代码的 uuid 规则；嵌入时区数据库 | 3、12 |
| `server/internal/shared/cursor.go`、`error.go`、`actor.go` | 页游标的封套；`CodeBadRequest`；`Actor.APITokenID` | 4、6 |
| `server/internal/modules/identity/domain/` | PAT、令牌规格、页大小、令牌列表的游标；资料和偏好的补丁检查、`contains_url`；两个模块错误；撤销的原因 | 4、6、8、10、11 |
| `server/migrations/sql/00004_identity_api_tokens.sql`、`schema_test.go`、`server/sqlc.yaml` | `api_tokens` 表 | 5 |
| `server/internal/modules/identity/adapter/postgres/`（含 `queries/`） | 令牌、资料、偏好、密码、停用的查询和存储 | 5、8、10、11 |
| `server/internal/modules/identity/adapter/postgres/gen/` | 生成 | 5、8、10、11 |
| `server/internal/modules/identity/app/` | 端口；PAT 认证；`CredentialLock`；八个用例 | 5、6、8–11 |
| `server/internal/modules/identity/adapter/authn/` | PAT 的限流键 | 6 |
| `api/common.yaml`、`api/openapi.yaml`、`api/modules/identity.yaml`、`api/modules/instance.yaml` | 分页组件；九个新操作；实例配置的三个字段 | 7、9–12 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`*/adapter/http/gen/*.gen.go`、`apigen/components.gen.go` | 生成 | 7、9–12 |
| `server/go.mod`、`server/go.sum` | `oapi-codegen/runtime` v1.7.0 | 7 |
| `server/internal/modules/identity/adapter/http/`、`module.go` | 八个新操作、`password_user` 桶、接线 | 6、7、9–11 |
| `server/internal/modules/instance/`、`server/cmd/nerve/main.go` | 实例配置、时区列表；`time/tzdata` | 12 |
| `server/internal/bootstrap/` | `password_user` 的接线、instance 的 `Deps`；PAT、参数、桶、时区的整程序测试 | 7、9–12 |
| `server/internal/modules/identity/interleavings_test.go` | 交错测试 4、5 | 13 |
| `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`、`e2e/stories/identity/a{7,8,9,10,11}-*.spec.ts`、`e2e/stories/smoke/s3-instance-info.spec.ts` | PAT 的 fixture；五个故事的接口版本；S3 | 12、14 |
| `docs/…`、`README.md` | 3.20 的 P3a 各行、8.7 的 P3a 两行、交接的处理记录 | 15 |

### 2.2 依赖

只加 `github.com/oapi-codegen/runtime` v1.7.0（M2 设计 3.12、6.6）：第一批带参数的操作（`listApiTokens`、`revokeApiToken`）的生成代码导入它。`go -C server get github.com/oapi-codegen/runtime@v1.7.0` 加 `go -C server mod tidy` 另带进间接依赖 `github.com/apapsch/go-jsonmerge/v2` v2.0.0；`github.com/google/uuid` v1.6.0 本来就是间接依赖。`server/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`；`server/tools/go.mod` 不变。不加 npm 包。

### 2.3 配置（M2 设计 3.10、5.3、6.5）

| 键 | 默认值 | 校验 |
|---|---|---|
| `ratelimit.password_user` | `{per_minute: 5, burst: 5}` | 两项都至少 1 |
| `workspace.creation_enabled` | `true` | — |
| `files.size_limit` | `5242880`（字节） | 至少 1 |

- `config.test.yaml` 把 `password_user` 调到 `{per_minute: 600000, burst: 100000}`，与其余六个桶相同。
- `Config.LogValue` 加上三项。`validConfig`（测试的基准配置）的三组新值都不同于默认值、也不同于别的键，日志从错的字段取值时测试失败（附录 A 的变异核对）。
- `auth.session_cleanup_interval`、`jobs.shutdown_timeout` 属于 P3b。
- 测试：`TestLoadAppliesLayersInOrder`（YAML 和三个环境变量）；`TestValidateReportsEveryInvalidKey`、`TestValidateCrossKeyRules`（负的上传上限）；`TestLogValueMasksDatabaseURL`；`TestBuiltInProfiles`（三个环境）。

### 2.4 平台：参数绑定的 400、请求体用例、写法规则（M2 设计 3.11、3.12）

**`APIErrors.BadRequest(w, r, err)`**（生成代码的 `ErrorHandlerFunc`）：

- 400 `bad_request`，`detail` 固定为 "The request parameters do not match the API description."；绑定错误的原文带着 Go 的函数名和类型名，只进 DEBUG "request parameters not bound"（`request_id`、`error`）。
- `errors` 里是参数：`parameterOf(err)` 读绑定错误的字符串字段 `ParamName`（第 3 节第 3 条）。类型名以 `Required` 开头时 `code` 为 `required`（`message` "is required"），否则 `invalid_format`（"has the wrong type or format"）；读不出时没有 `errors`。
- 测试：`TestAPIErrorsBadRequestNamesTheParameter`（7 个：格式不对、缺少查询参数、缺少响应头参数、值太多、不是绑定错误、`ParamName` 不是字符串、不是结构）；测试里的四个错误类型是 identity `server.gen.go` 中绑定错误的仿制品。真实生成代码的错误由整程序测试 5 覆盖（2.17）。

**`apitest.Operation.BodyCases()` 的第 7 种**（第 3 节第 1 条）："every problem at once" 由 schema 有的每一种问题组成：顶层未声明的属性、嵌套对象里未声明的属性、非空可选属性为 `null`、缺少的必填属性、写错的格式，每一种取一个别的种类没用过的属性；只有第一种时不生成。测试：`TestBodyCases`、`TestBodyCasesCombineEveryKindTheSchemaHas`（4 个）。

**`apitest.Operation.ParamCases()`**（Task 7）：每个路径或查询参数中，Go 类型会拒绝某些字符串的（整数、数、布尔、生成为 Go 类型的格式）得到一个用例：示例目标里只有这个参数写错（`not-a-number`、`not-a-boolean`、`not-a-uuid`），期望的 `field` 是参数名。枚举和自由字符串什么都能绑定，不生成。`Target()` 只填必填的查询参数；用例中写错的那个总会填上。测试：`TestParamCases`、`TestParamCasesOfAnOptionalParameter`。

**写法规则**（`apitest` 的规则测试）：参数写 `schema`，不写 `content`；`application/json` 的请求体 schema 是对象。两者都是整程序测试能从接口描述推出用例的前提。测试：`TestAuthoringRulesReportViolations` 的两个违规例子。

**`bodyshapegen`**：OpenAPI 3.1 的字符串没有 `format` 时，oapi-codegen v2.8.0 把带 `contentMediaType` 的按 `binary`、`contentEncoding: base64` 的按 `byte` 生成（`pkg/codegen/schema.go:1294-1330`，附录 A 的 E1）。生成器照同样的规则查格式，生成为没有检查器的类型时失败并说明原因。测试：`TestGenerateRejectsWhatItCannotCheck` 两个例子；`TestGenerateAcceptsTheStringsItCanCheck`（`base64url`；带 `format` 时不看另外两个关键字）。

### 2.5 archtest：google/uuid 与生成代码（M2 设计 3.12）

- `bannedImports(g, root, isBanned func(importer, path string) bool)` 按导入的边判断；报出的链以被禁的包结尾。
- `isBannedFromBinary(importer, path)`：`github.com/google/uuid` 由 `github.com/oapi-codegen/runtime` 模块的包导入时放行；别的导入者照旧报出，每一处各报一次。
- `TestGeneratedCodeUsesTheStandardUUID`：`internal/platform/httpserver/apigen/*.gen.go` 和 `internal/modules/*/adapter/http/gen/*.gen.go`（两个 glob 都必须有匹配）不得引用 `github.com/oapi-codegen/runtime/types` 的 `UUID`，不论导入时用什么名字（第 3 节第 6 条）。
- 测试：`TestBannedImports`（放行 runtime 和 runtime/types，仍报出名字相近的 `runtime-extra` 和平台包的导入）；`TestRuntimeUUIDUses`（6 个）。

### 2.6 页游标的封套（M2 设计 3.12）

```go
func EncodeCursor(payload any) (string, error)      // {"v":1,"p":…} 的无补位 base64url
func DecodeCursor(cursor string, payload any) error // 只认 EncodeCursor 写出的形式
func InvalidCursor() *Error                         // 400 bad_request，errors[{cursor, invalid_format}]
```

- `DecodeCursor` 拒绝：不是无补位 base64url；不是 JSON 对象；封套有别的成员；后面还有第二个值；版本不是 1 或没有版本；没有载荷或载荷为 `null`；载荷解不进 `payload`。
- `shared.CodeBadRequest` 加入 `shared` 的平台码。
- 测试：`TestCursorRoundTrip`；`TestDecodeCursorRejects`（13 个）。

### 2.7 PAT 与令牌规格（M2 设计 3.4、4.4、4.6）

- `domain.PATPrefix = "nrv_pat_"`；`domain.PAT [32]byte`；`String()` 是前缀加 43 个字符的无补位 base64url；`Hash()` 是**整个令牌**的 SHA-256；`ParsePAT(s) (PAT, bool)`：恰好 51 个字符、前缀正确、`base64.RawURLEncoding.Strict()` 解出 32 字节，不查库。
- `domain.APITokenSpec{Label *string; Description string; ExpiredAt *time.Time}`；`CheckAPIToken(spec, now) (APITokenSpec, error)`：标签 1–255 个字符（按字符计）；标签和说明不含 NUL（第 3 节第 9 条）；`expired_at` 晚于 `now`（`must_be_future`）；一次报出全部问题。返回的规格中 `expired_at` 转为 UTC、截到微秒（数据库存的精度），检查、插入和 201 都用它（收尾修复：原来 201 照原样回显带偏移或不足微秒的时间，与列表读回的不同）。
- `domain.PageSize(limit *int)`：`nil` 是 50；1–100 之外 422 `limit` `out_of_range`（第 3 节第 8 条）。
- `domain.APITokenCursor{CreatedAt; ID}`，JSON 是 `[created_at, id]`（RFC 3339 带微秒、uuid）；`created_at` 一律写成 UTC，每个位置只有一种写法（收尾修复）。
- 测试：`TestPATRoundTrip`；`TestParsePATRejects`（11 个）；`TestCheckAPITokenAcceptsAValidSpec`；`TestCheckAPITokenNormalizesTheExpiry`；`TestCheckAPITokenReportsEveryField`（7 个）；`TestPageSize`；`TestAPITokenCursorRoundTrip`；`TestAPITokenCursorRejects`（8 种载荷）。

### 2.8 `api_tokens` 表与查询（M2 设计 3.5、3.13、3.14、4.4）

迁移 `00004_identity_api_tokens.sql`：Plane 17 列留 12 列；`token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32)`；`label varchar(255) NOT NULL CHECK (label <> '')`；`description text NOT NULL DEFAULT ''`；`user_id` `ON DELETE CASCADE`；`created_by_id`、`updated_by_id` `ON DELETE SET NULL`；索引 `api_tokens_user_id_created_at_idx ON (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`。

```sql
-- name: ListAPITokens :many
SELECT id, label, description, expired_at, last_used, created_at
FROM api_tokens
WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL
  AND (sqlc.narg(cursor_created_at)::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: RevokeAPIToken :execrows
UPDATE api_tokens
SET updated_at = sqlc.arg(now), deleted_at = sqlc.arg(now), updated_by_id = sqlc.arg(user_id)::uuid
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)::uuid AND deleted_at IS NULL;

-- name: TouchAPIToken :exec
UPDATE api_tokens
SET last_used = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND (last_used IS NULL OR last_used < sqlc.arg(stale_before)::timestamptz);
```

- 另有 `CreateAPIToken`（创建者和更新者都是账户本身）、`GetAPITokenByHash`、`GetAPITokenByID`（都联 `users` 取 `is_active`）。
- `TouchAPIToken` 不改 `updated_at`（第 3 节第 17 条）。
- 端口：`NewAPIToken`、`APITokenCreator`、`APITokenLister`、`APITokenRevoker`、`APITokenCredential{ID, UserID, ExpiredAt, LastUsed, Revoked, UserActive}`、`APITokenReader{APITokenByHash; APITokenByID}`、`APITokenToucher`。
- 测试（真实数据库）：`TestCreateAPIToken`；`TestListAPITokensPageByPage`（5 个令牌中 3 个同一时刻，每页 2 个，翻完不重不漏，顺序与独立算出的相同；已撤销和别的账户的不列出）；`TestListAPITokensReadsEveryField`；`TestRevokeAPIToken`；`TestAPITokenCredentials`；`TestTouchAPIToken`。`schema_test.go`：四个迁移 up、down、再 up；约束和索引的清单；两个 CHECK 的反例。

### 2.9 PAT 认证与账户行锁的复核（M2 设计 3.5、3.10）

- `shared.Actor` 加上 `APITokenID`：`SessionID`、`APITokenID` 恰好一个有值。
- `app.NewAuthenticate(AuthenticateDeps{AccessTokens, Sessions, APITokens, Touch, Clock, Logger})`：`nrv_pat_` 开头的令牌走 PAT：`ParsePAT` 失败不查库；按哈希读一次；不存在、已撤销、`now >= expired_at`、账户停用都是 401（原因只进 DEBUG，P1 的做法）；`last_used` 为空或早于 `now - 1 分钟` 时写一次；读失败是内部错误。写 `last_used` 失败只记 WARN（`token_id` 和错误，不含令牌），认证照常通过：M2 设计 3.6 说 `last_used` 是尽力而为，收尾修复按设计改（原来写失败也是内部错误）。别的令牌照旧走访问令牌。
- `sessionInvalid`、`tokenInvalid`：认证和锁共用的判定，一处写。
- **`app.CredentialLock{Locker, Sessions, APITokens}`**（`credential_lock.go`，P2 spec 第 3 节第 7 条留给 P3 的共用部分）：

  ```go
  func (c CredentialLock) Lock(ctx context.Context, actor shared.Actor, now time.Time) (LockedAccount, error)
  ```

  在事务里先 `LockForCredentials`（`FOR NO KEY UPDATE`），再在锁下按 actor 的凭证种类复核：会话未撤销、未到期、属于这个账户；或 PAT 未撤销、未到期、属于这个账户；账户未停用。不成立是 401，返回锁住的行（哈希快照供修改密码用）。创建 PAT、修改密码、停用三个用例都经过它。
- `authn`：PAT 的限流键是 `pat:<id>`（第 3 节第 18 条）。
- 测试：`TestAuthenticateAPAT`；`TestAuthenticateAPATTouchesAtMostOnceAMinute`（从未用过、59 秒、恰好 1 分钟、61 秒）；`TestAuthenticateRejectsAPAT`（6 个）；`TestAuthenticateAPATDatabaseFailureIsNot401`；`TestAuthenticateAPATWhoseLastUsedIsNotWritten`；`TestAuthenticateKeysAPATByItsID`。锁的复核由三个用例的测试覆盖（`…RechecksTheCredentialUnderTheLock`）。

### 2.10 令牌的三个用例（M2 设计 3.5、3.12、4.6、5.4）

- `NewCreateAPIToken(CreateAPITokenDeps{Lock, Tokens, Tx, Clock, Logger})`：`CheckAPIToken` → 一个事务：`Lock` → 插入。没有标签时取一个 v4 uuid 的 32 位十六进制（Plane 的 `uuid4().hex`）。令牌 32 字节来自 `crypto/rand`；返回值是令牌唯一出现的地方。INFO "API token created"（`user_id`、`token_id`）。
- `NewListAPITokens(tokens)`：先判游标（400），再判 `limit`（422，第 3 节第 7 条）；读 `limit + 1` 行判断有没有下一页，`NextCursor` 是本页最后一行。
- `NewRevokeAPIToken(tokens, clock, logger)`：一条语句（M2 设计 6.4"单条语句的写入"）；不存在、已撤销、别的账户的令牌都是 404 `identity.api_token_not_found`；请求所用的令牌可以撤销自己。INFO "API token revoked"。
- 测试：`TestCreateAPIToken`（调用顺序；插入的行；日志中没有令牌和哈希的任何写法，含大写十六进制）；`TestCreateAPITokenGeneratesALabel`；`TestCreateAPITokenWithAToken`；`TestCreateAPITokenRechecksTheCredentialUnderTheLock`（9 个：账户不存在、停用；会话撤销、到期、不存在、属于别的账户；令牌撤销、到期、属于别的账户）；`TestCreateAPITokenLockFailureIsNot401`；`TestCreateAPITokenChecksTheSpecFirst`；`TestListAPITokensFirstPage`、`…NextPage`、`…LastFullPage`、`…Rejects`（4 个）；`TestRevokeAPIToken`、`TestRevokeAnAPITokenThatIsNotTheCallers`、`TestRevokeAPITokenDatabaseFailure`。

### 2.11 令牌的接口（M2 设计 3.11、3.12、5.1、5.2）

| 操作 | `x-problem-codes` | 成功 |
|---|---|---|
| `listApiTokens`：`GET /api/v0/me/api-tokens`（`Limit`、`Cursor`） | `validation_failed` | 200 `ApiTokenPage{data, next_cursor}` |
| `createApiToken`：`POST /api/v0/me/api-tokens` | `validation_failed` | 201 `ApiTokenCreated` |
| `revokeApiToken`：`DELETE /api/v0/api-tokens/{token_id}` | `identity.api_token_not_found` | 204 |

- `common.yaml`：参数 `Limit`（`integer`，1–100，默认 50）、`Cursor`（`string`）；schema `NextCursor`（`[string, 'null']`）。M0/P3 原定由 M3 加入（3.20 的一行）。
- `ApiToken` 的 `expired_at`、`last_used` 必有、可为 `null`；`ApiTokenCreate` 三个字段都可省略，`expired_at` 可为 `null`（第一批带格式的请求体字段，3.11）；`ApiTokenCreated` 单独写出全部字段加 `token`（5.2 不用 `allOf` 的理由）。
- 适配器 `api_tokens.go`：`nullableTime` 把 `nil` 显式写成 `null`（零值的 `Nullable` 表示"未指定"，会编码成空串）。
- 测试：`api_tokens_test.go` 的 9 个，其中 `TestParametersThatDoNotBind` 对 `limit=abc`、`token_id=not-a-uuid` 逐字核对 400 的响应体，用例没有被调用；`apitest.Main` 核对 `identity.api_token_not_found` 由 handler 测试答过。

### 2.12 资料与偏好的规则和存储（M2 设计 3.11、3.14、4.2、4.3）

- `domain.UserPatch{FirstName, LastName, DisplayName, Timezone *string}`、`CheckUserPatch(p)`：名、姓至多 255 个字符、不含网址（`contains_url`）；显示名 1–255 个字符；都不含 NUL；时区是 Go 的 `time` 包能加载的名字，不是 `Local`、不是空串（第 3 节第 10 条）；一次报出全部问题。没有 `email`：接口描述里不声明它，传了是 400 `not_allowed`（决策点 1）。
- `domain.containsURL`：Plane `plane/apps/api/plane/utils/url.py:12-53` 的移植。RE2 没有 Python 的 `\s` 和 `re.IGNORECASE` 的字母范围，按 Python 的语义写出（`pySpace`、`pyLetters`）；超过 1000 个字符的文本不看，每行只看前 500 个字符，都按字符计。`TestContainsURLAsPlane` 的 41 个期望值是 Plane 的函数在 Python 3 中对同一字符串的答案（附录 A 的 E5）。
- `domain.ProfilePatch{Theme, Language *string; StartOfTheWeek *int; OnboardingStep OnboardingStepsPatch; IsOnboarded, IsTourCompleted *bool; LastWorkspaceSet bool; LastWorkspaceID *uuid.UUID}`、`CheckProfilePatch(p)`：主题是五个之一、语言是 `en` 或 `zh-CN`（`invalid_format`），每周第一天 0–6（`out_of_range`）。结构（步骤的键和布尔类型）在边界上由 bodyshape 检查（3.11）；bodyshape 不检查枚举，所以主题和语言由领域检查。
- `OnboardingStepsPatch.JSON()` 只含设了的步骤，没有时是 `{}`。
- 查询（部分更新按 3.14 的 `CASE WHEN`，`updated_at` 放在 `SET` 的第一位）：

  ```sql
  -- name: UpdateProfile :one
  UPDATE profiles
  SET updated_at        = sqlc.arg(now),
      theme             = CASE WHEN sqlc.arg(set_theme)::boolean THEN sqlc.arg(theme)::text ELSE theme END,
      …
      onboarding_step   = onboarding_step || sqlc.arg(onboarding_step_patch)::jsonb,
      …
      last_workspace_id = CASE WHEN sqlc.arg(set_last_workspace_id)::boolean THEN sqlc.narg(last_workspace_id)::uuid ELSE last_workspace_id END
  WHERE user_id = sqlc.arg(user_id)
  RETURNING theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at;
  ```

  `UpdateUser` 同样写法，`RETURNING` 账户；`GetProfile`。空的补丁也更新 `updated_at`（第 3 节第 17 条）。
- 端口：`UserUpdater`、`ProfileReader`、`ProfileUpdater`；`Store.UpdateUser`、`GetProfile`、`UpdateProfile`（没有这一行是 `app.ErrNotFound`）。
- 测试：`TestCheckUserPatchAcceptsAValidPatch`、`…ReportsEveryField`（10 个）；`TestContainsURLAsPlane`（41 个）；`TestCheckProfilePatchAcceptsAValidPatch`、`…ReportsEveryField`；`TestOnboardingStepsPatchJSON`；真实数据库的 `TestUpdateUser`、`TestUpdateUnknownUser`、`TestGetProfileReadsTheDefaults`、`TestUpdateProfile`、`TestUpdateUnknownProfile`，以及 **`TestUpdateProfileMergesConcurrentSteps`**（M2 设计 3.14 的并发合并：第一个事务改 `profile_complete` 后持锁；等到第二条语句确实在等锁（`pg_stat_activity` 的 `wait_event_type = 'Lock'`）才放开；两个键都保留；每个等待有 10 秒的期限）。

### 2.13 资料和偏好的用例与接口（M2 设计 5.1、5.2）

| 操作 | `x-problem-codes` | 成功 |
|---|---|---|
| `updateMe`：`PATCH /api/v0/me` | `validation_failed` | 200 `User` |
| `getProfile`：`GET /api/v0/me/profile` | `[]` | 200 `Profile` |
| `updateProfile`：`PATCH /api/v0/me/profile` | `validation_failed` | 200 `Profile` |

- `NewUpdateMe(users, clock)`、`NewGetProfile(profiles)`、`NewUpdateProfile(profiles, clock)`：先检查补丁（422），再一条语句（不开事务）；账户或资料在认证之后已不存在是 401。
- 结构：`UserUpdate`（四个可选字符串，都不能为 `null`）；`Theme`、`Language`、`StartOfTheWeek`（0–6 的整数枚举）；`OnboardingSteps`（四个必有的布尔值）、`OnboardingStepsUpdate`（四个可选的布尔值）；`Profile`（全部必有，`last_workspace_id` 是 `[string, 'null']` 的 `uuid`）；`ProfileUpdate`（全部可省略，只有 `last_workspace_id` 可为 `null`）。
- 适配器：`me.go` 的 `GetMe`、`UpdateMe` 共用 `user(domain.User) gen.User`；`profile.go` 的 `profile(domain.Profile) gen.Profile`；`last_workspace_id` 用 `IsSpecified`、`Get` 区分"没传"和 `null`。
- 测试：`app/account_test.go` 的 5 个（`TestAccountPatchesAreChecked` 证明不合规时什么都不写；`TestAccountUseCasesErrors` 对三个用例各测没有 actor、账户已不存在、数据库错误）；`me_test.go` 的 `TestUpdateMe`、`TestUpdateMeProblems`；`profile_test.go` 的 `TestGetProfile`、`TestUpdateProfile`、`TestUpdateProfileProblems`（不认识的步骤 `onboarding_step.profile_completed` 400 `not_allowed`，用例没有运行）。

### 2.14 修改密码（M2 设计 3.5、3.8、3.10、5.4）

`POST /api/v0/me/change-password`（`changePassword`，204，`x-problem-codes: [validation_failed, identity.current_password_incorrect, server_busy]`），`ChangePasswordRequest{current_password, new_password}` 都必填。

`ChangePassword.Execute(ctx, ChangePasswordInput{Current, New})`（`ChangePasswordDeps{Accounts, Lock, Passwords, Sessions, Hasher, Rules, Tx, Clock, Logger}`）：

1. `PasswordAccount`（邮箱和哈希；已不存在是 401）；新密码按密码规则检查（含邮箱词干，3.8），不合规是 422 `new_password`，此时没有做任何 argon2（第 3 节第 13 条）；
2. 事务外对快照校验当前密码，新密码只哈希一次；
3. 事务：`Lock`（复核凭证）→ 锁下的哈希不等于快照时什么都不写 → `UpdatePasswordHash` → `RevokeSessions(userID, actor.SessionID, password_changed)`：用 PAT 修改时 `SessionID` 是零值，全部会话都撤销；PAT 不变（3.5 的撤销表）；
4. 哈希变了：对新哈希再校验一次，成立就再做一次第 3 步（`for range 2`）；不成立或又变了是 422 `identity.current_password_incorrect`（没有 `errors`，第 3 节第 2 条）；
5. 成功记 INFO "password changed"（`user_id`）。

```sql
-- name: RevokeSessions :execrows
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = sqlc.arg(reason)::text
WHERE user_id = sqlc.arg(user_id) AND id <> sqlc.arg(keep)
  AND revoked_at IS NULL AND expires_at > sqlc.arg(now);
```

- `RevokeSessions` 只撤销未撤销、未到期的会话：已撤销的保留原来的原因，已到期的留给清理任务（第 3 节第 14 条）。`keep` 为零值时一个都不保留。
- `domain.RevokeReason`（`password_changed`，Task 11 加上 `deactivated`）；`domain.ErrCurrentPasswordIncorrect`（422，"The current password is incorrect."）。
- **`password_user` 桶**：`httpadapter.Limits.PasswordUser`；`limitPassword` 在调用用例之前按账户 id 扣一个单位（3.10：键是认证过的账户）；模块入口 `RateLimits.PasswordUser`；`bootstrap` 建第七个桶。注册和修改密码共用一个 `domain.NewPasswordRules()`。
- 测试：`change_password_test.go` 的 7 个（`TestChangePassword`、`…WithATokenRevokesEverySession`、`…ChecksTheNewPasswordFirst`（3 个）、`…WithAWrongCurrentPassword`（不哈希、不开事务）、`…AfterAConcurrentChange`（3 个：并发登录重新哈希了 → 成功；密码被改了 → 422；变了两次 → 422；新密码都只哈希一次）、`…RechecksTheCredentialUnderTheLock`、`…Errors`（5 个））；`credentials_test.go` 的 `TestPasswordAccount`、`TestRevokeSessions`；`me_test.go` 的 `TestChangePassword`、`TestChangePasswordProblems`；`limits_test.go` 的 `TestChangePasswordLimitsByAccount`（`password_user` 为 2：同一账户第三次被拒，换了 IP 也一样；别的账户不受影响）。

### 2.15 停用账户（M2 设计 3.5、4.3，决策点 3）

`POST /api/v0/me/deactivate`（`deactivateMe`，没有请求体，204，`x-problem-codes: []`）。

- `NewDeactivate(DeactivateDeps{Lock, Users, Profiles, Sessions, Tx, Clock, Logger})`：一个事务，按全局加锁顺序（`users` → `profiles` → `auth_sessions`）：`Lock`（复核凭证）→ `DeactivateUser`（`is_active = false`，密码不变）→ `ResetOnboarding` → `RevokeSessions(userID, 零值, deactivated)`。PAT 不删除；账户停用期间认证拒绝它们。INFO "account deactivated"（`user_id`、`revoked_sessions`、`by: self`，第 3 节第 15 条）。
- `ResetOnboarding`：`onboarding_step`、`is_onboarded`、`is_tour_completed`、`last_workspace_id` 回到列的 `DEFAULT`（就是注册时的值），别的偏好不变。
- 端口：`UserDeactivator`、`OnboardingResetter`。
- 测试：`TestDeactivate`（会话、PAT 各一次：调用顺序、一个事务、三次写入的时刻、日志）；`TestDeactivateRechecksTheCredentialUnderTheLock`；`TestDeactivateWithoutAnActor`；真实数据库的 `TestDeactivateUserAndResetOnboarding`（两个账户，另一个完全不变）；`TestDeactivateMe`。A12 的端到端要用 `nerve users activate`，在 P3b（第 3 节第 24 条）。

### 2.16 实例配置与时区（M2 设计 3.19、4.2、5.3）

- `InstanceInfo` 加上必有的 `signup_enabled`、`workspace_creation_enabled`（布尔）、`file_size_limit`（整数，至少 1）；不定义 `is_self_managed`。
- `listTimezones`：`GET /api/v0/timezones`，公开，`x-problem-codes: []`，200 `TimezoneList{data: Timezone[]}`，`Timezone{label, value, utc_offset, gmt_offset}`（`UTC+08:00`、`GMT+08:00`）。说明写明 `user_timezone` 接受任何 IANA 名字，不只这些。
- `domain.Settings{SignupEnabled, WorkspaceCreationEnabled, FileSizeLimit}` 嵌在 `Info` 里；`domain.LoadTimezones()` 加载 Plane 的 120 个地点（`plane/apps/api/plane/app/views/timezone/base.py:30-178`）的时区，`Timezones.At(t)` 的偏移量按 `t` 时的时区数据计算，按偏移、再按标签排序（`base.py:209`），写成 `±hh:mm`（第 3 节第 11 条）。收尾修复前是每次请求都加载 120 个时区的 `domain.Timezones(t)`；现在只在构建模块时加载一次，加载不了的时区让启动失败，请求时没有错误的路径。
- `app.Clock`；`NewGetInfo(source, settings)`；`NewListTimezones(zones, clock)`：偏移按用例的时钟算。
- `instance.New(instance.Deps{SignupEnabled, WorkspaceCreationEnabled, FileSizeLimit, Clock}) (*Module, error)`（第 3 节第 23 条）。
- `cmd/nerve/main.go` 导入 `time/tzdata`；`TestNerveBinaryEmbedsTheTimeZoneDatabase` 守住（第 3 节第 12 条）。
- 测试：`TestTimezonesAreSorted`（120 个，第一个 American Samoa −11:00，最后一个 Kiritimati +14:00）；`TestTimezoneOffsets`（9 个地点在一月和七月：Marquesas −09:30、Newfoundland −03:30/−02:30、Kathmandu +05:45、Chatham +13:45/+12:45 等）；`TestLoadTimezonesFailsOnAnUnknownZone`；`TestGetInfoDescribesTheBuildAndTheSettings`；`TestListTimezonesAtTheClocksNow`；`TestGetInstanceMatchesTheContract`；`TestListTimezones`。

### 2.17 整程序测试（M2 设计 3.6、3.10、3.11，12 节 P3a 完成线）

- **5. `TestParametersThatDoNotBindAnswer400`**（`bootstrap/contract_test.go`，第 3 节第 5 条）：对契约中每个操作的每个 `ParamCase`，不带令牌、数据库不可达，发给真实组合出来的程序：400 `bad_request`，`errors` 恰好是 `[{field: <参数>, code: invalid_format}]`，并对响应 `CheckResponse`。参数在认证之前绑定（3.6），所以不需要令牌；推不出任何用例时失败（推导坏了不能当作"没什么可测"）。
- **`TestEachConfiguredBucketLimitsItsOperations`**（`bootstrap/limits_test.go`，第 3 节第 19 条）：七个桶的突发各不相同（`anonymous` 9、`authenticated` 8、`auth_failure` 3、`login_ip` 5、`login_ip_email` 2、`register_ip` 1、`password_user` 4），每分钟 1 个；可信代理是测试自己的对端（`127.0.0.0/8`），每个用例用自己的 `X-Forwarded-For`，按 IP 的桶互不干扰；每个操作恰好在自己桶的突发之后得到 429。
- **`TestAPersonalAccessTokenAuthenticates`**（`auth_test.go`）：访问令牌建第一个 PAT，第一个 PAT 建第二个；第一个读 `/me` 200、撤销自己 204、再读 401；第二个 200。
- **`account_test.go`**（真实数据库，只用 PAT）：`TestTheAccountAndItsPreferencesWithAPersonalAccessToken`；`TestChangingThePasswordWithAPersonalAccessToken`；`TestDeactivatingWithAPersonalAccessToken`。
- **`api_test.go`**：`TestServesTheInstanceAPI` 用不同于默认的设置（`true`、`false`、7340032）核对三个字段来自配置；`TestServesTheTimezones`。
- P1 的四个整程序测试自动覆盖九个新操作：公开集合等于契约（加上 `listTimezones`）、路由等于契约、默认拒绝；第 4 个对 `createApiToken`、`updateMe`、`updateProfile`、`changePassword` 逐项改坏请求体，其中 `createApiToken` 的 `expired_at`（`date-time`）和 `updateProfile` 的 `last_workspace_id`（`uuid`）是逐格式的第一批用例，`updateProfile` 的第 7 种用例一次带着五种问题。

### 2.18 账户行锁的交错测试 4、5（M2 设计 3.5、9.2）

`server/internal/modules/identity/interleavings_test.go`（包 `identity_test`，第 3 节第 16 条），真实数据库、真实的存储、事务和签名：

- `gatedHasher`：`Hash` 给出 `hashed:<密码>:<盐>`；`Verify` 认任何盐的 `hashed:`，把 `old:<密码>` 当作旧参数的哈希并要求重新哈希；带闸门时第一次 `Verify` 停在闸门上。每个等待都有 10 秒的期限，超时即失败，不会挂住。
- `TestALoginWithTheOldPasswordFailsWhenThePasswordChangesMeanwhile`（交错 4）：登录校验了旧密码后停住；修改密码提交；登录在锁下看到新哈希，再校验一次失败：401 `identity.invalid_credentials`，没有新会话，存的是新哈希；登录校验过的恰好是旧哈希和新哈希。
- `TestTwoLoginsWhileOneHashesThePasswordAgain`（交错 5）：两个登录都读到旧参数的哈希；第二个重新哈希并先提交；第一个在锁下看到新哈希，再校验一次后成功，不覆盖新哈希：三个会话，存的是第二个的哈希。
- 交错 1–3 要管理员重置密码，在 P3b；交错 6 在 P2（`TestTheCredentialLockBlocksLocksNotInserts`）。

### 2.19 端到端（M2 设计 2、9.5；M0-P6 交接）

- `auth.ts`：`ApiTokenCreated`、`bearer(token)`、`createPAT(api, token, body = {})`。
- `assert/identity.ts`：`AccountRow`、`accountOf`、`tokensOf`、`expectPasswordChanged(db, before, tokensBefore, survivingSession?)`（A7 的共用断言：哈希换成另一个 argon2id；除 `survivingSession` 外的会话都是 `password_changed`；PAT 不变。页面版本传入自己的会话）、`expectTokenStored(db, id, token, expiredAt)`（`token_hash` 是令牌的 SHA-256；`row_to_json` 的整行不含令牌；`expired_at`；未撤销）。
- 故事（接口版本，只用 PAT；页面版本随 P4、P5 加入）：
  - `a7-change-password.spec.ts`：当前密码错误 422，账户不变；正确时 204，全部会话 `password_changed`，PAT 不变仍可用；旧密码登录 401，新密码 200。
  - `a8-update-me.spec.ts`：四个字段；`updated_at` 变大；`{"first_name": null}` 400 `invalid_format`，数据库不变。
  - `a9-preferences.spec.ts`：主题、语言、每周第一天，数据库和读回的相同。
  - `a10-onboarding-profile.spec.ts`：设名；只带一个键的步骤合并进去；拼错的键 400 `not_allowed`，数据库不变。
  - `a11-api-tokens.spec.ts`：一个 PAT 创建另一个；每页 1 个翻完列表；使用写入 `last_used`；撤销后 401、从列表消失；`expired_at` 改到过去的令牌 401。
  - `s3-instance-info.spec.ts`：`toEqual` 加上三个字段的期望值。
- 新增的等待：没有。

### 2.20 文档、README 与交接（M2 设计 3.20、8.7、13.1）

plan 的 Task 15 逐行给出文字。要点：

- **总体设计**：3.1 修改密码和停用答 204；3.4 游标的封套、载荷和两种错误；3.6 `password_user` 的默认值；4.2 修改密码和停用时的会话撤销、账户行锁，`nerve users create` 改标 P3b；6.2 `shared` 有了游标的封套。
- **M0 设计**：3.2 实例配置的三个字段；3.7 传递依赖测试的改写和生成代码的 uuid 规则。
- **M0/P3 spec**：2.8 规则 8 一段的补注；第 7 节 uuid 一行和分页组件一行。
- **差异清单**：二 `api_tokens` 逐列；三 令牌的路径；四 4.6 中标 P3a 的行，以及密码规则、注册默认、限流三行的改写。
- **README** 部署：注册一条补上账户枚举的说明（8.2）；令牌的密钥扫描（8.6）。8.7 中"刷新令牌泄露时可能派生 PAT，以及恢复步骤（8.5）"要写 `reset-password`，随 P3b（第 3 节第 20 条）。
- **交接**：M0-P3 追加"处理结果（M2/P3a）"，状态改为 `done`；M0-P6、M1-P2、M1-P3 追加处理结果，仍为 `open`（第 7 节）。

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M2 设计的架构。

1. **第 7 种请求体用例合并 schema 有的每一种问题**（拆分时的缺口 1，控制者裁定为本地决定）：3.11 的第 7 种要"同一个请求里同时有未知字段、缺少的必填字段和写错的格式"。P3 的请求体没有一个同时有必填字段和带格式的字段（`createApiToken`、`updateProfile` 没有必填字段），照原写法这两个操作得不到多问题的用例。现在由 schema 有的种类组成，有两种以上就生成；`TestBodyCasesCombineEveryKindTheSchemaHas` 覆盖四种形状。
2. **`identity.current_password_incorrect` 没有 `errors`**：M2 设计 5.4 写"`errors[].field = current_password`"，但字段码是封闭的集合（3.11），没有一个表达"不对"。问题码本身已经指明是当前密码；前端按码显示在当前密码的字段下（7.7）。
3. **参数名由平台用反射读出**（拆分时的缺口 3，原型定下）：oapi-codegen 在每个模块的 `gen` 包里声明绑定错误（`InvalidParamFormatError` 等六种），只把参数名放在字符串字段 `ParamName` 里，平台不能导入 `gen`。候选是平台反射读字段，或每个模块的适配器各自翻译；后者是每个模块都要照抄的约定。原型选前者：一处代码，版本锁定的生成形状由整程序测试 5 对真实的生成代码核对（2.17），生成器升级改了形状时那个测试失败。
4. **参数绑定的 400 不带出原文**：3.11 只说"不再带出 Go 的类型名"；绑定错误的原文（如 `strconv.ParseInt: parsing "abc": invalid syntax`）整句都是 Go 的词，所以 `detail` 是固定的一句，原文只进 DEBUG。
5. **第五个整程序测试**：3.11 要求参数绑定的出口"由 P3 第一批带参数的操作测"。handler 测试只测两个已知的参数；整程序测试 5 从接口描述推出每个会拒绝某些字符串的参数，以后的 M 加参数不用记着补测试。它依赖 `apitest` 的两条新写法规则（参数写 `schema`、请求体是对象，2.4）。
6. **生成代码的 uuid 规则也检查 `apigen`**：3.12 写"`adapter/*/gen` 下的生成文件"。`apigen/components.gen.go` 同样由 oapi-codegen 生成（`common.yaml`），漏了映射一样会引用 runtime 的 `UUID`；规则按包路径和名字认出引用（不论导入时的名字），不只认 `openapi_types.UUID` 这一种写法。
7. **游标先于 `limit` 判定；`null` 载荷是 400**：两个都错时答 400（游标是请求的结构，3.11 的分层），不答 422。`{"v":1,"p":null}` 不是任何列表发出的游标，按不合法处理。
8. **页大小的规则放在 `identity/domain`**：1–100、默认 50 是 `common.yaml` 的 `Limit`。M2 只有一个列表，规则放在它的使用方；第二个列表（M3）出现时再移到 `shared`，不留没有第二个使用者的共用代码。
9. **文本字段拒绝 NUL**：标签、说明、名、姓、显示名含 `\u0000` 是 422 `invalid_format`。Postgres 的 `text` 存不下 NUL，不拒绝就是 500。
10. **时区按 Go 能加载的名字判断，宿主的文件系统会影响大小写**：`time.LoadLocation` 先查宿主的时区目录，再查内嵌的数据。macOS 的文件系统不区分大小写，开发机上 `asia/shanghai` 也能通过；Linux 容器里只有准确的写法能通过，有没有宿主的时区文件都一样（附录 A 的 E8）。部署只有 Linux 容器，不另写一份名单；开发机上存进去的非标准写法只影响开发库。
11. **Plane 的时区偏移错误不照搬**：Plane 把负的非整点偏移多算一小时（`base.py:192`，马克萨斯群岛的 −09:30 写成 −10:30）。这里写对，登记为差异（差异清单四"时区"一行）。
12. **`time/tzdata` 在 `cmd/nerve`**：嵌入时区数据是程序的属性，放在程序的入口；`TestNerveBinaryEmbedsTheTimeZoneDatabase` 用依赖图核对入口导入了它，删掉这行时测试失败。只有 `instance/domain` 的测试也导入它（第 6 节）。
13. **修改密码先检查新密码，新密码只哈希一次**：新密码不合规时不做 argon2（任何人都能用一个 PAT 反复提交弱密码，不该占用哈希的名额）；3.5 的重试中快照变了只重做校验和事务，新密码的哈希不变（它与快照无关）。
14. **`RevokeSessions` 跳过已撤销和已到期的会话**：已撤销的保留原来的原因（与 P2 的 `RevokeForReuse` 同理，P2 spec 第 3 节第 16 条），已到期的留给清理任务，撤销的个数只算真正结束的会话。
15. **停用**：`ResetOnboarding` 用列的 `DEFAULT`，与注册时的值由同一处定义；日志带 `by: self`：P3b 的 `deactivate` 命令也会停用账户（3.17），日志要分得出是谁停用的。
16. **交错测试放在模块根目录的测试包 `identity_test`**：它要把真实的存储、事务和签名接到用例上，这是模块入口做的组合；放进 `app` 的测试包就要让它导入适配器。
17. **`last_used` 不改 `updated_at`；空的 PATCH 仍更新 `updated_at`**：使用令牌不改变令牌（Plane 的 `save(update_fields=["last_used"])`）。PATCH 是一次写入，即使什么字段都没传，`updated_at` 也记下它（Plane 的 `save()` 同样更新 `updated_at`）。
18. **PAT 的限流键是 `pat:<id>`**：3.10 的 `authenticated` 按凭证计数；访问令牌是 `session:<id>`（P2），PAT 用自己的 id，同一账户的会话和 PAT 各有自己的额度。
19. **每个配置的桶都有整程序测试**：P2 评审移交（M7）"桶与配置的接线没有测试守住"，这里加入 `password_user` 时一并关闭（2.17）。每个桶的突发各不相同，接错位置的桶会在错误的次数上答 429。
20. **README 中 8.5 的一行随 P3b**：8.7 把"注册会暴露邮箱是否已注册（8.2）"和"刷新令牌泄露时可能派生 PAT，以及恢复步骤（8.5）"写在一行。后者的恢复步骤是 `nerve users reset-password`，P3b 才有；P3a 写前者，后者随命令写。
21. **差异清单登记"相同"的一行**：4.6 的"退出、修改密码、停用后旧凭证失效"在 Plane 中也是下一个请求时失效，结果相同；4.6 把它列为行为差异，这里照登，写明"相同"和做法。
22. **读活的头部映射**（P2 缺陷类别 D）：P3a 改到的两个测试文件（`apierrors_test.go`、`contract_test.go`）里，读响应头的地方都改读 `rec.Result().Header`；改之前，把 `Content-Type`、`Retry-After`、`WWW-Authenticate` 挪到状态写出之后的三个变异都没被发现（附录 A）。P1、P2 的另外六个测试文件有同样的写法，P3a 没有改到它们，交给控制者（第 7 节）。
23. **`instance.New(Deps)`**：模块入口原来没有参数。三个设置和时钟都来自 `bootstrap`，照 identity 的写法合成一个 `Deps`。
24. **A12 的端到端在 P3b**：A12 的接口版本要用 `nerve users deactivate` 和 `activate`。P3a 用整程序测试 `TestDeactivatingWithAPersonalAccessToken` 覆盖自助停用的数据库结果、PAT 失效和登录的 403。

## 4. 验收标准（完成线，M2 设计 12 节 P3a）

- [ ] A7–A11 的接口版本和 S3 通过，P1、P2 的故事仍然通过。
- [ ] 每个需要登录的新操作都有只用 PAT 的测试（`bootstrap` 的 `account_test.go`、`auth_test.go`，A7–A11）。
- [ ] 整程序测试覆盖新加的全部操作：每个带格式的字段写错时 400 `invalid_format`（`createApiToken` 的 `expired_at`、`updateProfile` 的 `last_workspace_id`）；每个会拒绝某些字符串的参数写错时 400（`TestParametersThatDoNotBindAnswer400`）。
- [ ] 3.5 的交错测试 4、5 在真实数据库上通过，`-count=5` 也通过。
- [ ] 带着 `oapi-codegen/runtime` 的 nerve 通过改写后的传递依赖测试，生成代码的 uuid 规则通过。
- [ ] `onboarding_step` 的并发合并测试通过；`limit` 为 0、101（422）、`abc`（400）的测试通过。
- [ ] 3.10 表中的每个桶都有整程序测试（`TestEachConfiguredBucketLimitsItsOperations`）；`password_user` 按账户计数（`TestChangePasswordLimitsByAccount`）。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make e2e` 通过。
- [ ] 3.20 中 P3a 的各行、8.7 中 P3a 的两行在同一次合并中写好；交接按第 7 节处理。

## 5. 不在 P3a 范围内

- River（迁移 `00005`、`platform/jobs`、清理任务、停机顺序）、`nerve users` 的五个命令和 `Admin()`、交错测试 1–3、A12–A14、A16、A17、`auth.session_cleanup_interval`、`jobs.shutdown_timeout`（P3b）。命令行的"只投递"River 客户端推迟到 M4（负责人 2026-09-26 批准，M2 设计 13.2）。
- A7–A11 的页面版本（P4、P5）；前端改用新的实例字段和生成的类型（P4）。
- 时区名单的大小写在不同宿主上一致（第 3 节第 10 条）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 参数名靠反射读生成代码的形状，oapi-codegen 升级后形状变了 | 版本锁定；整程序测试 5 对真实的生成代码核对每个参数的 400 带着参数名，形状一变就失败（第 3 节第 3 条） |
| 并发合并、交错测试依赖真实的时序 | 两者都用确定的闸门或等锁的探测排定顺序，不靠运气；每个等待都有期限；原型中 `-count=5` 全部通过（附录 A） |
| 测试进程用宿主的时区数据 | 持续集成的 ubuntu-24.04 带 tzdata；只有 `instance/domain` 的测试像程序一样嵌入它（它逐个加载 120 个时区）。宿主缺时区文件时，别的包里用到时区的测试（`identity` 的补丁检查、`instance/app`、`bootstrap`）会失败，不会误通过 |
| 修改密码的 `password_user` 额度被同一账户的 PAT 用完 | 这是按账户计数的本意（3.10）：每个账户每分钟最多 5 次，别的账户不受影响 |
| 契约说明与实现不一致（P1、P2 评审都发现过） | 九个新操作和新结构的说明逐句与代码核对：`listApiTokens` 说明只列未撤销的令牌、令牌原文只在创建时出现；`revokeApiToken` 写明三种 404；`listTimezones` 写明 `user_timezone` 不限于列表；`ApiToken.last_used` 写明精确到分钟 |

## 7. 交接的处理

| 交接 | P3a 处理的条目 | 留下的条目 | 状态 |
|---|---|---|---|
| M0-P3-api-codegen-notes | 1 中 google/uuid 的守卫（Task 3、7）；2 中参数绑定的出口（Task 2、7） | 无 | done |
| M0-P6-e2e-notes | PAT 对等验收和认证 fixture 的 PAT（Task 14）；S3 的新字段（Task 12） | 页面的登录状态、S2（P4）；River 停机与 fixture 的预算（P3b）；fixture 写法的延伸（M4、M5、M8） | open |
| M1-P2-trim-content | 接口描述的主题：五个值，没有 `custom` 和调色板（Task 9） | 前端的 `IUserTheme`（P4、P5） | open |
| M1-P3-trim-platform | 实例配置的三个字段、`is_self_managed` 不定义（Task 12）；不再读的字段和地址不出现在接口描述里；令牌地址不带结尾 `/`（Task 7） | Cookie 会话和 CSRF、认证错误、前端改读实例字段（P4）；`set-email`（P3b） | open |

P2 评审第 6 节交给 P3 的两项：

| 事项 | 落在 |
|---|---|
| `bootstrap` 把每个桶接到对应的配置，没有测试守住（评审 M7） | P3a Task 10（`TestEachConfiguredBucketLimitsItsOperations`，第 3 节第 19 条） |
| 任何"不可用的密码"形式都要保持登录的耗时 | P3b（M2 设计 12 节 P3b 的交付物 4；评审举的例子是 `nerve users create` 不设密码） |

留给控制者、没有落点的事项：P1、P2 的测试中还有六个文件读记录器上活的头部映射（`platform/httpserver` 的 `api_test.go`、`middleware_test.go`、`routes_test.go`、`problem_test.go`，`bootstrap/errors_test.go`，`platform/webui/handler_test.go`），第 3 节第 22 条。

## 附录 A：原型验证记录（2026-09-26）

原型在 `$M2TMP/p3proto`（`605f367` 的副本，Go 1.27.1、Node 24.15.0、Docker 29.7.2、Apple M5 Max）。plan 中的代码就是原型中运行过的代码，由脚本从原型文件原样拼入 plan（新文件和改动大的文件给完整内容，改动小的给对前一个版本的统一差异；拼接脚本用 `patch -p1` 把全部差异块逐个应用到前一个版本上，结果都与原型的文件逐字节相同）。另在 `$M2TMP/p3stage` 从 `605f367` 的副本出发，**按 plan 的 Task 1–15 逐个应用**（最终文件加上 plan 写明的过渡版本），有生成物的 Task（5、7–12）先运行 `make gen` 并核对生成物与 plan 的 SHA-256 相同，每个 Task 之后都运行 `make lint-go` 和 `make test`，全部通过；Task 7、9–12、14 之后另跑了前端检查和端到端，Task 12、14 之后跑了 `make knip`，Task 15 之后跑了关键词守卫。Task 15 之后 `p3stage` 与原型逐文件相同（含生成物；不提交的构建产物除外）。复现中有三次 `make test` 在启动容器时失败（testcontainers 的回收容器或 Postgres 容器在期限内没有就绪，本机 Docker 同时跑着别的项目的容器），从失败的 Task 重跑后通过，文件没有改动。

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物没有差异 | `make gen` 前后逐目录比较（原型不是 git 仓库，用脚本代替 `git status`） | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过 |
| 前端检查 | `turbo run check:types check:lint check:format check:sync`；`make knip`；`make test-web`；关键词守卫（遍历目录的等价脚本） | 全部通过 |
| 端到端 | `make e2e` | 17 个故事中 16 个通过（S1、S2 两个、S4、A1–A11、A15）。S3 失败的原因与 P1、P2 相同：副本不是 git 仓库，`commit` 是 `unknown`；它的差异只有这一行，三个新字段的期望值都相符 |
| 依赖 | 从 `605f367` 的 `go.mod`、`go.sum` 出发执行 plan 的 `go -C server get …@v1.7.0` 和 `go -C server mod tidy` | 结果与 plan 的差异逐字节相同 |
| 重复运行 | `go test -count=5` 跑两个交错测试和 `TestUpdateProfileMergesConcurrentSteps`；`-count=3` 跑 `TestEachConfiguredBucketLimitsItsOperations`、`TestParametersThatDoNotBindAnswer400` | 全部通过 |

**原型中定下的事实**：

- **E1** oapi-codegen v2.8.0 对 3.1 的字符串：`{type: string, contentMediaType: image/png}` 生成 `*openapi_types.File`；`contentEncoding: base64` 生成 `*[]byte`；`contentEncoding: base64url` 仍是 `*string`；`format: uuid` 加 `contentEncoding: base64` 按 `format` 生成。`bodyshapegen` 照此判断（2.4）。
- **E2** 接线核对：删掉 identity 的 `oapi-codegen.yaml` 中 uuid 的映射再 `make gen-go`，runtime 进入 `go.mod` 之前，`TestGeneratedCodeUsesTheStandardUUID` 报出 `server.gen.go` 中的位置（传递依赖测试另因缺模块失败）；进入之后，只有 `TestGeneratedCodeUsesTheStandardUUID` 失败（identity 的 `server.gen.go` 中 7 处），`TestNerveBinaryLinksNoBannedModule` 通过：google/uuid 只经 `runtime` 和 `runtime/types` 进入（`go list -deps ./cmd/nerve` 核对）。
- **E3** 标准库的 `uuid.UUID` 经 `runtime.BindStyledParameterWithOptions`（`TextUnmarshaler`）绑定路径参数：`DELETE /api/v0/api-tokens/not-a-uuid` 是 400 `errors[{token_id, invalid_format}]`，合法的 uuid 到达 handler。
- **E4** 绑定错误以 `*gen.InvalidParamFormatError{ParamName, Err}` 等形状原样到达 `APIErrors.BadRequest`，没有包装；整程序测试 5 对 `limit`、`token_id` 走真实的生成代码。
- **E5** `contains_url`：用 Plane 的 `url.py` 在 Python 3 中对 `TestContainsURLAsPlane` 的 41 个字符串逐个运行，答案与测试的期望值全部相同（含土耳其语的 İ、ı，长 s，开尔文符号，不换行空格、垂直制表符、信息分隔符，第 500 个字符和第 1000 个字符的边界按字符而不是字节）。
- **E6** 时区名单：用脚本从 Plane 的 `base.py:30-178` 抽出 120 个 `(名称, IANA)`，逐个写进 `timezones.go`；Go 1.27.1 的时区数据能加载全部 120 个。
- **E7** 读活的头部映射：把 `Content-Type`、`Retry-After`、`WWW-Authenticate` 挪到 `WriteHeader` 之后，用改之前的 `apierrors_test.go`、`contract_test.go` 三个变异都没被发现；改读 `rec.Result().Header` 之后三个都被发现（第 3 节第 22 条）。
- **E8** 时区名字的大小写：一个导入 `time/tzdata` 的小程序对 `asia/shanghai`、`Asia/Shanghai`、`ASIA/SHANGHAI`、`utc` 调用 `time.LoadLocation`。开发机（macOS，APFS 不区分大小写）上四个都成功，设了 `ZONEINFO=/nonexistent` 也一样（先查 `/usr/share/zoneinfo`）。同一程序编译成 linux/arm64，在 Alpine 容器里只有 `Asia/Shanghai` 成功；用空的 tmpfs 盖住容器的 `/usr/share/zoneinfo`、只剩内嵌数据时结果相同（第 3 节第 10 条）。

**变异核对**（`$M2TMP/p3tools/muts.py`：改一处代码，跑相关的包，恢复；端到端的变异由 `muts_e2e.py` 重新构建 `bin/nerve` 后跑一个故事）。共 215 个变异，按 P2 评审的缺陷类别归类，最后一次运行全部被测试发现：

| Task | A 测试不失败 | B 假实现忽略参数 | C 日志里的密文 | D 读活的头部映射 | E 测试挂住 | F 说明与代码不符 | G 接线没人看 | 合计 |
|---|---|---|---|---|---|---|---|---|
| 1 | 8 | | | | | | | 8 |
| 2 | 16 | | | 3 | | | | 19 |
| 3 | 6 | | | | | | 2 | 8 |
| 4 | 15 | | | | | | | 15 |
| 5 | 14 | | | | | | | 14 |
| 6 | 22 | 1 | 4 | | | | | 27 |
| 7 | 7 | 1 | | | | 1 | 2 | 11 |
| 8 | 21 | | | | | | | 21 |
| 9 | 17 | 1 | | | | | 3 | 21 |
| 10 | 22 | | 2 | | | | 9 | 33 |
| 11 | 12 | | | | | | 1 | 13 |
| 12 | 9 | | | | | | 5 | 14 |
| 13 | 4 | | | | 1 | | | 5 |
| 14 | 6（端到端） | | | | | | | 6 |

有代表性的几个：

| 变异 | 结果 |
|---|---|
| 第 7 种用例在没有必填属性时不生成 | `TestBodyCasesCombineEveryKindTheSchemaHas` 失败 |
| `BadRequest` 把绑定错误的原文写进 `detail` | `TestAPIErrorsBadRequestNamesTheParameter` 的 7 个子测试失败 |
| 传递依赖测试对 google/uuid 一律放行 | `TestBannedImports` 失败 |
| 生成代码的 uuid 规则不看导入时起的名字 | `TestRuntimeUUIDUses` 失败 |
| 列表的游标条件改为 `<=`（含上一页最后一行） | `TestListAPITokensPageByPage` 失败 |
| `TouchAPIToken` 每次都写 / 同时改 `updated_at` | `TestTouchAPIToken` 失败 |
| PAT 在 `expired_at` 那一刻仍有效 | `TestAuthenticateRejectsAPAT/expiring_now`、`TestCreateAPITokenRechecksTheCredentialUnderTheLock/token_expired` 失败 |
| 锁不复核凭证 | 创建 PAT、修改密码、停用的 `…RechecksTheCredentialUnderTheLock` 都失败 |
| 创建 PAT 的日志带哈希的大写十六进制 | `TestCreateAPIToken` 失败（P2 的 `assertNoSecret` 只查小写，Task 6 补上大写） |
| `UpdateProfile` 用子查询读语句开始时的步骤再合并，不用锁住的行 | `TestUpdateProfileMergesConcurrentSteps` 失败 |
| 修改密码不比较快照 / 不重试 / 每次重试都重新哈希 | `TestChangePasswordAfterAConcurrentChange` 失败 |
| `RevokeSessions` 连已到期、已撤销的也改写 | `TestRevokeSessions` 失败 |
| `bootstrap` 把一个桶接到别的桶的配置（七个桶各试一次），或不接 `password_user` | `TestEachConfiguredBucketLimitsItsOperations` 的对应子测试失败 |
| 登录在锁下不看新哈希 / 相信第一次校验 | `TestALoginWithTheOldPasswordFailsWhenThePasswordChangesMeanwhile` 失败 |
| 交错测试的闸门永不打开 | 测试在 10 秒后失败，不挂住 |
| `listApiTokens` 的 `x-problem-codes` 去掉 `validation_failed` | `TestListAPITokensProblems` 失败（`CheckResponse` 核对码已声明） |
| 时区偏移照 Plane 的写法 | `TestTimezoneOffsets` 失败 |
| 删掉 `cmd/nerve` 的 `time/tzdata` | `TestNerveBinaryEmbedsTheTimeZoneDatabase` 失败 |
| A7：修改密码不撤销会话；A11：不写 `last_used`、不看 `expired_at` | 对应的故事失败（重新构建 `bin/nerve` 后跑） |

原型中最初没被发现、改了测试之后才被发现的：

- 日志中 `files.size_limit` 取了常数：`validConfig` 用的正是默认值，测试分不出来。改用不同于默认值、也不同于别的键的值（2.3），从错的字段取值的三个变异都被发现。
- `contains_url` 按字节截断第 500 个字符：原来的用例里没有"地址在第 500 个字符之内、却在第 500 个字节之外"的文本。加上这个用例（Plane 在 Python 3 中答 `true`）后被发现。
- 主题名单少一个值：`TestCheckProfilePatchAcceptsAValidPatch` 原来遍历领域自己的名单，名单少了测试照样通过。改为独立写出的五个值（前端的 `THEME_OPTIONS` 和表的 CHECK）后被发现。
- 三个 D 类变异（E7）。

另有一个变异第一稿与原代码等价，改写变异本身，测试不变：交错 4 的"登录相信第一次校验"。

**P2 的缺陷类别**（P2 评审发现过的类别，对本 plan 逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试不失败（A） | 每个 Task 的规则、边界（`limit` 0/1/100/101、到期那一刻、恰好一分钟、第 500 和 1000 个字符、同一时刻的三行）、SQL 的条件 | 发现并修正三处：`validConfig` 用默认值；`contains_url` 缺第 500 个字节的边界；主题的测试遍历被测的名单 |
| 忽略参数的假实现（B） | `fakeAPITokens` 只按它持有的哈希和 id 找到令牌；`fakeCredentials`、`fakeAccount` 在调用记录里写下 id；HTTP 适配器的假用例记下入参 | 3 个"传错 id"的变异都被发现 |
| 日志里的密文漏掉某种写法（C） | 新增日志："API token created"、"API token revoked"、"password changed"、"account deactivated"、"request parameters not bound"（DEBUG） | `assertNoSecret` 补上大写十六进制；令牌、哈希、两个密码的原文、十六进制、base64 写法都查 |
| 读活的头部映射（D） | P3a 新写和改到的测试 | 发现并修正（E7）；另有六个 P1、P2 文件留给控制者（第 7 节） |
| 测试挂住（E） | 交错测试、并发合并、等锁的探测 | 每个等待都有 10 秒的期限；闸门不开的变异在期限后失败 |
| 说明与代码不符（F） | 九个新操作的说明、新结构的字段说明、README 的两条、交接的处理结果 | 逐句与代码核对；`x-problem-codes` 少声明一个码的变异由 `CheckResponse` 发现 |
| 接线没人看（G） | 每个新用例、每个桶、实例的三个设置、时区的公开、`time/tzdata` | 22 个接线变异都由整程序测试或 archtest 发现 |

**没有证明的**：

- 持续集成（ubuntu）上的运行；S3 在 git 仓库中的结果。P3a 合并后看持续集成。
- 3.5 的交错测试 1–3（P3b）。
- A7–A11 的页面版本（P4、P5）。
