# M1/收尾 closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 M1 设计第 11 节成立，清掉 P1–P5 留给收尾的事项（spec 第 2 节的分诊）：`window.open` 都带 `noopener,noreferrer`，菜单项不再调用 `window.close()`；删掉从不渲染的应用栏、没人用的包导出、评审点名的死代码、基线的退化结构、没有引用的文案和图片；守卫只剩跨 M 的例外，每个分支都有命中样本；删掉 `sanitize-html`、给 `prosemirror-codemark` 打补丁、格式检查覆盖配置；写出前端改动清单和给 M2–M8 的交接。

**Architecture:** 13 个 Task，每个一个提交：守卫（T1）→ 安全（T2）→ 死代码（T3 应用栏、T4 包导出、T5 点名的死代码）→ 退化结构（T6）→ 编辑器与 HTML 工具（T7、T8）→ 依赖与格式检查（T9、T10）→ 死文案、死图片（T11、T12）→ 文档（T13）。每一处改动都由 `$COTMP` 下的一次性脚本做精确替换（原文不在就停下、什么都不写），脚本不进仓库；机械的批量删除（T4、T6、T11、T12）由脚本完成，输出逐项可核对。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、jsdom 30.1.1；Playwright 1.63.0（取自 `e2e/`，只用于 `devwarn.mjs`、联系表和控制者的探测）。

**Spec:** `docs/v0/M1-frontend-trim/specs/closeout.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

**原型：** `$COTMP/proto`，分支 `proto-closeout`，基点 `c493255`。每个 Task 的原型提交：
T1 `c9eeb28`、T2 `5eb9494`、T3 `7b802a8`、T4 `cccf8be`、T5 `30c798d`、T6 `17becf9`、T7 `344ab76`、T8 `ec6c3e5`、T9 `2d51151`、T10 `3cb038f`、T11 `b13b3d8`、T12 `3ebfe70`、T13 `328843f`。
本计划的每个数字都来自对应的原型提交；按本计划的脚本步骤在基点的干净克隆上重放，13 个 Task 得到的树都与原型提交相同（`$COTMP/replay.mjs`），重放的每个提交都通过五个门禁。

---

## Global Constraints

P5 计划的 Global Constraints 和"控制者评审补充"在本 Phase 继续有效（下面是它们在收尾的写法），P2–P5 评审第 4 节的裁定也照旧；与 P5 冲突时以本节为准。

### 命令与环境

- **所有命令在 worktree 的根目录执行**，不要 `cd`，也不要把 `cd` 和任何命令写在一起。要在某个包里执行时用 `pnpm --dir <目录>`；要在另一个克隆的根目录跑工具时用 `bash $COTMP/at.sh <目录> <命令…>`（`at.sh` 没有执行权限，要用 `bash`）。开始之前执行一次 `pnpm install --frozen-lockfile`；不做任何全局安装，不运行 `corepack enable`。
- **`$COTMP`** 是
  `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-co`。
  原型留下的脚本都在那里，不需要另装依赖；不要用裸的 `/tmp`。每个 Task 开始时核对一次 `ls $COTMP/deadsym.mjs $COTMP/lib/edit.mjs $COTMP/t4/round.mjs`；不在时从备份恢复：`mkdir -p $COTMP`，再 `tar -xf .superpowers/sdd/closeout/cotmp.tar -C $COTMP`。实现时写进 `$COTMP/logs/` 的文件都以 `impl-` 开头，不覆盖原型留下的日志（控制者要拿它们对照）。
- **基线副本** `$COTMP/base`（`c493255`，已安装、已构建）：`lintdiff.sh` 拿它比较新增的 oxlint 警告，也是构建体积的基线。没有时 `git clone -q . $COTMP/base`、`git -C $COTMP/base checkout -q c493255`（两次调用），再 `pnpm --dir $COTMP/base install --frozen-lockfile`、`bash $COTMP/at.sh $COTMP/base make build-web`。
- **原型的提交**：先取进本仓库（只需一次，不建分支、不改引用）：
  `git fetch .superpowers/sdd/closeout/proto.bundle proto-closeout`。
  之后 `git show <原型提交> -- <路径>` 看某个文件的结果，`git diff --cached --stat <原型提交>` 在提交前比较整棵树。
- **一个 Bash 调用只跑一个 git 命令**：不把多个 git 命令串在一起（`;`、`&&`、管道都不行），不在 git 命令里用花括号展开，不写调用 git 的 shell 循环；不把算出来的 shell 变量传给 node 或 sed；多步逻辑写成 `$COTMP` 下的脚本。不用 `git stash`、`git clean`、`git reset --hard`；不做浅拉取；不碰 `plane/`、`refer/` 和其他 Phase 的临时目录；不停、不改 Docker 容器。
- **`replay.mjs`、`replay-range.mjs`、`pertask.mjs`、`gates-range.mjs` 只对 `$COTMP/replay` 用**（它们在那个克隆里检出、提交），不要在 worktree 里运行。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写（T4、T8、T9）；`patches/prosemirror-codemark@0.4.2.patch` 只由 `pnpm patch-commit` 生成（T9）；en、zh-CN 的文案只由 `delkeys.mjs` 改（T4、T11）。
- **文件内容用编辑工具写**，不用带反引号的 heredoc。
- **开发服务器**：只有 T8 的 `devwarn.mjs` 会起（它自己停掉，报告端口）；实现者不跑 `make web-dev`。

### 沿用的裁定（P2–P5 评审，P5 计划的控制者评审补充）

1. **原型提交是对照答案，不能整体照搬。** 按 Task 的步骤一步步做，每一步都跑写明的命令，把实测输出和预期并排写进报告。
   - 可以用 `git show <原型提交> -- <路径>` 看一个文件的结果，可以运行 `$COTMP/t<N>/` 的脚本。条件是：跑完一个脚本都要读它的 diff，确认每一处改动都对应本 Task 的一条说明。
   - **禁止以提交、目录或文件列表为单位取原型的内容**：`git checkout <原型提交> -- <目录或多个文件>`、`git cherry-pick`、`git read-tree`、`git diff … | git apply` 都不许用。
   - 与原型的差异可以有，每一处都写明原因；原型里你认为错的地方，照你认为对的做，并写进报告。
2. **基点是本分支上一个 Task 的实际提交**（T1 是 `c493255`）；`<基点>` 出现在孤儿核对的每个脚本里。
3. **数字以实测为准。** lint 上限等于实测、只降不升（T4、T5 调低，其余不变）。`check:types` 23 个任务、`make lint-web` 52 个（T10 起 54 个）、`make test-web` 16 个、`make build-web` 11 个。键用严格方法（`keyref.mjs`）。
4. **删除，不隐藏；退化结构收掉；孤儿由造成它的 Task 删**（包括同一文件内的使用者、对象成员和 prop，`deadorph.mjs` 看住）；以删掉的东西命名的文案和图片随之删除；一个值只有一个来源。提交前在本 Task 新增的行里查退化结构：`git diff --cached -U0` 的输出存到 `$COTMP/logs/impl-t<N>-added.txt`，再 `grep -E '^\+' $COTMP/logs/impl-t<N>-added.txt`，其中不能有 `={"`、`${"`，也不能有不带插值的模板字面量（原型整个 Phase 都没有）。
5. **导入分组不重排**：换掉一个导入时，新导入放在原来那一组；T6 只删注释行。
6. **knip 是门禁**，每个 Task 结束时 `make knip` 为零。
7. **测试输出要干净**：`bash $COTMP/tests.sh` 在 T1–T8 只有 editor 包的 5 行 `prosemirror-codemark`（显示为 `(recorded prosemirror-codemark sourcemap lines: 5)`），T9 起连这一行也没有；任何时候都不能有 `!` 开头的行。
8. **守卫**：每条规则的每个顶层分支和 `(?:a|b)` 的每一支都有真实代码里的命中样本，`files` 选择器也一样（`node $COTMP/alts.mjs all --files` → `alternatives or variants without a hit sample: 0`）；不命中样本都引用保留的代码（`node $COTMP/missquote.mjs` → `miss samples that quote nothing kept: 0`），删掉或改名被引用代码的 Task 在同一提交里改样本；例外精确到原文、带 `count` 和 `until`，只减不增。
9. **图片由人看**：删除图片之前拼成联系表看过（T12）；改到的、保留的图片按能看清 20 px 细节的尺寸看（本 Phase 没有）。
10. **报告放在最后一条回复里**（子 agent 不能写报告文件）：提交哈希；每一步的命令、实测输出和预期；与原型的差异及原因；下面的风险点；没做到的事和疑问。状态写 `DONE`、`DONE_WITH_CONCERNS` 或 `BLOCKED`。
11. **执行方式沿用 P5**：实现者不指定模型（继承 Opus 5.5），各 Task 评审用 sonnet，整分支评审用 opus；T4、T6、T11、T12 是机械改动，控制者先在基点的干净克隆上重放脚本、确认提交就是脚本的输出，评审者判断每一类删除的意思；控制者在 T5、T8 之后和修复轮之后跑浏览器核对（最后一节），并在基线的构建上做反向对照。

### 每个 Task 的固定节奏

开始 T1 之前记下基线的死符号：`pnpm --filter web exec react-router typegen`，`node $COTMP/deadsym.mjs --tsv > $COTMP/logs/impl-t0-deadsym.tsv`（stderr：`prop 638, member 1150, export 426`）。

1. **改代码**：按 Task 的步骤运行脚本，读 diff。
2. **类型检查**：`pnpm exec turbo run check:types` → `Tasks:    23 successful, 23 total`。报到没动过的行上时，先 `rm -f web/apps/web/.turbo/tsconfig.tsbuildinfo web/packages/*/.turbo/tsconfig.tsbuildinfo` 再跑一次。
3. **孤儿核对**（`<基点>` 见裁定 2）：
   - `pnpm --filter web exec react-router typegen`，`node $COTMP/deadsym.mjs --tsv > $COTMP/logs/impl-t<N>-deadsym.tsv`（stderr 的计数见各 Task），`node $COTMP/deadorph.mjs $COTMP/logs/impl-t<N-1>-deadsym.tsv $COTMP/logs/impl-t<N>-deadsym.tsv` → `new dead members, props and exports: 0`（T7 例外，见 T7）；
   - `node $COTMP/infile-orphans.mjs <基点>`：列出的每一行都必须以 `defined now: 0)` 结尾；行数见各 Task，没写的为 `…: 0`；
   - `node $COTMP/symref.mjs orphaned <基点>` → `exports used elsewhere at <基点> and not now: 0`；
   - `node $COTMP/dangling.mjs <基点>` → `new dangling import comments: 0`；
   - `node $COTMP/keyref.mjs orphaned <基点>` 第一行 → `keys referenced at <基点> and not now: 0`；
   - `bash $COTMP/headers.sh <基点>` 没有输出。
4. **守卫**：`node tools/keywords.mjs`（预期见各 Task）；`node $COTMP/alts.mjs all --files` → 0；`node $COTMP/missquote.mjs` → 0。
5. **lint**：`bash $COTMP/lintdiff.sh $COTMP/base` 没有输出；`bash $COTMP/caps.sh | tail -1` → 各 Task 写的合计。
6. **门禁**：`bash $COTMP/gates.sh impl-t<N>` → 五行，`types` 23、`lint` 带 `keywords:` 一行和 52（T10 起 54）、`test` 16、`build` 11、`knip` exit 0；再 `bash $COTMP/tests.sh`（裁定 7）。
7. **与原型对比**：`git add -A`，`git diff --cached --stat <原型提交>` 为空或每处差异都有说明。
8. **提交**：一个 Task 一个提交，提交信息用英文（原型的在 `$COTMP/t<N>/msg.txt`，可以照用或改写），最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只能有本 Task 的改动；**出现 `?? .superpowers/` 就停下来报告**。

### 风险点（每个 Task 报告必答）

1. **保留的东西没有失去最后的读取方或入口**：固定节奏第 3 步的六项；删掉的每个成员、prop、键、图片、包入口，写出 `git grep` 找不到读取方的证据（批量删除的 T4、T11、T12 按类别写）。
2. **依赖只按本 Task 说的变**：`git diff --stat <基点> -- pnpm-lock.yaml pnpm-workspace.yaml patches package.json web/apps/web/package.json 'web/packages/*/package.json'` 只列本 Task 写明的文件。预期：T4（propel、editor 的 `exports`，propel 的 `cmdk`，hooks、propel、ui、utils 的上限，锁文件少 3 行：propel 的 `cmdk`）、T5（propel 的上限）、T8（utils 的依赖、锁文件、catalog）、T9（补丁、锁文件、`pnpm-workspace.yaml`）、T10（根目录和两个配置包的 `package.json`），其余 Task 为空。
3. **没有新的共享或模块级可变状态**：`grep -E '^\+(export )?(let|var|const|class) ' $COTMP/logs/impl-t<N>-added.txt` 列出的每一行写明它是组件、函数、只读数据，还是去掉 `export` 的原有声明（T4）；不能有 `let`、`var`。原型里唯一的模块级可变值是 T7 测试文件里等 `afterEach` 销毁的编辑器列表。
4. **`window.open`、`target="_blank"`**：T2 起守卫的 `window-open` 规则看住前者；`node $COTMP/t2/blank.mjs | tail -1` → `JSX elements with a "_blank" target: 18; without a noopener/noreferrer rel: 0`（每个 Task 都是 18）。本 Task 改到的文件里有新的外链或弹窗时逐个写明。
5. **版权声明按来源**：新增的源文件带 `Copyright (c) 2026-present OpenNerve` 和 `SPDX-License-Identifier: AGPL-3.0-only`（T7、T8 的测试文件）；改写的 Plane 文件保留 Plane 的声明（`headers.sh` 看住）。

### 一次性脚本

都在 `$COTMP`（备份 `.superpowers/sdd/closeout/cotmp.tar`），从仓库根目录运行。P5 的同名脚本照搬（`symref`、`infile-orphans`、`keyref`、`keycount`、`headers`、`kw`、`caps`、`lintdiff`、`gates`、`tests`、`assets`、`web-size`、`lock-diff`、`dangling`），`alts.mjs` 加了路径规则和 `--files`。

| 脚本 | 用途 |
|---|---|
| `gates.sh <tag> [gate…]`、`tests.sh` | 五个门禁，日志在 `$COTMP/logs/<tag>-<gate>.log`；vitest 数量和记录在案之外的 stderr |
| `caps.sh`、`setcaps.mjs`、`lintdiff.sh <基线副本>` | 各包的警告数和上限；把上限设为实测值（只降）；比基线多出来的警告 |
| `symref.mjs orphaned <rev>`、`infile-orphans.mjs <rev>`、`dangling.mjs <rev>`、`headers.sh <rev>` | 孤儿、同文件孤儿、悬空的导入分组注释、许可证头 |
| `deadsym.mjs [--tsv]`、`lib/program.mjs`、`deadorph.mjs <前> <后>`、`domains.mjs <tsv> [--rows <M>]`、`compound.mjs` | 类型检查器找出的死导出、死成员、死 prop（附录 A）；一个 Task 新造成的；按领域分；复合组件没人读的成员 |
| `keyref.mjs unused \| orphaned <rev>`、`keycount.mjs`、`keyfalse.mjs`、`delkeys.mjs <列表>…` | 文案键（严格方法）；只被无关字面量命中的键；从 en、zh-CN 一起删键和空对象 |
| `kw.mjs add-rules \| set-phase`、`alts.mjs all --files`、`missquote.mjs` | 改 `tools/keywords.json`；分支的命中样本；不命中样本引用的代码是否存在 |
| `labels-all.mjs [--fix]`、`rulecount.mjs <根> <规则> [--fix]`、`degen-all.mjs` | 导入分组注释；只用一条 oxlint 规则计数或修复；全仓库的退化结构 |
| `assets.mjs`、`t12/sheet.mjs`、`t12/rm.mjs` | 没有引用的图片（列表在 stdout）；联系表；删除 |
| `devwarn.mjs <仓库> <端口>`、`web-size.mjs`、`size.sh <仓库> <tag>`、`lock-diff.mjs`、`lockrm.mjs`、`stale-config.mjs`、`linttable.mjs` | 开发服务器的外部化警告；构建体积；锁文件；失效的工作区配置；oxlint 按包、按规则的表 |
| `lib/edit.mjs`、`lib/locale.mjs`、`at.sh` | 精确替换；按键改文案；在另一个克隆的根目录跑命令 |
| `t<N>/*.mjs`、`t<N>/*.json`、`t<N>/msg.txt` | 各 Task 的脚本、规则、样本和原型的提交信息 |
| `replay.mjs`、`replay-range.mjs`、`gates-range.mjs`、`pertask.mjs`、`stats.mjs` | 只对 `$COTMP/replay`：重放、门禁、中间值；提交的统计 |
| `provenance/` | 封面照片的来源核对（spec 第 9 节第 1 条） |

### 原型的教训（执行前必读）

1. **T3、T4、T6、T8 的脚本之后要跑 oxfmt**（步骤里写了范围）：删除和替换会留下 oxfmt 要重排的行，不跑的话 `check:format` 失败，树也对不上原型。
2. **`deadsym.mjs` 读 react-router 的路由类型**：每次跑之前先 `pnpm --filter web exec react-router typegen`，否则路由模块的类型是 `any`，结果不准。它只统计项目自己的代码：交给框架或第三方库读取的成员会被报成死的（T7 的 `parseHTML`）。
3. **`deadsym.mjs` 的计数写在 stderr，行写在 stdout**：`--tsv > 文件` 只存行。
4. **T4 要删到某一轮为零为止**：删一轮会让上一层的导出变成没人用；原型是 424、26、62、0。第 4 轮输出 `round 4: no dead exports` 才算完。
5. **孤儿会晚一步出现**：原型第一次做 T4 时漏了 propel `EmptyState` 的 `asset` 属性（只有被删的包装组件传它），重放之后的 `deadorph.mjs` 才发现；所以固定节奏第 3 步每个 Task 都跑它。
6. **`devwarn.mjs` 第一次跑可能看到 Vite 重新优化依赖**（`504 (Outdated Optimize Dep)`、页面多加载一次）：依赖变了之后的第一次如此，再跑一次就是稳定的结果。
7. **`t10/negative.mjs` 直接跑包的脚本，不经过 turbo**：turbo 的缓存会替 `check:lint` 作答。

### 文件结构

| 位置 | 改动 | Task |
|---|---|---|
| `tools/keywords.mjs`、`tools/keywords.json` | `M<n>/closeout`；`phase`；样本 | 1 |
| 11 个文件（spec 3.7）；`tools/keywords.json` | `window.open` 的第三个参数；规则 `window-open` | 2 |
| `core/lib/app-rail/`、`navigation/`、`use-workspace-paths.ts`、`use-navigation-preferences.ts`、`content-wrapper.tsx`、工作区布局、`sidebar-item.tsx`、`@nerve/types`、propel 的 `context-menu`；`tools/keywords.json` | 应用栏 | 3 |
| `web/packages/*`（158 个文件改、166 个删）、en、zh-CN、`tools/keywords.json`、锁文件 | 包导出、包入口、`tlds.ts`、`asset`、22 个键、上限 | 4 |
| `issue-modal/form.tsx`、`use-project-issue-properties.ts`、`ui/empty-space.tsx`、propel 的 `menu`、`context-menu`，ui 的 `dropdowns`、`use-reload-confirmation.tsx`、`.oxlintrc.json` | 点名的死代码、`window.close()`、`no-restricted-globals` | 5 |
| `web/**`（216 个文件）、`.oxlintrc.json` | 退化结构、导入分组注释、`alt`、`TPlacement` | 6 |
| editor 的 `callout/` | `parseHTML`、测试 | 7 |
| utils 的 `string.ts`、`package.json`，editor 的 `callout/utils.ts`，`pnpm-workspace.yaml`、锁文件 | `sanitize-html` → `DOMParser`、测试 | 8 |
| `patches/`、`pnpm-workspace.yaml`、锁文件、editor 的 `vitest.config.ts` | 补丁 | 9 |
| 根目录 `package.json`、`knip.jsonc`、两个配置包的 `package.json` | 格式检查 | 10 |
| en、zh-CN（30 个文件）、`constants/src/themes.ts`、`theme-switch.tsx`、`tools/keywords.json` | 500 个键、主题标签 | 11 |
| `app/assets/`（133 张图片） | 删除 | 12 |
| `docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M2…M8/handoffs/` | 文档 | 13 |

---

## Tasks

### Task 1: 守卫认识收尾；每个分支都有命中样本，不命中样本都引用保留的代码

M1 设计 7.4、11 节；P1 评审第 6 节 (a)；P5 评审第 7 节。spec 2.5、3.6。原型提交 `c9eeb28`（2 个文件，+160 / −94）。

**Steps**

- [ ] **Step 1: 记下现状**

Run: `node tools/keywords.mjs`、`node $COTMP/alts.mjs all --files | tail -1`、`node $COTMP/missquote.mjs | tail -1`
Expected: `keywords: 48 rules, 5 exceptions, no hits.`；`alternatives or variants without a hit sample: 42`；`miss samples that quote nothing kept: 50`。

- [ ] **Step 2: 工具认识 `M<n>/closeout`**

Run: `node $COTMP/t1/edits.mjs` → `edited tools/keywords.mjs`。读 diff：`PHASE_RE` 多一支 `closeout`，`parsePhase` 把它排在该 M 的所有编号 Phase 之后（`Number.MAX_SAFE_INTEGER`），三条报错信息的示例加上 `M1/closeout`。

- [ ] **Step 3: 样本和 `phase`**

Run: `node $COTMP/t1/samples.mjs $COTMP/t1/samples.json` → `samples: 58 replaced, 41 added, 7 deleted`；`node $COTMP/kw.mjs set-phase M1/closeout`；`pnpm exec oxfmt tools`。
读 diff：新增的命中样本都是真实代码（大多是基线里、被后来的 Phase 删掉的行，可以用 `git log -S` 找到）；替换的不命中样本在当前的树里都能找到原文（`missquote.mjs`），与原样本处在规则的同一边界上。

- [ ] **Step 4: 核对**

Run: `node tools/keywords.mjs` → `keywords: 48 rules, 5 exceptions, no hits.`（5 条例外都跨 M，没有到期）；`alts.mjs all --files` → 0；`missquote.mjs` → 0。
Run: `node $COTMP/t1/expiry-probe.mjs`（改 `cycle.service.ts` 那条例外的 `until`，跑守卫，再逐字节恢复）
Expected:
```
until M1/P5: exit 1  keywords: 0 hits without an exception, 0 stale exceptions, 0 mismatched exceptions, 1 expired exception (tools/keywords.json).
until M1/closeout: exit 1  keywords: 0 hits without an exception, 0 stale exceptions, 0 mismatched exceptions, 1 expired exception (tools/keywords.json).
until M2: exit 0  keywords: 48 rules, 5 exceptions, no hits.
until M6: exit 0  keywords: 48 rules, 5 exceptions, no hits.
until M1/close: exit 2  keywords: exception {…"until":"M1/close"} needs a "reason" and an "until" such as "M3", "M1/P4" or "M1/closeout"
```
之后 `git status --short` 只有 `tools/keywords.mjs`、`tools/keywords.json`。

- [ ] **Step 5: 核对、门禁与提交**

固定节奏第 2–8 步：`deadsym` 与基线相同（`prop 638, member 1150, export 426`）；上限合计 711；`git diff --cached --stat c9eeb28` 为空。

---

### Task 2: 每个 `window.open` 都带 `noopener,noreferrer`；规则 `window-open`

P5 评审第 7 节（整分支评审 M6）。spec 2.1 A1、3.7、第 4 节第 2 条。原型提交 `5eb9494`（12 个文件，+42 / −12）。

**Steps**

- [ ] **Step 1: 记下现状**

Run: `git grep -n "window.open(" -- web` → 16 行；其中带 `noopener` 的 4 行是 `sibling-item.tsx`、`help-commands.ts` 两行、`help-section/root.tsx`。`node $COTMP/t2/blank.mjs | tail -1` → `JSX elements with a "_blank" target: 18; without a noopener/noreferrer rel: 0`。

- [ ] **Step 2: 加上特性参数**

Run: `node $COTMP/t2/edits.mjs` → 11 行 `edited …`（`cycles/quick-actions.tsx`、`quick-action-dropdowns/helper.tsx`、`modules/quick-actions.tsx`、`project/card.tsx`、`views/quick-actions.tsx`、`workspace/views/default-view-quick-action.tsx`、`workspace/views/quick-action.tsx`、`attachment-list-item.tsx`、editor 的 `toolbar/download.tsx`、`full-screen/modal.tsx`（两处）、`custom-link/helpers/clickHandler.ts`）。每处只在参数表末尾加 `"noopener,noreferrer"`。
Run: `git grep -n -E "(=|return|\() *window\.open\(" -- web` → 没有输出（没有调用方用返回值；带 `noopener` 时它是 `null`）。

- [ ] **Step 3: 规则**

Run: `node $COTMP/kw.mjs add-rules $COTMP/t2/rules.json`，`pnpm exec oxfmt tools`，`node tools/keywords.mjs`
Expected: `keywords: 49 rules, 5 exceptions, no hits.`（`window-open`：`files` `^web/.*\.[cm]?[jt]sx?$`，`content` `window\.open\((?![^\n]*noopener)`；命中样本是基线的 `clickHandler.ts:53` 和 `cycles/quick-actions.tsx:66`，不命中样本是帮助菜单和兄弟工作项的两行）。`alts`、`missquote` → 0。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步；上限合计 711；`git diff --cached --stat 5eb9494` 为空。

---

### Task 3: 从不渲染的应用栏；规则 `app-rail`、`app-rail-files`

P4 评审第 4 节裁定 9、第 7 节。spec 2.2 B1、3.8。原型提交 `7b802a8`（17 个文件：删 8、改 9，+73 / −453）。

**Steps**

- [ ] **Step 1: 改写使用处**

Run: `node $COTMP/t3/edits.mjs` → 7 行 `edited …`（工作区的 `layout.tsx`、`navigation/index.ts`、`workspace/content-wrapper.tsx`、`use-navigation-preferences.ts`、`@nerve/types` 的 `navigation-preferences.ts`、`top-navigation-root.tsx`、`sidebar/sidebar-item.tsx`）。读 diff：`AppRailVisibilityProvider` 的包裹、顶部栏读显示模式的 `px-2` 分支、`useAppRailPreferences` 和它的键、三个类型、`AppSidebarItem` 的 `label`、`showLabel` 和复合静态成员都删除，别的不动。

- [ ] **Step 2: 删文件，收掉只有它用的 propel 部分**

Run: `git rm -q -- web/apps/web/core/components/navigation/app-rail-hoc.tsx web/apps/web/core/components/navigation/app-rail-root.tsx web/apps/web/core/components/navigation/items-root.tsx web/apps/web/core/hooks/use-workspace-paths.ts web/apps/web/core/lib/app-rail/context.tsx web/apps/web/core/lib/app-rail/index.ts web/apps/web/core/lib/app-rail/provider.tsx web/apps/web/core/lib/app-rail/types.ts`（与 `$COTMP/t3/rm.txt` 相同）。
Run: `node $COTMP/t3/orphans.mjs` → `edited web/packages/propel/src/context-menu/context-menu.tsx`（`Separator`、`Trigger` 和 `Content` 的 `className`；弹层的 `cn()` 只剩常量，合成一个字符串）。

- [ ] **Step 3: 规则**

Run: `node $COTMP/kw.mjs add-rules $COTMP/t3/rules.json`，`pnpm exec oxfmt tools web/packages/propel/src/context-menu`，`node tools/keywords.mjs`
Expected: `keywords: 51 rules, 5 exceptions, no hits.`

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步：`deadsym` 的 stderr `prop 635, member 1146, export 426`；`infile-orphans` 12 行（`withDockItems`、`AppRailRoot`、`AppSidebarItemsRoot`、`useAppRailPreferences`、`useWorkspacePaths`、`AppRailVisibilityContext`、`useAppRailVisibility`、`AppRailVisibilityProvider`、`IAppRailVisibilityContext`、`TAppRailDisplayMode`、`TAppRailPreferences`、`DEFAULT_APP_RAIL_PREFERENCES`），都是 `defined now: 0)`；上限 711；`git diff --cached --stat 7b802a8` 为空。

---

### Task 4: 没人用的包导出（三轮）、`tlds.ts`、它们留下的 22 个键

P3 评审第 7 节、P3 spec 7.4、P4 评审第 7 节。spec 2.2 B2–B3、2.5 E1、3.8、第 4 节第 5–7 条。原型提交 `cccf8be`（324 个文件：删 166、改 158，+165 / −9186）。

**Steps**

- [ ] **Step 1: 三轮删除**

Run: `pnpm --filter web exec react-router typegen`，然后依次 `node $COTMP/t4/round.mjs 1`、`2`、`3`、`4`（每轮的日志在 `$COTMP/t4/r<N>-*`）
Expected:
```
round 1:  424 dead exports, 28 re-export specifiers
          unexport: delete 196, unexport 267, empty 76, skip 0
          89 files deleted
          deadlocals: 25 files, 63 deletions
round 2:  26 dead exports, 6 re-export specifiers
          unexport: delete 17, unexport 15, empty 9, skip 0
          10 files deleted
          deadlocals: 2 files, 2 deletions
round 3:  62 dead exports, 0 re-export specifiers
          unexport: delete 61, unexport 1, empty 60, skip 0
          65 files deleted
          deadlocals: no new no-unused-vars warnings
round 4:  0 dead exports, 0 re-export specifiers
          round 4: no dead exports
```
读 diff 的抽样（评审包会按类别看）：删掉的声明在 `web/` 里没有别的读取方；去掉 `export` 的声明在自己文件里还在用；删掉的文件连同桶文件的那一行和包的子路径。

- [ ] **Step 2: 包入口、`tlds.ts`、`asset` 属性**

Run: `node $COTMP/t4/subpaths.mjs` → `edited web/packages/propel/package.json`、`edited web/packages/propel/tsdown.config.ts`、`rm web/packages/propel/src/spinners/index.ts`、`edited web/packages/editor/package.json`、`edited web/packages/editor/tsdown.config.ts`。
Run: `node $COTMP/t4/tlds.mjs` → `rm web/packages/utils/src/tlds.ts`、`2 exceptions deleted`。
Run: `node $COTMP/t4/asset.mjs` → 3 行 `edited …`（propel `empty-state/` 的 `types.ts`、`compact-empty-state.tsx`、`detailed-empty-state.tsx`：`asset` 删除，`assetKey` 必填，选择二者的分支删除）。
Run: `pnpm install --offline --frozen-lockfile=false`（propel 不再依赖 `cmdk`；锁文件只少 propel 这一条依赖的 3 行）。

- [ ] **Step 3: 孤儿键**

Run: `node $COTMP/keyref.mjs orphaned <基点> > $COTMP/t4/keys-orphaned.txt`，`head -1 $COTMP/t4/keys-orphaned.txt` → `keys referenced at <基点> and not now: 22`（`common` 的 `no`、`docs`、`support`、`forum`、`bot`、`pin`、`common.upcoming`、`common.completed`、10 个 `user_roles.*`，`work-item` 的 4 个 `issue.priority.*`）。
Run: `node $COTMP/delkeys.mjs $COTMP/t4/keys-orphaned.txt` → 最后一行 `keys deleted: 22; empty objects deleted (both locales): 4`。

- [ ] **Step 4: 上限和格式**

Run: `bash $COTMP/caps.sh > $COTMP/logs/impl-t4-caps.txt`，`node $COTMP/setcaps.mjs < $COTMP/logs/impl-t4-caps.txt`
Expected: `web/packages/hooks: cap 4 -> 3`、`web/packages/propel: cap 21 -> 17`、`web/packages/ui: cap 28 -> 25`、`web/packages/utils: cap 25 -> 19`。
Run: `pnpm exec oxfmt tools web/packages`。

- [ ] **Step 5: 核对、门禁与提交**

固定节奏第 2–8 步：`node tools/keywords.mjs` → `keywords: 51 rules, 3 exceptions, no hits.`；`deadsym` 的 stderr `prop 468, member 895, export 2`（剩下的 2 个是 `api-client/test/client.typecheck.ts` 的两个导出，这个文件本身就是类型检查）；`deadorph` 0；`infile-orphans` 165 行，都是 `defined now: 0)`；`keycount.mjs` → `en: 1577 keys`；上限合计 697；`git diff --cached --stat cccf8be` 为空。

---

### Task 5: 评审点名的死代码；菜单项不再调用 `window.close()`；`no-restricted-globals`

P3、P4、P5 评审第 7 节。spec 2.1 A3、2.2 B4–B6、3.7、3.8、第 4 节第 3 条。原型提交 `30c798d`（11 个文件：删 1、改 10，+95 / −274）。

**Steps**

- [ ] **Step 1: 死代码和 `window.close()`**

Run: `node $COTMP/t5/edits.mjs`，`git rm -q -- web/apps/web/core/hooks/use-project-issue-properties.ts`。读 diff：
- 工作项表单（`issue-modal/form.tsx`）直接调用迭代 store 的 `fetchAllCycles`（原来经 `useProjectIssueProperties().fetchCycles`，同一个调用），挂载时把表单重置为初始值的那段删除；
- `ui/empty-space.tsx`：`Icon`、`description` 和两个只有一个子元素的片段；
- propel `ContextMenu.Submenu`、`SubmenuTrigger`，propel `Menu.SubMenu` 和只为它存在的子菜单上下文，ui `CustomMenu.Portal`、`SubMenuTrigger`、`SubMenuContent` 这三个静态成员（`compound.mjs` 列出，`Portal` 组件本身保留）；
- propel `MenuItem` 点击时的 `close()` 删除。

- [ ] **Step 2: `no-restricted-globals`**

Run: `node $COTMP/t5/globals-probe.mjs $COTMP/base | tail -3`（基线上加这条规则会报什么）
Expected: `use-reload-confirmation.tsx:39 Unexpected use of 'confirm'.`、`propel/src/menu/menu.tsx:59 Unexpected use of 'close'.`、`no-restricted-globals hits: 2`。
Run: `node $COTMP/t5/globals.mjs`，`pnpm exec oxfmt .oxlintrc.json`。读 diff：`.oxlintrc.json` 加这条规则（错误级别，ESLint 的 confusing-browser-globals 列表），`use-reload-confirmation.tsx` 的 `confirm(` 改为 `window.confirm(`。

- [ ] **Step 3: 上限**

Run: `bash $COTMP/caps.sh > $COTMP/logs/impl-t5-caps.txt`，`node $COTMP/setcaps.mjs < $COTMP/logs/impl-t5-caps.txt` → `web/packages/propel: cap 17 -> 16`（菜单上下文里每次渲染新建的值随子菜单删掉了）。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步：`deadsym` 的 stderr `prop 452, member 893, export 2`；`infile-orphans` 4 行（`useProjectIssueProperties`、`TSubMenuProps`、`ICustomSubMenuTriggerProps`、`ICustomSubMenuContentProps`），都是 `defined now: 0)`；`node $COTMP/t5/globals-probe.mjs .` → `no-restricted-globals hits: 0`；上限合计 696；`git diff --cached --stat 30c798d` 为空。

---

### Task 6: 退化结构、错标的导入分组注释、装饰插图的 `alt`、`TPlacement`

P3、P4、P5 评审第 7 节。spec 2.2 B8–B10、2.7 G1、G3、G4、3.9、第 4 节第 8–10 条。原型提交 `17becf9`（216 个文件，+101 / −341）。

**Steps**

- [ ] **Step 1: `={"…"}` 这类写法**

Run: `node $COTMP/rulecount.mjs . react/jsx-curly-brace-presence | tail -1` → `react/jsx-curly-brace-presence: 80 hits in 58 files`。
Run: `node $COTMP/rulecount.mjs . react/jsx-curly-brace-presence --fix | tail -1` → `react/jsx-curly-brace-presence (after --fix): 0 hits in 0 files`。读 diff：只有 `={"…"}` → `="…"`、`{"…"}` → 文字、`` {`…`} `` → `"…"` 这几种替换。

- [ ] **Step 2: 规则、别名、模板、`alt`、`TPlacement`**

Run: `node $COTMP/t6/edits.mjs`。读 diff：
- `.oxlintrc.json` 把 `react/jsx-curly-brace-presence` 设为错误；
- 4 个别名（`lite-text/toolbar.tsx`、`issue-layouts/utils.tsx` 两处、打盹弹窗）删除，代码直接读导入的常量；`no-projects.tsx` 的模板字面量改为字符串；收集箱描述的 `` t(`${…}`) `` 改为 `t(…)`；
- `app/error/prod.tsx`、`instance/maintenance-view.tsx`、`auth-screens/not-authorized-view.tsx` 的 `alt="ProjectSettingImg"` 改为 `alt=""`；
- `list-view-types.d.ts` 的 `TPlacement` 改为 `CustomMenu` 的 `placement` 属性的类型。

- [ ] **Step 3: 导入分组注释**

Run: `node $COTMP/labels-all.mjs | tail -1` → `dangling 206, repeat 4`。
Run: `node $COTMP/labels-all.mjs --fix | tail -1` → `dangling 206, repeat 4; 158 files fixed`。
Run: `pnpm exec oxfmt web`。
读 diff：只有注释行被删，没有导入移动。`git add -A` 之后把 `git diff --cached -U0` 存到 `$COTMP/logs/impl-t6-added.txt`（固定节奏的裁定 4 本来就要存），`grep -E '^[-+]import' $COTMP/logs/impl-t6-added.txt` → 只有 `list-view-types.d.ts` 的一对（`TPlacement` 的导入换成 `import type { CustomMenu } from "@nerve/ui";`，在原来的位置）。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步：`labels-all.mjs | tail -1` → `dangling 0, repeat 0`；`deadsym` 与 T5 相同；上限合计 696（`caps.sh` 没有变化）；`node $COTMP/degen-all.mjs alias | tail -1` 的 `alias 0`；`git diff --cached --stat 17becf9` 为空。

---

### Task 7: 标注块的属性按字符串读取

P4 评审第 7 节。spec 2.6 F6、3.10。原型提交 `344ab76`（3 个文件：增 1、改 2，+72 / −2）。

**Steps**

- [ ] **Step 1: 测试先失败**

把 `$COTMP/t7/extension-config.test.ts` 复制为 `web/packages/editor/src/extensions/callout/extension-config.test.ts`（它带 Nerve 的版权声明），`pnpm --dir web/packages/editor exec vitest run src/extensions/callout` → `Tests  1 failed | 3 passed (4)`，失败的是 `keeps a numeric emoji code point as the string the HTML holds`：`expected 128161 to be '128161'`。

- [ ] **Step 2: 修复**

Run: `node $COTMP/t7/edits.mjs` → 两行 `edited …` 和 `added …/extension-config.test.ts`（同一个文件再复制一次）。读 diff：`extension-config.ts` 的每个属性加 `parseHTML: (element) => element.getAttribute(value)`，累加器的类型加上这个成员；`logo-selector.tsx` 的 `.toString()` 删除。
Run: 同上的 vitest → `4 passed`。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–8 步：`deadorph` 报 1 行 `member … callout/extension-config.ts:48 parseHTML` 和 `new dead members, props and exports: 1`——这是已知的例外（spec 3.2 结论 3：TipTap 读取它，Step 1 的测试证明），写进报告；`deadsym` 的 stderr `prop 452, member 894, export 2`；vitest 81 个（editor 20）；`git diff --cached --stat 344ab76` 为空。

---

### Task 8: 删掉 `sanitize-html`，HTML 工具改用 `DOMParser`

P4 评审第 7 节。spec 2.6 F2、3.10、第 4 节第 11 条。原型提交 `ec6c3e5`（6 个文件：增 1、改 5，+60 / −173）。

**Steps**

- [ ] **Step 1: 记下现状**

Run: `node $COTMP/devwarn.mjs $COTMP/base 3310 | head -11`
Expected（端口可换；每次加载页面 22 条，这次加载两次）：
```
console	22	path
console	12	source-map-js
console	6	url
console	4	fs
console externalized warnings: 44
server	22	path
server	12	source-map-js
server	6	url
server	4	fs
server externalized warnings: 44
console messages: 54 (port 3310, server stopped)
```

- [ ] **Step 2: 改写**

Run: `node $COTMP/t8/edits.mjs` → `added web/packages/utils/src/string.test.ts` 和 `edited …` 行。读 diff：
- `utils/src/string.ts`：`sanitizeHTML` 删除；`stripAndTruncateHTML` 取 `DOMParser` 解析出的 `body.textContent` 并 `trim()`；`isEmptyHtmlString` 在没有文字、也没有允许的标签时为真；
- editor 的 `callout/utils.ts` 读本地存储时不再过 `sanitizeHTML`；
- utils 的 `package.json` 删掉 `sanitize-html`、`@types/sanitize-html`，开发依赖加 `jsdom`（`catalog:`）；`pnpm-workspace.yaml` 的 catalog 删掉这两条；
- `string.test.ts`（Nerve 的版权声明，`// @vitest-environment jsdom`）5 个用例。
Run: `pnpm exec oxfmt web/packages/utils/src web/packages/editor/src/extensions/callout`，`pnpm install --offline --frozen-lockfile=false`（pnpm 在这一步或在前面的 `pnpm exec` 里同步依赖，打印 `Packages: +1 -18`：utils 链上 `jsdom`，删掉 17 个包和 utils 的一个链接）。

- [ ] **Step 3: 锁文件和开发服务器**

Run: `node $COTMP/lock-diff.mjs`（与本分支的 `HEAD` 比）
Expected:
```
catalogs: 6 lines removed, 0 added
importers: 2 dependencies removed, 1 added or changed
  + web/packages/utils devDependencies jsdom: specifier: 'catalog:'  version: 30.1.1
packages: 17 removed, 0 added or changed
snapshots: 17 removed, 0 added or changed
```
（`jsdom` 30.1.1 已在锁文件里，editor、web 在用。）
Run: `node $COTMP/devwarn.mjs . 3311 | head -3`（第一次可能有 `504 (Outdated Optimize Dep)`，见原型的教训 6，再跑一次）
Expected: `console externalized warnings: 0`、`server externalized warnings: 0`，其余控制台消息与基线同类（`[vite] connecting…`、i18next、React DevTools、没有后端时的 502）。报告写明端口，并确认服务器已停（输出里的 `server stopped`）。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步：vitest 86 个（utils 29）；`deadsym` 与 T7 相同；`git diff --cached --stat ec6c3e5` 为空。

---

### Task 9: `prosemirror-codemark` 的补丁

P3 评审第 4 节裁定 6、第 7 节。spec 2.6 F3、3.11。原型提交 `2d51151`（4 个文件：增 1、改 3，+126 / −4）。

**Steps**

- [ ] **Step 1: 补丁**

Run: `pnpm patch prosemirror-codemark@0.4.2 --edit-dir $COTMP/t9/cm-impl`（目录已存在时先 `rm -rf $COTMP/t9/cm-impl`）
Run: `node $COTMP/t9/strip.mjs $COTMP/t9/cm-impl` → `sourceMappingURL comments dropped: 12`
Run: `pnpm patch-commit $COTMP/t9/cm-impl` → 生成 `patches/prosemirror-codemark@0.4.2.patch`，`pnpm-workspace.yaml` 的 `patchedDependencies` 加一条，锁文件带上补丁的哈希 `5ac87877…`。

- [ ] **Step 2: 注释**

Run: `node $COTMP/t9/edits.mjs`。读 diff：`pnpm-workspace.yaml` 在这条补丁上方用注释写明原因（发布的 source map 指向没有发布的源文件；0.4.2 是最新版本），editor 的 `vitest.config.ts` 的注释写明它现在怎样加载这个包。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–8 步：`node $COTMP/lock-diff.mjs` → `patchedDependencies: 0 lines removed, 1 added`（`+ prosemirror-codemark@0.4.2: 5ac87877…`）、`importers` 里 editor 的这条依赖带上 `patch_hash`、`snapshots: 1 removed, 1 added or changed`，别的没有；`bash $COTMP/tests.sh` 里没有 `recorded prosemirror-codemark` 一行，也没有 `!` 行；`git diff --cached --stat 2d51151` 为空（补丁文件逐字节相同；不同时说明原因，例如 pnpm 版本）。

---

### Task 10: 格式检查覆盖两个配置包和根目录的配置

P1 评审第 6 节 (b)。spec 2.6 F4、3.11、第 4 节第 15 条。原型提交 `3cb038f`（4 个文件，+18 / −15）。

**Steps**

- [ ] **Step 1: 改脚本**

Run: `node $COTMP/t10/edits.mjs`，`pnpm exec oxfmt package.json knip.jsonc`。读 diff：两个配置包加 `check:format`、`fix:format`（`oxfmt --check .`、`oxfmt .`），`typescript-config` 删掉 `files`；根目录的 `check:format` 加 6 个文件；oxfmt 重排了根 `package.json` 的键（`engines`、`packageManager` 移到最后）、给 `knip.jsonc` 补了结尾逗号。

- [ ] **Step 2: 反向对照**

Run: `node $COTMP/t10/negative.mjs`
Expected:
```
root format: turbo.json: exit 1 with the change, exit 0 restored
root format: pnpm-workspace.yaml: exit 1 with the change, exit 0 restored
tailwind-config format: exit 1 with the change, exit 0 restored
typescript-config format: exit 1 with the change, exit 0 restored
```
之后 `git status --short` 只有 Step 1 的 4 个文件。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–8 步：`make lint-web` 54 个任务；`git diff --cached --stat 3cb038f` 为空。

---

### Task 11: 没有代码引用的文案；主题选项的标签只剩一个来源

M1 设计 6、9 节；P2、P3、P5 评审第 7 节。spec 2.3、3.12、第 4 节第 12–13 条。原型提交 `b13b3d8`（33 个文件，+54 / −1656）。

**Steps**

- [ ] **Step 1: 主题选项**

Run: `node $COTMP/t11/theme.mjs` → `edited web/packages/constants/src/themes.ts`、`edited web/apps/web/core/components/core/theme/theme-switch.tsx`。读 diff：`I_THEME_OPTION` 的 `key` 和 5 个 `key:` 删除，`i18n_label` 改为 `system_preference`、`light`、`dark`、`light_contrast`、`dark_contrast`（`common.json` 的键）；设置页的开关两处 `t(….key)` 改为 `t(….i18n_label)`；Power K 的 `themes-menu.tsx` 不用改。

- [ ] **Step 2: 键**

Run: `node $COTMP/keyref.mjs unused > $COTMP/t11/unused.txt`，`head -1 $COTMP/t11/unused.txt` → `keys not referenced now: 482`。
Run: `node $COTMP/keyfalse.mjs | tail -1` → `keys referenced only by literals none of which looks like a translation: 30`（T10 上是 34；Step 1 之后主题选项的 4 个键改由 `i18n_label` 引用，脚本认得这种写法）；其中 18 个的字面量到不了 `t()`，列在 `$COTMP/t11/traced.txt`（`warning`、`role`、`cover_image`、`security`、`notifications`、`invitations`、`assigned`、`subscribed`、`mentions`、`history`、`subscriber`、`Cancel`、`attachments`、`declined`、`ideal`、`prev`、`next`、`date`，spec 第 4 节第 12 条），其余 12 个到得了；逐个用 `git grep -n -w` 核对，写进报告。
Run: `node $COTMP/delkeys.mjs $COTMP/t11/unused.txt $COTMP/t11/traced.txt | tail -1` → `keys deleted: 500; empty objects deleted (both locales): 298`。

- [ ] **Step 3: 守卫样本**

Run: `node $COTMP/missquote.mjs | tail -2` → 列出 `i18n-applications` 的一个不命中样本（它引用的文案刚删掉）和 `miss samples that quote nothing kept: 1`。
Run: `node $COTMP/t1/samples.mjs $COTMP/t11/samples.json` → `samples: 1 replaced, 0 added, 0 deleted`（改为同一段文字仍在的 `settings.json` 那一行），`pnpm exec oxfmt tools`，`missquote.mjs` → 0。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步：`keycount.mjs` → `en: 1077 keys`；`keyref.mjs unused | head -1` → `keys not referenced now: 0`；`keyfalse.mjs | tail -1` → `…: 12`；`git diff --cached --stat b13b3d8` 为空。

---

### Task 12: 没有被导入的图片

P2、P3、P5 评审第 7 节。spec 2.4 D1、3.12、第 4 节第 14 条。原型提交 `3ebfe70`（133 个文件删除，−475 行）。

**Steps**

- [ ] **Step 1: 列出并看过**

Run: `node $COTMP/assets.mjs > $COTMP/t12/unreferenced.txt`（stderr `133 of 246 images unreferenced`），`wc -l < $COTMP/t12/unreferenced.txt` → `133`。
Run: `node $COTMP/t12/sheet.mjs . $COTMP/t12/unreferenced.txt $COTMP/t12/sheets-impl` → 6 张联系表；用读图工具逐张打开，报告写明每张表上是什么（spec 3.12 的类别），有没有 Nerve 自己的图（原型：没有）。

- [ ] **Step 2: 删除**

Run: `node $COTMP/t12/rm.mjs $COTMP/t12/unreferenced.txt` → `images deleted: 133 (6225 KiB); directories left empty and deleted: 11`。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–8 步：`node $COTMP/assets.mjs 2>&1 >/dev/null` → `0 of 113 images unreferenced`；构建通过就证明没有删掉被导入的图片；`git diff --cached --stat 3ebfe70` 为空。

---

### Task 13: 前端改动清单、README、交接

M1 设计 9、11 节。spec 2.8、3.13、第 8 节。原型提交 `328843f`（26 个文件：增 7、改 19，+407 / −2）。

**Steps**

- [ ] **Step 1: 文档**

Run: `node $COTMP/t13/docs.mjs`
Expected: `edited docs/v0/frontend-changes.md`、`edited README.md`，17 行 `close condition: docs/v0/M…/handoffs/M1-P…md`，7 行 `new handoff: docs/v0/M…/handoffs/M1-closeout.md`。

- [ ] **Step 2: 读 diff，核对数字**

- 前端改动清单 1.6 节的 13 行与本分支的实际提交一致（执行中与原型不同的地方，按实际改）；第二节两行"已完成 / M1/收尾"。
- 7 份新交接的死成员、死 prop 数目：`pnpm --filter web exec react-router typegen`，`node $COTMP/deadsym.mjs --tsv > $COTMP/logs/impl-t13-deadsym.tsv`，`node $COTMP/domains.mjs $COTMP/logs/impl-t13-deadsym.tsv` → `M2 member 59`、`M2 prop 7`、`M3 member 142`、`M3 prop 68`、`M4 member 435`、`M4 prop 123`、`M5 member 40`、`M5 prop 8`、`M6 member 46`、`M6 prop 38`、`M7 member 63`、`M7 prop 25`、`M8 member 6`、`shared member 103`、`shared prop 183`，与各交接写的相同。
- 17 份旧交接只在"来源"之前加了"关闭条件"一节。

- [ ] **Step 3: 整个 Phase 的核对、门禁与提交**

- 固定节奏第 2–8 步。
- 对 Phase 基线：`symref.mjs orphaned c493255` → `…: 0`；`keyref.mjs orphaned c493255` 第一行 → `…: 0`；`dangling.mjs c493255` → 0；`headers.sh c493255` 没有输出。
- 构建体积：`bash $COTMP/size.sh . impl-end`（先删 `web/apps/web/build` 再构建）→ `js: 396 files, 6587323 bytes`、`css: 3 files, 293801 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 109 files, 6435038 bytes`、`largest chunk: assets/use-parse-editor-content-*.js, 1378717 bytes`、`locale chunks: 34`。
- 锁文件：`node $COTMP/lockrm.mjs $COTMP/base/pnpm-lock.yaml pnpm-lock.yaml | tail -1` → `packages removed: 17`；`node $COTMP/stale-config.mjs | tail -1` → `…; stale: 1`（`postcss`，spec 3.5）。
- oxlint 的表：`node $COTMP/linttable.mjs` 与 spec 3.3 相同。
- `git diff --cached --stat 328843f` 为空；提交信息见 `$COTMP/t13/msg.txt`。

---

## 控制者的浏览器核对（M1 设计 7.5）

由控制者写脚本、跑、把全文和输出写进收尾 review 的附录；实现者不写。做法沿用 P4、P5 review 附录：在 `$COTMP/probe-app`（本仓库的克隆，检出到被测提交，`make build-web`）上用 node 起静态服务器提供 `web/apps/web/build/client`，Playwright 取自 `e2e/`，`/api/`、`/auth/` 的请求全由桩回答，场景没有列出的请求一律算失败，每个场景检查没有未捕获的页面错误。在 T5 之后、T8 之后和修复轮之后各跑一次。

**A 组：重跑前面的 Phase**（收尾在全应用里删代码）

1. P4 的 9 段（`$P4TMP/probe-c/run-c.sh`，含 P2、P3 探测的改编版；`$P4TMP` = `…/scratchpad/nerve-p4`）：全部通过（P5 结束时 403 项）。
2. P5 的 `brand.mjs`（`$P5TMP/probe/brand.mjs`，`$P5TMP` = `…/scratchpad/nerve-p5`）A、B、C 三组：全部通过（P5 结束时 177 项）。
3. 原样显示的键和加载失败的图片：在 1、2 走过的每个页面上，文字节点里没有形如 `^[a-z0-9_-]+(\.[a-z0-9_-]+)+$` 的原样的键，每个 `<img>` 的 `naturalWidth` 大于 0（T4、T11、T12 删了 22 + 500 个键和 133 张图）。

**B 组：收尾自己的改动**

4. `window.open`（T2）：`addInitScript` 包一层 `window.open`，记下每次的参数再调用原函数。编辑器里点成员写的链接（`https://example.com`，按编辑器打开链接的方式）、附件列表打开附件、图片工具栏的下载、全屏预览的下载和打开原图，以及项目卡片、迭代、模块、视图、工作区视图、工作项的"在新标签页打开"：每次的第三个参数含 `noopener` 和 `noreferrer`；新页面（`popup` 事件）里 `window.opener` 为 `null`。
5. `window.close()`（T5）：`addInitScript` 把 `window.close` 换成计数器。项目导航为标签页模式（`navigation_control_preference: "TABBED"`），窗口窄到标签页溢出，打开溢出菜单（`TabNavigationOverflowMenu`，propel `Menu`）选一项：跳到那个标签页，菜单关闭，计数为 0。
6. 通知预览（T8）：一条评论通知的内容是 `<p>Tom &amp; Jerry &lt;3</p>`：预览显示 `Tom & Jerry <3`。
7. 评论和草稿的"是否为空"（T8）：评论编辑器里空、只有空格或 `&nbsp;`、只有换行时不能提交（没有请求）；有文字、只有一张图片、只有一个提及时能提交；草稿弹窗同样。
8. 标注块（T7、T8）：描述的 HTML 里有 `data-emoji-unicode="128161"` 的标注块，图标显示为 💡；本地存储 `editor-calloutComponent-logo` 里 emoji 地址带 `&` 时，新插入的标注块用的地址里是 `&`，不是 `&amp;`。
9. 主题名（T11）：界面语言为 zh-CN 时，Power K 的主题菜单显示"系统偏好""浅色""深色""浅色高对比度""深色高对比度"，个人设置的主题开关显示同样的名字；en 时两处都是 "System Preference" 等。
10. 工作项表单的迭代（T5）：在有迭代的项目里打开新建工作项的弹窗，迭代下拉列出迭代（`GET …/cycles/` 被调用），与基线相同。
11. 空状态的插图（T4）：项目列表、迭代列表、收集箱的空状态显示插图（`assetKey` 必填之后每个调用方都有图）。

**C 组：反向对照**

12. 在基线 `c493255` 的构建上跑 B 组：第 4 项的参数没有 `noopener`（同源的新页面 `opener` 不为 `null`），第 5 项计数为 1，第 6 项显示 `Tom &amp; Jerry &lt;3`，第 8 项地址里是 `&amp;`，第 9 项 Power K 显示英文——这几项应失败；第 7、10、11 项和 A 组在基线上也通过（它们核对的是收尾没有弄坏）。

另外，`devwarn.mjs` 的对照已在 T8 Step 1、Step 3 里（基线 44 条，终态 0）。

---

## 完成后

- 整分支评审之前，控制者在本分支的 13 个提交上跑一次中间值（`$COTMP/pertask.mjs` 的检查，或在本仓库里按固定节奏第 3–5 步逐个提交跑），与 spec 3.2 的表对照；再跑 A、B 两组核对。
- 收尾 review 写明：3.3 的表和清零计划、3.4 的体积对比、3.5 的锁文件、spec 第 9 节各项的裁定；把 M1 设计 12 节收尾一行改为"已完成"并加上 spec、plan、review 的链接，勾上 11 节，改总体设计 9.4 和 M1 的状态（时机按 spec 第 9 节第 3 条的裁定）。

---

## 附录 A：死成员和死 prop 的脚本

M2–M8 的 `handoffs/M1-closeout.md` 引用这三个脚本：把它们存到任意目录（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下），在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node <目录>/deadsym.mjs --tsv > dead.tsv`、`node <目录>/domains.mjs dead.tsv --rows M<n>`。`lib/program.mjs` 按 `node_modules/.pnpm/typescript@5.8.3/` 找 TypeScript，升级 TypeScript 之后改这一行。

### `deadsym.mjs`

```js
// One-off (M1/closeout): members and props that nothing in web/ reads or passes, found with the type checker over
// one program of the whole web source (lib/program.mjs), not by name. What knip and tsc cannot see:
//   member  a class member (method, property, accessor) or an interface / type-literal member that no code reads
//           (a read is any reference that is not its own declaration, not an object-literal key written to it and
//           not a JSX attribute passed to it; a class member also counts as read when the same-named member of an
//           interface or type it implements is read);
//   prop    a member of a component's props type (the first parameter's type of a function that returns JSX, or of
//           a forwardRef / memo callback) that no JSX attribute, spread or object literal passes;
//   export  a top-level declaration exported from web/packages that no other file references.
// Each finding is also marked `str` when its name appears as a whole string literal somewhere in web/ (it may be read
// by a computed key: obj[key] with key from a list), and `test` when its only references are in test files.
// usage: node deadsym.mjs [--tsv]   (from the repository root, after `pnpm --filter web exec react-router typegen`)
import fs from "node:fs";
import path from "node:path";
import { makeProgram } from "./lib/program.mjs";

const { program, checker, ts, files, ROOT } = makeProgram();
const rel = (f) => path.relative(ROOT, f);
const isTest = (f) => /\.test\.tsx?$|\/test\/|vitest\./.test(f);

// declaration node -> { r: count of reads, w: count of writes, rTest: reads from test files }
const refs = new Map();
const note = (decl, kind, file) => {
  let e = refs.get(decl);
  if (!e) refs.set(decl, (e = { r: 0, w: 0, rTest: 0, readers: new Set() }));
  if (kind === "w") e.w++;
  else {
    e.r++;
    e.readers.add(file);
    if (isTest(file)) e.rTest++;
  }
};
const markSym = (sym, kind, file, selfDecl) => {
  if (!sym) return;
  const seen = new Set();
  const visit = (s) => {
    if (!s || seen.has(s)) return;
    seen.add(s);
    if (s.flags & ts.SymbolFlags.Alias) {
      try {
        visit(checker.getAliasedSymbol(s));
      } catch {}
    }
    for (const d of s.declarations ?? []) if (d !== selfDecl) note(d, kind, file);
    for (const r of checker.getRootSymbols(s)) if (r !== s) visit(r);
    // a property of a mapped / instantiated type points back at its target
    if (s.links?.target) visit(s.links.target);
    if (s.target) visit(s.target);
    if (s.syntheticOrigin) visit(s.syntheticOrigin);
  };
  visit(sym);
};
const propOf = (type, name) => {
  if (!type) return [];
  const out = [];
  const types = type.isUnion() || type.isIntersection() ? [type, ...type.types] : [type];
  for (const t of types) {
    const p = checker.getPropertyOfType(checker.getApparentType(t), name);
    if (p) out.push(p);
  }
  return out;
};

const declNameParents = new Set([
  ts.SyntaxKind.PropertySignature,
  ts.SyntaxKind.MethodSignature,
  ts.SyntaxKind.PropertyDeclaration,
  ts.SyntaxKind.MethodDeclaration,
  ts.SyntaxKind.GetAccessor,
  ts.SyntaxKind.SetAccessor,
  ts.SyntaxKind.FunctionDeclaration,
  ts.SyntaxKind.ClassDeclaration,
  ts.SyntaxKind.InterfaceDeclaration,
  ts.SyntaxKind.TypeAliasDeclaration,
  ts.SyntaxKind.EnumDeclaration,
  ts.SyntaxKind.EnumMember,
  ts.SyntaxKind.VariableDeclaration,
  ts.SyntaxKind.Parameter,
  ts.SyntaxKind.TypeParameter,
  ts.SyntaxKind.ModuleDeclaration,
]);

for (const sf of program.getSourceFiles()) {
  if (!files.has(sf.fileName)) continue;
  const file = sf.fileName;
  const visit = (node) => {
    if (ts.isIdentifier(node) || ts.isPrivateIdentifier(node)) {
      const parent = node.parent;
      const isDeclName = parent && parent.name === node && declNameParents.has(parent.kind);
      const isObjKey =
        parent &&
        parent.name === node &&
        (ts.isPropertyAssignment(parent) || ts.isMethodDeclaration(parent) || ts.isShorthandPropertyAssignment(parent)) &&
        ts.isObjectLiteralExpression(parent.parent);
      const isBindingKey = parent && ts.isBindingElement(parent) && (parent.propertyName === node || (!parent.propertyName && parent.name === node));
      const isJsxAttr = parent && ts.isJsxAttribute(parent) && parent.name === node;
      const isImportExport = parent && (ts.isImportSpecifier(parent) || ts.isExportSpecifier(parent) || ts.isImportClause(parent) || ts.isNamespaceImport(parent));
      if (isJsxAttr) {
        // the attribute's own symbol is transient; its target is the same-named member of the props type
        const ctype = checker.getContextualType(parent.parent);
        for (const q of propOf(ctype, node.text)) markSym(q, "w", file);
        markSym(checker.getSymbolAtLocation(node), "w", file);
      } else if (isBindingKey) {
        // const { a } = x / const { a: b } = x: a read of x's member a
        const pattern = parent.parent;
        if (ts.isObjectBindingPattern(pattern)) {
          const t = checker.getTypeAtLocation(pattern);
          for (const p of propOf(t, node.text)) markSym(p, "r", file);
        }
        if (!parent.propertyName && parent.name === node) {
          // the local it declares is a declaration, nothing to mark
        }
      } else if (isObjKey) {
        // the literal's own key; its contextual target is marked below (ObjectLiteralExpression)
        if (ts.isShorthandPropertyAssignment(parent)) markSym(checker.getShorthandAssignmentValueSymbol(parent), "r", file);
      } else if (isImportExport) {
        // an import or re-export is not a use by itself, except an export of a local from another file
        if (ts.isExportSpecifier(parent) && parent.parent?.parent?.moduleSpecifier === undefined) {
          const local = checker.getExportSpecifierLocalTargetSymbol(parent);
          // exporting an imported binding again (import { A } from "./a"; export { A }) is a re-export, not a use
          const imported = local?.declarations?.some(
            (d) => ts.isImportSpecifier(d) || ts.isImportClause(d) || ts.isNamespaceImport(d)
          );
          if (!imported) markSym(local, "r", file);
        }
      } else if (!isDeclName) {
        const sym = checker.getSymbolAtLocation(node);
        const isWrite =
          parent &&
          ts.isPropertyAccessExpression(parent) &&
          parent.name === node &&
          ts.isBinaryExpression(parent.parent) &&
          parent.parent.left === parent &&
          parent.parent.operatorToken.kind === ts.SyntaxKind.EqualsToken;
        markSym(sym, isWrite ? "w" : "r", file);
      }
    } else if (ts.isStringLiteral(node) && node.parent && (ts.isElementAccessExpression(node.parent) || ts.isLiteralTypeNode(node.parent))) {
      markSym(checker.getSymbolAtLocation(node), "r", file);
    } else if (ts.isObjectLiteralExpression(node)) {
      const ctype = checker.getContextualType(node);
      for (const prop of node.properties) {
        if (ts.isSpreadAssignment(prop)) {
          const st = checker.getTypeAtLocation(prop.expression);
          for (const p of st.getProperties()) for (const q of propOf(ctype, p.name)) markSym(q, "w", file);
          continue;
        }
        const name = prop.name && (ts.isIdentifier(prop.name) || ts.isStringLiteral(prop.name)) ? prop.name.text : undefined;
        if (!name) continue;
        for (const q of propOf(ctype, name)) markSym(q, "w", file);
      }
    } else if (ts.isJsxElement(node) && node.children.some((c) => !(ts.isJsxText(c) && c.containsOnlyTriviaWhiteSpaces))) {
      // children between the tags pass the `children` prop
      const ctype = checker.getContextualType(node.openingElement.attributes);
      for (const q of propOf(ctype, "children")) markSym(q, "w", sf.fileName);
    } else if (ts.isJsxSpreadAttribute(node)) {
      const attrs = node.parent;
      const ctype = checker.getContextualType(attrs);
      const st = checker.getTypeAtLocation(node.expression);
      for (const p of st.getProperties()) for (const q of propOf(ctype, p.name)) markSym(q, "w", file);
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);
}

// string literals anywhere in web source (for computed-key reads)
const strings = new Set();
for (const sf of program.getSourceFiles()) {
  if (!files.has(sf.fileName)) continue;
  for (const m of sf.text.matchAll(/(["'`])([A-Za-z_$][\w$]*)\1/g)) strings.add(m[2]);
}

const findings = [];
const add = (kind, decl, name, extra = "") => {
  const sf = decl.getSourceFile();
  const line = sf.getLineAndCharacterOfPosition(decl.getStart(sf)).line + 1;
  const e = refs.get(decl) ?? { r: 0, w: 0, rTest: 0 };
  const flags = [strings.has(name) ? "str" : "", e.r > 0 && e.r === e.rTest ? "test" : ""].filter(Boolean).join(",");
  findings.push({ kind, file: rel(sf.fileName), line, name, r: e.r, w: e.w, flags, extra });
};

const readOf = (d) => {
  const e = refs.get(d);
  return e ? e.r - e.rTest : 0;
};
const writeOf = (d) => refs.get(d)?.w ?? 0;

// members an implemented interface / extended type declares, by class
const implementedRead = (cls, name) => {
  for (const h of cls.heritageClauses ?? []) {
    for (const t of h.types) {
      const type = checker.getTypeAtLocation(t);
      for (const p of propOf(type, name)) for (const d of p.declarations ?? []) if (readOf(d) > 0) return true;
      // and up the chain
      const sym = type.getSymbol?.();
      for (const d of sym?.declarations ?? []) if (ts.isClassDeclaration(d) && d !== cls && implementedRead(d, name)) return true;
    }
  }
  return false;
};
// a class member counts as read when a subclass's same-named member is read through the base, or the reverse
const subclassesOf = new Map();

const isComponentLike = (fn) => {
  // a function whose body returns JSX (or null alongside JSX)
  let jsx = false;
  const scan = (n) => {
    if (jsx) return;
    if (ts.isJsxElement(n) || ts.isJsxSelfClosingElement(n) || ts.isJsxFragment(n)) jsx = true;
    else if (n !== fn && ts.isFunctionLike(n)) return;
    else ts.forEachChild(n, scan);
  };
  if (fn.body) scan(fn.body);
  return jsx;
};

const propsTypes = new Set(); // type-literal / interface declaration nodes used as props
for (const sf of program.getSourceFiles()) {
  if (!files.has(sf.fileName) || !sf.fileName.endsWith(".tsx")) continue;
  const visit = (n) => {
    if ((ts.isFunctionDeclaration(n) || ts.isArrowFunction(n) || ts.isFunctionExpression(n)) && n.parameters.length > 0 && isComponentLike(n)) {
      const p0 = n.parameters[0];
      const t = checker.getTypeAtLocation(p0);
      const collect = (type) => {
        if (type.isUnion() || type.isIntersection()) for (const s of type.types) collect(s);
        for (const p of type.getProperties()) for (const d of p.declarations ?? []) if (ts.isPropertySignature(d)) propsTypes.add(d);
      };
      collect(t);
    }
    ts.forEachChild(n, visit);
  };
  visit(sf);
}

for (const sf of program.getSourceFiles()) {
  if (!files.has(sf.fileName) || sf.isDeclarationFile) continue;
  if (sf.fileName.includes("/.react-router/")) continue;
  if (sf.fileName.endsWith("schema.gen.ts")) continue;
  const visit = (n) => {
    if (ts.isClassDeclaration(n)) {
      for (const m of n.members) {
        if (!m.name || !(ts.isIdentifier(m.name) || ts.isPrivateIdentifier(m.name))) continue;
        if (ts.isConstructorDeclaration(m)) continue;
        const name = m.name.text;
        if (readOf(m) > 0) continue;
        if (implementedRead(n, name)) continue;
        add("member", m, name, `class ${n.name?.text ?? "?"}; w ${writeOf(m)}`);
      }
    } else if (ts.isPropertySignature(n) || ts.isMethodSignature(n)) {
      if (n.name && (ts.isIdentifier(n.name) || ts.isStringLiteral(n.name))) {
        const name = n.name.text;
        if (propsTypes.has(n)) {
          if (writeOf(n) === 0) add("prop", n, name, `read ${readOf(n)}`);
        } else if (readOf(n) === 0) {
          const owner = n.parent && (ts.isInterfaceDeclaration(n.parent) ? n.parent.name.text : ts.isTypeAliasDeclaration(n.parent.parent) ? n.parent.parent.name.text : "(literal)");
          add("member", n, name, `${owner}; w ${writeOf(n)}`);
        }
      }
    }
    ts.forEachChild(n, visit);
  };
  visit(sf);
  // exports of packages
  if (rel(sf.fileName).startsWith("web/packages/")) {
    for (const st of sf.statements) {
      const mods = ts.canHaveModifiers(st) ? ts.getModifiers(st) : undefined;
      if (!mods?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword)) continue;
      const decls = ts.isVariableStatement(st) ? st.declarationList.declarations : [st];
      for (const d of decls) {
        if (!d.name || !ts.isIdentifier(d.name)) continue;
        const e = refs.get(d);
        const others = e ? [...e.readers].filter((f) => f !== sf.fileName) : [];
        if (others.length === 0) add("export", d, d.name.text, e && e.readers.has(sf.fileName) ? "used in its own file" : "unused");
      }
      // a local exported by `export { a, b as c }` (no module specifier); reported under the exported name. The
      // specifier itself counts as a read in the file, so these always say "used in its own file": unexport.mjs
      // drops the specifier and deadlocals.mjs deletes the local if nothing else in the file uses it
    }
    for (const st of sf.statements) {
      if (!ts.isExportDeclaration(st) || st.moduleSpecifier || !st.exportClause || !ts.isNamedExports(st.exportClause)) continue;
      for (const el of st.exportClause.elements) {
        const local = checker.getExportSpecifierLocalTargetSymbol(el);
        const d = local?.declarations?.find((x) => x.getSourceFile() === sf);
        // an imported binding exported again (import { A } from "./a"; export { A }): its reads resolve to A's own
        // declaration, which the modifier pass above or the file that declares it reports
        if (!d || ts.isImportSpecifier(d) || ts.isImportClause(d) || ts.isNamespaceImport(d)) continue;
        const e = refs.get(d);
        const others = e ? [...e.readers].filter((f) => f !== sf.fileName) : [];
        if (others.length === 0) add("export", d, el.name.text, "used in its own file");
      }
    }
  }
}

const tsv = process.argv.includes("--tsv");
const counts = {};
for (const f of findings) counts[f.kind] = (counts[f.kind] ?? 0) + 1;
for (const f of findings.sort((a, b) => a.kind.localeCompare(b.kind) || a.file.localeCompare(b.file) || a.line - b.line)) {
  if (tsv) console.log([f.kind, `${f.file}:${f.line}`, f.name, `r${f.r}`, `w${f.w}`, f.flags, f.extra].join("\t"));
}
console.error(Object.entries(counts).map(([k, v]) => `${k} ${v}`).join(", "));
```

### `lib/program.mjs`

```js
// One-off (M1/closeout): one TypeScript program over the whole web/ source — the app and every workspace package —
// with each `@nerve/<pkg>[/sub]` import resolved to the package's src/ (not its built dist/), each package's own
// `@/…` alias resolved inside that package, and the app's route types (`./+types/…`, from `react-router typegen`)
// through rootDirs. So a symbol a package exports and the app uses is one symbol, which the per-package tsc runs
// and knip (which read dist/) cannot give.
// usage: import { makeProgram } from "./lib/program.mjs"; const { program, checker, ts, files } = makeProgram();
//        (from the repository root; run `pnpm --filter web exec react-router typegen` first)
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const ROOT = process.cwd();
const tsPath = path.join(ROOT, "node_modules/.pnpm/typescript@5.8.3/node_modules/typescript/lib/typescript.js");
const ts = (await import(tsPath)).default;

const WEB = path.join(ROOT, "web/apps/web");
const PKGS = path.join(ROOT, "web/packages");

function pkgJson(dir) {
  return JSON.parse(fs.readFileSync(path.join(dir, "package.json"), "utf8"));
}

function tryFile(p) {
  const bases = [p, p.replace(/\.(?:m?js)$/, "")];
  for (const b of bases) {
    for (const ext of ["", ".ts", ".tsx", ".d.ts", "/index.ts", "/index.tsx"]) {
      const f = b + ext;
      if (fs.existsSync(f) && fs.statSync(f).isFile() && /\.(?:tsx?|json)$/.test(f)) return f;
    }
  }
  return undefined;
}

// @nerve/<pkg>[/sub] -> the source file behind the package's export
function resolveNerve(spec) {
  const [, name, ...rest] = spec.split("/");
  const dir = path.join(PKGS, name);
  if (!fs.existsSync(path.join(dir, "package.json"))) return undefined;
  const j = pkgJson(dir);
  const key = rest.length ? `./${rest.join("/")}` : ".";
  const target = j.exports?.[key] ?? (key === "." ? j.main : undefined);
  if (typeof target === "string") {
    if (target.endsWith(".json")) return path.join(dir, target);
    const src = target.replace(/^\.\/dist\//, "./src/");
    return tryFile(path.join(dir, src));
  }
  return tryFile(path.join(dir, "src", ...rest));
}

function ownerDir(file) {
  if (file.startsWith(`${WEB}/`)) return WEB;
  if (file.startsWith(`${PKGS}/`)) return path.join(PKGS, file.slice(PKGS.length + 1).split("/")[0]);
  return undefined;
}

function resolveAlias(spec, from) {
  const owner = ownerDir(from);
  const rest = spec.slice(2);
  if (owner === WEB) {
    for (const [prefix, dir] of [
      ["app/", "app/"],
      ["helpers/", "helpers/"],
      ["styles/", "styles/"],
    ]) {
      if (rest.startsWith(prefix)) return tryFile(path.join(WEB, dir, rest.slice(prefix.length)));
    }
    return tryFile(path.join(WEB, "core", rest));
  }
  if (owner) return tryFile(path.join(owner, "src", rest));
  return undefined;
}

export function listFiles() {
  const tracked = execFileSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard", "web"], {
    encoding: "utf8",
    maxBuffer: 1 << 28,
  })
    .split("\0")
    .filter((f) => /\.(?:tsx?|mts|cts)$/.test(f) && fs.existsSync(f))
    .map((f) => path.join(ROOT, f));
  const typegen = [];
  const walk = (d) => {
    if (!fs.existsSync(d)) return;
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) walk(p);
      else if (/\.d?\.?tsx?$/.test(e.name)) typegen.push(p);
    }
  };
  walk(path.join(WEB, ".react-router/types"));
  return [...tracked, ...typegen];
}

export function makeProgram() {
  const files = listFiles();
  const options = {
    jsx: ts.JsxEmit.ReactJSX,
    module: ts.ModuleKind.ESNext,
    moduleResolution: ts.ModuleResolutionKind.Bundler,
    target: ts.ScriptTarget.ES2022,
    lib: ["lib.es2023.d.ts", "lib.dom.d.ts", "lib.dom.iterable.d.ts"],
    strict: true,
    skipLibCheck: true,
    resolveJsonModule: true,
    allowJs: false,
    noEmit: true,
    types: [],
    rootDirs: [WEB, path.join(WEB, ".react-router/types")],
    esModuleInterop: true,
  };
  const host = ts.createCompilerHost(options, true);
  const viteClient = path.join(WEB, "node_modules/vite/client.d.ts");
  host.resolveModuleNameLiterals = (literals, containingFile) =>
    literals.map((lit) => {
      const spec = lit.text;
      let resolved;
      if (spec.startsWith("@nerve/")) resolved = resolveNerve(spec);
      else if (spec.startsWith("@/")) resolved = resolveAlias(spec, containingFile);
      else if (spec === "package.json") resolved = path.join(ownerDir(containingFile) ?? WEB, "package.json");
      if (resolved) {
        return {
          resolvedModule: {
            resolvedFileName: resolved,
            extension: resolved.endsWith(".tsx")
              ? ts.Extension.Tsx
              : resolved.endsWith(".d.ts")
                ? ts.Extension.Dts
                : resolved.endsWith(".json")
                  ? ts.Extension.Json
                  : ts.Extension.Ts,
            isExternalLibraryImport: false,
          },
        };
      }
      return ts.resolveModuleName(spec, containingFile, options, host);
    });
  const program = ts.createProgram({ rootNames: [...files, viteClient], options, host });
  return { program, checker: program.getTypeChecker(), ts, files: new Set(files), ROOT };
}
```

### `domains.mjs`

```js
// One-off (M1/closeout): sorts the member and prop findings of a deadsym.mjs --tsv report into the later M whose
// domain owns the file (v0 design 9.2), by the first path pattern that matches. Prints the count per M and kind, and
// with --rows <M> the rows of that M.
// usage: node domains.mjs <report.tsv> [--rows <M>]
import fs from "node:fs";

// order matters: the first match wins
export const DOMAINS = [
  ["M5", /\/(?:file|attachment|asset|cover)[\w-]*(?:\/|\.)|\/services\/src\/file\/|image-upload|editor\/src\/.*\/(?:image|attachment)/i],
  ["M2", /\/(?:auth|account|onboarding|instance|api-token|profile-settings|user-settings)[\w-]*(?:\/|\.)|\/user\.(?:store|service|ts)|\/store\/user\/|settings\/profile|\/profile\/|packages\/services\//i],
  ["M6", /\/(?:cycle|module)[\w-]*(?:\/|\.)/i],
  ["M7", /\/(?:notification|inbox|intake|view|favorite|recent|home|workspace-views|global-view)[\w-]*(?:\/|\.)/i],
  ["M8", /\/webhook[\w-]*(?:\/|\.)/i],
  ["M4", /\/(?:issue|work-item|draft|comment|reaction|relation|filter|rich-filter|layout|activity|search|editor|description|label-select|estimate)[\w-]*(?:\/|\.)|packages\/editor\/|shared-state\//i],
  ["M3", /\/(?:workspace|project|member|state|label|permission|invitation)[\w-]*(?:\/|\.)/i],
];
export const domainOf = (file) => DOMAINS.find(([, re]) => re.test(file))?.[0] ?? "shared";

if (import.meta.url === `file://${process.argv[1]}`) {
  const [file, flag, which] = process.argv.slice(2);
  const rows = fs
    .readFileSync(file, "utf8")
    .trim()
    .split("\n")
    .map((l) => l.split("\t"))
    .filter(([k]) => k === "member" || k === "prop")
    .map(([k, loc, name]) => ({ k, loc, name, m: domainOf(loc.split(":")[0]) }));
  if (flag === "--rows") {
    for (const r of rows.filter((r) => r.m === which)) console.log(`${r.k}\t${r.loc}\t${r.name}`);
  } else {
    const by = new Map();
    for (const r of rows) by.set(`${r.m}\t${r.k}`, (by.get(`${r.m}\t${r.k}`) ?? 0) + 1);
    for (const [k, v] of [...by].sort()) console.log(`${k}\t${v}`);
  }
}
```
