---
status: open
from: M1/P5
to: M8
created: 2026-09-24
---

# 公开发布之前：propel 的对应源码、版本号和分享图的地址

## `@makeplane/propel` 的对应源码

前端依赖 `@makeplane/propel` 0.3.0（Plane 发布在 npm 上的组件库，AGPL-3.0-only），以编译后的形式随 Nerve 分发。它的上游仓库 `github.com/makeplane/propel` 不公开（2026-09 访问为 404），所以 AGPL 要求的"对应源码"现在拿不到公开的出处。

已知的线索（M1/P5 核对，记在 [前端改动清单](../../frontend-changes.md)第一节的"第三方依赖"一行）：

- npm 上这个版本的 SLSA 来源证明记录了仓库 `github.com/makeplane/propel`、目录 `packages/propel`、提交 `0a31b1529c0f249a058e59ade313d8b34ea8f964` 和工作流 `.github/workflows/release.yml`；
- 包里的 source map 列出 1376 个源文件，带着其中 1375 个的全文，缺 `src/internal/variant-props.ts`。

公开发布之前二选一：取得这个提交的源码（向 Plane 索取或等仓库公开），或者从 source map 还原源码并补上缺的那个文件，随 Nerve 的源码一起提供。

## 版本号

12 个工作区包和 web 应用的 `version` 仍是 Plane 的 1.4.2，帮助菜单显示 "Version: v1.4.2"。Nerve 的版本号随首次发布定，届时一起改，与 [M0/P6 的版本号交接](M0-P6-release-notes.md)（`Makefile` 的 `VERSION`、`buildinfo` 的默认值）用同一个来源。

`web/apps/web/core/components/global/version-number.tsx` 导入整个 `package.json`，客户端包因此带着 web 应用的全部依赖和版本；改为只取 `version`（具名的 JSON 导入，Vite 会摇掉其余字段；或构建时注入）。

## 分享图的地址

`app/root.tsx` 的 `og:image`、`twitter:image` 是相对地址（基线如此），抓取分享预览的一方不会解析相对地址。有了公开地址（或实例配置的站点地址）之后写成绝对地址。

## 关闭条件

M8 发布之前：

- propel 的对应源码随 Nerve 的源码一起提供（取得那个提交的源码，或者从 source map 还原并补上缺的文件），前端改动清单第一节写明出处；
- 版本号与 `Makefile` 的 `VERSION` 同源，`version-number.tsx` 只取 `version`；
- `og:image`、`twitter:image` 是绝对地址。

逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。

来源：[M1/P5 评审记录](../../M1-frontend-trim/reviews/P5-brand-review.md)第 7 节，[P5 spec](../../M1-frontend-trim/specs/P5-brand.md) 7.1。
