# M0/P2 服务端平台层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭好 Go 服务端的平台层：分环境配置、slog 日志、Postgres 连接池与 goose 迁移执行器、集成测试用的 `pgtest`、HTTP 平台（中间件、problem+json、健康检查、优雅停机）、组合根 `bootstrap`、命令行 `nerve`、架构测试和 depguard，以及 `make run`。

**Architecture:** `cmd/nerve` 只做命令行解析：加载配置后调用 `bootstrap` 中与命令同名的函数。`bootstrap` 是唯一的组合根，用 `platform/*` 中的技术基础件（`config`、`logging`、`postgres`、`httpserver`）拼出服务。`platform` 各包之间互不导入（`config` 除外）。配置文件和迁移文件通过 `embed.FS` 编进程序（`server/configs`、`server/migrations`）。架构规则写成导入边上的纯函数（`internal/archtest`），先用人造的导入边逐条证明有效，再应用到真实的导入关系上。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、goose v3.28.0、koanf v2.3.6（yaml v1.1.1、rawbytes v1.0.1、file v1.2.1、env/v2 v2.0.1）、mapstructure v2.5.0、cobra v1.10.2、testcontainers-go v0.44.0（postgres 模块）、golang.org/x/tools v0.50.0、golangci-lint 2.13.2、PostgreSQL 18.6。

**Spec:** `docs/v0/M0-foundation/specs/P2-server-platform.md`（上级：`docs/v0/M0-foundation/M0-design.md`）

## Global Constraints

- Go：`server/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`，**每次 `go get` / `go mod tidy` 之后都要检查**（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就会拒绝运行）。不改动 `server/tools/go.mod`。模块路径 `github.com/open-nerve/NerveProject/server`。
- 依赖版本（写死，不要用 `@latest`）：
  - `github.com/knadh/koanf/v2@v2.3.6`
  - `github.com/knadh/koanf/parsers/yaml@v1.1.1`
  - `github.com/knadh/koanf/providers/rawbytes@v1.0.1`
  - `github.com/knadh/koanf/providers/file@v1.2.1`
  - `github.com/knadh/koanf/providers/env/v2@v2.0.1`
  - `github.com/go-viper/mapstructure/v2@v2.5.0`
  - `github.com/jackc/pgx/v5@v5.11.0`
  - `github.com/pressly/goose/v3@v3.28.0`
  - `github.com/testcontainers/testcontainers-go/modules/postgres@v0.44.0`
  - `github.com/spf13/cobra@v1.10.2`
  - `golang.org/x/tools@v0.50.0`
- golangci-lint `2.13.2`（`make lint` 会先执行 `make tools` 下载到 `./bin`）。每个 Task 提交前，`make lint` 必须输出 `0 issues.`，`make test` 必须全部通过。
- 集成测试镜像 `postgres:18.6`；开发库地址 `postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable`（`make dev-db` 启动，容器 `nerve-dev-db-1`）。
- 从 Task 7 起 `make test` 需要 Docker 在运行。**只能通过 testcontainers 和 `make dev-db` 使用 Docker，不要停止或改动其他任何容器。**
- 规则：不写 `init()`，不用全局可变状态，不引入依赖注入框架，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行。
- 代码注释用英文；配置文件、Makefile 中的注释用中文（与 P1 一致）。
- 所有代码块都是完整的文件内容，照原样写入，不要改动。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。

| 文件 | 职责 | Task |
|---|---|---|
| `server/.golangci.yml` | lint 规则：standard + depguard，gofmt、goimports | 1 |
| `server/internal/platform/config/config.go` | 配置类型；打码后的日志输出 | 2 |
| `server/internal/platform/config/validate.go` | 取值校验 | 2 |
| `server/internal/platform/config/redact.go` | 数据库地址打码 | 2 |
| `server/internal/platform/config/validate_test.go`、`redact_test.go` | 测试 | 2 |
| `server/internal/platform/config/load.go` | 分层加载、环境变量映射、严格解码 | 3 |
| `server/internal/platform/config/load_test.go` | 测试 | 3 |
| `server/configs/embed.go`、`config.yaml`、`config.dev.yaml`、`config.test.yaml`、`config.prod.yaml` | 内嵌的配置文件 | 3 |
| `server/configs/embed_test.go` | 三个内置环境的测试 | 3 |
| `server/internal/platform/logging/logging.go`、`logging_test.go` | slog 初始化 | 4 |
| `server/internal/platform/httpserver/problem.go`、`problem_test.go` | problem+json | 5 |
| `server/internal/platform/httpserver/middleware.go`、`middleware_test.go` | 请求 ID、异常恢复、访问日志 | 5 |
| `server/internal/platform/httpserver/routes.go`、`routes_test.go` | `/healthz`、`/readyz`、`/api/` 兜底 | 6 |
| `server/internal/platform/httpserver/server.go`、`server_test.go` | 服务生命周期 | 6 |
| `server/migrations/embed.go`、`embed_test.go`、`sql/.gitkeep` | 内嵌的迁移目录 | 7 |
| `server/internal/platform/postgres/pool.go`、`pool_test.go` | 连接池 | 7 |
| `server/internal/platform/postgres/migrator.go`、`migrator_test.go` | 迁移执行器 | 7 |
| `server/internal/platform/postgres/pgtest/pgtest.go`、`pgtest_test.go` | 测试数据库 | 7 |
| `server/internal/bootstrap/app.go`、`app_test.go` | 接线、运行 | 8 |
| `server/internal/bootstrap/commands.go`、`commands_test.go` | 各命令的实现 | 8 |
| `server/cmd/nerve/main.go`、`commands.go`、`main_test.go` | 命令行 | 9 |
| `Makefile`（修改） | `make run` | 9 |
| `server/internal/archtest/rules_test.go`、`rules_cases_test.go`、`repo_test.go` | 架构测试 | 10 |
| `README.md`（修改） | 开发环境说明 | 11 |
| `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-go-db-notes.md`（修改） | 交接事项改为 done | 11 |

---

### Task 1: golangci-lint 配置（depguard 与格式检查）

**Files:**
- Create: `server/.golangci.yml`

**Interfaces:**
- Consumes: P1 的 `make lint`（`cd server && bin/golangci-lint run ./...`）
- Produces: 之后每个 Task 的代码都要通过的 lint 规则；depguard 规则列表名 `banned`

- [ ] **Step 1: 写一个临时的违规文件 `server/internal/platform/buildinfo/probe.go`**

它同时导入 `log`（应被禁止）和 `log/slog`（应被允许），用来证明规则有效，验证完就删除。

```go
package buildinfo

import (
	"log"
	"log/slog"
)

// Probe exists only to show depguard at work; delete it after the check.
func Probe() {
	log.Println("probe")
	slog.Info("probe")
}
```

- [ ] **Step 2: 运行 lint，确认目前的默认规则发现不了它**

Run: `make lint`
Expected: `0 issues.`（P1 使用 golangci-lint 的默认规则，没有 depguard）

- [ ] **Step 3: 写 `server/.golangci.yml`**

```yaml
version: "2"

linters:
  # standard = errcheck、govet、ineffassign、staticcheck、unused
  default: standard
  enable:
    - depguard
  settings:
    depguard:
      rules:
        # 整个项目都禁止使用的库。模块边界和分层方向由 internal/archtest 检查。
        banned:
          deny:
            - pkg: github.com/google/uuid
              desc: 用标准库 uuid
            - pkg: github.com/gofrs/uuid
              desc: 用标准库 uuid
            - pkg: github.com/satori/go.uuid
              desc: 用标准库 uuid
            - pkg: github.com/spf13/viper
              desc: 用 koanf（platform/config）
            - pkg: github.com/pkg/errors
              desc: 用标准库 errors
            # 末尾的 $ 表示精确匹配：只禁止 log，不影响 log/slog
            - pkg: log$
              desc: 用 log/slog

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/open-nerve/NerveProject
```

- [ ] **Step 4: 运行 lint，确认 depguard 生效，而且只禁止了 `log`**

Run: `make lint`
Expected: 失败（make 报 `Error 1`），只有一个问题：

```
internal/platform/buildinfo/probe.go:4:2: import 'log' is not allowed from list 'banned': 用 log/slog (depguard)
	"log"
	^
1 issues:
* depguard: 1
```

`log/slog` 没有被报告：`log$` 中的 `$` 表示精确匹配；只写 `log` 的话是前缀匹配，会把 `log/slog` 也禁掉（已核实）。

- [ ] **Step 5: 删除临时文件，确认现有代码通过全部规则**

```bash
rm server/internal/platform/buildinfo/probe.go
```

Run: `make lint`
Expected: `0 issues.`

Run: `git status --short`
Expected: 只有 `?? server/.golangci.yml`。

- [ ] **Step 6: 提交**

```bash
git add server/.golangci.yml
git commit -m "build(server): add golangci-lint config with depguard and formatters

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: 配置类型、校验与打码

**Files:**
- Create: `server/internal/platform/config/config.go`、`server/internal/platform/config/validate.go`、`server/internal/platform/config/redact.go`
- Test: `server/internal/platform/config/validate_test.go`、`server/internal/platform/config/redact_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - `const EnvDev = "dev"`、`EnvTest = "test"`、`EnvProd = "prod"`
  - `type Config struct { Env string; Server ServerConfig; Database DatabaseConfig; Log LogConfig }`
  - `type ServerConfig struct { Addr string; ReadHeaderTimeout time.Duration; ShutdownTimeout time.Duration }`
  - `type DatabaseConfig struct { URL string; MaxConns int32; AutoMigrate bool }`
  - `type LogConfig struct { Level string; Format string }`
  - `func (c Config) LogValue() slog.Value`（打码）
  - 包内：`func (c Config) validate() error`（Task 3 的 `Load` 调用）、`func redactURL(raw string) string`（`LogValue` 使用）

- [ ] **Step 1: 写失败的测试 `server/internal/platform/config/validate_test.go`**

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
			ShutdownTimeout:   20 * time.Second,
		},
		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10},
		Log:      LogConfig{Level: "info", Format: "json"},
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	if err := validConfig().validate(); err != nil {
		t.Fatalf("validate() = %v, want nil", err)
	}
}

func TestValidateReportsEveryInvalidKey(t *testing.T) {
	cfg := Config{
		Server:   ServerConfig{Addr: "8080", ReadHeaderTimeout: 0, ShutdownTimeout: -time.Second},
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
		"server.shutdown_timeout: must be positive, got -1s",
		"database.url: is required",
		"database.max_conns: must be at least 1, got 0",
		`log.level: must be one of debug, info, warn, error, got "verbose"`,
		`log.format: must be text or json, got "xml"`,
	}
	if got := err.Error(); got != strings.Join(want, "\n") {
		t.Errorf("validate() errors:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
}
```

- [ ] **Step 2: 写失败的测试 `server/internal/platform/config/redact_test.go`**

```go
package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty stays empty", "", ""},
		{"password in user info", "postgres://nerve:secret@localhost:5432/nerve", "postgres://nerve:xxxxx@localhost:5432/nerve"},
		{"password query parameter", "postgres://localhost/nerve?password=secret&sslmode=disable", "postgres://localhost/nerve?password=xxxxx&sslmode=disable"},
		{"no password is unchanged", "postgres://nerve@localhost/nerve?sslmode=disable", "postgres://nerve@localhost/nerve?sslmode=disable"},
		{"key/value string is masked entirely", "host=localhost user=nerve password=secret", "xxxxx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactURL(tt.in); got != tt.want {
				t.Errorf("redactURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLogValueMasksDatabasePassword(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Info("configuration loaded", "config", validConfig())

	out := buf.String()
	if strings.Contains(out, "secret") {
		t.Errorf("log output leaks the password: %s", out)
	}
	for _, want := range []string{
		"config.env=test",
		"config.server.addr=:8080",
		"config.server.shutdown_timeout=20s",
		"config.database.url=postgres://nerve:xxxxx@localhost:5432/nerve",
		"config.database.max_conns=10",
		"config.log.format=json",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output lacks %q: %s", want, out)
		}
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/config/`
Expected: 编译失败（`[build failed]`），报 `undefined: Config`、`undefined: EnvTest`、`undefined: ServerConfig`、`undefined: redactURL` 等。

- [ ] **Step 4: 写 `server/internal/platform/config/config.go`**

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
	Log      LogConfig      `koanf:"log"`
}

// ServerConfig configures the HTTP server.
type ServerConfig struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
}

// DatabaseConfig configures the PostgreSQL pool and schema migrations.
type DatabaseConfig struct {
	URL         string `koanf:"url"`
	MaxConns    int32  `koanf:"max_conns"`
	AutoMigrate bool   `koanf:"auto_migrate"`
}

// LogConfig configures the process logger.
type LogConfig struct {
	Level  string `koanf:"level"`  // debug, info, warn or error
	Format string `koanf:"format"` // text or json
}

// LogValue renders the configuration for logs with secrets masked, so the
// effective configuration can be logged at startup.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.Group("server",
			slog.String("addr", c.Server.Addr),
			slog.Duration("read_header_timeout", c.Server.ReadHeaderTimeout),
			slog.Duration("shutdown_timeout", c.Server.ShutdownTimeout),
		),
		slog.Group("database",
			slog.String("url", redactURL(c.Database.URL)),
			slog.Int("max_conns", int(c.Database.MaxConns)),
			slog.Bool("auto_migrate", c.Database.AutoMigrate),
		),
		slog.Group("log",
			slog.String("level", c.Log.Level),
			slog.String("format", c.Log.Format),
		),
	)
}
```

- [ ] **Step 5: 写 `server/internal/platform/config/validate.go`**

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
	if c.Server.ReadHeaderTimeout <= 0 {
		fail("server.read_header_timeout", "must be positive, got %s", c.Server.ReadHeaderTimeout)
	}
	if c.Server.ShutdownTimeout <= 0 {
		fail("server.shutdown_timeout", "must be positive, got %s", c.Server.ShutdownTimeout)
	}
	if c.Database.URL == "" {
		fail("database.url", "is required")
	}
	if c.Database.MaxConns < 1 {
		fail("database.max_conns", "must be at least 1, got %d", c.Database.MaxConns)
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
		fail("log.level", "must be one of debug, info, warn, error, got %q", c.Log.Level)
	}
	if c.Log.Format != "text" && c.Log.Format != "json" {
		fail("log.format", "must be text or json, got %q", c.Log.Format)
	}
	return errors.Join(errs...)
}
```

- [ ] **Step 6: 写 `server/internal/platform/config/redact.go`**

```go
package config

import "net/url"

const redacted = "xxxxx"

// redactURL masks the password of a PostgreSQL connection URL, both in the
// user info and in a password query parameter. A value that is not a URL is
// replaced entirely: key/value connection strings can carry a password too.
func redactURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return redacted
	}
	if q := u.Query(); q.Has("password") {
		q.Set("password", redacted)
		u.RawQuery = q.Encode()
	}
	return u.Redacted()
}
```

- [ ] **Step 7: 运行测试，确认通过**

Run: `cd server && go test ./internal/platform/config/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/config`

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 8: 提交**

```bash
git add server/internal/platform/config
git commit -m "feat(server): add config types, validation and redaction

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 分层加载配置与内嵌的配置文件

**Files:**
- Create: `server/internal/platform/config/load.go`
- Create: `server/configs/embed.go`、`server/configs/config.yaml`、`server/configs/config.dev.yaml`、`server/configs/config.test.yaml`、`server/configs/config.prod.yaml`
- Test: `server/internal/platform/config/load_test.go`、`server/configs/embed_test.go`
- Modify: `server/go.mod`、`server/go.sum`（添加依赖）

**Interfaces:**
- Consumes: Task 2 的 `Config`、`validate`
- Produces:
  - `type Sources struct { Embedded fs.FS; Environ []string; LocalFile string }`
  - `func Load(src Sources) (Config, error)`：加载顺序为内置 `config.yaml` → 内置 `config.<env>.yaml` → `$NERVE_CONFIG_DIR/config.yaml` → `$NERVE_CONFIG_DIR/config.<env>.yaml` → `LocalFile`（仅 dev）→ `NERVE_<SECTION>__<KEY>`
  - `configs.FS() fs.FS`：内置的四个配置文件。Task 9 的 `cmd/nerve` 用它调用 `Load`

- [ ] **Step 1: 写失败的测试 `server/internal/platform/config/load_test.go`**

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
  shutdown_timeout: 20s
database:
  url: ""
  max_conns: 10
  auto_migrate: true
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
			ShutdownTimeout:   40 * time.Second,
		},
		Database: DatabaseConfig{
			URL:         "postgres://embedded-dev", // built-in config.dev.yaml
			MaxConns:    30,                        // config dir: config.dev.yaml beats config.yaml
			AutoMigrate: false,                     // environment
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

- [ ] **Step 2: 写失败的测试 `server/configs/embed_test.go`**

```go
package configs_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

func TestBuiltInProfiles(t *testing.T) {
	const devURL = "postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable"
	tests := []struct {
		env         string
		url         string // expected database.url
		autoMigrate bool
		level       string
		format      string
	}{
		{env: "dev", url: devURL, autoMigrate: true, level: "debug", format: "text"},
		{env: "test", url: "postgres://from-env", autoMigrate: true, level: "warn", format: "text"},
		{env: "prod", url: "postgres://from-env", autoMigrate: false, level: "info", format: "json"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			environ := []string{"NERVE_ENV=" + tt.env}
			if tt.env != "dev" {
				environ = append(environ, "NERVE_DATABASE__URL=postgres://from-env")
			}
			cfg, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: environ})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			want := config.Config{
				Env: tt.env,
				Server: config.ServerConfig{
					Addr:              ":8080",
					ReadHeaderTimeout: 5 * time.Second,
					ShutdownTimeout:   20 * time.Second,
				},
				Database: config.DatabaseConfig{URL: tt.url, MaxConns: 10, AutoMigrate: tt.autoMigrate},
				Log:      config.LogConfig{Level: tt.level, Format: tt.format},
			}
			if cfg != want {
				t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
			}
		})
	}
}

func TestProdRequiresDatabaseURL(t *testing.T) {
	_, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=prod"}})
	if err == nil {
		t.Fatal("Load() error = nil, want database.url to be required")
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/config/ ./configs/`
Expected: 两个包都是 `[build failed]`。`internal/platform/config` 报 `undefined: Load`、`undefined: Sources`；`configs` 报 `no non-test Go files in …/server/configs`（包里还只有测试文件）。

- [ ] **Step 4: 写 `server/internal/platform/config/load.go`**

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
	"github.com/knadh/koanf/providers/file"
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

func loadFileIfExists(k *koanf.Koanf, path string) error {
	err := k.Load(file.Provider(path), yaml.Parser())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
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
			DecodeHook:       durationHook,
			ErrorUnused:      true,
			WeaklyTypedInput: true, // environment values are strings
		},
	})
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

- [ ] **Step 5: 写 `server/configs/embed.go`**

```go
// Package configs embeds the built-in configuration files, so a single nerve
// binary runs without any file next to it.
package configs

import (
	"embed"
	"io/fs"
)

// config.local.yaml is personal and never embedded.
//
//go:embed config.yaml config.dev.yaml config.test.yaml config.prod.yaml
var files embed.FS

// FS returns the built-in configuration files.
func FS() fs.FS {
	return files
}
```

- [ ] **Step 6: 写 `server/configs/config.yaml`**

```yaml
# 基础配置：列出所有配置项及其默认值。各环境的覆盖项在 config.<env>.yaml 中。
# 任何一项都可以用环境变量覆盖：NERVE_ 加上配置路径，层级之间用双下划线，
# 例如 database.url 对应 NERVE_DATABASE__URL。
server:
  addr: ":8080"
  read_header_timeout: 5s
  shutdown_timeout: 20s

database:
  # 必须提供。dev 环境写在 config.dev.yaml 中；test 和 prod 通过 NERVE_DATABASE__URL 提供。
  url: ""
  max_conns: 10
  # nerve serve 启动前是否自动执行迁移
  auto_migrate: true

log:
  level: info    # debug | info | warn | error
  format: json   # text | json
```

- [ ] **Step 7: 写 `server/configs/config.dev.yaml`**

```yaml
# 开发环境（NERVE_ENV=dev，默认）。个人覆盖项写在 config.local.yaml 中（不进仓库）。
database:
  # make dev-db 启动的本地开发库。账号密码只在本机使用，不是机密。
  # 用 NERVE_DEV_DB_PORT 换了端口时，要同时覆盖这一项（NERVE_DATABASE__URL 或 config.local.yaml）。
  url: postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable

log:
  level: debug
  format: text
```

- [ ] **Step 8: 写 `server/configs/config.test.yaml`**

```yaml
# 测试环境（NERVE_ENV=test）：集成测试和端到端测试使用。数据库地址通过 NERVE_DATABASE__URL 提供。
log:
  level: warn
  format: text
```

- [ ] **Step 9: 写 `server/configs/config.prod.yaml`**

```yaml
# 生产环境（NERVE_ENV=prod）。数据库地址通过 NERVE_DATABASE__URL 提供。
database:
  # 生产环境先执行 nerve migrate up，再启动 nerve serve
  auto_migrate: false
```

- [ ] **Step 10: 添加依赖**

```bash
cd server && go get github.com/knadh/koanf/v2@v2.3.6 github.com/knadh/koanf/parsers/yaml@v1.1.1 github.com/knadh/koanf/providers/rawbytes@v1.0.1 github.com/knadh/koanf/providers/file@v1.2.1 github.com/knadh/koanf/providers/env/v2@v2.0.1 github.com/go-viper/mapstructure/v2@v2.5.0 && go mod tidy
```

Run: `cd server && grep -E '^(go|toolchain) ' go.mod`
Expected: 两行，`go 1.27` 和 `toolchain go1.27.1`（没有被抬高）。

Run: `cd server && sed -n '/^require (/,/^)/p' go.mod | sed '/^)/q'`
Expected:

```
require (
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/knadh/koanf/parsers/yaml v1.1.1
	github.com/knadh/koanf/providers/env/v2 v2.0.1
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/providers/rawbytes v1.0.1
	github.com/knadh/koanf/v2 v2.3.6
)
```

- [ ] **Step 11: 运行测试，确认通过**

Run: `cd server && go test ./internal/platform/config/ ./configs/`
Expected:

```
ok  	github.com/open-nerve/NerveProject/server/internal/platform/config
ok  	github.com/open-nerve/NerveProject/server/configs
```

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 12: 提交**

```bash
git add server/internal/platform/config server/configs server/go.mod server/go.sum
git commit -m "feat(server): load layered config from embedded files and environment

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: 日志

**Files:**
- Create: `server/internal/platform/logging/logging.go`
- Test: `server/internal/platform/logging/logging_test.go`

**Interfaces:**
- Consumes: Task 2 的 `config.LogConfig`
- Produces: `func New(w io.Writer, cfg config.LogConfig) (*slog.Logger, error)`（Task 8 的 `bootstrap.Serve` 使用）

- [ ] **Step 1: 写失败的测试 `server/internal/platform/logging/logging_test.go`**

```go
package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

func TestNewJSON(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, config.LogConfig{Level: "info", Format: "json"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.Debug("hidden")
	logger.Info("shown", "key", "value")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("want exactly one JSON line, got %q: %v", buf.String(), err)
	}
	if entry["msg"] != "shown" || entry["level"] != "INFO" || entry["key"] != "value" {
		t.Errorf("entry = %v", entry)
	}
}

func TestNewText(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, config.LogConfig{Level: "debug", Format: "text"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.Debug("shown", "key", "value")

	if out := buf.String(); !strings.Contains(out, "level=DEBUG msg=shown key=value") {
		t.Errorf("output = %q", out)
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	for _, cfg := range []config.LogConfig{
		{Level: "verbose", Format: "json"},
		{Level: "info", Format: "xml"},
	} {
		if _, err := New(&bytes.Buffer{}, cfg); err == nil {
			t.Errorf("New(%+v) error = nil, want an error", cfg)
		}
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/logging/`
Expected: 编译失败（`[build failed]`），报 `undefined: New`。

- [ ] **Step 3: 写 `server/internal/platform/logging/logging.go`**

```go
// Package logging builds nerve's structured logger.
package logging

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// New returns a logger that writes to w with the configured level and
// format. It never touches the slog default logger.
func New(w io.Writer, cfg config.LogConfig) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}
	opts := &slog.HandlerOptions{Level: level}
	switch cfg.Format {
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("log format %q: must be text or json", cfg.Format)
	}
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd server && go test ./internal/platform/logging/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/logging`

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 5: 提交**

```bash
git add server/internal/platform/logging
git commit -m "feat(server): add slog logger factory

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: problem+json 与三个中间件

**Files:**
- Create: `server/internal/platform/httpserver/problem.go`、`server/internal/platform/httpserver/middleware.go`
- Test: `server/internal/platform/httpserver/problem_test.go`、`server/internal/platform/httpserver/middleware_test.go`

**Interfaces:**
- Consumes: 无（只用标准库，包括 Go 1.27 标准库的 `uuid`）
- Produces:
  - `const ContentTypeProblem = "application/problem+json"`；`const CodeNotFound = "not_found"`、`CodeInternal = "internal_error"`、`CodeNotReady = "not_ready"`
  - `type Problem struct { Status int; Code string; Title string; Detail string; Errors []FieldError }`（JSON 名 `status`、`code`、`title`、`detail`、`errors`，后两个可省略）
  - `type FieldError struct { Field string; Message string }`
  - `func WriteProblem(w http.ResponseWriter, p Problem)`
  - `const HeaderRequestID = "X-Request-Id"`
  - 包内：`func middleware(h http.Handler, logger *slog.Logger) http.Handler`（顺序：请求 ID → 异常恢复 → 访问日志）、`func requestID(ctx context.Context) string`。Task 6 的 `NewServer` 和 `/readyz` 使用
  - 测试辅助函数（`middleware_test.go`）：`captureLogs`、`findLog`、`serve`，Task 6 的测试也会用到

- [ ] **Step 1: 写失败的测试 `server/internal/platform/httpserver/problem_test.go`**

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
		Errors: []FieldError{{Field: "state_id", Message: "unknown state"}},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	want := `{"status":422,"code":"issue.state_not_in_project","title":"State is not in the project",` +
		`"errors":[{"field":"state_id","message":"unknown state"}]}` + "\n"
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

- [ ] **Step 2: 写失败的测试 `server/internal/platform/httpserver/middleware_test.go`**

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
		seen = requestID(r.Context())
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
				entry["request_id"] != "req-2" || entry["duration"] == nil {
				t.Errorf("access log = %v", entry)
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/httpserver/`
Expected: 编译失败（`[build failed]`），报 `undefined: middleware`、`undefined: requestID`、`undefined: HeaderRequestID`、`undefined: ContentTypeProblem` 等。

- [ ] **Step 4: 写 `server/internal/platform/httpserver/problem.go`**

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
	CodeNotFound = "not_found"
	CodeInternal = "internal_error"
	CodeNotReady = "not_ready"
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

- [ ] **Step 5: 写 `server/internal/platform/httpserver/middleware.go`**

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

// requestID returns the ID the request ID middleware assigned to the request.
func requestID(ctx context.Context) string {
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

// withRecover turns a panic into a logged 500 problem. If the response has
// already started, the connection is aborted instead so the client cannot
// mistake a truncated body for a complete one.
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
				slog.String("request_id", requestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
			)
			if rec.status != 0 {
				panic(http.ErrAbortHandler)
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
// answer the recover middleware gives.
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
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request",
				slog.String("request_id", requestID(r.Context())),
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

- [ ] **Step 6: 运行测试，确认通过**

Run: `cd server && go test -race ./internal/platform/httpserver/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver`

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 7: 提交**

```bash
git add server/internal/platform/httpserver
git commit -m "feat(server): add problem+json and the request ID, recover and access log middleware

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: 平台路由与服务生命周期

**Files:**
- Create: `server/internal/platform/httpserver/routes.go`、`server/internal/platform/httpserver/server.go`
- Test: `server/internal/platform/httpserver/routes_test.go`、`server/internal/platform/httpserver/server_test.go`

**Interfaces:**
- Consumes: Task 5 的 `WriteProblem`、`middleware`、`requestID` 和测试辅助函数；Task 2 的 `config.ServerConfig`
- Produces:
  - `type Check struct { Name string; Run func(ctx context.Context) error }`
  - `func NewMux(logger *slog.Logger, checks ...Check) *http.ServeMux`：`GET /healthz`、`GET /readyz`、`/api/` 兜底
  - `type Server struct{ … }`；`func NewServer(cfg config.ServerConfig, h http.Handler, logger *slog.Logger) *Server`
  - `func (s *Server) ListenAndServe(ctx context.Context) error`；`func (s *Server) Serve(ctx context.Context, ln net.Listener) error`

- [ ] **Step 1: 写失败的测试 `server/internal/platform/httpserver/routes_test.go`**

```go
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzDoesNotRunChecks(t *testing.T) {
	failing := Check{Name: "database", Run: func(context.Context) error { return errors.New("down") }}
	mux := NewMux(slog.New(slog.DiscardHandler), failing)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/healthz", nil))

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
	mux := NewMux(slog.New(slog.DiscardHandler), check("database"), check("migrations"))

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/readyz", nil))

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
	mux := NewMux(logger,
		Check{Name: "database", Run: func(context.Context) error { return errors.New("connection refused") }},
		Check{Name: "migrations", Run: func(context.Context) error { secondRan = true; return nil }},
	)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/readyz", nil))

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
	mux := NewMux(slog.New(slog.DiscardHandler))
	for _, target := range []string{"/api/", "/api/v0/nope"} {
		rec := serve(mux, httptest.NewRequest(http.MethodPost, target, nil))

		if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != ContentTypeProblem {
			t.Errorf("POST %s = %d %s, want 404 problem+json", target, rec.Code, rec.Header().Get("Content-Type"))
		}
		want := `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for POST ` + target + `"}` + "\n"
		if rec.Body.String() != want {
			t.Errorf("body = %s, want %s", rec.Body, want)
		}
	}
}

func TestOtherPathsAreLeftForTheWebUI(t *testing.T) {
	rec := serve(NewMux(slog.New(slog.DiscardHandler)), httptest.NewRequest(http.MethodGet, "/projects", nil))

	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") == ContentTypeProblem {
		t.Errorf("GET /projects = %d %s, want the mux's plain 404", rec.Code, rec.Header().Get("Content-Type"))
	}
}
```

- [ ] **Step 2: 写失败的测试 `server/internal/platform/httpserver/server_test.go`**

```go
package httpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
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
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.ServerConfig{Addr: ln.Addr().String(), ReadHeaderTimeout: time.Second, ShutdownTimeout: shutdownTimeout}
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
	url, cancel, done := startServer(t, time.Second, NewMux(slog.New(slog.DiscardHandler)))

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
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/httpserver/`
Expected: 编译失败（`[build failed]`），报 `undefined: Check`、`undefined: NewMux` 等。

- [ ] **Step 4: 写 `server/internal/platform/httpserver/routes.go`**

```go
package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
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

// NewMux returns a router holding the platform routes:
//
//   - GET /healthz: liveness; never touches a dependency.
//   - GET /readyz: runs checks in order; the first failure answers 503.
//   - /api/: every API path no module handles answers 404 problem+json,
//     never the web UI.
//
// Modules and the web UI register their own routes on the returned mux.
func NewMux(logger *slog.Logger, checks ...Check) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeStatusOK(w)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		for _, c := range checks {
			if err := c.Run(ctx); err != nil {
				logger.WarnContext(ctx, "readiness check failed",
					slog.String("request_id", requestID(ctx)),
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
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, Problem{
			Status: http.StatusNotFound,
			Code:   CodeNotFound,
			Title:  http.StatusText(http.StatusNotFound),
			Detail: "no API endpoint for " + r.Method + " " + r.URL.Path,
		})
	})
	return mux
}

func writeStatusOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

- [ ] **Step 5: 写 `server/internal/platform/httpserver/server.go`**

```go
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// Server is the HTTP server of a nerve process.
type Server struct {
	srv             *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewServer serves h behind the platform middleware chain
// (request ID -> recover -> access log).
func NewServer(cfg config.ServerConfig, h http.Handler, logger *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              cfg.Addr,
			Handler:           middleware(h, logger),
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// ListenAndServe listens on server.addr and then behaves like Serve.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.srv.Addr, err)
	}
	return s.Serve(ctx, ln)
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

- [ ] **Step 6: 运行测试，确认通过（带竞态检测，重复 3 次）**

Run: `cd server && go test -race -count=3 ./internal/platform/httpserver/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/httpserver`，用时约 2 秒。

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 7: 提交**

```bash
git add server/internal/platform/httpserver
git commit -m "feat(server): add health routes, API catch-all and graceful HTTP server

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: 连接池、迁移执行器与 pgtest

**Files:**
- Create: `server/migrations/embed.go`、`server/migrations/sql/.gitkeep`
- Create: `server/internal/platform/postgres/pool.go`、`server/internal/platform/postgres/migrator.go`
- Create: `server/internal/platform/postgres/pgtest/pgtest.go`
- Test: `server/migrations/embed_test.go`、`server/internal/platform/postgres/pool_test.go`、`server/internal/platform/postgres/migrator_test.go`、`server/internal/platform/postgres/pgtest/pgtest_test.go`
- Modify: `server/go.mod`、`server/go.sum`

**Interfaces:**
- Consumes: Task 2 的 `config.DatabaseConfig`
- Produces:
  - `migrations.FS() fs.FS`：以 `sql/` 为根的迁移文件（M0 只有 `.gitkeep`）
  - `func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error)`
  - `var ErrPendingMigrations`；`type Migration struct { Version int64; Source string }`；`type MigrationStatus struct { Migration; Applied bool; AppliedAt time.Time }`
  - `func NewMigrator(pool *pgxpool.Pool, fsys fs.FS) (*Migrator, error)`，方法 `Up(ctx) ([]Migration, error)`、`Down(ctx) (*Migration, error)`、`Status(ctx) ([]MigrationStatus, error)`、`CheckUpToDate(ctx) error`、`Close() error`
  - `pgtest.NewDatabase(t testing.TB) string`、`pgtest.NewEmptyDatabase(t testing.TB) string`：返回新测试库的连接 URL

- [ ] **Step 1: 确认 Docker 在运行**

Run: `docker info --format '{{.ServerVersion}}'`
Expected: 输出 Docker 版本号。本 Task 起 `go test` 会通过 testcontainers 启动 `postgres:18.6` 容器，测试进程退出后由 Ryuk 自动删除。

- [ ] **Step 2: 写失败的测试 `server/migrations/embed_test.go`**

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
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, e := range entries {
		if e.Name() == ".gitkeep" {
			continue
		}
		if e.IsDir() || !migrationName.MatchString(e.Name()) {
			t.Errorf("%s: want NNNNN_<module>_<description>.sql, e.g. 00001_identity_create_users.sql", e.Name())
		}
	}
}
```

- [ ] **Step 3: 写失败的测试 `server/internal/platform/postgres/pool_test.go`**

```go
package postgres_test

import (
	"context"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

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

func TestNewPoolRejectsMalformedURLWithoutLeakingPassword(t *testing.T) {
	_, err := postgres.NewPool(context.Background(), config.DatabaseConfig{
		URL:      "postgres://nerve:secret@localhost:notaport/nerve",
		MaxConns: 1,
	})
	if err == nil {
		t.Fatal("NewPool() error = nil, want a parse error")
	}
	if !strings.HasPrefix(err.Error(), "database.url: ") || strings.Contains(err.Error(), "secret") {
		t.Errorf("NewPool() error = %q, want a database.url error without the password", err)
	}
}
```

- [ ] **Step 4: 写失败的测试 `server/internal/platform/postgres/migrator_test.go`**

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

// sampleMigrations is a test-only migration set: M0 ships no migrations.
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
		"00001_probe_broken.sql": {Data: []byte("-- +goose Up\nCREATE TABLE broken (;\n")},
	})

	if _, err := m.Up(context.Background()); err == nil {
		t.Error("Up() error = nil, want the SQL error")
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

- [ ] **Step 5: 写失败的测试 `server/internal/platform/postgres/pgtest/pgtest_test.go`**

```go
package pgtest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func connect(t *testing.T, url string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func TestDatabasesAreIsolated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := connect(t, pgtest.NewDatabase(t))
	b := connect(t, pgtest.NewDatabase(t))

	if _, err := a.Exec(ctx, "CREATE TABLE only_in_a (id int)"); err != nil {
		t.Fatal(err)
	}
	var found bool
	if err := b.QueryRow(ctx, "SELECT to_regclass('only_in_a') IS NOT NULL").Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("a table created in one test database is visible in another")
	}
}

func TestEmptyDatabaseHasNoTables(t *testing.T) {
	t.Parallel()
	conn := connect(t, pgtest.NewEmptyDatabase(t))

	var tables int
	err := conn.QueryRow(context.Background(),
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tables)
	if err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Errorf("empty database has %d tables, want 0", tables)
	}
}

func TestDatabaseIsDroppedAfterTheTest(t *testing.T) {
	var url string
	t.Run("uses a database", func(t *testing.T) {
		url = pgtest.NewDatabase(t)
		connect(t, url) // an open connection must not block the drop
	})
	if url == "" {
		t.Skip("the subtest was skipped")
	}

	_, err := pgx.Connect(context.Background(), url)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "3D000" { // invalid_catalog_name
		t.Errorf("connect after the test = %v, want database does not exist (3D000)", err)
	}
}
```

- [ ] **Step 6: 运行测试，确认失败**

Run: `cd server && go test ./migrations/ ./internal/platform/postgres/...`
Expected: 三个包都失败（实现和依赖都还没有）：
- `migrations` 为 `[build failed]`：`migrations/embed_test.go:12:29: undefined: FS`
- `internal/platform/postgres` 为 `[setup failed]`：`no required module provides package github.com/jackc/pgx/v5/pgxpool`
- `internal/platform/postgres/pgtest` 为 `[setup failed]`：`no required module provides package github.com/jackc/pgx/v5`

依赖在写完实现之后一起加（Step 12），这样 `go mod tidy` 一次就能整理好。

- [ ] **Step 7: 写 `server/migrations/embed.go`**

```go
// Package migrations embeds the SQL schema migrations. Files live in sql/ and
// are named NNNNN_<module>_<description>.sql.
package migrations

import (
	"embed"
	"io/fs"
)

// "all:" keeps sql/.gitkeep, so the pattern still matches while no migration
// exists; a plain *.sql pattern would fail to compile then.
//
//go:embed all:sql
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

- [ ] **Step 8: 建立空的迁移目录**

```bash
mkdir -p server/migrations/sql && touch server/migrations/sql/.gitkeep
```

- [ ] **Step 9: 写 `server/internal/platform/postgres/pool.go`**

```go
// Package postgres connects nerve to PostgreSQL and runs its schema
// migrations.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// NewPool creates a connection pool for database.url with at most
// database.max_conns connections. It connects lazily: an unreachable database
// shows up on first use, e.g. in the /readyz check.
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("database.url: %w", err) // pgx masks the password
	}
	pc.MaxConns = cfg.MaxConns
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 10: 写 `server/internal/platform/postgres/migrator.go`**

```go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// ErrPendingMigrations is returned by CheckUpToDate when the schema is behind.
var ErrPendingMigrations = errors.New("database has pending migrations")

// Migration identifies one migration file.
type Migration struct {
	Version int64
	Source  string // file name, e.g. "00001_identity_create_users.sql"
}

// MigrationStatus tells whether a migration has been applied.
type MigrationStatus struct {
	Migration
	Applied   bool
	AppliedAt time.Time // zero while pending
}

// Migrator applies the goose SQL migrations at the root of a file system.
// A file system without migrations is valid; every operation is then a no-op.
type Migrator struct {
	provider *goose.Provider // nil when there are no migrations
}

// NewMigrator prepares the migrations in fsys for the database behind pool.
// Close releases the connection it borrows from the pool.
func NewMigrator(pool *pgxpool.Pool, fsys fs.FS) (*Migrator, error) {
	db := stdlib.OpenDBFromPool(pool)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys,
		goose.WithDisableGlobalRegistry(true), // no package-level Go migrations
	)
	if err != nil {
		_ = db.Close() // nothing has been borrowed from the pool yet
		if errors.Is(err, goose.ErrNoMigrations) {
			return &Migrator{}, nil
		}
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return &Migrator{provider: provider}, nil
}

// Up applies every pending migration in version order and returns them.
func (m *Migrator) Up(ctx context.Context) ([]Migration, error) {
	if m.provider == nil {
		return nil, nil
	}
	results, err := m.provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate up: %w", err)
	}
	applied := make([]Migration, 0, len(results))
	for _, r := range results {
		applied = append(applied, migrationOf(r.Source))
	}
	return applied, nil
}

// Down rolls back the most recently applied migration and returns it, or nil
// when nothing is applied.
func (m *Migrator) Down(ctx context.Context) (*Migration, error) {
	if m.provider == nil {
		return nil, nil
	}
	result, err := m.provider.Down(ctx)
	if errors.Is(err, goose.ErrNoNextVersion) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("migrate down: %w", err)
	}
	rolledBack := migrationOf(result.Source)
	return &rolledBack, nil
}

// Status lists every known migration in version order.
func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	if m.provider == nil {
		return nil, nil
	}
	statuses, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration status: %w", err)
	}
	out := make([]MigrationStatus, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, MigrationStatus{
			Migration: migrationOf(s.Source),
			Applied:   s.State == goose.StateApplied,
			AppliedAt: s.AppliedAt,
		})
	}
	return out, nil
}

// CheckUpToDate returns ErrPendingMigrations if any migration is pending. Its
// signature fits a readiness check.
func (m *Migrator) CheckUpToDate(ctx context.Context) error {
	if m.provider == nil {
		return nil
	}
	pending, err := m.provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("check migrations: %w", err)
	}
	if pending {
		return ErrPendingMigrations
	}
	return nil
}

// Close releases the database handle. Call it before closing the pool.
func (m *Migrator) Close() error {
	if m.provider == nil {
		return nil
	}
	return m.provider.Close()
}

func migrationOf(s *goose.Source) Migration {
	return Migration{Version: s.Version, Source: path.Base(s.Path)}
}
```

- [ ] **Step 11: 写 `server/internal/platform/postgres/pgtest/pgtest.go`**

```go
// Package pgtest gives integration tests their own PostgreSQL database. Only
// test code may import it (enforced by internal/archtest).
//
// The first call in a test binary starts one PostgreSQL container, shared by
// every test of that package, and migrates a template database with the
// production migrations. Each test then gets a copy made with
// CREATE DATABASE ... TEMPLATE, dropped when the test ends. The container is
// removed by the testcontainers reaper (Ryuk) after the test binary exits.
package pgtest

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// image is the PostgreSQL image, the same as the development database.
const image = "postgres:18.6"

const templateDB = "nerve_template"

// shared is the one container of this test binary, started on first use.
// A package-level singleton is inherent to "one container per test binary";
// pgtest is test-only code.
var shared = sync.OnceValues(startCluster)

type cluster struct {
	admin   *pgxpool.Pool // connected to the maintenance database "postgres"
	base    url.URL       // URL of "postgres"; per-test URLs swap the path
	counter atomic.Int64
}

// NewDatabase returns the URL of a new database with every production
// migration applied. The database is dropped when the test ends. Under
// go test -short the test is skipped instead.
func NewDatabase(t testing.TB) string {
	return newDatabase(t, templateDB)
}

// NewEmptyDatabase returns the URL of a new database without any migration,
// for tests that bring their own schema, such as the migrator's.
func NewEmptyDatabase(t testing.TB) string {
	return newDatabase(t, "template0")
}

func newDatabase(t testing.TB, template string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}
	c, err := shared()
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	name := fmt.Sprintf("test_%d", c.counter.Add(1))
	create := "CREATE DATABASE " + pgx.Identifier{name}.Sanitize() + " TEMPLATE " + pgx.Identifier{template}.Sanitize()
	if _, err := c.admin.Exec(context.Background(), create); err != nil {
		t.Fatalf("pgtest: create database: %v", err)
	}
	t.Cleanup(func() {
		drop := "DROP DATABASE " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)"
		if _, err := c.admin.Exec(context.Background(), drop); err != nil {
			t.Errorf("pgtest: drop database %s: %v", name, err)
		}
	})
	return c.url(name)
}

func (c *cluster) url(database string) string {
	u := c.base
	u.Path = "/" + database
	return u.String()
}

func startCluster() (*cluster, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, image,
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start %s container (is Docker running?): %w", image, err)
	}
	raw, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("container address: %w", err)
	}
	base, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("container address: %w", err)
	}
	admin, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: raw, MaxConns: 4})
	if err != nil {
		return nil, err
	}
	c := &cluster{admin: admin, base: *base}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+templateDB); err != nil {
		return nil, fmt.Errorf("create template database: %w", err)
	}
	if err := migrate(ctx, c.url(templateDB)); err != nil {
		return nil, fmt.Errorf("migrate template database: %w", err)
	}
	return c, nil
}

// migrate applies the production migrations, then closes every connection:
// CREATE DATABASE ... TEMPLATE needs a template nobody is connected to.
func migrate(ctx context.Context, dbURL string) error {
	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: dbURL, MaxConns: 2})
	if err != nil {
		return err
	}
	defer pool.Close()
	m, err := postgres.NewMigrator(pool, migrations.FS())
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()
	_, err = m.Up(ctx)
	return err
}
```

- [ ] **Step 12: 添加依赖**

```bash
cd server && go get github.com/jackc/pgx/v5@v5.11.0 github.com/pressly/goose/v3@v3.28.0 github.com/testcontainers/testcontainers-go/modules/postgres@v0.44.0 && go mod tidy
```

testcontainers-go 本身（v0.44.0）作为 postgres 模块的依赖进入 `go.mod` 的间接依赖部分。

Run: `cd server && grep -E '^(go|toolchain) ' go.mod`
Expected: `go 1.27` 和 `toolchain go1.27.1`。

Run: `cd server && sed -n '/^require (/,/^)/p' go.mod | sed '/^)/q'`
Expected:

```
require (
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/jackc/pgx/v5 v5.11.0
	github.com/knadh/koanf/parsers/yaml v1.1.1
	github.com/knadh/koanf/providers/env/v2 v2.0.1
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/providers/rawbytes v1.0.1
	github.com/knadh/koanf/v2 v2.3.6
	github.com/pressly/goose/v3 v3.28.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0
)
```

- [ ] **Step 13: 运行测试，确认通过**

Run: `cd server && go test ./migrations/ ./internal/platform/postgres/...`
Expected（第一次运行要拉取 `postgres:18.6` 和 `testcontainers/ryuk` 镜像，可能多花一些时间）：

```
ok  	github.com/open-nerve/NerveProject/server/migrations
ok  	github.com/open-nerve/NerveProject/server/internal/platform/postgres
ok  	github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest
```

Run: `cd server && go test -short -count=1 -v ./internal/platform/postgres/... | grep -E '^--- SKIP'`
Expected: 需要数据库的 5 个测试被跳过：`TestMigratorUpStatusDown`、`TestMigratorReportsFailingMigration`、`TestDatabasesAreIsolated`、`TestEmptyDatabaseHasNoTables`、`TestDatabaseIsDroppedAfterTheTest`。

Run: `sleep 15 && docker ps --filter ancestor=postgres:18.6 --format '{{.Names}}'`
Expected: 只有 `nerve-dev-db-1`（如果开发库在运行），没有测试遗留的容器。`sleep 15` 是给 Ryuk 留出清理的时间。

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 14: 提交**

```bash
git add server/migrations server/internal/platform/postgres server/go.mod server/go.sum
git commit -m "feat(server): add Postgres pool, goose migrator and pgtest

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: 组合根 bootstrap

**Files:**
- Create: `server/internal/bootstrap/app.go`、`server/internal/bootstrap/commands.go`
- Test: `server/internal/bootstrap/app_test.go`、`server/internal/bootstrap/commands_test.go`

**Interfaces:**
- Consumes: Task 2–7 的全部平台包；`migrations.FS()`；`pgtest`（仅测试）
- Produces（Task 9 的 `cmd/nerve` 调用）：
  - `func Serve(ctx context.Context, cfg config.Config, logOut io.Writer) error`
  - `func MigrateUp(ctx context.Context, cfg config.Config, out io.Writer) error`
  - `func MigrateDown(ctx context.Context, cfg config.Config, out io.Writer) error`
  - `func MigrateStatus(ctx context.Context, cfg config.Config, out io.Writer) error`
  - 包内：`newApp(ctx, cfg, logger, migrationFiles fs.FS) (*app, error)`、`(*app).run(ctx) error`、`(*app).close()`。P3 起在 `newApp` 中把模块挂到路由上

- [ ] **Step 1: 写失败的测试 `server/internal/bootstrap/app_test.go`**

```go
package bootstrap

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"testing/fstest"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// sampleMigrations stands in for the production set, which is empty in M0.
var sampleMigrations = fstest.MapFS{
	"00001_probe_create_widgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE widgets (id bigint);\n-- +goose Down\nDROP TABLE widgets;\n")},
	"00002_probe_create_gadgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE gadgets (id bigint);\n-- +goose Down\nDROP TABLE gadgets;\n")},
}

const unreachableDB = "postgres://nobody@127.0.0.1:1/nowhere"

var client = &http.Client{Timeout: 5 * time.Second}

func testConfig(t *testing.T, dbURL string, autoMigrate bool) config.Config {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return config.Config{
		Env:      config.EnvTest,
		Server:   config.ServerConfig{Addr: addr, ReadHeaderTimeout: time.Second, ShutdownTimeout: 5 * time.Second},
		Database: config.DatabaseConfig{URL: dbURL, MaxConns: 4, AutoMigrate: autoMigrate},
		Log:      config.LogConfig{Level: "error", Format: "text"},
	}
}

// startApp runs the app until the test ends and returns its base URL once it
// answers /healthz. The cleanup checks that run shut down cleanly.
func startApp(t *testing.T, cfg config.Config, migrations fs.FS) string {
	t.Helper()
	a, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), migrations)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run() = %v, want nil after cancel", err)
		}
		a.close()
	})

	base := "http://" + cfg.Server.Addr
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		select {
		case err := <-done:
			t.Fatalf("run() returned early: %v", err)
		default:
		}
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
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, true), slog.New(slog.DiscardHandler), sampleMigrations)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	defer a.close()

	if err := a.run(context.Background()); err == nil {
		t.Error("run() = nil, want the migration error")
	}
}
```

- [ ] **Step 2: 写失败的测试 `server/internal/bootstrap/commands_test.go`**

```go
package bootstrap

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

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

func TestMigrateCommandsWithoutMigrations(t *testing.T) {
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	tests := []struct {
		name string
		run  func(context.Context, *bytes.Buffer) error
		want string
	}{
		{"status", func(ctx context.Context, out *bytes.Buffer) error { return MigrateStatus(ctx, cfg, out) }, "no migrations\n"},
		{"up", func(ctx context.Context, out *bytes.Buffer) error { return MigrateUp(ctx, cfg, out) }, "no pending migrations\n"},
		{"down", func(ctx context.Context, out *bytes.Buffer) error { return MigrateDown(ctx, cfg, out) }, "no applied migrations to roll back\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := tt.run(context.Background(), &out); err != nil {
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
```

- [ ] **Step 3: 运行测试，确认失败**

Run: `cd server && go test ./internal/bootstrap/`
Expected: 编译失败（`[build failed]`），报 `undefined: migrationCommand`、`undefined: runMigration`、`undefined: migrateStatus`、`undefined: newApp` 等。

- [ ] **Step 4: 写 `server/internal/bootstrap/app.go`**

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
	return &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator, handler: mux}, nil
}

// run applies pending migrations when database.auto_migrate is on, then
// serves HTTP on server.addr until ctx is done (see httpserver.Server.Serve).
func (a *app) run(ctx context.Context) error {
	if a.cfg.Database.AutoMigrate {
		applied, err := a.migrator.Up(ctx)
		if err != nil {
			return err
		}
		for _, m := range applied {
			a.logger.InfoContext(ctx, "migration applied", slog.Int64("version", m.Version), slog.String("source", m.Source))
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

- [ ] **Step 5: 写 `server/internal/bootstrap/commands.go`**

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
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Serve implements `nerve serve`: it logs the effective configuration (secrets
// masked) to logOut, applies pending migrations when database.auto_migrate is
// on, and serves HTTP until ctx is done.
func Serve(ctx context.Context, cfg config.Config, logOut io.Writer) error {
	logger, err := logging.New(logOut, cfg.Log)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "configuration loaded", slog.Any("config", cfg))
	a, err := newApp(ctx, cfg, logger, migrations.FS())
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
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		return writeLine(out, "no pending migrations")
	}
	for _, mig := range applied {
		if err := writeLine(out, "applied "+mig.Source); err != nil {
			return err
		}
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

- [ ] **Step 6: 运行测试，确认通过**

Run: `cd server && go test -race ./internal/bootstrap/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/bootstrap`

这里包含了 spec 验收要求的"数据库不可用时 `/readyz` 返回 503"：`TestNotReadyWhenDatabaseIsUnavailable`。

Run: `make lint`
Expected: `0 issues.`

- [ ] **Step 7: 提交**

```bash
git add server/internal/bootstrap
git commit -m "feat(server): add bootstrap composition root and command implementations

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: 命令行 nerve 与 make run

**Files:**
- Create: `server/cmd/nerve/main.go`、`server/cmd/nerve/commands.go`
- Test: `server/cmd/nerve/main_test.go`
- Modify: `Makefile`、`server/go.mod`、`server/go.sum`

**Interfaces:**
- Consumes: Task 8 的 `bootstrap.Serve`、`MigrateUp`、`MigrateDown`、`MigrateStatus`；Task 3 的 `config.Load`、`configs.FS()`；P1 的 `buildinfo.Get()`
- Produces:
  - 命令 `nerve serve`、`nerve migrate up|down|status`、`nerve version`
  - 包内可测试入口 `func run(ctx context.Context, args, environ []string, stdout, stderr io.Writer) int`
  - `make run`

- [ ] **Step 1: 写失败的测试 `server/cmd/nerve/main_test.go`**

```go
package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func execute(ctx context.Context, environ []string, args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(ctx, args, environ, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestVersion(t *testing.T) {
	code, stdout, _ := execute(context.Background(), nil, "version")

	want := "nerve " + buildinfo.Get().Version + " commit="
	if code != 0 || !strings.HasPrefix(stdout, want) {
		t.Errorf("nerve version = %d %q, want 0 and a line starting with %q", code, stdout, want)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	code, _, stderr := execute(context.Background(), nil, "bogus")

	if code != 1 || !strings.Contains(stderr, `nerve: unknown command "bogus"`) {
		t.Errorf("nerve bogus = %d %q, want 1 and an unknown command error", code, stderr)
	}
}

func TestInvalidConfigurationIsReported(t *testing.T) {
	code, _, stderr := execute(context.Background(), []string{"NERVE_ENV=test"}, "migrate", "status")

	if code != 1 || !strings.Contains(stderr, "database.url: is required") {
		t.Errorf("nerve migrate status = %d %q, want 1 and the invalid key", code, stderr)
	}
}

func TestMigrateStatus(t *testing.T) {
	environ := []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=" + pgtest.NewDatabase(t)}

	code, stdout, stderr := execute(context.Background(), environ, "migrate", "status")

	if code != 0 || stdout != "no migrations\n" {
		t.Errorf("nerve migrate status = %d %q (stderr %q), want 0 and \"no migrations\"", code, stdout, stderr)
	}
}

func TestServeUntilCancelled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	environ := []string{
		"NERVE_ENV=test",
		"NERVE_DATABASE__URL=" + pgtest.NewDatabase(t),
		"NERVE_SERVER__ADDR=" + addr,
		"NERVE_LOG__LEVEL=info",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		code   int
		stderr string
	}
	done := make(chan result, 1)
	go func() {
		code, _, stderr := execute(ctx, environ, "serve")
		done <- result{code, stderr}
	}()

	client := &http.Client{Timeout: time.Second}
	ready := false
	for deadline := time.Now().Add(10 * time.Second); !ready && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if resp, err := client.Get("http://" + addr + "/readyz"); err == nil {
			_ = resp.Body.Close()
			ready = resp.StatusCode == http.StatusOK
		}
	}
	cancel()
	r := <-done

	if !ready {
		t.Fatalf("nerve serve never became ready; stderr:\n%s", r.stderr)
	}
	if r.code != 0 {
		t.Errorf("nerve serve exit code = %d, want 0; stderr:\n%s", r.code, r.stderr)
	}
	for _, want := range []string{"configuration loaded", "postgres://postgres:xxxxx@", "http server stopped"} {
		if !strings.Contains(r.stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, r.stderr)
		}
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd server && go test ./cmd/nerve/`
Expected: 编译失败（`[build failed]`），报 `undefined: run`。

- [ ] **Step 3: 写 `server/cmd/nerve/main.go`**

```go
// Command nerve runs the Nerve server and its operational commands.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// After the first signal starts the graceful shutdown, restore the
	// default handling so a second signal stops the process at once.
	context.AfterFunc(ctx, stop)
	code := run(ctx, os.Args[1:], os.Environ(), os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// run executes one command line and returns the process exit code. Results go
// to stdout; logs and errors go to stderr.
func run(ctx context.Context, args, environ []string, stdout, stderr io.Writer) int {
	root := newRootCommand(environ)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "nerve: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: 写 `server/cmd/nerve/commands.go`**

```go
package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/bootstrap"
	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// localConfigFile is the personal override file, relative to the working
// directory: `make run` starts nerve in server/.
const localConfigFile = "configs/config.local.yaml"

type configLoader func() (config.Config, error)

func newRootCommand(environ []string) *cobra.Command {
	root := &cobra.Command{
		Use:           "nerve",
		Short:         "Nerve: a lightweight project management server",
		SilenceErrors: true, // run prints the error once
		SilenceUsage:  true,
	}
	root.CompletionOptions.DisableDefaultCmd = true

	load := func() (config.Config, error) {
		return config.Load(config.Sources{Embedded: configs.FS(), Environ: environ, LocalFile: localConfigFile})
	}
	root.AddCommand(newServeCommand(load), newMigrateCommand(load), newVersionCommand())
	return root
}

func newServeCommand(load configLoader) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP server until SIGINT or SIGTERM",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return bootstrap.Serve(cmd.Context(), cfg, cmd.ErrOrStderr())
		},
	}
}

func newMigrateCommand(load configLoader) *cobra.Command {
	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Manage the database schema",
	}
	sub := func(use, short string, action func(context.Context, config.Config, io.Writer) error) *cobra.Command {
		return &cobra.Command{
			Use:   use,
			Short: short,
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, err := load()
				if err != nil {
					return err
				}
				return action(cmd.Context(), cfg, cmd.OutOrStdout())
			},
		}
	}
	migrate.AddCommand(
		sub("up", "Apply all pending migrations", bootstrap.MigrateUp),
		sub("down", "Roll back the most recent migration", bootstrap.MigrateDown),
		sub("status", "List migrations and whether they are applied", bootstrap.MigrateStatus),
	)
	return migrate
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the build version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := buildinfo.Get()
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "nerve %s commit=%s commit_time=%s modified=%t\n",
				info.Version, info.Commit, info.CommitTime, info.Modified)
			return err
		},
	}
}
```

- [ ] **Step 5: 添加依赖**

```bash
cd server && go get github.com/spf13/cobra@v1.10.2 && go mod tidy
```

Run: `cd server && grep -E '^(go|toolchain) ' go.mod && grep -E '^	github.com/spf13/cobra ' go.mod`
Expected: `go 1.27`、`toolchain go1.27.1`、`	github.com/spf13/cobra v1.10.2`。

- [ ] **Step 6: 运行测试，确认通过**

Run: `cd server && go test -race ./cmd/nerve/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/cmd/nerve`

- [ ] **Step 7: 写 `Makefile`（完整内容：在 `dev-db-reset` 之后新增 `run`，其余不变）**

注意：命令体必须用 **Tab** 缩进。

```make
# Nerve 开发命令入口。运行 `make` 或 `make help` 查看所有命令。
# 需兼容 macOS 自带的 GNU Make 3.81。

SHELL := /bin/bash
.DEFAULT_GOAL := help

DEV_COMPOSE := docker compose -f deploy/compose.dev.yaml
GOLANGCI_LINT_VERSION := 2.13.2
BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint

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

.PHONY: lint
lint: tools ## 运行 golangci-lint（server）
	cd server && $(GOLANGCI_LINT) run ./...

.PHONY: test
test: ## 运行 Go 测试（server）
	cd server && go test ./...
```

Run: `make`
Expected: 共 8 个命令，`run` 位于 `dev-db-reset` 和 `tools` 之间，说明为"以 dev 配置运行后端（需先 make dev-db），Ctrl-C 停止"。

- [ ] **Step 8: 手工验证 serve（手工验证）**

开发库在运行（`make dev-db`），8080 端口空闲（`lsof -nP -iTCP:8080 -sTCP:LISTEN` 没有输出）。

在一个终端执行 `make run`。Expected: 日志中先有一行 `level=INFO msg="configuration loaded" config.env=dev … config.database.url="postgres://nerve:xxxxx@localhost:55432/nerve?sslmode=disable" …`（密码已打码），然后是 `msg="http server listening" addr=[::]:8080`。

在另一个终端：

```bash
curl -si localhost:8080/healthz
curl -si localhost:8080/readyz
curl -si -H 'X-Request-Id: manual-check-1' localhost:8080/api/v0/nope
```

Expected:
- `/healthz` 和 `/readyz`：`HTTP/1.1 200 OK`，`Content-Type: application/json`，响应头带 `X-Request-Id`（UUIDv7），响应体 `{"status":"ok"}`。
- `/api/v0/nope`：`HTTP/1.1 404 Not Found`，`Content-Type: application/problem+json`，`X-Request-Id: manual-check-1`，响应体 `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}`。
- `make run` 的终端里每个请求一行 `msg="http request"`，带 `request_id`、`method`、`path`、`status`、`duration`。

对服务进程发送 SIGTERM（`go run` 编译出的进程才是服务本身）：

```bash
kill -TERM "$(lsof -nP -iTCP:8080 -sTCP:LISTEN -t)"
```

Expected: `make run` 的终端输出 `msg="http server shutting down" timeout=20s` 和 `msg="http server stopped"`，`make run` 以退出码 0 结束。

- [ ] **Step 9: 手工验证数据库不可用、迁移命令和版本（手工验证）**

```bash
cd server && go build -o ../bin/nerve ./cmd/nerve && cd ..
cd server && NERVE_DATABASE__URL=postgres://nerve:hunter2@127.0.0.1:1/nerve NERVE_DATABASE__AUTO_MIGRATE=false NERVE_SERVER__ADDR=127.0.0.1:8099 ../bin/nerve serve
```

在另一个终端：

```bash
curl -si 127.0.0.1:8099/healthz
curl -si 127.0.0.1:8099/readyz
kill -INT "$(lsof -nP -iTCP:8099 -sTCP:LISTEN -t)"
```

Expected:
- `/healthz` 为 200；`/readyz` 为 `HTTP/1.1 503 Service Unavailable`，`Content-Type: application/problem+json`，响应体 `{"status":503,"code":"not_ready","title":"Service Unavailable","detail":"database is not ready"}`。
- 服务日志中有 `level=WARN msg="readiness check failed" … check=database error="failed to connect to …connection refused…"`；启动日志中的地址是 `postgres://nerve:xxxxx@127.0.0.1:1/nerve`，看不到 `hunter2`。
- SIGINT 之后输出 `msg="http server stopped"`，退出码 0。

```bash
cd server && ../bin/nerve migrate status && ../bin/nerve migrate up && ../bin/nerve migrate down && ../bin/nerve version && cd ..
```

Expected（开发库，M0 没有迁移文件）：

```
no migrations
no pending migrations
no applied migrations to roll back
nerve 0.1.0-dev commit=<提交号> commit_time=<提交时间> modified=true
```

`modified=true` 是因为工作区有未提交的改动。

```bash
cd server && NERVE_ENV=prod ../bin/nerve migrate status; echo "exit=$?"; cd ..
```

Expected:

```
nerve: invalid configuration:
database.url: is required
exit=1
```

- [ ] **Step 10: lint、完整测试**

Run: `make lint`
Expected: `0 issues.`

Run: `make test`
Expected: 所有包都是 `ok`。

Run: `git status --short`
Expected: `M Makefile`、`M server/go.mod`、`M server/go.sum`、`?? server/cmd/`。`bin/` 已被忽略。

- [ ] **Step 11: 提交**

```bash
git add Makefile server/cmd server/go.mod server/go.sum
git commit -m "feat(server): add nerve CLI and make run

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: 架构测试

**Files:**
- Create（全部是测试文件）：`server/internal/archtest/rules_test.go`、`server/internal/archtest/rules_cases_test.go`、`server/internal/archtest/repo_test.go`
- Modify: `server/go.mod`、`server/go.sum`

**Interfaces:**
- Consumes: 整个模块的导入关系
- Produces: 随 `go test ./...` 运行的 8 条架构规则（spec 2.10）。以后新增规则：在 `rules()` 中加一条，并在 `TestRules` 的表格中加上违规和合法的例子

- [ ] **Step 1: 写失败的测试 `server/internal/archtest/rules_cases_test.go`**

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
		pure     = "domain imports only the standard library (not net/http or database/sql), its own module and internal/shared"
		isolated = "modules do not import each other"
		business = "platform does not import modules or bootstrap"
		entry    = "only bootstrap imports modules"
		gen      = "generated code is imported only by its module's http adapter"
		platform = "platform packages do not import each other, except config"
		pgtest   = "pgtest is imported only by tests"
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

		// Domain purity.
		{m("internal/modules/issue/domain"), "time", nil},
		{m("internal/modules/issue/domain"), m("internal/shared/id"), nil},
		{m("internal/modules/issue/domain"), "net/http", []string{pure}},
		{m("internal/modules/issue/domain"), "database/sql/driver", []string{pure}},
		{m("internal/modules/issue/domain"), "github.com/jackc/pgx/v5", []string{pure}},
		{m("internal/modules/issue/domain"), m("internal/platform/config"), []string{pure}},
		{m("internal/modules/issue/app"), "github.com/jackc/pgx/v5", nil},

		// Module isolation and the composition root.
		{m("internal/modules/issue/app"), m("internal/modules/project/domain"), []string{isolated}},
		{m("internal/bootstrap"), m("internal/modules/issue"), nil},
		{m("cmd/nerve"), m("internal/modules/issue"), []string{entry}},
		{m("internal/platform/httpserver"), m("internal/modules/issue"), []string{business, entry}},
		{m("internal/platform/httpserver"), m("internal/bootstrap"), []string{business}},

		// Generated code.
		{m("internal/modules/issue/adapter/http"), m("internal/modules/issue/adapter/http/gen"), nil},
		{m("internal/modules/issue/app"), m("internal/modules/issue/adapter/http/gen"), []string{inward, gen}},
		{m("internal/modules/issue/adapter/postgres"), m("internal/modules/issue/adapter/http/gen"), []string{gen}},

		// Platform packages.
		{m("internal/platform/logging"), m("internal/platform/config"), nil},
		{m("internal/platform/postgres/pgtest"), m("internal/platform/postgres"), nil},
		{m("internal/platform/httpserver"), m("internal/platform/postgres"), []string{platform}},

		// pgtest.
		{m("internal/bootstrap"), m("internal/platform/postgres/pgtest"), []string{pgtest}},
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

- [ ] **Step 2: 写失败的测试 `server/internal/archtest/repo_test.go`**

```go
package archtest

import (
	"io/fs"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/tools/go/packages"
)

// moduleRoot is server/, relative to this package: go test runs a test in its
// package directory.
const moduleRoot = "../.."

// loadGraph reads the import graph of every non-test package of the module.
func loadGraph(t *testing.T) graph {
	t.Helper()
	// go list runs in a subprocess, so the go test result cache would not see
	// edited or new source files and could replay a stale pass. Walking the
	// tree here makes every directory listing (with file sizes and times) an
	// input of this test.
	if err := filepath.WalkDir(moduleRoot, func(_ string, _ fs.DirEntry, err error) error { return err }); err != nil {
		t.Fatalf("walk %s: %v", moduleRoot, err)
	}
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports, Dir: moduleRoot}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		t.Fatalf("load packages: %v", err)
	}
	g := graph{}
	for _, p := range pkgs {
		for _, e := range p.Errors {
			t.Errorf("load %s: %v", p.PkgPath, e)
		}
		imports := make([]string, 0, len(p.Imports))
		for path := range p.Imports {
			imports = append(imports, path)
		}
		slices.Sort(imports)
		g[p.PkgPath] = imports
	}
	return g
}

func TestRepositoryFollowsArchitectureRules(t *testing.T) {
	g := loadGraph(t)
	// A loader problem must not pass as "no violations".
	for _, want := range []string{"cmd/nerve", "internal/bootstrap", "internal/platform/config"} {
		if _, ok := g[m(want)]; !ok {
			t.Fatalf("import graph lacks %s; loaded %d packages", want, len(g))
		}
	}
	for _, v := range check(g) {
		t.Error(v)
	}
}
```

- [ ] **Step 3: 添加依赖**

```bash
cd server && go get golang.org/x/tools@v0.50.0 && go mod tidy
```

Run: `cd server && grep -E '^(go|toolchain) ' go.mod && grep -E '^	golang.org/x/tools ' go.mod`
Expected: `go 1.27`、`toolchain go1.27.1`、`	golang.org/x/tools v0.50.0`。

- [ ] **Step 4: 运行测试，确认失败**

Run: `cd server && go test ./internal/archtest/`
Expected: 编译失败（`[build failed]`），报 `undefined: graph`、`undefined: check`、`undefined: modulePath`、`undefined: rules` 等。

- [ ] **Step 5: 写规则 `server/internal/archtest/rules_test.go`**

```go
// Package archtest enforces the architecture rules of M0 design 3.7 (and the
// platform rules of 3.1) as tests. Each rule is a pure predicate over one
// import edge, so rules are unit-tested on synthetic edges and then applied
// to the real import graph of the module.
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
		{"domain imports only the standard library (not net/http or database/sql), its own module and internal/shared", domainIsPure},
		{"modules do not import each other", modulesAreIsolated},
		{"platform does not import modules or bootstrap", platformIsBusinessFree},
		{"only bootstrap imports modules", onlyBootstrapImportsModules},
		{"generated code is imported only by its module's http adapter", generatedCodeStaysInAdapter},
		{"platform packages do not import each other, except config", platformPackagesAreIndependent},
		{"pgtest is imported only by tests", pgtestOnlyInTests},
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

func domainIsPure(from, to string) bool {
	if _, layer, ok := moduleOf(from); !ok || layer != "domain" {
		return false
	}
	if r, ok := local(to); ok {
		_, _, inModule := moduleOf(to) // module imports are covered by the layer and isolation rules
		return !inModule && !within(r, "internal/shared")
	}
	if !isStdlib(to) {
		return true
	}
	return within(to, "net/http") || within(to, "database/sql")
}

func modulesAreIsolated(from, to string) bool {
	fm, _, ok := moduleOf(from)
	if !ok {
		return false
	}
	tm, _, ok := moduleOf(to)
	return ok && fm != tm
}

func platformIsBusinessFree(from, to string) bool {
	return inModuleDir(from, "internal/platform") &&
		(inModuleDir(to, "internal/modules") || inModuleDir(to, "internal/bootstrap"))
}

func onlyBootstrapImportsModules(from, to string) bool {
	return inModuleDir(to, "internal/modules") &&
		!inModuleDir(from, "internal/modules") && !inModuleDir(from, "internal/bootstrap")
}

func generatedCodeStaysInAdapter(from, to string) bool {
	name, _, ok := moduleOf(to)
	if !ok {
		return false
	}
	adapter := "internal/modules/" + name + "/adapter/http"
	gen := adapter + "/gen"
	if !inModuleDir(to, gen) {
		return false
	}
	r, _ := local(from)
	return r != adapter && !within(r, gen)
}

func platformPackagesAreIndependent(from, to string) bool {
	fp, ok := platformOf(from)
	if !ok {
		return false
	}
	tp, ok := platformOf(to)
	return ok && tp != fp && tp != "config"
}

func pgtestOnlyInTests(_, to string) bool {
	// The graph holds no test files, so any importer is production code.
	return inModuleDir(to, "internal/platform/postgres/pgtest")
}
```

- [ ] **Step 6: 运行测试，确认通过**

Run: `cd server && go test -v ./internal/archtest/`
Expected: `--- PASS: TestRepositoryFollowsArchitectureRules`、`--- PASS: TestRules`，最后是 `ok  	github.com/open-nerve/NerveProject/server/internal/archtest`。

- [ ] **Step 7: 演示：故意写一个违规的导入，架构测试必须报错（验证后删除）**

先连跑两次，确认通过的结果已经进了测试缓存（上一步带了 `-v`，缓存键不同）：

Run: `cd server && go test ./internal/archtest/ && go test ./internal/archtest/`
Expected: 两行 `ok`，第二行以 `(cached)` 结尾。

加一个"领域层导入 `net/http`"的模块包：

```bash
mkdir -p server/internal/modules/probe/domain
printf 'package domain\n\nimport _ "net/http"\n' > server/internal/modules/probe/domain/probe.go
```

Run: `cd server && go test ./internal/archtest/`
Expected: 失败（不是缓存的结果），报：

```
internal/modules/probe/domain imports net/http: domain imports only the standard library (not net/http or database/sql), its own module and internal/shared
```

这同时证明了 `repo_test.go` 中遍历目录的做法有效：没有它，这一步会直接返回缓存的通过结果（spec 2.10）。

删除违规：

```bash
rm -r server/internal/modules
```

Run: `cd server && go test ./internal/archtest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/archtest`

- [ ] **Step 8: lint、完整测试**

Run: `make lint`
Expected: `0 issues.`

Run: `make test`
Expected: 所有包都是 `ok`。

Run: `git status --short`
Expected: `M server/go.mod`、`M server/go.sum`、`?? server/internal/archtest/`（没有遗留的 `server/internal/modules/`）。

- [ ] **Step 9: 提交**

```bash
git add server/internal/archtest server/go.mod server/go.sum
git commit -m "test(server): add architecture tests

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: 文档收尾与完整走查

**Files:**
- Modify: `README.md`
- Modify: `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-go-db-notes.md`
- 核对（不改）：`docs/v0/M0-foundation/M0-design.md` 第 11 节

**Interfaces:**
- Consumes: Task 1–10 的全部成果
- Produces: 开发者上手说明；关闭 P1 移交给 P2 的交接事项

- [ ] **Step 1: 更新 `README.md` 的"开发环境"一节**

把从 `第一次启动：` 这一行开始、到 `## 版权` 之前的全部内容，替换为下面的内容（"需要安装"那部分不变）：

````markdown
第一次启动：

```bash
make dev-db   # 启动本地 Postgres 18（端口 55432，可用 NERVE_DEV_DB_PORT 修改）
make test     # 运行测试（需要 Docker：集成测试用 testcontainers 启动 Postgres）
make run      # 以 dev 配置运行后端，监听 :8080；Ctrl-C 停止
make          # 查看所有命令
```

- 只跑单元测试、跳过需要 Docker 的集成测试：在 `server/` 下执行 `go test -short ./...`。
- 后端的配置文件在 `server/configs/`，任何一项都可以用环境变量覆盖：`NERVE_` 加上配置路径，层级之间用双下划线，例如 `database.url` 对应 `NERVE_DATABASE__URL`。个人的本地覆盖写在 `server/configs/config.local.yaml`（不进仓库，只在 dev 环境生效）。
- **换了开发库的端口时**：`NERVE_DEV_DB_PORT` 只改变 compose 映射的端口，`server/configs/config.dev.yaml` 中的数据库地址仍然是 55432。还需要覆盖 `database.url`，例如：

  ```bash
  NERVE_DEV_DB_PORT=55433 make dev-db
  NERVE_DATABASE__URL=postgres://nerve:nerve@localhost:55433/nerve?sslmode=disable make run
  ```

  或者在 `server/configs/config.local.yaml` 中写：

  ```yaml
  database:
    url: postgres://nerve:nerve@localhost:55433/nerve?sslmode=disable
  ```

````

- [ ] **Step 2: 关闭 P1 的交接事项**

在 `docs/v0/M0-foundation/handoffs/P1-repo-toolchain-go-db-notes.md` 中：
1. 把开头的 `status: open` 改为 `status: done`。
2. 在文末"来源："一行之前插入：

```markdown
## 处理结果（M0/P2）

1. P2 加入的全部依赖（见 [P2 spec](../specs/P2-server-platform.md) 2.2）都已核实不会抬高 `go` 行；P2 plan 的每个 Task 在 `go get` / `go mod tidy` 之后都检查了 `server/go.mod` 仍是 `go 1.27`。
2. P2 没有改动任何 `toolchain` 行，`server/go.mod` 和 `server/tools/go.mod` 都是 `toolchain go1.27.1`。
3. README 的"开发环境"一节和 `server/configs/config.dev.yaml` 的注释都写明了：用 `NERVE_DEV_DB_PORT` 换端口时，还要用 `NERVE_DATABASE__URL` 或 `config.local.yaml` 覆盖 `database.url`。

```

- [ ] **Step 3: 核对 M0 设计文档的 Phase 进度表**

Run: `grep -n '^| P2 ' docs/v0/M0-foundation/M0-design.md`
Expected（spec 和 plan 提交时已经更新，这里不需要改动；review 链接在代码评审后补上）：

```
| P2 | server-platform | 进行中 | [spec](specs/P2-server-platform.md) | [plan](plans/P2-server-platform.md) | — |
```

- [ ] **Step 4: 按 README 从头走一遍（完整走查）**

```bash
make dev-db
make test
make lint
make
```

Expected: `make dev-db` 输出 `Container nerve-dev-db-1  Healthy`；`make test` 所有包都是 `ok`；`make lint` 输出 `0 issues.`；`make` 列出 8 个命令。

然后执行 `make run`，在另一个终端：

```bash
curl -s localhost:8080/healthz; echo
curl -s localhost:8080/readyz; echo
```

Expected: 两行 `{"status":"ok"}`。在 `make run` 的终端按 Ctrl-C，输出 `msg="http server stopped"` 后退出。

Run: `cd server && grep -E '^(go|toolchain) ' go.mod tools/go.mod`
Expected:

```
go.mod:go 1.27
go.mod:toolchain go1.27.1
tools/go.mod:go 1.27
tools/go.mod:toolchain go1.27.1
```

- [ ] **Step 5: 提交**

```bash
git add README.md docs/v0/M0-foundation/handoffs/P1-repo-toolchain-go-db-notes.md
git commit -m "docs(M0/P2): document make run and close the P1 go/db handoff

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 6: 推送并确认持续集成通过（由控制者执行）**

推送分支后，确认 `CI` 的 `server` 任务通过（`make lint` 和 `make test`，其中包含 testcontainers 集成测试）。持续集成的工作流在 P2 中不需要改动（spec 2.11）。

---

## 完成后

P2 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M0-foundation/reviews/P2-server-platform-review.md`，并把 M0 设计文档中 P2 的状态改为"已完成"、补上 review 链接。
