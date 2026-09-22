---
status: open
from: M0/P5
to: M2
created: 2026-09-22
---

# M2 对接新接口时，与内嵌前端相关的注意事项

## 前端目前调用的还是 Plane 的接口

- 启动时的 `GET /api/instances/` 改为 `GET /api/v0/instance`。
- `webui` 对任何不是 `/api` 开头的路径都回答 `index.html`（200，`text/html`），`/.well-known/*` 这类路径也不例外。Plane 前端调用的 `/auth/...` 等旧路径，在 Nerve 里现在得到的是前端页面而不是 404；新的认证接口必须放在 `/api/v0/` 下，否则调用方解析到的是 HTML 而不是 JSON。同理，以后任何要对外暴露的非 `/api/v0/` 路径（比如 JWKS）都必须显式注册路由，不能指望默认行为落在 `/api/` 之外时会得到 404。
- 保持同源部署：不设置 `VITE_API_BASE_URL`，接口一律用相对路径；Vite 的开发代理只转发 `/api`（`web/apps/web/vite.config.ts`）。

## 安全响应头

- Plane 的 Caddyfile 设置了 `X-Frame-Options`、`X-Content-Type-Options`；总体设计 4.3 要求严格的 CSP。`webui` 目前不设置任何这类响应头。
- 加 `X-Content-Type-Options: nosniff` 时，和 CSP 的决定放在一起做——大概率是同一层中间件，不要分两次改动路由或中间件链。

来源：[M0/P5 评审记录](../../M0-foundation/reviews/P5-web-import-review.md)。
