# M2/收尾 closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 M2 设计第 14 节成立，按第 12 节的收尾一段关闭 M2（spec 第 2 节的分诊）：关键词守卫进入 `M2/closeout`，例外只剩跨 M 的三条；`/create-workspace` 挂载时不请求旧接口由故事守住；3.20 核对出的漏同步补上（各记为所在 Phase 的漏项）；M2 定下的长期规则（前端的会话分代、River 的升级）和负责人以后可选的两件事（argon2 调参、PAT 的派生）在总体设计里有唯一的落点；写好给 M3–M8 的交接（13.2 的每一行、M0-P2 和 M0-P6 剩下的两项、表外的遗留）；M2 收到的 10 份交接逐项写下结论并关闭；13.2 的每一行指向它的交接；全部门禁在收尾的头上重新证明一次。

**Architecture:** 9 个 Task，每个一个提交：守卫（T1）→ 故事（T2）→ 上级文档的漏同步（T3）→ 长期规则的落点（T4）→ 给 M3、M4 的交接（T5）→ 给 M5–M8 的交接和已有交接中过时的两处（T6）→ 关闭收到的交接（T7）→ M2 设计的 13.2 和 §16（T8）→ 全部门禁（T9）。除 T2 的故事以外都是文档和配置：每一处改动都写成下面的块，由 `$M2TMP/closeout/planapply.mjs` 精确地应用（原文不是恰好一处就停下、什么都不写）。状态翻转（总体设计 9.4 的"已完成"、M2 设计第 15 节的收尾一行、第 14 节的勾）按 M1 的先例放在收尾 review 的提交里，由控制者做，替换的全文在最后一节（spec 第 4 节 D1）。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、Playwright 1.63.0（取自 `e2e/`）、Go 1.27.1、golangci-lint 2.13.2；不加依赖。

**Spec:** `docs/v0/M2-auth/specs/closeout.md`（上级：`docs/v0/M2-auth/M2-design.md` 第 12 节的收尾、第 13、14 节）

**原型：** `$M2TMP/closeout/proto2`，本分支在 `11f773bd`（Task 0、Task 1 已提交）的克隆（`git clone --branch worktree-m2-closeout`），Task 2–8 依次用 `planapply.mjs` 应用本计划的块，每个 Task 之后跑它的检查，最后跑全部门禁（结果在各 Task 的预期和 spec 附录 A）。本计划的块依次应用到 `d97c513` 的另一份干净副本上，得到的树与原型逐字节相同（文末"块的核对"）。

**预检：** 预检的 19 条发现（`$M2TMP/m2-closeout/preflight.md`）和控制者的裁定（`$M2TMP/m2-closeout/preflight-rulings.md`）已并入本计划和 spec（spec 附录 C 逐条列出落在哪里）；裁定要求的对 M2 设计的重扫结果在 spec 2.4。

---

## Global Constraints

### 命令与环境

- **所有命令在 worktree 的根目录执行**，不要 `cd`，也不要把 `cd` 和任何命令写在一起。在包里执行用 `pnpm -C <目录>`，Go 用 `go -C server`。开始之前执行一次 `pnpm install --frozen-lockfile`；不做全局安装，不运行 `corepack enable`。
- **`$M2TMP`** 是
  `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-m2`。
  收尾的一次性脚本都在 `$M2TMP/closeout/`，不进仓库；不要用裸的 `/tmp`。
- **一个 Bash 调用只跑一个 git 命令**：不用 `;`、`&&`、管道串联 git，不用 `git -C`、`stash`、`clean`、`reset --hard`；不碰 `plane/`、`refer/`。
- **容器**：`make test`、`make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 不停、不重建，不执行 `make dev-db-down`、`make dev-db-reset`；不碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。跑完之后等一分钟，`docker ps -a --filter label=org.testcontainers=true --format '{{.Names}} {{.Status}} {{.CreatedAt}}'` 中没有这次运行建的容器（Ryuk 在测试进程退出后删除它们）。原型跑完时列出的是 14 个 2026-09-26、27 建的 `Created` 状态的旧容器，不是收尾建的，收尾不动它们。
- **文件内容**：块由 `planapply.mjs` 写入；手工补写时用编辑工具，不用带反引号的 heredoc。写完 `grep -rn -E '&lt;|&gt;|&amp;'` 本 Task 改过的文件，应当没有输出（Edit、Write 工具曾把 `<` 写成 `&lt;`）。
- **不留后台进程**：`make e2e` 自己停掉它起的 nerve 和容器。

### 块的写法和应用

- `` ````file <路径> ``：新文件的全文（块的每一行，外加结尾的换行）。
- `` ````old <路径> `` 紧跟 `` ````new <路径> ``：把恰好出现一次的旧文本换成新文本。
- 每个 Task 的块用一条命令应用：
  `node $M2TMP/closeout/planapply.mjs docs/v0/M2-auth/plans/closeout.md apply . <Task 编号>`
  它先核对本 Task 的每个旧文本在当前文件里恰好一处、新文件还不存在，全部成立才写，否则什么都不写并报出是哪一块。
- 全部块的核对：`node $M2TMP/closeout/planapply.mjs docs/v0/M2-auth/plans/closeout.md check $M2TMP/closeout/head`（`head/` 是 `git archive d97c513` 解开的树）。输出见文末"块的核对"。

### 沿用的裁定

1. **收尾不做首次同步**（M2 设计第 12 节）：3.20 的一行没有落地，就是那个 Phase 的漏项，收尾补上，spec 2.2 记为漏项并写明 Phase。
2. **状态翻转在收尾 review 的提交里**（M1 的先例：M1 收尾计划的"完成后"一节，时机按它"控制者评审补充"中对 M1 收尾 spec 第 9 节第 3 条的裁定）：总体设计 9.4 的"已完成"、M2 设计第 15 节收尾一行的"已完成"和 review 的链接、第 14 节的勾。T3 只把总体设计 9.4 的漏项（"未开始"、没有链接）改成"进行中"和设计文档的链接。
3. **一个事实只写在一处**：13.2 保留它的行（设计的记录），接收者一格改为指向交接的链接；交接是给接收 M 的工作副本，写给没读过 M2 的人，深入的理由链接 M2 设计的节，不照抄。长期规则（spec 第 4 节 D2、D3）写在总体设计，交接只指向它。
4. **架构层面的事不在收尾决定**：M3 的邀请令牌仍写"由 M3 的设计交负责人确认"；argon2 的缓解仍是负责人的事项。
5. **执行方式沿用 P5**：实现者不指定模型；T2 是唯一改代码的 Task，按常规评审；其余 Task 由评审者对照 spec 的分诊核对文字。

### 每个 Task 的固定节奏

1. 应用本 Task 的块（上面的命令），读 `git diff`，确认每一处都对应本 Task 的一条说明。
2. `grep -rn -E '&lt;|&gt;|&amp;'` 本 Task 改过的文件 → 没有输出。
3. `node tools/keywords.mjs` → `keywords: 60 rules, 3 exceptions, no hits.`（每个 Task 都一样）。
4. 本 Task 写明的其余检查（改到守卫读取的文件时 `make lint-web`）。
5. 提交：一个 Task 一个提交，提交信息用英文，最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只有本 Task 的文件。

### 风险点（每个 Task 报告必答）

1. **只改本 Task 的文件**：`git diff --stat HEAD~1` 与本 Task 的文件表一致。
2. **交接的每一项都能执行**：路径在 `d97c513` 上存在（T5、T6 的第 2 步核对），关闭条件可以核对（有命令、测试或 review 的一节）。
3. **没有复述**：新写的文字不照抄 M2 设计的长段；同一件事在交接和总体设计里都出现时，只有一处是全文。

### 一次性脚本

都在 `$M2TMP/closeout/`，从仓库根目录运行。

| 脚本 | 用途 |
|---|---|
| `planapply.mjs <plan> check <基树>`、`… apply <根> <Task>`、`… list` | 核对全部块；应用一个 Task 的块；列出块 |
| `gates.sh <根> <标签> [门禁…]` | 第 14 节的七个门禁，每个的全部输出在 `$M2TMP/closeout/logs/<标签>-<门禁>.log`，一行摘要在 `logs/<标签>.summary`（原型用它；T9 直接跑 `make`） |
| `probes/expiry.mjs` | 守卫的例外按 `phase` 过期（T1） |
| `probes/mount-mutation.sh <根>` | 挂载时的旧接口请求被 A3 发现（T2） |
| `probes/control-full.sh <根> <d97c513 的树>` | 对照：`d97c513` 的整套故事发现不了同样的请求（spec 附录 A.3） |
| `probes/ui-i18n.sh <根>` | `@nerve/ui` 不能从 `@nerve/i18n` 导入（T4） |
| `rows.mjs <根> M<n>…`、`… --design` | 13.2 的每一行落在接收 M 的交接的一节；13.2 的接收者一格链接到交接（T5、T6、T8） |
| `treediff.mjs <树 a> <树 b>` | 两棵树逐字节比较，不含 `node_modules`、`.git` 和构建产物（文末"块的核对"） |
| `at.mjs <目录> <命令…>` | 以 `<目录>` 为工作目录运行命令（原型里用，worktree 里不需要） |
| `measure/measure.mjs` | 7.10 的实测（spec 附录 B） |
| `measure/ci-times.sh <完整提交> …`、`… --run <运行 id> …` | 持续集成上 e2e 任务的时长（公开的 GitHub API，`curl` + `jq`） |
| `paths.mjs <根> <交接>…`、`… --links <文件>…` | 交接里写的每个仓库路径在 `<根>` 上存在（T5、T6）；相对链接的目标存在（T8） |

### 文件结构

| 文件 | 改动 | Task |
|---|---|---|
| `tools/keywords.json` | `phase` 改为 `M2/closeout` | 1 |
| `e2e/stories/identity/a3-sign-in.spec.ts` | 落到 `/create-workspace` 的四个页面各有 `watchPage`，断言没有旧接口请求、没有失败的接口请求、没有未处理的异常 | 2 |
| `docs/v0/v0-design.md` | 9.4 的 M2 一行（P1 的漏项）；7.1 的 `packages/services`（P5 的漏项） | 3 |
| `docs/v0/M0-foundation/M0-design.md` | 3.7 的规则 4、6、8（P1 的漏项） | 3 |
| `docs/v0/plane-diff.md` | 第四节"退出、修改密码、停用之后的旧凭证"（P3a 的漏项） | 3 |
| `docs/v0/v0-design.md` | 新的 7.7（stores 按会话分代）；6.3 规则 5 的契约文件；5.2 的 River 升级；第 10 节的校准一行、argon2 一行、PAT 派生一行 | 4 |
| `docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md` | 新文件 | 5 |
| `docs/v0/M5-files/handoffs/M2-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md`、`docs/v0/M7-collaboration/handoffs/M2-closeout.md`、`docs/v0/M8-open-release/handoffs/M2-closeout.md` | 新文件 | 6 |
| `docs/v0/M{3,4,5,6,7,8}-*/handoffs/M1-closeout.md` | 候选规则中的 `no-unneeded-ternary` 已由 M2/P5 清零 | 6 |
| `docs/v0/M5-files/handoffs/M1-P3-trim-platform.md` | 资源类型现在是 6 种 | 6 |
| `docs/v0/M2-auth/handoffs/*.md`（10 个） | `## 处理结果（M2/收尾）`，`status: closed` | 7 |
| `docs/v0/M2-auth/M2-design.md` | 13.2 的接收者指向交接（关系图一行加上 M7）；§16 的 argon2 一行、PAT 派生一行指向总体设计第 10 节 | 8 |
| — | 全部门禁；核对 | 9 |

收尾 review 的提交（控制者）另改 `docs/v0/v0-design.md` 9.4 和 `docs/v0/M2-auth/M2-design.md` 的第 14、15 节，见最后一节。

---

## Tasks

### Task 0: M2 设计第 15 节的收尾一行（已随 spec 和本计划提交）

不由实现者做：架构师提交 spec 和本计划时，同一个提交把收尾一行改为"进行中"，带上两个链接。写在这里，是为了让块的核对从 `d97c513` 出发、按顺序走到收尾 review 的替换。

````old docs/v0/M2-auth/M2-design.md
| 收尾 | closeout | 未开始 | — | — | — |
````

````new docs/v0/M2-auth/M2-design.md
| 收尾 | closeout | 进行中 | [spec](specs/closeout.md) | [plan](plans/closeout.md) | — |
````

### Task 1: 关键词守卫进入 `M2/closeout`（已提交：`11f773bd`）

预检之前已按下面的块提交，与块相同。

spec 2.5。`phase` 从 `M2/P5` 改为 `M2/closeout`。三条例外的 `until` 是 M3、M6、M9，都在 `M2/closeout` 之后，照旧有效；`until` 为 M2 的任何 Phase 的例外从此都算过期。P1–P5 的例外都已在各自的 Phase 删掉，没有要删的。

**文件：** `tools/keywords.json`

- [ ] **Step 1：应用块**

````old tools/keywords.json
{
  "phase": "M2/P5",
  "rules": [
````

````new tools/keywords.json
{
  "phase": "M2/closeout",
  "rules": [
````

- [ ] **Step 2：守卫**

`node tools/keywords.mjs` → `keywords: 60 rules, 3 exceptions, no hits.`

`node -e 'const k=require("./tools/keywords.json"); console.log(k.phase); for (const e of k.exceptions) console.log(e.until, e.rule, e.path)'` →

```
M2/closeout
M6 analytics web/apps/web/core/services/cycle.service.ts
M3 project-invitations web/apps/web/core/services/user.service.ts
M9 brand pnpm-workspace.yaml
```

- [ ] **Step 3：过期的探针**

`node $M2TMP/closeout/probes/expiry.mjs` 依次把第一条例外的 `until` 改为 `M2/P5`、`M2/closeout`、`M3`，各跑一次 `node tools/keywords.mjs`，最后写回原来的字节并核对。预期：

```
until M2/P5: exit 1, expired exception: analytics  web/apps/web/core/services/cycle.service.ts  "analytics" is until "M2/P5", at or before phase "M2/closeout"; delete it
until M2/closeout: exit 1, expired exception: analytics  web/apps/web/core/services/cycle.service.ts  "analytics" is until "M2/closeout", at or before phase "M2/closeout"; delete it
until M3: exit 0, keywords: 60 rules, 3 exceptions, no hits.
tools/keywords.json restored: identical
```

- [ ] **Step 4：`make lint-web`**（守卫读取的文件改了）→ `keywords: 60 rules, 3 exceptions, no hits.`，`Tasks:    54 successful, 54 total`。

- [ ] **Step 5：提交**（`chore(M2/closeout): the keyword guard enters M2/closeout`）

### Task 2: `/create-workspace` 挂载时不请求旧接口，由 A3 守住

spec 2.6 第 5 行、3.2。第 14 节要求"M2 能到达的页面挂载时不请求 M3 的旧接口"。M2 能到达的页面（M2 设计 3.1）中，登录页由 S2 的 `apiRequests`（只有 `GET /api/v0/instance`）守住，注册页和 `/onboarding` 由 A10 的 `oldApiRequests` 守住，个人设置的四个标签页由 A7、A8、A9、A11、A12 守住（`oldApiRequests`，或者 `apiFailures` 的全等：旧接口在 nerve 上是 404，会出现在失败的请求里）。只有 `/create-workspace` 没有常设的守卫：A3 的页面测试在不合格的 `next_path` 之后落到它，但不看请求；P4 的浏览器核对 C8 看过一次。本 Task 让这四个页面各自带上 `watchPage`，落到 `/create-workspace`、表单出现之后断言没有旧接口请求、没有失败的接口请求、没有未处理的异常。

**文件：** `e2e/stories/identity/a3-sign-in.spec.ts`

- [ ] **Step 1：应用块**

````old e2e/stories/identity/a3-sign-in.spec.ts
import { bearer, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````

````new e2e/stories/identity/a3-sign-in.spec.ts
import { bearer, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";
````

````old e2e/stories/identity/a3-sign-in.spec.ts
  // A next_path that could lead elsewhere is dropped: the account's default page instead.
  await Promise.all(
    ["//evil.example", "/\\evil.example", "javascript:alert(1)", "/\t/evil.example"].map(async (nextPath) => {
      const other = await freshPage(browser, nerve.baseURL);
      await other.goto(`/?next_path=${encodeURIComponent(nextPath)}`);
      expect(await submitSignIn(other, email, password), nextPath).toBe(200);
      await expect(other, nextPath).toHaveURL("/create-workspace");
      await other.context().close();
````

````new e2e/stories/identity/a3-sign-in.spec.ts
  // A next_path that could lead elsewhere is dropped: the account's default page instead. That page,
  // /create-workspace, is one M2 reaches, so it asks no older API as it mounts (M2 design 3.1): the
  // workspace addresses answer 404 until M3.
  await Promise.all(
    ["//evil.example", "/\\evil.example", "javascript:alert(1)", "/\t/evil.example"].map(async (nextPath) => {
      const other = await freshPage(browser, nerve.baseURL);
      const watch = await watchPage(other);
      await other.goto(`/?next_path=${encodeURIComponent(nextPath)}`);
      expect(await submitSignIn(other, email, password), nextPath).toBe(200);
      await expect(other, nextPath).toHaveURL("/create-workspace");
      await expect(other.locator("#workspaceName"), nextPath).toBeVisible();
      expect(watch.oldApiRequests, nextPath).toEqual([]);
      expect(watch.apiFailures, nextPath).toEqual([]);
      expect(watch.pageErrors, nextPath).toEqual([]);
      await other.context().close();
````

- [ ] **Step 2：格式和类型**：`make lint-web` → `keywords: 60 rules, 3 exceptions, no hits.`，`Tasks:    54 successful, 54 total`。

- [ ] **Step 3：故事**：`make e2e` → `48 passed`（测试数不变）。

- [ ] **Step 4：不改页面时稳定**：新的断言在表单出现之后把数组读一次，这一步看它在页面没有变异时是否稳定。`make build`，然后 `NERVE_VERSION=0.1.0-dev pnpm -C e2e exec playwright test stories/identity/a3-sign-in.spec.ts --repeat-each 10` → `30 passed`（A3 的三个测试各十遍）。原型（`$M2TMP/closeout/proto2`）：`30 passed (9.0s)`。

- [ ] **Step 5：变异**：`bash $M2TMP/closeout/probes/mount-mutation.sh .` 在 `web/apps/web/app/(all)/create-workspace/page.tsx` 里依次放进两种挂载时的旧接口请求（`useEffect` 里的 `fetch("/api/workspaces/")`；`useSWR` 的 fetcher 里的同一个请求），每种 `make build` 之后跑三遍 A3（`--repeat-each 3`），最后写回原文件、再 `make build` 一次。预期每种变异都让 A3 的第一个页面测试三遍都失败在 `watch.oldApiRequests`（收到 `"GET /api/workspaces/"`），另外两个测试（第二个页面测试和接口版本）三遍都通过：

```
mutation effect-fetch: a3 exit 1;  3 failed  6 passed (4.7s) ; failures quoting watch.oldApiRequests: 3; received: "GET /api/workspaces/" 
mutation swr-fetch: a3 exit 1;  3 failed  6 passed (4.6s) ; failures quoting watch.oldApiRequests: 3; received: "GET /api/workspaces/" 
restored: identical
rebuilt: ok
```

写回之后 `git status --short` 只有本 Task 的文件。原型另把 A3 换回 `d97c513` 的版本跑了同样两种变异：`9 passed`，两种都没有被发现；`d97c513` 的整套故事加上 `useEffect` 的变异也是 `48 passed`（spec 附录 A.3）。这就是本 Task 补上的缺口。

- [ ] **Step 6：提交**（`test(M2/closeout): A3 watches /create-workspace for older API requests as it mounts`）

### Task 3: 上级文档的漏同步

spec 2.2。3.20 的规则是"每处偏离，在实现它的那个 Phase 的同一次合并中同步"。逐行核对之后，下面四处没有同步，各是所在 Phase 的漏项：

| 文档与位置 | 漏的 Phase | 现在写的 | 事实（`d97c513`） |
|---|---|---|---|
| 总体设计 9.4 的 M2 一行 | 设计文档合并（`fd697367`）和 P1 | "未开始"，设计文档一格是"—" | M2 从 2026-09-25 起进行中，设计文档是 `M2-auth/M2-design.md`；M0、M1 两行的格式是状态加链接 |
| 总体设计 7.1"暂时使用" | P5 | web 的令牌设置页依赖 `packages/services`，M2 和 M5 对接时删除 | P5 删掉了令牌服务和 axios 基类，只剩地址规范化和文件工具，整包在 M5 删（前端改动清单 1 节第 18 行、3.1 最后一行已经这样写） |
| M0 设计 3.7 的规则 4、6、8 | P1 | 规则 4 只说 `modules`、`bootstrap`；规则 6 只说 http 适配器；规则 8 只有 `pgtest`、`apitest` | `server/internal/archtest/rules_test.go:40,42,44`：平台也不导入 `internal/shared`；生成的代码只能被它自己的适配器导入（含 sqlc 的 `adapter/postgres/gen`）；测试工具还有 `clocktest`（都在 P1 的 `448a138a`） |
| 差异清单第四节"修改密码、停用之后的旧凭证" | P3a | 行名少了"退出" | M2 设计 4.6 的行是"退出、修改密码、停用后旧凭证失效"，P3a spec 第 3 节第 21 条要求照登；内容"相同，由每个请求的会话检查做到"对退出同样成立 |

**文件：** `docs/v0/v0-design.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/plane-diff.md`

- [ ] **Step 1：应用块**

````old docs/v0/v0-design.md
| M2 | 账户认证 | 未开始 | — |
````

````new docs/v0/v0-design.md
| M2 | 账户认证 | 进行中 | [M2-design.md](M2-auth/M2-design.md) |
````

````old docs/v0/v0-design.md
- **暂时使用**：packages/services。web 中的令牌设置页和文件工具函数依赖它；M2（PAT）和 M5（文件）对接新接口时，将它删除。
````

````new docs/v0/v0-design.md
- **暂时使用**：packages/services。M2/P5 删掉了其中的令牌服务和 axios 基类，web 只剩地址规范化和上传文件的元数据工具依赖它；M5 对接文件接口时，将它删除（[前端改动清单](frontend-changes.md) 3.1）。
````

````old docs/v0/M0-foundation/M0-design.md
  4. `platform` 不能导入 `modules` 和 `bootstrap`。
````

````new docs/v0/M0-foundation/M0-design.md
  4. `platform` 不能导入 `modules`、`bootstrap` 和 `internal/shared`（M2/P1 加上后者：平台声明自己需要的小接口，`shared` 的类型按结构满足它们）。
````

````old docs/v0/M0-foundation/M0-design.md
  6. 生成的代码只能被本模块的 http 适配器导入。
````

````new docs/v0/M0-foundation/M0-design.md
  6. 生成的代码（`adapter/<技术>/gen`）只能被同一个适配器导入（M2/P1 从 http 适配器推广到所有适配器，包括 sqlc 的 `adapter/postgres/gen`）。
````

````old docs/v0/M0-foundation/M0-design.md
  8. 测试工具（`pgtest`、`apitest`）只能被测试代码导入。
````

````new docs/v0/M0-foundation/M0-design.md
  8. 测试工具（`pgtest`、`apitest`，M2/P1 加上 `clocktest`）只能被测试代码导入。
````

````old docs/v0/plane-diff.md
| 修改密码、停用之后的旧凭证 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（M2 设计 3.5） |
````

````new docs/v0/plane-diff.md
| 退出、修改密码、停用之后的旧凭证 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（M2 设计 3.5） |
````

- [ ] **Step 2：核对规则的原文**：`grep -n -E '"(platform does not import|generated code is imported|test helpers)' server/internal/archtest/rules_test.go` →

```
40:		{"platform does not import modules, bootstrap or internal/shared", platformIsBusinessFree},
42:		{"generated code is imported only by its own adapter", generatedCodeStaysInAdapter},
44:		{"test helpers (pgtest, apitest, clocktest) are imported only by tests", testHelpersOnlyInTests},
```

- [ ] **Step 3：守卫**（固定节奏第 3 步）。

- [ ] **Step 4：提交**（`docs(M2/closeout): what the phases did not carry into the parent documents`）

### Task 4: 长期规则的落点

spec 第 4 节 D2、D3、D4 和 2.4。五处都在总体设计：

1. **新的 7.7**：M2 定下的前端会话约定（M2 设计 13.2"M3 及以后有 stores 的 M"一行）。它不是一次性的任务：M3–M8 都要建 stores，交接关闭之后规则还要在。7.7 是它唯一的全文，4.3 讲浏览器端的行为，7.7 讲代码怎样做到；各 M 的交接只列自己的 stores 和 services，指向 7.7。
2. **6.3 规则 5**：约 400 行的规则不适用于接口描述的模块文件（M0-P3 交接第 5 条的组织规则：一个模块一个文件）。这是 P3a 评审交给"P3b 及以后的 plan"的事项，P3b、P4、P5 的 plan 各自写进 Global Constraints，没有长期的落点。
3. **第 10 节的校准一行**：总体设计让 M2 校准前端改动量的估算，结果写在收尾 spec 的附录 B，这一行指向它。
4. **第 10 节新加两行**：负责人以后可以选的两件事，都不是任何 M 的任务，M2 关闭之后要有地方记着（spec 第 4 节 D4）：
   - 调高 argon2 参数之后，登录耗时能区分休眠的账户和不存在的邮箱，是否加缓解（M2 设计 §16）；
   - 泄露的刷新令牌可以派生永不过期的 PAT：创建 PAT 时重新输入密码、限制派生是可选的产品选项（M2 设计 §16、8.5）。
5. **5.2 的 River 迁移一句之后**：升级 River 时另写一份迁移、不改已发布的文件（M2 设计 3.15）。这是以后每次升级都要照做的规则，只写在 M2 设计里（收尾重扫 M2 设计找到，spec 2.4）。

**文件：** `docs/v0/v0-design.md`

- [ ] **Step 1：应用块**

````old docs/v0/v0-design.md
   - 一个文件只做一件事；超过约 400 行的文件在评审时必须说明理由或者拆分。
````

````new docs/v0/v0-design.md
   - 一个文件只做一件事；超过约 400 行的文件在评审时必须说明理由或者拆分。接口描述按模块一个文件（3.1），不受这一条的限制（M2/P3a 评审）。
````

````old docs/v0/v0-design.md
- 新写的代码遵循 7.2 的职责划分，和 Plane 现有的写法保持一致（MobX store、`observer` 组件）。
````

````new docs/v0/v0-design.md
- 新写的代码遵循 7.2 的职责划分，和 Plane 现有的写法保持一致（MobX store、`observer` 组件）。

### 7.7 stores 按会话分代（长期有效，M2 起）
4.3 的"不以别的会话的身份发请求"在代码上靠下面几条做到（M2 设计 7.1、7.5，M2/P4 spec 2.8）。M3 以后每个接上新接口的 store、service 和页面都照做；各 M 的交接只列自己的范围，不重复这些规则。
- **每个会话一个 `RootStore`**：会话（`login_id`）每次变化，`store-context.tsx` 建一个新的 `RootStore`，交给它这个会话的客户端（`apiFor(loginId)`）。只有不属于账户的 `instance`、`router`、`theme` 从上一代沿用，其余 store 都新建。组件经 `StoreProvider` 拿到此刻的 `RootStore`，换代之后以新的一代重新渲染。
- **service 从构造函数拿这一代的客户端**：store 在构造时建它的 service，传入这一代的客户端。没有模块级的带令牌客户端，也没有用它的模块级 service 实例；只调公开操作的 service 用 `publicClient`，可以是模块级的（例如 `core/hooks/use-timezone.tsx` 的 `TimezoneService`）。
- **store 只经自己的 `RootStore` 找兄弟 store**。`store-context.tsx` 导出的 `rootStore` 只给没有自己 `RootStore` 的代码做同步读取，不发请求。
- **填充 stores 的 SWR 键带上 `loginId`**（例如 `["CURRENT_USER", loginId]`）：换了会话，新的一代取自己的数据。
- **一代退役时释放它的反应**：注册在跨代沿用的对象（`router` 等）上的 `reaction`、`autorun` 不会随旧的一代回收。M2 结束时 `RootStore` 还没有释放的方法：第一个接上这类 store 的 M 给 `RootStore` 加上它，由 `store-context.tsx` 在换代时调用，之后的 M 照做（要释放的反应列在各 M 的交接里）。只观察本代对象的反应不用释放。
- **`SessionChangedError` 不是认证失败**：它表示请求属于标签页已经离开的会话：请求在发出之前被拦下；或者请求（或续期之后重发的副本）得到 401 时，标签页已经离开它的会话，于是不再续期、重发，也不结束标签页此刻的会话。它只出现在请求发出之前或一个 401 之后，所以操作确实没有做。调用方不退出、不跳转；后台加载忽略它，由新的一代重新取数；用户发起的操作被它截断时可以提示失败。
- **页面级的状态从当前的 `RootStore` 派生**：界面语言、主题这类作用于整个页面的状态，由 `StoreWrapper` 读当前一代的资料来设置；store 不直接改全局状态，旧一代迟到的应答就改不到新会话的页面（M2/P5 spec 第 3 节第 16 条）。
- **`@nerve/ui` 不依赖 `@nerve/i18n`**：组件的文字由调用方经 `t()` 以 props 传入。`@nerve/ui` 的 `package.json` 不声明 `@nerve/i18n`，pnpm 不把没有声明的包放进它的 `node_modules`，从 `@nerve/i18n` 导入的名字过不了 `@nerve/ui` 的类型检查。
- 代码在 `web/apps/web/core/lib/store-context.tsx`、`core/store/root.store.ts`、`core/lib/auth/api-client.ts`；测试和变异见 M2/P4 spec 2.8、M2/P5 spec 2.4。
````

````old docs/v0/v0-design.md
| 前端对接新接口时，store 和组件的实际改动量 | 各 M 的设计文档中评估；M2 是第一个对接的领域，用它来校准后续的估算 |
````

````new docs/v0/v0-design.md
| 前端对接新接口时，store 和组件的实际改动量 | 各 M 的设计文档中评估；M2 是第一个对接的领域，用它来校准后续的估算：实测和对 M3 以后的含义见 [M2 收尾 spec](M2-auth/specs/closeout.md) 附录 B |
````

````old docs/v0/v0-design.md
| 没有邮件服务时账户如何找回（目前只能由管理员用命令行重置） | 以后接入邮件服务时再议（v0 之后） |
````

````new docs/v0/v0-design.md
| 没有邮件服务时账户如何找回（目前只能由管理员用命令行重置） | 以后接入邮件服务时再议（v0 之后） |
| 调高 argon2 参数（`argon2_memory_kib`、`argon2_iterations`）之后，登录耗时能区分休眠的账户和不存在的邮箱，直到每个账户重新登录一次（M2 设计 §16） | 是否加缓解由负责人以后决定；M8 的 argon2 实测提供数字（[M8 的交接](M8-open-release/handoffs/M2-closeout.md)第 2 节） |
| 泄露的刷新令牌可以派生永不过期的 PAT，影响不以刷新令牌的 30 天为限（M2 设计 §16、8.5） | 恢复步骤已写在 README 的"部署"一节（撤销不认识的 PAT，或由管理员 `nerve users reset-password`）；创建 PAT 时重新输入密码、不允许用 PAT 创建 PAT 或给 PAT 的有效期设上限，是负责人以后可选的产品选项，v0 不采用 |
````

````old docs/v0/v0-design.md
；不使用 River 自带的迁移命令（见 M2 的 [M0-P2-platform-notes](M2-auth/handoffs/M0-P2-platform-notes.md)）。
````

````new docs/v0/v0-design.md
；不使用 River 自带的迁移命令（见 M2 的 [M0-P2-platform-notes](M2-auth/handoffs/M0-P2-platform-notes.md)）。升级 River 时，用新版本的 CLI 导出新增的版本（`--version N`），另写一份 goose 迁移，不改已发布的迁移文件（M2 设计 3.15）。
````

- [ ] **Step 2：7.7 倒数第二条的探针**：`bash $M2TMP/closeout/probes/ui-i18n.sh .` 在 `web/packages/ui/src/index.ts` 末尾加一行 `export { useTranslation as probeUseTranslation } from "@nerve/i18n";`，跑 `pnpm -C web/packages/ui check:types`，再写回原文件。要用导入名字的写法：只有副作用的 `import "@nerve/i18n";` 解析不到时 TypeScript 不报错（原型第一次这样写，类型检查通过）。预期：

```
ui imports @nerve/i18n: check:types exit 2
src/index.ts(25,55): error TS2307: Cannot find module '@nerve/i18n' or its corresponding type declarations.
restored: identical
```

- [ ] **Step 3：守卫**（固定节奏第 3 步）。

- [ ] **Step 4：提交**（`docs(M2/closeout): the session rules of the stores, the contract files, River upgrades and the owner's two open options have a home`）

### Task 5: 给 M3、M4 的交接

spec 3.1。M2 设计 13.2 中接收者含 M3、M4 的每一行，拆成各自的部分；M4 另有 M0-P6 交接剩下的 `clock.ts`，以及表外的两项（游标不签名的提醒、定时任务漏注册 worker 时只有一条 WARN，spec 2.4）。负责人对只投递的 River 客户端的要求（"做好记录，以后别漏了"）是 M4 交接的第 1 节，关闭条件写明 M4 用不到时原样转交。

**文件：** `docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`（新文件）

- [ ] **Step 1：应用块**

````file docs/v0/M3-workspace-project/handoffs/M2-closeout.md
---
status: open
from: M2/closeout
to: M3
created: 2026-09-28
---

# M2 交给 M3：邀请与注册、登录后的落点、停用的端口、stores 按会话分代，以及 M2 改到却走不到的页面

M2（账户认证）做完了注册、登录、续期、退出、个人设置、PAT 和 `nerve users` 命令，前端有了令牌管理器和按会话分代的 stores（[M2 设计](../../M2-auth/M2-design.md)）。M2 没有工作区的接口，下面这些由 M3 接着做。每一节来自 M2 设计 13.2 中接收者含 M3 的一行（多个 M 的行只取 M3 的部分）；收尾重扫 M2 设计时另找到几项，放进相应的节：第 2 节的 `/api/users/me/settings/`，第 6 节的加锁顺序，第 8 节的 `Authorizer` 端口，第 11 节的 `owner` 类型和守卫的收紧。为什么这样定，看链接的节，这里写做什么、代码在哪、M2 做到哪一步、怎样算完。

## 1. 邀请：凭链接中的令牌接受；关闭注册时持有邀请的人仍可注册（交负责人确认）

- **做什么**：接受工作区邀请时，不能只靠邮箱匹配，要凭邀请链接中的令牌（M1 设计 3.15 留下的路径）；注册关闭时（prod 的默认，M2 设计决策点 2），持有有效邀请的人仍可注册。
- **为什么**：v0 没有邮件服务，邮箱未经验证。只靠邮箱匹配，谁先用别人的邮箱注册，谁就能接受发给那个邮箱的邀请。Plane 把任何未删除的工作区邀请都算上（`plane/apps/api/plane/authentication/adapter/base.py:102-120`）。prod 默认关闭注册不能代替接受邀请时的身份证明。
- **这是产品改动**：它改变总体设计 1.1"系统内接受邀请"和 4.2"被邀请的邮箱始终可以注册"的做法，**由 M3 的设计交负责人确认**。
- **代码在哪**：注册是否开放由端口 `SignupPolicy` 决定（`server/internal/modules/identity/app/ports.go:353`），现在由 `bootstrap` 接成配置开关 `signupSwitch`（`server/internal/bootstrap/app.go:191-194`）；注册用例在 `server/internal/modules/identity/app/register.go`。M3 扩展 `SignupPolicy` 的实现和注册请求（请求体加上邀请令牌），不另加端口（M2 设计第一稿评审 M16）。签发邀请令牌按账户行锁（M2 设计 3.5）。
- **关闭条件**：负责人的确认写进 M3 设计；只凭邮箱匹配的接受被拒绝，凭有效令牌的成功；`auth.signup_enabled = false` 时，带有效邀请令牌的注册成功、不带的得到"注册已关闭"；这些都有接口版本和页面版本的故事。

## 2. 登录后的落点与新手引导的取数

- **做什么**：在新接口上加回 M2 删掉的取数：`AuthenticationWrapper`（`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`）的落点数据（"上次的工作区"、工作区列表），新手引导页（`web/apps/web/app/(all)/onboarding/page.tsx`）的工作区和邀请。
- **为什么**：M2 的规则是"M2 能到达的页面挂载时不请求 M3 以后的旧接口"（M2 设计 3.1）。所以完成引导的用户登录后直接去 `/create-workspace`，新手引导不再预取，`OnboardingRoot` 的 `invitations` 取默认的空数组。
- **注意**：加回时，工作区取数的 SWR fetcher 要 `return`（或 `await`）`fetchWorkspaces()`（`web/apps/web/core/store/workspace/index.ts:141`）的 Promise。M2 删掉的原 fetcher（原在 `onboarding/page.tsx:33-37`）没有，失败会成为未处理的 Promise 拒绝。邀请页 `web/apps/web/app/(all)/invitations/page.tsx` 的同类缺陷 P5 已改：Promise 交给外层的 `.catch`，失败时提示（M2/P5 spec 第 3 节第 8 条）。SWR 键带上 `loginId`（第 3 节）。
- **Plane 的 `/api/users/me/settings/` 还在**（M2 设计 7.5：工作区数据留给 M3）：`web/apps/web/core/services/user.service.ts:43` 的 `currentUserSettings` 仍调它，nerve 答 404；`web/apps/web/core/store/user/settings.store.ts` 的 `fetchCurrentUserSettings` 把回答存成 `IUserSettings`，工作区 store 的 `getWorkspaceRedirectionUrl`（`web/apps/web/core/store/workspace/index.ts:93-99`）按其中的 `last_workspace_slug`、`fallback_workspace_slug` 算落点。创建、加入、删除工作区和移出成员之后有四处 `await fetchCurrentUserSettings()`（`web/apps/web/core/components/` 下的 `onboarding/steps/workspace/{create,join-invites}.tsx`、`workspace/delete-workspace-form.tsx`、`workspace/settings/members-list-item.tsx`）。M3 让落点由 `profiles.last_workspace_id` 和工作区列表算出，删掉这个方法、`IUserSettings` 的工作区部分和这四处调用，或者给它们新接口；不改的话，工作区接上之后这些 `await` 因 404 抛出，例如新手引导建好工作区却报失败。
- **守卫**：A3、A10 的页面测试用 `watchPage`（`e2e/fixtures/browser.ts`）断言这些页面挂载时没有旧接口请求（`oldApiRequests`）、没有失败的接口请求。加回的取数走 `/api/v0`，这些断言照旧通过；失败说明还在调旧地址。
- **关闭条件**：完成引导的用户登录后落到上次的工作区，没有时按 Plane 的规则落到第一个工作区或 `/create-workspace`；新手引导的工作区、邀请两步用新接口取数；A3、A10 的断言不改仍通过；取数失败时页面没有未处理的拒绝（故事或单元测试）；`currentUserSettings` 不再调 Plane 的地址（改接或删除），上面四处不再因它失败。

## 3. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M3 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。规则只写在那里；下面是 M3 的范围，在 `d97c513` 上数出。
  - **模块级的 service 实例**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 共 37 处，按调用的接口分，M3 的领域 10 处：`WorkspaceService` 8 处（`web/apps/web/app/(all)/invitations/page.tsx`、`web/apps/web/app/(all)/workspace-invitations/page.tsx`、`web/apps/web/core/components/account/auth-forms/auth-header.tsx`、`web/apps/web/core/components/onboarding/steps/team/root.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/{create,join-invites}.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/core/store/user/permissions.store.ts`），`ProjectService` 1 处（`web/apps/web/core/components/project/form.tsx`），`ProjectMemberService` 1 处（`web/apps/web/core/services/project/project-member.service.ts`）。同是这两个类的另外 9 处调的是搜索、编辑器的 @提及 和最近访问，在 [M4](../../M4-issue-core/handoffs/M2-closeout.md) 第 12 节、[M7](../../M7-collaboration/handoffs/M2-closeout.md) 第 2 节。它们现在经 web 的 axios 基类（`web/apps/web/core/services/api.service.ts`，不带令牌）调 Plane 的旧地址，nerve 答 404。接上新接口时改为 store 构造时用这一代的客户端建。
  - **注册在沿用的 `router` 上的反应**：`web/apps/web/core/store/project/project_filter.store.ts:63` 的 `reaction`。它只做本地的同步更新、不发请求，但不随退役的一代释放，每换一次会话留下一代 stores 的内存。
  - **SWR 键**：工作区、项目、成员的 SWR 键带上 `loginId`。
- **M2 做了什么**：P4 建立分代（`web/apps/web/core/lib/store-context.tsx`、`web/apps/web/core/store/root.store.ts`、`web/apps/web/core/lib/auth/api-client.ts` 的 `apiFor`），P5 的 PAT store（`web/apps/web/core/store/user/api-token.store.ts`）照做（M2/P4 spec 2.8、M2/P5 spec 2.4）。
- **关闭条件**：M3 合并时，上面的 `git grep` 中 M3 领域的 10 处都已消失（或是只调公开操作的 `publicClient`）；`project_filter` 的反应随退役的一代释放，有单元测试（换代之后改路由，旧一代的反应不再运行）；M3 新加的 SWR 键带 `loginId`。

## 4. `profiles.last_workspace_id` 的外键

- **做什么**：决定是否给 `profiles.last_workspace_id` 补外键 `REFERENCES workspaces ON DELETE SET NULL`。
- **为什么**：M2 建 `profiles` 时还没有 `workspaces`，这一列照搬 Plane，是可空、不是外键的 `uuid`（M2 设计 4.3）。
- **怎样补**：迁移归 `identity`：文件名 `<v>_identity_profiles_last_workspace_fk.sql`，列在 `server/sqlc.yaml` 中 `identity` 的条目里，版本号大于建 `workspaces` 的迁移（总体设计 5.6，M2 设计 3.14）。`TestSQLCSchemaScope`（`server/internal/archtest/sqlc_test.go`）核对 `ALTER TABLE` 的所有者。
- **关闭条件**：M3 设计写明补或不补和理由；补的话迁移照上面命名，`server/migrations/schema_test.go` 核对约束名，差异清单二·按表登记。

## 5. `workspace_creation_enabled` 的执行

- **做什么**：配置 `workspace.creation_enabled` 为 `false` 时，创建工作区的接口拒绝；决定是否提供创建工作区的命令（Nerve 没有实例管理员，M2 设计 3.16）。
- **M2 做了什么**：配置和实例接口有这个字段（`server/internal/modules/instance/domain/info.go:22`，`GET /api/v0/instance` 的 `workspace_creation_enabled`），前端四处读它（`/create-workspace`、新手引导的创建工作区、工作区菜单、命令面板）。服务端还没有执行它。
- **关闭条件**：接口在关闭时拒绝，有错误码和测试；是否提供命令写进 M3 设计。

## 6. 停用的端口（决策点 3 已裁定为 A）

- **做什么**：给停用用例加上它声明的端口，在停用的同一个事务里调用：唯一管理员时拒绝；停用成员关系；删除发给这个邮箱的邀请。给 `deactivateMe`（`api/modules/identity.yaml`）声明对应的错误码。
- **为什么**：M2 的停用只处理账户自己的数据（会话、新手引导，M2 设计 3.5）。唯一管理员的检查按 Plane 的本意做，修正它查询的缺陷（Plane 的检查从不拒绝，M2 设计决策点 3），登记差异清单第四节。
- **代码在哪**：`server/internal/modules/identity/app/deactivate.go`：自助停用（`POST /api/v0/me/deactivate`）走 `Execute`，`nerve users deactivate` 走 `ExecuteByEmail`，两者都在账户行锁下调同一个 `deactivate`；端口在 `deactivate` 里调用，两条路就都经过它。端口在 `identity/app` 声明，由 M3 的模块实现，`bootstrap` 接上（总体设计 6.2）。
- **加锁顺序**：端口在账户行锁之内被调用。M2 的全局加锁顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens`，停用按它依次写（M2 设计 3.5）；成员关系、邀请的写入排在这个顺序的哪里，由 M3 定下，写进 M3 设计（M2 设计 6.4 的停用一条："另按 M3 定下的顺序"）。
- **注意**：`nerve users deactivate` 也走这个用例，而命令行的组合没有 River 客户端（`server/internal/bootstrap/users.go:22-26`）。端口的实现如果投递任务，先要有"只投递"的 River 客户端：见 [M4 的交接](../../M4-issue-core/handoffs/M2-closeout.md)第 1 节，那一项随之提前到 M3。
- **关闭条件**：三件事在停用的事务里完成，唯一管理员时停用被拒绝，接口和命令两条路都有测试；`deactivateMe` 的错误码已声明，`apitest` 两个方向的核对通过；差异清单第四节"停用账户"一行更新；M3 的写入在全局加锁顺序中的位置写进 M3 设计。

## 7. 可空的引用字段（M2 设计 3.2）

- **做什么**：工作区图标、项目封面、成员头像随实体进入接口，M5 之前为 `null`；`IUserLite.avatar_url`（`web/packages/types/src/users.ts:24`，现在是 `string`）改为可为 `null`。读取它们的代码不动。
- **为什么**：M5 才有文件存储。先返回 `null`，免得 M3 删掉字段、M5 再加回，前端改两次。只适用于本来允许为空的引用字段。
- **关闭条件**：这些字段在 M3 的接口描述中可为 `null`；`IUserLite` 改用生成的类型，或已改为可空。

## 8. 模块边界：`Authorizer` 端口、sqlc 与 `TestSQLCSchemaScope`

- **做什么**：
  1. `access` 模块读成员表：在 `app` 层声明端口，还是作为 `TestSQLCSchemaScope` 中写明理由的例外（M2 设计 3.14）。
  2. `TestSQLCSchemaScope` 的所有者规则用正则识别 `ALTER TABLE`，漏掉带引号的标识符，以及别的模块的表上 `ALTER` 以外的 DDL（`CREATE INDEX … ON users`、`CREATE TRIGGER`、`DROP TABLE`）。M3 是第一个有跨模块外键的 M，补上这几种写法和它们的反例（M2/P1 评审 M5）。
  3. `Authorizer` 端口由 M3 加入 `internal/shared`（M2 设计 3.3；总体设计 6.2 列了它）。M2 的 `shared` 有 `Actor`、领域错误、`TxManager` 和游标的封套（`server/internal/shared/`），权限检查的端口还没有。
- **代码在哪**：`server/internal/archtest/sqlc_test.go`、`server/internal/archtest/sqlc_cases_test.go`、`server/sqlc.yaml`、`server/internal/shared/`。
- **关闭条件**：M3 设计写明第 1 件的选择；第 2 件的每种写法都有一个反例，规则漏掉它时测试失败；`Authorizer` 在 `shared` 声明，由 M3 的模块实现，`shared` 仍只依赖标准库（架构测试规则 10）。

## 9. 物理删除与跨模块外键的关系图（M3 的部分）

- **做什么**：M3 的每张新表照 M2 设计 3.13 搬 Django 模型的 `on_delete`，并在 M2 设计 4.7 的图上延伸，写明物理删除时每条外键的去向。
- **为什么**：Plane `project.py:77-89` 的项目负责人是 `CASCADE`：物理删除一个账户会连带删除项目。M2 只停用、不删除账户；M3 写明允许物理删除的范围。M4、M6、M7 各自延伸自己的表（它们的交接）。
- **关闭条件**：M3 设计有延伸后的图，每条指向 `users` 的外键写明去向；项目负责人的 `CASCADE` 有结论（照搬，或改为 `SET NULL`、`RESTRICT` 并登记差异）。

## 10. CSP：表情选择器的数据从本站提供

- **做什么**：表情选择器（`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx` 的 `EmojiPicker.Root`，来自 `frimousse`）运行时从 `cdn.jsdelivr.net/npm/emojibase-data` 下载数据。改为随前端一起构建、从本站提供（设置 `emojibaseUrl`，固定版本）。
- **为什么**：页面的 CSP 是 `connect-src 'self'`（`server/internal/platform/webui/csp.go:42`），这个下载现在被挡住。自托管的 Nerve 应当在没有外网时也能用，也不把使用情况告诉第三方（M2 设计 8.3）。项目图标最先用到它，所以在 M3。
- **与 M8 的关系**：[M8 的 M1 收尾交接](../../M8-open-release/handoffs/M1-closeout.md)"AGPL 的三项义务和发往第三方的请求"列了同一个请求；M3 改完之后，那一项对它只剩核对。
- **关闭条件**：打开表情选择器时页面不请求 `cdn.jsdelivr.net`，没有 CSP 违规（故事的 `cspViolations` 为空，或浏览器核对）；CSP 没有为它放开外部来源。

## 11. M2 留下的 M3 调用和类型

- 新手引导的创建工作区、加入工作区、邀请成员三步：界面照常出现，提交调 Plane 的旧接口，M3 之前得到 404（M2 设计 3.1）。
- `web/apps/web/core/services/user.service.ts` 中留下的 `leaveWorkspace`、`joinProject`、`leaveProject`（第 71–87 行）：改接新接口。`joinProject` 的关键词守卫例外见 [M1-P3 的交接](M1-P3-trim-platform.md)"项目成员"一节。
- `IUserLite` 的 `is_bot`（`web/packages/types/src/users.ts:29`）：Nerve 不区分人和机器人（总体设计 0.2），删除。
- 时区接口 `GET /api/v0/timezones`（公开操作，`web/apps/web/core/services/timezone.service.ts`）也供工作区和项目设置使用，不另建。
- 工作区类型的 `owner`（`web/packages/types/src/workspace.ts:20`）：M2 删掉 `IUser` 时改为 `IUserLite`，M3 对接工作区时按接口再定（M2 设计 7.5）。
- **守卫的收紧**：规则 `plane-user-urls`（`tools/keywords.json`）有意不禁止整个 `/api/users/me/`，因为 `joinProject`、工作区列表、邀请和 `/settings/` 这些 M3 的旧调用还在用它下面的地址（M2 设计 7.9）。M3 迁走 `/api/users/me/…` 下最后一个旧调用之后，把规则收紧为整个 `/api/users/me/`，`samples` 随之改。
- **关闭条件**：三步用新接口；这三个方法改用生成的客户端或删除；`is_bot` 删除；工作区、项目设置的时区选择用这个接口；`owner` 的类型取自生成的类型；`plane-user-urls` 已收紧（或 M3 的 review 写明还有哪个旧调用留在 `/api/users/me/` 下、交给哪个 M）。

## 12. 页大小的规则移到 `shared`

- **做什么**：`limit` 的 1–100、默认 50（`api/common.yaml` 的 `Limit`）现在在 `identity/domain`（`server/internal/modules/identity/domain/api_token.go:128-140` 的 `PageSize`），因为 M2 只有 PAT 列表一个使用者（M2/P3a spec 第 3 节第 8 条）。第二个分页列表出现时移到 `internal/shared`，两个列表共用。
- **同一处的提醒**：游标不签名（总体设计 3.4），改成另一个合格的位置照样可用。列表只能让游标决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：第二个分页列表合并时，页大小的规则在 `shared`，两个列表都用它；M3 没有分页列表时，M3 的 review 写明，把本节写进下一个 M 的交接。

## 13. P5 改到、M2 的页面走不到的地方（接上时核对）

它们在工作区、项目的路由上，M2 没有工作区的接口，故事和浏览器核对都到不了。M3 接上这些路由时核对：

- `WorkspaceAuthWrapper`（`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`）"找不到工作区"界面的退出登录改为 `<button type="button">`，名称和提示都经 `t("sign_out")`：Tab 能到，Enter、空格退出。
- 它和 `ProjectAuthWrapper`（`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`）删掉了没有调用方传入的 `isLoading`：工作区或权限加载时仍显示加载中，项目页在项目详情取到之后才显示。
- `ProfileSidebar`（`web/apps/web/core/components/profile/sidebar.tsx`）的窗口大小监听只注册一次，经 ref 读当前的折叠状态（原来读第一次渲染时的值）：宽屏下折叠，再把窗口拉过 768 像素，它重新展开。
- `WorkspaceLogo`（`web/apps/web/core/components/workspace/logo.tsx`）渲染 `<span>`：它在工作区菜单和邀请页的按钮里，按钮只能含短语内容，显示方式由它的 `grid` 类决定。P5 在邀请页上比对过改前改后的盒子和截图，M3 接上工作区菜单和卡片时核对。
- 新手引导加入工作区一步（`web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx`）的每个邀请由可点击的 `div` 改为 `<label>` 连着它的 `Checkbox`：点一行、在勾选框上按空格都勾选，Tab 能到勾选框。P5 用两条假的邀请比对过改前改后的截图（相同），M3 加回邀请的取数时核对。
- 这一步、邀请成员一步和导览的文案 P5 改经 `t()`，M3 接上时在中文下看一遍。
- **关闭条件**：M3 的浏览器核对逐条写明结果。

## 14. 下拉框和复制到剪贴板（M3 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。修法和根因见 M2/P5 spec 2.3。

- **`CustomSearchSelect` 的按钮不在 Tab 顺序里**（`web/packages/ui/src/dropdowns/custom-search-select.tsx`；Headless UI 的 `Combobox.Button` 固定 `tabIndex: -1`，P5 之前也是这样）：改成像 Popover 那样的按钮，`Combobox` 放在它的面板里。个人设置的时区选择就是它。
- **12 个同样写法的下拉框中 M3 的一个**：`dropdowns/member/member-options.tsx`（项目负责人，`projects/create/attributes.tsx`）。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。只跟随关闭（`onClose`）不够：点击打开之后焦点不在任何控件上，Escape 关不掉。其余 11 个在 M4、M6、M7 的交接。
- **P5 重建的 `CustomSelect`**（`web/packages/ui/src/dropdowns/custom-select.tsx`，改用 `Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M3 页面上的 9 个调用方：`workspace/create-workspace-form.tsx`（在 M2 已有的 `/create-workspace` 页上）、`workspace/{invite-modal/fields,settings/invitations-list-item,settings/member-columns,settings/workspace-details}.tsx`、`project/{add-project-members-modal,form,settings/member-columns}.tsx`、`projects/create/attributes.tsx`。
- **`CustomSearchSelect` 在 M3 的调用方**（只有 Headless UI 的一份打开状态，清空搜索不再交出 `null`，`onOpen` 每次打开调一次）：`navigation/project-header.tsx`、`project/{member-select,add-project-members-modal}.tsx`，以及项目页头上的 `BreadcrumbNavigationSearchDropdown`（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`，它的 `onOpen`）。
- **复制到剪贴板**：P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M3 的 3 处调用没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝：`workspace/sidebar/projects-list.tsx`、`project/card.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`。加上处理，经 `t()` 提示失败，像 `api-token/modal/generated-token-details.tsx`。
- **关闭条件**：M3 的 review 逐项写明：`CustomSearchSelect` 能用 Tab 到达、用键盘打开；`member-options` 的列表在按钮旁展开；列出的调用方在浏览器中核对过；3 处复制都处理失败。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M3 的各行；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
````

````file docs/v0/M4-issue-core/handoffs/M2-closeout.md
---
status: open
from: M2/closeout
to: M4
created: 2026-09-28
---

# M2 交给 M4：命令行的只投递 River 客户端、游标与请求体检查、清理任务、编辑器和下拉框

M2（账户认证）做完了注册、登录、会话、PAT、`nerve users` 命令和第一个 River 定时任务（[M2 设计](../../M2-auth/M2-design.md)）。下面这些是 M2 定下的约定第一次落到工作项上的地方，或 M2 改到而要由 M4 接上的代码。每一节来自 M2 设计 13.2 中接收者含 M4 的一行（多个 M 的行只取 M4 的部分），另有 M0-P6 交接剩下的一项（第 13 节）和收尾在 13.2 之外找到的几项：第 2 节的游标不签名，第 4 节的生成器遇到不支持的写法（M2 设计 §16）和第一个 `date` 字段，第 6 节的定时任务，第 8 节的 `worker-src`、`frame-src`。

## 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）

- **负责人的要求**：2026-09-26 批准把它推迟到 M4 时说，"做好记录，以后别漏了"。M4 用不到它，也要把这一节原样转进下一个 M 的交接，不能随本交接关闭而丢掉。
- **做什么**：第一个"用例会投递 River 任务、又能从命令行触发"的命令加入时，`platform/jobs` 提供一个只投递的客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`），`bootstrap` 的命令组合把它交给这些用例。
- **为什么**：命令行的组合只有连接池和 `identity` 的管理用例，没有 River 客户端（`server/internal/bootstrap/users.go:22-26`，M2 设计 3.17）。服务用的客户端（`server/internal/platform/jobs/jobs.go:70` 的 `New`）配队列、启动 worker，命令行不能用它。用例在事务里用 `InsertTx` 投递（M2 设计 3.15）；命令行上没有客户端时，任务要么没投递，要么用例在运行中失败。M2 的命令都不投递任务，所以推迟（M2 设计 3.15、3.17、13.2）。
- **可能早于 M4**：M3 给停用加端口（[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 6 节），`nerve users deactivate` 也走这个用例；端口的实现投递任务时，这一项提前到 M3。
- **建议的守法**：投递任务的端口在用例的构造参数里必填，命令的组合漏了它，在构造时就失败，而不是运行到一半；`server/internal/bootstrap/users_test.go` 这样的组合测试覆盖每个命令。
- **关闭条件**：M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。

## 2. 列表的游标

- **做什么**：工作项按 `sort_order`、优先级或日期排序，每种排序定义自己的游标载荷，最后以 `id` 保证稳定（M2 设计 3.12）；不复用 PAT 列表的 `(created_at, id)`。
- **代码在哪**：游标的封套在 `server/internal/shared/cursor.go`（`EncodeCursor`、`DecodeCursor`：版本、base64url 编码，解不开是 400，总体设计 3.4）；PAT 列表（`identity` 的 `listApiTokens`）是载荷的范例。
- **游标不签名**：改成另一个合格的位置照样可用。载荷只决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：每种排序有自己的载荷和测试（翻页不重复、不遗漏，排序键相同时按 `id`）；改过的游标只能在调用者看得到的数据里换起点，有测试。

## 3. 自动归档读 `updated_at`

- **做什么**：自动归档按"超过 `archive_in × 30` 天未更新"判断（总体设计 5.5），读 `updated_at`。它由用例的时钟显式写入（M2 设计 3.13，差异清单二·全局），测试用固定时钟（`server/internal/platform/clock/clocktest`）。
- **关闭条件**：写工作项的每个用例都用时钟写 `updated_at`，测试断言它等于固定时钟；自动归档的测试推进固定时钟，不等真实的时间。

## 4. 请求体检查的两处延伸和草稿发布

- **路径的排序**：字段错误的路径现在按字典序排序（`tags[10]` 在 `tags[2]` 之前，`server/internal/platform/httpserver/bodyshape/bodyshape.go:169`），改为按数组下标的数值排序。
- **数的范围**：不限类型的节点（`{}`、开放对象、没有 `items` 的数组）不看数的范围，`1e400` 这类 float64 放不下的数仍然得到解码器笼统的 400，改为在边界上报出。
- 两者都在第一个带数组或开放对象请求体的操作到来时处理（M2/P1 评审）；在那之前客户端不能依赖路径的顺序。
- **生成器**（`server/tools/bodyshapegen`）遇到不支持的 schema 写法时失败并说明原因；遇到它的 M 带着测试扩展生成器，不放过（M2 设计 §16）。
- **第一个 `format: date` 的字段**（工作项的开始、截止日期）：`bodyshape` 的格式检查器按生成的 Go 类型登记，现在只有 `time.Time` 和标准库的 `uuid.UUID`；M2 没有 `date` 字段，生成器遇到 `date` 就失败。M4 选定它的 Go 类型（不能是 `openapi_types.Date`：它所在的包导入 `github.com/google/uuid`），带着测试登记检查器（M2 设计 3.11）。
- **草稿发布**复用同一个结构检查：发布时，草稿的 `payload` 按"创建工作项"的请求 schema 走 `bodyshape` 的校验（M2 设计 3.11），与创建工作项走同一条路（总体设计 5.4，差异清单第四节"草稿发布"）。
- **关闭条件**：两处延伸各有单元测试（`tags[2]` 排在 `tags[10]` 之前；开放对象里的 `1e400` 得到带路径的 400）；同一个坏的请求体，草稿发布与创建工作项给出相同的 400；`date` 的检查器已登记，不合格的日期得到带路径的 400，有测试。

## 5. 60 天物理清理与删除关系图（M4 的部分）

- **做什么**：60 天物理清理（总体设计 5.5）把软删除的 `api_tokens` 纳入（撤销 PAT 就是软删除，M2 设计 4.4）。`issue_activities` 对工作项、评论的外键是 `DO_NOTHING`（照搬时不写 `ON DELETE`），先删工作项会被它挡住：先删除或置空这些引用，或者登记为 `SET NULL` 的差异。
- M4 的每张新表照 M2 设计 3.13 搬 `on_delete`，在 M2 设计 4.7 的图上延伸，写明物理删除时每条外键的去向（M3、M6、M7 各自延伸自己的表）。
- **关闭条件**：清理任务的集成测试覆盖软删除超过 60 天的 PAT 和带操作动态的工作项；M4 设计有延伸的图。

## 6. 定时任务和事件订阅者

- **漏注册 worker 时只有一条 WARN**（收尾找到，M2/P3b 评审第 7 节）：River 的 `Start` 因配置错误失败时，runner 当作数据库不可达一样无限重试，每次一条 WARN "jobs did not start; trying again"（`server/internal/platform/jobs/jobs.go:136`），nerve 照常服务，但没有任务。模块加了定时任务却漏了 worker，这条 WARN 是唯一的迹象。M4 是第一个加业务定时任务的 M（自动归档、60 天清理）：加一个测试，漏注册时失败（M2 的做法是 `bootstrap` 的测试核对 `identity.cleanup_expired_sessions` 已注册，M2 设计 3.15），或者让 `platform/jobs` 在构造时拒绝没有 worker 的定时任务。
- **事件订阅者的写法**：事务内投递用 `InsertTx(ctx, tx, …)`，回滚的事务不留任务（M2 设计 3.15 的 spike）；订阅者怎样写由第一个需要它的 M 定，就是 M4。
- **关闭条件**：漏注册 worker 的变异让测试失败；事件订阅者的写法写进 M4 设计。

## 7. 编辑器的代码分割

- **做什么**：未登录时直接打开需要登录的页面，页面模块先加载了编辑器（tiptap 的 Emoji 节点和 `is-emoji-supported`），包装层才跳到登录页。把编辑器拆出去，到用它的页面才加载。个人设置页的布局挂着命令面板（`ProjectsAppPowerKProvider`，`web/apps/web/app/(all)/settings/profile/layout.tsx`），它在打开之前就建了一个编辑器，一并处理（M2/P5 spec 第 3 节第 9 条）。
- 之后删除 e2e 对 `is-emoji-supported` 的 Chromium 警告（`willReadFrequently`）的预期 `EMOJI_CHECK_WARNING`（`e2e/fixtures/browser.ts`；S2 的深链接和个人设置的故事用它）。
- **关闭条件**：`EMOJI_CHECK_WARNING` 和它的使用都删除，S2 和个人设置的故事仍断言控制台没有别的警告，并通过。

## 8. CSP：编辑器 callout 的默认表情图；`worker-src`、`frame-src`

- **做什么**：`web/packages/editor/src/extensions/callout/utils.ts:19` 的默认表情图来自 `cdn.jsdelivr.net/npm/emoji-datasource-apple`，被页面 CSP 的 `img-src 'self' data: blob:`（`server/internal/platform/webui/csp.go:40`）挡住。改为本站资源或原生表情；核对表情回应（它也用表情选择器，数据由 M3 改为本站提供，[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 10 节）。
- **与 M8 的关系**：[M8 的 M1 收尾交接](../../M8-open-release/handoffs/M1-closeout.md)列了同一个请求；M4 改完之后，那一项对它只剩核对。
- **`worker-src`、`frame-src`**：页面的 CSP 没有写这两条，按 `default-src 'self'` 落回（M2/P4 评审第 4 节）。编辑器或它的扩展要用 `blob:` 的 worker（M4），收集箱或视图要嵌 iframe（M7，[M7 的交接](../../M7-collaboration/handoffs/M2-closeout.md)第 6 节）时，在 `server/internal/platform/webui/csp.go` 只加需要的来源，有测试。
- **关闭条件**：插入 callout、打开表情回应时没有 CSP 违规，页面不请求 `cdn.jsdelivr.net`；M4 的 review 写明编辑器是否用 `blob:` 的 worker，用的话 `csp.go` 只为它加了 `worker-src` 的来源，有测试。

## 9. 命令面板的主题命令

- **做什么**：`web/apps/web/core/components/power-k/config/preferences-commands.ts` 先 `setTheme`，再发 `PATCH /api/v0/me/profile`，失败时不撤回，页面停在 nerve 没有的主题；失败的提示是固定的文案，不按 `code`。P5 起 preferences 页（`web/apps/web/core/components/appearance/theme-switcher.tsx`）在 nerve 应答成功之后才应用主题（M2/P5 spec 2.7）；M4 处理命令面板时照样改。
- **关闭条件**：命令面板的主题命令在应答成功之后才应用，失败时按 `code` 提示，有故事或浏览器核对。

## 10. 下拉框（M4 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。根因和修法见 M2/P5 spec 2.3。

- **`DateDropdown`**（`dropdowns/date.tsx`）要只有一份打开状态的结构，例如 Popover（选中日期时调它的 `close`），连同它在列表和表格单元格里的懒渲染（`ComboDropDown` 的 `renderByDefault`）。P5 保留了两份状态，经 `onClose` 同步：键盘打不开日历；日历里按 Tab 关闭之后两份状态走散，再点另一个下拉框时两个列表同时开着。api-tokens 页的自定义有效期也用它。
- **`CustomSearchSelect` 的 `defaultOpen`**（挂载之后点一次按钮）没有在浏览器中核对，接上富筛选时核对。
- **12 个同样写法的下拉框中 M4 的 7 个**：`dropdowns/{date-range,priority,project/base,state/base}.tsx`、`issues/{issue-detail/label/select/label-select,issue-layouts/properties/label-dropdown,select/base}.tsx`。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。`date-range`、`project/base` 也在 M6 的周期、模块表单里，M6 接上时核对 M4 的修改。
- **P5 重建的 `CustomSelect`**（`Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M4 的 3 个调用方：`automation/auto-archive-automation.tsx`、`core/filters/date-filter-select.tsx`、`issues/peek-overview/header.tsx`。
- **`CustomSearchSelect` 在 M4 的调用方**（一份打开状态，清空搜索不再交出 `null`，`onOpen` 每次打开调一次）：`rich-filters/{add-filters/dropdown,filter-item/root,filter-value-input/select/single,filter-value-input/select/multi}.tsx`。
- **关闭条件**：M4 的 review 逐项写明：`DateDropdown` 能用键盘打开，只有一份状态；7 个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。

## 11. 复制到剪贴板（M4 的 6 处）

- P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M4 的 6 处没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝（路径在 `web/apps/web/core/components/` 下）：`issues/issue-layouts/quick-action-dropdowns/helper.tsx`、`issues/peek-overview/header.tsx`、`issues/issue-detail/issue-activity/helper.tsx`（`.then` 在一个不等待它的 `try` 里）、`issues/issue-detail/links/link-item.tsx`（不等待，照样提示已复制）、`issues/issue-detail-widgets/{relations/helper.tsx,sub-issues/helper.ts}`（`await` 了，调用方不处理）。
- **关闭条件**：6 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。

## 12. stores 按会话分代（规则在总体设计 7.7）和 M2 留下的调用

- **做什么**：M4 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M4 的范围（`d97c513` 上数出）：
  - **注册在沿用的 `router` 上的反应**：`IssueRootStore` 的 `autorun`（`web/apps/web/core/store/issue/root.store.ts:170`），随退役的一代释放。
  - **模块级的 service 实例**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中按调用的接口属于 M4 的 13 处：`IssueService` 2、`WorkItemVersionService` 2、`WorkspaceDraftService` 1；搜索的 4 处，`ProjectService` 的 `projectIssuesSearch` 3 处（`web/apps/web/core/components/core/modals/existing-issues-list-modal.tsx`、`web/apps/web/core/components/issues/parent-issues-list-modal.tsx`、`web/apps/web/core/components/inbox/modals/select-duplicate.tsx`）和 `WorkspaceService` 的 `searchWorkspace` 1 处（`web/apps/web/core/components/power-k/ui/modal/search-menu.tsx`）；编辑器 @提及 的 `WorkspaceService.searchEntity` 4 处（`web/apps/web/core/components/editor/lite-text/editor.tsx`、`web/apps/web/core/components/editor/rich-text/description-input/root.tsx`、`web/apps/web/core/components/issues/issue-modal/components/description-editor.tsx`、`web/apps/web/core/components/inbox/modals/create-modal/issue-description.tsx`）。
  - **`store-context.tsx` 导出的 `rootStore` 的三处同步读取**：`web/apps/web/core/components/issues/issue-layouts/utils.tsx`，以及 `web/apps/web/core/store/command-palette.store.ts`、`web/apps/web/core/store/issue/issue.store.ts` 中只读、不发请求的两处。它们接上新接口之后仍只能同步读取；要发请求就改经自己的 `RootStore`。
  - **SWR 键**：工作项、列表、评论等的 SWR 键带上 `loginId`。
- **M2 留下的调用**：`web/apps/web/core/services/user.service.ts:52` 的 `getUserProfileIssues` 调 Plane 的旧地址，改接新接口或删除。
- **关闭条件**：`IssueRootStore` 的反应随退役的一代释放，有单元测试；上面的 `git grep` 中 M4 的 13 处都已消失；三处同步读取不发请求；`getUserProfileIssues` 改接或删除。

## 13. e2e 的 `clock.ts`（M0-P6 交接剩下的一项）

- **做什么**：可注入的时钟用于"时间流逝"类故事（自动归档），按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/clock.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。nerve 这一侧的时钟怎样在测试中注入（例如 test 配置下的时钟），由 M4 设计定。
- **关闭条件**：`clock.ts` 按这个写法加入，自动归档的故事用它，不等真实的时间。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M4 的各行；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
````

- [ ] **Step 2：路径**：`node $M2TMP/closeout/paths.mjs . docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M2-closeout.md` → 退出码 1，输出（原型）：

  ```
  docs/v0/M3-workspace-project/handoffs/M2-closeout.md: 80 paths, 0 missing
  docs/v0/M4-issue-core/handoffs/M2-closeout.md: 54 paths, 1 missing
    missing: e2e/fixtures/clock.ts
  ```

  写出的仓库路径和展开的 `{…}` 都存在；`plane/` 下的是 Plane 的源码，不核对。唯一缺的 `e2e/fixtures/clock.ts` 是交接要 M4 新建的文件（第 13 节，M0-P6 的延伸），应当不存在。

- [ ] **Step 3：13.2 的覆盖**：`node $M2TMP/closeout/rows.mjs . M3 M4` 按 spec 3.1 的表，列出 13.2 中接收者含 M3、M4 的每一行落在哪个文件的哪一节 → `M3: 16 rows, 0 without a section`、`M4: 13 rows, 0 without a section`（"周期下拉框的取数"一行按 spec 第 4 节 D5 在 M6）。

- [ ] **Step 4：守卫**（固定节奏第 3 步）。

- [ ] **Step 5：提交**（`docs(M2/closeout): the handoffs to M3 and M4`）

### Task 6: 给 M5–M8 的交接；已有交接中过时的两处

spec 3.1、2.4。M2 设计 13.2 中接收者含 M5、M6、M7、M8 的每一行，拆成各自的部分；M0-P2 交接剩下的接口调用日志（M8）、M0-P6 交接剩下的 `storage.ts`（M5）和 `webhook.ts`（M8）；13.2 之外找到的遗留：§16 中写明"M8 实测"的四项、River 在数据库恢复之后的重新启动（M8）、`@nerve/services` 的整包删除和地址规范化的去处、守卫规则 `plane-user-assets` 的收窄（M5）；M7 的最近访问的接口放在它自己的模块文件、M7 的表在删除关系图上的延伸（M2 设计 3.12、3.14，收尾重扫 M2 设计找到）、CSP 的 `frame-src`（指向 M4 的交接）。"周期下拉框的取数"按 spec 第 4 节 D5 放在 M6。

M1 收尾给 M3–M8 的交接都把 `eslint(no-unneeded-ternary)` 列为可以集中清掉的一类规则，M2/P5 已把它全仓清零（M2/P5 评审第 2 节第 4 行）；M1-P3 给 M5 的交接说 `EFileAssetType` 剩 8 种，M2/P4 删掉了其中的 `USER_AVATAR`、`USER_COVER`（`cb2487c9`）。这两处改成现在的事实。

**文件：** `docs/v0/M5-files/handoffs/M2-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md`、`docs/v0/M7-collaboration/handoffs/M2-closeout.md`、`docs/v0/M8-open-release/handoffs/M2-closeout.md`（新文件）；`docs/v0/M{3,4,5,6,7,8}-*/handoffs/M1-closeout.md`、`docs/v0/M5-files/handoffs/M1-P3-trim-platform.md`

- [ ] **Step 1：应用块**

````file docs/v0/M5-files/handoffs/M2-closeout.md
---
status: open
from: M2/closeout
to: M5
created: 2026-09-28
---

# M2 交给 M5：头像和封面、上传与按路由的中间件、删除 `@nerve/services`

M2（账户认证）的接口里，头像、封面是可为 `null` 的地址，M2 没有文件存储（[M2 设计](../../M2-auth/M2-design.md) 3.2）。下面这些由 M5 接上。第 1、2 节来自 M2 设计 13.2 给 M5 的两行，第 3 节来自 M2/P5 spec 第 5 节"不在 P5 范围内"，第 4 节是 13.2"M3 及以后有 stores 的 M"一行中 M5 的部分，第 5 节是 M0-P6 交接剩下的一项。

## 1. 头像和封面

- **做什么**：加入 `users.avatar_asset_id`、`cover_image_asset_id`；`User.avatar_url`、`cover_image_url` 开始返回签名地址；按新的上传协议加回 general 页和新手引导资料步骤的上传控件（M2 设计 3.2）。
- **迁移的范例**（M2 设计 3.14，总体设计 5.6）：先在 M5 的模块里建 `file_assets`（`<v>_<模块>_file_assets.sql`），再写 `<v+1>_identity_users_avatar_asset.sql` 给 `users` 加列和外键；后者归 `identity` 的 sqlc 条目（`server/sqlc.yaml`）。`TestSQLCSchemaScope` 核对 `ALTER TABLE` 的所有者，它的用例里就有这个文件名（`server/internal/archtest/sqlc_cases_test.go:30-32`）。
- **M2 做了什么**：建 `users` 时不建这两列（差异清单二·按表），接口的两个地址一直是 `null`。M2/P4 删掉了头像、封面的上传控件和 Plane 的上传路径（`cb2487c9`），`EFileAssetType`（`web/packages/types/src/enums.ts:22-29`）的 `USER_AVATAR`、`USER_COVER` 随之删除，现在是 6 种。关键词规则 `plane-user-assets`（`tools/keywords.json`）挡住 Plane 的 `/api/assets/v2/user-assets/`、`/api/users/file-assets/`、`UserImageUploadModal` 和 `USER_AVATAR`、`USER_COVER`（M2 设计 7.9）。
- **加回时**：资源类型按 M5 的协议定。沿用 `USER_AVATAR`、`USER_COVER` 这两个名字时，从规则的 `content` 中去掉 `\bUSER_(?:AVATAR|COVER)\b` 一支，`samples` 随之改；Plane 的两个地址和弹窗名仍然挡住。
- **关闭条件**：两列和外键照上面的两个迁移加入，差异清单二·按表更新；两个地址返回签名地址；两处上传控件用新协议；`node tools/keywords.mjs` 通过，`plane-user-assets` 仍挡住 Plane 的地址。

## 2. 上传与按路由的中间件

- **做什么**：`/api/v0` 的每个操作都经过模块级的 `Middlewares`：请求体上限 `server.max_body_bytes`（默认 1 MiB）和请求期限 `server.request_timeout`（默认 15 秒；默认值在 `server/configs/config.yaml:15,17`，交给平台的字段在 `server/internal/platform/httpserver/api.go:42-43`），它们会让 `/api/v0` 下的上传失败。M5 在平台加按操作的放宽设置，或者把上传放在 `/api/v0` 之外（例如直传对象存储），并按 M2 设计 3.6 的整程序测试处理：写进接口描述，或在设计中说明（M2 设计控制者复核 m7）。
- **`file_size_limit` 的执行**：名称 M2 已定（`api/modules/instance.yaml:106`，前端 `web/apps/web/core/hooks/use-file-size.ts` 读它），值来自配置；上传时由 M5 执行。
- **CSP**：`img-src`、`connect-src`（`server/internal/platform/webui/csp.go:40,42`）加上存储的来源。只有对象存储这种本来就在外部的来源才加（M2 设计 8.3）。
- **与 [M0 加固的交接](M0-hardening-http-timeouts.md)的关系**：那一项管连接上的读写时限（`http.ResponseController`），这一项管按路由的请求体上限和请求期限，两者都要放宽。
- **关闭条件**：大于 1 MiB、耗时超过 15 秒的上传都成功（测试）；超过 `file_size_limit` 的被拒绝；整程序测试（`server/internal/bootstrap/contract_test.go`）仍通过；CSP 只为存储放开来源，有测试。

## 3. 删除 `@nerve/services`

- **做什么**：M5 重写文件的接口调用之后，删除整个 `web/packages/services`（总体设计 7.1，[前端改动清单](../../frontend-changes.md) 3.1 的最后一行）。
- **现在还剩**：上传文件的元数据工具（`web/packages/services/src/file/helper.ts` 的 `getFileMetaDataForUpload`、`generateFileUploadPayload`，调用方是 `web/apps/web/core/services/file.service.ts` 和 `web/apps/web/core/services/issue/issue_attachment.service.ts`），以及地址规范化（`web/packages/services/src/helpers/url.ts` 的 `normalizeAPIRequestURL`，带单元测试）。
- **注意**：地址规范化的调用方是 web 的 axios 基类 `web/apps/web/core/services/api.service.ts`，还没对接新接口的领域经它调 Plane 的旧地址（M5 之后是 M6–M8 的领域）。删包时它还有调用方，就把规范化连同测试挪进 web（例如 `web/apps/web/core/services/` 下），或者证明那时已没有旧地址要补结尾的 `/`。
- **守卫**：规则 `services-files`（`tools/keywords.json`，只允许包里现在的几个文件）随包删除；包不存在之后，这条规则删掉，或改为挡住包名重新出现。
- **关闭条件**：`web/packages/services` 不存在，任何 `package.json` 都不依赖 `@nerve/services`；`make knip`、`make lint-web`、`make test-web` 通过；前端改动清单 3.1 的那一行改为已完成。

## 4. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M5 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M5 的范围（`d97c513` 上数出）：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `FileService` 8 处。它们现在经 web 的 axios 基类调 Plane 的旧地址。
- **关闭条件**：这 8 处都已消失；M5 新加的 SWR 键带 `loginId`。

## 5. e2e 的 `storage.ts`（M0-P6 交接剩下的一项）

- **做什么**：检查文件是否真的写入本地存储，按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/storage.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。
- **关闭条件**：`storage.ts` 照这个写法加入，上传的故事用它核对存储里的文件。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M5 的各行；[M2/P5 spec](../../M2-auth/specs/P5-web-account.md) 第 5 节；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
````

````file docs/v0/M6-cycles-modules/handoffs/M2-closeout.md
---
status: open
from: M2/closeout
to: M6
created: 2026-09-28
---

# M2 交给 M6：迭代的跨模块写入、删除关系图、stores 按会话分代、周期和模块的下拉框

下面这些来自 [M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M6 的各行（多个 M 的行只取 M6 的部分）。第 4 节在 13.2 里写给 M4，收尾改给 M6：它取的是周期，周期的接口在 M6（[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 4 节 D5）。

## 1. 迭代跨模块的写入和查询

- **做什么**：迭代（`cycles`、`cycle_issues`）跨 `planning` 与工作项模块的写入，用端口和共享事务；必须联表的查询，事先列为 `TestSQLCSchemaScope` 的例外并写明理由，或者用端口拆开（M2 设计 3.14）。
- **为什么**：sqlc 按模块限定 `schema`，一个模块的查询只能碰本模块的表（总体设计 6.3）；`TestSQLCSchemaScope`（`server/internal/archtest/sqlc_test.go`）核对这份配置。
- **关闭条件**：M6 设计列出每个跨模块的写入和查询，以及它走端口还是例外；例外在 `TestSQLCSchemaScope` 中写明理由。

## 2. 物理删除与跨模块外键的关系图（M6 的部分）

- **做什么**：M6 的每张新表照 M2 设计 3.13 搬 Django 模型的 `on_delete`，在 M2 设计 4.7 的图上延伸，写明物理删除时每条外键的去向（M3、M4、M7 各自延伸自己的表）。
- **为什么**：Plane `cycle.py:65-68` 的迭代负责人是 `CASCADE`：物理删除一个账户会连带删除迭代。M2 只停用、不删除账户。
- **关闭条件**：M6 设计有延伸后的图；迭代负责人的 `CASCADE` 有结论（照搬，或改为 `SET NULL`、`RESTRICT` 并登记差异）。

## 3. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M6 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M6 的范围（`d97c513` 上数出）：
  - **注册在沿用的 `router` 上的反应**：`web/apps/web/core/store/cycle_filter.store.ts:67`、`web/apps/web/core/store/module_filter.store.ts:74` 的 `reaction`，随退役的一代释放。
  - **模块级的 service 实例**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `CycleService` 2 处。
- **关闭条件**：两个反应随退役的一代释放，有单元测试；这 2 处都已消失；M6 新加的 SWR 键带 `loginId`。

## 4. 周期下拉框的取数

- **做什么**：`web/apps/web/core/components/dropdowns/cycle/cycle-options.tsx` 原来从不取周期（`!cycleIds` 对一个总是数组的值），M2/P5 清 lint 时改为打开时这个项目的周期还没取过（`getProjectCycleIds` 为 `null`）就取。同一处改动还带来三点（M2/P5 Task 10），M6 接上周期时核对：
  - 列表开着时 `workspaceSlug` 或 `projectId` 变了，打开时的 effect 再运行一次（重新聚焦，新项目没取过就取）；
  - 第一次取数还在途中时关上再打开，会再发一次 `GET`，两个回答都写 `cycleMap`、`fetchedMap`，没有先后的保护，旧的回答可能最后到；
  - `status` 是非字符串的假值（`false`、`0`、`NaN`）时，`status?.toLowerCase()` 现在抛出（原来返回 `true`）；这样的值不在它的类型 `TCycleGroups` 里。
- **关闭条件**：M6 的 review 写明三点各自的结论（保留、修正并有测试）；并发的两次取数不会让旧的回答覆盖新的。

## 5. 下拉框（M6 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。根因和修法见 M2/P5 spec 2.3。

- **12 个同样写法的下拉框中 M6 的 2 个**：`dropdowns/cycle/cycle-options.tsx`、`dropdowns/module/module-options.tsx`。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。
- **P5 重建的 `CustomSelect`**（`Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M6 的 3 个调用方：`modules/{module-status-dropdown,progress-sidebar/root,select/status}.tsx`。
- **`BreadcrumbNavigationSearchDropdown`**（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`，P5 重建的 `CustomSearchSelect`：`onOpen` 每次打开调一次）在周期和模块的页头上。
- M4 改好的 `dropdowns/date-range.tsx`、`dropdowns/project/base.tsx` 也在周期、模块的表单里（`cycles/form.tsx`、`modules/form.tsx`），接上时核对。
- **关闭条件**：M6 的 review 逐项写明：两个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。

## 6. 复制到剪贴板（M6 的 3 处）

- P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M6 的 3 处没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝：`web/apps/web/core/components/cycles/quick-actions.tsx`、`web/apps/web/core/components/modules/quick-actions.tsx`、`web/apps/web/core/components/modules/links/list-item.tsx`。
- **关闭条件**：3 处都处理失败，经 `t()` 提示，像 `web/apps/web/core/components/api-token/modal/generated-token-details.tsx`。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M6 的各行和"周期下拉框的取数"一行；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
````

````file docs/v0/M7-collaboration/handoffs/M2-closeout.md
---
status: open
from: M2/closeout
to: M7
created: 2026-09-28
---

# M2 交给 M7：保存视图的游标、stores 按会话分代、收集箱和视图的下拉框、删除关系图

下面这些来自 [M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M7 的各行（多个 M 的行只取 M7 的部分）。另有收尾找到的三项：第 2 节中最近访问的接口放在哪个文件（M2 设计 3.12），第 5 节的删除关系图（M2 设计 3.14 把它交给 M3、M4、M6、M7，13.2 那一行漏了 M7），第 6 节的 CSP（M2/P4 评审第 4 节）。

## 1. 保存视图的列表：每种排序一个游标载荷

- **做什么**：保存视图的列表有多种排序，照 M2 设计 3.12 为每种排序定义游标载荷，最后以 `id` 保证稳定；不复用 PAT 列表的 `(created_at, id)`。跨模块的联表同 M6：事先列为 `TestSQLCSchemaScope` 的例外并写明理由，或者用端口拆开（M2 设计 3.14）。
- **代码在哪**：游标的封套在 `server/internal/shared/cursor.go`（`EncodeCursor`、`DecodeCursor`，总体设计 3.4）。
- **游标不签名**：改成另一个合格的位置照样可用。载荷只决定从哪里接着读；游标内容会影响能看到什么的列表（视图会带筛选条件），要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：每种排序有自己的载荷和测试（翻页不重复、不遗漏）；改过的游标只能在调用者看得到的数据里换起点，有测试；跨模块的联表有结论。

## 2. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M7 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M7 的范围（`d97c513` 上数出）：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `WorkspaceNotificationService`、`IntakeWorkItemVersionService` 各 1 处，以及最近访问的 `WorkspaceService.fetchWorkspaceRecents` 1 处（`web/apps/web/core/components/home/widgets/recents/index.tsx`）。
- **最近访问的接口**：`/me/recent-visits` 这样的路径属于最近访问自己的模块文件，不放进 `api/modules/identity.yaml`（M2 设计 3.12 的组织规则：一个路径只属于一个模块文件）。
- **关闭条件**：这 3 处都已消失；M7 新加的 SWR 键带 `loginId`；最近访问的操作在它自己的模块文件里。

## 3. 下拉框（M7 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。根因和修法见 M2/P5 spec 2.3。

- **12 个同样写法的下拉框中 M7 的 2 个**：`dropdowns/intake-state/base.tsx`（收集箱），`web/packages/ui/src/dropdown/single-select.tsx`（`Dropdown`，经 `dropdowns/layout.tsx` 用在 `views/form.tsx`）。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。
- **P5 重建的 `CustomSelect`**（`Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M7 的调用方：`workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx`。
- **`BreadcrumbNavigationSearchDropdown`**（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`，`onOpen` 每次打开调一次）在视图的页头上。
- **关闭条件**：M7 的 review 逐项写明：两个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。

## 4. 复制到剪贴板（M7 的 4 处）

- P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M7 的 4 处没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝（路径在 `web/apps/web/core/components/` 下）：`inbox/content/inbox-issue-header.tsx`、`views/quick-actions.tsx`、`workspace/views/{quick-action,default-view-quick-action}.tsx`。
- **关闭条件**：4 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。

## 5. 物理删除与跨模块外键的关系图（M7 的部分）

- **做什么**：M7 的每张新表（评论、通知、保存视图、收集箱等）照 M2 设计 3.13 搬 Django 模型的 `on_delete`，在 M2 设计 4.7 的图上延伸，写明物理删除时每条指向 `users` 和工作项的外键的去向（M3、M4、M6 各自延伸自己的表）。跨模块的联表见第 1 节。
- **为什么**：M2 设计 3.14 把"跨模块的联表查询、物理删除时的外键链由各 M 的设计逐个写明"交给 M3、M4、M6、M7；13.2 的关系图一行只写了 M3、M4、M6。
- **关闭条件**：M7 设计有延伸后的图，每条指向 `users`、工作项的外键写明去向。

## 6. CSP 的 `frame-src`

- 见 [M4 的交接](../../M4-issue-core/handoffs/M2-closeout.md)第 8 节：页面的 CSP 没有 `frame-src`，落回 `default-src 'self'`。
- **关闭条件**：M7 的 review 写明收集箱或视图是否嵌入 iframe；嵌入的话，`server/internal/platform/webui/csp.go` 只为它加那个来源，有测试。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M7 的各行、3.12、3.14；[M2/P4 评审](../../M2-auth/reviews/P4-web-auth-review.md)第 4 节；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
````

````file docs/v0/M8-open-release/handoffs/M2-closeout.md
---
status: open
from: M2/closeout
to: M8
created: 2026-09-28
---

# M2 交给 M8：接口调用日志、性能和内存的实测、部署、接口文档页

下面这些来自 [M2 设计](../../M2-auth/M2-design.md) 13.2 给 M8 的一行和"M3 及以后有 stores 的 M"一行中 M8 的部分；M0-P2、M0-P6 两份交接剩下的各一项（第 1、5 节）；以及收尾在 13.2 之外找到的几项：M2 设计 §16 中写明"M8 实测"的四项（第 2 节），River 在数据库恢复之后的重新启动（第 3 节）。

## 1. 接口调用日志挂在限流之后

- **做什么**：接口调用日志（总体设计 9.2 的 M8；差异清单第四节：记录所有写请求，不区分令牌类型，存储前去掉敏感请求头）作为按路由的中间件，挂在限流之后（M2 设计 3.6，总体设计 6.4）。
- **代码在哪**：`server/internal/platform/httpserver/api.go:121` 的 `Middlewares` 按顺序组装按路由的中间件；`httpserver.RequestID(ctx)`（`server/internal/platform/httpserver/middleware.go:62`）已导出，日志可以带上请求 ID。
- **M2 做了什么**：认证（P1）、失败闸门和限流（P2）都挂好了；这一项是 [M0-P2 交接](../../M2-auth/handoffs/M0-P2-platform-notes.md)第 2 条剩下的最后一件。
- **关闭条件**：中间件在限流之后，`API.Middlewares` 的顺序测试核对；被限流、认证失败的请求在它之前就结束、不进日志，M8 设计写明这与差异清单的"记录所有写请求"怎样对上。

## 2. 性能和内存的实测

M8 做 Go 进程的内存实测（总体设计 9.2：空闲时低于 50 MB）时，一并测下面几项。它们都是 M2 设计 §16 的应对里写明"M8 实测"的风险：

- **argon2 的内存**：并发上限 4 个（约 76 MiB，M2 设计 3.8）下的峰值。
- **argon2 的 CPU**：生产参数下登录、注册、修改密码的耗时；持续集成没有测过（测试环境用 m = 64 KiB、t = 1，M2/P1 评审第 7 节）。
- **常见密码名单的内存**：`server/internal/modules/identity/domain/common_passwords.txt`（33,904 行、278,161 字节）加载之后占用的内存。
- **失败闸门的并发**：认证之前的失败闸门按 IP 计数，先预留、后退回；同一个出口 IP 后同时在认证中的请求不能超过桶里当时剩下的单位（M2 设计 3.10）。实测同一 IP 的并发，看有效的调用方会不会得到 429。
- **请求体结构检查的开销**：请求体被解析两次（结构检查一次、生成代码解码一次，上限 1 MiB）。
- **每个带令牌的请求多一次数据库查询**：访问令牌查它的会话行，PAT 查它的哈希（M2 设计 6.4、§16）。
- **与负责人的事项的关系**：调高 argon2 参数之后，登录耗时能区分休眠的账户和不存在的邮箱，是否加缓解由负责人以后决定，记在[总体设计第 10 节](../../v0-design.md)。它不是 M8 的任务；M8 的 argon2 数字供负责人参考。
- **关闭条件**：M8 的 review 有每一项的数字、测法和结论；超出预期的写明处理（调参、改默认值，或登记风险）。

## 3. 部署

- **镜像设 `NERVE_ENV=prod`**（M2 设计决策点 2 的缓解，6.1）：不设时 nerve 按 dev 的默认值运行，注册开放，签名密钥是临时的（README"部署"一节的第一条）。
- **容器的停止宽限期**：默认配置下停机最坏约 36 秒（HTTP 20、任务 10+1、连接池 5，M2/P3b spec 第 3 节第 7 条），超过 Docker 默认的 10 秒。部署文件要设 `stop_grace_period`（或调小这几个期限），否则进程在收尾中被 SIGKILL。
- **数据库恢复之后 River 能重新启动**：M2 只读过代码（M2/P3b spec 附录 A 的 E1）并用假客户端测了重试；真实的客户端只测了"数据库不可达时启动失败、`Stop` 立即结束重试"（M2/P3b 评审第 7 节）。部署核对中停一次数据库、再恢复，看定时任务恢复运行（清理任务的日志，或 `river_job` 的新行）。
- **关闭条件**：镜像的环境里有 `NERVE_ENV=prod`；部署文件的停止宽限期不短于停机的最坏时间（或写明调小了哪些期限）；数据库重启的核对结果写进 M8 的 review。

## 4. 对外接口文档页不从 CDN 加载脚本

- **做什么**：对外接口文档页的脚本和样式从本站提供，不从 CDN 加载（M2 设计 8.3：自托管的 Nerve 在没有外网时也能用，不把使用情况告诉第三方）。同一个页面的其他事项见 [M0-P3 的交接](M0-P3-public-api-docs.md)。
- **关闭条件**：文档页不请求第三方的来源；页面带 CSP 时没有违规。

## 5. e2e 的 `webhook.ts`（M0-P6 交接剩下的一项）

- **做什么**：本地的 Webhook 接收端，按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/webhook.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。
- **关闭条件**：`webhook.ts` 照这个写法加入，Webhook 的故事用它核对收到的负载和签名。

## 6. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：Webhook 设置页和接口调用日志页的 stores（例如 `web/apps/web/core/store/workspace/webhook.store.ts`）、services 照 [总体设计 7.7](../../v0-design.md) 写。
- **关闭条件**：M8 领域没有模块级的带令牌 service 或客户端（`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中没有 M8 的）；M8 新加的 SWR 键带 `loginId`。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M8 的各行和 §16；[M0-P2 的交接](../../M2-auth/handoffs/M0-P2-platform-notes.md)；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
````

````old docs/v0/M3-workspace-project/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
````

````new docs/v0/M3-workspace-project/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````

````old docs/v0/M4-issue-core/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
````

````new docs/v0/M4-issue-core/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````

````old docs/v0/M5-files/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
````

````new docs/v0/M5-files/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````

````old docs/v0/M6-cycles-modules/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
````

````new docs/v0/M6-cycles-modules/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````

````old docs/v0/M7-collaboration/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
````

````new docs/v0/M7-collaboration/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````

````old docs/v0/M8-open-release/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
````

````new docs/v0/M8-open-release/handoffs/M1-closeout.md
`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````

````old docs/v0/M5-files/handoffs/M1-P3-trim-platform.md
`PROJECT_COVER`、`USER_AVATAR`、`USER_COVER`、`WORKSPACE_LOGO`。
````

````new docs/v0/M5-files/handoffs/M1-P3-trim-platform.md
`PROJECT_COVER`、`USER_AVATAR`、`USER_COVER`、`WORKSPACE_LOGO`。M2/P4 删掉头像、封面的上传时删除了 `USER_AVATAR`、`USER_COVER`，现在是 6 种；M5 加回头像、封面时见 [M2 收尾的交接](M2-closeout.md)第 1 节。
````

````old docs/v0/M5-files/handoffs/M1-P3-trim-platform.md
- 接口接受的资源类型正好是 `EFileAssetType` 剩下的 8 种，前端改用生成的类型；
````

````new docs/v0/M5-files/handoffs/M1-P3-trim-platform.md
- 接口接受的资源类型正好是 `EFileAssetType` 剩下的 6 种，加上 M5 为头像、封面定的类型，前端改用生成的类型；
````

- [ ] **Step 2：路径**：`node $M2TMP/closeout/paths.mjs . docs/v0/M5-files/handoffs/M2-closeout.md docs/v0/M6-cycles-modules/handoffs/M2-closeout.md docs/v0/M7-collaboration/handoffs/M2-closeout.md docs/v0/M8-open-release/handoffs/M2-closeout.md` → 退出码 1，输出（原型）：

  ```
  docs/v0/M5-files/handoffs/M2-closeout.md: 21 paths, 1 missing
    missing: e2e/fixtures/storage.ts
  docs/v0/M6-cycles-modules/handoffs/M2-closeout.md: 20 paths, 0 missing
  docs/v0/M7-collaboration/handoffs/M2-closeout.md: 18 paths, 0 missing
  docs/v0/M8-open-release/handoffs/M2-closeout.md: 6 paths, 1 missing
    missing: e2e/fixtures/webhook.ts
  ```

  缺的两个是交接要接收 M 新建的 fixture（M5 第 5 节、M8 第 5 节），应当不存在；别的路径都存在。

- [ ] **Step 3：13.2 的覆盖**：`node $M2TMP/closeout/rows.mjs . M5 M6 M7 M8` → `M5: 3 rows, 0 without a section`、`M6: 6 rows, 0 without a section`、`M7: 5 rows, 0 without a section`（关系图一行加上了 M7，第 5 节）、`M8: 2 rows, 0 without a section`。

- [ ] **Step 4：守卫**（固定节奏第 3 步）。

- [ ] **Step 5：提交**（`docs(M2/closeout): the handoffs to M5–M8; two facts in older handoffs brought up to date`）

### Task 7: 关闭 M2 收到的 10 份交接

spec 2.1。每份交接末尾加 `## 处理结果（M2/收尾）`：逐条写结论（哪个 Phase 做的，或转交给谁），并给出收尾在 `d97c513` 上的核对；`status` 改为 `closed`。两份还是 `open` 的（M0-P2、M0-P6）剩下的事项不属于 M2，转交到 T5、T6 写的交接。M1-P3 的"不再读的用户字段、不再调用的地址"按原文要求再核对一次。

**文件：** `docs/v0/M2-auth/handoffs/` 下的 10 个文件

- [ ] **Step 1：应用块**

````old docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md
status: done
````

````new docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md
来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.10。
````

````new docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md
来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.10。

## 处理结果（M2/收尾）

两条都在 M2/P1 完成，收尾在 `d97c513` 上核对仍然成立：

1. **cgo**：`Makefile` 的 `SQLC` 是 `CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc`；`make gen-check` 重新生成 sqlc 的代码，没有差异。
2. **PG 17 的解析器**：五个迁移（`server/migrations/sql/00001`–`00005`）都没有 `uuidv7()`（`git grep -n -i uuidv7 -- server/migrations/sql` 没有输出）；sqlc 解析 `identity` 的四个（`00001`–`00004`，`server/sqlc.yaml` 的 `schema`），`make gen-check` 没有差异。River 的 `00005` 是原样导出，不在 sqlc 的 `schema` 里。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
status: open
````

````new docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。

## 处理结果（M2/收尾）

逐条的结论：

1. **TxManager**：M2/P1 完成。
2. **按路由挂载**：认证（M2/P1）、失败闸门和限流（M2/P2）完成。接口调用日志不属于 M2，转交 M8：[M8 的交接](../../M8-open-release/handoffs/M2-closeout.md)第 1 节，挂在限流之后。
3. **`RequestID`**：M2/P1 完成（`server/internal/platform/httpserver/middleware.go:62`）。
4. **`LogValue`**：M2/P1 完成。
5. **期限、River 与停机顺序**：请求期限在 M2/P1；River、停机顺序、连接池关闭的时限和 River 的迁移在 M2/P3b。停机最坏约 36 秒，部署文件的停止宽限期转交 M8：[M8 的交接](../../M8-open-release/handoffs/M2-closeout.md)第 3 节。
6. **archtest**：M2/P1 完成（`server/internal/archtest/rules_test.go:40,42`）。M0 设计 3.7 的规则原文到收尾才同步，是 P1 的漏项（收尾 spec 2.2）。
7. **迁移与就绪检查**：M2/P1 完成（README"部署"一节的"迁移"一条）。
8. **三个小问题**：M2/P1 完成。

全部有结论，状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md
status: done
````

````new docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md
来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md
来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。

## 处理结果（M2/收尾）

五条都已在 M2/P1、P3a 完成，收尾在 `d97c513` 上核对：

1. **生成配置和 google/uuid 的守卫**：`TestNerveBinaryLinksNoBannedModule`、`TestGeneratedCodeUsesTheStandardUUID` 随 `make test` 通过；`make gen-check` 没有差异。
2. **错误映射**：三个出口由 identity 的 handler 测试和整程序测试（`server/internal/bootstrap/contract_test.go` 的 `TestBodiesThatBreakTheStructureAnswer400`、`TestParametersThatDoNotBindAnswer400`）覆盖，随 `make test` 通过。
3. **安全声明**：`apitest` 的写法检查随 `make test` 通过。
4. **模块入口**：`Register(router, api)`、`PublicOperations()`；`TestPublicOperationsAreTheContractsPublicOperations` 通过。
5. **组织规则**：照旧。接口描述按模块一个文件，不受约 400 行的限制（`api/modules/identity.yaml` 623 行），收尾写进总体设计 6.3 的规则 5。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md
status: done
````

````new docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md
来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.10。
````

````new docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md
来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.10。

## 处理结果（M2/收尾）

两条在 M2/P1 定下，M2/P3a 的 `00004_identity_api_tokens.sql` 照做，收尾在 `d97c513` 上核对：

1. **约定**：差异清单二·全局登记了全部约定；`server/migrations/schema_test.go` 核对约束名和 CHECK，随 `make test` 通过；五个迁移都没有 `DEFERRABLE` 和 `*_like` 索引（`git grep -n -i -E 'DEFERRABLE|_like' -- server/migrations/sql` 没有输出）。River 的表（`00005_river_main_v2_to_v7.sql`）是原样导出，名字随 River，不按这些约定（差异清单二·按表"River 的表"一行）。
2. **主键与 ID**：照旧，没有默认值的 `id uuid`，由应用生成。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
status: done
````

````new docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

两节都在 M2/P2、P4 完成，收尾在 `d97c513` 上核对：

- **前端调用的接口**：未登录的页面只请求 `GET /api/v0/instance`（S2 的 `apiRequests`，随 `make e2e` 通过）；关键词规则 `plane-auth-urls`、`plane-user-urls` 挡住 Plane 的认证和实例地址（`node tools/keywords.mjs` 没有命中）；同源部署：web 的客户端都由 `createClient()` 建成、不设 `baseUrl`，axios 基类也不设基础地址（M2/P4 的处理结果）。
- **安全响应头与 CSP**：`server/internal/bootstrap/headers_test.go` 的 `TestOnlyPagesHaveAContentSecurityPolicy` 随 `make test` 通过；S2 核对首页带 CSP、没有违规。分在两层的理由在 [M2/P4 评审](../reviews/P4-web-auth-review.md)第 6 节。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
status: open
````

````new docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
仍未处理，状态保持 `open`：fixture 写法的延伸（M4、M5、M8），由收尾转交给这些 M。

来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
仍未处理，状态保持 `open`：fixture 写法的延伸（M4、M5、M8），由收尾转交给这些 M。

来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

逐节的结论：

- **PAT 对等验收与认证 fixture**：M2/P1–P4 完成（`e2e/fixtures/auth.ts`）。
- **数据库断言**：M2/P1 完成（`e2e/fixtures/assert/`）。
- **S1、S3**：M2/P1、P3a 完成。
- **S2**：M2/P4 完成。
- **端口与停机**：M2/P1、P3b 完成。
- **录像**：M2/P1 定为不录像。
- **fixture 模式的延伸**：不属于 M2，转交：`clock.ts` 给 M4（[M4 的交接](../../M4-issue-core/handoffs/M2-closeout.md)第 13 节），`storage.ts` 给 M5（[M5 的交接](../../M5-files/handoffs/M2-closeout.md)第 5 节），`webhook.ts` 给 M8（[M8 的交接](../../M8-open-release/handoffs/M2-closeout.md)第 5 节）。

收尾的头上 `make e2e` 48 个测试通过（收尾 spec 附录 A）。全部有结论，状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
status: done
````

````new docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

关闭条件的两部分在 `d97c513` 上都成立：接口描述中的主题是枚举 `Theme`，只有五个值（`api/modules/identity.yaml`，没有 `custom`，也没有调色板字段）；前端没有 `IUserTheme`（`git grep -n IUserTheme -- web` 没有输出）。状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
status: done
````

````new docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
来源：[M2/P5 spec](../specs/P5-web-account.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
来源：[M2/P5 spec](../specs/P5-web-account.md) 第 7 节。

## 处理结果（M2/收尾）

逐条核对关闭条件（`d97c513`）：

- **CSRF**：`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；登录、注册、退出、修改密码都走令牌管理器（M2/P4）。
- **认证错误**：接口直接返回 problem，前端不读 `/?error_code=…`（关键词规则 `auth-error-code`，M2/P4）。
- **修改登录邮箱**：负责人裁定为 B，`nerve users set-email`（M2/P3b，A16）。
- **实例配置**：`InstanceInfo` 有 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（`api/modules/instance.yaml:83`）；`is_self_managed` 连同新手引导的两步删除（关键词规则 `is-self-managed`，M2/P4）。
- **不再读的用户字段、不再调用的地址**（原文要求收尾再核对一次）：`git grep -n -i -E 'is_password_autoset|last_login_medium|has_marketing_email_consent|billing_address|has_billing_address|property_change|state_change|issue_completed|comment|[^_]mention|provider|notification|email-check|magic|forgot|reset-password|set-password|generate-code|change-email|instance-admin|is_instance_admin|google|github|gitlab|gitea|[^0]/auth/' d97c513 -- api` 没有输出（退出码 1）。接口描述共 15 个操作、12 个路径，都在 `/api/v0/auth/`、`/api/v0/me`、`/api/v0/api-tokens/{token_id}`、`/api/v0/instance`、`/api/v0/timezones` 下。
- **令牌的地址**：`/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}`，结尾都没有 `/`（M2/P3a）。

状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M1-P4-router-native.md
status: done
````

````new docs/v0/M2-auth/handoffs/M1-P4-router-native.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M1-P4-router-native.md
来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
````

````new docs/v0/M2-auth/handoffs/M1-P4-router-native.md
来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

三项在 M2/P4 完成，收尾在 `d97c513` 上核对：`next_path` 的关口是 `@nerve/utils` 的 `isValidNextPath`（`web/packages/utils/src/url.ts`），单元测试 `web/packages/utils/src/next-path.test.ts` 随 `make test-web` 通过，A3 的页面测试打开四个不合格的 `next_path`，登录后都落到 `/create-workspace`（`make e2e`）；`AuthenticationWrapper` 和 401 处理（`web/apps/web/core/lib/auth/auth-middleware.ts`）随令牌管理器重写；"由 M2 决定"的三项在 M2 设计 3.18、7.2、7.7 有结论。状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

````old docs/v0/M2-auth/handoffs/M1-closeout.md
status: done
````

````new docs/v0/M2-auth/handoffs/M1-closeout.md
status: closed
````

````old docs/v0/M2-auth/handoffs/M1-closeout.md
中文下每周第一天的标题和星期名也是中文），C3 核对日历。
````

````new docs/v0/M2-auth/handoffs/M1-closeout.md
中文下每周第一天的标题和星期名也是中文），C3 核对日历。

来源：[M2/P5 spec](../specs/P5-web-account.md) 第 7 节。

## 处理结果（M2/收尾）

三节都在 M2/P5 完成，收尾在 `d97c513` 上核对：`make lint-web` 通过，各包的 oxlint 警告数等于上限（web 452、ui 19、utils 12、i18n 0）；`eslint(no-unneeded-ternary)` 全仓 0 条警告（[M2/P5 评审](../reviews/P5-web-account-review.md)第 2 节第 4 行；Plane 原有的两处禁用注释在 `issue-layouts/utils.tsx`、`issue-layouts/quick-add/root.tsx`）；剩下的 24 行死成员和死 prop 都不是死代码，逐行的理由在 [M2/P5 spec](../specs/P5-web-account.md) 附录 A；主题下拉框在按钮旁展开，由 A9 的页面版本在英文、中文下各核对一次，随 `make e2e` 通过。状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
````

- [ ] **Step 2：没有 `open` 的交接**：`grep -L '^status: closed$' docs/v0/M2-auth/handoffs/*.md` 没有输出；`grep -l -E '^status: (open|done)$' docs/v0/M2-auth/handoffs/*.md` 没有输出。

- [ ] **Step 3：守卫**（固定节奏第 3 步）。

- [ ] **Step 4：提交**（`docs(M2/closeout): the ten handoffs M2 received are closed`）

### Task 8: M2 设计：13.2 指向交接，§16 中负责人的两个事项指向总体设计

spec 2.7、第 4 节 D1、D4。13.2 保留它的行（设计的记录），每一行的接收者一格改为指向交接文件的链接；多个 M 的行链接到每个接收者；"周期下拉框的取数"一行写明收尾改由 M6 接（D5）；"其余 12 个下拉框"一行写明它也落到 M7；关系图一行加上 M7（M2 设计 3.14 把它交给了 M7，13.2 漏了）。§16 第二个 argon2 行和"泄露的刷新令牌可以派生永不过期的 PAT"一行写明负责人的事项记在总体设计第 10 节（T4）。

**文件：** `docs/v0/M2-auth/M2-design.md`

- [ ] **Step 1：应用块**

````old docs/v0/M2-auth/M2-design.md
### 13.2 M2 交给后续 M 的事项（收尾时写成交接）
| 接收 | 事项 |
````

````new docs/v0/M2-auth/M2-design.md
### 13.2 M2 交给后续 M 的事项（收尾时写成交接）
收尾已把每一行写成交接：接收者一格链接到它的交接文件（`docs/v0/M<n>-*/handoffs/M2-closeout.md`），交接是给接收 M 的工作副本，这里保留设计的记录。多个 M 的行拆到各自的文件，每一行落在哪一节见[收尾 spec](specs/closeout.md) 3.1；13.2 之外找到的遗留也在那些文件里（收尾 spec 2.4）。

| 接收 | 事项 |
````

````old docs/v0/M2-auth/M2-design.md
| M3 | **邀请**（
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | **邀请**（
````

````old docs/v0/M2-auth/M2-design.md
| M3 | **登录后的落点与新手引导的取数**
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | **登录后的落点与新手引导的取数**
````

````old docs/v0/M2-auth/M2-design.md
| M3 及以后有 stores 的 M | **stores 按会话分代**
````

````new docs/v0/M2-auth/M2-design.md
| M3 及以后有 stores 的 M：[M3](../M3-workspace-project/handoffs/M2-closeout.md)、[M4](../M4-issue-core/handoffs/M2-closeout.md)、[M5](../M5-files/handoffs/M2-closeout.md)、[M6](../M6-cycles-modules/handoffs/M2-closeout.md)、[M7](../M7-collaboration/handoffs/M2-closeout.md)、[M8](../M8-open-release/handoffs/M2-closeout.md)；规则的全文在[总体设计 7.7](../v0-design.md) | **stores 按会话分代**
````

````old docs/v0/M2-auth/M2-design.md
| M3 | `profiles.last_workspace_id` 是否补外键
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | `profiles.last_workspace_id` 是否补外键
````

````old docs/v0/M2-auth/M2-design.md
| M3 | `workspace_creation_enabled` 的执行
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | `workspace_creation_enabled` 的执行
````

````old docs/v0/M2-auth/M2-design.md
| M3 | **停用的端口**
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | **停用的端口**
````

````old docs/v0/M2-auth/M2-design.md
| M3 | 3.2 的字段规则
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | 3.2 的字段规则
````

````old docs/v0/M2-auth/M2-design.md
| M3 | sqlc 的模块边界
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | sqlc 的模块边界
````

````old docs/v0/M2-auth/M2-design.md
| M3 | `TestSQLCSchemaScope` 的所有者规则
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | `TestSQLCSchemaScope` 的所有者规则
````

````old docs/v0/M2-auth/M2-design.md
| M3、M4、M6 | **物理删除与跨模块外键的关系图**
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md)、[M4](../M4-issue-core/handoffs/M2-closeout.md)、[M6](../M6-cycles-modules/handoffs/M2-closeout.md)、[M7](../M7-collaboration/handoffs/M2-closeout.md)（M7 由收尾加上：3.14 把跨模块的联表和外键链交给 M3、M4、M6、M7，[收尾 spec](specs/closeout.md) 2.4） | **物理删除与跨模块外键的关系图**
````

````old docs/v0/M2-auth/M2-design.md
| M3 | CSP：表情选择器
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | CSP：表情选择器
````

````old docs/v0/M2-auth/M2-design.md
| M3 | 新手引导的创建工作区、加入工作区、邀请成员三步
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | 新手引导的创建工作区、加入工作区、邀请成员三步
````

````old docs/v0/M2-auth/M2-design.md
| M3 | **页大小的规则**
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | **页大小的规则**
````

````old docs/v0/M2-auth/M2-design.md
| M3 | **P5 改到、M2 的页面到不了的地方**
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md) | **P5 改到、M2 的页面到不了的地方**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **游标**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **游标**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **自动归档读 `updated_at`**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **自动归档读 `updated_at`**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **请求体检查的两处延伸**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **请求体检查的两处延伸**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **草稿发布**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **草稿发布**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **60 天物理清理**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **60 天物理清理**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | 命令行的"只投递"River 客户端
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | 命令行的"只投递"River 客户端
````

````old docs/v0/M2-auth/M2-design.md
| M4 | `user.service.ts` 中的 `getUserProfileIssues`
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | `user.service.ts` 中的 `getUserProfileIssues`
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **编辑器的代码分割**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **编辑器的代码分割**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **命令面板的主题命令**
````

````new docs/v0/M2-auth/M2-design.md
| [M4](../M4-issue-core/handoffs/M2-closeout.md) | **命令面板的主题命令**
````

````old docs/v0/M2-auth/M2-design.md
| M4 | **周期下拉框的取数**
````

````new docs/v0/M2-auth/M2-design.md
| M4；收尾改由 [M6](../M6-cycles-modules/handoffs/M2-closeout.md) 接（它取的是周期，周期在 M6，[收尾 spec](specs/closeout.md) 第 4 节 D5） | **周期下拉框的取数**
````

````old docs/v0/M2-auth/M2-design.md
| M3–M6（接上组件的 M） | **其余 12 个下拉框**
````

````new docs/v0/M2-auth/M2-design.md
| 接上组件的 M：[M3](../M3-workspace-project/handoffs/M2-closeout.md)、[M4](../M4-issue-core/handoffs/M2-closeout.md)、[M6](../M6-cycles-modules/handoffs/M2-closeout.md)、[M7](../M7-collaboration/handoffs/M2-closeout.md)（原写 M3–M6：M5 没有，收集箱和视图的两个在 M7，[收尾 spec](specs/closeout.md) 3.1） | **其余 12 个下拉框**
````

````old docs/v0/M2-auth/M2-design.md
| M3、M4、M6、M7（接上页面的 M） | **复制到剪贴板的失败**
````

````new docs/v0/M2-auth/M2-design.md
| [M3](../M3-workspace-project/handoffs/M2-closeout.md)、[M4](../M4-issue-core/handoffs/M2-closeout.md)、[M6](../M6-cycles-modules/handoffs/M2-closeout.md)、[M7](../M7-collaboration/handoffs/M2-closeout.md)（接上页面的 M） | **复制到剪贴板的失败**
````

````old docs/v0/M2-auth/M2-design.md
| M5 | **头像和封面**
````

````new docs/v0/M2-auth/M2-design.md
| [M5](../M5-files/handoffs/M2-closeout.md) | **头像和封面**
````

````old docs/v0/M2-auth/M2-design.md
| M5 | **上传与按路由的中间件**
````

````new docs/v0/M2-auth/M2-design.md
| [M5](../M5-files/handoffs/M2-closeout.md) | **上传与按路由的中间件**
````

````old docs/v0/M2-auth/M2-design.md
| M6 | 迭代（`cycles`、`cycle_issues`）
````

````new docs/v0/M2-auth/M2-design.md
| [M6](../M6-cycles-modules/handoffs/M2-closeout.md) | 迭代（`cycles`、`cycle_issues`）
````

````old docs/v0/M2-auth/M2-design.md
| M7 | 保存视图的列表有多种排序
````

````new docs/v0/M2-auth/M2-design.md
| [M7](../M7-collaboration/handoffs/M2-closeout.md) | 保存视图的列表有多种排序
````

````old docs/v0/M2-auth/M2-design.md
| M8 | 接口调用日志挂在限流之后
````

````new docs/v0/M2-auth/M2-design.md
| [M8](../M8-open-release/handoffs/M2-closeout.md) | 接口调用日志挂在限流之后
````

````old docs/v0/M2-auth/M2-design.md
是否加缓解，留给负责人以后决定 |
````

````new docs/v0/M2-auth/M2-design.md
是否加缓解，留给负责人以后决定；收尾之后记在[总体设计第 10 节](../v0-design.md) |
````

````old docs/v0/M2-auth/M2-design.md
重新输入密码、限制派生是负责人以后可选的产品选项 |
````

````new docs/v0/M2-auth/M2-design.md
重新输入密码、限制派生是负责人以后可选的产品选项；收尾之后记在[总体设计第 10 节](../v0-design.md) |
````

- [ ] **Step 2：每一行都有链接**：`node $M2TMP/closeout/rows.mjs . --design` → `13.2: 31 rows, 31 with a link to a handoff, 0 without`。

- [ ] **Step 3：链接的目标都存在**：收尾改动或新建的每一份文档（计划除外：它的块里的链接相对于块要写的文件），`node $M2TMP/closeout/paths.mjs . --links docs/v0/M2-auth/M2-design.md docs/v0/M2-auth/specs/closeout.md docs/v0/v0-design.md docs/v0/M0-foundation/M0-design.md docs/v0/plane-diff.md docs/v0/M2-auth/handoffs/*.md docs/v0/M[3-8]-*/handoffs/M2-closeout.md docs/v0/M[3-8]-*/handoffs/M1-closeout.md docs/v0/M5-files/handoffs/M1-P3-trim-platform.md` → 退出码 0，每一行 `… links, 0 missing`（原型）：

  ```
  docs/v0/M2-auth/M2-design.md: 76 links, 0 missing
  docs/v0/M2-auth/specs/closeout.md: 3 links, 0 missing
  docs/v0/v0-design.md: 26 links, 0 missing
  docs/v0/M0-foundation/M0-design.md: 32 links, 0 missing
  docs/v0/plane-diff.md: 3 links, 0 missing
  docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md: 4 links, 0 missing
  docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md: 7 links, 0 missing
  docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md: 4 links, 0 missing
  docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md: 5 links, 0 missing
  docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md: 5 links, 0 missing
  docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md: 11 links, 0 missing
  docs/v0/M2-auth/handoffs/M1-P2-trim-content.md: 4 links, 0 missing
  docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md: 6 links, 0 missing
  docs/v0/M2-auth/handoffs/M1-P4-router-native.md: 3 links, 0 missing
  docs/v0/M2-auth/handoffs/M1-closeout.md: 13 links, 0 missing
  docs/v0/M3-workspace-project/handoffs/M2-closeout.md: 9 links, 0 missing
  docs/v0/M4-issue-core/handoffs/M2-closeout.md: 10 links, 0 missing
  docs/v0/M5-files/handoffs/M2-closeout.md: 9 links, 0 missing
  docs/v0/M6-cycles-modules/handoffs/M2-closeout.md: 5 links, 0 missing
  docs/v0/M7-collaboration/handoffs/M2-closeout.md: 6 links, 0 missing
  docs/v0/M8-open-release/handoffs/M2-closeout.md: 10 links, 0 missing
  docs/v0/M3-workspace-project/handoffs/M1-closeout.md: 7 links, 0 missing
  docs/v0/M4-issue-core/handoffs/M1-closeout.md: 7 links, 0 missing
  docs/v0/M5-files/handoffs/M1-closeout.md: 10 links, 0 missing
  docs/v0/M6-cycles-modules/handoffs/M1-closeout.md: 8 links, 0 missing
  docs/v0/M7-collaboration/handoffs/M1-closeout.md: 7 links, 0 missing
  docs/v0/M8-open-release/handoffs/M1-closeout.md: 11 links, 0 missing
  docs/v0/M5-files/handoffs/M1-P3-trim-platform.md: 2 links, 0 missing
  ```

- [ ] **Step 4：守卫**（固定节奏第 3 步）。

- [ ] **Step 5：提交**（`docs(M2/closeout): each row of 13.2 points to its handoff`）

### Task 9: 全部门禁；第 14 节的技术项在收尾的头上重新证明

spec 2.6、第 5 节。不改文件，没有提交；结果写进收尾 review，控制者据此在收尾 review 的提交里打勾（最后一节）。

- [ ] **Step 1：七个门禁**，依次（`make e2e` 自己构建 `bin/nerve` 和前端，起自己的 testcontainers）：

| 命令 | 预期 |
|---|---|
| `make gen-check` | 退出码 0，没有"生成物与接口描述不一致" |
| `make lint-go` | 两段都是 `0 issues.` |
| `make test` | 32 行 `ok`，没有 `FAIL` |
| `make lint-web` | `keywords: 60 rules, 3 exceptions, no hits.`；`Tasks:    54 successful, 54 total` |
| `make knip` | 退出码 0，没有输出问题 |
| `make test-web` | `Tasks:    16 successful, 16 total` |
| `make e2e` | `48 passed` |

之后按 Global Constraints 的"容器"一条核对：没有这次运行建的容器。

- [ ] **Step 2：CSRF**：`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出（退出码 1）。

- [ ] **Step 3：7.9 的规则**：`node -e 'const k=require("./tools/keywords.json"); for (const r of k.rules) if (/^M2/.test(r.phase)) console.log(r.id, r.phase)'` →

```
is-self-managed M2/P4
plane-user-urls M2/P4
plane-user-assets M2/P4
csrf M2/P4
plane-auth-urls M2/P4
auth-error-code M2/P4
with-credentials M2/P5
plane-api-token-urls M2/P5
plane-timezone-urls M2/P5
```

7.9 的七行各有对应（spec 2.6 第 5 行的表）；`node tools/keywords.mjs` 在 Step 1 的 `make lint-web` 里没有命中。

- [ ] **Step 4：挂载时不请求旧接口**：`grep -n -E 'expect\(watch[AB]?\.(oldApiRequests|apiFailures|apiRequests)[,)]' e2e/stories/identity/a3-sign-in.spec.ts e2e/stories/identity/a7-change-password.spec.ts e2e/stories/identity/a8-update-me.spec.ts e2e/stories/identity/a9-preferences.spec.ts e2e/stories/identity/a10-onboarding-profile.spec.ts e2e/stories/identity/a11-api-tokens.spec.ts e2e/stories/identity/a12-deactivate.spec.ts e2e/stories/smoke/s2-web-app.spec.ts` 列出的断言对上 spec 2.6 第 5 行的页面表：M2 能到达的每个页面至少有一个故事在它挂载之后断言旧接口请求为空，或断言失败的接口请求全等（旧接口在 nerve 上是 404）。

- [ ] **Step 5：交接**：`grep -l -E '^status: (open|done)$' docs/v0/M2-auth/handoffs/*.md` 没有输出；`ls docs/v0/M*/handoffs/M2-closeout.md` 列出 M3–M8 六个文件，`grep -c '^status: open$'` 各为 1。

- [ ] **Step 6：本分支的改动就是文件表**：`git diff --stat d97c513 -- . ':!docs/v0/M2-auth/specs/closeout.md' ':!docs/v0/M2-auth/plans/closeout.md'` 列出的文件与"文件结构"的表相同（29 个文件：T1 一个、T2 一个、T3 三个、T5 两个、T6 十一个、T7 十个、T8 一个，T4 和 T3 改同一个 `docs/v0/v0-design.md`，Task 0 和 T8 改同一个 `docs/v0/M2-auth/M2-design.md`）。

---

## 收尾 review 提交里的替换（控制者）

收尾 review 写好之后，控制者在同一个提交里做下面三处替换（M1 的先例；spec 第 4 节 D1）。第 14 节每一项的证据是 T9 在收尾的头上的结果；review 的节号按 review 实际的写。

````old docs/v0/v0-design.md
| M2 | 账户认证 | 进行中 | [M2-design.md](M2-auth/M2-design.md) |
````

````new docs/v0/v0-design.md
| M2 | 账户认证 | 已完成 | [M2-design.md](M2-auth/M2-design.md) |
````

````old docs/v0/M2-auth/M2-design.md
| 收尾 | closeout | 进行中 | [spec](specs/closeout.md) | [plan](plans/closeout.md) | — |
````

````new docs/v0/M2-auth/M2-design.md
| 收尾 | closeout | 已完成 | [spec](specs/closeout.md) | [plan](plans/closeout.md) | [review](reviews/closeout-review.md) |
````

````old docs/v0/M2-auth/M2-design.md
- [ ] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。
- [ ] A1–A17 的页面版本（有页面的）和接口版本全部通过，S1–S4 通过。需要登录的每个操作都有 PAT 版本，调用同一组数据库断言；因凭证种类而不同的预期由参数表达，并写在故事里（A7）。
- [ ] 后端：单元测试、集成测试（含账户行锁的交错测试）、契约测试（四个整程序测试、`apitest` 对错误码的核对）、架构测试（含规则 4 的扩展和 `TestSQLCSchemaScope`）和 depguard 全部通过；`make gen-check` 覆盖 oapi-codegen、请求体结构表、sqlc 和 TS 客户端。
- [ ] 前端：类型检查通过，knip 为零；oxlint 等于上限，上限已按 7.8 调低；前端单元测试通过；关键词守卫没有未登记的命中。
- [ ] `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；前端不再调用 `/auth/…` 和 Plane 的用户、实例、令牌、时区地址（7.9 的规则守着）；M2 能到达的页面挂载时不请求 M3 的旧接口（3.1）。
- [ ] 4 张业务表（`users`、`profiles`、`auth_sessions`、`api_tokens`）由 M2 的迁移创建，以 Plane 表结构快照为起点；River 的表由它自己的迁移创建（3.15）；差异清单在建表的 Phase 登记了全局的建表约定（3.13）、每一列的改动和每一条行为差异（4.6），`sessions` 登记为替换模型。
- [ ] 第 10 节的四个决策点已由负责人裁定（2026-09-25）并已实现；第 11.1 节已由负责人批准，总体设计 4.2 已改写。
- [ ] 浏览器核对（9.6，含局域网 HTTP、两个账户两个标签页、会话暂不可用）的脚本全文写在各 Phase review 的附录中。
- [ ] `handoffs/` 中没有 `open` 的事项；交给 M3–M8 的交接已写好（13.2）。
- [ ] 总体设计、M0 设计、M0/P3 spec、差异清单、前端改动清单已在各 Phase 按 3.20 同步，收尾逐行核对过；README 已按 8.7 写好；总体设计中 M2 的状态改为"已完成"。
- [ ] 收尾 review 写明规模估计与实际的对比（7.10）。
````

````new docs/v0/M2-auth/M2-design.md
- [x] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。（第 15 节的七行都是"已完成"，各有三个链接。）
- [x] A1–A17 的页面版本（有页面的）和接口版本全部通过，S1–S4 通过。需要登录的每个操作都有 PAT 版本，调用同一组数据库断言；因凭证种类而不同的预期由参数表达，并写在故事里（A7）。（收尾的头上 `make e2e`：48 passed。故事在 `e2e/stories/identity/`（A1–A17）和 `e2e/stories/smoke/`（S1–S5）；页面版本和接口版本调用 `e2e/fixtures/assert/identity.ts` 的同一组断言。）
- [x] 后端：单元测试、集成测试（含账户行锁的交错测试）、契约测试（四个整程序测试、`apitest` 对错误码的核对）、架构测试（含规则 4 的扩展和 `TestSQLCSchemaScope`）和 depguard 全部通过；`make gen-check` 覆盖 oapi-codegen、请求体结构表、sqlc 和 TS 客户端。（收尾的头上 `make test`：32 个包 `ok`、没有 `FAIL`；`make lint-go`：两段 `0 issues.`；`make gen-check`：没有差异。交错测试在 `server/internal/modules/identity/interleavings_test.go`、`interleavings_reset_test.go`，整程序测试在 `server/internal/bootstrap/contract_test.go`，规则 4 的扩展是 `server/internal/archtest/rules_test.go:40`，`TestSQLCSchemaScope` 在 `server/internal/archtest/sqlc_test.go`；`make gen-go` 依次跑 oapi-codegen、`bodyshapegen`、sqlc，`make gen-web` 生成 TS 客户端。）
- [x] 前端：类型检查通过，knip 为零；oxlint 等于上限，上限已按 7.8 调低；前端单元测试通过；关键词守卫没有未登记的命中。（收尾的头上 `make lint-web`：`keywords: 60 rules, 3 exceptions, no hits.`，54 个任务成功（类型检查、oxlint 等于上限、格式、中英文的键）；`make knip` 为零；`make test-web`：16 个任务成功。上限由 M2/P5 调低：web 551 → 452，ui 25 → 19，utils 18 → 12，i18n 1 → 0。）
- [x] `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；前端不再调用 `/auth/…` 和 Plane 的用户、实例、令牌、时区地址（7.9 的规则守着）；M2 能到达的页面挂载时不请求 M3 的旧接口（3.1）。（收尾的头上 `git grep` 退出码 1；7.9 的规则是 `tools/keywords.json` 的 `csrf`、`plane-auth-urls`、`auth-error-code`、`is-self-managed`、`with-credentials`、`plane-user-urls`、`plane-api-token-urls`、`plane-timezone-urls`、`plane-user-assets`；挂载时的旧接口由故事守着：登录页 S2，注册页和 `/onboarding` A10，`/create-workspace` A3（收尾加），个人设置的四个标签页 A7、A8、A9、A11、A12，[收尾 spec](specs/closeout.md) 2.6。）
- [x] 4 张业务表（`users`、`profiles`、`auth_sessions`、`api_tokens`）由 M2 的迁移创建，以 Plane 表结构快照为起点；River 的表由它自己的迁移创建（3.15）；差异清单在建表的 Phase 登记了全局的建表约定（3.13）、每一列的改动和每一条行为差异（4.6），`sessions` 登记为替换模型。（`server/migrations/sql/00001`–`00004` 建四张表，`00005` 是 River 的迁移；差异清单一 B、二·全局、二·按表、第四节由收尾按第 4 节逐列核对，[收尾 spec](specs/closeout.md) 2.3。）
- [x] 第 10 节的四个决策点已由负责人裁定（2026-09-25）并已实现；第 11.1 节已由负责人批准，总体设计 4.2 已改写。（决策点 1 B：`nerve users set-email`，A16；2 C：prod 默认关闭注册、`nerve users create`，A2、A17、S3；3 A：自助停用和 `deactivate`、`activate` 两个命令，A12；4 A：两步删除，关键词规则 `is-self-managed`；11.1：退出只结束当前会话，A6；总体设计 4.2 第二条已改写。）
- [x] 浏览器核对（9.6，含局域网 HTTP、两个账户两个标签页、会话暂不可用）的脚本全文写在各 Phase review 的附录中。（P4、P5 两个有页面的 Phase 的 review 附录"9.6 的浏览器核对"：P4 的 C5 在局域网 HTTP 上，C4a–C4c 是两个账户两个标签页，C6b、C6c 是会话暂不可用；P1–P3b 只有后端。）
- [x] `handoffs/` 中没有 `open` 的事项；交给 M3–M8 的交接已写好（13.2）。（`grep -l -E '^status: (open|done)$' docs/v0/M2-auth/handoffs/*.md` 没有输出；M3–M8 各有 `handoffs/M2-closeout.md`，13.2 的每一行链接到它。）
- [x] 总体设计、M0 设计、M0/P3 spec、差异清单、前端改动清单已在各 Phase 按 3.20 同步，收尾逐行核对过；README 已按 8.7 写好；总体设计中 M2 的状态改为"已完成"。（逐行核对见[收尾 spec](specs/closeout.md) 2.2：四处漏同步由收尾补上，记为各 Phase 的漏项；README 的 8.7 各条在"部署"一节，同一节核对。）
- [x] 收尾 review 写明规模估计与实际的对比（7.10）。（[收尾 review](reviews/closeout-review.md)；实测和测法在[收尾 spec](specs/closeout.md) 附录 B。）
````

---

## 块的核对

写计划时在 `$M2TMP/closeout/` 下做了三件事，结果记在这里；执行者开始前可以重跑第一件，得到同样的输出。预检之后的修订都重做了一遍（原型换成新克隆的 `proto2`）。

**1. 每个旧文本在 `d97c513` 上恰好一处，新文件都不存在。** `node $M2TMP/closeout/planapply.mjs docs/v0/M2-auth/plans/closeout.md check $M2TMP/closeout/head`（`head/` 是 `d97c513` 的归档解开的树）。"in order"是依次应用本计划的块时这个旧文本在当时的文件里出现的次数；收尾 review 的三个替换（`TR`）中前两个的旧文本由 Task 3、Task 0 写入，在 `d97c513` 上是 0 处、按顺序应用到它们时恰好一处。退出码 0：

```
T0  old  docs/v0/M2-auth/M2-design.md (plan:111): at d97c513 1, in order 1
T1  old  tools/keywords.json (plan:129): at d97c513 1, in order 1
T2  old  e2e/stories/identity/a3-sign-in.spec.ts (plan:177): at d97c513 1, in order 1
T2  old  e2e/stories/identity/a3-sign-in.spec.ts (plan:188): at d97c513 1, in order 1
T3  old  docs/v0/v0-design.md (plan:251): at d97c513 1, in order 1
T3  old  docs/v0/v0-design.md (plan:259): at d97c513 1, in order 1
T3  old  docs/v0/M0-foundation/M0-design.md (plan:267): at d97c513 1, in order 1
T3  old  docs/v0/M0-foundation/M0-design.md (plan:275): at d97c513 1, in order 1
T3  old  docs/v0/M0-foundation/M0-design.md (plan:283): at d97c513 1, in order 1
T3  old  docs/v0/plane-diff.md (plan:291): at d97c513 1, in order 1
T4  old  docs/v0/v0-design.md (plan:327): at d97c513 1, in order 1
T4  old  docs/v0/v0-design.md (plan:335): at d97c513 1, in order 1
T4  old  docs/v0/v0-design.md (plan:355): at d97c513 1, in order 1
T4  old  docs/v0/v0-design.md (plan:363): at d97c513 1, in order 1
T4  old  docs/v0/v0-design.md (plan:373): at d97c513 1, in order 1
T5  new  docs/v0/M3-workspace-project/handoffs/M2-closeout.md: absent at the base (ok)
T5  new  docs/v0/M4-issue-core/handoffs/M2-closeout.md: absent at the base (ok)
T6  new  docs/v0/M5-files/handoffs/M2-closeout.md: absent at the base (ok)
T6  new  docs/v0/M6-cycles-modules/handoffs/M2-closeout.md: absent at the base (ok)
T6  new  docs/v0/M7-collaboration/handoffs/M2-closeout.md: absent at the base (ok)
T6  new  docs/v0/M8-open-release/handoffs/M2-closeout.md: absent at the base (ok)
T6  old  docs/v0/M3-workspace-project/handoffs/M1-closeout.md (plan:880): at d97c513 1, in order 1
T6  old  docs/v0/M4-issue-core/handoffs/M1-closeout.md (plan:888): at d97c513 1, in order 1
T6  old  docs/v0/M5-files/handoffs/M1-closeout.md (plan:896): at d97c513 1, in order 1
T6  old  docs/v0/M6-cycles-modules/handoffs/M1-closeout.md (plan:904): at d97c513 1, in order 1
T6  old  docs/v0/M7-collaboration/handoffs/M1-closeout.md (plan:912): at d97c513 1, in order 1
T6  old  docs/v0/M8-open-release/handoffs/M1-closeout.md (plan:920): at d97c513 1, in order 1
T6  old  docs/v0/M5-files/handoffs/M1-P3-trim-platform.md (plan:928): at d97c513 1, in order 1
T6  old  docs/v0/M5-files/handoffs/M1-P3-trim-platform.md (plan:936): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md (plan:971): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P1-sqlc-cgo.md (plan:979): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md (plan:998): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md (plan:1006): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md (plan:1031): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md (plan:1039): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md (plan:1061): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P4-schema-conventions.md (plan:1069): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md (plan:1088): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md (plan:1096): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md (plan:1115): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md (plan:1123): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-P2-trim-content.md (plan:1151): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-P2-trim-content.md (plan:1159): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md (plan:1173): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md (plan:1181): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-P4-router-native.md (plan:1204): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-P4-router-native.md (plan:1212): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-closeout.md (plan:1226): at d97c513 1, in order 1
T7  old  docs/v0/M2-auth/handoffs/M1-closeout.md (plan:1234): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1264): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1276): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1284): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1292): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1300): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1308): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1316): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1324): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1332): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1340): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1348): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1356): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1364): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1372): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1380): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1388): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1396): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1404): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1412): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1420): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1428): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1436): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1444): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1452): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1460): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1468): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1476): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1484): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1492): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1500): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1508): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1516): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1524): at d97c513 1, in order 1
T8  old  docs/v0/M2-auth/M2-design.md (plan:1532): at d97c513 1, in order 1
TR  old  docs/v0/v0-design.md (plan:1627): at d97c513 0, in order 1
TR  old  docs/v0/M2-auth/M2-design.md (plan:1635): at d97c513 0, in order 1
TR  old  docs/v0/M2-auth/M2-design.md (plan:1643): at d97c513 1, in order 1
80 replacements, 6 new files; all ok
```

**2. 块应用出来的树就是原型的树。** `$M2TMP/closeout/head-d97c513.tar` 另解开一份到 `$M2TMP/closeout/replay`，依次 `planapply.mjs … apply $M2TMP/closeout/replay 0` 到 `8`；原型 `proto2` 是本分支在 `11f773bd`（Task 0、Task 1 已提交）的克隆，依次应用了 `2` 到 `8`。两边都放进这一版的 spec 和本计划（`proto2` 带着的是 `7435bfb1` 提交的旧版），再逐字节比较（`node $M2TMP/closeout/treediff.mjs replay proto2`，不含 `node_modules`、`.git` 和构建产物）：

```
2651 and 2651 files; 0 differences
```

`replay` 放进 spec 和计划之前与 `d97c513` 比较（`treediff.mjs head replay`）：23 个改（`M`）、6 个新（`B`），`2643 and 2649 files; 29 differences`，与"文件结构"的表相同。

**3. 原型的全部门禁**（`proto2`，Task 2–8 之后，`bash $M2TMP/closeout/gates.sh $M2TMP/closeout/proto2 final2`）：

```
gen-check: exit 0 in 4s;
lint-go: exit 0 in 8s; 2 × '0 issues.'
test: exit 0 in 21s; 32 ok, 0 FAIL
lint-web: exit 0 in 15s; keywords: 60 rules, 3 exceptions, no hits.;  Tasks: 54 successful, 54 total
knip: exit 0 in 3s; pnpm --filter web exec react-router typegen pnpm exec knip --treat-config-hints-as-errors
test-web: exit 0 in 6s;  Tasks: 16 successful, 16 total
e2e: exit 0 in 14s;  48 passed (12.7s)
```

`proto2` 是新克隆，turbo 的缓存大多不在：`make lint-web` 54 个任务中缓存 10 个，`make test-web` 16 个中缓存 10 个。每个 Task 的检查结果与各 Task 写的预期相同（spec 附录 A.2）。
