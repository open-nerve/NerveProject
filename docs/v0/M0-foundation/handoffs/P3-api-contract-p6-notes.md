---
status: open
from: M0/P3
to: M0/P6
created: 2026-09-22
---

# P6 端到端测试调用接口的方式，以及持续集成的 e2e 任务

1. **通过 `@nerve/api-client` 调用接口：** `createClient({ baseUrl: "http://127.0.0.1:<端口>" })`，不要手写 fetch 和类型。
   - `e2e` 包依赖 `@nerve/api-client`（`workspace:*`），并参与 `make lint-web` 的类型检查。冒烟故事 S3 要求用生成的客户端调用，并通过类型检查。
   - S4 调用不存在的接口：客户端的类型不允许写不存在的路径。用 `fetch` 直接请求并断言 404 和 `application/problem+json`，或者用 `@ts-expect-error` 标注。
2. **Playwright 要能转译工作区中的 TS 源码。** `@nerve/api-client` 导出的是 TS 源码，它的真实路径不在 `node_modules` 下。确认 Playwright 的转译能处理它；不能处理时，在 Playwright 配置中显式包含这个路径。
3. **knip 的配置要忽略两类文件：**
   - `web/packages/api-client/src/schema.gen.ts`：它导出 `webhooks`、`$defs`、`operations` 等没有被使用的类型；
   - `web/packages/api-client/test/**`：类型测试中的函数只参与类型检查，不会被调用。
4. **持续集成的 `e2e` 任务。**
   - 同仓 PR 上 `server` 和 `web` 任务会被跳过（`if` 条件，见 P3 spec 2.10）。`e2e` 通过 `needs:` 依赖这两个任务，所以也会被跳过；GitHub 把因 `needs` 被跳过的任务也当作不阻止合并。
   - `e2e` 加上与另外两个任务相同的 `if` 条件，保持一致。
   - 以后如果要启用"必须通过的检查"，先去掉这个跳过条件，或者加一个 `if: always()` 的汇总任务，把它设为必须通过的检查（见 M0 设计 6.3）。

来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
