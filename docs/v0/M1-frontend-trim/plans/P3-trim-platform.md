# M1/P3 账户、平台与死代码 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删掉 M1 设计 2.2 的全部内容（公开发布与评论可见范围、管理后台入口、第三方登录与验证码登录、找回 / 重置 / 设置密码、"检查邮箱"步骤、邮件通知偏好与营销邮件同意、修改登录邮箱、AI 助手、Unsplash、遥测残留、项目邀请的名字、企业版残留与扩展点、Plane 自身的死代码），登录和注册各剩一个"邮箱 + 密码"表单，实例配置只剩 4 个字段，knip 清零并改为门禁；每删一个功能加守卫规则、删它的文案和图片、降它造成的 lint 上限。

**Architecture:** 7 个 Task，每个一个提交。Task 1–5 是 M1 设计 9 节 P3 的第 1–5 步，第 6 步拆成 Task 6（企业版扩展点、项目邀请）和 Task 7（死代码、knip 门禁）。每个 Task 的内部节奏固定：**删文件 → `check:types` 引导删到底 → 孤儿核对（knip、包导出、带词汇的死文件、图片）→ 文案 → 守卫 → lint 上限 → 门禁 → 与原型对比 → 提交**。共享文件（根 store、工作项 store、编辑器包、`app/routes/core.ts`、各包的 `index.ts`、`tools/keywords.json`）由当前 Task 一次改完，不并行。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、MobX 6、TipTap 2.26。

**Spec:** `docs/v0/M1-frontend-trim/specs/P3-trim-platform.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

**原型：** `$P3TMP/proto`，分支 `worktree-m1-p3-trim-platform`，基点 `6d9692b`。每个 Task 的原型提交：
T1 `0d85d9d`、T2 `381a34c`、T3 `7efb8ff`、T4 `3d08bf5`、T5 `301369a`、T6 `c08cc32`、T7 `98025c6`。
本计划的每个数字都注明来自哪个原型提交。

---

## Global Constraints

### 命令与环境

- **所有命令在仓库根目录（worktree 的根目录）执行**，不要 `cd`。要在某个包里执行时用 `pnpm --dir <目录>` 或
  `pnpm --filter <包名>`。开始之前执行一次 `pnpm install --frozen-lockfile`；不做任何全局安装。
- **`$P3TMP`** 是
  `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p3`。
  原型留下的脚本都在那里；不要用裸的 `/tmp`。每个 Task 开始时核对一次：脚本不在（换了机器或会话），就从
  worktree 里的备份恢复：`mkdir -p $P3TMP && cp -R .superpowers/sdd/P3-trim-platform/p3tmp/. $P3TMP/`。
  下面"一次性脚本"一节给出了全文，备份也丢了时按全文重新写入。
- **基点副本** `$P3TMP/base`：`lintdiff.sh` 拿它比较新增的 oxlint 警告。没有时 `bash $P3TMP/setup-base.sh`
  （从本仓库克隆到 `6d9692b` 并安装依赖）。它不需要构建。
- **原型提交是参考答案。** 每个 Task 的步骤写明了要改什么和为什么；每个文件改成什么样，以原型提交为准：
  先把原型的提交取进本仓库（只需一次，不建分支、不改引用）：

  ```
  git fetch .superpowers/sdd/P3-trim-platform/proto.bundle worktree-m1-p3-trim-platform
  ```

  之后 `git show <原型提交> -- <路径>` 看某个文件的结果，`git diff --stat <原型提交>` 在提交前比较整棵树。
  比较结果应当为空；不为空时，每一处差异都要能说明（例如找到了更好的根因修法），写进 Task 报告。
  `$P3TMP/t<N>/` 里的 Python 脚本是原型做每一步用的精确替换（原文不在就失败），可以照着读，**不要求运行**。
- **一个 git 命令一次调用**：不要把多个 git 命令、`cd` 和 git 串在一起；多步的逻辑写成 `$P3TMP` 下的脚本。
  不用 `git stash`、`git clean`、`git reset --hard`；不做浅拉取；不碰 `plane/`、`refer/`（`endpoints.py`、
  `upstream.py` 只读 `plane/`）；不停、不改其他项目的 Docker 容器，不运行 `make dev-db-down` / `make dev-db-reset`。
- 改过某个 `package.json` 之后，第一条 `pnpm exec` 会先核对依赖，多打印几行 `Scope: all 16 workspace projects`
  到 `Done in …`；本计划的预期输出省略这几行。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写（P3 预计不改它，见 spec 第 3 节第 14 条）；
  `web/apps/web/.react-router/` 由 `react-router typegen` 生成。

### 每个 Task 的固定节奏

1. **先删文件**（`git rm`；本地改过的文件用 `git rm -f`）。路由文件必须**同一步**删掉
   `web/apps/web/app/routes/core.ts` 里的条目，否则 `react-router typegen` 报 ENOENT，`check:types` 全线失败。
2. **跑类型检查**：`pnpm exec turbo run check:types`。它是删除的向导：报错列出的每一处都要**删掉**，不要改成空实现，
   不要为了编译把参数改成可选。反复跑到 `Tasks:    23 successful, 23 total`。
   - 报到没动过的行上时（常见 TS2339），先 `rm -f web/apps/web/.turbo/tsconfig.tsbuildinfo web/packages/*/.turbo/tsconfig.tsbuildinfo`
     再跑一次：那是上一次增量编译留下的 `tsbuildinfo`。
   - **恒为假的化简**：删掉一个在 CE 中恒为 `false`（或恒为某个常量）的参数时，把每个分支按常量化简，写出被选中的
     那一支的**原文**。三元式选中的是字符串时写那个字符串，不要写成 `${"text"}`。
3. **孤儿核对**（`<基点>` 是上一个 Task 的提交；Task 1 是 `6d9692b`）：
   - `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t<N>.flat`，与上一个 Task 的
     同名文件比：新出现的条目是本 Task 造成的孤儿，删掉（文件删掉、导出按 `unexport.mjs` 的规则处理）。
     Task 1 开始前先生成 `$P3TMP/knip-t0.flat`。
   - `node $P3TMP/symref.mjs orphaned <基点>`：knip 看不到的包级孤儿。列出的导出如果只剩自己文件在用，去掉 `export`；
     没人用了就删。
   - `node $P3TMP/deadvocab.mjs <基点> $P3TMP/knip-t0.flat $P3TMP/t<N>/rules.json`：基点就已死、但带着本 Task 词汇的文件，
     连同只有它们在用的代码在本 Task 删（spec 第 3 节第 4 条）。
   - `node $P3TMP/assets.mjs > $P3TMP/assets-t<N>.txt && python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t<N>.txt <基点>`：
     本 Task 造成的无引用图片，删掉；以被删功能命名的图片即使基点就无引用，也随功能删。
   - **退化结构同一个 Task 收掉**：只剩一个子元素的 fragment、只剩一个键的映射或枚举、只剩一个分支的三元式、
     只剩一个取值的参数（P2 评审裁定 11）。
   - `git diff HEAD | grep -F '${"'` 必须没有输出。
4. **删文案**：`node $P3TMP/keyref.mjs orphaned <基点>` 列出本 Task 弄成无引用的键；`node $P3TMP/keyref.mjs named '<正则>'`
   找出以被删功能命名、基点就无引用的键（标 `UNUSED` 的）。两者写进一个列表文件（每行 `<命名空间> <键>`），
   `node $P3TMP/i18n-del-list.mjs <列表>` 中英文一起删；键不在时它报错且什么都不写。
   整个命名空间删除时同时删 `web/packages/i18n/src/constants/namespaces.ts` 的条目（P3 预计没有）。
   改动的文案（例如 Task 5 的注销说明、Task 6 的添加成员）中英文成对改。
5. **加守卫规则**：`node $P3TMP/kw.mjs add-rules $P3TMP/t<N>/rules.json`，然后 `pnpm exec oxfmt tools`、
   `node tools/keywords.mjs`。把剩下的命中逐条判断：属于本功能的**删掉**；属于后续 Task 或 M 的**登记例外**
   （`node $P3TMP/kw.mjs add-exceptions <文件>`）：
   - 例外精确到符号：`match` 是**正则实际匹配到的那段文本**（例如规则 `epic` 不区分大小写时写 `Epic`），
     同一文件同一段文本出现多次时写 `"count"`；
   - `until` 写 `M1/P4` 到 `M9` 之间：顶层 `phase` 在 Task 5 之前是 `M1/P2`、之后是 `M1/P3`，工具把 `until` 早于或等于
     顶层 `phase` 的例外当作过期；本 Phase 内会删掉的例外写 `M1/P4`；
   - `until: M9` 表示 v0 内不到期，`reason` 必须写明它为什么在 v0 内不可能消失；
   - 例外不再命中时工具报 `stale exception`，删掉它（`kw.mjs del-exceptions <规则> <路径>` 或 `del-path <路径>`）。
   - `node $P3TMP/alts.mjs M1/P3` 必须输出 `alternatives or variants without a hit sample: 0`：每个顶层分支、
     每个 `(?:a|b)` 的每一支都有真实代码里的命中样本（`node $P3TMP/sample.mjs` 帮着找；原型的样本就在 `rules.json` 里）。
6. **降 lint 上限**：`bash $P3TMP/lintdiff.sh $P3TMP/base` 列出比基点多出来的警告（按 `== <包>` 分组，不带行号）。
   本 Task 造成的 `no-unused-vars` 逐个删掉（`node $P3TMP/deadlocals.mjs <文件>...` 可以代劳）；已有的警告只是
   因为代码搬了位置而出现时，说明原因、不改。然后 `bash $P3TMP/caps.sh`，把每个包的实测值写回：
   `node $P3TMP/pkg.mjs <package.json> set-script check:lint "node ../../../tools/lint-cap.mjs <n>"`。
   **上限只降不升**；**先做完第 3 步的孤儿，再做这一步**（原型的 Task 3 反过来做，上限多算了一个）。
   `lintdiff.sh` 按"规则 + 文件 + 消息"比较，所以下面几条已有的警告会一直出现，它们不是新问题，不改：
   - Task 1 起：`jsx-a11y(no-autofocus)`，`auth-forms/password.tsx`（`email.tsx` 里原有的 `autoFocus` 搬了过来）；
   - Task 3 起：`react-hooks(exhaustive-deps)`，`calendar/day-tile.tsx`（缺的依赖名随删除变了，条数不变）；
   - Task 6 起：`no-shadow`、`promise(always-return)`，`project/add-project-members-modal.tsx`（文件改名，警告跟着走）。
7. **门禁**：`make lint-web`、`make test-web`、`make build-web` 都通过；`bash $P3TMP/headers.sh 6d9692b` 没有输出
   （许可证头都在）；`git diff --stat <原型提交>` 为空或每处差异都有说明。
8. **提交**：一个 Task 一个提交，提交信息用英文，最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只能有本 Task
   的改动；**出现 `?? .superpowers/` 就停下来报告**（SDD 工作区由脚本写入自忽略的 `.gitignore`，看得到它说明忽略失效，
   而关键词守卫会扫描未跟踪、未忽略的文件）。

### 引用核对的口径

- **文案用严格方法**（M1 设计 2.3、P2 评审裁定 12）：一个键只有在"整个键名是一个字符串字面量"或"模板、拼接用以点结尾的
  字面量前缀拼出它"时才算有引用；键名只是出现在更长的字符串或标识符里不算。`keyref.mjs` 就是这个方法。
  以被删功能命名的键随功能删，即使基点就无引用（P2 评审裁定 3）；其余基点就无引用的键留给收尾。
- **包导出**：knip 不报工作区包的导出，`--include-entry-exports` 在包构建过之后把 `@plane/*` 解析到 `dist/`，不能用。
  用 `symref.mjs orphaned <基点>` 找本 Task 造成的包级孤儿。`symref.mjs unused web/packages/` 列出的基线死导出
  （P3 结束时 355 个）留给收尾，不在 P3 删（spec 第 3 节第 9 条），P2 评审点名的 5 个除外（Task 7）。
- **图片**：文件名不再出现在 `web/` 的任何文本文件里（图片自己除外）就算无引用。
- **孤儿的口径**：本 Task 删除的代码是某个符号唯一的使用者时，这个符号在本 Task 删掉（以 `git grep` 在本 Task 的基点核对）。
  基点就没人用、也不带本 Task 词汇的代码归 Task 7。
- **Plane 的地址**：`python3 $P3TMP/endpoints.py /Users/xiaoruan/project/nerve-project/plane` 把 web 的每个 service 调用按方法和路径去匹配 Plane 的 Django 路由。
  P3 结束时只剩 4 处，都只差结尾斜杠（spec 2.2 第 8 条）。

### 原型的教训（执行前必读）

1. **嵌套的类型体要按范围删**：删 `TPublicIssueResponseResults` 这类多层对象类型时，按"第一个 `};`"截断会截在内层。
2. **恒为假的化简会留下 `${"x"}`**：Task 3 第一次化简 service 类型时，`${serviceType === ... ? "issues" : "epics"}` 变成了
   `${"issues"}`。写出被选中的文本；第 3 步的 `grep` 守住。
3. **lint 上限在孤儿之后定**：见固定节奏第 6 步。
4. **工具删语句不能带走许可证头**：`unexport.mjs`、`deadlocals.mjs` 已修好（从许可证头之后删）；`headers.sh` 守住。
5. **`drop_unused_imports` 类的工具看不到 `import React, { useMemo } from "react"` 里的具名导入**：Task 4 的
   `useMemo`、`useState` 由 lintdiff 发现、手工删掉。
6. **守卫规则的样本要取自真实代码**：原型三次写过"编造"的不命中样本，已换成仓库里的真实行；`github-repository`、
   应用安装在 P3 开始时已无命中，样本取自 P2 删除之前的 `57fccb6` 和 `dc492a3`。
7. **`MARKETING_` 不区分大小写会命中保留的职业选项 `marketing_or_growth`**，`llm` 不区分大小写会命中 `scrollMode`：
   规则按确切写法收窄，不为躲命中去改代码。
8. **测试里的注释也受守卫检查**：Task 5 的测试注释不能写"billing"。

### 一次性脚本

下面每个脚本都在 `$P3TMP`（和 `.superpowers/sdd/P3-trim-platform/p3tmp/`）里，这里给出全文。除非另注，都在仓库根目录运行。
`pkg.mjs`、`lock-diff.mjs`、`web-size.mjs` 与 P2 的同名脚本相同。

`keyref.mjs`：严格方法的文案引用报告（`orphaned <rev>`、`named <正则>`、`unused`）。

```js
// One-off (M1/P3): which en locale keys the code references, by the strict method of M1 design 2.3 and the
// M1/P2 review ruling 12. A key counts as referenced only when
//   - its whole dotted name is a string literal ("a.b.c", 'a.b.c' or `a.b.c`), or
//   - a template or a concatenation builds it from a literal prefix that ends at a dot boundary
//     (`a.b.${x}`, "a.b." + x, 'a.b.' + x).
// A key whose text merely appears inside a longer string or identifier does not count (keyuse.mjs of
// M1/P2 counted those, and missed common-word keys such as "export" or "required").
// The corpus is every .ts/.tsx/.js/.jsx/.mjs file under web/ except the locale files: the working tree
// for "now" (git ls-files: tracked plus untracked, not ignored), `git cat-file` for a base revision.
// usage (from the repository root):
//   node keyref.mjs orphaned <base-rev>   keys referenced at <base-rev> and not referenced now
//   node keyref.mjs named <regex>         keys whose dotted name or en text matches <regex> (case-insensitive),
//                                         each marked "used" or "UNUSED" now
//   node keyref.mjs unused                every key not referenced now (the count is the first line)
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const LOCALES = "web/packages/i18n/src/locales";
const SOURCE = /^web\/.*\.(?:[cm]?[jt]sx?)$/;
const [mode, arg] = process.argv.slice(2);

const git = (args, input) =>
  execFileSync("git", args, { encoding: "utf8", maxBuffer: 1 << 30, ...(input === undefined ? {} : { input }) });

function corpusNow() {
  return git(["ls-files", "-z", "--cached", "--others", "--exclude-standard", "web"])
    .split("\0")
    .filter((f) => SOURCE.test(f) && !f.startsWith(`${LOCALES}/`) && fs.existsSync(f))
    .map((f) => fs.readFileSync(f, "utf8"))
    .join("\n");
}

function corpusAt(rev) {
  const files = git(["ls-tree", "-r", "-z", "--name-only", rev, "--", "web"])
    .split("\0")
    .filter((f) => SOURCE.test(f) && !f.startsWith(`${LOCALES}/`));
  // one `git cat-file --batch` for all files; each object comes back as "<sha> blob <size>\n<content>\n"
  const out = execFileSync("git", ["cat-file", "--batch"], {
    input: files.map((f) => `${rev}:${f}`).join("\n"),
    maxBuffer: 1 << 30,
  });
  const parts = [];
  let i = 0;
  while (i < out.length) {
    const nl = out.indexOf(10, i);
    const size = Number(out.subarray(i, nl).toString().split(" ")[2]);
    parts.push(out.subarray(nl + 1, nl + 1 + size).toString("utf8"));
    i = nl + 1 + size + 1;
  }
  return parts.join("\n");
}

function keysNow() {
  const keys = new Map(); // dotted key -> { ns, text }
  const walk = (obj, prefix, ns) => {
    for (const [k, v] of Object.entries(obj)) {
      const key = prefix ? `${prefix}.${k}` : k;
      if (v && typeof v === "object" && !Array.isArray(v)) walk(v, key, ns);
      else keys.set(key, { ns, text: String(v) });
    }
  };
  const dir = path.join(LOCALES, "en");
  for (const f of fs.readdirSync(dir).toSorted()) {
    walk(JSON.parse(fs.readFileSync(path.join(dir, f), "utf8")), "", path.basename(f, ".json"));
  }
  return keys;
}

const esc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

// every whole string literal and every literal prefix a template or a concatenation continues
function referencesIn(corpus) {
  const literals = new Set();
  for (const m of corpus.matchAll(/(["'`])([\w.-]+)\1/g)) literals.add(m[2]);
  const prefixes = new Set();
  for (const m of corpus.matchAll(/`([\w.-]+\.)\$\{/g)) prefixes.add(m[1]);
  for (const m of corpus.matchAll(/(["'])([\w.-]+\.)\1\s*\+/g)) prefixes.add(m[2]);
  return (key) => {
    if (literals.has(key)) return "literal";
    const parts = key.split(".");
    for (let n = parts.length - 1; n >= 1; n--) {
      const p = `${parts.slice(0, n).join(".")}.`;
      if (prefixes.has(p)) return `prefix ${p}`;
    }
    return "";
  };
}

const keys = keysNow();
if (mode === "orphaned" && arg) {
  const before = referencesIn(corpusAt(arg));
  const now = referencesIn(corpusNow());
  const rows = [...keys].filter(([k]) => before(k) && !now(k));
  console.log(`keys referenced at ${arg} and not now: ${rows.length}`);
  for (const [k, { ns }] of rows) console.log(`  ${ns}  ${k}  (was: ${before(k)})`);
} else if (mode === "named" && arg) {
  const re = new RegExp(arg, "i");
  const now = referencesIn(corpusNow());
  const rows = [...keys].filter(([k, { text }]) => re.test(k) || re.test(text));
  console.log(`keys named by /${arg}/i: ${rows.length}`);
  for (const [k, { ns, text }] of rows) {
    console.log(`  ${now(k) ? "used  " : "UNUSED"}  ${ns}  ${k}  ${JSON.stringify(text.slice(0, 70))}`);
  }
} else if (mode === "unused") {
  const now = referencesIn(corpusNow());
  const rows = [...keys].filter(([k]) => !now(k));
  console.log(`keys not referenced now: ${rows.length}`);
  for (const [k, { ns }] of rows) console.log(`  ${ns}  ${k}`);
} else {
  console.error("usage: node keyref.mjs orphaned <base-rev> | named <regex> | unused");
  process.exit(2);
}
```

`i18n-del-list.mjs`：按列表文件中英文一起删键。

```js
// One-off (M1/P3): deletes keys from both locales (en, zh-CN), keeping the repository's JSON format (2-space
// indent, trailing newline, key order); objects the deletions empty are dropped. Reads "<namespace> <dotted.key>"
// lines from a file (blank lines and lines starting with # are skipped). A key may name a whole object (all its
// leaves go). A key that is not there in either locale is an error, so a stale list fails loudly, and nothing
// is written unless every key was found.
// usage: node i18n-del-list.mjs <list-file>   (from the repository root)
import fs from "node:fs";

const [listFile] = process.argv.slice(2);
if (!listFile) {
  console.error("usage: node i18n-del-list.mjs <list-file>");
  process.exit(2);
}
const byNs = new Map();
for (const raw of fs.readFileSync(listFile, "utf8").split("\n")) {
  const line = raw.trim();
  if (!line || line.startsWith("#")) continue;
  const [ns, key] = line.split(/\s+/);
  if (!byNs.has(ns)) byNs.set(ns, []);
  byNs.get(ns).push(key);
}
const out = [];
const errors = [];
let total = 0;
for (const [ns, keys] of byNs) {
  for (const locale of ["en", "zh-CN"]) {
    const file = `web/packages/i18n/src/locales/${locale}/${ns}.json`;
    const data = JSON.parse(fs.readFileSync(file, "utf8"));
    for (const key of keys) {
      const parts = key.split(".");
      let node = data;
      for (const part of parts.slice(0, -1)) node = node?.[part];
      if (!node || typeof node !== "object" || !(parts.at(-1) in node)) {
        errors.push(`${file}: no ${key}`);
        continue;
      }
      delete node[parts.at(-1)];
    }
    const prune = (obj) => {
      for (const [k, v] of Object.entries(obj)) {
        if (v && typeof v === "object" && !Array.isArray(v)) {
          prune(v);
          if (Object.keys(v).length === 0) delete obj[k];
        }
      }
    };
    prune(data);
    out.push([file, `${JSON.stringify(data, null, 2)}\n`, keys.length]);
  }
  total += keys.length;
}
if (errors.length > 0) {
  for (const e of errors) console.error(e);
  process.exit(1);
}
for (const [file, text, n] of out) {
  fs.writeFileSync(file, text);
  console.log(`${file}: ${n} keys removed`);
}
console.log(`total: ${total} keys in each locale`);
```

`keycount.mjs`：每种语言的叶子键数。

```js
// One-off (M1/P3): the number of leaf keys in each locale. usage: node keycount.mjs  (from the repository root)
import fs from "node:fs";
import path from "node:path";

const count = (o) =>
  Object.values(o).reduce((n, v) => n + (v && typeof v === "object" && !Array.isArray(v) ? count(v) : 1), 0);
for (const locale of ["en", "zh-CN"]) {
  const dir = `web/packages/i18n/src/locales/${locale}`;
  let n = 0;
  for (const f of fs.readdirSync(dir)) n += count(JSON.parse(fs.readFileSync(path.join(dir, f), "utf8")));
  console.log(`${locale}: ${n} keys`);
}
```

`symref.mjs`：包级孤儿（`orphaned <rev>`）和没人用的导出（`unused <路径前缀>`）。

```js
// One-off (M1/P3): exported symbols under web/ that other files used at <base-rev> and no other file uses now.
// knip's default report does not look at the exports of the workspace packages (they are entry files), and its
// --include-entry-exports mode resolves @plane/* imports to dist/ once the packages are built, so it cannot tell
// which package exports a deletion orphaned. This report can: an export counts as used when its name appears as
// a whole word in any other .[cm]?[jt]sx? file under web/ (locales excluded). Names defined in more than one file
// are left out (the report cannot tell the definitions apart).
// usage (from the repository root):
//   node symref.mjs orphaned <base-rev>    exports other files used at <base-rev> and no other file uses now
//   node symref.mjs unused <path-prefix>   exports under <path-prefix> that no other file uses now (test files count
//                                          as users)
import { execFileSync } from "node:child_process";
import fs from "node:fs";

const SRC = /^web\/.*\.[cm]?[jt]sx?$/;
const EXPORT =
  /export\s+(?:declare\s+)?(?:default\s+)?(?:async\s+)?(?:abstract\s+)?(?:const|let|var|function\*?|class|type|interface|enum)\s+([A-Za-z_$][\w$]*)/g;
const WORD = /[A-Za-z_$][\w$]*/g;

const git = (args) => execFileSync("git", args, { encoding: "utf8", maxBuffer: 1 << 30 });

const filesNow = () => {
  const out = new Map();
  for (const f of git(["ls-files", "-z", "--cached", "--others", "--exclude-standard"]).split("\0")) {
    if (!SRC.test(f) || !fs.existsSync(f)) continue;
    out.set(f, fs.readFileSync(f, "utf8"));
  }
  return out;
};

const filesAt = (rev) => {
  const entries = git(["ls-tree", "-r", "-z", rev, "--", "web"])
    .split("\0")
    .filter(Boolean)
    .map((l) => {
      const [meta, path] = l.split("\t");
      return { sha: meta.split(" ")[2], path };
    })
    .filter((e) => SRC.test(e.path));
  const input = entries.map((e) => e.sha).join("\n") + "\n";
  const buf = execFileSync("git", ["cat-file", "--batch"], { input, maxBuffer: 1 << 30 });
  const out = new Map();
  let pos = 0;
  for (const e of entries) {
    const nl = buf.indexOf(10, pos);
    const size = Number(buf.subarray(pos, nl).toString().split(" ")[2]);
    out.set(e.path, buf.subarray(nl + 1, nl + 1 + size).toString("utf8"));
    pos = nl + 1 + size + 1;
  }
  return out;
};

// name -> defining files, and name -> files mentioning it
const index = (files) => {
  const defs = new Map();
  const mentions = new Map();
  for (const [f, text] of files) {
    for (const m of text.matchAll(EXPORT)) {
      if (!defs.has(m[1])) defs.set(m[1], new Set());
      defs.get(m[1]).add(f);
    }
    for (const w of new Set(text.match(WORD) ?? [])) {
      if (!mentions.has(w)) mentions.set(w, new Set());
      mentions.get(w).add(f);
    }
  }
  return { defs, mentions };
};

const usedElsewhere = ({ defs, mentions }, name) => {
  const d = defs.get(name);
  return [...(mentions.get(name) ?? [])].some((f) => !d.has(f));
};

const [mode, rev] = process.argv.slice(2);
if (!["orphaned", "unused"].includes(mode) || !rev) {
  console.error("usage: node symref.mjs orphaned <base-rev> | unused <path-prefix>");
  process.exit(2);
}
const now = index(filesNow());
if (mode === "unused") {
  const rows = [];
  for (const [name, d] of now.defs) {
    const f = [...d][0];
    if (d.size === 1 && f.startsWith(rev) && !usedElsewhere(now, name)) rows.push(`${f}\t${name}`);
  }
  rows.sort();
  console.log(`exports under ${rev} that no other file uses: ${rows.length}`);
  for (const r of rows) console.log(`  ${r}`);
  process.exit(0);
}
const base = index(filesAt(rev));
const rows = [];
for (const [name, d] of now.defs) {
  if (d.size !== 1 || !base.defs.has(name) || base.defs.get(name).size !== 1) continue;
  if (usedElsewhere(base, name) && !usedElsewhere(now, name)) rows.push(`${[...d][0]}\t${name}`);
}
rows.sort();
console.log(`exports used elsewhere at ${rev} and not now: ${rows.length}`);
for (const r of rows) console.log(`  ${r}`);
```

`knipflat.mjs`：把 `knip --reporter json` 摊平成可以 `diff` 的一行一条。

```js
// One-off (M1/P3): flattens `knip --reporter json` output into one sorted line per finding
// ("<kind>\t<file>\t<symbol>"), so that two runs can be compared with diff (no line numbers, no truncation).
// usage: pnpm exec knip --no-exit-code --reporter json [--include-entry-exports] | node knipflat.mjs > out.txt
import fs from "node:fs";

const input = fs.readFileSync(0, "utf8");
const data = JSON.parse(input.slice(input.indexOf("{")));
const lines = [];
for (const issue of data.issues ?? []) {
  for (const [kind, value] of Object.entries(issue)) {
    if (kind === "file" || kind === "owners") continue;
    if (Array.isArray(value)) {
      for (const v of value) lines.push(`${kind}\t${issue.file}\t${v.name ?? ""}`);
    } else if (value && typeof value === "object") {
      for (const [name, v] of Object.entries(value)) {
        if (kind === "enumMembers" || kind === "classMembers" || kind === "nsExports" || kind === "nsTypes") {
          for (const member of Object.keys(v)) lines.push(`${kind}\t${issue.file}\t${name}.${member}`);
        } else lines.push(`${kind}\t${issue.file}\t${name}`);
      }
    }
  }
}
for (const f of data.files ?? []) lines.push(`files\t${f}\t`);
console.log([...new Set(lines)].sort().join("\n"));
```

`deadvocab.mjs`：knip 报告的未使用文件中，带着给定守卫规则词汇的那些。

```js
// One-off (M1/P3): which knip-unused files carry a task's vocabulary. For every file named in <files-list> (one path
// per line, or a knipflat file whose "files" lines are used), reads the file at <rev> and prints the ids of the rules
// (from the given rules JSON files, arrays of guard rules) whose content regex matches it, with the first match.
// usage: node deadvocab.mjs <rev> <files-list> <rules.json>...   (from the repository root)
import { execFileSync } from "node:child_process";
import fs from "node:fs";

const [rev, listFile, ...ruleFiles] = process.argv.slice(2);
const rules = ruleFiles.flatMap((f) => JSON.parse(fs.readFileSync(f, "utf8")));
const files = fs
  .readFileSync(listFile, "utf8")
  .split("\n")
  .map((l) => l.split("\t"))
  .map((p) => (p.length === 3 ? (p[0] === "files" ? p[1] : "") : p[0]))
  .filter(Boolean);
for (const f of files) {
  let text;
  try {
    text = execFileSync("git", ["show", `${rev}:${f}`], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] });
  } catch {
    continue;
  }
  const hits = [];
  for (const r of rules) {
    const m = new RegExp(r.content.source, r.content.flags).exec(text);
    if (m) hits.push(`${r.id}(${m[0]})`);
  }
  if (hits.length) console.log(`${f}\t${hits.join(" ")}`);
}
```

`unexport.mjs`：按 knipflat 文件处理未使用的导出（文件内还在用的去掉 `export`，否则删声明）。

```js
// One-off (M1/P3): acts on knip's unused exports and exported types, read from a knipflat file ("exports|types\t
// <file>\t<name>" lines). For each name, parsed with the TypeScript compiler:
//   - still used inside its own file: the `export` goes (for `export { a }` / `export type { a }` the specifier goes);
//   - used nowhere: the whole top-level declaration goes, with its leading comments.
// The name of a named function expression (observer(function X() {})) and `X.displayName = ...` do not count as
// uses; the displayName statement goes with X. Prints one line per action ("unexport" / "delete" / "SKIP <why>"),
// and "EMPTY <file>" for a file left exporting nothing (what remains is only its own helpers). Imports the deletions orphan are left to lint.
// usage: node unexport.mjs <knip.flat>   (from the repository root; typescript is resolved from web/apps/web)
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(path.resolve("web/apps/web/package.json"));
const ts = require("typescript");

// A deleted statement takes its leading comments with it, except the file's license header, which is part of the
// first statement's leading trivia (the header and the blank line after it stay).
const deletionStart = (st, text) => {
  const full = st.getFullStart();
  const header = (ts.getLeadingCommentRanges(text, full) ?? []).filter((r) =>
    text.slice(r.pos, r.end).includes("SPDX-License-Identifier")
  );
  if (!header.length) return full;
  const end = header.at(-1).end;
  return end + /^\s*/.exec(text.slice(end))[0].length;
};

const byFile = new Map();
for (const line of fs.readFileSync(process.argv[2], "utf8").split("\n")) {
  const [kind, file, name] = line.split("\t");
  if (kind !== "exports" && kind !== "types") continue;
  if (!byFile.has(file)) byFile.set(file, []);
  byFile.get(file).push(name);
}

const declName = (st) => {
  if (ts.isVariableStatement(st)) return st.declarationList.declarations.map((d) => d.name.getText());
  if (st.name) return [st.name.getText()];
  return [];
};

for (const [file, names] of byFile) {
  let text = fs.readFileSync(file, "utf8");
  // apply edits from the end so that earlier positions stay valid
  const edits = [];
  const sf = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, file.endsWith("x") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
  // `X.displayName = "..."` belongs to the declaration of X: it is not a use, and it goes when X goes
  const displayNameStmts = (name) =>
    sf.statements.filter(
      (s) =>
        ts.isExpressionStatement(s) &&
        ts.isBinaryExpression(s.expression) &&
        ts.isPropertyAccessExpression(s.expression.left) &&
        s.expression.left.name.text === "displayName" &&
        s.expression.left.expression.getText() === name
    );
  const countUses = (name, except) => {
    let n = 0;
    const skip = [...except, ...displayNameStmts(name)];
    const visit = (node) => {
      if (
        ts.isIdentifier(node) &&
        node.text === name &&
        !skip.some((e) => node.pos >= e.pos && node.end <= e.end) &&
        // the name of a named function or class expression, as in observer(function X() {}), is not a use
        !((ts.isFunctionExpression(node.parent) || ts.isClassExpression(node.parent) ||
            ts.isPropertySignature(node.parent) || ts.isPropertyAssignment(node.parent) ||
            ts.isPropertyDeclaration(node.parent) || ts.isMethodDeclaration(node.parent) ||
            ts.isMethodSignature(node.parent) || ts.isPropertyAccessExpression(node.parent)) &&
            node.parent.name === node)
      )
        n++;
      ts.forEachChild(node, visit);
    };
    visit(sf);
    return n;
  };
  for (const name of names) {
    if (name === "default") {
      const st = sf.statements.find((s) => ts.isExportAssignment(s));
      if (!st) {
        console.log(`SKIP ${file} default: no export assignment`);
        continue;
      }
      edits.push({ start: deletionStart(st, text), end: st.getEnd(), text: "" });
      console.log(`delete ${file} default`);
      continue;
    }
    // export { a, b } / export type { a } / export { a } from "./b"
    const specStmt = sf.statements.find(
      (s) => ts.isExportDeclaration(s) && s.exportClause?.elements.some((e) => e.name.text === name)
    );
    if (specStmt) {
      const els = specStmt.exportClause.elements;
      if (els.length === 1) edits.push({ start: deletionStart(specStmt, text), end: specStmt.getEnd(), text: "" });
      else {
        const el = els.find((e) => e.name.text === name);
        const i = els.indexOf(el);
        const start = i === 0 ? el.getStart() : els[i - 1].getEnd();
        const end = i === 0 ? els[1].getStart() : el.getEnd();
        edits.push({ start, end, text: "" });
      }
      console.log(`unexport ${file} ${name} (specifier)`);
      continue;
    }
    // every statement declaring the name: a function's overload signatures are separate statements
    const sts = sf.statements.filter(
      (s) => declName(s).includes(name) && s.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword)
    );
    if (!sts.length) {
      console.log(`SKIP ${file} ${name}: no exported declaration found`);
      continue;
    }
    const nameNodes = sts.flatMap((st) =>
      ts.isVariableStatement(st) ? st.declarationList.declarations.map((d) => d.name) : [st.name]
    );
    const uses = countUses(name, nameNodes);
    if (uses > 0 || sts.some((st) => declName(st).length > 1)) {
      for (const st of sts) {
        const mod = st.modifiers.find((m) => m.kind === ts.SyntaxKind.ExportKeyword);
        edits.push({ start: mod.getStart(), end: mod.getEnd() + 1, text: "" });
      }
      console.log(`unexport ${file} ${name} (${uses} uses in file)`);
    } else {
      for (const st of [...sts, ...displayNameStmts(name)])
        edits.push({ start: deletionStart(st, text), end: st.getEnd(), text: "" });
      console.log(`delete ${file} ${name}`);
    }
  }
  edits.sort((a, b) => b.start - a.start);
  for (const e of edits) text = text.slice(0, e.start) + e.text + text.slice(e.end);
  fs.writeFileSync(file, text);
  // a file left exporting nothing is dead as a whole (delete it and any barrel line that names it)
  const after = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const exportsSomething = after.statements.some(
    (s) =>
      ts.isExportDeclaration(s) ||
      ts.isExportAssignment(s) ||
      s.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword)
  );
  if (!exportsSomething) console.log(`EMPTY ${file}`);
}
```

`deadlocals.mjs`：删掉删除留在文件内的未使用声明和 import。

```js
// One-off (M1/P3): in the given files, deletes what a deletion left unused inside the file: top-level declarations
// that are not exported and have no use in the file (with their `X.displayName = ...`), then import specifiers with no
// use (an import left empty goes; a side-effect import `import "x"` stays). Repeats until nothing changes. Only run
// it on files the current change made dead code in (lintdiff's new no-unused-vars lines), so that warnings the lint
// cap already counts stay as they are. Prints what it deleted.
// usage: node deadlocals.mjs <file>...   (from the repository root; typescript is resolved from web/apps/web)
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(path.resolve("web/apps/web/package.json"));
const ts = require("typescript");

// A deleted statement takes its leading comments with it, except the file's license header, which is part of the
// first statement's leading trivia (the header and the blank line after it stay).
const deletionStart = (st, text) => {
  const full = st.getFullStart();
  const header = (ts.getLeadingCommentRanges(text, full) ?? []).filter((r) =>
    text.slice(r.pos, r.end).includes("SPDX-License-Identifier")
  );
  if (!header.length) return full;
  const end = header.at(-1).end;
  return end + /^\s*/.exec(text.slice(end))[0].length;
};

const isExported = (s) =>
  s.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword || m.kind === ts.SyntaxKind.DefaultKeyword);
const declNames = (s) => {
  if (ts.isVariableStatement(s)) return s.declarationList.declarations.map((d) => d.name.getText());
  if ((ts.isFunctionDeclaration(s) || ts.isClassDeclaration(s) || ts.isInterfaceDeclaration(s) ||
       ts.isTypeAliasDeclaration(s) || ts.isEnumDeclaration(s)) && s.name) return [s.name.text];
  return [];
};

for (const file of process.argv.slice(2)) {
  for (let round = 0; round < 20; round++) {
    const text = fs.readFileSync(file, "utf8");
    const sf = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, file.endsWith("x") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
    const displayName = (name) =>
      sf.statements.filter(
        (s) => ts.isExpressionStatement(s) && ts.isBinaryExpression(s.expression) &&
          ts.isPropertyAccessExpression(s.expression.left) && s.expression.left.name.text === "displayName" &&
          s.expression.left.expression.getText() === name
      );
    const uses = (name, skip) => {
      let n = 0;
      const visit = (node) => {
        if (ts.isIdentifier(node) && node.text === name && !skip.some((e) => node.pos >= e.pos && node.end <= e.end) &&
            !((ts.isFunctionExpression(node.parent) || ts.isClassExpression(node.parent) ||
            ts.isPropertySignature(node.parent) || ts.isPropertyAssignment(node.parent) ||
            ts.isPropertyDeclaration(node.parent) || ts.isMethodDeclaration(node.parent) ||
            ts.isMethodSignature(node.parent) || ts.isPropertyAccessExpression(node.parent)) &&
            node.parent.name === node))
          n++;
        ts.forEachChild(node, visit);
      };
      visit(sf);
      return n;
    };
    const edits = [];
    for (const s of sf.statements) {
      if (isExported(s) || ts.isImportDeclaration(s)) continue;
      const names = declNames(s);
      if (names.length !== 1) continue;
      const [name] = names;
      if (uses(name, [s, ...displayName(name)]) === 0) {
        for (const st of [s, ...displayName(name)]) edits.push({ start: deletionStart(st, text), end: st.getEnd() });
        console.log(`delete ${file} ${name}`);
      }
    }
    for (const s of sf.statements) {
      if (!ts.isImportDeclaration(s) || !s.importClause) continue;
      const clause = s.importClause;
      const specs = [];
      if (clause.name) specs.push({ node: clause.name, name: clause.name.text });
      if (clause.namedBindings && ts.isNamedImports(clause.namedBindings))
        for (const el of clause.namedBindings.elements) specs.push({ node: el, name: el.name.text });
      if (clause.namedBindings && ts.isNamespaceImport(clause.namedBindings))
        specs.push({ node: clause.namedBindings, name: clause.namedBindings.name.text });
      const unused = specs.filter((sp) => uses(sp.name, [s]) === 0);
      if (!unused.length) continue;
      for (const sp of unused) console.log(`unimport ${file} ${sp.name}`);
      if (unused.length === specs.length) {
        edits.push({ start: deletionStart(s, text), end: s.getEnd() });
        continue;
      }
      // rewrite the import without the unused names
      const keepDefault = clause.name && !unused.some((u) => u.node === clause.name);
      const named = clause.namedBindings && ts.isNamedImports(clause.namedBindings)
        ? clause.namedBindings.elements.filter((el) => !unused.some((u) => u.node === el)).map((el) => el.getText())
        : [];
      const parts = [];
      if (keepDefault) parts.push(clause.name.text);
      if (named.length) parts.push(`{ ${named.join(", ")} }`);
      const typeKw = clause.isTypeOnly ? "type " : "";
      edits.push({ start: s.getStart(), end: s.getEnd(), text: `import ${typeKw}${parts.join(", ")} from ${s.moduleSpecifier.getText()};` });
    }
    if (!edits.length) break;
    edits.sort((a, b) => b.start - a.start);
    let out = text;
    for (const e of edits) out = out.slice(0, e.start) + (e.text ?? "") + out.slice(e.end);
    fs.writeFileSync(file, out);
  }
}
```

`delmethods.mjs`：删类的方法，并提示别处是否还有 `.name(` 调用。

```js
// One-off (M1/P3): deletes the named methods (with their leading comments) from the classes in a file; fails when a
// name is not a method of exactly one class member, or when anything else in web/ still calls it as `.name(`.
// usage: node delmethods.mjs <file> <method>...   (from the repository root; typescript is resolved from web/apps/web)
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const require = createRequire(path.resolve("web/apps/web/package.json"));
const ts = require("typescript");

const [file, ...names] = process.argv.slice(2);
let text = fs.readFileSync(file, "utf8");
const sf = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
const edits = [];
for (const name of names) {
  const found = [];
  const visit = (node) => {
    if ((ts.isMethodDeclaration(node) || ts.isPropertyDeclaration(node)) && node.name?.getText() === name) found.push(node);
    ts.forEachChild(node, visit);
  };
  visit(sf);
  if (found.length !== 1) {
    console.error(`${file}: ${name} is declared ${found.length} times`);
    process.exit(1);
  }
  edits.push({ start: found[0].getFullStart(), end: found[0].getEnd() });
  console.log(`delete ${file} ${name}`);
}
edits.sort((a, b) => b.start - a.start);
for (const e of edits) text = text.slice(0, e.start) + text.slice(e.end);
fs.writeFileSync(file, text);
for (const name of names) {
  let out = "";
  try {
    out = execFileSync("git", ["grep", "-n", "-P", `\\.${name}\\(`, "--", "web"], { encoding: "utf8" });
  } catch {
    out = "";
  }
  if (out) console.log(`still called (check the receiver): ${name}\n${out}`);
}
```

`assets.mjs`：无引用的图片。

```js
// One-off (M1/P3): image files under web/ whose file name no source file mentions any more. Prints them sorted;
// compare two runs to find the images a task orphaned. A file name counts as mentioned when it appears in any
// tracked or untracked (not ignored) text file under web/ other than the image itself.
// usage: node assets.mjs   (from the repository root)
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const files = execFileSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard", "web"], {
  encoding: "utf8",
  maxBuffer: 1 << 28,
})
  .split("\0")
  .filter((f) => f && fs.existsSync(f));
const IMAGE = /\.(?:svg|png|jpe?g|webp|gif|ico|avif)$/i;
const images = files.filter((f) => IMAGE.test(f));
let corpus = "";
for (const f of files) {
  if (IMAGE.test(f) && !f.endsWith(".svg")) continue;
  if (!/\.(?:[cm]?[jt]sx?|json|css|html|svg|md)$/.test(f)) continue;
  corpus += `\n${fs.readFileSync(f, "utf8")}`;
}
const orphans = images.filter((img) => !corpus.includes(path.basename(img)));
for (const img of orphans.sort()) console.log(img);
console.error(`${orphans.length} of ${images.length} images unreferenced`);
```

`assets-orphaned.py`：本次改动造成的无引用图片。

```python
# One-off (M1/P3): of the images assets.mjs lists as unreferenced now (<list-file>), the ones whose file name some
# other text file under web/ still mentioned at <rev>: the images this change orphaned.
# usage: python3 assets-orphaned.py <list-file> <rev>   (from the repository root)
import os
import subprocess
import sys

lst, rev = sys.argv[1], sys.argv[2]
for line in open(lst):
    img = line.strip()
    if not img.startswith("web/"):
        continue
    name = os.path.basename(img)
    out = subprocess.run(["git", "grep", "-l", "-F", name, rev, "--", "web"], capture_output=True, text=True).stdout
    users = [u.split(":", 1)[1] for u in out.split() if u.split(":", 1)[1] != img]
    if users:
        print(f"{img}\t(was: {', '.join(users)})")
```

`kw.mjs`：改 `tools/keywords.json`（加规则、加例外、删例外、改 phase）。

```js
// One-off (M1/P3): edits tools/keywords.json. Rules and exceptions come from a JSON file (an array); the file is
// written back with 2-space indent (run `pnpm exec oxfmt tools` afterwards).
// usage (from the repository root):
//   node kw.mjs add-rules <rules.json>            append rules (their ids must be new)
//   node kw.mjs add-exceptions <exceptions.json>  append exceptions
//   node kw.mjs del-exceptions <rule> <path>      delete every exception of <rule> on <path> (fails if none)
//   node kw.mjs del-path <path>                   delete every exception on <path> (fails if none)
//   node kw.mjs set-phase <M1/P3>                 set the top-level phase
import fs from "node:fs";

const FILE = "tools/keywords.json";
const cfg = JSON.parse(fs.readFileSync(FILE, "utf8"));
const [op, a, b] = process.argv.slice(2);
if (op === "add-rules") {
  for (const rule of JSON.parse(fs.readFileSync(a, "utf8"))) {
    if (cfg.rules.some((r) => r.id === rule.id)) throw new Error(`rule ${rule.id} exists`);
    cfg.rules.push(rule);
  }
} else if (op === "add-exceptions") {
  cfg.exceptions.push(...JSON.parse(fs.readFileSync(a, "utf8")));
} else if (op === "del-exceptions") {
  const before = cfg.exceptions.length;
  cfg.exceptions = cfg.exceptions.filter((e) => !(e.rule === a && e.path === b));
  if (cfg.exceptions.length === before) throw new Error(`no exception of ${a} on ${b}`);
  console.log(`${before - cfg.exceptions.length} exceptions deleted`);
} else if (op === "del-path") {
  const before = cfg.exceptions.length;
  cfg.exceptions = cfg.exceptions.filter((e) => e.path !== a);
  if (cfg.exceptions.length === before) throw new Error(`no exception on ${a}`);
  console.log(`${before - cfg.exceptions.length} exceptions deleted`);
} else if (op === "set-phase") {
  cfg.phase = a;
} else {
  throw new Error(`unknown operation ${op}`);
}
fs.writeFileSync(FILE, `${JSON.stringify(cfg, null, 2)}\n`);
```

`alts.mjs`：规则的每个分支都要有命中样本。

```js
// One-off (M1/P3): for every rule of the given phase, splits the content regex into its top-level
// alternatives and, for each (?:a|b) group inside one, into one variant per branch; prints each
// alternative or variant that no hit sample matches (so the self-test would not notice it breaking).
// usage: node alts.mjs <phase>   (from the repository root), e.g. node alts.mjs M1/P3
import fs from "node:fs";

const cfg = JSON.parse(fs.readFileSync("tools/keywords.json", "utf8"));

// splits at "|" characters at depth 0 of the given source (outside groups and character classes)
const split = (src) => {
  const out = [];
  let depth = 0;
  let cls = false;
  let cur = "";
  for (let i = 0; i < src.length; i++) {
    const c = src[i];
    if (c === "\\") {
      cur += c + src[++i];
      continue;
    }
    if (cls) {
      if (c === "]") cls = false;
    } else if (c === "[") cls = true;
    else if (c === "(") depth++;
    else if (c === ")") depth--;
    else if (c === "|" && depth === 0) {
      out.push(cur);
      cur = "";
      continue;
    }
    cur += c;
  }
  out.push(cur);
  return out;
};

// the variants of one alternative: for each top-level group with an alternation, one copy per branch
const variants = (alt) => {
  const out = [];
  let depth = 0;
  let cls = false;
  let start = -1;
  for (let i = 0; i < alt.length; i++) {
    const c = alt[i];
    if (c === "\\") {
      i++;
      continue;
    }
    if (cls) {
      if (c === "]") cls = false;
      continue;
    }
    if (c === "[") cls = true;
    else if (c === "(") {
      if (depth === 0) start = i;
      depth++;
    } else if (c === ")") {
      depth--;
      if (depth === 0) {
        const inner = alt.slice(start + 1, i).replace(/^\?:/, "");
        const branches = split(inner);
        if (branches.length > 1) {
          for (const b of branches) out.push(`${alt.slice(0, start)}(?:${b})${alt.slice(i + 1)}`);
        }
      }
    }
  }
  return out;
};

let gaps = 0;
for (const rule of cfg.rules.filter((r) => r.phase === process.argv[2] && r.content)) {
  const { source, flags } = rule.content;
  const hits = rule.samples.hit;
  for (const alt of split(source)) {
    const checks = [alt, ...variants(alt)];
    for (const check of checks) {
      const re = new RegExp(check, flags);
      if (!hits.some((s) => re.test(s))) {
        gaps++;
        console.log(`${rule.id}: no hit sample for ${check === alt ? "alternative" : "variant"} ${check}`);
      }
    }
  }
}
console.log(`alternatives or variants without a hit sample: ${gaps}`);
```

`sample.mjs`：从仓库里找真实的样本行。

```js
// One-off (M1/P3 prototype): prints up to <n> distinct raw lines (JSON-encoded) from git-listed files matching
// <regex>, with the file they come from, for use as guard hit/miss samples.
// usage: node sample.mjs <source> <flags> [n=2] [pathRegex=^web/]   (from the repository root)
import { execFileSync } from "node:child_process";
import fs from "node:fs";

// or: node sample.mjs -f <file>   with one "<flags>\t<source>" per line, one sample each
const argv = process.argv.slice(2);
const specs =
  argv[0] === "-f"
    ? fs
        .readFileSync(argv[1], "utf8")
        .split("\n")
        .filter(Boolean)
        .map((l) => l.split("\t"))
        .map(([flags, src]) => [src, flags, "1"])
    : [[argv[0], argv[1] ?? "", argv[2] ?? "2"]];
const pre = new RegExp(argv[3] ?? "^(web/|turbo.json|pnpm)");
const files = execFileSync("git", ["ls-files", "-z"], { encoding: "utf8", maxBuffer: 1 << 28 })
  .split("\0")
  .filter((f) => f && pre.test(f));
for (const [src, flags, n] of specs) {
const re = new RegExp(src, flags);
const seen = new Set();
let found = false;
for (const f of files) {
  if (found) break;
  let t;
  try {
    const b = fs.readFileSync(f);
    if (b.subarray(0, 8000).includes(0)) continue;
    t = b.toString("utf8");
  } catch {
    continue;
  }
  for (const line of t.split("\n")) {
    if (!re.test(line) || seen.has(line) || line.length > 150) continue;
    seen.add(line);
    console.log(`${JSON.stringify(line)}    // ${f}`);
    if (seen.size >= Number(n)) {
      found = true;
      break;
    }
  }
}
if (seen.size === 0) console.log(`NO SAMPLE for /${src}/${flags}`);
}
```

`lintdiff.sh`：与基点副本相比新增的 oxlint 警告。

```bash
#!/bin/bash
# One-off (M1/P3): oxlint warnings (rule, file, message) that are new in the working tree compared with
# <base-clone>, for every package that has an oxlint cap. Line numbers are left out, so moved code does not show.
# (pnpm may print its dependency check before the JSON; everything before the first "{" is skipped.)
# usage: bash lintdiff.sh <base-clone>   (from the repository root of the working clone)
set -eu
base=$1
flat() {
  (cd "$1/$2" && pnpm exec oxlint --format=json . 2>/dev/null) |
    node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{const j=JSON.parse(s.slice(s.indexOf("{")));for(const d of j.diagnostics??[])console.log(`${d.code}\t${d.filename}\t${d.message}`)})' |
    sort
}
for pkg in web/apps/web web/packages/*; do
  [ -f "$pkg/package.json" ] || continue
  grep -q lint-cap "$pkg/package.json" || continue
  [ -d "$base/$pkg" ] || continue
  new=$(comm -13 <(flat "$base" "$pkg") <(flat . "$pkg") || true)
  [ -n "$new" ] && printf '== %s\n%s\n' "$pkg" "$new"
done
true
```

`caps.sh`：每个包的警告数和上限。

```bash
#!/bin/bash
# One-off (M1/P3): for every package with an oxlint cap, the warning count now and the cap in its package.json,
# and the sum of the caps.
# usage: bash caps.sh   (from the repository root)
set -eu
total=0
for pkg in web/apps/web web/packages/*; do
  [ -f "$pkg/package.json" ] || continue
  cap=$(grep -o 'lint-cap.mjs [0-9]*' "$pkg/package.json" | awk '{print $2}') || true
  [ -n "$cap" ] || continue
  found=$(cd "$pkg" && pnpm exec oxlint . 2>/dev/null | grep -o -E 'Found [0-9]+ warnings?' | awk '{print $2}')
  total=$((total + cap))
  flag=""
  [ "$found" != "$cap" ] && flag="  <-- differs"
  echo "$pkg found=$found cap=$cap$flag"
done
echo "caps total: $total"
```

`pkg.mjs`：按仓库格式改 `package.json`。

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

`lock-diff.mjs`：锁文件核对（P3 预计用不到；删依赖时按 P2 的做法用）。

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

`web-size.mjs`：构建体积（M1 设计 7.6）。

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

`stats.sh`：每个提交的 shortstat 和删 / 改 / 重命名数。

```bash
#!/bin/bash
# One-off (M1/P3): shortstat and D/M/R counts of each given commit against its parent.
# usage: bash stats.sh <sha>...   (from the repository root)
set -eu
for c in "$@"; do
  s=$(git diff --shortstat "$c~1" "$c")
  ns=$(git diff --name-status "$c~1" "$c")
  d=$(printf '%s\n' "$ns" | grep -c '^D' || true)
  m=$(printf '%s\n' "$ns" | grep -c '^M' || true)
  r=$(printf '%s\n' "$ns" | grep -c '^R' || true)
  a=$(printf '%s\n' "$ns" | grep -c '^A' || true)
  echo "$c $(git log -1 --format=%s "$c" | cut -c1-60) |$s | D=$d M=$m R=$r A=$a"
done
```

`headers.sh`：许可证头有没有被工具删掉。

```bash
#!/bin/bash
# One-off (M1/P3): source files under web/ that differ from <base-rev> and carried the license header there but do
# not now (a tool that deletes a file's first statement together with its leading comments takes the header along).
# Prints nothing when every header is still there.
# usage: bash headers.sh <base-rev>   (from the repository root; compares the working tree)
set -eu
git diff --name-only --diff-filter=M "$1" -- 'web/*.ts' 'web/*.tsx' 'web/*.mjs' 'web/*.js' | while read -r f; do
  head -3 "$f" | grep -q 'Copyright (c)' && continue
  if git show "$1:$f" | head -3 | grep -q 'Copyright (c)'; then echo "$f"; fi
done
exit 0
```

`endpoints.py`：web 的 service 调用与 Plane 路由的核对（只读 `plane/`）。

```python
# One-off (M1/P3): the web service methods whose request (method + path) Plane's backend does not route.
# Reads Plane's Django URL configs (read-only, <plane-root>/apps/api/plane) and every `this.<verb>(<url>` call in the
# web services; a path is matched with each `${...}` as one path segment, and the verb against the view's methods
# (as_view({...}) mapping, else the `def get/post/...` of the view class and its bases found in the Plane sources).
# usage: python3 endpoints.py <plane-root>   (from the repository root)
import glob
import os
import re
import subprocess
import sys

plane = os.path.join(sys.argv[1], "apps/api/plane")
PREFIX = {"app": "api/", "space": "api/public/", "license": "api/instances/", "api": "api/v1/",
          "authentication": "auth/", "web": ""}

# ---- the view classes and the HTTP verbs they define
classes = {}
for f in glob.glob(plane + "/**/*.py", recursive=True):
    src = open(f).read()
    for m in re.finditer(r"^class (\w+)\(([^)]*)\):\n((?:[ \t]+.*\n|\n)*)", src, re.M):
        verbs = set(re.findall(r"^    (?:async )?def (get|post|patch|put|delete)\(", m.group(3), re.M))
        classes.setdefault(m.group(1), (verbs, [b.strip().split(".")[-1] for b in m.group(2).split(",")]))


def class_verbs(name, seen=()):
    if name not in classes or name in seen:
        return set()
    verbs, bases = classes[name]
    out = set(verbs)
    for b in bases:
        out |= class_verbs(b, seen + (name,))
    return out


# ---- the routes
routes = []
for mod, prefix in PREFIX.items():
    files = glob.glob(f"{plane}/{mod}/urls/*.py") + glob.glob(f"{plane}/{mod}/urls.py")
    for f in files:
        src = open(f).read()
        for m in re.finditer(r"(?:re_)?path\(\s*\"([^\"]*)\",\s*(\w+)\.as_view\(\s*(\{[^}]*\})?", src):
            pat, view, mapping = m.groups()
            if mapping:
                verbs = set(re.findall(r"\"(get|post|patch|put|delete)\"", mapping))
            else:
                verbs = class_verbs(view)
            rx = re.sub(r"<(?:\w+:)?\w+>", "[^/]+", re.escape(prefix + pat).replace(r"\<", "<").replace(r"\>", ">"))
            rx = re.sub(r"<(?:\w+:)?\w+>", "[^/]+", rx)
            routes.append((re.compile("^" + rx + "$"), verbs, prefix + pat, view))

# ---- the calls in the web services
files = subprocess.run(["git", "ls-files", "web/apps/web/core/services", "web/packages/services/src"],
                       capture_output=True, text=True).stdout.split()
missing = []
total = 0
for f in files:
    if not f.endswith(".ts"):
        continue
    src = open(f).read()
    for m in re.finditer(r"this\.(get|post|patch|put|delete)\(\s*(`[^`]*`|\"[^\"]*\")", src):
        verb, url = m.group(1), m.group(2)[1:-1]
        if "${" not in url and not url.startswith("/"):
            continue
        total += 1
        before = src[: m.start()]
        names = re.findall(r"^\s*(?:async\s+)?(\w+)\s*(?:<[^>]*>)?\(", before, re.M)
        method = names[-1] if names else "?"
        path = re.sub(r"\$\{[^}]*\}", "X", re.sub(r"\$\{\"([^\"]*)\"\}", r"\1", url)).split("?")[0].lstrip("/")
        hits = [r for r in routes if r[0].match(path)]
        if not hits and not path.endswith("/") and any(r[0].match(path + "/") for r in routes):
            missing.append((f, method, verb.upper(), url, "no trailing slash"))
            continue
        if not hits:
            missing.append((f, method, verb.upper(), url, "no route"))
        elif not any(verb in r[1] for r in hits):
            missing.append((f, method, verb.upper(), url, "route without " + verb.upper() + ": " + hits[0][3]))
print(f"{total} calls, {len(missing)} not routed by Plane")
for row in missing:
    print("\t".join(row))
```

`upstream.py`：knip 报告的未使用文件在 Plane 上游的导入方。

```python
# One-off (M1/P3): for each file knip reports unused in the fork, who imports it in upstream Plane (read-only,
# <plane-root>), and whether those importers still exist in the fork. "plane-dead" = no importer upstream either;
# otherwise the importers are listed with "(gone)" when the fork deleted them.
# usage: python3 upstream.py <plane-root> <knip-text-report>   (from the repository root)
import os
import re
import subprocess
import sys

plane, report = sys.argv[1], sys.argv[2]
files, on = [], False
for line in open(report):
    if line.startswith("Unused files"):
        on = True
        continue
    if on:
        if not line.strip() or re.match(r"^[A-Z]", line):
            break
        files.append(line.split()[0])


def up(path):  # fork path -> upstream path
    return os.path.join(plane, path[len("web/"):])


for f in files:
    stem = re.sub(r"\.(tsx?|jsx?)$", "", f)
    if stem.endswith("/index"):
        stem = stem[: -len("/index")]
    parts = stem.split("/")
    tail2 = "/".join(parts[-2:])
    base = parts[-1]
    upath = up(f)
    if not os.path.exists(upath):
        print(f"{f}\tnot in upstream (fork-made)")
        continue
    # importers: any specifier ending in <dir>/<name> or a relative ./<name> in the same directory
    pat = rf"from [\"'](?:[^\"']*/)?{re.escape(tail2)}[\"']|from [\"']\./{re.escape(base)}[\"']|from [\"']\.[\"']"
    out = subprocess.run(["grep", "-rlE", pat, os.path.join(plane, "apps/web"), os.path.join(plane, "packages"),
                          "--include=*.ts", "--include=*.tsx"], capture_output=True, text=True).stdout.split()
    importers = []
    for u in out:
        if os.path.samefile(u, upath) if os.path.exists(u) else False:
            continue
        text = open(u).read()
        # a relative "./name" or "." import only counts from the same directory
        same_dir = os.path.dirname(u) == os.path.dirname(upath)
        if not re.search(rf"from [\"'](?:[^\"']*/)?{re.escape(tail2)}[\"']", text) and not same_dir:
            continue
        if re.search(r"from [\"']\.[\"']", text) and not re.search(rf"\./{re.escape(base)}|{re.escape(tail2)}", text) \
                and not (same_dir and stem.endswith(os.path.basename(os.path.dirname(upath)))):
            continue
        fork = "web/" + os.path.relpath(u, plane)
        importers.append(fork + ("" if os.path.exists(fork) else " (gone)"))
    print(f"{f}\t" + ("plane-dead" if not importers else "; ".join(importers)))
```

`setup-base.sh`：基点副本。

```bash
#!/bin/bash
# One-off (M1/P3): the base clone that lintdiff.sh compares with: this repository at 6d9692b (the P3 base) in
# $P3TMP/base, with its dependencies installed. Does nothing when the clone is already there.
# usage: bash setup-base.sh   (from the repository root)
set -eu
P=$(cd "$(dirname "$0")" && pwd)
if [ -d "$P/base/.git" ]; then
  echo "base clone exists: $(cd "$P/base" && git rev-parse --short HEAD)"
  exit 0
fi
git clone -q --no-checkout "$(pwd)" "$P/base"
(cd "$P/base" && git checkout -q 6d9692b && pnpm install --frozen-lockfile > /dev/null)
echo "base clone at $(cd "$P/base" && git rev-parse --short HEAD)"
```

`ed.py`：原型的精确替换脚本共用的小工具（`$P3TMP/t<N>/*.py` 导入它）。

```python
# One-off (M1/P3 prototype): exact-text edits that fail loudly when the old text is not there exactly once.
import os
import re
import sys

ROOT = os.environ.get("PROTO", os.getcwd())


def _p(path):
    return path if os.path.isabs(path) else os.path.join(ROOT, path)


def edit(path, pairs, count_ok=False):
    fp = _p(path)
    s = open(fp).read()
    for old, new in pairs:
        n = s.count(old)
        if n == 0 or (n > 1 and not count_ok):
            sys.exit(f"{path}: expected exactly one occurrence, found {n}:\n{old[:200]}")
        s = s.replace(old, new)
    open(fp, "w").write(s)


def sub(path, pattern, repl, flags=0, expect=None):
    fp = _p(path)
    s = open(fp).read()
    s2, n = re.subn(pattern, repl, s, flags=flags)
    if n == 0 or (expect is not None and n != expect):
        sys.exit(f"{path}: pattern {pattern!r} replaced {n} times (expected {expect})")
    open(fp, "w").write(s2)


def cut(path, start, end, repl="", include_end=False):
    """Replace the text from `start` (inclusive) up to `end` (exclusive unless include_end)."""
    fp = _p(path)
    s = open(fp).read()
    i = s.index(start)
    j = s.index(end, i)
    if include_end:
        j += len(end)
    s = s[:i] + repl + s[j:]
    open(fp, "w").write(s)


def read(path):
    return open(_p(path)).read()


def write(path, text):
    open(_p(path), "w").write(text)


def drop_lines(path, lines):
    """Remove each given line (compared after strip) exactly once from the file; fail when one is missing."""
    fp = _p(path)
    src = open(fp).read().split("\n")
    for want in lines:
        idx = [i for i, l in enumerate(src) if l.strip() == want.strip()]
        if len(idx) != 1:
            sys.exit(f"{path}: line {want!r} found {len(idx)} times")
        del src[idx[0]]
    open(fp, "w").write("\n".join(src))
```

### 文件结构

| 位置 | 改动 | Task |
|---|---|---|
| `tools/keywords.json` | 新增 18 条规则；例外 14 → 4；顶层 `phase` 在 Task 5 改为 `M1/P3` | 1–7 |
| `web/apps/web/app/routes/core.ts`、`app/routes.ts` | 删找回 / 重置 / 设置密码、计费的条目；去掉 `extended` 路由表和 `mergeRoutes` | 1, 5, 6 |
| `web/apps/web/core/store/root.store.ts` 及各 store | 多选 store 的构造；`CoreRootStore` → `RootStore`，`Base*` 类改回本名 | 3, 6 |
| 工作项 store、service、hook（`core/store/issue/`、`core/services/issue/`） | `isEpic`、service 类型、团队 store 类型、批量更新 | 3, 7 |
| `web/packages/editor/` | 企业版扩展点、斜杠命令、`NodeHighlightPlugin`、"适应宽度"、linkify 协议 | 6, 7 |
| `web/packages/types/src/index.ts`、`constants/src/index.ts`、`utils/src/index.ts` | 逐个 Task 同步导出 | 全部 |
| `web/packages/constants/src/workspace.ts` 的 `RESTRICTED_URLS` | 去掉被删功能的地址和重复项 | 1, 2, 3, 5, 7 |
| `web/packages/constants/src/navigation.test.ts` | 每次加 2 个测试（5 → 11） | 1, 5, 7 |
| `web/packages/editor/src/extensions/slash-commands/command-items-list.test.ts` | 新增（2 个测试） | 6 |
| `turbo.json`、`web/apps/web/.env.example` | 去掉 `VITE_ADMIN_*`、`VITE_SPACE_*` | 1, 2 |
| `*/package.json` 的 `check:lint` | 上限只降不升 | 1, 3–7 |
| `Makefile`、`knip.jsonc`、`.github/workflows/ci.yml`、`README.md` | knip 门禁 | 7 |
| `docs/v0/frontend-changes.md` | 第二节 10 行改为已完成 | 7 |

---

## Tasks

### Task 1: 认证、实例配置、邮件——登录和注册各剩一个"邮箱 + 密码"表单

M1 设计 9 节 P3 第 1 步：认证、实例配置、邮件相关的表单，包括新手引导和安全页；实例字段和读它们的所有地方同时改（硬）。
spec 2.3、2.11。原型提交 `0d85d9d`。

**Files**（原型 `0d85d9d`：104 个文件，删 53、改 51）

- 删除（53）：
  - `web/apps/web/app/(all)/accounts/forgot-password/{layout.tsx,page.tsx}`
  - `web/apps/web/app/(all)/accounts/reset-password/{layout.tsx,page.tsx}`
  - `web/apps/web/app/(all)/accounts/set-password/{layout.tsx,page.tsx}`
  - `web/apps/web/app/assets/auth/{gradient-bg-logo.webp,gradient-logo.webp}`
  - `web/apps/web/app/assets/logos/{gitea-logo.svg,github-dark.svg,gitlab-logo.svg,google-logo.svg}`
  - `web/apps/web/core/components/account/auth-forms/common/{container.tsx,header.tsx}`
  - `web/apps/web/core/components/account/auth-forms/{email.tsx,forgot-password-popover.tsx,forgot-password.tsx,form-root.tsx,index.ts,reset-password.tsx,set-password.tsx,unique-code.tsx}`
  - `web/apps/web/core/components/core/modals/change-email-modal.tsx`
  - `web/apps/web/core/components/instance/not-ready-view.tsx`
  - `web/apps/web/core/components/onboarding/profile-setup.tsx`
  - `web/apps/web/core/components/onboarding/steps/profile/{consent.tsx,set-password.tsx}`
  - `web/apps/web/core/components/settings/profile/content/pages/notifications/{email-notification-form.tsx,index.ts,root.tsx}`
  - `web/apps/web/core/components/ui/loader/settings/email.tsx`
  - `web/apps/web/core/hooks/oauth/{core.tsx,extended.tsx,index.ts}`
  - `web/apps/web/core/hooks/use-timer.tsx`
  - `web/apps/web/core/store/user/account.store.ts`
  - `web/apps/web/google.d.ts`
  - `web/packages/constants/src/auth/{core.ts,extended.ts}`
  - `web/packages/propel/src/icons/{github-icon.tsx,gitlab-icon.tsx}`
  - `web/packages/types/src/current-user/{index.ts,profile.ts}`
  - `web/packages/types/src/instance/{ai.ts,auth-ee.ts,auth.ts,email.ts,image.ts,workspace.ts}`
  - `web/packages/ui/src/form-fields/password/password-input.tsx`
  - `web/packages/ui/src/oauth/{index.ts,oauth-button.tsx,oauth-options.tsx}`
- 修改（51）：
  - `tools/keywords.json`
  - `turbo.json`
  - `web/apps/web/{.env.example,package.json}`
  - `web/apps/web/app/routes/core.ts`
  - `web/apps/web/core/components/account/auth-forms/{auth-header.tsx,auth-root.tsx,password.tsx}`
  - `web/apps/web/core/components/account/terms-and-conditions.tsx`
  - `web/apps/web/core/components/instance/index.ts`
  - `web/apps/web/core/components/onboarding/steps/profile/root.tsx`
  - `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`
  - `web/apps/web/core/components/settings/profile/content/pages/{index.ts,security.tsx}`
  - `web/apps/web/core/components/settings/profile/sidebar/item-categories.tsx`
  - `web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`
  - `web/apps/web/core/components/workspace/sidebar/user-menu-root.tsx`
  - `web/apps/web/core/lib/wrappers/{authentication-wrapper.tsx,instance-wrapper.tsx}`
  - `web/apps/web/core/services/{auth.service.ts,instance.service.ts,user.service.ts}`
  - `web/apps/web/core/store/instance.store.ts`
  - `web/apps/web/core/store/user/{index.ts,profile.store.ts}`
  - `web/apps/web/helpers/authentication.helper.tsx`
  - `web/packages/constants/src/auth/index.ts`
  - `web/packages/constants/src/{endpoints.ts,navigation.test.ts,workspace.ts}`
  - `web/packages/constants/src/settings/profile.ts`
  - `web/packages/i18n/src/locales/en/{accessibility.json,auth.json,common.json,settings.json,workspace-settings.json}`
  - `web/packages/i18n/src/locales/zh-CN/{accessibility.json,auth.json,common.json,settings.json,workspace-settings.json}`
  - `web/packages/propel/src/icons/index.ts`
  - `web/packages/types/src/{auth.ts,settings.ts,users.ts,workspace.ts}`
  - `web/packages/types/src/instance/{base.ts,index.ts}`
  - `web/packages/ui/src/form-fields/password/index.ts`
  - `web/packages/ui/src/index.ts`
  - `web/packages/utils/src/auth.ts`

**Interfaces**

- Consumes：P2 的 `tools/keywords.mjs`、`tools/lint-cap.mjs`；`AuthService.requestCSRFToken`（保留，CSRF 留到 M2）。
- Produces：`AuthRoot`（只渲染错误提示、页头、`AuthPasswordForm`、条款）；`IInstanceInfo = { config }`，
  `IInstanceConfig` 只剩 `enable_signup`、`is_workspace_creation_disabled`、`has_unsplash_configured`、`has_llm_configured`、
  `file_size_limit`、`space_base_url`、`is_self_managed`（后三个分别由 Task 4、2 删掉，见 spec 2.11）；
  `InstanceStore { isLoading, config, fetchInstanceInfo }`；`PROFILE_SETTINGS_TABS = ["general", "security", "preferences", "api-tokens"]`；
  守卫规则 `admin`、`third-party-login`、`auth-methods`、`email-sending`。

**Steps**

- [ ] **Step 1: 准备脚本与基点副本，记录基线**

Run: `ls $P3TMP/keyref.mjs $P3TMP/t1/rules.json && bash $P3TMP/setup-base.sh`
Expected: 两个路径都列出；`base clone exists: 6d9692b` 或 `base clone at 6d9692b`

Run: `git fetch .superpowers/sdd/P3-trim-platform/proto.bundle worktree-m1-p3-trim-platform && git log --oneline -1 98025c6`
（两条命令分两次调用）
Expected: `98025c6 feat(web): delete Plane's dead code, and make knip a gate`

Run: `pnpm install --frozen-lockfile && make lint-web 2>&1 | grep -E "keywords:|Tasks:"`
Expected: `keywords: 24 rules, 14 exceptions, no hits.` + `Tasks:    52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → `Tasks:    15 successful, 15 total`

Run: `make build-web 2>&1 | grep Tasks: && node $P3TMP/web-size.mjs`
Expected（在基点副本 `$P3TMP/base` 上实测）：

```
 Tasks:    11 successful, 11 total
js: 439 files, 7319739 bytes
css: 3 files, 305244 bytes
fonts: 25 files, 3755608 bytes
other: 122 files, 8043211 bytes
largest chunk: assets/use-editor-flagging-<hash>.js, 1381850 bytes
locale chunks: 34
```

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t0.flat; cut -f1 $P3TMP/knip-t0.flat | sort | uniq -c`
Expected: `4 enumMembers`、`108 exports`、`85 files`、`61 types`

Run: `node $P3TMP/keycount.mjs && node $P3TMP/keyref.mjs unused | head -1 && bash $P3TMP/caps.sh | tail -1`
Expected: `en: 2179 keys`、`zh-CN: 2179 keys`、`keys not referenced now: 969`、`caps total: 809`

- [ ] **Step 2: 删除页面、表单和第三方登录**

Run（`git rm` 的完整列表就是上面 Files 的"删除"，这里按主题分两次）：

```
git rm -q -r "web/apps/web/app/(all)/accounts/forgot-password" "web/apps/web/app/(all)/accounts/reset-password" \
  "web/apps/web/app/(all)/accounts/set-password" web/apps/web/core/hooks/oauth web/packages/ui/src/oauth \
  web/apps/web/core/components/settings/profile/content/pages/notifications
git rm -q web/apps/web/core/components/account/auth-forms/{email,forgot-password-popover,forgot-password,form-root,reset-password,set-password,unique-code}.tsx \
  web/apps/web/core/components/account/auth-forms/index.ts \
  web/apps/web/core/components/core/modals/change-email-modal.tsx \
  web/apps/web/core/components/instance/not-ready-view.tsx \
  web/apps/web/core/components/onboarding/steps/profile/{consent,set-password}.tsx \
  web/apps/web/core/components/ui/loader/settings/email.tsx \
  web/apps/web/core/store/user/account.store.ts web/apps/web/google.d.ts \
  web/packages/constants/src/auth/{core,extended}.ts \
  web/packages/types/src/instance/{ai,auth,auth-ee,email,image,workspace}.ts
```

**同一步**删掉 `app/routes/core.ts` 里
"Account Routes - Password Management"一段的三个 `layout(...)`。

- [ ] **Step 3: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`
原型的顺序（`0d85d9d`）：第一次 18/23（`ui` 的构建因为 `./oauth` 的导出失败），然后 22/23，然后 23/23。

要改的东西（每个文件改完的样子见 `git show 0d85d9d -- <路径>`；原型脚本 `$P3TMP/t1/{password,helper,instance,instance_types,pkgs,app,app2,app3,onboarding}.py`）：

- `auth-forms/auth-root.tsx`：只剩 `errorInfo` 的 `Banner`、`AuthHeader`、`AuthPasswordForm`、`TermsAndConditions`；
  `authMode` 直接取 prop；`error_code` 经 `authErrorHandler` 变成提示。没有"无可用登录方式"分支、没有 OAuth。
- `auth-forms/password.tsx`：不再有"返回邮箱步骤"、找回密码链接、验证码切换；邮箱是表单里可编辑的输入框
  （原来在 `email.tsx` 的 `autoFocus` 搬到这里）；CSRF 令牌的取法不变。`auth-header.tsx` 去掉步骤相关的文案。
- `helpers/authentication.helper.tsx`：删掉已删登录方式、管理后台、实例设置、SMTP、机器人用户的错误码及其文案映射，
  删 `EAuthSteps`、`EPageTypes.SET_PASSWORD`；`lib/wrappers/authentication-wrapper.tsx` 删 `SET_PASSWORD` 分支。
- `web/packages/utils/src/auth.ts`：删掉重复的错误码枚举、`authErrorHandler`、`PasswordStrength`，只剩
  `getPasswordStrength`、`PasswordCriteria`、`getPasswordCriteria`；`web/packages/constants/src/auth/index.ts` 只剩
  `E_PASSWORD_STRENGTH`（登录方式标签、`SPACE_PASSWORD_CRITERIA` 和几个重复的枚举删除）。
- 实例：`types/src/instance/base.ts` 只剩 `IInstanceInfo { config }` 和 7 个字段的 `IInstanceConfig`（删 `IInstance`、
  `IInstanceAdmin`、`IInstanceConfiguration`、`TInstanceConfigurationKeys`、`IFormattedInstanceConfiguration`、`TLoginMediums`）；
  `instance/index.ts` 只导出 `./base`；`store/instance.store.ts` 只剩 `isLoading`、`config`；`lib/wrappers/instance-wrapper.tsx`
  去掉 `error` 分支和 `InstanceNotReady` 分支，加载判断改看 `config`；`services/instance.service.ts` 删掉没人调用的
  `requestCSRFToken` 副本；`components/instance/index.ts` 去掉 `not-ready-view`。
- 管理后台：`constants/src/endpoints.ts` 的 `ADMIN_BASE_URL`、`ADMIN_BASE_PATH`、`GOD_MODE_URL`；
  `workspace/sidebar/user-menu-root.tsx` 的"进入管理后台"菜单项（恒为 `false` 的 `isUserInstanceAdmin`）；
  `turbo.json` 的 `globalEnv` 和 `web/apps/web/.env.example` 去掉 `VITE_ADMIN_BASE_URL`、`VITE_ADMIN_BASE_PATH`；
  `constants/src/workspace.ts` 的 `RESTRICTED_URLS` 去掉 `god-mode`。
- 用户：`types/src/users.ts` 去掉 `is_password_autoset`、`last_login_medium`、`has_marketing_email_consent`、
  `IUserAccount`、`IInstanceAdminStatus`、`IUserEmailNotificationSettings`；`types/src/workspace.ts` 的成员去掉
  `last_login_medium`；`types/src/auth.ts` 只剩 `ICsrfTokenData`；`store/user/index.ts` 去掉 `accounts` 和
  `handleSetPassword`，`changePassword` 的 `old_password` 改为必填；`store/user/profile.store.ts` 的默认值去掉营销同意；
  `services/auth.service.ts` 只剩 `requestCSRFToken`、`signOut`；`services/user.service.ts` 删邮件通知、修改邮箱、实例管理员的方法。
- 个人设置：`settings/profile/content/pages/index.ts` 去掉 `notifications`；`sidebar/item-categories.tsx` 去掉它的图标；
  `constants/src/settings/profile.ts` 和 `types/src/settings.ts` 去掉 `notifications` 标签页；
  `pages/general/form.tsx` 去掉修改邮箱的按钮和弹窗；`pages/security.tsx` 去掉 `is_password_autoset` 分支，只剩
  "旧密码 + 新密码"。
- 新手引导：`onboarding/steps/profile/root.tsx` 去掉设置密码和营销同意两部分（连同它们的 CSRF 调用）。
- 成员表：`workspace/settings/useMemberColumns.tsx` 删"认证"列（`LOGIN_MEDIUM_LABELS`）。
- 包导出：`ui/src/index.ts` 去掉 `./oauth`；`propel/src/icons/index.ts` 去掉 GitHub、GitLab 图标（Step 4）；
  `ui/src/form-fields/password/index.ts` 去掉 `password-input`（Step 4）。

Expected（最后一次）：`Tasks:    23 successful, 23 total`

- [ ] **Step 4: 孤儿核对**

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t1.flat; diff $P3TMP/knip-t0.flat $P3TMP/knip-t1.flat | grep '^>'`
Expected（原型第一次）：新出现 `auth-forms/common/container.tsx`、`auth-forms/common/header.tsx`、`hooks/use-timer.tsx` 三个
未使用文件 → `git rm` 它们。

Run: `node $P3TMP/symref.mjs orphaned 6d9692b`
Expected（删之前）：`PasswordInput`（`ui/src/form-fields/password/password-input.tsx`）→ 删文件和它在
`password/index.ts` 的一行。

Run: `node $P3TMP/deadvocab.mjs 6d9692b $P3TMP/knip-t0.flat $P3TMP/t1/rules.json`
Expected:

```
web/apps/web/core/components/onboarding/profile-setup.tsx	auth-methods(is_password_autoset)
web/packages/types/src/current-user/profile.ts	email-sending(marketing_email)
```

→ `git rm` 这两个文件和 `types/src/current-user/index.ts`，`types/src/index.ts` 去掉 `./current-user`。
propel 的 `github-icon.tsx`、`gitlab-icon.tsx` 基点就无引用、以第三方登录命名，一起删（`icons/index.ts` 去掉两行）。

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t1.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t1.txt 6d9692b`
Expected（删之前）：`logos/{gitea-logo,github-dark,gitlab-logo,google-logo}.svg`、`auth/{gradient-bg-logo,gradient-logo}.webp`
→ `git rm` 这 6 张。

完成后四个命令都没有输出（`symref` 打印 `exports used elsewhere at 6d9692b and not now: 0`）。

- [ ] **Step 5: 文案**

Run: `node $P3TMP/keyref.mjs orphaned 6d9692b | head -1`
Expected（原型）：`keys referenced at 6d9692b and not now: 57`

列表文件就是原型的 `$P3TMP/t1/keys.txt`（orphaned 的结果按键的父节点合并，加上以被删登录方式命名、基点就无引用的键）：

```
accessibility aria_labels.auth_forms.close_popover
auth auth.common.email.errors.required
auth auth.common.password.submit
auth auth.common.back_to_sign_in
auth auth.common.resend_in
auth auth.common.sign_in_with_unique_code
auth auth.common.forgot_password
common email_notification_setting_updated_successfully
common failed_to_update_email_notification_setting
common property_changes
common property_changes_description
common state_change
common state_change_description
common issue_completed
common issue_completed_description
common comments
common comments_description
common mentions_description
common enter_god_mode
common common.continue
common common.resend
common common.errors.default.title
settings account_settings.notifications.heading
settings account_settings.notifications.description
settings profile.actions.notifications
workspace-settings workspace_settings.settings.members.details.authentication
auth auth.common.unique_code
auth auth.forgot_password
settings account_settings.profile.change_email_modal
# named after the deleted sign-in methods and steps, unreferenced already at the base
auth sso
auth auth.ldap
auth auth.common.username
auth auth.sign_up.header.step
auth auth.sign_in.header.step
auth auth.reset_password
auth auth.set_password
common email_notifications
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t1/keys.txt | tail -1 && node $P3TMP/keycount.mjs && node $P3TMP/keyref.mjs orphaned 6d9692b | head -1`
Expected: `total: 37 keys in each locale`、`en: 1984 keys`、`zh-CN: 1984 keys`、`keys referenced at 6d9692b and not now: 0`

- [ ] **Step 6: 守卫**

规则（`$P3TMP/t1/rules.json`，原样加入）：

```json
[
  {
    "id": "admin",
    "phase": "M1/P3",
    "why": "实例管理后台（god-mode，apps/admin 没有迁入）的入口、地址常量和环境变量，以及只在后台完成设置之前出现的\"实例未完成设置\"页；实例配置改由配置文件和命令行管理（M1 设计 3.7）。实例请求失败时的维护页 MaintenanceView 是另一个分支，保留；\"请联系实例管理员\"之类的文案说的是管理员这个角色，不命中",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "god[-_ ]?mode|admin[-_]base|instance-admin|InstanceNotReady|is_setup_done",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "VITE_ADMIN_BASE_PATH=\"/god-mode\"",
        "import { GOD_MODE_URL } from \"@plane/constants\";",
        "          {t(\"enter_god_mode\")}",
        "    \"VITE_ADMIN_BASE_PATH\",",
        "    return this.get(\"/api/users/me/instance-admin/\")",
        "export function InstanceNotReady() {",
        "  if (instance?.is_setup_done === false) return <InstanceNotReady />;"
      ],
      "miss": [
        "    \"title\": \"Only your instance admin can create workspaces\",",
        "  const isWorkspaceCreationDisabled = config?.is_workspace_creation_disabled ?? false;",
        "import { MaintenanceView } from \"@/components/instance\";"
      ],
      "files": {
        "hit": ["web/apps/web/.env.example", "turbo.json", "web/packages/constants/src/endpoints.ts"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "third-party-login",
    "phase": "M1/P3",
    "why": "第三方登录：OAuth（Google、GitHub、GitLab、Gitea）的按钮、配置和提供方账户，以及从未接入的企业版 SSO、LDAP、SAML、OIDC 文案（M1 设计 2.2、3.6）；v0 只有邮箱加密码",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "oauth|\\bsso\\b|\\bldap\\b|\\bsaml\\b|\\boidc\\b",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "import { OAuthOptions } from \"@plane/ui\";",
        "import { useOAuthConfig } from \"@/hooks/oauth\";",
        "  \"sso\": {",
        "        title: \"LDAP\",",
        "        title: \"SAML\",",
        "      \"oidc\": {"
      ],
      "miss": [
        "import { AuthService } from \"@/services/auth.service\";",
        "        header: \"Work in all dimensions.\",",
        "const authService = new AuthService();"
      ],
      "files": {
        "hit": ["web/packages/i18n/src/locales/en/auth.json", "web/packages/ui/src/index.ts"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "auth-methods",
    "phase": "M1/P3",
    "why": "验证码登录、找回 / 重置 / 设置密码、登录前的\"检查邮箱\"步骤、登录方式的实例开关与 SMTP 开关、登录方式标签（M1 设计 2.2、3.6、3.7）；登录页和注册页各剩一个\"邮箱 + 密码\"表单。不搜 set_password：注册表单密码框的标签 auth.common.password.set_password（\"设置密码\"）是保留的文案。CSRF 留到 M2，不在本规则",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "unique[-_]code|UniqueCode|magic[-_](?:generate|code|login)|MAGIC_|forgot[-_]password|reset[-_]password|set-password|SET_PASSWORD|\\bsetPassword\\b|email-check|[eE]mailCheck|gitea|gitlab|is_(?:google|github|magic_login|email_password)_enabled|is_smtp_configured|SMTP|is_password_autoset|EAuthSteps|last_login_medium|LOGIN_MEDIUM",
      "flags": ""
    },
    "samples": {
      "hit": [
        "import { AuthUniqueCodeForm } from \"./unique-code\";",
        "                  {t(\"auth.common.sign_in_with_unique_code\")}",
        "    return this.post(\"/auth/magic-generate/\", data, { headers: {} })",
        "        // magic_code error handler",
        "  const isEmailBasedAuthEnabled = config?.is_email_password_enabled || config?.is_magic_login_enabled;",
        "            EAuthenticationErrorCodes.INVALID_MAGIC_CODE_SIGN_UP,",
        "import { ForgotPasswordForm } from \"@/components/account/auth-forms/forgot-password\";",
        "          {t(\"auth.common.forgot_password\")}",
        "import { ResetPasswordForm } from \"@/components/account/auth-forms/reset-password\";",
        "    \"reset_password\": {",
        "  layout(\"./(all)/accounts/set-password/layout.tsx\", [",
        "      <AuthenticationWrapper pageType={EPageTypes.SET_PASSWORD}>",
        "    await authService.setPassword(token, { password });",
        "    this.post(\"/auth/email-check/\", data, { headers: {} })",
        "      .emailCheck(data)",
        "import type { IEmailCheckData } from \"@plane/types\";",
        "import giteaLogo from \"@/app/assets/logos/gitea-logo.svg?url\";",
        "import gitlabLogo from \"@/app/assets/logos/gitlab-logo.svg?url\";",
        "      (config?.is_google_enabled ||",
        "        config?.is_github_enabled ||",
        "  const isSMTPConfigured = config?.is_smtp_configured || false;",
        "    user?.is_password_autoset ? EProfileSetupSteps.USER_DETAILS : EProfileSetupSteps.ALL",
        "import { EAuthModes, EAuthSteps } from \"@/helpers/authentication.helper\";",
        "        const loginMedium = rowData.member.last_login_medium;",
        "import { EUserPermissions, EUserPermissionsLevel, LOGIN_MEDIUM_LABELS } from \"@plane/constants\";"
      ],
      "miss": [
        "            {mode === EAuthModes.SIGN_IN ? t(\"auth.common.password.label\") : t(\"auth.common.password.set_password\")}",
        "  const [passwordFormData, setPasswordFormData] = useState<TPasswordFormValues>({ ...defaultValues, email });",
        "  async requestCSRFToken(): Promise<ICsrfTokenData> {",
        "  { name: \"Gitlab\", element: Gitlab },"
      ],
      "files": {
        "hit": ["web/apps/web/core/services/auth.service.ts", "web/packages/i18n/src/locales/zh-CN/auth.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "email-sending",
    "phase": "M1/P3",
    "why": "所有邮件发送（总体设计 1.2）：邮件通知偏好页、营销邮件同意、靠邮件验证码完成的修改登录邮箱（M1 设计 3.13）；站内通知保留",
    "files": {
      "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
      "flags": ""
    },
    "content": {
      "source": "email_notification|marketing_email|EmailNotification|change_email|ChangeEmail|notification-preferences|email/generate-code",
      "flags": ""
    },
    "samples": {
      "hit": [
        "        message: t(\"email_notification_setting_updated_successfully\"),",
        "  has_marketing_email_consent?: boolean;",
        "import type { IUserEmailNotificationSettings } from \"@plane/types\";",
        "  const changeEmailT = (path: string) => t(`account_settings.profile.change_email_modal.${path}`);",
        "export const ChangeEmailModal = observer(function ChangeEmailModal(props: Props) {",
        "    return this.get(\"/api/users/me/notification-preferences/\")",
        "    return this.post(\"/api/users/me/email/generate-code/\", data)"
      ],
      "miss": [
        "  const { unreadNotificationsCount, getUnreadNotificationsCount } = useWorkspaceNotifications();",
        "                  {t(\"auth.common.email.label\")}&nbsp;"
      ],
      "files": {
        "hit": ["web/apps/web/core/services/user.service.ts", "web/packages/i18n/src/locales/en/common.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t1/rules.json && pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected（原型）：18 处命中，全部在计费文案 `workspace/billing/comparison/plans.tsx` 和 `constants/src/subscription.ts` 里，
Task 5 删。登记 8 条例外（`$P3TMP/t1/exceptions.json`）：

```json
[
  {
    "rule": "admin",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "God Mode",
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "SAML",
    "count": 3,
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "OIDC",
    "count": 3,
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "LDAP",
    "count": 4,
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/packages/constants/src/subscription.ts",
    "match": "LDAP",
    "reason": "订阅套餐的功能名，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/packages/constants/src/subscription.ts",
    "match": "OIDC",
    "count": 2,
    "reason": "订阅套餐的功能名，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/packages/constants/src/subscription.ts",
    "match": "SAML",
    "count": 2,
    "reason": "订阅套餐的功能名，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "third-party-login",
    "path": "web/packages/constants/src/subscription.ts",
    "match": "SSO",
    "count": 2,
    "reason": "订阅套餐的功能名，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  }
]
```

Run: `node $P3TMP/kw.mjs add-exceptions $P3TMP/t1/exceptions.json && pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3`
Expected: `keywords: 28 rules, 22 exceptions, no hits.`、`alternatives or variants without a hit sample: 0`

- [ ] **Step 7: 测试**

在 `web/packages/constants/src/navigation.test.ts` 的 import 加 `GROUPED_PROFILE_SETTINGS`、`PROFILE_SETTINGS_TABS`
（`./settings/profile`），文件末尾加：

```ts
// The profile settings are the only way to the password change (security) and to the personal access
// tokens (api-tokens); the email notification preferences left with all email sending (M1 design 2.2).
describe("the profile settings", () => {
  it("keeps exactly these tabs", () => {
    expect(PROFILE_SETTINGS_TABS).toEqual(["general", "security", "preferences", "api-tokens"]);
  });

  it("shows every tab in exactly one sidebar group", () => {
    const grouped = Object.values(GROUPED_PROFILE_SETTINGS).flatMap((items) => items.map((item) => item.key));
    expect(grouped.toSorted()).toEqual([...PROFILE_SETTINGS_TABS].toSorted());
  });
});
```

Run: `pnpm --filter @plane/constants test 2>&1 | grep -E "Tests "`
Expected: `Tests  7 passed (7)`

- [ ] **Step 8: lint 上限**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected（原型，清理前）：`== web/apps/web` 下 3 条新的 `no-unused-vars`（`useInstance`、`SubscribeOutline`、`router`）和 1 条
`jsx-a11y(no-autofocus)`。前三条删掉；`no-autofocus` 是 `email.tsx` 里原有的 `autoFocus` 搬进了 `password.tsx`，不是新问题，保留。
原型的 `fix:format` 改了 4 个文件（`security.tsx`、`auth.service.ts`、`user.service.ts`、`authentication.helper.tsx`）。

Run: `node $P3TMP/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 630" && bash $P3TMP/caps.sh | tail -1`
Expected: `caps total: 781`（web 658 → 630）

- [ ] **Step 9: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 28 rules, 22 exceptions, no hits.` + `Tasks:    52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → `Tasks:    15 successful, 15 total`

Run: `make build-web 2>&1 | grep Tasks:` → `Tasks:    11 successful, 11 total`

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat 0d85d9d | tail -1`
Expected: 没有输出（与原型相同），或每处差异都能说明

Run: `git status --short | grep -v '^[MADR] '` → 没有输出

Run: `git diff --cached --shortstat` → `104 files changed, 522 insertions(+), 5503 deletions(-)`

knip 此时（原型 `0d85d9d`）：文件 80、导出 107、类型 60、枚举成员 0。

提交信息：

```
feat(web): sign-in and sign-up become one email and password form

Third-party sign-in, sign-in codes, the forgot, reset and set password
pages and the email check step before the password go (M1 design 2.2,
3.6). The sign-in and sign-up pages each render one email and password
form whose mode is the route's, and the email stays editable. CSRF
stays until M2.

The admin app's entry points, the "instance not ready" page and the
admin configuration types go; the instance store keeps the config and
the loading flag. The maintenance view for a failed instance request
stays.

All email sending goes (design 1.2): the email notification settings
tab, the marketing email consent in onboarding, and the change of login
email, which needed an emailed code (M1 design 3.13). The security page
only changes a password; the member table loses its login-method column.

A constants test pins the four profile settings tabs. Guard rules:
admin, third-party-login, auth-methods, email-sending; the plan
comparison copy that still names these methods is an exception until
billing goes later in this phase.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

### Task 2: 公开发布，连同评论的可见范围

M1 设计 9 节 P3 第 2 步，3.8。spec 2.4。原型提交 `381a34c`。

**Files**（原型 `381a34c`：49 个文件，删 9、改 40）

- 删除（9）：
  - `web/apps/web/core/components/comments/{comments.tsx,index.ts}`
  - `web/apps/web/core/components/project/publish-project/modal.tsx`
  - `web/apps/web/core/components/views/publish/{index.ts,use-view-publish.tsx}`
  - `web/apps/web/core/hooks/store/use-project-publish.ts`
  - `web/apps/web/core/services/project/project-publish.service.ts`
  - `web/apps/web/core/store/project/project-publish.store.ts`
  - `web/packages/types/src/publish.ts`
- 修改（40）：
  - `tools/keywords.json`
  - `turbo.json`
  - `web/apps/web/.env.example`
  - `web/apps/web/core/components/comments/card/{display.tsx,root.tsx}`
  - `web/apps/web/core/components/comments/{comment-create.tsx,quick-actions.tsx}`
  - `web/apps/web/core/components/editor/lite-text/{editor.tsx,toolbar.tsx}`
  - `web/apps/web/core/components/issues/header.tsx`
  - `web/apps/web/core/components/issues/issue-detail/issue-activity/{activity-comment-root.tsx,root.tsx}`
  - `web/apps/web/core/components/navigation/{project-actions-menu.tsx,tab-navigation-root.tsx,use-project-actions.ts}`
  - `web/apps/web/core/components/views/{quick-actions.tsx,view-list-item-action.tsx}`
  - `web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`
  - `web/apps/web/core/services/project/index.ts`
  - `web/apps/web/core/store/project/index.ts`
  - `web/packages/constants/src/{endpoints.ts,metadata.ts,workspace.ts}`
  - `web/packages/constants/src/issue/common.ts`
  - `web/packages/i18n/src/locales/en/{common.json,project-settings.json,work-item.json}`
  - `web/packages/i18n/src/locales/zh-CN/{common.json,project-settings.json,work-item.json}`
  - `web/packages/types/src/{enums.ts,inbox.ts,index.ts,views.ts}`
  - `web/packages/types/src/instance/base.ts`
  - `web/packages/types/src/issues/activity/issue_comment.ts`
  - `web/packages/types/src/issues/{issue.ts,issue_reaction.ts}`
  - `web/packages/types/src/project/projects.ts`
  - `web/packages/utils/src/project-views.ts`

**Interfaces**

- Consumes：Task 1 的守卫与 lint 上限。
- Produces：评论类型没有 `access`；`EIssueCommentAccessSpecifier`、`SPACE_BASE_URL`、`SPACE_BASE_PATH`、`SITES_URL`、
  `getPublishViewLink` 不存在；`ProjectRootStore` 没有 `publish`；守卫规则 `publish`、`publish-anchor`。

**Steps**

- [ ] **Step 1: 删除**

```
git rm -q web/apps/web/core/components/project/publish-project/modal.tsx \
  web/apps/web/core/components/views/publish/index.ts web/apps/web/core/components/views/publish/use-view-publish.tsx \
  web/apps/web/core/hooks/store/use-project-publish.ts \
  web/apps/web/core/services/project/project-publish.service.ts \
  web/apps/web/core/store/project/project-publish.store.ts \
  web/packages/types/src/publish.ts
```

（`comments/comments.tsx` 和 `comments/index.ts` 在 Step 3 按 `deadvocab` 删。）

- [ ] **Step 2: 按类型检查删到底**

Run: `pnpm exec turbo run check:types`
原型（`381a34c`）：第一次 22/23（`types` 包的 `index.ts` 还导出 `./publish`），之后按报错逐层删，第 6 次 23/23；
最后两处是 `utils/src/project-views.ts` 和 `views/view-list-item-action.tsx` 的 import。

要改的东西（`git show 381a34c -- <路径>`；原型脚本 `$P3TMP/t2/{app,access,types_publish}.py`）：

- 发布本身：`store/project/index.ts` 去掉 `publish` store 的导入、字段和构造；`services/project/index.ts` 去掉
  `project-publish.service`；`navigation/{project-actions-menu,tab-navigation-root}.tsx`、`navigation/use-project-actions.ts`、
  `workspace/sidebar/projects-list-item.tsx`、`views/{quick-actions,view-list-item-action}.tsx` 去掉"发布"菜单项和弹窗；
  只喂发布菜单项的 `isAdmin`（`tab-navigation-root.tsx`、`projects-list-item.tsx`）随之删除；
  `issues/header.tsx` 去掉"已发布"链接（`SPACE_APP_URL`、`anchor`）。
- space 的地址：`constants/src/endpoints.ts` 的 `SPACE_BASE_URL`、`SPACE_BASE_PATH`、`SITES_URL`；`constants/src/metadata.ts`
  的 `SPACE_SITE_*`（基点就无引用、以发布命名）；`utils/src/project-views.ts` 的 `getPublishViewLink`；`turbo.json` 和
  `.env.example` 的 `VITE_SPACE_BASE_URL`、`VITE_SPACE_BASE_PATH`；`RESTRICTED_URLS` 的 `spaces`。
- 公开页的类型：`types/src/index.ts` 去掉 `./publish`；`issues/issue.ts` 的 `IPublicIssue`、`TPublicIssuesResponse` 等；
  `issues/issue_reaction.ts` 的 `IIssuePublicReaction`、`IPublicVote`；`issues/activity/issue_comment.ts` 的 `access` 和
  `TIssuePublicComment`；`views.ts` 的 `anchor`、`IPublishedProjectView`、`TPublishViewSettings`、`TPublishViewDetails`；
  `project/projects.ts` 的 `anchor`；`inbox.ts` 的 `TInboxForm`、`TAnchors`、`TInboxIssueForm`；`instance/base.ts` 的
  `space_base_url`。**嵌套的类型体按范围删**（`TPublicIssueResponseResults` 里有内层的 `};`）。
- 评论的可见范围（3.8）：`types/src/enums.ts` 和 `constants/src/issue/common.ts` 各有一份 `EIssueCommentAccessSpecifier`，
  都删；`comments/card/{display,root}.tsx`、`comments/comment-create.tsx`、`comments/quick-actions.tsx`、
  `editor/lite-text/{editor,toolbar}.tsx`、`issue-detail/issue-activity/{activity-comment-root,root}.tsx` 去掉
  `access`、`showAccessSpecifier`、"内部 / 外部"开关和标记。

Expected（最后一次）：`Tasks:    23 successful, 23 total`

- [ ] **Step 3: 孤儿核对**

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t2.flat; diff $P3TMP/knip-t1.flat $P3TMP/knip-t2.flat | grep '^>'`
Expected: 没有输出

Run: `node $P3TMP/deadvocab.mjs 0d85d9d $P3TMP/knip-t0.flat $P3TMP/t2/rules.json`
Expected:

```
web/apps/web/core/components/comments/comments.tsx	publish(AccessSpecifier)
```

→ `git rm -q web/apps/web/core/components/comments/comments.tsx web/apps/web/core/components/comments/index.ts`
（`index.ts` 只导出它）。

Run: `node $P3TMP/symref.mjs orphaned 0d85d9d | head -1` → `exports used elsewhere at 0d85d9d and not now: 0`

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t2.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t2.txt 0d85d9d` → 没有输出

- [ ] **Step 4: 文案**

Run: `node $P3TMP/keyref.mjs orphaned 0d85d9d | head -1` → `keys referenced at 0d85d9d and not now: 3`
（`publish_project` 和 `issue.comments.switch` 下的两个，列表按父节点写成一行）

列表（`$P3TMP/t2/keys.txt`）：

```
common publish_project
work-item issue.comments.switch
# named after the published boards, unreferenced already at the base
work-item issue.vote
project-settings project_settings.features.intake.form
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t2/keys.txt | tail -1 && node $P3TMP/keycount.mjs`
Expected: `total: 4 keys in each locale`、`en: 1959 keys`、`zh-CN: 1959 keys`

- [ ] **Step 5: 守卫**

规则（`$P3TMP/t2/rules.json`）：

```json
[
  {
    "id": "publish",
    "phase": "M1/P3",
    "why": "公开发布（apps/space 没有迁入）：项目和视图的发布弹窗、\"已发布\"标记、发布设置的 store 和 service、space 应用的地址常量和环境变量、公开页的工作项 / 评论 / 表情 / 投票类型和收集箱表单，以及只为公开发布存在的评论可见范围（M1 设计 3.8）。不搜单独的 publish：草稿的\"发布为工作项\"保留",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "SPACE_(?:BASE|APP|SITE|TWITTER)|SITES_URL|space_base_url|deploy-boards|PublishProject|ProjectPublish|publish_project|PublishView|ViewPublish|PublishedProjectView|Public(?:Issue|Comment|Vote|Reaction)|IssuePublic|TInboxForm|TAnchors|AccessSpecifier|comments\\.switch",
      "flags": ""
    },
    "samples": {
      "hit": [
        "    \"VITE_SPACE_BASE_PATH\",",
        "  const SPACE_APP_URL =",
        "export const SPACE_SITE_NAME = \"Plane Publish | Make your Plane boards and roadmaps pubic with just one-click. \";",
        "export const SPACE_TWITTER_USER_NAME = \"planepowers\";",
        "export const SITES_URL = encodeURI(`${SPACE_BASE_URL}${SPACE_BASE_PATH}`);",
        "  space_base_url: string | undefined;",
        "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectID}/project-deploy-boards/`)",
        "import { PublishProjectModal } from \"../project/publish-project/modal\";",
        "import type { TProjectPublishLayouts, TProjectPublishSettings } from \"@plane/types\";",
        "            <div>{t(\"publish_project\")}</div>",
        "import { getPublishViewLink } from \"@plane/utils\";",
        "export const useViewPublish = (isPublished: boolean, isAuthorized: boolean) => ({",
        "export interface IPublishedProjectView extends Omit<IProjectView, \"rich_filters\"> {",
        "export interface IPublicIssue extends Pick<",
        "export type TIssuePublicComment = {",
        "import type { TIssueReaction, IIssuePublicReaction, IPublicVote } from \"./issue_reaction\";",
        "export type TInboxForm = {",
        "export type TAnchors = { [key: string]: string };",
        "import { EIssueCommentAccessSpecifier } from \"@plane/types\";",
        "              ? t(\"issue.comments.switch.public\")"
      ],
      "miss": [
        "  const [moveToIssue, setMoveToIssue] = useState(false);",
        "        <Tooltip label={access === EViewAccess.PUBLIC ? \"Public\" : \"Private\"}>",
        "export const WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS: IWorkspaceSidebarNavigationItem[] = ["
      ],
      "files": {
        "hit": ["web/apps/web/.env.example", "turbo.json", "web/packages/types/src/views.ts"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "publish-anchor",
    "phase": "M1/P3",
    "why": "项目和视图的 anchor 是公开发布页的地址标识，只作用于应用和类型包（M1 设计 7.4）。只搜属性的读取和声明（.anchor、anchor:）：编辑器里的 anchor（选区、链接）和 use-reload-confirmation 里的 HTML <a> 元素是保留代码",
    "files": {
      "source": "^web/(?:apps/web|packages/types)/.*\\.[cm]?[jt]sx?$",
      "flags": ""
    },
    "content": {
      "source": "\\.anchor\\b|\\banchors?\\??:",
      "flags": ""
    },
    "samples": {
      "hit": [
        "  const publishedURL = `${SPACE_APP_URL}/issues/${currentProjectDetails?.anchor}`;",
        "  anchors: TAnchors;",
        "  anchor?: string | null;"
      ],
      "miss": [
        "      const anchorElement = eventTarget.closest(\"a\") as HTMLAnchorElement;",
        "      // check if the event target is an anchor or a child of an anchor tag"
      ],
      "files": {
        "hit": ["web/apps/web/core/components/issues/header.tsx", "web/packages/types/src/project/projects.ts"],
        "miss": ["web/packages/editor/src/helpers/common.ts", "web/packages/i18n/src/locales/en/common.json"]
      }
    }
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t2/rules.json && pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: `stale exception: gantt-timeline  web/packages/types/src/publish.ts  "gantt" matches nothing now; delete it`
（P2 登记在公开发布类型上的例外，文件已删），退出码非零

Run: `node $P3TMP/kw.mjs del-exceptions gantt-timeline web/packages/types/src/publish.ts && pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3`
Expected: `1 exceptions deleted`、`keywords: 30 rules, 21 exceptions, no hits.`、`alternatives or variants without a hit sample: 0`

- [ ] **Step 6: lint 上限**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected（原型，清理前）：3 条新的 `no-unused-vars`：`Circle`、`NewTabOutline`（`issues/header.tsx`）、`isAdmin` → 删掉。
清理后 lintdiff 只剩 Task 1 说明过的 `no-autofocus`。

Run: `bash $P3TMP/caps.sh | tail -1` → `caps total: 781`（web 仍是 630，不改上限）

- [ ] **Step 7: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 30 rules, 21 exceptions, no hits.` + 52

Run: `make test-web 2>&1 | grep Tasks:` → 15；`make build-web 2>&1 | grep Tasks:` → 11

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat 381a34c | tail -1` → 没有输出，或每处差异都能说明

Run: `git diff --cached --shortstat` → `49 files changed, 118 insertions(+), 1319 deletions(-)`

knip 此时（`381a34c`）：文件 78、导出 107、类型 60。

提交信息：

```
feat(web): delete project and view publishing, and the comment access it needed

Publishing projects and views to the space app goes (apps/space was not
imported): the publish modals, the published link in the work-item
header, the publish menu items, the publish store, service and hook, the
space URL constants and env variables, and the public issue, comment,
vote and reaction types with the project and view anchor.

A comment's internal or external access only meant something on a
published board (M1 design 3.8), so the access switch, the badge, the
field and the enum go too, with the unused comments list that still
carried the switch.

Guard rules: publish and publish-anchor. The P2 exception on the publish
types goes with the file.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

### Task 3: Epic、团队、工作项类型、批量操作与多选，以及模板、工时、重复工作项、工作流、项目更新的空壳

M1 设计 9 节 P3 第 3 步：它们改同一组 store 和 hook 的分支，放在同一个任务（硬）；`work-items` 地址和共享的
`ProjectIssues` 类保留（2.2）；批量操作与多选按 3.9 整条删除。工作流和项目更新也放在这里（spec 第 3 节第 1 条）。
spec 2.5。原型提交 `7efb8ff`。

这是 P3 最大的 Task（266 个文件）。按下面的小步走，每一小步之后跑一次类型检查，出错就只回看这一小步。

**Files**（原型 `7efb8ff`：266 个文件，删 40、改 225、重命名 1）

- 删除（40）：
  - `web/apps/web/app/assets/empty-state/empty-updates-light.png`
  - `web/apps/web/app/assets/empty-state/epics/{epics-dark.webp,epics-light.webp,settings-dark.webp,settings-light.webp}`
  - `web/apps/web/app/assets/empty-state/project-settings/{updates-dark.png,updates-light.png}`
  - `web/apps/web/core/components/common/activity/{activity-block.tsx,activity-item.tsx,helper.tsx,user.tsx}`
  - `web/apps/web/core/components/core/multiple-select/{entity-select-action.tsx,group-select-action.tsx,index.ts,select-group.tsx}`
  - `web/apps/web/core/components/epic-modal/{index.ts,modal.tsx}`
  - `web/apps/web/core/components/issues/bulk-operations/{index.ts,root.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/empty-states/project-epic.tsx`
  - `web/apps/web/core/components/issues/{issue-type-switcher.tsx,layout-quick-actions.tsx}`
  - `web/apps/web/core/components/power-k/ui/pages/work-item-selection-page.tsx`
  - `web/apps/web/core/components/workflow/{index.ts,use-workflow-drag-n-drop.ts}`
  - `web/apps/web/core/hooks/store/use-multiple-select-store.ts`
  - `web/apps/web/core/hooks/{use-bulk-operation-status.ts,use-issue-properties.tsx,use-multiple-select.ts}`
  - `web/apps/web/core/store/multiple_select.store.ts`
  - `web/packages/constants/src/spreadsheet.ts`
  - `web/packages/propel/src/empty-state/assets/horizontal-stack/{epic.tsx,worklog.tsx}`
  - `web/packages/propel/src/empty-state/assets/vertical-stack/{epic.tsx,teamspace.tsx}`
  - `web/packages/propel/src/icons/project/epic-icon.tsx`
  - `web/packages/types/src/{activity.ts,epics.ts}`
  - `web/packages/types/src/issues/issue-property-values.ts`
  - `web/packages/types/src/project/activity.ts`
- 重命名（`git mv`）（1）：
  - `web/apps/web/core/components/workflow/state-option.tsx` → `web/apps/web/core/components/dropdowns/state/state-option.tsx`
- 修改（225）：
  - `tools/keywords.json`
  - `web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx`
  - `web/apps/web/core/components/common/{quick-actions-factory.tsx,quick-actions-helper.tsx}`
  - `web/apps/web/core/components/core/modals/{bulk-delete-issues-modal-item.tsx,existing-issues-list-modal.tsx}`
  - `web/apps/web/core/components/dropdowns/intake-state/base.tsx`
  - `web/apps/web/core/components/dropdowns/state/base.tsx`
  - `web/apps/web/core/components/home/widgets/recents/issue.tsx`
  - `web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx`
  - `web/apps/web/core/components/issues/attachment/{attachment-item-list.tsx,attachment-list-item.tsx,delete-attachment-modal.tsx}`
  - `web/apps/web/core/components/issues/{create-issue-toast-action-items.tsx,delete-issue-modal.tsx,filters.tsx,parent-issues-list-modal.tsx}`
  - `web/apps/web/core/components/issues/issue-detail-widgets/{action-buttons.tsx,issue-detail-widget-collapsibles.tsx,issue-detail-widget-modals.tsx,root.tsx}`
  - `web/apps/web/core/components/issues/issue-detail-widgets/attachments/{content.tsx,helper.tsx,quick-action-button.tsx,root.tsx,title.tsx}`
  - `web/apps/web/core/components/issues/issue-detail-widgets/links/{content.tsx,helper.tsx,quick-action-button.tsx,root.tsx,title.tsx}`
  - `web/apps/web/core/components/issues/issue-detail-widgets/relations/{content.tsx,helper.tsx,quick-action-button.tsx,root.tsx,title.tsx}`
  - `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/{content.tsx,display-filters.tsx,helper.ts,quick-action-button.tsx,root.tsx,title-actions.tsx,title.tsx}`
  - `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/issues-list/{list-group.tsx,list-item.tsx,root.tsx}`
  - `web/apps/web/core/components/issues/issue-detail/label/root.tsx`
  - `web/apps/web/core/components/issues/issue-detail/links/{create-update-link-modal.tsx,link-item.tsx,link-list.tsx,root.tsx}`
  - `web/apps/web/core/components/issues/issue-detail/{main-content.tsx,parent-select.tsx,subscription.tsx}`
  - `web/apps/web/core/components/issues/issue-detail/parent/{root.tsx,sibling-item.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/calendar/{base-calendar-root.tsx,calendar.tsx,day-tile.tsx,issue-block-root.tsx,issue-block.tsx,issue-blocks.tsx,quick-add-issue-actions.tsx,week-days.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/empty-states/index.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/{display-filters-selection.tsx,display-properties.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/{group-drag-overlay.tsx,utils.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/kanban/{base-kanban-root.tsx,block.tsx,blocks-list.tsx,default.tsx,kanban-group.tsx,swimlanes.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/kanban/headers/group-by-card.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/list/{base-list-root.tsx,block-root.tsx,block.tsx,blocks-list.tsx,default.tsx,list-group.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/list/headers/group-by-card.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/properties/all-properties.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/quick-add/button/{kanban.tsx,list.tsx,spreadsheet.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/quick-add/form/{calendar.tsx,kanban.tsx,list.tsx,root.tsx,spreadsheet.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/spreadsheet/{base-spreadsheet-root.tsx,issue-row.tsx,spreadsheet-header-column.tsx,spreadsheet-header.tsx,spreadsheet-table.tsx,spreadsheet-view.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/{assignee-column.tsx,attachment-column.tsx,created-on-column.tsx,cycle-column.tsx,due-date-column.tsx,header-column.tsx,label-column.tsx,link-column.tsx,module-column.tsx,priority-column.tsx,start-date-column.tsx,state-column.tsx,sub-issue-column.tsx,updated-on-column.tsx}`
  - `web/apps/web/core/components/issues/issue-modal/{base.tsx,draft-issue-layout.tsx,form.tsx,modal.tsx,provider.tsx}`
  - `web/apps/web/core/components/issues/issue-modal/components/{default-properties.tsx,parent-tag.tsx}`
  - `web/apps/web/core/components/issues/issue-modal/context/issue-modal-context.tsx`
  - `web/apps/web/core/components/issues/peek-overview/{issue-detail.tsx,root.tsx,view.tsx}`
  - `web/apps/web/core/components/issues/preview-card/root.tsx`
  - `web/apps/web/core/components/issues/relations/{issue-list-item.tsx,issue-list.tsx,properties.tsx}`
  - `web/apps/web/core/components/modals/work-item-level.tsx`
  - `web/apps/web/core/components/navigation/{tab-navigation-utils.ts,use-active-tab.ts}`
  - `web/apps/web/core/components/power-k/ui/modal/search-results-map.tsx`
  - `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/{commands.ts,root.tsx}`
  - `web/apps/web/core/components/project/create-project-modal.tsx`
  - `web/apps/web/core/components/projects/create/root.tsx`
  - `web/apps/web/core/components/work-item-filters/filters-hoc/shared.ts`
  - `web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx`
  - `web/apps/web/core/components/workspace/sidebar/project-navigation.tsx`
  - `web/apps/web/core/hooks/store/{use-issue-detail.ts,use-issues.ts}`
  - `web/apps/web/core/hooks/{use-group-dragndrop.ts,use-issue-layout-store.ts,use-issue-peek-overview-redirection.tsx,use-issues-actions.tsx,use-notification-preview.tsx}`
  - `web/apps/web/core/services/issue/{issue.service.ts,issue_activity.service.ts,issue_archive.service.ts,issue_attachment.service.ts,issue_comment.service.ts,issue_reaction.service.ts,work_item_version.service.ts}`
  - `web/apps/web/core/services/issue_filter.service.ts`
  - `web/apps/web/core/store/issue/archived/issue.store.ts`
  - `web/apps/web/core/store/issue/cycle/issue.store.ts`
  - `web/apps/web/core/store/issue/helpers/{base-issues.store.ts,issue-filter-helper.store.ts}`
  - `web/apps/web/core/store/issue/issue-details/{activity.store.ts,attachment.store.ts,comment.store.ts,issue.store.ts,link.store.ts,reaction.store.ts,root.store.ts,sub_issues.store.ts,sub_issues_filter.store.ts,subscription.store.ts}`
  - `web/apps/web/core/store/issue/module/issue.store.ts`
  - `web/apps/web/core/store/issue/profile/issue.store.ts`
  - `web/apps/web/core/store/issue/project-views/issue.store.ts`
  - `web/apps/web/core/store/issue/project/issue.store.ts`
  - `web/apps/web/core/store/issue/root.store.ts`
  - `web/apps/web/core/store/issue/workspace-draft/issue.store.ts`
  - `web/apps/web/core/store/issue/workspace/issue.store.ts`
  - `web/apps/web/core/store/{root.store.ts,router.store.ts,theme.store.ts}`
  - `web/apps/web/package.json`
  - `web/packages/constants/package.json`
  - `web/packages/constants/src/{endpoints.ts,fetch-keys.ts,index.ts,state.ts,workspace.ts}`
  - `web/packages/constants/src/issue/{common.ts,filter.ts,modal.ts}`
  - `web/packages/i18n/src/locales/en/{common.json,empty-state.json,navigation.json,project-settings.json,work-item.json,workspace-settings.json}`
  - `web/packages/i18n/src/locales/zh-CN/{common.json,empty-state.json,navigation.json,project-settings.json,work-item.json,workspace-settings.json}`
  - `web/packages/propel/src/empty-state/assets/{asset-registry.tsx,asset-types.ts}`
  - `web/packages/propel/src/empty-state/assets/horizontal-stack/index.ts`
  - `web/packages/propel/src/empty-state/assets/vertical-stack/index.ts`
  - `web/packages/propel/src/icons/project/index.ts`
  - `web/packages/propel/src/icons/registry.ts`
  - `web/packages/shared-state/src/store/work-item-filters/filter.store.ts`
  - `web/packages/types/src/{de-dupe.ts,enums.ts,home.ts,index.ts,issues.ts,search.ts,view-props.ts,workspace.ts}`
  - `web/packages/types/src/issues/activity/base.ts`
  - `web/packages/types/src/issues/{issue-identifier.ts,issue.ts}`
  - `web/packages/types/src/project/{index.ts,projects.ts}`
  - `web/packages/types/src/workspace-draft-issues/base.ts`
  - `web/packages/utils/src/datetime.ts`
  - `web/packages/utils/src/work-item/{base.ts,modal.ts}`

**Interfaces**

- Consumes：Task 2 的结果。
- Produces：
  - `EIssueServiceType`、`TIssueServiceType` 不存在；工作项的 service 和详情 store 不带 service 类型参数，
    `useIssueDetail()` 不带参数；`WorkItemVersionService` 的两个地址写成 `/work-items/`，其余是 `/issues/`；
  - `EIssuesStoreType` 没有 `EPIC`、`TEAM`、`TEAM_VIEW`、`TEAM_PROJECT_WORK_ITEMS`；`TIssue` 没有 `is_epic`、`type_id`；
    显示属性和筛选项没有 `issue_type`，分组方式没有 `team_project`；
  - 工作项弹窗上下文只剩 `allowedProjectIds`、`selectedParentIssue`；
  - 没有多选 store、`selectionHelpers`、批量接口；
  - `StateOption`（`components/dropdowns/state/state-option.tsx`）只有 `option`、`className`；
  - 守卫规则 `epics`、`teamspaces`、`work-item-types`、`bulk-operations`、`templates-worklogs`、`workflows-project-updates`。

**Steps**

- [ ] **Step 1: Epic 的 `isEpic` 与 Epic 的组件**

`isEpic` 在 CE 中恒为 `false`，贯穿 59 个文件：删掉这个 prop / 参数，把每个 `isEpic ? A : B` 化简为 `B`、
`if (isEpic) …` 整段删除，写出被选中那一支的原文（原型脚本 `$P3TMP/t3/epic1.py`，用 `t3/cond.py` 的"恒为假"化简；
它化简不了的由 `epic2.py`、`epic2b.py` 手工处理）。然后：

```
git rm -q -r web/apps/web/core/components/epic-modal
git rm -q web/apps/web/core/components/issues/issue-layouts/empty-states/project-epic.tsx \
  web/packages/propel/src/empty-state/assets/horizontal-stack/epic.tsx \
  web/packages/propel/src/empty-state/assets/vertical-stack/epic.tsx \
  web/packages/propel/src/icons/project/epic-icon.tsx web/packages/types/src/epics.ts \
  web/apps/web/app/assets/empty-state/epics/{epics-dark,epics-light,settings-dark,settings-light}.webp
```

`TIssue.is_epic`、首页和工作区草稿里的 `is_epic`、propel 的注册表和导出同步删除。

- [ ] **Step 2: 工作项的 service 类型整个删除**

Epic 走后，每个工作项 service（`IssueService`、`IssueActivityService`、`IssueCommentService`、关联、子工作项、附件、链接、
表情等）和工作项详情 store 都用 `EIssueServiceType.ISSUES` 构造，描述历史的 `WorkItemVersionService` 用 `WORK_ITEMS`。
删除构造参数、字段和 `${this.serviceType}` 地址段，**写出选中的文本**：工作项的地址是 `/issues/`，`WorkItemVersionService`
的两个地址是 `/work-items/`，`retrieveWithIdentifier` 本来就是 `/work-items/`（M1 设计 2.2：不能一律改成 `issues`）。
`epicDetail`、`epicService`、`projectEpics` 删除；`useIssueDetail(serviceType?)` → `useIssueDetail()`，约 75 个文件
（原型脚本 `svc.py`、`svc2.py`、`svc2b.py`、`literals.py`）。最后删 `types/src/issues/issue.ts` 的 `EIssueServiceType` 和
`TIssueServiceType`，让类型检查列出剩下的引用。

Run: `git diff HEAD | grep -c -F '${"'` → `0`（原型第一次化简留下了 8 处 `${"sub-issues"}` 之类，见"原型的教训"第 2 条）

- [ ] **Step 3: 团队、工作项类型、模板**

- 团队（`team.py`、`team2.py`）：`EIssuesStoreType` 的 `TEAM`、`TEAM_VIEW`、`TEAM_PROJECT_WORK_ITEMS`、`EPIC` 和工作项
  根 store 里对应的 store 实例；`router.store.ts` 的 `teamspaceId`、`epicId`；分组方式 `team_project`（常量、类型、
  筛选、看板和列表的分组）；团队、Epic 的文件资源类型；propel 的 `teamspace` 插图。共享的 `ProjectIssues` 保留。
  `RESTRICTED_URLS` 去掉 `epics`、`epic`。常量包里一个 `no-duplicate-enum-values` 警告随之消失。
- 工作项类型（`tmpl.py`、`types2.py`）：`type_id`（`TIssue`、草稿、搜索、去重、动态）、`IssueIdentifier` 的
  `issueTypeId`、`issue-type-switcher.tsx`（它只渲染 `IssueIdentifier`，调用方直接写
  `<IssueIdentifier projectId={issue.project_id} …>`）、显示属性和筛选项 `issue_type`、`use-issue-properties.tsx`
  （`useWorkItemProperties`，CE 中为空，删掉它的两处调用）、`types/src/issues/issue-property-values.ts`。
- 模板：工作项弹窗的上下文（`issue-modal/context`）只留 `allowedProjectIds`、`selectedParentIssue`，模板的字段和
  provider 里的空函数删除；草稿的"移动到项目"（`handleMoveToProjects` → `moveIssue`）保留，只删企业版属性值的调用。

```
git rm -q web/apps/web/core/components/issues/issue-type-switcher.tsx web/apps/web/core/hooks/use-issue-properties.tsx \
  web/packages/types/src/issues/issue-property-values.ts \
  web/packages/propel/src/empty-state/assets/vertical-stack/teamspace.tsx
```

- [ ] **Step 4: 批量操作与工作项多选（M1 设计 3.9）**

```
git rm -q -r web/apps/web/core/components/core/multiple-select web/apps/web/core/components/issues/bulk-operations
git rm -q web/apps/web/core/hooks/store/use-multiple-select-store.ts web/apps/web/core/hooks/use-bulk-operation-status.ts \
  web/apps/web/core/hooks/use-multiple-select.ts web/apps/web/core/store/multiple_select.store.ts \
  web/packages/constants/src/spreadsheet.ts
```

然后（`bulk.py`、`bulk2.py`）：根 store 的两处 `MultipleSelectStore` 构造和字段；列表、看板、表格布局里逐层透传的
`selectionHelpers`、`isSelectionActive` 等 prop；组头和行上的选择框（`projectId && canSelectIssues && (...)` 这类守卫一起删）；
表格 13 列上只在选中时生效的 `selected-issue-row` 样式；`IssueService.bulkOperations` 和 8 个工作项 store 的
`bulkUpdateProperties`；`TBulkOperationsPayload`；`SPREADSHEET_SELECT_GROUP`。因此不再被读的 `canEditProperties`
（列表的 `HeaderGroupByCard`、表格的 `SpreadsheetHeader`）删除。**保留**命令面板的 `BulkDeleteIssuesModal`（社区版接口）
和表单的多选下拉框。

- [ ] **Step 5: 工时、重复工作项、工作流、项目更新**

- 工时（`worklog.py`、`worklog_rest.py`、`notif.py`、`notif2.py`）：动态里的 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`
  分支；通知卡片的 `estimate_time`；`utils/src/datetime.ts` 里三个"分钟"工具函数（两个只有通知卡片用，第三个基点就无引用）；
  propel 的 worklog 插图。`git rm -q web/packages/propel/src/empty-state/assets/horizontal-stack/worklog.tsx`
- 工作流（`wf.py`、`wf2.py`）：列表和看板中 CE 为空操作的 `useWorkFlowFDragNDrop`；
  `git mv web/apps/web/core/components/workflow/state-option.tsx web/apps/web/core/components/dropdowns/state/state-option.tsx`，
  props 只留 `option`、`className`；`git rm -q web/apps/web/core/components/workflow/{index.ts,use-workflow-drag-n-drop.ts}`；
  状态下拉框的 `alwaysAllowStateChange`、`filterAvailableStateIds`、`isForWorkItemCreation` 删除（两个调用方同步）。
- 项目动态：`is_{project_updates,epic,workflow,time_tracking,issue_type}_enabled` 的分支和图标。
- 项目更新的图片：
  `git rm -q web/apps/web/app/assets/empty-state/project-settings/{updates-dark,updates-light}.png web/apps/web/app/assets/empty-state/empty-updates-light.png`
- 只剩一个分支的结构收掉（`degen.py`）。

- [ ] **Step 6: 类型检查**

Run: `pnpm exec turbo run check:types`
原型（`7efb8ff`）：Step 1–5 做完后第一次 22/23（`types` 包的 `index.ts` 还导出已删的文件），然后 web 报 10 处，
按报错删（`fix1.py`）后 23/23。
Expected（最后一次）：`Tasks:    23 successful, 23 total`

- [ ] **Step 7: 孤儿核对**

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t3.flat; diff $P3TMP/knip-t2.flat $P3TMP/knip-t3.flat | grep '^>'`
Expected（原型）：列表布局的 `isSubGrouped` 成为未使用的导出（它唯一的读取方是批量操作的实体）→ 删掉它，
连同只有它的参数类型在用的 `TGroupedIssues` import（`knip1.py`）。

Run: `node $P3TMP/deadvocab.mjs 381a34c $P3TMP/knip-t0.flat $P3TMP/t3/rules.json`
Expected:

```
web/apps/web/core/components/common/activity/helper.tsx	epics(epic) work-item-types(is_issue_type_enabled) templates-worklogs(TimeTracking) workflows-project-updates(project_updates)
web/apps/web/core/components/issues/layout-quick-actions.tsx	epics(EPIC)
web/apps/web/core/components/power-k/ui/pages/work-item-selection-page.tsx	epics(Epic)
```

→ 连同只有它们在用的代码删（原型脚本 `deadchain.py`）：

```
git rm -q -r web/apps/web/core/components/common/activity
git rm -q web/apps/web/core/components/issues/layout-quick-actions.tsx \
  web/apps/web/core/components/power-k/ui/pages/work-item-selection-page.tsx \
  web/packages/types/src/activity.ts web/packages/types/src/project/activity.ts
```

`common/quick-actions-helper.tsx` 的 `useLayoutMenuItems`、`UseLayoutMenuItemsProps` 和两个工厂
`createOpenInNewTab`、`createCopyLayoutLinkMenuItem`；`types/src/index.ts`、`types/src/project/index.ts` 的两行导出。
`TIssueSearchResponse` 保留导出（它是 `TSearchResponse` 的组成部分，见 spec 第 4 节）。

Run: `node $P3TMP/symref.mjs orphaned 381a34c`
Expected（删之前）：`MARKETING_PLANE_ONE_PAGE_LINK`（只有批量操作的升级提示用）→ 删；删完后只剩
`web/packages/types/src/search.ts	TIssueSearchResponse` 一行。

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t3.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t3.txt 381a34c` → 没有输出
（Epic 和项目更新的 7 张图在 Step 1、5 已按"以被删功能命名"删掉）

Run: `git diff HEAD | grep -c -F '${"'` → `0`

- [ ] **Step 8: 文案**

Run: `node $P3TMP/keyref.mjs orphaned 381a34c | head -1` → `keys referenced at 381a34c and not now: 16`
（叶子键；列表按父节点写成 `keys.txt` 第一段的 8 行，加上工作项选择页删掉后的 `command_k.empty_state.search.title`，
写在 `keys-dead.txt`）

列表（`$P3TMP/t3/keys.txt` 和 `$P3TMP/t3/keys-dead.txt`）：

```
# referenced at the task's base and not after the deletions (keyref.mjs orphaned)
common common.team_project
common common.epic
work-item issue.remove.label
work-item issue.display.properties.work_item_count
work-item sub_work_item.empty_state.list_filters.title
work-item sub_work_item.empty_state.list_filters.description
work-item bulk_operations
work-item epic
# named after Epic, teamspaces, work-item types, templates, worklogs, recurring work items, workflows and
# project updates; unreferenced already at the base
common activity_empty_state.no_worklogs
common time_tracking
common time_tracking_description
common common.worklogs
common common.templates
common common.members_and_teamspaces
common common.recurring_work_items
empty-state project_empty_state.epics
empty-state project_empty_state.epic_work_items
empty-state workspace_empty_state.archive_epics
empty-state settings_empty_state.work_item_types
empty-state settings_empty_state.work_item_type_properties
empty-state settings_empty_state.templates
empty-state settings_empty_state.recurring_work_items
empty-state settings_empty_state.worklogs
empty-state settings_empty_state.template_setting
navigation sidebar.epics
project-settings project_settings.workflows
project-settings project_settings.work_item_types
project-settings project_settings.features.time_tracking
project-settings project_settings.templates
project-settings project_settings.project_updates
work-item issue.display.properties.issue_type
work-item recurring_work_items
workspace-settings workspace_settings.settings.worklogs
workspace-settings workspace_settings.settings.templates
common common.project_updates
```

```
# T3 rewrite: orphaned by the deletion of the base-dead work-item selection page (power-k)
navigation command_k.empty_state.search.title
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t3/keys.txt | tail -1 && node $P3TMP/i18n-del-list.mjs $P3TMP/t3/keys-dead.txt | tail -1 && node $P3TMP/keycount.mjs`
Expected: `total: 35 keys in each locale`、`total: 1 keys in each locale`、`en: 1804 keys`、`zh-CN: 1804 keys`

- [ ] **Step 9: 守卫**

规则（`$P3TMP/t3/rules.json`）：

```json
[
  {
    "id": "epics",
    "phase": "M1/P3",
    "why": "Epic（企业版）：Epic 的 store 类型、服务类型、详情 store、弹窗、空状态、导航项、插图和图标，以及贯穿工作项布局的 isEpic 参数和 is_epic 字段（M1 设计 2.2、7.4）。区分大小写：propel 的 ImagePicker 等名字里有 ePic",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "(?<![dD])epic|Epic|EPIC",
      "flags": ""
    },
    "samples": {
      "hit": [
        "    issue?.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES",
        "export interface EpicModalProps {"
      ],
      "miss": [
        "  const imagePickerRef = useRef<HTMLDivElement>(null);",
        "  const { issues: projectIssues } = useIssues(EIssuesStoreType.PROJECT);"
      ],
      "files": {
        "hit": ["web/apps/web/core/hooks/store/use-issues.ts", "web/packages/i18n/src/locales/en/work-item.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "teamspaces",
    "phase": "M1/P3",
    "why": "团队（企业版）：团队、团队视图、团队下项目的工作项 store 类型和实例、路由参数 teamspaceId、分组方式 team_project、团队的文件资源类型（M1 设计 7.4）。不搜普通英文单词 team（登录页、邀请页的文案）",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "teamspace|team_view|team_project|team_space|EIssuesStoreType\\.TEAM\\b",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "  entityType: EIssuesStoreType; // entity type (project, cycle, workspace, teamspace, etc)",
        "  | EIssuesStoreType.TEAM_VIEW",
        "    EIssuesStoreType.TEAM_PROJECT_WORK_ITEMS,",
        "  TEAM_SPACE_DESCRIPTION = \"TEAM_SPACE_DESCRIPTION\",",
        "  | EIssuesStoreType.TEAM"
      ],
      "miss": [
        "            ? \"Create an account to start managing work with your team.\"",
        "      id: \"invite-team\","
      ],
      "files": {
        "hit": ["web/apps/web/core/store/issue/root.store.ts", "web/packages/i18n/src/locales/en/common.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "work-item-types",
    "phase": "M1/P3",
    "why": "工作项类型（企业版）：type_id 字段、IssueIdentifier 的 issueTypeId、类型切换器、显示属性和筛选项 issue_type、自定义属性的值和动态、CE 中为空的 useWorkItemProperties（M1 设计 7.4）",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "type_id\\b|issueTypeId|IssueTypeSwitcher|IssueTypeIdentifier|work_item_type|issue_type\\b|ISSUE_ADDITIONAL_PROPERTIES|useWorkItemProperties|IssuePropertyValue|is_issue_type_enabled",
      "flags": ""
    },
    "samples": {
      "hit": [
        "          issueTypeId={issue.type_id}",
        "import { IssueTypeSwitcher } from \"@/components/issues/issue-type-switcher\";",
        "export type TIssueTypeIdentifier = {",
        "    \"work_item_types\": {",
        "              {displayProperties && (displayProperties.key || displayProperties.issue_type) && (",
        "      activity_type: \"ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY\";",
        "import { useWorkItemProperties } from \"@/hooks/use-issue-properties\";",
        "import type { ISearchIssueResponse, TIssue, TIssuePropertyValues, TIssuePropertyValueErrors } from \"@plane/types\";",
        "  is_issue_type_enabled: ToDoOutline,"
      ],
      "miss": [
        "      activity_type: \"DEFAULT\";",
        "import type { TIssue, TIssuePriorities } from \"@plane/types\";"
      ],
      "files": {
        "hit": ["web/apps/web/core/components/issues/issue-modal/form.tsx", "web/packages/i18n/src/locales/en/empty-state.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "bulk-operations",
    "phase": "M1/P3",
    "why": "批量操作与工作项多选（M1 设计 3.9）：批量接口、升级提示条、多选 store 和 hooks、组头和行上的选择框、逐层透传的 selectionHelpers、只在选中时生效的 selected-issue-row 样式。不在其中：命令面板的批量删除（BulkDeleteIssuesModal，Plane 社区版接口）、表单的多选下拉框（multiple）",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "bulk[-_]?operation|multipleselect|selectionhelper|selected-issue-row|spreadsheet_select_group",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "export const IssueBulkOperationsRoot = observer(function IssueBulkOperationsRoot(props: Props) {",
        "export const MultipleSelectEntityAction = observer(function MultipleSelectEntityAction(props: Props) {",
        "import type { TSelectionHelper } from \"@/hooks/use-multiple-select\";",
        "          \"rounded-none px-page-x text-left group-[.selected-issue-row]:bg-accent-primary/5 group-[.selected-issue-row]:hover:bg-accent-primary/10\",",
        "import { SPREADSHEET_SELECT_GROUP } from \"@plane/constants\";"
      ],
      "miss": [
        "import { BulkDeleteIssuesModalItem } from \"./bulk-delete-issues-modal-item\";",
        "                    multiple"
      ],
      "files": {
        "hit": ["web/apps/web/core/components/issues/issue-layouts/list/block.tsx", "web/packages/i18n/src/locales/en/work-item.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "templates-worklogs",
    "phase": "M1/P3",
    "why": "工作项模板、项目模板、工时记录、定时（重复）工作项（企业版，M1 设计 2.2）：弹窗上下文中 CE 固定为空的模板字段、操作动态的 WORKLOG 和 is_time_tracking_enabled 分支、worklog 插图、recurring_work_items 文案。不搜单独的 template：gridTemplateColumns 是样式",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "workItemTemplateId|isApplyingTemplate|handleTemplateChange|templateId|WORKLOG|[wW]orklog|is_time_tracking_enabled|time_tracking|TimeTracking|recurring_work_items|[rR]ecurring[ _]?[wW]ork",
      "flags": ""
    },
    "samples": {
      "hit": [
        "  workItemTemplateId: string | null;",
        "  isApplyingTemplate: boolean;",
        "  handleTemplateChange: (props: THandleTemplateChangeProps) => Promise<void>;",
        "  templateId?: string;",
        "      activity_type: \"WORKLOG\";",
        "        title: \"Time Tracking + Worklogs\",",
        "  is_time_tracking_enabled: TimeTrackingOutline,",
        "  TimeTrackingOutline,",
        "    \"recurring_work_items\": \"Recurring work items\","
      ],
      "miss": [
        "            gridTemplateColumns: `repeat(3, 1fr)`,",
        "  const { issues: draftIssues } = useIssues(EIssuesStoreType.WORKSPACE_DRAFT);"
      ],
      "files": {
        "hit": ["web/apps/web/core/components/issues/issue-modal/provider.tsx", "web/packages/i18n/src/locales/en/work-item.json"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  },
  {
    "id": "workflows-project-updates",
    "phase": "M1/P3",
    "why": "自定义工作流和项目动态（企业版）：列表和看板中 CE 为空操作的 useWorkFlowFDragNDrop、状态选项只为工作流声明的参数、项目操作动态的开关分支、SWR 键和文案。不搜单独的 workflow：工作区保留地址名单里的 workflow、workflows 留给 M3 与后端对齐",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "WorkFlow|isWorkflow|workflowDisabledSource|components/workflow|is_workflow_enabled|_WORKFLOWS?\\b|WORKFLOW_|project_updates|EUpdateStatus",
      "flags": ""
    },
    "samples": {
      "hit": [
        "import { useWorkFlowFDragNDrop } from \"@/components/workflow\";",
        "  const { workflowDisabledSource, isWorkflowDropDisabled, handleWorkFlowState, getIsWorkflowWorkItemCreationDisabled } =",
        "  workflowDisabledSource?: string;",
        "import { StateOption } from \"@/components/workflow\";",
        "  is_workflow_enabled: GitBranchOutline,",
        "export const PROJECT_WORKFLOWS = (projectId: string, projectRole: EUserPermissions | undefined) =>",
        "export const WORKSPACE_WORKFLOW_STATES = (workspaceSlug: string) =>",
        "  is_project_updates_enabled: SubscribeOutline,",
        "export enum EUpdateStatus {"
      ],
      "miss": [
        "  \"workflow\",",
        "import { StateOption } from \"./state-option\";"
      ],
      "files": {
        "hit": ["web/apps/web/core/components/issues/issue-layouts/list/list-group.tsx", "web/packages/constants/src/fetch-keys.ts"],
        "miss": ["web/apps/web/app/assets/logo.svg", "docs/v0/M1-frontend-trim/M1-design.md"]
      }
    }
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t3/rules.json && pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected（原型）：计费文案里的 Epic、Teamspace、Worklog 命中；两条 P2 例外过期：
`stale exception: analytics  web/packages/types/src/epics.ts  "Analytics" matches nothing now; delete it`、
`stale exception: estimates  web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx  "estimat" matches nothing now; delete it`。

例外（`$P3TMP/t3/exceptions.json`，Task 5 随计费文案删除）：

```json
[
  {
    "rule": "epics",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "Epic",
    "count": 2,
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "epics",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "epic",
    "count": 2,
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "teamspaces",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "Teamspace",
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "teamspaces",
    "path": "web/packages/constants/src/subscription.ts",
    "match": "Teamspace",
    "reason": "订阅套餐的功能名，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  },
  {
    "rule": "templates-worklogs",
    "path": "web/apps/web/core/components/workspace/billing/comparison/plans.tsx",
    "match": "Worklog",
    "reason": "计费对比表的营销文案，随计费在本 Phase 删除（P3 plan 的计费 Task）",
    "until": "M1/P4"
  }
]
```

Run:

```
node $P3TMP/kw.mjs del-exceptions analytics web/packages/types/src/epics.ts
node $P3TMP/kw.mjs del-exceptions estimates web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx
node $P3TMP/kw.mjs add-exceptions $P3TMP/t3/exceptions.json
pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3
```

Expected: `keywords: 36 rules, 24 exceptions, no hits.`、`alternatives or variants without a hit sample: 0`

- [ ] **Step 10: lint 上限**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected（原型，清理前）：18 条新的 `no-unused-vars`（图标、`router`、三处 `projectId`、`ALL_ISSUES`、`isSubGrouped`、
`is_list`、两处 `canEditProperties`、`useIssueModal`、`response`、`cn`、`useParams` 等）→ 全部删掉（`lint1.py`）。
`day-tile.tsx` 的 `exhaustive-deps` 会出现在列表里，只因为它的消息文本随依赖名变了，数量不变，不改
（固定节奏第 6 步的已知条目）。

Run:

```
node $P3TMP/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 617"
node $P3TMP/pkg.mjs web/packages/constants/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 1"
bash $P3TMP/caps.sh | tail -1
```

Expected: `caps total: 767`（web 630 → 617，constants 2 → 1）

- [ ] **Step 11: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 36 rules, 24 exceptions, no hits.` + 52

Run: `make test-web 2>&1 | grep Tasks:` → 15；`make build-web 2>&1 | grep Tasks:` → 11

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat 7efb8ff | tail -1` → 没有输出，或每处差异都能说明

Run: `git diff --cached --shortstat` → `266 files changed, 852 insertions(+), 5562 deletions(-)`

knip 此时（`7efb8ff`）：文件 72、导出 105、类型 52。

提交信息：

```
feat(web): delete epics, teamspaces, work-item types, bulk operations and the other enterprise work-item shells

Epics go, and with them the isEpic flag threaded through the layouts and
the issue service type: once epics are gone every work-item service is
built for issues and the description-version service for work items, so
the enum goes and the /work-items/ addresses are written out (M1 design
2.2). Teamspaces, team views and the team_project grouping go; the
shared ProjectIssues class stays.

Work-item types, templates, worklogs, recurring work items, custom
workflows and project updates were empty or constant in this edition;
their props, modal context fields, activity branches, copy and images
go. The state option moves next to the state dropdown with the two
props it uses.

Bulk operations and work-item multi-select go as one chain (M1 design
3.9): the banner, the multi-select store and hooks, the checkboxes and
the selection helpers passed down every layout. The command palette's
bulk delete and the multi-select dropdowns stay.

Dead files that still carried this vocabulary go with what only they
used. Guard rules: epics, teamspaces, work-item-types, bulk-operations,
templates-worklogs, workflows-project-updates; the plan comparison copy
that names them is an exception until billing goes.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

### Task 4: AI 助手（含 Plane AI）、Unsplash 封面、遥测

M1 设计 9 节 P3 第 4 步。spec 2.6。原型提交 `3d08bf5`。

**Files**（原型 `3d08bf5`：34 个文件，删 8、改 26）

- 删除（8）：
  - `web/apps/web/core/components/core/modals/gpt-assistant-popover.tsx`
  - `web/apps/web/core/components/workspace-notifications/notification-app-sidebar-option.tsx`
  - `web/apps/web/core/components/workspace/sidebar/{user-menu-item.tsx,user-menu.tsx}`
  - `web/apps/web/core/services/ai.service.ts`
  - `web/packages/propel/src/icons/ai-icon.tsx`
  - `web/packages/propel/src/icons/sub-brand/pi-chat.tsx`
  - `web/packages/types/src/ai.ts`
- 修改（26）：
  - `tools/keywords.json`
  - `web/apps/web/core/components/core/image-picker-popover.tsx`
  - `web/apps/web/core/components/inbox/modals/create-modal/issue-description.tsx`
  - `web/apps/web/core/components/issues/issue-modal/components/description-editor.tsx`
  - `web/apps/web/core/components/issues/issue-modal/form.tsx`
  - `web/apps/web/core/components/project/create/header.tsx`
  - `web/apps/web/core/components/project/form.tsx`
  - `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`
  - `web/apps/web/core/hooks/use-workspace-paths.ts`
  - `web/apps/web/core/services/file.service.ts`
  - `web/apps/web/helpers/cover-image.helper.ts`
  - `web/apps/web/package.json`
  - `web/packages/constants/src/tab-indices.ts`
  - `web/packages/i18n/src/locales/en/{common.json,navigation.json,workspace-settings.json}`
  - `web/packages/i18n/src/locales/zh-CN/{common.json,navigation.json,workspace-settings.json}`
  - `web/packages/propel/src/icons/{index.ts,registry.ts}`
  - `web/packages/propel/src/icons/sub-brand/index.ts`
  - `web/packages/types/src/{index.ts,workspace.ts}`
  - `web/packages/types/src/instance/base.ts`
  - `web/packages/types/src/project/projects.ts`

**Interfaces**

- Consumes：Task 3 的结果。
- Produces：`IInstanceConfig` 没有 `has_llm_configured`、`has_unsplash_configured`；`ImagePickerPopover` 没有 `control`
  prop 和泛型，标签页是常量 `[images, upload]`；`FileService` 没有 `getUnsplashImages`；守卫规则 `ai-assistant`、
  `unsplash-telemetry`。

**Steps**

- [ ] **Step 1: 删除**

```
git rm -q web/apps/web/core/components/core/modals/gpt-assistant-popover.tsx web/apps/web/core/services/ai.service.ts \
  web/packages/types/src/ai.ts web/packages/propel/src/icons/ai-icon.tsx
```

- [ ] **Step 2: AI 助手**

原型脚本 `$P3TMP/t4/ai_unsplash.py`、`imports.py`：

- `issues/issue-modal/components/description-editor.tsx`：AI 按钮那一行、"I'm feeling lucky"、GPT 弹窗、`AIService`、
  `useInstance`，以及 `issueName`、`gptAssistantModal`、`setGptAssistantModal`、`handleGptAssistantClose` 四个 prop；
  加载骨架里给两个 AI 按钮占位的元素；删完只剩一个子元素的 fragment 收掉。`issues/issue-modal/form.tsx` 和
  `inbox/modals/create-modal/issue-description.tsx` 去掉对应的状态和 prop。
- `constants/src/tab-indices.ts` 的 `feeling_lucky`；`types/src/index.ts` 的 `./ai`；`types/src/instance/base.ts` 的
  `has_llm_configured`；propel 的 `icons/index.ts`、`icons/registry.ts` 去掉 `AiIcon`。

- [ ] **Step 3: Unsplash 与遥测**

- `core/image-picker-popover.tsx`：Unsplash 标签页和**每次打开都发出的** `useSWR` 请求（`/api/unsplash/`）；
  标签页变成模块级常量 `TAB_OPTIONS`（只读的 `images`、`upload` 两项）；`control` prop 只喂 Unsplash 搜索框，
  连同组件的泛型删除，三个调用方（`project/create/header.tsx`、`project/form.tsx`、
  `settings/profile/content/pages/general/form.tsx`）同步。
- `helpers/cover-image.helper.ts` 的 `"unsplash"` 类型（绝对地址的封面仍按原样显示）；`services/file.service.ts` 的
  `getUnsplashImages` 和 `UnSplashImage` 类型；`types/src/instance/base.ts` 的 `has_unsplash_configured`。
- 遥测：代码里已经没有东西（`is_telemetry_enabled` 随 Task 1 的 `IInstance` 删除），只加守卫规则。

Run: `pnpm exec turbo run check:types`
Expected: `Tasks:    23 successful, 23 total`（原型第一次就通过）

- [ ] **Step 4: 孤儿核对与 Plane AI**

Run: `node $P3TMP/deadvocab.mjs 7efb8ff $P3TMP/knip-t0.flat $P3TMP/t4/rules.json`
Expected:

```
web/apps/web/core/components/workspace/sidebar/user-menu.tsx	ai-assistant(PiChat)
```

侧边栏的 `user-menu.tsx` 是"Plane AI"（pi-chat）剩下的唯一入口，基点就没人用。连同只有它在用的
`user-menu-item.tsx`、只有后者在用的 `notification-app-sidebar-option.tsx`、propel 的 `PiChatLogo`、
`use-workspace-paths.ts` 的 `isAiPath` 一起删（`pichat.py`）：

```
git rm -q web/apps/web/core/components/workspace/sidebar/user-menu.tsx \
  web/apps/web/core/components/workspace/sidebar/user-menu-item.tsx \
  web/apps/web/core/components/workspace-notifications/notification-app-sidebar-option.tsx \
  web/packages/propel/src/icons/sub-brand/pi-chat.tsx
```

propel 的 `sub-brand/index.ts`、`icons/registry.ts` 同步。

Run: `node $P3TMP/symref.mjs orphaned 7efb8ff`
Expected（删之前）：`IProjectLite`（`types/src/project/projects.ts`）、`IWorkspaceLite`（`types/src/workspace.ts`），
只有 `IGptResponse` 用它们 → 删（`orphans.py`）。

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t4.flat; diff $P3TMP/knip-t3.flat $P3TMP/knip-t4.flat | grep '^>'` → 没有输出

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t4.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t4.txt 7efb8ff` → 没有输出

- [ ] **Step 5: 文案**

Run: `node $P3TMP/keyref.mjs orphaned 7efb8ff | head -1` → `keys referenced at 7efb8ff and not now: 5`
（`workspace_dashboards`、`sidebar.{your_work,home,drafts,pi_chat}`，都只有被删的用户菜单在用）

另外用 `node $P3TMP/keyref.mjs named 'AI|intelligence|pi_chat'` 找出以 AI 命名、基点就无引用的键。列表
（`$P3TMP/t4/keys.txt`、`$P3TMP/t4/keys2.txt`）：

```
# "Plane AI": referenced only by the unused sidebar user menu, and the settings copy of the AI page
common pi_chat
navigation sidebar.pi_chat
workspace-settings workspace_settings.settings.plane-intelligence
```

```
# referenced only by the deleted sidebar user menu (keyref.mjs orphaned)
common workspace_dashboards
navigation sidebar.your_work
navigation sidebar.home
navigation sidebar.drafts
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t4/keys.txt | tail -1 && node $P3TMP/i18n-del-list.mjs $P3TMP/t4/keys2.txt | tail -1 && node $P3TMP/keycount.mjs`
Expected: `total: 3 keys in each locale`、`total: 4 keys in each locale`、`en: 1795 keys`、`zh-CN: 1795 keys`

- [ ] **Step 6: 守卫**

规则（`$P3TMP/t4/rules.json`）。`ai-assistant` 区分大小写：`llm` 不区分大小写会命中 `scrollMode`、`allMembers`；
设计中列出的 `PlaneAi`、`aiHandler`、`AI_` 常量、`rephrase` 在本 Phase 开始时已没有命中，不设规则；
`posthog`、`intercom` 的命中样本取自 Plane 接入这两个 SDK 的写法。

```json
[
  {
    "id": "ai-assistant",
    "phase": "M1/P3",
    "why": "AI 助手：工作项表单描述编辑器里的 GPT 弹窗和\"I'm feeling lucky\"、AI 服务、实例开关 has_llm_configured、AI 图标（M1 设计 2.2、7.4）；以及\"Plane AI\"（pi-chat）：没有使用的侧边栏用户菜单里的入口、它的图标、路径判断和文案。编辑器包里的 AI 部分已在 P2 随文档页删除。区分大小写：llm 不区分大小写时会命中 scrollMode、allMembers；设计中列出的 PlaneAi、aiHandler、AI_ 常量、rephrase 在本 Phase 开始时已没有命中，不设规则",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "[gG]pt|GPT|llm|feeling[ _]lucky|AIService|ai\\.service|AiIcon|ai-icon|AiStar1Outline|pi[-_]chat|PiChat|plane-intelligence|isAiPath",
      "flags": ""
    },
    "samples": {
      "hit": [
        "import { GptAssistantPopover } from \"@/components/core/modals/gpt-assistant-popover\";",
        "// TODO: have to implement GPT Assistance",
        "            {issueName && issueName.trim() !== \"\" && config?.has_llm_configured && (",
        "  \"feeling_lucky\",",
        "                    <AiStar1Outline className=\"h-3.5 w-3.5\" />I{\"'\"}m feeling lucky",
        "import { AIService } from \"@/services/ai.service\";",
        "export function AiIcon({ width = \"16\", height = \"16\", className, color = \"currentColor\" }: ISvgIcons) {",
        "export * from \"./ai-icon\";",
        "import { PiChatLogo } from \"@plane/propel/icons\";",
        "      key: \"pi-chat\",",
        "  \"pi_chat\": \"Plane AI\",",
        "      \"plane-intelligence\": {",
        "  const isAiPath = pathname.includes(`/${workspaceSlug}/pi-chat`);"
      ],
      "miss": [
        "      scrollMode: \"if-needed\",",
        "    const allMembers = this.getProjectMemberships(projectId);"
      ],
      "files": {
        "hit": [
          "web/apps/web/core/components/issues/issue-modal/components/description-editor.tsx",
          "web/packages/types/src/instance/base.ts"
        ],
        "miss": [
          "web/apps/web/app/assets/logo.svg",
          "docs/v0/M1-frontend-trim/M1-design.md"
        ]
      }
    }
  },
  {
    "id": "unsplash-telemetry",
    "phase": "M1/P3",
    "why": "Unsplash 封面（封面选择只留静态图和上传；原来每次打开选择器都会请求 /api/unsplash/）、实例开关 has_unsplash_configured；遥测：应用里已没有遥测 SDK，这条规则防止 PostHog、Sentry、Intercom 和实例的 is_telemetry_enabled 回来（M1 设计 2.2、7.4）。posthog、intercom 两条命中样本取自 Plane 接入这两个 SDK 的写法",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "unsplash|posthog|telemetry|sentry|intercom",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "    () => fileService.getUnsplashImages(searchParams),",
        "import posthog from \"posthog-js\";",
        "  is_telemetry_enabled: boolean;",
        "    \"VITE_SENTRY_DSN\",",
        "import { IntercomProvider } from \"react-use-intercom\";"
      ],
      "miss": [
        "              <Tabs variant=\"contained\" defaultValue=\"images\">",
        "export const DEFAULT_COVER_IMAGE_URL = STATIC_COVER_IMAGES.IMAGE_1;"
      ],
      "files": {
        "hit": [
          "web/apps/web/core/components/core/image-picker-popover.tsx",
          "turbo.json"
        ],
        "miss": [
          "web/apps/web/app/assets/logo.svg",
          "docs/v0/M1-frontend-trim/M1-design.md"
        ]
      }
    }
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t4/rules.json && pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3`
Expected: `keywords: 38 rules, 24 exceptions, no hits.`、`alternatives or variants without a hit sample: 0`

- [ ] **Step 7: lint 上限**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected（原型，清理前）：`useMemo`、`useState` 两条新的 `no-unused-vars`（`import React, { … } from "react"` 里的具名导入，
"原型的教训"第 5 条）→ 删掉。清理后只剩固定节奏第 6 步列出的 `no-autofocus`、`day-tile.tsx` 两条已知条目。

Run: `node $P3TMP/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 609" && bash $P3TMP/caps.sh | tail -1`
Expected: `caps total: 759`（web 617 → 609）

- [ ] **Step 8: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 38 rules, 24 exceptions, no hits.` + 52

Run: `make test-web 2>&1 | grep Tasks:` → 15；`make build-web 2>&1 | grep Tasks:` → 11

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat 3d08bf5 | tail -1` → 没有输出，或每处差异都能说明

Run: `git diff --cached --shortstat` → `34 files changed, 148 insertions(+), 1022 deletions(-)`

knip 此时（`3d08bf5`）：文件 69、导出 105、类型 51。

提交信息：

```
feat(web): delete the AI assistant and Unsplash covers, and guard against telemetry

The work-item form loses its AI buttons, the GPT popover and "I'm
feeling lucky"; the AI service, types and icon and the instance's
has_llm_configured go. "Plane AI" (pi-chat) had one entry left, in the
unused sidebar user menu; that dead chain goes with its logo, path check
and copy.

The cover picker keeps its images and upload tabs: the Unsplash tab, the
/api/unsplash/ request it sent on every open, the control prop that only
fed its search, the helper branch and the instance's
has_unsplash_configured go. Nothing is left of telemetry in the code; a
guard rule keeps it out.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

### Task 5: 计费和升级提示；守卫的 `phase` 改为 `M1/P3`

M1 设计 9 节 P3 第 5 步：计费和升级提示最后删。它引用的、P2 已删的类型已由 P2 处理。spec 2.7。原型提交 `301369a`。

**Files**（原型 `301369a`：60 个文件，删 31、改 29）

- 删除（31）：
  - `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/billing/{header.tsx,page.tsx}`
  - `web/apps/web/app/assets/scribble/{scribble-black.svg,scribble-white.svg}`
  - `web/apps/web/core/components/icons/locked-component.tsx`
  - `web/apps/web/core/components/license/index.ts`
  - `web/apps/web/core/components/license/modal/card/{base-paid-plan-card.tsx,checkout-button.tsx,discount-info.tsx,free-plan.tsx,index.ts,plan-upgrade.tsx,talk-to-sales.tsx}`
  - `web/apps/web/core/components/license/modal/{index.ts,upgrade-modal.tsx}`
  - `web/apps/web/core/components/workspace/billing/comparison/{base.tsx,feature-detail.tsx,frequency-toggle.tsx,index.ts,plan-detail.tsx,plans.tsx,root.tsx}`
  - `web/apps/web/core/components/workspace/billing/{index.ts,root.tsx}`
  - `web/apps/web/core/components/workspace/{edition-badge.tsx,upgrade-badge.tsx}`
  - `web/packages/constants/src/{payment.ts,subscription.ts}`
  - `web/packages/propel/src/icons/actions/upgrade-icon.tsx`
  - `web/packages/types/src/payment.ts`
  - `web/packages/utils/src/subscription.ts`
- 修改（29）：
  - `tools/keywords.json`
  - `web/apps/web/app/routes/core.ts`
  - `web/apps/web/core/components/project/settings/features-list.tsx`
  - `web/apps/web/core/components/settings/workspace/sidebar/item-icon.tsx`
  - `web/apps/web/core/components/sidebar/sidebar-wrapper.tsx`
  - `web/apps/web/core/components/workspace/sidebar/help-section/root.tsx`
  - `web/apps/web/core/store/user/profile.store.ts`
  - `web/apps/web/package.json`
  - `web/packages/constants/src/{endpoints.ts,index.ts,navigation.test.ts,workspace.ts}`
  - `web/packages/constants/src/settings/workspace.ts`
  - `web/packages/i18n/src/locales/en/{accessibility.json,common.json,home.json,navigation.json,workspace-settings.json}`
  - `web/packages/i18n/src/locales/zh-CN/{accessibility.json,common.json,home.json,navigation.json,workspace-settings.json}`
  - `web/packages/propel/src/icons/actions/index.ts`
  - `web/packages/propel/src/icons/registry.ts`
  - `web/packages/types/src/{index.ts,settings.ts,users.ts}`
  - `web/packages/utils/src/index.ts`

**Interfaces**

- Consumes：Task 1、3 登记在计费文案上的 13 条例外，P2 留下的 8 条 `until: M1/P3` 例外。
- Produces：`WORKSPACE_SETTINGS` 只剩 `general`、`members`、`webhooks`；`TWorkspaceSettingsTabs` 同步；用户资料没有
  `billing_address*`；`tools/keywords.json` 顶层 `phase` = `M1/P3`，例外只剩 3 条；守卫规则 `billing`。

**Steps**

- [ ] **Step 1: 删除**

```
git rm -q -r "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/billing" \
  web/apps/web/core/components/license web/apps/web/core/components/workspace/billing
git rm -q web/apps/web/core/components/workspace/edition-badge.tsx web/apps/web/core/components/workspace/upgrade-badge.tsx \
  web/packages/constants/src/payment.ts web/packages/constants/src/subscription.ts \
  web/packages/types/src/payment.ts web/packages/utils/src/subscription.ts \
  web/packages/propel/src/icons/actions/upgrade-icon.tsx
```

**同一步**删掉 `app/routes/core.ts` 里 `:workspaceSlug/settings/billing` 的 `route(...)`。

- [ ] **Step 2: 按类型检查删到底**

原型脚本 `$P3TMP/t5/billing.py`：

- 工作区设置：`constants/src/settings/workspace.ts` 的 `billing-and-plans` 记录和 `ADMINISTRATION` 分组里的一项；
  `types/src/settings.ts` 的 `TWorkspaceSettingsTabs`；`settings/workspace/sidebar/item-icon.tsx` 的 `BillingsOutline`。
- 侧边栏：`sidebar/sidebar-wrapper.tsx` 底部那一栏只放了版本徽标（外加一段注释掉的帮助菜单），整栏删除；
  `workspace/sidebar/help-section/root.tsx` 的"联系销售"（`mailto:sales@plane.so`）。
- 项目功能：`project/settings/features-list.tsx` 的 `isPro`（全部为 `false`）、`UpgradeBadge` 和它外面的 `Tooltip`；
  标题直接是 `t(featureItem.key)`。
- 用户：`types/src/users.ts` 的 `billing_address_country`、`billing_address`、`has_billing_address`，
  `store/user/profile.store.ts` 的默认值同步。
- 常量：`constants/src/index.ts`、`types/src/index.ts`、`utils/src/index.ts` 的导出；`constants/src/endpoints.ts` 基点就已死的
  `MARKETING_CONTACT_US_PAGE_LINK`；`RESTRICTED_URLS` 去掉 `plane-pro`、`plane-ultimate`、`enterprise`、`plane-enterprise`、
  `upgrade`、`billing`（`one`、`business` 等其余产品词留给 M3）。
- propel：`icons/actions/index.ts`、`icons/registry.ts` 去掉升级图标。

Run: `pnpm exec turbo run check:types`
Expected: `Tasks:    23 successful, 23 total`（原型第一次就通过）

- [ ] **Step 3: 孤儿核对**

Run: `node $P3TMP/deadvocab.mjs 3d08bf5 $P3TMP/knip-t0.flat $P3TMP/t5/rules.json`
Expected:

```
web/apps/web/core/components/icons/locked-component.tsx	billing(LockedComponent)
```

→ `git rm -q web/apps/web/core/components/icons/locked-component.tsx`

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t5.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t5.txt 3d08bf5`
Expected（删之前）：`scribble/scribble-black.svg`、`scribble/scribble-white.svg`（只有升级弹窗用）→ `git rm` 这两张。

Run: `node $P3TMP/symref.mjs orphaned 3d08bf5 | head -1` → `exports used elsewhere at 3d08bf5 and not now: 0`

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t5.flat; diff $P3TMP/knip-t4.flat $P3TMP/knip-t5.flat | grep '^>'` → 没有输出

- [ ] **Step 4: 文案**

Run: `node $P3TMP/keyref.mjs orphaned 3d08bf5 | head -1` → `keys referenced at 3d08bf5 and not now: 7`

列表（`$P3TMP/t5/keys.txt`；第二段是以套餐、升级、席位、试用命名、基点就无引用的键）：

```
# referenced at the task's base and not after the deletions (keyref.mjs orphaned)
accessibility aria_labels.projects_sidebar.edition_badge
common contact_sales
common common.upgrade_cta
navigation sidebar.pro
workspace-settings workspace_settings.settings.billing_and_plans
# named after plans, upgrades, seats and trials; unreferenced already at the base
common upgrade_request
common common.upgrade
common common.add_seats
common common.business
home home.business_trial_banner
navigation sidebar.upgrade
navigation sidebar.upgrade_plan
navigation sidebar.plane_pro
navigation sidebar.business
workspace-settings workspace_settings.settings.general.delete_modal.description
workspace-settings workspace_settings.settings.general.delete_modal.dismiss
workspace-settings workspace_settings.settings.general.delete_modal.cancel
workspace-settings workspace_settings.settings.cancel_trial
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t5/keys.txt | tail -1 && node $P3TMP/keycount.mjs`
Expected: `total: 18 keys in each locale`、`en: 1759 keys`、`zh-CN: 1759 keys`

注销账户的说明不再说计费（`t5/billed.py`，中英文同改）：`web/packages/i18n/src/locales/en/common.json` 的
`deactivate_your_account_description` 改为
`"Once deactivated, you can't be assigned work items. To reactivate your account, you will need an invite to a workspace at this email address."`；
`zh-CN/common.json` 同一个键改为
`"一旦停用，您将无法被分配工作项。要重新激活您的账户，您需要收到发送到此电子邮件地址的工作区邀请。"`
（去掉"也不会被计入工作区的账单"一句）。

- [ ] **Step 5: 守卫**

规则（`$P3TMP/t5/rules.json`；`bill(?:ing|ed)` 的 `billed` 分支的命中样本就是改之前的英文说明）：

```json
[
  {
    "id": "billing",
    "phase": "M1/P3",
    "why": "计费和升级提示（M1 设计 2.2、7.4）：计费设置页、升级弹窗（license/）、套餐对比、版本和升级徽标、付费套餐的常量类型和工具、\"联系销售\"、试用和席位的文案、付费套餐的保留地址、营销链接（marketing_ 只搜 contact、plane 两种：保留的职业选项 marketing_or_growth 不算）。不搜单独的 subscription：工作项的订阅（关注）保留；设计中列出的 ProIcon 在本 Phase 开始时已没有命中，不设规则",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "upgrade|bill(?:ing|ed)|EProductSubscription|contact_sales|plane-(?:pro|ultimate|enterprise)|LockedComponent|edition[-_ ]?badge|payment|\\btrial|add_seats|talk_to_sales|marketing_(?:contact|plane)",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "        title={t(\"bulk_operations.upgrade_banner.message\")}",
        "export const BillingWorkspaceSettingsHeader = observer(function BillingWorkspaceSettingsHeader() {",
        "import type { EProductSubscriptionEnum, TBillingFrequency, TSubscriptionPrice } from \"@plane/types\";",
        "          <span className=\"text-11\">{t(\"contact_sales\")}</span>",
        "  \"plane-pro\",",
        "  \"plane-ultimate\",",
        "  \"plane-enterprise\",",
        "export function LockedComponent(props: { toolTipContent?: string }) {",
        "import { WorkspaceEditionBadge } from \"@/components/workspace/edition-badge\";",
        "import type { EProductSubscriptionEnum, IPaymentProduct, TSubscriptionPrice } from \"@plane/types\";",
        "      \"title\": \"Your 14-day Business plan trial is live!\",",
        "    \"add_seats\": \"Add Seats\",",
        "import { TALK_TO_SALES_URL } from \"@plane/constants\";",
        "import { MARKETING_PLANE_ONE_PAGE_LINK } from \"@plane/constants\";",
        "export const MARKETING_CONTACT_US_PAGE_LINK = \"https://plane.so/contact\";",
        "  \"deactivate_your_account_description\": \"Once deactivated, you can't be assigned work items and be billed for your workspace. To reactivate your account, you will need an invite to a workspace at this email address.\","
      ],
      "miss": [
        "export class IssueSubscriptionStore implements IIssueSubscriptionStore {",
        "    \"planned\": \"Planned\","
      ],
      "files": {
        "hit": [
          "web/apps/web/core/components/sidebar/sidebar-wrapper.tsx",
          "web/packages/i18n/src/locales/en/common.json"
        ],
        "miss": [
          "web/apps/web/app/assets/logo.svg",
          "docs/v0/M1-frontend-trim/M1-design.md"
        ]
      }
    }
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t5/rules.json && pnpm exec oxfmt tools && node tools/keywords.mjs`
Expected: 21 行 `stale exception: … web/apps/web/core/components/workspace/billing/comparison/plans.tsx …` 或
`… web/packages/constants/src/subscription.ts …`（计费文案已删，P2 的 8 条和 Task 1、3 的 13 条都过期），退出码非零。

Run:

```
node $P3TMP/kw.mjs del-path web/apps/web/core/components/workspace/billing/comparison/plans.tsx
node $P3TMP/kw.mjs del-path web/packages/constants/src/subscription.ts
node $P3TMP/kw.mjs set-phase M1/P3
pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3
```

Expected: `15 exceptions deleted`、`6 exceptions deleted`、`keywords: 39 rules, 3 exceptions, no hits.`、
`alternatives or variants without a hit sample: 0`。剩下的 3 条是 P2 的 `tlds.ts`（`analytics`、`wiki`，M9）和
`cycle.service.ts`（`analytics`，M6）。

- [ ] **Step 6: 测试**

在 `navigation.test.ts` 的 import 加 `GROUPED_WORKSPACE_SETTINGS`、`WORKSPACE_SETTINGS`（`./settings/workspace`），末尾加
（注释里不写被守卫的词）：

```ts
// The workspace settings keep three tabs once the paid-plan pages are gone (M1 design 2.2); the members and webhooks
// pages have no other entry.
describe("the workspace settings", () => {
  it("keep exactly these tabs", () => {
    expect(Object.keys(WORKSPACE_SETTINGS)).toEqual(["general", "members", "webhooks"]);
  });

  it("show every tab in exactly one sidebar group", () => {
    const grouped = Object.values(GROUPED_WORKSPACE_SETTINGS).flatMap((items) => items.map((item) => item.key));
    expect(grouped.toSorted()).toEqual(Object.keys(WORKSPACE_SETTINGS).toSorted());
  });
});
```

Run: `pnpm --filter @plane/constants test 2>&1 | grep -E "Tests "` → `Tests  9 passed (9)`

- [ ] **Step 7: lint 上限**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected: 只有固定节奏第 6 步列出的 `no-autofocus`、`day-tile.tsx` 两条已知条目。

Run: `node $P3TMP/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 604" && bash $P3TMP/caps.sh | tail -1`
Expected: `caps total: 754`（web 609 → 604）

- [ ] **Step 8: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 39 rules, 3 exceptions, no hits.` + 52

Run: `make test-web 2>&1 | grep Tasks:` → 15；`make build-web 2>&1 | grep Tasks:` → 11

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat 301369a | tail -1` → 没有输出，或每处差异都能说明

Run: `git diff --cached --shortstat` → `60 files changed, 73 insertions(+), 3414 deletions(-)`

knip 此时（`301369a`）：文件 67、导出 104、类型 51。

提交信息：

```
feat(web): delete billing and the upgrade prompts, and move the guard to M1/P3

The billing settings page, the upgrade modal and plan comparison, the
edition and upgrade badges, the "contact sales" menu item, the Pro
markers in the project features, the payment and subscription
constants, types and utils, and the user's billing address fields go.
The reserved workspace addresses lose the paid-plan names, and account
deactivation no longer speaks of being billed.

With the plan comparison gone, the 21 guard exceptions on its copy are
stale and go, so the guard's phase becomes M1/P3. A constants test pins
the three workspace settings tabs.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

### Task 6: 企业版扩展点；项目"邀请"改为添加成员

M1 设计 9 节 P3 第 6 步的"接线"部分（2.2 表里的 `extended` 空壳），加上 2.2 表里的项目邀请。spec 2.8、第 3 节第 1、5、6、11、12 条。
原型提交 `c08cc32`。

原则：CE 中为空、恒定或只做转发的钩子、prop、类型和注册表删掉，**它原来的作用直接写在用到的地方**；
只为企业版子类存在的 `Base*` 类加别名改回本名。每删一个扩展点，先确认它在 CE 中确实是空的 / 恒定的 / 只转发的
（看 `git show 301369a:<文件>`），再删。

**Files**（原型 `c08cc32`：122 个文件，删 27、增 1、改 89、重命名 5）

- 删除（27）：
  - `web/apps/web/app/routes/{extended.ts,helper.ts}`
  - `web/apps/web/app/routes/redirects/core/index.ts`
  - `web/apps/web/app/routes/redirects/extended/index.ts`
  - `web/apps/web/app/routes/redirects/index.ts`
  - `web/apps/web/core/components/workspace-notifications/notification-card/content.ts`
  - `web/apps/web/core/hooks/editor/use-extended-editor-config.ts`
  - `web/apps/web/core/hooks/{use-additional-editor-mention.tsx,use-additional-favorite-item-details.tsx,use-editor-flagging.ts,use-workspace-issue-properties-extended.tsx}`
  - `web/packages/constants/src/rich-filters/operator-labels/{core.ts,extended.ts}`
  - `web/packages/editor/src/extensions/{additional-slash-command-options.tsx,rich-text-extensions.tsx}`
  - `web/packages/editor/src/extensions/core/{extensions.ts,index.ts}`
  - `web/packages/editor/src/types/{editor-extended.ts,utils.ts}`
  - `web/packages/types/src/rich-filters/derived/{core.ts,extended.ts}`
  - `web/packages/types/src/rich-filters/field-types/{core.ts,extended.ts}`
  - `web/packages/types/src/rich-filters/operator-configs/{core.ts,extended.ts}`
  - `web/packages/types/src/rich-filters/operators/{core.ts,extended.ts}`
- 新增（1）：
  - `web/packages/editor/src/extensions/slash-commands/command-items-list.test.ts`
- 重命名（`git mv`）（5）：
  - `web/apps/web/core/components/project/send-project-invitation-modal.tsx` → `web/apps/web/core/components/project/add-project-members-modal.tsx`
  - `web/apps/web/core/store/base-command-palette.store.ts` → `web/apps/web/core/store/command-palette.store.ts`
  - `web/apps/web/core/store/member/project/base-project-member.store.ts` → `web/apps/web/core/store/member/project/project-member.store.ts`
  - `web/apps/web/core/store/base-power-k.store.ts` → `web/apps/web/core/store/power-k.store.ts`
  - `web/apps/web/core/store/user/base-permissions.store.ts` → `web/apps/web/core/store/user/permissions.store.ts`
- 修改（89）：
  - `tools/keywords.json`
  - `web/apps/web/app/(all)/[workspaceSlug]/layout.tsx`
  - `web/apps/web/app/routes.ts`
  - `web/apps/web/core/components/common/modal/global.tsx`
  - `web/apps/web/core/components/common/quick-actions-helper.tsx`
  - `web/apps/web/core/components/cycles/quick-actions.tsx`
  - `web/apps/web/core/components/editor/lite-text/editor.tsx`
  - `web/apps/web/core/components/editor/rich-text/editor.tsx`
  - `web/apps/web/core/components/modules/quick-actions.tsx`
  - `web/apps/web/core/components/project/{confirm-project-member-remove.tsx,member-list-item.tsx,member-list.tsx}`
  - `web/apps/web/core/components/rich-filters/filter-value-input/root.tsx`
  - `web/apps/web/core/components/views/quick-actions.tsx`
  - `web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx`
  - `web/apps/web/core/components/workspace/sidebar/{dropdown-item.tsx,project-navigation.tsx,sidebar-item.tsx}`
  - `web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx`
  - `web/apps/web/core/components/workspace/sidebar/favorites/new-fav-folder.tsx`
  - `web/apps/web/core/components/workspace/views/quick-action.tsx`
  - `web/apps/web/core/hooks/editor/{use-editor-config.ts,use-editor-mention.tsx}`
  - `web/apps/web/core/hooks/rich-filters/use-filters-operator-configs.ts`
  - `web/apps/web/core/hooks/store/{use-command-palette.ts,use-power-k.ts}`
  - `web/apps/web/core/hooks/store/user/user-permissions.ts`
  - `web/apps/web/core/hooks/{use-favorite-item-details.tsx,use-workspace-issue-properties.ts}`
  - `web/apps/web/core/store/{cycle.store.ts,cycle_filter.store.ts,favorite.store.ts,global-view.store.ts,label.store.ts,module.store.ts,module_filter.store.ts,project-view.store.ts,root.store.ts}`
  - `web/apps/web/core/store/inbox/{inbox-issue.store.ts,project-inbox.store.ts}`
  - `web/apps/web/core/store/issue/issue-details/activity.store.ts`
  - `web/apps/web/core/store/member/index.ts`
  - `web/apps/web/core/store/notifications/{notification.ts,workspace-notifications.store.ts}`
  - `web/apps/web/core/store/project/{index.ts,project.store.ts,project_filter.store.ts}`
  - `web/apps/web/core/store/user/{index.ts,profile.store.ts}`
  - `web/apps/web/core/store/workspace/{api-token.store.ts,index.ts,webhook.store.ts}`
  - `web/apps/web/package.json`
  - `web/packages/constants/src/project.ts`
  - `web/packages/constants/src/rich-filters/operator-labels/index.ts`
  - `web/packages/editor/package.json`
  - `web/packages/editor/src/components/editors/editor-wrapper.tsx`
  - `web/packages/editor/src/components/editors/rich-text/editor.tsx`
  - `web/packages/editor/src/components/menus/block-menu.tsx`
  - `web/packages/editor/src/components/menus/bubble-menu/root.tsx`
  - `web/packages/editor/src/editor-interaction.test.ts`
  - `web/packages/editor/src/extensions/{extensions.test.ts,extensions.ts,index.ts,utility.ts}`
  - `web/packages/editor/src/extensions/slash-commands/{command-items-list.tsx,root.tsx}`
  - `web/packages/editor/src/hooks/use-editor.ts`
  - `web/packages/editor/src/plugins/drop.ts`
  - `web/packages/editor/src/types/{asset.ts,config.ts,editor.ts,hook.ts,index.ts}`
  - `web/packages/i18n/src/locales/en/{common.json,project-settings.json,workspace-settings.json,workspace.json}`
  - `web/packages/i18n/src/locales/zh-CN/{common.json,project-settings.json,workspace-settings.json,workspace.json}`
  - `web/packages/shared-state/src/store/rich-filters/config.ts`
  - `web/packages/types/src/favorite/favorite.ts`
  - `web/packages/types/src/rich-filters/derived/index.ts`
  - `web/packages/types/src/rich-filters/field-types/index.ts`
  - `web/packages/types/src/rich-filters/operator-configs/index.ts`
  - `web/packages/types/src/rich-filters/operators/index.ts`
  - `web/packages/utils/src/editor/markdown-parser/types.ts`

**Interfaces**

- Consumes：Task 5 的结果。
- Produces：
  - `routes.ts`：`const routes: RouteConfigEntry[] = [layout("./layout.tsx", [...coreRoutes, route("*", "./not-found.tsx")])];`
  - 编辑器包：没有 `flaggedExtensions`、`extendedEditorProps`、`IEditorPropsExtended`、`TExtendedEditorCommands`、
    `TExtendedEditorRefApi`、`TExtendedFileHandler`、`TAdditionalEditorAsset`、`TSlashCommandAdditionalOption`；
    `EditorRefApi`（原 `CoreEditorRefApi`）、`TEditorAsset`（原 `TEditorImageAsset`）、`TCustomComponentsMetaData`
    （原 `TCoreCustomComponentsMetaData`，在 `@plane/utils` 的 markdown 类型里）；
  - 菜单 hook（`useCycleMenuItems`、`useModuleMenuItems`、`useViewMenuItems`）返回 `TContextMenuItem[]`；
  - store：`CommandPaletteStore`、`PowerKStore`、`ProjectMemberStore`、`UserPermissionStore`、`WorkspaceRootStore`、`RootStore`；
  - 富文本筛选：`OPERATORS`（原 `CORE_OPERATORS`）及各类型的最终名字；
  - `TFavoriteEntityType = "project" | "view" | "cycle" | "module" | "folder"`；
  - `AddProjectMembersModal`（`project/add-project-members-modal.tsx`）；
  - 守卫规则 `enterprise-shells`、`project-invitations`，例外 `joinProject` 的地址（`until: M3`）。

**Steps**

- [ ] **Step 1: 路由**

```
git rm -q web/apps/web/app/routes/extended.ts web/apps/web/app/routes/helper.ts \
  web/apps/web/app/routes/redirects/index.ts web/apps/web/app/routes/redirects/core/index.ts \
  web/apps/web/app/routes/redirects/extended/index.ts
```

`redirects/` 下的三个 `index.ts` 是 `core.ts` 已经自己列出的重定向条目的另一份拷贝，没人用（`deadvocab` 也会列出后两个）。
`app/routes.ts` 去掉 `extendedRoutes`、`mergeRoutes` 的 import 和 `mergedRoutes`，直接用 `coreRoutes`（`$P3TMP/t6/routes.py`）。

- [ ] **Step 2: 编辑器包**

```
git rm -q -r web/packages/editor/src/extensions/core
git rm -q web/packages/editor/src/types/editor-extended.ts web/packages/editor/src/types/utils.ts \
  web/packages/editor/src/extensions/additional-slash-command-options.tsx \
  web/packages/editor/src/extensions/rich-text-extensions.tsx
```

然后（`editor.py`、`editor2.py`、`editor3.py`）：

- `flaggedExtensions`（从来不读）、`extendedEditorProps`（类型是 `unknown`）从每个 prop 类型、`Pick` / `Omit` 联合、组件参数、
  hook 参数、`useMemo` 依赖和两个测试文件里删除（`editor-wrapper.tsx`、`rich-text/editor.tsx`、`menus/block-menu.tsx`、
  `menus/bubble-menu/root.tsx`、`extensions/extensions.ts`、`extensions/utility.ts`、`hooks/use-editor.ts`、`plugins/drop.ts`、
  `types/{config,editor,hook,index,asset}.ts`）。
- 富文本编辑器：原来 `RichTextEditorAdditionalExtensions` 注册表只注册斜杠命令，改为直接写
  `...(disabledExtensions.includes("slash-commands") ? [] : [SlashCommands({ disabledExtensions })])`。
- 斜杠菜单：原来图片项经"附加选项"插在代码块之后。`slash-commands/command-items-list.tsx` 在 `code` 项之后直接写
  `...(disabledExtensions.includes("image") ? [] : [{ commandKey: "image", … } satisfies ISlashCommandItem])`
  （各字段取自删掉的 `additional-slash-command-options.tsx`）；`slash-commands/root.tsx` 删 `TSlashCommandAdditionalOption`。
- 改名：基点上 `EditorRefApi = CoreEditorRefApi & TExtendedEditorRefApi`（后者是 `unknown`），现在 `CoreEditorRefApi`
  直接叫 `EditorRefApi`；`TEditorAsset = TEditorImageAsset | TAdditionalEditorAsset`（后者为空），现在 `TEditorImageAsset`
  直接叫 `TEditorAsset`；`TFileHandler` 不再与空的 `TExtendedFileHandler`（`object`）相交；
  `TCustomComponentsMetaData = TCoreCustomComponentsMetaData & TExtended…`，现在前者直接叫 `TCustomComponentsMetaData`
  （`web/packages/utils/src/editor/markdown-parser/types.ts`）。
- 测试：`extensions/extensions.test.ts` 删两个 prop；`editor-interaction.test.ts` 改用 `SlashCommands({ disabledExtensions: [] })`。
  新增 `web/packages/editor/src/extensions/slash-commands/command-items-list.test.ts`：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
// local imports
import type { TExtensions } from "@/types";
import { getSlashCommandFilteredSections } from "./command-items-list";

// The slash menu's general section offers an image right after the code block unless the editor disables images.
// The image used to be pushed in after "code" through a list of "additional options" (M1/P3 writes it in place).
const generalKeys = (disabledExtensions: TExtensions[]) =>
  getSlashCommandFilteredSections({ disabledExtensions })({ query: "" })
    .find((section) => section.key === "general")
    ?.items.map((item) => item.key);

describe("the slash command list", () => {
  it("offers an image right after the code block", () => {
    const keys = generalKeys([]) ?? [];
    expect(keys[keys.indexOf("code") + 1]).toBe("image");
  });

  it("offers no image when the editor disables images", () => {
    expect(generalKeys(["image"])).not.toContain("image");
  });
});
```

Run: `pnpm --filter @plane/editor test 2>&1 | grep -E "Tests "` → `Tests  16 passed (16)`

- [ ] **Step 3: web 的编辑器 hook、菜单、store**

```
git rm -q web/apps/web/core/hooks/use-editor-flagging.ts web/apps/web/core/hooks/editor/use-extended-editor-config.ts \
  web/apps/web/core/hooks/use-additional-editor-mention.tsx \
  web/apps/web/core/components/workspace-notifications/notification-card/content.ts \
  web/apps/web/core/hooks/use-additional-favorite-item-details.tsx \
  web/apps/web/core/hooks/use-workspace-issue-properties-extended.tsx
git mv web/apps/web/core/store/base-command-palette.store.ts web/apps/web/core/store/command-palette.store.ts
git mv web/apps/web/core/store/base-power-k.store.ts web/apps/web/core/store/power-k.store.ts
git mv web/apps/web/core/store/member/project/base-project-member.store.ts web/apps/web/core/store/member/project/project-member.store.ts
git mv web/apps/web/core/store/user/base-permissions.store.ts web/apps/web/core/store/user/permissions.store.ts
```

然后（`web_editor.py`、`notif.py`、`menus.py`、`stores.py`、`misc.py`）：

- 编辑器 hook：`editor/lite-text/editor.tsx`、`editor/rich-text/editor.tsx`、`hooks/editor/use-editor-config.ts` 不再取 flagging 和
  扩展配置；`hooks/editor/use-editor-mention.tsx` 直接写 `query_type: ["user_mention"]`、不再合并"附加的分组"
  （基点的 `useAdditionalEditorMention` 返回的就是这两个值）。
- 通知：`sidebar/notification-card/content.tsx` 把原来 `content.ts` 里的兜底映射并进来，映射和类型改为文件内部的
  （`NOTIFICATION_CONTENT_MAP` 不导出）。
- 菜单：`common/quick-actions-helper.tsx` 删 `MenuResult` 和 `useIntakeHeaderMenuItems`，三个菜单 hook 直接返回 items；
  `cycles/quick-actions.tsx`、`modules/quick-actions.tsx`、`views/quick-actions.tsx`、`workspace/views/quick-action.tsx`
  不再渲染恒为 `null` 的 `modals`。
- store：四个文件里的类去掉 `Base` 前缀和别名导出（`BaseCommandPaletteStore` → `CommandPaletteStore` 等），
  `store/workspace/index.ts` 的 `BaseWorkspaceRootStore` → `WorkspaceRootStore`，`store/root.store.ts` 的 `CoreRootStore` →
  `RootStore`（去掉 `export { CoreRootStore as RootStore }`）；所有导入这些文件和类型的地方同步（约 25 个 store 文件、
  `hooks/store/use-command-palette.ts`、`use-power-k.ts`、`user/user-permissions.ts`）。
- 其他"附加"的一半：`store/workspace/index.ts` 等处的 `mutateWorkspaceMembersActivity`；`hooks/use-workspace-issue-properties.ts`
  不再调用扩展版；`common/modal/global.tsx` 的 prop 和 `app/(all)/[workspaceSlug]/layout.tsx` 传给它的路由 props；
  `rich-filters/filter-value-input/root.tsx` 把 `AdditionalFilterValueInput`（只渲染"不支持"提示）写进来；
  `shared-state/src/store/rich-filters/config.ts` 的 `_getAdditionalOperatorOptions`；
  `workspace/sidebar/project-navigation.tsx` 的 `additionalNavigationItems`，改为
  `useMemo(() => baseNavigation(...).sort(...))`；`workspace/sidebar/sidebar-item.tsx` 的 `additionalRender`。

- [ ] **Step 4: 富文本筛选的 core / extended 合并**

```
git rm -q web/packages/types/src/rich-filters/{operators,derived,field-types,operator-configs}/{core,extended}.ts \
  web/packages/constants/src/rich-filters/operator-labels/{core,extended}.ts
```

四组类型各自的 `index.ts` 直接定义合并后的类型，用最终的名字（`CORE_OPERATORS` + 空的 `EXTENDED_OPERATORS` → `OPERATORS`，
`TCoreSupportedOperators` → `TSupportedOperators` 等）；`operators/index.ts`、`operator-configs/index.ts` 在原型里是整个重写的，
照 `git show c08cc32 -- web/packages/types/src/rich-filters/operators/index.ts` 等写；常量的 `operator-labels/index.ts` 同样合并。
`web/apps/web/core/hooks/rich-filters/use-filters-operator-configs.ts` 是 `CORE_OPERATORS` 唯一的 web 读取方，改用 `OPERATORS`
（`richfilters.py`）。"显示用"的否定运算符那一层（`TAllAvailable*ForDisplay`、`getDisplayOperator` 等）**不动**，交给 M4。

- [ ] **Step 5: 收藏的实体类型（P2 评审交给 P3 的产品判断）**

- `web/packages/types/src/favorite/favorite.ts`：`export type TFavoriteEntityType = "project" | "view" | "cycle" | "module" | "folder";`，
  `IFavorite.entity_type` 用它。
- `favorite-items/common/favorite-item-icon.tsx`：`ICON_MAP: Record<TFavoriteEntityType, …>`，去掉兜底的 `PagesOutline`。
- `hooks/use-favorite-item-details.tsx`：`switch` 覆盖全部取值，包括 `folder`（否则 `itemTitle` 是 `string | undefined`），
  删 `useAdditionalFavoriteItemDetails` 的调用；`favorites/new-fav-folder.tsx` 写 `entity_type: "folder"`；
  `store/favorite.store.ts` 的参数用 `TFavoriteEntityType`（`favorites.py`、`fav_import.py`、`fav2.py`）。

- [ ] **Step 6: 项目"邀请"改为添加成员**

```
git mv web/apps/web/core/components/project/send-project-invitation-modal.tsx web/apps/web/core/components/project/add-project-members-modal.tsx
```

（`invitation.py` … `invitation4.py`）

- 组件改名 `AddProjectMembersModal`；局部变量 `nonProjectMemberIds`、`isProjectMember`；`project/member-list.tsx` 的
  `addMembersModal` / `setAddMembersModal`；`project/member-list-item.tsx` 同步。
- 文案（中英文成对）：`project-settings.json` 的 `project_settings.members.invite_members` → `add_members`：
  `title` "Add members" / "添加成员"，`sub_heading` "Add workspace members to work on your project." / "添加工作区成员参与您的项目。"，
  `select_co_worker` 删除。工作区下拉菜单（`workspace/sidebar/dropdown-item.tsx`）原来借用项目的这个键，改用新的
  `workspace-settings.json` 的 `workspace_settings.settings.members.invite_members`："Invite members" / "邀请成员"
  （工作区邀请保留，M1 设计 3.15）。
- 离开项目（`project/confirm-project-member-remove.tsx`）："…if a project admin adds you again or if it's public."；
  私有项目的说明 `workspace.json` 的 `workspace_projects.network.private.description`："Accessible only to its members" /
  "仅项目成员可以访问"；`constants/src/project.ts` 那一行后面过时的注释删掉。

- [ ] **Step 7: 类型检查与孤儿核对**

Run: `pnpm exec turbo run check:types`
原型（`c08cc32`）：编辑器包先报 2 处、web 1 处，收藏的 `switch` 补上 `folder` 后 2 处，最后 23/23。
Expected（最后一次）：`Tasks:    23 successful, 23 total`

Run: `node $P3TMP/deadvocab.mjs 301369a $P3TMP/knip-t0.flat $P3TMP/t6/rules.json`
Expected:

```
web/apps/web/app/routes/redirects/extended/index.ts	enterprise-shells(extendedRedirectRoutes)
web/apps/web/app/routes/redirects/index.ts	enterprise-shells(extendedRedirectRoutes)
```

（Step 1 已删；这一步在 Step 1 之前跑也行。）

Run: `node $P3TMP/symref.mjs orphaned 301369a | head -1` → `exports used elsewhere at 301369a and not now: 0`

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t6.flat; diff $P3TMP/knip-t5.flat $P3TMP/knip-t6.flat | grep '^>'` → 没有输出

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t6.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t6.txt 301369a` → 没有输出

- [ ] **Step 8: 文案**

Run: `node $P3TMP/keyref.mjs orphaned 301369a | head -1`
Expected: 改键之前 `keys referenced at 301369a and not now: 2`（`project_settings.members.invite_members.{title,sub_heading}`，
由 Step 6 改名为 `add_members`）；改完之后 `0`。

以邀请命名、没有读取方的键（`$P3TMP/t6/keys.txt`）：

```
# feature-named (project invitations) with no reader at the task base
common accessible_only_by_invite
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t6/keys.txt | tail -1 && node $P3TMP/keycount.mjs`
Expected: `total: 1 keys in each locale`、`en: 1758 keys`、`zh-CN: 1758 keys`

- [ ] **Step 9: 守卫**

规则（`$P3TMP/t6/rules.json`）。`enterprise-shells` 按确切符号列出 39 个分支，不用 `TExtended`、`TAdditional` 前缀
（会命中保留的扩展侧边栏和筛选组件）；`CE`、`EE` 只作为整词搜。

```json
[
  {
    "id": "enterprise-shells",
    "phase": "M1/P3",
    "why": "企业版的扩展点（M1 设计 2.2、7.4）：CE 中为空、恒定或只做转发的钩子、类型和注册表，以及为企业版子类准备的 Base 类加别名。只列确切的符号：TExtended、TAdditional 这类前缀会命中保留的扩展侧边栏和筛选组件；CE、EE 只作为整词搜（注释里的\"CE 或 EE\"说明）。设计中列出的 ExtendedBasePage 在本 Phase 开始时已没有命中，不设规则",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "\\b(?:CE|EE)\\b|auth-ee|extendedRoutes|mergeRoutes|extendedRedirectRoutes|useEditorFlagging|flaggedExtensions|extendedEditorProps|IEditorPropsExtended|editor-extended|TExtendedEditor(?:Commands|RefApi)|TExtendedFileHandler|TExtendedCommandExtraProps|CoreEditorAdditionalExtensions|RichTextEditorAdditionalExtensions|AdditionalSlashCommand|TSlashCommandAdditionalOption|TAdditionalEditorAsset|TAdditionalActiveDropbarExtensions|TExtendedCustomComponentsMetaData|useExtendedEditorConfig|useAdditional(?:EditorMention|FavoriteItemDetails)|use-additional-|useWorkspaceIssuePropertiesExtended|ADDITIONAL_NOTIFICATION_CONTENT_MAP|renderAdditional(?:Action|Value)|AdditionalFilterValueInput|_getAdditionalOperatorOptions|additionalNavigationItems|additionalRender|additionalModals|mutateWorkspaceMembersActivity|useIntakeHeaderMenuItems|EXTENDED_(?:LOGICAL|EQUALITY|COLLECTION|COMPARISON|MULTI_VALUE|FILTER_FIELD|OPERATOR|DATE_OPERATOR)|TExtended(?:Supported|AllAvailable|FilterFieldConfigs|ExactOperator|InOperator|RangeOperator|OperatorSpecific)|NEGATED_(?:DATE_)?OPERATOR_LABELS_MAP|Base(?:CommandPalette|PowerK|ProjectMember|UserPermission)Store|BaseWorkspaceRootStore|CoreRootStore",
      "flags": ""
    },
    "samples": {
      "hit": [
        "  // Use unified menu hook from plane-web (resolves to CE or EE)",
        "import type { TExtendedLoginMediums } from \"./auth-ee\";",
        "import { extendedRoutes } from \"./routes/extended\";",
        "import { mergeRoutes } from \"./routes/helper\";",
        "export const extendedRedirectRoutes: RouteConfigEntry[] = [];",
        "import { useEditorFlagging } from \"@/hooks/use-editor-flagging\";",
        "  \"disabledExtensions\" | \"flaggedExtensions\" | \"getEditorMetaData\"",
        "  Omit<ILiteTextEditorProps, \"fileHandler\" | \"mentionHandler\" | \"extendedEditorProps\">,",
        "import type { IEditorPropsExtended, TEditorCommands, TExtensions } from \"@/types\";",
        "import type { IEditorPropsExtended, TExtendedEditorCommands } from \"@/types/editor-extended\";",
        "export type TExtendedEditorRefApi = unknown;",
        "import type { TExtendedFileHandler } from \"@plane/editor\";",
        "export type TExtendedCommandExtraProps = unknown;",
        "export const CoreEditorAdditionalExtensions = (props: TCoreAdditionalExtensionsProps): Extensions => {",
        "import { RichTextEditorAdditionalExtensions } from \"@/extensions/rich-text-extensions\";",
        "export const coreEditorAdditionalSlashCommandOptions = (props: Props): TSlashCommandAdditionalOption[] => {",
        "export type TAdditionalEditorAsset = never;",
        "import type { TAdditionalActiveDropbarExtensions } from \"@/types/utils\";",
        "export type TExtendedCustomComponentsMetaData = unknown;",
        "import { useExtendedEditorConfig } from \"@/hooks/editor/use-extended-editor-config\";",
        "import { useAdditionalEditorMention } from \"@/hooks/use-additional-editor-mention\";",
        "import { useAdditionalFavoriteItemDetails } from \"@/hooks/use-additional-favorite-item-details\";",
        "export const useWorkspaceIssuePropertiesExtended = (workspaceSlug: string | string[] | undefined) => {};",
        "export const ADDITIONAL_NOTIFICATION_CONTENT_MAP: TNotificationContentMap = {};",
        "export const renderAdditionalAction = (notificationField: string, verb: string | undefined) => {",
        "    return renderAdditionalValue(notificationField, newValue, oldValue);",
        "  return <AdditionalFilterValueInput {...props} />;",
        "      const additionalOperatorOption = this._getAdditionalOperatorOptions(operator, value);",
        "  additionalNavigationItems?: (workspaceSlug: string, projectId: string) => TNavigationItem[];",
        "  additionalRender?: (itemKey: string, workspaceSlug: string) => ReactNode;",
        "  const additionalModals = Array.isArray(menuResult) ? null : menuResult.modals;",
        "  mutateWorkspaceMembersActivity: (workspaceSlug: string) => Promise<void>;",
        "export const useIntakeHeaderMenuItems = (props: {",
        "  ...EXTENDED_LOGICAL_OPERATOR,",
        "  ...EXTENDED_EQUALITY_OPERATOR,",
        "  ...EXTENDED_COLLECTION_OPERATOR,",
        "  ...EXTENDED_COMPARISON_OPERATOR,",
        "  ...EXTENDED_MULTI_VALUE_OPERATORS,",
        "  ...EXTENDED_FILTER_FIELD_TYPE,",
        "  ...EXTENDED_OPERATOR_LABELS_MAP,",
        "  ...EXTENDED_DATE_OPERATOR_LABELS_MAP,",
        "export type TSupportedOperators = TCoreSupportedOperators | TExtendedSupportedOperators;",
        "  | TExtendedAllAvailableDateFilterOperatorsForDisplay<V>;",
        "  | TExtendedFilterFieldConfigs<V>;",
        "export type TExactOperatorConfigs = TCoreExactOperatorConfigs | TExtendedExactOperatorConfigs;",
        "export type TInOperatorConfigs = TCoreInOperatorConfigs | TExtendedInOperatorConfigs;",
        "export type TRangeOperatorConfigs = TCoreRangeOperatorConfigs | TExtendedRangeOperatorConfigs;",
        "} & TExtendedOperatorSpecificConfigs;",
        "export const NEGATED_OPERATOR_LABELS_MAP: Record<never, string> = {} as const;",
        "export const NEGATED_DATE_OPERATOR_LABELS_MAP: Record<never, string> = {} as const;",
        "export { BaseCommandPaletteStore as CommandPaletteStore };",
        "import type { IBasePowerKStore as IPowerKStore } from \"@/store/base-power-k.store\";",
        "export { BaseProjectMemberStore as ProjectMemberStore };",
        "export { BaseUserPermissionStore as UserPermissionStore };",
        "export { BaseWorkspaceRootStore as WorkspaceRootStore };",
        "export { CoreRootStore as RootStore };"
      ],
      "miss": [
        "import { ExtendedAppHeader } from \"@/components/common/extended-app-header\";",
        "type TAdditionalWorkItemFiltersProps = {",
        "  const hasAdditionalChanges = useMemo(",
        "export interface ExtendedEmojiStorage extends EmojiStorage {",
        "  const { isExtendedProjectSidebarOpened, toggleExtendedProjectSidebar } = useAppTheme();",
        "export abstract class BaseIssuesStore implements IBaseIssuesStore {",
        "import { ACCEPTED_ATTACHMENT_MIME_TYPES, ACCEPTED_IMAGE_MIME_TYPES } from \"@/constants/config\";"
      ],
      "files": {
        "hit": [
          "web/apps/web/app/routes.ts",
          "web/packages/editor/src/types/editor.ts"
        ],
        "miss": [
          "web/apps/web/app/assets/logo.svg",
          "docs/v0/M1-frontend-trim/M1-design.md"
        ]
      }
    }
  },
  {
    "id": "project-invitations",
    "phase": "M1/P3",
    "why": "项目邀请（M1 设计 2.2、3.15）：web 里没有按邮件邀请进项目的流程，\"邀请成员\"弹窗是从工作区成员中直接添加，改名为添加成员；离开项目、私有项目的文案不再说\"受邀\"。工作区邀请保留，不搜单独的 invite、invitation。精确例外：joinProject 调用的 Plane 地址 /projects/invitations/，到 M3",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "ProjectInvitation|project.*invit(?:e|ation)",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "import { SendProjectInvitationModal } from \"./send-project-invitation-modal\";",
        "            {t(\"project_settings.members.invite_members.title\")}",
        "                    project? You will be able to join the project if invited again or if it{\"'\"}s public.",
        "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })"
      ],
      "miss": [
        "    return this.get(\"/api/users/me/workspaces/invitations/\")",
        "        \"title\": \"Invite people to collaborate\","
      ],
      "files": {
        "hit": [
          "web/apps/web/core/components/project/member-list.tsx",
          "web/packages/i18n/src/locales/en/project-settings.json"
        ],
        "miss": [
          "web/apps/web/app/assets/logo.svg",
          "docs/v0/M1-frontend-trim/M1-design.md"
        ]
      }
    }
  }
]
```

例外（`$P3TMP/t6/exceptions.json`）：

```json
[
  {
    "rule": "project-invitations",
    "path": "web/apps/web/core/services/user.service.ts",
    "match": "projects/invitation",
    "count": 1,
    "reason": "joinProject 调用的 Plane 地址 /projects/invitations/（加入公开项目），M3 定义项目成员接口时替换",
    "until": "M3"
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t6/rules.json && node $P3TMP/kw.mjs add-exceptions $P3TMP/t6/exceptions.json && pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3`
Expected: `keywords: 41 rules, 4 exceptions, no hits.`、`alternatives or variants without a hit sample: 0`

- [ ] **Step 10: lint 上限**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected（原型，清理前）：三处 `TContextMenuItem`、一处 `ReactNode` 的 `no-unused-vars` → 删掉（`lint1.py`）。
清理后只剩固定节奏第 6 步列出的已知条目，其中 `add-project-members-modal.tsx` 的两条是本 Task 改名带来的。

Run:

```
node $P3TMP/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 600"
node $P3TMP/pkg.mjs web/packages/editor/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 65"
bash $P3TMP/caps.sh | tail -1
```

Expected: `caps total: 748`（web 604 → 600，editor 67 → 65）

- [ ] **Step 11: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 41 rules, 4 exceptions, no hits.` + 52

Run: `make test-web 2>&1 | grep Tasks:` → 15；`make build-web 2>&1 | grep Tasks:` → 11

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat c08cc32 | tail -1` → 没有输出，或每处差异都能说明

Run: `git diff --cached --shortstat` → `122 files changed, 547 insertions(+), 1447 deletions(-)`

knip 此时（`c08cc32`）：文件 64、导出 96、类型 44。

提交信息：

```
feat(web): delete the enterprise extension points, and project invitations become adding members

The Community Edition code carried the seams the enterprise build plugged into: hooks, props,
types and registries that are empty, constant or pass-through here. They go, and what they did
is written where it applies:
- routes: no extended route table or merge; the unused redirect index copies go with it;
- editor: flaggedExtensions (never read) and extendedEditorProps (typed unknown) leave every
  prop list; the rich-text registry becomes "slash commands unless disabled"; the image slash
  command sits after the code block unless images are disabled (a new test pins both);
- web editor hooks: no editor flagging, no extended file handlers, mentions search users only;
- the menu hooks return their items (no always-null modals, no CE/EE branch); the store
  classes named Base plus an alias get their own names;
- notifications, favourites, filter values, operator options, sidebar items, project
  navigation, workspace issue properties and global modals lose their "additional" halves;
  favourites name the entity types v0 has, without the page icon fallback;
- rich filters: the core and empty extended halves of the operator, field and label types
  become one module each.
The project "invite" modal adds workspace members directly; it is named and worded for that,
and the leave-project and private-project copy no longer speaks of invitations.

Guard: enterprise-shells and project-invitations rules; the joinProject address
/projects/invitations/ is an exception until M3.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

### Task 7: Plane 自身的死代码和 P2 评审留下的事项；knip 改为门禁

M1 设计 9 节 P3 第 6 步的死代码、图片和 knip 门禁（7.2），以及 P2 评审第 7 节交给 P3 的清单。spec 2.9、第 3 节第 7、9、10、13 条。
原型提交 `98025c6`。原型用 `$P3TMP/t7/replay.sh` 从 Task 6 按下面的顺序重放过一遍，得到的树与 `98025c6` 完全相同。

**Files**（原型 `98025c6`：228 个文件，删 113、增 1、改 114；增的是 `links/types.ts`，见 Step 6）

- 删除（113）：
  - `web/apps/web/app/assets/attachment/img-icon.png`
  - `web/apps/web/app/assets/empty-state/{api-token.svg,web-hook.svg}`
  - `web/apps/web/app/assets/empty-state/project-settings/{integrations-dark-resp.webp,integrations-dark.webp,integrations-light-resp.webp,integrations-light.webp}`
  - `web/apps/web/app/assets/empty-state/search/{project-dark.webp,project-light.webp}`
  - `web/apps/web/app/assets/empty-state/workspace-settings/{integrations-dark-resp.webp,integrations-dark.webp,integrations-light-resp.webp,integrations-light.webp}`
  - `web/apps/web/app/assets/services/jira.svg`
  - `web/apps/web/core/components/api-token/empty-state.tsx`
  - `web/apps/web/core/components/auth-screens/workspace/not-a-member.tsx`
  - `web/apps/web/core/components/base-layouts/{constants.ts,layout-switcher.tsx}`
  - `web/apps/web/core/components/base-layouts/hooks/{use-group-drop-target.ts,use-layout-state.ts}`
  - `web/apps/web/core/components/base-layouts/kanban/{group-header.tsx,group.tsx,item.tsx,layout.tsx}`
  - `web/apps/web/core/components/base-layouts/list/{group-header.tsx,group.tsx,item.tsx,layout.tsx}`
  - `web/apps/web/core/components/base-layouts/loaders/layout-loader.tsx`
  - `web/apps/web/core/components/common/new-empty-state.tsx`
  - `web/apps/web/core/components/core/sidebar/sidebar-menu-hamburger-toggle.tsx`
  - `web/apps/web/core/components/cycles/list/cycle-list-project-group-header.tsx`
  - `web/apps/web/core/components/empty-state/{comic-box-button.tsx,helper.tsx}`
  - `web/apps/web/core/components/icons/attachment/{document-icon.tsx,img-file-icon.tsx,tune-icon.tsx}`
  - `web/apps/web/core/components/inbox/inbox-filter/filters/state.tsx`
  - `web/apps/web/core/components/issues/issue-detail/label/create-label.tsx`
  - `web/apps/web/core/components/issues/issue-detail/links/{link-detail.tsx,links.tsx,root.tsx}`
  - `web/apps/web/core/components/issues/issue-detail/reactions/issue-comment.tsx`
  - `web/apps/web/core/components/issues/issue-detail/relation-select.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/filters/applied-filters/{cycle.tsx,date.tsx,index.ts,label.tsx,members.tsx,module.tsx,priority.tsx,project.tsx,state-group.tsx,state.tsx}`
  - `web/apps/web/core/components/issues/issue-layouts/filters/header/filters/{created-by.tsx,cycle.tsx,labels.tsx,mentions.tsx,module.tsx}`
  - `web/apps/web/core/components/issues/label.tsx`
  - `web/apps/web/core/components/modules/sidebar-select/{index.ts,select-status.tsx}`
  - `web/apps/web/core/components/onboarding/{create-or-join-workspaces.tsx,create-workspace.tsx,invitations.tsx,invite-members.tsx,step-indicator.tsx}`
  - `web/apps/web/core/components/power-k/actions/helper.ts`
  - `web/apps/web/core/components/profile/profile-setting-content-wrapper.tsx`
  - `web/apps/web/core/components/project-states/state-delete-modal.tsx`
  - `web/apps/web/core/components/project/{empty-state.tsx,multi-select-modal.tsx}`
  - `web/apps/web/core/components/readonly/{cycle.tsx,date.tsx,index.tsx,labels.tsx,member.tsx,module.tsx,priority.tsx,state.tsx}`
  - `web/apps/web/core/components/settings/layout.tsx`
  - `web/apps/web/core/components/sidebar/search-button.tsx`
  - `web/apps/web/core/components/ui/loader/notification-loader.tsx`
  - `web/apps/web/core/components/ui/profile-empty-state.tsx`
  - `web/apps/web/core/components/user/{index.ts,user-greetings.tsx}`
  - `web/apps/web/core/components/web-hooks/empty-state.tsx`
  - `web/apps/web/core/components/workspace/ConfirmWorkspaceMemberRemove.tsx`
  - `web/apps/web/core/components/workspace/sidebar/help-section/index.ts`
  - `web/apps/web/core/components/workspace/views/header.tsx`
  - `web/apps/web/core/hooks/store/use-inbox-issues.ts`
  - `web/apps/web/core/hooks/store/workspace-draft/use-workspace-draft-issue-filters.ts`
  - `web/apps/web/core/hooks/use-debounced-duplicate-issues.tsx`
  - `web/apps/web/core/services/app_config.service.ts`
  - `web/apps/web/helpers/{emoji.helper.tsx,react-hook-form.helper.ts}`
  - `web/packages/editor/src/components/menus/block-menu-options.tsx`
  - `web/packages/editor/src/extensions/table/table/icons.ts`
  - `web/packages/editor/src/plugins/highlight.ts`
  - `web/packages/i18n/src/constants/index.ts`
  - `web/packages/i18n/src/hooks/index.ts`
  - `web/packages/propel/src/emoji-icon-picker/emoji/index.ts`
  - `web/packages/propel/src/emoji-icon-picker/icon/index.ts`
  - `web/packages/propel/src/icons/user-activity-icon.tsx`
  - `web/packages/shared-state/src/store/{user.store.ts,workspace.store.ts}`
  - `web/packages/types/src/base-layouts/{base.ts,index.ts,kanban.ts,list.ts}`
  - `web/packages/types/src/issues/issue_subscription.ts`
  - `web/packages/utils/src/rich-filters/factories/configs/index.ts`
  - `web/packages/utils/src/work-item-filters/configs/filters/shared.ts`
- 新增（1）：
  - `web/apps/web/core/components/issues/issue-detail/links/types.ts`
- 修改（114）：
  - `.github/workflows/ci.yml`
  - `Makefile`
  - `README.md`
  - `knip.jsonc`
  - `docs/v0/frontend-changes.md`
  - `tools/keywords.json`
  - `web/apps/web/core/components/auth-screens/header.tsx`
  - `web/apps/web/core/components/comments/comment-create.tsx`
  - `web/apps/web/core/components/dropdowns/constants.ts`
  - `web/apps/web/core/components/icons/attachment/index.ts`
  - `web/apps/web/core/components/inbox/modals/create-modal/create-root.tsx`
  - `web/apps/web/core/components/issues/attachment/delete-attachment-modal.tsx`
  - `web/apps/web/core/components/issues/issue-detail-widgets/attachments/helper.tsx`
  - `web/apps/web/core/components/issues/issue-detail/issue-activity/root.tsx`
  - `web/apps/web/core/components/issues/issue-detail/label/index.ts`
  - `web/apps/web/core/components/issues/issue-detail/links/{create-update-link-modal.tsx,index.ts,link-list.tsx}`
  - `web/apps/web/core/components/issues/issue-detail/reactions/index.ts`
  - `web/apps/web/core/components/issues/issue-layouts/filters/header/filters/index.ts`
  - `web/apps/web/core/components/issues/issue-layouts/filters/index.ts`
  - `web/apps/web/core/components/issues/issue-layouts/kanban/base-kanban-root.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/spreadsheet/base-spreadsheet-root.tsx`
  - `web/apps/web/core/components/issues/issue-layouts/utils.tsx`
  - `web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`
  - `web/apps/web/core/components/modules/index.ts`
  - `web/apps/web/core/components/navigation/tab-navigation-utils.ts`
  - `web/apps/web/core/components/power-k/core/{shortcut-handler.ts,types.ts}`
  - `web/apps/web/core/components/power-k/ui/modal/command-item-shortcut-badge.tsx`
  - `web/apps/web/core/components/project-states/index.ts`
  - `web/apps/web/core/components/project/dropdowns/filters/member-list.tsx`
  - `web/apps/web/core/components/project/settings/member-columns.tsx`
  - `web/apps/web/core/components/projects/create/attributes.tsx`
  - `web/apps/web/core/components/sidebar/sidebar-item.tsx`
  - `web/apps/web/core/components/views/helper.tsx`
  - `web/apps/web/core/components/web-hooks/form/individual-event-options.tsx`
  - `web/apps/web/core/components/web-hooks/index.ts`
  - `web/apps/web/core/hooks/store/workspace-draft/index.ts`
  - `web/apps/web/core/hooks/{use-intersection-observer.ts,use-local-storage.tsx}`
  - `web/apps/web/core/lib/idle-task.ts`
  - `web/apps/web/core/services/{file.service.ts,timezone.service.ts,user.service.ts,view.service.ts,workspace-notification.service.ts}`
  - `web/apps/web/core/services/issue/issue.service.ts`
  - `web/apps/web/core/services/project/project-state.service.ts`
  - `web/apps/web/core/store/{command-palette.store.ts,power-k.store.ts}`
  - `web/apps/web/core/store/issue/cycle/issue.store.ts`
  - `web/apps/web/core/store/issue/helpers/base-issues-utils.ts`
  - `web/apps/web/core/store/issue/issue-details/root.store.ts`
  - `web/apps/web/core/store/issue/workspace/filter.store.ts`
  - `web/apps/web/core/store/member/utils.ts`
  - `web/apps/web/helpers/cover-image.helper.ts`
  - `web/apps/web/package.json`
  - `web/packages/constants/src/issue/{filter.ts,layout.ts}`
  - `web/packages/constants/src/{navigation.test.ts,workspace.ts}`
  - `web/packages/constants/src/settings/workspace.ts`
  - `web/packages/editor/src/components/editors/editor-container.tsx`
  - `web/packages/editor/src/components/menus/{block-menu.tsx,menu-items.ts}`
  - `web/packages/editor/src/extensions/code-inline/index.tsx`
  - `web/packages/editor/src/extensions/code/code-block.ts`
  - `web/packages/editor/src/extensions/custom-image/{types.ts,utils.ts}`
  - `web/packages/editor/src/extensions/custom-link/extension.tsx`
  - `web/packages/editor/src/extensions/custom-list-keymap/list-helpers.ts`
  - `web/packages/editor/src/extensions/emoji/emoji.ts`
  - `web/packages/editor/src/extensions/slash-commands/root.tsx`
  - `web/packages/editor/src/extensions/table/table/utilities/helpers.ts`
  - `web/packages/editor/src/extensions/trailing-node.ts`
  - `web/packages/editor/src/extensions/unique-id/extension.ts`
  - `web/packages/editor/src/helpers/asset-duplication.ts`
  - `web/packages/editor/src/plugins/drag-handle.ts`
  - `web/packages/i18n/src/core/index.ts`
  - `web/packages/i18n/src/locales/en/{common.json,project-settings.json,project.json,workspace-settings.json,workspace.json}`
  - `web/packages/i18n/src/locales/zh-CN/{common.json,project-settings.json,project.json,workspace-settings.json,workspace.json}`
  - `web/packages/propel/src/banner/helper.tsx`
  - `web/packages/propel/src/card/helper.tsx`
  - `web/packages/propel/src/icons/index.ts`
  - `web/packages/propel/src/pill/pill.tsx`
  - `web/packages/types/{package.json,tsconfig.json}`
  - `web/packages/types/src/charts/index.ts`
  - `web/packages/types/src/{common.ts,index.ts,issues.ts,view-props.ts,workspace.ts}`
  - `web/packages/types/src/cycle/cycle.ts`
  - `web/packages/types/src/rich-filters/config/filter-config.ts`
  - `web/packages/types/src/rich-filters/field-types/shared.ts`
  - `web/packages/ui/src/card/helper.tsx`
  - `web/packages/ui/src/dropdowns/helper.tsx`
  - `web/packages/ui/src/header/helper.tsx`
  - `web/packages/ui/src/popovers/types.ts`
  - `web/packages/ui/src/tables/types.ts`
  - `web/packages/ui/src/tag/helper.tsx`
  - `web/packages/utils/package.json`
  - `web/packages/utils/src/datetime.ts`

**Interfaces**

- Consumes：Task 6 的结果；knip 在 Task 6 结束时报告的 64 个文件、96 个导出、44 个类型。
- Produces：knip 零；`make knip` 是门禁（先生成路由类型，`--treat-config-hints-as-errors`，没有 `--no-exit-code`）；
  `@plane/types` 用 `react-library.json`；`CommentCreate` 的 `projectId: string` 必填；
  `issue-detail/links/types.ts` 导出 `TLinkOperations`；守卫规则 `integrations`。

**Steps**

- [ ] **Step 1: knip 报告的 64 个未使用文件**

Run: `pnpm exec knip --no-exit-code --reporter json | node $P3TMP/knipflat.mjs > $P3TMP/knip-t6.flat; cut -f1 $P3TMP/knip-t6.flat | sort | uniq -c`
Expected: `96 exports`、`64 files`、`44 types`

先在 Plane 上游核对每个文件：`grep '^files' $P3TMP/knip-t6.flat | cut -f2` 就是 `$P3TMP/t7/files1.txt`，
把它写成 knip 文本报告的样子（第一行 `Unused files (64)`）存为 `$P3TMP/t7/knip-files-t6.txt`，然后

Run: `python3 $P3TMP/upstream.py /Users/xiaoruan/project/nerve-project/plane $P3TMP/t7/knip-files-t6.txt > $P3TMP/t7/upstream-t6.txt; grep -c plane-dead $P3TMP/t7/upstream-t6.txt`
Expected: `43`。其余 21 行列出的上游导入方全都在这 64 个文件之内（`base-layouts/` 内部、`readonly/index.tsx`、
`onboarding/create-or-join-workspaces.tsx`、`profile/profile-setting-content-wrapper.tsx`、`user/index.ts`、
shared-state 的 `user.store.ts`），也就是说每个文件在上游也是死的。

64 个文件（`$P3TMP/t7/files1.txt`）：

```
web/apps/web/core/components/api-token/empty-state.tsx
web/apps/web/core/components/auth-screens/workspace/not-a-member.tsx
web/apps/web/core/components/base-layouts/constants.ts
web/apps/web/core/components/base-layouts/hooks/use-group-drop-target.ts
web/apps/web/core/components/base-layouts/hooks/use-layout-state.ts
web/apps/web/core/components/base-layouts/kanban/group-header.tsx
web/apps/web/core/components/base-layouts/kanban/group.tsx
web/apps/web/core/components/base-layouts/kanban/item.tsx
web/apps/web/core/components/base-layouts/kanban/layout.tsx
web/apps/web/core/components/base-layouts/layout-switcher.tsx
web/apps/web/core/components/base-layouts/list/group-header.tsx
web/apps/web/core/components/base-layouts/list/group.tsx
web/apps/web/core/components/base-layouts/list/item.tsx
web/apps/web/core/components/base-layouts/list/layout.tsx
web/apps/web/core/components/base-layouts/loaders/layout-loader.tsx
web/apps/web/core/components/common/new-empty-state.tsx
web/apps/web/core/components/core/sidebar/sidebar-menu-hamburger-toggle.tsx
web/apps/web/core/components/cycles/list/cycle-list-project-group-header.tsx
web/apps/web/core/components/empty-state/comic-box-button.tsx
web/apps/web/core/components/empty-state/helper.tsx
web/apps/web/core/components/inbox/inbox-filter/filters/state.tsx
web/apps/web/core/components/issues/issue-detail/relation-select.tsx
web/apps/web/core/components/issues/label.tsx
web/apps/web/core/components/onboarding/create-or-join-workspaces.tsx
web/apps/web/core/components/onboarding/create-workspace.tsx
web/apps/web/core/components/onboarding/invitations.tsx
web/apps/web/core/components/onboarding/invite-members.tsx
web/apps/web/core/components/onboarding/step-indicator.tsx
web/apps/web/core/components/power-k/actions/helper.ts
web/apps/web/core/components/profile/profile-setting-content-wrapper.tsx
web/apps/web/core/components/project/empty-state.tsx
web/apps/web/core/components/project/multi-select-modal.tsx
web/apps/web/core/components/readonly/cycle.tsx
web/apps/web/core/components/readonly/date.tsx
web/apps/web/core/components/readonly/index.tsx
web/apps/web/core/components/readonly/labels.tsx
web/apps/web/core/components/readonly/member.tsx
web/apps/web/core/components/readonly/module.tsx
web/apps/web/core/components/readonly/priority.tsx
web/apps/web/core/components/readonly/state.tsx
web/apps/web/core/components/settings/layout.tsx
web/apps/web/core/components/sidebar/search-button.tsx
web/apps/web/core/components/ui/loader/notification-loader.tsx
web/apps/web/core/components/ui/profile-empty-state.tsx
web/apps/web/core/components/user/index.ts
web/apps/web/core/components/user/user-greetings.tsx
web/apps/web/core/components/workspace/ConfirmWorkspaceMemberRemove.tsx
web/apps/web/core/components/workspace/sidebar/help-section/index.ts
web/apps/web/core/components/workspace/views/header.tsx
web/apps/web/core/hooks/store/use-inbox-issues.ts
web/apps/web/core/hooks/use-debounced-duplicate-issues.tsx
web/apps/web/core/services/app_config.service.ts
web/apps/web/helpers/emoji.helper.tsx
web/apps/web/helpers/react-hook-form.helper.ts
web/packages/editor/src/extensions/table/table/icons.ts
web/packages/i18n/src/constants/index.ts
web/packages/i18n/src/hooks/index.ts
web/packages/propel/src/emoji-icon-picker/emoji/index.ts
web/packages/propel/src/emoji-icon-picker/icon/index.ts
web/packages/shared-state/src/store/user.store.ts
web/packages/shared-state/src/store/workspace.store.ts
web/packages/types/src/issues/issue_subscription.ts
web/packages/utils/src/rich-filters/factories/configs/index.ts
web/packages/utils/src/work-item-filters/configs/filters/shared.ts
```

Run:

```
git rm -q --pathspec-from-file=$P3TMP/t7/files1.txt
git rm -q -r web/packages/types/src/base-layouts web/apps/web/app/assets/empty-state/api-token.svg \
  web/apps/web/app/assets/empty-state/search/project-dark.webp web/apps/web/app/assets/empty-state/search/project-light.webp
python3 $P3TMP/t7/dead1.py
```

`types/src/base-layouts/` 只有 `base-layouts/` 组件在用；三张图片只有被删的文件提到。`dead1.py` 删 `types/src/index.ts` 里
`base-layouts` 的导出。

文案（`$P3TMP/t7/keys1.txt`，`keyref.mjs orphaned c08cc32` 的 6 个）：

```
# orphaned by the deletion of the files knip reported unused (keyref.mjs orphaned HEAD)
common confirming
common common.no_items_in_this_group
common common.drop_here_to_move
workspace workspace_creation.subheading
workspace workspace_projects.empty_state.filter.title
workspace workspace_projects.empty_state.filter.description
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t7/keys1.txt | tail -1` → `total: 6 keys in each locale`

**类型包的 React 类型**（`types_react.py`）：`@plane/types` 里有 6 个文件通过全局命名空间写 `React.ReactNode` 这类类型、
不带 DOM lib 写 DOM 名字，只因为 `base-layouts` 导入过 `react` 才能编译。每个文件改为 `import type { … } from "react"` 导入
自己用到的类型；`web/packages/types/tsconfig.json` 改为 `extends` 仓库已有的 `react-library.json`（ui、propel 已在用）。

Run: `pnpm exec turbo run check:types` → `Tasks:    23 successful, 23 total`

- [ ] **Step 2: P2 评审第 7 节的清单**

```
git rm -q web/packages/editor/src/plugins/highlight.ts web/packages/editor/src/components/menus/block-menu-options.tsx
python3 $P3TMP/t7/dead2.py
python3 $P3TMP/t7/p2items.py
```

- `dead2.py`：`NodeHighlightPlugin` 从未注册，`editor-container.tsx` 不再给它发 `setMeta` 事务、删掉 `once("focus")` 处理，
  只剩一个子元素的 fragment 收掉。
- `p2items.py`：
  - 工作区设置的 FEATURES 分组没有任何标签页（侧边栏把它藏起来），常量里删掉；
  - `RESTRICTED_URLS` 重复的 `config`、`mobile`、`monitor` 各删一个，`silo`（Plane 的导入服务）随集成删除；
  - 表格菜单的"适应宽度"读的 `--editor-content-width` 没有任何地方定义，删掉这一项（`block-menu-options.tsx`）；
  - 链接扩展删掉 `protocols` 选项和 `onCreate` / `onDestroy` 里的 `registerCustomProtocol` / `reset`：linkify 本来就支持
    `http`、`https`，第一个编辑器之后每个编辑器都打印 `linkifyjs: already initialized` 的噪声随之消失。
- P2 评审点名、没人用的包导出（`$P3TMP/t7/p2-exports.flat`）：

```
exports	web/packages/constants/src/issue/filter.ts	ISSUE_DISPLAY_FILTERS_BY_LAYOUT
exports	web/packages/utils/src/datetime.ts	generateDateArray
exports	web/packages/utils/src/datetime.ts	findTotalDaysInRange
types	web/packages/types/src/cycle/cycle.ts	TCycleProgress
types	web/packages/types/src/workspace.ts	IWorkspaceProgressResponse
exports	web/packages/propel/src/icons/user-activity-icon.tsx	UserActivityIcon
```

Run: `node $P3TMP/unexport.mjs $P3TMP/t7/p2-exports.flat | tee $P3TMP/t7/p2.log`
Expected:

```
delete web/packages/constants/src/issue/filter.ts ISSUE_DISPLAY_FILTERS_BY_LAYOUT
delete web/packages/utils/src/datetime.ts generateDateArray
unexport web/packages/utils/src/datetime.ts findTotalDaysInRange (1 uses in file)
delete web/packages/types/src/cycle/cycle.ts TCycleProgress
delete web/packages/types/src/workspace.ts IWorkspaceProgressResponse
delete web/packages/propel/src/icons/user-activity-icon.tsx UserActivityIcon
EMPTY web/packages/propel/src/icons/user-activity-icon.tsx
```

Run: `python3 $P3TMP/t7/empty.py $P3TMP/t7/p2.log && git rm -q -f --pathspec-from-file=$P3TMP/t7/empty-files.txt`
Expected: `1 files to delete`（`empty.py` 同时删掉桶文件里 `export * from "./user-activity-icon";` 这一行）

集成、导入、从 CSV 导入成员、自动提醒的文案（`$P3TMP/t7/keys2.txt`）：

```
# the integration and import copy (P2 review ruling 10; no reader since P2 deleted the integrations and importers)
common common.import
project-settings project_settings.empty_state.integrations
workspace-settings workspace_settings.settings.integrations
workspace-settings workspace_settings.settings.imports
workspace-settings workspace_settings.empty_state.imports
# member import from CSV: an importer, never wired in this fork
project project.members_import
workspace workspace.members_import
# automatic reminders by email (P2 review 7): email sending is gone
project-settings project_settings.automations.auto-remind
```

Run: `node $P3TMP/i18n-del-list.mjs $P3TMP/t7/keys2.txt | tail -1` → `total: 8 keys in each locale`

```
git rm -q web/apps/web/app/assets/services/jira.svg \
  web/apps/web/app/assets/empty-state/project-settings/{integrations-dark-resp,integrations-dark,integrations-light-resp,integrations-light}.webp \
  web/apps/web/app/assets/empty-state/workspace-settings/{integrations-dark-resp,integrations-dark,integrations-light-resp,integrations-light}.webp
```

- [ ] **Step 3: Plane 不路由、也没人调用的 service 方法**

Run: `python3 $P3TMP/endpoints.py /Users/xiaoruan/project/nerve-project/plane | head -1`
Expected：`240 calls, 12 not routed by Plane`（Task 6 结束时 `c08cc32` 是 241 / 13，多出的一个是 Step 1 删掉的
`app_config.service.ts` 的 `envConfig`；下面删掉 8 个方法之后是 232 / 4）

Run（每行一次调用）：

```
node $P3TMP/delmethods.mjs web/apps/web/core/services/issue/issue.service.ts deleteIssueRelation getIssueDisplayProperties updateIssueDisplayProperties bulkSubscribeIssues
node $P3TMP/delmethods.mjs web/apps/web/core/services/project/project-state.service.ts updateState
node $P3TMP/delmethods.mjs web/apps/web/core/services/user.service.ts userIssues
node $P3TMP/delmethods.mjs web/apps/web/core/services/view.service.ts getViewIssues
python3 $P3TMP/t7/comment_assets.py
node $P3TMP/delmethods.mjs web/apps/web/core/services/file.service.ts updateBulkWorkspaceAssetsUploadStatus
```

`delmethods.mjs` 对 `deleteIssueRelation`、`getViewIssues` 会提示"still called (check the receiver)"：调用方分别是
`issueRelationService.deleteIssueRelation`（`IssueRelationService`，保留）和 `workspaceService.getViewIssues`
（`WorkspaceService`，保留），都不是被删的这几个方法。`comment_assets.py`：评论框只用于工作项，总有项目，
`CommentCreate` 的 `projectId` 改为 `string` 必填，工作区级的上传分支（`POST /api/assets/v2/workspaces/<slug>/<entity>/bulk/`，
Plane 不路由）删除。

Run: `python3 $P3TMP/endpoints.py /Users/xiaoruan/project/nerve-project/plane`
Expected:

```
232 calls, 4 not routed by Plane
web/apps/web/core/services/cycle.service.ts	cycleDistribution	GET	/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/analytics?type=issues	no trailing slash
web/apps/web/core/services/project/project.service.ts	checkProjectIdentifierAvailability	GET	/api/workspaces/${workspaceSlug}/project-identifiers	no trailing slash
web/packages/services/src/developer/api-token.service.ts	retrieve	GET	/api/users/api-tokens/${tokenId}	no trailing slash
web/packages/services/src/developer/api-token.service.ts	destroy	DELETE	/api/users/api-tokens/${tokenId}	no trailing slash
```

这 4 处交给 M6、M3、M2（spec 第 7 节）。

- [ ] **Step 4: 测试、守卫、门禁、文档**

测试（`tests.py`）：`navigation.test.ts` 的 import 加 `RESTRICTED_URLS`，工作区设置那一组加一个测试，末尾加一组：

```ts
  // an empty group was hidden by the sidebar and kept only as a constant (the P2 review's empty FEATURES group)
  it("have no empty sidebar group", () => {
    for (const [category, items] of Object.entries(GROUPED_WORKSPACE_SETTINGS)) {
      expect(items.length, category).toBeGreaterThan(0);
    }
  });
```

```ts
// Workspace addresses a workspace cannot take; the list had config, mobile and monitor twice.
describe("the reserved workspace addresses", () => {
  it("list each address once", () => {
    expect(new Set(RESTRICTED_URLS).size).toBe(RESTRICTED_URLS.length);
  });
});
```

Run: `pnpm --filter @plane/constants test 2>&1 | grep -E "Tests "` → `Tests  11 passed (11)`

守卫规则（`$P3TMP/t7/rules.json`；`github-repository`、应用安装的命中样本取自 P2 删除之前的代码）：

```json
[
  {
    "id": "integrations",
    "phase": "M1/P3",
    "why": "集成、导入器和本地缓存（M1 设计 2.2、7.4 的死代码一行，P2 评审裁定 10）：集成设置、GitHub 同步、Slack 和应用安装、Jira 等导入器、从 CSV 导入成员、注释掉的 IndexedDB 缓存，以及保留地址中的 silo（Plane 的导入服务）。设计中的 app-installation 写成 app[-_]?installation：Plane 的写法是 AppInstallationService 和 app_installation.service；github-repository、app 安装在本 Phase 开始时已没有命中，命中样本取自 P2 删除之前的代码。不搜单独的 github：保留的\"在 GitHub 上加星\"链接",
    "files": {
      "source": "^(?:web/.*\\.(?:[cm]?[jt]sx?|json)|web/apps/web/\\.env\\.example|turbo\\.json)$",
      "flags": ""
    },
    "content": {
      "source": "integration|importer|jira|slack|github-repository|app[-_]?installation|indexeddb|members_import|\"silo\"",
      "flags": "i"
    },
    "samples": {
      "hit": [
        "    id: \"integrations\",",
        "    id: \"importers\",",
        "        title: \"Jira\",",
        "        title: \"Slack\",",
        "      `/api/workspaces/${workspaceSlug}/projects/${projectId}/workspace-integrations/${workspaceIntegrationId}/github-repository-sync/`,",
        "import { AppInstallationService } from \"@/services/app_installation.service\";",
        "//   private indexedDBService: IndexedDBService;",
        "    \"members_import\": {",
        "  \"silo\","
      ],
      "miss": [
        "import githubBlackImage from \"@/app/assets/logos/github-black.png?url\";",
        "    \"star_us_on_github\": \"Star us on GitHub\""
      ],
      "files": {
        "hit": [
          "web/packages/constants/src/workspace.ts",
          "web/packages/i18n/src/locales/en/workspace-settings.json"
        ],
        "miss": [
          "web/apps/web/app/assets/services/jira.svg",
          "docs/v0/M1-frontend-trim/M1-design.md"
        ]
      }
    }
  }
]
```

Run: `node $P3TMP/kw.mjs add-rules $P3TMP/t7/rules.json && pnpm exec oxfmt tools && node tools/keywords.mjs && node $P3TMP/alts.mjs M1/P3`
Expected: `keywords: 42 rules, 4 exceptions, no hits.`、`alternatives or variants without a hit sample: 0`

knip 门禁（`gate.py`）：

- `Makefile` 的 `knip` 目标改为（注释同步）：

  ```
  knip: ## 检查未使用的文件、导出和依赖（需要 Node；门禁）
  	pnpm --filter web exec react-router typegen
  	pnpm exec knip --treat-config-hints-as-errors
  ```

- `knip.jsonc` 删掉 `"web/apps/web": { "ignoreUnresolved": ["\\+types/"] }` 这一块和它的注释，文件头的说明改为"是门禁"；
- `.github/workflows/ci.yml` 的步骤名 `Unused code (report only)` → `Unused code`，注释改为门禁；
- `README.md` 的"未使用的代码"一条改为门禁的说法。

文档（`docs.py`）：`docs/v0/frontend-changes.md` 第二节，P3 的 10 行改为"已完成 | M1/P3"，文字按 spec 2.3–2.9 补全
（原样见 `git show 98025c6 -- docs/v0/frontend-changes.md`）。

- [ ] **Step 5: knip 清零**

`$P3TMP/t7/loop.sh` 反复做：对 `lintdiff.sh` 新报 `no-unused-vars` 的文件跑 `deadlocals.mjs` → knip → `unexport.mjs` →
`empty.py` 删掉不再导出任何东西的文件和它们的桶文件行，直到 knip 为零。

```bash
#!/bin/bash
# T7 step: repeat until nothing changes: delete what the deletions left unused inside files (deadlocals.mjs on the
# files of lintdiff's new no-unused-vars lines), then act on knip's unused exports and types (unexport.mjs) and delete
# the files left exporting nothing with their barrel lines (empty.py). Each round prints what it did; the last line is
# "round N: knip reports nothing".
# usage: bash t7/loop.sh   (from the repository root; $P3TMP is the directory above this script's)
set -eu
T7=$(cd "$(dirname "$0")" && pwd)
P3TMP=$(dirname "$T7")
for round in 1 2 3 4 5 6 7 8 9 10; do
  # lintdiff prints "== <package>" and then "<rule>\t<file in the package>\t<message>" lines
  files=$(bash "$P3TMP/lintdiff.sh" "$P3TMP/base" 2>/dev/null |
    awk -F'\t' '/^== / {pkg = substr($0, 4); next} $1 ~ /no-unused-vars/ {print pkg "/" $2}' | sort -u)
  if [ -n "$files" ]; then
    # shellcheck disable=SC2086
    node "$P3TMP/deadlocals.mjs" $files
  fi
  pnpm exec knip --no-exit-code --reporter json 2>/dev/null | node "$P3TMP/knipflat.mjs" > "$T7/loop.flat"
  if ! grep -q . "$T7/loop.flat"; then
    echo "round $round: knip reports nothing"
    exit 0
  fi
  echo "== round $round: $(grep -c . "$T7/loop.flat") knip findings"
  node "$P3TMP/unexport.mjs" "$T7/loop.flat" | tee "$T7/loop.log"
  python3 "$T7/empty.py" "$T7/loop.log"
  if grep -q . "$T7/empty-files.txt"; then
    git rm -q -f --pathspec-from-file="$T7/empty-files.txt"
  fi
done
echo "knip still reports findings after 10 rounds"
exit 1
```

`empty.py`：

```python
# T7 step: the files unexport.mjs left exporting nothing are deleted (list written to t7/empty-files.txt for
# `git rm --pathspec-from-file`), and the barrel lines that re-exported them go. Index files left empty are listed too.
# usage: python3 empty.py <unexport-log>   (from the repository root; then git rm --pathspec-from-file=t7/empty-files.txt)
import os
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
files = [l.split()[1] for l in open(sys.argv[1]) if l.startswith("EMPTY ")]
gone = set(files)
changed = True
while changed:
    changed = False
    for f in sorted(gone):
        d = os.path.dirname(f)
        stem = re.sub(r"\.(tsx?|jsx?)$", "", os.path.basename(f))
        if stem == "index":  # a deleted index: the parent barrel re-exported its directory
            d, stem = os.path.dirname(d), os.path.basename(d)
        for idx in ("index.ts", "index.tsx"):
            p = os.path.join(d, idx)
            if not os.path.exists(p) or p in gone:
                continue
            s = open(p).read()
            s2 = re.sub(rf'^export \* from "\./{re.escape(stem)}";\n', "", s, flags=re.M)
            if s2 != s:
                open(p, "w").write(s2)
                # an index that re-exports nothing any more is gone as well
                if not re.search(r"^export ", s2, flags=re.M):
                    gone.add(p)
                    changed = True
open(os.path.join(HERE, "empty-files.txt"), "w").write("\n".join(sorted(gone)) + "\n")
print(f"{len(gone)} files to delete")
```

Run: `bash $P3TMP/t7/loop.sh | grep -E '^(== )?round'`
Expected（原型）：

```
== round 1: 140 knip findings
== round 2: 2 knip findings
== round 3: 2 knip findings
round 4: knip reports nothing
```

第 1 轮删掉 25 个文件（旧的筛选组件、附件图标、标签创建、评论表情等）；第 2、3 轮是 `links/links.tsx`、`links/link-detail.tsx`。
每轮的动作写在 `$P3TMP/t7/loop.log`，逐条看：`unexport` 的名字在文件内还有用，`delete` 的没人用。

然后：

```
node $P3TMP/unexport.mjs $P3TMP/t7/orphan.flat
python3 $P3TMP/t7/links.py
git rm -q web/apps/web/app/assets/attachment/img-icon.png web/apps/web/app/assets/empty-state/web-hook.svg
```

- `orphan.flat`（`TIssueLayout`，删掉 `ISSUE_DISPLAY_FILTERS_BY_LAYOUT` 后只剩文件内使用）→ `unexport … TIssueLayout (1 uses in file)`；
- `links.py`：`issue-detail/links/root.tsx` 里没人用的 `IssueLinkRoot` 组件被第 1 轮删掉，文件只剩详情小组件在用的
  `TLinkOperations` 类型，`git mv` 为 `links/types.ts`，桶文件和两个导入它的文件同步；
- 两张图片只有被删的附件图标和 Webhook 空状态提到（`assets-orphaned.py` 对 `c08cc32`）。

Run: `pnpm exec knip --no-exit-code; echo "exit $?"`
Expected: 没有任何报告，`exit 0`

- [ ] **Step 6: 格式、lint 上限、核对**

Run: `pnpm exec turbo run fix:format --output-logs=errors-only && bash $P3TMP/lintdiff.sh $P3TMP/base`
Expected（原型 `98025c6`）：只有固定节奏第 6 步列出的 4 条已知条目：

```
== web/apps/web
eslint(no-shadow)	core/components/project/add-project-members-modal.tsx	'field' is already declared in the upper scope.
eslint-plugin-jsx-a11y(no-autofocus)	core/components/account/auth-forms/password.tsx	The `autoFocus` attribute is found here, which can cause usability issues for sighted and non-sighted users.
eslint-plugin-promise(always-return)	core/components/project/add-project-members-modal.tsx	Each then() should return a value or throw
eslint-plugin-react-hooks(exhaustive-deps)	core/components/issues/issue-layouts/calendar/day-tile.tsx	React Hook useEffect has missing dependencies: 'issues', and 'handleDragAndDrop'
```

Run:

```
node $P3TMP/pkg.mjs web/apps/web/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 568"
node $P3TMP/pkg.mjs web/packages/types/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 0"
node $P3TMP/pkg.mjs web/packages/utils/package.json set-script check:lint "node ../../../tools/lint-cap.mjs 25"
bash $P3TMP/caps.sh | tail -1
```

Expected: `caps total: 713`（web 600 → 568，types 1 → 0，utils 27 → 25）

Run: `node $P3TMP/keycount.mjs && node $P3TMP/keyref.mjs unused | head -1 && node $P3TMP/keyref.mjs orphaned 6d9692b | head -1`
Expected: `en: 1687 keys`、`zh-CN: 1687 keys`、`keys not referenced now: 570`、`keys referenced at 6d9692b and not now: 0`

Run: `node $P3TMP/symref.mjs orphaned 6d9692b`
Expected:

```
exports used elsewhere at 6d9692b and not now: 1
  web/packages/types/src/search.ts	TIssueSearchResponse
```

（保留导出的理由见 Task 3 Step 7。）

Run: `node $P3TMP/assets.mjs > $P3TMP/assets-t7.txt; python3 $P3TMP/assets-orphaned.py $P3TMP/assets-t7.txt 6d9692b` → 没有输出
（`assets.mjs` 的 stderr：`142 of 257 images unreferenced`）

Run: `node $P3TMP/symref.mjs unused web/packages/ | head -1` → `exports under web/packages/ that no other file uses: 355`（交给收尾）

- [ ] **Step 7: 门禁、核对与提交**

Run: `make lint-web 2>&1 | grep -E "keywords:|Tasks:"` → `keywords: 42 rules, 4 exceptions, no hits.` + `Tasks:    52 successful, 52 total`

Run: `make test-web 2>&1 | grep Tasks:` → `Tasks:    15 successful, 15 total`

Run: `make knip; echo "exit $?"`
Expected: 两行命令回显（`pnpm --filter web exec react-router typegen`、`pnpm exec knip --treat-config-hints-as-errors`），`exit 0`

Run: `make build-web 2>&1 | grep Tasks: && node $P3TMP/web-size.mjs`
Expected（原型 `98025c6`）：

```
 Tasks:    11 successful, 11 total
js: 422 files, 7009402 bytes
css: 3 files, 297032 bytes
fonts: 25 files, 3755608 bytes
other: 112 files, 7910515 bytes
largest chunk: assets/use-parse-editor-content-<hash>.js, 1380566 bytes
locale chunks: 34
```

Run: `bash $P3TMP/headers.sh 6d9692b; git diff HEAD | grep -c -F '${"'` → 没有文件名，`0`

Run: `git add -A && git diff --cached --stat 98025c6 | tail -1` → 没有输出，或每处差异都能说明

Run: `git diff --cached --shortstat` → `228 files changed, 228 insertions(+), 7356 deletions(-)`

提交信息：

```
feat(web): delete Plane's dead code, and make knip a gate

knip reported 64 unused files, 96 exports and 44 types. Every file is dead in upstream Plane
as well (its only importers are other dead files), so all of them go, with what only they
used: the base-layouts types, six keys and five images. Unused exports go when nothing in
their file uses them and lose `export` otherwise; files left exporting nothing go with their
barrel lines, and what the deletions left unused inside a file goes (old filter components,
link list, tab preferences, table and list helpers, and so on). The issue-detail links root
keeps only the operations type the detail widgets use, so it becomes links/types.ts.

The types package named React types through the global namespace and DOM names without
the DOM lib; it only compiled because the deleted base-layouts types imported "react". Each
file now imports the React types it names, and the package uses the React library tsconfig.

The P2 review's list: the node highlight plugin was never registered, so the editor container
stops sending it transactions; the table's "Fit to width" read a CSS variable nothing defines;
the link extension no longer registers http and https as custom linkify protocols (the
"already initialized" noise); the workspace settings' empty FEATURES group; RESTRICTED_URLS
listed config, mobile and monitor twice (silo goes with the integrations); the package
exports nothing uses (ISSUE_DISPLAY_FILTERS_BY_LAYOUT, generateDateArray, TCycleProgress,
IWorkspaceProgressResponse, UserActivityIcon; findTotalDaysInRange stays internal); the
integration, import, member-import and auto-reminder copy and images.

Service methods for addresses Plane does not route and nothing calls are deleted; the comment
box always has a project, so its workspace-level upload branch (an unrouted address) goes.

Two constants tests pin the settings group and reserved address cleanups. The integrations
guard rule covers the vocabulary. `make knip` generates web's route types first, fails on any
finding and on stale configuration; web's ignoreUnresolved goes and CI runs it as a gate.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```

---

## 完成后

- [ ] 整个分支对基线：`git diff --shortstat 6d9692b` → `741 files changed, 2366 insertions(+), 25501 deletions(-)`；
  `git diff --stat 98025c6` 为空（或每处差异都在 Task 报告里说明过）。
- [ ] `bash $P3TMP/stats.sh $(git rev-list --reverse 6d9692b..HEAD)` 的 7 行与 spec 2.1 的表一致。
- [ ] 保留行为的核对（M1 设计 7.5，spec 2.13）由控制者在 Task 之后写临时核对脚本：同时拦截 `**/auth/**` 和 `**/api/**`，
  脚本没有列出的请求一律算失败；全文、假数据、运行命令和断言写进 P3 review 的附录。
- [ ] 交接（spec 第 7 节）写进对应 M 的 `handoffs/`；M1 设计 12 节 P3 一行在合并时改为"已完成"并填上 review。
- [ ] S1–S4 通过；持续集成的 `server`、`web`、`e2e` 三个任务通过（`web` 任务里 `make knip` 已是门禁）。
