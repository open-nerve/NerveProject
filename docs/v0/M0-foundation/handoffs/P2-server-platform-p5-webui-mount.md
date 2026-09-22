---
status: done
from: M0/P2
to: M0/P5
created: 2026-09-22
---

# P5 挂载 webui 时的路由写法

- `webui` 必须注册为**不带方法**的模式 `/`。
- 如果写成 `GET /` 或 `GET /{path...}`，会和平台已经注册的 `/api/` 产生模式冲突：两个模式互相都不比对方更具体，ServeMux 在注册时会直接 panic。P2 的最终评审已经实际验证过。
- 在 `webui` 的 handler 内部只处理 GET 和 HEAD，其他方法返回 405。
- 同时在 `config.yaml` 中加入 `web.enabled`（P2 没有加入，见 P2 spec 第 3 节的差异 1）。

## 处理结果（M0/P5）

1. `bootstrap.newApp` 在挂上各模块之后执行 `mux.Handle("/", webui.Handler(webFiles))`，注册的是不带方法的 `/`（[P5 spec](../specs/P5-web-import.md) 2.10）。`/api/`、`/healthz`、`/readyz` 仍由平台处理：`bootstrap` 的测试验证了挂上前端之后，`/healthz` 仍返回 JSON，`GET /api/v0/nope`、`POST /api/v0/instance` 仍返回 404 problem+json，其他路径返回 `index.html`。
2. `webui` 的 handler 只处理 GET 和 HEAD，其他方法返回 405，带 `Allow: GET, HEAD`（spec 2.9）。
3. `web.enabled` 没有加入：v0 中没有需要关闭内嵌前端的部署方式，这个开关没有使用者（spec 第 3 节第 5 项）。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
