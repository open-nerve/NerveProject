---
status: done
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

## 处理结果（M0/P5）

1. `.oxlintrc.json` 的 `ignorePatterns` 排除 `web/packages/api-client/src/schema.gen.ts`；`.oxfmtrc.json` 的 `ignorePatterns` 排除它和 `api/dist/**`（oxlint 只检查 JS/TS 文件，不会碰 `api/dist/`）。见 [P5 spec](../specs/P5-web-import.md) 2.7。
2. `make lint-web` 改为 `turbo run check:types check:lint check:format`。`turbo run check:types --dry=json` 列出 12 个类型检查任务，其中有 `@nerve/api-client#check:types`。
3. 根目录的 `package.json` 保留 `@redocly/cli` 2.53.3。换成新的锁文件之后，`make gen-check-web` 没有差异；Plane 的覆盖项把 Redocly 间接依赖的 brace-expansion 从 2.1.7 换成 5.0.9，打包结果不变（spec 2.5）。
4. api-client 的 `typescript` 改为 `catalog:`（5.8.3）。
5. 包名同时存在 `@nerve/*` 和 `@plane/*` 两种前缀，照旧，M1 统一改名。
6. 可选的 tsconfig 继承没有做：api-client 的配置已经是 `strict` 加 `noUncheckedIndexedAccess`；继承 `@plane/typescript-config` 会让新包依赖一个 M1 要改名的 Plane 包。

来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
