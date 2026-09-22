# M0/P5 前端迁入与内嵌 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Plane 提交 `02c19e1` 的 web 应用和它用到的 12 个包原样迁入 `web/`，合并根目录的 pnpm 工作区、turbo 配置和锁文件；建立 lint 警告基线，`make lint-web` 改由 turbo 驱动；Vite 开发服务器转发 `/api`，加入 `make web-dev`；新增 `platform/webui`，内嵌前端并挂到根路由；`make build` 构建出内嵌前端的 `bin/nerve`；持续集成的 `web` 任务构建前端。

**Architecture:** 前端代码用 `git archive` 按完整 SHA 从本地参考源码 `plane/` 复制，提交前逐个比较 git 树对象，保证与 Plane 完全相同；之后的改动只有几行，全部登记在前端改动清单中。锁文件以 Plane 的锁文件为起点生成，迁入的包的依赖版本与 Plane 相同。前端任务由 turbo 按 `turbo.json` 编排，Makefile 仍是唯一的入口。Go 端新增的 platform 包 `webui` 只依赖标准库：`//go:embed all:dist` 内嵌 `make build` 复制进来的产物；handler 接收 `fs.FS`（测试用 `fstest.MapFS`），由 `bootstrap` 注册在不带方法的 `/` 上，`/api/`、`/healthz`、`/readyz` 不受影响。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、oxlint 1.51.0、oxfmt 0.35.0（都来自 Plane 的 catalog）；Go 1.27.1、golangci-lint 2.13.2。

**Spec:** `docs/v0/M0-foundation/specs/P5-web-import.md`（上级：`docs/v0/M0-foundation/M0-design.md`）

## Global Constraints

- 前端代码从 Plane 提交 `02c19e1341d93141e8ad7b3278298adce208bafc` 复制。**不要手改复制来的文件**，只做本计划写明的改动；spec 2.3 列为"不迁入"的内容一律不复制。
- 本计划的命令用到两个 shell 变量。每条用到它们的命令执行前都要先设置（它们不写进任何文件）：

  ```bash
  PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"
  SHA=02c19e1341d93141e8ad7b3278298adce208bafc
  ```

  `PLANE` 指向主检出目录下的参考源码 `plane/`（已加入 `.gitignore`），在主检出目录和 `.claude/worktrees/<名字>/` 工作区中都能用。`plane/` 是只读的：只对它执行 `git archive`、`git show`、`git rev-parse`。
- Node 依赖通过 pnpm 工作区安装，不做任何全局安装；pnpm 由 corepack 按 `packageManager` 字段提供（11.10.0）。开始之前执行一次 `corepack enable` 和 `pnpm install --frozen-lockfile`（P1 已要求）。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 由 Task 1 的命令生成。它的 SHA-256 是写作时的值：Nerve 自己的依赖（Redocly 等）的间接依赖取决于安装时 npm 上的版本，对不上时以 Task 1 Step 8 的核对为准，并以提交的锁文件为准。
- Go：`server/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`；本 Phase 不新增 Go 依赖。规则同 P3：不写 `init()`，不用全局可变状态，构造函数显式传入依赖，一个文件只做一件事。
- 每个改动了 Go 代码的 Task 提交前：`make lint` 输出 `0 issues.` 且前端检查通过，`make test` 全部通过。`make test` 需要 Docker 在运行（testcontainers）。**只能通过 testcontainers 和 `make dev-db` 使用 Docker，不要停止或改动其他任何容器。** 开发库：`make dev-db`，容器 `nerve-dev-db-1`，端口 55432。
- 代码注释用英文；配置文件、YAML、Makefile 中的注释用中文。从 Plane 复制来的英文注释保留原文。
- 所有代码块都是完整的文件内容（"修改"步骤除外，它给出"把……替换为……"的原文），照原样写入，不要改动。**Makefile 的命令行以 Tab 开头**：写入后用 `grep -c "$(printf '\t')" Makefile` 核对（各 Task 给出预期值）；Tab 变成空格时 make 会报 `missing separator`。Makefile 必须兼容 macOS 自带的 GNU Make 3.81。
- 后台启动的进程（`make run`、`make web-dev`、`bin/nerve serve`）在所在的步骤结束前停止，最后确认 `lsof -iTCP:8080 -sTCP:LISTEN` 和 `lsof -iTCP:3000 -sTCP:LISTEN` 都没有输出。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明（`cd server && …` 这类命令只在该行内切换目录）。

## 文件结构

路径相对于仓库根目录。

| 文件 | 职责 | Task |
|---|---|---|
| `web/apps/web/`、`web/packages/`下 12 个包、`patches/react-color@2.19.3.patch`（复制） | Plane 的前端，原样迁入 | 1 |
| `package.json`、`pnpm-workspace.yaml`、`turbo.json`（修改或新建） | 根目录的工作区和任务编排 | 1 |
| `pnpm-lock.yaml`（生成） | 锁文件 | 1 |
| `web/packages/api-client/package.json`（修改） | `typescript` 改用 `catalog:`（Task 1）；lint、格式检查脚本（Task 2） | 1、2 |
| `.gitignore`（修改） | 前端的生成物（Task 1）；`webui/dist/`（Task 4） | 1、4 |
| `.oxlintrc.json`、`.oxfmtrc.json` | lint 和格式配置 | 2 |
| `web/apps/web/package.json` 及 editor、i18n、propel、ui、utils 的 `package.json`（修改） | 警告上限 | 2 |
| `Makefile`（修改） | `lint-web`（Task 2）、`web-dev`（Task 3）、`build`、`build-web`（Task 5） | 2、3、5 |
| `web/apps/web/vite.config.ts`（修改） | 开发代理 | 3 |
| `docs/v0/frontend-changes.md`（修改） | 来源提交和迁入时的改动 | 3 |
| `server/internal/platform/webui/embed.go`、`handler.go`、`embed_test.go`、`handler_test.go`、`dist/.gitkeep` | 内嵌前端与单页应用回退 | 4 |
| `server/internal/bootstrap/app.go`、`commands.go`、`app_test.go`（修改）、`webui_test.go` | 挂载 `webui` | 4 |
| `.github/workflows/ci.yml`（修改） | `web` 任务：pnpm 缓存、前端构建 | 6 |
| `README.md`（修改） | 前端的开发和构建 | 6 |
| `docs/v0/M0-foundation/handoffs/P2-server-platform-p5-webui-mount.md`、`P3-api-contract-p5-notes.md`、`P4-plane-schema-p5-notes.md`（修改） | 交接事项改为 done | 6 |

---

### Task 1: 迁入前端源码、根目录的工作区和锁文件

**Files:**
- Create（从 Plane 复制）: `web/apps/web/`、`web/packages/{constants,editor,hooks,i18n,propel,services,shared-state,tailwind-config,types,typescript-config,ui,utils}/`、`patches/react-color@2.19.3.patch`
- Create: `turbo.json`
- Modify: `package.json`、`pnpm-workspace.yaml`、`web/packages/api-client/package.json`、`.gitignore`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: 参考源码 `plane/` 中的提交 `02c19e1341d93141e8ad7b3278298adce208bafc`；P3 的根 `package.json`、`@nerve/api-client`
- Produces:
  - 14 个工作区包（13 个来自 Plane，加上 `@nerve/api-client`），`pnpm install --frozen-lockfile` 通过
  - turbo 任务 `build`、`check:types`、`check:lint`、`check:format`、`dev`（`turbo.json`），Task 2、3、5 的 Makefile 命令调用它们
  - 前端的构建产物 `web/apps/web/build/client/`，Task 5 复制它

- [ ] **Step 1: 确认参考源码中有这个提交**

Run: `PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc; git -C "$PLANE" rev-parse "$SHA^{commit}"`
Expected:

```
02c19e1341d93141e8ad7b3278298adce208bafc
```

失败（`fatal: … unknown revision`）时停下来：`plane/` 中没有这个提交，不要用别的提交代替。

- [ ] **Step 2: 复制 web 应用、12 个包和补丁**

```bash
PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc
git -C "$PLANE" archive "$SHA" -- apps/web packages/constants packages/editor packages/hooks packages/i18n packages/propel packages/services packages/shared-state packages/tailwind-config packages/types packages/typescript-config packages/ui packages/utils | tar -x -C web
git -C "$PLANE" archive "$SHA" -- patches | tar -x
```

Run: `find web/apps/web web/packages -path web/packages/api-client -prune -o \( -type f -o -type l \) -print | wc -l && ls patches`
Expected:

```
    4060
react-color@2.19.3.patch
```

（4060 个文件中有一个符号链接：`web/packages/i18n/locales → src/locales`。）

- [ ] **Step 3: 写入 `package.json`（完整内容）**

```json
{
  "name": "nerve",
  "private": true,
  "license": "AGPL-3.0-only",
  "engines": {
    "node": "^24"
  },
  "packageManager": "pnpm@11.10.0+sha512.0b7f8b98060031904c017e3a41eb187a16d40eeb829b95c4f8cb03681761fc4ab53dd219115b9b447f4dce1a05a214764461e7d3703392a9f32f9511ce8c86c8",
  "devDependencies": {
    "@redocly/cli": "2.53.3",
    "oxfmt": "catalog:",
    "oxlint": "catalog:",
    "turbo": "catalog:"
  }
}
```

- [ ] **Step 4: 写入 `pnpm-workspace.yaml`（完整内容）**

```yaml
# pnpm 工作区的根（见 docs/v0/M0-foundation/M0-design.md 第 2 节）。
# catalog 及以下各节来自 Plane 提交 02c19e1 的 pnpm-workspace.yaml，只保留作用于迁入的包的条目；
# 英文注释是 Plane 的原文（见 docs/v0/M0-foundation/specs/P5-web-import.md 2.4）。
packages:
  - web/apps/*
  - web/packages/*
  - e2e

catalog:
  "@atlaskit/pragmatic-drag-and-drop": "1.7.4"
  "@atlaskit/pragmatic-drag-and-drop-auto-scroll": "1.4.0"
  "@atlaskit/pragmatic-drag-and-drop-hitbox": "1.1.0"
  "@base-ui-components/react": "1.0.0-beta.3"
  "@bprogress/core": "^1.3.4"
  "@chromatic-com/storybook": "5.2.1"
  "@floating-ui/dom": "^1.7.1"
  "@floating-ui/react": "^0.27.20"
  "@fontsource-variable/inter": "5.2.8"
  "@fontsource/ibm-plex-mono": "5.2.7"
  "@fontsource/material-symbols-rounded": "5.2.30"
  "@headlessui/react": "^2.2.10"
  "@hocuspocus/provider": "2.15.2"
  "@makeplane/propel": "0.3.0"
  "@popperjs/core": "^2.11.8"
  "@react-pdf/renderer": "^4.8.1"
  "@react-router/dev": "8.3.0"
  "@react-router/node": "8.3.0"
  "@storybook/addon-designs": "11.1.3"
  "@storybook/addon-docs": "10.4.6"
  "@storybook/addon-links": "10.4.6"
  "@storybook/addon-onboarding": "10.4.6"
  "@storybook/addon-styling-webpack": "3.0.2"
  "@storybook/addon-webpack5-compiler-swc": "4.0.3"
  "@storybook/react": "10.4.6"
  "@storybook/react-vite": "10.4.6"
  "@storybook/react-webpack5": "10.4.6"
  "@tailwindcss/postcss": "4.1.17"
  "@tailwindcss/typography": "0.5.19"
  "@tanstack/react-table": "^8.21.3"
  "@tiptap/core": "^2.22.3"
  "@tiptap/extension-blockquote": "^2.22.3"
  "@tiptap/extension-character-count": "^2.22.3"
  "@tiptap/extension-collaboration": "^2.22.3"
  "@tiptap/extension-document": "^2.22.3"
  "@tiptap/extension-emoji": "^2.22.3"
  "@tiptap/extension-heading": "^2.22.3"
  "@tiptap/extension-image": "^2.22.3"
  "@tiptap/extension-list-item": "^2.22.3"
  "@tiptap/extension-mention": "^2.22.3"
  "@tiptap/extension-placeholder": "^2.22.3"
  "@tiptap/extension-task-item": "^2.22.3"
  "@tiptap/extension-task-list": "^2.22.3"
  "@tiptap/extension-text": "^2.22.3"
  "@tiptap/extension-text-align": "^2.22.3"
  "@tiptap/extension-text-style": "^2.22.3"
  "@tiptap/extension-underline": "^2.22.3"
  "@tiptap/html": "^2.22.3"
  "@tiptap/pm": "^2.22.3"
  "@tiptap/react": "^2.22.3"
  "@tiptap/starter-kit": "^2.22.3"
  "@tiptap/suggestion": "^2.22.3"
  "@types/chroma-js": "^3.1.2"
  "@types/hast": "^3.0.4"
  "@types/lodash-es": "4.17.12"
  "@types/mdast": "^4.0.4"
  "@types/node": "22.12.0"
  "@types/react": "19.2.17"
  "@types/react-color": "^3.0.9"
  "@types/react-dom": "19.2.3"
  "@types/sanitize-html": "2.16.0"
  "autoprefixer": "^10.4.19"
  "axios": "1.18.1"
  "buffer": "^6.0.3"
  "chroma-js": "^3.2.0"
  "class-variance-authority": "0.7.1"
  "clsx": "^2.1.1"
  "cmdk": "^1.1.1"
  "comlink": "^4.4.1"
  "date-fns": "^4.1.0"
  "dotenv": "16.4.7"
  "emoji-picker-react": "^4.5.16"
  "emoji-regex": "^10.3.0"
  "export-to-csv": "^1.4.0"
  "file-type": "^21.3.1"
  "framer-motion": "^12.23.0"
  "frimousse": "^0.3.0"
  "hast": "^1.0.0"
  "hast-util-to-mdast": "^10.1.2"
  "highlight.js": "^11.8.0"
  "i18next": "25.10.9"
  "i18next-icu": "2.4.3"
  "i18next-resources-to-backend": "1.2.1"
  "is-emoji-supported": "^0.0.5"
  "isbot": "^5.1.31"
  "jsx-dom-cjs": "^8.0.3"
  "linkifyjs": "^4.3.2"
  "lodash-es": "4.18.1"
  "lowlight": "^3.0.0"
  "lucide-react": "0.469.0"
  "mdast": "^3.0.0"
  "mobx": "6.12.0"
  "mobx-react": "9.2.2"
  "mobx-utils": "6.0.8"
  "next-themes": "0.4.6"
  "oxfmt": "0.35.0"
  "oxlint": "1.51.0"
  "postcss": "8.5.25"
  "postcss-cli": "^11.0.0"
  "postcss-nested": "^6.0.1"
  "prosemirror-codemark": "^0.4.2"
  "react": "19.2.8"
  "react-color": "^2.19.3"
  "react-day-picker": "9.5.0"
  "react-dom": "19.2.8"
  "react-dropzone": "^14.2.3"
  "react-fast-compare": "^3.2.2"
  "react-hook-form": "^7.84.0"
  "react-i18next": "16.6.6"
  "react-is": "^19.2.8"
  "react-markdown": "^10.1.0"
  "react-masonry-component": "^6.3.0"
  "react-pdf-html": "^2.1.2"
  "react-popper": "^2.3.0"
  "react-router": "8.3.0"
  "recharts": "^2.15.4"
  "rehype-parse": "^9.0.1"
  "rehype-remark": "^10.0.1"
  "remark-gfm": "^4.0.1"
  "remark-stringify": "^11.0.0"
  "sanitize-html": "2.17.7"
  "serve": "14.2.5"
  "smooth-scroll-into-view-if-needed": "^2.0.2"
  "storybook": "10.4.6"
  "swr": "2.4.2"
  "tailwind-merge": "3.4.0"
  "tailwindcss": "4.1.17"
  "tippy.js": "^6.3.7"
  "tiptap-markdown": "^0.8.10"
  "tsdown": "0.16.0"
  "tsx": "4.20.6"
  "turbo": "2.10.11"
  "typescript": "5.8.3"
  "unified": "^11.0.5"
  "use-font-face-observer": "^1.3.0"
  "uuid": "14.0.0"
  "vite": "8.0.16"
  "vite-tsconfig-paths": "^5.1.4"
  "vitest": "^4.1.11"
  "y-indexeddb": "^9.0.12"
  "y-prosemirror": "^1.3.7"
  "y-protocols": "^1.0.6"
  "yjs": "^13.6.20"
  "zod": "^3.25.76"

overrides:
  # Force a single React across the whole graph. With node-linker=isolated,
  # auto-install-peers and resolution-mode=highest, a dependency still declaring a
  # React 18 peer can otherwise pull in a second copy, which surfaces as an invalid
  # hook call rather than as an install error. The vite `resolve.dedupe` entries only
  # cover the three app client bundles, not the SSR build, the tsdown package builds
  # or Storybook.
  react: "catalog:"
  react-dom: "catalog:"
  react-is: "catalog:"
  "@types/react": "catalog:"
  "@types/react-dom": "catalog:"
  # @react-router/serve v8 needs Express 5 — it mounts with the Express 5 path syntax
  # `app.all("/{*splat}")`, which Express 4 silently fails to match, 404ing every route.
  # The catalog stays on Express 4 for apps/live, which depends on express-ws (Express 4 only).
  "@react-router/serve>express": "^5.2.1"
  mdast-util-to-hast: 13.2.1
  valibot: 1.4.2
  glob: 11.1.0
  # brace-expansion <5.0.9 has two DoS advisories (GHSA-mh99-v99m-4gvg, GHSA-rgw5-rvv9-x895,
  # the second bypasses the first's mitigation) reachable via serve>serve-handler>minimatch.
  brace-expansion: 5.0.9
  nanoid: 3.3.18
  esbuild: 0.28.1
  "@babel/core": 7.29.7
  "@babel/helpers": 7.29.7
  "@babel/runtime": 7.29.7
  chokidar: 3.6.0
  prosemirror-view: 1.40.0
  typescript: "catalog:"
  vite: "catalog:"
  qs: 6.16.0
  diff: 5.2.2
  webpack: 5.104.1
  lodash-es: "catalog:"
  lodash: 4.18.1
  markdown-it: 14.2.0
  "minimatch@3": 3.1.4
  "minimatch@10": 10.2.3
  "ajv@6": 6.14.0
  "ajv@8": 8.18.0
  flatted: 3.4.2
  picomatch: 2.3.2
  "yaml@2": 2.8.3
  # Pinned for Express 4 (apps/live), which needs path-to-regexp ~0.1.12.
  path-to-regexp: 0.1.13
  # Express 5's router needs path-to-regexp 8; the pin above would otherwise reach it
  # and break route matching at startup with `pathRegexp.match is not a function`.
  "router>path-to-regexp": "^8.4.2"
  defu: 6.1.5
  postcss: 8.5.25
  axios: "catalog:"
  follow-redirects: 1.16.0
  uuid: "catalog:"
  # SSRF / host-confusion in the URI parser (percent-decoding, IDN
  # canonicalization and IPv6 normalization): CVE-2026-75899, -75931, -75975 and
  # -76172, all fixed in 3.1.6. The previous floor pinned 3.1.5, which is the
  # affected version. fast-uri reaches us only through ajv, so stay on the
  # patched 3.x rather than 4.x, which is outside ajv's ^3.0.1 range.
  fast-uri: 3.1.6
  "js-yaml@4": 4.3.2
  linkify-it: 5.0.2
  body-parser: 1.20.6
  form-data: 4.0.6
  "ws@8": 8.21.0
  morgan: 1.12.0
  # Prototype pollution and unbounded query-cache growth, both reachable as DoS
  # (CVE-2026-73088 / CVE-2026-73089). browserslist is build tooling only
  # (babel/postcss/vite), but the runtime images copy node_modules wholesale, so
  # it lands in the scanned surface.
  browserslist: ">=4.28.7"

allowBuilds:
  "@swc/core": true
  esbuild: true
  turbo: true

minimumReleaseAgeExclude:
  - "@storybook/addon-docs@10.4.6"
  - "@storybook/builder-vite@10.4.6"
  - "@storybook/csf-plugin@10.4.6"
  - "@storybook/react-dom-shim@10.4.6"
  - "@storybook/react-vite@10.4.6"
  - "@storybook/react@10.4.6"
  - storybook@10.4.6
  - "@makeplane/propel@0.3.0"

patchedDependencies:
  react-color@2.19.3: patches/react-color@2.19.3.patch

peerDependencyRules:
  allowedVersions:
    # react-popper 2.3.0 is unmaintained and its peer range stops at React 18, but it
    # only uses hooks — no findDOMNode, no string refs, no element.ref reads and no
    # defaultProps on function components — so it runs unchanged on React 19. All 32
    # call sites import usePopper only. Recorded here so the acceptance is explicit
    # rather than hidden behind the global strict-peer-dependencies=false.
    "react-popper>react": "19"
    # Same situation: peer range predates React 19, uses nothing React 19 removed.
    "react-masonry-component>react": "19"
    "use-font-face-observer>react": "19"
```

核对：从 `catalog:` 开始，与 Plane 的原文相比只有删除，没有新增的行（删掉 39 个 catalog 条目、14 个 overrides 及 toml 的 5 行注释、3 个 allowBuilds，见 spec 2.4）。

Run: `PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc; diff <(git -C "$PLANE" show "$SHA:pnpm-workspace.yaml" | sed -n '/^catalog:/,$p') <(sed -n '/^catalog:/,$p' pnpm-workspace.yaml) | grep -c '^>'; diff <(git -C "$PLANE" show "$SHA:pnpm-workspace.yaml" | sed -n '/^catalog:/,$p') <(sed -n '/^catalog:/,$p' pnpm-workspace.yaml) | grep -c '^<'`
Expected:

```
0
61
```

- [ ] **Step 5: 写入 `turbo.json`（完整内容）**

```json
{
  "$schema": "https://v2-8-12.turborepo.dev/schema.json",
  "globalDependencies": [".oxfmtrc.json", ".oxlintrc.json"],
  "globalEnv": [
    "DEV",
    "NODE_ENV",
    "VITE_ADMIN_BASE_PATH",
    "VITE_ADMIN_BASE_URL",
    "VITE_API_BASE_PATH",
    "VITE_API_BASE_URL",
    "VITE_LIVE_BASE_PATH",
    "VITE_LIVE_BASE_URL",
    "VITE_SPACE_BASE_PATH",
    "VITE_SPACE_BASE_URL",
    "VITE_SUPPORT_EMAIL",
    "VITE_WEB_BASE_PATH",
    "VITE_WEB_BASE_URL",
    "VITE_WEBSITE_URL"
  ],
  "remoteCache": {
    "enabled": false
  },
  "tasks": {
    "build": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", ".env*"],
      "outputs": ["dist/**", "build/**", ".react-router/**"]
    },
    "build-storybook": {
      "dependsOn": ["^build"],
      "outputs": ["storybook-static/**"]
    },
    "check": {
      "dependsOn": ["check:format", "check:lint", "check:types"]
    },
    "check:format": {
      "inputs": ["$TURBO_DEFAULT$"],
      "outputs": []
    },
    "check:lint": {
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md"],
      "outputs": []
    },
    "check:types": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md"],
      "outputs": []
    },
    "clean": {
      "cache": false
    },
    "dev": {
      "cache": false,
      "dependsOn": ["^build"],
      "persistent": true
    },
    "fix": {
      "cache": false,
      "dependsOn": ["fix:format", "fix:lint"]
    },
    "fix:format": {
      "inputs": ["$TURBO_DEFAULT$"],
      "outputs": []
    },
    "fix:lint": {
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md"],
      "outputs": []
    },
    "start": {
      "cache": false,
      "persistent": true
    },
    "test": {
      "dependsOn": ["^build"],
      "outputs": []
    }
  }
}
```

Run: `PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc; diff <(git -C "$PLANE" show "$SHA:turbo.json") turbo.json`
Expected（只删掉 `.npmrc` 和 13 个环境变量）:

```
3c3
<   "globalDependencies": [".npmrc", ".oxfmtrc.json", ".oxlintrc.json"],
---
>   "globalDependencies": [".oxfmtrc.json", ".oxlintrc.json"],
5d4
<     "APP_VERSION",
7d5
<     "LOG_LEVEL",
9,11d6
<     "SENTRY_DSN",
<     "SENTRY_ENVIRONMENT",
<     "SENTRY_TRACES_SAMPLE_RATE",
16d10
<     "VITE_APP_VERSION",
19,25d12
<     "VITE_SENTRY_DSN",
<     "VITE_SENTRY_ENVIRONMENT",
<     "VITE_SENTRY_PROFILES_SAMPLE_RATE",
<     "VITE_SENTRY_REPLAYS_ON_ERROR_SAMPLE_RATE",
<     "VITE_SENTRY_REPLAYS_SESSION_SAMPLE_RATE",
<     "VITE_SENTRY_SEND_DEFAULT_PII",
<     "VITE_SENTRY_TRACES_SAMPLE_RATE",
```

- [ ] **Step 6: 修改 `web/packages/api-client/package.json`（`typescript` 改用 catalog，P3 交接第 4 条）**

把：

```json
    "typescript": "5.8.3"
```

替换为：

```json
    "typescript": "catalog:"
```

- [ ] **Step 7: 修改 `.gitignore`（前端的生成物和构建产物）**

把：

```
coverage/
*.test
*.out
```

替换为：

```
coverage/
*.test
*.out

# 前端的生成物和构建产物
web/apps/web/.react-router/
web/apps/web/build/
web/packages/*/dist/
web/packages/i18n/src/types/keys.generated.ts
```

- [ ] **Step 8: 由 Plane 的锁文件生成 `pnpm-lock.yaml`**

以 Plane 的锁文件为起点，把 importers 的键移到 `web/` 下，再由 pnpm 删掉没有迁入的 importers、加入 Nerve 自己的依赖（spec 2.5）：

```bash
PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc
git -C "$PLANE" show "$SHA:pnpm-lock.yaml" | sed -E 's#^  (apps|packages)/([a-z0-9-]+):#  web/\1/\2:#' > pnpm-lock.yaml
pnpm install
```

Expected: 输出中有 `Packages: +1139`（从 P3 的依赖开始安装时数字不同，这没有关系）、`[WARN] Issues with peer dependencies found`（Plane 本身就有）和 `Done in …`。

Run: `wc -l pnpm-lock.yaml && shasum -a 256 pnpm-lock.yaml`
Expected（写作时的值，见 Global Constraints）:

```
   13835 pnpm-lock.yaml
d677458d62ab06d55c0975b3e571bb0620e49330cf041581cbadaf5fa71d18ca  pnpm-lock.yaml
```

锁文件已经稳定，再安装一次不会改变它：

Run: `pnpm install && pnpm install --frozen-lockfile && shasum -a 256 pnpm-lock.yaml`
Expected: 两次都是 `Done in …`，SHA-256 与上面相同。

核对迁入的 13 个包的解析结果与 Plane 的锁文件逐行相同：

```bash
PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc
plane_importers() { awk '/^importers:/{f=1;next} /^packages:/{f=0} f' | awk '/^  [^ ]/{keep = ($1 ~ /^web\/(apps\/web|packages\/(constants|editor|hooks|i18n|propel|services|shared-state|tailwind-config|types|typescript-config|ui|utils)):$/)} keep'; }
diff <(git -C "$PLANE" show "$SHA:pnpm-lock.yaml" | sed -E 's#^  (apps|packages)/([a-z0-9-]+):#  web/\1/\2:#' | plane_importers) <(plane_importers < pnpm-lock.yaml) && echo "Plane 的 13 个包：解析结果与 Plane 的锁文件相同"
plane_importers < pnpm-lock.yaml | grep -c '^  web/'
```

Expected:

```
Plane 的 13 个包：解析结果与 Plane 的锁文件相同
13
```

- [ ] **Step 9: 确认接口描述的生成物不受影响**

Plane 的覆盖项作用于整个依赖图，Redocly 间接依赖的 brace-expansion 从 2.1.7 换成了 5.0.9（spec 2.5）。

Run: `make gen-check-web`
Expected: 退出码 0，`git status --short -- api/dist web/packages/api-client/src/schema.gen.ts` 没有输出。

- [ ] **Step 10: 构建前端**

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run build --filter=web --output-logs=errors-only`
Expected（输出末尾；本机约 20 秒）:

```
 Tasks:    11 successful, 11 total
Cached:    0 cached, 11 total
```

Run: `ls -1 web/apps/web/build/client && find web/apps/web/build/client -type f | wc -l`
Expected（约 34 MB，其中 `assets/` 下 1226 个带哈希的文件）:

```
assets
favicon
icons
index.html
manifest.json
site.webmanifest.json
sw.js
sw.js.map
workbox-9f2f79cf.js
workbox-9f2f79cf.js.map
    1239
```

- [ ] **Step 11: 核对改动范围，确认复制的内容与 Plane 相同**

```bash
git add .gitignore package.json pnpm-workspace.yaml pnpm-lock.yaml turbo.json patches web
git status --short | grep -v '^A  web/'
git status --short | wc -l
```

Expected（构建产物都被 `.gitignore` 挡掉，没有 `??` 的行）:

```
M  .gitignore
M  package.json
A  patches/react-color@2.19.3.patch
M  pnpm-lock.yaml
M  pnpm-workspace.yaml
A  turbo.json
M  web/packages/api-client/package.json
    4067
```

逐个比较暂存区中的目录与 Plane 中对应的 git 树对象：

```bash
PLANE="$(git rev-parse --path-format=absolute --git-common-dir)/../plane"; SHA=02c19e1341d93141e8ad7b3278298adce208bafc
T=$(git write-tree)
for d in apps/web packages/constants packages/editor packages/hooks packages/i18n packages/propel packages/services packages/shared-state packages/tailwind-config packages/types packages/typescript-config packages/ui packages/utils; do
  if [ "$(git rev-parse "$T:web/$d")" = "$(git -C "$PLANE" rev-parse "$SHA:$d")" ]; then echo "same  $d"; else echo "DIFF  $d"; fi
done
[ "$(git rev-parse "$T:patches")" = "$(git -C "$PLANE" rev-parse "$SHA:patches")" ] && echo "same  patches"
```

Expected: 14 行 `same`，没有 `DIFF`：

```
same  apps/web
same  packages/constants
same  packages/editor
same  packages/hooks
same  packages/i18n
same  packages/propel
same  packages/services
same  packages/shared-state
same  packages/tailwind-config
same  packages/types
same  packages/typescript-config
same  packages/ui
same  packages/utils
same  patches
```

Go 代码没有改动，不需要执行 `make lint-go` 和 `make test`。`make lint-web` 在 Task 2 改为 turbo 驱动后再执行。

- [ ] **Step 12: 提交**

```bash
git commit -m "feat(web): import the Plane web app and its packages from 02c19e1

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: lint 警告基线；类型检查、lint、格式检查改由 turbo 驱动

**Files:**
- Create: `.oxlintrc.json`、`.oxfmtrc.json`
- Modify: `web/apps/web/package.json`、`web/packages/{editor,i18n,propel,ui,utils}/package.json`（各一行）、`web/packages/api-client/package.json`、`Makefile`

**Interfaces:**
- Consumes: Task 1 的工作区和 `turbo.json` 中的 `check:types`、`check:lint`、`check:format`
- Produces:
  - `make lint-web` = `turbo run check:types check:lint check:format`（46 个任务），持续集成的 `web` 任务调用
  - Makefile 变量 `TURBO`、`TURBO_QUIET`，Task 3、5 使用

- [ ] **Step 1: 写入 `.oxlintrc.json`（Plane 的原文，`ignorePatterns` 末尾加入生成的 TS 类型）**

```json
{
  "$schema": "./node_modules/oxlint/configuration_schema.json",
  "plugins": ["react", "typescript", "jsx-a11y", "import", "promise", "unicorn", "oxc"],
  "categories": {
    "correctness": "warn",
    "suspicious": "warn",
    "perf": "warn"
  },
  "env": {
    "browser": true,
    "node": true,
    "es2024": true
  },
  "settings": {
    "react": {
      "version": "19.0"
    },
    "jsx-a11y": {
      "polymorphicPropName": "as"
    }
  },
  "ignorePatterns": [
    ".cache/**",
    ".next/**",
    ".react-router/**",
    ".storybook/**",
    ".turbo/**",
    ".vite/**",
    "*.config.{js,mjs,cjs,ts}",
    "build/**",
    "coverage/**",
    "dist/**",
    "**/public/**",
    "storybook-static/**",
    "web/packages/api-client/src/schema.gen.ts"
  ],
  "rules": {
    "react/react-in-jsx-scope": "off",
    "react/prop-types": "off",
    "unicorn/filename-case": "off",
    "unicorn/no-null": "off",
    "unicorn/prevent-abbreviations": "off",
    "no-unused-vars": [
      "warn",
      {
        "argsIgnorePattern": "^_",
        "varsIgnorePattern": "^_",
        "caughtErrorsIgnorePattern": "^_",
        "destructuredArrayIgnorePattern": "^_",
        "ignoreRestSiblings": true
      }
    ]
  }
}
```

- [ ] **Step 2: 写入 `.oxfmtrc.json`**

与 Plane 的原文相比：Tailwind 样式表改为工作区中的路径（不改时 oxfmt 检查 web 会崩溃）；删掉只作用于 `packages/codemods` 的覆盖项；排除生成的文件（spec 2.7）。

```json
{
  "printWidth": 120,
  "tabWidth": 2,
  "trailingComma": "es5",
  "sortTailwindcss": {
    "stylesheet": "web/packages/tailwind-config/index.css",
    "functions": ["cn", "clsx", "cva"]
  },
  "ignorePatterns": [
    "api/dist/**",
    "web/packages/api-client/src/schema.gen.ts",
    "web/packages/i18n/src/types/keys.generated.ts"
  ]
}
```

- [ ] **Step 3: 测出每个包当前的警告数**

Run:

```bash
for d in web/apps/web web/packages/*; do
  printf '%-32s %-40s %s\n' "$d" "$(node -p "require('./$d/package.json').scripts?.['check:lint'] ?? '-'")" "$(cd "$d" && pnpm exec oxlint . 2>/dev/null | grep '^Found')"
done
```

Expected:

```
web/apps/web                     oxlint --max-warnings=11957 .            Found 779 warnings and 0 errors.
web/packages/api-client          -                                        Found 0 warnings and 0 errors.
web/packages/constants           oxlint --max-warnings=2 .                Found 2 warnings and 0 errors.
web/packages/editor              oxlint --max-warnings=416 .              Found 75 warnings and 0 errors.
web/packages/hooks               oxlint --max-warnings=4 .                Found 4 warnings and 0 errors.
web/packages/i18n                oxlint --max-warnings=9 .                Found 3 warnings and 0 errors.
web/packages/propel              oxlint --max-warnings=3605 .             Found 59 warnings and 0 errors.
web/packages/services            oxlint --max-warnings=6 .                Found 6 warnings and 0 errors.
web/packages/shared-state        oxlint --max-warnings=0 .                Found 0 warnings and 0 errors.
web/packages/tailwind-config     -                                        Found 0 warnings and 0 errors.
web/packages/types               oxlint --max-warnings=1 .                Found 1 warning and 0 errors.
web/packages/typescript-config   -                                        Found 0 warnings and 0 errors.
web/packages/ui                  oxlint --max-warnings=66 .               Found 32 warnings and 0 errors.
web/packages/utils               oxlint --max-warnings=38 .               Found 34 warnings and 0 errors.
```

第二列是 Plane 的上限，第三列是实测的警告数。数字对不上时停下来：说明配置或 oxlint 的版本与本计划不同。

- [ ] **Step 4: 把 6 个包的上限调低到实测的警告数（M0 设计 5.2：只降不升）**

在各自的 `package.json` 中，把左边的一行替换为右边的一行：

| 文件 | 把 | 替换为 |
|---|---|---|
| `web/apps/web/package.json` | `"check:lint": "oxlint --max-warnings=11957 .",` | `"check:lint": "oxlint --max-warnings=779 .",` |
| `web/packages/editor/package.json` | `"check:lint": "oxlint --max-warnings=416 .",` | `"check:lint": "oxlint --max-warnings=75 .",` |
| `web/packages/i18n/package.json` | `"check:lint": "oxlint --max-warnings=9 .",` | `"check:lint": "oxlint --max-warnings=3 .",` |
| `web/packages/propel/package.json` | `"check:lint": "oxlint --max-warnings=3605 .",` | `"check:lint": "oxlint --max-warnings=59 .",` |
| `web/packages/ui/package.json` | `"check:lint": "oxlint --max-warnings=66 .",` | `"check:lint": "oxlint --max-warnings=32 .",` |
| `web/packages/utils/package.json` | `"check:lint": "oxlint --max-warnings=38 .",` | `"check:lint": "oxlint --max-warnings=34 .",` |

Run: `git diff --stat`
Expected:

```
 web/apps/web/package.json        | 2 +-
 web/packages/editor/package.json | 2 +-
 web/packages/i18n/package.json   | 2 +-
 web/packages/propel/package.json | 2 +-
 web/packages/ui/package.json     | 2 +-
 web/packages/utils/package.json  | 2 +-
 6 files changed, 6 insertions(+), 6 deletions(-)
```

- [ ] **Step 5: 写入 `web/packages/api-client/package.json`（完整内容：新增 `check:lint`、`check:format`；`license` 按 oxfmt 的键顺序移到 `description` 之后）**

```json
{
  "name": "@nerve/api-client",
  "private": true,
  "description": "Typed client for the Nerve API, generated from api/dist/openapi.yaml",
  "license": "AGPL-3.0-only",
  "type": "module",
  "exports": {
    ".": "./src/index.ts"
  },
  "scripts": {
    "gen": "openapi-typescript ../../../api/dist/openapi.yaml --output src/schema.gen.ts",
    "check:types": "tsc --noEmit",
    "check:lint": "oxlint --max-warnings=0 .",
    "check:format": "oxfmt --check ."
  },
  "dependencies": {
    "openapi-fetch": "0.17.0"
  },
  "devDependencies": {
    "openapi-typescript": "7.13.0",
    "typescript": "catalog:"
  }
}
```

- [ ] **Step 6: 修改 `Makefile`（turbo 的变量）**

把：

```makefile
check-committed = test -z "$$(git status --porcelain -- $(1))" || { git status --short -- $(1); git --no-pager diff -- $(1); echo "生成物与接口描述不一致：执行 make gen，并提交生成的文件"; exit 1; }
```

替换为：

```makefile
check-committed = test -z "$$(git status --porcelain -- $(1))" || { git status --short -- $(1); git --no-pager diff -- $(1); echo "生成物与接口描述不一致：执行 make gen，并提交生成的文件"; exit 1; }

# 前端任务由 turbo 按 turbo.json 编排；关闭匿名使用数据上报，只打印失败任务的输出
TURBO := TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo
TURBO_QUIET := --output-logs=errors-only
```

- [ ] **Step 7: 修改 `Makefile`（`lint-web` 改由 turbo 驱动，P3 交接第 2 条）**

把：

```makefile
.PHONY: lint-web
lint-web: ## 前端类型检查（需要 Node）
	pnpm -r run check:types
```

替换为：

```makefile
.PHONY: lint-web
lint-web: ## 前端类型检查、oxlint（按警告基线）、格式检查（需要 Node）
	$(TURBO) run check:types check:lint check:format $(TURBO_QUIET)
```

Run: `grep -c "$(printf '\t')" Makefile`
Expected: `25`

- [ ] **Step 8: 执行 `make lint-web`，确认 api-client 仍在类型检查的范围内**

Run: `rm -rf .turbo && make lint-web`
Expected（输出末尾；本机约 35 秒）:

```
 Tasks:    46 successful, 46 total
Cached:    0 cached, 46 total
```

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:types --dry=json 2>/dev/null | node -e 'const t=JSON.parse(require("fs").readFileSync(0,"utf8")).tasks.filter(x=>x.task==="check:types"&&x.command!=="<NONEXISTENT>").map(x=>x.taskId); console.log(t.length, t.join(" "))'`
Expected:

```
12 @nerve/api-client#check:types @plane/constants#check:types @plane/editor#check:types @plane/hooks#check:types @plane/i18n#check:types @plane/propel#check:types @plane/services#check:types @plane/shared-state#check:types @plane/types#check:types @plane/ui#check:types @plane/utils#check:types web#check:types
```

- [ ] **Step 9: 确认警告数超过上限时 `make lint-web` 失败**

临时写入 `web/apps/web/core/lint-probe.ts`：

```ts
export function lintProbe(): void {
  const unused = 1;
}
```

Run: `make lint-web`
Expected: 失败（退出码 2），输出中有：

```
web:check:lint: Exceeded maximum number of warnings. Found 780.
Failed:    web#check:lint
```

Run: `rm web/apps/web/core/lint-probe.ts && make lint-web`
Expected: 通过，全部命中缓存：

```
Cached:    46 cached, 46 total
```

- [ ] **Step 10: 核对改动范围**

Run: `git status --short`
Expected:

```
 M Makefile
 M web/apps/web/package.json
 M web/packages/api-client/package.json
 M web/packages/editor/package.json
 M web/packages/i18n/package.json
 M web/packages/propel/package.json
 M web/packages/ui/package.json
 M web/packages/utils/package.json
?? .oxfmtrc.json
?? .oxlintrc.json
```

- [ ] **Step 11: 提交**

```bash
git add .oxlintrc.json .oxfmtrc.json Makefile web/apps/web/package.json web/packages/editor/package.json web/packages/i18n/package.json web/packages/propel/package.json web/packages/ui/package.json web/packages/utils/package.json web/packages/api-client/package.json
git commit -m "build(web): lint against a warning baseline and drive web checks with turbo

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Vite 开发代理、`make web-dev`；登记前端改动

**Files:**
- Modify: `web/apps/web/vite.config.ts`、`Makefile`、`docs/v0/frontend-changes.md`

**Interfaces:**
- Consumes: Task 2 的 `TURBO`；`make run`（P2）
- Produces: `make web-dev`：http://127.0.0.1:3000 ，`/api` 转发给 `http://127.0.0.1:8080`

- [ ] **Step 1: 修改 `web/apps/web/vite.config.ts`（Plane 文件中唯一的代码改动）**

把：

```ts
  server: {
    host: "127.0.0.1",
  },
```

替换为：

```ts
  server: {
    host: "127.0.0.1",
    // Nerve: during development the Go server (`make run`) answers the API.
    proxy: {
      "/api": "http://127.0.0.1:8080",
    },
  },
```

- [ ] **Step 2: 修改 `Makefile`（新增 `web-dev`）**

把：

```makefile
.PHONY: run
run: ## 以 dev 配置运行后端（需先 make dev-db），Ctrl-C 停止
	cd server && NERVE_ENV=dev go run ./cmd/nerve serve
```

替换为：

```makefile
.PHONY: run
run: ## 以 dev 配置运行后端（需先 make dev-db），Ctrl-C 停止
	cd server && NERVE_ENV=dev go run ./cmd/nerve serve

.PHONY: web-dev
web-dev: ## 启动前端开发服务器 http://127.0.0.1:3000，/api 转发给 make run 的后端；Ctrl-C 停止
	$(TURBO) run dev --filter=web
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected:

```
26
18
```

- [ ] **Step 3: 修改 `docs/v0/frontend-changes.md`（来源提交和迁入时的改动）**

把：

```markdown
| 项 | 内容 |
|---|---|
| 使用 | `apps/web`；packages 中的 types、constants、ui、propel、editor、i18n、hooks、utils、shared-state、tailwind-config、typescript-config |
| 暂时使用 | `packages/services`：web 的令牌设置页和文件工具函数依赖它。M2（PAT）和 M5（文件）重写对应的接口调用后，将它删除 |
| 不使用 | `apps/admin`、`apps/space`、`apps/live`、`apps/api`、`apps/proxy`、`packages/logger`、`packages/decorators`、`packages/codemods`（已核实 web 及其依赖的包都不引用它们） |
```

替换为：

```markdown
| 项 | 内容 |
|---|---|
| 来源提交 | Plane `02c19e1341d93141e8ad7b3278298adce208bafc`（`preview` 分支，`package.json` 中的版本是 1.4.2） |
| 迁入方式 | M0/P5 用 `git archive` 按上面的完整提交复制。迁入时 13 个目录与 Plane 中对应目录的 git 树对象完全相同，之后的每一处改动都登记在本清单中（见 [P5 spec](M0-foundation/specs/P5-web-import.md) 2.3） |
| 使用 | `apps/web` → `web/apps/web`；packages 中的 types、constants、ui、propel、editor、i18n、hooks、utils、shared-state、tailwind-config、typescript-config → `web/packages/<包名>`；`patches/react-color@2.19.3.patch` → 仓库根目录的 `patches/` |
| 暂时使用 | `packages/services`：web 的令牌设置页和文件工具函数依赖它。M2（PAT）和 M5（文件）重写对应的接口调用后，将它删除 |
| 不使用 | `apps/admin`、`apps/space`、`apps/live`、`apps/api`、`apps/proxy`、`packages/logger`、`packages/decorators`、`packages/codemods`（已核实 web 及其依赖的包都不引用它们）；根目录的 `.npmrc`（pnpm 11 只从 `.npmrc` 读取认证和仓库地址，其中的其他设置都不起作用）；husky、lint-staged（Git 钩子）和 react-doctor |

### 1.1 迁入时的改动（M0/P5）

只做让前端能安装、检查、构建和开发的最小改动，没有功能改动。

| 位置 | 改动 | 原因 |
|---|---|---|
| `pnpm-workspace.yaml`（仓库根目录） | 工作区的路径改为 `web/apps/*`、`web/packages/*`、`e2e`；catalog、overrides、allowBuilds 只保留作用于迁入的包的条目（删掉 39 个 catalog 条目、14 个 overrides、3 个 allowBuilds） | 工作区的路径；删掉的条目只属于没有迁入的应用和包，或者不作用于迁入的包的依赖 |
| `pnpm-lock.yaml`（仓库根目录） | 以 Plane 的锁文件为起点，把 importers 的路径移到 `web/` 下，再由 pnpm 加入 Nerve 自己的依赖 | 迁入的 13 个包的依赖版本与 Plane 完全相同 |
| `package.json`（仓库根目录） | 只加入开发依赖 `oxfmt`、`oxlint`、`turbo`（`catalog:`）；不加 husky、lint-staged、react-doctor，不加脚本 | 命令的入口是 Makefile |
| `turbo.json`（仓库根目录） | `globalDependencies` 去掉 `.npmrc`；`globalEnv` 去掉没有代码读取的 `APP_VERSION`、`LOG_LEVEL`、`VITE_APP_VERSION` 和 10 个 Sentry 变量 | 没有迁入 `.npmrc`；这些变量只属于没有迁入的应用 |
| `.oxlintrc.json`（仓库根目录） | `ignorePatterns` 加入 `web/packages/api-client/src/schema.gen.ts` | 生成的文件不做 lint |
| `.oxfmtrc.json`（仓库根目录） | `sortTailwindcss.stylesheet` 改为 `web/packages/tailwind-config/index.css`；删掉 `packages/codemods` 的覆盖项；加入 `ignorePatterns`：`api/dist/**`、`web/packages/api-client/src/schema.gen.ts`、`web/packages/i18n/src/types/keys.generated.ts` | 工作区的路径（路径不对时，oxfmt 检查 web 会崩溃）；codemods 没有迁入；生成的文件不检查格式 |
| `web/apps/web/package.json`，editor、i18n、propel、ui、utils 的 `package.json` | `check:lint` 的 `--max-warnings` 调低到实测的警告数：web 11957→779，editor 416→75，i18n 9→3，propel 3605→59，ui 66→32，utils 38→34 | lint 警告基线只降不升（M0 设计 5.2） |
| `web/apps/web/vite.config.ts` | 开发服务器加上代理：`/api` 转发到 `http://127.0.0.1:8080` | 开发时由 Go 后端（`make run`）回答接口请求 |
```

把：

```markdown
| `web/` 中来自 Plane 的文件保留原有的版权声明 | 计划中 | |
```

替换为：

```markdown
| `web/` 中来自 Plane 的文件保留原有的版权声明 | 已完成 | M0/P5 |
```

- [ ] **Step 4: 前端检查仍然通过**

`vite.config.ts` 不在 oxlint 的检查范围内（`*.config.{js,mjs,cjs,ts}`），但要符合 oxfmt 的格式。

Run: `make lint-web`
Expected: 通过；web 的 3 个任务重新执行，其余命中缓存：

```
Cached:    43 cached, 46 total
```

- [ ] **Step 5: 验证开发代理**

```bash
make dev-db
make run > "${TMPDIR:-/tmp}/nerve-run.log" 2>&1 &
for i in $(seq 1 60); do curl -s -o /dev/null http://127.0.0.1:8080/healthz && break; sleep 1; done
make web-dev > "${TMPDIR:-/tmp}/nerve-web-dev.log" 2>&1 &
for i in $(seq 1 120); do curl -s -o /dev/null http://127.0.0.1:3000/ && break; sleep 1; done
curl -s -i http://127.0.0.1:3000/api/v0/instance | grep -iE '^HTTP|^content-type|api_version'
curl -s -i http://127.0.0.1:3000/api/v0/nope | grep -iE '^HTTP|^content-type|not_found'
curl -s -o /dev/null -w '%{http_code} %{content_type}\n' http://127.0.0.1:3000/acme/projects
```

Expected（`/api` 由 Go 服务回答，其他路径由 Vite 返回页面）:

```
HTTP/1.1 200 OK
content-type: application/json
{"api_version":"v0","commit":"unknown","product":"Nerve","version":"0.1.0-dev"}
HTTP/1.1 404 Not Found
content-type: application/problem+json
{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}
200 text/html
```

（对照：不加代理时，前两个请求都返回 Vite 的 `index.html`。）

停止两个后台进程，确认端口已释放：

```bash
pkill -f 'turbo run dev --filter=web'; pkill -f 'react-router dev'; pkill -f 'nerve serve'
sleep 3; lsof -iTCP:8080 -sTCP:LISTEN; lsof -iTCP:3000 -sTCP:LISTEN
```

Expected: 两个 `lsof` 都没有输出。

- [ ] **Step 6: 提交**

Run: `git status --short`
Expected:

```
 M Makefile
 M docs/v0/frontend-changes.md
 M web/apps/web/vite.config.ts
```

```bash
git add Makefile web/apps/web/vite.config.ts docs/v0/frontend-changes.md
git commit -m "feat(web): proxy /api to the Go server in the Vite dev server

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: `platform/webui` 与挂载

**Files:**
- Create: `server/internal/platform/webui/embed.go`、`server/internal/platform/webui/handler.go`、`server/internal/platform/webui/dist/.gitkeep`
- Test: `server/internal/platform/webui/handler_test.go`、`server/internal/platform/webui/embed_test.go`、`server/internal/bootstrap/webui_test.go`
- Modify: `.gitignore`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/commands.go`、`server/internal/bootstrap/app_test.go`

**Interfaces:**
- Consumes: 标准库（`embed`、`io/fs`、`net/http`、`path`、`strings`）；`bootstrap` 中已有的 `newApp`、`startApp`、`testConfig`、`unreachableDB`、`client`
- Produces:
  - `func webui.FS() fs.FS`：内嵌的 `dist/`，构建后根目录下有 `index.html`
  - `func webui.Handler(files fs.FS) http.Handler`：GET/HEAD；文件、`assets/` 的 404、其余回退到 `index.html`；没有 `index.html` 时 404 加提示（spec 2.9）
  - `newApp(ctx, cfg, logger, migrationFiles, webFiles fs.FS)`：把 `webui.Handler(webFiles)` 注册在不带方法的 `/` 上；`Serve` 传入 `webui.FS()`
  - Task 5 的 `make build` 把前端复制进 `dist/`

- [ ] **Step 1: 修改 `.gitignore`，建立 `dist/.gitkeep`**

把：

```
web/packages/i18n/src/types/keys.generated.ts
```

替换为：

```
web/packages/i18n/src/types/keys.generated.ts

# 内嵌的前端：只提交 .gitkeep，其余由 make build 复制进来
server/internal/platform/webui/dist/*
!server/internal/platform/webui/dist/.gitkeep
```

Run: `mkdir -p server/internal/platform/webui/dist && touch server/internal/platform/webui/dist/.gitkeep`

- [ ] **Step 2: 写入失败的测试 `server/internal/platform/webui/handler_test.go`**

```go
package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const indexHTML = "<!doctype html><title>Nerve</title>"

// built mirrors the layout of web/apps/web/build/client.
var built = fstest.MapFS{
	".gitkeep":                  {},
	"index.html":                {Data: []byte(indexHTML)},
	"manifest.json":             {Data: []byte(`{"name":"Nerve"}`)},
	"favicon/android-192.png":   {Data: []byte("\x89PNG\r\n\x1a\n")},
	"assets/entry.client-a1.js": {Data: []byte("export {};")},
	"assets/globals-b2.css":     {Data: []byte("body{}")},
	"assets/.hidden":            {Data: []byte("secret")},
}

func serve(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

// wantResponse checks status, Content-Type, Cache-Control and body.
func wantResponse(t *testing.T, rec *httptest.ResponseRecorder, target string, status int, contentType, cacheControl, body string) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("%s: status = %d, want %d", target, rec.Code, status)
	}
	if got := rec.Header().Get("Content-Type"); got != contentType {
		t.Errorf("%s: Content-Type = %q, want %q", target, got, contentType)
	}
	if got := rec.Header().Get("Cache-Control"); got != cacheControl {
		t.Errorf("%s: Cache-Control = %q, want %q", target, got, cacheControl)
	}
	if got := rec.Body.String(); got != body {
		t.Errorf("%s: body = %q, want %q", target, got, body)
	}
}

func TestServesIndexAtRoot(t *testing.T) {
	rec := serve(Handler(built), http.MethodGet, "/")

	wantResponse(t, rec, "/", http.StatusOK, "text/html; charset=utf-8", "no-cache", indexHTML)
}

func TestPagePathsFallBackToIndex(t *testing.T) {
	h := Handler(built)
	for _, target := range []string{
		"/sign-up",
		"/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues",
		"/acme/settings/",
		"/favicon",       // a directory, not a file
		"/.gitkeep",      // hidden files are never served
		"/favicon.ico",   // no such file outside assets/
		"/acme?tab=home", // the query does not matter
	} {
		wantResponse(t, serve(h, http.MethodGet, target), target, http.StatusOK, "text/html; charset=utf-8", "no-cache", indexHTML)
	}
}

func TestServesFilesWithCachePolicy(t *testing.T) {
	h := Handler(built)
	tests := []struct {
		target, contentType, cacheControl, body string
	}{
		// Vite content-hashes everything under assets/: a changed file gets a new name.
		{"/assets/entry.client-a1.js", "text/javascript; charset=utf-8", "public, max-age=31536000, immutable", "export {};"},
		{"/assets/globals-b2.css", "text/css; charset=utf-8", "public, max-age=31536000, immutable", "body{}"},
		// Files copied from public/ keep their names, so they are revalidated.
		{"/manifest.json", "application/json", "no-cache", `{"name":"Nerve"}`},
		{"/favicon/android-192.png", "image/png", "no-cache", "\x89PNG\r\n\x1a\n"},
	}
	for _, tt := range tests {
		wantResponse(t, serve(h, http.MethodGet, tt.target), tt.target, http.StatusOK, tt.contentType, tt.cacheControl, tt.body)
	}
}

// A missing hashed file (for example a chunk of an older build) must fail as
// a missing file: answering index.html would hand HTML to a script loader.
func TestMissingAssetsAre404(t *testing.T) {
	h := Handler(built)
	for _, target := range []string{"/assets/entry.client-old.js", "/assets/", "/assets", "/assets/.hidden"} {
		rec := serve(h, http.MethodGet, target)
		if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), indexHTML) {
			t.Errorf("GET %s = %d %q, want 404 without index.html", target, rec.Code, rec.Body)
		}
	}
}

func TestIndexHTMLRedirectsToRoot(t *testing.T) {
	rec := serve(Handler(built), http.MethodGet, "/index.html")

	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "./" {
		t.Errorf("GET /index.html = %d Location %q, want 301 to ./", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHeadHasHeadersButNoBody(t *testing.T) {
	h := Handler(built)
	for _, target := range []string{"/", "/acme/projects", "/assets/entry.client-a1.js"} {
		rec := serve(h, http.MethodHead, target)
		if rec.Code != http.StatusOK || rec.Body.Len() != 0 || rec.Header().Get("Content-Length") == "" {
			t.Errorf("HEAD %s = %d, body %d bytes, Content-Length %q; want 200, no body, a length",
				target, rec.Code, rec.Body.Len(), rec.Header().Get("Content-Length"))
		}
	}
}

func TestOnlyGetAndHeadAreAllowed(t *testing.T) {
	for _, files := range []fs.FS{built, fstest.MapFS{}} {
		h := Handler(files)
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions} {
			rec := serve(h, method, "/assets/entry.client-a1.js")
			if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("%s = %d Allow %q, want 405 Allow \"GET, HEAD\"", method, rec.Code, rec.Header().Get("Allow"))
			}
		}
	}
}

// Without index.html (dist/ holds only .gitkeep) every page explains how to
// get the frontend.
func TestUnbuiltFrontendExplainsItself(t *testing.T) {
	h := Handler(fstest.MapFS{".gitkeep": {}})
	for _, target := range []string{"/", "/acme/projects", "/assets/entry.client-a1.js"} {
		wantResponse(t, serve(h, http.MethodGet, target), target, http.StatusNotFound, "text/plain; charset=utf-8", "no-cache", notBuiltMessage+"\n")
	}
}
```

- [ ] **Step 3: 写入失败的测试 `server/internal/platform/webui/embed_test.go`**

```go
package webui

import (
	"io/fs"
	"testing"
)

func TestFSIsTheDistDirectory(t *testing.T) {
	if _, err := fs.Stat(FS(), ".gitkeep"); err != nil {
		t.Errorf("FS() lacks .gitkeep, so it is not dist/: %v", err)
	}
}
```

Run: `cd server && go test ./internal/platform/webui/`
Expected: 编译失败，包括：

```
internal/platform/webui/embed_test.go:9:23: undefined: FS
internal/platform/webui/handler_test.go:49:15: undefined: Handler
```

- [ ] **Step 4: 写入 `server/internal/platform/webui/embed.go`**

```go
// Package webui serves the web frontend embedded in the nerve binary: the
// static files of the single-page app, and its index.html for every page path.
package webui

import (
	"embed"
	"io/fs"
)

// dist holds the frontend that `make build` copies from
// web/apps/web/build/client. Only dist/.gitkeep is committed; "all:" embeds
// it, so the pattern still matches while the frontend is not built.
//
//go:embed all:dist
var dist embed.FS

// FS returns the embedded frontend, with index.html at its root once built.
func FS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // unreachable: "dist" is a valid path embedded above
	}
	return sub
}
```

- [ ] **Step 5: 写入 `server/internal/platform/webui/handler.go`**

```go
package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const (
	indexFile = "index.html"
	// assetsDir holds Vite's content-hashed files: a changed file gets a new name.
	assetsDir = "assets"

	cacheImmutable  = "public, max-age=31536000, immutable"
	cacheRevalidate = "no-cache"

	notBuiltMessage = "The web frontend is not built into this binary: run `make build`, " +
		"or use the Vite dev server (`make web-dev`) during development."
)

// Handler serves the single-page app in files (GET and HEAD only):
//
//   - a file: served as is; files under assets/ are cached for good, others
//     (index.html, files copied from public/) are revalidated on every use;
//   - a missing path under assets/: 404, so a chunk of an older build fails
//     as a missing file instead of turning into HTML;
//   - any other path: index.html, and the client-side router renders the page.
//
// Hidden files (any name part starting with "."), such as dist/.gitkeep, are
// never served. Without index.html every request answers 404 with a hint.
// Mount it on the pattern "/" so that /api/ and the probes keep their routes.
func Handler(files fs.FS) http.Handler {
	_, err := fs.Stat(files, indexFile)
	return &handler{files: files, built: err == nil}
}

type handler struct {
	files fs.FS
	built bool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if !h.built {
		w.Header().Set("Cache-Control", cacheRevalidate)
		http.Error(w, notBuiltMessage, http.StatusNotFound)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "" {
		name = indexFile
	}
	switch {
	case h.isFile(name):
		h.serveFile(w, r, name)
	case name == assetsDir || strings.HasPrefix(name, assetsDir+"/"):
		http.NotFound(w, r)
	default:
		h.serveFile(w, r, indexFile)
	}
}

// isFile reports whether name is a regular, visible file in h.files.
func (h *handler) isFile(name string) bool {
	if strings.HasPrefix(name, ".") || strings.Contains(name, "/.") {
		return false
	}
	info, err := fs.Stat(h.files, name)
	return err == nil && info.Mode().IsRegular()
}

func (h *handler) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	cache := cacheRevalidate
	if strings.HasPrefix(name, assetsDir+"/") {
		cache = cacheImmutable
	}
	w.Header().Set("Cache-Control", cache)
	// ServeFileFS sets Content-Type from the extension (sniffing the content
	// otherwise), answers HEAD and Range, and redirects /index.html to ./.
	http.ServeFileFS(w, r, h.files, name)
}
```

Run: `cd server && go test -count=1 -v ./internal/platform/webui/ | grep -E '^(--- |ok|FAIL)'`
Expected:

```
--- PASS: TestFSIsTheDistDirectory (0.00s)
--- PASS: TestServesIndexAtRoot (0.00s)
--- PASS: TestPagePathsFallBackToIndex (0.00s)
--- PASS: TestServesFilesWithCachePolicy (0.00s)
--- PASS: TestMissingAssetsAre404 (0.00s)
--- PASS: TestIndexHTMLRedirectsToRoot (0.00s)
--- PASS: TestHeadHasHeadersButNoBody (0.00s)
--- PASS: TestOnlyGetAndHeadAreAllowed (0.00s)
--- PASS: TestUnbuiltFrontendExplainsItself (0.00s)
ok  	github.com/open-nerve/NerveProject/server/internal/platform/webui	…
```

- [ ] **Step 6: 写入失败的测试 `server/internal/bootstrap/webui_test.go`**

```go
package bootstrap

import (
	"io"
	"net/http"
	"testing"
	"testing/fstest"
)

// The web UI needs no database: the app runs against an unreachable one.
// startApp serves testWebUI.
func TestServesTheWebUIOnEveryOtherPath(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, path := range []string{"/", "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues"} {
		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusOK || string(body) != testIndexHTML {
			t.Errorf("GET %s = %d %q, want 200 with index.html", path, res.StatusCode, body)
		}
	}
}

// Mounting the web UI on "/" keeps the probes: /healthz still answers JSON.
func TestProbesAreNotTheWebUI(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	res, err := client.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("GET /healthz = %d %s, want 200 application/json", res.StatusCode, res.Header.Get("Content-Type"))
	}
}
```

- [ ] **Step 7: 修改 `server/internal/bootstrap/app_test.go`（测试用的前端；`newApp` 多一个参数）**

把：

```go
	"00002_probe_create_gadgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE gadgets (id bigint);\n-- +goose Down\nDROP TABLE gadgets;\n")},
}
```

替换为：

```go
	"00002_probe_create_gadgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE gadgets (id bigint);\n-- +goose Down\nDROP TABLE gadgets;\n")},
}

// testWebUI stands in for the built frontend, so tests do not depend on make build.
const testIndexHTML = "<!doctype html><title>Nerve test</title>"

var testWebUI = fstest.MapFS{"index.html": {Data: []byte(testIndexHTML)}}
```

把：

```go
// startApp runs the app until the test ends and returns its base URL once it
// answers /healthz. The cleanup checks that run shut down cleanly.
func startApp(t *testing.T, cfg config.Config, migrations fs.FS) string {
	t.Helper()
	a, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), migrations)
```

替换为：

```go
// startApp runs the app, serving testWebUI, until the test ends and returns
// its base URL once it answers /healthz. The cleanup checks that run shut
// down cleanly.
func startApp(t *testing.T, cfg config.Config, migrations fs.FS) string {
	t.Helper()
	a, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), migrations, testWebUI)
```

把：

```go
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, true), slog.New(slog.DiscardHandler), sampleMigrations)
```

替换为：

```go
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, true), slog.New(slog.DiscardHandler), sampleMigrations, testWebUI)
```

Run: `cd server && go test -count=1 ./internal/bootstrap/`
Expected: 编译失败：

```
internal/bootstrap/app_test.go:57:89: too many arguments in call to newApp
…
internal/bootstrap/app_test.go:133:126: too many arguments in call to newApp
```

- [ ] **Step 8: 修改 `server/internal/bootstrap/app.go`（挂载 `webui`）**

把：

```go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)
```

替换为：

```go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
)
```

把：

```go
// newApp wires the server described by cfg around the given migrations.
// close releases it.
func newApp(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles fs.FS) (*app, error) {
```

替换为：

```go
// newApp wires the server described by cfg around the given migrations and
// built web frontend. close releases it.
func newApp(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles, webFiles fs.FS) (*app, error) {
```

把：

```go
	instance.New().Register(mux, httpserver.NewAPIErrors(logger))
```

替换为：

```go
	instance.New().Register(mux, httpserver.NewAPIErrors(logger))
	// The web UI takes every path no other pattern claims. It must be the
	// method-less "/": "GET /" and the method-less "/api/" would conflict.
	mux.Handle("/", webui.Handler(webFiles))
```

- [ ] **Step 9: 修改 `server/internal/bootstrap/commands.go`（`nerve serve` 使用内嵌的前端）**

把：

```go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/migrations"
```

替换为：

```go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
	"github.com/open-nerve/NerveProject/server/migrations"
```

把：

```go
// Serve implements `nerve serve`: it logs the effective configuration (secrets
// masked) to logOut, applies pending migrations when database.auto_migrate is
// on, and serves HTTP until ctx is done.
```

替换为：

```go
// Serve implements `nerve serve`: it logs the effective configuration (secrets
// masked) to logOut, applies pending migrations when database.auto_migrate is
// on, and serves the API and the embedded web frontend until ctx is done.
```

把：

```go
	a, err := newApp(ctx, cfg, logger, migrations.FS())
```

替换为：

```go
	a, err := newApp(ctx, cfg, logger, migrations.FS(), webui.FS())
```

Run: `cd server && go test -count=1 -run 'WebUI|Probes|Instance|Unknown' -v ./internal/bootstrap/ | grep -E '^(--- |ok|FAIL)'`
Expected（这四个测试都不需要数据库）:

```
--- PASS: TestServesTheInstanceAPI (…)
--- PASS: TestUnknownAPIRequestsStillAnswerProblem404 (…)
--- PASS: TestServesTheWebUIOnEveryOtherPath (…)
--- PASS: TestProbesAreNotTheWebUI (…)
ok  	github.com/open-nerve/NerveProject/server/internal/bootstrap	…
```

- [ ] **Step 10: 确认架构测试守住 platform 包之间的边界**

临时写入 `server/internal/platform/webui/violation.go`：

```go
package webui

import "github.com/open-nerve/NerveProject/server/internal/platform/httpserver"

var _ = httpserver.CodeNotFound
```

Run: `cd server && go test -count=1 ./internal/archtest/`
Expected: 失败：

```
--- FAIL: TestRepositoryFollowsArchitectureRules (…)
    repo_test.go:60: internal/platform/webui imports internal/platform/httpserver: platform packages do not import each other, except config
```

Run: `rm server/internal/platform/webui/violation.go && cd server && go test -count=1 ./internal/archtest/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/archtest	…`

- [ ] **Step 11: lint 与全部测试**

Run: `make lint`
Expected: golangci-lint 输出 `0 issues.`，前端检查 `Tasks:    46 successful, 46 total`。

Run: `make test`
Expected: 所有包都是 `ok`，其中包括：

```
ok  	github.com/open-nerve/NerveProject/server/internal/archtest	…
ok  	github.com/open-nerve/NerveProject/server/internal/bootstrap	…
ok  	github.com/open-nerve/NerveProject/server/internal/platform/webui	…
```

Run: `cd server && grep -E '^(go|toolchain) ' go.mod`
Expected:

```
go 1.27
toolchain go1.27.1
```

- [ ] **Step 12: 提交**

Run: `git status --short --untracked-files=all`
Expected:

```
 M .gitignore
 M server/internal/bootstrap/app.go
 M server/internal/bootstrap/app_test.go
 M server/internal/bootstrap/commands.go
?? server/internal/bootstrap/webui_test.go
?? server/internal/platform/webui/dist/.gitkeep
?? server/internal/platform/webui/embed.go
?? server/internal/platform/webui/embed_test.go
?? server/internal/platform/webui/handler.go
?? server/internal/platform/webui/handler_test.go
```

```bash
git add .gitignore server/internal/platform/webui server/internal/bootstrap
git commit -m "feat(server): serve the embedded web frontend with the SPA fallback

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: `make build` 与本地验收

**Files:**
- Modify: `Makefile`

**Interfaces:**
- Consumes: Task 1 的构建产物 `web/apps/web/build/client/`；Task 2 的 `TURBO`、`TURBO_QUIET`；Task 4 的 `webui` 和 `dist/`
- Produces:
  - `make build-web`：只构建前端（只需要 Node），Task 6 的持续集成调用
  - `make build`：`bin/nerve`，内嵌前端（P6 的端到端测试使用）

- [ ] **Step 1: 构建之前，`make run` 的页面是一句提示**

```bash
make dev-db
make run > "${TMPDIR:-/tmp}/nerve-run.log" 2>&1 &
for i in $(seq 1 60); do curl -s -o /dev/null http://127.0.0.1:8080/healthz && break; sleep 1; done
curl -s -i http://127.0.0.1:8080/ | grep -vE '^(Date|X-Request-Id)'
pkill -f 'nerve serve'; sleep 2; lsof -iTCP:8080 -sTCP:LISTEN
```

Expected（`dist/` 中只有 `.gitkeep`；最后的 `lsof` 没有输出）:

```
HTTP/1.1 404 Not Found
Cache-Control: no-cache
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Content-Length: 130

The web frontend is not built into this binary: run `make build`, or use the Vite dev server (`make web-dev`) during development.
```

- [ ] **Step 2: 修改 `Makefile`（`dist/` 的位置）**

把：

```makefile
TURBO := TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo
TURBO_QUIET := --output-logs=errors-only
```

替换为：

```makefile
TURBO := TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo
TURBO_QUIET := --output-logs=errors-only
# make build 把前端的构建产物复制到这里，由 go:embed 编进 nerve
WEBUI_DIST := server/internal/platform/webui/dist
```

- [ ] **Step 3: 修改 `Makefile`（新增 `build`、`build-web`）**

把：

```makefile
.PHONY: test
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...
```

替换为：

```makefile
.PHONY: test
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...

.PHONY: build
build: build-web ## 构建前端并嵌入 Go 程序，编译出 bin/nerve（需要 Node 和 Go）
	find $(WEBUI_DIST) -mindepth 1 ! -name .gitkeep -delete
	cp -R web/apps/web/build/client/. $(WEBUI_DIST)/
	cd server && go build -o ../bin/nerve ./cmd/nerve

.PHONY: build-web
build-web: ## 构建前端，产物在 web/apps/web/build/client（需要 Node；持续集成 web 任务）
	$(TURBO) run build --filter=web $(TURBO_QUIET)
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected:

```
30
20
```

- [ ] **Step 4: `make build`**

Run: `rm -rf .turbo web/packages/*/dist web/apps/web/build && make build`
Expected（本机约 20 秒）:

```
TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run build --filter=web --output-logs=errors-only
…
 Tasks:    11 successful, 11 total
Cached:    0 cached, 11 total
…
find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete
cp -R web/apps/web/build/client/. server/internal/platform/webui/dist/
cd server && go build -o ../bin/nerve ./cmd/nerve
```

Run: `ls -l bin/nerve | awk '{print $5}' && find server/internal/platform/webui/dist -type f | wc -l && git status --short --untracked-files=all`
Expected（约 50 MB；1239 个前端文件加 `.gitkeep`；复制进来的文件都被 `.gitignore` 挡掉）:

```
50529714
    1240
 M Makefile
```

（`bin/nerve` 的字节数随 Go 版本和平台略有不同，约 50 MB 即可。）

- [ ] **Step 5: 用 `bin/nerve serve` 验收**

```bash
NERVE_ENV=dev bin/nerve serve > "${TMPDIR:-/tmp}/nerve-serve.log" 2>&1 &
for i in $(seq 1 30); do curl -s -o /dev/null http://127.0.0.1:8080/healthz && break; sleep 0.5; done
B=http://127.0.0.1:8080
H='^(HTTP|Cache-Control|Content-Type|Location|Allow)'
curl -s -D - -o /dev/null $B/ | grep -E "$H"
curl -s -D - -o /dev/null $B/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues | grep -E "$H"
A=$(curl -s $B/ | grep -o '/assets/entry.client-[A-Za-z0-9_-]*\.js' | head -1)
curl -s -D - -o /dev/null "$B$A" | grep -E "$H"
curl -s -o /dev/null -w '%{http_code}\n' $B/assets/entry.client-gone.js
curl -s -D - -o /dev/null -X POST $B/ | grep -E "$H"
curl -s $B/api/v0/instance
curl -s -D - $B/api/v0/nope | grep -E '^(HTTP|Content-Type)'
```

Expected（依次为：首页、深层路径、带哈希的 JS、缺失的哈希文件、POST、实例信息、不存在的接口）:

```
HTTP/1.1 200 OK
Cache-Control: no-cache
Content-Type: text/html; charset=utf-8
HTTP/1.1 200 OK
Cache-Control: no-cache
Content-Type: text/html; charset=utf-8
HTTP/1.1 200 OK
Cache-Control: public, max-age=31536000, immutable
Content-Type: text/javascript; charset=utf-8
404
HTTP/1.1 405 Method Not Allowed
Allow: GET, HEAD
Content-Type: text/plain; charset=utf-8
{"api_version":"v0","commit":"<当前提交>","product":"Nerve","version":"0.1.0-dev"}
HTTP/1.1 404 Not Found
Content-Type: application/problem+json
```

用浏览器打开 http://127.0.0.1:8080/ ：页面显示 Plane 的"🚧 Looks like Plane didn't start up correctly!"；开发者工具的网络面板中所有静态资源都是 200，唯一的 404 是 `GET /api/instances/`（Plane 的接口，Nerve 没有，spec 2.6）。

停止：

```bash
pkill -f 'bin/nerve serve'; sleep 2; lsof -iTCP:8080 -sTCP:LISTEN
```

Expected: 没有输出。

- [ ] **Step 6: 提交**

```bash
git add Makefile
git commit -m "build: embed the built web frontend in bin/nerve with make build

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: 持续集成、README、交接收尾与完整走查

**Files:**
- Modify: `.github/workflows/ci.yml`、`README.md`
- Modify: `docs/v0/M0-foundation/handoffs/P2-server-platform-p5-webui-mount.md`、`docs/v0/M0-foundation/handoffs/P3-api-contract-p5-notes.md`、`docs/v0/M0-foundation/handoffs/P4-plane-schema-p5-notes.md`
- 核对（不改）：`docs/v0/M0-foundation/M0-design.md` 第 11 节

**Interfaces:**
- Consumes: Task 1–5 的全部成果；`make gen-check-web`、`make lint-web`、`make build-web`
- Produces: 持续集成的 `web` 任务构建前端；开发者上手说明；关闭三个交给 P5 的交接事项

- [ ] **Step 1: 修改 `.github/workflows/ci.yml`（`web` 任务）**

把：

```yaml
      - name: Enable corepack
        run: corepack enable
      - name: Install
        run: pnpm install --frozen-lockfile
      - name: Generated code
        run: make gen-check-web
      - name: Lint
        run: make lint-web
```

替换为：

```yaml
      - name: Enable corepack
        run: corepack enable
      # pnpm 的包存储按锁文件缓存：锁文件变了就换一个缓存，旧的包不会越积越多
      - name: Locate the pnpm store
        id: pnpm-store
        run: echo "path=$(pnpm store path --silent)" >> "$GITHUB_OUTPUT"
      - name: Cache the pnpm store
        uses: actions/cache@v6
        with:
          path: ${{ steps.pnpm-store.outputs.path }}
          key: pnpm-store-${{ runner.os }}-${{ hashFiles('pnpm-lock.yaml') }}
      - name: Install
        run: pnpm install --frozen-lockfile
      - name: Generated code
        run: make gen-check-web
      - name: Lint
        run: make lint-web
      - name: Build
        run: make build-web
```

`server` 任务不变；两个任务的 `if` 条件不变。

- [ ] **Step 2: 修改 `README.md`（Node 的用途）**

把：

```markdown
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）。代码生成和前端检查需要它
```

替换为：

```markdown
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）。代码生成，以及前端的检查、开发和构建需要它
```

- [ ] **Step 3: 修改 `README.md`（`make lint` 的说明，新增"前端"一节）**

把：

```markdown
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（TS 类型检查）。
```

替换为：

```markdown
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（前端的类型检查、oxlint、格式检查）。

## 前端

`web/apps/web` 和 `web/packages/*` 从 Plane 原样迁入，来源提交和之后的每一处改动见[前端改动清单](docs/v0/frontend-changes.md)；`web/packages/api-client` 是生成的 TS 客户端。前端任务由 turbo 按 `turbo.json` 编排，入口仍是 Makefile：

- **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。改了 `web/packages/*` 下的代码，要重新执行 `make web-dev`。
- **构建**：`make build` 构建前端，复制到 `server/internal/platform/webui/dist/`，编译出内嵌前端的 `bin/nerve`；`NERVE_ENV=dev bin/nerve serve` 之后打开 http://127.0.0.1:8080 。`dist/` 中只提交了 `.gitkeep`，没有构建过前端时页面上只有一句提示。
- **M0 中看到的页面**：前端还在调用 Plane 的接口（例如 `/api/instances/`），Nerve 返回 404，页面显示 Plane 的"didn't start up correctly"。这是预期的，前端从 M2 起对接 Nerve 的接口。
- **lint 警告只降不升**：每个包的 `check:lint` 脚本用 `--max-warnings` 记着当前的警告数，警告多了 `make lint-web` 就失败；修掉警告后，在同一个提交里把这个数调低到新的警告数。
```

- [ ] **Step 4: 修改 `docs/v0/M0-foundation/handoffs/P2-server-platform-p5-webui-mount.md`（改为 done，写明处理结果）**

把：

```markdown
status: open
```

替换为：

```markdown
status: done
```

把：

```markdown
来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P5）

1. `bootstrap.newApp` 在挂上各模块之后执行 `mux.Handle("/", webui.Handler(webFiles))`，注册的是不带方法的 `/`（[P5 spec](../specs/P5-web-import.md) 2.10）。`/api/`、`/healthz`、`/readyz` 仍由平台处理：`bootstrap` 的测试验证了挂上前端之后，`/healthz` 仍返回 JSON，`GET /api/v0/nope`、`POST /api/v0/instance` 仍返回 404 problem+json，其他路径返回 `index.html`。
2. `webui` 的 handler 只处理 GET 和 HEAD，其他方法返回 405，带 `Allow: GET, HEAD`（spec 2.9）。
3. `web.enabled` 没有加入：v0 中没有需要关闭内嵌前端的部署方式，这个开关没有使用者（spec 第 3 节第 5 项）。

来源：[M0/P2 评审记录](../reviews/P2-server-platform-review.md)。
```

- [ ] **Step 5: 修改 `docs/v0/M0-foundation/handoffs/P3-api-contract-p5-notes.md`（改为 done，写明处理结果）**

把：

```markdown
status: open
```

替换为：

```markdown
status: done
```

把：

```markdown
来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P5）

1. `.oxlintrc.json` 的 `ignorePatterns` 排除 `web/packages/api-client/src/schema.gen.ts`；`.oxfmtrc.json` 的 `ignorePatterns` 排除它和 `api/dist/**`（oxlint 只检查 JS/TS 文件，不会碰 `api/dist/`）。见 [P5 spec](../specs/P5-web-import.md) 2.7。
2. `make lint-web` 改为 `turbo run check:types check:lint check:format`。`turbo run check:types --dry=json` 列出 12 个类型检查任务，其中有 `@nerve/api-client#check:types`。
3. 根目录的 `package.json` 保留 `@redocly/cli` 2.53.3。换成新的锁文件之后，`make gen-check-web` 没有差异；Plane 的覆盖项把 Redocly 间接依赖的 brace-expansion 从 2.1.7 换成 5.0.9，打包结果不变（spec 2.5）。
4. api-client 的 `typescript` 改为 `catalog:`（5.8.3）。
5. 包名同时存在 `@nerve/*` 和 `@plane/*` 两种前缀，照旧，M1 统一改名。
6. 可选的 tsconfig 继承没有做：api-client 的配置已经是 `strict` 加 `noUncheckedIndexedAccess`；继承 `@plane/typescript-config` 会让新包依赖一个 M1 要改名的 Plane 包。

来源：[M0/P3 评审记录](../reviews/P3-api-contract-review.md)。
```

- [ ] **Step 6: 修改 `docs/v0/M0-foundation/handoffs/P4-plane-schema-p5-notes.md`（改为 done，写明处理结果）**

把：

```markdown
status: open
```

替换为：

```markdown
status: done
```

把：

```markdown
来源：[M0/P4 评审记录](../reviews/P4-plane-schema-review.md)。
```

替换为：

```markdown
## 处理结果（M0/P5）

1. 前端从 `02c19e1` 迁入（[P5 spec](../specs/P5-web-import.md) 2.3）。
2. 复制命令用完整的 SHA `02c19e1341d93141e8ad7b3278298adce208bafc`，复制前先用 `git -C "$PLANE" rev-parse <SHA>^{commit}` 确认提交存在；迁入的 13 个目录和 `patches/` 与这个提交中对应的 git 树对象完全相同。
3. [前端改动清单](../../frontend-changes.md)"一、代码来源"记下了完整的 SHA。

来源：[M0/P4 评审记录](../reviews/P4-plane-schema-review.md)。
```

- [ ] **Step 7: 提交**

Run: `git status --short`
Expected:

```
 M .github/workflows/ci.yml
 M README.md
 M docs/v0/M0-foundation/handoffs/P2-server-platform-p5-webui-mount.md
 M docs/v0/M0-foundation/handoffs/P3-api-contract-p5-notes.md
 M docs/v0/M0-foundation/handoffs/P4-plane-schema-p5-notes.md
```

```bash
git add .github/workflows/ci.yml README.md docs/v0/M0-foundation/handoffs
git commit -m "ci: build the web frontend in the web job; document the frontend

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 8: 在干净的克隆上执行持续集成两个任务的步骤**

```bash
CI_DIR="$(mktemp -d)/nerve"
git clone -q --branch "$(git branch --show-current)" . "$CI_DIR"
cd "$CI_DIR"
pnpm install --frozen-lockfile
make gen-check-web
make lint-web
make build-web
make gen-check-go
make lint-go
make test
git status --short --untracked-files=all | wc -l
```

Expected:
- `pnpm install --frozen-lockfile` 输出 `Done in …`；
- `make gen-check-web`、`make gen-check-go` 退出码 0；
- `make lint-web` 输出 `Tasks:    46 successful, 46 total`；`make build-web` 输出 `Tasks:    11 successful, 11 total`（其中 10 个命中缓存）；
- `make lint-go` 输出 `0 issues.`（第一次会由 `make tools` 下载 golangci-lint 到这个克隆的 `bin/`）；`make test` 所有包都是 `ok`；
- 最后一行是 `0`：所有步骤都没有改动工作区。

完成后删除这个临时目录（`rm -rf "$CI_DIR"`，路径以 `mktemp -d` 的输出为准）。

- [ ] **Step 9: 核对 M0 设计文档的 Phase 进度表**

Run: `grep -n '^| P5 ' docs/v0/M0-foundation/M0-design.md`
Expected（spec 和 plan 提交时已经更新，这里不需要改动；review 链接在代码评审后补上）:

```
497:| P5 | web-import | 进行中 | [spec](specs/P5-web-import.md) | [plan](plans/P5-web-import.md) | — |
```

- [ ] **Step 10: 按 README 从头走一遍（完整走查）**

```bash
pnpm install --frozen-lockfile
make dev-db
make gen-check
make lint
make test
make build
make
```

Expected:
- `make gen-check` 退出码 0，工作区没有变化；
- `make lint` 输出 `0 issues.`，前端检查 `Tasks:    46 successful, 46 total`；
- `make test` 所有包都是 `ok`；
- `make build` 产出 `bin/nerve`；
- `make` 列出 20 个命令（Task 5 Step 3）。

然后按 Task 5 Step 5 启动 `bin/nerve serve`，重复那一组 `curl`，得到同样的结果；按 Task 3 Step 5 同时启动 `make run` 和 `make web-dev`，重复代理的检查。最后停止所有后台进程，`lsof -iTCP:8080 -sTCP:LISTEN` 和 `lsof -iTCP:3000 -sTCP:LISTEN` 都没有输出。

Run: `git status --short`
Expected: 没有输出。

- [ ] **Step 11: 推送并确认持续集成（由控制者执行）**

1. 推送分支，确认 `CI` 的 `server` 和 `web` 两个任务都通过：`web` 依次执行 `Locate the pnpm store`、`Cache the pnpm store`、`Install`、`Generated code`（`make gen-check-web`）、`Lint`（`make lint-web`）、`Build`（`make build-web`）。
2. 记录第一次运行（pnpm 缓存未命中）中 `web` 任务和各步骤的耗时；再推送一个只改文档的提交，确认 `Cache the pnpm store` 命中（`Cache restored from key: pnpm-store-Linux-…`），记录耗时。两次的耗时写进 P5 的 review（spec 2.12、第 6 节）。

---

## 完成后

P5 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M0-foundation/reviews/P5-web-import-review.md`：裁定 spec 第 3 节的差异（特别是第 1、2、3、4、5 项），记录持续集成的耗时，按 spec 第 7 节建立 handoff（M1、M2、P6），同步 spec 第 3 节末尾列出的上级文档；并把 M0 设计文档中 P5 的状态改为"已完成"、补上 review 链接。
