---
status: open
from: M0/P6
to: M1
created: 2026-09-22
---

# M1 把 knip 改为门禁时的注意事项

## knip 改为门禁

- `make knip` 去掉 `--no-exit-code`，报告清零，改为持续集成的门禁。
- 决定是否把配置提示也作为错误（`--treat-config-hints-as-errors`）。配置提示的数量随仓库状态变化：干净克隆上 1 条（`tailwind-config` 的 `main` 指向不存在的 `tailwind.config.js`）；i18n 的翻译键生成过之后再加 1 条，共 2 条；`react-router typegen` 也跑过之后（例如 `make lint-web` 之后）再加 1 条，共 3 条——持续集成的 `make knip` 在 `make lint-web` 之后执行，日志里总是 3 条。直接加这个参数会在构建过的克隆或持续集成上失败，除非重新组织 `ignoreUnresolved`，或者让 `make knip` 在 `make lint-web` 之前跑（这样两个提示 typegen 相关的条目还没有生成，不会出现）。
- M1 删掉 react-router 的 typegen 或 i18n 的生成步骤时，同步删掉 `knip.jsonc` 中对应的 `ignoreUnresolved`（`web/apps/web`、`web/packages/i18n` 两项）。
- `knip.jsonc:13` 的注释（P6 最终评审 Minor 1 修复，`a7d466d`）已经写清楚这一条不要删：干净克隆上这些导入解析不到，需要它；构建过之后 knip 能解析到，会提示可以删掉，仍然不要删。

## lint 上限

- 重新测出 lint 基线时，`@nerve/api-client`、`@nerve/e2e` 的上限保持 0：它们是 Nerve 自己的新代码，不像迁入的 Plane 代码那样带着历史警告。

## 锁文件核对

- 删减依赖后核对锁文件，沿用 P6 计划 Task 2 Step 3 的命令：确认没有意外删掉的包，`web` 下的 importers 只有对等依赖后缀（如 `(supports-color@10.2.2)`）的变化，没有别的改动。

## S2 与品牌替换

- 品牌替换之后，S2（`e2e/stories/smoke/s2-web-app.spec.ts`）仍然不应断言页面文字：M2 之前页面显示的还是 Plane 的启动错误页，替换品牌不会让它变成正确内容。
- 前端的包名从 `@plane/*` 改为 `@nerve/*` 时，`knip.jsonc` 里的工作区路径（`web/apps/web`、`web/packages/i18n`、`web/packages/api-client`）不用跟着改：那是目录路径，不是包名。

## `.env`

- `.env.example` 整体删除、`vite.config.ts` 的 dotenv 加载复查，已经在 P5 交给 M1 的 [M0-P5-frontend-trim-notes](M0-P5-frontend-trim-notes.md) 里，P6 不另建。
- P6 只补充：S2 的同源断言保留，作为回归检查，继续防止 `web/apps/web/.env` 悄悄混进构建；去掉 dotenv 加载之后，README"前端"一节"不要建立 `web/apps/web/.env`"一条和"端到端测试"一节"同源"一条里对它的引用要随之修改。

来源：[M0/P6 评审记录](../../M0-foundation/reviews/P6-e2e-ci-review.md)。
