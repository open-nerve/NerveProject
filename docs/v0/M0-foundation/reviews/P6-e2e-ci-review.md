# M0/P6 端到端测试骨架与持续集成：评审记录

| 项 | 内容 |
|---|---|
| Phase | M0/P6 `e2e-ci` |
| 日期 | 2026-09-22 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P6-e2e-ci.md) / [plan](../plans/P6-e2e-ci.md) |
| 分支 | `worktree-m0-p6-e2e-ci`，从 `main` 的 `376f662` 分出，提交从 `4f7cf8f` 到本评审记录所在的提交 |

## 1. 范围与结果

P6 是 M0 的最后一个 Phase，接上流水线的最后一环"端到端测试"，并放进持续集成（spec 1）：

- `make build` 注入版本号（`VERSION ?= 0.1.0-dev`，`-ldflags -X …buildinfo.version`，控制者裁定）。
- Playwright 工作区包 `@nerve/e2e`：`db.ts`、`server.ts`、`api.ts` 三个普通函数模块；`test.ts` 把它们接成 Playwright 的 fixture；`global-setup.ts` 每次运行启动一次 Postgres 容器、迁移出模板库。
- M0 设计第 9 节的四个冒烟故事 S1–S4（`stories/smoke/`）。
- `make e2e`；持续集成新增 `e2e` 任务。
- knip 的配置（`knip.jsonc`）和 `make knip`：M0 只出报告，M1 起作为门禁；`web` 任务执行它。
- README 新增"端到端测试"一节，"前端"一节加上未使用代码检查；关闭交给 P6 的三个交接（P2、P3、P5）。

spec §4 的 9 项验收标准全部满足：本地 `make e2e` 通过（5 个测试），之后没有遗留的 nerve 进程，容器在约 10 秒内消失；S1–S4 的全部反证都按预期失败、恢复后通过；`make build VERSION=9.9.9-p6` 之后 `bin/nerve version` 输出对应版本号，不指定时是 `0.1.0-dev`；`make lint-web` 49 个任务全部成功（含 `@nerve/e2e` 的三项检查）、`make lint-go` 输出 `0 issues.`、`make test` 全部通过、`server/go.mod` 未改动；`make knip` 退出码 0、报告 379 处（全部在迁入的 Plane 代码中）、配置出错时退出码非 0；`pnpm install --frozen-lockfile` 在干净克隆上成功、原有的包一个都没有少；按 `ci.yml` 顺序在干净克隆上走一遍三个任务全部通过，之后 `git status` 为空；持续集成 `server`/`web`/`e2e` 三个任务都通过，`web` 任务日志中有 knip 报告，演示分支上 `e2e` 任务按预期失败并上传 `playwright-report`；交给 P6 的三个 handoff 均为 `done`，`handoffs/` 中没有 `open` 的 M0 事项。

## 2. 原型验证的结论（spec 2.2）

- **环境**：Apple M 系列 18 核、macOS，Node 24.15.0、pnpm 11.10.0、Go 1.27.1、Docker Desktop 4.87（Engine 29.7.2）。先在仓库外的临时目录做原型，再在 P5 合并后的 `main`（`376f662`）新克隆上，按计划的 5 个 Task 完整重放一遍（每个 Task 一个提交：T1 `c7d3204`、T2 `43b69d0`、T3 `2de3db4`、T4 `b461d83`、T5 `a828768`），最后在干净克隆上走了一遍持续集成三个任务的全部步骤。
- **版本**：Playwright 1.63.0（Node 24 上可用，浏览器 Chromium 153）；`@testcontainers/postgresql` 12.1.0（依赖 `testcontainers` 12.1.0，回收容器 `testcontainers/ryuk:0.14.0`）；`pg` 8.23.0；knip 6.37.0。
- **fixture 耗时**：镜像已在本机时，容器启动 0.8–1.0 秒；建模板库并执行 `migrate up` 约 50 毫秒；4 个 worker 同时 `CREATE DATABASE … TEMPLATE` 各约 58 毫秒；nerve 从启动到 `/readyz` 返回 200 约 120 毫秒；SIGTERM 后约 2 毫秒以退出码 0 退出。全套 5 个测试、4 个 worker，本机 Playwright 报告 3.4–5.0 秒，`make e2e`（含未改动时约 1 秒的 `make build`）约 6 秒。
- **架构师在真实 `main` 上的复核**：不只在仓库外的临时原型目录验证，还在 P5 合并后的真实 `main` 新克隆上完整重放了计划的 5 个 Task，确认 spec 和计划里的数字（锁文件行数与哈希、各项耗时、knip 报告的分类计数等）在真实仓库状态下同样成立，然后才把 spec 和计划提交给控制者裁定。

## 3. 各 Task 的评审

| Task | 内容 | 提交范围 | 核实方式 |
|---|---|---|---|
| 1 | `make build` 版本号注入 | `4f7cf8f`..`316b63d` | 实现者 sonnet；**控制者直接复核**（5 行 `Makefile` 差异）：符合 spec 2.7 的 `VERSION ?=`、`GO_LDFLAGS`、`go build` 的 `-ldflags`；`VERSION=9.9.9-p6` 和默认值都按预期写入 |
| 2 | e2e 工作区包、fixtures（`db`/`server`/`api`/`test`）、Playwright 配置、故事 S1 | `316b63d`..`0b0cef4` | 实现者 sonnet；评审（sonnet），Approved：文件字节精确，锁文件 14777 行、SHA `8691dff8…` 复核一致，各路径下的清理都正确，SQL 转义，环境隔离；Minor 已 parked（`db.ts` 的 `connect()` 在 `try` 之外，连接失败时没有存活的 socket）。评审带着**计划规定的 Important**：S1 从 `fixtures/server.ts` 导入 `runNerve`、`applicationName`，与 spec §2.5 字面上"故事只从 `test.ts` 导入"不符，但和 §2.1 的导入图一致（→ Ruling 4：改 spec，不改代码） |
| 3 | 冒烟故事 S2–S4、`make e2e` | `0b0cef4`..`ef7d44c` | 实现者 sonnet；评审（sonnet），Approved：S2 的监听器、按路径分类、同源断言、深层路径用新页面、逐字节相同；S3 严格核对 `NERVE_VERSION` 加 `toEqual`；S4 精确核对 problem 内容；`make e2e` 不经过 turbo；README 指向 `.env` 提醒；6 个反证都复原、没有残留 |
| 4 | CI `e2e` 任务、knip 配置、`web` 任务加 knip 步骤 | `ef7d44c`..`945162b` | 实现者 sonnet（DONE_WITH_CONCERNS）；评审（sonnet），Approved：锁文件 15393 行、SHA `e47cdd6b…` 复核一致；`knip.jsonc` 与 spec 的配置表一致；`e2e` 任务的 `needs`/`if`/缓存/参数/顺序/上传/超时逐项对照 YAML 核实；`server`/`web` 未改动。评审带着**knip 配置提示条数的说明**：实现者在构建过的克隆上看到 3 条而不是 spec 原来写的 2 条（→ Ruling 5：改 spec 措辞为随仓库状态变化，不改代码），并记录 Minor：CI 的 `knip` 步骤总在 `lint-web` 之后（控制者复核：`make lint-web` 之后接着 `make knip` 实测 2 条，条数取决于本地生成过哪些文件，不是恒定值） |
| 5 | 关闭交给 P6 的三个 handoff、README 走查 | `945162b`..`3804f42` | 实现者 sonnet；**控制者直接复核**（3 个 handoff 的关闭）：处理结果与 spec、代码逐条一致，`grep` 确认没有 `open` 状态的 M0 handoff |

## 4. 整分支评审（opus）：With fixes

评审范围 `376f662..3804f42`（评审包排除锁文件和计划）。

| # | 类别 | 发现 | 处理 |
|---|---|---|---|
| Important 1 | Important | fixture 自己的就绪、停机超时都是 30 秒，永远抢不过 Playwright worker fixture 默认的 30 秒总预算（建立和收尾共用）：`nerve did not become ready (log: …)` 这条错误信息可能永远不会出现，SIGKILL 也可能来得太晚或根本没跑，nerve 有被遗留成孤儿进程的风险 | 已修（`a7d466d`）：`nerve` worker fixture 的 Playwright 超时改为 `readyTimeoutMs + stopTimeoutMs + 10` 秒（70 秒，`nerveFixtureTimeoutMs`，从 `server.ts` 导出）。**反证**：让 nerve 连一个不可达地址（`/readyz` 永远 503），单独重跑一个故事，约 30 秒后得到 fixture 自己的 `nerve did not become ready (log: …)` 错误（带 `[cause]`），不是 Playwright 的通用超时消息；`pgrep`/`docker ps` 确认没有遗留进程和容器 |
| Minor 1 | Minor | `knip.jsonc:12-13` 的中文注释写"生成之后也找不到"，是错的：typegen 跑过之后 knip 其实能解析到，会提示"可以从 `ignoreUnresolved` 删掉这一项" | 已修（`a7d466d`）：按 i18n 条目的写法重新措辞 |
| Minor 2 | Minor | `server.ts:41` 生成的子进程没有 `error` 监听器：生成失败（ENOENT/EACCES）会让 worker 直接崩溃，绕过"not ready (log …)"的报错路径 | 已修（`a7d466d`）：加 `child.once("error")`，`waitUntilReady` 通过一个 `spawnError` 取值器抛出同样的 log-path 错误。**反证**：把 `spawn()` 指向一个不存在的路径，1 秒内得到同样的 `(log: …)` 错误（`[cause]` 是 `ENOENT`），worker 本身不崩溃 |
| Minor 3 | Minor | 端口竞争：另一个 worker 的 nerve 短暂占用同一端口时，`/readyz` 可能先答 200，之后这个 worker 自己的子进程才真正退出 | 已修（`a7d466d`）：`/readyz` 第一次答 200 之后再等一个轮询间隔（100 毫秒），重新核对 `child.exitCode`/`signalCode` 是否仍在运行 |
| Minor 4 | Minor | `Makefile:120,128` 的 `VERSION` 未加引号 | 延后到 M8：见 [M0-P6-release-notes](../../M8-open-release/handoffs/M0-P6-release-notes.md)（语义化版本校验和加引号） |
| Minor 5 | Minor | spec §2.5:148 说"复制出的库带着 goose 的版本表"，在 M0 中是假的：没有迁移文件时 `provider` 为 `nil`，连接池惰性连接，`migrate up`/`status` 都不连接数据库，S1 的 `migrate status` 即使连错库也能通过 | 已修：本次 wrap-up（`docs(M0/P6): correct the spec after the final review`）改写 spec §2.5，并建了 M2 的 handoff |
| Minor 6 | Minor | `ci.yml:112` 的 `if: failure()` 在任务被 `timeout-minutes` 取消时不会上传报告 | 已修（`a7d466d`）：改为 `if: failure() \|\| cancelled()` |
| Minor 7 | Minor | 计划的"完成后"清单漏掉 M8 的 handoff；spec §7 本来就有 M8 一行 | 已修：本次 wrap-up 建了 [M0-P6-release-notes](../../M8-open-release/handoffs/M0-P6-release-notes.md) |

### 4.1 修复后的复核（控制者）

修复提交 `a7d466d`（`3804f42..a7d466d`，49 行差异：`.github/workflows/ci.yml`、`e2e/fixtures/server.ts`、`e2e/fixtures/test.ts`、`knip.jsonc`）由控制者直接复核：

- Important 1 已处理：`nerveFixtureTimeoutMs = readyTimeoutMs + stopTimeoutMs + 10_000`（70 秒）挂在 `nerve` worker fixture 上。
- Minor 1 已处理：`knip.jsonc` 的注释按 i18n 条目的写法重新措辞。
- Minor 2 已处理：`child.once("error")` 捕获，`waitUntilReady` 每次轮询都检查，报出同样的 log-path 错误。
- Minor 3 已处理：`/readyz` 200 之后多等一个轮询间隔，重新核对 `exitCode`/`signalCode`。
- Minor 6 已处理：上传条件改为 `failure() || cancelled()`。
- 没有引入新问题：fix-wave 自身的验证（`tsc --noEmit` 干净、`make lint-web` 49/49、`make knip` 退出码 0、报告 379 处、`make e2e` 5 个测试通过）与两个临时反证（超时、生成失败）都已还原，`git status --short` 干净。

## 5. 控制者裁定

| Ruling | 内容 |
|---|---|
| 1 | spec §3 第 1–19 项按原样接受（`make e2e` 唯一入口、浏览器单独安装一次、fixtures 拆成普通函数 + `test.ts` + `global-setup.ts`、`stories/smoke`、S2 两个测试、trace+截图+nerve 日志（M0 不录像不存快照）、`application_name=nerve`、S2 按路径分静态资源和接口、`VERSION ?= 0.1.0-dev` 经 `-ldflags -X`、CI 用 `0.0.0-ci.<run>`、S4 用 Playwright 的 `request`、e2e 任务顺序与 headless shell、不缓存浏览器、20 分钟超时、knip 以 `--no-exit-code` 跑作为 `web` 任务的普通步骤、api-client 的类型测试作为 knip 入口、构建产物靠 `.gitignore` 排除、knip 的三处 `ignoreUnresolved`/`ignoreDependencies`、`knip.jsonc` 与 `e2e/.gitignore`、`allowBuilds` 三项和锁文件的对等后缀变化、testcontainers-node 12.1.0 + `pg`）——都有原型证据支撑，改错代价只是小的配置改动 |
| 2 | `VERSION` 默认值写了 Makefile 和 buildinfo 两处 → 留给 M8 的 handoff；`application_name` 只在测试侧设置（产品行为不在 P6 范围内）；`make build` 留在 `e2e` 任务里（它是持续集成里唯一完整执行 `make build` 的地方，是 DoD 的证据）；`.turbo/cache` 产物交接只在 `e2e` 任务超过 5 分钟时才启用 |
| 3 | 执行分工：五个实现者都用 sonnet（要对 Docker、testcontainers、Playwright、server 的行为做判断）；评审用 sonnet；整分支评审用 opus；spec §4.8 要求的失败演示分支由控制者推送，不由实现者推送 |
| 4 | Task 2 评审带来的计划规定的 Important：S1 从 `fixtures/server.ts` 导入 `runNerve`、`applicationName`，与 spec §2.5"故事只从 `test.ts` 导入"字面不符，但符合 §2.1 的导入图（`runNerve` 是普通辅助函数，不是 fixture，硬塞进 `test.ts` 反而模糊它的角色）——接受实现，在 wrap-up 中改写 §2.5 这句话，不改代码 |
| 5 | 接受 Task 4 的关切：构建过的克隆上 knip 报 3 条配置提示而不是 spec 原来写的 2 条；379 处报告和退出码不受影响——在 wrap-up 中把 spec/README 的措辞改为"随仓库状态变化"；控制者复核后（`make lint-web` 之后 `make knip` 实测 2 条）最终措辞为"随本地生成过哪些文件而变化，实测 1–3 条"，spec、M1 handoff、`knip.jsonc` 注释一并改为这种与状态无关的说法 |
| 6 | 修复轮（一次派工，sonnet）= Important 1（fixture 超时预算改为 ready+stop+10 秒，反证复现"(log: …)"且不留孤儿）+ Minor 1（knip 注释）+ Minor 2（子进程 `error` 监听器）+ Minor 3（就绪后多等一轮再核对存活）+ Minor 6（`failure() \|\| cancelled()`）；Minor 4（`VERSION` 加引号）留给 M8；Minor 5、Minor 7 和全部文档同步、handoff 留给 wrap-up——理由：这几处代码修复都很小，且都在 spec 承诺过的失败路径上；代价是多一轮评审 |

## 6. CI 证据

- **`35702932216`**（提交 `945162b`，分支冷，第一次跑 `e2e` 任务）：成功。`server` 40 秒；`web` 149 秒（install 14、lint-web 90、knip 4、build-web 24、缓存保存 6）；`e2e` 120 秒（setup-go 12、pnpm 缓存恢复 6——命中同一次运行里 `web` 任务先保存的键、install 8、Playwright Headless Shell 18、`make build` 52、`make e2e` 17）；整条流水线约 270 秒。
- **`35703813772`**（提交 `3804f42`，分支热，最终评审前的最后一次推送）：成功。`server` 38 秒；`web` 111 秒（缓存恢复 3、install 12、lint-web 61、knip 3、build-web 16、保存 0——主键命中不再保存）；`e2e` 122 秒（缓存 5、install 9、浏览器 17、`make build` 53、`make e2e` 16）；整条流水线约 233 秒。
- **spec §4.8 的失败演示**：临时分支 `tmp-p6-e2e-fail-demo`（`d17eb38`，S4 的路径改成 `/v0/nope`）上的运行 `35703833783`：`server`、`web` 都通过，`e2e` 任务在 "E2E" 步骤失败；"Upload the Playwright report" 步骤执行，产物 `playwright-report`（640271 字节）已生成；临时分支之后在本地和远程都已删除。
- **`35705775920`**（提交 `a7d466d`，修复轮）：成功。`server` 37 秒；`web` 139 秒（缓存恢复 5、install 9、lint-web 90、knip 4、build-web 23）；`e2e` 112 秒（缓存 5、install 10、浏览器 17、`make build` 38、`make e2e` 19）；整条流水线约 250 秒。修复轮之后的提交只改文档。
- 合并到 `main` 之后的门禁运行（`main` 看不到分支的缓存，是冷运行）记在 M0 收尾时对本节的补充里。

## 7. 移交事项

**本次创建的 handoff（3 个）：**

| handoff | 去向 | 内容 |
|---|---|---|
| [M0-P6-knip-notes](../../M1-frontend-trim/handoffs/M0-P6-knip-notes.md) | M1 | knip 改为门禁；配置提示条数随本地生成过哪些文件而变化（1–3 条）和 `--treat-config-hints-as-errors` 的限制；lint 上限对 `@nerve/api-client`、`@nerve/e2e` 保持 0；锁文件核对命令；品牌替换后 S2 仍不断言页面文字；`.env` 交接指向 P5 给 M1 的 handoff |
| [M0-P6-e2e-notes](../../M2-auth/handoffs/M0-P6-e2e-notes.md) | M2 | PAT 对等故事和认证 fixture；每 worker 连接池和数据库断言函数、失败时的 DB 快照；S1"迁移版本正确"（`migrate status` 只有 M2 起才是真正的数据库检查）和模板库不变量；S3 的 `signup_enabled`；S2 在接口不再 404 之后的加强；`networkidle` 改轮询；端口竞争的根治办法；River 接入后复核停机时间；录像的决定；`clock.ts`/`storage.ts`/`webhook.ts` 沿用 P6 的 fixture 模式（M4/M5/M8） |
| [M0-P6-release-notes](../../M8-open-release/handoffs/M0-P6-release-notes.md) | M8 | 发布时 `VERSION` 的来源；`Makefile` 与 `buildinfo` 两处默认值合一；`VERSION` 加引号和语义化版本校验；把三个任务设为必须检查前按 M0 设计 6.3 处理跳过条件；Docker Hub 限流（`postgres:18.6`、`testcontainers/ryuk:0.14.0`）暂无负责人 |

**本次关闭的 handoff（3 个，均在 Task 5 处理）：**

- [P2-server-platform-p6-e2e-notes](../handoffs/P2-server-platform-p6-e2e-notes.md)：端口由 fixture 选定，经 `NERVE_SERVER__ADDR` 传入，轮询 `/readyz`，不解析日志；数据库从模板库复制，地址经 `NERVE_DATABASE__URL` 传入；SIGTERM 后等待退出，超时发送 SIGKILL。
- [P3-api-contract-p6-notes](../handoffs/P3-api-contract-p6-notes.md)：通过 `createClient` 调用接口，`@nerve/e2e` 参与 `make lint-web` 的类型检查，S4 用 Playwright 的 `request` 直接请求；Playwright 转译 `@nerve/api-client` 的 TS 源码不需要配置；knip 用 `test/*.typecheck.ts` 作为入口、忽略 `schema.gen.ts` 的未使用类型；`e2e` 任务的 `needs` 和 `if` 与另外两个任务相同。
- [P5-web-import-p6-notes](../handoffs/P5-web-import-p6-notes.md)：构建、缓存与版本号（`e2e` 任务沿用 `web` 任务的 pnpm 缓存步骤，自己完整执行 `make build`，按分支分别记录冷/热/合并后的耗时，`VERSION` 经 `-ldflags` 注入）；turbo 与环境变量（e2e 定义三项检查，不经过 turbo 也不用 `pnpm --filter`）；断言注意事项（S2 按路径分类、不碰 `web/apps/web/.env`、knip 读取 `.gitignore` 不需要另外忽略构建产物）。

## 8. M0 完成标准（对照 M0 设计第 10 节）

| 完成标准 | 证据 | 状态 |
|---|---|---|
| P1 到 P6 全部完成，每个 Phase 都有 spec、plan 和 review | P1–P5 均已完成，各有 review；本记录是 P6 的 review | 已勾选 |
| 持续集成中的全部门禁通过 | 分支上 `server`/`web`/`e2e` 三个任务都通过（第 6 节）；main 合并后的门禁运行留给控制者核对 | 待控制者在合并后勾选 |
| `make build` 能构建出单个可执行文件 `bin/nerve`；它加上一个 Postgres，就能完成 S1 到 S4 | `make e2e` 正是这样运行的：CI 的 `e2e` 任务里 `make build` 之后紧跟 `make e2e`，5 个测试全部通过（第 6 节） | 已勾选 |
| 第 7 节的调整建议已确认，并已同步更新到总体设计和差异清单 | P6 未涉及新的调整建议，沿用此前 Phase 已勾选的状态 | 已勾选 |
| 前端改动清单中已登记迁入时的改动 | `frontend-changes.md` §1.2 记录了 P6 对 `pnpm-workspace.yaml`、`pnpm-lock.yaml`、根 `package.json` 的改动 | 已勾选 |
| `handoffs/` 中没有 `open` 状态的事项；需要移交给后续 M 的事项，已放进对应 M 的 `handoffs/` 目录 | `grep -l '^status: open' docs/v0/M0-foundation/handoffs/*.md` 无输出；交给 M1、M2、M8 的 3 个 handoff 已建（第 7 节） | 已勾选 |
| 总体设计中 M0 的状态改为"已完成" | 留给控制者在 main 合并、CI 门禁转绿后一并修改（`v0-design.md` §9.4） | 待控制者勾选 |
