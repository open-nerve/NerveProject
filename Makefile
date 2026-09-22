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

# 前端任务由 turbo 按 turbo.json 编排；关闭匿名使用数据上报，只打印失败任务的输出
TURBO := TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo
TURBO_QUIET := --output-logs=errors-only
# make build 把前端的构建产物复制到这里，由 go:embed 编进 nerve
WEBUI_DIST := server/internal/platform/webui/dist
# nerve 的版本号：make build 把它写进 bin/nerve，端到端测试 S3 核对它。
# 默认值与 server/internal/platform/buildinfo 中的相同；发布时指定，例如 make build VERSION=0.1.0
VERSION ?= 0.1.0-dev
GO_LDFLAGS := -X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=$(VERSION)

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

.PHONY: web-dev
web-dev: ## 启动前端开发服务器 http://127.0.0.1:3000，/api 转发给 make run 的后端；Ctrl-C 停止
	$(TURBO) run dev --filter=web

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

# 关键词守卫（tools/keywords.mjs，规则在 tools/keywords.json）检查整个仓库，不属于任何工作区包，所以不经过 turbo
.PHONY: lint-web
lint-web: ## 关键词守卫；前端类型检查、oxlint（警告数等于上限）、格式检查、中英文翻译键一致（需要 Node）
	node tools/keywords.mjs
	$(TURBO) run check:types check:lint check:format check:sync $(TURBO_QUIET)

# M0 只出报告：发现未使用的代码时退出码仍为 0，knip 自身出错时才失败；M1 去掉 --no-exit-code，作为门禁
.PHONY: knip
knip: ## 报告未使用的文件、导出和依赖（需要 Node；M0 只出报告，M1 起作为门禁）
	pnpm exec knip --no-exit-code

# go test 的缓存不跟踪 server/ 之外的文件，契约测试读取的 api/dist/openapi.yaml 改了也会重放旧结果，所以不用缓存
.PHONY: test
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...

.PHONY: test-web
test-web: ## 运行前端单元测试（各包 test 脚本中的 vitest，经 turbo；需要 Node；持续集成 web 任务）
	$(TURBO) run test $(TURBO_QUIET)

.PHONY: build
build: build-web ## 构建前端并嵌入 Go 程序，编译出 bin/nerve（需要 Node 和 Go）
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
	cp -R web/apps/web/build/client/. $(WEBUI_DIST)/
	cd server && go build -ldflags "$(GO_LDFLAGS)" -o ../bin/nerve ./cmd/nerve

.PHONY: build-web
build-web: ## 构建前端，产物在 web/apps/web/build/client（需要 Node；持续集成 web 任务）
	$(TURBO) run build --filter=web $(TURBO_QUIET)

.PHONY: e2e
e2e: build ## 构建 bin/nerve 并运行端到端测试（需要 Node、Go、Docker 和 Playwright 的浏览器，见 README）
	cd e2e && NERVE_VERSION=$(VERSION) pnpm exec playwright test

.PHONY: plane-schema
plane-schema: ## 重新生成 Plane 表结构快照（需要 Docker，见 tools/plane-schema/README.md）
	tools/plane-schema/extract.sh
