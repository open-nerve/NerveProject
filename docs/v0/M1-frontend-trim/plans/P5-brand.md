# M1/P5 品牌与包名 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 工作区的包名 `@plane/*` 改为 `@nerve/*`（纯机械的提交，锁文件只随改名变化），导入的分组注释随之改名；Nerve 的图标、标志、网站图标和轻量的加载动画替换 Plane 的；文案、标题、元数据说 Nerve；指向 Plane 服务的链接删除，文档和问题反馈指向 Nerve 的仓库；标识符、存储键、剪贴板类型、组件名和注释不再带 plane；守卫加入 `plane-package`、`brand`、`brand-files`，Nerve 自己写的文件标注 Nerve 的版权，`@makeplane/propel` 的来源记进前端改动清单。

**Architecture:** 7 个 Task，每个一个提交，顺序是 M1 设计 9 节 P5 的三步：Task 1 包名 → Task 2 导入分组注释 → Task 3 图形 → Task 4 文案和元数据 → Task 5 外部链接 → Task 6 标识符、存储键、剪贴板类型、组件名和注释 → Task 7 守卫规则、版权、来源和文档。每一处改动都由 `$P5TMP` 下的一次性脚本做精确替换（原文不在就停下、什么都不写），脚本不进仓库；两个手写的文件（Task 3）全文写在本计划里。共享文件（`tools/keywords.json`、en 和 zh-CN 的文案、`@nerve/constants` 的 `metadata.ts`）由当前 Task 一次改完。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3；Playwright 1.63.0（取自 `e2e/`，只用来渲染图标和改图片里的字）。

**Spec:** `docs/v0/M1-frontend-trim/specs/P5-brand.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

**原型：** `$P5TMP/proto`，分支 `worktree-m1-p5-brand`，基点 `96d8c1d`。每个 Task 的原型提交：
T1 `40a54db`、T2 `be5666a`、T3 `a559e05`、T4 `61538ba`、T5 `6b440f0`、T6 `89ba562`、T7 `36680de`。本计划的每个数字都来自对应的原型提交；按本计划的脚本步骤在基点上重放，7 个 Task 得到的树都与原型提交相同（`$P5TMP/tools/replay.mjs`）。

---

## Global Constraints

P4 计划的 Global Constraints 和"控制者评审补充"在本 Phase 继续有效（下面是它们在 P5 的写法），P4 评审第 4 节的裁定也照旧；与 P4 冲突时以本节为准。

### 命令与环境

- **所有命令在 worktree 的根目录执行**，不要 `cd`。要在某个包里执行时用 `pnpm --dir <目录>`；要在另一个克隆的根目录跑工具时用 `bash $P5TMP/at.sh <目录> <命令…>`。开始之前执行一次 `pnpm install --frozen-lockfile`；不做任何全局安装，不运行 `corepack enable`。
- **`$P5TMP`** 是
  `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p5`。
  原型留下的脚本都在那里，不需要另装依赖（Playwright 取自 `e2e/`）；不要用裸的 `/tmp`。每个 Task 开始时核对一次 `ls $P5TMP/t1/rename.mjs $P5TMP/lib/edit.mjs $P5TMP/t3/brand/mark.svg`；不在时从备份恢复：`mkdir -p $P5TMP`，再 `tar -xf .superpowers/sdd/P5-brand/p5tmp.tar -C $P5TMP`。
- **基线副本** `$P5TMP/base`（`96d8c1d`，已安装、已构建）：`lintdiff.sh` 拿它比较新增的 oxlint 警告，Task 7 在它上面核对规则的命中，也是构建体积的基线。没有时 `git clone -q . $P5TMP/base`、`git -C $P5TMP/base checkout -q 96d8c1d`（两次调用），再 `pnpm --dir $P5TMP/base install --frozen-lockfile`、`bash $P5TMP/at.sh $P5TMP/base make build-web`。
- **原型的提交**：先取进本仓库（只需一次，不建分支、不改引用）：
  `git fetch .superpowers/sdd/P5-brand/proto.bundle worktree-m1-p5-brand`。
  之后 `git show <原型提交> -- <路径>` 看某个文件的结果，`git diff --cached --stat <原型提交>` 在提交前比较整棵树。
- **一个 Bash 调用只跑一个 git 命令**：不把多个 git 命令、`cd` 和 git 串在一起，git 命令后面不接管道，不在 git 命令里用花括号展开，不写调用 git 的 shell 循环；多步逻辑写成 `$P5TMP` 下的脚本。不用 `git stash`、`git clean`、`git reset --hard`；不做浅拉取；不碰 `plane/`、`refer/`；不停、不改 Docker 容器。
- **`$P5TMP/tools/replay.mjs`、`orphans-at.mjs` 只对原型克隆用**（它们在原型克隆的工作树里检出、提交）；`pertask.mjs` 只读提交的树，在哪个仓库都能用。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写（只有 Task 1 会改它）；PNG、ICO、WebP 只由 `rasterize.mjs`、`retext.mjs` 生成。
- **文件内容用编辑工具写**，不用带反引号的 heredoc。

### 沿用的裁定（P2–P4 评审，P4 计划的控制者评审补充）

1. **原型提交是对照答案，不能整体照搬。** 按 Task 的步骤一步步做，每一步都跑写明的命令，把实测输出和预期并排写进报告。
   - 可以用 `git show <原型提交> -- <路径>` 看一个文件的结果，可以运行 `$P5TMP/t<N>/` 的脚本。条件是：跑完一个脚本都要读它的 diff，确认每一处改动都对应本 Task 的一条说明。
   - **禁止以提交、目录或文件列表为单位取原型的内容**：`git checkout <原型提交> -- <目录或多个文件>`、`git cherry-pick`、`git read-tree`、`git diff … | git apply` 都不许用。
   - 与原型的差异可以有，每一处都写明原因；原型里你认为错的地方，照你认为对的做，并写进报告。
2. **基点是本分支上一个 Task 的实际提交**（Task 1 是 `96d8c1d`）；`<基点>` 出现在 `infile-orphans.mjs`、`symref.mjs orphaned`、`dangling.mjs` 里。对**整个 Phase 基线**的核对（`headers.sh 96d8c1d`、`keyref.mjs orphaned 96d8c1d`）仍然用 `96d8c1d`。
3. **数字以实测为准。** lint 上限只降不升；P5 不改任何上限（合计 711），实测不是 711 时先找原因。`check:types` 23 个任务、`make lint-web` 52 个、`make test-web` 16 个、`make build-web` 11 个；vitest 77 个测试。
4. **删除，不隐藏；退化结构收掉；孤儿由造成它的 Task 删**（包括同一文件内的使用者、对象成员和 prop）；以删掉的东西命名的文案和图片随之删除。提交前在本 Task 新增的行里查退化结构：`git diff --cached -U0` 的输出存到 `$P5TMP/logs/t<N>-added.txt`，再 `grep -E '^\+' $P5TMP/logs/t<N>-added.txt`，其中不能有 `={"`、`${"`，也不能有不带插值的模板字面量（原型整个 Phase 都没有）。
5. **导入分组不重排**（P4 裁定 10）：换掉一个导入时，新导入放在原来那一组，不为此移动别的导入。
6. **knip 是门禁**，每个 Task 结束时 `make knip` 为零。
7. **测试输出要干净**：`bash $P5TMP/tests.sh` 除 editor 包的 5 行 `prosemirror-codemark` 之外没有 `!` 开头的行。
8. **守卫**：每条规则的每个顶层分支和 `(?:a|b)` 的每一支都有真实代码里的命中样本（`node $P5TMP/alts.mjs M1/P5` 输出 `alternatives or variants without a hit sample: 0`）；例外精确到原文、带 `count` 和 `until`，本 Phase 只有 Task 7 的 1 条。
9. **报告放在最后一条回复里**（子 agent 不能写报告文件）：提交哈希；每一步的命令、实测输出和预期；与原型的差异及原因；下面的风险点；没做到的事和疑问。状态写 `DONE`、`DONE_WITH_CONCERNS` 或 `BLOCKED`。
10. **执行方式沿用 P4**：实现者不指定模型（继承 Opus 5.5），各 Task 评审用 sonnet，整分支评审用 opus；控制者在 Task 3、6 之后和修复轮之后跑浏览器核对（最后一节），并在基线的构建上做反向对照。

### 每个 Task 的固定节奏

1. **改代码**：按 Task 的步骤运行脚本，读 diff。
2. **类型检查**：`pnpm exec turbo run check:types` → `Tasks:    23 successful, 23 total`。报到没动过的行上时，先 `rm -f web/apps/web/.turbo/tsconfig.tsbuildinfo web/packages/*/.turbo/tsconfig.tsbuildinfo` 再跑一次。
3. **孤儿核对**（`<基点>` 见裁定 2）：
   - `node $P5TMP/infile-orphans.mjs <基点>`：列出的每一行都必须以 `defined now: 0)` 结尾；预期见各 Task。
   - `node $P5TMP/symref.mjs orphaned <基点>` → `exports used elsewhere at <基点> and not now: 0`。
   - `node $P5TMP/dangling.mjs <基点>` → `new dangling import comments: 0`。
   - `node $P5TMP/keyref.mjs orphaned 96d8c1d` → `keys referenced at 96d8c1d and not now: 0`；`bash $P5TMP/headers.sh 96d8c1d` 没有输出。
4. **品牌命中**：`node $P5TMP/tools/brandhits.mjs --summary`，最后一行与各 Task 写的相同（分类是启发式的，总数用的是 `brand` 规则的正则，是准的）。
5. **lint**：`bash $P5TMP/lintdiff.sh $P5TMP/base` 没有输出；`bash $P5TMP/caps.sh | tail -1` → `caps total: 711`。
6. **门禁**：`bash $P5TMP/gates.sh t<N>` → 五行，`types` 23、`lint` 带 `keywords:` 一行和 52、`test` 16、`build` 11、`knip` exit 0；再 `bash $P5TMP/tests.sh`。
7. **与原型对比**：`git add -A`，`git diff --cached --stat <原型提交>` 为空或每处差异都有说明。
8. **提交**：一个 Task 一个提交，提交信息用英文（原型的在 `$P5TMP/t<N>/msg.txt`，可以照用或改写），最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只能有本 Task 的改动；**出现 `?? .superpowers/` 就停下来报告**。

### 风险点（每个 Task 报告必答）

1. **`docs/` 以外没有 `@plane/`**（Task 1 起）：`git grep -c -F "@plane/" -- . ':!docs'` → 只有 `tools/keywords.json:6`（`plane-package` 规则的样本）。
2. **锁文件只随改名变化**：Task 1 是 `node $P5TMP/t1/lockcheck.mjs 96d8c1d` → `lockfile differs from the renamed base in 0 lines`；Task 2 起 `git diff --stat <Task 1 的提交> -- pnpm-lock.yaml pnpm-workspace.yaml package.json` 没有输出。
3. **没有升级任何依赖**：同上；Task 1 另报 `pnpm -r ls --depth -1`（包名变了，版本都不变）。
4. **本 Task 改到的页面上看不到 Plane**：列出本 Task 改到的页面或组件，逐个写明改后显示什么（文字、图片）；改了图片的，写出文件、大小和 SHA-256 前 16 位，并看过它（用读图工具打开 PNG；WebP、SVG 先用 `$P5TMP/t3/crop.mjs` 渲染成 PNG）。
5. **存储键读写一致**：本 Task 改到的存储键、剪贴板类型，列出它的每个读取方和写入方（`git grep -n -F <键> -- web`），确认一起改了；不写迁移（spec 第 1 节）。没有改到的写"无"。
6. **没有新的共享或模块级可变状态**：`git diff --cached -U0 -- web` 存到文件后，`grep -E '^\+(export )?(let|var|const|class) '` 列出的每个新增顶层声明写明它是组件、函数还是只读数据；不能有 `let`、`var`。
7. **版权声明按来源**：本 Task 新增的文件带 Nerve 的声明（`Copyright (c) 2026-present OpenNerve`、`SPDX-License-Identifier: AGPL-3.0-only`；SVG 用 XML 注释；放不下的登记在同目录的 `SOURCES.md`）；改写的 Plane 文件保留 Plane 的声明。

### 一次性脚本

都在 `$P5TMP`（备份 `.superpowers/sdd/P5-brand/p5tmp.tar`），从仓库根目录运行。P4 的同名脚本照搬（`symref`、`infile-orphans`、`keyref`、`keycount`、`headers`、`kw`、`alts`、`caps`、`lintdiff`、`stats`、`gates`、`tests`、`assets`、`web-size`、`lock-diff`、`pkg`），只有 `dangling.mjs` 改了一处（原型的教训 3）。

| 脚本 | 用途 |
|---|---|
| `gates.sh <tag> [gate…]`、`tests.sh` | 五个门禁，日志在 `$P5TMP/logs/<tag>-<gate>.log`；vitest 数量和记录在案之外的 stderr |
| `caps.sh`、`lintdiff.sh <基线副本>` | oxlint 上限；比基线多出来的警告 |
| `symref.mjs orphaned <rev>`、`infile-orphans.mjs <rev>`、`dangling.mjs <rev>`、`headers.sh <rev>` | 孤儿、同文件孤儿、悬空的导入分组注释、许可证头 |
| `keyref.mjs orphaned <rev>`、`keycount.mjs` | 文案键（严格方法） |
| `kw.mjs add-rules \| add-exceptions \| set-phase`、`alts.mjs <phase>` | 改 `tools/keywords.json`；规则分支的命中样本 |
| `assets.mjs` | 没有引用的图片（列表在 stdout，`<n> of <m> images unreferenced` 在 stderr） |
| `web-size.mjs`、`stats.sh <sha>…`、`at.sh <目录> <命令…>` | 构建体积；提交的 shortstat 和 D/M/R/A；在另一个克隆的根目录跑命令 |
| `lib/edit.mjs`、`lib/locale.mjs` | 各 Task 脚本用的精确替换（原文必须正好出现一次）和按键改 en、zh-CN 文案 |
| `tools/brandhits.mjs [--summary] [<rev>]` | `brand`、`brand-files` 规则会看到的命中，按类别；给 `<rev>` 时读那个提交的树 |
| `tools/pertask.mjs <rev>…`、`tools/replay.mjs [<Task>]`、`tools/orphans-at.mjs <工作树> <基点> <提交>…` | spec 2.2 的中间值；在原型克隆的工作树里重放本计划；在原型的各提交上重算孤儿核对 |
| `tools/contact.mjs`、`t3/crop.mjs` | 图片的联系表；把一张图的一块渲染成 PNG |
| `t1/rename.mjs`、`t1/lockcheck.mjs <rev>`、`t1/rules.json` | Task 1：`@plane/` → `@nerve/`；锁文件等价核对；规则 `plane-package` |
| `t2/survey.mjs`、`t2/labels.mjs`、`t2/verify.mjs` | Task 2：分组注释的清点、改名、只改了注释的核对 |
| `t3/brand/`、`t3/rasterize.mjs`、`t3/retext.mjs`、`t3/edits.mjs` | Task 3：SVG 和 `SOURCES.md`；渲染 PNG、ICO；改导览图片里的字；代码的替换 |
| `t4/edits.mjs`、`t5/edits.mjs`、`t6/edits.mjs` | Task 4–6 的精确替换 |
| `t7/newfiles.mjs`、`t7/origin.mjs`、`t7/added-p1-p4.txt`、`t7/headers.mjs`、`t7/rulehits.mjs`、`t7/rules.json`、`t7/exceptions.json`、`t7/docs.mjs` | Task 7：迁入后新增的文件；它们有多少来自 Plane（及 P1–P4 新增的 18 个文件的列表）；改版权声明；规则在某棵树上的命中；规则和例外；文档 |
| `probe/visual.mjs`、`probe/lib.mjs` | 原型的浏览器核对（截图和页面文字，最后一节；`lib.mjs` 抄自 P4） |
| `t<N>/msg.txt` | 原型的提交信息 |

### 原型的教训（执行前必读）

1. **oxfmt 也格式化 Markdown**（Task 3）：`SOURCES.md` 里的表格要跑一次 oxfmt，否则 web 的 `check:format` 失败。
2. **Task 4 的两处替换留下 oxfmt 要重排的行**（`peek-overview/properties.tsx` 合成一行、`workspace-wrapper.tsx` 的 `<img>` 合成一行）：脚本之后跑 `pnpm exec oxfmt web/apps/web`。
3. **P4 的 `dangling.mjs` 把后面不是导入的整个注释块都算作悬空**：Task 6 改写路由测试开头的说明时报了一条假阳性。P5 的副本在这种情况下只数形如 `// hooks` 的标签行（不分大小写），各 Task 的预期都是 0。
4. **`rasterize.mjs`、`retext.mjs` 在同一台机器上是确定的**：重放得到逐字节相同的文件。哈希对不上（例如 Chromium 升级了）时，打开图片看过、写进报告，不要改脚本去凑哈希。
5. **路由模块加载失败时 React Router 直接刷新页面，每次都刷新**（生产构建的 `routeModules.js`），不进 `ErrorBoundary`；要看错误页，用渲染错误或组件里 `lazy()` 的分块失败（最后一节）。
6. **"Plane" 也是普通单词**：`tailwind-config/AGENTS.md` 用它表示层次。规则不区分大小写，这类地方按含义改写（Task 6），不开例外。

### 文件结构

| 位置 | 改动 | Task |
|---|---|---|
| `web/**`（1099 个文件）、`tools/keywords.json`、`pnpm-lock.yaml` | `@plane/` → `@nerve/`；规则 `plane-package`，`phase` 改为 `M1/P5` | 1 |
| `web/**`（671 个文件） | 分组注释 `// plane …` | 2 |
| `web/apps/web/app/assets/brand/`（新增）、`public/icons/`、`public/site.webmanifest.json`、`app/root.tsx`、`core/components/common/{nerve-logo,logo-spinner}.tsx`、7 处调用方、`auth-screens/`、propel 的 `icons/`、两张维护页插图、`onboarding/issues.webp` | 图形 | 3 |
| en、zh-CN 的 8 个命名空间；`@nerve/constants` 的 `metadata.ts`；19 个页面和组件 | 文案、标题、元数据 | 4 |
| `metadata.ts`（`REPOSITORY_URL`）、帮助菜单、命令面板、错误页、维护页、登录表单、顶部栏、两个邀请页、项目设置页、`README.md` | 外部链接 | 5 |
| `version-number.tsx`、`stale-asset-error.ts`、editor 的剪贴板、propel 和 ui 的 12 个组件、4 处注释、`string.ts`、editor 的 `package.json` 和 Readme、`tailwind-config/AGENTS.md` | 标识符、存储键、剪贴板类型、组件名、注释 | 6 |
| `tools/keywords.json`、12 个 Nerve 写的文件、`docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P6-knip-notes.md` | 规则 `brand`、`brand-files` 和 1 条例外；版权声明；文档 | 7 |

---

## 控制者评审补充（执行前必读）

控制者在执行前复核了 spec 和本计划（原型 `40a54db..36680de` 的数字、每个 Task 的步骤、浏览器核对）。本节与上文冲突时以本节为准。

### spec 第 3 节和第 7 节的裁定

- 第 3 节第 1–16 条全部采纳。
- 第 7 节的三个待定项：版本号交 M8；新文件头的持有者是 OpenNerve；`web/` 以外不加文件头（spec 7.2）。封面照片交收尾；propel 的对应源码交 M8（spec 7.1、7.3）。

### 对计划的修正

1. **新文件头的持有者是 `OpenNerve`**：`Copyright (c) 2026-present OpenNerve`，与 README 的"Copyright © 2026 OpenNerve"一致。
   - `$P5TMP` 里写这一行的 7 个文件（`t3/brand/` 的三个 SVG 和两个 `SOURCES` 文件、`t7/headers.mjs`、`t7/docs.mjs`）已由控制者改好，`p5tmp.tar` 已更新；上文 Task 3 Step 2 的 `nerve-logo.tsx` 按这一行写。
   - 因此 Task 3、7 与原型提交的差异里会有这几行版权声明，三个 SVG 各小 9 字节（`mark.svg` 580、`lockup.svg` 1267、`lockup-on-dark.svg` 1194），构建的 other 相应小 27 字节；这是说明过的差异，写进报告。
   - XML 注释不参与渲染，`rasterize.mjs` 的 PNG、ICO 哈希应与 Task 3 Step 3 相同；不同时打开图片看过、写进报告（原型的教训 4）。
   - 风险点第 7 条和 Task 7 Step 6 的 `git grep -c` 都按 `Copyright (c) 2026-present OpenNerve`。
2. **执行方式沿用 P4**：
   - Task 1、2 是机械改动。评审包由控制者先在基点的干净克隆上重放脚本，确认提交的差异与脚本写出的相同，评审者只判断每一类替换的意思是否不变。
   - Task 3–7 按常规评审；Task 3 的评审者要打开新图片看过。
3. **浏览器核对由控制者在 Task 3、6 之后和修复轮之后跑**（最后一节），截图由控制者目视；实现者不写探测脚本。

---

## Tasks

### Task 1: 包名 `@plane/*` 改为 `@nerve/*`；锁文件等价核对；规则 `plane-package`

M1 设计 5"包名"、9 节 P5 第 1 步。spec 2.3。原型提交 `40a54db`（1101 个文件，全部修改，+2999 / −2964）。

**Steps**

- [ ] **Step 1: 记下现状**

Run: `node $P5TMP/tools/pertask.mjs HEAD`
Expected:

```
== 96d8c1d Merge M1/P4: router native
  @plane/ outside docs/: 2963 lines in 1101 files
  guard: phase M1/P4, 45 rules, 4 exceptions
  3	clipboard type | 15	comment | 21	copy en | 21	copy zh-CN | 13	displayName | 5	file name | 20	identifier | 780	import label | 16	markdown | 2945	package name | 1	storage key | 44	text | 23	url | brand hits: 3902, file names: 5
```

- [ ] **Step 2: 替换**

Run: `node $P5TMP/t1/rename.mjs > $P5TMP/logs/t1-rename.txt`，然后 `tail -1 $P5TMP/logs/t1-rename.txt`
Expected: `replaced 2922 in 1100 files`（`web/` 1099 个文件，加上 `tools/keywords.json` 里旧规则样本的 25 行）。脚本只处理 git 跟踪的文本文件，跳过 `docs/` 和 `pnpm-lock.yaml`；`@makeplane/` 的 `plane` 前面没有 `@`，碰不到。

- [ ] **Step 3: 锁文件**

Run: `pnpm install`，然后 `node $P5TMP/t1/lockcheck.mjs 96d8c1d`
Expected: `lockfile differs from the renamed base in 0 lines`（锁文件改了 41 行，全是包名）。

Run: `pnpm -r ls --depth -1`
Expected: 16 个包：`nerve`、`@nerve/e2e`、`web@1.4.2`、`@nerve/api-client`，以及 `@nerve/constants`、`editor`、`hooks`、`i18n`、`propel`、`services`、`shared-state`、`tailwind-config`、`types`、`typescript-config`、`ui`、`utils`，都是 `@1.4.2`、`(PRIVATE)`。`knip.jsonc` 不改（工作区路径是目录）。

- [ ] **Step 4: 守卫**

Run: `node $P5TMP/kw.mjs add-rules $P5TMP/t1/rules.json`，`node $P5TMP/kw.mjs set-phase M1/P5`，`pnpm exec oxfmt tools`，`node tools/keywords.mjs`
Expected: `keywords: 46 rules, 4 exceptions, no hits.`（`plane-package`：`@plane/`，文件范围 `^(?!docs/|tools/keywords\.json$)`；4 个命中样本、3 个不命中样本、文件样本各 4 个和 2 个，见 `t1/rules.json`）。

Run: `node $P5TMP/alts.mjs M1/P5` → `alternatives or variants without a hit sample: 0`；`git grep -c -F "@plane/" -- . ':!docs'` → `tools/keywords.json:6`。

- [ ] **Step 5: 核对、门禁与提交**

固定节奏第 2–6 步：`infile-orphans.mjs 96d8c1d` 没有列出任何行（`exports used at 96d8c1d and unused now in changed files: 0`）；`brandhits.mjs --summary` 最后一行 `brand hits: 964, file names: 5`（包名一类没有了；`import { PlaneLogo } from "@nerve/propel/icons"` 这类 7 行里的 `PlaneLogo` 转入"标识符"，27 处）。
`git diff --cached --stat 40a54db` 为空；提交信息见 `$P5TMP/t1/msg.txt`。

---

### Task 2: 导入的分组注释 `// plane …` 改名，悬空的删除

M1 设计 5、P3 评审第 7 节的 P5 一行。spec 2.4。原型提交 `be5666a`（671 个文件，全部修改，+755 / −788）。

**Steps**

- [ ] **Step 1: 改名**

Run: `node $P5TMP/t2/labels.mjs > $P5TMP/logs/t2-labels.txt`，然后 `tail -1 $P5TMP/logs/t2-labels.txt`
Expected: `A renamed 706, A deleted (dangling) 21, B deleted (dangling) 7, B merged 5, B relabelled 49; 671 files`。规则（脚本开头的注释）：
- A 类（讲包名：`// plane imports`、`// Plane Imports`、`// plane hooks`、`// plane-i18n` 等）：`plane` 改为 `nerve`，保留大小写，函数体里的也改；
- B 类（Plane 的 CE/EE 目录：`// plane web imports`、`// plane web components` 等）：改为下一行导入的来源——`@nerve/*` → `nerve imports`，`@/components/` → `components`，`@/hooks/` → `hooks`，`@/services/` → `services`，`@/store/` → `store`，`./`、`../` → `local imports`；紧挨着的上一组（中间只有导入）已经是这个标签时合并，即删除这一行；
- 两类中悬空的（它所在的注释块后面不是导入，或它不是块的最后一行）删除。

逐类抽看 `$P5TMP/logs/t2-labels.txt`，每类至少 3 行，对照文件确认。

- [ ] **Step 2: 只改了注释**

Run: `node $P5TMP/t2/verify.mjs` → `removed 788, added 755, other 0`（`other` 必须是 0；788 − 755 = 28 行删除 + 5 行合并）。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–6 步；`dangling.mjs <Task 1 的提交>` 为 0；`brandhits.mjs --summary` → `brand hits: 176, file names: 5`（没有"import label"一类）。`git diff --cached --stat be5666a` 为空；提交信息见 `$P5TMP/t2/msg.txt`。

---

### Task 3: Nerve 的图标、标志、网站图标和加载动画

M1 设计 5"Logo"、"新文件的版权"、"二进制品牌资源"，P1 评审第 6 节 (c)。spec 2.5。原型提交 `a559e05`（54 个文件：删 24、新增 12、改 18，+147 / −417）。

**Files**

- 新增：`web/apps/web/app/assets/brand/{mark.svg,lockup.svg,lockup-on-dark.svg,SOURCES.md,favicon-16x16.png,favicon-32x32.png,favicon.ico,icon-180x180.png,icon-512x512.png,og-image.png}`、`web/apps/web/public/icons/SOURCES.md`、`web/apps/web/core/components/common/nerve-logo.tsx`
- 修改：`public/icons/icon-192x192.png`、`icon-512x512.png`、`public/site.webmanifest.json`、`app/root.tsx`、`core/components/common/logo-spinner.tsx`、7 处调用方（`app/(all)/create-workspace/page.tsx`、`app/(all)/invitations/page.tsx`、`core/layouts/auth-layout/workspace-wrapper.tsx`、`core/components/auth-screens/header.tsx`、`core/components/onboarding/header.tsx`、`core/components/onboarding/tour/root.tsx`、`core/components/navigation/app-rail-hoc.tsx`）、`core/components/auth-screens/auth-base.tsx`、`app/assets/instance/maintenance-mode-{light,dark}.svg`、`app/assets/onboarding/issues.webp`、`web/packages/propel/src/icons/{index,registry}.ts`
- 删除：`app/assets/favicon/`（4 个）、`app/assets/icons/`（2 个）、`app/assets/og-image.png`、`app/assets/images/logo-spinner-{dark,light}.gif`、`app/assets/plane-takeoff.png`、`app/assets/users/user-profile-cover-default-img.png`、`public/manifest.json`、`public/icons/icon-348x348.png`、`core/components/auth-screens/footer.tsx`、propel 的 `src/icons/brand/`（8 个）和 `src/icons/sub-brand/`（2 个）

**Steps**

- [ ] **Step 1: 矢量图和来源说明**

Run: `mkdir -p web/apps/web/app/assets/brand`，
`cp $P5TMP/t3/brand/mark.svg $P5TMP/t3/brand/lockup.svg $P5TMP/t3/brand/lockup-on-dark.svg $P5TMP/t3/brand/SOURCES.md web/apps/web/app/assets/brand/`，
`cp $P5TMP/t3/brand/SOURCES-public-icons.md web/apps/web/public/icons/SOURCES.md`，
`pnpm exec oxfmt web/apps/web/app/assets/brand/SOURCES.md web/apps/web/public/icons/SOURCES.md`（原型的教训 1）。

读这 5 个文件：三个 SVG 开头是带 Nerve 版权声明的 XML 注释；`mark.svg` 是 64×64 的深青色（`#155E75`）圆角方块，白线折成 N、两端各一个 `#5EEAD4` 的圆点；两个横版标志是 132×32，左边半尺寸的图标，右边描边写成的小写 "nerve"（`#1F2937` / `#F9FAFB`）。

- [ ] **Step 2: 两个手写的文件**

写 `web/apps/web/core/components/common/nerve-logo.tsx`（新文件，Nerve 的版权声明）：

```tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { cn } from "@nerve/utils";
// assets
import lockupOnDark from "@/app/assets/brand/lockup-on-dark.svg?url";
import lockup from "@/app/assets/brand/lockup.svg?url";
import mark from "@/app/assets/brand/mark.svg?url";

// The brand is a set of files in app/assets/brand/ (SOURCES.md there): a new logo replaces the files, not this code.
// The images keep their own proportions, so give them a height and an automatic width.

type TProps = {
  className?: string;
};

/** The Nerve mark. It reads on light and dark backgrounds alike. */
export function NerveLogo({ className }: TProps) {
  return <img src={mark} alt="Nerve" className={className} />;
}

/**
 * The Nerve lockup: the mark and the wordmark. The wordmark follows the theme through CSS, as the theme is known
 * only in the browser; `onColor` keeps the light wordmark for a coloured background in either theme.
 */
export function NerveLockup({ className, onColor = false }: TProps & { onColor?: boolean }) {
  if (onColor) return <img src={lockupOnDark} alt="Nerve" className={className} />;
  return (
    <>
      <img src={lockup} alt="Nerve" className={cn(className, "dark:hidden")} />
      <img src={lockupOnDark} alt="Nerve" className={cn(className, "hidden dark:block")} />
    </>
  );
}
```

把 `web/apps/web/core/components/common/logo-spinner.tsx` 在 Plane 版权声明之后的内容整个换成（改写的 Plane 文件，保留 Plane 的声明）：

```tsx
// local imports
import { NerveLogo } from "./nerve-logo";

// The mark reads on light and dark backgrounds alike, so the markup does not depend on the theme and hydrates
// cleanly where it is prerendered (HydrateFallback in app/root.tsx).
export function LogoSpinner() {
  return (
    <div className="flex items-center justify-center">
      <NerveLogo className="h-6 w-auto animate-pulse sm:h-11" />
    </div>
  );
}
```

（基线用 `dark:` 在两张 GIF 之间选择；现在只有一张图，不再需要，spec 第 3 节第 3 条。）

- [ ] **Step 3: 渲染 PNG 和 ICO**

Run: `node $P5TMP/t3/rasterize.mjs`
Expected（文件、字节、SHA-256 前 16 位）：

```
web/apps/web/app/assets/brand/favicon-16x16.png	489	93dfe9b3fb1fd051
web/apps/web/app/assets/brand/favicon-32x32.png	927	619cb177489193c5
web/apps/web/app/assets/brand/icon-180x180.png	5095	8935e7dc682916cb
web/apps/web/app/assets/brand/icon-512x512.png	14873	67892675c9f33e78
web/apps/web/public/icons/icon-192x192.png	5055	8321618b7ed3bfca
web/apps/web/public/icons/icon-512x512.png	14873	67892675c9f33e78
web/apps/web/app/assets/brand/favicon.ico	2897	87815b86a2306832
web/apps/web/app/assets/brand/og-image.png	27992	b27543b337835f54
```

用读图工具打开 `og-image.png`、`icon-180x180.png`、`favicon-32x32.png` 看一遍（原型的教训 4）。

- [ ] **Step 4: 导览图片里的字**

Run: `node $P5TMP/t3/retext.mjs`
Expected: `background 255,255,255,255, ink 78,78,78; wrote web/apps/web/app/assets/onboarding/issues.webp, 228386 bytes`（基线 202,198 字节；标题框 {772, 950, 206, 40}，Inter 22px，基线 979）。

Run: `node $P5TMP/t3/crop.mjs web/apps/web/app/assets/onboarding/issues.webp $P5TMP/shots/t3-issues-crop.png 700 900 400 120 2`，打开这张 PNG：卡片标题是 "API integration"，颜色和位置与旁边的卡片一致。

- [ ] **Step 5: 代码**

Run: `node $P5TMP/t3/edits.mjs`
Expected: 12 行 `edited …`（7 处调用方、`auth-base.tsx`、`root.tsx`、`site.webmanifest.json`、propel 的 `icons/index.ts`、`registry.ts`）和 2 行 `deleted line … of …maintenance-mode-{light,dark}.svg`。它做的事：
- `PlaneLogo`（3 处）→ `<NerveLogo className="h-9 w-auto" />`；`PlaneLockup` → `<NerveLockup className="h-5 w-auto" />`（登录页和新手引导的页头），导览里是 `<NerveLockup onColor className="h-10 w-auto" />`；`PlaneNewIcon` → `<NerveLogo className="size-5" />`（应用栏，P4 裁定 9：从不渲染，交收尾）；新导入放在原来的组里；
- `auth-base.tsx` 删 `AuthFooter` 的导入和使用；
- `root.tsx` 的 6 个图片导入改到 `app/assets/brand/`，删 `{ rel: "manifest", href: "/manifest.json" }`；
- `site.webmanifest.json` 的 `name`、`short_name` 为 "Nerve"，`description` 为 "Open-source project management."；
- propel 的 `icons/index.ts` 删 `./brand`、`./sub-brand` 的导出；`registry.ts` 删 `PlaneNewIcon` 的导入、`"sub-brand.plane"` 一项和两处 `// Sub-brand icons`；
- 维护页的两张插图删掉屏幕上 Plane Logo 的那条 `<path d="M131.613 92.7631…`。

- [ ] **Step 6: 删除**

Run（一条 git 命令）：
`git rm -q -r web/apps/web/app/assets/favicon web/apps/web/app/assets/icons web/apps/web/app/assets/og-image.png web/apps/web/app/assets/images/logo-spinner-dark.gif web/apps/web/app/assets/images/logo-spinner-light.gif web/apps/web/app/assets/plane-takeoff.png web/apps/web/app/assets/users/user-profile-cover-default-img.png web/apps/web/public/manifest.json web/apps/web/public/icons/icon-348x348.png web/apps/web/core/components/auth-screens/footer.tsx web/packages/propel/src/icons/brand web/packages/propel/src/icons/sub-brand`

- [ ] **Step 7: 核对、门禁与提交**

- `infile-orphans.mjs <Task 2 的提交>` → 8 行，都以 `defined now: 0)` 结尾：`AuthFooter`、`AccentureLogo`、`DolbyLogo`、`PlaneLockup`、`PlaneLogo`、`SonyLogo`、`ZerodhaLogo`、`PlaneNewIcon`；`exports used at … and unused now in changed files: 8`。
- `node $P5TMP/assets.mjs > $P5TMP/logs/assets-t3.txt` → stderr `133 of 248 images unreferenced`；与基线的列表（`bash $P5TMP/at.sh $P5TMP/base node $P5TMP/assets.mjs`）相比只少了 `favicon/apple-touch-icon.png`、`plane-takeoff.png`、`users/user-profile-cover-default-img.png` 三行（删掉的没有引用的 Plane 图片），没有新增。
- `brandhits.mjs --summary` → `brand hits: 145, file names: 0`。
- 构建之后 `node $P5TMP/web-size.mjs` → `js: 398 files, 6829442 bytes`、`css: 3 files, 296816 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 112 files, 6488876 bytes`、最大 chunk 1379320、语言 chunk 34（两张 GIF 共 1,404,364 字节不在了）。
- 风险点第 4 条：列出 7 处调用方的页面、加载动画、网站图标、清单和两张改过的图片；第 7 条：新增的 12 个文件的声明。
- `git diff --cached --stat a559e05` 为空；提交信息见 `$P5TMP/t3/msg.txt`。之后控制者跑浏览器核对的第一轮（最后一节）。

---

### Task 4: 文案、页面标题和元数据

M1 设计 5 的"页面标题、`SITE_NAME` 等元数据""中英文文案中的 Plane""硬编码的文案"三行。spec 2.6。原型提交 `61538ba`（35 个文件，全部修改，+52 / −131）。

**Steps**

- [ ] **Step 1: 替换**

Run: `node $P5TMP/t4/edits.mjs`
Expected（按顺序）：
- `renamed auth: auth.common.new_to_plane -> new_to_nerve`；
- 5 行 `text <命名空间>: <键>`（`home.empty.create_project.description`、`home.empty.personalize_account.title`、`project_settings.automations.auto-archive.description`、`workspace_settings.copy_key`、`workspace_settings.settings.api_tokens.delete.description`）；
- 7 行 `deleted …`：`common: self_hosted_maintenance_message (2 leaves)`、`empty-state: common_empty_state.not_found (3 leaves)`、`project-settings: project_settings.features.intake.email (3 leaves)`、`project: project_issues.empty_state.no_issues.primary_button.comic (2 leaves)`、`workspace-settings: workspace_settings.empty_state.api_tokens (2 leaves)`、`workspace: …general.primary_button.comic (2 leaves)`、`workspace: …no_projects.primary_button.comic (2 leaves)`；
- 19 行 `edited …`（`workspace-invitations/page.tsx` 一行带 `(3 times)`）。

脚本的内容见 spec 2.6；每一处的新文字在 `$P5TMP/t4/edits.mjs` 里，读 diff 时逐处对照。几处要特别看：
- `metadata.ts`：`SITE_NAME = "Nerve"`，`SITE_TITLE`、`SITE_URL`、`TWITTER_USER_NAME` 删除（没有读取方）；
- `root.tsx`：`APP_TITLE` 删除，标题、`og:title`、`application-name` 用 `SITE_NAME`；`og:url`、`twitter:site` 删除；两个 `image:alt` 为 "Nerve"；
- `page-title.tsx`：`if (title) document.title = title;`（原来 `if` 里的 `?? "Plane…"` 永远不用）；
- `issue-creator.tsx`：`{customUserName}`（只在 `customUserName` 为真的分支里渲染）；
- `inbox-list-item.tsx`：删 `Avatar` 的导入和 `intake@plane.so` 的分支，剩 `{createdByDetails && <ButtonAvatars showTooltip={false} userIds={createdByDetails.id} />}`；`peek-overview/properties.tsx` 删两处 `-intake` 的判断（它们在 `createdByDetails &&` 之内，所以不用 `?.`）；
- 收集箱页面标题收成一次 `t("inbox_issue.page_label", { workspace: currentProjectDetails?.name || "Nerve" })`。

Run: `pnpm exec oxfmt web/apps/web`（原型的教训 2：`peek-overview/properties.tsx`、`workspace-wrapper.tsx` 各重排一处）。

- [ ] **Step 2: 核对、门禁与提交**

- `node $P5TMP/keycount.mjs` → `en: 1601 keys`、`zh-CN: 1601 keys`；`keyref.mjs orphaned 96d8c1d` → 0；`infile-orphans.mjs <Task 3 的提交>` 没有列出任何行。
- `brandhits.mjs --summary` → `brand hits: 71, file names: 0`；剩下的都属于 Task 5、6（链接 19、标识符 5、`displayName` 13、Markdown 16、注释 7、写死的文字 3、剪贴板类型 3、存储键 1，以及 `open_plane_documentation` 的 en、zh-CN 各 2 处）。
- 风险点第 4 条：登录、注册、新手引导、导览、邀请、首页、收集箱、动态、令牌设置、自动归档设置、工作区不存在页，逐个写出改后的文字。
- `git diff --cached --stat 61538ba` 为空；提交信息见 `$P5TMP/t4/msg.txt`。

---

### Task 5: Plane 服务的链接

M1 设计 5 的"Plane 的网址"一行。spec 2.7。原型提交 `6b440f0`（18 个文件：删 4、改 14，+27 / −207）。

**Steps**

- [ ] **Step 1: 替换**

Run: `node $P5TMP/t5/edits.mjs`
Expected: 10 行 `edited …`（`metadata.ts`、帮助菜单 `help-section/root.tsx`、`help-commands.ts`、`app/error/prod.tsx`、`maintenance-message.tsx`、`auth-root.tsx`、`top-navigation-root.tsx`、`workspace-invitations/page.tsx`、`README.md`、`settings/projects/page.tsx`），以及
`renamed power-k: power_k.help_actions.open_plane_documentation -> open_documentation`、
`deleted power-k: power_k.help_actions.join_forum (1 leaves)`、`deleted home: home.star_us_on_github (1 leaves)`。
逐处对照 spec 2.7 的表。几处要特别看：
- `metadata.ts` 新增 `export const REPOSITORY_URL = "https://github.com/open-nerve/NerveProject";`（带一行说明）；帮助菜单和 `help-commands.ts` 从 `@nerve/constants` 导入它，`window.open` 都带 `"noopener,noreferrer"`；`help-commands.ts` 删 `ChatOutline` 的导入；
- `maintenance-message.tsx` 删掉 `linkMap` 和链接之后，只剩一个元素的片段 `<>…</>` 收掉；
- `settings/projects/page.tsx` 删掉链接之后，只包着一个按钮的 `<div className="flex gap-2">` 收掉，`Link`、`cn`、`getButtonStyling` 的导入删除；
- `top-navigation-root.tsx` 删掉 `StarUsOnGitHubLink` 的导入时，它上面的 `// local imports` 一起删（否则悬空）；
- `workspace-invitations/page.tsx` 删掉两张卡片之后 `ShareAltOutline`、`StarOutline` 的导入删除，剩下的四个写成一行。

- [ ] **Step 2: 删除**

Run（一条 git 命令）：
`git rm -q "web/apps/web/core/components/account/terms-and-conditions.tsx" "web/apps/web/app/(all)/[workspaceSlug]/(projects)/star-us-link.tsx" web/apps/web/app/assets/logos/github-black.png web/apps/web/app/assets/logos/github-white.png`

- [ ] **Step 3: 核对、门禁与提交**

- `pnpm exec oxfmt --check` 本 Task 改过的 `.ts`、`.tsx` → `All matched files use the correct format.`
- `infile-orphans.mjs <Task 4 的提交>` → 2 行，`StarUsOnGitHubLink`、`TermsAndConditions`，都以 `defined now: 0)` 结尾。
- `keycount.mjs` → 1599 / 1599；`keyref.mjs orphaned 96d8c1d` → 0；`git grep -n -E "open_plane_documentation|join_forum|star_us_on_github|StarUsOnGitHub|TermsAndConditions" -- web` 没有输出。
- `node $P5TMP/assets.mjs` → stderr `133 of 246 images unreferenced`，列表与 Task 3 相同。
- `brandhits.mjs --summary` → `brand hits: 46, file names: 0`（链接只剩 `string.ts` 的邮箱示例，Task 6 改）。
- 风险点第 4 条：帮助菜单（Documentation、Keyboard shortcuts、版本号）、命令面板的帮助命令（Open keyboard shortcuts、Open documentation、Report a bug）、错误页、维护页、登录表单、顶部栏、两个邀请页、空的项目设置页。
- `git diff --cached --stat 6b440f0` 为空；提交信息见 `$P5TMP/t5/msg.txt`。

---

### Task 6: 标识符、存储键、剪贴板类型、组件名和注释

M1 设计 5 的"代码标识符、本地存储键、组件 `displayName`"一行。spec 2.8。原型提交 `89ba562`（26 个文件，全部修改，+43 / −43）。

**Steps**

- [ ] **Step 1: 替换**

Run: `node $P5TMP/t6/edits.mjs`
Expected: 27 行 `edited …`，其中 13 行带 `(n times)`：propel 的 `badge`、`button`、`card`、`icon-button`、`switch/root` 各 `(1 times)`，`skeleton/root.tsx` `(2 times)`，ui 的 `card`、`content-wrapper`、`header`、`loader`、`row`、`tag` 各 `(1 times)`，以及 `tailwind-config/AGENTS.md` 的第二行 `(2 times)`（它的第一行是 13 处逐一替换）。内容见 spec 2.8。

- [ ] **Step 2: 核对、门禁与提交**

- 风险点第 5 条：`git grep -n -E "chunk_reload|editor-html" -- web` → 4 行：`stale-asset-error.ts` 的 `STALE_ASSET_RELOAD_KEY = "__nerve_chunk_reload"`（读写都用这个常量），`editor-ref.ts`、`markdown-clipboard.ts` 的 `setData("text/nerve-editor-html", …)` 和 `props.ts` 的 `getData("text/nerve-editor-html")`。
- `git grep -n "plane-ui-" -- web` 没有输出（没有代码读取 `displayName`）。
- `infile-orphans.mjs <Task 5 的提交>` → 1 行，`PlaneVersionNumber … defined now: 0)`；`dangling.mjs <Task 5 的提交>` → 0（原型的教训 3）。
- `brandhits.mjs --summary` → `3	comment`、`brand hits: 3, file names: 0`（`pnpm-workspace.yaml` 第 2、3、168 行记录来源的注释，Task 7 登记例外）。
- `git diff --cached --stat 89ba562` 为空；提交信息见 `$P5TMP/t6/msg.txt`。之后控制者跑浏览器核对的第二轮。

---

### Task 7: 守卫规则 `brand`、`brand-files`；Nerve 文件的版权；propel 的来源；文档

M1 设计 5"新文件的版权"、3.10、7.4 的品牌一行、9 节 P5 第 3 步、9.7。spec 2.9。原型提交 `36680de`（16 个文件，全部修改，+113 / −29）。

**Steps**

- [ ] **Step 1: 迁入后新增的文件和它们的来源**

Run: `node $P5TMP/t7/newfiles.mjs`
Expected: `30 files added after 48e1a63`：P5 自己的 12 个（Task 3），其余 18 个是 P1–P4 新增的，第一行版权声明都是 Plane 的。

Run: `node $P5TMP/t7/origin.mjs @$P5TMP/t7/added-p1-p4.txt`（这 18 个文件的列表；每行 `<文件>: <n> of <m> lines found at 48e1a63`，`--lines` 另列出没找到的行）
Expected: `navigation.test.ts`（web）0/82、`profile-index.tsx` 2/5、`sidebar-chart.tsx` 13/16、`home-body.tsx` 4/7、`links/types.ts` 4/4、`project-navigation-dialog.tsx` 56/59、`use-profile-member.ts` 2/17、`vitest.config.ts`（web）0/1、`constants/src/navigation.test.ts` 0/36、`editor-interaction.test.ts` 2/136、`extensions.test.ts` 0/41、`command-items-list.test.ts` 0/10、`editor/vitest.config.ts` 1/5、`vitest.setup.ts` 0/4、`language.test.ts` 0/7、`next-path.test.ts` 0/23、`progress.test.ts` 0/63、`utils/src/theme.ts` 2/2。
判断（spec 2.9）：`home-body.tsx` 是 Plane 的 `home-dashboard-widgets.tsx` 删减而来（同样的结构和类名，`git show 48e1a63:web/apps/web/core/components/home/home-dashboard-widgets.tsx`），`profile-index.tsx` 照 Plane 的重定向模块写成（`git show 48e1a63:web/apps/web/app/routes/redirects/core/profile-settings.tsx`），它们和另外 4 个搬过来的文件保留 Plane 的声明；其余 12 个是 Nerve 写的。

- [ ] **Step 2: 改版权声明**

Run: `node $P5TMP/t7/headers.mjs` → 12 行 `notice …` 和 `12 files`（每个文件开头的 Plane 五行声明必须原样存在，换成 Nerve 的四行声明）。

- [ ] **Step 3: 规则先在两棵树上核对**

Run: `bash $P5TMP/at.sh $P5TMP/base node $P5TMP/t7/rulehits.mjs`
Expected: 5 行 `brand-files	<路径>`（`plane-takeoff.png`、propel 的 `plane-lockup.tsx`、`plane-logo.tsx`、`plane-wordmark.tsx`、`plane-icon.tsx`）、`brand-files: 5 paths`、`brand: 3902 hits in 1165 files`。

Run: `node $P5TMP/t7/rulehits.mjs --files`
Expected: `brand	pnpm-workspace.yaml	3`、`brand: 3 hits in 1 files`、`brand-files: 0 paths`。

- [ ] **Step 4: 加入规则和例外**

Run: `node $P5TMP/kw.mjs add-rules $P5TMP/t7/rules.json`，`node tools/keywords.mjs`
Expected: 以 1 退出，列出 `brand  pnpm-workspace.yaml:2  "Plane"`、`:3`、`:168` 三行和
`keywords: 3 hits without an exception, 0 stale exceptions, 0 mismatched exceptions, 0 expired exceptions (tools/keywords.json).`

Run: `node $P5TMP/kw.mjs add-exceptions $P5TMP/t7/exceptions.json`，`pnpm exec oxfmt tools`，`node tools/keywords.mjs`，`node $P5TMP/alts.mjs M1/P5`
Expected: `keywords: 48 rules, 5 exceptions, no hits.`；`alternatives or variants without a hit sample: 0`。

规则（`t7/rules.json`，spec 2.9 的表）：
- `brand`：`files` `^(?:web/.*|pnpm-(?:workspace|lock)\.yaml|turbo\.json)$`；`content` `(?!(?<=Copyright \(c\) 2023-present )Plane Software, Inc\.)(?!(?<=@make)plane\/propel\b)plane`，flags `i`。两个否定前瞻各自独立（不写成 `(?!a|b)`，`alts.mjs` 按 `|` 拆分支）。6 个命中样本（旧的 `PlaneLogo` 导入、`og:image:alt`、论坛链接、`__plane_chunk_reload`、`plane-ui-button`、`GlobalModals` 的注释）、5 个不命中样本（版权声明行、propel 的导入、catalog 和锁文件里的 `@makeplane/propel`、`__nerve_chunk_reload`），文件样本命中 4 个（每个分支一个）、不命中 4 个。
- `brand-files`：`path` `^web/.*plane`，flags `i`；命中样本是基线的 3 个路径，不命中样本是 `brand/mark.svg`、`nerve-logo.tsx`、propel 的 `icons/index.ts`。
- 例外（`t7/exceptions.json`）：`brand`、`pnpm-workspace.yaml`、`"Plane"`、`count: 3`、`until: "M9"`，理由写明这 3 行记录来源、在 v0 内不会消失。

- [ ] **Step 5: 文档**

Run: `node $P5TMP/t7/docs.mjs` → 3 行 `edited …`（`docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P6-knip-notes.md`）。读 diff：
- 前端改动清单第一节的表在"不使用"之后加"第三方依赖"一行：`@makeplane/propel` 0.3.0 的许可证、tarball 完整性哈希、SLSA 来源证明记录的仓库、目录、提交和工作流，仓库不公开，source map 带着 1376 个源文件中 1375 个的全文；
- 新增 1.5 节（10 行，spec 2.3–2.9 的摘要）；第四节 3 行改为 `已完成 / M1/P5`，另加 3 行（链接、标识符等、Nerve 的版权）；
- README"版权"一节加一句 Nerve 文件的声明和 `SOURCES.md`；
- `M0-P6-knip-notes` 的 `status` 改为 `closed`，加"处理结果（M1/P5）"。

- [ ] **Step 6: 整个 Phase 的核对、门禁与提交**

- 固定节奏第 2–6 步；`brandhits.mjs --summary` → `3	comment`、`brand hits: 3, file names: 0`（都是登记的例外）。
- 对 Phase 基线：`symref.mjs orphaned 96d8c1d`、`keyref.mjs orphaned 96d8c1d` → 0；`dangling.mjs 96d8c1d` → 0；`headers.sh 96d8c1d` 没有输出；`infile-orphans.mjs 96d8c1d` → 11 行（Task 3 的 8 个、Task 5 的 2 个、Task 6 的 1 个），都以 `defined now: 0)` 结尾。
- `git grep -c "Copyright (c) 2026-present OpenNerve" -- web` → 18 个文件。
- 构建之后 `node $P5TMP/web-size.mjs` → `js: 397 files, 6818890 bytes`、`css: 3 files, 296816 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 110 files, 6457945 bytes`、`largest chunk: … 1379320 bytes`、`locale chunks: 34`（基线见 spec 2.2）。
- 全 Phase 新增的行（`git diff -U0 96d8c1d -- web` 存到文件后查）没有 `={"`、`${"`、`let`、`var` 的顶层声明。
- `git diff --cached --stat 36680de` 为空；提交信息见 `$P5TMP/t7/msg.txt`。

---

## 控制者的浏览器核对（M1 设计 7.5 的 P5 一行）

由控制者写脚本、跑、把全文和输出写进 P5 review 的附录；实现者不写。做法沿用 P4 review 附录 6：在 `$P5TMP/probe-app`（本仓库的克隆，检出到被测提交，`make build-web`）上用 node 起静态服务器提供 `web/apps/web/build/client`（单页应用回退到 `index.html`），Playwright 取自 `e2e/`，`/api/`、`/auth/` 的请求全由桩回答，场景没有列出的请求一律算失败，每个场景检查"没有未捕获的页面错误"。原型的 `$P5TMP/probe/visual.mjs`（用 P4 的 `lib.mjs`）是一个可用的起点：在 `36680de` 的构建上 52 项全过，在基线上 16 项失败。

**每个场景都检查**：页面的文字节点和 `title`、`alt`、`aria-label`、`placeholder`、`href`、`content`、`src` 属性里没有 `/plane/i`（`data:` 地址除外）；页面标题是 "Nerve" 或 "… - Nerve"；`<link rel="icon">`、`apple-touch-icon`、`manifest` 指向 `app/assets/brand/` 的文件和 `site.webmanifest.json`（只有一个清单）。

**A 组：目视（截图由控制者看，Task 3 之后、Task 6 之后）**

1. 登录页 `/`（未登录的桩）：左上角是横版标志，浅色主题下字标是深色；以 `localStorage.theme = "dark"` 打开（`addInitScript`）时字标是浅色；右上角 "New to Nerve? Sign up"；副标题 "Welcome back to Nerve."；表单下没有服务条款和隐私政策，页面底部没有 "Join 10,000+ teams…" 和客户 Logo。
2. 工作区页 `/probe-ws`（已登录、已引导的桩）：首页引导卡片写 "Most things start with a project in Nerve."、"Make Nerve yours."；顶部栏没有 "Star us on GitHub"。
3. 帮助菜单（顶部栏的问号按钮）：只有 Documentation、Keyboard shortcuts 和 "Version: v1.4.2"，没有 Forum；点 Documentation 打开 `https://github.com/open-nerve/NerveProject`（拦截 `window.open` 或等 `popup`），不带 `opener`。
4. 加载动画，两种：禁用脚本打开 `/`（预渲染的 `HydrateFallback`）；让 `GET /api/instances/` 一直不回答（页面级的 `route` 不调用 `fulfill`）时的客户端加载动画。两种都是居中的 Nerve 图标（`img[alt=Nerve]`，带 `animate-pulse`），不是 GIF；深色主题下是同一张图。
5. 错误页：已登录的首页，让 `GET /api/workspaces/probe-ws/recent-visits/` 回答 `{}`（最近访问组件渲染时调用 `recents.filter`，抛出的错误进根 `ErrorBoundary`）：显示 "Looks like something went wrong!"、"Try refreshing the page. If the problem persists, contact your administrator." 和 "Go to home"，没有支持邮箱、状态页、X 账号；插图屏幕上没有 Plane Logo。
6. 维护页：`GET /api/instances/` 回答 500：标题 "Looks like Nerve didn't start up correctly!"，没有 "reach out to our support team" 和支持邮箱；插图同上。
7. 另外截图：`/create-workspace`（没有工作区的已登录用户，左上角是 Nerve 图标）、`/onboarding`（未引导，页头是横版标志，"This is how you will appear in Nerve."）、404 页、导览的第一屏（横版标志在彩色背景上是浅色字标）、工作区不存在页。

**B 组：行为（Task 6 之后）**

8. 命令面板的帮助命令：Open keyboard shortcuts、Open documentation（打开仓库）、Report a bug（打开 `…/NerveProject/issues`），没有 Join the forum。
9. 陈旧资源：已登录，中止 `**/assets/api-tokens-*.js` 的请求后打开 `/settings/profile/api-tokens`（令牌设置页是 `lazy()` 加载的分块）：页面刷新一次（`load` 事件两次），第二次失败时显示错误页；`sessionStorage` 里只有 `__nerve_chunk_reload`。
10. 编辑器的复制粘贴：在工作项描述的编辑器里选中带格式的内容，派发 `copy` 事件（自带 `DataTransfer`），它的类型里有 `text/nerve-editor-html`、没有 `text/plane-editor-html`；把这个 `DataTransfer` 以 `paste` 事件交给另一个编辑器，格式保留（沿用 P2 review 附录的编辑器脚本 2 的桩）。
11. 个人访问令牌和 Webhook 的设置页仍然可达：`/settings/profile/api-tokens` 列出令牌；新建令牌之后的提示是 "Copy and save this secret key somewhere safe. …"，不提 "Plane Pages"；删除确认框写 "…access to Nerve data…"；`/probe-ws/settings/webhooks` 列出 Webhook。
12. 收集箱列表和工作项详情：创建者邮箱是 `intake@plane.so` 或名字带 `-intake` 的工作项，按普通用户显示头像和名字（基线显示 Plane 的头像和名字）。
13. 重跑 P4 的 A、B、C 组（`$P4TMP/probe-c/run-c.sh`，地址不带结尾 `/`）：Task 1 改了一千多个文件的导入，这些场景全部照常通过。

**C 组：反向对照**

14. 在基线 `96d8c1d` 的构建上跑 A、B 两组：第 1–7 项的文字检查都应失败（页面上有 Plane），第 3 项多出 Forum，第 4 项找不到 `img[alt=Nerve]`，第 8 项有 Join the forum，第 9 项的键是 `__plane_chunk_reload`，第 10 项的类型是 `text/plane-editor-html`，第 12 项显示 Plane 的头像；第 11、13 项在基线上也通过（它们核对的是 P5 没有弄坏）。

---

## 完成后

- 整分支评审之前，控制者在本分支的 7 个提交上跑一次 `node $P5TMP/tools/pertask.mjs <提交>…`（在本仓库里也能用：它只读提交的树），与 spec 2.2 的中间值表对照；再跑 A、B 两组核对。
- review 写明交给 M4 的事项和待裁定的三项（spec 第 7 节），把 M1 设计 12 节 P5 一行改为"已完成"并加上 spec、plan、review 的链接。
