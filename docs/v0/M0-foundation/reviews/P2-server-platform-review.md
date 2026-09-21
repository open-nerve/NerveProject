# M0/P2 服务端平台层：评审记录

| 项 | 内容 |
|---|---|
| Phase | M0/P2 `server-platform` |
| 日期 | 2026-09-22 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完） |
| spec / plan | [spec](../specs/P2-server-platform.md) / [plan](../plans/P2-server-platform.md) |
| 分支 | `worktree-m0-p2-server-platform`，从 `main` 的 `d9e8808` 分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**：
  - 先在仓库外把整个 P2 做出来，测试和 lint 全部通过。
  - 再把代码按 Task 写进 plan。
  - 最后从 P1 的基线开始，按 plan 逐个 Task 重放，确认每个提交都能编译、通过 lint 和测试，结果与原型逐字节一致。
- **逐个 Task 评审**：11 个 Task 各由一个实现子任务照 plan 写入代码，随后由独立的评审子任务检查"是否符合 spec"和"代码质量"。
  - Task 2 需要修一轮：密码打码对查询参数名区分大小写。
  - 其余 Task 一次通过。其中 Task 4 只缺 `make test` 的记录，由控制者补跑。
- **整分支评审**：结论为"修复后可合并"，有 2 个 Important 和 11 个 Minor。修复在一轮中完成，之后对修复的提交做了一次范围评审。
- **持续集成**：推送后通过 GitHub 公开接口确认结果，见第 2 节。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | 单元测试：配置、日志、中间件、problem+json、路由、生命周期、连接池、迁移执行器、命令行 | 通过 |
| 2 | 集成测试（testcontainers）：迁移执行器、`pgtest`、`bootstrap`、`nerve serve` | 通过：本机和持续集成都通过；本机 `-race` 也通过 |
| 3 | 架构测试通过；故意加入违规导入后失败，删除后恢复 | 通过（Task 10）。第二次不加 `-count=1` 运行同样失败，说明测试缓存不会掩盖违规 |
| 4 | `make lint` 输出 `0 issues.`；故意导入 `log` 后 depguard 报错 | 通过（Task 1） |
| 5 | 手工验证：`make run` 后 `/healthz`、`/readyz` 返回 200，`/api/v0/nope` 返回 404 problem+json；数据库不可用时 `/readyz` 返回 503；SIGINT 和 SIGTERM 都能正常停机；`migrate` 各命令和 `version` 的输出 | 通过（Task 9、Task 11、修复轮） |
| 6 | `go 1.27` / `toolchain go1.27.1` 保持不变 | 通过：每次改依赖后都检查过 |
| 7 | 持续集成通过 | 通过：`ed3e7f7`（Task 1–10）的 server 任务 71 秒，其中测试 22 秒；修复轮 `7a3ab7d` 和最终提交见合并记录 |

## 3. 执行中的决定

| 决定 | 原因 |
|---|---|
| spec 第 3 节的 12 项差异全部接受 | 各有理由，都符合"不写用不上的代码"，没有违背已确认的决定 |
| 某个 Task 的评审和下一个 Task 的实现并行（同一时间只有一个实现者提交） | 节省时间；各 Task 改动的文件互不重叠 |
| 实现者必须贴出完整的 `make test` 和 `make lint` 输出 | 前两个 Task 的实现者只跑了本包的测试 |

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| `nerve migrate <拼错的子命令>` 打印帮助后以 0 退出 | Important（plan 带来的） | 已修：`migrate` 父命令拒绝未知子命令，退出码 1 |
| 架构测试明确允许 `app` 层导入 pgx，与 M0 3.1 和总体设计 6.1 矛盾 | Important（plan 带来的） | 已修：规则 2 同时约束 `domain` 和 `app`。`TxManager` 端口改为在 `internal/shared` 中声明、由 `platform/postgres` 实现 |
| `migrate up` 部分失败时，丢掉了已经执行的迁移列表 | Minor | 已修：返回已执行的迁移和错误，命令和自动迁移都会输出 |
| 异常恢复时保留了 handler 已经设置的响应头 | Minor | 已修：只保留 `X-Request-Id` |
| 打码的边界情况：缺少 `//` 的地址、`sslpassword` | Minor | 已修：只按 `postgres://` 和 `postgresql://` 前缀识别地址；键名中含 `password` 的查询参数都打码 |
| 只有 `Config` 实现了打码，单独记录 `DatabaseConfig` 会泄露密码 | Minor | 已修：`DatabaseConfig` 自己实现 `LogValue` |
| 部分测试默认生产迁移集为空 | Minor | 已修：测试自己传入空的迁移集 |
| `pgtest` 强制删除库的测试实际没有覆盖 `WITH (FORCE)` | Minor | 已修 |
| HTTP 服务没有空闲超时；dev 环境监听所有网卡 | Minor | 已修：空闲超时 2 分钟；dev 环境改为 `127.0.0.1:8080` |
| 没有启用 `gochecknoinits` | Minor | 已启用 |
| `providers/file` 把 fsnotify 编进了生产二进制 | Minor | 已修：改用 `os.ReadFile` 加 `rawbytes`，生产二进制中不再有 fsnotify |
| 一个测试用了不带超时的 context；迁移目录中的 `.DS_Store` 会让命名检查失败 | Minor | 已修 |
| 测试中"先监听再关闭"的选端口方式存在竞争 | Minor | 保留：持续集成中出错的概率很低 |
| 环境变量为空值时，布尔配置项被当作 false；`serve` 的致命错误没有结构化日志；探针的访问日志较多 | Minor | 移交 M2（见第 6 节） |

另外做了两项裁定：
- **problem+json 的 `title` 固定为 HTTP 状态短语**，具体说明放在 `detail`，程序判断用 `code`。这符合 RFC 9457 省略 `type` 时的语义，总体设计 3.5 的示例已同步。
- **dev 环境的 `server.addr` 改为 `127.0.0.1:8080`**，与开发库一样只监听本机。

逐个 Task 评审中留下的 Minor：
- Task 3：一个测试只断言没有错误。保留，因为 `TestLoadAppliesLayersInOrder` 已覆盖相关的值。
- Task 7：`pgtest` 内部两处错误没有逐个说明是哪一步。保留，这是只在测试中使用的代码。
- Task 10：缺少跨模块导入生成代码的例子。已在修复轮补上。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改。以 spec 和代码为准。

1. **Task 2 修复**：查询参数名不区分大小写。修复轮进一步改为"键名中含 `password`"，并按前缀识别地址。
2. **修复轮 F1–F12**：
   - `migrate` 父命令拒绝未知子命令；
   - 规则 2 同时约束 `app`；
   - 部分迁移失败时仍返回已执行的列表；
   - 异常恢复时清掉 handler 设置的响应头；
   - 空闲超时；
   - `DatabaseConfig.LogValue`；
   - 读取配置文件改用 `os.ReadFile`，依赖中去掉 `koanf/providers/file`；
   - 启用 `gochecknoinits`；
   - dev 监听 `127.0.0.1:8080`；
   - 测试的稳健性改进。
3. **父文档同步**：M0 设计 3.1、3.3–3.8、6.1、8、12 节，总体设计 3.5、6.3、6.4、6.8、6.9 节，都已按最终实现更新。

## 6. 移交事项

| handoff | 交给 | 内容 |
|---|---|---|
| [P1-repo-toolchain-go-db-notes](../handoffs/P1-repo-toolchain-go-db-notes.md) | M0/P2 | **已处理（done）** |
| [P2-server-platform-p3-notes](../handoffs/P2-server-platform-p3-notes.md) | M0/P3 | 路由挂在根路由上；生成代码的错误处理改为 problem+json；`Problem` 的契约校验 |
| [P2-server-platform-p5-webui-mount](../handoffs/P2-server-platform-p5-webui-mount.md) | M0/P5 | webui 注册为不带方法的 `/`；加入 `web.enabled` |
| [P2-server-platform-p6-e2e-notes](../handoffs/P2-server-platform-p6-e2e-notes.md) | M0/P6 | 端口事先选好并通过环境变量传入；停机方式 |
| [M0-P2-platform-notes](../../M2-auth/handoffs/M0-P2-platform-notes.md) | M2 | `TxManager` 端口的归属；中间件按路由挂载；River 与停机顺序；archtest 的补充；密钥配置项的打码；遗留的小问题 |

## 7. 已知限制

- 集成测试每个包启动一个容器。模块增多后，持续集成会变慢，届时再评估跨进程复用（见 M0 设计第 12 节）。
- `/readyz` 第一次检查从未迁移过的库时，会建出 `goose_db_version` 表（goose 的行为）。
