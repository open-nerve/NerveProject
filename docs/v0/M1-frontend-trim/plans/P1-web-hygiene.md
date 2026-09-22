# M1/P1 工具链、遗留物与多语言 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** lint 警告数必须等于每个包的上限；关键词守卫和前端单元测试进入 `make lint-web`、`make test-web` 和持续集成；删除部署遗留、没有调用方的 turbo 任务和脚本、Storybook、未使用的依赖、`packages/services` 中没有调用方的文件、Next.js 垫片中的死文件，每次删除都加守卫规则、核对锁文件；消除两条构建警告、每个包写自己的增量编译文件；修复 React #418；多语言只留中英文，接上双向的键一致性检查；`make web-dev` 同时监视各包；记录构建体积基线；同步文档。

**Architecture:** 不删产品功能，只动工具链、遗留物和多语言。新增的代码是两个仓库级的检查工具（`tools/lint-cap.mjs`：运行一次 oxlint、比较警告数与各包 `check:lint` 中的上限；`tools/keywords.mjs` + `tools/keywords.json`：关键词守卫）、i18n 中的 `toSupportedLanguage` 和它的单元测试、改写的 `sync-check.ts`。所有删除都有核对：守卫规则先命中、删完为零；依赖和 `pnpm-workspace.yaml` 条目用锁文件比较脚本和条目核对脚本；文案用"整键字面量 + 模板前缀"的引用统计；运行时行为（#418、语言回退、开发监视）用一次性的 Playwright 脚本。每个 Task 一个提交，结束时 `make lint-web`、`make build-web`（Task 3 起还有 `make test-web`）通过，`make knip` 的报告与本计划一致。Plane 的旧地址重定向不在 P1（M1 设计 3.12）。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、i18next 25.10.9、Playwright（`e2e/` 中的版本）。

**Spec:** `docs/v0/M1-frontend-trim/specs/P1-web-hygiene.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

## Global Constraints

- **所有命令在仓库根目录（worktree 的根目录）执行**，不要 `cd`：要在包目录中执行时用 `pnpm --dir <目录>` 或 `pnpm --filter <包名>`。开始之前执行一次 `corepack enable` 和 `pnpm install --frozen-lockfile`；不做任何全局安装。改过某个 `package.json` 之后，第一条 `pnpm exec`、`pnpm --dir … exec` 会先自动核对并安装一次依赖，多打印 `Scope: all 16 workspace projects` 到 `Done in …` 几行（pnpm 11 的 `verifyDepsBeforeRun`）；本计划的预期输出省略这几行。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写。锁文件的行数和 SHA-256 是写作时的值（P1 只删除依赖或复用锁文件中已有的解析结果，不需要从网络解析新版本，应当完全一致）；对不上时以本节"锁文件核对"中 `lock-diff.mjs` 的输出为准，它也对不上就停下来。
- **一次性脚本**放在仓库外的 `/tmp/nerve-p1/`（不进仓库；有沙箱限制时换成自己的临时目录，命令中的路径随之替换）。本节给出四个脚本，Task 1 Step 1 写入；各 Task 另有自己的一次性脚本。每个 Task 开始时，如果脚本不存在（例如换了机器或会话），按本节重新写入。

  `/tmp/nerve-p1/pkg.mjs`：修改 `package.json`，按仓库的格式（两个空格缩进的 JSON，末尾换行）写回；要删的脚本或依赖不存在时报错。仓库中每个 `package.json` 都是这个格式，写回不会带来其他改动。

  ```js
  // One-off (M1/P1): edits a package.json and writes it back in the repository's format (2-space JSON
  // and a trailing newline). Fails when a script or dependency to delete is not there.
  // usage: node pkg.mjs <package.json> del-script <name>... | del-dep <name>... | set-script <name> <command>
  //        | add-dev-dep <name> <specifier>   (kept in alphabetical order, as pnpm writes it)
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
  } else {
    throw new Error(`unknown operation ${op}`);
  }
  fs.writeFileSync(file, `${JSON.stringify(pkg, null, 2)}\n`);
  ```

  `/tmp/nerve-p1/unused-entries.mjs`：列出 `pnpm-workspace.yaml` 中不作用于任何东西的条目，没有输出表示每一条都有作用（M1 设计 8 列出的六类，spec 2.6）。

  ```js
  // One-off (M1/P1): prints the pnpm-workspace.yaml entries that act on nothing; no output means every
  // entry is used. Run from the repository root.
  //   catalog <name>                    no package.json and no override refers to the catalog entry
  //   override <selector>               a package the override names is not in pnpm-lock.yaml
  //   allowBuilds <name>                the package is not in pnpm-lock.yaml
  //   minimumReleaseAgeExclude <name@version>, patchedDependencies <name@version>
  //                                     that exact version is not in pnpm-lock.yaml
  //   peerDependencyRules <selector>    a package the rule names is not in pnpm-lock.yaml
  import fs from "node:fs";
  import path from "node:path";

  const read = (file) => fs.readFileSync(file, "utf8").split("\n");
  const workspace = read("pnpm-workspace.yaml");
  const lock = read("pnpm-lock.yaml");
  // the entries of a top-level section: lines indented by exactly `indent`, comments skipped
  const section = (lines, name, indent) => {
    const start = lines.indexOf(`${name}:`);
    const entries = [];
    for (let i = start + 1; start >= 0 && i < lines.length && !/^\S/.test(lines[i]); i++) {
      const rest = lines[i].slice(indent.length);
      if (lines[i].startsWith(indent) && /^[^\s#]/.test(rest)) entries.push(rest);
    }
    return entries;
  };
  const key = (entry) => entry.replace(/^["']([^"']+)["'].*$|^([^:"']+):.*$/, "$1$2");

  const manifests = ["package.json", "e2e/package.json"];
  for (const dir of ["web/apps", "web/packages"]) {
    for (const name of fs.readdirSync(dir)) manifests.push(path.join(dir, name, "package.json"));
  }
  const catalogRefs = new Set();
  for (const manifest of manifests) {
    const pkg = JSON.parse(fs.readFileSync(manifest, "utf8"));
    for (const deps of [pkg.dependencies, pkg.devDependencies, pkg.peerDependencies]) {
      for (const [name, spec] of Object.entries(deps ?? {})) if (spec === "catalog:") catalogRefs.add(name);
    }
  }
  const overrides = section(workspace, "overrides", "  ");
  for (const entry of overrides) if (entry.endsWith('"catalog:"')) catalogRefs.add(key(entry));
  for (const name of section(workspace, "catalog", "  ").map(key)) {
    if (!catalogRefs.has(name)) console.log(`catalog ${name}`);
  }

  const packages = section(lock, "packages", "  ").map((line) => line.replace(/^'|'?:.*$/g, ""));
  const versions = (name) => packages.filter((p) => p.startsWith(`${name}@`)).map((p) => p.slice(name.length + 1));
  const selectorIsDead = (selector) =>
    selector.split(">").some((part) => {
      const [, name, range] = /^(@?[^@]+)(?:@(.+))?$/.exec(part);
      return versions(name).filter((v) => range === undefined || v.startsWith(`${range}.`)).length === 0;
    });
  for (const selector of overrides.map(key)) if (selectorIsDead(selector)) console.log(`override ${selector}`);
  for (const name of section(workspace, "allowBuilds", "  ").map(key)) {
    if (versions(name).length === 0) console.log(`allowBuilds ${name}`);
  }
  for (const item of section(workspace, "minimumReleaseAgeExclude", "  ")) {
    const exact = item.replace(/^- /, "").replace(/^["']|["']$/g, "");
    if (!packages.includes(exact)) console.log(`minimumReleaseAgeExclude ${exact}`);
  }
  for (const exact of section(workspace, "patchedDependencies", "  ").map(key)) {
    if (!packages.includes(exact)) console.log(`patchedDependencies ${exact}`);
  }
  for (const selector of section(workspace, "peerDependencyRules", "    ").map(key)) {
    if (selectorIsDead(selector)) console.log(`peerDependencyRules ${selector}`);
  }
  ```

  `/tmp/nerve-p1/lock-diff.mjs`：把工作区中的 `pnpm-lock.yaml` 与提交中的比较，打印删除以外的全部差异（M1 设计 8：importers 和存活包的版本、完整性哈希、依赖边）。

  ```js
  // One-off (M1/P1): compares pnpm-lock.yaml with the committed one and prints everything that is not a
  // plain removal. packages and snapshots are compared entry by entry (version, integrity, dependency
  // edges), importers dependency by dependency, the other sections line by line; sections without
  // changes are not printed. Run from the repository root.
  // usage: node lock-diff.mjs [<old lockfile> <new lockfile>]
  import { execFileSync } from "node:child_process";
  import fs from "node:fs";

  const [oldFile, newFile] = process.argv.slice(2);
  const oldText = oldFile
    ? fs.readFileSync(oldFile, "utf8")
    : execFileSync("git", ["show", "HEAD:pnpm-lock.yaml"], { encoding: "utf8", maxBuffer: 64 * 1024 * 1024 });
  const newText = fs.readFileSync(newFile ?? "pnpm-lock.yaml", "utf8");

  // top-level section -> { lines: its trimmed lines, entries: entry key at two spaces -> the entry's
  // lines (its inline value, then the deeper lines) }
  function parse(text) {
    const sections = new Map();
    let section;
    let entry;
    for (const line of text.split("\n")) {
      if (line.trim() === "") continue;
      if (!line.startsWith(" ")) {
        section = { lines: [], entries: new Map() };
        sections.set(line.replace(/:.*$/, ""), section);
        continue;
      }
      section.lines.push(line.trim());
      const top = /^ {2}('[^']*'|[^\s:][^:]*):\s*(.*)$/.exec(line);
      if (top) {
        entry = [top[2]].filter((value) => value !== "" && value !== "{}");
        section.entries.set(top[1].replace(/^'|'$/g, ""), entry);
      } else {
        entry.push(line.trim());
      }
    }
    return sections;
  }

  // one importer's dependencies: "devDependencies tsdown" -> ["specifier: …", "version: …"]
  function dependencies(lines) {
    const deps = new Map();
    let group = "";
    for (let i = 0; i < lines.length; i++) {
      const next = lines[i + 1] ?? "";
      if (/^(specifier|version):/.test(lines[i])) continue;
      if (/^(specifier|version):/.test(next)) {
        deps.set(`${group} ${lines[i].replace(/:$/, "").replace(/^'|'$/g, "")}`, lines.slice(i + 1, i + 3));
      } else {
        group = lines[i].replace(/:$/, "");
      }
    }
    return deps;
  }

  // the lines of a that are not in b (as multisets)
  function minus(a, b) {
    const rest = [...b];
    return a.filter((line) => {
      const i = rest.indexOf(line);
      if (i >= 0) rest.splice(i, 1);
      return i < 0;
    });
  }
  const change = (a, b) => [...minus(a, b).map((l) => `- ${l}`), ...minus(b, a).map((l) => `+ ${l}`)].join("  ");

  const before = parse(oldText);
  const after = parse(newText);
  for (const name of new Set([...before.keys(), ...after.keys()])) {
    const a = before.get(name)?.entries ?? new Map();
    const b = after.get(name)?.entries ?? new Map();
    const notes = [];
    let removed = 0;
    if (name === "importers") {
      for (const importer of new Set([...a.keys(), ...b.keys()])) {
        const da = dependencies(a.get(importer) ?? []);
        const db = dependencies(b.get(importer) ?? []);
        for (const [dep, lines] of da) {
          if (!db.has(dep)) removed++;
          else if (lines.join() !== db.get(dep).join()) notes.push(`  ~ ${importer} ${dep}: ${change(lines, db.get(dep))}`);
        }
        for (const [dep, lines] of db) if (!da.has(dep)) notes.push(`  + ${importer} ${dep}: ${lines.join("  ")}`);
      }
      if (removed || notes.length) console.log(`importers: ${removed} dependencies removed, ${notes.length} added or changed`);
    } else if (name === "packages" || name === "snapshots") {
      for (const [key, lines] of a) {
        if (!b.has(key)) removed++;
        else if (lines.join("\n") !== b.get(key).join("\n")) notes.push(`  ~ ${key}: ${change(lines, b.get(key))}`);
      }
      for (const key of b.keys()) if (!a.has(key)) notes.push(`  + ${key}`);
      if (removed || notes.length) console.log(`${name}: ${removed} removed, ${notes.length} added or changed`);
    } else {
      const la = before.get(name)?.lines ?? [];
      const lb = after.get(name)?.lines ?? [];
      removed = minus(la, lb).length;
      notes.push(...minus(lb, la).map((line) => `  + ${line}`));
      if (removed || notes.length) console.log(`${name}: ${removed} lines removed, ${notes.length} added`);
    }
    for (const note of notes.toSorted()) console.log(note);
  }
  ```

  `/tmp/nerve-p1/web-size.mjs`：构建体积（M1 设计 7.6）。P1 和收尾用同一个脚本、同一套步骤（spec 2.14），所以它的全文在这里保留到收尾。

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

- **锁文件核对**（M1 设计 8，每次改依赖之后、提交之前）：

  ```bash
  node /tmp/nerve-p1/lock-diff.mjs
  wc -l < pnpm-lock.yaml
  shasum -a 256 pnpm-lock.yaml
  ```

  `lock-diff.mjs` 的输出必须与各 Task 给出的逐行相同：`packages` 一行必须是 `N removed, 0 added or changed`（没有新增包，没有存活包的版本或完整性哈希变化），其余差异都在 spec 2.6 的表中解释过。随后：`pnpm install --frozen-lockfile 2>&1 | tail -1` 输出 `Done in …`；`node /tmp/nerve-p1/unused-entries.mjs` 没有输出；`pnpm peers check` 与 P1 之前相同（只有 `@tailwindcss/typography@0.5.19` 缺少对等依赖 `tailwindcss` 一条）。`pnpm install` 的输出中出现 `[WARN] Issues with peer dependencies found` 是这一条，P1 之前就有。
- **关键词守卫的规则**（Task 2 起）：每个删除 Task 先把自己的规则加进 `tools/keywords.json`（插在 `rules` 数组的最后），运行一次守卫，确认它命中要删的东西；删完之后再运行，确认没有命中。数命中的命令：

  ```bash
  node tools/keywords.mjs 2>&1 | grep -v '^keywords:' | awk '{print $1}' | sort | uniq -c
  ```

  规则中的 `why` 用中文；正则写在 JSON 字符串里，反斜杠要写两遍。改了 `tools/` 下的任何文件，都执行一次 `pnpm exec oxfmt --check tools/ && pnpm exec oxlint tools/`（`make lint-web` 不检查仓库根目录的文件），预期 `All matched files use the correct format.` 和 `Found 0 warnings and 0 errors.`。
- **每个 Task 的收尾检查**（各 Task 给出预期值）：
  1. `make build-web`：`Tasks:    11 successful, 11 total`；
  2. `make lint-web`：Task 2 起第一行是守卫的 `keywords: N rules, 0 exceptions, no hits.`；turbo 部分 Task 1–8 为 `Tasks:    49 successful, 49 total`，Task 9、10 为 50；它包含 lint 上限的核对（Task 1 起）；
  3. `make test-web`（Task 3 起）：Task 3–7 为 `Tasks:    11 successful, 11 total`，Task 8 起为 12；
  4. `make knip 2>&1 | grep -E '^(Unused|Unlisted|Duplicate|Configuration)'`：M1 中 knip 只出报告（P3 才改为门禁），核对各类的条数与本计划相同。`Configuration hints` 的条数取决于本地生成过哪些文件（P6 交接），只核对它列出的内容；
  5. `git status --short | cut -c1-2 | sort | uniq -c` 与各 Task 给出的一致（用 `git rm` 删除的文件显示为 `D `，改动的文件为 ` M`，新文件为 `??`），没有多出的文件。
- **lint 上限**：`make lint-web` 报 `… fewer than the cap of N. Lower the cap … to M` 时，把这个包的上限改为 M（`node /tmp/nerve-p1/pkg.mjs <包目录>/package.json set-script check:lint "node <相对路径>/tools/lint-cap.mjs M"`）。本计划在会减少警告的 Task 中已经写出新的上限；实测值不同就停下来，不要照着输出改。
- **Docker**：只有 Task 10 的 `make e2e` 通过 testcontainers 使用 Docker。不要停止或改动其他任何容器（包括 `agentforge-*`、`plane-app-*`、`opennerve-*`、`nerve-dev-db-1`）。
- **后台进程**（Task 7、8 的静态服务器，Task 10 的 `make web-dev`）在所在的步骤结束前停止，并确认用过的端口没有监听。
- **删除就是删除**：不留空实现、开关或兼容代码。代码注释用英文；配置文件、YAML、Makefile 中的注释用中文；Plane 原有的英文注释保留原文（本计划写明要删的句子除外）。
- **代码块**：标明"完整内容"的照原样写入整个文件；"把……替换为……"给出原文和新文，原文在文件中只出现一次。Makefile 的命令行以 Tab 开头：写入后用 `grep -c "$(printf '\t')" Makefile` 核对（P1 之前 `32`，Task 2 之后 `33`，Task 3 起 `34`）。
- 提交信息用英文，末尾加：`Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`。提交前用 `git add -u`（加上新文件）暂存，提交后 `git status --short` 没有输出。

## 控制者评审补充（执行前必读）

评审本计划时做了两处补充，执行 Task 1、Task 2 时又各做了一处裁定（第 3、4 项），与正文冲突时以本节为准。

1. **一次性脚本的目录**：本会话的沙箱不允许随意使用 `/tmp`。正文中所有的 `/tmp/nerve-p1/` 一律换成 `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p1/`，下文写作 `$P1TMP`。它是会话的临时目录，不在仓库里。
2. **新增 Task 9A：门禁覆盖 `tools/`**。
   - **原因**：`tools/lint-cap.mjs`、`tools/keywords.mjs` 本身就是门禁，但它们不属于任何工作区包，`make lint-web` 不检查它们的 lint 和格式。正文的做法是"改了就手工检查一次"，以后的改动没有门禁看着，这是门禁自己的漏洞。
   - **做法**：在 Task 9 之后、Task 10 之前执行，单独一个提交：
     1. 根目录 `package.json` 加两个脚本：`"check:lint": "oxlint --max-warnings=0 tools"` 和 `"check:format": "oxfmt --check tools"`。
        - 先确认 oxfmt 只检查 `tools/` 下的 JS 和 JSON 文件。如果它也处理 `tools/plane-schema/` 中的 YAML 或 Markdown，而这些文件不符合格式，就把范围收窄到实际的脚本和规则文件，例如 `tools/*.mjs tools/*.json`。
        - 不要为了通过而改 `plane-schema` 的文件。选定的范围和理由写进提交说明。
     2. `turbo.json` 注册根任务：`"//#check:lint": { "inputs": ["tools/**"], "outputs": [] }`，`"//#check:format": { "cache": false }`（与各包的 `check:format` 一样不缓存，理由见 M0 加固）。
     3. **核对**：
        - `make lint-web` 的 turbo 部分多出 `//#check:lint`、`//#check:format` 两个任务（50 → 52），并且通过；
        - 在 `tools/keywords.mjs` 中临时加一个未使用的变量，`make lint-web` 在 `//#check:lint` 失败；
        - 临时打乱一处缩进，`make lint-web` 在 `//#check:format` 失败；
        - 每项之后恢复原状，恢复后再次通过，并且命中缓存（`check:format` 除外）；
        - `make knip` 的报告与 Task 9 之后相同（`tools/lint-cap.mjs` 由各包的脚本引用，不应报为未使用的文件）。
     4. 此后正文 Global Constraints 中"改了 `tools/` 下的任何文件，都执行一次 `pnpm exec oxfmt --check tools/ && pnpm exec oxlint tools/`"不再需要手工执行，由 `make lint-web` 覆盖。
   - **对 Task 10 的影响**：
     - 预期中的 `make lint-web` 任务数一律从 50 改为 52；
     - README、M0 设计 6.1 中 `lint-web` 的说明加上"也检查 `tools/` 下的脚本"；
     - 前端改动清单"1.3"一节登记根目录的这两个脚本和 turbo 根任务。
   - **提交信息**：`build(tools): lint and format-check the repository tools in make lint-web`。
3. **`tools/` 格式检查的范围（Task 1 执行时的裁定）**：
   - **问题**：`oxfmt --check tools/` 会连带检查 `tools/plane-schema/README.md`。这个文件不符合 oxfmt 的格式，检查因此失败，与新加的脚本无关。
   - **裁定**：这道检查看住的是门禁脚本本身。仓库里的 Markdown 文档（`docs/`、`README.md`）一直不做格式检查；`plane-schema` 是 M0 的快照工具，不单独开例外，也不改它的文件。
   - **做法**：
     - 正文中所有 `pnpm exec oxfmt --check tools/` 一律换成 `pnpm exec oxfmt --check tools/*.mjs tools/*.json`。Task 1 还没有 `tools/*.json`，只写 `tools/*.mjs`。
     - `oxlint tools/` 不变：oxlint 只检查 JS。
     - Task 9A 的根脚本：`"check:lint": "oxlint --max-warnings=0 tools"`；`"check:format": "oxfmt --check tools"`，豁免写在 `.oxfmtrc.json` 的 `ignorePatterns` 里（`tools/plane-schema/**`，`672f103`）。手工命令用通配符是因为它一次性，常设门禁必须递归覆盖：否则以后放在 `tools/` 子目录里的脚本，oxlint 看得见，格式检查看不见。
4. **例外必须覆盖确定的处数（Task 2 代码评审后的裁定）**：
   - **问题**：正文 Task 2 Step 1 给出的 `tools/keywords.mjs` 按 `(rule, path, match)` 三元组匹配例外。同一个文件里同一段原文出现多处时（锁文件里 `serve@14.2.5:` 就是两处），一条例外会把它们全盖住，与 7.4"一条例外只覆盖一处"不符，后来新增的同样一处也会被顺带盖住。
   - **裁定**：例外加一个可选的 `count`（不小于 1 的整数，默认 1），实际处数必须正好等于它，多了少了都失败，与 lint 上限"必须相等"的做法一致。不引入行号：行号随改动漂移，会逼着人频繁改例外。
   - **做法**（已在 `19663bf` 实现，正文 Step 1 的代码是改动之前的版本）：
     - 加载时校验 `count`；
     - 命中按三元组分组，每条例外查自己那一组的处数 `n`：`n` 为 0 报过期；`n` 与 `count` 不符时逐条报 `exception count: <rule>  <path>  "<match>" covers <n> hits, "count" says <m>`，这些命中不再重复列进未登记的命中；
     - 退出码不变：未登记的命中、过期的例外、处数不符都以 1 退出，其余错误以 2 退出；
     - 成功那行仍是 `keywords: N rules, M exceptions, no hits.`，失败那行多一段处数不符的计数。
     - 同一个 `(rule, path, match)` 只能登记一条例外，重复登记以 2 退出（`4eb7bc5`）：两条例外各自与整组命中比较，重复的那条不起作用又不报错。
   - **同步**：M1 设计 7.4、spec 2.4、spec 2.2 的原型结论表和 spec 第 4 节验收已相应改写。

## 文件结构

路径相对于仓库根目录。

| 文件 | 职责 | Task |
|---|---|---|
| `tools/lint-cap.mjs`（新增） | 运行 oxlint，要求警告数等于上限 | 1 |
| `turbo.json` | `check:lint` 的输入（1）；删掉没有调用方的任务，保留 `test`（3）；`check:sync`（9） | 1、3、9 |
| `e2e/package.json`、`web/apps/web/package.json`、`web/packages/*/package.json` | `check:lint`（1）；脚本和依赖（2、3、4、6、8、9）；上限（3、4、5、9） | 1–6、8、9 |
| `tools/keywords.mjs`、`tools/keywords.json`（新增） | 关键词守卫；每个删除 Task 加规则 | 2–5、8、9 |
| `Makefile` | `lint-web`（2、9）；`test-web`（3）；`web-dev`（10） | 2、3、9、10 |
| `knip.jsonc` | 守卫的 `entry`（2）；i18n 的 `ignoreUnresolved`（9） | 2、9 |
| `.github/workflows/ci.yml` | `web` 任务的单元测试一步 | 3 |
| `web/apps/web/` 的部署遗留、`public/` 的无用文件、13 个 `.prettierignore`（删除） | 部署遗留 | 2 |
| `.oxfmtrc.json` | 代替 `.prettierignore`（2）；`keys.generated.ts`（9） | 2、9 |
| `pnpm-workspace.yaml`、`pnpm-lock.yaml`（生成） | 失效条目、注释；锁文件随依赖变化 | 2、3、4、6（锁文件还有 8） |
| `.oxlintrc.json` | Storybook 的忽略规则 | 3 |
| ui、propel 的 Storybook 和只为它存在的文件、`web/packages/propel/src/toast/toast.tsx` 中的 `ToastStatic` | 没有入口 | 3 |
| 只为未使用依赖存在的文件、垫片的死文件（删除）；`web/apps/web/vite.config.ts` | 死代码 | 4、6 |
| `web/packages/services/src/`（49 个文件删除，三个入口文件） | 修剪 | 5 |
| `web/packages/tailwind-config/package.json`、`web/packages/typescript-config/base.json` | 构建警告、增量编译文件 | 6 |
| `web/apps/web/app/root.tsx`、`web/apps/web/core/components/common/logo-spinner.tsx` | React #418 | 7 |
| `web/packages/i18n/src/locales/` 的 19 种语言（删除）、`src/constants/language.ts`、`src/constants/language.test.ts`（新增）、`src/types/language.ts`、`src/index.ts`、`src/core/instance.ts`；web 的 `core/store/user/profile.store.ts`、`core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx` | 只留中英文、回退到英文 | 8 |
| `web/packages/i18n/scripts/`、`src/types/index.ts`、`src/index.ts`、`src/constants/namespaces.ts`、8 个命名空间（删除）、两个 `workspace-settings.json`；`.gitignore` | 翻译键生成、键一致性检查、死文案 | 9 |
| `docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md`、`M0-P6-knip-notes.md` | 文档同步 | 10 |

---

### Task 1: lint 警告数必须等于上限（`tools/lint-cap.mjs`）；P1 之前的基线

**Files:**
- Create: `tools/lint-cap.mjs`
- Modify: `turbo.json`；13 个 `package.json`（`e2e`、`web/apps/web`、`web/packages/{api-client,constants,editor,hooks,i18n,propel,services,shared-state,types,ui,utils}`）

**Interfaces:**
- Consumes: 各包现在的 `check:lint`（`oxlint --max-warnings=N .`）；`make lint-web`
- Produces: `node <相对路径>/tools/lint-cap.mjs <上限>`：警告数等于上限时退出码 0；多了、少了、有错误或 oxlint 无法运行时 1；参数不对时 2。之后每个 Task 的 `make lint-web` 都经过它

- [ ] **Step 1: 写入一次性脚本，确认 P1 之前 `pnpm-workspace.yaml` 没有失效条目**

按 Global Constraints 写入 `/tmp/nerve-p1/pkg.mjs`、`unused-entries.mjs`、`lock-diff.mjs`、`web-size.mjs`。

Run: `node /tmp/nerve-p1/unused-entries.mjs`
Expected: 没有输出。

- [ ] **Step 2: 记下 P1 之前的检查结果和构建体积的基线**

Run: `make lint-web 2>&1 | grep -E 'Tasks:'`
Expected: ` Tasks:    49 successful, 49 total`

Run: `make knip 2>&1 | grep -E '^(Unused|Unlisted|Duplicate|Configuration)'`
Expected（`Configuration hints` 为 1–3 条：tailwind-config 的 `main`、i18n 的 `ignoreUnresolved`、web 的 `+types/`，取决于本地生成过哪些文件）:

```
Unused files (131)
Unused dependencies (19)
Unused devDependencies (4)
Unlisted dependencies (2)
Unused exports (135)
Unused exported types (83)
Unused exported enum members (4)
Duplicate exports (1)
Configuration hints (3)
```

构建体积（M1 设计 7.6；先删掉 `build/`，turbo 从缓存恢复产物时不会删除目录中多出的旧文件）：

Run: `rm -rf web/apps/web/build && make build-web > /dev/null && node /tmp/nerve-p1/web-size.mjs`
Expected（写进 P1 的 review，收尾时用同一个脚本对比）:

```
js: 1055 files, 15151327 bytes
css: 3 files, 326068 bytes
fonts: 34 files, 6849620 bytes
other: 147 files, 9905117 bytes
largest chunk: assets/toolbar-Bnxh0Ws1.js, 1821937 bytes
locale chunks: 588
```

- [ ] **Step 3: 写入 `tools/lint-cap.mjs`（完整内容）**

```js
// Runs oxlint in the current directory (a workspace package) and requires the number of warnings
// to equal the cap given as the only argument. Each package's check:lint script is the one place
// that holds its cap; the cap may only go down (docs/v0/M0-foundation/M0-design.md 5.2,
// docs/v0/M1-frontend-trim/M1-design.md 7.1).
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";

const args = process.argv.slice(2);
if (args.length !== 1 || !/^\d+$/.test(args[0])) {
  console.error("usage: node tools/lint-cap.mjs <cap>");
  process.exit(2);
}
const cap = Number(args[0]);
const { name } = JSON.parse(readFileSync("package.json", "utf8"));

const oxlint = spawnSync("oxlint", ["--format=json", "."], { encoding: "utf8", maxBuffer: 256 * 1024 * 1024 });
if (oxlint.error) throw oxlint.error;

let diagnostics;
try {
  ({ diagnostics } = JSON.parse(oxlint.stdout));
} catch {
  process.stderr.write(oxlint.stdout + oxlint.stderr);
  console.error(`${name}: oxlint did not print a JSON report (exit code ${oxlint.status}).`);
  process.exit(1);
}

const position = (d) => d.labels[0]?.span ?? { line: 0, column: 0 };
const byLocation = (a, b) =>
  a.filename.localeCompare(b.filename) ||
  position(a).line - position(b).line ||
  position(a).column - position(b).column;
const print = (list) => {
  for (const d of list.toSorted(byLocation)) {
    const { line, column } = position(d);
    // a file that does not parse gives a diagnostic without a rule code
    console.error(`${d.filename}:${line}:${column}  ${d.code ?? "syntax"}  ${d.message}`);
  }
};

const count = (n, noun) => `${n} ${noun}${n === 1 ? "" : "s"}`;
const warnings = diagnostics.filter((d) => d.severity === "warning");
const errors = diagnostics.filter((d) => d.severity !== "warning");

if (errors.length > 0) {
  print(errors);
  console.error(`${name}: oxlint reports ${count(errors.length, "error")} (listed above); errors are never allowed.`);
  process.exit(1);
}
if (warnings.length > cap) {
  print(warnings);
  console.error(
    `${name}: more oxlint warnings than the cap of ${cap}. Fix the new warnings (every warning is listed above); the cap may only go down.`
  );
  process.exit(1);
}
if (warnings.length < cap) {
  console.error(
    `${name}: ${count(warnings.length, "oxlint warning")}, fewer than the cap of ${cap}. Lower the cap in the check:lint script of package.json to ${warnings.length} in the same commit.`
  );
  process.exit(1);
}
console.log(`${name}: ${count(warnings.length, "oxlint warning")}, equal to the cap.`);
```

Run: `pnpm exec oxfmt --check tools/ && pnpm exec oxlint tools/`
Expected: `All matched files use the correct format.` 和 `Found 0 warnings and 0 errors.`

- [ ] **Step 4: 13 个包的 `check:lint` 改为调用它，上限不变**

```bash
P=/tmp/nerve-p1/pkg.mjs
node $P web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 779"
node $P web/packages/editor/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 75"
node $P web/packages/propel/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 59"
node $P web/packages/utils/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 34"
node $P web/packages/ui/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 32"
node $P web/packages/services/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 6"
node $P web/packages/hooks/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 4"
node $P web/packages/i18n/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 3"
node $P web/packages/constants/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 2"
node $P web/packages/types/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 1"
node $P web/packages/shared-state/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 0"
node $P web/packages/api-client/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 0"
node $P e2e/package.json set-script check:lint "node ../tools/lint-cap.mjs 0"
```

Run: `git grep -c 'oxlint --max-warnings' -- '*package.json'; git diff --numstat -- '*package.json' | awk '{a += $1; d += $2} END {print NR, a, d}'`
Expected（第一条没有输出；13 个文件各改一行）:

```
13 13 13
```

- [ ] **Step 5: 修改 `turbo.json`（脚本列为 `check:lint` 的输入）**

包外的文件不在任务的哈希里；不加这一项，只改 `lint-cap.mjs` 时 turbo 会重放旧的结果（spec 2.3）。

把：

```json
    "check:lint": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md"],
```

替换为：

```json
    "check:lint": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md", "$TURBO_ROOT$/tools/lint-cap.mjs"],
```

- [ ] **Step 6: 执行 `make lint-web`，逐包核对**

Run: `make lint-web 2>&1 | grep -E 'Tasks:'`
Expected: ` Tasks:    49 successful, 49 total`

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:lint --output-logs=full 2>&1 | grep -oE '[@a-z0-9/-]+: [0-9]+ oxlint warnings?, equal to the cap' | sort`
Expected:

```
@nerve/api-client: 0 oxlint warnings, equal to the cap
@nerve/e2e: 0 oxlint warnings, equal to the cap
@plane/constants: 2 oxlint warnings, equal to the cap
@plane/editor: 75 oxlint warnings, equal to the cap
@plane/hooks: 4 oxlint warnings, equal to the cap
@plane/i18n: 3 oxlint warnings, equal to the cap
@plane/propel: 59 oxlint warnings, equal to the cap
@plane/services: 6 oxlint warnings, equal to the cap
@plane/shared-state: 0 oxlint warnings, equal to the cap
@plane/types: 1 oxlint warning, equal to the cap
@plane/ui: 32 oxlint warnings, equal to the cap
@plane/utils: 34 oxlint warnings, equal to the cap
web: 779 oxlint warnings, equal to the cap
```

- [ ] **Step 7: 收尾检查**

- `make build-web`：11 个任务；
- `make knip`：与 Step 2 相同（`tools/lint-cap.mjs` 由各包的脚本调用，knip 不报告它）；
- `git status --short | cut -c1-2 | sort | uniq -c`：

```
  14  M
   1 ??
```

（13 个 `package.json` 和 `turbo.json`；新文件 `tools/lint-cap.mjs`。）

- [ ] **Step 8: 提交**

```bash
git add -u && git add tools/lint-cap.mjs
git commit -m "build(web): require each package's oxlint warnings to equal its cap

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 9: 核对错误和缓存（M1 设计 7.1；每项之后恢复原状）**

错误与参数（`src/lint-probe.ts` 是一个无法解析的临时文件）：

```bash
printf 'export const = ;\n' > web/packages/types/src/lint-probe.ts
pnpm --dir web/packages/types exec node ../../../tools/lint-cap.mjs 1; echo "exit=$?"
rm web/packages/types/src/lint-probe.ts
pnpm --dir web/packages/types exec node ../../../tools/lint-cap.mjs; echo "exit=$?"
```

Expected:

```
src/lint-probe.ts:1:14  syntax  Unexpected token
@plane/types: oxlint reports 1 error (listed above); errors are never allowed.
exit=1
usage: node tools/lint-cap.mjs <cap>
exit=2
```

缓存：写入 `/tmp/nerve-p1/cache-proof.sh`（完整内容），用一个单独的缓存目录依次执行 M1 设计 7.1 列出的六种情况：

```bash
#!/bin/bash
# One-off (M1/P1 Task 1): turbo check:lint in the situations of M1 design 7.1, with a cache directory of its own
# (the first run is cold); every change is undone right after its run. Run from the repository root with a
# clean working tree.
set -uo pipefail
CACHE=/tmp/nerve-p1/turbo-cache
rm -rf "$CACHE"
run() {
  echo "== $1"
  TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:lint --cache-dir="$CACHE" --continue --output-logs=errors-only 2>&1 \
    | grep -E 'Tasks:|Cached:|Failed:|fewer than|more oxlint|lint-probe' | sed 's/^ *//'
}
run "cold"
run "warm"
rm web/packages/types/src/issues/issue_subscription.ts
run "one warning fewer: the empty file that is the only warning of types is gone"
git checkout -- web/packages/types/src/issues/issue_subscription.ts
run "restored"
printf 'export function lintProbe(): void {\n  const unused = 1;\n}\n' > web/apps/web/core/lint-probe.ts
run "one warning more: an unused variable in web"
rm web/apps/web/core/lint-probe.ts
run "restored"
node /tmp/nerve-p1/pkg.mjs web/packages/types/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 2"
run "cap changed: types 1 -> 2"
git checkout -- web/packages/types/package.json
run "restored"
printf '// cache probe\n' >> tools/lint-cap.mjs
run "the script changed"
git checkout -- tools/lint-cap.mjs
run "restored"
printf '// cache probe\n' >> web/packages/hooks/src/index.ts
run "a dependency changed: the source of hooks"
git checkout -- web/packages/hooks/src/index.ts
run "restored"
git status --short
```

Run: `bash /tmp/nerve-p1/cache-proof.sh`
Expected（23 个任务是 13 个 `check:lint` 和它们依赖的 10 个包的构建；每个 `restored` 都是 `Tasks:    23 successful, 23 total` 和 `Cached:    23 cached, 23 total`，下面省略；最后的 `git status --short` 没有输出）:

```
== cold
Tasks:    23 successful, 23 total
Cached:    0 cached, 23 total
== warm
Tasks:    23 successful, 23 total
Cached:    23 cached, 23 total
== one warning fewer: the empty file that is the only warning of types is gone
@plane/types:check:lint: @plane/types: 0 oxlint warnings, fewer than the cap of 1. Lower the cap in the check:lint script of package.json to 0 in the same commit.
Tasks:    22 successful, 23 total
Cached:    6 cached, 23 total
Failed:    @plane/types#check:lint
== one warning more: an unused variable in web
web:check:lint: core/lint-probe.ts:2:9  eslint(no-unused-vars)  Variable 'unused' is declared but never used. Unused variables should start with a '_'.
web:check:lint: web: more oxlint warnings than the cap of 779. Fix the new warnings (every warning is listed above); the cap may only go down.
Tasks:    22 successful, 23 total
Cached:    22 cached, 23 total
Failed:    web#check:lint
== cap changed: types 1 -> 2
@plane/types:check:lint: @plane/types: 1 oxlint warning, fewer than the cap of 2. Lower the cap in the check:lint script of package.json to 1 in the same commit.
Tasks:    22 successful, 23 total
Cached:    6 cached, 23 total
Failed:    @plane/types#check:lint
== the script changed
Tasks:    23 successful, 23 total
Cached:    10 cached, 23 total
== a dependency changed: the source of hooks
Tasks:    23 successful, 23 total
Cached:    14 cached, 23 total
```

读法：types 的源码或 `package.json` 变了，types 的构建和依赖它的包全部重跑（6 个命中）；只改 web 时只有 web 的 lint 重跑；改脚本时 13 个 `check:lint` 重跑、10 个构建命中；改 hooks 时 hooks 和依赖它的 editor、propel、ui 的构建和 lint 以及 web 的 lint 重跑（9 个）。失败的任务不缓存。输出写进 P1 的 review。

---

### Task 2: 关键词守卫；部署遗留；`serve` 与 `pnpm-workspace.yaml` 的注释

**Files:**
- Create: `tools/keywords.mjs`、`tools/keywords.json`
- Delete: `web/apps/web/{Dockerfile.web,Dockerfile.dev,caddy/Caddyfile,.dockerignore,.gitignore,manifest.json}`、`web/apps/web/public/{sw.js,sw.js.map,workbox-9f2f79cf.js,workbox-9f2f79cf.js.map}`、`web/apps/web/public/favicon/`（3 个文件）、13 个 `.prettierignore`
- Modify: `Makefile`、`knip.jsonc`、`.oxfmtrc.json`、`web/apps/web/package.json`、`pnpm-workspace.yaml`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 1 的 `make lint-web`
- Produces: `node tools/keywords.mjs`（从仓库根目录运行）：没有命中时退出码 0，有未登记的命中或过期的例外时 1，规则、正则、样本、git 或读取出错时 2；`make lint-web` 的第一行运行它。规则文件的格式见 spec 2.4，之后的 Task 往 `rules` 里加规则

- [ ] **Step 1: 写入 `tools/keywords.mjs`（完整内容）**

```js
// Keyword guard (docs/v0/M1-frontend-trim/M1-design.md 7.4): fails when a file hits a rule in
// tools/keywords.json that no exception covers, or when an exception no longer matches anything.
// The files are those git lists (tracked, plus untracked ones that are not ignored) as they are in the
// working tree. A path rule tests each path; a content rule tests the text of the files its `files`
// pattern selects (binary files are skipped). Patterns are JavaScript regular expressions with explicit
// flags, and every rule is first checked against its own samples. A problem with the rules file, a
// pattern, git or reading a file exits with 2, so an error is never taken for "no hits".
// usage: node tools/keywords.mjs   (from the repository root)
import { spawnSync } from "node:child_process";
import fs from "node:fs";

const RULES_FILE = "tools/keywords.json";

function fail(message) {
  console.error(`keywords: ${message}`);
  process.exit(2);
}

function compile(rule, key) {
  const spec = rule[key];
  if (typeof spec?.source !== "string" || typeof spec.flags !== "string") {
    fail(`rule ${rule.id}: "${key}" needs a "source" string and a "flags" string`);
  }
  if (!/^(?!.*(.).*\1)[imsu]*$/.test(spec.flags)) fail(`rule ${rule.id}: the flags of "${key}" may only be i, m, s, u`);
  try {
    return new RegExp(spec.source, spec.flags);
  } catch (error) {
    return fail(`rule ${rule.id}: "${key}" is not a valid regular expression: ${error.message}`);
  }
}

function loadRules() {
  let config;
  try {
    config = JSON.parse(fs.readFileSync(RULES_FILE, "utf8"));
  } catch (error) {
    fail(`cannot read ${RULES_FILE}: ${error.message}`);
  }
  if (!Array.isArray(config.rules) || !Array.isArray(config.exceptions)) {
    fail(`${RULES_FILE} needs "rules" and "exceptions" arrays`);
  }
  const ids = new Set();
  const rules = config.rules.map((rule) => {
    if (typeof rule.id !== "string" || !/^[a-z0-9-]+$/.test(rule.id) || ids.has(rule.id)) {
      fail(`every rule needs a unique id of lowercase letters, digits and dashes (got ${JSON.stringify(rule.id)})`);
    }
    ids.add(rule.id);
    if (typeof rule.phase !== "string" || typeof rule.why !== "string" || rule.why === "") {
      fail(`rule ${rule.id} needs "phase" and "why"`);
    }
    const isPath = "path" in rule;
    if (isPath === ("files" in rule || "content" in rule)) {
      fail(`rule ${rule.id} needs either "path", or "files" and "content"`);
    }
    const compiled = isPath
      ? { id: rule.id, path: compile(rule, "path") }
      : { id: rule.id, files: compile(rule, "files"), content: compile(rule, "content") };
    const test = isPath ? compiled.path : compiled.content;
    const { hit, miss } = rule.samples ?? {};
    if (!Array.isArray(hit) || hit.length === 0 || !Array.isArray(miss) || miss.length === 0) {
      fail(`rule ${rule.id} needs "samples" with at least one "hit" and one "miss"`);
    }
    for (const sample of hit) {
      if (!test.test(sample)) fail(`rule ${rule.id} does not match its hit sample ${JSON.stringify(sample)}`);
    }
    for (const sample of miss) {
      if (test.test(sample)) fail(`rule ${rule.id} matches its miss sample ${JSON.stringify(sample)}`);
    }
    return compiled;
  });
  for (const e of config.exceptions) {
    if (!ids.has(e.rule) || typeof e.path !== "string" || typeof e.match !== "string") {
      fail(`exception ${JSON.stringify(e)} needs an existing "rule", a "path" and a "match"`);
    }
    if (typeof e.reason !== "string" || e.reason === "" || !/^M\d+(?:\/P\d+)?$/.test(e.until ?? "")) {
      fail(`exception ${JSON.stringify(e)} needs a "reason" and an "until" such as "M3" or "M1/P4"`);
    }
  }
  return { rules, exceptions: config.exceptions };
}

// The text of a file; undefined for a binary file (a NUL byte in the first 8000 bytes, as git decides)
// or when the text is not needed; null for a file deleted in the working tree. git stores a symbolic
// link as the path it points to.
function read(path, needText) {
  try {
    const link = fs.lstatSync(path).isSymbolicLink();
    if (!needText) return undefined;
    const data = link ? Buffer.from(fs.readlinkSync(path)) : fs.readFileSync(path);
    return data.subarray(0, 8000).includes(0) ? undefined : data.toString("utf8");
  } catch (error) {
    if (error.code === "ENOENT") return null;
    return fail(`cannot read ${path}: ${error.message}`);
  }
}

const { rules, exceptions } = loadRules();
const git = spawnSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard"], {
  encoding: "utf8",
  maxBuffer: 256 * 1024 * 1024,
});
if (git.error || git.status !== 0) fail(`git ls-files failed: ${git.error?.message ?? git.stderr.trim()}`);

const hits = [];
for (const path of new Set(git.stdout.split("\0").filter(Boolean))) {
  const contentRules = rules.filter((rule) => rule.files?.test(path));
  const text = read(path, contentRules.length > 0);
  if (text === null) continue;
  for (const rule of rules) if (rule.path?.test(path)) hits.push({ rule: rule.id, path, line: 0, match: path });
  if (text === undefined) continue;
  for (const rule of contentRules) {
    for (const found of text.matchAll(new RegExp(rule.content.source, `${rule.content.flags}g`))) {
      if (found[0] === "") fail(`rule ${rule.id} matches an empty string in ${path}`);
      hits.push({ rule: rule.id, path, line: text.slice(0, found.index).split("\n").length, match: found[0] });
    }
  }
}

const count = (n, noun) => `${n} ${noun}${n === 1 ? "" : "s"}`;
const used = new Set();
const open = hits.filter((hit) => {
  const index = exceptions.findIndex((e) => e.rule === hit.rule && e.path === hit.path && e.match === hit.match);
  if (index >= 0) used.add(index);
  return index < 0;
});
const stale = exceptions.filter((_, index) => !used.has(index));
for (const hit of open) {
  console.error(`${hit.rule}  ${hit.path}${hit.line ? `:${hit.line}` : ""}  ${JSON.stringify(hit.match)}`);
}
for (const e of stale) {
  console.error(`stale exception: ${e.rule}  ${e.path}  ${JSON.stringify(e.match)} matches nothing now; delete it`);
}
if (open.length > 0 || stale.length > 0) {
  console.error(
    `keywords: ${count(open.length, "hit")} without an exception, ${count(stale.length, "stale exception")} (${RULES_FILE}).`
  );
  process.exit(1);
}
console.log(`keywords: ${count(rules.length, "rule")}, ${count(exceptions.length, "exception")}, no hits.`);
```

- [ ] **Step 2: 写入 `tools/keywords.json`（完整内容；本 Task 的三条规则）**

```json
{
  "rules": [
    {
      "id": "deploy-files",
      "phase": "M1/P1",
      "why": "Plane 的 Docker、Caddy 部署文件和从未注册的 service worker；Nerve 把前端编进 nerve，由 nerve 提供",
      "path": {
        "source": "^web/(?:[^/]+/)*(?:Dockerfile\\.(?:web|dev)$|\\.dockerignore$|caddy/|(?:sw|workbox-[^/]*)\\.js(?:\\.map)?$)",
        "flags": ""
      },
      "samples": {
        "hit": [
          "web/apps/web/Dockerfile.web",
          "web/apps/web/.dockerignore",
          "web/apps/web/caddy/Caddyfile",
          "web/apps/web/public/sw.js.map",
          "web/apps/web/public/workbox-9f2f79cf.js"
        ],
        "miss": ["web/apps/web/public/manifest.json", "web/apps/web/public/swagger.js", "deploy/compose.dev.yaml"]
      }
    },
    {
      "id": "prettierignore",
      "phase": "M1/P1",
      "why": "格式化工具是 oxfmt，它也会读取当前目录下的 .prettierignore；忽略规则只写在 .oxfmtrc.json 一处",
      "path": { "source": "(?:^|/)\\.prettierignore$", "flags": "" },
      "samples": {
        "hit": [".prettierignore", "web/packages/ui/.prettierignore"],
        "miss": ["web/packages/ui/.gitignore"]
      }
    },
    {
      "id": "serve",
      "phase": "M1/P1",
      "why": "Plane 用 serve 提供构建产物（start、preview 脚本）；Nerve 由 nerve 提供",
      "files": { "source": "(?:^|/)package\\.json$|^pnpm-(?:workspace|lock)\\.yaml$", "flags": "" },
      "content": { "source": "(?<![\\w@/.-])serve(?:@\\d|[\"']?:|\\s+-s\\b)", "flags": "" },
      "samples": {
        "hit": ["\"serve\": \"catalog:\"", "  serve@14.2.5:", "serve -s build/client -l 3000", "      serve:"],
        "miss": [
          "\"@react-router/serve>express\": \"^5.2.1\"",
          "  serve-static@1.16.2:",
          "preserve: true",
          "observe(value)"
        ]
      }
    }
  ],
  "exceptions": []
}
```

Run: `pnpm exec oxfmt --check tools/ && pnpm exec oxlint tools/`
Expected: `All matched files use the correct format.` 和 `Found 0 warnings and 0 errors.`

- [ ] **Step 3: 删除之前运行守卫，确认三条规则命中要删的东西**

Run: `node tools/keywords.mjs 2>&1 | grep -v '^keywords:' | awk '{print $1}' | sort | uniq -c; node tools/keywords.mjs 2>&1 | tail -1`
Expected:

```
   8 deploy-files
  13 prettierignore
   8 serve
keywords: 29 hits without an exception, 0 stale exceptions (tools/keywords.json).
```

（`serve` 的 8 处：web 的 `package.json` 中两个脚本和依赖，`pnpm-workspace.yaml` 的 catalog，锁文件中 web 的 importer 两处和 `serve@14.2.5` 的两个条目。`@react-router/serve`、`serve-static`、`serve-handler` 不命中。）

- [ ] **Step 4: 接进 `make lint-web` 和 knip**

`Makefile`，把：

```makefile
.PHONY: lint-web
lint-web: ## 前端类型检查、oxlint（按警告基线）、格式检查（需要 Node）
	$(TURBO) run check:types check:lint check:format $(TURBO_QUIET)
```

替换为：

```makefile
# 关键词守卫（tools/keywords.mjs，规则在 tools/keywords.json）检查整个仓库，不属于任何工作区包，所以不经过 turbo
.PHONY: lint-web
lint-web: ## 关键词守卫；前端类型检查、oxlint（警告数等于上限）、格式检查（需要 Node）
	node tools/keywords.mjs
	$(TURBO) run check:types check:lint check:format $(TURBO_QUIET)
```

Run: `grep -c "$(printf '\t')" Makefile`
Expected: `33`

`knip.jsonc`，把：

```jsonc
      "ignoreDependencies": ["@redocly/cli", "turbo"]
    },
```

替换为：

```jsonc
      "ignoreDependencies": ["@redocly/cli", "turbo"],
      // 关键词守卫由 make lint-web 调用，package.json 的脚本中看不到
      "entry": ["tools/keywords.mjs"]
    },
```

- [ ] **Step 5: 删除 26 个文件**

```bash
git rm -q web/apps/web/Dockerfile.web web/apps/web/Dockerfile.dev web/apps/web/caddy/Caddyfile web/apps/web/.dockerignore web/apps/web/.gitignore web/apps/web/manifest.json
git rm -q web/apps/web/public/sw.js web/apps/web/public/sw.js.map web/apps/web/public/workbox-9f2f79cf.js web/apps/web/public/workbox-9f2f79cf.js.map
git rm -q -r web/apps/web/public/favicon
git rm -q $(git ls-files '*.prettierignore')
```

Run: `git status --short | grep -c '^D '`
Expected: `26`

`web/apps/web/public/manifest.json`、`public/site.webmanifest.json` 保留：`app/root.tsx` 引用它们（spec 2.2）。

- [ ] **Step 6: 修改 `.oxfmtrc.json`（代替 `.prettierignore`）**

oxfmt 会读取当前目录下的 `.prettierignore`；删掉之后，各包的格式检查会扫到构建产物（spec 2.5）。这里的路径相对于仓库根目录，从包目录运行时同样生效。

把：

```json
    "api/dist/**",
    "web/packages/api-client/src/schema.gen.ts",
```

替换为：

```json
    "api/dist/**",
    "web/apps/web/.react-router/**",
    "web/apps/web/build/**",
    "web/packages/*/dist/**",
    "web/packages/api-client/src/schema.gen.ts",
```

- [ ] **Step 7: 删除 web 的 `preview`、`start` 脚本和 `serve` 依赖**

```bash
node /tmp/nerve-p1/pkg.mjs web/apps/web/package.json del-script preview start
node /tmp/nerve-p1/pkg.mjs web/apps/web/package.json del-dep serve
```

- [ ] **Step 8: 修改 `pnpm-workspace.yaml`（8 处）**

(1) 文件开头的说明。把：

```yaml
# 英文注释是 Plane 的原文（见 docs/v0/M0-foundation/specs/P5-web-import.md 2.4）。
```

替换为：

```yaml
# 英文注释是 Plane 的原文（见 docs/v0/M0-foundation/specs/P5-web-import.md 2.4），
# M1/P1 删掉了其中讲没有迁入的内容的句子（见 docs/v0/M1-frontend-trim/specs/P1-web-hygiene.md 2.6）。
```

(2) 删除 catalog 中的这一行：

```yaml
  "serve": "14.2.5"
```

(3) 把：

```yaml
  # hook call rather than as an install error. The vite `resolve.dedupe` entries only
  # cover the three app client bundles, not the SSR build, the tsdown package builds
  # or Storybook.
```

替换为：

```yaml
  # hook call rather than as an install error.
```

(4) 删除这一行：

```yaml
  # The catalog stays on Express 4 for apps/live, which depends on express-ws (Express 4 only).
```

(5) 把：

```yaml
  # the second bypasses the first's mitigation) reachable via serve>serve-handler>minimatch.
```

替换为：

```yaml
  # the second bypasses the first's mitigation).
```

(6) 删掉 `serve` 之后，锁文件中只剩 Express 5 的 `router` 依赖 path-to-regexp，它由下面更具体的一条决定，全局固定不再作用于任何包（spec 2.6）。把：

```yaml
  # Pinned for Express 4 (apps/live), which needs path-to-regexp ~0.1.12.
  path-to-regexp: 0.1.13
  # Express 5's router needs path-to-regexp 8; the pin above would otherwise reach it
  # and break route matching at startup with `pathRegexp.match is not a function`.
  "router>path-to-regexp": "^8.4.2"
```

替换为：

```yaml
  "router>path-to-regexp": "^8.4.2"
```

(7) 把：

```yaml
  # (CVE-2026-73088 / CVE-2026-73089). browserslist is build tooling only
  # (babel/postcss/vite), but the runtime images copy node_modules wholesale, so
  # it lands in the scanned surface.
```

替换为：

```yaml
  # (CVE-2026-73088 / CVE-2026-73089).
```

(8) 把：

```yaml
    # call sites import usePopper only. Recorded here so the acceptance is explicit
    # rather than hidden behind the global strict-peer-dependencies=false.
```

替换为：

```yaml
    # call sites import usePopper only.
```

Run: `git diff --numstat pnpm-workspace.yaml`
Expected: `6	16	pnpm-workspace.yaml`

- [ ] **Step 9: 安装，核对锁文件**

Run: `pnpm install 2>&1 | tail -1`
Expected: `Done in …`（没有 `[ERR_PNPM_…]`）

按 Global Constraints 执行锁文件核对。Expected:

```
catalogs: 3 lines removed, 0 added
overrides: 1 lines removed, 0 added
importers: 1 dependencies removed, 0 added or changed
packages: 43 removed, 0 added or changed
snapshots: 43 removed, 10 added or changed
  ~ bytes@3.1.2: + optional: true
  ~ compressible@2.0.18: + optional: true
  ~ compression@1.8.1: + optional: true
  ~ debug@2.6.9: + optional: true
  ~ mime-db@1.54.0: + optional: true
  ~ ms@2.0.0: + optional: true
  ~ negotiator@0.6.4: + optional: true
  ~ on-headers@1.1.0: + optional: true
  ~ tsx@4.20.6: - get-tsconfig: 4.13.7  + get-tsconfig: 4.14.3
  ~ vary@1.1.2: + optional: true
   15067
eae0d03a4805a7bf871abd95af024b176cb763c892e4355ef2a20ee8d4356263  pnpm-lock.yaml
```

（9 个包原来也经 `serve` 可达，现在只经可选的 `@react-router/serve` 可达；`get-tsconfig` 是重新解析时向图中已有的 4.14.3 去重，spec 2.6。）然后执行 `pnpm install --frozen-lockfile 2>&1 | tail -1`、`node /tmp/nerve-p1/unused-entries.mjs`、`pnpm peers check`，结果见 Global Constraints。

- [ ] **Step 10: 收尾检查**

先构建，让各包的 `dist/` 和 web 的 `build/` 都存在，再做格式检查，确认 oxfmt 不会扫到它们：

- `make build-web`：11 个任务；
- `make lint-web`：第一行 `keywords: 3 rules, 0 exceptions, no hits.`，49 个任务（其中 13 个 `check:format` 都通过）；
- `make knip`：与 Task 1 相比只有 `Unused files (129)`（`sw.js`、`workbox-9f2f79cf.js` 不再出现；`tools/keywords.mjs` 是 `entry`，不被报告）；
- `git status --short | cut -c1-2 | sort | uniq -c`：

```
   6  M
   2 ??
  26 D 
```

（6 个修改：`.oxfmtrc.json`、`Makefile`、`knip.jsonc`、`pnpm-lock.yaml`、`pnpm-workspace.yaml`、`web/apps/web/package.json`；两个新文件在 `tools/`。）

- [ ] **Step 11: 提交**

```bash
git add -u && git add tools/keywords.mjs tools/keywords.json
git commit -m "build(web): add the keyword guard; remove deployment leftovers and the serve dependency

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 12: 核对守卫的失败方式（每项之后恢复原状）**

写入 `/tmp/nerve-p1/keywords-proof.sh`（完整内容）：

```bash
#!/bin/bash
# One-off (M1/P1 Task 2): the keyword guard's failure modes. Each case edits tools/keywords.json (or a file) and
# is undone right after. Run from the repository root with a clean working tree.
set -uo pipefail
edit() {
  node -e '
    const fs = require("fs");
    const c = JSON.parse(fs.readFileSync("tools/keywords.json", "utf8"));
    new Function("c", process.argv[1])(c);
    fs.writeFileSync("tools/keywords.json", JSON.stringify(c, null, 2) + "\n");
  ' "$1"
}
guard() {
  node tools/keywords.mjs > /tmp/nerve-p1/keywords.out 2>&1
  echo "exit=$?  $(tail -1 /tmp/nerve-p1/keywords.out)"
}
echo "== an exception that matches nothing"
edit 'c.exceptions.push({ rule: "serve", path: "web/apps/web/package.json", match: "serve@9", reason: "probe", until: "M1/P2" });'
guard; git checkout -- tools/keywords.json
echo "== an invalid pattern"
edit 'c.rules[0].path.source = "(unclosed";'
guard; git checkout -- tools/keywords.json
echo "== a hit sample the pattern misses"
edit 'c.rules[1].samples.hit.push("web/packages/ui/.prettierrc");'
guard; git checkout -- tools/keywords.json
echo "== the g flag"
edit 'c.rules[2].content.flags = "g";'
guard; git checkout -- tools/keywords.json
echo "== an exception without an expiry"
edit 'c.exceptions.push({ rule: "serve", path: "a", match: "b", reason: "probe" });'
guard; git checkout -- tools/keywords.json
echo "== an untracked file that hits a rule"
mkdir -p web/apps/web/keywords-probe
printf '{ "dependencies": { "serve": "catalog:" } }\n' > web/apps/web/keywords-probe/package.json
guard; head -1 /tmp/nerve-p1/keywords.out
echo "== an unreadable file"
chmod 000 web/apps/web/keywords-probe/package.json
guard
chmod 644 web/apps/web/keywords-probe/package.json
rm -r web/apps/web/keywords-probe
git status --short
```

Run: `bash /tmp/nerve-p1/keywords-proof.sh`
Expected（最后的 `git status --short` 没有输出）:

```
== an exception that matches nothing
exit=1  keywords: 0 hits without an exception, 1 stale exception (tools/keywords.json).
== an invalid pattern
exit=2  keywords: rule deploy-files: "path" is not a valid regular expression: Invalid regular expression: /(unclosed/: Unterminated group
== a hit sample the pattern misses
exit=2  keywords: rule prettierignore does not match its hit sample "web/packages/ui/.prettierrc"
== the g flag
exit=2  keywords: rule serve: the flags of "content" may only be i, m, s, u
== an exception without an expiry
exit=2  keywords: exception {"rule":"serve","path":"a","match":"b","reason":"probe"} needs a "reason" and an "until" such as "M3" or "M1/P4"
== an untracked file that hits a rule
exit=1  keywords: 1 hit without an exception, 0 stale exceptions (tools/keywords.json).
serve  web/apps/web/keywords-probe/package.json:1  "serve\":"
== an unreadable file
exit=2  keywords: cannot read web/apps/web/keywords-probe/package.json: EACCES: permission denied, open 'web/apps/web/keywords-probe/package.json'
```

输出写进 P1 的 review。

---

### Task 3: 没有调用方的 turbo 任务和脚本；删除 Storybook；`make test-web`

**Files:**
- Modify: `turbo.json`、`.oxlintrc.json`、`pnpm-workspace.yaml`、`Makefile`、`.github/workflows/ci.yml`、`tools/keywords.json`、`web/packages/propel/src/toast/toast.tsx`；`e2e`、`web/apps/web`、`web/packages/{api-client,constants,editor,hooks,i18n,propel,services,shared-state,types,ui,utils}` 的 `package.json`
- Delete: 45 个 stories；`web/packages/ui/{.storybook/main.ts,.storybook/preview.ts,styles/globals.css,postcss.config.js}`；`web/packages/propel/{.storybook/main.ts,.storybook/manager.ts,.storybook/preview.ts,.storybook/tailwind.css,postcss.config.js,public/plane-lockup-light.svg}`；propel 中只被 stories 导入的 5 个文件
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 1、2
- Produces: turbo 任务只剩 `build`、`check:format`、`check:lint`、`check:types`、`dev`、`fix:format`、`test`；`make test-web` = `turbo run test`（持续集成 `web` 任务的 `Unit tests` 一步）；每个包都有 `fix:format`；上限 propel 29、ui 31；守卫规则 `storybook-files`、`storybook`

- [ ] **Step 1: 加入守卫规则，确认它们命中 Storybook**

`tools/keywords.json`，把 `rules` 数组的结尾：

```json
          "observe(value)"
        ]
      }
    }
  ],
```

替换为：

```json
          "observe(value)"
        ]
      }
    },
    {
      "id": "storybook-files",
      "phase": "M1/P1",
      "why": "Storybook 没有 Makefile 或持续集成入口，连同配置、stories 和依赖删除",
      "path": { "source": "(?:^|/)(?:\\.storybook|storybook-static)/|\\.stories\\.[cm]?[jt]sx?$", "flags": "" },
      "samples": {
        "hit": ["web/packages/ui/.storybook/main.ts", "web/packages/propel/src/button/button.stories.tsx"],
        "miss": ["web/packages/propel/src/button/button.tsx", "web/packages/ui/src/stories.ts"]
      }
    },
    {
      "id": "storybook",
      "phase": "M1/P1",
      "why": "Storybook 的依赖、脚本、任务、忽略规则，以及只为它存在的代码",
      "files": {
        "source": "^(?:web/|e2e/|package\\.json$|pnpm-(?:workspace|lock)\\.yaml$|turbo\\.json$|\\.oxlintrc\\.json$|knip\\.jsonc$)",
        "flags": ""
      },
      "content": { "source": "storybook|chromatic", "flags": "i" },
      "samples": {
        "hit": [
          "\"storybook\": \"catalog:\"",
          "\"@chromatic-com/storybook\": \"5.2.1\"",
          "// Static toast for Storybook"
        ],
        "miss": ["\"@tanstack/react-table\": \"catalog:\"", "user story"]
      }
    }
  ],
```

Run: `pnpm exec oxfmt --check tools/ && node tools/keywords.mjs 2>&1 | grep -v '^keywords:' | awk '{print $1}' | sort | uniq -c`
Expected（格式检查通过；锁文件 186 处、`pnpm-workspace.yaml` 19 处、ui 和 propel 的 `package.json`、`.storybook/`、每个 stories 文件、`turbo.json`、`.oxlintrc.json`，以及 `toast.tsx` 的一处注释）:

```
 305 storybook
  51 storybook-files
```

- [ ] **Step 2: 写入 `turbo.json`（完整内容）**

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
    "check:format": {
      "cache": false
    },
    "check:lint": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md", "$TURBO_ROOT$/tools/lint-cap.mjs"],
      "outputs": []
    },
    "check:types": {
      "dependsOn": ["^build"],
      "inputs": ["$TURBO_DEFAULT$", "!**/*.md"],
      "outputs": []
    },
    "dev": {
      "cache": false,
      "dependsOn": ["^build"],
      "persistent": true
    },
    "fix:format": {
      "cache": false
    },
    "test": {
      "dependsOn": ["^build"],
      "outputs": []
    }
  }
}
```

Run: `git diff --numstat turbo.json`
Expected: `0	23	turbo.json`（删掉 `build-storybook`、`check`、`clean`、`fix`、`fix:lint`、`start`）

- [ ] **Step 3: 删除各包没有调用方的脚本，补上 `fix:format`**

```bash
P=/tmp/nerve-p1/pkg.mjs
for d in web/apps/web web/packages/constants web/packages/editor web/packages/hooks web/packages/services web/packages/shared-state web/packages/types web/packages/utils; do
  node $P $d/package.json del-script clean fix:lint
done
node $P web/packages/i18n/package.json del-script clean fix:lint sync:check
node $P web/packages/propel/package.json del-script clean fix:lint storybook build-storybook
node $P web/packages/ui/package.json del-script clean fix:lint storybook build-storybook postcss
node $P web/packages/api-client/package.json set-script fix:format "oxfmt ."
node $P e2e/package.json set-script fix:format "oxfmt ."
```

services 的 `test` 脚本保留（Step 8 的 `make test-web` 调用它）。i18n 的 `check:sync` 暂时不动，Task 9 改写它并接进 `make lint-web`。

- [ ] **Step 4: 删除 Storybook 和 PostCSS 命令行的依赖**

```bash
P=/tmp/nerve-p1/pkg.mjs
node $P web/packages/propel/package.json del-dep @storybook/addon-designs @storybook/addon-docs @storybook/react-vite storybook @plane/tailwind-config
node $P web/packages/ui/package.json del-dep @chromatic-com/storybook @storybook/addon-docs @storybook/addon-links @storybook/addon-onboarding @storybook/addon-styling-webpack @storybook/addon-webpack5-compiler-swc @storybook/react @storybook/react-webpack5 storybook postcss-cli autoprefixer postcss-nested @plane/tailwind-config
```

- [ ] **Step 5: 删除 60 个文件和 `ToastStatic`**

```bash
git rm -q $(git ls-files 'web/packages/ui/src/*.stories.tsx' 'web/packages/propel/src/*.stories.tsx')
git rm -q -r web/packages/ui/.storybook web/packages/propel/.storybook
git rm -q web/packages/ui/styles/globals.css web/packages/ui/postcss.config.js web/packages/propel/postcss.config.js web/packages/propel/public/plane-lockup-light.svg
git rm -q web/packages/propel/src/empty-state/assets/horizontal-stack/constant.tsx web/packages/propel/src/empty-state/assets/illustration/constant.tsx web/packages/propel/src/empty-state/assets/vertical-stack/constant.tsx web/packages/propel/src/icons/constants.tsx web/packages/propel/src/separator/separator.tsx
```

Run: `git status --short | grep -c '^D '`
Expected: `60`（45 个 stories、6 个 `.storybook/` 文件、4 个 PostCSS 和样式文件、5 个只被 stories 导入的文件）

`ToastStatic` 的注释写明它"for Storybook and documentation"，没有其他调用方。删掉从这行注释到 `setToast` 之前的整段（类型 `ToastStaticProps` 和组件 `ToastStatic`，53 行加一个空行）：

```bash
node -e '
const fs = require("fs");
const file = "web/packages/propel/src/toast/toast.tsx";
const text = fs.readFileSync(file, "utf8");
const start = text.indexOf("// Static toast component for Storybook and documentation\n");
const end = text.indexOf("export const setToast = ");
if (start < 0 || end < start) throw new Error("markers not found");
fs.writeFileSync(file, text.slice(0, start) + text.slice(end));
'
```

Run: `git diff --numstat web/packages/propel/src/toast/toast.tsx; grep -c 'ToastStatic' web/packages/propel/src/toast/toast.tsx`
Expected:

```
0	54	web/packages/propel/src/toast/toast.tsx
0
```

- [ ] **Step 6: 修改 `.oxlintrc.json`（删掉 Storybook 的两条忽略规则）**

把：

```json
    ".react-router/**",
    ".storybook/**",
    ".turbo/**",
```

替换为：

```json
    ".react-router/**",
    ".turbo/**",
```

把：

```json
    "**/public/**",
    "storybook-static/**",
    "web/packages/api-client/src/schema.gen.ts"
```

替换为：

```json
    "**/public/**",
    "web/packages/api-client/src/schema.gen.ts"
```

- [ ] **Step 7: 修改 `pnpm-workspace.yaml`（删掉 33 行）**

把下面 33 行原样写入 `/tmp/nerve-p1/t3-drop.txt`（每一行在 `pnpm-workspace.yaml` 中只出现一次；文件中不要有空行）。依次是：catalog 中只给 Storybook 和 PostCSS 命令行用的 14 条；删掉 Storybook 之后不再作用于任何包的 overrides：`webpack`、`minimatch@3`、`ajv@6`、`ajv@8`、`flatted`、`fast-uri`（连同它的 5 行注释）；allowBuilds 的 `@swc/core`；minimumReleaseAgeExclude 中的 7 个 Storybook 包（spec 2.6；Step 9 的 `unused-entries.mjs` 在删之前就会逐条报出它们）。`vitest` 的 catalog 条目保留。

```
  "@chromatic-com/storybook": "5.2.1"
  "@storybook/addon-designs": "11.1.3"
  "@storybook/addon-docs": "10.4.6"
  "@storybook/addon-links": "10.4.6"
  "@storybook/addon-onboarding": "10.4.6"
  "@storybook/addon-styling-webpack": "3.0.2"
  "@storybook/addon-webpack5-compiler-swc": "4.0.3"
  "@storybook/react": "10.4.6"
  "@storybook/react-vite": "10.4.6"
  "@storybook/react-webpack5": "10.4.6"
  "autoprefixer": "^10.4.19"
  "postcss-cli": "^11.0.0"
  "postcss-nested": "^6.0.1"
  "storybook": "10.4.6"
  webpack: 5.104.1
  "minimatch@3": 3.1.4
  "ajv@6": 6.14.0
  "ajv@8": 8.18.0
  flatted: 3.4.2
  # SSRF / host-confusion in the URI parser (percent-decoding, IDN
  # canonicalization and IPv6 normalization): CVE-2026-75899, -75931, -75975 and
  # -76172, all fixed in 3.1.6. The previous floor pinned 3.1.5, which is the
  # affected version. fast-uri reaches us only through ajv, so stay on the
  # patched 3.x rather than 4.x, which is outside ajv's ^3.0.1 range.
  fast-uri: 3.1.6
  "@swc/core": true
  - "@storybook/addon-docs@10.4.6"
  - "@storybook/builder-vite@10.4.6"
  - "@storybook/csf-plugin@10.4.6"
  - "@storybook/react-dom-shim@10.4.6"
  - "@storybook/react-vite@10.4.6"
  - "@storybook/react@10.4.6"
  - storybook@10.4.6
```

```bash
grep -vxF -f /tmp/nerve-p1/t3-drop.txt pnpm-workspace.yaml > /tmp/nerve-p1/workspace.yaml && cp /tmp/nerve-p1/workspace.yaml pnpm-workspace.yaml
```

Run: `git diff --numstat pnpm-workspace.yaml`
Expected: `0	33	pnpm-workspace.yaml`

- [ ] **Step 8: 前端单元测试的入口：`make test-web` 和持续集成**

`Makefile`，把：

```makefile
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...
```

替换为：

```makefile
test: ## 运行 Go 测试（server，不用测试缓存）
	cd server && go test -count=1 ./...

.PHONY: test-web
test-web: ## 运行前端单元测试（各包 test 脚本中的 vitest，经 turbo；需要 Node；持续集成 web 任务）
	$(TURBO) run test $(TURBO_QUIET)
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected:

```
34
23
```

`.github/workflows/ci.yml`（`web` 任务），把：

```yaml
      - name: Lint
        run: make lint-web
```

替换为：

```yaml
      - name: Lint
        run: make lint-web
      - name: Unit tests
        run: make test-web
```

- [ ] **Step 9: 安装，核对锁文件**

Run: `pnpm install 2>&1 | tail -1`
Expected: `Done in …`

按 Global Constraints 执行锁文件核对。Expected:

```
catalogs: 42 lines removed, 0 added
overrides: 6 lines removed, 0 added
importers: 18 dependencies removed, 1 added or changed
  ~ web/packages/constants devDependencies tsdown: - version: 0.16.0(@emnapi/core@1.10.0)(@emnapi/runtime@1.11.3)(oxc-resolver@11.20.0)(supports-color@10.2.2)(typescript@5.8.3)  + version: 0.16.0(@emnapi/core@1.10.0)(@emnapi/runtime@1.11.3)(oxc-resolver@11.24.2)(supports-color@10.2.2)(typescript@5.8.3)
packages: 300 removed, 0 added or changed
snapshots: 308 removed, 11 added or changed
  + rolldown-plugin-dts@0.17.8(oxc-resolver@11.24.2)(rolldown@1.0.0-beta.46(@emnapi/core@1.10.0)(@emnapi/runtime@1.11.3))(typescript@5.8.3)
  + tsdown@0.16.0(@emnapi/core@1.10.0)(@emnapi/runtime@1.11.3)(oxc-resolver@11.24.2)(supports-color@10.2.2)(typescript@5.8.3)
  ~ @jridgewell/source-map@0.3.11: + optional: true
  ~ acorn@8.16.0: + optional: true
  ~ buffer-from@1.1.2: + optional: true
  ~ commander@2.20.3: + optional: true
  ~ has-flag@4.0.0: + optional: true
  ~ range-parser@1.2.1: + optional: true
  ~ source-map-support@0.5.21: + optional: true
  ~ supports-color@7.2.0: + optional: true
  ~ terser@5.43.1: + optional: true
   11903
1ba45d3fdb8b61723c2fa3dc8af1e32b3af64ca91552d4ea8630ef650086a614  pnpm-lock.yaml
```

（`oxc-resolver@11.20.0` 是 Storybook 带进来的，随之删除，constants 的 `tsdown` 改用其余 9 个包已经在用的 11.24.2；9 个包原来经 webpack 可达，现在只经 Vite 的可选对等依赖可达，spec 2.6。）然后执行 `pnpm install --frozen-lockfile 2>&1 | tail -1`、`node /tmp/nerve-p1/unused-entries.mjs`、`pnpm peers check`，结果见 Global Constraints。

- [ ] **Step 10: 调低 propel、ui 的上限**

先确认检查会要求调低（stories 带走了 propel 的 30 条、ui 的 1 条警告）：

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:lint --filter=@plane/propel --filter=@plane/ui --continue --output-logs=errors-only 2>&1 | grep -oE '@plane/[a-z]+: [0-9]+ oxlint warnings, fewer than the cap of [0-9]+' | sort -u`
Expected:

```
@plane/propel: 29 oxlint warnings, fewer than the cap of 59
@plane/ui: 31 oxlint warnings, fewer than the cap of 32
```

```bash
node /tmp/nerve-p1/pkg.mjs web/packages/propel/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 29"
node /tmp/nerve-p1/pkg.mjs web/packages/ui/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 31"
```

- [ ] **Step 11: 收尾检查**

- `node tools/keywords.mjs`：`keywords: 5 rules, 0 exceptions, no hits.`；
- `make build-web`：11 个任务；
- `make lint-web`：49 个任务；
- `make test-web`：`Tasks:    11 successful, 11 total`（services 的测试和它依赖的 10 个包的构建）。`TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run test --output-logs=full 2>&1 | perl -pe 's/\e\[[0-9;]*m//g' | grep -E ':test: +Tests '` 输出 `@plane/services:test:       Tests  13 passed (13)`；
- `make knip`：

```
Unused files (128)
Unused dependencies (19)
Unused devDependencies (1)
Unlisted dependencies (1)
Unused exports (135)
Unused exported types (82)
Unused exported enum members (4)
Duplicate exports (1)
```

（外加 `Configuration hints`。剩下的开发依赖是 types 的 `@types/react-dom`，未列出的依赖是 editor 的 `postcss.config.js` 中的 `@tailwindcss/postcss`，Task 4 处理。）

- `git status --short | cut -c1-2 | sort | uniq -c`：

```
  21  M
  60 D 
```

- [ ] **Step 12: 提交**

```bash
git add -u
git commit -m "chore(web): drop turbo tasks and scripts without callers and Storybook; run unit tests in CI

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: 未使用的依赖、只为它们存在的文件；Next.js 垫片中的死文件

**Files:**
- Modify: `web/apps/web/package.json`、`web/apps/web/vite.config.ts`、`web/packages/{editor,propel,shared-state,types,ui}/package.json`、`pnpm-workspace.yaml`、`tools/keywords.json`
- Delete: `web/apps/web/core/components/ui/markdown-to-component.tsx`、`web/apps/web/use-font-face-observer.d.ts`、`web/apps/web/app/compat/next/image.tsx`、`web/apps/web/app/compat/next/script.tsx`、`web/apps/web/app/types/next-script.d.ts`、`web/packages/editor/postcss.config.js`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 3
- Produces: knip 的依赖三类只剩 editor 的 `buffer`；垫片只剩 `link.tsx`、`navigation.ts`（P4 删除）；web 的上限 777；守卫规则 `next-shim-dead-files`、`next-script-image`

- [ ] **Step 1: 加入守卫规则，确认它们命中垫片的死文件**

`tools/keywords.json`，把 `rules` 数组的结尾：

```json
        "miss": ["\"@tanstack/react-table\": \"catalog:\"", "user story"]
      }
    }
  ],
```

替换为：

```json
        "miss": ["\"@tanstack/react-table\": \"catalog:\"", "user story"]
      }
    },
    {
      "id": "next-shim-dead-files",
      "phase": "M1/P1",
      "why": "Next.js 垫片中没人用的 image、script 及 next/script 的类型声明；其余垫片在 M1/P4 删除",
      "path": {
        "source": "^web/apps/web/app/(?:compat/next/(?:image|script)\\.tsx?|types/next-(?:image|script)\\.d\\.ts)$",
        "flags": ""
      },
      "samples": {
        "hit": ["web/apps/web/app/compat/next/image.tsx", "web/apps/web/app/types/next-script.d.ts"],
        "miss": ["web/apps/web/app/compat/next/link.tsx", "web/apps/web/app/types/next-link.d.ts"]
      }
    },
    {
      "id": "next-script-image",
      "phase": "M1/P1",
      "why": "next/script、next/image 的导入、别名和模块声明；next/link、next/navigation 在 M1/P4 删除",
      "files": { "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$", "flags": "" },
      "content": { "source": "[\"'`]next/(?:script|image)[\"'`]", "flags": "" },
      "samples": {
        "hit": [
          "import Script from \"next/script\";",
          "      \"next/script\": path.resolve(__dirname, \"app/compat/next/script.tsx\"),"
        ],
        "miss": ["import Link from \"next/link\";", "// Minimal shim so code using next/image compiles"]
      }
    }
  ],
```

Run: `pnpm exec oxfmt --check tools/ && node tools/keywords.mjs 2>&1 | grep -v '^keywords:'`
Expected（格式检查通过）:

```
next-shim-dead-files  web/apps/web/app/compat/next/image.tsx  "web/apps/web/app/compat/next/image.tsx"
next-shim-dead-files  web/apps/web/app/compat/next/script.tsx  "web/apps/web/app/compat/next/script.tsx"
next-shim-dead-files  web/apps/web/app/types/next-script.d.ts  "web/apps/web/app/types/next-script.d.ts"
next-script-image  web/apps/web/app/types/next-script.d.ts:1  "\"next/script\""
next-script-image  web/apps/web/vite.config.ts:30  "\"next/script\""
```

- [ ] **Step 2: 删除依赖**

```bash
P=/tmp/nerve-p1/pkg.mjs
node $P web/apps/web/package.json del-dep clsx comlink emoji-picker-react react-fast-compare react-is react-markdown recharts use-font-face-observer
node $P web/packages/editor/package.json del-dep @tiptap/extension-list-item @plane/tailwind-config postcss
node $P web/packages/propel/package.json del-dep @plane/hooks @plane/utils @tanstack/react-table
node $P web/packages/shared-state/package.json del-dep zod
node $P web/packages/ui/package.json del-dep clsx lucide-react react-day-picker tailwind-merge use-font-face-observer
node $P web/packages/types/package.json del-dep @types/react-dom
```

editor 的 `@plane/tailwind-config`、`postcss` 只给它的 `postcss.config.js` 用；那个文件下一步删除（它引用的 `@tailwindcss/postcss` 是 knip 报出的未列出依赖；删掉它，editor 构建出的样式表逐字节相同，spec 2.2）。`buffer` 保留（spec 2.8）。

- [ ] **Step 3: 删除 6 个文件**

```bash
git rm -q web/apps/web/core/components/ui/markdown-to-component.tsx web/apps/web/use-font-face-observer.d.ts web/apps/web/app/compat/next/image.tsx web/apps/web/app/compat/next/script.tsx web/apps/web/app/types/next-script.d.ts web/packages/editor/postcss.config.js
```

- [ ] **Step 4: 修改 `web/apps/web/vite.config.ts`（删掉 `next/script` 的别名）**

把：

```ts
      "next/navigation": path.resolve(__dirname, "app/compat/next/navigation.ts"),
      "next/script": path.resolve(__dirname, "app/compat/next/script.tsx"),
```

替换为：

```ts
      "next/navigation": path.resolve(__dirname, "app/compat/next/navigation.ts"),
```

- [ ] **Step 5: 修改 `pnpm-workspace.yaml`（删掉 6 个 catalog 条目）**

把下面 6 行写入 `/tmp/nerve-p1/t4-drop.txt`：

```
  "@tiptap/extension-list-item": "^2.22.3"
  "comlink": "^4.4.1"
  "emoji-picker-react": "^4.5.16"
  "react-fast-compare": "^3.2.2"
  "react-markdown": "^10.1.0"
  "zod": "^3.25.76"
```

```bash
grep -vxF -f /tmp/nerve-p1/t4-drop.txt pnpm-workspace.yaml > /tmp/nerve-p1/workspace.yaml && cp /tmp/nerve-p1/workspace.yaml pnpm-workspace.yaml
```

Run: `git diff --numstat pnpm-workspace.yaml`
Expected: `0	6	pnpm-workspace.yaml`

其他删掉的依赖（`clsx`、`react-is`、`recharts`、`lucide-react` 等）仍被别的包或覆盖项使用，catalog 条目保留；下一步的脚本会核对。

- [ ] **Step 6: 安装，核对锁文件**

Run: `pnpm install 2>&1 | tail -1`
Expected: `Done in …`

按 Global Constraints 执行锁文件核对。Expected:

```
catalogs: 18 lines removed, 0 added
importers: 21 dependencies removed, 1 added or changed
  ~ web/packages/utils dependencies remark-gfm: - version: 4.0.1  + version: 4.0.1(supports-color@10.2.2)
packages: 23 removed, 0 added or changed
snapshots: 27 removed, 9 added or changed
  + mdast-util-from-markdown@2.0.2(supports-color@10.2.2)
  + mdast-util-gfm@3.1.0(supports-color@10.2.2)
  + micromark@4.0.2(supports-color@10.2.2)
  + remark-gfm@4.0.1(supports-color@10.2.2)
  ~ mdast-util-gfm-footnote@2.1.0: - mdast-util-from-markdown: 2.0.2  + mdast-util-from-markdown: 2.0.2(supports-color@10.2.2)
  ~ mdast-util-gfm-strikethrough@2.0.0: - mdast-util-from-markdown: 2.0.2  + mdast-util-from-markdown: 2.0.2(supports-color@10.2.2)
  ~ mdast-util-gfm-table@2.0.0: - mdast-util-from-markdown: 2.0.2  + mdast-util-from-markdown: 2.0.2(supports-color@10.2.2)
  ~ mdast-util-gfm-task-list-item@2.0.0: - mdast-util-from-markdown: 2.0.2  + mdast-util-from-markdown: 2.0.2(supports-color@10.2.2)
  ~ remark-parse@11.0.0: - mdast-util-from-markdown: 2.0.2  + mdast-util-from-markdown: 2.0.2(supports-color@10.2.2)
   11608
555441ed03625a163fda5e0bed75594190c191785c18a17ea95c2d0bd3b7c491  pnpm-lock.yaml
```

（utils 的 `remark-gfm` 带上对等后缀 `(supports-color@10.2.2)`，与 P6 中其他依赖 `debug` 的包一致；这一支的快照随之带上同一后缀，spec 2.6。）然后执行 `pnpm install --frozen-lockfile 2>&1 | tail -1`、`node /tmp/nerve-p1/unused-entries.mjs`、`pnpm peers check`，结果见 Global Constraints。

- [ ] **Step 7: 调低 web 的上限（779 → 777）**

删掉的死文件带走了 web 的 2 条警告。`make lint-web` 会报 `web: 777 oxlint warnings, fewer than the cap of 779`。

```bash
node /tmp/nerve-p1/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 777"
```

- [ ] **Step 8: 收尾检查**

- `node tools/keywords.mjs`：`keywords: 7 rules, 0 exceptions, no hits.`；
- `make build-web`：11 个任务；
- `make lint-web`：49 个任务；
- `make test-web`：11 个任务；
- `make knip`（不再有 `Unused devDependencies`、`Unlisted dependencies` 两类）：

```
Unused files (125)
Unused dependencies (1)
Unused exports (135)
Unused exported types (82)
Unused exported enum members (4)
Duplicate exports (1)
```

`make knip 2>&1 | grep -A1 '^Unused dependencies'` 的第二行是 `buffer  web/packages/editor/package.json:…`。

- `git status --short | cut -c1-2 | sort | uniq -c`：

```
  10  M
   6 D 
```

- [ ] **Step 9: 提交**

```bash
git add -u
git commit -m "chore(web): remove unused dependencies and the dead Next.js shim files

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: 修剪 `packages/services`

**Files:**
- Delete: `web/packages/services/` 中的 49 个文件
- Modify: `web/packages/services/src/index.ts`、`src/developer/index.ts`、`src/file/index.ts`、`web/packages/services/package.json`、`tools/keywords.json`

**Interfaces:**
- Consumes: web 对 `@plane/services` 的导入：`APITokenService`、`helpers`（地址规范化）、`file/helper`（上传元数据）
- Produces: services 只剩 12 个文件，上限 0，`make test-web` 的 13 个地址测试照常通过；守卫规则 `services-files`；包本身留到 M5（M1 设计 3.5）

- [ ] **Step 1: 加入守卫规则，确认它命中要删的 49 个文件**

`tools/keywords.json`，把 `rules` 数组的结尾：

```json
        "miss": ["import Link from \"next/link\";", "// Minimal shim so code using next/image compiles"]
      }
    }
  ],
```

替换为：

```json
        "miss": ["import Link from \"next/link\";", "// Minimal shim so code using next/image compiles"]
      }
    },
    {
      "id": "services-files",
      "phase": "M1/P1",
      "why": "packages/services 只留 web 用到的令牌服务、地址工具和上传元数据工具，整包在 M5 删除；新的接口调用用生成的客户端",
      "path": {
        "source": "^web/packages/services/(?!(?:package\\.json|tsconfig\\.json|tsdown\\.config\\.ts|src/(?:index|api\\.service)\\.ts|src/helpers/(?:index|url|url\\.test)\\.ts|src/developer/(?:index|api-token\\.service)\\.ts|src/file/(?:index|helper)\\.ts)$)",
        "flags": ""
      },
      "samples": {
        "hit": ["web/packages/services/src/issue/issue.service.ts", "web/packages/services/src/live.service.ts"],
        "miss": [
          "web/packages/services/src/helpers/url.test.ts",
          "web/packages/services/src/developer/api-token.service.ts"
        ]
      }
    }
  ],
```

Run: `pnpm exec oxfmt --check tools/ && node tools/keywords.mjs 2>&1 | grep -v '^keywords:' | awk '{print $1}' | sort | uniq -c`
Expected（格式检查通过）:

```
  49 services-files
```

- [ ] **Step 2: 列出要删的文件并删除**

```bash
git ls-files web/packages/services \
  | grep -vxE 'web/packages/services/(package\.json|tsconfig\.json|tsdown\.config\.ts|src/index\.ts|src/api\.service\.ts|src/helpers/(index|url|url\.test)\.ts|src/developer/(index|api-token\.service)\.ts|src/file/(index|helper)\.ts)' \
  > /tmp/nerve-p1/services-delete.txt
wc -l < /tmp/nerve-p1/services-delete.txt
```

Expected: `      49`（`ai/`、`auth/`、`cycle/`、`dashboard/`、`instance/`、`intake/`、`issue/`、`label/`、`module/`、`project/`、`state/`、`user/`、`workspace/` 整个目录，`developer/webhook.service.ts`，`file/` 中的 `file-upload.service.ts`、`file.service.ts`、`sites-file.service.ts`，`indexedDB.service.ts`、`live.service.ts`；与 Step 1 守卫报出的 49 个相同）

```bash
git rm -q --pathspec-from-file=/tmp/nerve-p1/services-delete.txt
```

- [ ] **Step 3: 写入 `web/packages/services/src/index.ts`（完整内容）**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export * from "./developer";
export * from "./file";
export * from "./helpers";
```

- [ ] **Step 4: 写入 `web/packages/services/src/developer/index.ts`（完整内容）**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export * from "./api-token.service";
```

- [ ] **Step 5: 写入 `web/packages/services/src/file/index.ts`（完整内容）**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export * from "./helper";
```

- [ ] **Step 6: services 的上限 6 → 0**

```bash
node /tmp/nerve-p1/pkg.mjs web/packages/services/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 0"
```

Run: `git ls-files web/packages/services`
Expected:

```
web/packages/services/package.json
web/packages/services/src/api.service.ts
web/packages/services/src/developer/api-token.service.ts
web/packages/services/src/developer/index.ts
web/packages/services/src/file/helper.ts
web/packages/services/src/file/index.ts
web/packages/services/src/helpers/index.ts
web/packages/services/src/helpers/url.test.ts
web/packages/services/src/helpers/url.ts
web/packages/services/src/index.ts
web/packages/services/tsconfig.json
web/packages/services/tsdown.config.ts
```

- [ ] **Step 7: 收尾检查**

- `node tools/keywords.mjs`：`keywords: 8 rules, 0 exceptions, no hits.`；
- `make build-web`：11 个任务；
- `make lint-web`：49 个任务；
- `make test-web`：11 个任务（13 个地址测试照常通过）；
- `make knip`：与 Task 4 相比只有 `Unused files (123)`（services 的 `indexedDB.service.ts`、`live.service.ts` 不再出现）；
- `git status --short | cut -c1-2 | sort | uniq -c`：

```
   5  M
  49 D 
```

- [ ] **Step 8: 提交**

```bash
git add -u
git commit -m "refactor(services): keep only the files web uses

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: 两条构建警告；每个包自己的增量编译文件

**Files:**
- Modify: `web/packages/tailwind-config/package.json`、`web/apps/web/vite.config.ts`、`web/apps/web/package.json`、`pnpm-workspace.yaml`、`web/packages/typescript-config/base.json`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 5
- Produces: 构建只剩 Plane 自己的"代码块大于 500 kB"提示；web 和继承 `base.json` 的 10 个包各写 `<包目录>/.turbo/tsconfig.tsbuildinfo`；knip 不再有 tailwind-config 的提示

- [ ] **Step 1: 写入 `web/packages/tailwind-config/package.json`（完整内容）**

加 `"type": "module"`（它的 `postcss.config.js` 是 ES 模块）；删掉指向不存在文件的 `"main": "tailwind.config.js"`：包没有 JS 入口，真实的入口 `./index.css`、`./postcss.config.js` 已经由 `exports` 声明（spec 2.10）。

```json
{
  "name": "@plane/tailwind-config",
  "version": "1.4.2",
  "private": true,
  "description": "common tailwind configuration across monorepo",
  "license": "AGPL-3.0",
  "type": "module",
  "sideEffects": [
    "**/*.css"
  ],
  "exports": {
    "./index.css": "./index.css",
    "./postcss.config.js": "./postcss.config.js"
  },
  "dependencies": {
    "@makeplane/propel": "catalog:",
    "@tailwindcss/postcss": "catalog:",
    "postcss": "catalog:"
  },
  "devDependencies": {
    "tailwindcss": "catalog:"
  }
}
```

- [ ] **Step 2: 修改 `web/apps/web/vite.config.ts`（Vite 8 内置的 tsconfig 路径解析）**

把：

```ts
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";
```

替换为：

```ts
import { defineConfig } from "vite";
```

把：

```ts
  plugins: [reactRouter(), tsconfigPaths({ projects: [path.resolve(__dirname, "tsconfig.json")] })],
  resolve: {
    alias: {
```

替换为：

```ts
  plugins: [reactRouter()],
  resolve: {
    tsconfigPaths: true,
    alias: {
```

- [ ] **Step 3: 删除 `vite-tsconfig-paths` 依赖和 catalog 条目**

```bash
node /tmp/nerve-p1/pkg.mjs web/apps/web/package.json del-dep vite-tsconfig-paths
grep -vxF -e '  "vite-tsconfig-paths": "^5.1.4"' pnpm-workspace.yaml > /tmp/nerve-p1/workspace.yaml && cp /tmp/nerve-p1/workspace.yaml pnpm-workspace.yaml
```

Run: `git diff --numstat pnpm-workspace.yaml`
Expected: `0	1	pnpm-workspace.yaml`

- [ ] **Step 4: 修改 `web/packages/typescript-config/base.json`**

`${configDir}` 是继承链最末端的 `tsconfig.json`（各包自己的）所在的目录，TypeScript 5.5 起支持。

把：

```json
    "tsBuildInfoFile": ".turbo/tsconfig.tsbuildinfo",
```

替换为：

```json
    "tsBuildInfoFile": "${configDir}/.turbo/tsconfig.tsbuildinfo",
```

- [ ] **Step 5: 安装，核对锁文件**

Run: `pnpm install 2>&1 | tail -1`
Expected: `Done in …`

按 Global Constraints 执行锁文件核对。Expected:

```
catalogs: 3 lines removed, 0 added
importers: 1 dependencies removed, 0 added or changed
packages: 3 removed, 0 added or changed
snapshots: 3 removed, 0 added or changed
   11563
b84db26a73be95b316b77f97ff56ce536f1c1dcdcb81ff3ea8eefedadbc0b8cd  pnpm-lock.yaml
```

然后执行 `pnpm install --frozen-lockfile 2>&1 | tail -1`、`node /tmp/nerve-p1/unused-entries.mjs`、`pnpm peers check`，结果见 Global Constraints。

- [ ] **Step 6: 确认两条构建警告消失**

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run build --filter=web --output-logs=full --force > /tmp/nerve-p1/build.log 2>&1; grep -cE 'MODULE_TYPELESS_PACKAGE_JSON|vite-tsconfig-paths' /tmp/nerve-p1/build.log; grep -c 'Some chunks are larger than 500 kB' /tmp/nerve-p1/build.log`
Expected:

```
0
1
```

（第二个是 Plane 自己的提示，保留。）

- [ ] **Step 7: 确认每个包写自己的增量编译文件**

改之前所有包共用 `web/packages/typescript-config/.turbo/tsconfig.tsbuildinfo`，先删掉这个旧文件；`typescript-config` 没有 `build` 任务，改它不会让其他包的 `check:types` 缓存失效，所以这里用 `--force`：

```bash
rm -f web/packages/typescript-config/.turbo/tsconfig.tsbuildinfo
TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:types --force --output-logs=errors-only 2>&1 | grep 'Tasks:'
ls web/apps/web/.turbo/tsconfig.tsbuildinfo web/packages/*/.turbo/tsconfig.tsbuildinfo | wc -l
ls web/packages/typescript-config/.turbo/tsconfig.tsbuildinfo
```

Expected:

```
 Tasks:    23 successful, 23 total
      11
ls: web/packages/typescript-config/.turbo/tsconfig.tsbuildinfo: No such file or directory
```

（11 个是 web 和继承 `base.json` 的 10 个包；api-client、e2e 不继承它。`.turbo/` 已在 `.gitignore` 中。）

- [ ] **Step 8: 收尾检查**

- `make build-web`：11 个任务；
- `make lint-web`：守卫 8 条规则没有命中，49 个任务；
- `make test-web`：11 个任务；
- `make knip`：条数与 Task 5 相同；`Configuration hints` 中不再有 `tailwind.config.js … Package entry file not found`；
- `git status --short | cut -c1-2 | sort | uniq -c`：

```
   6  M
```

（`pnpm-lock.yaml`、`pnpm-workspace.yaml`、`web/apps/web/package.json`、`web/apps/web/vite.config.ts`、`web/packages/tailwind-config/package.json`、`web/packages/typescript-config/base.json`。）

- [ ] **Step 9: 提交**

```bash
git add -u
git commit -m "build(web): fix the two build warnings and give each package its own tsbuildinfo

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: React #418

**Files:**
- Modify: `web/apps/web/app/root.tsx`、`web/apps/web/core/components/common/logo-spinner.tsx`

**Interfaces:**
- Consumes: Task 6；`e2e/` 中的 Playwright 和它的 Chromium（没有装过时执行 `pnpm --dir e2e exec playwright install chromium`，README"端到端测试"一节）
- Produces: 预渲染的 `index.html` 与首次渲染的标记相同；`LogoSpinner` 用 CSS 选图片；`/tmp/nerve-p1/probe.sh`（Task 8 也用它）

- [ ] **Step 1: 写入一次性脚本**

`/tmp/nerve-p1/spa-serve.mjs`（在空闲端口上提供构建产物，打印端口；`/api/*` 像 nerve 一样返回 404）：

```js
// One-off (M1/P1 Task 7): serves a built SPA on 127.0.0.1 at a free port and prints the port.
// Files that exist are served, every other path gets index.html; /api/* answers 404 like nerve.
// usage: node spa-serve.mjs <dir>
import fs from "node:fs";
import http from "node:http";
import path from "node:path";

const root = path.resolve(process.argv[2]);
const types = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".png": "image/png",
  ".gif": "image/gif",
  ".svg": "image/svg+xml",
  ".webp": "image/webp",
  ".woff2": "font/woff2",
  ".ico": "image/x-icon",
};
const server = http.createServer((req, res) => {
  const { pathname } = new URL(req.url, "http://localhost");
  if (pathname.startsWith("/api/")) {
    res.writeHead(404, { "content-type": "application/problem+json" });
    res.end('{"title":"Not Found","status":404}');
    return;
  }
  let file = path.join(root, decodeURIComponent(pathname));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
    file = path.join(root, "index.html");
  }
  res.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream" });
  fs.createReadStream(file).pipe(res);
});
server.listen(0, "127.0.0.1", () => console.log(server.address().port));
```

`/tmp/nerve-p1/hydration-probe.mjs`（亮色、暗色各打开几个页面，报告有没有 #418；再拦截应用脚本，看预渲染的页面显示哪张加载动画）：

```js
// One-off (M1/P1 Task 7): opens pages of the served SPA in light and dark mode and reports React
// error #418 (hydration mismatch); then blocks the app's scripts to see which spinner image the
// prerendered index.html shows. Run from the repository root (Playwright comes from e2e/).
// usage: node hydration-probe.mjs <port> <path>...
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const [port, ...paths] = process.argv.slice(2);
const base = `http://127.0.0.1:${port}`;
const browser = await chromium.launch();
try {
  for (const colorScheme of ["light", "dark"]) {
    for (const p of paths) {
      const context = await browser.newContext({ colorScheme });
      const page = await context.newPage();
      const messages = [];
      page.on("console", (m) => messages.push(m.text()));
      page.on("pageerror", (e) => messages.push(e.message));
      await page.goto(base + p, { waitUntil: "networkidle" });
      await page.waitForTimeout(1500);
      console.log(`[${colorScheme}] ${p}: #418 ${messages.some((m) => m.includes("#418"))}`);
      await context.close();
    }
    const context = await browser.newContext({ colorScheme });
    const page = await context.newPage();
    await page.route("**/assets/*.js", (route) => route.abort());
    await page.goto(`${base}/`, { waitUntil: "load" });
    const theme = await page.evaluate(() => document.documentElement.getAttribute("data-theme"));
    const visible = await page.$$eval("img", (images) =>
      images.filter((i) => getComputedStyle(i).display !== "none").map((i) => i.getAttribute("src").replace(/-[\w-]{8}\.gif$/, ".gif"))
    );
    console.log(`[${colorScheme}] prerendered index.html: data-theme=${theme}, visible images ${JSON.stringify(visible)}`);
    await context.close();
  }
} finally {
  await browser.close();
}
```

`/tmp/nerve-p1/probe.sh`（启动静态服务器，运行给出的探测脚本，停止服务器并确认端口已释放）：

```bash
#!/bin/bash
# One-off (M1/P1 Tasks 7, 8): serve web/apps/web/build/client, run one probe against it, stop the server.
# usage: bash /tmp/nerve-p1/probe.sh <probe.mjs> [args...]   (from the repository root)
set -uo pipefail
: > /tmp/nerve-p1/serve.port
node /tmp/nerve-p1/spa-serve.mjs web/apps/web/build/client > /tmp/nerve-p1/serve.port &
SERVER=$!
for _ in $(seq 1 100); do [ -s /tmp/nerve-p1/serve.port ] && break; sleep 0.2; done
PORT=$(head -1 /tmp/nerve-p1/serve.port)
node "$1" "$PORT" "${@:2}"
kill "$SERVER"; wait "$SERVER" 2>/dev/null
lsof -nP -iTCP:"$PORT" -sTCP:LISTEN || echo "port $PORT is free"
```

- [ ] **Step 2: 修复之前，确认 #418 存在**

Run: `make build-web > /dev/null && bash /tmp/nerve-p1/probe.sh /tmp/nerve-p1/hydration-probe.mjs / /acme/projects/x/issues`
Expected（预渲染的 `HydrateFallback` 是空的 `<div />`，所以看不到任何图片）:

```
[light] /: #418 true
[light] /acme/projects/x/issues: #418 true
[light] prerendered index.html: data-theme=light, visible images []
[dark] /: #418 true
[dark] /acme/projects/x/issues: #418 true
[dark] prerendered index.html: data-theme=dark, visible images []
port <端口> is free
```

- [ ] **Step 3: 修改 `web/apps/web/app/root.tsx`**

把：

```tsx
import { ThemeProvider, useTheme } from "next-themes";
```

替换为：

```tsx
import { ThemeProvider } from "next-themes";
```

把：

```tsx
export function HydrateFallback() {
  const { resolvedTheme } = useTheme();

  // if we are on the server or the theme is not resolved, return an empty div
  if (typeof window === "undefined" || resolvedTheme === undefined) return <div />;

  return (
```

替换为：

```tsx
// Prerendered into index.html, so it must render the same markup on the server and on the
// first client render: nothing here may depend on the window or the resolved theme.
export function HydrateFallback() {
  return (
```

- [ ] **Step 4: 写入 `web/apps/web/core/components/common/logo-spinner.tsx`（完整内容）**

```tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// assets
import LogoSpinnerDark from "@/app/assets/images/logo-spinner-dark.gif?url";
import LogoSpinnerLight from "@/app/assets/images/logo-spinner-light.gif?url";

// The theme picks the image through CSS (the dark variant keys off the data-theme attribute that
// next-themes sets before the first paint), so the markup does not depend on the resolved theme and
// hydrates cleanly where it is prerendered (HydrateFallback in app/root.tsx).
export function LogoSpinner() {
  return (
    <div className="flex items-center justify-center">
      <img src={LogoSpinnerLight} alt="logo" className="h-6 w-auto object-contain sm:h-11 dark:hidden" />
      <img src={LogoSpinnerDark} alt="logo" className="hidden h-6 w-auto object-contain sm:h-11 dark:block" />
    </div>
  );
}
```

- [ ] **Step 5: 修复之后再跑一次**

Run: `make build-web > /dev/null && bash /tmp/nerve-p1/probe.sh /tmp/nerve-p1/hydration-probe.mjs / /acme/projects/x/issues`
Expected:

```
[light] /: #418 false
[light] /acme/projects/x/issues: #418 false
[light] prerendered index.html: data-theme=light, visible images ["/assets/logo-spinner-light.gif"]
[dark] /: #418 false
[dark] /acme/projects/x/issues: #418 false
[dark] prerendered index.html: data-theme=dark, visible images ["/assets/logo-spinner-dark.gif"]
port <端口> is free
```

修复前后两次的输出和脚本全文写进 P1 review 的附录（M1 设计 7.5）。

- [ ] **Step 6: 收尾检查**

- `make build-web`：11 个任务；
- `make lint-web`：49 个任务（web 的上限仍是 777）；
- `make test-web`：11 个任务；
- `make knip`：与 Task 6 相同；
- `git status --short | cut -c1-2 | sort | uniq -c`：

```
   2  M
```

- [ ] **Step 7: 提交**

```bash
git add -u
git commit -m "fix(web): hydrate the prerendered page without React error #418

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: 只留中英文，不支持的语言回退到英文

**Files:**
- Delete: `web/packages/i18n/src/locales/` 下 19 种语言的目录（532 个文件）
- Create: `web/packages/i18n/src/constants/language.test.ts`
- Overwrite: `web/packages/i18n/src/constants/language.ts`、`web/packages/i18n/src/types/language.ts`
- Modify: `web/packages/i18n/package.json`、`src/index.ts`、`src/core/instance.ts`；`web/apps/web/core/store/user/profile.store.ts`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`；`tools/keywords.json`
- Generate: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 7（`/tmp/nerve-p1/probe.sh`）
- Produces: `TLanguage = "en" | "zh-CN"`；`@plane/i18n` 导出 `toSupportedLanguage(language: string | null | undefined): TLanguage`；i18n 的 `test` 脚本（`make test-web` 12 个任务）；守卫规则 `locales`

- [ ] **Step 1: 加入守卫规则，确认它命中 19 种语言**

`tools/keywords.json`，把 `rules` 数组的结尾：

```json
        "miss": [
          "web/packages/services/src/helpers/url.test.ts",
          "web/packages/services/src/developer/api-token.service.ts"
        ]
      }
    }
  ],
```

替换为：

```json
        "miss": [
          "web/packages/services/src/helpers/url.test.ts",
          "web/packages/services/src/developer/api-token.service.ts"
        ]
      }
    },
    {
      "id": "locales",
      "phase": "M1/P1",
      "why": "多语言只留 en 和 zh-CN（M1 设计第 6 节）",
      "path": { "source": "^web/packages/i18n/src/locales/(?!(?:en|zh-CN)/)", "flags": "" },
      "samples": {
        "hit": ["web/packages/i18n/src/locales/fr/common.json", "web/packages/i18n/src/locales/zh-TW/common.json"],
        "miss": ["web/packages/i18n/src/locales/en/common.json", "web/packages/i18n/src/locales/zh-CN/common.json"]
      }
    }
  ],
```

Run: `pnpm exec oxfmt --check tools/ && node tools/keywords.mjs 2>&1 | grep -v '^keywords:' | awk '{print $1}' | sort | uniq -c`
Expected（格式检查通过）:

```
 532 locales
```

- [ ] **Step 2: 删除 19 种语言**

```bash
git rm -r -q web/packages/i18n/src/locales/{cs,de,es,fr,id,it,ja,ka-ge,ko,nl,pl,pt-BR,ro,ru,sk,tr-TR,ua,vi-VN,zh-TW}
```

Run: `git status --short | grep -c '^D '; git ls-files web/packages/i18n/src/locales | wc -l`
Expected:

```
532
      56
```

- [ ] **Step 3: 写入 `web/packages/i18n/src/constants/language.ts`（完整内容）**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TLanguage, ILanguageOption } from "../types";

export const FALLBACK_LANGUAGE: TLanguage = "en";

export const SUPPORTED_LANGUAGES: ILanguageOption[] = [
  { label: "English", value: "en" },
  { label: "简体中文", value: "zh-CN" },
];

export const LANGUAGE_STORAGE_KEY = "userLanguage";

/**
 * Returns the language if it is supported, otherwise the fallback language. A language kept in
 * local storage or in the user's profile can be one that is no longer shipped, such as "fr".
 */
export function toSupportedLanguage(language: string | null | undefined): TLanguage {
  return SUPPORTED_LANGUAGES.find((option) => option.value === language)?.value ?? FALLBACK_LANGUAGE;
}
```

- [ ] **Step 4: 写入 `web/packages/i18n/src/types/language.ts`（完整内容）**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type TLanguage = "en" | "zh-CN";

export interface ILanguageOption {
  label: string;
  value: TLanguage;
}
```

- [ ] **Step 5: 修改 `web/packages/i18n/src/index.ts`（导出 `toSupportedLanguage`）**

把：

```ts
export { FALLBACK_LANGUAGE, SUPPORTED_LANGUAGES, LANGUAGE_STORAGE_KEY } from "./constants/language";
```

替换为：

```ts
export {
  FALLBACK_LANGUAGE,
  SUPPORTED_LANGUAGES,
  LANGUAGE_STORAGE_KEY,
  toSupportedLanguage,
} from "./constants/language";
```

- [ ] **Step 6: 修改 `web/packages/i18n/src/core/instance.ts`（初始语言）**

把：

```ts
import { SUPPORTED_LANGUAGES, FALLBACK_LANGUAGE, LANGUAGE_STORAGE_KEY } from "../constants/language";
```

替换为：

```ts
import {
  SUPPORTED_LANGUAGES,
  FALLBACK_LANGUAGE,
  LANGUAGE_STORAGE_KEY,
  toSupportedLanguage,
} from "../constants/language";
```

把：

```ts
  typeof window !== "undefined" ? localStorage.getItem(LANGUAGE_STORAGE_KEY) || FALLBACK_LANGUAGE : FALLBACK_LANGUAGE;
```

替换为：

```ts
  typeof window !== "undefined" ? toSupportedLanguage(localStorage.getItem(LANGUAGE_STORAGE_KEY)) : FALLBACK_LANGUAGE;
```

- [ ] **Step 7: 修改 `web/apps/web/core/store/user/profile.store.ts`（不再强转用户资料中的语言）**

把：

```ts
import { setLanguage } from "@plane/i18n";
import type { TLanguage } from "@plane/i18n";
```

替换为：

```ts
import { setLanguage, toSupportedLanguage } from "@plane/i18n";
```

把：

```ts
        void setLanguage(userProfile.language as TLanguage);
```

替换为：

```ts
        void setLanguage(toSupportedLanguage(userProfile.language));
```

把：

```ts
        void setLanguage(data.language as TLanguage);
```

替换为：

```ts
        void setLanguage(toSupportedLanguage(data.language));
```

- [ ] **Step 8: 修改 `web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`（选择框显示回退后的语言）**

把：

```tsx
import { SUPPORTED_LANGUAGES, useTranslation } from "@plane/i18n";
```

替换为：

```tsx
import { SUPPORTED_LANGUAGES, toSupportedLanguage, useTranslation } from "@plane/i18n";
```

把：

```tsx
    const getLanguageLabel = (value: string) => {
      const selectedLanguage = SUPPORTED_LANGUAGES.find((l) => l.value === value);
      if (!selectedLanguage) return value;
      return selectedLanguage.label;
    };
```

替换为：

```tsx
    // a language that is no longer supported shows as the fallback language, which is what the app uses
    const language = toSupportedLanguage(profile?.language);
    const languageLabel = SUPPORTED_LANGUAGES.find((l) => l.value === language)?.label;
```

把：

```tsx
              value={profile?.language}
              label={profile?.language ? getLanguageLabel(profile?.language) : "Select a language"}
```

替换为：

```tsx
              value={language}
              label={languageLabel}
```

- [ ] **Step 9: 单元测试：写入 `web/packages/i18n/src/constants/language.test.ts`（完整内容），接上 vitest**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
import { SUPPORTED_LANGUAGES, toSupportedLanguage } from "./language";

describe("toSupportedLanguage", () => {
  it("supports exactly en and zh-CN", () => {
    expect(SUPPORTED_LANGUAGES.map((option) => option.value)).toEqual(["en", "zh-CN"]);
  });

  it.each(["en", "zh-CN"])("keeps %j", (language) => {
    expect(toSupportedLanguage(language)).toBe(language);
  });

  it.each(["fr", "zh-TW", "zh", "EN", "", null, undefined])("turns %j into en", (language) => {
    expect(toSupportedLanguage(language)).toBe("en");
  });
});
```

```bash
node /tmp/nerve-p1/pkg.mjs web/packages/i18n/package.json add-dev-dep vitest "catalog:"
node /tmp/nerve-p1/pkg.mjs web/packages/i18n/package.json set-script test "vitest run"
```

Run: `git diff web/packages/i18n/package.json | grep '^[-+] '`
Expected:

```
-    "fix:format": "oxfmt ."
+    "fix:format": "oxfmt .",
+    "test": "vitest run"
-    "typescript": "catalog:"
+    "typescript": "catalog:",
+    "vitest": "catalog:"
```

- [ ] **Step 10: 安装，核对锁文件**

Run: `pnpm install 2>&1 | tail -1`
Expected: `Done in …`

按 Global Constraints 执行锁文件核对。Expected（只多 i18n 的一条 importer，解析结果与 services 的 `vitest` 完全相同，没有新增包）:

```
importers: 0 dependencies removed, 1 added or changed
  + web/packages/i18n devDependencies vitest: specifier: 'catalog:'  version: 4.1.11(@opentelemetry/api@1.9.1)(@types/node@22.12.0)(@vitest/coverage-v8@4.1.11)(vite@8.0.16(@types/node@22.12.0)(esbuild@0.28.1)(jiti@2.7.0)(terser@5.43.1)(tsx@4.20.6)(yaml@2.8.3))
   11566
fc0e87397936ae9f66e937e3c44dfd8b2fb01003951652bfbb98e2b6c3e45a4f  pnpm-lock.yaml
```

然后执行 `pnpm install --frozen-lockfile 2>&1 | tail -1`、`node /tmp/nerve-p1/unused-entries.mjs`、`pnpm peers check`，结果见 Global Constraints。

- [ ] **Step 11: 核对单元测试进入 `make test-web`，失败时它也失败**

Run: `make test-web 2>&1 | grep 'Tasks:'; TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run test --output-logs=full 2>&1 | perl -pe 's/\e\[[0-9;]*m//g' | grep -E ':test: +Tests '`
Expected:

```
 Tasks:    12 successful, 12 total
@plane/i18n:test:       Tests  10 passed (10)
@plane/services:test:       Tests  13 passed (13)
```

把一个断言临时改错，确认 `make test-web` 失败，然后恢复：

```bash
cp web/packages/i18n/src/constants/language.test.ts /tmp/nerve-p1/language.test.ts
node -e 'const fs = require("fs"); const f = "web/packages/i18n/src/constants/language.test.ts"; fs.writeFileSync(f, fs.readFileSync(f, "utf8").replace(`toBe("en");`, `toBe("fr");`));'
make test-web > /tmp/nerve-p1/test-web.log 2>&1; echo "exit=$?"
grep -E 'Failed:' /tmp/nerve-p1/test-web.log
cp /tmp/nerve-p1/language.test.ts web/packages/i18n/src/constants/language.test.ts
```

Expected:

```
exit=2
Failed:    @plane/i18n#test
```

- [ ] **Step 12: 在浏览器中核对回退（M1 设计 7.5）**

写入 `/tmp/nerve-p1/lang-probe.mjs`（完整内容）：

```js
// One-off (M1/P1 Task 8): signs a stubbed user in whose profile language is each of the given values, and
// reports the language the app then uses (<html lang>) and stores (localStorage userLanguage). The stubs
// answer the Plane endpoints the app calls at start; every other /api request gets 404. Run from the
// repository root (Playwright comes from e2e/). usage: node lang-probe.mjs <port> <language>...
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const [port, ...languages] = process.argv.slice(2);
const stubs = (language) => ({
  "/api/instances/": { instance: { is_setup_done: true }, config: { is_email_password_enabled: true } },
  "/api/users/me/": { id: "u1", email: "probe@example.com", display_name: "probe", first_name: "", last_name: "" },
  "/api/users/me/profile/": { id: "p1", user: "u1", language, is_onboarded: true, theme: {} },
  "/api/users/me/settings/": { id: "u1", email: "probe@example.com", workspace: {} },
  "/api/users/me/workspaces/": [],
});
const browser = await chromium.launch();
try {
  for (const language of languages) {
    const page = await browser.newPage();
    await page.route("**/api/**", (route) => {
      const json = stubs(language)[new URL(route.request().url()).pathname];
      return json ? route.fulfill({ json }) : route.fulfill({ status: 404, json: {} });
    });
    await page.goto(`http://127.0.0.1:${port}/`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1500);
    const result = await page.evaluate(() => ({
      lang: document.documentElement.lang,
      stored: localStorage.getItem("userLanguage"),
    }));
    console.log(`profile language ${JSON.stringify(language)}: <html lang>=${result.lang}, stored=${result.stored}`);
    await page.close();
  }
} finally {
  await browser.close();
}
```

Run: `make build-web > /dev/null && bash /tmp/nerve-p1/probe.sh /tmp/nerve-p1/lang-probe.mjs fr zh-CN en`
Expected（`zh-CN` 是对照：资料中的语言确实被应用了）:

```
profile language "fr": <html lang>=en, stored=en
profile language "zh-CN": <html lang>=zh-CN, stored=zh-CN
profile language "en": <html lang>=en, stored=en
port <端口> is free
```

（修复之前同一个脚本的第一行是 `<html lang>=fr, stored=fr`，spec 2.2。）脚本和输出写进 P1 review 的附录。

- [ ] **Step 13: 收尾检查**

- `node tools/keywords.mjs`：`keywords: 9 rules, 0 exceptions, no hits.`；
- `make build-web`：11 个任务；
- `make lint-web`：49 个任务（上限都不变）；
- `make test-web`：12 个任务；
- `make knip`：与 Task 7 相同；
- `git status --short | cut -c1-2 | sort | uniq -c`：

```
   9  M
   1 ??
 532 D 
```

- [ ] **Step 14: 提交**

```bash
git add -u && git add web/packages/i18n/src/constants/language.test.ts
git commit -m "feat(i18n): ship only en and zh-CN and fall back to en

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: 删除翻译键的生成；中英文键一致性检查接进 `make lint-web`；删除死文案

**Files:**
- Delete: `web/packages/i18n/scripts/generate-types.ts`、`web/packages/i18n/scripts/lib/locale-io.ts`；en、zh-CN 各 8 个命名空间文件（`automation`、`editor`、`template`、`tour`、`update`、`wiki`、`work-item-type`、`workflow`）
- Overwrite: `web/packages/i18n/scripts/sync-check.ts`、`web/packages/i18n/src/constants/namespaces.ts`
- Modify: `web/packages/i18n/package.json`、`src/types/index.ts`、`src/index.ts`、`src/locales/{en,zh-CN}/workspace-settings.json`；`turbo.json`、`Makefile`、`.gitignore`、`.oxfmtrc.json`、`knip.jsonc`、`tools/keywords.json`

**Interfaces:**
- Consumes: Task 8
- Produces: turbo 任务 `check:sync`（i18n）；`make lint-web` = 守卫 + `turbo run check:types check:lint check:format check:sync`（50 个任务）；i18n 的上限 1；守卫规则 `i18n-key-generator`、`i18n-dead-namespaces`、`i18n-applications`

- [ ] **Step 1: 重新核对死文案没有引用**

写入 `/tmp/nerve-p1/key-refs.mjs`（M1 设计 2.3 的方法："整键字面量 + 模板前缀"）：

```js
// One-off (M1/P1 Task 9): counts how many keys of an en namespace (or of one key prefix in it) the
// code references. A key counts as referenced when it appears as a whole quoted literal ("a.b",
// 'a.b' or `a.b`), or when a template literal `a.${…}` covers it (M1 design 2.3). Run from the
// repository root. usage: node key-refs.mjs <namespace | namespace:key.prefix.>...
import { execFileSync } from "node:child_process";
import fs from "node:fs";

const files = execFileSync("git", ["ls-files", "-z", "--", "web/*.ts", "web/*.tsx"], { encoding: "utf8" })
  .split("\0")
  .filter((f) => f && !f.includes("/locales/"));
const code = files.map((f) => fs.readFileSync(f, "utf8")).join("\n");
const literals = new Set([...code.matchAll(/["'`]([\w.-]+)["'`]/g)].map((m) => m[1]));
const prefixes = [...code.matchAll(/`([\w-]+(?:\.[\w-]+)*)\.\$\{/g)].map((m) => `${m[1]}.`);
const flatten = (obj, prefix = "") =>
  Object.entries(obj).flatMap(([k, v]) => (v && typeof v === "object" ? flatten(v, `${prefix}${k}.`) : [`${prefix}${k}`]));

for (const arg of process.argv.slice(2)) {
  const [namespace, keyPrefix = ""] = arg.split(":");
  const json = JSON.parse(fs.readFileSync(`web/packages/i18n/src/locales/en/${namespace}.json`, "utf8"));
  const keys = flatten(json).filter((k) => k.startsWith(keyPrefix));
  const used = keys.filter((k) => literals.has(k) || prefixes.some((p) => k.startsWith(p)));
  console.log(`${arg}: ${keys.length} keys, ${used.length} referenced ${JSON.stringify(used.slice(0, 5))}`);
}
```

Run: `node /tmp/nerve-p1/key-refs.mjs automation editor template tour update wiki work-item-type workflow workspace-settings:workspace_settings.settings.applications. workspace-settings`
Expected（前 9 行都是 0 引用；最后一行是对照：同一个文件的其余键确实能被找到）:

```
automation: 141 keys, 0 referenced []
editor: 29 keys, 0 referenced []
template: 137 keys, 0 referenced []
tour: 89 keys, 0 referenced []
update: 29 keys, 0 referenced []
wiki: 73 keys, 0 referenced []
work-item-type: 207 keys, 0 referenced []
workflow: 40 keys, 0 referenced []
workspace-settings:workspace_settings.settings.applications.: 161 keys, 0 referenced []
workspace-settings: 352 keys, 84 referenced ["workspace_settings.page_label","workspace_settings.key_created","workspace_settings.copy_key","workspace_settings.token_copied","workspace_settings.settings.general.title"]
```

任何一个不是 0 就停下来：那是还在用的文案，不属于本 Task。

- [ ] **Step 2: 加入守卫规则，确认它们命中**

`tools/keywords.json`，把 `rules` 数组的结尾：

```json
        "miss": ["web/packages/i18n/src/locales/en/common.json", "web/packages/i18n/src/locales/zh-CN/common.json"]
      }
    }
  ],
```

替换为：

```json
        "miss": ["web/packages/i18n/src/locales/en/common.json", "web/packages/i18n/src/locales/zh-CN/common.json"]
      }
    },
    {
      "id": "i18n-key-generator",
      "phase": "M1/P1",
      "why": "生成的翻译键类型 keys.generated.ts 没有任何地方使用，生成脚本已删除",
      "files": { "source": "^(?:web/|\\.gitignore$|\\.oxfmtrc\\.json$|knip\\.jsonc$|turbo\\.json$)", "flags": "" },
      "content": { "source": "keys\\.generated|TTranslationKeys|generate:types", "flags": "" },
      "samples": {
        "hit": [
          "export type { TTranslationKeys } from \"./keys.generated\";",
          "\"build\": \"pnpm run generate:types && tsdown\""
        ],
        "miss": ["export type { TLanguage } from \"./types\";", "\"generate\": \"tsx scripts/generate.ts\""]
      }
    },
    {
      "id": "i18n-dead-namespaces",
      "phase": "M1/P1",
      "why": "企业版的命名空间（自动化、模板、工作项类型、工作流、Wiki、产品导览、更新、编辑器），代码中没有任何引用",
      "path": {
        "source": "^web/packages/i18n/src/locales/[^/]+/(?:automation|editor|template|tour|update|wiki|work-item-type|workflow)\\.json$",
        "flags": ""
      },
      "samples": {
        "hit": [
          "web/packages/i18n/src/locales/en/tour.json",
          "web/packages/i18n/src/locales/zh-CN/work-item-type.json"
        ],
        "miss": ["web/packages/i18n/src/locales/en/work-item.json", "web/packages/i18n/src/locales/en/workspace.json"]
      }
    },
    {
      "id": "i18n-applications",
      "phase": "M1/P1",
      "why": "工作区设置中企业版\"应用\"一节的文案（workspace_settings.settings.applications），代码中没有任何引用",
      "files": { "source": "^web/packages/i18n/src/locales/[^/]+/workspace-settings\\.json$", "flags": "" },
      "content": { "source": "\"applications\"\\s*:", "flags": "" },
      "samples": {
        "hit": ["    \"applications\": {"],
        "miss": ["    \"api_tokens\": {", "\"title\": \"Applications\""]
      }
    }
  ],
```

Run: `pnpm exec oxfmt --check tools/ && node tools/keywords.mjs 2>&1 | grep -v '^keywords:' | awk '{print $1}' | sort | uniq -c`
Expected（格式检查通过；本地构建过时 `src/types/keys.generated.ts` 被 `.gitignore` 挡掉，不计入）:

```
   2 i18n-applications
  16 i18n-dead-namespaces
  13 i18n-key-generator
```

- [ ] **Step 3: 删除翻译键的生成**

```bash
git rm -q web/packages/i18n/scripts/generate-types.ts web/packages/i18n/scripts/lib/locale-io.ts
rm -f web/packages/i18n/src/types/keys.generated.ts
P=/tmp/nerve-p1/pkg.mjs
node $P web/packages/i18n/package.json del-script generate:types
node $P web/packages/i18n/package.json set-script build "tsdown"
node $P web/packages/i18n/package.json set-script check:types "tsc --noEmit"
node $P web/packages/i18n/package.json set-script check:sync "tsx scripts/sync-check.ts"
```

`keys.generated.ts` 是以前构建生成的、不进仓库的文件；Step 5 之后它不再被 `.gitignore` 挡掉，必须删掉（spec 第 6 节）。

`web/packages/i18n/src/types/index.ts`，把：

```ts
export * from "./language";
export type { TTranslationKeys } from "./keys.generated";
```

替换为：

```ts
export * from "./language";
```

`web/packages/i18n/src/index.ts`，把：

```ts
export type { TLanguage, ILanguageOption } from "./types";
export type { TTranslationKeys } from "./types";
```

替换为：

```ts
export type { TLanguage, ILanguageOption } from "./types";
```

- [ ] **Step 4: 写入 `web/packages/i18n/scripts/sync-check.ts`（完整内容）**

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Usage: tsx scripts/sync-check.ts
// Fails unless src/locales has both en and zh-CN, they have the same namespace files with the same keys,
// and in each of them no key is defined in two namespace files or is also the prefix of another key. All
// namespaces share one key space (src/core/instance.ts makes every namespace a fallback), so such a key
// would be ambiguous.

import fs from "node:fs";
import path from "node:path";

const LOCALES_DIR = path.resolve(import.meta.dirname, "../src/locales");
const SOURCE = "en";
const TARGET = "zh-CN";

/** Recursively flatten an object into dot-notation keys. */
function flattenKeys(obj: Record<string, unknown>, prefix = ""): string[] {
  return Object.entries(obj).flatMap(([key, value]) => {
    const full = prefix ? `${prefix}.${key}` : key;
    return value !== null && typeof value === "object" && !Array.isArray(value)
      ? flattenKeys(value as Record<string, unknown>, full)
      : [full];
  });
}

/** Every namespace file of a locale (namespace -> its keys), or undefined when the locale directory is missing. */
function loadLocale(locale: string): Map<string, Set<string>> | undefined {
  const dir = path.join(LOCALES_DIR, locale);
  if (!fs.existsSync(dir)) return undefined;
  const namespaces = new Map<string, Set<string>>();
  for (const file of fs.readdirSync(dir).filter((f) => f.endsWith(".json"))) {
    const data = JSON.parse(fs.readFileSync(path.join(dir, file), "utf-8")) as Record<string, unknown>;
    namespaces.set(path.basename(file, ".json"), new Set(flattenKeys(data)));
  }
  return namespaces;
}

/** Keys of one locale that are defined in two namespace files, or that are also the prefix of another key. */
function findConflicts(locale: string, namespaces: Map<string, Set<string>>): string[] {
  const files = new Map<string, string[]>();
  for (const [namespace, keys] of namespaces) {
    for (const key of keys) files.set(key, [...(files.get(key) ?? []), `${namespace}.json`]);
  }
  const conflicts = new Set<string>();
  for (const [key, where] of files) {
    if (where.length > 1) conflicts.add(`${locale}: ${key} is defined in ${where.join(" and ")}`);
    const parts = key.split(".");
    for (let i = 1; i < parts.length; i++) {
      const prefix = parts.slice(0, i).join(".");
      if (files.has(prefix)) conflicts.add(`${locale}: ${prefix} is a key and also the prefix of other keys`);
    }
  }
  return [...conflicts];
}

const problems: string[] = [];
const [source, target] = [SOURCE, TARGET].map((locale) => {
  const namespaces = loadLocale(locale);
  if (namespaces) problems.push(...findConflicts(locale, namespaces));
  else problems.push(`src/locales/${locale} is missing`);
  return namespaces;
});
if (source && target) {
  for (const namespace of new Set([...source.keys(), ...target.keys()])) {
    const sourceKeys = source.get(namespace);
    const targetKeys = target.get(namespace);
    if (!sourceKeys || !targetKeys) {
      problems.push(`${namespace}.json exists only in ${sourceKeys ? SOURCE : TARGET}`);
      continue;
    }
    for (const key of sourceKeys) {
      if (!targetKeys.has(key)) problems.push(`${namespace}: ${key} is missing in ${TARGET}`);
    }
    for (const key of targetKeys) {
      if (!sourceKeys.has(key)) problems.push(`${namespace}: ${key} is missing in ${SOURCE}`);
    }
  }
}

if (problems.length > 0) {
  console.error(problems.toSorted().join("\n"));
  console.error(
    `${SOURCE} and ${TARGET} must have the same namespaces and keys, each key defined once: fix the above.`
  );
  process.exit(1);
}
console.log(`${SOURCE} and ${TARGET} have the same namespaces and keys, each key defined once.`);
```

- [ ] **Step 5: 接进 turbo 和 `make lint-web`；删掉 `keys.generated.ts` 的配置**

`turbo.json`，把：

```json
    "check:types": {
```

替换为：

```json
    "check:sync": {},
    "check:types": {
```

（默认的输入就是包内的全部文件，脚本和语言文件都在其中，缓存有效；它不需要构建。）

`Makefile`，把：

```makefile
lint-web: ## 关键词守卫；前端类型检查、oxlint（警告数等于上限）、格式检查（需要 Node）
	node tools/keywords.mjs
	$(TURBO) run check:types check:lint check:format $(TURBO_QUIET)
```

替换为：

```makefile
lint-web: ## 关键词守卫；前端类型检查、oxlint（警告数等于上限）、格式检查、中英文翻译键一致（需要 Node）
	node tools/keywords.mjs
	$(TURBO) run check:types check:lint check:format check:sync $(TURBO_QUIET)
```

Run: `grep -c "$(printf '\t')" Makefile`
Expected: `34`

`.gitignore`，删除这一行：

```
web/packages/i18n/src/types/keys.generated.ts
```

`.oxfmtrc.json`，把：

```json
    "web/packages/api-client/src/schema.gen.ts",
    "web/packages/i18n/src/types/keys.generated.ts"
```

替换为：

```json
    "web/packages/api-client/src/schema.gen.ts"
```

`knip.jsonc`，把：

```jsonc
    // 下面两处导入的是生成的、不进仓库的文件，knip 找不到它们；它们由 tsc 检查
```

替换为：

```jsonc
    // 下面这处导入的是生成的、不进仓库的文件，knip 找不到它们；它们由 tsc 检查
```

再删除这 4 行：

```jsonc
    "web/packages/i18n": {
      // 翻译键 src/types/keys.generated.ts 由 i18n 的构建生成；构建过之后 knip 能找到，会提示可以删掉这一项，不要删
      "ignoreUnresolved": ["^\\./keys\\.generated$"]
    },
```

- [ ] **Step 6: 删除 8 个命名空间和 `applications` 子树**

```bash
for ns in automation editor template tour update wiki work-item-type workflow; do
  git rm -q "web/packages/i18n/src/locales/en/$ns.json" "web/packages/i18n/src/locales/zh-CN/$ns.json"
done
for locale in en zh-CN; do
  node -e '
    const fs = require("fs");
    const file = process.argv[1];
    const data = JSON.parse(fs.readFileSync(file, "utf8"));
    delete data.workspace_settings.settings.applications;
    fs.writeFileSync(file, JSON.stringify(data, null, 2) + "\n");
  ' "web/packages/i18n/src/locales/$locale/workspace-settings.json"
done
```

（en、zh-CN 的 JSON 都是 `JSON.stringify(…, null, 2)` 加换行的格式，改写不会带来其他改动，spec 2.2。）

写入 `web/packages/i18n/src/constants/namespaces.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export const NAMESPACES = [
  "accessibility",
  "auth",
  "common",
  "cycle",
  "empty-state",
  "home",
  "inbox",
  "integration",
  "module",
  "navigation",
  "notification",
  "page",
  "power-k",
  "project",
  "project-settings",
  "settings",
  "stickies",
  "work-item",
  "workspace",
  "workspace-settings",
] as const;

export type TNamespace = (typeof NAMESPACES)[number];

export const DEFAULT_NAMESPACE: TNamespace = "common";
```

Run: `git diff --numstat -- web/packages/i18n/src/locales web/packages/i18n/src/constants/namespaces.ts`
Expected:

```
0	8	web/packages/i18n/src/constants/namespaces.ts
0	185	web/packages/i18n/src/locales/en/workspace-settings.json
0	185	web/packages/i18n/src/locales/zh-CN/workspace-settings.json
```

- [ ] **Step 7: i18n 的上限 3 → 1**

删掉的生成脚本带走了 2 条警告（剩下的一条是 `core/instance.ts` 的 `no-named-as-default-member`）。

```bash
node /tmp/nerve-p1/pkg.mjs web/packages/i18n/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 1"
```

Run: `git diff web/packages/i18n/package.json | grep '^[-+] '`
Expected:

```
-    "build": "pnpm run generate:types && tsdown",
-    "generate:types": "tsx scripts/generate-types.ts",
-    "check:sync": "tsx scripts/sync-check.ts --ci",
-    "check:lint": "node ../../../tools/lint-cap.mjs 3",
-    "check:types": "pnpm run generate:types && tsc --noEmit",
+    "build": "tsdown",
+    "check:sync": "tsx scripts/sync-check.ts",
+    "check:lint": "node ../../../tools/lint-cap.mjs 1",
+    "check:types": "tsc --noEmit",
```

- [ ] **Step 8: 收尾检查**

- `node tools/keywords.mjs`：`keywords: 12 rules, 0 exceptions, no hits.`；
- `make build-web`：11 个任务；
- `make lint-web`：`Tasks:    50 successful, 50 total`；
- `make test-web`：12 个任务；
- `make knip`：

```
Unused files (123)
Unused dependencies (1)
Unused exports (133)
Unused exported types (81)
Unused exported enum members (4)
Duplicate exports (1)
```

（`locale-io.ts` 的两个导出和一个类型不再出现。`Configuration hints` 最多剩一条：`\+types/  web/apps/web  knip.jsonc  Remove from ignoreUnresolved`，是否出现取决于本地是否运行过 react-router 的类型生成，P3 处理。）

- `git status --short | cut -c1-2 | sort | uniq -c`：

```
  13  M
  18 D 
```

- [ ] **Step 9: 提交**

```bash
git add -u
git commit -m "build(i18n): drop the unused key generator, gate en/zh-CN key parity, delete dead namespaces

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 10: 核对键一致性检查的各种失败（每项之后恢复原状）**

写入 `/tmp/nerve-p1/sync-proof.sh`（完整内容）：

```bash
#!/bin/bash
# One-off (M1/P1 Task 9): the en/zh-CN check passes as committed and fails in each case it must catch; every
# change is undone right after its run. Run from the repository root with a clean working tree.
set -uo pipefail
L=web/packages/i18n/src/locales
run() {
  echo "== $1"
  # the command of i18n's check:sync, without pnpm's own lines around it
  pnpm --dir web/packages/i18n exec tsx scripts/sync-check.ts 2>&1
  echo "exit=$?"
}
edit() {
  node -e '
    const fs = require("fs");
    const file = process.argv[1];
    const data = JSON.parse(fs.readFileSync(file, "utf8"));
    new Function("data", process.argv[2])(data);
    fs.writeFileSync(file, JSON.stringify(data, null, 2) + "\n");
  ' "$1" "$2"
}
run "as committed"
edit "$L/zh-CN/accessibility.json" 'delete data.aria_labels.projects_sidebar.workspace_logo;'
run "a key missing in zh-CN"
git checkout -- "$L/zh-CN/accessibility.json"
edit "$L/zh-CN/accessibility.json" 'data.aria_labels.probe = "探针";'
run "a key only in zh-CN"
git checkout -- "$L/zh-CN/accessibility.json"
for locale in en zh-CN; do edit "$L/$locale/home.json" 'data.aria_labels = { projects_sidebar: { workspace_logo: "x" } };'; done
run "one key in two namespaces"
git checkout -- "$L/en/home.json" "$L/zh-CN/home.json"
for locale in en zh-CN; do edit "$L/$locale/home.json" 'data["aria_labels.projects_sidebar"] = "x";'; done
run "a key that is also a prefix"
git checkout -- "$L/en/home.json" "$L/zh-CN/home.json"
mv "$L/zh-CN" /tmp/nerve-p1/zh-CN
run "zh-CN missing"
mv /tmp/nerve-p1/zh-CN "$L/zh-CN"
git status --short
```

Run: `bash /tmp/nerve-p1/sync-proof.sh`
Expected（最后的 `git status --short` 没有输出）:

```
== as committed
en and zh-CN have the same namespaces and keys, each key defined once.
exit=0
== a key missing in zh-CN
accessibility: aria_labels.projects_sidebar.workspace_logo is missing in zh-CN
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== a key only in zh-CN
accessibility: aria_labels.probe is missing in en
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== one key in two namespaces
en: aria_labels.projects_sidebar.workspace_logo is defined in accessibility.json and home.json
zh-CN: aria_labels.projects_sidebar.workspace_logo is defined in accessibility.json and home.json
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== a key that is also a prefix
en: aria_labels.projects_sidebar is a key and also the prefix of other keys
zh-CN: aria_labels.projects_sidebar is a key and also the prefix of other keys
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
== zh-CN missing
src/locales/zh-CN is missing
en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
exit=1
```

再确认 `make lint-web` 会因此失败、没有改动时命中缓存：

```bash
node -e '
  const fs = require("fs");
  const file = "web/packages/i18n/src/locales/zh-CN/accessibility.json";
  const data = JSON.parse(fs.readFileSync(file, "utf8"));
  delete data.aria_labels.projects_sidebar.workspace_logo;
  fs.writeFileSync(file, JSON.stringify(data, null, 2) + "\n");
'
make lint-web 2>&1 | grep -E 'check:sync: (accessibility|en and)|Failed:'
git checkout -- web/packages/i18n/src/locales/zh-CN/accessibility.json
TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run check:sync 2>&1 | grep 'Cached:'
```

Expected（`make` 以非 0 退出）:

```
@plane/i18n:check:sync: accessibility: aria_labels.projects_sidebar.workspace_logo is missing in zh-CN
@plane/i18n:check:sync: en and zh-CN must have the same namespaces and keys, each key defined once: fix the above.
Failed:    @plane/i18n#check:sync
Cached:    1 cached, 1 total
```

输出写进 P1 的 review。

---

### Task 10: `make web-dev` 同时监视各包；文档与交接；完整验收

**Files:**
- Modify: `Makefile`、`docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P6-knip-notes.md`
- 核对（不改）：`docs/v0/M1-frontend-trim/M1-design.md` 第 12 节

**Interfaces:**
- Consumes: Task 1–9
- Produces: `make web-dev` = `turbo run dev --filter=web... --concurrency=12`；P1 的文档同步；两份交接写明 P1 的处理结果（仍为 `open`）

- [ ] **Step 1: 写入一次性脚本**

`/tmp/nerve-p1/dev-probe.mjs`（在运行中的开发服务器上改 `@plane/constants` 的一个符号，看页面是否自己更新）：

```js
// One-off (M1/P1 Task 10): with `make web-dev` running, opens http://127.0.0.1:3000/, changes SITE_NAME in
// web/packages/constants/src/metadata.ts and waits until the open page shows the new value (app/root.tsx renders
// it as <meta name="apple-mobile-web-app-title">) without the probe reloading it; then restores the file and
// waits for the old value. Also counts console errors about hydration (#418) or `process`.
// Run from the repository root (Playwright comes from e2e/). usage: node dev-probe.mjs
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const file = "web/packages/constants/src/metadata.ts";
const original = fs.readFileSync(file, "utf8");
const marker = "export const SITE_NAME = ";
if (original.split(marker).length !== 2) throw new Error(`${marker} not found once in ${file}`);

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  const errors = [];
  let loads = 0;
  page.on("load", () => loads++);
  page.on("console", (m) => m.type() === "error" && errors.push(m.text()));
  page.on("pageerror", (e) => errors.push(e.message));
  const siteName = () =>
    page
      .evaluate(() => document.querySelector('meta[name="apple-mobile-web-app-title"]')?.getAttribute("content"))
      .catch(() => undefined);
  const waitFor = async (expected) => {
    const started = Date.now();
    while (Date.now() - started < 60_000) {
      if ((await siteName()) === expected) return `${Date.now() - started} ms`;
      await page.waitForTimeout(250);
    }
    return "not within 60 s";
  };

  await page.goto("http://127.0.0.1:3000/", { waitUntil: "networkidle", timeout: 180_000 });
  const before = await siteName();
  console.log(`before: ${JSON.stringify(before)}`);

  const loadsBefore = loads;
  fs.writeFileSync(file, original.replace(marker, `${marker}"P1 watch probe";\nexport const SITE_NAME_BEFORE_PROBE = `));
  console.log(`edited: "P1 watch probe" shown after ${await waitFor("P1 watch probe")}; page loads: ${loads - loadsBefore}`);
  fs.writeFileSync(file, original);
  console.log(`restored: the old value shown after ${await waitFor(before)}`);
  console.log(`console errors about #418 or process: ${errors.filter((e) => /#418|hydrat|process is not defined/i.test(e)).length}`);
} finally {
  fs.writeFileSync(file, original);
  await browser.close();
}
```

`/tmp/nerve-p1/web-dev-probe.sh`（在自己的进程组中启动 `make web-dev`，运行上面的脚本，结束时停止整个进程组：turbo、Vite 和各包的 tsdown 监视进程）：

```bash
#!/bin/bash
# One-off (M1/P1 Task 10): starts `make web-dev` in its own process group, waits until Vite is ready, runs
# dev-probe.mjs, then stops the whole group and checks that nothing is left and port 3000 is free.
# usage: bash /tmp/nerve-p1/web-dev-probe.sh   (from the repository root)
set -uo pipefail
LOG=/tmp/nerve-p1/web-dev.log
lsof -nP -iTCP:3000 -sTCP:LISTEN && { echo "port 3000 is in use; stop that first"; exit 1; }
perl -e 'setpgrp(0, 0); exec @ARGV' make web-dev > "$LOG" 2>&1 &
GROUP=$!
for _ in $(seq 1 480); do
  kill -0 "$GROUP" 2>/dev/null || { echo "make web-dev exited; see $LOG"; break; }
  grep -q 'Local:' "$LOG" && break
  sleep 0.5
done
grep -q 'Local:' "$LOG" && node /tmp/nerve-p1/dev-probe.mjs
kill -TERM -- -"$GROUP"
for _ in $(seq 1 20); do kill -0 -- -"$GROUP" 2>/dev/null || break; sleep 0.5; done
kill -KILL -- -"$GROUP" 2>/dev/null
wait "$GROUP" 2>/dev/null
pgrep -g "$GROUP" || echo "all web-dev processes stopped"
lsof -nP -iTCP:3000 -sTCP:LISTEN || echo "port 3000 is free"
git status --short
```

- [ ] **Step 2: 修改之前：包的改动到不了运行中的页面**

Run: `bash /tmp/nerve-p1/web-dev-probe.sh 2>&1 | grep -v Terminated`
Expected（M0 的 `make web-dev` 只运行 web 自己的开发服务器；最后的 `git status --short` 没有输出）:

```
before: "Plane | Simple, extensible, open-source project management tool."
edited: "P1 watch probe" shown after not within 60 s; page loads: 0
restored: the old value shown after <几> ms
console errors about #418 or process: 0
all web-dev processes stopped
port 3000 is free
```

- [ ] **Step 3: 修改 `Makefile`（`web-dev`）**

把：

```makefile
.PHONY: web-dev
web-dev: ## 启动前端开发服务器 http://127.0.0.1:3000，/api 转发给 make run 的后端；Ctrl-C 停止
	$(TURBO) run dev --filter=web
```

替换为：

```makefile
# web 和它依赖的 10 个包各有一个常驻的 dev 任务（共 11 个）；turbo 要求并发数大于常驻任务数，否则拒绝启动
.PHONY: web-dev
web-dev: ## 启动前端开发服务器 http://127.0.0.1:3000，同时监视 web/packages/*；/api 转发给 make run 的后端；Ctrl-C 停止
	$(TURBO) run dev --filter=web... --concurrency=12
```

Run: `grep -c "$(printf '\t')" Makefile && make | grep -c .`
Expected:

```
34
23
```

Run: `TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo run dev --filter=web... --dry=json | node -e 'const t = JSON.parse(require("fs").readFileSync(0, "utf8")).tasks.filter((x) => x.task === "dev" && x.command !== "<NONEXISTENT>"); console.log(t.length, t.map((x) => x.package).sort().join(" "))'`
Expected:

```
11 @plane/constants @plane/editor @plane/hooks @plane/i18n @plane/propel @plane/services @plane/shared-state @plane/types @plane/ui @plane/utils web
```

（`--concurrency=11` 时 turbo 拒绝启动：`You have 11 persistent tasks but turbo is configured for concurrency of 11. Set --concurrency to at least 12`。）

- [ ] **Step 4: 修改之后：包的改动出现在运行中的页面上（M1 设计 7.5、8）**

Run: `bash /tmp/nerve-p1/web-dev-probe.sh 2>&1 | grep -v Terminated`
Expected（constants 的 tsdown 监视进程重新构建 `dist/`，Vite 刷新页面一次；原型中约 1.3 秒）:

```
before: "Plane | Simple, extensible, open-source project management tool."
edited: "P1 watch probe" shown after <几千以内> ms; page loads: 1
restored: the old value shown after <几千以内> ms
console errors about #418 or process: 0
all web-dev processes stopped
port 3000 is free
```

两次的输出和脚本全文写进 P1 review 的附录。第一次冷启动时 Vite 要预构建依赖，页面可能短暂出现"504 Outdated Optimize Dep"，浏览器中刷新即可（README 写明）。

- [ ] **Step 5: 修改 `docs/v0/frontend-changes.md`**

(1) 把：

```markdown
| 暂时使用 | `packages/services`：web 的令牌设置页和文件工具函数依赖它。M2（PAT）和 M5（文件）重写对应的接口调用后，将它删除 |
```

替换为：

```markdown
| 暂时使用 | `packages/services`：web 的令牌设置页和文件工具函数依赖它。M1/P1 删掉了其中没有调用方的 49 个文件，只剩令牌服务、地址规范化（带单元测试）和上传文件的元数据工具；M2（PAT）和 M5（文件）重写对应的接口调用后，将它删除 |
```

(2) 把：

```markdown
| `package.json`（仓库根目录） | 开发依赖新增 `knip` 6.37.0 | 未使用代码检查（M0 只出报告，M1 起作为门禁；[P6 spec](M0-foundation/specs/P6-e2e-ci.md) 2.8） |
```

替换为：

```markdown
| `package.json`（仓库根目录） | 开发依赖新增 `knip` 6.37.0 | 未使用代码检查（M0 只出报告，M1 起作为门禁；[P6 spec](M0-foundation/specs/P6-e2e-ci.md) 2.8） |

### 1.3 工具链、遗留物与多语言（M1/P1）

没有删除产品功能（删除的功能见第二节）。详见 [M1/P1 spec](M1-frontend-trim/specs/P1-web-hygiene.md)。

| 位置 | 改动 | 原因 |
|---|---|---|
| 各包 `package.json` 的 `check:lint`；新增 `tools/lint-cap.mjs`；`turbo.json` | `oxlint --max-warnings=N .` 改为 `node …/tools/lint-cap.mjs N`，警告数必须等于上限；`check:lint` 的输入加上这个脚本。上限：web 777、editor 75、utils 34、ui 31、propel 29、hooks 4、constants 2、types 1、i18n 1，其余为 0 | "只降不升"由检查强制（M1 设计 7.1） |
| 新增 `tools/keywords.mjs`、`tools/keywords.json`；`Makefile` 的 `lint-web`；`knip.jsonc` | 关键词守卫：`make lint-web` 先运行它，P1 的 12 条规则看住本表删掉的东西 | 删掉的东西不再长回来（M1 设计 7.4） |
| `turbo.json`、各包 `package.json` 的脚本；`Makefile`、`.github/workflows/ci.yml` | 删掉没有调用方的任务 `build-storybook`、`check`、`clean`、`fix`、`fix:lint`、`start`，以及各包的 `clean`、`fix:lint`、`sync:check`、`storybook`、`build-storybook`、`postcss` 和 web 的 `preview`、`start`；api-client、e2e 补上 `fix:format`；`test` 任务保留，新增 `make test-web`，持续集成的 `web` 任务运行它；i18n 加 `test` 脚本和 `vitest`；新增任务 `check:sync` | 只保留 Makefile、持续集成或 README 用到的入口；前端单元测试进入持续集成（M1 设计 7.5、8） |
| ui、propel 的 Storybook | 删除配置、45 个 stories、只为它们存在的文件和代码（propel 中 5 个只被 stories 导入的文件和 `ToastStatic`、两个包的 PostCSS 配置和样式）及依赖 | 没有入口 |
| 未使用的依赖 | 删除 knip 报出的未使用依赖（editor 的 `buffer` 除外），以及只为它们存在的 `markdown-to-component.tsx`、`use-font-face-observer.d.ts`、editor 的 `postcss.config.js` | knip 的报告 |
| `pnpm-workspace.yaml`、`pnpm-lock.yaml`（仓库根目录） | 删掉随依赖失效的 catalog 条目、不再作用于任何包的 7 条 overrides（含 `path-to-regexp: 0.1.13`）、allowBuilds 的 `@swc/core`、minimumReleaseAgeExclude 中的 7 个 Storybook 包；删掉英文注释中讲没有迁入内容的句子。锁文件一共删掉 369 个包，没有新增包，存活包的版本和完整性哈希都不变 | 锁文件和一次性脚本逐条核对 |
| 13 个 `.prettierignore`、`.oxfmtrc.json`（仓库根目录） | 删除 `.prettierignore`；`ignorePatterns` 加上 `web/apps/web/.react-router/**`、`web/apps/web/build/**`、`web/packages/*/dist/**`，删掉 `keys.generated.ts` | oxfmt 会读取 `.prettierignore`，改为统一放在一处 |
| `.oxlintrc.json`、`knip.jsonc`、`.gitignore`（仓库根目录） | 删掉 Storybook 的忽略规则、i18n 的 `ignoreUnresolved`、`keys.generated.ts` | 对应的东西已删除 |
| `web/apps/web/.gitignore`、`manifest.json`、`public/favicon/` | 删除 | 只有 Sentry 的条目；没有被引用 |
| `packages/services` | 删掉没有调用方的 49 个文件 | M1 设计 3.5 |
| `packages/tailwind-config/package.json` | 加 `"type": "module"`；删掉指向不存在文件的 `main`，入口由 `exports` 声明 | 构建警告 `MODULE_TYPELESS_PACKAGE_JSON`；knip 的提示 |
| `web/apps/web/vite.config.ts` | `vite-tsconfig-paths` 插件换成 `resolve.tsconfigPaths: true`；删掉 `next/script` 的别名，连同垫片中没人用的 `image.tsx`、`script.tsx`、`next-script.d.ts` | Vite 8 内置；死代码 |
| `packages/typescript-config/base.json` | `tsBuildInfoFile` 改为 `${configDir}/.turbo/tsconfig.tsbuildinfo` | 原来各包写同一个文件 |
| `app/root.tsx`、`core/components/common/logo-spinner.tsx` | `HydrateFallback` 不再读取主题；加载动画用 CSS 的 `dark:` 变体选图片 | 修复 React #418：预渲染与首次渲染的标记不一致 |
| `packages/i18n`；web 的 `profile.store.ts` 和语言选择框 | 只留 `en`、`zh-CN`；新增 `toSupportedLanguage`（带单元测试），不支持的语言按英文处理；删除翻译键的生成；`sync-check.ts` 改为双向核对 en 与 zh-CN，保留跨命名空间的冲突检查；删除 8 个企业版命名空间和 `workspace_settings.settings.applications` | 只留中英文（M1 设计 6） |
| `Makefile` 的 `web-dev` | `turbo run dev --filter=web... --concurrency=12` | 改了 `web/packages/*` 的源码，运行中的页面随之更新 |
```

(3) 第二节的 5 行。把：

```markdown
| 多语言：只保留 `zh-CN` 和 `en` | 计划中 | |
```

替换为：

```markdown
| 多语言：只保留 `zh-CN` 和 `en` | 已完成 | M1/P1 |
```

把：

```markdown
去掉强制结尾 `/` 和延迟跳转，修复因此暴露出的"渲染时跳转"问题；删除垫片和两个未使用的文件（`script.tsx`、`image.tsx`） | 计划中 | |
```

替换为：

```markdown
去掉强制结尾 `/` 和延迟跳转，修复因此暴露出的"渲染时跳转"问题；删除垫片（两个未使用的文件 `script.tsx`、`image.tsx` 已在 M1/P1 删除） | 计划中 | |
```

把：

```markdown
| web 中的部署遗留：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore` | 计划中 | |
| `serve` 依赖及其 `start`、`preview` 脚本（当前运行即崩溃） | 计划中 | |
| `public/` 中从未注册的 `sw.js` 及 workbox 相关文件 | 计划中 | |
```

替换为：

```markdown
| web 中的部署遗留：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore` | 已完成 | M1/P1 |
| `serve` 依赖及其 `start`、`preview` 脚本（当前运行即崩溃） | 已完成 | M1/P1 |
| `public/` 中从未注册的 `sw.js` 及 workbox 相关文件 | 已完成 | M1/P1 |
```

旧地址重定向一行不动（P2、P4）。

- [ ] **Step 6: 修改 `README.md`**

(1) 把：

```markdown
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（前端的类型检查、oxlint、格式检查）。
```

替换为：

```markdown
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（关键词守卫，前端的类型检查、oxlint 警告数核对、格式检查、中英文翻译键一致性检查）。
```

(2) 把：

```markdown
- **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。改了 `web/packages/*` 下的代码，要重新执行 `make web-dev`。
```

替换为：

```markdown
- **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。`make web-dev` 同时监视 `web/packages/*`：改了某个包的代码，这个包重新构建，页面随之更新。第一次启动时 Vite 要预构建依赖，页面如果报"Outdated Optimize Dep"，刷新一次即可。
- **单元测试**：`make test-web` 运行各包 `test` 脚本中的 vitest（经 turbo，持续集成的 `web` 任务也执行它）。只给以后仍然有效的稳定逻辑写小测试，测试文件 `*.test.ts` 放在被测代码旁边；包里还没有 `test` 脚本时，加上 `"test": "vitest run"` 和开发依赖 `vitest`（`catalog:`）。`make test` 只运行 Go 测试。
```

(3) 把：

```markdown
- **lint 警告只降不升**：每个包的 `check:lint` 脚本用 `--max-warnings` 记着当前的警告数，警告多了 `make lint-web` 就失败；修掉警告后，在同一个提交里把这个数调低到新的警告数。`make lint-web` 用 `--output-logs=errors-only`，看不到具体的警告数；要看某个包当前的警告数，执行 `pnpm --filter <包名> run check:lint`，输出末尾的 `Found N warnings` 就是这个数。
```

替换为：

```markdown
- **lint 警告数等于上限**：每个包的 `check:lint` 脚本是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，它运行 oxlint，要求警告数正好等于上限，有任何错误都失败。警告多了，`make lint-web` 失败并列出这个包的全部警告：修掉新增的那几条，上限只能调低。警告少了（修掉了警告，或者删掉了带警告的代码），同样失败，并给出应调低到的数值：在同一个提交里把上限改成这个数。`make lint-web` 只打印失败任务的输出；要看某个包的全部警告，执行 `pnpm --filter <包名> exec oxlint .`。
- **关键词守卫**：`make lint-web` 的第一步是 `node tools/keywords.mjs`，规则在 `tools/keywords.json`（M1 设计 7.4）。它检查 git 列出的文件（包括还没 `git add` 的新文件），命中规则、又没有登记例外就失败，并列出规则、文件、行号和命中的原文；规则或文件读取有问题时以 2 退出。删掉一个功能时，在同一个提交里加上它的规则（每条规则带理由和命中、不命中的样本）。确实要保留的命中登记为例外：规则、文件、命中的原文、理由和到期的 M 或 Phase，一条例外只覆盖一处；例外不再命中任何内容时守卫会提醒删掉它。改了 `tools/` 下的文件，另外执行 `pnpm exec oxfmt --check tools/ && pnpm exec oxlint tools/`（`make lint-web` 不检查仓库根目录的文件）。
```

(4) 把：

```markdown
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包。
```

替换为：

```markdown
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包。
- **多语言**：只有 `en` 和 `zh-CN`（`web/packages/i18n/src/locales/`）。两种语言的命名空间文件和键必须完全一致，同一个键不能出现在两个命名空间里，`make lint-web` 检查（i18n 包的 `check:sync`）；加、删文案时两种语言一起改。本地存储或用户资料中的其他语言按英文处理。
```

(5) 把：

```markdown
- **未使用的代码**：`make knip` 用 knip 报告未使用的文件、导出和依赖，配置在 `knip.jsonc`。M0 只出报告：迁入的 Plane 代码有 379 处，退出码仍为 0，只有 knip 自身出错时才失败；M1 删减之后清零，改为门禁。持续集成的 `web` 任务执行它。
```

替换为：

```markdown
- **未使用的代码**：`make knip` 用 knip 报告未使用的文件、导出和依赖，配置在 `knip.jsonc`。目前只出报告（M0 结束时 379 处，M1/P1 之后 343 处），退出码仍为 0，只有 knip 自身出错时才失败；M1/P3 清零之后改为门禁。持续集成的 `web` 任务执行它。
```

- [ ] **Step 7: 修改 `docs/v0/M0-foundation/M0-design.md`**

(1) 第 2 节，把：

```
  tools/plane-schema/           Plane 表结构快照工具和快照文件
```

替换为：

```
  tools/plane-schema/           Plane 表结构快照工具和快照文件
  tools/lint-cap.mjs            前端各包的 oxlint 警告数与上限的核对（M1/P1 加入，见 5.2）
  tools/keywords.mjs            关键词守卫，规则在 tools/keywords.json（M1/P1 加入，见 M1 设计 7.4）
```

(2) 5.2 节，把：

```markdown
- **后续**：M1 删掉大量代码后，重新测出一组更低的基线，并在 M1 设计文档里制定逐步清零的计划，同时决定是否加"警告减少后必须调低上限"的自动检查。
```

替换为：

```markdown
- **自动核对（M1/P1 起）**：每个包的 `check:lint` 是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，警告数必须**等于**上限，多了或少了 `make lint-web` 都失败；少了时提示应调低到的数值，所以上限总是实测值，不需要单独重新测量。M1/P1 之后的上限：web 777、editor 75、utils 34、ui 31、propel 29、hooks 4、constants 2、types 1、i18n 1，其余为 0，合计 954 条（[M1/P1 spec](../M1-frontend-trim/specs/P1-web-hygiene.md) 2.3）。逐步清零的计划见 [M1 设计](../M1-frontend-trim/M1-design.md) 7.3。
```

(3) 6.1 节，把：

```markdown
| `make web-dev` | 启动前端开发服务器（Vite，把 `/api` 转发给后端） |
```

替换为：

```markdown
| `make web-dev` | 启动前端开发服务器（Vite，把 `/api` 转发给后端），同时监视 `web/packages/*`（M1/P1） |
```

把：

```markdown
| `make lint` | golangci-lint；前端的类型检查、oxlint（按警告基线）、格式检查 |
| `make test` | Go 单元测试、集成测试、架构测试。需要 Docker（集成测试用 testcontainers） |
```

替换为：

```markdown
| `make lint` | golangci-lint；关键词守卫，前端的类型检查、oxlint（警告数等于上限）、格式检查、中英文翻译键一致性（守卫和后两项的改动来自 M1/P1） |
| `make test` | Go 单元测试、集成测试、架构测试。需要 Docker（集成测试用 testcontainers） |
| `make test-web` | 前端单元测试（各包的 vitest，经 turbo；需要 Node；M1/P1 加入） |
```

把：

```markdown
`gen`、`gen-check`、`lint` 按区域拆分出 `-go`（只需要 Go）和 `-web`（需要 Node，先执行 `pnpm install`）两个后缀（`gen-go`/`gen-web`、`gen-check-go`/`gen-check-web`、`lint-go`/`lint-web`）；不带后缀的命令依次执行两个区域，供本地使用（M0/P3，见 [P3 spec](specs/P3-api-contract.md) 2.10）。
```

替换为：

```markdown
`gen`、`gen-check`、`lint` 按区域拆分出 `-go`（只需要 Go）和 `-web`（需要 Node，先执行 `pnpm install`）两个后缀（`gen-go`/`gen-web`、`gen-check-go`/`gen-check-web`、`lint-go`/`lint-web`）；不带后缀的命令依次执行两个区域，供本地使用（M0/P3，见 [P3 spec](specs/P3-api-contract.md) 2.10）。`test-web` 是前端单独的入口；`make test` 仍只运行 Go 测试，持续集成的 `server` 任务调用它，不需要 Node（M1/P1）。
```

(4) 6.2 节，把：

```markdown
3. 执行 `make web-dev`，前端运行在 http://127.0.0.1:3000 ，接口请求会被转发到后端。只运行 web 自己的开发服务器；改了 `web/packages/*` 下的代码，要重新执行 `make web-dev`。
```

替换为：

```markdown
3. 执行 `make web-dev`，前端运行在 http://127.0.0.1:3000 ，接口请求会被转发到后端。它同时监视 `web/packages/*`：改了某个包的代码，这个包的 `dev` 任务（tsdown）重新构建，页面随之更新（M1/P1；M0 中只运行 web 自己的开发服务器）。
```

(5) 6.3 节，把：

```markdown
| `web` | `corepack enable` → 缓存 pnpm 存储（按锁文件的哈希）→ `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web`（类型检查、oxlint 按警告基线、格式检查）→ `make knip`（未使用代码报告，M0/P6 加入）→ `make build-web`（M0/P5 加入） |
```

替换为：

```markdown
| `web` | `corepack enable` → 缓存 pnpm 存储（按锁文件的哈希）→ `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web`（关键词守卫、类型检查、oxlint 警告数等于上限、格式检查、中英文翻译键一致性；守卫和后两项的改动来自 M1/P1）→ `make test-web`（前端单元测试，M1/P1 加入）→ `make knip`（未使用代码报告，M0/P6 加入）→ `make build-web`（M0/P5 加入） |
```

- [ ] **Step 8: 两份 M1 交接写明 P1 的处理结果（状态仍为 `open`）**

`docs/v0/M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md`，把：

```markdown
来源：[M0/P5 评审记录](../../M0-foundation/reviews/P5-web-import-review.md)。
```

替换为：

```markdown
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

来源：[M0/P5 评审记录](../../M0-foundation/reviews/P5-web-import-review.md)。
```

`docs/v0/M1-frontend-trim/handoffs/M0-P6-knip-notes.md`，把：

```markdown
来源：[M0/P6 评审记录](../../M0-foundation/reviews/P6-e2e-ci-review.md)。
```

替换为：

```markdown
## 处理结果（M1/P1）

状态仍为 `open`：knip 改为门禁和配置提示在 M1/P3，S2 与包名在 M1/P5，`.env` 在 M1/P4。M1/P1 处理了以下几项（[P1 spec](../specs/P1-web-hygiene.md)）：

1. 配置提示的来源：`tailwind-config` 的 `main` 已删除；i18n 的翻译键生成和 `knip.jsonc` 中它的 `ignoreUnresolved` 已删除。现在最多剩 web 的 `+types/` 一条，是否出现取决于本地是否运行过 react-router 的类型生成（spec 2.10、2.12）。
2. lint 上限：`@nerve/api-client`、`@nerve/e2e` 保持 0，由 `tools/lint-cap.mjs` 核对（spec 2.3）。
3. 锁文件核对改用一次性脚本逐项比较 importers 和存活包（spec 2.6）；importers 的变化只有删除、两处对等后缀和 i18n 新增的 `vitest`。
4. 新增的 `tools/keywords.mjs` 由 `make lint-web` 调用，`knip.jsonc` 的根工作区把它列为 `entry`（spec 2.4）。

来源：[M0/P6 评审记录](../../M0-foundation/reviews/P6-e2e-ci-review.md)。
```

- [ ] **Step 9: 收尾检查并提交**

- `make build-web`：11 个任务；
- `make lint-web`：`keywords: 12 rules, 0 exceptions, no hits.`，50 个任务；
- `make test-web`：12 个任务；
- `make knip`：与 Task 9 相同；
- `git status --short`：

```
 M Makefile
 M README.md
 M docs/v0/M0-foundation/M0-design.md
 M docs/v0/M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md
 M docs/v0/M1-frontend-trim/handoffs/M0-P6-knip-notes.md
 M docs/v0/frontend-changes.md
```

```bash
git add -u
git commit -m "build(web): watch every package in make web-dev; record M1/P1 in the docs

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 10: 在干净的克隆上执行持续集成 `web` 任务的步骤，并测构建体积**

干净的克隆里没有本地留下的 `keys.generated.ts`、旧的 `dist/` 和 turbo 缓存，能发现只在本机成立的结果：

```bash
CI_DIR="$(mktemp -d)/nerve"
git clone -q --branch "$(git branch --show-current)" . "$CI_DIR"
pnpm --dir "$CI_DIR" install --frozen-lockfile 2>&1 | tail -1
make -C "$CI_DIR" gen-check-web > /dev/null && echo gen-check-web ok
make -C "$CI_DIR" lint-web 2>&1 | grep -E '^keywords:|Tasks:'
make -C "$CI_DIR" test-web 2>&1 | grep 'Tasks:'
make -C "$CI_DIR" knip 2>&1 | grep -E '^(Unused|Unlisted|Duplicate|Configuration)'
make -C "$CI_DIR" build-web 2>&1 | grep 'Tasks:'
pnpm --dir "$CI_DIR" exec node /tmp/nerve-p1/web-size.mjs
git -C "$CI_DIR" status --short --untracked-files=all | wc -l
```

Expected:

```
Done in …
gen-check-web ok
keywords: 12 rules, 0 exceptions, no hits.
 Tasks:    50 successful, 50 total
 Tasks:    12 successful, 12 total
Unused files (123)
Unused dependencies (1)
Unused exports (133)
Unused exported types (81)
Unused exported enum members (4)
Duplicate exports (1)
Configuration hints (1)
 Tasks:    11 successful, 11 total
js: 505 files, 10270415 bytes
css: 3 files, 322306 bytes
fonts: 34 files, 6849620 bytes
other: 142 files, 9719426 bytes
largest chunk: assets/toolbar-Dhce5n2w.js, 1821937 bytes
locale chunks: 40
       0
```

（`Configuration hints (1)` 是 web 的 `+types/`：`lint-web` 已经运行过 react-router 的类型生成。构建体积与 Task 1 Step 2 的基线一起写进 P1 的 review，spec 2.14。）完成后删除这个临时目录（`rm -rf "$CI_DIR"`，路径以 `mktemp -d` 的输出为准）。

- [ ] **Step 11: 端到端测试**

需要 Docker 和 Playwright 的 Chromium（README"端到端测试"一节）。测试只通过 testcontainers 启动和删除自己的 Postgres 容器。

Run: `make e2e 2>&1 | grep -E 'passed|failed'`
Expected: `  5 passed (…)`

- [ ] **Step 12: 核对多语言、M1 设计的进度表和本机状态**

Run: `git ls-files web/packages/i18n/src/locales | grep -vE '/locales/(en|zh-CN)/'`
Expected: 没有输出。

Run: `grep -n '^| P1 ' docs/v0/M1-frontend-trim/M1-design.md`
Expected（spec 和计划提交时控制者已经更新，这里不需要改动；review 链接在代码评审后补上）:

```
670:| P1 | web-hygiene | 进行中 | [spec](specs/P1-web-hygiene.md) | [plan](plans/P1-web-hygiene.md) | — |
```

Run: `git status --short; lsof -nP -iTCP:3000 -sTCP:LISTEN`
Expected: 都没有输出。

- [ ] **Step 13: 推送并确认持续集成（由控制者执行）**

推送分支，确认 `server`、`web`、`e2e` 三个任务都通过；`web` 任务的 `Lint` 一步先打印 `keywords: 12 rules, 0 exceptions, no hits.`，turbo 部分 50 个任务，`Unit tests` 一步 12 个任务。

---

## 完成后

P1 的所有 Task 完成、持续集成通过后，进行代码评审，把评审结论写进 `docs/v0/M1-frontend-trim/reviews/P1-web-hygiene-review.md`：
- 裁定 spec 第 3 节的差异；
- 附录（M1 设计 7.5）：Task 1 Step 9、Task 2 Step 12、Task 9 Step 10 的核对输出；Task 7、Task 8、Task 10 的探测脚本全文和修复前后的输出；Task 1 Step 2 和 Task 10 Step 10 的构建体积；
- 提醒已有的工作区删除本地的 `web/packages/i18n/src/types/keys.generated.ts`（否则守卫、格式检查和 knip 都会扫到它）；
- 按 spec 第 7 节交接；同步 spec 第 3 节末尾列出的 M1 设计各节；
- 把 M1 设计第 12 节中 P1 的状态改为"已完成"、补上 review 链接。
