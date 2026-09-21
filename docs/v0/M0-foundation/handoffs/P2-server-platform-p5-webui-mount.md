---
status: open
from: M0/P2
to: M0/P5
created: 2026-09-22
---

# P5 挂载 webui 时的路由写法

- `webui` 必须注册为**不带方法**的模式 `/`。
- 如果写成 `GET /` 或 `GET /{path...}`，会和平台已经注册的 `/api/` 产生模式冲突：两个模式互相都不比对方更具体，ServeMux 在注册时会直接 panic。P2 的最终评审已经实际验证过。
- 在 `webui` 的 handler 内部只处理 GET 和 HEAD，其他方法返回 405。
- 同时在 `config.yaml` 中加入 `web.enabled`（P2 没有加入，见 P2 spec 第 3 节的差异 1）。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
