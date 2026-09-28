---
status: closed
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

## 处理结果（M2/P4）

- **CSP**（完成，M2 设计 8.3）：`webui.Handler` 构造时从内嵌的 `index.html` 取出内联脚本（不带 `src` 的 `<script>`），各算一个 SHA-256，只在 `.html` 的响应上设 `Content-Security-Policy`：`default-src 'self'`，`script-src 'self'` 加每个内联脚本的哈希，`connect-src 'self'`、`object-src 'none'`、`base-uri 'none'`、`form-action 'self'`、`frame-ancestors 'none'` 等（全文见 P4 spec 2.12）；没有构建前端时不设。`webui` 的单元测试用包外另算的哈希核对；`bootstrap` 的测试经整个服务核对页面带它，静态文件、健康检查和接口（包括 `/api/` 下的 404）都不带任何 CSP。S2 核对首页带它，S2、A2、A10 核对没有 CSP 违规；原型的浏览器核对 C9 逐个打开 M2 能到达的 8 个页面，都没有违规（P4 spec 附录 A）。
- **安全响应头与 CSP 分在两层**（关闭，M2 设计 8.3）：三个安全响应头对接口的响应同样有意义，放在 `httpserver` 的固定链上（M2/P2）；CSP 只对页面有意义，而且要用 `index.html` 里内联脚本的哈希，只有 `webui` 知道这些脚本，所以由 `webui` 设置。交接原文"大概率是同一层中间件"的本意是两者都由服务端在 M2 加入，这一点已经做到；分在两层是有意的安排，P4 的评审记录要写下这条理由（P4 spec 第 4 节）。
- **前端改调新接口**（完成）：启动时的 `GET /api/instances/` 改为 `GET /api/v0/instance`；注册、登录、续期、退出走 `/api/v0/auth/` 下的接口，当前账户、资料、修改密码、停用走 `/api/v0/me` 下的接口。前端不再调用 `/auth/…`，关键词规则 `plane-auth-urls` 看住。未登录的页面只请求 `GET /api/v0/instance`（S2）。
- **同源部署**（核对）：前端没有环境变量（M1/P4）；web 的客户端（`publicClient`，和 `apiFor` 给每一代 stores 建的客户端）都由 `createClient()` 建成，不设 `baseUrl`，还没对接的领域用的 axios 基类也不设基础地址，接口一律用相对路径；S2 断言页面的请求都发往 nerve 自身；原型的浏览器核对 C5 在局域网地址上用 HTTP 打开页面，注册和两个标签页的续期都成功（P4 spec 附录 A）。

全部处理完，状态改为 `done`。

来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

两节都在 M2/P2、P4 完成，收尾在 `d97c513` 上核对：

- **前端调用的接口**：未登录的页面只请求 `GET /api/v0/instance`（S2 的 `apiRequests`，随 `make e2e` 通过）；关键词规则 `plane-auth-urls`、`plane-user-urls` 挡住 Plane 的认证和实例地址（`node tools/keywords.mjs` 没有命中）；同源部署：web 的客户端都由 `createClient()` 建成、不设 `baseUrl`，axios 基类也不设基础地址（M2/P4 的处理结果）。
- **安全响应头与 CSP**：`server/internal/bootstrap/headers_test.go` 的 `TestOnlyPagesHaveAContentSecurityPolicy` 随 `make test` 通过；S2 核对首页带 CSP、没有违规。分在两层的理由在 [M2/P4 评审](../reviews/P4-web-auth-review.md)第 6 节。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
