---
status: closed
from: M0/P5
to: M1
created: 2026-09-22
---

# M1 前端瘦身时的注意事项

## 部署遗留与未使用的依赖

- 删除迁入时留下、M0 用不到的部署遗留：web 中的 `Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`；`serve` 依赖及其 `start`、`preview` 脚本（当前运行即崩溃：`pathToRegExp.compile is not a function`）；`public/` 中从未注册的 `sw.js`、`workbox-*.js` 及其 source map。
- **`.env.example` 整个删除**（不是最初计划的只删 admin、space、live 的地址；P5 最终评审 Important 2：留着这个文件，配上 `vite.config.ts` 里的 dotenv 加载，本地建一个 `web/apps/web/.env` 就会把 `VITE_API_BASE_URL` 悄悄打进构建产物，破坏同源部署）。删除时一并决定 `vite.config.ts` 顶部的 `dotenv.config(...)` 加载，以及 `define: { "process.env": ... }` 这个 Next.js 兼容垫片是否还需要保留。
- 删掉 `serve` 之后复查 `pnpm-workspace.yaml` 中与 Express 相关的覆盖项（`@react-router/serve>express`、`path-to-regexp`、`router>path-to-regexp`、`body-parser`、`morgan`、`qs`），以及 `turbo.json` 中 admin、space、live 的环境变量。
- 删掉 `pnpm-workspace.yaml` 中提到没有迁入内容的英文注释：`.npmrc` 相关设置、Express 4（写着"apps/live"用 express-ws）、"runtime images copy node_modules wholesale"这类部署相关的注释，以及其他提到 apps/live 的地方。
- 功能删减之后，重新用锁文件核对 catalog、overrides、allowBuilds：证明修剪前后解析结果不变，方法与 P5（spec 2.4）相同。
- 清理 `turbo.json` 中没有调用方的任务：`start`、`build-storybook`、`clean`、`check`、`fix*`（`fix`、`fix:format`、`fix:lint`）、`test`。

## 警告基线

- 重新测出每个包的 oxlint 警告上限（M0 设计 5.2）；决定是否加"警告减少后必须调低上限"的自动检查（目前只靠评审把关）。

## 已知的行为差异，处理删减时留意

- 控制台固定有一条 React #418（水合不匹配），来自 Plane 构建时预渲染的 `index.html`，是上游行为。
- 深层路径会被前端补上结尾的 `/`，是 Next.js 兼容垫片的行为；垫片删除后要核实这个行为是否还需要、是否跟着一起消失。
- 构建时的两条警告：`tailwind-config` 的 `package.json` 缺少 `"type": "module"`（Node 的 `MODULE_TYPELESS_PACKAGE_JSON`）；Vite 8 已支持 `resolve.tsconfigPaths`，`vite-tsconfig-paths` 插件可以去掉。
- `make web-dev` 只运行 web 自己的开发服务器，改了 `web/packages/*` 下的代码要重新执行；需要同时监视各包时改为 `turbo run dev --filter=web...`，并把并发数设为 11 以上。

## 处理结果（M1/P1）

状态仍为 `open`：`.env.example`、dotenv 加载和 `define: { "process.env": … }`，以及深层路径的结尾 `/`，按 M1 设计第 4 节在 M1/P4 处理。其余事项已在 M1/P1 处理（[P1 spec](../specs/P1-web-hygiene.md)）：

1. 部署遗留：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`，`serve` 依赖和 `start`、`preview` 脚本，`sw.js`、`workbox-*.js` 及其 source map 已删除；另外删了 `web/apps/web/.gitignore`、`manifest.json` 和 `public/favicon/`（spec 2.5）。关键词守卫的规则看住它们（spec 2.4）。
2. Express 相关的覆盖项：`path-to-regexp: 0.1.13` 删除（删掉 `serve` 之后只剩 Express 5 的 `router` 依赖 path-to-regexp，由 `router>path-to-regexp` 决定）；`@react-router/serve>express`、`router>path-to-regexp`、`body-parser`、`morgan`、`qs` 仍作用于锁文件中的包，保留。`turbo.json` 中 admin、space、live 的环境变量随 `process.env` 注入在 M1/P4 删除（M1 设计 4.1）。
3. `pnpm-workspace.yaml` 中讲 apps/live、Express 4、部署镜像、`.npmrc` 的英文句子已删除（spec 2.6）。
4. 锁文件核对：P1 每次改依赖之后都用一次性脚本比较了 importers 和存活包的版本、完整性哈希、依赖边，一共删掉 369 个包，没有新增；catalog、overrides、allowBuilds、minimumReleaseAgeExclude、patchedDependencies、peerDependencyRules 中没有失效的条目（spec 2.6）。P2、P3 和收尾各再核对一次（M1 设计 9.7）。
5. `turbo.json` 删掉 `start`、`build-storybook`、`clean`、`check`、`fix`、`fix:lint`；`fix:format` 是 README 写明的入口，保留；`test` 由新的 `make test-web` 调用，保留（spec 2.7）。
6. 警告基线改为自动核对：警告数必须等于上限（`tools/lint-cap.mjs`，spec 2.3）。
7. React #418 已修复（spec 2.11）。
8. 两条构建警告已消除（spec 2.10）。
9. `make web-dev` 改为 `turbo run dev --filter=web... --concurrency=12`，同时监视各包，已在运行中的页面上核对（spec 2.13）。

## 处理结果（M1/P4）

状态改为 `closed`：剩下的两项已在 M1/P4 处理（[P4 spec](../specs/P4-router-native.md)）：

1. `.env.example` 已删除；`vite.config.ts` 不再用 dotenv 加载 `.env`，也不再 `define` `process.env`；dotenv 从 web 的开发依赖和 catalog 中删除；`turbo.json` 的 `globalEnv` 只留 `NODE_ENV`。各包不再读取 `process.env` 和 `VITE_*` 变量（i18n 只用 Vite 按构建模式给出的常量 `import.meta.env.DEV`），接口一律用相对路径。关键词守卫的 `frontend-env` 规则看住这一点。
2. 深层路径的结尾 `/` 不再需要，已随垫片删除：`app/layout.tsx` 不再把地址 308 到带 `/` 的形式；应用内部生成的地址一律不带结尾 `/`，"当前是哪一项"的判断对带和不带 `/` 的地址给出相同的结果（M1 设计 4.1）。

来源：[M0/P5 评审记录](../../M0-foundation/reviews/P5-web-import-review.md)。
