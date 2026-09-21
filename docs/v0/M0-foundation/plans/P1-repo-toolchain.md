# M0/P1 仓库与工具链 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立仓库的工程骨架：锁定工具链版本、提供开发数据库和统一的 Makefile 命令、实现 `platform/buildinfo` 包、搭起持续集成。

**Architecture:** 仓库根目录是 pnpm 工作区的根（P1 中还没有包）；`server/` 是 Go 模块，开发工具通过独立的 `server/tools/go.mod` 和 `go tool -modfile` 锁定版本；golangci-lint 由 `make tools` 下载锁定版本的二进制；本地和持续集成执行同一套 Makefile 命令。

**Tech Stack:** Go 1.27.1、pnpm 11.10.0（corepack）、Node 24、Docker Compose、PostgreSQL 18.6、golangci-lint 2.13.2、oapi-codegen 2.8.0、GitHub Actions（checkout / setup-go / setup-node 均为 v7）。

**Spec:** `docs/v0/M0-foundation/specs/P1-repo-toolchain.md`（上级：`docs/v0/M0-foundation/M0-design.md`）

## Global Constraints

- Go：`server/go.mod` 中写 `go 1.27` 和 `toolchain go1.27.1`；模块路径 `github.com/open-nerve/NerveProject/server`。
- pnpm：`packageManager` 由 `corepack use pnpm@11.10.0` 写入（带 sha512 校验值）；Node 版本文件 `.node-version` 内容为 `24`。
- PostgreSQL 镜像写死为 `postgres:18.6`；数据卷挂载到 `/var/lib/postgresql`（不是 `/var/lib/postgresql/data`）。
- golangci-lint 版本 `2.13.2`；oapi-codegen 版本 `v2.8.0`。
- Makefile 必须兼容 macOS 自带的 GNU Make 3.81；只写 P1 中能正常工作的命令，不写占位命令。
- 除 Docker、Go、Node 以外，不要求全局安装任何工具。
- GitHub Actions：`actions/checkout@v7`、`actions/setup-go@v7`、`actions/setup-node@v7`（2026-09-22 已通过 GitHub 接口核实）。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录 `/Users/xiaoruan/project/nerve-project` 下执行，除非步骤中另有说明。

## 文件结构

| 文件 | 职责 | Task |
|---|---|---|
| `.editorconfig` | 编辑器格式约定 | 1 |
| `.node-version` | Node 主版本 | 1 |
| `.gitignore`（修改） | 忽略构建和依赖产物 | 1 |
| `package.json` | pnpm 工作区根，锁定 pnpm 版本 | 1 |
| `pnpm-workspace.yaml` | 工作区包的路径 | 1 |
| `pnpm-lock.yaml` | 依赖锁文件（生成） | 1 |
| `server/go.mod` | Go 主模块 | 2 |
| `server/tools/go.mod`、`server/tools/go.sum` | 开发工具版本锁定 | 2 |
| `server/internal/platform/buildinfo/buildinfo.go` | 构建信息 | 2 |
| `server/internal/platform/buildinfo/buildinfo_test.go` | 构建信息的单元测试 | 2 |
| `deploy/compose.dev.yaml` | 开发数据库 | 3 |
| `Makefile` | 统一命令入口 | 3、4 |
| `.github/workflows/ci.yml` | 持续集成 | 5 |
| `README.md`（修改） | 开发环境说明 | 6 |
| `docs/README.md`（修改） | 跨 M handoff 的命名规则 | 6 |
| `docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md` | 移交给 M2 的事项 | 6 |
| `docs/v0/M0-foundation/M0-design.md`（修改） | Phase 进度表 | 6 |

---

### Task 1: 根目录工程文件与 pnpm 工作区

**Files:**
- Create: `.editorconfig`、`.node-version`、`package.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`（生成）
- Modify: `.gitignore`

**Interfaces:**
- Consumes: 无
- Produces: 根目录 `package.json`（P5 往里加脚本和依赖）；`pnpm-workspace.yaml` 的 `packages` 列表（P5、P6 的包会自动被纳入）

- [ ] **Step 1: 写 `.editorconfig`**

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 2

[*.go]
indent_style = tab
indent_size = 4

[Makefile]
indent_style = tab

[*.md]
trim_trailing_whitespace = false
```

- [ ] **Step 2: 写 `.node-version`**

内容只有一行：

```
24
```

- [ ] **Step 3: 在 `.gitignore` 末尾追加构建和依赖产物**

```gitignore

# 构建与依赖产物
bin/
node_modules/
.turbo/
coverage/
*.test
*.out
```

- [ ] **Step 4: 写 `pnpm-workspace.yaml`**

```yaml
packages:
  - web/apps/*
  - web/packages/*
  - e2e
```

- [ ] **Step 5: 写 `package.json`（先不含 packageManager 字段）**

```json
{
  "name": "nerve",
  "private": true,
  "license": "AGPL-3.0-only",
  "engines": {
    "node": ">=24"
  }
}
```

- [ ] **Step 6: 用 corepack 锁定 pnpm 版本并生成 lockfile**

Run: `COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack use pnpm@11.10.0`
Expected: 命令成功；`package.json` 多出 `"packageManager": "pnpm@11.10.0+sha512.0b7f8b…"`；根目录生成 `pnpm-lock.yaml`，其中含 `lockfileVersion: '9.0'` 和 `importers: .: {}`。

- [ ] **Step 7: 验证锁文件模式安装能成功**

Run: `corepack enable && pnpm install --frozen-lockfile`
Expected: 输出 `Already up to date`，退出码 0。

Run: `git status --short`
Expected: 列出 `.editorconfig`、`.node-version`、`.gitignore`、`package.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`；**不应**出现 `node_modules/`（已被忽略）。

- [ ] **Step 8: 提交**

```bash
git add .editorconfig .node-version .gitignore package.json pnpm-workspace.yaml pnpm-lock.yaml
git commit -m "build: add root pnpm workspace, node version and editorconfig

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Go 模块、开发工具锁定与 buildinfo 包

**Files:**
- Create: `server/go.mod`、`server/tools/go.mod`、`server/tools/go.sum`（生成）
- Create: `server/internal/platform/buildinfo/buildinfo.go`
- Test: `server/internal/platform/buildinfo/buildinfo_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - `buildinfo.Get() buildinfo.Info`
  - `type Info struct { Version string; Commit string; CommitTime string; Modified bool }`
  - 包内变量 `version`（默认 `"0.1.0-dev"`），发布构建时用 `-ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=<版本>"` 覆盖。P2 的 `nerve version` 和 P3 的 `instance` 模块会用到。

- [ ] **Step 1: 写 `server/go.mod`**

```
module github.com/open-nerve/NerveProject/server

go 1.27

toolchain go1.27.1
```

Run: `cd server && go version`
Expected: `go version go1.27.1 darwin/arm64`（如果本机是 Go 1.26，这一步会自动下载 1.27.1）

- [ ] **Step 2: 建立工具模块并锁定 oapi-codegen**

```bash
mkdir -p server/tools
cat > server/tools/go.mod <<'EOF'
module github.com/open-nerve/NerveProject/server/tools

go 1.27

toolchain go1.27.1
EOF
cd server/tools && go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0
```

Expected: `server/tools/go.mod` 中出现 `tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` 和 `github.com/oapi-codegen/oapi-codegen/v2 v2.8.0`；生成 `server/tools/go.sum`。

- [ ] **Step 3: 验证工具能通过 `go tool` 调用**

Run: `cd server && go tool -modfile=tools/go.mod oapi-codegen -version`
Expected: 输出两行：`github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` 和 `v2.8.0`

- [ ] **Step 4: 写失败的测试 `server/internal/platform/buildinfo/buildinfo_test.go`**

```go
package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestFromSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings []debug.BuildSetting
		want     Info
	}{
		{
			name:     "no vcs settings keeps unknown defaults",
			settings: nil,
			want:     Info{Version: "1.2.3", Commit: "unknown", CommitTime: "unknown"},
		},
		{
			name: "vcs settings are applied and other settings ignored",
			settings: []debug.BuildSetting{
				{Key: "GOOS", Value: "linux"},
				{Key: "vcs.revision", Value: "abc123"},
				{Key: "vcs.time", Value: "2026-09-22T10:00:00Z"},
				{Key: "vcs.modified", Value: "true"},
			},
			want: Info{Version: "1.2.3", Commit: "abc123", CommitTime: "2026-09-22T10:00:00Z", Modified: true},
		},
		{
			name:     "clean working tree is not modified",
			settings: []debug.BuildSetting{{Key: "vcs.modified", Value: "false"}},
			want:     Info{Version: "1.2.3", Commit: "unknown", CommitTime: "unknown"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fromSettings("1.2.3", tt.settings); got != tt.want {
				t.Errorf("fromSettings() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetReportsStampedVersion(t *testing.T) {
	if got := Get().Version; got != version {
		t.Errorf("Get().Version = %q, want %q", got, version)
	}
}
```

- [ ] **Step 5: 运行测试，确认失败**

Run: `cd server && go test ./internal/platform/buildinfo/`
Expected: 编译失败，报 `undefined: Info`、`undefined: fromSettings`、`undefined: Get`、`undefined: version`

- [ ] **Step 6: 写实现 `server/internal/platform/buildinfo/buildinfo.go`**

```go
// Package buildinfo reports which build of nerve is running.
package buildinfo

import "runtime/debug"

// version is the product version. Release builds override it with
//
//	-ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=<version>"
var version = "0.1.0-dev"

const unknown = "unknown"

// Info describes the running binary.
type Info struct {
	Version    string // product version, e.g. "0.1.0"
	Commit     string // git revision the binary was built from
	CommitTime string // time of that revision, RFC 3339
	Modified   bool   // built from a working tree with uncommitted changes
}

// Get returns the build metadata of the running binary. Commit details come
// from the VCS stamp the Go toolchain embeds when building inside a git
// checkout; they are "unknown" for test binaries and `go run`.
func Get() Info {
	var settings []debug.BuildSetting
	if bi, ok := debug.ReadBuildInfo(); ok {
		settings = bi.Settings
	}
	return fromSettings(version, settings)
}

func fromSettings(productVersion string, settings []debug.BuildSetting) Info {
	info := Info{Version: productVersion, Commit: unknown, CommitTime: unknown}
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			info.Commit = s.Value
		case "vcs.time":
			info.CommitTime = s.Value
		case "vcs.modified":
			info.Modified = s.Value == "true"
		}
	}
	return info
}
```

- [ ] **Step 7: 运行测试，确认通过**

Run: `cd server && go test ./...`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/buildinfo`

Run: `cd server && go vet ./... && gofmt -l .`
Expected: 两条命令都没有输出，退出码 0。

- [ ] **Step 8: 提交**

```bash
git add server/go.mod server/tools/go.mod server/tools/go.sum server/internal/platform/buildinfo
git commit -m "build(server): add Go module, pinned dev tools and buildinfo package

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: 开发数据库与 Makefile 基础命令

**Files:**
- Create: `deploy/compose.dev.yaml`
- Create: `Makefile`

**Interfaces:**
- Consumes: 无
- Produces: `make help`、`make dev-db`、`make dev-db-down`、`make dev-db-reset`；Compose 项目名 `nerve-dev`，服务名 `db`；开发库连接串 `postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable`（P2 的 `config.dev.yaml` 会用到）

- [ ] **Step 1: 写 `deploy/compose.dev.yaml`**

```yaml
# 本地开发数据库。账号密码只在本机使用，不是机密。
name: nerve-dev

services:
  db:
    image: postgres:18.6
    environment:
      POSTGRES_USER: nerve
      POSTGRES_PASSWORD: nerve
      POSTGRES_DB: nerve
    ports:
      - "${NERVE_DEV_DB_PORT:-55432}:5432"
    volumes:
      # Postgres 18 起数据目录为 /var/lib/postgresql/18/docker，
      # 数据卷必须挂在 /var/lib/postgresql，挂在旧路径 /var/lib/postgresql/data 数据不会进卷。
      - db-data:/var/lib/postgresql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nerve -d nerve"]
      interval: 2s
      timeout: 5s
      retries: 30

volumes:
  db-data:
```

- [ ] **Step 2: 写 `Makefile`（本 Task 只含 help 和数据库命令）**

注意：命令体必须用 **Tab** 缩进。

```make
# Nerve 开发命令入口。运行 `make` 或 `make help` 查看所有命令。
# 需兼容 macOS 自带的 GNU Make 3.81。

SHELL := /bin/bash
.DEFAULT_GOAL := help

DEV_COMPOSE := docker compose -f deploy/compose.dev.yaml

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
```

- [ ] **Step 3: 验证 help**

Run: `make`
Expected: 列出 `help`、`dev-db`、`dev-db-down`、`dev-db-reset` 四个命令及中文说明。

- [ ] **Step 4: 验证数据库能启动、版本正确**

Run: `make dev-db`
Expected: 命令在数据库健康后返回，输出中包含 `Container nerve-dev-db-1  Healthy`。

Run: `docker compose -f deploy/compose.dev.yaml exec -T db psql -U nerve -d nerve -tAc 'select version()'`
Expected: 以 `PostgreSQL 18.6` 开头。

- [ ] **Step 5: 验证数据持久化与重置**

```bash
docker compose -f deploy/compose.dev.yaml exec -T db psql -U nerve -d nerve -c 'create table p1_probe(id int)'
make dev-db-down && make dev-db
docker compose -f deploy/compose.dev.yaml exec -T db psql -U nerve -d nerve -tAc "select to_regclass('p1_probe')"
```
Expected: 最后一条输出 `p1_probe`（重启后数据还在）。

```bash
make dev-db-reset && make dev-db
docker compose -f deploy/compose.dev.yaml exec -T db psql -U nerve -d nerve -tAc "select to_regclass('p1_probe')"
```
Expected: 最后一条输出为空行（重置后表不存在）。

- [ ] **Step 6: 提交**

```bash
git add deploy/compose.dev.yaml Makefile
git commit -m "build: add dev Postgres 18 compose file and Makefile entry point

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Makefile 的 Go 质量命令（tools / lint / test）

**Files:**
- Modify: `Makefile`

**Interfaces:**
- Consumes: Task 2 的 `server/` 模块和 `buildinfo` 包；Task 3 的 `Makefile`
- Produces: `make tools`（安装 `bin/golangci-lint` v2.13.2）、`make lint`、`make test`。Task 5 的持续集成直接调用 `make lint` 和 `make test`。

- [ ] **Step 1: 在 `Makefile` 中追加变量（放在 `DEV_COMPOSE` 那一行之后）**

```make
GOLANGCI_LINT_VERSION := 2.13.2
BIN_DIR := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint
```

- [ ] **Step 2: 在 `Makefile` 末尾追加命令**

```make
.PHONY: tools
tools: ## 安装锁定版本的 golangci-lint 到 ./bin
	@if $(GOLANGCI_LINT) --version 2>/dev/null | grep -q "version $(GOLANGCI_LINT_VERSION) "; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) 已安装"; \
	else \
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(BIN_DIR) v$(GOLANGCI_LINT_VERSION); \
	fi

.PHONY: lint
lint: tools ## 运行 golangci-lint（server）
	cd server && $(GOLANGCI_LINT) run ./...

.PHONY: test
test: ## 运行 Go 测试（server）
	cd server && go test ./...
```

- [ ] **Step 3: 验证 tools 能安装，并且重复执行会跳过**

Run: `make tools`
Expected: 输出 `installed …/bin/golangci-lint`；`bin/golangci-lint --version` 输出以 `golangci-lint has version 2.13.2` 开头。

Run: `make tools`
Expected: 输出 `golangci-lint 2.13.2 已安装`，不再下载。

- [ ] **Step 4: 验证 lint 和 test**

Run: `make lint`
Expected: 输出 `0 issues.`，退出码 0。

Run: `make test`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/buildinfo`，退出码 0。

- [ ] **Step 5: 验证 lint 能发现问题（确认门禁有效，验证后撤销）**

在 `server/internal/platform/buildinfo/buildinfo.go` 末尾临时追加：

```go
func unusedProbe() {}
```

Run: `make lint`
Expected: 失败，报告 `func unusedProbe is unused (unused)`，退出码非 0。

撤销这段临时代码：`git checkout server/internal/platform/buildinfo/buildinfo.go`，然后再执行 `make lint`，确认输出 `0 issues.`。

- [ ] **Step 6: 验证 help 列出了新命令**

Run: `make`
Expected: 共 7 个命令：`help`、`dev-db`、`dev-db-down`、`dev-db-reset`、`tools`、`lint`、`test`。

- [ ] **Step 7: 提交**

```bash
git add Makefile
git commit -m "build: add tools, lint and test Makefile targets

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: 持续集成

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: Task 1 的 `package.json`、`pnpm-lock.yaml`、`.node-version`；Task 2 的 `server/go.mod`；Task 4 的 `make lint`、`make test`
- Produces: 持续集成中的 `server`、`web` 两个任务。之后的 Phase 往里面追加步骤：P2 加测试，P3 加 `make gen-check`，P5 加前端检查和构建，P6 加 `e2e` 任务。

- [ ] **Step 1: 写 `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
  pull_request:

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read

jobs:
  server:
    name: server
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: server/go.mod
          cache-dependency-path: server/**/go.sum
      - name: Go version
        working-directory: server
        run: go version
      - name: Lint
        run: make lint
      - name: Test
        run: make test

  web:
    name: web
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-node@v7
        with:
          node-version-file: .node-version
      - name: Enable corepack
        run: corepack enable
      - name: Install
        run: pnpm install --frozen-lockfile
```

- [ ] **Step 2: 用 actionlint 在本地检查工作流文件**

Run: `docker run --rm -v "$PWD":/repo -w /repo rhysd/actionlint:latest -color`
Expected: 没有输出，退出码 0。

- [ ] **Step 3: 提交并推送**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add server and web CI jobs

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
git push
```

- [ ] **Step 4: 确认持续集成通过**

仓库是私有的，本机无法匿名查询运行结果。请用户打开 `https://github.com/open-nerve/NerveProject/actions`，确认最新一次 `CI` 运行中 `server` 和 `web` 两个任务都是绿色，并确认 `Go version` 这一步输出的是 `go1.27.1`。

如果 `Go version` 显示的不是 1.27.1（`setup-go` 没有采用 `toolchain` 指令），在 `setup-go` 步骤中把 `go-version-file` 改成 `go-version: '1.27.1'`，重新提交并推送，直到通过。

---

### Task 6: 文档收尾

**Files:**
- Modify: `README.md`
- Modify: `docs/README.md`
- Create: `docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md`
- Modify: `docs/v0/M0-foundation/M0-design.md`

**Interfaces:**
- Consumes: Task 1–5 的全部成果
- Produces: 开发者上手说明；移交给 M2 的事项

- [ ] **Step 1: 在 `README.md` 的"## 版权"之前插入"开发环境"一节**

````markdown
## 开发环境

需要安装：
- Docker（含 Compose v2）
- Go 1.26 或更高。第一次在 `server/` 下执行 Go 命令时，会自动下载 `server/go.mod` 指定的 Go 1.27.1
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）

第一次启动：

```bash
make dev-db   # 启动本地 Postgres 18
make test     # 运行测试
make          # 查看所有命令
```

````

- [ ] **Step 2: 在 `docs/README.md` 的"## handoff 规则"一节末尾追加跨 M 的命名规则**

```markdown
- 放进其他 M 的 handoff，文件名以来源开头，例如 `docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md`（来自 M0/P1）。
```

- [ ] **Step 3: 写 `docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md`**

```markdown
---
status: open
from: M0/P1
to: M2
created: 2026-09-22
---

# sqlc 接入时的两个注意事项

M2 首次接入 sqlc（v1.31.1，放进 `server/tools/go.mod`）时处理：

1. **sqlc 依赖 cgo**：它通过 pg_query_go 解析 SQL，需要 C 编译器。
   - 验证本机（macOS，需要 Xcode Command Line Tools）和持续集成（ubuntu）上，`go tool -modfile=tools/go.mod sqlc version` 都能编译运行。
   - 如果 cgo 构建有问题，改用 sqlc 官方 Docker 镜像（`sqlc/sqlc:1.31.1`）运行代码生成。
2. **sqlc 的解析器基于 PG 17**：它不认识 PG 18 新增的函数，例如 `uuidv7()`。我们的 ID 由应用生成（Go 1.27 标准库的 `uuid.NewV7()`），迁移中不要写 `DEFAULT uuidv7()`；如果需要用到其他 PG 18 特有的语法，先验证 sqlc 能否解析。

来源：[M0/P1 spec](../../M0-foundation/specs/P1-repo-toolchain.md) 第 5 节。
```

- [ ] **Step 4: 更新 `docs/v0/M0-foundation/M0-design.md` 的 Phase 进度表**

把 P1 那一行改为（review 在代码评审后补上）：

```markdown
| P1 | repo-toolchain | 进行中 | [spec](specs/P1-repo-toolchain.md) | [plan](plans/P1-repo-toolchain.md) | — |
```

- [ ] **Step 5: 按 README 的说明，从头走一遍上手流程**

```bash
make dev-db-reset
make dev-db && make test && make lint && make
```
Expected: 全部成功；`make` 列出 7 个命令。

- [ ] **Step 6: 提交并推送**

```bash
git add README.md docs/README.md docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md docs/v0/M0-foundation/M0-design.md
git commit -m "docs(M0/P1): add dev setup guide and sqlc handoff to M2

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
git push
```

---

## 完成后

P1 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M0-foundation/reviews/P1-repo-toolchain-review.md`，并把 M0 设计文档中 P1 的状态改为"已完成"、补上 review 链接。
