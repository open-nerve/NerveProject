---
status: open
from: M0/P3
to: M0/P5
created: 2026-09-22
---

# P5 迁入前端时，与接口契约相关的注意事项

1. **格式和 lint 检查排除生成的文件。** 加入 oxfmt、oxlint 时，排除 `web/packages/api-client/src/schema.gen.ts` 和 `api/dist/`。这两个文件的格式由生成工具决定，不手改。
2. **`make lint-web` 改由 turbo 驱动。**
   - P3 中它是 `pnpm -r run check:types`。api-client 的脚本已经按 Plane 的习惯命名为 `check:types`。
   - 改用 turbo 之后，要确认 api-client 仍在类型检查的范围内。
3. **合并根目录的 `package.json` 时，保留 `@redocly/cli`。** 它是 `make gen-web` 打包接口描述所需的工具（`2.53.3`）。
4. **`typescript` 的版本。** 引入 Plane 的依赖版本表（catalog）后，api-client 的 `typescript` 改为 `catalog:`，与 Plane 的包使用同一个版本（5.8.3）。
5. **包名同时存在两种前缀。** `@nerve/api-client` 从一开始就用最终的名字；Plane 的包在 M1 之前仍是 `@plane/*`。这是预期的。
6. **tsconfig 的兼容性。**
   - P3 的最终评审已经验证：Plane 的 composite、strict 配置能通过 `node_modules` 编译 api-client 导出的 TS 源码。
   - 可选：让 api-client 继承 `@plane/typescript-config/base.json`，使它的类型测试在最严格的配置下运行（`exactOptionalPropertyTypes`、`noUnused*`）。

来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
