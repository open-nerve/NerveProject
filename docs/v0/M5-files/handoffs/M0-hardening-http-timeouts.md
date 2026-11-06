---
status: closed
from: M0 加固
to: M5
created: 2026-09-22
---

# M5 文件上传、下载与 HTTP 超时

M0 加固之后，HTTP 服务在连接上的读写都有全局上限：

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

## 处理结果（M5/P2）

P1已通过raw路由逐请求期限与大小上限验证；P2六个业务JSON操作仍受普通1MiB/15s限制，不承载文件字节。T6保持大小/中间件目录与反例，T10真实PUT故事通过；S3和页面慢速上传仍在后续阶段，暂不关闭。证据见[P2阶段评审](../reviews/P2-assets-review.md)，整审/集成结果仍单独核验。

## 处理结果（M5/P8）

P2留下的“S3和页面慢速上传仍在后续阶段”由T10补齐：显式开启的Go演练`TestFilesCleanupDrill`在local与S3上各以160 KiB/s直传3 MiB（约19秒，均超过15秒），完成后经API读回原字节，local另核在其传输预算内；页面慢流`e2e/drills/f1-slow-page-upload.spec.ts`（`test:m5-slow`，Chromium经DevTools把上行限到72 KiB/s）从详情页附件区上传1,310,720字节，PUT约17.8秒，页面给出“File uploaded.”，服务端字节一致。P1的raw路由逐请求期限、大小上限与413（`TestRawStorageDefaultSlowUpload`等）在T12的Go全量测试中重跑通过；全局`read_timeout`/`write_timeout`没有调大。证据见[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)的T10与T12。条款全部完成，closeout改为closed。

## 关闭（M5/closeout，2026-11-06）

逐条结论见上面的处理结果：全局读写上限不调大，文件字节走逐请求放宽期限的raw路由，大小上限与413、local与S3的慢速直传和页面慢流都有本次运行证据（[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)的T10与T12）。没有转交项，`status`改为`closed`。原文与各阶段处理结果保留不改。
