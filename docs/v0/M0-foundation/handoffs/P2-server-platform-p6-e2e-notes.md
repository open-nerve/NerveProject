---
status: open
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

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
