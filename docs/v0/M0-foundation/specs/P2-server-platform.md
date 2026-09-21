# M0/P2 服务端平台层：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M0/P2 `server-platform` |
| 日期 | 2026-09-22 |
| 状态 | 已批准 |
| 上级文档 | [M0 设计文档](../M0-design.md) 第 1、2、3.1、3.3、3.5–3.8、6.1、8 节；[v0 总体设计](../../v0-design.md) 3.5、6.1–6.4、6.8 节 |
| 前置交接 | [P1-repo-toolchain-go-db-notes](../handoffs/P1-repo-toolchain-go-db-notes.md)（三项都在本 Phase 处理，见 2.11） |

## 1. 目标
搭好 Go 服务端与业务无关的平台层，让 P3 起的模块只需要"写模块、在 `bootstrap` 里接线"：
- 分环境的配置（内嵌进程序、可用环境变量覆盖、启动即校验、日志中打码）；
- slog 日志；
- Postgres 连接池、goose 迁移执行器、集成测试用的 `pgtest`；
- HTTP 服务：固定顺序的三个中间件、problem+json、`/healthz`、`/readyz`、`/api/` 兜底 404、优雅停机；
- 唯一的组合根 `bootstrap`；命令行 `nerve serve | migrate up|down|status | version`；
- 架构守护：`internal/archtest` 和 depguard；
- `make run`。

M0 没有任何迁移文件，迁移机制由只存在于测试中的迁移集验证。

## 2. 交付物

### 2.1 文件总览
路径都相对于 `server/`。

| 路径 | 内容 |
|---|---|
| `.golangci.yml` | golangci-lint v2 配置：standard 规则集 + depguard；格式检查 gofmt、goimports（2.10） |
| `configs/embed.go`，`configs/config{,.dev,.test,.prod}.yaml` | 内嵌的配置文件（2.3） |
| `migrations/embed.go`，`migrations/sql/.gitkeep` | 内嵌的迁移目录（2.5） |
| `internal/platform/config/` | 配置类型、加载、校验、打码 |
| `internal/platform/logging/` | slog 初始化 |
| `internal/platform/postgres/` | 连接池、迁移执行器 |
| `internal/platform/postgres/pgtest/` | 集成测试数据库（只能被测试代码导入） |
| `internal/platform/httpserver/` | 中间件、problem+json、平台路由、服务生命周期 |
| `internal/bootstrap/` | 组合根和各命令的实现 |
| `cmd/nerve/` | cobra 命令行，只做解析和转交 |
| `internal/archtest/` | 架构测试（只有测试文件） |
| 仓库根目录 `Makefile` | 新增 `make run` |
| 仓库根目录 `README.md` | 开发环境一节补充 `make run`、测试需要 Docker、换开发库端口的说明 |

包之间的依赖（箭头表示"导入"，均为非测试代码）：
```
cmd/nerve ──→ bootstrap, platform/config, platform/buildinfo, configs
bootstrap ──→ platform/{config, logging, postgres, httpserver}, migrations
platform/{logging, postgres, httpserver} ──→ platform/config
platform/postgres/pgtest ──→ platform/postgres, platform/config, migrations   （只被 _test.go 导入）
```

### 2.2 依赖版本（2026-09-22 通过 Go 模块代理核实，写死）

| 模块 | 版本 | 用途 |
|---|---|---|
| `github.com/jackc/pgx/v5` | v5.11.0 | 连接池；`stdlib.OpenDBFromPool` 给 goose 提供 `*sql.DB` |
| `github.com/pressly/goose/v3` | v3.28.0 | 迁移（Provider API） |
| `github.com/knadh/koanf/v2` | v2.3.6 | 配置分层 |
| `github.com/knadh/koanf/parsers/yaml` | v1.1.1 | YAML 解析 |
| `github.com/knadh/koanf/providers/rawbytes` | v1.0.1 | 读取内嵌文件 |
| `github.com/knadh/koanf/providers/file` | v1.2.1 | 读取 `NERVE_CONFIG_DIR` 和 `config.local.yaml` |
| `github.com/knadh/koanf/providers/env/v2` | v2.0.1 | 环境变量 |
| `github.com/go-viper/mapstructure/v2` | v2.5.0 | koanf 解码用的库；直接导入以配置严格解码（2.3） |
| `github.com/spf13/cobra` | v1.10.2 | 命令行 |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.44.0 | `pgtest`（testcontainers-go v0.44.0 作为它的间接依赖） |
| `golang.org/x/tools` | v0.50.0 | archtest 用 `go/packages` 读取导入关系 |

- 以上都没有抬高 `go 1.27` 这一行（每次 `go mod tidy` 之后检查，见 2.11）。
- 生产二进制只链接 pgx、goose、koanf、mapstructure、cobra 及其依赖；testcontainers 只出现在测试二进制中：只有 `pgtest` 导入它，而 archtest 规则 8 保证 `pgtest` 只被测试代码导入（已用 `go version -m` 核实）。

### 2.3 配置：`platform/config` 与 `server/configs`

**配置项**（`configs/config.yaml` 列出全部配置项及默认值）：

| 键 | 类型 | 默认值 | dev | test | prod |
|---|---|---|---|---|---|
| `server.addr` | host:port | `":8080"` | | | |
| `server.read_header_timeout` | 时长 | `5s` | | | |
| `server.shutdown_timeout` | 时长 | `20s` | | | |
| `database.url` | 字符串，必填 | `""` | `postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable` | 由环境变量提供 | 由环境变量提供 |
| `database.max_conns` | 整数 ≥ 1 | `10` | | | |
| `database.auto_migrate` | 布尔 | `true` | | | `false` |
| `log.level` | debug / info / warn / error | `info` | `debug` | `warn` | |
| `log.format` | text / json | `json` | `text` | `text` | |

M0 设计 3.6 中的 `app.name` 和 `web.enabled` 不在 P2 加入，见第 3 节。

**内嵌**：`configs/embed.go`（包 `configs`）用 `//go:embed config.yaml config.dev.yaml config.test.yaml config.prod.yaml` 列出文件名，所以 `config.local.yaml` 永远不会被编进程序。对外只有 `func FS() fs.FS`，内嵌变量不导出，避免被其他包改写。

**加载顺序**（后加载的覆盖先加载的，逐个键覆盖）：
1. 内置 `config.yaml`：所有键及默认值。它就是总体设计 6.8 中的"代码中的默认值"：已经编进程序，不再在 Go 代码里重复一份默认值。
2. 内置 `config.<env>.yaml`。
3. 设置了 `NERVE_CONFIG_DIR` 时：该目录下的 `config.yaml`，然后是 `config.<env>.yaml`。文件不存在就跳过；目录本身不存在则报错。
4. 仅 dev 环境：`Sources.LocalFile`（`cmd/nerve` 传入 `configs/config.local.yaml`，相对当前工作目录；`make run` 在 `server/` 下运行，所以就是 `server/configs/config.local.yaml`）。不存在就跳过。
5. 环境变量 `NERVE_<SECTION>__<KEY>`。

`<env>` 来自 `NERVE_ENV`，取值 `dev`（默认）、`test`、`prod`，其他值直接报错。

**环境变量映射**：
- 去掉前缀 `NERVE_`，`__` 换成 `.`，再转小写：`NERVE_DATABASE__URL` → `database.url`，`NERVE_DATABASE__MAX_CONNS` → `database.max_conns`（单个 `_` 保留在键名里）。
- **没有 `__` 的 `NERVE_*` 变量不是配置键，直接忽略**。理由：所有配置键都在某个分组之下（至少两级），真正的配置键一定含 `__`。这样 `NERVE_ENV`、`NERVE_CONFIG_DIR` 这两个控制变量不会漏进配置树，`NERVE_DEV_DB_PORT`（compose 用）这类同前缀的变量也不会被误认为配置键。
- 环境变量的值都是字符串，解码时按目标类型转换（`"false"` → bool，`"20"` → int32）。

**解码与校验**（`Load` 的最后两步，任何一步出错都返回 `invalid configuration:` 开头、每行一个问题的错误）：
- **严格解码**：用 mapstructure 的 `ErrorUnused`，配置文件或环境变量中出现未知的键（拼写错误）直接报错，例如 `'database' has invalid keys: max_con`。不会因为拼错而悄悄用了默认值。
- **时长**：只接受 Go 时长字符串（`5s`、`1m30s`）。纯数字被拒绝，因为 YAML 中的 `20` 否则会被当作 20 纳秒。错误例如 `'server.shutdown_timeout' time: invalid duration "soon"`。
- **类型错误**（例如 `max_conns: many`）由解码报告，并指明键名。
- **取值校验**（解码成功后，一次列出所有问题）：

  | 键 | 规则 | 错误信息 |
  |---|---|---|
  | `server.addr` | `net.SplitHostPort` 能解析 | `server.addr: must be host:port, e.g. ":8080", got "8080"` |
  | `server.read_header_timeout` | > 0 | `server.read_header_timeout: must be positive, got 0s` |
  | `server.shutdown_timeout` | > 0 | `server.shutdown_timeout: must be positive, got -1s` |
  | `database.url` | 非空 | `database.url: is required` |
  | `database.max_conns` | ≥ 1 | `database.max_conns: must be at least 1, got 0` |
  | `log.level` | slog 能解析 | `log.level: must be one of debug, info, warn, error, got "verbose"` |
  | `log.format` | `text` 或 `json` | `log.format: must be text or json, got "xml"` |

  `database.url` 的格式不在这里校验，由 pgx 在创建连接池时解析（pgx 的解析错误会把密码打码）。

**打码**：`Config` 实现 `slog.LogValuer`。启动日志 `logger.Info("configuration loaded", "config", cfg)` 输出所有配置项，`database.url` 中的密码（用户信息中的密码和 `password` 查询参数）替换为 `xxxxx`；不是 URL 形式的连接串（`host=… password=…`）整体替换为 `xxxxx`。日志中只出现 `LogValue` 明确列出的字段，以后新增的密钥类配置项不会被意外打印出来。

**接口**：
```go
package config

const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

type Config struct {
	Env      string         `koanf:"-"` // 来自 NERVE_ENV，不是配置键
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	Log      LogConfig      `koanf:"log"`
}
type ServerConfig struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
}
type DatabaseConfig struct {
	URL         string `koanf:"url"`
	MaxConns    int32  `koanf:"max_conns"`
	AutoMigrate bool   `koanf:"auto_migrate"`
}
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

func (c Config) LogValue() slog.Value

type Sources struct {
	Embedded  fs.FS    // 内置的配置文件（configs.FS()）
	Environ   []string // os.Environ() 形式的环境变量
	LocalFile string   // 个人覆盖文件，仅 dev 环境、存在时读取
}

func Load(src Sources) (Config, error)
```
```go
package configs

func FS() fs.FS
```
- 环境变量以参数传入，而不是在包里读 `os.Environ()`：测试可以并行，不需要 `t.Setenv`。
- 校验函数不导出：唯一的入口是 `Load`。

### 2.4 日志：`platform/logging`
```go
func New(w io.Writer, cfg config.LogConfig) (*slog.Logger, error)
```
- 按 `log.format` 选 `slog.NewJSONHandler` 或 `slog.NewTextHandler`，按 `log.level` 设置级别。
- **不调用 `slog.SetDefault`**（不改全局状态）。logger 由 `bootstrap` 创建，通过构造函数传给需要的地方。
- 日志写到 stderr；命令的结果（`version`、`migrate`）写到 stdout。

### 2.5 数据库与迁移：`platform/postgres` 与 `server/migrations`

**连接池**：
```go
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error)
```
- `pgxpool.ParseConfig(database.url)`，再把 `MaxConns` 设为 `database.max_conns`（覆盖 URL 中的 `pool_max_conns`）。
- **不在创建时连接数据库**（pgxpool 默认按需建立连接）。数据库不可用时，`nerve serve` 照样能启动（`auto_migrate=false` 或没有迁移文件时），由 `/readyz` 报告 503。
- 解析失败时返回 `database.url: …`，pgx 会把错误中的密码打码。

**迁移文件的内嵌**：`migrations/embed.go`（包 `migrations`）用 `//go:embed all:sql` 内嵌整个目录（`all:` 让 `.gitkeep` 也算在内；否则目录里没有 `.sql` 文件时编译失败），`func FS() fs.FS` 返回以 `sql/` 为根的文件系统。命名规则 `NNNNN_<模块>_<说明>.sql` 由 `migrations/embed_test.go` 检查（M0 没有文件，测试自然通过；M2 起生效）。

**迁移执行器**：
```go
var ErrPendingMigrations = errors.New("database has pending migrations")

type Migration struct {
	Version int64
	Source  string // 文件名，例如 "00001_identity_create_users.sql"
}
type MigrationStatus struct {
	Migration
	Applied   bool
	AppliedAt time.Time // 未执行时为零值
}
type Migrator struct{ /* 未导出字段 */ }

func NewMigrator(pool *pgxpool.Pool, fsys fs.FS) (*Migrator, error)
func (m *Migrator) Up(ctx context.Context) ([]Migration, error)     // 按版本顺序执行所有待执行的迁移
func (m *Migrator) Down(ctx context.Context) (*Migration, error)    // 回滚最近一个；没有可回滚的返回 nil, nil
func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error)
func (m *Migrator) CheckUpToDate(ctx context.Context) error         // 有待执行的迁移时返回 ErrPendingMigrations
func (m *Migrator) Close() error                                    // 在关闭连接池之前调用
```
- 内部用 `goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(pool), fsys, goose.WithDisableGlobalRegistry(true))`。关闭 goose 的全局 Go 迁移注册表，不依赖包级状态。
- **没有迁移文件**：goose 返回 `ErrNoMigrations`，`NewMigrator` 把它当作"无事可做"，返回一个所有操作都是空操作的 `Migrator`：`Up` 返回空、`Down` 返回 nil、`Status` 返回空、`CheckUpToDate` 返回 nil，并且**完全不访问数据库**。
- 对外只暴露自己的类型，不把 goose 的类型泄露给调用方。
- `CheckUpToDate` 的签名正好符合就绪检查（2.7）。注意 goose 的 `HasPending` 在版本表不存在时会先建表（`goose_db_version`），所以 `/readyz` 第一次检查一个从未迁移过的库时，会建出这张表并写入版本 0 的记录（goose 的初始状态）。它不影响之后的 `migrate up`，接受这个副作用。
- 不启用 goose 的会话锁：v0 是单实例部署，生产环境"先迁移、再启动"；以后需要多实例同时启动时再加。

### 2.6 测试数据库：`platform/postgres/pgtest`
```go
func NewDatabase(t testing.TB) string      // 执行过全部生产迁移的新库，返回连接 URL
func NewEmptyDatabase(t testing.TB) string // 没有任何迁移的新库，给自带迁移集的测试（如迁移执行器的测试）
```
- **容器**：测试二进制中第一次调用时，用 testcontainers 启动 `postgres:18.6`（与开发库同一个镜像），并创建模板库 `nerve_template`、执行生产迁移（`migrations.FS()`），然后关闭所有到模板库的连接。
- **每个测试一个库**：`CREATE DATABASE test_<n> TEMPLATE nerve_template`（`NewEmptyDatabase` 用 `template0`），`t.Cleanup` 中 `DROP DATABASE … WITH (FORCE)`，测试没有关闭的连接不会阻止删除。并行测试可以同时复制。
- **容器怎么共享**：Go 为每个包单独启动一个测试进程，所以"一次测试运行共用一个容器"落实为**每个包的测试进程共用一个容器**（`sync.OnceValues`）。不做跨进程复用（testcontainers 的按名复用），理由：跨进程复用需要处理并发创建容器和模板库的竞争，复杂度高；P2 只有 4 个包用到数据库，并行启动，本机完整跑一遍 `go test ./...` 不到 10 秒。模块变多、持续集成明显变慢时再评估（见风险）。
- **清理**：容器由 testcontainers 的回收容器（Ryuk）在测试进程退出后删除，不需要在每个包里写 `TestMain`。已验证测试结束后没有遗留容器。
- **`go test -short`**：`pgtest` 直接跳过测试，没有 Docker 时也能跑单元测试。`make test` 和持续集成不加 `-short`。
- **只能被测试导入**：archtest 规则 8。包级单例（`sync.OnceValues`）是"每个测试进程一个容器"本身的要求，是"禁止全局可变状态"规则在测试代码中的唯一例外。

### 2.7 HTTP：`platform/httpserver`

**路由**（`NewMux` 注册平台路由，返回的 `*http.ServeMux` 供 P3 的模块和 P5 的 `webui` 继续注册）：

| 路由 | 行为 |
|---|---|
| `GET /healthz` | 存活检查，不访问任何依赖：`200 {"status":"ok"}` |
| `GET /readyz` | 按顺序执行就绪检查，全部通过返回 `200 {"status":"ok"}`；第一个失败的检查返回 503 problem+json，后面的检查不再执行 |
| `/api/`（任意方法） | 兜底：返回 `404` problem+json，`detail` 为 `no API endpoint for <方法> <路径>` |
| 其余路径 | P2 不处理（ServeMux 的纯文本 404），P5 由 `webui` 接管 |

- **`/api/` 兜底放在 P2**：它是平台的路由规则，与具体模块无关（M0 设计 3.3 与平台路由一起列出），P2 可以直接测试。P3 的验收"`GET /api/v0/不存在的路径` 返回 404 problem+json"在挂上模块之后再验证一次。
- 因为兜底模式匹配任意方法，`/api/` 下"路径存在但方法不对"的请求（例如 `POST /api/v0/instance`）也会得到 404 problem+json，而不是 405。v0 接受这一点。

**就绪检查**：
```go
type Check struct {
	Name string
	Run  func(ctx context.Context) error
}
```
- 每次 `/readyz` 请求的全部检查共用一个 2 秒的超时（常量），依赖卡住时返回 503，而不是让探针一直等。
- `bootstrap` 注册两个检查：`database`（`pool.Ping`）和 `migrations`（`migrator.CheckUpToDate`，没有待执行的迁移）。
- 失败时的响应体：`{"status":503,"code":"not_ready","title":"Service Unavailable","detail":"database is not ready"}`。**具体错误只写进日志**（warn 级别，带检查名和请求 ID），不返回给客户端，避免在公开端口上泄露内部主机名、账号等信息。

**中间件**（顺序固定，由 `NewServer` 统一套上，调用方无法漏掉或调换）：
1. **请求 ID**：读取 `X-Request-Id`（常量 `HeaderRequestID`）；只有 1–128 个 `[A-Za-z0-9._:-]` 字符时才采用调用方的值（防止日志注入），否则用标准库 `uuid.NewV7()` 生成。写回响应头，并放进请求的 context（包内的 `requestID(ctx)` 读取，供后两个中间件和 `/readyz` 的日志使用）。
2. **异常恢复**：捕获 panic，记录 error 日志（请求 ID、方法、路径、panic 值、调用栈），返回 `500 {"status":500,"code":"internal_error","title":"Internal Server Error"}`。如果响应已经开始写出，就改为 `panic(http.ErrAbortHandler)` 中断连接，避免客户端把截断的响应当作完整响应。handler 自己抛出的 `http.ErrAbortHandler` 原样继续抛出，不记为异常。
3. **访问日志**：每个请求一条 info 日志 `http request`，字段 `request_id`、`method`、`path`、`status`、`duration`。handler panic 时记为 500（与异常恢复返回的状态一致）；handler 什么都没写时记为 200。

**problem+json**（与总体设计 3.5 一致，P3 的 `api/common.yaml` 照此定义 `Problem` 组件）：
```go
const ContentTypeProblem = "application/problem+json"

const (
	CodeNotFound = "not_found"
	CodeInternal = "internal_error"
	CodeNotReady = "not_ready"
)

type Problem struct {
	Status int          `json:"status"`
	Code   string       `json:"code"`
	Title  string       `json:"title"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func WriteProblem(w http.ResponseWriter, p Problem)
```
- `status`、`code`、`title`、`errors` 来自总体设计 3.5；`detail` 是 RFC 9457 的标准成员，用于说明这一次出错的具体情况。
- 不包含 `type`（缺省即 RFC 9457 的 `about:blank`）和 `instance`。总体设计 3.5 没有要求在响应体中带请求 ID，请求 ID 只在 `X-Request-Id` 响应头中。
- 平台自己的错误码不带模块前缀（`not_found`、`internal_error`、`not_ready`）；模块的错误码带模块前缀（`issue.state_not_in_project`），错误码体系在 M2 建立。`title` 取 `http.StatusText(status)`。

**生命周期**：
```go
type Server struct{ /* 未导出字段 */ }

func NewServer(cfg config.ServerConfig, h http.Handler, logger *slog.Logger) *Server
func (s *Server) ListenAndServe(ctx context.Context) error // 监听 server.addr，然后同 Serve
func (s *Server) Serve(ctx context.Context, ln net.Listener) error
```
- `Serve` 一直服务到 `ctx` 结束，然后停止接收新连接，最多等待 `server.shutdown_timeout` 让进行中的请求完成：按时完成返回 nil；超时则强制关闭剩余连接，返回包含 `context.DeadlineExceeded` 的错误。
- **可测试的接缝**：停机由 `ctx` 触发，不直接处理信号；测试传入自己的 listener 和可取消的 context。信号只在 `main` 中转换为 context（2.9）。
- `http.Server` 设置 `ReadHeaderTimeout`；`ErrorLog` 用 `slog.NewLogLogger` 接到同一个 logger（不导入标准库 `log`）。

### 2.8 组合根：`bootstrap`
对 `cmd/nerve` 只暴露四个函数，每个对应一个命令：
```go
func Serve(ctx context.Context, cfg config.Config, logOut io.Writer) error
func MigrateUp(ctx context.Context, cfg config.Config, out io.Writer) error
func MigrateDown(ctx context.Context, cfg config.Config, out io.Writer) error
func MigrateStatus(ctx context.Context, cfg config.Config, out io.Writer) error
```
- **`Serve`**：创建 logger（写到 `logOut`，即 stderr）→ 打印生效的配置（打码）→ 创建连接池和迁移执行器 → 注册就绪检查、创建路由（P3 起在这里挂模块）→ `database.auto_migrate=true` 时先执行迁移并逐条记录日志（迁移失败则退出）→ 监听并服务，直到 ctx 结束 → 按"迁移执行器、连接池"的顺序释放资源。
- 内部的 `app` 类型（未导出）负责接线和运行；`newApp` 额外接收迁移文件系统参数，测试用自己的迁移集替换生产迁移。
- **迁移命令**：各自创建连接池和迁移执行器，结果写到 `out`（stdout），用完释放。输出格式：

  | 命令 | 输出 |
  |---|---|
  | `migrate up` | 每个执行的迁移一行 `applied <文件名>`；没有待执行的：`no pending migrations` |
  | `migrate down` | `rolled back <文件名>`；没有可回滚的：`no applied migrations to roll back` |
  | `migrate status` | 没有迁移文件：`no migrations`；否则一张对齐的表，时间为 UTC RFC 3339，未执行的显示 `-` |

  `migrate status` 的表：
  ```
  VERSION  STATE    APPLIED AT            SOURCE
  1        applied  2026-09-22T10:00:00Z  00001_identity_create_users.sql
  2        pending  -                     00002_identity_create_sessions.sql
  ```
- 迁移命令不创建 logger、不打印配置：它们只有 stdout 上的结果和失败时的错误。

### 2.9 命令行：`cmd/nerve`
- `main.go`：`signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` 得到 ctx；第一个信号触发优雅停机后立刻恢复默认的信号处理（`context.AfterFunc(ctx, stop)`），所以再按一次 Ctrl-C 会立即结束进程。然后调用 `run` 并以它的返回值退出。
- `func run(ctx context.Context, args, environ []string, stdout, stderr io.Writer) int`：可测试的入口，出错时向 stderr 打印 `nerve: <错误>`，返回 1。
- `commands.go`：cobra 命令树 `serve`、`migrate up|down|status`、`version`。
  - 需要配置的命令用 `config.Load(config.Sources{Embedded: configs.FS(), Environ: environ, LocalFile: "configs/config.local.yaml"})` 加载配置，再调用 `bootstrap` 中对应的函数。每个命令的实现只有"加载配置 + 转交"两步。
  - `version` 不加载配置，输出一行：`nerve <版本> commit=<提交号> commit_time=<提交时间> modified=<true|false>`（来自 `buildinfo.Get()`）。
  - 关闭 cobra 默认的 `completion` 命令；错误和用法提示不重复打印（`SilenceErrors`、`SilenceUsage`，错误由 `run` 统一打印）。
  - 所有命令都不接受位置参数。

### 2.10 架构守护

**`internal/archtest`**（只有测试文件，随 `go test ./...` 运行）：
- 每条规则是一个纯函数 `func(from, to string) bool`，判断一条导入边是否违规。规则只看导入路径，不依赖真实代码，所以可以用人造的导入边逐条测试。
- **规则**（路径相对于模块 `github.com/open-nerve/NerveProject/server`）：

  | # | 规则 | 来源 |
  |---|---|---|
  | 1 | 模块内的依赖只能向内：`adapter → app → domain`（模块根包 `module.go` 在最外层） | M0 3.7 第 1 条 |
  | 2 | `domain` 只能导入标准库（不含 `net/http`、`database/sql` 及其子包）、本模块的包和 `internal/shared`；不能导入 `platform`、第三方库 | M0 3.7 第 1 条 |
  | 3 | 模块之间不能互相导入 | M0 3.7 第 2 条 |
  | 4 | `platform` 不能导入 `modules` 和 `bootstrap` | M0 3.7 第 3 条 |
  | 5 | 只有 `bootstrap` 能导入 `modules`（模块内部的导入由规则 1–3 管） | M0 3.7 第 4 条 |
  | 6 | `modules/<m>/adapter/http/gen` 只能被 `modules/<m>/adapter/http` 导入 | M0 3.7 第 5 条 |
  | 7 | `platform` 的各个包之间互不导入，`config` 除外（同一个包的子包不算，例如 `postgres/pgtest` → `postgres`） | M0 3.1 |
  | 8 | `pgtest` 只能被测试代码导入 | 新增，见第 3 节 |

- **规则测试**：一张"导入边 → 应违反的规则"表格，每条规则至少有一个违规的例子和一个合法的例子；测试还检查每条规则在表格中至少触发过一次，防止以后新增的规则没有测试。M0 还没有任何模块，规则 1、2、3、5、6 只能这样证明有效。
- **仓库测试**：用 `golang.org/x/tools/go/packages` 加载模块中所有非测试包（`NeedName | NeedImports`），对每条导入边应用全部规则，每个违规报一条错误，例如：
  `internal/modules/probe/domain imports net/http: domain imports only the standard library (not net/http or database/sql), its own module and internal/shared`。
  加载结果必须包含 `cmd/nerve`、`internal/bootstrap`、`internal/platform/config`，否则报错，防止"加载失败"被当成"没有违规"。
- **测试缓存问题（已核实并处理）**：`go/packages` 通过子进程 `go list` 读取源码，`go test` 的结果缓存看不到这些文件读取。已复现：先跑一次通过，再加一个违规导入，`go test` 直接返回缓存的通过结果。持续集成的 `setup-go` 会恢复构建缓存，同样会受影响。处理：测试在进程内先遍历一遍模块目录（`filepath.WalkDir`），把每个目录的文件列表（含文件大小和修改时间）登记为这个测试的输入；任何源码的增改都会让缓存失效。已验证修复后违规能被发现。
- **验收演示**：计划中故意新建 `internal/modules/probe/domain/probe.go` 导入 `net/http`，架构测试必须失败并报出上面的错误，然后删除这个目录。

**`.golangci.yml`**（golangci-lint v2 格式，`server/.golangci.yml`）：
- `linters.default: standard`（errcheck、govet、ineffassign、staticcheck、unused）+ `depguard`。
- **depguard 禁用列表**：`github.com/google/uuid`、`github.com/gofrs/uuid`、`github.com/satori/go.uuid`（用标准库 `uuid`）；`github.com/spf13/viper`（用 koanf）；`github.com/pkg/errors`（用标准库 `errors`）；`log$`（用 `log/slog`）。
- **已核实**：depguard 的 `pkg` 是前缀匹配，写 `log` 会同时禁掉 `log/slog` 和 `log/syslog`；末尾加 `$` 表示精确匹配，`log$` 只禁标准库 `log`。
- **格式检查**：`gofmt` 和 `goimports`（`local-prefixes: github.com/open-nerve/NerveProject`，导入分三组：标准库、第三方、本仓库）。代价很低，能避免格式差异进入评审。
- 不再启用其他 linter（如 `gochecknoinits`、`gochecknoglobals`）：standard 已覆盖常见错误；"禁止 `init()`、禁止全局可变状态"由评审把关，`gochecknoglobals` 对 `embed.FS`、`sync.OnceValues` 等合理用法会误报。
- **验收演示**：计划中临时加入一个导入 `log` 的文件，`make lint` 必须报 `import 'log' is not allowed from list 'banned': 用 log/slog (depguard)`，然后删除。

### 2.11 Makefile、持续集成、文档与交接
- **`make run`**：`cd server && NERVE_ENV=dev go run ./cmd/nerve serve`。显式设置 `NERVE_ENV=dev`，即使开发者的 shell 里导出了别的 `NERVE_ENV` 也不受影响。
- **`make test`**：命令不变（`go test ./...`），但从 P2 起需要 Docker（集成测试）。
- **持续集成：不改动**。`ubuntu-24.04` 运行环境自带 Docker；testcontainers 在第一次使用时拉取 `postgres:18.6`。本机完整运行 `go test ./...` 不到 10 秒（镜像已在本地），冷启动时拉取镜像多出十几秒，15 分钟的任务超时足够。
- **README 开发环境一节**：加入 `make run`（需先 `make dev-db`）；说明 `make test` 需要 Docker、`go test -short ./...` 可跳过集成测试；说明用 `NERVE_DEV_DB_PORT` 换了开发库端口时，还要用 `NERVE_DATABASE__URL` 或 `server/configs/config.local.yaml` 覆盖 `database.url`。
- **处理 P1 交接**（[P1-repo-toolchain-go-db-notes](../handoffs/P1-repo-toolchain-go-db-notes.md)，完成后改为 `status: done`）：
  1. 每次 `go get` / `go mod tidy` 后检查 `server/go.mod` 仍是 `go 1.27`：本 Phase 的全部依赖都已核实不会抬高这一行。
  2. `toolchain go1.27.1` 在 `server/go.mod` 和 `server/tools/go.mod` 中保持一致：本 Phase 不改动两处的 `toolchain` 行。
  3. `config.dev.yaml` 中写死端口 55432：README 和 `config.dev.yaml` 的注释都写明换端口时要同时覆盖 `database.url`。

## 3. 与上级设计的差异和补充（请控制者裁定）

| # | 上级设计 | P2 的做法 | 理由 |
|---|---|---|---|
| 1 | M0 3.6 的配置项包含 `app.name: Nerve` 和 `web.enabled: true` | P2 不加入这两项 | P2 没有使用者，按"不写用不上的代码"。`web.enabled` 由 P5（`webui`）加入；`app.name` 目前看不到用途（instance 返回的 `product` 是品牌常量），等第一个真正使用它的 Phase 再决定是否加入 |
| 2 | 总体设计 6.8 的加载顺序第 1 步是"代码中的默认值" | 由内置的 `config.yaml` 承担，不在 Go 代码中再写一份默认值 | `config.yaml` 本来就要"列出所有配置项及其默认值"，而且已经编进程序；两份默认值会不一致 |
| 3 | 总体设计 6.8："用 `NERVE_CONFIG_DIR` 指向外部目录，覆盖内置的配置文件" | 外部目录作为内置文件之上的**一层**，逐个键覆盖；目录里的文件可以只写需要改的键，缺失的文件跳过 | 如果整组替换，外部目录必须包含完整的 `config.yaml`，漏掉一个键就会出现空值；逐键覆盖时内置默认值始终存在 |
| 4 | M0 3.6 的 dev 数据库地址 `postgres://nerve:nerve@localhost:55432/nerve` | 加上 `?sslmode=disable` | 开发库没有开启 TLS；显式关闭，免得 pgx 每次先尝试 TLS 再回退 |
| 5 | M0 3.8：一个 Postgres 容器"同一次测试运行中所有测试共用" | 每个包的测试进程一个容器 | Go 为每个包单独启动测试进程；跨进程共用容器需要处理并发竞争，P2 的规模下不值得（见 2.6） |
| 6 | M0 3.8 只描述了"执行迁移后的模板库" | `pgtest` 另外提供 `NewEmptyDatabase` | 迁移执行器的测试需要一个没有执行过生产迁移的库；M2 有了生产迁移后，两者就不同了 |
| 7 | M0 8 中"`/api/` 下不存在的路径返回 404 problem+json"列在 P3 的验收中 | P2 在 `httpserver` 中实现并测试，P3 再验证一次 | 这是平台的路由规则，与模块无关 |
| 8 | M0 3.1：`cmd/nerve` 依赖 `bootstrap`、`platform/config`、`platform/buildinfo` | 另外导入 `server/configs`（只有内嵌数据） | 程序入口负责提供内嵌的配置文件，`platform/config` 本身不绑定具体文件，便于测试 |
| 9 | M0 3.7 列出 5 条架构规则 | 另加两条：`platform` 包之间互不导入（`config` 除外，来自 M0 3.1）；`pgtest` 只能被测试导入 | 前者是 M0 3.1 已有的约定，写进测试才能守住；后者防止 testcontainers 及 Docker 客户端被编进生产程序 |
| 10 | 总体设计 3.5 的错误示例没有 `detail` | `Problem` 增加可选的 `detail`（RFC 9457 标准成员） | 503、404 等平台错误需要一句具体说明；P3 的 `api/common.yaml` 同步定义 |
| 11 | — | 配置中的未知键直接报错；时长只接受字符串 | 拼写错误不会悄悄回落到默认值；避免 `20` 被当作 20 纳秒 |
| 12 | — | 调用方传入的 `X-Request-Id` 只有是安全字符时才采用 | 防止日志注入 |

## 4. 验收标准
1. **单元测试**（`go test ./...` 中）：
   - 配置：五层加载顺序和覆盖规则；环境变量映射（`__` 分级、单个 `_` 保留、控制变量和 `NERVE_DEV_DB_PORT` 被忽略）；dev 以外的环境不读 `config.local.yaml`；缺失的可选文件被跳过；`NERVE_ENV` 非法、`NERVE_CONFIG_DIR` 不存在、必填项缺失、未知的键、非法时长、数字时长、非法数字、非法 YAML 都报错并指明键名；七条取值校验一次全部列出；打码；三个内置环境都能加载并得到预期的值。
   - 日志：JSON 和文本格式、级别过滤、非法配置报错。
   - 中间件：请求 ID 的生成、采用、拒绝；panic 返回 500 problem+json 并记录日志；响应开始后 panic 会中断连接；`http.ErrAbortHandler` 不被当作异常；访问日志的字段和状态码。
   - problem+json：字段、`Content-Type`、可选成员省略。
   - 路由：`/healthz` 不执行检查；`/readyz` 全部通过为 200、第一个失败为 503 且记录日志、检查带超时；`/api/` 兜底 404 problem+json；其余路径不被接管。
   - 生命周期：取消 ctx 后正常停机返回 nil；停机开始后不再接受新连接，进行中的请求能完成；超过 `shutdown_timeout` 返回超时错误；监听失败报错。
   - 连接池：`max_conns` 生效；URL 解析失败时错误中不含密码。
   - 迁移执行器：没有迁移文件时所有操作都是空操作且不访问数据库；数据库不可用时 `CheckUpToDate` 返回连接错误。
   - 命令行：`version` 输出；未知命令、非法配置以退出码 1 结束并打印错误。
2. **集成测试**（testcontainers，随 `go test ./...` 运行）：
   - 迁移执行器：用测试迁移集依次验证 status（全部待执行）→ up → status（全部已执行）→ 再次 up 无事可做 → down 两次 → 第三次 down 无事可做；表结构随之变化；失败的迁移报错。
   - `pgtest`：库之间相互隔离；空库没有表；测试结束后库被删除。
   - `bootstrap`：自动迁移后 `/readyz` 为 200；有待执行的迁移时 503（`migrations is not ready`）；**数据库不可用时 503（`database is not ready`）**；自动迁移失败时启动失败；三个迁移命令的输出（有迁移和没有迁移两种情况）。
   - `cmd/nerve`：`nerve serve` 在 `/readyz` 就绪后，取消 ctx 以退出码 0 结束，日志中的密码已打码；`nerve migrate status` 输出 `no migrations`。
3. **架构测试通过**；故意加入一个违规导入后架构测试失败并指出违规，删除后恢复通过。
4. **`make lint` 输出 `0 issues.`**；故意导入标准库 `log` 后 depguard 报错，删除后恢复。
5. **手工验证**（开发库已由 `make dev-db` 启动）：
   - `make run` 后 `curl localhost:8080/healthz` 和 `/readyz` 都返回 200；`curl localhost:8080/api/v0/nope` 返回 404 problem+json。
   - 用一个连不上的 `NERVE_DATABASE__URL` 启动（`NERVE_DATABASE__AUTO_MIGRATE=false`），`/healthz` 为 200、`/readyz` 为 503 problem+json，启动日志中的密码已打码。
   - SIGINT 和 SIGTERM 都能让 `nerve serve` 正常停机（日志出现 `http server stopped`，退出码 0）。
   - `nerve migrate status` 输出 `no migrations`；`nerve version` 输出版本行。
6. `server/go.mod` 仍是 `go 1.27` 和 `toolchain go1.27.1`。
7. 持续集成通过。

## 5. 不在 P2 范围内
- `instance` 模块、OpenAPI、代码生成、`make gen` / `make gen-check`（P3）；持续集成的任务拆分（P3，见 P1 的 handoff）。
- `webui`、`web.enabled`、前端（P5）。
- 端到端测试（P6）。
- River、sqlc、`TxManager`、领域错误码体系（M2）。
- 任何迁移文件（由建表的 M 提供）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 模块增多后，每个用到数据库的包各启动一个容器，持续集成变慢 | P2 只有 4 个包，并行启动，本机总计不到 10 秒。M2 之后如果明显变慢，评估 testcontainers 的跨进程复用，或限制 `go test -p` |
| 持续集成拉取 `postgres:18.6` 受 Docker Hub 匿名拉取频率限制 | 目前每次运行只拉一次。出现限流时，在持续集成中登录 Docker Hub 或改用镜像缓存 |
| 设置了 `TESTCONTAINERS_RYUK_DISABLED=true` 的环境中，测试容器不会被自动删除 | 不设置这个变量（本机和持续集成都没有设置） |
| `/readyz` 在从未迁移过的库上会建出 `goose_db_version` 表 | goose 的行为，无害；之后的 `migrate up` 照常执行 |
| 用 `go/packages` 加载导入关系时，测试结果缓存可能掩盖新的违规 | 已通过遍历模块目录解决（2.10），并验证 |
| mapstructure 的解码错误格式（`decoding failed due to the following error(s)`）不够整齐 | 每个错误都带键名，满足"指出是哪个配置项"；不值得为它包一层转换 |
