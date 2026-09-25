# M2/P1 平台约定与第一个认证 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 任何调用方都能注册，并用注册得到的令牌访问 `GET /me`；不带令牌访问非公开操作一律 401；不合契约的请求体一律 400；后续 M 照做的平台约定全部定下：共享内核、`TxManager`、按路由的中间件链、`ProblemError` 的映射、`x-problem-codes` 与 `apitest` 的核对、请求体结构表的生成器、表结构约定与 sqlc。

**Architecture:** 平台（`internal/platform`）只声明小接口（`Authenticator`、`ProblemError`），`internal/shared` 的类型按结构满足它们，两边互不导入，`bootstrap` 用编译期断言把它们对上。每个模块的 HTTP 适配器把 `API.Middlewares(gen.BodyShapes())` 交给生成代码：请求元信息 → 期限 → 请求体上限 → 默认拒绝的认证 → 由接口描述生成的请求体结构检查；handler 和中间件的一切错误都经 `APIErrors.Write` 这"一条路"变成 problem。`identity` 按端口与适配器分层：`domain` 是纯规则（邮箱、密码与名单、令牌布局），`app` 声明端口并编排用例，`adapter/{argon2,signing,postgres,authn,http}` 实现端口；`bootstrap` 读配置和密钥文件、接线，并用四个整程序测试把组合出来的程序与契约逐个操作核对。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、goose v3.28.0、oapi-codegen v2.8.0、sqlc v1.31.1（`CGO_ENABLED=0`）、golang-jwt/jwt/v5 v5.3.1、golang.org/x/crypto v0.57.0、oapi-codegen/nullable v1.2.0、kin-openapi v0.149.0（测试）、golangci-lint 2.13.2、PostgreSQL 18.6；Node 24、pnpm 11.10.0、Playwright 1.63.0、pg 8.23.0。

**Spec:** `docs/v0/M2-auth/specs/P1-platform-core.md`（上级：`docs/v0/M2-auth/M2-design.md`）

## Global Constraints

- **Go 版本**：`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`，**每次 `go get` / `go mod tidy` 之后都用 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod` 核对**（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。
- **依赖版本写死，不用 `@latest`**：
  - `server/tools`：`github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1`（tool，Task 6）；kin-openapi v0.142.0、oapi-codegen v2.8.0、`go.yaml.in/yaml/v3` v3.0.4 由 `go mod tidy` 改为直接依赖，版本不变（Task 1）。
  - `server`：`github.com/golang-jwt/jwt/v5@v5.3.1`、`golang.org/x/crypto@v0.57.0`（Task 9；会把间接依赖 `golang.org/x/text` 抬到 v0.42.0）；`github.com/oapi-codegen/nullable@v1.2.0`（Task 10）。
  - 不加 `github.com/oapi-codegen/runtime`（P3 才有带参数的操作）。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（从本 plan 起 `make test` 也跑 `server/tools`）。改了接口描述、`sqlc.yaml`、查询或迁移的 Task 先执行 `make gen`，**提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。改了 `e2e/`、`tools/` 或 `knip.jsonc` 的 Task 另执行 `make lint-web` 和 `make knip`。
- **生成的文件不手写、不从本 plan 复制**：执行生成命令，提交它的输出。最终版本的生成物给出 SHA-256（`shasum -a 256`）和行数；对不上时停下来，说明某个输入与本 plan 不一致。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。Go 模块只按本 plan 写明的命令 `go get`。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；不留没有使用者的代码（设计中暂无使用者的成员随使用者在 P2、P3 加入，spec 第 3 节第 3 条）。
- **注释**：Go、TS、JS 代码的注释用英文；配置文件、YAML、SQL 迁移的文件头、Makefile、生成脚本的中文说明照本 plan 原样。
- **代码块**：除标为 `diff` 的块外，都是**完整的文件内容**，照原样写入，不要改动（原型中逐字节运行过）。`diff` 块是对当前文件的统一差异，照差异修改；Makefile 的命令行以 Tab 开头，改完用 `make -n gen-go` 确认没有 `missing separator`。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再写成最终版本；每个过渡版本都在原型的逐 Task 复现中运行过（spec 附录 A）。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；"过渡"表示这个 Task 写过渡版本，后面的 Task 写最终版本。

| 文件 | 职责 | Task |
|---|---|---|
| `Makefile`（修改） | `make test`、`make lint-go` 进入 `server/tools`；`gen-go` 运行 `bodyshapegen` | 1（过渡） |
| `server/tools/go.mod`、`go.sum`（修改） | 生成器的直接依赖 | 1 |
| `server/internal/platform/httpserver/bodyshape/bodyshape.go`、`middleware.go`、`bodyshape_test.go`、`middleware_test.go` | 请求体结构检查 | 1 |
| `server/tools/bodyshapegen/main.go`、`generate.go`、`generate_test.go`、`testdata/{common.yaml,things.yaml,oapi-codegen.yaml,things.golden}` | 结构表的生成器 | 1 |
| `server/internal/modules/instance/adapter/http/gen/bodyshape.gen.go`（生成） | instance 的结构表（没有请求体） | 1 |
| `server/internal/shared/error.go`、`error_test.go`、`actor.go`、`actor_test.go`、`tx.go` | 共享内核 | 2 |
| `server/internal/platform/clock/clock.go`、`clock_test.go`、`clocktest/clocktest.go` | 时钟 | 2 |
| `server/internal/platform/postgres/tx.go`、`tx_test.go`、`pool.go`、`pool_test.go`（后两个修改） | `TxManager`；UTC | 2 |
| `server/internal/archtest/rules_test.go`、`rules_cases_test.go`（修改） | 规则 4、6、测试工具 | 2 |
| `server/internal/bootstrap/app.go`（修改） | `TxManager` 的编译期断言 | 2（过渡） |
| `server/internal/platform/config/config.go`、`load.go`、`validate.go` 及三个测试（修改） | P1 的配置项 | 3 |
| `server/configs/config.yaml`、`config.dev.yaml`、`config.test.yaml`、`embed_test.go`（修改） | 默认值和按环境的覆盖 | 3 |
| `server/internal/platform/httpserver/problem.go`、`middleware.go`、`routes.go`、`server.go`、`apierrors.go` 及各自的测试（修改） | 平台码、`RequestID`、`Router`、`addr_file`、`ProblemError` 映射 | 4 |
| `server/internal/platform/httpserver/api.go`、`api_test.go` | `API`、按路由的中间件、默认拒绝 | 4 |
| `server/internal/platform/httpserver/contract_test.go`（修改） | 平台写出的 problem 的契约测试 | 4 |
| `api/common.yaml`（修改） | `FieldError.code` | 4 |
| `server/internal/platform/httpserver/apigen/components.gen.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 4、5、10 |
| `server/internal/platform/httpserver/apitest/apitest_test.go`（修改） | 字段码的样例 | 4（过渡）、5 |
| `server/internal/modules/instance/module.go`、`adapter/http/handler.go`、`handler_test.go`（修改） | 挂到 `Router`；`BodyError`、`Write` | 4（过渡）、11 |
| `server/internal/bootstrap/app.go`（修改） | `NewRouter` | 4（过渡） |
| `api/openapi.yaml`（修改） | 顶层 `x-problem-codes`、`securitySchemes` | 5（过渡）、10 |
| `api/modules/instance.yaml`、`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`（修改） | `security`、`x-problem-codes`；模块模板 | 5 |
| `server/internal/modules/instance/adapter/http/main_test.go` | `apitest.Main` | 5 |
| `server/internal/platform/httpserver/apitest/apitest.go`（修改）、`problems.go`、`problems_test.go` | `CheckRequest`、问题码核对、`Main` | 5 |
| `server/internal/platform/httpserver/apitest/rules_test.go`、`rules_cases_test.go`（修改） | `security`、`x-problem-codes` 的写法 | 5 |
| `Makefile`（修改） | sqlc | 6 |
| `server/tools/go.mod`、`go.sum`（修改） | sqlc v1.31.1 | 6 |
| `server/migrations/sql/00001_identity_users.sql`、`00002_identity_profiles.sql`、`00003_identity_auth_sessions.sql`；删除 `sql/.gitkeep` | 三张表 | 6 |
| `server/migrations/embed.go`、`embed_test.go`（修改）、`schema_test.go` | 内嵌；约束名和反例 | 6 |
| `server/internal/platform/postgres/migrator_test.go`（修改） | 注释 | 6 |
| `server/sqlc.yaml`、`server/internal/modules/identity/adapter/postgres/queries/{users,profiles,sessions}.sql` | sqlc | 6 |
| `server/internal/modules/identity/adapter/postgres/gen/*.go`（生成） | | 6 |
| `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go` | `TestSQLCSchemaScope` | 6 |
| `server/internal/bootstrap/app_test.go`（修改） | 注释 | 6（过渡） |
| `e2e/stories/smoke/s1-server-ready.spec.ts`、`e2e/tsconfig.json`（修改） | S1 的迁移断言 | 6 |
| `server/internal/modules/identity/domain/{errors,email,password,session,user}.go` 及测试 | 领域规则 | 7 |
| `server/internal/modules/identity/domain/common_passwords.txt`（生成） | 常见密码名单 | 7 |
| `tools/password-blocklist/build.mjs`、`knip.jsonc`（修改） | 名单的生成脚本 | 7 |
| `server/internal/modules/identity/app/{ports,create_user,register,get_me,authenticate}.go` 及测试、`fakes_test.go` | 端口和用例 | 8 |
| `server/internal/modules/identity/adapter/argon2/`、`signing/`、`postgres/{store,users,sessions}.go`、`authn/` 及测试 | 适配器 | 9 |
| `server/go.mod`、`go.sum`（修改） | jwt、x/crypto；nullable | 9、10 |
| `api/modules/identity.yaml`、`server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml` | identity 的接口 | 10 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`（生成） | | 10 |
| `server/internal/modules/identity/adapter/http/handler.go`、`handler_test.go` | HTTP 适配器 | 10 |
| `server/internal/modules/identity/module.go` | 模块入口 | 11 |
| `server/internal/platform/httpserver/apitest/operations.go`、`operations_test.go` | 整程序测试用的操作列表和请求体样例 | 11 |
| `server/internal/bootstrap/app.go`、`app_test.go`、`commands.go`、`commands_test.go`（修改）、`contract_test.go`、`auth_test.go`、`errors_test.go` | 接线和整程序测试 | 11 |
| `e2e/fixtures/server.ts`、`db.ts`、`test.ts`、`e2e/global-setup.ts`（修改）、`e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts` | fixture | 12 |
| `e2e/stories/identity/a1-sign-up.spec.ts`、`a2-sign-up-refused.spec.ts` | A1、A2 的接口版本 | 12 |
| `docs/v0/v0-design.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/plane-diff.md`、`README.md`、`docs/v0/M2-auth/handoffs/M0-P{1,2,3,4,6}-*.md`（修改） | 文档同步、交接 | 13 |

---

### Task 1: 工具模块进持续集成；请求体结构检查 `bodyshape` 与生成器 `bodyshapegen`

**Files:**
- Modify: `Makefile`（过渡：Task 6 再加 sqlc）
- Create: `server/internal/platform/httpserver/bodyshape/bodyshape.go`、`middleware.go`、`bodyshape_test.go`、`middleware_test.go`
- Create: `server/tools/bodyshapegen/main.go`、`generate.go`、`generate_test.go`、`testdata/common.yaml`、`testdata/things.yaml`、`testdata/oapi-codegen.yaml`、`testdata/things.golden`
- Modify: `server/tools/go.mod`（`go mod tidy`）
- Generate: `server/internal/modules/instance/adapter/http/gen/bodyshape.gen.go`

**Interfaces:**
- Consumes: M0 的 `make gen-go` 循环和 `server/tools` 模块（oapi-codegen v2.8.0 的 `pkg/codegen`、`pkg/util`）。
- Produces:
  - `bodyshape.Table{Nodes []Node; Roots map[string]int}`、`(*Table).Check(pattern string, body []byte) []FieldError`、`bodyshape.Error{Fields}`（400 `bad_request` 的 `ProblemError`）、`bodyshape.ErrNotJSON`、`bodyshape.Middleware(t *Table, onError func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler`（Task 4 的 `API.Middlewares` 用它）；
  - 每个模块的 `gen.BodyShapes() *bodyshape.Table`（`make gen-go` 生成）。

**Tests:**
- `bodyshape_test.go`：`TestCheck`（19 个情况：合法、未知字段、嵌套和数组里的未知字段、不合法的 `null`、缺必填、类型错、整数与小数、格式错、`Open` 与 `Closed`，一次返回全部问题并排序）；`TestCheckOfARouteWithoutABody`；`TestFormatsMatchTheDecoder`（13 个 `date-time`、`uuid` 样例，含 JSON 转义写法，检查器与 `json.Unmarshal` 进 `time.Time`、`uuid.UUID` 的判断逐个相同）；`TestErrorIsAProblem`（400、`bad_request`、字段）。
- `middleware_test.go`：合法的请求体原样交给下一层；结构问题一次交给 `onError`；不是 JSON 时是 `ErrNotJSON`；没有结构表或没有请求体时放行；超过上限的读取错误原样交给 `onError`。
- `generate_test.go`：`TestGenerateMatchesTheGoldenFile`（`-update` 重写黄金文件）、`TestGenerateIsDeterministic`（连续 10 次相同）、`TestGenerateFollowsTheTypeMapping`（默认映射下 `email` 生成为 `openapi_types.Email`，没有检查器，生成失败）、`TestGenerateRejectsWhatItCannotCheck`（10 个：`date`、`byte`、`x-go-type`、`x-go-type-import`、`oneOf`、`allOf`、`not`、两个类型的 `anyOf`、OpenAPI 3.0 的 `nullable`、嵌套数组里的错误带完整路径）。

- [ ] **Step 1: 写 `bodyshape` 包**

`server/internal/platform/httpserver/bodyshape/bodyshape.go`：

```go
// Package bodyshape checks the structure of JSON request bodies at the API
// boundary, before the generated strict handler decodes them (M2 design
// 3.11): JSON types, undeclared properties, null where the contract does not
// allow it, missing required properties, and the string formats that the
// generated code decodes into Go types. Every problem is collected in one
// pass. Values (lengths, enums, ranges, e-mail syntax) are the domain's.
//
// The tables are generated per module from the API description by
// server/tools/bodyshapegen, next to the module's server.gen.go. The package
// uses only the standard library.
package bodyshape

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"
	"uuid"
)

// Type is a set of JSON types.
type Type uint8

// The JSON types. Integer is a number literal that a Go integer can hold; it
// also counts as a Number.
const (
	Null Type = 1 << iota
	Boolean
	Integer
	Number
	String
	Array
	Object
)

// Any accepts every JSON type: a schema without `type`.
const Any Type = 0

// Format is a string format that the generated code decodes into a Go type,
// named after that type. A wrong value cannot be held by the generated type,
// so it is checked before decoding.
type Format uint8

// The formats with a checker.
const (
	FormatNone Format = iota
	FormatTime        // time.Time (date-time)
	FormatUUID        // the standard library's uuid.UUID (uuid)
)

// Node.Extra and Node.Items hold a node index or one of these.
const (
	// Closed rejects undeclared properties: additionalProperties: false.
	Closed = -1
	// Open does not check them, or the items of an array without an item
	// schema: additionalProperties: true, or no schema.
	Open = -2
)

// Node is one schema of a request body. Child schemas are indexes into
// Table.Nodes, so recursive schemas are fine.
type Node struct {
	Types    Type
	Format   Format
	Props    map[string]int // object: declared property → node
	Required []string       // object: required properties
	Extra    int            // object: node of undeclared properties, Closed or Open
	Items    int            // array: node of the items, or Open
}

// Table holds the request-body schemas of one module.
type Table struct {
	Nodes []Node
	Roots map[string]int // route pattern, e.g. "POST /api/v0/auth/register" → body node
}

// Field codes, the subset of the closed set of api/common.yaml FieldError
// that the structure check produces.
const (
	codeRequired      = "required"
	codeInvalidFormat = "invalid_format"
	codeNotAllowed    = "not_allowed"
)

// FieldError is one structural problem. Field is the JSON path, e.g.
// tags[1].name; the empty path is the body itself. It satisfies the field
// interface of httpserver.ProblemError by structure.
type FieldError struct {
	Field string
	Code  string
}

func (f FieldError) Error() string {
	switch f.Code {
	case codeRequired:
		return "is required"
	case codeNotAllowed:
		return "is not a property of this request"
	default:
		return "has the wrong type or format"
	}
}

// ProblemField returns the JSON path.
func (f FieldError) ProblemField() string { return f.Field }

// ProblemCode returns the field code.
func (f FieldError) ProblemCode() string { return f.Code }

// Error lists every structural problem of one request body. It satisfies
// httpserver.ProblemError by structure: 400 bad_request with the fields.
type Error struct {
	Fields []FieldError
}

func (e *Error) Error() string { return "The request body does not match the API description." }

// ProblemStatus is 400: the request breaks the contract's structure.
func (e *Error) ProblemStatus() int { return 400 }

// ProblemCode is the platform's bad_request.
func (e *Error) ProblemCode() string { return "bad_request" }

// ProblemFields returns the field errors.
func (e *Error) ProblemFields() []error {
	errs := make([]error, len(e.Fields))
	for i, f := range e.Fields {
		errs[i] = f
	}
	return errs
}

// Check reports the structural problems of body, a syntactically valid JSON
// document, against the root of pattern, sorted by path. A pattern without a
// root has no JSON body to check.
func (t *Table) Check(pattern string, body []byte) []FieldError {
	root, ok := t.Roots[pattern]
	if !ok {
		return nil
	}
	var errs []FieldError
	t.walk(root, body, "", &errs)
	slices.SortFunc(errs, func(a, b FieldError) int {
		return cmp.Or(cmp.Compare(a.Field, b.Field), cmp.Compare(a.Code, b.Code))
	})
	return errs
}

// walk checks raw, the exact bytes of one JSON value, against node i.
func (t *Table) walk(i int, raw []byte, path string, errs *[]FieldError) {
	n := t.Nodes[i]
	kind := kindOf(raw)
	if n.Types != Any && kind&n.Types == 0 {
		*errs = append(*errs, FieldError{path, codeInvalidFormat})
		return
	}
	switch kind {
	case String:
		if n.Format != FormatNone && checkFormat(n.Format, raw) != nil {
			*errs = append(*errs, FieldError{path, codeInvalidFormat})
		}
	case Object:
		var props map[string]json.RawMessage
		_ = json.Unmarshal(raw, &props) // the whole body was valid JSON
		for name, value := range props {
			child, declared := n.Props[name]
			switch {
			case declared:
				t.walk(child, value, join(path, name), errs)
			case n.Extra == Closed:
				*errs = append(*errs, FieldError{join(path, name), codeNotAllowed})
			case n.Extra >= 0:
				t.walk(n.Extra, value, join(path, name), errs)
			}
		}
		for _, name := range n.Required {
			if _, ok := props[name]; !ok {
				*errs = append(*errs, FieldError{join(path, name), codeRequired})
			}
		}
	case Array:
		if n.Items < 0 {
			return
		}
		var items []json.RawMessage
		_ = json.Unmarshal(raw, &items)
		for j, item := range items {
			t.walk(n.Items, item, fmt.Sprintf("%s[%d]", path, j), errs)
		}
	}
}

// kindOf returns the JSON type of a valid JSON value.
func kindOf(raw []byte) Type {
	switch raw[0] {
	case 'n':
		return Null
	case 't', 'f':
		return Boolean
	case '"':
		return String
	case '[':
		return Array
	case '{':
		return Object
	}
	if _, err := strconv.ParseInt(string(raw), 10, 64); err == nil {
		return Integer | Number
	}
	return Number
}

// checkFormat decodes the value's own bytes into the format's Go type with
// encoding/json: exactly what the generated decoder does with this field, so
// the two accept the same values by construction (M2 design 3.11).
func checkFormat(f Format, raw []byte) error {
	switch f {
	case FormatTime:
		return json.Unmarshal(raw, new(time.Time))
	case FormatUUID:
		return json.Unmarshal(raw, new(uuid.UUID))
	}
	return fmt.Errorf("bodyshape: unknown format %d", f)
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
```

`server/internal/platform/httpserver/bodyshape/middleware.go`：

```go
package bodyshape

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrNotJSON is answered for a body that is not one valid JSON document.
var ErrNotJSON = errors.New("the request body is not valid JSON")

// Middleware checks the body of every request whose route has a root in t,
// before the generated strict handler decodes it. It reads the whole body
// (already bounded by the platform's body limit) and puts it back unchanged.
//
// onError answers a body that could not be read (the read error, e.g.
// *http.MaxBytesError), that is not JSON (ErrNotJSON) or that breaks the
// structure (*Error). An empty body is passed on: the generated decoder
// answers it.
func Middleware(t *Table, onError func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := t.Roots[r.Pattern]; !ok || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				onError(w, r, err)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			if len(bytes.TrimSpace(body)) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if !json.Valid(body) {
				onError(w, r, ErrNotJSON)
				return
			}
			if fields := t.Check(r.Pattern, body); len(fields) > 0 {
				onError(w, r, &Error{Fields: fields})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

`server/internal/platform/httpserver/bodyshape/bodyshape_test.go`。其中三行在反引号字符串里写的是 JSON 的转义：反斜杠、`u005a`（两行）和反斜杠、`u0030`（一行），检查器必须看到转义的原样。有的编辑工具会把这种写法换成字符本身，测试就悄悄变弱了，所以写完后核对：

```go
package bodyshape

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
	"uuid"
)

const pattern = "POST /api/v0/things"

// things is what bodyshapegen writes for a schema with every supported
// construct: required and optional properties, a nullable object and
// nullable scalars, nesting, an array of objects, an open map, formats.
func things() *Table {
	return &Table{
		Nodes: []Node{
			/* 0 */ {Types: Object, Extra: Closed, Items: Open, Required: []string{"name", "nested"},
				Props: map[string]int{"count": 2, "id": 4, "labels": 10, "maybe": 5, "name": 1, "nested": 6, "note": 11, "tags": 8, "when": 3}},
			/* 1 */ {Types: String, Extra: Open, Items: Open},
			/* 2 */ {Types: Integer, Extra: Open, Items: Open},
			/* 3 */ {Types: String, Format: FormatTime, Extra: Open, Items: Open},
			/* 4 */ {Types: String | Null, Format: FormatUUID, Extra: Open, Items: Open},
			/* 5 */ {Types: Object | Null, Extra: Closed, Items: Open, Props: map[string]int{"a": 1}, Required: []string{"a"}},
			/* 6 */ {Types: Object, Extra: Closed, Items: Open, Props: map[string]int{"a": 1, "b": 7}, Required: []string{"a"}},
			/* 7 */ {Types: Integer | Null, Extra: Open, Items: Open},
			/* 8 */ {Types: Array, Extra: Open, Items: 9},
			/* 9 */ {Types: Object, Extra: Closed, Items: Open, Props: map[string]int{"name": 1}, Required: []string{"name"}},
			/* 10 */ {Types: Object, Extra: 1, Items: Open, Props: map[string]int{}},
			/* 11 */ {Types: Number, Extra: Open, Items: Open},
		},
		Roots: map[string]int{pattern: 0},
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []FieldError
	}{
		{"valid", `{"name":"a","count":3,"when":"2026-09-25T10:00:00Z","id":"0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70",` +
			`"maybe":{"a":"x"},"nested":{"a":"y","b":2},"tags":[{"name":"t"}],"labels":{"k":"v"},"note":1.5}`, nil},
		{"only the required properties", `{"name":"a","nested":{"a":"y"}}`, nil},
		{"unknown top-level property", `{"name":"a","nested":{"a":"y"},"extra":1}`,
			[]FieldError{{"extra", "not_allowed"}}},
		{"unknown nested property", `{"name":"a","nested":{"a":"y","x":true}}`,
			[]FieldError{{"nested.x", "not_allowed"}}},
		{"null for an optional non-nullable property", `{"name":"a","nested":{"a":"y"},"count":null}`,
			[]FieldError{{"count", "invalid_format"}}},
		{"null for a required non-nullable property", `{"name":null,"nested":{"a":"y"}}`,
			[]FieldError{{"name", "invalid_format"}}},
		{"null where the contract allows it", `{"name":"a","nested":{"a":"y","b":null},"maybe":null,"id":null}`, nil},
		{"missing required properties", `{"nested":{}}`,
			[]FieldError{{"name", "required"}, {"nested.a", "required"}}},
		{"wrong types", `{"name":5,"nested":[],"count":"3"}`,
			[]FieldError{{"count", "invalid_format"}, {"name", "invalid_format"}, {"nested", "invalid_format"}}},
		{"a decimal is not an integer", `{"name":"a","nested":{"a":"y"},"count":1.5}`,
			[]FieldError{{"count", "invalid_format"}}},
		{"an exponent is not an integer", `{"name":"a","nested":{"a":"y"},"count":1e2}`,
			[]FieldError{{"count", "invalid_format"}}},
		{"an integer is a number", `{"name":"a","nested":{"a":"y"},"note":7}`, nil},
		{"array items", `{"name":"a","nested":{"a":"y"},"tags":[{"name":"t"},{},{"name":"u","x":1}]}`,
			[]FieldError{{"tags[1].name", "required"}, {"tags[2].x", "not_allowed"}}},
		{"open map values", `{"name":"a","nested":{"a":"y"},"labels":{"k":5,"j":"ok"}}`,
			[]FieldError{{"labels.k", "invalid_format"}}},
		{"wrong date-time", `{"name":"a","nested":{"a":"y"},"when":"2026-09-25 10:00:00Z"}`,
			[]FieldError{{"when", "invalid_format"}}},
		{"date-time with an escaped Z", `{"name":"a","nested":{"a":"y"},"when":"2026-09-25T10:00:00\u005a"}`, nil},
		{"wrong uuid", `{"name":"a","nested":{"a":"y"},"id":"xyz"}`,
			[]FieldError{{"id", "invalid_format"}}},
		{"the body is not an object", `["name"]`, []FieldError{{"", "invalid_format"}}},
		{"every problem at once, sorted by path", `{"when":"yesterday","zzz":1,"nested":{"a":"y","b":"2"}}`,
			[]FieldError{{"name", "required"}, {"nested.b", "invalid_format"}, {"when", "invalid_format"}, {"zzz", "not_allowed"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := things().Check(pattern, []byte(tt.body)); !slices.Equal(got, tt.want) {
				t.Errorf("Check(%s) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestCheckOfARouteWithoutABody(t *testing.T) {
	if got := things().Check("GET /api/v0/things", []byte(`{"x":1}`)); got != nil {
		t.Errorf("Check() = %v, want nil for a route without a root", got)
	}
}

// The checker must accept exactly what the generated decoder accepts: it
// decodes the field's own bytes into the same Go type (M2 design 3.11).
func TestFormatsMatchTheDecoder(t *testing.T) {
	type decoded struct {
		When time.Time `json:"when"`
		ID   uuid.UUID `json:"id"`
	}
	tests := []struct {
		field  string
		format Format
		raw    string
	}{
		{"when", FormatTime, `"2026-09-25T10:00:00Z"`},
		{"when", FormatTime, `"2026-09-25T10:00:00+08:00"`},
		{"when", FormatTime, `"2026-09-25 10:00:00Z"`},
		{"when", FormatTime, `"2026-09-25"`},
		{"when", FormatTime, `"2026-09-25T10:00:00\u005a"`},
		{"when", FormatTime, `"not a time"`},
		{"id", FormatUUID, `"0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`},
		{"id", FormatUUID, `"0199A2B4-7C3E-7D2A-9F10-2B3C4D5E6F70"`},
		{"id", FormatUUID, `"0199a2b47c3e7d2a9f102b3c4d5e6f70"`},
		{"id", FormatUUID, `"{0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70}"`},
		{"id", FormatUUID, `"urn:uuid:0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`},
		{"id", FormatUUID, `"\u0030199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`},
		{"id", FormatUUID, `"xyz"`},
	}
	accepted := 0
	for _, tt := range tests {
		got := checkFormat(tt.format, []byte(tt.raw))
		want := json.Unmarshal([]byte(`{"`+tt.field+`":`+tt.raw+`}`), new(decoded))
		if (got == nil) != (want == nil) {
			t.Errorf("%s %s: checker %v, decoder %v", tt.field, tt.raw, got, want)
		}
		if got == nil {
			accepted++
		}
	}
	// Both outcomes occur, so the comparison is not trivially one-sided.
	if accepted == 0 || accepted == len(tests) {
		t.Errorf("%d of %d values accepted, want some of each", accepted, len(tests))
	}
}

func TestErrorIsAProblem(t *testing.T) {
	err := &Error{Fields: []FieldError{{"name", "required"}, {"extra", "not_allowed"}}}

	if err.ProblemStatus() != 400 || err.ProblemCode() != "bad_request" {
		t.Errorf("problem = %d %s, want 400 bad_request", err.ProblemStatus(), err.ProblemCode())
	}
	fields := err.ProblemFields()
	if len(fields) != 2 {
		t.Fatalf("ProblemFields() = %v, want 2", fields)
	}
	f, ok := fields[0].(interface {
		ProblemField() string
		ProblemCode() string
	})
	if !ok || f.ProblemField() != "name" || f.ProblemCode() != "required" || fields[0].Error() != "is required" {
		t.Errorf("fields[0] = %v, want name required with a message", fields[0])
	}
}
```

Run: `grep -cE '\\u(005a|0030)' server/internal/platform/httpserver/bodyshape/bodyshape_test.go`
Expected: `3`。不是 3 时，把那几行改回反斜杠加四位十六进制的写法。

`server/internal/platform/httpserver/bodyshape/middleware_test.go`：

```go
package bodyshape

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// run sends body to the middleware for pattern and reports what reached the
// next handler, or the error given to onError.
func run(t *testing.T, pattern, body string, limit int64) (reached string, called bool, err error) {
	t.Helper()
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		data, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Fatal(readErr)
		}
		reached = string(data)
	})
	h := Middleware(things(), func(_ http.ResponseWriter, _ *http.Request, e error) { err = e })(next)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/things", strings.NewReader(body))
	req.Pattern = pattern
	if limit > 0 {
		req.Body = http.MaxBytesReader(rec, req.Body, limit)
	}
	h.ServeHTTP(rec, req)
	return reached, called, err
}

func TestMiddlewarePassesAValidBodyOnUnchanged(t *testing.T) {
	body := `{ "name": "a",  "nested": {"a": "y"} }`

	reached, called, err := run(t, pattern, body, 0)

	if !called || err != nil || reached != body {
		t.Errorf("next called %v with %q, error %v; want the body unchanged", called, reached, err)
	}
}

func TestMiddlewareAnswersEveryStructuralProblem(t *testing.T) {
	_, called, err := run(t, pattern, `{"extra":1,"nested":{"a":"y"}}`, 0)

	var shapeErr *Error
	if called || !errors.As(err, &shapeErr) {
		t.Fatalf("next called %v, error %v; want a *Error", called, err)
	}
	want := []FieldError{{"extra", "not_allowed"}, {"name", "required"}}
	if !slices.Equal(shapeErr.Fields, want) {
		t.Errorf("fields = %v, want %v", shapeErr.Fields, want)
	}
}

func TestMiddlewareAnswersABodyThatIsNotJSON(t *testing.T) {
	for _, body := range []string{`{"name":`, `{"name":"a","nested":{"a":"y"}} trailing`, `nope`} {
		_, called, err := run(t, pattern, body, 0)

		if called || !errors.Is(err, ErrNotJSON) {
			t.Errorf("body %q: next called %v, error %v; want ErrNotJSON", body, called, err)
		}
	}
}

// The generated decoder answers an empty body; routes without a root are
// not this middleware's business.
func TestMiddlewarePassesOnWhatItDoesNotCheck(t *testing.T) {
	tests := []struct{ name, pattern, body string }{
		{"empty body", pattern, ""},
		{"blank body", pattern, "  \n"},
		{"route without a body schema", "GET /api/v0/things", "not even JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached, called, err := run(t, tt.pattern, tt.body, 0)

			if !called || err != nil || reached != tt.body {
				t.Errorf("next called %v with %q, error %v; want the request passed on", called, reached, err)
			}
		})
	}
}

func TestMiddlewareReportsAnOversizedBody(t *testing.T) {
	_, called, err := run(t, pattern, `{"name":"`+strings.Repeat("a", 100)+`"}`, 16)

	var tooLarge *http.MaxBytesError
	if called || !errors.As(err, &tooLarge) {
		t.Errorf("next called %v, error %v; want the *http.MaxBytesError", called, err)
	}
}
```

- [ ] **Step 2: 运行 `bodyshape` 的测试**

Run: `go -C server test -count=1 ./internal/platform/httpserver/bodyshape/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape`

- [ ] **Step 3: 写生成器 `server/tools/bodyshapegen`**

`server/tools/bodyshapegen/main.go`：

```go
// Command bodyshapegen writes the request-body structure table of one module
// (M2 design 3.11): for every operation with a JSON request body, the
// schema's JSON types, properties, required properties, additionalProperties
// and the string formats the generated code decodes into Go types. The
// platform's bodyshape middleware checks each request body against it before
// the generated strict handler decodes the body.
//
// It reads the module's API description with oapi-codegen's own loader and
// the module's oapi-codegen configuration, so the formats follow the same
// type-mapping as server.gen.go. make gen-go runs it for every module:
//
//	go -C server/tools run ./bodyshapegen -config <oapi-codegen.yaml> -out <bodyshape.gen.go> <api/modules/m.yaml>
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	config := flag.String("config", "", "the module's oapi-codegen configuration")
	out := flag.String("out", "", "the Go file to write")
	flag.Parse()
	if *config == "" || *out == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: bodyshapegen -config <oapi-codegen.yaml> -out <bodyshape.gen.go> <module.yaml>")
		os.Exit(2)
	}
	src, err := generate(flag.Arg(0), *config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bodyshapegen: %s: %v\n", flag.Arg(0), err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "bodyshapegen: %v\n", err)
		os.Exit(1)
	}
}
```

`server/tools/bodyshapegen/generate.go`：

```go
package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/util"
	"go.yaml.in/yaml/v3"
)

const bodyshapeImport = "github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"

// checkers maps the Go types that have a bodyshape format checker to the
// checker's name. A string format generated as any other Go type but string
// fails the generation: its wrong values could not be answered field by
// field.
var checkers = map[codegen.SimpleTypeSpec]string{
	{Type: "time.Time", Import: "time"}: "FormatTime",
	{Type: "uuid.UUID", Import: "uuid"}: "FormatUUID",
}

// The JSON types in bodyshape's bit order.
var typeNames = []string{"null", "boolean", "integer", "number", "string", "array", "object"}

// node is one bodyshape.Node before it is written out.
type node struct {
	types    []string // JSON types; empty accepts any
	format   string   // checker name, or ""
	props    map[string]int
	required []string
	extra    int // node index, closed or open
	items    int // node index or open
}

const (
	closed = -1
	open   = -2
)

type generator struct {
	types codegen.TypeMapping
	nodes []node
	seen  map[*openapi3.Schema]int
}

// generate returns the Go source of the table for the module described by
// specPath, in the package and with the type-mapping of configPath.
func generate(specPath, configPath string) ([]byte, error) {
	var cfg codegen.Configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", configPath, err)
	}
	if cfg.PackageName == "" {
		return nil, fmt.Errorf("%s: no package", configPath)
	}
	// The same merge as oapi-codegen's Generate (pkg/codegen/codegen.go:162-165).
	g := &generator{types: codegen.DefaultTypeMapping, seen: map[*openapi3.Schema]int{}}
	if cfg.OutputOptions.TypeMapping != nil {
		g.types = codegen.DefaultTypeMapping.Merge(*cfg.OutputOptions.TypeMapping)
	}
	doc, err := util.LoadSwagger(specPath)
	if err != nil {
		return nil, err
	}

	roots := map[string]int{}
	// Paths, methods and properties are visited in sorted order, so the node
	// numbers are the same on every run.
	paths := doc.Paths.Map()
	for _, path := range slices.Sorted(maps.Keys(paths)) {
		ops := paths[path].Operations()
		for _, method := range slices.Sorted(maps.Keys(ops)) {
			body := ops[method].RequestBody
			if body == nil || body.Value == nil {
				continue
			}
			media := body.Value.Content.Get("application/json")
			if media == nil || media.Schema == nil {
				continue
			}
			pattern := strings.ToUpper(method) + " " + path
			root, err := g.add(media.Schema, "")
			if err != nil {
				return nil, fmt.Errorf("%s: %w", pattern, err)
			}
			roots[pattern] = root
		}
	}
	return g.source(cfg.PackageName, filepath.Base(specPath), roots)
}

// add returns the node for s, adding it and its children first. at is the
// JSON path, for error messages.
func (g *generator) add(ref *openapi3.SchemaRef, at string) (int, error) {
	s := ref.Value
	if i, ok := g.seen[s]; ok {
		return i, nil
	}
	fail := func(format string, args ...any) (int, error) {
		where := at
		if where == "" {
			where = "request body"
		}
		return 0, fmt.Errorf("%s: %s", where, fmt.Sprintf(format, args...))
	}
	for _, ext := range []string{"x-go-type", "x-go-type-import"} {
		if _, ok := s.Extensions[ext]; ok {
			return fail("%s bypasses the type-mapping, so the generated Go type is unknown", ext)
		}
	}
	switch {
	case len(s.AllOf) > 0, len(s.OneOf) > 0, s.Not != nil:
		return fail("allOf, oneOf and not are not supported")
	case s.Nullable:
		return fail("nullable is OpenAPI 3.0; write type: [X, 'null']")
	case len(s.AnyOf) > 0:
		return g.addNullable(s, at, fail)
	}

	i := len(g.nodes)
	g.nodes = append(g.nodes, node{}) // reserved first: a schema may refer to itself
	g.seen[s] = i
	n := node{extra: open, items: open}
	if s.Type != nil {
		for _, t := range s.Type.Slice() {
			if !slices.Contains(typeNames, t) {
				return fail("unknown type %q", t)
			}
		}
		n.types = s.Type.Slice()
	}
	if s.Format != "" && s.Type.Includes("string") {
		spec := g.types.String.Resolve(s.Format)
		switch checker, ok := checkers[spec]; {
		case ok:
			n.format = checker
		case spec.Type != "string":
			return fail("format %q is generated as %s, which has no bodyshape checker", s.Format, spec.Type)
		}
	}
	if s.Type.Includes("object") || len(s.Properties) > 0 {
		n.props = map[string]int{}
		for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
			child, err := g.add(s.Properties[name], join(at, name))
			if err != nil {
				return 0, err
			}
			n.props[name] = child
		}
		n.required = slices.Sorted(slices.Values(s.Required))
		switch ap := s.AdditionalProperties; {
		case ap.Schema != nil:
			child, err := g.add(ap.Schema, join(at, "*"))
			if err != nil {
				return 0, err
			}
			n.extra = child
		case ap.Has != nil && !*ap.Has:
			n.extra = closed
		}
	}
	if s.Items != nil {
		child, err := g.add(s.Items, at+"[]")
		if err != nil {
			return 0, err
		}
		n.items = child
	}
	g.nodes[i] = n
	return i, nil
}

// addNullable handles anyOf: [X, {type: 'null'}], "X or null": X's node
// with null added.
func (g *generator) addNullable(s *openapi3.Schema, at string, fail func(string, ...any) (int, error)) (int, error) {
	var other *openapi3.SchemaRef
	nulls := 0
	for _, alt := range s.AnyOf {
		if alt.Value.Type != nil && alt.Value.Type.Is("null") {
			nulls++
			continue
		}
		other = alt
	}
	if len(s.AnyOf) != 2 || nulls != 1 {
		return fail("anyOf is supported only as [X, {type: 'null'}]")
	}
	x, err := g.add(other, at)
	if err != nil {
		return 0, err
	}
	n := g.nodes[x]
	if len(n.types) > 0 && !slices.Contains(n.types, "null") {
		n.types = append(slices.Clone(n.types), "null")
	}
	g.nodes = append(g.nodes, n)
	g.seen[s] = len(g.nodes) - 1
	return len(g.nodes) - 1, nil
}

func join(at, name string) string {
	if at == "" {
		return name
	}
	return at + "." + name
}

// source renders the table as a gofmt-ed Go file.
func (g *generator) source(pkg, specName string, roots map[string]int) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by bodyshapegen from %s. DO NOT EDIT.\n\n", specName)
	fmt.Fprintf(&b, "package %s\n\nimport %q\n\n", pkg, bodyshapeImport)
	b.WriteString("// BodyShapes returns the structure of every JSON request body of this module,\n")
	b.WriteString("// by route pattern. The platform's bodyshape middleware checks each request\n")
	b.WriteString("// body against it before the strict handler decodes it.\n")
	b.WriteString("func BodyShapes() *bodyshape.Table {\n\treturn &bodyshape.Table{\n\t\tNodes: []bodyshape.Node{\n")
	for i, n := range g.nodes {
		fmt.Fprintf(&b, "\t\t\t/* %d */ {Types: %s", i, typeExpr(n.types))
		if n.format != "" {
			fmt.Fprintf(&b, ", Format: bodyshape.%s", n.format)
		}
		fmt.Fprintf(&b, ", Extra: %s, Items: %s", indexExpr(n.extra), indexExpr(n.items))
		if n.props != nil {
			b.WriteString(", Props: map[string]int{")
			for j, name := range slices.Sorted(maps.Keys(n.props)) {
				if j > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%q: %d", name, n.props[name])
			}
			b.WriteString("}")
		}
		if len(n.required) > 0 {
			fmt.Fprintf(&b, ", Required: %#v", n.required)
		}
		b.WriteString("},\n")
	}
	b.WriteString("\t\t},\n\t\tRoots: map[string]int{\n")
	for _, pattern := range slices.Sorted(maps.Keys(roots)) {
		fmt.Fprintf(&b, "\t\t\t%q: %d,\n", pattern, roots[pattern])
	}
	b.WriteString("\t\t},\n\t}\n}\n")
	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, errors.Join(errors.New("format the generated source"), err)
	}
	return src, nil
}

func typeExpr(types []string) string {
	var bits []string
	for _, t := range typeNames {
		if slices.Contains(types, t) {
			bits = append(bits, "bodyshape."+goTypeName(t))
		}
	}
	if len(bits) == 0 {
		return "bodyshape.Any"
	}
	return strings.Join(bits, " | ")
}

func goTypeName(t string) string {
	return strings.ToUpper(t[:1]) + t[1:]
}

func indexExpr(i int) string {
	switch i {
	case closed:
		return "bodyshape.Closed"
	case open:
		return "bodyshape.Open"
	}
	return fmt.Sprint(i)
}
```

`server/tools/bodyshapegen/generate_test.go`：

```go
package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/things.golden")

const (
	thingsSpec = "testdata/things.yaml"
	thingsConf = "testdata/oapi-codegen.yaml"
)

// The golden file is the reviewed output: every supported construct, the
// type-mapping (uuid checked, email left to the domain), a reference into
// another file, and one root per operation with a JSON body.
func TestGenerateMatchesTheGoldenFile(t *testing.T) {
	got, err := generate(thingsSpec, thingsConf)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "things.golden")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("generate() differs from %s (go test -run Golden -update rewrites it):\n%s", golden, got)
	}
}

// Maps are iterated in random order: numbering must not depend on it, or
// make gen-check reports a difference on an unchanged description.
func TestGenerateIsDeterministic(t *testing.T) {
	first, err := generate(thingsSpec, thingsConf)
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		again, err := generate(thingsSpec, thingsConf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again, first) {
			t.Fatal("two runs of generate() on the same input differ")
		}
	}
}

// Without the module template's type-mapping, uuid is generated as
// openapi_types.UUID, which has no checker: the table follows the mapping.
func TestGenerateFollowsTheTypeMapping(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "oapi-codegen.yaml")
	if err := os.WriteFile(conf, []byte("package: gen\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := generate(thingsSpec, conf)

	if err == nil || !strings.Contains(err.Error(), `format "email" is generated as openapi_types.Email`) {
		t.Errorf("generate() = %v, want the default email mapping rejected", err)
	}
}

func TestGenerateRejectsWhatItCannotCheck(t *testing.T) {
	tests := []struct{ name, property, want string }{
		{"date", "{type: string, format: date}", `due: format "date" is generated as openapi_types.Date`},
		{"byte", "{type: string, format: byte}", `due: format "byte" is generated as []byte`},
		{"x-go-type", "{type: string, x-go-type: civil.Date}", "due: x-go-type bypasses the type-mapping"},
		{"x-go-type-import", "{type: string, x-go-type-import: {path: time}}", "due: x-go-type-import bypasses the type-mapping"},
		{"oneOf", "{oneOf: [{type: string}, {type: integer}]}", "due: allOf, oneOf and not are not supported"},
		{"allOf", "{allOf: [{type: string}]}", "due: allOf, oneOf and not are not supported"},
		{"not", "{not: {type: string}}", "due: allOf, oneOf and not are not supported"},
		{"anyOf of two types", "{anyOf: [{type: string}, {type: integer}]}", "due: anyOf is supported only as [X, {type: 'null'}]"},
		{"nullable", "{type: string, nullable: true}", "due: nullable is OpenAPI 3.0"},
		{"nested", "{type: object, properties: {at: {type: array, items: {type: string, format: date}}}}", `due.at[]: format "date"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := filepath.Join(t.TempDir(), "m.yaml")
			src := "openapi: 3.1.0\ninfo: {title: t, version: v0}\npaths:\n  /api/v0/x:\n    post:\n" +
				"      requestBody:\n        content:\n          application/json:\n            schema:\n" +
				"              type: object\n              properties:\n                due: " + tt.property + "\n" +
				"      responses: {'204': {description: none}}\n"
			if err := os.WriteFile(spec, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := generate(spec, thingsConf)

			if err == nil || !strings.Contains(err.Error(), "POST /api/v0/x: "+tt.want) {
				t.Errorf("generate() = %v, want an error containing %q", err, "POST /api/v0/x: "+tt.want)
			}
		})
	}
}
```

测试数据 `server/tools/bodyshapegen/testdata/common.yaml`：

```yaml
openapi: 3.1.0
info:
  title: shared components of the bodyshapegen test
  version: v0
components:
  schemas:
    Tag:
      type: object
      additionalProperties: false
      required: [name]
      properties:
        name:
          type: string
        color:
          type: [string, 'null']
```

`server/tools/bodyshapegen/testdata/things.yaml`：

```yaml
# Every construct bodyshapegen supports, across two operations with a body,
# one without, and a reference into another file.
openapi: 3.1.0
info:
  title: bodyshapegen test
  version: v0
paths:
  /api/v0/things:
    get:
      operationId: listThings
      responses:
        '204':
          description: none
    post:
      operationId: createThing
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateThing'
      responses:
        '204':
          description: none
  /api/v0/things/{thing_id}:
    patch:
      operationId: updateThing
      parameters:
        - name: thing_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              properties:
                name:
                  type: string
                owner_id:
                  type: [string, 'null']
                  format: uuid
                settings:
                  anyOf:
                    - $ref: '#/components/schemas/Settings'
                    - type: 'null'
      responses:
        '204':
          description: none
components:
  schemas:
    CreateThing:
      type: object
      additionalProperties: false
      required: [name, email, settings]
      properties:
        name:
          type: string
        email:
          type: string
          format: email
        count:
          type: integer
        ratio:
          type: number
        due:
          type: string
          format: date-time
        tags:
          type: array
          items:
            $ref: 'common.yaml#/components/schemas/Tag'
        labels:
          type: object
          additionalProperties:
            type: string
        extra:
          type: object
          additionalProperties: true
        anything: {}
        settings:
          $ref: '#/components/schemas/Settings'
    Settings:
      type: object
      additionalProperties: false
      properties:
        notify:
          type: boolean
```

`server/tools/bodyshapegen/testdata/oapi-codegen.yaml`：

```yaml
# The module template's options that matter to bodyshapegen (M2 design 3.12).
package: gen
output: internal/modules/things/adapter/http/gen/server.gen.go
generate:
  models: true
  std-http-server: true
  strict-server: true
output-options:
  nullable-type: true
  type-mapping:
    string:
      formats:
        uuid:
          type: uuid.UUID
          import: uuid
        email:
          type: string
```

黄金文件 `server/tools/bodyshapegen/testdata/things.golden`（也可以由 `go -C server/tools test ./bodyshapegen -run TestGenerateMatchesTheGoldenFile -update` 生成，结果必须与下面逐字节相同）：

```go
// Code generated by bodyshapegen from things.yaml. DO NOT EDIT.

package gen

import "github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"

// BodyShapes returns the structure of every JSON request body of this module,
// by route pattern. The platform's bodyshape middleware checks each request
// body against it before the strict handler decodes it.
func BodyShapes() *bodyshape.Table {
	return &bodyshape.Table{
		Nodes: []bodyshape.Node{
			/* 0 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"anything": 1, "count": 2, "due": 3, "email": 4, "extra": 5, "labels": 6, "name": 8, "ratio": 9, "settings": 10, "tags": 12}, Required: []string{"email", "name", "settings"}},
			/* 1 */ {Types: bodyshape.Any, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 2 */ {Types: bodyshape.Integer, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 3 */ {Types: bodyshape.String, Format: bodyshape.FormatTime, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 4 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 5 */ {Types: bodyshape.Object, Extra: bodyshape.Open, Items: bodyshape.Open, Props: map[string]int{}},
			/* 6 */ {Types: bodyshape.Object, Extra: 7, Items: bodyshape.Open, Props: map[string]int{}},
			/* 7 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 8 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 9 */ {Types: bodyshape.Number, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 10 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"notify": 11}},
			/* 11 */ {Types: bodyshape.Boolean, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 12 */ {Types: bodyshape.Array, Extra: bodyshape.Open, Items: 13},
			/* 13 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"color": 14, "name": 15}, Required: []string{"name"}},
			/* 14 */ {Types: bodyshape.Null | bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 15 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 16 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"name": 17, "owner_id": 18, "settings": 19}},
			/* 17 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 18 */ {Types: bodyshape.Null | bodyshape.String, Format: bodyshape.FormatUUID, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 19 */ {Types: bodyshape.Null | bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"notify": 11}},
		},
		Roots: map[string]int{
			"PATCH /api/v0/things/{thing_id}": 16,
			"POST /api/v0/things":             0,
		},
	}
}
```

- [ ] **Step 4: 整理工具模块的依赖**

Run: `go -C server/tools mod tidy`
Expected: `server/tools/go.mod` 中 `github.com/getkin/kin-openapi v0.142.0`、`github.com/oapi-codegen/oapi-codegen/v2 v2.8.0`、`go.yaml.in/yaml/v3 v3.0.4` 移到第一个 `require` 块、去掉 `// indirect`，版本不变；`go.sum` 不变。再核对 `go 1.27` / `toolchain go1.27.1`。

Run: `go -C server/tools test -count=1 ./...`
Expected: `ok  	github.com/open-nerve/NerveProject/server/tools/bodyshapegen`

- [ ] **Step 5: 修改 `Makefile`**

照下面的差异修改（`gen-go` 在每个模块的 oapi-codegen 之后运行 `bodyshapegen`；`lint-go`、`test` 进入工具模块）：

```diff
--- a/Makefile
+++ b/Makefile
@@ -13,6 +13,8 @@
 
 # 代码生成（见 docs/v0/M0-foundation/specs/P3-api-contract.md）
 OAPI_CODEGEN := go tool -modfile=tools/go.mod oapi-codegen
+# 请求体结构表的生成器在工具模块里（M2 设计 3.11）；go -C 切到 server/tools，所以参数都用绝对路径
+BODYSHAPEGEN := go -C server/tools run ./bodyshapegen
 REDOCLY := REDOCLY_SUPPRESS_UPDATE_NOTICE=true pnpm exec redocly
 # 每个模块一个描述文件 api/modules/<模块>.yaml，生成到该模块的 adapter/http/gen
 API_MODULES := $(basename $(notdir $(wildcard api/modules/*.yaml)))
@@ -70,12 +72,15 @@
 gen: gen-go gen-web ## 重新生成全部代码：Go 接口层、api/dist、TS 客户端
 
 .PHONY: gen-go
-gen-go: ## 由 api/ 生成 Go 接口层（只需要 Go）
+gen-go: ## 由 api/ 生成 Go 接口层和请求体结构表（只需要 Go）
 	rm -f server/internal/platform/httpserver/apigen/*.gen.go server/internal/modules/*/adapter/http/gen/*.gen.go
 	cd server && $(OAPI_CODEGEN) -config internal/platform/httpserver/apigen/oapi-codegen.yaml ../api/common.yaml
 	@set -e; for m in $(API_MODULES); do \
+		gen=$(CURDIR)/server/internal/modules/$$m/adapter/http/gen; \
 		echo "cd server && $(OAPI_CODEGEN) -config internal/modules/$$m/adapter/http/gen/oapi-codegen.yaml ../api/modules/$$m.yaml"; \
 		(cd server && $(OAPI_CODEGEN) -config internal/modules/$$m/adapter/http/gen/oapi-codegen.yaml ../api/modules/$$m.yaml); \
+		echo "$(BODYSHAPEGEN) -config $$gen/oapi-codegen.yaml -out $$gen/bodyshape.gen.go $(CURDIR)/api/modules/$$m.yaml"; \
+		$(BODYSHAPEGEN) -config $$gen/oapi-codegen.yaml -out $$gen/bodyshape.gen.go $(CURDIR)/api/modules/$$m.yaml; \
 	done
 
 .PHONY: gen-web
@@ -98,8 +103,9 @@
 lint: lint-go lint-web ## 运行全部静态检查
 
 .PHONY: lint-go
-lint-go: tools ## 运行 golangci-lint（server）
+lint-go: tools ## 运行 golangci-lint（server 和它的工具模块）
 	cd server && $(GOLANGCI_LINT) run ./...
+	cd server/tools && $(GOLANGCI_LINT) run --config ../.golangci.yml ./...
 
 # 关键词守卫（tools/keywords.mjs，规则在 tools/keywords.json）检查整个仓库，不属于任何工作区包，所以不经过 turbo
 .PHONY: lint-web
@@ -114,10 +120,12 @@
 	pnpm --filter web exec react-router typegen
 	pnpm exec knip --treat-config-hints-as-errors
 
-# go test 的缓存不跟踪 server/ 之外的文件，契约测试读取的 api/dist/openapi.yaml 改了也会重放旧结果，所以不用缓存
+# go test 的缓存不跟踪 server/ 之外的文件，契约测试读取的 api/dist/openapi.yaml 改了也会重放旧结果，所以不用缓存。
+# server/tools 是嵌套的独立模块，server 的 ./... 不进入它，另跑一次（M2 设计 3.11）
 .PHONY: test
-test: ## 运行 Go 测试（server，不用测试缓存）
+test: ## 运行 Go 测试（server 和它的工具模块，不用测试缓存）
 	cd server && go test -count=1 ./...
+	go -C server/tools test -count=1 ./...
 
 .PHONY: test-web
 test-web: ## 运行前端单元测试（各包 test 脚本中的 vitest，经 turbo；需要 Node；持续集成 web 任务）
```

- [ ] **Step 6: 生成 instance 的结构表**

Run: `make gen-go`
Expected: 输出中有 `go -C server/tools run ./bodyshapegen -config …/instance/adapter/http/gen/oapi-codegen.yaml -out …/instance/adapter/http/gen/bodyshape.gen.go …/api/modules/instance.yaml`。新文件 `server/internal/modules/instance/adapter/http/gen/bodyshape.gen.go` 15 行，SHA-256 `58984645db9d05e171021d200a9959593a0f73b86769b9c9f44e49c909399629`（instance 没有请求体，`Roots` 为空）；`server.gen.go` 不变。

- [ ] **Step 7: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（第二段是 `cd server/tools && … run --config ../.golangci.yml ./...`）。

Run: `make test`
Expected: server 的所有包 `ok`，最后一行是 `ok  	github.com/open-nerve/NerveProject/server/tools/bodyshapegen`。

- [ ] **Step 8: 提交**

```bash
git add Makefile server/tools server/internal/platform/httpserver/bodyshape server/internal/modules/instance/adapter/http/gen/bodyshape.gen.go
```
```bash
git commit -m "feat(M2/P1): check request body structure with bodyshape and bodyshapegen

make test and make lint-go also run in the server/tools module.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check-go`
Expected: 没有输出差异，退出码 0。

**Done when:** `bodyshape` 和 `bodyshapegen` 的测试通过；`make test` 的输出里有 `server/tools/bodyshapegen`；`make gen-check-go` 干净。

---

### Task 2: 共享内核、时钟、`TxManager` 与架构测试

**Files:**
- Create: `server/internal/shared/error.go`、`error_test.go`、`actor.go`、`actor_test.go`、`tx.go`
- Create: `server/internal/platform/clock/clock.go`、`clock_test.go`、`clocktest/clocktest.go`
- Create: `server/internal/platform/postgres/tx.go`、`tx_test.go`
- Modify: `server/internal/platform/postgres/pool.go`、`pool_test.go`
- Modify: `server/internal/archtest/rules_test.go`、`rules_cases_test.go`
- Modify: `server/internal/bootstrap/app.go`（过渡：编译期断言；Task 4、11 继续改）

**Interfaces:**
- Produces:
  - `shared.Kind`（8 个）、平台码与字段码常量、`shared.FieldError{Field, Code, Message}`、`shared.Error{Kind, Code, Detail, Fields, RetryDelay}` 及 `NewError`、`Invalid`、`Unauthenticated`、`ServerBusy`；`shared.Actor{UserID, SessionID}`、`WithActor`、`RequireActor`；`shared.TxManager`。
  - `clock.System{}`；`clocktest.At(t) *Fixed`（`Now()`、`Advance(d)`；UTC，截到微秒）。
  - `postgres.NewTxManager(pool, commitTimeout) *TxManager`（`WithinTx`）、`postgres.DB(ctx, pool) Querier`；连接池按 UTC 读 `timestamptz`。
- 规则：平台不导入 `internal/shared`；`adapter/<a>/gen` 只被 `adapter/<a>` 导入；`clocktest` 只被测试导入。

**Tests:**
- `shared/error_test.go`：`TestKindStatus`（8 个 Kind 的状态，未知 Kind 是 500）；`TestConstructors`（四个构造函数的码、detail、字段、`RetryAfter`）；`TestProblemFields`（每个字段满足 `ProblemField`/`ProblemCode`/`Error`）；`TestIsMatchesKindAndCode`。
- `shared/actor_test.go`：`TestRequireActor`、`TestRequireActorWithoutActorIsUnauthenticated`。
- `clock/clock_test.go`：两个实现都满足同一个契约（UTC、截到微秒，`Now` 不倒退）；`TestFixedStandsStillUntilAdvanced`。
- `postgres/tx_test.go`（testcontainers）：提交、出错回滚、panic 回滚、嵌套加入外层、事务之外 `DB` 是连接池、**语句做完后取消 `context` 仍然提交**、**语句失败后取消仍然回滚且连接可以再用**。
- `postgres/pool_test.go`：`TestPoolScansTimestamptzInUTC`（`+08` 的时刻读出为 UTC 的同一时刻，`Location()` 是 `time.UTC`）。
- `archtest/rules_cases_test.go` 的 `TestRules`：新增的导入边——`platform/postgres`、`platform/httpserver` 导入 `shared` 违反规则 4；`adapter/postgres` 导入自己的 `adapter/postgres/gen` 通过，`adapter/http` 或模块根导入它违反规则 6；`bootstrap` 导入 `clocktest` 违反测试工具规则。

- [ ] **Step 1: 写共享内核**

`server/internal/shared/error.go`：

```go
// Package shared is the shared kernel: the few values that cross module
// boundaries (M2 design 3.3). Every module returns its errors as *Error, the
// authentication puts the Actor in the context, and TxManager carries one
// transaction through the repositories of several modules. It imports only
// the standard library, and the platform does not import it: the platform
// declares the small interfaces these types satisfy by structure.
package shared

import "time"

// Kind is the category of a domain error. It decides the HTTP status of the
// problem the error becomes (M2 design 3.11).
type Kind int

// The kinds of domain errors and the status each one maps to.
const (
	KindInvalid         Kind = iota + 1 // 422: a value breaks a domain rule
	KindBadRequest                      // 400: a malformed value, e.g. a cursor
	KindUnauthenticated                 // 401: no valid credential
	KindForbidden                       // 403
	KindNotFound                        // 404: missing, or not visible to the caller
	KindConflict                        // 409
	KindRateLimited                     // 429, with Retry-After
	KindUnavailable                     // 503, with Retry-After
)

// Codes of the platform problems that domain errors carry. Module codes are
// prefixed with the module, e.g. "identity.email_taken".
const (
	CodeValidationFailed = "validation_failed"
	CodeUnauthorized     = "unauthorized"
	CodeServerBusy       = "server_busy"
)

// Field codes: the closed set of FieldError.Code values that clients
// translate (M2 design 3.11). api/common.yaml lists the same set.
const (
	FieldRequired       = "required"
	FieldInvalidFormat  = "invalid_format"
	FieldTooShort       = "too_short"
	FieldTooLong        = "too_long"
	FieldOutOfRange     = "out_of_range"
	FieldNotAllowed     = "not_allowed"
	FieldWeakPassword   = "weak_password"
	FieldCommonPassword = "common_password"
	FieldMustBeFuture   = "must_be_future"
	FieldContainsURL    = "contains_url"
)

// status is the HTTP status of the problem an error of kind k becomes. The
// numbers are literal: shared must not import net/http (architecture rule 10).
func (k Kind) status() int {
	switch k {
	case KindInvalid:
		return 422
	case KindBadRequest:
		return 400
	case KindUnauthenticated:
		return 401
	case KindForbidden:
		return 403
	case KindNotFound:
		return 404
	case KindConflict:
		return 409
	case KindRateLimited:
		return 429
	case KindUnavailable:
		return 503
	}
	return 500
}

// FieldError is the problem with one field of a request. Message is English
// for humans; clients translate Code.
type FieldError struct {
	Field   string // JSON path, e.g. "email" or "onboarding_step.profile_complete"
	Code    string // one of the Field* codes
	Message string
}

func (f FieldError) Error() string        { return f.Message }
func (f FieldError) ProblemField() string { return f.Field }
func (f FieldError) ProblemCode() string  { return f.Code }

// Error is the error every module returns for an expected failure. The
// platform maps it to a problem+json response through the httpserver
// ProblemError interface, which Error satisfies by structure. Carrying the
// HTTP status (ProblemStatus) is the one deliberate trade-off of M2 design
// 3.11: the platform can map it without importing shared.
type Error struct {
	Kind       Kind
	Code       string        // the problem code
	Detail     string        // becomes the problem's detail
	Fields     []FieldError  // the invalid fields, if any
	RetryDelay time.Duration // becomes Retry-After when positive
}

// NewError returns an error of kind with a module code, e.g.
// NewError(KindConflict, "identity.email_taken", "…").
func NewError(kind Kind, code, detail string) *Error {
	return &Error{Kind: kind, Code: code, Detail: detail}
}

// Invalid reports the fields that break domain rules: 422 validation_failed.
func Invalid(fields ...FieldError) *Error {
	return &Error{Kind: KindInvalid, Code: CodeValidationFailed, Detail: "The request has invalid values.", Fields: fields}
}

// Unauthenticated reports a request without a valid credential: 401 unauthorized.
func Unauthenticated() *Error {
	return &Error{Kind: KindUnauthenticated, Code: CodeUnauthorized, Detail: "Authentication is required."}
}

// ServerBusy reports that the server is temporarily overloaded, whoever the
// caller is: 503 server_busy with Retry-After.
func ServerBusy(retry time.Duration) *Error {
	return &Error{Kind: KindUnavailable, Code: CodeServerBusy, Detail: "The server is busy; retry shortly.", RetryDelay: retry}
}

func (e *Error) Error() string { return e.Detail }

// Is makes errors.Is match an error of the same kind and code, so a caller
// can test for, say, identity.email_taken without comparing pointers.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind && t.Code == e.Code
}

// ProblemStatus is the HTTP status of the problem.
func (e *Error) ProblemStatus() int { return e.Kind.status() }

// ProblemCode is the problem's code.
func (e *Error) ProblemCode() string { return e.Code }

// ProblemFields lists the invalid fields; each element has ProblemField and
// ProblemCode.
func (e *Error) ProblemFields() []error {
	out := make([]error, len(e.Fields))
	for i, f := range e.Fields {
		out[i] = f
	}
	return out
}

// RetryAfter is how long the caller should wait before retrying; zero for none.
func (e *Error) RetryAfter() time.Duration { return e.RetryDelay }
```

`server/internal/shared/error_test.go`：

```go
package shared_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestKindStatus(t *testing.T) {
	tests := []struct {
		kind shared.Kind
		want int
	}{
		{shared.KindInvalid, 422},
		{shared.KindBadRequest, 400},
		{shared.KindUnauthenticated, 401},
		{shared.KindForbidden, 403},
		{shared.KindNotFound, 404},
		{shared.KindConflict, 409},
		{shared.KindRateLimited, 429},
		{shared.KindUnavailable, 503},
		{shared.Kind(0), 500},
		{shared.Kind(99), 500},
	}
	for _, tt := range tests {
		if got := shared.NewError(tt.kind, "m.code", "d").ProblemStatus(); got != tt.want {
			t.Errorf("Kind %d: ProblemStatus() = %d, want %d", tt.kind, got, tt.want)
		}
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name   string
		err    *shared.Error
		status int
		code   string
		retry  time.Duration
	}{
		{"Invalid", shared.Invalid(), 422, "validation_failed", 0},
		{"Unauthenticated", shared.Unauthenticated(), 401, "unauthorized", 0},
		{"ServerBusy", shared.ServerBusy(time.Second), 503, "server_busy", time.Second},
		{"NewError", shared.NewError(shared.KindConflict, "identity.email_taken", "taken"), 409, "identity.email_taken", 0},
	}
	for _, tt := range tests {
		if tt.err.ProblemStatus() != tt.status || tt.err.ProblemCode() != tt.code || tt.err.RetryAfter() != tt.retry || tt.err.Error() == "" {
			t.Errorf("%s = %d %q retry %s detail %q, want %d %q retry %s and a detail",
				tt.name, tt.err.ProblemStatus(), tt.err.ProblemCode(), tt.err.RetryAfter(), tt.err.Error(), tt.status, tt.code, tt.retry)
		}
	}
}

func TestProblemFields(t *testing.T) {
	err := shared.Invalid(
		shared.FieldError{Field: "email", Code: shared.FieldInvalidFormat, Message: "not an email address"},
		shared.FieldError{Field: "password", Code: shared.FieldWeakPassword, Message: "too weak"},
	)

	fields := err.ProblemFields()

	if len(fields) != 2 {
		t.Fatalf("ProblemFields() = %v, want 2 fields", fields)
	}
	type field interface {
		error
		ProblemField() string
		ProblemCode() string
	}
	var f field
	if !errors.As(fields[1], &f) || f.ProblemField() != "password" || f.ProblemCode() != "weak_password" || f.Error() != "too weak" {
		t.Errorf("second field = %v, want password weak_password \"too weak\"", fields[1])
	}
}

func TestIsMatchesKindAndCode(t *testing.T) {
	taken := shared.NewError(shared.KindConflict, "identity.email_taken", "taken")
	wrapped := fmt.Errorf("register: %w", shared.NewError(shared.KindConflict, "identity.email_taken", "another detail"))

	if !errors.Is(wrapped, taken) {
		t.Error("errors.Is(wrapped identity.email_taken, identity.email_taken) = false, want true")
	}
	if errors.Is(wrapped, shared.NewError(shared.KindForbidden, "identity.email_taken", "")) {
		t.Error("errors.Is matched another kind")
	}
	if errors.Is(wrapped, shared.NewError(shared.KindConflict, "identity.other", "")) {
		t.Error("errors.Is matched another code")
	}
}
```

`server/internal/shared/actor.go`：

```go
package shared

import (
	"context"
	"uuid"
)

// Actor is the account a request acts as. Authentication puts it in the
// request context; handlers of every module read it with RequireActor. It
// tells which credential authenticated the request, never what kind of
// account it is (v0 design 0.2, principle 1).
type Actor struct {
	UserID    uuid.UUID
	SessionID uuid.UUID // the login session of the access token
}

type actorKey struct{}

// WithActor returns ctx carrying a.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, a)
}

// RequireActor returns the actor of the request, or 401 unauthorized when
// there is none. With the deny-by-default authentication (M2 design 3.6) the
// error only happens when an operation is wired wrongly.
func RequireActor(ctx context.Context) (Actor, error) {
	a, ok := ctx.Value(actorKey{}).(Actor)
	if !ok {
		return Actor{}, Unauthenticated()
	}
	return a, nil
}
```

`server/internal/shared/actor_test.go`：

```go
package shared_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestRequireActor(t *testing.T) {
	want := shared.Actor{UserID: uuid.NewV7(), SessionID: uuid.NewV7()}

	got, err := shared.RequireActor(shared.WithActor(context.Background(), want))

	if err != nil || got != want {
		t.Errorf("RequireActor() = %+v, %v; want %+v", got, err, want)
	}
}

func TestRequireActorWithoutActorIsUnauthenticated(t *testing.T) {
	_, err := shared.RequireActor(context.Background())

	if !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("RequireActor() error = %v, want 401 unauthorized", err)
	}
}
```

`server/internal/shared/tx.go`：

```go
package shared

import "context"

// TxManager runs fn in one database transaction (v0 design 6.4). The
// transaction travels in the context fn receives: the repositories of every
// module run their statements in it, and a nested WithinTx joins it. fn's
// error rolls the transaction back and is returned; otherwise it commits.
// platform/postgres implements it; bootstrap asserts that at compile time.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

- [ ] **Step 2: 写时钟**

`server/internal/platform/clock/clock.go`：

```go
// Package clock is the real time source. Each module declares its own Clock
// port (Now() time.Time) in its app layer; System satisfies it by structure.
package clock

import "time"

// System reads the system clock. Its instants are in UTC and truncated to
// microseconds, the precision of PostgreSQL's timestamptz, so a time a use
// case writes reads back from the database unchanged (M2 design 3.13).
type System struct{}

// Now returns the current instant.
func (System) Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
```

`server/internal/platform/clock/clocktest/clocktest.go`：

```go
// Package clocktest provides a clock that tests set and move by hand. Only
// test code may import it (enforced by internal/archtest).
package clocktest

import "time"

// Fixed stands still until the test moves it. Like clock.System, its instants
// are in UTC and truncated to microseconds. It is not safe for concurrent use.
type Fixed struct {
	now time.Time
}

// At returns a clock stopped at t.
func At(t time.Time) *Fixed {
	return &Fixed{now: normalize(t)}
}

// Now returns the instant the clock stands at.
func (f *Fixed) Now() time.Time {
	return f.now
}

// Advance moves the clock forward by d.
func (f *Fixed) Advance(d time.Duration) {
	f.now = normalize(f.now.Add(d))
}

func normalize(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}
```

`server/internal/platform/clock/clock_test.go`：

```go
package clock_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

type nower interface{ Now() time.Time }

// contract is what every Clock port implementation promises (v0 design 6.1:
// Liskov substitution): UTC, no finer than timestamptz's microseconds, and
// never going backwards.
func contract(t *testing.T, c nower) {
	t.Helper()
	first := c.Now()
	second := c.Now()
	for _, now := range []time.Time{first, second} {
		if now.Location() != time.UTC {
			t.Errorf("Now() = %v, want UTC", now)
		}
		if now.Nanosecond()%1000 != 0 {
			t.Errorf("Now() = %v, want whole microseconds", now)
		}
		if now.IsZero() {
			t.Error("Now() is the zero time")
		}
	}
	if second.Before(first) {
		t.Errorf("Now() went backwards: %v then %v", first, second)
	}
}

func TestSystemFollowsTheClockContract(t *testing.T) {
	contract(t, clock.System{})
}

func TestFixedFollowsTheClockContract(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	contract(t, clocktest.At(time.Date(2026, 9, 25, 18, 0, 0, 123456789, loc)))
}

func TestFixedStandsStillUntilAdvanced(t *testing.T) {
	c := clocktest.At(time.Date(2026, 9, 25, 10, 0, 0, 999, time.UTC))
	start := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	if got := c.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}
	c.Advance(15*time.Minute + time.Nanosecond)
	if got, want := c.Now(), start.Add(15*time.Minute); !got.Equal(want) {
		t.Errorf("Now() after Advance = %v, want %v", got, want)
	}
}
```

- [ ] **Step 3: 写 `TxManager`，连接池改为按 UTC 读出**

`server/internal/platform/postgres/tx.go`（`COMMIT`、`ROLLBACK` 在 `context.WithoutCancel` 下执行，有自己的期限；M2 设计 3.6）：

```go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier runs statements: the transaction WithinTx opened, or the pool.
// sqlc's generated DBTX interface has the same methods, so repositories pass
// DB(ctx, pool) to their generated queries.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// DB returns the transaction that TxManager.WithinTx put in ctx, or pool
// when ctx carries none.
func DB(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// TxManager runs functions in transactions on one pool. It satisfies
// shared.TxManager by structure; the platform does not import shared.
//
// The statements of a transaction run under the caller's context and are
// cancelled with it, e.g. when the request deadline passes. COMMIT and
// ROLLBACK are not: they run under context.WithoutCancel, bounded by their own
// commitTimeout (database.commit_timeout). A transaction whose statements
// have all finished therefore commits even if the request is cancelled
// meanwhile, and a failed one is rolled back and its connection returned to
// the pool in a known state (M2 design 3.6). A COMMIT that gets no answer
// within commitTimeout has an unknown outcome and returns an error.
type TxManager struct {
	pool          *pgxpool.Pool
	commitTimeout time.Duration
}

// NewTxManager returns a TxManager on pool.
func NewTxManager(pool *pgxpool.Pool, commitTimeout time.Duration) *TxManager {
	return &TxManager{pool: pool, commitTimeout: commitTimeout}
}

// WithinTx runs fn in a transaction and commits it when fn returns nil. The
// context fn receives carries the transaction; a nested WithinTx on it runs
// fn in the same transaction. fn's error, or a panic, rolls it back.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	returned := false
	defer func() {
		if !returned { // fn panicked: roll back, then let the panic go on
			_ = m.end(ctx, tx.Rollback)
		}
	}()
	err = fn(context.WithValue(ctx, txKey{}, tx))
	returned = true
	if err != nil {
		if rbErr := m.end(ctx, tx.Rollback); rbErr != nil {
			return errors.Join(err, fmt.Errorf("roll back transaction: %w", rbErr))
		}
		return err
	}
	if err := m.end(ctx, tx.Commit); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// end runs COMMIT or ROLLBACK detached from ctx's cancellation, within the
// commit timeout.
func (m *TxManager) end(ctx context.Context, finish func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.commitTimeout)
	defer cancel()
	return finish(ctx)
}
```

`server/internal/platform/postgres/tx_test.go`：

```go
package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

const commitTimeout = 2 * time.Second

// newNotes returns a pool of at most maxConns connections on a new database
// holding one table, notes.
func newNotes(t *testing.T, maxConns int32) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewEmptyDatabase(t), MaxConns: maxConns})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), "CREATE TABLE notes (id int PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	return pool
}

func insertNote(ctx context.Context, pool *pgxpool.Pool, id int) error {
	_, err := postgres.DB(ctx, pool).Exec(ctx, "INSERT INTO notes (id) VALUES ($1)", id)
	return err
}

func countNotes(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM notes").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWithinTxCommits(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)

	err := tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		return insertNote(ctx, pool, 2)
	})

	if err != nil || countNotes(t, pool) != 2 {
		t.Errorf("WithinTx() = %v with %d notes, want nil and 2", err, countNotes(t, pool))
	}
}

func TestWithinTxRollsBackOnError(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)
	boom := errors.New("boom")

	err := tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		return boom
	})

	if !errors.Is(err, boom) || countNotes(t, pool) != 0 {
		t.Errorf("WithinTx() = %v with %d notes, want boom and 0", err, countNotes(t, pool))
	}
}

func TestWithinTxRollsBackOnPanic(t *testing.T) {
	pool := newNotes(t, 1)
	tm := postgres.NewTxManager(pool, commitTimeout)

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not reach the caller")
			}
		}()
		_ = tm.WithinTx(context.Background(), func(ctx context.Context) error {
			if err := insertNote(ctx, pool, 1); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	// One connection: the count only runs if the rollback released it.
	if n := countNotes(t, pool); n != 0 {
		t.Errorf("notes after a panic = %d, want 0", n)
	}
}

func TestNestedWithinTxJoinsTheOuterTransaction(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)
	boom := errors.New("boom")

	err := tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		inner := tm.WithinTx(ctx, func(ctx context.Context) error {
			var seen int
			// The inner call sees the outer insert: it runs in the same transaction.
			if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM notes").Scan(&seen); err != nil {
				return err
			}
			if seen != 1 {
				t.Errorf("inner transaction sees %d notes, want the outer insert", seen)
			}
			return insertNote(ctx, pool, 2)
		})
		if inner != nil {
			return inner
		}
		return boom
	})

	if !errors.Is(err, boom) || countNotes(t, pool) != 0 {
		t.Errorf("WithinTx() = %v with %d notes, want boom and 0: the outer rollback undoes the inner insert", err, countNotes(t, pool))
	}
}

func TestDBOutsideATransactionIsThePool(t *testing.T) {
	pool := newNotes(t, 1)

	if got := postgres.DB(context.Background(), pool); got != postgres.Querier(pool) {
		t.Errorf("DB() = %T, want the pool", got)
	}
}

// The request deadline cancels statements, never the COMMIT of a transaction
// whose statements have all finished (M2 design 3.6).
func TestWithinTxCommitsAfterTheContextIsCancelled(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)
	ctx, cancel := context.WithCancel(context.Background())

	err := tm.WithinTx(ctx, func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		cancel() // e.g. the request deadline passes after the last statement
		return nil
	})

	if err != nil || countNotes(t, pool) != 1 {
		t.Errorf("WithinTx() = %v with %d notes, want nil and 1", err, countNotes(t, pool))
	}
}

// A failed statement, then a cancelled context: the ROLLBACK still runs, and
// the pool's only connection comes back usable.
func TestWithinTxRollsBackAfterAFailedStatementAndCancel(t *testing.T) {
	pool := newNotes(t, 1)
	tm := postgres.NewTxManager(pool, commitTimeout)
	ctx, cancel := context.WithCancel(context.Background())
	var backend int

	err := tm.WithinTx(ctx, func(ctx context.Context) error {
		if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&backend); err != nil {
			return err
		}
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		err := insertNote(ctx, pool, 1) // duplicate key: the transaction is now aborted
		cancel()
		return err
	})

	if err == nil {
		t.Fatal("WithinTx() = nil, want the duplicate-key error")
	}
	var again int
	if err := pool.QueryRow(context.Background(), "SELECT pg_backend_pid()").Scan(&again); err != nil {
		t.Fatalf("the pool is not usable after the rollback: %v", err)
	}
	if again != backend {
		t.Errorf("backend %d after the rollback, want %d: the connection was not returned to the pool", again, backend)
	}
	if n := countNotes(t, pool); n != 0 {
		t.Errorf("notes = %d, want 0", n)
	}
}
```

`server/internal/platform/postgres/pool.go`：

```go
// Package postgres connects nerve to PostgreSQL and runs its schema
// migrations.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// errUnusableURL replaces pgx's parse error, which quotes the connection
// string and masks the password only on a best-effort basis (it misses the
// legal key/value form "password = secret"); even its inner causes can quote
// pieces of the string. pgx also rejects well-formed strings, e.g. when a
// file they name cannot be read, so the message points at every source.
var errUnusableURL = errors.New("database.url: pgx cannot use it; check its syntax, the files it names " +
	"(sslrootcert, sslcert, sslkey) and any PG* environment variables (details not shown, as they may contain the password)")

// NewPool creates a connection pool for database.url with at most
// database.max_conns connections. It connects lazily: an unreachable database
// shows up on first use, e.g. in the /readyz check. Every connection scans
// timestamptz values in UTC (scanTimestamptzInUTC).
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, errUnusableURL
	}
	pc.MaxConns = cfg.MaxConns
	pc.AfterConnect = scanTimestamptzInUTC
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}

// scanTimestamptzInUTC makes the connection return timestamptz values in UTC.
// pgx returns them in time.Local by default; the API writes every time in UTC
// (v0 design 3.1, M2 design 3.13).
func scanTimestamptzInUTC(_ context.Context, conn *pgx.Conn) error {
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name:  "timestamptz",
		OID:   pgtype.TimestamptzOID,
		Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
	})
	return nil
}
```

`server/internal/platform/postgres/pool_test.go`：

```go
package postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// pgx returns timestamptz in time.Local by default; the pool returns UTC.
func TestPoolScansTimestamptzInUTC(t *testing.T) {
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	var got time.Time

	err := pool.QueryRow(context.Background(), "SELECT '2026-09-25 18:00:00+08'::timestamptz").Scan(&got)

	want := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	if err != nil || got.Location() != time.UTC || !got.Equal(want) {
		t.Errorf("scanned %v (%v), want %v in UTC", got, err, want)
	}
}

func TestNewPoolAppliesMaxConns(t *testing.T) {
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{
		URL:      "postgres://nerve:secret@127.0.0.1:1/nerve",
		MaxConns: 3,
	})
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()
	if got := pool.Config().MaxConns; got != 3 {
		t.Errorf("MaxConns = %d, want 3", got)
	}
}

// pgx quotes the connection string in its parse error and masks the password
// only on a best-effort basis, e.g. not in the legal key/value form
// "password = secret"; NewPool shows none of that text, whether the string is
// malformed or names a file that cannot be read.
func TestNewPoolRejectsUnusableURLWithoutLeakingPassword(t *testing.T) {
	const want = "database.url: pgx cannot use it; check its syntax, the files it names (sslrootcert, sslcert, sslkey) " +
		"and any PG* environment variables (details not shown, as they may contain the password)"
	for _, url := range []string{
		"postgres://nerve:secret@localhost:notaport/nerve",
		"host=localhost port=1 password = secret sslmode=bogus",
		`host=localhost password=sec\ secret sslmode=bogus`,
		"postgres://nerve:secret@localhost/nerve?sslmode=verify-full&sslrootcert=/nonexistent/ca.pem",
	} {
		_, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 1})
		if err == nil || err.Error() != want || strings.Contains(err.Error(), "secret") {
			t.Errorf("NewPool(%q) error = %v, want %q", url, err, want)
		}
	}
}
```

- [ ] **Step 4: 改架构测试**

`server/internal/archtest/rules_test.go`：

```go
// Package archtest enforces the architecture rules of M0 design 3.7 (and the
// platform rules of 3.1) as tests. Each rule is a pure predicate over one
// import edge, so rules are unit-tested on synthetic edges and then applied
// to the real import graph of the module. Two more tests walk transitive
// dependencies: the nerve binary's, for modules it must not link, and those
// of domain, app and internal/shared, for infrastructure.
package archtest

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const modulePath = "github.com/open-nerve/NerveProject/server"

// graph maps each package of the module to the import paths it imports.
// Test files are not part of it.
type graph map[string][]string

type rule struct {
	name      string
	forbidden func(from, to string) bool
}

type violation struct {
	from, to, rule string
}

func (v violation) String() string {
	return fmt.Sprintf("%s imports %s: %s", rel(v.from), rel(v.to), v.rule)
}

func rules() []rule {
	return []rule{
		{"module layers point inward: adapter -> app -> domain", layersPointInward},
		{"domain and app import only the standard library (not net/http or database/sql), their own module's inner layers and internal/shared", innerLayersArePure},
		{"modules do not import each other", modulesAreIsolated},
		{"platform does not import modules, bootstrap or internal/shared", platformIsBusinessFree},
		{"only bootstrap imports modules", onlyBootstrapImportsModules},
		{"generated code is imported only by its own adapter", generatedCodeStaysInAdapter},
		{"platform packages do not import each other, except config", platformPackagesAreIndependent},
		{"test helpers (pgtest, apitest, clocktest) are imported only by tests", testHelpersOnlyInTests},
		{"module packages live in domain, app or adapter, or at the module root", moduleLayoutIsKnown},
		{"internal/shared imports only the standard library (not net/http or database/sql) and internal/shared", sharedKernelIsPure},
	}
}

// check applies every rule to every edge of g, in a stable order.
func check(g graph) []violation {
	var found []violation
	for _, from := range slices.Sorted(maps.Keys(g)) {
		for _, to := range g[from] {
			for _, r := range rules() {
				if r.forbidden(from, to) {
					found = append(found, violation{from, to, r.name})
				}
			}
		}
	}
	return found
}

// local returns the module-relative path of a package of this module.
func local(path string) (string, bool) {
	return strings.CutPrefix(path, modulePath+"/")
}

func rel(path string) string {
	if r, ok := local(path); ok {
		return r
	}
	return path
}

// within reports whether path is dir or inside it.
func within(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+"/")
}

// moduleOf splits internal/modules/<name>/<layer>/... of this module.
func moduleOf(path string) (name, layer string, ok bool) {
	r, ok := local(path)
	if !ok {
		return "", "", false
	}
	rest, ok := strings.CutPrefix(r, "internal/modules/")
	if !ok {
		return "", "", false
	}
	name, sub, _ := strings.Cut(rest, "/")
	layer, _, _ = strings.Cut(sub, "/")
	return name, layer, true
}

// platformOf returns the platform package a path belongs to, e.g. "postgres"
// for internal/platform/postgres/pgtest.
func platformOf(path string) (string, bool) {
	r, ok := local(path)
	if !ok {
		return "", false
	}
	rest, ok := strings.CutPrefix(r, "internal/platform/")
	if !ok {
		return "", false
	}
	name, _, _ := strings.Cut(rest, "/")
	return name, true
}

func isStdlib(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// isInfrastructure reports whether an import outside this module is
// technology the pure layers must not see: any third-party module, net/http
// or database/sql.
func isInfrastructure(path string) bool {
	return !isStdlib(path) || within(path, "net/http") || within(path, "database/sql")
}

func inModuleDir(path, dir string) bool {
	r, ok := local(path)
	return ok && within(r, dir)
}

// layerRank orders the layers of a module from the inside out; "" is the
// module root (module.go).
func layerRank(layer string) (int, bool) {
	switch layer {
	case "domain":
		return 0, true
	case "app":
		return 1, true
	case "adapter":
		return 2, true
	case "":
		return 3, true
	}
	return 0, false
}

func layersPointInward(from, to string) bool {
	fm, fl, ok := moduleOf(from)
	if !ok {
		return false
	}
	tm, tl, ok := moduleOf(to)
	if !ok || fm != tm {
		return false
	}
	fromRank, fromKnown := layerRank(fl)
	toRank, toKnown := layerRank(tl)
	return fromKnown && toKnown && toRank > fromRank
}

// moduleLayoutIsKnown keeps every module package in a layer the other rules
// know. A package elsewhere in a module would escape them: the layer rule
// cannot rank it and the purity rule leaves in-module imports to the layer
// rule, so domain -> issue/transport -> net/http would pass.
func moduleLayoutIsKnown(from, to string) bool {
	return inUnknownLayer(from) || inUnknownLayer(to)
}

func inUnknownLayer(path string) bool {
	if _, layer, ok := moduleOf(path); ok {
		_, known := layerRank(layer)
		return !known
	}
	return false
}

// innerLayersArePure keeps domain and app free of infrastructure: no
// third-party module, no platform package, no net/http or database/sql.
// Imports of modules are judged by the layer, layout and isolation rules,
// which together restrict them to the own module's same or inner layers.
func innerLayersArePure(from, to string) bool {
	if _, layer, ok := moduleOf(from); !ok || (layer != "domain" && layer != "app") {
		return false
	}
	if r, ok := local(to); ok {
		_, _, inModule := moduleOf(to)
		return !inModule && !within(r, "internal/shared")
	}
	return isInfrastructure(to)
}

// sharedKernelIsPure holds internal/shared to the purity of the domain and
// app layers that may import it; otherwise it would carry infrastructure
// into them, as in domain -> shared/x -> net/http.
func sharedKernelIsPure(from, to string) bool {
	if !inModuleDir(from, "internal/shared") {
		return false
	}
	if r, ok := local(to); ok {
		return !within(r, "internal/shared")
	}
	return isInfrastructure(to)
}

func modulesAreIsolated(from, to string) bool {
	fm, _, ok := moduleOf(from)
	if !ok {
		return false
	}
	tm, _, ok := moduleOf(to)
	return ok && fm != tm
}

// platformIsBusinessFree keeps the platform free of business code. That
// includes internal/shared (M2 design 3.3): the platform declares the small
// interfaces it needs (Authenticator, ProblemError) and shared's types
// satisfy them by structure.
func platformIsBusinessFree(from, to string) bool {
	return inModuleDir(from, "internal/platform") &&
		(inModuleDir(to, "internal/modules") || inModuleDir(to, "internal/bootstrap") || inModuleDir(to, "internal/shared"))
}

func onlyBootstrapImportsModules(from, to string) bool {
	return inModuleDir(to, "internal/modules") &&
		!inModuleDir(from, "internal/modules") && !inModuleDir(from, "internal/bootstrap")
}

// generatedCodeStaysInAdapter lets only an adapter import its own generated
// code: modules/<m>/adapter/<a>/gen (oapi-codegen for http, sqlc for
// postgres) is imported by modules/<m>/adapter/<a> and its own subpackages.
func generatedCodeStaysInAdapter(from, to string) bool {
	adapter, ok := generatedCodeOwner(to)
	if !ok {
		return false
	}
	r, _ := local(from)
	return r != adapter && !within(r, adapter+"/gen")
}

// generatedCodeOwner returns the adapter that owns path when path is inside
// internal/modules/<m>/adapter/<a>/gen.
func generatedCodeOwner(path string) (string, bool) {
	r, ok := local(path)
	if !ok {
		return "", false
	}
	parts := strings.Split(r, "/")
	// internal/modules/<m>/adapter/<a>/gen[/...]
	if len(parts) < 6 || parts[0] != "internal" || parts[1] != "modules" || parts[3] != "adapter" || parts[5] != "gen" {
		return "", false
	}
	return strings.Join(parts[:5], "/"), true
}

func platformPackagesAreIndependent(from, to string) bool {
	fp, ok := platformOf(from)
	if !ok {
		return false
	}
	tp, ok := platformOf(to)
	return ok && tp != fp && tp != "config"
}

func testHelpersOnlyInTests(_, to string) bool {
	// The graph holds no test files, so any importer is production code.
	return inModuleDir(to, "internal/platform/postgres/pgtest") ||
		inModuleDir(to, "internal/platform/httpserver/apitest") ||
		inModuleDir(to, "internal/platform/clock/clocktest")
}
```

`server/internal/archtest/rules_cases_test.go`：

```go
package archtest

import (
	"slices"
	"testing"
)

// m turns a module-relative path into a full import path.
func m(rel string) string { return modulePath + "/" + rel }

// TestRules proves every rule both fires and stays quiet, on synthetic edges:
// M0 has no module yet, so the real graph cannot exercise most rules.
func TestRules(t *testing.T) {
	const (
		inward   = "module layers point inward: adapter -> app -> domain"
		layout   = "module packages live in domain, app or adapter, or at the module root"
		pure     = "domain and app import only the standard library (not net/http or database/sql), their own module's inner layers and internal/shared"
		kernel   = "internal/shared imports only the standard library (not net/http or database/sql) and internal/shared"
		isolated = "modules do not import each other"
		business = "platform does not import modules, bootstrap or internal/shared"
		entry    = "only bootstrap imports modules"
		gen      = "generated code is imported only by its own adapter"
		platform = "platform packages do not import each other, except config"
		testOnly = "test helpers (pgtest, apitest, clocktest) are imported only by tests"
	)
	tests := []struct {
		from, to string
		want     []string // violated rules; empty means allowed
	}{
		// Layers inside one module.
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/app"), nil},
		{m("internal/modules/issue/app"), m("internal/modules/issue/domain"), nil},
		{m("internal/modules/issue"), m("internal/modules/issue/adapter/postgres"), nil},
		{m("internal/modules/issue/domain"), m("internal/modules/issue/app"), []string{inward}},
		{m("internal/modules/issue/app"), m("internal/modules/issue/adapter/http"), []string{inward}},
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue"), []string{inward}},
		{m("internal/modules/issue/domain/state"), m("internal/modules/issue/domain"), nil},

		// A package outside the known layers would escape the layer and purity
		// rules, e.g. domain -> issue/transport -> net/http.
		{m("internal/modules/issue/domain"), m("internal/modules/issue/transport"), []string{layout}},
		{m("internal/modules/issue/transport"), "net/http", []string{layout}},
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/helper"), []string{layout}},

		// Purity of domain and app (app -> own domain is allowed, see above).
		{m("internal/modules/issue/domain"), "time", nil},
		{m("internal/modules/issue/domain"), m("internal/shared/id"), nil},
		{m("internal/modules/issue/domain"), "net/http", []string{pure}},
		{m("internal/modules/issue/domain"), "database/sql/driver", []string{pure}},
		{m("internal/modules/issue/domain"), "github.com/jackc/pgx/v5", []string{pure}},
		{m("internal/modules/issue/domain"), m("internal/platform/config"), []string{pure}},
		{m("internal/modules/issue/app"), "context", nil},
		{m("internal/modules/issue/app"), m("internal/shared/id"), nil},
		{m("internal/modules/issue/app"), "net/http", []string{pure}},
		{m("internal/modules/issue/app"), "github.com/jackc/pgx/v5", []string{pure}},
		{m("internal/modules/issue/app"), m("internal/platform/postgres"), []string{pure}},

		// The shared kernel is as pure as the layers that import it, or it would
		// carry infrastructure into them: domain -> shared/x -> net/http.
		{m("internal/shared/id"), "time", nil},
		{m("internal/shared/tx"), m("internal/shared/id"), nil},
		{m("internal/shared/id"), "net/http", []string{kernel}},
		{m("internal/shared/id"), "github.com/jackc/pgx/v5", []string{kernel}},
		{m("internal/shared/id"), m("internal/platform/config"), []string{kernel}},
		{m("internal/shared/id"), m("internal/modules/issue/domain"), []string{entry, kernel}},

		// The platform declares the interfaces shared satisfies; it never imports it.
		{m("internal/platform/postgres"), m("internal/shared"), []string{business}},
		{m("internal/platform/httpserver"), m("internal/shared"), []string{business}},

		// Module isolation and the composition root.
		{m("internal/modules/issue/app"), m("internal/modules/project/domain"), []string{isolated}},
		{m("internal/bootstrap"), m("internal/modules/issue"), nil},
		{m("cmd/nerve"), m("internal/modules/issue"), []string{entry}},
		{m("internal/platform/httpserver"), m("internal/modules/issue"), []string{business, entry}},
		{m("internal/platform/httpserver"), m("internal/bootstrap"), []string{business}},

		// Generated code.
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/adapter/http/gen"), nil},
		{m("internal/modules/issue/adapter/http/gen"), m("internal/platform/httpserver/apigen"), nil},
		{m("internal/modules/issue/app"), m("internal/modules/issue/adapter/http/gen"), []string{inward, gen}},
		{m("internal/modules/issue/adapter/postgres"), m("internal/modules/issue/adapter/http/gen"), []string{gen}},
		{m("internal/modules/project/adapter/http"), m("internal/modules/issue/adapter/http/gen"), []string{isolated, gen}},
		// sqlc's code under adapter/postgres/gen follows the same rule.
		{m("internal/modules/issue/adapter/postgres"), m("internal/modules/issue/adapter/postgres/gen"), nil},
		{m("internal/modules/issue/adapter/postgres/gen"), "github.com/jackc/pgx/v5", nil},
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/adapter/postgres/gen"), []string{gen}},
		{m("internal/modules/issue"), m("internal/modules/issue/adapter/postgres/gen"), []string{gen}},

		// Platform packages.
		{m("internal/platform/logging"), m("internal/platform/config"), nil},
		{m("internal/platform/postgres/pgtest"), m("internal/platform/postgres"), nil},
		{m("internal/platform/httpserver"), m("internal/platform/postgres"), []string{platform}},

		// Test helpers.
		{m("internal/bootstrap"), m("internal/platform/postgres/pgtest"), []string{testOnly}},
		{m("internal/modules/instance/adapter/http"), m("internal/platform/httpserver/apitest"), []string{testOnly}},
		{m("internal/platform/httpserver"), m("internal/platform/httpserver/apitest"), []string{testOnly}},
		{m("internal/bootstrap"), m("internal/platform/clock/clocktest"), []string{testOnly}},
		{m("internal/platform/clock/clocktest"), "time", nil},
	}
	fired := map[string]bool{}
	for _, tt := range tests {
		var got []string
		for _, v := range check(graph{tt.from: {tt.to}}) {
			got = append(got, v.rule)
			fired[v.rule] = true
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s -> %s: violated %q, want %q", rel(tt.from), rel(tt.to), got, tt.want)
		}
	}
	for _, r := range rules() {
		if !fired[r.name] {
			t.Errorf("rule %q has no violating case above", r.name)
		}
	}
}
```

- [ ] **Step 5: `bootstrap` 的编译期断言**

平台的 `TxManager` 按结构满足 `shared.TxManager`，两边互不导入；在组合根断言（M2 设计 3.3）。照下面的差异修改 `server/internal/bootstrap/app.go`：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -15,8 +15,13 @@
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
 	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
 	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
+	"github.com/open-nerve/NerveProject/server/internal/shared"
 )
 
+// The platform and the shared kernel meet here by structure: neither imports
+// the other (M2 design 3.3, 3.11).
+var _ shared.TxManager = (*postgres.TxManager)(nil)
+
 // app is a fully wired nerve server.
 type app struct {
 	cfg      config.Config
```

- [ ] **Step 6: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，其中有 `internal/shared`、`internal/platform/clock`、`internal/platform/postgres`、`internal/archtest`。

- [ ] **Step 7: 提交**

```bash
git add server/internal/shared server/internal/platform/clock server/internal/platform/postgres server/internal/archtest server/internal/bootstrap/app.go
```
```bash
git commit -m "feat(M2/P1): shared kernel, clock and a TxManager whose COMMIT outlives the request

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestWithinTxCommitsAfterTheContextIsCancelled`、`TestWithinTxRollsBackAfterAFailedStatementAndCancel`、`TestPoolScansTimestamptzInUTC` 和 `TestRules` 通过；`server/internal/platform` 下没有任何包导入 `internal/shared`。

---

### Task 3: 配置

**Files:**
- Modify: `server/internal/platform/config/config.go`、`load.go`、`validate.go`、`config_test.go`、`load_test.go`、`validate_test.go`
- Modify: `server/configs/config.yaml`、`config.dev.yaml`、`config.test.yaml`、`embed_test.go`

**Interfaces:**
- Produces:
  - `config.ServerConfig` 加 `RequestTimeout`、`MaxBodyBytes`、`AddrFile`；`config.DatabaseConfig` 加 `CommitTimeout`；
  - `config.AuthConfig{SignupEnabled, AccessTokenTTL, SessionTTL, JWT JWTConfig{PrivateKeyFile}, Password PasswordConfig{Argon2MemoryKiB, Argon2Iterations, Argon2Parallelism, MaxConcurrentHashes, MaxWait}}`，`Config.Auth`；
  - 键名、默认值和校验见 spec 2.7。
- 环境变量给非字符串的键传空值时报错（M0-P2 交接 8）；`LogValue` 对 `*_file` 只记是否设置（M0-P2 交接 4）。

**Tests:**
- `config_test.go`：新增 `TestLogValueHidesFilePaths`（`addr_file_set`、`private_key_file_set` 为 `true`，日志里没有路径）。
- `load_test.go`：`TestLoadAppliesLayersInOrder` 覆盖新键和两个环境变量；`TestLoadErrors` 新增三个情况：空的布尔、数字、时长环境变量报 `'<键>' must not be empty`。
- `validate_test.go`：`TestValidateReportsEveryInvalidKey` 覆盖每个新键的错误原文；`TestValidateCrossKeyRules`（原来的读请求头超时一条并入，另加 `request_timeout` 不小于 `write_timeout`、`session_ttl` 不长于 `access_token_ttl`、argon2 内存低于每条并行 8 KiB、prod 缺签名密钥）；`TestValidateAcceptsProdWithASigningKeyFile`。
- `server/configs/embed_test.go`：`TestBuiltInProfiles` 三个环境逐项相等（**prod 的 `SignupEnabled` 为 `false`**，test 的 argon2 为 64 KiB、1 次）；`TestProdRequiresDatabaseURLAndSigningKey` 同时报出两个缺失的键。

- [ ] **Step 1: 写配置类型、加载和校验**

`server/internal/platform/config/config.go`：

```go
// Package config loads, validates and describes nerve's configuration.
package config

import (
	"log/slog"
	"time"
)

// Profiles, selected with NERVE_ENV.
const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

// Config is the effective configuration of a nerve process.
type Config struct {
	// Env is the profile the configuration was loaded for. It comes from
	// NERVE_ENV and is not a configuration key.
	Env      string         `koanf:"-"`
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	Auth     AuthConfig     `koanf:"auth"`
	Log      LogConfig      `koanf:"log"`
}

// ServerConfig configures the HTTP server. The timeouts bound the reads and
// writes on a connection: reading the request headers, reading the whole
// request (headers and body), and writing the response; idle keep-alive
// connections have a fixed timeout in httpserver. RequestTimeout bounds each
// API request's context, which write_timeout does not cancel.
type ServerConfig struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ReadTimeout       time.Duration `koanf:"read_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
	RequestTimeout    time.Duration `koanf:"request_timeout"`
	MaxBodyBytes      int64         `koanf:"max_body_bytes"`
	// AddrFile, when set, receives the address the server listens on once it
	// does, e.g. for addr ":0".
	AddrFile string `koanf:"addr_file"`
}

// DatabaseConfig configures the PostgreSQL pool and schema migrations.
type DatabaseConfig struct {
	URL         string `koanf:"url"`
	MaxConns    int32  `koanf:"max_conns"`
	AutoMigrate bool   `koanf:"auto_migrate"`
	// CommitTimeout bounds COMMIT and ROLLBACK, which the request deadline
	// does not cancel.
	CommitTimeout time.Duration `koanf:"commit_timeout"`
}

// AuthConfig configures accounts and credentials (M2 design 6.5).
type AuthConfig struct {
	SignupEnabled  bool           `koanf:"signup_enabled"`
	AccessTokenTTL time.Duration  `koanf:"access_token_ttl"`
	SessionTTL     time.Duration  `koanf:"session_ttl"`
	JWT            JWTConfig      `koanf:"jwt"`
	Password       PasswordConfig `koanf:"password"`
}

// JWTConfig locates the Ed25519 signing key.
type JWTConfig struct {
	// PrivateKeyFile is a PKCS#8 PEM Ed25519 private key. Required in prod;
	// empty elsewhere means an ephemeral key generated at startup.
	PrivateKeyFile string `koanf:"private_key_file"`
}

// PasswordConfig configures argon2id and how many hashes run at once.
type PasswordConfig struct {
	Argon2MemoryKiB     uint32        `koanf:"argon2_memory_kib"`
	Argon2Iterations    uint32        `koanf:"argon2_iterations"`
	Argon2Parallelism   uint8         `koanf:"argon2_parallelism"`
	MaxConcurrentHashes int           `koanf:"max_concurrent_hashes"`
	MaxWait             time.Duration `koanf:"max_wait"`
}

// LogConfig configures the process logger.
type LogConfig struct {
	Level  string `koanf:"level"`  // debug, info, warn or error
	Format string `koanf:"format"` // text or json
}

// LogValue renders the configuration for logs with secrets masked, so the
// effective configuration can be logged at startup. Only the keys listed here
// reach the log; every *_file key logs whether it is set, never its path.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.Group("server",
			slog.String("addr", c.Server.Addr),
			slog.Duration("read_header_timeout", c.Server.ReadHeaderTimeout),
			slog.Duration("read_timeout", c.Server.ReadTimeout),
			slog.Duration("write_timeout", c.Server.WriteTimeout),
			slog.Duration("shutdown_timeout", c.Server.ShutdownTimeout),
			slog.Duration("request_timeout", c.Server.RequestTimeout),
			slog.Int64("max_body_bytes", c.Server.MaxBodyBytes),
			slog.Bool("addr_file_set", c.Server.AddrFile != ""),
		),
		slog.Any("database", c.Database),
		slog.Group("auth",
			slog.Bool("signup_enabled", c.Auth.SignupEnabled),
			slog.Duration("access_token_ttl", c.Auth.AccessTokenTTL),
			slog.Duration("session_ttl", c.Auth.SessionTTL),
			slog.Group("jwt",
				slog.Bool("private_key_file_set", c.Auth.JWT.PrivateKeyFile != ""),
			),
			slog.Group("password",
				slog.Uint64("argon2_memory_kib", uint64(c.Auth.Password.Argon2MemoryKiB)),
				slog.Uint64("argon2_iterations", uint64(c.Auth.Password.Argon2Iterations)),
				slog.Uint64("argon2_parallelism", uint64(c.Auth.Password.Argon2Parallelism)),
				slog.Int("max_concurrent_hashes", c.Auth.Password.MaxConcurrentHashes),
				slog.Duration("max_wait", c.Auth.Password.MaxWait),
			),
		),
		slog.Group("log",
			slog.String("level", c.Log.Level),
			slog.String("format", c.Log.Format),
		),
	)
}

// redacted stands in for a secret in log output.
const redacted = "xxxxx"

// LogValue renders the database settings with the URL masked as a whole, so
// they are safe to log on their own too. pgx parses the URL with its own
// libpq-compatible grammar, which accepts forms that other parsers read
// differently, so masking only the password another parser finds could leak
// the rest; log the target from the parsed pool configuration instead.
func (d DatabaseConfig) LogValue() slog.Value {
	url := ""
	if d.URL != "" {
		url = redacted
	}
	return slog.GroupValue(
		slog.String("url", url),
		slog.Int("max_conns", int(d.MaxConns)),
		slog.Bool("auto_migrate", d.AutoMigrate),
		slog.Duration("commit_timeout", d.CommitTimeout),
	)
}
```

`server/internal/platform/config/load.go`：

```go
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	// Control variables: they steer loading and are never configuration keys.
	envProfile   = "NERVE_ENV"
	envConfigDir = "NERVE_CONFIG_DIR"

	envPrefix = "NERVE_"
	envKeySep = "__" // separates levels: NERVE_DATABASE__MAX_CONNS -> database.max_conns
	baseFile  = "config.yaml"
)

// Sources tells Load where configuration comes from.
type Sources struct {
	// Embedded holds the built-in config.yaml and config.<env>.yaml files.
	Embedded fs.FS
	// Environ is the process environment in os.Environ form.
	Environ []string
	// LocalFile is the personal override file. It is read only in the dev
	// profile, and only when it exists.
	LocalFile string
}

// Load builds the effective configuration. Each layer overrides the ones
// before it:
//
//  1. the built-in config.yaml, which lists every key with its default;
//  2. the built-in config.<env>.yaml;
//  3. config.yaml, then config.<env>.yaml, in $NERVE_CONFIG_DIR, when set and present;
//  4. LocalFile, in the dev profile only, when present;
//  5. NERVE_<SECTION>__<KEY> environment variables.
//
// The result is validated; the error lists every invalid key.
func Load(src Sources) (Config, error) {
	profile := lookupEnv(src.Environ, envProfile)
	if profile == "" {
		profile = EnvDev
	}
	if !slices.Contains([]string{EnvDev, EnvTest, EnvProd}, profile) {
		return Config{}, fmt.Errorf("%s must be one of dev, test, prod, got %q", envProfile, profile)
	}
	profileFile := "config." + profile + ".yaml"

	k := koanf.New(".")
	for _, name := range []string{baseFile, profileFile} {
		data, err := fs.ReadFile(src.Embedded, name)
		if err != nil {
			return Config{}, fmt.Errorf("read built-in %s: %w", name, err)
		}
		if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
			return Config{}, fmt.Errorf("parse built-in %s: %w", name, err)
		}
	}
	if dir := lookupEnv(src.Environ, envConfigDir); dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return Config{}, fmt.Errorf("%s=%s is not a directory", envConfigDir, dir)
		}
		for _, name := range []string{baseFile, profileFile} {
			if err := loadFileIfExists(k, filepath.Join(dir, name)); err != nil {
				return Config{}, err
			}
		}
	}
	if profile == EnvDev && src.LocalFile != "" {
		if err := loadFileIfExists(k, src.LocalFile); err != nil {
			return Config{}, err
		}
	}
	environ := env.Provider(".", env.Opt{
		Prefix:        envPrefix,
		TransformFunc: envKey,
		EnvironFunc:   func() []string { return src.Environ },
	})
	if err := k.Load(environ, nil); err != nil {
		return Config{}, fmt.Errorf("read environment: %w", err)
	}

	cfg := Config{Env: profile}
	if err := decode(k, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid configuration:\n%w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration:\n%w", err)
	}
	return cfg, nil
}

func lookupEnv(environ []string, name string) string {
	var value string
	for _, kv := range environ {
		if k, v, ok := strings.Cut(kv, "="); ok && k == name {
			value = v
		}
	}
	return value
}

// loadFileIfExists merges the YAML file at path into k and skips a missing
// file. It reads the file itself rather than through koanf's file provider,
// which would link fsnotify into the binary for a watch nerve never uses.
func loadFileIfExists(k *koanf.Koanf, path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	return nil
}

// envKey maps NERVE_DATABASE__MAX_CONNS to database.max_conns. Variables
// without the "__" separator (NERVE_ENV, NERVE_CONFIG_DIR, NERVE_DEV_DB_PORT
// and the like) are not configuration keys and are skipped: every key lives in
// a section, so a real key always has a separator.
func envKey(name, value string) (string, any) {
	key := strings.TrimPrefix(name, envPrefix)
	if !strings.Contains(key, envKeySep) {
		return "", nil
	}
	return strings.ToLower(strings.ReplaceAll(key, envKeySep, ".")), value
}

// decode copies the merged layers into cfg. Unknown keys are errors, so a typo
// never silently falls back to a default.
func decode(k *koanf.Koanf, cfg *Config) error {
	return k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.ComposeDecodeHookFunc(emptyValueHook, durationHook),
			ErrorUnused:      true,
			WeaklyTypedInput: true, // environment values are strings
		},
	})
}

// emptyValueHook rejects an empty string for a key that is not a string.
// Weakly typed decoding would otherwise turn NERVE_AUTH__SIGNUP_ENABLED=
// into false and NERVE_DATABASE__MAX_CONNS= into 0 without a word.
func emptyValueHook(from, to reflect.Type, data any) (any, error) {
	if from.Kind() != reflect.String || to.Kind() == reflect.String || data != "" {
		return data, nil
	}
	switch to.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return nil, errors.New("must not be empty")
	}
	return data, nil
}

// durationHook decodes Go duration strings such as "5s". Bare numbers are
// rejected: a YAML 5 would otherwise silently mean 5ns.
func durationHook(_ reflect.Type, to reflect.Type, data any) (any, error) {
	if to != reflect.TypeFor[time.Duration]() {
		return data, nil
	}
	s, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("must be a duration such as \"5s\", got %v", data)
	}
	return time.ParseDuration(s)
}
```

`server/internal/platform/config/validate.go`：

```go
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
)

// validate reports every invalid key at once, one "key: problem" line each.
func (c Config) validate() error {
	var errs []error
	fail := func(key, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: "+format, append([]any{key}, args...)...))
	}

	if _, _, err := net.SplitHostPort(c.Server.Addr); err != nil {
		fail("server.addr", "must be host:port, e.g. \":8080\", got %q", c.Server.Addr)
	}
	switch {
	case c.Server.ReadHeaderTimeout <= 0:
		fail("server.read_header_timeout", "must be positive, got %s", c.Server.ReadHeaderTimeout)
	case c.Server.ReadTimeout > 0 && c.Server.ReadHeaderTimeout > c.Server.ReadTimeout:
		// read_timeout is the budget for the whole request: a longer header limit
		// would let the headers alone outlast it and leave the body no time.
		fail("server.read_header_timeout", "must not exceed server.read_timeout (%s), got %s", c.Server.ReadTimeout, c.Server.ReadHeaderTimeout)
	}
	if c.Server.ReadTimeout <= 0 {
		fail("server.read_timeout", "must be positive, got %s", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout <= 0 {
		fail("server.write_timeout", "must be positive, got %s", c.Server.WriteTimeout)
	}
	if c.Server.ShutdownTimeout <= 0 {
		fail("server.shutdown_timeout", "must be positive, got %s", c.Server.ShutdownTimeout)
	}
	switch {
	case c.Server.RequestTimeout <= 0:
		fail("server.request_timeout", "must be positive, got %s", c.Server.RequestTimeout)
	case c.Server.WriteTimeout > 0 && c.Server.RequestTimeout >= c.Server.WriteTimeout:
		// A request that runs into its deadline still has to write its error
		// response before write_timeout cuts the connection.
		fail("server.request_timeout", "must be less than server.write_timeout (%s), got %s", c.Server.WriteTimeout, c.Server.RequestTimeout)
	}
	if c.Server.MaxBodyBytes < 1 {
		fail("server.max_body_bytes", "must be at least 1, got %d", c.Server.MaxBodyBytes)
	}
	if c.Database.URL == "" {
		fail("database.url", "is required")
	}
	if c.Database.MaxConns < 1 {
		fail("database.max_conns", "must be at least 1, got %d", c.Database.MaxConns)
	}
	if c.Database.CommitTimeout <= 0 {
		fail("database.commit_timeout", "must be positive, got %s", c.Database.CommitTimeout)
	}
	c.Auth.validate(c.Env, fail)
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
		fail("log.level", "must be one of debug, info, warn, error, got %q", c.Log.Level)
	}
	if c.Log.Format != "text" && c.Log.Format != "json" {
		fail("log.format", "must be text or json, got %q", c.Log.Format)
	}
	return errors.Join(errs...)
}

func (a AuthConfig) validate(env string, fail func(key, format string, args ...any)) {
	if a.AccessTokenTTL <= 0 {
		fail("auth.access_token_ttl", "must be positive, got %s", a.AccessTokenTTL)
	}
	switch {
	case a.SessionTTL <= 0:
		fail("auth.session_ttl", "must be positive, got %s", a.SessionTTL)
	case a.SessionTTL <= a.AccessTokenTTL:
		fail("auth.session_ttl", "must be longer than auth.access_token_ttl (%s), got %s", a.AccessTokenTTL, a.SessionTTL)
	}
	if env == EnvProd && a.JWT.PrivateKeyFile == "" {
		// The file itself is read when nerve starts (bootstrap), not here.
		fail("auth.jwt.private_key_file", "is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519")
	}
	p := a.Password
	// golang.org/x/crypto/argon2 panics below one iteration or one lane, and
	// silently raises memory below 8 KiB per lane: reject those instead.
	if p.Argon2Iterations < 1 {
		fail("auth.password.argon2_iterations", "must be at least 1, got %d", p.Argon2Iterations)
	}
	if p.Argon2Parallelism < 1 {
		fail("auth.password.argon2_parallelism", "must be at least 1, got %d", p.Argon2Parallelism)
	}
	if minMemory := 8 * uint32(p.Argon2Parallelism); p.Argon2MemoryKiB < max(minMemory, 8) {
		fail("auth.password.argon2_memory_kib", "must be at least 8 per lane (%d), got %d", max(minMemory, 8), p.Argon2MemoryKiB)
	}
	if p.MaxConcurrentHashes < 1 {
		fail("auth.password.max_concurrent_hashes", "must be at least 1, got %d", p.MaxConcurrentHashes)
	}
	if p.MaxWait <= 0 {
		fail("auth.password.max_wait", "must be positive, got %s", p.MaxWait)
	}
}
```

- [ ] **Step 2: 写测试**

`server/internal/platform/config/config_test.go`：

```go
package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogValueMasksDatabaseURL(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", validConfig())

	out := buf.String()
	if strings.Contains(out, "secret") {
		t.Errorf("log output leaks the password: %s", out)
	}
	for _, want := range []string{
		"config.env=test",
		"config.server.addr=:8080",
		"config.server.read_header_timeout=5s",
		"config.server.read_timeout=30s",
		"config.server.write_timeout=1m0s",
		"config.server.shutdown_timeout=20s",
		"config.server.request_timeout=15s",
		"config.server.max_body_bytes=1048576",
		"config.server.addr_file_set=false",
		"config.database.url=xxxxx",
		"config.database.max_conns=10",
		"config.database.commit_timeout=2s",
		"config.auth.signup_enabled=true",
		"config.auth.access_token_ttl=15m0s",
		"config.auth.session_ttl=720h0m0s",
		"config.auth.jwt.private_key_file_set=false",
		"config.auth.password.argon2_memory_kib=19456",
		"config.auth.password.argon2_iterations=2",
		"config.auth.password.argon2_parallelism=1",
		"config.auth.password.max_concurrent_hashes=4",
		"config.auth.password.max_wait=2s",
		"config.log.format=json",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output lacks %q: %s", want, out)
		}
	}
}

// Every *_file key logs whether it is set, never the path (M0-P2 handoff 4,
// M2 design 3.7): the path of a key file tells where secrets live.
func TestLogValueHidesFilePaths(t *testing.T) {
	cfg := validConfig()
	cfg.Server.AddrFile = "/run/nerve/addr-secret-dir"
	cfg.Auth.JWT.PrivateKeyFile = "/etc/nerve/secret-dir/jwt.pem"
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", cfg)

	out := buf.String()
	if strings.Contains(out, "secret-dir") {
		t.Errorf("log output shows a file path: %s", out)
	}
	for _, want := range []string{"config.server.addr_file_set=true", "config.auth.jwt.private_key_file_set=true"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output lacks %q: %s", want, out)
		}
	}
}

// The URL is masked as a whole, whatever its form: pgx parses it with its own
// libpq-compatible grammar, which accepts forms other URL parsers misread.
func TestDatabaseConfigLogValueMasksTheWholeURL(t *testing.T) {
	for _, url := range []string{
		"postgres://nerve:secret@localhost:5432/nerve",
		"postgres://localhost/nerve?password=secret&sslmode=disable",
		"postgres://localhost/nerve?password=secret;more&sslmode=disable", // raw ';': net/url drops the pair, pgx keeps it
		"postgres://localhost/nerve?pass%77ord=secret",
		"postgres://localhost/nerve?sslpassword=secret",
		"postgres://nerve:pa@ss@localhost/nerve?password=a%ZZsecret",
		"host=localhost user=nerve password=secret",
	} {
		var buf bytes.Buffer
		slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "db", DatabaseConfig{URL: url, MaxConns: 10})

		out := buf.String()
		if strings.Contains(out, "secret") || !strings.Contains(out, "db.url=xxxxx ") {
			t.Errorf("DatabaseConfig{URL: %q} logs %s; want db.url=xxxxx and no password", url, out)
		}
		for _, want := range []string{"db.max_conns=10", "db.auto_migrate=false"} {
			if !strings.Contains(out, want) {
				t.Errorf("log output lacks %q: %s", want, out)
			}
		}
	}
}

func TestDatabaseConfigLogValueShowsAnEmptyURL(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("x", "db", DatabaseConfig{})

	if out := buf.String(); !strings.Contains(out, `db.url="" `) {
		t.Errorf("log output = %s, want an empty db.url", out)
	}
}
```

`server/internal/platform/config/load_test.go`：

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const testBase = `
server:
  addr: ":8080"
  read_header_timeout: 5s
  read_timeout: 30s
  write_timeout: 60s
  shutdown_timeout: 20s
  request_timeout: 15s
  max_body_bytes: 1048576
  addr_file: ""
database:
  url: ""
  max_conns: 10
  auto_migrate: true
  commit_timeout: 2s
auth:
  signup_enabled: false
  access_token_ttl: 15m
  session_ttl: 720h
  jwt:
    private_key_file: ""
  password:
    argon2_memory_kib: 19456
    argon2_iterations: 2
    argon2_parallelism: 1
    max_concurrent_hashes: 4
    max_wait: 2s
log:
  level: info
  format: json
`

func embedded(profiles map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{"config.yaml": {Data: []byte(testBase)}}
	for name, content := range profiles {
		fsys["config."+name+".yaml"] = &fstest.MapFile{Data: []byte(content)}
	}
	return fsys
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAppliesLayersInOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "log:\n  format: text\ndatabase:\n  max_conns: 20\n")
	writeFile(t, dir, "config.dev.yaml", "database:\n  max_conns: 30\nserver:\n  shutdown_timeout: 30s\n")
	local := writeFile(t, t.TempDir(), "config.local.yaml", "server:\n  shutdown_timeout: 40s\n  read_header_timeout: 7s\n")

	cfg, err := Load(Sources{
		Embedded: embedded(map[string]string{"dev": "database:\n  url: postgres://embedded-dev\nlog:\n  level: debug\n"}),
		Environ: []string{
			"NERVE_CONFIG_DIR=" + dir,
			"NERVE_SERVER__READ_HEADER_TIMEOUT=9s",
			"NERVE_DATABASE__AUTO_MIGRATE=false",
			"NERVE_AUTH__SIGNUP_ENABLED=true",
			"NERVE_AUTH__PASSWORD__ARGON2_MEMORY_KIB=64",
		},
		LocalFile: local,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		Env: EnvDev, // NERVE_ENV defaults to dev
		Server: ServerConfig{
			Addr:              ":8080",         // built-in config.yaml
			ReadHeaderTimeout: 9 * time.Second, // environment beats config.local.yaml
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			ShutdownTimeout:   40 * time.Second,
			RequestTimeout:    15 * time.Second,
			MaxBodyBytes:      1048576,
		},
		Database: DatabaseConfig{
			URL:           "postgres://embedded-dev", // built-in config.dev.yaml
			MaxConns:      30,                        // config dir: config.dev.yaml beats config.yaml
			AutoMigrate:   false,                     // environment
			CommitTimeout: 2 * time.Second,
		},
		Auth: AuthConfig{
			SignupEnabled:  true, // environment
			AccessTokenTTL: 15 * time.Minute,
			SessionTTL:     720 * time.Hour,
			Password: PasswordConfig{
				Argon2MemoryKiB:     64, // environment
				Argon2Iterations:    2,
				Argon2Parallelism:   1,
				MaxConcurrentHashes: 4,
				MaxWait:             2 * time.Second,
			},
		},
		Log: LogConfig{Level: "debug", Format: "text"},
	}
	if cfg != want {
		t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
	}
}

func TestLoadSelectsProfile(t *testing.T) {
	local := writeFile(t, t.TempDir(), "config.local.yaml", "log:\n  level: error\n")
	cfg, err := Load(Sources{
		Embedded: embedded(map[string]string{"test": "log:\n  level: warn\n"}),
		Environ:  []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://from-env"},
		// Only the dev profile reads the local file.
		LocalFile: local,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != EnvTest || cfg.Log.Level != "warn" || cfg.Database.URL != "postgres://from-env" {
		t.Errorf("Load() = %+v, want test profile, level warn and the URL from the environment", cfg)
	}
}

func TestLoadIgnoresMissingOptionalFiles(t *testing.T) {
	_, err := Load(Sources{
		Embedded:  embedded(map[string]string{"dev": "database:\n  url: postgres://embedded-dev\n"}),
		Environ:   []string{"NERVE_CONFIG_DIR=" + t.TempDir()},
		LocalFile: filepath.Join(t.TempDir(), "config.local.yaml"),
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadSkipsVariablesThatAreNotKeys(t *testing.T) {
	cfg, err := Load(Sources{
		Embedded: embedded(map[string]string{"prod": ""}),
		Environ: []string{
			"NERVE_ENV=prod",
			"NERVE_CONFIG_DIR=" + t.TempDir(),
			"NERVE_DEV_DB_PORT=55433",
			"NERVE_DATABASE__URL=postgres://from-env",
			"NERVE_AUTH__JWT__PRIVATE_KEY_FILE=/etc/nerve/jwt.pem",
			"OTHER__VAR=ignored",
		},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.URL != "postgres://from-env" {
		t.Errorf("database.url = %q, want the value of NERVE_DATABASE__URL", cfg.Database.URL)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		dirFile string // content of config.yaml in NERVE_CONFIG_DIR, if any
		want    string
	}{
		{
			name:    "unknown profile",
			environ: []string{"NERVE_ENV=staging"},
			want:    `NERVE_ENV must be one of dev, test, prod, got "staging"`,
		},
		{
			name:    "config dir does not exist",
			environ: []string{"NERVE_ENV=test", "NERVE_CONFIG_DIR=/does/not/exist"},
			want:    "NERVE_CONFIG_DIR=/does/not/exist is not a directory",
		},
		{
			name:    "required key missing",
			environ: []string{"NERVE_ENV=test"},
			want:    "invalid configuration:\ndatabase.url: is required",
		},
		{
			name:    "unknown key in the environment",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_DATABSE__URL=postgres://x"},
			want:    "has invalid keys: databse",
		},
		{
			name:    "unknown key in a file",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
			dirFile: "database:\n  max_con: 5\n",
			want:    "'database' has invalid keys: max_con",
		},
		{
			name:    "malformed duration",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_SERVER__SHUTDOWN_TIMEOUT=soon"},
			want:    `'server.shutdown_timeout' time: invalid duration "soon"`,
		},
		{
			name:    "number where a duration is expected",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
			dirFile: "server:\n  shutdown_timeout: 20\n",
			want:    `'server.shutdown_timeout' must be a duration such as "5s", got 20`,
		},
		{
			name:    "malformed number",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_DATABASE__MAX_CONNS=many"},
			want:    "'database.max_conns'",
		},
		{
			name:    "malformed YAML",
			environ: []string{"NERVE_ENV=test"},
			dirFile: "server: [",
			want:    "config.yaml: yaml:",
		},
		// Weakly typed decoding would read an empty value as false or 0 (M0-P2 handoff 8).
		{
			name:    "empty boolean in the environment",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_AUTH__SIGNUP_ENABLED="},
			want:    "'auth.signup_enabled' must not be empty",
		},
		{
			name:    "empty number in the environment",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_DATABASE__MAX_CONNS="},
			want:    "'database.max_conns' must not be empty",
		},
		{
			name:    "empty duration in the environment",
			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_AUTH__SESSION_TTL="},
			want:    "'auth.session_ttl' must not be empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environ := tt.environ
			if tt.dirFile != "" {
				dir := t.TempDir()
				writeFile(t, dir, "config.yaml", tt.dirFile)
				environ = append(environ, "NERVE_CONFIG_DIR="+dir)
			}
			_, err := Load(Sources{Embedded: embedded(map[string]string{"test": ""}), Environ: environ})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}
```

`server/internal/platform/config/validate_test.go`：

```go
package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Env: EnvTest,
		Server: ServerConfig{
			Addr:              ":8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			ShutdownTimeout:   20 * time.Second,
			RequestTimeout:    15 * time.Second,
			MaxBodyBytes:      1 << 20,
		},
		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10, CommitTimeout: 2 * time.Second},
		Auth: AuthConfig{
			SignupEnabled:  true,
			AccessTokenTTL: 15 * time.Minute,
			SessionTTL:     720 * time.Hour,
			Password: PasswordConfig{
				Argon2MemoryKiB:     19456,
				Argon2Iterations:    2,
				Argon2Parallelism:   1,
				MaxConcurrentHashes: 4,
				MaxWait:             2 * time.Second,
			},
		},
		Log: LogConfig{Level: "info", Format: "json"},
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	if err := validConfig().validate(); err != nil {
		t.Fatalf("validate() = %v, want nil", err)
	}
}

func TestValidateReportsEveryInvalidKey(t *testing.T) {
	cfg := Config{
		Env:      EnvProd,
		Server:   ServerConfig{Addr: "8080", ReadHeaderTimeout: 0, ReadTimeout: 0, WriteTimeout: -time.Second, ShutdownTimeout: -time.Second},
		Database: DatabaseConfig{URL: "", MaxConns: 0},
		Log:      LogConfig{Level: "verbose", Format: "xml"},
	}
	err := cfg.validate()
	if err == nil {
		t.Fatal("validate() = nil, want errors")
	}
	want := []string{
		`server.addr: must be host:port, e.g. ":8080", got "8080"`,
		"server.read_header_timeout: must be positive, got 0s",
		"server.read_timeout: must be positive, got 0s",
		"server.write_timeout: must be positive, got -1s",
		"server.shutdown_timeout: must be positive, got -1s",
		"server.request_timeout: must be positive, got 0s",
		"server.max_body_bytes: must be at least 1, got 0",
		"database.url: is required",
		"database.max_conns: must be at least 1, got 0",
		"database.commit_timeout: must be positive, got 0s",
		"auth.access_token_ttl: must be positive, got 0s",
		"auth.session_ttl: must be positive, got 0s",
		"auth.jwt.private_key_file: is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519",
		"auth.password.argon2_iterations: must be at least 1, got 0",
		"auth.password.argon2_parallelism: must be at least 1, got 0",
		"auth.password.argon2_memory_kib: must be at least 8 per lane (8), got 0",
		"auth.password.max_concurrent_hashes: must be at least 1, got 0",
		"auth.password.max_wait: must be positive, got 0s",
		`log.level: must be one of debug, info, warn, error, got "verbose"`,
		`log.format: must be text or json, got "xml"`,
	}
	if got := err.Error(); got != strings.Join(want, "\n") {
		t.Errorf("validate() errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
}

func TestValidateCrossKeyRules(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Config)
		want   string
	}{
		{
			// read_timeout is the budget for the whole request, so the headers
			// alone may not be allowed longer.
			name:   "header timeout above read timeout",
			change: func(c *Config) { c.Server.ReadHeaderTimeout = 40 * time.Second },
			want:   "server.read_header_timeout: must not exceed server.read_timeout (30s), got 40s",
		},
		{
			name:   "request timeout not below write timeout",
			change: func(c *Config) { c.Server.RequestTimeout = 60 * time.Second },
			want:   "server.request_timeout: must be less than server.write_timeout (1m0s), got 1m0s",
		},
		{
			name:   "session not longer than an access token",
			change: func(c *Config) { c.Auth.SessionTTL = 15 * time.Minute },
			want:   "auth.session_ttl: must be longer than auth.access_token_ttl (15m0s), got 15m0s",
		},
		{
			name: "argon2 memory below 8 KiB per lane",
			change: func(c *Config) {
				c.Auth.Password.Argon2Parallelism = 4
				c.Auth.Password.Argon2MemoryKiB = 31
			},
			want: "auth.password.argon2_memory_kib: must be at least 8 per lane (32), got 31",
		},
		{
			name:   "prod without a signing key",
			change: func(c *Config) { c.Env = EnvProd },
			want:   "auth.jwt.private_key_file: is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.change(&cfg)
			if err := cfg.validate(); err == nil || err.Error() != tt.want {
				t.Errorf("validate() = %v, want %q", err, tt.want)
			}
		})
	}
}

// Only prod requires the signing key file; dev and test generate a key.
func TestValidateAcceptsProdWithASigningKeyFile(t *testing.T) {
	cfg := validConfig()
	cfg.Env = EnvProd
	cfg.Auth.JWT.PrivateKeyFile = "/etc/nerve/jwt.pem"

	if err := cfg.validate(); err != nil {
		t.Errorf("validate() = %v, want nil", err)
	}
}
```

- [ ] **Step 3: 内嵌的配置文件**

`server/configs/config.yaml`：

```yaml
# 基础配置：列出所有配置项及其默认值。各环境的覆盖项在 config.<env>.yaml 中。
# 任何一项都可以用环境变量覆盖：NERVE_ 加上配置路径，层级之间用双下划线，
# 例如 database.url 对应 NERVE_DATABASE__URL。
server:
  addr: ":8080"
  # 连接上的读写都有上限：读请求头、读整个请求（含请求体）、写响应。
  # 上传这类确实要更久的接口，由自己的 handler 单独放宽，不要调大全局值。
  # handler 自身的执行时间不受这些上限约束，它里面的阻塞调用要自带期限。
  read_header_timeout: 5s
  read_timeout: 30s
  write_timeout: 60s
  shutdown_timeout: 20s
  # 每个接口请求的期限：到期取消请求的 context，数据库调用随之结束。必须短于 write_timeout
  request_timeout: 15s
  # JSON 接口的请求体上限（字节），超出时 413
  max_body_bytes: 1048576
  # 非空时，监听成功后把实际地址写进这个文件（端到端测试用 127.0.0.1:0 监听）
  addr_file: ""

database:
  # 必须提供。dev 环境写在 config.dev.yaml 中；test 和 prod 通过 NERVE_DATABASE__URL 提供。
  url: ""
  max_conns: 10
  # nerve serve 启动前是否自动执行迁移
  auto_migrate: true
  # COMMIT、ROLLBACK 自己的期限：它们不随请求期限取消
  commit_timeout: 2s

auth:
  # 是否开放注册。基础配置（也就是 prod）关闭；config.dev.yaml、config.test.yaml 覆盖为 true。
  # 关闭时第一个账户用 nerve users create 创建
  signup_enabled: false
  access_token_ttl: 15m
  # 会话从登录起算的期限（30 天），续期不延长
  session_ttl: 720h
  jwt:
    # PKCS#8 PEM 格式的 Ed25519 私钥文件：openssl genpkey -algorithm ed25519 -out nerve-jwt.pem
    # prod 必填；dev、test 为空时，启动时生成一把临时密钥（重启后旧的访问令牌失效）
    private_key_file: ""
  password:
    # argon2id 的参数（OWASP 推荐的最低配置）
    argon2_memory_kib: 19456
    argon2_iterations: 2
    argon2_parallelism: 1
    # 同时进行的哈希计算上限；拿不到名额时最多等 max_wait，然后 503 server_busy
    max_concurrent_hashes: 4
    max_wait: 2s

log:
  level: info    # debug | info | warn | error
  format: json   # text | json
```

`server/configs/config.dev.yaml`：

```yaml
# 开发环境（NERVE_ENV=dev，默认）。个人覆盖项写在 config.local.yaml 中（不进仓库）。
server:
  # 开发环境只监听本机，与开发库一样不对局域网开放。
  addr: "127.0.0.1:8080"

database:
  # make dev-db 启动的本地开发库。账号密码只在本机使用，不是机密。
  # 用 NERVE_DEV_DB_PORT 换了端口时，要同时覆盖这一项（NERVE_DATABASE__URL 或 config.local.yaml）。
  url: postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable

auth:
  # 开发环境开放注册（prod 默认关闭，见 config.yaml）
  signup_enabled: true

log:
  level: debug
  format: text
```

`server/configs/config.test.yaml`：

```yaml
# 测试环境（NERVE_ENV=test）：集成测试和端到端测试使用。数据库地址通过 NERVE_DATABASE__URL 提供。
auth:
  # 测试环境开放注册（prod 默认关闭，见 config.yaml）
  signup_enabled: true
  password:
    # 测试用最低的 argon2 参数，让大量注册的测试不被哈希拖慢
    argon2_memory_kib: 64
    argon2_iterations: 1

log:
  level: warn
  format: text
```

`server/configs/embed_test.go`：

```go
package configs_test

import (
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

func TestBuiltInProfiles(t *testing.T) {
	const devURL = "postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable"
	tests := []struct {
		env         string
		addr        string // expected server.addr
		url         string // expected database.url
		autoMigrate bool
		signup      bool   // auth.signup_enabled: closed in prod unless overridden (M2 design decision 2)
		argon2      uint32 // auth.password.argon2_memory_kib
		iterations  uint32 // auth.password.argon2_iterations
		keyFile     string // auth.jwt.private_key_file
		level       string
		format      string
	}{
		{env: "dev", addr: "127.0.0.1:8080", url: devURL, autoMigrate: true, signup: true, argon2: 19456, iterations: 2, level: "debug", format: "text"},
		{env: "test", addr: ":8080", url: "postgres://from-env", autoMigrate: true, signup: true, argon2: 64, iterations: 1, level: "warn", format: "text"},
		{env: "prod", addr: ":8080", url: "postgres://from-env", autoMigrate: false, signup: false, argon2: 19456, iterations: 2, keyFile: "/etc/nerve/jwt.pem", level: "info", format: "json"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			environ := []string{"NERVE_ENV=" + tt.env}
			if tt.env != "dev" {
				environ = append(environ, "NERVE_DATABASE__URL=postgres://from-env")
			}
			if tt.keyFile != "" {
				environ = append(environ, "NERVE_AUTH__JWT__PRIVATE_KEY_FILE="+tt.keyFile)
			}
			cfg, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: environ})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			want := config.Config{
				Env: tt.env,
				Server: config.ServerConfig{
					Addr:              tt.addr,
					ReadHeaderTimeout: 5 * time.Second,
					ReadTimeout:       30 * time.Second,
					WriteTimeout:      60 * time.Second,
					ShutdownTimeout:   20 * time.Second,
					RequestTimeout:    15 * time.Second,
					MaxBodyBytes:      1 << 20,
				},
				Database: config.DatabaseConfig{URL: tt.url, MaxConns: 10, AutoMigrate: tt.autoMigrate, CommitTimeout: 2 * time.Second},
				Auth: config.AuthConfig{
					SignupEnabled:  tt.signup,
					AccessTokenTTL: 15 * time.Minute,
					SessionTTL:     30 * 24 * time.Hour,
					JWT:            config.JWTConfig{PrivateKeyFile: tt.keyFile},
					Password: config.PasswordConfig{
						Argon2MemoryKiB:     tt.argon2,
						Argon2Iterations:    tt.iterations,
						Argon2Parallelism:   1,
						MaxConcurrentHashes: 4,
						MaxWait:             2 * time.Second,
					},
				},
				Log: config.LogConfig{Level: tt.level, Format: tt.format},
			}
			if cfg != want {
				t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
			}
		})
	}
}

func TestProdRequiresDatabaseURLAndSigningKey(t *testing.T) {
	_, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=prod"}})
	if err == nil {
		t.Fatal("Load() error = nil, want database.url and auth.jwt.private_key_file to be required")
	}
	for _, key := range []string{"database.url: is required", "auth.jwt.private_key_file: is required in prod"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Load() error = %v, want it to report %q", err, key)
		}
	}
}
```

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。`cmd/nerve` 的测试用内嵌的 test 配置启动 `nerve serve`，证明新键都有默认值、校验通过。

- [ ] **Step 5: 提交**

```bash
git add server/internal/platform/config server/configs
```
```bash
git commit -m "feat(M2/P1): configuration for request limits, commit timeout and auth

Empty environment values of non-string keys are errors now, and every
*_file key logs whether it is set, never its path.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestBuiltInProfiles` 证明 prod 默认关闭注册；空的环境变量值报错；`make test` 通过。

---

### Task 4: `httpserver`：Router、`API` 值、`ProblemError` 映射、默认拒绝的认证

**Files:**
- Modify: `server/internal/platform/httpserver/problem.go`、`problem_test.go`、`middleware.go`、`middleware_test.go`、`routes.go`、`routes_test.go`、`server.go`、`server_test.go`、`apierrors.go`、`apierrors_test.go`、`contract_test.go`
- Create: `server/internal/platform/httpserver/api.go`、`api_test.go`
- Modify: `api/common.yaml`
- Modify: `server/internal/platform/httpserver/apitest/apitest_test.go`（过渡：只改字段错误的样例；Task 5 写最终版本）
- Modify: `server/internal/bootstrap/app.go`（过渡：`NewRouter`）、`server/internal/modules/instance/module.go`、`adapter/http/handler.go`、`handler_test.go`（过渡：挂到 `Router`，错误出口改为 `BodyError`、`Write`；Task 11 写最终版本）
- Generate: `server/internal/platform/httpserver/apigen/components.gen.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Consumes: `bodyshape.Middleware`、`bodyshape.Table`（Task 1）；`config.ServerConfig.AddrFile`（Task 3）。
- Produces:
  - 平台码 `CodeBadRequest`、`CodeUnauthorized`、`CodeNotFound`、`CodePayloadTooLarge`、`CodeInternal`、`CodeNotReady`；`FieldError.Code`；
  - `RequestID(ctx) string`（导出）；
  - `NewRouter(logger, checks...) *Router`，方法 `HandleFunc`、`Handle`、`ServeHTTP`、`Patterns()`（`NewMux` 删除）；
  - `server.addr_file`：`ListenAndServe` 写出实际地址；
  - `ProblemError` 接口（可选 `ProblemFields() []error`、`RetryAfter() time.Duration`）；`APIErrors` 的 `BadRequest`、`BodyError`、`Write`（`InternalError` 删除）；
  - `Authenticator` 接口、`APIConfig`、`NewAPI(cfg) *API`（字段 `Errors`）、`(*API).Middlewares(bodies *bodyshape.Table) []func(http.Handler) http.Handler`、`RequestMeta{ClientIP, UserAgent}`、`RequestMetaFrom(ctx)`。
- 行为见 spec 2.8；`Middlewares` 按"请求元信息 → 期限 → 请求体上限 → 认证 → 请求体结构"排好后反转返回。

**Tests:**
- `problem_test.go`：`FieldError` 带 `code`。
- `middleware_test.go`：新增 `TestAccessLogOfProbesIsDebug`（`/healthz`、`/readyz` 的访问日志是 DEBUG，其他路径 INFO）。
- `routes_test.go`：新增 `TestRouterRecordsEveryPattern`（平台的三个模式加上后来注册的，按注册顺序）、`TestRouterRoutesToTheRegisteredHandler`。
- `server_test.go`：新增 `TestListenAndServeWritesTheAddrFile`（`127.0.0.1:0` 监听，文件里是能连上的实际地址）、`TestListenAndServeReportsAnUnwritableAddrFile`（错误以 `server.addr_file: write:` 开头，不含路径）。
- `apierrors_test.go`：`TestAPIErrorsBadRequest`；`TestAPIErrorsBodyErrorHidesTheDecoderMessage`（400，固定的 detail，解码器的原话只在 DEBUG 日志）；`TestAPIErrorsBodyErrorOfAnOversizedBodyIs413`；`TestWriteMapsProblemErrors`（7 个：状态、码、detail、字段且跳过不是字段的元素、`Retry-After` 1.5 秒写成 `2`、包在 `errors.Join` 里、413）；`TestWriteLogsAndHidesAnInternalError`；`TestWriteOfACancelledRequestIsNot500`（499、DEBUG）；`TestWriteOfACanceledErrorOnALiveRequestIs500`；`TestWriteAbortsAStartedResponse`；`TestResponseStartedSeesThroughWrappers`。
- `api_test.go`：`TestMetaAndDeadlineRunBeforeAuthentication`、`TestAuthenticationRunsBeforeTheBodyCheck`、`TestBodyLimitRunsBeforeTheBodyCheck`（三个一起从外部核对顺序；去掉 `slices.Reverse` 时连同下一个共 4 个失败）、`TestBodyCheckAnswersEveryProblemAs400`（逐字核对 JSON）、`TestBodyThatIsNotJSONIs400WithAGenericDetail`、`TestAuthenticationDeniesByDefault`（9 个：没有 `Authorization`、别的方案、空令牌都是 401 `Bearer` 且不调用认证器；无效令牌 401 带 `error="invalid_token"`；有效令牌 204；方案名小写也认；认证器的内部错误是 500；公开操作没有令牌 204、带着坏令牌也 204 且不调用认证器）、`TestAuthenticationFailureIsLoggedAtDebugLevel`、`TestRequestMetaClientIP`（IPv4、IPv6、IPv4 映射、带 zone）、`TestRequestMetaOutsideTheMiddlewaresIsZero`。
- `contract_test.go`：平台写出的每种 problem 都合 `Problem` 的 schema，新增请求体解码失败、413、401、带字段码的 422。
- `apitest_test.go` 的 `TestValidateSchema`：字段错误必须带 `code`、`code` 必须在枚举里。

- [ ] **Step 1: 平台码、`FieldError.code`、`RequestID`、探测的日志级别**

`server/internal/platform/httpserver/problem.go`：

```go
// Package httpserver provides nerve's HTTP platform: the fixed middleware
// chain, the router, the per-route middlewares that API operations share
// (API), problem+json errors, health endpoints and the server lifecycle.
package httpserver

import (
	"encoding/json"
	"net/http"
)

// ContentTypeProblem is the media type of RFC 9457 problem details.
const ContentTypeProblem = "application/problem+json"

// Codes of the problems the platform itself reports. The other platform codes
// (validation_failed, server_busy) come from domain errors through
// ProblemError; module codes are namespaced by module, e.g.
// "identity.email_taken" (M2 design 3.11).
const (
	CodeBadRequest      = "bad_request"
	CodeUnauthorized    = "unauthorized"
	CodeNotFound        = "not_found"
	CodePayloadTooLarge = "payload_too_large"
	CodeInternal        = "internal_error"
	CodeNotReady        = "not_ready"
)

// Problem is an RFC 9457 problem details body (v0 design, section 3.5).
// Code is the stable identifier clients branch on; Title is for humans.
type Problem struct {
	Status int          `json:"status"`
	Code   string       `json:"code"`
	Title  string       `json:"title"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError points at one invalid field of a request. Code is one of the
// closed set of field codes (api/common.yaml); clients translate it rather
// than show Message.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteProblem sends p with status p.Status as application/problem+json.
func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", ContentTypeProblem)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p) // nothing useful to do if the client is gone
}
```

`server/internal/platform/httpserver/problem_test.go`：

```go
package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, Problem{
		Status: http.StatusUnprocessableEntity,
		Code:   "issue.state_not_in_project",
		Title:  "State is not in the project",
		Errors: []FieldError{{Field: "state_id", Code: "not_allowed", Message: "unknown state"}},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	want := `{"status":422,"code":"issue.state_not_in_project","title":"State is not in the project",` +
		`"errors":[{"field":"state_id","code":"not_allowed","message":"unknown state"}]}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestWriteProblemOmitsEmptyOptionalMembers(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, Problem{Status: http.StatusNotFound, Code: CodeNotFound, Title: "Not Found"})

	want := `{"status":404,"code":"not_found","title":"Not Found"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}
```

`server/internal/platform/httpserver/middleware.go`：

```go
package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
	"uuid"
)

// HeaderRequestID carries the request ID in requests and responses.
const HeaderRequestID = "X-Request-Id"

const maxRequestIDLen = 128

type requestIDKey struct{}

// middleware wraps h in the platform chain. The order is fixed, outermost
// first: request ID -> recover -> access log.
func middleware(h http.Handler, logger *slog.Logger) http.Handler {
	return withRequestID(withRecover(logger, withAccessLog(logger, h)))
}

// RequestID returns the ID the request ID middleware assigned to the request,
// for log lines outside this package; "" outside a request.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// withRequestID keeps the caller's X-Request-Id when it is a safe token and
// otherwise assigns a new UUIDv7. The ID is echoed in the response.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if !validRequestID(id) {
			id = uuid.NewV7().String()
		}
		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// validRequestID accepts 1 to 128 characters from [A-Za-z0-9._:-], which
// keeps caller-supplied IDs safe to log.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, c := range id {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		case c == '-', c == '_', c == '.', c == ':':
		default:
			return false
		}
	}
	return true
}

// withRecover turns a panic into a logged 500 problem. Headers the handler set
// are dropped, except the request ID: a Set-Cookie must not leak, and a stale
// Content-Length or Content-Encoding would corrupt the problem body. If the
// response has already started, the connection is aborted instead so the
// client cannot mistake a truncated body for a complete one.
func withRecover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler { // deliberate abort: let net/http handle it
				panic(v)
			}
			logger.ErrorContext(r.Context(), "panic serving request",
				slog.String("request_id", RequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
			)
			if rec.status != 0 {
				panic(http.ErrAbortHandler)
			}
			header := rec.Header()
			for name := range header {
				if name != HeaderRequestID {
					delete(header, name)
				}
			}
			WriteProblem(rec, Problem{
				Status: http.StatusInternalServerError,
				Code:   CodeInternal,
				Title:  http.StatusText(http.StatusInternalServerError),
			})
		}()
		next.ServeHTTP(rec, r)
	})
}

// withAccessLog logs one line per request: method, path, status, duration
// and request ID. A request whose handler panicked is logged as a 500, the
// answer the recover middleware gives. The health probes log at debug level:
// an orchestrator probes every few seconds (M0-P2 handoff 8).
func withAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		completed := false
		defer func() {
			status := rec.status
			switch {
			case !completed:
				status = http.StatusInternalServerError
			case status == 0:
				status = http.StatusOK
			}
			level := slog.LevelInfo
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				level = slog.LevelDebug
			}
			logger.LogAttrs(r.Context(), level, "http request",
				slog.String("request_id", RequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
			)
		}()
		next.ServeHTTP(rec, r)
		completed = true
	})
}

// statusRecorder remembers the final status code written through it.
type statusRecorder struct {
	http.ResponseWriter
	status int // 0 until a final (non-1xx) status is written
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 && code >= http.StatusOK {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
```

`server/internal/platform/httpserver/middleware_test.go`：

```go
package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"
)

// captureLogs returns a JSON logger and a function that decodes what it wrote.
func captureLogs(t *testing.T) (*slog.Logger, func() []map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logger, func() []map[string]any {
		var entries []map[string]any
		for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			var entry map[string]any
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatalf("log line %q: %v", line, err)
			}
			entries = append(entries, entry)
		}
		return entries
	}
}

func findLog(entries []map[string]any, msg string) map[string]any {
	for _, e := range entries {
		if e["msg"] == msg {
			return e
		}
	}
	return nil
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestRequestIDIsGeneratedWhenMissing(t *testing.T) {
	var seen string
	h := middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = RequestID(r.Context())
	}), slog.New(slog.DiscardHandler))

	rec := serve(h, httptest.NewRequest(http.MethodGet, "/", nil))

	id := rec.Header().Get(HeaderRequestID)
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("X-Request-Id = %q, want a UUID: %v", id, err)
	}
	if seen != id {
		t.Errorf("handler saw request ID %q, response has %q", seen, id)
	}
}

func TestRequestIDFromCaller(t *testing.T) {
	tests := []struct {
		name, header string
		kept         bool
	}{
		{"safe token is kept", "req-42_a.b:c", true},
		{"spaces are rejected", "req 42", false},
		{"control characters are rejected", "req\n42", false},
		{"overlong IDs are rejected", strings.Repeat("a", 129), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := middleware(http.NotFoundHandler(), slog.New(slog.DiscardHandler))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(HeaderRequestID, tt.header)

			got := serve(h, req).Header().Get(HeaderRequestID)
			if kept := got == tt.header; kept != tt.kept {
				t.Errorf("X-Request-Id = %q for caller ID %q, kept = %v, want %v", got, tt.header, kept, tt.kept)
			}
		})
	}
}

func TestPanicBecomes500Problem(t *testing.T) {
	logger, logs := captureLogs(t)
	h := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), logger)
	req := httptest.NewRequest(http.MethodGet, "/explode", nil)
	req.Header.Set(HeaderRequestID, "req-1")

	rec := serve(h, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ContentTypeProblem {
		t.Errorf("Content-Type = %q, want %q", ct, ContentTypeProblem)
	}
	want := `{"status":500,"code":"internal_error","title":"Internal Server Error"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	entries := logs()
	panicLog := findLog(entries, "panic serving request")
	if panicLog == nil || panicLog["panic"] != "boom" || panicLog["request_id"] != "req-1" || panicLog["stack"] == "" {
		t.Errorf("panic log = %v, want panic, request_id and stack", panicLog)
	}
	access := findLog(entries, "http request")
	if access == nil || access["status"] != float64(500) || access["request_id"] != "req-1" {
		t.Errorf("access log = %v, want status 500 and request_id req-1", access)
	}
}

func TestPanicDiscardsHeadersSetBeforeIt(t *testing.T) {
	// A real server: a stale Content-Length would truncate the problem body.
	srv := httptest.NewServer(middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "a=b")
		w.Header().Set("Content-Length", "5")
		panic("boom")
	}), slog.New(slog.DiscardHandler)))
	defer srv.Close()
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(HeaderRequestID, "req-3")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError || resp.Header.Get("Content-Type") != ContentTypeProblem {
		t.Errorf("response = %d %q, want 500 %s", resp.StatusCode, resp.Header.Get("Content-Type"), ContentTypeProblem)
	}
	if cookies := resp.Header.Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("Set-Cookie = %q, want none: the handler's headers must not leak", cookies)
	}
	if id := resp.Header.Get(HeaderRequestID); id != "req-3" {
		t.Errorf("X-Request-Id = %q, want req-3", id)
	}
	var p Problem
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil || p.Code != CodeInternal {
		t.Errorf("body = %+v, %v; want the complete internal_error problem", p, err)
	}
}

func TestPanicAfterResponseStartedAbortsConnection(t *testing.T) {
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("partial"))
		panic("boom")
	}), slog.New(slog.DiscardHandler))

	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	serve(h, httptest.NewRequest(http.MethodGet, "/", nil))
	t.Error("ServeHTTP returned normally, want a panic")
}

func TestAbortHandlerPanicIsNotRecovered(t *testing.T) {
	logger, logs := captureLogs(t)
	h := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}), logger)

	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("recovered %v, want http.ErrAbortHandler", v)
		}
		if findLog(logs(), "panic serving request") != nil {
			t.Error("a deliberate abort was logged as a panic")
		}
	}()
	serve(h, httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestAccessLog(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    float64
	}{
		{"explicit status", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }, 201},
		{"implicit 200", func(http.ResponseWriter, *http.Request) {}, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logs := captureLogs(t)
			req := httptest.NewRequest(http.MethodPost, "/things?x=1", nil)
			req.Header.Set(HeaderRequestID, "req-2")

			serve(middleware(tt.handler, logger), req)

			entry := findLog(logs(), "http request")
			if entry == nil {
				t.Fatal("no access log entry")
			}
			if entry["method"] != "POST" || entry["path"] != "/things" || entry["status"] != tt.want ||
				entry["request_id"] != "req-2" || entry["duration"] == nil || entry["level"] != "INFO" {
				t.Errorf("access log = %v", entry)
			}
		})
	}
}

// Orchestrators probe every few seconds: the probes' access log lines are
// debug level (M0-P2 handoff 8).
func TestAccessLogOfProbesIsDebug(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz"} {
		logger, logs := captureLogs(t)

		serve(middleware(http.NotFoundHandler(), logger), httptest.NewRequest(http.MethodGet, path, nil))

		if entry := findLog(logs(), "http request"); entry == nil || entry["level"] != "DEBUG" {
			t.Errorf("GET %s access log = %v, want level DEBUG", path, entry)
		}
	}
}
```

- [ ] **Step 2: `Router` 和 `server.addr_file`**

`server/internal/platform/httpserver/routes.go`：

```go
package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// readinessTimeout bounds one /readyz evaluation, so a hung dependency
// answers 503 instead of hanging the probe.
const readinessTimeout = 2 * time.Second

// Check is one readiness condition, such as "the database answers".
type Check struct {
	Name string
	Run  func(ctx context.Context) error
}

// Router is nerve's root router: an http.ServeMux that remembers every
// pattern registered on it, so a whole-program test can compare the API
// routes with the contract (M2 design 3.6). It satisfies the ServeMux
// interface of the generated code (HandleFunc and ServeHTTP).
type Router struct {
	mux      *http.ServeMux
	patterns []string
}

// NewRouter returns a router holding the platform routes:
//
//   - GET /healthz: liveness; never touches a dependency.
//   - GET /readyz: runs checks in order; the first failure answers 503.
//   - /api/: every API path no module handles answers 404 problem+json,
//     never the web UI.
//
// Modules and the web UI register their own routes on it.
func NewRouter(logger *slog.Logger, checks ...Check) *Router {
	r := &Router{mux: http.NewServeMux()}
	r.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeStatusOK(w)
	})
	r.HandleFunc("GET /readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), readinessTimeout)
		defer cancel()
		for _, c := range checks {
			if err := c.Run(ctx); err != nil {
				logger.WarnContext(ctx, "readiness check failed",
					slog.String("request_id", RequestID(ctx)),
					slog.String("check", c.Name),
					slog.Any("error", err),
				)
				WriteProblem(w, Problem{
					Status: http.StatusServiceUnavailable,
					Code:   CodeNotReady,
					Title:  http.StatusText(http.StatusServiceUnavailable),
					Detail: c.Name + " is not ready",
				})
				return
			}
		}
		writeStatusOK(w)
	})
	r.HandleFunc("/api/", func(w http.ResponseWriter, req *http.Request) {
		WriteProblem(w, Problem{
			Status: http.StatusNotFound,
			Code:   CodeNotFound,
			Title:  http.StatusText(http.StatusNotFound),
			Detail: "no API endpoint for " + req.Method + " " + req.URL.Path,
		})
	})
	return r
}

// HandleFunc registers handler for pattern and records the pattern.
func (r *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.patterns = append(r.patterns, pattern)
	r.mux.HandleFunc(pattern, handler)
}

// Handle registers h for pattern and records the pattern.
func (r *Router) Handle(pattern string, h http.Handler) {
	r.patterns = append(r.patterns, pattern)
	r.mux.Handle(pattern, h)
}

// ServeHTTP dispatches the request to the handler of the matching pattern.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Patterns returns every pattern registered so far, in registration order.
func (r *Router) Patterns() []string {
	return slices.Clone(r.patterns)
}

func writeStatusOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

`server/internal/platform/httpserver/routes_test.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestHealthzDoesNotRunChecks(t *testing.T) {
	failing := Check{Name: "database", Run: func(context.Context) error { return errors.New("down") }}
	router := NewRouter(slog.New(slog.DiscardHandler), failing)

	rec := serve(router, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}`+"\n" {
		t.Errorf("GET /healthz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
}

func TestReadyzWhenAllChecksPass(t *testing.T) {
	var ran []string
	check := func(name string) Check {
		return Check{Name: name, Run: func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Errorf("check %s ran without a deadline", name)
			}
			ran = append(ran, name)
			return nil
		}}
	}
	router := NewRouter(slog.New(slog.DiscardHandler), check("database"), check("migrations"))

	rec := serve(router, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}`+"\n" {
		t.Errorf("GET /readyz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
	if strings.Join(ran, ",") != "database,migrations" {
		t.Errorf("checks ran = %v, want database then migrations", ran)
	}
}

func TestReadyzReportsFirstFailingCheck(t *testing.T) {
	logger, logs := captureLogs(t)
	secondRan := false
	router := NewRouter(logger,
		Check{Name: "database", Run: func(context.Context) error { return errors.New("connection refused") }},
		Check{Name: "migrations", Run: func(context.Context) error { secondRan = true; return nil }},
	)

	rec := serve(router, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ContentTypeProblem {
		t.Errorf("Content-Type = %q, want %q", ct, ContentTypeProblem)
	}
	want := `{"status":503,"code":"not_ready","title":"Service Unavailable","detail":"database is not ready"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	if secondRan {
		t.Error("checks after the failing one ran")
	}
	entry := findLog(logs(), "readiness check failed")
	if entry == nil || entry["check"] != "database" || entry["error"] != "connection refused" {
		t.Errorf("log = %v, want the failing check and its error", entry)
	}
}

func TestUnknownAPIPathIsProblem404(t *testing.T) {
	router := NewRouter(slog.New(slog.DiscardHandler))
	for _, target := range []string{"/api/", "/api/v0/nope"} {
		rec := serve(router, httptest.NewRequest(http.MethodPost, target, nil))

		if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != ContentTypeProblem {
			t.Errorf("POST %s = %d %s, want 404 problem+json", target, rec.Code, rec.Header().Get("Content-Type"))
		}
		want := `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for POST ` + target + `"}` + "\n"
		if rec.Body.String() != want {
			t.Errorf("body = %s, want %s", rec.Body, want)
		}
	}
}

// The whole-program tests compare the registered API patterns with the
// contract, so both registration methods must record (M2 design 3.6).
func TestRouterRecordsEveryPattern(t *testing.T) {
	router := NewRouter(slog.New(slog.DiscardHandler))
	router.HandleFunc("GET /api/v0/things", func(http.ResponseWriter, *http.Request) {})
	router.Handle("/", http.NotFoundHandler())

	want := []string{"GET /healthz", "GET /readyz", "/api/", "GET /api/v0/things", "/"}
	if got := router.Patterns(); !slices.Equal(got, want) {
		t.Errorf("Patterns() = %q, want %q", got, want)
	}
}

func TestRouterRoutesToTheRegisteredHandler(t *testing.T) {
	router := NewRouter(slog.New(slog.DiscardHandler))
	var pattern string
	router.HandleFunc("GET /api/v0/things/{id}", func(_ http.ResponseWriter, r *http.Request) { pattern = r.Pattern })

	serve(router, httptest.NewRequest(http.MethodGet, "/api/v0/things/7", nil))

	if pattern != "GET /api/v0/things/{id}" {
		t.Errorf("r.Pattern = %q, want the registered pattern", pattern)
	}
}

func TestOtherPathsAreLeftForTheWebUI(t *testing.T) {
	rec := serve(NewRouter(slog.New(slog.DiscardHandler)), httptest.NewRequest(http.MethodGet, "/projects", nil))

	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") == ContentTypeProblem {
		t.Errorf("GET /projects = %d %s, want the router's plain 404", rec.Code, rec.Header().Get("Content-Type"))
	}
}
```

`server/internal/platform/httpserver/server.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// idleTimeout closes keep-alive connections that stay idle this long, so idle
// clients cannot hold connections open indefinitely.
const idleTimeout = 2 * time.Minute

// Server is the HTTP server of a nerve process.
type Server struct {
	srv             *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
	addrFile        string
}

// NewServer serves h behind the platform middleware chain
// (request ID -> recover -> access log). Reading and writing on a connection
// are bounded: request headers (server.read_header_timeout), the whole
// request with its body (server.read_timeout), the response
// (server.write_timeout) and idle keep-alive (idleTimeout). A handler that
// legitimately needs longer, such as a file upload, extends its own deadlines
// with http.ResponseController instead of raising them for every request.
// A handler's own run time is not bounded: write_timeout fails its writes but
// neither stops it nor cancels its context, so its blocking calls need their
// own deadlines.
func NewServer(cfg config.ServerConfig, h http.Handler, logger *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              cfg.Addr,
			Handler:           middleware(h, logger),
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       idleTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
		addrFile:        cfg.AddrFile,
	}
}

// ListenAndServe listens on server.addr, writes the address it got to
// server.addr_file when that is set (the end-to-end tests listen on port 0),
// and then behaves like Serve.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.srv.Addr, err)
	}
	if s.addrFile != "" {
		if err := writeAddrFile(s.addrFile, ln.Addr().String()); err != nil {
			_ = ln.Close()
			return err
		}
	}
	return s.Serve(ctx, ln)
}

// writeAddrFile writes addr to path through a temporary file and a rename,
// so a reader never sees a partial address. The error leaves the path out:
// like every *_file key, server.addr_file is not logged.
func writeAddrFile(path, addr string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(addr), 0o600); err != nil {
		return fmt.Errorf("server.addr_file: write: %w", errors.Unwrap(err))
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("server.addr_file: rename: %w", errors.Unwrap(err))
	}
	return nil
}

// Serve serves HTTP on ln until ctx is done. It then stops accepting
// connections and gives in-flight requests up to server.shutdown_timeout to
// finish; connections still open after that are closed and an error is
// returned. A clean shutdown returns nil.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	s.logger.InfoContext(ctx, "http server listening", slog.String("addr", ln.Addr().String()))
	served := make(chan error, 1)
	go func() { served <- s.srv.Serve(ln) }()

	select {
	case err := <-served:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	s.logger.InfoContext(ctx, "http server shutting down", slog.Duration("timeout", s.shutdownTimeout))
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()
	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		closeErr := s.srv.Close()
		<-served
		return fmt.Errorf("http server shutdown: %w", errors.Join(err, closeErr))
	}
	<-served // http.ErrServerClosed once Shutdown has begun
	s.logger.InfoContext(ctx, "http server stopped")
	return nil
}
```

`server/internal/platform/httpserver/server_test.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

var client = &http.Client{Timeout: 5 * time.Second}

// startServer runs a Server on a random local port. Cancelling the returned
// context starts the shutdown; done yields the result of Serve.
func startServer(t *testing.T, shutdownTimeout time.Duration, h http.Handler) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	cfg := config.ServerConfig{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		ShutdownTimeout:   shutdownTimeout,
	}
	return startServerWith(t, cfg, h)
}

// startServerWith is startServer with the given timeouts; cfg.Addr is ignored.
func startServerWith(t *testing.T, cfg config.ServerConfig, h http.Handler) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Addr = ln.Addr().String()
	srv := NewServer(cfg, h, slog.New(slog.DiscardHandler))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()
	return "http://" + ln.Addr().String(), cancel, done
}

func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
		return nil
	}
}

func TestServeAppliesMiddlewareAndStopsOnCancel(t *testing.T) {
	url, cancel, done := startServer(t, time.Second, NewRouter(slog.New(slog.DiscardHandler)))

	resp, err := client.Get(url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get(HeaderRequestID) == "" {
		t.Errorf("GET /healthz = %d, X-Request-Id %q; want 200 with a request ID", resp.StatusCode, resp.Header.Get(HeaderRequestID))
	}

	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("Serve() = %v, want nil after a clean shutdown", err)
	}
}

func TestShutdownDrainsInFlightRequests(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	url, cancel, done := startServer(t, 5*time.Second, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "finished")
	}))
	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := client.Get(url + "/slow")
		if err != nil {
			got <- result{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		got <- result{string(body), err}
	}()
	<-started

	cancel()
	addr := strings.TrimPrefix(url, "http://")
	for deadline := time.Now().Add(2 * time.Second); ; {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			break // the listener is closed: no new connections
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("server still accepts connections after shutdown began")
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(release)

	if r := <-got; r.err != nil || r.body != "finished" {
		t.Errorf("in-flight request = %q, %v; want it to finish", r.body, r.err)
	}
	if err := wait(t, done); err != nil {
		t.Errorf("Serve() = %v, want nil", err)
	}
}

func TestShutdownGivesUpAfterTimeout(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(release) })
	url, cancel, done := startServer(t, 100*time.Millisecond, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
	}))
	go func() {
		if resp, err := client.Get(url + "/stuck"); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-started

	begin := time.Now()
	cancel()
	err := wait(t, done)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve() = %v, want a shutdown deadline error", err)
	}
	if elapsed := time.Since(begin); elapsed > 2*time.Second {
		t.Errorf("shutdown took %s, want about the 100ms timeout", elapsed)
	}
}

// read_header_timeout stops at the headers. A client that sends complete
// headers but never the body it announced must still let go of the
// connection: read_timeout bounds the whole request.
func TestReadTimeoutReleasesARequestWhoseBodyNeverArrives(t *testing.T) {
	cfg := config.ServerConfig{
		ReadHeaderTimeout: 100 * time.Millisecond,
		ReadTimeout:       300 * time.Millisecond,
		WriteTimeout:      5 * time.Second,
		ShutdownTimeout:   time.Second,
	}
	url, _, _ := startServerWith(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	conn, err := net.Dial("tcp", strings.TrimPrefix(url, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 1\r\n\r\n"); err != nil {
		t.Fatal(err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	reply, err := io.ReadAll(conn) // returns once the server closes the connection
	if ne, ok := errors.AsType[net.Error](err); ok && ne.Timeout() {
		t.Fatal("the server still holds the connection after 3s, want it closed after the 300ms read_timeout")
	}
	// The handler answered, so the headers were read in time: it was the
	// missing body, not the header timeout, that ended the connection.
	if !strings.HasPrefix(string(reply), "HTTP/1.1 200 ") {
		t.Errorf("reply = %q, want the handler's 200 before the connection closed", reply)
	}
}

// write_timeout bounds how long the response may take to produce and send: a
// handler that answers late, or a client that stops reading, cannot hold the
// response past it. It fails the writes; it neither stops the handler nor
// cancels its context.
func TestWriteTimeoutCutsOffALateResponse(t *testing.T) {
	cfg := config.ServerConfig{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      200 * time.Millisecond,
		ShutdownTimeout:   time.Second,
	}
	url, _, _ := startServerWith(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = io.WriteString(w, "late")
	}))

	resp, err := client.Get(url + "/late")
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("GET /late = %d, want the connection cut off after the 200ms write_timeout", resp.StatusCode)
	}
}

func TestListenAndServeReportsListenError(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = taken.Close() }()
	cfg := config.ServerConfig{Addr: taken.Addr().String(), ReadHeaderTimeout: time.Second, ShutdownTimeout: time.Second}

	err = NewServer(cfg, http.NotFoundHandler(), slog.New(slog.DiscardHandler)).ListenAndServe(context.Background())

	if err == nil || !strings.Contains(err.Error(), "listen on "+taken.Addr().String()) {
		t.Errorf("ListenAndServe() = %v, want a listen error", err)
	}
}

// Listening on port 0, the server tells where it listens through
// server.addr_file; the end-to-end fixture reads it instead of guessing a
// free port (M0-P6 handoff).
func TestListenAndServeWritesTheAddrFile(t *testing.T) {
	addrFile := filepath.Join(t.TempDir(), "addr")
	cfg := config.ServerConfig{Addr: "127.0.0.1:0", AddrFile: addrFile, ReadHeaderTimeout: time.Second, ShutdownTimeout: time.Second}
	srv := NewServer(cfg, NewRouter(slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe(ctx) }()

	var addr []byte
	for deadline := time.Now().Add(5 * time.Second); len(addr) == 0 && time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		addr, _ = os.ReadFile(addrFile)
	}
	resp, err := client.Get("http://" + string(addr) + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz at the address in addr_file %q: %v", addr, err)
	}
	_ = resp.Body.Close()
	cancel()
	if err := wait(t, done); err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz = %d, ListenAndServe() = %v; want 200 and nil", resp.StatusCode, err)
	}
}

// The error names the key, never the path: *_file keys are not logged.
func TestListenAndServeReportsAnUnwritableAddrFile(t *testing.T) {
	addrFile := filepath.Join(t.TempDir(), "secret-dir", "missing", "addr")
	cfg := config.ServerConfig{Addr: "127.0.0.1:0", AddrFile: addrFile, ReadHeaderTimeout: time.Second, ShutdownTimeout: time.Second}

	err := NewServer(cfg, http.NotFoundHandler(), slog.New(slog.DiscardHandler)).ListenAndServe(context.Background())

	if err == nil || !strings.HasPrefix(err.Error(), "server.addr_file: ") || strings.Contains(err.Error(), "secret-dir") {
		t.Errorf("ListenAndServe() = %v, want a server.addr_file error without the path", err)
	}
}
```

- [ ] **Step 3: `ProblemError` 与 `APIErrors`**

`server/internal/platform/httpserver/apierrors.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"
)

// ProblemError is an error the platform maps to a problem+json response
// (M2 design 3.11). Error() becomes the problem's detail. The platform does
// not import internal/shared: shared.Error satisfies this by structure, and so
// can any other error type, e.g. bodyshape's.
//
// Two methods are optional:
//
//	ProblemFields() []error    each element has ProblemField() and
//	                           ProblemCode() string, and Error() is the message
//	RetryAfter() time.Duration a positive value becomes Retry-After
type ProblemError interface {
	error
	ProblemStatus() int
	ProblemCode() string
}

type problemFields interface {
	ProblemFields() []error
}

type problemField interface {
	error
	ProblemField() string
	ProblemCode() string
}

type retryAfter interface {
	RetryAfter() time.Duration
}

// statusClientClosedRequest is logged when the client went away before the
// answer (nginx's 499); the client never sees it.
const statusClientClosedRequest = 499

// APIErrors answers errors as problem+json. Every module wires it into its
// generated handler, and the per-route middlewares use it:
//
//   - StdHTTPServerOptions.ErrorHandlerFunc: BadRequest (parameter binding)
//   - StrictHTTPServerOptions.RequestErrorHandlerFunc: BodyError
//   - StrictHTTPServerOptions.ResponseErrorHandlerFunc: Write
//
// Write is the one road from an error to a problem (M2 design 3.11): a
// ProblemError becomes its own problem, anything else a logged 500.
type APIErrors struct {
	logger *slog.Logger
}

// NewAPIErrors returns APIErrors that log internal errors to logger.
func NewAPIErrors(logger *slog.Logger) APIErrors {
	return APIErrors{logger: logger}
}

// BadRequest answers 400 bad_request for a request whose parameters the
// generated code could not bind; err says what was wrong and becomes the
// detail. (M2/P3 derives the field from binding errors instead.)
func (APIErrors) BadRequest(w http.ResponseWriter, _ *http.Request, err error) {
	WriteProblem(w, Problem{
		Status: http.StatusBadRequest,
		Code:   CodeBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
		Detail: err.Error(),
	})
}

// BodyError answers a request body that could not be read, parsed or
// decoded, from the bodyshape middleware or the generated strict handler. A
// ProblemError (bodyshape's structural 400 with its fields) and
// *http.MaxBytesError (413) go through Write. Anything else, e.g. a body
// that is not JSON or is empty, is 400 bad_request with a generic detail:
// the decoder's message names Go types, so it is logged at debug level only.
func (e APIErrors) BodyError(w http.ResponseWriter, r *http.Request, err error) {
	var pe ProblemError
	var tooLarge *http.MaxBytesError
	if errors.As(err, &pe) || errors.As(err, &tooLarge) {
		e.Write(w, r, err)
		return
	}
	e.logger.LogAttrs(r.Context(), slog.LevelDebug, "request body not decoded",
		slog.String("request_id", RequestID(r.Context())), slog.Any("error", err))
	WriteProblem(w, Problem{
		Status: http.StatusBadRequest,
		Code:   CodeBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
		Detail: "The request body could not be decoded.",
	})
}

// Write answers err, returned by a handler or a per-route middleware:
//
//   - a ProblemError: its status, code, detail, fields and Retry-After;
//   - *http.MaxBytesError: 413 payload_too_large;
//   - context.Canceled while the request's context is cancelled: the client
//     went away; logged at debug level, no 500;
//   - anything else: logged, and 500 internal_error without detail.
//
// If the response has already started (the generated code reports a failed
// write that way), a problem appended would corrupt it: the connection is
// aborted instead and the error logged at warn level, as the recover
// middleware does.
func (e APIErrors) Write(w http.ResponseWriter, r *http.Request, err error) {
	attrs := []slog.Attr{
		slog.String("request_id", RequestID(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	}
	if responseStarted(w) {
		e.logger.LogAttrs(r.Context(), slog.LevelWarn, "response failed after it started", attrs...)
		panic(http.ErrAbortHandler)
	}
	var pe ProblemError
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &pe):
		p, retry := problemOf(pe)
		if retry > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
		}
		WriteProblem(w, p)
	case errors.As(err, &tooLarge):
		WriteProblem(w, Problem{
			Status: http.StatusRequestEntityTooLarge,
			Code:   CodePayloadTooLarge,
			Title:  http.StatusText(http.StatusRequestEntityTooLarge),
			Detail: fmt.Sprintf("The request body exceeds %d bytes.", tooLarge.Limit),
		})
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		e.logger.LogAttrs(r.Context(), slog.LevelDebug, "client went away", attrs...)
		w.WriteHeader(statusClientClosedRequest)
	default:
		e.logger.LogAttrs(r.Context(), slog.LevelError, "API handler failed", attrs...)
		WriteProblem(w, Problem{
			Status: http.StatusInternalServerError,
			Code:   CodeInternal,
			Title:  http.StatusText(http.StatusInternalServerError),
		})
	}
}

// problemOf builds the problem for pe and returns its retry delay.
func problemOf(pe ProblemError) (Problem, time.Duration) {
	status := pe.ProblemStatus()
	p := Problem{Status: status, Code: pe.ProblemCode(), Title: http.StatusText(status), Detail: pe.Error()}
	if pf, ok := pe.(problemFields); ok {
		for _, fe := range pf.ProblemFields() {
			var f problemField
			if errors.As(fe, &f) {
				p.Errors = append(p.Errors, FieldError{Field: f.ProblemField(), Code: f.ProblemCode(), Message: f.Error()})
			}
		}
	}
	var retry time.Duration
	if ra, ok := pe.(retryAfter); ok {
		retry = ra.RetryAfter()
	}
	return p, retry
}

// responseStarted reports whether a status has already gone out through w.
// Generated code receives the access log's statusRecorder, possibly wrapped
// by a module's StdHTTPServerOptions.Middlewares, so wrappers are unwrapped
// the way http.ResponseController does it.
func responseStarted(w http.ResponseWriter) bool {
	for {
		switch rw := w.(type) {
		case *statusRecorder:
			return rw.status != 0
		case interface{ Unwrap() http.ResponseWriter }:
			w = rw.Unwrap()
		default:
			return false
		}
	}
}
```

`server/internal/platform/httpserver/apierrors_test.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// problemErr is a ProblemError as a module's error type would be.
type problemErr struct {
	status int
	code   string
	detail string
	fields []error
	retry  time.Duration
}

func (e problemErr) Error() string             { return e.detail }
func (e problemErr) ProblemStatus() int        { return e.status }
func (e problemErr) ProblemCode() string       { return e.code }
func (e problemErr) ProblemFields() []error    { return e.fields }
func (e problemErr) RetryAfter() time.Duration { return e.retry }

type fieldErr struct{ field, code, message string }

func (f fieldErr) Error() string        { return f.message }
func (f fieldErr) ProblemField() string { return f.field }
func (f fieldErr) ProblemCode() string  { return f.code }

// minimalErr has only the required methods of ProblemError.
type minimalErr struct{}

func (minimalErr) Error() string       { return "the thing is gone" }
func (minimalErr) ProblemStatus() int  { return http.StatusNotFound }
func (minimalErr) ProblemCode() string { return "things.not_found" }

func TestAPIErrorsBadRequest(t *testing.T) {
	errs := NewAPIErrors(slog.New(slog.DiscardHandler))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.BadRequest(w, r, errors.New("Invalid format for parameter limit: not a number"))
	})

	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v0/things?limit=x", nil))

	if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != ContentTypeProblem {
		t.Errorf("response = %d %s, want 400 problem+json", rec.Code, rec.Header().Get("Content-Type"))
	}
	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"Invalid format for parameter limit: not a number"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
}

// The decoder's messages name Go types: the answer is generic and the
// message goes to the debug log (M0-P3 handoff 2).
func TestAPIErrorsBodyErrorHidesTheDecoderMessage(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	decodeErr := errors.New("can't decode JSON body: json: cannot unmarshal number into Go struct field RegisterRequest.email of type string")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errs.BodyError(w, r, decodeErr) })

	rec := serve(h, httptest.NewRequest(http.MethodPost, "/api/v0/things", nil))

	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request body could not be decoded."}` + "\n"
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want {
		t.Errorf("response = %d %s, want 400 %s", rec.Code, rec.Body, want)
	}
	if entry := findLog(logs(), "request body not decoded"); entry == nil || entry["level"] != "DEBUG" || entry["error"] != decodeErr.Error() {
		t.Errorf("log = %v, want the decoder's message at debug level", entry)
	}
}

func TestAPIErrorsBodyErrorOfAnOversizedBodyIs413(t *testing.T) {
	errs := NewAPIErrors(slog.New(slog.DiscardHandler))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.BodyError(w, r, fmt.Errorf("can't decode JSON body: %w", &http.MaxBytesError{Limit: 1024}))
	})

	rec := serve(h, httptest.NewRequest(http.MethodPost, "/api/v0/things", nil))

	want := `{"status":413,"code":"payload_too_large","title":"Request Entity Too Large","detail":"The request body exceeds 1024 bytes."}` + "\n"
	if rec.Code != http.StatusRequestEntityTooLarge || rec.Body.String() != want {
		t.Errorf("response = %d %s, want %s", rec.Code, rec.Body, want)
	}
}

func TestWriteMapsProblemErrors(t *testing.T) {
	conflict := problemErr{status: http.StatusConflict, code: "things.taken", detail: "The name is taken."}
	invalid := problemErr{
		status: http.StatusUnprocessableEntity, code: "validation_failed", detail: "The request has invalid values.",
		fields: []error{
			fieldErr{"name", "too_long", "at most 255 characters"},
			errors.New("not a field error: skipped"),
			fieldErr{"tags[1].name", "required", "is required"},
		},
	}
	tests := []struct {
		name       string
		err        error
		status     int
		body       string
		retryAfter string
	}{
		{"problem error", conflict, http.StatusConflict,
			`{"status":409,"code":"things.taken","title":"Conflict","detail":"The name is taken."}`, ""},
		{"wrapped", fmt.Errorf("create thing: %w", conflict), http.StatusConflict,
			`{"status":409,"code":"things.taken","title":"Conflict","detail":"The name is taken."}`, ""},
		{"joined", errors.Join(errors.New("context"), conflict), http.StatusConflict,
			`{"status":409,"code":"things.taken","title":"Conflict","detail":"The name is taken."}`, ""},
		{"only the required methods", minimalErr{}, http.StatusNotFound,
			`{"status":404,"code":"things.not_found","title":"Not Found","detail":"the thing is gone"}`, ""},
		{"field errors", invalid, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"name","code":"too_long","message":"at most 255 characters"},{"field":"tags[1].name","code":"required","message":"is required"}]}`, ""},
		{"retry after rounds up", problemErr{status: http.StatusServiceUnavailable, code: "server_busy", detail: "busy", retry: 1500 * time.Millisecond},
			http.StatusServiceUnavailable, `{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"busy"}`, "2"},
		{"payload too large", fmt.Errorf("read body: %w", &http.MaxBytesError{Limit: 1048576}), http.StatusRequestEntityTooLarge,
			`{"status":413,"code":"payload_too_large","title":"Request Entity Too Large","detail":"The request body exceeds 1048576 bytes."}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NewAPIErrors(slog.New(slog.DiscardHandler))
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errs.Write(w, r, tt.err) })

			rec := serve(h, httptest.NewRequest(http.MethodPost, "/api/v0/things", nil))

			if rec.Code != tt.status || rec.Header().Get("Content-Type") != ContentTypeProblem || rec.Body.String() != tt.body+"\n" {
				t.Errorf("response = %d %s %s, want %d problem+json %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body, tt.status, tt.body)
			}
			if got := rec.Header().Get("Retry-After"); got != tt.retryAfter {
				t.Errorf("Retry-After = %q, want %q", got, tt.retryAfter)
			}
		})
	}
}

// Nothing was written yet, so the platform chain's recorder reports the
// response as not started and Write answers the 500 problem.
func TestWriteLogsAndHidesAnInternalError(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.Write(w, r, errors.New("dial tcp 10.0.0.5:5432: connection refused"))
	}), logger)
	req := httptest.NewRequest(http.MethodGet, "/api/v0/things", nil)
	req.Header.Set(HeaderRequestID, "req-7")

	rec := serve(h, req)

	if rec.Code != http.StatusInternalServerError || rec.Header().Get("Content-Type") != ContentTypeProblem {
		t.Errorf("response = %d %s, want 500 problem+json", rec.Code, rec.Header().Get("Content-Type"))
	}
	want := `{"status":500,"code":"internal_error","title":"Internal Server Error"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	entry := findLog(logs(), "API handler failed")
	if entry == nil || entry["level"] != "ERROR" || entry["error"] != "dial tcp 10.0.0.5:5432: connection refused" ||
		entry["request_id"] != "req-7" || entry["method"] != "GET" || entry["path"] != "/api/v0/things" {
		t.Errorf("log = %v, want the error with request_id, method and path at level ERROR", entry)
	}
}

// A client that went away is not a server fault: no 500, no ERROR line
// (M0-P3 handoff 2).
func TestWriteOfACancelledRequestIsNot500(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v0/things", nil).WithContext(ctx)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.Write(w, r, fmt.Errorf("query things: %w", context.Canceled))
	})

	rec := serve(h, req)

	if rec.Code != statusClientClosedRequest {
		t.Errorf("status = %d, want %d", rec.Code, statusClientClosedRequest)
	}
	for _, e := range logs() {
		if e["level"] == "ERROR" {
			t.Errorf("log %v at level ERROR, want none", e)
		}
	}
	if entry := findLog(logs(), "client went away"); entry == nil || entry["level"] != "DEBUG" {
		t.Errorf("log = %v, want the cancellation at debug level", entry)
	}
}

// context.Canceled while the request itself is still live is a server fault.
func TestWriteOfACanceledErrorOnALiveRequestIs500(t *testing.T) {
	errs := NewAPIErrors(slog.New(slog.DiscardHandler))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errs.Write(w, r, context.Canceled) })

	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v0/things", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// Generated strict code also reports a write that failed after the response
// started (WriteHeader(200), then the buffered body fails to go out). A
// problem appended there would corrupt the 200, so the response is aborted.
func TestWriteAbortsAStartedResponse(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	srv := httptest.NewServer(middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"product":`)
		_ = http.NewResponseController(w).Flush()
		errs.Write(w, r, errors.New("write tcp: broken pipe"))
	}), logger))
	t.Cleanup(srv.Close)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v0/things", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(HeaderRequestID, "req-8")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	srv.Close() // waits for the handler, so its log lines are complete

	if resp.StatusCode != http.StatusOK || readErr == nil {
		t.Errorf("response = %d, read error %v; want the started 200 cut off", resp.StatusCode, readErr)
	}
	if string(body) != `{"product":` {
		t.Errorf("body = %q, want only what the handler wrote: no problem appended", body)
	}
	entries := logs()
	for _, e := range entries {
		if e["level"] == "ERROR" {
			t.Errorf("log %v at level ERROR, want none: the client may simply be gone", e)
		}
	}
	entry := findLog(entries, "response failed after it started")
	if entry == nil || entry["level"] != "WARN" || entry["error"] != "write tcp: broken pipe" ||
		entry["request_id"] != "req-8" || entry["method"] != "GET" || !strings.HasSuffix(entry["path"].(string), "/api/v0/things") {
		t.Errorf("log = %v, want the error with request_id, method and path at level WARN", entry)
	}
}

// wrappingWriter stands for a module middleware that wraps the writer.
type wrappingWriter struct{ http.ResponseWriter }

func (w wrappingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func TestResponseStartedSeesThroughWrappers(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	w := wrappingWriter{rec}
	if responseStarted(w) {
		t.Error("responseStarted() = true before anything was written")
	}
	rec.WriteHeader(http.StatusOK)
	if !responseStarted(w) {
		t.Error("responseStarted() = false after WriteHeader, through a wrapper")
	}
	if responseStarted(httptest.NewRecorder()) {
		t.Error("responseStarted() = true for a writer outside the platform chain")
	}
}
```

- [ ] **Step 4: `API`：按路由的中间件和默认拒绝**

`server/internal/platform/httpserver/api.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

// Authenticator checks a bearer token (M2 design 3.6). The identity module
// implements it; the platform does not know what an account is.
type Authenticator interface {
	// Authenticate returns a context carrying the caller, and the caller's
	// rate-limit key (session:<id> or pat:<id>, used from M2/P2 on). An
	// invalid token is an error with ProblemStatus() 401; any other error is
	// an internal fault.
	Authenticate(ctx context.Context, token string) (context.Context, string, error)
}

// APIConfig is what the platform's per-route middlewares need.
type APIConfig struct {
	Logger        *slog.Logger
	Authenticator Authenticator
	// PublicOperations are the route patterns that need no token, e.g.
	// "POST /api/v0/auth/register": the union of every module's list.
	PublicOperations []string
	MaxBodyBytes     int64         // server.max_body_bytes
	RequestTimeout   time.Duration // server.request_timeout
}

// API is what the platform hands to every module's HTTP adapter: the error
// mapping for the generated code, and the per-route middlewares.
type API struct {
	Errors         APIErrors
	logger         *slog.Logger
	authenticator  Authenticator
	public         map[string]bool
	maxBodyBytes   int64
	requestTimeout time.Duration
}

// NewAPI returns the API value for cfg.
func NewAPI(cfg APIConfig) *API {
	public := make(map[string]bool, len(cfg.PublicOperations))
	for _, p := range cfg.PublicOperations {
		public[p] = true
	}
	return &API{
		Errors:         NewAPIErrors(cfg.Logger),
		logger:         cfg.Logger,
		authenticator:  cfg.Authenticator,
		public:         public,
		maxBodyBytes:   cfg.MaxBodyBytes,
		requestTimeout: cfg.RequestTimeout,
	}
}

// Middlewares returns the per-route middlewares for a module's generated
// StdHTTPServerOptions.Middlewares; bodies is the module's generated
// bodyshape table. They run in this order (M2 design 3.6):
//
//	request meta → request deadline → body limit → authentication → body structure
//
// The generated code wraps the last middleware of its list outermost, so
// the list is in reverse.
func (a *API) Middlewares(bodies *bodyshape.Table) []func(http.Handler) http.Handler {
	inOrder := []func(http.Handler) http.Handler{
		a.requestMeta,
		a.deadline,
		a.bodyLimit,
		a.authenticate,
		bodyshape.Middleware(bodies, a.Errors.BodyError),
	}
	slices.Reverse(inOrder)
	return inOrder
}

// RequestMeta describes the client of a request.
type RequestMeta struct {
	// ClientIP is the connection's peer address, without port and zone; an
	// IPv4-mapped IPv6 address is its IPv4 address. (M2/P2 adds trusted
	// proxies.) The zero Addr when the peer address cannot be parsed.
	ClientIP  netip.Addr
	UserAgent string
}

type metaKey struct{}

// RequestMetaFrom returns the request's meta; the zero value outside the
// per-route middlewares.
func RequestMetaFrom(ctx context.Context) RequestMeta {
	m, _ := ctx.Value(metaKey{}).(RequestMeta)
	return m
}

func (a *API) requestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ip netip.Addr
		if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
			ip = ap.Addr().Unmap().WithZone("")
		}
		meta := RequestMeta{ClientIP: ip, UserAgent: r.UserAgent()}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), metaKey{}, meta)))
	})
}

// deadline bounds the handler: server.write_timeout only fails the writes
// and never cancels the request's context (M0-P2 handoff 5).
func (a *API) deadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), a.requestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bodyLimit makes reading more than server.max_body_bytes fail with
// *http.MaxBytesError, which APIErrors answers with 413.
func (a *API) bodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, a.maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// authenticate denies by default (M2 design 3.6): every operation needs a
// valid bearer token, except the public ones, which never look at it.
func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.public[r.Pattern] {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			a.unauthorized(w, r, errors.New("no bearer token"), false)
			return
		}
		ctx, _, err := a.authenticator.Authenticate(r.Context(), token)
		if err != nil {
			var pe ProblemError
			if errors.As(err, &pe) && pe.ProblemStatus() == http.StatusUnauthorized {
				a.unauthorized(w, r, err, true)
				return
			}
			a.Errors.Write(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// unauthorized answers 401 with WWW-Authenticate (RFC 6750 3). Why the
// credential failed goes to the debug log only.
func (a *API) unauthorized(w http.ResponseWriter, r *http.Request, reason error, invalidToken bool) {
	a.logger.LogAttrs(r.Context(), slog.LevelDebug, "authentication failed",
		slog.String("request_id", RequestID(r.Context())), slog.String("route", r.Pattern), slog.Any("error", reason))
	challenge, detail := "Bearer", "This operation requires a bearer token."
	if invalidToken {
		challenge, detail = `Bearer error="invalid_token"`, "The bearer token is invalid or has expired."
	}
	w.Header().Set("WWW-Authenticate", challenge)
	WriteProblem(w, Problem{
		Status: http.StatusUnauthorized,
		Code:   CodeUnauthorized,
		Title:  http.StatusText(http.StatusUnauthorized),
		Detail: detail,
	})
}

// bearerToken returns the token of an "Authorization: Bearer <token>"
// header; the scheme is case-insensitive (RFC 9110 11.1).
func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}
```

`server/internal/platform/httpserver/api_test.go`：

```go
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

const (
	privateRoute = "POST /api/v0/things"
	publicRoute  = "POST /api/v0/open"
)

type callerKey struct{}

// fakeAuth accepts every token except "bad" (401) and "boom" (a fault).
type fakeAuth struct {
	calls int
	ctx   context.Context // what Authenticate was given
	token string
}

func (f *fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	f.calls++
	f.ctx, f.token = ctx, token
	switch token {
	case "bad":
		return nil, "", problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "Authentication is required."}
	case "boom":
		return nil, "", errors.New("database is down")
	}
	return context.WithValue(ctx, callerKey{}, "caller-"+token), "session:" + token, nil
}

// thingBody is a table as bodyshapegen writes it: {"name": string}, closed.
func thingBody() *bodyshape.Table {
	return &bodyshape.Table{
		Nodes: []bodyshape.Node{
			{Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"name": 1}, Required: []string{"name"}},
			{Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
		},
		Roots: map[string]int{privateRoute: 0, publicRoute: 0},
	}
}

type reached struct {
	called bool
	ctx    context.Context
	body   string
}

// mount registers h on a router behind api's middlewares, applied the way
// the generated code does: for each middleware in the list, h = m(h).
func mount(t *testing.T, api *API, logger *slog.Logger) (*Router, *reached) {
	t.Helper()
	got := &reached{}
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.called, got.ctx = true, r.Context()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			api.Errors.BodyError(w, r, err)
			return
		}
		got.body = string(data)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, m := range api.Middlewares(thingBody()) {
		h = m(h)
	}
	router := NewRouter(logger)
	router.Handle(privateRoute, h)
	router.Handle(publicRoute, h)
	return router, got
}

func newTestAPI(auth Authenticator, logger *slog.Logger) *API {
	return NewAPI(APIConfig{
		Logger:           logger,
		Authenticator:    auth,
		PublicOperations: []string{publicRoute},
		MaxBodyBytes:     64,
		RequestTimeout:   2 * time.Second,
	})
}

func post(path, token, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.RemoteAddr = "203.0.113.7:5555"
	r.Header.Set("User-Agent", "agent/1.0")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) Problem {
	t.Helper()
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return p
}

// The authenticator already sees the request meta and the deadline: both run
// before authentication (M2 design 3.6).
func TestMetaAndDeadlineRunBeforeAuthentication(t *testing.T) {
	auth := &fakeAuth{}
	router, got := mount(t, newTestAPI(auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"a"}`))

	if rec.Code != http.StatusNoContent || !got.called {
		t.Fatalf("status = %d, handler called %v; want 204", rec.Code, got.called)
	}
	meta := RequestMetaFrom(auth.ctx)
	if meta.ClientIP != netip.MustParseAddr("203.0.113.7") || meta.UserAgent != "agent/1.0" {
		t.Errorf("meta seen by the authenticator = %+v", meta)
	}
	deadline, ok := auth.ctx.Deadline()
	if !ok || time.Until(deadline) > 2*time.Second {
		t.Errorf("authenticator's deadline = %v, %v; want within the 2s request timeout", deadline, ok)
	}
	if got.ctx.Value(callerKey{}) != "caller-tok" || got.body != `{"name":"a"}` {
		t.Errorf("handler saw caller %v and body %q; want the authenticator's context and the body unchanged", got.ctx.Value(callerKey{}), got.body)
	}
}

// Without a token the body is never looked at: authentication runs before
// the body structure check.
func TestAuthenticationRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "", `{"nope":1}`))

	if rec.Code != http.StatusUnauthorized || got.called {
		t.Errorf("status = %d, handler called %v; want 401 before the body is checked", rec.Code, got.called)
	}
}

// The body check reads the body through the body limit: the limit runs
// before it, and the check's own read is what fails.
func TestBodyLimitRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"`+strings.Repeat("a", 100)+`"}`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusRequestEntityTooLarge || p.Code != CodePayloadTooLarge || got.called {
		t.Errorf("response = %d %+v, handler called %v; want 413 payload_too_large", rec.Code, p, got.called)
	}
}

func TestBodyCheckAnswersEveryProblemAs400(t *testing.T) {
	router, got := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"extra":1}`))

	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request body does not match the API description.",` +
		`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"name","code":"required","message":"is required"}]}` + "\n"
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want || got.called {
		t.Errorf("response = %d %s, handler called %v; want 400 %s", rec.Code, rec.Body, got.called, want)
	}
}

func TestBodyThatIsNotJSONIs400WithAGenericDetail(t *testing.T) {
	router, _ := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusBadRequest || p.Detail != "The request body could not be decoded." || len(p.Errors) != 0 {
		t.Errorf("response = %d %+v, want 400 with the generic detail", rec.Code, p)
	}
}

func TestAuthenticationDeniesByDefault(t *testing.T) {
	tests := []struct {
		name          string
		route, header string
		status        int
		challenge     string
		authCalls     int
	}{
		{"no token", "/api/v0/things", "", 401, "Bearer", 0},
		{"another scheme", "/api/v0/things", "Basic dXNlcjpwYXNz", 401, "Bearer", 0},
		{"empty bearer", "/api/v0/things", "Bearer ", 401, "Bearer", 0},
		{"invalid token", "/api/v0/things", "Bearer bad", 401, `Bearer error="invalid_token"`, 1},
		{"valid token", "/api/v0/things", "Bearer tok", 204, "", 1},
		{"scheme in lower case", "/api/v0/things", "bearer tok", 204, "", 1},
		{"authenticator fault", "/api/v0/things", "Bearer boom", 500, "", 1},
		{"public without a token", "/api/v0/open", "", 204, "", 0},
		{"public ignores a bad token", "/api/v0/open", "Bearer bad", 204, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuth{}
			router, _ := mount(t, newTestAPI(auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
			req := post(tt.route, "", `{"name":"a"}`)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			rec := serve(router, req)

			if rec.Code != tt.status || rec.Header().Get("WWW-Authenticate") != tt.challenge || auth.calls != tt.authCalls {
				t.Errorf("response = %d, WWW-Authenticate %q, authenticator called %d times; want %d, %q, %d",
					rec.Code, rec.Header().Get("WWW-Authenticate"), auth.calls, tt.status, tt.challenge, tt.authCalls)
			}
			if tt.status == 401 {
				if p := decodeProblem(t, rec); p.Code != CodeUnauthorized {
					t.Errorf("problem code = %q, want unauthorized", p.Code)
				}
			}
		})
	}
}

// Why a credential failed goes to the debug log, never into the response.
func TestAuthenticationFailureIsLoggedAtDebugLevel(t *testing.T) {
	logger, logs := captureLogs(t)
	router, _ := mount(t, newTestAPI(&fakeAuth{}, logger), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))

	entry := findLog(logs(), "authentication failed")
	if entry == nil || entry["level"] != "DEBUG" || entry["route"] != privateRoute || entry["error"] != "Authentication is required." {
		t.Errorf("log = %v, want the reason at debug level", entry)
	}
	if strings.Contains(rec.Body.String(), "Authentication is required.") {
		t.Errorf("body %s carries the authenticator's reason", rec.Body)
	}
}

func TestRequestMetaClientIP(t *testing.T) {
	tests := []struct{ remote, want string }{
		{"203.0.113.7:5555", "203.0.113.7"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"[::ffff:192.0.2.1]:80", "192.0.2.1"},
		{"[fe80::1%en0]:80", "fe80::1"},
	}
	for _, tt := range tests {
		auth := &fakeAuth{}
		router, _ := mount(t, newTestAPI(auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
		req := post("/api/v0/things", "tok", `{"name":"a"}`)
		req.RemoteAddr = tt.remote

		serve(router, req)

		if got := RequestMetaFrom(auth.ctx).ClientIP; got != netip.MustParseAddr(tt.want) {
			t.Errorf("RemoteAddr %s: ClientIP = %v, want %s", tt.remote, got, tt.want)
		}
	}
}

func TestRequestMetaOutsideTheMiddlewaresIsZero(t *testing.T) {
	if m := RequestMetaFrom(context.Background()); m.ClientIP.IsValid() || m.UserAgent != "" {
		t.Errorf("RequestMetaFrom() = %+v, want the zero value", m)
	}
}
```

`server/internal/platform/httpserver/contract_test.go`：

```go
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// TestPlatformProblemsMatchTheContract keeps Problem in step with the Problem
// schema of api/common.yaml: every problem the platform writes must validate.
func TestPlatformProblemsMatchTheContract(t *testing.T) {
	contract := apitest.Load(t)
	discard := slog.New(slog.DiscardHandler)
	errs := NewAPIErrors(discard)
	notReady := Check{Name: "database", Run: func(context.Context) error { return errors.New("down") }}
	tests := []struct {
		name   string
		h      http.Handler
		target string
		status int
	}{
		{"unknown API path", NewRouter(discard), "/api/v0/nope", http.StatusNotFound},
		{"not ready", NewRouter(discard, notReady), "/readyz", http.StatusServiceUnavailable},
		{"panic", middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }), discard), "/api/v0/boom", http.StatusInternalServerError},
		{"bad request", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.BadRequest(w, r, errors.New("Invalid format for parameter limit"))
		}), "/api/v0/things?limit=x", http.StatusBadRequest},
		{"internal error", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.Write(w, r, errors.New("boom"))
		}), "/api/v0/things", http.StatusInternalServerError},
		{"body not decoded", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.BodyError(w, r, errors.New("EOF"))
		}), "/api/v0/things", http.StatusBadRequest},
		{"payload too large", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.Write(w, r, &http.MaxBytesError{Limit: 1024})
		}), "/api/v0/things", http.StatusRequestEntityTooLarge},
		{"unauthorized", newTestAPI(&fakeAuth{}, discard).authenticate(http.NotFoundHandler()), "/api/v0/things", http.StatusUnauthorized},
		{"field errors", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.Write(w, r, problemErr{
				status: http.StatusUnprocessableEntity, code: "validation_failed", detail: "The request has invalid values.",
				fields: []error{fieldErr{"name", "required", "is required"}, fieldErr{"password", "common_password", "is too common"}},
			})
		}), "/api/v0/issues", http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(tt.h, httptest.NewRequest(http.MethodGet, tt.target, nil))

			if rec.Code != tt.status || rec.Header().Get("Content-Type") != ContentTypeProblem {
				t.Fatalf("response = %d %s, want %d problem+json", rec.Code, rec.Header().Get("Content-Type"), tt.status)
			}
			contract.CheckSchema(t, "Problem", rec.Body.Bytes())
		})
	}
}
```

- [ ] **Step 5: `api/common.yaml` 的 `FieldError.code`，重新生成**

`api/common.yaml`：

```yaml
# 各模块共用的组件，Go 类型生成在 server/internal/platform/httpserver/apigen（make gen-go）。
# 模块这样引用：$ref: '../common.yaml#/components/schemas/<名字>'。
# 只放 schemas 和 parameters，不放 responses：跨文件引用 response 时，oapi-codegen
# 要求共享包也生成 strict server（见 docs/v0/M0-foundation/specs/P3-api-contract.md 2.4）。
openapi: 3.1.0
info:
  title: Nerve common components
  version: v0
components:
  schemas:
    Problem:
      description: >-
        RFC 9457 problem details (v0 design 3.5). `title` is the HTTP status
        phrase, `detail` explains this occurrence, and clients branch on `code`.
        Must match httpserver.Problem; the platform's contract test checks it.
      type: object
      additionalProperties: false
      required: [status, code, title]
      properties:
        status:
          description: HTTP status code.
          type: integer
        code:
          description: >-
            Stable error code. Platform codes have no prefix (bad_request,
            unauthorized, not_found, payload_too_large, validation_failed,
            server_busy, internal_error, not_ready); module codes are prefixed
            with the module, e.g. identity.email_taken. Each operation lists
            the codes it can answer in x-problem-codes.
          type: string
        title:
          description: HTTP status phrase, e.g. "Not Found".
          type: string
        detail:
          description: What went wrong in this occurrence.
          type: string
        errors:
          description: The invalid fields of the request.
          type: array
          items:
            $ref: '#/components/schemas/FieldError'
    FieldError:
      description: >-
        One invalid field of a request. Clients show text looked up by `code`;
        `message` is an English explanation for developers. Must match
        httpserver.FieldError and the field codes of internal/shared.
      type: object
      additionalProperties: false
      required: [field, code, message]
      properties:
        field:
          description: The field's path in the request body, e.g. password or tags[1].name.
          type: string
        code:
          description: What is wrong with the field.
          type: string
          enum:
            - required
            - invalid_format
            - too_short
            - too_long
            - out_of_range
            - not_allowed
            - weak_password
            - common_password
            - must_be_future
            - contains_url
        message:
          type: string
```

Run: `make gen`
Expected: `server/internal/platform/httpserver/apigen/components.gen.go` 77 行，SHA-256 `a934f66c6b667a4edbb9ec525d9dbce84a41d7758af7977e321278f06d717f63`（这是它的最终版本）；`api/dist/openapi.yaml` 和 `schema.gen.ts` 的 `FieldError` 多出 `code`（这两个文件的最终版本在 Task 10）。

`apitest` 的样例随之修改，照下面的差异改 `server/internal/platform/httpserver/apitest/apitest_test.go`：

```diff
--- a/server/internal/platform/httpserver/apitest/apitest_test.go
+++ b/server/internal/platform/httpserver/apitest/apitest_test.go
@@ -180,7 +180,9 @@
 		valid              bool
 	}{
 		{"problem", "Problem", `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}`, true},
-		{"problem with field errors", "Problem", `{"status":422,"code":"issue.invalid","title":"Unprocessable Entity","errors":[{"field":"name","message":"is required"}]}`, true},
+		{"problem with field errors", "Problem", `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","errors":[{"field":"name","code":"required","message":"is required"}]}`, true},
+		{"field error without code", "Problem", `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","errors":[{"field":"name","message":"is required"}]}`, false},
+		{"field error with an unknown code", "Problem", `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","errors":[{"field":"name","code":"blank","message":"is required"}]}`, false},
 		{"problem without code", "Problem", `{"status":404,"title":"Not Found"}`, false},
 		{"problem with an unknown member", "Problem", `{"status":404,"code":"not_found","title":"Not Found","instance":"/x"}`, false},
 		{"unknown schema", "Nope", `{}`, false},
```

- [ ] **Step 6: 使用方改为 `Router`、`BodyError`、`Write`（过渡）**

`NewMux` 和 `InternalError` 已删除，`bootstrap` 和 `instance` 随之修改。模块入口的最终形状（`Register(router, api)`、`PublicOperations`）在 Task 11 与 `identity` 一起接线。

`server/internal/bootstrap/app.go`：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -49,17 +49,17 @@
 		pool.Close()
 		return nil, err
 	}
-	mux := httpserver.NewMux(logger,
+	router := httpserver.NewRouter(logger,
 		httpserver.Check{Name: "database", Run: pool.Ping},
 		httpserver.Check{Name: "migrations", Run: migrator.CheckUpToDate},
 	)
-	// Modules mount their generated routes on this root mux, next to the
+	// Modules mount their generated routes on this root router, next to the
 	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
-	instance.New().Register(mux, httpserver.NewAPIErrors(logger))
+	instance.New().Register(router, httpserver.NewAPIErrors(logger))
 	// The web UI takes every path no other pattern claims. It must be the
 	// method-less "/": "GET /" and the method-less "/api/" would conflict.
-	mux.Handle("/", webui.Handler(webFiles))
-	return &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator, handler: mux}, nil
+	router.Handle("/", webui.Handler(webFiles))
+	return &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator, handler: router}, nil
 }
 
 // run applies pending migrations when database.auto_migrate is on, then
```

`server/internal/modules/instance/module.go`：

```diff
--- a/server/internal/modules/instance/module.go
+++ b/server/internal/modules/instance/module.go
@@ -4,8 +4,6 @@
 package instance
 
 import (
-	"net/http"
-
 	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
 	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
 	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
@@ -22,8 +20,8 @@
 	return &Module{getInfo: app.NewGetInfo(buildinfo.Source{})}
 }
 
-// Register mounts the module's API on mux, the root router from
-// httpserver.NewMux; apiErrors answers binding and handler errors.
-func (m *Module) Register(mux *http.ServeMux, apiErrors httpserver.APIErrors) {
-	httpadapter.Register(mux, m.getInfo, apiErrors)
+// Register mounts the module's API on router, the root router from
+// httpserver.NewRouter; apiErrors answers binding and handler errors.
+func (m *Module) Register(router *httpserver.Router, apiErrors httpserver.APIErrors) {
+	httpadapter.Register(router, m.getInfo, apiErrors)
 }
```

`server/internal/modules/instance/adapter/http/handler.go`：

```diff
--- a/server/internal/modules/instance/adapter/http/handler.go
+++ b/server/internal/modules/instance/adapter/http/handler.go
@@ -5,24 +5,23 @@
 
 import (
 	"context"
-	"net/http"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http/gen"
 	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
 )
 
-// Register mounts the module's routes on mux, the root router from
-// httpserver.NewMux. They are more specific than the platform's /api/
+// Register mounts the module's routes on router, the root router from
+// httpserver.NewRouter. They are more specific than the platform's /api/
 // fallback, which keeps answering every other API path with a 404 problem.
-// Binding and handler errors are answered as problem+json by apiErrors.
-func Register(mux *http.ServeMux, getInfo *app.GetInfo, apiErrors httpserver.APIErrors) {
+// Binding, body and handler errors are answered as problem+json by apiErrors.
+func Register(router *httpserver.Router, getInfo *app.GetInfo, apiErrors httpserver.APIErrors) {
 	strict := gen.NewStrictHandlerWithOptions(handler{getInfo: getInfo}, nil, gen.StrictHTTPServerOptions{
-		RequestErrorHandlerFunc:  apiErrors.BadRequest,
-		ResponseErrorHandlerFunc: apiErrors.InternalError,
+		RequestErrorHandlerFunc:  apiErrors.BodyError,
+		ResponseErrorHandlerFunc: apiErrors.Write,
 	})
 	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
-		BaseRouter:       mux,
+		BaseRouter:       router,
 		ErrorHandlerFunc: apiErrors.BadRequest,
 	})
 }
```

`server/internal/modules/instance/adapter/http/handler_test.go`：

```diff
--- a/server/internal/modules/instance/adapter/http/handler_test.go
+++ b/server/internal/modules/instance/adapter/http/handler_test.go
@@ -20,13 +20,14 @@
 
 func TestGetInstanceMatchesTheContract(t *testing.T) {
 	contract := apitest.Load(t)
-	mux := http.NewServeMux()
+	logger := slog.New(slog.DiscardHandler)
+	router := httpserver.NewRouter(logger)
 	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"})
-	httpadapter.Register(mux, getInfo, httpserver.NewAPIErrors(slog.New(slog.DiscardHandler)))
+	httpadapter.Register(router, getInfo, httpserver.NewAPIErrors(logger))
 	req := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
 	rec := httptest.NewRecorder()
 
-	mux.ServeHTTP(rec, req)
+	router.ServeHTTP(rec, req)
 	res := rec.Result()
 
 	contract.CheckResponse(t, req, res)
```

- [ ] **Step 7: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过（`schema.gen.ts` 变了，前端的类型检查照常通过）。

- [ ] **Step 8: 提交**

```bash
git add server/internal/platform/httpserver api server/internal/bootstrap/app.go server/internal/modules/instance web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M2/P1): router, per-route API middlewares, deny-by-default authentication and one road to problems

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** `api_test.go` 的顺序测试和默认拒绝的 9 个情况通过；`contract_test.go` 覆盖 400、401、413、422、500；`NewMux`、`(*APIErrors).InternalError`、未导出的 `requestID` 在仓库中没有引用（`grep -rn -e "NewMux" -e "\.InternalError(" -e "requestID(" server` 没有输出）。

---

### Task 5: 接口描述约定：`security`、`x-problem-codes`、模块模板；`apitest` 的问题码核对和 `CheckRequest`

**Files:**
- Modify: `api/openapi.yaml`（过渡：Task 10 加上 identity）、`api/modules/instance.yaml`
- Modify: `server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`
- Create: `server/internal/modules/instance/adapter/http/main_test.go`
- Modify: `server/internal/platform/httpserver/apitest/apitest.go`、`apitest_test.go`、`rules_test.go`、`rules_cases_test.go`
- Create: `server/internal/platform/httpserver/apitest/problems.go`、`problems_test.go`
- Generate: `api/dist/openapi.yaml`（`schema.gen.ts` 不变：openapi-typescript 不输出 `security` 和 `x-` 扩展，原型核对过）

**Interfaces:**
- Consumes: M0 的 `apitest.Load`、`CheckResponse`、`authoringViolations`。
- Produces:
  - 契约写法：每个操作显式写 `security` 和 `x-problem-codes`；顶层 `x-problem-codes: [bad_request, payload_too_large, internal_error]`；`securitySchemes.bearer` 同时写在 `api/openapi.yaml` 和每个模块文件（spec 2.9）；
  - `(*Contract).CheckRequest(t, req)`；`CheckResponse` 另核对 problem 的码并记下；`apitest.Main(m, module)`；
  - oapi-codegen 的模块模板：`nullable-type: true`，`type-mapping` 把 `uuid` 映射为标准库 `uuid.UUID`、`email` 映射为 `string`（以后每个模块照抄）。

**Tests:**
- `problems_test.go`：`TestValidateResponseChecksTheProblemCode`（6 个：操作自己的码、顶层的码通过；未声明的码、公开操作答 `unauthorized`、别的操作的码失败；需要令牌的操作答 `unauthorized` 通过）；`TestCheckResponseRecordsTheAnsweredCode`；`TestUnansweredListsTheCodesNoTestAnswered`；`TestValidateRequest`（5 个：合法请求体、缺必填、多字段、不核对 `security`、没有的路径）。
- `rules_cases_test.go`：新增 `TestModuleCodesPass`；`TestAuthoringRulesReportViolations` 新增 9 个反例（没有顶层码、顶层有模块码、没有 `security`、未声明的 scheme、操作没有码、码不是列表、码的写法错、别的模块的前缀、不带前缀又不是平台码），每个核对完整的报错原文。
- `apitest_test.go`：`TestComponentNamesAreTypeNames` 跳过 `securitySchemes`。
- `instance/adapter/http/main_test.go`：`TestMain` 用 `apitest.Main(m, "instance")`（`getInstance` 声明 `[]`，没有需要答出的码）。

- [ ] **Step 1: 契约的写法**

`api/openapi.yaml`（过渡版本；Task 10 加上 identity 的 tag 和两个路径）：

```yaml
# Nerve 接口描述的入口：列出所有路径，指向各模块的描述文件。
# make gen-web 用 Redocly 把它打包成 dist/openapi.yaml。
openapi: 3.1.0
info:
  title: Nerve API
  version: v0
  description: >-
    The HTTP API of Nerve. The web app is one client of it; every account can
    use every endpoint the same way. Every operation lists the problem codes
    it can answer in x-problem-codes; the top-level x-problem-codes can come
    from every operation, and an operation that needs a bearer token can also
    answer unauthorized.
  license:
    name: AGPL-3.0-only
    identifier: AGPL-3.0-only
# 所有操作都可能返回的平台错误码，只写在这里（M2 设计 3.11）。rate_limited 随 M2/P2 的限流加入
x-problem-codes: [bad_request, payload_too_large, internal_error]
tags:
  - name: instance
    description: What this instance runs.
paths:
  /api/v0/instance:
    $ref: 'modules/instance.yaml#/paths/~1api~1v0~1instance'
components:
  # 模块文件的 securitySchemes 在打包时被丢掉，操作引用的 scheme 写在这里（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
      description: >-
        An access token (JWT) from register, login or refresh, or a personal
        access token (nrv_pat_…).
```

`api/modules/instance.yaml`：

```yaml
# instance 模块的接口。生成的 Go 代码在
# server/internal/modules/instance/adapter/http/gen（make gen-go）。
openapi: 3.1.0
info:
  title: Nerve instance API
  version: v0
paths:
  /api/v0/instance:
    get:
      operationId: getInstance
      tags: [instance]
      summary: Describe this instance
      description: >-
        Reports the product, the build and the API version this instance runs.
        Public: needs no authentication.
      security: []
      x-problem-codes: []
      responses:
        '200':
          description: What this instance runs.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/InstanceInfo'
        default:
          $ref: '#/components/responses/Problem'
components:
  # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  responses:
    Problem:
      description: Error (RFC 9457 problem details).
      content:
        application/problem+json:
          schema:
            $ref: '../common.yaml#/components/schemas/Problem'
  schemas:
    InstanceInfo:
      type: object
      additionalProperties: false
      required: [product, version, commit, api_version]
      properties:
        product:
          description: Product name.
          type: string
          examples: [Nerve]
        version:
          description: Product version of the running build.
          type: string
          examples: [0.1.0-dev]
        commit:
          description: Git revision of the running build; "unknown" when the build carries no VCS stamp.
          type: string
        api_version:
          description: Version of this HTTP API; every path starts with /api/{api_version}.
          type: string
          enum: [v0]
```

`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`（模块模板）：

```yaml
# oapi-codegen 配置：api/modules/instance.yaml → server.gen.go。
# 由 make gen-go 在 server/ 下执行，路径相对于 server/。
package: gen
output: internal/modules/instance/adapter/http/gen/server.gen.go
generate:
  models: true
  std-http-server: true
  strict-server: true
compatibility:
  # 枚举常量总是带类型名前缀：以后别的枚举出现同名的值，也不会改掉已有常量的名字
  always-prefix-enum-values: true
# output-options 是每个模块照抄的模板（M2 设计 3.12），bodyshapegen 读同一份 type-mapping
output-options:
  name-normalizer: ToCamelCaseWithInitialisms
  # 可为空又可省略的字段生成 nullable.Nullable[T]：PATCH 能区分"没传"和"传 null"
  nullable-type: true
  type-mapping:
    string:
      formats:
        # 标准库的 uuid：默认的 openapi_types.UUID 会带进 github.com/google/uuid
        uuid:
          type: uuid.UUID
          import: uuid
        # 邮箱的格式由领域层校验（422）；默认的 openapi_types.Email 在解码时自己校验，绕过这一层
        email:
          type: string
import-mapping:
  # common.yaml 的组件生成在共享包 apigen 里，这里只引用
  ../common.yaml: github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen
```

- [ ] **Step 2: `apitest` 的问题码核对、`CheckRequest` 和 `Main`**

`server/internal/platform/httpserver/apitest/apitest.go`：

```go
// Package apitest checks HTTP responses against nerve's OpenAPI contract,
// api/dist/openapi.yaml, with kin-openapi. Architecture rule 8 lets only
// tests import this package, which keeps it out of the nerve binary. That
// kin-openapi stays out of the binary by any other route too (generated code
// with an embedded spec, a validator middleware) is guarded separately, by
// archtest's check of the binary's transitive dependencies.
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// Contract is the loaded and validated OpenAPI document.
type Contract struct {
	doc    *openapi3.T
	router routers.Router
}

// Load reads api/dist/openapi.yaml and fails t unless it is a valid OpenAPI
// document.
func Load(t testing.TB) *Contract {
	t.Helper()
	c, err := load(contractPath())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// CheckResponse fails t unless res, the response to req, is documented by
// req's operation: the status, the Content-Type, the body schema and, for a
// problem, a code the operation may answer (x-problem-codes, M2 design
// 3.11). The code is recorded for Main. res.Body stays readable for the
// caller.
func (c *Contract) CheckResponse(t testing.TB, req *http.Request, res *http.Response) {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	res.Body = io.NopCloser(bytes.NewReader(body))
	if err := c.validateResponse(req, res.StatusCode, res.Header, body); err != nil {
		t.Errorf("%s %s answered %d %s: %v", req.Method, req.URL.Path, res.StatusCode, body, err)
	}
}

// CheckRequest fails t unless req, a request a test is about to send, is
// documented: path, method, parameters and body. Security is not checked:
// tests send tokens the contract cannot judge. req.Body stays readable. A
// test that sends a request breaking the contract on purpose skips it.
func (c *Contract) CheckRequest(t testing.TB, req *http.Request) {
	t.Helper()
	if err := c.validateRequest(req); err != nil {
		t.Errorf("%s %s does not follow the contract: %v", req.Method, req.URL.Path, err)
	}
}

// CheckSchema fails t unless body is a JSON value valid against the schema
// components.schemas[name], e.g. "Problem".
func (c *Contract) CheckSchema(t testing.TB, name string, body []byte) {
	t.Helper()
	if err := c.validateSchema(name, body); err != nil {
		t.Errorf("%s: %v", body, err)
	}
}

// apiDir locates the repository's api/ from this file, five directories below
// the repository root (tests run without -trimpath).
func apiDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "api")
}

// contractPath is api/dist/openapi.yaml, the bundled contract.
func contractPath() string {
	return filepath.Join(apiDir(), "dist", "openapi.yaml")
}

// newLoader returns the loader for contract documents. IncludeOrigin records
// the keywords each schema spells out: the authoring-rules test needs them to
// see `const: null` and `nullable: false`, which the decoded fields cannot
// tell from an absent keyword.
func newLoader() *openapi3.Loader {
	loader := openapi3.NewLoader()
	loader.IncludeOrigin = true
	return loader
}

func load(path string) (*Contract, error) {
	loader := newLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
	}
	// Route by path and method only: with servers, the router would also
	// match each request's scheme and host, which no test host satisfies.
	doc.Servers = nil
	for _, item := range doc.Paths.Map() {
		item.Servers = nil
		for _, op := range item.Operations() {
			op.Servers = nil
		}
	}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("route %s: %w", path, err)
	}
	return &Contract{doc: doc, router: router}, nil
}

func (c *Contract) validateResponse(req *http.Request, status int, header http.Header, body []byte) error {
	route, pathParams, err := c.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("no documented operation: %w", err)
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: pathParams, Route: route},
		Status:                 status,
		Header:                 header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, MultiError: true},
	}
	in.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(context.Background(), in); err != nil {
		return err
	}
	return c.checkProblemCode(route.Operation, header, body)
}

func (c *Contract) validateRequest(req *http.Request) error {
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return fmt.Errorf("read request body: %w", err)
		}
	}
	defer func() { req.Body = io.NopCloser(bytes.NewReader(body)) }()
	req.Body = io.NopCloser(bytes.NewReader(body))
	route, pathParams, err := c.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("no documented operation: %w", err)
	}
	return openapi3filter.ValidateRequest(context.Background(), &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, MultiError: true},
	})
}

func (c *Contract) validateSchema(name string, body []byte) error {
	schema, ok := c.doc.Components.Schemas[name]
	if !ok {
		return fmt.Errorf("no schema %q in the contract", name)
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("not JSON: %w", err)
	}
	opts := []openapi3.SchemaValidationOption{openapi3.MultiErrors()}
	if c.doc.IsOpenAPI31OrLater() {
		opts = append(opts, openapi3.EnableJSONSchema2020())
	}
	return schema.Value.VisitJSON(value, opts...)
}
```

`server/internal/platform/httpserver/apitest/problems.go`：

```go
package apitest

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// problemCodesKey is the extension that lists the problem codes an
// operation can answer; at the top level, the codes every operation can
// answer (M2 design 3.11).
const problemCodesKey = "x-problem-codes"

// platformCodes are the codes without a module prefix (M2 design 3.11).
var platformCodes = []string{
	"bad_request", "unauthorized", "not_found", "payload_too_large", "validation_failed",
	"rate_limited", "internal_error", "not_ready", "server_busy",
}

// codePattern is the spelling of a code: a platform code, or a module's
// code prefixed with the module, e.g. identity.email_taken.
var codePattern = regexp.MustCompile(`^([a-z]+\.)?[a-z_]+$`)

// problemCodes reads the x-problem-codes of ext: present reports whether the
// extension is there at all.
func problemCodes(ext map[string]any) (codes []string, present bool, err error) {
	raw, present := ext[problemCodesKey]
	if !present {
		return nil, false, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, true, fmt.Errorf("%s is not a list", problemCodesKey)
	}
	for _, item := range list {
		code, ok := item.(string)
		if !ok {
			return nil, true, fmt.Errorf("%s holds %v, which is not a string", problemCodesKey, item)
		}
		codes = append(codes, code)
	}
	return codes, true, nil
}

// needsToken reports whether op declares a bearer requirement; such an
// operation can also answer unauthorized.
func needsToken(op *openapi3.Operation) bool {
	return op.Security != nil && len(*op.Security) > 0
}

// checkProblemCode fails a problem whose code op may not answer: the
// top-level codes, op's own, and unauthorized when op needs a token. The
// code is recorded for Main.
func (c *Contract) checkProblemCode(op *openapi3.Operation, header http.Header, body []byte) error {
	if media, _, _ := mime.ParseMediaType(header.Get("Content-Type")); media != "application/problem+json" {
		return nil
	}
	var p struct{ Code string }
	if err := json.Unmarshal(body, &p); err != nil {
		return fmt.Errorf("problem body: %w", err)
	}
	top, _, err := problemCodes(c.doc.Extensions)
	if err != nil {
		return err
	}
	own, _, err := problemCodes(op.Extensions)
	if err != nil {
		return err
	}
	allowed := slices.Concat(top, own)
	if needsToken(op) {
		allowed = append(allowed, "unauthorized")
	}
	if !slices.Contains(allowed, p.Code) {
		return fmt.Errorf("problem code %q is not declared: %s may answer %q", p.Code, op.OperationID, allowed)
	}
	answered.record(op.OperationID, p.Code)
	return nil
}

// answered records, process-wide, the problem codes each operation answered
// through CheckResponse.
var answered = &recorder{codes: map[string]map[string]bool{}}

type recorder struct {
	mu    sync.Mutex
	codes map[string]map[string]bool // operationId → code → answered
}

func (r *recorder) record(operationID, code string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.codes[operationID] == nil {
		r.codes[operationID] = map[string]bool{}
	}
	r.codes[operationID][code] = true
}

func (r *recorder) snapshot() map[string]map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]map[string]bool, len(r.codes))
	for op, codes := range r.codes {
		out[op] = maps.Clone(codes)
	}
	return out
}

// Main runs the tests of a module's HTTP adapter, then fails the run unless
// every code that the module's operations declare in their own
// x-problem-codes was answered by some test through CheckResponse (M2 design
// 3.11): a declared code that nothing answers is a wrong contract. A run
// that failed already, or was narrowed by -run, -skip or -short, is not
// checked. Every module's adapter/http tests use it:
//
//	func TestMain(m *testing.M) { apitest.Main(m, "identity") }
func Main(m *testing.M, module string) {
	code := m.Run()
	if code == 0 && !narrowed() {
		doc, err := loadModule(module)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if missing := unanswered(doc, answered.snapshot()); len(missing) > 0 {
			fmt.Fprintf(os.Stderr, "api/modules/%s.yaml declares problem codes that no test answered through CheckResponse:\n", module)
			for _, m := range missing {
				fmt.Fprintln(os.Stderr, "  "+m)
			}
			code = 1
		}
	}
	os.Exit(code)
}

// narrowed reports whether this run leaves tests out.
func narrowed() bool {
	for _, name := range []string{"test.run", "test.skip"} {
		if f := flag.Lookup(name); f != nil && f.Value.String() != "" {
			return true
		}
	}
	return testing.Short()
}

// unanswered lists "operationId: code" for each code of doc's operations
// that answered does not hold, sorted.
func unanswered(doc *openapi3.T, answered map[string]map[string]bool) []string {
	var missing []string
	for _, item := range doc.Paths.Map() {
		for _, op := range item.Operations() {
			codes, _, _ := problemCodes(op.Extensions)
			for _, code := range codes {
				if !answered[op.OperationID][code] {
					missing = append(missing, op.OperationID+": "+code)
				}
			}
		}
	}
	slices.Sort(missing)
	return missing
}

// loadModule loads api/modules/<module>.yaml.
func loadModule(module string) (*openapi3.T, error) {
	loader := newLoader()
	loader.IsExternalRefsAllowed = true // module files reference ../common.yaml
	path := filepath.Join(apiDir(), "modules", module+".yaml")
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	return doc, nil
}

// moduleNames lists the modules that have an api/modules/<m>.yaml.
func moduleNames() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(apiDir(), "modules", "*.yaml"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f[:len(f)-len(".yaml")]))
	}
	return names, nil
}
```

`server/internal/platform/httpserver/apitest/problems_test.go`：

```go
package apitest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// thingsContract has a public operation with a body and one that needs a
// token, each with its own problem codes.
const thingsContract = `
openapi: 3.1.0
info: {title: things, version: v0}
x-problem-codes: [bad_request, internal_error]
paths:
  /api/v0/things:
    post:
      operationId: createThing
      security: []
      x-problem-codes: [things.taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [name]
              properties:
                name: {type: string}
      responses:
        '204': {description: created}
        default: {$ref: '#/components/responses/Problem'}
    get:
      operationId: listThings
      security: [{bearer: []}]
      x-problem-codes: []
      responses:
        '204': {description: none}
        default: {$ref: '#/components/responses/Problem'}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
  responses:
    Problem:
      description: Error.
      content:
        application/problem+json:
          schema:
            type: object
            required: [code]
            properties:
              code: {type: string}
`

func contractFrom(t *testing.T, doc string) *Contract {
	t.Helper()
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestValidateResponseChecksTheProblemCode(t *testing.T) {
	c := contractFrom(t, thingsContract)
	tests := []struct {
		name, method, code string
		valid              bool
	}{
		{"the operation's own code", "POST", "things.taken", true},
		{"a top-level code", "POST", "internal_error", true},
		{"an undeclared code", "POST", "not_found", false},
		{"unauthorized from a public operation", "POST", "unauthorized", false},
		{"unauthorized from an operation that needs a token", "GET", "unauthorized", true},
		{"another operation's code", "GET", "things.taken", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v0/things", nil)
			header := http.Header{"Content-Type": {"application/problem+json"}}
			err := c.validateResponse(req, 409, header, []byte(`{"code":"`+tt.code+`"}`))
			if (err == nil) != tt.valid {
				t.Errorf("validateResponse() = %v, want valid = %v", err, tt.valid)
			}
		})
	}
}

func TestCheckResponseRecordsTheAnsweredCode(t *testing.T) {
	c := contractFrom(t, thingsContract)
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/problem+json")
	rec.WriteHeader(http.StatusConflict)
	_, _ = rec.WriteString(`{"code":"things.taken"}`)

	c.CheckResponse(t, httptest.NewRequest(http.MethodPost, "/api/v0/things", nil), rec.Result())

	if !answered.snapshot()["createThing"]["things.taken"] {
		t.Error("createThing: things.taken was not recorded")
	}
}

func TestUnansweredListsTheCodesNoTestAnswered(t *testing.T) {
	doc := parse(t, strings.Replace(thingsContract, "x-problem-codes: []", "x-problem-codes: [things.gone, not_found]", 1))
	got := unanswered(doc, map[string]map[string]bool{"listThings": {"not_found": true}, "other": {"things.gone": true}})

	want := []string{"createThing: things.taken", "listThings: things.gone"}
	if !slices.Equal(got, want) {
		t.Errorf("unanswered() = %q, want %q", got, want)
	}
}

func TestValidateRequest(t *testing.T) {
	c := contractFrom(t, thingsContract)
	tests := []struct {
		name, method, path, body string
		valid                    bool
	}{
		{"documented body", "POST", "/api/v0/things", `{"name":"a"}`, true},
		{"missing required field", "POST", "/api/v0/things", `{}`, false},
		{"undocumented field", "POST", "/api/v0/things", `{"name":"a","x":1}`, false},
		{"security is not checked", "GET", "/api/v0/things", "", true},
		{"undocumented path", "GET", "/api/v0/nope", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			err := c.validateRequest(req)
			if (err == nil) != tt.valid {
				t.Errorf("validateRequest() = %v, want valid = %v", err, tt.valid)
			}
			if body, _ := io.ReadAll(req.Body); string(body) != tt.body {
				t.Errorf("body after validation = %q, want %q", body, tt.body)
			}
		})
	}
}
```

- [ ] **Step 3: 写法检查**

`server/internal/platform/httpserver/apitest/rules_test.go`：

```go
package apitest

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// TestContractFollowsAuthoringRules holds api/dist/openapi.yaml to the
// authoring rules of spec P3 2.4. Generated Go code and the TS client rely
// on them, but neither the bundler nor doc.Validate enforces them.
func TestContractFollowsAuthoringRules(t *testing.T) {
	c := Load(t)
	if c.doc.Paths.Len() == 0 {
		t.Fatal("the contract has no paths")
	}
	for _, v := range authoringViolations(c.doc, pathOwners(t)) {
		t.Error(v)
	}
}

// pathOwners maps every path of api/modules/*.yaml to its module.
func pathOwners(t *testing.T) map[string]string {
	t.Helper()
	names, err := moduleNames()
	if err != nil || len(names) == 0 {
		t.Fatalf("module files = %q, %v; want at least one", names, err)
	}
	owners := map[string]string{}
	for _, name := range names {
		doc, err := loadModule(name)
		if err != nil {
			t.Fatal(err)
		}
		for path := range doc.Paths.Map() {
			owners[path] = name
		}
	}
	return owners
}

// Go serves every path of api/modules/*.yaml, but the TS client and the
// contract tests know only the paths that the root api/openapi.yaml lists.
func TestRootListsEveryModulePath(t *testing.T) {
	c := Load(t)
	owners := pathOwners(t)
	for _, path := range slices.Sorted(maps.Keys(owners)) {
		if c.doc.Paths.Value(path) == nil {
			t.Errorf("api/modules/%s.yaml declares %s, which api/dist/openapi.yaml lacks: list it in api/openapi.yaml and run make gen",
				owners[path], path)
		}
	}
}

// authoringViolations reports where doc breaks the authoring rules of spec P3
// 2.4 and M2 design 3.11–3.12; owners maps each path to the module file that
// declares it. It takes the document so that each check is proven on
// hand-built bad documents (rules_cases_test.go) as well as applied to the
// real contract.
func authoringViolations(doc *openapi3.T, owners map[string]string) []string {
	r := &ruleCheck{doc: doc, owners: owners, lowerCamel: regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)}
	r.topCodes()
	for _, path := range slices.Sorted(maps.Keys(doc.Paths.Map())) {
		r.path(path, doc.Paths.Value(path))
	}
	if doc.Components != nil {
		r.components(doc.Components)
	}
	return r.found
}

// ruleCheck walks one document and collects its violations: path checks the
// /api/v0/ prefix, operation the operationId, tags, default problem,
// security and problem codes, closedObject additionalProperties, and schema
// the keywords that oapi-codegen mistranslates.
type ruleCheck struct {
	doc        *openapi3.T
	owners     map[string]string
	lowerCamel *regexp.Regexp
	found      []string
}

func (r *ruleCheck) report(where, format string, args ...any) {
	r.found = append(r.found, where+": "+fmt.Sprintf(format, args...))
}

func (r *ruleCheck) path(path string, item *openapi3.PathItem) {
	if !strings.HasPrefix(path, "/api/v0/") {
		r.report(path, "does not start with /api/v0/")
	}
	for _, p := range item.Parameters {
		r.parameter(path+" parameters/"+p.Value.Name, p)
	}
	ops := item.Operations()
	for _, method := range slices.Sorted(maps.Keys(ops)) {
		r.operation(method+" "+path, r.owners[path], ops[method])
	}
}

// topCodes checks the codes every operation can answer: only platform
// codes, listed once at the top level.
func (r *ruleCheck) topCodes() {
	codes, present, err := problemCodes(r.doc.Extensions)
	switch {
	case err != nil:
		r.report("top level", "%v", err)
	case !present:
		r.report("top level", "has no %s", problemCodesKey)
	}
	for _, code := range codes {
		if !slices.Contains(platformCodes, code) {
			r.report("top level", "%s holds %q, which is not a platform code", problemCodesKey, code)
		}
	}
}

// security: every operation declares its own security, [{bearer: []}] or []
// for a public one (M0-P3 handoff 3): the bundler drops a module file's
// top-level security, and doc.Validate accepts a reference to a scheme the
// bundle lacks.
func (r *ruleCheck) security(where string, op *openapi3.Operation) {
	if op.Security == nil {
		r.report(where, "declares no security; write [{bearer: []}], or [] for a public operation")
		return
	}
	for _, req := range *op.Security {
		for _, name := range slices.Sorted(maps.Keys(req)) {
			if r.doc.Components == nil || r.doc.Components.SecuritySchemes[name] == nil {
				r.report(where, "security scheme %q is not declared in components.securitySchemes", name)
			}
		}
	}
}

// problemCodes: every operation lists the codes it can answer beyond the
// top-level ones (M2 design 3.11). A module's code is prefixed with the
// module file that declares the path; a code without prefix is a platform
// code.
func (r *ruleCheck) problemCodes(where, owner string, op *openapi3.Operation) {
	codes, present, err := problemCodes(op.Extensions)
	switch {
	case err != nil:
		r.report(where, "%v", err)
	case !present:
		r.report(where, "has no %s; write [] when it answers only the top-level codes", problemCodesKey)
	}
	for _, code := range codes {
		prefix, _, prefixed := strings.Cut(code, ".")
		switch {
		case !codePattern.MatchString(code):
			r.report(where, "problem code %q is not spelled [module.]lower_snake", code)
		case prefixed && prefix != owner:
			r.report(where, "problem code %q is not prefixed with its module %q", code, owner)
		case !prefixed && !slices.Contains(platformCodes, code):
			r.report(where, "problem code %q has no module prefix and is not a platform code", code)
		}
	}
}

func (r *ruleCheck) operation(where, owner string, op *openapi3.Operation) {
	r.security(where, op)
	r.problemCodes(where, owner, op)
	if !r.lowerCamel.MatchString(op.OperationID) {
		r.report(where, "operationId %q is not lower camelCase", op.OperationID)
	}
	if len(op.Tags) == 0 {
		r.report(where, "has no tag")
	}
	for _, tag := range op.Tags {
		if r.doc.Tags.Get(tag) == nil {
			r.report(where, "tag %q is not declared in the top-level tags", tag)
		}
	}
	if !r.defaultIsProblem(op) {
		r.report(where, "has no default response whose application/problem+json schema is the Problem component")
	}
	for _, p := range op.Parameters {
		r.parameter(where+" parameters/"+p.Value.Name, p)
	}
	if body := op.RequestBody; body != nil && body.Ref == "" {
		r.content(where+" requestBody", body.Value.Content)
	}
	responses := op.Responses.Map()
	for _, status := range slices.Sorted(maps.Keys(responses)) {
		r.response(where+" responses/"+status, responses[status])
	}
}

// defaultIsProblem compares by schema identity: the bundler turns every
// reference into a local $ref, and the loader resolves each one to the
// component's own *Schema.
func (r *ruleCheck) defaultIsProblem(op *openapi3.Operation) bool {
	res := op.Responses.Default()
	if r.doc.Components == nil || res == nil || res.Value == nil {
		return false
	}
	problem := r.doc.Components.Schemas["Problem"]
	media := res.Value.Content["application/problem+json"]
	return problem != nil && media != nil && media.Schema != nil && media.Schema.Value == problem.Value
}

func (r *ruleCheck) components(c *openapi3.Components) {
	for _, name := range slices.Sorted(maps.Keys(c.Schemas)) {
		r.closedObject("components/schemas/"+name, c.Schemas[name])
		r.schema("components/schemas/"+name, c.Schemas[name])
	}
	for _, name := range slices.Sorted(maps.Keys(c.Parameters)) {
		r.parameter("components/parameters/"+name, c.Parameters[name])
	}
	for _, name := range slices.Sorted(maps.Keys(c.Headers)) {
		if h := c.Headers[name]; h.Ref == "" {
			r.schema("components/headers/"+name, h.Value.Schema)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(c.RequestBodies)) {
		if body := c.RequestBodies[name]; body.Ref == "" {
			r.content("components/requestBodies/"+name, body.Value.Content)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(c.Responses)) {
		r.response("components/responses/"+name, c.Responses[name])
	}
}

// closedObject lets contract tests catch undocumented fields: a component
// object schema must set additionalProperties: false. The rule targets
// response objects and is applied to request schemas as well. Exempt are
// composition wrappers (allOf, oneOf or anyOf without properties of their
// own): additionalProperties does not see the subschemas' properties, so
// closing a wrapper would reject every instance.
func (r *ruleCheck) closedObject(where string, ref *openapi3.SchemaRef) {
	s := ref.Value
	if ref.Ref != "" || !s.Type.Includes("object") {
		return
	}
	if len(s.Properties) == 0 && len(s.AllOf)+len(s.OneOf)+len(s.AnyOf) > 0 {
		return
	}
	if closed := s.AdditionalProperties.Has; closed == nil || *closed {
		r.report(where, "object schema does not set additionalProperties: false")
	}
}

func (r *ruleCheck) parameter(where string, p *openapi3.ParameterRef) {
	if p.Ref != "" {
		return
	}
	r.schema(where, p.Value.Schema)
	r.content(where, p.Value.Content)
}

func (r *ruleCheck) response(where string, res *openapi3.ResponseRef) {
	if res.Ref != "" {
		return
	}
	for _, name := range slices.Sorted(maps.Keys(res.Value.Headers)) {
		if h := res.Value.Headers[name]; h.Ref == "" {
			r.schema(where+" headers/"+name, h.Value.Schema)
		}
	}
	r.content(where, res.Value.Content)
}

func (r *ruleCheck) content(where string, content openapi3.Content) {
	for _, media := range slices.Sorted(maps.Keys(content)) {
		r.schema(where+" "+media, content[media].Schema)
	}
}

// schema checks the keywords that oapi-codegen mistranslates, in the schema
// and, recursively, its inline subschemas; a $ref is checked where its target
// is defined. kin-openapi decodes `nullable: false` and `const: null` to zero
// values that look like an absent keyword, so the keys the loader recorded
// per schema (Origin) are checked as well as the decoded fields. Walking the
// raw YAML instead would need a direct YAML dependency and would have to
// tell a schema's keywords from, say, a property named const.
func (r *ruleCheck) schema(where string, ref *openapi3.SchemaRef) {
	if ref == nil || ref.Ref != "" || ref.Value == nil {
		return
	}
	s := ref.Value
	if s.Nullable || spells(s, "nullable") {
		r.report(where, "uses nullable, the OpenAPI 3.0 keyword; write type: [T, 'null']")
	}
	if s.Const != nil || spells(s, "const") {
		r.report(where, "uses const, which oapi-codegen turns into interface{}; write a single-value enum")
	}
	if slices.Contains(s.Enum, nil) {
		r.report(where, `has null in enum, which adds a "<nil>" Go constant; write oneOf: [{$ref: …}, {type: 'null'}]`)
	}
	for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
		r.schema(where+"/properties/"+name, s.Properties[name])
	}
	r.schema(where+"/items", s.Items)
	r.schema(where+"/additionalProperties", s.AdditionalProperties.Schema)
	r.schema(where+"/not", s.Not)
	for _, list := range []struct {
		keyword string
		refs    openapi3.SchemaRefs
	}{{"allOf", s.AllOf}, {"oneOf", s.OneOf}, {"anyOf", s.AnyOf}} {
		for i, sub := range list.refs {
			r.schema(fmt.Sprintf("%s/%s/%d", where, list.keyword, i), sub)
		}
	}
}

// spells reports whether the source of s wrote keyword, whatever its value.
func spells(s *openapi3.Schema, keyword string) bool {
	if s.Origin == nil {
		return false
	}
	_, ok := s.Origin.Fields.Lookup(keyword)
	return ok
}
```

`server/internal/platform/httpserver/apitest/rules_cases_test.go`：

```go
package apitest

import (
	"slices"
	"strings"
	"testing"
)

// ruleCasesBase follows every authoring rule; each case of
// TestAuthoringRulesReportViolations breaks exactly one.
const ruleCasesBase = `
openapi: 3.1.0
info: {title: rule cases, version: v0}
x-problem-codes: [bad_request, internal_error]
tags:
  - name: things
paths:
  /api/v0/things:
    get:
      operationId: listThings
      tags: [things]
      security: [{bearer: []}]
      x-problem-codes: [not_found]
      parameters:
        - {name: kind, in: query, schema: {type: string, enum: [a, b]}}
      responses:
        '200':
          description: The things.
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Thing'}
        default:
          $ref: '#/components/responses/Problem'
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
  responses:
    Problem:
      description: Error.
      content:
        application/problem+json:
          schema: {$ref: '#/components/schemas/Problem'}
  schemas:
    Problem:
      type: object
      additionalProperties: false
      properties:
        code: {type: string}
    Thing:
      type: object
      additionalProperties: false
      properties:
        name: {type: string}
        note: {type: [string, 'null']}
        labels: {type: array, items: {type: string}}
    Target:
      oneOf:
        - $ref: '#/components/schemas/Thing'
        - {type: 'null'}
    Named:
      type: object
      allOf:
        - $ref: '#/components/schemas/Thing'
        - {required: [name]}
`

// ruleCasesOwners says which module file declares each path of the base.
var ruleCasesOwners = map[string]string{"/api/v0/things": "things"}

// A module's own prefixed code passes.
func TestModuleCodesPass(t *testing.T) {
	doc := parse(t, strings.Replace(ruleCasesBase, "x-problem-codes: [not_found]", "x-problem-codes: [things.taken, validation_failed]", 1))
	if got := authoringViolations(doc, ruleCasesOwners); len(got) != 0 {
		t.Errorf("violations = %q, want none", got)
	}
}

// TestAuthoringRulesReportViolations proves each check of
// authoringViolations on a hand-built document, since the real contract
// passes them all.
func TestAuthoringRulesReportViolations(t *testing.T) {
	if got := authoringViolations(parse(t, ruleCasesBase), ruleCasesOwners); len(got) != 0 {
		t.Fatalf("the base document breaks rules: %q", got)
	}
	const (
		noProblem = "GET /api/v0/things: has no default response whose application/problem+json schema is the Problem component"
		nullable  = ": uses nullable, the OpenAPI 3.0 keyword; write type: [T, 'null']"
		constant  = ": uses const, which oapi-codegen turns into interface{}; write a single-value enum"
		nullEnum  = `: has null in enum, which adds a "<nil>" Go constant; write oneOf: [{$ref: …}, {type: 'null'}]`
		open      = ": object schema does not set additionalProperties: false"
		thing     = "    Thing:\n      type: object\n      additionalProperties: false\n"
	)
	tests := []struct {
		name, old, new, want string
	}{
		{"path outside /api/v0/", "  /api/v0/things:", "  /things:", "/things: does not start with /api/v0/"},
		{"operationId not lower camelCase", "operationId: listThings", "operationId: ListThings",
			`GET /api/v0/things: operationId "ListThings" is not lower camelCase`},
		{"no operationId", "      operationId: listThings\n", "", `GET /api/v0/things: operationId "" is not lower camelCase`},
		{"no tag", "      tags: [things]\n", "", "GET /api/v0/things: has no tag"},
		{"undeclared tag", "tags: [things]", "tags: [stuff]", `GET /api/v0/things: tag "stuff" is not declared in the top-level tags`},
		{"no default response", "        default:\n          $ref: '#/components/responses/Problem'\n", "", noProblem},
		{"default with another schema", "schema: {$ref: '#/components/schemas/Problem'}", "schema: {$ref: '#/components/schemas/Thing'}", noProblem},
		{"default as application/json", "        application/problem+json:\n", "        application/json:\n", noProblem},
		{"nullable", "note: {type: [string, 'null']}", "note: {type: string, nullable: true}", "components/schemas/Thing/properties/note" + nullable},
		{"nullable: false", "note: {type: [string, 'null']}", "note: {type: string, nullable: false}", "components/schemas/Thing/properties/note" + nullable},
		{"const", "name: {type: string}", "name: {type: string, const: thing}", "components/schemas/Thing/properties/name" + constant},
		{"const: null", "name: {type: string}", "name: {const: null}", "components/schemas/Thing/properties/name" + constant},
		{"null in enum", "note: {type: [string, 'null']}", "note: {type: [string, 'null'], enum: [a, null]}", "components/schemas/Thing/properties/note" + nullEnum},
		{"in items", "items: {type: string}", "items: {type: string, nullable: true}", "components/schemas/Thing/properties/labels/items" + nullable},
		{"in oneOf", "- {type: 'null'}", "- {const: null}", "components/schemas/Target/oneOf/1" + constant},
		{"in allOf", "- {required: [name]}", "- {required: [name], nullable: true}", "components/schemas/Named/allOf/1" + nullable},
		{"in a parameter", "enum: [a, b]", "enum: [a, null]", "GET /api/v0/things parameters/kind" + nullEnum},
		{"in an inline response schema", "schema: {$ref: '#/components/schemas/Thing'}", "schema: {type: string, const: x}",
			"GET /api/v0/things responses/200 application/json" + constant},
		{"open object", thing, "    Thing:\n      type: object\n", "components/schemas/Thing" + open},
		{"additionalProperties: true", thing, "    Thing:\n      type: object\n      additionalProperties: true\n", "components/schemas/Thing" + open},
		{"composition with properties of its own", "    Named:\n      type: object\n",
			"    Named:\n      type: object\n      properties: {id: {type: string}}\n", "components/schemas/Named" + open},
		{"no top-level codes", "x-problem-codes: [bad_request, internal_error]\n", "", "top level: has no x-problem-codes"},
		{"module code at the top level", "[bad_request, internal_error]", "[bad_request, things.taken]",
			`top level: x-problem-codes holds "things.taken", which is not a platform code`},
		{"no security", "      security: [{bearer: []}]\n", "",
			"GET /api/v0/things: declares no security; write [{bearer: []}], or [] for a public operation"},
		{"undeclared scheme", "security: [{bearer: []}]", "security: [{apiKey: []}]",
			`GET /api/v0/things: security scheme "apiKey" is not declared in components.securitySchemes`},
		{"no operation codes", "      x-problem-codes: [not_found]\n", "",
			"GET /api/v0/things: has no x-problem-codes; write [] when it answers only the top-level codes"},
		{"codes not a list", "x-problem-codes: [not_found]", "x-problem-codes: not_found",
			"GET /api/v0/things: x-problem-codes is not a list"},
		{"code misspelled", "x-problem-codes: [not_found]", "x-problem-codes: [Things.Taken]",
			`GET /api/v0/things: problem code "Things.Taken" is not spelled [module.]lower_snake`},
		{"code of another module", "x-problem-codes: [not_found]", "x-problem-codes: [stuff.taken]",
			`GET /api/v0/things: problem code "stuff.taken" is not prefixed with its module "things"`},
		{"unprefixed code that is not the platform's", "x-problem-codes: [not_found]", "x-problem-codes: [taken]",
			`GET /api/v0/things: problem code "taken" has no module prefix and is not a platform code`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(ruleCasesBase, tt.old) {
				t.Fatalf("the base document has no %q", tt.old)
			}
			doc := parse(t, strings.Replace(ruleCasesBase, tt.old, tt.new, 1))
			if got := authoringViolations(doc, ruleCasesOwners); !slices.Equal(got, []string{tt.want}) {
				t.Errorf("violations = %q, want %q", got, tt.want)
			}
		})
	}
}
```

照下面的差异修改 `server/internal/platform/httpserver/apitest/apitest_test.go`（在 Task 4 的版本上）：

```diff
--- a/server/internal/platform/httpserver/apitest/apitest_test.go
+++ b/server/internal/platform/httpserver/apitest/apitest_test.go
@@ -20,7 +20,8 @@
 
 // Component names become Go and TypeScript type names. Redocly's bundler
 // renames a clash between module files to "Name-2" and only warns, so a
-// clash has to fail here instead.
+// clash has to fail here instead. Security schemes are the exception: they
+// name no type, and every module declares the same bearer (M2 design 3.12).
 func TestComponentNamesAreTypeNames(t *testing.T) {
 	c := Load(t)
 	valid := regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
@@ -29,7 +30,11 @@
 		t.Fatal("the contract has no components")
 	}
 	for _, name := range names {
-		if _, base, _ := strings.Cut(name, "/"); !valid.MatchString(base) {
+		kind, base, _ := strings.Cut(name, "/")
+		if kind == "securitySchemes" {
+			continue
+		}
+		if !valid.MatchString(base) {
 			t.Errorf("component %s is not a PascalCase type name; two module files may define it differently", name)
 		}
 	}
```

- [ ] **Step 4: instance 的 `TestMain`**

`server/internal/modules/instance/adapter/http/main_test.go`：

```go
package httpadapter_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11).
func TestMain(m *testing.M) { apitest.Main(m, "instance") }
```

- [ ] **Step 5: 重新生成**

Run: `make gen`
Expected: `api/dist/openapi.yaml` 多出顶层 `x-problem-codes`、`securitySchemes.bearer`，`getInstance` 多出 `security: []` 和 `x-problem-codes: []`；`server/internal/modules/instance/adapter/http/gen/server.gen.go` 不变（SHA-256 `106c4214c9f5aa5c7813f50444a351dac1edb03eedee3e29407c1da192cfb3de`，321 行，与 M0 相同，证明新模板不改变已有生成代码）；`bodyshape.gen.go`、`components.gen.go`、`schema.gen.ts` 不变。

- [ ] **Step 6: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`internal/platform/httpserver/apitest` 的 `TestContractFollowsAuthoringRules` 用真实契约通过。

- [ ] **Step 7: 提交**

```bash
git add api server/internal/platform/httpserver/apitest server/internal/modules/instance
```
```bash
git commit -m "feat(M2/P1): every operation declares security and x-problem-codes; apitest checks both

apitest.Main fails a module's tests when a declared code is never answered,
and CheckRequest checks the requests tests send. The module template of
oapi-codegen maps uuid to the standard library and email to string.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** `TestContractFollowsAuthoringRules` 和 `TestAuthoringRulesReportViolations` 的 9 个新反例通过；instance 的 `server.gen.go` 逐字节不变；`make gen-check` 干净。

---

### Task 6: 三张表的迁移、sqlc 与 `TestSQLCSchemaScope`；S1 的迁移断言

**Files:**
- Create: `server/migrations/sql/00001_identity_users.sql`、`00002_identity_profiles.sql`、`00003_identity_auth_sessions.sql`
- Delete: `server/migrations/sql/.gitkeep`
- Modify: `server/migrations/embed.go`、`embed_test.go`；Create: `server/migrations/schema_test.go`
- Modify: `server/internal/platform/postgres/migrator_test.go`（注释）、`server/internal/bootstrap/app_test.go`（过渡：注释；Task 11 写最终版本）
- Create: `server/sqlc.yaml`、`server/internal/modules/identity/adapter/postgres/queries/users.sql`、`profiles.sql`、`sessions.sql`
- Create: `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go`
- Modify: `Makefile`（最终版本）、`server/tools/go.mod`、`go.sum`
- Modify: `e2e/tsconfig.json`、`e2e/stories/smoke/s1-server-ready.spec.ts`
- Generate: `server/internal/modules/identity/adapter/postgres/gen/db.go`、`models.go`、`profiles.sql.go`、`sessions.sql.go`、`users.sql.go`

**Interfaces:**
- Consumes: M0 的 goose 迁移器和 `pgtest`；Task 1 的 `Makefile`。
- Produces:
  - 表 `users`、`profiles`、`auth_sessions`（列、约束名、CHECK 见 spec 2.10 和 M2 设计 4.2、4.3、4.5）；
  - `gen.New(db gen.DBTX) *gen.Queries`，方法 `CreateUser`、`GetUser`、`CreateProfile`、`CreateSession`、`GetSessionCredential`（Task 9 的仓储用它）；
  - `make gen-go` 最后执行 sqlc；`GEN_GO_OUT` 包括 `adapter/postgres/gen`。

**Tests:**
- `schema_test.go`（testcontainers）：`TestMigrationsGoUpDownAndUpAgain`；`TestConstraintAndIndexNames`（19 个名字，外键的 `ON DELETE CASCADE`）；`TestChecksRejectCounterexamples`（25 个反例，每个核对 `23514` 和约束名；正例含 `élodie@exämple.com` 和两次 `||` 合并）。
- `embed_test.go`：至少一个迁移文件，文件名合约定。
- `sqlc_test.go`：`TestSQLCSchemaScope`（真实的 `server/sqlc.yaml` 和迁移目录）。
- `sqlc_cases_test.go`：`TestSQLCScopeOfTheBaseLayoutPasses`、`TestSQLCScopeReportsViolations`（12 个人造布局，每个核对报错原文）。
- S1：`nerve migrate status` 的每一行都是 `applied`、来源依次等于迁移文件；`goose_db_version` 的最大版本等于最后一个文件的序号。

- [ ] **Step 1: 迁移文件**

`server/migrations/sql/00001_identity_users.sql`：

```sql
-- users：Plane 的 users 表（40 列）按 M2 设计 4.2 保留 10 列。约定见 M2 设计 3.13：
-- id 由应用生成（uuid.NewV7），created_at、updated_at 由应用按用例的时钟写入，DEFAULT now() 只是兜底。

-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY,
    -- 领域层先规范化（小写、去掉首尾空白）再校验；CHECK 挡住绕过应用写入的大写和空白（4.2）
    email varchar(255) NOT NULL UNIQUE CHECK (email = lower(email) AND email !~ '[[:space:]]'),
    -- argon2id 的 PHC 字符串（3.8）
    password varchar(128) NOT NULL,
    first_name varchar(255) NOT NULL DEFAULT '',
    last_name varchar(255) NOT NULL DEFAULT '',
    display_name varchar(255) NOT NULL CHECK (display_name <> ''),
    user_timezone varchar(255) NOT NULL DEFAULT 'UTC',
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
```

`server/migrations/sql/00002_identity_profiles.sql`：

```sql
-- profiles：Plane 的 profiles 表（29 列）按 M2 设计 4.3 保留 11 列，默认值和取值范围取自 Plane 的模型。

-- +goose Up
CREATE TABLE profiles (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL UNIQUE REFERENCES users ON DELETE CASCADE,
    theme varchar(20) NOT NULL DEFAULT 'system'
        CHECK (theme IN ('system', 'light', 'dark', 'light-contrast', 'dark-contrast')),
    is_tour_completed boolean NOT NULL DEFAULT false,
    -- 恰好这四个键、值都是布尔值；外层的 CASE 让数组、标量和 JSON null 也得到 check_violation（3.13）
    onboarding_step jsonb NOT NULL
        DEFAULT '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'
        CHECK (CASE WHEN jsonb_typeof(onboarding_step) = 'object' THEN
            onboarding_step ?& array['profile_complete', 'workspace_create', 'workspace_invite', 'workspace_join']
            AND (onboarding_step - array['profile_complete', 'workspace_create', 'workspace_invite', 'workspace_join']) = '{}'::jsonb
            AND jsonb_typeof(onboarding_step -> 'profile_complete') = 'boolean'
            AND jsonb_typeof(onboarding_step -> 'workspace_create') = 'boolean'
            AND jsonb_typeof(onboarding_step -> 'workspace_invite') = 'boolean'
            AND jsonb_typeof(onboarding_step -> 'workspace_join') = 'boolean'
        ELSE false END),
    is_onboarded boolean NOT NULL DEFAULT false,
    -- 客户端写入的工作区 id，不是外键（和 Plane 一样，3.2）
    last_workspace_id uuid,
    language varchar(255) NOT NULL DEFAULT 'en' CHECK (language IN ('en', 'zh-CN')),
    start_of_the_week smallint NOT NULL DEFAULT 0 CHECK (start_of_the_week BETWEEN 0 AND 6),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE profiles;
```

`server/migrations/sql/00003_identity_auth_sessions.sql`：

```sql
-- auth_sessions：一次登录一行，替代 Plane 的 Django 会话表 sessions（M2 设计 3.5、4.5）。
-- 旧代的刷新令牌不存：它们由令牌里的 MAC 标签认出（3.4）。

-- +goose Up
CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 当前这一代刷新令牌密文的 SHA-256
    token_hash bytea NOT NULL CHECK (octet_length(token_hash) = 32),
    generation integer NOT NULL DEFAULT 0 CHECK (generation >= 0),
    user_agent text NOT NULL DEFAULT '',
    ip inet,
    -- 登录时刻加 auth.session_ttl，之后不变
    expires_at timestamptz NOT NULL,
    last_refreshed_at timestamptz,
    revoked_at timestamptz,
    revoke_reason varchar(20) CHECK (revoke_reason IN
        ('logout', 'password_changed', 'password_reset', 'email_changed', 'deactivated', 'reuse_detected')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_sessions_revoked_consistent_check CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))
);
-- 撤销某个账户的全部会话
CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id);
-- 清理过期会话的定时任务（M2/P3）
CREATE INDEX auth_sessions_expires_at_idx ON auth_sessions (expires_at);

-- +goose Down
DROP TABLE auth_sessions;
```

删除占位文件（有了迁移文件，`//go:embed all:sql` 保留它的理由不在了）：

```bash
git rm server/migrations/sql/.gitkeep
```

`server/migrations/embed.go`：

```go
// Package migrations embeds the SQL schema migrations. Files live in sql/ and
// are named NNNNN_<module>_<description>.sql; a migration belongs to the
// module that owns the table it changes (M2 design 3.14).
package migrations

import (
	"embed"
	"io/fs"
)

//go:embed sql/*.sql
var files embed.FS

// FS returns the migrations, with the files at its root.
func FS() fs.FS {
	sub, err := fs.Sub(files, "sql")
	if err != nil {
		panic(err) // unreachable: "sql" is a valid path embedded above
	}
	return sub
}
```

`server/migrations/embed_test.go`：

```go
package migrations

import (
	"io/fs"
	"regexp"
	"testing"
)

var migrationName = regexp.MustCompile(`^\d{5}_[a-z][a-z0-9]*_[a-z0-9_]+\.sql$`)

func TestMigrationFilesFollowNamingConvention(t *testing.T) {
	entries, err := fs.ReadDir(FS(), ".")
	if err != nil || len(entries) == 0 {
		t.Fatalf("read migrations = %d entries, %v; want some", len(entries), err)
	}
	for _, e := range entries {
		if e.IsDir() || !migrationName.MatchString(e.Name()) {
			t.Errorf("%s: want NNNNN_<module>_<description>.sql, e.g. 00001_identity_users.sql", e.Name())
		}
	}
}
```

`server/migrations/schema_test.go`：

```go
package migrations_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func newPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func tables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'goose_db_version' ORDER BY 1")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

// Every migration can go up, down and up again (M2 design 4.1).
func TestMigrationsGoUpDownAndUpAgain(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	m, err := postgres.NewMigrator(pool, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })

	up, err := m.Up(ctx)
	if err != nil || len(up) != 3 {
		t.Fatalf("Up() = %d migrations, %v; want 3", len(up), err)
	}
	if got, want := tables(t, pool), []string{"auth_sessions", "profiles", "users"}; !slices.Equal(got, want) {
		t.Errorf("tables after Up = %q, want %q", got, want)
	}
	for range up {
		if _, err := m.Down(ctx); err != nil {
			t.Fatalf("Down() error = %v", err)
		}
	}
	if got := tables(t, pool); len(got) != 0 {
		t.Errorf("tables after every Down = %q, want none", got)
	}
	if again, err := m.Up(ctx); err != nil || len(again) != 3 {
		t.Errorf("Up() again = %d migrations, %v; want 3", len(again), err)
	}
}

// Constraint and index names are the stable names of M2 design 3.13: errors
// are mapped by them, and later migrations drop them by name.
func TestConstraintAndIndexNames(t *testing.T) {
	pool := newPool(t, pgtest.NewDatabase(t))
	rows, err := pool.Query(context.Background(), `
		SELECT conname || ' ' || contype::text || CASE WHEN contype = 'f' THEN ' ' || confdeltype::text ELSE '' END FROM pg_constraint
		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass)
			AND contype <> 'n' -- PG 18 lists NOT NULL as constraints too
		UNION ALL
		SELECT indexname || ' i' FROM pg_indexes WHERE tablename IN ('users', 'profiles', 'auth_sessions')
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	// contype: p primary key, u unique, f foreign key (confdeltype c: ON DELETE CASCADE), c check.
	want := []string{
		"auth_sessions_expires_at_idx i",
		"auth_sessions_generation_check c",
		"auth_sessions_pkey i",
		"auth_sessions_pkey p",
		"auth_sessions_revoke_reason_check c",
		"auth_sessions_revoked_consistent_check c",
		"auth_sessions_token_hash_check c",
		"auth_sessions_user_id_fkey f c",
		"auth_sessions_user_id_idx i",
		"profiles_language_check c",
		"profiles_onboarding_step_check c",
		"profiles_pkey i",
		"profiles_pkey p",
		"profiles_start_of_the_week_check c",
		"profiles_theme_check c",
		"profiles_user_id_fkey f c",
		"profiles_user_id_key i",
		"profiles_user_id_key u",
		"users_display_name_check c",
		"users_email_check c",
		"users_email_key i",
		"users_email_key u",
		"users_pkey i",
		"users_pkey p",
	}
	if !slices.Equal(got, want) {
		t.Errorf("constraints and indexes =\n%q\nwant\n%q", got, want)
	}
}

// The CHECKs accept what the domain writes and reject what bypasses it
// (M2 design 4.2, 4.3, 4.5).
func TestChecksRejectCounterexamples(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const user = "'0199a2b4-0000-7000-8000-000000000001'"
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + user + ", 'élodie@exämple.com', 'x', 'élodie')",
		"INSERT INTO profiles (id, user_id) VALUES ('0199a2b4-0000-7000-8000-000000000002', " + user + ")",
		"INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES ('0199a2b4-0000-7000-8000-000000000003', " + user + ", sha256('x'), now())",
		`UPDATE profiles SET onboarding_step = onboarding_step || '{"profile_complete": true}'`,
		"UPDATE profiles SET onboarding_step = onboarding_step || '{}'",
		"UPDATE auth_sessions SET revoked_at = now(), revoke_reason = 'logout'",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	insertUser := func(email, displayName string) string {
		return "INSERT INTO users (id, email, password, display_name) VALUES (gen_random_uuid(), " + email + ", 'x', " + displayName + ")"
	}
	tests := []struct{ name, stmt, constraint string }{
		{"upper-case ASCII e-mail", insertUser("'Bob@corp.com'", "'b'"), "users_email_check"},
		{"upper-case non-ASCII e-mail", insertUser("'Élodie@corp.com'", "'e'"), "users_email_check"},
		{"leading space", insertUser("' carol@corp.com'", "'c'"), "users_email_check"},
		{"trailing tab", insertUser(`E'carol@corp.com\t'`, "'c'"), "users_email_check"},
		{"trailing newline", insertUser(`E'carol@corp.com\n'`, "'c'"), "users_email_check"},
		{"inner space", insertUser("'car ol@corp.com'", "'c'"), "users_email_check"},
		{"leading U+3000", insertUser(`U&'\3000carol@corp.com'`, "'c'"), "users_email_check"},
		{"trailing U+2028", insertUser(`U&'carol@corp.com\2028'`, "'c'"), "users_email_check"},
		{"empty display name", insertUser("'dave@corp.com'", "''"), "users_display_name_check"},
		{"onboarding_step an array", `UPDATE profiles SET onboarding_step = '["profile_complete"]'`, "profiles_onboarding_step_check"},
		{"onboarding_step a scalar", "UPDATE profiles SET onboarding_step = '5'", "profiles_onboarding_step_check"},
		{"onboarding_step JSON null", "UPDATE profiles SET onboarding_step = 'null'", "profiles_onboarding_step_check"},
		{"onboarding_step missing a key", `UPDATE profiles SET onboarding_step = onboarding_step - 'workspace_join'`, "profiles_onboarding_step_check"},
		{"onboarding_step extra key", `UPDATE profiles SET onboarding_step = onboarding_step || '{"extra": true}'`, "profiles_onboarding_step_check"},
		{"onboarding_step string value", `UPDATE profiles SET onboarding_step = onboarding_step || '{"workspace_join": "yes"}'`, "profiles_onboarding_step_check"},
		{"onboarding_step null value", `UPDATE profiles SET onboarding_step = onboarding_step || '{"workspace_join": null}'`, "profiles_onboarding_step_check"},
		{"onboarding_step number value", `UPDATE profiles SET onboarding_step = onboarding_step || '{"workspace_join": 1}'`, "profiles_onboarding_step_check"},
		{"unknown theme", "UPDATE profiles SET theme = 'custom'", "profiles_theme_check"},
		{"unknown language", "UPDATE profiles SET language = 'fr'", "profiles_language_check"},
		{"start_of_the_week 7", "UPDATE profiles SET start_of_the_week = 7", "profiles_start_of_the_week_check"},
		{"token hash not 32 bytes", "UPDATE auth_sessions SET token_hash = '\\x00'", "auth_sessions_token_hash_check"},
		{"negative generation", "UPDATE auth_sessions SET generation = -1", "auth_sessions_generation_check"},
		{"unknown revoke reason", "UPDATE auth_sessions SET revoke_reason = 'expired'", "auth_sessions_revoke_reason_check"},
		{"revoked without a reason", "UPDATE auth_sessions SET revoke_reason = NULL", "auth_sessions_revoked_consistent_check"},
		{"a reason without revoked_at", "UPDATE auth_sessions SET revoked_at = NULL", "auth_sessions_revoked_consistent_check"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.stmt)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
				t.Errorf("%s = %v, want check_violation (23514) of %s", tt.stmt, err, tt.constraint)
			}
		})
	}
}
```

两个用样例迁移的测试只改注释。`server/internal/platform/postgres/migrator_test.go`：

```go
package postgres_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// sampleMigrations is a test-only migration set, independent of the
// production schema (whose own up/down test is in server/migrations).
var sampleMigrations = fstest.MapFS{
	"00001_probe_create_widgets.sql": {Data: []byte(`-- +goose Up
CREATE TABLE widgets (id bigint PRIMARY KEY);

-- +goose Down
DROP TABLE widgets;
`)},
	"00002_probe_add_color.sql": {Data: []byte(`-- +goose Up
ALTER TABLE widgets ADD COLUMN color text;

-- +goose Down
ALTER TABLE widgets DROP COLUMN color;
`)},
}

func newPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newMigrator(t *testing.T, pool *pgxpool.Pool, fsys fstest.MapFS) *postgres.Migrator {
	t.Helper()
	m, err := postgres.NewMigrator(pool, fsys)
	if err != nil {
		t.Fatalf("NewMigrator() error = %v", err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return m
}

func hasColumn(t *testing.T, pool *pgxpool.Pool, table, column string) bool {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2",
		table, column).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func applied(t *testing.T, m *postgres.Migrator) []bool {
	t.Helper()
	statuses, err := m.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	out := make([]bool, len(statuses))
	for i, s := range statuses {
		out[i] = s.Applied
		if s.Applied == s.AppliedAt.IsZero() {
			t.Errorf("status %+v: AppliedAt must be set exactly when applied", s)
		}
	}
	return out
}

func TestMigratorUpStatusDown(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	m := newMigrator(t, pool, sampleMigrations)

	if got := applied(t, m); len(got) != 2 || got[0] || got[1] {
		t.Fatalf("applied before Up = %v, want [false false]", got)
	}
	if err := m.CheckUpToDate(ctx); !errors.Is(err, postgres.ErrPendingMigrations) {
		t.Errorf("CheckUpToDate() before Up = %v, want ErrPendingMigrations", err)
	}

	ran, err := m.Up(ctx)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	want := []postgres.Migration{
		{Version: 1, Source: "00001_probe_create_widgets.sql"},
		{Version: 2, Source: "00002_probe_add_color.sql"},
	}
	if len(ran) != 2 || ran[0] != want[0] || ran[1] != want[1] {
		t.Errorf("Up() = %+v, want %+v", ran, want)
	}
	if !hasColumn(t, pool, "widgets", "color") {
		t.Error("widgets.color missing after Up")
	}
	if got := applied(t, m); !got[0] || !got[1] {
		t.Errorf("applied after Up = %v, want [true true]", got)
	}
	if err := m.CheckUpToDate(ctx); err != nil {
		t.Errorf("CheckUpToDate() after Up = %v, want nil", err)
	}
	if again, err := m.Up(ctx); err != nil || len(again) != 0 {
		t.Errorf("second Up() = %+v, %v; want nothing to do", again, err)
	}

	back, err := m.Down(ctx)
	if err != nil || back == nil || *back != want[1] {
		t.Fatalf("Down() = %+v, %v; want %+v", back, err, want[1])
	}
	if hasColumn(t, pool, "widgets", "color") {
		t.Error("widgets.color still present after Down")
	}
	if got := applied(t, m); !got[0] || got[1] {
		t.Errorf("applied after Down = %v, want [true false]", got)
	}

	if back, err := m.Down(ctx); err != nil || back == nil || *back != want[0] {
		t.Fatalf("second Down() = %+v, %v; want %+v", back, err, want[0])
	}
	if back, err := m.Down(ctx); err != nil || back != nil {
		t.Errorf("Down() with nothing applied = %+v, %v; want nil, nil", back, err)
	}
}

func TestMigratorReportsFailingMigration(t *testing.T) {
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	m := newMigrator(t, pool, fstest.MapFS{
		"00001_probe_create_widgets.sql": sampleMigrations["00001_probe_create_widgets.sql"],
		"00002_probe_broken.sql":         {Data: []byte("-- +goose Up\nCREATE TABLE broken (;\n")},
	})

	ran, err := m.Up(context.Background())
	if err == nil {
		t.Error("Up() error = nil, want the SQL error")
	}
	want := postgres.Migration{Version: 1, Source: "00001_probe_create_widgets.sql"}
	if len(ran) != 1 || ran[0] != want {
		t.Errorf("Up() applied %+v, want only %+v, the migration before the failure", ran, want)
	}
	if !hasColumn(t, pool, "widgets", "id") {
		t.Error("widgets table missing: the migration before the failure must stay applied")
	}
}

func TestMigratorWithoutMigrationsIsNoOp(t *testing.T) {
	ctx := context.Background()
	// Nothing may touch the database: this pool points nowhere.
	pool := newPool(t, "postgres://nobody@127.0.0.1:1/nowhere")
	m := newMigrator(t, pool, fstest.MapFS{".gitkeep": {}})

	if ran, err := m.Up(ctx); err != nil || len(ran) != 0 {
		t.Errorf("Up() = %v, %v; want nothing", ran, err)
	}
	if back, err := m.Down(ctx); err != nil || back != nil {
		t.Errorf("Down() = %v, %v; want nothing", back, err)
	}
	if statuses, err := m.Status(ctx); err != nil || len(statuses) != 0 {
		t.Errorf("Status() = %v, %v; want nothing", statuses, err)
	}
	if err := m.CheckUpToDate(ctx); err != nil {
		t.Errorf("CheckUpToDate() = %v, want nil", err)
	}
}

func TestCheckUpToDateFailsWhenDatabaseIsUnreachable(t *testing.T) {
	pool := newPool(t, "postgres://nobody@127.0.0.1:1/nowhere")
	m := newMigrator(t, pool, sampleMigrations)

	err := m.CheckUpToDate(context.Background())
	if err == nil || errors.Is(err, postgres.ErrPendingMigrations) {
		t.Errorf("CheckUpToDate() = %v, want a connection error", err)
	}
}
```

`server/internal/bootstrap/app_test.go`（过渡）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -18,7 +18,8 @@
 	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
 )
 
-// sampleMigrations stands in for the production set, which is empty in M0.
+// sampleMigrations are two probe tables, for the tests of how the app applies
+// and reports migrations, whatever the production set holds.
 var sampleMigrations = fstest.MapFS{
 	"00001_probe_create_widgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE widgets (id bigint);\n-- +goose Down\nDROP TABLE widgets;\n")},
 	"00002_probe_create_gadgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE gadgets (id bigint);\n-- +goose Down\nDROP TABLE gadgets;\n")},
```

Run: `go -C server test -count=1 ./migrations/ ./internal/platform/postgres/ ./internal/bootstrap/`
Expected: 三个包 `ok`。

- [ ] **Step 2: sqlc**

Run: `go -C server/tools get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1`

Run: `go -C server/tools mod tidy`

Run: `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`
Expected: 两个文件都是 `go 1.27` 和 `toolchain go1.27.1`。

Expected: `server/tools/go.mod` 的 `tool` 块多出 `github.com/sqlc-dev/sqlc/cmd/sqlc`，间接依赖只增加、不改已有的版本（oapi-codegen 的依赖不变）；`go.mod` 的 SHA-256 为 `6192316fb2aafe42156283a889d901c7be883e22720a2fa320a0eefe2165c973`，`go.sum` 为 `4820bb75ea39beee4cb85f846a892fb5d70f42cb391478e19bba249e41cc319d`（只执行 `go get -tool` 而不 `go mod tidy` 时，`go.sum` 与这里不同）。

`server/sqlc.yaml`：

```yaml
# sqlc 配置（M2 设计 3.14）：每个模块一个条目，由 make gen-go 在 server/ 下执行
#   CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc generate
# schema 只列本模块的迁移：查询碰到别的模块的表，生成时就报错。archtest 的 TestSQLCSchemaScope 核对这份配置。
# 类型覆盖：uuid 用标准库的 uuid.UUID，timestamptz 用 time.Time（db_type 写 timestamptz，写 pg_catalog.timestamptz 不匹配）。
version: "2"
sql:
  - engine: postgresql
    schema:
      - migrations/sql/00001_identity_users.sql
      - migrations/sql/00002_identity_profiles.sql
      - migrations/sql/00003_identity_auth_sessions.sql
    queries: internal/modules/identity/adapter/postgres/queries
    gen:
      go:
        package: gen
        out: internal/modules/identity/adapter/postgres/gen
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        overrides:
          - db_type: uuid
            go_type: {import: uuid, type: UUID}
          - db_type: uuid
            nullable: true
            go_type: {import: uuid, type: UUID, pointer: true}
          - db_type: timestamptz
            go_type: {import: time, type: Time}
          - db_type: timestamptz
            nullable: true
            go_type: {import: time, type: Time, pointer: true}
```

`server/internal/modules/identity/adapter/postgres/queries/users.sql`：

```sql
-- name: CreateUser :exec
-- The other columns take their defaults; the audit columns come from the use case's clock (M2 design 3.13).
INSERT INTO users (id, email, password, display_name, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password), sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now));

-- name: GetUser :one
SELECT id, email, first_name, last_name, display_name, user_timezone, created_at
FROM users
WHERE id = sqlc.arg(id);
```

`server/internal/modules/identity/adapter/postgres/queries/profiles.sql`：

```sql
-- name: CreateProfile :exec
-- Every preference takes its default (Plane's model defaults, M2 design 4.3).
INSERT INTO profiles (id, user_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));
```

`server/internal/modules/identity/adapter/postgres/queries/sessions.sql`：

```sql
-- name: CreateSession :exec
-- A new login: generation 0 (M2 design 3.5).
INSERT INTO auth_sessions (id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), 0, sqlc.arg(user_agent), sqlc.arg(ip),
        sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now));

-- name: GetSessionCredential :one
-- What authentication checks on every request, by primary key (M2 design 3.5);
-- the use case judges it against its clock.
SELECT s.user_id, s.expires_at, s.revoked_at, u.is_active AS user_active
FROM auth_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.id = sqlc.arg(id);
```

照下面的差异修改 `Makefile`（在 Task 1 的版本上；这是它的最终版本）：

```diff
--- a/Makefile
+++ b/Makefile
@@ -15,10 +15,12 @@
 OAPI_CODEGEN := go tool -modfile=tools/go.mod oapi-codegen
 # 请求体结构表的生成器在工具模块里（M2 设计 3.11）；go -C 切到 server/tools，所以参数都用绝对路径
 BODYSHAPEGEN := go -C server/tools run ./bodyshapegen
+# sqlc 一律不用 cgo 运行：它改用编译成 wasm 的 libpg_query，不需要 C 编译器（M2 设计 3.14，M0-P1 交接 1）
+SQLC := CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc
 REDOCLY := REDOCLY_SUPPRESS_UPDATE_NOTICE=true pnpm exec redocly
 # 每个模块一个描述文件 api/modules/<模块>.yaml，生成到该模块的 adapter/http/gen
 API_MODULES := $(basename $(notdir $(wildcard api/modules/*.yaml)))
-GEN_GO_OUT := server/internal/platform/httpserver/apigen server/internal/modules/*/adapter/http/gen
+GEN_GO_OUT := server/internal/platform/httpserver/apigen server/internal/modules/*/adapter/http/gen server/internal/modules/*/adapter/postgres/gen
 GEN_WEB_OUT := api/dist web/packages/api-client/src/schema.gen.ts
 # 生成物必须已提交且没有差异；$(1) 是生成物的路径
 check-committed = test -z "$$(git status --porcelain -- $(1))" || { git status --short -- $(1); git --no-pager diff -- $(1); echo "生成物与接口描述不一致：执行 make gen，并提交生成的文件"; exit 1; }
@@ -72,8 +74,8 @@
 gen: gen-go gen-web ## 重新生成全部代码：Go 接口层、api/dist、TS 客户端
 
 .PHONY: gen-go
-gen-go: ## 由 api/ 生成 Go 接口层和请求体结构表（只需要 Go）
-	rm -f server/internal/platform/httpserver/apigen/*.gen.go server/internal/modules/*/adapter/http/gen/*.gen.go
+gen-go: ## 由 api/ 生成 Go 接口层和请求体结构表，由 server/sqlc.yaml 生成查询代码（只需要 Go）
+	rm -f server/internal/platform/httpserver/apigen/*.gen.go server/internal/modules/*/adapter/http/gen/*.gen.go server/internal/modules/*/adapter/postgres/gen/*.go
 	cd server && $(OAPI_CODEGEN) -config internal/platform/httpserver/apigen/oapi-codegen.yaml ../api/common.yaml
 	@set -e; for m in $(API_MODULES); do \
 		gen=$(CURDIR)/server/internal/modules/$$m/adapter/http/gen; \
@@ -82,6 +84,7 @@
 		echo "$(BODYSHAPEGEN) -config $$gen/oapi-codegen.yaml -out $$gen/bodyshape.gen.go $(CURDIR)/api/modules/$$m.yaml"; \
 		$(BODYSHAPEGEN) -config $$gen/oapi-codegen.yaml -out $$gen/bodyshape.gen.go $(CURDIR)/api/modules/$$m.yaml; \
 	done
+	cd server && $(SQLC) generate
 
 .PHONY: gen-web
 gen-web: ## 打包 api/dist/openapi.yaml，生成 TS 客户端的类型（需要 Node）
```

Run: `make -n gen-go`
Expected: 最后一行是 `cd server && CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc generate`，没有 `missing separator`。

Run: `make gen-go`
Expected: 生成 `server/internal/modules/identity/adapter/postgres/gen/` 下五个文件，SHA-256 和行数：

| 文件 | 行数 | SHA-256 |
|---|---|---|
| `db.go` | 32 | `1dfb2b6312c3c3db25a8cad84c031995c3b0546c73b3585a0282c98e11397676` |
| `models.go` | 54 | `0a1a7f0b3fba26cb5fb063c67ae7376912ce718b1448ffa3225f7a0c3604f55d` |
| `profiles.sql.go` | 30 | `9d350e7d040396ee6c7c422e5f0179fc1150099a1761125be706ba170d9191dd` |
| `sessions.sql.go` | 72 | `e988878081daaa0d2d43cd84e5c5205a452999e7fb230f7bc929b2a7250fb6bc` |
| `users.sql.go` | 69 | `7480141b2a0ae19767c3e6ffaa01057246269d457ea7b0c9417fb0bf9a94913c` |

`instance` 的 `server.gen.go`、`bodyshape.gen.go` 不变。sqlc 不需要 C 编译器（`CGO_ENABLED=0` 时用 wasm 的 libpg_query；原型核对过 cgo 与非 cgo 的输出逐字节相同）。

- [ ] **Step 3: `TestSQLCSchemaScope`**

`server/internal/archtest/sqlc_test.go`：

```go
package archtest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
)

// TestSQLCSchemaScope holds server/sqlc.yaml to M2 design 3.14: a module's
// queries see only its own migrations, so a query can touch only its own
// tables; sqlc itself then rejects the rest.
func TestSQLCSchemaScope(t *testing.T) {
	registerSources(t)
	entries := readSQLCEntries(t, filepath.Join(moduleRoot, "sqlc.yaml"))
	migrations := readMigrations(t, filepath.Join(moduleRoot, "migrations", "sql"))
	queryModules, err := filepath.Glob(filepath.Join(moduleRoot, "internal", "modules", "*", "adapter", "postgres", "queries"))
	if err != nil {
		t.Fatal(err)
	}
	var modules []string
	for _, dir := range queryModules {
		modules = append(modules, filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(dir)))))
	}
	if len(entries) == 0 || len(migrations) == 0 || len(modules) == 0 {
		t.Fatalf("read %d sqlc entries, %d migrations, %d modules with queries; want some of each", len(entries), len(migrations), len(modules))
	}
	for _, v := range sqlcScopeViolations(entries, migrations, modules) {
		t.Error(v)
	}
}

// sqlcEntry is one sql entry of sqlc.yaml, paths relative to server/.
type sqlcEntry struct {
	schema  []string
	queries string
	out     string
}

func readSQLCEntries(t *testing.T, path string) []sqlcEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := yaml.Parser().Unmarshal(data)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	raw, _ := doc["sql"].([]any)
	var entries []sqlcEntry
	for i, item := range raw {
		m, _ := item.(map[string]any)
		gen, _ := m["gen"].(map[string]any)
		goGen, _ := gen["go"].(map[string]any)
		e := sqlcEntry{queries: fmt.Sprint(m["queries"]), out: fmt.Sprint(goGen["out"])}
		schema, ok := m["schema"].([]any)
		if !ok {
			t.Fatalf("%s: sql[%d].schema is not a list of migration files", path, i)
		}
		for _, s := range schema {
			e.schema = append(e.schema, fmt.Sprint(s))
		}
		entries = append(entries, e)
	}
	return entries
}

// migrationFile is one migration: its file name and SQL.
type migrationFile struct {
	name string
	sql  string
}

func readMigrations(t *testing.T, dir string) []migrationFile {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var out []migrationFile
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, migrationFile{name: filepath.Base(f), sql: string(data)})
	}
	return out
}

var (
	migrationFileName = regexp.MustCompile(`^\d{5}_([a-z][a-z0-9]*)_[a-z0-9_]+\.sql$`)
	sqlComment        = regexp.MustCompile(`--[^\n]*`)
	createTable       = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
	alterTable        = regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
	moduleQueries     = regexp.MustCompile(`^internal/modules/([a-z][a-z0-9]*)/adapter/postgres/queries$`)
)

// sqlcScopeViolations checks, in M2 design 3.14's words:
//   - migration files are named <version>_<module>_<content>.sql;
//   - the target of every ALTER TABLE is created by the file name's module,
//     so a migration belongs to the module that owns the table it changes;
//   - each sqlc entry is a module's (queries in its adapter/postgres) and
//     lists exactly that module's migrations: no other module's, and never
//     River's, which no module queries;
//   - every module with queries has an entry.
func sqlcScopeViolations(entries []sqlcEntry, migrations []migrationFile, queryModules []string) []string {
	var found []string
	report := func(format string, args ...any) { found = append(found, fmt.Sprintf(format, args...)) }

	moduleOf := map[string]string{}   // migration file → module
	byModule := map[string][]string{} // module → its migrations, as sqlc.yaml lists them
	owner := map[string]string{}      // table → module that creates it
	for _, m := range migrations {
		match := migrationFileName.FindStringSubmatch(m.name)
		if match == nil {
			report("migration %s is not named <version>_<module>_<content>.sql", m.name)
			continue
		}
		moduleOf[m.name] = match[1]
		byModule[match[1]] = append(byModule[match[1]], "migrations/sql/"+m.name)
		for _, c := range createTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(m.sql, ""), -1) {
			table := strings.ToLower(c[1])
			if prev, ok := owner[table]; ok && prev != match[1] {
				report("tables: %s is created by both %s and %s", table, prev, match[1])
				continue
			}
			owner[table] = match[1]
		}
	}
	for _, m := range migrations {
		module, ok := moduleOf[m.name]
		if !ok {
			continue
		}
		var altered []string // once per table: Up and Down both alter it
		for _, a := range alterTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(m.sql, ""), -1) {
			if table := strings.ToLower(a[1]); !slices.Contains(altered, table) {
				altered = append(altered, table)
			}
		}
		for _, table := range altered {
			switch tableOwner, known := owner[table]; {
			case !known:
				report("migration %s alters %s, which no migration creates", m.name, table)
			case tableOwner != module:
				report("migration %s alters %s, which module %s creates: the migration belongs to %s", m.name, table, tableOwner, tableOwner)
			}
		}
	}

	covered := map[string]bool{}
	for _, e := range entries {
		match := moduleQueries.FindStringSubmatch(e.queries)
		if match == nil {
			report("sqlc entry %s: queries must be internal/modules/<module>/adapter/postgres/queries", e.queries)
			continue
		}
		module := match[1]
		covered[module] = true
		if want := "internal/modules/" + module + "/adapter/postgres/gen"; e.out != want {
			report("sqlc entry %s: out is %s, want %s", module, e.out, want)
		}
		for _, s := range e.schema {
			if other, ok := moduleOf[filepath.Base(s)]; ok && other != module {
				report("sqlc entry %s lists %s, a migration of %s", module, s, other)
			} else if !ok {
				report("sqlc entry %s lists %s, which is not a migration", module, s)
			}
		}
		for _, own := range byModule[module] {
			if !slices.Contains(e.schema, own) {
				report("sqlc entry %s does not list its migration %s", module, own)
			}
		}
	}
	for _, module := range slices.Sorted(slices.Values(queryModules)) {
		if !covered[module] {
			report("module %s has adapter/postgres/queries but no sqlc entry", module)
		}
	}
	return found
}
```

`server/internal/archtest/sqlc_cases_test.go`：

```go
package archtest

import (
	"slices"
	"testing"
)

// The spike's layout (M2 design 3.14): identity creates users; a later
// module, asset, creates assets; identity's own later migration alters
// users to reference assets; River's migration belongs to no entry.
func sqlcBase() ([]sqlcEntry, []migrationFile, []string) {
	entries := []sqlcEntry{
		{
			schema:  []string{"migrations/sql/00001_identity_users.sql", "migrations/sql/00021_identity_users_avatar_asset.sql"},
			queries: "internal/modules/identity/adapter/postgres/queries",
			out:     "internal/modules/identity/adapter/postgres/gen",
		},
		{
			schema:  []string{"migrations/sql/00020_asset_assets.sql"},
			queries: "internal/modules/asset/adapter/postgres/queries",
			out:     "internal/modules/asset/adapter/postgres/gen",
		},
	}
	migrations := []migrationFile{
		{"00001_identity_users.sql", "-- +goose Up\nCREATE TABLE users (id uuid PRIMARY KEY);\n-- +goose Down\nDROP TABLE users;\n"},
		{"00005_river_main_v2_to_v7.sql", "-- +goose Up\nCREATE TABLE river_job (id bigint);\nALTER TABLE river_job ADD COLUMN x int;\n"},
		{"00020_asset_assets.sql", "-- +goose Up\nCREATE TABLE IF NOT EXISTS assets (id uuid PRIMARY KEY);\n"},
		{"00021_identity_users_avatar_asset.sql", "-- +goose Up\n-- ALTER TABLE assets would be wrong; a comment is not SQL\n" +
			"ALTER TABLE users ADD COLUMN avatar_asset_id uuid REFERENCES assets ON DELETE SET NULL;\n" +
			"-- +goose Down\nALTER TABLE ONLY public.users DROP COLUMN avatar_asset_id;\n"},
	}
	return entries, migrations, []string{"identity", "asset"}
}

func TestSQLCScopeOfTheBaseLayoutPasses(t *testing.T) {
	if got := sqlcScopeViolations(sqlcBase()); len(got) != 0 {
		t.Errorf("violations = %q, want none", got)
	}
}

func TestSQLCScopeReportsViolations(t *testing.T) {
	tests := []struct {
		name   string
		change func(entries []sqlcEntry, migrations []migrationFile, modules []string) ([]sqlcEntry, []migrationFile, []string)
		want   string
	}{
		{"ALTER TABLE in a file of another module", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[3].name = "00021_asset_users_avatar.sql"
			e[0].schema = e[0].schema[:1]
			e[1].schema = append(e[1].schema, "migrations/sql/00021_asset_users_avatar.sql")
			return e, m, mods
		}, "migration 00021_asset_users_avatar.sql alters users, which module identity creates: the migration belongs to identity"},
		{"ALTER TABLE of an unknown table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[0].sql = "CREATE TABLE people (id uuid);"
			return e, m, mods
		}, "migration 00021_identity_users_avatar_asset.sql alters users, which no migration creates"},
		{"a migration of another module", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].schema = append(e[1].schema, "migrations/sql/00001_identity_users.sql")
			return e, m, mods
		}, "sqlc entry asset lists migrations/sql/00001_identity_users.sql, a migration of identity"},
		{"River's migration", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[0].schema = append(e[0].schema, "migrations/sql/00005_river_main_v2_to_v7.sql")
			return e, m, mods
		}, "sqlc entry identity lists migrations/sql/00005_river_main_v2_to_v7.sql, a migration of river"},
		{"an own migration missing", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[0].schema = e[0].schema[:1]
			return e, m, mods
		}, "sqlc entry identity does not list its migration migrations/sql/00021_identity_users_avatar_asset.sql"},
		{"not a migration", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].schema = append(e[1].schema, "migrations/sql/schema.sql")
			return e, m, mods
		}, "sqlc entry asset lists migrations/sql/schema.sql, which is not a migration"},
		{"a module with queries but no entry", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			return e, m, append(mods, "workspace")
		}, "module workspace has adapter/postgres/queries but no sqlc entry"},
		{"queries outside a module's adapter", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].queries = "queries/asset"
			return e, m, mods[:1]
		}, "sqlc entry queries/asset: queries must be internal/modules/<module>/adapter/postgres/queries"},
		{"out outside the module's adapter", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].out = "internal/db"
			return e, m, mods
		}, "sqlc entry asset: out is internal/db, want internal/modules/asset/adapter/postgres/gen"},
		{"a misnamed migration", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m = append(m, migrationFile{"00030-assets.sql", ""})
			return e, m, mods
		}, "migration 00030-assets.sql is not named <version>_<module>_<content>.sql"},
		{"a table created twice", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "CREATE TABLE users (id uuid);"
			return e, m, mods
		}, "tables: users is created by both identity and asset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqlcScopeViolations(tt.change(sqlcBase()))
			if !slices.Equal(got, []string{tt.want}) {
				t.Errorf("violations = %q, want %q", got, tt.want)
			}
		})
	}
}
```

Run: `go -C server test -count=1 ./internal/archtest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/archtest`

- [ ] **Step 4: S1 的迁移断言**

`e2e/tsconfig.json`（`lib` 改为 `ES2023`，S1 用 `toSorted`）：

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "verbatimModuleSyntax": true,
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "skipLibCheck": true,
    "noEmit": true,
    "types": ["node"]
  },
  "include": ["**/*.ts"]
}
```

`e2e/stories/smoke/s1-server-ready.spec.ts`：

```ts
import { readdirSync } from "node:fs";
import path from "node:path";

import { applicationName, runNerve } from "../../fixtures/server";
import { expect, test } from "../../fixtures/test";

/** The migration files bin/nerve embeds, in order. */
const migrationFiles = readdirSync(path.resolve(import.meta.dirname, "../../../server/migrations/sql"))
  .filter((name) => name.endsWith(".sql"))
  .toSorted();

test("S1: an operator starts nerve with the test configuration and it becomes ready", async ({ request, db }) => {
  const healthz = await request.get("/healthz");
  expect(healthz.status()).toBe(200);
  expect(await healthz.json()).toEqual({ status: "ok" });

  const readyz = await request.get("/readyz");
  expect(readyz.status()).toBe(200);
  expect(await readyz.json()).toEqual({ status: "ok" });

  // Since /readyz, nerve keeps a pooled connection to this worker's database.
  const [sessions] = await db.query<{ count: number }>(
    "SELECT count(*)::int AS count FROM pg_stat_activity WHERE datname = current_database() AND application_name = $1",
    [applicationName]
  );
  expect(sessions?.count).toBeGreaterThan(0);

  // Every migration file is applied, and the database's latest version is
  // the last file's.
  expect(migrationFiles.length).toBeGreaterThan(0);
  const { stdout } = await runNerve(["migrate", "status"], db.url);
  const [header, ...rows] = stdout.trimEnd().split("\n");
  expect(header?.split(/\s{2,}/)).toEqual(["VERSION", "STATE", "APPLIED AT", "SOURCE"]);
  expect(rows.map((row) => row.split(/\s+/)).map(([, state, , source]) => [state, source])).toEqual(
    migrationFiles.map((file) => ["applied", file])
  );
  const [latest] = await db.query<{ version: number }>("SELECT max(version_id)::int AS version FROM goose_db_version");
  expect(latest?.version).toBe(Number.parseInt(migrationFiles.at(-1) ?? "", 10));
});
```

- [ ] **Step 5: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。生成的 `adapter/postgres/gen` 还没有使用者，编译通过即可（Task 9 使用）。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make e2e`
Expected: 全部通过，S1 核对三个迁移都是 `applied`、`goose_db_version` 的最大版本是 3。

- [ ] **Step 6: 提交**

```bash
git add server/migrations server/sqlc.yaml server/internal/modules/identity/adapter/postgres server/internal/archtest server/internal/platform/postgres/migrator_test.go server/internal/bootstrap/app_test.go server/tools/go.mod server/tools/go.sum Makefile e2e/tsconfig.json e2e/stories/smoke/s1-server-ready.spec.ts
```
```bash
git commit -m "feat(M2/P1): users, profiles and auth_sessions; sqlc scoped to each module's own migrations

sqlc runs without cgo from the tools module, and make gen-go writes its
output. TestSQLCSchemaScope keeps every module's queries on its own tables,
and S1 checks that every migration is applied.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 三个迁移 up、down、再 up 通过；19 个约束和索引名、25 个反例通过；`make gen-check` 覆盖 sqlc 的输出且干净；`TestSQLCSchemaScope` 通过；S1 通过。

---

### Task 7: `identity` 领域：邮箱、密码与常见密码名单、刷新令牌的布局、新账户

**Files:**
- Create: `server/internal/modules/identity/domain/errors.go`、`email.go`、`email_test.go`、`password.go`、`password_test.go`、`session.go`、`session_test.go`、`user.go`、`user_test.go`
- Create: `tools/password-blocklist/build.mjs`
- Generate: `server/internal/modules/identity/domain/common_passwords.txt`（由 `build.mjs` 生成并提交）
- Modify: `knip.jsonc`

**Interfaces:**
- Consumes: `shared.FieldError`、字段码常量、`shared.NewError`、`shared.Invalid`（Task 2）。
- Produces:
  - `domain.ErrSignupDisabled`（403 `identity.signup_disabled`）、`domain.ErrEmailTaken`（409 `identity.email_taken`）；
  - `MaxEmailLength`、`NormalizeEmail`、`ValidEmail`、`DisplayNameFromEmail`；
  - `PasswordRules`、`NewPasswordRules()`、`(*PasswordRules).Check(field, password, email) *shared.FieldError`；
  - `RefreshTokenPrefix`、`RefreshToken{SessionID, Generation, Secret, Tag}` 及 `MACMessage()`、`SecretHash()`、`String()`；`MaxUserAgentLength`、`SanitizeUserAgent`；
  - `User`（`GetMe` 读出的账户）、`NewAccount(rules, email, password) (string, error)`。
- 规则见 spec 2.11。`domain` 只导入标准库和 `internal/shared`。

**Tests:**
- `email_test.go`：`TestNormalizeEmail`；`TestValidEmail`（Django 5.2 自己测试 `EmailValidator` 的样例中适用于规范化地址的部分，合法、不合法两组；设计阶段已用 Python 的原始正则逐条对照）；`TestValidEmailLengthLimit`（255 个字符合法，256 个不合法）；`TestDisplayNameFromEmail`。
- `password_test.go`：`TestPasswordComposition`（16 个：空、7/8/128/129 个字符、缺每一类、集合之外的特殊字符、非 ASCII 字母不算大小写、按 UTF-16 码元计的长度）；`TestCommonPasswords`（M2 设计 3.8 列出的 14 个拒绝、3 个接受）；`TestPasswordWhoseCoreIsTheEmailsLocalPart`；`TestCore`；`TestCommonPasswordList`（条数 33,887、排序去重、每条满足某个过滤条件、文件头的 SHA-256 和 OGL 两行）。
- `session_test.go`：`TestRefreshTokenLayout`（合 `nrv_rt_[A-Za-z0-9_-]{91}`，解码后 68 字节：会话 id、大端的代数、密文、标签，`MACMessage` 是前 52 字节）；`TestRefreshTokenSecretHash`；`TestSanitizeUserAgent`。
- `user_test.go`：`TestNewAccountNormalizesTheEmail`；`TestNewAccountReportsEveryField`（邮箱和密码的问题一次返回，422 `validation_failed`）。

- [ ] **Step 1: 名单的生成脚本**

`tools/password-blocklist/build.mjs`：

```js
// 生成常见密码名单 server/internal/modules/identity/domain/common_passwords.txt（M2 设计 3.8）。
//
// 用法：node tools/password-blocklist/build.mjs <PwnedPasswordsTop100k.txt>
//
// 原文件是英国国家网络安全中心（NCSC）发布的泄露最多的前 10 万个密码，不进仓库。NCSC 的原地址
// 已失效（2026-09 实测 404）；SecLists 收录的是同一个文件（SHA-256 相同），从这里下载：
//   https://raw.githubusercontent.com/danielmiessler/SecLists/master/Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt
// 脚本先核对它的 SHA-256，再只留下运行时的两种查找可能命中的条目（小写的整个密码、主干），
// 转小写、去重、按 UTF-8 字节排序（Go 的 sort.Strings 的顺序），写出文件头和名单。
// 主干的规则必须与 Go 的 identity/domain.core 相同；Go 的单元测试核对名单满足这里的过滤条件。
import { createHash } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const NCSC_URL = "https://www.ncsc.gov.uk/static-assets/documents/PwnedPasswordsTop100k.txt";
const SECLISTS_PATH = "Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt";
const SOURCE_SHA256 = "c2e5696882c603b76bb67a47ee970897e5a76fc4c3f5547abe3d0ca340c576e0";
const OUT = join(
  dirname(fileURLToPath(import.meta.url)),
  "../../server/internal/modules/identity/domain/common_passwords.txt"
);

// 组合规则的特殊字符，与 web/packages/utils/src/auth.ts 的 getPasswordStrength 相同。
const SPECIAL = `!@#$%^&*()-_+=[]{}|;:'",.<>?/`;
const hasSpecial = (s) => [...s].some((c) => SPECIAL.includes(c));
const asciiLetters = (s) => (s.match(/[a-z]/g) ?? []).length;
// 主干：小写后从两端去掉所有不是字母（Unicode 字母类）的字符。
const core = (s) => s.toLowerCase().replace(/^[^\p{L}]+|[^\p{L}]+$/gu, "");

// e（已转小写）被留下，当且仅当：
// 1. 某个满足组合规则的密码转小写后等于 e：长度 8–128，至少两个 ASCII 字母（一个可改成大写）、一个数字、一个特殊字符；
// 2. 或者某个满足组合规则的密码的主干等于 e：e 的主干是它自己，至少两个 ASCII 字母，
//    长度不超过 126（数字和特殊字符可以在去掉的两端）。
// 长度按 UTF-16 码元计，与界面的 password.length 相同。
const matchesInFull = (e) =>
  e.length >= 8 && e.length <= 128 && asciiLetters(e) >= 2 && /[0-9]/.test(e) && hasSpecial(e);
const matchesAsCore = (e) => e.length > 0 && e.length <= 126 && core(e) === e && asciiLetters(e) >= 2;

const [source] = process.argv.slice(2);
if (!source) {
  console.error("usage: node tools/password-blocklist/build.mjs <PwnedPasswordsTop100k.txt>");
  process.exit(2);
}
const data = readFileSync(source);
const sha256 = createHash("sha256").update(data).digest("hex");
if (sha256 !== SOURCE_SHA256) {
  console.error(`${source}: SHA-256 is ${sha256}, want ${SOURCE_SHA256}`);
  process.exit(1);
}
const entries = new Set(
  data
    .toString("utf8")
    .split(/\r?\n/)
    .filter(Boolean)
    .map((s) => s.toLowerCase())
);
const kept = [...entries]
  .filter((e) => matchesInFull(e) || matchesAsCore(e))
  .toSorted((a, b) => Buffer.compare(Buffer.from(a), Buffer.from(b)));

const header = [
  "# Common passwords that Nerve rejects (M2 design 3.8). Generated by",
  "# tools/password-blocklist/build.mjs; do not edit.",
  "#",
  "# Source: PwnedPasswordsTop100k.txt, the NCSC list of the 100,000 passwords most",
  "# used in breaches, from Have I Been Pwned. NCSC published it at",
  `# ${NCSC_URL}`,
  "# (gone since); SecLists keeps the same file as",
  `# ${SECLISTS_PATH}.`,
  `# SHA-256 ${SOURCE_SHA256}`,
  "# Contains public sector information licensed under the Open Government Licence v3.0:",
  "# https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
  "#",
  "# Kept: the lowercased entries that a password passing the composition rules can",
  "# equal in full (8-128 characters, two ASCII letters, a digit, a special character)",
  "# or as its core (the entry is its own core, has two ASCII letters and at most 126",
  "# characters). One per line, sorted by bytes, after the blank line below.",
];
writeFileSync(OUT, `${header.join("\n")}\n\n${kept.join("\n")}\n`);
console.log(`${source}: ${entries.size} distinct lowercased entries, kept ${kept.length} in ${OUT}`);
```

`knip.jsonc`（脚本由人手动运行，`package.json` 的脚本中看不到它，列为入口）：

```jsonc
// knip：找出未使用的文件、导出和依赖，是门禁（M1/P3 起；make knip 先生成 web 的路由类型，见 docs/v0/M1-frontend-trim/M1-design.md 7.2）。
// 构建产物都在 .gitignore 中，knip 读取 .gitignore，不需要另外忽略。
{
  "$schema": "https://unpkg.com/knip@6/schema-jsonc.json",
  "workspaces": {
    ".": {
      // 由 Makefile 调用（make gen-web、make lint-web 等），package.json 的脚本中看不到
      "ignoreDependencies": ["@redocly/cli", "turbo"],
      // 关键词守卫由 make lint-web 调用；常见密码名单的生成脚本由人手动运行（M2 设计 3.8）。package.json 的脚本中都看不到
      "entry": ["tools/keywords.mjs", "tools/password-blocklist/build.mjs"],
    },
    "web/packages/api-client": {
      // 类型测试只由 tsc 检查，其中的函数不会被调用
      "entry": ["test/*.typecheck.ts"],
      // 生成的类型：webhooks、$defs、operations 等没有被使用
      "ignoreIssues": { "src/schema.gen.ts": ["types"] },
    },
  },
}
```

- [ ] **Step 2: 下载原文件，生成名单**

原文件不进仓库，下载到仓库之外的临时目录（下面用 `<tmp>` 表示，例如本会话的 scratchpad）。NCSC 的原地址已失效，SecLists 收录的是同一个文件（spec 2.11）：

Run: `curl -fsSL -o <tmp>/100k-most-used-passwords-NCSC.txt https://raw.githubusercontent.com/danielmiessler/SecLists/master/Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt`

Run: `shasum -a 256 <tmp>/100k-most-used-passwords-NCSC.txt`
Expected: `c2e5696882c603b76bb67a47ee970897e5a76fc4c3f5547abe3d0ca340c576e0`。不同就停下来报告，不要改脚本里的 SHA-256。

Run: `node tools/password-blocklist/build.mjs <tmp>/100k-most-used-passwords-NCSC.txt`
Expected: `<tmp>/100k-most-used-passwords-NCSC.txt: 97746 distinct lowercased entries, kept 33887 in …/server/internal/modules/identity/domain/common_passwords.txt`

Run: `shasum -a 256 server/internal/modules/identity/domain/common_passwords.txt`
Expected: `74d064a43b9eac19a5e4d58b06ceca3dc740e1df7c0de4a6d95c6efbf5167914`（33,904 行、278,161 字节：16 行文件头、一个空行、33,887 行名单）。

- [ ] **Step 3: 领域代码**

`server/internal/modules/identity/domain/errors.go`：

```go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The identity module's errors (M2 design 5.4). api/modules/identity.yaml
// declares their codes in x-problem-codes.
var (
	// ErrSignupDisabled answers every registration while sign-up is off,
	// before any other check (M2 design 3.9).
	ErrSignupDisabled = shared.NewError(shared.KindForbidden, "identity.signup_disabled", "Sign-up is disabled on this instance.")
	// ErrEmailTaken answers a registration with an address in use.
	ErrEmailTaken = shared.NewError(shared.KindConflict, "identity.email_taken", "An account with this e-mail address already exists.")
)
```

`server/internal/modules/identity/domain/email.go`：

```go
package domain

import (
	"net/netip"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxEmailLength is the length of users.email, varchar(255), in characters.
const MaxEmailLength = 255

// NormalizeEmail is what Plane does before it validates or looks up an
// e-mail address (email.strip().lower(), plane/apps/api/plane/
// authentication/views/app/email.py:73).
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidEmail reports whether a normalized address is acceptable (M2 design
// 4.2): at most 255 characters, no white space or control character
// anywhere, and valid by Django's EmailValidator, which Plane uses
// (plane/apps/api/plane/authentication/adapter/base.py:79).
func ValidEmail(email string) bool {
	if email == "" || utf8.RuneCountInString(email) > MaxEmailLength || !utf8.ValidString(email) {
		return false
	}
	for _, r := range email {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return djangoEmail(email)
}

// The regular expressions of Django 5.2's EmailValidator
// (django/core/validators.py), with every look-around rewritten for RE2 into
// the equivalent explicit form: a domain label of 1–63 characters neither
// starts nor ends with a hyphen.
//
// Django compiles them with re.IGNORECASE on str patterns, under which
// [A-Z] and [a-z] also match four non-ASCII letters (Python's re
// documentation): İ, ı, ſ and K. After lower-casing only ı (U+0131) and
// ſ (U+017F) are left, so the local part's classes add those two.
const (
	djangoUL          = `\x{00a1}-\x{ffff}` // Django's "Unicode letters" range
	djangoLabelChar   = `[a-z` + djangoUL + `0-9]`
	djangoLabel       = djangoLabelChar + `(?:[a-z` + djangoUL + `0-9-]{0,61}` + djangoLabelChar + `)?`
	djangoTLDChar     = `[a-z` + djangoUL + `]`
	djangoTLD         = `\.(?:` + djangoTLDChar + `[a-z` + djangoUL + `-]{0,61}` + djangoTLDChar + `|xn--[a-z0-9]{1,59})`
	djangoAtom        = "[-!#$%&'*+/=?^_" + "`" + `{}|~0-9a-z\x{0131}\x{017f}]+`
	djangoQuotedChars = `[\x01-\x08\x0b\x0c\x0e-\x1f!#-\[\]-\x7f\x{0131}\x{017f}]|\\[\x01-\x09\x0b\x0c\x0e-\x7f\x{0131}\x{017f}]`
)

var (
	djangoUser    = regexp.MustCompile(`(?i)^(?:` + djangoAtom + `(?:\.` + djangoAtom + `)*|"(?:` + djangoQuotedChars + `)*")$`)
	djangoDomain  = regexp.MustCompile(`(?i)^` + djangoLabel + `(?:\.` + djangoLabel + `)*` + djangoTLD + `$`)
	djangoLiteral = regexp.MustCompile(`(?i)^\[([a-f0-9:.]+)\]$`)
)

// djangoEmail is EmailValidator.__call__ with the default allow list, except
// for the length, which ValidEmail bounds more tightly.
func djangoEmail(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	user, domain := email[:at], email[at+1:]
	if !djangoUser.MatchString(user) {
		return false
	}
	if domain == "localhost" || djangoDomain.MatchString(domain) {
		return true
	}
	// A literal address: validate_ipv46_address.
	m := djangoLiteral.FindStringSubmatch(domain)
	if m == nil {
		return false
	}
	_, err := netip.ParseAddr(m[1])
	return err == nil
}

// DisplayNameFromEmail is the display name a new account gets: the part of
// the address before its first @, as Plane's User.save() does
// (plane/apps/api/plane/db/models/user.py:169-187). It is never empty for a
// valid address.
func DisplayNameFromEmail(email string) string {
	name, _, _ := strings.Cut(email, "@")
	return name
}
```

`server/internal/modules/identity/domain/password.go`：

```go
package domain

import (
	_ "embed"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Password lengths, in UTF-16 code units like the web app's password.length.
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// passwordSpecials are the special characters of the composition rules,
// the same as getPasswordStrength in web/packages/utils/src/auth.ts.
const passwordSpecials = `!@#$%^&*()-_+=[]{}|;:'",.<>?/`

//go:embed common_passwords.txt
var commonPasswordsFile string

// PasswordRules are the server's password rules (M2 design 3.8): the
// composition rules the web app shows, and the common-password list.
type PasswordRules struct {
	common []string // sorted: binary search
}

// NewPasswordRules parses the embedded common-password list: a header, a
// blank line, then one lowercased entry per line, sorted by bytes.
func NewPasswordRules() *PasswordRules {
	_, list, _ := strings.Cut(commonPasswordsFile, "\n\n")
	return &PasswordRules{common: strings.Split(strings.TrimSuffix(list, "\n"), "\n")}
}

// Check returns the field error for a new password of the account with the
// given normalized e-mail address, or nil when it is acceptable.
//
//   - weak_password: not 8–128 characters, or it lacks an upper-case letter,
//     a lower-case letter, a digit or a special character (ASCII classes);
//   - common_password: the lowercased password or its core is on the list,
//     or its core is the core of the address's local part.
func (p *PasswordRules) Check(field, password, email string) *shared.FieldError {
	switch {
	case password == "":
		return &shared.FieldError{Field: field, Code: shared.FieldRequired, Message: "is required"}
	case !composed(password):
		return &shared.FieldError{Field: field, Code: shared.FieldWeakPassword,
			Message: "must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of " + passwordSpecials}
	case p.isCommon(password, email):
		return &shared.FieldError{Field: field, Code: shared.FieldCommonPassword, Message: "is too common"}
	}
	return nil
}

func composed(password string) bool {
	n := len(utf16.Encode([]rune(password)))
	return n >= MinPasswordLength && n <= MaxPasswordLength &&
		strings.ContainsFunc(password, func(r rune) bool { return r >= 'A' && r <= 'Z' }) &&
		strings.ContainsFunc(password, func(r rune) bool { return r >= 'a' && r <= 'z' }) &&
		strings.ContainsFunc(password, func(r rune) bool { return r >= '0' && r <= '9' }) &&
		strings.ContainsAny(password, passwordSpecials)
}

func (p *PasswordRules) isCommon(password, email string) bool {
	c := core(password)
	local, _, _ := strings.Cut(email, "@")
	return p.listed(strings.ToLower(password)) || p.listed(c) || (c != "" && c == core(local))
}

func (p *PasswordRules) listed(s string) bool {
	_, found := slices.BinarySearch(p.common, s)
	return found
}

// core lowercases s and trims every character that is not a letter
// (unicode.IsLetter, \p{L}) from both ends: Password1!~, ~Password1! and
// "Password1! " all have the core "password". tools/password-blocklist
// applies the same rule when it builds the list.
func core(s string) string {
	return strings.TrimFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) })
}
```

`server/internal/modules/identity/domain/session.go`：

```go
package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"unicode/utf8"
	"uuid"
)

// RefreshTokenPrefix starts every refresh token (M2 design 3.4).
const RefreshTokenPrefix = "nrv_rt_"

// The layout of a refresh token's 68 bytes (M2 design 3.4): session id,
// generation (big endian), secret, and the MAC tag of the first 52 bytes.
const (
	refreshTokenLen = 16 + 4 + 32 + 16
	macMessageLen   = 16 + 4 + 32
)

// RefreshToken is the content of a refresh token. The server stores only
// SecretHash of the current generation; Tag proves that the server issued a
// generation without storing it.
type RefreshToken struct {
	SessionID  uuid.UUID
	Generation uint32
	Secret     [32]byte
	Tag        [16]byte
}

// MACMessage is what the tag authenticates: session id ‖ generation ‖ secret.
func (t RefreshToken) MACMessage() []byte {
	return t.bytes()[:macMessageLen]
}

// SecretHash is the SHA-256 of the secret, what auth_sessions.token_hash holds.
func (t RefreshToken) SecretHash() []byte {
	h := sha256.Sum256(t.Secret[:])
	return h[:]
}

// String is the token the client holds: nrv_rt_ and the 68 bytes in
// unpadded base64url, 91 characters.
func (t RefreshToken) String() string {
	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(t.bytes())
}

func (t RefreshToken) bytes() []byte {
	b := make([]byte, 0, refreshTokenLen)
	b = append(b, t.SessionID[:]...)
	b = binary.BigEndian.AppendUint32(b, t.Generation)
	b = append(b, t.Secret[:]...)
	return append(b, t.Tag[:]...)
}

// MaxUserAgentLength bounds auth_sessions.user_agent, in characters.
const MaxUserAgentLength = 512

// SanitizeUserAgent is the User-Agent a session records: valid UTF-8, no
// NUL (Postgres text cannot hold it), at most 512 characters.
func SanitizeUserAgent(ua string) string {
	ua = strings.ReplaceAll(strings.ToValidUTF8(ua, string(utf8.RuneError)), "\x00", "")
	if utf8.RuneCountInString(ua) <= MaxUserAgentLength {
		return ua
	}
	return string([]rune(ua)[:MaxUserAgentLength])
}
```

`server/internal/modules/identity/domain/user.go`：

```go
// Package domain holds the identity module's rules (M2 design 6.2): pure
// functions and values, no I/O.
package domain

import (
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// User is an account as the API shows it.
type User struct {
	ID          uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
	Timezone    string
	CreatedAt   time.Time
}

// NewAccount checks the e-mail address and the password of a new account
// and returns the normalized address. Every problem is reported at once, as
// one 422 validation_failed.
func NewAccount(rules *PasswordRules, email, password string) (string, error) {
	email = NormalizeEmail(email)
	var fields []shared.FieldError
	switch {
	case email == "":
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldRequired, Message: "is required"})
	case utf8.RuneCountInString(email) > MaxEmailLength:
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldTooLong, Message: "must be at most 255 characters"})
	case !ValidEmail(email):
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldInvalidFormat, Message: "is not a valid e-mail address"})
	}
	if f := rules.Check("password", password, email); f != nil {
		fields = append(fields, *f)
	}
	if len(fields) > 0 {
		return "", shared.Invalid(fields...)
	}
	return email, nil
}
```

- [ ] **Step 4: 测试**

`server/internal/modules/identity/domain/email_test.go`：

```go
package domain

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"  Alice@Corp.COM\t\n": "alice@corp.com",
		"ÉLODIE@EXÄMPLE.COM":   "élodie@exämple.com",
		"\xc2\xa0bob@corp.com": "bob@corp.com", // U+00A0: TrimSpace takes it too
	} {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

// The cases follow Django 5.2's own tests of EmailValidator
// (tests/validators/tests.py) where they apply to normalized addresses.
var (
	validEmails = []string{
		"email@here.com",
		"weirder-email@here.and.there.com",
		"email@[127.0.0.1]",
		"email@[2001:dB8::1]",
		"email@[2001:db8:0:0:0:0:0:1]",
		"email@[::fffF:127.0.0.1]",
		"example@valid-----hyphens.com",
		"example@valid-with-hyphens.com",
		"test@domain.with.idn.tld.उदाहरण.परीक्षा",
		"email@localhost",
		`"test@test"@example.com`,
		"example@atm." + strings.Repeat("a", 63),
		"example@" + strings.Repeat("a", 63) + ".atm",
		"example@" + strings.Repeat("a", 63) + "." + strings.Repeat("b", 10) + ".atm",
		"a.b+c@sub.example.co.uk",
		"x@xn--80ak6aa92e.xn--p1ai",
		"elodie@exämple.com",
		"ıſ@example.com", // Python's IGNORECASE matches ı and ſ with [A-Z]
	}
	invalidEmails = []string{
		"",
		"abc",
		"abc@",
		"@abc.com",
		"a @x.cz",
		"abc@.com",
		"something@@somewhere.com",
		"email@127.0.0.1",
		"email@[127.0.0.256]",
		"email@[2001:db8::12345]",
		"email@[2001:db8:0:0:0:0:1]",
		"email@[::ffff:127.0.0.256]",
		"email@[2001:dg8::1]",
		"email@[2001:dG8:0:0:0:0:0:1]",
		"email@[::fTzF:127.0.0.1]",
		"example@invalid-.com",
		"example@-invalid.com",
		"example@invalid.com-",
		"example@inv-.alid-.com",
		"example@inv-.-alid.com",
		"test@example.com\n\n<script src=\"x.js\">",
		"\"\\\t\"@here.com", // an escaped tab: Django accepts it, control characters are refused first
		"trailingdot@shouldfail.com.",
		"a@b.com\n",
		"a\n@b.com",
		`"test@test"\n@example.com`,
		"a@[127.0.0.1]\n",
		"example@atm." + strings.Repeat("a", 64),
		"example@" + strings.Repeat("b", 64) + ".atm.localhost",
		"example@atm." + strings.Repeat("a", 59) + "xn--", // a TLD ends in a letter
		"a@b",
		"a@b.c",
		"a@b.c0m",
		"a..b@c.com",
		".a@c.com",
		"a.@c.com",
		"élodie@exämple.com", // Django's local part is ASCII
		"é@x.com",
		"a@😀.com",
		`"a b"@example.com`,
		"a@ex\xe3\x80\x80ample.com", // U+3000: Django accepts it in a domain; white space is refused first
		strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 60) + ".com",
	}
)

func TestValidEmail(t *testing.T) {
	for _, e := range validEmails {
		if !ValidEmail(e) {
			t.Errorf("ValidEmail(%q) = false, want true", e)
		}
	}
	for _, e := range invalidEmails {
		if ValidEmail(e) {
			t.Errorf("ValidEmail(%q) = true, want false", e)
		}
	}
}

func TestValidEmailLengthLimit(t *testing.T) {
	address := func(last int) string {
		return strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", last) + ".com"
	}
	if at255 := address(58); len(at255) != 255 || !ValidEmail(at255) {
		t.Errorf("a %d-character address: valid = %v, want true", len(at255), ValidEmail(at255))
	}
	// Longer than users.email holds, though Django allows 320.
	if at256 := address(59); len(at256) != 256 || ValidEmail(at256) {
		t.Errorf("a %d-character address is valid, want invalid", len(at256))
	}
}

func TestDisplayNameFromEmail(t *testing.T) {
	for in, want := range map[string]string{
		"alice@corp.com":          "alice",
		`"a@b"@example.com`:       `"a`, // Plane's email.split("@")[0]
		"élodie@exämple.com":      "élodie",
		"first.last+tag@corp.com": "first.last+tag",
	} {
		if got := DisplayNameFromEmail(in); got != want {
			t.Errorf("DisplayNameFromEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
```

`server/internal/modules/identity/domain/password_test.go`：

```go
package domain

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var rules = NewPasswordRules()

func checkCode(password, email string) string {
	if f := rules.Check("password", password, email); f != nil {
		return f.Code
	}
	return ""
}

func TestPasswordComposition(t *testing.T) {
	tests := []struct{ name, password, want string }{
		{"empty", "", shared.FieldRequired},
		{"7 characters", "Xq7!vbn", shared.FieldWeakPassword},
		{"8 characters", "Xq7!vbnz", ""},
		{"128 characters", "Xq7!" + strings.Repeat("vbnz", 31), ""},
		{"129 characters", "Xq7!" + strings.Repeat("vbnz", 31) + "k", shared.FieldWeakPassword},
		{"no upper-case letter", "xq7!vbnzk", shared.FieldWeakPassword},
		{"no lower-case letter", "XQ7!VBNZK", shared.FieldWeakPassword},
		{"no digit", "Xqa!vbnzk", shared.FieldWeakPassword},
		{"no special character", "Xq7avbnzk", shared.FieldWeakPassword},
		{"a special outside the set", "Xq7~vbnzk", shared.FieldWeakPassword},
		{"non-ASCII letters are not upper or lower case", "ÄÖ7!äöüß", shared.FieldWeakPassword},
		{"non-ASCII beside the classes", "Xq7!vbnzé", ""},
		// Length is UTF-16 code units, like password.length in the web app.
		{"two emoji make 8 units", "Xq7!😀😀", ""},
		{"one emoji makes 7 units", "Xq7!v😀", shared.FieldWeakPassword},
		{"64 emoji are 128 units", "Xq7!" + strings.Repeat("😀", 62), ""},
		{"65 emoji are 130 units", "Xq7!" + strings.Repeat("😀", 63), shared.FieldWeakPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkCode(tt.password, "someone@example.com"); got != tt.want {
				t.Errorf("Check(%q) = %q, want %q (%d UTF-16 units)", tt.password, got, tt.want, len(utf16.Encode([]rune(tt.password))))
			}
		})
	}
}

// M2 design 3.8 lists these: the common words that the composition rules
// let through, and the seven spellings that got past the second draft.
func TestCommonPasswords(t *testing.T) {
	common := []string{
		"Password1!", "Summer2024!", "Qwerty123!", "Welcome1!", "P@ssw0rd1", "Dragon#2026", "Zxcvbnm1!",
		"Password1!~", "Password1! ", "~Password1!", "Summer2024!`", `Welcome1!\`, "Qwerty123!~", "Dragon#2026 ",
	}
	for _, p := range common {
		if got := checkCode(p, "someone@example.com"); got != shared.FieldCommonPassword {
			t.Errorf("Check(%q) = %q, want common_password", p, got)
		}
	}
	for _, p := range []string{"Tr0ub4dor&3", "Correct-Horse-9", "Nerve2026!"} {
		if got := checkCode(p, "someone@example.com"); got != "" {
			t.Errorf("Check(%q) = %q, want accepted", p, got)
		}
	}
}

func TestPasswordWhoseCoreIsTheEmailsLocalPart(t *testing.T) {
	if got := checkCode("Liuwei123!", "liuwei@example.com"); got != shared.FieldCommonPassword {
		t.Errorf("Check(Liuwei123!, liuwei@…) = %q, want common_password", got)
	}
	if got := checkCode("Liuwei123!", "someone@example.com"); got != "" {
		t.Errorf("Check(Liuwei123!, someone@…) = %q, want accepted", got)
	}
	// Only the ends are trimmed: the dot inside stays in both cores.
	if got := checkCode("~Zhang.San9", "zhang.san@example.com"); got != shared.FieldCommonPassword {
		t.Errorf("Check(~Zhang.San9, zhang.san@…) = %q, want common_password", got)
	}
}

func TestCore(t *testing.T) {
	for in, want := range map[string]string{
		"Password1!~":  "password",
		"~Password1!":  "password",
		"Password1! ":  "password",
		"12!Pass-word": "pass-word",
		"Ünïcödé9!":    "ünïcödé",
		"2024!!":       "",
	} {
		if got := core(in); got != want {
			t.Errorf("core(%q) = %q, want %q", in, got, want)
		}
	}
}

// The list is what tools/password-blocklist/build.mjs writes: the header
// names the source and its licence, the entries are sorted, lowercase, and
// each meets one of the build filters. Checking the filters here keeps the
// JavaScript and Go core rules in step (M2 design 3.8).
func TestCommonPasswordList(t *testing.T) {
	header, _, _ := strings.Cut(commonPasswordsFile, "\n\n")
	for _, want := range []string{
		"SHA-256 c2e5696882c603b76bb67a47ee970897e5a76fc4c3f5547abe3d0ca340c576e0",
		"Contains public sector information licensed under the Open Government Licence v3.0:",
		"https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("the list's header lacks %q", want)
		}
	}
	list := rules.common
	if len(list) != 33887 {
		t.Errorf("the list has %d entries, want 33887", len(list))
	}
	if !slices.IsSorted(list) || len(slices.Compact(slices.Clone(list))) != len(list) {
		t.Error("the list is not sorted by bytes without duplicates")
	}
	for _, e := range list {
		units := len(utf16.Encode([]rune(e)))
		inFull := units >= 8 && units <= 128 && asciiLetters(e) >= 2 &&
			strings.ContainsAny(e, "0123456789") && strings.ContainsAny(e, passwordSpecials)
		asCore := e != "" && units <= 126 && core(e) == e && asciiLetters(e) >= 2
		if kept := inFull || asCore; strings.ToLower(e) != e || !kept {
			t.Errorf("entry %q is not lowercase or meets no build filter", e)
		}
	}
}

func asciiLetters(s string) int {
	n := 0
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			n++
		}
	}
	return n
}
```

`server/internal/modules/identity/domain/session_test.go`：

```go
package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
	"uuid"
)

func sampleToken() RefreshToken {
	t := RefreshToken{SessionID: uuid.MustParse("0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"), Generation: 0x01020304}
	for i := range t.Secret {
		t.Secret[i] = byte(0x40 + i)
	}
	for i := range t.Tag {
		t.Tag[i] = byte(0xa0 + i)
	}
	return t
}

// The 68 bytes: session id 0–15, generation 16–19 big endian, secret 20–51,
// tag 52–67 (M2 design 3.4).
func TestRefreshTokenLayout(t *testing.T) {
	tok := sampleToken()
	s := tok.String()

	// The secret-scanning pattern of M2 design 8.6.
	if !regexp.MustCompile(`^nrv_rt_[A-Za-z0-9_-]{91}$`).MatchString(s) {
		t.Fatalf("token %q does not match nrv_rt_[A-Za-z0-9_-]{91}", s)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, RefreshTokenPrefix))
	if err != nil || len(raw) != 68 {
		t.Fatalf("decoded %d bytes, %v; want 68", len(raw), err)
	}
	if !bytes.Equal(raw[0:16], tok.SessionID[:]) || !bytes.Equal(raw[16:20], []byte{1, 2, 3, 4}) ||
		!bytes.Equal(raw[20:52], tok.Secret[:]) || !bytes.Equal(raw[52:68], tok.Tag[:]) {
		t.Errorf("layout = % x", raw)
	}
	if !bytes.Equal(tok.MACMessage(), raw[:52]) {
		t.Errorf("MACMessage() = % x, want the first 52 bytes", tok.MACMessage())
	}
}

func TestRefreshTokenSecretHash(t *testing.T) {
	tok := sampleToken()
	want := sha256.Sum256(tok.Secret[:])
	if got := tok.SecretHash(); !bytes.Equal(got, want[:]) || len(got) != 32 {
		t.Errorf("SecretHash() = % x, want the SHA-256 of the secret", got)
	}
}

func TestSanitizeUserAgent(t *testing.T) {
	long := strings.Repeat("é", 600)
	tests := []struct{ in, want string }{
		{"Mozilla/5.0", "Mozilla/5.0"},
		{"agent\x00/1", "agent/1"},
		{"bad \xff byte", "bad " + string(utf8.RuneError) + " byte"},
		{long, strings.Repeat("é", 512)},
	}
	for _, tt := range tests {
		got := SanitizeUserAgent(tt.in)
		if got != tt.want || !utf8.ValidString(got) {
			t.Errorf("SanitizeUserAgent(%.20q) = %.20q (%d runes), want %.20q", tt.in, got, utf8.RuneCountInString(got), tt.want)
		}
	}
}
```

`server/internal/modules/identity/domain/user_test.go`：

```go
package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestNewAccountNormalizesTheEmail(t *testing.T) {
	email, err := NewAccount(rules, "  Alice@Corp.COM ", "Tr0ub4dor&3")
	if err != nil || email != "alice@corp.com" {
		t.Errorf("NewAccount() = %q, %v; want alice@corp.com", email, err)
	}
}

// Every problem is reported at once, as one 422.
func TestNewAccountReportsEveryField(t *testing.T) {
	tests := []struct {
		name, email, password string
		want                  []shared.FieldError
	}{
		{"both empty", "", "", []shared.FieldError{
			{Field: "email", Code: "required", Message: "is required"},
			{Field: "password", Code: "required", Message: "is required"},
		}},
		{"bad address, weak password", "not-an-address", "short", []shared.FieldError{
			{Field: "email", Code: "invalid_format", Message: "is not a valid e-mail address"},
			{Field: "password", Code: "weak_password", Message: "must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of " + passwordSpecials},
		}},
		{"too long", strings.Repeat("a", 250) + "@x.com", "Tr0ub4dor&3", []shared.FieldError{
			{Field: "email", Code: "too_long", Message: "must be at most 255 characters"},
		}},
		{"common password", "bob@corp.com", "Password1!", []shared.FieldError{
			{Field: "password", Code: "common_password", Message: "is too common"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAccount(rules, tt.email, tt.password)
			var se *shared.Error
			if !errors.As(err, &se) || se.Kind != shared.KindInvalid || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("NewAccount() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}
```

Run: `go -C server test -count=1 ./internal/modules/identity/domain/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/modules/identity/domain`

- [ ] **Step 5: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`internal/archtest` 通过（`domain` 只导入标准库和 `internal/shared`）。

Run: `make lint-web`
Expected: 通过（关键词守卫也扫描名单，没有命中；`build.mjs` 通过 oxlint 和 oxfmt）。

Run: `make knip`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/identity/domain tools/password-blocklist knip.jsonc
```
```bash
git commit -m "feat(M2/P1): identity domain rules for e-mail, passwords, refresh tokens and new accounts

The e-mail rule ports Django 5.2's EmailValidator. The common-password list
is built from the NCSC top 100k (Open Government Licence v3.0) by
tools/password-blocklist/build.mjs and keeps only the entries a password
passing the composition rules can hit.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `common_passwords.txt` 的 SHA-256 等于 `74d064a4…5167914`；`TestCommonPasswordList`、`TestCommonPasswords` 通过；`make lint-web`、`make knip` 通过。

---

### Task 8: `identity` 的端口与用例：`Register`、`GetMe`、`Authenticate`

**Files:**
- Create: `server/internal/modules/identity/app/ports.go`、`create_user.go`、`register.go`、`get_me.go`、`authenticate.go`
- Create: `server/internal/modules/identity/app/fakes_test.go`、`register_test.go`、`get_me_test.go`、`authenticate_test.go`

**Interfaces:**
- Consumes: `domain`（Task 7）；`shared.TxManager`、`shared.Actor`、`WithActor`、`RequireActor`、`Unauthenticated`（Task 2）。
- Produces（端口在使用方 `app` 中声明，spec 2.12）：
  - `ErrNotFound`、`ErrAccessTokenExpired`；
  - `Clock`、`NewUser`、`UserCreator`、`UserReader`、`ProfileCreator`、`NewSession`、`SessionCreator`、`SessionCredential`、`SessionReader`、`PasswordHasher`、`AccessClaims{UserID, SessionID, ExpiresAt}`、`AccessTokens`、`RefreshTokenMAC`、`SignupPolicy`；
  - `NewRegister(RegisterDeps) *Register`，`Execute(ctx, RegisterInput{Email, Password, UserAgent, IP}) (Tokens, error)`；`Tokens{AccessToken, AccessExpiresIn, RefreshToken, RefreshExpiresAt}`；
  - `NewGetMe(UserReader) *GetMe`，`Execute(ctx) (domain.User, error)`；
  - `NewAuthenticate(AccessTokens, SessionReader, Clock) *Authenticate`，`Execute(ctx, token) (shared.Actor, error)`。
- 账户的创建（账户加默认资料）在未导出的 `accounts` 中，P3 的 `nerve users create` 复用它。

**Tests:**（全部用 `fakes_test.go` 中的假实现，不连数据库）
- `register_test.go`：`TestRegisterCreatesTheAccountAndSignsIn`（一个事务写入账户、资料、会话，事务之外没有写入；邮箱已规范化、`display_name`、哈希、审计时间等于时钟、id 是 v7；会话的 UA 已清理、IP、`expires_at = now + 720h`；刷新令牌第 0 代、`token_hash` 是密文的 SHA-256、标签是前 52 字节的 MAC；访问令牌的声明和 15 分钟的期限）；`TestRegisterWhileSignupIsOff`（非法的地址也答 `identity.signup_disabled`，不哈希、不开事务）；`TestRegisterValidatesBeforeHashing`（两个字段一次返回，不哈希）；`TestRegisterWithATakenAddress`（409）；`TestRegisterWhenTheHasherIsBusy`（503 `server_busy` 原样返回，不开事务）；`TestRegisterPolicyFailureIsAnError`；`TestRegisterLogsTheAccountButNoSecret`（"account registered" 带 `user_id`；日志中没有密码、哈希、两个令牌和令牌密文的哈希）。
- `get_me_test.go`：`TestGetMe`；`TestGetMeIsUnauthenticated`（没有 actor、账户已不存在，都是 401）。
- `authenticate_test.go`：`TestAuthenticateAValidToken`；`TestAuthenticateRejects`（8 个，每个是 401 且包着原因：不是我们签的令牌、把刷新令牌当 bearer、过期的访问令牌、会话不存在、别的账户的会话、已撤销、正好在此刻到期、账户已停用）；`TestAuthenticateTellsAnExpiredAccessToken`（`errors.Is(err, app.ErrAccessTokenExpired)`）；`TestAuthenticateDatabaseFailureIsNot401`。

- [ ] **Step 1: 端口**

`server/internal/modules/identity/app/ports.go`：

```go
// Package app holds the identity module's use cases. It declares the ports
// it needs (M2 design 6.3); adapters implement them and module.go wires
// them.
package app

import (
	"context"
	"errors"
	"net/netip"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// ErrNotFound is what a repository returns for a missing row.
var ErrNotFound = errors.New("not found")

// Clock tells the time: every business time comes from it and goes to SQL
// as a parameter (M2 design 3.5, 3.13). platform/clock implements it.
type Clock interface {
	Now() time.Time
}

// NewUser is an account to insert. Its audit columns are Now.
type NewUser struct {
	ID           uuid.UUID
	Email        string // normalized
	PasswordHash string // argon2id PHC string
	DisplayName  string
	Now          time.Time
}

// UserCreator inserts accounts.
type UserCreator interface {
	// CreateUser returns domain.ErrEmailTaken when the address is in use.
	CreateUser(ctx context.Context, u NewUser) error
}

// UserReader reads accounts.
type UserReader interface {
	// GetUser returns ErrNotFound when there is no such account.
	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
}

// ProfileCreator inserts the preferences of a new account.
type ProfileCreator interface {
	// CreateDefaultProfile inserts profile id of userID with every default.
	CreateDefaultProfile(ctx context.Context, id, userID uuid.UUID, now time.Time) error
}

// NewSession is a login to insert, at generation 0.
type NewSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte // SHA-256 of the refresh token's secret
	UserAgent string
	IP        netip.Addr // the zero Addr when unknown
	ExpiresAt time.Time
	Now       time.Time
}

// SessionCreator inserts sessions.
type SessionCreator interface {
	CreateSession(ctx context.Context, s NewSession) error
}

// SessionCredential is what authentication checks of a session.
type SessionCredential struct {
	UserID     uuid.UUID
	ExpiresAt  time.Time
	Revoked    bool
	UserActive bool
}

// SessionReader reads what authentication needs, by primary key.
type SessionReader interface {
	// SessionCredential returns ErrNotFound when there is no such session.
	SessionCredential(ctx context.Context, id uuid.UUID) (SessionCredential, error)
}

// PasswordHasher hashes passwords with argon2id. Hash returns a *shared.Error
// of 503 server_busy when no slot frees up within the wait limit (M2 design 3.8).
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
}

// AccessClaims are the claims of an access token: nothing about permissions
// (M2 design 3.4).
type AccessClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	ExpiresAt time.Time
}

// ErrAccessTokenExpired is what AccessTokens.Verify returns for a token
// whose signature is valid but whose exp has passed: the client's cue to
// refresh (M2 design 3.6). Any other failure is a different error.
var ErrAccessTokenExpired = errors.New("access token expired")

// AccessTokens signs and verifies access tokens (JWT, EdDSA).
type AccessTokens interface {
	Issue(c AccessClaims) (string, error)
	// Verify checks the token at now; ErrAccessTokenExpired when only the
	// expiry fails.
	Verify(token string, now time.Time) (AccessClaims, error)
}

// RefreshTokenMAC tags refresh tokens (M2 design 3.4): the first 16 bytes of
// HMAC-SHA256 under a key derived from the signing key.
type RefreshTokenMAC interface {
	Tag(message []byte) [16]byte
}

// SignupPolicy decides whether registration is open (M2 design 3.9). From
// bootstrap it is auth.signup_enabled; M3 extends it to invitations.
type SignupPolicy interface {
	AllowSignup(ctx context.Context) (bool, error)
}
```

- [ ] **Step 2: 用例**

`server/internal/modules/identity/app/create_user.go`：

```go
package app

import (
	"context"
	"uuid"
)

// accounts creates an account with its default profile: the step that
// registration and, from M2/P3, `nerve users create` share (M2 design 6.2).
// Run it inside the caller's transaction.
type accounts struct {
	users    UserCreator
	profiles ProfileCreator
}

func (a accounts) create(ctx context.Context, u NewUser) error {
	if err := a.users.CreateUser(ctx, u); err != nil {
		return err
	}
	return a.profiles.CreateDefaultProfile(ctx, uuid.NewV7(), u.ID, u.Now)
}
```

`server/internal/modules/identity/app/register.go`：

```go
package app

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/netip"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Tokens are what registration (and from M2/P2 login and refresh) returns.
type Tokens struct {
	AccessToken      string
	AccessExpiresIn  time.Duration
	RefreshToken     string
	RefreshExpiresAt time.Time // the session's absolute end (M2 design 3.5)
}

// RegisterDeps are Register's collaborators and settings.
type RegisterDeps struct {
	Policy     SignupPolicy
	Rules      *domain.PasswordRules
	Hasher     PasswordHasher
	Tx         shared.TxManager
	Users      UserCreator
	Profiles   ProfileCreator
	Sessions   SessionCreator
	Tokens     AccessTokens
	MAC        RefreshTokenMAC
	Clock      Clock
	Logger     *slog.Logger
	AccessTTL  time.Duration // auth.access_token_ttl
	SessionTTL time.Duration // auth.session_ttl
}

// Register creates an account and signs it in: POST /api/v0/auth/register.
type Register struct {
	d        RegisterDeps
	accounts accounts
}

// NewRegister returns the use case.
func NewRegister(d RegisterDeps) *Register {
	return &Register{d: d, accounts: accounts{users: d.Users, profiles: d.Profiles}}
}

// RegisterInput is a registration and where it comes from.
type RegisterInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        netip.Addr
}

// Execute registers in.Email:
//
//  1. closed sign-up answers 403 before any other check (M2 design 3.9);
//  2. the address and the password are validated, all fields at once (422);
//  3. the password is hashed outside the transaction (M2 design 3.5);
//  4. one transaction inserts the account, its profile and its first
//     session; an address in use is 409 identity.email_taken.
//
// The tokens are made before the transaction, so nothing can fail after it
// commits.
func (r *Register) Execute(ctx context.Context, in RegisterInput) (Tokens, error) {
	allowed, err := r.d.Policy.AllowSignup(ctx)
	if err != nil {
		return Tokens{}, err
	}
	if !allowed {
		return Tokens{}, domain.ErrSignupDisabled
	}
	email, err := domain.NewAccount(r.d.Rules, in.Email, in.Password)
	if err != nil {
		return Tokens{}, err
	}
	hash, err := r.d.Hasher.Hash(ctx, in.Password)
	if err != nil {
		return Tokens{}, err
	}

	now := r.d.Clock.Now()
	user := NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: hash, DisplayName: domain.DisplayNameFromEmail(email), Now: now}
	refresh := domain.RefreshToken{SessionID: uuid.NewV7(), Generation: 0}
	_, _ = rand.Read(refresh.Secret[:]) // never fails since Go 1.24
	refresh.Tag = r.d.MAC.Tag(refresh.MACMessage())
	session := NewSession{
		ID:        refresh.SessionID,
		UserID:    user.ID,
		TokenHash: refresh.SecretHash(),
		UserAgent: domain.SanitizeUserAgent(in.UserAgent),
		IP:        in.IP,
		ExpiresAt: now.Add(r.d.SessionTTL),
		Now:       now,
	}
	access, err := r.d.Tokens.Issue(AccessClaims{UserID: user.ID, SessionID: session.ID, ExpiresAt: now.Add(r.d.AccessTTL)})
	if err != nil {
		return Tokens{}, err
	}

	err = r.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := r.accounts.create(ctx, user); err != nil {
			return err
		}
		return r.d.Sessions.CreateSession(ctx, session)
	})
	if err != nil {
		return Tokens{}, err
	}
	r.d.Logger.InfoContext(ctx, "account registered", slog.String("user_id", user.ID.String()))
	return Tokens{
		AccessToken:      access,
		AccessExpiresIn:  r.d.AccessTTL,
		RefreshToken:     refresh.String(),
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}
```

`server/internal/modules/identity/app/get_me.go`：

```go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetMe reads the caller's account: GET /api/v0/me.
type GetMe struct {
	users UserReader
}

// NewGetMe returns the use case.
func NewGetMe(users UserReader) *GetMe {
	return &GetMe{users: users}
}

// Execute returns the account of the request's actor.
func (g *GetMe) Execute(ctx context.Context) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	u, err := g.users.GetUser(ctx, actor.UserID)
	if errors.Is(err, ErrNotFound) {
		// Authentication found the account a moment ago; it is gone now.
		return domain.User{}, shared.Unauthenticated()
	}
	return u, err
}
```

`server/internal/modules/identity/app/authenticate.go`：

```go
package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Authenticate turns a bearer token into the request's actor (M2 design
// 3.4–3.6): the access token's signature and expiry, then one primary-key
// query for the session and its account. Personal access tokens join in
// M2/P3; until then a nrv_pat_ token fails as a JWT like any other.
type Authenticate struct {
	tokens   AccessTokens
	sessions SessionReader
	clock    Clock
}

// NewAuthenticate returns the use case.
func NewAuthenticate(tokens AccessTokens, sessions SessionReader, clock Clock) *Authenticate {
	return &Authenticate{tokens: tokens, sessions: sessions, clock: clock}
}

// The reasons a credential fails. They go to the debug log only; the
// caller always sees 401 unauthorized.
var (
	errSessionUnknown  = errors.New("session does not exist")
	errSessionMismatch = errors.New("session belongs to another account")
	errSessionRevoked  = errors.New("session is revoked")
	errSessionExpired  = errors.New("session has expired")
	errUserDeactivated = errors.New("account is deactivated")
)

// Execute returns the actor of token. An invalid credential is a
// *shared.Error of 401 that wraps the reason; an expired access token also
// matches ErrAccessTokenExpired. Any other error is an internal fault.
func (a *Authenticate) Execute(ctx context.Context, token string) (shared.Actor, error) {
	now := a.clock.Now()
	claims, err := a.tokens.Verify(token, now)
	if err != nil {
		return shared.Actor{}, unauthenticated(err)
	}
	cred, err := a.sessions.SessionCredential(ctx, claims.SessionID)
	switch {
	case errors.Is(err, ErrNotFound):
		return shared.Actor{}, unauthenticated(errSessionUnknown)
	case err != nil:
		return shared.Actor{}, err
	case cred.UserID != claims.UserID:
		return shared.Actor{}, unauthenticated(errSessionMismatch)
	case cred.Revoked:
		return shared.Actor{}, unauthenticated(errSessionRevoked)
	case !now.Before(cred.ExpiresAt):
		return shared.Actor{}, unauthenticated(errSessionExpired)
	case !cred.UserActive:
		return shared.Actor{}, unauthenticated(errUserDeactivated)
	}
	return shared.Actor{UserID: claims.UserID, SessionID: claims.SessionID}, nil
}

func unauthenticated(reason error) error {
	return fmt.Errorf("%w: %w", shared.Unauthenticated(), reason)
}
```

- [ ] **Step 3: 测试**

`server/internal/modules/identity/app/fakes_test.go`：

```go
package app_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// fakeTx runs fn in a context marked as inside the transaction; fakeStore
// records whether each write happened there.
type fakeTx struct{ calls int }

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

func inTx(ctx context.Context) bool { return ctx.Value(inTxKey{}) == true }

// fakeStore is every repository port, in memory.
type fakeStore struct {
	users       []app.NewUser
	profiles    []uuid.UUID // user ids
	sessions    []app.NewSession
	outsideTx   []string // writes made outside a transaction
	createErr   error    // CreateUser's error
	getUser     domain.User
	getUserErr  error
	credential  app.SessionCredential
	credErr     error
	credentials []uuid.UUID // sessions looked up
}

func (s *fakeStore) CreateUser(ctx context.Context, u app.NewUser) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "user")
	}
	if s.createErr != nil {
		return s.createErr
	}
	s.users = append(s.users, u)
	return nil
}

func (s *fakeStore) CreateDefaultProfile(ctx context.Context, _, userID uuid.UUID, _ time.Time) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "profile")
	}
	s.profiles = append(s.profiles, userID)
	return nil
}

func (s *fakeStore) CreateSession(ctx context.Context, n app.NewSession) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "session")
	}
	s.sessions = append(s.sessions, n)
	return nil
}

func (s *fakeStore) GetUser(context.Context, uuid.UUID) (domain.User, error) {
	return s.getUser, s.getUserErr
}

func (s *fakeStore) SessionCredential(_ context.Context, id uuid.UUID) (app.SessionCredential, error) {
	s.credentials = append(s.credentials, id)
	return s.credential, s.credErr
}

// fakeHasher "hashes" by prefixing, and counts its calls.
type fakeHasher struct {
	calls int
	err   error
}

func (h *fakeHasher) Hash(_ context.Context, password string) (string, error) {
	h.calls++
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + password, nil
}

// fakeTokens issues "access:<sid>" and verifies what it issued; tokens in
// expired are expired.
type fakeTokens struct {
	issued  []app.AccessClaims
	claims  map[string]app.AccessClaims
	expired map[string]bool
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{claims: map[string]app.AccessClaims{}, expired: map[string]bool{}}
}

func (f *fakeTokens) Issue(c app.AccessClaims) (string, error) {
	f.issued = append(f.issued, c)
	token := "access:" + c.SessionID.String()
	f.claims[token] = c
	return token, nil
}

var errBadSignature = errors.New("signature is invalid")

func (f *fakeTokens) Verify(token string, _ time.Time) (app.AccessClaims, error) {
	if f.expired[token] {
		return app.AccessClaims{}, app.ErrAccessTokenExpired
	}
	c, ok := f.claims[token]
	if !ok {
		return app.AccessClaims{}, errBadSignature
	}
	return c, nil
}

// fakeMAC tags with the first 16 bytes of SHA-256: deterministic, and
// different for every message.
type fakeMAC struct{}

func (fakeMAC) Tag(message []byte) [16]byte {
	sum := sha256.Sum256(message)
	return [16]byte(sum[:16])
}

type fixedPolicy struct {
	allow bool
	err   error
}

func (p fixedPolicy) AllowSignup(context.Context) (bool, error) { return p.allow, p.err }
```

`server/internal/modules/identity/app/register_test.go`：

```go
package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var now = time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC)

type registerFixture struct {
	store  *fakeStore
	tx     *fakeTx
	hasher *fakeHasher
	tokens *fakeTokens
	logs   *bytes.Buffer
	uc     *app.Register
}

func newRegister(policy app.SignupPolicy) *registerFixture {
	f := &registerFixture{store: &fakeStore{}, tx: &fakeTx{}, hasher: &fakeHasher{}, tokens: newFakeTokens(), logs: &bytes.Buffer{}}
	f.uc = app.NewRegister(app.RegisterDeps{
		Policy:     policy,
		Rules:      domain.NewPasswordRules(),
		Hasher:     f.hasher,
		Tx:         f.tx,
		Users:      f.store,
		Profiles:   f.store,
		Sessions:   f.store,
		Tokens:     f.tokens,
		MAC:        fakeMAC{},
		Clock:      clocktest.At(now),
		Logger:     slog.New(slog.NewJSONHandler(f.logs, nil)),
		AccessTTL:  15 * time.Minute,
		SessionTTL: 720 * time.Hour,
	})
	return f
}

// isV7 reports whether id is a version 7 UUID (RFC 9562 5.7): ids are
// generated by the application, time-ordered.
func isV7(id uuid.UUID) bool { return id[6]>>4 == 7 }

var input = app.RegisterInput{
	Email:     "  Alice@Corp.com ",
	Password:  "Tr0ub4dor&3",
	UserAgent: "agent\x00/1",
	IP:        netip.MustParseAddr("203.0.113.7"),
}

func TestRegisterCreatesTheAccountAndSignsIn(t *testing.T) {
	f := newRegister(fixedPolicy{allow: true})

	tokens, err := f.uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	if f.tx.calls != 1 || len(f.store.outsideTx) != 0 {
		t.Errorf("transactions = %d, writes outside one = %q; want every write in one transaction", f.tx.calls, f.store.outsideTx)
	}
	if len(f.store.users) != 1 || len(f.store.profiles) != 1 || len(f.store.sessions) != 1 {
		t.Fatalf("wrote %d users, %d profiles, %d sessions; want one each", len(f.store.users), len(f.store.profiles), len(f.store.sessions))
	}
	u, s := f.store.users[0], f.store.sessions[0]
	if u.Email != "alice@corp.com" || u.DisplayName != "alice" || u.PasswordHash != "hashed:Tr0ub4dor&3" || !u.Now.Equal(now) || !isV7(u.ID) {
		t.Errorf("user = %+v", u)
	}
	if f.store.profiles[0] != u.ID {
		t.Errorf("profile of %v, want the new user %v", f.store.profiles[0], u.ID)
	}
	if s.UserID != u.ID || s.UserAgent != "agent/1" || s.IP != input.IP || !s.ExpiresAt.Equal(now.Add(720*time.Hour)) || !s.Now.Equal(now) || !isV7(s.ID) {
		t.Errorf("session = %+v", s)
	}

	// The refresh token: generation 0 of the new session, its secret's hash
	// stored, its tag the MAC of the first 52 bytes.
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(tokens.RefreshToken, domain.RefreshTokenPrefix))
	if err != nil || len(raw) != 68 {
		t.Fatalf("refresh token %q: %d bytes, %v", tokens.RefreshToken, len(raw), err)
	}
	secretHash := sha256.Sum256(raw[20:52])
	if uuid.UUID(raw[0:16]) != s.ID || !bytes.Equal(raw[16:20], []byte{0, 0, 0, 0}) ||
		!bytes.Equal(s.TokenHash, secretHash[:]) || [16]byte(raw[52:68]) != (fakeMAC{}).Tag(raw[:52]) {
		t.Errorf("refresh token layout = % x, session %+v", raw, s)
	}
	if !tokens.RefreshExpiresAt.Equal(s.ExpiresAt) {
		t.Errorf("RefreshExpiresAt = %v, want the session's %v", tokens.RefreshExpiresAt, s.ExpiresAt)
	}

	want := app.AccessClaims{UserID: u.ID, SessionID: s.ID, ExpiresAt: now.Add(15 * time.Minute)}
	if len(f.tokens.issued) != 1 || f.tokens.issued[0] != want || tokens.AccessToken != "access:"+s.ID.String() || tokens.AccessExpiresIn != 15*time.Minute {
		t.Errorf("access token %q expiring in %v, claims %+v; want claims %+v", tokens.AccessToken, tokens.AccessExpiresIn, f.tokens.issued, want)
	}
}

// Closed sign-up answers the same for every address: before validation,
// any lookup or any hash (M2 design 3.9).
func TestRegisterWhileSignupIsOff(t *testing.T) {
	f := newRegister(fixedPolicy{allow: false})

	_, err := f.uc.Execute(context.Background(), app.RegisterInput{Email: "not an address", Password: "x"})

	if !errors.Is(err, domain.ErrSignupDisabled) || f.hasher.calls != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() = %v, hashes %d, transactions %d; want identity.signup_disabled and nothing else", err, f.hasher.calls, f.tx.calls)
	}
}

func TestRegisterValidatesBeforeHashing(t *testing.T) {
	f := newRegister(fixedPolicy{allow: true})

	_, err := f.uc.Execute(context.Background(), app.RegisterInput{Email: "not an address", Password: "Password1!"})

	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 2 || f.hasher.calls != 0 {
		t.Errorf("Execute() = %+v, hashes %d; want validation_failed on both fields and no hash", err, f.hasher.calls)
	}
}

func TestRegisterWithATakenAddress(t *testing.T) {
	f := newRegister(fixedPolicy{allow: true})
	f.store.createErr = domain.ErrEmailTaken

	_, err := f.uc.Execute(context.Background(), input)

	if !errors.Is(err, domain.ErrEmailTaken) || len(f.store.sessions) != 0 {
		t.Errorf("Execute() = %v with %d sessions; want identity.email_taken and none", err, len(f.store.sessions))
	}
}

func TestRegisterWhenTheHasherIsBusy(t *testing.T) {
	f := newRegister(fixedPolicy{allow: true})
	f.hasher.err = shared.ServerBusy(time.Second)

	_, err := f.uc.Execute(context.Background(), input)

	if !errors.Is(err, shared.ServerBusy(0)) || f.tx.calls != 0 {
		t.Errorf("Execute() = %v, transactions %d; want server_busy before any write", err, f.tx.calls)
	}
}

func TestRegisterPolicyFailureIsAnError(t *testing.T) {
	boom := errors.New("policy store down")
	f := newRegister(fixedPolicy{err: boom})

	if _, err := f.uc.Execute(context.Background(), input); !errors.Is(err, boom) {
		t.Errorf("Execute() = %v, want the policy's error", err)
	}
}

// The log names the account; it never holds the password, a token or a
// token's hash (M2 design 8.4).
func TestRegisterLogsTheAccountButNoSecret(t *testing.T) {
	f := newRegister(fixedPolicy{allow: true})

	tokens, err := f.uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"account registered"`) || !strings.Contains(logs, f.store.users[0].ID.String()) {
		t.Errorf("logs = %s, want the account registered with its user_id", logs)
	}
	hash := base64.StdEncoding.EncodeToString(f.store.sessions[0].TokenHash)
	for _, secret := range []string{input.Password, tokens.AccessToken, tokens.RefreshToken, hash, "hashed:"} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs hold %q", secret)
		}
	}
}
```

`server/internal/modules/identity/app/get_me_test.go`：

```go
package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestGetMe(t *testing.T) {
	want := domain.User{ID: userID, Email: "alice@corp.com", DisplayName: "alice", Timezone: "UTC", CreatedAt: now}
	uc := app.NewGetMe(&fakeStore{getUser: want})
	ctx := shared.WithActor(context.Background(), shared.Actor{UserID: userID, SessionID: sessionID})

	got, err := uc.Execute(ctx)

	if err != nil || got != want {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
}

func TestGetMeIsUnauthenticated(t *testing.T) {
	authed := shared.WithActor(context.Background(), shared.Actor{UserID: userID})
	tests := []struct {
		name  string
		ctx   context.Context
		store *fakeStore
	}{
		{"without an actor", context.Background(), &fakeStore{}},
		{"when the account is gone", authed, &fakeStore{getUserErr: app.ErrNotFound}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.NewGetMe(tt.store).Execute(tt.ctx)
			if !errors.Is(err, shared.Unauthenticated()) {
				t.Errorf("Execute() = %v, want 401 unauthorized", err)
			}
		})
	}
}
```

`server/internal/modules/identity/app/authenticate_test.go`：

```go
package app_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	userID    = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	sessionID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
)

func newAuthenticate(cred app.SessionCredential, credErr error) (*app.Authenticate, *fakeTokens, *fakeStore) {
	tokens := newFakeTokens()
	store := &fakeStore{credential: cred, credErr: credErr}
	return app.NewAuthenticate(tokens, store, clocktest.At(now)), tokens, store
}

func validCredential() app.SessionCredential {
	return app.SessionCredential{UserID: userID, ExpiresAt: now.Add(time.Hour), UserActive: true}
}

func TestAuthenticateAValidToken(t *testing.T) {
	uc, tokens, store := newAuthenticate(validCredential(), nil)
	token, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(time.Minute)})

	actor, err := uc.Execute(context.Background(), token)

	if err != nil || actor != (shared.Actor{UserID: userID, SessionID: sessionID}) {
		t.Errorf("Execute() = %+v, %v", actor, err)
	}
	if len(store.credentials) != 1 || store.credentials[0] != sessionID {
		t.Errorf("sessions looked up = %v, want one lookup of %v", store.credentials, sessionID)
	}
}

func TestAuthenticateRejects(t *testing.T) {
	other := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	tests := []struct {
		name    string
		cred    func(*app.SessionCredential)
		credErr error
		token   string
		reason  string
	}{
		{"a token that is not ours", nil, nil, "garbage", "signature is invalid"},
		{"a refresh token as bearer", nil, nil, domain.RefreshTokenPrefix + "AAAA", "signature is invalid"},
		{"an expired access token", nil, nil, "expired", "access token expired"},
		{"an unknown session", nil, app.ErrNotFound, "", "session does not exist"},
		{"another account's session", func(c *app.SessionCredential) { c.UserID = other }, nil, "", "session belongs to another account"},
		{"a revoked session", func(c *app.SessionCredential) { c.Revoked = true }, nil, "", "session is revoked"},
		{"a session expiring now", func(c *app.SessionCredential) { c.ExpiresAt = now }, nil, "", "session has expired"},
		{"a deactivated account", func(c *app.SessionCredential) { c.UserActive = false }, nil, "", "account is deactivated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := validCredential()
			if tt.cred != nil {
				tt.cred(&cred)
			}
			uc, tokens, _ := newAuthenticate(cred, tt.credErr)
			token := tt.token
			if token == "" {
				token, _ = tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(time.Minute)})
			}
			if token == "expired" {
				tokens.expired[token] = true
			}

			_, err := uc.Execute(context.Background(), token)

			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != 401 || se.Code != shared.CodeUnauthorized {
				t.Fatalf("Execute() = %v, want 401 unauthorized", err)
			}
			if want := "Authentication is required.: " + tt.reason; err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
		})
	}
}

// Only a valid signature with a past exp is "expired": the client's cue to
// refresh, which the M2/P2 failure gate does not count.
func TestAuthenticateTellsAnExpiredAccessToken(t *testing.T) {
	uc, tokens, _ := newAuthenticate(validCredential(), nil)
	tokens.expired["old"] = true

	_, expired := uc.Execute(context.Background(), "old")
	_, forged := uc.Execute(context.Background(), "forged")

	if !errors.Is(expired, app.ErrAccessTokenExpired) || errors.Is(forged, app.ErrAccessTokenExpired) {
		t.Errorf("expired = %v, forged = %v; want only the first to be ErrAccessTokenExpired", expired, forged)
	}
}

func TestAuthenticateDatabaseFailureIsNot401(t *testing.T) {
	boom := errors.New("connection refused")
	uc, tokens, _ := newAuthenticate(app.SessionCredential{}, boom)
	token, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID})

	_, err := uc.Execute(context.Background(), token)

	var se *shared.Error
	if !errors.Is(err, boom) || errors.As(err, &se) {
		t.Errorf("Execute() = %v, want the database error, not a problem", err)
	}
}
```

Run: `go -C server test -count=1 ./internal/modules/identity/app/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/modules/identity/app`

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`internal/archtest` 通过（`app` 不导入适配器、平台和生成代码）。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/identity/app
```
```bash
git commit -m "feat(M2/P1): identity ports and the register, get-me and authenticate use cases

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `app` 的 13 个测试通过；`TestRegisterWhileSignupIsOff` 证明关闭注册时不做任何其他检查；`TestAuthenticateDatabaseFailureIsNot401` 通过。

---

### Task 9: `identity` 的适配器：argon2、签名、仓储、认证器

**Files:**
- Create: `server/internal/modules/identity/adapter/argon2/hasher.go`、`hasher_test.go`
- Create: `server/internal/modules/identity/adapter/signing/keys.go`、`jwt.go`、`mac.go`、`signing_test.go`
- Create: `server/internal/modules/identity/adapter/postgres/store.go`、`users.go`、`sessions.go`、`store_test.go`
- Create: `server/internal/modules/identity/adapter/authn/authenticator.go`、`authenticator_test.go`
- Modify: `server/go.mod`、`server/go.sum`

**Interfaces:**
- Consumes: `app` 的端口（Task 8）；`adapter/postgres/gen`（Task 6）；`postgres.DB`、`clocktest`、`pgtest`（Task 2、M0）；`shared.ServerBusy`、`WithActor`（Task 2）。
- Produces（spec 2.13）：
  - `argon2adapter.New(Params{MemoryKiB, Iterations, Parallelism, MaxConcurrent, MaxWait}, logger) *Hasher`，满足 `app.PasswordHasher`；
  - `signing.ParseKeys(pem) (*Keys, error)`、`signing.EphemeralKeys() *Keys`、`signing.NewAccessTokens(keys)`（满足 `app.AccessTokens`）、`signing.NewRefreshTokenMAC(keys)`（满足 `app.RefreshTokenMAC`）；
  - `postgresadapter.New(pool) *Store`，满足 `UserCreator`、`UserReader`、`ProfileCreator`、`SessionCreator`、`SessionReader`；
  - `authn.New(uc) *Authenticator`，满足 `httpserver.Authenticator`（成功时返回带 actor 的 context 和 `session:<sid>`）。

**Tests:**
- `hasher_test.go`：`TestHashIsAnArgon2idPHCString`（`$argon2id$v=19$m=…,t=…,p=…$盐$哈希`，盐 16 字节、哈希 32 字节，按其中的参数用 `argon2.IDKey` 重算得到同一个哈希，不超过 `users.password` 的 128 个字符）；`TestHashSaltsEveryHash`；`TestHashIsBusyWhenNoSlotFreesUp`（503 `server_busy`、`Retry-After` 1 秒，INFO "password hashing is saturated"）；`TestHashWaitsForAFreedSlot`；`TestHashStopsWaitingWhenTheRequestEnds`（`context.Canceled`）；`BenchmarkHashDefaultParams`。
- `signing_test.go`：`TestParseKeysReadsAnOpenSSLKey`（`openssl genpkey -algorithm ed25519` 生成的测试密钥）；`TestParseKeysRejects`（4 个，错误原文不含输入）；`TestEphemeralKeysDiffer`；`TestAccessTokenRoundTrip`（头部逐字是 `{"alg":"EdDSA","typ":"JWT"}`，载荷逐字只有 `sub`、`exp`、`sid`）；`TestAccessTokenExpiry`（`exp` 前一秒有效，`exp` 那一刻起 `ErrAccessTokenExpired`；签名坏了的过期令牌是无效而不是过期）；`TestAccessTokenVerifyRejects`（10 个，见 spec 2.13）；`TestRefreshTokenMAC`（同一消息的标签相同，改动任一字节标签就变，别的密钥得到别的标签）。
- `store_test.go`（testcontainers，真实迁移）：`TestCreateAndGetUser`（审计列等于固定时钟）；`TestGetUnknownUser`（`app.ErrNotFound`）；`TestCreateUserWithATakenAddress`（`domain.ErrEmailTaken`）；`TestCreateUserBreakingACheckIsInternal`（不是领域错误）；`TestCreateDefaultProfile`（默认值逐列）；`TestCreateSessionAndReadItsCredential`；`TestCreateSessionWithoutAnIP`（`NULL`）；`TestUnknownSessionCredential`。
- `authenticator_test.go`：`TestAuthenticatePutsTheActorInTheContext`（限流键 `session:<sid>`）；`TestAuthenticatePassesErrorsThrough`。

- [ ] **Step 1: 依赖**

Run: `go -C server get github.com/golang-jwt/jwt/v5@v5.3.1 golang.org/x/crypto@v0.57.0`

先写下面的代码再整理依赖（`go mod tidy` 会删掉还没有使用者的直接依赖）。

- [ ] **Step 2: argon2**

`server/internal/modules/identity/adapter/argon2/hasher.go`：

```go
// Package argon2adapter hashes passwords with argon2id (M2 design 3.8), with a cap
// on concurrent hashes and on how long a caller waits for one.
package argon2adapter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Params are auth.password in the configuration.
type Params struct {
	MemoryKiB     uint32        // argon2_memory_kib
	Iterations    uint32        // argon2_iterations
	Parallelism   uint8         // argon2_parallelism
	MaxConcurrent int           // max_concurrent_hashes
	MaxWait       time.Duration // max_wait
}

const (
	saltLen = 16
	keyLen  = 32
	// retryAfter is the Retry-After of the 503 when no slot frees up.
	retryAfter = time.Second
)

// Hasher implements app.PasswordHasher.
type Hasher struct {
	p       Params
	logger  *slog.Logger
	slots   chan struct{}
	waiting atomic.Int64
}

// New returns a hasher with params p.
func New(p Params, logger *slog.Logger) *Hasher {
	return &Hasher{p: p, logger: logger, slots: make(chan struct{}, p.MaxConcurrent)}
}

// Hash returns the PHC string of password:
// $argon2id$v=19$m=<KiB>,t=<iterations>,p=<lanes>$<salt>$<key>, base64
// without padding. When no slot frees up within MaxWait it returns 503
// server_busy with Retry-After: 1 and logs the queue length.
func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-h.slots }()
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt) // never fails since Go 1.24
	key := argon2.IDKey([]byte(password), salt, h.p.Iterations, h.p.MemoryKiB, h.p.Parallelism, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, h.p.MemoryKiB, h.p.Iterations, h.p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.slots <- struct{}{}:
		return nil
	default:
	}
	queued := h.waiting.Add(1)
	defer h.waiting.Add(-1)
	timer := time.NewTimer(h.p.MaxWait)
	defer timer.Stop()
	select {
	case h.slots <- struct{}{}:
		return nil
	case <-timer.C:
		h.logger.LogAttrs(ctx, slog.LevelInfo, "password hashing is saturated", slog.Int64("queued", queued))
		return shared.ServerBusy(retryAfter)
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

`server/internal/modules/identity/adapter/argon2/hasher_test.go`：

```go
package argon2adapter

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// testParams are config.test.yaml's: m = 64 KiB, t = 1.
var testParams = Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 2, MaxWait: 50 * time.Millisecond}

func TestHashIsAnArgon2idPHCString(t *testing.T) {
	phc, err := New(testParams, slog.New(slog.DiscardHandler)).Hash(context.Background(), "Tr0ub4dor&3")
	if err != nil {
		t.Fatal(err)
	}
	var version int
	var m, iterations uint32
	var p uint8
	var salt64, key64 string
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		t.Fatalf("hash %q is not $argon2id$v$params$salt$key", phc)
	}
	if _, err := fmt.Sscanf(parts[2]+" "+parts[3], "v=%d m=%d,t=%d,p=%d", &version, &m, &iterations, &p); err != nil {
		t.Fatalf("hash %q: %v", phc, err)
	}
	salt64, key64 = parts[4], parts[5]
	salt, errSalt := base64.RawStdEncoding.DecodeString(salt64)
	key, errKey := base64.RawStdEncoding.DecodeString(key64)
	if version != argon2.Version || m != 64 || iterations != 1 || p != 1 || errSalt != nil || errKey != nil || len(salt) != 16 || len(key) != 32 {
		t.Errorf("hash %q: v=%d m=%d t=%d p=%d, salt %d bytes, key %d bytes", phc, version, m, iterations, p, len(salt), len(key))
	}
	// What login (M2/P2) will do: the same derivation gives the same key.
	if again := argon2.IDKey([]byte("Tr0ub4dor&3"), salt, iterations, m, p, 32); subtle.ConstantTimeCompare(again, key) != 1 {
		t.Error("re-deriving the key from the PHC parameters gives another key")
	}
	if len(phc) > 128 {
		t.Errorf("hash is %d characters; users.password is varchar(128)", len(phc))
	}
}

func TestHashSaltsEveryHash(t *testing.T) {
	h := New(testParams, slog.New(slog.DiscardHandler))
	a, _ := h.Hash(context.Background(), "Tr0ub4dor&3")
	b, _ := h.Hash(context.Background(), "Tr0ub4dor&3")
	if a == b {
		t.Error("two hashes of one password are equal")
	}
}

// With every slot taken, a caller waits max_wait and gets 503 server_busy
// with Retry-After: 1; the queue length goes to the log (M2 design 3.8, 8.4).
func TestHashIsBusyWhenNoSlotFreesUp(t *testing.T) {
	var logs bytes.Buffer
	h := New(testParams, slog.New(slog.NewJSONHandler(&logs, nil)))
	h.slots <- struct{}{}
	h.slots <- struct{}{}

	start := time.Now()
	_, err := h.Hash(context.Background(), "Tr0ub4dor&3")

	var se *shared.Error
	if !errors.As(err, &se) || se.ProblemStatus() != 503 || se.Code != shared.CodeServerBusy || se.RetryAfter() != time.Second {
		t.Errorf("Hash() = %v, want 503 server_busy with Retry-After 1s", err)
	}
	if waited := time.Since(start); waited < testParams.MaxWait {
		t.Errorf("gave up after %v, want at least max_wait %v", waited, testParams.MaxWait)
	}
	if !strings.Contains(logs.String(), `"msg":"password hashing is saturated","queued":1`) {
		t.Errorf("logs = %s, want the saturation with the queue length", logs.String())
	}
}

func TestHashWaitsForAFreedSlot(t *testing.T) {
	h := New(Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: 5 * time.Second}, slog.New(slog.DiscardHandler))
	h.slots <- struct{}{}
	time.AfterFunc(20*time.Millisecond, func() { <-h.slots })

	if _, err := h.Hash(context.Background(), "Tr0ub4dor&3"); err != nil {
		t.Errorf("Hash() = %v, want a hash once the slot frees up", err)
	}
	if len(h.slots) != 0 {
		t.Errorf("%d slots still taken, want the hash to release its slot", len(h.slots))
	}
}

func TestHashStopsWaitingWhenTheRequestEnds(t *testing.T) {
	h := New(Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: 5 * time.Second}, slog.New(slog.DiscardHandler))
	h.slots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	if _, err := h.Hash(ctx, "Tr0ub4dor&3"); !errors.Is(err, context.Canceled) {
		t.Errorf("Hash() = %v, want context.Canceled", err)
	}
}

// BenchmarkHashDefaultParams measures one hash with the default parameters
// (m = 19456 KiB, t = 2, p = 1): go test -bench . -run ^$ ./internal/modules/identity/adapter/argon2
func BenchmarkHashDefaultParams(b *testing.B) {
	h := New(Params{MemoryKiB: 19456, Iterations: 2, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second}, slog.New(slog.DiscardHandler))
	for b.Loop() {
		if _, err := h.Hash(context.Background(), "Tr0ub4dor&3"); err != nil {
			b.Fatal(err)
		}
	}
}
```

- [ ] **Step 3: 签名**

`server/internal/modules/identity/adapter/signing/keys.go`：

```go
// Package signing holds the instance's Ed25519 key (M2 design 3.7): it signs
// the access tokens and, through a key derived from it, tags the refresh
// tokens (3.4). The key never leaves this package and is never logged.
package signing

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// macInfo separates the MAC key from the signing key (HKDF info, M2 design 3.4).
const macInfo = "nerve refresh-token mac v1"

// Keys are the signing key and the MAC key derived from its seed.
type Keys struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
	mac     []byte
}

// ParseKeys reads a PKCS#8 PEM Ed25519 private key, the format of
// `openssl genpkey -algorithm ed25519`. Errors never quote the input.
func ParseKeys(pemData []byte) (*Keys, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("not a PEM \"PRIVATE KEY\" block (PKCS#8)")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#8: %w", err)
	}
	private, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("the key is %T, want an Ed25519 key", key)
	}
	return newKeys(private)
}

// EphemeralKeys generates a key for this process only: dev and test without
// auth.jwt.private_key_file (M2 design 3.7).
func EphemeralKeys() *Keys {
	_, private, _ := ed25519.GenerateKey(nil) // crypto/rand; never fails
	k, _ := newKeys(private)                  // cannot fail for a generated key
	return k
}

func newKeys(private ed25519.PrivateKey) (*Keys, error) {
	mac, err := hkdf.Key(sha256.New, private.Seed(), nil, macInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive the MAC key: %w", err)
	}
	return &Keys{private: private, public: private.Public().(ed25519.PublicKey), mac: mac}, nil
}
```

`server/internal/modules/identity/adapter/signing/jwt.go`：

```go
package signing

import (
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// claims are the access token's only claims: sub, sid and exp (M2 design 3.4).
type claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
}

// AccessTokens implements app.AccessTokens: JWTs signed with EdDSA
// (Ed25519), header {"alg":"EdDSA","typ":"JWT"}.
type AccessTokens struct {
	keys *Keys
}

// NewAccessTokens returns access tokens signed with keys.
func NewAccessTokens(keys *Keys) *AccessTokens {
	return &AccessTokens{keys: keys}
}

// Issue signs c.
func (a *AccessTokens) Issue(c app.AccessClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: c.UserID.String(), ExpiresAt: jwt.NewNumericDate(c.ExpiresAt)},
		SessionID:        c.SessionID.String(),
	})
	return token.SignedString(a.keys.private)
}

// Verify checks token at now: only EdDSA, strict base64url, exp required.
// A valid signature whose exp has passed is app.ErrAccessTokenExpired: the
// signature is checked before the claims (jwt/v5 parser.go).
func (a *AccessTokens) Verify(token string, now time.Time) (app.AccessClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithStrictDecoding(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	var c claims
	_, err := parser.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return a.keys.public, nil })
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return app.AccessClaims{}, app.ErrAccessTokenExpired
	case err != nil:
		return app.AccessClaims{}, err
	}
	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return app.AccessClaims{}, fmt.Errorf("sub: %w", err)
	}
	sessionID, err := uuid.Parse(c.SessionID)
	if err != nil {
		return app.AccessClaims{}, fmt.Errorf("sid: %w", err)
	}
	return app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: c.ExpiresAt.Time}, nil
}
```

`server/internal/modules/identity/adapter/signing/mac.go`：

```go
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
)

// RefreshTokenMAC implements app.RefreshTokenMAC: the first 16 bytes of
// HMAC-SHA256 under K = HKDF-SHA256(the signing key's seed, info "nerve
// refresh-token mac v1") (M2 design 3.4). The tag lets the server recognize
// an old generation it issued without storing it (M2/P2).
type RefreshTokenMAC struct {
	keys *Keys
}

// NewRefreshTokenMAC returns the MAC of keys.
func NewRefreshTokenMAC(keys *Keys) *RefreshTokenMAC {
	return &RefreshTokenMAC{keys: keys}
}

// Tag returns the tag of message.
func (m *RefreshTokenMAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, m.keys.mac)
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}
```

`server/internal/modules/identity/adapter/signing/signing_test.go`：

```go
package signing

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// testKeyPEM is what `openssl genpkey -algorithm ed25519` writes (OpenSSL
// 3.6.3). A key for tests only.
const testKeyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIGqen6oN2FFQjS+yPQPHLVBIW0B2O9faCmNwftOWxqyE
-----END PRIVATE KEY-----
`

func testKeys(t *testing.T) *Keys {
	t.Helper()
	k, err := ParseKeys([]byte(testKeyPEM))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestParseKeysReadsAnOpenSSLKey(t *testing.T) {
	k := testKeys(t)
	if len(k.private) != 64 || len(k.public) != 32 || len(k.mac) != 32 {
		t.Errorf("key sizes = %d, %d, %d; want 64, 32, 32", len(k.private), len(k.public), len(k.mac))
	}
	if bytes.Equal(k.mac, k.private.Seed()) {
		t.Error("the MAC key equals the seed, want a derived key")
	}
}

func TestParseKeysRejects(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecPKCS8, err := x509.MarshalPKCS8PrivateKey(ec)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ name, pem, want string }{
		{"not PEM", "secret-looking-garbage", `not a PEM "PRIVATE KEY" block (PKCS#8)`},
		{"another block type", strings.ReplaceAll(testKeyPEM, "PRIVATE KEY", "EC PRIVATE KEY"), `not a PEM "PRIVATE KEY" block (PKCS#8)`},
		{"an EC key", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecPKCS8})), "the key is *ecdsa.PrivateKey, want an Ed25519 key"},
		{"not PKCS#8", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("secret-looking-garbage")})), "parse PKCS#8: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseKeys([]byte(tt.pem))
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) || strings.Contains(err.Error(), "secret-looking-garbage") {
				t.Errorf("ParseKeys() = %v, want %q without the input", err, tt.want)
			}
		})
	}
}

func TestEphemeralKeysDiffer(t *testing.T) {
	a, b := EphemeralKeys(), EphemeralKeys()
	if bytes.Equal(a.private, b.private) || bytes.Equal(a.mac, b.mac) {
		t.Error("two ephemeral keys are equal")
	}
}

var (
	userID    = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	sessionID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	now       = time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
)

func issue(t *testing.T, a *AccessTokens, exp time.Time) string {
	t.Helper()
	token, err := a.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: exp})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func segment(t *testing.T, token string, i int) string {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[i])
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestAccessTokenRoundTrip(t *testing.T) {
	a := NewAccessTokens(testKeys(t))
	token := issue(t, a, now.Add(15*time.Minute))

	if h := segment(t, token, 0); h != `{"alg":"EdDSA","typ":"JWT"}` {
		t.Errorf("header = %s", h)
	}
	if p := segment(t, token, 1); p != `{"sub":"`+userID.String()+`","exp":`+strconv.FormatInt(now.Add(15*time.Minute).Unix(), 10)+`,"sid":"`+sessionID.String()+`"}` {
		t.Errorf("payload = %s, want only sub, exp and sid", p)
	}
	got, err := a.Verify(token, now)
	want := app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(15 * time.Minute)}
	if err != nil || got.UserID != want.UserID || got.SessionID != want.SessionID || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("Verify() = %+v, %v; want %+v", got, err, want)
	}
}

func TestAccessTokenExpiry(t *testing.T) {
	a := NewAccessTokens(testKeys(t))
	token := issue(t, a, now.Add(time.Minute))

	if _, err := a.Verify(token, now.Add(59*time.Second)); err != nil {
		t.Errorf("Verify() a second before exp = %v, want valid", err)
	}
	if _, err := a.Verify(token, now.Add(time.Minute)); !errors.Is(err, app.ErrAccessTokenExpired) {
		t.Errorf("Verify() at exp = %v, want ErrAccessTokenExpired", err)
	}
	// An expired token with a bad signature is invalid, not expired.
	forged := token[:len(token)-4] + "AAAA"
	if _, err := a.Verify(forged, now.Add(time.Hour)); err == nil || errors.Is(err, app.ErrAccessTokenExpired) {
		t.Errorf("Verify() of a forged expired token = %v, want invalid and not expired", err)
	}
}

func TestAccessTokenVerifyRejects(t *testing.T) {
	keys := testKeys(t)
	a := NewAccessTokens(keys)
	valid := issue(t, a, now.Add(time.Minute))
	parts := strings.Split(valid, ".")
	sign := func(c jwt.Claims) string {
		s, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, c).SignedString(keys.private)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	hs256, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))},
		SessionID:        sessionID.String(),
	}).SignedString([]byte(keys.public)) // the public key as an HMAC secret: the classic confusion
	if err != nil {
		t.Fatal(err)
	}
	tampered := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(segment(t, valid, 1), userID.String(), sessionID.String(), 1)))
	tests := []struct{ name, token string }{
		{"another key", issue(t, NewAccessTokens(EphemeralKeys()), now.Add(time.Minute))},
		{"HS256", hs256},
		{"alg none", base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) + "." + parts[1] + "."},
		{"tampered payload", parts[0] + "." + tampered + "." + parts[2]},
		{"padded base64", parts[0] + "." + parts[1] + "=." + parts[2]},
		{"no exp", sign(claims{RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String()}, SessionID: sessionID.String()})},
		{"no sub", sign(claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}, SessionID: sessionID.String()})},
		{"no sid", sign(claims{RegisteredClaims: jwt.RegisteredClaims{Subject: userID.String(), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute))}})},
		{"a refresh token", "nrv_rt_" + strings.Repeat("A", 91)},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := a.Verify(tt.token, now); err == nil || errors.Is(err, app.ErrAccessTokenExpired) {
				t.Errorf("Verify() = %v, want invalid", err)
			}
		})
	}
}

func TestRefreshTokenMAC(t *testing.T) {
	keys := testKeys(t)
	m := NewRefreshTokenMAC(keys)
	msg := bytes.Repeat([]byte{7}, 52)
	tag := m.Tag(msg)

	if m.Tag(bytes.Clone(msg)) != tag {
		t.Error("the tag of the same message differs")
	}
	for i := range msg {
		changed := bytes.Clone(msg)
		changed[i] ^= 1
		if m.Tag(changed) == tag {
			t.Errorf("changing byte %d keeps the tag", i)
		}
	}
	if NewRefreshTokenMAC(EphemeralKeys()).Tag(msg) == tag {
		t.Error("another key gives the same tag")
	}
}
```

- [ ] **Step 4: 仓储**

`server/internal/modules/identity/adapter/postgres/store.go`：

```go
// Package postgresadapter is the identity module's repository adapter: sqlc
// queries (queries/, generated into gen/) over the transaction that the
// context carries, or the pool.
package postgresadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Store implements the identity repository ports of app.
type Store struct {
	pool *pgxpool.Pool
}

// New returns the store over pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// queries runs in the context's transaction when there is one.
func (s *Store) queries(ctx context.Context) *gen.Queries {
	return gen.New(postgres.DB(ctx, s.pool))
}

// uniqueViolation reports whether err broke the unique constraint name.
func uniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

// notFound turns pgx.ErrNoRows into app.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return app.ErrNotFound
	}
	return err
}
```

`server/internal/modules/identity/adapter/postgres/users.go`：

```go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// CreateUser inserts u. A taken address is domain.ErrEmailTaken. The domain
// validated every value, so a CHECK violation is a bug: an internal error
// (500), not a domain error.
func (s *Store) CreateUser(ctx context.Context, u app.NewUser) error {
	err := s.queries(ctx).CreateUser(ctx, gen.CreateUserParams{
		ID: u.ID, Email: u.Email, Password: u.PasswordHash, DisplayName: u.DisplayName, Now: u.Now,
	})
	switch {
	case uniqueViolation(err, "users_email_key"):
		return domain.ErrEmailTaken
	case err != nil:
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUser reads account id; app.ErrNotFound when there is none.
func (s *Store) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := s.queries(ctx).GetUser(ctx, id)
	if err != nil {
		return domain.User{}, notFound(err)
	}
	return domain.User{
		ID:          row.ID,
		Email:       row.Email,
		FirstName:   row.FirstName,
		LastName:    row.LastName,
		DisplayName: row.DisplayName,
		Timezone:    row.UserTimezone,
		CreatedAt:   row.CreatedAt,
	}, nil
}

// CreateDefaultProfile inserts the profile of a new account with Plane's
// model defaults.
func (s *Store) CreateDefaultProfile(ctx context.Context, id, userID uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).CreateProfile(ctx, gen.CreateProfileParams{ID: id, UserID: userID, Now: now}); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}
```

`server/internal/modules/identity/adapter/postgres/sessions.go`：

```go
package postgresadapter

import (
	"context"
	"fmt"
	"net/netip"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// CreateSession inserts a login at generation 0. An unknown client IP is
// stored as NULL.
func (s *Store) CreateSession(ctx context.Context, n app.NewSession) error {
	var ip *netip.Addr
	if n.IP.IsValid() {
		ip = &n.IP
	}
	err := s.queries(ctx).CreateSession(ctx, gen.CreateSessionParams{
		ID: n.ID, UserID: n.UserID, TokenHash: n.TokenHash, UserAgent: n.UserAgent, Ip: ip, ExpiresAt: n.ExpiresAt, Now: n.Now,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// SessionCredential reads what authentication checks of session id;
// app.ErrNotFound when there is none.
func (s *Store) SessionCredential(ctx context.Context, id uuid.UUID) (app.SessionCredential, error) {
	row, err := s.queries(ctx).GetSessionCredential(ctx, id)
	if err != nil {
		return app.SessionCredential{}, notFound(err)
	}
	return app.SessionCredential{
		UserID:     row.UserID,
		ExpiresAt:  row.ExpiresAt,
		Revoked:    row.RevokedAt != nil,
		UserActive: row.UserActive,
	}, nil
}
```

`server/internal/modules/identity/adapter/postgres/store_test.go`：

```go
package postgresadapter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/netip"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 9, 25, 10, 0, 0, 123456789, time.UTC)).Now()

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func newUser(email string) app.NewUser {
	return app.NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5", DisplayName: domain.DisplayNameFromEmail(email), Now: now}
}

func mustCreate(t *testing.T, s *postgresadapter.Store, u app.NewUser) {
	t.Helper()
	if err := s.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndGetUser(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)

	got, err := s.GetUser(context.Background(), u.ID)

	want := domain.User{ID: u.ID, Email: u.Email, DisplayName: "alice", Timezone: "UTC", CreatedAt: now}
	if err != nil || got != want {
		t.Errorf("GetUser() = %+v, %v; want %+v", got, err, want)
	}
	if got.CreatedAt.Location() != time.UTC {
		t.Errorf("created_at in %v, want UTC", got.CreatedAt.Location())
	}
	var password string
	var active bool
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(), "SELECT password, is_active, created_at, updated_at FROM users WHERE id = $1", u.ID).
		Scan(&password, &active, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if password != u.PasswordHash || !active || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("row = %q active=%v created %v updated %v; want the hash, active, and both audit columns at the clock's %v", password, active, created, updated, now)
	}
}

func TestGetUnknownUser(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.GetUser(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetUser() = %v, want app.ErrNotFound", err)
	}
}

// The unique constraint turns a race between two registrations into 409.
func TestCreateUserWithATakenAddress(t *testing.T) {
	s, pool := newStore(t)
	mustCreate(t, s, newUser("alice@corp.com"))
	tx := postgres.NewTxManager(pool, 2*time.Second)

	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		return s.CreateUser(ctx, newUser("alice@corp.com"))
	})

	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("CreateUser() = %v, want identity.email_taken", err)
	}
}

// The domain validates every value first, so a CHECK violation is a bug
// that got past it: an internal error, never a domain error.
func TestCreateUserBreakingACheckIsInternal(t *testing.T) {
	s, _ := newStore(t)

	err := s.CreateUser(context.Background(), newUser("Alice@corp.com"))

	var se *shared.Error
	var pgErr *pgconn.PgError
	if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != "users_email_check" {
		t.Errorf("CreateUser() = %v, want the check_violation of users_email_check, not a domain error", err)
	}
}

func TestCreateDefaultProfile(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	profileID := uuid.NewV7()

	if err := s.CreateDefaultProfile(context.Background(), profileID, u.ID, now); err != nil {
		t.Fatal(err)
	}

	var userID uuid.UUID
	var theme, language string
	var defaultSteps bool
	var week int16
	var onboarded bool
	var created, updated time.Time
	err := pool.QueryRow(context.Background(), `SELECT user_id, theme, language, start_of_the_week, is_onboarded,
		onboarding_step = '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'::jsonb, created_at, updated_at
		FROM profiles WHERE id = $1`, profileID).Scan(&userID, &theme, &language, &week, &onboarded, &defaultSteps, &created, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if userID != u.ID || theme != "system" || language != "en" || week != 0 || onboarded || !defaultSteps ||
		!created.Equal(now) || !updated.Equal(now) {
		t.Errorf("profile = %v %s %s %d onboarded=%v default steps=%v %v %v", userID, theme, language, week, onboarded, defaultSteps, created, updated)
	}
}

func TestCreateSessionAndReadItsCredential(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	hash := sha256.Sum256([]byte("secret"))
	n := app.NewSession{
		ID: uuid.NewV7(), UserID: u.ID, TokenHash: hash[:], UserAgent: "agent/1",
		IP: netip.MustParseAddr("2001:db8::7"), ExpiresAt: now.Add(720 * time.Hour), Now: now,
	}

	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}

	var tokenHash []byte
	var generation int32
	var ua string
	var ip *netip.Addr
	var expires, created, updated time.Time
	err := pool.QueryRow(context.Background(), `SELECT token_hash, generation, user_agent, ip, expires_at, created_at, updated_at
		FROM auth_sessions WHERE id = $1`, n.ID).Scan(&tokenHash, &generation, &ua, &ip, &expires, &created, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tokenHash, hash[:]) || generation != 0 || ua != "agent/1" || ip == nil || *ip != n.IP ||
		!expires.Equal(n.ExpiresAt) || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("session row = % x g=%d %q %v %v %v %v", tokenHash, generation, ua, ip, expires, created, updated)
	}

	cred, err := s.SessionCredential(context.Background(), n.ID)
	want := app.SessionCredential{UserID: u.ID, ExpiresAt: n.ExpiresAt, Revoked: false, UserActive: true}
	if err != nil || cred != want {
		t.Errorf("SessionCredential() = %+v, %v; want %+v", cred, err, want)
	}

	if _, err := pool.Exec(context.Background(), "UPDATE auth_sessions SET revoked_at = now(), revoke_reason = 'logout'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE users SET is_active = false"); err != nil {
		t.Fatal(err)
	}
	if cred, err := s.SessionCredential(context.Background(), n.ID); err != nil || !cred.Revoked || cred.UserActive {
		t.Errorf("SessionCredential() after revoke and deactivate = %+v, %v", cred, err)
	}
}

func TestCreateSessionWithoutAnIP(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	hash := sha256.Sum256([]byte("secret"))
	n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: hash[:], ExpiresAt: now.Add(time.Hour), Now: now}

	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}

	var ip *netip.Addr
	if err := pool.QueryRow(context.Background(), "SELECT ip FROM auth_sessions WHERE id = $1", n.ID).Scan(&ip); err != nil || ip != nil {
		t.Errorf("ip = %v, %v; want NULL", ip, err)
	}
}

func TestUnknownSessionCredential(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.SessionCredential(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("SessionCredential() = %v, want app.ErrNotFound", err)
	}
}
```

- [ ] **Step 5: 认证器**

`server/internal/modules/identity/adapter/authn/authenticator.go`：

```go
// Package authn implements the platform's httpserver.Authenticator with the
// identity module's authentication use case (M2 design 3.6).
package authn

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AuthenticateUseCase is app.Authenticate.
type AuthenticateUseCase interface {
	Execute(ctx context.Context, token string) (shared.Actor, error)
}

// Authenticator puts the request's actor in the context.
type Authenticator struct {
	uc AuthenticateUseCase
}

// New returns the authenticator over uc.
func New(uc AuthenticateUseCase) *Authenticator {
	return &Authenticator{uc: uc}
}

// Authenticate returns a context carrying the actor of token and the
// caller's rate-limit key, session:<id> (used from M2/P2 on). An invalid
// token is the use case's 401 *shared.Error; any other error passes through
// as an internal fault.
func (a *Authenticator) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	actor, err := a.uc.Execute(ctx, token)
	if err != nil {
		return nil, "", err
	}
	return shared.WithActor(ctx, actor), "session:" + actor.SessionID.String(), nil
}
```

`server/internal/modules/identity/adapter/authn/authenticator_test.go`：

```go
package authn_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The platform's port, satisfied by structure.
var _ httpserver.Authenticator = (*authn.Authenticator)(nil)

type fakeUseCase struct {
	actor shared.Actor
	err   error
}

func (f fakeUseCase) Execute(context.Context, string) (shared.Actor, error) { return f.actor, f.err }

func TestAuthenticatePutsTheActorInTheContext(t *testing.T) {
	actor := shared.Actor{UserID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"), SessionID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")}

	ctx, key, err := authn.New(fakeUseCase{actor: actor}).Authenticate(context.Background(), "token")

	got, actorErr := shared.RequireActor(ctx)
	if err != nil || actorErr != nil || got != actor || key != "session:0199a2b4-0000-7000-8000-000000000002" {
		t.Errorf("Authenticate() = actor %+v (%v), key %q, %v", got, actorErr, key, err)
	}
}

func TestAuthenticatePassesErrorsThrough(t *testing.T) {
	invalid := shared.Unauthenticated()
	boom := errors.New("database is down")
	for _, want := range []error{invalid, boom} {
		ctx, key, err := authn.New(fakeUseCase{err: want}).Authenticate(context.Background(), "token")
		if !errors.Is(err, want) || ctx != nil || key != "" {
			t.Errorf("Authenticate() = %v, %q, %v; want nil, \"\", %v", ctx, key, err, want)
		}
	}
}
```

- [ ] **Step 6: 整理依赖**

Run: `go -C server mod tidy`

Run: `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`
Expected: 两个文件都是 `go 1.27` 和 `toolchain go1.27.1`。

Expected: `server/go.mod` 的第一个 `require` 块多出 `github.com/golang-jwt/jwt/v5 v5.3.1` 和 `golang.org/x/crypto v0.57.0`；间接依赖中删去 `golang.org/x/crypto v0.55.0`，`golang.org/x/text` 从 v0.41.0 抬到 v0.42.0（spec 第 3 节第 11 条）；其余不变。

Run: `go -C server test -count=1 ./internal/modules/identity/...`
Expected: `adapter/argon2`、`adapter/authn`、`adapter/postgres`、`adapter/signing`、`app`、`domain` 都是 `ok`。

Run: `go -C server test -run '^$' -bench BenchmarkHashDefaultParams -count=3 ./internal/modules/identity/adapter/argon2/`
Expected: 本机每次十几毫秒（原型 Apple M5 Max 上 13.9–14.2 毫秒、约 19 MiB）。这是记录，不是门槛；把结果写进本 Task 的报告。

- [ ] **Step 7: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`internal/archtest` 通过（`adapter/postgres/gen` 只被 `adapter/postgres` 导入；`TestNerveBinaryLinksNoBannedModule` 接受新依赖）。

- [ ] **Step 8: 提交**

```bash
git add server/internal/modules/identity/adapter server/go.mod server/go.sum
```
```bash
git commit -m "feat(M2/P1): identity adapters for argon2id, Ed25519 tokens, the store and the authenticator

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 适配器的 23 个测试（含基准）通过；`TestCreateAndGetUser` 证明审计列等于固定时钟；`TestAccessTokenVerifyRejects` 的 10 个情况通过；`go.mod` 的 `go` 和 `toolchain` 行不变。

---

### Task 10: 接口 `register` 与 `getMe`，identity 的 HTTP 适配器

**Files:**
- Create: `api/modules/identity.yaml`
- Modify: `api/openapi.yaml`（最终版本）
- Create: `server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml`
- Create: `server/internal/modules/identity/adapter/http/handler.go`、`handler_test.go`
- Modify: `server/go.mod`、`server/go.sum`
- Generate: `server/internal/modules/identity/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Consumes: `app.Register`、`app.GetMe`（Task 8，经两个小接口）；`httpserver.Router`、`API`、`RequestMetaFrom`（Task 4）；`apitest.Main`、`CheckRequest`、`CheckResponse`（Task 5）。
- Produces（spec 2.14）：
  - 契约：`register`（`POST /api/v0/auth/register`，`security: []`，码 `identity.signup_disabled`、`validation_failed`、`identity.email_taken`、`server_busy`，201 `AuthTokens`）；`getMe`（`GET /api/v0/me`，`security: [{bearer: []}]`，码 `[]`，200 `User`）；
  - `httpadapter.UseCases{Register, GetMe}`、`httpadapter.PublicOperations()`（`["POST /api/v0/auth/register"]`）、`httpadapter.Register(router, api, uc)`；
  - `User.avatar_url`、`cover_image_url` 生成为 `nullable.Nullable[string]`，handler 显式写 `null`。

**Tests:**（`handler_test.go`，假用例和假认证器，经真实的 `Router` 和 `API` 中间件）
- `TestMain`：`apitest.Main(m, "identity")`，四个声明的码都必须被答过。
- `TestRegisterAnswers201WithTheTokens`（请求过 `CheckRequest`；响应逐字：`access_token_expires_in` 900、`refresh_token_expires_at` 带微秒、`token_type` `Bearer`；用例收到原样的邮箱、UA 和 `203.0.113.7`）。
- `TestRegisterProblems`（4 个：403、422、409、503 带 `Retry-After: 1`）。
- `TestRegisterBodyProblems`（5 个：未知和缺少的字段一次返回、字符串传 `null`、不是 JSON、空请求体、413；用例都没有被调用，响应中没有 Go 的类型名）。
- `TestGetMe`（逐字核对 JSON，`avatar_url`、`cover_image_url` 是 `null`）；`TestGetMeWithoutAValidToken`（没有令牌、伪造的令牌，都是 401 `unauthorized`）。

- [ ] **Step 1: 契约**

`api/modules/identity.yaml`：

```yaml
# identity 模块的接口（M2 设计 5.1、5.2）。生成的 Go 代码在
# server/internal/modules/identity/adapter/http/gen（make gen-go）。
openapi: 3.1.0
info:
  title: Nerve identity API
  version: v0
paths:
  /api/v0/auth/register:
    post:
      operationId: register
      tags: [identity]
      summary: Create an account and sign in
      description: >-
        Creates an account with its default profile and signs it in: the
        response holds a new session's tokens. While sign-up is off, every
        request answers identity.signup_disabled before anything else is
        checked, whether the address is registered or not. The password needs
        8–128 characters with an upper-case letter, a lower-case letter, a
        digit and a special character, and must not be a common password.
      security: []
      x-problem-codes: [identity.signup_disabled, validation_failed, identity.email_taken, server_busy]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RegisterRequest'
      responses:
        '201':
          description: The account exists and is signed in.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuthTokens'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me:
    get:
      operationId: getMe
      tags: [identity]
      summary: Read the caller's account
      security: [{bearer: []}]
      x-problem-codes: []
      responses:
        '200':
          description: The caller's account.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        default:
          $ref: '#/components/responses/Problem'
components:
  # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  responses:
    Problem:
      description: Error (RFC 9457 problem details).
      content:
        application/problem+json:
          schema:
            $ref: '../common.yaml#/components/schemas/Problem'
  schemas:
    RegisterRequest:
      type: object
      additionalProperties: false
      required: [email, password]
      properties:
        email:
          description: The sign-in address; stored trimmed and in lower case.
          type: string
          format: email
        password:
          type: string
    AuthTokens:
      description: >-
        A session's tokens. Send access_token as "Authorization: Bearer"; when
        it expires, exchange refresh_token for a new pair (M2/P2).
      type: object
      additionalProperties: false
      required: [token_type, access_token, access_token_expires_in, refresh_token, refresh_token_expires_at]
      properties:
        token_type:
          type: string
          enum: [Bearer]
        access_token:
          type: string
        access_token_expires_in:
          description: Seconds from this response until the access token expires.
          type: integer
        refresh_token:
          description: An opaque nrv_rt_ token.
          type: string
        refresh_token_expires_at:
          description: When the session ends; refreshing never extends it.
          type: string
          format: date-time
    User:
      type: object
      additionalProperties: false
      required: [id, email, first_name, last_name, display_name, user_timezone, avatar_url, cover_image_url, created_at]
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        first_name:
          type: string
        last_name:
          type: string
        display_name:
          type: string
        user_timezone:
          description: An IANA time zone name.
          type: string
        avatar_url:
          description: Null until uploads arrive (M5).
          type: [string, 'null']
        cover_image_url:
          description: Null until uploads arrive (M5).
          type: [string, 'null']
        created_at:
          type: string
          format: date-time
```

`api/openapi.yaml`（最终版本：加上 identity 的 tag 和两个路径）：

```yaml
# Nerve 接口描述的入口：列出所有路径，指向各模块的描述文件。
# make gen-web 用 Redocly 把它打包成 dist/openapi.yaml。
openapi: 3.1.0
info:
  title: Nerve API
  version: v0
  description: >-
    The HTTP API of Nerve. The web app is one client of it; every account can
    use every endpoint the same way. Every operation lists the problem codes
    it can answer in x-problem-codes; the top-level x-problem-codes can come
    from every operation, and an operation that needs a bearer token can also
    answer unauthorized.
  license:
    name: AGPL-3.0-only
    identifier: AGPL-3.0-only
# 所有操作都可能返回的平台错误码，只写在这里（M2 设计 3.11）。rate_limited 随 M2/P2 的限流加入
x-problem-codes: [bad_request, payload_too_large, internal_error]
tags:
  - name: identity
    description: Accounts, sign-in and sessions.
  - name: instance
    description: What this instance runs.
paths:
  /api/v0/auth/register:
    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1register'
  /api/v0/me:
    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me'
  /api/v0/instance:
    $ref: 'modules/instance.yaml#/paths/~1api~1v0~1instance'
components:
  # 模块文件的 securitySchemes 在打包时被丢掉，操作引用的 scheme 写在这里（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
      description: >-
        An access token (JWT) from register, login or refresh, or a personal
        access token (nrv_pat_…).
```

`server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml`（照抄 instance 的模块模板，只改注释和输出路径）：

```yaml
# oapi-codegen 配置：api/modules/identity.yaml → server.gen.go。
# 由 make gen-go 在 server/ 下执行，路径相对于 server/。
package: gen
output: internal/modules/identity/adapter/http/gen/server.gen.go
generate:
  models: true
  std-http-server: true
  strict-server: true
compatibility:
  # 枚举常量总是带类型名前缀：以后别的枚举出现同名的值，也不会改掉已有常量的名字
  always-prefix-enum-values: true
# output-options 是每个模块照抄的模板（M2 设计 3.12），bodyshapegen 读同一份 type-mapping
output-options:
  name-normalizer: ToCamelCaseWithInitialisms
  # 可为空又可省略的字段生成 nullable.Nullable[T]：PATCH 能区分"没传"和"传 null"
  nullable-type: true
  type-mapping:
    string:
      formats:
        # 标准库的 uuid：默认的 openapi_types.UUID 会带进 github.com/google/uuid
        uuid:
          type: uuid.UUID
          import: uuid
        # 邮箱的格式由领域层校验（422）；默认的 openapi_types.Email 在解码时自己校验，绕过这一层
        email:
          type: string
import-mapping:
  # common.yaml 的组件生成在共享包 apigen 里，这里只引用
  ../common.yaml: github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen
```

- [ ] **Step 2: 生成，加入 `nullable`**

Run: `make gen`
Expected: 生成 identity 的两个文件；最终版本的生成物：

| 文件 | 行数 | SHA-256 |
|---|---|---|
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | 439 | `671bb7d5dc8b69e6d23015ffd3b8c6d09c6c4e612dfd56bc549cf192a1121547` |
| `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go` | 21 | `5af65bdbfb1f707ef7f0c1c67fd121c5cd5eb001e1174b6481db78e6c5ba35a8` |
| `api/dist/openapi.yaml` | 259 | `39f1119f308f01df14d3f98c58e4f80c90aaf161a1fbf40c4631027c0a0c8b62` |
| `web/packages/api-client/src/schema.gen.ts` | 237 | `d3046f6d792adf2a49269e395bb7fe0b94b090a3db3b00c9290e8e2ed955292b` |

`server.gen.go` 导入 `github.com/oapi-codegen/nullable`，`uuid` 字段是标准库的 `uuid.UUID`，`email` 字段是 `string`（没有 `openapi_types`，也没有 `github.com/google/uuid`）：

Run: `grep -n "google/uuid\|openapi_types" server/internal/modules/identity/adapter/http/gen/server.gen.go`
Expected: 没有输出。

Run: `go -C server get github.com/oapi-codegen/nullable@v1.2.0`

- [ ] **Step 3: HTTP 适配器**

`server/internal/modules/identity/adapter/http/handler.go`：

```go
// Package httpadapter serves the identity module's API: it implements the
// strict server that oapi-codegen generates from api/modules/identity.yaml
// into the gen package, and only translates between the generated types and
// the use cases.
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// RegisterUseCase is app.Register.
type RegisterUseCase interface {
	Execute(ctx context.Context, in app.RegisterInput) (app.Tokens, error)
}

// GetMeUseCase is app.GetMe.
type GetMeUseCase interface {
	Execute(ctx context.Context) (domain.User, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	Register RegisterUseCase
	GetMe    GetMeUseCase
}

// PublicOperations are the module's routes that need no token (M2 design
// 3.6), as the generated code registers them.
func PublicOperations() []string {
	return []string{"POST /api/v0/auth/register"}
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}

// Register serves POST /api/v0/auth/register.
func (h handler) Register(ctx context.Context, req gen.RegisterRequestObject) (gen.RegisterResponseObject, error) {
	meta := httpserver.RequestMetaFrom(ctx)
	tokens, err := h.uc.Register.Execute(ctx, app.RegisterInput{
		Email:     req.Body.Email,
		Password:  req.Body.Password,
		UserAgent: meta.UserAgent,
		IP:        meta.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	return gen.Register201JSONResponse(authTokens(tokens)), nil
}

// GetMe serves GET /api/v0/me.
func (h handler) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	u, err := h.uc.GetMe.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetMe200JSONResponse{
		ID:           u.ID,
		Email:        u.Email,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		DisplayName:  u.DisplayName,
		UserTimezone: u.Timezone,
		// Required and null until M5. The zero Nullable is "unspecified" and
		// would marshal as "": set null explicitly.
		AvatarURL:     nullable.NewNullNullable[string](),
		CoverImageURL: nullable.NewNullNullable[string](),
		CreatedAt:     u.CreatedAt,
	}, nil
}

func authTokens(t app.Tokens) gen.AuthTokens {
	return gen.AuthTokens{
		TokenType:             gen.AuthTokensTokenTypeBearer,
		AccessToken:           t.AccessToken,
		AccessTokenExpiresIn:  int(t.AccessExpiresIn.Seconds()),
		RefreshToken:          t.RefreshToken,
		RefreshTokenExpiresAt: t.RefreshExpiresAt,
	}
}
```

`server/internal/modules/identity/adapter/http/handler_test.go`：

```go
package httpadapter_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11).
func TestMain(m *testing.M) { apitest.Main(m, "identity") }

var (
	userID    = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	sessionID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	created   = time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC)
)

type fakeRegister struct {
	got    app.RegisterInput
	tokens app.Tokens
	err    error
}

func (f *fakeRegister) Execute(_ context.Context, in app.RegisterInput) (app.Tokens, error) {
	f.got = in
	return f.tokens, f.err
}

type fakeGetMe struct{}

func (fakeGetMe) Execute(ctx context.Context) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: actor.UserID, Email: "alice@corp.com", DisplayName: "alice", Timezone: "UTC", CreatedAt: created}, nil
}

// fakeAuth accepts the token "valid" as the account userID.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	if token != "valid" {
		return nil, "", shared.Unauthenticated()
	}
	return shared.WithActor(ctx, shared.Actor{UserID: userID, SessionID: sessionID}), "session:" + sessionID.String(), nil
}

func newServer(register *fakeRegister) http.Handler {
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	api := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		Authenticator:    fakeAuth{},
		PublicOperations: httpadapter.PublicOperations(),
		MaxBodyBytes:     1024,
		RequestTimeout:   5 * time.Second,
	})
	httpadapter.Register(router, api, httpadapter.UseCases{Register: register, GetMe: fakeGetMe{}})
	return router
}

func do(t *testing.T, h http.Handler, req *http.Request) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func registerRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v0/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "agent/1")
	req.RemoteAddr = "203.0.113.7:5555"
	return req
}

func TestRegisterAnswers201WithTheTokens(t *testing.T) {
	register := &fakeRegister{tokens: app.Tokens{
		AccessToken: "access", AccessExpiresIn: 15 * time.Minute, RefreshToken: "nrv_rt_x", RefreshExpiresAt: created.Add(720 * time.Hour),
	}}
	req := registerRequest(`{"email":"Alice@Corp.com","password":"Tr0ub4dor&3"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(register), req)

	want := `{"access_token":"access","access_token_expires_in":900,"refresh_token":"nrv_rt_x",` +
		`"refresh_token_expires_at":"2026-10-25T10:00:00.123456Z","token_type":"Bearer"}` + "\n"
	if res.StatusCode != http.StatusCreated || body != want {
		t.Errorf("POST /auth/register = %d %s, want 201 %s", res.StatusCode, body, want)
	}
	wantIn := app.RegisterInput{Email: "Alice@Corp.com", Password: "Tr0ub4dor&3", UserAgent: "agent/1", IP: netip.MustParseAddr("203.0.113.7")}
	if register.got != wantIn {
		t.Errorf("use case got %+v, want %+v", register.got, wantIn)
	}
}

// The handler exit: every error the use case returns becomes its problem.
func TestRegisterProblems(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		retryAfter string
	}{
		{"sign-up off", domain.ErrSignupDisabled, 403, "identity.signup_disabled", ""},
		{"invalid values", shared.Invalid(shared.FieldError{Field: "password", Code: shared.FieldCommonPassword, Message: "is too common"}), 422, "validation_failed", ""},
		{"address taken", domain.ErrEmailTaken, 409, "identity.email_taken", ""},
		{"hashing saturated", shared.ServerBusy(time.Second), 503, "server_busy", "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(&fakeRegister{err: tt.err}), registerRequest(`{"email":"a@b.co","password":"x"}`))

			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) || res.Header.Get("Retry-After") != tt.retryAfter {
				t.Errorf("response = %d %s Retry-After %q, want %d %s", res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.code)
			}
		})
	}
}

// The body decoding exit (M0-P3 handoff 2): the structure check answers
// with fields, anything else with a generic detail, never a Go type name.
func TestRegisterBodyProblems(t *testing.T) {
	tests := []struct {
		name, body string
		status     int
		want       string
	}{
		{"unknown and missing fields", `{"email":"a@b.co","extra":1}`, 400,
			`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"password","code":"required","message":"is required"}]`},
		{"null for a string", `{"email":null,"password":"x"}`, 400, `"errors":[{"field":"email","code":"invalid_format"`},
		{"not JSON", `{"email":`, 400, `"detail":"The request body could not be decoded."`},
		{"empty", ``, 400, `"detail":"The request body could not be decoded."`},
		{"too large", `{"email":"` + strings.Repeat("a", 2000) + `"}`, 413, `"code":"payload_too_large"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			register := &fakeRegister{}
			res, body := do(t, newServer(register), registerRequest(tt.body))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) || strings.Contains(body, "Go struct") {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
			if register.got != (app.RegisterInput{}) {
				t.Errorf("the use case ran with %+v", register.got)
			}
		})
	}
}

func TestGetMe(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v0/me", nil)
	req.Header.Set("Authorization", "Bearer valid")
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(&fakeRegister{}), req)

	want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
		`"email":"alice@corp.com","first_name":"","id":"` + userID.String() + `","last_name":"","user_timezone":"UTC"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}

func TestGetMeWithoutAValidToken(t *testing.T) {
	for _, header := range []string{"", "Bearer forged"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v0/me", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}

		res, body := do(t, newServer(&fakeRegister{}), req)

		if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"unauthorized"`) {
			t.Errorf("GET /me with %q = %d %s, want 401 unauthorized", header, res.StatusCode, body)
		}
	}
}
```

Run: `go -C server mod tidy`

Run: `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`
Expected: 两个文件都是 `go 1.27` 和 `toolchain go1.27.1`。

Expected: `server/go.mod` 的第一个 `require` 块多出 `github.com/oapi-codegen/nullable v1.2.0`；这是 `server/go.mod`、`go.sum` 的最终版本：SHA-256 分别为 `04a34a7d7703f59c61dc1fe4eb33961ec84502cf06194c93ee9cb4adb252be9d`、`7918d856723b4c18dc85318e1daf98a97b7f1c0c6d42cb04dc2953e3d6428e62`。

Run: `go -C server test -count=1 ./internal/modules/identity/adapter/http/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http`

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`apitest` 的 `TestContractFollowsAuthoringRules`、`TestRootListsEveryModulePath` 用加上 identity 的契约通过。模块还没有接进程序（Task 11）。

Run: `make lint-web`
Expected: 通过（`schema.gen.ts` 多出 identity 的类型）。

Run: `make knip`
Expected: 通过。

- [ ] **Step 5: 提交**

```bash
git add api server/internal/modules/identity/adapter/http server/go.mod server/go.sum web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M2/P1): register and getMe in the contract, served by the identity HTTP adapter

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 四个生成物的 SHA-256 与上表相同；`apitest.Main` 对 identity 的覆盖核对通过；`TestGetMe` 逐字核对 `null`；`make gen-check` 干净。

---

### Task 11: 模块入口、`bootstrap` 接线与四个整程序测试

**Files:**
- Create: `server/internal/modules/identity/module.go`
- Modify: `server/internal/modules/instance/module.go`、`adapter/http/handler.go`、`handler_test.go`（最终版本）
- Create: `server/internal/platform/httpserver/apitest/operations.go`、`operations_test.go`
- Modify: `server/internal/bootstrap/app.go`、`app_test.go`、`commands.go`、`commands_test.go`（最终版本）
- Create: `server/internal/bootstrap/contract_test.go`、`auth_test.go`、`errors_test.go`

**Interfaces:**
- Consumes: Task 2–10 的全部产出。
- Produces（spec 2.14、2.15）：
  - `identity.New(Deps) (*Module, error)`，`Deps{Pool, Tx, Clock, Logger, SignupPolicy, SigningKeyPEM, AccessTokenTTL, SessionTTL, Password PasswordHashing}`；方法 `PublicOperations()`、`Authenticator()`、`Register(router, api)`；
  - `instance.New()` 的 `*Module` 有同样的 `PublicOperations()`、`Register(router, api)`（模块入口的统一形状）；
  - `apitest.Operation{Method, Path, Public, …}`，方法 `Pattern()`、`HasJSONBody()`、`Target()`、`BodyCases()`；`(*Contract).Operations()`；`BodyCase`、`FieldProblem`；
  - `bootstrap`：`newApp` 读签名密钥、接线两个模块和 `API`，`app.publicOperations` 是两个模块的并集；`signupSwitch`；`warnIfExposed`；`Serve` 的致命错误另记一条 ERROR。

**Tests:**
- `operations_test.go`：`TestOperations`（人造契约的三个操作：模式、是否公开、是否有请求体，按模式排序）；`TestTarget`（没有参数时是路径本身；路径级的 uuid 参数和必填的查询参数填上合法值：`/api/v0/things/00000000-0000-0000-0000-000000000000?limit=1&view=full`，可选的 `q` 不出现）；`TestBodyCases`（七种情况展开为 10 个请求体，逐字核对；"一次返回全部问题"期望 `name required`、`nerve_undeclared not_allowed`、`owner_id invalid_format`）。
- `contract_test.go`：`TestPublicOperationsAreTheContractsPublicOperations`；`TestAPIRoutesAreTheContractsOperations`；`TestOperationsThatNeedATokenAnswer401WithoutOne`；`TestBodiesThatBreakTheStructureAnswer400`（真实数据库；`register` 产生四个情况）。
- `auth_test.go`：`TestRegisterThenGetMe`（密钥文件、真实数据库；访问令牌确实由这把密钥签名）；`TestBadSigningKeyFileStopsTheApp`（文件不存在、不是 PKCS#8 的 PEM，错误原文只有键名和原因，没有路径）；`TestEphemeralSigningKeyIsAWarning`；`TestWarnIfExposed`（8 个地址）。
- `errors_test.go`：`TestEveryKindBecomesItsProblem`（8 个，逐字核对 JSON 和 `Retry-After`，包括包了一层的错误）。
- `commands_test.go`：`TestServeLogsAFatalError`（JSON 日志中有 `"level":"ERROR","msg":"nerve serve failed"` 和原因）。
- `app_test.go`：原有的 5 个测试改用 `127.0.0.1:0` 和 `server.addr_file`。
- instance 的 `handler_test.go`：经 `Router` 和 `API` 挂载，响应过 `CheckResponse`。

- [ ] **Step 1: 模块入口**

`server/internal/modules/identity/module.go`：

```go
// Package identity is the accounts module (M2 design 3.3, 6.2): accounts,
// profiles, sessions and, from later phases, personal access tokens. M2/P1
// brings registration, GET /me and the authentication every other operation
// goes through.
package identity

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Deps are what bootstrap builds for the module.
type Deps struct {
	Pool   *pgxpool.Pool
	Tx     shared.TxManager
	Clock  app.Clock
	Logger *slog.Logger
	// SignupPolicy is auth.signup_enabled (M2 decision 2).
	SignupPolicy app.SignupPolicy
	// SigningKeyPEM is the content of auth.jwt.private_key_file; nil for
	// none, then the key is ephemeral (dev and test only, M2 design 3.7).
	SigningKeyPEM  []byte
	AccessTokenTTL time.Duration
	SessionTTL     time.Duration
	Password       PasswordHashing
}

// PasswordHashing is auth.password: argon2id's parameters and the limits on
// concurrent hashes (M2 design 3.8).
type PasswordHashing struct {
	MemoryKiB     uint32
	Iterations    uint32
	Parallelism   uint8
	MaxConcurrent int
	MaxWait       time.Duration
}

// Module is the wired identity module.
type Module struct {
	uc            httpadapter.UseCases
	authenticator *authn.Authenticator
}

// New wires the module. A signing key that cannot be parsed is an error
// that never quotes the key.
func New(d Deps) (*Module, error) {
	keys, err := signingKeys(d)
	if err != nil {
		return nil, err
	}
	store := postgresadapter.New(d.Pool)
	tokens := signing.NewAccessTokens(keys)
	register := app.NewRegister(app.RegisterDeps{
		Policy:     d.SignupPolicy,
		Rules:      domain.NewPasswordRules(),
		Hasher:     argon2adapter.New(argon2adapter.Params(d.Password), d.Logger),
		Tx:         d.Tx,
		Users:      store,
		Profiles:   store,
		Sessions:   store,
		Tokens:     tokens,
		MAC:        signing.NewRefreshTokenMAC(keys),
		Clock:      d.Clock,
		Logger:     d.Logger,
		AccessTTL:  d.AccessTokenTTL,
		SessionTTL: d.SessionTTL,
	})
	return &Module{
		uc:            httpadapter.UseCases{Register: register, GetMe: app.NewGetMe(store)},
		authenticator: authn.New(app.NewAuthenticate(tokens, store, d.Clock)),
	}, nil
}

func signingKeys(d Deps) (*signing.Keys, error) {
	if d.SigningKeyPEM == nil {
		d.Logger.Warn("auth.jwt.private_key_file is not set: signing with an ephemeral key; " +
			"access tokens stop verifying at restart (dev and test only)")
		return signing.EphemeralKeys(), nil
	}
	keys, err := signing.ParseKeys(d.SigningKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return keys, nil
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// Authenticator checks the bearer token of every non-public operation.
func (m *Module) Authenticator() httpserver.Authenticator {
	return m.authenticator
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}
```

`server/internal/modules/instance/module.go`：

```go
// Package instance is the pilot module and the template for every module
// (M0 design 3.2): GET /api/v0/instance tells API clients what this nerve
// instance runs.
package instance

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Module is the wired instance module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module: GetInfo reads the build of the running binary.
func New() *Module {
	return &Module{uc: httpadapter.UseCases{GetInfo: app.NewGetInfo(buildinfo.Source{})}}
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// Register mounts the module's API on router, the root router from
// httpserver.NewRouter, behind api's per-route middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}
```

`server/internal/modules/instance/adapter/http/handler.go`：

```go
// Package httpadapter serves the instance module's API: it implements the
// strict server that oapi-codegen generates from api/modules/instance.yaml
// into the gen package.
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	GetInfo *app.GetInfo
}

// PublicOperations are the module's routes that need no token (M2 design
// 3.6), as the generated code registers them.
func PublicOperations() []string {
	return []string{"GET /api/v0/instance"}
}

// Register mounts the module's routes on router, the root router from
// httpserver.NewRouter, behind the platform's per-route middlewares. They
// are more specific than the platform's /api/ fallback, which keeps
// answering every other API path with a 404 problem. Binding, decoding and
// handler errors are answered as problem+json by api.Errors.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface: it only translates between
// the generated types and the use cases.
type handler struct {
	uc UseCases
}

// GetInstance serves GET /api/v0/instance.
func (h handler) GetInstance(context.Context, gen.GetInstanceRequestObject) (gen.GetInstanceResponseObject, error) {
	info := h.uc.GetInfo.Execute()
	return gen.GetInstance200JSONResponse{
		Product:    info.Product,
		Version:    info.Version,
		Commit:     info.Commit,
		APIVersion: gen.InstanceInfoAPIVersion(info.APIVersion),
	}, nil
}
```

`server/internal/modules/instance/adapter/http/handler_test.go`：

```go
package httpadapter_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

type fixedSource domain.Build

func (s fixedSource) Build() domain.Build { return domain.Build(s) }

func TestGetInstanceMatchesTheContract(t *testing.T) {
	contract := apitest.Load(t)
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	api := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		PublicOperations: httpadapter.PublicOperations(),
		MaxBodyBytes:     1 << 20,
		RequestTimeout:   time.Second,
	})
	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"})
	httpadapter.Register(router, api, httpadapter.UseCases{GetInfo: getInfo})
	req := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	res := rec.Result()

	contract.CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	want := `{"api_version":"v0","commit":"4f2a9c1","product":"Nerve","version":"1.2.3"}` + "\n"
	if res.StatusCode != http.StatusOK || string(body) != want {
		t.Errorf("GET /api/v0/instance = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}
```

- [ ] **Step 2: `apitest` 的操作列表和请求体样例**

`server/internal/platform/httpserver/apitest/operations.go`：

```go
package apitest

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Operation is one operation of the contract, as the whole-program tests of
// bootstrap see it (M2 design 3.6, 3.11).
type Operation struct {
	Method string // upper case
	Path   string
	Public bool // security: []
	params openapi3.Parameters
	body   *openapi3.Schema
}

// Pattern is the route pattern the generated code registers, e.g.
// "POST /api/v0/auth/register".
func (o Operation) Pattern() string { return o.Method + " " + o.Path }

// HasJSONBody reports whether the operation takes an application/json body.
func (o Operation) HasJSONBody() bool { return o.body != nil }

// Target is the request target of an example call: the path with each path
// parameter, and each required query parameter, set to a valid value.
// Parameters bind before the middlewares, so a wrong one would answer 400
// before anything else runs (M2 design 3.6).
func (o Operation) Target() string {
	path, query := o.Path, url.Values{}
	for _, ref := range o.params {
		p := ref.Value
		value := fmt.Sprint(validValue(p.Schema.Value))
		switch {
		case p.In == openapi3.ParameterInPath:
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(value))
		case p.In == openapi3.ParameterInQuery && p.Required:
			query.Set(p.Name, value)
		}
	}
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

// Operations lists every operation of the contract, sorted by pattern.
func (c *Contract) Operations() []Operation {
	var ops []Operation
	for path, item := range c.doc.Paths.Map() {
		for method, op := range item.Operations() {
			o := Operation{Method: strings.ToUpper(method), Path: path, Public: !needsToken(op),
				params: slices.Concat(item.Parameters, op.Parameters)}
			if rb := op.RequestBody; rb != nil && rb.Value != nil {
				if media := rb.Value.Content.Get("application/json"); media != nil && media.Schema != nil {
					o.body = media.Schema.Value
				}
			}
			ops = append(ops, o)
		}
	}
	slices.SortFunc(ops, func(a, b Operation) int { return strings.Compare(a.Pattern(), b.Pattern()) })
	return ops
}

// FieldProblem is one entry of a problem's errors: field and code.
type FieldProblem struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// BodyCase is a request body that breaks the operation's structure in one
// way, and what it must get: 400 bad_request with exactly Fields, in order;
// or, when Accepted, anything but 400.
type BodyCase struct {
	Name     string
	Body     []byte
	Fields   []FieldProblem
	Accepted bool
}

// unknownField is the property no schema declares.
const unknownField = "nerve_undeclared"

// BodyCases derives the cases of M2 design 3.11's fourth whole-program test
// from the operation's body schema: a valid body, broken one way at a time.
// Cases whose kind of field the schema lacks are left out.
//
//  1. an undeclared top-level property;
//  2. an undeclared property in a nested object;
//  3. null for an optional property that is not nullable;
//  4. each required property missing;
//  5. null for a nullable property: not 400;
//  6. each format property with a wrong string;
//  7. an undeclared property, a missing required property and a wrong
//     format together: every problem in one answer.
func (o Operation) BodyCases() []BodyCase {
	s := o.body
	valid := validValue(s).(map[string]any)
	with := func(change func(body map[string]any)) []byte {
		body := maps.Clone(valid)
		change(body)
		out, _ := json.Marshal(body)
		return out
	}
	var cases []BodyCase
	cases = append(cases, BodyCase{Name: "undeclared property", Body: with(func(b map[string]any) { b[unknownField] = 1 }),
		Fields: []FieldProblem{{unknownField, "not_allowed"}}})
	names := slices.Sorted(maps.Keys(s.Properties))
	for _, name := range names {
		p := s.Properties[name].Value
		if isObject(p) {
			nested := validValue(p).(map[string]any)
			nested[unknownField] = 1
			cases = append(cases, BodyCase{Name: "undeclared property in " + name, Body: with(func(b map[string]any) { b[name] = nested }),
				Fields: []FieldProblem{{name + "." + unknownField, "not_allowed"}}})
			break
		}
	}
	for _, name := range names {
		switch p := s.Properties[name].Value; {
		case isNullable(p):
			cases = append(cases, BodyCase{Name: "null for nullable " + name, Body: with(func(b map[string]any) { b[name] = nil }), Accepted: true})
		case !slices.Contains(s.Required, name):
			cases = append(cases, BodyCase{Name: "null for optional " + name, Body: with(func(b map[string]any) { b[name] = nil }),
				Fields: []FieldProblem{{name, "invalid_format"}}})
		}
	}
	for _, name := range slices.Sorted(slices.Values(s.Required)) {
		cases = append(cases, BodyCase{Name: "missing " + name, Body: with(func(b map[string]any) { delete(b, name) }),
			Fields: []FieldProblem{{name, "required"}}})
	}
	var formatted string
	for _, name := range names {
		if p := s.Properties[name].Value; checkedFormat(p) {
			formatted = name
			cases = append(cases, BodyCase{Name: "wrong " + p.Format + " in " + name, Body: with(func(b map[string]any) { b[name] = "not-a-" + p.Format }),
				Fields: []FieldProblem{{name, "invalid_format"}}})
		}
	}
	if len(s.Required) > 0 {
		missing := slices.Sorted(slices.Values(s.Required))[0]
		all := []FieldProblem{{unknownField, "not_allowed"}, {missing, "required"}}
		if formatted != "" && formatted != missing {
			all = append(all, FieldProblem{formatted, "invalid_format"})
		}
		slices.SortFunc(all, func(a, b FieldProblem) int { return strings.Compare(a.Field, b.Field) })
		cases = append(cases, BodyCase{Name: "every problem at once", Body: with(func(b map[string]any) {
			b[unknownField] = 1
			delete(b, missing)
			if formatted != "" && formatted != missing {
				b[formatted] = "not-a-format"
			}
		}), Fields: all})
	}
	return cases
}

// validValue is a value that s's structure accepts: every required
// property, the first enum value, a valid string for each checked format.
func validValue(s *openapi3.Schema) any {
	if len(s.AnyOf) > 0 {
		for _, alt := range s.AnyOf {
			if !alt.Value.Type.Is("null") {
				return validValue(alt.Value)
			}
		}
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}
	switch {
	case isObject(s):
		obj := map[string]any{}
		for _, name := range s.Required {
			obj[name] = validValue(s.Properties[name].Value)
		}
		return obj
	case s.Type.Includes("array"):
		return []any{}
	case s.Type.Includes("integer"), s.Type.Includes("number"):
		return 1
	case s.Type.Includes("boolean"):
		return false
	}
	switch s.Format {
	case "date-time":
		return "2026-01-01T00:00:00Z"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "email":
		return "someone@example.com"
	}
	return "x"
}

func isObject(s *openapi3.Schema) bool {
	return s.Type.Includes("object") && len(s.AnyOf) == 0
}

func isNullable(s *openapi3.Schema) bool {
	if s.Type.IncludesNull() {
		return true
	}
	for _, alt := range s.AnyOf {
		if alt.Value.Type.Is("null") {
			return true
		}
	}
	return false
}

// checkedFormat reports a string format that bodyshape checks: those the
// module template generates as Go types (M2 design 3.12).
func checkedFormat(s *openapi3.Schema) bool {
	return s.Type.Includes("string") && (s.Format == "date-time" || s.Format == "uuid")
}
```

`server/internal/platform/httpserver/apitest/operations_test.go`：

```go
package apitest

import (
	"slices"
	"testing"
)

const bodiesContract = `
openapi: 3.1.0
info: {title: bodies, version: v0}
x-problem-codes: [bad_request]
paths:
  /api/v0/things:
    post:
      operationId: createThing
      security: [{bearer: []}]
      x-problem-codes: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [name, owner_id]
              properties:
                name: {type: string}
                owner_id: {type: string, format: uuid}
                note: {type: [string, 'null']}
                count: {type: integer}
                kind: {type: string, enum: [a, b]}
                settings:
                  type: object
                  additionalProperties: false
                  properties:
                    notify: {type: boolean}
      responses:
        '204': {description: created}
    get:
      operationId: listThings
      security: []
      x-problem-codes: []
      responses:
        '204': {description: none}
  /api/v0/things/{thing_id}:
    parameters:
      - {name: thing_id, in: path, required: true, schema: {type: string, format: uuid}}
    get:
      operationId: getThing
      security: [{bearer: []}]
      x-problem-codes: []
      parameters:
        - {name: limit, in: query, required: true, schema: {type: integer}}
        - {name: view, in: query, required: true, schema: {type: string, enum: [full, short]}}
        - {name: q, in: query, schema: {type: string}}
      responses:
        '204': {description: none}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
`

func TestOperations(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	if len(ops) != 3 || ops[0].Pattern() != "GET /api/v0/things" || !ops[0].Public || ops[0].HasJSONBody() ||
		ops[1].Pattern() != "GET /api/v0/things/{thing_id}" || ops[1].Public || ops[1].HasJSONBody() ||
		ops[2].Pattern() != "POST /api/v0/things" || ops[2].Public || !ops[2].HasJSONBody() {
		t.Errorf("Operations() = %+v", ops)
	}
}

func TestTarget(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	if got := ops[0].Target(); got != "/api/v0/things" {
		t.Errorf("Target() without parameters = %q", got)
	}
	// The path-level parameter is filled; only the required query parameters
	// are set.
	if got, want := ops[1].Target(), "/api/v0/things/00000000-0000-0000-0000-000000000000?limit=1&view=full"; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

func TestBodyCases(t *testing.T) {
	post := contractFrom(t, bodiesContract).Operations()[2]

	var got []string
	for _, c := range post.BodyCases() {
		got = append(got, c.Name+" "+string(c.Body))
		if c.Accepted != (len(c.Fields) == 0) {
			t.Errorf("case %s: accepted %v with fields %v", c.Name, c.Accepted, c.Fields)
		}
	}
	const owner = `"owner_id":"00000000-0000-0000-0000-000000000000"`
	valid := `"name":"x",` + owner
	want := []string{
		`undeclared property {"name":"x","nerve_undeclared":1,` + owner + `}`,
		`undeclared property in settings {` + valid + `,"settings":{"nerve_undeclared":1}}`,
		`null for optional count {"count":null,` + valid + `}`,
		`null for optional kind {"kind":null,` + valid + `}`,
		`null for nullable note {"name":"x","note":null,` + owner + `}`,
		`null for optional settings {` + valid + `,"settings":null}`,
		`missing name {"owner_id":"00000000-0000-0000-0000-000000000000"}`,
		`missing owner_id {"name":"x"}`,
		`wrong uuid in owner_id {"name":"x","owner_id":"not-a-uuid"}`,
		`every problem at once {"nerve_undeclared":1,"owner_id":"not-a-format"}`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("BodyCases() =\n%q\nwant\n%q", got, want)
	}
	last := post.BodyCases()[len(want)-1]
	if wantFields := []FieldProblem{{"name", "required"}, {"nerve_undeclared", "not_allowed"}, {"owner_id", "invalid_format"}}; !slices.Equal(last.Fields, wantFields) {
		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
	}
}
```

Run: `go -C server test -count=1 ./internal/platform/httpserver/apitest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest`

- [ ] **Step 3: 组合根**

`server/internal/bootstrap/app.go`：

```go
// Package bootstrap is nerve's only composition root: it builds every adapter
// from the configuration, wires them together and runs the commands.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The platform and the shared kernel meet here by structure: neither imports
// the other (M2 design 3.3, 3.11).
var (
	_ shared.TxManager        = (*postgres.TxManager)(nil)
	_ httpserver.ProblemError = (*shared.Error)(nil)
)

// app is a fully wired nerve server.
type app struct {
	cfg      config.Config
	logger   *slog.Logger
	pool     *pgxpool.Pool
	migrator *postgres.Migrator
	router   *httpserver.Router
	// publicOperations are the routes that need no token: the union of what
	// the modules declare, checked against the contract by a test.
	publicOperations []string
}

// newApp wires the server described by cfg around the given migrations and
// built web frontend. close releases it.
func newApp(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles, webFiles fs.FS) (*app, error) {
	signingKey, err := readSigningKey(cfg.Auth.JWT.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	// The configuration log masks database.url as a whole; say where the pool
	// connects from pgx's own parse of it, which holds no password.
	target := pool.Config().ConnConfig
	logger.InfoContext(ctx, "database pool created",
		slog.String("host", target.Host), slog.Int("port", int(target.Port)),
		slog.String("database", target.Database), slog.String("user", target.User))
	migrator, err := postgres.NewMigrator(pool, migrationFiles)
	if err != nil {
		pool.Close()
		return nil, err
	}
	a := &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator}

	ident, err := identity.New(identity.Deps{
		Pool:           pool,
		Tx:             postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
		Clock:          clock.System{},
		Logger:         logger,
		SignupPolicy:   signupSwitch(cfg.Auth.SignupEnabled),
		SigningKeyPEM:  signingKey,
		AccessTokenTTL: cfg.Auth.AccessTokenTTL,
		SessionTTL:     cfg.Auth.SessionTTL,
		Password: identity.PasswordHashing{
			MemoryKiB:     cfg.Auth.Password.Argon2MemoryKiB,
			Iterations:    cfg.Auth.Password.Argon2Iterations,
			Parallelism:   cfg.Auth.Password.Argon2Parallelism,
			MaxConcurrent: cfg.Auth.Password.MaxConcurrentHashes,
			MaxWait:       cfg.Auth.Password.MaxWait,
		},
	})
	if err != nil {
		a.close()
		return nil, err
	}
	inst := instance.New()

	a.router = httpserver.NewRouter(logger,
		httpserver.Check{Name: "database", Run: pool.Ping},
		httpserver.Check{Name: "migrations", Run: migrator.CheckUpToDate},
	)
	a.publicOperations = slices.Concat(ident.PublicOperations(), inst.PublicOperations())
	api := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		Authenticator:    ident.Authenticator(),
		PublicOperations: a.publicOperations,
		MaxBodyBytes:     cfg.Server.MaxBodyBytes,
		RequestTimeout:   cfg.Server.RequestTimeout,
	})
	// Modules mount their generated routes on this root router, next to the
	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
	ident.Register(a.router, api)
	inst.Register(a.router, api)
	// The web UI takes every path no other pattern claims. It must be the
	// method-less "/": "GET /" and the method-less "/api/" would conflict.
	a.router.Handle("/", webui.Handler(webFiles))
	return a, nil
}

// readSigningKey returns the content of auth.jwt.private_key_file, or nil
// when it is not set. Errors name the key, never the path (M0-P2 handoff 4).
func readSigningKey(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return data, nil
}

// signupSwitch is auth.signup_enabled as identity's SignupPolicy.
type signupSwitch bool

func (s signupSwitch) AllowSignup(context.Context) (bool, error) { return bool(s), nil }

// warnIfExposed warns once when a non-prod nerve listens beyond loopback:
// most likely a deployment without NERVE_ENV=prod (M2 design 6.1).
func warnIfExposed(ctx context.Context, logger *slog.Logger, cfg config.Config) {
	if cfg.Env == config.EnvProd || loopback(cfg.Server.Addr) {
		return
	}
	logger.WarnContext(ctx, "not running as prod but listening beyond this machine: sign-up is open by default, "+
		"and without auth.jwt.private_key_file the signing key is ephemeral; set NERVE_ENV=prod to deploy",
		slog.String("env", cfg.Env), slog.String("addr", cfg.Server.Addr))
}

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

// run applies pending migrations when database.auto_migrate is on, then
// serves HTTP on server.addr until ctx is done (see httpserver.Server.Serve).
func (a *app) run(ctx context.Context) error {
	warnIfExposed(ctx, a.logger, a.cfg)
	if a.cfg.Database.AutoMigrate {
		applied, err := a.migrator.Up(ctx)
		// Log what was applied even when a later migration failed.
		for _, m := range applied {
			a.logger.InfoContext(ctx, "migration applied", slog.Int64("version", m.Version), slog.String("source", m.Source))
		}
		if err != nil {
			return err
		}
	}
	return httpserver.NewServer(a.cfg.Server, a.router, a.logger).ListenAndServe(ctx)
}

// close releases the database resources. Call it after run has returned.
func (a *app) close() {
	if err := a.migrator.Close(); err != nil {
		a.logger.Warn("close migrator", slog.Any("error", err))
	}
	a.pool.Close()
}
```

`server/internal/bootstrap/commands.go`：

```go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"text/tabwriter"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/logging"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Serve implements `nerve serve`: it logs the effective configuration (secrets
// masked) to logOut, applies pending migrations when database.auto_migrate is
// on, and serves the API and the embedded web frontend until ctx is done.
// Once the logger exists, a fatal error is also logged, as a structured
// record (M0-P2 handoff 8).
func Serve(ctx context.Context, cfg config.Config, logOut io.Writer) (err error) {
	logger, err := logging.New(logOut, cfg.Log)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			logger.ErrorContext(ctx, "nerve serve failed", slog.Any("error", err))
		}
	}()
	logger.InfoContext(ctx, "configuration loaded", slog.Any("config", cfg))
	a, err := newApp(ctx, cfg, logger, migrations.FS(), webui.FS())
	if err != nil {
		return err
	}
	defer a.close()
	return a.run(ctx)
}

// MigrateUp implements `nerve migrate up`: it applies every pending migration
// and prints one "applied <file>" line each.
func MigrateUp(ctx context.Context, cfg config.Config, out io.Writer) error {
	return runMigration(ctx, cfg, migrations.FS(), out, migrateUp)
}

// MigrateDown implements `nerve migrate down`: it rolls back the latest
// migration and prints "rolled back <file>".
func MigrateDown(ctx context.Context, cfg config.Config, out io.Writer) error {
	return runMigration(ctx, cfg, migrations.FS(), out, migrateDown)
}

// MigrateStatus implements `nerve migrate status`: it prints a table of every
// migration and whether it has been applied.
func MigrateStatus(ctx context.Context, cfg config.Config, out io.Writer) error {
	return runMigration(ctx, cfg, migrations.FS(), out, migrateStatus)
}

type migrationCommand func(context.Context, *postgres.Migrator, io.Writer) error

func runMigration(ctx context.Context, cfg config.Config, files fs.FS, out io.Writer, run migrationCommand) (err error) {
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	m, err := postgres.NewMigrator(pool, files)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.Close()) }()
	return run(ctx, m, out)
}

func migrateUp(ctx context.Context, m *postgres.Migrator, out io.Writer) error {
	applied, err := m.Up(ctx)
	// Print what was applied even when a later migration failed.
	for _, mig := range applied {
		if writeErr := writeLine(out, "applied "+mig.Source); writeErr != nil {
			return errors.Join(err, writeErr)
		}
	}
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		return writeLine(out, "no pending migrations")
	}
	return nil
}

func migrateDown(ctx context.Context, m *postgres.Migrator, out io.Writer) error {
	rolledBack, err := m.Down(ctx)
	if err != nil {
		return err
	}
	if rolledBack == nil {
		return writeLine(out, "no applied migrations to roll back")
	}
	return writeLine(out, "rolled back "+rolledBack.Source)
}

func migrateStatus(ctx context.Context, m *postgres.Migrator, out io.Writer) error {
	statuses, err := m.Status(ctx)
	if err != nil {
		return err
	}
	if len(statuses) == 0 {
		return writeLine(out, "no migrations")
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	// tabwriter buffers; write errors are reported by Flush.
	_, _ = fmt.Fprintln(tw, "VERSION\tSTATE\tAPPLIED AT\tSOURCE")
	for _, s := range statuses {
		state, appliedAt := "pending", "-"
		if s.Applied {
			state, appliedAt = "applied", s.AppliedAt.UTC().Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", s.Version, state, appliedAt, s.Source)
	}
	return tw.Flush()
}

func writeLine(w io.Writer, line string) error {
	_, err := io.WriteString(w, line+"\n")
	return err
}
```

- [ ] **Step 4: 整程序测试**

`server/internal/bootstrap/app_test.go`：

```go
package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// sampleMigrations are two probe tables, for the tests of how the app applies
// and reports migrations, whatever the production set holds.
var sampleMigrations = fstest.MapFS{
	"00001_probe_create_widgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE widgets (id bigint);\n-- +goose Down\nDROP TABLE widgets;\n")},
	"00002_probe_create_gadgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE gadgets (id bigint);\n-- +goose Down\nDROP TABLE gadgets;\n")},
}

// testWebUI stands in for the built frontend, so tests do not depend on make build.
const testIndexHTML = "<!doctype html><title>Nerve test</title>"

var testWebUI = fstest.MapFS{"index.html": {Data: []byte(testIndexHTML)}}

const unreachableDB = "postgres://nobody@127.0.0.1:1/nowhere"

var client = &http.Client{Timeout: 5 * time.Second}

// testConfig listens on a port the system picks and reports it through
// server.addr_file. Password hashing is cheap: the tests hash many times.
func testConfig(t *testing.T, dbURL string, autoMigrate bool) config.Config {
	t.Helper()
	return config.Config{
		Env: config.EnvTest,
		Server: config.ServerConfig{
			Addr:              "127.0.0.1:0",
			AddrFile:          filepath.Join(t.TempDir(), "addr"),
			ReadHeaderTimeout: time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
			ShutdownTimeout:   5 * time.Second,
			RequestTimeout:    5 * time.Second,
			MaxBodyBytes:      1 << 20,
		},
		Database: config.DatabaseConfig{URL: dbURL, MaxConns: 4, AutoMigrate: autoMigrate, CommitTimeout: 2 * time.Second},
		Auth: config.AuthConfig{
			SignupEnabled:  true,
			AccessTokenTTL: 15 * time.Minute,
			SessionTTL:     720 * time.Hour,
			Password: config.PasswordConfig{
				Argon2MemoryKiB:     64,
				Argon2Iterations:    1,
				Argon2Parallelism:   1,
				MaxConcurrentHashes: 4,
				MaxWait:             2 * time.Second,
			},
		},
		Log: config.LogConfig{Level: "error", Format: "text"},
	}
}

// buildApp wires the app, serving testWebUI, and closes it when the test ends.
func buildApp(t *testing.T, cfg config.Config, migrations fs.FS) *app {
	t.Helper()
	a, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), migrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	t.Cleanup(a.close)
	return a
}

// startApp runs the app, serving testWebUI, until the test ends and returns
// its base URL once it answers /healthz. The cleanup checks that run shut
// down cleanly.
func startApp(t *testing.T, cfg config.Config, migrations fs.FS) string {
	t.Helper()
	a := buildApp(t, cfg, migrations)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.run(ctx) }()
	// Cleanups run last-in first-out: run stops before buildApp's close.
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run() = %v, want nil after cancel", err)
		}
	})

	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		select {
		case err := <-done:
			t.Fatalf("run() returned early: %v", err)
		default:
		}
		addr, err := os.ReadFile(cfg.Server.AddrFile)
		if err != nil {
			continue
		}
		base := "http://" + string(addr)
		if resp, err := client.Get(base + "/healthz"); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
	}
	t.Fatal("server did not become healthy")
	return ""
}

func getReadyz(t *testing.T, base string) (int, httpserver.Problem) {
	t.Helper()
	resp, err := client.Get(base + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var p httpserver.Problem
	if resp.StatusCode != http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			t.Fatalf("decode problem: %v", err)
		}
	}
	return resp.StatusCode, p
}

func TestReadyWhenAutoMigrateAppliedEverything(t *testing.T) {
	base := startApp(t, testConfig(t, pgtest.NewEmptyDatabase(t), true), sampleMigrations)

	if status, p := getReadyz(t, base); status != http.StatusOK {
		t.Errorf("GET /readyz = %d %+v, want 200", status, p)
	}
}

func TestNotReadyWithPendingMigrations(t *testing.T) {
	base := startApp(t, testConfig(t, pgtest.NewEmptyDatabase(t), false), sampleMigrations)

	status, p := getReadyz(t, base)
	if status != http.StatusServiceUnavailable || p.Code != httpserver.CodeNotReady || p.Detail != "migrations is not ready" {
		t.Errorf("GET /readyz = %d %+v, want 503 not_ready for migrations", status, p)
	}
}

func TestNotReadyWhenDatabaseIsUnavailable(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), sampleMigrations)

	status, p := getReadyz(t, base)
	if status != http.StatusServiceUnavailable || p.Code != httpserver.CodeNotReady || p.Detail != "database is not ready" {
		t.Errorf("GET /readyz = %d %+v, want 503 not_ready for database", status, p)
	}
}

func TestRunFailsWhenAutoMigrateFails(t *testing.T) {
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, true), slog.New(slog.DiscardHandler), sampleMigrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	defer a.close()
	// Bounded: should run succeed instead, it would serve until ctx is done.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.run(ctx); err == nil {
		t.Error("run() = nil, want the migration error")
	}
}

// database.url is masked as a whole in the configuration log, so newApp logs
// where the pool connects, as pgx parsed it, and never the password.
func TestNewAppLogsTheDatabaseTarget(t *testing.T) {
	var logs bytes.Buffer
	cfg := testConfig(t, "postgres://nobody:secret@127.0.0.1:1/nowhere?password=secret;more", false)
	a, err := newApp(context.Background(), cfg, slog.New(slog.NewTextHandler(&logs, nil)), sampleMigrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	defer a.close()

	out := logs.String()
	if strings.Contains(out, "secret") {
		t.Errorf("log output leaks the password: %s", out)
	}
	if !strings.Contains(out, `msg="database pool created" host=127.0.0.1 port=1 database=nowhere user=nobody`) {
		t.Errorf("log output lacks the database target: %s", out)
	}
}
```

`server/internal/bootstrap/commands_test.go`：

```go
package bootstrap

import (
	"bytes"
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func runCommand(t *testing.T, dbURL string, cmd migrationCommand) string {
	t.Helper()
	var out bytes.Buffer
	if err := runMigration(context.Background(), testConfig(t, dbURL, false), sampleMigrations, &out, cmd); err != nil {
		t.Fatalf("command error = %v", err)
	}
	return out.String()
}

func TestMigrateCommandsOutput(t *testing.T) {
	db := pgtest.NewEmptyDatabase(t)

	status := runCommand(t, db, migrateStatus)
	wantPending := "VERSION  STATE    APPLIED AT  SOURCE\n" +
		"1        pending  -           00001_probe_create_widgets.sql\n" +
		"2        pending  -           00002_probe_create_gadgets.sql\n"
	if status != wantPending {
		t.Errorf("status before up:\n%s\nwant:\n%s", status, wantPending)
	}

	if got, want := runCommand(t, db, migrateUp), "applied 00001_probe_create_widgets.sql\napplied 00002_probe_create_gadgets.sql\n"; got != want {
		t.Errorf("up = %q, want %q", got, want)
	}
	if got, want := runCommand(t, db, migrateUp), "no pending migrations\n"; got != want {
		t.Errorf("second up = %q, want %q", got, want)
	}

	applied := regexp.MustCompile(`^VERSION +STATE +APPLIED AT +SOURCE\n` +
		`1 +applied +\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ +00001_probe_create_widgets\.sql\n` +
		`2 +applied +\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ +00002_probe_create_gadgets\.sql\n$`)
	if status := runCommand(t, db, migrateStatus); !applied.MatchString(status) {
		t.Errorf("status after up:\n%s", status)
	}

	if got, want := runCommand(t, db, migrateDown), "rolled back 00002_probe_create_gadgets.sql\n"; got != want {
		t.Errorf("down = %q, want %q", got, want)
	}
	runCommand(t, db, migrateDown)
	if got, want := runCommand(t, db, migrateDown), "no applied migrations to roll back\n"; got != want {
		t.Errorf("down with nothing applied = %q, want %q", got, want)
	}
}

func TestMigrateUpReportsMigrationsAppliedBeforeAFailure(t *testing.T) {
	files := fstest.MapFS{
		"00001_probe_create_widgets.sql": sampleMigrations["00001_probe_create_widgets.sql"],
		"00002_probe_broken.sql":         {Data: []byte("-- +goose Up\nCREATE TABLE broken (;\n")},
	}
	var out bytes.Buffer

	err := runMigration(context.Background(), testConfig(t, pgtest.NewEmptyDatabase(t), false), files, &out, migrateUp)

	if err == nil {
		t.Error("migrate up error = nil, want the failing migration's error")
	}
	if got, want := out.String(), "applied 00001_probe_create_widgets.sql\n"; got != want {
		t.Errorf("migrate up output = %q, want %q", got, want)
	}
}

func TestMigrateCommandsWithoutMigrations(t *testing.T) {
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	tests := []struct {
		name string
		cmd  migrationCommand
		want string
	}{
		{"status", migrateStatus, "no migrations\n"},
		{"up", migrateUp, "no pending migrations\n"},
		{"down", migrateDown, "no applied migrations to roll back\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			// An empty set of its own: the production set is not empty from M2 on.
			if err := runMigration(context.Background(), cfg, fstest.MapFS{}, &out, tt.cmd); err != nil {
				t.Fatalf("error = %v", err)
			}
			if out.String() != tt.want {
				t.Errorf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestMigrateCommandReportsBadURL(t *testing.T) {
	err := MigrateStatus(context.Background(), testConfig(t, "postgres://nerve:secret@localhost:notaport/nerve", false), &bytes.Buffer{})
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Errorf("MigrateStatus() = %v, want a database.url error without the password", err)
	}
}

// Once the logger exists, a fatal error of serve is also a structured log
// record (M0-P2 handoff 8).
func TestServeLogsAFatalError(t *testing.T) {
	cfg := testConfig(t, unreachableDB, false)
	cfg.Log = config.LogConfig{Level: "error", Format: "json"}
	cfg.Auth.JWT.PrivateKeyFile = filepath.Join(t.TempDir(), "missing.pem")
	var logs bytes.Buffer

	err := Serve(context.Background(), cfg, &logs)

	want := `"level":"ERROR","msg":"nerve serve failed","error":"auth.jwt.private_key_file: no such file or directory"}`
	if err == nil || !strings.Contains(logs.String(), want) {
		t.Errorf("Serve() = %v, logs:\n%s\nwant a record ending %s", err, logs.String(), want)
	}
}
```

`server/internal/bootstrap/contract_test.go`：

```go
package bootstrap

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The whole-program tests (M2 design 3.6, 3.11): the wired app against the
// contract, operation by operation, so an operation added later is covered
// without a new test.

// 1. The union of the modules' public operations is the contract's
// security: [] operations.
func TestPublicOperationsAreTheContractsPublicOperations(t *testing.T) {
	contract := apitest.Load(t)
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	var want []string
	for _, op := range contract.Operations() {
		if op.Public {
			want = append(want, op.Pattern())
		}
	}
	if got := slices.Sorted(slices.Values(a.publicOperations)); !slices.Equal(got, want) {
		t.Errorf("the modules declare the public operations %q; the contract has security: [] on %q", got, want)
	}
}

// 2. The routes registered under /api/v0 are the contract's operations: a
// route the contract does not describe fails here.
func TestAPIRoutesAreTheContractsOperations(t *testing.T) {
	contract := apitest.Load(t)
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	var got []string
	for _, pattern := range a.router.Patterns() {
		path := pattern
		if _, rest, ok := strings.Cut(pattern, " "); ok {
			path = rest
		}
		if strings.HasPrefix(path, "/api/v0/") {
			got = append(got, pattern)
		}
	}
	slices.Sort(got)
	var want []string
	for _, op := range contract.Operations() {
		want = append(want, op.Pattern())
	}
	if !slices.Equal(got, want) {
		t.Errorf("routes under /api/v0 = %q, want the contract's operations %q", got, want)
	}
}

// 3. Every operation that is not public answers 401 without a token.
func TestOperationsThatNeedATokenAnswer401WithoutOne(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	for _, op := range contract.Operations() {
		if op.Public {
			continue
		}
		t.Run(op.Pattern(), func(t *testing.T) {
			req := newRequest(t, op.Method, base+op.Target(), "", nil)
			res, body := send(t, req)

			contract.CheckResponse(t, req, res)
			if res.StatusCode != http.StatusUnauthorized || problemCode(t, body) != "unauthorized" ||
				res.Header.Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("answer = %d %s WWW-Authenticate %q, want 401 unauthorized Bearer",
					res.StatusCode, body, res.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

// 4. Every operation with a JSON body answers a body that breaks its
// structure with 400 and every broken field, and lets null through where the
// schema allows it. Operations that need a token get a valid one, so the
// body check, not the authentication, answers.
func TestBodiesThatBreakTheStructureAnswer400(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
	token := registerAccount(t, contract, base, "body-cases@example.com").AccessToken

	for _, op := range contract.Operations() {
		if !op.HasJSONBody() {
			continue
		}
		for _, c := range op.BodyCases() {
			t.Run(op.Pattern()+"/"+c.Name, func(t *testing.T) {
				auth := token
				if op.Public {
					auth = ""
				}
				req := newRequest(t, op.Method, base+op.Target(), auth, c.Body)
				res, body := send(t, req)

				contract.CheckResponse(t, req, res)
				if c.Accepted {
					if res.StatusCode == http.StatusBadRequest {
						t.Errorf("%s = 400 %s, want it accepted", c.Body, body)
					}
					return
				}
				var p struct {
					Code   string                 `json:"code"`
					Errors []apitest.FieldProblem `json:"errors"`
				}
				if err := json.Unmarshal(body, &p); err != nil {
					t.Fatalf("decode %s: %v", body, err)
				}
				if res.StatusCode != http.StatusBadRequest || p.Code != "bad_request" || !slices.Equal(p.Errors, c.Fields) {
					t.Errorf("%s = %d %s, want 400 bad_request with %v", c.Body, res.StatusCode, body, c.Fields)
				}
			})
		}
	}
}

// authTokens is the AuthTokens answer.
type authTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// registerAccount signs up email through the app and returns its tokens.
func registerAccount(t *testing.T, contract *apitest.Contract, base, email string) authTokens {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+"/api/v0/auth/register", "",
		[]byte(`{"email":"`+email+`","password":"Tr0ub4dor&3"}`))
	contract.CheckRequest(t, req)
	res, body := send(t, req)
	contract.CheckResponse(t, req, res)
	var tokens authTokens
	if res.StatusCode != http.StatusCreated || json.Unmarshal(body, &tokens) != nil {
		t.Fatalf("register %s = %d %s, want 201 with tokens", email, res.StatusCode, body)
	}
	return tokens
}

// newRequest builds a request with an optional bearer token and JSON body.
func newRequest(t *testing.T, method, url, token string, body []byte) *http.Request {
	t.Helper()
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// send sends req and returns the response with its body read.
func send(t *testing.T, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	res.Body = io.NopCloser(bytes.NewReader(body))
	return res, body
}

func problemCode(t *testing.T, body []byte) string {
	t.Helper()
	var p httpserver.Problem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decode problem %s: %v", body, err)
	}
	return p.Code
}
```

`server/internal/bootstrap/auth_test.go`：

```go
package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// testKeyPEM is an Ed25519 key made by `openssl genpkey -algorithm ed25519`
// for these tests only.
const testKeyPEM = "-----BEGIN PRIVATE KEY-----\n" +
	"MC4CAQAwBQYDK2VwBCIEIGqen6oN2FFQjS+yPQPHLVBIW0B2O9faCmNwftOWxqyE\n" +
	"-----END PRIVATE KEY-----\n"

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Registration and GET /me through the wired app and a real database; the
// access token is signed with the key of auth.jwt.private_key_file.
func TestRegisterThenGetMe(t *testing.T) {
	contract := apitest.Load(t)
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	cfg.Auth.JWT.PrivateKeyFile = writeFile(t, testKeyPEM)
	base := startApp(t, cfg, migrations.FS())

	tokens := registerAccount(t, contract, base, "Alice@Example.com")
	req := newRequest(t, http.MethodGet, base+"/api/v0/me", tokens.AccessToken, nil)
	res, body := send(t, req)

	contract.CheckResponse(t, req, res)
	var me struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(body, &me); err != nil || res.StatusCode != http.StatusOK ||
		me.Email != "alice@example.com" || me.DisplayName != "alice" {
		t.Errorf("GET /me = %d %s, want 200 alice@example.com", res.StatusCode, body)
	}
	if !signedBy(t, tokens.AccessToken, testKeyPEM) {
		t.Error("the access token is not signed with auth.jwt.private_key_file")
	}
	if !strings.HasPrefix(tokens.RefreshToken, "nrv_rt_") {
		t.Errorf("refresh token %q lacks the nrv_rt_ prefix", tokens.RefreshToken)
	}
}

// signedBy reports whether the JWT's EdDSA signature verifies with the
// public half of the PKCS#8 key.
func signedBy(t *testing.T, token, keyPEM string) bool {
	t.Helper()
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	public := key.(ed25519.PrivateKey).Public().(ed25519.PublicKey)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q is not a JWS", token)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	return ed25519.Verify(public, []byte(parts[0]+"."+parts[1]), sig)
}

// A bad auth.jwt.private_key_file stops the app. The error names the key,
// never the file's path or content (M0-P2 handoff 4).
func TestBadSigningKeyFileStopsTheApp(t *testing.T) {
	const secret = "top-secret-content"
	missing := filepath.Join(t.TempDir(), "secret-dir", "missing.pem")
	tests := []struct {
		name, path, want string
	}{
		{"missing", missing, "auth.jwt.private_key_file: no such file or directory"},
		{"not a key", writeFile(t, secret), `auth.jwt.private_key_file: not a PEM "PRIVATE KEY" block (PKCS#8)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig(t, unreachableDB, false)
			cfg.Auth.JWT.PrivateKeyFile = tt.path

			_, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), fstest.MapFS{}, testWebUI)

			if err == nil || err.Error() != tt.want {
				t.Errorf("newApp() error = %v, want %q", err, tt.want)
			}
		})
	}
}

// Without auth.jwt.private_key_file the key is ephemeral, and the app says so.
func TestEphemeralSigningKeyIsAWarning(t *testing.T) {
	var logs bytes.Buffer
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, false),
		slog.New(slog.NewTextHandler(&logs, nil)), fstest.MapFS{}, testWebUI)
	if err != nil {
		t.Fatal(err)
	}
	defer a.close()

	if !strings.Contains(logs.String(), `level=WARN msg="auth.jwt.private_key_file is not set: signing with an ephemeral key;`) {
		t.Errorf("logs lack the ephemeral key warning:\n%s", logs.String())
	}
}

// Outside prod, listening beyond loopback is warned about once at startup
// (M2 design 6.1).
func TestWarnIfExposed(t *testing.T) {
	tests := []struct {
		env, addr string
		warn      bool
	}{
		{config.EnvDev, "127.0.0.1:8080", false},
		{config.EnvDev, "[::1]:8080", false},
		{config.EnvDev, "localhost:8080", false},
		{config.EnvDev, "0.0.0.0:8080", true},
		{config.EnvDev, ":8080", true},
		{config.EnvTest, "[::]:0", true},
		{config.EnvDev, "nerve.example.com:8080", true},
		{config.EnvProd, "0.0.0.0:8080", false},
	}
	for _, tt := range tests {
		var logs bytes.Buffer
		cfg := config.Config{Env: tt.env, Server: config.ServerConfig{Addr: tt.addr}}

		warnIfExposed(context.Background(), slog.New(slog.NewTextHandler(&logs, nil)), cfg)

		if got := strings.Contains(logs.String(), "level=WARN msg=\"not running as prod but listening beyond this machine"); got != tt.warn {
			t.Errorf("%s on %s: warned = %v, want %v\n%s", tt.env, tt.addr, got, tt.warn, logs.String())
		}
	}
}
```

`server/internal/bootstrap/errors_test.go`：

```go
package bootstrap

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// shared.Error reaches the platform only by structure (M2 design 3.11): each
// Kind becomes its status, with the error's code, detail, fields and
// Retry-After, even wrapped.
func TestEveryKindBecomesItsProblem(t *testing.T) {
	tests := []struct {
		err        error
		want       string
		retryAfter string
	}{
		{shared.Invalid(shared.FieldError{Field: "email", Code: shared.FieldRequired, Message: "is required"}),
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"email","code":"required","message":"is required"}]}`, ""},
		{shared.NewError(shared.KindBadRequest, "things.bad_cursor", "d"), `{"status":400,"code":"things.bad_cursor","title":"Bad Request","detail":"d"}`, ""},
		{shared.Unauthenticated(), `{"status":401,"code":"unauthorized","title":"Unauthorized","detail":"Authentication is required."}`, ""},
		{shared.NewError(shared.KindForbidden, "things.forbidden", "d"), `{"status":403,"code":"things.forbidden","title":"Forbidden","detail":"d"}`, ""},
		{shared.NewError(shared.KindNotFound, "things.missing", "d"), `{"status":404,"code":"things.missing","title":"Not Found","detail":"d"}`, ""},
		{shared.NewError(shared.KindConflict, "things.taken", "d"), `{"status":409,"code":"things.taken","title":"Conflict","detail":"d"}`, ""},
		{&shared.Error{Kind: shared.KindRateLimited, Code: "rate_limited", Detail: "d", RetryDelay: 1500 * time.Millisecond},
			`{"status":429,"code":"rate_limited","title":"Too Many Requests","detail":"d"}`, "2"},
		{fmt.Errorf("register: %w", shared.ServerBusy(time.Second)),
			`{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"The server is busy; retry shortly."}`, "1"},
	}
	errs := httpserver.NewAPIErrors(slog.New(slog.DiscardHandler))
	for _, tt := range tests {
		rec := httptest.NewRecorder()

		errs.Write(rec, httptest.NewRequest(http.MethodGet, "/api/v0/things", nil), tt.err)

		if got := rec.Body.String(); got != tt.want+"\n" || rec.Header().Get("Retry-After") != tt.retryAfter {
			t.Errorf("Write(%v) = %s Retry-After %q, want %s Retry-After %q", tt.err, got, rec.Header().Get("Retry-After"), tt.want, tt.retryAfter)
		}
	}
}
```

Run: `go -C server mod tidy`
Expected: `server/go.mod`、`go.sum` 不变（仍是 Task 10 给出的 SHA-256）。

Run: `go -C server test -count=1 ./internal/bootstrap/ ./cmd/...`
Expected: 都是 `ok`。

- [ ] **Step 5: 核对整程序测试确实有效（不提交）**

临时做两处修改，各自运行 `go -C server test -count=1 ./internal/bootstrap/`，看到失败后改回：

1. 在 `app.go` 中把 `slices.Concat(ident.PublicOperations(), inst.PublicOperations())` 改成 `slices.Concat(inst.PublicOperations())`：`TestRegisterThenGetMe`、`TestPublicOperationsAreTheContractsPublicOperations`、`TestBodiesThatBreakTheStructureAnswer400` 三个失败。
2. 在 `newApp` 的 `inst.Register(a.router, api)` 之后加一行 `a.router.Handle("GET /api/v0/debug", webui.Handler(webFiles))`：只有 `TestAPIRoutesAreTheContractsOperations` 失败。

改回之后 `app.go` 与 Step 3 的内容逐字相同；再运行一次，全部通过。

- [ ] **Step 6: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 7: 提交**

```bash
git add server/internal/modules/identity/module.go server/internal/modules/instance server/internal/platform/httpserver/apitest server/internal/bootstrap
```
```bash
git commit -m "feat(M2/P1): wire identity into nerve and check the program against the contract

Four whole-program tests compare the running program with every operation
of the contract: the public set, the routes, 401 without a token and 400
for bodies that break the structure.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 四个整程序测试、`TestRegisterThenGetMe`、`TestEveryKindBecomesItsProblem` 通过；Step 5 的两处变异各自使相应的测试失败；`make test` 通过。

---

### Task 12: 端到端：地址文件、每个 worker 的连接池、失败时的数据库快照；A1、A2 的接口版本

**Files:**
- Modify: `e2e/fixtures/server.ts`、`e2e/fixtures/db.ts`、`e2e/fixtures/test.ts`、`e2e/global-setup.ts`
- Create: `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`
- Create: `e2e/stories/identity/a1-sign-up.spec.ts`、`e2e/stories/identity/a2-sign-up-refused.spec.ts`

**Interfaces:**
- Consumes: `server.addr_file`（Task 4）；`register`、`getMe`（Task 10、11）；`@nerve/api-client` 的 `components` 类型（Task 10 生成）。
- Produces（spec 2.16）：
  - `startNerve(databaseUrl, logFile, env = {})`：在 `127.0.0.1:0` 上监听，经 `<日志名>.addr` 取得地址；M0 的空闲端口猜测删除；
  - `createDatabase(name, template?) → URL`；`openDatabase(name) → Database{name, url, query, dump, close}`（每个 worker 一个 `pg` 连接池）；
  - fixture：`db`、`nerve`、`baseURL`、`api`、`nerveWith(env)`，自动 fixture `databaseSnapshot`（测试失败时 `pg_dump` 到 `database.sql` 并作为附件）；
  - `auth.ts`：`password`、`emailFor(testInfo, label)`、`register(api, email, headers)`；`assert/identity.ts`：`expectRegistered`、`countIdentity`、`expectNothingAdded`。

**Tests:**
- A1（"A1 (API): a caller signs up and gets a session"）：201 的令牌、访问令牌立即能读 `/me`；数据库断言（邮箱已转小写、`$argon2id$`、默认资料逐列、会话的 id 与令牌一致、第 0 代、`token_hash`、UA、IP、`expires_at - created_at` 恰好 30 天、未撤销）。
- A2（"A2 (API): a refused sign-up answers why and adds nothing"）：大小写不同的已有地址 409；`weak_password`、`common_password`（`Password1!`、`Password1!~`）422；关闭注册的独立 nerve 对新地址和已有地址都 403；三张表的行数不变。
- S1–S4 照常通过（S1 改为经地址文件启动的 nerve）。

- [ ] **Step 1: fixture**

`e2e/fixtures/server.ts`：

```ts
import { execFile, spawn, type ChildProcess } from "node:child_process";
import { once } from "node:events";
import { closeSync, mkdirSync, openSync, readFileSync, rmSync } from "node:fs";
import path from "node:path";
import { setTimeout as sleep } from "node:timers/promises";
import { promisify } from "node:util";

/** The binary under test: make build compiles it with the web frontend embedded. */
const binary = path.resolve(import.meta.dirname, "../../bin/nerve");

/** nerve connects with this application_name, so its sessions show in pg_stat_activity. */
export const applicationName = "nerve";

const readyTimeoutMs = 30_000;
const stopTimeoutMs = 30_000;
const commandTimeoutMs = 60_000;
const pollIntervalMs = 100;

/**
 * The worker-fixture timeout for `nerve` in test.ts: Playwright's default
 * (30 s, shared by setup and teardown) can be shorter than readyTimeoutMs
 * alone, so the fixture's own timeouts — and the "(log: …)" errors they
 * produce — could never fire first.
 */
export const nerveFixtureTimeoutMs = readyTimeoutMs + stopTimeoutMs + 10_000;

/** A nerve serve process of this run. */
export interface Nerve {
  readonly baseURL: string;
  /** Sends SIGTERM and waits for nerve to exit with code 0. */
  stop(): Promise<void>;
}

/**
 * Runs a nerve command, such as migrate up, with the test configuration on
 * the database at databaseUrl. It rejects when the command exits non-zero,
 * and kills it after commandTimeoutMs: global setup runs it before any
 * Playwright timeout applies.
 */
export async function runNerve(args: string[], databaseUrl: string): Promise<{ stdout: string; stderr: string }> {
  return promisify(execFile)(binary, args, {
    env: nerveEnv(databaseUrl),
    timeout: commandTimeoutMs,
    killSignal: "SIGKILL",
  });
}

/**
 * Starts nerve serve with the test configuration, plus the variables of env
 * (e.g. NERVE_AUTH__SIGNUP_ENABLED=false for a nerve with sign-up off), and
 * waits until /readyz answers 200. nerve listens on a port the system picks
 * and writes its address to server.addr_file, next to logFile, which gets
 * its output.
 */
export async function startNerve(
  databaseUrl: string,
  logFile: string,
  env: Record<string, string> = {}
): Promise<Nerve> {
  mkdirSync(path.dirname(logFile), { recursive: true });
  const addrFile = logFile.replace(/\.log$/, "") + ".addr";
  rmSync(addrFile, { force: true });
  const log = openSync(logFile, "w");
  const child = spawn(binary, ["serve"], {
    env: {
      ...nerveEnv(databaseUrl),
      NERVE_SERVER__ADDR: "127.0.0.1:0",
      NERVE_SERVER__ADDR_FILE: addrFile,
      ...env,
    },
    stdio: ["ignore", log, log],
  });
  closeSync(log); // the child has its own copy
  // A spawn failure (ENOENT/EACCES) emits "error" instead of "exit"; capture it
  // so waitUntilReady can surface it through the same log-path error below.
  let spawnError: Error | undefined;
  child.once("error", (err) => {
    spawnError = err;
  });
  try {
    const baseURL = await waitUntilReady(child, addrFile, Date.now() + readyTimeoutMs, () => spawnError);
    return { baseURL, stop: () => stop(child, logFile) };
  } catch (err) {
    await kill(child);
    throw new Error(`nerve did not become ready (log: ${logFile})`, { cause: err });
  }
}

/** The test configuration on the given database; the caller's own NERVE_* variables are left out. */
function nerveEnv(databaseUrl: string): NodeJS.ProcessEnv {
  const url = new URL(databaseUrl);
  url.searchParams.set("application_name", applicationName);
  const inherited = Object.entries(process.env).filter(([name]) => !name.startsWith("NERVE_"));
  return { ...Object.fromEntries(inherited), NERVE_ENV: "test", NERVE_DATABASE__URL: url.toString() };
}

/** The address nerve wrote to addrFile; undefined until it has. nerve renames the file into place, so it is never partial. */
function readAddr(addrFile: string): string | undefined {
  try {
    return readFileSync(addrFile, "utf8");
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") {
      return undefined;
    }
    throw err;
  }
}

/**
 * Waits until nerve has written its address and /readyz there answers 200,
 * and returns the base URL; fails when nerve exits, fails to spawn, or the
 * deadline passes. Each request may only use the time left, so a server that
 * accepts the connection but never answers cannot hold the wait past the
 * deadline.
 */
async function waitUntilReady(
  child: ChildProcess,
  addrFile: string,
  deadline: number,
  spawnError: () => Error | undefined
): Promise<string> {
  const failure = spawnError();
  if (failure) {
    throw failure;
  }
  if (child.exitCode !== null) {
    throw new Error(`nerve exited with code ${child.exitCode}`);
  }
  const timeLeft = deadline - Date.now();
  if (timeLeft <= 0) {
    throw new Error(`nerve did not answer /readyz with 200 within ${readyTimeoutMs} ms`);
  }
  const addr = readAddr(addrFile);
  if (addr !== undefined) {
    const baseURL = `http://${addr}`;
    const ready = await fetch(`${baseURL}/readyz`, { signal: AbortSignal.timeout(timeLeft) }).then(
      (res) => res.ok,
      () => false // no answer before the deadline
    );
    if (ready) {
      return baseURL;
    }
  }
  await sleep(pollIntervalMs);
  return waitUntilReady(child, addrFile, deadline, spawnError);
}

/** Kills a nerve that never became ready and waits until it is gone. */
async function kill(child: ChildProcess): Promise<void> {
  // No pid: the spawn failed, so there is no process and no "exit" to wait for.
  if (child.pid === undefined || child.exitCode !== null || child.signalCode !== null) {
    return;
  }
  const exited = once(child, "exit");
  child.kill("SIGKILL");
  await exited;
}

async function stop(child: ChildProcess, logFile: string): Promise<void> {
  if (child.exitCode === null && child.signalCode === null) {
    const exited = once(child, "exit");
    child.kill("SIGTERM");
    const timer = setTimeout(() => child.kill("SIGKILL"), stopTimeoutMs);
    await exited;
    clearTimeout(timer);
  }
  if (child.exitCode !== 0) {
    throw new Error(`nerve did not exit cleanly: code ${child.exitCode}, signal ${child.signalCode} (log: ${logFile})`);
  }
}
```

`e2e/fixtures/db.ts`：

```ts
import { execFile } from "node:child_process";
import { writeFile } from "node:fs/promises";
import { promisify } from "node:util";

import { PostgreSqlContainer } from "@testcontainers/postgresql";
import { Client, Pool, escapeIdentifier, type QueryResultRow } from "pg";

/** The PostgreSQL image, the same as the development database (deploy/compose.dev.yaml). */
const image = "postgres:18.6";

/** The database global setup migrates once; every worker gets a copy of it. */
export const templateDatabase = "nerve_template";

/** Global setup hands the server's URL and its container to the workers in these variables. */
const serverUrlVariable = "NERVE_E2E_POSTGRES_URL";
const containerVariable = "NERVE_E2E_POSTGRES_CONTAINER";

/** A database of this run: nerve serves from it, stories assert on it. */
export interface Database {
  readonly name: string;
  readonly url: string;
  /** Runs one statement on this database, through its own pool, and returns the rows. */
  query<Row extends QueryResultRow>(sql: string, params?: unknown[]): Promise<Row[]>;
  /** Writes a plain SQL pg_dump of this database to file, for the artifacts of a failed test. */
  dump(file: string): Promise<void>;
  /** Closes the pool. */
  close(): Promise<void>;
}

/**
 * Starts the PostgreSQL server of this run and publishes its URL and
 * container to the workers. The testcontainers reaper (Ryuk) removes the
 * container if the run dies before stop is called.
 */
export async function startPostgres(): Promise<{ stop: () => Promise<void> }> {
  const container = await new PostgreSqlContainer(image).withDatabase("postgres").start();
  process.env[serverUrlVariable] = databaseUrl(container.getConnectionUri(), "postgres");
  process.env[containerVariable] = container.getId();
  return {
    stop: async () => {
      await container.stop();
    },
  };
}

// pg waits forever by default; a server that accepts connections but never
// answers must fail the query instead of hanging the run. pg_dump runs in
// the container and is killed after dumpTimeoutMs.
const connectTimeoutMs = 10_000;
const queryTimeoutMs = 30_000;
const dumpTimeoutMs = 60_000;

/**
 * Creates database name, as a copy of template when given, on the server
 * startPostgres started, and returns its URL.
 */
export async function createDatabase(name: string, template?: string): Promise<string> {
  const serverUrl = requireVariable(serverUrlVariable);
  const copy = template ? ` TEMPLATE ${escapeIdentifier(template)}` : "";
  const admin = new Client({
    connectionString: serverUrl,
    connectionTimeoutMillis: connectTimeoutMs,
    query_timeout: queryTimeoutMs,
  });
  await admin.connect();
  try {
    await admin.query(`CREATE DATABASE ${escapeIdentifier(name)}${copy}`);
  } finally {
    await admin.end();
  }
  return databaseUrl(serverUrl, name);
}

/** Opens a pool on database name of the server startPostgres started: a worker's own database. */
export function openDatabase(name: string): Database {
  const serverUrl = requireVariable(serverUrlVariable);
  const url = databaseUrl(serverUrl, name);
  const pool = new Pool({
    connectionString: url,
    max: 2,
    connectionTimeoutMillis: connectTimeoutMs,
    query_timeout: queryTimeoutMs,
  });
  return {
    name,
    url,
    query: async <Row extends QueryResultRow>(sql: string, params?: unknown[]) =>
      (await pool.query<Row>(sql, params)).rows,
    dump: (file) => dump(serverUrl, name, file),
    close: () => pool.end(),
  };
}

async function dump(serverUrl: string, name: string, file: string): Promise<void> {
  const user = decodeURIComponent(new URL(serverUrl).username);
  const { stdout } = await promisify(execFile)(
    "docker",
    ["exec", requireVariable(containerVariable), "pg_dump", "--username", user, "--no-owner", "--dbname", name],
    { timeout: dumpTimeoutMs, killSignal: "SIGKILL", maxBuffer: 256 * 1024 * 1024 }
  );
  await writeFile(file, stdout);
}

function requireVariable(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is not set: run the stories with playwright test (see global-setup.ts)`);
  }
  return value;
}

function databaseUrl(serverUrl: string, name: string): string {
  const url = new URL(serverUrl);
  url.pathname = `/${name}`;
  url.searchParams.set("sslmode", "disable");
  return url.toString();
}
```

`e2e/global-setup.ts`（只连接模板库执行迁移，不开连接池：模板库上有会话时 `CREATE DATABASE … TEMPLATE` 会失败）：

```ts
import { createDatabase, startPostgres, templateDatabase } from "./fixtures/db";
import { runNerve } from "./fixtures/server";

/**
 * Starts PostgreSQL once per run and migrates the template database with
 * bin/nerve migrate up; every worker copies the template (fixtures/test.ts).
 * The returned function is the global teardown.
 */
export default async function globalSetup(): Promise<() => Promise<void>> {
  const postgres = await startPostgres();
  await runNerve(["migrate", "up"], await createDatabase(templateDatabase));
  return postgres.stop;
}
```

`e2e/fixtures/test.ts`：

```ts
import path from "node:path";

import { test as base } from "@playwright/test";

import { createApi, type Api } from "./api";
import { createDatabase, openDatabase, templateDatabase, type Database } from "./db";
import { nerveFixtureTimeoutMs, startNerve, type Nerve } from "./server";

export { expect } from "@playwright/test";

interface WorkerFixtures {
  /** The worker's own database, a copy of the migrated template, with a pool for the assertions. */
  db: Database;
  /** The worker's own nerve serve, on that database. */
  nerve: Nerve;
}

interface TestFixtures {
  /** The typed API client for the worker's nerve. */
  api: Api;
  /**
   * Starts another nerve on the worker's database, with extra variables such
   * as NERVE_AUTH__SIGNUP_ENABLED=false; it stops when the test ends. Each
   * start adds the nerve fixture's budget to the test's timeout, so the
   * fixture's own timeouts fire first.
   */
  nerveWith: (env: Record<string, string>) => Promise<Nerve>;
  /** When the test fails, a pg_dump of the worker's database joins its trace, screenshot and nerve log. */
  databaseSnapshot: void;
}

/** Stories import test from here: every worker runs its own nerve on its own database. */
export const test = base.extend<TestFixtures, WorkerFixtures>({
  db: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads a fixture's dependencies from this pattern
    async ({}, use, workerInfo) => {
      const name = `e2e_w${workerInfo.workerIndex}`;
      await createDatabase(name, templateDatabase);
      const db = openDatabase(name);
      await use(db);
      await db.close();
    },
    { scope: "worker" },
  ],
  nerve: [
    async ({ db }, use, workerInfo) => {
      const logFile = path.join(workerInfo.project.outputDir, `nerve-w${workerInfo.workerIndex}.log`);
      const nerve = await startNerve(db.url, logFile);
      await use(nerve);
      await nerve.stop();
    },
    // Playwright's default worker-fixture budget (30 s, shared by setup and
    // teardown) is shorter than the fixture's own timeouts; give it room to
    // let those fire and report first.
    { scope: "worker", timeout: nerveFixtureTimeoutMs },
  ],
  // page and request resolve relative URLs against the worker's nerve.
  baseURL: async ({ nerve }, use) => {
    await use(nerve.baseURL);
  },
  api: async ({ nerve }, use) => {
    await use(createApi(nerve.baseURL));
  },
  nerveWith: async ({ db }, use, testInfo) => {
    const started: Nerve[] = [];
    await use(async (env) => {
      testInfo.setTimeout(testInfo.timeout + nerveFixtureTimeoutMs);
      const nerve = await startNerve(db.url, testInfo.outputPath(`nerve-${started.length + 1}.log`), env);
      started.push(nerve);
      return nerve;
    });
    await Promise.all(started.map((nerve) => nerve.stop()));
  },
  databaseSnapshot: [
    async ({ db }, use, testInfo) => {
      await use();
      if (testInfo.status !== testInfo.expectedStatus) {
        const file = testInfo.outputPath("database.sql");
        await db.dump(file);
        await testInfo.attach("database", { path: file, contentType: "application/sql" });
      }
    },
    { auto: true },
  ],
});
```

`e2e/fixtures/auth.ts`：

```ts
import type { components } from "@nerve/api-client";
import { expect, type TestInfo } from "@playwright/test";

import type { Api } from "./api";

export type AuthTokens = components["schemas"]["AuthTokens"];

/** A password that meets the rules and is not common. */
export const password = "Tr0ub4dor&3";

/**
 * An address of this run of this test: the tests of a worker share its
 * database, and --repeat-each runs a test again in the same worker.
 */
export function emailFor(testInfo: TestInfo, label = "user"): string {
  return `${label}-${testInfo.testId}-${testInfo.repeatEachIndex}-${testInfo.retry}@example.com`;
}

/** Signs email up through the API and returns the new session's tokens. */
export async function register(api: Api, email: string, headers: Record<string, string> = {}): Promise<AuthTokens> {
  const { data, error, response } = await api.POST("/api/v0/auth/register", {
    body: { email, password },
    headers,
  });
  expect(response.status, `register ${email}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`register ${email} answered 201 without tokens`);
  }
  return data;
}
```

`e2e/fixtures/assert/identity.ts`：

```ts
import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the identity stories. The page version and the API
// version of a story call the same function (v0 design 8.2).

const refreshTokenPrefix = "nrv_rt_";
const dayMs = 24 * 60 * 60 * 1000;

/** A refresh token's content (M2 design 3.4): nrv_rt_, then session id, generation, secret and tag in base64url. */
interface RefreshTokenParts {
  sessionId: string;
  generation: number;
  secret: Buffer;
}

function parseRefreshToken(token: string): RefreshTokenParts {
  expect(token.startsWith(refreshTokenPrefix), `${token} starts with ${refreshTokenPrefix}`).toBe(true);
  const raw = Buffer.from(token.slice(refreshTokenPrefix.length), "base64url");
  expect(raw.length, "a refresh token holds 68 bytes").toBe(68);
  const id = raw.subarray(0, 16).toString("hex");
  return {
    sessionId: `${id.slice(0, 8)}-${id.slice(8, 12)}-${id.slice(12, 16)}-${id.slice(16, 20)}-${id.slice(20)}`,
    generation: raw.readUInt32BE(16),
    secret: raw.subarray(20, 52),
  };
}

/** What a sign-up sent and got back. */
export interface Registration {
  /** The address as typed; the account holds it lowercased. */
  email: string;
  refreshToken: string;
  userAgent: string;
  ip: string;
}

/**
 * A1: registration added one account with the address lowercased, an
 * argon2id hash and the display name from the address; its default profile;
 * and a session of generation 0 whose hash is that of the refresh token's
 * secret, with the caller's User-Agent and IP, ending about 30 days later.
 */
export async function expectRegistered(db: Database, r: Registration): Promise<void> {
  const email = r.email.toLowerCase();
  const users = await db.query<{ id: string; password: string; display_name: string; is_active: boolean }>(
    "SELECT id, password, display_name, is_active FROM users WHERE email = $1",
    [email]
  );
  expect(users).toHaveLength(1);
  const [user] = users;
  expect(user?.password).toMatch(/^\$argon2id\$/);
  expect(user?.display_name).toBe(email.slice(0, email.indexOf("@")));
  expect(user?.is_active).toBe(true);

  const profiles = await db.query(
    `SELECT theme, is_tour_completed, onboarding_step, is_onboarded, last_workspace_id, language, start_of_the_week
       FROM profiles WHERE user_id = $1`,
    [user?.id]
  );
  expect(profiles).toEqual([
    {
      theme: "system",
      is_tour_completed: false,
      onboarding_step: {
        profile_complete: false,
        workspace_create: false,
        workspace_invite: false,
        workspace_join: false,
      },
      is_onboarded: false,
      last_workspace_id: null,
      language: "en",
      start_of_the_week: 0,
    },
  ]);

  const token = parseRefreshToken(r.refreshToken);
  const sessions = await db.query<{
    id: string;
    token_hash: Buffer;
    generation: number;
    user_agent: string;
    ip: string;
    expires_at: Date;
    created_at: Date;
    revoked_at: Date | null;
  }>(
    "SELECT id, token_hash, generation, user_agent, ip, expires_at, created_at, revoked_at FROM auth_sessions WHERE user_id = $1",
    [user?.id]
  );
  expect(sessions).toHaveLength(1);
  const [session] = sessions;
  expect(session?.id).toBe(token.sessionId);
  expect(token.generation).toBe(0);
  expect(session?.generation).toBe(0);
  expect(session?.token_hash.equals(createHash("sha256").update(token.secret).digest())).toBe(true);
  expect(session?.user_agent).toBe(r.userAgent);
  expect(session?.ip).toBe(r.ip);
  // auth.session_ttl is 720h: the session ends 30 days after it began.
  expect((session?.expires_at.getTime() ?? 0) - (session?.created_at.getTime() ?? 0)).toBe(30 * dayMs);
  expect(session?.revoked_at).toBeNull();
}

/** How many accounts, profiles and sessions there are. */
export interface IdentityCounts {
  users: number;
  profiles: number;
  sessions: number;
}

export async function countIdentity(db: Database): Promise<IdentityCounts> {
  const [counts] = await db.query<{ users: number; profiles: number; sessions: number }>(
    `SELECT (SELECT count(*)::int FROM users) AS users,
            (SELECT count(*)::int FROM profiles) AS profiles,
            (SELECT count(*)::int FROM auth_sessions) AS sessions`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}

/** A2: a refused sign-up added no account, profile or session. */
export async function expectNothingAdded(db: Database, before: IdentityCounts): Promise<void> {
  expect(await countIdentity(db)).toEqual(before);
}
```

- [ ] **Step 2: 故事**

`e2e/stories/identity/a1-sign-up.spec.ts`：

```ts
import { expectRegistered } from "../../fixtures/assert/identity";
import { emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A1, a new account (M2 design 2). The page version joins in M2/P4.

test("A1 (API): a caller signs up and gets a session", async ({ api, db }, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  const userAgent = "nerve-e2e/A1";

  const tokens = await register(api, email, { "User-Agent": userAgent });

  expect(tokens.token_type).toBe("Bearer");
  expect(tokens.access_token_expires_in).toBe(15 * 60);
  await expectRegistered(db, { email, refreshToken: tokens.refresh_token, userAgent, ip: "127.0.0.1" });

  // The access token works at once.
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${tokens.access_token}` } });
  expect(me.response.status).toBe(200);
  expect(me.data?.email).toBe(email.toLowerCase());
});
```

`e2e/stories/identity/a2-sign-up-refused.spec.ts`：

```ts
import { createApi, type Api } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { emailFor, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A2, sign-up refused (M2 design 2). The page version joins in M2/P4.

async function expectRefused(
  api: Api,
  body: { email: string; password: string },
  want: { status: number; code: string; fields?: { field: string; code: string }[] }
): Promise<void> {
  const { response, error } = await api.POST("/api/v0/auth/register", { body });
  const label = `${body.email} ${body.password}`;
  expect(response.status, label).toBe(want.status);
  expect(error?.code, label).toBe(want.code);
  if (want.fields) {
    expect(
      error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      label
    ).toEqual(want.fields);
  }
}

test("A2 (API): a refused sign-up answers why and adds nothing", async ({ api, db, nerveWith }, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const before = await countIdentity(db);
  const newEmail = emailFor(testInfo, "new");
  const invalid = (pw: string, code: string) =>
    expectRefused(
      api,
      { email: newEmail, password: pw },
      { status: 422, code: "validation_failed", fields: [{ field: "password", code }] }
    );

  await Promise.all([
    // The address is taken, whatever its case.
    expectRefused(api, { email: email.toUpperCase(), password }, { status: 409, code: "identity.email_taken" }),
    // A weak password, and common ones.
    invalid("password", "weak_password"),
    invalid("Password1!", "common_password"),
    invalid("Password1!~", "common_password"),
  ]);

  // With sign-up off, every address gets the same answer, a taken one too.
  const closed = createApi((await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" })).baseURL);
  await Promise.all(
    [newEmail, email].map((address) =>
      expectRefused(closed, { email: address, password }, { status: 403, code: "identity.signup_disabled" })
    )
  );

  await expectNothingAdded(db, before);
});
```

- [ ] **Step 3: 运行端到端**

Run: `make e2e`
Expected: S1、S2（两个）、S3、S4、A1、A2 全部通过。

核对失败时的数据库快照（不提交）：在 A1 的最后临时加一行 `expect(1).toBe(2);`，再运行 `make e2e`。
Expected: A1 失败；`e2e/test-results/` 下 A1 的目录里有 `database.sql`（含 `users`、`profiles`、`auth_sessions` 三张表），报告中它是附件。改回 A1，再运行 `make e2e`，全部通过。

- [ ] **Step 4: 静态检查**

Run: `make lint-web`
Expected: 通过（`e2e` 的 oxlint 上限是 0）。

Run: `make knip`
Expected: 通过。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 5: 提交**

```bash
git add e2e
```
```bash
git commit -m "test(M2/P1): A1 and A2 against the API, nerve on port 0 via its address file

Each worker holds one pg pool for database assertions, and a failed test
attaches a pg_dump of its database.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过；故意失败的测试带着 `database.sql` 附件；`make lint-web`、`make knip` 通过。

---

### Task 13: 上级文档、差异清单、README 与交接

**Files:**
- Modify: `docs/v0/v0-design.md`（3.1、3.5、4.1、4.2、5.5、5.6、6.2、6.3、6.4、8.2）
- Modify: `docs/v0/M0-foundation/M0-design.md`（3.3、3.5）
- Modify: `docs/v0/plane-diff.md`（一 B、二·全局、二·按表、三、四）
- Modify: `README.md`
- Modify: `docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md`、`M0-P2-platform-notes.md`、`M0-P3-api-codegen-notes.md`、`M0-P4-schema-conventions.md`、`M0-P6-e2e-notes.md`

**Interfaces:** 无代码。依据：M2 设计 3.20 中 P1 的各行、8.7 中 P1 的各行、12 节 P1 的"关闭"；spec 2.17 和第 3 节第 10 条（3.20 表漏掉的同步位置）。

**Tests:** 无新测试。`make lint-web` 的关键词守卫扫描这些文件；每一步的"把"必须在文件中恰好出现一次。

每一步的做法：用编辑工具把"把"下面的原文逐字替换为"替换为"下面的内容；"加在以……开头的那一行之后（之前）"的内容插在那一行之后（之前）；"末尾加上"的内容空一行加在文件最后。

- [ ] **Step 1: 总体设计 3.1、3.5**

`docs/v0/v0-design.md`，把：

```markdown
  - **POST 和 PATCH 都返回改完之后的完整资源。**
```

替换为：

```markdown
  - **POST 和 PATCH 都返回改完之后的完整资源。** 例外：注册（`POST /api/v0/auth/register`）返回令牌，不返回新建的账户，账户由 `GET /api/v0/me` 读取（M2 设计 5.1）。
```

把：

```markdown
    "errors": [{ "field": "state_id", "message": "..." }] }
```

替换为：

```markdown
    "errors": [{ "field": "state_id", "code": "not_allowed", "message": "..." }] }
```

把：

```markdown
- 平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；模块的错误码带模块前缀。
```

替换为：

```markdown
- 平台自己的错误码不带模块前缀：`bad_request`（400）、`unauthorized`（401）、`not_found`（404）、`payload_too_large`（413）、`validation_failed`（422）、`internal_error`（500）、`not_ready`（503，只用于 `/readyz`）、`server_busy`（503，带 `Retry-After`）；模块的错误码带模块前缀，例如 `identity.email_taken`（M2 设计 3.11）。
- **结构在接口边界，取值在领域**（M2 设计 3.11）：请求体不是合法 JSON、有未声明的字段、不可为空的字段传了 `null`、缺少必填字段、生成为 Go 类型的格式（`date-time`、`uuid`）写错，一律 400 `bad_request`，`errors` 一次列出全部问题；长度、其余格式、枚举、取值范围和跨字段的规则由领域层校验，一次返回 422 `validation_failed`。
- `errors` 的每一项是 `{field, code, message}`：`field` 是 JSON 路径，`code` 取自一个封闭的集合（`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`weak_password`、`common_password`、`must_be_future`、`contains_url`），前端按 `code` 显示文案。
- **错误码写进接口描述**：每个操作用扩展字段 `x-problem-codes` 列出它可能返回的码；所有操作都可能返回的平台码只写在 `api/openapi.yaml` 的顶层，声明了 `bearer` 的操作另外隐含 `unauthorized`。`apitest` 核对码的写法、测试中返回的码都已声明、每个声明的码都有测试返回过（M2 设计 3.11）。
```

- [ ] **Step 2: 总体设计 4.1、4.2**

把：

```markdown
| 刷新令牌 | 登录时一并下发 | 随机字符串，数据库只存哈希（`auth_sessions`） | 30 天；每次使用后换新，并检测旧令牌是否被重复使用 |
```

替换为：

```markdown
| 刷新令牌 | 注册、登录时一并下发 | `nrv_rt_` 加 68 字节的 base64url：会话 id、代数、32 字节的随机密文和 16 字节的 HMAC 标签，MAC 密钥从签名密钥派生；数据库只存当前一代密文的哈希（`auth_sessions`，M2 设计 3.4） | 30 天；每次使用后换新，并检测旧令牌是否被重复使用 |
```

把：

```markdown
- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置控制；被邀请的邮箱始终可以注册。
```

替换为：

```markdown
- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2/P3 加入；在那之前用 `NERVE_AUTH__SIGNUP_ENABLED=true` 临时打开注册）（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
```

- [ ] **Step 3: 总体设计 5.5、5.6**

把：

```markdown
- **审计字段**：`created_by` / `updated_by` 在业务代码中显式赋值为当前账户。
```

替换为：

```markdown
- **审计字段**：`created_by` / `updated_by` 在业务代码中显式赋值为当前账户；`created_at` / `updated_at` 同样由应用在每次插入、每次业务更新时显式写入，取自用例的时钟，数据库的 `DEFAULT now()` 只是应用之外写入时的兜底（M2 设计 3.13）。
```

把：

```markdown
- **外键方向**：每个模块的迁移只建自己的表，以及指向更早建立的模块的外键；指向更晚建立的模块的外键，由后建的模块用 `ALTER TABLE` 补上。
```

替换为：

```markdown
- **外键方向**：每个模块的迁移只建自己的表，以及指向更早建立的模块的外键；指向更晚建立的模块的外键，由后建的 M 用 `ALTER TABLE` 补上。这个迁移文件归**被改表的模块**：文件名带那个模块的名字，列在那个模块的 sqlc 条目里。例如 M5 给 `users` 加头像的外键，写 `<v>_identity_users_avatar_asset.sql`（M2 设计 3.14）。
```

- [ ] **Step 4: 总体设计 6.2、6.3**

把：

```markdown
    shared/                 共享内核，尽量小：ID 类型、Actor（当前账户）、领域错误与错误码、
                            分页游标、领域事件接口、Authorizer 端口
```

替换为：

```markdown
    shared/                 共享内核，尽量小，只放值会跨越模块边界的东西（M2 设计 3.3）：Actor（当前账户）、
                            领域错误与错误码、TxManager 端口；以后加入分页游标的封套、领域事件接口、
                            Authorizer 端口。平台不导入它；时钟等其余端口由使用方的 app 层声明
```

把：

```markdown
  domain/            实体、值对象、业务规则、领域事件、领域错误，以及本模块需要的端口（仓储接口等）
  app/               用例，一个用例一个文件（create_issue.go、archive_issue.go……）：
                     负责事务边界、权限检查和流程编排
```

替换为：

```markdown
  domain/            实体、值对象、业务规则、领域事件、领域错误：纯规则，没有 I/O
  app/               用例，一个用例一个文件（create_issue.go、archive_issue.go……）：
                     负责事务边界、权限检查和流程编排；本模块需要的端口（仓储、时钟等小接口）
                     声明在 app/ports.go，由使用方定义（M2 设计 3.3）
```

把：

```markdown
  module.go          模块入口：New(依赖) *Module；(*Module).Register(mux, apiErrors) 把模块生成的路由挂到 bootstrap 的根路由上（M0/P3）
```

替换为：

```markdown
  module.go          模块入口：New(依赖)；(*Module).Register(router, api) 把模块生成的路由挂到 bootstrap 的根路由上，
                     api（httpserver.API）提供错误映射和按路由的中间件；PublicOperations() 列出不需要令牌的操作；
                     其他模块或 bootstrap 要用的能力由访问方法导出，例如 Authenticator()（M2 设计 3.3、3.6）
```

把：

```markdown
`platform` 不依赖模块，`platform` 的各个包之间也不互相依赖；只有组合根能导入各个模块；生成的代码只能被本模块的适配器导入；测试工具只能被测试代码导入。
```

替换为：

```markdown
`platform` 不依赖模块、`bootstrap` 和 `shared`（平台声明自己需要的小接口，`shared` 的类型按结构满足它们，`bootstrap` 用编译期断言对上），`platform` 的各个包之间也不互相依赖；只有组合根能导入各个模块；`adapter/<技术>/gen` 下生成的代码只能被同一个适配器导入；测试工具只能被测试代码导入。
   - **sqlc 按模块限定**：`server/sqlc.yaml` 中每个模块的 `schema` 只列这个模块的迁移，一个模块的查询只能碰本模块的表（跨模块的读取在 `app` 层声明端口）；给别的模块的表加列、加约束的迁移归被改表的模块。架构测试 `TestSQLCSchemaScope` 核对这份配置（M2 设计 3.14）。
```

- [ ] **Step 5: 总体设计 6.4**

把（从代码块开始，到"用例代码不接触任何数据库类型。"为止）：

````markdown
```
请求 → 请求 ID → 异常恢复 → 访问日志 → 认证（识别 JWT 或 PAT，得到 Actor）
     → 限流 → 接口调用日志（只记写操作，异步批量写入）
     → handler（生成的代码只做参数绑定和 JSON 解码，不校验取值；校验放在哪一层由 M2 决定，见 [P3 spec](M0-foundation/specs/P3-api-contract.md) 7）
     → 用例：TxManager.WithinTx { 权限 → 业务规则 → 写数据 → 发布领域事件 } 提交
     → 响应 / problem+json
```
- **请求 ID → 异常恢复 → 访问日志**这三个平台中间件固定在 `httpserver.NewServer` 内部，不可漏掉或调换。
- **认证、限流、接口调用日志**挂在访问日志之后，按路由分别接到各个 API 路由上，不作用于健康检查和前端页面。
- **每个写操作对应一个事务。** 业务数据、操作动态、历史版本、投递给 River 的任务，要么一起成功，要么一起回滚。
- **事务的传递**：`TxManager` 端口声明在 `internal/shared`（由使用方定义接口），`platform/postgres` 提供实现，`bootstrap` 负责接线；仓储从 `ctx` 中取出当前事务。用例代码不接触任何数据库类型。
````

替换为：

````markdown
```
请求 → 请求 ID → 异常恢复 → 访问日志                            （固定链，httpserver.NewServer）
     → 路由匹配 → 生成的代码绑定路径参数和查询参数（格式错误 → 400）
     → 请求元信息（客户端 IP、UA）→ 请求期限 → 请求体上限           （按路由，httpserver.API）
     → 认证（识别 JWT 或 PAT，得到 Actor；公开操作不看令牌）→ 限流
     → 请求体结构检查（不合契约 → 400）→ 生成的代码解码 JSON 请求体
     → handler（只做类型转换）
     → 用例：TxManager.WithinTx { 权限 → 领域校验（不合规 → 422）→ 业务规则 → 写数据 → 发布领域事件 } 提交
     → 响应；或者 error → APIErrors → problem+json
```
- **请求 ID → 异常恢复 → 访问日志**这三个平台中间件固定在 `httpserver.NewServer` 内部，不可漏掉或调换。
- **按路由的中间件**由 `httpserver.API.Middlewares` 按上图的顺序交给每个模块的生成代码，只作用于 `/api/v0` 的操作，不作用于健康检查和前端页面（M2 设计 3.6）。认证默认拒绝：除了模块声明为公开的操作，没有有效令牌一律 401。限流由 M2/P2 加入，接口调用日志由 M8 挂在限流之后。
- **参数先于这些中间件绑定**：生成的代码在它们之前绑定路径参数和查询参数，参数格式错误的请求在认证之前就得到 400；请求体在它们之后才解码，没有通过认证的请求不会被解析请求体。
- **每个写操作对应一个事务。** 业务数据、操作动态、历史版本、投递给 River 的任务，要么一起成功，要么一起回滚。
- **事务的传递**：`TxManager` 端口声明在 `internal/shared`（由使用方定义接口），`platform/postgres` 提供实现，`bootstrap` 负责接线；仓储从 `ctx` 中取出当前事务。用例代码不接触任何数据库类型。
- **提交不受请求期限的取消**：事务里的语句带请求的 `context`，请求期限（`server.request_timeout`）到了就取消；`COMMIT` 和 `ROLLBACK` 在 `context.WithoutCancel` 下执行，有自己的期限 `database.commit_timeout`。语句都已做完的事务不会因为请求期限在提交时被取消，失败的事务也总能回滚（M2 设计 3.6）。
````

- [ ] **Step 6: 总体设计 8.2**

把：

```markdown
      server.ts    每个 Playwright 并行进程启动一个 nerve（NERVE_ENV=test，随机端口）
      db.ts        每个并行进程一个独立数据库（从模板库复制），以及数据库断言函数
```

替换为：

```markdown
      server.ts    每个 Playwright 并行进程启动一个 nerve（NERVE_ENV=test，在 127.0.0.1:0 上监听，
                   从 server.addr_file 读出实际地址）；需要另一种配置的故事在同一个库上另起一个
      db.ts        每个并行进程一个独立数据库（从模板库复制）和一个连接池
      assert/      数据库断言函数，按表分文件；页面版本和接口版本调用同一个函数
      auth.ts      通过接口注册、登录，创建 PAT；页面的登录状态（M2 起）
```

把：

```markdown
  - 失败时保存操作记录（trace）和截图；M0 另外保存 nerve 的日志。录像和数据库快照从有业务表的 M 起再加入（M0/P6：trace 已含每一步的截屏，录像还要多下载 ffmpeg；M0 没有业务表）。
```

替换为：

```markdown
  - 失败时保存操作记录（trace）、截图、nerve 的日志和本 worker 数据库的快照（`pg_dump`，M2/P1 起）。不录像：trace 已含每一步的截屏，录像还要多下载 ffmpeg（M2 设计 9.5）。
```

- [ ] **Step 7: M0 设计 3.1、3.3、3.5**

`docs/v0/M0-foundation/M0-design.md`，把：

```markdown
| `internal/platform/*` | 与业务无关的技术基础件，各包之间互不依赖（`config` 除外，它可以被任何包使用） | 标准库和第三方库；**不能依赖 `modules`** |
```

替换为：

```markdown
| `internal/platform/*` | 与业务无关的技术基础件，各包之间互不依赖（`config` 除外，它可以被任何包使用） | 标准库和第三方库；**不能依赖 `modules`、`bootstrap` 和 `internal/shared`**（M2/P1：平台声明自己需要的小接口，`shared` 的类型按结构满足） |
```

把：

```markdown
| `internal/modules/<m>` 下的 `module.go` | 模块的对外入口：`New(依赖) *Module`；`(*Module).Register(mux, apiErrors)` 把模块生成的路由挂到根路由上 | 本模块内部的各个包 |

`internal/shared`（共享内核）在第一次真正需要跨模块共享类型时才建立（预计在 M2），M0 不建空包。
```

替换为：

```markdown
| `internal/modules/<m>` 下的 `module.go` | 模块的对外入口：`New(依赖)`；`(*Module).Register(router, api)` 把模块生成的路由挂到根路由上，`api`（`httpserver.API`）提供错误映射和按路由的中间件；`PublicOperations()` 列出不需要令牌的操作（M2/P1） | 本模块内部的各个包 |

`internal/shared`（共享内核）在第一次真正需要跨模块共享类型时才建立，M0 不建空包。M2/P1 建立它：`Actor`、领域错误 `Error`、`TxManager` 端口（M2 设计 3.3）。
```

把：

```markdown
- **路由**：Go 标准库的 `ServeMux`（Go 1.22 以后支持按方法和路径匹配）。挂载规则如下：
```

替换为：

```markdown
- **路由**：Go 标准库的 `ServeMux`（Go 1.22 以后支持按方法和路径匹配）。M2/P1 起由 `httpserver.Router` 包一层，记下注册的每个模式，供整程序测试与接口描述核对（M2 设计 3.6）。挂载规则如下：
```

把：

```markdown
  3. 访问日志：用 slog 记录方法、路径、状态码、耗时、请求 ID。
```

替换为：

```markdown
  3. 访问日志：用 slog 记录方法、路径、状态码、耗时、请求 ID。`/healthz`、`/readyz` 的记录是 DEBUG 级别，其余是 INFO（M2/P1）。
- **按路由的中间件**（M2/P1，M2 设计 3.6）：`/api/v0` 的操作另有一串中间件，由 `httpserver.API.Middlewares` 交给每个模块的生成代码，在访问日志之后、按这个顺序：请求元信息（客户端 IP、UA）→ 请求期限（`server.request_timeout`）→ 请求体上限（`server.max_body_bytes`，超过是 413）→ 默认拒绝的认证（模块声明为公开的操作之外，没有有效令牌一律 401）→ 请求体结构检查（不合契约是 400）。生成代码在它们之前绑定路径参数和查询参数，在它们之后解码请求体。
```

把：

```markdown
- **problem+json**：M0 定义统一的写出函数和 `Problem` 结构（与 `api/common.yaml` 中的定义一致），包含可选的 `detail`。平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；领域错误码的体系在 M2 建立。
- **生成代码的错误出口**（M0/P3）：oapi-codegen 生成的代码在参数绑定、请求体解码、handler 返回错误三处默认输出纯文本；`httpserver.APIErrors` 把三处都接成 problem+json：绑定或解码失败 → `BadRequest`（400 `bad_request`，`detail` 是失败原因）；handler 出错或响应写出失败 → `InternalError`（500 `internal_error`，不带 `detail`；响应已经开始时改为记录日志并中断连接，不追加 problem）。
```

替换为：

```markdown
- **problem+json**：M0 定义统一的写出函数和 `Problem` 结构（与 `api/common.yaml` 中的定义一致），包含可选的 `detail`。平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；M2/P1 加入 `unauthorized`、`payload_too_large`、`validation_failed`、`server_busy` 和领域错误码的体系（M2 设计 3.11）。
- **生成代码的错误出口**（M0/P3，M2/P1 修改）：oapi-codegen 生成的代码在参数绑定、请求体解码、handler 返回错误三处默认输出纯文本；`httpserver.APIErrors` 把三处都接成 problem+json：参数绑定失败 → `BadRequest`（400 `bad_request`，`detail` 是失败原因）；请求体解码失败 → `BodyError`（400 `bad_request`，`detail` 是通用的一句话，不带出 Go 的类型名；超过请求体上限是 413 `payload_too_large`）；handler 出错或响应写出失败 → `Write`：满足 `httpserver.ProblemError` 的错误映射为它的状态、码、`detail` 和字段，其余是 500 `internal_error`（不带 `detail`，原因只进日志）；客户端断开（`context.Canceled`）不算 500；响应已经开始时改为记录日志并中断连接，不追加 problem。
```

把：

```markdown
- **迁移文件如何内嵌**：`server/migrations/embed.go` 用 `//go:embed all:sql` 内嵌整个目录。之所以不写 `//go:embed *.sql`，是因为它在一个 `.sql` 文件都没有时会编译失败。
```

替换为：

```markdown
- **迁移文件如何内嵌**：`server/migrations/embed.go` 用 `//go:embed sql/*.sql` 内嵌迁移文件。M0 没有迁移文件，当时写的是 `//go:embed all:sql`（`*.sql` 在一个文件都没有时会编译失败）；M2/P1 加入第一批迁移后改为现在的写法，删掉了占位的 `sql/.gitkeep`。
```

把：

```markdown
- **M0 不包含任何迁移文件**：迁移机制（执行、回滚、查看状态、没有迁移文件时的处理）由集成测试验证，测试使用专门的测试迁移集，只存在于测试中。
```

替换为：

```markdown
- **迁移文件从 M2 开始**：M0 不包含任何迁移文件，迁移机制（执行、回滚、查看状态、没有迁移文件时的处理）由集成测试验证，测试使用专门的测试迁移集，只存在于测试中。M2/P1 加入第一批迁移（`users`、`profiles`、`auth_sessions`），`server/migrations/schema_test.go` 核对它们能 up、down、再 up，以及约束名和 CHECK。
```

- [ ] **Step 8: 差异清单一 B、二·全局**

`docs/v0/plane-diff.md`，把：

```markdown
| `sessions` | 改用 JWT 和 `auth_sessions` |
```

替换为：

```markdown
| `sessions` | 替换模型，不逐列继承：Django 的会话存储由 JWT 访问令牌和 `auth_sessions`（一次登录一行，见二·按表）替代（M2 设计 4.5） |
```

在二·全局的表中，加在以 `| ID 改由应用生成 UUIDv7` 开头的那一行之后：

```markdown
| 外键写上 `ON DELETE`，照搬每个外键在 Django 模型中的 `on_delete`：`CASCADE` → `ON DELETE CASCADE`，`SET_NULL` → `ON DELETE SET NULL`，`PROTECT` → `ON DELETE RESTRICT`，`DO_NOTHING` 不写；不用 `DEFERRABLE` | Plane 在 Python 中处理级联，快照中 474 个外键都是 `DEFERRABLE INITIALLY DEFERRED`、没有 `ON DELETE`；Nerve 在同一个事务里按先父后子的顺序写（M2 设计 3.13，删除关系图见 M2 设计 4.7） |
| 约束和索引一律改名：主键、外键、唯一约束和一列唯一的 CHECK 用 Postgres 的默认名（`<表>_pkey`、`<表>_<列>_fkey`、`<表>_<列>_key`、`<表>_<列>_check`）；涉及多列的 CHECK 显式命名为 `<表>_<含义>_check`；索引命名为 `<表>_<列>_idx` | 快照中的名字带 Django 的哈希后缀，29 张表的主键名与表名不对应；按约束名映射错误、以后 `DROP CONSTRAINT` 都需要稳定的名字（M2 设计 3.13） |
| 不建 `*_like`（`varchar_pattern_ops`）索引；外键列只在查询或级联用到时建索引，`created_by_id`、`updated_by_id` 不建 | 前者只服务 Django 的 `LIKE 'x%'`；后者是 Django 给每个外键默认建的，多数用不上（M2 设计 3.13） |
| 审计时间列 `created_at`、`updated_at` 由应用按用例的时钟显式写入；`DEFAULT now()` 是新加的兜底，只在应用之外写入时起作用 | 时间只有一个来源，测试用固定时钟；自动归档按 `updated_at` 判断（M2 设计 3.13） |
| 对象型的 `jsonb` 列用 CHECK 保证是对象、键的集合和每个值的类型 | Plane 只在 Python 中保证（M2 设计 3.13） |
```

- [ ] **Step 9: 差异清单二·按表**

把：

```markdown
| `users` | 删除 `is_bot`、`bot_type` | 核心原则：不区分人和机器人 |
| `users` | 删除旧的头像和封面 URL 列（`avatar`、`cover_image`），改用 `*_asset_id` | 遗留列 |
| `profiles` | 删除账单字段和移动端字段 | Plane 云服务专用或已废弃 |
```

替换为：

```markdown
| `users` | 40 列保留 10 列（M2/P1，`00001_identity_users.sql`）；主键名 `users_pkey`、唯一约束名 `users_email_key`（Plane 是 `user_pkey`、`user_email_key`） | M2 设计 4.2 |
| `users` | `email`：改为 `NOT NULL`（Plane 可为空）；新加 `CHECK (email = lower(email) AND email !~ '[[:space:]]')` | 规范化原来只在 Python 里做 |
| `users` | `password`：列名和类型照搬，内容改为 argon2id 的 PHC 字符串 | 密码哈希改用 argon2id（M2 设计 3.8） |
| `users` | `first_name`、`last_name`：新加 `DEFAULT ''` | 模型的默认值 |
| `users` | `display_name`：新加 `CHECK (display_name <> '')`；注册时取邮箱 @ 之前的部分 | Plane 由 `User.save()` 填上，数据库不约束 |
| `users` | `user_timezone`：新加 `DEFAULT 'UTC'` | 模型的默认值（取值范围的差异由 M2/P3 登记在第四节） |
| `users` | `is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `users` | 删除 Django 与管理后台的列 `last_login`、`is_superuser`、`is_staff`、`username`，以及与 `created_at` 重复的 `date_joined` | Nerve 没有 Django 的管理后台；`username` 前端从不读取 |
| `users` | 删除登录记录 `last_login_time`、`last_logout_time`、`last_login_ip`、`last_logout_ip`、`last_login_medium`、`last_login_uagent`、`last_active`、`token`、`token_updated_at` | 改由 `auth_sessions` 承担 |
| `users` | 删除 `is_password_autoset`、`is_email_verified`、`is_email_valid`、`mobile_number`、`last_location`、`created_location`、`is_managed`、`is_password_expired`、`is_password_reset_required`、`masked_at` | 第三方登录、验证码登录、邮箱验证已砍掉；其余在 Plane 代码中已不使用 |
| `users` | 删除 `is_bot`、`bot_type` | 核心原则：不区分人和机器人 |
| `users` | 删除旧的头像和封面 URL 列（`avatar`、`cover_image`）；`avatar_asset_id`、`cover_image_asset_id` 暂不建，由 M5 随文件存储加入（迁移归 `identity`） | 遗留列；在那之前接口的 `avatar_url`、`cover_image_url` 是 `null` |
| `profiles` | 29 列保留 11 列（M2/P1，`00002_identity_profiles.sql`） | M2 设计 4.3 |
| `profiles` | `user_id`：唯一约束照搬（`OneToOne`），加上 `ON DELETE CASCADE` | 二·全局 |
| `profiles` | `theme`：`jsonb`（默认 `{}`，存自定义调色板）改为 `varchar(20) NOT NULL DEFAULT 'system'`，`CHECK (theme IN ('system','light','dark','light-contrast','dark-contrast'))` | 自定义主题已删除（M1），只剩一个值 |
| `profiles` | `is_tour_completed`、`is_onboarded`：新加 `DEFAULT false` | 模型的默认值 |
| `profiles` | `onboarding_step`：新加默认值（四个键都是 `false`）和 CHECK（恰好四个键，值都是布尔值） | 模型的 `get_default_onboarding()`；Plane 只在 Python 中保证 |
| `profiles` | `language`：新加 `DEFAULT 'en'` 和 `CHECK (language IN ('en','zh-CN'))` | Nerve 只有两种语言；Plane 接受任意字符串 |
| `profiles` | `start_of_the_week`：新加 `DEFAULT 0`；CHECK 由 `>= 0` 改为 `BETWEEN 0 AND 6` | 模型的 choices |
| `profiles` | `created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `profiles` | 删除 `role`、`use_case` | 新手引导的"角色""用途"两步已删除（M2 设计 3.19） |
| `profiles` | 删除账单和移动端字段 `billing_address_country`、`billing_address`、`has_billing_address`、`company_name`、`is_mobile_onboarded`、`mobile_onboarding_step`、`mobile_timezone_auto_set` | Plane 云服务专用或已废弃 |
| `profiles` | 删除 `is_smooth_cursor_enabled`、`is_app_rail_docked`、`background_color`、`goals`、`is_navigation_tour_completed`、`product_tour`、`notification_view_mode`、`has_marketing_email_consent`、`is_subscribed_to_changelog` | 前端不用的 Plane 新功能，以及营销邮件、更新日志 |
```

把：

```markdown
| `auth_sessions` | **新增**：刷新令牌的哈希、令牌轮换链（用于重复使用检测）、UA、IP、过期和撤销时间 | JWT 认证 |
```

替换为：

```markdown
| `auth_sessions` | **新增**（M2/P1，`00003_identity_auth_sessions.sql`，替代 `sessions`，见一 B）：一次登录一行，12 列：`id`（访问令牌中的 `sid`）、`user_id`（`ON DELETE CASCADE`）、`token_hash`（当前一代刷新令牌密文的 SHA-256，32 字节）、`generation`（代数）、`user_agent`、`ip`（`inet`）、`expires_at`（登录时刻加会话期限，之后不变）、`last_refreshed_at`、`revoked_at`、`revoke_reason`（六个取值）、`created_at`、`updated_at`；`auth_sessions_revoked_consistent_check` 要求 `revoked_at` 与 `revoke_reason` 同时为空或同时有值；索引 `auth_sessions_user_id_idx`、`auth_sessions_expires_at_idx`。旧代的刷新令牌不存，由令牌里的 HMAC 标签认出 | JWT 认证；刷新令牌的轮换和重复使用检测（M2 设计 3.4、3.5、4.5） |
```

- [ ] **Step 10: 差异清单三、四**

在第三节的表中，加在以 `| 无权限 |` 开头的那一行之后：

```markdown
| 错误码 | 接口描述中没有 | 每个操作在接口描述中用 `x-problem-codes` 声明它可能返回的错误码；字段错误带 `code`（M2 设计 3.11） |
| 请求体 | DRF 的序列化器忽略未知字段 | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段：400 `bad_request`，一次列出全部问题（M2 设计 3.11） |
```

在第四节的表中，加在以 `| 忘记密码 |` 开头的那一行之后：

```markdown
| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行；修改密码、创建账户和重置密码两个命令随 M2/P3 加入 |
| 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2/P3 加入）（M2 设计决策点 2） |
| 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（M2 设计 3.11） |
| 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`，带 `Retry-After: 1`（M2 设计 3.8） |
```

- [ ] **Step 11: README**

`README.md`，把：

```markdown
- `make test` 不用 Go 的测试缓存：缓存不跟踪 `server/` 之外的文件，契约测试读取的 `api/` 改了也会重放旧结果。直接执行 `go test` 时，改了 `api/` 要加 `-count=1`。
```

替换为：

```markdown
- `make test` 不用 Go 的测试缓存：缓存不跟踪 `server/` 之外的文件，契约测试读取的 `api/` 改了也会重放旧结果。直接执行 `go test` 时，改了 `api/` 要加 `-count=1`。
- `server/tools` 是独立的 Go 模块（代码生成工具），`server/` 下的 `go test ./...` 不进入它；`make test` 和 `make lint-go` 都另外在它里面跑一遍。
```

把：

```markdown
  - `make gen-go`：Go 接口层（`server/internal/modules/<模块>/adapter/http/gen/`、`server/internal/platform/httpserver/apigen/`），只需要 Go。
```

替换为：

```markdown
  - `make gen-go`：只需要 Go。生成 Go 接口层（`server/internal/modules/<模块>/adapter/http/gen/server.gen.go`、`server/internal/platform/httpserver/apigen/`）；每个模块的请求体结构表（同一目录的 `bodyshape.gen.go`，由 `server/tools/bodyshapegen` 读同一份接口描述生成，服务端据此在 handler 之前拒绝不合契约的请求体）；以及 sqlc 按 `server/sqlc.yaml` 从 `server/internal/modules/<模块>/adapter/postgres/queries/*.sql` 生成的查询代码（`adapter/postgres/gen/`；sqlc 以 `CGO_ENABLED=0` 运行，不需要 C 编译器）。
```

把：

```markdown
- 生成的文件不要手改。`make gen-check` 会重新生成一遍，检查生成物已经提交、没有差异；持续集成也执行这项检查。
```

替换为：

```markdown
- 生成的文件不要手改。`make gen-check` 会重新生成一遍，检查生成物已经提交、没有差异；持续集成也执行这项检查。
- 每个操作都写明 `security`（需要令牌的写 `[{bearer: []}]`，公开的写 `[]`）和 `x-problem-codes`（它可能返回的错误码；所有操作都可能返回的写在 `api/openapi.yaml` 顶层）。`apitest` 的测试核对这些写法，模块的 handler 测试要把声明的每个错误码都返回一次。
```

把：

```markdown
- **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，在一个空闲的本机端口上用 test 配置启动自己的 `nerve serve`，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库。
```

替换为：

```markdown
- **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，用 test 配置启动自己的 `nerve serve`：nerve 在 `127.0.0.1:0` 上监听，把实际地址写进地址文件（`server.addr_file`），fixture 读出地址，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库（每个 worker 一个连接池；断言函数按表放在 `e2e/fixtures/assert/`）。需要另一种配置的故事（例如关闭注册）用 `nerveWith` 在同一个库上另起一个 nerve。
```

把：

```markdown
- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）和截图在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。持续集成的 `e2e` 任务失败时，把这两个目录上传为 `playwright-report`。
```

替换为：

```markdown
- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）、截图和数据库快照（`database.sql`，本 worker 数据库的 `pg_dump`，也是报告中的附件）在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。持续集成的 `e2e` 任务失败时，把这两个目录上传为 `playwright-report`。
```

加在以 `## Plane 表结构快照` 开头的那一行之前（新的一节，后面留一个空行）：

````markdown
## 部署

- **设 `NERVE_ENV=prod`**。不设时 nerve 按 dev 的默认值运行：注册默认开放；没有配置签名密钥时用临时密钥，重启后已签发的访问令牌全部失效。nerve 不是 prod、却监听在本机回环地址之外时，启动日志会记一条 WARN 提醒。
- **签名密钥**：prod 必须提供 Ed25519 私钥（PKCS#8 PEM 文件），用 `auth.jwt.private_key_file`（环境变量 `NERVE_AUTH__JWT__PRIVATE_KEY_FILE`）指向它；没有提供或读不出来时 nerve 拒绝启动，错误指出这个配置项。生成：

  ```bash
  openssl genpkey -algorithm ed25519 -out nerve-jwt.pem
  chmod 600 nerve-jwt.pem
  ```

  访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create` 创建（M2/P3 加入；在那之前临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`）。
- **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。
````

`README.md` 的末尾（`## 版权` 一节的最后一段之后，空一行）加上：

```markdown
服务端的常见密码名单 `server/internal/modules/identity/domain/common_passwords.txt` 由 `tools/password-blocklist/build.mjs` 从英国国家网络安全中心（NCSC）发布的 `PwnedPasswordsTop100k.txt`（泄露最多的前 10 万个密码，数据来自 Have I Been Pwned 的 Pwned Passwords）过滤生成。Contains public sector information licensed under the [Open Government Licence v3.0](https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/). Pwned Passwords 本身没有许可和署名要求。
```

- [ ] **Step 12: 交接**

`docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md`、`M0-P4-schema-conventions.md`：把文件头的 `status: open` 替换为 `status: done`。

`docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md` 的末尾（"来源"一行之后，空一行）加上：

```markdown
## 处理结果（M2/P1）

1. **cgo**：sqlc 一律以 `CGO_ENABLED=0` 运行（`Makefile` 的 `SQLC`）。这时它用编译成 wasm 的 libpg_query（`wasilibs/go-pgquery`），不需要 C 编译器，也就不需要 Docker 镜像这条退路。原型用 cgo 和非 cgo 各生成一次，输出逐字节相同（[P1 spec](../specs/P1-platform-core.md) 附录 A）。
2. **PG 17 解析器**：迁移不写 PG 18 的语法，也不写 `DEFAULT uuidv7()`，ID 由应用生成（M2 设计 3.13）。`00001`–`00003` 由 sqlc 解析通过。

来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.10。
```

`docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md` 的末尾加上：

```markdown
## 处理结果（M2/P1）

1. **约定**：按 M2 设计 3.13 一次定下，差异清单二·全局已登记。外键照搬 Django 模型的 `on_delete`，不用 `DEFERRABLE`；主键、外键、唯一约束和一列唯一的 CHECK 用 Postgres 的默认名，多列的 CHECK 显式命名，索引命名为 `<表>_<列>_idx`；默认值和取值范围从 Plane 的模型读出写进数据库，差异清单逐列写明"新加"；系统表不照搬；不建 `*_like` 索引，外键列只在用到时建索引；`varchar(n)` 照搬。`00001`–`00003` 按这些约定建出 `users`、`profiles`、`auth_sessions`；`server/migrations/schema_test.go` 核对 19 个约束和索引名，以及 25 个 CHECK 的反例。
2. **主键与 ID**：照旧，没有默认值的 `id uuid`，由应用用 `uuid.NewV7()` 生成。

来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.10。
```

`docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md` 的末尾加上（`status` 仍为 `open`）：

```markdown
## 处理结果（M2/P1）

1. **TxManager**（完成）：`shared.TxManager` 由使用方声明，`platform/postgres.TxManager` 按结构满足，`bootstrap` 编译期断言。`COMMIT`、`ROLLBACK` 在 `context.WithoutCancel` 下执行，有自己的期限 `database.commit_timeout`（M2 设计 3.6）。
2. **按路由挂载**（认证完成）：`httpserver.API.Middlewares` 把请求元信息、请求期限、请求体上限、默认拒绝的认证和请求体结构检查挂在每个模块的生成代码上，`NewServer` 的固定链不变。
3. **`RequestID`**（完成）：已导出，`APIErrors` 记 500 时带上它。
4. **`LogValue`**（完成）：`*_file` 只记是否设置（`addr_file_set`、`private_key_file_set`），不记路径。
5. **期限**（请求期限完成）：`server.request_timeout`（默认 15 秒）给每个接口请求一个期限。
6. **archtest**（完成）：规则 6 推广到 `adapter/*/gen`；规则 4 加上"平台不导入 `internal/shared`"。
7. **迁移与就绪检查**（完成）：第一批迁移已加入；S1 核对全部迁移都已应用、`goose_db_version` 的最大版本等于最后一个迁移文件；README 的"部署"一节写明服务的角色要能读 `goose_db_version`。
8. **三个小问题**（完成）：环境变量给布尔、数字、时长配置项传空值时报错；`nerve serve` 在 logger 建好之后出现致命错误时另记一条 ERROR "nerve serve failed"；`/healthz`、`/readyz` 的访问日志降为 DEBUG。

仍未处理，状态保持 `open`：第 2 条的限流（M2/P2）和接口调用日志（M8，挂在限流之后）；第 5 条的 River、停机顺序、连接池关闭的时限和 River 的迁移（M2/P3）。

来源：[M2/P1 spec](../specs/P1-platform-core.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md` 的末尾加上（`status` 仍为 `open`）：

```markdown
## 处理结果（M2/P1）

1. **生成配置**（google/uuid 的守卫之外完成）：模块模板写 `nullable-type: true`，`type-mapping` 把 `uuid` 映射为标准库的 `uuid.UUID`、`email` 映射为 `string`；`prefer-skip-optional-pointer` 保持默认。`github.com/oapi-codegen/nullable` v1.2.0 随 identity 的生成代码加入。google/uuid 的守卫按 M2 设计 3.12 改为直接检查，随第一个带参数的操作（`oapi-codegen/runtime`）在 M2/P3 实现。
2. **错误映射**（请求体解码和 handler 两个出口完成）：结构在接口边界（`bodyshape`，400 `bad_request`、`errors[{field, code}]`），取值在领域（422 `validation_failed`）；解码失败的 `detail` 是通用的一句话，不带出 Go 的类型名；`http.MaxBytesError` 是 413 `payload_too_large`；`context.Canceled` 不记 500；handler 返回 `error`，由 `APIErrors.Write` 按 `ProblemError` 映射。identity 的 handler 测试覆盖请求体解码和 handler 两个出口，并对 problem 做 `CheckResponse`；`apitest.CheckRequest` 已加入。
3. **安全声明**（完成）：每个操作写 `security`；`securitySchemes.bearer` 同时写在 `api/openapi.yaml` 和每个模块文件；`apitest` 的写法检查拒绝没有 `security` 的操作和未声明的 scheme。
4. **模块入口**（完成）：`Register(router, api)`，`api` 是 `httpserver.API`；`httpadapter.Register` 接收用例集合；`API.Middlewares` 按相反的顺序返回，有测试核对；`identity` 导出 `Authenticator()` 和 `PublicOperations()`。
5. **组织规则**（完成）：照旧；M2 没有 map 型对象，`closedObject` 不需要例外。

仍未处理，状态保持 `open`：第 1 条中 google/uuid 的守卫，第 2 条中参数绑定的出口（`listApiTokens`、`revokeApiToken`），都在 M2/P3。

来源：[M2/P1 spec](../specs/P1-platform-core.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md` 的末尾加上（`status` 仍为 `open`）：

```markdown
## 处理结果（M2/P1）

- **认证 fixture**（注册完成）：`e2e/fixtures/auth.ts` 提供 `password`、`emailFor`、`register`；A1、A2 的接口版本调用 `e2e/fixtures/assert/identity.ts` 的断言函数。
- **数据库断言**（完成）：每个 worker 一个 `pg` 连接池（`openDatabase`），断言函数按表放在 `e2e/fixtures/assert/`。测试失败时，自动 fixture `databaseSnapshot` 用 `docker exec … pg_dump` 导出本 worker 的库到 `database.sql`，作为附件和 trace、截图、nerve 日志放在一起。
- **S1**（完成）：核对 `nerve migrate status` 的每一行都是 `applied`、来源依次等于迁移文件，`goose_db_version` 的最大版本等于最后一个文件的序号。模板库仍只由全局准备连接。
- **端口与等待**（完成）：nerve 在 `127.0.0.1:0` 上监听，把实际地址写进 `server.addr_file`，fixture 读取；空闲端口的猜测和"就绪后再等一个轮询间隔"的缓解一起删除。读地址文件和轮询 `/readyz` 共用 30 秒的期限，每个请求只用剩余时间；`pg_dump` 超过 60 秒就 SIGKILL；`nerveWith` 另起的 nerve 把自己的预算加到测试的超时上。
- **录像**（完成）：不录像，总体设计 8.2 已改为"trace、截图、nerve 日志和数据库快照"。

仍未处理，状态保持 `open`：PAT 对等验收，认证 fixture 的登录、PAT 和页面的登录状态（M2/P2–P4）；S3 的 `signup_enabled`（M2/P3）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3）；fixture 写法的延伸（M4、M5、M8）。

来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.16。
```

- [ ] **Step 13: 检查**

Run: `make lint-web`
Expected: 通过（关键词守卫扫描改过的文档）。

Run: `grep -n "Register(mux, apiErrors)\|随机端口\|InternalError\|all:sql\|预计在 M2" docs/v0/v0-design.md docs/v0/M0-foundation/M0-design.md`
Expected: 只剩 M0 设计 3.5 中说明历史写法的 `all:sql` 一处。

- [ ] **Step 14: 提交**

```bash
git add docs/v0/v0-design.md docs/v0/M0-foundation/M0-design.md docs/v0/plane-diff.md README.md docs/v0/M2-auth/handoffs
```
```bash
git commit -m "docs(M2/P1): sync the design documents, plane-diff and README; record the handoff results

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** M2 设计 3.20 中 P1 的 15 行和 spec 第 3 节第 10 条的四处都已同步；README 有 8.7 中 P1 的四项；M0-P1、M0-P4 交接为 `done`，M0-P2、M0-P3、M0-P6 追加了处理结果并保持 `open`。
