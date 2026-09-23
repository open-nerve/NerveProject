# M1/P2 内容类功能 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删掉 M1 设计 2.1 的全部内容类功能（侧边栏自定义导航、数据分析、活跃迭代推广页、个人主页统计、估算、甘特图与时间线、自动关闭、文档页与协作编辑、更新日志、编辑器里的协作残留、便签、首页快捷链接与个性化、自定义主题、旧仪表盘、导出与集成服务），把侧边栏、个人主页、首页、编辑器内核四处保留功能的形态固定下来，把迭代和模块的进度改成只按工作项数计算；每删一个功能加一条关键词守卫规则、删它的文案和图片、降它造成的 lint 上限，并核对锁文件。

**Architecture:** 12 个 Task，顺序即 M1 设计 9 节 P2 的顺序，每个 Task 一个提交。每个 Task 的内部节奏固定：**删文件 → `check:types` → 按报错删下一层 → 删文案 → 加守卫规则、登记例外 → 清理新出现的 `no-unused-vars` 并降上限 → 三个门禁 → 提交**。类型检查和 knip 是删除的向导：删掉一层就再跑一次，把它们报出来的下一层接着删。共享文件（根 store、命令面板 store、工作项 store、编辑器包、`app/routes/core.ts`、各包的 `index.ts`）由当前 Task 一次改完，不并行。三处保留行为用进仓库的单元测试守住（侧边栏与个人主页标签、数量进度、编辑器组成），其余用 review 附录里的临时脚本。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、MobX 6、TipTap 2.26。

**Spec:** `docs/v0/M1-frontend-trim/specs/P2-trim-content.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

---

## Global Constraints

### 命令与环境

- **所有命令在仓库根目录（worktree 的根目录）执行**，不要 `cd`。要在某个包里执行时用
  `pnpm --dir <目录>` 或 `pnpm --filter <包名>`。开始之前执行一次 `corepack enable` 和
  `pnpm install --frozen-lockfile`；不做任何全局安装。
- 改过某个 `package.json` 之后，第一条 `pnpm exec` 会先自动核对并安装依赖，多打印
  `Scope: all 16 workspace projects` 到 `Done in …` 几行（pnpm 11 的 `verifyDepsBeforeRun`）；
  本计划的预期输出省略这几行。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写；
  `web/packages/i18n/src/types/keys.generated.ts` 不应存在（P1 已删，老工作区里若有，先删掉）。
- **一次性脚本**放在仓库外的 `/tmp/nerve-p2/`（不进仓库；有沙箱限制时换成自己的临时目录，
  命令中的路径随之替换）。本节给出六个脚本，Task 1 Step 1 写入。每个 Task 开始时，
  如果脚本不存在（换了机器或会话），按本节重新写入。

### 每个 Task 的固定节奏

1. **先删文件**（`rm`、`rm -r`），路由文件必须**同一步**删掉 `web/apps/web/app/routes/core.ts`
   里的条目，否则 `react-router typegen` 报 ENOENT，`check:types` 全线失败。
2. **跑类型检查**：`pnpm exec turbo run check:types`。它是删除的向导：报错列出的每一处都要
   **删掉**，不要改成空实现、不要加可选参数。反复跑到
   `Tasks:   22 successful, 22 total`。
   - **报到没动过的行上时**（常见 TS2339，指向一个本 Task 没碰过的文件）先执行
     `rm -f web/apps/web/.turbo/tsconfig.tsbuildinfo web/packages/*/.turbo/tsconfig.tsbuildinfo`
     再跑一次：那是上一次增量编译留下的 `tsbuildinfo`。
3. **删文案**：`node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-after.txt`，和上一 Task 的
   同名文件比差集，得到"本 Task 造成的无引用键"，用 `i18n-del.mjs` 中英文一起删。
   命名空间整删时同时删 `web/packages/i18n/src/constants/namespaces.ts` 里的条目。
4. **加守卫规则**：把本 Task 的规则写进 `tools/keywords.json`，跑 `node tools/keywords.mjs`，
   把剩下的命中逐条判断：属于本功能的**删掉**，属于后续 Phase 的**登记例外**。
   - 例外的 `match` 必须写**正则实际匹配到的那段文本**，不是整词。
     例如规则是 `estimat`（不区分大小写）时，`Estimates` 这一处的 `match` 写 `Estimat`。
     写错时工具报 `stale exception … matches nothing now`。
   - 同一文件里同一段文本出现多次时加 `"count": <次数>`。
   - `until` 写 `M1/P3`、`M6` 或 `M9`（`M9` = v0 内不到期）。
   - 改过 `tools/keywords.json` 之后跑 `pnpm exec oxfmt tools`（根 `check:format` 覆盖 `tools/`）。
5. **降 lint 上限**：`pnpm exec turbo run check:lint`。删代码会留下未使用的 import 和局部变量，
   警告数可能先**变多**。按 `pnpm --dir web/apps/web exec oxlint --format=json .` 里的
   `no-unused-vars` 逐个删掉，再把 `check:lint` 报的新数值写回对应 `package.json`：
   `node /tmp/nerve-p2/pkg.mjs <package.json> set-script check:lint "node ../../../tools/lint-cap.mjs <n>"`。
6. **三个门禁**：`make lint-web`、`make test-web`、`make build-web`。
7. **提交**：一个 Task 一个提交，提交信息用英文，最后一行是
   `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`。
   提交前 `git status --short` 只应有本 Task 的改动（`?? .superpowers/` 是本仓库未被忽略的
   工作目录，属于预期）。

### 一次性脚本（Task 1 Step 1 写入 `/tmp/nerve-p2/`）

`pkg.mjs`：按仓库格式（两个空格缩进的 JSON、末尾换行）改写 `package.json`；要删的东西不存在时报错。

```js
// One-off (M1/P2): edits a package.json and writes it back in the repository's format (2-space JSON
// and a trailing newline). Fails when a script or dependency to delete is not there.
// usage: node pkg.mjs <package.json> del-script <name>... | del-dep <name>... | set-script <name> <command>
//        | del-export <subpath>... | add-dev-dep <name> <specifier>
import fs from "node:fs";

const [file, op, ...args] = process.argv.slice(2);
const pkg = JSON.parse(fs.readFileSync(file, "utf8"));
if (op === "del-script") {
  for (const name of args) {
    if (!(name in pkg.scripts)) throw new Error(`${file}: no script ${name}`);
    delete pkg.scripts[name];
  }
} else if (op === "del-dep") {
  for (const name of args) {
    const sections = ["dependencies", "devDependencies"].filter((s) => pkg[s]?.[name] !== undefined);
    if (sections.length === 0) throw new Error(`${file}: no dependency ${name}`);
    for (const s of sections) delete pkg[s][name];
  }
} else if (op === "set-script") {
  pkg.scripts[args[0]] = args[1];
} else if (op === "add-dev-dep") {
  const deps = { ...pkg.devDependencies, [args[0]]: args[1] };
  pkg.devDependencies = Object.fromEntries(Object.entries(deps).toSorted(([a], [b]) => a.localeCompare(b)));
} else if (op === "del-export") {
  for (const name of args) {
    if (!(name in pkg.exports)) throw new Error(`${file}: no export ${name}`);
    delete pkg.exports[name];
  }
} else {
  throw new Error(`unknown operation ${op}`);
}
fs.writeFileSync(file, `${JSON.stringify(pkg, null, 2)}\n`);
```

`i18n-del.mjs`：按命名空间删中英文两份里的键，保持仓库的 JSON 格式和键顺序，删空的对象一并去掉；
键不存在就报错（陈旧的清单会大声失败）。

```js
// Deletes keys from both locales of one namespace, keeping the repository's JSON format (2-space
// indent, trailing newline, key order). A key that is not there is an error, so a stale list fails
// loudly. usage: node i18n-del.mjs <namespace> <dotted.key>...   (from the repository root)
import fs from "node:fs";

const [ns, ...keys] = process.argv.slice(2);
if (!ns || keys.length === 0) {
  console.error("usage: node i18n-del.mjs <namespace> <dotted.key>...");
  process.exit(2);
}
for (const locale of ["en", "zh-CN"]) {
  const file = `web/packages/i18n/src/locales/${locale}/${ns}.json`;
  const data = JSON.parse(fs.readFileSync(file, "utf8"));
  for (const key of keys) {
    const parts = key.split(".");
    let node = data;
    for (const part of parts.slice(0, -1)) {
      node = node?.[part];
      if (node === undefined) throw new Error(`${file}: no ${key}`);
    }
    if (!(parts.at(-1) in node)) throw new Error(`${file}: no ${key}`);
    delete node[parts.at(-1)];
  }
  // drop objects that the deletions emptied
  const prune = (obj) => {
    for (const [k, v] of Object.entries(obj)) {
      if (v && typeof v === "object" && !Array.isArray(v)) {
        prune(v);
        if (Object.keys(v).length === 0) delete obj[k];
      }
    }
  };
  prune(data);
  fs.writeFileSync(file, `${JSON.stringify(data, null, 2)}\n`);
  console.log(`${file}: ${keys.length} keys removed`);
}
```

`keyuse.mjs`：按"整键字面量 + 模板前缀"（M1 设计 2.3）列出没有代码引用的 en 键。

```js
// Reports every en locale key that no source file references, by the "whole-key literal + template
// prefix" method (M1 design 2.3): a key counts as referenced when its full dotted name appears as a
// literal anywhere outside the locale files, or when some literal is a prefix of it up to a dot
// boundary and the source builds the rest (a template or a variable). Prefix hits are reported
// separately so they can be checked by hand.
// usage: node keyuse.mjs [<key prefix filter>]   (from the repository root)
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const LOCALES = "web/packages/i18n/src/locales";
const filter = process.argv[2];

const flatten = (obj, prefix, out) => {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === "object" && !Array.isArray(v)) flatten(v, key, out);
    else out.set(key, true);
  }
};

const keys = new Map(); // key -> namespace
for (const file of fs.readdirSync(path.join(LOCALES, "en"))) {
  const ns = path.basename(file, ".json");
  const flat = new Map();
  flatten(JSON.parse(fs.readFileSync(path.join(LOCALES, "en", file), "utf8")), "", flat);
  for (const k of flat.keys()) keys.set(k, ns);
}

const files = execFileSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard", "web", "tools"], {
  encoding: "utf8",
  maxBuffer: 256 * 1024 * 1024,
})
  .split("\0")
  .filter((f) => f && !f.startsWith(`${LOCALES}/`) && /\.(tsx?|jsx?|mjs|json)$/.test(f));

let corpus = "";
for (const f of files) {
  try {
    corpus += `\n${fs.readFileSync(f, "utf8")}`;
  } catch {}
}

const exact = [];
const prefixOnly = [];
for (const [key, ns] of keys) {
  if (filter && !key.startsWith(filter)) continue;
  if (corpus.includes(key)) continue;
  const parts = key.split(".");
  let hit = "";
  for (let i = parts.length - 1; i >= 1; i--) {
    const p = `${parts.slice(0, i).join(".")}.`;
    if (corpus.includes(p)) {
      hit = p;
      break;
    }
  }
  if (hit) prefixOnly.push(`${ns}  ${key}  (prefix "${hit}" appears)`);
  else exact.push(`${ns}  ${key}`);
}
console.log(`unreferenced (no literal, no prefix): ${exact.length}`);
for (const line of exact) console.log(`  ${line}`);
console.log(`unreferenced as a whole key but a prefix appears: ${prefixOnly.length}`);
for (const line of prefixOnly) console.log(`  ${line}`);
```

`keydiff.sh`：比较两次 `keyuse.mjs` 的输出，列出"本 Task 新造成的无引用键"。

```bash
#!/bin/bash
# One-off (M1/P2): the keys that became unreferenced between two keyuse.mjs reports.
# usage: bash keydiff.sh <before.txt> <after.txt>
set -eu
norm() { awk '{print $1, $2}' "$1" | grep -v '^unreferenced' | sort -u; }
comm -13 <(norm "$1") <(norm "$2")
```

`web-size.mjs`：构建体积（M1 设计 7.6，与 P1 同一个脚本）。

```js
// M1 build-size measurement (M1 design 7.6): the same script at M1/P1 and at the M1 closeout. Run from the
// repository root after `make build-web`. Prints the files and bytes of JS, CSS, fonts and everything else
// in web/apps/web/build/client, the largest JS chunk, and the number of locale chunks (a chunk named after
// an i18n namespace that only default-exports its data and imports nothing).
// usage: node web-size.mjs
import fs from "node:fs";
import path from "node:path";

const root = "web/apps/web/build/client";
const namespaces = fs.readdirSync("web/packages/i18n/src/locales/en").map((f) => path.basename(f, ".json"));
const kinds = { js: [0, 0], css: [0, 0], fonts: [0, 0], other: [0, 0] };
let largest = ["", 0];
let localeChunks = 0;
for (const rel of fs.readdirSync(root, { recursive: true })) {
  const file = path.join(root, rel);
  const stat = fs.statSync(file);
  if (!stat.isFile()) continue;
  const kind = /\.js$/.test(rel) ? "js" : /\.css$/.test(rel) ? "css" : /\.(woff2?|ttf|otf|eot)$/.test(rel) ? "fonts" : "other";
  kinds[kind][0] += 1;
  kinds[kind][1] += stat.size;
  if (kind === "js" && stat.size > largest[1]) largest = [rel, stat.size];
  const name = path.basename(rel);
  if (kind === "js" && namespaces.some((ns) => new RegExp(`^${ns}-[\\w-]{8}\\.js$`).test(name))) {
    const text = fs.readFileSync(file, "utf8");
    if (!text.includes('from"./') && /\bas default[,}]/.test(text)) localeChunks += 1;
  }
}
for (const [kind, [count, bytes]] of Object.entries(kinds)) console.log(`${kind}: ${count} files, ${bytes} bytes`);
console.log(`largest chunk: ${largest[0]}, ${largest[1]} bytes`);
console.log(`locale chunks: ${localeChunks}`);
```

`lock-diff.mjs`：锁文件核对（M1 设计 8，与 P1 同一个脚本）。把 P1 plan 的 Global Constraints 中
`lock-diff.mjs` 的全文原样写入 `/tmp/nerve-p2/lock-diff.mjs`（那份脚本没有改动；
不要凭记忆重写，从 `docs/v0/M1-frontend-trim/plans/P1-web-hygiene.md` 里复制）。
它把 `pnpm-lock.yaml` 和 `git show HEAD:pnpm-lock.yaml` 比较，只打印不是"纯删除"的差异。

### 图片资源的口径

只删本 Phase 弄成没人引用的图片。判断办法：图片文件名在 `web/` 的任何源码里都不再出现。
基线（`57fccb6`）就已经没人引用的 254 个图片属于 Plane 自带的死资源，由 M1 收尾处理，不在 P2。

### 文件结构

| 位置 | 改动 | Task |
|---|---|---|
| `tools/keywords.json` | 顶层 `phase` 改为 `M1/P2`；新增 12 条规则、14 条例外 | 1–11 |
| `web/apps/web/app/routes/core.ts` | 删掉数据分析、活跃迭代、个人主页、估算、文档页、便签、导出的条目 | 2,3,4,7,9,11 |
| `web/apps/web/app/routes/redirects/core/` | 删 `analytics.tsx`，增 `profile-index.tsx` | 2,3 |
| `web/apps/web/core/store/root.store.ts` | 逐个 Task 去掉被删 store 的接线和 `resetOnSignOut` | 2,4,5,7,9,10 |
| `web/packages/*/src/index.ts` | 逐个 Task 同步导出 | 全部 |
| `web/packages/i18n/src/constants/namespaces.ts` | 删 `page`、`stickies`、`integration` | 7,9,11 |
| `pnpm-workspace.yaml` | catalog 条目随依赖删除 | 2,7,9,12 |
| `*/package.json` 的 `check:lint` | 上限只降不升 | 1–11 |
| `docs/v0/frontend-changes.md` | 第二节同步 | 12 |

---

### Task 1: 侧边栏改为固定列表

**Files**

- Create: `web/apps/web/core/components/navigation/project-navigation-dialog.tsx`
- Delete:
  - `web/apps/web/core/components/navigation/customize-navigation-dialog.tsx`
  - `web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-sidebar.tsx`
  - `web/apps/web/core/components/workspace/sidebar/extended-sidebar-item.tsx`
- Modify（19 个）：`tools/keywords.json`、`web/apps/web/package.json`、
  `app/(all)/[workspaceSlug]/(projects)/_sidebar.tsx`、
  `core/components/sidebar/{resizable-sidebar,sidebar-wrapper}.tsx`、
  `core/components/workspace/sidebar/{sidebar-item,sidebar-menu-items}.tsx`、
  `core/hooks/use-navigation-preferences.ts`、`core/layouts/auth-layout/workspace-wrapper.tsx`、
  `core/services/workspace.service.ts`、`core/store/theme.store.ts`、
  `core/store/user/base-permissions.store.ts`、`core/store/workspace/index.ts`、
  `web/packages/constants/src/{fetch-keys,workspace}.ts`、
  `web/packages/types/src/{navigation-preferences,workspace}.ts`、
  `web/packages/i18n/src/locales/{en,zh-CN}/common.json`

**Interfaces**

- Consumes：P1 的 `tools/keywords.mjs`、`tools/lint-cap.mjs`；`useProjectNavigationPreferences`（保留）。
- Produces：`ProjectNavigationDialog`（项目导航偏好的唯一写入入口）；
  固定的 `WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS` / `WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS`；
  守卫规则 `sidebar-customization`；`tools/keywords.json` 的顶层 `phase` = `M1/P2`。

**Steps**

- [ ] **Step 1: 写入一次性脚本**

Run: `mkdir -p /tmp/nerve-p2`

把 Global Constraints 里的 `pkg.mjs`、`i18n-del.mjs`、`keyuse.mjs`、`keydiff.sh`、`web-size.mjs`
写入 `/tmp/nerve-p2/`，并从 P1 plan 复制 `lock-diff.mjs`。

Run: `ls /tmp/nerve-p2`
Expected:

```
i18n-del.mjs
keydiff.sh
keyuse.mjs
lock-diff.mjs
pkg.mjs
web-size.mjs
```

- [ ] **Step 2: 记录基线**

Run: `pnpm install --frozen-lockfile && make lint-web 2>&1 | tail -3`
Expected: `Tasks:   52 successful, 52 total`；前面有一行 `keywords: 12 rules, 0 exceptions, no hits.`

Run: `make test-web 2>&1 | grep Tasks:`
Expected: `Tasks:   12 successful, 12 total`

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected:

```
 Tasks:    11 successful, 11 total
js: 505 files, 10270419 bytes
css: 3 files, 322306 bytes
fonts: 34 files, 6849620 bytes
other: 142 files, 9719426 bytes
largest chunk: assets/toolbar-<hash>.js, 1821937 bytes
locale chunks: 40
```

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-base.txt; head -1 /tmp/nerve-p2/keys-base.txt`
Expected: `unreferenced (no literal, no prefix): 564`（这 564 个属于 M1 收尾，P2 不删）

- [ ] **Step 3: 新建 `ProjectNavigationDialog`**

`customize-navigation-dialog.tsx` 有"个人""工作区""项目"三节。**项目**一节配置的是保留的项目导航
偏好（`workspace_user_properties`），而且是它唯一的写入入口，所以把它搬进新文件，另外两节随旧文件删除。

写入 `web/apps/web/core/components/navigation/project-navigation-dialog.tsx`：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { CloseOutline } from "@makeplane/propel/icons";
// plane imports
import { useTranslation } from "@plane/i18n";
import { Checkbox } from "@makeplane/propel/components/checkbox";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { cn } from "@plane/utils";
// hooks
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";

type TProjectNavigationDialogProps = {
  isOpen: boolean;
  onClose: () => void;
};

export const ProjectNavigationDialog = observer(function ProjectNavigationDialog(props: TProjectNavigationDialogProps) {
  const { isOpen, onClose } = props;
  const { t } = useTranslation();

  // store hooks
  const {
    preferences: projectPreferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  } = useProjectNavigationPreferences();

  // local state for limited projects count input
  const [projectCountInput, setProjectCountInput] = useState(projectPreferences.limitedProjectsCount.toString());

  // Prevent typing invalid characters in number input
  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Block: e, E, +, -, .
    if (["e", "E", "+", "-", "."].includes(e.key)) {
      e.preventDefault();
    }
  };

  // Handle project count input change
  const handleProjectCountChange = (value: string) => {
    // Strip any non-digit characters
    const cleanedValue = value.replace(/\D/g, "");
    setProjectCountInput(cleanedValue);

    // Parse and validate the value
    const numValue = parseInt(cleanedValue, 10);

    // If valid number, enforce minimum of 1
    if (!isNaN(numValue)) {
      const validValue = Math.max(1, numValue);
      updateLimitedProjectsCount(validValue);
    }
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <div className="flex max-h-[90vh] flex-col rounded-lg bg-surface-1">
        {/* Header */}
        <div className="flex justify-between px-6 pt-4">
          <div>
            <h2 className="text-18 font-semibold text-primary">{t("projects")}</h2>
          </div>
          <button
            onClick={onClose}
            className="flex size-5 flex-shrink-0 items-center justify-center rounded-sm text-placeholder hover:bg-layer-1"
            aria-label={t("close")}
          >
            <CloseOutline className="size-4" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-4">
          <div className="rounded-md border border-subtle bg-surface-2 px-2 py-2">
            <div className="space-y-3">
              {/* Navigation Mode Radio Buttons */}
              <div className="space-y-2">
                {/* oxlint-disable-next-line jsx_a11y/label-has-associated-control */}
                <label className="flex cursor-pointer gap-2 rounded-md px-2 py-1.5 hover:bg-surface-2">
                  <input
                    type="radio"
                    name="navigation-mode"
                    value="ACCORDION"
                    checked={projectPreferences.navigationMode === "ACCORDION"}
                    onChange={() => updateNavigationMode("ACCORDION")}
                    className="mt-1 size-4 text-accent-primary focus:ring-accent-strong"
                  />
                  <div className="flex-1">
                    <div className="text-13 text-primary">{t("accordion_navigation_control")}</div>
                    <div className="text-11 text-secondary">
                      Feature tabs will appear as nested items under project and acts as accordion.
                    </div>
                  </div>
                </label>

                {/* oxlint-disable-next-line jsx_a11y/label-has-associated-control */}
                <label className="flex cursor-pointer gap-2 rounded-md px-2 py-1.5 hover:bg-surface-2">
                  <input
                    type="radio"
                    name="navigation-mode"
                    value="TABBED"
                    checked={projectPreferences.navigationMode === "TABBED"}
                    onChange={() => updateNavigationMode("TABBED")}
                    className="mt-1 size-4 text-accent-primary focus:ring-accent-strong"
                  />
                  <div className="flex-1">
                    <div className="text-13 text-primary">{t("horizontal_navigation_bar")}</div>
                    <div className="text-11 text-secondary">
                      Feature tabs will appear as horizontal tabs inside a project.
                    </div>
                  </div>
                </label>
              </div>

              {/* Limited Projects Checkbox */}
              <div className="space-y-1">
                <div className="rounded-md px-2 py-1.5 hover:bg-surface-2">
                  <Checkbox
                    label={t("show_limited_projects_on_sidebar")}
                    stretch="full"
                    checked={projectPreferences.showLimitedProjects}
                    onCheckedChange={updateShowLimitedProjects}
                  />
                </div>

                {projectPreferences.showLimitedProjects && (
                  <div className="pl-8">
                    <div className="flex w-full flex-col gap-1">
                      <div className="flex w-full flex-col gap-2 pb-1.5">
                        <label className="w-full text-11 text-secondary">{t("enter_number_of_projects")}</label>
                        <input
                          type="number"
                          min="1"
                          step="1"
                          value={projectCountInput}
                          onKeyDown={handleKeyDown}
                          onChange={(e) => handleProjectCountChange(e.target.value)}
                          className={cn(
                            "w-full rounded-md px-2 py-1 text-13",
                            "border bg-surface-2",
                            "text-secondary",
                            parseInt(projectCountInput) >= 1
                              ? "border-strong focus:border-accent-strong focus:ring-1 focus:ring-accent-strong"
                              : "border-danger-strong focus:border-danger-strong focus:ring-1 focus:ring-danger-strong"
                          )}
                        />
                      </div>
                      {parseInt(projectCountInput) < 1 && projectCountInput !== "" && (
                        <span className="pl-0.5 text-11 text-danger-primary">Minimum value is 1</span>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </ModalCore>
  );
});
```

- [ ] **Step 4: 删文件**

Run:

```
rm web/apps/web/core/components/navigation/customize-navigation-dialog.tsx \
   "web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-sidebar.tsx" \
   web/apps/web/core/components/workspace/sidebar/extended-sidebar-item.tsx
```

Expected: 没有输出。

- [ ] **Step 5: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

按报错逐层删除，直到 22 个任务通过。要删的东西：

- `constants/src/workspace.ts`：`WORKSPACE_SIDEBAR_DYNAMIC_NAVIGATION_ITEMS_LINKS`、
  `WORKSPACE_SIDEBAR_STATIC_NAVIGATION_ITEMS_LINKS` 和 `WORKSPACE_SIDEBAR_PREFERENCES_*`
  这一组"可排序、可隐藏"的记录，换成两个固定数组
  `WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS`（`home`、`stickies`、`your_work`、`drafts`）和
  `WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS`（`projects`、`views`、`analytics`、`archives`），
  **按常量里的声明顺序全部显示**；顺带删掉零引用的 `inbox` 条目。
- `types/src/navigation-preferences.ts`、`types/src/workspace.ts`：个人与工作区偏好的类型；
  项目偏好的类型保留。
- `core/services/workspace.service.ts`：`/sidebar-preferences/` 的读写方法。
- `core/store/workspace/index.ts`、`core/hooks/use-navigation-preferences.ts`：
  个人与工作区偏好的 store 和 hook；`useProjectNavigationPreferences` 保留。
- `core/store/theme.store.ts`：`isExtendedSidebarOpened`、`toggleExtendedSidebar` 和
  localStorage 键 `extended_sidebar_collapsed`；`isExtendedProjectSidebarOpened` 保留。
- `core/store/user/base-permissions.store.ts`：`hasPageAccess`（零调用方，是动态导航常量在侧边栏
  之外的唯一读取方）。
- `sidebar-menu-items.tsx`、`sidebar-item.tsx`、`sidebar-wrapper.tsx`、`resizable-sidebar.tsx`、
  `_sidebar.tsx`、`workspace-wrapper.tsx`：改为渲染两个固定数组，弹窗入口改成
  `ProjectNavigationDialog`。

Expected（最后一次）: `Tasks:   22 successful, 22 total`

- [ ] **Step 6: 删文案**

Run: `node /tmp/nerve-p2/i18n-del.mjs common customize_navigation personal`
Expected:

```
web/packages/i18n/src/locales/en/common.json: 2 keys removed
web/packages/i18n/src/locales/zh-CN/common.json: 2 keys removed
```

- [ ] **Step 7: 守卫规则**

把 `tools/keywords.json` 的顶层 `"phase"` 从 `"M1/P1"` 改为 `"M1/P2"`，并在 `rules` 末尾追加：

```json
{
  "id": "sidebar-customization",
  "phase": "M1/P2",
  "why": "侧边栏的自定义导航（固定、排序、隐藏菜单项）读写 Plane 的 /sidebar-preferences/，数据在已砍掉的 workspace_user_preferences 表；侧边栏改为固定列表（M1 设计 3.3）",
  "files": {
    "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
    "flags": ""
  },
  "content": {
    "source": "sidebar-preferences|SidebarPreference|customize-navigation|CustomizeNavigation|customize_navigation|navigationPreferencesMap|(?:Personal|Workspace)NavigationPreferences|WORKSPACE_SIDEBAR_(?:DYNAMIC|STATIC|PREFERENCES)|ExtendedSidebarItem|ExtendedAppSidebar|toggleExtendedSidebar\\b|extended_sidebar_collapsed",
    "flags": ""
  },
  "samples": {
    "hit": [
      "    return this.get(`/api/workspaces/${workspaceSlug}/sidebar-preferences/`)",
      "import { CustomizeNavigationDialog } from \"@/components/navigation/customize-navigation-dialog\";",
      "  const { preferences } = useWorkspaceNavigationPreferences();",
      "  WORKSPACE_SIDEBAR_STATIC_NAVIGATION_ITEMS_LINKS,",
      "  const { isExtendedSidebarOpened, toggleExtendedSidebar } = useAppTheme();",
      "    localStorage.setItem(\"extended_sidebar_collapsed\", updatedState.toString());",
      "  \"customize_navigation\": \"Customize navigation\","
    ],
    "miss": [
      "  const { preferences: projectPreferences } = useProjectNavigationPreferences();",
      "import { ProjectNavigationDialog } from \"@/components/navigation/project-navigation-dialog\";",
      "  WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS,",
      "  const { isExtendedProjectSidebarOpened, toggleExtendedProjectSidebar } = useAppTheme();"
    ],
    "files": {
      "hit": [
        "web/apps/web/core/components/sidebar/sidebar-wrapper.tsx",
        "web/packages/constants/src/workspace.ts"
      ],
      "miss": [
        "web/apps/web/app/assets/logo.svg",
        "docs/v0/M1-frontend-trim/M1-design.md"
      ]
    }
  }
}
```

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 13 rules, 0 exceptions, no hits.`

- [ ] **Step 8: 降上限**

Run: `pnpm exec turbo run check:lint 2>&1 | grep -E "fewer than|more than" || echo none`
Expected: `web: 776 oxlint warnings, fewer than the cap of 777. …`

Run: `node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 776"`
Expected: 没有输出。

- [ ] **Step 9: 门禁与提交**

Run: `make lint-web 2>&1 | tail -3`
Expected: `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:`
Expected: `Tasks:   12 successful, 12 total`

Run: `make build-web 2>&1 | grep Tasks:`
Expected: `Tasks:   11 successful, 11 total`

Run: `git add -A && git diff --cached --shortstat`
Expected: `23 files changed, 315 insertions(+), 1381 deletions(-)`（行数可差几行，文件数应一致）

提交信息：

```
feat(web): the sidebar becomes a fixed list

The sidebar no longer lets a person pin, reorder or hide its items: it
shows the fixed lists of the constants, in the order they declare (M1
design 3.3). The preferences behind it lived in a table the schema no
longer has.

The dialog is not deleted outright. Its project section configures the
project navigation preferences, which stay, and it is their only way in,
so that section becomes ProjectNavigationDialog and the other two go.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 2: 数据分析、活跃迭代推广页；进度代码由 analytics 改名为 progress

**Files**

- Delete（88 个）：
  - 路由 4：`app/(all)/[workspaceSlug]/(projects)/analytics/[tabId]/{header,layout,page}.tsx`、
    `app/routes/redirects/core/analytics.tsx`
  - 活跃迭代推广页 4：`app/(all)/[workspaceSlug]/(projects)/active-cycles/{header,layout,page}.tsx`、
    `core/components/active-cycles/workspace-active-cycles-upgrade.tsx`
  - `core/components/analytics/` 整目录（35）、`core/components/chart/utils.ts`
  - `core/hooks/store/use-analytics.ts`、`core/services/analytics.service.ts`、
    `core/store/analytics.store.ts`、`helpers/graph.helper.ts`
  - 已死的侧边栏文件 3：`core/components/workspace/sidebar/workspace-menu{,-item,-header}.tsx`
  - `packages/constants/src/analytics/`（2）、`packages/constants/src/graph.ts`
  - `packages/types/src/analytics.ts`、`packages/types/src/charts/common.ts`
  - propel：`src/charts/{line-chart,radar-chart,scatter-chart,tree-map}/`（10）、`src/table/`（2）、
    `src/icons/workspace/analytics-icon.tsx`
  - 图片 19：`app/assets/empty-state/analytics/`（10）、`empty-state/empty_analytics.webp`、
    `empty-state/onboarding/analytics-{dark,light}.webp`、`app/assets/workspace-active-cycles/`（6）
- Rename（11）：
  - `core/components/cycles/analytics-sidebar/{index.ts,issue-progress.tsx,progress-stats.tsx,root.tsx,sidebar-chart.tsx,sidebar-details.tsx,sidebar-header.tsx}`
    → `core/components/cycles/progress-sidebar/…`
  - `core/components/modules/analytics-sidebar/{index.ts,issue-progress.tsx,progress-stats.tsx,root.tsx}`
    → `core/components/modules/progress-sidebar/…`
- Modify（55）：`tools/keywords.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`、
  `web/apps/web/package.json`、`web/packages/propel/{package.json,tsdown.config.ts}`、
  `app/routes/core.ts`、`app/routes/redirects/core/index.ts`、迭代与模块的详情页和头部 7、
  `core/components/issues/{filters,header}.tsx`、power-k 导航配置 2、
  `core/components/workspace/sidebar/helper.tsx`、`core/store/{cycle.store.ts,root.store.ts,theme.store.ts}`、
  `core/store/issue/cycle/issue.store.ts`、`core/store/project/project.store.ts`、
  `core/services/{cycle.service.ts,project/project.service.ts}`、
  `packages/constants/src/{chart,fetch-keys,index,workspace}.ts`、
  `packages/types/src/{index,workspace}.ts`、`packages/types/src/charts/index.ts`、
  `packages/types/src/project/projects.ts`、`packages/propel/src/icons/{registry.ts,workspace/index.ts}`、
  i18n 中英文 13 个文件

**Interfaces**

- Consumes：Task 1 的固定侧边栏常量（这里去掉 `analytics` 一项）。
- Produces：改名后的 `CycleProgress` / `ModuleProgressSidebar` / `ModuleProgress` /
  `fetchActiveCycleDistribution` / `cycleDistribution` / `cycleProgress`；
  守卫规则 `analytics`、`active-cycles-promo` 及 5 条例外。
- 顺序约束（M1 设计 9 节，硬）：数据分析必须在估算（Task 4）之前；
  propel 的 `bar-chart`、`pie-chart` 留到 Task 3 删。

**Steps**

- [ ] **Step 1: 先改名（`git mv`），再删**

改名必须先做：先删数据分析会让改名的文件在半路上编译不过。

Run:

```
git mv web/apps/web/core/components/cycles/analytics-sidebar web/apps/web/core/components/cycles/progress-sidebar
git mv web/apps/web/core/components/modules/analytics-sidebar web/apps/web/core/components/modules/progress-sidebar
```

Expected: 没有输出。

再按下表改符号（用编辑器全局替换，改完 `grep` 核对）：

| 旧 | 新 |
|---|---|
| `CycleAnalyticsProgress` | `CycleProgress` |
| `ModuleAnalyticsSidebar` | `ModuleProgressSidebar` |
| `ModuleAnalyticsProgress` | `ModuleProgress` |
| `fetchActiveCycleAnalytics` | `fetchActiveCycleDistribution` |
| `workspaceActiveCyclesAnalytics` | `cycleDistribution` |
| `workspaceActiveCyclesProgress` | `cycleProgress` |
| `cycle-analytics-tab-` | `cycle-progress-tab-` |
| `module-analytics-tab-` | `module-progress-tab-` |
| `analytics-sidebar` | `progress-sidebar` |

Run: `pnpm exec turbo run check:types 2>&1 | grep -E "error TS|Tasks:"`
Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 2: 删数据分析与活跃迭代推广页**

Run:

```
rm -r web/apps/web/core/components/analytics \
      web/apps/web/core/components/active-cycles \
      "web/apps/web/app/(all)/[workspaceSlug]/(projects)/analytics" \
      "web/apps/web/app/(all)/[workspaceSlug]/(projects)/active-cycles" \
      web/apps/web/app/assets/empty-state/analytics \
      web/apps/web/app/assets/workspace-active-cycles \
      web/packages/constants/src/analytics \
      web/packages/propel/src/charts/line-chart \
      web/packages/propel/src/charts/radar-chart \
      web/packages/propel/src/charts/scatter-chart \
      web/packages/propel/src/charts/tree-map \
      web/packages/propel/src/table
rm web/apps/web/app/routes/redirects/core/analytics.tsx \
   web/apps/web/core/components/chart/utils.ts \
   web/apps/web/core/components/workspace/sidebar/workspace-menu.tsx \
   web/apps/web/core/components/workspace/sidebar/workspace-menu-item.tsx \
   web/apps/web/core/components/workspace/sidebar/workspace-menu-header.tsx \
   web/apps/web/core/hooks/store/use-analytics.ts \
   web/apps/web/core/services/analytics.service.ts \
   web/apps/web/core/store/analytics.store.ts \
   web/apps/web/helpers/graph.helper.ts \
   web/apps/web/app/assets/empty-state/empty_analytics.webp \
   web/apps/web/app/assets/empty-state/onboarding/analytics-dark.webp \
   web/apps/web/app/assets/empty-state/onboarding/analytics-light.webp \
   web/packages/constants/src/graph.ts \
   web/packages/types/src/analytics.ts \
   web/packages/types/src/charts/common.ts \
   web/packages/propel/src/icons/workspace/analytics-icon.tsx
```

**同一步**删掉 `web/apps/web/app/routes/core.ts` 里 `analytics/[tabId]` 和 `active-cycles` 的条目，
以及 `app/routes/redirects/core/index.ts` 里 `analytics` 的那一条。

- [ ] **Step 3: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

要删的东西：

- `constants/src/workspace.ts`：固定列表里的 `analytics` 一项（Task 1 刚加的两个数组之一）。
- `constants/src/index.ts`、`types/src/index.ts`、`types/src/charts/index.ts`、
  `propel/src/icons/{registry.ts,workspace/index.ts}`、`propel/tsdown.config.ts`：同步导出；
  `propel` 的 `package.json` 用 `node /tmp/nerve-p2/pkg.mjs web/packages/propel/package.json del-export ./table`。
- `core/store/root.store.ts`：`analytics` 的接线和 `resetOnSignOut`。
- `cycle.service.ts`、`project.service.ts`、`cycle.store.ts`、`project.store.ts`、
  `issue/cycle/issue.store.ts`：工作区级活跃迭代的接口方法和 store 成员
  （`IWorkspaceActiveCyclesResponse`、`ProgressPro` 等）；迭代自身的进度方法保留（已改名）。
- `theme.store.ts`：数据分析弹窗的开关。
- `issues/{filters,header}.tsx`：工作项列表头部的"分析"按钮和弹窗。
- power-k 的 `config/navigation/{commands,root}.ts`：`nav_workspace_analytics` 命令。
- `workspace/sidebar/helper.tsx`：侧边栏图标映射里的 analytics。
- 迭代与模块的详情页、头部、移动端头部：传给已改名组件的 props；
  **`cycles/(detail)/mobile-header.tsx` 和 `modules/(detail)/mobile-header.tsx` 去掉
  `cycleDetails` / `moduleDetails` 之后，`getCycleById` / `getModuleById` 的解构和 import 也要删掉**
  （否则 Task 4 才会以 `no-unused-vars` 的形式暴露出来）。
- `types/src/workspace.ts`、`types/src/project/projects.ts`：分析相关字段。

Expected（最后一次）: `Tasks:   22 successful, 22 total`

- [ ] **Step 4: 删依赖**

Run:

```
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json del-dep @tanstack/react-table export-to-csv
```

把 `pnpm-workspace.yaml` 的 catalog 里 `@tanstack/react-table`、`export-to-csv` 两条删掉。

Run: `pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs`
Expected:

```
catalogs: 6 lines removed, 0 added
importers: 2 dependencies removed, 0 added or changed
packages: 3 removed, 0 added or changed
snapshots: 3 removed, 0 added or changed
```

- [ ] **Step 5: 删文案**

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t2.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-base.txt /tmp/nerve-p2/keys-t2.txt`

按输出删除（中英文成对）：`workspace.json` 的 `workspace_analytics` 整块和 `active_cycles*`、
`common.json` 的 `analytics`、`overview`、`no_of`、`admins`、`guests`、`users`、`work_items`、`epics`，
`navigation.json` 的 `sidebar.analytics`、`sidebar.projects`（推广页用），
`power-k.json` 的 `navigation_actions.nav_workspace_analytics`，
`empty-state.json` 的 `workspace_empty_state.analytics_*`、`workspace_empty_state.active_cycles`，
`cycle.json` 的 `active_cycle_analytics`，`project.json` 里推广页的文案。
另外把 `common.json`、`cycle.json` 里两处正文中的 "analytics" 改写为 "progress"。

- [ ] **Step 6: 守卫规则与例外**

在 `rules` 末尾追加 `analytics` 和 `active-cycles-promo` 两条（正文见 spec 2.15；
`analytics` 的 `files.source` 是 `^(?:web/.*|pnpm-(?:workspace|lock)\.yaml|turbo\.json)$`，
`content` 是 `analytics` 且 `flags` 为 `i`）。

Run: `node tools/keywords.mjs`

把剩下的命中逐条判断，属于后续 Phase 的登记为例外（`exceptions` 数组）：

```json
[
  { "rule": "analytics", "path": "web/packages/utils/src/tlds.ts", "match": "analytics", "reason": "顶级域名数据里的 .analytics，不是功能代码", "until": "M9" },
  { "rule": "analytics", "path": "web/apps/web/core/services/cycle.service.ts", "match": "analytics", "reason": "Plane 的迭代进度接口地址带 analytics，M1 不改接口（M1 设计 3.4）", "until": "M6" },
  { "rule": "analytics", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "analytics", "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "analytics", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "Analytics", "count": 3, "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "analytics", "path": "web/packages/types/src/epics.ts", "match": "Analytics", "count": 2, "reason": "Epic 的企业版类型，随 Epic 在 P3 删除", "until": "M1/P3" }
]
```

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 15 rules, 5 exceptions, no hits.`

- [ ] **Step 7: 降上限、门禁与提交**

Run: `pnpm exec turbo run check:lint 2>&1 | grep -E "fewer than|more than" || echo none`

先按 `pnpm --dir web/apps/web exec oxlint --format=json .` 里的 `no-unused-vars` 清理，再写回上限：

Run:

```
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 766"
node /tmp/nerve-p2/pkg.mjs web/packages/propel/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 24"
```

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 15 rules, 5 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:`
Expected: `Tasks:   12 successful, 12 total`

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 486 files, 10096756 bytes`、
`css: 3 files, 319557 bytes`、`other: 136 files, 8961872 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `154 files changed, 155 insertions(+), 5712 deletions(-)`

提交信息：

```
feat(web): delete workspace analytics and the active-cycles promo page

Workspace analytics goes with its old-address redirect, its sidebar
entry, its power-k command and the charts and table in propel that only
it drew. The area chart the burn-down uses and recharts stay.

The cycle and module progress code was called analytics too, which is
the reason the two could not be told apart: it is now called progress
(M1 design 3.4). The Plane endpoints it calls keep their own analytics
in the path; the keyword guard has an exception for them until M6.

The workspace-level active-cycles page is a promotion for a paid Plane
feature. The current-cycle block of the project cycle list stays.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 3: 个人主页只剩工作项标签和用户卡片

**Files**

- Create：`web/apps/web/app/routes/redirects/core/profile-index.tsx`、
  `web/packages/constants/src/navigation.test.ts`
- Delete（20）：
  - `app/(all)/[workspaceSlug]/(projects)/profile/[userId]/page.tsx`
  - `app/(all)/[workspaceSlug]/(projects)/profile/[userId]/activity/page.tsx`
  - `core/components/profile/overview/`（5）、`core/components/profile/activity/`（3）、
    `core/components/profile/time.tsx`
  - `core/components/core/activity.tsx`、`core/components/ui/loader/settings/activity.tsx`
  - `packages/propel/src/charts/bar-chart/`（3）、`packages/propel/src/charts/pie-chart/`（4）
- Modify（20）：`tools/keywords.json`、`pnpm-lock.yaml`、`web/apps/web/package.json`、
  `web/packages/constants/package.json`、`web/packages/propel/package.json`、
  `app/routes/core.ts`、`profile/[userId]/{header,layout,navbar}.tsx`、
  `core/components/profile/sidebar.tsx`、`core/services/user.service.ts`、
  `packages/constants/src/{fetch-keys,profile}.ts`、
  `packages/propel/src/charts/components/tick.tsx`、`packages/types/src/users.ts`、
  `packages/types/src/charts/index.ts`、i18n 中英文 4 个文件

**Interfaces**

- Consumes：工作区成员 store（`useMember().workspace`）、`WORKSPACE_MEMBERS` fetch key。
- Produces：`PROFILE_TABS`（合并 viewer/admin 两组）、`/profile/:userId` 的重定向路由、
  `navigation.test.ts`（侧边栏与个人主页标签的仓库内测试）、守卫规则 `profile-stats`。
- 顺序约束（M1 设计 9 节，硬）：用户卡片和重定向先到位，再删统计数据源；
  propel 的 `bar-chart`、`pie-chart` 必须等数据分析（Task 2）删完再删。

**Steps**

- [ ] **Step 1: 新建重定向路由**

写入 `web/apps/web/app/routes/redirects/core/profile-index.tsx`：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { redirect } from "react-router";
import type { Route } from "./+types/profile-index";

// The profile page itself has no content of its own any more (M1 design 3.2): it redirects to the
// "assigned" tab, so every existing link to /:workspaceSlug/profile/:userId keeps working.
export const clientLoader = ({ params, request }: Route.ClientLoaderArgs) => {
  const searchParams = new URL(request.url).searchParams;
  const query = searchParams.toString();
  throw redirect(`/${params.workspaceSlug}/profile/${params.userId}/assigned${query ? `?${query}` : ""}`);
};

export default function ProfileIndex() {
  return null;
}
```

在 `web/apps/web/app/routes/core.ts` 里把 `:workspaceSlug/profile/:userId` 的 index 路由指向
`./routes/redirects/core/profile-index.tsx`，并删掉 `:workspaceSlug/profile/:userId/activity` 的条目。

- [ ] **Step 2: 重写用户卡片**

把 `web/apps/web/core/components/profile/sidebar.tsx` 整个替换为（数据改从工作区成员 store 读，
四种状态分开显示；时区删掉，成员数据里没有）：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "next/navigation";
import useSWR from "swr";
// plane imports
import { WORKSPACE_MEMBERS } from "@plane/constants";
import { useOutsideClickDetector } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
import { IconButton } from "@plane/propel/icon-button";
import { EditOutline } from "@makeplane/propel/icons";
import { Loader } from "@plane/ui";
import { cn, renderFormattedDate, getFileURL } from "@plane/utils";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useMember } from "@/hooks/store/use-member";
import { useUser } from "@/hooks/store/user";

type TProfileSidebar = {
  className?: string;
};

export const ProfileSidebar = observer(function ProfileSidebar(props: TProfileSidebar) {
  const { className = "" } = props;
  // refs
  const ref = useRef<HTMLDivElement>(null);
  // router
  const { workspaceSlug, userId } = useParams();
  // store hooks
  const { data: currentUser } = useUser();
  const { profileSidebarCollapsed, toggleProfileSidebar } = useAppTheme();
  const { toggleProfileSettingsModal } = useCommandPalette();
  const {
    workspace: { fetchWorkspaceMembers, getWorkspaceMemberDetails },
  } = useMember();
  const { t } = useTranslation();

  const slug = workspaceSlug?.toString() ?? "";
  // The workspace wrapper already fetches the members under this key, so SWR serves the same request;
  // subscribing here is what tells the card whether the members are still loading or failed to load.
  const { error, isLoading } = useSWR(
    slug ? WORKSPACE_MEMBERS(slug) : null,
    slug ? () => fetchWorkspaceMembers(slug) : null,
    {
      revalidateIfStale: false,
      revalidateOnFocus: false,
    }
  );
  const memberDetails = userId ? getWorkspaceMemberDetails(userId.toString()) : null;
  const userData = memberDetails?.member;

  useOutsideClickDetector(ref, () => {
    if (profileSidebarCollapsed === false) {
      if (window.innerWidth < 768) {
        toggleProfileSidebar();
      }
    }
  });

  useEffect(() => {
    const handleToggleProfileSidebar = () => {
      if (window && window.innerWidth < 768) {
        toggleProfileSidebar(true);
      }
      if (window && profileSidebarCollapsed && window.innerWidth >= 768) {
        toggleProfileSidebar(false);
      }
    };

    window.addEventListener("resize", handleToggleProfileSidebar);
    handleToggleProfileSidebar();
    return () => window.removeEventListener("resize", handleToggleProfileSidebar);
  }, []);

  const renderContent = () => {
    if (error) return <div className="px-5 py-6 text-13 text-secondary">{t("profile.details.load_failed")}</div>;
    if (isLoading && !userData)
      return (
        <Loader className="space-y-7 px-5 py-6">
          <Loader.Item height="52px" width="52px" />
          <div className="space-y-5">
            <Loader.Item height="20px" />
            <Loader.Item height="20px" />
          </div>
        </Loader>
      );
    if (!userData) return <div className="px-5 py-6 text-13 text-secondary">{t("profile.details.not_a_member")}</div>;
    return (
      <div className="px-5 py-6">
        <div className="flex items-center gap-4">
          <div className="h-[52px] w-[52px] flex-shrink-0 rounded-sm">
            {userData.avatar_url && userData.avatar_url !== "" ? (
              <img
                src={getFileURL(userData.avatar_url)}
                alt={userData.display_name}
                className="h-full w-full rounded-sm object-cover"
              />
            ) : (
              <div className="flex h-[52px] w-[52px] items-center justify-center rounded-sm bg-accent-primary text-on-color capitalize">
                {userData.first_name?.[0]}
              </div>
            )}
          </div>
          <div className="min-w-0">
            <h4 className="truncate text-16 font-semibold">
              {userData.first_name} {userData.last_name}
            </h4>
            <h6 className="truncate text-13 text-secondary">({userData.display_name})</h6>
          </div>
          {currentUser?.id === userId && (
            <div className="ml-auto">
              <IconButton
                variant="secondary"
                icon={EditOutline}
                onClick={() =>
                  toggleProfileSettingsModal({
                    activeTab: "general",
                    isOpen: true,
                  })
                }
              />
            </div>
          )}
        </div>
        <div className="mt-6 flex items-center gap-4 text-13">
          <div className="w-2/5 flex-shrink-0 text-secondary">{t("profile.details.joined_on")}</div>
          <div className="w-3/5 font-medium break-words">{renderFormattedDate(userData.joining_date ?? "")}</div>
        </div>
      </div>
    );
  };

  return (
    <div
      ref={ref}
      className={cn(
        `vertical-scrollbar fixed z-5 scrollbar-md h-full w-full shrink-0 overflow-hidden overflow-y-auto border-l border-subtle bg-surface-1 shadow-raised-200 transition-all md:relative md:w-[300px]`,
        className
      )}
      style={profileSidebarCollapsed ? { marginLeft: `${window?.innerWidth || 0}px` } : {}}
    >
      {renderContent()}
    </div>
  );
});
```

`profile/[userId]/header.tsx` 里的用户名同样改从 `getWorkspaceMemberDetails` 取，不再调统计接口。

- [ ] **Step 3: 删文件**

Run:

```
rm -r web/apps/web/core/components/profile/overview \
      web/apps/web/core/components/profile/activity \
      "web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/activity" \
      web/packages/propel/src/charts/bar-chart \
      web/packages/propel/src/charts/pie-chart
rm "web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/page.tsx" \
   web/apps/web/core/components/profile/time.tsx \
   web/apps/web/core/components/core/activity.tsx \
   web/apps/web/core/components/ui/loader/settings/activity.tsx
```

- [ ] **Step 4: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

要删的东西：

- `constants/src/profile.ts`：`PROFILE_VIEWER_TAB` 和 `PROFILE_ADMINS_TAB` 合并成一个
  `PROFILE_TABS`（`assigned`、`created`、`subscribed`），`navbar.tsx`、`layout.tsx` 改用它。
- `core/services/user.service.ts`：`getUserProfileData`、`getUserProfileProjectsSegregation`、
  `getUserProfileActivity`、`downloadProfileActivity`。
- `constants/src/fetch-keys.ts`：`USER_PROFILE_DATA`、`USER_PROFILE_ACTIVITY`、
  `USER_PROFILE_PROJECT_SEGREGATION`。
- `types/src/users.ts`：`IUserProfileData`、`IUserProfileProjectSegregation`、
  `IUserStateDistribution`、`IUserPriorityDistribution`、`IUserActivity`、`IUserActivityResponse`。
- `types/src/charts/index.ts`、`propel/src/charts/components/tick.tsx`：柱状图和饼图的类型与刻度。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 5: 新增仓库内单元测试**

写入 `web/packages/constants/src/navigation.test.ts`：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import { PROFILE_TABS } from "./profile";
import { WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS, WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS } from "./workspace";

// The sidebar is a fixed list (M1 design 3.3) and the profile page keeps only its three work-item tabs
// (M1 design 3.2). Both lists are the only navigation entry of the features they name, so a deletion
// that empties one of them has to fail here rather than silently remove the last way in.
describe("the workspace sidebar is a fixed list", () => {
  it("shows these items above the workspace group, in this order", () => {
    expect(WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS.map((item) => item.key)).toEqual([
      "home",
      "stickies",
      "your_work",
      "drafts",
    ]);
  });

  it("shows these items inside the workspace group, in this order", () => {
    expect(WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS.map((item) => item.key)).toEqual([
      "projects",
      "views",
      "archives",
    ]);
  });

  it("gives every item a label, a link and at least one role", () => {
    for (const item of [
      ...WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS,
      ...WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS,
    ]) {
      expect(item.labelTranslationKey).not.toBe("");
      expect(item.href.startsWith("/")).toBe(true);
      expect(item.access.length).toBeGreaterThan(0);
    }
  });
});

describe("the profile page", () => {
  it("keeps exactly the three work-item tabs", () => {
    expect(PROFILE_TABS.map((tab) => tab.key)).toEqual(["assigned", "created", "subscribed"]);
  });

  it("points each tab at its own route", () => {
    for (const tab of PROFILE_TABS) {
      expect(tab.route).toBe(tab.key);
      expect(tab.selected).toBe(`/${tab.key}/`);
    }
  });
});
```

Run:

```
node /tmp/nerve-p2/pkg.mjs web/packages/constants/package.json set-script test "vitest run"
node /tmp/nerve-p2/pkg.mjs web/packages/constants/package.json add-dev-dep vitest "catalog:"
pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs
```

Expected: 只有一行 `importers: 0 dependencies removed, 1 added or changed` 加它下面的 `+ web/packages/constants devDependencies vitest: …`

- [ ] **Step 6: 文案**

Run:

```
node /tmp/nerve-p2/i18n-del.mjs settings profile.details.time_zone profile.stats profile.actions profile.tabs.summary profile.tabs.activity profile.empty_state.activity
node /tmp/nerve-p2/i18n-del.mjs empty-state workspace_empty_state.your_work_by_priority workspace_empty_state.your_work_by_state
```

再往 `settings.json` 的 `profile.details` 里补两个键（中英文各一份）：

```json
"load_failed": "Could not load this workspace's members.",
"not_a_member": "This user is not a member of this workspace."
```

中文：`"load_failed": "无法加载该工作区的成员。"`、`"not_a_member": "该用户不是这个工作区的成员。"`

- [ ] **Step 7: 守卫规则**

在 `rules` 末尾追加 `profile-stats`（`content.source` 为
`user-stats|user-profile/|user-activity/|IUserProfileData|IUserProfileProjectSegregation|IUserStateDistribution|IUserPriorityDistribution|\bIUserActivity\b|IUserActivityResponse|USER_PROFILE_(?:DATA|ACTIVITY|PROJECT_SEGREGATION)|profile\.stats|profile\.tabs\.(?:summary|activity)|PROFILE_(?:VIEWER|ADMINS)_TAB|ProfileActivityListPage|WorkspaceActivityList|downloadProfileActivity|your_work_by_`，
`flags` 为空；miss 样本要包含 `import UserActivityIcon from "./user-activity-icon";`
和 `/api/workspaces/${workspaceSlug}/user-issues/${userId}/`，它们是保留代码）。

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 16 rules, 5 exceptions, no hits.`

- [ ] **Step 8: 降上限、门禁与提交**

Run:

```
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 765"
node /tmp/nerve-p2/pkg.mjs web/packages/propel/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 21"
```

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 16 rules, 5 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:`
Expected: `Tasks:   13 successful, 13 total`

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 476 files, 9997650 bytes`、
`css: 3 files, 318759 bytes`、`other: 136 files, 8961872 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `42 files changed, 231 insertions(+), 2898 deletions(-)`

提交信息：

```
feat(web): the profile page keeps its work-item tabs and a user card

The per-project counts, the state and priority charts and the activity
page go (M1 design 3.2). What is left is the three work-item tabs and a
card that reads the workspace member store, so the page no longer calls
an endpoint of its own; loading, a failed load and "not a member of this
workspace" each say so rather than rendering an empty name.

The page address itself redirects to the assigned tab, so the sidebar,
g y, the member list and every mention keep working unchanged.

The bar and pie charts in propel had no other caller once the analytics
went, so they go here.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 4: 估算；迭代和模块的进度改为只按工作项数计算

**Files**

- Create：`web/packages/utils/src/progress.test.ts`
- Delete（53）：
  - `app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/estimates/{header,page}.tsx`
  - `core/components/estimates/`（23，含 `create/`、`delete/`、`inputs/`、`points/`）
  - `core/store/estimates/`（3）、`core/hooks/store/estimates/`（4）、`core/services/estimate.service.ts`
  - `core/components/dropdowns/estimate.tsx`、`core/components/readonly/estimate.tsx`、
    `core/components/cycles/dropdowns/estimate-type-dropdown.tsx`、
    `core/components/issues/issue-layouts/spreadsheet/columns/estimate-column.tsx`、
    `core/components/issues/issue-detail/issue-activity/activity/actions/estimate.tsx`、
    `core/components/power-k/ui/pages/context-based/work-item/estimates-menu.tsx`
  - `helpers/issue-filter.helper.ts`
  - `packages/types/src/estimate.ts`、`packages/constants/src/estimates.ts`、`packages/utils/src/estimates.ts`
  - `packages/propel/src/icons/properties/estimate-icon.tsx`、
    `packages/propel/src/empty-state/assets/horizontal-stack/estimate.tsx`
  - 图片 8：`app/assets/empty-state/estimates/`（2）、
    `app/assets/empty-state/project-settings/estimates-*`（6）
- Modify（87）：见下面的步骤；关键的是
  `packages/utils/src/{cycle.ts,distribution-update.ts,index.ts}`、
  `packages/constants/src/state.ts`、`core/store/{cycle.store.ts,module.store.ts,root.store.ts}`、
  `core/store/issue/helpers/base-issues.store.ts`、表格布局 6 个文件、
  `core/components/cycles/progress-sidebar/`（5）、`core/components/modules/progress-sidebar/`（3）

**Interfaces**

- Consumes：Task 2 改名后的 progress 组件与 store 方法。
- Produces：只按工作项数计算的 `calculateCycleProgress` 和 `updateDistribution`；
  仓库内单元测试 `progress.test.ts`；守卫规则 `estimates` 及 2 条例外。
- 顺序约束（M1 设计 9 节，硬）：**数量计算和旧的估算字段必须在同一个 Task 里改**。

**Steps**

- [ ] **Step 1: 删文件与路由条目**

Run:

```
rm -r web/apps/web/core/components/estimates \
      web/apps/web/core/store/estimates \
      web/apps/web/core/hooks/store/estimates \
      "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/estimates" \
      web/apps/web/app/assets/empty-state/estimates
rm web/apps/web/core/services/estimate.service.ts \
   web/apps/web/core/components/dropdowns/estimate.tsx \
   web/apps/web/core/components/readonly/estimate.tsx \
   web/apps/web/core/components/cycles/dropdowns/estimate-type-dropdown.tsx \
   web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/estimate-column.tsx \
   web/apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/estimate.tsx \
   web/apps/web/core/components/power-k/ui/pages/context-based/work-item/estimates-menu.tsx \
   web/apps/web/helpers/issue-filter.helper.ts \
   web/apps/web/app/assets/empty-state/project-settings/estimates-dark.png \
   web/apps/web/app/assets/empty-state/project-settings/estimates-light.png \
   web/apps/web/app/assets/empty-state/project-settings/estimates-dark.webp \
   web/apps/web/app/assets/empty-state/project-settings/estimates-light.webp \
   web/apps/web/app/assets/empty-state/project-settings/estimates-dark-resp.webp \
   web/apps/web/app/assets/empty-state/project-settings/estimates-light-resp.webp \
   web/packages/types/src/estimate.ts \
   web/packages/constants/src/estimates.ts \
   web/packages/utils/src/estimates.ts \
   web/packages/propel/src/icons/properties/estimate-icon.tsx \
   web/packages/propel/src/empty-state/assets/horizontal-stack/estimate.tsx
```

**同一步**删掉 `web/apps/web/app/routes/core.ts` 里 `settings/projects/:projectId/estimates` 的条目。

- [ ] **Step 2: 改数量进度的三处计算**

`web/packages/utils/src/distribution-update.ts`：

- `getDistributionPathsPostUpdate` 去掉 `estimatePointById` 参数；
- 去掉 `total_estimates`、`completed_estimates`、`backlog_estimates` 这类字段的路径；
- 去掉 `total_estimate_points` 的路径和 `estimate_distribution.completion_chart` 的那一段；
- 去掉按点数累加负责人和标签的两段代码（按工作项数的两段保留）。

`web/packages/utils/src/cycle.ts` 只保留 `orderCycles`、`shouldFilterCycle`、
`calculateCycleProgress`；删掉零调用方的 `scope`、`ideal`、`formatV1Data`、`formatV2Data`、
`formatActiveCycle`。`calculateCycleProgress` 去掉 `estimateType` 参数后是：

```ts
export const calculateCycleProgress = (cycle: ICycle | undefined, includeInProgress: boolean = false): number => {
  if (!cycle) return 0;
  const progressSnapshot: TProgressSnapshot | undefined = cycle.progress_snapshot;
  const cycleDetails = progressSnapshot && !isEmpty(progressSnapshot) ? progressSnapshot : cycle;
  let completed = cycleDetails.completed_issues || 0;
  const cancelled = cycleDetails.cancelled_issues || 0;
  const total = cycleDetails.total_issues || 0;
  if (includeInProgress) {
    completed += cycleDetails.started_issues || 0;
  }
  const adjustedTotal = total - cancelled;
  if (adjustedTotal <= 0) return 0;
  return Math.round((completed / adjustedTotal) * 100);
};
```

`web/packages/constants/src/state.ts`：`STATE_DISTRIBUTION` 的每一行去掉 `points`。

- [ ] **Step 3: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

要删的东西：

- `types/src/cycle/cycle.ts`、`types/src/module/modules.ts`：`TCycleEstimateDistribution`、
  六个 `*_estimate_points`、`TCycleEstimateType`、`TCyclePlotType`、`TModulePlotType`。
- `types/src/issues/issue.ts`、`types/src/project/projects.ts`、`types/src/workspace.ts`、
  `types/src/workspace-draft-issues/base.ts`、`types/src/view-props.ts`、`types/src/enums.ts`、
  `types/src/settings.ts`：`estimate_point`、`estimate_id`、`EEstimateSystem` 等。
- `core/store/cycle.store.ts`：`plotType`、`estimatedType`、`getPlotTypeByCycleId`、
  `getEstimateTypeByCycleId`、`setPlotType`、`setEstimateType`、`getIsPointsDataAvailable`；
  `fetchActiveCycleDistribution` 去掉 `analytic_type` 参数。
- `core/store/module.store.ts`：`plotType` 三件套。
- `core/store/root.store.ts`：`projectEstimate` 的接线和 `resetOnSignOut`。
- `core/store/issue/helpers/base-issues.store.ts`：两处 `estimate_point__key` 的排序和
  `populateIssueDataForSorting` 里的 `estimate_point` 分支。
- 表格布局：`isEstimateEnabled` 从 `spreadsheet-view.tsx` 一路穿到 `issue-column.tsx` 的整条 prop 链
  （`spreadsheet-table.tsx`、`spreadsheet-header.tsx`、`spreadsheet-header-column.tsx`、
  `issue-row.tsx`）；`shouldRenderColumn` 随 `helpers/issue-filter.helper.ts` 一起删（它只判断估算这一列）。
- `constants/src/issue/{common,modal}.ts`、`constants/src/tab-indices.ts`、
  `constants/src/settings/project.ts`、`constants/src/fetch-keys.ts`、`constants/src/index.ts`：
  估算的属性、标签页、设置项和 fetch key。
- `core/components/settings/project/sidebar/item-icon.tsx`、
  `propel/src/icons/{registry.ts,properties/index.ts}`、
  `propel/src/empty-state/assets/{asset-registry.tsx,asset-types.ts,horizontal-stack/index.ts}`：
  图标与插图的注册表。
- power-k 的 `core/types.ts`、`ui/modal/constants.ts`、
  `ui/pages/context-based/work-item/{commands.ts,root.tsx}`：估算菜单。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 4: 新增仓库内单元测试**

写入 `web/packages/utils/src/progress.test.ts`（13 个测试，覆盖 M1 设计 7.5 "P2 进度"那一行）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import type { ICycle, IModule, IState, TIssue } from "@plane/types";
import { calculateCycleProgress } from "./cycle";
import { getDistributionPathsPostUpdate, updateDistribution } from "./distribution-update";

// Cycle and module progress is a work-item count and nothing else (M1 design 3.4). These tests pin
// the two calculations the product depends on: the percentage a cycle reports, and the optimistic
// update that runs when a work item is created, completed, cancelled or reopened.

const cycle = (fields: Partial<ICycle>) => fields as ICycle;

describe("calculateCycleProgress", () => {
  it("is zero without a cycle", () => {
    expect(calculateCycleProgress(undefined)).toBe(0);
  });

  it("is zero for an empty cycle", () => {
    expect(calculateCycleProgress(cycle({ total_issues: 0, completed_issues: 0, cancelled_issues: 0 }))).toBe(0);
  });

  it("leaves the cancelled work items out of the total", () => {
    // 3 of the 8 remaining work items are done: 6 of 10 were not cancelled.
    expect(calculateCycleProgress(cycle({ total_issues: 10, completed_issues: 3, cancelled_issues: 2 }))).toBe(38);
  });

  it("counts the started work items as well when asked to", () => {
    expect(
      calculateCycleProgress(
        cycle({ total_issues: 10, completed_issues: 3, cancelled_issues: 2, started_issues: 1 }),
        true
      )
    ).toBe(50);
  });

  it("reports 100 once everything that was not cancelled is done", () => {
    expect(calculateCycleProgress(cycle({ total_issues: 10, completed_issues: 8, cancelled_issues: 2 }))).toBe(100);
  });

  it("reports zero when every work item was cancelled", () => {
    expect(calculateCycleProgress(cycle({ total_issues: 4, completed_issues: 0, cancelled_issues: 4 }))).toBe(0);
  });

  it("prefers the snapshot of a completed cycle over the live counts", () => {
    const completed = cycle({
      total_issues: 100,
      completed_issues: 0,
      cancelled_issues: 0,
      progress_snapshot: { total_issues: 4, completed_issues: 2, cancelled_issues: 0 } as ICycle["progress_snapshot"],
    });
    expect(calculateCycleProgress(completed)).toBe(50);
  });
});

const STATE_MAP: Record<string, IState> = {
  backlog: { id: "backlog", group: "backlog" } as IState,
  started: { id: "started", group: "started" } as IState,
  done: { id: "done", group: "completed" } as IState,
  cancelled: { id: "cancelled", group: "cancelled" } as IState,
};

const workItem = (stateId: string, fields: Partial<TIssue> = {}) =>
  ({
    id: "work-item-1",
    state_id: stateId,
    assignee_ids: [],
    label_ids: [],
    completed_at: null,
    ...fields,
  }) as TIssue;

const emptyCycle = () =>
  ({
    total_issues: 0,
    backlog_issues: 0,
    unstarted_issues: 0,
    started_issues: 0,
    completed_issues: 0,
    cancelled_issues: 0,
    distribution: { assignees: [], labels: [], completion_chart: {} },
  }) as unknown as ICycle;

describe("the optimistic count update", () => {
  it("adds a new work item to the total and to its own state group", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("backlog"), STATE_MAP));
    expect(target.total_issues).toBe(1);
    expect(target.backlog_issues).toBe(1);
    expect(target.completed_issues).toBe(0);
  });

  it("moves the count between state groups when a work item is completed", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("started"), STATE_MAP));
    updateDistribution(
      target,
      getDistributionPathsPostUpdate(workItem("started"), workItem("done", { completed_at: "2026-01-02" }), STATE_MAP)
    );
    expect(target.total_issues).toBe(1);
    expect(target.started_issues).toBe(0);
    expect(target.completed_issues).toBe(1);
  });

  it("moves it back when the work item is reopened", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("done"), STATE_MAP));
    updateDistribution(target, getDistributionPathsPostUpdate(workItem("done"), workItem("started"), STATE_MAP));
    expect(target.completed_issues).toBe(0);
    expect(target.started_issues).toBe(1);
    expect(target.total_issues).toBe(1);
  });

  it("takes a removed work item out of the total", () => {
    const target = emptyCycle();
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, workItem("cancelled"), STATE_MAP));
    updateDistribution(target, getDistributionPathsPostUpdate(workItem("cancelled"), undefined, STATE_MAP));
    expect(target.total_issues).toBe(0);
    expect(target.cancelled_issues).toBe(0);
  });

  it("keeps the assignee and label counts of a module in step", () => {
    const target = {
      total_issues: 0,
      backlog_issues: 0,
      unstarted_issues: 0,
      started_issues: 0,
      completed_issues: 0,
      cancelled_issues: 0,
      distribution: {
        assignees: [{ assignee_id: "user-1", completed_issues: 0, pending_issues: 0, total_issues: 0 }],
        labels: [{ label_id: "label-1", completed_issues: 0, pending_issues: 0, total_issues: 0 }],
        completion_chart: {},
      },
    } as unknown as IModule;
    const item = workItem("started", { assignee_ids: ["user-1"], label_ids: ["label-1"] });
    updateDistribution(target, getDistributionPathsPostUpdate(undefined, item, STATE_MAP));
    expect(target.distribution?.assignees[0]).toMatchObject({
      pending_issues: 1,
      completed_issues: 0,
      total_issues: 1,
    });
    expect(target.distribution?.labels[0]).toMatchObject({ pending_issues: 1, completed_issues: 0, total_issues: 1 });

    const done = workItem("done", { assignee_ids: ["user-1"], label_ids: ["label-1"], completed_at: "2026-01-02" });
    updateDistribution(target, getDistributionPathsPostUpdate(item, done, STATE_MAP));
    expect(target.distribution?.assignees[0]).toMatchObject({
      pending_issues: 0,
      completed_issues: 1,
      total_issues: 1,
    });
    expect(target.distribution?.labels[0]).toMatchObject({ pending_issues: 0, completed_issues: 1, total_issues: 1 });
  });

  it("takes the completed work item off the burn-down chart on the day it was completed", () => {
    const target = emptyCycle();
    // the chart only carries the days the cycle already knows about, so seed the day under test
    target.distribution!.completion_chart = { "2026-01-02": 5 };
    const started = workItem("started");
    const done = workItem("done", { completed_at: "2026-01-02T10:00:00Z" });
    updateDistribution(target, getDistributionPathsPostUpdate(started, done, STATE_MAP));
    expect(target.distribution?.completion_chart["2026-01-02"]).toBe(4);
  });
});
```

Run:

```
node /tmp/nerve-p2/pkg.mjs web/packages/utils/package.json set-script test "vitest run"
node /tmp/nerve-p2/pkg.mjs web/packages/utils/package.json add-dev-dep vitest "catalog:"
pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs
```

Expected: `importers: 0 dependencies removed, 1 added or changed` 加它下面的
`+ web/packages/utils devDependencies vitest: …`

Run: `pnpm --dir web/packages/utils exec vitest run 2>&1 | tail -4`
Expected: `Tests  13 passed (13)`

- [ ] **Step 5: 文案**

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t4.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-t2.txt /tmp/nerve-p2/keys-t4.txt`

Run:

```
node /tmp/nerve-p2/i18n-del.mjs common estimate points common.estimate common.estimates common.coming_soon
node /tmp/nerve-p2/i18n-del.mjs empty-state settings_empty_state.estimates
node /tmp/nerve-p2/i18n-del.mjs power-k contextual_actions.work_item.change_estimate page_placeholders.update_work_item_estimate
node /tmp/nerve-p2/i18n-del.mjs project-settings project_settings.estimates project_settings.empty_state.estimates
```

再把 `common.json` 的 `property_changes_description` 中英文里的"估算 / estimate"去掉（这条文案列举
可改的属性，估算已经不在其中）。

- [ ] **Step 6: 守卫规则与例外**

在 `rules` 末尾追加：

```json
{
  "id": "estimates",
  "phase": "M1/P2",
  "why": "估算体系（估算设置页、估算下拉框、estimate_point 字段、按点数的进度和燃尽）；迭代和模块的进度改为只按工作项数计算（M1 设计 3.4）",
  "files": { "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$", "flags": "" },
  "content": { "source": "estimat|估算", "flags": "i" },
  "samples": {
    "hit": [
      "import { EstimateDropdown } from \"@/components/dropdowns/estimate\";",
      "  estimate_point: string | null;",
      "export enum EEstimateSystem {",
      "  const { areEstimateEnabledByProjectId } = useProjectEstimates();",
      "  \"estimate\": \"估算\",",
      "      total: assignee.total_estimates,"
    ],
    "miss": [
      "export const calculateCycleProgress = (cycle: ICycle | undefined, includeInProgress: boolean = false): number => {",
      "  const totalIssues = cycleDetails?.total_issues || 0;",
      "export type TCycleDistribution = {",
      "  \"points\": \"Points\","
    ],
    "files": {
      "hit": ["web/apps/web/core/store/cycle.store.ts", "web/packages/i18n/src/locales/en/common.json"],
      "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
    }
  }
}
```

例外（**`match` 写正则实际匹配到的那段文本**，不是整词）：

```json
[
  { "rule": "estimates", "path": "web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx", "match": "estimat", "reason": "通知卡片里的 estimate_time 是工时，随工时空壳在 P3 删除", "until": "M1/P3" },
  { "rule": "estimates", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "Estimat", "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" }
]
```

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 17 rules, 7 exceptions, no hits.`

- [ ] **Step 7: 降上限、门禁与提交**

Run: `pnpm exec turbo run check:lint 2>&1 | grep -E "fewer than|more than" || echo none`

清理 `no-unused-vars`（实测 5 处：`all-properties.tsx` 的 `projectId`、
power-k `commands.ts` 的 `Triangle`、`cycle.store.ts` 的 `isEmpty`，以及 Task 2 步骤 3 已经
处理掉的两个 mobile-header），再写回上限：

```
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 737"
node /tmp/nerve-p2/pkg.mjs web/packages/utils/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 33"
```

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 17 rules, 7 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:`
Expected: `Tasks:   14 successful, 14 total`

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 478 files, 9929020 bytes`、
`css: 3 files, 318199 bytes`、`other: 136 files, 8961872 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `141 files changed, 367 insertions(+), 5158 deletions(-)`

提交信息：

```
feat(web): delete the estimates and make progress a work-item count

Estimates go: the project settings page, the dropdown, the property, the
spreadsheet column, the activity entry and the stores behind them.

Cycle and module progress therefore counts work items and nothing else
(M1 design 3.4): the points/count switch, the point-only burn-down
states and the point arithmetic of distribution-update all go with them.
The calculation now has unit tests of its own, for the percentage a
cycle reports and for the optimistic update that runs when a work item
is created, completed, cancelled or reopened.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 5: 甘特图与模块时间线

**Files**

- Rename：`web/packages/constants/src/gantt-chart.ts` → `web/packages/constants/src/issue/relation.ts`
- Delete（73）：
  - `core/components/gantt-chart/`（34）、`core/components/issues/issue-layouts/gantt/`（3）、
    `core/components/modules/gantt-chart/`（3）、`core/components/base-layouts/gantt/`（3）
  - `core/components/issues/issue-layouts/quick-add/{button,form}/gantt.tsx`
  - `core/components/ui/loader/layouts/gantt-layout-loader.tsx`
  - `core/hooks/use-timeline-chart.ts`、`core/store/timeline/`（4）、
    `core/store/issue/issue_gantt_view.store.ts`
  - `packages/types/src/layout/`（2）、`packages/types/src/base-layouts/gantt/`（3）
  - `packages/propel/src/icons/layouts/timeline-icon.tsx`
  - 图片 10：`app/assets/empty-state/{cycle-issues,module-issues}/gantt_chart-*`（8）、
    `app/assets/empty-state/empty-filters/gantt_chart-{dark,light}.webp`
- Modify（60）：见步骤

**Interfaces**

- Consumes：Task 4 之后的工作项 store 与布局常量。
- Produces：`constants/src/issue/relation.ts`（`REVERSE_RELATIONS`）；守卫规则 `gantt-timeline` 及 3 条例外。
- 顺序约束（M1 设计 9 节，硬）：**先搬 `REVERSE_RELATIONS`**。

**Steps**

- [ ] **Step 1: 先搬 `REVERSE_RELATIONS`**

Run: `git mv web/packages/constants/src/gantt-chart.ts web/packages/constants/src/issue/relation.ts`

把文件里只给甘特图用的常量（列宽、缩放档位等）删掉，只留 `REVERSE_RELATIONS`；
`constants/src/index.ts` 里 `./gantt-chart` 的导出删掉，`constants/src/issue/index.ts` 加上 `./relation`。

Run: `pnpm exec turbo run check:types 2>&1 | grep -E "error TS|Tasks:"`
Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 2: 删文件与路由/布局条目**

Run:

```
rm -r web/apps/web/core/components/gantt-chart \
      web/apps/web/core/components/issues/issue-layouts/gantt \
      web/apps/web/core/components/modules/gantt-chart \
      web/apps/web/core/components/base-layouts/gantt \
      web/apps/web/core/store/timeline \
      web/packages/types/src/layout \
      web/packages/types/src/base-layouts/gantt
rm web/apps/web/core/components/issues/issue-layouts/quick-add/button/gantt.tsx \
   web/apps/web/core/components/issues/issue-layouts/quick-add/form/gantt.tsx \
   web/apps/web/core/components/ui/loader/layouts/gantt-layout-loader.tsx \
   web/apps/web/core/hooks/use-timeline-chart.ts \
   web/apps/web/core/store/issue/issue_gantt_view.store.ts \
   web/packages/propel/src/icons/layouts/timeline-icon.tsx \
   web/apps/web/app/assets/empty-state/empty-filters/gantt_chart-dark.webp \
   web/apps/web/app/assets/empty-state/empty-filters/gantt_chart-light.webp
rm web/apps/web/app/assets/empty-state/cycle-issues/gantt_chart-*.webp \
   web/apps/web/app/assets/empty-state/module-issues/gantt_chart-*.webp
```

- [ ] **Step 3: 按类型检查删到底（`EIssueLayoutTypes.GANTT` 最后删）**

Run: `pnpm exec turbo run check:types`

要删的东西：

- `core/services/issue/issue.service.ts` 的 `updateIssueDates` 和
  `core/store/issue/helpers/base-issues.store.ts` 的 `updateIssueDates`（`POST /issue-dates/`，交接 M4）；
- `ENABLE_ISSUE_DEPENDENCIES`；
- `useTimeLineRelationOptions`：删掉，7 个调用方改成
  `import { ISSUE_RELATION_OPTIONS } from "@/components/relations";`；
- `core/store/root.store.ts` 的时间线 store 接线；
- 各布局根组件、`issue-layout-HOC.tsx`、`layout-icon.tsx`、`module-layout-icon.tsx`、
  `base-layouts/constants.ts`、`quick-add/{button,form}/index.ts` 的甘特分支；
- `constants/src/issue/{filter,layout}.ts`、`constants/src/module.ts`、
  `types/src/view-props.ts`、`types/src/cycle/cycle_filters.ts`、`types/src/module/module_filters.ts`、
  `types/src/base-layouts/base.ts`、`utils/src/work-item/base.ts` 的甘特与时间线布局；
- **最后**把 `EIssueLayoutTypes.GANTT` 从枚举里删掉，再跑一次类型检查，把它列出的每一处菜单、
  图标、筛选、移动端头部都删干净。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 4: 文案**

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t5.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-t4.txt /tmp/nerve-p2/keys-t5.txt`

按输出删除 `common.json`、`project.json`、`work-item.json` 里甘特和时间线布局的名称与空状态文案。

- [ ] **Step 5: 守卫规则与例外**

在 `rules` 末尾追加 `gantt-timeline`：`content.source` 为
`gantt|(?<!view_)time_?line|issue-dates`，`flags` 为 `i`；miss 样本必须含
`      "view_timeline",`（图标选择器里的 Material Symbols 名字）和
`import { ISSUE_RELATION_OPTIONS } from "@/components/relations";`。

例外：

```json
[
  { "rule": "gantt-timeline", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "Gantt", "count": 3, "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "gantt-timeline", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "timeline", "count": 2, "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "gantt-timeline", "path": "web/packages/types/src/publish.ts", "match": "gantt", "count": 2, "reason": "公开发布的视图类型，随公开发布在 P3 删除", "until": "M1/P3" }
]
```

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 18 rules, 10 exceptions, no hits.`

- [ ] **Step 6: 降上限、门禁与提交**

Run: `node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 701"`

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 18 rules, 10 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → `Tasks:   14 successful, 14 total`

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 472 files, 9860670 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `134 files changed, 92 insertions(+), 5313 deletions(-)`

提交信息：

```
feat(web): delete the gantt layout and the module timeline

The gantt chart, the work-item gantt layout, the module timeline layout,
the timeline stores and the POST /issue-dates/ that only dragging a bar
called all go. EIssueLayoutTypes.GANTT is removed last, so the type
checker names every menu, icon and filter that still mentions it.

REVERSE_RELATIONS moves out first: work-item relations stay, and their
constants had no reason to live in a file named after the gantt chart.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 6: "自动化"设置页只留自动归档

**Files**

- Delete：`web/apps/web/core/components/automation/auto-close-automation.tsx`
- Modify（8）：`tools/keywords.json`、
  `app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、
  `core/components/automation/{auto-archive-automation,index,select-month-modal}.tsx`、
  `packages/types/src/project/projects.ts`、`packages/i18n/src/locales/{en,zh-CN}/project-settings.json`

**Interfaces**

- Produces：守卫规则 `auto-close`；交接 M3 的 `IProject.close_in`、`IProject.default_state`。

**Steps**

- [ ] **Step 1: 删除**

Run: `rm web/apps/web/core/components/automation/auto-close-automation.tsx`

`select-month-modal.tsx` 去掉 `type` prop 和 `close_in` 的分支，只剩归档月份；
`index.ts` 去掉导出；`automations/page.tsx` 只渲染 `AutoArchiveAutomation`；
`IProject` 去掉 `close_in` 和 `default_state`（`default_state` 在 Plane 里是自动关闭的目标状态，
不是新建工作项的默认状态）。

Run: `pnpm exec turbo run check:types 2>&1 | grep -E "error TS|Tasks:"`
Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 2: 文案与守卫**

Run: `node /tmp/nerve-p2/i18n-del.mjs project-settings project_settings.automations.auto-close`

在 `rules` 末尾追加 `auto-close`：`content.source` 为
`close_in|auto[-_ ]?close|default_state`，`flags` 为 `i`；miss 样本含 `  archive_in?: number;`
和 `  const defaultState = projectStates?.[0]?.id;`。

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 19 rules, 10 exceptions, no hits.`

- [ ] **Step 3: 门禁与提交**

上限不变（web 仍是 701）。

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 19 rules, 10 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → 14；`make build-web 2>&1 | grep Tasks:` → 11

Run: `git add -A && git diff --cached --shortstat`
Expected: `9 files changed, 69 insertions(+), 291 deletions(-)`

提交信息：

```
feat(web): the automation settings keep auto-archive only

Auto-close closed a work item that had not moved for a number of months,
by moving it to a state the project picked for the purpose. Both that
setting and the state it pointed at go; auto-archive stays, and the
month picker now serves it alone.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 7: 文档页、协作编辑、更新日志

本 Phase 最大的一个 Task（实测删 203 个文件）。分成"web 应用 → 编辑器包 → 依赖 → 文案"四段做，
每段之后跑一次 `check:types`。

**Files**

- Delete（203），按段：
  - **路由 8**：`app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/pages/`（6）、
    `app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/pages/`（2）
  - **组件约 60**：`core/components/pages/`（整目录）、`core/components/editor/pdf/`（2）、
    `core/components/editor/document/editor.tsx`、
    `core/components/global/product-updates/`（6，更新日志）、
    `core/components/common/{latest-feature-block,page-access-icon}.tsx`、
    `core/components/home/widgets/recents/page.tsx`、
    `core/components/power-k/ui/pages/context-based/page/commands.ts`、
    `core/components/ui/loader/pages-loader.tsx`
  - **store / service / hook 20**：`core/store/pages/`（5）、`core/services/page/`（3）、
    `core/hooks/pages/`（3）、`core/hooks/store/{use-page,use-page-store}.ts`、
    `core/hooks/{use-collaborative-page-actions,use-page-fallback,use-page-filters,use-page-flag,use-page-operations,use-realtime-page-events}.ts(x)`
  - **编辑器包 32**：`packages/editor/src/components/editors/document/`（5）、
    `packages/editor/src/components/document-editor-side-effects.ts`、
    `packages/editor/src/contexts/`（2）、
    `packages/editor/src/extensions/{core-without-props.ts,document-extensions.tsx,headings-list.ts,title-extension.ts}`、
    `packages/editor/src/extensions/code/without-props.tsx`、
    `packages/editor/src/extensions/core/without-props.ts`、
    `packages/editor/src/extensions/work-item-embed/`（4）、
    `packages/editor/src/helpers/{get-document-server-event.ts,parser.ts,yjs-utils.ts}`、
    `packages/editor/src/hooks/{use-collaborative-editor,use-editor-navigation,use-title-editor,use-yjs-setup}.ts`、
    `packages/editor/src/constants/document-collaborative-events.ts`、
    `packages/editor/src/styles/title-editor.css`、
    `packages/editor/src/types/{collaboration,document-collaborative-events,embed,issue-embed}.ts`、
    `packages/editor/src/lib.ts`
  - **类型、常量、工具、服务 8**：`packages/types/src/page/`（3）、`packages/constants/src/page.ts`、
    `packages/utils/src/page.ts`、`packages/services/src/live.service.ts`、
    `packages/propel/src/icons/{project/page-icon.tsx,sub-brand/wiki-icon.tsx}`、
    `packages/propel/src/empty-state/assets/vertical-stack/{changelog,page}.tsx`
  - **文案 2**：`packages/i18n/src/locales/{en,zh-CN}/page.json`
  - **图片约 70**：`app/assets/empty-state/wiki/`（16）、`app/assets/empty-state/onboarding/`（32）、
    `app/assets/empty-state/disabled-feature/pages-{dark,light}.webp`、
    `app/assets/empty-state/empty_page.png`、`app/assets/onboarding/{onboarding-pages,pages}.webp`
- Modify（110）：见步骤

**Interfaces**

- Consumes：Task 5 之后的工作项 store 和路由表。
- Produces：不带协作的 `EditorRefApi`（`getDocument().binary`、`getDocumentInfo`、`getHeadings`、
  `emitRealTimeUpdate`、`listenToRealTimeUpdate`、`onDocumentInfoChange`、`onHeadingChange`、
  `scrollSummary`、`setProviderDocument` 全部去掉）；`getEditorRefHelpers` 不再有 `provider` 参数；
  `UniqueID` 不再有 `provider`；`EditorContainer` 不再有 `provider` / `state`；
  守卫规则 `pages-collaboration`、`changelog` 及 4 条例外。
- 顺序约束（M1 设计 9 节，硬）：文档页必须在编辑器收敛（Task 8）之前，
  也必须在首页（Task 10）之前。

**Steps**

- [ ] **Step 1: 删 web 应用里的文档页与更新日志**

Run:

```
rm -r "web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/pages" \
      "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/pages" \
      web/apps/web/core/components/pages \
      web/apps/web/core/components/editor/pdf \
      web/apps/web/core/components/global/product-updates \
      web/apps/web/core/store/pages \
      web/apps/web/core/services/page \
      web/apps/web/core/hooks/pages \
      web/apps/web/app/assets/empty-state/wiki \
      web/apps/web/app/assets/empty-state/onboarding \
      web/apps/web/app/assets/onboarding
rm web/apps/web/core/components/editor/document/editor.tsx \
   web/apps/web/core/components/common/latest-feature-block.tsx \
   web/apps/web/core/components/common/page-access-icon.tsx \
   web/apps/web/core/components/home/widgets/recents/page.tsx \
   web/apps/web/core/components/power-k/ui/pages/context-based/page/commands.ts \
   web/apps/web/core/components/ui/loader/pages-loader.tsx \
   web/apps/web/core/hooks/store/use-page.ts \
   web/apps/web/core/hooks/store/use-page-store.ts \
   web/apps/web/core/hooks/use-collaborative-page-actions.tsx \
   web/apps/web/core/hooks/use-page-fallback.ts \
   web/apps/web/core/hooks/use-page-filters.ts \
   web/apps/web/core/hooks/use-page-flag.ts \
   web/apps/web/core/hooks/use-page-operations.ts \
   web/apps/web/core/hooks/use-realtime-page-events.tsx \
   web/apps/web/app/assets/empty-state/empty_page.png \
   web/apps/web/app/assets/empty-state/disabled-feature/pages-dark.webp \
   web/apps/web/app/assets/empty-state/disabled-feature/pages-light.webp
```

**同一步**删掉 `web/apps/web/app/routes/core.ts` 里 `projects/:projectId/pages`（列表和详情）和
`settings/projects/:projectId/features/pages` 的全部条目。

Run: `pnpm exec turbo run check:types`

按报错删：`core/store/root.store.ts` 的 `projectPages` 接线、
`core/store/base-command-palette.store.ts` 的建页弹窗、`core/store/favorite.store.ts` 的 `page` 分支、
`constants/src/{project,sidebar-favorites,tab-indices,workspace,endpoints}.ts` 和
`constants/src/settings/project.ts` 的文档页条目（`RESTRICTED_URLS` 去掉 `pages`）、
power-k 的上下文检测与命令、`navigation/{tab-navigation-utils,use-navigation-items,use-tab-preferences}.ts`、
`project/settings/features-list.tsx`、`workspace/sidebar/project-navigation.tsx`、
`workspace/sidebar/favorites/…/favorite-item-icon.tsx`、
`hooks/{use-favorite-item-details,use-workspace-paths,use-parse-editor-content,use-editor-flagging,use-additional-editor-mention}.ts(x)`、
`services/file.service.ts` 的 `PAGE_DESCRIPTION` 资源类型（交接 M5）、
`onboarding/tour/{root,sidebar}.tsx` 的文档页一步（**导览本身保留，"视图"成为最后一步**）、
`home/{home-dashboard-widgets.tsx,widgets/recents/index.tsx,widgets/empty-states/recents.tsx}` 的
`page` 实体分支、`types/src/instance/base.ts` 的 `instance_changelog_url`、
`global/index.ts`、`modals/project-level.tsx`、`project/archive-restore-modal.tsx`、
`settings/project/sidebar/item-icon.tsx`。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 2: 收敛编辑器包（去掉协作）**

Run:

```
rm -r web/packages/editor/src/components/editors/document \
      web/packages/editor/src/contexts \
      web/packages/editor/src/extensions/work-item-embed
rm web/packages/editor/src/lib.ts \
   web/packages/editor/src/components/document-editor-side-effects.ts \
   web/packages/editor/src/constants/document-collaborative-events.ts \
   web/packages/editor/src/extensions/core-without-props.ts \
   web/packages/editor/src/extensions/document-extensions.tsx \
   web/packages/editor/src/extensions/headings-list.ts \
   web/packages/editor/src/extensions/title-extension.ts \
   web/packages/editor/src/extensions/code/without-props.tsx \
   web/packages/editor/src/extensions/core/without-props.ts \
   web/packages/editor/src/helpers/get-document-server-event.ts \
   web/packages/editor/src/helpers/parser.ts \
   web/packages/editor/src/helpers/yjs-utils.ts \
   web/packages/editor/src/hooks/use-collaborative-editor.ts \
   web/packages/editor/src/hooks/use-editor-navigation.ts \
   web/packages/editor/src/hooks/use-title-editor.ts \
   web/packages/editor/src/hooks/use-yjs-setup.ts \
   web/packages/editor/src/styles/title-editor.css \
   web/packages/editor/src/types/collaboration.ts \
   web/packages/editor/src/types/document-collaborative-events.ts \
   web/packages/editor/src/types/embed.ts \
   web/packages/editor/src/types/issue-embed.ts \
   web/packages/types/src/page/core.ts \
   web/packages/types/src/page/extended.ts \
   web/packages/types/src/page/index.ts \
   web/packages/constants/src/page.ts \
   web/packages/utils/src/page.ts \
   web/packages/services/src/live.service.ts \
   web/packages/propel/src/icons/project/page-icon.tsx \
   web/packages/propel/src/icons/sub-brand/wiki-icon.tsx \
   web/packages/propel/src/empty-state/assets/vertical-stack/changelog.tsx \
   web/packages/propel/src/empty-state/assets/vertical-stack/page.tsx
rmdir web/packages/types/src/page
```

Run: `pnpm exec turbo run check:types`

按报错改：`editor/src/index.ts`、`editor/tsdown.config.ts`、
`editor/src/components/editors/{index.ts,editor-container.tsx}`、
`editor/src/constants/extension.ts`、`editor/src/extensions/{index.ts,extensions.ts}`、
`editor/src/extensions/code/lowlight-plugin.ts`、`editor/src/extensions/unique-id/{extension.ts,plugin.ts}`、
`editor/src/helpers/editor-ref.ts`、`editor/src/hooks/use-editor.ts`、
`editor/src/plugins/drag-handle.ts`、`editor/src/styles/{index.css,drag-drop.css}`、
`editor/src/types/{editor.ts,editor-extended.ts,extensions.ts,hook.ts,index.ts}`；
各包的 `index.ts`、`utils/src/{index.ts,file.ts,editor/common.ts}`、
`propel/src/icons/registry.ts`、`propel/src/icons/{project,sub-brand}/index.ts`、
`propel/src/empty-state/assets/{asset-registry.tsx,asset-types.ts,vertical-stack/index.ts}`。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 3: 删依赖与环境变量**

Run:

```
node /tmp/nerve-p2/pkg.mjs web/packages/editor/package.json del-dep \
  @hocuspocus/provider yjs y-indexeddb y-prosemirror y-protocols \
  @tiptap/extension-collaboration @tiptap/extension-character-count @tiptap/html buffer
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json del-dep @react-pdf/renderer react-pdf-html
```

把 `pnpm-workspace.yaml` 的 catalog 里这 11 条删掉；
`turbo.json` 的 `globalEnv` 去掉 `VITE_LIVE_BASE_PATH`、`VITE_LIVE_BASE_URL`；
`web/apps/web/.env.example` 末尾去掉两行 `VITE_LIVE_*`。

Run: `pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs`
Expected:

```
catalogs: 33 lines removed, 0 added
importers: 11 dependencies removed, 0 added or changed
packages: 72 removed, 0 added or changed
snapshots: 72 removed, 0 added or changed
```

- [ ] **Step 4: 文案**

Run:

```
rm web/packages/i18n/src/locales/en/page.json web/packages/i18n/src/locales/zh-CN/page.json
```

`web/packages/i18n/src/constants/namespaces.ts` 去掉 `"page",`。

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t7.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-t5.txt /tmp/nerve-p2/keys-t7.txt`

按输出删除 `common.json`（更新日志的 `full_changelog`、`whats_new` 等）、`empty-state.json`
（`workspace_empty_state.wiki`、`project_page.*`）、`home.json`、`navigation.json`、`power-k.json`、
`project.json`、`project-settings.json`、`work-item.json`（`issue.pages`）里的文档页文案。

- [ ] **Step 5: 守卫规则与例外**

在 `rules` 末尾追加两条。`pages-collaboration` 按 M1 设计 7.4 的确切清单：

```json
{
  "id": "pages-collaboration",
  "phase": "M1/P2",
  "why": "文档页和协作编辑（Yjs、Hocuspocus、live 服务、PDF 导出、工作项嵌入）；按 M1 设计 7.4 的确切符号搜，不搜单独的 page / pages（路由文件本身就叫 page.tsx）",
  "files": {
    "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json|css|ya?ml)|web/apps/web/\\.env\\.example|turbo\\.json|pnpm-workspace\\.yaml)$",
    "flags": ""
  },
  "content": {
    "source": "page_view|description_binary|\\byjs\\b|y-prosemirror|y-indexeddb|y-protocols|hocuspocus|Collaborative|extension-collaboration|\\bpageId\\b|\\bpage_id\\b|WorkItemEmbed|issue-embed|LIVE_BASE|VITE_LIVE|comlink|react-pdf|@tiptap/html|wiki",
    "flags": "i"
  },
  "samples": {
    "hit": [
      "  page_view: boolean;",
      "import type { HocuspocusProvider } from \"@hocuspocus/provider\";",
      "  \"yjs\": \"catalog:\"",
      "export const LIVE_BASE_URL = process.env.VITE_LIVE_BASE_URL || \"\";",
      "      const pageInstance = pageId ? getPageById(pageId.toString()) : null;",
      "  CollaborativeDocumentEditorWithRef,",
      "        : `/${page?.workspace__slug}/wiki/${page?.id}`;"
    ],
    "miss": [
      "import { RecentProject } from \"./project\";",
      "import { find } from \"linkifyjs\";",
      "export default function ProjectIssuesPage() {",
      "  const pathname = usePathname();"
    ],
    "files": {
      "hit": ["web/packages/editor/package.json", "web/packages/types/src/project/projects.ts"],
      "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
    }
  }
}
```

`changelog`：`content.source` 为 `changelog|product-updates|ProductUpdates|what.s new`，`flags` 为 `i`。

例外：

```json
[
  { "rule": "pages-collaboration", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "Wiki", "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "pages-collaboration", "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx", "match": "wiki", "reason": "计费对比表的文案，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "pages-collaboration", "path": "web/packages/constants/src/subscription.ts", "match": "Wiki", "reason": "订阅套餐的功能名，随计费在 P3 删除", "until": "M1/P3" },
  { "rule": "pages-collaboration", "path": "web/packages/utils/src/tlds.ts", "match": "wiki", "reason": "顶级域名数据里的 .wiki，不是功能代码", "until": "M9" }
]
```

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 21 rules, 14 exceptions, no hits.`

- [ ] **Step 6: 降上限、门禁与提交**

Run:

```
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 685"
node /tmp/nerve-p2/pkg.mjs web/packages/utils/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 28"
node /tmp/nerve-p2/pkg.mjs web/packages/editor/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 68"
```

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 21 rules, 14 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → 14

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 450 files, 7609235 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `313 files changed, 140 insertions(+), 14930 deletions(-)`

提交信息：

```
feat(web): delete pages, the collaborative editor and the changelog

Pages go whole: the routes, the components, the stores, the services,
the hooks, the types, the copy and the images. So does everything that
only pages used -- the PDF export, the work-item embed, the page
branches of search, favourites, recents and power-k, and the page step
of the tour, which now ends on views.

The editor core loses collaboration with them: Yjs, Hocuspocus, the
provider argument, the document info and heading callbacks, and eleven
dependencies. Work-item descriptions, comments, intake and drafts all
use the local editor and never used any of it.

The changelog goes too: its type borrowed the page type.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 8: 编辑器内核只留本地能力

**Files**

- Delete：`web/packages/editor/src/plugins/ai-handle.ts`、
  `web/packages/editor/src/components/menus/ai-menu.tsx`、`web/packages/editor/src/types/ai.ts`
- Modify（6）：`web/packages/editor/package.json`、
  `web/packages/editor/src/components/editors/rich-text/editor.tsx`、
  `web/packages/editor/src/components/menus/index.ts`、
  `web/packages/editor/src/extensions/side-menu.ts`、
  `web/packages/editor/src/styles/variables.css`、`web/packages/editor/src/types/index.ts`

**Interfaces**

- Produces：`SideMenuExtension` 不再有 `aiEnabled`；`TAIHandler` 相关类型全部消失。
- 保留（M1 设计 3.1，Task 12 的测试会断言）：`UniqueID`、`copyMarkdownToClipboard`、
  用户 @提及、图片与附件节点。

**Steps**

- [ ] **Step 1: 删除**

Run:

```
rm web/packages/editor/src/plugins/ai-handle.ts \
   web/packages/editor/src/components/menus/ai-menu.tsx \
   web/packages/editor/src/types/ai.ts
```

`side-menu.ts` 去掉 `aiEnabled` 选项和它挂的处理器；`components/menus/index.ts`、
`types/index.ts` 去掉导出；`rich-text/editor.tsx` 去掉传进来的 AI 参数。

`web/packages/editor/src/styles/variables.css` 去掉只给文档页用的 `/* layout config */` 整块，
以及 `--normal-content-margin`、`--wide-content-margin` 两个变量。

Run: `pnpm exec turbo run check:types 2>&1 | grep -E "error TS|Tasks:"`
Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 2: 降上限、门禁与提交**

Run: `node /tmp/nerve-p2/pkg.mjs web/packages/editor/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 67"`

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 21 rules, 14 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → 14；`make build-web 2>&1 | grep Tasks:` → 11

Run: `git add -A && git diff --cached --shortstat`
Expected: `9 files changed, 2 insertions(+), 377 deletions(-)`

提交信息：

```
feat(editor): the editor core keeps only the local capabilities

With the collaborative document editor gone, the AI handle of the side
menu, the AI menu component and the TAIHandler prop have no caller: they
only ever hung off that editor (M1 design 3.1). The page-only layout
block of variables.css goes with them.

UniqueID stays -- every editor installs it and the non-collaborative
editors use the node ids to locate a node -- as do
copyMarkdownToClipboard, the user mentions and the image and attachment
nodes.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 9: 便签；工具栏收敛为一组

**Files**

- Delete（37）：
  - 路由 3：`app/(all)/[workspaceSlug]/(projects)/stickies/{header,layout,page}.tsx`
  - `core/components/stickies/`（17）、`core/components/editor/sticky-editor/`（4）、
    `core/components/home/widgets/empty-states/stickies.tsx`
  - `core/hooks/use-stickies.tsx`、`core/services/sticky.service.ts`、`core/store/sticky/sticky.store.ts`
  - `packages/constants/src/stickies.ts`、`packages/types/src/stickies.ts`
  - `packages/i18n/src/locales/{en,zh-CN}/stickies.json`
  - propel 4：`src/icons/{multiple-sticky.tsx,sticky-note-icon.tsx}`、
    `src/icons/workspace/multiple-sticky-icon.tsx`、
    `src/empty-state/assets/horizontal-stack/note.tsx`
  - 图片 4：`app/assets/empty-state/stickies/`
- Modify（33）：`tools/keywords.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`、
  `web/apps/web/package.json`、`app/routes/core.ts`、
  `core/components/editor/lite-text/toolbar.tsx`、`core/components/home/home-dashboard-widgets.tsx`、
  `core/components/home/widgets/empty-states/index.ts`、
  `core/components/workspace/sidebar/helper.tsx`、
  `core/store/{base-command-palette.store.ts,root.store.ts}`、`web/apps/web/styles/globals.css`、
  `packages/constants/src/{index,workspace,navigation.test}.ts`、
  `packages/editor/src/constants/common.ts`、`packages/i18n/src/constants/namespaces.ts`、
  i18n 中英文 4 个文件、propel 注册表 6 个文件、`packages/types/src/{home,index}.ts`、
  `packages/utils/src/theme/constants.ts`

**Interfaces**

- Consumes：Task 7 之后的编辑器包（文档页编辑器已删）。
- Produces：`TOOLBAR_ITEMS` 从"按编辑器类型分组"塌成一组；守卫规则 `stickies`。
- 顺序约束（M1 设计 9 节，硬）：便签必须在首页（Task 10）之前。

**Steps**

- [ ] **Step 1: 删除**

Run:

```
rm -r "web/apps/web/app/(all)/[workspaceSlug]/(projects)/stickies" \
      web/apps/web/core/components/stickies \
      web/apps/web/core/components/editor/sticky-editor \
      web/apps/web/core/store/sticky \
      web/apps/web/app/assets/empty-state/stickies
rm web/apps/web/core/components/home/widgets/empty-states/stickies.tsx \
   web/apps/web/core/hooks/use-stickies.tsx \
   web/apps/web/core/services/sticky.service.ts \
   web/packages/constants/src/stickies.ts \
   web/packages/types/src/stickies.ts \
   web/packages/i18n/src/locales/en/stickies.json \
   web/packages/i18n/src/locales/zh-CN/stickies.json \
   web/packages/propel/src/icons/multiple-sticky.tsx \
   web/packages/propel/src/icons/sticky-note-icon.tsx \
   web/packages/propel/src/icons/workspace/multiple-sticky-icon.tsx \
   web/packages/propel/src/empty-state/assets/horizontal-stack/note.tsx
```

**同一步**删掉 `web/apps/web/app/routes/core.ts` 里 `:workspaceSlug/stickies` 的条目，
`packages/i18n/src/constants/namespaces.ts` 里的 `"stickies",`，以及
`packages/constants/src/workspace.ts` 固定侧边栏里的 `stickies` 一项和
`packages/constants/src/navigation.test.ts` 里对应的 `"stickies",` 一行。

- [ ] **Step 2: 工具栏收敛**

`web/packages/editor/src/constants/common.ts`：

- 删 `export type TEditorTypes = "lite" | "document" | "sticky";`；
- `ToolbarMenuItem` 去掉 `editors: TEditorTypes[];` 字段，各条目里的 `editors: [...]` 一并删掉；
- 删 `TYPOGRAPHY_ITEMS` 整块（只有文档页显示）；
- 删 `COMPLEX_ITEMS` 里的 `{ itemKey: "table", … }` 一项；
- `TOOLBAR_ITEMS` 改为：

```ts
export const TOOLBAR_ITEMS: {
  [key: string]: ToolbarMenuItem[];
} = {
  basic: BASIC_MARK_ITEMS,
  alignment: TEXT_ALIGNMENT_ITEMS,
  list: LIST_ITEMS,
  userAction: USER_ACTION_ITEMS,
  complex: COMPLEX_ITEMS,
};
```

`web/apps/web/core/components/editor/lite-text/toolbar.tsx` 改为 `const toolbarItems = TOOLBAR_ITEMS;`。

- [ ] **Step 3: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

要删的东西：`core/store/root.store.ts` 的便签 store 接线、
`core/store/base-command-palette.store.ts` 的便签弹窗、
`home-dashboard-widgets.tsx` 和 `widgets/empty-states/index.ts` 的便签组件、
`workspace/sidebar/helper.tsx` 的便签图标、`constants/src/index.ts`、`types/src/index.ts`、
`types/src/home.ts` 的 `stickies` 组件键、`utils/src/theme/constants.ts` 的便签配色、
propel 的 `icons/{index.ts,registry.ts}`、`icons/workspace/index.ts`、
`empty-state/assets/{asset-registry.tsx,asset-types.ts,horizontal-stack/index.ts}`、
`web/apps/web/styles/globals.css` 里便签专用的样式块。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 4: 删依赖**

Run: `node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json del-dep react-masonry-component`

`pnpm-workspace.yaml` 删掉 catalog 里的 `react-masonry-component` 和
`overrides` 里的 `react-masonry-component>react`（它已不作用于任何包）。

Run: `pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs`
Expected: 只有 `catalogs`、`overrides`、`importers`、`packages`、`snapshots` 的纯删除行。

- [ ] **Step 5: 文案与守卫**

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t9.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-t7.txt /tmp/nerve-p2/keys-t9.txt`

按输出删除 `empty-state.json`、`navigation.json` 里的便签文案。

在 `rules` 末尾追加 `stickies`（**区分大小写**，`flags` 为空）：

```json
{
  "id": "stickies",
  "phase": "M1/P2",
  "why": "便签（工作区便签页、首页便签组件、便签编辑器）；区分大小写，不搜 stick（不区分大小写时命中 AxisTick），也不搜单独的 sticky（Tailwind 的 position: sticky）",
  "files": { "source": "^web/.*\\.(?:[cm]?[jt]sx?|json|css|ya?ml)$", "flags": "" },
  "content": { "source": "Sticky|STICKY|[Ss]tickies|STICKIES|sticky[-_.:]", "flags": "" },
  "samples": {
    "hit": [
      "import { StickiesWidget } from \"../stickies/widget\";",
      "  stickyStore: IStickyStore;",
      "    key: \"stickies\",",
      "  \"stickies\": \"Stickies\",",
      "      allStickiesModal: observable,"
    ],
    "miss": [
      "        <div className=\"sticky top-0 z-10\">",
      "  position: sticky;",
      "const CustomXAxisTick = (props: TAxisTickProps) => {",
      "            {/* Single sticky column */}"
    ],
    "files": {
      "hit": ["web/apps/web/core/store/root.store.ts", "web/packages/i18n/src/locales/en/navigation.json"],
      "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
    }
  }
}
```

miss 样本里**不要**写 `MultipleStickyOutline` 这类保留的标识符：规则区分大小写，`Sticky` 会命中它。

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 22 rules, 14 exceptions, no hits.`

- [ ] **Step 6: 降上限、门禁与提交**

Run: `node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 673"`

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 22 rules, 14 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → 14

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 439 files, 7503359 bytes`、`other: 124 files, 8112616 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `70 files changed, 51 insertions(+), 2855 deletions(-)`

提交信息：

```
feat(web): delete the stickies

The stickies page, the home widget, the sticky editor, the store and the
service go, and react-masonry-component -- which only the sticky list
used -- leaves the workspace.

With the page editor and the sticky editor both gone the toolbar has one
editor type left, so TEditorTypes, the per-item editors flag and the two
items that only the page editor showed (the typography picker and the
table) go with them.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 10: 首页固定内容、自定义主题、旧仪表盘

**Files**

- Create：`web/apps/web/core/components/home/home-body.tsx`
- Rename：`web/packages/utils/src/theme-legacy.ts` → `web/packages/utils/src/theme.ts`
- Delete（59）：
  - 首页 15：`core/components/home/home-dashboard-widgets.tsx`、
    `core/components/home/widgets/links/`（7）、`core/components/home/widgets/manage/`（5）、
    `core/components/home/widgets/empty-states/links.tsx`、
    `core/components/home/widgets/loaders/quick-links.tsx`
  - store / hook 3：`core/store/workspace/{home.ts,link.store.ts}`、`core/hooks/store/use-home.ts`
  - 旧仪表盘 6：`core/store/dashboard.store.ts`、`core/services/dashboard.service.ts`、
    `core/hooks/store/use-dashboard.ts`、`helpers/dashboard.helper.ts`、
    `packages/types/src/dashboard.ts`、`packages/constants/src/dashboard.ts`
  - 自定义主题 13：`core/components/core/theme/`（5）、`packages/utils/src/theme/`（7）、
    `packages/ui/src/form-fields/input-color-picker.tsx`
  - 图片 21：`app/assets/empty-state/dashboard/`（20）、
    `app/assets/empty-state/dashboard_empty_project.webp`
- Modify（33）：见步骤

**Interfaces**

- Consumes：Task 7、Task 9（文档页和便签的首页组件已删）。
- Produces：`HomeBody`（首页正文的唯一来源）；`IUserTheme` 只剩 `{ theme }`（交接 M2）；
  守卫规则 `home-theme`。
- 顺序约束（M1 设计 9 节，硬）：文档页和便签必须在首页之前。

**Steps**

- [ ] **Step 1: 先写首页正文（保留"最近访问"的唯一入口）**

唯一渲染"最近访问"的地方是要删的小组件列表，所以先新建正文组件，再删小组件（M1 设计 3.14）。

写入 `web/apps/web/core/components/home/home-body.tsx`：

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "next/navigation";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { HomeLoader, NoProjectsEmptyState, RecentActivityWidget } from "./widgets";

/** The home body is a fixed list (M1 design 3.14): the no-projects empty state and the recent visits. */
export const HomeBody = observer(function HomeBody() {
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { loader } = useProject();

  if (!workspaceSlug) return null;
  if (loader !== "loaded") return <HomeLoader />;

  return (
    <div className="relative flex h-full w-full flex-col gap-7">
      <NoProjectsEmptyState />
      <div className="py-4">
        <RecentActivityWidget workspaceSlug={workspaceSlug.toString()} />
      </div>
    </div>
  );
});
```

`core/components/home/index.ts` 改为 `export * from "./home-body";`；
`core/components/home/root.tsx` 里 `<DashboardWidgets />` 换成 `<HomeBody />`，并去掉拉取组件配置的
`useEffect` 和 `useHome`。问候语、导览和工作项 peek 仍由 `root.tsx` 挂载，不要动。

- [ ] **Step 2: 删除**

Run:

```
rm -r web/apps/web/core/components/home/widgets/links \
      web/apps/web/core/components/home/widgets/manage \
      web/apps/web/core/components/core/theme \
      web/packages/utils/src/theme \
      web/apps/web/app/assets/empty-state/dashboard
rm web/apps/web/core/components/home/home-dashboard-widgets.tsx \
   web/apps/web/core/components/home/widgets/empty-states/links.tsx \
   web/apps/web/core/components/home/widgets/loaders/quick-links.tsx \
   web/apps/web/core/store/workspace/home.ts \
   web/apps/web/core/store/workspace/link.store.ts \
   web/apps/web/core/hooks/store/use-home.ts \
   web/apps/web/core/store/dashboard.store.ts \
   web/apps/web/core/services/dashboard.service.ts \
   web/apps/web/core/hooks/store/use-dashboard.ts \
   web/apps/web/helpers/dashboard.helper.ts \
   web/apps/web/app/assets/empty-state/dashboard_empty_project.webp \
   web/packages/types/src/dashboard.ts \
   web/packages/constants/src/dashboard.ts \
   web/packages/ui/src/form-fields/input-color-picker.tsx
git mv web/packages/utils/src/theme-legacy.ts web/packages/utils/src/theme.ts
```

`web/packages/utils/src/theme.ts` 里只剩 `resolveGeneralTheme`，文件头的注释全部指向已删除的文件，
整块换成一行：

```ts
/** Resolves the theme a name stands for: every theme of THEMES is either light or dark, anything else follows the system. */
```

`web/packages/utils/src/index.ts` 里 `export { resolveGeneralTheme } from "./theme-legacy";`
改成 `from "./theme"`，并去掉 `export * from "./theme";`（指向已删目录的那一行）。

- [ ] **Step 3: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

要删的东西：

- `core/store/root.store.ts`：`dashboard` 的 import、字段、两处 `new DashboardStore(this)`。
- `core/store/workspace/index.ts`：`home` 的 import、字段、构造。
- `core/components/appearance/theme-switcher.tsx`：`applyCustomTheme` 的 import、
  `themeOption.value === "custom"` 的整个分支、`CustomThemeSelector` 的 import 与渲染；
  `useCallback` 的依赖数组去掉 `userProfile`；返回值从 `<>…</>` 收成单个 `SettingsControlItem`。
- `core/lib/wrappers/store-wrapper.tsx`：`applyCustomTheme`、`clearCustomTheme` 的 import、
  `previousThemeRef`、整个 "Effect 2: Custom theme CSS application" 副作用；
  "Effect 1" 的注释改成 "Initial theme sync from server (one-time only)"。
- `web/apps/web/app/root.tsx`：`<ThemeProvider themes={[...]}>` 去掉 `"custom"`。
- `packages/constants/src/themes.ts`：`THEMES` 去掉 `"custom"`，`THEME_OPTIONS` 去掉整条 `custom` 记录。
- `packages/types/src/users.ts`：`IUserTheme` 只剩
  `theme: string | undefined; // one of THEMES, or 'system'`；文件末尾注释掉的
  `ICurrentUser`、`ICustomTheme`、`ICurrentUserSettings` 三段死注释一并删掉。
- `core/services/workspace.service.ts`：`fetchWorkspaceLinks`、`createWorkspaceLink`、
  `updateWorkspaceLink`、`deleteWorkspaceLink`、`fetchWorkspaceWidgets`、`updateWorkspaceWidget`
  六个方法和它们的 `TLink`、`TWidgetEntityData` import。
- `packages/types/src/home.ts`：`THomeWidgetKeys`、`THomeWidgetProps`、`TLinkEditableFields`、
  `TLink`、`TLinkMap`、`TLinkIdMap`、`TWidgetEntityData`；
  `core/components/home/widgets/recents/index.tsx` 的 `TRecentWidgetProps` 改为自带
  `workspaceSlug: string;`。
- `packages/types/src/enums.ts`：`EDurationFilters`（只有旧仪表盘在用）。
- `packages/types/src/index.ts`、`packages/constants/src/index.ts`、
  `packages/ui/src/form-fields/index.ts`：同步导出。
- `app/(all)/[workspaceSlug]/(projects)/header.tsx`：首页头部的"管理小组件"按钮，
  连同 `useHome`、`WidgetOutline`、`Button` 的 import。
- `core/components/home/widgets/loaders/loader.tsx`：`QuickLinksWidgetLoader` 和
  `EWidgetKeys.QUICK_LINKS`。
- `core/components/home/widgets/empty-states/index.ts`：快捷链接空状态的导出。

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 4: 删依赖**

`@plane/ui` 不再用取色器：

Run:

```
node /tmp/nerve-p2/pkg.mjs web/packages/ui/package.json del-dep react-color
node /tmp/nerve-p2/pkg.mjs web/packages/ui/package.json del-dep @types/react-color
pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs
```

Expected: 只有 `importers: 2 dependencies removed, 0 added or changed`。
`pnpm-workspace.yaml` 的 catalog **不要动**：`web` 仍在标签、状态和工作项标签的取色器里用 `react-color`。

- [ ] **Step 5: 文案**

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t10.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-t9.txt /tmp/nerve-p2/keys-t10.txt`

Run:

```
node /tmp/nerve-p2/i18n-del.mjs common custom customize_your_theme background_color text_color primary_color sidebar_background_color sidebar_text_color set_theme enter_a_valid_hex_code_of_6_characters background_color_is_required text_color_is_required primary_color_is_required sidebar_background_color_is_required sidebar_text_color_is_required updating_theme theme_updated_successfully failed_to_update_the_theme common.saving view_link_copied_to_clipboard link.modal
node /tmp/nerve-p2/i18n-del.mjs home home.empty.widgets home.quick_links home.new_at_plane home.quick_tutorial home.widget home.manage_widgets
node /tmp/nerve-p2/i18n-del.mjs empty-state workspace_empty_state.home_widget_quick_links
node /tmp/nerve-p2/i18n-del.mjs settings themes.theme_options.custom
```

Expected:

```
web/packages/i18n/src/locales/en/common.json: 20 keys removed
web/packages/i18n/src/locales/zh-CN/common.json: 20 keys removed
web/packages/i18n/src/locales/en/home.json: 6 keys removed
web/packages/i18n/src/locales/zh-CN/home.json: 6 keys removed
web/packages/i18n/src/locales/en/empty-state.json: 1 keys removed
web/packages/i18n/src/locales/zh-CN/empty-state.json: 1 keys removed
web/packages/i18n/src/locales/en/settings.json: 1 keys removed
web/packages/i18n/src/locales/zh-CN/settings.json: 1 keys removed
```

`common.custom`（"Custom theme"）是主题选择框用 `t(themeOption.key)` 动态取的，
`keyuse.mjs` 看不出来，所以在上面的清单里显式列出。

- [ ] **Step 6: 守卫规则**

在 `rules` 末尾追加（M1 设计 7.4 的清单；改为不区分大小写，好覆盖 `EWidgetKeys.QUICK_LINKS`）：

```json
{
  "id": "home-theme",
  "phase": "M1/P2",
  "why": "首页快捷链接、首页个性化（组件开关）、自定义主题，以及已经没有入口的旧首页仪表盘；不搜单独的 widget、dashboard、custom（首页最近访问组件与四个固定主题保留）",
  "files": { "source": "^web/.*\\.(?:[cm]?[jt]sx?|json|css)$", "flags": "" },
  "content": {
    "source": "quick_links|quick-links|QuickLink|HomeWidget|manage_widgets|home-preferences|CustomTheme|customize_your_theme|palette-generator|DashboardStore|useDashboard",
    "flags": "i"
  },
  "samples": {
    "hit": [
      "import { QuickLinksWidget } from \"./links\";",
      "    return this.get(`/api/workspaces/${workspaceSlug}/quick-links/`)",
      "      key: EWidgetKeys.QUICK_LINKS,",
      "  \"manage_widgets\": \"Manage widgets\",",
      "import { DashboardStore } from \"./dashboard.store\";",
      "import { useDashboard } from \"@/hooks/store/use-dashboard\";",
      "export const CustomThemeSelector = observer(() => {"
    ],
    "miss": [
      "import { RecentActivityWidget } from \"./recents\";",
      "  \"custom\": \"Custom\",",
      "        <div className=\"h-full w-full\">",
      "import { useProject } from \"@/hooks/store/use-project\";"
    ],
    "files": {
      "hit": ["web/apps/web/core/store/root.store.ts", "web/packages/i18n/src/locales/en/home.json"],
      "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
    }
  }
}
```

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 23 rules, 14 exceptions, no hits.`

- [ ] **Step 7: 降上限、门禁与提交**

Run:

```
node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 668"
node /tmp/nerve-p2/pkg.mjs web/packages/utils/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 27"
```

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 23 rules, 14 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → 14

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 443 files, 7421846 bytes`、
`css: 3 files, 307888 bytes`、`other: 122 files, 8043211 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `94 files changed, 102 insertions(+), 4235 deletions(-)`

提交信息：

```
feat(web): the home page becomes a fixed list

The home body is no longer assembled from widgets a person turns on and
reorders: it is the no-projects empty state and the recent visits, in
that order (M1 design 3.14). The quick links, the widget manager and the
home preferences of the workspace service go with it.

The old dashboard behind them goes too. It had lost its entry point long
ago and calls endpoints Plane itself no longer serves, so its store,
service, hook, helper, types and constants leave the workspace.

The custom theme goes as well: light, dark, system and the two
high-contrast themes are unrelated to it and stay. With the OKLCH theme
application gone theme-legacy.ts holds nothing legacy any more, so it
becomes theme.ts.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 11: 导出与从未创建过的集成服务

两者同时删（M1 设计 9 节，硬）：导出记录列表借用了集成服务。

**Files**

- Delete（28）：
  - `app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/exports/{header,page}.tsx`
  - `app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/integrations/page.tsx`
    （**这个页面本来就没有路由条目，从未可达**）
  - `core/components/exporter/`（6）、`core/components/integration/`（3）、
    `core/components/project/integration-card.tsx`（已无调用方）
  - `core/services/integrations/`（4）、`core/services/project/project-export.service.ts`、
    `core/services/app_installation.service.ts`、`core/hooks/use-integration-popup.tsx`
  - `core/components/ui/loader/settings/{import-and-export,integration}.tsx`
  - `packages/types/src/importer/`（3）、`packages/types/src/integration.ts`
  - `packages/i18n/src/locales/{en,zh-CN}/integration.json`（291 个键全部没有代码引用）
  - 图片 3：`app/assets/logos/github-square.png`、`app/assets/services/{github,slack}.png`
- Modify（20）：`tools/keywords.json`、`web/apps/web/package.json`、`app/routes/core.ts`、
  `core/components/settings/workspace/sidebar/item-icon.tsx`、`core/services/project/index.ts`、
  `packages/constants/src/{fetch-keys,workspace}.ts`、`packages/constants/src/settings/workspace.ts`、
  `packages/i18n/src/constants/namespaces.ts`、`packages/types/src/{index,settings}.ts`、
  i18n 中英文 6 个文件

**Steps**

- [ ] **Step 1: 删除**

Run:

```
rm -r web/apps/web/core/components/exporter \
      web/apps/web/core/components/integration \
      web/apps/web/core/services/integrations \
      "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/exports" \
      "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/integrations" \
      web/packages/types/src/importer
rm web/apps/web/core/components/project/integration-card.tsx \
   web/apps/web/core/components/ui/loader/settings/import-and-export.tsx \
   web/apps/web/core/components/ui/loader/settings/integration.tsx \
   web/apps/web/core/hooks/use-integration-popup.tsx \
   web/apps/web/core/services/app_installation.service.ts \
   web/apps/web/core/services/project/project-export.service.ts \
   web/apps/web/app/assets/logos/github-square.png \
   web/apps/web/app/assets/services/github.png \
   web/apps/web/app/assets/services/slack.png \
   web/packages/types/src/integration.ts \
   web/packages/i18n/src/locales/en/integration.json \
   web/packages/i18n/src/locales/zh-CN/integration.json
```

**同一步**删掉 `web/apps/web/app/routes/core.ts` 里 `:workspaceSlug/settings/exports` 的条目，
以及 `packages/i18n/src/constants/namespaces.ts` 里的 `"integration",`。

- [ ] **Step 2: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`

要删的东西：

- `core/services/project/index.ts`：`export * from "./project-export.service";`
- `packages/types/src/index.ts`：`export * from "./importer";` 和 `export * from "./integration";`
- `packages/types/src/settings.ts`：`TWorkspaceSettingsTabs` 去掉 `"export"`
- `packages/constants/src/settings/workspace.ts`：`WORKSPACE_SETTINGS` 的 `export` 整条记录，
  以及 `GROUPED_WORKSPACE_SETTINGS` 里的 `WORKSPACE_SETTINGS["export"]`
- `core/components/settings/workspace/sidebar/item-icon.tsx`：`ExportOutline` 的 import 和
  `export: ExportOutline,`
- `packages/constants/src/workspace.ts`：`IMPORTERS_LIST`、`EXPORTERS_LIST` 两个数组；
  `RESTRICTED_URLS` 去掉 `"import"`、`"importers"`、`"integrations"`、`"integration"`
- `packages/constants/src/fetch-keys.ts`：`JIRA_IMPORTER_DETAIL`、`IMPORTER_SERVICES_LIST`、
  `EXPORT_SERVICES_LIST`、`GITHUB_REPOSITORY_INFO`、`SLACK_CHANNEL_INFO` 五个键，
  以及 import 里的 `IJiraMetadata`

Expected: `Tasks:   22 successful, 22 total`

- [ ] **Step 3: 文案与守卫**

Run: `node /tmp/nerve-p2/keyuse.mjs > /tmp/nerve-p2/keys-t11.txt && bash /tmp/nerve-p2/keydiff.sh /tmp/nerve-p2/keys-t10.txt /tmp/nerve-p2/keys-t11.txt`

Run:

```
node /tmp/nerve-p2/i18n-del.mjs common exporter refresh_status
node /tmp/nerve-p2/i18n-del.mjs empty-state settings_empty_state.exports
node /tmp/nerve-p2/i18n-del.mjs workspace-settings workspace_settings.settings.exports
```

在 `rules` 末尾追加：

```json
{
  "id": "export",
  "phase": "M1/P2",
  "why": "工作区的导出设置页及其导出记录列表；不搜单独的 export（保留代码里到处都是 export 语句）",
  "files": { "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$", "flags": "" },
  "content": { "source": "exporter|settings/exports|export-issues|EXPORTERS_LIST|IExportData", "flags": "i" },
  "samples": {
    "hit": [
      "import { ExportGuide } from \"@/components/exporter/guide\";",
      "          router.push(`/${workspaceSlug}/settings/exports`);",
      "    return this.post(`/api/workspaces/${workspaceSlug}/export-issues/`, data)",
      "      provider: EXPORTERS_LIST[0],",
      "type RowData = IExportData;",
      "  \"exporter\": {"
    ],
    "miss": [
      "export const RESTRICTED_URLS: string[] = [",
      "export * from \"./project.service\";",
      "import { exportAsCSV } from \"./csv\";",
      "  \"webhooks\": \"Webhooks\","
    ],
    "files": {
      "hit": ["web/apps/web/app/routes/core.ts", "web/packages/constants/src/workspace.ts"],
      "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
    }
  }
}
```

集成与导入器的 `integration` / `importer` / `jira` / `slack` 规则按 M1 设计 7.4 归 P3，这里不加。

Run: `pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `keywords: 24 rules, 14 exceptions, no hits.`

- [ ] **Step 4: 降上限、门禁与提交**

Run: `node /tmp/nerve-p2/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 661"`

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 24 rules, 14 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → 14

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected: `Tasks:   11 successful, 11 total`；`js: 439 files, 7369397 bytes`、
`css: 3 files, 307728 bytes`、`other: 122 files, 8043211 bytes`

Run: `git add -A && git diff --cached --shortstat`
Expected: `48 files changed, 39 insertions(+), 2979 deletions(-)`

提交信息：

```
feat(web): delete the export settings page and the integration services

The workspace export settings page goes, and the integration services go
with it: the list of previous exports read them, and nothing else in the
workspace did (M1 design 2.1). Those services were never reachable --
the integrations settings page had no route, the importer types had no
component and the GitHub, Jira and Slack pickers had no caller -- so the
whole integration i18n namespace was dead and goes too.

The reserved workspace URLs lose import, importers, integrations and
integration, which no longer name a path of this app.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

### Task 12: 收尾——遗留依赖、编辑器测试、文档

**Files**

- Create：`web/packages/editor/vitest.config.ts`、`web/packages/editor/src/extensions/extensions.test.ts`
- Modify（5）：`pnpm-workspace.yaml`、`pnpm-lock.yaml`、`web/packages/editor/package.json`、
  `web/packages/utils/package.json`、`docs/v0/frontend-changes.md`

**Interfaces**

- Consumes：Task 7、8、9 之后的编辑器包。
- Produces：`make test-web` 15 个任务；knip 的"未使用依赖"为零；前端改动清单同步。

**Steps**

- [ ] **Step 1: 确认 knip 报的遗留依赖**

Run: `make knip 2>&1 | sed -n '/^Unused dependencies/,/^Unused exports/p'`
Expected:

```
Unused dependencies (7)
@plane/constants            web/packages/editor/package.json:…
@plane/ui                   web/packages/editor/package.json:…
@tiptap/extension-document  web/packages/editor/package.json:…
@tiptap/extension-heading   web/packages/editor/package.json:…
@tiptap/extension-text      web/packages/editor/package.json:…
tippy.js                    web/packages/editor/package.json:…
chroma-js                   web/packages/utils/package.json:…
Unused devDependencies (1)
@types/chroma-js  web/packages/utils/package.json:…
```

前五个是 Task 7 造成的，`tippy.js` 是 Task 8 造成的，`chroma-js` 两条是 Task 10 造成的。
集中在这里删，锁文件只动一次。

- [ ] **Step 2: 删依赖与 catalog 条目**

Run:

```
node /tmp/nerve-p2/pkg.mjs web/packages/editor/package.json del-dep \
  @plane/constants @plane/ui @tiptap/extension-document @tiptap/extension-heading @tiptap/extension-text tippy.js
node /tmp/nerve-p2/pkg.mjs web/packages/utils/package.json del-dep chroma-js @types/chroma-js
```

`pnpm-workspace.yaml` 的 catalog 删掉这六条：`@tiptap/extension-document`、
`@tiptap/extension-heading`、`@tiptap/extension-text`、`@types/chroma-js`、`chroma-js`、`tippy.js`
（`@plane/constants`、`@plane/ui` 是工作区包，不在 catalog 里）。

- [ ] **Step 3: 编辑器的仓库内测试**

写入 `web/packages/editor/vitest.config.ts`：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
});
```

写入 `web/packages/editor/src/extensions/extensions.test.ts`：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Extensions } from "@tiptap/core";
import { describe, expect, it } from "vitest";
// local imports
import { TOOLBAR_ITEMS } from "@/constants/common";
import type { TExtensions, TFileHandler, TMentionHandler } from "@/types";
import { CoreEditorExtensions } from "./extensions";

const fileHandler: TFileHandler = {
  assetsUploadStatus: {},
  cancel: () => {},
  checkIfAssetExists: async () => true,
  delete: async () => {},
  getAssetDownloadSrc: async (path) => path,
  getAssetSrc: async (path) => path,
  restore: async () => {},
  upload: async () => "asset-id",
  duplicate: async () => "asset-id",
  validation: { maxFileSize: 5 * 1024 * 1024 },
};

const mentionHandler: TMentionHandler = { renderComponent: () => null };

const buildExtensions = (
  overrides: { editable?: boolean; enableHistory?: boolean; disabledExtensions?: TExtensions[] } = {}
): Extensions =>
  CoreEditorExtensions({
    disabledExtensions: overrides.disabledExtensions ?? [],
    editable: overrides.editable ?? true,
    enableHistory: overrides.enableHistory ?? true,
    extendedEditorProps: {},
    fileHandler,
    flaggedExtensions: [],
    getEditorMetaData: () => ({}) as never,
    mentionHandler,
  });

const names = (overrides?: Parameters<typeof buildExtensions>[0]) =>
  buildExtensions(overrides).map((extension) => extension.name);

const starterKitOf = (overrides?: Parameters<typeof buildExtensions>[0]) =>
  buildExtensions(overrides).find((extension) => extension.name === "starterKit");

describe("the extensions every editor installs", () => {
  it("installs no collaboration extension", () => {
    expect(names()).not.toContain("collaboration");
    expect(names()).not.toContain("collaborationCursor");
  });

  it("keeps undo and redo local, in the history of the starter kit", () => {
    expect(starterKitOf()?.options.history).not.toBe(false);
  });

  it("switches the history off when the caller asks it to", () => {
    expect(starterKitOf({ editable: false, enableHistory: false })?.options.history).toBe(false);
  });

  it("keeps the nodes the description, the comments and the history view need", () => {
    for (const name of ["mention", "imageComponent", "table", "markdown", "uniqueID", "emoji"]) {
      expect(names()).toContain(name);
    }
  });

  it("leaves the image out when the caller disables it", () => {
    expect(names({ disabledExtensions: ["image"] })).not.toContain("imageComponent");
  });
});

describe("the toolbar", () => {
  it("has one set of items, since one editor type is left", () => {
    expect(Object.keys(TOOLBAR_ITEMS)).toEqual(["basic", "alignment", "list", "userAction", "complex"]);
  });

  it("offers no item that only the deleted page editor showed", () => {
    const itemKeys = Object.values(TOOLBAR_ITEMS).flatMap((items) => items.map((item) => item.itemKey));
    expect(itemKeys).not.toContain("table");
    expect(itemKeys).toContain("bold");
    expect(itemKeys).toContain("to-do-list");
  });
});
```

这些测试断言的是**扩展的组成**，不构造 TipTap 的 `Editor`，所以不需要 DOM 环境
（`environment` 用 vitest 默认的 node）。原因见 spec 第 3 节第 13 条。

Run:

```
node /tmp/nerve-p2/pkg.mjs web/packages/editor/package.json set-script test "vitest run"
node /tmp/nerve-p2/pkg.mjs web/packages/editor/package.json add-dev-dep vitest "catalog:"
pnpm install --silent && node /tmp/nerve-p2/lock-diff.mjs
```

Expected:

```
catalogs: 18 lines removed, 0 added
importers: 8 dependencies removed, 1 added or changed
  + web/packages/editor devDependencies vitest: specifier: 'catalog:'  version: 4.1.11(…)
packages: 2 removed, 0 added or changed
snapshots: 2 removed, 0 added or changed
```

（`packages` 只少 2 个：`tippy.js` 和 `chroma-js`；三个 tiptap 扩展仍是 `@tiptap/starter-kit`
的传递依赖，`@plane/constants`、`@plane/ui` 是工作区链接。）

Run: `pnpm --dir web/packages/editor exec vitest run 2>&1 | tail -4`
Expected: `Tests  7 passed (7)`

- [ ] **Step 4: 同步前端改动清单**

`docs/v0/frontend-changes.md` 第二节"删除的功能（M1）"的表格，按下面的对应关系改：

| 原来那一行（开头） | 改成 |
|---|---|
| `\| 文档页（Pages）及协作编辑模式（Yjs、Hocuspocus） \| 计划中 \| \|` | 行尾补上"；随它一起删除的更新日志（"what's new"）、新手导览中的文档页一步、编辑器内核中只为协作和文档页存在的部分（AI 处理器、AI 菜单、文档信息与标题回调、页面专用的排版变量）"，状态改 `已完成 \| M1/P2` |
| `\| 估算（Estimates） \| 计划中 \| \|` | 补"；迭代和模块的进度改为只按工作项数计算，`calculateCycleProgress` 和乐观更新有单元测试"，改 `已完成 \| M1/P2` |
| `\| 甘特图与时间线（包括模块的时间线视图） \| 计划中 \| \|` | 补"；保留的 `REVERSE_RELATIONS` 从 `constants/gantt-chart.ts` 移到 `constants/issue/relation.ts`"，改 `已完成 \| M1/P2` |
| `\| "自动化"设置页中的自动关闭（自动归档保留） \| 计划中 \| \|` | 改 `已完成 \| M1/P2` |
| `\| 数据分析 \| 计划中 \| \|` | 补"，连同 `:workspaceSlug/analytics` 旧地址重定向、侧边栏入口和 propel 的 `bar-chart`、`pie-chart`；保留的迭代、模块进度代码由 analytics 改名为 progress"，改 `已完成 \| M1/P2` |
| `\| 导出 \| 计划中 \| \|` | 补"，连同它借用的、从未创建过的集成服务（集成与导入器的 service、类型、文案命名空间和图片）"，改 `已完成 \| M1/P2` |
| `\| 便签 \| 计划中 \| \|` | 补"，连同 `react-masonry-component`"，改 `已完成 \| M1/P2` |
| 首页快捷链接和首页个性化那一行 | 行尾补"；已经没有入口的旧首页仪表盘"，改 `已完成 \| M1/P2` |
| 侧边栏的自定义导航那一行 | 行尾补"；保留的项目导航偏好改由 `ProjectNavigationDialog` 配置"，改 `已完成 \| M1/P2` |
| `\| 个人主页的统计和动态 \| 计划中 \| \|` | 补"；个人主页只剩用户卡片和工作项分页，`/profile/:userId` 重定向到"分配给他的""，改 `已完成 \| M1/P2` |
| 企业版残留那一行 | 拆成两行：新增 `\| 企业版残留中的"活跃迭代"推广页（工作区级；项目迭代列表中的"当前迭代"区块保留） \| 已完成 \| M1/P2 \|`，原行去掉`"活跃迭代"推广页、` |
| Plane 自身的死代码那一行 | 去掉`从未被创建过的集成服务、`，改成`…、调用不存在接口的 service 方法、knip 报告的未使用文件和导出`，状态仍是 `计划中` |

- [ ] **Step 5: 完整验收**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 24 rules, 14 exceptions, no hits.` + `Tasks:   52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:`
Expected: `Tasks:   15 successful, 15 total`

Run: `make build-web 2>&1 | grep Tasks: && node /tmp/nerve-p2/web-size.mjs`
Expected:

```
 Tasks:    11 successful, 11 total
js: 439 files, 7369397 bytes
css: 3 files, 307728 bytes
fonts: 25 files, 3755608 bytes
other: 122 files, 8043211 bytes
largest chunk: assets/use-editor-flagging-<hash>.js, 1382279 bytes
locale chunks: 34
```

Run: `make knip 2>&1 | grep -E "^(Unused|Unlisted|Unresolved|Configuration|Duplicate)"`
Expected:

```
Unused files (94)
Unused exports (111)
Unused exported types (61)
Unused exported enum members (4)
Duplicate exports (1)
Configuration hints (1)
```

（未使用依赖和开发依赖两行消失。剩下的 94 / 111 / 61 / 4 / 1 属于 P3 的死代码；
配置提示是 web 的 `+types/`，按 M1 设计 7.2 由 P3 处理。）

Run: `pnpm exec turbo run check:lint 2>&1 | grep -c "equal to the cap"`
Expected: `12`

Run: `git diff --stat 57fccb6 -- . | tail -1`
Expected: `862 files changed, 1673 insertions(+), 46198 deletions(-)`（加上本 Task 的文档改动会略多）

Run: `grep -n '^| P2 ' docs/v0/M1-frontend-trim/M1-design.md`
Expected（spec 和 plan 提交时控制者已经更新，这里不需要改动；review 链接在代码评审后补上）:

```
| P2 | trim-content | 进行中 | [spec](specs/P2-trim-content.md) | [plan](plans/P2-trim-content.md) | — |
```

Run: `git status --short; lsof -nP -iTCP:3000 -sTCP:LISTEN`
Expected: 只有 `?? .superpowers/`（本仓库未被忽略的工作目录）；端口没有输出。

- [ ] **Step 6: 提交**

Run: `git add -A && git diff --cached --shortstat`
Expected: `7 files changed, 121 insertions(+), 80 deletions(-)`（文档表格的改动会让行数多一些）

提交信息：

```
chore(web): close out the content trim

The seven dependencies the deleted features left behind -- tippy.js and
five tiptap and workspace packages in the editor, chroma-js and its
types in utils -- leave the workspace and the catalog.

The editor package gets its own tests. They assert what the two editor
tasks changed: no collaboration extension is installed, undo and redo
come from the starter kit's own history, the nodes the description, the
comments and the history view need are still there, and the toolbar has
one set of items.

The frontend change list records the eleven features this phase deleted.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

- [ ] **Step 7: 推送并确认持续集成（由控制者执行）**

推送分支，确认 `server`、`web`、`e2e` 三个任务都通过；`web` 任务的 `Lint` 一步先打印
`keywords: 24 rules, 14 exceptions, no hits.`，turbo 部分 52 个任务，
`Unit tests` 一步 15 个任务。

---

## 完成后

P2 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进
`docs/v0/M1-frontend-trim/reviews/P2-trim-content-review.md`：

- 裁定 spec 第 3 节的每一条差异，特别是第 15 条（`.superpowers/` 没有被 `.gitignore` 忽略）；
- **附录**（M1 设计 7.5）：三处进仓库测试的输出（`progress.test.ts` 13 个、
  `navigation.test.ts` 5 个、`extensions.test.ts` 7 个），以及三段临时核对脚本的全文、
  最小假数据、运行命令和断言：
  1. 首页、个人主页、侧边栏：最近访问的"有数据"和"空状态"；没有文档页的入口；
     `/profile/:userId` 重定向到"分配给他的"；成员、访客、成员不存在、成员接口失败四种状态；
     侧边栏为固定列表；
  2. 编辑器：工作项富文本的输入、保存、撤销 / 重做；评论的 @成员；描述历史的查看、还原、
     复制 Markdown；图片和节点定位；
  3. 动态、通知、工作项列表：用不含估算、文档页、Epic 的代表性数据核对显示和 store 的选择；
     列表、看板、表格、日历四个入口都在。
- 构建体积对比（spec 2.2 的表）；
- 按 spec 第 7 节交接：把 M2、M3、M4、M5、M6、M7 的事项分别写进对应 M 的 `handoffs/`；
- 同步 M1 设计第 12 节中 P2 的状态为"已完成"、补上 review 链接。
