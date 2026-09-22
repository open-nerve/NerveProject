---
status: open
from: M0 加固
to: M5
created: 2026-09-22
---

# M5 文件上传、下载与 HTTP 超时

M0 加固之后，HTTP 服务的每个连接阶段都有全局上限：

- `server.read_header_timeout`：读请求头，默认 5 秒；
- `server.read_timeout`：读整个请求（含请求体），默认 30 秒；
- `server.write_timeout`：写响应，默认 60 秒；
- 空闲的 keep-alive 连接：2 分钟，是包内常量。

`local` 存储的上传和下载由 Go 进程自己处理（[v0 设计](../../v0-design.md) 6.7），大文件在慢速网络上会超过这些全局值。处理方式：

- 在上传和下载 handler 内用 `http.ResponseController` 的 `SetReadDeadline` / `SetWriteDeadline` 为这一个请求放宽期限，上限按文件大小上限和可接受的最低速率算出来。
- 不要为此调大全局的 `read_timeout` / `write_timeout`：全局值防的是只发请求头、不发请求体、也不读响应的客户端长期占住连接，调大就让所有接口重新暴露在这个问题下。
- 放宽之后补一个测试：请求体按约定速率发送时能完成；一直不发时仍会在放宽后的期限内断开。
- 请求体大小限制（`http.MaxBytesReader`）和上面的期限是两件事，都要做：前者限制多大，后者限制多久。超限错误映射为 413 的要求见 M2 的 [M0-P3-api-codegen-notes](../../M2-auth/handoffs/M0-P3-api-codegen-notes.md)。

来源：[M0 对抗性评审](../../M0-foundation/reviews/M0-codex-adversarial-review.md) Critical 2 及其处理结果。
