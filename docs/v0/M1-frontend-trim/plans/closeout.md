# M1/收尾 closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 M1 设计第 11 节成立，清掉 P1–P5 留给收尾的事项（spec 第 2 节的分诊）：`window.open` 都带 `noopener,noreferrer`，菜单项不再调用 `window.close()`；删掉从不渲染的应用栏、没人用的包导出、评审点名的死代码、基线的退化结构、没有引用的文案和图片；守卫只剩跨 M 的例外，每个分支都有命中样本；删掉 `sanitize-html`、给 `prosemirror-codemark` 打补丁、格式检查覆盖配置；写出前端改动清单和给 M2–M8 的交接；来源和许可查不到的预设封面换成 Nerve 自己画的图，保留的图片里不留真人的照片和名字，`handler_test.go` 的夹具用构建真实产出的文件名（spec 第 9 节第 1、2 条的裁定）；Codex 对抗评审的四项（spec 2.9）：粘贴编辑器自己的剪贴板类型时在惰性文档里解析，`make build-web` 从空的产物目录开始，活动迭代卡片用共享的进度口径，保留的截图里不再显示已删的功能。

**Architecture:** 20 个 Task，每个一个提交：守卫（T1）→ 安全（T2）→ 死代码（T3 应用栏、T4 包导出、T5 点名的死代码）→ 退化结构（T6）→ 编辑器与 HTML 工具（T7、T8）→ 依赖与格式检查（T9、T10）→ 死文案、死图片（T11、T12）→ 文档（T13）→ 图片与测试夹具（T14 预设封面、T15 头像和人名、T16 `handler_test.go`）→ 修复轮（任务评审和浏览器核对发现的已知问题，控制者派发，10 个提交，spec 3.16）→ Codex 对抗评审的四项（T17 粘贴、T18 构建目录、T19 活动迭代的进度、T20 截图）。每一处改动都由 `$COTMP` 下的一次性脚本做精确替换（原文不在就停下、什么都不写），脚本不进仓库；机械的批量删除（T4、T6、T11、T12）由脚本完成，输出逐项可核对。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、jsdom 30.1.1；Playwright 1.63.0（取自 `e2e/`，只用于 `devwarn.mjs`、联系表、T14 的渲染、T15 和 T20 的修图和控制者的探测）；Go 与 golangci-lint 2.13.2（T16 的 `make test`、`make lint-go`）。

**Spec:** `docs/v0/M1-frontend-trim/specs/closeout.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

**原型：** `$COTMP/proto`，分支 `proto-closeout`，基点 `c493255`。每个 Task 的原型提交：
T1 `c9eeb28`、T2 `5eb9494`、T3 `7b802a8`、T4 `cccf8be`、T5 `30c798d`、T6 `17becf9`、T7 `344ab76`、T8 `ec6c3e5`、T9 `2d51151`、T10 `3cb038f`、T11 `b13b3d8`、T12 `3ebfe70`、T13 `328843f`、T14 `131e1de`、T15 `f997e8a`、T16 `3c64669`。
本计划的每个数字都来自对应的原型提交。按本计划的脚本步骤重放：T1–T13 在基点的干净克隆上（`$COTMP/replay.mjs`），每个 Task 得到的树都与原型提交相同；T14–T16 在 `328843f` 的干净克隆上，三棵树与原型提交相同（T14、T15 的 WebP 逐字节相同），又在本分支 T13（`300eff0`）的克隆上重放，每个 Task 改到的图片、代码和 `server/` 路径与原型提交相同；重放的每个提交都通过五个门禁。T15 的 `t15/handoff.mjs`（M5 交接）是原型之后按裁定加的一步，不在原型提交 `f997e8a` 里，只在 `300eff0` 的重放上跑过。
T17–T20 的原型在分支 `proto-codex`，基点是本分支修复轮的最后一个提交 `9a98148`：T17 `bad3483`、T18 `b85f861`、T19 `a994925`、T20 `c9b24e1`。在 `9a98148` 的干净检出上按本计划的步骤重放，四棵树与原型提交相同（T20 的 WebP 逐字节相同），每个原型提交都通过五个门禁。

**本分支的实际提交：** T1 `aea47e4` 和跟进提交 `483ca9d`（5 个不命中样本，spec 3.6）、T2 `1cfc629`、T3 `4e1893c`、T4 `7f189da`、T5 `4cf483f`、T6 `20ac5b6`、T7 `acdbf50`、T8 `d25f08c`、T9 `98ecc86`、T10 `263aef9`、T11 `f9fb015`、T12 `4797772`、T13 `300eff0`、T14 `89244e3`、T15 `e349924`、T16 `985675a`；修复轮 `1896bca`–`9a98148`（10 个提交，spec 3.16）。本分支是准绳：它与原型不同的地方（T1 的样本、T4、T6、T9 的行数，T11 的样本等，spec 3.1、3.6、3.9）在 spec 里写明；所以 `300eff0` 的树与 `328843f` 不同，T14–T16 只比较本 Task 改到的路径（见各 Task 的最后一步）。T17–T20 也只比较本 Task 的路径：本分支在 `9a98148` 之后多了一个加入它们的文档提交（改本计划和 spec），原型没有。

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
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写（T4、T8、T9）；`patches/prosemirror-codemark@0.4.2.patch` 只由 `pnpm patch-commit` 生成（T9）；en、zh-CN 的文案只由脚本改（T4、T11 的 `delkeys.mjs`，T19 的 `t19/edits.mjs`）。
- **文件内容用编辑工具写**，不用带反引号的 heredoc。
- **开发服务器**：只有 T8 的 `devwarn.mjs` 会起（它自己停掉，报告端口）；实现者不跑 `make web-dev`。
- **T14–T16 的原型和脚本**：T14 开始之前（只需一次）`git fetch .superpowers/sdd/closeout/proto-2.bundle proto-closeout`（包含 T1–T16 的原型提交；`proto.bundle` 只到 T13）。每个 Task 开始时核对 `ls $COTMP/t14/apply.mjs $COTMP/t15/apply.mjs $COTMP/t16/edit.mjs`；不在时从 `cotmp-2.tar` 恢复：`mkdir -p $COTMP`，再 `tar -xf .superpowers/sdd/closeout/cotmp-2.tar -C $COTMP`（它包含 `cotmp.tar` 的全部内容和 `t14/`–`t16/`）。
- **T14、T15、T20 用 Chromium 出图**（取 `e2e/` 的 Playwright，不起服务器）：T14 把 SVG 渲染成 WebP，T15 画字母头像、写名字，T20 涂掉已删功能的控件。在同一台机器上重放，字节与原型相同；不同时写明 Chromium 的版本，并照各自的"看图"一步看过新的图。
- **T17–T20 的原型和脚本**：T17 开始之前（只需一次）`git fetch .superpowers/sdd/closeout/proto-codex.bundle proto-codex`（T17–T20 的原型提交；这个 bundle 以 `9a98148` 为前提，本分支有它）。每个 Task 开始时核对 `ls $COTMP/t17/edits.mjs $COTMP/t18/prove.sh $COTMP/t19/edits.mjs $COTMP/t20/patch.mjs`；不在时先照上一条恢复 `cotmp-2.tar`，再 `tar -xf .superpowers/sdd/closeout/cotmp-3.tar -C $COTMP`（它只有 `t17/`–`t20/`）。T17 的测试只在 editor 包里跑；T18 的 `prove.sh` 构建两次，每次几分钟。
- **T16 的 Go 检查**：`make test` 经 testcontainers 启动临时的 PostgreSQL 容器，测试进程退出后由 Ryuk 删除，不碰开发用的容器；跑完之后 `docker ps -a --filter label=org.testcontainers=true` 应为空。`make lint-go` 第一次会把锁定版本的 golangci-lint 2.13.2 下载到仓库的 `bin/`（`.gitignore` 已忽略，不是全局安装）。

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
8. **守卫**：每条规则的每个顶层分支和 `(?:a|b)` 的每一支都有真实代码里的命中样本（三个例外是示意的写法，因为守卫看住的写法在仓库里从来没有过，spec 3.6），`files` 选择器也一样（`node $COTMP/alts.mjs all --files` → `alternatives or variants without a hit sample: 0`）；不命中样本都引用保留的代码：`node $COTMP/missquote.mjs` → `miss samples that quote nothing kept: 0`，`node $COTMP/missreal.mjs` → `miss samples that do not quote kept code: 0`（`missquote` 在任何被跟踪的文件里找，样本自己在 `keywords.json` 里的那一行也算；`missreal` 只在规则读取的文件里找）。删掉或改名被引用代码的 Task 在同一提交里改样本；例外精确到原文、带 `count` 和 `until`，只减不增。
9. **图片由人看**：删除图片之前拼成联系表看过（T12）；新画的和改过的图片按能看清 20 px 细节的尺寸看（T14、T15、T20）；保留的其余图片由原型按同样的尺寸复看过，结果见 spec 3.15 的表。
10. **报告放在最后一条回复里**（子 agent 不能写报告文件）：提交哈希；每一步的命令、实测输出和预期；与原型的差异及原因；下面的风险点；没做到的事和疑问。状态写 `DONE`、`DONE_WITH_CONCERNS` 或 `BLOCKED`。
11. **执行方式沿用 P5**：实现者不指定模型（继承 Opus 5.5），各 Task 评审用 sonnet，整分支评审用 opus；T4、T6、T11、T12 是机械改动，控制者先在基点的干净克隆上重放脚本、确认提交就是脚本的输出，评审者判断每一类删除的意思；控制者在 T5、T8 之后、修复轮之后和 T20 之后跑浏览器核对（最后一节），并在基线的构建上做反向对照。

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
4. **守卫**：`node tools/keywords.mjs`（预期见各 Task）；`node $COTMP/alts.mjs all --files` → 0；`node $COTMP/missquote.mjs` → 0；`node $COTMP/missreal.mjs` → 0。
5. **lint**：`bash $COTMP/lintdiff.sh $COTMP/base` 没有输出；`bash $COTMP/caps.sh | tail -1` → 各 Task 写的合计。
6. **门禁**：`bash $COTMP/gates.sh impl-t<N>` → 五行，`types` 23、`lint` 带 `keywords:` 一行和 52（T10 起 54）、`test` 16、`build` 11、`knip` exit 0；再 `bash $COTMP/tests.sh`（裁定 7）。
7. **与原型对比**：`git add -A`，`git diff --cached --stat <原型提交>` 为空或每处差异都有说明。
8. **提交**：一个 Task 一个提交，提交信息用英文（原型的在 `$COTMP/t<N>/msg.txt`，可以照用或改写），最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只能有本 Task 的改动；**出现 `?? .superpowers/` 就停下来报告**。

T14–T16：第 3 步 `deadsym.mjs` 的 stderr 都是 `prop 452, member 894, export 2`，`infile-orphans.mjs` 没有行，其余各项都是 0。第 7 步只比较本 Task 改到的路径（本分支 T13 的树与原型 `328843f` 不同，见开头"本分支的实际提交"），命令写在各 Task 的最后一步。

T17–T20：T17 的基点是加入 T17–T20 的文档提交（`9a98148` 之后的那一个），其余是上一个 Task 的提交（裁定 2）。第 3 步 `deadsym.mjs` 的 stderr 都是 `prop 452, member 898, export 2`；T17 的 `deadorph.mjs` 以修复轮最后一个提交 `9a98148` 的 `$COTMP/logs/impl-fb3-deadsym.tsv` 为前一个（控制者留下的）；`infile-orphans.mjs` 没有行，其余各项都是 0。第 5 步上限合计 695。第 6 步 `tests.sh` 的 editor 一行 T17 起是 35 个测试（共 101 个）。第 7 步只比较本 Task 改到的路径。

### 风险点（每个 Task 报告必答）

1. **保留的东西没有失去最后的读取方或入口**：固定节奏第 3 步的六项；删掉的每个成员、prop、键、图片、包入口，写出 `git grep` 找不到读取方的证据（批量删除的 T4、T11、T12 按类别写）。
2. **依赖只按本 Task 说的变**：`git diff --stat <基点> -- pnpm-lock.yaml pnpm-workspace.yaml patches package.json web/apps/web/package.json 'web/packages/*/package.json'` 只列本 Task 写明的文件。预期：T4（propel、editor 的 `exports`，propel 的 `cmdk`，hooks、propel、ui、utils 的上限，锁文件少 3 行：propel 的 `cmdk`）、T5（propel 的上限）、T8（utils 的依赖、锁文件、catalog）、T9（补丁、锁文件、`pnpm-workspace.yaml`）、T10（根目录和两个配置包的 `package.json`），其余 Task 为空。
3. **没有新的共享或模块级可变状态**：`grep -E '^\+(export )?(let|var|const|class) ' $COTMP/logs/impl-t<N>-added.txt` 列出的每一行写明它是组件、函数、只读数据，还是去掉 `export` 的原有声明（T4）；不能有 `let`、`var`。原型里唯一的模块级可变值是 T7 测试文件里等 `afterEach` 销毁的编辑器列表。
4. **`window.open`、`target="_blank"`**：T2 起守卫的 `window-open` 规则看住前者；`node $COTMP/t2/blank.mjs | tail -1` → `JSX elements with a "_blank" target: 18; without a noopener/noreferrer rel: 0`（每个 Task 都是 18）。本 Task 改到的文件里有新的外链或弹窗时逐个写明。
5. **版权声明按来源**：新增的源文件带 `Copyright (c) 2026-present OpenNerve` 和 `SPDX-License-Identifier: AGPL-3.0-only`（T7、T8 的测试文件）；改写的 Plane 文件保留 Plane 的声明（`headers.sh` 看住）。

### 一次性脚本

都在 `$COTMP`（备份 `.superpowers/sdd/closeout/cotmp.tar`；T14–T16 的在 `cotmp-2.tar`，它也包含前者的全部内容；T17–T20 的在 `cotmp-3.tar`，只有这四个目录），从仓库根目录运行。P5 的同名脚本照搬（`symref`、`infile-orphans`、`keyref`、`keycount`、`headers`、`kw`、`caps`、`lintdiff`、`gates`、`tests`、`assets`、`web-size`、`lock-diff`、`dangling`），`alts.mjs` 加了路径规则和 `--files`。

| 脚本 | 用途 |
|---|---|
| `gates.sh <tag> [gate…]`、`tests.sh` | 五个门禁，日志在 `$COTMP/logs/<tag>-<gate>.log`；vitest 数量和记录在案之外的 stderr |
| `caps.sh`、`setcaps.mjs`、`lintdiff.sh <基线副本>` | 各包的警告数和上限；把上限设为实测值（只降）；比基线多出来的警告 |
| `symref.mjs orphaned <rev>`、`infile-orphans.mjs <rev>`、`dangling.mjs <rev>`、`headers.sh <rev>` | 孤儿、同文件孤儿、悬空的导入分组注释、许可证头 |
| `deadsym.mjs [--tsv]`、`lib/program.mjs`、`deadorph.mjs <前> <后>`、`domains.mjs <tsv> [--rows <M>]`、`compound.mjs` | 类型检查器找出的死导出、死成员、死 prop（附录 A）；一个 Task 新造成的；按领域分；复合组件没人读的成员 |
| `keyref.mjs unused \| orphaned <rev>`、`keycount.mjs`、`keyfalse.mjs`、`delkeys.mjs <列表>…` | 文案键（严格方法）；只被无关字面量命中的键；从 en、zh-CN 一起删键和空对象 |
| `kw.mjs add-rules \| set-phase`、`alts.mjs all --files`、`missquote.mjs`、`missreal.mjs` | 改 `tools/keywords.json`；分支的命中样本；不命中样本引用的代码是否存在（任何被跟踪的文件里；只在规则读取的文件里） |
| `labels-all.mjs [--fix]`、`rulecount.mjs <根> <规则> [--fix]`、`degen-all.mjs` | 导入分组注释；只用一条 oxlint 规则计数或修复；全仓库的退化结构 |
| `assets.mjs`、`t12/sheet.mjs <仓库的绝对路径> …`、`t12/rm.mjs` | 没有引用的图片（列表在 stdout）；联系表；删除 |
| `devwarn.mjs <仓库的绝对路径> <端口>`、`web-size.mjs`、`size.sh <仓库> <tag>`、`lock-diff.mjs`、`lockrm.mjs`、`stale-config.mjs`、`linttable.mjs` | 开发服务器的外部化警告；构建体积；锁文件；失效的工作区配置；oxlint 按包、按规则的表 |
| `lib/edit.mjs`、`lib/locale.mjs`、`at.sh` | 精确替换；按键改文案；在另一个克隆的根目录跑命令 |
| `t<N>/*.mjs`、`t<N>/*.json`、`t<N>/msg.txt` | 各 Task 的脚本、规则、样本和原型的提交信息 |
| `replay.mjs`、`replay-range.mjs`、`gates-range.mjs`、`pertask.mjs`、`stats.mjs` | 只对 `$COTMP/replay`：重放、门禁、中间值；提交的统计 |
| `provenance/` | 封面照片的来源核对（spec 第 9 节第 1 条） |
| `t14/covers.mjs <目录>`、`t14/render.mjs <仓库> <源目录> <目标目录> <格式> <宽> <质量>`、`t14/apply.mjs`、`t14/ledger.mjs`、`t14/sha.mjs <目录> <正则>`、`t14/sheet.mjs`、`t14/lum.mjs <仓库> <目录> <扩展名>`、`t14/measure.sh` | 画封面（带种子，可重复）；渲染；T14 的全部改动；清单的一行；大小和哈希；联系表和应用的裁切；白字的对比度；格式实测 |
| `t15/spec.json`、`t15/retouch.mjs <仓库> <spec> <输出> [--dry] [筛选]`、`t15/apply.mjs`、`t15/ledger.mjs`、`t15/handoff.mjs`、`t15/context.mjs <仓库> <spec> <输出> <倍数> <边距>`、`t15/assetpaths.mjs`、`t15/uses.mjs <仓库>`、`t15/inventory.mjs`、`t15/look.mjs`、`t15/tiles.mjs` | 每张图的头像组和名字框；字母头像和名字；T15 的全部改动；清单的一行；M5 交接的两项；改过的地方放大看；按路径没有被引用的图片；导入了、却传给没人读的值的图片；保留图片的清单和复看 |
| `t16/edit.mjs`、`t16/names.mjs` | 夹具改名；夹具名与构建产物对照 |
| `t17/test.mjs`、`t17/edits.mjs` | 先失败的编辑器测试；粘贴的 HTML 在惰性文档里解析 |
| `t18/edits.mjs`、`t18/prove.sh <仓库的绝对路径>` | `build-web` 先清空产物目录；在产物目录里放一个文件，冷、热构建各一次，看它还在不在，比较两次的文件清单 |
| `t19/edits.mjs` | 活动迭代卡片的进度和两个文案键 |
| `t20/spec.json`、`t20/patch.mjs <仓库> <spec> <输出> [--dry] [筛选]`、`t20/sources.mjs`、`t20/topng.mjs <仓库> <输出> [条高] [--crop x,y,w,h,倍数] <图>…` | 每张图要涂掉的框和取色点；涂掉、重新编码，写出每个框的前后对照；两个 `SOURCES.md` 的记录；看图用的 PNG（整张和分条） |
| `t20/investb-hardcoded.mjs`、`t20/investb-untranslated.mjs` | zh-CN 界面上的英文从哪来（spec 2.10 的 I2，只读） |
| `t14/probe/covers-probe.mjs`、`t15/probe/pictures-probe.mjs`、`t17/probe/copy-probe.mjs` | 控制者的浏览器核对第 12、13、14 项 |

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
| `app/assets/cover-images/`（删 29、增 59）、`helpers/cover-image.helper.ts`、`docs/v0/frontend-changes.md` | 预设封面 | 14 |
| `app/assets/onboarding/`、`app/assets/empty-state/`（改 11、增 2、删 8）、`cycles/active-cycle/root.tsx`、`docs/v0/frontend-changes.md`、`docs/v0/M5-files/handoffs/M1-closeout.md` | 头像、人名、没有显示的图；交给 M5 的两项 | 15 |
| `server/internal/platform/webui/handler_test.go` | 夹具名 | 16 |
| editor 的 `helpers/paste-asset.ts`、`helpers/asset-duplication.ts`、`editor-interaction.test.ts`、`vitest.setup.ts` | 粘贴的惰性解析、测试 | 17 |
| `Makefile` | `build-web` 先清空产物目录 | 18 |
| `cycles/active-cycle/progress.tsx`、en 和 zh-CN 的 `project.json` | 进度口径、两个键 | 19 |
| `app/assets/onboarding/`、`app/assets/empty-state/disabled-feature/`（改 9 张图、2 个 `SOURCES.md`） | 涂掉已删功能的控件 | 20 |

---

## 控制者评审补充（执行前必读）

控制者在执行前复核了 spec 和本计划：分诊表、原型 `c493255..328843f` 的数字、T2 和 T8 的改动（`window.open` 的调用方都不读返回值；propel `MenuItem` 的 `close()` 确是全局的 `window.close()`；`DOMParser` 解析出的文档是惰性的，标注块的本地存储值只经 DOM 接口写进属性）、浏览器核对。本节与上文冲突时以本节为准。

### spec 第 4 节和第 9 节的裁定

- 第 4 节第 1–18 条全部采纳。
- **第 9 节第 1 条（来源）：替换。** 来源和许可查不到的图片不随 Nerve 发布；功能保留。
  - **T14 封面**：29 张预设封面换成 Nerve 自己做的抽象图。仓库里要有它们可修改的形式（直接发布 SVG；或者 SVG 源文件和生成方式一起进仓库、登记在 `SOURCES.md`），由原型实测后选定。项目保存的封面值（预设封面怎样存、怎样读回）要先查清，文件名或格式变了就把写入方和读取方一起改，不留兼容层。旧的 29 张删除。
  - **T15 导览截图**：`onboarding/cycles.webp`、`modules.webp`、`views.webp` 里的真人头像换成字母头像，人名换成中性的名字（P5 修复轮的做法）。同时把保留下来的其余图片（T12 之后 113 张）全部按能看清 20 px 细节的尺寸看一遍，找真人照片、真实人名和 Plane 的内容，发现的一起处理或上报。
  - 新图和改过的图都按能看清 20 px 细节的尺寸看过，登记在同目录的 `SOURCES.md`（Nerve 的版权声明）。
- **第 9 节第 2 条（`handler_test.go`）：收尾做，T16。** 夹具名改为构建真实产出的文件名，注释因此成立。只改 Go 的测试，不改接口，不算跨模块的协同；`make test`、`make lint-go` 通过。
- **第 9 节第 3 条（M1 何时完成）：以收尾合并为终点。** Codex 对整个 M1 的对抗评审已经由负责人在 `c493255` 上启动，早于收尾合并。它的发现逐条分诊进收尾（追加的 Task 或修复轮），处理结果写进 Codex 评审报告的"处理结果"一节；M1 设计 12 节、11 节的勾、总体设计 9.4 和 M1 的状态在收尾 review 的提交里改，那时 Codex 的发现都已处理。
- 第 9 节第 4 条（死成员和死 prop 按领域交后续 M）、第 5 条（删除的图片按 300×200 看）：采纳。

### 对计划的修正

1. **Task 增加到 20 个，另有一个修复轮。** T14–T16 由架构师接着在 `$COTMP/proto`（`328843f` 之上）做原型，控制者在 T13 提交之后、T14 开始之前把它们的步骤追加进本计划（一个文档提交）。T16 之后，各 Task 的评审和控制者的浏览器核对发现的已知问题由控制者派发成修复轮（10 个提交，spec 3.16）。Codex 的发现在报告到达后分诊（spec 2.9）：四项成为 T17–T20，由架构师在 `$COTMP/proto` 的 `proto-codex` 分支（`9a98148` 之上）做原型，在修复轮之后、T17 开始之前由一个文档提交追加进本计划（就是加入本条的提交）；其余几项写进交接或设计文档，由控制者在收尾 review 的提交里改。
2. **并行时不碰别人的东西。** T1–T13 执行期间，架构师只在 `$COTMP/proto` 和 `$COTMP/t14/`–`t16/` 下工作，不改本计划点名的任何脚本；实现者不碰 `$COTMP/proto`。T17–T20 的原型同样只在 `$COTMP/proto` 和 `$COTMP/t17/`–`t20/` 下做。Codex 在主检出（`/Users/xiaoruan/project/nerve-project`）里评审：实现者和评审者都只在本 worktree 里工作，不读写主检出。
3. **机械的 Task（T4、T6、T11、T12）**：控制者核对提交的树与原型提交相同（`git diff --stat <原型提交> <实际提交>`，差异逐处有说明），评审者判断每一类删除的意思。其余 Task 按常规评审。
4. **`window-open` 规则只看同一行**：跨行的调用即使带了 `noopener` 也会被报出（宁可误报）。规则的 `why` 已写明"每次调用都在同一行传"；T2 的评审者核对 12 处都在一行里。

### T14–T16 原型之后的裁定

- T14–T16 的原型（`131e1de`、`f997e8a`、`3c64669`）采纳；控制者看过新封面和导览 `cycles.webp` 的头像。
- T15 改的是导览的 `cycles.webp`、`views.webp`、`issues.webp`：`modules.webp` 没有头像，也没有名字，不改（spec 第 4 节第 24 条）。
- 采纳原型的两处补充：T15 删掉活动迭代根组件传给内层组件、却没人读的属性，连同它按主题选出的两张图；T15 删掉 T12 按文件名查时漏掉的 6 张同名图（不改 T12）。
- **spec 第 9 节第 6 条（附件图标里的第三方标志）：交 M5。** 已裁定，不再待定：spec 8.1 的 M5 一行，关闭条件是 `app/assets/attachment/` 里没有第三方的标志；spec 第 6 节把它列为收尾范围之外。spec 8.1 另一条 M5 的事项（新建项目时的封面值）同样确认。两条由 T15 写进 `docs/v0/M5-files/handoffs/M1-closeout.md`（T15 Step 2 的 `t15/handoff.mjs`）。
- 本计划和 spec 的数字以本分支的实际提交为准，与原型不同的地方逐处写明（开头"本分支的实际提交"，spec 3.1、3.6、3.9）。

### Codex 对抗评审之后的裁定

- **负责人批准按建议分诊**（2026-09-25）Codex 对整个 M1 的对抗评审（`reviews/M1-codex-adversarial-review.md`，提交 `d2bd4fc`，评审对象 `c493255`）。逐条的去向见 spec 2.9：Critical 1 → T17；Important 1 → T18；Important 2 → T19；Important 4 → T20；Important 3（AGPL 第 5(a)、5(d)、13 条的验收，jsDelivr 的第三方请求）→ 控制者在收尾 review 的提交里写进 M8 收尾交接的关闭条件，时点是"任何对外的网络部署之前"，条款的解读由负责人在 M8 定；Minor 2 → 控制者在收尾 review 的提交里改 M1 设计 3.1 的写法；Minor 1 → 分诊时写的是"T4 已解决"，依据是 spec B12 原来的一格，核对之后不成立（`ensureAPITrailingSlash` 仍经 `services` 的两个 `export *` 桶文件公开），架构师建议放进整分支评审之后的修复轮，待控制者确认（spec 第 9 节第 7 条）。
- T17–T20 的原型（`bad3483`、`b85f861`、`a994925`、`c9b24e1`）采纳。
- 控制者第 3 轮浏览器核对顺带看到的两件事（spec 2.10），基线上相同：个人设置的主题下拉框画在左上角，交 M2（个人设置），控制者在收尾 review 的提交里写进 M2 的收尾交接；zh-CN 界面上的英文，交 M8（"每个 M 把改到的界面文字接入 `t()`；M8 发布前中文覆盖"），同样在收尾 review 的提交里写进，其中 zh-CN 与英文相同的 5 个值在整分支评审之后的修复轮里翻译。

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
各数的单位：`dead exports` 是没有别的文件读取的导出名（三轮共 424 + 26 + 62 = 512 个），`re-export specifiers` 是桶文件里按名字转出它们的说明符（34 个）；`delete`、`unexport` 是 `unexport.mjs` 的操作行，不是导出：`delete` 合计 274 行，每行删掉一个声明，`unexport` 合计 283 行，是 220 个 `export` 关键字和 63 个说明符（29 个在声明所在的文件里，34 个在桶文件里）；11 个 `export { … }` 里的名字先去掉说明符，文件里没有别处用它，声明随之删除，各占一行 `unexport` 和一行 `delete`。所以 512 = 274 + 283 − 11 − 34：274 个导出连同声明删除，238 个只去掉导出（220 个关键字、18 个说明符）。`empty` 是什么都不再导出、随后整个删除的文件（spec 3.8）。
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

**本分支的实际执行**（`20ac5b6`，以它为准）：脚本的写法在两个文件里留下了一行裸的 `//`，却把它上面真正的分组标签当作悬空删掉。本分支恢复了这两个标签，改删裸的 `//`，另删一行 `//hooks`。共删 212 行注释：悬空的标签 204 行、重复的 4 行、裸的 `//` 3 行、`//hooks` 1 行；`labels-all.mjs | tail -1` 仍是 `dangling 0, repeat 0`。T6 因此是 +101 / −343（原型 −341）。

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

Run: `node $COTMP/devwarn.mjs $COTMP/base 3310 | head -11`（`devwarn.mjs` 用 `createRequire` 从仓库里加载 Playwright，仓库路径要写绝对路径；`$COTMP/base` 已构建）
Expected（端口可换；页面加载两次，每次浏览器、服务端各 22 条，共 44 + 44）：
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
Run: `make build-web`（开发服务器加载各包的 `dist/`，本分支上要先构建），再 `node $COTMP/devwarn.mjs /Users/xiaoruan/project/nerve-project/.claude/worktrees/m1-closeout 3311 | head -3`（仓库路径写绝对路径，见 Step 1；第一次可能有 `504 (Outdated Optimize Dep)`，见原型的教训 6，再跑一次）
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
Run: `node $COTMP/t1/samples.mjs $COTMP/t11/samples.json` → `samples: 1 replaced, 0 added, 0 deleted`（原型改为同一段文字仍在的 `settings.json` 那一行），`pnpm exec oxfmt tools`，`missquote.mjs` → 0。
**本分支的实际执行**（`f9fb015`，以它为准）：`settings.json` 不是这条规则读取的文件，`missreal.mjs` 报 1；样本改为规则读取的 `workspace-settings.json` 第 103 行（`"description": "Any application using this token will no longer have the access to Nerve data. …"`），`missquote.mjs`、`missreal.mjs` 都是 0。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步：`keycount.mjs` → `en: 1077 keys`；`keyref.mjs unused | head -1` → `keys not referenced now: 0`；`keyfalse.mjs | tail -1` → `…: 12`；`git diff --cached --stat b13b3d8` 为空。

---

### Task 12: 没有被导入的图片

P2、P3、P5 评审第 7 节。spec 2.4 D1、3.12、第 4 节第 14 条。原型提交 `3ebfe70`（133 个文件删除，−475 行）。

**Steps**

- [ ] **Step 1: 列出并看过**

Run: `node $COTMP/assets.mjs > $COTMP/t12/unreferenced.txt`（stderr `133 of 246 images unreferenced`），`wc -l < $COTMP/t12/unreferenced.txt` → `133`。
Run: `node $COTMP/t12/sheet.mjs /Users/xiaoruan/project/nerve-project/.claude/worktrees/m1-closeout $COTMP/t12/unreferenced.txt $COTMP/t12/sheets-impl` → 6 张联系表（`sheet.mjs` 用 `createRequire` 从仓库里加载 Playwright，仓库路径要写绝对路径，写 `.` 会报错）；用读图工具逐张打开，报告写明每张表上是什么（spec 3.12 的类别），有没有 Nerve 自己的图（原型：没有）。

- [ ] **Step 2: 删除**

Run: `node $COTMP/t12/rm.mjs $COTMP/t12/unreferenced.txt` → `images deleted: 133 (6225 KiB); directories left empty and deleted: 11`。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–8 步：`node $COTMP/assets.mjs 2>&1 >/dev/null` → `0 of 113 images unreferenced`；构建通过就证明没有删掉被导入的图片；`git diff --cached --stat 3ebfe70` 为空。

---

### Task 13: 前端改动清单、README、交接

M1 设计 9、11 节。spec 2.8、3.13、第 8 节。原型提交 `328843f`（26 个文件：增 7、改 19，+413 / −2）。

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

### Task 14: 预设封面换成 Nerve 自己画的图

spec 第 9 节第 1 条的裁定；spec 3.15、第 4 节第 19–20 条。原型提交 `131e1de`（90 个文件：删 29、增 59、改 2，+4658 / −30）。

**Steps**

- [ ] **Step 1: 核对封面值怎样存、怎样读（只读）**

Run: `git grep -l -e "cover-image.helper" -e "cover-images/" -- web`
Expected:
```
web/apps/web/core/components/common/cover-image.tsx
web/apps/web/core/components/core/image-picker-popover.tsx
web/apps/web/core/components/project/form.tsx
web/apps/web/core/components/projects/create/root.tsx
web/apps/web/core/components/projects/create/utils.ts
web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
web/apps/web/helpers/cover-image.helper.ts
```
Run: `git grep -n -i -w -e cover -e cover_image -e cover_image_url -- server api e2e` → 没有输出。Go 服务、接口契约和 e2e 都不存、也不校验封面。

读 `helpers/cover-image.helper.ts`，把原型查到的事实逐条核对，写进报告：
- **预设只有一个来源**：29 个 `?url` 导入组成的 `STATIC_COVER_IMAGES`，构建后是带哈希的 `/assets/image_<n>-<hash>.<扩展名>`；`DEFAULT_COVER_IMAGE_URL` 是第 1 张（用户菜单的兜底）。
- **读取方**：`getCoverImageType`、`getCoverImageDisplayURL` 按这张表区分"构建里的预设"和"上传的资源"，另外 6 个文件经这些函数或这张表读写。
- **写入方**：新建项目（随机选一张）、封面选择器、项目设置和个人资料都经 `handleCoverImageChange`。值是预设时，`uploadCoverImage` 取回这张图，按文件名上传一份副本；项目或用户保存的是副本的地址。所以已保存的值不指向预设文件，没有要兼容的旧值；格式变了只改这张表和上传的兜底文件名。
- **格式受上传限制**：上传的元数据按文件头判断类型（`packages/services` 的 `fileTypeFromBuffer`），SVG 没有文件头；Plane 的资源接口只收 JPEG、PNG、WebP、GIF。所以应用导入栅格图，SVG 作为可修改的源文件进仓库。

- [ ] **Step 2: 画、渲染、改辅助函数，写清单**

Run: `node $COTMP/t14/apply.mjs`
Expected:
```
total	29 files	1379140
edited web/apps/web/helpers/cover-image.helper.ts

29 photos deleted; 29 SVGs, 29 WebP files and SOURCES.md written; helper edited
```
它做的事：
- 删掉 `app/assets/cover-images/image_1.jpg`–`image_29.jpg`；
- `t14/covers.mjs` 画 29 张 SVG：带种子，可重复；1600 × 900 的画布，尺寸 1920 × 1080；XML 注释里有 Nerve 的版权声明和一句说明；
- `t14/render.mjs` 在 Chromium 里把每张渲染成 1920 × 1080、质量 0.9 的 WebP；
- 写 `SOURCES.md`（Nerve 的版权声明、做法、每张一行）并用 oxfmt 排版；
- 辅助函数的 29 个导入从 `.jpg?url` 改为 `.webp?url`，上传的兜底文件名从 `image.jpg` 改为 `image.webp`。

读 diff：辅助函数只有这 30 处改动。

Run: `node $COTMP/t14/ledger.mjs` → `edited docs/v0/frontend-changes.md`（1.6 节末尾加一行）。
Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/cover-images '\.svg$' | tail -1` → `29 files	523700	list 2a7e480b68241354`
Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/cover-images '\.webp$' | tail -1` → `29 files	1379140	list 08069aa00767356e`
Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/cover-images '\.md$' | tail -1` → `1 files	3266	list 2148fe063deffb5b`

- [ ] **Step 3: 看图**

Run: `node $COTMP/t14/sheet.mjs . web/apps/web/app/assets/cover-images webp $COTMP/t14/look-impl final 960 1 3`
→ 10 行，从 `final-1.png: 3 covers` 到 `final-10.png: 2 covers`。每张封面缩到一半（原图的 20 px 在表上是 10 px）。

Run: `node $COTMP/t14/sheet.mjs . web/apps/web/app/assets/cover-images webp $COTMP/t14/look-impl bands 480 2 8 --bands`
→ `bands-1.png: 8 covers` … `bands-4.png: 5 covers`。每张封面加上应用的两种裁切：设置页和新建弹窗的 1000 × 176；项目卡片的 350 × 118，带卡片的暗色渐变和白色标题。

Run: `node $COTMP/t14/lum.mjs . web/apps/web/app/assets/cover-images webp | grep -c LOW` → `0`。白字在关闭按钮、"更换封面"和卡片标题三处的对比度都不低于 3；原型的最低值依次是 3.7、3.3、7.4。

用读图工具逐张打开这 14 张表，报告写明三点：没有照片、人、文字或产品标识；没有编码造成的色块或接缝；裁切后中部仍有图案。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步，另外核对：
- `node $COTMP/assets.mjs 2>&1 >/dev/null` → `0 of 142 images unreferenced`；
- 构建体积 `bash $COTMP/size.sh . impl-t14` → `js: 396 files, 6587353 bytes`、`css: 3 files, 293801 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 109 files, 4301559 bytes`、`largest chunk: assets/use-parse-editor-content-*.js, 1378717 bytes`、`locale chunks: 34`（"其他"比 T13 少 2,133,479 字节）；
- 与原型对比（第 7 步）：`git diff --cached --stat 131e1de -- web/apps/web/app/assets/cover-images web/apps/web/helpers/cover-image.helper.ts` 为空；`git diff --cached --numstat -- docs` → `1	0	docs/v0/frontend-changes.md`；`git diff --cached --shortstat` → `90 files changed, 4658 insertions(+), 30 deletions(-)`；
- 提交信息见 `$COTMP/t14/msg.txt`。

---

### Task 15: 保留的图片里没有真人；没有显示的图片删除

spec 第 9 节第 1 条的裁定；spec 2.4 D3–D4、3.15、第 4 节第 21–25 条、8.1。原型提交 `f997e8a`（23 个文件：删 8、增 2、改 13，+51 / −94）。本 Task 比原型多一步（Step 2 的 M5 交接，控制者在原型之后的裁定），所以是 24 个文件，+63 / −95。

**Steps**

- [ ] **Step 1: 找出没有显示的图片**

Run: `node $COTMP/t15/assetpaths.mjs`
Expected（列表在 stdout，计数在 stderr）:
```
web/apps/web/app/assets/empty-state/intake/intake-dark.webp
web/apps/web/app/assets/empty-state/intake/intake-light.webp
web/apps/web/app/assets/empty-state/label.svg
web/apps/web/app/assets/empty-state/project/name-filter.svg
web/apps/web/app/assets/empty-state/search/views-dark.webp
web/apps/web/app/assets/empty-state/search/views-light.webp
6 of 142 images unreferenced by path
```
`assets.mjs` 只看文件名。这 6 张与在用的图同名，或者是另一张图文件名的结尾，所以 T12 把它们当作在用。`intake/` 的两张与 `disabled-feature/intake-*.webp` 逐字节相同。

Run: `git grep -n -e "intake/intake-" -e "empty-state/label.svg" -e "project/name-filter" -e "search/views-" -- web` → 没有输出。

Run: `node $COTMP/t15/uses.mjs . | grep -A4 "cycle/active-dark"` → 显示第 110 行按主题选出图片，传给 `ActiveCyclesComponent` 的 `activeCycleResolvedPath`；第 49 行把这个属性解构成不读的 `_activeCycleResolvedPath`，空状态画的是 propel 的 `assetKey="cycle"`。所以这两张图从不显示。

- [ ] **Step 2: 头像、人名、删除、清单和 M5 交接**

Run: `node $COTMP/t15/apply.mjs`
Expected（每张图一行；缩进的测量行进日志）:
```
web/apps/web/app/assets/onboarding/cycles.webp: 11 avatars, 1 texts; WebP quality 0.72: 154204 -> 154282 bytes
web/apps/web/app/assets/onboarding/views.webp: 5 avatars, 0 texts; WebP quality 0.77: 134924 -> 135462 bytes
web/apps/web/app/assets/onboarding/issues.webp: 5 avatars, 0 texts; WebP quality 0.9: 228386 -> 229264 bytes
web/apps/web/app/assets/empty-state/disabled-feature/cycles-light.webp: 15 avatars, 0 texts; WebP quality 0.62: 67312 -> 67230 bytes
web/apps/web/app/assets/empty-state/disabled-feature/cycles-dark.webp: 15 avatars, 0 texts; WebP quality 0.58: 72322 -> 72372 bytes
web/apps/web/app/assets/empty-state/disabled-feature/intake-light.webp: 3 avatars, 0 texts; WebP quality 0.64: 87120 -> 86990 bytes
web/apps/web/app/assets/empty-state/disabled-feature/intake-dark.webp: 3 avatars, 0 texts; WebP quality 0.64: 83506 -> 83694 bytes
web/apps/web/app/assets/empty-state/disabled-feature/modules-light.webp: 9 avatars, 0 texts; WebP quality 0.66: 67536 -> 67542 bytes
web/apps/web/app/assets/empty-state/disabled-feature/modules-dark.webp: 9 avatars, 0 texts; WebP quality 0.65: 64282 -> 64154 bytes
web/apps/web/app/assets/empty-state/disabled-feature/views-light.webp: 1 avatars, 3 texts; WebP quality 0.76: 55462 -> 55798 bytes
web/apps/web/app/assets/empty-state/disabled-feature/views-dark.webp: 1 avatars, 3 texts; WebP quality 0.76: 55132 -> 55302 bytes
deleted 8 pictures nothing shows
edited web/apps/web/core/components/cycles/active-cycle/root.tsx
wrote web/apps/web/app/assets/onboarding/SOURCES.md
wrote web/apps/web/app/assets/empty-state/disabled-feature/SOURCES.md
```
它做的事：
- **头像**：`t15/spec.json` 给出每张图的头像组（位置、半径、哪几个圆是照片、用什么字母）和名字框。`retouch.mjs` 在 Chromium 里量出每个圆的范围，画一个平的圆：按字母取六种颜色之一，计数徽章用 `#1E3A8A`，上面是白色 Inter 500 的字母或数字。叠在它上面的圆按原位置剪开，中间的缝用背景色补回。
- **名字**：旧字用背景色盖住，新字用 Inter，字重、字号和基线按旧字拟合。
- **编码**：WebP 按"文件大小最接近原图"的质量重新编码。改前改后的放大图写到 `$COTMP/t15/crops/`。
- **删除**：Step 1 的 6 张和活动迭代的 2 张；变空的 `intake/`、`project/` 目录一并删除。
- **`root.tsx`**：只删不加（−11 行）：`useTheme` 的导入、两张图的导入和 `// assets` 标签、类型里的属性、`_activeCycleResolvedPath` 的解构、主题 hook、按主题选图的一行、两处 JSX 属性。
- **`SOURCES.md`**：写在 `onboarding/` 和 `empty-state/disabled-feature/`，内容依次是 Nerve 的声明、上游的声明、做法、每个文件改了什么；用 oxfmt 排版。

Run: `node $COTMP/t15/ledger.mjs` → `edited docs/v0/frontend-changes.md`（1.6 节末尾、T14 那一行之后）。
Run: `node $COTMP/t15/handoff.mjs` → `edited docs/v0/M5-files/handoffs/M1-closeout.md`。在"来源"一行之前加两节：附件图标里的第三方标志（关闭条件：`app/assets/attachment/` 里没有第三方的标志）；新建项目时的封面值（关闭条件：本 M 的封面接口只保存上传后的资源或预设的编号，不保存构建路径；新建项目只写一次封面值）。"来源"一行加上 spec 的 2.4 D4、3.15、第 9 节第 6 条（spec 8.1）。
Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/onboarding '\.(webp|md)$' | tail -1` → `5 files	684662	list e3036e9bc685461f`
Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/empty-state/disabled-feature '\.(webp|md)$' | tail -1` → `9 files	555579	list e5787ca2b2f6e98d`

逐个文件（字节，SHA-256 前 16 位）：
- `onboarding/`：`cycles.webp` 154282 `22759ed7dc4f3f64`、`issues.webp` 229264 `b950e9fe390b626a`、`views.webp` 135462 `9dce61fb608874f7`、`modules.webp` 163952 `8aa036aad5c3d3f3`（不变）、`SOURCES.md` 1702 `ec60b2ab85f4276d`；
- `disabled-feature/`：`cycles-light` 67230 `004f8bd8303beeb0`、`cycles-dark` 72372 `e7cd804ed51974b2`、`intake-light` 86990 `55b836932249ba46`、`intake-dark` 83694 `a52888a874b13c5f`、`modules-light` 67542 `5e10d65bb414a8c6`、`modules-dark` 64154 `bc0335e45eb9f041`、`views-light` 55798 `4716aabaa3f60b95`、`views-dark` 55302 `1acf41cc0299b8ac`、`SOURCES.md` 2497 `43f18341c7563b19`。

- [ ] **Step 3: 看改过的图**

Run: `node $COTMP/t15/context.mjs . $COTMP/t15/spec.json $COTMP/t15/look-impl 2 30`
→ 11 行：`onboarding_cycles_webp: 9 regions, 1 pieces`、`onboarding_views_webp: 4 …`、`onboarding_issues_webp: 3 …`、`empty-state_disabled-feature_cycles-light_webp: 5 …`、`…cycles-dark…: 5`、`…intake-light…: 1`、`…intake-dark…: 1`、`…modules-light…: 3`、`…modules-dark…: 3`、`…views-light…: 4`、`…views-dark…: 4`。

每张表是一张图的每组头像和每个名字，放大 2 倍，四周留 30 px。逐张打开，报告写明：
- 每个头像都是平的色块加白字（计数徽章是深蓝底）；
- 叠在上面的头像和边框完整，没有照片的残边。例外：`onboarding/views.webp` 视图面板边缘下有一条 5 px 宽的深色头像残边，原型保留，写在 `SOURCES.md` 里；
- 名字的字体、大小和颜色与周围的字一致，旧字没有残留。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步，另外核对：
- `node $COTMP/assets.mjs 2>&1 >/dev/null` → `0 of 134 images unreferenced`；
- `node $COTMP/t15/assetpaths.mjs 2>&1 >/dev/null` → `0 of 134 images unreferenced by path`；
- `git grep -n -e activeCycleResolvedPath -e "cycle/active-" -- web` 没有输出；
- 构建体积 `bash $COTMP/size.sh . impl-t15` → `js: 396 files, 6587153 bytes`、`css: 3 files, 293801 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 107 files, 4162901 bytes`、`largest chunk: assets/use-parse-editor-content-*.js, 1378717 bytes`、`locale chunks: 34`；
- 与原型对比（第 7 步）：`git diff --cached --stat f997e8a -- web/apps/web/app/assets web/apps/web/core/components/cycles/active-cycle/root.tsx` 为空；`git diff --cached --numstat -- docs` → `12	1	docs/v0/M5-files/handoffs/M1-closeout.md` 和 `1	0	docs/v0/frontend-changes.md`；`git diff --cached --shortstat` → `24 files changed, 63 insertions(+), 95 deletions(-)`；
- 提交信息见 `$COTMP/t15/msg.txt`。

---

### Task 16: `handler_test.go` 的夹具用构建真实产出的文件名

spec 第 9 节第 2 条的裁定；spec 2.7 G2、3.15、第 4 节第 26 条。原型提交 `3c64669`（1 个文件，+5 / −5）。只改 Go 测试的数据；`handler.go` 和接口都不改。

**Steps**

- [ ] **Step 1: 改名**

Run: `node $COTMP/t16/edit.mjs` → `edited server/internal/platform/webui/handler_test.go`

读 diff，只有三处：
- `built` 里的 `manifest.json` → `site.webmanifest.json`，`favicon/android-192.png` → `icons/icon-192x192.png`；
- 回退用例里"是目录不是文件"的 `/favicon` → `/icons`；
- 缓存用例的两行同样改名。

Run: `gofmt -l server/internal/platform/webui` → 没有输出。

- [ ] **Step 2: 与构建产物对照**

Run（在构建之后，固定节奏的门禁会构建）: `node $COTMP/t16/names.mjs`
Expected:
```
ok   fixture index.html: a file in the build
ok   fixture site.webmanifest.json: a file in the build
ok   fixture icons/icon-192x192.png: a file in the build
ok   fallback /icons: a directory in the build
ok   fallback /favicon.ico: no such file at the build's root
fixture names not in the build: 0
```
反向对照：`bash $COTMP/at.sh $COTMP/base node $COTMP/t16/names.mjs` → 3 行 `BAD`（`manifest.json`、`favicon/android-192.png`、`/favicon`）和 `fixture names not in the build: 3`，以 1 退出。

- [ ] **Step 3: Go 的检查**

Run: `make test` → 每个包都是 `ok` 或 `[no test files]`，其中有 `ok  	github.com/open-nerve/NerveProject/server/internal/platform/webui`。跑完核对 `docker ps -a --filter label=org.testcontainers=true` 为空（见"命令与环境"）。
Run: `make lint-go` → `0 issues.`

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步（web 没有改动，数字与 T15 相同）；与原型对比：`git diff --cached --stat 3c64669 -- server` 为空，`git diff --cached --shortstat` → `1 file changed, 5 insertions(+), 5 deletions(-)`；提交信息见 `$COTMP/t16/msg.txt`。

---

### Task 17: 粘贴编辑器自己的剪贴板类型时，在惰性文档里解析

spec 2.9 的 Critical 1、3.17、第 4 节第 27 条。原型提交 `bad3483`（4 个文件，+56 / −53）。基点是加入 T17–T20 的文档提交；T17–T20 的数字见"每个 Task 的固定节奏"最后一段。

**Steps**

- [ ] **Step 1: 先写会失败的测试**

Run: `node $COTMP/t17/test.mjs` → `edited web/packages/editor/vitest.setup.ts`、`edited web/packages/editor/src/editor-interaction.test.ts`

读 diff：
- `vitest.setup.ts` 声明 jsdom 没有的 `ClipboardEvent`：编辑器的粘贴处理调用 ProseMirror 的 `pasteHTML`，它会新建一个（不带剪贴板数据）；
- 测试文件加一个用例：往编辑器粘贴 `text/nerve-editor-html` 类型的 HTML，里面有一个带 `onerror` 的 `<img>` 和一张已上传的图（`src` 是资源的 id）。它监视 `Element.prototype.innerHTML` 的 setter，断言带 `onerror` 的标记从不写进活动文档里的元素；文字粘进来了；编辑器里没有 `[onerror]`；那张图被标为复制（`status` 是 `duplicating`，`id` 换成新的）。

Run: `pnpm --dir web/packages/editor exec vitest run src/editor-interaction.test.ts`
Expected: `Tests  1 failed | 7 passed (8)`，失败的是新用例，`AssertionError: expected [ …(2) ] to deeply equal []`：旧代码把这段 HTML 两次赋给活动文档里一个 `div` 的 `innerHTML`（解析一次，处理完再解析一次）。

- [ ] **Step 2: 在惰性文档里解析**

Run: `node $COTMP/t17/edits.mjs` → `edited web/packages/editor/src/helpers/paste-asset.ts`、`edited web/packages/editor/src/helpers/asset-duplication.ts`

读 diff：
- `processAssetDuplication` 用 `new DOMParser().parseFromString(htmlContent, "text/html")` 解析：这个文档没有浏览上下文，里面的图片不加载，事件属性不执行。每类节点的处理函数原地标记元素，最后序列化一次 `body.innerHTML`。原来"改一个元素、在字符串里替换它的旧标记、再解析一遍"的做法删除；
- `asset-duplication.ts` 的处理函数只接收元素，原地设 `status` 和新的 `id`；只有它用的 `AssetDuplicationContext`、`AssetDuplicationResult` 两个类型删除。粘贴处理（`props.ts`）不变：它把结果交给 `pasteHTML`，ProseMirror 在它自己新建的文档里按编辑器的 schema 再解析一次，只留 schema 声明的节点和属性。

Run: 同上的 vitest → `Tests  8 passed (8)`。

- [ ] **Step 3: 核对、门禁与提交**

固定节奏第 2–8 步，另外核对：
- `bash $COTMP/tests.sh` 的 editor 一行是 `Test Files 5 passed (5) | Tests 35 passed (35)`；
- 与原型对比（第 7 步）：`git diff --cached --stat bad3483 -- web/packages/editor` 为空；`git diff --cached --shortstat` → `4 files changed, 56 insertions(+), 53 deletions(-)`；
- 提交信息见 `$COTMP/t17/msg.txt`。

浏览器里的复制、粘贴和跨源载荷由控制者核对（最后一节第 14 项）。

---

### Task 18: `make build-web` 从空的产物目录开始

spec 2.9 的 Important 1、3.17、第 4 节第 28 条。原型提交 `b85f861`（`Makefile`，+2）。

**Steps**

- [ ] **Step 1: 反向对照（旧的 `Makefile`）**

Run: `bash $COTMP/t18/prove.sh <仓库的绝对路径>`（构建两次）
Expected:
```
== cold build (TURBO_FORCE=true: Turbo rebuilds, no cache restore)
 Tasks: 11 successful, 11 total
Cached: 0 cached, 11 total
  planted file: gone
cold: 531 files
== warm build (cache hit: Turbo restores the recorded outputs)
 Tasks: 11 successful, 11 total
Cached: 11 cached, 11 total
  planted file: PRESENT
warm: 532 files
== the two manifests
DIFFER:
443a444
> 3bb22bc0e99919e72cf42236d7e871ddf85fec0da19426d82e9a7eadcc5dda4f  ./assets/t18-stale-probe.txt
STALE FILE IN BUILD
```
脚本每次构建之前在 `web/apps/web/build/client/assets/` 放一个文件。冷构建时 Vite 自己清空输出目录；热构建由 Turbo 从缓存恢复，只写不删，文件留下，`make build` 会把它复制进 `server/internal/platform/webui/dist`、嵌进 Go 程序（Codex 的实验）。

- [ ] **Step 2: 改 `Makefile`**

Run: `node $COTMP/t18/edits.mjs` → `edited Makefile`

读 diff，只有两行：`WEBUI_DIST` 下面加 `WEB_CLIENT := web/apps/web/build/client`；`build-web` 在 Turbo 之前 `rm -rf $(WEB_CLIENT)`。

- [ ] **Step 3: 冷、热构建都从空目录开始**

Run: `bash $COTMP/t18/prove.sh <仓库的绝对路径>`
Expected: 两次都是 `planted file: gone`、`531 files`，最后两行：
```
cold and warm build/client are byte-identical (531 files)
no stale file in either build
```
持续集成（`ci.yml`）在干净的工作区里构建，Turbo 的缓存不跨任务保存，`rm -rf` 在那里什么都不删。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步（web 没有改动，数字与 T17 相同）；与原型对比：`git diff --cached --stat b85f861 -- Makefile` 为空，`git diff --cached --shortstat` → `1 file changed, 2 insertions(+)`；提交信息见 `$COTMP/t18/msg.txt`。

---

### Task 19: 活动迭代卡片的进度用共享的口径

spec 2.9 的 Important 2、3.17、第 4 节第 29 条；M1 设计 3.4。原型提交 `a994925`（3 个文件，+14 / −12）。

**Steps**

- [ ] **Step 1: 卡片和两个文案键**

Run: `node $COTMP/t19/edits.mjs` → `edited web/apps/web/core/components/cycles/active-cycle/progress.tsx`、`edited web/packages/i18n/src/locales/en/project.json`、`edited web/packages/i18n/src/locales/zh-CN/project.json`

读 diff：
- 进度条的值改为 `calculateCycleProgress(cycle ?? undefined)`（`@nerve/utils`，迭代列表也用它）：完成 / (总数 − 取消)。原来是 (完成 + 取消) / (总数 − 取消)，取消 2、完成 8、共 10 时是 125%，文字写 "10/8 closed"；
- 进度条上方的文字由 `t("project_cycles.active_cycle.work_items_completed", { completed, total })` 写出，`total` 是总数 − 取消；取消的说明由 `t("project_cycles.active_cycle.cancelled_excluded", { count })` 写出。原来两句都是模板字面量拼的英文；
- en 的两个值用 ICU 复数：`{completed}/{total} {total, plural, one {work item} other {work items}} completed`、`{count, plural, one {# cancelled work item is} other {# cancelled work items are}} excluded from this report.`；zh-CN：`{completed}/{total} 个工作项已完成`、`报告中已排除 {count} 个已取消的工作项。`

Run: `node $COTMP/keycount.mjs` → `en: 1079 keys`、`zh-CN: 1079 keys`
Run: `node $COTMP/keyref.mjs unused 2>&1 | tail -1` → `keys not referenced now: 0`

Codex 的两个例子已在共享函数的测试里（`web/packages/utils/src/progress.test.ts`：共 10、完成 3、取消 2 → 38；共 10、完成 8、取消 2 → 100），卡片改为经它计算，不另加测试（spec 第 4 节第 29 条）。

- [ ] **Step 2: 核对、门禁与提交**

固定节奏第 2–8 步；与原型对比：`git diff --cached --stat a994925 -- web/apps/web/core/components/cycles web/packages/i18n` 为空，`git diff --cached --shortstat` → `3 files changed, 14 insertions(+), 12 deletions(-)`；提交信息见 `$COTMP/t19/msg.txt`。

---

### Task 20: 保留的截图里不再显示已删的功能

spec 2.9 的 Important 4、3.17、第 4 节第 30 条。原型提交 `c9b24e1`（11 个文件：改 9 张 WebP 和 2 个 `SOURCES.md`，+20 行）。

**Steps**

- [ ] **Step 1: 先看前后对照**

Run: `node $COTMP/t20/patch.mjs . $COTMP/t20/spec.json $COTMP/t20/look-impl --dry`
Expected（每张图一行，缩进的是每个框）:
```
web/apps/web/app/assets/onboarding/issues.webp: 1 covers; WebP quality 0.95: 229264 -> 194670 bytes
  cover 1 [1345,141,138,52] solid rgb(255,255,255)
web/apps/web/app/assets/onboarding/cycles.webp: 2 covers; WebP quality 0.88: 154282 -> 153810 bytes
  cover 1 [610,226,146,57] solid rgb(255,255,255)
  cover 2 [230,233,38,44] solid rgb(232,232,232)
web/apps/web/app/assets/onboarding/views.webp: 1 covers; WebP quality 0.85: 135462 -> 134650 bytes
  cover 1 [750,213,36,34] solid rgb(232,232,232)
web/apps/web/app/assets/empty-state/disabled-feature/modules-light.webp: 1 covers; WebP quality 0.79: 67542 -> 67726 bytes
  cover 1 [2212,221,38,34] solid rgb(243,243,243)
web/apps/web/app/assets/empty-state/disabled-feature/cycles-light.webp: 1 covers; WebP quality 0.76: 67230 -> 67294 bytes
  cover 1 [2468,214,36,32] solid rgb(255,255,255)
web/apps/web/app/assets/empty-state/disabled-feature/modules-dark.webp: 1 covers; WebP quality 0.79: 64154 -> 64440 bytes
  cover 1 [2212,221,38,34] solid rgb(15,16,20)
web/apps/web/app/assets/empty-state/disabled-feature/cycles-dark.webp: 1 covers; WebP quality 0.73: 72372 -> 72190 bytes
  cover 1 [2460,210,30,28] solid rgb(17,17,19)
web/apps/web/app/assets/empty-state/disabled-feature/intake-light.webp: 1 covers; WebP quality 0.77: 86990 -> 87122 bytes
  cover 1 [1700,1090,650,44] solid rgb(255,255,255)
web/apps/web/app/assets/empty-state/disabled-feature/intake-dark.webp: 1 covers; WebP quality 0.78: 83694 -> 83140 bytes
  cover 1 [1828,956,536,50] solid rgb(17,17,19)
```
`t20/spec.json` 给出每个框（`[x, y, 宽, 高]`，原图像素）和取色的点（框外紧挨着的平的背景）。框住的是 M1 删掉的功能留下的控件：
- 导览 `issues.webp`：面包屑旁的 "Public" 可见性标签（项目发布）；
- 导览 `cycles.webp`：Add Issue 左边的数据分析按钮，工作项栏里的甘特图布局图标；导览 `views.webp`：同一个甘特图图标；
- "功能未开启"的 `modules-light`、`modules-dark`、`cycles-light`、`cycles-dark`：标题栏的甘特图/时间线布局图标；
- `intake-light`、`intake-dark`：工作项属性里 "Estimate: 2 Months" 一行（两张图的面板滚动位置不同，框分别量）。

`--dry` 不写图，只把每个框的前后对照（框四周加 8 px，放大 4 倍，不平滑）写成 `$COTMP/t20/look-impl/<图名>-cover<k>-before.png`、`-after.png`，20 张。逐张打开，报告写明：after 里控件不见了；四周的线、字和图标没有被盖住；填的颜色与背景一致，看不出方块。

- [ ] **Step 2: 写图和 `SOURCES.md`**

Run: `node $COTMP/t20/patch.mjs . $COTMP/t20/spec.json $COTMP/t20/look-impl` → 与 Step 1 相同的输出，这次写入 9 张图。WebP 按"文件大小最接近原图"的质量重新编码（T15 的做法）；`issues.webp` 少了 34,594 字节：去掉蓝色的字之后，最高的质量 0.95 也到不了原来的大小。其余 8 张相差不到 600 字节。
Run: `node $COTMP/t20/sources.mjs` → `edited web/apps/web/app/assets/onboarding/SOURCES.md`、`edited web/apps/web/app/assets/empty-state/disabled-feature/SOURCES.md`

每个 `SOURCES.md` 在最后一个表之后加一段做法说明和一个 "Removed" 表（按 oxfmt 的排版写，不用再格式化）。不写 "Analytics" 这个词：守卫的 `analytics` 规则读 `web/` 下的所有文件，`SOURCES.md` 也在内，所以写作 "the reporting button"。

Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/onboarding '\.(webp|md)$' | tail -1` → `5 files	649617	list 3baee9777c5f2795`
Run: `node $COTMP/t14/sha.mjs web/apps/web/app/assets/empty-state/disabled-feature '\.(webp|md)$' | tail -1` → `9 files	556337	list 0dad81521de949eb`

逐个文件（字节，SHA-256 前 16 位）：
- `onboarding/`：`cycles.webp` 153810 `b5fe3d4045f2c10d`、`issues.webp` 194670 `61eb8fed12e72872`、`views.webp` 134650 `500c5336e002728e`、`modules.webp` 163952 `8aa036aad5c3d3f3`（不变）、`SOURCES.md` 2535 `160d83e1a3d5f788`；
- `disabled-feature/`：`cycles-light` 67294 `976132e22b2fb46e`、`cycles-dark` 72190 `a017867c15d63d47`、`intake-light` 87122 `71eb35783ace0bad`、`intake-dark` 83140 `a3e769ed53eae5ae`、`modules-light` 67726 `3f7d752ebb406807`、`modules-dark` 64440 `61c4a07e29bc4de7`、`views-light` 55798 `4716aabaa3f60b95`（不变）、`views-dark` 55302 `1acf41cc0299b8ac`（不变）、`SOURCES.md` 3325 `4f99c5db8376b76b`。

- [ ] **Step 3: 看整张图**

Run: `node $COTMP/t20/topng.mjs . $COTMP/t20/look-impl/full 900 web/apps/web/app/assets/onboarding/cycles.webp web/apps/web/app/assets/onboarding/views.webp web/apps/web/app/assets/onboarding/issues.webp web/apps/web/app/assets/onboarding/modules.webp web/apps/web/app/assets/empty-state/disabled-feature/cycles-light.webp web/apps/web/app/assets/empty-state/disabled-feature/cycles-dark.webp web/apps/web/app/assets/empty-state/disabled-feature/modules-light.webp web/apps/web/app/assets/empty-state/disabled-feature/modules-dark.webp web/apps/web/app/assets/empty-state/disabled-feature/views-light.webp web/apps/web/app/assets/empty-state/disabled-feature/views-dark.webp web/apps/web/app/assets/empty-state/disabled-feature/intake-light.webp web/apps/web/app/assets/empty-state/disabled-feature/intake-dark.webp`
→ 12 行，导览的是 `1920x1080, 2 strips, 0 crops`，"功能未开启"的是 `2631x1200, 2 strips, 0 crops`（`modules-light` 是 `2632x1201`，原图如此）。

每张图拆成高 900 px 的两条，按原尺寸逐条打开（20 px 的细节看得清），报告写明：Step 1 的 10 个控件都不在了；没有别的 M1 删掉的功能（页面、估算、甘特图和时间线、数据分析、导出、便签、发布、史诗、多选等）的按钮、图标、属性或菜单。

- [ ] **Step 4: 核对、门禁与提交**

固定节奏第 2–8 步，另外核对：
- `node $COTMP/assets.mjs 2>&1 >/dev/null` → `0 of 134 images unreferenced`；`node $COTMP/t15/assetpaths.mjs 2>&1 >/dev/null` → `0 of 134 images unreferenced by path`；
- 构建体积 `bash $COTMP/size.sh . impl-t20` → `js: 396 files, 6586872 bytes`、`css: 3 files, 293801 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 107 files, 4126953 bytes`、`largest chunk: assets/use-parse-editor-content-*.js, 1378580 bytes`、`locale chunks: 34`（"其他"比 T19 少 35,948 字节；spec 3.4）；
- 与原型对比（第 7 步）：`git diff --cached --stat c9b24e1 -- web/apps/web/app/assets` 为空（WebP 逐字节相同）；`git diff --cached --shortstat` → `11 files changed, 20 insertions(+)`；
- 提交信息见 `$COTMP/t20/msg.txt`。

---

## 控制者的浏览器核对（M1 设计 7.5）

由控制者写脚本、跑、把全文和输出写进收尾 review 的附录；实现者不写。做法沿用 P4、P5 review 附录：在 `$COTMP/probe-app`（本仓库的克隆，检出到被测提交，`make build-web`）上用 node 起静态服务器提供 `web/apps/web/build/client`，Playwright 取自 `e2e/`，`/api/`、`/auth/` 的请求全由桩回答，场景没有列出的请求一律算失败，每个场景检查没有未捕获的页面错误。在 T5 之后、T8 之后、修复轮之后和 T20 之后各跑一次；第 12、13 项在 T15 之后加跑一次，此后随每次核对；第 14 项从 T20 之后的一次起随每次核对。

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
12. 预设封面（T14）：`OUT=<目录> bash $COTMP/at.sh $COTMP/probe-app node $COTMP/t14/probe/covers-probe.mjs` → `32 passed, 0 failed`。它核对：构建里有 29 张 WebP，没有封面 JPEG；项目卡片显示 8 张预设；新建项目弹窗的头部是 29 张之一，选择器按顺序列出 29 张，选第 17 张后头部换成它；提交时上传的副本类型是 `image/webp`、名字是预设的文件名，项目保存的是副本的地址；项目设置、个人资料、用户菜单（默认第 1 张）都显示预设；没有加载失败的图片。
13. 改过的图（T15）：`OUT=<目录> bash $COTMP/at.sh $COTMP/probe-app node $COTMP/t15/probe/pictures-probe.mjs` → `81 passed, 0 failed`。它核对：四个"功能未开启"页面（迭代、模块、视图、收集箱，浅色和深色）和导览的四步显示的图，字节与仓库里的文件相同，改过的 11 张与 `c493255` 的不同；项目只有已完成迭代、没有活动迭代时，活动迭代一节显示 "No active cycle" 和 propel 的插图，不请求活动迭代的图片。T20 又改了其中 9 张：脚本在运行时读仓库里的文件，比较的是页面拿到的字节与文件，所以预期不变，T20 之后仍是 `81 passed, 0 failed`（在原型 `c9b24e1` 的构建上实测）。
14. 粘贴编辑器自己的剪贴板类型（T17）：`OUT=<目录> bash $COTMP/at.sh $COTMP/probe-app node $COTMP/t17/probe/copy-probe.mjs` → `22 passed, 0 failed`。工作项 PRB-1 的描述里有文字和一张已上传的图（`src` 是资源的 id），三个场景：
    - C1：在描述里全选、用键盘复制，粘贴进评论编辑器：剪贴板带 `text/nerve-editor-html`，应用请求复制资源正好一次，评论里的文字在，图片指向新资源；
    - C2：同一段复制内容只以 `text/html` 粘贴（删掉自定义类型那一支后的样子，不改代码量出来）：不请求复制，图片仍指向描述的资源；
    - C3：模拟别的网页在复制事件里写下的自定义类型，内容是 `<p>pwned?</p><img src="x" onerror="window.__xss = true">`，粘贴进评论编辑器：`window.__xss` 仍为 `false`，文字粘进来了，编辑器里没有 `onerror`。

第 12、13、14 项的脚本引用 `$P5TMP/probe/lib.mjs` 和 `$P4TMP/probe-c/` 的数据，`pictures-probe.mjs` 从 `$COTMP/base` 读 `c493255` 的图片；截图写到 `$OUT`。

**C 组：反向对照**

15. 在基线 `c493255` 的构建上跑 B 组：第 4 项的参数没有 `noopener`（同源的新页面 `opener` 不为 `null`），第 5 项计数为 1，第 6 项显示 `Tom &amp; Jerry &lt;3`，第 8 项地址里是 `&amp;`，第 9 项 Power K 显示英文，第 12 项是 30 passed、2 failed（构建里是 JPEG，上传的副本是 `image/jpeg`），第 13 项是 70 passed、11 failed（正好是 11 张图的"differs from c493255's"，T20 之后也是这样），第 14 项是 21 passed、1 failed（C3 的 `window.__xss` 为 `true`：旧代码在活动文档里解析，`onerror` 执行了）——这几项应失败；第 7、10、11 项、第 14 项的 C1、C2 和 A 组在基线上也通过（它们核对的是收尾没有弄坏）。

另外，`devwarn.mjs` 的对照已在 T8 Step 1、Step 3 里（基线浏览器 44 条、服务端 44 行，终态 0）；`handler_test.go` 的夹具名的对照在 T16 Step 2 里（基线 3，终态 0）；构建目录的对照在 T18 Step 1、Step 3 里（旧的 `Makefile` 热构建留下事先放进去的文件，终态冷、热构建都删掉它）。

---

## 完成后

- T20 之后，控制者在一个文档提交里给前端改动清单 1.6 节加 T17–T20 各一行（与修复轮的 `4be6cba` 同一种做法；T17–T20 的原型没有改清单）。
- 整分支评审之前，控制者在本分支的 20 个 Task 和修复轮的提交上跑一次中间值（`$COTMP/pertask.mjs` 的检查，或在本仓库里按固定节奏第 3–5 步逐个提交跑），与 spec 3.2 的表对照；再跑 A、B 两组核对。
- 整分支评审之后的修复轮里已定的一项：zh-CN 与英文相同的 5 个值（`common` 的 `developer`、`your_profile`、`work_structure`、`execution`、`administration`）翻译（spec 2.10 的 I2）。建议的一项，待控制者确认：`@nerve/services` 的 `helpers/index.ts` 只按名字转出 `normalizeAPIRequestURL`（spec 2.9 的 Minor 1、第 9 节第 7 条）。
- 收尾 review 写明：3.3 的表和清零计划、3.4 的体积对比、3.5 的锁文件、spec 第 9 节各项的裁定；把 M1 设计 12 节收尾一行改为"已完成"并加上 spec、plan、review 的链接，勾上 11 节，改总体设计 9.4 和 M1 的状态（时机按 spec 第 9 节第 3 条的裁定）。
- 收尾 review 的提交另外改（spec 2.9、2.10、8.1）：M8 的收尾交接加 AGPL 第 5(a)、5(d)、13 条的逐项验收和 jsDelivr 请求的处理，时点是"任何对外的网络部署之前"，条款的解读由负责人在 M8 定；M8 的收尾交接加界面文字的覆盖（"每个 M 把改到的界面文字接入 `t()`；M8 发布前中文覆盖"）；M2 的收尾交接加个人设置的主题下拉框的位置；M1 设计 3.1 的写法（"图片节点保留；附件由工作项的附件区承担，编辑器里是否支持由 M5 决定"）；Codex 评审报告的"处理结果"一节（第 9 节第 3 条的裁定）。

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
