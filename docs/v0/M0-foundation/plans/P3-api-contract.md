# M0/P3 接口契约流水线与试点模块 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 走通"接口描述 → 生成代码 → 模块 → 接线 → HTTP → TS 客户端"的链路：OpenAPI 3.1 的 `api/` 目录与 Redocly 打包、按模块运行的 oapi-codegen 与共享包 `apigen`、生成代码的错误改为 problem+json、试点模块 `instance`（`GET /api/v0/instance`）、kin-openapi 契约测试、`web/packages/api-client`、`make gen` / `make gen-check`，以及按区域拆分的 Makefile 命令和持续集成。

**Architecture:** 接口描述是唯一的依据：`api/modules/<模块>.yaml` 各是一份完整的 OpenAPI 3.1 文档，`api/common.yaml` 放公共组件，`api/openapi.yaml` 是打包入口。Go 端每个模块单独运行 oapi-codegen（strict-server + std-http-server），公共组件通过 import-mapping 生成到 `platform/httpserver/apigen`；模块的 `adapter/http` 实现生成的强类型接口，把路由直接注册到 `httpserver.NewMux` 返回的根路由上，错误出口接平台的 `APIErrors`。Redocly 把入口打包成 `api/dist/openapi.yaml`，它既是 TS 类型的来源，也是契约测试（`platform/httpserver/apitest`，只被测试导入）的依据。所有生成物都提交到仓库，`make gen-check` 保证它们与描述一致。

**Tech Stack:** Go 1.27.1、oapi-codegen v2.8.0（`server/tools/go.mod`，已有）、kin-openapi v0.149.0、golangci-lint 2.13.2；Node 24、pnpm 11.10.0、@redocly/cli 2.53.3、openapi-typescript 7.13.0、openapi-fetch 0.17.0、TypeScript 5.8.3。

**Spec:** `docs/v0/M0-foundation/specs/P3-api-contract.md`（上级：`docs/v0/M0-foundation/M0-design.md`）

## Global Constraints

- Go：`server/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`，**每次 `go get` / `go mod tidy` 之后都要检查**。不改动 `server/tools/go.mod`。本 Phase 只新增一个 Go 依赖：`github.com/getkin/kin-openapi@v0.149.0`（写死，不要用 `@latest`）。
- Node 依赖一律写精确版本，通过 pnpm 工作区安装，不做任何全局安装：`@redocly/cli` `2.53.3`（根 `package.json`）、`openapi-typescript` `7.13.0`、`typescript` `5.8.3`、`openapi-fetch` `0.17.0`（`web/packages/api-client`）。pnpm 由 corepack 按 `packageManager` 字段提供（11.10.0）。
- **生成的文件不手写、不手改、不从本计划复制**：执行步骤中的生成命令，再提交它的输出。每个生成步骤都给出预期的行数和 SHA-256（`shasum -a 256`）；对不上时停下来，说明某个输入文件与本计划不一致。
- 每个改动了 Go 代码的 Task 提交前：`make lint` 必须输出 `0 issues.`（Task 8 起还包括 TS 类型检查），`make test` 必须全部通过；Task 2、Task 7 只改动 Node 和接口描述，按各自的步骤检查。`make test` 需要 Docker 在运行（testcontainers）。**只能通过 testcontainers 和 `make dev-db` 使用 Docker，不要停止或改动其他任何容器。** 开发库：`make dev-db`，容器 `nerve-dev-db-1`，端口 55432。
- 规则：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内。
- 代码注释用英文；配置文件、YAML、Makefile 中的注释用中文（与 P1、P2 一致）。
- 所有代码块都是完整的文件内容（"修改"步骤除外，它给出"把……替换为……"的原文），照原样写入，不要改动。**Makefile 的命令行以 Tab 开头**：写入后用 `grep -c "$(printf '\t')" Makefile` 核对（Task 8 给出预期值）；Tab 变成空格时 make 会报 `missing separator`。
- Makefile 必须兼容 macOS 自带的 GNU Make 3.81。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明（`cd server && …` 这类命令只在该行内切换目录）。
- 开始之前执行一次 `corepack enable` 和 `pnpm install --frozen-lockfile`（P1 已要求）。

## 文件结构

路径相对于仓库根目录。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/platform/httpserver/problem.go`（修改） | 新增 `CodeBadRequest` | 1 |
| `server/internal/platform/httpserver/apierrors.go`、`apierrors_test.go` | `APIErrors`：生成代码的错误出口输出 problem+json | 1 |
| `package.json`（修改）、`pnpm-lock.yaml`（生成） | 根目录的开发依赖 `@redocly/cli` | 2 |
| `api/openapi.yaml`、`api/common.yaml`、`api/modules/instance.yaml`、`api/redocly.yaml` | 接口描述和打包配置 | 2 |
| `api/dist/openapi.yaml`（生成） | 打包结果 | 2 |
| `server/internal/platform/httpserver/apigen/oapi-codegen.yaml`、`components.gen.go`（生成） | 公共组件的 Go 类型 | 3 |
| `server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`、`server.gen.go`（生成） | instance 的接口层 | 3 |
| `server/go.mod`、`server/go.sum`（修改） | kin-openapi | 4 |
| `server/internal/platform/httpserver/apitest/apitest.go`、`apitest_test.go` | 契约校验工具（只被测试导入） | 4 |
| `server/internal/platform/httpserver/contract_test.go` | 平台写出的 problem 的契约测试 | 4 |
| `server/internal/archtest/rules_test.go`、`rules_cases_test.go`（修改） | 规则 8 扩展到 `apitest` | 4 |
| `server/internal/modules/instance/domain/info.go` | 值对象和常量 | 5 |
| `server/internal/modules/instance/app/ports.go`、`get_info.go`、`get_info_test.go` | 端口和用例 | 5 |
| `server/internal/modules/instance/adapter/buildinfo/source.go`、`source_test.go` | `InfoSource` 的实现 | 5 |
| `server/internal/modules/instance/adapter/http/handler.go`、`handler_test.go` | 实现生成的接口，注册路由 | 5 |
| `server/internal/modules/instance/module.go` | 模块入口 | 5 |
| `server/internal/bootstrap/app.go`（修改）、`api_test.go` | 接线和测试 | 6 |
| `web/packages/api-client/package.json`、`tsconfig.json`、`src/index.ts`、`test/client.typecheck.ts` | TS 客户端 | 7 |
| `web/packages/api-client/src/schema.gen.ts`（生成）、`pnpm-lock.yaml`（生成） | 生成的类型；锁文件 | 7 |
| `Makefile`（修改） | `gen`、`gen-check`、`lint` 及按区域拆分的命令 | 8 |
| `.github/workflows/ci.yml`（修改） | 按区域的检查；同仓 PR 不重复运行 | 8 |
| `README.md`（修改） | 接口与代码生成 | 9 |
| `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-ci-split.md`、`P2-server-platform-p3-notes.md`（修改） | 交接事项改为 done | 9 |

---

### Task 1: 平台错误码 `bad_request` 与 `APIErrors`

**Files:**
- Modify: `server/internal/platform/httpserver/problem.go`
- Create: `server/internal/platform/httpserver/apierrors.go`
- Test: `server/internal/platform/httpserver/apierrors_test.go`

**Interfaces:**
- Consumes: 包内的 `WriteProblem`、`Problem`、`CodeInternal`、`requestID(ctx)`、`withRequestID`；测试辅助函数 `serve`、`captureLogs`、`findLog`（都在 `middleware_test.go`）
- Produces:
  - `const CodeBadRequest = "bad_request"`
  - `type APIErrors struct{ /* logger */ }`、`func NewAPIErrors(logger *slog.Logger) APIErrors`
  - `func (APIErrors) BadRequest(w http.ResponseWriter, r *http.Request, err error)`：400 `bad_request`，`detail` 为 `err.Error()`
  - `func (e APIErrors) InternalError(w http.ResponseWriter, r *http.Request, err error)`：记录 error 日志，返回不带 `detail` 的 500 `internal_error`
  - Task 5 把它们接到生成代码的三个错误出口上

- [ ] **Step 1: 写入失败的测试 `server/internal/platform/httpserver/apierrors_test.go`**

```go
package httpserver

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestAPIErrorsInternalErrorLogsAndHidesTheError(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	h := withRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.InternalError(w, r, errors.New("dial tcp 10.0.0.5:5432: connection refused"))
	}))
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
	if entry == nil || entry["error"] != "dial tcp 10.0.0.5:5432: connection refused" ||
		entry["request_id"] != "req-7" || entry["method"] != "GET" || entry["path"] != "/api/v0/things" {
		t.Errorf("log = %v, want the error with request_id, method and path", entry)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/httpserver/`
Expected: 编译失败：

```
internal/platform/httpserver/apierrors_test.go:12:10: undefined: NewAPIErrors
internal/platform/httpserver/apierrors_test.go:30:10: undefined: NewAPIErrors
FAIL	github.com/open-nerve/NerveProject/server/internal/platform/httpserver [build failed]
```

- [ ] **Step 3: 写入 `server/internal/platform/httpserver/problem.go`（完整内容；只在错误码常量中加了 `CodeBadRequest`）**

```go
// Package httpserver provides nerve's HTTP platform: the fixed middleware
// chain, problem+json errors, health endpoints and the server lifecycle.
package httpserver

import (
	"encoding/json"
	"net/http"
)

// ContentTypeProblem is the media type of RFC 9457 problem details.
const ContentTypeProblem = "application/problem+json"

// Codes of the problems the platform itself reports. Module codes are
// namespaced by module, e.g. "issue.state_not_in_project".
const (
	CodeNotFound   = "not_found"
	CodeBadRequest = "bad_request"
	CodeInternal   = "internal_error"
	CodeNotReady   = "not_ready"
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

// FieldError points at one invalid field of a request.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteProblem sends p with status p.Status as application/problem+json.
func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", ContentTypeProblem)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p) // nothing useful to do if the client is gone
}
```

- [ ] **Step 4: 写入 `server/internal/platform/httpserver/apierrors.go`**

```go
package httpserver

import (
	"log/slog"
	"net/http"
)

// APIErrors answers the errors that code generated from the OpenAPI
// description reports, as problem+json instead of oapi-codegen's default
// text/plain http.Error. Every module wires it into its generated handler:
//
//   - StdHTTPServerOptions.ErrorHandlerFunc: BadRequest
//   - StrictHTTPServerOptions.RequestErrorHandlerFunc: BadRequest
//   - StrictHTTPServerOptions.ResponseErrorHandlerFunc: InternalError
type APIErrors struct {
	logger *slog.Logger
}

// NewAPIErrors returns APIErrors that log internal errors to logger.
func NewAPIErrors(logger *slog.Logger) APIErrors {
	return APIErrors{logger: logger}
}

// BadRequest answers 400 bad_request for a request whose parameters or body
// the generated code could not bind or decode; err says what was wrong and
// becomes the detail.
func (APIErrors) BadRequest(w http.ResponseWriter, _ *http.Request, err error) {
	WriteProblem(w, Problem{
		Status: http.StatusBadRequest,
		Code:   CodeBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
		Detail: err.Error(),
	})
}

// InternalError logs err and answers 500 internal_error. Like a recovered
// panic, the answer carries no detail: err may describe internals.
func (e APIErrors) InternalError(w http.ResponseWriter, r *http.Request, err error) {
	e.logger.ErrorContext(r.Context(), "API handler failed",
		slog.String("request_id", requestID(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	)
	WriteProblem(w, Problem{
		Status: http.StatusInternalServerError,
		Code:   CodeInternal,
		Title:  http.StatusText(http.StatusInternalServerError),
	})
}
```

- [ ] **Step 5: 运行测试，确认通过**

Run: `cd server && go test -run TestAPIErrors -v ./internal/platform/httpserver/`
Expected: `--- PASS: TestAPIErrorsBadRequest`、`--- PASS: TestAPIErrorsInternalErrorLogsAndHidesTheError`，最后一行 `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver`。

- [ ] **Step 6: lint 和全部测试**

Run: `make lint`
Expected: `0 issues.`

Run: `make test`
Expected: 所有包都是 `ok`。

Run: `git status --short -uall`
Expected:

```
 M server/internal/platform/httpserver/problem.go
?? server/internal/platform/httpserver/apierrors.go
?? server/internal/platform/httpserver/apierrors_test.go
```

- [ ] **Step 7: 提交**

```bash
git add server/internal/platform/httpserver/problem.go server/internal/platform/httpserver/apierrors.go server/internal/platform/httpserver/apierrors_test.go
git commit -m "feat(server): answer errors of generated API code as problem+json

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: 接口描述与 Redocly 打包

**Files:**
- Modify: `package.json`
- Create: `api/openapi.yaml`、`api/common.yaml`、`api/modules/instance.yaml`、`api/redocly.yaml`
- Generate: `pnpm-lock.yaml`（`pnpm install`）、`api/dist/openapi.yaml`（`redocly bundle`）

**Interfaces:**
- Consumes: P1 的 pnpm 工作区（`packageManager: pnpm@11.10.0`）
- Produces:
  - `api/dist/openapi.yaml`：OpenAPI 3.1，路径 `GET /api/v0/instance`（`operationId: getInstance`），组件 `schemas.{FieldError, Problem, InstanceInfo}`、`responses.Problem`。Task 4 的契约测试、Task 7 的 TS 类型都以它为准
  - 命令 `pnpm exec redocly`（Task 8 的 `make gen-web` 使用）

- [ ] **Step 1: 写入 `package.json`（完整内容；加入开发依赖 `@redocly/cli`）**

```json
{
  "name": "nerve",
  "private": true,
  "license": "AGPL-3.0-only",
  "engines": {
    "node": "^24"
  },
  "packageManager": "pnpm@11.10.0+sha512.0b7f8b98060031904c017e3a41eb187a16d40eeb829b95c4f8cb03681761fc4ab53dd219115b9b447f4dce1a05a214764461e7d3703392a9f32f9511ce8c86c8",
  "devDependencies": {
    "@redocly/cli": "2.53.3"
  }
}
```

- [ ] **Step 2: 安装，更新锁文件**

Run: `pnpm install`
Expected: 输出中有 `+ @redocly/cli 2.53.3` 和 `Done in …`。`pnpm-lock.yaml` 的 `importers` 一节变为：

```yaml
importers:

  .:
    devDependencies:
      '@redocly/cli':
        specifier: 2.53.3
        version: 2.53.3
```

（`@redocly/cli` 2.53.3 没有依赖，`packages` 和 `snapshots` 中只有它一项。）

Run: `pnpm exec redocly --version`
Expected: `2.53.3`

- [ ] **Step 3: 写入 `api/common.yaml`**

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
            Stable error code. Platform codes have no prefix (not_found,
            bad_request, internal_error, not_ready); module codes are prefixed
            with the module, e.g. issue.state_not_in_project.
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
      description: One invalid field of a request.
      type: object
      additionalProperties: false
      required: [field, message]
      properties:
        field:
          type: string
        message:
          type: string
```

- [ ] **Step 4: 写入 `api/modules/instance.yaml`**

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

- [ ] **Step 5: 写入 `api/openapi.yaml`**

```yaml
# Nerve 接口描述的入口：列出所有路径，指向各模块的描述文件。
# make gen-web 用 Redocly 把它打包成 dist/openapi.yaml。
openapi: 3.1.0
info:
  title: Nerve API
  version: v0
  description: >-
    The HTTP API of Nerve. The web app is one client of it; every account can
    use every endpoint the same way.
  license:
    name: AGPL-3.0-only
    identifier: AGPL-3.0-only
tags:
  - name: instance
    description: What this instance runs.
paths:
  /api/v0/instance:
    $ref: 'modules/instance.yaml#/paths/~1api~1v0~1instance'
```

- [ ] **Step 6: 写入 `api/redocly.yaml`**

```yaml
# Redocly CLI 配置：make gen-web 执行 redocly bundle --config api/redocly.yaml。
# 路径相对于本文件所在的目录。
telemetry: off
apis:
  nerve:
    root: openapi.yaml
    output: dist/openapi.yaml
```

- [ ] **Step 7: 打包，核对生成的 `api/dist/openapi.yaml`**

Run: `REDOCLY_SUPPRESS_UPDATE_NOTICE=true pnpm exec redocly bundle --config api/redocly.yaml`
Expected:

```
bundling api/openapi.yaml using configuration for api 'nerve'...
📦 Created a bundle for api/openapi.yaml at <仓库路径>/api/dist/openapi.yaml <n>ms.
```

Run: `wc -l api/dist/openapi.yaml && shasum -a 256 api/dist/openapi.yaml`
Expected:

```
     102 api/dist/openapi.yaml
6059a92815de2336503a73073e3d375ccfb2101437de3b24cde9bfd6c2162d56  api/dist/openapi.yaml
```

再执行一次打包命令，SHA-256 不变（打包结果是确定的）。

- [ ] **Step 8: 核对改动范围**

本 Task 没有改动 Go 代码，不需要执行 `make lint` 和 `make test`。

Run: `git status --short -uall`
Expected:

```
 M package.json
 M pnpm-lock.yaml
?? api/common.yaml
?? api/dist/openapi.yaml
?? api/modules/instance.yaml
?? api/openapi.yaml
?? api/redocly.yaml
```

- [ ] **Step 9: 提交**

```bash
git add package.json pnpm-lock.yaml api
git commit -m "feat(api): add the OpenAPI 3.1 contract and its Redocly bundle

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 生成 Go 代码：共享包 apigen 与 instance 的接口层

**Files:**
- Create: `server/internal/platform/httpserver/apigen/oapi-codegen.yaml`、`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`
- Generate: `server/internal/platform/httpserver/apigen/components.gen.go`、`server/internal/modules/instance/adapter/http/gen/server.gen.go`

**Interfaces:**
- Consumes: Task 2 的 `api/common.yaml`、`api/modules/instance.yaml`；P1 的 `go tool -modfile=tools/go.mod oapi-codegen`（v2.8.0）
- Produces（生成的代码，Task 5 使用）：
  - 包 `apigen`：`type Problem struct{ Code string; Detail *string; Errors *[]FieldError; Status int; Title string }`、`type FieldError struct{ Field, Message string }`
  - 包 `gen`：`type InstanceInfo struct{ APIVersion InstanceInfoAPIVersion; Commit, Product, Version string }`、`type InstanceInfoAPIVersion string`、`const InstanceInfoAPIVersionV0`、`type StrictServerInterface interface{ GetInstance(ctx, GetInstanceRequestObject) (GetInstanceResponseObject, error) }`、`type GetInstance200JSONResponse InstanceInfo`、`func NewStrictHandlerWithOptions(StrictServerInterface, []StrictMiddlewareFunc, StrictHTTPServerOptions) ServerInterface`、`func HandlerWithOptions(ServerInterface, StdHTTPServerOptions) http.Handler`、`type StdHTTPServerOptions struct{ BaseURL string; BaseRouter ServeMux; Middlewares []MiddlewareFunc; ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error) }`、`type StrictHTTPServerOptions struct{ RequestErrorHandlerFunc, ResponseErrorHandlerFunc func(http.ResponseWriter, *http.Request, error) }`

- [ ] **Step 1: 写入 `server/internal/platform/httpserver/apigen/oapi-codegen.yaml`**

```yaml
# oapi-codegen 配置：api/common.yaml → components.gen.go（只有类型）。
# 由 make gen-go 在 server/ 下执行，路径相对于 server/。
package: apigen
output: internal/platform/httpserver/apigen/components.gen.go
generate:
  models: true
output-options:
  # common.yaml 没有 paths：默认的裁剪会把没被路径引用的组件全部删掉
  skip-prune: true
  name-normalizer: ToCamelCaseWithInitialisms
```

- [ ] **Step 2: 写入 `server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`**

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
output-options:
  name-normalizer: ToCamelCaseWithInitialisms
import-mapping:
  # common.yaml 的组件生成在共享包 apigen 里，这里只引用
  ../common.yaml: github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen
```

- [ ] **Step 3: 生成公共组件，核对输出**

Run: `cd server && go tool -modfile=tools/go.mod oapi-codegen -config internal/platform/httpserver/apigen/oapi-codegen.yaml ../api/common.yaml`
Expected: 没有输出，退出码 0。

Run: `wc -l server/internal/platform/httpserver/apigen/components.gen.go && shasum -a 256 server/internal/platform/httpserver/apigen/components.gen.go`
Expected:

```
      28 server/internal/platform/httpserver/apigen/components.gen.go
2fdb0184ac371d8c7ce99c2f11150570fa904b2b874baa7b59cbac78bf7faa9b  server/internal/platform/httpserver/apigen/components.gen.go
```

- [ ] **Step 4: 生成 instance 的接口层，核对输出**

Run: `cd server && go tool -modfile=tools/go.mod oapi-codegen -config internal/modules/instance/adapter/http/gen/oapi-codegen.yaml ../api/modules/instance.yaml`
Expected: 没有输出，退出码 0。

Run: `wc -l server/internal/modules/instance/adapter/http/gen/server.gen.go && shasum -a 256 server/internal/modules/instance/adapter/http/gen/server.gen.go`
Expected:

```
     321 server/internal/modules/instance/adapter/http/gen/server.gen.go
106c4214c9f5aa5c7813f50444a351dac1edb03eedee3e29407c1da192cfb3de  server/internal/modules/instance/adapter/http/gen/server.gen.go
```

Run: `grep -n 'externalRef0 "' server/internal/modules/instance/adapter/http/gen/server.gen.go`
Expected（import-mapping 生效：公共组件引用共享包，没有在模块里重复生成）：

```
15:	externalRef0 "github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
```

- [ ] **Step 5: 编译、lint、测试**

Run: `cd server && go build ./...`
Expected: 没有输出。生成的代码只导入标准库和 `apigen`，`server/go.mod` 不变。

Run: `make lint`
Expected: `0 issues.`（生成的文件带 `Code generated … DO NOT EDIT.`，golangci-lint 自动跳过）

Run: `make test`
Expected: 所有包都是 `ok`；`apigen` 和 `gen` 两个包显示 `[no test files]`。

Run: `git status --short -uall`
Expected:

```
?? server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml
?? server/internal/modules/instance/adapter/http/gen/server.gen.go
?? server/internal/platform/httpserver/apigen/components.gen.go
?? server/internal/platform/httpserver/apigen/oapi-codegen.yaml
```

- [ ] **Step 6: 提交**

```bash
git add server/internal/platform/httpserver/apigen server/internal/modules/instance/adapter/http/gen
git commit -m "feat(server): generate the instance API layer and the shared apigen package

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: 契约校验工具 apitest、平台 problem 的契约测试、架构规则 8

**Files:**
- Modify: `server/go.mod`、`server/go.sum`
- Create: `server/internal/platform/httpserver/apitest/apitest.go`
- Test: `server/internal/platform/httpserver/apitest/apitest_test.go`、`server/internal/platform/httpserver/contract_test.go`
- Modify: `server/internal/archtest/rules_test.go`、`server/internal/archtest/rules_cases_test.go`

**Interfaces:**
- Consumes: Task 2 的 `api/dist/openapi.yaml`；Task 1 的 `APIErrors`；P2 的 `NewMux`、`Check`、`middleware`、`WriteProblem`
- Produces（包 `apitest`，只被测试导入）：
  - `type Contract struct{ /* 文档和路由 */ }`
  - `func Load(t testing.TB) *Contract`：读取并校验 `api/dist/openapi.yaml`
  - `func (c *Contract) CheckResponse(t testing.TB, req *http.Request, res *http.Response)`：按 `req` 找到操作，校验状态码、`Content-Type`、响应体；`res.Body` 仍可读
  - `func (c *Contract) CheckSchema(t testing.TB, name string, body []byte)`：按 `components.schemas[name]` 校验
  - Task 5、Task 6 的测试使用它们

- [ ] **Step 1: 加入 kin-openapi**

Run: `cd server && go get github.com/getkin/kin-openapi@v0.149.0`
Expected: `go: added github.com/getkin/kin-openapi v0.149.0`

- [ ] **Step 2: 写入失败的测试 `server/internal/platform/httpserver/apitest/apitest_test.go`**

```go
package apitest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

const instanceJSON = `{"product":"Nerve","version":"0.1.0-dev","commit":"unknown","api_version":"v0"}`

// Component names become Go and TypeScript type names. Redocly's bundler
// renames a clash between module files to "Name-2" and only warns, so a
// clash has to fail here instead.
func TestComponentNamesAreTypeNames(t *testing.T) {
	c := Load(t)
	valid := regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	var names []string
	for name := range c.doc.Components.Schemas {
		names = append(names, "schemas/"+name)
	}
	for name := range c.doc.Components.Responses {
		names = append(names, "responses/"+name)
	}
	for name := range c.doc.Components.Parameters {
		names = append(names, "parameters/"+name)
	}
	for name := range c.doc.Components.RequestBodies {
		names = append(names, "requestBodies/"+name)
	}
	if len(names) == 0 {
		t.Fatal("the contract has no components")
	}
	for _, name := range names {
		if _, base, _ := strings.Cut(name, "/"); !valid.MatchString(base) {
			t.Errorf("component %s is not a PascalCase type name; two module files may define it differently", name)
		}
	}
}

func TestValidateResponse(t *testing.T) {
	c := Load(t)
	tests := []struct {
		name, method, path string
		status             int
		contentType, body  string
		valid              bool
	}{
		{"documented 200", "GET", "/api/v0/instance", 200, "application/json", instanceJSON, true},
		{"problem as default", "GET", "/api/v0/instance", 500, "application/problem+json", `{"status":500,"code":"internal_error","title":"Internal Server Error"}`, true},
		{"missing field", "GET", "/api/v0/instance", 200, "application/json", `{"product":"Nerve","version":"0.1.0-dev","api_version":"v0"}`, false},
		{"value outside the enum", "GET", "/api/v0/instance", 200, "application/json", strings.Replace(instanceJSON, `"v0"`, `"v1"`, 1), false},
		{"undocumented field", "GET", "/api/v0/instance", 200, "application/json", strings.Replace(instanceJSON, `}`, `,"extra":1}`, 1), false},
		{"undocumented content type", "GET", "/api/v0/instance", 200, "text/plain", "Nerve", false},
		{"problem without code", "GET", "/api/v0/instance", 500, "application/problem+json", `{"status":500,"title":"Internal Server Error"}`, false},
		{"undocumented path", "GET", "/api/v0/nope", 404, "application/problem+json", `{"status":404,"code":"not_found","title":"Not Found"}`, false},
		{"undocumented method", "POST", "/api/v0/instance", 404, "application/problem+json", `{"status":404,"code":"not_found","title":"Not Found"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			header := http.Header{"Content-Type": {tt.contentType}}
			err := c.validateResponse(req, tt.status, header, []byte(tt.body))
			if (err == nil) != tt.valid {
				t.Errorf("validateResponse() = %v, want valid = %v", err, tt.valid)
			}
		})
	}
}

func TestCheckResponseLeavesTheBodyReadable(t *testing.T) {
	c := Load(t)
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteHeader(http.StatusOK)
	_, _ = rec.WriteString(instanceJSON)
	res := rec.Result()

	c.CheckResponse(t, httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil), res)

	body, err := io.ReadAll(res.Body)
	if err != nil || string(body) != instanceJSON {
		t.Errorf("body after CheckResponse = %q, %v; want %s", body, err, instanceJSON)
	}
}

func TestValidateSchema(t *testing.T) {
	c := Load(t)
	tests := []struct {
		name, schema, body string
		valid              bool
	}{
		{"problem", "Problem", `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}`, true},
		{"problem with field errors", "Problem", `{"status":422,"code":"issue.invalid","title":"Unprocessable Content","errors":[{"field":"name","message":"is required"}]}`, true},
		{"problem without code", "Problem", `{"status":404,"title":"Not Found"}`, false},
		{"problem with an unknown member", "Problem", `{"status":404,"code":"not_found","title":"Not Found","instance":"/x"}`, false},
		{"unknown schema", "Nope", `{}`, false},
		{"not JSON", "Problem", `not json`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validateSchema(tt.schema, []byte(tt.body))
			if (err == nil) != tt.valid {
				t.Errorf("validateSchema() = %v, want valid = %v", err, tt.valid)
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/httpserver/apitest/`
Expected: 编译失败：

```
internal/platform/httpserver/apitest/apitest_test.go:18:7: undefined: Load
internal/platform/httpserver/apitest/apitest_test.go:44:7: undefined: Load
internal/platform/httpserver/apitest/apitest_test.go:74:7: undefined: Load
internal/platform/httpserver/apitest/apitest_test.go:90:7: undefined: Load
FAIL	github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest [build failed]
```

- [ ] **Step 4: 写入 `server/internal/platform/httpserver/apitest/apitest.go`**

```go
// Package apitest checks HTTP responses against nerve's OpenAPI contract,
// api/dist/openapi.yaml, with kin-openapi. Only tests import it (architecture
// rule 8), so kin-openapi never reaches the nerve binary.
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
// req's operation: the status, the Content-Type and the body schema. res.Body
// stays readable for the caller.
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

// CheckSchema fails t unless body is a JSON value valid against the schema
// components.schemas[name], e.g. "Problem".
func (c *Contract) CheckSchema(t testing.TB, name string, body []byte) {
	t.Helper()
	if err := c.validateSchema(name, body); err != nil {
		t.Errorf("%s: %v", body, err)
	}
}

// contractPath locates api/dist/openapi.yaml from this file, five directories
// below the repository root (tests run without -trimpath).
func contractPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "api", "dist", "openapi.yaml")
}

func load(path string) (*Contract, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
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
	return openapi3filter.ValidateResponse(context.Background(), in)
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

- [ ] **Step 5: 整理依赖，检查 `go` 行**

Run: `cd server && go mod tidy && grep -E '^(go|toolchain) ' go.mod`
Expected:

```
go 1.27
toolchain go1.27.1
```

Run: `git diff server/go.mod`
Expected: 第一组 `require` 中新增 `github.com/getkin/kin-openapi v0.149.0`；间接依赖中新增 `github.com/go-openapi/jsonpointer v0.22.5`、`github.com/go-openapi/swag/jsonname v0.25.5`、`github.com/oasdiff/yaml v0.1.1`、`github.com/oasdiff/yaml3 v0.0.14`、`github.com/santhosh-tekuri/jsonschema/v6 v6.0.3`（都带 `// indirect`）。没有其他改动。

- [ ] **Step 6: 运行测试，确认通过**

Run: `cd server && go test -v ./internal/platform/httpserver/apitest/`
Expected: `TestComponentNamesAreTypeNames`、`TestValidateResponse`（9 个子测试）、`TestCheckResponseLeavesTheBodyReadable`、`TestValidateSchema`（6 个子测试）全部 `PASS`，最后一行 `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest`。

- [ ] **Step 7: 写入 `server/internal/platform/httpserver/contract_test.go`**

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
		{"unknown API path", NewMux(discard), "/api/v0/nope", http.StatusNotFound},
		{"not ready", NewMux(discard, notReady), "/readyz", http.StatusServiceUnavailable},
		{"panic", middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }), discard), "/api/v0/boom", http.StatusInternalServerError},
		{"bad request", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.BadRequest(w, r, errors.New("Invalid format for parameter limit"))
		}), "/api/v0/things?limit=x", http.StatusBadRequest},
		{"internal error", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.InternalError(w, r, errors.New("boom"))
		}), "/api/v0/things", http.StatusInternalServerError},
		{"field errors", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			WriteProblem(w, Problem{
				Status: http.StatusUnprocessableEntity,
				Code:   "issue.invalid",
				Title:  http.StatusText(http.StatusUnprocessableEntity),
				Detail: "the issue is invalid",
				Errors: []FieldError{{Field: "name", Message: "is required"}},
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

- [ ] **Step 8: 运行契约测试，并确认它能发现 `Problem` 与描述不一致**

Run: `cd server && go test -run TestPlatformProblemsMatchTheContract ./internal/platform/httpserver/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver`

临时把 `Problem.Code` 的 JSON 字段名改成 `error_code`：

Run:

```bash
perl -pi -e 's/json:"code"/json:"error_code"/' server/internal/platform/httpserver/problem.go
cd server && go test -run TestPlatformProblemsMatchTheContract ./internal/platform/httpserver/
```

Expected: 失败，每个子测试都报告多出来的字段，例如：

```
--- FAIL: TestPlatformProblemsMatchTheContract (…s)
    --- FAIL: TestPlatformProblemsMatchTheContract/unknown_API_path (0.00s)
        contract_test.go:53: {"status":404,"error_code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}
            : property "error_code" is unsupported
```

恢复（`problem.go` 已在 Task 1 提交）：

Run: `git checkout -- server/internal/platform/httpserver/problem.go && git status --short server/internal/platform/httpserver/problem.go`
Expected: 没有输出。

- [ ] **Step 9: 修改 `server/internal/archtest/rules_test.go`（规则 8 扩展到 `apitest`）**

把：

```go
		{"pgtest is imported only by tests", pgtestOnlyInTests},
```

替换为：

```go
		{"test helpers (pgtest, apitest) are imported only by tests", testHelpersOnlyInTests},
```

把：

```go
func pgtestOnlyInTests(_, to string) bool {
	// The graph holds no test files, so any importer is production code.
	return inModuleDir(to, "internal/platform/postgres/pgtest")
}
```

替换为：

```go
func testHelpersOnlyInTests(_, to string) bool {
	// The graph holds no test files, so any importer is production code.
	return inModuleDir(to, "internal/platform/postgres/pgtest") ||
		inModuleDir(to, "internal/platform/httpserver/apitest")
}
```

- [ ] **Step 10: 修改 `server/internal/archtest/rules_cases_test.go`（规则名和用例）**

把：

```go
		pgtest   = "pgtest is imported only by tests"
```

替换为：

```go
		testOnly = "test helpers (pgtest, apitest) are imported only by tests"
```

把：

```go
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/adapter/http/gen"), nil},
```

替换为：

```go
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/adapter/http/gen"), nil},
		{m("internal/modules/issue/adapter/http/gen"), m("internal/platform/httpserver/apigen"), nil},
```

把：

```go
		// pgtest.
		{m("internal/bootstrap"), m("internal/platform/postgres/pgtest"), []string{pgtest}},
```

替换为：

```go
		// Test helpers.
		{m("internal/bootstrap"), m("internal/platform/postgres/pgtest"), []string{testOnly}},
		{m("internal/modules/instance/adapter/http"), m("internal/platform/httpserver/apitest"), []string{testOnly}},
		{m("internal/platform/httpserver"), m("internal/platform/httpserver/apitest"), []string{testOnly}},
```

- [ ] **Step 11: 运行架构测试，并确认规则 8 能发现非测试代码导入 `apitest`**

Run: `cd server && go test ./internal/archtest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/archtest`

临时写入一个违规文件 `server/internal/platform/httpserver/probe.go`：

```go
package httpserver

import "github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"

// probe exists only to show architecture rule 8 at work; delete it after the check.
var probe = apitest.Load
```

Run: `cd server && go test ./internal/archtest/`
Expected: 失败：

```
--- FAIL: TestRepositoryFollowsArchitectureRules (…s)
    repo_test.go:55: internal/platform/httpserver imports internal/platform/httpserver/apitest: test helpers (pgtest, apitest) are imported only by tests
FAIL
```

Run: `rm server/internal/platform/httpserver/probe.go && cd server && go test ./internal/archtest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/archtest`

- [ ] **Step 12: lint、全部测试，确认生产程序不链接 kin-openapi**

Run: `make lint`
Expected: `0 issues.`

Run: `make test`
Expected: 所有包都是 `ok`。

Run: `cd server && go build -o ../bin/nerve ./cmd/nerve && go version -m ../bin/nerve | grep -E 'kin-openapi|jsonschema'; echo "exit=$?"; rm ../bin/nerve`
Expected: 只有一行 `exit=1`（grep 没有找到任何匹配）。

Run: `git status --short -uall`
Expected:

```
 M server/go.mod
 M server/go.sum
 M server/internal/archtest/rules_cases_test.go
 M server/internal/archtest/rules_test.go
?? server/internal/platform/httpserver/apitest/apitest.go
?? server/internal/platform/httpserver/apitest/apitest_test.go
?? server/internal/platform/httpserver/contract_test.go
```

- [ ] **Step 13: 提交**

```bash
git add server/go.mod server/go.sum server/internal/archtest server/internal/platform/httpserver/apitest server/internal/platform/httpserver/contract_test.go
git commit -m "test(server): check responses and platform problems against the OpenAPI contract

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: 试点模块 instance

**Files:**
- Create: `server/internal/modules/instance/domain/info.go`
- Create: `server/internal/modules/instance/app/ports.go`、`server/internal/modules/instance/app/get_info.go`
- Test: `server/internal/modules/instance/app/get_info_test.go`
- Create: `server/internal/modules/instance/adapter/buildinfo/source.go`
- Test: `server/internal/modules/instance/adapter/buildinfo/source_test.go`
- Create: `server/internal/modules/instance/adapter/http/handler.go`
- Test: `server/internal/modules/instance/adapter/http/handler_test.go`
- Create: `server/internal/modules/instance/module.go`

**Interfaces:**
- Consumes: Task 3 生成的包 `gen`；Task 1 的 `httpserver.APIErrors`；Task 4 的 `apitest`（测试）；P1 的 `platform/buildinfo.Get()`
- Produces:
  - `domain`：`const Product = "Nerve"`、`const APIVersion = "v0"`、`type Build struct{ Version, Commit string }`、`type Info struct{ Product, Version, Commit, APIVersion string }`
  - `app`：`type InfoSource interface{ Build() domain.Build }`、`type GetInfo`、`func NewGetInfo(source InfoSource) *GetInfo`、`func (uc *GetInfo) Execute() domain.Info`
  - `adapter/buildinfo`（包 `buildinfo`）：`type Source struct{}`，实现 `app.InfoSource`
  - `adapter/http`（包 `httpadapter`）：`func Register(mux *http.ServeMux, getInfo *app.GetInfo, apiErrors httpserver.APIErrors)`
  - `instance`：`type Module`、`func New() *Module`、`func (m *Module) Register(mux *http.ServeMux, apiErrors httpserver.APIErrors)`（Task 6 在 `bootstrap` 中调用）

- [ ] **Step 1: 写入 `server/internal/modules/instance/domain/info.go`**

```go
// Package domain holds the instance module's model.
package domain

// Product is the product name every instance reports.
const Product = "Nerve"

// APIVersion is the version of the HTTP API this build serves. It follows
// the product's major version (v0 design 3.1) and prefixes every API path.
const APIVersion = "v0"

// Build identifies the binary an instance runs.
type Build struct {
	Version string // product version, e.g. "0.1.0-dev"
	Commit  string // git revision, "unknown" without a VCS stamp
}

// Info is what an instance tells API clients about itself.
type Info struct {
	Product    string
	Version    string
	Commit     string
	APIVersion string
}
```

- [ ] **Step 2: 写入失败的测试 `server/internal/modules/instance/app/get_info_test.go`**

```go
package app_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

type fixedSource domain.Build

func (s fixedSource) Build() domain.Build { return domain.Build(s) }

func TestGetInfoDescribesTheBuild(t *testing.T) {
	uc := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"})

	got := uc.Execute()

	want := domain.Info{Product: "Nerve", Version: "1.2.3", Commit: "4f2a9c1", APIVersion: "v0"}
	if got != want {
		t.Errorf("Execute() = %+v, want %+v", got, want)
	}
}
```

Run: `cd server && go test ./internal/modules/instance/app/`
Expected: 失败：`no non-test Go files in …/internal/modules/instance/app`，`FAIL … [build failed]`。

- [ ] **Step 3: 写入 `server/internal/modules/instance/app/ports.go`**

```go
// Package app holds the instance module's use cases and the ports they need.
package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// InfoSource reports the build this instance runs. GetInfo declares it;
// adapter/buildinfo implements it.
type InfoSource interface {
	Build() domain.Build
}
```

- [ ] **Step 4: 写入 `server/internal/modules/instance/app/get_info.go`**

```go
package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// GetInfo tells API clients what this instance runs.
type GetInfo struct {
	source InfoSource
}

// NewGetInfo returns the use case, reading the build from source.
func NewGetInfo(source InfoSource) *GetInfo {
	return &GetInfo{source: source}
}

// Execute describes the instance.
func (uc *GetInfo) Execute() domain.Info {
	build := uc.source.Build()
	return domain.Info{
		Product:    domain.Product,
		Version:    build.Version,
		Commit:     build.Commit,
		APIVersion: domain.APIVersion,
	}
}
```

Run: `cd server && go test ./internal/modules/instance/app/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/modules/instance/app`

- [ ] **Step 5: 写入失败的测试 `server/internal/modules/instance/adapter/buildinfo/source_test.go`**

```go
package buildinfo_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	platformbuildinfo "github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
)

var _ app.InfoSource = buildinfo.Source{}

func TestSourceReportsTheRunningBuild(t *testing.T) {
	info := platformbuildinfo.Get()

	got := buildinfo.Source{}.Build()

	want := domain.Build{Version: info.Version, Commit: info.Commit}
	if got != want {
		t.Errorf("Build() = %+v, want %+v", got, want)
	}
}
```

Run: `cd server && go test ./internal/modules/instance/adapter/buildinfo/`
Expected: 失败：`no non-test Go files in …/internal/modules/instance/adapter/buildinfo`。

- [ ] **Step 6: 写入 `server/internal/modules/instance/adapter/buildinfo/source.go`**

```go
// Package buildinfo implements the instance module's InfoSource port with the
// build metadata of the running binary.
package buildinfo

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	platformbuildinfo "github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
)

// Source reports the build of the running binary.
type Source struct{}

// Build implements app.InfoSource.
func (Source) Build() domain.Build {
	info := platformbuildinfo.Get()
	return domain.Build{Version: info.Version, Commit: info.Commit}
}
```

Run: `cd server && go test ./internal/modules/instance/adapter/buildinfo/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo`

- [ ] **Step 7: 写入失败的测试 `server/internal/modules/instance/adapter/http/handler_test.go`**

```go
package httpadapter_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
	mux := http.NewServeMux()
	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"})
	httpadapter.Register(mux, getInfo, httpserver.NewAPIErrors(slog.New(slog.DiscardHandler)))
	req := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	res := rec.Result()

	contract.CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	want := `{"api_version":"v0","commit":"4f2a9c1","product":"Nerve","version":"1.2.3"}` + "\n"
	if res.StatusCode != http.StatusOK || string(body) != want {
		t.Errorf("GET /api/v0/instance = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}
```

Run: `cd server && go test ./internal/modules/instance/adapter/http/`
Expected: 失败：`no non-test Go files in …/internal/modules/instance/adapter/http`。

- [ ] **Step 8: 写入 `server/internal/modules/instance/adapter/http/handler.go`**

包名是 `httpadapter`（目录仍是 `adapter/http`）：包名 `http` 会遮住标准库 `net/http`。

```go
// Package httpadapter serves the instance module's API: it implements the
// strict server that oapi-codegen generates from api/modules/instance.yaml
// into the gen package.
package httpadapter

import (
	"context"
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Register mounts the module's routes on mux, the root router from
// httpserver.NewMux. They are more specific than the platform's /api/
// fallback, which keeps answering every other API path with a 404 problem.
// Binding and handler errors are answered as problem+json by apiErrors.
func Register(mux *http.ServeMux, getInfo *app.GetInfo, apiErrors httpserver.APIErrors) {
	strict := gen.NewStrictHandlerWithOptions(handler{getInfo: getInfo}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  apiErrors.BadRequest,
		ResponseErrorHandlerFunc: apiErrors.InternalError,
	})
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: apiErrors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface: it only translates between
// the generated types and the use cases.
type handler struct {
	getInfo *app.GetInfo
}

// GetInstance serves GET /api/v0/instance.
func (h handler) GetInstance(context.Context, gen.GetInstanceRequestObject) (gen.GetInstanceResponseObject, error) {
	info := h.getInfo.Execute()
	return gen.GetInstance200JSONResponse{
		Product:    info.Product,
		Version:    info.Version,
		Commit:     info.Commit,
		APIVersion: gen.InstanceInfoAPIVersion(info.APIVersion),
	}, nil
}
```

Run: `cd server && go test -v ./internal/modules/instance/adapter/http/`
Expected: `--- PASS: TestGetInstanceMatchesTheContract`，最后一行 `ok  	github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http`。

- [ ] **Step 9: 写入 `server/internal/modules/instance/module.go`**

```go
// Package instance is the pilot module and the template for every module
// (M0 design 3.2): GET /api/v0/instance tells API clients what this nerve
// instance runs.
package instance

import (
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Module is the wired instance module.
type Module struct {
	getInfo *app.GetInfo
}

// New wires the module: GetInfo reads the build of the running binary.
func New() *Module {
	return &Module{getInfo: app.NewGetInfo(buildinfo.Source{})}
}

// Register mounts the module's API on mux, the root router from
// httpserver.NewMux; apiErrors answers binding and handler errors.
func (m *Module) Register(mux *http.ServeMux, apiErrors httpserver.APIErrors) {
	httpadapter.Register(mux, m.getInfo, apiErrors)
}
```

- [ ] **Step 10: 编译、架构测试、lint、全部测试**

Run: `cd server && go build ./... && go test ./internal/archtest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/archtest`（instance 的 `app` 只导入本模块的 `domain`，`gen` 只被 `adapter/http` 导入）

Run: `make lint`
Expected: `0 issues.`

Run: `make test`
Expected: 所有包都是 `ok`；`instance`、`instance/domain`、`gen`、`apigen` 显示 `[no test files]`。

Run: `git status --short -uall`
Expected:

```
?? server/internal/modules/instance/adapter/buildinfo/source.go
?? server/internal/modules/instance/adapter/buildinfo/source_test.go
?? server/internal/modules/instance/adapter/http/handler.go
?? server/internal/modules/instance/adapter/http/handler_test.go
?? server/internal/modules/instance/app/get_info.go
?? server/internal/modules/instance/app/get_info_test.go
?? server/internal/modules/instance/app/ports.go
?? server/internal/modules/instance/domain/info.go
?? server/internal/modules/instance/module.go
```

- [ ] **Step 11: 提交**

```bash
git add server/internal/modules/instance
git commit -m "feat(server): add the instance pilot module

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: 在 bootstrap 中挂上 instance 模块

**Files:**
- Modify: `server/internal/bootstrap/app.go`
- Test: `server/internal/bootstrap/api_test.go`

**Interfaces:**
- Consumes: Task 5 的 `instance.New()`、`(*Module).Register`；Task 1 的 `httpserver.NewAPIErrors`；Task 4 的 `apitest`（测试）；P2 测试中的 `startApp`、`testConfig`、`unreachableDB`、`client`（`app_test.go`）
- Produces: `nerve serve` 提供 `GET /api/v0/instance`

- [ ] **Step 1: 写入失败的测试 `server/internal/bootstrap/api_test.go`**

```go
package bootstrap

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The API needs no database: the app runs against an unreachable one.
func TestServesTheInstanceAPI(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	req, err := http.NewRequest(http.MethodGet, base+"/api/v0/instance", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()

	contract.CheckResponse(t, req, res)
	var got struct {
		Product    string `json:"product"`
		Version    string `json:"version"`
		APIVersion string `json:"api_version"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.StatusCode != http.StatusOK || got.Product != "Nerve" || got.Version != buildinfo.Get().Version || got.APIVersion != "v0" {
		t.Errorf("GET /api/v0/instance = %d %+v, want 200 Nerve %s v0", res.StatusCode, got, buildinfo.Get().Version)
	}
	if res.Header.Get(httpserver.HeaderRequestID) == "" {
		t.Error("response has no X-Request-Id: the platform middleware did not run")
	}
}

// Mounting a module keeps the platform's /api/ fallback: an unknown path and
// a known path with the wrong method both answer 404 problem+json.
func TestUnknownAPIRequestsStillAnswerProblem404(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/api/v0/nope"},
		{http.MethodPost, "/api/v0/instance"},
	} {
		req, err := http.NewRequest(tt.method, base+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != httpserver.ContentTypeProblem {
			t.Errorf("%s %s = %d %s, want 404 problem+json", tt.method, tt.path, res.StatusCode, res.Header.Get("Content-Type"))
		}
		contract.CheckSchema(t, "Problem", body)
		var p httpserver.Problem
		if err := json.Unmarshal(body, &p); err != nil || p.Detail != "no API endpoint for "+tt.method+" "+tt.path {
			t.Errorf("%s %s detail = %q (%v), want no API endpoint for %s %s", tt.method, tt.path, p.Detail, err, tt.method, tt.path)
		}
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd server && go test -short -run 'TestServesTheInstanceAPI|TestUnknownAPIRequestsStillAnswerProblem404' ./internal/bootstrap/`
Expected: `TestServesTheInstanceAPI` 失败（路由还没有挂上，请求落到平台的 `/api/` 兜底；404 problem 也符合 `default` 响应，所以契约校验本身不报错）：

```
--- FAIL: TestServesTheInstanceAPI (…s)
    api_test.go:40: GET /api/v0/instance = 404 {Product: Version: APIVersion:}, want 200 Nerve 0.1.0-dev v0
FAIL
```

- [ ] **Step 3: 写入 `server/internal/bootstrap/app.go`（完整内容；导入 `modules/instance`，在创建路由之后挂上模块）**

```go
// Package bootstrap is nerve's only composition root: it builds every adapter
// from the configuration, wires them together and runs the commands.
package bootstrap

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// app is a fully wired nerve server.
type app struct {
	cfg      config.Config
	logger   *slog.Logger
	pool     *pgxpool.Pool
	migrator *postgres.Migrator
	handler  http.Handler
}

// newApp wires the server described by cfg around the given migrations.
// close releases it.
func newApp(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles fs.FS) (*app, error) {
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	migrator, err := postgres.NewMigrator(pool, migrationFiles)
	if err != nil {
		pool.Close()
		return nil, err
	}
	mux := httpserver.NewMux(logger,
		httpserver.Check{Name: "database", Run: pool.Ping},
		httpserver.Check{Name: "migrations", Run: migrator.CheckUpToDate},
	)
	// Modules mount their generated routes on this root mux, next to the
	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
	instance.New().Register(mux, httpserver.NewAPIErrors(logger))
	return &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator, handler: mux}, nil
}

// run applies pending migrations when database.auto_migrate is on, then
// serves HTTP on server.addr until ctx is done (see httpserver.Server.Serve).
func (a *app) run(ctx context.Context) error {
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
	return httpserver.NewServer(a.cfg.Server, a.handler, a.logger).ListenAndServe(ctx)
}

// close releases the database resources. Call it after run has returned.
func (a *app) close() {
	if err := a.migrator.Close(); err != nil {
		a.logger.Warn("close migrator", slog.Any("error", err))
	}
	a.pool.Close()
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd server && go test -short -v -run 'TestServesTheInstanceAPI|TestUnknownAPIRequestsStillAnswerProblem404' ./internal/bootstrap/`
Expected: 两个测试都 `PASS`，最后一行 `ok  	github.com/open-nerve/NerveProject/server/internal/bootstrap`。

- [ ] **Step 5: lint 和全部测试**

Run: `make lint`
Expected: `0 issues.`

Run: `make test`
Expected: 所有包都是 `ok`。

- [ ] **Step 6: 手工验证**

Run: `make dev-db`
Expected: `Container nerve-dev-db-1  Healthy`

执行 `make run`，在另一个终端：

```bash
curl -si localhost:8080/api/v0/instance; echo
curl -si localhost:8080/api/v0/nope; echo
curl -si -X POST localhost:8080/api/v0/instance; echo
```

Expected（`X-Request-Id`、`Date` 每次不同）：

```
HTTP/1.1 200 OK
Content-Type: application/json
…
{"api_version":"v0","commit":"unknown","product":"Nerve","version":"0.1.0-dev"}

HTTP/1.1 404 Not Found
Content-Type: application/problem+json
…
{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}

HTTP/1.1 404 Not Found
Content-Type: application/problem+json
…
{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for POST /api/v0/instance"}
```

（`go run` 构建的程序没有 VCS 信息，所以 `commit` 是 `unknown`；方法不对返回 404 是 spec 2.6 的决定。）在 `make run` 的终端按 Ctrl-C，输出 `msg="http server stopped"` 后退出。

- [ ] **Step 7: 提交**

```bash
git add server/internal/bootstrap/app.go server/internal/bootstrap/api_test.go
git commit -m "feat(server): serve GET /api/v0/instance

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: TS 客户端 web/packages/api-client

**Files:**
- Create: `web/packages/api-client/package.json`、`web/packages/api-client/tsconfig.json`、`web/packages/api-client/src/index.ts`
- Test: `web/packages/api-client/test/client.typecheck.ts`（只参与类型检查）
- Generate: `web/packages/api-client/src/schema.gen.ts`、`pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 2 的 `api/dist/openapi.yaml`
- Produces:
  - 包 `@nerve/api-client`：`createClient(options?: ClientOptions)`（openapi-fetch 的 `createClient<paths>`），类型 `paths`、`components`
  - 包脚本 `gen`（生成 `src/schema.gen.ts`）和 `typecheck`（`tsc --noEmit`），Task 8 的 `make gen-web`、`make lint-web` 调用它们

- [ ] **Step 1: 写入 `web/packages/api-client/package.json`**

```json
{
  "name": "@nerve/api-client",
  "private": true,
  "license": "AGPL-3.0-only",
  "description": "Typed client for the Nerve API, generated from api/dist/openapi.yaml",
  "type": "module",
  "exports": {
    ".": "./src/index.ts"
  },
  "scripts": {
    "gen": "openapi-typescript ../../../api/dist/openapi.yaml --output src/schema.gen.ts",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "openapi-fetch": "0.17.0"
  },
  "devDependencies": {
    "openapi-typescript": "7.13.0",
    "typescript": "5.8.3"
  }
}
```

- [ ] **Step 2: 写入 `web/packages/api-client/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022", "DOM"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "verbatimModuleSyntax": true,
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "skipLibCheck": true,
    "noEmit": true
  },
  "include": ["src", "test"]
}
```

- [ ] **Step 3: 安装，更新锁文件**

Run: `pnpm install`
Expected: 输出中有 `Scope: all 2 workspace projects` 和 `Done in …`。`pnpm-lock.yaml` 的 `importers` 一节变为：

```yaml
importers:

  .:
    devDependencies:
      '@redocly/cli':
        specifier: 2.53.3
        version: 2.53.3

  web/packages/api-client:
    dependencies:
      openapi-fetch:
        specifier: 0.17.0
        version: 0.17.0
    devDependencies:
      openapi-typescript:
        specifier: 7.13.0
        version: 7.13.0(typescript@5.8.3)
      typescript:
        specifier: 5.8.3
        version: 5.8.3
```

（`packages` 一节中间接依赖的版本取决于安装时 npm 上的最新版本，可能与本计划写作时不同，这没有关系：以提交的锁文件为准。）

- [ ] **Step 4: 生成 TS 类型，核对输出**

Run: `pnpm --filter @nerve/api-client gen`
Expected:

```
$ openapi-typescript ../../../api/dist/openapi.yaml --output src/schema.gen.ts
✨ openapi-typescript 7.13.0
🚀 ../../../api/dist/openapi.yaml → src/schema.gen.ts [<n>ms]
```

Run: `wc -l web/packages/api-client/src/schema.gen.ts && shasum -a 256 web/packages/api-client/src/schema.gen.ts`
Expected:

```
     108 web/packages/api-client/src/schema.gen.ts
6d07937e1389c39a2cc3465ac272f09afa84583dc547c0849322802678544638  web/packages/api-client/src/schema.gen.ts
```

- [ ] **Step 5: 写入 `web/packages/api-client/src/index.ts`**

```ts
import createFetchClient, { type ClientOptions } from "openapi-fetch";

import type { paths } from "./schema.gen";

export type { components, paths } from "./schema.gen";

/**
 * Creates a client for the Nerve API. Paths, parameters, request bodies and
 * responses are typed from api/dist/openapi.yaml; error bodies are
 * problem+json, components["schemas"]["Problem"].
 */
export function createClient(options?: ClientOptions) {
  return createFetchClient<paths>(options);
}
```

- [ ] **Step 6: 写入 `web/packages/api-client/test/client.typecheck.ts`**

```ts
// Compile-time checks, run by `pnpm typecheck`: tsc fails when the generated
// types stop matching how clients call the API. Nothing here is executed.
import { createClient, type components } from "../src/index";

type InstanceInfo = components["schemas"]["InstanceInfo"];

export async function describeInstance(baseUrl: string): Promise<string> {
  const client = createClient({ baseUrl });
  const { data, error } = await client.GET("/api/v0/instance");
  if (error) {
    // Error bodies are problem+json; clients branch on code.
    const code: string = error.code;
    throw new Error(`${error.status} ${code}`);
  }
  const info: InstanceInfo = data;
  const apiVersion: "v0" = info.api_version;
  return `${info.product} ${info.version} (${info.commit}), API ${apiVersion}`;
}

export async function rejectsUndocumentedCalls(baseUrl: string): Promise<void> {
  const client = createClient({ baseUrl });
  // @ts-expect-error: /api/v0/nope is not in the contract
  await client.GET("/api/v0/nope");
  // @ts-expect-error: /api/v0/instance has no POST
  await client.POST("/api/v0/instance");
}
```

- [ ] **Step 7: 类型检查，并确认它能发现错误的调用**

Run: `pnpm -r run typecheck`
Expected: 输出 `$ tsc --noEmit`，退出码 0。两处 `@ts-expect-error` 确实各拦住了一个错误，否则 tsc 会报 `Unused '@ts-expect-error' directive`。

临时写入 `web/packages/api-client/test/negative.typecheck.ts`：

```ts
import { createClient } from "../src/index";

// Must not compile: /api/v0/instance has no POST.
export const wrong = () => createClient().POST("/api/v0/instance");
```

Run: `pnpm -r run typecheck`
Expected: 失败：

```
test/negative.typecheck.ts(4,48): error TS2345: Argument of type '"/api/v0/instance"' is not assignable to parameter of type 'never'.
…
[ERR_PNPM_RECURSIVE_RUN_FIRST_FAIL] @nerve/api-client@ typecheck: `tsc --noEmit`
```

Run: `rm web/packages/api-client/test/negative.typecheck.ts && pnpm -r run typecheck`
Expected: 退出码 0。

- [ ] **Step 8: 核对锁文件和改动范围**

Run: `pnpm install --frozen-lockfile`
Expected: `Done in …`（锁文件与 `package.json` 一致）。

Go 代码没有改动，不需要执行 `make lint` 和 `make test`。

Run: `git status --short -uall`
Expected:

```
 M pnpm-lock.yaml
?? web/packages/api-client/package.json
?? web/packages/api-client/src/index.ts
?? web/packages/api-client/src/schema.gen.ts
?? web/packages/api-client/test/client.typecheck.ts
?? web/packages/api-client/tsconfig.json
```

- [ ] **Step 9: 提交**

```bash
git add pnpm-lock.yaml web/packages/api-client
git commit -m "feat(web): add the generated TypeScript API client package

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Makefile 的 gen、gen-check、lint 按区域拆分，持续集成

**Files:**
- Modify: `Makefile`、`.github/workflows/ci.yml`

**Interfaces:**
- Consumes: Task 2–7 的生成命令和包脚本
- Produces: `make gen`、`gen-go`、`gen-web`、`gen-check`、`gen-check-go`、`gen-check-web`、`lint`、`lint-go`、`lint-web`；持续集成的 `server` 任务只需要 Go，`web` 任务只需要 Node

- [ ] **Step 1: 写入 `Makefile`（完整内容；命令行以 Tab 开头）**

```makefile
# Nerve 开发命令入口。运行 `make` 或 `make help` 查看所有命令。
# 需兼容 macOS 自带的 GNU Make 3.81。
# 命令按区域分组：*-go 只需要 Go，*-web 需要 Node（先执行 pnpm install）；
# 不带后缀的 gen、gen-check、lint 依次执行两个区域，供本地使用。

SHELL := /bin/bash
.DEFAULT_GOAL := help

DEV_COMPOSE := docker compose -f deploy/compose.dev.yaml
GOLANGCI_LINT_VERSION := 2.13.2
BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

# 代码生成（见 docs/v0/M0-foundation/specs/P3-api-contract.md）
OAPI_CODEGEN := go tool -modfile=tools/go.mod oapi-codegen
REDOCLY := REDOCLY_SUPPRESS_UPDATE_NOTICE=true pnpm exec redocly
# 每个模块一个描述文件 api/modules/<模块>.yaml，生成到该模块的 adapter/http/gen
API_MODULES := $(basename $(notdir $(wildcard api/modules/*.yaml)))
GEN_GO_OUT := server/internal/platform/httpserver/apigen server/internal/modules/*/adapter/http/gen
GEN_WEB_OUT := api/dist web/packages/api-client/src/schema.gen.ts
# 生成物必须已提交且没有差异；$(1) 是生成物的路径
check-committed = test -z "$$(git status --porcelain -- $(1))" || { git status --short -- $(1); git --no-pager diff -- $(1); echo "生成物与接口描述不一致：执行 make gen，并提交生成的文件"; exit 1; }

.PHONY: help
help: ## 列出所有命令
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: dev-db
dev-db: ## 启动开发数据库（Postgres 18），等待就绪
	$(DEV_COMPOSE) up -d --wait db

.PHONY: dev-db-down
dev-db-down: ## 停止开发数据库，保留数据
	$(DEV_COMPOSE) down

.PHONY: dev-db-reset
dev-db-reset: ## 停止开发数据库并删除数据卷
	$(DEV_COMPOSE) down -v

.PHONY: run
run: ## 以 dev 配置运行后端（需先 make dev-db），Ctrl-C 停止
	cd server && NERVE_ENV=dev go run ./cmd/nerve serve

.PHONY: tools
tools: ## 安装锁定版本的 golangci-lint 到 ./bin
	@set -o pipefail; \
	if $(GOLANGCI_LINT) --version 2>/dev/null | grep -q "version $(GOLANGCI_LINT_VERSION) "; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) 已安装"; \
	else \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v$(GOLANGCI_LINT_VERSION)/install.sh | sh -s -- -b $(BIN_DIR) v$(GOLANGCI_LINT_VERSION); \
	fi

.PHONY: gen
gen: gen-go gen-web ## 重新生成全部代码：Go 接口层、api/dist、TS 客户端

.PHONY: gen-go
gen-go: ## 由 api/ 生成 Go 接口层（只需要 Go）
	rm -f server/internal/platform/httpserver/apigen/*.gen.go server/internal/modules/*/adapter/http/gen/*.gen.go
	cd server && $(OAPI_CODEGEN) -config internal/platform/httpserver/apigen/oapi-codegen.yaml ../api/common.yaml
	@set -e; for m in $(API_MODULES); do \
		echo "cd server && $(OAPI_CODEGEN) -config internal/modules/$$m/adapter/http/gen/oapi-codegen.yaml ../api/modules/$$m.yaml"; \
		(cd server && $(OAPI_CODEGEN) -config internal/modules/$$m/adapter/http/gen/oapi-codegen.yaml ../api/modules/$$m.yaml); \
	done

.PHONY: gen-web
gen-web: ## 打包 api/dist/openapi.yaml，生成 TS 客户端的类型（需要 Node）
	$(REDOCLY) bundle --config api/redocly.yaml
	pnpm --filter @nerve/api-client gen

.PHONY: gen-check
gen-check: gen-check-go gen-check-web ## 重新生成全部代码，检查生成物已提交且没有差异

.PHONY: gen-check-go
gen-check-go: gen-go ## 重新生成 Go 接口层并检查（持续集成 server 任务）
	@$(call check-committed,$(GEN_GO_OUT))

.PHONY: gen-check-web
gen-check-web: gen-web ## 重新生成 api/dist 和 TS 类型并检查（持续集成 web 任务）
	@$(call check-committed,$(GEN_WEB_OUT))

.PHONY: lint
lint: lint-go lint-web ## 运行全部静态检查

.PHONY: lint-go
lint-go: tools ## 运行 golangci-lint（server）
	cd server && $(GOLANGCI_LINT) run ./...

.PHONY: lint-web
lint-web: ## 前端类型检查（需要 Node）
	pnpm -r run typecheck

.PHONY: test
test: ## 运行 Go 测试（server）
	cd server && go test ./...
```

Run: `grep -c "$(printf '\t')" Makefile`
Expected: `24`

- [ ] **Step 2: 核对命令列表**

Run: `make`
Expected（颜色代码略去）：

```
  help           列出所有命令
  dev-db         启动开发数据库（Postgres 18），等待就绪
  dev-db-down    停止开发数据库，保留数据
  dev-db-reset   停止开发数据库并删除数据卷
  run            以 dev 配置运行后端（需先 make dev-db），Ctrl-C 停止
  tools          安装锁定版本的 golangci-lint 到 ./bin
  gen            重新生成全部代码：Go 接口层、api/dist、TS 客户端
  gen-go         由 api/ 生成 Go 接口层（只需要 Go）
  gen-web        打包 api/dist/openapi.yaml，生成 TS 客户端的类型（需要 Node）
  gen-check      重新生成全部代码，检查生成物已提交且没有差异
  gen-check-go   重新生成 Go 接口层并检查（持续集成 server 任务）
  gen-check-web  重新生成 api/dist 和 TS 类型并检查（持续集成 web 任务）
  lint           运行全部静态检查
  lint-go        运行 golangci-lint（server）
  lint-web       前端类型检查（需要 Node）
  test           运行 Go 测试（server）
```

- [ ] **Step 3: `make gen` 重现了前面提交的生成物**

Run: `make gen`
Expected: 依次打印并执行：

```
rm -f server/internal/platform/httpserver/apigen/*.gen.go server/internal/modules/*/adapter/http/gen/*.gen.go
cd server && go tool -modfile=tools/go.mod oapi-codegen -config internal/platform/httpserver/apigen/oapi-codegen.yaml ../api/common.yaml
cd server && go tool -modfile=tools/go.mod oapi-codegen -config internal/modules/instance/adapter/http/gen/oapi-codegen.yaml ../api/modules/instance.yaml
REDOCLY_SUPPRESS_UPDATE_NOTICE=true pnpm exec redocly bundle --config api/redocly.yaml
bundling api/openapi.yaml using configuration for api 'nerve'...
📦 Created a bundle for api/openapi.yaml at <仓库路径>/api/dist/openapi.yaml <n>ms.
pnpm --filter @nerve/api-client gen
$ openapi-typescript ../../../api/dist/openapi.yaml --output src/schema.gen.ts
✨ openapi-typescript 7.13.0
🚀 ../../../api/dist/openapi.yaml → src/schema.gen.ts [<n>ms]
```

Run: `git status --short`
Expected: 只有 ` M Makefile`（生成物与提交的版本完全相同）。

Run: `make gen-check >/dev/null 2>&1; echo "exit=$?"`
Expected: `exit=0`

- [ ] **Step 4: 演示：修改描述但不重新生成，`gen-check` 必须失败**

Run: `perl -pi -e 's/description: Product name\./description: Name of the product./' api/modules/instance.yaml && make gen-check-go`
Expected: 失败，打印 Go 生成物的差异：

```
 M server/internal/modules/instance/adapter/http/gen/server.gen.go
diff --git a/server/internal/modules/instance/adapter/http/gen/server.gen.go b/server/internal/modules/instance/adapter/http/gen/server.gen.go
…
-	// Product Product name.
+	// Product Name of the product.
…
生成物与接口描述不一致：执行 make gen，并提交生成的文件
make: *** [gen-check-go] Error 1
```

Run: `make gen-check-web`
Expected: 失败，`api/dist/openapi.yaml` 和 `web/packages/api-client/src/schema.gen.ts` 都有差异（`- description: Product name.` / `+ description: Name of the product.`），最后两行：

```
生成物与接口描述不一致：执行 make gen，并提交生成的文件
make: *** [gen-check-web] Error 1
```

恢复：

Run: `git checkout -- api/modules/instance.yaml && make gen >/dev/null 2>&1 && git status --short`
Expected: 只有 ` M Makefile`。

- [ ] **Step 5: 演示：新模块的生成物没有提交，`gen-check-go` 必须失败**

Run（复制出一个临时的 probe 模块：描述和生成配置）：

```bash
mkdir -p server/internal/modules/probe/adapter/http/gen
perl -pe 's#modules/instance/#modules/probe/#' server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml > server/internal/modules/probe/adapter/http/gen/oapi-codegen.yaml
perl -pe 's#/api/v0/instance#/api/v0/probe#; s#getInstance#getProbe#; s#InstanceInfo#ProbeInfo#g' api/modules/instance.yaml > api/modules/probe.yaml
make gen-check-go
```

Expected: 失败，最后几行：

```
cd server && go tool -modfile=tools/go.mod oapi-codegen -config internal/modules/probe/adapter/http/gen/oapi-codegen.yaml ../api/modules/probe.yaml
?? server/internal/modules/probe/adapter/http/gen/
生成物与接口描述不一致：执行 make gen，并提交生成的文件
make: *** [gen-check-go] Error 1
```

Run: `rm -r server/internal/modules/probe api/modules/probe.yaml && git status --short`
Expected: 只有 ` M Makefile`。

- [ ] **Step 6: 写入 `.github/workflows/ci.yml`（完整内容）**

```yaml
name: CI

on:
  push:
  pull_request:

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}

permissions:
  contents: read

jobs:
  server:
    name: server
    # 同仓库分支的 PR 已由 push 事件在同一个提交上跑过；pull_request 事件只为 fork 的 PR 运行
    if: github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: server/go.mod
          cache-dependency-path: server/**/go.sum
      - name: Go version
        working-directory: server
        run: go version
      - name: Generated code
        run: make gen-check-go
      - name: Lint
        run: make lint-go
      - name: Test
        run: make test

  web:
    name: web
    if: github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-node@v7
        with:
          node-version-file: .node-version
      - name: Enable corepack
        run: corepack enable
      - name: Install
        run: pnpm install --frozen-lockfile
      - name: Generated code
        run: make gen-check-web
      - name: Lint
        run: make lint-web
```

- [ ] **Step 7: lint 和全部测试**

Run: `make lint`
Expected: 先是 golangci-lint 的 `0 issues.`，然后 `pnpm -r run typecheck` 输出 `$ tsc --noEmit`，退出码 0。

Run: `make test`
Expected: 所有包都是 `ok`。

Run: `git status --short`
Expected:

```
 M .github/workflows/ci.yml
 M Makefile
```

- [ ] **Step 8: 提交**

```bash
git add Makefile .github/workflows/ci.yml
git commit -m "build: split gen, gen-check and lint by area and check generated code in CI

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: 文档收尾与完整走查

**Files:**
- Modify: `README.md`
- Modify: `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-ci-split.md`、`docs/v0/M0-foundation/handoffs/P2-server-platform-p3-notes.md`
- 核对（不改）：`docs/v0/M0-foundation/M0-design.md` 第 11 节

**Interfaces:**
- Consumes: Task 1–8 的全部成果
- Produces: 开发者上手说明；关闭两个交给 P3 的交接事项

- [ ] **Step 1: 修改 `README.md`（第一次启动加上 `pnpm install`）**

把：

````markdown
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）

第一次启动：

```bash
make dev-db   # 启动本地 Postgres 18（端口 55432，可用 NERVE_DEV_DB_PORT 修改）
````

替换为：

````markdown
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）。代码生成和前端检查需要它

第一次启动：

```bash
pnpm install  # 安装 Node 依赖（代码生成和前端检查要用）
make dev-db   # 启动本地 Postgres 18（端口 55432，可用 NERVE_DEV_DB_PORT 修改）
````

- [ ] **Step 2: 修改 `README.md`（新增"接口与代码生成"一节）**

把：

````markdown
  ```yaml
  database:
    url: postgres://nerve:nerve@localhost:55433/nerve?sslmode=disable
  ```

## 版权
````

替换为：

````markdown
  ```yaml
  database:
    url: postgres://nerve:nerve@localhost:55433/nerve?sslmode=disable
  ```

## 接口与代码生成

接口先写描述（OpenAPI 3.1），再写实现：

- `api/openapi.yaml` 是入口，列出所有路径；各模块的描述在 `api/modules/<模块>.yaml`，公共组件在 `api/common.yaml`。
- 改完描述后执行 `make gen`，把生成的文件和描述一起提交：
  - `make gen-go`：Go 接口层（`server/internal/modules/<模块>/adapter/http/gen/`、`server/internal/platform/httpserver/apigen/`），只需要 Go。
  - `make gen-web`：打包好的 `api/dist/openapi.yaml`，以及 TS 客户端 `web/packages/api-client` 的类型，需要 Node（先执行 `pnpm install`）。
- 生成的文件不要手改。`make gen-check` 会重新生成一遍，检查生成物已经提交、没有差异；持续集成也执行这项检查。
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（TS 类型检查）。

## 版权
````

- [ ] **Step 3: 修改 `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-ci-split.md`（改为 done，写明处理结果）**

把：

```markdown
status: open
```

替换为：

```markdown
status: done
```

把：

```markdown
来源：[M0/P1 评审记录](../reviews/P1-repo-toolchain-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P3）

1. **命令按区域拆分**（[P3 spec](../specs/P3-api-contract.md) 2.10）：`lint-go` / `lint-web`、`gen-go` / `gen-web`、`gen-check-go` / `gen-check-web`。`*-go` 只需要 Go，`*-web` 只需要 Node；不带后缀的 `lint`、`gen`、`gen-check` 依次执行两个区域，供本地使用。
2. **持续集成的每个任务只调用自己区域的命令**：`server` 任务执行 `make gen-check-go` → `make lint-go` → `make test`；`web` 任务执行 `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web`。本地和持续集成仍然使用同一套 Makefile 命令。
3. **同仓 PR 的重复运行**：两个任务都加了条件 `github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository`。同仓库分支的 PR 已经由 push 事件在同一个提交上跑过，PR 页面显示的就是这次的结果；`pull_request` 事件只为来自 fork 的 PR 运行（fork 的推送不会在本仓库触发 push 事件）。

来源：[M0/P1 评审记录](../reviews/P1-repo-toolchain-review.md)。
```

- [ ] **Step 4: 修改 `docs/v0/M0-foundation/handoffs/P2-server-platform-p3-notes.md`（改为 done，写明处理结果）**

把：

```markdown
status: open
```

替换为：

```markdown
status: done
```

把：

```markdown
来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P3）

1. `bootstrap` 把 instance 的生成路由挂在 `httpserver.NewMux` 返回的根路由上：`instance.New().Register(mux, …)`（[P3 spec](../specs/P3-api-contract.md) 2.7）。`bootstrap` 的测试验证了挂上模块之后，`GET /api/v0/nope` 和 `POST /api/v0/instance` 仍然返回 404 problem+json。
2. 新增 `httpserver.APIErrors`（spec 2.6）：
   - `BadRequest` 返回 400 `bad_request`，`detail` 是绑定或解码失败的原因；接到 `StdHTTPServerOptions.ErrorHandlerFunc` 和 `StrictHTTPServerOptions.RequestErrorHandlerFunc`。
   - `InternalError` 记录日志，返回不带 `detail` 的 500 `internal_error`；接到 `StrictHTTPServerOptions.ResponseErrorHandlerFunc`。
   - 新增平台错误码 `bad_request`。带 `errors` 的参数校验错误码没有加：生成的代码只做类型绑定和 JSON 解码，不按 schema 校验取值，P3 中没有产生这种错误的地方；随 M2 的错误码体系一起确定（spec 第 3 节差异 7）。
3. `api/common.yaml` 的 `Problem` 与 `httpserver.Problem` 字段一致，并设了 `additionalProperties: false`。`httpserver` 的契约测试用 kin-openapi 校验平台写出的 404、500（panic 和 `InternalError`）、503、400，以及带 `errors` 的响应体（spec 2.8）；把 `Problem` 的某个 JSON 字段名改掉，这个测试就会失败。
4. `/api/` 下方法不对的请求仍然返回 404 problem+json，P3 不改（spec 2.6）。
5. instance 的 `app` 只导入本模块的 `domain`，archtest 通过。
6. P1 的持续集成拆分已处理，见[该 handoff](P1-repo-toolchain-ci-split.md) 的处理结果。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
```

- [ ] **Step 5: 核对 M0 设计文档的 Phase 进度表**

Run: `grep -n '^| P3 ' docs/v0/M0-foundation/M0-design.md`
Expected（spec 和 plan 提交时已经更新，这里不需要改动；review 链接在代码评审后补上）：

```
484:| P3 | api-contract | 进行中 | [spec](specs/P3-api-contract.md) | [plan](plans/P3-api-contract.md) | — |
```

- [ ] **Step 6: 按 README 从头走一遍（完整走查）**

```bash
pnpm install --frozen-lockfile
make dev-db
make gen-check
make lint
make test
make
```

Expected:
- `pnpm install --frozen-lockfile` 输出 `Done in …`；
- `make dev-db` 输出 `Container nerve-dev-db-1  Healthy`；
- `make gen-check` 退出码 0，工作区没有变化；
- `make lint` 输出 `0 issues.`，类型检查通过；
- `make test` 所有包都是 `ok`；
- `make` 列出 16 个命令（Task 8 Step 2）。

然后执行 `make run`，在另一个终端重复 Task 6 Step 6 的三个 `curl`，得到同样的结果；Ctrl-C 停止。

Run: `cd server && grep -E '^(go|toolchain) ' go.mod tools/go.mod`
Expected:

```
go.mod:go 1.27
go.mod:toolchain go1.27.1
tools/go.mod:go 1.27
tools/go.mod:toolchain go1.27.1
```

Run: `git status --short`
Expected:

```
 M README.md
 M docs/v0/M0-foundation/handoffs/P1-repo-toolchain-ci-split.md
 M docs/v0/M0-foundation/handoffs/P2-server-platform-p3-notes.md
```

- [ ] **Step 7: 提交**

```bash
git add README.md docs/v0/M0-foundation/handoffs/P1-repo-toolchain-ci-split.md docs/v0/M0-foundation/handoffs/P2-server-platform-p3-notes.md
git commit -m "docs(M0/P3): document code generation and close the P1 and P2 handoffs

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 8: 推送并确认持续集成（由控制者执行）**

1. 推送分支，确认 `CI` 的 `server` 和 `web` 两个任务都通过：`server` 依次执行 `make gen-check-go`、`make lint-go`、`make test`；`web` 依次执行 `pnpm install --frozen-lockfile`、`make gen-check-web`、`make lint-web`。
2. 演示"修改描述但不重新生成，持续集成必须失败"（M0 设计第 8 节 P3 的验收）：从本分支建一个临时分支，只提交 Task 8 Step 4 中对 `api/modules/instance.yaml` 的那处修改，推送；确认 `server` 任务在 `Generated code` 步骤失败（`make gen-check-go`），`web` 任务也在 `Generated code` 步骤失败（`make gen-check-web`）。然后删除这个临时分支（本地和远端）。

---

## 完成后

P3 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M0-foundation/reviews/P3-api-contract-review.md`：复述 spec 2.2 的 OpenAPI 3.1 结论，裁定 spec 第 3 节的差异，按 spec 第 7 节建立 handoff，同步 spec 第 3 节末尾列出的上级文档；并把 M0 设计文档中 P3 的状态改为"已完成"、补上 review 链接。
