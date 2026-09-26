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

## 处理结果（M2/P2）

- **安全响应头**（完成）：`httpserver.NewServer` 的固定链加上第四个中间件，每个响应（接口、页面、健康检查、panic 之后的 500）都带 `X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`。
- **与 CSP 分在两层**（有意的安排，M2 设计 8.3）：这三个响应头对接口的响应同样有意义，所以放在固定链上；CSP 只对页面有意义，而且要用 `index.html` 里内联脚本的哈希，只有 `webui` 知道这些脚本，所以由 M2/P4 在 `webui` 中加入。交接原文"和 CSP 的决定放在一起做——大概率是同一层中间件"的本意是两者都由服务端在 M2 加入，这一点照做；这一项在 P4 加入 CSP 时正式关闭。
- **认证接口在 `/api/v0/` 下**（接口完成）：`register`、`login`、`refreshTokens`、`logout` 都在 `/api/v0/auth/` 下。

仍未处理，状态保持 `open`：CSP（M2/P4）；前端改调 `/api/v0/instance` 和认证接口，以及同源部署的核对（M2/P4）。

来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
