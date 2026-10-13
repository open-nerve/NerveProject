# M3/收尾 closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** M3 的收尾。负责人的两件：`ModalCore` 改用 Headless UI v2 的 `Dialog` 的打开方式，弹窗打开时后面的页面 `inert`、`aria-hidden`（只在打开时才挂载的弹窗也是），Tab 走不出弹窗，请求在路上时关不掉、组字的 Escape 不关、关上之后焦点回到打开它的控件照旧，`AlertModalCore` 和快捷键的搜索框不再是正的 `tabIndex`，命令面板、快捷键和切换账户三个弹窗改经 `ModalCore`，设置页的手机菜单降到弹窗之下；侧边栏的工作区菜单加"离开工作区"，每个角色都有，经成员页的同一个确认框和命令，唯一的管理员被拒绝时确认框留着。各 Phase 留给收尾的事（spec 2.4）：M3 页面上的英文和角色名、`IconButton` 的名字和收尾的键盘清单、共用的写法（复制、问题码、拒绝的文字）和文案的检查、输入法和选择上的 Enter、工作区的地址经路由、测试的整理和契约的说明。收尾的文档：M3 收到的 6 份交接关闭，M4–M8 的 5 份交接，README、差异清单、M3 设计和前端改动清单。

**Architecture:** 改 web（应用和 `@nerve/ui`、`@nerve/propel`、`@nerve/utils`、`@nerve/constants`、`@nerve/i18n` 五个包）、`e2e/`、`tools/`（关键词规则，新的 `i18n-keys.mjs`）、`Makefile`、`knip.jsonc`、文档；Go 只改测试（Task 14）；契约只改 `project.yaml` 中五个字段的说明，`make gen` 写出的三个生成文件在 Task 14 钉住。共用的新部分：`@nerve/ui` 的 `ModalCore`（Task 1）；`@nerve/utils` 的 `isComposing`（Task 10，`@nerve/propel` 因此依赖 `@nerve/utils`）；web 的 `components/common/translated.tsx`（`Translated`，一句一个键，Task 4）、`components/common/color-picker.tsx`（Task 8）、`core/hooks/use-route-workspace.ts`（Task 11）、`use-copy-link.ts` 的 `useCopyText`、`use-refusal-toast.ts` 的 `useRefusalMessage`、`lib/api-error.ts` 的 `problemCode`（Task 9）；e2e 的 `dialogs.ts` 的 `modalDialog`、`pageBehindHidden`、`announced`（Task 1、2），`e2e/fixtures/assert/session.ts`、`auth.ts` 的 `registeredPAT`（Task 14）。页面调用 store 的方法，修改的跟进经 `followInSession`；类型只来自生成的客户端。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、Headless UI 2.2.10、Base UI 1.0.0-beta.3（`@base-ui-components/react`）、react-hook-form 7.84、react-color 2.19.3、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1（只改测试）。版本都由 `pnpm-lock.yaml` 和 `go.mod` 固定。

**Spec:** `docs/v0/M3-workspace-project/specs/closeout.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **基点**：本 plan 和 spec 与 Task 0（M3 设计第 15 节收尾一行改为"进行中"、9.7 加 C11、C12）在同一个提交里；Task 1 从这个提交开始。
- **依赖**：不加任何 npm 包或 Go 模块，不执行 `go get`、`pnpm add`。Task 10 让 `@nerve/propel` 依赖工作区内的 `@nerve/utils`（`"@nerve/utils": "workspace:*"`），`pnpm-lock.yaml` 随之多 3 行（Task 10 的块）；Task 10 的块写入之后执行 `pnpm install --frozen-lockfile`（锁文件已是新的，它只链接）。
- **生成的文件**：只有 Task 14 改契约（`api/modules/project.yaml` 的说明文字），之后执行 `make gen`；它写出的三个文件钉在 Task 14 的末尾（SHA-256 和行数）。其余 Task 不执行 `make gen`，`make gen-check` 照旧没有差异。
- **每个 Task 提交前**：`make lint-web`（关键词守卫、`tsc`、oxlint 在上限、格式、`en` 与 `zh-CN` 的键一致、读了却没有定义的键（Task 9 起））、`make knip`、`make test-web`、`make build`、`make e2e` 都通过；Task 14 另跑 `make gen-check`、`make lint-go`、`make test`。Task 15、16 只改文档：跑 `make lint-web`（文档不在它的范围内，跑它是为了看守卫照旧）和 `doclinks.py`（下面"一次性的脚本"）。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 110 个；Task 2 起 112、Task 4 起 113、Task 5 起 114、Task 6 起 115、Task 8 起 116、Task 12 起 117、Task 13 起 118。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（P4b spec F4），其余照上面的数目减一全部通过。
- **容器**：`make e2e`、`make test` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**（总体设计 7.7；P8a–P11 的裁定）：会话的每个取数经 `useSessionSWR`；修改经 store 的方法，之后的跳转、提示、界面变化经 `followInSession` 只在发出它的会话里（Task 3 的离开：hook 的 vitest 的"换账户之后"一行）。弹窗：请求在路上时关不掉（F-1）、被拒绝时留着、只在成功的跟进中关上（A-M2）、组字的 Escape 不关、打开时后面的页面 `inert` 和 `aria-hidden`、Tab 不出去、关上之后焦点回到打开它的控件（spec 2.5）；每一条由一个会失败的端到端检查看住（各 Task 的变异表）。每个控件 Tab 能到、有名字、键盘取得焦点时看得见，不加正的 `tabIndex`。页面读地址只经路由参数（`project-address-route`、`workspace-address-route`）。文案：页面上的文字经 `t()`，en 与 zh-CN 同时加，一句话带名字时一个键加占位（`Translated`）；不再读的键在两种语言中都删除。类型只来自生成的客户端：没有新的 `as`、`any`、`!`（spec 附录 A.5）。删除的代码删干净（组件、类型、两种语言的文案、常量），不加 knip 的忽略、开关或桩。本 plan 写或重写的文件都在约 400 行以内（spec 附录 A.5）；只为使用方改到的 Plane 文件不变长，记录在案的例外一个：`IconButton` 的名字在根上必传，25 个使用方的文件只加名字（`aria-label`、`addReactionLabel`、`useTranslation` 的导入和 `t` 一行，以及 oxfmt 的重排），其中 21 个变长（控制者已裁定，spec 第 3 节第 1 条、附录 A.5）。
- **oxlint**（M3 设计 7.9）：有手改的文件在它的 Task 提交时没有 oxlint 警告。收尾改到、仍有警告的 5 个文件照控制者的裁定交出（spec 第 3 节第 2 条）：只为 `IconButton` 的名字或 `showTooltip` 改到的 4 个给 M4、M6、M7，只改了占位文字的页头搜索 `navigation/top-nav-power-k.tsx` 给 M8（Task 15 写进各自的交接）。上限：web 303 → 301（Task 2）→ 300（Task 8）→ 298（Task 11）→ 295（Task 12）；`@nerve/ui` 15 → 14（Task 1）→ 12（Task 10）；`@nerve/propel` 11 → 10（Task 7）；utils 7、constants 1 不变。抑制多两条（Task 2 的快捷键搜索框和 Task 12 的筛选，`autoFocus` 写明理由，spec 第 3 节第 2、20 条），删去两条：181 条照旧。
- **关键词守卫**：69 条规则、3 个例外；Task 9 起 70 条（`copy-hook`）、23 个例外；Task 11 起 71 条（`workspace-address-route`）。规则的 `phase` 写 `M3/closeout`，例外的 `until` 写 `M4/closeout` 等（spec 第 3 节第 5 条）：在那个 M 的收尾开始、`phase` 改为它时到期，Task 9 把这一点写进守卫的说明和 M1 设计 7.4，Task 15 写进收到例外的 M 的交接。
- **注释**：TS 代码、测试、JSON 的说明用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 Task 0 之后的树起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同；之后在另一个副本上从本提交逐 Task 复现，每个 Task 之后跑它的门禁，最后的树与原型逐个文件相同（spec 附录 A.9）。块写入之后不必再格式化（原型中每个 Task 都经过 `oxfmt`）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（文件结构一节的"Task"一列列出每个文件的全部 Task）：`e2e/fixtures/dialogs.ts`（Task 1、2、10）、S6（Task 2、9、10）、S7（Task 4–8、12、13：每个 Task 加它的页面）、P5、P6、P7、W7（几个 Task 各加一处检查）、两种语言的文案文件（每个 Task 加它读的键）、`tools/keywords.json`（Task 9、11）、web、`@nerve/ui` 和 `@nerve/propel` 的 `package.json`（上限）、`modal-core.tsx`（Task 1、10）、`switch-account-modal.tsx`（Task 2、4）、设置页的手机菜单（Task 2、7）、`workspace-menu-root.tsx`（Task 3、11）、`member-columns.tsx`（Task 4、6、11）、侧边栏的项目列表（Task 7、8、11）、`card.tsx`（Task 5、12）、P1、P2（各几个 Task）、M3 设计（Task 9、11、15、16）、README（Task 3、16）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（静态：`make lint-web` 的 `tsc`、oxlint、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest：`make test-web`；Go：`make test` 的一个包；端到端：`make e2e` 的故事）。它们在各自 Task 的树上跑过，也在最终的原型上逐个跑过（`$M3TMP/closeout-tools/mutants_closeout.py`，`mut.py` 在它写的每个检查上各跑一次；spec 附录 A.2）。**会话、权限、焦点或取数的性质只由评审才能发现的，算缺口**（brief）：表中每一条这类性质都有一个会失败的检查。
- **评审敏感**（spec 第 1 节）：弹窗的每一条规则（上面"规则"一条）；离开的入口只做 nerve 允许的（唯一的管理员被拒绝时确认框留着，访客能离开），跟进只在发出它的会话里；每一处可见的不同写进前端改动清单（spec 附录 A.8）。改动这些之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性的脚本

`$M3TMP/closeout-plan/doclinks.py`（Task 15、16 的文档核对：给出的 markdown 文件中每个相对链接都指向仓库中存在的文件或目录；本 plan 和 spec 与 Task 0 同在一个提交里，指向它们的链接也算）：

```python
# M3 closeout plan, Tasks 15 and 16: every relative link of the markdown files given resolves, from the file's folder,
# to a file or folder of the repository (a link's #anchor is not checked). Prints one line per file and each link that
# does not resolve; exits 1 when one does not. usage: python3 doclinks.py <root> <markdown file>...
import os
import re
import sys

root = sys.argv[1]
LINK = re.compile(r'\]\(([^)\s]+)\)')
broken = 0
for rel in sys.argv[2:]:
    text = open(os.path.join(root, rel), encoding='utf-8').read()
    base = os.path.dirname(os.path.join(root, rel))
    links = [m.group(1) for m in LINK.finditer(text) if not re.match(r'[a-z]+:', m.group(1))]
    missing = [link for link in links
               if link.split('#')[0] and not os.path.exists(os.path.normpath(os.path.join(base, link.split('#')[0])))]
    print(f'{rel}: {len(links)} links, {len(missing)} broken')
    for link in missing:
        print('   ', link)
    broken += len(missing)
print('broken:', broken)
sys.exit(1 if broken else 0)
```

## 文件结构

路径相对于仓库根目录。一个文件由多个 Task 修改时，"Task"一列都列出（"新"是新文件，"删除"是删除）。生成的三个文件（Task 14 的 `make gen`）不在表中，钉在 Task 14 的末尾。

| 文件 | 职责 | Task |
|---|---|---|
| `Makefile` | `make lint-web` 跑 `tools/i18n-keys.mjs`（读了却没有定义的键） | 9 |
| `README.md` | 工作区菜单加"离开工作区"（Task 3）；数据库的 locale、"或经接口"（Task 16） | 3、16 |
| `api/modules/project.yaml` | 五个字段的说明说 nerve 对它们做什么（K24；`make gen`） | 14 |
| `docs/v0/M1-frontend-trim/M1-design.md` | 7.4 写上 `until` 为 `M<n>/closeout` 的例外何时到期 | 9 |
| `docs/v0/M3-workspace-project/M3-design.md` | 7.10 的规则表补上 P9、P10 和收尾的规则（Task 9、11）；13.2 的接收者链接交接（Task 15）；5.1 写裁定 B1（Task 16） | 9、11、15、16 |
| `docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`、`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`、`docs/v0/M3-workspace-project/handoffs/M1-closeout.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md` | M3 收到的交接：处理结果，`status: closed` | 15 |
| `docs/v0/M4-issue-core/handoffs/M1-closeout.md`、`docs/v0/M5-files/handoffs/M1-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M1-closeout.md`、`docs/v0/M7-collaboration/handoffs/M1-closeout.md`、`docs/v0/M8-open-release/handoffs/M1-closeout.md` | `always-return` 已由 M3/P11 全仓清零 | 15 |
| `docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md`、`docs/v0/M7-collaboration/handoffs/M2-closeout.md` | 复制一节的关闭条件经 `useCopyLink`、`useCopyText` | 15 |
| `docs/v0/M4-issue-core/handoffs/M3-closeout.md`、`docs/v0/M5-files/handoffs/M3-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M3-closeout.md`、`docs/v0/M7-collaboration/handoffs/M3-closeout.md`、`docs/v0/M8-open-release/handoffs/M3-closeout.md` | M3 交给这个 M 的（新文件，spec 第 4 节） | 15 新 |
| `docs/v0/frontend-changes.md` | 3.1 的 M3 一行、输入法和 P11 键盘两行；3.2 加收尾的 7 行 | 16 |
| `docs/v0/plane-diff.md` | 邀请链接一行的 Phase 说法 | 16 |
| `e2e/fixtures/assert/identity.ts` | 会话的断言移出，`accountStateOf` 经 `sessionsOf` | 14 |
| `e2e/fixtures/assert/session.ts` | 会话的断言和 `sessionsOf`（新文件，从 `identity.ts` 拆出） | 14 新 |
| `e2e/fixtures/auth.ts` | `registeredPAT(api, email)` | 14 |
| `e2e/fixtures/browser.ts` | `expectNoAddress` 移来 | 14 |
| `e2e/fixtures/dialogs.ts` | `modalDialog`、`pageBehindHidden`（Task 1）；`announced`（Task 2）；`composingEscape` 发两种顺序（Task 10） | 1、2、10 |
| `e2e/fixtures/mounts.ts` | `WORKSPACE_MENU`：工作区菜单打开时不发请求（W2 的常驻检查） | 3 |
| `e2e/fixtures/project-pages.ts` | `labelRow` 移来（Task 6）；成员一行的菜单按它的名字，`pageContent`：侧边栏之外的页面内容（Task 8） | 6、8 |
| `e2e/stories/identity/a10-onboarding-profile.spec.ts`、`e2e/stories/identity/a12-deactivate.spec.ts`、`e2e/stories/identity/a13-reset-password.spec.ts`、`e2e/stories/identity/a14-session-cleanup.spec.ts`、`e2e/stories/identity/a16-set-email.spec.ts`、`e2e/stories/identity/a3-sign-in.spec.ts`、`e2e/stories/identity/a4-refresh.spec.ts`、`e2e/stories/identity/a5-refresh-reuse.spec.ts`、`e2e/stories/identity/a7-change-password.spec.ts`、`e2e/stories/identity/a8-update-me.spec.ts`、`e2e/stories/identity/a9-preferences.spec.ts`、`e2e/stories/project/p9-guest-profile.spec.ts`、`e2e/stories/workspace/w1-create-workspace.spec.ts`、`e2e/stories/workspace/w10-admin-create-workspace.spec.ts`、`e2e/stories/workspace/w12-reactivate-member.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`、`e2e/stories/workspace/w9-deactivation.spec.ts` | `registeredPAT`；会话断言的导入 | 14 |
| `e2e/stories/identity/a11-api-tokens.spec.ts` | 令牌复制到剪贴板（Task 9）；`registeredPAT`（Task 14） | 9、14 |
| `e2e/stories/identity/a6-sign-out.spec.ts` | 切换账户的弹窗在退出路上时关不掉（Task 2）；导入（Task 14） | 2、14 |
| `e2e/stories/project/p1-create-project.spec.ts` | 以标题找卡片（Task 5）；在页面内容中找卡片（Task 8）；功能的开关以功能为名（Task 13）；`registeredPAT`（Task 14） | 5、8、13、14 |
| `e2e/stories/project/p2-visibility.spec.ts` | "Joining"（Task 5）；在页面内容中找页头的面包屑（Task 8）；`registeredPAT`（Task 14） | 5、8、14 |
| `e2e/stories/project/p3-project-settings.spec.ts` | 换到另一个项目的设置再后退，开着的归档确认框关上（Task 3）；自定义范围说 Update（Task 13）；`registeredPAT`（Task 14） | 3、13、14 |
| `e2e/stories/project/p8-project-preferences.spec.ts` | 访客在 Docs 的页头打开离开的确认框，后退到 Web 的工作项时关上（Task 3）；`registeredPAT`（Task 14） | 3、14 |
| `e2e/stories/project/p4-archive.spec.ts` | 删除项目的弹窗：后面的页面隐藏、焦点回来（Task 1）；确认的提示正是要输入的词（Task 5）；`registeredPAT`（Task 14） | 1、5、14 |
| `e2e/stories/project/p5-project-members.spec.ts` | 提示被读出（Task 2）；"Leaving"、确认的提示正是要输入的词（Task 5）；一行的菜单（Task 8）；角色上按 Enter（Task 10） | 2、5、8、10 |
| `e2e/stories/project/p6-states-page.spec.ts` | 删除确认框的 Tab 循环、隐藏（Task 1）；提示被读出（Task 2）；组的名字（Task 6）；名称的焦点、颜色框的轮廓（Task 8） | 1、2、6、8 |
| `e2e/stories/project/p7-labels-page.spec.ts` | 删除确认框的 Tab 循环、隐藏（Task 1）；提示被读出（Task 2）；"Delete label"（Task 6） | 1、2、6 |
| `e2e/stories/smoke/s2-web-app.spec.ts` | 每个角色打开工作区菜单，不发请求（`WORKSPACE_MENU`） | 3 |
| `e2e/stories/smoke/s6-dialogs.spec.ts` | S6：命令面板、快捷键（以标题为名）、嵌套的弹窗是模态的（新文件，Task 2）；命令面板的复制（Task 9）；组字的 Escape（Task 10） | 2 新、9、10 |
| `e2e/stories/smoke/s7-chinese.spec.ts` | S7：中文的页面和弹窗（新文件，逐 Task 加） | 4 新、5、6、7、8、12、13 |
| `e2e/stories/smoke/s8-sidebar.spec.ts` | S8：侧边栏的键盘，项目的一行以项目为名（新文件） | 8 新 |
| `e2e/fixtures/workspace-pages.ts` | `askToEndMembership`：打开一行的菜单、选移出或离开，到确认框为止（`endMembership` 经它） | 3 |
| `e2e/stories/workspace/w11-guest-bounds.spec.ts` | 访客从菜单离开（双击只发一次，请求在路上时确认框关不掉）、唯一的管理员被拒绝（Task 3）；`registeredPAT`、导入（Task 14） | 3、14 |
| `e2e/stories/workspace/w4-invite-members.spec.ts` | 删除邀请被拒绝时确认框留着（Task 3）；`registeredPAT`（Task 14） | 3、14 |
| `e2e/stories/workspace/w5-invitation-link.spec.ts` | 邀请被拒绝之后的页面的说法（Task 4）；`registeredPAT`、会话断言的导入（Task 14） | 4、14 |
| `e2e/stories/workspace/w7-member-management.spec.ts` | 条件挂载的确认框隐藏后面的页面（Task 1）；被拒绝时确认框留着（Task 3）；`registeredPAT`（Task 14） | 1、3、14 |
| `e2e/stories/workspace/w8-navigation-preferences.spec.ts` | 弹窗关上之后再读侧边栏（Task 1）；`registeredPAT`（Task 14） | 1、14 |
| `knip.jsonc` | knip 的入口加 `tools/i18n-keys.mjs` | 9 |
| `pnpm-lock.yaml` | `@nerve/propel` 依赖 `@nerve/utils`（工作区内） | 10 |
| `server/internal/modules/project/adapter/http/projects_test.go`、`server/internal/modules/project/adapter/http/update_test.go` | `identifierTakenJSON`、`nameTakenJSON` | 14 |
| `server/internal/platform/jobs/jobs_test.go` | 期限代替余量，定期的运行按 `ScheduledAt` 排序 | 14 |
| `tools/i18n-keys.mjs` | 页面读的每个键在 en 中都有定义（新文件，spec 2.10） | 9 新 |
| `tools/keywords.json` | 关键词规则 `copy-hook`（20 个有期限的例外）、`refusal-toast`、`sign-out-toast` 的写法（Task 9）；`workspace-address-route`（Task 11） | 9、11 |
| `tools/keywords.mjs` | 说明写上 `until` 为 `M<n>/closeout` 的例外何时到期 | 9 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx` | 工作区经 `useRouteWorkspace()` 或写明 slug（Task 11）；侧边栏的标题经 `t()`（Task 12） | 11、12 |
| `web/apps/web/app/(all)/[workspaceSlug]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx` | 工作区的内容以路由的 slug 为 key，项目的页面和设置以路由的项目为 key：换工作区、换项目时开着的确认框和组件的状态不带过去（Task 3） | 3 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx` | 工作区经 `useRouteWorkspace()` 或写明 slug（Task 11） | 11 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx` | `IconButton` 的名字 | 7 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx` | 成员页的标题经 `t()`（Task 4）；工作区经 `useRouteWorkspace()`（Task 11） | 4、11 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx`、`web/apps/web/core/components/project/card-list.tsx`、`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx` | 工作区经 `useRouteWorkspace()`（Task 11） | 11 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/page.tsx` | 项目设置各页的标题经 `t()`；没有项目时的设置页；它们的 vitest 照新的标题 | 13 |
| `web/apps/web/app/(all)/create-workspace/layout.tsx`、`web/apps/web/app/(all)/create-workspace/page.tsx`、`web/apps/web/app/(all)/onboarding/layout.tsx`、`web/apps/web/app/(all)/onboarding/page.tsx` | 页面标题经 `PageHead`、`t()`，布局的英文 `meta` 删除；装饰的图片 `alt=""` | 13 |
| `web/apps/web/app/(all)/workspace-invitations/layout.tsx`、`web/apps/web/app/(all)/workspace-invitations/page.tsx` | 页面标题经 `PageHead`、`t()`，布局的英文 `meta` 删除；装饰的图片 `alt=""` | 4 |
| `web/apps/web/app/provider.tsx` | `AppToast`：提示的区域经 `t()` 命名 | 13 |
| `web/apps/web/core/components/api-token/modal/create-token-modal.tsx` | 不再传空的 `handleClose` | 2 |
| `web/apps/web/core/components/api-token/modal/generated-token-details.tsx`、`web/apps/web/core/components/power-k/config/miscellaneous-commands.ts` | 复制经 `useCopyText`、`useCopyLink` | 9 |
| `web/apps/web/core/components/appearance/theme-switcher.tsx` | 失败的提示经 `useRefusalMessage` | 9 |
| `web/apps/web/core/components/auth-screens/not-authorized-view.tsx`、`web/apps/web/core/components/home/user-greetings.tsx`、`web/apps/web/core/components/navigation/project-navigation-dialog.tsx`、`web/apps/web/core/components/navigation/top-nav-power-k.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/core/components/workspace/delete-workspace-modal.tsx` | M3 页面上的英文经 `t()` | 4 |
| `web/apps/web/core/components/automation/auto-archive-automation.tsx`、`web/apps/web/core/components/automation/select-month-modal.test.tsx`、`web/apps/web/core/components/automation/select-month-modal.tsx` | 自动归档的月数和自定义范围经 `t()`；它的测试 | 13 |
| `web/apps/web/core/components/comments/card/display.tsx`、`web/apps/web/core/components/comments/comment-reaction.tsx`、`web/apps/web/core/components/comments/quick-actions.tsx`、`web/apps/web/core/components/core/description-versions/modal.tsx`、`web/apps/web/core/components/cycles/cycles-view-header.tsx`、`web/apps/web/core/components/cycles/quick-actions.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx`、`web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx`、`web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx`、`web/apps/web/core/components/issues/peek-overview/header.tsx`、`web/apps/web/core/components/modules/module-view-header.tsx`、`web/apps/web/core/components/modules/quick-actions.tsx`、`web/apps/web/core/components/rich-filters/filters-toggle.tsx`、`web/apps/web/core/components/settings/profile/modal.tsx`、`web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx`、`web/apps/web/core/components/views/quick-actions.tsx`、`web/apps/web/core/components/views/view-list-header.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx` | `IconButton` 的名字（M2、M4、M6、M7 的调用方） | 7 |
| `web/apps/web/core/components/common/color-picker.tsx` | 颜色弹层：十六进制输入框的轮廓（新文件） | 8 新 |
| `web/apps/web/core/components/common/translated.test.tsx`、`web/apps/web/core/components/common/translated.tsx` | `Translated`：带名字的一句一个键（新文件）和它的测试 | 4 新 |
| `web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx`、`web/apps/web/core/components/dropdowns/member/avatar.tsx`、`web/apps/web/core/components/dropdowns/member/base.tsx`、`web/apps/web/core/components/inbox/sidebar/inbox-list-item.tsx`、`web/apps/web/core/components/issues/attachment/attachment-list-item.tsx`、`web/apps/web/core/components/issues/issue-detail/sidebar.tsx`、`web/apps/web/core/components/issues/peek-overview/properties.tsx`、`web/apps/web/core/components/modules/module-card-item.tsx`、`web/apps/web/core/components/modules/module-list-item-action.tsx`、`web/apps/web/core/components/views/view-list-item-action.tsx` | `ButtonAvatars` 不再收 `showTooltip` | 9 |
| `web/apps/web/core/components/dropdowns/project/base.test.tsx` | 项目下拉框的停留点是它的按钮；它的测试（新文件） | 8 新 |
| `web/apps/web/core/components/dropdowns/project/base.tsx` | 项目下拉框的停留点是它的按钮；它的测试（新文件） | 8 |
| `web/apps/web/core/components/editor/embeds/mentions/user.tsx` | 角色名经 `ROLE_DETAILS` | 6 |
| `web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx` | 工作区经 `useRouteWorkspace()` | 11 |
| `web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`、`web/apps/web/core/components/issues/select/base.tsx`、`web/apps/web/core/hooks/use-dropdown-key-down.tsx`、`web/apps/web/core/hooks/use-keypress.tsx`、`web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx` | 组字经 `isComposing` | 10 |
| `web/apps/web/core/components/labels/create-update-label-inline.tsx` | 颜色弹层（Task 8） | 8 |
| `web/apps/web/core/components/labels/delete-label-modal.tsx`、`web/apps/web/core/components/labels/project-setting-label-group.tsx`、`web/apps/web/core/components/labels/project-setting-label-item.tsx` | 标签的英文经 `t()`（Task 6） | 6 |
| `web/apps/web/core/components/navigation/tab-navigation-overflow-menu.test.tsx`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`、`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx` | 标签栏的菜单和提示经 `t()`；它们的测试 | 12 |
| `web/apps/web/core/components/navigation/tab-navigation-visible-item.test.tsx` | 标签栏的菜单和提示经 `t()`；它们的测试 | 12 新 |
| `web/apps/web/core/components/navigation/top-navigation-root.tsx` | 工作区菜单不再收 `variant` | 3 |
| `web/apps/web/core/components/navigation/use-tab-preferences.test.ts`、`web/apps/web/core/components/navigation/use-tab-preferences.ts` | 标签栏的菜单和提示经 `t()`；它们的测试 | 5 |
| `web/apps/web/core/components/onboarding/switch-account-modal.test.tsx` | 切换账户的那句话是一个键，地址在其中（新文件，Task 4）；共用的替身 `fake-i18n-values.ts`（Task 5） | 4 新、5 |
| `web/apps/web/core/components/onboarding/switch-account-modal.tsx` | 经 `ModalCore`，退出路上时关不掉（Task 2）；那句话一个键，经 `Translated`（Task 4） | 2、4 |
| `web/apps/web/core/components/power-k/ui/modal/shortcuts-root.tsx` | 经 `ModalCore`，标题和搜索框经 `t()`，关闭按钮在标题之外，`autoFocus` 写明理由 | 2 |
| `web/apps/web/core/components/power-k/ui/modal/wrapper.tsx` | 经 `ModalCore`（Task 2）；组字的守卫（Task 10） | 2、10 |
| `web/apps/web/core/components/profile/sidebar.tsx` | `IconButton` 的名字（Task 7）；工作区经 `useRouteWorkspace()`（Task 11） | 7、11 |
| `web/apps/web/core/components/project-states/create-update/create-update.test.tsx`、`web/apps/web/core/components/project-states/options/delete.test.tsx` | 读键的测试 | 6 |
| `web/apps/web/core/components/project-states/create-update/create.tsx`、`web/apps/web/core/components/project-states/create-update/update.tsx`、`web/apps/web/core/components/project-states/group-item.tsx`、`web/apps/web/core/components/project-states/options/delete.tsx`、`web/apps/web/core/components/project-states/options/mark-as-default.tsx` | 状态的英文经 `t()`、组的名字（Task 6） | 6 |
| `web/apps/web/core/components/project-states/create-update/form.tsx` | 状态的英文经 `t()`、组的名字（Task 6）；颜色弹层、名称的焦点（Task 8） | 6、8 |
| `web/apps/web/core/components/project/add-project-members-modal.test.tsx`、`web/apps/web/core/components/project/archive-restore-modal.test.tsx`、`web/apps/web/core/components/project/archive-restore-modal.tsx`、`web/apps/web/core/components/project/confirm-project-member-remove.tsx`、`web/apps/web/core/components/project/delete-project-modal.test.tsx`、`web/apps/web/core/components/project/delete-project-modal.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/project/join-project-modal.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx`、`web/apps/web/core/components/project/member-select.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/components/project/settings/control-section.tsx` | 项目的弹窗和设置的英文经 `t()`，一句一个键；它们的 vitest | 5 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx` | 项目的弹窗和设置的英文经 `t()`，一句一个键；它们的 vitest | 5、6 |
| `web/apps/web/core/components/project/applied-filters/access.tsx`、`web/apps/web/core/components/project/applied-filters/date.tsx`、`web/apps/web/core/components/project/applied-filters/members.tsx`、`web/apps/web/core/components/project/applied-filters/project-display-filters.tsx`、`web/apps/web/core/components/project/applied-filters/root.tsx`、`web/apps/web/core/components/project/dropdowns/filters/access.tsx`、`web/apps/web/core/components/project/dropdowns/filters/created-at.tsx`、`web/apps/web/core/components/project/dropdowns/filters/member-list.tsx`、`web/apps/web/core/components/project/dropdowns/filters/members.tsx`、`web/apps/web/core/components/project/dropdowns/filters/root.tsx` | 筛选的名字、选项和移除按钮经 `t()`；负责人的筛选是成员筛选的另一个调用；成员筛选的按钮 | 12 |
| `web/apps/web/core/components/project/applied-filters/root.test.tsx` | 筛选的名字、选项和移除按钮经 `t()`；负责人的筛选是成员筛选的另一个调用；成员筛选的按钮 | 12 新 |
| `web/apps/web/core/components/project/card.test.tsx` | 卡片的文字（Task 5、12）；设置的链接的名字（Task 5） | 5、12 |
| `web/apps/web/core/components/project/card.tsx` | 项目页的英文（Task 5、12）；设置的链接的名字（Task 5） | 5、12 |
| `web/apps/web/core/components/project/dropdowns/filters/lead.tsx` | 筛选的名字、选项和移除按钮经 `t()`；负责人的筛选是成员筛选的另一个调用；成员筛选的按钮 | 12 删除 |
| `web/apps/web/core/components/project/form-loader.tsx`、`web/apps/web/core/components/project/settings/helper.tsx` | 没有用到的加载、包了一层的开关删除 | 13 删除 |
| `web/apps/web/core/components/project/header.tsx` | 项目页的英文（Task 12）；工作区经 `useRouteWorkspace()`（Task 11） | 11、12 |
| `web/apps/web/core/components/project/member-list.tsx` | 搜索的占位或名字 | 13 |
| `web/apps/web/core/components/project/project-dialog-texts.test.tsx` | 项目的弹窗和设置的英文经 `t()`，一句一个键；它们的 vitest | 5 新 |
| `web/apps/web/core/components/project/root.tsx` | 工作区经 `useRouteWorkspace()`，回调和副作用写全依赖（Task 11） | 11 |
| `web/apps/web/core/components/project/search-projects.tsx` | 搜索的占位或名字 | 7 |
| `web/apps/web/core/components/project/settings/features-list.tsx` | 功能的开关以功能为名（Task 13） | 13 |
| `web/apps/web/core/components/project/settings/member-columns.tsx` | 成员一行的菜单（Task 8） | 8 |
| `web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx` | 功能开关的提示经 `t()`（Task 5）；开关的名字（Task 13） | 5、13 |
| `web/apps/web/core/components/project/settings/use-feature-toggle.ts` | 功能开关的提示经 `t()`（Task 5） | 5 |
| `web/apps/web/core/components/settings/mobile/nav.tsx` | 设置页的手机菜单降到弹窗之下（`z-20`，Task 2）；`IconButton` 的名字（Task 7） | 2、7 |
| `web/apps/web/core/components/settings/profile/content/pages/security.tsx` | `signOut` 的取法（`sign-out-toast`） | 9 |
| `web/apps/web/core/components/settings/project/content/feature-control-item.tsx` | `title` 是字符串，给开关命名 | 13 |
| `web/apps/web/core/components/settings/project/sidebar/header.tsx` | 设置侧边栏的标题经 `t()`（Task 4） | 4 |
| `web/apps/web/core/components/settings/workspace/sidebar/header.tsx` | 设置侧边栏的标题经 `t()`（Task 4）；名字（Task 7）；写明 slug（Task 11） | 4、7、11 |
| `web/apps/web/core/components/sidebar/resizable-sidebar.tsx` | 名字经 `t()`（Task 4）；不显示的一份 `inert`，死的状态删除（Task 8） | 4、8 |
| `web/apps/web/core/components/sidebar/sidebar-wrapper.tsx` | 标题经 `t()`，没有调用方的 `title` 删除 | 12 |
| `web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx` | 确认框不再自己关上，被拒绝时留着（Task 3）；标题和说明经 `t()`（Task 4） | 3、4 |
| `web/apps/web/core/components/workspace/leave-workspace-modal.tsx` | 侧边栏的离开：成员页的确认框和命令（新文件） | 3 新 |
| `web/apps/web/core/components/workspace/settings/invitations-list-item.tsx` | 确认框不再自己关上，被拒绝时留着（Task 3）；角色名（Task 6） | 3、6 |
| `web/apps/web/core/components/workspace/settings/member-columns.tsx` | M3 页面上的英文（Task 4）；角色名（Task 6）；工作区经 `useRouteWorkspace()`（Task 11） | 4、6、11 |
| `web/apps/web/core/components/modals/workspace-level.tsx` | 宿主卸载时关上它渲染的弹窗的开关（创建项目，开关在命令面板的 store 里，在 key 之上；Task 3） | 3 |
| `web/apps/web/core/components/power-k/projects-app-provider.tsx`、`web/apps/web/core/components/power-k/global-shortcuts.tsx` | 宿主卸载时关上命令面板（并清掉它的页）、快捷键列表（开关在 power-K 的 store 里，在 key 之上；Task 3） | 3 |
| `web/apps/web/core/components/workspace/settings/members-list-item.tsx` | 确认框不再自己关上，被拒绝时留着（Task 3） | 3 |
| `web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts` | 移出、离开、删除邀请收 `done`，只在成功时调用；它们的测试（Task 3）；删除邀请的提示经 `t()`（Task 4） | 3、4 |
| `web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-membership-changes.ts` | 移出、离开、删除邀请收 `done`，只在成功时调用；它们的测试（Task 3） | 3 |
| `web/apps/web/core/components/workspace/settings/workspace-details.test.tsx` | M3 页面上的英文（Task 4） | 4 |
| `web/apps/web/core/components/workspace/settings/workspace-details.tsx` | M3 页面上的英文（Task 4）；工作区经 `useRouteWorkspace()`（Task 11） | 4、11 |
| `web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx` | 工作区的一项在方向键选中时显出底色（Task 3）；角色名（Task 6） | 3、6 |
| `web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx` | `IconButton` 的名字（Task 7）；"+"、一行的菜单、手柄、展开箭头的键盘，一行以项目为名（Task 8） | 7、8 |
| `web/apps/web/core/components/workspace/sidebar/projects-list.tsx` | `IconButton` 的名字（Task 7）；"+"、一行的菜单、手柄、展开箭头的键盘，一行以项目为名（Task 8）；写明 slug（Task 11） | 7、8、11 |
| `web/apps/web/core/components/workspace/sidebar/quick-actions.tsx` | 写明 slug（Task 11） | 11 |
| `web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx` | "离开工作区"一项，菜单的按钮和三项在键盘下看得见，没有调用方的写法删除（Task 3）；工作区经 `useRouteWorkspace()`（Task 11） | 3、11 |
| `web/apps/web/core/components/workspace/use-create-workspace.ts`、`web/apps/web/core/components/workspace/use-invitation-answer.ts` | 经 `problemCode()` 或 `useRefusalMessage` | 9 |
| `web/apps/web/core/hooks/use-copy-link.test.ts`、`web/apps/web/core/hooks/use-copy-link.ts` | `useCopy`、`useCopyText`；它的测试 | 9 |
| `web/apps/web/core/hooks/use-refusal-toast.test.ts`、`web/apps/web/core/hooks/use-refusal-toast.ts` | `useRefusalMessage`；它的测试 | 9 |
| `web/apps/web/core/hooks/use-route-workspace.test.ts`、`web/apps/web/core/hooks/use-route-workspace.ts` | `useRouteWorkspace()`（新文件）和它的测试 | 11 新 |
| `web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`、`web/apps/web/core/lib/form-refusal.ts` | 经 `problemCode()` | 9 |
| `web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx` | "找不到工作区"的标题经 `PageHead` | 4 |
| `web/apps/web/core/lib/api-error.test.ts`、`web/apps/web/core/lib/api-error.ts` | `problemCode()`；它的测试 | 9 |
| `web/apps/web/core/lib/fake-i18n-values.ts` | 替身：文字是键和它收到的值（新文件） | 5 新 |
| `web/apps/web/core/lib/fake-controls.ts` | 替身：`AlertModalCore` 的文字（Task 6）；开关的名字（Task 13） | 6、13 |
| `web/apps/web/core/store/member/project/project-member.store.test.ts` | 资料来自 `userOf` | 14 |
| `web/apps/web/package.json` | web 的 oxlint 上限 303 → 301（Task 2）；301 → 300（Task 8）；300 → 298（Task 11）；298 → 295（Task 12） | 2、8、11、12 |
| `web/packages/constants/src/filter.ts` | 删去没有读者的日期选项（Task 9）；创建日期的选项 `i18n_name`（Task 12） | 9、12 |
| `web/packages/constants/src/project.ts` | `PROJECT_FILTER_I18N_LABEL` | 12 |
| `web/packages/constants/src/state.ts` | 状态组的 `i18n_label` | 6 |
| `web/packages/constants/src/workspace.ts` | `ORGANIZATION_SIZE_I18N_LABEL`（Task 4）；角色的 `i18n_title`（Task 6） | 4、6 |
| `web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 4、7、13 |
| `web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 3、4、12、13 |
| `web/packages/i18n/src/locales/en/navigation.json`、`web/packages/i18n/src/locales/zh-CN/navigation.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 7、12 |
| `web/packages/i18n/src/locales/en/power-k.json`、`web/packages/i18n/src/locales/zh-CN/power-k.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 2、4、9 |
| `web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 4、5、6、13 |
| `web/packages/i18n/src/locales/en/project.json`、`web/packages/i18n/src/locales/zh-CN/project.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 5、6、12 |
| `web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 3、4、9 |
| `web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json` | 两种语言的新键（每个 Task 加它读的，删它不再读的） | 4、12 |
| `web/packages/propel/package.json` | oxlint 上限 11 → 10（Task 7）；依赖 `@nerve/utils`（Task 10） | 7、10 |
| `web/packages/propel/src/emoji-icon-picker/logo.tsx` | 图标的连字和表情 `aria-hidden` | 7 |
| `web/packages/propel/src/emoji-reaction/emoji-reaction.tsx` | 表情回应的按钮收 `aria-label`，加回应的按钮收 `addReactionLabel`，回应按表情作 `key` | 7 |
| `web/packages/propel/src/icon-button/helper.tsx` | `aria-label` 必传 | 7 |
| `web/packages/propel/src/toast/toast.tsx` | 提示的区域收 `label` | 13 |
| `web/packages/ui/package.json` | `@nerve/ui` 的 oxlint 上限 15 → 14（Task 1）；14 → 12（Task 10） | 1、10 |
| `web/packages/ui/src/dropdowns/custom-search-select.tsx`、`web/packages/ui/src/dropdowns/custom-select.tsx`、`web/packages/ui/src/hooks/use-dropdown-key-down.tsx` | 组字经 `isComposing`；`use-dropdown-key-down` 的两条警告（Task 10）；`CustomSelect` 上按 Enter 打开列表 | 10 |
| `web/packages/ui/src/modals/alert-modal.tsx` | 确认按钮不再是 `tabIndex={1}` | 1 |
| `web/packages/ui/src/modals/modal-core.tsx` | `ModalCore`：v2 的 `Dialog` 打开方式，从挂载之后的渲染打开（Task 1）；组字经 `isComposing`（Task 10） | 1、10 |
| `web/packages/utils/src/index.ts` | 不再导出 `permission/`（Task 6）；导出 `keyboard`（Task 10） | 6、10 |
| `web/packages/utils/src/keyboard.test.ts`、`web/packages/utils/src/keyboard.ts` | `isComposing`：`isComposing` 或 `keyCode` 229（新文件）和它的测试 | 10 新 |
| `web/packages/utils/src/permission/index.ts`、`web/packages/utils/src/permission/role.ts` | `ROLE`、`getUserRole` 删除 | 6 删除 |

---

### Task 1: `ModalCore` 改用 Headless UI v2 的打开方式；后面的页面 `inert`、隐藏，Tab 不出去

**Files:**
- Modify: `e2e/fixtures/dialogs.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/project/p6-states-page.spec.ts`、`e2e/stories/project/p7-labels-page.spec.ts`、`e2e/stories/workspace/w7-member-management.spec.ts`、`e2e/stories/workspace/w8-navigation-preferences.spec.ts`、`web/packages/ui/package.json`、`web/packages/ui/src/modals/alert-modal.tsx`、`web/packages/ui/src/modals/modal-core.tsx`

**Interfaces:**（spec 2.5）
- `@nerve/ui` 的 `ModalCore`（`modals/modal-core.tsx`，整个文件）：`Dialog` 一直挂载，`open={isOpen && mounted}`，`mounted` 在第一次挂载之后的副作用里设为真（调用方只在打开时挂载的弹窗也先挂载、后打开：Headless UI 在挂载之后才找到它的主树）；过渡经 `DialogBackdrop`、`DialogPanel` 的 `transition`（`data-closed`、`data-leave`，300/200 毫秒照旧）；`onClose` 调 `handleClose?.()`，不传时 Escape 和点外面都不关；面板的 `onKeyDown` 在组字的 Escape 上阻止默认（Headless UI 的 `useEscape` 因此不关）。props 不变。
- `AlertModalCore` 的确认按钮去掉 `tabIndex={1}`；`@nerve/ui` 的上限 15 → 14。
- `e2e/fixtures/dialogs.ts`：`modalDialog(page)`（打开的模态弹窗）、`pageBehindHidden(page)`（主树的 `body` 子元素有 `inert` 和 `aria-hidden`）。

**Tests:** 没有新的测试文件。P4（删除项目的弹窗第一次打开时后面的页面隐藏，关上之后焦点回到打开它的按钮）、P6、P7（删除确认框：从取消起 Tab 依次是删除、取消，打开时隐藏、关上之后焦点回来）、W7（成员页的移出确认框是条件挂载的：扣住移出的请求时后面的页面仍隐藏）；W8 的项目数在导航弹窗关上之后再读（后面的页面现在 `aria-hidden`）。

- [ ] **Step 1: 弹窗**

`web/packages/ui/package.json`（修改，1 处）：

````old web/packages/ui/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 15",
````
````new web/packages/ui/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 14",
````

`web/packages/ui/src/modals/alert-modal.tsx`（修改，1 处）：

````old web/packages/ui/src/modals/alert-modal.tsx
        <Button variant={BUTTON_VARIANTS[variant]} tabIndex={1} onClick={handleSubmit} loading={isSubmitting}>
````
````new web/packages/ui/src/modals/alert-modal.tsx
        <Button variant={BUTTON_VARIANTS[variant]} onClick={handleSubmit} loading={isSubmitting}>
````

`web/packages/ui/src/modals/modal-core.tsx`（整个文件，74 行）：

````whole web/packages/ui/src/modals/modal-core.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Dialog, DialogBackdrop, DialogPanel } from "@headlessui/react";
import React, { useEffect, useState } from "react";
// constants
import { cn } from "../utils";
import { EModalPosition, EModalWidth } from "./constants";

type Props = {
  children: React.ReactNode;
  /** Closes the dialog on Escape and a click on the backdrop; without it, neither closes it. */
  handleClose?: () => void;
  isOpen: boolean;
  position?: EModalPosition;
  width?: EModalWidth;
  className?: string;
};

/**
 * A modal dialog: Headless UI's Dialog, shown while isOpen, its backdrop and panel fading in and out; its caller may
 * mount it with its page or only while it is open. While it is open the page behind it is inert and hidden from
 * assistive technology (aria-hidden): Tab stays within the dialog, the pointer reaches nothing behind it, and a screen
 * reader reads the dialog alone; the lists, pickers and toasts that open from it render outside the page (portals) and
 * stay usable and read. Closed, it gives the focus back to what had it before it opened.
 */
export function ModalCore(props: Props) {
  const {
    children,
    handleClose,
    isOpen,
    position = EModalPosition.CENTER,
    width = EModalWidth.XXL,
    className = "",
  } = props;
  // Headless UI's Dialog finds the page behind it (its main tree) only once it has mounted, and makes that page inert
  // as it opens: one mounted open would leave the page live (P10's defect ii). So it opens from the render after its
  // first, a caller's dialog mounted only while open too.
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  return (
    <Dialog open={isOpen && mounted} onClose={() => handleClose?.()} className="relative z-30">
      <DialogBackdrop
        transition
        className="fixed inset-0 bg-backdrop transition-opacity duration-300 ease-out data-closed:opacity-0 data-leave:duration-200 data-leave:ease-in"
      />
      <div className="fixed inset-0 z-30 overflow-y-auto">
        <div className={position}>
          <DialogPanel
            transition
            className={cn(
              "relative w-full transform rounded-lg bg-surface-1 text-left shadow-raised-200 transition-all duration-300 ease-out data-closed:translate-y-4 data-closed:opacity-0 data-leave:duration-200 data-leave:ease-in sm:data-closed:translate-y-0 sm:data-closed:scale-95",
              width,
              className
            )}
            // an Escape that ends an IME composition is the composition's: Headless UI's Dialog closes on a keydown
            // of Escape on the window unless its default is prevented (useEscape), so this prevents it on the way
            // out, once the controls inside have had it (in the capture phase, it would make Headless UI skip their
            // own handlers, which it does for a prevented event: a select's, which keeps its Escape from the page)
            onKeyDown={(event) => {
              if (event.key === "Escape" && event.nativeEvent.isComposing) event.preventDefault();
            }}
          >
            {children}
          </DialogPanel>
        </div>
      </div>
    </Dialog>
  );
}
````

- [ ] **Step 2: 端到端**

`e2e/fixtures/dialogs.ts`（修改，2 处）：

````old e2e/fixtures/dialogs.ts

/**
 * Resolves with whether the one modal dialog page shows closes within a second, such as one that should stay once
````
````new e2e/fixtures/dialogs.ts

/**
 * The one modal dialog page shows: a toast is a dialog too, not a modal one, and does not count.
 */
export function modalDialog(page: Page): Locator {
  return page.locator('[role="dialog"][aria-modal="true"]');
}

/**
 * Resolves with whether the one modal dialog page shows closes within a second, such as one that should stay once
````
````old e2e/fixtures/dialogs.ts
  return page
    .locator('[role="dialog"][aria-modal="true"]')
    .waitFor({ state: "detached", timeout: 1_000 })
    .then(() => true, timedOut);
````
````new e2e/fixtures/dialogs.ts
  return modalDialog(page)
    .waitFor({ state: "detached", timeout: 1_000 })
    .then(() => true, timedOut);
}

/**
 * Resolves with whether the page behind page's dialogs, its first main element, is inert and hidden from assistive
 * technology (aria-hidden), as ModalCore makes it while a dialog is open: nothing behind the dialog is reached by Tab
 * or the pointer, or read; and neither once the dialog has closed. The page is read by its element, not by a role:
 * Playwright's role queries do not see what is aria-hidden.
 */
export async function pageBehindHidden(page: Page): Promise<boolean> {
  return page
    .locator("main")
    .first()
    .evaluate((main) => main.closest("[inert]") !== null && main.closest('[aria-hidden="true"]') !== null);
````

`e2e/stories/project/p4-archive.spec.ts`（修改，3 处）：

````old e2e/stories/project/p4-archive.spec.ts
import { closedByComposingEscape, closedByEscape, enabledWithin } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p4-archive.spec.ts
import { closedByComposingEscape, closedByEscape, enabledWithin, pageBehindHidden } from "../../fixtures/dialogs";
````
````old e2e/stories/project/p4-archive.spec.ts
  // stays open, and a plain Escape still closes it.
  await page.getByRole("button", { name: "Delete", exact: true }).click();
  await page.locator("#projectName").fill("We");
````
````new e2e/stories/project/p4-archive.spec.ts
  // stays open, and a plain Escape still closes it. While it is open the page behind it is inert and hidden; closed,
  // the page is back, and so is the focus, on the button that opened it.
  const deleteButton = page.getByRole("button", { name: "Delete", exact: true });
  await deleteButton.click();
  await page.locator("#projectName").fill("We");
  expect(await pageBehindHidden(page)).toBe(true);
````
````old e2e/stories/project/p4-archive.spec.ts
  expect(await closedByEscape(page)).toBe(true);
````
````new e2e/stories/project/p4-archive.spec.ts
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(deleteButton).toBeFocused();
````

`e2e/stories/project/p6-states-page.spec.ts`（修改，3 处）：

````old e2e/stories/project/p6-states-page.spec.ts
import { closedByEscape, closedWithin, enabledWithin, sentWithin } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p6-states-page.spec.ts
import {
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  pageBehindHidden,
  sentWithin,
} from "../../fixtures/dialogs";
````
````old e2e/stories/project/p6-states-page.spec.ts
  // In Progress made the default meanwhile, elsewhere: nerve refuses its deletion, said in a toast, the dialog open.
````
````new e2e/stories/project/p6-states-page.spec.ts
  // In Progress made the default meanwhile, elsewhere: nerve refuses its deletion, said in a toast, the dialog open,
  // Tab going round its two buttons and the page behind it hidden; Cancel closes it, and the page and the focus are
  // back.
````
````old e2e/stories/project/p6-states-page.spec.ts
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);
````
````new e2e/stories/project/p6-states-page.spec.ts
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).focus();
  expect(await tabCycle(page, modalDialog(page), 4)).toEqual(["Delete", "Cancel"]);
  expect(await pageBehindHidden(page)).toBe(true);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(inProgress.getByRole("button", { name: "Delete" })).toBeFocused();
````

`e2e/stories/project/p7-labels-page.spec.ts`（修改，3 处）：

````old e2e/stories/project/p7-labels-page.spec.ts
import { closedByEscape, closedWithin, enabledWithin } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p7-labels-page.spec.ts
import { closedByEscape, closedWithin, enabledWithin, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
````
````old e2e/stories/project/p7-labels-page.spec.ts
  // Docs deleted meanwhile, elsewhere: nerve refuses its deletion here, said in a toast, the dialog open; Cancel closes
  // it.
````
````new e2e/stories/project/p7-labels-page.spec.ts
  // Docs deleted meanwhile, elsewhere: nerve refuses its deletion here, said in a toast, the dialog open, Tab going
  // round its two buttons and the page behind it hidden; Cancel closes it, and the page and the focus are back.
````
````old e2e/stories/project/p7-labels-page.spec.ts
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);
````
````new e2e/stories/project/p7-labels-page.spec.ts
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).focus();
  expect(await tabCycle(page, modalDialog(page), 4)).toEqual(["Delete", "Cancel"]);
  expect(await pageBehindHidden(page)).toBe(true);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(docsDelete).toBeFocused();
````

`e2e/stories/workspace/w7-member-management.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w7-member-management.spec.ts
import { closedByEscape } from "../../fixtures/dialogs";
````
````new e2e/stories/workspace/w7-member-management.spec.ts
import { closedByEscape, pageBehindHidden } from "../../fixtures/dialogs";
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  // While the removal is out the dialog that asked cannot be dismissed: Cancel is disabled, and Escape leaves it open.
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  // While the removal is out the dialog that asked cannot be dismissed: Cancel is disabled, and Escape leaves it open.
  // The page mounts the dialog only while it is open: the page behind it is hidden all the same.
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await pageBehindHidden(page)).toBe(true);
````

`e2e/stories/workspace/w8-navigation-preferences.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { closedWithin, pageBehindHidden } from "../../fixtures/dialogs";
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  // A count beyond nerve's bounds: nerve refuses it, the page says why, and the field and the sidebar keep 3.
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  // A count beyond nerve's bounds: nerve refuses it, the page says why, and the field keeps 3; the sidebar, hidden
  // behind the dialog while it is open, keeps 3 too once it has closed.
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  await expect(page.getByText("Some fields are not valid.")).toBeVisible();
  await expect(count).toHaveValue("3");
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  await expect(page.getByText("Some fields are not valid.")).toBeVisible();
  await expect(count).toHaveValue("3");
  expect(await pageBehindHidden(page)).toBe(true);
  await dialog.getByRole("button", { name: "Close" }).click();
  expect(await closedWithin(page)).toBe(true);
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 303 条、`@nerve/ui` 14 条、`@nerve/propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 110 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/dialogs.ts e2e/stories/project/p4-archive.spec.ts e2e/stories/project/p6-states-page.spec.ts e2e/stories/project/p7-labels-page.spec.ts e2e/stories/workspace/w7-member-management.spec.ts e2e/stories/workspace/w8-navigation-preferences.spec.ts web/packages/ui/package.json web/packages/ui/src/modals/alert-modal.tsx web/packages/ui/src/modals/modal-core.tsx
```
```bash
git commit -m "feat(M3/closeout): ModalCore opens Headless UI's Dialog in place, the page behind it inert and hidden from its first opening

The Dialog stays mounted and opens with isOpen, from the render after its
first: Headless UI finds the page behind it only once it has mounted, so a
dialog its caller mounts only while open left that page live (P10's defect
ii). The backdrop and the panel fade as before; an Escape that ends a
composition still closes nothing, nor does Escape while a request is out.
AlertModalCore's confirm button is no longer tabIndex={1}: Tab goes round
Cancel and the confirm button. The stories check the page behind each
dialog hidden while it is open, the focus back on its opener after, and
the confirmation's Tab cycle.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | `Dialog` 只在打开时挂载，像原来 `Transition.Root` 那样（P10 的缺陷 ii）：第一次打开时后面的页面既不 `inert` 也不隐藏 | 故事 P4、故事 P6、故事 P7 | 端到端 |
| `T1.2` | 确认框的确认按钮回到 `tabIndex={1}`：Tab 走出弹窗的顺序 | `@nerve/ui` 的 oxlint（`check:lint`）、故事 P6、故事 P7 | 静态；端到端 |
| `T1.3` | 弹窗不理 `handleClose`：Escape 和背景都关不掉它 | 故事 P4 | 端到端 |
| `T1.4` | 确认框在请求路上时被 Escape 关上（F-1） | 故事 P6、故事 P7 | 端到端 |
| `T1.5` | 结束组字的 Escape 关上弹窗：面板的 keydown 放它过去 | 故事 P4 | 端到端 |
| `T1.6` | `Dialog` 从 `ModalCore` 的第一次渲染就打开：调用方只在打开时才挂载的弹窗，后面的页面照旧能操作 | 故事 W7 | 端到端 |

---

### Task 2: 命令面板、快捷键和切换账户经 `ModalCore`；S6

**Files:**
- Create: `e2e/stories/smoke/s6-dialogs.spec.ts`
- Modify: `e2e/fixtures/dialogs.ts`、`e2e/stories/identity/a6-sign-out.spec.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`e2e/stories/project/p6-states-page.spec.ts`、`e2e/stories/project/p7-labels-page.spec.ts`、`web/apps/web/core/components/api-token/modal/create-token-modal.tsx`、`web/apps/web/core/components/onboarding/switch-account-modal.tsx`、`web/apps/web/core/components/power-k/ui/modal/shortcuts-root.tsx`、`web/apps/web/core/components/power-k/ui/modal/wrapper.tsx`、`web/apps/web/core/components/settings/mobile/nav.tsx`、`web/apps/web/package.json`、`web/packages/i18n/src/locales/en/power-k.json`、`web/packages/i18n/src/locales/zh-CN/power-k.json`

**Interfaces:**（spec 2.6）
- `power-k/ui/modal/wrapper.tsx`（命令面板）、`shortcuts-root.tsx`（快捷键，整个文件）、`onboarding/switch-account-modal.tsx`（整个文件）经 `ModalCore`：快捷键的搜索框不再是 `tabIndex={1}`，它的 `autoFocus` 写明理由（`oxlint-disable-next-line`，同 Task 12 的筛选；web 的上限 303 → 301）；快捷键的关闭按钮名为 `t("close")`，在 `DialogTitle` 之外（弹窗的名字只是标题，A-L5）；标题经 `t("keyboard_shortcuts")`（已有的键），搜索框的占位和名字经 `t("power_k.search_shortcuts")`（新键）；切换账户在退出的请求路上时 `handleClose` 为空（F-1）。
- `settings/mobile/nav.tsx`：设置页的手机菜单（外层、弹层、按钮）从 `z-50` 降到 `z-20`：页面上的元素停在弹窗层（`ModalCore` 的 `z-30`）之下，弹窗盖住它（spec 2.6）。
- API 令牌的新建弹窗不再传 `handleClose={() => {}}`（与不传相同）。
- `e2e/fixtures/dialogs.ts`：`announced(element)`（元素在 `aria-live` 的区域里）。

**Tests:** 新的故事 S6（`e2e/stories/smoke/s6-dialogs.spec.ts`，2 个）：命令面板和快捷键列表是模态的，快捷键弹窗的名字就是它的标题；设置弹窗上的撤销确认框（嵌套），Escape 只关上面的一个。A6：切换账户在退出被扣住时关不掉。P5、P6、P7：弹窗开着时被拒绝的提示被读出（`announced`）。

- [ ] **Step 1: 三个弹窗**

`web/apps/web/core/components/api-token/modal/create-token-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/api-token/modal/create-token-modal.tsx
    <ModalCore isOpen={isOpen} handleClose={() => {}} position={EModalPosition.TOP} width={EModalWidth.XXL}>
````
````new web/apps/web/core/components/api-token/modal/create-token-modal.tsx
    <ModalCore isOpen={isOpen} position={EModalPosition.TOP} width={EModalWidth.XXL}>
````

`web/apps/web/core/components/onboarding/switch-account-modal.tsx`（整个文件，82 行）：

````whole web/apps/web/core/components/onboarding/switch-account-modal.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { ArrowRightLeft } from "lucide-react";
import { DialogTitle } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useUser } from "@/hooks/store/user";
import { useSignOut } from "@/hooks/use-sign-out";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function SwitchAccountModal(props: Props) {
  const { isOpen, onClose } = props;
  const { t } = useTranslation();
  // states
  const [switchingAccount, setSwitchingAccount] = useState(false);
  // store hooks
  const { data: userData } = useUser();
  const signOut = useSignOut();

  const handleClose = () => {
    setSwitchingAccount(false);
    onClose();
  };

  const handleSwitchAccount = async () => {
    setSwitchingAccount(true);
    if (await signOut()) handleClose();
    setSwitchingAccount(false);
  };

  // While the sign-out is out the dialog cannot be dismissed (Escape, the backdrop): it closes once nerve has signed
  // the session out, and stays, the switch offered again, when it has not.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={switchingAccount ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
      className="overflow-hidden"
    >
      <div className="p-6 pb-1">
        <div className="flex gap-x-4">
          <div className="flex items-start">
            <div className="grid place-items-center rounded-full bg-accent-primary/20 p-4">
              <ArrowRightLeft className="h-5 w-5 text-accent-primary" aria-hidden="true" />
            </div>
          </div>
          <div className="flex flex-col gap-y-6 py-3">
            <DialogTitle as="h3" className="text-20 leading-6 font-medium text-primary">
              {t("onboarding.switch_account.title")}
            </DialogTitle>
            {userData?.email && (
              <div className="text-14 font-regular text-secondary">
                {t("onboarding.switch_account.description_before_email")}{" "}
                <span className="text-accent-primary">{userData.email}</span>{" "}
                {t("onboarding.switch_account.description_after_email")}
              </div>
            )}
          </div>
        </div>
      </div>
      <div className="mb-2 flex items-center justify-end gap-3 p-4 sm:px-6">
        <Button variant="secondary" size="lg" onClick={handleSwitchAccount} disabled={switchingAccount}>
          {switchingAccount ? t("onboarding.switch_account.switching") : t("onboarding.switch_account.switch")}
        </Button>
      </div>
    </ModalCore>
  );
}
````

`web/apps/web/core/components/power-k/ui/modal/shortcuts-root.tsx`（整个文件，81 行）：

````whole web/apps/web/core/components/power-k/ui/modal/shortcuts-root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { DialogTitle } from "@headlessui/react";
// nerve imports
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { ScrollArea } from "@nerve/propel/scrollarea";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { usePowerK } from "@/hooks/store/use-power-k";
// local imports
import { ShortcutRenderer } from "../renderer/shortcut";

type Props = {
  isOpen: boolean;
  onClose: () => void;
};

export function ShortcutsModal(props: Props) {
  const { isOpen, onClose } = props;
  // states
  const [query, setQuery] = useState("");
  // store hooks
  const { commandRegistry } = usePowerK();
  const { t } = useTranslation();

  // Get all commands from registry
  const allCommandsWithShortcuts = commandRegistry.getAllCommandsWithShortcuts();

  const handleClose = () => {
    onClose();
    setQuery("");
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={handleClose}
      position={EModalPosition.TOP}
      width={EModalWidth.MD}
      className="flex h-[61vh] flex-col space-y-4 overflow-hidden py-5"
    >
      {/* the title names the dialog alone: its close button stands beside it */}
      <div className="flex justify-between px-5">
        <DialogTitle as="h3" className="text-16 font-medium">
          {t("keyboard_shortcuts")}
        </DialogTitle>
        <button type="button" onClick={handleClose} aria-label={t("close")}>
          <CloseOutline className="h-4 w-4 text-secondary hover:text-primary" aria-hidden="true" />
        </button>
      </div>
      <div className="px-5">
        <InputGroup size="2xl">
          <SearchOutline className="h-3.5 w-3.5 text-secondary" />
          <Input
            size="2xl"
            id="search"
            name="search"
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("power_k.search_shortcuts")}
            aria-label={t("power_k.search_shortcuts")}
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- the dialog opens on its search, as the palette does
            autoFocus
          />
        </InputGroup>
      </div>

      <ScrollArea size="sm" rootClassName="overflow-y-scroll px-5">
        <ShortcutRenderer searchQuery={query} commands={allCommandsWithShortcuts} />
      </ScrollArea>
    </ModalCore>
  );
}
````

`web/apps/web/core/components/power-k/ui/modal/wrapper.tsx`（修改，2 处）：

````old web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
import { Dialog, Transition } from "@headlessui/react";
````
````new web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````old web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
    <Transition.Root show={isOpen} as={React.Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        {/* Backdrop */}
        <Transition.Child
          as={React.Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-backdrop transition-opacity" />
        </Transition.Child>
        {/* Modal Container */}
        <div className="fixed inset-0 z-30 overflow-y-auto">
          <div className="flex items-center justify-center p-4 sm:p-6 md:p-20">
            <Transition.Child
              as={React.Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
              enterTo="opacity-100 translate-y-0 sm:scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 translate-y-0 sm:scale-100"
              leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
            >
              <Dialog.Panel className="divide-opacity-10 relative flex w-full max-w-2xl transform flex-col items-center justify-center divide-y divide-subtle-1 rounded-lg bg-surface-1 shadow-raised-200 transition-all">
                <Command
                  filter={(i18nValue: string, search: string) => {
                    if (i18nValue === "no-results") return 1;
                    if (i18nValue.toLowerCase().includes(search.toLowerCase())) return 1;
                    return 0;
                  }}
                  shouldFilter={searchTerm.length > 0}
                  onKeyDown={handleKeyDown}
                  className="w-full"
                >
                  <PowerKModalHeader
                    activePage={activePage}
                    context={context}
                    onSearchChange={setSearchTerm}
                    searchTerm={searchTerm}
                  />
                  <Command.List className="vertical-scrollbar scrollbar-sm max-h-96 overflow-scroll outline-none">
                    <CommandsListComponent
                      activePage={activePage}
                      context={context}
                      handleCommandSelect={handleCommandSelect}
                      handlePageDataSelection={handlePageDataSelection}
                      isWorkspaceLevel={isWorkspaceLevel}
                      searchTerm={searchTerm}
                      setSearchTerm={setSearchTerm}
                    />
                  </Command.List>
                  {/* Footer hints */}
                  {!hideFooter && (
                    <PowerKModalFooter
                      isWorkspaceLevel={isWorkspaceLevel}
                      projectId={context.params.projectId}
                      onWorkspaceLevelChange={setIsWorkspaceLevel}
                    />
                  )}
                </Command>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition.Root>
````
````new web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
    <ModalCore
      isOpen={isOpen}
      handleClose={onClose}
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
      className="divide-opacity-10 flex flex-col items-center justify-center divide-y divide-subtle-1"
    >
      <Command
        filter={(i18nValue: string, search: string) => {
          if (i18nValue === "no-results") return 1;
          if (i18nValue.toLowerCase().includes(search.toLowerCase())) return 1;
          return 0;
        }}
        shouldFilter={searchTerm.length > 0}
        onKeyDown={handleKeyDown}
        className="w-full"
      >
        <PowerKModalHeader
          activePage={activePage}
          context={context}
          onSearchChange={setSearchTerm}
          searchTerm={searchTerm}
        />
        <Command.List className="vertical-scrollbar scrollbar-sm max-h-96 overflow-scroll outline-none">
          <CommandsListComponent
            activePage={activePage}
            context={context}
            handleCommandSelect={handleCommandSelect}
            handlePageDataSelection={handlePageDataSelection}
            isWorkspaceLevel={isWorkspaceLevel}
            searchTerm={searchTerm}
            setSearchTerm={setSearchTerm}
          />
        </Command.List>
        {/* Footer hints */}
        {!hideFooter && (
          <PowerKModalFooter
            isWorkspaceLevel={isWorkspaceLevel}
            projectId={context.params.projectId}
            onWorkspaceLevelChange={setIsWorkspaceLevel}
          />
        )}
      </Command>
    </ModalCore>
````

`web/apps/web/core/components/settings/mobile/nav.tsx`（修改，4 处）：

````old web/apps/web/core/components/settings/mobile/nav.tsx
  });

````
````new web/apps/web/core/components/settings/mobile/nav.tsx
  });

  // The menu is the page's: it stays below the dialogs' layer (z-30), so a dialog opened over the settings covers it.
````
````old web/apps/web/core/components/settings/mobile/nav.tsx
      <div ref={sidebarRef} className="relative z-50 w-fit">
````
````new web/apps/web/core/components/settings/mobile/nav.tsx
      <div ref={sidebarRef} className="relative z-20 w-fit">
````
````old web/apps/web/core/components/settings/mobile/nav.tsx
          <div className="absolute top-10.5 left-0 z-50">
````
````new web/apps/web/core/components/settings/mobile/nav.tsx
          <div className="absolute top-10.5 left-0 z-20">
````
````old web/apps/web/core/components/settings/mobile/nav.tsx
          className="group z-50 shrink-0"
````
````new web/apps/web/core/components/settings/mobile/nav.tsx
          className="group z-20 shrink-0"
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 303",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 301",
````

`web/packages/i18n/src/locales/en/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/power-k.json
      "report_bug": "Report a bug"
    },
````
````new web/packages/i18n/src/locales/en/power-k.json
      "report_bug": "Report a bug"
    },
    "search_shortcuts": "Search for shortcuts",
````

`web/packages/i18n/src/locales/zh-CN/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/power-k.json
      "report_bug": "报告 Bug"
    },
````
````new web/packages/i18n/src/locales/zh-CN/power-k.json
      "report_bug": "报告 Bug"
    },
    "search_shortcuts": "搜索快捷键",
````

- [ ] **Step 2: 端到端**

`e2e/fixtures/dialogs.ts`（修改，1 处）：

````old e2e/fixtures/dialogs.ts
}

/** Presses Escape on page, which shows one modal dialog, and resolves with whether that dialog closed (closedWithin). */
````
````new e2e/fixtures/dialogs.ts
}

/**
 * Resolves with whether element, shown while a dialog is open, is read out: within a live region (a toast's), and not
 * in the page behind the dialog, which is inert and hidden then (pageBehindHidden). The toast that says why nerve
 * refused what a dialog sent is read while the dialog stays open (A-M2).
 */
export async function announced(element: Locator): Promise<boolean> {
  return element.evaluate(
    (shown) =>
      shown.closest("[aria-live]") !== null &&
      shown.closest("[inert]") === null &&
      shown.closest('[aria-hidden="true"]') === null
  );
}

/** Presses Escape on page, which shows one modal dialog, and resolves with whether that dialog closed (closedWithin). */
````

`e2e/stories/identity/a6-sign-out.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a6-sign-out.spec.ts
import { expectQuietConsole, followAccessToken, watchPage, type PageWatch } from "../../fixtures/browser";
import { saveProfileStep } from "../../fixtures/onboarding-pages";
````
````new e2e/stories/identity/a6-sign-out.spec.ts
import { expectQuietConsole, followAccessToken, watchPage, type PageWatch } from "../../fixtures/browser";
import { closedByEscape, enabledWithin, pageBehindHidden } from "../../fixtures/dialogs";
import { saveProfileStep } from "../../fixtures/onboarding-pages";
import { holdAnswer } from "../../fixtures/settings-pages";
````
````old e2e/stories/identity/a6-sign-out.spec.ts
  await signOutThroughMenu(tabA, email);
````
````new e2e/stories/identity/a6-sign-out.spec.ts
  // The switch's dialog hides the onboarding behind it; while the sign-out is out it cannot be dismissed, its button
  // busy.
  const release = await holdAnswer(tabA, "POST", "/api/v0/auth/logout");
  await accountMenu(tabA, email).click();
  await tabA.getByRole("menuitem", { name: "Wrong e-mail address?" }).click();
  expect(await pageBehindHidden(tabA)).toBe(true);
  await tabA.getByRole("button", { name: "Switch account" }).click();
  expect(await closedByEscape(tabA)).toBe(false);
  expect(await enabledWithin(tabA.getByRole("button", { name: "Switching..." }))).toBe(false);
  await release();
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，2 处）：

````old e2e/stories/project/p5-project-members.spec.ts
import { closedByEscape, closedWithin, enabledWithin } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p5-project-members.spec.ts
import { closedByEscape, closedWithin, enabledWithin, pageBehindHidden } from "../../fixtures/dialogs";
````
````old e2e/stories/project/p5-project-members.spec.ts
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
````
````new e2e/stories/project/p5-project-members.spec.ts
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
  // the list opens outside the page behind the dialog, which is hidden: its options are picked all the same
  expect(await pageBehindHidden(page)).toBe(true);
````

`e2e/stories/project/p6-states-page.spec.ts`（修改，3 处）：

````old e2e/stories/project/p6-states-page.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
````
````new e2e/stories/project/p6-states-page.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  announced,
````
````old e2e/stories/project/p6-states-page.spec.ts
  // In Progress made the default meanwhile, elsewhere: nerve refuses its deletion, said in a toast, the dialog open,
````
````new e2e/stories/project/p6-states-page.spec.ts
  // In Progress made the default meanwhile, elsewhere: nerve refuses its deletion, said in a toast read out beside the
  // dialog, which stays open,
````
````old e2e/stories/project/p6-states-page.spec.ts
  await expect(
    page.getByText("The default state cannot be deleted. Make another state the default first.")
  ).toBeVisible();
````
````new e2e/stories/project/p6-states-page.spec.ts
  const refusal = page.getByText("The default state cannot be deleted. Make another state the default first.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
````

`e2e/stories/project/p7-labels-page.spec.ts`（修改，3 处）：

````old e2e/stories/project/p7-labels-page.spec.ts
import { closedByEscape, closedWithin, enabledWithin, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p7-labels-page.spec.ts
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  pageBehindHidden,
} from "../../fixtures/dialogs";
````
````old e2e/stories/project/p7-labels-page.spec.ts
  // Docs deleted meanwhile, elsewhere: nerve refuses its deletion here, said in a toast, the dialog open, Tab going
````
````new e2e/stories/project/p7-labels-page.spec.ts
  // Docs deleted meanwhile, elsewhere: nerve refuses its deletion here, said in a toast read out beside the dialog,
  // which stays open, Tab going
````
````old e2e/stories/project/p7-labels-page.spec.ts
  await expect(page.getByText("The label does not exist, or you cannot see its project.")).toBeVisible();
````
````new e2e/stories/project/p7-labels-page.spec.ts
  const refusal = page.getByText("The label does not exist, or you cannot see its project.");
  await expect(refusal).toBeVisible();
  expect(await announced(refusal)).toBe(true);
````

`e2e/stories/smoke/s6-dialogs.spec.ts`（新文件，102 行）：

````file e2e/stories/smoke/s6-dialogs.spec.ts
import { createWorkspace, slugFor } from "../../fixtures/api";
import { createPAT, emailFor, shownNameOf } from "../../fixtures/auth";
import { expectQuietConsole, EMOJI_CHECK_WARNING, watchPage } from "../../fixtures/browser";
import { closedByEscape, closedWithin, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { tabCycle } from "../../fixtures/project-pages";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// S6, the modal dialogs the workspace's pages open that no other story does (M3 closeout: every dialog is ModalCore,
// Headless UI's Dialog, which makes the page behind it inert and hidden while it is open): the command palette, its
// list of keyboard shortcuts, and a dialog opened over another, the settings dialog's revoke dialog.

test("S6 (page): the command palette and its list of keyboard shortcuts are modal dialogs: while each is open the page behind it is inert and hidden and Tab stays in it; Escape closes each and gives the page back", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  await expect(page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();

  // The palette: its search has the focus, and Tab goes round what it holds.
  await page.keyboard.press("ControlOrMeta+k");
  await expect(modalDialog(page).getByRole("combobox")).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(true);
  const stops = await tabCycle(page, modalDialog(page));
  expect(stops.filter((stop) => stop.startsWith("outside: "))).toEqual([]);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);

  // The shortcuts, named by their title alone: from its search Tab goes to its list, which scrolls, then to its close
  // button, named, and round; no stop outside it.
  await page.keyboard.press("ControlOrMeta+/");
  await expect(modalDialog(page)).toHaveAccessibleName("Keyboard shortcuts");
  const search = page.getByRole("textbox", { name: "Search for shortcuts" });
  await expect(search).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(true);
  const shortcutStops = await tabCycle(page, modalDialog(page), 4);
  expect(shortcutStops).toHaveLength(3);
  expect(shortcutStops[0]).toMatch(/^Work item actions/);
  expect(shortcutStops.slice(1)).toEqual(["Close", "Search for shortcuts"]);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("S6 (page): a dialog opened over another, the revoke dialog over the settings dialog of the user menu: the settings dialog is inert and hidden behind it; Escape closes the top one alone, the focus back where it was, then its tooltip, then the other", async ({
  api,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await createPAT(api, tokens.access_token, { label: "deploy" });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);

  await page
    .getByRole("button", { name: shownNameOf(email) })
    .first()
    .click();
  await page.getByRole("menuitem", { name: "Settings" }).click();
  await modalDialog(page).getByRole("button", { name: "Personal Access Tokens" }).click();
  const revoke = modalDialog(page).getByRole("button", { name: "Revoke token" });
  await revoke.click();
  const dialogs = modalDialog(page);
  await expect(dialogs).toHaveCount(2);
  await expect(dialogs.last().getByRole("button", { name: "Delete" })).toBeVisible();
  // The settings dialog beneath, as the page behind both: inert and hidden.
  expect(
    await dialogs
      .first()
      .evaluate((settings) => settings.closest("[inert]") !== null && settings.closest('[aria-hidden="true"]') !== null)
  ).toBe(true);
  expect(await pageBehindHidden(page)).toBe(true);
  // Escape closes the revoke dialog alone; the settings dialog is back, the button that opened it focused.
  await page.keyboard.press("Escape");
  await expect(dialogs).toHaveCount(1);
  expect(
    await dialogs.evaluate(
      (settings) => settings.closest("[inert]") === null && settings.closest('[aria-hidden="true"]') === null
    )
  ).toBe(true);
  await expect(revoke).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(true);
  // The focus is the keyboard's, so the button's tooltip shows: Escape closes it first, then the settings dialog.
  const tooltip = page.getByText("Revoke token", { exact: true });
  await expect(tooltip).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(tooltip).toHaveCount(0);
  expect(await closedWithin(page)).toBe(false);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 301 条、`@nerve/ui` 14 条、`@nerve/propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 112 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/dialogs.ts e2e/stories/identity/a6-sign-out.spec.ts e2e/stories/project/p5-project-members.spec.ts e2e/stories/project/p6-states-page.spec.ts e2e/stories/project/p7-labels-page.spec.ts e2e/stories/smoke/s6-dialogs.spec.ts web/apps/web/core/components/api-token/modal/create-token-modal.tsx web/apps/web/core/components/onboarding/switch-account-modal.tsx web/apps/web/core/components/power-k/ui/modal/shortcuts-root.tsx web/apps/web/core/components/power-k/ui/modal/wrapper.tsx web/apps/web/core/components/settings/mobile/nav.tsx web/apps/web/package.json web/packages/i18n/src/locales/en/power-k.json web/packages/i18n/src/locales/zh-CN/power-k.json
```
```bash
git commit -m "feat(M3/closeout): the command palette, the shortcuts and the account switch are ModalCore dialogs

The three dialogs that used Headless UI's Dialog directly now go through
ModalCore, so the page behind each is inert and hidden while it is open.
The shortcuts' search loses its tabIndex={1}; its title and search read
through t(), and its close button, named, stands beside the title, which
alone names the dialog. The account switch cannot be dismissed while its
sign-out is out. The settings' phone menu sits below the dialogs' layer.
S6 checks the palette, the shortcuts and a dialog opened over another;
the stories check that a refusal shown inside an open dialog is
announced.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 快捷键的搜索框回到 `tabIndex={1}`：Tab 走出弹窗的顺序 | oxlint（`check:lint`）、故事 S6 | 静态；端到端 |
| `T2.2` | 快捷键弹窗的关闭按钮没有名字 | 故事 S6 | 端到端 |
| `T2.3` | 快捷键弹窗不理 Escape 和背景 | 故事 S6 | 端到端 |
| `T2.4` | 切换账户的弹窗在退出的请求路上时被 Escape 关上（F-1） | 故事 A6 | 端到端 |
| `T2.5` | 快捷键弹窗的关闭按钮回到标题里：弹窗的名字成了"Keyboard shortcuts Close" | 故事 S6 | 端到端 |
| `T2.6` | 快捷键的搜索框的 `autoFocus` 不写理由：oxlint 的 `no-autofocus` 超出上限 | oxlint（`check:lint`） | 静态 |
| `T2.7` | 关提示的 Escape 也关上下面的弹窗：设置弹窗跟着提示一起关上（F3 的顺序） | 故事 S6 | 端到端 |

---

### Task 3: 侧边栏的工作区菜单加"离开工作区"；成员页的确认框被拒绝时留着

**Files:**
- Create: `web/apps/web/core/components/workspace/leave-workspace-modal.tsx`
- Modify: `README.md`、`e2e/fixtures/mounts.ts`、`e2e/fixtures/workspace-pages.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`e2e/stories/project/p8-project-preferences.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`、`e2e/stories/workspace/w4-invite-members.spec.ts`、`e2e/stories/workspace/w7-member-management.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx`、`web/apps/web/core/components/modals/workspace-level.tsx`、`web/apps/web/core/components/navigation/top-navigation-root.tsx`、`web/apps/web/core/components/power-k/global-shortcuts.tsx`、`web/apps/web/core/components/power-k/projects-app-provider.tsx`、`web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx`、`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`、`web/apps/web/core/components/workspace/settings/members-list-item.tsx`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`、`web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-membership-changes.ts`、`web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx`、`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`

**Interfaces:**（spec 2.7）
- `workspace/sidebar/workspace-menu-root.tsx`（整个文件）：没有调用方的"sidebar"写法删除（216 → 205 行）；"离开工作区"一项（`Menu.Item`，每个角色），确认框挂在菜单之外；菜单的按钮在键盘取得焦点时有 `@nerve/ui` 的选择框那样的内圈轮廓（Headless UI 的 `data-focus`），三项在方向键或指针选中时显出底色（`data-focus:bg-layer-transparent-hover`，原来只有指针悬停时）；`workspace/sidebar/dropdown-item.tsx`：工作区的一项同样（别的工作区的显出悬停的底色，当前工作区的从 `active` 深一级到 `selected`；原来当前的那一项选中时不变）；`navigation/top-navigation-root.tsx` 不再传 `variant`。
- `workspace/leave-workspace-modal.tsx`（新文件）：`ConfirmWorkspaceMemberRemove` 加 `useMembershipChanges().leave(done)`。
- `useMembershipChanges` 的 `remove(…, done)`、`leave(done)` 和 `useInvitationChanges` 的删除：`done` 只在成功的跟进中调用（A-M2）；`ConfirmWorkspaceMemberRemove` 不再自己关上，离开的标题经 `t()`；成员和邀请的一行传 `closeDialog`。
- 布局的 key（负责人 2026-10-13 的 B，spec 2.7；总体设计 7.7 的"状态按 id"）：`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`（页头和它的工作区菜单、`GlobalModals`、页面）以路由的 `workspaceSlug` 为 key；项目的页面（`(projects)/projects/(detail)/[projectId]/layout.tsx`，页头的标签栏和它的离开确认框在 `ProjectAuthWrapper` 之外，所以 key 在返回的根上）和项目的设置（`(settings)/settings/projects/[projectId]/layout.tsx`）整个返回的内容以路由的 `projectId` 为 key（`Fragment`）。换工作区、换项目（后退、前进也是）时其下重新挂载，开着的确认框和组件的状态都不带过去；key 都取自路由的参数，不取自 store。key 结束的是组件的状态：开关在命令面板的 store 里（key 之上）的弹窗由渲染它的宿主在卸载时关上，`modals/workspace-level.tsx` 的 `WorkspaceLevelModals`（在工作区的 key 之下，渲染创建项目，`isCreateProjectModalOpen`）以 `useLayoutEffect` 的清理在下一个工作区绘制之前关上它（控制者的裁定，修复轮 3）；M4 的 `WorkItemLevelModals`、M6 的 `ProjectLevelModals` 在项目的 key 之上，交给它们（Task 15）。命令面板和它的快捷键列表同样（开关在 power-K 的 store 里）：`power-k/projects-app-provider.tsx` 的 `ProjectsAppPowerKProvider` 关上面板并清掉它的页（`activePage`），`power-k/global-shortcuts.tsx` 的 `GlobalShortcutsProvider` 关上快捷键列表。
- 文案：`leave_workspace`（common）、`workspace_settings.settings.members.leave_title`。README 的工作区菜单加"离开工作区"。

**Tests:** 两个 hook 的 vitest：`done` 在成功时调用、被拒绝时不调用、换账户之后不调用。W11 的页面版本：访客由键盘从菜单离开（菜单的按钮显出轮廓，打开时选中的当前工作区的一项深一级，方向键选中的一项显出底色、别的不显），双击"离开"只发一次请求，请求在路上时取消不能用、Escape 不关，落到新建工作区的页面；唯一的管理员从菜单离开被拒绝，原因在提示里，确认框留着，取消之后焦点回到菜单的按钮；他的另一个工作区 Beta 的一项在方向键选中时显出底色；从 Beta 经菜单到 acme 打开离开的确认框，后退到 Beta 时它关上、前进回来也不再打开；从 Beta 的成员页经应用的链接到 acme 的成员页打开移出 gus 的确认框，一次后退两步回到 Beta 的成员页时它关上（两者看着工作区的 key）；从 Beta 的项目页到 acme 的（路由自己的移动），Add Project 打开创建的确认框，后退到 Beta 的项目页时它关上（宿主的清理）；命令面板和快捷键列表各以自己的键在 acme 的项目页打开，后退到 Beta 的项目页时各自关上（各自的宿主）。P3：Web 的设置的归档确认框开着，后退到 Ops 的设置（同一个布局）时关上（项目设置的 key）。P8：访客在 Docs 的页头打开离开的确认框，后退到 Web 的工作项时关上（项目页面的 key）。S2：每个角色在最后一页打开工作区菜单，不发请求（`mounts.ts` 的 `WORKSPACE_MENU`，W2 的常驻检查）。W7：唯一的管理员在成员页离开被拒绝时确认框留着。W4：已被删除的邀请再删除被拒绝时确认框留着。

- [ ] **Step 1: 菜单的入口和确认框**

`web/apps/web/app/(all)/[workspaceSlug]/layout.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/layout.tsx
import { WorkspaceAuthWrapper } from "@/layouts/auth-layout/workspace-wrapper";

export default function WorkspaceLayout() {
````
````new web/apps/web/app/(all)/[workspaceSlug]/layout.tsx
import { WorkspaceAuthWrapper } from "@/layouts/auth-layout/workspace-wrapper";
import type { Route } from "./+types/layout";

export default function WorkspaceLayout({ params }: Route.ComponentProps) {
````
````old web/apps/web/app/(all)/[workspaceSlug]/layout.tsx
        <WorkspaceContentWrapper>
````
````new web/apps/web/app/(all)/[workspaceSlug]/layout.tsx
        {/* keyed by the route's workspace (v0 design 7.7, state keyed by id): a switch to another workspace, by a Back
            or Forward too, mounts its header, sidebar, dialogs and page afresh, so no dialog or state opened in one
            workspace carries into the other */}
        <WorkspaceContentWrapper key={params.workspaceSlug}>
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
import { observer } from "mobx-react";
import { Outlet } from "react-router";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
import { Fragment } from "react";
import { observer } from "mobx-react";
import { Outlet } from "react-router";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
  const { preferences: projectPreferences } = useProjectNavigationPreferences();

  return (
    <>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
  const { preferences: projectPreferences } = useProjectNavigationPreferences();

  // keyed by the route's project (v0 design 7.7, state keyed by id): a switch to another project, by a Back or Forward
  // too, mounts its header and its page afresh, so no dialog or state opened in one project (the header's leaving
  // among them, outside the project's wrapper) carries into the other
  return (
    <Fragment key={projectId}>
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
    </>
  );
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
    </Fragment>
  );
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
import { observer } from "mobx-react";
import { Outlet, useLocation } from "react-router";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
import { Fragment } from "react";
import { observer } from "mobx-react";
import { Outlet, useLocation } from "react-router";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
  const { pathname } = useLocation();

  return (
    <>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
  const { pathname } = useLocation();

  // keyed by the route's project (v0 design 7.7, state keyed by id): a switch to another project's settings, by a Back
  // or Forward too, mounts its sidebar and its page afresh, so no dialog or state opened in one project carries into
  // the other
  return (
    <Fragment key={projectId}>
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
    </>
  );
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
    </Fragment>
  );
````

`web/apps/web/core/components/modals/workspace-level.tsx`（修改，2 处）：

````old web/apps/web/core/components/modals/workspace-level.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/core/components/modals/workspace-level.tsx
import { useLayoutEffect } from "react";
import { observer } from "mobx-react";
````
````old web/apps/web/core/components/modals/workspace-level.tsx
  const { isCreateProjectModalOpen, toggleCreateProjectModal } = useCommandPalette();
````
````new web/apps/web/core/components/modals/workspace-level.tsx
  const { isCreateProjectModalOpen, toggleCreateProjectModal } = useCommandPalette();
  // The open flags of the dialogs this host renders (the project's creation, isCreateProjectModalOpen) are the command
  // palette store's, which outlives the workspace's keyed content (v0 design 7.7, state keyed by id): the host closes
  // them as it unmounts with that content, before the next workspace paints, so a switch, a Back or Forward too, never
  // opens one workspace's dialog in another.
  useLayoutEffect(() => () => toggleCreateProjectModal(false), [toggleCreateProjectModal]);
````

`web/apps/web/core/components/navigation/top-navigation-root.tsx`（修改，1 处）：

````old web/apps/web/core/components/navigation/top-navigation-root.tsx
        <WorkspaceMenuRoot variant="top-navigation" />
````
````new web/apps/web/core/components/navigation/top-navigation-root.tsx
        <WorkspaceMenuRoot />
````

`web/apps/web/core/components/power-k/projects-app-provider.tsx`（修改，2 处）：

````old web/apps/web/core/components/power-k/projects-app-provider.tsx
import { useMemo, useState } from "react";
````
````new web/apps/web/core/components/power-k/projects-app-provider.tsx
import { useLayoutEffect, useMemo, useState } from "react";
````
````old web/apps/web/core/components/power-k/projects-app-provider.tsx
  const { activeContext, isPowerKModalOpen, togglePowerKModal, setActivePage } = usePowerK();
````
````new web/apps/web/core/components/power-k/projects-app-provider.tsx
  const { activeContext, isPowerKModalOpen, togglePowerKModal, setActivePage } = usePowerK();
  // The palette's open flag and its page are the power-K store's, which outlives the workspace's keyed content (v0
  // design 7.7, state keyed by id): this host closes the palette and clears its page as it unmounts with that content,
  // before the next page paints, so nothing the palette found in one workspace shows, or acts, in another.
  useLayoutEffect(
    () => () => {
      togglePowerKModal(false);
      setActivePage(null);
    },
    [togglePowerKModal, setActivePage]
  );
````

`web/apps/web/core/components/power-k/global-shortcuts.tsx`（修改，2 处）：

````old web/apps/web/core/components/power-k/global-shortcuts.tsx
import { useEffect, useRef } from "react";
````
````new web/apps/web/core/components/power-k/global-shortcuts.tsx
import { useEffect, useLayoutEffect, useRef } from "react";
````
````old web/apps/web/core/components/power-k/global-shortcuts.tsx
  const { commandRegistry, isShortcutsListModalOpen, setActiveContext, togglePowerKModal, toggleShortcutsListModal } =
    usePowerK();
````
````new web/apps/web/core/components/power-k/global-shortcuts.tsx
  const { commandRegistry, isShortcutsListModalOpen, setActiveContext, togglePowerKModal, toggleShortcutsListModal } =
    usePowerK();
  // The shortcuts list's open flag is the power-K store's, which outlives the workspace's keyed content (v0 design 7.7,
  // state keyed by id): this host closes it as it unmounts with that content, before the next page paints, so a switch,
  // a Back or Forward too, never opens it again in another workspace.
  useLayoutEffect(() => () => toggleShortcutsListModal(false), [toggleShortcutsListModal]);
````

`web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
  onClose: () => void;
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
  onClose: () => void;
  /** Makes the removal or the leaving; settles once the page has followed it: its follow-up of success closes this. */
````
````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
    setIsRemoving(true);

````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
    setIsRemoving(true);
````
````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx

    handleClose();
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
    setIsRemoving(false);
````
````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
  // While the removal is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): it closes once nerve has
  // answered, and no dialog opened again offers the removal while the request is out.
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
  // While the removal or the leaving is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): it closes
  // once nerve has made the change, and no dialog opened again offers it while the request is out. One nerve refuses
  // leaves it open, its buttons enabled again, the reason in the refusal's toast (A-M2).
````
````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
              {currentUser?.id === userDetails.id ? "Leave workspace?" : `Remove ${userDetails?.display_name}?`}
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
              {currentUser?.id === userDetails.id
                ? t("workspace_settings.settings.members.leave_title")
                : `Remove ${userDetails?.display_name}?`}
````

`web/apps/web/core/components/workspace/leave-workspace-modal.tsx`（新文件，40 行）：

````file web/apps/web/core/components/workspace/leave-workspace-modal.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
// hooks
import { useUser } from "@/hooks/store/user";
// local imports
import { ConfirmWorkspaceMemberRemove } from "./confirm-workspace-member-remove";
import { useMembershipChanges } from "./settings/use-membership-changes";

type Props = {
  /** The workspace the caller leaves. */
  workspaceSlug: string;
  isOpen: boolean;
  onClose: () => void;
};

/**
 * The workspace menu's leaving of the workspace of workspaceSlug, for every role (M3 closeout): the dialog and the
 * leaving the members page offers on the caller's own row. Closed once nerve has ended the membership, the root then
 * landing him; one nerve refuses leaves it open, the reason in the refusal's toast: the workspace's only admin's
 * (workspace.sole_admin), or that of the only admin of one of its projects that has other members (project.sole_admin).
 */
export const LeaveWorkspaceModal = observer(function LeaveWorkspaceModal(props: Props) {
  const { workspaceSlug, isOpen, onClose } = props;
  const { data: currentUser } = useUser();
  const { leave } = useMembershipChanges(workspaceSlug);

  if (!currentUser) return null;
  return (
    <ConfirmWorkspaceMemberRemove
      isOpen={isOpen}
      onClose={onClose}
      userDetails={{ id: currentUser.id, display_name: currentUser.display_name }}
      onSubmit={() => leave(onClose)}
    />
  );
});
````

`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
        onSubmit={() => remove(workspaceSlug, invitationDetails.id)}
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
        onSubmit={() => remove(workspaceSlug, invitationDetails.id, () => setRemoveMemberModal(false))}
````

`web/apps/web/core/components/workspace/settings/members-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
  // the caller's own row offers him to leave; another's, to remove that member (an admin's)
  const handleRemove = (memberId: string) => (memberId === currentUser?.id ? leave() : remove(memberId));
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
  // the caller's own row offers him to leave; another's, to remove that member (an admin's); the dialog that asked
  // closes once nerve has done it
  const handleRemove = (memberId: string) => {
    const closeDialog = () => setRemoveMemberModal(null);
    return memberId === currentUser?.id ? leave(closeDialog) : remove(memberId, closeDialog);
  };
````

`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`（修改，3 处）：

````old web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
 * page is that account's, and says nothing of the change.
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
 * page is that account's, and says nothing of the change. The dialog that sent a deletion gives done, its closing,
 * which follows nerve's having made it alone: one nerve refuses leaves it open.
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
    /** Deletes the invitation, and says so. */
    remove: (workspaceSlug: string, invitationId: string) =>
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
    /** Deletes the invitation; once nerve has, done, and says so. */
    remove: (workspaceSlug: string, invitationId: string, done: () => void) =>
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
        done: () =>
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Invitation removed successfully." }),
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
        done: () => {
          done();
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Invitation removed successfully." });
        },
````

`web/apps/web/core/components/workspace/settings/use-membership-changes.ts`（修改，5 处）：

````old web/apps/web/core/components/workspace/settings/use-membership-changes.ts
 * page is that account's, and says nothing of the change.
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.ts
 * page is that account's, and says nothing of the change. A dialog that sent the removal or the leaving gives done, its
 * closing, which follows nerve's having made it alone: one nerve refuses leaves it open.
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.ts
    /** Ends the membership of the member of userId. */
    remove: (userId: string) => followInSession(() => removeMemberFromWorkspace(workspaceSlug, userId), { failed }),
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.ts
    /** Ends the membership of the member of userId; once nerve has, done. */
    remove: (userId: string, done: () => void) =>
      followInSession(() => removeMemberFromWorkspace(workspaceSlug, userId), { done, failed }),
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.ts
     * Ends the caller's own membership; then the root lands him where his workspaces, as they are now, say (M3 design
     * 3.14).
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.ts
     * Ends the caller's own membership; once nerve has, and only then, done, and the root lands him where his
     * workspaces, as they are now, say (M3 design 3.14). Settles once it has.
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.ts
    leave: async () => {
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.ts
    leave: async (done: () => void) => {
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.ts
      await followInSession(() => leaveWorkspace(workspace), { done: () => void navigate("/"), failed });
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.ts
      await followInSession(() => leaveWorkspace(workspace), {
        done: () => {
          done();
          return navigate("/");
        },
        failed,
      });
````

`web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
      <Menu.Item
        as="div"
        className={cn("px-4 py-2", {
          "bg-layer-transparent-active": workspace.id === activeWorkspace?.id,
          "hover:bg-layer-transparent-hover": workspace.id !== activeWorkspace?.id,
        })}
````
````new web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
      {/* the entry shows as the menu marks it, by the arrows as by the pointer (Headless UI's data-focus): a step
          darker than it rests, the current workspace's too */}
      <Menu.Item
        as="div"
        className={cn("px-4 py-2", {
          "bg-layer-transparent-active data-focus:bg-layer-transparent-selected": workspace.id === activeWorkspace?.id,
          "data-focus:bg-layer-transparent-hover": workspace.id !== activeWorkspace?.id,
        })}
````

`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`（整个文件，205 行）：

````whole web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useState, useEffect } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
// icons
import { ChevronDownOutline, LogOutOutline, PlusCircleOutline } from "@makeplane/propel/icons";
// ui
import { Menu, Transition } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import type { Workspace } from "@nerve/api-client";
import { Loader } from "@nerve/ui";
import { orderWorkspacesList, cn } from "@nerve/utils";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserProfile } from "@/hooks/store/user";
import { useInstance } from "@/hooks/store/use-instance";
import { useSignOut } from "@/hooks/use-sign-out";
// components
import { LeaveWorkspaceModal } from "../leave-workspace-modal";
import { WorkspaceLogo } from "../logo";
import SidebarDropdownItem from "./dropdown-item";

/** The workspace menu of the top navigation: the account's workspaces, a new one, leaving this one, signing out. */
export const WorkspaceMenuRoot = observer(function WorkspaceMenuRoot() {
  // store hooks
  const { toggleSidebar, toggleAnySidebarDropdown } = useAppTheme();
  const { config } = useInstance();
  const { data: currentUser } = useUser();
  const signOut = useSignOut();
  const { updateUserProfile } = useUserProfile();
  const { currentWorkspace: activeWorkspace, workspaces } = useWorkspace();
  // derived values
  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
  // translation
  const { t } = useTranslation();
  // local state
  const [isWorkspaceMenuOpen, setIsWorkspaceMenuOpen] = useState(false);
  // the leaving's dialog outlives the menu, which closes as its entry is picked
  const [isLeaveModalOpen, setIsLeaveModalOpen] = useState(false);

  // the workspace opened last is a best-effort preference: the link opens the workspace whether nerve saves it or not
  const handleWorkspaceNavigation = (workspace: Workspace) => {
    void updateUserProfile({ last_workspace_id: workspace?.id }).catch(() => undefined);
  };

  const handleSignOut = () => void signOut();

  const handleItemClick = () => {
    if (window.innerWidth < 768) {
      toggleSidebar();
    }
  };
  const workspacesList = workspaces && orderWorkspacesList(workspaces);

  // Toggle sidebar dropdown state when the menu is open
  useEffect(() => {
    toggleAnySidebarDropdown(isWorkspaceMenuOpen);
  }, [isWorkspaceMenuOpen, toggleAnySidebarDropdown]);

  return (
    <>
      <Menu
        as="div"
        className="relative flex h-full w-fit max-w-48 flex-grow justify-stretch truncate text-left whitespace-nowrap"
      >
        {({ open, close }: { open: boolean; close: () => void }) => {
          // Update local state directly
          if (isWorkspaceMenuOpen !== open) {
            setIsWorkspaceMenuOpen(open);
          }

          return (
            <>
              {/* as the keyboard focuses it (Headless UI's data-focus), the ring @nerve/ui's selects show on their button,
                  inside its edge: the menu's root clips what is outside it */}
              <Menu.Button
                className={cn(
                  "group/menu-button flex flex-grow items-center justify-between gap-1 truncate rounded-sm p-1 text-13 font-medium text-secondary hover:bg-layer-1 focus:outline-none data-focus:outline-1 data-focus:-outline-offset-1 data-focus:outline-accent-strong data-focus:outline-solid",
                  {
                    "bg-layer-1": open,
                  }
                )}
                aria-label={t("aria_labels.projects_sidebar.open_workspace_switcher")}
              >
                <div className="flex flex-grow items-center gap-2 truncate">
                  <WorkspaceLogo
                    logo={activeWorkspace?.logo_url}
                    name={activeWorkspace?.name}
                    classNames="border border-subtle rounded-md size-7"
                  />
                  <h4 className="truncate text-14 font-medium text-primary">{activeWorkspace?.name ?? t("loading")}</h4>
                </div>
                <ChevronDownOutline
                  className={cn("size-4 flex-shrink-0 text-placeholder duration-300", {
                    "rotate-180": open,
                  })}
                />
              </Menu.Button>
              <Transition
                as={Fragment}
                enter="transition ease-out duration-100"
                enterFrom="transform opacity-0 scale-95"
                enterTo="transform opacity-100 scale-100"
                leave="transition ease-in duration-75"
                leaveFrom="transform opacity-100 scale-100"
                leaveTo="transform opacity-0 scale-95"
              >
                <Menu.Items as={Fragment}>
                  <div className="fixed top-10 left-4 z-21 mt-1 flex w-[19rem] origin-top-left flex-col divide-y divide-subtle rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200 outline-none">
                    <div className="vertical-scrollbar scrollbar-sm flex max-h-96 flex-col items-start justify-start overflow-x-hidden overflow-y-scroll">
                      <span className="sticky top-0 z-21 h-full w-full flex-shrink-0 truncate rounded-md bg-surface-1 px-4 pt-3 pb-1 text-left text-13 font-medium text-placeholder">
                        {currentUser?.email}
                      </span>
                      {workspacesList ? (
                        <div className="flex size-full flex-col items-start justify-start">
                          {(activeWorkspace
                            ? [
                                activeWorkspace,
                                ...workspacesList.filter((workspace) => workspace.id !== activeWorkspace?.id),
                              ]
                            : workspacesList
                          ).map((workspace) => (
                            <SidebarDropdownItem
                              key={workspace.id}
                              workspace={workspace}
                              activeWorkspace={activeWorkspace}
                              handleItemClick={handleItemClick}
                              handleWorkspaceNavigation={handleWorkspaceNavigation}
                              handleClose={close}
                            />
                          ))}
                        </div>
                      ) : (
                        <div className="w-full">
                          <Loader className="space-y-2">
                            <Loader.Item height="30px" />
                            <Loader.Item height="30px" />
                          </Loader>
                        </div>
                      )}
                    </div>
                    {/* each item shows as the menu marks it, by the arrows as by the pointer (Headless UI's data-focus) */}
                    <div className="flex w-full flex-col items-start justify-start gap-2 px-4 py-2 text-13">
                      {!isWorkspaceCreationDisabled && (
                        <Link to="/create-workspace" className="w-full">
                          <Menu.Item
                            as="div"
                            className="flex items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-secondary data-focus:bg-layer-transparent-hover"
                          >
                            <PlusCircleOutline className="size-4 flex-shrink-0" />
                            {t("create_workspace")}
                          </Menu.Item>
                        </Link>
                      )}
                      {/* every member may leave the workspace he is in; nerve refuses its only admin (workspace.sole_admin)
                          and the only admin of one of its projects that has other members (project.sole_admin), M3
                          design 3.7 */}
                      {activeWorkspace && (
                        <Menu.Item
                          as="button"
                          type="button"
                          className="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-secondary data-focus:bg-layer-transparent-hover"
                          onClick={() => setIsLeaveModalOpen(true)}
                        >
                          <LogOutOutline className="size-4 flex-shrink-0" />
                          {t("leave_workspace")}
                        </Menu.Item>
                      )}

                      <div className="w-full">
                        <Menu.Item
                          as="button"
                          type="button"
                          className="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-danger-primary data-focus:bg-layer-transparent-hover"
                          onClick={handleSignOut}
                        >
                          <LogOutOutline className="size-4 flex-shrink-0" />
                          {t("sign_out")}
                        </Menu.Item>
                      </div>
                    </div>
                  </div>
                </Menu.Items>
              </Transition>
            </>
          );
        }}
      </Menu>
      {activeWorkspace && (
        <LeaveWorkspaceModal
          workspaceSlug={activeWorkspace.slug}
          isOpen={isLeaveModalOpen}
          onClose={() => setIsLeaveModalOpen(false)}
        />
      )}
    </>
  );
});
````

`web/packages/i18n/src/locales/en/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/common.json
  "leave_project": "Leave project",
````
````new web/packages/i18n/src/locales/en/common.json
  "leave_project": "Leave project",
  "leave_workspace": "Leave workspace",
````

`web/packages/i18n/src/locales/en/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace-settings.json
        "invitations_sent_successfully": "Invitations sent successfully",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
        "invitations_sent_successfully": "Invitations sent successfully",
        "leave_title": "Leave workspace?",
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "leave_project": "离开项目",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "leave_project": "离开项目",
  "leave_workspace": "离开工作区",
````

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "invitations_sent_successfully": "邀请发送成功",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "invitations_sent_successfully": "邀请发送成功",
        "leave_title": "离开工作区？",
````

- [ ] **Step 2: 测试和 README**

`README.md`（修改，1 处）：

````old README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区，再邀请成员（规模选"Just myself"时创建之后就完成），发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速入门指南"，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面（M3/P10 起）：项目列表（卡片复制项目的链接；看得到而还不是成员的项目，卡片上加入）和已归档的项目（恢复、删除）；创建项目（负责人从工作区的管理员和成员中选；表情和图标的选择器，表情的数据由 Nerve 自己提供）；打开项目的地址时，不是成员的看到"加入项目"，看不到的看到"找不到项目"，已归档的看到它已归档；项目设置的 general（名称、标识、说明、可见性、时区、图标；归档、删除）、members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部）、features（迭代、模块、视图、收集箱的开关）和 automations（自动归档）；侧边栏拖动项目的顺序；标签式导航下项目页头的标签栏（默认打开的标签页、收进"更多"）。项目设置的 states 和 labels（M3/P11 起）：项目管理员和是工作区管理员的项目成员增删改、拖动排序，状态还能设为默认，标签能拖进、拖出标签组；项目的其他成员只能看，访客看到没有权限。个人主页（M3/P11 起）：访客打开别人的或自己的主页，看到用户卡片，工作项一栏说他没有权限；管理员和成员看到的工作项列表，与工作项、迭代、模块、视图、收藏、通知、Webhook 等页面一样还在调用 Plane 的接口，Nerve 返回 404，由 M4–M8 对接。
````
````new README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区，再邀请成员（规模选"Just myself"时创建之后就完成），发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速入门指南"，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、离开工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面（M3/P10 起）：项目列表（卡片复制项目的链接；看得到而还不是成员的项目，卡片上加入）和已归档的项目（恢复、删除）；创建项目（负责人从工作区的管理员和成员中选；表情和图标的选择器，表情的数据由 Nerve 自己提供）；打开项目的地址时，不是成员的看到"加入项目"，看不到的看到"找不到项目"，已归档的看到它已归档；项目设置的 general（名称、标识、说明、可见性、时区、图标；归档、删除）、members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部）、features（迭代、模块、视图、收集箱的开关）和 automations（自动归档）；侧边栏拖动项目的顺序；标签式导航下项目页头的标签栏（默认打开的标签页、收进"更多"）。项目设置的 states 和 labels（M3/P11 起）：项目管理员和是工作区管理员的项目成员增删改、拖动排序，状态还能设为默认，标签能拖进、拖出标签组；项目的其他成员只能看，访客看到没有权限。个人主页（M3/P11 起）：访客打开别人的或自己的主页，看到用户卡片，工作项一栏说他没有权限；管理员和成员看到的工作项列表，与工作项、迭代、模块、视图、收藏、通知、Webhook 等页面一样还在调用 Plane 的接口，Nerve 返回 404，由 M4–M8 对接。
````

`e2e/fixtures/mounts.ts`（修改，2 处）：

````old e2e/fixtures/mounts.ts
// What the pages ask nerve for as they load: S2's REQUESTS is built of these lists, and a page's story that checks what
// it loads reads them here rather than writing its own.
````
````new e2e/fixtures/mounts.ts
// What the pages ask nerve for as they load, and the workspace menu as it opens: S2's REQUESTS is built of these lists,
// S2 opens the menu after them, and a page's story that checks what it loads reads them here rather than writing its
// own.
````
````old e2e/fixtures/mounts.ts
export const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];
````
````new e2e/fixtures/mounts.ts
export const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];
/**
 * The workspace menu asks nerve for nothing as it opens, whatever the role (M3 closeout, W2): its entries are the
 * workspaces the app's start loaded (APP), and its leaving asks the dialog's question before it sends anything.
 */
export const WORKSPACE_MENU: string[] = [];
````

`e2e/fixtures/workspace-pages.ts`（修改，1 处）：

````old e2e/fixtures/workspace-pages.ts
/**
 * Ends a membership from the members page, which page shows: opens the menu of the row of email, picks its entry
 * (Leave on the caller's own row, Remove on another's), and confirms it in the dialog that asks. Resolves with nerve's
 * answer to the request of method to path that the page sends.
 */
export async function endMembership(
  page: Page,
  email: string,
  entry: "Leave" | "Remove",
  request: { method: string; path: string }
): Promise<Response> {
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
  await page.getByRole("button", { name: entry, exact: true }).click();
````
````new e2e/fixtures/workspace-pages.ts
/**
 * Asks to end a membership from the members page, which page shows: opens the menu of the row of email and picks its
 * entry (Leave on the caller's own row, Remove on another's), which opens the dialog that asks.
 */
export async function askToEndMembership(page: Page, email: string, entry: "Leave" | "Remove"): Promise<void> {
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
  await page.getByRole("button", { name: entry, exact: true }).click();
}

/**
 * Ends a membership from the members page, which page shows: asks it (askToEndMembership) and confirms it in the
 * dialog that asks. Resolves with nerve's answer to the request of method to path that the page sends.
 */
export async function endMembership(
  page: Page,
  email: string,
  entry: "Leave" | "Remove",
  request: { method: string; path: string }
): Promise<Response> {
  await askToEndMembership(page, email, entry);
````

`e2e/stories/project/p3-project-settings.spec.ts`（修改，3 处）：

````old e2e/stories/project/p3-project-settings.spec.ts
import { closedByEscape, enabledWithin } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p3-project-settings.spec.ts
import { closedByEscape, closedWithin, enabledWithin, modalDialog } from "../../fixtures/dialogs";
````
````old e2e/stories/project/p3-project-settings.spec.ts
test("P3 (page): another project's general page, reached without leaving the route, shows that project's values", async ({
````
````new e2e/stories/project/p3-project-settings.spec.ts
test("P3 (page): another project's general page, reached without leaving the route, shows that project's values; a dialog opened on one ends as a Back takes the route to the other", async ({
````
````old e2e/stories/project/p3-project-settings.spec.ts
  // Web's read is in the session's cache: the wrapper shows its page at once, the route mounted throughout
  await moveWithinApp(page, `/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#identifier")).toHaveValue("WEB");
````
````new e2e/stories/project/p3-project-settings.spec.ts
  // Web's read is in the session's cache: the wrapper, mounted afresh for Web (the layout's content is keyed by the
  // project), shows its page at once
  await moveWithinApp(page, `/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#identifier")).toHaveValue("WEB");
  // Web's archiving, asked, ends as a Back takes the route to Ops's general page, under the same layout: the dialog
  // does not stay to archive Ops (v0 design 7.7).
  await page.getByRole("button", { name: "Archive", exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(1);
  await page.goBack();
  await expect(page).toHaveURL(`/${slug}/settings/projects/${ops.id}`);
  expect(await closedWithin(page)).toBe(true);
  await expect(page.locator("#name")).toHaveValue("Ops");
````

`e2e/stories/project/p8-project-preferences.spec.ts`（修改，4 处）：

````old e2e/stories/project/p8-project-preferences.spec.ts
import { shownWithin } from "../../fixtures/dialogs";
````
````new e2e/stories/project/p8-project-preferences.spec.ts
import { closedWithin, modalDialog, shownWithin } from "../../fixtures/dialogs";
````
````old e2e/stories/project/p8-project-preferences.spec.ts
/** The projects of page's sidebar, in its order. */
````
````new e2e/stories/project/p8-project-preferences.spec.ts
/** The quick actions' button of the project's header on page: the sidebar's projects have one each too. */
function headerActions(page: Page): Locator {
  return page
    .getByRole("button", { name: "Toggle quick actions menu" })
    .and(page.locator("xpath=//*[not(ancestor::aside)]"));
}

/** The projects of page's sidebar, in its order. */
````
````old e2e/stories/project/p8-project-preferences.spec.ts
test("P8 (page): the admin makes modules the tab Web opens on and moves its views under more, by the menus of its header's tabs, which offer neither until nerve has given his tab bar; he drags his third project first in his sidebar; each lasts after a refresh; Web's guest is offered the menus and the drag as well", async ({
````
````new e2e/stories/project/p8-project-preferences.spec.ts
test("P8 (page): the admin makes modules the tab Web opens on and moves its views under more, by the menus of its header's tabs, which offer neither until nerve has given his tab bar; he drags his third project first in his sidebar; each lasts after a refresh; Web's guest is offered the menus and the drag as well, and his leaving of Docs, asked in its header, ends as a Back takes him to Web's work items", async ({
````
````old e2e/stories/project/p8-project-preferences.spec.ts
    errors: ["Failed to load resource: the server responded with a status of 404 (Not Found)"],
  });
  await theGuest.close();
````
````new e2e/stories/project/p8-project-preferences.spec.ts
    errors: ["Failed to load resource: the server responded with a status of 404 (Not Found)"],
  });
  // His sidebar's Docs, under the same layout as Web's work items, offers a guest his leaving in its header; asked,
  // it ends as a Back takes him to Web's work items, rather than ask it of Web (v0 design 7.7: a project's pages are
  // keyed by it).
  await inSidebar(theGuest.page, docs.id).locator("a").first().click();
  await expect(theGuest.page).toHaveURL(`/${slug}/projects/${docs.id}/issues`);
  await headerActions(theGuest.page).click();
  await theGuest.page.getByRole("menuitem", { name: "Leave project" }).click();
  await expect(modalDialog(theGuest.page)).toHaveCount(1);
  await theGuest.page.goBack();
  await expect(theGuest.page).toHaveURL(`/${slug}/projects/${web.id}/issues`);
  expect(await closedWithin(theGuest.page)).toBe(true);
  expect(gusWatch.pageErrors).toEqual([]);
  await theGuest.close();
````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，4 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
import { APP, ARCHIVED, GENERAL, INVITATIONS, PROJECT, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
````
````new e2e/stories/smoke/s2-web-app.spec.ts
import {
  APP,
  ARCHIVED,
  GENERAL,
  INVITATIONS,
  PROJECT,
  PROJECT_MEMBER,
  WORKSPACE,
  WORKSPACE_MENU,
  valued,
} from "../../fixtures/mounts";
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  test(`S2: the workspace's ${account} signs in and opens ${visits.map(([path]) => path).join(", then ")}`, async ({
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  test(`S2: the workspace's ${account} signs in and opens ${visits.map(([path]) => path).join(", then ")}, then the workspace menu`, async ({
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    else await expect(join).toHaveCount(0);
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    else await expect(join).toHaveCount(0);
    // The workspace menu, opened on that page with its leaving, asks for what WORKSPACE_MENU lists (nothing).
    const menu = { from: requests.next(), expected: WORKSPACE_MENU.toSorted() };
    await page.getByRole("button", { name: "Open workspace switcher" }).click();
    await expect(page.getByRole("menuitem", { name: "Leave workspace" })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menuitem", { name: "Leave workspace" })).toHaveCount(0);
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    // Nothing came after any list was whole.
    for (const [index, { from, expected }] of loads.entries()) {
      expect(requests.between(from, loads[index + 1]?.from)).toEqual({ pending: 0, requests: expected });
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    // Nothing came after any list was whole, the menu's last.
    const lists = [...loads, menu];
    for (const [index, { from, expected }] of lists.entries()) {
      expect(requests.between(from, lists[index + 1]?.from)).toEqual({ pending: 0, requests: expected });
````

`e2e/stories/workspace/w11-guest-bounds.spec.ts`（修改，5 处）：

````old e2e/stories/workspace/w11-guest-bounds.spec.ts
import { workspaceTables } from "../../fixtures/assert/workspace";
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
import { expectMembership, expectMembershipEnded, workspaceTables } from "../../fixtures/assert/workspace";
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
import {
  announced,
  closedByEscape,
  closedWithin,
  modalDialog,
  pageBehindHidden,
  shownWithin,
} from "../../fixtures/dialogs";
import { answerTo, holdAnswer, moveWithinApp, registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, askToEndMembership } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
// its page version what a guest's pages show him.
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
// its page version what a guest's pages show him, and his leaving.
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state and label settings tell him he may not see them; no page he opens shows an address, his own neither", async ({
  api,
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state and label settings tell him he may not see them; no page he opens shows an address, his own neither; the workspace menu offers him his leaving, reached by the keyboard and sent once by a double click, its dialog held while it is out, which ends his memberships and lands him on the page that creates a workspace; the only admin's leaving there is refused with the reason, its dialog open; his leaving's dialog, a removal's, a project's creation, the command palette and its shortcuts end as a Back takes him to another workspace", async ({
  api,
  baseURL,
  browser,
  db,
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
  await opened(`/${slug}/settings/projects/${web.id}/labels`, denied);
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
  await opened(`/${slug}/settings/projects/${web.id}/labels`, denied);

  // The admin's dialogs end with the workspace they were opened in, whose content is keyed by it (v0 design 7.7); he
  // has another, Beta, whose pages share acme's layout. From Beta's home his menu's acme entry, a link of the app's,
  // takes him to acme's, where he asks his leaving: a Back to Beta's home ends that dialog rather than ask it of Beta,
  // and a Forward to acme's does not bring it back. Then, from Beta's members page, the app takes him to acme's (the
  // acme entry, then its "Invite members"), where he asks gus's removal: a Back of two entries at once (the Back
  // button's list) to Beta's members page ends that dialog too.
  const betaSlug = slugFor(testInfo, "beta");
  await createWorkspace(api, admin.access_token, { name: "Beta", slug: betaSlug });
  const theAdmin = await anotherBrowser(browser, baseURL ?? "", admin);
  const adminsMenu = theAdmin.page.getByRole("button", { name: "Open workspace switcher" });
  /** Opens the admin's menu and follows the link of its entry named to the path given. */
  const throughMenu = async (role: "menuitem" | "link", name: string | RegExp, path: string) => {
    await adminsMenu.click();
    await theAdmin.page.getByRole("menu").getByRole(role, { name, exact: true }).click();
    await expect(theAdmin.page).toHaveURL(path);
  };
  await theAdmin.page.goto(`/${betaSlug}`);
  await throughMenu("menuitem", /Acme/, `/${slug}`);
  await adminsMenu.click();
  await theAdmin.page.getByRole("menuitem", { name: "Leave workspace" }).click();
  await expect(modalDialog(theAdmin.page)).toHaveCount(1);
  await theAdmin.page.goBack();
  await expect(theAdmin.page).toHaveURL(`/${betaSlug}`);
  expect(await closedWithin(theAdmin.page)).toBe(true);
  await theAdmin.page.goForward();
  await expect(theAdmin.page).toHaveURL(`/${slug}`);
  expect(await shownWithin(modalDialog(theAdmin.page))).toBe(false);
  await throughMenu("menuitem", /Beta/, `/${betaSlug}`);
  await throughMenu("link", "Invite members", `/${betaSlug}/settings/members`);
  await throughMenu("menuitem", /Acme/, `/${slug}`);
  await throughMenu("link", "Invite members", `/${slug}/settings/members`);
  await askToEndMembership(theAdmin.page, gusEmail, "Remove");
  await expect(modalDialog(theAdmin.page)).toHaveCount(1);
  await theAdmin.page.evaluate(() => window.history.go(-2));
  await expect(theAdmin.page).toHaveURL(`/${betaSlug}/settings/members`);
  expect(await closedWithin(theAdmin.page)).toBe(true);
  // So does the projects page's creation, whose open flag the command palette's store holds, above the key: its host
  // closes it as acme's content goes. From Beta's projects page to acme's (the router's own move), Add Project, and a
  // Back to Beta's: the dialog does not open there, to create the project in Beta.
  await moveWithinApp(theAdmin.page, `/${betaSlug}/projects`);
  await moveWithinApp(theAdmin.page, `/${slug}/projects`);
  await theAdmin.page.getByRole("button", { name: "Add Project", exact: true }).click();
  await expect(modalDialog(theAdmin.page)).toHaveCount(1);
  await theAdmin.page.goBack();
  await expect(theAdmin.page).toHaveURL(`/${betaSlug}/projects`);
  expect(await closedWithin(theAdmin.page)).toBe(true);
  // So do the command palette and its list of shortcuts, whose open flags the power-K store holds, each by its own
  // host: opened by its keys on acme's projects page, each ends with the Back to Beta's.
  /** From Beta's projects page to acme's, opens a dialog by keys, Backs to Beta's: whether the dialog ended. */
  const endsOnBack = async (keys: string) => {
    await moveWithinApp(theAdmin.page, `/${slug}/projects`);
    await expect(theAdmin.page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();
    await theAdmin.page.keyboard.press(keys);
    await expect(modalDialog(theAdmin.page)).toHaveCount(1);
    await theAdmin.page.goBack();
    await expect(theAdmin.page).toHaveURL(`/${betaSlug}/projects`);
    return closedWithin(theAdmin.page);
  };
  expect(await endsOnBack("ControlOrMeta+k"), "the palette").toBe(true);
  expect(await endsOnBack("ControlOrMeta+/"), "the shortcuts list").toBe(true);

  // The only admin's leaving, through the workspace menu: nerve refuses it, the reason read beside the dialog, which
  // stays until Cancel closes it and gives the focus back to the menu's button; he is acme's admin still. His menu
  // lists Beta after acme: the arrows mark its entry as they reach it, as the pointer does.
  const leaving = { method: "POST", path: `/api/v0/workspaces/${slug}/leave` };
  await throughMenu("menuitem", /Acme/, `/${slug}`);
  await adminsMenu.click();
  const beta = theAdmin.page.getByRole("menuitem", { name: "Beta" });
  const betaMarked = () => beta.evaluate((entry) => getComputedStyle(entry).backgroundColor !== "rgba(0, 0, 0, 0)");
  expect(await betaMarked()).toBe(false);
  await theAdmin.page.keyboard.press("ArrowDown");
  await theAdmin.page.keyboard.press("ArrowDown");
  await expect.poll(betaMarked).toBe(true);
  await theAdmin.page.getByRole("menuitem", { name: "Leave workspace" }).click();
  const adminLeaving = await answerTo(theAdmin.page, leaving.method, leaving.path, () =>
    modalDialog(theAdmin.page).getByRole("button", { name: "Leave", exact: true }).click()
  );
  expect(adminLeaving.status()).toBe(409);
  const reason = theAdmin.page.getByText(
    "The workspace would be left without an admin. Make another member an admin first."
  );
  await expect(reason).toBeVisible();
  expect(await announced(reason)).toBe(true);
  expect(await closedWithin(theAdmin.page)).toBe(false);
  await modalDialog(theAdmin.page).getByRole("button", { name: "Cancel" }).click();
  await expect(modalDialog(theAdmin.page)).toHaveCount(0);
  await expect(adminsMenu).toBeFocused();
  await expect(theAdmin.page).toHaveURL(`/${slug}`);
  await expectMembership(db, slug, adminEmail, { role: 20, is_active: true });
  await theAdmin.close();

  // gus's leaving, by the keyboard: the menu's button, its ring shown, opens the menu on its first item, acme's entry,
  // the current workspace's, marked a step darker than it rests; then his leaving, from the last item, signing out, one
  // up, past it to creating a workspace and back, each action marked as the arrows reach it, the others not; the
  // dialog hides the page behind it. He leaves with a double click, nerve's answer held: while it is out the dialog
  // cannot be dismissed (Cancel is disabled, Escape leaves it open), and the page sends the leaving once. Leaving ends
  // his memberships, Web's too, and the root lands him on the page that creates a workspace.
  const menu = page.getByRole("button", { name: "Open workspace switcher" });
  await menu.focus();
  await expect(menu).toHaveCSS("outline-style", "solid");
  await page.keyboard.press("Enter");
  const acme = page.getByRole("menuitem", { name: "Acme" });
  const acmeBackground = () => acme.evaluate((entry) => getComputedStyle(entry).backgroundColor);
  await expect(acme).toHaveAttribute("data-focus");
  const acmeMarked = await acmeBackground();
  const actions = ["Create workspace", "Leave workspace", "Sign out"].map((name) =>
    page.getByRole("menuitem", { name, exact: true })
  );
  const marked = () =>
    Promise.all(
      actions.map((action) => action.evaluate((item) => getComputedStyle(item).backgroundColor !== "rgba(0, 0, 0, 0)"))
    );
  await page.keyboard.press("End");
  await expect.poll(marked).toEqual([false, false, true]);
  expect(await acmeBackground()).not.toBe(acmeMarked);
  await page.keyboard.press("ArrowUp");
  await expect.poll(marked).toEqual([false, true, false]);
  await page.keyboard.press("ArrowUp");
  await expect.poll(marked).toEqual([true, false, false]);
  await page.keyboard.press("ArrowDown");
  await expect.poll(marked).toEqual([false, true, false]);
  await page.keyboard.press("Enter");
  const leave = modalDialog(page).getByRole("button", { name: "Leave", exact: true });
  await expect(modalDialog(page).getByRole("heading", { name: "Leave workspace?" })).toBeVisible();
  expect(await pageBehindHidden(page)).toBe(true);
  const release = await holdAnswer(page, leaving.method, leaving.path);
  const left = answerTo(page, leaving.method, leaving.path, () => leave.dblclick());
  await expect(modalDialog(page).getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  await release();
  expect((await left).status()).toBe(204);
  await expect(page).toHaveURL("/create-workspace");
  expect(watch.apiRequests.filter((request) => request === `POST ${leaving.path}`)).toEqual([`POST ${leaving.path}`]);
  await expectMembershipEnded(db, slug, gusEmail, gusEmail, 5, [{ identifier: "WEB", role: 5 }]);
````

`e2e/stories/workspace/w4-invite-members.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w4-invite-members.spec.ts
import { closedByEscape } from "../../fixtures/dialogs";
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
import { announced, closedByEscape, closedWithin, modalDialog } from "../../fixtures/dialogs";
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
  // frank's invitation, deleted meanwhile through the API: the page's removal is refused, and says why.
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
  // frank's invitation, deleted meanwhile through the API: the page's removal is refused, and says why, read beside
  // the dialog, which stays until Cancel closes it.
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
  await expect(page.getByText("The invitation does not exist, or its link is not valid.")).toBeVisible();
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
  const notFound = page.getByText("The invitation does not exist, or its link is not valid.");
  await expect(notFound).toBeVisible();
  expect(await announced(notFound)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await modalDialog(page).getByRole("button", { name: "Cancel" }).click();
  await expect(modalDialog(page)).toHaveCount(0);
````

`e2e/stories/workspace/w7-member-management.spec.ts`（修改，5 处）：

````old e2e/stories/workspace/w7-member-management.spec.ts
import { closedByEscape, pageBehindHidden } from "../../fixtures/dialogs";
````
````new e2e/stories/workspace/w7-member-management.spec.ts
import { announced, closedByEscape, closedWithin, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  // carol, a member, sees each role as text; her own row's menu offers her leaving.
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  // carol, a member, sees each role as text; her own row's menu offers her leaving, and so does the workspace menu.
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  await expect(theMember.page.getByRole("button", { name: "Leave", exact: true })).toBeVisible();
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  await expect(theMember.page.getByRole("button", { name: "Leave", exact: true })).toBeVisible();
  await theMember.page.getByRole("button", { name: "Open workspace switcher" }).click();
  await expect(theMember.page.getByRole("menuitem", { name: "Leave workspace" })).toBeVisible();
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  // His leaving, as the only admin: nerve refuses it, and the page says why and stays.
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  // His leaving, as the only admin: nerve refuses it, and the page says why, read beside the dialog, which stays
  // until Cancel closes it.
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  await expect(
    page.getByText("The workspace would be left without an admin. Make another member an admin first.")
  ).toBeVisible();
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  const reason = page.getByText("The workspace would be left without an admin. Make another member an admin first.");
  await expect(reason).toBeVisible();
  expect(await announced(reason)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  await modalDialog(page).getByRole("button", { name: "Cancel" }).click();
  await expect(modalDialog(page)).toHaveCount(0);
````

`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`（修改，7 处）：

````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
const page = vi.hoisted(() => ({ updateMemberInvitation: vi.fn(), deleteMemberInvitation: vi.fn() }));
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
const page = vi.hoisted(() => ({
  updateMemberInvitation: vi.fn(),
  deleteMemberInvitation: vi.fn(),
  closeDialog: vi.fn(),
}));
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    make: () => useInvitationChanges().remove("acme", "i-dan"),
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    make: () => useInvitationChanges().remove("acme", "i-dan", page.closeDialog),
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    store.mockResolvedValue(undefined);
  }
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    store.mockResolvedValue(undefined);
  }
  page.closeDialog.mockReset();
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
  it("deletes an invitation of the workspace, and says so", async () => {
    await useInvitationChanges().remove("acme", "i-dan");
    expect(page.deleteMemberInvitation.mock.calls).toEqual([["acme", "i-dan"]]);
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
  it("deletes an invitation of the workspace, and says so, its dialog closed once nerve has", async () => {
    const deletion = heldChange<undefined>();
    page.deleteMemberInvitation.mockReturnValueOnce(deletion.sent);
    const deleted = useInvitationChanges().remove("acme", "i-dan", page.closeDialog);
    expect(page.deleteMemberInvitation.mock.calls).toEqual([["acme", "i-dan"]]);
    expect([page.closeDialog.mock.calls, toasts]).toEqual([[], []]);
    deletion.answer(undefined);
    await deleted;
    expect(page.closeDialog.mock.calls.length).toBe(1);
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
  it.each(changes)("shows nerve's reason when it refuses $change", async ({ store, make }) => {
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
  it.each(changes)("shows nerve's reason when it refuses $change, its dialog open", async ({ store, make }) => {
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    await make();
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    await make();
    expect(page.closeDialog.mock.calls).toEqual([]);
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
        expect(toasts).toEqual([]);
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
        expect([page.closeDialog.mock.calls, toasts]).toEqual([[], []]);
````

`web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts`（修改，9 处）：

````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
import { heldChange, lateSettlings, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  navigate: vi.fn(),
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  navigate: vi.fn(),
  closeDialog: vi.fn(),
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
    make: () => useMembershipChanges("acme").remove("u-bob"),
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
    make: () => useMembershipChanges("acme").remove("u-bob", page.closeDialog),
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  { change: "the leaving", store: page.leaveWorkspace, make: () => useMembershipChanges("acme").leave() },
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  {
    change: "the leaving",
    store: page.leaveWorkspace,
    make: () => useMembershipChanges("acme").leave(page.closeDialog),
  },
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  page.navigate.mockClear();
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  page.navigate.mockReset();
  page.closeDialog.mockReset();
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  it("removes a member of the workspace, and stays", async () => {
    await useMembershipChanges("acme").remove("u-bob");
    expect(page.removeMemberFromWorkspace.mock.calls).toEqual([["acme", "u-bob"]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  it("removes a member of the workspace, and stays, its dialog closed once nerve has", async () => {
    const removal = heldChange<undefined>();
    page.removeMemberFromWorkspace.mockReturnValueOnce(removal.sent);
    const removed = useMembershipChanges("acme").remove("u-bob", page.closeDialog);
    expect(page.removeMemberFromWorkspace.mock.calls).toEqual([["acme", "u-bob"]]);
    expect(page.closeDialog).not.toHaveBeenCalled();
    removal.answer(undefined);
    await removed;
    expect([page.closeDialog.mock.calls.length, page.navigate.mock.calls, toasts]).toEqual([1, [], []]);
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  it("leaves the workspace of the address, then lands at the root", async () => {
    await useMembershipChanges("acme").leave();
    expect([page.leaveWorkspace.mock.calls, page.navigate.mock.calls]).toEqual([[[acme]], [["/"]]]);
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  it("leaves the workspace of the address, and closes its dialog and lands at the root only once nerve has made it, settling once it has", async () => {
    const leaving = heldChange<undefined>();
    page.leaveWorkspace.mockReturnValueOnce(leaving.sent);
    const landed = heldChange<undefined>();
    page.navigate.mockReturnValueOnce(landed.sent);
    const left = useMembershipChanges("acme").leave(page.closeDialog);
    const settled = settledYet(left);
    expect(page.leaveWorkspace.mock.calls).toEqual([[acme]]);
    expect([page.closeDialog.mock.calls, page.navigate.mock.calls]).toEqual([[], []]);
    leaving.answer(undefined);
    await vi.waitFor(() => expect(page.navigate.mock.calls).toEqual([["/"]]));
    expect([page.closeDialog.mock.calls.length, settled()]).toEqual([1, false]);
    landed.answer(undefined);
    await left;
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  it.each(changes)("shows nerve's reason when it refuses $change, and stays", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "workspace.sole_admin"));
    await make();
    expect(page.navigate.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_sole_admin" }]);
  });
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
  it.each(changes)(
    "shows nerve's reason when it refuses $change, and stays, its dialog open",
    async ({ store, make }) => {
      store.mockRejectedValueOnce(refusal(409, "workspace.sole_admin"));
      await make();
      expect([page.closeDialog.mock.calls, page.navigate.mock.calls]).toEqual([[], []]);
      expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_sole_admin" }]);
    }
  );
````
````old web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
        expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
````
````new web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
        expect([page.closeDialog.mock.calls, page.navigate.mock.calls, toasts]).toEqual([[], [], []]);
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 301 条、`@nerve/ui` 14 条、`@nerve/propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 112 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add README.md e2e/fixtures/mounts.ts e2e/fixtures/workspace-pages.ts e2e/stories/project/p3-project-settings.spec.ts e2e/stories/project/p8-project-preferences.spec.ts e2e/stories/smoke/s2-web-app.spec.ts e2e/stories/workspace/w11-guest-bounds.spec.ts e2e/stories/workspace/w4-invite-members.spec.ts e2e/stories/workspace/w7-member-management.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/layout.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx' web/apps/web/core/components/modals/workspace-level.tsx web/apps/web/core/components/navigation/top-navigation-root.tsx web/apps/web/core/components/power-k/global-shortcuts.tsx web/apps/web/core/components/power-k/projects-app-provider.tsx web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx web/apps/web/core/components/workspace/leave-workspace-modal.tsx web/apps/web/core/components/workspace/settings/invitations-list-item.tsx web/apps/web/core/components/workspace/settings/members-list-item.tsx web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts web/apps/web/core/components/workspace/settings/use-invitation-changes.ts web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts web/apps/web/core/components/workspace/settings/use-membership-changes.ts web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/workspace-settings.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/workspace-settings.json
```
```bash
git commit -m "feat(M3/closeout): the sidebar's workspace menu offers leaving the workspace, to every role

The menu's new item opens the members page's leaving dialog through the
same command: once nerve has made it, the session that left follows it to
where the members page goes; a refusal (the only admin) keeps the dialog
open with the reason; a double click sends it once, the dialog held while
it is out. The members page's removal, leaving and an invitation's
deletion keep their dialog on a refusal too, closing it in their success
follow-up alone. The menu's button shows a ring, and each of its items, the
workspaces' entries too, a mark, as the keyboard reaches it. The
workspace's content is keyed by its slug, and a project's pages and
settings by the project: a switch, by a Back or Forward too, ends any
dialog opened in the other; the workspace-level dialogs', the command
palette's and its shortcuts list's hosts close the store-held flags they
render as they unmount. Opening the menu requests nothing, which S2
checks for every role. The menu's variant no caller used goes.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 成员页的确认框在 nerve 回答之后就关上，拒绝也是：唯一的管理员被拒绝的离开带走了它的确认框（A-M2） | 故事 W4、故事 W7、故事 W11 | 端到端 |
| `T3.2` | 离开不管 nerve 回答什么都关上确认框：被拒绝也关 | `use-membership-changes.test.ts`、故事 W7、故事 W11 | vitest；端到端 |
| `T3.3` | 菜单只给工作区管理员"离开工作区"：成员和访客没有这一项 | 故事 W7、故事 W11 | 端到端 |
| `T3.4` | "离开工作区"是菜单里的普通按钮，不是菜单项：菜单的方向键越过它，它也不是 `menuitem` | 故事 W7、故事 W11 | 端到端 |
| `T3.5` | 离开的确认框只在菜单打开时存在（像挂在菜单里那样）：选中这一项时菜单关上，确认框跟着没了 | 故事 W11 | 端到端 |
| `T3.6` | 删除邀请不管 nerve 回答什么都关上确认框：被拒绝也关 | `use-invitation-changes.test.ts`、故事 W4 | vitest；端到端 |
| `T3.7` | 确认框的按钮在请求路上时不禁用：双击发两次 | 故事 W11 | 端到端 |
| `T3.8` | "离开工作区"一项只在指针悬停时显出底色，像原来那样：方向键选中它时看不出来 | 故事 W11 | 端到端 |
| `T3.9` | 菜单的按钮没有内圈的轮廓，像原来那样只有 `focus:outline-none`：Tab 到达时看不出焦点 | 故事 W11 | 端到端 |
| `T3.10` | 离开的确认框打开、关上时菜单重建（以确认框的状态为 `key`）：打开它的按钮没了，焦点回不去 | 故事 W11 | 端到端 |
| `T3.11` | 菜单中别的工作区的一项只在指针悬停时显出底色，像原来那样：方向键选中它时看不出来 | 故事 W11 | 端到端 |
| `T3.12` | 当前工作区的一项选中时不变深，像原来那样：键盘打开菜单时先选中它，看不出来 | 故事 W11 | 端到端 |
| `T3.13` | 工作区的内容不以 slug 为 key，像原来那样：后退、前进到别的工作区时它留着，开着的确认框（菜单的离开、成员页的移出）跟过去，作用于那个工作区（T3.14 收进这一行） | 故事 W11 | 端到端 |
| `T3.15` | 工作区菜单打开时再取一次工作区列表：原来只有 W2 一次性的清单看得到这个请求 | 故事 S2 | 端到端 |
| `T3.16` | 项目的设置不以项目为 key，像原来那样：后退到另一个项目的设置时它们留着，开着的确认框（归档）跟过去，作用于那个项目 | 故事 P3 | 端到端 |
| `T3.17` | 项目的页面不以项目为 key，像原来那样：后退到另一个项目的页面时页头留着，开着的确认框（离开）跟过去，作用于那个项目 | 故事 P8 | 端到端 |
| `T3.18` | 工作区级弹窗的宿主卸载时不关上它们，像原来那样：项目的创建由 key 之上的命令面板 store 的标志打开，后退到别的工作区时又打开，在那个工作区里创建项目 | 故事 W11 | 端到端 |
| `T3.19` | 命令面板的宿主卸载时不关上它，像原来那样：开关在 key 之上的 power-K store 里，后退到别的工作区时面板又打开，带着它在前一个工作区找到的 | 故事 W11 | 端到端 |
| `T3.20` | 快捷键列表的宿主卸载时不关上它，像原来那样：开关在 key 之上的 power-K store 里，后退到别的工作区时列表又打开 | 故事 W11 | 端到端 |

---

### Task 4: 工作区的页面上的英文经 `t()`；一句话带名字一个键；S7

**Files:**
- Create: `e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/core/components/common/translated.test.tsx`、`web/apps/web/core/components/common/translated.tsx`、`web/apps/web/core/components/onboarding/switch-account-modal.test.tsx`
- Modify: `e2e/stories/workspace/w5-invitation-link.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`、`web/apps/web/app/(all)/workspace-invitations/layout.tsx`、`web/apps/web/app/(all)/workspace-invitations/page.tsx`、`web/apps/web/core/components/auth-screens/not-authorized-view.tsx`、`web/apps/web/core/components/home/user-greetings.tsx`、`web/apps/web/core/components/navigation/project-navigation-dialog.tsx`、`web/apps/web/core/components/navigation/top-nav-power-k.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/onboarding/switch-account-modal.tsx`、`web/apps/web/core/components/settings/project/sidebar/header.tsx`、`web/apps/web/core/components/settings/workspace/sidebar/header.tsx`、`web/apps/web/core/components/sidebar/resizable-sidebar.tsx`、`web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/core/components/workspace/delete-workspace-modal.tsx`、`web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`、`web/apps/web/core/components/workspace/settings/workspace-details.test.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/packages/constants/src/workspace.ts`、`web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/power-k.json`、`web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/power-k.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json`

**Interfaces:**（spec 2.8）
- `components/common/translated.tsx`（新文件）：`Translated({ text, values })`：把一个键的文字按占位切开，占位换成调用方给的节点（名字照旧加粗）。
- `@nerve/constants` 的 `ORGANIZATION_SIZE_I18N_LABEL`（规模的三处）。
- 经 `t()` 的：`workspace-details`、新建工作区的表单和新手引导的一步、删除工作区的弹窗（三段）、删除邀请的提示、成员的一列（离开、移出、已停用）、成员页的标题、移出的标题和说明、项目导航对话框、设置侧边栏的标题、命令面板的占位、邀请页的标题（`PageHead`，布局的 `meta` 删除）、没有权限的页面、"找不到工作区"的标题、首页的问候（`greeting.*`，每个时段一个带 `{name}` 的键，名字是显示名，同其他带名字的句子；`good`、`morning`、`afternoon`、`evening` 删除）、侧边栏的三个名字。两处装饰的图片 `alt=""`。
- 切换账户的那句话是一个带 `{email}` 的键（`onboarding.switch_account.description`，经 `Translated`；`description_before_email`、`description_after_email` 在两种语言中删除）。邀请被拒绝之后的页面说"This invitation was declined."/"这份邀请已被拒绝。"，不再对打开链接的人说"您已忽略"（负责人 2026-10-13 的答复 2，spec 3.2）。

**Tests:** `translated.test.tsx`、`onboarding/switch-account-modal.test.tsx`（新文件：那句话是一个键，地址在其中）；`workspace-details.test.tsx`、`use-invitation-changes.test.ts` 读键。新的故事 S7（`e2e/stories/smoke/s7-chinese.spec.ts`）的第一个：语言是中文的管理员看工作区的首页（问候带他的名字、快捷键弹窗的标题和搜索框）、菜单、general 页和成员页和它们的弹窗。W5：邀请被拒绝之后的页面的新说法。

- [ ] **Step 1: 页面和文案**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace.name} - Members` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const pageTitle = currentWorkspace?.name
    ? t("workspace_settings.settings.members.page_label", { workspace: currentWorkspace.name })
    : undefined;
````

`web/apps/web/app/(all)/workspace-invitations/layout.tsx`（整个文件，11 行）：

````whole web/apps/web/app/(all)/workspace-invitations/layout.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";

export default function WorkspaceInvitationsLayout() {
  return <Outlet />;
}
````

`web/apps/web/app/(all)/workspace-invitations/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/workspace-invitations/page.tsx
import { LogoSpinner } from "@/components/common/logo-spinner";
````
````new web/apps/web/app/(all)/workspace-invitations/page.tsx
import { LogoSpinner } from "@/components/common/logo-spinner";
import { PageHead } from "@/components/core/page-title";
````
````old web/apps/web/app/(all)/workspace-invitations/page.tsx
  return <div className="flex h-full w-full flex-col items-center justify-center px-3">{content()}</div>;
````
````new web/apps/web/app/(all)/workspace-invitations/page.tsx
  return (
    <div className="flex h-full w-full flex-col items-center justify-center px-3">
      <PageHead title={t("workspace_invitations")} />
      {content()}
    </div>
  );
````

`web/apps/web/core/components/auth-screens/not-authorized-view.tsx`（修改，5 处）：

````old web/apps/web/core/components/auth-screens/not-authorized-view.tsx
import { observer } from "mobx-react";
// assets
````
````new web/apps/web/core/components/auth-screens/not-authorized-view.tsx
import { observer } from "mobx-react";
// assets
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/auth-screens/not-authorized-view.tsx
import WorkspaceNotAuthorizedImg from "@/app/assets/auth/workspace-not-authorized.svg?url";
````
````new web/apps/web/core/components/auth-screens/not-authorized-view.tsx
import WorkspaceNotAuthorizedImg from "@/app/assets/auth/workspace-not-authorized.svg?url";
// components
import { PageHead } from "@/components/core/page-title";
````
````old web/apps/web/core/components/auth-screens/not-authorized-view.tsx
  const { section = "general", isProjectView = false, className } = props;
````
````new web/apps/web/core/components/auth-screens/not-authorized-view.tsx
  const { section = "general", isProjectView = false, className } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/auth-screens/not-authorized-view.tsx
    <DefaultLayout className={cn("bg-surface-1", className)}>
````
````new web/apps/web/core/components/auth-screens/not-authorized-view.tsx
    <DefaultLayout className={cn("bg-surface-1", className)}>
      <PageHead title={t("not_authorized_to_view")} />
````
````old web/apps/web/core/components/auth-screens/not-authorized-view.tsx
        <h1 className="text-18 font-medium text-primary">Oops! You are not authorized to view this page</h1>
````
````new web/apps/web/core/components/auth-screens/not-authorized-view.tsx
        <h1 className="text-18 font-medium text-primary">{t("not_authorized_to_view")}</h1>
````

`web/apps/web/core/components/common/translated.tsx`（新文件，31 行）：

````file web/apps/web/core/components/common/translated.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { Children } from "react";
import type { ReactNode } from "react";
import { useTranslation } from "@nerve/i18n";

// what stands in the translated text for a placeholder's node: no translation holds it
const MARK = "\u0000";

type Props = {
  /** The key of a sentence with placeholders, such as "Delete {name}?". */
  i18nKey: string;
  /** The node of each placeholder, such as the name set in bold. */
  values: Record<string, ReactNode>;
};

/**
 * A sentence in the page's language whose placeholders are nodes, not text: one key for the whole sentence, so that
 * each language orders it as it does, and the thing it names set apart within it.
 */
export function Translated(props: Props) {
  const { i18nKey, values } = props;
  const { t } = useTranslation();
  const names = Object.keys(values);
  const text = t(i18nKey, Object.fromEntries(names.map((name) => [name, `${MARK}${name}${MARK}`])));
  // the marks split the text into its own parts and, at odd places, the placeholders' names; Children keys the parts
  return Children.toArray(text.split(MARK).map((part, index) => (index % 2 === 1 ? values[part] : part)));
}
````

`web/apps/web/core/components/home/user-greetings.tsx`（修改，1 处）：

````old web/apps/web/core/components/home/user-greetings.tsx
      <h2 className="text-center text-20 font-semibold">
        {t("good")} {t(greeting)}, {user?.first_name} {user?.last_name}
      </h2>
````
````new web/apps/web/core/components/home/user-greetings.tsx
      <h2 className="text-center text-20 font-semibold">{t(`greeting.${greeting}`, { name: user.display_name })}</h2>
````

`web/apps/web/core/components/navigation/project-navigation-dialog.tsx`（修改，3 处）：

````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    <div className="text-11 text-secondary">
                      Feature tabs will appear as nested items under project and acts as accordion.
                    </div>
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    <div className="text-11 text-secondary">{t("accordion_navigation_description")}</div>
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    <div className="text-11 text-secondary">
                      Feature tabs will appear as horizontal tabs inside a project.
                    </div>
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    <div className="text-11 text-secondary">{t("horizontal_navigation_description")}</div>
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                        <span className="pl-0.5 text-11 text-danger-primary">Minimum value is 1</span>
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                        <span className="pl-0.5 text-11 text-danger-primary">{t("navigation_count_minimum")}</span>
````

`web/apps/web/core/components/navigation/top-nav-power-k.tsx`（修改，3 处）：

````old web/apps/web/core/components/navigation/top-nav-power-k.tsx
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/navigation/top-nav-power-k.tsx
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/navigation/top-nav-power-k.tsx
  const { projectId: routerProjectId, workItem: workItemIdentifier } = params;

````
````new web/apps/web/core/components/navigation/top-nav-power-k.tsx
  const { projectId: routerProjectId, workItem: workItemIdentifier } = params;

  const { t } = useTranslation();
````
````old web/apps/web/core/components/navigation/top-nav-power-k.tsx
            placeholder="Search commands..."
````
````new web/apps/web/core/components/navigation/top-nav-power-k.tsx
            placeholder={t("power_k.search_commands")}
````

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（修改，2 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { ORGANIZATION_SIZE } from "@nerve/constants";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { ORGANIZATION_SIZE, ORGANIZATION_SIZE_I18N_LABEL } from "@nerve/constants";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                        <span className="font-medium">{size}</span>
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                        <span className="font-medium">{t(ORGANIZATION_SIZE_I18N_LABEL[size])}</span>
````

`web/apps/web/core/components/onboarding/switch-account-modal.tsx`（修改，2 处）：

````old web/apps/web/core/components/onboarding/switch-account-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/onboarding/switch-account-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/onboarding/switch-account-modal.tsx
                {t("onboarding.switch_account.description_before_email")}{" "}
                <span className="text-accent-primary">{userData.email}</span>{" "}
                {t("onboarding.switch_account.description_after_email")}
````
````new web/apps/web/core/components/onboarding/switch-account-modal.tsx
                <Translated
                  i18nKey="onboarding.switch_account.description"
                  values={{ email: <span className="text-accent-primary">{userData.email}</span> }}
                />
````

`web/apps/web/core/components/settings/project/sidebar/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/settings/project/sidebar/header.tsx
        <p>Project settings</p>
````
````new web/apps/web/core/components/settings/project/sidebar/header.tsx
        <p>{t("project_settings.sidebar_title")}</p>
````

`web/apps/web/core/components/settings/workspace/sidebar/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
        <p>Workspace settings</p>
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
        <p>{t("workspace_settings.sidebar_title")}</p>
````

`web/apps/web/core/components/sidebar/resizable-sidebar.tsx`（修改，6 处）：

````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
import { usePlatformOS } from "@nerve/hooks";
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
import { usePlatformOS } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
  const { isMobile } = usePlatformOS();
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label="Main sidebar"
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label={t("aria_labels.sidebar.main")}
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
            // onDoubleClick toggle sidebar
            onDoubleClick={() => toggleCollapsed()}
            onMouseDown={(e) => startResizing(e)}
            role="separator"
            aria-label="Resize sidebar"
          />
        </aside>
      </div>
      {/* Peek View */}
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
            // onDoubleClick toggle sidebar
            onDoubleClick={() => toggleCollapsed()}
            onMouseDown={(e) => startResizing(e)}
            role="separator"
            aria-label={t("aria_labels.sidebar.resize")}
          />
        </aside>
      </div>
      {/* Peek View */}
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label="Sidebar peek view"
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label={t("aria_labels.sidebar.peek")}
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
            // onDoubleClick toggle sidebar
            onDoubleClick={() => toggleCollapsed()}
            onMouseDown={(e) => startResizing(e)}
            role="separator"
            aria-label="Resize sidebar"
          />
        </aside>
      </div>
    </>
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
            // onDoubleClick toggle sidebar
            onDoubleClick={() => toggleCollapsed()}
            onMouseDown={(e) => startResizing(e)}
            role="separator"
            aria-label={t("aria_labels.sidebar.resize")}
          />
        </aside>
      </div>
    </>
````

`web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
                : `Remove ${userDetails?.display_name}?`}
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
                : t("workspace_settings.settings.members.remove_title", { name: userDetails.display_name })}
````
````old web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
                  {/* TODO: Add translation here */}
                  Are you sure you want to remove member- <span className="font-bold">{userDetails?.display_name}</span>
                  ? They will no longer have access to this workspace. This action cannot be undone.
````
````new web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx
                  <Translated
                    i18nKey="workspace_settings.settings.members.remove_confirmation"
                    values={{ name: <span className="font-bold">{userDetails.display_name}</span> }}
                  />
````

`web/apps/web/core/components/workspace/create-workspace-form.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import { ORGANIZATION_SIZE } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import { ORGANIZATION_SIZE, ORGANIZATION_SIZE_I18N_LABEL } from "@nerve/constants";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
                    ORGANIZATION_SIZE.find((c) => c === value) ?? (
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
                    value ? (
                      t(ORGANIZATION_SIZE_I18N_LABEL[value])
                    ) : (
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
                      {item}
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
                      {t(ORGANIZATION_SIZE_I18N_LABEL[item])}
````

`web/apps/web/core/components/workspace/delete-workspace-modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/workspace/delete-workspace-modal.tsx
import { cn } from "@nerve/utils";
````
````new web/apps/web/core/components/workspace/delete-workspace-modal.tsx
import { cn } from "@nerve/utils";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/workspace/delete-workspace-modal.tsx
                You are about to delete the workspace{" "}
                <span className="text-body-xs-semibold break-words">{workspace.name}</span>. If you confirm, you will
                lose access to all your work data in this workspace without any way to restore it. Tread very carefully.
````
````new web/apps/web/core/components/workspace/delete-workspace-modal.tsx
                <Translated
                  i18nKey="workspace_settings.settings.general.delete_modal.body"
                  values={{ name: <span className="text-body-xs-semibold break-words">{workspace.name}</span> }}
                />
````
````old web/apps/web/core/components/workspace/delete-workspace-modal.tsx
              <p className="text-body-xs-regular break-words">Type in this workspace&apos;s name to continue.</p>
````
````new web/apps/web/core/components/workspace/delete-workspace-modal.tsx
              <p className="text-body-xs-regular break-words">
                {t("workspace_settings.settings.general.delete_modal.type_name")}
              </p>
````
````old web/apps/web/core/components/workspace/delete-workspace-modal.tsx
                For final confirmation, type <span className="text-body-xs-medium text-primary">{CONFIRMATION} </span>
                below.
````
````new web/apps/web/core/components/workspace/delete-workspace-modal.tsx
                <Translated
                  i18nKey="workspace_settings.settings.general.delete_modal.type_confirmation"
                  values={{ confirmation: <span className="text-body-xs-medium text-primary">{CONFIRMATION}</span> }}
                />
````

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/settings/member-columns.tsx
import { ROLE, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
import { ROLE, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { rowData, workspaceSlug, isAdmin, currentUser, setRemoveMemberModal } = props;
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { rowData, workspaceSlug, isAdmin, currentUser, setRemoveMemberModal } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
                    <DeleteOutline className="size-3.5 align-middle" /> {id === currentUser?.id ? "Leave " : "Remove "}
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
                    <DeleteOutline className="size-3.5 align-middle" />{" "}
                    {id === currentUser?.id ? t("leave") : t("remove")}
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { changeRole } = useMembershipChanges(workspaceSlug);
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { changeRole } = useMembershipChanges(workspaceSlug);
  const { t } = useTranslation();
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
            Suspended
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
            {t("workspace_settings.settings.members.suspended")}
````

`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`（修改，3 处）：

````old web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
import type { WorkspaceRole } from "@nerve/api-client";
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
import type { WorkspaceRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
  const failed = useRefusalToast();
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
  const failed = useRefusalToast();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Invitation removed successfully." });
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("workspace_settings.settings.members.invitation_removed"),
          });
````

`web/apps/web/core/components/workspace/settings/workspace-details.tsx`（修改，7 处）：

````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { ORGANIZATION_SIZE, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import {
  ORGANIZATION_SIZE,
  ORGANIZATION_SIZE_I18N_LABEL,
  EUserPermissions,
  EUserPermissionsLevel,
} from "@nerve/constants";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
          title: "Success!",
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
          title: t("toast.success"),
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
          message: "Workspace updated successfully",
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
          message: t("workspace_settings.settings.general.updated"),
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
    void copyLink(workspace.slug, "Workspace URL copied to the clipboard.");
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
    void copyLink(workspace.slug, t("workspace_settings.settings.general.url_copied"));
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
                  alt="Workspace Logo"
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
                  alt=""
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
                      ORGANIZATION_SIZE.find((c) => c === value) ??
                      t("workspace_settings.settings.general.errors.company_size.select_a_range")
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
                      value
                        ? t(ORGANIZATION_SIZE_I18N_LABEL[value])
                        : t("workspace_settings.settings.general.errors.company_size.select_a_range")
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
                        {item}
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
                        {t(ORGANIZATION_SIZE_I18N_LABEL[item])}
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，2 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { NerveLogo } from "@/components/common/nerve-logo";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { NerveLogo } from "@/components/common/nerve-logo";
import { PageHead } from "@/components/core/page-title";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              <img src={WorkSpaceNotAvailable} className="h-[220px] object-contain object-center" alt="" />
            </div>
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              <img src={WorkSpaceNotAvailable} className="h-[220px] object-contain object-center" alt="" />
            </div>
            <PageHead title={t("workspace_not_found.title")} />
````

`web/packages/constants/src/workspace.ts`（修改，1 处）：

````old web/packages/constants/src/workspace.ts
export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];
````
````new web/packages/constants/src/workspace.ts
export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];

/** The key of each organization size's label in the page's language: nerve keeps the size itself. */
export const ORGANIZATION_SIZE_I18N_LABEL: Record<OrganizationSize, string> = {
  "Just myself": "workspace_creation.form.organization_size.sizes.just_myself",
  "2-10": "workspace_creation.form.organization_size.sizes.2_10",
  "11-50": "workspace_creation.form.organization_size.sizes.11_50",
  "51-200": "workspace_creation.form.organization_size.sizes.51_200",
  "201-500": "workspace_creation.form.organization_size.sizes.201_500",
  "500+": "workspace_creation.form.organization_size.sizes.500_plus",
};
````

`web/packages/i18n/src/locales/en/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/accessibility.json
    },
````
````new web/packages/i18n/src/locales/en/accessibility.json
    },
    "sidebar": {
      "main": "Main sidebar",
      "resize": "Resize sidebar",
      "peek": "Sidebar peek view"
    },
````

`web/packages/i18n/src/locales/en/common.json`（修改，4 处）：

````old web/packages/i18n/src/locales/en/common.json
  "leave_workspace": "Leave workspace",
````
````new web/packages/i18n/src/locales/en/common.json
  "leave_workspace": "Leave workspace",
  "workspace_invitations": "Workspace Invitations",
  "not_authorized_to_view": "Oops! You are not authorized to view this page",
````
````old web/packages/i18n/src/locales/en/common.json
  "good": "Good",
  "morning": "morning",
  "afternoon": "afternoon",
  "evening": "evening",
````
````new web/packages/i18n/src/locales/en/common.json
  "greeting": {
    "morning": "Good morning, {name}",
    "afternoon": "Good afternoon, {name}",
    "evening": "Good evening, {name}"
  },
````
````old web/packages/i18n/src/locales/en/common.json
  "horizontal_navigation_bar": "Tabbed Navigation",
````
````new web/packages/i18n/src/locales/en/common.json
  "horizontal_navigation_bar": "Tabbed Navigation",
  "accordion_navigation_description": "Feature tabs will appear as nested items under project and acts as accordion.",
  "horizontal_navigation_description": "Feature tabs will appear as horizontal tabs inside a project.",
  "navigation_count_minimum": "Minimum value is 1",
````
````old web/packages/i18n/src/locales/en/common.json
      "description_before_email": "If you have signed up via",
      "description_after_email": "un-intentionally, you can switch your account to a different one from here.",
````
````new web/packages/i18n/src/locales/en/common.json
      "description": "If you have signed up via {email} un-intentionally, you can switch your account to a different one from here.",
````

`web/packages/i18n/src/locales/en/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/power-k.json
    "search_shortcuts": "Search for shortcuts",
````
````new web/packages/i18n/src/locales/en/power-k.json
    "search_shortcuts": "Search for shortcuts",
    "search_commands": "Search commands...",
````

`web/packages/i18n/src/locales/en/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project-settings.json
  "project_settings": {
````
````new web/packages/i18n/src/locales/en/project-settings.json
  "project_settings": {
    "sidebar_title": "Project settings",
````

`web/packages/i18n/src/locales/en/workspace-settings.json`（修改，4 处）：

````old web/packages/i18n/src/locales/en/workspace-settings.json
    "page_label": "{workspace} - General settings",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
    "page_label": "{workspace} - General settings",
    "sidebar_title": "Workspace settings",
````
````old web/packages/i18n/src/locales/en/workspace-settings.json
        "update_workspace": "Update workspace",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
        "update_workspace": "Update workspace",
        "updated": "Workspace updated successfully",
        "url_copied": "Workspace URL copied to the clipboard.",
````
````old web/packages/i18n/src/locales/en/workspace-settings.json
          "title": "Are you sure you want to delete this workspace?",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
          "title": "Are you sure you want to delete this workspace?",
          "body": "You are about to delete the workspace {name}. If you confirm, you will lose access to all your work data in this workspace without any way to restore it. Tread very carefully.",
          "type_name": "Type in this workspace's name to continue.",
          "type_confirmation": "For final confirmation, type {confirmation} below.",
````
````old web/packages/i18n/src/locales/en/workspace-settings.json
        "title": "Members",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
        "title": "Members",
        "page_label": "{workspace} - Members",
        "suspended": "Suspended",
        "remove_title": "Remove {name}?",
        "remove_confirmation": "Are you sure you want to remove {name}? They will no longer have access to this workspace. This action cannot be undone.",
        "invitation_removed": "Invitation removed successfully.",
````

`web/packages/i18n/src/locales/en/workspace.json`（修改，2 处）：

````old web/packages/i18n/src/locales/en/workspace.json
        "placeholder": "Select a range"
````
````new web/packages/i18n/src/locales/en/workspace.json
        "placeholder": "Select a range",
        "sizes": {
          "just_myself": "Just myself",
          "2_10": "2-10",
          "11_50": "11-50",
          "51_200": "51-200",
          "201_500": "201-500",
          "500_plus": "500+"
        }
````
````old web/packages/i18n/src/locales/en/workspace.json
      "title": "You declined this invitation.",
````
````new web/packages/i18n/src/locales/en/workspace.json
      "title": "This invitation was declined.",
````

`web/packages/i18n/src/locales/zh-CN/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/accessibility.json
    },
````
````new web/packages/i18n/src/locales/zh-CN/accessibility.json
    },
    "sidebar": {
      "main": "主侧边栏",
      "resize": "调整侧边栏宽度",
      "peek": "侧边栏预览"
    },
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，4 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "leave_workspace": "离开工作区",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "leave_workspace": "离开工作区",
  "workspace_invitations": "工作区邀请",
  "not_authorized_to_view": "抱歉，您没有权限查看这个页面",
````
````old web/packages/i18n/src/locales/zh-CN/common.json
  "good": "早上",
  "morning": "早上",
  "afternoon": "下午",
  "evening": "晚上",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "greeting": {
    "morning": "早上好，{name}",
    "afternoon": "下午好，{name}",
    "evening": "晚上好，{name}"
  },
````
````old web/packages/i18n/src/locales/zh-CN/common.json
  "horizontal_navigation_bar": "标签式导航",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "horizontal_navigation_bar": "标签式导航",
  "accordion_navigation_description": "功能标签显示为项目下的嵌套项，可以展开和收起。",
  "horizontal_navigation_description": "功能标签显示为项目内的横向标签。",
  "navigation_count_minimum": "最小值为 1",
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "description_before_email": "如果您无意中使用",
      "description_after_email": "注册了账户，可以在这里切换到其他账户。",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "description": "如果您无意中使用 {email} 注册了账户，可以在这里切换到其他账户。",
````

`web/packages/i18n/src/locales/zh-CN/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/power-k.json
    "search_shortcuts": "搜索快捷键",
````
````new web/packages/i18n/src/locales/zh-CN/power-k.json
    "search_shortcuts": "搜索快捷键",
    "search_commands": "搜索命令...",
````

`web/packages/i18n/src/locales/zh-CN/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project-settings.json
  "project_settings": {
````
````new web/packages/i18n/src/locales/zh-CN/project-settings.json
  "project_settings": {
    "sidebar_title": "项目设置",
````

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（修改，4 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
    "page_label": "{workspace} - 常规设置",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
    "page_label": "{workspace} - 常规设置",
    "sidebar_title": "工作区设置",
````
````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "update_workspace": "更新工作区",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "update_workspace": "更新工作区",
        "updated": "工作区已更新",
        "url_copied": "工作区网址已复制到剪贴板。",
````
````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
          "title": "确定要删除此工作区吗？",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
          "title": "确定要删除此工作区吗？",
          "body": "您将要删除工作区 {name}。确认之后，您将失去这个工作区中的全部工作数据，无法恢复。请务必谨慎。",
          "type_name": "输入这个工作区的名称以继续。",
          "type_confirmation": "最后确认，请在下方输入 {confirmation}。",
````
````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "title": "成员",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "title": "成员",
        "page_label": "{workspace} - 成员",
        "suspended": "已停用",
        "remove_title": "移出 {name}？",
        "remove_confirmation": "确定要将 {name} 移出工作区吗？对方将不能再访问这个工作区。此操作无法撤销。",
        "invitation_removed": "邀请已删除。",
````

`web/packages/i18n/src/locales/zh-CN/workspace.json`（修改，2 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace.json
        "placeholder": "选择一个范围"
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
        "placeholder": "选择一个范围",
        "sizes": {
          "just_myself": "只有我自己",
          "2_10": "2-10 人",
          "11_50": "11-50 人",
          "51_200": "51-200 人",
          "201_500": "201-500 人",
          "500_plus": "500 人以上"
        }
````
````old web/packages/i18n/src/locales/zh-CN/workspace.json
      "title": "您已忽略这份邀请。",
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
      "title": "这份邀请已被拒绝。",
````

- [ ] **Step 2: 测试**

`e2e/stories/smoke/s7-chinese.spec.ts`（新文件，92 行）：

````file e2e/stories/smoke/s7-chinese.spec.ts
import type { Page, TestInfo } from "@playwright/test";

import { createWorkspace, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
import { bearer, emailFor, shownNameOf, type AuthTokens } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { modalDialog } from "../../fixtures/dialogs";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { memberRow } from "../../fixtures/workspace-pages";

// S7, the workspace's pages in Chinese (M3 closeout: the text Plane wrote in English on M3's pages is in the user's
// language): an account whose language is zh-CN sees its pages, menus and dialogs in Chinese.

/**
 * An admin of this run of the test whose language is Chinese, the home's tour taken already, and Acme, his workspace,
 * of the size "Just myself", with bob, a member.
 */
async function chineseAdmin(
  api: Api,
  testInfo: TestInfo
): Promise<{ admin: AuthTokens; slug: string; bobEmail: string }> {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const bobEmail = emailFor(testInfo, "bob");
  const bob = await registerOnboarded(api, bobEmail);
  const chinese = await api.PATCH("/api/v0/me/profile", {
    body: { language: "zh-CN", is_tour_completed: true },
    headers: bearer(admin.access_token),
  });
  expect(chinese.response.status).toBe(200);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug, organization_size: "Just myself" });
  await inviteAndAccept(api, admin.access_token, slug, { email: bobEmail, token: bob.access_token }, 15);
  return { admin, slug, bobEmail };
}

/** Closes the one modal dialog page shows by its Cancel (取消). */
async function cancel(page: Page): Promise<void> {
  await modalDialog(page).getByRole("button", { name: "取消" }).click();
  await expect(modalDialog(page)).toHaveCount(0);
}

test("S7 (page): an admin whose language is Chinese sees the workspace's home, menu, general and members pages and their dialogs in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug, bobEmail } = await chineseAdmin(api, testInfo);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The home's greeting, his name in it, the search's placeholder, the keyboard shortcuts' dialog (its title, which
  // names it, and its search), the workspace menu's leaving.
  await page.goto(`/${slug}`);
  const greeting = new RegExp(`^(早上好|下午好|晚上好)，${shownNameOf(emailFor(testInfo, "admin"))}$`);
  await expect(page.getByRole("heading", { level: 2 })).toHaveText(greeting);
  await expect(page.getByPlaceholder("搜索命令...")).toBeVisible();
  await page.keyboard.press("ControlOrMeta+/");
  await expect(modalDialog(page)).toHaveAccessibleName("键盘快捷键");
  await expect(modalDialog(page).getByRole("textbox", { name: "搜索快捷键" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await page.getByRole("button", { name: "打开工作空间切换器" }).click();
  await expect(page.getByRole("menuitem", { name: "离开工作区" })).toBeVisible();
  await page.keyboard.press("Escape");

  // The sidebar, by its name; the project navigation's dialog, each mode's description.
  await page.getByRole("complementary", { name: "主侧边栏" }).getByRole("button", { name: "项目导航" }).click();
  await expect(modalDialog(page).getByText("功能标签显示为项目下的嵌套项，可以展开和收起。")).toBeVisible();
  await expect(modalDialog(page).getByText("功能标签显示为项目内的横向标签。")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);

  // The general page: the settings' sidebar, the size, and the deletion's dialog, the workspace's name in it.
  await page.goto(`/${slug}/settings`);
  await expect(page.getByText("工作区设置", { exact: true })).toBeVisible();
  await expect(page.getByText("只有我自己", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "删除", exact: true }).click();
  await expect(modalDialog(page).getByText(/^您将要删除工作区 Acme。/)).toBeVisible();
  await expect(modalDialog(page).getByText("输入这个工作区的名称以继续。")).toBeVisible();
  await expect(modalDialog(page).getByText("最后确认，请在下方输入 delete my workspace。")).toBeVisible();
  await cancel(page);

  // The members page: its title, and the removal's dialog, the member's name in it.
  await page.goto(`/${slug}/settings/members`);
  await expect(page).toHaveTitle("Acme - 成员");
  await memberRow(page, bobEmail).locator("button").first().click();
  await page.getByRole("button", { name: "移除", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText(`移出 ${shownNameOf(bobEmail)}？`);
  await expect(modalDialog(page).getByText(/^确定要将 .+ 移出工作区吗？/)).toBeVisible();
  await cancel(page);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});
````

`e2e/stories/workspace/w5-invitation-link.spec.ts`（修改，1 处）：

````old e2e/stories/workspace/w5-invitation-link.spec.ts
  await expect(page.getByText("You declined this invitation.")).toBeVisible();
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
  await expect(page.getByText("This invitation was declined.")).toBeVisible();
````

`web/apps/web/core/components/common/translated.test.tsx`（新文件，38 行）：

````file web/apps/web/core/components/common/translated.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { Translated } from "./translated";

// A sentence whose placeholders are nodes (M3 closeout): the stand-in for @nerve/i18n fills a key's placeholders as
// ICU does, from two sentences that order them differently.

const sentences: Record<string, string> = {
  remove: "Remove {name} from {workspace}?",
  reversed: "从 {workspace} 移出 {name}？",
};
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({
    t: (key: string, params: Record<string, string>) =>
      sentences[key].replace(/\{(\w+)\}/g, (_, name: string) => params[name]),
  }),
}));

const values = { name: <b>Bob</b>, workspace: <i>Acme</i> };

describe("Translated", () => {
  it("puts each placeholder's node where the sentence has it", () => {
    expect(renderToStaticMarkup(<Translated i18nKey="remove" values={values} />)).toBe(
      "Remove <b>Bob</b> from <i>Acme</i>?"
    );
  });

  it("follows the order of the page's language", () => {
    expect(renderToStaticMarkup(<Translated i18nKey="reversed" values={values} />)).toBe(
      "从 <i>Acme</i> 移出 <b>Bob</b>？"
    );
  });
});
````

`web/apps/web/core/components/onboarding/switch-account-modal.test.tsx`（新文件，32 行）：

````file web/apps/web/core/components/onboarding/switch-account-modal.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { SwitchAccountModal } from "./switch-account-modal";

// The account switch's sentence (M3 closeout): one key, the address set apart within it, so that each language orders
// the sentence as it does. The stand-in for @nerve/i18n says a key, then the values it was given.

vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({
    t: (key: string, values: Record<string, string> = {}) => [key, ...Object.values(values)].join(" "),
  }),
}));
vi.mock("@headlessui/react", () => ({ DialogTitle: ({ children }: { children?: ReactNode }) => <h3>{children}</h3> }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { email: "bob@example.com" } }) }));
vi.mock("@/hooks/use-sign-out", () => ({ useSignOut: () => vi.fn() }));

describe("the account switch", () => {
  it("says its sentence as one key, the address set apart within it", () => {
    const markup = renderToStaticMarkup(<SwitchAccountModal isOpen onClose={() => {}} />);
    expect(markup).toContain(
      'onboarding.switch_account.description <span class="text-accent-primary">bob@example.com</span>'
    );
  });
});
````

`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Invitation removed successfully." }]);
````
````new web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
    expect(toasts).toEqual([
      { type: "success", title: "toast.success", message: "workspace_settings.settings.members.invitation_removed" },
    ]);
````

`web/apps/web/core/components/workspace/settings/workspace-details.test.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
/** Renders the page, has edit change its fields, and clicks the update; settles once the page has followed nerve. */
````
````new web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
/**
 * Renders the page, has edit change its fields (the size by the label the select shows), and clicks the update;
 * settles once the page has followed nerve.
 */
````
````old web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
    await update((form) => Object.assign(form, { name: "Acme Corp", size: "11-50", zone: "Asia/Shanghai" }));
````
````new web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
    const size = "workspace_creation.form.organization_size.sizes.11_50";
    await update((form) => Object.assign(form, { name: "Acme Corp", size, zone: "Asia/Shanghai" }));
````
````old web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Workspace updated successfully" }]);
````
````new web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
    expect(toasts).toEqual([
      { type: "success", title: "toast.success", message: "workspace_settings.settings.general.updated" },
    ]);
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 301 条、`@nerve/ui` 14 条、`@nerve/propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 113 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/smoke/s7-chinese.spec.ts e2e/stories/workspace/w5-invitation-link.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx' 'web/apps/web/app/(all)/workspace-invitations/layout.tsx' 'web/apps/web/app/(all)/workspace-invitations/page.tsx' web/apps/web/core/components/auth-screens/not-authorized-view.tsx web/apps/web/core/components/common/translated.test.tsx web/apps/web/core/components/common/translated.tsx web/apps/web/core/components/home/user-greetings.tsx web/apps/web/core/components/navigation/project-navigation-dialog.tsx web/apps/web/core/components/navigation/top-nav-power-k.tsx web/apps/web/core/components/onboarding/steps/workspace/create.tsx web/apps/web/core/components/onboarding/switch-account-modal.test.tsx web/apps/web/core/components/onboarding/switch-account-modal.tsx web/apps/web/core/components/settings/project/sidebar/header.tsx web/apps/web/core/components/settings/workspace/sidebar/header.tsx web/apps/web/core/components/sidebar/resizable-sidebar.tsx web/apps/web/core/components/workspace/confirm-workspace-member-remove.tsx web/apps/web/core/components/workspace/create-workspace-form.tsx web/apps/web/core/components/workspace/delete-workspace-modal.tsx web/apps/web/core/components/workspace/settings/member-columns.tsx web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts web/apps/web/core/components/workspace/settings/use-invitation-changes.ts web/apps/web/core/components/workspace/settings/workspace-details.test.tsx web/apps/web/core/components/workspace/settings/workspace-details.tsx web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/packages/constants/src/workspace.ts web/packages/i18n/src/locales/en/accessibility.json web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/power-k.json web/packages/i18n/src/locales/en/project-settings.json web/packages/i18n/src/locales/en/workspace-settings.json web/packages/i18n/src/locales/en/workspace.json web/packages/i18n/src/locales/zh-CN/accessibility.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/power-k.json web/packages/i18n/src/locales/zh-CN/project-settings.json web/packages/i18n/src/locales/zh-CN/workspace-settings.json web/packages/i18n/src/locales/zh-CN/workspace.json
```
```bash
git commit -m "feat(M3/closeout): the workspace's pages speak the user's language, a sentence that names something one key

The workspace's general and members pages, their dialogs, the invitation
and no-permission pages, the project navigation dialog, the settings'
sidebar, the palette's placeholder and the home's greeting read their text
through t(), in English and Chinese. Translated puts a sentence's names
where the language has them, bold as before; the greeting is one key per
time of day with the name the pages show, and the account switch's
sentence one key with the address. The declined invitation's page says
the invitation was declined, addressing no one. S7 reads these pages in
Chinese.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 删除工作区的弹窗不管什么语言都用英文要求输入名称 | 故事 S7 | 端到端 |
| `T4.2` | 句子的占位放在文字之后，不在语言放它的地方 | `translated.test.tsx`、故事 S7 | vitest；端到端 |
| `T4.3` | general 页的规模照 nerve 存的值显示，不是用户的语言 | 故事 S7 | 端到端 |
| `T4.4` | 规模的选项是英文：选择框列出 nerve 的值 | `workspace-details.test.tsx` | vitest |
| `T4.5` | 成员页的标题是英文 | 故事 S7 | 端到端 |
| `T4.6` | 移出成员的标题是英文 | 故事 S7 | 端到端 |
| `T4.7` | 问候又是英文的"Good"加时段，不经 `t()` | 故事 S7 | 端到端 |
| `T4.8` | 项目导航对话框的说明是英文 | 故事 S7 | 端到端 |
| `T4.9` | 侧边栏的名字不管什么语言都是英文 | 故事 S7 | 端到端 |
| `T4.10` | 删除邀请的提示是英文 | `use-invitation-changes.test.ts` | vitest |
| `T4.11` | 快捷键弹窗的标题和搜索框又是英文（Task 2 的 `t()`） | 故事 S7 | 端到端 |
| `T4.12` | 切换账户的那句话又是邮箱两边的两个键，语序是英文的 | `switch-account-modal.test.tsx` | vitest |
| `T4.13` | 问候带 Plane 的名和姓，不是页面上显示的名字 | 故事 S7 | 端到端 |
| `T4.14` | 已拒绝的邀请页又对打开链接的人说"You declined this invitation." | 故事 W5 | 端到端 |

---

### Task 5: 项目的弹窗和设置上的英文经 `t()`

**Files:**
- Create: `web/apps/web/core/components/project/project-dialog-texts.test.tsx`、`web/apps/web/core/lib/fake-i18n-values.ts`
- Modify: `e2e/stories/project/p1-create-project.spec.ts`、`e2e/stories/project/p2-visibility.spec.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/core/components/navigation/use-tab-preferences.test.ts`、`web/apps/web/core/components/navigation/use-tab-preferences.ts`、`web/apps/web/core/components/onboarding/switch-account-modal.test.tsx`、`web/apps/web/core/components/project/add-project-members-modal.test.tsx`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/archive-restore-modal.test.tsx`、`web/apps/web/core/components/project/archive-restore-modal.tsx`、`web/apps/web/core/components/project/card.test.tsx`、`web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/confirm-project-member-remove.tsx`、`web/apps/web/core/components/project/delete-project-modal.test.tsx`、`web/apps/web/core/components/project/delete-project-modal.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/project/join-project-modal.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx`、`web/apps/web/core/components/project/member-select.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/components/project/settings/control-section.tsx`、`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx`、`web/apps/web/core/components/project/settings/use-feature-toggle.ts`、`web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/en/project.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/project.json`

**Interfaces:**（spec 2.8）
- 经 `t()` 的（多数经 `Translated`）：删除、离开、加入、归档和恢复项目的弹窗，移出项目成员，添加成员（提示、"Select co-worker"），成员的默认值，功能开关和标签栏的提示，控制区的说明，项目表单（`alt=""`、长度的提示、"Project ID"），卡片的菜单和状态（"Archived"，访客看到的"Joined"，不是成员的人看到的"Join"按钮），成员的下拉的"None"。卡片上设置的链接只有一个 `aria-hidden` 的图标，原来没有名字：以项目为名（`project_dialogs.settings_link`，"{name} settings"，与侧边栏的一行、"Remove X"同一个做法）。归档和恢复的提示按完整的键读（`tools/i18n-keys.mjs` 看得到）；成员下拉框的"无"按页面上的词搜索。新键在 `project.json` 的 `project_dialogs.*` 和 `project-settings.json`。
- 离开时要输入的文字是常量 `CONFIRMATION`（"Leave Project"），名称的规则改为 `required: true`；删除和离开的输入框的提示（共用的 `project_dialogs.confirmation_placeholder`）和离开的不一致提示都收 `{ confirmation: CONFIRMATION }`，文案里不再抄这几个词（原来英文提示让人输入"leave project"，比对却要"Leave Project"，照提示输入就被拒绝）；忙时的"Leaving..."、"Joining..."改用共用的"Leaving"、"Joining"（W17）。

**Tests:** `project-dialog-texts.test.tsx`（新文件：加入、离开、移出的文字）；`add-project-members-modal`、`archive-restore-modal`、`card`、`delete-project-modal`、`use-feature-toggle`、`use-tab-preferences` 的测试读键；`card.test.tsx` 加一个：设置的链接以项目为名、访客的"Joined"、不是成员的人的"Join"（替身 `fake-i18n-values.ts` 给键和它的值，切换账户的测试也改用它）；`project-dialog-texts.test.tsx` 另有自己从成员页离开的一行。P2、P5 照新的忙时文字，P4、P5 核对确认的提示正是要输入的词；P1 以标题找卡片（卡片里设置的链接也以项目为名）；S7 的成员页在负责人的下拉框中输入"无"找到它；S7 的第二个：项目的 general 页和成员页。

- [ ] **Step 1: 弹窗和设置**

`web/apps/web/core/components/navigation/use-tab-preferences.ts`（修改，3 处）：

````old web/apps/web/core/components/navigation/use-tab-preferences.ts
import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
````
````new web/apps/web/core/components/navigation/use-tab-preferences.ts
import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/navigation/use-tab-preferences.ts
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/navigation/use-tab-preferences.ts
  const toastRefusal = useRefusalToast();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/navigation/use-tab-preferences.ts
                title: "Success!",
                message: "Default tab updated successfully.",
````
````new web/apps/web/core/components/navigation/use-tab-preferences.ts
                title: t("toast.success"),
                message: t("project_dialogs.default_tab_updated"),
````

`web/apps/web/core/components/project/add-project-members-modal.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/add-project-members-modal.tsx
          title: "Success!",
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
          title: t("toast.success"),
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
          message: "Members added successfully.",
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
          message: t("project_dialogs.members_added"),
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                                <div className="flex items-center gap-2 py-0.5">Select co-worker</div>
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                                <div className="flex items-center gap-2 py-0.5">
                                  {t("project_dialogs.select_member")}
                                </div>
````

`web/apps/web/core/components/project/archive-restore-modal.tsx`（修改，7 处）：

````old web/apps/web/core/components/project/archive-restore-modal.tsx
// ui
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
// ui
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
  const toastRefusal = useRefusalToast();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
        setToast(
          archive
            ? {
                type: TOAST_TYPE.SUCCESS,
                title: "Archive success",
                message: `${projectDetails.name} has been archived successfully`,
              }
            : {
                type: TOAST_TYPE.SUCCESS,
                title: "Restore success",
                message: `You can find ${projectDetails.name} in your projects.`,
              }
        );
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t(archive ? "project_dialogs.archive.done_title" : "project_dialogs.restore.done_title"),
          message: t(archive ? "project_dialogs.archive.done" : "project_dialogs.restore.done", {
            name: projectDetails.name,
          }),
        });
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
          {archive ? "Archive" : "Restore"} {projectDetails.name}
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
          {t(archive ? "project_dialogs.archive.title" : "project_dialogs.restore.title", {
            name: projectDetails.name,
          })}
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
          {archive
            ? "This project and its work items, cycles and modules will be archived. Its work items won't appear in search. Only project admins can restore the project."
            : "Restoring a project will activate it and make it visible to all members of the project. Are you sure you want to continue?"}
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
          {t(archive ? "project_dialogs.archive.body" : "project_dialogs.restore.body")}
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
            Cancel
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
            {t("cancel")}
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
            {archive ? (isLoading ? "Archiving" : "Archive") : isLoading ? "Restoring" : "Restore"}
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
            {archive
              ? t(isLoading ? "common.archiving" : "archive")
              : t(isLoading ? "project_dialogs.restore.restoring" : "restore")}
````

`web/apps/web/core/components/project/card.tsx`（修改，13 处）：

````old web/apps/web/core/components/project/card.tsx
      title: "Settings",
````
````new web/apps/web/core/components/project/card.tsx
      title: t("settings"),
````
````old web/apps/web/core/components/project/card.tsx
      title: "Join",
````
````new web/apps/web/core/components/project/card.tsx
      title: t("common.join"),
````
````old web/apps/web/core/components/project/card.tsx
      title: "Open in new tab",
````
````new web/apps/web/core/components/project/card.tsx
      title: t("open_in_new_tab"),
````
````old web/apps/web/core/components/project/card.tsx
      title: "Copy link",
````
````new web/apps/web/core/components/project/card.tsx
      title: t("copy_link"),
````
````old web/apps/web/core/components/project/card.tsx
      title: "Restore",
````
````new web/apps/web/core/components/project/card.tsx
      title: t("restore"),
````
````old web/apps/web/core/components/project/card.tsx
      title: "Delete",
````
````new web/apps/web/core/components/project/card.tsx
      title: t("delete"),
````
````old web/apps/web/core/components/project/card.tsx
                  <span className="text-13 text-placeholder italic">No Member Yet</span>
````
````new web/apps/web/core/components/project/card.tsx
                  <span className="text-13 text-placeholder italic">{t("project_dialogs.no_member_yet")}</span>
````
````old web/apps/web/core/components/project/card.tsx
              {isArchived && <div className="text-11 font-medium text-placeholder">Archived</div>}
````
````new web/apps/web/core/components/project/card.tsx
              {isArchived && (
                <div className="text-11 font-medium text-placeholder">{t("project_dialogs.archived")}</div>
              )}
````
````old web/apps/web/core/components/project/card.tsx
                      Restore
````
````new web/apps/web/core/components/project/card.tsx
                      {t("restore")}
````
````old web/apps/web/core/components/project/card.tsx
                    aria-label="Delete"
````
````new web/apps/web/core/components/project/card.tsx
                    aria-label={t("delete")}
````
````old web/apps/web/core/components/project/card.tsx
                      to={`/${workspaceSlug}/settings/projects/${project.id}`}
````
````new web/apps/web/core/components/project/card.tsx
                      to={`/${workspaceSlug}/settings/projects/${project.id}`}
                      aria-label={t("project_dialogs.settings_link", { name: project.name })}
````
````old web/apps/web/core/components/project/card.tsx
                      Joined
````
````new web/apps/web/core/components/project/card.tsx
                      {t("project_dialogs.joined")}
````
````old web/apps/web/core/components/project/card.tsx
                      Join
                    </Button>
````
````new web/apps/web/core/components/project/card.tsx
                      {t("common.join")}
                    </Button>
````

`web/apps/web/core/components/project/confirm-project-member-remove.tsx`（修改，8 处）：

````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
// types
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
// types
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
  const { getProjectById } = useProject();
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
  const { getProjectById } = useProject();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
              {isCurrentUser ? "Leave project?" : `Remove ${data?.display_name}?`}
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
              {isCurrentUser
                ? t("project_dialogs.remove_member.leave_title")
                : t("project_dialogs.remove_member.remove_title", { name: data.display_name })}
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
                  <>
                    Are you sure you want to leave the <span className="font-bold">{currentProjectDetails?.name}</span>{" "}
                    project? You will be able to join the project if a project admin adds you again or if it's public.
                  </>
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
                  <Translated
                    i18nKey="project_dialogs.remove_member.leave_body"
                    values={{ name: <span className="font-bold">{currentProjectDetails?.name}</span> }}
                  />
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
                  <>
                    Are you sure you want to remove member- <span className="font-bold">{data?.display_name}</span>?
                    They will no longer have access to this project. This action cannot be undone.
                  </>
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
                  <Translated
                    i18nKey="project_dialogs.remove_member.remove_body"
                    values={{ name: <span className="font-bold">{data.display_name}</span> }}
                  />
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
          Cancel
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
          {t("cancel")}
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
          {isCurrentUser ? (isDeleteLoading ? "Leaving..." : "Leave") : isDeleteLoading ? "Removing..." : "Remove"}
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
          {isCurrentUser ? t(isDeleteLoading ? "leaving" : "leave") : t(isDeleteLoading ? "removing" : "remove")}
````

`web/apps/web/core/components/project/delete-project-modal.tsx`（修改，12 处）：

````old web/apps/web/core/components/project/delete-project-modal.tsx
import { Input, InputGroup } from "@makeplane/propel/components/input";
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
  const toastRefusal = useRefusalToast();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
          title: "Success!",
          message: "Project deleted successfully.",
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
          title: t("toast.success"),
          message: t("project_dialogs.delete.deleted"),
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            <h3 className="text-18 font-medium 2xl:text-20">Delete project</h3>
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            <h3 className="text-18 font-medium 2xl:text-20">{t("project_dialogs.delete.title")}</h3>
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            Are you sure you want to delete project <span className="font-semibold break-words">{project.name}</span>?
            All of the data related to the project will be permanently removed. This action cannot be undone
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            <Translated
              i18nKey="project_dialogs.delete.body"
              values={{ name: <span className="font-semibold break-words">{project.name}</span> }}
            />
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            Enter the project name <span className="font-medium text-primary">{project.name}</span> to continue:
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            <Translated
              i18nKey="project_dialogs.type_project_name"
              values={{ name: <span className="font-medium text-primary">{project.name}</span> }}
            />
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
                    placeholder="Project name"
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
                    placeholder={t("project_name")}
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            To confirm, type <span className="font-medium text-primary">{CONFIRMATION}</span> below:
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            <Translated
              i18nKey="project_dialogs.type_confirmation"
              values={{ confirmation: <span className="font-medium text-primary">{CONFIRMATION}</span> }}
            />
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
                    placeholder="Enter 'delete my project'"
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
                    placeholder={t("project_dialogs.confirmation_placeholder", { confirmation: CONFIRMATION })}
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            Cancel
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            {t("cancel")}
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            {isSubmitting ? "Deleting" : "Delete project"}
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            {isSubmitting ? t("deleting") : t("project_dialogs.delete.title")}
````

`web/apps/web/core/components/project/form.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/form.tsx
          alt="Project cover image"
````
````new web/apps/web/core/components/project/form.tsx
          alt=""
````
````old web/apps/web/core/components/project/form.tsx
                message: "Project name should be less than 255 characters",
````
````new web/apps/web/core/components/project/form.tsx
                message: t("project_dialogs.name_max_length"),
````
````old web/apps/web/core/components/project/form.tsx
            <h4 className="text-13">Project ID</h4>
````
````new web/apps/web/core/components/project/form.tsx
            <h4 className="text-13">{t("project_id")}</h4>
````

`web/apps/web/core/components/project/join-project-modal.tsx`（修改，7 处）：

````old web/apps/web/core/components/project/join-project-modal.tsx
// types
````
````new web/apps/web/core/components/project/join-project-modal.tsx
// types
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/join-project-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/project/join-project-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/project/join-project-modal.tsx
  const navigate = useNavigate();
````
````new web/apps/web/core/components/project/join-project-modal.tsx
  const navigate = useNavigate();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/join-project-modal.tsx
        <h3 className="text-16 leading-6 font-medium text-primary">Join Project?</h3>
````
````new web/apps/web/core/components/project/join-project-modal.tsx
        <h3 className="text-16 leading-6 font-medium text-primary">{t("project_dialogs.join.title")}</h3>
````
````old web/apps/web/core/components/project/join-project-modal.tsx
          Are you sure you want to join the project <span className="font-semibold break-words">{project?.name}</span>?
          Please click the &apos;Join Project&apos; button below to continue.
````
````new web/apps/web/core/components/project/join-project-modal.tsx
          <Translated
            i18nKey="project_dialogs.join.body"
            values={{ name: <span className="font-semibold break-words">{project.name}</span> }}
          />
````
````old web/apps/web/core/components/project/join-project-modal.tsx
          Cancel
````
````new web/apps/web/core/components/project/join-project-modal.tsx
          {t("cancel")}
````
````old web/apps/web/core/components/project/join-project-modal.tsx
          {isJoiningLoading ? "Joining..." : "Join Project"}
````
````new web/apps/web/core/components/project/join-project-modal.tsx
          {isJoiningLoading ? t("project_dialogs.join.joining") : t("project_dialogs.join.submit")}
````

`web/apps/web/core/components/project/leave-project-modal.tsx`（修改，16 处）：

````old web/apps/web/core/components/project/leave-project-modal.tsx
import { Input, InputGroup } from "@makeplane/propel/components/input";
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
// components
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
  confirmLeave: "",
};
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
  confirmLeave: "",
};

/** The words that confirm the leaving. */
const CONFIRMATION = "Leave Project";
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
  const { leave } = useProjectMembershipChanges(workspaceSlug ?? "", project.id);
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
  const { leave } = useProjectMembershipChanges(workspaceSlug ?? "", project.id);
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
        title: "Error!",
        message: "Please enter the project name as shown in the description.",
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
        title: t("toast.error"),
        message: t("project_dialogs.leave.name_mismatch"),
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
    if (data.confirmLeave !== "Leave Project") {
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
    if (data.confirmLeave !== CONFIRMATION) {
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
        title: "Error!",
        message: "Please confirm leaving the project by typing the 'Leave Project'.",
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
        title: t("toast.error"),
        message: t("project_dialogs.leave.confirmation_mismatch", { confirmation: CONFIRMATION }),
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            <h3 className="text-18 font-medium 2xl:text-20">Leave Project</h3>
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            <h3 className="text-18 font-medium 2xl:text-20">{t("project_dialogs.leave.title")}</h3>
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            Are you sure you want to leave the project -
            <span className="font-medium text-primary">{` "${project?.name}" `}</span>? All of the work items associated
            with you will become inaccessible.
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            <Translated
              i18nKey="project_dialogs.leave.body"
              values={{ name: <span className="font-medium text-primary">{project.name}</span> }}
            />
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            Enter the project name <span className="font-medium text-primary">{project?.name}</span> to continue:
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            <Translated
              i18nKey="project_dialogs.type_project_name"
              values={{ name: <span className="font-medium text-primary">{project.name}</span> }}
            />
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            rules={{
              required: "Label title is required",
            }}
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            rules={{ required: true }}
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
                    placeholder="Enter project name"
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
                    placeholder={t("project_dialogs.leave.name_placeholder")}
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            To confirm, type <span className="font-medium text-primary">Leave Project</span> below:
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            <Translated
              i18nKey="project_dialogs.type_confirmation"
              values={{ confirmation: <span className="font-medium text-primary">{CONFIRMATION}</span> }}
            />
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
                    placeholder="Enter 'leave project'"
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
                    placeholder={t("project_dialogs.confirmation_placeholder", { confirmation: CONFIRMATION })}
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            Cancel
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            {t("cancel")}
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
            {isSubmitting ? "Leaving..." : "Leave Project"}
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
            {isSubmitting ? t("leaving") : t("project_dialogs.leave.title")}
````

`web/apps/web/core/components/project/member-select.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/member-select.tsx
import { Avatar } from "@makeplane/propel/components/avatar";
````
````new web/apps/web/core/components/project/member-select.tsx
import { Avatar } from "@makeplane/propel/components/avatar";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/member-select.tsx
  const { value, onChange, isDisabled = false } = props;
````
````new web/apps/web/core/components/project/member-select.tsx
  const { value, onChange, isDisabled = false } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/member-select.tsx
              <span className="text-13 text-placeholder">None</span>
````
````new web/apps/web/core/components/project/member-select.tsx
              <span className="text-13 text-placeholder">{t("none")}</span>
````
````old web/apps/web/core/components/project/member-select.tsx
                <span className="py-0.5 text-13 text-placeholder">None</span>
````
````new web/apps/web/core/components/project/member-select.tsx
                <span className="py-0.5 text-13 text-placeholder">{t("none")}</span>
````
````old web/apps/web/core/components/project/member-select.tsx
            query: "none",
````
````new web/apps/web/core/components/project/member-select.tsx
            query: t("none"),
````

`web/apps/web/core/components/project/project-settings-member-defaults.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
      <DefaultSettingItem title="Project Lead" description="Select the project lead for the project.">
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
      <DefaultSettingItem
        title={t("project_settings.members.defaults.lead.title")}
        description={t("project_settings.members.defaults.lead.description")}
      >
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
      <DefaultSettingItem title="Default Assignee" description="Select the default assignee for the project.">
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
      <DefaultSettingItem
        title={t("project_settings.members.defaults.assignee.title")}
        description={t("project_settings.members.defaults.assignee.description")}
      >
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
          title="Guest access"
          description="This will allow guests to have view access to all the project work items."
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
          title={t("project_settings.members.defaults.guest_access.title")}
          description={t("project_settings.members.defaults.guest_access.description")}
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
              aria-label="Guest access"
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
              aria-label={t("project_settings.members.defaults.guest_access.title")}
````

`web/apps/web/core/components/project/settings/control-section.tsx`（修改，2 处）：

````old web/apps/web/core/components/project/settings/control-section.tsx
          description="Archiving a project will unlist your project from your side navigation although you will still be able to access it from your projects page. You can restore the project or delete it whenever you want."
````
````new web/apps/web/core/components/project/settings/control-section.tsx
          description={t("project_dialogs.archive.description")}
````
````old web/apps/web/core/components/project/settings/control-section.tsx
          description="When deleting a project, all of the data and resources within that project will be permanently removed and cannot be recovered."
````
````new web/apps/web/core/components/project/settings/control-section.tsx
          description={t("project_dialogs.delete.description")}
````

`web/apps/web/core/components/project/settings/use-feature-toggle.ts`（修改，3 处）：

````old web/apps/web/core/components/project/settings/use-feature-toggle.ts
// nerve imports
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.ts
// nerve imports
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.ts
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.ts
  const toastRefusal = useRefusalToast();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.ts
          title: "Success!",
          message: "Project feature updated successfully.",
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.ts
          title: t("toast.success"),
          message: t("project_dialogs.feature_updated"),
````

`web/packages/i18n/src/locales/en/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project-settings.json
    "members": {
````
````new web/packages/i18n/src/locales/en/project-settings.json
    "members": {
      "defaults": {
        "lead": {
          "title": "Project Lead",
          "description": "Select the project lead for the project."
        },
        "assignee": {
          "title": "Default Assignee",
          "description": "Select the default assignee for the project."
        },
        "guest_access": {
          "title": "Guest access",
          "description": "This will allow guests to have view access to all the project work items."
        }
      },
````

`web/packages/i18n/src/locales/en/project.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project.json
    "role": "Role"
````
````new web/packages/i18n/src/locales/en/project.json
    "role": "Role"
  },
  "project_dialogs": {
    "type_project_name": "Enter the project name {name} to continue:",
    "type_confirmation": "To confirm, type {confirmation} below:",
    "confirmation_placeholder": "Enter “{confirmation}”",
    "delete": {
      "title": "Delete project",
      "body": "Are you sure you want to delete project {name}? All of the data related to the project will be permanently removed. This action cannot be undone",
      "deleted": "Project deleted successfully.",
      "description": "When deleting a project, all of the data and resources within that project will be permanently removed and cannot be recovered."
    },
    "leave": {
      "title": "Leave Project",
      "body": "Are you sure you want to leave the project {name}? All of the work items associated with you will become inaccessible.",
      "name_placeholder": "Enter project name",
      "name_mismatch": "Please enter the project name as shown in the description.",
      "confirmation_mismatch": "Please confirm leaving the project by typing “{confirmation}”."
    },
    "join": {
      "title": "Join Project?",
      "body": "Are you sure you want to join the project {name}? Please click the 'Join Project' button below to continue.",
      "submit": "Join Project",
      "joining": "Joining"
    },
    "remove_member": {
      "leave_title": "Leave project?",
      "remove_title": "Remove {name}?",
      "leave_body": "Are you sure you want to leave the {name} project? You will be able to join the project if a project admin adds you again or if it's public.",
      "remove_body": "Are you sure you want to remove {name}? They will no longer have access to this project. This action cannot be undone."
    },
    "archive": {
      "title": "Archive {name}",
      "body": "This project and its work items, cycles and modules will be archived. Its work items won't appear in search. Only project admins can restore the project.",
      "done_title": "Archive success",
      "done": "{name} has been archived successfully",
      "description": "Archiving a project will unlist your project from your side navigation although you will still be able to access it from your projects page. You can restore the project or delete it whenever you want."
    },
    "restore": {
      "title": "Restore {name}",
      "body": "Restoring a project will activate it and make it visible to all members of the project. Are you sure you want to continue?",
      "done_title": "Restore success",
      "done": "You can find {name} in your projects.",
      "restoring": "Restoring"
    },
    "members_added": "Members added successfully.",
    "select_member": "Select co-worker",
    "feature_updated": "Project feature updated successfully.",
    "default_tab_updated": "Default tab updated successfully.",
    "name_max_length": "Project name should be less than 255 characters",
    "no_member_yet": "No Member Yet",
    "archived": "Archived",
    "joined": "Joined",
    "settings_link": "{name} settings"
````

`web/packages/i18n/src/locales/zh-CN/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project-settings.json
    "members": {
````
````new web/packages/i18n/src/locales/zh-CN/project-settings.json
    "members": {
      "defaults": {
        "lead": {
          "title": "项目负责人",
          "description": "选择这个项目的负责人。"
        },
        "assignee": {
          "title": "默认负责人",
          "description": "选择这个项目工作项的默认负责人。"
        },
        "guest_access": {
          "title": "访客访问",
          "description": "允许访客查看这个项目的全部工作项。"
        }
      },
````

`web/packages/i18n/src/locales/zh-CN/project.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project.json
    "role": "角色"
````
````new web/packages/i18n/src/locales/zh-CN/project.json
    "role": "角色"
  },
  "project_dialogs": {
    "type_project_name": "输入项目名称 {name} 以继续：",
    "type_confirmation": "请在下方输入 {confirmation} 以确认：",
    "confirmation_placeholder": "输入“{confirmation}”",
    "delete": {
      "title": "删除项目",
      "body": "确定要删除项目 {name} 吗？这个项目的全部数据将被永久删除。此操作无法撤销。",
      "deleted": "项目已删除。",
      "description": "删除项目时，项目中的全部数据和资源将被永久删除，且无法恢复。"
    },
    "leave": {
      "title": "离开项目",
      "body": "确定要离开项目 {name} 吗？与您相关的全部工作项将无法访问。",
      "name_placeholder": "输入项目名称",
      "name_mismatch": "请输入说明中显示的项目名称。",
      "confirmation_mismatch": "请输入“{confirmation}”以确认离开项目。"
    },
    "join": {
      "title": "加入项目？",
      "body": "确定要加入项目 {name} 吗？请点击下方的“加入项目”按钮继续。",
      "submit": "加入项目",
      "joining": "正在加入"
    },
    "remove_member": {
      "leave_title": "离开项目？",
      "remove_title": "移出 {name}？",
      "leave_body": "确定要离开项目 {name} 吗？项目管理员再次添加您，或者项目公开时，您可以重新加入。",
      "remove_body": "确定要将 {name} 移出项目吗？对方将不能再访问这个项目。此操作无法撤销。"
    },
    "archive": {
      "title": "归档 {name}",
      "body": "这个项目和它的工作项、迭代、模块将被归档，它的工作项不会出现在搜索中。只有项目管理员可以恢复这个项目。",
      "done_title": "已归档",
      "done": "{name} 已归档",
      "description": "归档项目后，它不再显示在侧边导航中，您仍可以在项目页中访问它。您可以随时恢复或删除这个项目。"
    },
    "restore": {
      "title": "恢复 {name}",
      "body": "恢复项目会重新启用它，项目的全部成员都能看到它。确定要继续吗？",
      "done_title": "已恢复",
      "done": "您可以在项目中找到 {name}。",
      "restoring": "正在恢复"
    },
    "members_added": "成员已添加。",
    "select_member": "选择同事",
    "feature_updated": "项目功能已更新。",
    "default_tab_updated": "默认标签已更新。",
    "name_max_length": "项目名称应少于 255 个字符",
    "no_member_yet": "还没有成员",
    "archived": "已归档",
    "joined": "已加入",
    "settings_link": "{name} 设置"
````

- [ ] **Step 2: 测试**

`e2e/stories/project/p1-create-project.spec.ts`（修改，2 处）：

````old e2e/stories/project/p1-create-project.spec.ts
  await expect(page.getByRole("link", { name: /Web App/ })).toBeVisible();
````
````new e2e/stories/project/p1-create-project.spec.ts
  // its card, by its heading (the card's link to its settings is named by the project too)
  await expect(page.getByRole("heading", { name: "Web App", exact: true })).toBeVisible();
````
````old e2e/stories/project/p1-create-project.spec.ts
  await expect(page.getByRole("link", { name: /Ops/ })).toBeVisible();
````
````new e2e/stories/project/p1-create-project.spec.ts
  await expect(page.getByRole("heading", { name: "Ops", exact: true })).toBeVisible();
````

`e2e/stories/project/p2-visibility.spec.ts`（修改，1 处）：

````old e2e/stories/project/p2-visibility.spec.ts
  expect(await enabledWithin(page.getByRole("dialog").getByRole("button", { name: "Joining..." }))).toBe(false);
````
````new e2e/stories/project/p2-visibility.spec.ts
  expect(await enabledWithin(page.getByRole("dialog").getByRole("button", { name: "Joining" }))).toBe(false);
````

`e2e/stories/project/p4-archive.spec.ts`（修改，1 处）：

````old e2e/stories/project/p4-archive.spec.ts
  await page.locator("#confirmDelete").fill("delete my project");
````
````new e2e/stories/project/p4-archive.spec.ts
  // the hint names the words to type, as the dialog compares them
  await expect(page.locator("#confirmDelete")).toHaveAttribute("placeholder", "Enter “delete my project”");
  await page.locator("#confirmDelete").fill("delete my project");
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，3 处）：

````old e2e/stories/project/p5-project-members.spec.ts
  const busy = theMember.page.getByRole("dialog").getByRole("button", { name: "Leaving..." });
````
````new e2e/stories/project/p5-project-members.spec.ts
  const busy = theMember.page.getByRole("dialog").getByRole("button", { name: "Leaving" });
````
````old e2e/stories/project/p5-project-members.spec.ts
  await theGuest.page.locator("#confirmLeave").fill("Leave Project");
````
````new e2e/stories/project/p5-project-members.spec.ts
  // the hint names the words to type, as the modal compares them
  await expect(theGuest.page.locator("#confirmLeave")).toHaveAttribute("placeholder", "Enter “Leave Project”");
  await theGuest.page.locator("#confirmLeave").fill("Leave Project");
````
````old e2e/stories/project/p5-project-members.spec.ts
  expect(await enabledWithin(theGuest.page.getByRole("button", { name: "Leaving..." }))).toBe(false);
````
````new e2e/stories/project/p5-project-members.spec.ts
  expect(await enabledWithin(theGuest.page.getByRole("button", { name: "Leaving" }))).toBe(false);
````

`e2e/stories/smoke/s7-chinese.spec.ts`（修改，2 处）：

````old e2e/stories/smoke/s7-chinese.spec.ts
import { createWorkspace, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
````
````new e2e/stories/smoke/s7-chinese.spec.ts
import { createProject, createWorkspace, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});

````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): an admin whose language is Chinese sees a project's general and members pages and their dialogs in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The general page: the settings' sidebar, the deletion's description, the archive's and the deletion's dialogs,
  // the project's name in them.
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(page.getByText("项目设置", { exact: true })).toBeVisible();
  await expect(page.getByText("删除项目时，项目中的全部数据和资源将被永久删除，且无法恢复。")).toBeVisible();
  await page.getByRole("button", { name: "归档", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText("归档 Web");
  await cancel(page);
  await page.getByRole("button", { name: "删除", exact: true }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText("删除项目");
  await expect(modalDialog(page).getByText(/^确定要删除项目 Web 吗？/)).toBeVisible();
  await cancel(page);

  // The members page: the member defaults.
  await page.goto(`/${slug}/settings/projects/${web.id}/members`);
  await expect(page.getByText("项目负责人", { exact: true })).toBeVisible();
  await expect(page.getByText("默认负责人", { exact: true })).toBeVisible();
  // The lead's select finds its "none" by the word it shows.
  await page.getByRole("heading", { name: "项目负责人" }).locator("xpath=following::button[1]").click();
  await page.keyboard.type("无");
  await expect(page.getByRole("option")).toHaveText(["无"]);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(2).fill(EMOJI_CHECK_WARNING) });
});

````

`web/apps/web/core/components/navigation/use-tab-preferences.test.ts`（修改，1 处）：

````old web/apps/web/core/components/navigation/use-tab-preferences.test.ts
      { type: "success", title: "Success!", message: "Default tab updated successfully." },
````
````new web/apps/web/core/components/navigation/use-tab-preferences.test.ts
      { type: "success", title: "toast.success", message: "project_dialogs.default_tab_updated" },
````

`web/apps/web/core/components/onboarding/switch-account-modal.test.tsx`（修改，1 处）：

````old web/apps/web/core/components/onboarding/switch-account-modal.test.tsx
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({
    t: (key: string, values: Record<string, string> = {}) => [key, ...Object.values(values)].join(" "),
  }),
}));
````
````new web/apps/web/core/components/onboarding/switch-account-modal.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
````

`web/apps/web/core/components/project/add-project-members-modal.test.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/add-project-members-modal.test.tsx
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Members added successfully." }]);
````
````new web/apps/web/core/components/project/add-project-members-modal.test.tsx
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "project_dialogs.members_added" }]);
````

`web/apps/web/core/components/project/archive-restore-modal.test.tsx`（修改，2 处）：

````old web/apps/web/core/components/project/archive-restore-modal.test.tsx
      said: { title: "Archive success", message: "Web has been archived successfully" },
````
````new web/apps/web/core/components/project/archive-restore-modal.test.tsx
      said: { title: "project_dialogs.archive.done_title", message: "project_dialogs.archive.done" },
````
````old web/apps/web/core/components/project/archive-restore-modal.test.tsx
      said: { title: "Restore success", message: "You can find Web in your projects." },
````
````new web/apps/web/core/components/project/archive-restore-modal.test.tsx
      said: { title: "project_dialogs.restore.done_title", message: "project_dialogs.restore.done" },
````

`web/apps/web/core/components/project/card.test.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/card.test.tsx
// guests among them); its other guests are told they joined it. The card renders on the server, at the workspace's
// projects, with stand-ins for its menu (fake-controls.ts, which keeps the items it was given) and for its dialogs;
// the caller's permissions are the store's own, over acme and its project web as nerve lists them to him
// (fake-permissions.ts).
````
````new web/apps/web/core/components/project/card.test.tsx
// guests among them); its other guests are told they joined it, and one who is no member of it is offered joining it.
// The card renders on the server, at the workspace's projects, with stand-ins for its menu and its buttons
// (fake-controls.ts, which keeps the items and props it was given), for its dialogs and for @nerve/i18n, whose text is
// its key, then the values it was given (fake-i18n-values.ts); the caller's permissions are the store's own, over acme
// and its project web as nerve lists them to him (fake-permissions.ts).
````
````old web/apps/web/core/components/project/card.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
````
````new web/apps/web/core/components/project/card.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
````
````old web/apps/web/core/components/project/card.test.tsx
  const inCard = {
    restore: />Restore</.test(card),
    delete: card.includes('aria-label="Delete"'),
    settings: card.includes(`href="/${acme.slug}/settings/projects/${web.id}"`),
  };
````
````new web/apps/web/core/components/project/card.test.tsx
  // the link to its settings (an icon alone), as the card renders it, if it does
  const settingsLink = new RegExp(`<a [^>]*href="/${acme.slug}/settings/projects/${web.id}"[^>]*>`).exec(card)?.[0];
  const inCard = {
    restore: />restore</.test(card),
    delete: card.includes('aria-label="delete"'),
    settings: settingsLink !== undefined,
    settingsName: settingsLink && /aria-label="([^"]*)"/.exec(settingsLink)?.[1],
    joined: />project_dialogs\.joined</.test(card),
    buttons: shown.buttons.map((button) => button.children),
  };
````
````old web/apps/web/core/components/project/card.test.tsx
  return { inCard, menu };
````
````new web/apps/web/core/components/project/card.test.tsx
  return { inCard, menu, project };
````
````old web/apps/web/core/components/project/card.test.tsx
      expect([inCard.settings, menu.includes("settings")]).toEqual([row.linked, row.linked]);
    }
  );
````
````new web/apps/web/core/components/project/card.test.tsx
      expect([inCard.settings, menu.includes("settings")]).toEqual([row.linked, row.linked]);
    }
  );

  it("names its settings link by the project, tells a guest he joined it and offers one who is no member joining, in the page's language", async () => {
    const admin = await offeredTo(its.admin.workspaceRole, its.admin.projectRole, false);
    const guest = await offeredTo(its.guest.workspaceRole, its.guest.projectRole, false);
    const outsider = await offeredTo(its.outsider.workspaceRole, its.outsider.projectRole, false);
    expect([admin.inCard.settingsName, guest.inCard.joined, outsider.inCard.buttons]).toEqual([
      `project_dialogs.settings_link ${admin.project.name}`,
      true,
      ["common.join"],
    ]);
  });
````

`web/apps/web/core/components/project/delete-project-modal.test.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/delete-project-modal.test.tsx
const deleted = { type: "success", title: "Success!", message: "Project deleted successfully." };
````
````new web/apps/web/core/components/project/delete-project-modal.test.tsx
const deleted = { type: "success", title: "toast.success", message: "project_dialogs.delete.deleted" };
````

`web/apps/web/core/components/project/project-dialog-texts.test.tsx`（新文件，89 行）：

````file web/apps/web/core/components/project/project-dialog-texts.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyShown, shown } from "@/lib/fake-controls";
import { projectOf } from "@/store/project/fake-projects";
import { ConfirmProjectMemberRemove } from "./confirm-project-member-remove";
import { JoinProjectModal } from "./join-project-modal";
import { LeaveProjectModal } from "./leave-project-modal";

// The texts of the project's dialogs that no story opens in Chinese (M3 closeout): each renders on the server, open,
// with the stand-in for @nerve/i18n, whose text is its key, and stand-ins for its controls (fake-controls.ts), which
// keep the props they were given: each title, sentence and button is a key of the page's language.

const web = projectOf("WEB", "w-acme", { name: "Web" });
vi.mock("react-router", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ workspaceSlug: "acme", projectId: web.id }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectById: () => web }) }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "u-admin" } }) }));
vi.mock("./use-join-project", () => ({ useJoinProject: () => vi.fn() }));
vi.mock("@/components/project/settings/use-project-membership-changes", () => ({
  useProjectMembershipChanges: () => ({ leave: vi.fn() }),
}));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  emptyShown();
});

describe("the project's dialogs", () => {
  it.each<{ dialog: string; open: () => ReactElement; sentences: string[]; buttons: string[] }>([
    {
      dialog: "the joining",
      open: () => <JoinProjectModal isOpen workspaceSlug="acme" project={web} handleClose={() => {}} />,
      sentences: ["project_dialogs.join.title", "project_dialogs.join.body"],
      buttons: ["cancel", "project_dialogs.join.submit"],
    },
    {
      dialog: "the leaving",
      open: () => <LeaveProjectModal isOpen project={web} onClose={() => {}} />,
      sentences: [
        "project_dialogs.leave.title",
        "project_dialogs.leave.body",
        "project_dialogs.type_project_name",
        "project_dialogs.type_confirmation",
      ],
      buttons: ["cancel", "project_dialogs.leave.title"],
    },
    {
      dialog: "a member's removal",
      open: () => (
        <ConfirmProjectMemberRemove
          isOpen
          data={{ id: "u-bob", display_name: "bob" }}
          onSubmit={() => Promise.resolve()}
          onClose={() => {}}
        />
      ),
      sentences: ["project_dialogs.remove_member.remove_title", "project_dialogs.remove_member.remove_body"],
      buttons: ["cancel", "remove"],
    },
    {
      dialog: "one's own leaving, from the members",
      open: () => (
        <ConfirmProjectMemberRemove
          isOpen
          data={{ id: "u-admin", display_name: "admin" }}
          onSubmit={() => Promise.resolve()}
          onClose={() => {}}
        />
      ),
      sentences: ["project_dialogs.remove_member.leave_title", "project_dialogs.remove_member.leave_body"],
      buttons: ["cancel", "leave"],
    },
  ])("$dialog says each of its texts in the page's language", ({ open, sentences, buttons }) => {
    const markup = renderToStaticMarkup(open());
    expect(sentences.filter((key) => !markup.includes(key))).toEqual([]);
    expect(shown.buttons.map((button) => button.children)).toEqual(buttons);
  });
});
````

`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
      { type: "success", title: "Success!", message: "Project feature updated successfully." },
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
      { type: "success", title: "toast.success", message: "project_dialogs.feature_updated" },
````

`web/apps/web/core/lib/fake-i18n-values.ts`（新文件，12 行）：

````file web/apps/web/core/lib/fake-i18n-values.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for @nerve/i18n, for the tests that read what a page puts into a text (a name, an address): a test file
// mocks that module with this one, vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values")). A text is its key,
// then the values it was given, each after a space. fake-i18n.ts gives the key alone.

export function useTranslation() {
  return { t: (key: string, values: Record<string, unknown> = {}) => [key, ...Object.values(values)].join(" ") };
}
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 301 条、`@nerve/ui` 14 条、`@nerve/propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 114 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/project/p1-create-project.spec.ts e2e/stories/project/p2-visibility.spec.ts e2e/stories/project/p4-archive.spec.ts e2e/stories/project/p5-project-members.spec.ts e2e/stories/smoke/s7-chinese.spec.ts web/apps/web/core/components/navigation/use-tab-preferences.test.ts web/apps/web/core/components/navigation/use-tab-preferences.ts web/apps/web/core/components/onboarding/switch-account-modal.test.tsx web/apps/web/core/components/project/add-project-members-modal.test.tsx web/apps/web/core/components/project/add-project-members-modal.tsx web/apps/web/core/components/project/archive-restore-modal.test.tsx web/apps/web/core/components/project/archive-restore-modal.tsx web/apps/web/core/components/project/card.test.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/confirm-project-member-remove.tsx web/apps/web/core/components/project/delete-project-modal.test.tsx web/apps/web/core/components/project/delete-project-modal.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/join-project-modal.tsx web/apps/web/core/components/project/leave-project-modal.tsx web/apps/web/core/components/project/member-select.tsx web/apps/web/core/components/project/project-dialog-texts.test.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/project/settings/control-section.tsx web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx web/apps/web/core/components/project/settings/use-feature-toggle.ts web/apps/web/core/lib/fake-i18n-values.ts web/packages/i18n/src/locales/en/project-settings.json web/packages/i18n/src/locales/en/project.json web/packages/i18n/src/locales/zh-CN/project-settings.json web/packages/i18n/src/locales/zh-CN/project.json
```
```bash
git commit -m "feat(M3/closeout): the project's dialogs and settings speak the user's language

Deleting, leaving, joining, archiving and restoring a project, removing
and adding members, the member defaults, the feature toggles and the tab
bar's toasts read their text through t(), a sentence with a name one key.
The busy labels are the shared Leaving and Joining; the typed
confirmation's hint shows the words the dialog compares. The project
card's Joined and Join read t() too, and its settings link is named by
the project. S7 reads a project's general and members pages in Chinese.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | 删除项目的标题是英文 | 故事 S7 | 端到端 |
| `T5.2` | 归档的弹窗标题是英文 | 故事 S7 | 端到端 |
| `T5.3` | 项目负责人的默认值的标题是英文 | 故事 S7 | 端到端 |
| `T5.4` | 访客的开关的名字是英文 | 故事 S7 | 端到端 |
| `T5.5` | 删除项目的说明是英文 | 故事 S7 | 端到端 |
| `T5.6` | 删除项目的提示是英文 | `delete-project-modal.test.tsx` | vitest |
| `T5.7` | 归档的提示是英文 | `archive-restore-modal.test.tsx` | vitest |
| `T5.8` | 添加成员的提示是英文 | `add-project-members-modal.test.tsx` | vitest |
| `T5.9` | 功能开关的提示是英文 | `use-feature-toggle.test.tsx` | vitest |
| `T5.10` | 新的默认标签页的提示是英文 | `use-tab-preferences.test.ts` | vitest |
| `T5.11` | 归档的卡片上"恢复"是英文 | `card.test.tsx` | vitest |
| `T5.12` | 加入项目的弹窗标题是英文 | `project-dialog-texts.test.tsx` | vitest |
| `T5.13` | 离开项目的那句话是英文 | `project-dialog-texts.test.tsx` | vitest |
| `T5.14` | 移出项目成员的标题是英文 | `project-dialog-texts.test.tsx` | vitest |
| `T5.15` | 访客的卡片上"已加入"是英文 | `card.test.tsx` | vitest |
| `T5.16` | 不是成员的人的卡片上"加入"是英文 | `card.test.tsx` | vitest |
| `T5.17` | 卡片上设置的链接没有名字 | `card.test.tsx` | vitest |
| `T5.18` | 归档的提示的标题（按完整的键读）在两种语言中都没有定义 | `tools/i18n-keys.mjs`（Task 9 起） | 静态 |
| `T5.19` | 成员下拉框的"无"又按英文的词搜索 | 故事 S7 | 端到端 |
| `T5.20` | 确认的提示又写着自己的词（Plane 的"leave project"） | 故事 P5 | 端到端 |
| `T5.21` | 删除项目的提示不是弹窗比对的那几个词 | 故事 P4 | 端到端 |
| `T5.22` | 自己从成员页离开的那句话又是英文的几段 | `project-dialog-texts.test.tsx` | vitest |
| `T5.23` | 卡片上设置的链接只叫"Settings"，每张卡片都一样 | `card.test.tsx` | vitest |

---

### Task 6: 状态、标签的设置页上的英文；角色名只有一个来源

**Files:**
- Modify: `e2e/fixtures/project-pages.ts`、`e2e/stories/project/p6-states-page.spec.ts`、`e2e/stories/project/p7-labels-page.spec.ts`、`e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/core/components/editor/embeds/mentions/user.tsx`、`web/apps/web/core/components/labels/delete-label-modal.tsx`、`web/apps/web/core/components/labels/project-setting-label-group.tsx`、`web/apps/web/core/components/labels/project-setting-label-item.tsx`、`web/apps/web/core/components/project-states/create-update/create-update.test.tsx`、`web/apps/web/core/components/project-states/create-update/create.tsx`、`web/apps/web/core/components/project-states/create-update/form.tsx`、`web/apps/web/core/components/project-states/create-update/update.tsx`、`web/apps/web/core/components/project-states/group-item.tsx`、`web/apps/web/core/components/project-states/options/delete.test.tsx`、`web/apps/web/core/components/project-states/options/delete.tsx`、`web/apps/web/core/components/project-states/options/mark-as-default.tsx`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`、`web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx`、`web/apps/web/core/lib/fake-controls.ts`、`web/packages/constants/src/state.ts`、`web/packages/constants/src/workspace.ts`、`web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/en/project.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/project.json`、`web/packages/utils/src/index.ts`
- Delete: `web/packages/utils/src/permission/index.ts`、`web/packages/utils/src/permission/role.ts`

**Interfaces:**（spec 2.8）
- 状态：建、改的提示和按钮（`common.create`、`creating`、`update`、`updating`），表单的占位和取消，删除的标题和说明（`Translated`）和提示，设为默认的三处；组的标题读组的名字（`STATE_GROUPS[g].i18n_label`，新字段，C9）。标签：菜单的三项，删除的标题（"Delete label"）和说明。两个删除确认框给 `AlertModalCore` 传经 `t()` 的按钮文字。
- 角色名经 `t(ROLE_DETAILS[role].i18n_title)`：工作区成员的一列、邀请的一行、工作区菜单的一项、编辑器的提及卡片；`@nerve/utils` 的 `ROLE`、`getUserRole`（`permission/`）删除。添加项目成员的角色总有值，"Select role"的分支、规则和规则的错误一行删除。
- `e2e/fixtures/project-pages.ts`：`labelRow` 从 P7 移来（S7 也读它）；`fake-controls.ts` 的 `Alert` 收标题、内容和按钮文字。

**Tests:** 状态的表单和删除的测试读键。P6 照组的名字，P7 照"Delete label"。S7：第一个加菜单里的角色、成员和邀请的角色；第二个加状态页（组名、设为默认）、标签页（菜单、删除的弹窗）和添加成员一行的错误；第三个（新）：邀请页的标题说角色（P9 的存活变异 `T4f.M3a`）。

- [ ] **Step 1: 状态、标签和角色名**

`web/apps/web/core/components/editor/embeds/mentions/user.tsx`（修改，3 处）：

````old web/apps/web/core/components/editor/embeds/mentions/user.tsx
import { ROLE } from "@nerve/constants";
````
````new web/apps/web/core/components/editor/embeds/mentions/user.tsx
import { ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/editor/embeds/mentions/user.tsx
  const { data: currentUser } = useUser();
````
````new web/apps/web/core/components/editor/embeds/mentions/user.tsx
  const { data: currentUser } = useUser();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/editor/embeds/mentions/user.tsx
                {roleDetails && <p className="text-11 text-secondary">{ROLE[roleDetails]}</p>}
````
````new web/apps/web/core/components/editor/embeds/mentions/user.tsx
                {roleDetails && <p className="text-11 text-secondary">{t(ROLE_DETAILS[roleDetails].i18n_title)}</p>}
````

`web/apps/web/core/components/labels/delete-label-modal.tsx`（修改，5 处）：

````old web/apps/web/core/components/labels/delete-label-modal.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
import type { Label } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
import { AlertModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
import { AlertModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
  const { remove } = useLabelChanges();
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
  const { remove } = useLabelChanges();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
      title="Delete Label"
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
      title={t("project_settings.labels.delete.title")}
      primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
      secondaryButtonText={t("cancel")}
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
        <>
          Are you sure you want to delete <span className="font-medium text-primary">{data?.name}</span>? This will
          remove the label from all the work item and from any views where the label is being filtered upon.
        </>
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
        <Translated
          i18nKey="project_settings.labels.delete.body"
          values={{ name: <span className="font-medium text-primary">{data?.name}</span> }}
        />
````

`web/apps/web/core/components/labels/project-setting-label-group.tsx`（修改，4 处）：

````old web/apps/web/core/components/labels/project-setting-label-group.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
import type { Label } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
  const [isEditLabelForm, setEditLabelForm] = useState(false);
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
  const [isEditLabelForm, setEditLabelForm] = useState(false);
  const { t } = useTranslation();
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
      text: "Edit label",
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
      text: t("project_settings.labels.edit"),
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
      text: "Delete label",
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
      text: t("project_settings.labels.delete.title"),
````

`web/apps/web/core/components/labels/project-setting-label-item.tsx`（修改，4 处）：

````old web/apps/web/core/components/labels/project-setting-label-item.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
import type { Label } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
  const { removeFromGroup } = useLabelChanges();
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
  const { removeFromGroup } = useLabelChanges();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
      text: "Remove from group",
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
      text: t("project_settings.labels.remove_from_group"),
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
      text: "Edit label",
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
      text: t("project_settings.labels.edit"),
````

`web/apps/web/core/components/project-states/create-update/create.tsx`（修改，4 处）：

````old web/apps/web/core/components/project-states/create-update/create.tsx
import { STATE_GROUPS } from "@nerve/constants";
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
import { STATE_GROUPS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
  const { create } = useStateChanges();
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
  const { create } = useStateChanges();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
        title: "Success!",
        message: "State created successfully.",
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
        title: t("toast.success"),
        message: t("project_settings.states.created"),
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
      buttonTitle="Create"
      busyButtonTitle="Creating"
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
      buttonTitle={t("common.create")}
      busyButtonTitle={t("common.creating")}
````

`web/apps/web/core/components/project-states/create-update/form.tsx`（修改，3 处）：

````old web/apps/web/core/components/project-states/create-update/form.tsx
                  placeholder="Name"
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
                  placeholder={t("name")}
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
                  placeholder="Describe this state for your members."
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
                  placeholder={t("project_settings.states.description_placeholder")}
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
            Cancel
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
            {t("cancel")}
````

`web/apps/web/core/components/project-states/create-update/update.tsx`（修改，4 处）：

````old web/apps/web/core/components/project-states/create-update/update.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
  const { update } = useStateChanges();
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
  const { update } = useStateChanges();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
        title: "Success!",
        message: "State updated successfully.",
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
        title: t("toast.success"),
        message: t("project_settings.states.updated"),
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
      buttonTitle="Update"
      busyButtonTitle="Updating"
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
      buttonTitle={t("common.update")}
      busyButtonTitle={t("common.updating")}
````

`web/apps/web/core/components/project-states/group-item.tsx`（修改，3 处）：

````old web/apps/web/core/components/project-states/group-item.tsx
import { EIconSize } from "@nerve/constants";
````
````new web/apps/web/core/components/project-states/group-item.tsx
import { EIconSize, STATE_GROUPS } from "@nerve/constants";
````
````old web/apps/web/core/components/project-states/group-item.tsx
          <span className="px-1 text-14 font-medium text-secondary capitalize">{groupKey}</span>
````
````new web/apps/web/core/components/project-states/group-item.tsx
          <span className="px-1 text-14 font-medium text-secondary">{t(STATE_GROUPS[groupKey].i18n_label)}</span>
````
````old web/apps/web/core/components/project-states/group-item.tsx
          <div>{t("project_settings.states.empty_state.title", { groupKey })}</div>
````
````new web/apps/web/core/components/project-states/group-item.tsx
          <div>
            {t("project_settings.states.empty_state.title", { groupKey: t(STATE_GROUPS[groupKey].i18n_label) })}
          </div>
````

`web/apps/web/core/components/project-states/options/delete.tsx`（修改，4 处）：

````old web/apps/web/core/components/project-states/options/delete.tsx
import { cn } from "@nerve/utils";
````
````new web/apps/web/core/components/project-states/options/delete.tsx
import { cn } from "@nerve/utils";
// components
import { Translated } from "@/components/common/translated";
````
````old web/apps/web/core/components/project-states/options/delete.tsx
        title="Delete State"
````
````new web/apps/web/core/components/project-states/options/delete.tsx
        title={t("project_settings.states.delete.title")}
        primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
        secondaryButtonText={t("cancel")}
````
````old web/apps/web/core/components/project-states/options/delete.tsx
          <>
            Are you sure you want to delete state- <span className="font-medium text-primary">{state?.name}</span>? All
            of the data related to the state will be permanently removed. This action cannot be undone.
          </>
````
````new web/apps/web/core/components/project-states/options/delete.tsx
          <Translated
            i18nKey="project_settings.states.delete.body"
            values={{ name: <span className="font-medium text-primary">{state.name}</span> }}
          />
````
````old web/apps/web/core/components/project-states/options/delete.tsx
            state.default ? "Cannot delete the default state." : totalStates === 1 ? `Cannot have an empty group.` : ``
````
````new web/apps/web/core/components/project-states/options/delete.tsx
            state.default
              ? t("project_settings.states.delete.default")
              : totalStates === 1
                ? t("project_settings.states.delete.only_one")
                : ""
````

`web/apps/web/core/components/project-states/options/mark-as-default.tsx`（修改，3 处）：

````old web/apps/web/core/components/project-states/options/mark-as-default.tsx
// nerve imports
````
````new web/apps/web/core/components/project-states/options/mark-as-default.tsx
// nerve imports
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project-states/options/mark-as-default.tsx
  const { markDefault } = useStateChanges();
````
````new web/apps/web/core/components/project-states/options/mark-as-default.tsx
  const { markDefault } = useStateChanges();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project-states/options/mark-as-default.tsx
      {isLoading ? "Marking as default" : isDefault ? `Default` : `Mark as default`}
````
````new web/apps/web/core/components/project-states/options/mark-as-default.tsx
      {t(
        isLoading
          ? "project_settings.states.default.marking"
          : isDefault
            ? "project_settings.states.default.is_default"
            : "project_settings.states.default.mark"
      )}
````

`web/apps/web/core/components/project/add-project-members-modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/add-project-members-modal.tsx
                    rules={{ required: "Please select a member" }}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                    rules={{ required: t("project_dialogs.member_required") }}
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                      control={control}
                      rules={{ required: "Select Role" }}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                      control={control}
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                              <span className="capitalize">
                                {roleField.value ? t(ROLE_DETAILS[roleField.value].i18n_title) : "Select role"}
                              </span>
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                              <span className="capitalize">{t(ROLE_DETAILS[roleField.value].i18n_title)}</span>
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                    />
                    {errors.members && errors.members[index]?.role && (
                      <span className="px-1 text-13 text-danger-primary">{errors.members[index]?.role?.message}</span>
                    )}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                    />
````

`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`（修改，4 处）：

````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
import { ROLE } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
import { ROLE_DETAILS } from "@nerve/constants";
````
````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
            <span className="px-2 py-0.5 text-caption-sm-medium">{ROLE[invitationDetails.role]}</span>
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
            <span className="px-2 py-0.5 text-caption-sm-medium">
              {t(ROLE_DETAILS[invitationDetails.role].i18n_title)}
            </span>
````
````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
                    {ROLE[invitationDetails.role]}
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
                    {t(ROLE_DETAILS[invitationDetails.role].i18n_title)}
````
````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
                  {ROLE[role]}
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
                  {t(ROLE_DETAILS[role].i18n_title)}
````

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（修改，4 处）：

````old web/apps/web/core/components/workspace/settings/member-columns.tsx
import { ROLE, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
import { EUserPermissions, EUserPermissionsLevel, ROLE_DETAILS } from "@nerve/constants";
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
        <div className="flex w-32">
          <span>{ROLE[rowData.role]}</span>
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
        <div className="flex w-32">
          <span>{t(ROLE_DETAILS[rowData.role].i18n_title)}</span>
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
              <span>{ROLE[rowData.role]}</span>
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
              <span>{t(ROLE_DETAILS[rowData.role].i18n_title)}</span>
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
              {ROLE[role]}
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
              {t(ROLE_DETAILS[role].i18n_title)}
````

`web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
import { EUserPermissions } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
import { EUserPermissions, ROLE_DETAILS } from "@nerve/constants";
````
````old web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
import { cn, getFileURL, getUserRole } from "@nerve/utils";
````
````new web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
import { cn, getFileURL } from "@nerve/utils";
````
````old web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
                <span>{getUserRole(workspace.role)?.toLowerCase() || "guest"}</span>
````
````new web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
                <span>{t(ROLE_DETAILS[workspace.role].i18n_title)}</span>
````

`web/packages/constants/src/state.ts`（修改，6 处）：

````old web/packages/constants/src/state.ts
    label: string;
````
````new web/packages/constants/src/state.ts
    label: string;
    /** The key of the group's name in the page's language. */
    i18n_label: string;
````
````old web/packages/constants/src/state.ts
    label: "Backlog",
````
````new web/packages/constants/src/state.ts
    label: "Backlog",
    i18n_label: "project_settings.states.groups.backlog",
````
````old web/packages/constants/src/state.ts
    label: "Unstarted",
````
````new web/packages/constants/src/state.ts
    label: "Unstarted",
    i18n_label: "project_settings.states.groups.unstarted",
````
````old web/packages/constants/src/state.ts
    label: "Started",
````
````new web/packages/constants/src/state.ts
    label: "Started",
    i18n_label: "project_settings.states.groups.started",
````
````old web/packages/constants/src/state.ts
    label: "Completed",
````
````new web/packages/constants/src/state.ts
    label: "Completed",
    i18n_label: "project_settings.states.groups.completed",
````
````old web/packages/constants/src/state.ts
    label: "Canceled",
````
````new web/packages/constants/src/state.ts
    label: "Canceled",
    i18n_label: "project_settings.states.groups.cancelled",
````

`web/packages/constants/src/workspace.ts`（修改，1 处）：

````old web/packages/constants/src/workspace.ts
  "500+": "workspace_creation.form.organization_size.sizes.500_plus",
};

export const ROLE = {
  [EUserWorkspaceRoles.GUEST]: "Guest",
  [EUserWorkspaceRoles.MEMBER]: "Member",
  [EUserWorkspaceRoles.ADMIN]: "Admin",
````
````new web/packages/constants/src/workspace.ts
  "500+": "workspace_creation.form.organization_size.sizes.500_plus",
````

`web/packages/i18n/src/locales/en/project-settings.json`（修改，2 处）：

````old web/packages/i18n/src/locales/en/project-settings.json
        "description": "Please create a new state"
````
````new web/packages/i18n/src/locales/en/project-settings.json
        "description": "Please create a new state"
      },
      "groups": {
        "backlog": "Backlog",
        "unstarted": "Unstarted",
        "started": "Started",
        "completed": "Completed",
        "cancelled": "Canceled"
      },
      "created": "State created successfully.",
      "updated": "State updated successfully.",
      "description_placeholder": "Describe this state for your members.",
      "delete": {
        "title": "Delete State",
        "body": "Are you sure you want to delete state {name}? All of the data related to the state will be permanently removed. This action cannot be undone.",
        "default": "Cannot delete the default state.",
        "only_one": "Cannot have an empty group."
      },
      "default": {
        "mark": "Mark as default",
        "marking": "Marking as default",
        "is_default": "Default"
````
````old web/packages/i18n/src/locales/en/project-settings.json
      "parent_not_allowed": "A label can go only under a label at the top of its project, and a label with labels under it stays at the top: labels have two levels."
````
````new web/packages/i18n/src/locales/en/project-settings.json
      "parent_not_allowed": "A label can go only under a label at the top of its project, and a label with labels under it stays at the top: labels have two levels.",
      "edit": "Edit label",
      "remove_from_group": "Remove from group",
      "delete": {
        "title": "Delete label",
        "body": "Are you sure you want to delete {name}? This will remove the label from all the work item and from any views where the label is being filtered upon."
      }
````

`web/packages/i18n/src/locales/en/project.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project.json
    "members_added": "Members added successfully.",
````
````new web/packages/i18n/src/locales/en/project.json
    "members_added": "Members added successfully.",
    "member_required": "Please select a member",
````

`web/packages/i18n/src/locales/zh-CN/project-settings.json`（修改，2 处）：

````old web/packages/i18n/src/locales/zh-CN/project-settings.json
        "description": "请创建一个新状态"
````
````new web/packages/i18n/src/locales/zh-CN/project-settings.json
        "description": "请创建一个新状态"
      },
      "groups": {
        "backlog": "待规划",
        "unstarted": "未开始",
        "started": "已开始",
        "completed": "已完成",
        "cancelled": "已取消"
      },
      "created": "状态已创建。",
      "updated": "状态已更新。",
      "description_placeholder": "向成员说明这个状态。",
      "delete": {
        "title": "删除状态",
        "body": "确定要删除状态 {name} 吗？这个状态的全部数据将被永久删除。此操作无法撤销。",
        "default": "不能删除默认状态。",
        "only_one": "组中至少要有一个状态。"
      },
      "default": {
        "mark": "设为默认",
        "marking": "正在设为默认",
        "is_default": "默认"
````
````old web/packages/i18n/src/locales/zh-CN/project-settings.json
      "parent_not_allowed": "标签只能放在本项目的顶层标签之下，下面有标签的标签留在顶层：标签只有两层。"
````
````new web/packages/i18n/src/locales/zh-CN/project-settings.json
      "parent_not_allowed": "标签只能放在本项目的顶层标签之下，下面有标签的标签留在顶层：标签只有两层。",
      "edit": "编辑标签",
      "remove_from_group": "移出组",
      "delete": {
        "title": "删除标签",
        "body": "确定要删除 {name} 吗？这个标签将从全部工作项和按它筛选的视图中移除。"
      }
````

`web/packages/i18n/src/locales/zh-CN/project.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project.json
    "members_added": "成员已添加。",
````
````new web/packages/i18n/src/locales/zh-CN/project.json
    "members_added": "成员已添加。",
    "member_required": "请选择成员",
````

`web/packages/utils/src/index.ts`（修改，1 处）：

````old web/packages/utils/src/index.ts
export * from "./notification";
export * from "./permission";
````
````new web/packages/utils/src/index.ts
export * from "./notification";
````

`web/packages/utils/src/permission/index.ts`（删除）：

````delete web/packages/utils/src/permission/index.ts
````

`web/packages/utils/src/permission/role.ts`（删除）：

````delete web/packages/utils/src/permission/role.ts
````

- [ ] **Step 2: 测试**

`e2e/fixtures/project-pages.ts`（修改，1 处）：

````old e2e/fixtures/project-pages.ts
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
````
````new e2e/fixtures/project-pages.ts
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}

/** The row of the label of name on page's label settings: its grip, its name, its menu; the innermost row with it. */
export function labelRow(page: Page, name: string): Locator {
  return page.locator("div.group", { has: page.getByRole("heading", { name, exact: true, level: 6 }) }).last();
````

`e2e/stories/project/p6-states-page.spec.ts`（修改，7 处）：

````old e2e/stories/project/p6-states-page.spec.ts
/** The heading of the group of key on page's state settings, which opens and closes its list. */
function groupHeading(page: Page, key: string): Locator {
  return page.getByRole("button", { name: key, exact: true });
````
````new e2e/stories/project/p6-states-page.spec.ts
/** The heading of the group named name on page's state settings, which opens and closes its list. */
function groupHeading(page: Page, name: string): Locator {
  return page.getByRole("button", { name, exact: true });
````
````old e2e/stories/project/p6-states-page.spec.ts
/** The button beside the heading of the group of key that adds a state to it. */
function addTo(page: Page, key: string): Locator {
  return groupHeading(page, key).locator("xpath=..").getByRole("button", { name: "Add" });
````
````new e2e/stories/project/p6-states-page.spec.ts
/** The button beside the heading of the group named name that adds a state to it. */
function addTo(page: Page, name: string): Locator {
  return groupHeading(page, name).locator("xpath=..").getByRole("button", { name: "Add" });
````
````old e2e/stories/project/p6-states-page.spec.ts
    ["backlog", "unstarted", "started", "completed", "cancelled"].map((key) =>
      expect(groupHeading(page, key), key).toHaveAttribute("aria-expanded", "true")
````
````new e2e/stories/project/p6-states-page.spec.ts
    ["Backlog", "Unstarted", "Started", "Completed", "Canceled"].map((name) =>
      expect(groupHeading(page, name), name).toHaveAttribute("aria-expanded", "true")
````
````old e2e/stories/project/p6-states-page.spec.ts
  const completed = groupHeading(page, "completed");
````
````new e2e/stories/project/p6-states-page.spec.ts
  const completed = groupHeading(page, "Completed");
````
````old e2e/stories/project/p6-states-page.spec.ts
  await addTo(page, "started").click();
````
````new e2e/stories/project/p6-states-page.spec.ts
  await addTo(page, "Started").click();
````
````old e2e/stories/project/p6-states-page.spec.ts
  await addTo(page, "unstarted").click();
````
````new e2e/stories/project/p6-states-page.spec.ts
  await addTo(page, "Unstarted").click();
````
````old e2e/stories/project/p6-states-page.spec.ts
    await expect(addTo(other.page, "started"), who).toBeEnabled({ enabled: controls });
````
````new e2e/stories/project/p6-states-page.spec.ts
    await expect(addTo(other.page, "Started"), who).toBeEnabled({ enabled: controls });
````

`e2e/stories/project/p7-labels-page.spec.ts`（修改，2 处）：

````old e2e/stories/project/p7-labels-page.spec.ts
import { dragOnto, tabCycle } from "../../fixtures/project-pages";
````
````new e2e/stories/project/p7-labels-page.spec.ts
import { dragOnto, labelRow, tabCycle } from "../../fixtures/project-pages";
````
````old e2e/stories/project/p7-labels-page.spec.ts
// as its member and a workspace admin who is its member see them. The API version is p7-labels.spec.ts.

/** The row of the label of name on page's label settings: its grip, its name, its menu; the innermost row with it. */
function labelRow(page: Page, name: string): Locator {
  return page.locator("div.group", { has: page.getByRole("heading", { name, exact: true, level: 6 }) }).last();
}
````
````new e2e/stories/project/p7-labels-page.spec.ts
// as its member and a workspace admin who is its member see them. The API version is p7-labels.spec.ts.
````

`e2e/stories/smoke/s7-chinese.spec.ts`（修改，14 处）：

````old e2e/stories/smoke/s7-chinese.spec.ts
import { createProject, createWorkspace, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
````
````new e2e/stories/smoke/s7-chinese.spec.ts
import {
  createLabel,
  createProject,
  createWorkspace,
  invitationTo,
  invite,
  inviteAndAccept,
  slugFor,
  type Api,
} from "../../fixtures/api";
````
````old e2e/stories/smoke/s7-chinese.spec.ts
import { modalDialog } from "../../fixtures/dialogs";
````
````new e2e/stories/smoke/s7-chinese.spec.ts
import { modalDialog } from "../../fixtures/dialogs";
import { labelRow } from "../../fixtures/project-pages";
````
````old e2e/stories/smoke/s7-chinese.spec.ts
import { memberRow } from "../../fixtures/workspace-pages";
````
````new e2e/stories/smoke/s7-chinese.spec.ts
import { invitationLinkOf, memberRow } from "../../fixtures/workspace-pages";
````
````old e2e/stories/smoke/s7-chinese.spec.ts
// language): an account whose language is zh-CN sees its pages, menus and dialogs in Chinese.
````
````new e2e/stories/smoke/s7-chinese.spec.ts
// language): an account whose language is zh-CN sees its pages, menus and dialogs in Chinese.

/** Makes the language of the account of tokens Chinese, the home's tour taken already. */
async function speakChinese(api: Api, tokens: AuthTokens): Promise<void> {
  const chinese = await api.PATCH("/api/v0/me/profile", {
    body: { language: "zh-CN", is_tour_completed: true },
    headers: bearer(tokens.access_token),
  });
  expect(chinese.response.status).toBe(200);
}
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  const chinese = await api.PATCH("/api/v0/me/profile", {
    body: { language: "zh-CN", is_tour_completed: true },
    headers: bearer(admin.access_token),
  });
  expect(chinese.response.status).toBe(200);
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await speakChinese(api, admin);
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  const { admin, slug, bobEmail } = await chineseAdmin(api, testInfo);
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  const { admin, slug, bobEmail } = await chineseAdmin(api, testInfo);
  const carolEmail = emailFor(testInfo, "carol");
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 5 }]);
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  await page.getByRole("button", { name: "打开工作空间切换器" }).click();
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  // The menu names his role in Acme in Chinese too.
  await page.getByRole("button", { name: "打开工作空间切换器" }).click();
  await expect(page.getByRole("menu").getByText("管理员", { exact: true })).toBeVisible();
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  // The members page: its title, and the removal's dialog, the member's name in it.
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  // The members page: its title, bob's role, carol's invitation's, and the removal's dialog, the member's name in it.
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page).toHaveTitle("Acme - 成员");
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page).toHaveTitle("Acme - 成员");
  await expect(memberRow(page, bobEmail).getByRole("button", { name: "成员", exact: true })).toBeVisible();
  const carols = page.locator("div.group", { has: page.getByRole("heading", { name: carolEmail, exact: true }) });
  await expect(carols.getByRole("button", { name: "访客", exact: true })).toBeVisible();
````
````old e2e/stories/smoke/s7-chinese.spec.ts
test("S7 (page): an admin whose language is Chinese sees a project's general and members pages and their dialogs in Chinese", async ({
````
````new e2e/stories/smoke/s7-chinese.spec.ts
test("S7 (page): an admin whose language is Chinese sees a project's general, states, labels and members pages and their dialogs in Chinese", async ({
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const bug = await createLabel(api, admin.access_token, web.id, { name: "Bug" });
  await createLabel(api, admin.access_token, web.id, { name: "UI", parent_id: bug.id });
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await cancel(page);

  // The members page: the member defaults.
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await cancel(page);

  // The states page: a group by its name, the states not the default offered to be it.
  await page.goto(`/${slug}/settings/projects/${web.id}/states`);
  await expect(page.getByRole("button", { name: "已开始", exact: true })).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("button", { name: "设为默认", exact: true })).toHaveCount(4);

  // The labels page: a group's menu, the menu of a label in it, and the deletion's dialog, the group's name in it.
  await page.goto(`/${slug}/settings/projects/${web.id}/labels`);
  await labelRow(page, "UI").getByRole("button", { name: "选项" }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["移出组", "编辑标签"]);
  await page.keyboard.press("Escape");
  await labelRow(page, "Bug").getByRole("button", { name: "选项" }).click();
  await expect(page.getByRole("menuitem")).toHaveText(["编辑标签", "删除标签"]);
  await page.getByRole("menuitem", { name: "删除标签" }).click();
  await expect(modalDialog(page).getByRole("heading")).toHaveText("删除标签");
  await expect(modalDialog(page).getByText(/^确定要删除 Bug 吗？/)).toBeVisible();
  await cancel(page);

  // The members page: the member defaults.
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();

  // The adding of members: a row with no member picked says so.
  await page.getByRole("button", { name: "添加成员", exact: true }).click();
  await modalDialog(page).getByRole("button", { name: "添加成员", exact: true }).click();
  await expect(modalDialog(page).getByText("请选择成员", { exact: true })).toBeVisible();
  await cancel(page);
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(2).fill(EMOJI_CHECK_WARNING) });
});

````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese reads her invitation to Acme, as a guest, in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const carolEmail = emailFor(testInfo, "carol");
  const carol = await registerOnboarded(api, carolEmail);
  await speakChinese(api, carol);
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 5 }]);
  const page = await signedInPage(carol);
  const watch = await watchPage(page);

  await page.goto(invitationLinkOf(await invitationTo(api, admin.access_token, slug, carolEmail)));
  await expect(page.getByText("您受邀以 访客 的身份加入 Acme。", { exact: true })).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

````

`web/apps/web/core/components/project-states/create-update/create-update.test.tsx`（修改，3 处）：

````old web/apps/web/core/components/project-states/create-update/create-update.test.tsx
    said: "State created successfully.",
````
````new web/apps/web/core/components/project-states/create-update/create-update.test.tsx
    said: "project_settings.states.created",
````
````old web/apps/web/core/components/project-states/create-update/create-update.test.tsx
    said: "State updated successfully.",
````
````new web/apps/web/core/components/project-states/create-update/create-update.test.tsx
    said: "project_settings.states.updated",
````
````old web/apps/web/core/components/project-states/create-update/create-update.test.tsx
      [{ type: "success", title: "Success!", message: row.said }],
````
````new web/apps/web/core/components/project-states/create-update/create-update.test.tsx
      [{ type: "success", title: "toast.success", message: row.said }],
````

`web/apps/web/core/components/project-states/options/delete.test.tsx`（修改，6 处）：

````old web/apps/web/core/components/project-states/options/delete.test.tsx
// nerve refuses (409), and their button is disabled; nerve's refusal all the same (another tab changed them meanwhile)
// is said in a toast. Its session is fake-tab.ts's.
````
````new web/apps/web/core/components/project-states/options/delete.test.tsx
// nerve refuses (409), and their button is disabled, its tooltip (its label rendered beside it) saying why; nerve's
// refusal all the same (another tab changed them meanwhile) is said in a toast. Its session is fake-tab.ts's; its
// texts are the keys of the page's language (fake-i18n.ts).
````
````old web/apps/web/core/components/project-states/options/delete.test.tsx
  Tooltip: ({ children }: { children: ReactNode }) => children,
````
````new web/apps/web/core/components/project-states/options/delete.test.tsx
  Tooltip: ({ label, children }: { label: ReactNode; children: ReactNode }) => (
    <>
      {label}
      {children}
    </>
  ),
````
````old web/apps/web/core/components/project-states/options/delete.test.tsx
/** Renders the deletion of state among totalStates of its group: whether its button is disabled, and its dialog. */
````
````new web/apps/web/core/components/project-states/options/delete.test.tsx
/** Renders the deletion of state among totalStates of its group: whether its button is disabled, its dialog, its text. */
````
````old web/apps/web/core/components/project-states/options/delete.test.tsx
  return { disabled: /<button[^>]* disabled=""/.test(markup), alert };
````
````new web/apps/web/core/components/project-states/options/delete.test.tsx
  return { disabled: /<button[^>]* disabled=""/.test(markup), alert, markup };
````
````old web/apps/web/core/components/project-states/options/delete.test.tsx
    { state: "the default", deleted: { ...review, default: true }, totalStates: 2 },
    { state: "its group's only state", deleted: review, totalStates: 1 },
  ])("offers no deletion of $state, and sends none", async ({ deleted, totalStates }) => {
    const { disabled, alert } = deletionOf(deleted, totalStates);
````
````new web/apps/web/core/components/project-states/options/delete.test.tsx
    {
      state: "the default",
      deleted: { ...review, default: true },
      totalStates: 2,
      why: "project_settings.states.delete.default",
    },
    {
      state: "its group's only state",
      deleted: review,
      totalStates: 1,
      why: "project_settings.states.delete.only_one",
    },
  ])("offers no deletion of $state, says why, and sends none", async ({ deleted, totalStates, why }) => {
    const { disabled, alert, markup } = deletionOf(deleted, totalStates);
````
````old web/apps/web/core/components/project-states/options/delete.test.tsx
    expect([disabled, store.deleteState.mock.calls]).toEqual([true, []]);
````
````new web/apps/web/core/components/project-states/options/delete.test.tsx
    expect([disabled, markup.includes(why), store.deleteState.mock.calls]).toEqual([true, true, []]);
  });

  it("asks in the page's language, its buttons too", () => {
    const { alert } = deletionOf();
    const { title, content, primaryButtonText, secondaryButtonText } = alert;
    expect([title, renderToStaticMarkup(content), primaryButtonText, secondaryButtonText]).toEqual([
      "project_settings.states.delete.title",
      "project_settings.states.delete.body",
      { default: "delete", loading: "deleting" },
      "cancel",
    ]);
````

`web/apps/web/core/lib/fake-controls.ts`（修改，1 处）：

````old web/apps/web/core/lib/fake-controls.ts
/** A confirmation's dialog (AlertModalCore): whether it shows, its button's press, and whether it is busy. */
type Alert = { isOpen: boolean; handleSubmit: () => unknown; isSubmitting: boolean };
````
````new web/apps/web/core/lib/fake-controls.ts
/** A confirmation's dialog (AlertModalCore): whether it shows, its button's press, whether it is busy, its words. */
type Alert = {
  isOpen: boolean;
  handleSubmit: () => unknown;
  isSubmitting: boolean;
  title: string;
  content: ReactNode;
  primaryButtonText?: { default: string; loading: string };
  secondaryButtonText?: string;
};
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 301 条、`@nerve/ui` 14 条、`@nerve/propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 115 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/project-pages.ts e2e/stories/project/p6-states-page.spec.ts e2e/stories/project/p7-labels-page.spec.ts e2e/stories/smoke/s7-chinese.spec.ts web/apps/web/core/components/editor/embeds/mentions/user.tsx web/apps/web/core/components/labels/delete-label-modal.tsx web/apps/web/core/components/labels/project-setting-label-group.tsx web/apps/web/core/components/labels/project-setting-label-item.tsx web/apps/web/core/components/project-states/create-update/create-update.test.tsx web/apps/web/core/components/project-states/create-update/create.tsx web/apps/web/core/components/project-states/create-update/form.tsx web/apps/web/core/components/project-states/create-update/update.tsx web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/options/delete.test.tsx web/apps/web/core/components/project-states/options/delete.tsx web/apps/web/core/components/project-states/options/mark-as-default.tsx web/apps/web/core/components/project/add-project-members-modal.tsx web/apps/web/core/components/workspace/settings/invitations-list-item.tsx web/apps/web/core/components/workspace/settings/member-columns.tsx web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx web/apps/web/core/lib/fake-controls.ts web/packages/constants/src/state.ts web/packages/constants/src/workspace.ts web/packages/i18n/src/locales/en/project-settings.json web/packages/i18n/src/locales/en/project.json web/packages/i18n/src/locales/zh-CN/project-settings.json web/packages/i18n/src/locales/zh-CN/project.json web/packages/utils/src/index.ts web/packages/utils/src/permission/index.ts web/packages/utils/src/permission/role.ts
```
```bash
git commit -m "feat(M3/closeout): states and labels speak the user's language; a role's name has one source

The states' and labels' forms, menus and deletion dialogs read their text
through t(), and a state group's heading reads the group's name, not its
key. Every role's name comes from ROLE_DETAILS' i18n_title: ROLE and
getUserRole go. S7 reads the states, the labels and an invitation's page
in Chinese.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | 状态组的标题又是它们的键 | 故事 P6、故事 S7 | 端到端 |
| `T6.2` | 建状态的提示是英文 | `create-update.test.tsx` | vitest |
| `T6.3` | 改状态的提示是英文 | `create-update.test.tsx` | vitest |
| `T6.4` | 删除状态的询问是英文 | `delete.test.tsx` | vitest |
| `T6.5` | 删除状态的"取消"是英文 | `delete.test.tsx` | vitest |
| `T6.6` | 拒绝删除默认状态的说明是英文 | `delete.test.tsx` | vitest |
| `T6.7` | "默认"是英文 | 故事 S7 | 端到端 |
| `T6.8` | 编辑标签组的菜单项是英文 | 故事 S7 | 端到端 |
| `T6.9` | 把标签移出组的菜单项是英文 | 故事 S7 | 端到端 |
| `T6.10` | 删除标签的询问是英文 | 故事 S7 | 端到端 |
| `T6.11` | 删除标签的"取消"是英文 | 故事 S7 | 端到端 |
| `T6.12` | 工作区菜单里的角色是英文 | 故事 S7 | 端到端 |
| `T6.13` | 成员的角色是英文 | 故事 S7 | 端到端 |
| `T6.14` | 邀请的角色是英文 | 故事 S7 | 端到端 |
| `T6.15` | 没有成员的一行的说明是英文 | 故事 S7 | 端到端 |
| `T6.16` | 邀请页的标题用英文说角色（P9 的 `T4f.M3a`） | 故事 S7 | 端到端 |

---

### Task 7: `IconButton` 的名字在根上必传；侧边栏的名字；图标不并进名字

**Files:**
- Modify: `e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx`、`web/apps/web/core/components/comments/card/display.tsx`、`web/apps/web/core/components/comments/comment-reaction.tsx`、`web/apps/web/core/components/comments/quick-actions.tsx`、`web/apps/web/core/components/core/description-versions/modal.tsx`、`web/apps/web/core/components/cycles/cycles-view-header.tsx`、`web/apps/web/core/components/cycles/quick-actions.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx`、`web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx`、`web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx`、`web/apps/web/core/components/issues/peek-overview/header.tsx`、`web/apps/web/core/components/modules/module-view-header.tsx`、`web/apps/web/core/components/modules/quick-actions.tsx`、`web/apps/web/core/components/profile/sidebar.tsx`、`web/apps/web/core/components/project/search-projects.tsx`、`web/apps/web/core/components/rich-filters/filters-toggle.tsx`、`web/apps/web/core/components/settings/mobile/nav.tsx`、`web/apps/web/core/components/settings/profile/modal.tsx`、`web/apps/web/core/components/settings/workspace/sidebar/header.tsx`、`web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx`、`web/apps/web/core/components/views/quick-actions.tsx`、`web/apps/web/core/components/views/view-list-header.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/en/navigation.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/navigation.json`、`web/packages/propel/package.json`、`web/packages/propel/src/emoji-icon-picker/logo.tsx`、`web/packages/propel/src/emoji-reaction/emoji-reaction.tsx`、`web/packages/propel/src/icon-button/helper.tsx`

**Interfaces:**（spec 2.9）
- `@nerve/propel` 的 `IconButtonProps`：`"aria-label": string` 必传（从按钮的属性中 `Omit`）。应用中的 28 处都传经 `t()` 的名字（M2 1、M3 6、M4 8、M6 6、M7 6；表情回应的按钮在内）。
- `EmojiReactionButton` 收 `aria-label`（提示显示它），`EmojiReactionGroup` 收 `addReactionLabel`，回应按表情作 `key`（原来按位置，`no-array-index-key`；`@nerve/propel` 的上限 11 → 10）；`Logo` 的图标连字和表情 `aria-hidden`。
- 文案：`aria_labels.app_sidebar.*`（展开、收起和工作区菜单的按钮，原来没有定义）、`aria_labels.{settings_menu, toggle_sort_order, toggle_details, add_reaction}`、`sidebar.{more_projects, fewer_projects}`。

**Tests:** `tsc` 看住名字必传。S7：第一个加侧边栏的展开按钮、工作区菜单的名字、项目页的搜索、设置的返回；第二个加表情图标的项目的链接名和"打开扩展侧边栏"的"更多"。

- [ ] **Step 1: 名字**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx
            icon={RightSidePaneOutline}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx
            icon={RightSidePaneOutline}
            aria-label={t("aria_labels.toggle_details")}
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx
} from "@nerve/constants";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx
} from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx
  // hooks
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx
  // hooks
  const { t } = useTranslation();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx
          icon={RightSidePaneOutline}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx
          icon={RightSidePaneOutline}
          aria-label={t("aria_labels.toggle_details")}
````

`web/apps/web/core/components/comments/card/display.tsx`（修改，3 处）：

````old web/apps/web/core/components/comments/card/display.tsx
import { useHashScroll } from "@nerve/hooks";
````
````new web/apps/web/core/components/comments/card/display.tsx
import { useHashScroll } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/comments/card/display.tsx
  } = props;
````
````new web/apps/web/core/components/comments/card/display.tsx
  } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/comments/card/display.tsx
              label={<EmojiReactionButton onAddReaction={() => setIsPickerOpen(true)} />}
````
````new web/apps/web/core/components/comments/card/display.tsx
              label={
                <EmojiReactionButton
                  onAddReaction={() => setIsPickerOpen(true)}
                  aria-label={t("aria_labels.add_reaction")}
                />
              }
````

`web/apps/web/core/components/comments/comment-reaction.tsx`（修改，3 处）：

````old web/apps/web/core/components/comments/comment-reaction.tsx
// nerve imports
````
````new web/apps/web/core/components/comments/comment-reaction.tsx
// nerve imports
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/comments/comment-reaction.tsx
  const { comment, activityOperations, disabled = false } = props;
````
````new web/apps/web/core/components/comments/comment-reaction.tsx
  const { comment, activityOperations, disabled = false } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/comments/comment-reaction.tsx
            onAddReaction={() => setIsPickerOpen(true)}
````
````new web/apps/web/core/components/comments/comment-reaction.tsx
            onAddReaction={() => setIsPickerOpen(true)}
            addReactionLabel={t("aria_labels.add_reaction")}
````

`web/apps/web/core/components/comments/quick-actions.tsx`（修改，1 处）：

````old web/apps/web/core/components/comments/quick-actions.tsx
    <CustomMenu customButton={<IconButton icon={MoreHorizontalOutline} variant="ghost" size="sm" />} closeOnSelect>
````
````new web/apps/web/core/components/comments/quick-actions.tsx
    <CustomMenu
      customButton={
        <IconButton icon={MoreHorizontalOutline} variant="ghost" size="sm" aria-label={t("common.options")} />
      }
      closeOnSelect
    >
````

`web/apps/web/core/components/core/description-versions/modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/core/description-versions/modal.tsx
            <IconButton type="button" variant="ghost" size="base" onClick={handleCopyMarkdown} icon={CopyOutline} />
````
````new web/apps/web/core/components/core/description-versions/modal.tsx
            <IconButton
              type="button"
              variant="ghost"
              size="base"
              onClick={handleCopyMarkdown}
              icon={CopyOutline}
              aria-label={t("common.actions.copy_markdown")}
            />
````

`web/apps/web/core/components/cycles/cycles-view-header.tsx`（修改，1 处）：

````old web/apps/web/core/components/cycles/cycles-view-header.tsx
          icon={SearchOutline}
````
````new web/apps/web/core/components/cycles/cycles-view-header.tsx
          icon={SearchOutline}
          aria-label={t("common.search.label")}
````

`web/apps/web/core/components/cycles/quick-actions.tsx`（修改，1 处）：

````old web/apps/web/core/components/cycles/quick-actions.tsx
        customButton={<IconButton variant="tertiary" size="lg" icon={MoreHorizontalOutline} />}
````
````new web/apps/web/core/components/cycles/quick-actions.tsx
        customButton={
          <IconButton variant="tertiary" size="lg" icon={MoreHorizontalOutline} aria-label={t("common.options")} />
        }
````

`web/apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx
          <IconButton variant="tertiary" icon={FilterOutline} />
````
````new web/apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx
          <IconButton variant="tertiary" icon={FilterOutline} aria-label={t("common.filters")} />
````

`web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx`（修改，3 处）：

````old web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx
import { E_SORT_ORDER } from "@nerve/constants";
````
````new web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx
import { E_SORT_ORDER } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx
export const ActivitySortRoot = memo(function ActivitySortRoot(props: TActivitySortRoot) {
````
````new web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx
export const ActivitySortRoot = memo(function ActivitySortRoot(props: TActivitySortRoot) {
  const { t } = useTranslation();
````
````old web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx
  return <IconButton variant="tertiary" icon={SortIcon} onClick={props.toggleSort} />;
````
````new web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx
  return (
    <IconButton
      variant="tertiary"
      icon={SortIcon}
      aria-label={t("aria_labels.toggle_sort_order")}
      onClick={props.toggleSort}
    />
  );
````

`web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx
              <IconButton variant="secondary" size="lg" onClick={handleCopyText} icon={LinkOutline} />
````
````new web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx
              <IconButton
                variant="secondary"
                size="lg"
                onClick={handleCopyText}
                icon={LinkOutline}
                aria-label={t("common.actions.copy_link")}
              />
````

`web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx`（修改，3 处）：

````old web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
  const { workspaceSlug, projectId, issueId, currentUser, disabled = false, className = "" } = props;
````
````new web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
  const { workspaceSlug, projectId, issueId, currentUser, disabled = false, className = "" } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
            onAddReaction={() => setIsPickerOpen(true)}
````
````new web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
            onAddReaction={() => setIsPickerOpen(true)}
            addReactionLabel={t("aria_labels.add_reaction")}
````

`web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx`（修改，3 处）：

````old web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx
import { ARCHIVABLE_STATE_GROUPS, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx
import { ARCHIVABLE_STATE_GROUPS, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx
  props: TWorkItemDetailQuickActionProps
) {
````
````new web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx
  props: TWorkItemDetailQuickActionProps
) {
  const { t } = useTranslation();
````
````old web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx
        customButton={<IconButton size="lg" variant="secondary" icon={MoreHorizontalOutline} />}
````
````new web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx
        customButton={
          <IconButton size="lg" variant="secondary" icon={MoreHorizontalOutline} aria-label={t("common.options")} />
        }
````

`web/apps/web/core/components/issues/peek-overview/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/peek-overview/header.tsx
            <IconButton variant="secondary" size="lg" onClick={handleCopyText} icon={LinkOutline} />
````
````new web/apps/web/core/components/issues/peek-overview/header.tsx
            <IconButton
              variant="secondary"
              size="lg"
              onClick={handleCopyText}
              icon={LinkOutline}
              aria-label={t("common.actions.copy_link")}
            />
````

`web/apps/web/core/components/modules/module-view-header.tsx`（修改，1 处）：

````old web/apps/web/core/components/modules/module-view-header.tsx
            icon={SearchOutline}
````
````new web/apps/web/core/components/modules/module-view-header.tsx
            icon={SearchOutline}
            aria-label={t("common.search.label")}
````

`web/apps/web/core/components/modules/quick-actions.tsx`（修改，3 处）：

````old web/apps/web/core/components/modules/quick-actions.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/modules/quick-actions.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/modules/quick-actions.tsx
  const { parentRef, moduleId, projectId, workspaceSlug, customClassName } = props;
````
````new web/apps/web/core/components/modules/quick-actions.tsx
  const { parentRef, moduleId, projectId, workspaceSlug, customClassName } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/modules/quick-actions.tsx
        customButton={<IconButton variant="tertiary" size="lg" icon={MoreHorizontalOutline} />}
````
````new web/apps/web/core/components/modules/quick-actions.tsx
        customButton={
          <IconButton variant="tertiary" size="lg" icon={MoreHorizontalOutline} aria-label={t("common.options")} />
        }
````

`web/apps/web/core/components/profile/sidebar.tsx`（修改，1 处）：

````old web/apps/web/core/components/profile/sidebar.tsx
                icon={EditOutline}
````
````new web/apps/web/core/components/profile/sidebar.tsx
                icon={EditOutline}
                aria-label={t("common.actions.edit")}
````

`web/apps/web/core/components/project/search-projects.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/search-projects.tsx
          className="-mr-1"
````
````new web/apps/web/core/components/project/search-projects.tsx
          className="-mr-1"
          aria-label={t("common.search.label")}
````

`web/apps/web/core/components/rich-filters/filters-toggle.tsx`（修改，3 处）：

````old web/apps/web/core/components/rich-filters/filters-toggle.tsx
// nerve imports
````
````new web/apps/web/core/components/rich-filters/filters-toggle.tsx
// nerve imports
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/rich-filters/filters-toggle.tsx
  const { filter } = props;
````
````new web/apps/web/core/components/rich-filters/filters-toggle.tsx
  const { filter } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/rich-filters/filters-toggle.tsx
      icon={showFilterRowChangesPill ? SelectedFilterOutline : FilterOutline}
````
````new web/apps/web/core/components/rich-filters/filters-toggle.tsx
      icon={showFilterRowChangesPill ? SelectedFilterOutline : FilterOutline}
      aria-label={t("common.filters")}
````

`web/apps/web/core/components/settings/mobile/nav.tsx`（修改，1 处）：

````old web/apps/web/core/components/settings/mobile/nav.tsx
          icon={Menu}
````
````new web/apps/web/core/components/settings/mobile/nav.tsx
          icon={Menu}
          aria-label={t("aria_labels.settings_menu")}
````

`web/apps/web/core/components/settings/profile/modal.tsx`（修改，3 处）：

````old web/apps/web/core/components/settings/profile/modal.tsx
// nerve imports
````
````new web/apps/web/core/components/settings/profile/modal.tsx
// nerve imports
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/settings/profile/modal.tsx
  const { profileSettingsModal, toggleProfileSettingsModal } = useCommandPalette();
````
````new web/apps/web/core/components/settings/profile/modal.tsx
  const { profileSettingsModal, toggleProfileSettingsModal } = useCommandPalette();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/settings/profile/modal.tsx
          <IconButton size="base" variant="tertiary" icon={CloseOutline} onClick={handleClose} />
````
````new web/apps/web/core/components/settings/profile/modal.tsx
          <IconButton
            size="base"
            variant="tertiary"
            icon={CloseOutline}
            aria-label={t("close")}
            onClick={handleClose}
          />
````

`web/apps/web/core/components/settings/workspace/sidebar/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
          icon={ArrowNarrowLeftOutline}
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
          icon={ArrowNarrowLeftOutline}
          aria-label={t("common.go_back")}
````

`web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx`（修改，3 处）：

````old web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx
import { LeftSidePaneOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx
import { LeftSidePaneOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx
  const { toggleSidebar, sidebarPeek, toggleSidebarPeek } = useAppTheme();
````
````new web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx
  const { toggleSidebar, sidebarPeek, toggleSidebarPeek } = useAppTheme();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx
      icon={LeftSidePaneOutline}
````
````new web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx
      icon={LeftSidePaneOutline}
      aria-label={t("aria_labels.app_sidebar.toggle")}
````

`web/apps/web/core/components/views/quick-actions.tsx`（修改，3 处）：

````old web/apps/web/core/components/views/quick-actions.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/views/quick-actions.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/views/quick-actions.tsx
  const { parentRef, projectId, view, workspaceSlug, customClassName } = props;
````
````new web/apps/web/core/components/views/quick-actions.tsx
  const { parentRef, projectId, view, workspaceSlug, customClassName } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/views/quick-actions.tsx
        customButton={<IconButton variant="tertiary" size="lg" icon={MoreHorizontalOutline} />}
````
````new web/apps/web/core/components/views/quick-actions.tsx
        customButton={
          <IconButton variant="tertiary" size="lg" icon={MoreHorizontalOutline} aria-label={t("common.options")} />
        }
````

`web/apps/web/core/components/views/view-list-header.tsx`（修改，3 处）：

````old web/apps/web/core/components/views/view-list-header.tsx
import { useOutsideClickDetector } from "@nerve/hooks";
````
````new web/apps/web/core/components/views/view-list-header.tsx
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/views/view-list-header.tsx
export const ViewListHeader = observer(function ViewListHeader() {
````
````new web/apps/web/core/components/views/view-list-header.tsx
export const ViewListHeader = observer(function ViewListHeader() {
  const { t } = useTranslation();
````
````old web/apps/web/core/components/views/view-list-header.tsx
            icon={SearchOutline}
````
````new web/apps/web/core/components/views/view-list-header.tsx
            icon={SearchOutline}
            aria-label={t("common.search.label")}
````

`web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx
          <IconButton size="base" variant="ghost" icon={FilterOutline} />
````
````new web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx
          <IconButton size="base" variant="ghost" icon={FilterOutline} aria-label={t("notification.options.filters")} />
````

`web/apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx
      button={<IconButton size="base" variant="ghost" icon={MoreVerticalOutline} />}
````
````new web/apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx
      button={<IconButton size="base" variant="ghost" icon={MoreVerticalOutline} aria-label={t("common.options")} />}
````

`web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx
          icon={loader === ENotificationLoader.MARK_ALL_AS_READY ? Spinner : CheckDoneOutline}
````
````new web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx
          icon={loader === ENotificationLoader.MARK_ALL_AS_READY ? Spinner : CheckDoneOutline}
          aria-label={t("notification.options.mark_all_as_read")}
````
````old web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx
          icon={RefreshOutline}
````
````new web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx
          icon={RefreshOutline}
          aria-label={t("notification.options.refresh")}
````

`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                      icon={MoreHorizontalOutline}
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                      icon={MoreHorizontalOutline}
                      aria-label={t("aria_labels.projects_sidebar.toggle_quick_actions_menu")}
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                          <span>{isExtendedProjectSidebarOpened ? "Hide" : "More"}</span>
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                          <span>
                            {t(isExtendedProjectSidebarOpened ? "sidebar.fewer_projects" : "sidebar.more_projects")}
                          </span>
````

`web/packages/i18n/src/locales/en/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/accessibility.json
  "aria_labels": {
````
````new web/packages/i18n/src/locales/en/accessibility.json
  "aria_labels": {
    "settings_menu": "Settings menu",
    "toggle_sort_order": "Toggle sort order",
    "toggle_details": "Toggle details",
    "add_reaction": "Add reaction",
    "app_sidebar": {
      "toggle": "Toggle sidebar",
      "close_workspace_menu": "Close workspace menu",
      "open_workspace_menu": "Open workspace menu",
      "close_extended_sidebar": "Close extended sidebar",
      "open_extended_sidebar": "Open extended sidebar"
    },
````

`web/packages/i18n/src/locales/en/navigation.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/navigation.json
  "sidebar": {
````
````new web/packages/i18n/src/locales/en/navigation.json
  "sidebar": {
    "more_projects": "More",
    "fewer_projects": "Hide",
````

`web/packages/i18n/src/locales/zh-CN/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/accessibility.json
  "aria_labels": {
````
````new web/packages/i18n/src/locales/zh-CN/accessibility.json
  "aria_labels": {
    "settings_menu": "设置菜单",
    "toggle_sort_order": "切换排序方向",
    "toggle_details": "显示或隐藏详情",
    "add_reaction": "添加表情回应",
    "app_sidebar": {
      "toggle": "切换侧边栏",
      "close_workspace_menu": "收起工作区菜单",
      "open_workspace_menu": "展开工作区菜单",
      "close_extended_sidebar": "关闭扩展侧边栏",
      "open_extended_sidebar": "打开扩展侧边栏"
    },
````

`web/packages/i18n/src/locales/zh-CN/navigation.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/navigation.json
  "sidebar": {
````
````new web/packages/i18n/src/locales/zh-CN/navigation.json
  "sidebar": {
    "more_projects": "更多",
    "fewer_projects": "收起",
````

`web/packages/propel/package.json`（修改，1 处）：

````old web/packages/propel/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 11",
````
````new web/packages/propel/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 10",
````

`web/packages/propel/src/emoji-icon-picker/logo.tsx`（修改，2 处）：

````old web/packages/propel/src/emoji-icon-picker/logo.tsx
    return (
      <span
        className="flex items-center justify-center"
````
````new web/packages/propel/src/emoji-icon-picker/logo.tsx
    return (
      <span
        aria-hidden="true"
        className="flex items-center justify-center"
````
````old web/packages/propel/src/emoji-icon-picker/logo.tsx
    return (
      <span
        className="material-symbols-rounded"
````
````new web/packages/propel/src/emoji-icon-picker/logo.tsx
    return (
      <span
        aria-hidden="true"
        className="material-symbols-rounded"
````

`web/packages/propel/src/emoji-reaction/emoji-reaction.tsx`（修改，7 处）：

````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
  showAddButton?: boolean;
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
  showAddButton?: boolean;
  /** The name of the button that adds a reaction, in the page's language. */
  addReactionLabel: string;
````
````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
interface EmojiReactionButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  onAddReaction?: () => void;
  className?: string;
}
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
interface EmojiReactionButtonProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "aria-label"> {
  onAddReaction?: () => void;
  className?: string;
  /** The button's name, in the page's language, which its tooltip shows too. */
  "aria-label": string;
}
````
````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
    <Tooltip tooltipContent="Add reaction">
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
    <Tooltip tooltipContent={props["aria-label"]}>
````
````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
    showAddButton = true,
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
    showAddButton = true,
    addReactionLabel,
````
````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
      {reactions.map((reaction, index) => (
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
      {reactions.map((reaction) => (
````
````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
          key={`${reaction.emoji}-${index}`}
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
          key={reaction.emoji}
````
````old web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
      {showAddButton && <EmojiReactionButton onAddReaction={onAddReaction} />}
````
````new web/packages/propel/src/emoji-reaction/emoji-reaction.tsx
      {showAddButton && <EmojiReactionButton onAddReaction={onAddReaction} aria-label={addReactionLabel} />}
````

`web/packages/propel/src/icon-button/helper.tsx`（修改，2 处）：

````old web/packages/propel/src/icon-button/helper.tsx
type IconButtonPropsWithChildren = React.ButtonHTMLAttributes<HTMLButtonElement> &
````
````new web/packages/propel/src/icon-button/helper.tsx
type IconButtonPropsWithChildren = Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "aria-label"> &
````
````old web/packages/propel/src/icon-button/helper.tsx
    iconClassName?: string;
````
````new web/packages/propel/src/icon-button/helper.tsx
    iconClassName?: string;
    /** The button's name: it shows an icon alone, no text to be named by. */
    "aria-label": string;
````

- [ ] **Step 2: S7**

`e2e/stories/smoke/s7-chinese.spec.ts`（修改，9 处）：

````old e2e/stories/smoke/s7-chinese.spec.ts

import {
  createLabel,
````
````new e2e/stories/smoke/s7-chinese.spec.ts

import {
  changeWorkspacePreferences,
  createLabel,
````
````old e2e/stories/smoke/s7-chinese.spec.ts
test("S7 (page): an admin whose language is Chinese sees the workspace's home, menu, general and members pages and their dialogs in Chinese", async ({
````
````new e2e/stories/smoke/s7-chinese.spec.ts
test("S7 (page): an admin whose language is Chinese sees the workspace's home, menu, projects, general and members pages and their dialogs in Chinese", async ({
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  // The sidebar, by its name; the project navigation's dialog, each mode's description.
  await page.getByRole("complementary", { name: "主侧边栏" }).getByRole("button", { name: "项目导航" }).click();
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  // The sidebar, by its name: its buttons by theirs; the project navigation's dialog, each mode's description.
  const sidebar = page.getByRole("complementary", { name: "主侧边栏" });
  await expect(sidebar.getByRole("button", { name: "切换侧边栏" })).toBeVisible();
  await expect(sidebar.getByRole("button", { name: /^(收起|展开)工作区菜单$/ })).toHaveCount(2);
  await sidebar.getByRole("button", { name: "项目导航" }).click();
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  // The general page: the settings' sidebar, the size, and the deletion's dialog, the workspace's name in it.
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  // The projects page: its search, by its name.
  await page.goto(`/${slug}/projects`);
  await expect(page.getByRole("button", { name: "搜索", exact: true })).toBeVisible();

  // The general page: the settings' sidebar and its way back, the size, and the deletion's dialog, the workspace's
  // name in it.
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page.getByText("工作区设置", { exact: true })).toBeVisible();
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page.getByText("工作区设置", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "返回", exact: true })).toBeVisible();
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await createProject(api, admin.access_token, slug, {
    name: "Ops",
    identifier: "OPS",
    logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  });
  await changeWorkspacePreferences(api, admin.access_token, slug, {
    navigation_control_preference: "TABBED",
    navigation_project_limit: 1,
  });
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await createLabel(api, admin.access_token, web.id, { name: "UI", parent_id: bug.id });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await createLabel(api, admin.access_token, web.id, { name: "UI", parent_id: bug.id });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The projects page: the sidebar shows one project, as its limit says, Ops, made last, which it puts first, and
  // offers the rest; its link to Ops, its tabs beside it, is named by Ops's name alone, its logo no part of it.
  await page.goto(`/${slug}/projects`);
  const sidebar = page.getByRole("complementary", { name: "主侧边栏" });
  await expect(sidebar.getByRole("link", { name: "Ops", exact: true })).toBeVisible();
  await expect(sidebar.getByRole("button", { name: "打开扩展侧边栏" })).toHaveText("更多");

````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese reads her invitation to Acme, as a guest, in Chinese", async ({
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(5).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese reads her invitation to Acme, as a guest, in Chinese", async ({
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 301 条、`@nerve/ui` 14 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 115 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/smoke/s7-chinese.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx' web/apps/web/core/components/comments/card/display.tsx web/apps/web/core/components/comments/comment-reaction.tsx web/apps/web/core/components/comments/quick-actions.tsx web/apps/web/core/components/core/description-versions/modal.tsx web/apps/web/core/components/cycles/cycles-view-header.tsx web/apps/web/core/components/cycles/quick-actions.tsx web/apps/web/core/components/issues/issue-detail/issue-activity/activity-filter.tsx web/apps/web/core/components/issues/issue-detail/issue-activity/sort-root.tsx web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx web/apps/web/core/components/issues/peek-overview/header.tsx web/apps/web/core/components/modules/module-view-header.tsx web/apps/web/core/components/modules/quick-actions.tsx web/apps/web/core/components/profile/sidebar.tsx web/apps/web/core/components/project/search-projects.tsx web/apps/web/core/components/rich-filters/filters-toggle.tsx web/apps/web/core/components/settings/mobile/nav.tsx web/apps/web/core/components/settings/profile/modal.tsx web/apps/web/core/components/settings/workspace/sidebar/header.tsx web/apps/web/core/components/sidebar/sidebar-toggle-button.tsx web/apps/web/core/components/views/quick-actions.tsx web/apps/web/core/components/views/view-list-header.tsx web/apps/web/core/components/workspace-notifications/sidebar/filters/menu/root.tsx web/apps/web/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx web/apps/web/core/components/workspace-notifications/sidebar/header/options/root.tsx web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/packages/i18n/src/locales/en/accessibility.json web/packages/i18n/src/locales/en/navigation.json web/packages/i18n/src/locales/zh-CN/accessibility.json web/packages/i18n/src/locales/zh-CN/navigation.json web/packages/propel/package.json web/packages/propel/src/emoji-icon-picker/logo.tsx web/packages/propel/src/emoji-reaction/emoji-reaction.tsx web/packages/propel/src/icon-button/helper.tsx
```
```bash
git commit -m "feat(M3/closeout): every IconButton has a name, required at the root

IconButton's aria-label is a required string, so each of its callers names
its button through t() and a new one cannot be unnamed. The emoji
reactions take their names and are keyed by their emoji, the sidebar's
toggle and workspace menu buttons get the keys they lacked, and a logo's
icon ligature or emoji is hidden from the name of the link it is in.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 项目页的搜索按钮没有名字 | `tsc` | 静态 |
| `T7.2` | 侧边栏的展开按钮的名字是英文 | 故事 S7 | 端到端 |
| `T7.3` | 设置的返回按钮的名字是英文 | 故事 S7 | 端到端 |
| `T7.4` | 项目页的搜索按钮的名字是英文 | 故事 S7 | 端到端 |
| `T7.5` | 工作区菜单的按钮名字在中文里没有定义（K9） | ``、故事 S7 | vitest；端到端 |
| `T7.6` | 扩展侧边栏的"更多"是英文 | 故事 S7 | 端到端 |
| `T7.7` | 项目的表情图标并进链接的名字（K3g） | 故事 S7 | 端到端 |
| `T7.8` | 表情回应又以下标为 key：oxlint 的 `no-array-index-key` 超出 propel 的上限 | `@nerve/propel` 的 oxlint（`check:lint`） | 静态 |

---

### Task 8: 收尾的键盘清单：颜色框、状态表单的焦点、侧边栏、成员一行、项目下拉框；S8

**Files:**
- Create: `e2e/stories/smoke/s8-sidebar.spec.ts`、`web/apps/web/core/components/common/color-picker.tsx`、`web/apps/web/core/components/dropdowns/project/base.test.tsx`
- Modify: `e2e/fixtures/project-pages.ts`、`e2e/stories/project/p1-create-project.spec.ts`、`e2e/stories/project/p2-visibility.spec.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`e2e/stories/project/p6-states-page.spec.ts`、`e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/core/components/dropdowns/project/base.tsx`、`web/apps/web/core/components/labels/create-update-label-inline.tsx`、`web/apps/web/core/components/project-states/create-update/form.tsx`、`web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/sidebar/resizable-sidebar.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/package.json`

**Interfaces:**（spec 2.9）
- `components/common/color-picker.tsx`（新文件）：`TwitterPicker`，输入框的 `outline` 是浏览器的（`"revert"`）；状态的表单、标签的行内表单用它。状态的表单打开时 `setFocus("name")`。
- 侧边栏：项目列表的"+"是 `opacity-0`，`group-hover`、`focus-visible` 时显示；项目一行的菜单是 `CustomMenu` 的省略号按钮（`menuButtonOnClick`；`actionSectionRef` 和它的点外面检测删除），行在 `focus-within` 时显示；手柄是 `span`；展开箭头 `tabIndex={-1}`。`resizable-sidebar.tsx`：收起时主侧边栏 `inert`，没有浮出时副本 `inert`；从不设置的 `isHoveringTrigger` 和它的副作用删除（web 的上限 301 → 300）。项目一行的 `Disclosure.Button` 不再有 `aria-label`：它的名字是项目的名字（`Logo` 已 `aria-hidden`，Task 7），展开与否由 `aria-expanded` 说出；展开箭头的名字照旧（F6，A-M2）。
- 项目成员页一行的菜单名为 `common.options`，`focus-visible:opacity-100`，菜单项经 `t()`。项目下拉框：`tabIndex` 给 `DropdownButton`，根不再可以聚焦。
- `e2e/fixtures/project-pages.ts` 的 `pageContent(page)`：页面的内容（应用的 `main` 里的 `main`，侧边栏之外）。侧边栏的一行现在也以项目为名，P1 在这里找项目的卡片，P2 找页头的面包屑。

**Tests:** `dropdowns/project/base.test.tsx`（新文件）。新的故事 S8（`e2e/stories/smoke/s8-sidebar.spec.ts`），以项目的名字找到一行；P6：名称有焦点、十六进制框有轮廓；P5：一行的菜单由 Tab 到达，以"Options"为名；S7：项目成员一行的菜单的"离开"。

- [ ] **Step 1: 键盘**

`web/apps/web/core/components/common/color-picker.tsx`（新文件，16 行）：

````file web/apps/web/core/components/common/color-picker.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { TwitterPicker } from "react-color";
import type { TwitterPickerProps } from "react-color";

// react-color draws its hex field with an inline outline: none, which hides where the keyboard is. "revert" gives the
// field the browser's own outline back, which shows as the keyboard reaches it.
const STYLES = { default: { input: { outline: "revert" } } };

/** react-color's TwitterPicker, whose hex field shows the keyboard's focus. */
export function ColorPicker(props: TwitterPickerProps) {
  return <TwitterPicker {...props} styles={STYLES} />;
}
````

`web/apps/web/core/components/dropdowns/project/base.tsx`（修改，2 处）：

````old web/apps/web/core/components/dropdowns/project/base.tsx
        variant={buttonVariant}
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
        variant={buttonVariant}
        tabIndex={tabIndex}
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
      ref={dropdownRef}
      tabIndex={tabIndex}
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
      ref={dropdownRef}
````

`web/apps/web/core/components/labels/create-update-label-inline.tsx`（修改，3 处）：

````old web/apps/web/core/components/labels/create-update-label-inline.tsx
import { observer } from "mobx-react";
import { TwitterPicker } from "react-color";
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
import { observer } from "mobx-react";
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
import type { Label } from "@nerve/api-client";
// components
import { ColorPicker } from "@/components/common/color-picker";
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
                        <TwitterPicker
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
                        <ColorPicker
````

`web/apps/web/core/components/project-states/create-update/form.tsx`（修改，7 处）：

````old web/apps/web/core/components/project-states/create-update/form.tsx
import { TwitterPicker } from "react-color";
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
import { useEffect } from "react";
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
import { Popover } from "@nerve/ui";
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
import { Popover } from "@nerve/ui";
// components
import { ColorPicker } from "@/components/common/color-picker";
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
    watch,
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
    watch,
    setFocus,
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
  } = useForm<TStateFormData>({ defaultValues: data });
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
  } = useForm<TStateFormData>({ defaultValues: data });

  // the name has the focus once the form shows
  useEffect(() => {
    setFocus("name");
  }, [setFocus]);
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
              <TwitterPicker color={value} onChange={(picked) => onChange(picked.hex)} />
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
              <ColorPicker color={value} onChange={(picked) => onChange(picked.hex)} />
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
          rules={{ required: t("errors.field.required") }}
          render={({ field: { value, onChange } }) => (
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
          rules={{ required: t("errors.field.required") }}
          render={({ field: { value, onChange, ref } }) => (
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
                  value={value}
                  onChange={onChange}
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
                  value={value}
                  onChange={onChange}
                  ref={ref}
````

`web/apps/web/core/components/project/settings/member-columns.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/settings/member-columns.tsx
  const { rowData, workspaceSlug, own, removable, setRemoveMemberModal } = props;
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
  const { rowData, workspaceSlug, own, removable, setRemoveMemberModal } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
                buttonClassName="p-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
                ariaLabel={t("common.options")}
                buttonClassName="p-0.5 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity"
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
                    {own ? "Leave " : "Remove "}
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
                    {t(own ? "leave" : "remove")}
````

`web/apps/web/core/components/sidebar/resizable-sidebar.tsx`（修改，6 处）：

````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
  const [isResizing, setIsResizing] = useState(false);
  const [isHoveringTrigger, setIsHoveringTrigger] = useState(false);
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
  const [isResizing, setIsResizing] = useState(false);
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
    setShowPeek(false);
    setIsHoveringTrigger(false);
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
    setShowPeek(false);
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx

  useEffect(() => {
    if (!isAnySidebarDropdownOpen && isCollapsed && isHoveringTrigger) {
      handlePeekLeave();
    }
  }, [isAnySidebarDropdownOpen]);

````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx

````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
      setShowPeek(false);
      setIsHoveringTrigger(false);
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
      setShowPeek(false);
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label={t("aria_labels.sidebar.main")}
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label={t("aria_labels.sidebar.main")}
        // collapsed, it shows nothing: none of its controls is the keyboard's or assistive technology's
        inert={isCollapsed}
````
````old web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label={t("aria_labels.sidebar.peek")}
````
````new web/apps/web/core/components/sidebar/resizable-sidebar.tsx
        aria-label={t("aria_labels.sidebar.peek")}
        // its copy of the sidebar's controls is for the pointer that asks for it, the keyboard's only while it shows
        inert={!(isCollapsed && showPeek)}
````

`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`（修改，11 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  LogOutOutline,
  MoreHorizontalOutline,
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  LogOutOutline,
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  // refs
  const actionSectionRef = useRef<HTMLButtonElement | null>(null);
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  // refs
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const dragHandleRef = useRef<HTMLButtonElement | null>(null);
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const dragHandleRef = useRef<HTMLSpanElement | null>(null);
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx

  useOutsideClickDetector(actionSectionRef, () => setIsMenuActive(false));
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx

````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
              <button
                type="button"
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
              {/* the pointer's grip, no control of the keyboard's */}
              <span
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
              </button>
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
              </span>
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                    className={cn("flex w-full flex-grow items-center gap-1.5 text-left select-none", {})}
                    aria-label={
                      isProjectListOpen
                        ? t("aria_labels.projects_sidebar.close_project_menu")
                        : t("aria_labels.projects_sidebar.open_project_menu")
                    }
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                    className={cn("flex w-full flex-grow items-center gap-1.5 text-left select-none", {})}
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                  customButton={
                    <IconButton
                      ref={actionSectionRef}
                      variant="ghost"
                      size="sm"
                      icon={MoreHorizontalOutline}
                      aria-label={t("aria_labels.projects_sidebar.toggle_quick_actions_menu")}
                      onClick={() => setIsMenuActive(!isMenuActive)}
                      className="text-placeholder"
                    />
                  }
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                  ellipsis
                  buttonClassName="text-placeholder"
                  menuButtonOnClick={() => setIsMenuActive(!isMenuActive)}
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                    "pointer-events-none flex-shrink-0 opacity-0 group-hover/project-item:pointer-events-auto group-hover/project-item:opacity-100",
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                    "pointer-events-none flex-shrink-0 opacity-0 group-hover/project-item:pointer-events-auto group-hover/project-item:opacity-100 focus-within:pointer-events-auto focus-within:opacity-100",
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                  )}
                  customButtonClassName="grid place-items-center"
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                  )}
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                    onClick={() => setIsProjectListOpen(!isProjectListOpen)}
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                    onClick={() => setIsProjectListOpen(!isProjectListOpen)}
                    // the row's own button opens and closes the project as this does: the pointer's alone
                    tabIndex={-1}
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                      className="hidden text-placeholder group-hover:inline-flex"
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                      className="text-placeholder opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 301",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 300",
````

- [ ] **Step 2: 测试**

`e2e/fixtures/project-pages.ts`（修改，2 处）：

````old e2e/fixtures/project-pages.ts
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
````
````new e2e/fixtures/project-pages.ts
  await memberRow(page, email).getByRole("button", { name: "Options" }).click();
````
````old e2e/fixtures/project-pages.ts
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
````
````new e2e/fixtures/project-pages.ts
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}

/**
 * What page shows beside its sidebars: the main region within the app's, which holds them too. A project's row in the
 * sidebar is named by the project, as its card and its header's crumb are: a story finds those here.
 */
export function pageContent(page: Page): Locator {
  return page.getByRole("main").getByRole("main");
````

`e2e/stories/project/p1-create-project.spec.ts`（修改，3 处）：

````old e2e/stories/project/p1-create-project.spec.ts
import { tabCycle } from "../../fixtures/project-pages";
````
````new e2e/stories/project/p1-create-project.spec.ts
import { pageContent, tabCycle } from "../../fixtures/project-pages";
````
````old e2e/stories/project/p1-create-project.spec.ts
  await expect(page.getByRole("heading", { name: "Web App", exact: true })).toBeVisible();
````
````new e2e/stories/project/p1-create-project.spec.ts
  await expect(pageContent(page).getByRole("heading", { name: "Web App", exact: true })).toBeVisible();
````
````old e2e/stories/project/p1-create-project.spec.ts
  await expect(page.getByRole("heading", { name: "Ops", exact: true })).toBeVisible();
````
````new e2e/stories/project/p1-create-project.spec.ts
  await expect(pageContent(page).getByRole("heading", { name: "Ops", exact: true })).toBeVisible();
````

`e2e/stories/project/p2-visibility.spec.ts`（修改，2 处）：

````old e2e/stories/project/p2-visibility.spec.ts
import { PROJECT_MEMBER, valued } from "../../fixtures/mounts";
````
````new e2e/stories/project/p2-visibility.spec.ts
import { PROJECT_MEMBER, valued } from "../../fixtures/mounts";
import { pageContent } from "../../fixtures/project-pages";
````
````old e2e/stories/project/p2-visibility.spec.ts
  await page.getByRole("button", { name: "Web", exact: true }).first().focus();
````
````new e2e/stories/project/p2-visibility.spec.ts
  await pageContent(page).getByRole("button", { name: "Web", exact: true }).first().focus();
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，1 处）：

````old e2e/stories/project/p5-project-members.spec.ts
  await expect(memberRow(page, ann.email).getByRole("button", { name: "Guest", exact: true })).toBeVisible();

````
````new e2e/stories/project/p5-project-members.spec.ts
  await expect(memberRow(page, ann.email).getByRole("button", { name: "Guest", exact: true })).toBeVisible();

  // max's row menu shows, and is marked, as Tab reaches it from his avatar.
  const maxsMenu = memberRow(page, max.email).getByRole("button", { name: "Options" });
  await memberRow(page, max.email).getByRole("link").first().focus();
  await page.keyboard.press("Tab");
  await expect(maxsMenu).toBeFocused();
  await expect(maxsMenu).toHaveCSS("opacity", "1");
  await expect(maxsMenu).not.toHaveCSS("outline-style", "none");

````

`e2e/stories/project/p6-states-page.spec.ts`（修改，2 处）：

````old e2e/stories/project/p6-states-page.spec.ts
  await addTo(page, "Started").click();
````
````new e2e/stories/project/p6-states-page.spec.ts
  await addTo(page, "Started").click();
  await expect(page.locator("#name")).toBeFocused();
````
````old e2e/stories/project/p6-states-page.spec.ts
  await colour.click();
````
````new e2e/stories/project/p6-states-page.spec.ts
  await colour.click();
  // its hex field shows where the keyboard is (react-color draws it with no outline of its own)
  const hex = page.locator('input[id^="rc-editable-input"]');
  await hex.focus();
  await expect(hex).not.toHaveCSS("outline-style", "none");
````

`e2e/stories/smoke/s7-chinese.spec.ts`（修改，1 处）：

````old e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expect(page.getByRole("switch", { name: "访客访问" })).toBeVisible();
  // His row's menu, by its name, offers his leaving.
  await page.getByRole("button", { name: "选项", exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "离开" })).toBeVisible();
  await page.keyboard.press("Escape");
````

`e2e/stories/smoke/s8-sidebar.spec.ts`（新文件，65 行）：

````file e2e/stories/smoke/s8-sidebar.spec.ts
import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// S8, the sidebar by the keyboard (M3 closeout, P11's keyboard list): what the keyboard and assistive technology
// reach in the workspace's sidebar is what it shows.

test("S8 (page): the sidebar's copy for peeking holds no control while it is hidden, and the sidebar none while it is collapsed; the new project's button and a project's menu each show and are marked as Tab reaches them, the menu one Tab stop; a project's row is named by its project, and its grip is none", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await createProject(api, tokens.access_token, slug, { name: "Web", identifier: "WEB" });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  const sidebar = page.getByRole("complementary", { name: "Main sidebar" });
  // whether each of the two Project navigation buttons, the sidebar's and its copy's for peeking, is inert: neither
  // the keyboard nor assistive technology reaches it (Playwright's roles do not leave inert elements out)
  const navigation = page.getByRole("button", { name: "Project navigation" });
  const inert = () => navigation.evaluateAll((buttons) => buttons.map((button) => button.closest("[inert]") !== null));

  // The sidebar's copy for peeking, unseen until the pointer asks for it, holds no control.
  await expect(navigation).toHaveCount(2);
  expect(await inert()).toEqual([false, true]);

  // The new project's button, unseen until the pointer is over the list's heading, shows and is marked as Tab reaches
  // it from the heading.
  const create = sidebar.getByRole("button", { name: "Create new project" });
  await sidebar.getByRole("button", { name: "Close projects menu" }).first().focus();
  await page.keyboard.press("Tab");
  await expect(create).toBeFocused();
  await expect(create).toHaveCSS("opacity", "1");
  await expect(create).not.toHaveCSS("outline-style", "none");

  // Web's menu: one Tab stop after Web's row, which is named Web, shown and marked as Tab reaches it; its chevron,
  // which does what the row does, and its grip, which the pointer drags, are none, though the pointer over the row
  // shows them.
  const menu = sidebar.getByRole("button", { name: "Toggle quick actions menu" });
  await expect(menu).toHaveCount(1);
  const row = sidebar.getByRole("button", { name: "Web", exact: true });
  await row.focus();
  await page.keyboard.press("Tab");
  await expect(menu).toBeFocused();
  await expect(menu.locator("xpath=ancestor::div[@data-main-menu][1]")).toHaveCSS("opacity", "1");
  await expect(menu).not.toHaveCSS("outline-style", "none");
  await row.hover();
  await page.keyboard.press("Tab");
  await expect(menu).not.toBeFocused();
  await expect(sidebar.getByRole("button", { name: "Open project menu" })).not.toBeFocused();
  await expect(sidebar.locator(".cursor-grab:is(button, a, [tabindex]):not([tabindex='-1'])")).toHaveCount(0);

  // Collapsed, the sidebar holds no control the keyboard reaches; opened again, it does.
  await sidebar.getByRole("button", { name: "Toggle sidebar" }).click();
  await expect.poll(inert).toEqual([true, true]);
  // the page's header has its own, outside either sidebar, seen while the sidebar is collapsed
  await page.locator("xpath=//button[@aria-label='Toggle sidebar'][not(ancestor::*[@role='complementary'])]").click();
  await expect.poll(inert).toEqual([false, true]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
````

`web/apps/web/core/components/dropdowns/project/base.test.tsx`（新文件，33 行）：

````file web/apps/web/core/components/dropdowns/project/base.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { ProjectDropdownBase } from "./base";

// Where the project dropdown takes the keyboard (M3 closeout, P11's keyboard list): it renders on the server, closed,
// with the tabIndex its caller gives it. As in every dropdown of dropdowns/, its Tab stop is the DropdownButton in
// it, where the caller's tabIndex puts it (the combobox's own button, round it, Headless UI keeps out of the Tab
// order); its root, a box round the button and the list, is no Tab stop and no control (role presentation).

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

describe("ProjectDropdownBase", () => {
  it("puts its DropdownButton where its caller's tabIndex says, its root no Tab stop", () => {
    const markup = renderToStaticMarkup(
      <ProjectDropdownBase
        buttonVariant="border-with-text"
        getProjectById={() => undefined}
        projectIds={[]}
        multiple={false}
        value={null}
        onChange={() => {}}
        tabIndex={0}
      />
    );
    const tags = markup.match(/<(div|button)[^>]*>/g) ?? [];
    expect(tags.map((tag) => /tabindex="(-?\d+)"/.exec(tag)?.[1] ?? "none")).toEqual(["none", "-1", "0"]);
  });
});
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `69 rules, 3 exceptions, no hits.`；web 的 oxlint 300 条、`@nerve/ui` 14 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 116 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/project-pages.ts e2e/stories/project/p1-create-project.spec.ts e2e/stories/project/p2-visibility.spec.ts e2e/stories/project/p5-project-members.spec.ts e2e/stories/project/p6-states-page.spec.ts e2e/stories/smoke/s7-chinese.spec.ts e2e/stories/smoke/s8-sidebar.spec.ts web/apps/web/core/components/common/color-picker.tsx web/apps/web/core/components/dropdowns/project/base.test.tsx web/apps/web/core/components/dropdowns/project/base.tsx web/apps/web/core/components/labels/create-update-label-inline.tsx web/apps/web/core/components/project-states/create-update/form.tsx web/apps/web/core/components/project/settings/member-columns.tsx web/apps/web/core/components/sidebar/resizable-sidebar.tsx web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/package.json
```
```bash
git commit -m "feat(M3/closeout): the closeout's keyboard list: the hex field, the state form's focus, the sidebar and the rows' menus

The colour pickers' hex field shows the keyboard's focus; the state form
opens on its name; the sidebar's new project button and a project's menu
show as Tab reaches them, the menu one Tab stop and the grip none, and a
project's row is named by its project; the
hidden copy of the sidebar and the collapsed sidebar are inert; the
project members' row menu has a name and shows on focus; the project
dropdown's Tab stop is its button. S8 walks the sidebar by keyboard.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | 颜色弹层的十六进制输入框又藏起键盘的焦点（K3a） | 故事 P6 | 端到端 |
| `T8.2` | 状态的表单打开时焦点留在原处（K3f） | 故事 P6 | 端到端 |
| `T8.3` | 侧边栏的"+"在指针经过之前不在 Tab 的顺序里（K3c） | 故事 S8 | 端到端 |
| `T8.4` | Tab 到项目一行的菜单时看不见它（K3c） | 故事 S8 | 端到端 |
| `T8.5` | 指针在项目一行上时，它的展开箭头是一个停留点 | 故事 S8 | 端到端 |
| `T8.6` | 项目的拖动手柄又是一个停留点（K3e） | 故事 S8 | 端到端 |
| `T8.7` | 项目成员一行的菜单没有名字 | 故事 P5 | 端到端 |
| `T8.8` | Tab 到项目成员一行的菜单时看不见它（K3c） | 故事 P5 | 端到端 |
| `T8.9` | 项目成员一行的菜单项是英文 | 故事 S7 | 端到端 |
| `T8.10` | 项目下拉框的按钮不在调用方的 Tab 顺序里（K3d） | `base.test.tsx` | vitest |
| `T8.11` | 收起的侧边栏仍能用 Tab 走到 | 故事 S8 | 端到端 |
| `T8.12` | 没有浮出的侧边栏副本仍能用 Tab 走到 | 故事 S8 | 端到端 |
| `T8.13` | 项目一行的按钮回到"Open project menu"的名字，每一行都一样（F6） | 故事 S8 | 端到端 |

---

### Task 9: 共用的写法：`showTooltip`、复制、问题码、拒绝的文字；读了却没有定义的键

**Files:**
- Create: `tools/i18n-keys.mjs`
- Modify: `Makefile`、`docs/v0/M1-frontend-trim/M1-design.md`、`docs/v0/M3-workspace-project/M3-design.md`、`e2e/stories/identity/a11-api-tokens.spec.ts`、`e2e/stories/smoke/s6-dialogs.spec.ts`、`knip.jsonc`、`tools/keywords.json`、`tools/keywords.mjs`、`web/apps/web/core/components/api-token/modal/generated-token-details.tsx`、`web/apps/web/core/components/appearance/theme-switcher.tsx`、`web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx`、`web/apps/web/core/components/dropdowns/member/avatar.tsx`、`web/apps/web/core/components/dropdowns/member/base.tsx`、`web/apps/web/core/components/inbox/sidebar/inbox-list-item.tsx`、`web/apps/web/core/components/issues/attachment/attachment-list-item.tsx`、`web/apps/web/core/components/issues/issue-detail/sidebar.tsx`、`web/apps/web/core/components/issues/peek-overview/properties.tsx`、`web/apps/web/core/components/modules/module-card-item.tsx`、`web/apps/web/core/components/modules/module-list-item-action.tsx`、`web/apps/web/core/components/power-k/config/miscellaneous-commands.ts`、`web/apps/web/core/components/settings/profile/content/pages/security.tsx`、`web/apps/web/core/components/views/view-list-item-action.tsx`、`web/apps/web/core/components/workspace/use-create-workspace.ts`、`web/apps/web/core/components/workspace/use-invitation-answer.ts`、`web/apps/web/core/hooks/use-copy-link.test.ts`、`web/apps/web/core/hooks/use-copy-link.ts`、`web/apps/web/core/hooks/use-refusal-toast.test.ts`、`web/apps/web/core/hooks/use-refusal-toast.ts`、`web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`、`web/apps/web/core/lib/api-error.test.ts`、`web/apps/web/core/lib/api-error.ts`、`web/apps/web/core/lib/form-refusal.ts`、`web/packages/constants/src/filter.ts`、`web/packages/i18n/src/locales/en/power-k.json`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/power-k.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`

**Interfaces:**（spec 2.10）
- `ButtonAvatars` 的 `showTooltip` 删除，10 处调用。
- `core/hooks/use-copy-link.ts`：不导出的 `useCopy(write, titleKey)`，`useCopyLink`、`useCopyText`；API 令牌弹窗和命令面板的复制经它们；`workspace_settings.token_not_copied`、命令面板的复制失败的键删除。
- `core/lib/api-error.ts` 的 `problemCode(error)`：五处问题码经它。`use-refusal-toast.ts` 的 `useRefusalMessage`：`useRefusalToast` 和主题切换的失败经它。
- `tools/keywords.json`：`copy-hook`（20 个例外，`until` 是各 M 的收尾）；`refusal-toast` 也看 `setPromiseToast(…errorMessageKey(`；`sign-out-toast` 加两种写法；`phase` 是 `M3/closeout`。
- `tools/i18n-keys.mjs`（新文件）进 `make lint-web` 和 knip 的入口；`@nerve/constants` 的 `filter.ts` 中没有读者的 `date_filters.1_month_ago` 删除。
- `tools/keywords.mjs` 的说明、M1 设计 7.4：`until` 写 `M<n>/closeout` 的例外，在那个 M 的收尾开始、`phase` 改为它时到期；每个 Phase 开始时改 `phase`（spec 第 3 节第 5 条）。M3 设计 7.10 的规则表补上 P9 的三条、P10 的两条和 `copy-hook`。

**Tests:** `use-copy-link.test.ts`、`api-error.test.ts`、`use-refusal-toast.test.ts`。A11：点令牌复制到剪贴板并提示；S6：命令面板复制当前页面的地址并提示。

- [ ] **Step 1: 共用的写法和检查**

`Makefile`（修改，2 处）：

````old Makefile
# 关键词守卫（tools/keywords.mjs，规则在 tools/keywords.json）检查整个仓库，不属于任何工作区包，所以不经过 turbo
````
````new Makefile
# 关键词守卫（tools/keywords.mjs，规则在 tools/keywords.json）检查整个仓库，翻译键的检查（tools/i18n-keys.mjs）读 web
# 应用和它的几个包，都不属于任何工作区包，所以不经过 turbo
````
````old Makefile
lint-web: ## 关键词守卫；前端类型检查、oxlint（警告数等于上限）、格式检查、中英文翻译键一致（需要 Node）
	node tools/keywords.mjs
````
````new Makefile
lint-web: ## 关键词守卫；读到的翻译键都有定义；前端类型检查、oxlint（警告数等于上限）、格式检查、中英文翻译键一致（需要 Node）
	node tools/keywords.mjs
	node tools/i18n-keys.mjs
````

`docs/v0/M1-frontend-trim/M1-design.md`（修改，1 处）：

````old docs/v0/M1-frontend-trim/M1-design.md
- **例外**：必须精确，包括规则编号、路径、命中的文本或符号、理由和到期的 Phase 或 M。一条例外默认只覆盖一处命中；同一个文件里同一段原文出现多处时，例外里写明处数（`count`），实际处数必须与它相等，多了少了都失败——否则一条例外会把后来新增的同样一处也顺带盖住。同一处只能登记一条例外，重复登记报错。一条例外不再命中任何内容时（例如代码已经删了），工具报错，提醒删掉它。到期也由工具管：规则文件里写着当前的 Phase（`phase`），例外的 `until` 到了或早于它，工具就报错——也就是说，到期的那个 Phase 一开始，这条例外就必须已经删掉，而不是靠人记得。确实永远不会到期的（例如顶级域名数据里恰好有个 `wiki`），`until` 写 v0 之外的 `M9`，理由里写明为什么它不可能在 v0 内消失；收尾时重新过一遍所有 `until` 超出 M8 的例外（11 节）。
````
````new docs/v0/M1-frontend-trim/M1-design.md
- **例外**：必须精确，包括规则编号、路径、命中的文本或符号、理由和到期的 Phase 或 M。一条例外默认只覆盖一处命中；同一个文件里同一段原文出现多处时，例外里写明处数（`count`），实际处数必须与它相等，多了少了都失败——否则一条例外会把后来新增的同样一处也顺带盖住。同一处只能登记一条例外，重复登记报错。一条例外不再命中任何内容时（例如代码已经删了），工具报错，提醒删掉它。到期也由工具管：规则文件里写着当前的 Phase（`phase`），例外的 `until` 到了或早于它，工具就报错——也就是说，到期的那个 Phase 一开始，这条例外就必须已经删掉，而不是靠人记得。例外的 `until` 也可以写 `M<n>/closeout`：在那个 M 的收尾开始、规则文件的 `phase` 改为 `M<n>/closeout` 时到期；每个 M 的每个 Phase 开始时改 `phase`。确实永远不会到期的（例如顶级域名数据里恰好有个 `wiki`），`until` 写 v0 之外的 `M9`，理由里写明为什么它不可能在 v0 内消失；收尾时重新过一遍所有 `until` 超出 M8 的例外（11 节）。
````

`docs/v0/M3-workspace-project/M3-design.md`（修改，3 处）：

````old docs/v0/M3-workspace-project/M3-design.md
### 7.10 关键词守卫的新规则（`tools/keywords.json`，P8a、P8b）
````
````new docs/v0/M3-workspace-project/M3-design.md
### 7.10 关键词守卫的新规则（`tools/keywords.json`，P8a 起）
````
````old docs/v0/M3-workspace-project/M3-design.md
| `user-settings` | `IUserSettings`、`currentUserSettings`、`fetchCurrentUserSettings` |
````
````new docs/v0/M3-workspace-project/M3-design.md
| `user-settings` | `IUserSettings`、`currentUserSettings`、`fetchCurrentUserSettings` |
| `workspaces-list-fetch`（P9） | 在 `useWorkspacesFetch` 之外取调用者的工作区列表（调 store 的 `fetchWorkspaces`）；store 自己和测试不算 |
| `refusal-toast`（P9，收尾放宽） | 在 `useRefusalToast` 之外用 `setToast` 按 `errorMessageKey` 说拒绝；收尾加上 `setPromiseToast` 的失败一支（它的 message 是 `useRefusalMessage`） |
| `sign-out-toast`（P9，收尾放宽） | 在 `useSignOut` 之外退出登录再自己提示；收尾加上从 store 取 `signOut` 的写法（`const { signOut } = useUser()`、`useUser().signOut`） |
| `project-leave`（P10） | 在 `useProjectMembershipChanges` 的 `leave` 之外调 store 的 `leaveProject` |
| `project-address-route`（P10） | P10、P11 改过的项目页从 router store 读项目和工作区（`currentProjectDetails`、`projectMemberIds`、`currentWorkspace`、不传 slug 和项目的 `allowPermissions`）；M4、M6、M7 改它们的页面时把那些文件加进规则的 `files` |
| `copy-hook`（收尾） | 在 `useCopyLink`、`useCopyText` 之外复制到剪贴板（`copyTextToClipboard`、`copyUrlToClipboard`、`navigator.clipboard`）；M4、M6、M7、M8 还自己复制的文件各有一条例外，`until` 是各自的 `M<n>/closeout`，到那个 M 的收尾开始、`phase` 改为它时到期 |
````
````old docs/v0/M3-workspace-project/M3-design.md
- 在 P8a、P8b 加入：P8a 加入工作区一侧的地址的模式和 `restricted-urls`、`in-system-invitations`、`user-settings`；P8b 补全项目一侧的模式、`/user-properties/` 的例外和不命中样例，删除 `project-invitations` 的例外，收紧 `plane-user-urls`。规则在数据层的第二段（P8b）合并时完整，在任何页面的 Phase 之前；P11 只核对 M3 加的例外只剩跨 M 的一条（`/user-properties/` 的两处，`until: "M4"`），`project-invitations` 的 `until: "M3"` 已删除。
````
````new docs/v0/M3-workspace-project/M3-design.md
- 在 P8a、P8b 加入：P8a 加入工作区一侧的地址的模式和 `restricted-urls`、`in-system-invitations`、`user-settings`；P8b 补全项目一侧的模式、`/user-properties/` 的例外和不命中样例，删除 `project-invitations` 的例外，收紧 `plane-user-urls`。规则在数据层的第二段（P8b）合并时完整，在任何页面的 Phase 之前；P11 只核对 M3 加的例外只剩跨 M 的一条（`/user-properties/` 的两处，`until: "M4"`），`project-invitations` 的 `until: "M3"` 已删除。
- P9、P10 和收尾加入的规则在表的后几行，括号里是加入它的 Phase；每条规则的 `why` 写明它守的那一处和不算的地方。
````

`knip.jsonc`（修改，1 处）：

````old knip.jsonc
      // 关键词守卫由 make lint-web 调用；常见密码名单的生成脚本由人手动运行（M2 设计 3.8）。package.json 的脚本中都看不到
      "entry": ["tools/keywords.mjs", "tools/password-blocklist/build.mjs"],
````
````new knip.jsonc
      // 关键词守卫和翻译键的检查由 make lint-web 调用；常见密码名单的生成脚本由人手动运行（M2 设计 3.8）。package.json 的脚本中都看不到
      "entry": ["tools/keywords.mjs", "tools/i18n-keys.mjs", "tools/password-blocklist/build.mjs"],
````

`tools/i18n-keys.mjs`（新文件，66 行）：

````file tools/i18n-keys.mjs
// Translation keys read (M3 closeout): fails when the web app reads an i18n key that the English locale does not
// define, which the page would show as the key itself. A key read is one the code holds as a literal: a t("…") call's,
// a Translated's i18nKey="…", a constant's i18n_…: "…", and any string of dotted words whose first word is a top-level
// key of the locale ("project_settings.states.groups.backlog"), wherever the code holds it. A key the locale defines
// only longer keys under (a prefix the code completes) counts as defined; a key the code builds from a template is not
// seen. The sources: web/apps/web's app and core, the constants and types packages; tests and their fakes aside. The
// keys are en's: check:sync keeps zh-CN's the same.
// usage: node tools/i18n-keys.mjs   (from anywhere)
import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..");
const LOCALE = "web/packages/i18n/src/locales/en";
const SOURCES = ["web/apps/web/app", "web/apps/web/core", "web/packages/constants/src", "web/packages/types/src"];

// every key of the locale's files, dotted
const keys = new Set();
function flatten(node, prefix) {
  if (node !== null && typeof node === "object" && !Array.isArray(node)) {
    for (const [key, value] of Object.entries(node)) flatten(value, prefix ? `${prefix}.${key}` : key);
  } else {
    keys.add(prefix);
  }
}
for (const file of fs.readdirSync(path.join(ROOT, LOCALE)).filter((name) => name.endsWith(".json"))) {
  flatten(JSON.parse(fs.readFileSync(path.join(ROOT, LOCALE, file), "utf8")), "");
}
const prefixes = new Set();
for (const key of keys) {
  const parts = key.split(".");
  for (let i = 1; i < parts.length; i++) prefixes.add(parts.slice(0, i).join("."));
}
const tops = [...new Set([...keys].filter((key) => key.includes(".")).map((key) => key.split(".")[0]))];

const KEY = "[a-z0-9_]+(?:\\.[a-z0-9_]+)*";
const READS = [
  new RegExp(`\\bt\\(\\s*"(${KEY})"`, "g"),
  new RegExp(`\\bi18nKey="(${KEY})"`, "g"),
  new RegExp(`\\bi18n_[a-z_]+:\\s*"(${KEY})"`, "g"),
  new RegExp(`"((?:${tops.join("|")})(?:\\.[a-z0-9_]+)+)"`, "g"),
];

const read = new Set();
const undefinedKeys = [];
for (const source of SOURCES) {
  for (const name of fs.readdirSync(path.join(ROOT, source), { recursive: true })) {
    const base = path.basename(name);
    if (!/\.tsx?$/.test(base) || base.includes(".test.") || base.startsWith("fake-")) continue;
    const file = path.join(source, name);
    const text = fs.readFileSync(path.join(ROOT, file), "utf8");
    const found = new Set(READS.flatMap((pattern) => [...text.matchAll(pattern)].map((match) => match[1])));
    for (const key of found) {
      read.add(key);
      if (!keys.has(key) && !prefixes.has(key)) undefinedKeys.push(`${file}: ${key}`);
    }
  }
}

if (undefinedKeys.length > 0) {
  console.error(undefinedKeys.toSorted().join("\n"));
  console.error(
    `i18n-keys: ${undefinedKeys.length} keys read that ${LOCALE} does not define: define them, or read others.`
  );
  process.exit(1);
}
console.log(`i18n-keys: ${read.size} keys read, each defined in ${LOCALE}.`);
````

`tools/keywords.json`（修改，12 处）：

````old tools/keywords.json
  "phase": "M2/closeout",
````
````new tools/keywords.json
  "phase": "M3/closeout",
````
````old tools/keywords.json
      "why": "页面用 setToast 按 errorMessageKey 把失败说成提示只有一处：useRefusalToast（web/apps/web/core/hooks/use-refusal-toast.ts），按 errorMessageKey 给出 nerve 的原因，标题默认 toast.error（M3/P9 修复轮，裁定 F-4）；别处的 setToast 里再写 errorMessageKey 就是又一份。主题切换的 setPromiseToast 自己说失败，本规则不管，留在 P9 规格 §5 的收尾清单上。错误放在字段下、对话框里的不是提示，不算；测试不算",
````
````new tools/keywords.json
      "why": "页面用 setToast 按 errorMessageKey 把失败说成提示只有一处：useRefusalToast（web/apps/web/core/hooks/use-refusal-toast.ts），按 errorMessageKey 给出 nerve 的原因，标题默认 toast.error（M3/P9 修复轮，裁定 F-4）；别处的 setToast 里再写 errorMessageKey 就是又一份。setPromiseToast 的失败一支也一样，它的 message 是同一模块的 useRefusalMessage（主题切换，M3 收尾）。错误放在字段下、对话框里的不是提示，不算；测试不算",
````
````old tools/keywords.json
        "source": "setToast\\(\\{[^}]*\\berrorMessageKey\\(",
````
````new tools/keywords.json
        "source": "setToast\\(\\{[^}]*\\berrorMessageKey\\(|setPromiseToast\\([^;]*\\berrorMessageKey\\(",
````
````old tools/keywords.json
          "      setToast({\n        type: TOAST_TYPE.ERROR,\n        title: t(\"auth.common.password.toast.change_password.error.title\"),\n        message: t(errorMessageKey(error)),\n      });"
````
````new tools/keywords.json
          "      setToast({\n        type: TOAST_TYPE.ERROR,\n        title: t(\"auth.common.password.toast.change_password.error.title\"),\n        message: t(errorMessageKey(error)),\n      });",
          "      setPromiseToast(updatePromise, {\n        loading: t(\"power_k.preferences_actions.toast.theme.updating\"),\n        error: {\n          title: t(\"toast.error\"),\n          message: (error) => t(errorMessageKey(error)),\n        },\n      });"
````
````old tools/keywords.json
          "      setToast({ type: TOAST_TYPE.SUCCESS, title: t(\"toast.success\"), message: t(\"profile_updated\") });"
````
````new tools/keywords.json
          "      setToast({ type: TOAST_TYPE.SUCCESS, title: t(\"toast.success\"), message: t(\"profile_updated\") });",
          "      setPromiseToast(updatePromise, {\n        error: { title: t(\"toast.error\"), message: refusalMessage },\n      });\n      const key = errorMessageKey(error);"
````
````old tools/keywords.json
        "files": {
          "hit": [
            "web/apps/web/core/components/labels/delete-label-modal.tsx",
````
````new tools/keywords.json
        "files": {
          "hit": [
            "web/apps/web/core/components/appearance/theme-switcher.tsx",
            "web/apps/web/core/components/labels/delete-label-modal.tsx",
````
````old tools/keywords.json
      "why": "页面退出登录只有一处：useSignOut（web/apps/web/core/hooks/use-sign-out.ts），nerve 的登出失败时提示 auth.sign_out.toast.error.*、会话不变（M3/P9 修复轮，裁定 F-4）；别处直接调 store 的 signOut 再自己提示就是又一份，没有提示的一份就是没有处理的拒绝。测试不算",
````
````new tools/keywords.json
      "why": "页面退出登录只有一处：useSignOut（web/apps/web/core/hooks/use-sign-out.ts），nerve 的登出失败时提示 auth.sign_out.toast.error.*、会话不变（M3/P9 修复轮，裁定 F-4）；别处直接调 store 的 signOut 再自己提示就是又一份，没有提示的一份就是没有处理的拒绝；从 store 取 signOut（const { signOut } = useUser()、useUser().signOut）的地方也算，那一份不经 useSignOut（M3/P9 评审给出的第二个模式，M3 收尾加）。测试不算",
````
````old tools/keywords.json
        "source": "\\bauth\\.sign_out\\.toast\\.error\\b",
````
````new tools/keywords.json
        "source": "\\bauth\\.sign_out\\.toast\\.error\\b|\\{[^}]*\\bsignOut\\b[^}]*\\}\\s*=\\s*useUser\\(\\)|useUser\\(\\)\\.signOut\\b",
````
````old tools/keywords.json
          "        message: t(\"auth.sign_out.toast.error.message\"),"
````
````new tools/keywords.json
          "        message: t(\"auth.sign_out.toast.error.message\"),",
          "  const { signOut } = useUser();",
          "  const {\n    data: currentUser,\n    signOut,\n  } = useUser();",
          "    await useUser().signOut();"
````
````old tools/keywords.json
        "miss": ["  const signOut = useSignOut();", "        {t(\"sign_out\")}"],
````
````new tools/keywords.json
        "miss": [
          "  const signOut = useSignOut();",
          "        {t(\"sign_out\")}",
          "  const { data: currentUser } = useUser();",
          "  const { signOut } = useSession();"
        ],
````
````old tools/keywords.json
          ]
        }
      }
    }
  ],
````
````new tools/keywords.json
          ]
        }
      }
    },
    {
      "id": "copy-hook",
      "phase": "M3/closeout",
      "why": "页面复制到剪贴板只有一处：web/apps/web/core/hooks/use-copy-link.ts 的 useCopyLink（链接）和 useCopyText（原样的文字，如令牌），成功时说复制了什么，浏览器不让页面写剪贴板（没有权限，或 http）时说没能复制（M3 设计 7.7；P10 规格 §5，M3 收尾）。别处再调 copyTextToClipboard、copyUrlToClipboard 或 navigator.clipboard 就是又一份，没有失败一支的一份就是不说的失败。M4、M6、M7、M8 的页面中还自己复制的文件各有一条例外，到各自的收尾为止。编辑器包（@nerve/editor）的代码块和链接预览不经 web 的 hook，不在本规则内；测试不算",
      "files": {
        "source": "^web/apps/web/(?:app|core)/(?!hooks/use-copy-link\\.ts$)(?!.*\\.test\\.[jt]sx?$).*\\.[jt]sx?$",
        "flags": ""
      },
      "content": {
        "source": "\\bcopy(?:Text|Url)ToClipboard\\(|\\bnavigator\\.clipboard\\b",
        "flags": ""
      },
      "samples": {
        "hit": [
          "    void copyTextToClipboard(token)",
          "    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/cycles/${cycleId}`).then(() => {",
          "      navigator.clipboard\n        .writeText(text)"
        ],
        "miss": [
          "import { copyTextToClipboard, copyUrlToClipboard } from \"@nerve/utils\";",
          "  const copyLink = useCopyLink();",
          "        onClick={() => void copyText(tokenDetails.token, t(\"workspace_settings.token_copied\"))}"
        ],
        "files": {
          "hit": [
            "web/apps/web/core/components/api-token/modal/generated-token-details.tsx",
            "web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx"
          ],
          "miss": [
            "web/apps/web/core/hooks/use-copy-link.ts",
            "web/apps/web/core/hooks/use-copy-link.test.ts",
            "web/packages/editor/src/extensions/code/code-block-node-view.tsx"
          ]
        }
      }
    }
  ],
````
````old tools/keywords.json
      "until": "M4"
````
````new tools/keywords.json
      "until": "M4"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/create-issue-toast-action-items.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-detail-widgets/relations/helper.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/helper.ts",
      "match": "copyUrlToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-detail/identifier-text.tsx",
      "match": "navigator.clipboard",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx",
      "match": "copyTextToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-detail/links/link-item.tsx",
      "match": "copyTextToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/issues/peek-overview/header.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts",
      "match": "copyTextToClipboard(",
      "count": 3,
      "reason": "M4 的工作项页面自己复制到剪贴板，成功和失败各自说；M4 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M4/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/cycles/quick-actions.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M6 的迭代、模块页面自己复制到剪贴板，成功和失败各自说；M6 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M6/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/modules/links/list-item.tsx",
      "match": "copyTextToClipboard(",
      "reason": "M6 的迭代、模块页面自己复制到剪贴板，成功和失败各自说；M6 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M6/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/modules/quick-actions.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M6 的迭代、模块页面自己复制到剪贴板，成功和失败各自说；M6 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M6/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts",
      "match": "copyTextToClipboard(",
      "reason": "M6 的迭代、模块页面自己复制到剪贴板，成功和失败各自说；M6 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M6/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx",
      "match": "copyTextToClipboard(",
      "reason": "M6 的迭代、模块页面自己复制到剪贴板，成功和失败各自说；M6 改这些页面时改用 useCopyLink 或 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M6/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/inbox/content/inbox-issue-header.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M7 的收集箱和视图页面自己复制到剪贴板，成功和失败各自说；M7 改这些页面时改用 useCopyLink（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M7/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/views/quick-actions.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M7 的收集箱和视图页面自己复制到剪贴板，成功和失败各自说；M7 改这些页面时改用 useCopyLink（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M7/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/workspace/views/default-view-quick-action.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M7 的收集箱和视图页面自己复制到剪贴板，成功和失败各自说；M7 改这些页面时改用 useCopyLink（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M7/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/workspace/views/quick-action.tsx",
      "match": "copyUrlToClipboard(",
      "reason": "M7 的收集箱和视图页面自己复制到剪贴板，成功和失败各自说；M7 改这些页面时改用 useCopyLink（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M7/closeout"
    },
    {
      "rule": "copy-hook",
      "path": "web/apps/web/core/components/web-hooks/form/secret-key.tsx",
      "match": "copyTextToClipboard(",
      "reason": "M8 的 Webhook 页面自己复制密钥，成功和失败各自说；M8 改这个页面时改用 useCopyText（web/apps/web/core/hooks/use-copy-link.ts），删除本例外（M3 收尾）",
      "until": "M8/closeout"
````

`tools/keywords.mjs`（修改，1 处）：

````old tools/keywords.mjs
// (its "until" is at or before the file's top-level "phase").
````
````new tools/keywords.mjs
// (its "until" is at or before the file's top-level "phase"). An "until" of "M<n>/closeout" falls due
// when the top-level "phase" is set to "M<n>/closeout", which each M sets as its closeout starts (and
// "M<n>/P<k>" as each phase starts).
````

`web/apps/web/core/components/api-token/modal/generated-token-details.tsx`（修改，4 处）：

````old web/apps/web/core/components/api-token/modal/generated-token-details.tsx
import { CopyOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````new web/apps/web/core/components/api-token/modal/generated-token-details.tsx
import { CopyOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/api-token/modal/generated-token-details.tsx
import { renderFormattedDate, renderFormattedTime, copyTextToClipboard } from "@nerve/utils";
// types
````
````new web/apps/web/core/components/api-token/modal/generated-token-details.tsx
import { renderFormattedDate, renderFormattedTime } from "@nerve/utils";
// hooks
import { useCopyText } from "@/hooks/use-copy-link";
````
````old web/apps/web/core/components/api-token/modal/generated-token-details.tsx
  const copyApiToken = (token: string) => {
    void copyTextToClipboard(token)
      .then(() =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: `${t("success")}!`,
          message: t("workspace_settings.token_copied"),
        })
      )
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: `${t("error")}!`,
          message: t("workspace_settings.token_not_copied"),
        })
      );
  };
````
````new web/apps/web/core/components/api-token/modal/generated-token-details.tsx
  const copyText = useCopyText();
````
````old web/apps/web/core/components/api-token/modal/generated-token-details.tsx
        onClick={() => copyApiToken(tokenDetails.token)}
````
````new web/apps/web/core/components/api-token/modal/generated-token-details.tsx
        onClick={() => void copyText(tokenDetails.token, t("workspace_settings.token_copied"))}
````

`web/apps/web/core/components/appearance/theme-switcher.tsx`（修改，4 处）：

````old web/apps/web/core/components/appearance/theme-switcher.tsx
import { useUserProfile } from "@/hooks/store/user";
// lib
import { errorMessageKey } from "@/lib/error-messages";
````
````new web/apps/web/core/components/appearance/theme-switcher.tsx
import { useUserProfile } from "@/hooks/store/user";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
// lib
````
````old web/apps/web/core/components/appearance/theme-switcher.tsx
  const { t } = useTranslation();
````
````new web/apps/web/core/components/appearance/theme-switcher.tsx
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
````
````old web/apps/web/core/components/appearance/theme-switcher.tsx
          message: (error) => t(errorMessageKey(error)),
````
````new web/apps/web/core/components/appearance/theme-switcher.tsx
          message: refusalMessage,
````
````old web/apps/web/core/components/appearance/theme-switcher.tsx
    [setTheme, updateUserTheme, t]
````
````new web/apps/web/core/components/appearance/theme-switcher.tsx
    [setTheme, updateUserTheme, t, refusalMessage]
````

`web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx`（修改，2 处）：

````old web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx
            {createdByDetails && <ButtonAvatars showTooltip={false} userIds={createdByDetails?.id} />}
````
````new web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx
            {createdByDetails && <ButtonAvatars userIds={createdByDetails?.id} />}
````
````old web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx
      {createdByDetails && !isActive && <ButtonAvatars showTooltip={false} userIds={createdByDetails?.id} />}
````
````new web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx
      {createdByDetails && !isActive && <ButtonAvatars userIds={createdByDetails?.id} />}
````

`web/apps/web/core/components/dropdowns/member/avatar.tsx`（修改，1 处）：

````old web/apps/web/core/components/dropdowns/member/avatar.tsx
type AvatarProps = {
  showTooltip: boolean;
````
````new web/apps/web/core/components/dropdowns/member/avatar.tsx
type AvatarProps = {
````

`web/apps/web/core/components/dropdowns/member/base.tsx`（修改，1 处）：

````old web/apps/web/core/components/dropdowns/member/base.tsx
      {!hideIcon && <ButtonAvatars showTooltip={showTooltip} userIds={value} icon={icon} />}
````
````new web/apps/web/core/components/dropdowns/member/base.tsx
      {!hideIcon && <ButtonAvatars userIds={value} icon={icon} />}
````

`web/apps/web/core/components/inbox/sidebar/inbox-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/inbox/sidebar/inbox-list-item.tsx
            {createdByDetails && <ButtonAvatars showTooltip={false} userIds={createdByDetails.id} />}
````
````new web/apps/web/core/components/inbox/sidebar/inbox-list-item.tsx
            {createdByDetails && <ButtonAvatars userIds={createdByDetails.id} />}
````

`web/apps/web/core/components/issues/attachment/attachment-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/attachment/attachment-list-item.tsx
                    <ButtonAvatars showTooltip userIds={attachment?.created_by} />
````
````new web/apps/web/core/components/issues/attachment/attachment-list-item.tsx
                    <ButtonAvatars userIds={attachment?.created_by} />
````

`web/apps/web/core/components/issues/issue-detail/sidebar.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/issue-detail/sidebar.tsx
                  <ButtonAvatars showTooltip userIds={createdByDetails.id} />
````
````new web/apps/web/core/components/issues/issue-detail/sidebar.tsx
                  <ButtonAvatars userIds={createdByDetails.id} />
````

`web/apps/web/core/components/issues/peek-overview/properties.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/peek-overview/properties.tsx
            <ButtonAvatars showTooltip userIds={createdByDetails.id} />
````
````new web/apps/web/core/components/issues/peek-overview/properties.tsx
            <ButtonAvatars userIds={createdByDetails.id} />
````

`web/apps/web/core/components/modules/module-card-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/modules/module-card-item.tsx
                  <ButtonAvatars showTooltip={false} userIds={moduleLeadDetails?.id} />
````
````new web/apps/web/core/components/modules/module-card-item.tsx
                  <ButtonAvatars userIds={moduleLeadDetails?.id} />
````

`web/apps/web/core/components/modules/module-list-item-action.tsx`（修改，1 处）：

````old web/apps/web/core/components/modules/module-list-item-action.tsx
          <ButtonAvatars showTooltip={false} userIds={moduleLeadDetails?.id} />
````
````new web/apps/web/core/components/modules/module-list-item-action.tsx
          <ButtonAvatars userIds={moduleLeadDetails?.id} />
````

`web/apps/web/core/components/power-k/config/miscellaneous-commands.ts`（修改，4 处）：

````old web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
import { useTranslation } from "@nerve/i18n";
// nerve imports
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { copyTextToClipboard } from "@nerve/utils";
````
````new web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
import { usePowerK } from "@/hooks/store/use-power-k";
````
````new web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
import { usePowerK } from "@/hooks/store/use-power-k";
import { useCopyLink } from "@/hooks/use-copy-link";
````
````old web/apps/web/core/components/power-k/config/miscellaneous-commands.ts

  const copyCurrentPageUrlToClipboard = useCallback(() => {
    const url = new URL(window.location.href);
    copyTextToClipboard(url.href)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.miscellaneous_actions.copy_current_page_url_toast_success"),
        });
        return;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.miscellaneous_actions.copy_current_page_url_toast_error"),
        });
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
````
````new web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
  const copyLink = useCopyLink();
````
````old web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
      action: copyCurrentPageUrlToClipboard,
````
````new web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
      action: () =>
        copyLink(window.location.href, t("power_k.miscellaneous_actions.copy_current_page_url_toast_success")),
````

`web/apps/web/core/components/settings/profile/content/pages/security.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/profile/content/pages/security.tsx
import { ApiError } from "@/lib/api-error";
````
````new web/apps/web/core/components/settings/profile/content/pages/security.tsx
import { problemCode } from "@/lib/api-error";
````
````old web/apps/web/core/components/settings/profile/content/pages/security.tsx
        if (error instanceof ApiError && error.problem?.code === "identity.current_password_incorrect") {
````
````new web/apps/web/core/components/settings/profile/content/pages/security.tsx
        if (problemCode(error) === "identity.current_password_incorrect") {
````

`web/apps/web/core/components/views/view-list-item-action.tsx`（修改，1 处）：

````old web/apps/web/core/components/views/view-list-item-action.tsx
      <ButtonAvatars showTooltip={false} userIds={ownedByDetails?.id ?? []} />
````
````new web/apps/web/core/components/views/view-list-item-action.tsx
      <ButtonAvatars userIds={ownedByDetails?.id ?? []} />
````

`web/apps/web/core/components/workspace/use-create-workspace.ts`（修改，2 处）：

````old web/apps/web/core/components/workspace/use-create-workspace.ts
import { ApiError } from "@/lib/api-error";
````
````new web/apps/web/core/components/workspace/use-create-workspace.ts
import { problemCode } from "@/lib/api-error";
````
````old web/apps/web/core/components/workspace/use-create-workspace.ts
  if (error instanceof ApiError && error.problem?.code === "workspace.slug_taken")
    return { kind: "fields", fields: { slug: SLUG_MESSAGES.taken } };
````
````new web/apps/web/core/components/workspace/use-create-workspace.ts
  if (problemCode(error) === "workspace.slug_taken") return { kind: "fields", fields: { slug: SLUG_MESSAGES.taken } };
````

`web/apps/web/core/components/workspace/use-invitation-answer.ts`（修改，2 处）：

````old web/apps/web/core/components/workspace/use-invitation-answer.ts
import { ApiError } from "@/lib/api-error";
````
````new web/apps/web/core/components/workspace/use-invitation-answer.ts
import { problemCode } from "@/lib/api-error";
````
````old web/apps/web/core/components/workspace/use-invitation-answer.ts
    if (error instanceof ApiError && error.problem?.code === "workspace.invitation_email_mismatch") {
````
````new web/apps/web/core/components/workspace/use-invitation-answer.ts
    if (problemCode(error) === "workspace.invitation_email_mismatch") {
````

`web/apps/web/core/hooks/use-copy-link.ts`（整个文件，54 行）：

````whole web/apps/web/core/hooks/use-copy-link.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { copyTextToClipboard, copyUrlToClipboard } from "@nerve/utils";

/**
 * Writes a text to the clipboard by write, and says whether it could (M3 design 7.7): the success under the title of
 * titleKey, its message copied, which names what was copied; the browser may not let the page write the clipboard (no
 * permission, or plain http), which the failure says. Every copy of the web app is one of this module's (the keyword
 * rule copy-hook).
 */
function useCopy(
  write: (text: string) => Promise<void>,
  titleKey: string
): (text: string, copied: string) => Promise<void> {
  const { t } = useTranslation();
  return async (text, copied) => {
    try {
      await write(text);
      setToast({ type: TOAST_TYPE.SUCCESS, title: t(titleKey), message: copied });
    } catch {
      // the browser did not let the page write the clipboard
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: t("something_went_wrong_please_try_again"),
      });
    }
  };
}

/** Copies a link, a path of this origin or a whole address, as useCopy does, under "Link copied". */
export function useCopyLink(): (link: string, copied: string) => Promise<void> {
  return useCopy(copyUrlToClipboard, "common.link_copied");
}

/** Copies a text as it is (a token, say), as useCopy does, under "Success". */
export function useCopyText(): (text: string, copied: string) => Promise<void> {
  return useCopy(copyTextToClipboard, "toast.success");
}

/** Copies the link of a project of the address's workspace, its work items' page, as useCopyLink. */
export function useCopyProjectLink(): (projectId: string) => Promise<void> {
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  return (projectId) =>
    copyLink(`/${workspaceSlug}/projects/${projectId}/issues`, t("project_link_copied_to_clipboard"));
}
````

`web/apps/web/core/hooks/use-refusal-toast.ts`（修改，2 处）：

````old web/apps/web/core/hooks/use-refusal-toast.ts
 * Says in a toast why a change failed (M2 design 7.3): nerve's refusal by its problem's code, or why nerve could not be
 * reached (errorMessageKey), under the title of titleKey. Every setToast that says a failure by errorMessageKey is this
 * one (the keyword rule refusal-toast); the theme switcher's promise toast says its failure itself (setPromiseToast,
 * P9 spec §5). A page that shows some refusals under its fields decides that itself, and calls this for the others.
````
````new web/apps/web/core/hooks/use-refusal-toast.ts
 * Why a change failed (M2 design 7.3), as a toast says it: nerve's refusal by its problem's code, or why nerve could not
 * be reached (errorMessageKey), in the page's language. The message of every toast that says such a failure, a
 * promise toast's (setPromiseToast) too (the keyword rule refusal-toast).
 */
export function useRefusalMessage(): (error: unknown) => string {
  const { t } = useTranslation();
  return (error) => t(errorMessageKey(error));
}

/**
 * Says in a toast why a change failed, useRefusalMessage's, under the title of titleKey. Every setToast that says such
 * a failure is this one (the keyword rule refusal-toast). A page that shows some refusals under its fields decides that
 * itself, and calls this for the others.
````
````old web/apps/web/core/hooks/use-refusal-toast.ts
  const { t } = useTranslation();
  return (error) => {
    setToast({ type: TOAST_TYPE.ERROR, title: t(titleKey), message: t(errorMessageKey(error)) });
````
````new web/apps/web/core/hooks/use-refusal-toast.ts
  const { t } = useTranslation();
  const refusalMessage = useRefusalMessage();
  return (error) => {
    setToast({ type: TOAST_TYPE.ERROR, title: t(titleKey), message: refusalMessage(error) });
````

`web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`（修改，2 处）：

````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
import { ApiError } from "@/lib/api-error";
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
import { problemCode } from "@/lib/api-error";
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
    const notFound = read.error instanceof ApiError && read.error.problem?.code === "project.not_found";
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
    const notFound = problemCode(read.error) === "project.not_found";
````

`web/apps/web/core/lib/api-error.ts`（修改，1 处）：

````old web/apps/web/core/lib/api-error.ts
}

/** The data of an answer of the generated client, or the answer as an ApiError when it is not a success. */
````
````new web/apps/web/core/lib/api-error.ts
}

/** nerve's problem code of an error of a call to nerve ("workspace.slug_taken"), or undefined: no answer, no problem. */
export function problemCode(error: unknown): string | undefined {
  return error instanceof ApiError ? error.problem?.code : undefined;
}

/** The data of an answer of the generated client, or the answer as an ApiError when it is not a success. */
````

`web/apps/web/core/lib/form-refusal.ts`（修改，2 处）：

````old web/apps/web/core/lib/form-refusal.ts
import { ApiError } from "@/lib/api-error";
````
````new web/apps/web/core/lib/form-refusal.ts
import { problemCode } from "@/lib/api-error";
````
````old web/apps/web/core/lib/form-refusal.ts
  const code = error instanceof ApiError ? (error.problem?.code ?? "") : "";
  const field = taken[code];
````
````new web/apps/web/core/lib/form-refusal.ts
  const field = taken[problemCode(error) ?? ""];
````

`web/packages/constants/src/filter.ts`（修改，1 处）：

````old web/packages/constants/src/filter.ts
    name: "1 month ago",
    i18n_name: "date_filters.1_month_ago",
````
````new web/packages/constants/src/filter.ts
    name: "1 month ago",
````

`web/packages/i18n/src/locales/en/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/power-k.json
      "copy_current_page_url_toast_success": "Current page URL copied to clipboard.",
      "copy_current_page_url_toast_error": "Some error occurred while copying the current page URL to clipboard.",
````
````new web/packages/i18n/src/locales/en/power-k.json
      "copy_current_page_url_toast_success": "Current page URL copied to clipboard.",
````

`web/packages/i18n/src/locales/en/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace-settings.json
    "token_copied": "Token copied to clipboard.",
    "token_not_copied": "The token could not be copied to the clipboard.",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
    "token_copied": "Token copied to clipboard.",
````

`web/packages/i18n/src/locales/zh-CN/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/power-k.json
      "copy_current_page_url_toast_success": "当前页面 URL 已复制到剪贴板。",
      "copy_current_page_url_toast_error": "复制当前页面 URL 到剪贴板时出错。",
````
````new web/packages/i18n/src/locales/zh-CN/power-k.json
      "copy_current_page_url_toast_success": "当前页面 URL 已复制到剪贴板。",
````

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
    "token_copied": "令牌已复制到剪贴板。",
    "token_not_copied": "无法将令牌复制到剪贴板。",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
    "token_copied": "令牌已复制到剪贴板。",
````

- [ ] **Step 2: 测试**

`e2e/stories/identity/a11-api-tokens.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a11-api-tokens.spec.ts
test("A11 (page): a token created on the page shows once, and nowhere after; the list shows it without it, and revokes it", async ({
````
````new e2e/stories/identity/a11-api-tokens.spec.ts
test("A11 (page): a token created on the page shows once, a click on it copies it and says so, and nowhere after; the list shows it without it, and revokes it", async ({
````
````old e2e/stories/identity/a11-api-tokens.spec.ts
  await expect(page.getByText(created.token)).toBeVisible();
````
````new e2e/stories/identity/a11-api-tokens.spec.ts
  await expect(page.getByText(created.token)).toBeVisible();
  // A click on it copies it as it is, and the page says so.
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.getByText(created.token).click();
  await expect(page.getByText("Token copied to clipboard.")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(created.token);
````

`e2e/stories/smoke/s6-dialogs.spec.ts`（修改，2 处）：

````old e2e/stories/smoke/s6-dialogs.spec.ts
test("S6 (page): the command palette and its list of keyboard shortcuts are modal dialogs: while each is open the page behind it is inert and hidden and Tab stays in it; Escape closes each and gives the page back", async ({
````
````new e2e/stories/smoke/s6-dialogs.spec.ts
test("S6 (page): the command palette and its list of keyboard shortcuts are modal dialogs: while each is open the page behind it is inert and hidden and Tab stays in it; Escape closes each and gives the page back; the palette's command that copies the page's address copies it and says so", async ({
````
````old e2e/stories/smoke/s6-dialogs.spec.ts
  expect(shortcutStops.slice(1)).toEqual(["Close", "Search for shortcuts"]);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
````
````new e2e/stories/smoke/s6-dialogs.spec.ts
  expect(shortcutStops.slice(1)).toEqual(["Close", "Search for shortcuts"]);
  expect(await closedByEscape(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(false);

  // The palette's command that copies the page's address.
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.keyboard.press("ControlOrMeta+k");
  await page.keyboard.type("Copy current page URL");
  await page.keyboard.press("Enter");
  await expect(page.getByText("Current page URL copied to clipboard.")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(page.url());
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
````

`web/apps/web/core/hooks/use-copy-link.test.ts`（修改，4 处）：

````old web/apps/web/core/hooks/use-copy-link.test.ts
import { useCopyLink, useCopyProjectLink } from "./use-copy-link";
````
````new web/apps/web/core/hooks/use-copy-link.test.ts
import { useCopyLink, useCopyProjectLink, useCopyText } from "./use-copy-link";
````
````old web/apps/web/core/hooks/use-copy-link.test.ts
const browser = vi.hoisted(() => ({ copyUrlToClipboard: vi.fn() }));
vi.mock("@nerve/utils", () => ({ copyUrlToClipboard: browser.copyUrlToClipboard }));
````
````new web/apps/web/core/hooks/use-copy-link.test.ts
const browser = vi.hoisted(() => ({ copyTextToClipboard: vi.fn(), copyUrlToClipboard: vi.fn() }));
vi.mock("@nerve/utils", () => ({
  copyTextToClipboard: browser.copyTextToClipboard,
  copyUrlToClipboard: browser.copyUrlToClipboard,
}));
````
````old web/apps/web/core/hooks/use-copy-link.test.ts
beforeEach(() => {
````
````new web/apps/web/core/hooks/use-copy-link.test.ts
beforeEach(() => {
  browser.copyTextToClipboard.mockReset();
  browser.copyTextToClipboard.mockResolvedValue(undefined);
````
````old web/apps/web/core/hooks/use-copy-link.test.ts
});

describe("useCopyProjectLink", () => {
````
````new web/apps/web/core/hooks/use-copy-link.test.ts
});

describe("useCopyText", () => {
  it("writes the text to the clipboard as it is, not as a link, and says what was copied", async () => {
    await useCopyText()("nerve_pat_1", "the copied");
    expect([browser.copyTextToClipboard.mock.calls, browser.copyUrlToClipboard.mock.calls]).toEqual([
      [["nerve_pat_1"]],
      [],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "the copied" }]);
  });

  it("says it could not when the browser does not let the page write the clipboard, and does not reject", async () => {
    browser.copyTextToClipboard.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
    await expect(useCopyText()("nerve_pat_1", "the copied")).resolves.toBeUndefined();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
  });
});

describe("useCopyProjectLink", () => {
````

`web/apps/web/core/hooks/use-refusal-toast.test.ts`（修改，2 处）：

````old web/apps/web/core/hooks/use-refusal-toast.test.ts
import { useRefusalToast } from "./use-refusal-toast";
````
````new web/apps/web/core/hooks/use-refusal-toast.test.ts
import { useRefusalMessage, useRefusalToast } from "./use-refusal-toast";
````
````old web/apps/web/core/hooks/use-refusal-toast.test.ts
  toasts.length = 0;
````
````new web/apps/web/core/hooks/use-refusal-toast.test.ts
  toasts.length = 0;
});

describe("useRefusalMessage", () => {
  it("is nerve's reason for a refusal, by its problem's code", () => {
    expect(useRefusalMessage()(refusal(409, "workspace.slug_taken"))).toBe("errors.workspace_slug_taken");
  });
````

`web/apps/web/core/lib/api-error.test.ts`（修改，2 处）：

````old web/apps/web/core/lib/api-error.test.ts
import { ApiError, unwrap } from "./api-error";
````
````new web/apps/web/core/lib/api-error.test.ts
import { ApiError, problemCode, unwrap } from "./api-error";
````
````old web/apps/web/core/lib/api-error.test.ts
const problem = { status: 409, code: "identity.email_taken", title: "Conflict", detail: "The email is taken." };
````
````new web/apps/web/core/lib/api-error.test.ts
const problem = { status: 409, code: "identity.email_taken", title: "Conflict", detail: "The email is taken." };

describe("problemCode", () => {
  it("is the code of nerve's problem, and none for an answer without a problem or an error of no answer", () => {
    expect([
      problemCode(new ApiError(409, problem)),
      problemCode(new ApiError(502, undefined)),
      problemCode(new SessionChangedError()),
      problemCode(new TypeError("Failed to fetch")),
    ]).toEqual(["identity.email_taken", undefined, undefined, undefined]);
  });
});
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `70 rules, 23 exceptions, no hits.`；web 的 oxlint 300 条、`@nerve/ui` 14 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 116 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add Makefile docs/v0/M1-frontend-trim/M1-design.md docs/v0/M3-workspace-project/M3-design.md e2e/stories/identity/a11-api-tokens.spec.ts e2e/stories/smoke/s6-dialogs.spec.ts knip.jsonc tools/i18n-keys.mjs tools/keywords.json tools/keywords.mjs web/apps/web/core/components/api-token/modal/generated-token-details.tsx web/apps/web/core/components/appearance/theme-switcher.tsx web/apps/web/core/components/cycles/list/cycle-list-item-action.tsx web/apps/web/core/components/dropdowns/member/avatar.tsx web/apps/web/core/components/dropdowns/member/base.tsx web/apps/web/core/components/inbox/sidebar/inbox-list-item.tsx web/apps/web/core/components/issues/attachment/attachment-list-item.tsx web/apps/web/core/components/issues/issue-detail/sidebar.tsx web/apps/web/core/components/issues/peek-overview/properties.tsx web/apps/web/core/components/modules/module-card-item.tsx web/apps/web/core/components/modules/module-list-item-action.tsx web/apps/web/core/components/power-k/config/miscellaneous-commands.ts web/apps/web/core/components/settings/profile/content/pages/security.tsx web/apps/web/core/components/views/view-list-item-action.tsx web/apps/web/core/components/workspace/use-create-workspace.ts web/apps/web/core/components/workspace/use-invitation-answer.ts web/apps/web/core/hooks/use-copy-link.test.ts web/apps/web/core/hooks/use-copy-link.ts web/apps/web/core/hooks/use-refusal-toast.test.ts web/apps/web/core/hooks/use-refusal-toast.ts web/apps/web/core/layouts/auth-layout/use-project-fetch.ts web/apps/web/core/lib/api-error.test.ts web/apps/web/core/lib/api-error.ts web/apps/web/core/lib/form-refusal.ts web/packages/constants/src/filter.ts web/packages/i18n/src/locales/en/power-k.json web/packages/i18n/src/locales/en/workspace-settings.json web/packages/i18n/src/locales/zh-CN/power-k.json web/packages/i18n/src/locales/zh-CN/workspace-settings.json
```
```bash
git commit -m "refactor(M3/closeout): one way to copy, to read a problem's code and to say a refusal; every key a page reads is defined

ButtonAvatars' unread showTooltip goes from its ten callers. Copying goes
through useCopyLink and the new useCopyText, the token dialog's and the
palette's too, and the copy-hook rule keeps it there, with dated
exceptions for M4-M8's files. problemCode() reads a problem's code;
useRefusalMessage gives a refusal's text, the theme switcher's too.
tools/i18n-keys.mjs, in make lint-web, fails on a key a page reads that
no language defines. The guard's header and the M1 design say when an
exception until an M's closeout falls due.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | `useCopyText` 把文字当链接写（K5） | `use-copy-link.test.ts`、故事 A11 | vitest；端到端 |
| `T9.2` | `useCopy` 复制失败时什么也不说（K5） | `use-copy-link.test.ts` | vitest |
| `T9.3` | 令牌弹窗复制的不是令牌（K5） | 故事 A11 | 端到端 |
| `T9.4` | 命令面板的复制命令什么也不复制（K5） | 故事 S6 | 端到端 |
| `T9.5` | M3 的页面又自己复制（`copy-hook`） | 关键词守卫 | 静态 |
| `T9.6` | `problemCode` 给不出问题码（K6） | `api-error.test.ts` | vitest |
| `T9.7` | `problemCode` 给没有回答的错误一个问题码（K6） | `api-error.test.ts` | vitest |
| `T9.8` | `useRefusalMessage` 不管什么拒绝都给通常的原因（K7） | `use-refusal-toast.test.ts` | vitest |
| `T9.9` | 主题切换又自己说它的失败（`refusal-toast`，K7） | 关键词守卫 | 静态 |
| `T9.10` | 页面从用户 store 取 `signOut`（`sign-out-toast`，K8） | 关键词守卫 | 静态 |
| `T9.11` | 侧边栏读的一个键在两种语言里都改了名，两边仍一致（K13） | `tools/i18n-keys.mjs` | 静态 |

---

### Task 10: 组字的判断只有一个（`keyCode` 229）；选择上按 Enter 打开列表

**Files:**
- Create: `web/packages/utils/src/keyboard.test.ts`、`web/packages/utils/src/keyboard.ts`
- Modify: `e2e/fixtures/dialogs.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`e2e/stories/smoke/s6-dialogs.spec.ts`、`pnpm-lock.yaml`、`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`、`web/apps/web/core/components/issues/select/base.tsx`、`web/apps/web/core/components/power-k/ui/modal/wrapper.tsx`、`web/apps/web/core/hooks/use-dropdown-key-down.tsx`、`web/apps/web/core/hooks/use-keypress.tsx`、`web/packages/propel/package.json`、`web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx`、`web/packages/ui/package.json`、`web/packages/ui/src/dropdowns/custom-search-select.tsx`、`web/packages/ui/src/dropdowns/custom-select.tsx`、`web/packages/ui/src/hooks/use-dropdown-key-down.tsx`、`web/packages/ui/src/modals/modal-core.tsx`、`web/packages/utils/src/index.ts`

**Interfaces:**（spec 2.11）
- `@nerve/utils` 的 `keyboard.ts`（新文件）：`isComposing(event)`：`event.isComposing` 或 `event.keyCode === 229`；`@nerve/ui` 的 `ModalCore`、`CustomSearchSelect`、`use-dropdown-key-down`，`@nerve/propel` 的表情选择器，web 的 `use-keypress`、`use-dropdown-key-down`、标签的选择、`issues/select/base.tsx`、命令面板都经它；命令面板的 `handleKeyDown` 开头加同一个守卫。`@nerve/propel` 的 `package.json` 依赖 `@nerve/utils`。
- `@nerve/ui` 的 `use-dropdown-key-down`：`selectActiveItem?.()` 和依赖，两条警告没有了（上限 14 → 12）。
- `CustomSelect`：模块级的 `openOnEnter`：Enter 上先 `preventDefault`，再 `currentTarget.click()`；两种按钮都用它。

**Tests:** `keyboard.test.ts`（新文件）。`dialogs.ts` 的 `composingEscape` 两种顺序都发（Chromium、Firefox 的 `isComposing`；Safari 的 `compositionend` 之后 `keyCode` 229 的 keydown）；S6：命令面板的空搜索框中结束组字的 Escape 不关面板；P5：角色的选择上按 Enter 打开列表。

- [ ] **Step 1: 判断和守卫**

`pnpm-lock.yaml`（修改，1 处）：

````old pnpm-lock.yaml
        specifier: workspace:*
        version: link:../types
      class-variance-authority:
````
````new pnpm-lock.yaml
        specifier: workspace:*
        version: link:../types
      '@nerve/utils':
        specifier: workspace:*
        version: link:../utils
      class-variance-authority:
````

`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
import { getTabIndex } from "@nerve/utils";
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
import { getTabIndex, isComposing } from "@nerve/utils";
````
````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
    if (query !== "" && e.key === "Enter" && !e.nativeEvent.isComposing && canCreateLabel) {
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
    if (query !== "" && e.key === "Enter" && !isComposing(e.nativeEvent) && canCreateLabel) {
````

`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
import { sortBySelectedFirst } from "@nerve/utils";
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
import { isComposing, sortBySelectedFirst } from "@nerve/utils";
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    if (query !== "" && e.key === "Enter" && !e.nativeEvent.isComposing && canCreateLabel) {
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    if (query !== "" && e.key === "Enter" && !isComposing(e.nativeEvent) && canCreateLabel) {
````

`web/apps/web/core/components/issues/select/base.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/select/base.tsx
import { cn } from "@nerve/utils";
````
````new web/apps/web/core/components/issues/select/base.tsx
import { cn, isComposing } from "@nerve/utils";
````
````old web/apps/web/core/components/issues/select/base.tsx
      !e.nativeEvent.isComposing &&
````
````new web/apps/web/core/components/issues/select/base.tsx
      !isComposing(e.nativeEvent) &&
````

`web/apps/web/core/components/power-k/ui/modal/wrapper.tsx`（修改，2 处）：

````old web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````new web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { isComposing } from "@nerve/utils";
````
````old web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
    (e: React.KeyboardEvent) => {
````
````new web/apps/web/core/components/power-k/ui/modal/wrapper.tsx
    (e: React.KeyboardEvent) => {
      // a key an input method takes is the method's: an Escape that ends a composition in the search neither clears it
      // nor closes the palette, a Backspace in one leaves the page where it is
      if (isComposing(e.nativeEvent)) return;
````

`web/apps/web/core/hooks/use-dropdown-key-down.tsx`（修改，2 处）：

````old web/apps/web/core/hooks/use-dropdown-key-down.tsx
import { useCallback } from "react";
````
````new web/apps/web/core/hooks/use-dropdown-key-down.tsx
import { useCallback } from "react";
import { isComposing } from "@nerve/utils";
````
````old web/apps/web/core/hooks/use-dropdown-key-down.tsx
      if (event.key === "Enter" && !event.nativeEvent.isComposing) {
````
````new web/apps/web/core/hooks/use-dropdown-key-down.tsx
      if (event.key === "Enter" && !isComposing(event.nativeEvent)) {
````

`web/apps/web/core/hooks/use-keypress.tsx`（修改，2 处）：

````old web/apps/web/core/hooks/use-keypress.tsx
import { useEffect, useRef } from "react";
````
````new web/apps/web/core/hooks/use-keypress.tsx
import { useEffect, useRef } from "react";
import { isComposing } from "@nerve/utils";
````
````old web/apps/web/core/hooks/use-keypress.tsx
      // a key pressed while an IME composes (an Escape that ends the composition) is the composition's, not the page's
      if (event.isComposing) return;
````
````new web/apps/web/core/hooks/use-keypress.tsx
      // a key an input method takes (an Escape that ends a composition) is the method's, not the page's
      if (isComposing(event)) return;
````

`web/packages/propel/package.json`（修改，1 处）：

````old web/packages/propel/package.json
    "@nerve/types": "workspace:*",
````
````new web/packages/propel/package.json
    "@nerve/types": "workspace:*",
    "@nerve/utils": "workspace:*",
````

`web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx`（修改，2 处）：

````old web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
import { Tabs } from "@base-ui-components/react";
````
````new web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
import { Tabs } from "@base-ui-components/react";
import { isComposing } from "@nerve/utils";
````
````old web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
            if (e.nativeEvent.isComposing) return;
````
````new web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
            if (isComposing(e.nativeEvent)) return;
````

`web/packages/ui/package.json`（修改，1 处）：

````old web/packages/ui/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 14",
````
````new web/packages/ui/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 12",
````

`web/packages/ui/src/dropdowns/custom-search-select.tsx`（修改，2 处）：

````old web/packages/ui/src/dropdowns/custom-search-select.tsx
import { Tooltip } from "@nerve/propel/tooltip";
````
````new web/packages/ui/src/dropdowns/custom-search-select.tsx
import { Tooltip } from "@nerve/propel/tooltip";
import { isComposing } from "@nerve/utils";
````
````old web/packages/ui/src/dropdowns/custom-search-select.tsx
                if (event.nativeEvent.isComposing) return;
````
````new web/packages/ui/src/dropdowns/custom-search-select.tsx
                if (isComposing(event.nativeEvent)) return;
````

`web/packages/ui/src/dropdowns/custom-select.tsx`（修改，4 处）：

````old web/packages/ui/src/dropdowns/custom-select.tsx
import type { ICustomSelectItemProps, ICustomSelectProps } from "./helper";
````
````new web/packages/ui/src/dropdowns/custom-select.tsx
import type { ICustomSelectItemProps, ICustomSelectProps } from "./helper";

// Enter on the button opens the list, as a click does. Headless UI's button would submit the form around it, as a
// native select's Enter does, and a keyboard user would send a form (a member's role, say) unawares; prevented here,
// before Headless UI's handler, the Enter reaches none of Headless UI's.
function openOnEnter(event: React.KeyboardEvent<HTMLButtonElement>) {
  if (event.key !== "Enter") return;
  event.preventDefault();
  event.currentTarget.click();
}
````
````old web/packages/ui/src/dropdowns/custom-select.tsx
    // A select, so Headless UI's Listbox, whose open state is the only one. Its button opens the list (a click,
    // Space, the arrows; Enter submits an enclosing form, as a native select's does), and the list takes the focus:
    // the arrows move, Enter or Space picks, typing jumps to a name. A pick, Escape, Tab or a click outside closes it.
````
````new web/packages/ui/src/dropdowns/custom-select.tsx
    // A select, so Headless UI's Listbox, whose open state is the only one. Its button opens the list (a click, Enter,
    // Space, the arrows), and the list takes the focus: the arrows move, Enter or Space picks, typing jumps to a name.
    // A pick, Escape, Tab or a click outside closes it.
````
````old web/packages/ui/src/dropdowns/custom-select.tsx
                type="button"
                tabIndex={tabIndex}
                className={`flex items-center justify-between gap-1 rounded text-11 ${
````
````new web/packages/ui/src/dropdowns/custom-select.tsx
                type="button"
                tabIndex={tabIndex}
                onKeyDown={openOnEnter}
                className={`flex items-center justify-between gap-1 rounded text-11 ${
````
````old web/packages/ui/src/dropdowns/custom-select.tsx
                type="button"
                tabIndex={tabIndex}
                className={cn(
````
````new web/packages/ui/src/dropdowns/custom-select.tsx
                type="button"
                tabIndex={tabIndex}
                onKeyDown={openOnEnter}
                className={cn(
````

`web/packages/ui/src/hooks/use-dropdown-key-down.tsx`（修改，4 处）：

````old web/packages/ui/src/hooks/use-dropdown-key-down.tsx
import { useCallback } from "react";
````
````new web/packages/ui/src/hooks/use-dropdown-key-down.tsx
import { useCallback } from "react";
import { isComposing } from "@nerve/utils";
````
````old web/packages/ui/src/hooks/use-dropdown-key-down.tsx
      if (event.key === "Enter" && !event.nativeEvent.isComposing) {
````
````new web/packages/ui/src/hooks/use-dropdown-key-down.tsx
      if (event.key === "Enter" && !isComposing(event.nativeEvent)) {
````
````old web/packages/ui/src/hooks/use-dropdown-key-down.tsx
          selectActiveItem && selectActiveItem();
````
````new web/packages/ui/src/hooks/use-dropdown-key-down.tsx
          selectActiveItem?.();
````
````old web/packages/ui/src/hooks/use-dropdown-key-down.tsx
    [isOpen, onOpen, onClose]
````
````new web/packages/ui/src/hooks/use-dropdown-key-down.tsx
    [isOpen, onOpen, onClose, selectActiveItem]
````

`web/packages/ui/src/modals/modal-core.tsx`（修改，3 处）：

````old web/packages/ui/src/modals/modal-core.tsx
import React, { useEffect, useState } from "react";
````
````new web/packages/ui/src/modals/modal-core.tsx
import React, { useEffect, useState } from "react";
import { isComposing } from "@nerve/utils";
````
````old web/packages/ui/src/modals/modal-core.tsx
            // an Escape that ends an IME composition is the composition's: Headless UI's Dialog closes on a keydown
            // of Escape on the window unless its default is prevented (useEscape), so this prevents it on the way
            // out, once the controls inside have had it (in the capture phase, it would make Headless UI skip their
            // own handlers, which it does for a prevented event: a select's, which keeps its Escape from the page)
````
````new web/packages/ui/src/modals/modal-core.tsx
            // an Escape that ends an input method's composition is the method's (isComposing):
            // Headless UI's Dialog closes on a keydown of Escape on the window unless its default is prevented
            // (useEscape), so this prevents it on the way out, once the controls inside have had it (in the capture
            // phase, it would make Headless UI skip their own handlers, which it does for a prevented event: a
            // select's, which keeps its Escape from the page)
````
````old web/packages/ui/src/modals/modal-core.tsx
              if (event.key === "Escape" && event.nativeEvent.isComposing) event.preventDefault();
````
````new web/packages/ui/src/modals/modal-core.tsx
              if (event.key === "Escape" && isComposing(event.nativeEvent)) event.preventDefault();
````

`web/packages/utils/src/index.ts`（修改，1 处）：

````old web/packages/utils/src/index.ts
export * from "./intake";
````
````new web/packages/utils/src/index.ts
export * from "./intake";
export * from "./keyboard";
````

`web/packages/utils/src/keyboard.ts`（新文件，14 行）：

````file web/packages/utils/src/keyboard.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * Whether a keydown is an input method's (Chinese, say), not the page's: one while the method composes (isComposing),
 * which Chromium and Firefox give, or the one that ends the composition, which Safari gives after the composition's
 * end, isComposing false and keyCode 229, the code every browser gives a key the method takes. Takes the DOM's
 * KeyboardEvent, a React keyboard event's nativeEvent.
 */
export function isComposing(event: Pick<KeyboardEvent, "isComposing" | "keyCode">): boolean {
  return event.isComposing || event.keyCode === 229;
}
````

- [ ] **Step 2: 安装**

Run: `pnpm install --frozen-lockfile`
Expected: 通过；`@nerve/propel` 链接到 `@nerve/utils`，锁文件不变。

- [ ] **Step 3: 测试**

`e2e/fixtures/dialogs.ts`（修改，2 处）：

````old e2e/fixtures/dialogs.ts
 * Dispatches on field, at once, what the browser gives the page when an Escape ends an input method's composition
 * (Chinese, say): the composition's start, the Escape's keydown while composing, and the composition's end. A dropdown
 * that follows the composition by its events (Base UI's popover) sees one, as a page does with a real input method.
````
````new e2e/fixtures/dialogs.ts
 * Dispatches on field, at once, what the browsers give the page when an Escape ends an input method's composition
 * (Chinese, say), in both their orders: Chromium's and Firefox's, the Escape's keydown while composing (isComposing),
 * between the composition's start and end; then Safari's, the keydown after the composition's end, isComposing false
 * and keyCode 229. A dropdown that follows the composition by its events (Base UI's popover) sees each, as a page does
 * with a real input method.
````
````old e2e/fixtures/dialogs.ts
    element.dispatchEvent(new CompositionEvent("compositionend", init));
````
````new e2e/fixtures/dialogs.ts
    element.dispatchEvent(new CompositionEvent("compositionend", init));
    element.dispatchEvent(new CompositionEvent("compositionstart", init));
    element.dispatchEvent(new CompositionEvent("compositionend", init));
    element.dispatchEvent(new KeyboardEvent("keydown", { ...init, key: "Escape", keyCode: 229 }));
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，2 处）：

````old e2e/stories/project/p5-project-members.spec.ts
test("P5 (page): a project admin who is no workspace admin adds a member and a guest from the workspace's members who are not the project's, the modal held until nerve answers; he is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
````
````new e2e/stories/project/p5-project-members.spec.ts
test("P5 (page): a project admin who is no workspace admin adds a member and a guest from the workspace's members who are not the project's, Enter on a select opening its list, the modal held until nerve answers; he is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
````
````old e2e/stories/project/p5-project-members.spec.ts
  await page.getByRole("button", { name: "Guest", exact: true }).click();
````
````new e2e/stories/project/p5-project-members.spec.ts
  // the role select by the keyboard: Enter opens its list, and sends no form
  await page.getByRole("button", { name: "Guest", exact: true }).focus();
  await page.keyboard.press("Enter");
````

`e2e/stories/smoke/s6-dialogs.spec.ts`（修改，4 处）：

````old e2e/stories/smoke/s6-dialogs.spec.ts
import { closedByEscape, closedWithin, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
````
````new e2e/stories/smoke/s6-dialogs.spec.ts
import {
  closedByComposingEscape,
  closedByEscape,
  closedWithin,
  modalDialog,
  pageBehindHidden,
} from "../../fixtures/dialogs";
````
````old e2e/stories/smoke/s6-dialogs.spec.ts
test("S6 (page): the command palette and its list of keyboard shortcuts are modal dialogs: while each is open the page behind it is inert and hidden and Tab stays in it; Escape closes each and gives the page back; the palette's command that copies the page's address copies it and says so", async ({
````
````new e2e/stories/smoke/s6-dialogs.spec.ts
test("S6 (page): the command palette and its list of keyboard shortcuts are modal dialogs, the palette's search keeping an Escape that ends a composition: while each is open the page behind it is inert and hidden and Tab stays in it; Escape closes each and gives the page back; the palette's command that copies the page's address copies it and says so", async ({
````
````old e2e/stories/smoke/s6-dialogs.spec.ts
  // The palette: its search has the focus, and Tab goes round what it holds.
````
````new e2e/stories/smoke/s6-dialogs.spec.ts
  // The palette: its search has the focus, and Tab goes round what it holds. An Escape that ends an input method's
  // composition in the search is the method's: the palette, which an Escape in its empty search closes, stays open.
````
````old e2e/stories/smoke/s6-dialogs.spec.ts
  await expect(modalDialog(page).getByRole("combobox")).toBeFocused();
````
````new e2e/stories/smoke/s6-dialogs.spec.ts
  await expect(modalDialog(page).getByRole("combobox")).toBeFocused();
  expect(await closedByComposingEscape(modalDialog(page).getByRole("combobox"))).toBe(false);
````

`web/packages/utils/src/keyboard.test.ts`（新文件，25 行）：

````file web/packages/utils/src/keyboard.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { isComposing } from "./keyboard";

// A keydown an input method takes is the method's: the browsers give it in two ways.

describe("isComposing", () => {
  it.each([
    { browser: "Chromium's and Firefox's, while the method composes", event: { isComposing: true, keyCode: 27 } },
    { browser: "Safari's, after the composition's end", event: { isComposing: false, keyCode: 229 } },
  ])("takes $browser for the method's", ({ event }) => {
    expect(isComposing(event)).toBe(true);
  });

  it("leaves the page any other keydown", () => {
    expect([
      isComposing({ isComposing: false, keyCode: 27 }),
      isComposing({ isComposing: false, keyCode: 13 }),
    ]).toEqual([false, false]);
  });
});
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `70 rules, 23 exceptions, no hits.`；web 的 oxlint 300 条、`@nerve/ui` 12 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 116 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add e2e/fixtures/dialogs.ts e2e/stories/project/p5-project-members.spec.ts e2e/stories/smoke/s6-dialogs.spec.ts pnpm-lock.yaml web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx web/apps/web/core/components/issues/select/base.tsx web/apps/web/core/components/power-k/ui/modal/wrapper.tsx web/apps/web/core/hooks/use-dropdown-key-down.tsx web/apps/web/core/hooks/use-keypress.tsx web/packages/propel/package.json web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx web/packages/ui/package.json web/packages/ui/src/dropdowns/custom-search-select.tsx web/packages/ui/src/dropdowns/custom-select.tsx web/packages/ui/src/hooks/use-dropdown-key-down.tsx web/packages/ui/src/modals/modal-core.tsx web/packages/utils/src/index.ts web/packages/utils/src/keyboard.test.ts web/packages/utils/src/keyboard.ts
```
```bash
git commit -m "feat(M3/closeout): one test for a key that belongs to an input method, and Enter on a select opens its list

isComposing(), in @nerve/utils, takes a keydown with isComposing or keyCode
229, the order Safari gives once a composition ends; every guard of
@nerve/ui, @nerve/propel and the web goes through it, and the palette's
empty search no longer closes on the Escape that ends a composition.
Enter on a CustomSelect's button opens its list instead of submitting the
form around it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | `isComposing` 认不出 Safari 在组字结束之后交出的 keydown（K10） | `keyboard.test.ts`、故事 P1 | vitest；端到端 |
| `T10.2` | `isComposing` 认不出组字中的 keydown（K10） | `keyboard.test.ts`、故事 P1 | vitest；端到端 |
| `T10.3` | 命令面板把结束组字的 Escape 当成自己的（K10） | 故事 S6 | 端到端 |
| `T10.4` | `ModalCore` 在 Safari 结束组字的 Escape 上关上（K10） | 故事 P4 | 端到端 |
| `T10.5` | `CustomSearchSelect` 的列表在 Safari 结束组字的 Escape 上关上（K10） | 故事 P1 | 端到端 |
| `T10.6` | 表情选择器在 Safari 结束组字的 Escape 上关上（K10） | 故事 P1 | 端到端 |
| `T10.7` | `useKeypress` 把 Safari 结束组字的 Escape 当成页面的（K10） | 故事 P1 | 端到端 |
| `T10.8` | `CustomSelect` 上按 Enter 又提交外面的表单（K11） | 故事 P5 | 端到端 |
| `T10.9` | `CustomSelect` 上按 Enter 什么也不做（K11） | 故事 P5 | 端到端 |

---

### Task 11: 只读工作区的页面经路由的地址；`workspace-address-route`

**Files:**
- Create: `web/apps/web/core/hooks/use-route-workspace.test.ts`、`web/apps/web/core/hooks/use-route-workspace.ts`
- Modify: `docs/v0/M3-workspace-project/M3-design.md`、`tools/keywords.json`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx`、`web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx`、`web/apps/web/core/components/profile/sidebar.tsx`、`web/apps/web/core/components/project/card-list.tsx`、`web/apps/web/core/components/project/header.tsx`、`web/apps/web/core/components/project/root.tsx`、`web/apps/web/core/components/settings/workspace/sidebar/header.tsx`、`web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/core/components/workspace/sidebar/quick-actions.tsx`、`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`、`web/apps/web/package.json`

**Interfaces:**（spec 2.12）
- `core/hooks/use-route-workspace.ts`（新文件）：`useRouteWorkspace()`：`useParams()` 的 `workspaceSlug` 经 `getWorkspaceBySlug`，没有时 `null`。
- 19 个文件改用它，或在工作区级的 `allowPermissions` 写明 slug（spec 2.12 列出）。
- `tools/keywords.json` 的 `workspace-address-route`：这 19 个文件中没有 `currentWorkspace`，没有两个参数的 `WORKSPACE` 级 `allowPermissions`；M3 设计 7.10 的规则表加上它。
- `project/root.tsx`：`handleClearAllFilters` 和设显示筛选的副作用写全依赖（`isArchived`、`updateDisplayFilters`、`workspaceSlug`）。`updateDisplayFilters` 是 store 的动作，引用不变；列表和归档各自挂载这个根，一次挂载里 `isArchived` 不变，副作用照旧在挂载和换工作区时运行（spec 第 3 节第 2 条）。web 的上限 300 → 298。

**Tests:** `use-route-workspace.test.ts`（新文件）；关键词守卫看住 19 个文件。没有端到端的改动：页面显示的照旧，S2、S7、W 系列的故事照旧通过；P4 和 S7 走过归档的往返。

- [ ] **Step 1: 地址**

`docs/v0/M3-workspace-project/M3-design.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/M3-design.md
| `project-address-route`（P10） | P10、P11 改过的项目页从 router store 读项目和工作区（`currentProjectDetails`、`projectMemberIds`、`currentWorkspace`、不传 slug 和项目的 `allowPermissions`）；M4、M6、M7 改它们的页面时把那些文件加进规则的 `files` |
````
````new docs/v0/M3-workspace-project/M3-design.md
| `project-address-route`（P10） | P10、P11 改过的项目页从 router store 读项目和工作区（`currentProjectDetails`、`projectMemberIds`、`currentWorkspace`、不传 slug 和项目的 `allowPermissions`）；M4、M6、M7 改它们的页面时把那些文件加进规则的 `files` |
| `workspace-address-route`（收尾） | 收尾改过的 19 个工作区页面文件从 router store 读工作区（`currentWorkspace`、不传 slug 的工作区级 `allowPermissions`），而不是经 `useRouteWorkspace`；M4、M7、M8 改它们的页面时把那些文件加进规则的 `files` |
````

`tools/keywords.json`（修改，1 处）：

````old tools/keywords.json
    },
    {
      "id": "copy-hook",
````
````new tools/keywords.json
    },
    {
      "id": "workspace-address-route",
      "phase": "M3/closeout",
      "why": "工作区的页面从路由读工作区，不从 router store 读：workspace store 的 currentWorkspace 读 router store 的 workspaceSlug，工作区级的 allowPermissions 不传 slug 时也退回 router store（permissions.store.ts）。router store 由 StoreWrapper 的 effect 在渲染之后写入，换了工作区时晚一次提交，那一次页面按上一个工作区显示名称、判断权限（与 project-address-route 同理；M3 收尾，K12）。页面要的工作区是 useRouteWorkspace（web/apps/web/core/hooks/use-route-workspace.ts），工作区级的 allowPermissions 传路由的 slug。本规则只管收尾改过来的 19 个文件：工作区首页、个人主页的页头和布局、侧栏、项目列表页、工作区的常规和成员设置页、设置侧栏的头部；M4、M7、M8 的页面改它们的读法时，把那些文件加进本规则的 files。store 自己的 getter（如 joinedProjectIds、workspaceMemberIds）仍按 router store，同 project-address-route 留给 store 的那些。测试不算",
      "files": {
        "source": "^web/apps/web/(?:app/\\(all\\)/\\[workspaceSlug\\]/\\(projects\\)/(?:page|sidebar|extended-project-sidebar|profile/\\[userId\\]/(?:header|layout))|app/\\(all\\)/\\[workspaceSlug\\]/\\(settings\\)/settings/\\(workspace\\)/(?:page|members/page)|core/components/(?:home/widgets/empty-states/no-projects|profile/sidebar|project/(?:card-list|header|root)|settings/workspace/sidebar/header|workspace/settings/(?:member-columns|useMemberColumns|workspace-details)|workspace/sidebar/(?:projects-list|quick-actions|workspace-menu-root)))\\.tsx$",
        "flags": ""
      },
      "content": {
        "source": "\\bcurrentWorkspace\\b|\\ballowPermissions\\(\\s*\\[[^\\]]*\\]\\s*,\\s*EUserPermissionsLevel\\.WORKSPACE\\s*,?\\s*\\)",
        "flags": ""
      },
      "samples": {
        "hit": [
          "  const { currentWorkspace } = useWorkspace();",
          "  const { currentWorkspace: activeWorkspace, workspaces } = useWorkspace();",
          "  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);",
          "  const isAuthorizedUser = allowPermissions(\n    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],\n    EUserPermissionsLevel.WORKSPACE\n  );"
        ],
        "miss": [
          "  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE, workspaceSlug);",
          "  const isAuthorizedUser = allowPermissions(\n    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],\n    EUserPermissionsLevel.WORKSPACE,\n    workspaceSlug\n  );",
          "  const { currentWorkspaceFilters, currentWorkspaceAppliedDisplayFilters } = useProjectFilter();",
          "  const workspace = useRouteWorkspace();"
        ],
        "files": {
          "hit": [
            "web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx",
            "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx",
            "web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx"
          ],
          "miss": [
            "web/apps/web/core/hooks/use-route-workspace.ts",
            "web/apps/web/app/(all)/[workspaceSlug]/(projects)/drafts/header.tsx",
            "web/apps/web/core/components/workspace/settings/members-list.tsx"
          ]
        }
      }
    },
    {
      "id": "copy-hook",
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx
  const pageTitle = currentWorkspace?.name ? `${currentWorkspace?.name} - ${t("home.title")}` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx
  const pageTitle = workspace ? `${workspace.name} - ${t("home.title")}` : undefined;
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  const { member } = useProfileMember(currentWorkspace, userId ?? "");
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  const { member } = useProfileMember(workspace, userId ?? "");
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx
  const { profileViewId } = params;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx
  const { workspaceSlug, profileViewId } = params;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
import { observer } from "mobx-react";
import { useParams } from "react-router";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
export const AppSidebar = observer(function AppSidebar() {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
export const AppSidebar = observer(function AppSidebar() {
  // router
  const { workspaceSlug } = useParams();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`（修改，5 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { useMember } from "@/hooks/store/use-member";
import { useUserPermissions } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const canPerformWorkspaceAdminActions = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const canPerformWorkspaceAdminActions = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
  );
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
    EUserPermissionsLevel.WORKSPACE
  );
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
  );
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const pageTitle = currentWorkspace?.name
    ? t("workspace_settings.settings.members.page_label", { workspace: currentWorkspace.name })
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const pageTitle = workspace
    ? t("workspace_settings.settings.members.page_label", { workspace: workspace.name })
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx
  const pageTitle = currentWorkspace?.name
    ? t("workspace_settings.page_label", { workspace: currentWorkspace.name })
    : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx
  const pageTitle = workspace ? t("workspace_settings.page_label", { workspace: workspace.name }) : undefined;
````

`web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx`（修改，4 处）：

````old web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
import { useProject } from "@/hooks/store/use-project";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
import { useProject } from "@/hooks/store/use-project";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
  const { currentWorkspace: activeWorkspace } = useWorkspace();
````
````new web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
  const activeWorkspace = useRouteWorkspace();
````
````old web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````
````old web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
  const isWorkspaceAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);
````
````new web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx
  const isWorkspaceAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE, workspaceSlug);
````

`web/apps/web/core/components/profile/sidebar.tsx`（修改，3 处）：

````old web/apps/web/core/components/profile/sidebar.tsx
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser } from "@/hooks/store/user";
````
````new web/apps/web/core/components/profile/sidebar.tsx
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useUser } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/core/components/profile/sidebar.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/core/components/profile/sidebar.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/core/components/profile/sidebar.tsx
  const profileMember = useProfileMember(currentWorkspace, userId ?? "");
````
````new web/apps/web/core/components/profile/sidebar.tsx
  const profileMember = useProfileMember(workspace, userId ?? "");
````

`web/apps/web/core/components/project/card-list.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/card-list.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/core/components/project/card-list.tsx
import { observer } from "mobx-react";
import { useParams } from "react-router";
````
````old web/apps/web/core/components/project/card-list.tsx
export const ProjectCardList = observer(function ProjectCardList() {
````
````new web/apps/web/core/components/project/card-list.tsx
export const ProjectCardList = observer(function ProjectCardList() {
  // router
  const { workspaceSlug } = useParams();
````
````old web/apps/web/core/components/project/card-list.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/core/components/project/card-list.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/core/components/project/header.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/header.tsx
import { useMatch } from "react-router";
````
````new web/apps/web/core/components/project/header.tsx
import { useMatch, useParams } from "react-router";
````
````old web/apps/web/core/components/project/header.tsx
export const ProjectsBaseHeader = observer(function ProjectsBaseHeader() {
````
````new web/apps/web/core/components/project/header.tsx
export const ProjectsBaseHeader = observer(function ProjectsBaseHeader() {
  // router
  const { workspaceSlug } = useParams();
````
````old web/apps/web/core/components/project/header.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/core/components/project/header.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/core/components/project/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/root.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````new web/apps/web/core/components/project/root.tsx
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/core/components/project/root.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/core/components/project/root.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/core/components/project/root.tsx
  const pageTitle = currentWorkspace?.name
    ? `${currentWorkspace?.name} - ${t("workspace_projects.label", { count: 2 })}`
    : undefined;
````
````new web/apps/web/core/components/project/root.tsx
  const pageTitle = workspace ? `${workspace.name} - ${t("workspace_projects.label", { count: 2 })}` : undefined;
````
````old web/apps/web/core/components/project/root.tsx
  }, [clearAllFilters, clearAllAppliedDisplayFilters, workspaceSlug]);

````
````new web/apps/web/core/components/project/root.tsx
  }, [clearAllFilters, clearAllAppliedDisplayFilters, isArchived, updateDisplayFilters, workspaceSlug]);

  // The archives show the archived projects, the list the others. Each mounts this root apart, so the effect runs at
  // the mount and when the workspace changes, as it did on the address.
````
````old web/apps/web/core/components/project/root.tsx
  }, [pathname]);
````
````new web/apps/web/core/components/project/root.tsx
  }, [isArchived, updateDisplayFilters, workspaceSlug]);
````

`web/apps/web/core/components/settings/workspace/sidebar/header.tsx`（修改，6 处）：

````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useNavigate } from "react-router";
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
import { useNavigate } from "react-router";
````
````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
  const { currentWorkspace } = useWorkspace();
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
  const workspace = useRouteWorkspace();
````
````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
  const currentWorkspaceRole = currentWorkspace?.slug
    ? getWorkspaceRoleByWorkspaceSlug(currentWorkspace.slug)
    : undefined;
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
  const currentWorkspaceRole = workspace ? getWorkspaceRoleByWorkspaceSlug(workspace.slug) : undefined;
````
````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
          onClick={() => navigate(`/${currentWorkspace?.slug}`)}
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
          onClick={() => navigate(`/${workspace?.slug}`)}
````
````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
            logo={currentWorkspace?.logo_url}
            name={currentWorkspace?.name}
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
            logo={workspace?.logo_url}
            name={workspace?.name}
````
````old web/apps/web/core/components/settings/workspace/sidebar/header.tsx
            <p className="truncate text-body-sm-medium">{currentWorkspace?.name}</p>
````
````new web/apps/web/core/components/settings/workspace/sidebar/header.tsx
            <p className="truncate text-body-sm-medium">{workspace?.name}</p>
````

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/member-columns.tsx
  const isAdminRole = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
  const isAdminRole = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE, workspaceSlug);
````

`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE, workspaceSlug);
````

`web/apps/web/core/components/workspace/settings/workspace-details.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const { getWorkspaceBySlug, updateWorkspace } = useWorkspace();
  const workspace = workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const { updateWorkspace } = useWorkspace();
  const workspace = useRouteWorkspace();
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE);
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.WORKSPACE, workspaceSlug);
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/core/components/workspace/sidebar/quick-actions.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/quick-actions.tsx
    EUserPermissionsLevel.WORKSPACE
````
````new web/apps/web/core/components/workspace/sidebar/quick-actions.tsx
    EUserPermissionsLevel.WORKSPACE,
    workspaceSlug
````

`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
import { useInstance } from "@/hooks/store/use-instance";
````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
import { useInstance } from "@/hooks/store/use-instance";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
````
````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
  const { currentWorkspace: activeWorkspace, workspaces } = useWorkspace();
````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
  const { workspaces } = useWorkspace();
  const activeWorkspace = useRouteWorkspace();
````

`web/apps/web/core/hooks/use-route-workspace.ts`（新文件，20 行）：

````file web/apps/web/core/hooks/use-route-workspace.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Workspace } from "@nerve/api-client";
import { useParams } from "react-router";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";

/**
 * The workspace the route names (its workspaceSlug), as the caller's list holds it, or null: read from the route, not
 * from the router store, which StoreWrapper's effect writes after the render, so that it follows a change of
 * workspace one commit late (the keyword rule workspace-address-route; P10's project-address-route for projects).
 */
export function useRouteWorkspace(): Workspace | null {
  const { workspaceSlug } = useParams();
  const { getWorkspaceBySlug } = useWorkspace();
  return workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;
}
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 300",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 298",
````

- [ ] **Step 2: 测试**

`web/apps/web/core/hooks/use-route-workspace.test.ts`（新文件，38 行）：

````file web/apps/web/core/hooks/use-route-workspace.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useRouteWorkspace } from "./use-route-workspace";

// Which workspace a page of the workspace is about: the hook runs as a plain function, with stand-ins for the route's
// params and for the caller's list, which holds acme and beta, while the router store (not read) names neither.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const route = vi.hoisted((): { params: { workspaceSlug?: string } } => ({ params: {} }));
vi.mock("react-router", () => ({ useParams: () => route.params }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) => [acme, beta].find((workspace) => workspace.slug === slug) ?? null,
  }),
}));

beforeEach(() => {
  route.params = {};
});

describe("useRouteWorkspace", () => {
  it("is the workspace the route names, as the caller's list holds it", () => {
    route.params = { workspaceSlug: "beta" };
    expect(useRouteWorkspace()).toBe(beta);
  });

  it("is none for a route that names no workspace, or one the caller's list does not hold", () => {
    expect(useRouteWorkspace()).toBeNull();
    route.params = { workspaceSlug: "gamma" };
    expect(useRouteWorkspace()).toBeNull();
  });
});
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `71 rules, 23 exceptions, no hits.`；web 的 oxlint 298 条、`@nerve/ui` 12 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 116 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add docs/v0/M3-workspace-project/M3-design.md tools/keywords.json 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/page.tsx' web/apps/web/core/components/home/widgets/empty-states/no-projects.tsx web/apps/web/core/components/profile/sidebar.tsx web/apps/web/core/components/project/card-list.tsx web/apps/web/core/components/project/header.tsx web/apps/web/core/components/project/root.tsx web/apps/web/core/components/settings/workspace/sidebar/header.tsx web/apps/web/core/components/workspace/settings/member-columns.tsx web/apps/web/core/components/workspace/settings/useMemberColumns.tsx web/apps/web/core/components/workspace/settings/workspace-details.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/components/workspace/sidebar/quick-actions.tsx web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx web/apps/web/core/hooks/use-route-workspace.test.ts web/apps/web/core/hooks/use-route-workspace.ts web/apps/web/package.json
```
```bash
git commit -m "refactor(M3/closeout): pages that read the workspace read it from the route

useRouteWorkspace() finds the route's workspace in the caller's list; the
19 files of M3's pages that read the router store's currentWorkspace, or
judged a workspace permission without naming the workspace, now use it or
name the slug. The workspace-address-route rule keeps them so. The
projects root's callback and effect name what they read.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T11.1` | `useRouteWorkspace` 给不出工作区（K12） | `use-route-workspace.test.ts`、故事 W3 | vitest；端到端 |
| `T11.2` | `useRouteWorkspace` 读 router store 的工作区，不是路由的（K12） | `use-route-workspace.test.ts` | vitest |
| `T11.3` | 工作区菜单又读 router store 的工作区（`workspace-address-route`） | 关键词守卫 | 静态 |
| `T11.4` | 侧边栏的快捷操作按 router store 的工作区判断权限（`workspace-address-route`） | 关键词守卫 | 静态 |
| `T11.5` | 项目页的根的副作用又只依赖地址：oxlint 的 `exhaustive-deps` 超出上限 | oxlint（`check:lint`） | 静态 |

---

### Task 12: 项目列表页、侧边栏和标签栏上的英文；筛选的名字

**Files:**
- Create: `web/apps/web/core/components/navigation/tab-navigation-visible-item.test.tsx`、`web/apps/web/core/components/project/applied-filters/root.test.tsx`
- Modify: `e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.test.tsx`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`、`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`、`web/apps/web/core/components/project/applied-filters/access.tsx`、`web/apps/web/core/components/project/applied-filters/date.tsx`、`web/apps/web/core/components/project/applied-filters/members.tsx`、`web/apps/web/core/components/project/applied-filters/project-display-filters.tsx`、`web/apps/web/core/components/project/applied-filters/root.tsx`、`web/apps/web/core/components/project/card.test.tsx`、`web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/dropdowns/filters/access.tsx`、`web/apps/web/core/components/project/dropdowns/filters/created-at.tsx`、`web/apps/web/core/components/project/dropdowns/filters/member-list.tsx`、`web/apps/web/core/components/project/dropdowns/filters/members.tsx`、`web/apps/web/core/components/project/dropdowns/filters/root.tsx`、`web/apps/web/core/components/project/header.tsx`、`web/apps/web/core/components/sidebar/sidebar-wrapper.tsx`、`web/apps/web/package.json`、`web/packages/constants/src/filter.ts`、`web/packages/constants/src/project.ts`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/navigation.json`、`web/packages/i18n/src/locales/en/project.json`、`web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/navigation.json`、`web/packages/i18n/src/locales/zh-CN/project.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json`
- Delete: `web/apps/web/core/components/project/dropdowns/filters/lead.tsx`

**Interfaces:**（spec 2.8）
- 侧边栏的标题经 `t("projects")`：`SidebarWrapper` 的 `title`（只有一个调用方，总是"Projects"）和 `title === "Projects"` 的条件删除；扩展侧边栏的标题同样。标签栏的菜单经 `navigation.json` 的 `tab_navigation.*`。项目页页头的"已归档"，卡片的创建日期和成员数。
- 筛选：`@nerve/constants` 的 `PROJECT_FILTER_I18N_LABEL`（下拉框和已选筛选的一栏共用）；`lead.tsx`（`members.tsx` 的副本）删除，`FilterMembers` 收 `title`；网络按页面上的名字搜索；创建日期的选项 `i18n_name`；已选筛选的 9 个移除按钮名为 `common.remove_item`；数目的提示。成员筛选：角色经 `ROLE_DETAILS` 和"已停用"，按钮里不再有按钮（`customButtonClassName`）。
- 这几个文件自己的警告：已选筛选的两个数组改为 `Set`；筛选的搜索框的 `autoFocus` 写明理由（`oxlint-disable-next-line`）。web 的上限 298 → 295。

**Tests:** `tab-navigation-visible-item.test.tsx`、`applied-filters/root.test.tsx`（新文件）；`card.test.tsx`、`tab-navigation-overflow-menu.test.tsx` 读键。S7：第二个加扩展侧边栏的标题；第四个（新）：项目页的标题、卡片、筛选、已选筛选的一栏、已归档、成员筛选。

- [ ] **Step 1: 页面和筛选**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
            <span className="py-1.5 text-13 font-semibold text-tertiary">Projects</span>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
            <span className="py-1.5 text-13 font-semibold text-tertiary">{t("projects")}</span>
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
    <SidebarWrapper title="Projects" quickActions={<SidebarQuickActions />}>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx
    <SidebarWrapper quickActions={<SidebarQuickActions />}>
````

`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`（修改，3 处）：

````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                    title="Show"
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                    title={t("tab_navigation.show")}
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                  <Tooltip label={isDefault ? "Clear default" : "Set as default"}>
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                  <Tooltip label={t(isDefault ? "tab_navigation.clear_default" : "tab_navigation.set_as_default")}>
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                      title={isDefault ? "Clear default" : "Set as default"}
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                      title={t(isDefault ? "tab_navigation.clear_default" : "tab_navigation.set_as_default")}
````

`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
                  <span className="text-11">{isDefault ? "Clear default" : "Set as default"}</span>
````
````new web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
                  <span className="text-11">
                    {t(isDefault ? "tab_navigation.clear_default" : "tab_navigation.set_as_default")}
                  </span>
````
````old web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
                  <span className="text-11">Hide in more menu</span>
````
````new web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
                  <span className="text-11">{t("tab_navigation.hide_in_more_menu")}</span>
````

`web/apps/web/core/components/project/applied-filters/access.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/applied-filters/access.tsx
        const accessDetails = NETWORK_CHOICES.find((s) => `${s.key}` === status);
````
````new web/apps/web/core/components/project/applied-filters/access.tsx
        const accessDetails = NETWORK_CHOICES.find((s) => `${s.key}` === status);
        const label = accessDetails ? t(accessDetails.i18n_label) : "";
````
````old web/apps/web/core/components/project/applied-filters/access.tsx
            {accessDetails && t(accessDetails?.i18n_label)}
````
````new web/apps/web/core/components/project/applied-filters/access.tsx
            {label}
````
````old web/apps/web/core/components/project/applied-filters/access.tsx
                onClick={() => handleRemove(status)}
````
````new web/apps/web/core/components/project/applied-filters/access.tsx
                onClick={() => handleRemove(status)}
                aria-label={t("common.remove_item", { name: label })}
````

`web/apps/web/core/components/project/applied-filters/date.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/applied-filters/date.tsx
import { PROJECT_CREATED_AT_FILTER_OPTIONS } from "@nerve/constants";
````
````new web/apps/web/core/components/project/applied-filters/date.tsx
import { PROJECT_CREATED_AT_FILTER_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/applied-filters/date.tsx
  const { editable, handleRemove, values } = props;
````
````new web/apps/web/core/components/project/applied-filters/date.tsx
  const { editable, handleRemove, values } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/applied-filters/date.tsx
    if (dateDetails) dateLabel = dateDetails.name;
````
````new web/apps/web/core/components/project/applied-filters/date.tsx
    if (dateDetails) dateLabel = t(dateDetails.i18n_name);
````
````old web/apps/web/core/components/project/applied-filters/date.tsx
              onClick={() => handleRemove(date)}
````
````new web/apps/web/core/components/project/applied-filters/date.tsx
              onClick={() => handleRemove(date)}
              aria-label={t("common.remove_item", { name: getDateLabel(date) })}
````

`web/apps/web/core/components/project/applied-filters/members.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/applied-filters/members.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/core/components/project/applied-filters/members.tsx
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/applied-filters/members.tsx
  } = useMember();
````
````new web/apps/web/core/components/project/applied-filters/members.tsx
  } = useMember();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/applied-filters/members.tsx
                onClick={() => handleRemove(memberId)}
````
````new web/apps/web/core/components/project/applied-filters/members.tsx
                onClick={() => handleRemove(memberId)}
                aria-label={t("common.remove_item", { name: memberDetails.display_name })}
````

`web/apps/web/core/components/project/applied-filters/project-display-filters.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/applied-filters/project-display-filters.tsx
        const filterLabel = PROJECT_DISPLAY_FILTER_OPTIONS.find((s) => s.key === key)?.i18n_label;
````
````new web/apps/web/core/components/project/applied-filters/project-display-filters.tsx
        const filterLabel = PROJECT_DISPLAY_FILTER_OPTIONS.find((s) => s.key === key)?.i18n_label;
        const label = filterLabel ? t(filterLabel) : "";
````
````old web/apps/web/core/components/project/applied-filters/project-display-filters.tsx
            {filterLabel && t(filterLabel)}
````
````new web/apps/web/core/components/project/applied-filters/project-display-filters.tsx
            {label}
````
````old web/apps/web/core/components/project/applied-filters/project-display-filters.tsx
                onClick={() => handleRemove(key)}
````
````new web/apps/web/core/components/project/applied-filters/project-display-filters.tsx
                onClick={() => handleRemove(key)}
                aria-label={t("common.remove_item", { name: label })}
````

`web/apps/web/core/components/project/applied-filters/root.tsx`（修改，9 处）：

````old web/apps/web/core/components/project/applied-filters/root.tsx
// nerve imports
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
// nerve imports
import { PROJECT_FILTER_I18N_LABEL } from "@nerve/constants";
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
import { EHeaderVariant, Header } from "@nerve/ui";
import { replaceUnderscoreIfSnakeCase } from "@nerve/utils";
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
import { EHeaderVariant, Header } from "@nerve/ui";
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
const MEMBERS_FILTERS = ["lead", "members"];
const DATE_FILTERS = ["created_at"];
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
const MEMBERS_FILTERS = new Set(["lead", "members"]);
const DATE_FILTERS = new Set(["created_at"]);
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
          if (Array.isArray(value) && value.length === 0) return;
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
          if (Array.isArray(value) && value.length === 0) return;
          const label = t(PROJECT_FILTER_I18N_LABEL[filterKey]);
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
              <span className="text-11 text-tertiary">{replaceUnderscoreIfSnakeCase(filterKey)}</span>
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
              <span className="text-11 text-tertiary">{label}</span>
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
              {DATE_FILTERS.includes(filterKey) && (
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
              {DATE_FILTERS.has(filterKey) && (
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
              {MEMBERS_FILTERS.includes(filterKey) && (
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
              {MEMBERS_FILTERS.has(filterKey) && (
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
                  onClick={() => handleRemoveFilter(filterKey, null)}
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
                  onClick={() => handleRemoveFilter(filterKey, null)}
                  aria-label={t("common.remove_item", { name: label })}
````
````old web/apps/web/core/components/project/applied-filters/root.tsx
        <Tooltip label={`${filteredProjects} of ${totalProjects} projects match the applied filters.`} layout="stacked">
````
````new web/apps/web/core/components/project/applied-filters/root.tsx
        <Tooltip
          label={t("workspace_projects.applied_filters_count", { filtered: filteredProjects, total: totalProjects })}
          layout="stacked"
        >
````

`web/apps/web/core/components/project/card.tsx`（修改，2 处）：

````old web/apps/web/core/components/project/card.tsx
              : `Created on ${renderFormattedDate(project.created_at)}`}
````
````new web/apps/web/core/components/project/card.tsx
              : `${t("common.created_on")} ${renderFormattedDate(project.created_at)}`}
````
````old web/apps/web/core/components/project/card.tsx
                label={projectMembersIds.length ? `Members: ${projectMembersIds.length}` : "No members"}
````
````new web/apps/web/core/components/project/card.tsx
                label={
                  projectMembersIds.length
                    ? t("project_dialogs.members_count", { count: projectMembersIds.length })
                    : t("project_dialogs.no_member_yet")
                }
````

`web/apps/web/core/components/project/dropdowns/filters/access.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/dropdowns/filters/access.tsx
import { NETWORK_CHOICES } from "@nerve/constants";
````
````new web/apps/web/core/components/project/dropdowns/filters/access.tsx
import { NETWORK_CHOICES, PROJECT_FILTER_I18N_LABEL } from "@nerve/constants";
````
````old web/apps/web/core/components/project/dropdowns/filters/access.tsx
  const filteredOptions = NETWORK_CHOICES.filter((a) => a.i18n_label.includes(searchQuery.toLowerCase()));
````
````new web/apps/web/core/components/project/dropdowns/filters/access.tsx
  // the search matches each access as the page names it
  const filteredOptions = NETWORK_CHOICES.filter((a) =>
    t(a.i18n_label).toLowerCase().includes(searchQuery.toLowerCase())
  );
````
````old web/apps/web/core/components/project/dropdowns/filters/access.tsx
        title={`Access${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
````
````new web/apps/web/core/components/project/dropdowns/filters/access.tsx
        title={`${t(PROJECT_FILTER_I18N_LABEL.access)}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
````
````old web/apps/web/core/components/project/dropdowns/filters/access.tsx
            <p className="text-11 text-placeholder italic">No matches found</p>
````
````new web/apps/web/core/components/project/dropdowns/filters/access.tsx
            <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
````

`web/apps/web/core/components/project/dropdowns/filters/created-at.tsx`（修改，8 处）：

````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
import { PROJECT_CREATED_AT_FILTER_OPTIONS } from "@nerve/constants";
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
import { PROJECT_CREATED_AT_FILTER_OPTIONS, PROJECT_FILTER_I18N_LABEL } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
  const [isDateFilterModalOpen, setIsDateFilterModalOpen] = useState(false);
  // derived values
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
  const [isDateFilterModalOpen, setIsDateFilterModalOpen] = useState(false);
  const { t } = useTranslation();
  // derived values
  const title = t(PROJECT_FILTER_I18N_LABEL.created_at);
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
    d.name.toLowerCase().includes(searchQuery.toLowerCase())
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
    t(d.i18n_name).toLowerCase().includes(searchQuery.toLowerCase())
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
          title="Created date"
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
          title={title}
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
        title={`Created date${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
        title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
                  title={option.name}
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
                  title={t(option.i18n_name)}
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
                title="Custom"
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
                title={t("workspace_projects.created_at_filter.custom")}
````
````old web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
            <p className="text-11 text-placeholder italic">No matches found</p>
````
````new web/apps/web/core/components/project/dropdowns/filters/created-at.tsx
            <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
````

`web/apps/web/core/components/project/dropdowns/filters/lead.tsx`（删除）：

````delete web/apps/web/core/components/project/dropdowns/filters/lead.tsx
````

`web/apps/web/core/components/project/dropdowns/filters/member-list.tsx`（整个文件，123 行）：

````whole web/apps/web/core/components/project/dropdowns/filters/member-list.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { ROLE_DETAILS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import { ChevronDownOutline } from "@makeplane/propel/icons";
import { EUserWorkspaceRoles } from "@nerve/types";
import { cn } from "@nerve/utils";
// nerve ui
import { CustomMenu } from "@nerve/ui";
// components
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";

/** A role the members are filtered by: its value in the filter, and the key of its name. */
interface IRoleOption {
  value: string;
  i18n_label: string;
}

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (role: string) => void;
  memberType: "project" | "workspace";
};

// a project's roles have the workspace's values (EUserProjectRoles), and the same names
const PROJECT_ROLE_OPTIONS: IRoleOption[] = [
  EUserWorkspaceRoles.ADMIN,
  EUserWorkspaceRoles.MEMBER,
  EUserWorkspaceRoles.GUEST,
].map((role) => ({ value: String(role), i18n_label: ROLE_DETAILS[role].i18n_title }));

const WORKSPACE_ROLE_OPTIONS: IRoleOption[] = [
  ...PROJECT_ROLE_OPTIONS,
  { value: "suspended", i18n_label: "workspace_settings.settings.members.suspended" },
];

// Role filter group component
const RoleFilterGroup = observer(function RoleFilterGroup({
  appliedFilters,
  handleUpdate,
  memberType,
}: {
  appliedFilters: string[] | null;
  handleUpdate: (role: string) => void;
  memberType: "project" | "workspace";
}) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { t } = useTranslation();
  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const roleOptions = memberType === "project" ? PROJECT_ROLE_OPTIONS : WORKSPACE_ROLE_OPTIONS;

  return (
    <div className="space-y-2">
      <FilterHeader
        title={`${t("common.roles")}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={isExpanded}
        handleIsPreviewEnabled={() => setIsExpanded(!isExpanded)}
      />

      {isExpanded && (
        <div className="space-y-1">
          {roleOptions.map((role) => {
            const isSelected = appliedFilters?.includes(role.value) ?? false;
            return (
              <FilterOption
                key={`role-${role.value}`}
                isChecked={isSelected}
                title={t(role.i18n_label)}
                onClick={() => handleUpdate(role.value)}
              />
            );
          })}
        </div>
      )}
    </div>
  );
});

const MemberListFilters = observer(function MemberListFilters(props: Props) {
  const { appliedFilters, handleUpdate, memberType } = props;

  return (
    <div className="space-y-4">
      {/* Role Filter Group */}
      <RoleFilterGroup appliedFilters={appliedFilters} handleUpdate={handleUpdate} memberType={memberType} />
    </div>
  );
});

// Dropdown component for member list filters
export const MemberListFiltersDropdown = observer(function MemberListFiltersDropdown(props: Props) {
  const { appliedFilters, handleUpdate, memberType } = props;
  const { t } = useTranslation();

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  // the menu's own button looks like a button: no button inside it
  return (
    <CustomMenu
      customButtonClassName={cn(getButtonStyling("secondary", "lg"), "relative gap-2")}
      customButton={
        <>
          <span>{t("common.filters")}</span>
          <ChevronDownOutline className="h-3 w-3" />
          {appliedFiltersCount > 0 && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-accent-primary" />
          )}
        </>
      }
      placement="bottom-start"
    >
      <MemberListFilters appliedFilters={appliedFilters} handleUpdate={handleUpdate} memberType={memberType} />
    </CustomMenu>
  );
});
````

`web/apps/web/core/components/project/dropdowns/filters/members.tsx`（修改，9 处）：

````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
// nerve ui
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
// nerve ui
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
type Props = {
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
type Props = {
  /** The filter's name: the projects' lead, or their members. */
  title: string;
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
};

export const FilterMembers = observer(function FilterMembers(props: Props) {
  const { appliedFilters, handleUpdate, memberIds, searchQuery } = props;
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
};

/** A filter of the projects by workspace members, who lead them or are their members. */
export const FilterMembers = observer(function FilterMembers(props: Props) {
  const { title, appliedFilters, handleUpdate, memberIds, searchQuery } = props;
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
  const { data: currentUser } = useUser();
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
  const { data: currentUser } = useUser();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
        title={`Members${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
        title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
                      key={`member-${member.id}`}
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
                      key={member.id}
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
                      title={currentUser?.id === member.id ? "You" : member?.display_name}
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
                      title={currentUser?.id === member.id ? t("you") : member?.display_name}
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
                    {itemsToRender === sortedOptions.length ? "View less" : "View all"}
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
                    {t(itemsToRender === sortedOptions.length ? "common.view_less" : "common.view_all")}
````
````old web/apps/web/core/components/project/dropdowns/filters/members.tsx
              <p className="text-11 text-placeholder italic">No matches found</p>
````
````new web/apps/web/core/components/project/dropdowns/filters/members.tsx
              <p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>
````

`web/apps/web/core/components/project/dropdowns/filters/root.tsx`（修改，8 处）：

````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
// nerve imports
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
// nerve imports
import { PROJECT_FILTER_I18N_LABEL } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
import { FilterCreatedDate } from "./created-at";
import { FilterLead } from "./lead";
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
import { FilterCreatedDate } from "./created-at";
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
  const { isMobile } = usePlatformOS();
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
            placeholder="Search"
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
            placeholder={t("search")}
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
            onChange={(e) => setFiltersSearchQuery(e.target.value)}
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
            onChange={(e) => setFiltersSearchQuery(e.target.value)}
            // oxlint-disable-next-line jsx_a11y/no-autofocus -- the dropdown opens on its search, as a combobox does
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
            title="My projects"
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
            title={t("workspace_projects.scope.my_projects")}
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
          <FilterLead
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
          <FilterMembers
            title={t(PROJECT_FILTER_I18N_LABEL.lead)}
````
````old web/apps/web/core/components/project/dropdowns/filters/root.tsx
          <FilterMembers
            appliedFilters={filters.members ?? null}
````
````new web/apps/web/core/components/project/dropdowns/filters/root.tsx
          <FilterMembers
            title={t(PROJECT_FILTER_I18N_LABEL.members)}
            appliedFilters={filters.members ?? null}
````

`web/apps/web/core/components/project/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/header.tsx
          {isArchived && <Breadcrumbs.Item component={<BreadcrumbLink label="Archived" />} />}
````
````new web/apps/web/core/components/project/header.tsx
          {isArchived && (
            <Breadcrumbs.Item component={<BreadcrumbLink label={t("workspace_projects.scope.archived_projects")} />} />
          )}
````

`web/apps/web/core/components/sidebar/sidebar-wrapper.tsx`（修改，4 处）：

````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
type TSidebarWrapperProps = {
  title: string;
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
type TSidebarWrapperProps = {
````
````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
  const { title, children, quickActions } = props;
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
  const { children, quickActions } = props;
````
````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
            <span className="pt-1 text-16 font-medium text-primary">{title}</span>
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
            <span className="pt-1 text-16 font-medium text-primary">{t("projects")}</span>
````
````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
              {title === "Projects" && (
                <IconButton
                  size="base"
                  variant="ghost"
                  icon={PreferencesOutline}
                  onClick={() => setIsProjectNavDialogOpen(true)}
                  aria-label={t("aria_labels.projects_sidebar.project_navigation")}
                />
              )}
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
              <IconButton
                size="base"
                variant="ghost"
                icon={PreferencesOutline}
                onClick={() => setIsProjectNavDialogOpen(true)}
                aria-label={t("aria_labels.projects_sidebar.project_navigation")}
              />
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 298",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 295",
````

`web/packages/constants/src/filter.ts`（修改，4 处）：

````old web/packages/constants/src/filter.ts
    name: "Today",
````
````new web/packages/constants/src/filter.ts
    i18n_name: "common.today",
````
````old web/packages/constants/src/filter.ts
    name: "Yesterday",
````
````new web/packages/constants/src/filter.ts
    i18n_name: "workspace_projects.created_at_filter.yesterday",
````
````old web/packages/constants/src/filter.ts
    name: "Last 7 days",
````
````new web/packages/constants/src/filter.ts
    i18n_name: "workspace_projects.created_at_filter.last_7_days",
````
````old web/packages/constants/src/filter.ts
    name: "Last 30 days",
````
````new web/packages/constants/src/filter.ts
    i18n_name: "workspace_projects.created_at_filter.last_30_days",
````

`web/packages/constants/src/project.ts`（修改，2 处）：

````old web/packages/constants/src/project.ts
import type { TProjectAppliedDisplayFilterKeys, TProjectOrderByOptions } from "@nerve/types";
````
````new web/packages/constants/src/project.ts
import type { TProjectAppliedDisplayFilterKeys, TProjectFilters, TProjectOrderByOptions } from "@nerve/types";
````
````old web/packages/constants/src/project.ts
];

export const PROJECT_DISPLAY_FILTER_OPTIONS: {
````
````new web/packages/constants/src/project.ts
];

/** The key of each filter's name of the projects, as their filters' dropdown and the bar of those applied say it. */
export const PROJECT_FILTER_I18N_LABEL: Record<keyof TProjectFilters, string> = {
  access: "workspace_projects.network.label",
  lead: "lead",
  members: "common.members",
  created_at: "workspace_projects.sort.created_at",
};

export const PROJECT_DISPLAY_FILTER_OPTIONS: {
````

`web/packages/i18n/src/locales/en/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/common.json
    "members": "Members",
````
````new web/packages/i18n/src/locales/en/common.json
    "members": "Members",
    "roles": "Roles",
    "view_all": "View all",
    "view_less": "View less",
    "remove_item": "Remove {name}",
````

`web/packages/i18n/src/locales/en/navigation.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/navigation.json
{
  "sidebar": {
````
````new web/packages/i18n/src/locales/en/navigation.json
{
  "tab_navigation": {
    "set_as_default": "Set as default",
    "clear_default": "Clear default",
    "show": "Show",
    "hide_in_more_menu": "Hide in more menu"
  },
  "sidebar": {
````

`web/packages/i18n/src/locales/en/project.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project.json
    "no_member_yet": "No Member Yet",
````
````new web/packages/i18n/src/locales/en/project.json
    "no_member_yet": "No Member Yet",
    "members_count": "Members: {count}",
````

`web/packages/i18n/src/locales/en/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace.json
      "archived_projects": "Archived"
    },
````
````new web/packages/i18n/src/locales/en/workspace.json
      "archived_projects": "Archived"
    },
    "applied_filters_count": "{filtered} of {total} projects match the applied filters.",
    "created_at_filter": {
      "yesterday": "Yesterday",
      "last_7_days": "Last 7 days",
      "last_30_days": "Last 30 days",
      "custom": "Custom"
    },
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
    "members": "成员",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
    "members": "成员",
    "roles": "角色",
    "view_all": "查看全部",
    "view_less": "收起",
    "remove_item": "移除{name}",
````

`web/packages/i18n/src/locales/zh-CN/navigation.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/navigation.json
{
  "sidebar": {
````
````new web/packages/i18n/src/locales/zh-CN/navigation.json
{
  "tab_navigation": {
    "set_as_default": "设为默认",
    "clear_default": "取消默认",
    "show": "显示",
    "hide_in_more_menu": "收进更多菜单"
  },
  "sidebar": {
````

`web/packages/i18n/src/locales/zh-CN/project.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project.json
    "no_member_yet": "还没有成员",
````
````new web/packages/i18n/src/locales/zh-CN/project.json
    "no_member_yet": "还没有成员",
    "members_count": "成员：{count}",
````

`web/packages/i18n/src/locales/zh-CN/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace.json
      "archived_projects": "已归档"
    },
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
      "archived_projects": "已归档"
    },
    "applied_filters_count": "{total} 个项目中有 {filtered} 个符合筛选条件。",
    "created_at_filter": {
      "yesterday": "昨天",
      "last_7_days": "最近 7 天",
      "last_30_days": "最近 30 天",
      "custom": "自定义"
    },
````

- [ ] **Step 2: 测试**

`e2e/stories/smoke/s7-chinese.spec.ts`（修改，2 处）：

````old e2e/stories/smoke/s7-chinese.spec.ts
  await expect(sidebar.getByRole("button", { name: "打开扩展侧边栏" })).toHaveText("更多");
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expect(sidebar.getByRole("button", { name: "打开扩展侧边栏" })).toHaveText("更多");
  // the extended sidebar, which the button opens, by its heading
  await sidebar.getByRole("button", { name: "打开扩展侧边栏" }).click();
  await expect(page.locator("#extended-project-sidebar-toggle").getByText("项目", { exact: true })).toBeVisible();
````
````old e2e/stories/smoke/s7-chinese.spec.ts
});

test("S7 (page): one whose language is Chinese reads her invitation to Acme, as a guest, in Chinese", async ({
````
````new e2e/stories/smoke/s7-chinese.spec.ts
});

test("S7 (page): an admin whose language is Chinese filters the projects and the members in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { admin, slug } = await chineseAdmin(api, testInfo);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);

  // The projects page: the sidebar's heading, the card's date and its members.
  await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await page.goto(`/${slug}/projects`);
  const navigation = page.getByRole("complementary", { name: "主侧边栏" }).getByRole("button", { name: "项目导航" });
  await expect(navigation.locator("xpath=../preceding-sibling::span")).toHaveText("项目");
  await expect(page.getByText(/^创建于 /)).toBeVisible();
  // The filters: each by its name, the accesses searched as the page names them.
  await page.getByRole("button", { name: "筛选", exact: true }).click();
  const filters = page.getByRole("button", { name: "我的项目" }).locator("xpath=../..");
  await expect(filters.getByText(/^(网络|负责人|成员|创建日期)$/)).toHaveText(["网络", "负责人", "成员", "创建日期"]);
  await expect(filters.getByRole("button", { name: /^(今天|昨天|最近 7 天|最近 30 天|自定义)$/ })).toHaveCount(5);
  // the caller, in the lead's filter and the members', is 您
  await expect(filters.getByRole("button", { name: / 您$/ })).toHaveCount(2);
  await page.getByPlaceholder("搜索", { exact: true }).last().fill("公开");
  await expect(page.getByRole("button", { name: "公开", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "私有", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "公开", exact: true }).click();
  await page.keyboard.press("Escape");
  // The bar of the filters applied: the filter by its name, each button named by what it removes, and the count.
  await expect(page.getByRole("button", { name: "移除公开", exact: true })).toBeVisible();
  await page.getByText("1/1", { exact: true }).hover();
  await expect(page.getByText("1 个项目中有 1 个符合筛选条件。", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "移除网络", exact: true }).click();
  await expect(page.getByRole("button", { name: "移除公开", exact: true })).toHaveCount(0);

  // The archived projects: the header's breadcrumb says so.
  await page.goto(`/${slug}/projects/archives`);
  await expect(page.getByText("已归档", { exact: true })).toHaveCount(1);

  // The members page: one button names its filter, whose roles have their names.
  await page.goto(`/${slug}/settings/members`);
  await expect(page.getByRole("button", { name: "筛选" })).toHaveCount(1);
  await page.getByRole("button", { name: "筛选" }).click();
  const roleFilter = page.getByRole("menu", { name: "筛选" });
  await expect(roleFilter.getByText("角色", { exact: true })).toBeVisible();
  const roles = roleFilter.getByRole("button", { name: /^(管理员|成员|访客|已停用)$/ });
  await expect(roles).toHaveText(["管理员", "成员", "访客", "已停用"]);
  await page.keyboard.press("Escape");
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese reads her invitation to Acme, as a guest, in Chinese", async ({
````

`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.test.tsx`（修改，1 处）：

````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.test.tsx
      ["Clear default"],
      ["Show", "Set as default"],
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.test.tsx
      ["tab_navigation.clear_default"],
      ["tab_navigation.show", "tab_navigation.set_as_default"],
````

`web/apps/web/core/components/navigation/tab-navigation-visible-item.test.tsx`（新文件，60 行）：

````file web/apps/web/core/components/navigation/tab-navigation-visible-item.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { ProjectNavigation } from "@nerve/api-client";
import type { TNavigationItem } from "./tab-navigation-root";
import { DEFAULT_NAVIGATION } from "./tab-navigation-utils";
import { TabNavigationVisibleItem } from "./tab-navigation-visible-item";
import type { TTabChanges } from "./use-tab-preferences";

// What the menu of a tab of a project's header offers, in the page's language (M3 design 3.18): making the tab the one
// the project opens on, or no longer, and moving it to the "more" menu. The tab renders on the server with a stand-in
// for propel's context menu, which renders its items as the open menu does, and the language says each key
// (fake-i18n.ts).

vi.mock("@nerve/propel/context-menu", () => ({
  ContextMenu: Object.assign(Shown, { Trigger: Shown, Portal: Shown, Content: Shown, Item: Shown }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Each part of the menu, shown with what it holds. */
function Shown({ children }: { children?: ReactNode }) {
  return <>{children}</>;
}

const cycles: TNavigationItem = {
  name: "cycles",
  href: "/acme/projects/p-web/cycles",
  icon: () => null,
  access: [],
  shouldRender: true,
  sortOrder: 2,
  i18n_key: "sidebar.cycles",
  key: "cycles",
};
const changes = { toggleDefault: () => undefined, hide: () => undefined, show: () => undefined } satisfies TTabChanges;

/** The menu of the cycles' tab, with the tab bar given. */
function menuOf(navigation: ProjectNavigation): string {
  return renderToStaticMarkup(
    <MemoryRouter>
      <TabNavigationVisibleItem item={cycles} isActive={false} navigation={navigation} changes={changes} />
    </MemoryRouter>
  );
}

describe("the menu of a tab of a project's header", () => {
  it("offers making the tab the one the project opens on, or no longer, and moving it to the more menu", () => {
    const menu = menuOf(DEFAULT_NAVIGATION);
    expect([menu.includes("tab_navigation.set_as_default"), menu.includes("tab_navigation.hide_in_more_menu")]).toEqual(
      [true, true]
    );
    expect(menuOf({ ...DEFAULT_NAVIGATION, default_tab: "cycles" })).toContain("tab_navigation.clear_default");
  });
});
````

`web/apps/web/core/components/project/applied-filters/root.test.tsx`（新文件，50 行）：

````file web/apps/web/core/components/project/applied-filters/root.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { ProjectAppliedFiltersList } from "./root";

// The bar of the projects' filters applied (the workspace's projects page): it names each filter in the page's
// language, and each button that removes a filter, or one of its values. The bar renders on the server, with a
// stand-in for the workspace's members (bob) and for the language, which says each key (fake-i18n.ts).

vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: { getWorkspaceMemberDetails: () => ({ member: { display_name: "bob", avatar_url: null } }) },
  }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

describe("the bar of the projects' filters applied", () => {
  it("names each filter, and each button that removes a filter or one of its values", () => {
    const bar = renderToStaticMarkup(
      <ProjectAppliedFiltersList
        appliedFilters={{ access: ["2"], lead: ["u-bob"], members: ["u-bob"], created_at: ["today;custom;custom"] }}
        appliedDisplayFilters={["my_projects"]}
        handleClearAllFilters={() => {}}
        handleRemoveFilter={() => {}}
        handleRemoveDisplayFilter={() => {}}
        filteredProjects={1}
        totalProjects={2}
        alwaysAllowEditing
      />
    );
    const names = [...bar.matchAll(/<span class="text-11 text-tertiary">([^<]*)<\/span>/g)].map(([, name]) => name);
    expect(names).toEqual([
      "workspace_projects.network.label",
      "lead",
      "common.members",
      "workspace_projects.sort.created_at",
      "common.projects",
    ]);
    // the four filters and their four values, and the projects' value: each its button, named; the bar's clearing
    // says what it does
    const buttons = bar.match(/<button[^>]*>/g) ?? [];
    expect(buttons.filter((button) => button.includes('aria-label="common.remove_item"'))).toHaveLength(9);
    expect(buttons).toHaveLength(10);
    expect(bar).toContain("common.clear_all");
  });
});
````

`web/apps/web/core/components/project/card.test.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/card.test.tsx
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/card.test.tsx
import type { ReactNode } from "react";
import type { Project, ProjectRole, WorkspaceRole } from "@nerve/api-client";
import { renderFormattedDate } from "@nerve/utils";
````
````old web/apps/web/core/components/project/card.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
````
````new web/apps/web/core/components/project/card.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
// a tooltip shows its label beside what it is the tooltip of
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ label, children }: { label: ReactNode; children: ReactNode }) => (
    <>
      <span data-tooltip="">{label}</span>
      {children}
    </>
  ),
}));
````
````old web/apps/web/core/components/project/card.test.tsx
  });

  it.each<Caller & { linked: boolean }>([
````
````new web/apps/web/core/components/project/card.test.tsx
  });

  it("says, in the page's language, when it was made, having no description, and how many members it has", async () => {
    const { permissions, acme, web } = await callerInWeb(its.admin.workspaceRole, its.admin.projectRole);
    caller.permissions = permissions;
    const cardOf = (project: Project) =>
      renderToStaticMarkup(
        <MemoryRouter initialEntries={[`/${acme.slug}/projects`]}>
          <Routes>
            <Route path=":workspaceSlug/projects" element={<ProjectCard project={project} />} />
          </Routes>
        </MemoryRouter>
      );
    const alone = cardOf({ ...web, member_ids: [] });
    expect(alone).toContain(`common.created_on ${renderFormattedDate(web.created_at)}`);
    expect(alone).toContain('<span data-tooltip="">project_dialogs.no_member_yet</span>');
    expect(cardOf({ ...web, member_ids: ["u-bob"] })).toContain(
      '<span data-tooltip="">project_dialogs.members_count 1</span>'
    );
  });

  it.each<Caller & { linked: boolean }>([
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `71 rules, 23 exceptions, no hits.`；web 的 oxlint 295 条、`@nerve/ui` 12 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 117 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/smoke/s7-chinese.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/sidebar.tsx' web/apps/web/core/components/navigation/tab-navigation-overflow-menu.test.tsx web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx web/apps/web/core/components/navigation/tab-navigation-visible-item.test.tsx web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx web/apps/web/core/components/project/applied-filters/access.tsx web/apps/web/core/components/project/applied-filters/date.tsx web/apps/web/core/components/project/applied-filters/members.tsx web/apps/web/core/components/project/applied-filters/project-display-filters.tsx web/apps/web/core/components/project/applied-filters/root.test.tsx web/apps/web/core/components/project/applied-filters/root.tsx web/apps/web/core/components/project/card.test.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/dropdowns/filters/access.tsx web/apps/web/core/components/project/dropdowns/filters/created-at.tsx web/apps/web/core/components/project/dropdowns/filters/lead.tsx web/apps/web/core/components/project/dropdowns/filters/member-list.tsx web/apps/web/core/components/project/dropdowns/filters/members.tsx web/apps/web/core/components/project/dropdowns/filters/root.tsx web/apps/web/core/components/project/header.tsx web/apps/web/core/components/sidebar/sidebar-wrapper.tsx web/apps/web/package.json web/packages/constants/src/filter.ts web/packages/constants/src/project.ts web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/navigation.json web/packages/i18n/src/locales/en/project.json web/packages/i18n/src/locales/en/workspace.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/navigation.json web/packages/i18n/src/locales/zh-CN/project.json web/packages/i18n/src/locales/zh-CN/workspace.json
```
```bash
git commit -m "feat(M3/closeout): the projects page, the sidebar's heading and the tab bar speak the user's language

The projects page's filters name themselves from one map, shared by the
dropdown and the bar of the filters applied, whose removing buttons are
named; the lead's filter, a copy of the members', is the members' filter
named by its caller; the members' filter offers the roles by their names
and is one button. The sidebar's and the extended sidebar's headings, the
tab bar's menus and the cards read their text through t(). S7 filters the
projects and the members in Chinese.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T12.1` | 侧边栏的标题又是英文（E3） | 故事 S7 | 端到端 |
| `T12.2` | 扩展侧边栏的标题又是英文（E3） | 故事 S7 | 端到端 |
| `T12.3` | "更多"菜单又用英文给出"Show"（E3） | `tab-navigation-overflow-menu.test.tsx` | vitest |
| `T12.4` | 标签页的菜单又用英文给出"Hide in more menu"（E3） | `tab-navigation-visible-item.test.tsx` | vitest |
| `T12.5` | 已归档项目的面包屑又是英文（E3） | 故事 S7 | 端到端 |
| `T12.6` | 卡片上的日期又是英文（E3） | `card.test.tsx` | vitest |
| `T12.7` | 卡片上成员的提示又是英文（E3） | `card.test.tsx` | vitest |
| `T12.8` | 负责人的筛选用成员的名字（E3：一个筛选，由调用方命名） | 故事 S7 | 端到端 |
| `T12.9` | 网络的搜索又按键匹配，不按页面上的名字（E3） | 故事 S7 | 端到端 |
| `T12.10` | 创建日期的选项给出键，不是名字（E3） | 故事 S7 | 端到端 |
| `T12.11` | 成员筛选里的调用者又是"You"（E3） | 故事 S7 | 端到端 |
| `T12.12` | 已选筛选的一栏又用键称呼筛选（E3） | `root.test.tsx` | vitest |
| `T12.13` | 已选的网络，它的移除按钮又没有名字（E3） | `root.test.tsx`、故事 S7 | vitest；端到端 |
| `T12.14` | 筛选匹配的项目数又是英文（E3） | 故事 S7 | 端到端 |
| `T12.15` | 成员的筛选用键给出角色（E3） | 故事 S7 | 端到端 |
| `T12.16` | 成员的筛选按钮里又有一个自己的控件（E3） | 故事 S7 | 端到端 |
| `T12.17` | 页面读的一个键没有定义（`i18n-keys`） | `tools/i18n-keys.mjs` | 静态 |

---

### Task 13: 项目设置的标题、功能的开关、自动归档、没有项目的设置页、标题和提示的区域

**Files:**
- Modify: `e2e/stories/project/p1-create-project.spec.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`e2e/stories/smoke/s7-chinese.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/page.tsx`、`web/apps/web/app/(all)/create-workspace/layout.tsx`、`web/apps/web/app/(all)/create-workspace/page.tsx`、`web/apps/web/app/(all)/onboarding/layout.tsx`、`web/apps/web/app/(all)/onboarding/page.tsx`、`web/apps/web/app/provider.tsx`、`web/apps/web/core/components/automation/auto-archive-automation.tsx`、`web/apps/web/core/components/automation/select-month-modal.test.tsx`、`web/apps/web/core/components/automation/select-month-modal.tsx`、`web/apps/web/core/components/project/member-list.tsx`、`web/apps/web/core/components/project/settings/features-list.tsx`、`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx`、`web/apps/web/core/components/settings/project/content/feature-control-item.tsx`、`web/apps/web/core/lib/fake-controls.ts`、`web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json`、`web/packages/propel/src/toast/toast.tsx`
- Delete: `web/apps/web/core/components/project/form-loader.tsx`、`web/apps/web/core/components/project/settings/helper.tsx`

**Interfaces:**（spec 2.8）
- 项目设置各页的标题经 `t()`（general、成员、四个功能页不再带"settings"、状态、标签、自动化）；没有项目时的设置页经 `workspace_projects.empty_state.no_projects.*`。`project/form-loader.tsx` 删除（包装层只在有项目时显示设置页）。
- 功能的开关：`project/settings/helper.tsx` 删除，功能列表直接用 `Switch`，`aria-label` 是功能的名字；`feature-control-item.tsx` 的 `title: string` 给开关命名。
- 自动归档：月数经 `workspace_projects.common.months_count`；自定义范围的标题、占位、单位、范围的提示、Update/Updating 经 `t()`，规则 `required: true`。
- 新建工作区、新手引导的标题经 `PageHead`（布局的 `meta` 删除）；提示的区域：`@nerve/propel` 的 `Toast` 收 `label`，`app/provider.tsx` 的 `AppToast` 给它 `t("aria_labels.toasts")`。项目成员页的搜索的占位。

**Tests:** `admin-pages.test.tsx`、`states-labels.test.tsx` 照新的标题；`use-feature-toggle.test.tsx` 看开关的名字；`select-month-modal.test.tsx` 读键。P1：功能一步的 Tab 循环按功能的名字；P3：自定义范围说 Update。S7 的第四个加项目设置的标题、成员的搜索、功能开关的提示和提示的区域、自动归档；第五个（新）：没有完成引导的人看到新手引导的标题。

- [ ] **Step 1: 设置和标题**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  const pageTitle = project?.name ? `${project.name} - Automations` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  const pageTitle = project?.name ? `${project.name} - ${t("project_settings.automations.label")}` : undefined;
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx
    ? `${project.name} settings - ${t("project_settings.features.cycles.short_title")}`
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx
    ? `${project.name} - ${t("project_settings.features.cycles.short_title")}`
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
    ? `${project.name} settings - ${t("project_settings.features.intake.short_title")}`
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
    ? `${project.name} - ${t("project_settings.features.intake.short_title")}`
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx
    ? `${project.name} settings - ${t("project_settings.features.modules.short_title")}`
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx
    ? `${project.name} - ${t("project_settings.features.modules.short_title")}`
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx
  const pageTitle = project?.name
    ? `${project.name} settings - ${t("project_settings.features.views.short_title")}`
    : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx
  const pageTitle = project?.name ? `${project.name} - ${t("project_settings.features.views.short_title")}` : undefined;
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const pageTitle = project ? `${project.name} - Labels` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const pageTitle = project ? `${project.name} - ${t("common.labels")}` : undefined;
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx
  const pageTitle = project?.name ? `${project.name} - Members` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx
  const pageTitle = project?.name ? `${project.name} - ${t("common.members")}` : undefined;
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx`（修改，6 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
import { ProjectDetailsForm } from "@/components/project/form";
import { ProjectDetailsFormLoader } from "@/components/project/form-loader";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
import { ProjectDetailsForm } from "@/components/project/form";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const { allowPermissions } = useUserPermissions();
  const { t } = useTranslation();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const pageTitle = project ? `${project.name} - General Settings` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const pageTitle = project ? `${project.name} - ${t("general_settings")}` : undefined;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
        {project ? (
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
        {project && (
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
          <ProjectDetailsForm key={project.id} project={project} workspaceSlug={workspaceSlug} isAdmin={isAdmin} />
        ) : (
          <ProjectDetailsFormLoader />
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
          <ProjectDetailsForm key={project.id} project={project} workspaceSlug={workspaceSlug} isAdmin={isAdmin} />
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const pageTitle = project ? `${project.name} - States` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const pageTitle = project ? `${project.name} - ${t("common.states")}` : undefined;
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/page.tsx`（整个文件，39 行）：

````whole web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/page.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useTheme } from "next-themes";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// assets
import ProjectDarkEmptyState from "@/app/assets/empty-state/project-settings/no-projects-dark.png?url";
import ProjectLightEmptyState from "@/app/assets/empty-state/project-settings/no-projects-light.png?url";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";

function ProjectSettingsPage() {
  // store hooks
  const { resolvedTheme } = useTheme();
  const { toggleCreateProjectModal } = useCommandPalette();
  const { t } = useTranslation();
  // derived values
  const resolvedPath = resolvedTheme === "dark" ? ProjectDarkEmptyState : ProjectLightEmptyState;
  return (
    <div className="mx-auto flex h-full max-w-[480px] flex-col items-center justify-center gap-4">
      <img src={resolvedPath} alt="" />
      <div className="text-16 font-semibold text-tertiary">{t("workspace_projects.empty_state.no_projects.title")}</div>
      <div className="text-center text-13 text-tertiary">
        {t("workspace_projects.empty_state.no_projects.description")}
      </div>
      <Button onClick={() => toggleCreateProjectModal(true)}>
        {t("workspace_projects.empty_state.no_projects.primary_button.text")}
      </Button>
    </div>
  );
}

export default observer(ProjectSettingsPage);
````

`web/apps/web/app/(all)/create-workspace/layout.tsx`（整个文件，11 行）：

````whole web/apps/web/app/(all)/create-workspace/layout.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";

export default function CreateWorkspaceLayout() {
  return <Outlet />;
}
````

`web/apps/web/app/(all)/create-workspace/page.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/create-workspace/page.tsx
import { NerveLogo } from "@/components/common/nerve-logo";
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
import { NerveLogo } from "@/components/common/nerve-logo";
import { PageHead } from "@/components/core/page-title";
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
    <AuthenticationWrapper>
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
    <AuthenticationWrapper>
      <PageHead title={t("workspace_creation.heading")} />
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
              <img
                src={WorkspaceCreationDisabled}
                className="mb-4 h-full w-full object-contain"
                alt="Workspace creation disabled"
              />
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
              <img src={WorkspaceCreationDisabled} className="mb-4 h-full w-full object-contain" alt="" />
````

`web/apps/web/app/(all)/onboarding/layout.tsx`（整个文件，11 行）：

````whole web/apps/web/app/(all)/onboarding/layout.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Outlet } from "react-router";

export default function OnboardingLayout() {
  return <Outlet />;
}
````

`web/apps/web/app/(all)/onboarding/page.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/onboarding/page.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/app/(all)/onboarding/page.tsx
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/app/(all)/onboarding/page.tsx
import { LogoSpinner } from "@/components/common/logo-spinner";
````
````new web/apps/web/app/(all)/onboarding/page.tsx
import { LogoSpinner } from "@/components/common/logo-spinner";
import { PageHead } from "@/components/core/page-title";
````
````old web/apps/web/app/(all)/onboarding/page.tsx
  const { data: user } = useUser();
````
````new web/apps/web/app/(all)/onboarding/page.tsx
  const { data: user } = useUser();
  const { t } = useTranslation();
````
````old web/apps/web/app/(all)/onboarding/page.tsx
    <AuthenticationWrapper pageType={EPageTypes.ONBOARDING}>
````
````new web/apps/web/app/(all)/onboarding/page.tsx
    <AuthenticationWrapper pageType={EPageTypes.ONBOARDING}>
      <PageHead title={t("onboarding.title")} />
````

`web/apps/web/app/provider.tsx`（修改，3 处）：

````old web/apps/web/app/provider.tsx
import { TranslationProvider } from "@nerve/i18n";
````
````new web/apps/web/app/provider.tsx
import { TranslationProvider, useTranslation } from "@nerve/i18n";
````
````old web/apps/web/app/provider.tsx
});

export interface IAppProvider {
````
````new web/apps/web/app/provider.tsx
});

/** The toasts, their region named in the user's language: the translation's provider holds it. */
function AppToast({ theme }: { theme: string | undefined }) {
  const { t } = useTranslation();
  return <Toast theme={resolveGeneralTheme(theme)} label={t("aria_labels.toasts")} />;
}

export interface IAppProvider {
````
````old web/apps/web/app/provider.tsx
          <Toast theme={resolveGeneralTheme(resolvedTheme)} />
````
````new web/apps/web/app/provider.tsx
          <AppToast theme={resolvedTheme} />
````

`web/apps/web/core/components/automation/auto-archive-automation.tsx`（修改，1 处）：

````old web/apps/web/core/components/automation/auto-archive-automation.tsx
                    label={`${project?.archive_in} ${project?.archive_in === 1 ? "month" : "months"}`}
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
                    label={t("workspace_projects.common.months_count", { months: project.archive_in })}
````

`web/apps/web/core/components/automation/select-month-modal.tsx`（修改，9 处）：

````old web/apps/web/core/components/automation/select-month-modal.tsx
import type { ProjectUpdate } from "@nerve/api-client";
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
import type { ProjectUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
  const { workspaceSlug, projectId } = useParams();
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
  const { workspaceSlug, projectId } = useParams();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
          <h3 className="text-16 leading-6 font-medium text-primary">Customize time range</h3>
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
          <h3 className="text-16 leading-6 font-medium text-primary">{t("common.customize_time_range")}</h3>
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
                  required: "Select a month between 1 and 12.",
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
                  required: true,
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
                          placeholder="Enter Months"
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
                          placeholder={t("project_settings.automations.auto-archive.months_placeholder")}
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
                    <span className="absolute top-2.5 right-8 text-13 text-secondary">Months</span>
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
                    <span className="absolute top-2.5 right-8 text-13 text-secondary">
                      {t("project_settings.automations.auto-archive.months")}
                    </span>
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
                <span className="px-1 text-13 text-danger-primary">Select a month between 1 and 12.</span>
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
                <span className="px-1 text-13 text-danger-primary">
                  {t("project_settings.automations.auto-archive.months_range")}
                </span>
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
            Cancel
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
            {t("cancel")}
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
            {isSubmitting ? "Submitting..." : "Submit"}
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
            {isSubmitting ? t("updating") : t("update")}
````

`web/apps/web/core/components/project/form-loader.tsx`（删除）：

````delete web/apps/web/core/components/project/form-loader.tsx
````

`web/apps/web/core/components/project/member-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/member-list.tsx
              placeholder="Search"
````
````new web/apps/web/core/components/project/member-list.tsx
              placeholder={t("search")}
````

`web/apps/web/core/components/project/settings/features-list.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/settings/features-list.tsx
// nerve imports
````
````new web/apps/web/core/components/project/settings/features-list.tsx
// nerve imports
import { Switch } from "@makeplane/propel/components/switch";
````
````old web/apps/web/core/components/project/settings/features-list.tsx
// local imports
import { ProjectFeatureToggle } from "./helper";
````
````new web/apps/web/core/components/project/settings/features-list.tsx
// local imports
````
````old web/apps/web/core/components/project/settings/features-list.tsx
                <ProjectFeatureToggle
                  featureItem={featureItem}
                  value={Boolean(currentProjectDetails?.[featureItem.property])}
                  handleSubmit={toggleFeature}
````
````new web/apps/web/core/components/project/settings/features-list.tsx
                <Switch
                  size="sm"
                  checked={Boolean(currentProjectDetails?.[featureItem.property])}
                  onCheckedChange={() => toggleFeature(featureItem.property)}
                  aria-label={t(featureItem.i18n_label)}
````

`web/apps/web/core/components/project/settings/helper.tsx`（删除）：

````delete web/apps/web/core/components/project/settings/helper.tsx
````

`web/apps/web/core/components/settings/project/content/feature-control-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/project/content/feature-control-item.tsx
  title: React.ReactNode;
````
````new web/apps/web/core/components/settings/project/content/feature-control-item.tsx
  /** The feature's name, which names its switch too. */
  title: string;
````
````old web/apps/web/core/components/settings/project/content/feature-control-item.tsx
        <Switch
          size="sm"
          checked={value}
          onCheckedChange={() => toggleFeature(featureProperty)}
          aria-label={typeof title === "string" ? title : "Toggle project feature"}
        />
````
````new web/apps/web/core/components/settings/project/content/feature-control-item.tsx
        <Switch size="sm" checked={value} onCheckedChange={() => toggleFeature(featureProperty)} aria-label={title} />
````

`web/packages/i18n/src/locales/en/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/accessibility.json
    "add_reaction": "Add reaction",
````
````new web/packages/i18n/src/locales/en/accessibility.json
    "add_reaction": "Add reaction",
    "toasts": "Notifications",
````

`web/packages/i18n/src/locales/en/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/common.json
  "onboarding": {
````
````new web/packages/i18n/src/locales/en/common.json
  "onboarding": {
    "title": "Onboarding",
````

`web/packages/i18n/src/locales/en/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project-settings.json
        "duration": "Auto-archive work items that are closed for"
````
````new web/packages/i18n/src/locales/en/project-settings.json
        "duration": "Auto-archive work items that are closed for",
        "months_placeholder": "Enter months",
        "months": "Months",
        "months_range": "Select a month between 1 and 12."
````

`web/packages/i18n/src/locales/zh-CN/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/accessibility.json
    "add_reaction": "添加表情回应",
````
````new web/packages/i18n/src/locales/zh-CN/accessibility.json
    "add_reaction": "添加表情回应",
    "toasts": "通知",
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "onboarding": {
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "onboarding": {
    "title": "新手引导",
````

`web/packages/i18n/src/locales/zh-CN/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project-settings.json
        "duration": "自动归档已关闭"
````
````new web/packages/i18n/src/locales/zh-CN/project-settings.json
        "duration": "自动归档已关闭",
        "months_placeholder": "输入月数",
        "months": "个月",
        "months_range": "请选择 1 到 12 之间的月数。"
````

`web/packages/propel/src/toast/toast.tsx`（修改，2 处）：

````old web/packages/propel/src/toast/toast.tsx
  theme: "light" | "dark" | "system";
````
````new web/packages/propel/src/toast/toast.tsx
  theme: "light" | "dark" | "system";
  /** The name of the toasts' region, in the user's language. */
  label: string;
````
````old web/packages/propel/src/toast/toast.tsx
        <BaseToast.Viewport data-theme={props.theme}>
````
````new web/packages/propel/src/toast/toast.tsx
        <BaseToast.Viewport data-theme={props.theme} aria-label={props.label}>
````

- [ ] **Step 2: 测试**

`e2e/stories/project/p1-create-project.spec.ts`（修改，1 处）：

````old e2e/stories/project/p1-create-project.spec.ts
    `Open project, ${"Toggle project feature, ".repeat(4)}Close`
````
````new e2e/stories/project/p1-create-project.spec.ts
    "Open project, Cycles, Modules, Views, Intake, Close"
````

`e2e/stories/project/p3-project-settings.spec.ts`（修改，2 处）：

````old e2e/stories/project/p3-project-settings.spec.ts
    page.getByRole("button", { name: "Submit" }).click()
````
````new e2e/stories/project/p3-project-settings.spec.ts
    page.getByRole("button", { name: "Update", exact: true }).click()
````
````old e2e/stories/project/p3-project-settings.spec.ts
  await expect(page.getByRole("button", { name: "Submitting..." })).toBeDisabled();
````
````new e2e/stories/project/p3-project-settings.spec.ts
  await expect(page.getByRole("button", { name: "Updating" })).toBeDisabled();
````

`e2e/stories/smoke/s7-chinese.spec.ts`（修改，5 处）：

````old e2e/stories/smoke/s7-chinese.spec.ts
import { bearer, emailFor, shownNameOf, type AuthTokens } from "../../fixtures/auth";
````
````new e2e/stories/smoke/s7-chinese.spec.ts
import { bearer, emailFor, register, shownNameOf, type AuthTokens } from "../../fixtures/auth";
````
````old e2e/stories/smoke/s7-chinese.spec.ts
test("S7 (page): an admin whose language is Chinese filters the projects and the members in Chinese", async ({
````
````new e2e/stories/smoke/s7-chinese.spec.ts
test("S7 (page): an admin whose language is Chinese filters the projects and the members, and reads a project's settings, its custom range and a toast in Chinese", async ({
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  const watch = await watchPage(page);

  // The projects page: the sidebar's heading, the card's date and its members.
  await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  const watch = await watchPage(page);

  // The project settings of a workspace with no project yet, and the create-workspace page's title.
  await page.goto(`/${slug}/settings/projects`);
  await expect(page.getByText("没有项目", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "开始您的第一个项目" })).toBeVisible();
  await page.goto("/create-workspace");
  await expect(page).toHaveTitle("创建您的工作区");

  // The projects page: the sidebar's heading, the card's date and its members.
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expect(roles).toHaveText(["管理员", "成员", "访客", "已停用"]);
  await page.keyboard.press("Escape");
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expect(roles).toHaveText(["管理员", "成员", "访客", "已停用"]);
  await page.keyboard.press("Escape");

  // A project's settings: each page's title, the members' search; a feature's switch, whose toast says it, in the
  // region of the toasts, named.
  const settings = `/${slug}/settings/projects/${web.id}`;
  await page.goto(settings);
  await expect(page).toHaveTitle("Web - 常规设置");
  await page.goto(`${settings}/members`);
  await expect(page).toHaveTitle("Web - 成员");
  await expect(page.getByPlaceholder("搜索", { exact: true })).toBeVisible();
  await page.goto(`${settings}/features/cycles`);
  await page.getByRole("switch", { name: "启用周期" }).click();
  const toasts = page.getByRole("region", { name: "通知" });
  await expect(toasts.getByText("项目功能已更新。", { exact: true })).toBeVisible();

  // The auto-archiving: its months, and its custom range, which says what it takes.
  await page.goto(`${settings}/automations`);
  await page.getByRole("switch", { name: "自动归档已关闭的工作项" }).click();
  await page.getByRole("button", { name: "1 个月", exact: true }).click();
  await page.getByRole("button", { name: "自定义时间范围" }).click();
  const months = modalDialog(page).getByPlaceholder("输入月数");
  await expect(modalDialog(page).getByText("个月", { exact: true })).toBeVisible();
  await months.fill("");
  await modalDialog(page).getByRole("button", { name: "更新", exact: true }).click();
  await expect(modalDialog(page).getByText("请选择 1 到 12 之间的月数。", { exact: true })).toBeVisible();
  await cancel(page);
````
````old e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
````
````new e2e/stories/smoke/s7-chinese.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(8).fill(EMOJI_CHECK_WARNING) });
});

test("S7 (page): one whose language is Chinese, not onboarded yet, reads the onboarding's title in Chinese", async ({
  api,
  signedInPage,
}, testInfo) => {
  const dave = await register(api, emailFor(testInfo, "dave"));
  await speakChinese(api, dave);
  const page = await signedInPage(dave);
  const watch = await watchPage(page);

  await page.goto("/onboarding");
  await expect(page).toHaveTitle("新手引导");
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx`（修改，5 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} settings - project_settings.features.cycles.short_title]switch ${web.cycle_view}`,
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} - project_settings.features.cycles.short_title]switch ${web.cycle_view}`,
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} settings - project_settings.features.intake.short_title]switch ${web.intake_view}`,
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} - project_settings.features.intake.short_title]switch ${web.intake_view}`,
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} settings - project_settings.features.modules.short_title]switch ${web.module_view}`,
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} - project_settings.features.modules.short_title]switch ${web.module_view}`,
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) =>
      `[${web.name} settings - project_settings.features.views.short_title]switch ${web.issue_views_view}`,
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} - project_settings.features.views.short_title]switch ${web.issue_views_view}`,
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} - Automations]auto-archive`,
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx
    shows: (web) => `[${web.name} - project_settings.automations.label]auto-archive`,
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`（修改，5 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
    expect(text).toBe(`[${name} - States]Backlog, In Progress and their controls`);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
    expect(text).toBe(`[${name} - common.states]Backlog, In Progress and their controls`);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
    expect(text).toBe(`[${name} - States]Backlog, In Progress`);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
    expect(text).toBe(`[${name} - common.states]Backlog, In Progress`);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
      expect(text).toBe(`[${name} - Labels][common.add_label]Bug and its controls`);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
      expect(text).toBe(`[${name} - common.labels][common.add_label]Bug and its controls`);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
    expect(text).toBe(`[${name} - Labels]Bug`);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
    expect(text).toBe(`[${name} - common.labels]Bug`);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
      `[${admin.name} - Labels][common.add_label][settings_empty_state.labels.cta_primary]`,
      `[${member.name} - Labels]`,
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
      `[${admin.name} - common.labels][common.add_label][settings_empty_state.labels.cta_primary]`,
      `[${member.name} - common.labels]`,
````

`web/apps/web/core/components/automation/select-month-modal.test.tsx`（修改，1 处）：

````old web/apps/web/core/components/automation/select-month-modal.test.tsx
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
````
````new web/apps/web/core/components/automation/select-month-modal.test.tsx
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
````

`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
  it("is what each switch of the features list flips, its own feature", () => {
    renderToStaticMarkup(<ProjectFeaturesList workspaceSlug="acme" projectId={web.id} />);
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
  it("is what each switch of the features list flips, its own feature, which names it", () => {
    renderToStaticMarkup(<ProjectFeaturesList workspaceSlug="acme" projectId={web.id} />);
    expect(shown.switches.map((toggle) => toggle["aria-label"])).toEqual(["cycles", "modules", "views", "intake"]);
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
  it("is what the switch of a feature's own page flips", () => {
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
  it("is what the switch of a feature's own page flips, the feature's name its name", () => {
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
    );
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
    );
    expect(shown.switches.map((toggle) => toggle["aria-label"])).toEqual(["Intake"]);
````

`web/apps/web/core/lib/fake-controls.ts`（修改，1 处）：

````old web/apps/web/core/lib/fake-controls.ts
/** A switch: what a flip does, and whether it can be flipped. */
type Toggle = { onCheckedChange: (checked: boolean) => unknown; disabled?: boolean };
````
````new web/apps/web/core/lib/fake-controls.ts
/** A switch: what a flip does, whether it can be flipped, and its name. */
type Toggle = { onCheckedChange: (checked: boolean) => unknown; disabled?: boolean; "aria-label"?: string };
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `71 rules, 23 exceptions, no hits.`；web 的 oxlint 295 条、`@nerve/ui` 12 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 118 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/project/p1-create-project.spec.ts e2e/stories/project/p3-project-settings.spec.ts e2e/stories/smoke/s7-chinese.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/admin-pages.test.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/page.tsx' 'web/apps/web/app/(all)/create-workspace/layout.tsx' 'web/apps/web/app/(all)/create-workspace/page.tsx' 'web/apps/web/app/(all)/onboarding/layout.tsx' 'web/apps/web/app/(all)/onboarding/page.tsx' web/apps/web/app/provider.tsx web/apps/web/core/components/automation/auto-archive-automation.tsx web/apps/web/core/components/automation/select-month-modal.test.tsx web/apps/web/core/components/automation/select-month-modal.tsx web/apps/web/core/components/project/form-loader.tsx web/apps/web/core/components/project/member-list.tsx web/apps/web/core/components/project/settings/features-list.tsx web/apps/web/core/components/project/settings/helper.tsx web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx web/apps/web/core/components/settings/project/content/feature-control-item.tsx web/apps/web/core/lib/fake-controls.ts web/packages/i18n/src/locales/en/accessibility.json web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/project-settings.json web/packages/i18n/src/locales/zh-CN/accessibility.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/project-settings.json web/packages/propel/src/toast/toast.tsx
```
```bash
git commit -m "feat(M3/closeout): the project settings' titles, the feature switches, the auto-archiving and the toasts' region speak the user's language

Each project settings page's title reads through t(); a feature's switch
is named by its feature (the wrapper that named them all alike goes); the
auto-archiving's months and its custom range, the settings with no
project, the create-workspace and onboarding titles and the toasts' region
read through t(). The form loader the project wrapper never shows goes.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T13.1` | 项目的 general 页的标题又是英文（E3） | 故事 S7 | 端到端 |
| `T13.2` | 功能页的标题又带英文的"settings"（E3） | `admin-pages.test.tsx` | vitest |
| `T13.3` | 功能列表的开关又都是同一个名字（E3） | `use-feature-toggle.test.tsx`、故事 P1 | vitest；端到端 |
| `T13.4` | 功能页的开关没有名字（E3：它的标题就是名字） | `use-feature-toggle.test.tsx`、故事 S7 | vitest；端到端 |
| `T13.5` | 自动归档的月数又是英文（E3） | 故事 S7 | 端到端 |
| `T13.6` | 自定义时间范围的占位又是英文（E3） | 故事 S7 | 端到端 |
| `T13.7` | 自定义时间范围的按钮又说"Submit"（E3：它改的是项目） | 故事 P3、故事 S7 | 端到端 |
| `T13.8` | 没有项目时的项目设置又是英文（E3） | 故事 S7 | 端到端 |
| `T13.9` | 新建工作区的页面保留布局的英文标题（E3） | 故事 S7 | 端到端 |
| `T13.10` | 新手引导保留布局的英文标题（E3） | 故事 S7 | 端到端 |
| `T13.11` | 提示区域的名字又是 Base UI 的英文（E3） | 故事 S7 | 端到端 |

---

### Task 14: 测试的整理：会话的断言、`registeredPAT`、`jobs` 的期限、HTTP 测试的常量；契约的说明

**Files:**
- Create: `e2e/fixtures/assert/session.ts`
- Modify: `api/modules/project.yaml`、`e2e/fixtures/assert/identity.ts`、`e2e/fixtures/auth.ts`、`e2e/fixtures/browser.ts`、`e2e/stories/identity/a10-onboarding-profile.spec.ts`、`e2e/stories/identity/a11-api-tokens.spec.ts`、`e2e/stories/identity/a12-deactivate.spec.ts`、`e2e/stories/identity/a13-reset-password.spec.ts`、`e2e/stories/identity/a14-session-cleanup.spec.ts`、`e2e/stories/identity/a16-set-email.spec.ts`、`e2e/stories/identity/a3-sign-in.spec.ts`、`e2e/stories/identity/a4-refresh.spec.ts`、`e2e/stories/identity/a5-refresh-reuse.spec.ts`、`e2e/stories/identity/a6-sign-out.spec.ts`、`e2e/stories/identity/a7-change-password.spec.ts`、`e2e/stories/identity/a8-update-me.spec.ts`、`e2e/stories/identity/a9-preferences.spec.ts`、`e2e/stories/project/p1-create-project.spec.ts`、`e2e/stories/project/p2-visibility.spec.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/project/p8-project-preferences.spec.ts`、`e2e/stories/project/p9-guest-profile.spec.ts`、`e2e/stories/workspace/w1-create-workspace.spec.ts`、`e2e/stories/workspace/w10-admin-create-workspace.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`、`e2e/stories/workspace/w12-reactivate-member.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`e2e/stories/workspace/w4-invite-members.spec.ts`、`e2e/stories/workspace/w5-invitation-link.spec.ts`、`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`、`e2e/stories/workspace/w7-member-management.spec.ts`、`e2e/stories/workspace/w8-navigation-preferences.spec.ts`、`e2e/stories/workspace/w9-deactivation.spec.ts`、`server/internal/modules/project/adapter/http/projects_test.go`、`server/internal/modules/project/adapter/http/update_test.go`、`server/internal/platform/jobs/jobs_test.go`、`web/apps/web/core/store/member/project/project-member.store.test.ts`

**Interfaces:**（spec 2.13）
- `e2e/fixtures/assert/session.ts`（新文件，165 行）：会话的断言从 `identity.ts` 移来，`sessionsOf(db, userId)`；`identity.ts` 485 → 329 行，`accountStateOf` 经 `sessionsOf`。`expectNoAddress` 移到 `browser.ts`。`auth.ts` 的 `registeredPAT(api, email)`：23 个故事用它。
- `server/internal/platform/jobs/jobs_test.go`：`startLimit` 2 秒；定期的运行按 `ScheduledAt` 排序，每两次之间 1 秒；取消的上界 `timeout + cancelGrace`。项目的 HTTP 测试：`identifierTakenJSON`、`nameTakenJSON`。`project-member.store.test.ts` 的资料来自 `userOf`。
- `api/modules/project.yaml`：`identifier`、`default_assignee_id`、`guest_view_all_features`、两处 `archive_in` 的说明说 nerve 对字段做什么；形状不变。

**Tests:** 没有新的测试；移动的断言照旧失败（本 Task 的变异表：`T14.6`–`T14.8` 由移动之后的断言发现）。

- [ ] **Step 1: 契约的说明**

`api/modules/project.yaml`（修改，4 处）：

````old api/modules/project.yaml
          description: Upper case; it prefixes the numbers of the project's work items.
````
````new api/modules/project.yaml
          description: Upper case; unique among the workspace's projects.
````
````old api/modules/project.yaml
          description: Whom a new work item is assigned to when nobody is given.
````
````new api/modules/project.yaml
          description: The member set as the project's default assignee; null for none.
````
````old api/modules/project.yaml
          description: Whether the project's guests see every work item, not only their own.
          type: boolean
        archive_in:
          description: After how many months a closed work item is archived, 0–12; 0 never.
          type: integer
        archived_at:
````
````new api/modules/project.yaml
          description: A setting of the project for its guests, kept as set.
          type: boolean
        archive_in:
          description: A number of months, 0–12, set for automatic archiving and kept as set; 0 for none.
          type: integer
        archived_at:
````
````old api/modules/project.yaml
          type: boolean
        archive_in:
          description: After how many months a closed work item is archived, 0–12; 0 never.
          type: integer
        logo_props:
````
````new api/modules/project.yaml
          type: boolean
        archive_in:
          description: A number of months, 0–12, set for automatic archiving and kept as set; 0 for none.
          type: integer
        logo_props:
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 通过；`git status --short` 列出 `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/project/adapter/http/gen/server.gen.go`。它们的 SHA-256 和行数是：

| SHA-256 | 行 | 文件 |
|---|---|---|
| `7a073fee2aea7bd6a58addef13bd7baa4bf540a342795b95f0c0fb958043cf74` | 2948 | `api/dist/openapi.yaml` |
| `c3b78516688a7dc8f4054e2a32da8ef649d439a1fffde68862d4b77270299c1e` | 3505 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `a6e9c6898768e8ae7e14169ce2aa19c81db4b7a5725095a16eaa5aa0eefc89be` | 3271 | `web/packages/api-client/src/schema.gen.ts` |

- [ ] **Step 3: Go 的测试**

`server/internal/modules/project/adapter/http/projects_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/projects_test.go
		`"detail":"The workspace does not exist, or you are not a member of it."}`
````
````new server/internal/modules/project/adapter/http/projects_test.go
		`"detail":"The workspace does not exist, or you are not a member of it."}`
	// identifierTakenJSON and nameTakenJSON answer a project made or changed
	// with the identifier, or the name, of another project of its workspace.
	identifierTakenJSON = `{"status":409,"code":"project.identifier_taken","title":"Conflict",` +
		`"detail":"A project of the workspace has this identifier."}`
	nameTakenJSON = `{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`
````
````old server/internal/modules/project/adapter/http/projects_test.go
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict,
			`{"status":409,"code":"project.identifier_taken","title":"Conflict","detail":"A project of the workspace has this identifier."}`},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`},
````
````new server/internal/modules/project/adapter/http/projects_test.go
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict, identifierTakenJSON},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict, nameTakenJSON},
````

`server/internal/modules/project/adapter/http/update_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/update_test.go
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict,
			`{"status":409,"code":"project.identifier_taken","title":"Conflict","detail":"A project of the workspace has this identifier."}`},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`},
````
````new server/internal/modules/project/adapter/http/update_test.go
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict, identifierTakenJSON},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict, nameTakenJSON},
````

`server/internal/platform/jobs/jobs_test.go`（修改，9 处）：

````old server/internal/platform/jobs/jobs_test.go
	"log/slog"
````
````new server/internal/platform/jobs/jobs_test.go
	"log/slog"
	"slices"
````
````old server/internal/platform/jobs/jobs_test.go
// start runs r.Start(ctx) and fails the test unless it returns at once:
// Start must not wait for River, which may not start for a long while.
````
````new server/internal/platform/jobs/jobs_test.go
// startLimit is how long start waits for Start to return. Start must not
// wait for River, which may not start for a long while (in
// TestStartWithoutADatabase it never does): the limit is the deadline that
// tells a Start that waits from one that does not, not a measure of how
// fast Start returns, which a loaded machine slows.
const startLimit = 2 * time.Second

// start runs r.Start(ctx) and fails the test unless it returns within
// startLimit.
````
````old server/internal/platform/jobs/jobs_test.go
	receive(t, returned, 100*time.Millisecond, "return from Start")
````
````new server/internal/platform/jobs/jobs_test.go
	receive(t, returned, startLimit, "return from Start")
````
````old server/internal/platform/jobs/jobs_test.go
// or at once and in either order.
````
````new server/internal/platform/jobs/jobs_test.go
// or at once and in either order: the runs worked before Stop returns,
// sorted by that time, are each the interval after the one before.
````
````old server/internal/platform/jobs/jobs_test.go
	first := receive(t, runs, 15*time.Second, "first run")
	second := receive(t, runs, 10*time.Second, "second run")
	if second.Before(first) {
		first, second = second, first
	}
	if gap := second.Sub(first); gap != time.Second {
		t.Errorf("runs scheduled at %v and %v, %s apart; want the 1s interval", first, second, gap)
	}
````
````new server/internal/platform/jobs/jobs_test.go
	scheduled := []time.Time{receive(t, runs, 15*time.Second, "first run"), receive(t, runs, 10*time.Second, "second run")}
````
````old server/internal/platform/jobs/jobs_test.go
	for len(runs) > 0 { // a run that was fetched before Stop
		<-runs
````
````new server/internal/platform/jobs/jobs_test.go
	for len(runs) > 0 { // the runs fetched before Stop
		scheduled = append(scheduled, <-runs)
	}
	slices.SortFunc(scheduled, time.Time.Compare)
	for i := 1; i < len(scheduled); i++ {
		if gap := scheduled[i].Sub(scheduled[i-1]); gap != time.Second {
			t.Errorf("runs scheduled at %v: run %d %s after the one before; want the 1s interval", scheduled, i+1, gap)
		}
````
````old server/internal/platform/jobs/jobs_test.go
			took := time.Since(stopping)

````
````new server/internal/platform/jobs/jobs_test.go
			took := time.Since(stopping)

			// the deadlines are the design's: the context is cancelled once the
			// timeout has passed, before Stop would give up on the job
````
````old server/internal/platform/jobs/jobs_test.go
			if waited := at.Sub(stopping); waited < timeout || waited > timeout+500*time.Millisecond {
				t.Errorf("the job's context was cancelled %s after Stop began, want %s", waited, timeout)
````
````new server/internal/platform/jobs/jobs_test.go
			if waited := at.Sub(stopping); waited < timeout || waited >= timeout+cancelGrace {
				t.Errorf("the job's context was cancelled %s after Stop began, want %s to %s", waited, timeout, timeout+cancelGrace)
````
````old server/internal/platform/jobs/jobs_test.go
	took := time.Since(stopping)

	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "jobs still running 1s after jobs.shutdown_timeout (1s)") ||
		took < time.Second+cancelGrace || took > time.Second+cancelGrace+500*time.Millisecond {
````
````new server/internal/platform/jobs/jobs_test.go
	took := time.Since(stopping)

	// stop's own limit is the deadline: a Stop that waited for the job would
	// not return before the test's cleanup releases it
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "jobs still running 1s after jobs.shutdown_timeout (1s)") ||
		took < time.Second+cancelGrace {
````

- [ ] **Step 4: e2e 的 fixture**

`e2e/fixtures/assert/identity.ts`（修改，6 处）：

````old e2e/fixtures/assert/identity.ts
import { expect, type Page } from "@playwright/test";
````
````new e2e/fixtures/assert/identity.ts
import { expect } from "@playwright/test";
````
````old e2e/fixtures/assert/identity.ts
import type { Database } from "../db";
````
````new e2e/fixtures/assert/identity.ts
import type { Database } from "../db";
import {
  expectAllSessionsRevoked,
  expectNewSession,
  sessionOf,
  sessionsOf,
  type SessionRow,
  type SignIn,
} from "./session";
````
````old e2e/fixtures/assert/identity.ts
// Database assertions of the identity stories, and what a page shows of an
// account. The page version and the API version of a story call the same
// function (v0 design 8.2).

const refreshTokenPrefix = "nrv_rt_";
const dayMs = 24 * 60 * 60 * 1000;

/** A refresh token's content (M2 design 3.4): nrv_rt_, then session id, generation, secret and tag in base64url. */
interface RefreshTokenParts {
  sessionId: string;
  generation: number;
  secret: Buffer;
}

function parseRefreshToken(token: string): RefreshTokenParts {
  expect(token.startsWith(refreshTokenPrefix), `${token} starts with ${refreshTokenPrefix}`).toBe(true);
  const raw = Buffer.from(token.slice(refreshTokenPrefix.length), "base64url");
  expect(raw.length, "a refresh token holds 68 bytes").toBe(68);
  const id = raw.subarray(0, 16).toString("hex");
  return {
    sessionId: `${id.slice(0, 8)}-${id.slice(8, 12)}-${id.slice(12, 16)}-${id.slice(16, 20)}-${id.slice(20)}`,
    generation: raw.readUInt32BE(16),
    secret: raw.subarray(20, 52),
  };
}

/** The generation a refresh token carries: 0 at the sign-in, one more at each refresh. */
export function generationOf(refreshToken: string): number {
  return parseRefreshToken(refreshToken).generation;
}

function secretHash(refreshToken: string): Buffer {
  return createHash("sha256").update(parseRefreshToken(refreshToken).secret).digest();
}

/** A session's row, as the session assertions read it. */
export interface SessionRow {
  id: string;
  user_id: string;
  token_hash: Buffer;
  generation: number;
  user_agent: string;
  ip: string;
  expires_at: Date;
  created_at: Date;
  last_refreshed_at: Date | null;
  revoked_at: Date | null;
  revoke_reason: string | null;
}

/** The columns of a SessionRow. */
const sessionColumns = `id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, last_refreshed_at,
       revoked_at, revoke_reason`;

/** The row of the session that refreshToken belongs to. */
export async function sessionOf(db: Database, refreshToken: string): Promise<SessionRow> {
  const rows = await db.query<SessionRow>(`SELECT ${sessionColumns} FROM auth_sessions WHERE id = $1`, [
    parseRefreshToken(refreshToken).sessionId,
  ]);
  expect(rows, "the session of the refresh token").toHaveLength(1);
  return rows[0] as SessionRow;
}

/** What a sign-up or a login sent and got back. */
export interface SignIn {
  /** The address as typed; the account holds it lowercased. */
  email: string;
  refreshToken: string;
  userAgent: string;
  ip: string;
}

/**
 * The session of s.refreshToken is new: generation 0 of the account userId,
 * holding the hash of the token's secret, with the caller's User-Agent and
 * IP, never refreshed, live, ending 30 days after it began.
 */
async function expectNewSession(db: Database, userId: string | undefined, s: SignIn): Promise<void> {
  const session = await sessionOf(db, s.refreshToken);
  expect(session.user_id).toBe(userId);
  expect(parseRefreshToken(s.refreshToken).generation).toBe(0);
  expect(session.generation).toBe(0);
  expect(session.token_hash.equals(secretHash(s.refreshToken))).toBe(true);
  expect(session.user_agent).toBe(s.userAgent);
  expect(session.ip).toBe(s.ip);
  // auth.session_ttl is 720h: the session ends 30 days after it began.
  expect(session.expires_at.getTime() - session.created_at.getTime()).toBe(30 * dayMs);
  expect(session.last_refreshed_at).toBeNull();
  expect(session.revoked_at).toBeNull();
}
````
````new e2e/fixtures/assert/identity.ts
// Database assertions of the identity stories' accounts: their rows, profiles, passwords and personal access tokens,
// their sessions' being session.ts's. The page version and the API version of a story call the same function (v0
// design 8.2).
````
````old e2e/fixtures/assert/identity.ts
  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [id])).toEqual([]);
}

/** A3: a login added a new session to the account of the address. */
export async function expectSignedIn(db: Database, s: SignIn): Promise<void> {
  const users = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [s.email.toLowerCase()]);
  expect(users).toHaveLength(1);
  await expectNewSession(db, users[0]?.id, s);
}

/**
 * A4: after `refreshes` refreshes, each with the token the last one
 * returned, the session is live at that generation, holds the hash of the
 * latest token's secret, was refreshed, and still ends at sessionEnd, the
 * refresh_token_expires_at of the login (M2 design 3.5).
 */
export async function expectRefreshed(
  db: Database,
  latest: string,
  refreshes: number,
  sessionEnd: string
): Promise<void> {
  const session = await sessionOf(db, latest);
  expect(parseRefreshToken(latest).generation).toBe(refreshes);
  expect(session.generation).toBe(refreshes);
  expect(session.token_hash.equals(secretHash(latest))).toBe(true);
  expect(session.last_refreshed_at).not.toBeNull();
  expect(session.expires_at.getTime()).toBe(new Date(sessionEnd).getTime());
  expect(session.revoked_at).toBeNull();
}

/**
 * A12, A13, A16: every session of the account userId, and it has one at
 * least, is revoked for reason.
 */
async function expectAllSessionsRevoked(
  db: Database,
  userId: string,
  reason: "deactivated" | "password_reset" | "email_changed"
): Promise<void> {
  const sessions = await db.query<{ id: string; revoked: boolean; revoke_reason: string | null }>(
    "SELECT id, revoked_at IS NOT NULL AS revoked, revoke_reason FROM auth_sessions WHERE user_id = $1",
    [userId]
  );
  expect(sessions.length).toBeGreaterThan(0);
  for (const s of sessions) {
    expect({ revoked: s.revoked, revoke_reason: s.revoke_reason }, `session ${s.id}`).toEqual({
      revoked: true,
      revoke_reason: reason,
    });
  }
}

/** A5, A6: the session of refreshToken is revoked, for reason. */
export async function expectRevoked(
  db: Database,
  refreshToken: string,
  reason: "logout" | "reuse_detected"
): Promise<void> {
  const session = await sessionOf(db, refreshToken);
  expect(session.revoked_at).not.toBeNull();
  expect(session.revoke_reason).toBe(reason);
````
````new e2e/fixtures/assert/identity.ts
  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [id])).toEqual([]);
````
````old e2e/fixtures/assert/identity.ts
  const sessions = await db.query<SessionRow>(
    `SELECT ${sessionColumns} FROM auth_sessions WHERE user_id = $1 ORDER BY created_at, id`,
    [account.id]
  );
  return { account, sessions, tokens: await tokensOf(db, account.id) };
````
````new e2e/fixtures/assert/identity.ts
  return { account, sessions: await sessionsOf(db, account.id), tokens: await tokensOf(db, account.id) };
````
````old e2e/fixtures/assert/identity.ts

/**
 * Checks that page's document, as it is now, holds none of emails: accounts' addresses that the page must not show its
 * viewer (a guest, say). where names the page in the failure's message.
 */
export async function expectNoAddress(page: Page, emails: readonly string[], where: string): Promise<void> {
  const html = await page.content();
  expect(
    emails.filter((email) => html.includes(email)),
    `the addresses on ${where}`
  ).toEqual([]);
}

````
````new e2e/fixtures/assert/identity.ts

````

`e2e/fixtures/assert/session.ts`（新文件，165 行）：

````file e2e/fixtures/assert/session.ts
import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the identity stories' sessions: a refresh token's content, a session's row, and what a
// sign-in, a refresh and a revocation leave of it (M2 design 3.4, 3.5). The page version and the API version of a story
// call the same function (v0 design 8.2).

const refreshTokenPrefix = "nrv_rt_";
const dayMs = 24 * 60 * 60 * 1000;

/** A refresh token's content (M2 design 3.4): nrv_rt_, then session id, generation, secret and tag in base64url. */
interface RefreshTokenParts {
  sessionId: string;
  generation: number;
  secret: Buffer;
}

function parseRefreshToken(token: string): RefreshTokenParts {
  expect(token.startsWith(refreshTokenPrefix), `${token} starts with ${refreshTokenPrefix}`).toBe(true);
  const raw = Buffer.from(token.slice(refreshTokenPrefix.length), "base64url");
  expect(raw.length, "a refresh token holds 68 bytes").toBe(68);
  const id = raw.subarray(0, 16).toString("hex");
  return {
    sessionId: `${id.slice(0, 8)}-${id.slice(8, 12)}-${id.slice(12, 16)}-${id.slice(16, 20)}-${id.slice(20)}`,
    generation: raw.readUInt32BE(16),
    secret: raw.subarray(20, 52),
  };
}

/** The generation a refresh token carries: 0 at the sign-in, one more at each refresh. */
export function generationOf(refreshToken: string): number {
  return parseRefreshToken(refreshToken).generation;
}

function secretHash(refreshToken: string): Buffer {
  return createHash("sha256").update(parseRefreshToken(refreshToken).secret).digest();
}

/** A session's row, as the session assertions read it. */
export interface SessionRow {
  id: string;
  user_id: string;
  token_hash: Buffer;
  generation: number;
  user_agent: string;
  ip: string;
  expires_at: Date;
  created_at: Date;
  last_refreshed_at: Date | null;
  revoked_at: Date | null;
  revoke_reason: string | null;
}

/** The columns of a SessionRow. */
const sessionColumns = `id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, last_refreshed_at,
       revoked_at, revoke_reason`;

/** The row of the session that refreshToken belongs to. */
export async function sessionOf(db: Database, refreshToken: string): Promise<SessionRow> {
  const rows = await db.query<SessionRow>(`SELECT ${sessionColumns} FROM auth_sessions WHERE id = $1`, [
    parseRefreshToken(refreshToken).sessionId,
  ]);
  expect(rows, "the session of the refresh token").toHaveLength(1);
  return rows[0] as SessionRow;
}

/** The rows of the sessions of the account userId, oldest first. */
export async function sessionsOf(db: Database, userId: string): Promise<SessionRow[]> {
  return db.query<SessionRow>(
    `SELECT ${sessionColumns} FROM auth_sessions WHERE user_id = $1 ORDER BY created_at, id`,
    [userId]
  );
}

/** What a sign-up or a login sent and got back. */
export interface SignIn {
  /** The address as typed; the account holds it lowercased. */
  email: string;
  refreshToken: string;
  userAgent: string;
  ip: string;
}

/**
 * The session of s.refreshToken is new: generation 0 of the account userId,
 * holding the hash of the token's secret, with the caller's User-Agent and
 * IP, never refreshed, live, ending 30 days after it began.
 */
export async function expectNewSession(db: Database, userId: string | undefined, s: SignIn): Promise<void> {
  const session = await sessionOf(db, s.refreshToken);
  expect(session.user_id).toBe(userId);
  expect(parseRefreshToken(s.refreshToken).generation).toBe(0);
  expect(session.generation).toBe(0);
  expect(session.token_hash.equals(secretHash(s.refreshToken))).toBe(true);
  expect(session.user_agent).toBe(s.userAgent);
  expect(session.ip).toBe(s.ip);
  // auth.session_ttl is 720h: the session ends 30 days after it began.
  expect(session.expires_at.getTime() - session.created_at.getTime()).toBe(30 * dayMs);
  expect(session.last_refreshed_at).toBeNull();
  expect(session.revoked_at).toBeNull();
}

/** A3: a login added a new session to the account of the address. */
export async function expectSignedIn(db: Database, s: SignIn): Promise<void> {
  const users = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [s.email.toLowerCase()]);
  expect(users).toHaveLength(1);
  await expectNewSession(db, users[0]?.id, s);
}

/**
 * A4: after `refreshes` refreshes, each with the token the last one
 * returned, the session is live at that generation, holds the hash of the
 * latest token's secret, was refreshed, and still ends at sessionEnd, the
 * refresh_token_expires_at of the login (M2 design 3.5).
 */
export async function expectRefreshed(
  db: Database,
  latest: string,
  refreshes: number,
  sessionEnd: string
): Promise<void> {
  const session = await sessionOf(db, latest);
  expect(parseRefreshToken(latest).generation).toBe(refreshes);
  expect(session.generation).toBe(refreshes);
  expect(session.token_hash.equals(secretHash(latest))).toBe(true);
  expect(session.last_refreshed_at).not.toBeNull();
  expect(session.expires_at.getTime()).toBe(new Date(sessionEnd).getTime());
  expect(session.revoked_at).toBeNull();
}

/**
 * A12, A13, A16: every session of the account userId, and it has one at
 * least, is revoked for reason.
 */
export async function expectAllSessionsRevoked(
  db: Database,
  userId: string,
  reason: "deactivated" | "password_reset" | "email_changed"
): Promise<void> {
  const sessions = await db.query<{ id: string; revoked: boolean; revoke_reason: string | null }>(
    "SELECT id, revoked_at IS NOT NULL AS revoked, revoke_reason FROM auth_sessions WHERE user_id = $1",
    [userId]
  );
  expect(sessions.length).toBeGreaterThan(0);
  for (const s of sessions) {
    expect({ revoked: s.revoked, revoke_reason: s.revoke_reason }, `session ${s.id}`).toEqual({
      revoked: true,
      revoke_reason: reason,
    });
  }
}

/** A5, A6: the session of refreshToken is revoked, for reason. */
export async function expectRevoked(
  db: Database,
  refreshToken: string,
  reason: "logout" | "reuse_detected"
): Promise<void> {
  const session = await sessionOf(db, refreshToken);
  expect(session.revoked_at).not.toBeNull();
  expect(session.revoke_reason).toBe(reason);
}
````

`e2e/fixtures/auth.ts`（修改，2 处）：

````old e2e/fixtures/auth.ts
}

/** Signs up the account of label in this run of the test, and returns its address, a personal access token of it and its id. */
````
````new e2e/fixtures/auth.ts
}

/** A personal access token of a new account of email: the account registered, then the token created. */
export async function registeredPAT(api: Api, email: string): Promise<ApiTokenCreated> {
  return createPAT(api, (await register(api, email)).access_token);
}

/** Signs up the account of label in this run of the test, and returns its address, a personal access token of it and its id. */
````
````old e2e/fixtures/auth.ts
  const token = (await createPAT(api, (await register(api, email)).access_token)).token;
````
````new e2e/fixtures/auth.ts
  const token = (await registeredPAT(api, email)).token;
````

`e2e/fixtures/browser.ts`（修改，1 处）：

````old e2e/fixtures/browser.ts
    .toEqual({ errors: [...(expected.errors ?? []), probe], warnings: [...(expected.warnings ?? []), probe] });
}

````
````new e2e/fixtures/browser.ts
    .toEqual({ errors: [...(expected.errors ?? []), probe], warnings: [...(expected.warnings ?? []), probe] });
}

/**
 * Checks that page's document, as it is now, holds none of emails: accounts' addresses that the page must not show its
 * viewer (a guest, say). where names the page in the failure's message.
 */
export async function expectNoAddress(page: Page, emails: readonly string[], where: string): Promise<void> {
  const html = await page.content();
  expect(
    emails.filter((email) => html.includes(email)),
    `the addresses on ${where}`
  ).toEqual([]);
}

````

- [ ] **Step 5: 故事和 vitest**

`e2e/stories/identity/a10-onboarding-profile.spec.ts`（修改，3 处）：

````old e2e/stories/identity/a10-onboarding-profile.spec.ts
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a10-onboarding-profile.spec.ts
import { bearer, emailFor, password, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a10-onboarding-profile.spec.ts
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);

  // Another step is done already, so keeping it differs from resetting it
````
````new e2e/stories/identity/a10-onboarding-profile.spec.ts
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await registeredPAT(api, email);

  // Another step is done already, so keeping it differs from resetting it
````
````old e2e/stories/identity/a10-onboarding-profile.spec.ts
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const stepsBefore = await onboardingStepsOf(db, email);
````
````new e2e/stories/identity/a10-onboarding-profile.spec.ts
  const email = emailFor(testInfo);
  const pat = await registeredPAT(api, email);
  const stepsBefore = await onboardingStepsOf(db, email);
````

`e2e/stories/identity/a11-api-tokens.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a11-api-tokens.spec.ts
import { anotherTabSignsIn, bearer, createPAT, emailFor, login, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a11-api-tokens.spec.ts
import { anotherTabSignsIn, bearer, createPAT, emailFor, login, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a11-api-tokens.spec.ts
  const other = await createPAT(api, (await register(api, emailFor(testInfo, "other"))).access_token);
````
````new e2e/stories/identity/a11-api-tokens.spec.ts
  const other = await registeredPAT(api, emailFor(testInfo, "other"));
````

`e2e/stories/identity/a12-deactivate.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a12-deactivate.spec.ts
import { bearer, createPAT, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a12-deactivate.spec.ts
import { bearer, createPAT, emailFor, login, password, recordOf, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a12-deactivate.spec.ts
  const pat = await createPAT(api, (await register(api, email)).access_token);
````
````new e2e/stories/identity/a12-deactivate.spec.ts
  const pat = await registeredPAT(api, email);
````

`e2e/stories/identity/a13-reset-password.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a13-reset-password.spec.ts
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a13-reset-password.spec.ts
import { bearer, createPAT, emailFor, login, password, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a13-reset-password.spec.ts
  const otherPAT = await createPAT(api, (await register(api, other)).access_token);
````
````new e2e/stories/identity/a13-reset-password.spec.ts
  const otherPAT = await registeredPAT(api, other);
````

`e2e/stories/identity/a14-session-cleanup.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a14-session-cleanup.spec.ts
import { sessionOf } from "../../fixtures/assert/identity";
````
````new e2e/stories/identity/a14-session-cleanup.spec.ts
import { sessionOf } from "../../fixtures/assert/session";
````

`e2e/stories/identity/a16-set-email.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a16-set-email.spec.ts
import { accountOf, accountStateOf, expectEmailChanged, sessionOf, tokensOf } from "../../fixtures/assert/identity";
````
````new e2e/stories/identity/a16-set-email.spec.ts
import { accountOf, accountStateOf, expectEmailChanged, tokensOf } from "../../fixtures/assert/identity";
import { sessionOf } from "../../fixtures/assert/session";
````

`e2e/stories/identity/a3-sign-in.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a3-sign-in.spec.ts
import { countIdentity, expectNothingAdded, expectSignedIn } from "../../fixtures/assert/identity";
````
````new e2e/stories/identity/a3-sign-in.spec.ts
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { expectSignedIn } from "../../fixtures/assert/session";
````

`e2e/stories/identity/a4-refresh.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a4-refresh.spec.ts
import { expectRefreshed, generationOf } from "../../fixtures/assert/identity";
````
````new e2e/stories/identity/a4-refresh.spec.ts
import { expectRefreshed, generationOf } from "../../fixtures/assert/session";
````

`e2e/stories/identity/a5-refresh-reuse.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a5-refresh-reuse.spec.ts
import { expectRevoked, sessionOf } from "../../fixtures/assert/identity";
````
````new e2e/stories/identity/a5-refresh-reuse.spec.ts
import { expectRevoked, sessionOf } from "../../fixtures/assert/session";
````

`e2e/stories/identity/a6-sign-out.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a6-sign-out.spec.ts
import { accountOf, expectRevoked, onboardingStepsOf, sessionOf } from "../../fixtures/assert/identity";
````
````new e2e/stories/identity/a6-sign-out.spec.ts
import { accountOf, onboardingStepsOf } from "../../fixtures/assert/identity";
import { expectRevoked, sessionOf } from "../../fixtures/assert/session";
````

`e2e/stories/identity/a7-change-password.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a7-change-password.spec.ts
import { bearer, createPAT, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a7-change-password.spec.ts
import { bearer, createPAT, emailFor, login, password, recordOf, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a7-change-password.spec.ts
  const pat = await createPAT(api, (await register(api, email)).access_token);
````
````new e2e/stories/identity/a7-change-password.spec.ts
  const pat = await registeredPAT(api, email);
````

`e2e/stories/identity/a8-update-me.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a8-update-me.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a8-update-me.spec.ts
import { bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a8-update-me.spec.ts
  const pat = await createPAT(api, (await register(api, email)).access_token);
````
````new e2e/stories/identity/a8-update-me.spec.ts
  const pat = await registeredPAT(api, email);
````

`e2e/stories/identity/a9-preferences.spec.ts`（修改，2 处）：

````old e2e/stories/identity/a9-preferences.spec.ts
import { anotherTabSignsIn, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/identity/a9-preferences.spec.ts
import { anotherTabSignsIn, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/identity/a9-preferences.spec.ts
  const pat = await createPAT(api, (await register(api, email)).access_token);
````
````new e2e/stories/identity/a9-preferences.spec.ts
  const pat = await registeredPAT(api, email);
````

`e2e/stories/project/p1-create-project.spec.ts`（修改，4 处）：

````old e2e/stories/project/p1-create-project.spec.ts
import { accountId, bearer, createPAT, emailFor, named, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p1-create-project.spec.ts
import { accountId, bearer, emailFor, named, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/project/p1-create-project.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/project/p1-create-project.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/project/p1-create-project.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/project/p1-create-project.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````
````old e2e/stories/project/p1-create-project.spec.ts
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
````
````new e2e/stories/project/p1-create-project.spec.ts
  const guest = (await registeredPAT(api, guestEmail)).token;
````

`e2e/stories/project/p2-visibility.spec.ts`（修改，4 处）：

````old e2e/stories/project/p2-visibility.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p2-visibility.spec.ts
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/project/p2-visibility.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/project/p2-visibility.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/project/p2-visibility.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/project/p2-visibility.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````
````old e2e/stories/project/p2-visibility.spec.ts
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
````
````new e2e/stories/project/p2-visibility.spec.ts
  const guest = (await registeredPAT(api, guestEmail)).token;
````

`e2e/stories/project/p3-project-settings.spec.ts`（修改，5 处）：

````old e2e/stories/project/p3-project-settings.spec.ts
import { accountId, bearer, createPAT, emailFor, newAccount, register, shownNameOf } from "../../fixtures/auth";
````
````new e2e/stories/project/p3-project-settings.spec.ts
import { accountId, bearer, emailFor, newAccount, registeredPAT, shownNameOf } from "../../fixtures/auth";
````
````old e2e/stories/project/p3-project-settings.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/project/p3-project-settings.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/project/p3-project-settings.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/project/p3-project-settings.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````
````old e2e/stories/project/p3-project-settings.spec.ts
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
````
````new e2e/stories/project/p3-project-settings.spec.ts
  const guest = (await registeredPAT(api, guestEmail)).token;
````
````old e2e/stories/project/p3-project-settings.spec.ts
  const other = (await createPAT(api, (await register(api, otherEmail)).access_token)).token;
````
````new e2e/stories/project/p3-project-settings.spec.ts
  const other = (await registeredPAT(api, otherEmail)).token;
````

`e2e/stories/project/p4-archive.spec.ts`（修改，3 处）：

````old e2e/stories/project/p4-archive.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p4-archive.spec.ts
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/project/p4-archive.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/project/p4-archive.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/project/p4-archive.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/project/p4-archive.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````

`e2e/stories/project/p8-project-preferences.spec.ts`（修改，3 处）：

````old e2e/stories/project/p8-project-preferences.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p8-project-preferences.spec.ts
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/project/p8-project-preferences.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/project/p8-project-preferences.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/project/p8-project-preferences.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/project/p8-project-preferences.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````

`e2e/stories/project/p9-guest-profile.spec.ts`（修改，2 处）：

````old e2e/stories/project/p9-guest-profile.spec.ts
import { createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectNoAddress } from "../../fixtures/assert/identity";
````
````new e2e/stories/project/p9-guest-profile.spec.ts
import { createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
````
````old e2e/stories/project/p9-guest-profile.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
````
````new e2e/stories/project/p9-guest-profile.spec.ts
import { EMOJI_CHECK_WARNING, expectNoAddress, expectQuietConsole, watchPage } from "../../fixtures/browser";
````

`e2e/stories/workspace/w1-create-workspace.spec.ts`（修改，2 处）：

````old e2e/stories/workspace/w1-create-workspace.spec.ts
import { anotherTabSignsIn, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
import { anotherTabSignsIn, bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
  const pat = (await registeredPAT(api, email)).token;
````

`e2e/stories/workspace/w10-admin-create-workspace.spec.ts`（修改，2 处）：

````old e2e/stories/workspace/w10-admin-create-workspace.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w10-admin-create-workspace.spec.ts
import { bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w10-admin-create-workspace.spec.ts
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
````
````new e2e/stories/workspace/w10-admin-create-workspace.spec.ts
  const pat = (await registeredPAT(api, email)).token;
````

`e2e/stories/workspace/w11-guest-bounds.spec.ts`（修改，2 处）：

````old e2e/stories/workspace/w11-guest-bounds.spec.ts
} from "../../fixtures/api";
import { expectNoAddress } from "../../fixtures/assert/identity";
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
} from "../../fixtures/api";
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
import { EMOJI_CHECK_WARNING, expectNoAddress, expectQuietConsole, watchPage } from "../../fixtures/browser";
````

`e2e/stories/workspace/w12-reactivate-member.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w12-reactivate-member.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w12-reactivate-member.spec.ts
import { accountId, bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w12-reactivate-member.spec.ts
  const a = (await createPAT(api, (await register(api, aEmail)).access_token)).token;
````
````new e2e/stories/workspace/w12-reactivate-member.spec.ts
  const a = (await registeredPAT(api, aEmail)).token;
````
````old e2e/stories/workspace/w12-reactivate-member.spec.ts
  const b = (await createPAT(api, (await register(api, bEmail)).access_token)).token;
````
````new e2e/stories/workspace/w12-reactivate-member.spec.ts
  const b = (await registeredPAT(api, bEmail)).token;
````
````old e2e/stories/workspace/w12-reactivate-member.spec.ts
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
````
````new e2e/stories/workspace/w12-reactivate-member.spec.ts
  const carol = (await registeredPAT(api, carolEmail)).token;
````

`e2e/stories/workspace/w2-landing.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w2-landing.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w2-landing.spec.ts
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w2-landing.spec.ts
  const alice = (await createPAT(api, (await register(api, aliceEmail)).access_token)).token;
````
````new e2e/stories/workspace/w2-landing.spec.ts
  const alice = (await registeredPAT(api, aliceEmail)).token;
````
````old e2e/stories/workspace/w2-landing.spec.ts
  const bob = (await createPAT(api, (await register(api, bobEmail)).access_token)).token;
````
````new e2e/stories/workspace/w2-landing.spec.ts
  const bob = (await registeredPAT(api, bobEmail)).token;
````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { accountId, anotherTabSignsIn, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import { accountId, anotherTabSignsIn, bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  const coAdmin = (await createPAT(api, (await register(api, coAdminEmail)).access_token)).token;
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  const coAdmin = (await registeredPAT(api, coAdminEmail)).token;
````

`e2e/stories/workspace/w4-invite-members.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w4-invite-members.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
import { bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
  const guest = (await registeredPAT(api, guestEmail)).token;
````

`e2e/stories/workspace/w5-invitation-link.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w5-invitation-link.spec.ts
import { anotherTabSignsIn, bearer, createPAT, emailFor, password, register, removeRecord } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
import {
  anotherTabSignsIn,
  bearer,
  emailFor,
  password,
  register,
  registeredPAT,
  removeRecord,
} from "../../fixtures/auth";
````
````old e2e/stories/workspace/w5-invitation-link.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/workspace/w5-invitation-link.spec.ts
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
  const carol = (await registeredPAT(api, carolEmail)).token;
````
````old e2e/stories/workspace/w5-invitation-link.spec.ts
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
  const dave = (await registeredPAT(api, daveEmail)).token;
````

`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { bearer, createPAT, emailFor, password, register, type RegisterInvitation } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { bearer, emailFor, password, registeredPAT, type RegisterInvitation } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registeredPAT(api, adminEmail)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await registeredPAT(api, daveEmail)).token;
````
````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const slug = slugFor(testInfo);
````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registeredPAT(api, adminEmail)).token;
  const slug = slugFor(testInfo);
````

`e2e/stories/workspace/w7-member-management.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w7-member-management.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w7-member-management.spec.ts
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  const admin = (await registeredPAT(api, adminEmail)).token;
````
````old e2e/stories/workspace/w7-member-management.spec.ts
        const token = (await createPAT(api, (await register(api, email)).access_token)).token;
````
````new e2e/stories/workspace/w7-member-management.spec.ts
        const token = (await registeredPAT(api, email)).token;
````

`e2e/stories/workspace/w8-navigation-preferences.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
import { bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  const pat = (await registeredPAT(api, email)).token;
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  const member = (await registeredPAT(api, memberEmail)).token;
````

`e2e/stories/workspace/w9-deactivation.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w9-deactivation.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
import { accountId, bearer, emailFor, registeredPAT } from "../../fixtures/auth";
````
````old e2e/stories/workspace/w9-deactivation.spec.ts
  const a = (await createPAT(api, (await register(api, aEmail)).access_token)).token;
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
  const a = (await registeredPAT(api, aEmail)).token;
````
````old e2e/stories/workspace/w9-deactivation.spec.ts
  const b = (await createPAT(api, (await register(api, bEmail)).access_token)).token;
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
  const b = (await registeredPAT(api, bEmail)).token;
````
````old e2e/stories/workspace/w9-deactivation.spec.ts
  const c = (await createPAT(api, (await register(api, cEmail)).access_token)).token;
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
  const c = (await registeredPAT(api, cEmail)).token;
````

`web/apps/web/core/store/member/project/project-member.store.test.ts`（修改，7 处）：

````old web/apps/web/core/store/member/project/project-member.store.test.ts
import { membershipOf } from "@/store/member/workspace/fake-members";
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
import { userOf } from "@/store/member/fake-users";
````
````old web/apps/web/core/store/member/project/project-member.store.test.ts
const adding: ProjectMembersAdd = { members: [{ member_id: "u-dee", role: 15 }] };
/** The profiles the workspace's members gave. */
const profile = (name: string) => membershipOf(name).member;
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
const adding: ProjectMembersAdd = { members: [{ member_id: "u-dee", role: 15 }] };
````
````old web/apps/web/core/store/member/project/project-member.store.test.ts
    user.data = { ...profile("ann"), ...account, created_at: "2026-09-01T09:00:00Z" };
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
    user.data = { ...userOf("ann"), ...account, created_at: "2026-09-01T09:00:00Z" };
````
````old web/apps/web/core/store/member/project/project-member.store.test.ts
  const users = Object.fromEntries(["ann", "bob", "cat", "dee"].map((name) => [`u-${name}`, profile(name)]));
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
  const users = Object.fromEntries(["ann", "bob", "cat", "dee"].map((name) => [`u-${name}`, userOf(name)]));
````
````old web/apps/web/core/store/member/project/project-member.store.test.ts
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...bob, member: profile("bob") });
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...bob, member: userOf("bob") });
````
````old web/apps/web/core/store/member/project/project-member.store.test.ts
    expect(store.getFilteredProjectMemberDetails("u-cat", web.id)).toEqual({ ...cat, member: profile("cat") });
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
    expect(store.getFilteredProjectMemberDetails("u-cat", web.id)).toEqual({ ...cat, member: userOf("cat") });
````
````old web/apps/web/core/store/member/project/project-member.store.test.ts
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...demoted, member: profile("bob") });
````
````new web/apps/web/core/store/member/project/project-member.store.test.ts
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...demoted, member: userOf("bob") });
````

- [ ] **Step 6: 运行检查**

Run: `make gen-check`
Expected: 通过，没有差异。

Run: `make lint-go`
Expected: 通过；两段 `0 issues.`。

Run: `make test`
Expected: 通过；42 个包 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `71 rules, 23 exceptions, no hits.`；web 的 oxlint 295 条、`@nerve/ui` 12 条、`@nerve/propel` 10 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make build`
Expected: 通过。

Run: `make e2e`
Expected: 118 个全部通过。

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml e2e/fixtures/assert/identity.ts e2e/fixtures/assert/session.ts e2e/fixtures/auth.ts e2e/fixtures/browser.ts e2e/stories/identity/a10-onboarding-profile.spec.ts e2e/stories/identity/a11-api-tokens.spec.ts e2e/stories/identity/a12-deactivate.spec.ts e2e/stories/identity/a13-reset-password.spec.ts e2e/stories/identity/a14-session-cleanup.spec.ts e2e/stories/identity/a16-set-email.spec.ts e2e/stories/identity/a3-sign-in.spec.ts e2e/stories/identity/a4-refresh.spec.ts e2e/stories/identity/a5-refresh-reuse.spec.ts e2e/stories/identity/a6-sign-out.spec.ts e2e/stories/identity/a7-change-password.spec.ts e2e/stories/identity/a8-update-me.spec.ts e2e/stories/identity/a9-preferences.spec.ts e2e/stories/project/p1-create-project.spec.ts e2e/stories/project/p2-visibility.spec.ts e2e/stories/project/p3-project-settings.spec.ts e2e/stories/project/p4-archive.spec.ts e2e/stories/project/p8-project-preferences.spec.ts e2e/stories/project/p9-guest-profile.spec.ts e2e/stories/workspace/w1-create-workspace.spec.ts e2e/stories/workspace/w10-admin-create-workspace.spec.ts e2e/stories/workspace/w11-guest-bounds.spec.ts e2e/stories/workspace/w12-reactivate-member.spec.ts e2e/stories/workspace/w2-landing.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts e2e/stories/workspace/w4-invite-members.spec.ts e2e/stories/workspace/w5-invitation-link.spec.ts e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts e2e/stories/workspace/w7-member-management.spec.ts e2e/stories/workspace/w8-navigation-preferences.spec.ts e2e/stories/workspace/w9-deactivation.spec.ts server/internal/modules/project/adapter/http/projects_test.go server/internal/modules/project/adapter/http/update_test.go server/internal/platform/jobs/jobs_test.go web/apps/web/core/store/member/project/project-member.store.test.ts
```
```bash
git commit -m "test(M3/closeout): the session assertions have a file of their own, stories register a PAT by one helper, the jobs tests have deadlines

The sessions' assertions move from identity.ts to session.ts, with
sessionsOf, and expectNoAddress to browser.ts; registeredPAT(api, email)
replaces 23 stories' inline register-then-createPAT. The jobs tests tell
a slow start from none by a deadline and sort the periodic runs; the
project HTTP tests share the taken-identifier and taken-name bodies. The
project contract's five sentences about work items say what nerve does
with the fields.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_closeout.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、`tools/i18n-keys.mjs`，以及 `make knip`；vitest 是 `make test-web`；Go 是 `make test` 的一个包；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T14.1` | `Start` 等 River 启动完（K15：启动的期限把两种情况分开） | Go 的 `internal/platform/jobs`（`make test`） | Go |
| `T14.2` | 运行中的任务的 context 只在 `Stop` 放弃时才取消（K15：按期限，之前就取消） | Go 的 `internal/platform/jobs`（`make test`） | Go |
| `T14.3` | 不排任何定期任务（K15：定期任务的测试照旧等它的运行） | Go 的 `internal/platform/jobs`（`make test`） | Go |
| `T14.4` | 标识已被占用的回答说的是另一件事（K16：两张表都读 `identifierTakenJSON`） | Go 的 `internal/modules/project/adapter/http`（`make test`） | Go |
| `T14.5` | 名称已被占用的回答说的是另一件事（K16：两张表都读 `nameTakenJSON`） | Go 的 `internal/modules/project/adapter/http`（`make test`） | Go |
| `T14.6` | 个人主页的卡片显示看的人的邮箱（K14：`expectNoAddress` 移到 `browser.ts` 之后照旧失败；P11 的 P6.3） | 故事 P9 | 端到端 |
| `T14.7` | 登录的会话不记 User-Agent（K14：`expectSignedIn` 移到 `session.ts` 之后照旧失败） | 故事 A3 | 端到端 |
| `T14.8` | 撤销一个账户的会话撤销了所有账户的（K14：`accountStateOf` 读 `sessionsOf`） | 故事 A12 | 端到端 |
| `T14.9` | 项目成员的详情带着另一个人的资料（K18：测试的资料来自 `userOf`） | `project-member.store.test.ts` | vitest |

---

### Task 15: 交接：M3 收到的 6 份关闭，M4–M8 的 5 份，13.2 的链接

**Files:**
- Create: `docs/v0/M4-issue-core/handoffs/M3-closeout.md`、`docs/v0/M5-files/handoffs/M3-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M3-closeout.md`、`docs/v0/M7-collaboration/handoffs/M3-closeout.md`、`docs/v0/M8-open-release/handoffs/M3-closeout.md`
- Modify: `docs/v0/M3-workspace-project/M3-design.md`、`docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`、`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`、`docs/v0/M3-workspace-project/handoffs/M1-closeout.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M1-closeout.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M5-files/handoffs/M1-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M1-closeout.md`、`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md`、`docs/v0/M7-collaboration/handoffs/M1-closeout.md`、`docs/v0/M7-collaboration/handoffs/M2-closeout.md`、`docs/v0/M8-open-release/handoffs/M1-closeout.md`

**Interfaces:**（spec 2.1、2.14、第 4 节）
- M3 收到的 6 份交接各加"处理结果（M3/收尾）"，`status: closed`。
- M4–M8 各一份 `handoffs/M3-closeout.md`（新文件）：收到 `copy-hook` 例外的 M4、M6、M7、M8 写明收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M<n>/closeout`、每个 Phase 开始时改为 `M<n>/P<k>`；收尾改到、仍有 oxlint 警告的 5 个文件各归它的 M（控制者的裁定：M4 两个、M6 一个、M7 一个、M8 页头的搜索）；层叠的规则（M4 第 7 节的元素、M8 的发布核对）；命令面板自己的文字（M8 第 3 节）。M4–M8 的 `M1-closeout.md`：`always-return` 已清零；M4、M6、M7 的 `M2-closeout.md`：复制经 hook。M3 设计 13.2：接收者链接到交接，表前一句。M1 收尾交接的处理结果写新的上限（web 303 → 295、`@nerve/ui` 15 → 12、`@nerve/propel` 11 → 10）。

**Tests:** 只改文档。`doclinks.py`（plan 的"一次性的脚本"）：本 Task 改的文档中每个相对链接都指向存在的文件。

- [ ] **Step 1: 交接**

`docs/v0/M3-workspace-project/M3-design.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/M3-design.md
### 13.2 M3 交给后续 M 的事项（收尾时写成交接）
````
````new docs/v0/M3-workspace-project/M3-design.md
### 13.2 M3 交给后续 M 的事项（收尾时写成交接）
每一行写进接收者的 `handoffs/M3-closeout.md`（收尾），连同各 Phase 的 spec 第 5 节、评审第 6 节交给它的各条和收尾找到的（[收尾 spec](specs/closeout.md) 第 2 节）。

````
````old docs/v0/M3-workspace-project/M3-design.md
| M4 | **事件**：M3 没有领域事件，删除工作区、降为访客、结束成员关系的连带经 `ProjectCascade` 同步完成（3.3）。M4 随第一个异步订阅者引入事件（M2 设计 3.15、M4 的 M2 收尾交接第 6 节不变），届时可以把删除工作区的连带改挂到 `WorkspaceDeleted` 上，并为工作项加上删除项目、删除工作区的连带。但邀请的软删除必须留在删除工作区的事务里：注册时的邀请检查（`SignupInvitations`）只看邀请行、不看工作区，连带改为异步之后，从删除到任务运行之间，已删除工作区的邀请链接会让被邀请的邮箱在注册关闭时注册；所以要么邀请这一步保持同步，要么让这个检查也读工作区。**约定**：先锁父行再判定、加锁顺序、成员关系集合的增长与收缩（3.6 约定六，以后加入新的成员关系时照它写，恢复"不比新授予给得更多，也不比原来那一行更多"）、共享锁之下的批量插入按唯一键排序（约定五）、账户行只作第一把锁（约定一）；页面按权限取数、`await` 之后先核对会话（7.1）；声明在一个模块上的跨模块错误码由这个模块的 HTTP 测试返回（9.4）；规则表和操作名在各模块、完整性测试（3.4）；权限矩阵的写法（9.2）；错误码前缀规则的修订和 `forbidden`（11.7）；规则表的 `AllowCreator` 和 `guest_view_all_features` 对工作项的约束（3.4）。**数据**：删除状态前检查它的工作项、删除标签时处理 `issue_labels`；工作项编号取 `projects.last_issue_sequence`（4.6）；工作项的筛选和显示列加在 `workspace_user_properties`、`project_user_properties`（3.18），旧 `ProjectService` 的 `/user-properties/` 两个方法和关键词例外（`until: "M4"`）；旧 `ProjectService` 的 `projectIssuesSearch` 和它的三个调用方（M4 的选父工作项、M6 的添加已有工作项、M7 的收集箱查重）随 M4 的工作项搜索接口替换，之后删除旧 service（7.3）；60 天清理包括 M3 的表，指向 `workspaces`、`projects`、`labels.parent_id` 的外键已有不带条件的索引，工作项的表照做（4.12）；工作项引用状态、标签、项目的外键在 4.12 的图上延伸。**其他**：页大小的规则移到 `shared`、游标不签名的提醒（M2 交接第 12 节原样，3.12）；决策点 3 的规则行（个人主页的工作项列表）；`issue/root.store.ts` 的 `autorun` 不在 action 之外写 `router` 的副本（计算属性，或由 `RootStore.dispose()` 释放的 `reaction`；7.1，P8a 评审记录第 6 节）；工作项的分页、分组列表扩展 `core/lib/reconciled.ts`，不另写一份（总体设计 7.7）；`workspace-draft-issues/base.ts` 的 16 行死成员；M3 页面上指向工作项页面的链接接上之后，这些页面进入 `watchPage` 的范围（3.1） |
| M5 | `workspaces.logo_asset_id`、`projects.cover_image_asset_id` 的迁移归各自模块，图标和封面的上传控件从 Plane 的源码加回，`logo_url`、`cover_image_url`、`MemberUser.avatar_url` 有真值（3.2）；P8b 删除的 28 张预设封面和随机封面随封面的上传加回（7.8，只留了显示用的默认图 `image_1.webp`）；新建项目的封面值（M5 的 M1-closeout，M3/P8b 已删掉上传预设封面的一步） |
| M6 | `ProjectAuthWrapper` 取迭代、模块（3.1，按 7.1 由权限启用）；`existing-issues-list-modal.tsx` 改用 M4 的工作项搜索（7.3）；`cycle_filter`、`module_filter` 的反应的释放（7.1）；`core/sidebar/progress-stats/` 的 8 行死成员；迭代、模块的外键在 4.12 的图上延伸 |
| M7 | 侧边栏的收藏、未读通知数、首页的"最近"小部件（连同它的工作项预览 `peek-overviews.tsx`，P8a 随小部件删除，M1 3.14）、项目的视图和分诊状态的取数、项目侧边栏的 `intake_count`（3.1）；打开 `intake_view` 时建默认收集箱、分诊状态的接口（3.17）；跨项目的标签列表（3.16、7.3）；项目的收藏（`is_favorite`、卡片上的收藏按钮、建项目之后加入收藏、`favorite.store.ts` 的项目分支、`favoriteProjectIds` getter，3.2）和归档项目时的收藏处理；收集箱查重 `select-duplicate.tsx` 改用 M4 的工作项搜索（7.3）；`workspace-notifications.ts` 的 17 行死成员；工作区视图的筛选列（3.18） |
| M8 | 工作区、项目、成员的 Webhook 事件：在 M4 引入的事件机制上加（M3 没有事件）；表情选择器的数据请求只剩核对（7.7）；README 中邀请链接与签名密钥的说明随发布核对（8.7）；性能测量加上大工作区的成员列表、项目列表（不分页的集合，11.3）：回答的大小和页面的加载时间，按测到的数字决定要不要给其中一个分页 |
````
````new docs/v0/M3-workspace-project/M3-design.md
| [M4](../M4-issue-core/handoffs/M3-closeout.md) | **事件**：M3 没有领域事件，删除工作区、降为访客、结束成员关系的连带经 `ProjectCascade` 同步完成（3.3）。M4 随第一个异步订阅者引入事件（M2 设计 3.15、M4 的 M2 收尾交接第 6 节不变），届时可以把删除工作区的连带改挂到 `WorkspaceDeleted` 上，并为工作项加上删除项目、删除工作区的连带。但邀请的软删除必须留在删除工作区的事务里：注册时的邀请检查（`SignupInvitations`）只看邀请行、不看工作区，连带改为异步之后，从删除到任务运行之间，已删除工作区的邀请链接会让被邀请的邮箱在注册关闭时注册；所以要么邀请这一步保持同步，要么让这个检查也读工作区。**约定**：先锁父行再判定、加锁顺序、成员关系集合的增长与收缩（3.6 约定六，以后加入新的成员关系时照它写，恢复"不比新授予给得更多，也不比原来那一行更多"）、共享锁之下的批量插入按唯一键排序（约定五）、账户行只作第一把锁（约定一）；页面按权限取数、`await` 之后先核对会话（7.1）；声明在一个模块上的跨模块错误码由这个模块的 HTTP 测试返回（9.4）；规则表和操作名在各模块、完整性测试（3.4）；权限矩阵的写法（9.2）；错误码前缀规则的修订和 `forbidden`（11.7）；规则表的 `AllowCreator` 和 `guest_view_all_features` 对工作项的约束（3.4）。**数据**：删除状态前检查它的工作项、删除标签时处理 `issue_labels`；工作项编号取 `projects.last_issue_sequence`（4.6）；工作项的筛选和显示列加在 `workspace_user_properties`、`project_user_properties`（3.18），旧 `ProjectService` 的 `/user-properties/` 两个方法和关键词例外（`until: "M4"`）；旧 `ProjectService` 的 `projectIssuesSearch` 和它的三个调用方（M4 的选父工作项、M6 的添加已有工作项、M7 的收集箱查重）随 M4 的工作项搜索接口替换，之后删除旧 service（7.3）；60 天清理包括 M3 的表，指向 `workspaces`、`projects`、`labels.parent_id` 的外键已有不带条件的索引，工作项的表照做（4.12）；工作项引用状态、标签、项目的外键在 4.12 的图上延伸。**其他**：页大小的规则移到 `shared`、游标不签名的提醒（M2 交接第 12 节原样，3.12）；决策点 3 的规则行（个人主页的工作项列表）；`issue/root.store.ts` 的 `autorun` 不在 action 之外写 `router` 的副本（计算属性，或由 `RootStore.dispose()` 释放的 `reaction`；7.1，P8a 评审记录第 6 节）；工作项的分页、分组列表扩展 `core/lib/reconciled.ts`，不另写一份（总体设计 7.7）；`workspace-draft-issues/base.ts` 的 16 行死成员；M3 页面上指向工作项页面的链接接上之后，这些页面进入 `watchPage` 的范围（3.1） |
| [M5](../M5-files/handoffs/M3-closeout.md) | `workspaces.logo_asset_id`、`projects.cover_image_asset_id` 的迁移归各自模块，图标和封面的上传控件从 Plane 的源码加回，`logo_url`、`cover_image_url`、`MemberUser.avatar_url` 有真值（3.2）；P8b 删除的 28 张预设封面和随机封面随封面的上传加回（7.8，只留了显示用的默认图 `image_1.webp`）；新建项目的封面值（M5 的 M1-closeout，M3/P8b 已删掉上传预设封面的一步） |
| [M6](../M6-cycles-modules/handoffs/M3-closeout.md) | `ProjectAuthWrapper` 取迭代、模块（3.1，按 7.1 由权限启用）；`existing-issues-list-modal.tsx` 改用 M4 的工作项搜索（7.3）；`cycle_filter`、`module_filter` 的反应的释放（7.1）；`core/sidebar/progress-stats/` 的 8 行死成员；迭代、模块的外键在 4.12 的图上延伸 |
| [M7](../M7-collaboration/handoffs/M3-closeout.md) | 侧边栏的收藏、未读通知数、首页的"最近"小部件（连同它的工作项预览 `peek-overviews.tsx`，P8a 随小部件删除，M1 3.14）、项目的视图和分诊状态的取数、项目侧边栏的 `intake_count`（3.1）；打开 `intake_view` 时建默认收集箱、分诊状态的接口（3.17）；跨项目的标签列表（3.16、7.3）；项目的收藏（`is_favorite`、卡片上的收藏按钮、建项目之后加入收藏、`favorite.store.ts` 的项目分支、`favoriteProjectIds` getter，3.2）和归档项目时的收藏处理；收集箱查重 `select-duplicate.tsx` 改用 M4 的工作项搜索（7.3）；`workspace-notifications.ts` 的 17 行死成员；工作区视图的筛选列（3.18） |
| [M8](../M8-open-release/handoffs/M3-closeout.md) | 工作区、项目、成员的 Webhook 事件：在 M4 引入的事件机制上加（M3 没有事件）；表情选择器的数据请求只剩核对（7.7）；README 中邀请链接与签名密钥的说明随发布核对（8.7）；性能测量加上大工作区的成员列表、项目列表（不分页的集合，11.3）：回答的大小和页面的加载时间，按测到的数字决定要不要给其中一个分页 |
````

`docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md
status: done
````
````new docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md
status: closed
````
````old docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md
来源：[M2/P3a spec](../../M2-auth/specs/P3a-account-api.md) 2.11、第 3 节第 8 条。

````
````new docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md
来源：[M2/P3a spec](../../M2-auth/specs/P3a-account-api.md) 2.11、第 3 节第 8 条。

## 处理结果（M3/收尾）

M3 的列表都是集合型的，不分页（M3 设计 3.12、11.3）：M3 的操作不引用 `Limit`、`Cursor`、`NextCursor`，它们只由 M2 的 `listApiTokens` 引用（`api/modules/identity.yaml`）；`apitest` 的组件名检查随 `make test` 通过，没有 `Name-2`。页大小的规则移到 `shared` 一事（M2 收尾交接第 12 节）原样写进 [M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 9 节。

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。

````

`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
status: open
````
````new docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
status: closed
````
````old docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
状态保持 `open`：由 M3 的收尾关闭（M3/P11 spec 第 7 节）。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
状态保持 `open`：由 M3 的收尾关闭（M3/P11 spec 第 7 节）。

## 处理结果（M3/收尾）

逐项的结论，关闭条件都在 M3 中做到：

1. **项目字段**：M3/P4a。`api/` 里没有 `close_in`、`default_state`、`page_view`、`estimate_id`，也没有 `/sidebar-preferences/`（收尾在 `49190525` 上核对：`git grep -n -E "close_in|default_state|page_view|estimate_id|sidebar-preferences" -- api` 没有输出）；项目的"自动化"只剩 `archive_in`。
2. **侧边栏偏好**：接口一侧在 M3/P2（`WorkspacePreferences`），页面一侧在 M3/P9（`ProjectNavigationDialog` 经 `useProjectNavigationPreferences()`），W8 的两个版本核对。
3. **保留的工作区地址**：M3/P1（服务端的名单三段）和 M3/P8a（`RESTRICTED_URLS` 删除，前后端一份，"应用"一段有 vitest），与 M1-P3、M1-P4 的同一项一起关闭。
4. **个人主页**：访客能看别人的个人主页（决策点 3，A），写进权限矩阵的 `listWorkspaceMembers` 一行（M3/P2，M3 设计 9.2）；页面在 M3/P11，浏览器核对 C10 通过（[M3/P11 评审](../reviews/P11-web-states-labels-and-cleanup-review.md)第 2 节）。个人主页上工作项列表的规则行交给 M4（[M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 3 节）。

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。

````

`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
status: open
````
````new docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
status: closed
````
````old docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M3/P8b spec](../specs/P8b-web-project-data.md) 第 7 节；[M3/P8a spec](../specs/P8a-web-workspace-data.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M3/P8b spec](../specs/P8b-web-project-data.md) 第 7 节；[M3/P8a spec](../specs/P8a-web-workspace-data.md) 第 7 节。

## 处理结果（M3/收尾）

逐项的结论，关闭条件都在 M3 中做到：

1. **不再读的字段**：项目的接口没有 `anchor` 和发布设置，Nerve 没有项目动态（M3/P4a；收尾在 `49190525` 上核对：`git grep -n -E "anchor|is_form_enabled" -- api` 没有输出）。视图、收集箱的接口不在 M3，由 M7 的同名交接（`docs/v0/M7-collaboration/handoffs/M1-P3-trim-platform.md`）约束。
2. **项目成员**：只有"从工作区成员中添加"一种方式（M3/P4b 的 `addProjectMembers`）；加入公开项目改调 `POST /api/v0/projects/{project_id}/join`，守卫的 `project-invitations` 例外删掉（M3/P8b），规则本身留着看住"受邀进项目"的说法。
3. **保留的工作区地址**：与后端同源，Plane 的产品词逐个有结论（全部不保留，M3 设计 3.10；M3/P1、P8a）。
4. **地址**：`project-identifiers` 等新地址都不带结尾 `/`（M3/P4a），项目设置页经项目 store 调它（M3/P8b）。

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。

````

`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
status: open
````
````new docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
status: closed
````
````old docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
来源：[M3/P10 spec](../specs/P10-web-project-pages.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
来源：[M3/P10 spec](../specs/P10-web-project-pages.md) 第 7 节。

## 处理结果（M3/收尾）

两项都已做到：

1. **保留的工作区名**：名单的"应用"一段正好是 `web/apps/web/app/routes/core.ts` 的顶层静态路由段加 `public/` 的顶层目录，前后端用同一份（`server/internal/modules/workspace/domain/reserved_slugs.txt`），vitest 守"应用"一段、`bootstrap` 的 Go 测试守"服务端"一段（M3/P1、P8a，M3 设计 3.10）。
2. **离开项目的顺序**：先等接口成功再跳转（M3/P10），vitest 和故事 P5 的页面版本核对；工作区的离开照同一顺序，M3 收尾加的侧边栏入口也经它（[M3 收尾 spec](../specs/closeout.md) 2.7）。

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。

````

`docs/v0/M3-workspace-project/handoffs/M1-closeout.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-closeout.md
status: open
````
````new docs/v0/M3-workspace-project/handoffs/M1-closeout.md
status: closed
````
````old docs/v0/M3-workspace-project/handoffs/M1-closeout.md
状态保持 `open`：P11 的评审写明剩下的行和上限之后，M3 的收尾关闭。

````
````new docs/v0/M3-workspace-project/handoffs/M1-closeout.md
状态保持 `open`：P11 的评审写明剩下的行和上限之后，M3 的收尾关闭。

## 处理结果（M3/收尾）

两节的结论：

1. **死成员和死 prop**：M3/P11 删掉了改到的文件中真正没有读者的行，`--rows M3` 中产品代码的行 80 → 54；剩下的每一行写进 [M3/P11 spec](../specs/P11-web-states-labels-and-cleanup.md) 附录 A.9（文件、名字、谁在读或传它）。收尾在它的最后一个代码 Task 之后重跑（`deadsym.mjs`、`domains.mjs --rows M3`）：218 → 225 行，产品代码的 54 行和按路径误归的 41 行不变，多出的 7 行是收尾新的测试中 `it.each` 的列和替身的 prop（收尾 spec 附录 A.5）。误归的 41 行写进各自的交接：M4 16 行（[M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 8 节），M6 8 行（[M6 的交接](../../M6-cycles-modules/handoffs/M3-closeout.md)第 2 节），M7 17 行（[M7 的交接](../../M7-collaboration/handoffs/M3-closeout.md)第 2 节）。
2. **oxlint**：M3 改到的文件在各 Phase 结束时没有警告，`promise(always-return)` 由 M3/P11 全仓清零，各包的上限照新的条数（M3/P11：web 348 → 303、editor 65 → 64）。收尾再调低上限：web 303 → 295、`@nerve/ui` 15 → 12、`@nerve/propel` 11 → 10。收尾改到的文件中仍有警告的 5 个照控制者的裁定写进交接：只为 `IconButton` 的名字或 `showTooltip` 改到的 `core/description-versions/modal.tsx`、`issues/issue-detail/issue-activity/activity-filter.tsx`（[M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 8 节）、`modules/module-card-item.tsx`（[M6 的交接](../../M6-cycles-modules/handoffs/M3-closeout.md)第 4 节）、通知的筛选菜单（[M7 的交接](../../M7-collaboration/handoffs/M3-closeout.md)第 4 节），和只改了占位文字的页头搜索 `navigation/top-nav-power-k.tsx`（[M8 的交接](../../M8-open-release/handoffs/M3-closeout.md)第 4 节）；收尾 spec 第 3 节第 2 条和附录 A.5 逐个写明。M4–M8 的五份 M1 收尾交接写明 `always-return` 已清零。

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。

````

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，2 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
status: open
````
````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
status: closed
````
````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P10 spec](../specs/P10-web-project-pages.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P10 spec](../specs/P10-web-project-pages.md) 第 7 节。

## 处理结果（M3/收尾）

十四节都有结论，各 Phase 的处理结果写在上面：

| 节 | 结论 | 在哪里 |
|---|---|---|
| 1 邀请与注册 | 完成 | 接口 M3/P3，页面 M3/P9；负责人的裁定在 M3 设计第 10 节 |
| 2 落点与新手引导的取数 | 完成 | M3/P8a、P9 |
| 3 stores 按会话分代 | 完成 | M3/P8a、P8b；`web/apps/web` 中模块级的 service 实例剩 22 处，都是 M4–M7 的 |
| 4 `last_workspace_id` 的外键 | 不加，理由在 M3 设计 3.14 | M3/P1 |
| 5 `workspace_creation_enabled` | 完成 | M3/P1 |
| 6 停用的端口 | 完成 | M3/P6 |
| 7 可空的引用字段 | 完成 | M3/P1、P2、P4a、P8a、P8b；有真值在 M5（[M5 的交接](../../M5-files/handoffs/M3-closeout.md)第 1 节） |
| 8 模块边界 | 完成 | M3/P1 |
| 9 删除关系图 | 完成 | M3/P4a（M3 设计 4.12） |
| 10 表情选择器的数据 | 完成 | M3/P10 |
| 11 M2 留下的调用和类型 | 完成；M4 的活动类型中的 `is_bot` 交给 M4 | M3/P8a、P8b、P9、P10；[M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 5 节 |
| 12 页大小的规则 | M3 没有分页的列表，原样交给 M4 | [M4 的交接](../../M4-issue-core/handoffs/M3-closeout.md)第 9 节 |
| 13 M2 走不到的页面 | 完成，浏览器核对 C4 | M3/P9 |
| 14 下拉框和复制 | 完成，浏览器核对 C7、C8；M4、M6、M7 的复制一节改为经 `useCopyLink`（收尾） | M3/P10；[M3 收尾 spec](../specs/closeout.md) 2.10 |

状态改为 `closed`。

来源：[M3 收尾 spec](../specs/closeout.md) 2.1。

````

`docs/v0/M4-issue-core/handoffs/M1-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````
````new docs/v0/M4-issue-core/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
````

`docs/v0/M4-issue-core/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M2-closeout.md
- **关闭条件**：6 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。
````
````new docs/v0/M4-issue-core/handoffs/M2-closeout.md
- **关闭条件**：6 处都改经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`，M3/P10；复制文字的是同一文件的 `useCopyText`，M3 收尾），失败时经 `t()` 提示；关键词规则 `copy-hook` 中它们的例外随之删除（M3 收尾，[M3 的交接](M3-closeout.md)）。
````

`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（新文件，94 行）：

````file docs/v0/M4-issue-core/handoffs/M3-closeout.md
---
status: open
from: M3/closeout
to: M4
created: 2026-10-13
---

# M3 交给 M4：事件和连带、加锁约定、权限规则、状态和标签的数据、工作项页面上留下的前端

M3（工作区与项目）做完了工作区、成员、邀请、项目、项目成员、状态、标签、个人显示设置、权限框架和这些页面（[M3 设计](../../M3-workspace-project/M3-design.md)）。下面是 M3 定下的约定第一次落到工作项上的地方，和 M3 改到、要由 M4 接上的代码。每一节来自 M3 设计 13.2 的 M4 一行，以及各 Phase 的 spec 第 5 节、评审第 6 节交给 M4 的各条和 M3 收尾找到的（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明每一条的来处）；第 9 节是 M2 收尾交接第 12 节的原文。为什么这样定，看链接的节；这里写做什么、代码在哪、M3 做到哪一步、怎样算完。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. 事件与删除工作区的连带

- **做什么**：M3 没有领域事件，删除工作区、降为访客、结束成员关系的连带经 `ProjectCascade` 同步完成（M3 设计 3.3）。M4 随第一个异步订阅者引入事件（M2 设计 3.15、M4 的 [M2 收尾交接](M2-closeout.md)第 6 节不变），届时可以把删除工作区的连带改挂到 `WorkspaceDeleted` 上，并为工作项加上删除项目、删除工作区的连带。
- **不能改成异步的一步**：邀请的软删除留在删除工作区的事务里。注册时的邀请检查（`SignupInvitations`）只看邀请行、不看工作区，连带改为异步之后，从删除到任务运行之间，已删除工作区的邀请链接会让被邀请的邮箱在注册关闭时注册；要么邀请这一步保持同步，要么让这个检查也读工作区（M3/P3 评审第 6 节，F-M7）。
- **由请求入队的任务**：`server/internal/bootstrap/table_rows_test.go` 的 `registrationRows` 排除 River 的表（选主、定时任务在测试期间写它们）。有了由请求入队的任务之后，"被拒的注册不写任何表"要另外核对 `river_job`，例如按任务种类或参数（M3/P3 评审第 6 节，R2-M3）。
- **关闭条件**：M4 设计写明连带同步还是异步；改成异步时，邀请一步的选择和它的测试；`registrationRows` 的核对覆盖新的任务。

## 2. 加锁的约定和方案 E

- **约定**（M3 设计 3.6、11.2，总体设计 4.2）：写事务先锁父行再判定；全局加锁顺序；成员关系集合的增长与收缩（约定六，以后加入新的成员关系时照它写，恢复"不比新授予给得更多，也不比原来那一行更多"）；共享锁之下的批量插入按唯一键排序（约定五）和它的例外：一个批量写的行若有别的写不持父行的锁也能写到，先按 `id` 锁住、再只写锁住的那些（M3/P6 的裁定 G-1；今天只有停用按地址删除的邀请）；账户行只作第一把锁（约定一）。M4 的批量写（工作项、评论等的连带）逐个照约定五核对。
- **方案 E 的范围**（M3/P4b 整分支评审的 I2，负责人 2026-10-02）：工作项的写是否最先取工作区行的 `FOR SHARE` 由 M4 的设计决定，不默认沿用约定二。取的话，新的 `FOR SHARE` 越过等待中的 `FOR NO KEY UPDATE`，高频的工作项写入可以让工作区一级的写（修改、删除工作区，改角色，接受邀请，移出、离开、恢复成员，停用）一直等到请求期限，M4 要说明怎样不被饿死，或证明在它的负载下不会；不取的话，要说明工作项一侧的行与工作区一级的连带怎样不出现 F-M3 那样的时刻倒退（M3 设计 3.3）。另算进去：先锁后判定让任何已认证的账户在得到 404 之前短暂取得一个工作区的 `FOR NO KEY UPDATE`（离开只要 slug，移出要一个成员关系的 id），方案 E 之下每一次都短暂挡住这个工作区的项目级的写，限速约束它的频率（M3/P5a、P5b 评审第 6 节）。
- **新的成员关系增长**：工作项若加入新的增长（例如指派时自动加入项目），照约定六与停用串行（项目一侧的增长取工作区的 S），并加一条与停用的交错测试（M3/P6 评审第 6 节）。
- **物理删除账户**：`workspace_members.member_id`、`project_members.member_id`、`project_user_properties.user_id` 的 `ON DELETE CASCADE` 会不经工作区的锁删除成员关系和显示设置。M4 的清理若物理删除账户，先锁工作区，或写明为什么不需要（M3/P2 评审 T11 C2、P4a 评审 F-M4）。
- **关闭条件**：M4 设计写明方案 E 的选择和代价；每个新的批量写按约定五核对；每条新的增长路径有与停用的交错测试。

## 3. 权限规则和错误码

- 规则表在 `server/internal/modules/access/domain/rules.go`，操作名常量在各模块的 `domain/actions.go`，`bootstrap` 的完整性测试核对两者一一对应（M3 设计 3.4）；每个操作在权限矩阵里有一行，写法照 M3 设计 9.2（逐格，含已不是成员的身份）。
- 错误码前缀的规则（M3 设计 11.7）：带前缀的码，前缀是 nerve 的一个模块；平台码 `forbidden` 用于"看得到而角色不允许"。声明在一个模块上的跨模块码由这个模块自己的 HTTP 测试返回过（`apitest.Main` 两个方向，M3 设计 9.4）。
- 规则表的 `AllowCreator`（"创建者本人可以修改或删除自己创建的对象"）随第一条用到它的规则加入；`guest_view_all_features` 对工作项的约束（M3 设计 3.4）。
- 决策点 3 的规则行：访客看不到别人的个人主页上的工作项列表（M3 设计 9.2 的那一格）；页面一侧（个人主页的工作项一栏和访客看到的筛选工具栏，`user-issues` 的地址今天答 404）见第 6 节。
- **关闭条件**：工作项的每个操作在规则表、操作名、矩阵中各有一行；完整性测试和 `apitest.Main` 通过。

## 4. 状态、标签和项目的数据

- **状态**：删除状态之前检查它下面的工作项（"状态下还有工作项时不能删"，M3 设计 3.17）；新工作项取项目的默认状态（`states."default"`，每个项目至多一个），那时 `State.default` 的说明补上这一句（M3/P7a spec 第 3 节第 7 条）。
- **标签**：工作项带标签（`issue_labels`，部分唯一 `(issue_id, label_id)`）；定下删除标签时工作项上的标签怎么办；`issue_labels` 的写如果不持有项目或工作区的 `FOR NO KEY UPDATE`，就落在约定五的例外里，`DeleteLabel` 的一条语句改为按 `id` 先锁；任何新的设 `parent_id` 的路径再核对两层和已删除的父标签的规则，"给已有父标签的标签换父标签"要有自己的拒绝行（M3/P7b 评审第 6 节）。
- **项目**：工作项编号取 `projects.last_issue_sequence`（`UPDATE … SET last_issue_sequence = last_issue_sequence + 1 … RETURNING`，M3 设计 4.6）。`Project` 的 `identifier`、`default_assignee_id`、`guest_view_all_features`、`archive_in` 四个字段的说明现在只说 nerve 存了什么（M3 收尾，控制者的裁定 A1）：工作项读它们之后，说明改成它们那时的作用（编号的前缀、新工作项的默认负责人、访客看到的工作项、自动归档）。
- **表和删除**：工作项引用状态、标签、项目的外键在 M3 设计 4.12 的图上延伸；60 天清理包括 M3 的表，指向 `workspaces`、`projects`、`labels.parent_id` 的外键已有不带条件的 `*_idx`，工作项的表照做。新表带 `workspace_id`，否则按目录的删除守卫拒绝它（M3/P4a 评审 F-M1）；端到端的 `workspaceTables`（`e2e/fixtures/assert/workspace.ts`）加上工作项的表，并考虑像 `expectProjectDeleted` 那样的目录自查；`expectProjectDeleted` 对有 `project_id` 而没有指向 `projects` 的外键的新表失败（有意的），两个列表保持 `COLLATE "C"`（M3/P7b 评审第 6 节）。
- **关闭条件**：M4 设计写明删除状态、标签时工作项的去向；4.12 的图延伸到工作项的表；端到端的删除断言含工作项的表。

## 5. 显示设置与旧的 service

- 工作项列表的筛选和显示列加在 `workspace_user_properties`、`project_user_properties`（M3 设计 3.18），格式由 M4 定；旧 `ProjectService`（`web/apps/web/core/services/project/project.service.ts`）的 `/user-properties/` 两个方法和关键词守卫的例外（`until: "M4"`）随之删除；今天工作项页面上 `user-properties` 的请求答 404（M3/P8b spec 2.12）。
- 旧 `ProjectService` 的 `projectIssuesSearch` 和它的三个调用方（M4 的选父工作项、M6 的添加已有工作项、M7 的收集箱查重）随 M4 的工作项搜索接口替换，之后删除旧 service（M3 设计 7.3）。旧 `WorkspaceService`（`web/apps/web/core/services/workspace.service.ts`）的视图、草稿、搜索方法同样随各自的接口改接或删除。
- 工作区搜索的类型和参数：`@nerve/types` 的 `workspace.ts` 24 行、`project/projects.ts` 9 行，`workspace.service.ts` 3 行（`domains.mjs --rows M3` 列出的 36 行，属于 M4，M3/P11 spec 附录 A.9）。
- 活动类型中的 `is_bot`（`@nerve/types` 的 `issues/activity/base.ts` 的 `TIssueActivityUserDetail`、`comments/card/display.tsx`）：Nerve 不区分人和机器人，删除（M2 收尾交接第 11 节留下的两处）。
- **关闭条件**：`/user-properties/` 的例外从 `tools/keywords.json` 删掉；旧 `ProjectService`、`WorkspaceService` 中工作项的方法都已改接或删除；`is_bot` 不再出现在 `web/` 里。

## 6. stores、会话和页面的取数（总体设计 7.7）

- `issue/root.store.ts` 的 `autorun` 不在 action 之外写 `router` 的副本：把 `workspaceSlug`、`projectId` 等做成 `rootStore.router` 上的计算属性，或改成 `reaction`，由 `RootStore.dispose()` 调用它的释放函数（在那之前 `store-context.test.ts` 有两条 MobX 严格模式的警告，M3/P8a 评审第 6 节）。工作项的分页、分组列表扩展 `core/lib/reconciled.ts`，不另写一份。
- **按需的取数按权限启用**（M3/P8b spec 第 5 节）：状态、成员、标签的下拉框打开时取（`dropdowns/state/dropdown.tsx`、`dropdowns/member/dropdown.tsx`、`issues/select/dropdown.tsx`、`issues/issue-layouts/properties/label-dropdown.tsx`、`issues/issue-detail/label/select/label-select.tsx`），子工作项的 store 取别的项目的状态、成员、标签（`store/issue/issue-details/sub_issues.store.ts`，调用者可能不是那些项目的成员）。被拒绝时：状态下拉框的加载一直不结束、是未处理的拒绝，成员下拉框是未处理的拒绝，工作项 store 不等结果的取数（`issue.store.ts` 取状态的两处、`sub_issues.store.ts` 取状态、成员、标签）是未处理的拒绝，三个标签选择器的取数被拒绝时一直"加载中"或被吞掉（Plane 原有，M3/P8b 评审的 P9）。
- 错误经文案表（`errorMessageKey`）：工作项的创建、修改读 `error?.error` 的两处（`issues/issue-modal/base.tsx`，M3/P9 spec 第 5 节）。
- 工作项页面上的标签新建（`issues/` 下的 `issue-detail/label/root.tsx`、`issue-detail/label/select/root.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`select/dropdown.tsx`、`select/base.tsx` 直接调 store 的 `createLabel`）改经 `useLabelChanges().create`（会话的核对、名称已被占用在名称下），之后可以加关键词规则（M3/P11 spec 第 3 节第 13 条）。
- 工作区一级显示的工作项不显示没有取过标签的项目的标签（M3/P8b 删了为每个项目取标签的工作区取数，M7 加回）；`fetchParentStats` 不再重取项目，工作项的修改若动了项目的字段，加一个不等的重取；工作项的键宽读回 `next_work_item_sequence`，`TIssueIdentifierProps.projectIdentifier` 在项目未到时可以是 `undefined`（M3/P8b spec 第 5 节）。
- 个人主页的工作项一栏（成员和管理员；`/api/workspaces/{slug}/user-issues/…` 今天答 404）和访客看到的筛选工具栏（布局、筛选、Display 的控件在"没有权限"之上，M3/P11 评审 C10）。
- M3 页面上指向工作项页面的链接接上之后，这些页面进入 `watchPage` 的范围（M3 设计 3.1）；S2 的 `REQUESTS` 是一份要维护的名单，在挂载路径上加取数时同时改它（M3/P11 spec 第 6 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，开着的弹窗和组件的状态不带过去。工作项的弹窗、选中、草稿等属于一个工作区或项目的界面状态放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）；要在切换之间留着的，写明为什么。
- **工作项级弹窗的宿主**（M3 收尾）：`modals/work-item-level.tsx` 的 `WorkItemLevelModals`（创建工作项 `isCreateIssueModalOpen`、删除工作项 `isDeleteIssueModalOpen`）。开关在命令面板的 store（`web/apps/web/core/store/command-palette.store.ts`）里，宿主由 `power-k/projects-app-provider.tsx` 渲染，在 `(projects)/layout.tsx` 和 `(settings)/layout.tsx` 的 `ProjectsAppPowerKProvider` 里，在项目的 key 之上：换项目（后退、前进也是）时开着的弹窗留着，作用于路由新指向的项目。M4 接上工作项页面时两件都做：把宿主移到项目的 key 之下，或在项目的范围结束时关上它的开关；并像 `modals/workspace-level.tsx` 的 `WorkspaceLevelModals` 那样在卸载时关上它渲染的每个开关（`useLayoutEffect` 的清理，在下一个页面绘制之前）。加一个像 P3、P8 那样的后退检查（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）。
- **嵌套的 id 没有 key**（M3 收尾）：工作项的页面 `projects/(detail)/[projectId]/issues/(detail)/[issueId]`、`archives/issues/(detail)/[archivedIssueId]`、`browse/[workItem]` 和个人主页 `profile/[userId]`（连同 `[profileViewId]`）只在工作区、项目的 key 之下，换一个工作项或一个人（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M4 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：上面每一处的拒绝有提示或交回调用方，没有未处理的拒绝（vitest 或故事）；`autorun` 随退役的一代释放，`store-context.test.ts` 没有严格模式的警告。

## 7. 工作项页面的键盘、下拉框和弹窗

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：弹窗打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉、取消不能用。工作项的弹窗在 M3 的故事里走不到：M4 对接时逐个核对 Tab 在弹窗里转、Escape 和点外面关上、关上之后焦点回到打开它的控件，以及弹窗里经 portal 渲染的东西（日期选择、下拉框、表情）能用（收尾 spec 附录 A.3 的门户清单只证明了 M3 页面上的几种）。`AlertModalCore` 的确认框照 A-M2：被拒绝时留着，只在成功的跟进中关上；按钮的文字经 `t()` 传入（`@nerve/ui` 的默认值是英文，见 M8 的交接）。
- **正的 `tabIndex`**：工作项的表单（`getTabIndex(ETabIndices.ISSUE_FORM)`）：浏览器先走完正的、再走焦点陷阱的哨兵，Tab 在最后一个之后离开弹窗，M3/P10 的创建表单因此删去了它的，M4 照样删去。同一个机制的另几处（M3/P10 的裁定 F-17、F-18）：`core/modals/existing-issues-list-modal.tsx`、`issues/parent-issues-list-modal.tsx` 的搜索框和 `issues/issue-detail/label/select/label-select.tsx` 收的 `baseTabIndex`；`issues/issue-modal/components/default-properties.tsx` 给父工作项的 `CustomMenu` 正的 `tabIndex`（落在 Menu 的根上）；`issues/archive-issue-modal.tsx`、`core/description-versions/modal.tsx` 写死的 `tabIndex={1}`。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个，同时删除 `getTabIndex`（`@nerve/utils` 的 `tab-indices.ts`）和 `@nerve/constants` 的 `tab-indices.ts`，`CustomMenu` 不再读 `tabIndex`（M3/P10 评审第 6 节）。
- **嵌套的按钮**：`dropdowns/` 的 `DropdownButton` 放在 Headless UI 的按钮里（M3/P10 的裁定 R3），面包屑的下拉框之前一节的标题按钮也在选择的按钮里，M4 随工作项的属性下拉框和页头一起改。菜单自己的按钮里又放了 `IconButton` 的三处（M3 收尾找到，`customButton` 或 `PopoverMenu` 的 `button` 中的 `IconButton`，两个 Tab 停留点、按钮里有按钮）：`comments/quick-actions.tsx`、`issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx`、`issues/issue-detail/issue-activity/activity-filter.tsx`（外面的按钮 `outline-none`，焦点看不到）；改法照 M3 收尾的项目一行和成员筛选（`CustomMenu` 的 `customButtonClassName` 加上按钮的样式，内容是图标，不再嵌一个按钮）。
- **下拉框**：成员下拉框以 `CustomSearchSelect` 为底之后 M4 的调用方看得到的不同（按键由 Headless UI 处理，`WhileOpen` 记下打开时的回调，Escape 之后焦点在选择自己的按钮上，列表的新样子），在 M4 的页面上核对，C7 没有核对 M4 页面上的调用方（成员下拉框的 8 个文件、4 个筛选）。`issues/select/base.tsx` 的根与 M3 收尾改过的项目下拉框是同一种写法：调用方给根 `tabIndex`，根是一个没有名字的停留点；照收尾把 `tabIndex` 给里面的按钮。筛选的共用件（`issues/issue-layouts/filters/` 的 `FilterHeader`、`FilterOption`，M3 的项目筛选也用）：标题的展开按钮没有名字，选项不说出选中与否（M3 收尾找到）。`getButtonStyling` 的两处（`rich-filters/add-filters/button.tsx`、`dropdowns/layout.tsx`）用键盘核对轮廓。
- **右键菜单**：工作项和草稿的列表的右键菜单关着时 `inert`（M3/P10 的裁定 F-14），在 M4 的页面上核对右键打开它、点它的项。
- **层叠顺序**：页面上的元素停在弹窗层（`ModalCore` 的 `z-30`）之下，`z-30` 及以上留给弹窗和从弹窗打开、经 portal 渲染到 `body` 的东西（下拉框、提示、表情）；M3 收尾因此把设置页的手机菜单从 `z-50` 降到 `z-20`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.6）。工作项页面上画在任何弹窗之上的元素（行号是 M3 收尾时的）：`issues/issue-layouts/roots/project-layout-root.tsx:80`（`fixed … z-50`）；日历的两个 `Popover.Panel`（`issues/issue-layouts/calendar/dropdowns/options-dropdown.tsx:128`、`months-dropdown.tsx:102`，`fixed z-50`，不经 portal）；`issues/issue-layouts/quick-add/form/calendar.tsx:22`（`z-50`）；更新中的遮罩 `issues/issue-layouts/calendar/issue-block.tsx:110`、`issues/issue-layouts/list/block.tsx:217`（`z-[99999]`）；拖动中的 `issues/issue-layouts/kanban/block.tsx:261`（`z-[100]`）。与弹窗同层或之上的还有 `issues/attachment/attachment-item-list.tsx:123`（`z-30`）和拖动中的 `issues/issue-layouts/kanban/kanban-delete-drop-zone.tsx:52`（`z-40`）。M4 改这些页面时把它们降到 `z-30` 之下，或开着一个弹窗核对它们不画在弹窗上。
- 共用组件的焦点轮廓（`Button`、`IconButton`、`getButtonStyling`、`CustomMenu`）是否被 `overflow-hidden` 裁掉，在 M4 的页面上用键盘核对（M3/P11 评审第 6 节）。`IconButton` 的 `aria-label` 自 M3 收尾必传，新加的每一个都有名字。
- **关闭条件**：M4 的评审写明上面每一项的浏览器核对；`getTabIndex` 在最后一个删去正的 `tabIndex` 的 M 删除；上面层叠的每个元素降到 `z-30` 之下，或有开着弹窗的核对。

## 8. 地址、复制、文案和留下的行

- **地址只从路由参数**：工作项的页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节 M4 一行列出：6 个，和裁定 F-12 补上的 16 个），M4 改成读路由参数，并把它们加进关键词规则 `project-address-route` 的范围。只读工作区的 4 个（`app/(all)/[workspaceSlug]/(projects)/drafts/header.tsx`、`issues/workspace-draft/empty-state.tsx`、`issues/workspace-draft/root.tsx`、`power-k/config/creation/command.ts`）改用 `useRouteWorkspace()`（`core/hooks/use-route-workspace.ts`）或路由的 slug，并加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）。
- **复制只经 hook**：工作项页面的 10 个文件（`issues/` 下的 `create-issue-toast-action-items.tsx`、`issue-detail-widgets/relations/helper.tsx`、`issue-detail-widgets/sub-issues/helper.ts`、`issue-detail/identifier-text.tsx`、`issue-detail/issue-activity/helper.tsx`、`issue-detail/issue-detail-quick-actions.tsx`、`issue-detail/links/link-item.tsx`、`issue-layouts/quick-action-dropdowns/helper.tsx`、`peek-overview/header.tsx`，和 `power-k/ui/pages/context-based/work-item/commands.ts`）改经 `useCopyLink` 或 `useCopyText`（`core/hooks/use-copy-link.ts`），失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 10 个例外（`until: "M4/closeout"`）随之删除。评论链接和工作项链接的复制今天没有处理拒绝（M3/P11 spec 附录 A.6）。
- **oxlint**：`issues/issue-detail/issue-activity/helper.tsx` 的 2 条（`preserve-caught-error`、`exhaustive-deps`），M4 改它时清零。收尾只为 `IconButton` 的名字改到、仍有警告的两个（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条）：`core/description-versions/modal.tsx` 1 条（`tabindex-no-positive`，第 7 节写死的 `tabIndex={1}`）、`issues/issue-detail/issue-activity/activity-filter.tsx` 2 条（`click-events-have-key-events`、`no-static-element-interactions`，第 7 节嵌套的按钮），M4 改它们时清零。`power-k/ui/pages/context-based/work-item/commands.ts` 418 行，M4 改写时拆开。两个工作项的删除确认框的 `onSubmit` 必传，改写时保持。
- **死成员**：`@nerve/types` 的 `workspace-draft-issues/base.ts` 的 16 行（`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **共用件**：`@nerve/ui` 下拉框的 `any`（`dropdowns/helper.tsx` 7 处、`dropdowns/custom-search-select.tsx` 2 处），改成泛型要改 13 个调用方，多数在 M4–M7 的页面上（M3/P10 spec 第 5 节）：M4 改它们时一起改。成员 store 的 `projectMemberIds` getter 在 M4 的读者是 `issues/issue-layouts/utils.tsx` 的两处，M7 的是视图、收集箱的筛选；M4、M7 中最后改完读者的那一个删除它（M3/P10 spec 第 3 节，裁定 F-2）。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M4/closeout`，`copy-hook` 的 10 个例外（`until: "M4/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M4/P<k>`。第 5 节 `/user-properties/` 的例外写的是 `until: "M4"`，`phase` 改为 `M4/P1` 时就到期：M4 的第一个 Phase 改筛选，或由 M4 的设计把它的 `until` 定到改筛选的那个 Phase。
- **关闭条件**：`project-address-route`、`workspace-address-route` 的范围含上面的文件，`copy-hook` 没有 `M4/closeout` 的例外，`--rows M3` 中 `workspace-draft-issues/base.ts` 的行在 M4 的评审中有结论。

## 9. 页大小的规则移到 `shared`（M2 收尾交接第 12 节，原样）

- **做什么**：`limit` 的 1–100、默认 50（`api/common.yaml` 的 `Limit`）现在在 `identity/domain`（`server/internal/modules/identity/domain/api_token.go:128-140` 的 `PageSize`），因为 M2 只有 PAT 列表一个使用者（M2/P3a spec 第 3 节第 8 条）。第二个分页列表出现时移到 `internal/shared`，两个列表共用。
- **同一处的提醒**：游标不签名（总体设计 3.4），改成另一个合格的位置照样可用。列表只能让游标决定从哪里接着读；游标内容会影响能看到什么的列表，要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：第二个分页列表合并时，页大小的规则在 `shared`，两个列表都用它；M3 没有分页列表时，M3 的 review 写明，把本节写进下一个 M 的交接。

M3 的列表都是集合型的，不分页（M3 设计 3.12、11.3），所以本节原样交给 M4（M3/P1、P2 评审第 6 节）。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M4 一行；M3 各 Phase 的 spec 第 5 节和评审第 6 节中交给 M4 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
````

`docs/v0/M5-files/handoffs/M1-closeout.md`（修改，1 处）：

````old docs/v0/M5-files/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````
````new docs/v0/M5-files/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
````

`docs/v0/M5-files/handoffs/M3-closeout.md`（新文件，32 行）：

````file docs/v0/M5-files/handoffs/M3-closeout.md
---
status: open
from: M3/closeout
to: M5
created: 2026-10-13
---

# M3 交给 M5：工作区图标、项目封面和头像，`organization_size` 的取值，附件的确认框

M3（工作区与项目）把工作区图标、项目封面和成员头像留成可为 `null` 的引用字段，等 M5 的文件存储（[M3 设计](../../M3-workspace-project/M3-design.md) 3.2）。下面每一节来自 M3 设计 13.2 的 M5 一行，和 M3 各 Phase、M3 收尾交给 M5 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。

## 1. 图标、封面和头像

- **做什么**：`workspaces.logo_asset_id`、`projects.cover_image_asset_id` 的迁移归各自的模块（`workspace`、`project`，文件名按总体设计 5.6）；工作区图标和项目封面的上传控件从 Plane 的源码加回；接口的 `Workspace.logo_url`、`Project.cover_image_url`、`MemberUser.avatar_url` 有真值（M3 设计 3.2；今天它们在接口中必有、总是 `null`）。
- **预设封面**：M3/P8b 删掉了上传预设封面的一步、随机封面和 28 张预设封面，只留显示用的默认图 `image_1.webp`（M3 设计 7.8）。它们随封面的上传加回，新建项目时的封面值照 M5 的 [M1 收尾交接](M1-closeout.md)"新建项目时的封面值"做。
- **关闭条件**：两个迁移和差异清单二·按表的两行；三个字段有真值的接口版本和页面版本的故事；预设封面和随机封面的取舍写进 M5 的设计。

## 2. `Workspace.organization_size` 收紧到数据库的取值

- **做什么**：数据库的 CHECK 只收前端的六个选项（`Just myself`、`2-10`、`11-50`、`51-200`、`201-500`、`500+`，`server/migrations/sql/00006_workspace_workspaces.sql`），请求（`WorkspaceCreate`、`WorkspaceUpdate`）引用同样六个值的枚举 `OrganizationSize`，回答的 `Workspace.organization_size` 却仍是可为 `null` 的任意字符串（`api/modules/workspace.yaml`）。把回答也改为引用 `OrganizationSize`（可为 `null`），前端读生成的类型。
- **为什么在 M5**：M3/P8a 修复轮的搁置项 P11 定为"下一个改 `api/modules/workspace.yaml` 的人做"，M5 加工作区图标时第一个改它。这是契约的改动（回答的类型变窄；库里的值都在六个值里）；负责人另有裁定时照裁定。
- **关闭条件**：`Workspace.organization_size` 引用 `OrganizationSize`，`make gen-check` 没有差异，读它的前端代码用生成的类型。

## 3. 附件的确认框和弹窗

- M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：弹窗打开时后面的页面 `inert`、`aria-hidden`；`AlertModalCore` 的确认框在请求路上时关不掉、取消不能用（M3/P11 spec 第 3 节第 2 条）。附件的删除确认框随之改变，M5 的页面能运行时核对：被拒绝时照 A-M2 留着、只在成功的跟进中关上，按钮的文字经 `t()` 传入（`@nerve/ui` 的默认值是英文，见 M8 的交接）。
- 共用组件的焦点轮廓在 M5 的页面上是否被 `overflow-hidden` 裁掉，用键盘核对（M3/P11 评审第 6 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，开着的弹窗和组件的状态不带过去。附件的确认框、上传的弹窗和进度属于一个工作区或项目，放在这些布局之下，不要放到它们之上。
- **嵌套的 id 没有 key**（M3 收尾）：附件的确认框和上传的弹窗在工作项的页面上（`[issueId]`、`browse/[workItem]`），随 M4 给这些页面的 key 结束（M4 的交接第 6 节）；M5 自己加的、在一个 id 之下的页面同样以那个 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：M5 的评审写明附件的确认框和上传弹窗的浏览器核对。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M5 一行；M3/P8a、P8b 的 spec 第 5 节，P8a 修复轮的搁置项 P11，M3/P11 评审第 6 节；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
````

`docs/v0/M6-cycles-modules/handoffs/M1-closeout.md`（修改，1 处）：

````old docs/v0/M6-cycles-modules/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````
````new docs/v0/M6-cycles-modules/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
````

`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M6-cycles-modules/handoffs/M2-closeout.md
- **关闭条件**：3 处都处理失败，经 `t()` 提示，像 `web/apps/web/core/components/api-token/modal/generated-token-details.tsx`。
````
````new docs/v0/M6-cycles-modules/handoffs/M2-closeout.md
- **关闭条件**：3 处都改经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`，M3/P10；复制文字的是同一文件的 `useCopyText`，M3 收尾），失败时经 `t()` 提示；关键词规则 `copy-hook` 中它们的例外随之删除（M3 收尾，[M3 的交接](M3-closeout.md)）。
````

`docs/v0/M6-cycles-modules/handoffs/M3-closeout.md`（新文件，47 行）：

````file docs/v0/M6-cycles-modules/handoffs/M3-closeout.md
---
status: open
from: M3/closeout
to: M6
created: 2026-10-13
---

# M3 交给 M6：项目包装层取迭代和模块、筛选的反应、迭代和模块页面上留下的前端

M3（工作区与项目）的项目包装层不再取迭代、模块，迭代和模块的页面在 M3 里走不到（[M3 设计](../../M3-workspace-project/M3-design.md) 3.1）。下面每一节来自 M3 设计 13.2 的 M6 一行，和 M3 各 Phase、M3 收尾交给 M6 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. 取数和 stores（总体设计 7.7）

- `ProjectAuthWrapper`（`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`）取迭代、模块，照 M3 设计 7.1 由权限启用（项目的有效成员才取）；`ModulesStore.fetchModulesSlim` 照新接口加回（M3/P8a spec 第 5 节）。
- `cycle_filter`、`module_filter` 跟随 `router` 的反应由 `RootStore.dispose()` 随退役的一代释放（M3 设计 7.1）；在那之前，退役的模块筛选的反应在每次换路由时仍写 localStorage（M3/P8a 评审第 6 节）。
- 迭代的 `user-properties` 不在关键词规则（旧 `ProjectService` 的 `/user-properties/`）的模式里，不命中它的样例（M3/P8b spec 第 5 节）。
- `core/modals/existing-issues-list-modal.tsx`（添加已有工作项）改用 M4 的工作项搜索接口（M3 设计 7.3）。
- 迭代、模块的弹窗读 `error?.error` 的三处（`cycles/transfer-issues-modal.tsx`、`modules/modal.tsx` 两处）改经文案表（`errorMessageKey`）；视图的修改之后的跟进经 `followInSession`（M3/P9 spec 第 5 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，开着的弹窗和组件的状态不带过去。迭代、模块、视图的弹窗和选中放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）。
- **项目级弹窗的宿主**（M3 收尾）：`modals/project-level.tsx` 的 `ProjectLevelModals`（创建迭代 `isCreateCycleModalOpen`、模块 `isCreateModuleModalOpen`、视图 `isCreateViewModalOpen`）。开关在命令面板的 store（`web/apps/web/core/store/command-palette.store.ts`）里，宿主由 `power-k/projects-app-provider.tsx` 渲染，在 `(projects)/layout.tsx` 和 `(settings)/layout.tsx` 的 `ProjectsAppPowerKProvider` 里，在项目的 key 之上：换项目（后退、前进也是）时开着的弹窗留着，作用于路由新指向的项目。M6 接上迭代、模块、视图的页面时两件都做：把宿主移到项目的 key 之下，或在项目的范围结束时关上它的开关；并像 `modals/workspace-level.tsx` 的 `WorkspaceLevelModals` 那样在卸载时关上它渲染的每个开关（`useLayoutEffect` 的清理，在下一个页面绘制之前）。加一个像 P3、P8 那样的后退检查（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）。
- **嵌套的 id 没有 key**（M3 收尾）：迭代、模块的页面 `cycles/(detail)/[cycleId]`、`modules/(detail)/[moduleId]`只在工作区、项目的 key 之下，换一个迭代或模块（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M6 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：包装层的取数有按权限的 vitest；`dispose()` 释放两个反应，`store-context.test.ts` 核对换代之后它们不再运行。

## 2. 数据

- 迭代、模块的外键在 M3 设计 4.12 的删除关系图上延伸。
- `core/sidebar/progress-stats/` 的 8 行死成员（`label.tsx` 5 行、`state_group.tsx` 3 行，`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **关闭条件**：4.12 的图延伸到迭代、模块的表；这 8 行在 M6 的评审中有结论。

## 3. 键盘、弹窗和下拉框

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉。迭代、模块、视图的弹窗在 M3 的故事里走不到：对接时逐个核对 Tab 在弹窗里转、Escape 和点外面、关上之后的焦点，以及弹窗里经 portal 渲染的日期选择、下拉框能用（收尾 spec 附录 A.3 只证明了 M3 页面上的几种）。`AlertModalCore` 的确认框照 A-M2：被拒绝时留着，只在成功的跟进中关上；按钮的文字经 `t()` 传入。
- **正的 `tabIndex`**：迭代、模块、视图的表单（`PROJECT_CYCLE`、`PROJECT_MODULE`、`PROJECT_VIEW`）同 M4 的交接第 7 节：Tab 离开它们的弹窗，照样删去；迭代、模块的归档弹窗的归档按钮（`cycles/archived-cycles/modal.tsx`、`modules/archived-modules/modal.tsx`）是 `tabIndex={1}`（M3/P10 的裁定 F-18）。模块的状态 `ModuleStatusSelect`（`modules/select/status.tsx`）给 `CustomSelect` 的 `tabIndex` 落在它的按钮上（裁定 F-13），删去模块表单的正的 `tabIndex` 之后，它不再收 `tabIndex`，`CustomSelect` 不再读它（裁定 F-16）。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`。
- **嵌套的按钮**：菜单自己的按钮里又放了 `IconButton` 的两处（M3 收尾找到）：`cycles/quick-actions.tsx`、`modules/quick-actions.tsx`；改法照 M3 收尾的项目一行（`CustomMenu` 的 `customButtonClassName`，内容是图标，不再嵌一个按钮）。
- **右键菜单**：迭代、模块、视图的列表的右键菜单关着时 `inert`（裁定 F-14），在 M6 的页面上核对右键打开它、点它的项。
- **下拉框**：C7 没有核对 M6 页面上的 `CustomSearchSelect` 调用方（成员下拉框的 2 个文件，迭代、模块、视图的页头），对接时用键盘核对。共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉，在 M6 的页面上核对。
- **关闭条件**：M6 的评审写明上面每一项的浏览器核对。

## 4. 地址、复制和 oxlint

- **地址只从路由参数**：迭代、模块、视图的页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节 M6 一行列出：11 个，和裁定 F-12 补上的 11 个），改成读路由参数，并加进关键词规则 `project-address-route` 的范围。
- **复制只经 hook**：`cycles/quick-actions.tsx`、`modules/links/list-item.tsx`、`modules/quick-actions.tsx`、`power-k/ui/pages/context-based/cycle/commands.ts`、`power-k/ui/pages/context-based/module/commands.tsx` 改经 `useCopyLink`（`core/hooks/use-copy-link.ts`），失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 5 个例外（`until: "M6/closeout"`）随之删除。两个 quick-actions 今天把复制交给菜单的 `action`（`() => void`），丢下 promise，拒绝没有处理（M3/P11 spec 附录 A.6）。
- **oxlint**：`cycles/modal.tsx` 1 条（`no-shadow`）、`modules/module-card-item.tsx` 5 条（M3/P11 spec 第 5 节），M6 改它们时清零。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M6/closeout`，`copy-hook` 的 5 个例外（`until: "M6/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M6/P<k>`。`analytics` 规则对 `core/services/cycle.service.ts` 的例外写的是 `until: "M6"`，`phase` 改为 `M6/P1` 时就到期（M1 设计 3.4）。
- **关闭条件**：`project-address-route` 的范围含这些文件，`copy-hook` 没有 `M6/closeout` 的例外。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M6 一行；M3/P8a、P8b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M6 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
````

`docs/v0/M7-collaboration/handoffs/M1-closeout.md`（修改，1 处）：

````old docs/v0/M7-collaboration/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````
````new docs/v0/M7-collaboration/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
````

`docs/v0/M7-collaboration/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M7-collaboration/handoffs/M2-closeout.md
- **关闭条件**：4 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。
````
````new docs/v0/M7-collaboration/handoffs/M2-closeout.md
- **关闭条件**：4 处都改经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`，M3/P10；复制文字的是同一文件的 `useCopyText`，M3 收尾），失败时经 `t()` 提示；关键词规则 `copy-hook` 中它们的例外随之删除（M3 收尾，[M3 的交接](M3-closeout.md)）。
````

`docs/v0/M7-collaboration/handoffs/M3-closeout.md`（新文件，49 行）：

````file docs/v0/M7-collaboration/handoffs/M3-closeout.md
---
status: open
from: M3/closeout
to: M7
created: 2026-10-13
---

# M3 交给 M7：收藏、通知、"最近"、视图和收集箱的取数，跨项目的标签，这些页面上留下的前端

M3（工作区与项目）的页面挂载时不再取 M7 的数据（收藏、未读通知数、首页的"最近"、视图、分诊状态），工作区级的标签删除，这些页面在 M3 里走不到（[M3 设计](../../M3-workspace-project/M3-design.md) 3.1）。下面每一节来自 M3 设计 13.2 的 M7 一行，和 M3 各 Phase、M3 收尾交给 M7 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. 接口和挂载时的取数

- 侧边栏的收藏、未读通知数、首页的"最近"小部件（连同它的工作项预览 `peek-overviews.tsx`，M3/P8a 随小部件删除，M1 设计 3.14）、项目的视图和分诊状态的取数、项目侧边栏的 `intake_count`（侧边栏的数字和收集箱 store 写它的两段），随各自的接口加回，照 M3 设计 7.1 由权限启用（M3 设计 3.1；M3/P8a、P8b spec 第 5 节）。
- 打开 `intake_view` 时建默认收集箱、分诊状态的接口（M3 设计 3.17）：分诊状态由 `"group" = 'triage'` 识别，每个项目至多一个（`states_project_id_triage_key`），它的名称也占用项目内的状态名（M3/P7a spec 第 3 节第 6 条）；收集箱的分诊状态的前端（store、下拉框、类型、`@nerve/propel` 的两个图标）在 M3/P8b 的 Task 7 删除，随接口加回。
- 跨项目的标签列表是各项目列表的并集，没有工作区级的标签（M3 设计 3.16、11.5）；工作区级标签的前端（store 的四个成员、筛选的标签一项、按标签分组）在 M3/P8b 删除，随它加回，连同工作区一级显示的工作项（所有工作项、视图、草稿、通知的预览）上的标签（M3/P8b spec 第 3 节第 21 条）。
- 项目的收藏：`is_favorite`、卡片上的收藏按钮、建项目之后加入收藏、`favorite.store.ts` 的项目分支、`favoriteProjectIds` getter（M3 设计 3.2），和归档项目时的收藏处理。
- 收集箱查重 `inbox/modals/select-duplicate.tsx` 改用 M4 的工作项搜索（M3 设计 7.3）；工作区视图的筛选列（M3 设计 3.18）。
- **关闭条件**：每一项的接口版本和页面版本的故事；挂载时的取数按权限启用，S2 的清单随之改（M3/P8b 的裁定 D5）。

## 2. stores 和会话（总体设计 7.7）

- 读工作区列表而没有取数的三处：新手引导的 `hasWorkspaces`、个人设置侧边栏的工作区、power-K 在工作区页面之外打开工作区的菜单；直接打开 `/settings/profile/*` 时三处都没有工作区。创建成功之后再取列表被拒绝，被当成"创建失败"，重试时遇到"slug 已占用"，根在 `createWorkspace` 只写进已经取到的列表（M3/P8a 评审第 6 节）。
- 已挂载的工作区包装层换 slug 时不重取列表，在别的标签页加入的工作区要到下一次挂载才显示；要改，改 `useWorkspaceFetch` 这一个判断（M3/P9 spec 第 5 节，P19）。
- `ProjectViewStore.loader` 和 `fetchedMap` 没有写者了：视图的取数设置它们，否则删除它们（M3/P8a 评审的 P30）。通知页的收集箱项经原来的 `useSWR` 读项目（`workspace-notifications/root.tsx`），写项目自己的读，可以盖过包装层的读：照 7.1 改成按权限的取数（M3/P8b 的 PF-L3）。收藏的 `workspace/sidebar/favorites/favorite-items/common/helper.tsx` 的 `console.error` 改成提示（M3/P9 spec 第 5 节）。
- `@nerve/types` 的 `workspace-notifications.ts` 的 17 行死成员（`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，开着的弹窗和组件的状态不带过去。收集箱、视图、通知、收藏的弹窗和选中放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）。
- **嵌套的 id 没有 key**（M3 收尾）：视图的页面 `views/(detail)/[viewId]`、`workspace-views/[globalViewId]`只在工作区、项目的 key 之下，换一个视图（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M7 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：上面每一处有 vitest 或故事；这 17 行在 M7 的评审中有结论。

## 3. 键盘、弹窗和下拉框

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉。收集箱、视图、通知的弹窗（通知的打盹 `workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx` 在内）对接时逐个核对 Tab、Escape、点外面、关上之后的焦点和弹窗里经 portal 渲染的东西（收尾 spec 附录 A.3 只证明了 M3 页面上的几种）；`AlertModalCore` 的确认框照 A-M2，按钮的文字经 `t()` 传入。
- **正的 `tabIndex`**：收集箱的创建表单（`INTAKE_ISSUE_FORM`）同 M4 的交接第 7 节，其中给父工作项的 `CustomMenu` 的一处（`inbox/modals/create-modal/issue-properties.tsx`）落在 Menu 的根上（M3/P10 的裁定 F-17）。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`。
- **嵌套的按钮和看不到的轮廓**（M3/P11 spec 第 5 节、M3 收尾找到）：菜单自己的按钮里又放了 `IconButton` 的三处（`views/quick-actions.tsx`、`workspace-notifications/sidebar/filters/menu/root.tsx`、`workspace-notifications/sidebar/header/options/menu-option/root.tsx`）；收藏的菜单 `workspace/sidebar/favorites/favorites-menu.tsx` 的 `IconButton` 外层 `opacity-0`、没有 `focus-within`，焦点看不到；收藏夹的外层手柄按钮（`workspace/sidebar/favorites/favorite-folder.tsx`）显示时是一个没有名字、按 Enter 什么都不做的停留点。改法照 M3 收尾的项目一行（`CustomMenu` 的 `customButtonClassName`；行的 `focus-within:opacity-100`；手柄不是停留点）。页头的通知链接（`navigation/top-navigation-root.tsx`）没有名字，提示是写死的"Inbox"：随未读数一起给它经 `t()` 的名字。
- **下拉框**：收集箱的两个指派人的成员下拉框（`inbox/content/issue-properties.tsx`、`inbox/modals/create-modal/issue-properties.tsx`）以 `CustomSearchSelect` 为底之后看得到的不同同 M4 的交接第 7 节，在 M7 的页面上核对（C7 没有核对）。右键菜单关着时 `inert`，在 M7 的页面上核对右键打开。共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉，在 M7 的页面上核对。
- **关闭条件**：M7 的评审写明上面每一项的浏览器核对。

## 4. 地址、复制、文案和其余

- **地址只从路由参数**：收集箱的页面从 router store 读地址的 4 个文件（`app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx`、`inbox/content/issue-properties.tsx`、`inbox/sidebar/root.tsx`、`projects/settings/intake/header.tsx`）改成读路由参数，加进关键词规则 `project-address-route` 的范围（M3/P10 spec 第 5 节）。只读工作区的 8 个（`app/(all)/[workspaceSlug]/(projects)/notifications/page.tsx`、`workspace-views/[globalViewId]/page.tsx`、`workspace-views/page.tsx`，`issues/issue-layouts/empty-states/global-view.tsx`、`workspace-notifications/root.tsx`、`workspace/views/quick-action.tsx`，`web/apps/web/core/store/favorite.store.ts`、`global-view.store.ts`）改用 `useRouteWorkspace()` 或路由的 slug（store 的由调用者传入），加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）。
- **复制只经 hook**：`inbox/content/inbox-issue-header.tsx`、`views/quick-actions.tsx`、`workspace/views/default-view-quick-action.tsx`、`workspace/views/quick-action.tsx` 改经 `useCopyLink`，失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 4 个例外（`until: "M7/closeout"`）随之删除。默认视图的复制今天没有处理拒绝（M3/P11 spec 附录 A.6）。
- **文案**：收集箱的页头（面包屑的"Intake"）、收藏、视图的页面仍有写死的英文（M3 收尾的扫描），对接时经 `t()`。
- **oxlint**：通知打盹的弹窗（第 3 节）5 条，M7 改它时清零。通知的筛选菜单（`workspace-notifications/sidebar/filters/menu/root.tsx`）1 条（`no-map-spread`）：收尾只为 `IconButton` 的名字改到它（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条），M7 改它时清零。
- **共用的 getter**：成员 store 的 `projectMemberIds` 在 M7 的读者是视图的两处和收集箱的筛选，M4 的是 `issues/issue-layouts/utils.tsx` 的两处；M4、M7 中最后改完读者的那一个删除它（M3/P10 spec 第 3 节，裁定 F-2）。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M7/closeout`，`copy-hook` 的 4 个例外（`until: "M7/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M7/P<k>`。
- **关闭条件**：两条地址规则的范围含这些文件，`copy-hook` 没有 `M7/closeout` 的例外。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M7 一行；M3/P7a、P7b、P8a、P8b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M7 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
````

`docs/v0/M8-open-release/handoffs/M1-closeout.md`（修改，1 处）：

````old docs/v0/M8-open-release/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节）；
````
````new docs/v0/M8-open-release/handoffs/M1-closeout.md
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`；`eslint(no-unneeded-ternary)` 已由 M2/P5 全仓清零，见 [M2/P5 评审](../../M2-auth/reviews/P5-web-account-review.md)第 2 节；`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零，见 [M3/P11 spec](../../M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md) 2.9）；
````

`docs/v0/M8-open-release/handoffs/M3-closeout.md`（新文件，53 行）：

````file docs/v0/M8-open-release/handoffs/M3-closeout.md
---
status: open
from: M3/closeout
to: M8
created: 2026-10-13
---

# M3 交给 M8：Webhook 的事件和页面、发布前的核对、全应用的文案和可访问性

M3（工作区与项目）没有领域事件，Webhook 的页面在 M3 里走不到；M3 和它的收尾在全应用的范围上留下几件发布之前要定的事（[M3 设计](../../M3-workspace-project/M3-design.md)）。下面每一节来自 M3 设计 13.2 的 M8 一行，和 M3 各 Phase、M3 收尾交给 M8 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. Webhook

- **事件**：工作区、项目、成员的 Webhook 事件加在 M4 引入的事件机制上（M3 没有事件，M3 设计 3.3）。
- **页面**：webhooks 读 `error?.error` 的三处（`web-hooks/create-webhook-modal.tsx`、`web-hooks/form/secret-key.tsx`、`app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/[webhookId]/page.tsx`）改经文案表（`errorMessageKey`，M3/P9 spec 第 5 节）；密钥的复制（`web-hooks/form/secret-key.tsx`）改经 `useCopyText`（`core/hooks/use-copy-link.ts`），关键词规则 `copy-hook` 的这个例外（`until: "M8/closeout"`）随之删除；只读工作区的 3 个文件（webhooks 的两个页面和 `secret-key.tsx`）改用 `useRouteWorkspace()` 或路由的 slug，加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）；页面上写死的英文经 `t()`；`web/apps/web/core/store/workspace/webhook.store.ts` 的 1 条警告（`no-accumulating-spread`）清零。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M8/closeout`，`copy-hook` 的这个例外（`until: "M8/closeout"`）那时到期，守卫要求它已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M8/P<k>`。
- **关闭条件**：Webhook 的事件有接口版本的故事；`copy-hook` 没有例外，`workspace-address-route` 的范围含这 3 个文件。

## 2. 发布之前的核对

- 表情选择器的数据从本站提供（M3/P10 已完成，M3 设计 7.7）：发布之前只剩核对（M8 的 [M1 收尾交接](M1-closeout.md)中同一项）。
- README 中邀请链接与签名密钥的说明（M3 设计 8.7）、数据库用 UTF-8 的 `LC_CTYPE` 初始化（标签名不分大小写按数据库的 locale 折叠，M3/P7b spec 第 7 节；M3 收尾已写进 README 的"部署"一节）随发布的部署文件核对。
- **层叠顺序**：全应用核对一遍页面上的元素停在弹窗层（`ModalCore` 的 `z-30`）之下：每类页面开着一个弹窗（命令面板就行），手机和桌面的宽度下都没有页面元素画在它上面。M3 收尾把设置页的手机菜单从 `z-50` 降到 `z-20`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.6），M4 的交接第 7 节列出工作项页面上 `z-30` 及以上的元素；从弹窗打开、经 portal 渲染到 `body` 的下拉框、提示、表情在弹窗之上，靠的是 DOM 的先后，一并核对。
- **性能**：大工作区的成员列表、项目列表（不分页的集合，M3 设计 11.3）的回答大小和页面的加载时间，按测到的数字决定要不要给其中一个分页。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，开着的弹窗和组件的状态不带过去。Webhook 的弹窗同样，`settings/(workspace)/webhooks/[webhookId]` 的页面以它的 id 给子树 key（加一个后退的检查）；发布之前全应用核对没有属于一个工作区或项目的弹窗、选中放在这些布局之上。
- **关闭条件**：M8 的发布核对清单含这几项和它们的结果。

## 3. 全应用的文案

M3 和它的收尾把 M3 页面上的英文都经了 `t()`，下面这几处是全应用的，一次定：

- **日期**：首页的日期（"Friday, Oct 9"）、日期筛选的弹窗（`core/filters/date-filter-modal.tsx` 的"After:""Before:"）、项目筛选中自定义日期的标签，都按英文格式化；日期的本地化是一个全应用的决定（M3/P9 评审的浏览器核对，M3 收尾）。
- `app/not-found.tsx` 的英文；`@nerve/propel` 的英文默认值（`toast/toast.tsx` 的"Loading..."）。
- 命令面板自己的文字：`power-k/menus/empty-state.tsx` 的默认值"No results found"，快捷键的标记里键与键之间的"then"（`power-k/ui/modal/command-item-shortcut-badge.tsx`）。M3 收尾只改了它整个重写的两个弹窗：快捷键列表的标题和搜索框经 `t()`，切换账户的那句话是一个带 `{email}` 的键（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.6）。
- `@nerve/ui` 的 `AlertModalCore`（`modals/alert-modal.tsx`）的按钮默认值是英文，`@nerve/ui` 不依赖 i18n：M3 的两个删除确认框和 M2 的令牌确认框传入经 `t()` 的文字，M4–M7 的调用方由各 M 照做（各自的交接）；最后一个调用方改完之后，把这几个 prop 改为必传、删去默认值（M3 收尾的建议，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节）。
- **关闭条件**：日期的决定写进 M8 的设计并实现；`web/` 的页面上没有写死的英文（M8 的扫描）。

## 4. 可访问性

- Base UI 的 `Toast.Close`（`@nerve/propel` 的 `toast/toast.tsx`）渲染成 `aria-hidden="true"` 而 `tabindex="0"`：Tab 能到、读屏读不到（M3 收尾找到）。
- 页头（`navigation/top-navigation-root.tsx`，每个工作区页面都有）：帮助菜单（`workspace/sidebar/help-section/root.tsx`）和账户菜单（`workspace/sidebar/user-menu-root.tsx`）的按钮里又是按钮，帮助菜单的没有名字；改法照 M3 收尾的成员筛选（`CustomMenu` 的 `customButtonClassName`，内容不再是按钮）。页头的通知链接在 M7 的交接。
- 页头的搜索（`navigation/top-nav-power-k.tsx`）的 4 条 oxlint 警告：`exhaustive-deps` 2 条，`click-events-have-key-events`、`prefer-tag-over-role` 各 1 条。M3 收尾只把它的占位文字改经 `t()`（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条）；后两条是它的点击区不是按钮，与上面帮助、账户菜单的按钮是同一类改法、同在页头，一起改，改时清零。
- 焦点和输入法只在 Chromium 中逐个核对过：发布之前在 Firefox、Safari 中看一遍焦点的轮廓、弹窗的 Tab 循环和输入法组字时的 Escape、Enter（M3/P11 评审第 6 节、M3/P10 spec 第 5 节；M3 收尾的 C11 看过的部分写在它的评审里）。
- 命令面板打开之后约 200 毫秒之内输入的字会被清空（`power-k/ui/modal/wrapper.tsx` 打开时的重置，Plane 原有；之后的输入留着，M3 收尾的探查）。
- **客户端导航之后的焦点**（M3 收尾）：工作区的内容以 slug 为 key 之后，经菜单换工作区时焦点落到 `body`，像一次页面的加载（原来回到菜单的按钮；控制者裁定接受，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 附录 A.8）。全应用的客户端导航之后焦点去哪里（`body`、页面的标题、主区域）是一个全应用的决定：发布之前定下并实现，读屏核对换页之后读出什么。
- **关闭条件**：M8 的评审写明每一项的修法或结论。

## 5. 嵌入的文件加 `ETag`

- `server/internal/platform/webui` 提供的嵌入文件没有 `ETag`，浏览器每次都整个下载（M3/P10 spec 第 5 节，表情的数据最大）。发布的性能核对时决定加不加（M3 收尾的建议）。
- **关闭条件**：加的话，`webui` 的测试核对 `ETag` 和 304；不加的话写明理由。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M8 一行；M3/P7b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M8 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2、3 节。
````

- [ ] **Step 2: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `71 rules, 23 exceptions, no hits.`。

Run: `python3 $M3TMP/closeout-plan/doclinks.py . docs/v0/M3-workspace-project/M3-design.md docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md docs/v0/M3-workspace-project/handoffs/M1-closeout.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M1-closeout.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/M5-files/handoffs/M1-closeout.md docs/v0/M5-files/handoffs/M3-closeout.md docs/v0/M6-cycles-modules/handoffs/M1-closeout.md docs/v0/M6-cycles-modules/handoffs/M2-closeout.md docs/v0/M6-cycles-modules/handoffs/M3-closeout.md docs/v0/M7-collaboration/handoffs/M1-closeout.md docs/v0/M7-collaboration/handoffs/M2-closeout.md docs/v0/M7-collaboration/handoffs/M3-closeout.md docs/v0/M8-open-release/handoffs/M1-closeout.md docs/v0/M8-open-release/handoffs/M3-closeout.md`
Expected: 退出码 0；20 个文件每个一行 `0 broken`，最后一行 `broken: 0`。

- [ ] **Step 3: 提交**

```bash
git add docs/v0/M3-workspace-project/M3-design.md docs/v0/M3-workspace-project/handoffs/M0-P3-pagination-components.md docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md docs/v0/M3-workspace-project/handoffs/M1-closeout.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M1-closeout.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/M5-files/handoffs/M1-closeout.md docs/v0/M5-files/handoffs/M3-closeout.md docs/v0/M6-cycles-modules/handoffs/M1-closeout.md docs/v0/M6-cycles-modules/handoffs/M2-closeout.md docs/v0/M6-cycles-modules/handoffs/M3-closeout.md docs/v0/M7-collaboration/handoffs/M1-closeout.md docs/v0/M7-collaboration/handoffs/M2-closeout.md docs/v0/M7-collaboration/handoffs/M3-closeout.md docs/v0/M8-open-release/handoffs/M1-closeout.md docs/v0/M8-open-release/handoffs/M3-closeout.md
```
```bash
git commit -m "docs(M3/closeout): the handoffs M3 received are closed, and M3 hands its rest to M4-M8

Each of the six handoffs M3 received says what came of each item, with
its evidence, and is closed. M4-M8 each get M3-closeout.md: M3 design
13.2's rows, the phases' items for them and what the closeout found, each
with its close condition. M4-M8's M1 closeout handoffs say always-return
is at zero; the M2 closeout handoffs' copying goes through the hooks.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异（链接和路径由 `links.py`、`shortpaths.py` 核对，spec 附录 A.4）。

---

### Task 16: 上级文档：README、差异清单、M3 设计的 B1、前端改动清单；第 14 节的证据

**Files:**
- Modify: `README.md`、`docs/v0/M3-workspace-project/M3-design.md`、`docs/v0/frontend-changes.md`、`docs/v0/plane-diff.md`

**Interfaces:**（spec 2.2、2.14、2.15、附录 A.8）
- README："部署"加数据库的 locale；两处"（页面：P9；此前用接口）"改为"（或经接口）"。差异清单：邀请链接一行的 Phase 说法。M3 设计 5.1：裁定 B1。前端改动清单：3.1 的 M3 一行、输入法和 P11 键盘两行，3.2 加收尾的 7 行。

**Tests:** 只改文档（同 Task 15 的核对）。

- [ ] **Step 1: 文档**

`README.md`（修改，3 处）：

````old README.md
- **邀请链接**：工作区管理员在成员页（页面：P9；此前用接口）邀请邮箱，复制每份邀请的链接交给对方（nerve 不发邮件）；对方用这个邮箱的账户登录后凭链接接受。链接里的令牌由签名密钥派生（M3 设计 3.8），数据库里不存：
````
````new README.md
- **邀请链接**：工作区管理员在成员页（或经接口）邀请邮箱，复制每份邀请的链接交给对方（nerve 不发邮件）；对方用这个邮箱的账户登录后凭链接接受。链接里的令牌由签名密钥派生（M3 设计 3.8），数据库里不存：
````
````old README.md
  - 换签名密钥之后，待接受的链接全部失效，管理员在成员页（页面：P9；此前用接口）重新复制即可；dev、test 没有配置密钥时，nerve 重启也让链接失效。
````
````new README.md
  - 换签名密钥之后，待接受的链接全部失效，管理员在成员页（或经接口）重新复制即可；dev、test 没有配置密钥时，nerve 重启也让链接失效。
````
````old README.md
  - 少了 River 的权限时 `/readyz` 仍是 200，River 的任务（会话清理）和索引重建因 `permission denied`（42501）失败，只记在日志里；少了业务表的权限时接口请求失败；少了 `goose_db_version` 的读时 `/readyz` 是 503。`server/internal/bootstrap/runtime_role_test.go` 用恰好这些权限的角色运行 nerve（就绪、会话清理、索引重建），`public` 里有表、视图、序列或函数的权限与文件不符时失败。
````
````new README.md
  - 少了 River 的权限时 `/readyz` 仍是 200，River 的任务（会话清理）和索引重建因 `permission denied`（42501）失败，只记在日志里；少了业务表的权限时接口请求失败；少了 `goose_db_version` 的读时 `/readyz` 是 503。`server/internal/bootstrap/runtime_role_test.go` 用恰好这些权限的角色运行 nerve（就绪、会话清理、索引重建），`public` 里有表、视图、序列或函数的权限与文件不符时失败。
- **数据库的 locale**：用 UTF-8 的 locale 初始化数据库（官方的 PostgreSQL 镜像是 `en_US.utf8`）。标签的名称在项目内不分大小写，按数据库的 `LC_CTYPE` 折叠（`lower(name)`）：以 `C` 初始化的数据库只折叠 ASCII，`Ä` 与 `ä` 算两个名称（M3/P7b spec 第 7 节）。
````

`docs/v0/M3-workspace-project/M3-design.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/M3-design.md
两个新模块文件：`api/modules/workspace.yaml`、`api/modules/project.yaml`（模块文件名等于模块目录名，M2 设计 3.12）；`access` 没有操作。路径都不带结尾 `/`（M1-P3 交接：项目标识的地址随新接口统一）。每个操作写 `security` 和 `x-problem-codes`（M2 设计 3.11、3.12）；下表"主要错误"一栏就是后者的内容，不再列出所有操作都可能返回的平台码。带 `bearer` 的操作另隐含 `unauthorized`；带请求体的都可能返回 400 `bad_request`（结构）。平台码 `forbidden` 写作 403 F，`validation_failed` 写作 422 V。一个操作可以声明别的模块的码（例如 `project.yaml` 的操作答 `workspace.not_found`），前缀是拒绝它的那个模块（按 11.7 修订的 M2 设计 3.11）。
````
````new docs/v0/M3-workspace-project/M3-design.md
两个新模块文件：`api/modules/workspace.yaml`、`api/modules/project.yaml`（模块文件名等于模块目录名，M2 设计 3.12）；`access` 没有操作。路径都不带结尾 `/`（M1-P3 交接：项目标识的地址随新接口统一）。每个操作写 `security` 和 `x-problem-codes`（M2 设计 3.11、3.12）；下表"主要错误"一栏就是后者的内容，不再列出所有操作都可能返回的平台码。带 `bearer` 的操作另隐含 `unauthorized`；带请求体的都可能返回 400 `bad_request`（结构）。平台码 `forbidden` 写作 403 F，`validation_failed` 写作 422 V。一个操作可以声明别的模块的码（例如 `project.yaml` 的操作答 `workspace.not_found`），前缀是拒绝它的那个模块（按 11.7 修订的 M2 设计 3.11）。

契约按模块一个文件，不按资源拆：`api/modules/project.yaml` 在 P7b 之后 1,348 行，超过约 400 行的规则，是写明的例外（M3/P7b spec 第 3 节第 1 条，控制者的裁定 B1）。仓库的约定由文件名得出模块：`Makefile` 按 `api/modules/<模块>.yaml` 生成到 `internal/modules/<模块>/adapter/http/gen`，`apitest` 要求一个文件中每个操作的标签就是这个模块；按资源拆出的文件会被当成另一个模块。要拆，是改这几处约定的单独任务；`project` 模块的接口在 P7b 已齐，这个文件不再长。
````

`docs/v0/frontend-changes.md`（修改，3 处）：

````old docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；页面已对接（工作区的 M3/P9，项目的 M3/P10，状态和标签的设置页、访客看到的个人主页 M3/P11），M3 的收尾之后改为已完成 |
````
````new docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；页面已对接（工作区的 M3/P9，项目的 M3/P10，状态和标签的设置页、访客看到的个人主页 M3/P11）；M3 的收尾重写了弹窗、加了离开工作区的入口、清完 M3 页面上的英文和键盘的遗留（3.2 中"完成于 M3/收尾"的各行），收尾合并之后改为已完成 |
````
````old docs/v0/frontend-changes.md
| 输入法组字时的 Escape | 用输入法（中文、日文等）组字时按 Escape 只结束组字，不再关上弹窗或表情选择器（原来创建项目的表单、删除项目等弹窗和表情选择器都随之关上）：`useKeypress`（`core/hooks/use-keypress.tsx`）不处理组字中的按键；`ModalCore`（`@nerve/ui`）的面板在组字中的 Escape 冒泡出去时 `preventDefault`，Headless UI 的 `Dialog` 照它的约定不关；`@nerve/propel` 的表情选择器的搜索框不关选择器。带搜索的选择的列表同样（"下拉框"一行）。以上是 Chromium 中的：Safari 先结束组字、再交出这个 Escape，只查 `isComposing` 的守卫拦不住（M3/P10 spec 第 5 节收尾） | 组字中的按键属于输入法（M3/P10 的评审） | 已完成 | M3/P10 |
````
````new docs/v0/frontend-changes.md
| 输入法组字时的 Escape | 用输入法（中文、日文等）组字时按 Escape 只结束组字，不再关上弹窗或表情选择器（原来创建项目的表单、删除项目等弹窗和表情选择器都随之关上）：`useKeypress`（`core/hooks/use-keypress.tsx`）不处理组字中的按键；`ModalCore`（`@nerve/ui`）的面板在组字中的 Escape 冒泡出去时 `preventDefault`，Headless UI 的 `Dialog` 照它的约定不关；`@nerve/propel` 的表情选择器的搜索框不关选择器。带搜索的选择的列表同样（"下拉框"一行）。Safari 先结束组字、再交出这个 Escape（`keyCode` 229）：M3 的收尾起，这些守卫都经共用的判断 `isComposing`（`@nerve/utils`），它也认这个顺序（"输入法和选择上的 Enter"一行） | 组字中的按键属于输入法（M3/P10 的评审） | 已完成 | M3/P10 |
````
````old docs/v0/frontend-changes.md
| 共用组件的键盘焦点 | propel 的 `Button` 在键盘取得焦点时显示浏览器的焦点轮廓，离按钮 2 像素（`focus-visible:outline-offset-2`；原来是 `focus-visible:outline-none`，Tab 到达时看不出焦点），应用中的每个 `Button` 都是这样。propel 的 `IconButton` 同样：应用中 32 个文件的 37 处（个人设置（M2）1 处；M3 的 10 个文件 12 处，侧边栏的收藏在内；工作项（M4）8 个文件 8 处；迭代、模块（M6）6 个文件 6 处；收集箱、通知和视图（M7）7 个文件 10 处），另有收集箱的两个菜单按钮（`getIconButtonStyling`）和 propel 的表情回应的添加按钮（3 个调用方）；其中五处的轮廓照旧看不到：侧边栏收藏的新建文件夹、侧边栏项目一行的菜单（外层 `opacity-0`）、工作项动态的筛选（外面的按钮 `outline-none`），侧边栏项目列表的两处在指针经过之前不显示、Tab 到不了（交给 M3 的收尾：M3/P11 spec 第 5 节收尾的键盘清单）。`@nerve/ui` 的 `CustomMenu` 的省略号按钮去掉 `outline-none`，Tab 到达时显示焦点的轮廓：17 个文件的 17 处（工作项一侧的 11 个文件（M4、M5），成员页的 `member-columns.tsx`，邀请行的 `invitations-list-item.tsx`，工作区视图的 3 个文件，标签的一行）；`member-columns.tsx` 给这个按钮自己的 `opacity-0 group-hover:opacity-100`，所以它的轮廓在键盘取得焦点时仍看不见（M3/P10 的成员页原有的，同在那份键盘清单）。`getButtonStyling` 的类名同样不再有 `focus-visible:outline-none`：应用中 9 个文件的 12 处，其中 4 个文件的 6 处在能取得焦点的元素上：M3 的两个文件的 4 个链接（创建项目之后功能一步的"Open project"，`project-feature-update.tsx`；"找不到工作区"页的三个链接，`workspace-wrapper.tsx`）在键盘取得焦点时显示离它 2 像素的轮廓（原来没有）；M4 的两处照旧，由它们自己的样式决定：工作项筛选的"添加筛选"按钮（`rich-filters/add-filters/button.tsx`）的轮廓是 `CustomSearchSelect` 在 Headless UI 的 `data-focus` 上画的那一条（`data-*` 的类排在 `focus-visible` 的之后），布局的下拉框（`dropdowns/layout.tsx`）的按钮自带 `outline-none`，轮廓照旧看不到；其余 6 处给菜单按钮里面的 `div`、`span`（项目、模块、视图的排序，收集箱的筛选），它们自己不取得焦点。用指针点的人看不到不同：`:focus-visible` 在点击时不生效 | Tab 到达的控件看得出焦点（M3/P11 的状态、标签的设置页用到它们；改在共用组件里） | 已完成 | M3/P11 |
````
````new docs/v0/frontend-changes.md
| 共用组件的键盘焦点 | propel 的 `Button` 在键盘取得焦点时显示浏览器的焦点轮廓，离按钮 2 像素（`focus-visible:outline-offset-2`；原来是 `focus-visible:outline-none`，Tab 到达时看不出焦点），应用中的每个 `Button` 都是这样。propel 的 `IconButton` 同样：应用中 32 个文件的 37 处（M3 的收尾删去侧边栏项目一行的那一处之后是 36 处；个人设置（M2）1 处；M3 的 10 个文件 12 处，侧边栏的收藏在内；工作项（M4）8 个文件 8 处；迭代、模块（M6）6 个文件 6 处；收集箱、通知和视图（M7）7 个文件 10 处），另有收集箱的两个菜单按钮（`getIconButtonStyling`）和 propel 的表情回应的添加按钮（3 个调用方）；其中五处的轮廓照旧看不到：侧边栏收藏的新建文件夹、侧边栏项目一行的菜单（外层 `opacity-0`）、工作项动态的筛选（外面的按钮 `outline-none`），侧边栏项目列表的两处在指针经过之前不显示、Tab 到不了（M3 的收尾改好了侧边栏项目的三处，工作项动态的筛选交给 M4、收藏的新建文件夹交给 M7："名字和键盘"一行）。`@nerve/ui` 的 `CustomMenu` 的省略号按钮去掉 `outline-none`，Tab 到达时显示焦点的轮廓：17 个文件的 17 处（工作项一侧的 11 个文件（M4、M5），成员页的 `member-columns.tsx`，邀请行的 `invitations-list-item.tsx`，工作区视图的 3 个文件，标签的一行）；`member-columns.tsx` 给这个按钮自己的 `opacity-0 group-hover:opacity-100`，所以它的轮廓在键盘取得焦点时仍看不见（M3/P10 的成员页原有的；M3 的收尾给它 `focus-visible:opacity-100`）。`getButtonStyling` 的类名同样不再有 `focus-visible:outline-none`：应用中 9 个文件的 12 处，其中 4 个文件的 6 处在能取得焦点的元素上：M3 的两个文件的 4 个链接（创建项目之后功能一步的"Open project"，`project-feature-update.tsx`；"找不到工作区"页的三个链接，`workspace-wrapper.tsx`）在键盘取得焦点时显示离它 2 像素的轮廓（原来没有）；M4 的两处照旧，由它们自己的样式决定：工作项筛选的"添加筛选"按钮（`rich-filters/add-filters/button.tsx`）的轮廓是 `CustomSearchSelect` 在 Headless UI 的 `data-focus` 上画的那一条（`data-*` 的类排在 `focus-visible` 的之后），布局的下拉框（`dropdowns/layout.tsx`）的按钮自带 `outline-none`，轮廓照旧看不到；其余 6 处给菜单按钮里面的 `div`、`span`（项目、模块、视图的排序，收集箱的筛选），它们自己不取得焦点。用指针点的人看不到不同：`:focus-visible` 在点击时不生效 | Tab 到达的控件看得出焦点（M3/P11 的状态、标签的设置页用到它们；改在共用组件里） | 已完成 | M3/P11 |
| 弹窗 | `ModalCore`（`@nerve/ui`）改用 Headless UI v2 的 `Dialog`，以 `DialogBackdrop`、`DialogPanel` 做过渡（原来是 `Transition.Root`、`Transition.Child`）：弹窗打开时后面的页面 `inert`、`aria-hidden`，指针点不到、读屏读不到、Tab 走不出弹窗，关上之后都恢复，焦点回到打开它的控件；只在打开时才挂载的弹窗也是这样（它从挂载之后的那次渲染打开；原来这样的弹窗后面的页面照旧能操作）。从弹窗打开的列表、选择器和提示渲染在页面之外，照旧能用、被读出。请求在路上时关不掉、输入法组字时的 Escape 不关，照旧。`AlertModalCore` 的确认按钮不再是 `tabIndex={1}`：Tab 在取消和确认之间转（原来先到确认，走过最后一个就离开弹窗）。命令面板、快捷键和切换账户三个弹窗改经 `ModalCore`（原来直接用 Headless UI 的 `Dialog`）：快捷键的搜索框不再是 `tabIndex={1}`，关闭按钮有名字、在标题之外（弹窗的名字就是标题），标题左对齐（原来居中）；切换账户的弹窗在退出的请求在路上时关不掉，层和宽度随 `ModalCore`（`z-20` → `z-30`，`w-[40rem]` → `sm:max-w-2xl`）。只在打开时才挂载的弹窗现在也有打开的过渡（原来立即出现）。命令面板从 `z-50` 到了弹窗的一层（`z-30`；提示、tooltip、表情和右键菜单的弹层照旧在它之上）：页面上的元素停在这一层之下，设置页的手机菜单（`settings/mobile/nav.tsx`）因此从 `z-50` 降到 `z-20`（原来它画在任何弹窗之上） | 弹窗之外的页面不能操作，也不被读出（负责人 2026-10-10；M3/P10 的缺陷 ii） | 已完成 | M3/收尾 |
| 离开工作区 | 侧边栏的工作区菜单加"离开工作区"，每个角色都有，访客由此第一次能在页面上离开工作区。它打开成员页离开时的同一个确认框，经同一个命令（`useMembershipChanges` 的 `leave`）：成功之后只在发出它的会话里跟进，去成员页离开之后去的地方；nerve 拒绝时（唯一的管理员）确认框留着，原因在提示里。工作区菜单的按钮在键盘取得焦点时显出内圈的轮廓，方向键选中的一项（工作区和三个操作）与指针悬停时一样显出底色，当前工作区的一项选中时深一级（原来都看不出）。换工作区、换项目（后退、前进也是）时工作区的内容、项目的页面和设置重新挂载，开着的确认框、命令面板和快捷键列表关上（原来跟过去，它的"离开""移出""归档"作用于那个工作区或项目）；经菜单换工作区之后焦点在页面的开头（原来回到菜单的按钮），侧边栏的"Projects"一节重新展开，滚动回到顶上，窄屏（768 px 以下）上开着的侧边栏收起。成员页的移出、离开和邀请的删除被拒绝时，确认框同样留着（原来关上）；离开的确认框的标题经 `t()`。工作区菜单只有顶部导航一种用法，没有调用方的"侧边栏"写法删除 | 负责人 2026-10-11（"菜单加入口，收尾做"）；被拒绝时弹窗留着（总体设计 7.7） | 已完成 | M3/收尾 |
| M3 页面上的文案 | M3 页面上写死的英文都经 `t()`（en、zh-CN）：工作区的 general 页和删除弹窗、成员页（列、角色、移出和离开的确认框）、邀请页和"找不到工作区"、没有权限的页面的标题、项目导航对话框、设置侧边栏的标题、命令面板的占位、快捷键弹窗的标题和搜索框、切换账户的那句话、首页的问候（一个时段一句、带名字：中文原来把"good"也译作"早上"，读作"早上 下午""早上 晚上"；名字改为显示名，原来是英文语序的名和姓）、侧边栏、项目列表（卡片、筛选的下拉框和已选的筛选）、项目的弹窗（删除、离开、加入、归档和恢复、移出成员、添加成员）、项目设置各页的标题、功能、自动归档、状态和标签的设置页（状态组的标题读组的名字，原来读屏读到小写的键）、新建工作区和新手引导的页面标题、提示区域的名字。一句话里带名字的是一个键，名字照旧加粗（`components/common/translated.tsx`）。角色名只有一个来源（`ROLE_DETAILS` 的 `i18n_title`；`ROLE`、`getUserRole` 删除）。邀请被拒绝之后的页面说"This invitation was declined."/"这份邀请已被拒绝。"（原来对打开链接的人说"You declined this invitation."/"您已忽略这份邀请。"；负责人 2026-10-13）。英文改了的说法（忙时的"Leaving""Joining"不带"..."，"Delete label"，项目设置的标题等）逐条在 [M3 收尾 spec](M3-workspace-project/specs/closeout.md) 附录 A.8 | 中文界面不夹英文 | 已完成 | M3/收尾 |
| 名字和键盘 | propel 的 `IconButton` 的 `aria-label` 必传，应用中 36 处都有名字（个人设置 1、M3 11、工作项 8、迭代和模块 6、收集箱、通知和视图 10）；表情回应的按钮、项目列表已选筛选的移除按钮（"移除 X"）、项目卡片上设置的链接（以项目为名，"Web 设置"）、项目的功能开关有名字；侧边栏的展开、收起和工作区菜单的按钮的名字有了文案（原来是没有定义的键）；图标的连字和表情不再并进旁边的名字。状态、标签的颜色弹层的十六进制输入框在键盘取得焦点时有轮廓（`components/common/color-picker.tsx`）；状态的表单打开时焦点在名称上；侧边栏项目一行的按钮以项目为名（原来每一行都叫"Open project menu"），列表的"+"、项目一行的菜单在键盘取得焦点时显示，一行的菜单只是一个停留点（菜单的按钮里不再嵌一个按钮），拖动的手柄不再是停留点；项目成员页的筛选同样只是一个按钮；项目成员页一行的菜单有名字（"Options"），键盘取得焦点时显示；项目下拉框的停留点是它的按钮（根不再可以聚焦）；收起的侧边栏和没有浮出的侧边栏 `inert`（原来 Tab 和读屏都走到看不见的那一份） | 每个控件 Tab 能到、有名字、键盘取得焦点时看得见（总体设计 7.7；M3/P11 spec 第 5 节收尾的键盘清单） | 已完成 | M3/收尾 |
| 输入法和选择上的 Enter | 组字中的按键由一个共用的判断认出：`@nerve/utils` 的 `isComposing`（`isComposing` 或 `keyCode` 229，Safari 先结束组字、再交出 229 的按键也认得），`@nerve/ui`、`@nerve/propel` 和 web 的守卫都经它（`@nerve/propel` 因此依赖 `@nerve/utils`）；命令面板的搜索框为空时，结束组字的 Escape 不再关上面板。`CustomSelect` 的按钮上按 Enter 打开列表（原来提交所在的表单：添加成员的弹窗中在角色上按 Enter 就立即添加） | 组字中的按键属于输入法；Enter 不意外提交（M3/P10 的浏览器核对） | 已完成 | M3/收尾 |
| 工作区的地址 | M3 页面上只读工作区的 19 个文件不再读路由 store 的 `currentWorkspace`：经 `useRouteWorkspace()`（`core/hooks/use-route-workspace.ts`：路由的 slug 经调用者的工作区列表找到），工作区级的权限判断写明 slug；关键词规则 `workspace-address-route` 看住它们，与项目一侧的 `project-address-route` 同一个规则 | 地址只从路由参数（总体设计 7.7） | 已完成 | M3/收尾 |
| 共用的写法 | 复制只经 `useCopyLink`、`useCopyText`（`core/hooks/use-copy-link.ts`）：API 令牌弹窗的复制和命令面板的"复制当前页面的地址"也经它们（复制失败是共用的提示，两条专用的文案删除）；关键词规则 `copy-hook` 看住，M4、M6、M7、M8 的 20 个文件是带期限的例外。问题码只经 `problemCode()`（`core/lib/api-error.ts`），拒绝的文字只经 `useRefusalMessage`（主题切换的失败提示也是）。`ButtonAvatars` 的 `showTooltip` 删除（没有读者）。`make lint-web` 加 `tools/i18n-keys.mjs`：页面读的每个键在语言文件中都有（它第一次运行就找到侧边栏的四个没有定义的键和一个没有读者的日期筛选项） | 一件事只有一份写法 | 已完成 | M3/收尾 |
````

`docs/v0/plane-diff.md`（修改，1 处）：

````old docs/v0/plane-diff.md
| 邀请的链接 | `/workspace-invitations/?invitation_id=…&slug=…&token=…`；另有系统内的接受：`/invitations` 页和新手引导的"加入工作区"一步按账户的邮箱列出发给他的邀请，批量接受 | 只有链接一条路：`/workspace-invitations?invitation_id=…&token=…`；接受要登录，账户的邮箱须与邀请的相同（M3 设计 3.8、决策点 2）（页面：P9；`/invitations` 页和新手引导的一步由 P8–P11 删除） |
````
````new docs/v0/plane-diff.md
| 邀请的链接 | `/workspace-invitations/?invitation_id=…&slug=…&token=…`；另有系统内的接受：`/invitations` 页和新手引导的"加入工作区"一步按账户的邮箱列出发给他的邀请，批量接受 | 只有链接一条路：`/workspace-invitations?invitation_id=…&token=…`；接受要登录，账户的邮箱须与邀请的相同（M3 设计 3.8、决策点 2）（页面在 M3/P9；`/invitations` 页和新手引导的一步在 M3/P8a 删除） |
````

- [ ] **Step 2: 第 14 节的证据（评审的提交用）**

在收尾的头上照 spec 2.15 的表逐项重跑，把命令和输出交给评审：`make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e`（118 个）；`grep -l -E '^status: (open|done)$' docs/v0/M3-workspace-project/handoffs/*.md` 没有输出；M1 收尾计划附录 A 的 `deadsym.mjs --tsv` 和 `domains.mjs --rows M3`（225 行，spec 附录 A.5）；spec 附录 A.7 的页面与故事的表。第 14、15 节、总体设计 9.4 和前端改动清单 3.1 的状态翻转是控制者在评审的提交里做的（spec 2.16），本 Task 不改。

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `71 rules, 23 exceptions, no hits.`。

Run: `python3 $M3TMP/closeout-plan/doclinks.py . README.md docs/v0/M3-workspace-project/M3-design.md docs/v0/frontend-changes.md docs/v0/plane-diff.md`
Expected: 退出码 0；4 个文件每个一行 `0 broken`，最后一行 `broken: 0`。

- [ ] **Step 4: 提交**

```bash
git add README.md docs/v0/M3-workspace-project/M3-design.md docs/v0/frontend-changes.md docs/v0/plane-diff.md
```
```bash
git commit -m "docs(M3/closeout): README, plane-diff, the M3 design and the frontend change list say what the closeout did

README's deployment notes say the database's LC_CTYPE and lose two stale
'page: P9' notes; plane-diff's invitation-link row says where its pages
went; the M3 design records ruling B1 (one contract file per module); the
frontend change list gets the closeout's seven rows and its IME and
keyboard rows say what is true now.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异（同上）。
