---
status: done
from: M0/P2
to: M0/P6
created: 2026-09-22
---

# P6 端到端测试启动 nerve 时的注意事项

1. **从日志中拿不到端口。**
   - test 环境的日志级别是 `warn`，所以 `http server listening` 这一行（info 级别）不会输出。
   - 每个 worker 启动一个 `nerve` 时，应当事先选好端口，通过 `NERVE_SERVER__ADDR` 传入，再轮询 `/readyz` 直到返回 200。
   - 不要依赖从日志中解析端口；如果确实需要，就用 `NERVE_LOG__LEVEL=info` 覆盖日志级别。
2. **数据库由环境变量提供。**
   - test 环境的 `database.url` 由 `NERVE_DATABASE__URL` 提供。
   - 端到端测试的 fixture 负责从模板库复制出每个 worker 自己的数据库，然后把地址传给 nerve。
3. **停机。**
   - 向 nerve 进程发送 SIGTERM，它会在 `server.shutdown_timeout`（默认 20 秒）内处理完进行中的请求，然后以退出码 0 退出。
   - fixture 应当等待进程退出，避免遗留进程。

## 处理结果（M0/P6）

1. **端口**：`e2e/fixtures/server.ts` 先在 `127.0.0.1` 上取一个空闲端口，通过 `NERVE_SERVER__ADDR` 传给 `nerve serve`，再每 100 毫秒请求一次 `/readyz`，直到返回 200（最多 30 秒；nerve 提前退出时立即失败）。不从日志中解析端口；nerve 的输出写进 `e2e/test-results/nerve-w<编号>.log`（[P6 spec](../specs/P6-e2e-ci.md) 2.5）。实测从启动到就绪约 120 毫秒。
2. **数据库**：全局准备（`e2e/global-setup.ts`）启动一个 Postgres 容器，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`；每个 worker 用 `CREATE DATABASE … TEMPLATE nerve_template` 复制出自己的库（约 60 毫秒），地址通过 `NERVE_DATABASE__URL` 传给 nerve，另加 `application_name=nerve`，S1 据此在 `pg_stat_activity` 中找到 nerve 的连接。调用方自己的 `NERVE_*` 环境变量不传给 nerve。
3. **停机**：worker 结束时向 nerve 发送 SIGTERM，等它退出，退出码不是 0 就报错；30 秒内没有退出就发送 SIGKILL 并报错。实测停机约 2 毫秒，运行结束后没有遗留的 nerve 进程。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
