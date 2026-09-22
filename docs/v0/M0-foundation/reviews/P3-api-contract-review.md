# M0/P3 接口契约流水线与试点模块：评审记录

| 项 | 内容 |
|---|---|
| Phase | M0/P3 `api-contract` |
| 日期 | 2026-09-22 |
| 结论 | **通过**（整分支评审提出的问题已在修复轮中处理完；随后按 Ruling 8 处理了评审中新发现的测试缓存问题） |
| spec / plan | [spec](../specs/P3-api-contract.md) / [plan](../plans/P3-api-contract.md) |
| 分支 | `worktree-m0-p3-api-contract`，从 `main` 的 `ebe1890` 分出，提交从 `591972a` 到本评审记录所在的提交 |

## 1. 范围与结果

P3 把"接口描述 → 生成代码 → 模块 → 接线 → HTTP → TS 客户端"这条链路走通，定下以后每个模块照着做的写法（spec 2.1）：
- 验证 OpenAPI 3.1 在整条工具链上可用，定下版本（见第 2 节）。
- `api/` 目录、Redocly 打包（入口 `api/openapi.yaml`、`common.yaml` 放 `Problem`/`FieldError`、打包结果 `api/dist/openapi.yaml`）。
- 每个模块单独运行 oapi-codegen，公共组件通过 import-mapping 生成到共享包 `platform/httpserver/apigen`。
- 生成代码的三处错误出口改为输出 problem+json（`httpserver.APIErrors`），新增平台错误码 `bad_request`。
- 试点模块 `instance`（`GET /api/v0/instance`），在 `bootstrap` 中接线。
- 用 kin-openapi 做契约测试（`platform/httpserver/apitest`，只被测试导入）：模块的响应、平台写出的全部 problem、写法约定（`TestContractFollowsAuthoringRules`）。
- `web/packages/api-client`：生成的 TS 类型和基于 openapi-fetch 的 `createClient`。
- `make gen`/`make gen-check` 按区域拆分为 `-go`/`-web`，持续集成的 `server`/`web` 任务只跑自己区域的命令，同仓 PR 不重复运行。

## 2. OpenAPI 3.1 验证结论

**结论：使用 OpenAPI 3.1（`openapi: 3.1.0`）。** 整条链路（Redocly 2.53.3 打包 → oapi-codegen v2.8.0 strict-server/std-http-server → kin-openapi v0.149.0 校验 → openapi-typescript 7.13.0/openapi-fetch 0.17.0）在 M0 设计第 4 节列出的六类写法（可为空字段、枚举、`oneOf`、日期时间、problem+json、游标分页）上全部通过。验证中发现的三处 oapi-codegen 不足（跨文件引用 `components/responses` 编译失败、可为空枚举写成 `enum: [..., null]` 多出一个 `"<nil>"` 常量、`const` 生成 `interface{}`）都有等价写法绕开，定为写法约定，不影响结论。

验证方法：在仓库外的临时目录里搭一个假的 `sample` 模块，结构与 P3 的正式布局相同，按正式流程打包、生成 Go 和 TS 代码，写一个 strict server 实现，用 kin-openapi 校验真实的请求和响应（10 组正例全部通过，14 个故意改坏的反例全部按预期原因被拒绝）；TS 侧用 4 处 `@ts-expect-error` 证明类型检查确实生效。详细证据表和附录 A 的样例见 [P3 spec](../specs/P3-api-contract.md) 2.2。

## 3. 各 Task 的评审

9 个 Task 全部一次通过（评审或控制者直接核实均无发现需要修复的问题）：

| Task | 内容 | 提交范围 | 核实方式 |
|---|---|---|---|
| 1 | `httpserver.APIErrors`：生成代码的错误改为 problem+json | 591972a..f273b94 | 评审（sonnet），clean |
| 2 | `api/` 的 OpenAPI 3.1 描述与 Redocly 打包 | f273b94..05ec416 | 评审（sonnet），clean；`Problem` schema 与 `httpserver.Problem` 一致 |
| 3 | instance 的 Go 接口层与共享包 `apigen` | 05ec416..629deb8 | 控制者直接核实：生成物与 brief 字节级一致（SHA-256），不含 google/uuid |
| 4 | `apitest` 契约校验工具与 `contract_test.go` | 629deb8..bd1c02c | 评审（sonnet），clean；kin-openapi 行为对照源码核实 |
| 5 | 试点模块 `instance`（domain/app/adapter） | bd1c02c..6b81b0d | 评审（sonnet），clean；模块模板分层核实 |
| 6 | `GET /api/v0/instance` 在 `bootstrap` 中接线 | 6b81b0d..fab5aec | 控制者直接核实：两个改动文件与 brief 字节级一致，lint/test/curl 证据齐全 |
| 7 | `web/packages/api-client` TS 客户端 | fab5aec..41d4349 | 评审（sonnet），clean；`@ts-expect-error` 断言现场核实 |
| 8 | `gen`/`gen-check`/`lint` 按区域拆分，CI 拆分为 `server`/`web` | 41d4349..58bff57 | 评审（sonnet），clean；CI 在拆分后的任务上跑绿，CI 失败验收演示完成（第 6 节） |
| 9 | README、关闭 P1/P2 handoff、文档收尾 | 58bff57..30946b0 | 控制者直接核实：handoff 关闭理由准确、提交署名正确、走查证据齐全 |

实现者分工：haiku 做转录性 Task（1、5、7、9），sonnet 做生成器/工具/CI 类 Task（2、3、4、6、8）；评审用 sonnet；同一时间只有一个实现者在跑，评审(N) 与实现(N+1) 流水线并行。

## 4. 整分支评审（opus）：With fixes

整分支评审在 30946b0 上进行，跑了 `go vet`、完整与 `-short` 的 `go test`、golangci-lint（0 issues）、`pnpm typecheck`、Make 3.81 下的 `make -n`；核实了 `nerve` 二进制不含 kin-openapi/jsonschema/oasdiff/google-uuid；核实了 4 份生成物从 `git archive HEAD` 重新生成后字节级一致；用 Redocly 探针确认未解析的 `$ref` 会让打包失败（退出码 1），组件名冲突会改名为 `X-2`（退出码 0，被 apitest 的命名测试拦住）。

**Important（4 项，全部处理）：**

| # | 发现 | 处理 |
|---|---|---|
| 1 | `InternalError` 在响应已经开始后仍把 problem 追加到 200 响应体，触发 "superfluous WriteHeader" | 已修（G1，提交 `57da518`）：响应已开始时改为记 WARN 日志并 `panic(http.ErrAbortHandler)` 中断连接 |
| 2 | spec 2.4 的写法约定基本没有测试兜底，只有跨文件 response（编译失败）和组件名冲突（apitest）两项被拦住 | 已修（G2，提交 `627e824`）：新增 `TestContractFollowsAuthoringRules`，遍历 `dist` 和模块文件，检查全部写法约定 |
| 3 | 同仓 PR 跳过的任务用同一个检查名字报告 Success，一旦把它设为必须通过的检查，跳过的运行也能满足要求 | 目前没有设置必须通过的检查，不是现实风险；裁定为文档说明（Ruling 6），已写入 [M0 设计](../M0-design.md) 6.3、[P1 handoff](../handoffs/P1-repo-toolchain-ci-split.md)、P6 handoff |
| 4 | 架构规则 8 只挡住 `apitest` 包本身，挡不住生成代码或其他途径间接引入 kin-openapi、google/uuid | 已修（G3，提交 `16bbfb8`）：新增 `TestNerveBinaryLinksNoBannedModule`，用 `packages.Load` 读取 `./cmd/nerve` 的全部传递依赖并拦住禁用前缀 |

**Minor（8 项，全部处理）：**

| # | 发现 | 处理 |
|---|---|---|
| 5 | `apigen` 的配置缺 `always-prefix-enum-values` | 已修（G4，提交 `7bf56fa`），生成结果不变 |
| 6 | `prefer-skip-optional-pointer` 应与 `nullable-type`、uuid 的 type-mapping 一起决定 | 移交 M2 handoff（Ruling 6） |
| 7 | Go 1.27 的 `http.StatusText(422)` 是 "Unprocessable Entity"，v0 §3.5 和 `apitest_test.go` 却写 "Unprocessable Content" | 已修（G5，提交 `627e824`）；v0 设计 §3.5 的示例已在本次收尾更正（Ruling 7） |
| 8 | `BadRequest` 的请求体解码错误会带出 Go 的类型名 | 移交 M2 handoff（Ruling 6） |
| 9 | 非规范的 `/api/` 路径（`/api/v0//instance`、`/api/v0/./instance`、`/api`）会先被 ServeMux 307 跳转 | 接受，不作特殊处理；已在本次收尾写入 [M0 设计](../M0-design.md) 3.3（Ruling 6） |
| 10 | 接口描述一旦声明 `servers`，apitest 的 legacy 路由会连带匹配 scheme/host，导致每个 `CheckResponse` 失败 | 已修（G6，提交 `627e824`）：`load` 清空 `doc.Servers` 及路径/操作级的 `Servers` |
| 11 | 组件名检查只覆盖 4 类组件 | 已修（G7，提交 `627e824`）：补齐全部 9 类组件 |
| 12 | api-client 的脚本叫 `typecheck`，与 Plane 和 turbo 的 `check:types` 不一致 | 已修（G8，提交 `7bf56fa`） |

修复轮之后新增两个测试期间发现的问题：
- **C1（Important，测试缓存）**：Go 的测试缓存不跟踪 `server/` 模块根目录之外的文件，只改 `api/` 时缓存的 `go test` 会重放旧结果，CI 也会因为 `actions/setup-go` 恢复 GOCACHE 而受影响。按 Ruling 8 采用方案 (a)：`make test` 加 `-count=1`（提交 `0cd0b92`），spec 2.8 的相关描述已更正。
- **C2（Minor，接受）**：G1 中断一个已开始的响应后，访问日志仍记录 500（handler 没有正常结束）。这与既有的"panic 后中断"行为一致，按 Ruling 9 接受，不改。
- **C3（Minor，移交）**：写法约定里"object 组件必须 `additionalProperties: false`"没有 map 型对象（`type: object, additionalProperties: {type: string}`）的例外。按 Ruling 9 移交 M2 的 codegen handoff，本次收尾已把这条写进 [M0-P3-api-codegen-notes](../../M2-auth/handoffs/M0-P3-api-codegen-notes.md) 第 5 节。

### 4.1 修复后的复核（sonnet）

复核范围是修复轮的全部提交（`30946b0..0cd0b92`）。结论：G1–G8、spec 同步和 C1 全部处理到位，修复本身没有引入新问题。
- 复核者对照代码逐项核实，并直接运行了相关测试（`-count=1`）：G1 的中断测试和 `Unwrap()` 遍历、写法约定的 21 个反例、`TestLoadIgnoresServers`、`TestComponentNamesCoverEveryKind`、`TestBannedImports`，全部通过。
- `go build ./...`、`go vet ./...` 无问题；`go list -deps ./cmd/nerve` 中没有禁用的包。
- plan 文件中仍保留修复前的写法（`typecheck`、"Unprocessable Content"）。plan 记录的是执行时的计划，不回头修改；以 spec 为准。

## 5. 控制者裁定

| Ruling | 内容 |
|---|---|
| 1 | spec 第 3 节 15 项与上级设计的差异全部接受（入口文件 `api/openapi.yaml`；分页推迟到 M3；`module.go` 为 `New()` + `Register(mux, apiErrors)`；包名 `httpadapter`；instance 适配器只读 `buildinfo`；规则 8 覆盖 `apitest`；只新增 `bad_request`；按区域拆分命令和 CI 任务；同仓 PR 跳过；`@nerve/api-client` 现在就定名；3.1 结论写入 spec；额外的 platform/bootstrap 契约测试；`always-prefix-enum-values` + `ToCamelCaseWithInitialisms`；写法约定），每条都有理由且与已确认的决定一致 |
| 2 | 上级文档的同步（M0 设计 §1、§2、§3.2、§3.7、§4、§6.1、§6.3；v0 设计 §3.5 的 `bad_request`）放到阶段收尾统一处理 |
| 3 | 实现者分工：haiku 做转录性 Task（1、5、7、9），sonnet 做生成器/工具/CI 类 Task（2、3、4、6、8）；评审用 sonnet；整分支评审用 opus；评审(N) 与实现(N+1) 流水线并行，但不同时有两个实现者 |
| 4 | CI 失败验收（spec 4.9）由控制者推送一个只改描述、不重新生成的临时分支，确认两个任务都在生成物检查这一步失败，随后删除该远程分支 |
| 5 | M2 的 handoff（uuid type-mapping、`nullable`、`oapi-codegen/runtime` 版本、校验层归属）连同 M3/P5/P6/M8 的 handoff，在评审阶段一并创建 |
| 6 | 修复轮范围 = Important 1、2、4 + Minor 5、7、10、11、12；Important 3 只需文档说明（不改代码，目前没有设置必须通过的检查）；Minor 6、8 移交 M2；Minor 9 接受并记录 |
| 7 | problem 的 `title` 用 Go 的 `http.StatusText`（422 对应 "Unprocessable Entity"）；v0 设计 §3.5 的示例已按此更正 |
| 8 | C1：Go 测试缓存不跟踪 `server/` 之外的文件——采用方案 (a)，`make test` 用 `go test -count=1 ./...`（提交 `0cd0b92`），spec §2.8 的相关描述已更正 |
| 9 | C2 接受（与既有的 panic-后中断行为一致）；C3 移交 M2 的 codegen handoff；C4（v0 设计 §3.5 的 "Unprocessable Content"）、C5（handoff 与 M0/v0 文档同步）是控制者的文档收尾工作，即本次处理 |

## 6. 持续集成证据

- **`35672786671`**（提交 `58bff57`，Task 1–8，拆分后的任务）：成功。`server`：Generated code / Lint / Test 全部通过；`web`：Install / Generated code / Lint 全部通过。
- **`35672796385`**（临时分支 `tmp-p3-gencheck-demo`，提交 `51e5a6d`，只改接口描述不重新生成）：`server` 和 `web` 都在 "Generated code" 这一步失败，后续步骤被跳过，按预期演示了 `make gen-check` 门禁在 CI 中生效。验收后该临时分支已删除（本地和远程）。
- **`35683962686`**（提交 `0cd0b92`，修复轮和 C1 之后的分支）：成功。两个任务的全部步骤通过。

## 7. 移交事项

**本次创建并安装的 handoff（5 个）：**

| handoff | 去向 | 内容 |
|---|---|---|
| [M0-P3-api-codegen-notes](../../M2-auth/handoffs/M0-P3-api-codegen-notes.md) | M2 | 生成配置（uuid type-mapping、nullable-type、`prefer-skip-optional-pointer`）、错误映射、安全声明写法、模块入口演进、接口描述组织规则、写法检查的 map 型例外 |
| [P3-api-contract-p5-notes](../handoffs/P3-api-contract-p5-notes.md) | M0/P5 | 排除生成文件、`lint-web` 改用 turbo、保留 `@redocly/cli`、`typescript` 改 `catalog:`、包名前缀、tsconfig 兼容性 |
| [P3-api-contract-p6-notes](../handoffs/P3-api-contract-p6-notes.md) | M0/P6 | 用 `@nerve/api-client` 调用接口、Playwright 转译工作区 TS 源码、knip 配置、`e2e` 任务的 `if`/`needs` |
| [M0-P3-pagination-components](../../M3-workspace-project/handoffs/M0-P3-pagination-components.md) | M3 | 第一个列表接口加入 `Limit`/`Cursor`/`NextCursor` |
| [M0-P3-public-api-docs](../../M8-open-release/handoffs/M0-P3-public-api-docs.md) | M8 | 对外文档以 `dist` 为准、`redocly lint` 的评估、405 vs 404 |

**本次关闭的 handoff（2 个，均已改为 `status: done` 并写明处理结果）：**

| handoff | 来自 | 处理结果 |
|---|---|---|
| [P1-repo-toolchain-ci-split](../handoffs/P1-repo-toolchain-ci-split.md) | M0/P1 | 命令按区域拆分、CI 任务只跑自己区域、同仓 PR 不重复运行；本次收尾又补上了必须检查的注意事项 |
| [P2-server-platform-p3-notes](../handoffs/P2-server-platform-p3-notes.md) | M0/P2 | 生成路由挂在根路由、三处错误出口输出 problem+json、`Problem` 与契约测试一致、方法不对仍返回 404、instance 的 `app` 只依赖本模块 `domain` |
