# M0 基础骨架：Codex 对抗性评审

## 1. 评审信息与验证边界

| 项目 | 内容 |
|---|---|
| 对象 | `main`，`e1f0dc6e8e5a4c41d4bd2343db4b6c9d6b6d4f39`，`Merge M0 closeout`；与预期一致 |
| 日期 | 2026-09-22，Asia/Shanghai |
| 评审者 | Codex（GPT-6；三个独立子审使用 GPT-6 Astra）；主审负责运行证据、交叉复核、定级与汇总 |
| 环境 | macOS / Darwin 25.6.0，arm64；系统 Go 1.26.2，模块实际自动选择 Go 1.27.1；Node 24.15.0；pnpm 11.10.0；GNU Make 3.81；Docker Engine 29.7.2 / Desktop 4.87.0；Compose 5.4.0 |
| 工具 | golangci-lint 2.13.2、Turbo 2.10.11、oxfmt 0.35.0；使用已有工具和项目锁定依赖，无全局安装 |
| 材料 | 总体设计、M0 设计，P1–P6 spec / plan / review，9 个内部与 12 个外部 handoff，README、两份差异清单及指定代码配置；plan 作为历史执行记录，与最终 spec / 实现交叉核对 |
| 操作隔离 | 主目录始终为 main；违规导入、生成负例、格式缓存实验在仓库外副本中执行；仅本报告是持久改动。不 push，不改 `plane/`、`refer/`，不操作已有容器生命周期 |
| 结论口径 | “成立”说明该项交付或验收有证据，不代表所有边界输入安全；“已复现”与“读代码确认／推断”分开。通过旧测试不能反证新增发现 |

### 1.1 实际执行的检查

以下是本次执行结果，不引用原评审的“已通过”作为本次证据。`server/` 下的命令已注明工作目录；日志保留在本机 `/tmp`，报告内同时写出关键结果，避免结论依赖临时文件长期存在。

| 命令或实验 | 结果 |
|---|---|
| `git status --porcelain`、`git branch --show-current`、`git rev-parse HEAD` | 开始及写报告前均干净，main / `e1f0dc6…` |
| `make help`、`make --version` | 正常；实际为 GNU Make 3.81，recipe 可执行 |
| `pnpm install --frozen-lockfile` | 退出 0，16 个工作区；本机已有 pnpm 可直接使用，未再执行全局 `corepack enable` |
| 仓库外 `git clone --local --no-hardlinks --branch main …` 后执行 frozen install | 退出 0，实际安装 1266 个包；克隆的 `git status --porcelain` 为空 |
| `make gen-check` | 退出 0，Go、Redocly bundle、TS 类型重生成后无差异 |
| 独立克隆仅改 instance 的属性描述，再分别 `make gen-check-go` / `make gen-check-web` | 两者均退出 2，明确输出“生成物与接口描述不一致”；还原后克隆干净。证明失败门禁真实执行，不只验证正常路径 |
| `make lint` | 退出 0；Go `0 issues.`，前端 49/49，初次前端全命中缓存 |
| `TURBO_FORCE=true make lint-web` | 主审重跑退出 0，49/49，**0 cached**，约 26.5 秒 |
| `make test` | 退出 0；使用 `-count=1`，包括架构、契约、Postgres/testcontainers 集成测试 |
| `go test -race -count=1 ./...`（server） | 退出 0，包括真实数据库测试；本次执行未报告 race |
| `make build`；`TURBO_FORCE=true make build-web` | 均退出 0；后者 11/11、0 cached，约 13.8 秒。完整 build 的顺序是前端构建、清理并复制 dist、Go 编译 |
| `make knip` | 退出 0，379 项报告；这是有意的 report-only，不能称为“死代码为零” |
| `cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test` | 5 passed，4 workers，约 3.6 秒，真实临时 Postgres |
| `make e2e` | 再从正式入口执行，5 passed，约 4.3 秒；包括构建与 S1–S4 |
| 独立启动 Go 与 Vite，访问 Vite 的 `/api/v0/instance` | 200，返回 Nerve / v0 / 正确 commit，证明开发代理实际可用；仅停止本次自己启动的进程 |
| 直接 `go build`、`go run ./cmd/nerve version`、带 `-ldflags -X …buildinfo.version=9.9.9-audit` 编译 | 直接 build 有正确 commit 与默认版本；go run 的 commit 为 unknown；注入版本为 9.9.9-audit；写本报告后再编译 `/tmp/nerve-m0-dirty`，正确报告原 commit 且 modified=true |
| `/tmp/nerve-m0-runtime-probe.py` | 独立本地进程验证 CLI、不可达 DB、路由、路径编码、静态回退、请求 ID、秘密脱敏和 SIGTERM；结果见 §4、§5.7 |
| `/tmp/nerve-m0-slowbody-default.py` | 默认 5 秒头超时下，16 个缺少 1 字节正文的请求在 6 秒后仍占用连接；补正文后全部返回 200，其他健康请求仍正常 |
| `go run /tmp/nerve-m0-pgx-url-probe.go`（server） | pgx 接受含原始分号的查询参数密码，与日志泄露实验形成完整证据链 |
| 临时 server/api 副本的 15 组架构注入 | 直接违规均被挡，发现未知层绕行；详情 §5.6。另用实际 HTTP 类型别名确认并非仅空导入 |
| `/tmp/nerve-format-cache-repro.py` | 共享 CSS 修改后 Turbo 哈希不变、缓存成功，但直接 oxfmt 退出 1；见 Minor 1 |
| `node /tmp/nerve-e2e-ready-hang.mjs` | 实际导入原 server.ts，只将它启动的子程序替换为接受请求但不响应的受控进程；35003 ms 后 startNerve 仍未 settle、child 仍存活，外部清理自己创建的 child 后才出现 30000 ms 就绪错误；脚本退出 0 |
| 在临时目录复制原版 `tools/plane-schema/extract.sh` / `compose.yaml` 后执行脚本 | 退出 0；11707 行、379933 字节、110 张表；与仓库快照**逐字节相同**，SHA-256 `4080c81e8b137c19a64acb1599c66b384b32c21162fda6d77f745cb374c54e70`。事前确认无同项目容器，事后脚本自建容器清理完毕 |
| Plane 迁入树、patch、锁文件、许可证比对 | 4061 个文件中仅 7 处已登记代码差异；13 个迁入 importer 的差异全部能由已登记依赖变化解释；1477 个 lock package 条目有 integrity；LICENSE 与 Plane 原文件相同 |
| 只读查询 GitHub Actions API | 被审 SHA 对应 [run 35710931757](https://github.com/open-nerve/NerveProject/actions/runs/35710931757) 为 push / success；server、web、e2e 实际成功，不是跳过成功 |

本次主要日志：`/tmp/nerve-m0-{install,gen-check,lint,test,race,build,knip,e2e-root,e2e-make,schema,runtime-probe,slowbody-default,lint-force-root,build-force-root,negative-gen-check-go,negative-gen-check-web}.log`、`/tmp/nerve-arch-injections.json`、`/tmp/nerve-format-cache-repro.log`、`/tmp/nerve-e2e-ready-hang.log`。这些仅用于本机追溯，不作为仓库的新交付物。

### 1.2 未完成的验证与限制

- 子审最初的无缓存 lint 因 tsx IPC `listen EPERM` 失败，E2E 因其环境无法访问 Docker socket 失败；主审在可用环境重跑均成功。这两次不能算项目失败，也没有拿缓存命中替代后续执行。
- 本地 `plane/` 缺少 `5f7d927` 对象，`git diff 5f7d927 02c19e1 -- …/migrations` 退出 128。未 fetch 或改变 shallow 状态，**未独立证实两个上游提交的迁移目录完全相同**；官方固定镜像的重新提取与快照一致已经验证。
- 未在 GitHub 制造 fork PR、连续 main 推送、取消运行或失败上传；不 push 的约束下，相关行为依据 YAML / 官方语义判断。现有成功运行只证明被审提交的三个任务成功，不证明所有事件组合正确。
- 未在全新机器安装 Docker/Go/Node，未运行 `make dev-db-down/reset`，也未重建现有开发库。README 的 frozen install 在新克隆验证，其余构建与故事在当前环境验证；不声称完成裸机安装认证。
- 未重演 P3 历史上的全部 OpenAPI 3.1 假业务 schema 原型；当前描述、写法守护、实际生成与响应契约已执行。M2 请求体、JWT/PAT、River、sqlc 的运行行为尚不存在，不能用 M0 通过替它们背书。
- 未做联网 CVE 扫描、真实资源耗尽压测或完整法律合规审计。安全测试仅针对本次自建本地进程；许可判断限定为来源、LICENSE 和版权实物保留。

## 2. 总体结论

**修完列出的 Critical 后可以开始 M1。** 新发现为 **2 项 Critical、2 项 Important、3 项 Minor**；另有 1 项可选建议。需要修的是几个明确边界，不需要重写模块模板或推倒工程链。

正常路径的地基可证实：单二进制能服务内嵌前端和接口，真实 Postgres 冒烟通过；生成一致性门禁既能过正常输入，也会拒绝未提交的生成变化；Plane 迁入与 schema 快照可独立核对。配置、组合根、端口与适配器没有发现必须返工的结构性问题。

但“敏感配置可安全写日志”与“HTTP 有超时保护”的信心过高：有效 pgx URL 的密码能被直接记录，未认证客户端可以让请求体读取长期占用连接。架构测试和 E2E 失败路径另各有一个可穿过正常测试的缺口。CI 全绿是事实，不能据此宣布这些边界可靠。

## 3. 各领域评估

评分衡量本阶段交付的可靠程度，不是对未实现业务功能的打分；5 为本次核查没有重要缺口，1 为需要重做。

| 领域 | 评分 | 主要依据 | 最大风险 |
|---|---:|---|---|
| 范围闭环 | 4 | P1–P6、S1–S4 与实体交付齐全；9 个内部 handoff 有处理结果 | 交付存在被误读为承诺的边界均成立 |
| 架构 | 4 | 当前依赖干净、组合明确，直接违规与生产传递依赖测试有效 | 未识别模块目录绕过 domain/app 纯净性 |
| 逻辑完整性 | 3 | 启动、DB 就绪、常规 SIGTERM、迁移空集语义可复现 | E2E readiness 的单次请求无期限 |
| 安全 | 2 | prod 无数据库默认密码、dev 回环、Problem 与请求 ID 基础可靠 | 密码日志泄露与慢请求体资源占用 |
| 正确性 | 4 | 无缓存测试、race、生成负例、浏览器故事均通过 | 边界输入未被原测试覆盖 |
| 代码质量 | 4 | 职责小、无空 Deps 框架，无需拆除的运行时抽象 | 测试辅助代码的失败分支比正常路径薄弱 |
| 测试 | 3 | 真实 DB、编译负例、HTTP 契约、浏览器同源断言有实际价值 | 架构与缓存假绿、fixture 超时漏口 |
| 工程与 CI | 4 | 固定版本、Make 3.81、完整 build/e2e 和真实 CI 成功 | 已知同仓 PR skip、main pending 取消、格式缓存输入遗漏 |
| 文档 | 4 | handoff 具体，范围与历史裁定清楚 | River 迁移所有权冲突，少数“已修”超过实际保障 |

## 4. 新发现

### Critical 1：合法数据库 URL 中的原始分号可绕过脱敏，把密码写入启动日志

- **类别**：安全 / Bug。
- **位置**：`server/internal/platform/config/redact.go:28`、`:36`；`server/internal/platform/config/config.go:66`；`server/internal/bootstrap/commands.go:28`。
- **可信度**：已复现，另一个子审独立确认 pgx 能解析该 URL。
- **证据**（摘录 7 行）：

```go
q := u.Query()
masked := false
// … only keys present in q can set masked …
if masked {
    u.RawQuery = q.Encode()
}
return u.Redacted()
```

`URL.Query()` 不返回解析错误；含未编码分号的参数被它丢弃，`masked` 保持 false，原 `RawQuery` 原样留下。pgx 的连接串解析却接受该密码。用纯假密码设置 `NERVE_DATABASE__URL='postgres://probe@127.0.0.1:1/db?password=FAKE_QUERY_SECRET;stillsecret&sslmode=disable'`，以 prod 配置在临时回环端口启动 `bin/nerve serve`：`configuration loaded` 的 `config.database.url` 明文包含 `FAKE_QUERY_SECRET;stillsecret`。无需成功连接数据库，因为日志先于 `newApp` 写出。调用当前依赖的 `pgxpool.ParseConfig` 得到 `PASSWORD="FAKE_QUERY_SECRET;stillsecret"`，不是仅非法配置会泄露；`pass%77ord` 也可复现。`%3B` 编码形式能正确打码，构成对照。

- **影响**：使用合法但带原始分号的查询密码时，凭据流入标准日志及其集中收集系统。能读取日志的人无需有配置权限即可取得密码。与 P2 已修复的 `sslpassword`、单独记录 `DatabaseConfig` 不同，是解析器语义不一致的新入口。
- **处理建议（权衡）**：
  1. **推荐，M1 开始前**：显式 `url.ParseQuery(u.RawQuery)`，有任何解析错误就整体打码，再处理可解析查询；增加分号、编码键、错误转义的回归。改动小，保留正常地址的诊断价值，代价是某些 pgx 可用但 URL 标准解析失败的配置只显示掩码。
  2. 彻底不记录连接串，仅记录单独的安全字段或固定掩码。攻击面最小，代价是日志少一些定位信息；不要为此再发明一套连接串解析器。
  3. 不处理，仅要求运维编码分号。零代码成本，但安全边界依赖输入规范与部署人员，现有代码仍把未经证明已脱敏的原值写出；不接受。

### Critical 2：完整请求头加未发送的正文，能绕过现有超时长期占用 HTTP 连接

- **类别**：安全 / 逻辑。
- **位置**：`server/internal/platform/httpserver/server.go:30`；`server/internal/platform/config/config.go:27`。
- **可信度**：连接占用已复现；进一步大规模资源耗尽为推断，未执行破坏性压测。
- **证据**（摘录 5 行）：

```go
srv: &http.Server{
    Handler: middleware(h, logger),
    ReadHeaderTimeout: cfg.ReadHeaderTimeout,
    IdleTimeout: idleTimeout,
},
```

没有正文读取期限。对独立测试进程，用 HTTP/1.1 默认 keep-alive 发送 `GET /healthz`，头含 `Host: localhost`、`Content-Length: 1`，结束头后不发送正文；**不要加 `Connection: close`**。默认请求头超时 5 秒，等待 6 秒，16 个测试连接全部仍打开且没有响应；另一个正常 `/healthz` 仍为 200。给每个连接补 `x` 后，16 个立即返回 200，关闭连接并 SIGTERM 后进程正常退出。

本机实际 Go 1.27.1 的 `net/http/server.go` 在只有 `ReadHeaderTimeout` 时，读完头会把读取 deadline 设回零；响应完成前可能排空小请求体。`IdleTimeout` 管的是请求间空闲，不能覆盖这里。GET 没有业务正文也能触发，因此不必等 M2 的 POST 或 M5 上传才成立。

- **影响**：直接暴露 Go 服务时，未认证远程客户端可以维持大量未完成请求，消耗连接、文件描述符和 goroutine。未证明 16 个连接造成服务不可用；风险是没有期限的资源占用可以放大。总体设计允许单二进制部署，不能假设始终有正确限制的代理代为保护。
- **处理建议（权衡）**：
  1. **推荐，M1 开始前**：给当前普通请求设置有限 `ReadTimeout`，结合实际响应设置写入期限，补一个完整头但缺正文的黑盒回归。配置与测试改动小，无需引入中间件框架；M5 大文件上传时再按路由调整期限，避免统一短超时伤害上传。
  2. 明确强制所有部署经能限制正文读取的代理，并验证该边界。能集中治理，但改变现有部署前提且依赖运维配置，不能只写一句“建议反代”。
  3. 不处理，留到有带体业务接口时再做。当前 GET 已可触发，原有 2 分钟 idle timeout 不解决它；不接受。请求体大小限制与 413 仍按既有 handoff 在 M2 落实，它们不替代读取超时。

### Important 1：未知模块子目录能把基础设施类型带进 domain，而架构测试仍通过

- **类别**：架构 / 测试。
- **位置**：`server/internal/archtest/rules_test.go:121`、`:144`、`:157`。
- **可信度**：已复现。
- **证据**（摘录 6 行）：

```go
fromRank, fromKnown := layerRank(fl)
toRank, toKnown := layerRank(tl)
return fromKnown && toKnown && toRank > fromRank
// innerLayersArePure allows any in-module target:
_, _, inModule := moduleOf(to)
return !inModule && !within(r, "internal/shared")
```

在临时副本增加 `instance/transport`，其中 `type Request = http.Request`；再在 `instance/domain` 声明 `type InfrastructureRequest = transport.Request`。`go test -count=1 ./internal/archtest ./internal/modules/instance/domain` 退出 0。`layerRank` 不认识 transport，方向规则不判违规，纯净性规则又把模块内目标整体让给方向规则。`domain → instance/helper → net/http` 同样通过。

- **影响**：M2 起照模板扩展 service / repository / transport 目录时，领域公开类型可能已经绑定 HTTP/数据库而仍显示架构绿色。当前实际目录没有这种污染，因此不是“现有模块必须返工”。这也不同于已移交 M2 的 shared 绕行：不需 shared 即可发生。
- **处理建议（权衡）**：
  1. **推荐，最迟 M2 新增第二个模块前**：domain/app 的本模块导入仅允许明确的同层或内层目标，未知层默认拒绝；加入这条真实导入链反例。改动小，能与既定 shared/sqlc 守护同批落地。
  2. 对所有模块目录建立全局结构白名单。边界更强，但特殊适配器布局要不断维护名单，当前收益有限。
  3. 不处理，依靠代码评审识别新目录。当前运行无影响，却让宣称自动守护的边界继续可绕过；多模块开发后不推荐。

### Important 2：E2E readiness 的 fetch 没有期限，30 秒就绪预算不能包住单次请求

- **类别**：逻辑 / 测试。
- **位置**：`e2e/fixtures/server.ts:63`、`:102`、`:122`；`e2e/fixtures/test.ts:37`。
- **可信度**：已复现超出就绪期限；Playwright 终止 worker 后是否最终遗留孤儿未实测。
- **证据**（摘录 5 行）：

```ts
const ready = await fetch(readyzUrl).then(
  (res) => res.ok, () => false
);
if (Date.now() >= deadline) {
  throw new Error(`${readyzUrl} did not answer 200 within ${readyTimeoutMs} ms`);
```

外层 deadline 只在 `fetch` 返回后检查；没有 AbortSignal 或单次超时。若 readiness 接受连接但不发回响应头，30 秒不是这个函数的硬上限。把 worker fixture 改为 70 秒，修好了原来“总预算先耗尽”的问题，却没有约束 await 本身。正常 `/readyz` 的数据库 ping 有期限，故健康路径和一直返回 503 的旧反证均不能发现本问题。

实际运行 `node /tmp/nerve-e2e-ready-hang.mjs`：通过 Node 的 `syncBuiltinESMExports` 仅替换 spawn 所执行的二进制，直接导入未修改的 `e2e/fixtures/server.ts`；受控 HTTP child 接受请求但不回响应。35003 ms 后记录 `settled:false, exitCode:null, signalCode:null`。实验 finally 从外部终止这个 child，真实 `startNerve` 才抛出带日志路径、原因是“did not answer 200 within 30000 ms”的错误。没有修改仓库文件、没有使用 Docker，脚本正常退出，未遗留其受控 child。这证明测试骨架面对故障程序的期限失效，不声称当前正常 readyz 会自行挂死。

- **影响**：未来探针回归、监听竞争遇到挂起服务或连接迟迟不返回时，错误由 Playwright 外层超时接管，`startNerve` 的 catch/SIGKILL 与日志路径说明可能来不及执行；遗留子进程风险是控制流推断，不能从正常 E2E 全绿断言已排除。
- **处理建议（权衡）**：
  1. **推荐，并入 M1，最迟 M2 扩展 fixture 前**：每次 fetch 用剩余就绪预算限制 AbortSignal；检查剩余时间并在失败时终止、等待自己创建的 child 退出。只需局部改动与一个接受连接却不响应的反例，不需要进程管理框架。
  2. 用独立就绪 watchdog 同时取消请求并清理 child。上限明确，但比直接给 fetch 期限多一套状态/计时器，当前不优先。
  3. 不处理，依赖 Node 默认网络超时或 Playwright 70 秒预算。可维持正常测试，但错误定位与清理承诺继续不成立；不推荐。

### Minor 1：共享 Tailwind CSS 不在格式检查缓存键内，缓存可重放错误的成功结果

- **类别**：工程 / 测试。
- **位置**：`turbo.json:3`、`:36`；`.oxfmtrc.json:5`。
- **可信度**：已复现，使用项目真实 Turbo / oxfmt 配置的仓库外最小工作区。
- **证据**（摘录 5 行）：

```json
"globalDependencies": [".oxfmtrc.json", ".oxlintrc.json"],
"check:format": {
  "inputs": ["$TURBO_DEFAULT$"],
  "outputs": []
}
```

格式配置的 Tailwind 排序读取 `web/packages/tailwind-config/index.css`，该文件不在消费包自身输入中。测试 TSX 含 `className="audit-order flex"`：CSS 只有 Tailwind import 时，强制格式检查成功，哈希为 `c0fd0a96d66fc88b`；仅向共享 CSS 加 `@utility audit-order { padding: 1px; }`，Turbo 仍同一哈希、cache hit、退出 0；直接 `oxfmt --check probe.tsx` 退出 1，要求 `flex audit-order`。最小工作区仅额外禁用 pnpm 自动依赖补装，未改变任务缓存语义；真实仓库 dry graph 也确认此任务无依赖任务及共享 CSS 输入。

- **影响**：M1 改样式时，本地和共享 worktree 缓存可能假绿，冷 CI 才发现。CI 当前不跨运行保存 Turbo 缓存，影响有限；不是声称构建产物错误。
- **处理建议（权衡）**：
  1. **推荐，并入 M1**：共享 CSS 加入 `globalDependencies`，后续拆分本地 CSS import 时同步补输入。一处配置即可，代价是样式变更使更多缓存失效。
  2. 仅在 `check:format.inputs` 声明 `$TURBO_ROOT$/web/packages/tailwind-config/index.css` 及本地依赖。更精确，但需验证 root glob 和传递输入。
  3. 不处理。冷 CI 仍能拦截，代码风险低，但反复出现“本地绿、CI 红”；M1 集中改样式前修复成本更低。

### Minor 2：River 迁移由谁管理，总体设计与 M2 handoff 给出相反答案

- **类别**：文档 / 架构。
- **位置**：`docs/v0/v0-design.md:272`；`docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md:26`。
- **可信度**：读文档确认；当前无 River 实现。
- **证据**：总体设计说 River 表“由 River 自己的迁移管理”；handoff 要求“用 goose 的 SQL 迁移建立（而不是 River 的迁移命令）”。
- **影响**：M2 不同实施者会设计出一条或两条迁移链，连带影响 readiness、模板库、升级与回滚。现在是局部文档冲突，实现后才统一会返工。
- **处理建议（权衡）**：
  1. **推荐，M2 设计前**：统一为 handoff 的单 goose 链，并明确所选 River 版本的上游 SQL 来源、升级策略。保持现有平台简单；须核实所选版本支持的维护方式，不能自行仿造内部表。
  2. 统一为 River 自带 migrator，同时规定它与 goose 的执行/状态/模板库顺序。尊重独立上游生命周期，但增加两条迁移链协调成本。
  3. 不处理。M1 不受影响，M2 分工则容易冲突；只改文档的成本远低于之后改运行链。此处不主张提前接入 River。

### Minor 3：main 的运行中任务不取消，但排队任务仍会被后续推送替换

- **类别**：工程 / 文档。
- **位置**：`.github/workflows/ci.yml:7`；`docs/v0/M0-foundation/reviews/P1-repo-toolchain-review.md:56`。
- **可信度**：配置与官方语义确认；未远程制造连续推送。
- **证据**（摘录 3 行）：

```yaml
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}
```

P1 修复结论为“main 分支的每次运行都保留”。当 A 运行、B 排队、C 到来，同组默认只保留一个 pending，B 会被 C 替换；`cancel-in-progress: false` 只保护 A。该语义见 [GitHub concurrency 文档](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#concurrency)。这是旧修复未覆盖的 pending 情形，区别于已知 PR skip。

- **影响**：密集 main 推送时，中间提交可能没有完整检查结果；最新 main 仍会得到验证。不是当前被审 SHA 缺失 CI，因此定为 Minor。
- **处理建议（权衡）**：
  1. **推荐，并入 M1**：main 的并发组加入唯一 run id，其他分支保留按 ref 取消。配置小，符合“每次保留”，代价是并行 runner 消耗更多。
  2. 使用有界扩容队列并为 main 单独设置其兼容的取消策略。可串行保留更多运行，但有队列长度与等待时间；不能直接把允许多个 pending 的队列与非 main 的 `cancel-in-progress: true` 混用。
  3. 不改代码，明确只要求“当前运行不中断、最终 main 会检查”，修正文档。不影响最新版本保障，是可接受的产品取舍，但不能继续声称每次都保留。

### 建议 1：第二个 API 模块出现时，把路径所有权约定转成小型守护

- **类别**：测试 / 架构；**位置**：`server/internal/platform/httpserver/apitest/rules_test.go:43`，M2 codegen handoff 的路径归属约定。
- **可信度**：局部实验已复现，**不是当前功能缺陷**。在临时副本增加第二个模块文件，同一路径仅声明 POST，当前路径列表测试仍通过；没有生成/接线完整第二模块，不能声称整条 CI 都可绕过。
- **影响**：现有检查偏向路径是否出现，不能替代“一个路径属于一个模块文件”的所有权审查，当前单模块没有冲突。
- **权衡**：推荐在 M2 以少量断言检查路径唯一归属和根 `$ref`；另一个方案是继续人工核对（当前可接受，零新增代码）；不建议现在引入新的 API 注册框架或第二套 schema 工具。收益是减少多人改根描述时的遗漏，代价仅为维护一条既有约定。

## 5. 闭环核对表

### 5.1 M0-design §0.2：7 项范围

| 范围 | 判断 | 本次证据 |
|---|---|---|
| 仓库、工具链、开发数据库与 Makefile | 成立 | frozen install、新克隆安装、Make 3.81、实际工具运行；Compose 开发库回环绑定，未违反约束重建它 |
| 服务端 config / logging / postgres / migrator / HTTP / bootstrap / CLI | 部分成立 | 正常生命周期与集成测试通过；配置安全日志和 HTTP 读取期限分别有 Critical 1、2 |
| OpenAPI 契约与 instance 试点 | 成立 | 实际重生成无差异，遗漏生成的负例失败；GET、404、TS 客户端与响应契约通过 |
| 架构测试与 depguard | 部分成立 | 直接违规和生产传递依赖受控；未知模块层绕行 Important 1 |
| Plane schema 快照 | 成立 | 原脚本重新提取逐字节一致；44 张目标 Plane 表全部存在，完整快照 110 张表 |
| Plane 前端迁入、构建、内嵌 | 成立 | 来源树比对、11/11 无缓存构建、实际开发代理、内嵌 S2 通过；格式缓存缺口另见 Minor 1 |
| E2E 骨架和 CI | 部分成立 | 4 个故事、5 个测试和被审提交三个 CI job 成功；故障就绪期限 Important 2；事件与队列限制分别见已知 K16、Minor 3 |

范围外的业务表、认证、River/sqlc、前端瘦身与业务对接未被偷渡进 M0；也不把它们尚不存在报成缺陷。

### 5.2 M0-design §8：逐项交付物

| Phase / 交付物 | 判断 | 证据 |
|---|---|---|
| P1 根 package / workspace / Node 版本 / editorconfig | 成立 | 文件存在，16 个工作区 frozen install 成功，工具版本实测 |
| P1 server 与 tools 两份 go.mod | 成立 | 两份 `go 1.27` / `toolchain go1.27.1`，实际生成、测试、编译可用 |
| P1 Makefile 骨架 | 成立 | GNU Make 3.81 下 help、gen、lint、test、build、knip、e2e 均实际执行 |
| P1 Postgres 18 开发 Compose | 成立 | `deploy/compose.dev.yaml:6` 为 18.6、端口回环、PG18 数据卷路径正确；已有 dev-db 保持运行 |
| P1 CI 骨架 | 成立 | 已演进为 server/web/e2e；被审 SHA 的实际 run 成功；并发保留措辞需 Minor 3 修正 |
| P1 gitignore | 成立 | 依赖、bin、embed dist、浏览器产物被排除；生成源码仍被跟踪 |
| P1 buildinfo | 成立 | 默认／注入版本、直接 build 的 VCS 信息、go run 的 unknown 语义有实测 |
| P2 config | 部分成立 | 层叠、未知键、环境与时长测试通过；Critical 1 推翻完整脱敏保障 |
| P2 logging | 成立 | JSON/text 与级别检查、结构化访问日志和 panic 测试通过；输入脱敏漏洞位于 config |
| P2 postgres / migrator / pgtest | 成立 | 真实 DB 的 up/status/down、部分失败、空迁移、不可达、强制删测试库覆盖执行；未声称支持并发迁移 |
| P2 HTTP / 3 中间件 / Problem / 探针 / 优雅停机 | 部分成立 | 响应、请求 ID、panic、drain/超时关闭测试通过；缺正文读取期限，Critical 2 |
| P2 bootstrap | 成立 | `server/internal/bootstrap/app.go:30` 之后显式组装，`:47` 注册模块、`:50` 挂 webui，失败与结束释放 pool |
| P2 三组 CLI | 成立 | serve、version、migrate 三子命令可运行；拼错命令/多余参数/非法配置为非零退出，空迁移语义明确 |
| P2 archtest / golangci depguard | 部分成立 | 全量 lint/test 和反向注入有效；Important 1 补未知目标层规则 |
| P3 OpenAPI 3.1 链路结论 | 成立（当前采用的写法） | 当前真实描述通过 Go/Redocly/TS；已知不支持的写法有守护。历史所有假接口原型未全部重放 |
| P3 api 目录 / Redocly | 成立 | 根入口、common、modules、dist、bundle 配置与实际生成链一致 |
| P3 分模块 oapi-codegen 与公共包 | 成立 | 模块 strict server + model，公共包 models；生产依赖图没有 kin-openapi，违规注入会失败 |
| P3 instance 完整实现与接线 | 成立 | domain/app/两个 adapter/module 各有职责，handler/bootstrap/契约测试和实际 GET 通过 |
| P3 `@nerve/api-client` | 成立 | 14 行 openapi-fetch 工厂，泛型来自生成 schema；类型负例、E2E S3 实际调用，无数据转换兼容层 |
| P3 `make gen` / `make gen-check` | 成立 | 全链正例通过；隔离克隆的 Go 与 web 负例分别失败，未依赖测试缓存 |
| P4 Compose | 成立 | 两个镜像均 tag@digest，无主机端口、tmpfs，未引入 Redis 等常驻依赖 |
| P4 extract 脚本 | 成立 | 在确认无同名项目后于临时目录运行原脚本成功，自建项目自动清理；主快照未写入 |
| P4 README | 成立 | 版本、SHA、字节、行、表数与重提取一致；上游两提交迁移目录对比的本次证据限制见 §1.2 |
| P4 SQL 快照 | 成立 | 110 表，44 个目标 Plane 表缺失集合为空，字节完全相等 |
| P4 `make plane-schema` | 成立（接线及原脚本执行） | `Makefile:132` 调同一脚本；为不改主快照，未在主目录运行该写入入口 |
| P5 `web/` 原样迁入 | 成立 | 4060 前端文件加 1 个 patch 对比，7 处差异均已登记；不评原样 Plane 业务代码质量 |
| P5 workspace / turbo | 部分成立 | 工作区、strict env、编译依赖与三类检查运行成功；格式外部输入漏哈希，Minor 1 |
| P5 Vite API 代理 | 成立 | `web/apps/web/vite.config.ts:36` 与实测 Vite → Go 200 对应，回环监听 |
| P5 webui | 成立 | 真实首页/深链/资产与路径对抗测试通过；CSP 等为既有 M2 handoff |
| P5 单二进制 build | 成立 | `Makefile:117` 明确先 build-web、再清理/复制、再 Go build；独立二进制完成冒烟 |
| P5 迁入改动登记 | 成立 | 六处 lint 上限和一处代理，P6 锁文件变动也能解释，无未登记源码差异 |
| P6 Playwright 项目 | 成立 | 配置、类型/lint/format 检查、5 个测试、4 worker 实际运行 |
| P6 server fixture | 部分成立 | 每 worker 自己的进程、日志和退出检查存在；Important 2 |
| P6 db fixture | 成立 | 一容器、多 worker 副本，SQL identifier 转义，模板完成后再复制；正常清理成功 |
| P6 api fixture | 成立 | 实际 generated typed client，S3 使用它，S4 使用 request 是明确裁定 |
| P6 test fixture 接线 | 成立 | `e2e/fixtures/test.ts:25` 起 worker 生命周期与 baseURL/api 依赖明确；70 秒预算确已写入 |
| P6 global setup | 成立（正常路径） | `e2e/global-setup.ts:9` 启库、迁移模板、返回 teardown；非正常容器回收依赖 Ryuk，未遍历所有杀进程时序 |
| P6 S1–S4 | 成立 | 本节下一表；M0 的 UI 加载验收不等同业务验收 |
| P6 CI e2e job | 成立 | `needs` 和条件一致，独立完整 build，实际 run 成功；失败上传条件静态确认，未重新远程演示 |
| P6 knip 配置 | 成立 | 379 项只报告，类型测试作为入口；M1 门禁的后续工作仍存在 |

### 5.3 M0-design §9：S1–S4

| 故事 | 判断 | 实测断言及界限 |
|---|---|---|
| S1 服务就绪 | 成立 | healthz / readyz 200；`migrate status` 为 no migrations；用本 worker 数据库的 `pg_stat_activity` 查到 application_name=nerve，避免拿空迁移命令冒充连接证明 |
| S2 首页与深层路径 | 成立 | 两个浏览器测试；静态资源加载、所有请求同源、深层 HTML 与首页逐字节相同。Plane 启动错误页是 M0 明示结果，未声称登录可用 |
| S3 实例信息 | 成立 | 用生成 TS client 得到 v0、构建版本、40 位 commit；主审另外核对实际 commit 就是被审 SHA |
| S4 未知 API | 成立 | 404 与 application/problem+json、响应内容通过；没有回退到 SPA |

### 5.4 M0-design §10：7 条完成标准

| 标准 | 判断 | 证据与解释 |
|---|---|---|
| P1–P6 完成且各有 spec/plan/review | 部分成立 | 18 份材料齐全，交付也齐全；“全部完成”应受 Critical 1/2 与具体边界缺口限定，不能仅由文档存在推出 |
| 全部门禁通过 | 成立 | 真实被审 CI 的 server/web/e2e 与本地无缓存执行通过；这是运行事实，门禁覆盖不足另行列出 |
| 单二进制加 Postgres 完成 S1–S4 | 成立 | make e2e 的 5 个测试实证 |
| §7 调整同步总体设计与差异清单 | 成立 | 五类调整逐项核对见下表；River 所有权冲突是另一处文档问题 |
| 前端最小改动已登记 | 成立 | 完整来源树与锁文件对比支持，不只按提交信息判断 |
| 内部 handoff 无 open，后续事项已移交 | 部分成立 | 9 份确为 done、12 份外部 handoff 存在；旧事项大部分具体，但 P2→P6 的期限未兑现，新发现尚需实施者按本报告接收；未擅改 handoff |
| 总体设计 M0 状态为已完成 | 成立（文本事实） | `docs/v0/v0-design.md:666`；该状态文字不推翻本次“先修 Critical”的裁定 |

| M0 §7 调整 | 判断 | 核查 |
|---|---|---|
| 业务表按所属 M 建，不在 M0 全量建表 | 成立 | v0 §5.6 / §9.3、plane-diff 的建表说明一致；migrations 为空 |
| River / sqlc 首次接入 M2 | 成立 | v0 §9.2 M0 明确不接入、M2 首任务；当前依赖和代码没有预建空壳 |
| 前端警告按迁入基线 | 成立 | v0 §7.6 与包脚本实测总基线 1005 相符；“只降不升”仍依赖评审，见 K20 |
| Go 架构测试代替 go-arch-lint | 成立 | v0 §6.3 与实际 archtest / depguard 一致，未保留第二套空工具链 |
| Go/Node/TypeScript/UUID/OpenAPI 版本与写法调整 | 成立 | Go 1.27 标准库 uuid、Node 24、TS 5.8.3、OpenAPI 3.1 当前链路真实运行；工具兼容限制有 P3/M2 交接 |

### 5.5 9 份内部 handoff

均位于 `docs/v0/M0-foundation/handoffs/`；下表核对“处理结果”，不是只读取 status。

| 文件 | 判断 | 实现证据 |
|---|---|---|
| `P1-repo-toolchain-ci-split.md` | 成立 | `Makefile:68`、`:85`、`:96` 按 Go/web 拆分；CI 对应调用，if 在三个 job 中一致；skip 风险没有自动消失 |
| `P1-repo-toolchain-go-db-notes.md` | 成立 | 两份 go.mod 的工具链；`README.md:31`、`server/configs/config.dev.yaml:8` 的端口与 URL 同步说明 |
| `P2-server-platform-p3-notes.md` | 成立 | `server/internal/bootstrap/app.go:47` 根 mux 注册；`server/internal/modules/instance/adapter/http/handler.go:19` 三个 APIErrors 出口；平台契约及 bootstrap unknown/wrong-method 回归实际通过；app 仅依赖 domain |
| `P2-server-platform-p5-webui-mount.md` | 成立 | `server/internal/bootstrap/app.go:50` 不带方法的 `/`；`server/internal/platform/webui/handler.go:43` 处理 GET/HEAD 与其他方法；`web.enabled` 由处理结果明确裁定删除，未漏做 |
| `P2-server-platform-p6-e2e-notes.md` | 部分成立 | `e2e/fixtures/server.ts:46` 端口 env、`:71` 环境隔离，模板副本和 SIGTERM 都实现；“最多 30 秒”被 Important 2 推翻 |
| `P3-api-contract-p5-notes.md` | 成立 | 两个 ox 配置排除生成物、根 Redocly、TS catalog、api-client 三类检查实际存在；不继承上游 tsconfig 是已裁定选择 |
| `P3-api-contract-p6-notes.md` | 成立 | `e2e/fixtures/api.ts:7`、S3/S4、knip 类型测试入口、CI needs/if、工作区 TS 导入均由执行验证 |
| `P4-plane-schema-p5-notes.md` | 成立 | `docs/v0/frontend-changes.md:15` 完整来源 SHA 与实际 Plane HEAD 一致；4061 个文件对比及锁文件差异登记闭环 |
| `P5-web-import-p6-notes.md` | 成立 | CI 独立 build、版本注入、e2e 三项检查、直接执行 Playwright、S2 分类/同源、产物 ignore 均存在；缓存不跨 job 保存，未假称复证历史耗时 |

### 5.6 架构对抗试验完整结果

所有注入在临时副本，逐个运行 `go test -count=1 ./internal/archtest`，不污染主目录。

| 注入路径 | 实测 | 判断 |
|---|---|---|
| domain → net/http | 失败，规则 2 | 有效守护 |
| domain → 同模块 app | 失败，规则 1 | 有效守护 |
| probe/adapter → instance/domain | 失败，规则 3 | 模块隔离有效 |
| platform/config → instance/domain | 失败，规则 4/5 | 平台不能绕入业务 |
| cmd/nerve → instance/domain | 失败，规则 5 | 组合根边界有效 |
| 非 HTTP adapter → 模块 HTTP gen | 失败，规则 6 | 生成类型限制有效 |
| platform/config → platform/buildinfo | 失败，规则 7 | 平台包间规则有效 |
| 生产 adapter → apitest | 失败，规则 8 与传递依赖 | 测试契约库不进生产 |
| 公共 apigen → google/uuid | 失败，显示完整依赖链 | apigen 不是禁库绕行入口 |
| `_test.go` → 别模块 domain | 通过 | 测试文件不在生产图，属合理范围，不是生产依赖漏洞 |
| bootstrap → instance/app | 通过 | bootstrap 本就是组合根，设计允许 |
| cmd/nerve → platform/postgres | 通过 | 规则无 cmd 平台白名单，当前没有此依赖；可人工职责约束，不列缺陷 |
| adapter/buildinfo → 公共 apigen | 通过 | 规则 6 针对模块 HTTP gen，外层用公共 DTO 不违反内层纯净性 |
| domain → 同模块 helper → net/http | 通过 | Important 1，进一步 HTTP 类型别名试验也通过 |
| domain → shared/auditprobe → net/http | 通过 | 已知 M2 shared 守护缺口，记入 K6，不重复报新发现 |

### 5.7 其余起点假设的核查结果

| 主题 | 核查结果与证据 |
|---|---|
| 配置优先级 | `server/internal/platform/config/load_test.go:43` 的分层测试与实际运行通过：内嵌 base/env → 外置 base/env → dev 专用 config.local.yaml → 环境变量依次覆盖；不支持的 profile、未知键、缺 DB、数值越界、不带单位的 duration 均非零退出并指出键；未发现层次反转 |
| dev/test/prod | dev 服务和 DB 仅回环；test/prod 需要显式 URL；prod 关闭 auto_migrate、无内置生产口令。缺少环境选择的实际默认行为与 README 相符，不是自动把 dev 密码带入 prod |
| 敏感错误与日志 | 普通 URL userinfo、编码查询密码、sslpassword、KV/opaque 地址的样本已掩码；非法 URL 的 pgx 错误样本未见测试密码，readyz 仅回通用 not-ready；Critical 1 是发现的例外。访问日志不记录 query/body；panic 值与 stack 会进入服务器日志，M2 不得把令牌作为 panic/error 内容 |
| HTTP 错误与契约 | 当前 GET 与所有平台 Problem 响应有契约测试；生成器三处错误回调接到 APIErrors。API 错方法返回 404 是已有裁定；非 API 的 webui 405 是文本，并不声称所有静态错误也套 API Problem。真实请求体解码/值校验在 M2 首接口补验 |
| 非规范路径 | `/api`、`/api/v0//instance` 返回本地 307；`//evil.com` 的 Location 为 `/evil.com`，不是外部站点；编码斜杠/反斜杠样本无外部 Location；编码 `..` 为 400 或 API Problem 404。未发现开放重定向 |
| 静态文件 | `/.gitkeep`、`/.git/config` 得到 SPA 而不是隐藏文件；缺 assets 返回 404，不回 HTML；`/index.html` 的 301 指向 `./`；assets 长缓存、index/public no-cache 与单测一致。ServeFileFS 的扩展名类型/嗅探行为仍存在，安全头是 K22 |
| 请求 ID / 注入 | 合法 token 保留，含空格或 129 字节值替换 UUID；middleware 的白名单及 slog 结构化编码有测试，未发现换行伪造日志记录的路径 |
| HTTP 容量边界 | 没有自定义 MaxHeaderBytes，继承 net/http 默认限制，不是无限头大小；只有限头读取和连接空闲期限，正文读取见 Critical 2。M2 的 MaxBytes/413 与 M5 上传策略不能提前假称已有 |
| 迁移生命周期 | M0 nil provider 时 up/down/status 均成功，不访问 DB；实测 URL 不可达也这样。serve 仍通过 pool.Ping 判就绪，故 readyz 503；首迁移后的 CheckUpToDate 和真实 goose 行为由集成测试验证 |
| 停机和异常启动 | `server/internal/platform/httpserver/server_test.go:65` / `:112` 覆盖排空与超时强关，`:137` 覆盖端口占用；实际独立 serve 收 SIGTERM 退出 0。HTTP 结束后 bootstrap 再关 pool；River 后新增顺序与上限须 M2 重审，不能靠当前无后台任务场景推断 |
| 全局状态 / 抽象 | 生产无 init 副作用或默认 mux/logger 服务定位；包级 embed、ldflags 版本、错误哨兵合理。InfoSource 在测试中真实隔离构建信息，不是仅为未来预留的空接口；New() 无空 Deps，模块入口无需返工 |
| 生成物管理 | Makefile 对受控 Go gen、api/dist、TS 文件使用 git status，而非只查工作树 diff，包含未跟踪/暂存状态；删除旧 Go gen 避免陈旧生成文件。负例实测会失败；不把被 .gitignore 的前端构建产物当需提交的 API 生成物 |
| worker 与模板库 | 每次运行一临时容器、模板迁移结束后按 workerIndex 克隆，NERVE_* 调用环境剥离后注入 test/独库；数据库查询连接 finally 关闭。正常执行结束资源回收成立；异常就绪请求见 Important 2 |
| CI 事件 | push 无 branch/tag 过滤；fork PR 运行，同仓 PR skip；三个任务的 if 一致，e2e 有 needs。只读确认权限为 `contents: read`，无 pull_request_target；未远程制造全部事件组合 |
| CI 缓存 | pnpm store 按 OS/lock 哈希，Go 缓存覆盖 go.sum；Turbo 不跨 job 保存，Playwright 不经 Turbo；用 force 排除了本轮仅靠旧缓存的疑问。Minor 1 是本地输入不完整的新问题 |
| 供应链 | schema 两镜像固定 digest；dev/e2e postgres:18.6、Ryuk:0.14.0 与 Action major tags 不是不可变 pin；golangci 安装脚本按版本标签。pnpm packageManager 带哈希，allowBuilds/overrides/patches 显式，迁入锁文件 integrity 完整。不能由这些推出“无供应链风险” |
| AGPL 来源 | 根 LICENSE 与 Plane LICENSE.txt 同字节，迁入文件 blob 对比证明原版权头保留，来源 SHA/改动有登记；未发现删除声明的新问题。公开运行修改版本的对应源代码提供方式与发布资料在 M8 落实，源码实物保留不等于已完成所有发布义务 |

## 6. 已知事项复核

以下不计入新发现数量。为便于行动清单引用赋予 K 编号；同源风险放在同一行，判断给出处理时点。

| 编号 / 来源 | 已有裁定与本次判断 |
|---|---|
| K1：P1 review 的工具与 Actions 固定方式 | **同意阶段性接受**。标签/version 可复现性有限但无本次已知入侵证据；M8 发布前固定 Actions SHA、评估工具校验与镜像 digest 更新流程。contents:read 降低风险但不解决构建供应链本身 |
| K2：P1 工具缓存收益低、占位目录、历史提交署名 | **同意接受**。当前命令实跑成本可接受，没必要为小额耗时增加缓存交接层；不回写历史提交，空目录占位不等于运行时空抽象 |
| K3：P1 / P2 sqlc、TxManager、River 后移 | **同意 M2**。当前无 SQL 用例，预写端口/实现违反 YAGNI；M2 首批查询时验证 cgo/解析器版本，事务端口放 shared、技术实现放 postgres。River 迁移所有权另见 Minor 2 |
| K4：P2 不启用 goose advisory lock | **同意 M0，限定 M2 首个真实迁移时重新决定**。当前空集无需锁；若 M2 允许并行 auto_migrate 或多实例部署，应启用协调或明确单独迁移进程。不能把 M0 的无数据安全性外推为生产并发安全 |
| K5：P2 空迁移 / 惰性连接；P6 改正模板版本表措辞 | **同意**。实测 no-op 命令不证明数据库可用；S1 的 pg_stat_activity 已补足当前连接证明。M2 第一份迁移必须增加真实版本断言，不提前造业务表 |
| K6：P2→M2 shared/sqlc 的传递约束 | **同意到首次新增时一起补**。shared 绕行实际成立且已记录，不能先放共享技术类型再等待下一阶段；同批修 Important 1，后者是不依赖 shared 的新路径 |
| K7：P2 不启用禁止所有全局变量规则 | **同意**。当前只见必要 embed/version/哨兵与测试单例；gochecknoinits 与依赖检查在运行。没有理由为了规则名改成空包装 |
| K8：P2 每测试包一容器、测试辅助错误上下文与弱断言 | **同意接受**。全量/race 可运行；相关值由分层配置测试覆盖，pgtest FORCE 有回归；待真实耗时成为问题再集中容器，当前不引入共享测试状态 |
| K9：P2 / P6 先选端口再关闭、就绪后额外等 100ms | **同意 M2 根治**。额外等待降低碰撞概率，不证明端口所有权；M2 可监听 :0 后交付实际地址。不同于 Important 2 的 fetch 无期限 |
| K10：P2 空布尔环境变量、致命错误非结构化、探针日志量 | **同意 M2**。已有具体位置，当前不是数据错误；接入密钥和运维配置时同时确定空值语义、错误/日志策略 |
| K11：P2 / P3 API 错方法 404、路径规范化 307 | **同意接受**。是明确协议取舍，有回归；本次没有发现外部重定向。M8 可重新评估 405，但无需为 M0 静态服务引入另一套路由器 |
| K12：P3 strict 不做 schema 值校验、解码错误带 Go 类型、413/取消映射 | **同意 M2 首个带体接口前完成**。三错误出口接线已核实，但生成器不保证业务值约束；不能仅依靠 handler 类型名称宣称请求契约全覆盖 |
| K13：P3 uuid mapping / nullable / optional pointer / runtime 联动 | **同意 M2**。当 PATCH/UUID 参数真实出现时一起定，当前 enum/value 写法可生成，无需维持假接口 |
| K14：P3 auth security 声明打包、map 型 object 例外、模块入口演进 | **同意 M2 且必须验收**。顶层 security/scheme 的丢失风险已有明确约定，需核对入口/模块/操作级定义和包后文档，不可只跑 doc.Validate；第二模块可按实际需求增加 Deps，见建议 1 |
| K15：P3 panic 后中断响应而日志记 500；const/nullable enum/跨文件 response 的替代写法 | **同意现有裁定**。响应开始后不能改写正文，中断有测试；访问日志的 500 表示 handler 未正常结束，不必增加未使用的日志状态模型。工具写法反例守护仍有效，升级工具再重验 |
| K16：`docs/v0/M0-foundation/reviews/P3-api-contract-review.md:57` 的同仓 PR skip；M8 release handoff | **不同意将充分处理时点一概放到 M8**。原理由是“目前没有设置必须通过的检查，不是现实风险”；这只解释分支保护绕过尚未启用，不能证明 push 的 head 与 PR 的合并结果等价。当前 `.github/workflows/ci.yml:18` 等跳过同仓 PR 的合并结果验证；main 前进时 head 成功不代表合并后成功（推断，未制造远程 PR）。建议并入 M1／启用协作 PR 门禁前去掉 skip，或提供真正验证合并结果的必需聚合检查；增加 CI 消耗但减少晚发现冲突。仅新增总是成功的聚合 job 无效。不重复作为新发现计数 |
| K17：P3 M3 分页、M8 对外文档 lint | **同意**。现在不造无人使用的分页 DTO；M3 统一游标、M8 以 dist 为外部文档来源并评估 lint，有明确使用时点 |
| K18：P4 全量快照、不做 Python fallback、不在 CI 重提取 | **同意**。110 张表是原始证据，不能裁成 44 表冒充来源；固定输入、原子输出、本次重提取一致已提供信心。PG15 是 Plane 输入的版本，不是误把 Nerve PG18 改回去 |
| K19：P4 固定 compose 项目不支持并发提取 | **同意现阶段人工工具限制**。本次运行前确认无同名项目，脚本只清自己的项目；若将来自动化才加唯一项目名或锁，不建常驻编排服务 |
| K20：P5 oxlint “只降不升”依赖人工评审 | **同意 M1 收紧**。固定上限确会失败，但抬上限、改规则、遗漏新包脚本仍需审查；未发现本次基线已被恶意抬高。M1 减少警告时同步降数，自动比较 base ref 可按实际协作成本决定 |
| K21：`docs/v0/M0-foundation/reviews/P5-web-import-review.md:54` 将 `.env` 风险称为“已修” | **同意 M1 删除，不同意把文档提醒等同技术阻断**。`web/apps/web/vite.config.ts:7` 仍加载 .env；README 提醒和 S2 同源断言降低风险。M1 必须连 dotenv 加载路径一起清理，不能只删 .env.example |
| K22：P5→M2 安全头与 `/.well-known/*` 回退 | **同意 M2、在浏览器令牌启用前完成**。当前静态响应缺 CSP/nosniff/Referrer-Policy，样本回退无文件泄露；认证/XSS 威胁出现时需落实，不能再后移到 M8。这里只是响应头，并不替代 Critical 2 |
| K23：P5 不加 web.enabled、原样错误页、Next/SW/Docker/Caddy/serve 遗留 | **同意**。开关没消费者，删除优于预留；M0 加载错误页明确在验收内；M1 清理上游部署与兼容遗留，不把原样上游质量凑作新发现 |
| K24：P5 旧 embed 产物、P6 job 不交接前端 cache | **同意**。make build 清理并复制产物，README 解释 go run 可能看到旧嵌入文件；当前 e2e 明显未超过约定的 5 分钟门槛，不引入跨 job 产物下载复杂度 |
| K25：P6 70 秒 fixture 修复 | **部分同意**。`ready+stop+10` 确已落实，快返 503 的原问题获修；挂起请求没有期限，是 Important 2 的新增失败模式，不能只继续扩大外层预算 |
| K26：P6 knip report-only、提示数随本地生成物变化 | **同意 M1**。379 项不阻塞当前构建是显式范围；1–3 条提示不应成为硬编码验收。M1 去 --no-exit-code 并清理无用依赖；api-client/e2e 上限保持 0 |
| K27：P6 trace/截图/日志够用，录像与 DB 快照后移 | **同意 M2 真实业务时决定**。M0 没有业务数据，不为录像多装依赖；M2 增加快照、PAT 对等、认证 fixture、DB 查询池、networkidle 替换、S2 业务断言 |
| K28：P6 VERSION 两个默认值和 shell quoting | **同意 M8 发布输入接入前完成**。当前 CI 生成可控固定格式，不能把操作员自选构建参数称远程漏洞；从 tag/外部输入取版本前统一来源、校验语义并正确引用 |
| K29：P6 Docker Hub 限流暂无负责人 | **同意不阻塞 M0，不同意永远无负责人**。M8 准备发布 CI 时指定负责人决定凭据/镜像代理与 pin 更新，当前无已发生限流证据，不提前铺运维平台 |
| K30：P2/P3 plan 保留历史内容 | **同意**。历史 plan 中旧框架/命令片段不能等同现有源码；后续以 spec、当前代码和 handoff 为准，不为“文档一致”篡改历史过程 |

## 7. 对 M1–M8 的前瞻

### 7.1 12 个外部 handoff 的可执行性

| 接收方 / 文件（位于对应 M 的 handoffs） | 判断及必须落地的边界 |
|---|---|
| M1 `M0-P5-frontend-trim-notes.md` | 具体：目录、部署遗留、品牌/包名前缀、dotenv、警告基线均可直接开展；加上 Minor 1 的缓存输入，避免裁剪期假绿 |
| M1 `M0-P6-knip-notes.md` | 具体：去 report-only、处理提示特例、保持新包零警告、复核锁文件与 S2；不必增加业务框架 |
| M2 `M0-P1-sqlc-cgo.md` | 具体：首次 sqlc 时验证 cgo 与解析器方言；M0 未提前把技术类型塞入 app，可正常接适配器 |
| M2 `M0-P2-platform-notes.md` | 具体且关键：TxManager、认证中间件、River/停机、shared/sqlc 守护、密钥打码；需先解决 Minor 2，并纳入 Important 1，不能靠目前 instance 的简单构造证明事务正确 |
| M2 `M0-P3-api-codegen-notes.md` | 具体：UUID/PATCH/校验/错误/auth 声明/模块入口/路径组织；最容易漏的是 bundle 后安全声明和请求值校验，需真实接口回归 |
| M2 `M0-P4-schema-conventions.md` | 具体：Django 外键、命名、应用默认值/CHECK、系统表及 UUID 例外。快照是起点，不是可以直接迁成业务库的最终设计 |
| M2 `M0-P5-frontend-api-notes.md` | 具体：Plane 调用替换、无兼容层、同源、webui 特殊路径和安全头；应在令牌管理器上线前验证 CSP 等 |
| M2 `M0-P6-e2e-notes.md` | 具体：PAT、真实迁移版本、模板不变量、S2/S3 扩展、端口、River、快照/录像；加 Important 2，请求挂起必须能停止自己创建的 child |
| M3 `M0-P3-pagination-components.md` | 具体：真实列表再加统一分页/游标和 TS 类型，避免每模块自行定协议 |
| M4 `M0-P4-pg-trgm.md` | 具体：issues.name 搜索索引及向 M8 部署移交扩展要求；建表时决定，当前不创建不用的扩展 |
| M8 `M0-P3-public-api-docs.md` | 具体：以打包后的文档为准、lint/405 再评估；M2 的安全语义不能等到这份交接才查 |
| M8 `M0-P6-release-notes.md` | 大体具体：版本来源/引用、required checks、镜像限流；K16 建议提前，K29 需要负责人。发布前补来源提供方式与供应链 pin 策略，不代表当前许可实物丢失 |

9 个内部 done、12 个外部 handoff 的数量成立；没有发现整项业务范围无接收方。新增发现通过本报告明确归属，遵守本次不改其他文档的约束。M5–M7 没有单独的 M0 handoff 不等于遗漏：存储、clock、webhook fixture 演进已在 M2 E2E 交接中标出后续时点。

### 7.2 返工风险与现在处理的代价

| 里程碑 | 可能阻碍或返工 | 现在做什么、代价与收益 |
|---|---|---|
| M1 前端瘦身 | 品牌 CSS 改动触发 Minor 1；dotenv、knip、人工基线旧约定继续累积；CI skip 让合并组合缺少验证 | 修缓存/fixture/CI 都是小配置或函数改动，可随裁剪做；两项 Critical 先完成。无需重写后端模板 |
| M2 认证与首批业务表 | shared/未知层污染领域；River 两迁移链冲突；schema 值/安全声明遗漏；首迁移打破 nil-provider 与无连接模板假设 | Important 1、Minor 2 和已有六份交接在设计/首接口时闭环，成本小于多个模块照抄后回改；事务、认证和错误出口用真实行为验收，勿预建用不到的接口 |
| M3 工作区与权限 | JWT 无权限字段的设计要求按数据库关系授权；跨模块接口和统一分页可能被随意打破 | 现在保留模块边界即可；M3 用成员移除/角色变更与 PAT 对等故事验证，不提前加权限缓存。成本随真实用例承担，避免缓存失效返工 |
| M4 工作项核心 | 列表组合复杂、事件与任务事务一致性、pg_trgm；simple instance 不能证明这些已可靠 | 现在解决架构守护，M4 再按列表/事件用例补契约及真实 DB 测试；不提前建通用仓储、事件总线或查询 DSL |
| M5 文件 | Critical 2 修复后的普通请求期限不能直接套大型上传；签名 URL、Range、CSP、存储端口需要共同决策 | 现在修无限读取；M5 按上传大小/速率/超时扩展并验证取消和残留对象清理，少量路由差异优于全局关闭期限 |
| M6 迭代与模块 | 多模块事务、日期边界、归档查询可能暴露跨模块捷径 | 复用 TxManager/时钟真实端口与架构规则，不先创建空的 cycles/modules 包；当前没发现 M0 强制它们返工的结构 |
| M7 协作 | 通知生成、幂等投递、权限变化与跨项目可见性 | M2/M4 的事务任务接入先用真实故事校准，M7 再扩展 River 工作流；不把 M0 的无后台任务停机结论当担保 |
| M8 开放与发布 | Webhook 外连安全、重试幂等、版本/签名、备份恢复、源代码提供、供应链与内存目标都未被 M0 验证 | 到发布前以具体交付验收，K1/K28/K29 纳入发布计划；现在只留明确责任与边界。提前造发布平台或宣称当前达到 50 MB 目标都没有证据收益 |

## 8. 做得好的地方

- **迁入和快照有可核对的实物**：全树对比能解释全部最小改动，schema 重新提取完全一致，比信任操作记录更强。
- **工程链确实能运行**：无缓存 lint/build、无结果缓存 Go 测试、race、真实 DB 和浏览器均通过；生成负例也会红。
- **模块模板保持克制**：app 不依赖 pgx/HTTP，bootstrap 显式组装，InfoSource 有真实测试用途；没有为 M2 虚构空 Deps 或未使用的事务接口。
- **错误响应和路由有实测约束**：API 未知路径没有落入 SPA，panic 前后响应处理有测试；请求 ID 与通用 Problem 没有在本次样本中漏出异常细节。
- **交接大部分可以直接执行**：已把 nullable、security、模板库、PAT、安全头、schema 约定等难点交给正确时点，避免把“还没实现”包装成“已经解决”。

## 9. 处理建议清单

### M1 开始前必须做

- [ ] **Critical 1**：查询解析失败时整体脱敏，补合法原始分号密码等反例。
- [ ] **Critical 2**：设置有限正文读取期限，补 keep-alive 完整头但缺正文的回归；用独立少量连接验证即可。

### 可以并入 M1

- [ ] **Important 2**：readiness 请求可取消、失败时等待自身 child 退出；最迟 M2 扩展 E2E 前完成。
- [ ] **Minor 1**：共享 CSS 进入格式缓存输入。
- [ ] **Minor 3**：选择 main 每次保留或允许 pending 替换，并使配置与措辞一致。
- [ ] **K16**：在协作 PR 门禁投入使用前验证合并结果，修同仓 PR skip 的已知弱点，不拖到发布末期。
- [ ] **K20、K21、K23、K26**：随瘦身清 dotenv 与上游遗留、重测并降警告基线、开启 knip 门禁；不用额外造兼容层。

### 随后续指定 M 处理

- [ ] **M2 开始设计／首模块前：Important 1、Minor 2、K3–K6、建议 1**。明确迁移所有权，修架构边界，真实 SQL/River 接入时验证锁、事务、模板与停机。
- [ ] **M2 首带体／认证接口前：K10、K12–K14、K22、K27**。完成请求值/大小/错误/安全声明，密钥日志、浏览器安全头、PAT 对等和 E2E 失败证据。
- [ ] **M2 fixture 扩展：K9、K25**。可靠端口交付、实际迁移版本、故障上限与清理；保留已修的 70 秒预算但不拿它替代内部期限。
- [ ] **M3：K17**。分页/游标和权限故事按真实列表落地。
- [ ] **M4/M5**：落实 pg_trgm、时钟/存储 fixture 与文件路由期限，见 §7；不新增无消费者接口。
- [ ] **M8：K1、K17、K28、K29**。发布版本输入、公开文档、源代码提供方式、供应链固定/更新与镜像限流负责人；验证备份恢复、Webhook 与资源目标。

### 不建议做

- 以“CI 全绿”为由不处理 **Critical 1/2**，或只增加外层超时来掩盖 **Important 2**。
- 因 **Important 1** 推倒现有模块模板，或现在引入全局注册框架、通用仓储、空 Deps；局部边界规则足以解决已证实问题。
- 为 **Minor 1** 关闭全部缓存；准确声明输入比持续丢弃缓存更简单。
- 把未修改的 Plane 业务源码、已登记的错误页与 Next 兼容垫片重新计为 M0 新缺陷；按 M1 计划删除。
- 为一次性 schema 提取增加常驻服务、整套调度锁或每次 CI 拉镜像重提取；当前固定输入加受控人工执行足够。
- 为了验证迁入来源去改 shallow 状态、重写历史、操作既有容器或推送演示分支；本次证据不需要这些高影响动作。

## 10. 本次交付与最终核验

本报告是唯一允许的持久改动。提交前核对 main、被审基线与工作区文件清单；只暂存本文件，提交到本地 main，不 push。报告中的 Critical / Important 仍是待修发现，本次没有实施修复。

最终运行 `git status --porcelain` 仅显示本报告新增。Docker 中原有 20 个容器的 ID 与名称保持一致，包括 `nerve-dev-db-1`；进程核查没有本次遗留的 nerve 或挂起探针子进程。未停止、重建或修改任何原有容器。

## 11. 处理结果（控制者核实与修复）

**做法**：
- 每条发现都先核实，能复现的都复现了，再找根因、修复，并补上回归验证。
- 每条 Codex 发现各用一个提交修复，全部放在分支 `worktree-m0-hardening`（基于 `e340c72`）上，最后以 `--no-ff` 合并进 `main`。
- 修完之后先做一次整分支评审（opus），再对它的修复做一次限定范围的复审；两轮评审提出的问题也在这个分支上修复，写在下表的"评审补充"里。
- 第 9 节的勾选框保留 Codex 原稿的样子，处理状态以本节为准。

### 11.1 逐条处理

| 发现 | 核实 | 根因 | 修复 | 提交 |
|---|---|---|---|---|
| Critical 1 | 成立。`?password=FAKE;still` 原样出现在启动日志里；`pgconn.ParseConfig` 从同一地址解析出的正是这个密码。配置在校验之前就写进日志，所以非法地址（如 `%ZZ`）也会原样打印 | 用另一种解析器（`net/url`）判断哪一段是密码，而 pgx 用的是自己的 libpq 兼容解析器，两者的结论不一致；认不出的部分原样放行 | 采用第 4 节 Critical 1 的方案 2：删除 `redactURL`，`database.url` 整体打码；创建连接池之后，按 pgx 自己的解析结果另记一条连接目标日志（主机、端口、库名、用户），其中没有密码。**评审补充**：pgx 的解析错误会引用连接串，里面的打码同样只是尽量而为，合法写法 `password = secret` 会原样出现在 stderr 上；它的内部错误也可能引用连接串的片段。`NewPool` 不再转述其中任何一段，只返回一条固定信息，指出要检查的地方（语法、它引用的文件、`PG*` 环境变量），不显示任何细节 | `6cd4105`、`dfe3c02`、`12cd9cc` |
| Critical 2 | 成立。请求头读取超时设为 200 毫秒时，一个声明了请求体却不发的请求，2 秒后仍没有响应，连接也没有释放 | 连接的读写阶段里，只有"读请求头"和"空闲"有上限，读请求体和写响应都没有 | 新增 `server.read_timeout`（30 秒）和 `server.write_timeout`（60 秒）并校验：两者都要为正数，且 `read_header_timeout` 不超过 `read_timeout`。回归测试覆盖"请求体不到"和"响应迟到"。上传、下载接口如何单独放宽期限，交给 M5。**评审补充**：<br>- 校验理由写错了机制（Go 会单独使用请求头的上限），`validate.go` 和测试中的说明已改正，规则本身不变；<br>- `write_timeout` 只让写出失败，既不停止 handler，也不取消它的 context，测试注释和文档中"连接的每个阶段都有上限"已改成"连接上的读写都有上限"，并要求 handler 里的阻塞调用自带期限；<br>- 读超时测试加上"handler 已回复 200"的断言，排除请求头超时造成的误判；<br>- bootstrap 的测试配置补上了这两个超时 | `b86bb72`、`fa16c97`、`12cd9cc` |
| Important 1 | 成立。临时加入 `instance/transport` 之后，`domain → transport → net/http` 能通过架构测试 | 模块内的目录不设限：未知目录既排不进依赖方向，也不受纯净性约束。`internal/shared` 也一样：它是允许导入的目标，它自己导入什么却没人检查（K6） | 新增规则 9：模块内的包只能放在 `domain`、`app`、`adapter` 或模块根目录。新增规则 10：`internal/shared` 只能依赖标准库（不含 `net/http`、`database/sql`）和它自己。两条都放在末尾，已有编号不变。评审中的注入复现现在会失败，K6 也随之关闭。**评审补充**：<br>- 边规则只看直接导入，而标准库里的 `expvar`、`net/rpc` 自己就导入 `net/http`。新增 `TestPureLayersReachNoInfrastructure`，沿全部传递依赖检查 `domain`、`app`、`internal/shared`。<br>- 复审发现，依赖图的边用的是源码里写的导入路径，节点用的是解析后的包路径，标准库 vendor 的 `golang.org/x/...` 因此被误判成第三方。现在边改用解析后的包路径，并用真实加载器验证 `net/mail`、`crypto/x509` 算作纯净 | `3a8ebc2`、`81ca2d2`、`413641e` |
| Important 2 | 成立：读代码确认，又用一个接受连接但不响应的替身进程复现 | 就绪期限只在两次请求之间检查，`fetch` 本身没有上限 | 每次请求用剩余时间做 `AbortSignal.timeout`。nerve 没有就绪时，先 SIGKILL 并等它退出，再报错；进程没启动成功时没有进程可等，直接报错。替身验证：30.1 秒时报出带日志路径的错误，子进程已经退出；可执行文件不存在时，约 100 毫秒报错。**评审补充**：`runNerve` 超过 60 秒用 SIGKILL 结束；`db.ts` 连接数据库最多等 10 秒，每条查询最多 30 秒 | `1ac2822`、`4f64f05` |
| Minor 1 | 成立。在真实仓库复现：样式表加一个 `@utility` 后，`@plane/ui#check:format` 的哈希不变，直接命中缓存"通过"，而直接运行 oxfmt 失败 | 格式检查要读的样式表放在另一个工作区包（`tailwind-config`）里，又经 `@import` 引入 npm 包的样式，这些文件都不在任务哈希里 | `check:format` 和 `fix:format` 不再缓存，全部包的格式检查约 2.5 秒。这和第 9 节"不建议关闭缓存、改为声明输入"不同，原因是：只关掉了格式检查这两个任务的缓存；经 `@import` 引入的 npm 样式列不进任务输入，声明出来的输入永远不完整。**评审补充**：原提交说明中"`check:lint` 不读其他包"的说法不对，oxlint 的 import 规则会读依赖包的 `dist/`。现在 `check:lint`、`fix:lint` 依赖 `^build`；`check:types` 本来就依赖它，所以 `lint-web` 不会因此变慢 | `86ac47b`、`bedb00f` |
| Minor 2 | 成立 | 总体设计写在 P2 做出决定之前，P2 的决定只写进了 M2 handoff | 总体设计 5.2 统一为一条 goose 迁移链：River 的 SQL 取自锁定版本的 `river migrate-get`。M2 handoff 补上首次导出和升级时用的参数 | `2da4281`、`573e3e2` |
| Minor 3 | 成立（依据 GitHub 并发组的语义判断） | main 的所有运行共用一个并发组，而一个组里只保留一个排队的运行 | main 的每次运行各占一个组（`ci-main-<run_id>`）；其他分支照旧，按分支取消较早的运行 | `6a58936` |

### 11.2 其他裁定

- **K16（同仓库的 PR 不跑 CI）**：维持原裁定。
  - 本项目在本地以 `--no-ff` 合并后推送 main，不经过 PR；main 的 CI 验证的就是合并提交本身。PR 门禁留到 M8 开放协作、设置必须通过的检查时一起处理。
  - 评审指出，`if: always()` 的汇总任务只能避免"被跳过的任务算作通过"，保证不了检查的是合并后的结果。M0 设计 6.3 和 M8 的 [M0-P6-release-notes](../../M8-open-release/handoffs/M0-P6-release-notes.md) 已补充：如果保留跳过条件，还要在分支保护里打开"合并前分支必须与 main 同步"。
- **建议 1（接口路径归属的守护）**：要等出现第二个 API 模块时才有意义，留给 M2。

### 11.3 同步的文档与交接

- M0 设计：3.3、3.6、3.7、6.3。
- 总体设计：5.2、6.3。
- 前端改动清单：`turbo.json` 一行。
- P2 spec：2.3、2.7、2.10 中已经变化的地方，都加了"M0 加固后的变化"注记。
- M2 的 [M0-P2-platform-notes](../../M2-auth/handoffs/M0-P2-platform-notes.md)：shared 规则已完成；补充 River 迁移 SQL 的来源和导出参数；handler 里的阻塞调用要自带期限。
- M2 的 [M0-P6-e2e-notes](../../M2-auth/handoffs/M0-P6-e2e-notes.md)：fixture 里每一次等待的上限。
- M8 的 [M0-P6-release-notes](../../M8-open-release/handoffs/M0-P6-release-notes.md)：汇总任务的局限。
- 新建 M5 的 [M0-hardening-http-timeouts](../../M5-files/handoffs/M0-hardening-http-timeouts.md)。

### 11.4 验证

- **本地（分支最终提交）**：以下检查全部通过——
  - `make gen-check`；
  - `make lint`：Go 0 issues，前端 49/49，不走缓存重跑时各包警告数仍等于基线；
  - `make test`；
  - `go test -race ./...`；
  - `make build`；
  - `make e2e`：5 个测试通过。
- **对构建出的 `bin/nerve` 做黑盒复查**：
  - 含 `;` 的查询密码不会出现在日志里，日志中是 `database.url=xxxxx` 加上单独一条连接目标；
  - 格式错误的键值形式连接串只得到固定信息，不含密码；
  - 8 个缺请求体的连接都在 `read_timeout`（测试中设为 2 秒）到期时释放，正常请求照常返回 200。
- **评审**：
  - 整分支评审（opus）：With fixes，1 条 Important、8 条 Minor，已全部处理；
  - 修复轮复审（opus）：9 条全部处理到位，并新提出 1 条 Important、2 条 Minor、1 个小问题，也已全部处理（`413641e`、`12cd9cc`）。
