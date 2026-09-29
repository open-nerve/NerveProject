# Nerve

一个轻量的多人协作项目管理系统，页面和接口能力完全对等：同一个账户，既可以由人在页面上使用，也可以由 Agent 通过接口使用。

- 后端：Go（单个可执行文件）+ PostgreSQL
- 前端：基于 [Plane](https://github.com/makeplane/plane) 前端分叉并裁剪
- 协议：[AGPL-3.0](LICENSE)

项目处于 v0 开发阶段，从 [docs/](docs/README.md) 开始阅读。

## 开发环境

需要安装：
- Docker（含 Compose v2；`make plane-schema` 需要 Compose 2.22 或更高）
- Go 1.26 或更高。第一次在 `server/` 下执行 Go 命令时，会自动下载 `server/go.mod` 指定的 Go 1.27.1（前提是 `GOTOOLCHAIN=auto`，这是 Go 官方安装包的默认值；部分 Linux 发行版自带的 Go 默认是 `local`，需要先执行 `go env -w GOTOOLCHAIN=auto`）
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）。代码生成，以及前端的检查、开发和构建需要它

第一次启动：

```bash
pnpm install  # 安装 Node 依赖（代码生成和前端检查要用）
make dev-db   # 启动本地 Postgres 18（端口 55432，可用 NERVE_DEV_DB_PORT 修改）
make test     # 运行测试（需要 Docker：集成测试用 testcontainers 启动 Postgres）
make run      # 以 dev 配置运行后端，监听 127.0.0.1:8080；Ctrl-C 停止
make          # 查看所有命令
```

- 只跑单元测试、跳过需要 Docker 的集成测试：在 `server/` 下执行 `go test -short ./...`。
- `make test` 不用 Go 的测试缓存：缓存不跟踪 `server/` 之外的文件，契约测试读取的 `api/` 改了也会重放旧结果。直接执行 `go test` 时，改了 `api/` 要加 `-count=1`。
- `server/tools` 是独立的 Go 模块（代码生成工具），`server/` 下的 `go test ./...` 不进入它；`make test` 和 `make lint-go` 都另外在它里面跑一遍。
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

## 接口与代码生成

接口先写描述（OpenAPI 3.1），再写实现：

- `api/openapi.yaml` 是入口，列出所有路径；各模块的描述在 `api/modules/<模块>.yaml`，公共组件在 `api/common.yaml`。
- 改完描述后执行 `make gen`，把生成的文件和描述一起提交：
  - `make gen-go`：只需要 Go。生成 Go 接口层（`server/internal/modules/<模块>/adapter/http/gen/server.gen.go`、`server/internal/platform/httpserver/apigen/`）；每个模块的请求体结构表（同一目录的 `bodyshape.gen.go`，由 `server/tools/bodyshapegen` 读同一份接口描述生成，服务端据此在 handler 之前拒绝不合契约的请求体）；以及 sqlc 按 `server/sqlc.yaml` 从 `server/internal/modules/<模块>/adapter/postgres/queries/*.sql` 生成的查询代码（`adapter/postgres/gen/`；sqlc 以 `CGO_ENABLED=0` 运行，不需要 C 编译器）。
  - `make gen-web`：打包好的 `api/dist/openapi.yaml`，以及 TS 客户端 `web/packages/api-client` 的类型，需要 Node（先执行 `pnpm install`）。
- 生成的文件不要手改。`make gen-check` 会重新生成一遍，检查生成物已经提交、没有差异；持续集成也执行这项检查。
- 每个操作都写明 `security`（需要令牌的写 `[{bearer: []}]`，公开的写 `[]`）和 `x-problem-codes`（它可能返回的错误码；所有操作都可能返回的写在 `api/openapi.yaml` 顶层）。`apitest` 的测试核对这些写法，模块的 handler 测试要把声明的每个错误码都返回一次。
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（关键词守卫，前端的类型检查、oxlint 警告数核对、格式检查、中英文翻译键一致性检查，以及 `tools/` 下脚本的 lint、`tools/` 和根目录配置文件的格式检查）。

## 前端

`web/apps/web` 和 `web/packages/*` 从 Plane 原样迁入，来源提交和之后的每一处改动见[前端改动清单](docs/v0/frontend-changes.md)；`web/packages/api-client` 是生成的 TS 客户端。前端任务由 turbo 按 `turbo.json` 编排，入口仍是 Makefile：

- **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。`make web-dev` 同时监视 `web/packages/*`：改了某个包的代码，这个包重新构建，页面随之更新。第一次启动时 Vite 要预构建依赖，页面如果报"Outdated Optimize Dep"，刷新一次即可。
- **单元测试**：`make test-web` 运行各包 `test` 脚本中的 vitest（经 turbo，持续集成的 `web` 任务也执行它）。只给以后仍然有效的稳定逻辑写小测试，测试文件 `*.test.ts` 放在被测代码旁边；包里还没有 `test` 脚本时，加上 `"test": "vitest run"` 和开发依赖 `vitest`（`catalog:`）。`make test` 只运行 Go 测试。
- **没有前端环境变量**：前端与接口同源部署，接口一律用相对路径（`/api/…`），代码不读取 `process.env` 或 `import.meta.env.VITE_*`，关键词守卫看住这一点。`import.meta.env.DEV`、`PROD` 是 Vite 按构建模式给出的常量，不是环境变量。
- **构建**：`make build` 构建前端，复制到 `server/internal/platform/webui/dist/`，编译出内嵌前端的 `bin/nerve`。运行时要在 `server/` 目录下，`config.local.yaml` 才能生效：`cd server && NERVE_ENV=dev ../bin/nerve serve`，之后打开 http://127.0.0.1:8080 。`dist/` 中只提交了 `.gitkeep`，没有构建过前端时页面上只有一句提示。
- **清掉旧的构建产物**：`make build` 之后，`make run` 和 `go test` 都会继续内嵌这份构建。要去掉它：`find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete`（与 Makefile 里 `make build` 自己的清理命令相同）。
- **登录状态**：会话由令牌管理器（`web/apps/web/core/lib/auth/`）保管，不用 Cookie：注册、登录得到的令牌交给它，续期和退出由它完成。访问令牌只在内存里；刷新令牌和 `login_id` 存在 localStorage 的 `nerve.auth` 中，同一个浏览器的标签页共用这一个会话，一次只有一个标签页在续期（`navigator.locks`，没有它时用 localStorage 的租约，见"部署"一节的"HTTPS"一条）。带令牌的请求得到 401 时，认证中间件（同一目录）续期一次、重发一次；只有续期得到 401、或者重发后仍是 401 才结束会话，页面回到登录页。续期超时（8 秒）、断网或得到 401 以外的错误（如 429、5xx）时会话保留：打开或刷新页面时的第一次续期这样失败，页面显示"会话暂不可用"的界面（"Cannot reach the server for now"），按 `Retry-After` 或退避自动重试，也可以点"Try again"；使用中的续期这样失败，需要它的请求失败，不自动重试，其中取账户的请求失败时也显示这个界面，但只能手动重试。会话每次变化（本标签页或另一个标签页登录、退出，会话结束），stores 都按新会话重建；属于旧会话的请求以 `SessionChangedError` 结束，不再发出，也不续期重发。
- **M2 中看到的页面**：登录、注册；新手引导（三步中只有资料一步能用，工作区、邀请两步要用 M3 的接口）；完成了新手引导的账户登录后的落点 `/create-workspace`（创建工作区在 M3）；个人设置的四个标签页（general、preferences、security、api-tokens）。工作区相关的页面和操作还在调用 Plane 的接口，Nerve 返回 404，从 M3 起对接。
- **lint 警告数等于上限**：每个包的 `check:lint` 脚本是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，它运行 oxlint，要求警告数正好等于上限，有任何错误都失败。警告多了，`make lint-web` 失败并列出这个包的全部警告：修掉新增的那几条，上限只能调低。警告少了（修掉了警告，或者删掉了带警告的代码），同样失败，并给出应调低到的数值：在同一个提交里把上限改成这个数。`make lint-web` 只打印失败任务的输出；要看某个包的全部警告，执行 `pnpm --filter <包名> exec oxlint .`。
- **关键词守卫**：`make lint-web` 的第一步是 `node tools/keywords.mjs`，规则在 `tools/keywords.json`（M1 设计 7.4）。它检查 git 列出的文件（包括还没 `git add` 的新文件），命中规则、又没有登记例外就失败，并列出规则、文件、行号和命中的原文；规则或文件读取有问题时以 2 退出。删掉一个功能时，在同一个提交里加上它的规则（每条规则带理由和命中、不命中的样本）。确实要保留的命中登记为例外：规则、文件、命中的原文、理由和到期的 M 或 Phase，一条例外只覆盖一处；例外不再命中任何内容时守卫会提醒删掉它。`tools/` 下的脚本本身也由 `make lint-web` 检查：oxlint 不允许警告（根目录 `package.json` 的 `check:lint`）；oxfmt 检查它们和根目录的工具链配置的格式，文件列表只写在根目录 `package.json` 的 `fix:format` 里，`check:format` 就是带 `--check` 运行它。
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包，以及根目录 `fix:format` 列出的文件。
- **多语言**：只有 `en` 和 `zh-CN`（`web/packages/i18n/src/locales/`）。两种语言的命名空间文件和键必须完全一致，同一个键不能出现在两个命名空间里，`make lint-web` 检查（i18n 包的 `check:sync`）；加、删文案时两种语言一起改。本地存储或用户资料中的其他语言按英文处理。
- **未使用的代码**：`make knip` 先生成 web 的路由类型，再用 knip 检查未使用的文件、导出和依赖，配置在 `knip.jsonc`。它是门禁（M1/P3 清零之后）：有任何未使用的代码，或配置本身过时，都会失败。持续集成的 `web` 任务执行它。

## 端到端测试

`e2e/` 是 Playwright 项目（工作区包 `@nerve/e2e`），一个用户故事一个测试文件，放在 `e2e/stories/` 下。被测对象是 `make build` 编译出的 `bin/nerve`（内嵌前端）加上 Postgres 18。

- **第一次运行之前**，安装 Playwright 用的 Chromium（下载到本机的缓存目录；升级 Playwright 之后再执行一次）：

  ```bash
  cd e2e && pnpm exec playwright install chromium
  ```

- **运行**：`make e2e`。它先执行 `make build`（没有改动时约 1 秒），再运行全部故事。需要 Docker：测试用 testcontainers 启动一个 Postgres 容器，运行结束后自动删除。
- **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，用 test 配置启动自己的 `nerve serve`：nerve 在 `127.0.0.1:0` 上监听，把实际地址写进地址文件（`server.addr_file`），fixture 读出地址，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库（每个 worker 一个连接池；断言函数按表放在 `e2e/fixtures/assert/`）。需要另一种配置的故事（例如关闭注册）用 `nerveWith` 在同一个库上另起一个 nerve。
- **页面的登录状态**：`signedInPage(tokens)` 给出一个已经登录、还没有加载任何东西的页面，`tokens` 来自 `login` 或 `register`（`nerveWith` 另起的 nerve 把它的地址作为第二个参数）。页面第一次加载时，在它的任何脚本运行之前，一条新的 `nerve.auth` 记录写进 localStorage；之后的加载、刷新和同一上下文里的其他标签页不再写，看到的是页面自己续期或删除之后的记录，和真实的浏览器一样。`e2e/fixtures/browser.ts` 的 `watchPage` 记录页面的接口请求、失败的接口请求、发往旧接口（`/api/v0` 之外）的请求、未处理的异常、控制台的错误和警告、CSP 违规，要在页面第一次打开之前调用；同一文件的 `expectQuietConsole` 先让页面写一条探针错误和一条探针警告、要求收到，再断言控制台没有别的错误和警告（参数点名允许的第三方警告，以及浏览器对 401 这类回答的报错）。
- **版本号**：`make build` 把 `VERSION`（默认 `0.1.0-dev`）写进 `bin/nerve`，例如 `make build VERSION=0.1.0`。`make e2e` 把同一个值放进环境变量 `NERVE_VERSION` 交给测试，S3 核对 `/api/v0/instance` 返回的版本号。
- **同源**：S2 断言页面的所有请求都发往 nerve 自身（见"前端"一节的"没有前端环境变量"一条）；有请求发往别处时 S2 失败，失败信息列出这些请求。S2 还断言：未登录的页面只请求 `GET /api/v0/instance`，没有失败的请求、未处理的异常和 CSP 违规，控制台没有错误和警告（深链接点名允许一条第三方的警告：编辑器加载时 `is-emoji-supported` 反复读画布，Chromium 提示 `willReadFrequently`）；首页带 `Content-Security-Policy`。S2 等 `GET /api/v0/instance` 的回答和登录页的"Go to workspace"按钮出现，不等 `networkidle`：页面不读回答的请求一直算在途中，`networkidle` 会等到测试超时。
- **只运行部分故事、打开浏览器调试**：先 `make build`，再直接运行 Playwright，`NERVE_VERSION` 要与构建时的 `VERSION` 相同：

  ```bash
  cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s2 --headed
  ```

- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）、截图和数据库快照（`database.sql`，本 worker 数据库的 `pg_dump`，也是报告中的附件）在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。持续集成的 `e2e` 任务失败时，把这两个目录上传为 `playwright-report`。

## 部署

- **设 `NERVE_ENV=prod`**。不设时 nerve 按 dev 的默认值运行：注册默认开放；没有配置签名密钥时用临时密钥，重启后已签发的访问令牌全部失效。nerve 不是 prod、却监听在本机回环地址之外时，启动日志会记一条 WARN 提醒。
- **签名密钥**：prod 必须提供 Ed25519 私钥（PKCS#8 PEM 文件），用 `auth.jwt.private_key_file`（环境变量 `NERVE_AUTH__JWT__PRIVATE_KEY_FILE`）指向它；没有提供或读不出来时 nerve 拒绝启动，错误指出这个配置项。生成：

  ```bash
  openssl genpkey -algorithm ed25519 -out nerve-jwt.pem
  chmod 600 nerve-jwt.pem
  ```

  访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
- **反向代理**：nerve 前面有反向代理（例如 Caddy）时，把代理的地址写进 `server.trusted_proxies`（CIDR 列表；环境变量用逗号分隔，例如 `NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8`）。只有连接的对端在这个列表中时，nerve 才从 `X-Forwarded-For` 自右向左取第一个不可信的地址作为客户端 IP。不配置时，所有请求都算作代理的地址：按 IP 的限流（匿名请求、登录、注册、认证失败）让所有人共用一份额度；nerve 第一次收到不可信对端带来的 `X-Forwarded-For` 时记一条 WARN 提醒。代理必须往 `X-Forwarded-For` 里写不带端口的 IP 地址：某一项带端口或是主机名时，nerve 在转发它的那个代理处停下，这个代理后面的客户端都算作代理的地址（第一次遇到时同样记一条 WARN）。只信任你自己的代理的地址：`0.0.0.0/0`、`::/0` 这样信任所有地址的前缀让任何客户端都能自己选 IP，启动时被拒绝。IPv6 客户端按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
- **HTTPS**：放在公网上的部署用 HTTPS（例如在前面加 Caddy，按上一条配置 `server.trusted_proxies`）。直接用 HTTP 部署也可以，例如在局域网里打开 `http://192.168.1.20:8080`：这时浏览器不在安全上下文中，没有 `navigator.locks`，同一个浏览器的多个标签页改用 localStorage 的租约轮流续期。租约不是原子的：极少数情况下两个标签页同时续期，或者持有租约的标签页被浏览器冻结超过 10 秒、另一个标签页接着续期，都可能触发刷新令牌的重复使用检测，用户要重新登录（M2 设计 7.1、§16）。
- **CSP**：nerve 给每个页面（HTML 响应）设 `Content-Security-Policy`：内联脚本按 `index.html` 里各自的 SHA-256 放行，请求只能发往本站（`connect-src 'self'`）。前端和接口必须同源；反向代理不要改写或另加这个响应头。
- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create --email <邮箱>` 创建（见下一条），注册关闭时也能用。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在；存储的哈希都用当前的 argon2 参数时，耗时也相同。调高 `auth.password.argon2_memory_kib`、`argon2_iterations` 之后，还没有重新登录过的账户能从耗时上与不存在的邮箱区分开，直到它们各登录一次，所以这两个参数少调（M2 设计 §16）。关闭注册时，已注册和未注册的邮箱得到同一个 403。
- **管理命令**：`nerve users` 下的五个命令直接连数据库执行，用与 `nerve serve` 相同的配置（`NERVE_ENV`、`NERVE_DATABASE__URL` 等），服务不用停。都用 `--email` 指定账户，邮箱按注册时的规则规范化（去掉首尾空白、转小写）。账户不存在、邮箱已被使用、密码不合规时，退出码为 1，打印一行说明，数据库不变。需要密码的 `create`、`reset-password` 在终端上不回显地提示输入两次；标准输入不是终端时读一行，供脚本使用（例如 `printf '%s\n' "$PASSWORD" | nerve users create --email ada@example.com`）。
  - `create --email <邮箱>`：建账户，不建会话。
  - `reset-password --email <邮箱>`：设新密码，结束该账户的全部会话，撤销全部个人访问令牌（PAT）。
  - `set-email --email <旧邮箱> --new-email <新邮箱>`：修改登录邮箱（邮箱只能这样改），结束全部会话，**不撤销 PAT**。
  - `deactivate --email <邮箱>`：停用账户，与用户自己停用相同：结束全部会话，重置新手引导，不改密码；PAT 保留，但停用期间认证失败。
  - `activate --email <邮箱>`：恢复账户，没有过期的 PAT **重新可用**，输出它们的个数。

  `set-email` 和 `activate` 都不是账户被盗后的恢复手段：怀疑账户被盗时，另外执行 `reset-password`。
- **建工作区**：`workspace.creation_enabled`（默认 `true`）决定用户能否经接口建工作区；关闭时 `POST /api/v0/workspaces` 答 403 `workspace.creation_disabled`，前端隐藏入口。这时由服务器管理员建：`nerve workspaces create --slug <slug> --name <名称> --admin-email <邮箱>`，不受这个开关限制，建出的工作区与接口建的相同，这个邮箱的账户是它的管理员和唯一成员。它与 `nerve users` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。slug 已被占用或是保留名、名称不合规、邮箱没有账户、账户已停用时，退出码为 1，打印一行说明，数据库不变。
- **令牌泄露后的恢复**：刷新令牌存在浏览器的 localStorage 里，页面上的 XSS 能读出它，换来访问令牌后创建一个永不过期的 PAT（创建 PAT 不要求输入密码）。这个 PAT 不受退出、修改密码和会话 30 天期限的影响，还能再创建 PAT，所以泄露的影响不以 30 天为限（M2 设计 8.5）。怀疑泄露时，按这个顺序做：
  1. **先退出、重新登录（或在另一个浏览器登录），在新的会话里修改密码**（个人设置的 security 页）：修改密码结束除当前会话以外的全部会话。被盗的刷新令牌多半就是这个浏览器会话的（XSS 读的是本页的 localStorage），在它里面改密码不会结束它；退出也不一定结束它（对方续期之后，本页的刷新令牌已是旧的一代，退出不改变什么）。从新会话改密码之后，被盗的会话和由它换来的访问令牌在下一个请求就失效，对方不能再用会话创建 PAT。
  2. **再撤销不认识的 PAT**：在 api-tokens 页或 security 页的列表中（或 `GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，全部撤销，拿不准的也撤销；然后重新打开列表核对。PAT 能创建 PAT：撤销的同时，对方可能用还没撤销的 PAT 建了新的，列表里不再出现不认识的令牌才算完成。

  只撤销 PAT、不先改密码是不够的：对方还握着会话，随时能再建一个 PAT。对方不断建新令牌、撤销跟不上，或者用户已无法登录时，请服务器管理员执行 `nerve users reset-password --email <邮箱>`：它在一个事务里设新密码、结束该账户的全部会话、撤销全部 PAT。
- **令牌的密钥扫描**：个人访问令牌以 `nrv_pat_` 开头，刷新令牌以 `nrv_rt_` 开头，但前缀不会让代码托管平台自动识别它们。要让平台发现提交里泄露的令牌，在它的密钥扫描中加自定义规则，例如 GitHub 仓库或组织设置的 Secret scanning → Custom patterns（需要平台提供这项功能）：个人访问令牌 `nrv_pat_[A-Za-z0-9_-]{43}`，刷新令牌 `nrv_rt_[A-Za-z0-9_-]{91}`。
- **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。
  - **迁移和服务分用两个数据库角色时**（表的所有者执行迁移，服务用另一个角色），服务的角色需要的权限全部写在 [`deploy/runtime-grants.sql`](deploy/runtime-grants.sql)：业务表的读写，River 的表和序列，`river_job` 的 `MAINTAIN`（River 每天 00:00 UTC 用 `REINDEX INDEX CONCURRENTLY` 重建它的索引，需要 PostgreSQL 17 起），`goose_db_version` 的读（`/readyz` 靠它判断迁移是否已完成）。按表给读写（DML），不给 `TRUNCATE`、`REFERENCES`、`TRIGGER` 和 DDL。函数和类型不在文件里：PostgreSQL 默认让 PUBLIC 执行函数、使用类型，River 的 `river_job_state_in_bitmask` 和 `river_job_state` 靠的就是它；迁移收回了 PUBLIC 的这些权限时，把它们加进文件。文件授权给组角色 `nerve_runtime`：先建一次组角色，服务登录用的角色加入它，例如 `CREATE ROLE nerve_runtime NOLOGIN;`、`CREATE ROLE nerve_app LOGIN PASSWORD '…' IN ROLE nerve_runtime;`。
  - **每次 `nerve migrate up` 之后**，以表的所有者执行一次这个文件，例如 `psql -v ON_ERROR_STOP=1 -f deploy/runtime-grants.sql`：新的迁移可能加了表，文件逐个列出表和序列；重复执行没有影响。
  - 少了 River 的权限时 `/readyz` 仍是 200，River 的任务（会话清理）和索引重建因 `permission denied`（42501）失败，只记在日志里；少了业务表的权限时接口请求失败；少了 `goose_db_version` 的读时 `/readyz` 是 503。`server/internal/bootstrap/runtime_role_test.go` 用恰好这些权限的角色运行 nerve（就绪、会话清理、索引重建），`public` 里有表、视图、序列或函数的权限与文件不符时失败。
- **停机**：收到 SIGTERM 或 SIGINT 后，nerve 依次停下：先是 HTTP 优雅停机，不再接受新连接，处理中的请求最多等 `server.shutdown_timeout`（默认 20 秒）；然后是后台任务（River），不再取新任务，正在运行的任务最多再等 `jobs.shutdown_timeout`（默认 10 秒），到期取消它们，再等 1 秒；最后关闭连接池，最多等 5 秒。前两步到期时 nerve 记 ERROR 并以退出码 1 退出，连接池到期只记 WARN。默认配置下最坏约 36 秒；实测空闲的 nerve 只要几到二十几毫秒。容器或进程管理器的停止宽限期要比 36 秒长：Docker 默认只等 10 秒（Compose 用 `stop_grace_period` 调大），Kubernetes 默认 30 秒（`terminationGracePeriodSeconds`）；也可以调小这几个期限。宽限期不够时，nerve 在收尾中被 SIGKILL（再收到一次 SIGTERM 或 SIGINT 时也立即退出）：处理中的请求被切断，正在运行的任务留在 `running`，River 在它开始运行 1 小时后才把它当作卡住的任务重新排入重试。
- **停机时 River 的日志**：nerve 启动后的最初几秒内停机时，River 可能记几条 ERROR：它的维护服务还在错开启动时是 `maintenance.PeriodicJobEnqueuer: Error starting transaction`（`context canceled`）；nerve 还没记 `jobs started` 时，还可能有 `notifier.Notifier: Error running listener (will attempt reconnect after backoff)` 等几条 `conn closed`。这是 River 停止时自己记的，不是故障：退出码为 0，没有任务停在 `running`，没有连接泄漏；这次没投递的会话清理，下次启动时投递。运行中的 nerve 停机，只有恰好落在清理任务投递的那几毫秒（每 `auth.session_cleanup_interval` 一次）才会出现第一条。River 每天 00:00 UTC 用 `REINDEX INDEX CONCURRENTLY` 重建 `river_job` 的索引；停机打断重建时，River 先删掉没建完的 `*_ccnew` 索引再停下，这次删除最多等 15 秒。有访问过 `river_job` 的长事务挡住删除、超过 `jobs.shutdown_timeout` 加 1 秒时，nerve 记 ERROR（`jobs still running …`）并以退出码 1 退出；挡满 15 秒时索引留下，River 此后跳过这个索引，每次重建都记 WARN `maintenance.Reindexer: Found reindex artifact … skipping reindex`：用 `DROP INDEX CONCURRENTLY` 删掉这条 WARN 的 `artifact_names` 中列出的索引。迁移和服务分用两个角色时，索引属于表的所有者，服务的角色删不掉它（River 停机时的那次删除也会因此失败，索引留下）：由表的所有者执行这条删除。

## Plane 表结构快照

`tools/plane-schema/plane-v1.4.2-schema.sql` 是 Plane v1.4.2 的完整表结构，各个 M 为自己的模块建表时以它为起点，再按[差异清单](docs/v0/plane-diff.md)修改。它是生成物，不要手改；只有升级 Plane 基线时才需要用 `make plane-schema` 重新生成（需要 Docker），说明见 [tools/plane-schema/README.md](tools/plane-schema/README.md)。

## 版权

Copyright © 2026 OpenNerve。以 [GNU AGPL-3.0](LICENSE) 协议发布。

前端的部分代码来自 [Plane](https://github.com/makeplane/plane)（Copyright © Plane Software, Inc. and contributors，AGPL-3.0），相关文件保留了原有的版权声明。`web/apps` 和 `web/packages` 里与 Plane 的文件放在一起的 Nerve 文件带 `Copyright (c) 2026-present OpenNerve` 和 `SPDX-License-Identifier: AGPL-3.0-only`；放不下文件头的资源（例如 PNG）登记在 `web/apps/web/app/assets/brand/SOURCES.md` 里。`web/packages/api-client` 和 `web/` 以外 Nerve 自己的代码不加文件头，以本仓库的 [LICENSE](LICENSE) 为准。

服务端的常见密码名单 `server/internal/modules/identity/domain/common_passwords.txt` 由 `tools/password-blocklist/build.mjs` 从英国国家网络安全中心（NCSC）发布的 `PwnedPasswordsTop100k.txt`（泄露最多的前 10 万个密码，数据来自 Have I Been Pwned 的 Pwned Passwords）过滤生成。NCSC 的原地址已失效，SecLists 收录的是同一个文件（SHA-256 相同）：[`Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt`](https://raw.githubusercontent.com/danielmiessler/SecLists/master/Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt)。Contains public sector information licensed under the [Open Government Licence v3.0](https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/). Pwned Passwords 本身没有许可和署名要求。
