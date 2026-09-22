---
status: done
from: M0/P5
to: M0/P6
created: 2026-09-22
---

# P6 端到端测试骨架搭建时的注意事项

## 构建、缓存与版本号

- `make build` 需要 Node 和 Go，产出约 50 MB 的 `bin/nerve`；`e2e` 任务可以沿用 `web` 任务的 pnpm 缓存步骤（`pnpm store path --silent` + `actions/cache@v6`，按 `hashFiles('pnpm-lock.yaml')` 做键）。
- turbo 的缓存只在任务内部使用，不跨运行保存（P5 spec 2.12）。`web` 和 `e2e` 是两个独立的 job，`e2e` 任务里的 `make build` 会重新跑一遍 `build-web` 依赖的全部任务，用不上 `web` 任务已经构建好的产物。评估是否值得从 `web` 任务把 `build/client` 上传为构建产物（artifact），`e2e` 任务下载后跳过前端构建。
- pnpm 存储缓存（以及如果加了构建产物缓存）都是按 ref 隔离的：PR 分支第一次跑是冷的，合并到 `main` 之后 `main` 上的第一次运行同样是冷的；评估耗时时不要拿 PR 分支热缓存的数字套到 `main` 上。
- **S3 要求 `version` 与构建时注入的版本号一致**：`make build` 目前不注入版本号（默认 `0.1.0-dev`）。控制者裁定（P5 progress ledger Ruling 2）：版本注入是 P6 的事，因为只有 P6 的冒烟故事需要它。需要时加 `-ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=<版本>"`。

## turbo 与环境变量

- `e2e/` 目录本身要定义 `check:types`、`check:lint`（`--max-warnings=0`）、`check:format`，否则 turbo 的 `lint-web` 会静默跳过它——没有这些脚本时，turbo 认为这个包没有对应的任务，不会报错。
- 如果 Playwright 通过 turbo 运行：turbo 默认的严格环境模式只把 `turbo.json` 里列出的变量传给任务，会剥掉 `CI`、`DOCKER_HOST`、`TESTCONTAINERS_*`、`PLAYWRIGHT_*` 这些变量。要么给对应任务加 `passThroughEnv`，要么不经过 turbo，直接用 `pnpm --filter e2e run <脚本>`。

## 断言时的注意事项

- S2 的"静态资源没有 404"：M0 中唯一的 404 是前端调用的 `/api/instances/`（接口，不是静态资源），断言时按资源类型区分；页面文字仍是 Plane 的，M1 才替换品牌，不要断言"Plane"字样；控制台固定有 React #418 和这个 404，M0 中不要断言"控制台没有错误"。
- 不要在 `web/apps/web/` 下建 `.env`：`vite.config.ts` 用 dotenv 加载它，会把 `VITE_API_BASE_URL` 打进构建产物，破坏同源部署（P5 最终评审 Important 2）。e2e 的测试环境如果需要环境变量，走 `e2e/` 自己的 fixtures，不要碰 `web/apps/web/.env`。
- knip 忽略构建产物：`web/apps/web/build/`、`web/apps/web/.react-router/`、`web/packages/*/dist/`、`server/internal/platform/webui/dist/`（P3 交给 P6 的事项仍然有效）。

## 处理结果（M0/P6）

构建、缓存与版本号：

1. **`e2e` 任务的构建和 pnpm 缓存**：`e2e` 任务安装 Go 和 Node，缓存 pnpm 存储的步骤与 `web` 任务相同（同一个键）。先 `make build`，再 `make e2e`；后者中的 `make build` 命中上一步留下的 turbo 和 Go 缓存，约 1 秒（[P6 spec](../specs/P6-e2e-ci.md) 2.9）。
2. **是否从 `web` 任务上传 `build/client`**：评估后不采用。`e2e` 任务自己执行完整的 `make build`，约多花 1 分钟：持续集成中只有这里完整执行 `make build`，它是 M0 完成标准"`make build` 能构建出单个可执行文件"的证据；上传 `build/client`（34 MB、1239 个文件）还要拆分 Makefile。`e2e` 任务超过 5 分钟时，改为在同一次运行中把 `web` 任务的 turbo 缓存交给 `e2e` 任务（spec 2.9）。
3. **缓存按 ref 隔离**：推送后分别记录分支冷、分支热、合并后 `main` 冷三次运行的耗时，不拿分支的热缓存数字套到 `main` 上（[P6 计划](../plans/P6-e2e-ci.md) Task 5 Step 8）。
4. **版本号**：`make build` 用 `VERSION`（默认 `0.1.0-dev`）经 `-ldflags -X` 注入；`make e2e` 把同一个值通过 `NERVE_VERSION` 交给测试，S3 核对；持续集成注入与默认值不同的 `0.0.0-ci.<运行编号>`（spec 2.7）。

turbo 与环境变量：

5. **e2e 的三项检查**：`e2e/package.json` 定义 `check:types`、`check:lint`（`oxlint --max-warnings=0 .`）、`check:format`，`make lint-web` 从 46 个任务变为 49 个（spec 2.4）。
6. **Playwright 不经 turbo 运行，也不用 `pnpm --filter`**：Makefile 直接执行 `cd e2e && pnpm exec playwright test`，Playwright 继承调用方的全部环境变量。`--filter` 写错时 pnpm 只打印 "No projects matched the filters"，退出码为 0，测试会被悄悄跳过（spec 2.7）。

断言：

7. **S2 按路径区分静态资源和接口，不按资源类型**：`/api/` 以外的请求都是静态资源（M0 设计 3.3 的路由规则），必须全部加载成功；接口请求（包括 `GET /api/instances/` 的 404）不计入。M2 起前端用 `fetch` 调用接口，也可能用 `fetch` 取静态文件，路径才是稳定的分界。S2 不断言页面文字和控制台（spec 2.6）。
8. **`web/apps/web/.env`**：e2e 不读写它，需要的环境变量都由 `e2e/` 的 fixtures 设置。S2 另外断言页面的所有请求都发往 nerve 自身：带着从 Plane 的 `.env.example` 复制的 `.env` 构建时，S2 失败，失败信息列出发往 `http://localhost:8000` 的请求（spec 2.6）。
9. **knip 与构建产物**：knip 读取 `.gitignore`，这四类构建产物已被排除，不需要另外配置；在 `make build` 之后运行，报告中没有这些路径（spec 2.8）。

来源：[M0/P5 评审记录](../reviews/P5-web-import-review.md)。
