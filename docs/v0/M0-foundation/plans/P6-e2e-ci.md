# M0/P6 端到端测试骨架与持续集成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `make build` 注入版本号；新建 Playwright 工作区包 `@nerve/e2e`：fixtures（testcontainers 启动的 Postgres 18.6、从模板库复制的每个 worker 的库、每个 worker 一个 `bin/nerve serve`、生成的 TS 客户端）和 M0 的四个冒烟故事 S1–S4；`make e2e`；knip 的配置和 `make knip`（M0 只出报告）；持续集成新增 `e2e` 任务，`web` 任务输出 knip 报告；README 和交给 P6 的交接收尾。

**Architecture:** 被测对象是真正要发布的产物：`make build` 编译出的 `bin/nerve`（内嵌前端）加一个 Postgres。Playwright 的全局准备（`e2e/global-setup.ts`）每次运行启动一个 Postgres 容器，用 `bin/nerve migrate up` 迁移模板库；每个 worker 的 fixture 用 `CREATE DATABASE … TEMPLATE` 复制出自己的库，在空闲端口上用 test 配置启动自己的 `nerve serve`，轮询 `/readyz` 直到就绪，结束时发送 SIGTERM 并检查退出码。`fixtures/` 下的 `db.ts`、`server.ts`、`api.ts` 只是普通函数，`fixtures/test.ts` 把它们接成 Playwright 的 fixture（`db`、`nerve`、`baseURL`、`api`），故事只从它导入 `test`。`make e2e` 依赖 `make build`，并把构建时注入的 `VERSION` 交给测试，S3 据此核对版本号。

**Tech Stack:** Node 24、pnpm 11.10.0、TypeScript 5.8.3、@playwright/test 1.63.0（Chromium 153，playwright chromium v1243）、@testcontainers/postgresql 12.1.0（testcontainers-node 12.1.0，Ryuk 0.14.0）、pg 8.23.0、@types/pg 8.23.1、knip 6.37.0、oxlint 1.51.0、oxfmt 0.35.0、turbo 2.10.11；Go 1.27.1；Postgres 镜像 `postgres:18.6`；GitHub Actions `actions/upload-artifact@v7`。

**Spec:** `docs/v0/M0-foundation/specs/P6-e2e-ci.md`（上级：`docs/v0/M0-foundation/M0-design.md`）

## Global Constraints

- 本计划基于 P5 合并后的 `main`（`376f662`）。开始之前执行一次 `corepack enable` 和 `pnpm install --frozen-lockfile`。
- Node 依赖一律写精确版本，通过 pnpm 工作区安装，不做任何全局安装：`@playwright/test` `1.63.0`、`@testcontainers/postgresql` `12.1.0`、`pg` `8.23.0`、`@types/pg` `8.23.1`（`e2e/package.json`）；`knip` `6.37.0`（根 `package.json`）；`typescript`、`@types/node` 用 `catalog:`。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 由步骤中的 `pnpm install` 生成。每次生成都给出预期的行数和 SHA-256（`shasum -a 256`），以及一条核对命令；对不上时停下来，先确认输入的 `package.json`、`pnpm-workspace.yaml` 与本计划一致。
- **Docker 只通过 testcontainers 使用**（它给自己的容器加标签，由回收容器 Ryuk 清理）。不设置 `TESTCONTAINERS_RYUK_DISABLED`；不要停止、删除、进入或改动任何不是本计划创建的容器，不执行任何 prune 命令。
- **Playwright 的浏览器**下载到本机的缓存目录（macOS 是 `~/Library/Caches/ms-playwright`）。在本机执行安装命令时加 `--no-remove`：不删除本机其他 Playwright 安装使用的浏览器。
- 本 Phase 不改动 Go 代码和 `server/go.mod`。`make test`、`make lint-go` 的结果与 P5 相同（Task 5 核对）。
- TS 代码必须通过 `make lint-web` 中 e2e 的三项检查：`tsc --noEmit`、`oxlint --max-warnings=0`、`oxfmt --check`。`e2e/package.json` 必须有这三个脚本，否则 turbo 悄悄跳过这个包。
- **Playwright 不经 turbo 运行**：Makefile 直接在 `e2e/` 中执行 `pnpm exec playwright test`（spec 2.7：turbo 的严格环境变量模式会去掉 `CI`、`DOCKER_HOST`、`TESTCONTAINERS_*`、`PLAYWRIGHT_*`、`NERVE_VERSION`）。`e2e/package.json` 不写 `test` 脚本。
- **构建不依赖 `web/apps/web/.env`**：开始之前确认没有这个文件（`ls web/apps/web/.env` 报 No such file or directory）；有的话先移走。只有 Task 3 Step 10 临时创建它，并在同一步中删掉。
- 代码注释用英文；配置文件、YAML、Makefile 中的注释用中文。测试名用英文，以故事编号开头（`S1: …`）。
- 所有代码块都是完整的文件内容（"修改"步骤除外，它给出"把……替换为……"的原文），照原样写入，不要改动。**Makefile 的命令行以 Tab 开头**：写入后用 `grep -c "$(printf '\t')" Makefile` 核对（各 Task 给出预期值）。Makefile 必须兼容 macOS 自带的 GNU Make 3.81。
- 端到端测试启动的 `nerve serve` 由 fixture 停止。每次运行之后 `pgrep -fl 'bin/nerve serve'` 应当没有输出；testcontainers 的容器在运行结束后约 10 秒内消失（`docker ps --filter label=org.testcontainers=true` 没有输出）。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明（`cd e2e && …`、`cd server && …` 只在该行内切换目录）。

## 文件结构

路径相对于仓库根目录。

| 文件 | 职责 | Task |
|---|---|---|
| `Makefile`（修改） | `VERSION` 与版本号注入（Task 1）；`e2e`（Task 3）；`knip`（Task 4） | 1、3、4 |
| `e2e/package.json`、`e2e/tsconfig.json`、`e2e/.gitignore` | 工作区包 `@nerve/e2e`：依赖、类型检查、lint、格式检查；忽略 Playwright 的输出 | 2 |
| `pnpm-workspace.yaml`（修改） | `allowBuilds` 拒绝 testcontainers 间接依赖的三个安装脚本 | 2 |
| `pnpm-lock.yaml`（生成） | 锁文件 | 2、4 |
| `e2e/playwright.config.ts`、`e2e/global-setup.ts` | Playwright 配置；每次运行启动 Postgres、迁移模板库 | 2 |
| `e2e/fixtures/db.ts` | Postgres 容器、复制数据库、查询 | 2 |
| `e2e/fixtures/server.ts` | 运行 `bin/nerve` 的命令，启动和停止 `nerve serve` | 2 |
| `e2e/fixtures/api.ts` | 生成的 TS 客户端 | 2 |
| `e2e/fixtures/test.ts` | Playwright 的 fixture：`db`、`nerve`、`baseURL`、`api` | 2 |
| `e2e/stories/smoke/s1-server-ready.spec.ts` | S1 | 2 |
| `e2e/stories/smoke/s2-web-app.spec.ts`、`s3-instance-info.spec.ts`、`s4-unknown-api.spec.ts` | S2、S3、S4 | 3 |
| `README.md`（修改） | "端到端测试"一节（Task 3）；knip（Task 4） | 3、4 |
| `package.json`（修改）、`knip.jsonc` | knip 的依赖和配置 | 4 |
| `.github/workflows/ci.yml`（修改） | `web` 任务输出 knip 报告；新增 `e2e` 任务 | 4 |
| `docs/v0/M0-foundation/handoffs/P2-server-platform-p6-e2e-notes.md`、`P3-api-contract-p6-notes.md`、`P5-web-import-p6-notes.md`（修改） | 交接事项改为 done | 5 |

---

### Task 1: `make build` 注入版本号

**Files:**
- Modify: `Makefile`

**Interfaces:**
- Consumes: `server/internal/platform/buildinfo` 的包变量 `version`（默认 `0.1.0-dev`，注释中写明用 `-ldflags -X` 覆盖）；P5 的 `make build`
- Produces:
  - Make 变量 `VERSION ?= 0.1.0-dev`：`make build VERSION=<版本>`，或环境变量 `VERSION`
  - `make build` 用 `-ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=$(VERSION)"` 编译 `bin/nerve`；Task 3 的 `make e2e` 把同一个 `VERSION` 交给 S3

- [ ] **Step 1: 修改 `Makefile`（`VERSION` 和链接参数）**

把：

```makefile
# make build 把前端的构建产物复制到这里，由 go:embed 编进 nerve
WEBUI_DIST := server/internal/platform/webui/dist
```

替换为：

```makefile
# make build 把前端的构建产物复制到这里，由 go:embed 编进 nerve
WEBUI_DIST := server/internal/platform/webui/dist
# nerve 的版本号：make build 把它写进 bin/nerve，端到端测试 S3 核对它。
# 默认值与 server/internal/platform/buildinfo 中的相同；发布时指定，例如 make build VERSION=0.1.0
VERSION ?= 0.1.0-dev
GO_LDFLAGS := -X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=$(VERSION)
```

- [ ] **Step 2: 修改 `Makefile`（`make build` 带上链接参数）**

把：

```makefile
	cd server && go build -o ../bin/nerve ./cmd/nerve
```

替换为：

```makefile
	cd server && go build -ldflags "$(GO_LDFLAGS)" -o ../bin/nerve ./cmd/nerve
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected（以 Tab 开头的行和命令数都不变）:

```
30
20
```

- [ ] **Step 3: 确认版本号写进了 `bin/nerve`**

Run: `make build VERSION=9.9.9-p6 2>&1 | tail -1 && bin/nerve version`
Expected（第一次构建前端约 20–30 秒；`commit` 是当前的提交号，工作区有未提交的改动时 `modified=true`）:

```
cd server && go build -ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=9.9.9-p6" -o ../bin/nerve ./cmd/nerve
nerve 9.9.9-p6 commit=<当前提交> commit_time=<提交时间> modified=true
```

Run: `make build 2>&1 | tail -1 && bin/nerve version`
Expected（不指定时是默认值；前端命中 turbo 的缓存，整个命令约 1 秒）:

```
cd server && go build -ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=0.1.0-dev" -o ../bin/nerve ./cmd/nerve
nerve 0.1.0-dev commit=<当前提交> commit_time=<提交时间> modified=true
```

`GET /api/v0/instance` 返回的 `version` 由 Task 3 的 S3 核对。

- [ ] **Step 4: 提交**

```bash
git add Makefile
git commit -m "build: stamp VERSION into bin/nerve with make build

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

Run: `git status --short`
Expected: 没有输出（`bin/` 和 `webui/dist/` 中复制进来的文件都被 `.gitignore` 挡掉）。

---

### Task 2: 工作区包 `@nerve/e2e`、fixtures 和 S1

**Files:**
- Create: `e2e/package.json`、`e2e/tsconfig.json`、`e2e/.gitignore`、`e2e/playwright.config.ts`、`e2e/global-setup.ts`
- Create: `e2e/fixtures/db.ts`、`e2e/fixtures/server.ts`、`e2e/fixtures/api.ts`、`e2e/fixtures/test.ts`
- Create: `e2e/stories/smoke/s1-server-ready.spec.ts`
- Modify: `pnpm-workspace.yaml`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 1 的 `make build`（`bin/nerve`）；`@nerve/api-client` 的 `createClient`；`nerve serve`、`nerve migrate up|status` 和 test 配置（`NERVE_ENV=test`，`NERVE_DATABASE__URL`、`NERVE_SERVER__ADDR` 覆盖配置项）
- Produces:
  - `fixtures/db.ts`：`startPostgres(): Promise<{ stop }>`（全局准备调用；把服务器地址放进环境变量 `NERVE_E2E_POSTGRES_URL` 交给 worker）；`createDatabase(name, template?): Promise<Database>`；`templateDatabase = "nerve_template"`；`interface Database { name; url; query<Row>(sql, params?) }`
  - `fixtures/server.ts`：`runNerve(args, databaseUrl)`（执行一条 nerve 命令，退出码非 0 时 reject）；`startNerve(databaseUrl, logFile): Promise<Nerve>`；`interface Nerve { baseURL; stop() }`；`applicationName = "nerve"`
  - `fixtures/api.ts`：`createApi(baseURL): Api`，`type Api`
  - `fixtures/test.ts`：`test`（worker 级的 `db`、`nerve`；测试级的 `baseURL`、`api`）和 `expect`。Task 3 的故事从这里导入
  - `make lint-web` 多出 `@nerve/e2e` 的 `check:types`、`check:lint`、`check:format` 三个任务

- [ ] **Step 1: 写入 `e2e/package.json`**

```json
{
  "name": "@nerve/e2e",
  "private": true,
  "description": "End-to-end user stories: Playwright against the built bin/nerve and PostgreSQL",
  "license": "AGPL-3.0-only",
  "type": "module",
  "scripts": {
    "check:types": "tsc --noEmit",
    "check:lint": "oxlint --max-warnings=0 .",
    "check:format": "oxfmt --check ."
  },
  "devDependencies": {
    "@nerve/api-client": "workspace:*",
    "@playwright/test": "1.63.0",
    "@testcontainers/postgresql": "12.1.0",
    "@types/node": "catalog:",
    "@types/pg": "8.23.1",
    "pg": "8.23.0",
    "typescript": "catalog:"
  }
}
```

- [ ] **Step 2: 修改 `pnpm-workspace.yaml`（拒绝三个安装脚本）**

testcontainers 经 dockerode 间接依赖 `cpu-features`、`protobufjs`、`ssh2`，它们带安装脚本。pnpm 11 遇到没有登记在 `allowBuilds` 中的安装脚本时，`pnpm install` 报 `ERR_PNPM_IGNORED_BUILDS` 并以退出码 1 结束。

把：

```yaml
allowBuilds:
  "@swc/core": true
  esbuild: true
  turbo: true
```

替换为：

```yaml
allowBuilds:
  "@swc/core": true
  esbuild: true
  turbo: true
  # 以下三项不来自 Plane，是端到端测试的 testcontainers 经 dockerode 间接依赖的包。它们的安装脚本
  # 编译通过 SSH 连接 Docker 时用的可选扩展（cpu-features、ssh2），或只检查版本号（protobufjs）；用不到，不运行
  cpu-features: false
  protobufjs: false
  ssh2: false
```

- [ ] **Step 3: 安装，生成锁文件**

Run: `pnpm install 2>&1 | grep -E '^(Packages|Done|\[ERR)'`
Expected（没有 `[ERR_PNPM_…]`）:

```
Packages: +129 -19
Done in 3.2s using pnpm v11.10.0
```

（`-19` 是 node_modules 中被替换的包的旧写法，见下面的说明。）

Run: `wc -l < pnpm-lock.yaml && shasum -a 256 pnpm-lock.yaml`
Expected:

```
   14777
8691dff83cbd89d35d6cff4abb3a66fb47073d6a99d23c492e284192ea623fe0  pnpm-lock.yaml
```

核对锁文件：原有的包一个都没有少，web 下 13 个包的依赖只多了对等依赖的后缀 `(supports-color@10.2.2)`。在 bash 或 zsh 中执行：

```bash
pkgs() { sed -n '/^packages:/,/^snapshots:/p' | grep -E "^  '?[@a-z]" | sed -E "s/^  '?//; s/'?:\$//" | sort; }
importers() { sed -n '/^importers:/,/^packages:/p' | sed 's/(supports-color@10\.2\.2)//g' | awk '/^  [^ ]/{keep = ($1 ~ /^web\//)} keep'; }
comm -23 <(git show HEAD:pnpm-lock.yaml | pkgs) <(pkgs < pnpm-lock.yaml) | wc -l
comm -13 <(git show HEAD:pnpm-lock.yaml | pkgs) <(pkgs < pnpm-lock.yaml) | wc -l
diff <(git show HEAD:pnpm-lock.yaml | importers) <(importers < pnpm-lock.yaml) && echo identical
```

Expected（依次为：少了的包、新增的包、web 下的 importers）:

```
       0
     109
identical
```

说明：`debug` 有一个可选的对等依赖 `supports-color`。`pnpm install` 重新解析锁文件时，把它解析为图中已有的 `supports-color@10.2.2`（P3 加入的 openapi-typescript 引入），依赖 `debug` 的包的键因此多了这个后缀；包的版本都不变（spec 2.4）。

Run: `pnpm install --frozen-lockfile 2>&1 | tail -1 && git status --short pnpm-lock.yaml pnpm-workspace.yaml`
Expected（锁文件与依赖一致；第二次安装不再改动任何文件）:

```
Done in 161ms using pnpm v11.10.0
 M pnpm-lock.yaml
 M pnpm-workspace.yaml
```

（两个文件与上一个提交相比有改动；这次安装没有再改动它们：再执行一次 `wc -l < pnpm-lock.yaml && shasum -a 256 pnpm-lock.yaml`，结果与上面相同。）

- [ ] **Step 4: 写入 `e2e/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022"],
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

- [ ] **Step 5: 写入 `e2e/.gitignore`**

Playwright 的输出目录。放在包内而不是根目录的 `.gitignore`：`oxfmt` 只读取当前目录下的 `.gitignore`，`check:format` 在 `e2e/` 中运行，这样一个文件同时挡住 git 和格式检查。

```gitignore
playwright-report/
test-results/
```

- [ ] **Step 6: 写入 `e2e/fixtures/db.ts`**

```ts
import { PostgreSqlContainer } from "@testcontainers/postgresql";
import { Client, escapeIdentifier, type QueryResultRow } from "pg";

/** The PostgreSQL image, the same as the development database (deploy/compose.dev.yaml). */
const image = "postgres:18.6";

/** The database global setup migrates once; every worker gets a copy of it. */
export const templateDatabase = "nerve_template";

/** Global setup hands the server's URL to the workers in this variable. */
const serverUrlVariable = "NERVE_E2E_POSTGRES_URL";

/** A database of this run: nerve serves from it, stories assert on it. */
export interface Database {
  readonly name: string;
  readonly url: string;
  /** Runs one statement on this database and returns the rows. */
  query<Row extends QueryResultRow>(sql: string, params?: unknown[]): Promise<Row[]>;
}

/**
 * Starts the PostgreSQL server of this run and publishes its URL to the
 * workers. The testcontainers reaper (Ryuk) removes the container if the run
 * dies before stop is called.
 */
export async function startPostgres(): Promise<{ stop: () => Promise<void> }> {
  const container = await new PostgreSqlContainer(image).withDatabase("postgres").start();
  process.env[serverUrlVariable] = databaseUrl(container.getConnectionUri(), "postgres");
  return {
    stop: async () => {
      await container.stop();
    },
  };
}

/** Creates database name, as a copy of template when given, on the server startPostgres started. */
export async function createDatabase(name: string, template?: string): Promise<Database> {
  const serverUrl = process.env[serverUrlVariable];
  if (!serverUrl) {
    throw new Error(`${serverUrlVariable} is not set: run the stories with playwright test (see global-setup.ts)`);
  }
  const copy = template ? ` TEMPLATE ${escapeIdentifier(template)}` : "";
  await query(serverUrl, `CREATE DATABASE ${escapeIdentifier(name)}${copy}`);
  const url = databaseUrl(serverUrl, name);
  return { name, url, query: (sql, params) => query(url, sql, params) };
}

async function query<Row extends QueryResultRow>(url: string, sql: string, params?: unknown[]): Promise<Row[]> {
  const client = new Client({ connectionString: url });
  await client.connect();
  try {
    return (await client.query<Row>(sql, params)).rows;
  } finally {
    await client.end();
  }
}

function databaseUrl(serverUrl: string, name: string): string {
  const url = new URL(serverUrl);
  url.pathname = `/${name}`;
  url.searchParams.set("sslmode", "disable");
  return url.toString();
}
```

- [ ] **Step 7: 写入 `e2e/fixtures/server.ts`**

```ts
import { execFile, spawn, type ChildProcess } from "node:child_process";
import { once } from "node:events";
import { closeSync, mkdirSync, openSync } from "node:fs";
import { createServer, type AddressInfo } from "node:net";
import path from "node:path";
import { setTimeout as sleep } from "node:timers/promises";
import { promisify } from "node:util";

/** The binary under test: make build compiles it with the web frontend embedded. */
const binary = path.resolve(import.meta.dirname, "../../bin/nerve");

/** nerve connects with this application_name, so its sessions show in pg_stat_activity. */
export const applicationName = "nerve";

const readyTimeoutMs = 30_000;
const stopTimeoutMs = 30_000;

/** A nerve serve process of this run. */
export interface Nerve {
  readonly baseURL: string;
  /** Sends SIGTERM and waits for nerve to exit with code 0. */
  stop(): Promise<void>;
}

/**
 * Runs a nerve command, such as migrate up, with the test configuration on
 * the database at databaseUrl. It rejects when the command exits non-zero.
 */
export async function runNerve(args: string[], databaseUrl: string): Promise<{ stdout: string; stderr: string }> {
  return promisify(execFile)(binary, args, { env: nerveEnv(databaseUrl) });
}

/**
 * Starts nerve serve with the test configuration on a free local port and
 * waits until /readyz answers 200. Its output goes to logFile.
 */
export async function startNerve(databaseUrl: string, logFile: string): Promise<Nerve> {
  const addr = `127.0.0.1:${await freePort()}`;
  mkdirSync(path.dirname(logFile), { recursive: true });
  const log = openSync(logFile, "w");
  const child = spawn(binary, ["serve"], {
    env: { ...nerveEnv(databaseUrl), NERVE_SERVER__ADDR: addr },
    stdio: ["ignore", log, log],
  });
  closeSync(log); // the child has its own copy
  const baseURL = `http://${addr}`;
  try {
    await waitUntilReady(child, `${baseURL}/readyz`, Date.now() + readyTimeoutMs);
  } catch (err) {
    child.kill("SIGKILL");
    throw new Error(`nerve did not become ready (log: ${logFile})`, { cause: err });
  }
  return { baseURL, stop: () => stop(child, logFile) };
}

/** The test configuration on the given database; the caller's own NERVE_* variables are left out. */
function nerveEnv(databaseUrl: string): NodeJS.ProcessEnv {
  const url = new URL(databaseUrl);
  url.searchParams.set("application_name", applicationName);
  const inherited = Object.entries(process.env).filter(([name]) => !name.startsWith("NERVE_"));
  return { ...Object.fromEntries(inherited), NERVE_ENV: "test", NERVE_DATABASE__URL: url.toString() };
}

async function freePort(): Promise<number> {
  const server = createServer().listen(0, "127.0.0.1");
  await once(server, "listening");
  const { port } = server.address() as AddressInfo;
  server.close();
  await once(server, "close");
  return port;
}

/** Polls /readyz until it answers 200; fails when nerve exits or the deadline passes. */
async function waitUntilReady(child: ChildProcess, readyzUrl: string, deadline: number): Promise<void> {
  if (child.exitCode !== null) {
    throw new Error(`nerve exited with code ${child.exitCode}`);
  }
  const ready = await fetch(readyzUrl).then(
    (res) => res.ok,
    () => false // not listening yet
  );
  if (ready) {
    return;
  }
  if (Date.now() >= deadline) {
    throw new Error(`${readyzUrl} did not answer 200 within ${readyTimeoutMs} ms`);
  }
  await sleep(100);
  return waitUntilReady(child, readyzUrl, deadline);
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

- [ ] **Step 8: 写入 `e2e/fixtures/api.ts`**

```ts
import { createClient } from "@nerve/api-client";

/** The typed Nerve API client, generated from api/dist/openapi.yaml. */
export type Api = ReturnType<typeof createClient>;

/** Returns a client for the nerve at baseURL. */
export function createApi(baseURL: string): Api {
  return createClient({ baseUrl: baseURL });
}
```

- [ ] **Step 9: 写入 `e2e/fixtures/test.ts`**

```ts
import path from "node:path";

import { test as base } from "@playwright/test";

import { createApi, type Api } from "./api";
import { createDatabase, templateDatabase, type Database } from "./db";
import { startNerve, type Nerve } from "./server";

export { expect } from "@playwright/test";

interface WorkerFixtures {
  /** The worker's own database, a copy of the migrated template. */
  db: Database;
  /** The worker's own nerve serve, on that database. */
  nerve: Nerve;
}

interface TestFixtures {
  /** The typed API client for the worker's nerve. */
  api: Api;
}

/** Stories import test from here: every worker runs its own nerve on its own database. */
export const test = base.extend<TestFixtures, WorkerFixtures>({
  db: [
    // oxlint-disable-next-line no-empty-pattern -- Playwright reads a fixture's dependencies from this pattern
    async ({}, use, workerInfo) => {
      await use(await createDatabase(`e2e_w${workerInfo.workerIndex}`, templateDatabase));
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
    { scope: "worker" },
  ],
  // page and request resolve relative URLs against the worker's nerve.
  baseURL: async ({ nerve }, use) => {
    await use(nerve.baseURL);
  },
  api: async ({ nerve }, use) => {
    await use(createApi(nerve.baseURL));
  },
});
```

- [ ] **Step 10: 写入 `e2e/global-setup.ts`**

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
  const template = await createDatabase(templateDatabase);
  await runNerve(["migrate", "up"], template.url);
  return postgres.stop;
}
```

- [ ] **Step 11: 写入 `e2e/playwright.config.ts`**

```ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "stories",
  // Starts PostgreSQL and migrates the template database once per run.
  globalSetup: "./global-setup.ts",
  forbidOnly: !!process.env.CI,
  // The html report keeps the traces and screenshots of failed tests; CI uploads it.
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
```

- [ ] **Step 12: 写入 `e2e/stories/smoke/s1-server-ready.spec.ts`**

```ts
import { applicationName, runNerve } from "../../fixtures/server";
import { expect, test } from "../../fixtures/test";

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

  // M0 has no migration files; from M2 on this lists them as applied.
  const { stdout } = await runNerve(["migrate", "status"], db.url);
  expect(stdout).toBe("no migrations\n");
});
```

- [ ] **Step 13: 类型检查、lint、格式检查**

Run: `cd e2e && pnpm exec tsc --noEmit && pnpm exec oxlint --max-warnings=0 . && pnpm exec oxfmt --check .`
Expected（`tsc` 没有输出；`playwright.config.ts` 被 `.oxlintrc.json` 的 `*.config.{js,mjs,cjs,ts}` 排除，所以 oxlint 检查 6 个文件）:

```
Found 0 warnings and 0 errors.
Finished in 11ms on 6 files with 93 rules using 18 threads.
Checking formatting...

All matched files use the correct format.
Finished in 87ms on 9 files using 18 threads.
```

- [ ] **Step 14: 运行 S1**

S1 不打开浏览器，不需要先安装 Playwright 的浏览器（Task 3 安装）。

Run: `make build 2>&1 | tail -1 && cd e2e && pnpm exec playwright test`
Expected（本机镜像已拉取时整个运行约 2–3 秒；第一次运行要先拉取 `postgres:18.6` 和 `testcontainers/ryuk:0.14.0`）:

```
cd server && go build -ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=0.1.0-dev" -o ../bin/nerve ./cmd/nerve

Running 1 test using 1 worker

  ✓  1 [chromium] › stories/smoke/s1-server-ready.spec.ts:4:1 › S1: an operator starts nerve with the test configuration and it becomes ready (33ms)

  1 passed (2.7s)
```

Run: `ls e2e/test-results && pgrep -fl 'bin/nerve serve'; sleep 15; docker ps --filter label=org.testcontainers=true --format '{{.Image}}'`
Expected（worker 的 nerve 日志；test 配置的日志级别是 warn，正常运行时是空文件。之后两条命令都没有输出：nerve 已停止，容器已删除）:

```
nerve-w0.log
```

- [ ] **Step 15: 确认 S1 能发现 nerve 连错了数据库**

临时让 worker 的 nerve 连接 `postgres` 库，而不是本 worker 的库：

```bash
sed -i.orig 's|startNerve(db.url, logFile)|startNerve(db.url.replace(db.name, "postgres"), logFile)|' e2e/fixtures/test.ts
cd e2e && pnpm exec playwright test; cd ..
mv e2e/fixtures/test.ts.orig e2e/fixtures/test.ts
```

Expected（`/healthz`、`/readyz` 仍然通过，`pg_stat_activity` 的断言失败）:

```
  ✘  1 [chromium] › stories/smoke/s1-server-ready.spec.ts:4:1 › S1: an operator starts nerve with the test configuration and it becomes ready (29ms)
…
    Error: expect(received).toBeGreaterThan(expected)

    Expected: > 0
    Received:   0
…
    > 18 |   expect(sessions?.count).toBeGreaterThan(0);
…
  1 failed
```

恢复后 `git status --short e2e/fixtures` 只列出 `?? e2e/fixtures/`（整个目录尚未提交），`e2e/fixtures/test.ts.orig` 不存在。

- [ ] **Step 16: `make lint-web` 包含 e2e**

Run: `make lint-web 2>&1 | tail -4`
Expected（原来的 46 个任务加上 e2e 的 3 个）:

```
 Tasks:    49 successful, 49 total
Cached:    10 cached, 49 total
  Time:    22.117s 

```

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:types check:lint check:format --dry=json | grep -o '"taskId": "@nerve/e2e#[^"]*"'`
Expected:

```
"taskId": "@nerve/e2e#check:format"
"taskId": "@nerve/e2e#check:lint"
"taskId": "@nerve/e2e#check:types"
```

- [ ] **Step 17: 核对改动范围**

Run: `git status --short --untracked-files=all`
Expected:

```
 M pnpm-lock.yaml
 M pnpm-workspace.yaml
?? e2e/.gitignore
?? e2e/fixtures/api.ts
?? e2e/fixtures/db.ts
?? e2e/fixtures/server.ts
?? e2e/fixtures/test.ts
?? e2e/global-setup.ts
?? e2e/package.json
?? e2e/playwright.config.ts
?? e2e/stories/smoke/s1-server-ready.spec.ts
?? e2e/tsconfig.json
```

`e2e/test-results/`、`e2e/playwright-report/`、`e2e/node_modules/` 都没有出现（前两个被 `e2e/.gitignore` 挡掉，最后一个被根目录的 `node_modules/` 挡掉）。

- [ ] **Step 18: 提交**

```bash
git add e2e pnpm-workspace.yaml pnpm-lock.yaml
git commit -m "test(e2e): add the Playwright package, per-worker fixtures and story S1

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 冒烟故事 S2–S4 与 `make e2e`

**Files:**
- Create: `e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/smoke/s3-instance-info.spec.ts`、`e2e/stories/smoke/s4-unknown-api.spec.ts`
- Modify: `Makefile`、`README.md`

**Interfaces:**
- Consumes: Task 2 的 `test`、`expect`（`fixtures/test.ts`）；Task 1 的 `VERSION`
- Produces:
  - `make e2e`：先 `make build`，再在 `e2e/` 中直接运行 `pnpm exec playwright test`（不经过 turbo，spec 2.7），环境变量 `NERVE_VERSION=$(VERSION)`；Task 4 的持续集成调用它
  - S2 另外断言页面只向 nerve 自己发请求（同源部署），本地的 `web/apps/web/.env` 让构建不再同源时 S2 失败
  - S3 从 `NERVE_VERSION` 读取期望的版本号，没有设置时失败并说明原因

- [ ] **Step 1: 写入 `e2e/stories/smoke/s2-web-app.spec.ts`**

```ts
import type { Page, Response } from "@playwright/test";

import { expect, test } from "../../fixtures/test";

/** A page of the frontend's router, not a file: nerve answers it with index.html. */
const deepLink = "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues";

/**
 * Static resources are everything outside /api/ (M0 design 3.3). API calls are
 * left out: in M0 the Plane frontend still calls Plane's API, such as
 * GET /api/instances/, and nerve answers those with 404.
 */
function isStatic(url: string): boolean {
  return !new URL(url).pathname.startsWith("/api/");
}

interface Visit {
  document: Response;
  /** Static resources that loaded, as "<status> <url>". */
  loaded: string[];
  /** Static resources that failed: an HTTP error, or refused or aborted by the browser. */
  failed: string[];
  /** Requests to any origin other than nerve's; the frontend is served same-origin. */
  elsewhere: string[];
}

/** Opens path and waits until the network is idle. */
async function open(page: Page, path: string): Promise<Visit> {
  const requested: string[] = [];
  const loaded: string[] = [];
  const failed: string[] = [];
  page.on("request", (req) => {
    requested.push(req.url());
  });
  page.on("response", (res) => {
    if (isStatic(res.url())) {
      (res.status() < 400 ? loaded : failed).push(`${res.status()} ${res.url()}`);
    }
  });
  page.on("requestfailed", (req) => {
    if (isStatic(req.url())) {
      failed.push(`${req.failure()?.errorText} ${req.url()}`);
    }
  });
  const document = await page.goto(path, { waitUntil: "networkidle" });
  if (!document) {
    throw new Error(`no document response for ${path}`);
  }
  const origin = new URL(document.url()).origin;
  const elsewhere = requested.filter((url) => new URL(url).origin !== origin);
  return { document, loaded, failed, elsewhere };
}

test("S2: a user opens the home page in a browser", async ({ page }) => {
  const { document, loaded, failed, elsewhere } = await open(page, "/");

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(loaded).toContainEqual(expect.stringMatching(/^200 .*\/assets\/[^/]+\.js$/));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
});

test("S2: a user opens a deep link directly", async ({ page, request }) => {
  const { document, failed, elsewhere } = await open(page, deepLink);

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(await document.body()).toEqual(await (await request.get("/")).body());
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
});
```

- [ ] **Step 2: 写入 `e2e/stories/smoke/s3-instance-info.spec.ts`**

```ts
import { expect, test } from "../../fixtures/test";

test("S3: a caller reads the instance information with the typed client", async ({ api }) => {
  const version = process.env.NERVE_VERSION;
  expect(version, "NERVE_VERSION, the version make build stamped into bin/nerve (make e2e sets it)").toBeTruthy();

  const { data, error, response } = await api.GET("/api/v0/instance");

  expect(response.status).toBe(200);
  expect(error).toBeUndefined();
  expect(data).toEqual({
    product: "Nerve",
    version,
    commit: expect.stringMatching(/^[0-9a-f]{40}$/),
    api_version: "v0",
  });
});
```

- [ ] **Step 3: 写入 `e2e/stories/smoke/s4-unknown-api.spec.ts`**

```ts
import { expect, test } from "../../fixtures/test";

// The typed client cannot express a path the API does not have, so the
// request goes out directly.
test("S4: a caller requests an API path that does not exist", async ({ request }) => {
  const res = await request.get("/api/v0/nope");

  expect(res.status()).toBe(404);
  expect(res.headers()["content-type"]).toBe("application/problem+json");
  expect(await res.json()).toEqual({
    status: 404,
    code: "not_found",
    title: "Not Found",
    detail: "no API endpoint for GET /api/v0/nope",
  });
});
```

- [ ] **Step 4: 修改 `Makefile`（新增 `e2e`）**

把：

```makefile
.PHONY: build-web
build-web: ## 构建前端，产物在 web/apps/web/build/client（需要 Node；持续集成 web 任务）
	$(TURBO) run build --filter=web $(TURBO_QUIET)
```

替换为：

```makefile
.PHONY: build-web
build-web: ## 构建前端，产物在 web/apps/web/build/client（需要 Node；持续集成 web 任务）
	$(TURBO) run build --filter=web $(TURBO_QUIET)

.PHONY: e2e
e2e: build ## 构建 bin/nerve 并运行端到端测试（需要 Node、Go、Docker 和 Playwright 的浏览器，见 README）
	cd e2e && NERVE_VERSION=$(VERSION) pnpm exec playwright test
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected:

```
31
21
```

- [ ] **Step 5: 安装 Playwright 用的 Chromium**

Run: `cd e2e && pnpm exec playwright install --no-remove chromium`
Expected（已经装过的部分不再下载，全部装过时没有输出；本机下载一个浏览器约十几秒）:

```
Downloading Chrome for Testing 153.0.8010.12 (playwright chromium v1243) from https://cdn.playwright.dev/builds/cft/153.0.8010.12/mac-arm64/chrome-mac-arm64.zip
…
Chrome for Testing 153.0.8010.12 (playwright chromium v1243) downloaded to /Users/<用户>/Library/Caches/ms-playwright/chromium-1243
Downloading Chrome Headless Shell 153.0.8010.12 (playwright chromium-headless-shell v1243) from https://cdn.playwright.dev/builds/cft/153.0.8010.12/mac-arm64/chrome-headless-shell-mac-arm64.zip
…
Chrome Headless Shell 153.0.8010.12 (playwright chromium-headless-shell v1243) downloaded to /Users/<用户>/Library/Caches/ms-playwright/chromium_headless_shell-1243
```

测试以无界面方式运行，用的是 Headless Shell；完整的 Chrome for Testing 用于 `--headed` 和 `--ui` 调试。

- [ ] **Step 6: `make e2e`**

Run: `make e2e 2>&1 | tail -10`
Expected（4 个测试文件分给 4 个 worker，每个 worker 有自己的库和 nerve；本机全部约 5 秒）:

```

Running 5 tests using 4 workers

  ✓  1 [chromium] › stories/smoke/s3-instance-info.spec.ts:3:1 › S3: a caller reads the instance information with the typed client (6ms)
  ✓  2 [chromium] › stories/smoke/s4-unknown-api.spec.ts:5:1 › S4: a caller requests an API path that does not exist (15ms)
  ✓  3 [chromium] › stories/smoke/s1-server-ready.spec.ts:4:1 › S1: an operator starts nerve with the test configuration and it becomes ready (35ms)
  ✓  4 [chromium] › stories/smoke/s2-web-app.spec.ts:54:1 › S2: a user opens the home page in a browser (836ms)
  ✓  5 [chromium] › stories/smoke/s2-web-app.spec.ts:64:1 › S2: a user opens a deep link directly (868ms)

  5 passed (5.2s)
```

（编号的先后随 worker 完成的顺序变化。）

- [ ] **Step 7: 注入别的版本号，S3 仍然通过**

Run: `make e2e VERSION=0.0.0-p6 2>&1 | grep -E 'go build|passed|failed'`
Expected（`/api/v0/instance` 返回的是注入的版本号）:

```
cd server && go build -ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=0.0.0-p6" -o ../bin/nerve ./cmd/nerve
  5 passed (4.7s)
```

- [ ] **Step 8: 确认 S3 能发现版本号不一致**

此时 `bin/nerve` 的版本是 `0.0.0-p6`，让测试期望另一个值：

Run: `cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s3 2>&1 | grep -E '^\s+[-+] |failed'`
Expected:

```
    - Expected  - 1
    + Received  + 1
    -   "version": "0.1.0-dev",
    +   "version": "0.0.0-p6",
  1 failed
```

Run: `cd e2e && pnpm exec playwright test s3 2>&1 | grep -E 'Error: NERVE_VERSION|failed'`
Expected（没有设置 `NERVE_VERSION`）:

```
    Error: NERVE_VERSION, the version make build stamped into bin/nerve (make e2e sets it)
  1 failed
```

- [ ] **Step 9: 确认 S2 能发现加载失败的静态资源**

删掉首页引用的一个样式表，只重新编译 Go 程序（`make build` 会重新复制它），再运行 S2：

```bash
CSS=$(grep -o 'assets/globals-[A-Za-z0-9_-]*\.css' server/internal/platform/webui/dist/index.html | head -1)
rm "server/internal/platform/webui/dist/$CSS"
cd server && go build -o ../bin/nerve ./cmd/nerve; cd ..
cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s2 2>&1 | grep -E 'ERR_|✘|^  [0-9]+ failed'; cd ..
make build 2>&1 | tail -1
```

Expected（两个测试都失败，失败的请求是这个样式表：服务端返回 404 `text/plain`，浏览器按 `X-Content-Type-Options: nosniff` 拒绝它，报 `net::ERR_ABORTED`；最后的 `make build` 把它复制回来，版本号恢复为默认值）:

```
  ✘  1 [chromium] › stories/smoke/s2-web-app.spec.ts:54:1 › S2: a user opens the home page in a browser (786ms)
  ✘  2 [chromium] › stories/smoke/s2-web-app.spec.ts:64:1 › S2: a user opens a deep link directly (959ms)
    +   "net::ERR_ABORTED http://127.0.0.1:<端口>/assets/globals-<哈希>.css",
    +   "net::ERR_ABORTED http://127.0.0.1:<端口>/assets/globals-<哈希>.css",
    +   "net::ERR_ABORTED http://127.0.0.1:<端口>/assets/globals-<哈希>.css",
    +   "net::ERR_ABORTED http://127.0.0.1:<端口>/assets/globals-<哈希>.css",
  2 failed
cd server && go build -ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=0.1.0-dev" -o ../bin/nerve ./cmd/nerve
```

（每个测试中这个样式表各被请求两次：`<link rel="stylesheet">` 和预加载。）

- [ ] **Step 10: 确认 S2 能发现不是同源部署的构建**

Plane 的 `vite.config.ts` 用 dotenv 读取 `web/apps/web/.env`（不进仓库）。从 Plane 的 `.env.example` 复制出这个文件后，构建出的前端把接口请求发到 `http://localhost:8000`，而不是 nerve 自己：

```bash
cp web/apps/web/.env.example web/apps/web/.env
make build 2>&1 | grep -E 'Cached:'
cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s2 2>&1 | grep -E 'localhost:8000|✘|^  [0-9]+ failed'; cd ..
rm web/apps/web/.env
make build 2>&1 | grep -E 'Cached:'
```

Expected（`.env` 是 turbo 构建任务的输入，web 重新构建；两个测试都因为 `elsewhere` 失败；删掉 `.env` 之后，turbo 从缓存恢复原来的构建）:

```
Cached:    10 cached, 11 total
  ✘  1 [chromium] › stories/smoke/s2-web-app.spec.ts:54:1 › S2: a user opens the home page in a browser (776ms)
  ✘  2 [chromium] › stories/smoke/s2-web-app.spec.ts:64:1 › S2: a user opens a deep link directly (935ms)
    +   "http://localhost:8000/api/instances/",
    +   "http://localhost:8000/api/instances/",
  2 failed
Cached:    11 cached, 11 total
```

- [ ] **Step 11: 确认 S4 能发现"返回了前端页面"**

临时把 S4 请求的路径改到 `/api/` 之外，nerve 返回前端页面：

```bash
sed -i.orig 's|request.get("/api/v0/nope")|request.get("/v0/nope")|' e2e/stories/smoke/s4-unknown-api.spec.ts
cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s4 2>&1 | grep -E 'Expected:|Received:|failed'; cd ..
mv e2e/stories/smoke/s4-unknown-api.spec.ts.orig e2e/stories/smoke/s4-unknown-api.spec.ts
```

Expected:

```
    Expected: 404
    Received: 200
  1 failed
```

恢复后 `git status --short e2e/stories` 只列出三个新的故事文件，没有 `.orig`。

- [ ] **Step 12: 类型检查、lint、格式检查**

Run: `cd e2e && pnpm exec tsc --noEmit && pnpm exec oxlint --max-warnings=0 . && pnpm exec oxfmt --check .`
Expected:

```
Found 0 warnings and 0 errors.
Finished in 6ms on 9 files with 93 rules using 18 threads.
Checking formatting...

All matched files use the correct format.
Finished in 74ms on 12 files using 18 threads.
```

确认 S3 的调用确实经过类型检查：

Run: `sed -i.orig 's|api.GET("/api/v0/instance")|api.GET("/api/v0/instances")|' e2e/stories/smoke/s3-instance-info.spec.ts; cd e2e && pnpm exec tsc --noEmit; cd ..; mv e2e/stories/smoke/s3-instance-info.spec.ts.orig e2e/stories/smoke/s3-instance-info.spec.ts`
Expected:

```
stories/smoke/s3-instance-info.spec.ts(7,51): error TS2345: Argument of type '"/api/v0/instances"' is not assignable to parameter of type '"/api/v0/instance"'.
```

- [ ] **Step 13: 修改 `README.md`（新增"端到端测试"一节）**

把：

````markdown
## Plane 表结构快照
````

替换为：

````markdown
## 端到端测试

`e2e/` 是 Playwright 项目（工作区包 `@nerve/e2e`），一个用户故事一个测试文件，放在 `e2e/stories/` 下。被测对象是 `make build` 编译出的 `bin/nerve`（内嵌前端）加上 Postgres 18。

- **第一次运行之前**，安装 Playwright 用的 Chromium（下载到本机的缓存目录；升级 Playwright 之后再执行一次）：

  ```bash
  cd e2e && pnpm exec playwright install chromium
  ```

- **运行**：`make e2e`。它先执行 `make build`（没有改动时约 1 秒），再运行全部故事。需要 Docker：测试用 testcontainers 启动一个 Postgres 容器，运行结束后自动删除。
- **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，在一个空闲的本机端口上用 test 配置启动自己的 `nerve serve`，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库。
- **版本号**：`make build` 把 `VERSION`（默认 `0.1.0-dev`）写进 `bin/nerve`，例如 `make build VERSION=0.1.0`。`make e2e` 把同一个值放进环境变量 `NERVE_VERSION` 交给测试，S3 核对 `/api/v0/instance` 返回的版本号。
- **同源**：S2 断言页面的所有请求都发往 nerve 自身。构建时有 `web/apps/web/.env`（见"前端"一节的"不要建立"一条），S2 失败，失败信息列出发往别处的请求。
- **只运行部分故事、打开浏览器调试**：先 `make build`，再直接运行 Playwright，`NERVE_VERSION` 要与构建时的 `VERSION` 相同：

  ```bash
  cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s2 --headed
  ```

- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）和截图在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。

## Plane 表结构快照
````

- [ ] **Step 14: 核对改动范围**

Run: `git status --short --untracked-files=all`
Expected:

```
 M Makefile
 M README.md
?? e2e/stories/smoke/s2-web-app.spec.ts
?? e2e/stories/smoke/s3-instance-info.spec.ts
?? e2e/stories/smoke/s4-unknown-api.spec.ts
```

Run: `pgrep -fl 'bin/nerve serve'`
Expected: 没有输出。

- [ ] **Step 15: 提交**

```bash
git add Makefile README.md e2e/stories
git commit -m "test(e2e): add smoke stories S2 to S4 and make e2e

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: knip 与持续集成的 `e2e` 任务

**Files:**
- Create: `knip.jsonc`
- Modify: `package.json`、`Makefile`、`.github/workflows/ci.yml`、`README.md`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 3 的 `make e2e`；P5 的 `web` 任务（pnpm 存储的缓存）
- Produces:
  - `make knip`：`knip --no-exit-code`，只出报告；发现未使用的代码时退出码为 0，knip 自身出错（例如配置有误）时退出码非 0。M1 去掉 `--no-exit-code`，改为门禁
  - 持续集成：`web` 任务在 `make lint-web` 之后执行 `make knip`；新增 `e2e` 任务（`needs: [server, web]`）

- [ ] **Step 1: 写入 `package.json`（完整内容；加入开发依赖 `knip`）**

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
    "@redocly/cli": "2.53.3",
    "knip": "6.37.0",
    "oxfmt": "catalog:",
    "oxlint": "catalog:",
    "turbo": "catalog:"
  }
}
```

- [ ] **Step 2: 安装，更新锁文件**

Run: `pnpm install 2>&1 | grep -E '^(Packages|Done|\[ERR)'`
Expected:

```
Packages: +27 -8
Done in 3.2s using pnpm v11.10.0
```

Run: `wc -l < pnpm-lock.yaml && shasum -a 256 pnpm-lock.yaml`
Expected:

```
   15393
e47cdd6b2666af3e3282c2bfaf51d46fca59f347d34727ca6c9ea209d7629c87  pnpm-lock.yaml
```

核对锁文件（`pkgs`、`importers` 与 Task 2 Step 3 相同，在同一个 shell 中先定义它们）：

```bash
pkgs() { sed -n '/^packages:/,/^snapshots:/p' | grep -E "^  '?[@a-z]" | sed -E "s/^  '?//; s/'?:\$//" | sort; }
importers() { sed -n '/^importers:/,/^packages:/p' | sed 's/(supports-color@10\.2\.2)//g' | awk '/^  [^ ]/{keep = ($1 ~ /^web\//)} keep'; }
comm -23 <(git show HEAD:pnpm-lock.yaml | pkgs) <(pkgs < pnpm-lock.yaml) | wc -l
comm -13 <(git show HEAD:pnpm-lock.yaml | pkgs) <(pkgs < pnpm-lock.yaml) | wc -l
diff <(git show HEAD:pnpm-lock.yaml | importers) <(importers < pnpm-lock.yaml) | grep '^[<>]' | grep -oE '^[<>]|@emnapi/core@[0-9.]+|oxc-resolver@[0-9.]+' | paste -d' ' - - - | sort | uniq -c
```

Expected（依次为：少了的包、新增的包、web 下 importers 的差异）:

```
       0
      56
   9 < @emnapi/core@1.10.0 oxc-resolver@11.20.0
   9 > @emnapi/core@1.11.2 oxc-resolver@11.24.2
```

（9 行都是 web 下各包的 `tsdown` 依赖的对等后缀。）

说明：`tsdown` 构建 Plane 的包时用到的 `dts-resolver` 有一个可选的对等依赖 `oxc-resolver`。knip 写死依赖 `oxc-resolver` 11.24.2，加入 knip 之后，pnpm 把这个对等依赖从图中已有的 11.20.0 改为解析到 11.24.2（`@emnapi/core` 随之从 1.10.0 变为 1.11.2）；`tsdown` 等包自身的版本不变，11.20.0 仍在锁文件中。Step 8 确认前端的检查和构建仍然通过（spec 2.4、2.8）。

Run: `pnpm install --frozen-lockfile 2>&1 | tail -1`
Expected:

```
Done in 163ms using pnpm v11.10.0
```

- [ ] **Step 3: 写入 `knip.jsonc`**

```jsonc
// knip：找出未使用的文件、导出和依赖（M0 只出报告，M1 起作为门禁，见 docs/v0/M0-foundation/specs/P6-e2e-ci.md）。
// 构建产物都在 .gitignore 中，knip 读取 .gitignore，不需要另外忽略。
{
  "$schema": "https://unpkg.com/knip@6/schema-jsonc.json",
  "workspaces": {
    ".": {
      // 由 Makefile 调用（make gen-web、make lint-web 等），package.json 的脚本中看不到
      "ignoreDependencies": ["@redocly/cli", "turbo"]
    },
    // 下面两处导入的是生成的、不进仓库的文件，knip 找不到它们；它们由 tsc 检查
    "web/apps/web": {
      // react-router typegen 为每个路由生成的类型，在 .react-router/types/ 中，经 tsconfig 的 rootDirs 以 ./+types/… 导入；
      // knip 不按 rootDirs 解析，生成之后也找不到
      "ignoreUnresolved": ["\\+types/"]
    },
    "web/packages/i18n": {
      // 翻译键 src/types/keys.generated.ts 由 i18n 的构建生成；构建过之后 knip 能找到，会提示可以删掉这一项，不要删
      "ignoreUnresolved": ["^\\./keys\\.generated$"]
    },
    "web/packages/api-client": {
      // 类型测试只由 tsc 检查，其中的函数不会被调用
      "entry": ["test/*.typecheck.ts"],
      // 生成的类型：webhooks、$defs、operations 等没有被使用
      "ignoreIssues": { "src/schema.gen.ts": ["types"] }
    }
  }
}
```

- [ ] **Step 4: 修改 `Makefile`（新增 `knip`）**

把：

```makefile
	$(TURBO) run check:types check:lint check:format $(TURBO_QUIET)
```

替换为：

```makefile
	$(TURBO) run check:types check:lint check:format $(TURBO_QUIET)

# M0 只出报告：发现未使用的代码时退出码仍为 0，knip 自身出错时才失败；M1 去掉 --no-exit-code，作为门禁
.PHONY: knip
knip: ## 报告未使用的文件、导出和依赖（需要 Node；M0 只出报告，M1 起作为门禁）
	pnpm exec knip --no-exit-code
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected:

```
32
22
```

- [ ] **Step 5: `make knip`**

Run: `make knip > "${TMPDIR:-/tmp}/knip.txt" 2>&1; echo "exit=$?"; grep -E '^[A-Z][a-zA-Z ]+ \([0-9]+\)' "${TMPDIR:-/tmp}/knip.txt"`
Expected（本机约 2–3 秒；报告的都是迁入的 Plane 代码，M1 清零）:

```
exit=0
Unused files (131)
Unused dependencies (19)
Unused devDependencies (4)
Unlisted dependencies (2)
Unused exports (135)
Unused exported types (83)
Unused exported enum members (4)
Duplicate exports (1)
Configuration hints (2)
```

合计 379 处（不含配置提示）。

Run: `grep -E 'api-client|e2e/|^package.json|build/|dist/' "${TMPDIR:-/tmp}/knip.txt"; sed -n '/^Configuration hints/,$p' "${TMPDIR:-/tmp}/knip.txt"`
Expected（Nerve 自己的代码和构建产物都不在报告中；两条配置提示：第一条是因为已经构建过，`keys.generated.ts` 存在，i18n 的 `ignoreUnresolved` 暂时用不上；第二条是 Plane 的 `tailwind-config` 的 `main` 指向不存在的文件，交给 M1）:

```
Configuration hints (2)
^\./keys\.generated$  web/packages/i18n  knip.jsonc                                 Remove from ignoreUnresolved
tailwind.config.js                       web/packages/tailwind-config/package.json  Package entry file not found
```

不要按第一条提示删掉 i18n 的 `ignoreUnresolved`：在没有构建过的克隆上（Task 5 Step 5），`keys.generated.ts` 不存在，删掉之后报告多出 1 条"Unresolved imports"，报告随本地是否构建过而变化。web 的 `\+types/` 无论是否构建过都在起作用：去掉它，报告多出 61 条（knip 不按 `rootDirs` 解析 `./+types/…`）。

- [ ] **Step 6: 确认 knip 自身出错时 `make knip` 失败**

```bash
sed -i.orig 's/"ignoreIssues"/"ignoreIssuez"/' knip.jsonc
make knip > "${TMPDIR:-/tmp}/knip-bad.txt" 2>&1; echo "exit=$?"; grep -E 'ERROR|Error' "${TMPDIR:-/tmp}/knip-bad.txt"
mv knip.jsonc.orig knip.jsonc
```

Expected（配置中出现未知的键，knip 以退出码 2 结束，make 随之失败）:

```
exit=2
ERROR: Invalid input (location: workspaces.web/packages/api-client, unrecognized_keys: ignoreIssuez)
make: *** [knip] Error 2
```

- [ ] **Step 7: 修改 `.github/workflows/ci.yml`（`web` 任务输出 knip 报告，新增 `e2e` 任务）**

把：

```yaml
      - name: Lint
        run: make lint-web
      - name: Build
        run: make build-web
```

替换为：

```yaml
      - name: Lint
        run: make lint-web
      # M0 只出报告：发现未使用的代码不会失败，knip 自身出错（例如配置有误）才失败；M1 起作为门禁
      - name: Unused code (report only)
        run: make knip
      - name: Build
        run: make build-web

  e2e:
    name: e2e
    # server、web 都通过之后才运行；if 与这两个任务相同
    needs: [server, web]
    if: github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    env:
      # 写进 bin/nerve 的版本号，与默认值不同，S3 由此确认 make build 的注入生效
      VERSION: 0.0.0-ci.${{ github.run_number }}
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: server/go.mod
          cache-dependency-path: server/**/go.sum
      - uses: actions/setup-node@v7
        with:
          node-version-file: .node-version
      - name: Enable corepack
        run: corepack enable
      - name: Locate the pnpm store
        id: pnpm-store
        run: echo "path=$(pnpm store path --silent)" >> "$GITHUB_OUTPUT"
      - name: Cache the pnpm store
        uses: actions/cache@v6
        with:
          path: ${{ steps.pnpm-store.outputs.path }}
          key: pnpm-store-${{ runner.os }}-${{ hashFiles('pnpm-lock.yaml') }}
      - name: Install
        run: pnpm install --frozen-lockfile
      # 只装无界面的 Chromium 和它需要的系统库；不缓存浏览器：系统库缓存不了，下载和恢复缓存的耗时相当
      - name: Install the Playwright browser
        working-directory: e2e
        run: pnpm exec playwright install --with-deps --only-shell chromium
      # turbo 的缓存不跨任务保存，前端在这里再构建一次；持续集成中只有这一步完整执行 make build（见 P6 spec 2.9）
      - name: Build
        run: make build
      - name: E2E
        run: make e2e
      # 报告中带着失败用例的操作记录（trace）和截图；test-results/ 中另有每个 worker 的 nerve 日志
      - name: Upload the Playwright report
        if: failure()
        uses: actions/upload-artifact@v7
        with:
          name: playwright-report
          path: |
            e2e/playwright-report/
            e2e/test-results/
```

Run: `ruby -ryaml -e 'j = YAML.load_file(".github/workflows/ci.yml")["jobs"]; puts j.keys.join(" "); puts j["e2e"]["needs"].join(" "); puts j["e2e"]["steps"].map { |s| s["name"] || s["uses"] }.join(" | ")'`
Expected（用 macOS 自带的 Ruby 解析 YAML，确认结构）:

```
server web e2e
server web
actions/checkout@v7 | actions/setup-go@v7 | actions/setup-node@v7 | Enable corepack | Locate the pnpm store | Cache the pnpm store | Install | Install the Playwright browser | Build | E2E | Upload the Playwright report
```

- [ ] **Step 8: 锁文件改动之后，前端检查、构建和端到端测试仍然通过**

Run: `make lint-web 2>&1 | tail -4 && make e2e 2>&1 | tail -1`
Expected:

```
 Tasks:    49 successful, 49 total
Cached:    0 cached, 49 total
  Time:    36.08s 

  5 passed (4.7s)
```

（锁文件变了，turbo 的缓存全部失效，49 个任务都重新执行；`make e2e` 中的 `make build` 随之重新构建前端。）

- [ ] **Step 9: 修改 `README.md`（前端：未使用的代码）**

把：

```markdown
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包。
```

替换为：

```markdown
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包。
- **未使用的代码**：`make knip` 用 knip 报告未使用的文件、导出和依赖，配置在 `knip.jsonc`。M0 只出报告：迁入的 Plane 代码有 379 处，退出码仍为 0，只有 knip 自身出错时才失败；M1 删减之后清零，改为门禁。持续集成的 `web` 任务执行它。
```

- [ ] **Step 10: 修改 `README.md`（端到端测试：持续集成上传的内容）**

把：

```markdown
- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）和截图在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。
```

替换为：

```markdown
- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）和截图在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。持续集成的 `e2e` 任务失败时，把这两个目录上传为 `playwright-report`。
```

- [ ] **Step 11: 核对改动范围**

Run: `git status --short --untracked-files=all`
Expected:

```
 M .github/workflows/ci.yml
 M Makefile
 M README.md
 M package.json
 M pnpm-lock.yaml
?? knip.jsonc
```

- [ ] **Step 12: 提交**

```bash
git add .github/workflows/ci.yml Makefile README.md package.json pnpm-lock.yaml knip.jsonc
git commit -m "ci: run the e2e stories in an e2e job; report unused code with knip

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: 交接收尾、干净克隆上的持续集成步骤与完整走查

**Files:**
- Modify: `docs/v0/M0-foundation/handoffs/P2-server-platform-p6-e2e-notes.md`、`P3-api-contract-p6-notes.md`、`P5-web-import-p6-notes.md`
- 核对（不改）：`docs/v0/M0-foundation/M0-design.md` 第 11 节

**Interfaces:**
- Consumes: Task 1–4 的全部成果
- Produces: 关闭三个交给 P6 的交接事项；持续集成三个任务的步骤在干净的克隆上通过的证据和耗时

- [ ] **Step 1: 修改 `docs/v0/M0-foundation/handoffs/P2-server-platform-p6-e2e-notes.md`（改为 done，写明处理结果）**

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
## 处理结果（M0/P6）

1. **端口**：`e2e/fixtures/server.ts` 先在 `127.0.0.1` 上取一个空闲端口，通过 `NERVE_SERVER__ADDR` 传给 `nerve serve`，再每 100 毫秒请求一次 `/readyz`，直到返回 200（最多 30 秒；nerve 提前退出时立即失败）。不从日志中解析端口；nerve 的输出写进 `e2e/test-results/nerve-w<编号>.log`（[P6 spec](../specs/P6-e2e-ci.md) 2.5）。实测从启动到就绪约 120 毫秒。
2. **数据库**：全局准备（`e2e/global-setup.ts`）启动一个 Postgres 容器，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`；每个 worker 用 `CREATE DATABASE … TEMPLATE nerve_template` 复制出自己的库（约 60 毫秒），地址通过 `NERVE_DATABASE__URL` 传给 nerve，另加 `application_name=nerve`，S1 据此在 `pg_stat_activity` 中找到 nerve 的连接。调用方自己的 `NERVE_*` 环境变量不传给 nerve。
3. **停机**：worker 结束时向 nerve 发送 SIGTERM，等它退出，退出码不是 0 就报错；30 秒内没有退出就发送 SIGKILL 并报错。实测停机约 2 毫秒，运行结束后没有遗留的 nerve 进程。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
```

- [ ] **Step 2: 修改 `docs/v0/M0-foundation/handoffs/P3-api-contract-p6-notes.md`（改为 done，写明处理结果）**

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
来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P6）

1. **通过 `@nerve/api-client` 调用接口**：`e2e/fixtures/api.ts` 用 `createClient({ baseUrl })` 创建客户端，作为 `api` fixture 交给故事；`@nerve/e2e` 依赖 `@nerve/api-client`（`workspace:*`），`make lint-web` 包含 e2e 的类型检查（[P6 spec](../specs/P6-e2e-ci.md) 2.4）。S3 用它调用 `GET /api/v0/instance`；把路径改成 `/api/v0/instances` 时类型检查报 TS2345。S4 请求不存在的路径，用 Playwright 的 `request` 直接发出，不经过生成的客户端。
2. **转译工作区中的 TS 源码**：Playwright 1.63.0 直接转译 `@nerve/api-client` 的 `src/index.ts`（真实路径不在 `node_modules` 下），不需要额外配置（spec 2.2）。
3. **knip**（`knip.jsonc`）：`test/*.typecheck.ts` 作为 api-client 的入口文件（它们由 tsc 检查），不再报"未使用的文件"；`src/schema.gen.ts` 忽略"未使用的导出类型"（spec 2.8）。
4. **`e2e` 任务**：`needs: [server, web]`，加上与另外两个任务相同的 `if` 条件。设为必须通过的检查之前的注意事项不变（M0 设计 6.3）。

来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
```

- [ ] **Step 3: 修改 `docs/v0/M0-foundation/handoffs/P5-web-import-p6-notes.md`（改为 done，写明处理结果）**

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
来源：[M0/P5 评审记录](../reviews/P5-web-import-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P6）

构建、缓存与版本号：

1. **`e2e` 任务的构建和 pnpm 缓存**：`e2e` 任务安装 Go 和 Node，缓存 pnpm 存储的步骤与 `web` 任务相同（同一个键）。先 `make build`，再 `make e2e`；后者中的 `make build` 命中上一步留下的 turbo 和 Go 缓存，约 1 秒（[P6 spec](../specs/P6-e2e-ci.md) 2.9）。
2. **是否从 `web` 任务上传 `build/client`**：评估后不采用。`e2e` 任务自己执行完整的 `make build`，约多花 1 分钟：持续集成中只有这里完整执行 `make build`，它是 M0 完成标准"`make build` 能构建出单个可执行文件"的证据；上传 `build/client`（34 MB、1239 个文件）还要拆分 Makefile。`e2e` 任务超过 5 分钟时，改为在同一次运行中把 `web` 任务的 turbo 缓存交给 `e2e` 任务（spec 2.9）。
3. **缓存按 ref 隔离**：推送后分别记录分支冷、分支热、合并后 `main` 冷三次运行的耗时，不拿分支的热缓存数字套到 `main` 上（[P6 计划](../plans/P6-e2e-ci.md) Task 5 Step 8）。
4. **版本号**：`make build` 用 `VERSION`（默认 `0.1.0-dev`）经 `-ldflags -X` 注入；`make e2e` 把同一个值通过 `NERVE_VERSION` 交给测试，S3 核对；持续集成注入与默认值不同的 `0.0.0-ci.<运行编号>`（spec 2.7）。

turbo 与环境变量：

5. **e2e 的三项检查**：`e2e/package.json` 定义 `check:types`、`check:lint`（`oxlint --max-warnings=0 .`）、`check:format`，`make lint-web` 从 46 个任务变为 49 个（spec 2.4）。
6. **Playwright 不经 turbo 运行，也不用 `pnpm --filter`**：Makefile 直接执行 `cd e2e && pnpm exec playwright test`，Playwright 继承调用方的全部环境变量。`--filter` 写错时 pnpm 只打印 "No projects matched the filters"，退出码为 0，测试会被悄悄跳过（spec 2.7）。

断言：

7. **S2 按路径区分静态资源和接口，不按资源类型**：`/api/` 以外的请求都是静态资源（M0 设计 3.3 的路由规则），必须全部加载成功；接口请求（包括 `GET /api/instances/` 的 404）不计入。M2 起前端用 `fetch` 调用接口，也可能用 `fetch` 取静态文件，路径才是稳定的分界。S2 不断言页面文字和控制台（spec 2.6）。
8. **`web/apps/web/.env`**：e2e 不读写它，需要的环境变量都由 `e2e/` 的 fixtures 设置。S2 另外断言页面的所有请求都发往 nerve 自身：带着从 Plane 的 `.env.example` 复制的 `.env` 构建时，S2 失败，失败信息列出发往 `http://localhost:8000` 的请求（spec 2.6）。
9. **knip 与构建产物**：knip 读取 `.gitignore`，这四类构建产物已被排除，不需要另外配置；在 `make build` 之后运行，报告中没有这些路径（spec 2.8）。

来源：[M0/P5 评审记录](../reviews/P5-web-import-review.md)。
```

- [ ] **Step 4: 提交**

```bash
git add docs/v0/M0-foundation/handoffs
git commit -m "docs(M0/P6): close the P2, P3 and P5 handoffs to P6

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

Run: `grep -l '^status: open' docs/v0/M0-foundation/handoffs/*.md`
Expected: 没有输出（M0 的交接全部为 done）。

- [ ] **Step 5: 在干净的克隆上执行持续集成三个任务的步骤**

在一个新克隆中按 `ci.yml` 的顺序执行，确认不依赖本地的构建产物和 turbo 缓存。`make knip` 在安装之后先执行一次：没有构建过、生成的文件都不存在时，报告与 Task 4 相同。

```bash
CLONE="${TMPDIR:-/tmp}/nerve-p6-ci"
rm -rf "$CLONE" && git clone -q . "$CLONE" && cd "$CLONE"
pnpm install --frozen-lockfile 2>&1 | tail -1
make knip 2>&1 | grep -E '^[A-Z][a-zA-Z ]+ \([0-9]+\)'
make gen-check-web && make lint-web 2>&1 | tail -4 && make knip > /dev/null 2>&1 && make build-web 2>&1 | tail -3
make gen-check-go && make lint-go 2>&1 | tail -1 && make test 2>&1 | grep -c '^ok'
(cd e2e && pnpm exec playwright install --no-remove --only-shell chromium)
VERSION=0.0.0-ci.1 make build 2>&1 | tail -1 && VERSION=0.0.0-ci.1 make e2e 2>&1 | tail -2
git status --short
cd - && rm -rf "$CLONE"
```

Expected（本机实测：安装 6 秒，第一次 `make knip` 3 秒，`make lint-web` 32 秒，`make build-web` 7 秒，`make lint-go` 7 秒，`make test` 8 秒，`make build` 1 秒，`make e2e` 5 秒）:

```
Done in 6.7s using pnpm v11.10.0
Unused files (131)
Unused dependencies (19)
Unused devDependencies (4)
Unlisted dependencies (2)
Unused exports (135)
Unused exported types (83)
Unused exported enum members (4)
Duplicate exports (1)
Configuration hints (1)
 Tasks:    49 successful, 49 total
Cached:    0 cached, 49 total
  Time:    30.517s 

Cached:    10 cached, 11 total
  Time:    6.608s 

0 issues.
16
cd server && go build -ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=0.0.0-ci.1" -o ../bin/nerve ./cmd/nerve

  5 passed (4.1s)
```

- 没有构建过时 knip 的报告与 Task 4 Step 5 相同，只剩一条配置提示（`tailwind-config`）：两项 `ignoreUnresolved` 这时都起作用（去掉它们，报告多出 62 条"Unresolved imports"：61 条 `./+types/…`，1 条 `./keys.generated`）。
- `make test` 的 16 个包都是 `ok`。
- `VERSION=0.0.0-ci.1` 与持续集成的写法相同（环境变量），S3 通过。
- 最后的 `git status --short` 没有输出：所有产物都被 `.gitignore` 挡掉。

- [ ] **Step 6: 核对 M0 设计文档的 Phase 进度表**

Run: `grep '^| P6 ' docs/v0/M0-foundation/M0-design.md`
Expected（控制者提交 spec 和 plan 时已经更新，这里不需要改动；review 链接在代码评审后补上）:

```
| P6 | e2e-ci | 进行中 | [spec](specs/P6-e2e-ci.md) | [plan](plans/P6-e2e-ci.md) | — |
```

- [ ] **Step 7: 按 README 从头走一遍（与 P6 有关的部分）**

```bash
pnpm install --frozen-lockfile
make gen-check
make lint
make test
(cd e2e && pnpm exec playwright install --no-remove chromium)
make e2e
make knip
make
```

Expected:
- `make gen-check` 退出码 0，工作区没有变化；
- `make lint` 输出 `0 issues.`，`make lint-web` 的 49 个任务全部成功；
- `make test` 所有包都是 `ok`；
- `make e2e` 输出 `5 passed`；
- `make knip` 输出 Task 4 Step 5 的报告，退出码 0；
- `make` 列出 22 个命令（Task 4 Step 4）；
- 之后 `pgrep -fl 'bin/nerve serve'` 没有输出；约 10 秒后 `docker ps --filter label=org.testcontainers=true` 没有输出；`git status --short` 没有输出。

Run: `cd server && grep -E '^(go|toolchain) ' go.mod tools/go.mod`
Expected（本 Phase 没有改动 Go 模块）:

```
go.mod:go 1.27
go.mod:toolchain go1.27.1
tools/go.mod:go 1.27
tools/go.mod:toolchain go1.27.1
```

- [ ] **Step 8: 推送并确认持续集成（由控制者执行）**

1. 推送分支，确认 `CI` 的三个任务都通过：`server`、`web`（含 `Unused code (report only)` 一步，日志中是 Task 4 Step 5 的报告）、`e2e`（`needs` 两者，最后一步 `Upload the Playwright report` 被跳过）。
2. 记录耗时，写进 P6 的 review（spec 2.9 的估计：`e2e` 任务约 3–4 分钟，整个工作流约 6 分钟）：
   - 分支上的第一次运行：锁文件变了，`web` 任务的 pnpm 缓存未命中（冷），`e2e` 任务命中 `web` 任务刚保存的缓存；
   - 分支上再推送一个只改文档的提交：全部命中缓存；
   - 合并到 `main` 之后的第一次运行：GitHub 的缓存按分支隔离，`main` 读不到分支上保存的缓存，又是冷的。
   每次分别记下 `e2e` 任务中 `Install`、`Install the Playwright browser`、`Build`、`E2E` 四步和整个任务的耗时。`e2e` 任务超过 5 分钟时，按 spec 2.9 的备选办法处理（在同一次运行中把 `web` 任务的 turbo 缓存交给 `e2e` 任务）。
3. 确认失败时上传报告：从本分支建一个临时分支，把 `e2e/stories/smoke/s4-unknown-api.spec.ts` 中的 `toBe(404)` 改为 `toBe(418)` 并推送；确认 `e2e` 任务在 `E2E` 一步失败，`Upload the Playwright report` 上传了 `playwright-report`，其中有 S4 的 trace。然后删除这个临时分支（本地和远端）。

---

## 完成后

P6 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M0-foundation/reviews/P6-e2e-ci-review.md`：记录 Task 5 Step 8 的耗时，裁定 spec 第 3 节的差异，按 spec 第 7 节建立交给 M1、M2 的 handoff，同步 spec 第 3 节末尾列出的上级文档；把 M0 设计文档中 P6 的状态改为"已完成"、补上 review 链接。P6 是 M0 的最后一个 Phase：随后由控制者按 M0 设计第 10 节逐项核对完成标准（spec 第 8 节列出了 P6 提供的证据），并把总体设计中 M0 的状态改为"已完成"。
