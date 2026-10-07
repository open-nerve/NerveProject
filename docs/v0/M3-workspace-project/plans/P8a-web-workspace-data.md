# M3/P8a 工作区一侧的数据层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 会话分代的基础：`RootStore.dispose()` 释放 `project_filter` 的反应，`sessionGuard()`，带 `loginId` 的 SWR 键只有一种写法（`sessionKey`、`useSessionSWR`）。problem 码的文案表移到 `core/lib/error-messages.ts` 和 `errors` 命名空间。系统内接受删除。工作区一侧的类型、service、store（工作区、成员、邀请、邀请链接的查看与接受、工作区的显示设置、权限 store 的工作区一半）迁到 `/api/v0`，按会话分代，修改一个接一个。落点：`landingPath` 算出地址，`useLanding` 做出全部判断，`AuthenticationWrapper` 照它渲染；`RESTRICTED_URLS`、设置 store 删除，保留名单前后端一份。两个包装层和顶部导航中 M6、M7 的挂载时取数删除，工作区一侧的挂载时取数改调新 store、按权限启用。使用方改到能编译、行为不变（spec 第 3 节列出的除外）。

**Architecture:** 只改 web（另有 `server/.../reserved_slugs.txt` 一个文本文件）。`core/lib`：`session-key.ts`、`use-session-swr.ts`、`in-session.ts`、`error-messages.ts`、`landing.ts`、`use-landing.ts`，测试用的 `fake-session-swr.ts`。`core/services/workspace/`：`workspaces`、`workspace-members`、`workspace-invitations`、`workspace-preferences` 四个 service，每个都是生成的客户端之上的薄封装，由 store 用这一代的 `ApiClient` 构造；`invitation-preview.service.ts` 是一个函数，调用方传 `publicClient`。store：`WorkspaceRootStore`（重写，带子 store `WorkspacePreferencesStore`）、`WorkspaceMemberStore`（成员和邀请，重写）、`UserPermissionStore` 的工作区一半、`ThemeStore` 接过设置的侧边栏。挂载时的取数：`useLanding`（登录、注册、引导页的落点）、`useWorkspaceFetch`（工作区包装层）、`useMembersSettingsFetch`（成员页）、`useInvitationPreview`（登录、注册、邀请页），按权限的条件都由 hook 自己从 store 算出。测试的共用部分：`fake-queue.ts`、`fake-root.ts`、`fake-workspaces.ts`、`fake-members.ts`、`FakeNerve.replacedSessionClient`。静态检查：根目录 `.oxlintrc.json` 的 `overrides`（Task 8）不让 P8a 的会话取数绕过 `useSessionSWR`，也不让 P8a 的 M3 路径用非空断言；web 应用的 TypeScript `lib` 改为 ES2023（Task 7，`toSorted`）。契约不变；不加 npm 包，`@nerve/types`、`@nerve/utils` 加工作区内的依赖 `@nerve/api-client`（Task 4，锁文件两处）。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、openapi-fetch 0.17.0、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1（Go 代码不变）。版本都由 `pnpm-lock.yaml` 固定。

**Spec:** `docs/v0/M3-workspace-project/specs/P8a-web-workspace-data.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 包或 Go 模块，不执行 `go get`、`pnpm add`。`pnpm-lock.yaml` 只在 Task 4 改两处（`@nerve/types`、`@nerve/utils` 依赖工作区内的 `@nerve/api-client`），之后执行 `pnpm install --frozen-lockfile`。`git diff --stat 7cf3a286 -- server/go.mod server/go.sum server/tools` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-web`（关键词守卫、`tsc`、oxlint 在上限、格式、`en` 与 `zh-CN` 的键一致）、`make knip`、`make test-web`、`make e2e` 都通过。改了 `server/` 的 Task 3、5 另执行 `make lint-go`、`make test`。本 plan 不改接口描述，不生成任何文件，不执行 `make gen`。
- **e2e 的已知失败**：只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（P4b spec F4）；在 worktree 中 `make e2e` 必须全部通过（70 个）。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**：store 每个资源一个，按会话分代，由 `RootStore` 用这一代的 `ApiClient` 建；没有模块级的 service 实例（只调公开操作的 `previewInvitation` 由调用方传 `publicClient`）；修改经 `oneAtATime()` 排队，取数不排队；修改不乐观，store 写入 nerve 的回答；取数遇到 `SessionChangedError` 给出 `undefined`、不改本代的状态。会话的每个取数都经 `useSessionSWR`（键 `[取数的名称, loginId, ...取数的参数]`，fetcher 交回 store 的 Promise），没有权限时键为 `null`，权限的条件由取数的 hook 自己从 store 算出、不由调用方传入；唯一的例外是经 `publicClient` 的公开操作（`useInvitationPreview`，键只有链接，spec 第 3 节）。这一条有静态检查：根目录 `.oxlintrc.json` 的 `no-restricted-imports` 不让范围内的文件从 `swr` 导入值（只能导入类型；Task 8，范围和它在 P8b 之前的限制见 spec 附录 A.6）。类型只来自生成的客户端：不写重述契约的类型，不留旧名字的别名，不写把新接口映射成 Plane 形状的适配层。删除的代码删干净（路由、入口、store 和 service 的方法、两种语言的文案、常量、它们的测试），不加 knip 的忽略、开关或桩。新的 `as`、`any` 只有 spec 第 3 节写明的一处（`fake-root.ts`）；没有新的 `!`，P8a 的 M3 路径上 `typescript/no-non-null-assertion` 是错误（Task 8 的同一组 `overrides`）。本 plan 写或重写的文件都在约 400 行以内（最终原型上量的：最长的是 `core/store/member/workspace/workspace-member.store.ts` 358 行、`core/store/workspace/index.test.ts` 293 行、`core/store/user/permissions.store.ts` 277 行）；只为使用方改到的 Plane 文件不拆，也不变长：有手改的、超过 400 行的五个都变短了（`module.store.ts` 613 → 573、`project-member.store.ts` 497 → 491、`favorite.store.ts` 441 → 412、`workspace-draft/issue.store.ts` 422 → 408、`workspace-notifications.store.ts` 402 → 401）；只经机械步骤改到的、超过 400 行的三个中，`use-work-item-filters-config.tsx` 404 行不变，`power-k/config/navigation/commands.ts` 464 → 465、`project/form.tsx` 461 → 462 各多一行：`rename_type.py` 能把新类型并进已有的 `import type` 时就并进（不加行），这两个文件原来没有从 `@nerve/api-client` 的导入，生成的 `Workspace` 只能另起一行导入。
- **oxlint**（M3 设计 7.9，裁定 R3）：有手改的文件（每个有块的 TS 文件）在它的 Task 提交时没有 oxlint 警告；只经本 plan 的机械步骤改到的文件（Task 2、4、7 的 Step 1 的文件中没有块的那些）不在此列，它们的警告留给 P11 的第 4 个任务，spec 附录 A 列出它们。没有新的 oxlint 抑制，删去四处（Task 7、8 各一处 `promise/always-return`，Task 7 两处 `unicorn/no-array-sort`，改为 `toSorted`）。web 应用的上限在改变条数的 Task 里调到新的条数：452 → 451（Task 4）→ 448（Task 6）→ 445（Task 7）→ 444（Task 8）→ 442（Task 9）→ 435（Task 11）；`types` 0、`constants` 1、`utils` 12 不变。
- **注释**：TS 代码、测试、JSON 的说明用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `7cf3a286` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **机械步骤**（brief 的规则）：Task 2、4、7 的 Step 1 只改导入路径或类型名，由命令对写明的文件列表执行，不写成块：`Run（机械步骤）:` 的命令照原样从仓库根执行，它们用的两个一次性脚本在下面"一次性脚本"一节全文给出，写到 `$P8ATMP`（实现者自己的临时目录，在仓库之外；每条命令之前 `export P8ATMP=<那个目录>`）。步骤之后 `git diff --numstat` 必须恰好是步骤给出的那几行，`shasum -a 256` 的结果与表中的散列和行数相同；对不上时停下来：说明某个输入与本 plan 不一致。Task 4、7 的块写在机械步骤之后的文件上（Task 7 有几个文件既经机械步骤改名、又有手改）。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：`tools/keywords.json`（Task 3–10 各加本 Task 的规则）、旧的 `core/services/workspace.service.ts`（Task 3–11 逐个删去搬走的方法）、`core/layouts/auth-layout/workspace-wrapper.tsx` 和 `use-workspace-fetch.ts`、`use-workspace-fetch.test.ts`（Task 6 取列表，Task 8 加成员，Task 10 加显示设置）、`core/store/workspace/index.ts`、`index.test.ts`（Task 4、5、9、10）、`workspaces.service.ts`（Task 4、5、9）、`workspace-member.store.ts`（Task 7、8）、`permissions.store.ts`（Task 5、6）、`packages/constants/src/fetch-keys.ts`、`packages/types/src/workspace.ts`、`web/apps/web/package.json` 的上限，以及 spec 第 2 节逐个列出的使用方。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（静态：`make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫、i18n 的 `check:sync`，和 `make knip`；vitest：`make test-web`；端到端：`make e2e` 的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p8tools/mutants_p8a.py`，`mut.py` 在它写的每个检查上各跑一次；spec 附录 A：125 个变异，123 个被发现，2 个由之后的 Phase 发现，各写明由谁）；一个变异列在它改的代码第一次出现的 Task。实现者可以照表抽查，改坏之后必须恢复。表中"（之后的 Task 起）"标出的检查在这一行所在的 Task 还不发现它（那个测试在之后的 Task 才有或才改成最终的样子）；没有标的在这一行所在的 Task 的树上就失败。**会话、权限或取数的性质只由评审才能发现的，算缺口**（brief 的缺陷类别）：表中每一条这类性质都有一个会失败的检查，例外写在 spec 附录 A。
- **评审敏感**（M3 设计 12 节 P8a 的评审重点）：换代之后旧的一代不再写、它的反应已释放（Task 1）；页面只取调用者有权读的（成员页的邀请只为工作区管理员取，工作区的子资源只在列表说明它是调用者的工作区之后才取，条件由 hook 自己从工作区列表算出，Task 6、8、10）；落点只认此刻工作区列表中的工作区，判断的每一项都在 `useLanding` 里（Task 5）。改动这些取数的条件、键或 store 的会话处理之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性脚本

Task 2、4、7 的机械步骤用下面两个脚本，照原样写到 `$P8ATMP`（不提交）。

`$P8ATMP/t2_imports.py`（Task 2：读错误文案的组件改从新位置导入）：

```python
# -*- coding: utf-8 -*-
# One-off (M3/P8a, Task 2's mechanical step): the components that read the messages of nerve's errors import them
# from their new place, core/lib/error-messages.ts, instead of the sign-in helper; password.tsx, which also
# imports EAuthModes, keeps that import from the helper. Only the import lines change. Run from the repository
# root; it checks that each file has exactly the import it replaces.
# usage: python3 t2_imports.py [<root>]
import os
import sys

root = sys.argv[1] if len(sys.argv) > 1 else '.'
C = 'web/apps/web/core/components/'
HELPER = 'from "@/helpers/authentication.helper";'
NEW = 'from "@/lib/error-messages";'
FILES = [
    C + 'account/deactivate-account-modal.tsx',
    C + 'api-token/delete-token-modal.tsx',
    C + 'api-token/modal/form.tsx',
    C + 'appearance/theme-switcher.tsx',
    C + 'onboarding/root.tsx',
    C + 'onboarding/steps/profile/root.tsx',
    C + 'profile/start-of-week-preference.tsx',
    C + 'settings/profile/content/pages/general/form.tsx',
    C + 'settings/profile/content/pages/preferences/language-and-timezone-list.tsx',
    C + 'settings/profile/content/pages/security.tsx',
]
for rel in FILES:
    path = os.path.join(root, rel)
    text = open(path, encoding='utf-8').read()
    assert text.count(HELPER) == 1 and 'EAuthModes' not in text and 'EPageTypes' not in text, rel
    open(path, 'w', encoding='utf-8').write(text.replace(HELPER, NEW))
PASSWORD = os.path.join(root, C + 'account/auth-forms/password.tsx')
text = open(PASSWORD, encoding='utf-8').read()
OLD = 'import { EAuthModes, errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/helpers/authentication.helper";\n'
assert text.count(OLD) == 1, PASSWORD
text = text.replace(OLD, 'import { EAuthModes } from "@/helpers/authentication.helper";\n'
                         'import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";\n')
open(PASSWORD, 'w', encoding='utf-8').write(text)
print(len(FILES) + 1, 'files')
```

`$P8ATMP/rename_type.py`（Task 4、7：把一个 Plane 类型换成生成的类型）：

```python
# -*- coding: utf-8 -*-
# M3/P8a's mechanical step for a type name (the plan gives this script whole): in each file named, the Plane type
# <old>, imported from @nerve/types (or, with --from, from that module: the types package's own files import it by a
# relative path), becomes the generated type <new> of @nerve/api-client. Every whole-word <old> becomes <new>; <old>
# leaves the file's `import type { … } from "<module>";` (the line goes when nothing is left in it), and <new> joins
# the file's `import type { … } from "@nerve/api-client";`, or such a line is added where the old import was. It
# checks first that each file imports <old> from the module in one `import type` line and does not use <new>
# already, and changes nothing unless every file passes.
# usage: python3 rename_type.py <root> <old> <new> [--from <module>] <file>...   (files relative to <root>)
import re
import sys

root, old, new, rest = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4:]
module = '@nerve/types'
if rest[:1] == ['--from']:
    module, rest = rest[1], rest[2:]
files = rest
# an `import type { … }` on one line, or over several (one name a line, as oxfmt writes a long one); the step writes
# it back on one line, and oxfmt, which the plan runs after it, lays it out again
TYPES = re.compile(r'^import type \{\s*([^}]*?)\s*\} from "' + re.escape(module) + r'";\n', re.M)
CLIENT = re.compile(r'^import type \{\s*([^}]*?)\s*\} from "@nerve/api-client";\n', re.M)
word = re.compile(r'\b' + re.escape(old) + r'\b')


def names(group):
    return [n.strip() for n in group.split(',') if n.strip()]


results = {}
for rel in files:
    path = f'{root}/{rel}'
    text = open(path, encoding='utf-8').read()
    found = [m for m in TYPES.finditer(text) if old in names(m.group(1))]
    if len(found) != 1:
        sys.exit(f'{rel}: imports {old} from {module} in {len(found)} import type lines')
    if re.search(r'\b' + re.escape(new) + r'\b', text):
        sys.exit(f'{rel}: already uses {new}')
    line = found[0]
    others = [n for n in names(line.group(1)) if n != old]
    client = CLIENT.search(text)
    replacement = f'import type {{ {", ".join(others)} }} from "{module}";\n' if others else ''
    if client is None:
        replacement = f'import type {{ {new} }} from "@nerve/api-client";\n' + replacement
    text = text[:line.start()] + replacement + text[line.end():]
    if client is not None:
        client = CLIENT.search(text)
        listed = names(client.group(1)) + [new]
        text = text[:client.start()] + f'import type {{ {", ".join(listed)} }} from "@nerve/api-client";\n' + text[client.end():]
    results[path] = word.sub(new, text)
for path, text in results.items():
    open(path, 'w', encoding='utf-8').write(text)
print(f'{old} -> {new} in {len(results)} files')
```

## 文件结构

路径相对于仓库根目录；"web/"之下的应用写作 `web/apps/web/…`。一个文件由多个 Task 修改时，"Task"一列都列出；"机械"表示只经那个 Task 的机械步骤改到。

| 文件 | 职责 | Task |
|---|---|---|
| `web/apps/web/core/lib/session-key.ts`、`web/apps/web/core/lib/session-key.test.ts` | `sessionKey`：会话的取数的 SWR 键（名称、`loginId`、参数），没有登录或参数未知时为 `null` | 1 |
| `web/apps/web/core/lib/use-session-swr.ts`、`web/apps/web/core/lib/use-session-swr.test.ts` | `useSessionSWR`：组件发出会话的取数的唯一写法，fetcher 收到参数、交回 store 的 Promise | 1 |
| `web/apps/web/core/lib/in-session.ts`、`web/apps/web/core/lib/in-session.test.ts` | `sessionGuard()`：发出修改时取会话，之后核对 | 1 |
| `web/apps/web/core/lib/store-context.tsx`、`web/apps/web/core/lib/store-context.test.ts`、`web/apps/web/core/store/project/project_filter.store.ts` | 换代时调用退役一代的 `dispose()`；`project_filter` 保存反应的释放函数 | 1 |
| `web/apps/web/core/store/root.store.ts` | `dispose()`（Task 1）；工作区 store 用这一代的客户端建（Task 4）；成员 store 同样（Task 7） | 1、4、7 |
| `web/apps/web/core/components/appearance/theme-switcher.tsx` | 改用 `sessionGuard()`（Task 1）；错误文案的导入（Task 2 机械） | 1、2 机械 |
| `web/apps/web/core/components/api-token/token-list.tsx` | 改用 `useSessionSWR` | 1 |
| `web/apps/web/core/lib/wrappers/authentication-wrapper.tsx` | 改用 `useSessionSWR`（Task 1）；照 `useLanding` 渲染落点，四处 `fetchCurrentUserSettings` 删除（Task 5） | 1、5 |
| `web/apps/web/core/lib/error-messages.ts`、`web/apps/web/core/lib/error-messages.test.ts` | `PROBLEM_MESSAGES`、`FIELD_ERROR_MESSAGES`、`errorMessageKey`、`fieldErrorKeys`、`needsErrorBanner` 和它们与契约一致的测试，从登录的辅助文件移来 | 2 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/apps/web/helpers/authentication.helper.test.ts` | 只留 `EAuthModes`、`EPageTypes`（整个文件）；测试随表移走（删除） | 2 |
| `web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`、`web/packages/i18n/src/constants/namespaces.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json` | `errors` 命名空间：problem 码和字段错误的文案从 `auth` 移来 | 2 |
| `web/apps/web/core/components/account/auth-forms/password.tsx`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`、`web/apps/web/core/components/api-token/delete-token-modal.tsx`、`web/apps/web/core/components/api-token/modal/form.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`web/apps/web/core/components/profile/start-of-week-preference.tsx`、`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`、`web/apps/web/core/components/settings/profile/content/pages/security.tsx` | 错误文案改从 `@/lib/error-messages` 导入 | 2 机械 |
| `web/apps/web/app/(all)/invitations/layout.tsx`、`web/apps/web/app/(all)/invitations/page.tsx`、`web/apps/web/app/routes/core.ts` | `/invitations` 页和它的路由（删除） | 3 |
| `web/apps/web/app/routes/navigation.test.ts` | `/invitations` 落到与未知地址相同的地方（Task 3）；保留名单"应用"一段与路由、`public/` 两个方向一致（Task 5） | 3、5 |
| `web/apps/web/core/components/onboarding/header.tsx`、`web/apps/web/core/components/onboarding/steps/root.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/index.ts`、`web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/root.tsx` | 新手引导的"加入工作区"一步删除（`join-invites.tsx` 删除，`root.tsx` 整个文件） | 3 |
| `web/apps/web/core/components/onboarding/root.tsx` | 错误文案的导入（Task 2 机械）；`invitations` 删除（Task 3）；`Workspace`（Task 4） | 2 机械、3、4 |
| `web/apps/web/core/components/onboarding/steps/workspace/create.tsx` | 系统内接受的入口（Task 3）；`Workspace`、新 store（Task 4）；slug 检查改问 nerve（Task 5） | 3、4、5 |
| `web/apps/web/core/components/power-k/config/account-commands.ts`、`web/packages/i18n/src/locales/en/power-k.json`、`web/packages/i18n/src/locales/zh-CN/power-k.json` | 命令面板的"工作区邀请"删除 | 3 |
| `web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`、`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx` | 邀请的入口删除（Task 3）；`Workspace`（Task 4） | 3、4 |
| `web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json` | 系统内接受的文案（Task 3）；`show_all`、`show_less`（Task 11）删除 | 3、11 |
| `server/internal/modules/workspace/domain/reserved_slugs.txt` | "应用"一段：`invitations` 删除（Task 3）；与路由和 `public/` 一致（Task 5） | 3、5 |
| `tools/keywords.json` | 每个 Task 的关键词规则（`in-system-invitations`、`plane-workspace-urls`、`restricted-urls`、`user-settings` 和工作区一侧的地址模式） | 3–10 |
| `web/packages/constants/src/fetch-keys.ts` | 搬走或删除的取数键 | 3、6、8、9、10、11 |
| `web/packages/types/src/workspace.ts` | M3 的 Plane 类型逐个删除，只留 M4 搜索用的类型、`EUserWorkspaceRoles`、新手引导的步骤 | 3、4、6、7、7 机械、8、9 |
| `web/apps/web/core/services/workspace.service.ts` | 旧 `WorkspaceService`：M3 的方法逐个删去，只留 M4、M7 的（搜索、视图、草稿） | 3–11 |
| `web/apps/web/core/services/workspace/workspaces.service.ts` | 工作区：列出、创建、修改、删除（Task 4）；离开、slug 检查（Task 5）；接受、忽略邀请（Task 9） | 4、5、9 |
| `web/apps/web/core/store/workspace/index.ts`、`web/apps/web/core/store/workspace/index.test.ts` | `WorkspaceRootStore`：重写（Task 4）；slug 检查、离开（Task 5）；接受、忽略（Task 9）；显示设置的子 store（Task 10） | 4、5、9、10 |
| `web/apps/web/core/store/fake-queue.ts`、`web/apps/web/core/store/fake-root.ts`、`web/apps/web/core/store/workspace/fake-workspaces.ts` | store 测试的共用部分：`inTurn`、`fetchedWhileChangeIsOut`、`fakeRoot`、`workspaceOf`、`loadWorkspaces` | 4 |
| `web/apps/web/core/lib/auth/fake-nerve.ts` | `replacedSessionClient()`：会话已换的客户端 | 4 |
| `web/apps/web/core/components/core/modals/workspace-image-upload-modal.tsx`、`web/apps/web/core/services/file.service.ts`、`web/packages/constants/src/file.ts`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json` | 工作区图标的上传删除（`WorkspaceUpdate` 没有 `logo_url`，spec 第 3 节）；general 页改用新 store | 4 |
| `web/apps/web/app/(all)/create-workspace/page.tsx`、`web/apps/web/core/components/workspace/delete-workspace-form.tsx`、`web/apps/web/core/components/power-k/ui/pages/open-entity/workspaces-menu.tsx`、`web/apps/web/core/store/user/profile.store.ts` | `Workspace`、新 store 的使用方 | 4 |
| `web/apps/web/core/components/workspace/create-workspace-form.tsx` | `Workspace`（Task 4）；slug 检查改问 nerve，没有调用方传的两个 prop 删除（Task 5） | 4、5 |
| `web/apps/web/core/components/workspace/settings/members-list-item.tsx` | `Workspace`（Task 4）；离开改调新 store（Task 5）；`WorkspaceMember`、`MemberUser`（Task 7） | 4、5、7 |
| `web/packages/types/package.json`、`web/packages/utils/package.json`、`pnpm-lock.yaml`、`web/packages/utils/src/workspace.ts` | 两个包依赖 `@nerve/api-client`；`orderWorkspacesList` 用 `Workspace`（整个文件） | 4 |
| `web/packages/constants/src/workspace.ts` | `ORGANIZATION_SIZE` 的类型用生成的 `OrganizationSize`（Task 4）；`RESTRICTED_URLS` 删除（Task 5） | 4、5 |
| `web/apps/web/core/components/power-k/config/navigation/commands.ts`、`web/apps/web/core/components/power-k/menus/workspaces.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/web-hooks/create-webhook-modal.tsx`、`web/apps/web/core/components/web-hooks/utils.ts`、`web/apps/web/core/components/workspace/delete-workspace-modal.tsx`、`web/apps/web/core/components/workspace/delete-workspace-section.tsx`、`web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx` | `IWorkspace` → `Workspace` | 4 机械 |
| `web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/packages/types/src/search.ts`、`web/packages/types/src/project/projects.ts` | `IWorkspace` → `Workspace`（Task 4）；`IUserLite` → `MemberUser`（Task 7） | 4 机械、7 机械 |
| `web/apps/web/package.json` | oxlint 的上限 | 4、6、7、8、9、11 |
| `web/apps/web/core/lib/landing.ts`、`web/apps/web/core/lib/landing.test.ts` | `landingPath`：上次的工作区、最早创建的、没有时 `/create-workspace` | 5 |
| `web/apps/web/core/lib/use-landing.ts`、`web/apps/web/core/lib/use-landing.test.ts` | `useLanding`：已登录账户的落点的全部判断（谁落、取列表、列表失败、上次的工作区），`AuthenticationWrapper` 照它渲染 | 5 |
| `web/apps/web/core/lib/fake-session-swr.ts` | 测试取数的 hook 时代替 `use-session-swr.ts`（`handed`、`response`） | 5 |
| `web/apps/web/core/store/user/settings.store.ts`、`web/apps/web/core/hooks/store/user/user-user-settings.ts`、`web/apps/web/core/hooks/store/user/index.ts`、`web/apps/web/core/store/user/index.ts`、`web/apps/web/core/services/user.service.ts` | 设置 store、`currentUserSettings`、`leaveWorkspace` 删除 | 5 |
| `web/packages/types/src/users.ts` | `IUserSettings` 删除（Task 5）；`IUserLite` 删除（Task 7） | 5、7 |
| `web/apps/web/core/store/theme.store.ts`、`web/apps/web/core/components/settings/mobile/nav.tsx` | 设置页的侧边栏状态移到 `ThemeStore`（R4） | 5 |
| `web/apps/web/core/store/user/permissions.store.ts` | 不再读设置 store（Task 5）；工作区角色取自 `Workspace.role`，`workspace-members/me` 的取数删除（Task 6） | 5、6 |
| `web/apps/web/core/store/user/permissions.store.test.ts` | 与 9.2 同一组身份的工作区角色和权限 | 6 |
| `web/packages/constants/src/navigation.test.ts` | `RESTRICTED_URLS` 的测试随它删除 | 5 |
| `e2e/stories/identity/a3-sign-in.spec.ts` | 登录之后的落点：没有工作区时去 `/create-workspace`（3.14） | 5 |
| `web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts` | 工作区包装层挂载时的取数：列表（Task 6），成员（Task 8），显示设置（Task 10），后两者只在列表说明它是调用者的工作区之后 | 6、8、10 |
| `web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx` | 邀请的链接（Task 3）；改用 `useWorkspaceFetch`、`Workspace.role`（Task 6、8、10）；取收藏的 SWR 删除（Task 11） | 3、6、8、10、11 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx` | 页面读调用者的角色：`workspaceInfoBySlug` 改为 `getWorkspaceRoleByWorkspaceSlug` | 6 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx` | 同上（Task 6）；成员页的取数改用 `useMembersSettingsFetch`（Task 8） | 6、8 |
| `web/apps/web/core/components/workspace/invite-modal/fields.tsx`、`web/apps/web/core/store/issue/workspace-draft/issue.store.ts` | 读调用者的角色的使用方；没有调用方传的 `className` 删除 | 6 |
| `web/apps/web/core/components/workspace/settings/invitations-list-item.tsx` | 角色（Task 6）；`WorkspaceInvitation`（Task 8） | 6、8 |
| `web/apps/web/core/services/workspace/workspace-members.service.ts` | 工作区成员：列出、改角色、移出 | 7 |
| `web/apps/web/core/store/member/workspace/workspace-member.store.ts` | 成员（Task 7）、邀请（Task 8），按会话分代，修改经 `oneAtATime()` | 7、8 |
| `web/apps/web/core/store/member/workspace/workspace-member.store.test.ts`、`web/apps/web/core/store/member/workspace/fake-members.ts` | 成员 store 的测试；`membershipOf`、`memberStore` | 7 |
| `web/apps/web/core/store/member/index.ts`、`web/apps/web/core/store/member/utils.ts`、`web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts`、`web/apps/web/core/store/member/project/project-member.store.ts` | 成员的公开资料（`MemberUser`）、按角色和加入时间排序（`toSorted`）；没有读者的 `getMemberIds` 删除 | 7 机械、7 |
| `web/packages/typescript-config/react-router.json`、`web/apps/web/core/components/navigation/use-navigation-items.ts`、`web/apps/web/core/components/navigation/tab-navigation-root.tsx` | web 应用的 TypeScript `lib` 改为 ES2023；导航的两处排序改为 `toSorted`，抑制删除（裁定 A3） | 7 |
| `web/apps/web/core/components/profile/use-profile-member.ts` | `MemberUser`（Task 7 机械、Task 7）；成员的取数（Task 8） | 7 机械、7、8 |
| `web/apps/web/core/components/modules/links/list-item.tsx`、`web/apps/web/core/components/profile/sidebar.tsx`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx`、`web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`、`web/packages/utils/src/file.ts` | `MemberUser` 的使用方：可空的 `avatar_url`、`email` 按 `null` 处理（W20） | 7 |
| `web/apps/web/core/components/dropdowns/member/base.tsx`、`web/apps/web/core/components/dropdowns/member/member-options.tsx`、`web/apps/web/core/components/project/confirm-project-member-remove.tsx`、`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/store/issue/root.store.ts`、`web/apps/web/core/store/member/project/project-member-filters.store.ts`、`web/apps/web/core/store/notifications/notification.ts`、`web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts`、`web/packages/types/src/workspace-notifications.ts` | `IUserLite` → `MemberUser`（`root.store.ts` 另有 `IWorkspaceMembership` → `WorkspaceMember`） | 7 机械 |
| `web/apps/web/core/services/workspace/workspace-invitations.service.ts`、`web/apps/web/core/store/member/workspace/workspace-invitations.test.ts` | 邀请：列出、批量创建、改角色、删除；它们的 store 测试 | 8 |
| `web/apps/web/core/components/workspace/settings/use-members-settings-fetch.ts`、`web/apps/web/core/components/workspace/settings/use-members-settings-fetch.test.ts`、`web/apps/web/core/components/workspace/settings/members-list.tsx` | 成员页的取数：成员给每个人，邀请只给管理员（7.1；角色由 hook 从工作区列表读） | 8 |
| `.oxlintrc.json` | `overrides`：P8a 的会话取数不从 `swr` 导入值（`no-restricted-imports`）；P8a 的 M3 路径没有非空断言（`typescript/no-non-null-assertion`）（裁定 A6、L4） | 8 |
| `web/apps/web/core/components/workspace/members/invite-modal.tsx`、`web/apps/web/core/components/onboarding/steps/team/root.tsx` | 邀请的创建改调新 store | 8 |
| `web/apps/web/core/services/workspace/invitation-preview.service.ts`、`web/apps/web/core/services/workspace/invitation-preview.service.test.ts`、`web/apps/web/core/hooks/use-invitation-preview.ts` | 公开的邀请查看（`publicClient`） | 9 |
| `web/apps/web/app/(all)/workspace-invitations/page.tsx`、`web/apps/web/core/components/account/auth-forms/auth-header.tsx`、`web/apps/web/core/components/account/auth-forms/auth-root.tsx` | 邀请页（整个文件）和登录、注册页读查看的结果；接受、忽略经 `WorkspaceRootStore` | 9 |
| `web/apps/web/core/services/workspace/workspace-preferences.service.ts`、`web/apps/web/core/store/workspace/preferences.store.ts`、`web/apps/web/core/store/workspace/preferences.store.test.ts` | 工作区的显示设置（3.18） | 10 |
| `web/apps/web/core/hooks/use-navigation-preferences.ts`、`web/packages/types/src/navigation-preferences.ts`、`web/packages/types/src/view-props.ts`、`web/apps/web/core/services/issue_filter.service.ts` | 侧边栏的导航设置读写新 store（整个文件）；`TProjectNavigationMode`、`IWorkspaceUserPropertiesResponse` 和旧服务的注释块删除 | 10 |
| `web/apps/web/core/components/home/home-body.tsx`、`web/apps/web/core/components/home/widgets/index.ts`、`web/apps/web/core/components/home/widgets/empty-states/index.ts`、`web/apps/web/core/components/home/widgets/loaders/index.ts`、`web/apps/web/core/components/home/widgets/recents/filters.tsx`、`web/apps/web/core/components/home/widgets/recents/index.tsx`、`web/apps/web/core/components/home/widgets/recents/issue.tsx`、`web/apps/web/core/components/home/widgets/recents/project.tsx`、`web/apps/web/core/components/home/widgets/empty-states/recents.tsx`、`web/apps/web/core/components/home/widgets/loaders/recent-activity.tsx`、`web/apps/web/core/components/core/content-overflow-HOC.tsx`、`web/packages/types/src/home.ts`、`web/packages/types/src/index.ts`、`web/packages/i18n/src/locales/en/home.json`、`web/packages/i18n/src/locales/zh-CN/home.json` | 首页的"最近"小部件删除（3.1，M7 加回） | 11 |
| `web/apps/web/core/components/navigation/top-navigation-root.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx` | 未读通知数（M7）和迭代、模块、视图、分诊状态（M6、M7）的挂载时取数删除 | 11 |
| `web/apps/web/core/store/favorite.store.ts`、`web/apps/web/core/services/favorite/favorite.service.ts`、`web/apps/web/core/store/module.store.ts`、`web/apps/web/core/store/project-view.store.ts`、`web/apps/web/core/services/view.service.ts`、`web/apps/web/core/store/notifications/workspace-notifications.store.ts`、`web/apps/web/core/components/core/list/list-item.tsx` | 上面的删除留下的、没有调用方的方法和 prop 一并删除（M3 设计 3.2 的写法，M6、M7 加回）；两个 store 和列表项的 oxlint 警告清零 | 11 |
| `docs/v0/v0-design.md`、`docs/v0/frontend-changes.md` | 3.20 中 P8a 的行：总体设计 7.7（取数的原则、例外和静态检查，problem 码的文案表）；前端改动清单的 M3 行和五行新的差异 | 12 |

---

### Task 1: 会话分代的基础：`dispose()`、`sessionGuard()`、`sessionKey` 与 `useSessionSWR`

**Files:**
- Create: `web/apps/web/core/lib/in-session.test.ts`、`web/apps/web/core/lib/in-session.ts`、`web/apps/web/core/lib/session-key.test.ts`、`web/apps/web/core/lib/session-key.ts`、`web/apps/web/core/lib/use-session-swr.test.ts`、`web/apps/web/core/lib/use-session-swr.ts`
- Modify: `web/apps/web/core/components/api-token/token-list.tsx`、`web/apps/web/core/components/appearance/theme-switcher.tsx`、`web/apps/web/core/lib/store-context.test.ts`、`web/apps/web/core/lib/store-context.tsx`、`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`、`web/apps/web/core/store/project/project_filter.store.ts`、`web/apps/web/core/store/root.store.ts`

**Interfaces:**
- Produces（spec 2.1；M3 设计 7.1；总体设计 7.7）：
  - `core/lib/session-key.ts`：`type SessionKey = readonly [name: string, loginId: string, ...args: string[]]`；`sessionKey(session: SessionState, name: string, ...args: (string | undefined)[]): SessionKey | null`：标签页已登录（`signed-in`）且每个参数已知时给出 `[name, loginId, ...args]`，否则 `null`（不取）。
  - `core/lib/use-session-swr.ts`：`type SessionFetch = readonly [name: string, ...args: (string | undefined)[]]`；`useSessionSWR<T>(fetch: SessionFetch | null, fetcher: (...args: string[]) => Promise<T>, config?: SWRConfiguration<T>): SWRResponse<T>`：键是 `sessionKey(useSession(), ...fetch)`；SWR 调 fetcher 时，fetcher 收到键中的参数（不含名称和 `loginId`），交回 store 的 Promise，所以取数失败是 SWR 的 `error`，不是未处理的拒绝。`fetch` 为 `null` 时不取：页面按权限不取的东西就这样写（7.1）。
  - `core/lib/in-session.ts`：`sessionGuard(): () => boolean`：调用时取 `tokenManager.state.loginId`，返回的核对比较它与此刻的 `loginId`。`theme-switcher.tsx` 的闭包改用它（7.1 的搬移）。
  - `IProjectFilterStore.dispose: () => void`：`reaction(...)` 的返回值；`RootStore.dispose()` 调用它；`store-context.tsx` 的换代在建好新的一代之后调用退役一代的 `dispose()`。
  - 已有的两处手写的会话键改用 `useSessionSWR`：令牌列表（`["API_TOKENS", loginId]`）、`AuthenticationWrapper` 取当前用户（`["CURRENT_USER", loginId]`）；键的形状不变。
- Consumes：`useSession()`、`tokenManager`、`SessionState`（M2）。

**Tests:**（vitest）
- `session-key.test.ts`：`keys a fetch by its name, the session's loginId and its arguments`；`gives the same fetch in another session another key, so that the new session fetches it again`（另一个账户、同一账户的另一次登录）；`keys nothing unless the tab is signed in`（`starting`、`signed-out`，以及第一次刷新因暂时的原因失败的 `unavailable`）；`keys nothing while an argument is unknown`。
- `use-session-swr.test.ts`（SWR 和会话是替身，hook 作为普通函数运行）：`keys the fetch by the session, and its fetcher gives the store's Promise for the fetch's arguments`（交给 SWR 的键、fetcher 交回的正是 store 的那个 Promise、fetcher 收到的参数只有 `acme`）；`fetches nothing for no fetch, nor before the tab is signed in`。
- `in-session.test.ts`：`holds while the tab stays in the session the change was sent in, and fails once it is in another`（同一会话的 `unavailable` 仍算；换到 Y 之后为假；此刻新取的核对是 Y 的）；`fails once the tab signed out, and after a sign-in as the same account again`。
- `store-context.test.ts` 加 `releases a retired session's reactions: the address's next workspace reaches the new stores only`：X 换到 Y 之后两代共用路由 store；改路由的工作区，Y 的筛选 store 跟着走（清空搜索、建这个工作区的筛选），X 的不再运行（W3）。

- [ ] **Step 1: 会话的 SWR 键和 hook**

`web/apps/web/core/lib/session-key.test.ts`（新文件，34 行）：

````file web/apps/web/core/lib/session-key.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { sessionKey } from "./session-key";

const X = { status: "signed-in", loginId: "x" } as const;
const Y = { status: "signed-in", loginId: "y" } as const;

describe("sessionKey", () => {
  it("keys a fetch by its name, the session's loginId and its arguments", () => {
    expect(sessionKey(X, "WORKSPACE_MEMBERS", "acme")).toEqual(["WORKSPACE_MEMBERS", "x", "acme"]);
    expect(sessionKey(X, "WORKSPACES")).toEqual(["WORKSPACES", "x"]);
  });

  it("gives the same fetch in another session another key, so that the new session fetches it again", () => {
    expect(sessionKey(Y, "WORKSPACE_MEMBERS", "acme")).not.toEqual(sessionKey(X, "WORKSPACE_MEMBERS", "acme"));
    // another sign-in as the same account is another session too
    expect(sessionKey({ status: "signed-in", loginId: "x2" }, "WORKSPACES")).not.toEqual(sessionKey(X, "WORKSPACES"));
  });

  it("keys nothing unless the tab is signed in", () => {
    for (const status of ["starting", "signed-out"] as const) expect(sessionKey({ status }, "WORKSPACES")).toBeNull();
    // the first refresh failed for a passing reason: the account is not known to be signed in yet
    expect(sessionKey({ status: "unavailable", loginId: "x", retryAt: 1 }, "WORKSPACES")).toBeNull();
  });

  it("keys nothing while an argument is unknown", () => {
    expect(sessionKey(X, "WORKSPACE_MEMBERS", undefined)).toBeNull();
    expect(sessionKey(X, "PROJECT", "acme", undefined)).toBeNull();
  });
});
````

`web/apps/web/core/lib/session-key.ts`（新文件，21 行）：

````file web/apps/web/core/lib/session-key.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SessionState } from "@/lib/auth/token-manager";

/** The SWR key of a fetch of the tab's session: the fetch's name, the session's loginId, the fetch's arguments. */
export type SessionKey = readonly [name: string, loginId: string, ...args: string[]];

/**
 * The SWR key of a fetch the tab's session makes (M3 design 7.1), the one way one is written (useSessionSWR writes
 * them all): it carries the session's loginId, so that a new session, another account's or another sign-in's,
 * fetches again into its own stores instead of showing the answer of the session before. Null, which fetches
 * nothing, unless the tab is signed in and every argument is known.
 */
export function sessionKey(session: SessionState, name: string, ...args: (string | undefined)[]): SessionKey | null {
  const known = args.filter((arg): arg is string => arg !== undefined);
  if (session.status !== "signed-in" || session.loginId === undefined || known.length !== args.length) return null;
  return [name, session.loginId, ...known];
}
````

`web/apps/web/core/lib/use-session-swr.test.ts`（新文件，57 行）：

````file web/apps/web/core/lib/use-session-swr.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SessionState } from "@/lib/auth/token-manager";
import type { SessionKey } from "@/lib/session-key";

// useSessionSWR hands SWR the key and a fetcher; SWR and the session are stand-ins here, so the hook runs as a
// plain function, outside React. What the key is for each session is session-key.test.ts.

/** What the hook hands SWR: the key, the fetcher that SWR calls with the key, the configuration. */
type SWRCall = [key: SessionKey | null, fetcher: (key: SessionKey) => unknown, config?: unknown];

const tab = vi.hoisted((): { session: SessionState } => ({ session: { status: "starting" } }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => tab.session }));
const swr = vi.hoisted((): { calls: SWRCall[] } => ({ calls: [] }));
vi.mock("swr", () => ({
  default: (...call: SWRCall) => {
    swr.calls.push(call);
    return {};
  },
}));

const { useSessionSWR } = await import("./use-session-swr");

/** A store's fetch, which tells the arguments it was given; its Promise is what SWR must get. */
let given: string[][] = [];
const answer = Promise.resolve([]);
const fetchList = (...args: string[]) => {
  given.push(args);
  return answer;
};

beforeEach(() => {
  swr.calls = [];
  given = [];
  tab.session = { status: "signed-in", loginId: "x" };
});

describe("useSessionSWR", () => {
  it("keys the fetch by the session, and its fetcher gives the store's Promise for the fetch's arguments", () => {
    const config = { revalidateOnFocus: false };
    useSessionSWR(["WORKSPACE_MEMBERS", "acme"], fetchList, config);
    expect(swr.calls).toEqual([[["WORKSPACE_MEMBERS", "x", "acme"], expect.any(Function), config]]);
    expect(swr.calls[0]?.[1](["WORKSPACE_MEMBERS", "x", "acme"])).toBe(answer);
    expect(given).toEqual([["acme"]]);
  });

  it("fetches nothing for no fetch, nor before the tab is signed in", () => {
    useSessionSWR(null, fetchList);
    tab.session = { status: "starting" };
    useSessionSWR(["WORKSPACES"], fetchList);
    expect(swr.calls.map(([key]) => key)).toEqual([null, null]);
  });
});
````

`web/apps/web/core/lib/use-session-swr.ts`（新文件，29 行）：

````file web/apps/web/core/lib/use-session-swr.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import useSWR from "swr";
import type { SWRConfiguration, SWRResponse } from "swr";
import { useSession } from "@/lib/auth/use-session";
import { sessionKey } from "@/lib/session-key";
import type { SessionKey } from "@/lib/session-key";

/** A fetch of the tab's session: its name and its arguments, of which an unknown one fetches nothing yet. */
export type SessionFetch = readonly [name: string, ...args: (string | undefined)[]];

/**
 * SWR for a fetch the tab's session makes into its stores (M3 design 7.1), the one way a component makes one: the
 * key is sessionKey's, so it carries the session's loginId; null fetches nothing, which is how a page leaves out
 * what its role may not read. The fetcher is given the fetch's arguments, every one known by then, so that what it
 * fetches is what the key names; it gives the store's Promise, so that a failure is the error SWR returns, never an
 * unhandled rejection.
 */
export function useSessionSWR<T>(
  fetch: SessionFetch | null,
  fetcher: (...args: string[]) => Promise<T>,
  config?: SWRConfiguration<T>
): SWRResponse<T> {
  const session = useSession();
  return useSWR(fetch && sessionKey(session, ...fetch), ([, , ...args]: SessionKey) => fetcher(...args), config);
}
````

- [ ] **Step 2: `sessionGuard()`，`theme-switcher.tsx` 改用它**

`web/apps/web/core/components/appearance/theme-switcher.tsx`（修改，2 处）：

````old web/apps/web/core/components/appearance/theme-switcher.tsx
import { tokenManager } from "@/lib/auth/api-client";
````
````new web/apps/web/core/components/appearance/theme-switcher.tsx
import { sessionGuard } from "@/lib/in-session";
````
````old web/apps/web/core/components/appearance/theme-switcher.tsx
      const loginId = tokenManager.state.loginId;
      const inSession = () => tokenManager.state.loginId === loginId;
````
````new web/apps/web/core/components/appearance/theme-switcher.tsx
      const inSession = sessionGuard();
````

`web/apps/web/core/lib/in-session.test.ts`（新文件，39 行）：

````file web/apps/web/core/lib/in-session.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it, vi } from "vitest";
import type { SessionState } from "@/lib/auth/token-manager";

// The tab's session as the token manager has it, which the test moves from one account to another.
const tab = vi.hoisted(() => ({ state: { status: "signed-in", loginId: "x" } as SessionState }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: tab }));

const { sessionGuard } = await import("./in-session");

describe("sessionGuard", () => {
  it("holds while the tab stays in the session the change was sent in, and fails once it is in another", () => {
    tab.state = { status: "signed-in", loginId: "x" };
    const inSession = sessionGuard();
    expect(inSession()).toBe(true);
    // a refresh within the session, or a passing failure, keeps the session
    tab.state = { status: "unavailable", loginId: "x", retryAt: 1 };
    expect(inSession()).toBe(true);
    // another tab signs in as Y, and this one follows
    tab.state = { status: "signed-in", loginId: "y" };
    expect(inSession()).toBe(false);
    // the check is of the session the change was sent in: one taken now is Y's
    expect(sessionGuard()()).toBe(true);
  });

  it("fails once the tab signed out, and after a sign-in as the same account again", () => {
    tab.state = { status: "signed-in", loginId: "x" };
    const inSession = sessionGuard();
    tab.state = { status: "signed-out" };
    expect(inSession()).toBe(false);
    // a new sign-in is a new session: another loginId, though the account is the same
    tab.state = { status: "signed-in", loginId: "x2" };
    expect(inSession()).toBe(false);
  });
});
````

`web/apps/web/core/lib/in-session.ts`（新文件，17 行）：

````file web/apps/web/core/lib/in-session.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { tokenManager } from "@/lib/auth/api-client";

/**
 * Takes the tab's session as a component sends a change, and gives the check of it: whether the tab is still in
 * that session (v0 design 7.7, M3 design 7.1). A change sent before another tab moved this one to another account
 * may still succeed afterwards, and the component that sent it still gets the answer: it follows the answer
 * (navigates, says so, changes the page) only while the check holds, since the page is then the other account's.
 */
export function sessionGuard(): () => boolean {
  const loginId = tokenManager.state.loginId;
  return () => tokenManager.state.loginId === loginId;
}
````

- [ ] **Step 3: 换代时释放退役一代的反应**

`web/apps/web/core/lib/store-context.test.ts`（修改，1 处）：

````old web/apps/web/core/lib/store-context.test.ts
  });

  it("gives the code that reads the stores outside the components the RootStore of the session now", async () => {
````
````new web/apps/web/core/lib/store-context.test.ts
  });

  it("releases a retired session's reactions: the address's next workspace reaches the new stores only", async () => {
    const { context, signedIn, follow } = await load();
    await signedIn();
    const x = context.rootStore;
    await follow(Y);
    const y = context.rootStore;
    // The address's parameters are the page's: Y's stores go on with X's RouterStore, which X's project filters
    // followed (project_filter.store.ts).
    expect(y.router).toBe(x.router);
    x.projectRoot.projectFilter.updateSearchQuery("x");
    y.projectRoot.projectFilter.updateSearchQuery("y");
    y.router.setQuery({ workspaceSlug: "acme" });
    // Y's filters follow the new workspace (they start it and clear the search); X's no longer run.
    expect(y.projectRoot.projectFilter.searchQuery).toBe("");
    expect(y.projectRoot.projectFilter.getDisplayFiltersByWorkspaceSlug("acme")).toBeDefined();
    expect(x.projectRoot.projectFilter.searchQuery).toBe("x");
    expect(x.projectRoot.projectFilter.getDisplayFiltersByWorkspaceSlug("acme")).toBeUndefined();
  });

  it("gives the code that reads the stores outside the components the RootStore of the session now", async () => {
````

`web/apps/web/core/lib/store-context.tsx`（修改，2 处）：

````old web/apps/web/core/lib/store-context.tsx
 * old session's: nothing they still do reaches the new one. The language was the account's: the new session shows
 * the default until its profile sets it. The theme follows the session in StoreWrapper.
````
````new web/apps/web/core/lib/store-context.tsx
 * old session's: nothing they still do reaches the new one, and what they registered on the page's stores is
 * released (dispose). The language was the account's: the new session shows the default until its profile sets
 * it. The theme follows the session in StoreWrapper.
````
````old web/apps/web/core/lib/store-context.tsx
  rootStore = new RootStore(apiFor(next), rootStore);
````
````new web/apps/web/core/lib/store-context.tsx
  const retired = rootStore;
  rootStore = new RootStore(apiFor(next), retired);
  retired.dispose();
````

`web/apps/web/core/store/project/project_filter.store.ts`（修改，3 处）：

````old web/apps/web/core/store/project/project_filter.store.ts
  clearAllAppliedDisplayFilters: (workspaceSlug: string) => void;
````
````new web/apps/web/core/store/project/project_filter.store.ts
  clearAllAppliedDisplayFilters: (workspaceSlug: string) => void;
  /** Stops following the address's workspace: the RouterStore it follows outlives the session (RootStore.dispose). */
  dispose: () => void;
````
````old web/apps/web/core/store/project/project_filter.store.ts
  rootStore: RootStore;
````
````new web/apps/web/core/store/project/project_filter.store.ts
  rootStore: RootStore;
  dispose: () => void;
````
````old web/apps/web/core/store/project/project_filter.store.ts
    reaction(
````
````new web/apps/web/core/store/project/project_filter.store.ts
    this.dispose = reaction(
````

`web/apps/web/core/store/root.store.ts`（修改，1 处）：

````old web/apps/web/core/store/root.store.ts
  }
````
````new web/apps/web/core/store/root.store.ts
  }

  /**
   * Releases what the stores of this session registered on the page's stores, which go on with the next session
   * (v0 design 7.7, M3 design 7.1): the project filters' reaction to the address's workspace. store-context.tsx
   * calls it as the next session's RootStore takes over, so that nothing of a retired session runs again.
   */
  dispose(): void {
    this.projectRoot.projectFilter.dispose();
  }
````

- [ ] **Step 4: 两处已有的会话键改用 `useSessionSWR`**

`web/apps/web/core/components/api-token/token-list.tsx`（修改，4 处）：

````old web/apps/web/core/components/api-token/token-list.tsx
import { observer } from "mobx-react";
import useSWR from "swr";
````
````new web/apps/web/core/components/api-token/token-list.tsx
import { observer } from "mobx-react";
````
````old web/apps/web/core/components/api-token/token-list.tsx
import { useSession } from "@/lib/auth/use-session";
````
````new web/apps/web/core/components/api-token/token-list.tsx
import { useSessionSWR } from "@/lib/use-session-swr";
````
````old web/apps/web/core/components/api-token/token-list.tsx
  // store hooks
  const session = useSession();
````
````new web/apps/web/core/components/api-token/token-list.tsx
  // store hooks
````
````old web/apps/web/core/components/api-token/token-list.tsx
  const { error, isValidating, mutate } = useSWR(
    session.status === "signed-in" ? ["API_TOKENS", session.loginId] : null,
    () => fetchTokens(),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );
````
````new web/apps/web/core/components/api-token/token-list.tsx
  const { error, isValidating, mutate } = useSessionSWR(["API_TOKENS"], () => fetchTokens(), {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });
````

`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`（修改，3 处）：

````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
import { Navigate, useLocation, useSearchParams } from "react-router";
import useSWR from "swr";
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
import { Navigate, useLocation, useSearchParams } from "react-router";
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
import { useSession } from "@/lib/auth/use-session";
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
import { useSession } from "@/lib/auth/use-session";
import { useSessionSWR } from "@/lib/use-session-swr";
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  const { error, mutate } = useSWR(
    session.status === "signed-in" ? ["CURRENT_USER", session.loginId] : null,
    () => fetchCurrentUser(),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  const { error, mutate } = useSessionSWR(["CURRENT_USER"], () => fetchCurrentUser(), {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；web 的 oxlint 452 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过，包括上面的测试。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add web/apps/web/core/components/api-token/token-list.tsx web/apps/web/core/components/appearance/theme-switcher.tsx web/apps/web/core/lib/in-session.test.ts web/apps/web/core/lib/in-session.ts web/apps/web/core/lib/session-key.test.ts web/apps/web/core/lib/session-key.ts web/apps/web/core/lib/store-context.test.ts web/apps/web/core/lib/store-context.tsx web/apps/web/core/lib/use-session-swr.test.ts web/apps/web/core/lib/use-session-swr.ts web/apps/web/core/lib/wrappers/authentication-wrapper.tsx web/apps/web/core/store/project/project_filter.store.ts web/apps/web/core/store/root.store.ts
```
```bash
git commit -m "feat(M3/P8a): session generations release their reactions; one way to key a session's fetch

RootStore.dispose() releases what a generation registered on the
page's stores, the project filters' reaction to the address, and
store-context.tsx calls it on the generation it retires. sessionKey and
useSessionSWR are the one way a component keys a fetch of the session:
the key carries the loginId, null fetches nothing, and the fetcher gets
the key's arguments and gives the store's Promise. sessionGuard() moves
out of the theme switcher into core/lib/in-session.ts.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t1-no-dispose` | `store-context.tsx` 换代时不调用退役一代的 `dispose()` | `store-context.test.ts` | vitest |
| `t1-dispose-empty` | `RootStore.dispose()` 什么都不释放 | `store-context.test.ts` | vitest |
| `t1-guard-always` | `sessionGuard()` 的核对恒为真 | `in-session.test.ts` | vitest |
| `t1-guard-now` | `sessionGuard()` 在核对时才取会话，不是在发出修改时 | `in-session.test.ts` | vitest |
| `t1-key-no-login` | `sessionKey()` 的键不带 `loginId` | `session-key.test.ts` | vitest |
| `t1-key-signed-out` | `sessionKey()` 在没有登录时也给出键 | `session-key.test.ts` | vitest |
| `t1-key-unknown-arg` | `sessionKey()` 在参数未知时也给出键 | `session-key.test.ts` | vitest |
| `t1-theme-own-guard` | `theme-switcher.tsx` 应答之后不核对会话（`inSession` 恒为真） | 故事 A9 | 端到端 |
| `t1-hook-bypass` | `useSessionSWR` 不论会话都用同一个 `loginId` 作键 | `use-session-swr.test.ts` | vitest |
| `t1-hook-drops` | `useSessionSWR` 交给 SWR 的 fetcher 丢掉 store 的 Promise | `tsc`、`use-session-swr.test.ts` | 静态；vitest |
| `t1-hook-args` | `useSessionSWR` 的 fetcher 连 `loginId` 一起收到参数 | `use-session-swr.test.ts` | vitest |
| `t1-fetcher-drops` | 令牌列表的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |

---

### Task 2: problem 码的文案表移到 `core/lib/error-messages.ts` 和 `errors` 命名空间

**Files:**
- Create: `web/apps/web/core/lib/error-messages.test.ts`、`web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`
- Modify: `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/constants/namespaces.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Delete: `web/apps/web/helpers/authentication.helper.test.ts`
- 机械步骤（Step 1）改到：`web/apps/web/core/components/account/auth-forms/password.tsx`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`、`web/apps/web/core/components/api-token/delete-token-modal.tsx`、`web/apps/web/core/components/api-token/modal/form.tsx`、`web/apps/web/core/components/appearance/theme-switcher.tsx`、`web/apps/web/core/components/onboarding/root.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`web/apps/web/core/components/profile/start-of-week-preference.tsx`、`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`、`web/apps/web/core/components/settings/profile/content/pages/security.tsx`

**Interfaces:**
- Produces（spec 2.2；M3 设计 12 节约束 4）：
  - `core/lib/error-messages.ts`：`PROBLEM_MESSAGES`（契约的每个 `x-problem-codes` 的码 → `errors.<名>`）、`FIELD_ERROR_MESSAGES`（`FieldError.code` → `errors.field.<名>`）、`errorMessageKey(error)`、`fieldErrorKeys(error)`、`needsErrorBanner(error, fields)`，与原来在 `helpers/authentication.helper.ts` 的写法和行为相同，只是文案的键从 `auth.errors.*` 改为 `errors.*`。
  - `errors` 命名空间（`namespaces.ts` 加 `"errors"`）：`en`、`zh-CN` 两份 `errors.json`，内容是原来两份 `auth.json` 的 `errors` 对象，原样移来（P6 的两条 `sole_admin` 文案不变，P6 spec 第 3 节第 17 条）；`auth.json` 删去这个对象。
  - `helpers/authentication.helper.ts` 只留 `EPageTypes`、`EAuthModes`。
  - 之后的 Phase 声明新码时，把它加进 `core/lib/error-messages.ts` 和两份 `errors.json`。M3 设计第 12 节约束 4 已在 P8a 的修订（spec 和 plan 的同一个提交）中改指这里；Task 12 写进总体设计 7.7 的一条也写明这张表在 `core/lib/error-messages.ts`。
- Consumes：生成的 `FieldError`、`ApiError`、`SessionUnavailableError`（M2）。

**Tests:**（vitest）`core/lib/error-messages.test.ts`，即原来的 `helpers/authentication.helper.test.ts` 随表移来，16 个标题不变：`the message tables`（`read the contract: both lists are found in openapi.yaml`、`have a message for every problem code of the contract, and no other`、`have a message for every FieldError.code of the contract, and no other`、`point at messages that exist in English (sync-check keeps zh-CN the same)`）、`errorMessageKey`、`fieldErrorKeys`、`needsErrorBanner` 各组。表与契约在两个方向上都核对（少一个码、多一个码各失败，W14）。

- [ ] **Step 1: 机械步骤：读错误文案的 11 个组件改从新位置导入**

`$P8ATMP/t2_imports.py` 见"一次性脚本"。它只改导入行：10 个文件的 `from "@/helpers/authentication.helper";` 换成 `from "@/lib/error-messages";`；`password.tsx` 还要 `EAuthModes`，它的一行拆成两行。

Run（机械步骤）: `python3 $P8ATMP/t2_imports.py`
Expected: `11 files`。

`git diff --numstat` 必须恰好是：

```text
2	1	web/apps/web/core/components/account/auth-forms/password.tsx
1	1	web/apps/web/core/components/account/deactivate-account-modal.tsx
1	1	web/apps/web/core/components/api-token/delete-token-modal.tsx
1	1	web/apps/web/core/components/api-token/modal/form.tsx
1	1	web/apps/web/core/components/appearance/theme-switcher.tsx
1	1	web/apps/web/core/components/onboarding/root.tsx
1	1	web/apps/web/core/components/onboarding/steps/profile/root.tsx
1	1	web/apps/web/core/components/profile/start-of-week-preference.tsx
1	1	web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
1	1	web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
1	1	web/apps/web/core/components/settings/profile/content/pages/security.tsx
```

Run（机械步骤）: `shasum -a 256 web/apps/web/core/components/account/auth-forms/password.tsx web/apps/web/core/components/account/deactivate-account-modal.tsx web/apps/web/core/components/api-token/delete-token-modal.tsx web/apps/web/core/components/api-token/modal/form.tsx web/apps/web/core/components/appearance/theme-switcher.tsx web/apps/web/core/components/onboarding/root.tsx web/apps/web/core/components/onboarding/steps/profile/root.tsx web/apps/web/core/components/profile/start-of-week-preference.tsx web/apps/web/core/components/settings/profile/content/pages/general/form.tsx web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx web/apps/web/core/components/settings/profile/content/pages/security.tsx`
Expected: 每个文件的散列和行数（`wc -l`）与下表相同：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `20c1005dd4ae638f0581fcb7fa2a00b7294e477c2187af73902e35cfac6cf781` | 283 | `web/apps/web/core/components/account/auth-forms/password.tsx` |
| `740aad32bdcd0062a799eaa1d3683a6c0e5fbddd25f7877c5ca92abd32830891` | 93 | `web/apps/web/core/components/account/deactivate-account-modal.tsx` |
| `e5b7be49798f8cd894307111d0b308626ab0275640d0aad844dce4c701edd4bd` | 69 | `web/apps/web/core/components/api-token/delete-token-modal.tsx` |
| `73f54dc4ff6a2c0019771bccabeb61c2261f045e6e07141e7b5eac24563925d6` | 234 | `web/apps/web/core/components/api-token/modal/form.tsx` |
| `c4009e9a3b8cd48932278358bbeb0fd21f82923b65f7ff2cc3e5df3ca29641a8` | 92 | `web/apps/web/core/components/appearance/theme-switcher.tsx` |
| `c4806561d013f70ae24703e28f0e180fe937044d19c6a3be0f99b4297bafdd73` | 127 | `web/apps/web/core/components/onboarding/root.tsx` |
| `d9706167ba8313e8c0125aefdbe1858692a4070c7bc502b382c1189b6b9f9669` | 147 | `web/apps/web/core/components/onboarding/steps/profile/root.tsx` |
| `3aa7f1e7b90e8a9988af6157adc004ae0092b5002e49ee9d92395f733f989299` | 72 | `web/apps/web/core/components/profile/start-of-week-preference.tsx` |
| `59e014c234cbe45b7aa5cece5f43f431903dd27beb49919c80fc823c0f4dea4f` | 278 | `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx` |
| `96da453d107b79bb8129aa389a5841e480bf41638d03e0e4b3ace3bf2de974ce` | 101 | `web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx` |
| `127acb2f4712c0f0803fc9d4b286a52c99c6002bddbb129bcd002c746b985223` | 287 | `web/apps/web/core/components/settings/profile/content/pages/security.tsx` |

- [ ] **Step 2: 文案表和它的测试移到 `core/lib/error-messages.ts`**

`web/apps/web/core/lib/error-messages.test.ts`（新文件，154 行）：

````file web/apps/web/core/lib/error-messages.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError, SessionUnavailableError } from "@/lib/auth/token-manager";
import {
  FIELD_ERROR_MESSAGES,
  PROBLEM_MESSAGES,
  errorMessageKey,
  fieldErrorKeys,
  needsErrorBanner,
} from "./error-messages";

// The message tables against the API's contract (M2 design 3.11, 7.3, 9.4): api/dist/openapi.yaml is the
// bundled description the server is checked against, so a code added there fails here until it has a message.

const spec = readFileSync(new URL("../../../../../api/dist/openapi.yaml", import.meta.url), "utf8").split("\n");
const en = JSON.parse(
  readFileSync(new URL("../../../../packages/i18n/src/locales/en/errors.json", import.meta.url), "utf8")
) as Record<string, unknown>;

/** The items of every list under a line that matches `key` (block lists: "- item", deeper than the key). */
function listsUnder(key: RegExp): string[] {
  const items: string[] = [];
  spec.forEach((line, i) => {
    const head = key.exec(line);
    if (!head) return;
    const indent = line.length - line.trimStart().length;
    for (const next of spec.slice(i + 1)) {
      const item = /^(\s*)- (\S+)$/.exec(next);
      if (!item || item[1].length < indent) break;
      items.push(item[2]);
    }
  });
  return items;
}

const problemCodes = new Set(listsUnder(/^\s*x-problem-codes:\s*$/));
// FieldError.code's enum: the only "enum:" list under the FieldError schema.
const fieldErrorStart = spec.findIndex((line) => /^ {4}FieldError:$/.test(line));
const fieldCodes = (() => {
  const enumLine = spec.findIndex((line, i) => i > fieldErrorStart && /^\s+enum:$/.test(line));
  const codes: string[] = [];
  for (const line of spec.slice(enumLine + 1)) {
    const item = /^\s+- (\S+)$/.exec(line);
    if (!item) break;
    codes.push(item[1]);
  }
  return new Set(codes);
})();

/** The value at a dotted i18n key of en's errors namespace, or undefined. */
function english(key: string): unknown {
  return key.split(".").reduce<unknown>((node, part) => (node as Record<string, unknown> | undefined)?.[part], en);
}

describe("the message tables", () => {
  it("read the contract: both lists are found in openapi.yaml", () => {
    expect(problemCodes).toContain("identity.email_taken");
    expect(problemCodes).toContain("bad_request");
    expect(fieldCodes).toContain("common_password");
  });

  it("have a message for every problem code of the contract, and no other", () => {
    expect(new Set(Object.keys(PROBLEM_MESSAGES))).toEqual(problemCodes);
  });

  it("have a message for every FieldError.code of the contract, and no other", () => {
    expect(new Set(Object.keys(FIELD_ERROR_MESSAGES))).toEqual(fieldCodes);
  });

  it("point at messages that exist in English (sync-check keeps zh-CN the same)", () => {
    const keys = [...Object.values(PROBLEM_MESSAGES), ...Object.values(FIELD_ERROR_MESSAGES)];
    for (const key of [...keys, "errors.unknown", "errors.unreachable"]) {
      expect(typeof english(key), key).toBe("string");
    }
  });
});

const problem = (code: string) => ({ status: 400, code, title: "" });

describe("errorMessageKey", () => {
  it("gives the message of the problem's code", () => {
    expect(errorMessageKey(new ApiError(409, problem("identity.email_taken")))).toBe("errors.email_taken");
    expect(errorMessageKey(new ApiError(429, problem("rate_limited")))).toBe("errors.rate_limited");
  });

  it("gives the general message for a code it does not know, or an answer without a problem", () => {
    expect(errorMessageKey(new ApiError(418, problem("teapot")))).toBe("errors.unknown");
    expect(errorMessageKey(new ApiError(502, undefined))).toBe("errors.unknown");
    expect(errorMessageKey(new TypeError("Failed to fetch"))).toBe("errors.unknown");
    expect(errorMessageKey(new SessionChangedError())).toBe("errors.unknown");
  });

  it("says the server cannot be reached when the session is unavailable", () => {
    expect(errorMessageKey(new SessionUnavailableError(0))).toBe("errors.unreachable");
  });
});

describe("fieldErrorKeys", () => {
  it("gives each named field the message of its code", () => {
    const error = new ApiError(422, {
      status: 422,
      code: "validation_failed",
      title: "",
      errors: [
        { field: "password", code: "common_password", message: "is too common" },
        { field: "email", code: "required", message: "is required" },
      ],
    });
    expect(fieldErrorKeys(error)).toEqual({
      password: "errors.field.common_password",
      email: "errors.field.required",
    });
  });

  it("gives nothing for a problem without fields, or for another error", () => {
    expect(fieldErrorKeys(new ApiError(409, { status: 409, code: "identity.email_taken", title: "" }))).toEqual({});
    expect(fieldErrorKeys(new TypeError("Failed to fetch"))).toEqual({});
  });
});

describe("needsErrorBanner", () => {
  /** A problem that names these fields. */
  const naming = (...fields: string[]) =>
    new ApiError(422, {
      status: 422,
      code: "validation_failed",
      title: "",
      errors: fields.map((field) => ({ field, code: "required" as const, message: "is required" })),
    });
  const form = ["email", "password"];

  it("leaves the messages under the fields when the form has every field the error names", () => {
    expect(needsErrorBanner(naming("email"), form)).toBe(false);
    expect(needsErrorBanner(naming("email", "password"), form)).toBe(false);
  });

  it("shows the error above the form when it names a field the form does not have", () => {
    expect(needsErrorBanner(naming("first_name"), form)).toBe(true);
    expect(needsErrorBanner(naming("email", "first_name"), form)).toBe(true);
  });

  it("shows an error without fields above the form", () => {
    expect(needsErrorBanner(new ApiError(409, { status: 409, code: "identity.email_taken", title: "" }), form)).toBe(
      true
    );
    expect(needsErrorBanner(new TypeError("Failed to fetch"), form)).toBe(true);
  });
});
````

`web/apps/web/core/lib/error-messages.ts`（新文件，92 行）：

````file web/apps/web/core/lib/error-messages.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { FieldError } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { SessionUnavailableError } from "@/lib/auth/token-manager";

/**
 * The message of every problem code an operation lists in x-problem-codes of api/dist/openapi.yaml, which a
 * test keeps equal to the keys (M2 design 3.11, 7.3; M3 design 12 constraint 4: a phase that declares a code adds
 * its message here, in the errors namespace of both languages). Any other code, such as the unauthorized that an
 * operation needing a bearer token can also answer, gets errors.unknown from errorMessageKey.
 */
export const PROBLEM_MESSAGES: Readonly<Record<string, string>> = {
  bad_request: "errors.bad_request",
  payload_too_large: "errors.payload_too_large",
  rate_limited: "errors.rate_limited",
  internal_error: "errors.internal_error",
  validation_failed: "errors.validation_failed",
  server_busy: "errors.server_busy",
  forbidden: "errors.forbidden",
  "identity.signup_disabled": "errors.signup_disabled",
  "identity.email_taken": "errors.email_taken",
  "identity.invalid_credentials": "errors.invalid_credentials",
  "identity.account_deactivated": "errors.account_deactivated",
  "identity.refresh_token_invalid": "errors.refresh_token_invalid",
  "identity.current_password_incorrect": "errors.current_password_incorrect",
  "identity.api_token_not_found": "errors.api_token_not_found",
  "workspace.not_found": "errors.workspace_not_found",
  "workspace.creation_disabled": "errors.workspace_creation_disabled",
  "workspace.slug_taken": "errors.workspace_slug_taken",
  "workspace.member_not_found": "errors.workspace_member_not_found",
  "workspace.own_membership": "errors.workspace_own_membership",
  "workspace.sole_admin": "errors.workspace_sole_admin",
  "workspace.invitation_not_found": "errors.workspace_invitation_not_found",
  "workspace.invitation_responded": "errors.workspace_invitation_responded",
  "workspace.invitation_email_mismatch": "errors.workspace_invitation_email_mismatch",
  "project.identifier_taken": "errors.project_identifier_taken",
  "project.name_taken": "errors.project_name_taken",
  "project.not_found": "errors.project_not_found",
  "project.archived": "errors.project_archived",
  "project.member_not_found": "errors.project_member_not_found",
  "project.own_membership": "errors.project_own_membership",
  "project.role_too_high": "errors.project_role_too_high",
  "project.sole_admin": "errors.project_sole_admin",
  "project.state_name_taken": "errors.project_state_name_taken",
  "project.state_not_found": "errors.project_state_not_found",
  "project.state_last_in_group": "errors.project_state_last_in_group",
  "project.state_default": "errors.project_state_default",
  "project.label_name_taken": "errors.project_label_name_taken",
  "project.label_not_found": "errors.project_label_not_found",
};

/** The message of every FieldError.code, shown under the field it names. */
export const FIELD_ERROR_MESSAGES: Readonly<Record<FieldError["code"], string>> = {
  required: "errors.field.required",
  invalid_format: "errors.field.invalid_format",
  too_short: "errors.field.too_short",
  too_long: "errors.field.too_long",
  out_of_range: "errors.field.out_of_range",
  not_allowed: "errors.field.not_allowed",
  duplicate: "errors.field.duplicate",
  weak_password: "errors.field.weak_password",
  common_password: "errors.field.common_password",
  must_be_future: "errors.field.must_be_future",
  contains_url: "errors.field.contains_url",
};

/** The i18n key of the message for an error of a call to nerve. */
export function errorMessageKey(error: unknown): string {
  if (error instanceof ApiError) return PROBLEM_MESSAGES[error.problem?.code ?? ""] ?? "errors.unknown";
  if (error instanceof SessionUnavailableError) return "errors.unreachable";
  return "errors.unknown";
}

/** The i18n keys of the messages for the fields a problem names, by field. */
export function fieldErrorKeys(error: unknown): Partial<Record<string, string>> {
  if (!(error instanceof ApiError)) return {};
  return Object.fromEntries((error.problem?.errors ?? []).map((e) => [e.field, FIELD_ERROR_MESSAGES[e.code]]));
}

/**
 * Whether a form with the given fields shows an error of a call to nerve above it: unless every field the
 * error names is one of them, whose messages show under the fields. An error that names a field the form
 * does not have would otherwise show nothing.
 */
export function needsErrorBanner(error: unknown, fields: readonly string[]): boolean {
  const named = Object.keys(fieldErrorKeys(error));
  return named.length === 0 || named.some((field) => !fields.includes(field));
}
````

`web/apps/web/helpers/authentication.helper.test.ts`（删除）：

````delete web/apps/web/helpers/authentication.helper.test.ts
````

`web/apps/web/helpers/authentication.helper.ts`（整个文件，17 行）：

````whole web/apps/web/helpers/authentication.helper.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export enum EPageTypes {
  PUBLIC = "PUBLIC",
  NON_AUTHENTICATED = "NON_AUTHENTICATED",
  ONBOARDING = "ONBOARDING",
  AUTHENTICATED = "AUTHENTICATED",
}

export enum EAuthModes {
  SIGN_IN = "SIGN_IN",
  SIGN_UP = "SIGN_UP",
}
````

- [ ] **Step 3: `errors` 命名空间**

`web/packages/i18n/src/constants/namespaces.ts`（修改，1 处）：

````old web/packages/i18n/src/constants/namespaces.ts
  "empty-state",
````
````new web/packages/i18n/src/constants/namespaces.ts
  "empty-state",
  "errors",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "retry": "Try again"
    },
    "errors": {
      "bad_request": "The request was not understood. Check the fields and try again.",
      "payload_too_large": "The request is too large.",
      "rate_limited": "Too many attempts. Please try again later.",
      "internal_error": "Something went wrong on the server. Please try again.",
      "validation_failed": "Some fields are not valid.",
      "server_busy": "The server is busy. Please try again later.",
      "signup_disabled": "Sign-up is closed.",
      "email_taken": "An account with this email already exists.",
      "invalid_credentials": "The email or the password is wrong.",
      "account_deactivated": "This account is deactivated.",
      "refresh_token_invalid": "You are signed out. Please sign in again.",
      "current_password_incorrect": "The current password is wrong.",
      "api_token_not_found": "The token does not exist or has been revoked.",
      "workspace_not_found": "The workspace does not exist, or you are not a member of it.",
      "workspace_creation_disabled": "Creating workspaces is turned off on this server.",
      "workspace_slug_taken": "A workspace with this URL already exists.",
      "forbidden": "Your role does not allow this.",
      "workspace_member_not_found": "The member does not exist, or you cannot see the workspace.",
      "workspace_own_membership": "You cannot change your own membership.",
      "workspace_sole_admin": "The workspace would be left without an admin. Make another member an admin first.",
      "workspace_invitation_not_found": "The invitation does not exist, or its link is not valid.",
      "workspace_invitation_responded": "The invitation has been answered already.",
      "workspace_invitation_email_mismatch": "This invitation was sent to another email address.",
      "project_identifier_taken": "A project of this workspace already has this identifier.",
      "project_name_taken": "A project of this workspace already has this name.",
      "project_not_found": "The project does not exist, or you cannot see it.",
      "project_archived": "The project is archived. Restore it to change it.",
      "project_member_not_found": "The project member does not exist, or you cannot see the project.",
      "project_own_membership": "You cannot remove yourself from the project, nor change your own role in it unless you are a workspace admin.",
      "project_role_too_high": "The role is too high for you. Unless you are a workspace admin, you change only members whose project role is below yours, to roles below yours. You remove only members whose project role is not above yours.",
      "project_sole_admin": "The project would be left without an admin: its only admin cannot leave it, nor can his membership end while it has other members. Give the project another admin first, or delete it.",
      "project_state_name_taken": "A state of this project already has this name.",
      "project_state_not_found": "The state does not exist, cannot be changed through this API, or you cannot see its project.",
      "project_state_last_in_group": "This is the only state of its group, and every group keeps one. Add another state to the group first.",
      "project_state_default": "The default state cannot be deleted. Make another state the default first.",
      "project_label_name_taken": "A label of this project already has this name, in this case or another.",
      "project_label_not_found": "The label does not exist, or you cannot see its project.",
      "unreachable": "Cannot reach the server for now. Please try again.",
      "unknown": "Something went wrong. Please try again.",
      "field": {
        "required": "Required",
        "invalid_format": "The format is not valid",
        "too_short": "Too short",
        "too_long": "Too long",
        "out_of_range": "Out of range",
        "not_allowed": "Not allowed",
        "duplicate": "Given more than once",
        "weak_password": "Use 8–128 characters with an upper-case letter, a lower-case letter, a digit and a special character",
        "common_password": "This password is too common",
        "must_be_future": "Must be in the future",
        "contains_url": "Must not contain a web address"
      }
````
````new web/packages/i18n/src/locales/en/auth.json
      "retry": "Try again"
````

`web/packages/i18n/src/locales/en/errors.json`（新文件，56 行）：

````file web/packages/i18n/src/locales/en/errors.json
{
  "errors": {
    "bad_request": "The request was not understood. Check the fields and try again.",
    "payload_too_large": "The request is too large.",
    "rate_limited": "Too many attempts. Please try again later.",
    "internal_error": "Something went wrong on the server. Please try again.",
    "validation_failed": "Some fields are not valid.",
    "server_busy": "The server is busy. Please try again later.",
    "signup_disabled": "Sign-up is closed.",
    "email_taken": "An account with this email already exists.",
    "invalid_credentials": "The email or the password is wrong.",
    "account_deactivated": "This account is deactivated.",
    "refresh_token_invalid": "You are signed out. Please sign in again.",
    "current_password_incorrect": "The current password is wrong.",
    "api_token_not_found": "The token does not exist or has been revoked.",
    "workspace_not_found": "The workspace does not exist, or you are not a member of it.",
    "workspace_creation_disabled": "Creating workspaces is turned off on this server.",
    "workspace_slug_taken": "A workspace with this URL already exists.",
    "forbidden": "Your role does not allow this.",
    "workspace_member_not_found": "The member does not exist, or you cannot see the workspace.",
    "workspace_own_membership": "You cannot change your own membership.",
    "workspace_sole_admin": "The workspace would be left without an admin. Make another member an admin first.",
    "workspace_invitation_not_found": "The invitation does not exist, or its link is not valid.",
    "workspace_invitation_responded": "The invitation has been answered already.",
    "workspace_invitation_email_mismatch": "This invitation was sent to another email address.",
    "project_identifier_taken": "A project of this workspace already has this identifier.",
    "project_name_taken": "A project of this workspace already has this name.",
    "project_not_found": "The project does not exist, or you cannot see it.",
    "project_archived": "The project is archived. Restore it to change it.",
    "project_member_not_found": "The project member does not exist, or you cannot see the project.",
    "project_own_membership": "You cannot remove yourself from the project, nor change your own role in it unless you are a workspace admin.",
    "project_role_too_high": "The role is too high for you. Unless you are a workspace admin, you change only members whose project role is below yours, to roles below yours. You remove only members whose project role is not above yours.",
    "project_sole_admin": "The project would be left without an admin: its only admin cannot leave it, nor can his membership end while it has other members. Give the project another admin first, or delete it.",
    "project_state_name_taken": "A state of this project already has this name.",
    "project_state_not_found": "The state does not exist, cannot be changed through this API, or you cannot see its project.",
    "project_state_last_in_group": "This is the only state of its group, and every group keeps one. Add another state to the group first.",
    "project_state_default": "The default state cannot be deleted. Make another state the default first.",
    "project_label_name_taken": "A label of this project already has this name, in this case or another.",
    "project_label_not_found": "The label does not exist, or you cannot see its project.",
    "unreachable": "Cannot reach the server for now. Please try again.",
    "unknown": "Something went wrong. Please try again.",
    "field": {
      "required": "Required",
      "invalid_format": "The format is not valid",
      "too_short": "Too short",
      "too_long": "Too long",
      "out_of_range": "Out of range",
      "not_allowed": "Not allowed",
      "duplicate": "Given more than once",
      "weak_password": "Use 8–128 characters with an upper-case letter, a lower-case letter, a digit and a special character",
      "common_password": "This password is too common",
      "must_be_future": "Must be in the future",
      "contains_url": "Must not contain a web address"
    }
  }
}
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "retry": "重试"
    },
    "errors": {
      "bad_request": "请求无法识别，请检查填写的内容后重试。",
      "payload_too_large": "请求过大。",
      "rate_limited": "尝试次数过多，请稍后再试。",
      "internal_error": "服务器出错，请重试。",
      "validation_failed": "部分内容不符合要求。",
      "server_busy": "服务器繁忙，请稍后再试。",
      "signup_disabled": "注册已关闭。",
      "email_taken": "该邮箱已注册。",
      "invalid_credentials": "邮箱或密码错误。",
      "account_deactivated": "账户已停用。",
      "refresh_token_invalid": "登录已失效，请重新登录。",
      "current_password_incorrect": "当前密码错误。",
      "api_token_not_found": "令牌不存在或已被撤销。",
      "workspace_not_found": "工作区不存在，或你不是它的成员。",
      "workspace_creation_disabled": "本服务器已关闭创建工作区。",
      "workspace_slug_taken": "已有工作区使用这个地址。",
      "forbidden": "你的角色不能做这件事。",
      "workspace_member_not_found": "成员不存在，或你看不到这个工作区。",
      "workspace_own_membership": "不能修改自己的成员身份。",
      "workspace_sole_admin": "工作区会因此没有管理员。请先把另一位成员设为管理员。",
      "workspace_invitation_not_found": "邀请不存在，或链接无效。",
      "workspace_invitation_responded": "这份邀请已经回应过了。",
      "workspace_invitation_email_mismatch": "这份邀请发给了另一个邮箱。",
      "project_identifier_taken": "这个工作区已有项目使用这个标识。",
      "project_name_taken": "这个工作区已有项目使用这个名称。",
      "project_not_found": "项目不存在，或你看不到它。",
      "project_archived": "项目已归档，恢复之后才能修改。",
      "project_member_not_found": "项目成员不存在，或你看不到这个项目。",
      "project_own_membership": "不能把自己移出项目；不是工作区管理员时，也不能修改自己在项目里的角色。",
      "project_role_too_high": "角色太高。除非你是工作区管理员，否则你只能修改项目角色比你低的成员，并且只能改为比你低的角色。你只能移出项目角色不比你高的成员。",
      "project_sole_admin": "项目会因此没有管理员：它唯一的管理员不能离开它；它还有别的成员时，他的成员关系也不能结束。请先给项目另一位管理员，或者删除这个项目。",
      "project_state_name_taken": "这个项目里已有同名的状态。",
      "project_state_not_found": "状态不存在、不能经这个接口修改，或者你看不到它所在的项目。",
      "project_state_last_in_group": "这是它所在分组里唯一的状态，每个分组都要保留一个状态。请先给这个分组添加另一个状态。",
      "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
      "project_label_name_taken": "这个项目里已有同名的标签（不区分大小写）。",
      "project_label_not_found": "标签不存在，或者你看不到它所在的项目。",
      "unreachable": "暂时连不上服务器，请稍后再试。",
      "unknown": "出错了，请重试。",
      "field": {
        "required": "必填",
        "invalid_format": "格式不正确",
        "too_short": "太短",
        "too_long": "太长",
        "out_of_range": "超出范围",
        "not_allowed": "不允许",
        "duplicate": "重复出现",
        "weak_password": "密码需要 8–128 个字符，包含大写字母、小写字母、数字和特殊字符",
        "common_password": "密码太常见",
        "must_be_future": "必须是将来的时间",
        "contains_url": "不能包含网址"
      }
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "retry": "重试"
````

`web/packages/i18n/src/locales/zh-CN/errors.json`（新文件，56 行）：

````file web/packages/i18n/src/locales/zh-CN/errors.json
{
  "errors": {
    "bad_request": "请求无法识别，请检查填写的内容后重试。",
    "payload_too_large": "请求过大。",
    "rate_limited": "尝试次数过多，请稍后再试。",
    "internal_error": "服务器出错，请重试。",
    "validation_failed": "部分内容不符合要求。",
    "server_busy": "服务器繁忙，请稍后再试。",
    "signup_disabled": "注册已关闭。",
    "email_taken": "该邮箱已注册。",
    "invalid_credentials": "邮箱或密码错误。",
    "account_deactivated": "账户已停用。",
    "refresh_token_invalid": "登录已失效，请重新登录。",
    "current_password_incorrect": "当前密码错误。",
    "api_token_not_found": "令牌不存在或已被撤销。",
    "workspace_not_found": "工作区不存在，或你不是它的成员。",
    "workspace_creation_disabled": "本服务器已关闭创建工作区。",
    "workspace_slug_taken": "已有工作区使用这个地址。",
    "forbidden": "你的角色不能做这件事。",
    "workspace_member_not_found": "成员不存在，或你看不到这个工作区。",
    "workspace_own_membership": "不能修改自己的成员身份。",
    "workspace_sole_admin": "工作区会因此没有管理员。请先把另一位成员设为管理员。",
    "workspace_invitation_not_found": "邀请不存在，或链接无效。",
    "workspace_invitation_responded": "这份邀请已经回应过了。",
    "workspace_invitation_email_mismatch": "这份邀请发给了另一个邮箱。",
    "project_identifier_taken": "这个工作区已有项目使用这个标识。",
    "project_name_taken": "这个工作区已有项目使用这个名称。",
    "project_not_found": "项目不存在，或你看不到它。",
    "project_archived": "项目已归档，恢复之后才能修改。",
    "project_member_not_found": "项目成员不存在，或你看不到这个项目。",
    "project_own_membership": "不能把自己移出项目；不是工作区管理员时，也不能修改自己在项目里的角色。",
    "project_role_too_high": "角色太高。除非你是工作区管理员，否则你只能修改项目角色比你低的成员，并且只能改为比你低的角色。你只能移出项目角色不比你高的成员。",
    "project_sole_admin": "项目会因此没有管理员：它唯一的管理员不能离开它；它还有别的成员时，他的成员关系也不能结束。请先给项目另一位管理员，或者删除这个项目。",
    "project_state_name_taken": "这个项目里已有同名的状态。",
    "project_state_not_found": "状态不存在、不能经这个接口修改，或者你看不到它所在的项目。",
    "project_state_last_in_group": "这是它所在分组里唯一的状态，每个分组都要保留一个状态。请先给这个分组添加另一个状态。",
    "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
    "project_label_name_taken": "这个项目里已有同名的标签（不区分大小写）。",
    "project_label_not_found": "标签不存在，或者你看不到它所在的项目。",
    "unreachable": "暂时连不上服务器，请稍后再试。",
    "unknown": "出错了，请重试。",
    "field": {
      "required": "必填",
      "invalid_format": "格式不正确",
      "too_short": "太短",
      "too_long": "太长",
      "out_of_range": "超出范围",
      "not_allowed": "不允许",
      "duplicate": "重复出现",
      "weak_password": "密码需要 8–128 个字符，包含大写字母、小写字母、数字和特殊字符",
      "common_password": "密码太常见",
      "must_be_future": "必须是将来的时间",
      "contains_url": "不能包含网址"
    }
  }
}
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过（`en` 与 `zh-CN` 的 `errors.json` 键一致）；web 的 oxlint 452 条。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过；`error-messages.test.ts` 的 16 个标题都在，`authentication.helper.test.ts` 不再存在。

Run: `make e2e`
Expected: 70 个全部通过（A2、A3、A15 等读这些文案的故事照旧）。

- [ ] **Step 5: 提交**

```bash
git add web/apps/web/core/components/account/auth-forms/password.tsx web/apps/web/core/components/account/deactivate-account-modal.tsx web/apps/web/core/components/api-token/delete-token-modal.tsx web/apps/web/core/components/api-token/modal/form.tsx web/apps/web/core/components/appearance/theme-switcher.tsx web/apps/web/core/components/onboarding/root.tsx web/apps/web/core/components/onboarding/steps/profile/root.tsx web/apps/web/core/components/profile/start-of-week-preference.tsx web/apps/web/core/components/settings/profile/content/pages/general/form.tsx web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx web/apps/web/core/components/settings/profile/content/pages/security.tsx web/apps/web/core/lib/error-messages.test.ts web/apps/web/core/lib/error-messages.ts web/apps/web/helpers/authentication.helper.test.ts web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/constants/namespaces.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/en/errors.json web/packages/i18n/src/locales/zh-CN/auth.json web/packages/i18n/src/locales/zh-CN/errors.json
```
```bash
git commit -m "refactor(M3/P8a): the messages of nerve's problem codes move to core/lib and the errors namespace

PROBLEM_MESSAGES and FIELD_ERROR_MESSAGES, with errorMessageKey,
fieldErrorKeys and needsErrorBanner, move out of the sign-in helper to
core/lib/error-messages.ts, and their strings out of auth.json into an
errors namespace in both languages, unchanged. The test that keeps the
table equal to the contract's codes moves with them. The components
that read them import the new place; the helper keeps the page and auth
modes.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t2-code-missing` | `PROBLEM_MESSAGES` 少一个契约里的码 | `error-messages.test.ts` | vitest |
| `t2-code-extra` | `PROBLEM_MESSAGES` 多一个契约没有的码 | `error-messages.test.ts` | vitest |
| `t2-message-missing` | 一个码指向 `errors` 命名空间没有的文案 | `error-messages.test.ts` | vitest |
| `t2-zh-missing` | `zh-CN` 少一条 `en` 有的文案 | i18n 的 `check:sync` | 静态 |
| `t2-old-home` | 一个使用方又从登录的 helper 导入文案表 | `tsc` | 静态 |

---

### Task 3: 系统内接受删除：`/invitations` 页、它的入口、新手引导的"加入工作区"一步

**Files:**
- Modify: `server/internal/modules/workspace/domain/reserved_slugs.txt`、`tools/keywords.json`、`web/apps/web/app/routes/core.ts`、`web/apps/web/app/routes/navigation.test.ts`、`web/apps/web/core/components/onboarding/header.tsx`、`web/apps/web/core/components/onboarding/root.tsx`、`web/apps/web/core/components/onboarding/steps/root.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/index.ts`、`web/apps/web/core/components/onboarding/steps/workspace/root.tsx`、`web/apps/web/core/components/power-k/config/account-commands.ts`、`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`、`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/workspace.service.ts`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/power-k.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/power-k.json`、`web/packages/types/src/workspace.ts`
- Delete: `web/apps/web/app/(all)/invitations/layout.tsx`、`web/apps/web/app/(all)/invitations/page.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx`

**Interfaces:**
- Removes（spec 2.3；决策点 2；M3 设计 7.4、7.8）：
  - `/invitations` 页（`app/(all)/invitations/` 两个文件）和 `routes/core.ts` 中它的 `layout`、`route`。`/invitations` 之后落到与未知地址相同的地方：它是一个工作区的地址（3.10），由工作区的页面回答（"找不到工作区"）。
  - 它的入口：工作区菜单（`workspace-menu-root.tsx`）、个人设置的工作区列表（`workspace-options.tsx`）、命令面板的"工作区邀请"（`account-commands.ts` 和两份 `power-k.json` 的 `power_k.account_actions.workspace_invites`）、`WorkspaceAuthWrapper` 中"不是成员"界面的"Check pending invites"链接和那半句说明。
  - 新手引导：`WorkspaceJoinInvitesStep`（`join-invites.tsx`）、`ECreateOrJoinWorkspaceViews`；`onboarding/steps/workspace/root.tsx` 只渲染创建一步（整个文件），`OnboardingRoot`、`OnboardingStepRoot`、`OnboardingHeader` 的 `invitations` 删除（之前用于决定是否显示加入一步）。
  - 旧 `WorkspaceService` 的 `joinWorkspaces`、`userWorkspaceInvitations`，取数键 `USER_WORKSPACES_LIST`，只为它们存在的文案，每种语言 14 条：`common.json` 中 `/invitations` 页的 `workspace_invites`、`accept_and_join`、`join_a_workspace`、`no_pending_invites`、`please_select_at_least_one_invitation`、`we_see_that_someone_has_invited_you_to_join_a_workspace`、`you_can_see_here_if_someone_invites_you_to_a_workspace`、`back_to_home`、`go_home`，加入一步的 `onboarding.workspace.create_new`、`join_existing`、`join_title`、`no_invitations`；`power-k.json` 的一条（上一条）。
  - 服务端的保留名单"应用"一段删去 `invitations`（它是 `/invitations` 页的段；与 Task 5 的 vitest 一致，spec 第 3 节）。
- Adds：关键词规则 `in-system-invitations`（7.10）：`/invitations` 路由、`joinWorkspaces`、`getUserWorkspaceInvitations` 一类名称和 `/api/users/me/workspaces/invitations/`；不命中样例覆盖 `/workspace-invitations`、工作区的邀请接口和 P8b 的 `projects/invitations/`（项目一侧，负面样例）。

**Tests:**
- vitest：`navigation.test.ts` 加 `gives /invitations, the in-app accept's page that is gone, to the workspace of that name`（`/invitations`、`/login` 的叶子路由都是 `:workspaceSlug`）。
- Go：`make test` 中保留名单的测试（P1 的 `reserved_test.go`）照旧通过，名单少了一项。
- 关键词：规则自己的命中、不命中样例和 `files` 样例由 `make lint-web` 的守卫核对。

- [ ] **Step 1: 删除 `/invitations` 页和它的路由、入口**

`web/apps/web/app/(all)/invitations/layout.tsx`（删除）：

````delete web/apps/web/app/(all)/invitations/layout.tsx
````

`web/apps/web/app/(all)/invitations/page.tsx`（删除）：

````delete web/apps/web/app/(all)/invitations/page.tsx
````

`web/apps/web/app/routes/core.ts`（修改，1 处）：

````old web/apps/web/app/routes/core.ts
  layout("./(all)/onboarding/layout.tsx", [route("onboarding", "./(all)/onboarding/page.tsx")]),

  // Invitations
  layout("./(all)/invitations/layout.tsx", [route("invitations", "./(all)/invitations/page.tsx")]),
````
````new web/apps/web/app/routes/core.ts
  layout("./(all)/onboarding/layout.tsx", [route("onboarding", "./(all)/onboarding/page.tsx")]),
````

`web/apps/web/app/routes/navigation.test.ts`（修改，1 处）：

````old web/apps/web/app/routes/navigation.test.ts
    expect(reaches("/*/settings/projects/*/features")).toBe(false);
  });
````
````new web/apps/web/app/routes/navigation.test.ts
    expect(reaches("/*/settings/projects/*/features")).toBe(false);
  });

  it("gives /invitations, the in-app accept's page that is gone, to the workspace of that name", () => {
    // decision 2 (M3 design 7.8): no page of the app is /invitations any more; like /login, it is a workspace's
    // address (3.10), which the workspace's pages answer
    const leaf = (url: string) => matchRoutes(table, url)?.at(-1)?.route.path;
    expect(leaf("/invitations")).toBe(":workspaceSlug");
    expect(leaf("/login")).toBe(":workspaceSlug");
  });
````

`web/apps/web/core/components/power-k/config/account-commands.ts`（修改，4 处）：

````old web/apps/web/core/components/power-k/config/account-commands.ts
import { LogOutOutline, MailOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/power-k/config/account-commands.ts
import { LogOutOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/power-k/config/account-commands.ts
import { useUser } from "@/hooks/store/user";
import { useNavigate } from "react-router";
````
````new web/apps/web/core/components/power-k/config/account-commands.ts
import { useUser } from "@/hooks/store/user";
````
````old web/apps/web/core/components/power-k/config/account-commands.ts
export const usePowerKAccountCommands = (): TPowerKCommandConfig[] => {
  // navigation
  const navigate = useNavigate();
````
````new web/apps/web/core/components/power-k/config/account-commands.ts
export const usePowerKAccountCommands = (): TPowerKCommandConfig[] => {
````
````old web/apps/web/core/components/power-k/config/account-commands.ts
    {
      id: "workspace_invites",
      type: "action",
      group: "account",
      i18n_title: "power_k.account_actions.workspace_invites",
      icon: MailOutline,
      action: () => navigate("/invitations"),
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
    {
````
````new web/apps/web/core/components/power-k/config/account-commands.ts
    {
````

`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
import { MailOutline, PlusCircleOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
import { PlusCircleOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
          />
          <SettingsSidebarItem
            as="link"
            href="/invitations"
            icon={MailOutline}
            label={t("workspace_invites")}
            isActive={false}
          />
````
````new web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
          />
````

`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
import { ChevronDownOutline, LogOutOutline, MailOutline, PlusCircleOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
import { ChevronDownOutline, LogOutOutline, PlusCircleOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx

                    <Link to="/invitations" className="w-full" onClick={handleItemClick}>
                      <Menu.Item
                        as="div"
                        className="flex items-center gap-2 rounded-sm px-2 py-1 text-13 font-medium text-secondary hover:bg-layer-transparent-hover"
                      >
                        <MailOutline className="h-4 w-4 flex-shrink-0" />
                        {t("workspace_invites")}
                      </Menu.Item>
                    </Link>

````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx

````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，2 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                You're not a member of this workspace. Please contact the workspace admin to get an invitation or check
                your pending invitations.
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                You're not a member of this workspace. Please contact the workspace admin to get an invitation.
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
            <div className="flex items-center justify-center gap-2">
              <Link to="/invitations">
                <span>
                  <Button variant="secondary">Check pending invites</Button>
                </span>
              </Link>
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
            <div className="flex items-center justify-center gap-2">
````

`web/packages/i18n/src/locales/en/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/power-k.json
      "sign_out": "Sign out",
      "workspace_invites": "Workspace invites"
````
````new web/packages/i18n/src/locales/en/power-k.json
      "sign_out": "Sign out"
````

`web/packages/i18n/src/locales/zh-CN/power-k.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/power-k.json
      "sign_out": "退出登录",
      "workspace_invites": "工作区邀请"
````
````new web/packages/i18n/src/locales/zh-CN/power-k.json
      "sign_out": "退出登录"
````

- [ ] **Step 2: 删除新手引导的"加入工作区"一步**

`web/apps/web/core/components/onboarding/header.tsx`（修改，4 处）：

````old web/apps/web/core/components/onboarding/header.tsx
  updateCurrentStep: (step: EOnboardingSteps) => void;
  hasInvitations: boolean;
````
````new web/apps/web/core/components/onboarding/header.tsx
  updateCurrentStep: (step: EOnboardingSteps) => void;
````
````old web/apps/web/core/components/onboarding/header.tsx
  const { currentStep, updateCurrentStep, hasInvitations } = props;
````
````new web/apps/web/core/components/onboarding/header.tsx
  const { currentStep, updateCurrentStep } = props;
````
````old web/apps/web/core/components/onboarding/header.tsx
  // step order for progress tracking — include INVITE_MEMBERS if user is currently on it
  const showInviteStep = !hasInvitations || currentStep === EOnboardingSteps.INVITE_MEMBERS;
````
````new web/apps/web/core/components/onboarding/header.tsx
  // step order for progress tracking
````
````old web/apps/web/core/components/onboarding/header.tsx
    ...(showInviteStep ? [EOnboardingSteps.INVITE_MEMBERS] : []),
````
````new web/apps/web/core/components/onboarding/header.tsx
    EOnboardingSteps.INVITE_MEMBERS,
````

`web/apps/web/core/components/onboarding/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/onboarding/root.tsx
import type { IWorkspaceMemberInvitation, TOnboardingStep } from "@nerve/types";
````
````new web/apps/web/core/components/onboarding/root.tsx
import type { TOnboardingStep } from "@nerve/types";
````
````old web/apps/web/core/components/onboarding/root.tsx
type Props = {
  invitations?: IWorkspaceMemberInvitation[];
};

export const OnboardingRoot = observer(function OnboardingRoot({ invitations = [] }: Props) {
````
````new web/apps/web/core/components/onboarding/root.tsx
export const OnboardingRoot = observer(function OnboardingRoot() {
````
````old web/apps/web/core/components/onboarding/root.tsx
  const workspacesList = Object.values(workspaces ?? {});

  // Calculate total steps based on whether invitations are available
  const hasInvitations = invitations.length > 0;
````
````new web/apps/web/core/components/onboarding/root.tsx
  const workspacesList = Object.values(workspaces ?? {});
````
````old web/apps/web/core/components/onboarding/root.tsx
      <OnboardingHeader
        currentStep={currentStep}
        updateCurrentStep={updateCurrentStep}
        hasInvitations={hasInvitations}
      />
````
````new web/apps/web/core/components/onboarding/root.tsx
      <OnboardingHeader currentStep={currentStep} updateCurrentStep={updateCurrentStep} />
````
````old web/apps/web/core/components/onboarding/root.tsx
      <OnboardingStepRoot currentStep={currentStep} invitations={invitations} handleStepChange={handleStepChange} />
````
````new web/apps/web/core/components/onboarding/root.tsx
      <OnboardingStepRoot currentStep={currentStep} handleStepChange={handleStepChange} />
````

`web/apps/web/core/components/onboarding/steps/root.tsx`（修改，4 处）：

````old web/apps/web/core/components/onboarding/steps/root.tsx
// nerve imports
import type { IWorkspaceMemberInvitation } from "@nerve/types";
````
````new web/apps/web/core/components/onboarding/steps/root.tsx
// nerve imports
````
````old web/apps/web/core/components/onboarding/steps/root.tsx
  currentStep: EOnboardingSteps;
  invitations: IWorkspaceMemberInvitation[];
````
````new web/apps/web/core/components/onboarding/steps/root.tsx
  currentStep: EOnboardingSteps;
````
````old web/apps/web/core/components/onboarding/steps/root.tsx
function OnboardingStepContent({ currentStep, invitations, handleStepChange }: Props) {
````
````new web/apps/web/core/components/onboarding/steps/root.tsx
function OnboardingStepContent({ currentStep, handleStepChange }: Props) {
````
````old web/apps/web/core/components/onboarding/steps/root.tsx
      return <WorkspaceSetupStep invitations={invitations ?? []} handleStepChange={handleStepChange} />;
````
````new web/apps/web/core/components/onboarding/steps/root.tsx
      return <WorkspaceSetupStep handleStepChange={handleStepChange} />;
````

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（修改，3 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  onComplete: (skipInvites?: boolean) => void;
  handleCurrentViewChange: () => void;
  hasInvitations?: boolean;
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  onComplete: (skipInvites?: boolean) => void;
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
export const WorkspaceCreateStep = observer(function WorkspaceCreateStep({
  user,
  onComplete,
  handleCurrentViewChange,
  hasInvitations = false,
}: Props) {
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
export const WorkspaceCreateStep = observer(function WorkspaceCreateStep({ user, onComplete }: Props) {
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      <div className="flex flex-col gap-4">
        <Button variant="primary" type="submit" size="xl" className="w-full" disabled={isButtonDisabled}>
          {isSubmitting ? <Spinner height="20px" width="20px" /> : t("workspace_creation.button.default")}
        </Button>
        {hasInvitations && (
          <Button variant="ghost" size="xl" className="w-full" onClick={handleCurrentViewChange}>
            {t("onboarding.workspace.join_existing")}
          </Button>
        )}
      </div>
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      <Button variant="primary" type="submit" size="xl" className="w-full" disabled={isButtonDisabled}>
        {isSubmitting ? <Spinner height="20px" width="20px" /> : t("workspace_creation.button.default")}
      </Button>
````

`web/apps/web/core/components/onboarding/steps/workspace/index.ts`（修改，1 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/index.ts
export * from "./create";
export * from "./join-invites";
````
````new web/apps/web/core/components/onboarding/steps/workspace/index.ts
export * from "./create";
````

`web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx`（删除）：

````delete web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx
````

`web/apps/web/core/components/onboarding/steps/workspace/root.tsx`（整个文件，29 行）：

````whole web/apps/web/core/components/onboarding/steps/workspace/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { EOnboardingSteps } from "@nerve/types";
// hooks
import { useUser } from "@/hooks/store/user";
// local components
import { WorkspaceCreateStep } from "./";

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => void;
};

export const WorkspaceSetupStep = observer(function WorkspaceSetupStep({ handleStepChange }: Props) {
  // store hooks
  const { data: user } = useUser();

  return (
    <WorkspaceCreateStep
      user={user}
      onComplete={(skipInvites) => handleStepChange(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN, skipInvites)}
    />
  );
});
````

- [ ] **Step 3: 旧的 service 方法、取数键、类型和文案；保留名单；关键词规则**

`server/internal/modules/workspace/domain/reserved_slugs.txt`（修改，2 处）：

````old server/internal/modules/workspace/domain/reserved_slugs.txt
#   加上 web/apps/web/public/ 的顶层目录。核对它的 web vitest 随前端的数据层加入（M3 设计 9.5）。
#   invitations 是 /invitations 页的段：删掉这一页的改动（决策点 2）同时把它移出名单。
````
````new server/internal/modules/workspace/domain/reserved_slugs.txt
#   加上 web/apps/web/public/ 的顶层目录。核对它的 web vitest 随前端的数据层加入（M3 设计 9.5）。
````
````old server/internal/modules/workspace/domain/reserved_slugs.txt
create-workspace
invitations
````
````new server/internal/modules/workspace/domain/reserved_slugs.txt
create-workspace
````

`tools/keywords.json`（修改，1 处）：

````old tools/keywords.json
          "miss": ["docs/v0/M2-auth/M2-design.md", "server/internal/modules/instance/app/list_timezones.go"]
        }
      }
    }
````
````new tools/keywords.json
          "miss": ["docs/v0/M2-auth/M2-design.md", "server/internal/modules/instance/app/list_timezones.go"]
        }
      }
    },
    {
      "id": "in-system-invitations",
      "phase": "M3/P8a",
      "why": "系统内接受邀请（决策点 2，M3 设计 7.8）：/invitations 页、它的路由和入口、新手引导的\"加入工作区\"一步，以及按邮箱列出、批量接受邀请的旧接口和方法都已删除；接受邀请只经邀请链接 /workspace-invitations。/workspace-invitations 和工作区的邀请接口不在其内",
      "files": {
        "source": "^web/(?!.*\\.test\\.[jt]sx?$).*\\.(?:[cm]?[jt]sx?|json)$",
        "flags": ""
      },
      "content": {
        "source": "[\"'`]/invitations(?![\\w-])|route\\(\"invitations\"|\\b(?:joinWorkspaces|userWorkspaceInvitations|getUserWorkspaceInvitations|WorkspaceJoinInvitesStep|ECreateOrJoinWorkspaceViews)\\b|/api/users/me/workspaces/invitations/",
        "flags": ""
      },
      "samples": {
        "hit": [
          "              <Link to=\"/invitations\">",
          "      action: () => navigate(\"/invitations\"),",
          "  layout(\"./(all)/invitations/layout.tsx\", [route(\"invitations\", \"./(all)/invitations/page.tsx\")]),",
          "  async joinWorkspaces(data: any): Promise<any> {",
          "    return this.get(\"/api/users/me/workspaces/invitations/\")",
          "import { WorkspaceCreateStep, WorkspaceJoinInvitesStep } from \"./\";"
        ],
        "miss": [
          "    route(\"workspace-invitations\", \"./(all)/workspace-invitations/page.tsx\"),",
          "  return `/workspace-invitations?invitation_id=${id}&token=${token}`;",
          "    return this.post(`/api/workspaces/${workspaceSlug}/invitations/`, data)",
          "    return unwrap(await this.api.GET(\"/api/v0/workspaces/{slug}/invitations\", { params: { path: { slug } } }));",
          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })"
        ],
        "files": {
          "hit": ["web/apps/web/app/routes/core.ts", "web/apps/web/core/components/power-k/config/account-commands.ts"],
          "miss": [
            "docs/v0/M3-workspace-project/M3-design.md",
            "server/internal/modules/workspace/domain/reserved_slugs.txt",
            "web/apps/web/app/routes/navigation.test.ts"
          ]
        }
      }
    }
````

`web/apps/web/core/services/workspace.service.ts`（修改，2 处）：

````old web/apps/web/core/services/workspace.service.ts

  async joinWorkspaces(data: any): Promise<any> {
    return this.post("/api/users/me/workspaces/invitations/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts

````
````old web/apps/web/core/services/workspace.service.ts
    return this.get("/api/users/last-visited-workspace/")
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async userWorkspaceInvitations(): Promise<IWorkspaceMemberInvitation[]> {
    return this.get("/api/users/me/workspaces/invitations/")
````
````new web/apps/web/core/services/workspace.service.ts
    return this.get("/api/users/last-visited-workspace/")
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts
};

export const USER_WORKSPACES_LIST = "USER_WORKSPACES_LIST";
````
````new web/packages/constants/src/fetch-keys.ts
};
````

`web/packages/i18n/src/locales/en/common.json`（修改，3 处）：

````old web/packages/i18n/src/locales/en/common.json
  "sign_out": "Sign out",
  "workspace_invites": "Workspace invites",
````
````new web/packages/i18n/src/locales/en/common.json
  "sign_out": "Sign out",
````
````old web/packages/i18n/src/locales/en/common.json
  "version": "Version",
  "please_select_at_least_one_invitation": "Please select at least one invitation.",
  "we_see_that_someone_has_invited_you_to_join_a_workspace": "We see that someone has invited you to join a workspace",
  "join_a_workspace": "Join a workspace",
  "accept_and_join": "Accept & Join",
  "go_home": "Go Home",
  "no_pending_invites": "No pending invites",
  "you_can_see_here_if_someone_invites_you_to_a_workspace": "You can see here if someone invites you to a workspace",
  "back_to_home": "Back to home",
````
````new web/packages/i18n/src/locales/en/common.json
  "version": "Version",
````
````old web/packages/i18n/src/locales/en/common.json
      "join_existing": "Join existing workspace",
      "creation_disabled": "You don't seem to have any invites to a workspace and your instance admin has restricted creation of new workspaces. Please ask a workspace owner or admin to invite you to a workspace first and come back to this screen to join.",
      "join_title": "Join invites or create a workspace",
      "create_new": "Create new workspace",
      "no_invitations": "No Invitations found"
````
````new web/packages/i18n/src/locales/en/common.json
      "creation_disabled": "You don't seem to have any invites to a workspace and your instance admin has restricted creation of new workspaces. Please ask a workspace owner or admin to invite you to a workspace first and come back to this screen to join."
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，3 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "sign_out": "退出登录",
  "workspace_invites": "工作区邀请",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "sign_out": "退出登录",
````
````old web/packages/i18n/src/locales/zh-CN/common.json
  "version": "版本",
  "please_select_at_least_one_invitation": "请至少选择一个邀请。",
  "we_see_that_someone_has_invited_you_to_join_a_workspace": "我们看到有人邀请您加入工作区",
  "join_a_workspace": "加入工作区",
  "accept_and_join": "接受并加入",
  "go_home": "返回首页",
  "no_pending_invites": "没有待处理的邀请",
  "you_can_see_here_if_someone_invites_you_to_a_workspace": "如果有人邀请您加入工作区，您可以在这里看到",
  "back_to_home": "返回首页",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "version": "版本",
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "join_existing": "加入已有的工作区",
      "creation_disabled": "您似乎还没有收到任何工作区的邀请，而实例管理员限制了创建新工作区。请先请工作区的所有者或管理员邀请您加入工作区，再回到这里加入。",
      "join_title": "接受邀请，或创建工作区",
      "create_new": "创建新工作区",
      "no_invitations": "没有找到邀请"
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "creation_disabled": "您似乎还没有收到任何工作区的邀请，而实例管理员限制了创建新工作区。请先请工作区的所有者或管理员邀请您加入工作区，再回到这里加入。"
````

`web/packages/types/src/workspace.ts`（修改，1 处）：

````old web/packages/types/src/workspace.ts

export enum ECreateOrJoinWorkspaceViews {
  WORKSPACE_CREATE = "WORKSPACE_CREATE",
  WORKSPACE_JOIN = "WORKSPACE_JOIN",
}

````
````new web/packages/types/src/workspace.ts

````

- [ ] **Step 4: 运行检查**

Run: `make lint-go`
Expected: 两段都输出 `0 issues.`。

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 61 条规则，没有命中；web 的 oxlint 452 条。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

完成时：本 Task 删除的 14 个文案键（上面 Interfaces 中列出的 `common.json` 13 条和 `power-k.json` 的 `power_k.account_actions.workspace_invites`）在 `en`、`zh-CN` 中都已不在（spec 附录 A.7 的键表；`check:sync` 只核对两种语言一致，两种语言都留下的键要评审照键表核对）。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/workspace/domain/reserved_slugs.txt tools/keywords.json 'web/apps/web/app/(all)/invitations/layout.tsx' 'web/apps/web/app/(all)/invitations/page.tsx' web/apps/web/app/routes/core.ts web/apps/web/app/routes/navigation.test.ts web/apps/web/core/components/onboarding/header.tsx web/apps/web/core/components/onboarding/root.tsx web/apps/web/core/components/onboarding/steps/root.tsx web/apps/web/core/components/onboarding/steps/workspace/create.tsx web/apps/web/core/components/onboarding/steps/workspace/index.ts web/apps/web/core/components/onboarding/steps/workspace/join-invites.tsx web/apps/web/core/components/onboarding/steps/workspace/root.tsx web/apps/web/core/components/power-k/config/account-commands.ts web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/workspace.service.ts web/packages/constants/src/fetch-keys.ts web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/power-k.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/power-k.json web/packages/types/src/workspace.ts
```
```bash
git commit -m "feat(M3/P8a): the in-app accept of invitations is gone

Decision 2 keeps one way to accept an invitation, its link. The
/invitations page goes with its route and every way to it: the
workspace menu, the profile's workspace list, the command palette and
the not-a-member screen. Onboarding loses its join step, the old
service its two methods, and the reserved names the page's segment. A
keyword rule keeps them out.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t3-route-back` | `/invitations` 的路由加回 | 关键词守卫、`navigation.test.ts` | 静态；vitest |
| `t3-link-back` | 个人设置的侧边栏又链接到 `/invitations` | 关键词守卫 | 静态 |
| `t3-method-back` | 旧 service 又按邮箱列出调用者收到的邀请 | 关键词守卫 | 静态 |

---

### Task 4: 工作区的类型、service 和 store；`IWorkspace` 的使用方；工作区图标的上传删除

**Files:**
- Create: `web/apps/web/core/services/workspace/workspaces.service.ts`、`web/apps/web/core/store/fake-queue.ts`、`web/apps/web/core/store/fake-root.ts`、`web/apps/web/core/store/workspace/fake-workspaces.ts`、`web/apps/web/core/store/workspace/index.test.ts`
- Modify: `pnpm-lock.yaml`、`tools/keywords.json`、`web/apps/web/app/(all)/create-workspace/page.tsx`、`web/apps/web/core/components/onboarding/root.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/power-k/ui/pages/open-entity/workspaces-menu.tsx`、`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/core/components/workspace/delete-workspace-form.tsx`、`web/apps/web/core/components/workspace/settings/members-list-item.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`、`web/apps/web/core/lib/auth/fake-nerve.ts`、`web/apps/web/core/services/file.service.ts`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/root.store.ts`、`web/apps/web/core/store/user/profile.store.ts`、`web/apps/web/core/store/workspace/index.ts`、`web/apps/web/package.json`、`web/packages/constants/src/file.ts`、`web/packages/constants/src/workspace.ts`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`、`web/packages/types/package.json`、`web/packages/types/src/workspace.ts`、`web/packages/utils/package.json`、`web/packages/utils/src/workspace.ts`
- Delete: `web/apps/web/core/components/core/modals/workspace-image-upload-modal.tsx`
- 机械步骤（Step 1）改到：`web/apps/web/core/components/power-k/config/navigation/commands.ts`、`web/apps/web/core/components/power-k/menus/workspaces.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/components/web-hooks/create-webhook-modal.tsx`、`web/apps/web/core/components/web-hooks/utils.ts`、`web/apps/web/core/components/workspace/delete-workspace-modal.tsx`、`web/apps/web/core/components/workspace/delete-workspace-section.tsx`、`web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx`、`web/packages/types/src/project/projects.ts`、`web/packages/types/src/search.ts`

**Interfaces:**
- Produces（spec 2.4；M3 设计 7.2、7.3）：
  - `core/services/workspace/workspaces.service.ts`：`class WorkspacesService { constructor(api: ApiClient); list(): Promise<Workspace[]>; create(data: WorkspaceCreate): Promise<Workspace>; update(slug, data: WorkspaceUpdate): Promise<Workspace>; delete(slug): Promise<void> }`，每个方法一次调用生成的客户端（`unwrap`），`list` 交回 `WorkspaceList.data`（nerve 的顺序：名称、id）。
  - `core/store/workspace/index.ts`（整个文件）：`IWorkspaceRootStore` 的 `workspaces: Workspace[] | undefined`（没有取过时 `undefined`）、`currentWorkspace`（路由的 slug 指的、调用者是成员的工作区）、`getWorkspaceBySlug`、`fetchWorkspaces(): Promise<Workspace[] | undefined>`（`SessionChangedError` 时给出 `undefined`、不改列表；其他失败原样抛出、列表不变）、`createWorkspace`、`updateWorkspace`、`deleteWorkspace`：都经 `changes = oneAtATime()`，nerve 回答之后才改列表（创建的放在已取的列表末尾，修改的换成回答，删除的移出），被拒绝时什么都不改。`WorkspaceService` 的显示设置的三个方法暂时留在这里，原样（Task 10 换掉）。`WorkspaceRootStore(_rootStore, api)`：`RootStore` 传这一代的客户端。
  - `Workspace` 取代 `IWorkspace`（7.2）：`owner` 没有了；`logo_url` 可空，读作 `null`；`organization_size` 是 `OrganizationSize | null`，`ORGANIZATION_SIZE` 的类型随之改为 `OrganizationSize[]`；`role` 是调用者在工作区里的角色。`ILastActiveWorkspaceDetails`、`IWorkspace` 从 `packages/types` 删除；`orderWorkspacesList` 收 `readonly Workspace[]`，交回新的有序列表（整个文件）。
  - `@nerve/types`、`@nerve/utils` 依赖 `@nerve/api-client`（`workspace:*`），锁文件随之两处。
  - 工作区图标的上传删除（spec 第 3 节：`WorkspaceUpdate` 没有 `logo_url`，3.2 的规则 2）：`workspace-image-upload-modal.tsx`、general 页的上传按钮、`FileService.deleteWorkspaceAsset`、`ACCEPTED_AVATAR_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE` 和两条文案；显示图标的地方不改（没有图标时显示首字母）。
  - 旧 `WorkspaceService` 删去 `userWorkspaces`、`getWorkspace`、`createWorkspace`、`updateWorkspace`、`deleteWorkspace`、`getLastActiveWorkspaceAndProjects`。
  - 测试的共用部分：`core/store/fake-queue.ts` 的 `inTurn(nerve, k, [method, path], answer)`（第 k 个请求是这次修改，前面的请求有了回答它才发出）和 `fetchedWhileChangeIsOut(nerve, change, fetch, [method, path], answer)`（一个修改发出、还没有回答时取数：取数紧跟着发出、先得到回答，之后那个修改被拒绝；工作区、成员（Task 7）、显示设置（Task 10）三个 store 的"取数不排队"都用它）；`core/store/fake-root.ts` 的 `fakeRoot(siblings)`（被测 store 读了没给的兄弟 store 时测试失败；它的一处 `as` 见 spec 第 3 节）；`core/store/workspace/fake-workspaces.ts` 的 `workspaceOf(slug, fields)`、`loadWorkspaces(nerve, store, workspaces)`；`FakeNerve.replacedSessionClient()`：每个请求在发出之前以 `SessionChangedError` 失败。
- Adds：关键词规则 `plane-workspace-urls`（7.10），只写到本 Phase 换掉的地址为止；不命中样例有项目一侧的地址（P8b 换掉时补进模式）和 `/search-issues/`。

**Tests:**（vitest）`core/store/workspace/index.test.ts`：
- `WorkspaceRootStore, the list`：`lists the caller's workspaces as nerve gives them, and finds them by slug and by the address`；`fails when nerve cannot list them, keeping the list it had`；`fetches the list while a change is out: a fetch does not wait for it`（`fetchedWhileChangeIsOut`）；`gives nothing, and does not fail, when the session changes as it fetches`（`replacedSessionClient`）。
- `WorkspaceRootStore, the changes`：`adds a created workspace to the list it has, and to none it has not fetched`；`puts nerve's answer to a change in the list, in the changed workspace's place`（回答与请求不同，store 存回答）；`takes a deleted workspace off the list`；`fails, changing nothing, when nerve refuses $change`（创建、修改、删除各一，`it.each`）；`sends each change once nerve has answered the one before it, refused or not`（`inTurn`）。

- [ ] **Step 1: 机械步骤：其余 11 个文件的 `IWorkspace` 换成 `Workspace`**

`$P8ATMP/rename_type.py` 见"一次性脚本"。9 个应用的文件从 `@nerve/types` 导入它，`types` 包里的两个文件用相对路径导入它；之后 oxfmt 排版这些文件。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IWorkspace Workspace web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/workspaces.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/web-hooks/create-webhook-modal.tsx web/apps/web/core/components/web-hooks/utils.ts web/apps/web/core/components/workspace/delete-workspace-modal.tsx web/apps/web/core/components/workspace/delete-workspace-section.tsx web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx`
Expected: `IWorkspace -> Workspace in 9 files`。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IWorkspace Workspace --from ./workspace web/packages/types/src/search.ts`
Expected: `IWorkspace -> Workspace in 1 files`。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IWorkspace Workspace --from ../workspace web/packages/types/src/project/projects.ts`
Expected: `IWorkspace -> Workspace in 1 files`。

Run（机械步骤）: `pnpm exec oxfmt web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/workspaces.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/web-hooks/create-webhook-modal.tsx web/apps/web/core/components/web-hooks/utils.ts web/apps/web/core/components/workspace/delete-workspace-modal.tsx web/apps/web/core/components/workspace/delete-workspace-section.tsx web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx web/packages/types/src/search.ts web/packages/types/src/project/projects.ts`
Expected: 退出码 0。

`git diff --numstat` 必须恰好是：

```text
3	2	web/apps/web/core/components/power-k/config/navigation/commands.ts
3	3	web/apps/web/core/components/power-k/menus/workspaces.tsx
4	3	web/apps/web/core/components/project/form.tsx
3	2	web/apps/web/core/components/project/project-settings-member-defaults.tsx
3	2	web/apps/web/core/components/web-hooks/create-webhook-modal.tsx
3	2	web/apps/web/core/components/web-hooks/utils.ts
2	2	web/apps/web/core/components/workspace/delete-workspace-modal.tsx
2	2	web/apps/web/core/components/workspace/delete-workspace-section.tsx
4	4	web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx
4	4	web/packages/types/src/search.ts
2	2	web/packages/types/src/project/projects.ts
```

Run（机械步骤）: `shasum -a 256 web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/workspaces.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/web-hooks/create-webhook-modal.tsx web/apps/web/core/components/web-hooks/utils.ts web/apps/web/core/components/workspace/delete-workspace-modal.tsx web/apps/web/core/components/workspace/delete-workspace-section.tsx web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx web/packages/types/src/search.ts web/packages/types/src/project/projects.ts`
Expected: 每个文件的散列和行数与下表相同：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `65adb199162939a1e3d57aef5071b696aa4d67e3ab71e66aa20e18a3ea647c77` | 465 | `web/apps/web/core/components/power-k/config/navigation/commands.ts` |
| `adf905bdbacd8e84d2464a1d2707d3e32ed64878524d8276875cbda2ec35ef21` | 34 | `web/apps/web/core/components/power-k/menus/workspaces.tsx` |
| `3f7a404c1bee99e1517493d041369e8887dae984fe566d5b89641e78c109acdd` | 462 | `web/apps/web/core/components/project/form.tsx` |
| `1ae70f22769949154f82f06eb73bcd958c0b5edee8171558b304ecff5fe2408d` | 204 | `web/apps/web/core/components/project/project-settings-member-defaults.tsx` |
| `d225a65158406d8fe97a6ff6907d7a0d15c3dbfd05d0a77ae09d471a2ad4d0af` | 117 | `web/apps/web/core/components/web-hooks/create-webhook-modal.tsx` |
| `1fb8b1463df13d2ac2b5219221443a08ad3f1e0913a2e3b7fb6690961d571ffe` | 29 | `web/apps/web/core/components/web-hooks/utils.ts` |
| `bcd1566e2345ee043e10fdc825167670b9135218807cceaccfec8a4dac7b026f` | 30 | `web/apps/web/core/components/workspace/delete-workspace-modal.tsx` |
| `c82b8bfc8fcaea18bd6d744d483f76b373ad9b022b1794114b91f3b2f6ef6a1c` | 47 | `web/apps/web/core/components/workspace/delete-workspace-section.tsx` |
| `f2c07ef86996de2eb38ad122dce5abd2c89e92f624b155a441df6c9ba6c21ae4` | 124 | `web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx` |
| `732c9179b15fbfb0b325bd01d810e10e702b9032bd5910cd744bcefa8178f314` | 73 | `web/packages/types/src/search.ts` |
| `22274bce76c8257cf4628625af86df074c3be63932a83615b7e357df438f4ca3` | 112 | `web/packages/types/src/project/projects.ts` |

- [ ] **Step 2: 两个包依赖生成的客户端；`Workspace` 取代 `IWorkspace`**

`pnpm-lock.yaml`（修改，2 处）：

````old pnpm-lock.yaml
  web/packages/types:
    dependencies:
````
````new pnpm-lock.yaml
  web/packages/types:
    dependencies:
      '@nerve/api-client':
        specifier: workspace:*
        version: link:../api-client
````
````old pnpm-lock.yaml
    dependencies:
      '@makeplane/propel':
        specifier: 'catalog:'
        version: 0.3.0(@date-fns/tz@1.4.1)(@types/react@19.2.17)(date-fns@4.1.0)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)(tailwindcss@4.1.17)
      '@nerve/constants':
        specifier: workspace:*
````
````new pnpm-lock.yaml
    dependencies:
      '@makeplane/propel':
        specifier: 'catalog:'
        version: 0.3.0(@date-fns/tz@1.4.1)(@types/react@19.2.17)(date-fns@4.1.0)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)(tailwindcss@4.1.17)
      '@nerve/api-client':
        specifier: workspace:*
        version: link:../api-client
      '@nerve/constants':
        specifier: workspace:*
````

`web/packages/constants/src/workspace.ts`（修改，2 处）：

````old web/packages/constants/src/workspace.ts
 */

import type { TStaticViewTypes, IWorkspaceSearchResults } from "@nerve/types";
````
````new web/packages/constants/src/workspace.ts
 */

import type { OrganizationSize } from "@nerve/api-client";
import type { TStaticViewTypes, IWorkspaceSearchResults } from "@nerve/types";
````
````old web/packages/constants/src/workspace.ts
export const ORGANIZATION_SIZE: string[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];
````
````new web/packages/constants/src/workspace.ts
export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];
````

`web/packages/types/package.json`（修改，1 处）：

````old web/packages/types/package.json
    "fix:format": "oxfmt ."
  },
````
````new web/packages/types/package.json
    "fix:format": "oxfmt ."
  },
  "dependencies": {
    "@nerve/api-client": "workspace:*"
  },
````

`web/packages/types/src/workspace.ts`（修改，3 处）：

````old web/packages/types/src/workspace.ts
import type { TUserPermissions } from "./enums";
import type { TProjectMembership } from "./project";
````
````new web/packages/types/src/workspace.ts
import type { TUserPermissions } from "./enums";
````
````old web/packages/types/src/workspace.ts
  GUEST = 5,
}

export interface IWorkspace {
  readonly id: string;
  readonly owner: IUserLite;
  readonly created_at: Date;
  readonly updated_at: Date;
  name: string;
  url: string;
  logo_url: string | null;
  readonly total_members: number;
  readonly slug: string;
  readonly created_by: string;
  readonly updated_by: string;
  organization_size: string;
  total_projects?: number;
  role: number;
  timezone: string;
````
````new web/packages/types/src/workspace.ts
  GUEST = 5,
````
````old web/packages/types/src/workspace.ts

export interface ILastActiveWorkspaceDetails {
  workspace_details: IWorkspace;
  project_details?: TProjectMembership[];
}

````
````new web/packages/types/src/workspace.ts

````

`web/packages/utils/package.json`（修改，1 处）：

````old web/packages/utils/package.json
    "@makeplane/propel": "catalog:",
````
````new web/packages/utils/package.json
    "@makeplane/propel": "catalog:",
    "@nerve/api-client": "workspace:*",
````

`web/packages/utils/src/workspace.ts`（整个文件，12 行）：

````whole web/packages/utils/src/workspace.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { Workspace } from "@nerve/api-client";

/** The workspaces by name, as the sidebar lists them: a new list, the one given unchanged. */
export const orderWorkspacesList = (workspaces: readonly Workspace[]): Workspace[] =>
  workspaces.toSorted((a, b) => a.name.localeCompare(b.name));
````

- [ ] **Step 3: service、store 和测试的共用部分**

`web/apps/web/core/lib/auth/fake-nerve.ts`（修改，2 处）：

````old web/apps/web/core/lib/auth/fake-nerve.ts
import type { AuthTokens } from "@nerve/api-client";
````
````new web/apps/web/core/lib/auth/fake-nerve.ts
import type { AuthTokens } from "@nerve/api-client";
import { SessionChangedError } from "./token-manager";
````
````old web/apps/web/core/lib/auth/fake-nerve.ts
  }

  /** The requests to path so far. */
````
````new web/apps/web/core/lib/auth/fake-nerve.ts
  }

  /**
   * A client of this fake for a session another has replaced as the tab's: each request fails before it is sent,
   * with SessionChangedError, as the session's middleware fails it then (auth-middleware.ts).
   */
  replacedSessionClient() {
    const api = this.client();
    api.use({
      onRequest: () => {
        throw new SessionChangedError();
      },
    });
    return api;
  }

  /** The requests to path so far. */
````

`web/apps/web/core/services/workspace.service.ts`（修改，4 处）：

````old web/apps/web/core/services/workspace.service.ts
import type {
  IWorkspace,
````
````new web/apps/web/core/services/workspace.service.ts
import type {
````
````old web/apps/web/core/services/workspace.service.ts
  IWorkspaceMemberInvitation,
  ILastActiveWorkspaceDetails,
````
````new web/apps/web/core/services/workspace.service.ts
  IWorkspaceMemberInvitation,
````
````old web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
  async userWorkspaces(): Promise<IWorkspace[]> {
    return this.get("/api/users/me/workspaces/")
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getWorkspace(workspaceSlug: string): Promise<IWorkspace> {
    return this.get(`/api/workspaces/${workspaceSlug}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async createWorkspace(data: Partial<IWorkspace>): Promise<IWorkspace> {
    return this.post("/api/workspaces/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateWorkspace(workspaceSlug: string, data: Partial<IWorkspace>): Promise<IWorkspace> {
    return this.patch(`/api/workspaces/${workspaceSlug}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteWorkspace(workspaceSlug: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
````
````old web/apps/web/core/services/workspace.service.ts
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getLastActiveWorkspaceAndProjects(): Promise<ILastActiveWorkspaceDetails> {
    return this.get("/api/users/last-visited-workspace/")
````
````new web/apps/web/core/services/workspace.service.ts
    })
````

`web/apps/web/core/services/workspace/workspaces.service.ts`（新文件，32 行）：

````file web/apps/web/core/services/workspace/workspaces.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's workspaces (M3 design 5.1, 7.3). */
export class WorkspacesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The workspaces of which the caller is an active member, each with the caller's role, by name and then by id. */
  async list(): Promise<Workspace[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces")).data;
  }

  /** Creates a workspace with the caller as its admin. */
  async create(data: WorkspaceCreate): Promise<Workspace> {
    return unwrap(await this.api.POST("/api/v0/workspaces", { body: data }));
  }

  /** Changes the fields data names; the answer is the workspace as nerve now holds it. */
  async update(slug: string, data: WorkspaceUpdate): Promise<Workspace> {
    return unwrap(await this.api.PATCH("/api/v0/workspaces/{slug}", { params: { path: { slug } }, body: data }));
  }

  async delete(slug: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } } }));
  }
}
````

`web/apps/web/core/store/fake-queue.ts`（新文件，45 行）：

````file web/apps/web/core/store/fake-queue.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The queue of a store's changes (oneAtATime, v0 design 7.7), for the tests of the stores that queue them, against
// a fake nerve and fake timers.

import { expect, vi } from "vitest";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";

/** The k-th request is the change sent, the last one out until nerve gives it this answer. */
export async function inTurn(nerve: FakeNerve, k: number, [method, path]: [string, string], answer: Response) {
  await until(() => nerve.calls.length === k + 1, `request ${k}`);
  await vi.advanceTimersByTimeAsync(1_000);
  expect(nerve.calls).toHaveLength(k + 1);
  expect(nerve.calls[k]).toMatchObject({ method, path });
  nerve.calls[k]?.answer(answer);
}

/**
 * A fetch while a change is out (fetches do not queue): the change is sent and left unanswered, the fetch is sent,
 * checked against [method, path] and given answer, and only then is the change refused. The test then checks what the
 * store holds: the fetch's answer.
 */
export async function fetchedWhileChangeIsOut(
  nerve: FakeNerve,
  change: () => Promise<unknown>,
  fetch: () => Promise<unknown>,
  [method, path]: [string, string],
  answer: Response
) {
  const at = nerve.calls.length;
  const changed = track(change());
  await until(() => nerve.calls.length === at + 1, "the change");
  const fetched = track(fetch());
  await until(() => nerve.calls.length === at + 2, "the fetch");
  expect(nerve.calls[at + 1]).toMatchObject({ method, path });
  nerve.calls[at + 1]?.answer(answer);
  await until(() => fetched.settled, "the fetch");
  nerve.calls[at]?.answer(problem(403, "forbidden"));
  await until(() => changed.settled, "the refusal");
}
````

`web/apps/web/core/store/fake-root.ts`（新文件，20 行）：

````file web/apps/web/core/store/fake-root.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The RootStore of the tests of one store: the sibling stores that store reads, and no other. The stores take the
// whole RootStore; a store that reads a sibling the test did not give fails the test, instead of reading undefined.

import type { RootStore } from "@/store/root.store";

export function fakeRoot(siblings: Partial<RootStore>): RootStore {
  const root = new Proxy(siblings, {
    get(target, name, receiver) {
      if (!Reflect.has(target, name)) throw new Error(`the store under test read root.${String(name)}, not given`);
      return Reflect.get(target, name, receiver);
    },
  });
  // The one cast of the tests' roots: a Partial is not a RootStore, and the proxy answers for the rest by failing.
  return root as RootStore;
}
````

`web/apps/web/core/store/root.store.ts`（修改，1 处）：

````old web/apps/web/core/store/root.store.ts
    this.workspaceRoot = new WorkspaceRootStore(this);
````
````new web/apps/web/core/store/root.store.ts
    this.workspaceRoot = new WorkspaceRootStore(this, api);
````

`web/apps/web/core/store/user/profile.store.ts`（修改，1 处）：

````old web/apps/web/core/store/user/profile.store.ts
    const firstWorkspace = Object.values(this.store.workspaceRoot.workspaces ?? {})[0];
````
````new web/apps/web/core/store/user/profile.store.ts
    const firstWorkspace = this.store.workspaceRoot.workspaces?.[0];
````

`web/apps/web/core/store/workspace/fake-workspaces.ts`（新文件，38 行）：

````file web/apps/web/core/store/workspace/fake-workspaces.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The caller's workspaces, for the tests of the stores that read them, against a fake nerve.

import type { Workspace } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { json } from "@/lib/auth/fake-nerve";
import { settle, until } from "@/lib/auth/fake-time";
import type { IWorkspaceRootStore } from "@/store/workspace";

/** A workspace of the caller's, as nerve lists it: the slug names it; a member's, unless fields say otherwise. */
export function workspaceOf(slug: string, fields: Partial<Workspace> = {}): Workspace {
  return {
    id: `id-${slug}`,
    name: slug,
    slug,
    organization_size: null,
    timezone: "UTC",
    logo_url: null,
    role: 15,
    total_members: 1,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** The store fetches the caller's workspaces, and nerve lists these. */
export async function loadWorkspaces(nerve: FakeNerve, store: IWorkspaceRootStore, workspaces: Workspace[]) {
  const at = nerve.calls.length;
  const fetched = store.fetchWorkspaces();
  await until(() => nerve.calls.length === at + 1, "the list");
  nerve.calls[at]?.answer(json(200, { data: workspaces }));
  return settle(fetched, "the list");
}
````

`web/apps/web/core/store/workspace/index.test.ts`（新文件，197 行）：

````file web/apps/web/core/store/workspace/index.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's workspaces (M3 design 7.3), against a fake nerve that answers each request when the test says. That
// the store sends as the session of its RootStore is root.store.test.ts.

const LIST = "/api/v0/workspaces";
const acme = workspaceOf("acme", { role: 20 });
const beta = workspaceOf("beta");

function setUp() {
  const nerve = new FakeNerve();
  const router = new RouterStore();
  const store = new WorkspaceRootStore(fakeRoot({ router }), nerve.client());
  return { nerve, router, store };
}

/** A store whose list nerve gave as acme and beta. */
async function loaded() {
  const { nerve, router, store } = setUp();
  await loadWorkspaces(nerve, store, [acme, beta]);
  return { nerve, router, store };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspaceRootStore, the list", () => {
  it("lists the caller's workspaces as nerve gives them, and finds them by slug and by the address", async () => {
    const { nerve, router, store } = setUp();
    expect(store.workspaces).toBeUndefined();

    const fetched = await loadWorkspaces(nerve, store, [acme, beta]);
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: LIST });
    expect(fetched.value).toEqual([acme, beta]);
    expect(store.workspaces).toEqual([acme, beta]);
    expect(store.getWorkspaceBySlug("beta")).toEqual(beta);
    expect(store.getWorkspaceBySlug("gamma")).toBeNull();

    expect(store.currentWorkspace).toBeNull();
    router.setQuery({ workspaceSlug: "acme" });
    expect(store.currentWorkspace).toEqual(acme);
    router.setQuery({ workspaceSlug: "gamma" });
    expect(store.currentWorkspace).toBeNull();
  });

  it("fails when nerve cannot list them, keeping the list it had", async () => {
    const { nerve, store } = setUp();
    const first = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 1, "the list");
    nerve.calls[0]?.answer(problem(503, "server_busy"));
    await until(() => first.settled, "the failure");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.workspaces).toBeUndefined();

    await loadWorkspaces(nerve, store, [acme, beta]);
    const again = track(store.fetchWorkspaces());
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.fail();
    await until(() => again.settled, "the failure");
    // The failure is the caller's to handle (SWR's error), not an unhandled rejection.
    expect(again.error).toBeInstanceOf(TypeError);
    expect(store.workspaces).toEqual([acme, beta]);
  });

  it("fetches the list while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      () => store.updateWorkspace("acme", { name: "Acme Inc" }),
      () => store.fetchWorkspaces(),
      ["GET", LIST],
      json(200, { data: [acme, beta] })
    );
    expect(store.workspaces).toEqual([acme, beta]);
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const nerve = new FakeNerve();
    const store = new WorkspaceRootStore(fakeRoot({ router: new RouterStore() }), nerve.replacedSessionClient());

    const fetched = await settle(store.fetchWorkspaces(), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.workspaces).toBeUndefined();
  });
});

describe("WorkspaceRootStore, the changes", () => {
  it("adds a created workspace to the list it has, and to none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const body: WorkspaceCreate = { name: "Gamma", slug: "gamma", organization_size: "2-10" };
    const gamma = workspaceOf("gamma", { name: "Gamma", organization_size: "2-10", role: 20 });
    const created = track(store.createWorkspace(body));
    await until(() => nerve.calls.length === 2, "the creation");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: LIST, body });
    // until nerve answers, the list is as it was
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(json(201, gamma));
    await until(() => created.settled, "the new workspace");
    expect(created.value).toEqual(gamma);
    expect(store.workspaces).toEqual([acme, beta, gamma]);

    const fresh = setUp();
    const alone = track(fresh.store.createWorkspace(body));
    await until(() => fresh.nerve.calls.length === 1, "the creation");
    fresh.nerve.calls[0]?.answer(json(201, gamma));
    await until(() => alone.settled, "the new workspace");
    // a list of the new workspace alone would show as the whole list
    expect(fresh.store.workspaces).toBeUndefined();
  });

  it("puts nerve's answer to a change in the list, in the changed workspace's place", async () => {
    const { nerve, store } = await loaded();
    const renamed = { ...acme, name: "Acme Inc", updated_at: "2026-10-07T09:00:00Z" };
    const updated = track(store.updateWorkspace("acme", { name: "Acme Inc" }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/workspaces/acme",
      body: { name: "Acme Inc" },
    });
    nerve.calls[1]?.answer(json(200, renamed));
    await until(() => updated.settled, "the answer");
    expect(updated.value).toEqual(renamed);
    expect(store.workspaces).toEqual([renamed, beta]);
  });

  it("takes a deleted workspace off the list", async () => {
    const { nerve, store } = await loaded();
    const deleted = track(store.deleteWorkspace("acme"));
    await until(() => nerve.calls.length === 2, "the deletion");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: "/api/v0/workspaces/acme" });
    // until nerve answers, the workspace is still the caller's
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(noContent());
    await until(() => deleted.settled, "the deletion");
    expect(deleted.error).toBeUndefined();
    expect(store.workspaces).toEqual([beta]);
  });

  const refusals: { change: string; send: (store: WorkspaceRootStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createWorkspace({ name: "A", slug: "acme" }),
      refusal: problem(409, "workspace.slug_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateWorkspace("beta", { name: "B" }),
      refusal: problem(403, "forbidden"),
    },
    { change: "a deletion", send: (store) => store.deleteWorkspace("beta"), refusal: problem(403, "forbidden") },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const sent = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => sent.settled, "the refusal");
    expect(sent.error).toBeInstanceOf(ApiError);
    expect(store.workspaces).toEqual([acme, beta]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const gamma = workspaceOf("gamma", { role: 20 });
    const updated = track(store.updateWorkspace("acme", { timezone: "Asia/Shanghai" }));
    const created = track(store.createWorkspace({ name: "gamma", slug: "gamma" }));
    const deleted = track(store.deleteWorkspace("beta"));
    await inTurn(nerve, 1, ["PATCH", "/api/v0/workspaces/acme"], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["POST", LIST], json(201, gamma));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspaces/beta"], noContent());
    await until(() => deleted.settled, "the last change");
    expect(updated.error).toBeInstanceOf(ApiError);
    expect(created.value).toEqual(gamma);
    expect(deleted.error).toBeUndefined();
    expect(store.workspaces).toEqual([acme, gamma]);
  });
});
````

`web/apps/web/core/store/workspace/index.ts`（整个文件，203 行）：

````whole web/apps/web/core/store/workspace/index.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { clone } from "lodash-es";
import { action, computed, observable, makeObservable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
import type { IWorkspaceUserPropertiesResponse } from "@nerve/types";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { WorkspaceService } from "@/services/workspace.service";
import { WorkspacesService } from "@/services/workspace/workspaces.service";
// store
import type { RootStore } from "@/store/root.store";
// sub-stores
import type { IWebhookStore } from "./webhook.store";
import { WebhookStore } from "./webhook.store";

export interface IWorkspaceRootStore {
  /** The caller's workspaces, in nerve's order (by name, then id); undefined until fetched. */
  workspaces: Workspace[] | undefined;
  projectNavigationPreferencesMap: Record<string, IWorkspaceUserPropertiesResponse>;
  // computed
  currentWorkspace: Workspace | null;
  // computed actions
  getWorkspaceBySlug: (workspaceSlug: string) => Workspace | null;
  // fetch actions
  fetchWorkspaces: () => Promise<Workspace[] | undefined>;
  // crud actions
  createWorkspace: (data: WorkspaceCreate) => Promise<Workspace>;
  updateWorkspace: (workspaceSlug: string, data: WorkspaceUpdate) => Promise<Workspace>;
  deleteWorkspace: (workspaceSlug: string) => Promise<void>;
  getProjectNavigationPreferences: (workspaceSlug: string) => IWorkspaceUserPropertiesResponse | undefined;
  fetchProjectNavigationPreferences: (workspaceSlug: string) => Promise<void>;
  updateProjectNavigationPreferences: (
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ) => Promise<void>;
  // sub-stores
  webhook: IWebhookStore;
}

/**
 * The workspaces of the account of a session (M3 design 7.3): its service sends with the session's client, which
 * the RootStore of the session hands down. Changes go one at a time (v0 design 7.7); fetches do not queue.
 */
export class WorkspaceRootStore implements IWorkspaceRootStore {
  workspaces: Workspace[] | undefined = undefined;
  projectNavigationPreferencesMap: Record<string, IWorkspaceUserPropertiesResponse> = {};
  // services
  workspaceService;
  private readonly service: WorkspacesService;
  /** The changes of the workspaces, sent one at a time. */
  private readonly changes = oneAtATime();
  // root store
  router;
  // sub-stores
  webhook: IWebhookStore;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // observables
      workspaces: observable.ref,
      projectNavigationPreferencesMap: observable,
      // computed
      currentWorkspace: computed,
      // actions
      fetchWorkspaces: action,
      createWorkspace: action,
      updateWorkspace: action,
      deleteWorkspace: action,
      fetchProjectNavigationPreferences: action,
      updateProjectNavigationPreferences: action,
    });

    // services
    this.workspaceService = new WorkspaceService();
    this.service = new WorkspacesService(api);
    // root store
    this.router = _rootStore.router;
    // sub-stores
    this.webhook = new WebhookStore(_rootStore);
  }

  /** The workspace the address names, when the caller is a member of it. */
  get currentWorkspace() {
    const workspaceSlug = this.router.workspaceSlug;
    return workspaceSlug ? this.getWorkspaceBySlug(workspaceSlug) : null;
  }

  /** The workspace of the caller's that slug names, or null. */
  getWorkspaceBySlug = (workspaceSlug: string) =>
    this.workspaces?.find((workspace) => workspace.slug === workspaceSlug) ?? null;

  /**
   * @description fetches the caller's workspaces and gives them; a change of session while they load is no failure:
   * the new session's store fetches its own (store-context.tsx), and this one gives undefined
   * @returns {Promise<Workspace[] | undefined>}
   */
  fetchWorkspaces = async (): Promise<Workspace[] | undefined> => {
    try {
      const workspaces = await this.service.list();
      runInAction(() => {
        this.workspaces = workspaces;
      });
      return workspaces;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description creates a workspace, with the caller as its admin; once the list is fetched it has the new one
   * last, until the next fetch puts it in nerve's order. Fails, changing nothing, when nerve refuses.
   * @returns {Promise<Workspace>}
   */
  createWorkspace = (data: WorkspaceCreate): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.create(data);
      runInAction(() => {
        if (this.workspaces) this.workspaces = [...this.workspaces, workspace];
      });
      return workspace;
    });

  /**
   * @description changes a workspace's name, organization size or time zone; the list then has nerve's answer.
   * Fails, changing nothing, when nerve refuses.
   * @returns {Promise<Workspace>}
   */
  updateWorkspace = (workspaceSlug: string, data: WorkspaceUpdate): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.update(workspaceSlug, data);
      runInAction(() => {
        this.workspaces = this.workspaces?.map((w) => (w.id === workspace.id ? workspace : w));
      });
      return workspace;
    });

  /**
   * @description deletes a workspace, which then leaves the list; fails, changing nothing, when nerve refuses
   * @returns {Promise<void>}
   */
  deleteWorkspace = (workspaceSlug: string): Promise<void> =>
    this.changes(async () => {
      await this.service.delete(workspaceSlug);
      runInAction(() => {
        this.workspaces = this.workspaces?.filter((workspace) => workspace.slug !== workspaceSlug);
      });
    });

  getProjectNavigationPreferences = computedFn(
    (workspaceSlug: string): IWorkspaceUserPropertiesResponse | undefined =>
      this.projectNavigationPreferencesMap[workspaceSlug]
  );

  fetchProjectNavigationPreferences = async (workspaceSlug: string) => {
    try {
      const response = await this.workspaceService.fetchWorkspaceFilters(workspaceSlug);

      runInAction(() => {
        this.projectNavigationPreferencesMap[workspaceSlug] = response;
      });
    } catch (error) {
      console.error("Failed to fetch project navigation preferences:", error);
      throw error;
    }
  };

  updateProjectNavigationPreferences = async (
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ) => {
    const beforeUpdateData = clone(this.projectNavigationPreferencesMap[workspaceSlug]);

    try {
      // Optimistically update store
      runInAction(() => {
        this.projectNavigationPreferencesMap[workspaceSlug] = {
          ...this.projectNavigationPreferencesMap[workspaceSlug],
          ...data,
        };
      });

      // Call API to persist changes
      await this.workspaceService.patchWorkspaceFilters(workspaceSlug, data);
    } catch (error) {
      // Rollback on failure
      runInAction(() => {
        this.projectNavigationPreferencesMap[workspaceSlug] = beforeUpdateData;
      });
      console.error("Failed to update project navigation preferences:", error);
      throw error;
    }
  };
}
````

- [ ] **Step 4: 使用方改用 `Workspace` 和新的 store；工作区图标的上传删除**

`web/apps/web/app/(all)/create-workspace/page.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/create-workspace/page.tsx
import type { IWorkspace } from "@nerve/types";
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
import type { Workspace, WorkspaceCreate } from "@nerve/api-client";
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
  const [defaultValues, setDefaultValues] = useState<Pick<IWorkspace, "name" | "slug" | "organization_size">>({
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
  const [defaultValues, setDefaultValues] = useState<Pick<WorkspaceCreate, "name" | "slug" | "organization_size">>({
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
    slug: "",
    organization_size: "",
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
    slug: "",
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
  const onSubmit = async (workspace: IWorkspace) => {
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
  const onSubmit = async (workspace: Workspace) => {
````

`web/apps/web/core/components/core/modals/workspace-image-upload-modal.tsx`（删除）：

````delete web/apps/web/core/components/core/modals/workspace-image-upload-modal.tsx
````

`web/apps/web/core/components/onboarding/root.tsx`（修改，3 处）：

````old web/apps/web/core/components/onboarding/root.tsx
  const workspacesList = Object.values(workspaces ?? {});
````
````new web/apps/web/core/components/onboarding/root.tsx
  const hasWorkspaces = (workspaces?.length ?? 0) > 0;
````
````old web/apps/web/core/components/onboarding/root.tsx
          if (workspacesList.length > 0) finishOnboarding();
````
````new web/apps/web/core/components/onboarding/root.tsx
          if (hasWorkspaces) finishOnboarding();
````
````old web/apps/web/core/components/onboarding/root.tsx
    [stepChange, finishOnboarding, workspacesList]
````
````new web/apps/web/core/components/onboarding/root.tsx
    [stepChange, finishOnboarding, hasWorkspaces]
````

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（修改，4 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import type { User } from "@nerve/api-client";
import type { IWorkspace } from "@nerve/types";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import type { User, WorkspaceCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  } = useForm<IWorkspace>({
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  } = useForm<WorkspaceCreate>({
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      slug: "",
      organization_size: "",
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      slug: "",
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const handleCreateWorkspace = async (formData: IWorkspace) => {
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const handleCreateWorkspace = async (formData: WorkspaceCreate) => {
````

`web/apps/web/core/components/power-k/ui/pages/open-entity/workspaces-menu.tsx`（整个文件，25 行）：

````whole web/apps/web/core/components/power-k/ui/pages/open-entity/workspaces-menu.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve types
import type { Workspace } from "@nerve/api-client";
// components
import { PowerKWorkspacesMenu } from "@/components/power-k/menus/workspaces";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";

type Props = {
  handleSelect: (workspace: Workspace) => void;
};

export const PowerKOpenWorkspaceMenu = observer(function PowerKOpenWorkspaceMenu(props: Props) {
  const { handleSelect } = props;
  // store hooks
  const { workspaces } = useWorkspace();

  return <PowerKWorkspacesMenu workspaces={workspaces ?? []} onSelect={handleSelect} />;
});
````

`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`（修改，1 处）：

````old web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
        {Object.values(workspaces).map((workspace) => (
````
````new web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
        {(workspaces ?? []).map((workspace) => (
````

`web/apps/web/core/components/workspace/create-workspace-form.tsx`（修改，4 处）：

````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import type { IWorkspace } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import type { Workspace, WorkspaceCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  onSubmit?: (res: IWorkspace) => Promise<void>;
  defaultValues: {
    name: string;
    slug: string;
    organization_size: string;
  };
  setDefaultValues: Dispatch<SetStateAction<Pick<IWorkspace, "name" | "slug" | "organization_size">>>;
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  onSubmit?: (res: Workspace) => Promise<void>;
  defaultValues: Pick<WorkspaceCreate, "name" | "slug" | "organization_size">;
  setDefaultValues: Dispatch<SetStateAction<Pick<WorkspaceCreate, "name" | "slug" | "organization_size">>>;
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  } = useForm<IWorkspace>({ defaultValues, mode: "onChange" });
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  } = useForm<WorkspaceCreate>({ defaultValues, mode: "onChange" });
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  const handleCreateWorkspace = async (formData: IWorkspace) => {
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  const handleCreateWorkspace = async (formData: WorkspaceCreate) => {
````

`web/apps/web/core/components/workspace/delete-workspace-form.tsx`（修改，6 处）：

````old web/apps/web/core/components/workspace/delete-workspace-form.tsx
import type { IWorkspace } from "@nerve/types";

````
````new web/apps/web/core/components/workspace/delete-workspace-form.tsx
import type { Workspace } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/delete-workspace-form.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserSettings } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace/delete-workspace-form.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````old web/apps/web/core/components/workspace/delete-workspace-form.tsx
  data: IWorkspace | null;
````
````new web/apps/web/core/components/workspace/delete-workspace-form.tsx
  data: Workspace | null;
````
````old web/apps/web/core/components/workspace/delete-workspace-form.tsx
  const { t } = useTranslation();
  const { getWorkspaceRedirectionUrl } = useWorkspace();
  const { fetchCurrentUserSettings } = useUserSettings();
````
````new web/apps/web/core/components/workspace/delete-workspace-form.tsx
  const { t } = useTranslation();
````
````old web/apps/web/core/components/workspace/delete-workspace-form.tsx
      await deleteWorkspace(data.slug);
      await fetchCurrentUserSettings();
````
````new web/apps/web/core/components/workspace/delete-workspace-form.tsx
      await deleteWorkspace(data.slug);
````
````old web/apps/web/core/components/workspace/delete-workspace-form.tsx
      navigate(getWorkspaceRedirectionUrl());
````
````new web/apps/web/core/components/workspace/delete-workspace-form.tsx
      // the root lands the caller where his workspaces, as they are now, say (M3 design 3.14)
      navigate("/");
````

`web/apps/web/core/components/workspace/settings/members-list-item.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserPermissions, useUserSettings } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
  const { leaveWorkspace } = useUserPermissions();
  const { getWorkspaceRedirectionUrl } = useWorkspace();
  const { fetchCurrentUserSettings } = useUserSettings();
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
  const { leaveWorkspace } = useUserPermissions();
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
      await fetchCurrentUserSettings();
      navigate(getWorkspaceRedirectionUrl());
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
      // the root lands the caller where his workspaces, as they are now, say (M3 design 3.14)
      navigate("/");
````

`web/apps/web/core/components/workspace/settings/workspace-details.tsx`（修改，14 处）：

````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { Button } from "@nerve/propel/button";
import { EditOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { Button } from "@nerve/propel/button";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import type { IWorkspace } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import type { Workspace, WorkspaceUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
// components
import { WorkspaceImageUploadModal } from "@/components/core/modals/workspace-image-upload-modal";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
// components
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
const defaultValues: Partial<IWorkspace> = {
  name: "",
  url: "",
  organization_size: "2-10",
  logo_url: null,
  timezone: "UTC",
};
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
/** The form's values: what an admin may change of the workspace. */
type TWorkspaceForm = Required<Pick<WorkspaceUpdate, "name" | "timezone">> & Pick<WorkspaceUpdate, "organization_size">;

/** The form's values for a workspace; a size the form does not offer (none was given) shows as none. */
const formValues = (workspace: Workspace): TWorkspaceForm => ({
  name: workspace.name,
  organization_size: ORGANIZATION_SIZE.find((size) => size === workspace.organization_size),
  timezone: workspace.timezone,
});
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const [isLoading, setIsLoading] = useState(false);
  const [isImageUploadModalOpen, setIsImageUploadModalOpen] = useState(false);
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const [isLoading, setIsLoading] = useState(false);
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  } = useForm<IWorkspace>({
    defaultValues: { ...defaultValues, ...currentWorkspace },
  });
  // derived values
  const workspaceLogo = watch("logo_url");
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  } = useForm<TWorkspaceForm>({
    defaultValues: currentWorkspace
      ? formValues(currentWorkspace)
      : { name: "", organization_size: "2-10", timezone: "UTC" },
  });
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const onSubmit = async (formData: IWorkspace) => {
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const onSubmit = async (formData: TWorkspaceForm) => {
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
    const payload: Partial<IWorkspace> = {
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
    const payload: WorkspaceUpdate = {
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx

  const handleRemoveLogo = async () => {
    if (!currentWorkspace) return;

    try {
      await updateWorkspace(currentWorkspace.slug, {
        logo_url: "",
      });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Workspace picture removed successfully.",
      });
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "There was some error in deleting your profile picture. Please try again.",
      });
    }
  };

````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx

````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
    if (currentWorkspace) reset({ ...currentWorkspace });
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
    if (currentWorkspace) reset(formValues(currentWorkspace));
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
    <>
      <Controller
        control={control}
        name="logo_url"
        render={({ field: { onChange, value } }) => (
          <WorkspaceImageUploadModal
            isOpen={isImageUploadModalOpen}
            onClose={() => setIsImageUploadModalOpen(false)}
            handleRemove={handleRemoveLogo}
            onSuccess={(imageUrl) => {
              onChange(imageUrl);
              setIsImageUploadModalOpen(false);
            }}
            value={value}
          />
        )}
      />
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
    <>
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
            <button type="button" onClick={() => setIsImageUploadModalOpen(true)} disabled={!isAdmin}>
              {workspaceLogo && workspaceLogo !== "" ? (
                <div className="relative flex size-14">
                  <img
                    src={getFileURL(workspaceLogo)}
                    className="absolute top-0 left-0 size-full rounded-md object-cover"
                    alt="Workspace Logo"
                  />
                </div>
              ) : (
                <div className="relative grid size-14 place-items-center rounded-md bg-accent-primary text-24 text-on-color uppercase">
                  {currentWorkspace?.name?.charAt(0) ?? "N"}
                </div>
              )}
            </button>
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
            {currentWorkspace.logo_url ? (
              <div className="relative flex size-14">
                <img
                  src={getFileURL(currentWorkspace.logo_url)}
                  className="absolute top-0 left-0 size-full rounded-md object-cover"
                  alt="Workspace Logo"
                />
              </div>
            ) : (
              <div className="relative grid size-14 place-items-center rounded-md bg-accent-primary text-24 text-on-color uppercase">
                {currentWorkspace.name.charAt(0)}
              </div>
            )}
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
            }/${currentWorkspace.slug}`}</button>
            {isAdmin && (
              <button
                type="button"
                className="flex items-center gap-1.5 text-left text-caption-sm-medium text-accent-primary"
                onClick={() => setIsImageUploadModalOpen(true)}
              >
                {workspaceLogo && workspaceLogo !== "" ? (
                  <>
                    <EditOutline className="h-3 w-3" />
                    {t("workspace_settings.settings.general.edit_logo")}
                  </>
                ) : (
                  t("workspace_settings.settings.general.upload_logo")
                )}
              </button>
            )}
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
            }/${currentWorkspace.slug}`}</button>
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
              <Controller
                control={control}
                name="url"
                render={({ field: { onChange, ref } }) => (
                  <Field name="url" invalid={Boolean(errors.url)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="url"
                        name="url"
                        type="url"
                        value={`${
                          typeof window !== "undefined" &&
                          window.location.origin.replace("http://", "").replace("https://", "")
                        }/${currentWorkspace.slug}`}
                        onChange={onChange}
                        ref={ref}
                        disabled
                      />
                    </InputGroup>
                  </Field>
                )}
              />
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
              <Field name="url">
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="url"
                    name="url"
                    type="url"
                    value={`${
                      typeof window !== "undefined" &&
                      window.location.origin.replace("http://", "").replace("https://", "")
                    }/${currentWorkspace.slug}`}
                    readOnly
                    disabled
                  />
                </InputGroup>
              </Field>
````

`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
import type { IWorkspace } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
import type { Workspace } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
  const handleWorkspaceNavigation = (workspace: IWorkspace) => {
````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
  const handleWorkspaceNavigation = (workspace: Workspace) => {
````
````old web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
  const workspacesList = orderWorkspacesList(Object.values(workspaces ?? {}));
````
````new web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
  const workspacesList = workspaces && orderWorkspacesList(workspaces);
````

- [ ] **Step 5: 文案、上限、关键词规则**

`tools/keywords.json`（修改，1 处）：

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
      "id": "plane-workspace-urls",
      "phase": "M3/P8a",
      "why": "M3 换掉的 Plane 地址（M3 设计 7.10）：工作区一侧的 store 改调 /api/v0 之后，旧 service 中这些地址的方法都已删除。模式只写到 P8a 已换掉的地址为止；项目一侧的地址（项目、项目成员、项目角色、状态、标签、显示设置）P8b 换掉时补进来，在那之前它们不命中。/search-issues/ 不是 M3 的地址",
      "files": {
        "source": "^web/(?!.*\\.test\\.[jt]sx?$).*\\.(?:[cm]?[jt]sx?|json)$",
        "flags": ""
      },
      "content": {
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/",
        "flags": ""
      },
      "samples": {
        "hit": [
          "    return this.get(\"/api/users/me/workspaces/\")",
          "    return this.post(\"/api/workspaces/\", data)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/`)",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/`, data)",
          "    return this.delete(`/api/workspaces/${workspaceSlug}/`)",
          "    return this.get(\"/api/users/last-visited-workspace/\")"
        ],
        "miss": [
          "    return unwrap(await this.api.GET(\"/api/v0/workspaces\"));",
          "    return unwrap(await this.api.DELETE(\"/api/v0/workspaces/{slug}\", { params: { path: { slug } } }));",
          "    return this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`)",
          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/search-issues/`, {",
          "    return this.get(`/api/workspaces/${workspaceSlug}/states/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/views/`)"
        ],
        "files": {
          "hit": ["web/apps/web/core/services/workspace.service.ts"],
          "miss": ["docs/v0/M3-workspace-project/M3-design.md", "web/apps/web/core/store/workspace/index.test.ts"]
        }
      }
    }
  ],
````

`web/apps/web/core/services/file.service.ts`（修改，1 处）：

````old web/apps/web/core/services/file.service.ts
      })
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteWorkspaceAsset(workspaceSlug: string, assetId: string): Promise<void> {
    return this.delete(`/api/assets/v2/workspaces/${workspaceSlug}/${assetId}/`)
      .then((response) => response?.data)
````
````new web/apps/web/core/services/file.service.ts
      })
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 452",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 451",
````

`web/packages/constants/src/file.ts`（修改，1 处）：

````old web/packages/constants/src/file.ts

export const ACCEPTED_AVATAR_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE = {
  "image/jpeg": [],
  "image/jpg": [],
  "image/png": [],
  "image/webp": [],
};
````
````new web/packages/constants/src/file.ts

````

`web/packages/i18n/src/locales/en/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace-settings.json
        "title": "General",
        "upload_logo": "Upload logo",
        "edit_logo": "Edit logo",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
        "title": "General",
````

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "title": "常规",
        "upload_logo": "上传标志",
        "edit_logo": "编辑标志",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "title": "常规",
````

- [ ] **Step 6: 安装和运行检查**

Run: `pnpm install --frozen-lockfile`
Expected: 通过，锁文件不变（只有工作区内的两条链接）。

Run: `make lint-web`
Expected: 通过；关键词守卫 62 条规则，没有命中；web 的 oxlint 451 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过（W1、W10 建工作区之后进入它，W3 的 general 页改名）。

完成时：本 Task 删除的两个文案键（`workspace-settings.json` 的 `workspace_settings.settings.general.edit_logo`、`upload_logo`）在 `en`、`zh-CN` 中都已不在（spec 附录 A.7 的键表；`check:sync` 只核对两种语言一致，两种语言都留下的键要评审照键表核对）。

- [ ] **Step 7: 提交**

```bash
git add pnpm-lock.yaml tools/keywords.json 'web/apps/web/app/(all)/create-workspace/page.tsx' web/apps/web/core/components/core/modals/workspace-image-upload-modal.tsx web/apps/web/core/components/onboarding/root.tsx web/apps/web/core/components/onboarding/steps/workspace/create.tsx web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/workspaces.tsx web/apps/web/core/components/power-k/ui/pages/open-entity/workspaces-menu.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx web/apps/web/core/components/web-hooks/create-webhook-modal.tsx web/apps/web/core/components/web-hooks/utils.ts web/apps/web/core/components/workspace/create-workspace-form.tsx web/apps/web/core/components/workspace/delete-workspace-form.tsx web/apps/web/core/components/workspace/delete-workspace-modal.tsx web/apps/web/core/components/workspace/delete-workspace-section.tsx web/apps/web/core/components/workspace/settings/members-list-item.tsx web/apps/web/core/components/workspace/settings/workspace-details.tsx web/apps/web/core/components/workspace/sidebar/dropdown-item.tsx web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx web/apps/web/core/lib/auth/fake-nerve.ts web/apps/web/core/services/file.service.ts web/apps/web/core/services/workspace.service.ts web/apps/web/core/services/workspace/workspaces.service.ts web/apps/web/core/store/fake-queue.ts web/apps/web/core/store/fake-root.ts web/apps/web/core/store/root.store.ts web/apps/web/core/store/user/profile.store.ts web/apps/web/core/store/workspace/fake-workspaces.ts web/apps/web/core/store/workspace/index.test.ts web/apps/web/core/store/workspace/index.ts web/apps/web/package.json web/packages/constants/src/file.ts web/packages/constants/src/workspace.ts web/packages/i18n/src/locales/en/workspace-settings.json web/packages/i18n/src/locales/zh-CN/workspace-settings.json web/packages/types/package.json web/packages/types/src/project/projects.ts web/packages/types/src/search.ts web/packages/types/src/workspace.ts web/packages/utils/package.json web/packages/utils/src/workspace.ts
```
```bash
git commit -m "feat(M3/P8a): the workspaces come from /api/v0, by a store of the session

WorkspacesService wraps the generated client, and WorkspaceRootStore,
built per session with that session's client, keeps the caller's
workspaces as nerve lists them. Its changes go one at a time and take
nerve's answer; a change of session while the list loads is no
failure. The generated Workspace replaces IWorkspace in every consumer,
eleven of them by a mechanical rename. The workspace icon's upload goes:
the contract cannot set it. A keyword rule keeps the old workspace URLs
out.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t4-store-unqueued` | 工作区的修改不互相等待 | `index.test.ts` | vitest |
| `t4-store-fetch-swallows` | 列表取数失败时给出 `undefined`，不失败 | `index.test.ts` | vitest |
| `t4-store-session-fails` | 列表加载时换了会话，取数失败 | `index.test.ts` | vitest |
| `t4-store-delete-early` | 删除的工作区在 nerve 应答之前离开列表 | `index.test.ts` | vitest |
| `t4-store-update-stale` | 修改的应答之后列表仍是原来的工作区 | `index.test.ts` | vitest |
| `t4-store-create-unloaded` | 没有取过列表时，新建的工作区自成一个列表 | `index.test.ts` | vitest |
| `t4-store-fetch-queued` | 取数等待正在发出的修改 | `index.test.ts` | vitest |
| `t4-store-current-any` | 当前工作区是调用者的第一个工作区，不论地址 | `index.test.ts` | vitest |
| `t4-old-call-back` | 旧 service 又列出工作区 | 关键词守卫 | 静态 |
| `t4-create-unread` | service 不读 nerve 的 problem：被拒绝的创建当作成功 | `index.test.ts` | vitest |
| `t4-update-unread` | service 不读 nerve 的 problem：被拒绝的修改当作成功 | `index.test.ts` | vitest |

---

### Task 5: 落点、设置 store 删除、保留名单一份、离开工作区；设置页的侧边栏移到 `ThemeStore`

**Files:**
- Create: `web/apps/web/core/lib/fake-session-swr.ts`、`web/apps/web/core/lib/landing.test.ts`、`web/apps/web/core/lib/landing.ts`、`web/apps/web/core/lib/use-landing.test.ts`、`web/apps/web/core/lib/use-landing.ts`
- Modify: `e2e/stories/identity/a3-sign-in.spec.ts`、`server/internal/modules/workspace/domain/reserved_slugs.txt`、`tools/keywords.json`、`web/apps/web/app/routes/navigation.test.ts`、`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/settings/mobile/nav.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/core/components/workspace/settings/members-list-item.tsx`、`web/apps/web/core/hooks/store/user/index.ts`、`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`、`web/apps/web/core/services/user.service.ts`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/services/workspace/workspaces.service.ts`、`web/apps/web/core/store/theme.store.ts`、`web/apps/web/core/store/user/index.ts`、`web/apps/web/core/store/user/permissions.store.ts`、`web/apps/web/core/store/workspace/index.test.ts`、`web/apps/web/core/store/workspace/index.ts`、`web/packages/constants/src/navigation.test.ts`、`web/packages/constants/src/workspace.ts`、`web/packages/types/src/users.ts`
- Delete: `web/apps/web/core/hooks/store/user/user-user-settings.ts`、`web/apps/web/core/store/user/settings.store.ts`

**Interfaces:**
- Produces（spec 2.5；M3 设计 3.10、3.14、7.4、7.8；裁定 R4）：
  - `core/lib/landing.ts`：`landingPath(workspaces: readonly Workspace[], lastWorkspaceId: string | null): string`：上次的工作区仍在列表中 → `/<它的 slug>`；否则最早创建的（`created_at` 按时刻比较，同一时刻取列表中靠前的，即 nerve 的顺序）；一个都没有 → `/create-workspace`。
  - `core/lib/use-landing.ts`：`useLanding(pageType: EPageTypes, validNextPath: string | undefined): Landing`，落点的全部判断（裁定 A4）：从 store 读资料和工作区列表，资料已到、已完成引导（`isOnboarded`，从包装层移来并导出）、没有有效的 `next_path`、停在登录页或引导页时（`lands`）经 `useSessionSWR(lands ? ["WORKSPACES"] : null, () => fetchWorkspaces(), …)` 取列表，给出 `{ kind: "none" }`（不落）、`{ kind: "loading" }`（列表未到）、`{ kind: "unavailable", retry }`（取列表失败，`retry` 重取）或 `{ kind: "go", to: landingPath(workspaces, profile.last_workspace_id) }`。
  - `AuthenticationWrapper` 只照 `useLanding` 的结果渲染：`unavailable` 显示 `SessionUnavailable`（不自动重试，按钮调 `retry`），`loading` 显示加载，`go` 是 `<Navigate to={landing.to} replace />`；M2 为"完成引导的用户直接去 `/create-workspace`"写的两个分支改为这条规则。
  - `core/lib/fake-session-swr.ts`：测试取数的 hook 时用 `vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"))` 代替 `useSessionSWR`：`handed` 记下每次交给它的取数、fetcher 和配置，`response.current` 是每次调用交回的（默认 `{}`，即还没有数据）。本 Task 的 `use-landing.test.ts` 第一个用它，Task 6、8 的 hook 测试照用。
  - `WorkspacesService.leave(slug)`、`checkSlug(slug): Promise<SlugAvailability>`；`IWorkspaceRootStore.checkWorkspaceSlug(slug)`（不排队，是读）、`leaveWorkspace(slug)`（经 `changes`，成功之后把这个工作区移出列表）。成员页的"离开"改调 `useWorkspace().leaveWorkspace`；`UserPermissionStore.leaveWorkspace` 和 `UserService.leaveWorkspace` 删除（M2 交接第 11 节）。
  - 创建工作区的两处表单（`create-workspace-form.tsx`、新手引导的 `create.tsx`）改问 `checkWorkspaceSlug`，不再查 `RESTRICTED_URLS`；`create-workspace-form.tsx` 没有调用方传的 `secondaryButton`、`primaryButtonText` 删除（M3 设计 7.9，按钮照默认的文案）。
  - 删除（3.14、7.8）：`RESTRICTED_URLS` 和它的测试、`IUserSettings`、`UserSettingsStore`（`settings.store.ts`）、`useUserSettings`、`UserService.currentUserSettings`、四处 `await fetchCurrentUserSettings()`（`join-invites.tsx` 的一处随它在 Task 3 删除，删除工作区和离开工作区之后的两处在 Task 4 随 `getWorkspaceRedirectionUrl` 删去、改为跳到 `/` 由落点决定，这里是新手引导建工作区之后的一处）。
  - `ThemeStore.settingsSidebarCollapsed: boolean`（初值 `true`）和 `toggleSettingsSidebar(collapsed?)`：设置页在窄屏上的导航开合（原来在设置 store；R4 的理由见 spec 第 3 节），`settings/mobile/nav.tsx` 改读它。
  - 保留名单只有服务端的一份（`reserved_slugs.txt` 的说明改写）；`navigation.test.ts` 读它的"应用"一段，与路由的第一段静态段和 `public/` 的顶层目录两个方向相等（W15）。
- Adds：关键词规则 `restricted-urls`、`user-settings`；`plane-workspace-urls` 加 slug 检查和离开工作区的旧地址，不命中样例加项目的离开（P8b）和两个新地址。

**Tests:**
- vitest：`landing.test.ts`：`goes to the workspace opened last, while it is still one of the caller's`；`goes to the workspace created first when the last one is not listed, or there is none`；`goes to the page that creates a workspace when the caller has none`；`takes, of the workspaces created first at the same moment, the first in nerve's order`；`compares the moments, not their text`（时区不同的同一时刻）。
- vitest：`use-landing.test.ts`（`fake-session-swr.ts` 代替 `useSessionSWR`，资料和工作区列表的 store 换成桩）：`sends an onboarded account on the sign-in page where its workspaces decide, the one it opened last first`；`lands an onboarded account on the onboarding page the same way`；`it.each` 的 `leaves $who to the wrapper's other rules, and fetches nothing`（资料未到、仍在引导、有有效的 `next_path`、需要登录的页面、公开页面，5 行）；`waits for the caller's workspaces, and says when nerve cannot give them, with a retry`。每一项判断各有一个变异（`t5-auth-*`），都由它发现。
- vitest：`index.test.ts` 加 `asks nerve whether a slug is free, and fails when nerve cannot say`；`takes a deleted workspace off the list, and one the caller left`（原来的 `takes a deleted workspace off the list` 改成这个）；拒绝的 `it.each` 加离开一行。
- vitest：`navigation.test.ts` 的 `the reserved workspace addresses` / `reserve, in the app's section, the first static segment of every page and each directory of public/`；`constants/src/navigation.test.ts` 中 `RESTRICTED_URLS` 的测试随它删除。
- 端到端：A3 的说明改写，断言不变（没有工作区的账户落到 `/create-workspace`，落点取列表时不请求旧接口）。

- [ ] **Step 1: 落点函数、`useLanding` 和 `AuthenticationWrapper`**

`e2e/stories/identity/a3-sign-in.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a3-sign-in.spec.ts
  // A next_path that could lead elsewhere is dropped: the account's default page instead. That page,
  // /create-workspace, is one M2 reaches, so it asks no older API as it mounts (M2 design 3.1): the
  // workspace addresses answer 404 until M3.
````
````new e2e/stories/identity/a3-sign-in.spec.ts
  // A next_path that could lead elsewhere is dropped: the landing instead, which for an account with no
  // workspace is /create-workspace (M3 design 3.14). Neither the landing nor that page asks an older API as
  // it mounts (M2 design 3.1): the landing asks nerve for the account's workspaces.
````

`web/apps/web/core/lib/fake-session-swr.ts`（新文件，29 行）：

````file web/apps/web/core/lib/fake-session-swr.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for use-session-swr.ts, for the tests of the hooks that make a page's fetches (M3 design 7.1): a test
// file mocks that module with this one, vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr")), and
// reads what its hook handed over. The hooks then run as plain functions, outside React. How a fetch is keyed is
// use-session-swr.test.ts.

import type { SessionFetch } from "./use-session-swr";

/** What a hook handed useSessionSWR: the fetch, and the fetcher of its arguments. */
export type HandedFetch = [
  fetch: SessionFetch | null,
  fetcher: (...args: string[]) => Promise<unknown>,
  config?: unknown,
];

/** Every fetch handed over so far, in order; a test empties it before each case. */
export const handed: HandedFetch[] = [];

/** What each call gives back, as SWR would (by default, nothing yet); a test that sets it resets it before each case. */
export const response: { current: object } = { current: {} };

export function useSessionSWR(...call: HandedFetch): object {
  handed.push(call);
  return response.current;
}
````

`web/apps/web/core/lib/landing.test.ts`（新文件，48 行）：

````file web/apps/web/core/lib/landing.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { landingPath } from "./landing";

/** A workspace of the caller's, created at that moment. */
const workspace = (slug: string, created_at: string) => workspaceOf(slug, { created_at, updated_at: created_at });

// nerve's order: by name, then id
const acme = workspace("acme", "2026-10-02T09:00:00Z");
const beta = workspace("beta", "2026-10-01T09:00:00Z");
const gamma = workspace("gamma", "2026-10-03T09:00:00Z");

describe("landingPath", () => {
  it("goes to the workspace opened last, while it is still one of the caller's", () => {
    expect(landingPath([acme, beta, gamma], gamma.id)).toBe("/gamma");
  });

  it("goes to the workspace created first when the last one is not listed, or there is none", () => {
    // left, removed or deleted since: the hint is not checked to exist (3.14)
    expect(landingPath([acme, beta, gamma], "id-gone")).toBe("/beta");
    expect(landingPath([acme, beta, gamma], null)).toBe("/beta");
    expect(landingPath([gamma, acme], null)).toBe("/acme");
  });

  it("goes to the page that creates a workspace when the caller has none", () => {
    expect(landingPath([], null)).toBe("/create-workspace");
    expect(landingPath([], beta.id)).toBe("/create-workspace");
  });

  it("takes, of the workspaces created first at the same moment, the first in nerve's order", () => {
    const delta = workspace("delta", beta.created_at);
    expect(landingPath([beta, delta], null)).toBe("/beta");
    expect(landingPath([delta, beta], null)).toBe("/delta");
  });

  it("compares the moments, not their text", () => {
    // nerve leaves a fraction's trailing zeros out of a moment (Go's RFC 3339): 09:00:00Z is before 09:00:00.5Z,
    // though its text sorts after
    const whole = workspace("whole", "2026-10-01T09:00:00Z");
    const half = workspace("half", "2026-10-01T09:00:00.5Z");
    expect(landingPath([half, whole], null)).toBe("/whole");
  });
});
````

`web/apps/web/core/lib/landing.ts`（新文件，23 行）：

````file web/apps/web/core/lib/landing.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Workspace } from "@nerve/api-client";

/**
 * Where an onboarded account goes when no page is asked for (M3 design 3.14, 7.4): the workspace it opened last, while
 * that is still one of its workspaces; else the one created first, and of those created at the same moment, the
 * first in the list's order (nerve's: by name, then id); with none, the page that creates one. The last workspace is
 * only a hint the web app wrote (Profile.last_workspace_id): it counts only when the list has it.
 */
export function landingPath(workspaces: readonly Workspace[], lastWorkspaceId: string | null): string {
  const landing =
    workspaces.find((workspace) => workspace.id === lastWorkspaceId) ??
    workspaces.reduce<Workspace | undefined>(
      (first, workspace) =>
        first === undefined || Date.parse(workspace.created_at) < Date.parse(first.created_at) ? workspace : first,
      undefined
    );
  return landing ? `/${landing.slug}` : "/create-workspace";
}
````

`web/apps/web/core/lib/use-landing.test.ts`（新文件，120 行）：

````file web/apps/web/core/lib/use-landing.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OnboardingSteps, Profile, Workspace } from "@nerve/api-client";
import { EPageTypes } from "@/helpers/authentication.helper";
import { handed, response } from "@/lib/fake-session-swr";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The landing of a signed-in account (M3 design 3.14) as AuthenticationWrapper is told it, with fake-session-swr.ts's
// stand-in for useSessionSWR and stand-ins for the stores. Where the landing goes, of the caller's workspaces, is
// landing.test.ts.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const stores = vi.hoisted(
  (): { profile: Profile | undefined; workspaces: Workspace[] | undefined; calls: string[] } => ({
    profile: undefined,
    workspaces: undefined,
    calls: [],
  })
);
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: stores.profile }) }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    workspaces: stores.workspaces,
    fetchWorkspaces: () => Promise.resolve(stores.calls.push("the workspaces")),
  }),
}));

const { useLanding } = await import("./use-landing");

const done: OnboardingSteps = {
  profile_complete: true,
  workspace_create: true,
  workspace_invite: true,
  workspace_join: true,
};

/** The caller's profile as nerve gives it: onboarded, and the workspace he opened last is beta, unless fields say not. */
function profileOf(fields: Partial<Profile> = {}): Profile {
  return {
    theme: "system",
    language: "en",
    start_of_the_week: 0,
    onboarding_step: done,
    is_onboarded: true,
    is_tour_completed: false,
    last_workspace_id: "id-beta",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}
const acme = workspaceOf("acme", { created_at: "2026-09-01T09:00:00Z" });
const beta = workspaceOf("beta", { created_at: "2026-09-02T09:00:00Z" });
const { AUTHENTICATED, NON_AUTHENTICATED, ONBOARDING, PUBLIC } = EPageTypes;

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  stores.profile = profileOf();
  stores.workspaces = [acme, beta];
  stores.calls = [];
});

describe("useLanding", () => {
  it("sends an onboarded account on the sign-in page where its workspaces decide, the one it opened last first", async () => {
    expect(useLanding(NON_AUTHENTICATED, undefined)).toEqual({ kind: "go", to: "/beta" });
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher()));
    expect(stores.calls).toEqual(["the workspaces"]);

    stores.profile = profileOf({ last_workspace_id: null });
    expect(useLanding(NON_AUTHENTICATED, undefined)).toEqual({ kind: "go", to: "/acme" });
  });

  it("lands an onboarded account on the onboarding page the same way", () => {
    expect(useLanding(ONBOARDING, undefined)).toEqual({ kind: "go", to: "/beta" });
  });

  it.each([
    {
      who: "an account whose profile has not come",
      pageType: NON_AUTHENTICATED,
      nextPath: undefined,
      profile: undefined,
    },
    {
      who: "an account still onboarding",
      pageType: NON_AUTHENTICATED,
      nextPath: undefined,
      profile: profileOf({ is_onboarded: false, onboarding_step: { ...done, workspace_create: false } }),
    },
    {
      who: "an account with a valid next_path",
      pageType: NON_AUTHENTICATED,
      nextPath: "/acme/projects",
      profile: profileOf(),
    },
    { who: "an account on a page that needs one", pageType: AUTHENTICATED, nextPath: undefined, profile: profileOf() },
    { who: "an account on a public page", pageType: PUBLIC, nextPath: undefined, profile: profileOf() },
  ])("leaves $who to the wrapper's other rules, and fetches nothing", ({ pageType, nextPath, profile }) => {
    stores.profile = profile;
    expect(useLanding(pageType, nextPath)).toEqual({ kind: "none" });
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
  });

  it("waits for the caller's workspaces, and says when nerve cannot give them, with a retry", () => {
    stores.workspaces = undefined;
    expect(useLanding(NON_AUTHENTICATED, undefined)).toEqual({ kind: "loading" });

    const mutate = vi.fn(() => Promise.resolve(undefined));
    response.current = { error: new Error("nerve cannot be reached"), mutate };
    const landing = useLanding(NON_AUTHENTICATED, undefined);
    expect(landing.kind).toBe("unavailable");
    if (landing.kind === "unavailable") landing.retry();
    expect(mutate).toHaveBeenCalledOnce();
  });
});
````

`web/apps/web/core/lib/use-landing.ts`（新文件，55 行）：

````file web/apps/web/core/lib/use-landing.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Profile } from "@nerve/api-client";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserProfile } from "@/hooks/store/user";
// lib
import { landingPath } from "@/lib/landing";
import { useSessionSWR } from "@/lib/use-session-swr";

/** Whether the account is done with onboarding (M2 design 7.4): its profile says so, or every step is done. */
export const isOnboarded = (profile: Profile) =>
  profile.is_onboarded ||
  (profile.onboarding_step.profile_complete &&
    profile.onboarding_step.workspace_create &&
    profile.onboarding_step.workspace_invite &&
    profile.onboarding_step.workspace_join);

/**
 * What AuthenticationWrapper does about the landing: nothing, where the page does not land the account; wait for the
 * caller's workspaces; say nerve cannot be reached, with a retry; or go to the landing.
 */
export type Landing =
  | { kind: "none" }
  | { kind: "loading" }
  | { kind: "unavailable"; retry: () => void }
  | { kind: "go"; to: string };

/**
 * The landing of a signed-in account (M3 design 3.14), the whole decision: an onboarded account that the sign-in or
 * sign-up page, or the onboarding page, sends on without a valid next_path goes where its workspaces decide, the one
 * it opened last first (landingPath). The caller's workspaces are fetched, for the session, only then.
 */
export function useLanding(pageType: EPageTypes, validNextPath: string | undefined): Landing {
  const { data: profile } = useUserProfile();
  const { workspaces, fetchWorkspaces } = useWorkspace();
  const lands =
    profile !== undefined &&
    isOnboarded(profile) &&
    !validNextPath &&
    (pageType === EPageTypes.NON_AUTHENTICATED || pageType === EPageTypes.ONBOARDING);
  const listed = useSessionSWR(lands ? ["WORKSPACES"] : null, () => fetchWorkspaces(), {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });
  if (!lands) return { kind: "none" };
  if (listed.error) return { kind: "unavailable", retry: () => void listed.mutate() };
  if (!workspaces) return { kind: "loading" };
  return { kind: "go", to: landingPath(workspaces, profile.last_workspace_id) };
}
````

`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`（修改，10 处）：

````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
// nerve imports
import type { Profile } from "@nerve/api-client";
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
// nerve imports
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
import { useSession } from "@/lib/auth/use-session";
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
import { useSession } from "@/lib/auth/use-session";
import { isOnboarded, useLanding } from "@/lib/use-landing";
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
};

const isOnboarded = (profile: Profile) =>
  profile.is_onboarded ||
  (profile.onboarding_step.profile_complete &&
    profile.onboarding_step.workspace_create &&
    profile.onboarding_step.workspace_invite &&
    profile.onboarding_step.workspace_join);
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
};
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
 * to a valid next_path, else to /onboarding until onboarded, else to /create-workspace. While nerve cannot
 * be reached the session is kept, and the page says so instead of moving.
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
 * to a valid next_path, else to /onboarding until onboarded, else to the landing (M3 design 3.14), as from
 * the onboarding page once onboarded: useLanding decides it, and the wrapper renders what it says. While nerve
 * cannot be reached the session is kept, and the page says so instead of moving.
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  const { data: currentProfile } = useUserProfile();
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  const { data: currentProfile } = useUserProfile();
  // derived values
  const validNextPath = nextPath && isValidNextPath(nextPath) ? nextPath : undefined;
  const onboarded = currentProfile !== undefined && isOnboarded(currentProfile);
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  // renders again for the new session, with its new stores.
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  // renders again for the new session, with its new stores. The workspaces are fetched the same way, when the
  // landing needs them (useLanding).
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  });
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  });
  const landing = useLanding(pageType, validNextPath);
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  const validNextPath = nextPath && isValidNextPath(nextPath) ? nextPath : undefined;
  const onboarded = isOnboarded(currentProfile);
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
  if (landing.kind === "unavailable") return <SessionUnavailable autoRetry={false} onRetry={landing.retry} />;
  if (landing.kind === "loading") return <Loading />;
  if (landing.kind === "go") return <Navigate to={landing.to} replace />;
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
      return <Navigate to={validNextPath ?? (onboarded ? "/create-workspace" : "/onboarding")} replace />;
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
      return <Navigate to={validNextPath ?? "/onboarding"} replace />;
````
````old web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
      return onboarded ? <Navigate to={validNextPath ?? "/create-workspace"} replace /> : <>{children}</>;
````
````new web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
      return validNextPath && onboarded ? <Navigate to={validNextPath} replace /> : <>{children}</>;
````

- [ ] **Step 2: 离开工作区和 slug 检查改调新的 store**

`web/apps/web/core/components/workspace/settings/members-list-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser } from "@/hooks/store/user";
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
  const { leaveWorkspace } = useUserPermissions();
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
  const { leaveWorkspace } = useWorkspace();
````

`web/apps/web/core/services/workspace.service.ts`（修改，1 处）：

````old web/apps/web/core/services/workspace.service.ts
    return this.delete(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async workspaceSlugCheck(slug: string): Promise<any> {
    return this.get(`/api/workspace-slug-check/?slug=${slug}`)
````
````new web/apps/web/core/services/workspace.service.ts
    return this.delete(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`)
````

`web/apps/web/core/services/workspace/workspaces.service.ts`（修改，2 处）：

````old web/apps/web/core/services/workspace/workspaces.service.ts
import type { ApiClient, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
````
````new web/apps/web/core/services/workspace/workspaces.service.ts
import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/services/workspace/workspaces.service.ts
    unwrap(await this.api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } } }));
  }
````
````new web/apps/web/core/services/workspace/workspaces.service.ts
    unwrap(await this.api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } } }));
  }

  /** Ends the caller's own membership of the workspace. */
  async leave(slug: string): Promise<void> {
    unwrap(await this.api.POST("/api/v0/workspaces/{slug}/leave", { params: { path: { slug } } }));
  }

  /** Whether slug can name a new workspace, or why not. */
  async checkSlug(slug: string): Promise<SlugAvailability> {
    return unwrap(await this.api.GET("/api/v0/workspace-slugs/{slug}", { params: { path: { slug } } }));
  }
````

`web/apps/web/core/store/user/permissions.store.ts`（修改，3 处）：

````old web/apps/web/core/store/user/permissions.store.ts
  fetchUserWorkspaceInfo: (workspaceSlug: string) => Promise<IWorkspaceMemberMe>;
  leaveWorkspace: (workspaceSlug: string) => Promise<void>;
````
````new web/apps/web/core/store/user/permissions.store.ts
  fetchUserWorkspaceInfo: (workspaceSlug: string) => Promise<IWorkspaceMemberMe>;
````
````old web/apps/web/core/store/user/permissions.store.ts
      fetchUserWorkspaceInfo: action,
      leaveWorkspace: action,
````
````new web/apps/web/core/store/user/permissions.store.ts
      fetchUserWorkspaceInfo: action,
````
````old web/apps/web/core/store/user/permissions.store.ts
  /**
   * @description Leaves a workspace
   * @param { string } workspaceSlug
   * @returns { Promise<void | undefined> }
   */
  leaveWorkspace = async (workspaceSlug: string): Promise<void> => {
    try {
      await this.userService.leaveWorkspace(workspaceSlug);
      runInAction(() => {
        unset(this.workspaceUserInfo, workspaceSlug);
        unset(this.projectUserInfo, workspaceSlug);
        unset(this.workspaceProjectsPermissions, workspaceSlug);
      });
    } catch (error) {
      console.error("Error user leaving the workspace", error);
      throw error;
    }
  };

  /**
````
````new web/apps/web/core/store/user/permissions.store.ts
  /**
````

`web/apps/web/core/store/workspace/index.test.ts`（修改，7 处）：

````old web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toBeUndefined();
  });
````
````new web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toBeUndefined();
  });

  it("asks nerve whether a slug is free, and fails when nerve cannot say", async () => {
    const { nerve, store } = setUp();
    const checked = track(store.checkWorkspaceSlug("acme"));
    await until(() => nerve.calls.length === 1, "the check");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: "/api/v0/workspace-slugs/acme" });
    nerve.calls[0]?.answer(json(200, { available: false, reason: "taken" }));
    await until(() => checked.settled, "the answer");
    expect(checked.value).toEqual({ available: false, reason: "taken" });

    const failed = track(store.checkWorkspaceSlug("beta"));
    await until(() => nerve.calls.length === 2, "the second check");
    nerve.calls[1]?.answer(problem(500, "internal_error"));
    await until(() => failed.settled, "the failure");
    expect(failed.error).toBeInstanceOf(ApiError);
  });
````
````old web/apps/web/core/store/workspace/index.test.ts
  it("takes a deleted workspace off the list", async () => {
````
````new web/apps/web/core/store/workspace/index.test.ts
  it("takes a deleted workspace off the list, and one the caller left", async () => {
````
````old web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toEqual([beta]);
````
````new web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toEqual([beta]);

    const left = track(store.leaveWorkspace("beta"));
    await until(() => nerve.calls.length === 3, "the leave");
    expect(nerve.calls[2]).toMatchObject({ method: "POST", path: "/api/v0/workspaces/beta/leave" });
    nerve.calls[2]?.answer(noContent());
    await until(() => left.settled, "the leave");
    expect(left.error).toBeUndefined();
    expect(store.workspaces).toEqual([]);
````
````old web/apps/web/core/store/workspace/index.test.ts
    { change: "a deletion", send: (store) => store.deleteWorkspace("beta"), refusal: problem(403, "forbidden") },
````
````new web/apps/web/core/store/workspace/index.test.ts
    { change: "a deletion", send: (store) => store.deleteWorkspace("beta"), refusal: problem(403, "forbidden") },
    {
      change: "a leave",
      send: (store) => store.leaveWorkspace("acme"),
      refusal: problem(409, "workspace.sole_admin"),
    },
````
````old web/apps/web/core/store/workspace/index.test.ts
    const deleted = track(store.deleteWorkspace("beta"));
````
````new web/apps/web/core/store/workspace/index.test.ts
    const deleted = track(store.deleteWorkspace("beta"));
    const left = track(store.leaveWorkspace("acme"));
````
````old web/apps/web/core/store/workspace/index.test.ts
    await until(() => deleted.settled, "the last change");
````
````new web/apps/web/core/store/workspace/index.test.ts
    await inTurn(nerve, 4, ["POST", "/api/v0/workspaces/acme/leave"], noContent());
    await until(() => left.settled, "the last change");
````
````old web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toEqual([acme, gamma]);
````
````new web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toEqual([gamma]);
````

`web/apps/web/core/store/workspace/index.ts`（修改，6 处）：

````old web/apps/web/core/store/workspace/index.ts
import type { ApiClient, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
````
````new web/apps/web/core/store/workspace/index.ts
import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/store/workspace/index.ts
  fetchWorkspaces: () => Promise<Workspace[] | undefined>;
````
````new web/apps/web/core/store/workspace/index.ts
  fetchWorkspaces: () => Promise<Workspace[] | undefined>;
  checkWorkspaceSlug: (slug: string) => Promise<SlugAvailability>;
````
````old web/apps/web/core/store/workspace/index.ts
  deleteWorkspace: (workspaceSlug: string) => Promise<void>;
````
````new web/apps/web/core/store/workspace/index.ts
  deleteWorkspace: (workspaceSlug: string) => Promise<void>;
  leaveWorkspace: (workspaceSlug: string) => Promise<void>;
````
````old web/apps/web/core/store/workspace/index.ts
      deleteWorkspace: action,
````
````new web/apps/web/core/store/workspace/index.ts
      deleteWorkspace: action,
      leaveWorkspace: action,
````
````old web/apps/web/core/store/workspace/index.ts

  /**
   * @description creates a workspace, with the caller as its admin; once the list is fetched it has the new one
````
````new web/apps/web/core/store/workspace/index.ts

  /**
   * @description whether slug can name a new workspace, or why not (reserved, taken, invalid)
   * @returns {Promise<SlugAvailability>}
   */
  checkWorkspaceSlug = (slug: string): Promise<SlugAvailability> => this.service.checkSlug(slug);

  /**
   * @description creates a workspace, with the caller as its admin; once the list is fetched it has the new one
````
````old web/apps/web/core/store/workspace/index.ts
      runInAction(() => {
        this.workspaces = this.workspaces?.filter((workspace) => workspace.slug !== workspaceSlug);
      });
    });
````
````new web/apps/web/core/store/workspace/index.ts
      this.drop(workspaceSlug);
    });

  /**
   * @description ends the caller's membership of a workspace, which then leaves the list; fails, changing nothing,
   * when nerve refuses (the only admin of the workspace or of one of its projects, 409)
   * @returns {Promise<void>}
   */
  leaveWorkspace = (workspaceSlug: string): Promise<void> =>
    this.changes(async () => {
      await this.service.leave(workspaceSlug);
      this.drop(workspaceSlug);
    });

  /** The workspace leaves the caller's list: it was deleted, or he is no longer a member. */
  private drop(workspaceSlug: string): void {
    runInAction(() => {
      this.workspaces = this.workspaces?.filter((workspace) => workspace.slug !== workspaceSlug);
    });
  }
````

- [ ] **Step 3: 创建工作区的两处表单改问 nerve；`RESTRICTED_URLS` 删除，保留名单一份**

`server/internal/modules/workspace/domain/reserved_slugs.txt`（修改，2 处）：

````old server/internal/modules/workspace/domain/reserved_slugs.txt
# 工作区 slug 的保留名单（M3 设计 3.10）。前端的副本 RESTRICTED_URLS 随前端的数据层删除，
# 之后前后端只有这一份。
````
````new server/internal/modules/workspace/domain/reserved_slugs.txt
# 工作区 slug 的保留名单（M3 设计 3.10），前后端只有这一份：前端没有副本，创建表单问
# GET /api/v0/workspace-slugs/{slug}。
````
````old server/internal/modules/workspace/domain/reserved_slugs.txt
#   加上 web/apps/web/public/ 的顶层目录。核对它的 web vitest 随前端的数据层加入（M3 设计 9.5）。
````
````new server/internal/modules/workspace/domain/reserved_slugs.txt
#   加上 web/apps/web/public/ 的顶层目录，由 web 的 vitest 核对（web/apps/web/app/routes/navigation.test.ts）。
````

`web/apps/web/app/routes/navigation.test.ts`（修改，2 处）：

````old web/apps/web/app/routes/navigation.test.ts
const appDirectory = fileURLToPath(new URL("../..", import.meta.url));
````
````new web/apps/web/app/routes/navigation.test.ts
const appDirectory = fileURLToPath(new URL("../..", import.meta.url));
const reservedSlugs = join(appDirectory, "../../../server/internal/modules/workspace/domain/reserved_slugs.txt");
````
````old web/apps/web/app/routes/navigation.test.ts
  });
});

````
````new web/apps/web/app/routes/navigation.test.ts
  });
});

/** The names of a section of the reserved list: the lines under its "[name]", but blank lines and comments. */
const reservedSection = (name: string): string[] => {
  let section = "";
  return readFileSync(reservedSlugs, "utf8")
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => {
      if (/^\[\w+\]$/.test(line)) section = line.slice(1, -1);
      return line !== "" && !line.startsWith("#") && !line.startsWith("[") && section === name;
    });
};

// A workspace may not take an address the app serves itself (M3 design 3.10): the reserved list, the server's and the
// only one, has the app's top-level segments in its "app" section, so that a route added without its name, or a name
// left behind by a route that went, fails here.
describe("the reserved workspace addresses", () => {
  it("reserve, in the app's section, the first static segment of every page and each directory of public/", () => {
    const routed = pages.map((page) => page.split("/")[1]).filter((first) => first && !/^[:*]/.test(first));
    const published = readdirSync(join(appDirectory, "public"), { withFileTypes: true })
      .filter((entry) => entry.isDirectory())
      .map((entry) => entry.name);
    expect(new Set(reservedSection("app"))).toEqual(new Set([...routed, ...published]));
  });
});

````

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（修改，6 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { ORGANIZATION_SIZE, RESTRICTED_URLS } from "@nerve/constants";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { ORGANIZATION_SIZE } from "@nerve/constants";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { useUserProfile, useUserSettings } from "@/hooks/store/user";
// services
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { useUserProfile } from "@/hooks/store/user";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
};

const workspaceService = new WorkspaceService();
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
};
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const { fetchCurrentUserSettings } = useUserSettings();
  const { createWorkspace, fetchWorkspaces } = useWorkspace();
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const { createWorkspace, fetchWorkspaces, checkWorkspaceSlug } = useWorkspace();
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      const res = (await workspaceService.workspaceSlugCheck(formData.slug)) as { status: boolean };
      if (res.status === true && !RESTRICTED_URLS.includes(formData.slug)) {
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      const { available } = await checkWorkspaceSlug(formData.slug);
      if (available) {
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
    await updateUserProfile({ last_workspace_id: workspaceId }).catch(() => undefined);
    await fetchCurrentUserSettings();
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
    await updateUserProfile({ last_workspace_id: workspaceId }).catch(() => undefined);
````

`web/apps/web/core/components/workspace/create-workspace-form.tsx`（修改，9 处）：

````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import { ORGANIZATION_SIZE, RESTRICTED_URLS } from "@nerve/constants";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import { ORGANIZATION_SIZE } from "@nerve/constants";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import { useNavigate } from "react-router";
// services
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import { useNavigate } from "react-router";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  setDefaultValues: Dispatch<SetStateAction<Pick<WorkspaceCreate, "name" | "slug" | "organization_size">>>;
  secondaryButton?: React.ReactNode;
  primaryButtonText?: {
    loading: string;
    default: string;
  };
};

const workspaceService = new WorkspaceService();
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  setDefaultValues: Dispatch<SetStateAction<Pick<WorkspaceCreate, "name" | "slug" | "organization_size">>>;
};
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  const {
    onSubmit,
    defaultValues,
    setDefaultValues,
    secondaryButton,
    primaryButtonText = {
      loading: "workspace_creation.button.loading",
      default: "workspace_creation.button.default",
    },
  } = props;
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  const { onSubmit, defaultValues, setDefaultValues } = props;
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  const { createWorkspace } = useWorkspace();
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  const { createWorkspace, checkWorkspaceSlug } = useWorkspace();
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
      const res = (await workspaceService.workspaceSlugCheck(formData.slug)) as { status: boolean };
      if (res.status === true && !RESTRICTED_URLS.includes(formData.slug)) {
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
      const { available } = await checkWorkspaceSlug(formData.slug);
      if (available) {
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
      <div className="flex items-center gap-4">
        {secondaryButton}
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
      <div className="flex items-center gap-4">
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
          {isSubmitting ? t(primaryButtonText.loading) : t(primaryButtonText.default)}
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
          {isSubmitting ? t("workspace_creation.button.loading") : t("workspace_creation.button.default")}
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
        {!secondaryButton && (
          <Button variant="secondary" type="button" size="xl" onClick={() => navigate(-1)}>
            {t("common.go_back")}
          </Button>
        )}
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
        <Button variant="secondary" type="button" size="xl" onClick={() => navigate(-1)}>
          {t("common.go_back")}
        </Button>
````

`web/packages/constants/src/navigation.test.ts`（修改，2 处）：

````old web/packages/constants/src/navigation.test.ts
import {
  RESTRICTED_URLS,
  WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS,
  WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS,
} from "./workspace";
````
````new web/packages/constants/src/navigation.test.ts
import { WORKSPACE_SIDEBAR_PERSONAL_NAVIGATION_ITEMS, WORKSPACE_SIDEBAR_WORKSPACE_NAVIGATION_ITEMS } from "./workspace";
````
````old web/packages/constants/src/navigation.test.ts

// Workspace addresses a workspace cannot take; the list had config, mobile and monitor twice.
describe("the reserved workspace addresses", () => {
  it("list each address once", () => {
    expect(new Set(RESTRICTED_URLS).size).toBe(RESTRICTED_URLS.length);
  });
});

````
````new web/packages/constants/src/navigation.test.ts

````

`web/packages/constants/src/workspace.ts`（修改，1 处）：

````old web/packages/constants/src/workspace.ts
export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];

export const RESTRICTED_URLS: string[] = [
  "404",
  "accounts",
  "api",
  "create-workspace",
  "installations",
  "invitations",
  "onboarding",
  "profile",
  "workspace-invitations",
  "password",
  "flags",
  "monitor",
  "monitoring",
  "ingest",
  "disco",
  "chat",
  "calendar",
  "drive",
  "channels",
  "sign-in",
  "sign-up",
  "signin",
  "signup",
  "config",
  "admin",
  "m",
  "configuration",
  "initiatives",
  "initiative",
  "workflow",
  "workflows",
  "story",
  "mobile",
  "dashboard",
  "desktop",
  "onload",
  "real-time",
  "one",
  "business",
  "pro",
  "settings",
  "license",
  "licenses",
  "instances",
  "instance",
];
````
````new web/packages/constants/src/workspace.ts
export const ORGANIZATION_SIZE: OrganizationSize[] = ["Just myself", "2-10", "11-50", "51-200", "201-500", "500+"];
````

- [ ] **Step 4: 设置 store 删除；设置页的侧边栏移到 `ThemeStore`**

`web/apps/web/core/components/settings/mobile/nav.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/mobile/nav.tsx
import { useUserSettings } from "@/hooks/store/user";
````
````new web/apps/web/core/components/settings/mobile/nav.tsx
import { useAppTheme } from "@/hooks/store/use-app-theme";
````
````old web/apps/web/core/components/settings/mobile/nav.tsx
  const { sidebarCollapsed, toggleSidebar } = useUserSettings();
````
````new web/apps/web/core/components/settings/mobile/nav.tsx
  const { settingsSidebarCollapsed: sidebarCollapsed, toggleSettingsSidebar: toggleSidebar } = useAppTheme();
````

`web/apps/web/core/hooks/store/user/index.ts`（修改，1 处）：

````old web/apps/web/core/hooks/store/user/index.ts
export * from "./user-user-profile";
export * from "./user-user-settings";
````
````new web/apps/web/core/hooks/store/user/index.ts
export * from "./user-user-profile";
````

`web/apps/web/core/hooks/store/user/user-user-settings.ts`（删除）：

````delete web/apps/web/core/hooks/store/user/user-user-settings.ts
````

`web/apps/web/core/services/user.service.ts`（修改，3 处）：

````old web/apps/web/core/services/user.service.ts
import type { IUserSettings, TIssuesResponse } from "@nerve/types";
````
````new web/apps/web/core/services/user.service.ts
import type { TIssuesResponse } from "@nerve/types";
````
````old web/apps/web/core/services/user.service.ts

  async currentUserSettings(bustCache: boolean = false): Promise<IUserSettings> {
    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : "/api/users/me/settings/";
    return this.get(url)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

````
````new web/apps/web/core/services/user.service.ts

````
````old web/apps/web/core/services/user.service.ts
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveWorkspace(workspaceSlug: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)
````
````new web/apps/web/core/services/user.service.ts
    )
````

`web/apps/web/core/store/theme.store.ts`（修改，6 处）：

````old web/apps/web/core/store/theme.store.ts
  issueDetailSidebarCollapsed: boolean | undefined;
````
````new web/apps/web/core/store/theme.store.ts
  issueDetailSidebarCollapsed: boolean | undefined;
  /** Whether the settings' navigation is closed, on a narrow screen: closed until opened. */
  settingsSidebarCollapsed: boolean;
````
````old web/apps/web/core/store/theme.store.ts
  toggleIssueDetailSidebar: (collapsed?: boolean) => void;
````
````new web/apps/web/core/store/theme.store.ts
  toggleIssueDetailSidebar: (collapsed?: boolean) => void;
  toggleSettingsSidebar: (collapsed?: boolean) => void;
````
````old web/apps/web/core/store/theme.store.ts
  issueDetailSidebarCollapsed: boolean | undefined = undefined;
````
````new web/apps/web/core/store/theme.store.ts
  issueDetailSidebarCollapsed: boolean | undefined = undefined;
  settingsSidebarCollapsed: boolean = true;
````
````old web/apps/web/core/store/theme.store.ts
      issueDetailSidebarCollapsed: observable.ref,
````
````new web/apps/web/core/store/theme.store.ts
      issueDetailSidebarCollapsed: observable.ref,
      settingsSidebarCollapsed: observable.ref,
````
````old web/apps/web/core/store/theme.store.ts
      toggleIssueDetailSidebar: action,
````
````new web/apps/web/core/store/theme.store.ts
      toggleIssueDetailSidebar: action,
      toggleSettingsSidebar: action,
````
````old web/apps/web/core/store/theme.store.ts
    localStorage.setItem("issue_detail_sidebar_collapsed", this.issueDetailSidebarCollapsed.toString());
  };
````
````new web/apps/web/core/store/theme.store.ts
    localStorage.setItem("issue_detail_sidebar_collapsed", this.issueDetailSidebarCollapsed.toString());
  };

  /**
   * Open or close the settings' navigation on a narrow screen
   * @param collapsed
   */
  toggleSettingsSidebar = (collapsed?: boolean) => {
    this.settingsSidebarCollapsed = collapsed ?? !this.settingsSidebarCollapsed;
  };
````

`web/apps/web/core/store/user/index.ts`（修改，5 处）：

````old web/apps/web/core/store/user/index.ts
import { ApiTokenStore } from "./api-token.store";
import type { IUserSettingsStore } from "./settings.store";
import { UserSettingsStore } from "./settings.store";
````
````new web/apps/web/core/store/user/index.ts
import { ApiTokenStore } from "./api-token.store";
````
````old web/apps/web/core/store/user/index.ts
  data: User | undefined;
  // store observables
  userProfile: IUserProfileStore;
  userSettings: IUserSettingsStore;
  permission: IUserPermissionStore;
````
````new web/apps/web/core/store/user/index.ts
  data: User | undefined;
  // store observables
  userProfile: IUserProfileStore;
  permission: IUserPermissionStore;
````
````old web/apps/web/core/store/user/index.ts
  data: User | undefined = undefined;
  // store observables
  userProfile: IUserProfileStore;
  userSettings: IUserSettingsStore;
  permission: IUserPermissionStore;
````
````new web/apps/web/core/store/user/index.ts
  data: User | undefined = undefined;
  // store observables
  userProfile: IUserProfileStore;
  permission: IUserPermissionStore;
````
````old web/apps/web/core/store/user/index.ts
    this.userProfile = new ProfileStore(store, api);
    this.userSettings = new UserSettingsStore(api);
````
````new web/apps/web/core/store/user/index.ts
    this.userProfile = new ProfileStore(store, api);
````
````old web/apps/web/core/store/user/index.ts
      userProfile: observable,
      userSettings: observable,
````
````new web/apps/web/core/store/user/index.ts
      userProfile: observable,
````

`web/apps/web/core/store/user/settings.store.ts`（删除）：

````delete web/apps/web/core/store/user/settings.store.ts
````

`web/packages/types/src/users.ts`（修改，1 处）：

````old web/packages/types/src/users.ts

export interface IUserSettings {
  id: string | undefined;
  email: string | undefined;
  workspace: {
    last_workspace_id: string | undefined;
    last_workspace_slug: string | undefined;
    last_workspace_name: string | undefined;
    last_workspace_logo: string | undefined;
    fallback_workspace_id: string | undefined;
    fallback_workspace_slug: string | undefined;
    invites: number | undefined;
  };
}

````
````new web/packages/types/src/users.ts

````

- [ ] **Step 5: 关键词规则**

`tools/keywords.json`（修改，4 处）：

````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/leave/",
````
````old tools/keywords.json
          "    return this.get(\"/api/users/last-visited-workspace/\")"
````
````new tools/keywords.json
          "    return this.get(\"/api/users/last-visited-workspace/\")",
          "    return this.get(`/api/workspace-slug-check/?slug=${slug}`)",
          "    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)"
````
````old tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/views/`)"
````
````new tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/views/`)",
          "    return unwrap(await this.api.GET(\"/api/v0/workspace-slugs/{slug}\", { params: { path: { slug } } }));",
          "    unwrap(await this.api.POST(\"/api/v0/workspaces/{slug}/leave\", { params: { path: { slug } } }));",
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)"
````
````old tools/keywords.json
          "miss": ["docs/v0/M3-workspace-project/M3-design.md", "web/apps/web/core/store/workspace/index.test.ts"]
````
````new tools/keywords.json
          "miss": ["docs/v0/M3-workspace-project/M3-design.md", "web/apps/web/core/store/workspace/index.test.ts"]
        }
      }
    },
    {
      "id": "restricted-urls",
      "phase": "M3/P8a",
      "why": "前端的保留名单副本 RESTRICTED_URLS（M3 设计 3.10）：保留名单只有服务端的一份（server/internal/modules/workspace/domain/reserved_slugs.txt），创建表单问 GET /api/v0/workspace-slugs/{slug}；web 的 vitest 读那份名单核对应用一段",
      "files": {
        "source": "^web/(?!.*\\.test\\.[jt]sx?$).*\\.(?:[cm]?[jt]sx?|json)$",
        "flags": ""
      },
      "content": {
        "source": "\\bRESTRICTED_URLS\\b",
        "flags": ""
      },
      "samples": {
        "hit": [
          "export const RESTRICTED_URLS: string[] = [",
          "      if (available && !RESTRICTED_URLS.includes(formData.slug)) {"
        ],
        "miss": [
          "      const { available } = await checkWorkspaceSlug(formData.slug);",
          "  return readFileSync(reservedSlugs, \"utf8\")"
        ],
        "files": {
          "hit": [
            "web/packages/constants/src/workspace.ts",
            "web/apps/web/core/components/workspace/create-workspace-form.tsx"
          ],
          "miss": [
            "server/internal/modules/workspace/domain/reserved_slugs.txt",
            "web/apps/web/app/routes/navigation.test.ts"
          ]
        }
      }
    },
    {
      "id": "user-settings",
      "phase": "M3/P8a",
      "why": "Plane 的用户设置（M3 设计 3.14、7.8）：/api/users/me/settings/ 和它的类型、store、hook 都已删除；登录之后的落点由工作区列表和 Profile.last_workspace_id 算出（core/lib/landing.ts），设置页的移动端导航的开合在 ThemeStore",
      "files": {
        "source": "^web/(?!.*\\.test\\.[jt]sx?$).*\\.(?:[cm]?[jt]sx?|json)$",
        "flags": ""
      },
      "content": {
        "source": "\\b(?:IUserSettings|IUserSettingsStore|UserSettingsStore|currentUserSettings|fetchCurrentUserSettings|useUserSettings|getWorkspaceRedirectionUrl)\\b|/api/users/me/settings/",
        "flags": ""
      },
      "samples": {
        "hit": [
          "export interface IUserSettings {",
          "  const { fetchCurrentUserSettings } = useUserSettings();",
          "    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : \"/api/users/me/settings/\";",
          "      navigate(getWorkspaceRedirectionUrl());"
        ],
        "miss": [
          "  const { settingsSidebarCollapsed: sidebarCollapsed, toggleSettingsSidebar: toggleSidebar } = useAppTheme();",
          "    return <Navigate to={landingPath(workspaces, currentProfile.last_workspace_id)} replace />;",
          "  const { data: profile } = useUserProfile();"
        ],
        "files": {
          "hit": ["web/apps/web/core/store/user/settings.store.ts", "web/packages/types/src/users.ts"],
          "miss": ["docs/v0/M3-workspace-project/M3-design.md", "e2e/stories/identity/a3-sign-in.spec.ts"]
````

- [ ] **Step 6: 运行检查**

Run: `make lint-go`
Expected: 两段都输出 `0 issues.`。

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（保留名单的 Go 测试照旧）。

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则，没有命中；web 的 oxlint 451 条。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过（A3、A10 的落点；W1、W10 建工作区之后的跳转）。

- [ ] **Step 7: 提交**

```bash
git add e2e/stories/identity/a3-sign-in.spec.ts server/internal/modules/workspace/domain/reserved_slugs.txt tools/keywords.json web/apps/web/app/routes/navigation.test.ts web/apps/web/core/components/onboarding/steps/workspace/create.tsx web/apps/web/core/components/settings/mobile/nav.tsx web/apps/web/core/components/workspace/create-workspace-form.tsx web/apps/web/core/components/workspace/settings/members-list-item.tsx web/apps/web/core/hooks/store/user/index.ts web/apps/web/core/hooks/store/user/user-user-settings.ts web/apps/web/core/lib/fake-session-swr.ts web/apps/web/core/lib/landing.test.ts web/apps/web/core/lib/landing.ts web/apps/web/core/lib/use-landing.test.ts web/apps/web/core/lib/use-landing.ts web/apps/web/core/lib/wrappers/authentication-wrapper.tsx web/apps/web/core/services/user.service.ts web/apps/web/core/services/workspace.service.ts web/apps/web/core/services/workspace/workspaces.service.ts web/apps/web/core/store/theme.store.ts web/apps/web/core/store/user/index.ts web/apps/web/core/store/user/permissions.store.ts web/apps/web/core/store/user/settings.store.ts web/apps/web/core/store/workspace/index.test.ts web/apps/web/core/store/workspace/index.ts web/packages/constants/src/navigation.test.ts web/packages/constants/src/workspace.ts web/packages/types/src/users.ts
```
```bash
git commit -m "feat(M3/P8a): the landing comes from the caller's workspaces; one reserved list

An onboarded account with no page asked for lands on the workspace it
opened last while that is still one of its own, else on the one created
first, else on /create-workspace: useLanding makes the whole decision,
fetching the list by the session, landingPath computes the address, and
the authentication wrapper renders what useLanding says. The user settings
go, with the four awaits of them; the settings' mobile navigation state
moves to the theme store. The forms ask nerve whether a slug is free,
RESTRICTED_URLS goes, and a vitest keeps the server's reserved list
equal to the app's routes and public directories. Leaving a workspace
goes through the workspaces store.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t5-landing-tie` | 同一时刻建的两个工作区，落点取列表中靠后的一个 | `landing.test.ts` | vitest |
| `t5-landing-latest` | 落点取最后建的工作区 | `landing.test.ts` | vitest |
| `t5-landing-text` | 落点按时刻的文字比较 | `landing.test.ts` | vitest |
| `t5-landing-no-last` | 落点不看上次的工作区 | `landing.test.ts` | vitest |
| `t5-landing-first-listed` | 落点取列表的第一个，不是最早建的 | `landing.test.ts` | vitest |
| `t5-landing-none` | 没有工作区时落点是根路径 | `landing.test.ts`、故事 A3 | vitest；端到端 |
| `t5-auth-m2-landing` | 落点照 M2 把完成引导的账户一律送到 `/create-workspace` | `use-landing.test.ts` | vitest |
| `t5-auth-drops` | 落点的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |
| `t5-auth-no-hint` | 落点不看账户上次打开的工作区 | `use-landing.test.ts` | vitest |
| `t5-auth-onboarding-page` | 引导页上已完成引导的账户不再被送去落点 | `use-landing.test.ts` | vitest |
| `t5-auth-list-error` | nerve 给不出列表时落点一直停在加载 | `use-landing.test.ts` | vitest |
| `t5-auth-next-path` | 带合法 `next_path` 的账户也被送去落点 | `use-landing.test.ts` | vitest |
| `t5-auth-no-profile` | 资料还没到就送去落点 | `use-landing.test.ts` | vitest |
| `t5-auth-still-onboarding` | 仍在引导的账户也被送去落点 | `use-landing.test.ts` | vitest |
| `t5-auth-any-page` | 任何页面都送去落点，不只登录页和引导页 | `use-landing.test.ts` | vitest |
| `t5-store-leave-stays` | 离开的工作区留在列表里 | `index.test.ts` | vitest |
| `t5-store-leave-unqueued` | 离开不等待前一个修改 | `index.test.ts` | vitest |
| `t5-reserved-route-added` | 加一个顶层页面，保留名单没有它的名字 | `navigation.test.ts` | vitest |
| `t5-reserved-name-orphan` | 保留名单的"应用"一段有一个没有页面、没有文件的名字 | `navigation.test.ts` | vitest |
| `t5-reserved-public-unlisted` | 保留名单的"应用"一段漏了 `public/` 的一个目录 | `navigation.test.ts` | vitest |
| `t5-restricted-back` | 前端的保留名单副本加回 | 关键词守卫 | 静态 |
| `t5-settings-back` | 用户设置的取数回到 user service | 关键词守卫 | 静态 |
| `t5-slug-old` | slug 的检查又发往 Plane 的地址 | 关键词守卫 | 静态 |
| `t5-leave-old` | 离开又发往 Plane 的地址 | 关键词守卫 | 静态 |
| `t5-slug-ignored` | 创建表单不管 nerve 对 slug 的回答都创建 | oxlint（`check:lint`） | 静态 |
| `t5-slug-unread` | service 不读 nerve 的 problem：答不出的 slug 检查给出空，不失败 | `index.test.ts` | vitest |
| `t5-leave-unread` | service 不读 nerve 的 problem：被拒绝的离开当作成功 | `index.test.ts` | vitest |

---

### Task 6: 权限 store 的工作区一半；工作区包装层取列表

**Files:**
- Create: `web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/store/user/permissions.store.test.ts`
- Modify: `tools/keywords.json`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`、`web/apps/web/core/components/workspace/invite-modal/fields.tsx`、`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/issue/workspace-draft/issue.store.ts`、`web/apps/web/core/store/user/permissions.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/types/src/workspace.ts`

**Interfaces:**
- Produces（spec 2.6；M3 设计 7.1、7.2、7.3、8.3）：
  - `UserPermissionStore`：`getWorkspaceRoleByWorkspaceSlug(slug): WorkspaceRole | undefined` 取自 `workspaceRoot.getWorkspaceBySlug(slug)?.role`（列表没有取到、或不是调用者的工作区时为 `undefined`）；`allowPermissions` 的工作区一级和项目一级中"工作区管理员"的判断都读它。删除：`loader`、`workspaceUserInfo`、`workspaceInfoBySlug`、`fetchUserWorkspaceInfo`、`workspace-members/me` 的取数（旧 `WorkspaceService.workspaceMemberMe`、`IWorkspaceMemberMe`、取数键 `WORKSPACE_MEMBER_ME_INFORMATION`）。模块级的 `workspaceService` 改为 store 的字段（它剩下的 `project-roles` 一次调用是 P8b 的，spec 第 3 节）；`getProjectRole` 的规则不变。
  - 读调用者角色的使用方：`invite-modal/fields.tsx`、`invitations-list-item.tsx` 的 `workspaceInfoBySlug(slug)?.role` 改为 `getWorkspaceRoleByWorkspaceSlug(slug)`，原来的 `as` 随之消失。12 个页面（工作区设置的布局、10 个设置页、个人主页的页头）原来以 `workspaceUserInfo && …` 守着无权限的界面（页头是 `if (!workspaceUserInfo) return null`）：`workspaceUserInfo` 是一个对象，恒为真，这一层删去，行为不变。`workspace-draft/issue.store.ts` 往 `workspaceUserInfo` 写草稿数的一段（没有读者）删除。
  - `core/layouts/auth-layout/use-workspace-fetch.ts`：`useWorkspaceFetch(): SWRResponse<Workspace[] | undefined>`：`useSessionSWR(["WORKSPACES"], () => fetchWorkspaces(), { revalidateIfStale: false, revalidateOnFocus: false, shouldRetryOnError: false })`。Task 8、10 在这里加上成员和显示设置。
  - `WorkspaceAuthWrapper`：调用 `useWorkspaceFetch()`；`listed.error` 时显示 `SessionUnavailable`（按钮重取）；列表未到时加载中；路由的 slug 不在调用者的列表中时显示"找不到工作区"（不存在与不是成员同一个界面，8.3），Plane 的"Not Authorized"分支随 `workspaceInfoBySlug` 删除。项目一侧的取数（项目角色、项目列表、工作区的状态）不变，属于 P8b。
- Consumes：Task 5 的 `core/lib/fake-session-swr.ts`（`use-workspace-fetch.test.ts` 用它代替 `useSessionSWR`）。
- Adds：`plane-workspace-urls` 加 `workspace-members/me`；不命中样例加 `project-members/me`（P8b）。

**Tests:**（vitest）
- `permissions.store.test.ts`，与 9.2 同一组身份（管理员、成员、访客、不是成员的人）：`UserPermissionStore, in a workspace` / `gives $who the role nerve lists`；`allows $who what nerve allows, in the workspace named or in the address's`；`allows nothing before the list is fetched`（W9）。
- `use-workspace-fetch.test.ts`：`useWorkspaceFetch` / `fetches the caller's workspaces`（交给 `useSessionSWR` 的键是 `["WORKSPACES"]`，fetcher 交回 store 的 Promise）。

- [ ] **Step 1: 权限 store 的工作区角色取自 `Workspace.role`**

`web/apps/web/core/services/workspace.service.ts`（修改，2 处）：

````old web/apps/web/core/services/workspace.service.ts
import type {
  IWorkspaceMemberMe,
````
````new web/apps/web/core/services/workspace.service.ts
import type {
````
````old web/apps/web/core/services/workspace.service.ts
        throw error?.response?.data;
      });
  }

  async workspaceMemberMe(workspaceSlug: string): Promise<IWorkspaceMemberMe> {
    return this.get(`/api/workspaces/${workspaceSlug}/workspace-members/me/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
````
````new web/apps/web/core/services/workspace.service.ts
        throw error?.response?.data;
````

`web/apps/web/core/store/user/permissions.store.test.ts`（新文件，86 行）：

````file web/apps/web/core/store/user/permissions.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceRole } from "@nerve/api-client";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
import { RouterStore } from "@/store/router.store";
import { UserPermissionStore } from "@/store/user/permissions.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's role in a workspace is the one nerve lists with the workspace (Workspace.role, M3 design 7.2, 7.3),
// and the page's permissions follow it as nerve's do (9.2's workspace columns). On the web the caller who was never
// a member, the one removed and the one whose workspace was deleted are alike: nerve does not list the workspace.

const { ADMIN, MEMBER, GUEST } = EUserPermissions;
const WORKSPACE = EUserPermissionsLevel.WORKSPACE;

/** The checks the pages make: an admin's (settings, invitations), a member's (projects), anyone's (the workspace). */
type Check = "admin" | "member" | "anyone";
const CHECKS: Record<Check, EUserPermissions[]> = {
  admin: [ADMIN],
  member: [ADMIN, MEMBER],
  anyone: [ADMIN, MEMBER, GUEST],
};
const CHECKED: Check[] = ["admin", "member", "anyone"];

const IDENTITIES: { who: string; role: WorkspaceRole | undefined; allowed: Check[] }[] = [
  { who: "an admin", role: 20, allowed: ["admin", "member", "anyone"] },
  { who: "a member", role: 15, allowed: ["member", "anyone"] },
  { who: "a guest", role: 5, allowed: ["anyone"] },
  { who: "no member", role: undefined, allowed: [] },
];

/** The caller's workspaces as nerve lists them: one in which he has each role. */
async function setUp() {
  const nerve = new FakeNerve();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), nerve.client());
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }), nerve.client());
  const listed = IDENTITIES.flatMap(({ role }) => (role ? [workspaceOf(`ws-${role}`, { role })] : []));
  await loadWorkspaces(nerve, workspaceRoot, listed);
  return { router, permissions };
}

/** The workspace in which the caller has role: one the list does not have when he has none. */
const slugOf = (role: WorkspaceRole | undefined) => (role ? `ws-${role}` : "ws-elsewhere");

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("UserPermissionStore, in a workspace", () => {
  it.each(IDENTITIES)("gives $who the role nerve lists", async ({ role }) => {
    const { permissions } = await setUp();
    expect(permissions.getWorkspaceRoleByWorkspaceSlug(slugOf(role))).toBe(role);
  });

  it.each(IDENTITIES)("allows $who what nerve allows, in the workspace named or in the address's", async (identity) => {
    const { router, permissions } = await setUp();
    const slug = slugOf(identity.role);
    for (const check of CHECKED) {
      const roles = CHECKS[check];
      const allowed = identity.allowed.includes(check);
      expect(permissions.allowPermissions(roles, WORKSPACE, slug), `${check}, named`).toBe(allowed);
      router.setQuery({ workspaceSlug: slug });
      expect(permissions.allowPermissions(roles, WORKSPACE), `${check}, the address's`).toBe(allowed);
      router.setQuery({});
    }
  });

  it("allows nothing before the list is fetched", () => {
    const router = new RouterStore();
    const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), new FakeNerve().client());
    const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }), new FakeNerve().client());
    expect(permissions.getWorkspaceRoleByWorkspaceSlug("ws-20")).toBeUndefined();
    expect(permissions.allowPermissions(CHECKS.anyone, WORKSPACE, "ws-20")).toBe(false);
  });
});
````

`web/apps/web/core/store/user/permissions.store.ts`（修改，18 处）：

````old web/apps/web/core/store/user/permissions.store.ts
import type { ApiClient } from "@nerve/api-client";
````
````new web/apps/web/core/store/user/permissions.store.ts
import type { ApiClient, WorkspaceRole } from "@nerve/api-client";
````
````old web/apps/web/core/store/user/permissions.store.ts
import type { EUserProjectRoles, IUserProjectsRole, IWorkspaceMemberMe, TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/store/user/permissions.store.ts
import type { EUserProjectRoles, IUserProjectsRole, TProjectMembership } from "@nerve/types";
````
````old web/apps/web/core/store/user/permissions.store.ts

// derived services
const workspaceService = new WorkspaceService();

````
````new web/apps/web/core/store/user/permissions.store.ts

````
````old web/apps/web/core/store/user/permissions.store.ts
export interface IUserPermissionStore {
  loader: boolean;
  // observables
  workspaceUserInfo: Record<string, IWorkspaceMemberMe>; // workspaceSlug -> IWorkspaceMemberMe
````
````new web/apps/web/core/store/user/permissions.store.ts
export interface IUserPermissionStore {
  // observables
````
````old web/apps/web/core/store/user/permissions.store.ts
  workspaceInfoBySlug: (workspaceSlug: string | undefined) => IWorkspaceMemberMe | undefined;
  getWorkspaceRoleByWorkspaceSlug: (workspaceSlug: string) => TUserPermissions | EUserWorkspaceRoles | undefined;
````
````new web/apps/web/core/store/user/permissions.store.ts
  getWorkspaceRoleByWorkspaceSlug: (workspaceSlug: string) => WorkspaceRole | undefined;
````
````old web/apps/web/core/store/user/permissions.store.ts
  // actions
  fetchUserWorkspaceInfo: (workspaceSlug: string) => Promise<IWorkspaceMemberMe>;
````
````new web/apps/web/core/store/user/permissions.store.ts
  // actions
````
````old web/apps/web/core/store/user/permissions.store.ts
 * It manages workspace and project level permissions, roles and access control.
````
````new web/apps/web/core/store/user/permissions.store.ts
 * It manages workspace and project level permissions, roles and access control. The caller's role in a workspace
 * is the one nerve gives with the workspace (Workspace.role, M3 design 7.2).
````
````old web/apps/web/core/store/user/permissions.store.ts
export class UserPermissionStore implements IUserPermissionStore {
  loader: boolean = false;
  // constants
  workspaceUserInfo: Record<string, IWorkspaceMemberMe> = {};
````
````new web/apps/web/core/store/user/permissions.store.ts
export class UserPermissionStore implements IUserPermissionStore {
  // constants
````
````old web/apps/web/core/store/user/permissions.store.ts
  userService: UserService;
````
````new web/apps/web/core/store/user/permissions.store.ts
  userService: UserService;
  private readonly workspaceService = new WorkspaceService();
````
````old web/apps/web/core/store/user/permissions.store.ts
      // observables
      loader: observable.ref,
      workspaceUserInfo: observable,
````
````new web/apps/web/core/store/user/permissions.store.ts
      // observables
````
````old web/apps/web/core/store/user/permissions.store.ts
      // actions
      fetchUserWorkspaceInfo: action,
````
````new web/apps/web/core/store/user/permissions.store.ts
      // actions
````
````old web/apps/web/core/store/user/permissions.store.ts
   * @description Returns the current workspace information
   * @param { string | undefined } workspaceSlug
   * @returns { IWorkspaceMemberMe | undefined }
   */
  workspaceInfoBySlug = computedFn((workspaceSlug: string | undefined): IWorkspaceMemberMe | undefined => {
    if (!workspaceSlug) return undefined;
    return this.workspaceUserInfo[workspaceSlug] || undefined;
  });

  /**
   * @description Returns the workspace role by slug
````
````new web/apps/web/core/store/user/permissions.store.ts
   * @description Returns the caller's role in the workspace, from the caller's workspaces; undefined while they are
   * not fetched, or for a workspace the caller is not a member of
````
````old web/apps/web/core/store/user/permissions.store.ts
   * @returns { TUserPermissions | EUserWorkspaceRoles | undefined }
````
````new web/apps/web/core/store/user/permissions.store.ts
   * @returns { WorkspaceRole | undefined }
````
````old web/apps/web/core/store/user/permissions.store.ts
    (workspaceSlug: string): TUserPermissions | EUserWorkspaceRoles | undefined => {
      if (!workspaceSlug) return undefined;
      return this.workspaceUserInfo[workspaceSlug]?.role as TUserPermissions | EUserWorkspaceRoles | undefined;
    }
````
````new web/apps/web/core/store/user/permissions.store.ts
    (workspaceSlug: string): WorkspaceRole | undefined =>
      this.store.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.role
````
````old web/apps/web/core/store/user/permissions.store.ts
    const workspaceRole = this.workspaceUserInfo?.[workspaceSlug]?.role;
````
````new web/apps/web/core/store/user/permissions.store.ts
    const workspaceRole = this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug);
````
````old web/apps/web/core/store/user/permissions.store.ts
      currentUserRole = (workspaceSlug && this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug)) as
        | EUserPermissions
        | undefined;
````
````new web/apps/web/core/store/user/permissions.store.ts
      currentUserRole = workspaceSlug ? this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;
````
````old web/apps/web/core/store/user/permissions.store.ts
  // actions
  /**
   * @description Fetches the user's workspace information
   * @param { string } workspaceSlug
   * @returns { Promise<IWorkspaceMemberMe | undefined> }
   */
  fetchUserWorkspaceInfo = async (workspaceSlug: string): Promise<IWorkspaceMemberMe> => {
    try {
      this.loader = true;
      const response = await workspaceService.workspaceMemberMe(workspaceSlug);
      if (response) {
        runInAction(() => {
          set(this.workspaceUserInfo, [workspaceSlug], response);
          this.loader = false;
        });
      }
      return response;
    } catch (error) {
      console.error("Error fetching user workspace information", error);
      this.loader = false;
      throw error;
    }
  };

````
````new web/apps/web/core/store/user/permissions.store.ts
  // actions
````
````old web/apps/web/core/store/user/permissions.store.ts
      const response = await workspaceService.getWorkspaceUserProjectsRole(workspaceSlug);
````
````new web/apps/web/core/store/user/permissions.store.ts
      const response = await this.workspaceService.getWorkspaceUserProjectsRole(workspaceSlug);
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_MEMBER_ME_INFORMATION = (workspaceSlug: string) =>
  `WORKSPACE_MEMBER_ME_INFORMATION_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/types/src/workspace.ts`（修改，2 处）：

````old web/packages/types/src/workspace.ts
import type { IUserLite } from "./users";
import type { IWorkspaceViewProps } from "./view-props";
````
````new web/packages/types/src/workspace.ts
import type { IUserLite } from "./users";
````
````old web/packages/types/src/workspace.ts
  is_active?: boolean;
}

export interface IWorkspaceMemberMe {
  company_role: string | null;
  created_at: Date;
  created_by: string;
  default_props: IWorkspaceViewProps;
  id: string;
  member: string;
  role: TUserPermissions | EUserWorkspaceRoles;
  updated_at: Date;
  updated_by: string;
  view_props: IWorkspaceViewProps;
  workspace: string;
  draft_issue_count: number;
````
````new web/packages/types/src/workspace.ts
  is_active?: boolean;
````

- [ ] **Step 2: 包装层取列表**

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`（新文件，34 行）：

````file web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { handed } from "@/lib/fake-session-swr";

// A page of a workspace fetches what its caller may read (M3 design 3.1, 7.1), with fake-session-swr.ts's stand-in for
// useSessionSWR and a stand-in for the store.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const fetched = vi.hoisted((): { calls: string[] } => ({ calls: [] }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    fetchWorkspaces: () => Promise.resolve(fetched.calls.push("the workspaces")),
  }),
}));

const { useWorkspaceFetch } = await import("./use-workspace-fetch");

beforeEach(() => {
  handed.length = 0;
  fetched.calls = [];
});

describe("useWorkspaceFetch", () => {
  it("fetches the caller's workspaces", async () => {
    useWorkspaceFetch();
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher()));
    expect(fetched.calls).toEqual(["the workspaces"]);
  });
});
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`（新文件，24 行）：

````file web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SWRResponse } from "swr";
import type { Workspace } from "@nerve/api-client";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/** Fetched once a session: a page that mounts again shows what the stores have. */
const ONCE = { revalidateIfStale: false, revalidateOnFocus: false };

/**
 * The workspace side of what a page of a workspace fetches as it mounts (M3 design 3.1, 7.1): the caller's
 * workspaces, which decide whether he may see the address's one. Gives the list's response, whose failure the page
 * shows.
 */
export function useWorkspaceFetch(): SWRResponse<Workspace[] | undefined> {
  const { fetchWorkspaces } = useWorkspace();
  return useSessionSWR(["WORKSPACES"], () => fetchWorkspaces(), { ...ONCE, shouldRetryOnError: false });
}
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，15 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { Button, getButtonStyling } from "@nerve/propel/button";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { getButtonStyling } from "@nerve/propel/button";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
// components
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  WORKSPACE_PARTIAL_PROJECTS,
  WORKSPACE_MEMBER_ME_INFORMATION,
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  WORKSPACE_PARTIAL_PROJECTS,
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { usePlatformOS } from "@/hooks/use-platform-os";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { useWorkspaceFetch } from "./use-workspace-fetch";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { workspaces, fetchProjectNavigationPreferences } = useWorkspace();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { workspaces, getWorkspaceBySlug, fetchProjectNavigationPreferences } = useWorkspace();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { loader, workspaceInfoBySlug, fetchUserWorkspaceInfo, fetchUserProjectPermissions, allowPermissions } =
    useUserPermissions();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchUserProjectPermissions, allowPermissions } = useUserPermissions();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const allWorkspaces = workspaces ? Object.values(workspaces) : undefined;
  const currentWorkspace =
    (allWorkspaces && allWorkspaces.find((workspace) => workspace?.slug === workspaceSlug)) || undefined;
  const currentWorkspaceInfo = workspaceSlug && workspaceInfoBySlug(workspaceSlug);
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  // The caller's workspaces decide whether he may see this one, and his role in it (M3 design 7.2).
  const currentWorkspace = workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  // fetching user workspace information
  useSWR(
    workspaceSlug && currentWorkspace ? WORKSPACE_MEMBER_ME_INFORMATION(workspaceSlug) : null,
    workspaceSlug && currentWorkspace ? () => fetchUserWorkspaceInfo(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  // the workspace side of what every page of a workspace fetches (M3 design 7.1)
  const listed = useWorkspaceFetch();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  };

````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  };

  // nerve could not be reached: the page says so, and tries again when asked (M2 design 7.1)
  if (listed.error) return <SessionUnavailable autoRetry={false} onRetry={() => void listed.mutate()} />;

````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  if (allWorkspaces === undefined || loader) {
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  if (workspaces === undefined) {
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  // if workspaces are there and we are trying to access the workspace that we are not part of then show the existing workspaces
  if (currentWorkspace === undefined && !currentWorkspaceInfo) {
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  // a workspace that is not among the caller's: it does not exist, or he is not a member of it (M3 design 8.3)
  if (!currentWorkspace) {
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              {allWorkspaces && allWorkspaces.length > 0 && (
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              {workspaces.length > 0 && (
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              {allWorkspaces?.length > 0 && (
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              {workspaces.length > 0 && (
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              {allWorkspaces && allWorkspaces.length === 0 && (
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
              {workspaces.length === 0 && (
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx

  // while user does not have access to view that workspace
  if (currentWorkspaceInfo === undefined) {
    return (
      <div className="h-screen w-full overflow-hidden bg-surface-1">
        <div className="grid h-full place-items-center p-4">
          <div className="space-y-8 text-center">
            <div className="space-y-2">
              <h3 className="text-16 font-semibold">Not Authorized!</h3>
              <p className="mx-auto w-1/2 text-13 text-secondary">
                You're not a member of this workspace. Please contact the workspace admin to get an invitation.
              </p>
            </div>
            <div className="flex items-center justify-center gap-2">
              <Link to="/create-workspace">
                <span>
                  <Button variant="primary">Create new workspace</Button>
                </span>
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }

````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx

````

- [ ] **Step 3: 读角色的使用方**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  );

  if (!workspaceUserInfo) return null;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx
  );
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx
  const { workspaceUserInfo, getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx
        {workspaceUserInfo && !isAuthorized ? (
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx
        {!isAuthorized ? (
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  if (workspaceUserInfo && !canPerformWorkspaceMemberActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  if (!canPerformWorkspaceMemberActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx
  if (workspaceUserInfo && !canPerformWorkspaceAdminActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx
  if (!canPerformWorkspaceAdminActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  if (workspaceUserInfo && !canPerformProjectAdminActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  if (!canPerformProjectAdminActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx
  if (workspaceUserInfo && !canPerformProjectAdminActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx
  if (!canPerformProjectAdminActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
  if (workspaceUserInfo && !canPerformProjectAdminActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
  if (!canPerformProjectAdminActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx
  if (workspaceUserInfo && !canPerformProjectAdminActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx
  if (!canPerformProjectAdminActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx
  if (workspaceUserInfo && !canPerformProjectAdminActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx
  if (!canPerformProjectAdminActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  if (workspaceUserInfo && !canPerformProjectMemberActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  if (!canPerformProjectMemberActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx
  if (workspaceUserInfo && !canPerformProjectMemberActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx
  if (!canPerformProjectMemberActions) {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const { workspaceUserInfo, allowPermissions } = useUserPermissions();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  if (workspaceUserInfo && !canPerformProjectMemberActions) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  if (!canPerformProjectMemberActions) {
````

`web/apps/web/core/components/workspace/invite-modal/fields.tsx`（修改，8 处）：

````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
import { CustomSelect } from "@nerve/ui";
import { cn } from "@nerve/utils";
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
import { CustomSelect } from "@nerve/ui";
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
  remove: (index: number) => void;
  className?: string;
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
  remove: (index: number) => void;
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
    remove,
    className,
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
    remove,
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
  const { workspaceInfoBySlug } = useUserPermissions();
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
  const currentWorkspaceRole = workspaceInfoBySlug(workspaceSlug)?.role;
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
  const currentWorkspaceRole = getWorkspaceRoleByWorkspaceSlug(workspaceSlug);
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
    <div className={cn("mb-3 space-y-4", className)}>
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
    <div className="mb-3 space-y-4">
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                    {Object.entries(ROLE).map(([key, value]) => {
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                    {Object.entries(ROLE).map(([key, label]) => {
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                            {value}
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                            {label}
````

`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
  const { allowPermissions, workspaceInfoBySlug } = useUserPermissions();
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
  const { allowPermissions, getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
````
````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
  const currentWorkspaceMemberInfo = workspaceInfoBySlug(workspaceSlug);
  const currentWorkspaceRole = currentWorkspaceMemberInfo?.role;
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
  const currentWorkspaceRole = workspaceSlug ? getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;
````
````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
  if (!invitationDetails || !currentWorkspaceMemberInfo) return null;
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
  if (!invitationDetails || currentWorkspaceRole === undefined) return null;
````

`web/apps/web/core/store/issue/workspace-draft/issue.store.ts`（修改，6 处）：

````old web/apps/web/core/store/issue/workspace-draft/issue.store.ts

  private updateWorkspaceUserDraftIssueCount(workspaceSlug: string, increment: number) {
    const workspaceUserInfo = this.issueStore.rootStore.user.permission.workspaceUserInfo;
    const currentCount = workspaceUserInfo[workspaceSlug]?.draft_issue_count ?? 0;

    set(workspaceUserInfo, [workspaceSlug, "draft_issue_count"], currentCount + increment);
  }

````
````new web/apps/web/core/store/issue/workspace-draft/issue.store.ts

````
````old web/apps/web/core/store/issue/workspace-draft/issue.store.ts
          update(this.issueMapIds, [workspaceSlug], (existingIssueIds = []) => [...newIssueIds, ...existingIssueIds]);
````
````new web/apps/web/core/store/issue/workspace-draft/issue.store.ts
          update(this.issueMapIds, [workspaceSlug], (listedIssueIds = []) => [...newIssueIds, ...listedIssueIds]);
````
````old web/apps/web/core/store/issue/workspace-draft/issue.store.ts
          }
          // Update draft issue count in workspaceUserInfo
          this.updateWorkspaceUserDraftIssueCount(workspaceSlug, 1);
````
````new web/apps/web/core/store/issue/workspace-draft/issue.store.ts
          }
````
````old web/apps/web/core/store/issue/workspace-draft/issue.store.ts
          ...{ updated_at: getCurrentDateTimeInISO() },
````
````new web/apps/web/core/store/issue/workspace-draft/issue.store.ts
          updated_at: getCurrentDateTimeInISO(),
````
````old web/apps/web/core/store/issue/workspace-draft/issue.store.ts
        }
        // Update draft issue count in workspaceUserInfo
        this.updateWorkspaceUserDraftIssueCount(workspaceSlug, -1);
````
````new web/apps/web/core/store/issue/workspace-draft/issue.store.ts
        }
````
````old web/apps/web/core/store/issue/workspace-draft/issue.store.ts
        }

        // Update draft issue count in workspaceUserInfo
        this.updateWorkspaceUserDraftIssueCount(workspaceSlug, -1);
````
````new web/apps/web/core/store/issue/workspace-draft/issue.store.ts
        }
````

- [ ] **Step 4: 上限、关键词规则**

`tools/keywords.json`（修改，3 处）：

````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/leave/",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/leave/|/workspace-members/me/",
````
````old tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)"
````
````new tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/workspace-members/me/`)"
````
````old tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)"
````
````new tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/project-members/me/`)"
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 451",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 448",
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则，没有命中；web 的 oxlint 448 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过（工作区的页面现在能显示：W1、W3、W10 进入工作区）。

- [ ] **Step 6: 提交**

```bash
git add tools/keywords.json 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/header.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/cycles/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/modules/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/views/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/members/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx' web/apps/web/core/components/workspace/invite-modal/fields.tsx web/apps/web/core/components/workspace/settings/invitations-list-item.tsx web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/workspace.service.ts web/apps/web/core/store/issue/workspace-draft/issue.store.ts web/apps/web/core/store/user/permissions.store.test.ts web/apps/web/core/store/user/permissions.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/types/src/workspace.ts
```
```bash
git commit -m "feat(M3/P8a): the caller's role in a workspace is Workspace.role; the wrapper fetches the list

The permission store reads the caller's role in a workspace from the
workspaces nerve lists for him, and its workspace-members/me fetch
goes, with its loader and the module-level service. The workspace
wrapper fetches the list by the session through useWorkspaceFetch, says
so when nerve cannot be reached, and shows a workspace not among the
caller's as not found. The invite fields and the invitation rows read
the role from the store's getter; the settings pages drop a check on
workspaceUserInfo, an object, which never failed.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t6-perm-admin` | 管理员算作成员 | `permissions.store.test.ts` | vitest |
| `t6-perm-member` | 成员算作访客 | `permissions.store.test.ts` | vitest |
| `t6-perm-guest` | 访客算作不是成员 | `permissions.store.test.ts` | vitest |
| `t6-perm-none` | 不是成员算作访客 | `permissions.store.test.ts` | vitest |
| `t6-perm-no-address` | 不指明工作区的核对不取地址中的工作区 | `permissions.store.test.ts` | vitest |
| `t6-list-drops` | `useWorkspaceFetch` 取列表的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |
| `t6-list-old-key` | `useWorkspaceFetch` 的列表键不带会话（改用 SWR 本身，不经 `useSessionSWR`） | `use-workspace-fetch.test.ts` | vitest |
| `t6-member-me-back` | 旧 service 又取调用者自己的成员关系 | 关键词守卫 | 静态 |

---

### Task 7: `MemberUser` 取代 `IUserLite`；工作区成员的 service 和 store

**Files:**
- Create: `web/apps/web/core/services/workspace/workspace-members.service.ts`、`web/apps/web/core/store/member/workspace/fake-members.ts`、`web/apps/web/core/store/member/workspace/workspace-member.store.test.ts`
- Modify: `tools/keywords.json`、`web/apps/web/core/components/modules/links/list-item.tsx`、`web/apps/web/core/components/navigation/tab-navigation-root.tsx`、`web/apps/web/core/components/navigation/use-navigation-items.ts`、`web/apps/web/core/components/profile/sidebar.tsx`、`web/apps/web/core/components/profile/use-profile-member.ts`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx`、`web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/settings/members-list-item.tsx`、`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/member/index.ts`、`web/apps/web/core/store/member/project/project-member.store.ts`、`web/apps/web/core/store/member/utils.ts`、`web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts`、`web/apps/web/core/store/member/workspace/workspace-member.store.ts`、`web/apps/web/core/store/root.store.ts`、`web/apps/web/package.json`、`web/packages/types/src/users.ts`、`web/packages/types/src/workspace.ts`、`web/packages/typescript-config/react-router.json`、`web/packages/utils/src/file.ts`
- 机械步骤（Step 1）改到：`web/apps/web/core/components/dropdowns/member/base.tsx`、`web/apps/web/core/components/dropdowns/member/member-options.tsx`、`web/apps/web/core/components/project/confirm-project-member-remove.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/store/issue/root.store.ts`、`web/apps/web/core/store/member/project/project-member-filters.store.ts`、`web/apps/web/core/store/notifications/notification.ts`、`web/packages/types/src/project/projects.ts`、`web/packages/types/src/search.ts`、`web/packages/types/src/workspace-notifications.ts`、`web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts`

**Interfaces:**
- Produces（spec 2.7；M3 设计 5.2、7.2、7.3；M2 交接第 7、11 节）：
  - `MemberUser` 取代 `IUserLite`（`is_bot`、`joining_date` 随它删除）；`avatar_url`、`email` 可空，读作 `null`：图片的 `src` 在没有头像时不给，`alt` 用 `display_name || (email ?? undefined)`（W20）；`getFileURL` 收 `string | null`。成员加入的时刻取 `WorkspaceMember.created_at`（`sortWorkspaceMembers`、`sortProjectMembers` 的 `getJoinedAt`）。
  - `WorkspaceMember` 取代 `IWorkspaceMember`、`IWorkspaceMembership`（`root.store.ts` 的 issue 一侧经机械步骤）；页面的成员行（`member-columns.tsx`、`useMemberColumns.tsx`、`members-list-item.tsx`）直接用它，`RowData` 删除。
  - `core/services/workspace/workspace-members.service.ts`：`class WorkspaceMembersService { list(slug); update(membershipId, data: WorkspaceMemberUpdate); remove(membershipId) }`（`/api/v0/workspaces/{slug}/members`、`/api/v0/workspace-members/{id}`）。
  - `WorkspaceMemberStore(memberRoot: Pick<IMemberRootStore, "memberMap">, rootStore, api)`：`workspaceMemberMap[slug][userId]` 是这个工作区的全部成员关系（已结束的也在，`is_active` 为假）；每次取数整份换掉，成员的公开资料写进 `memberRoot.memberMap`。`fetchWorkspaceMembers(slug): Promise<WorkspaceMember[] | undefined>`（`SessionChangedError` 给出 `undefined`）；`updateMember(slug, userId, data)`、`removeMemberFromWorkspace(slug, userId)` 经 `changes = oneAtATime()`，store 没有这个人的成员关系时不问 nerve、直接失败；改角色写入 nerve 的回答，移出之后把这条成员关系标为已结束（与 nerve 再列出时相同）。`getWorkspaceMemberIds` 把调用者排在最前，其余按显示名。邀请的部分本 Task 不动（Task 8）。
  - `MemberRootStore(rootStore, api)`：没有读者的 `getMemberIds` 删除；`RootStore` 传这一代的客户端。
  - 旧 `WorkspaceService` 删去 `fetchWorkspaceMembers`、`updateWorkspaceMember`、`deleteWorkspaceMember`。
  - 测试的共用部分：`core/store/member/workspace/fake-members.ts` 的 `membershipOf(name, fields?)`（`acme` 的一条成员关系，名字给出账户和资料，默认是在职的成员）、`memberStore(client?)`（地址指向 `acme` 的一个标签页的成员 store 和共用的用户表；`client` 从假 nerve 建客户端，默认 `nerve.client()`，会话已换的测试传 `replacedSessionClient()`）。
  - web 应用的 TypeScript `lib` 改为 ES2023（`web/packages/typescript-config/react-router.json`，只有 `web/apps/web/tsconfig.json` 继承它；`target` 不变；裁定 A3）：`member/utils.ts` 用 `members.toSorted(…)` 排序，不改 store 交出的数组；导航的两处排序（`use-navigation-items.ts`、`tab-navigation-root.tsx`，排的都是 `filter` 新给的数组）同样改为 `toSorted`，它们的 `oxlint-disable-next-line unicorn/no-array-sort` 删除。本 Task 没有新的 oxlint 抑制。
- Adds：`plane-workspace-urls` 的模式扩到 `/api/workspaces/${…}/members/` 下的全部地址；不命中样例加项目成员的地址（P8b）和新的成员地址。

**Tests:**（vitest）`core/store/member/workspace/workspace-member.store.test.ts`：
- `WorkspaceMemberStore, the members`：`keeps a workspace's memberships as nerve lists them, ended ones too, and shares the members' profiles`；`lists the caller first, then the others by display name`；`fails when nerve cannot list them, keeping the members it had`；`gives nothing, and does not fail, when the session changes as it fetches`。
- `WorkspaceMemberStore, the changes`：`changes a member's role to nerve's answer, once nerve gives it`；`keeps a removed member's membership, ended, as nerve lists it`；`fails, changing nothing, when nerve refuses $change`；`fails, asking nerve nothing, for $change of a member it has not listed`；`sends each change once nerve has answered the one before it, refused or not`；`fetches the members while a change is out: a fetch does not wait for it`（Task 4 的 `fetchedWhileChangeIsOut`）。

- [ ] **Step 1: 机械步骤：`IUserLite` 换成 `MemberUser`，issue 一侧的 `IWorkspaceMembership` 换成 `WorkspaceMember`**

`$P8ATMP/rename_type.py` 见"一次性脚本"。18 个文件；其中 `use-profile-member.ts`、`member/index.ts`、`project-member.store.ts`、`member/utils.ts`、`workspace-member-filters.store.ts`、`types/src/workspace.ts` 在本 Task 另有手改，它们的块写在这一步之后的文件上。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IUserLite MemberUser web/apps/web/core/components/dropdowns/member/base.tsx web/apps/web/core/components/dropdowns/member/member-options.tsx web/apps/web/core/components/profile/use-profile-member.ts web/apps/web/core/components/project/confirm-project-member-remove.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/member/index.ts web/apps/web/core/store/member/project/project-member-filters.store.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/member/utils.ts web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts web/apps/web/core/store/notifications/notification.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts`
Expected: `IUserLite -> MemberUser in 14 files`。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IUserLite MemberUser --from ./users web/packages/types/src/workspace.ts web/packages/types/src/search.ts web/packages/types/src/workspace-notifications.ts`
Expected: `IUserLite -> MemberUser in 3 files`。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IUserLite MemberUser --from ../users web/packages/types/src/project/projects.ts`
Expected: `IUserLite -> MemberUser in 1 files`。

Run（机械步骤）: `python3 $P8ATMP/rename_type.py . IWorkspaceMembership WorkspaceMember --from @/store/member/workspace/workspace-member.store web/apps/web/core/store/issue/root.store.ts`
Expected: `IWorkspaceMembership -> WorkspaceMember in 1 files`。

Run（机械步骤）: `pnpm exec oxfmt web/apps/web/core/components/dropdowns/member/base.tsx web/apps/web/core/components/dropdowns/member/member-options.tsx web/apps/web/core/components/profile/use-profile-member.ts web/apps/web/core/components/project/confirm-project-member-remove.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/member/index.ts web/apps/web/core/store/member/project/project-member-filters.store.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/member/utils.ts web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts web/apps/web/core/store/notifications/notification.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts web/packages/types/src/workspace.ts web/packages/types/src/search.ts web/packages/types/src/workspace-notifications.ts web/packages/types/src/project/projects.ts`
Expected: 退出码 0。

`git diff --numstat` 必须恰好是：

```text
2	2	web/apps/web/core/components/dropdowns/member/base.tsx
2	2	web/apps/web/core/components/dropdowns/member/member-options.tsx
2	2	web/apps/web/core/components/profile/use-profile-member.ts
2	2	web/apps/web/core/components/project/confirm-project-member-remove.tsx
6	6	web/apps/web/core/components/project/project-settings-member-defaults.tsx
4	4	web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
6	7	web/apps/web/core/store/issue/root.store.ts
5	5	web/apps/web/core/store/member/index.ts
4	3	web/apps/web/core/store/member/project/project-member-filters.store.ts
2	2	web/apps/web/core/store/member/project/project-member.store.ts
6	5	web/apps/web/core/store/member/utils.ts
3	3	web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
3	2	web/apps/web/core/store/notifications/notification.ts
5	4	web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts
2	2	web/packages/types/src/workspace.ts
4	5	web/packages/types/src/search.ts
2	2	web/packages/types/src/workspace-notifications.ts
3	4	web/packages/types/src/project/projects.ts
```

Run（机械步骤）: `shasum -a 256 web/apps/web/core/components/dropdowns/member/base.tsx web/apps/web/core/components/dropdowns/member/member-options.tsx web/apps/web/core/components/profile/use-profile-member.ts web/apps/web/core/components/project/confirm-project-member-remove.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/member/index.ts web/apps/web/core/store/member/project/project-member-filters.store.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/member/utils.ts web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts web/apps/web/core/store/notifications/notification.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts web/packages/types/src/workspace.ts web/packages/types/src/search.ts web/packages/types/src/workspace-notifications.ts web/packages/types/src/project/projects.ts`
Expected: 每个文件的散列和行数与下表相同：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `c456fdc032e5c147ac2ce061bafa358c25365472bad95b0c7b53e8a3e44c3cac` | 187 | `web/apps/web/core/components/dropdowns/member/base.tsx` |
| `715c56843304a4569dc9a40257a6fa6d96ef1282b4314c7f73d88ef4bdcf7765` | 201 | `web/apps/web/core/components/dropdowns/member/member-options.tsx` |
| `1807fadf9600844eca27c9da85dbbf9b215f35c4c8d7b311e4e3a8cd13c4ef3b` | 45 | `web/apps/web/core/components/profile/use-profile-member.ts` |
| `1e1cb70183d8245800720cb24d501902f932888c4ecc77d94a3c82b0c2729bc4` | 94 | `web/apps/web/core/components/project/confirm-project-member-remove.tsx` |
| `711133e364c0cda412635064fb3c4358b0c43125c8e5147c497f38a5c1d436f6` | 204 | `web/apps/web/core/components/project/project-settings-member-defaults.tsx` |
| `e41f4b083f08fe5d096b1a28e4b032d3377809e666d5f63645738912246b1c3b` | 404 | `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx` |
| `463aa3895fa46a7d46de6edb58f901bcc0aa3be524f030e41f4c7c8e75b9ff8b` | 222 | `web/apps/web/core/store/issue/root.store.ts` |
| `9fa4e9a06d1513d8929b9ebd9bddc932473f025cafa4a95050735213f1ecb273` | 57 | `web/apps/web/core/store/member/index.ts` |
| `fa2e2317a0a1ecdd332a368319608810af7922d4edf429d74cf1a456a982f096` | 78 | `web/apps/web/core/store/member/project/project-member-filters.store.ts` |
| `9027fe0f388164c67327a00b6bb023cb8742af6dbc75348b53d49546d0018de4` | 497 | `web/apps/web/core/store/member/project/project-member.store.ts` |
| `7ebb2170e6cdf98a8c9d34c69ec1724d4207f3fb039bc62f39a8965e7bf4ff8c` | 191 | `web/apps/web/core/store/member/utils.ts` |
| `cf9d6b837527a75d0ccae457c979987d8d974f8b546c51ce69366d344526f666` | 78 | `web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts` |
| `62752be963eedc6fad6804ae19c6b9698fdf6e271c4ae2efae77659e207fb87d` | 326 | `web/apps/web/core/store/notifications/notification.ts` |
| `b2715e54038d611350ac97e75e08062d0550cf1d9deb9fdabe45c1a03b9242c7` | 93 | `web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts` |
| `a682048c167e301205afe16fbfbfde2cbb3e42fc1f08e93201fbae3e711f1a58` | 97 | `web/packages/types/src/workspace.ts` |
| `1b109fa20c7b45d711532ea82ed7dc83072f3bdfc8da7509459493715b593b03` | 72 | `web/packages/types/src/search.ts` |
| `a5a5af0b7a295ebedfe2dc09ff10177fe8f25c836be7b2e105a6a2f8c1f15f86` | 106 | `web/packages/types/src/workspace-notifications.ts` |
| `7e7eac28e83ed9199f4e0e6545bfa6106ccc810dbe18263674f2af18403ccb67` | 111 | `web/packages/types/src/project/projects.ts` |

- [ ] **Step 2: `lib` 改为 ES2023，导航的两处排序改为 `toSorted`**

`web/apps/web/core/components/navigation/tab-navigation-root.tsx`（修改，1 处）：

````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
    // oxlint-disable-next-line unicorn/no-array-sort
    .sort((a: TNavigationItem, b: TNavigationItem) => a.sortOrder - b.sortOrder);
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
    .toSorted((a: TNavigationItem, b: TNavigationItem) => a.sortOrder - b.sortOrder);
````

`web/apps/web/core/components/navigation/use-navigation-items.ts`（修改，1 处）：

````old web/apps/web/core/components/navigation/use-navigation-items.ts
    // oxlint-disable-next-line unicorn/no-array-sort
    return filteredItems.sort((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0));
````
````new web/apps/web/core/components/navigation/use-navigation-items.ts
    return filteredItems.toSorted((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0));
````

`web/packages/typescript-config/react-router.json`（修改，1 处）：

````old web/packages/typescript-config/react-router.json
    "lib": ["DOM", "DOM.Iterable", "ES2022"],
````
````new web/packages/typescript-config/react-router.json
    "lib": ["DOM", "DOM.Iterable", "ES2023"],
````

- [ ] **Step 3: 成员的 service 和 store**

`web/apps/web/core/services/workspace.service.ts`（修改，3 处）：

````old web/apps/web/core/services/workspace.service.ts
import type {
  IWorkspaceMember,
````
````new web/apps/web/core/services/workspace.service.ts
import type {
````
````old web/apps/web/core/services/workspace.service.ts

  async fetchWorkspaceMembers(workspaceSlug: string): Promise<IWorkspaceMember[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/members/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateWorkspaceMember(
    workspaceSlug: string,
    memberId: string,
    data: Partial<IWorkspaceMember>
  ): Promise<IWorkspaceMember> {
    return this.patch(`/api/workspaces/${workspaceSlug}/members/${memberId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteWorkspaceMember(workspaceSlug: string, memberId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/members/${memberId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts

````
````old web/apps/web/core/services/workspace.service.ts
    invitationId: string,
    data: Partial<IWorkspaceMember>
````
````new web/apps/web/core/services/workspace.service.ts
    invitationId: string,
    data: Partial<IWorkspaceMemberInvitation>
````

`web/apps/web/core/services/workspace/workspace-members.service.ts`（新文件，37 行）：

````file web/apps/web/core/services/workspace/workspace-members.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, WorkspaceMember, WorkspaceMemberUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A workspace's memberships (M3 design 5.1, 7.3). */
export class WorkspaceMembersService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** Every membership of the workspace, those that ended too (is_active false), in the order they began. */
  async list(slug: string): Promise<WorkspaceMember[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/members", { params: { path: { slug } } })).data;
  }

  /** Changes a member's role; the answer is the membership with its new role. */
  async update(membershipId: string, data: WorkspaceMemberUpdate): Promise<WorkspaceMember> {
    return unwrap(
      await this.api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
        params: { path: { workspace_member_id: membershipId } },
        body: data,
      })
    );
  }

  /** Ends a member's membership, and his memberships of the workspace's projects with it. */
  async remove(membershipId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
        params: { path: { workspace_member_id: membershipId } },
      })
    );
  }
}
````

`web/apps/web/core/store/member/index.ts`（修改，5 处）：

````old web/apps/web/core/store/member/index.ts
import type { MemberUser } from "@nerve/api-client";
````
````new web/apps/web/core/store/member/index.ts
import type { ApiClient, MemberUser } from "@nerve/api-client";
````
````old web/apps/web/core/store/member/index.ts
  // computed actions
  getMemberIds: () => string[];
````
````new web/apps/web/core/store/member/index.ts
  // computed actions
````
````old web/apps/web/core/store/member/index.ts
  constructor(_rootStore: RootStore) {
````
````new web/apps/web/core/store/member/index.ts
  constructor(_rootStore: RootStore, api: ApiClient) {
````
````old web/apps/web/core/store/member/index.ts
    this.workspace = new WorkspaceMemberStore(this, _rootStore);
````
````new web/apps/web/core/store/member/index.ts
    this.workspace = new WorkspaceMemberStore(this, _rootStore, api);
````
````old web/apps/web/core/store/member/index.ts
  }

  /**
   * @description get all member ids
   */
  getMemberIds = computedFn(() => Object.keys(this.memberMap));
````
````new web/apps/web/core/store/member/index.ts
  }
````

`web/apps/web/core/store/member/project/project-member.store.ts`（修改，2 处）：

````old web/apps/web/core/store/member/project/project-member.store.ts
    if (!projectMember || !userDetails) return null;
    const memberDetails: IProjectMemberDetails = {
      id: projectMember.id,
      role: projectMember.role,
      original_role: projectMember.original_role,
      member: {
        ...userDetails,
        joining_date: projectMember.created_at ?? undefined,
      },
      created_at: projectMember.created_at,
    };
    return memberDetails;
  });
````
````new web/apps/web/core/store/member/project/project-member.store.ts
    if (!projectMember || !userDetails) return null;
    const memberDetails: IProjectMemberDetails = {
      id: projectMember.id,
      role: projectMember.role,
      original_role: projectMember.original_role,
      member: userDetails,
      created_at: projectMember.created_at,
    };
    return memberDetails;
  });
````
````old web/apps/web/core/store/member/project/project-member.store.ts
    if (!filteredMemberIds.includes(userId)) return null;

    const memberDetails: IProjectMemberDetails = {
      id: projectMember.id,
      role: projectMember.role,
      original_role: projectMember.original_role,
      member: {
        ...userDetails,
        joining_date: projectMember.created_at ?? undefined,
      },
      created_at: projectMember.created_at,
    };
    return memberDetails;
  });

````
````new web/apps/web/core/store/member/project/project-member.store.ts
    if (!filteredMemberIds.includes(userId)) return null;

    const memberDetails: IProjectMemberDetails = {
      id: projectMember.id,
      role: projectMember.role,
      original_role: projectMember.original_role,
      member: userDetails,
      created_at: projectMember.created_at,
    };
    return memberDetails;
  });

````

`web/apps/web/core/store/member/utils.ts`（修改，9 处）：

````old web/apps/web/core/store/member/utils.ts
// Unified function to get sort key for any member type
const getMemberSortKey = (memberDetails: MemberUser, field: string, memberRole?: string): string | Date => {
````
````new web/apps/web/core/store/member/utils.ts
// Unified function to get sort key for any member type; a member joined when his membership began (joinedAt)
const getMemberSortKey = (
  memberDetails: MemberUser,
  field: string,
  memberRole?: string,
  joinedAt?: string | null
): string | Date => {
````
````old web/apps/web/core/store/member/utils.ts
      if (!memberDetails.joining_date) {
````
````new web/apps/web/core/store/member/utils.ts
      if (!joinedAt) {
````
````old web/apps/web/core/store/member/utils.ts
      const date = new Date(memberDetails.joining_date);
````
````new web/apps/web/core/store/member/utils.ts
      const date = new Date(joinedAt);
````
````old web/apps/web/core/store/member/utils.ts
  getMemberRole: (member: T) => string,
````
````new web/apps/web/core/store/member/utils.ts
  getMemberRole: (member: T) => string,
  getJoinedAt: (member: T) => string | null,
````
````old web/apps/web/core/store/member/utils.ts
  return [...members].sort((a, b) => {
````
````new web/apps/web/core/store/member/utils.ts
  return members.toSorted((a, b) => {
````
````old web/apps/web/core/store/member/utils.ts
    const aValue = getMemberSortKey(aMemberDetails, field, aRole);
    const bValue = getMemberSortKey(bMemberDetails, field, bRole);
````
````new web/apps/web/core/store/member/utils.ts
    const aValue = getMemberSortKey(aMemberDetails, field, aRole, getJoinedAt(a));
    const bValue = getMemberSortKey(bMemberDetails, field, bRole, getJoinedAt(b));
````
````old web/apps/web/core/store/member/utils.ts
    (member) => String(member.role ?? member.original_role ?? ""),
````
````new web/apps/web/core/store/member/utils.ts
    (member) => String(member.role ?? member.original_role ?? ""),
    (member) => member.created_at,
````
````old web/apps/web/core/store/member/utils.ts
export const sortWorkspaceMembers = <T extends { role: string | EUserPermissions; is_active?: boolean }>(
````
````new web/apps/web/core/store/member/utils.ts
export const sortWorkspaceMembers = <T extends { role: string | EUserPermissions; created_at: string }>(
````
````old web/apps/web/core/store/member/utils.ts
    (member) => String(member.role ?? ""),
````
````new web/apps/web/core/store/member/utils.ts
    (member) => String(member.role ?? ""),
    (member) => member.created_at,
````

`web/apps/web/core/store/member/workspace/fake-members.ts`（新文件，47 行）：

````file web/apps/web/core/store/member/workspace/fake-members.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The workspace members' store, for the tests of its members and of its invitations, against a fake nerve. A test
// file that uses it mocks @/lib/auth/api-client: the account's store, which it builds, imports the tab's session.

import type { ApiClient, MemberUser, WorkspaceMember } from "@nerve/api-client";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
import { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";
import { RouterStore } from "@/store/router.store";
import { UserStore } from "@/store/user";

/** A membership of acme as nerve lists it: the name names the member; an active member's, unless fields say not. */
export function membershipOf(name: string, fields: Partial<WorkspaceMember> = {}): WorkspaceMember {
  const member: MemberUser = {
    id: `u-${name}`,
    display_name: name,
    first_name: name,
    last_name: "Doe",
    avatar_url: null,
    email: `${name}@example.com`,
  };
  return {
    id: `m-${name}`,
    workspace_id: "id-acme",
    role: 15,
    is_active: true,
    created_at: "2026-10-01T09:00:00Z",
    member,
    ...fields,
  };
}

/** The store, and the users the stores share, of a tab whose address names acme; client builds its client. */
export function memberStore(client: (nerve: FakeNerve) => ApiClient = (nerve) => nerve.client()) {
  const nerve = new FakeNerve();
  const api = client(nerve);
  const router = new RouterStore();
  router.setQuery({ workspaceSlug: "acme" });
  const user = new UserStore(fakeRoot({ router }), api);
  const users: Record<string, MemberUser> = {};
  const store = new WorkspaceMemberStore({ memberMap: users }, fakeRoot({ router, user }), api);
  return { nerve, user, users, store };
}
````

`web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts`（修改，5 处）：

````old web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
import type { EUserPermissions } from "@nerve/constants";
import type { MemberUser } from "@nerve/api-client";
````
````new web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
import type { MemberUser, WorkspaceMember } from "@nerve/api-client";
````
````old web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
import { sortWorkspaceMembers } from "../utils";

// Workspace membership interface matching the store structure
interface IWorkspaceMembership {
  id: string;
  member: string;
  role: EUserPermissions;
  is_active?: boolean;
}
````
````new web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
import { sortWorkspaceMembers } from "../utils";
````
````old web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
  getFilteredMemberIds: (
    members: IWorkspaceMembership[],
    memberDetailsMap: Record<string, MemberUser>,
    getMemberKey: (member: IWorkspaceMembership) => string
````
````new web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
  getFilteredMemberIds: (
    members: WorkspaceMember[],
    memberDetailsMap: Record<string, MemberUser>,
    getMemberKey: (member: WorkspaceMember) => string
````
````old web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
      members: IWorkspaceMembership[],
````
````new web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
      members: WorkspaceMember[],
````
````old web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
      getMemberKey: (member: IWorkspaceMembership) => string
````
````new web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts
      getMemberKey: (member: WorkspaceMember) => string
````

`web/apps/web/core/store/member/workspace/workspace-member.store.test.ts`（新文件，201 行）：

````file web/apps/web/core/store/member/workspace/workspace-member.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceMember } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { memberStore, membershipOf } from "@/store/member/workspace/fake-members";
import type { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";

// The members of a workspace (M3 design 7.3), against a fake nerve that answers each request when the test says.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const MEMBERS = "/api/v0/workspaces/acme/members";

const ann = membershipOf("ann", { role: 20 });
const bob = membershipOf("bob");
/** A member who was removed: nerve lists his membership, ended. */
const cat = membershipOf("cat", { role: 5, is_active: false });

/** The store fetches acme's members, and nerve lists these. */
async function load(nerve: FakeNerve, store: WorkspaceMemberStore, memberships: WorkspaceMember[]) {
  const at = nerve.calls.length;
  const fetched = store.fetchWorkspaceMembers("acme");
  await until(() => nerve.calls.length === at + 1, "the members");
  nerve.calls[at]?.answer(json(200, { data: memberships }));
  return settle(fetched, "the members");
}

/** A store whose members of acme nerve gave as ann, bob and cat. */
async function loaded() {
  const tab = memberStore();
  await load(tab.nerve, tab.store, [ann, bob, cat]);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspaceMemberStore, the members", () => {
  it("keeps a workspace's memberships as nerve lists them, ended ones too, and shares the members' profiles", async () => {
    const { nerve, users, store } = memberStore();
    const fetched = await load(nerve, store, [ann, bob, cat]);
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: MEMBERS });
    expect(fetched.value).toEqual([ann, bob, cat]);
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": bob, "u-cat": cat } });
    expect(users).toEqual({ "u-ann": ann.member, "u-bob": bob.member, "u-cat": cat.member });
    expect(store.getWorkspaceMemberDetails("u-bob")).toEqual(bob);
    expect(store.getWorkspaceMemberDetails("u-zed")).toBeNull();
    expect(store.isUserSuspended("u-cat", "acme")).toBe(true);
    expect(store.isUserSuspended("u-bob", "acme")).toBe(false);

    // a fetch again holds the list as nerve has it now
    const promoted = membershipOf("bob", { role: 20 });
    await load(nerve, store, [ann, promoted]);
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": promoted } });
  });

  it("lists the caller first, then the others by display name", async () => {
    const { user, store } = await loaded();
    expect(store.getWorkspaceMemberIds("acme")).toEqual(["u-ann", "u-bob", "u-cat"]);
    runInAction(() => {
      user.data = {
        ...bob.member,
        email: "bob@example.com",
        user_timezone: "UTC",
        cover_image_url: null,
        created_at: "2026-09-01T09:00:00Z",
      };
    });
    expect(store.getWorkspaceMemberIds("acme")).toEqual(["u-bob", "u-ann", "u-cat"]);
  });

  it("fails when nerve cannot list them, keeping the members it had", async () => {
    const { nerve, store } = memberStore();
    const first = track(store.fetchWorkspaceMembers("acme"));
    await until(() => nerve.calls.length === 1, "the members");
    nerve.calls[0]?.answer(problem(404, "workspace.not_found"));
    await until(() => first.settled, "the failure");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberMap).toEqual({});

    await load(nerve, store, [ann, bob]);
    const again = track(store.fetchWorkspaceMembers("acme"));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.fail();
    await until(() => again.settled, "the failure");
    // The failure is the caller's to handle (SWR's error), not an unhandled rejection.
    expect(again.error).toBeInstanceOf(TypeError);
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": bob } });
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = memberStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchWorkspaceMembers("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.workspaceMemberMap).toEqual({});
  });
});

describe("WorkspaceMemberStore, the changes", () => {
  it("changes a member's role to nerve's answer, once nerve gives it", async () => {
    const { nerve, store } = await loaded();
    const demoted = membershipOf("bob", { role: 5 });
    const updated = track(store.updateMember("acme", "u-bob", { role: 5 }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/workspace-members/m-bob",
      body: { role: 5 },
    });
    // until nerve answers, the role is as it was
    expect(store.workspaceMemberMap.acme?.["u-bob"]).toEqual(bob);
    nerve.calls[1]?.answer(json(200, demoted));
    await until(() => updated.settled, "the answer");
    expect(updated.value).toEqual(demoted);
    expect(store.workspaceMemberMap.acme).toEqual({ "u-ann": ann, "u-bob": demoted, "u-cat": cat });
  });

  it("keeps a removed member's membership, ended, as nerve lists it", async () => {
    const { nerve, store } = await loaded();
    const removed = track(store.removeMemberFromWorkspace("acme", "u-bob"));
    await until(() => nerve.calls.length === 2, "the removal");
    expect(nerve.calls[1]).toMatchObject({ method: "DELETE", path: "/api/v0/workspace-members/m-bob" });
    expect(store.isUserSuspended("u-bob", "acme")).toBe(false);
    nerve.calls[1]?.answer(noContent());
    await until(() => removed.settled, "the removal");
    expect(removed.error).toBeUndefined();
    expect(store.workspaceMemberMap.acme?.["u-bob"]).toEqual({ ...bob, is_active: false });
    expect(store.isUserSuspended("u-bob", "acme")).toBe(true);
  });

  const refusals: { change: string; send: (store: WorkspaceMemberStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a role change",
      send: (store) => store.updateMember("acme", "u-ann", { role: 15 }),
      refusal: problem(409, "workspace.own_membership"),
    },
    {
      change: "a removal",
      send: (store) => store.removeMemberFromWorkspace("acme", "u-bob"),
      refusal: problem(409, "project.sole_admin"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const sent = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => sent.settled, "the refusal");
    expect(sent.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberMap.acme).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
  });

  it.each(refusals)("fails, asking nerve nothing, for $change of a member it has not listed", async ({ send }) => {
    const { nerve, store } = memberStore();
    const sent = await settle(send(store), "the change");
    expect(sent).toMatchObject({ settled: true, error: new Error("Member not found") });
    expect(nerve.calls).toEqual([]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const demoted = membershipOf("bob", { role: 5 });
    const promoted = track(store.updateMember("acme", "u-bob", { role: 20 }));
    const changed = track(store.updateMember("acme", "u-bob", { role: 5 }));
    const removed = track(store.removeMemberFromWorkspace("acme", "u-bob"));
    await inTurn(nerve, 1, ["PATCH", "/api/v0/workspace-members/m-bob"], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-members/m-bob"], json(200, demoted));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-members/m-bob"], noContent());
    await until(() => removed.settled, "the last change");
    expect(promoted.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(demoted);
    expect(store.workspaceMemberMap.acme?.["u-bob"]).toEqual({ ...demoted, is_active: false });
  });

  it("fetches the members while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      () => store.updateMember("acme", "u-bob", { role: 5 }),
      () => store.fetchWorkspaceMembers("acme"),
      ["GET", MEMBERS],
      json(200, { data: [ann, bob] })
    );
    expect(store.workspaceMemberMap).toEqual({ acme: { "u-ann": ann, "u-bob": bob } });
  });
});
````

`web/apps/web/core/store/member/workspace/workspace-member.store.ts`（修改，28 处）：

````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
// types
import type { EUserPermissions } from "@nerve/constants";
import type { IWorkspaceBulkInviteFormData, IWorkspaceMember, IWorkspaceMemberInvitation } from "@nerve/types";
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
// nerve imports
import type { ApiClient, WorkspaceMember, WorkspaceMemberUpdate } from "@nerve/api-client";
import type { IWorkspaceBulkInviteFormData, IWorkspaceMemberInvitation } from "@nerve/types";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
import { WorkspaceService } from "@/services/workspace.service";
import { WorkspaceMembersService } from "@/services/workspace/workspace-members.service";
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts

export interface IWorkspaceMembership {
  id: string;
  member: string;
  role: EUserPermissions;
  is_active?: boolean;
}

````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts

````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberMap: Record<string, Record<string, IWorkspaceMembership>>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  /** Each workspace's memberships by the member's account id, those that ended too (is_active false). */
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  memberMap: Record<string, IWorkspaceMembership> | null;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  memberMap: Record<string, WorkspaceMember> | null;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  getWorkspaceMemberDetails: (workspaceMemberId: string) => IWorkspaceMember | null;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  getWorkspaceMemberDetails: (userId: string) => WorkspaceMember | null;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMembers: (workspaceSlug: string) => Promise<IWorkspaceMember[]>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMembers: (workspaceSlug: string) => Promise<WorkspaceMember[] | undefined>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  updateMember: (workspaceSlug: string, userId: string, data: { role: EUserPermissions }) => Promise<void>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  updateMember: (workspaceSlug: string, userId: string, data: WorkspaceMemberUpdate) => Promise<WorkspaceMember>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
}

export class WorkspaceMemberStore implements IWorkspaceMemberStore {
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
}

/**
 * The members of the workspaces of a session (M3 design 7.3): its service sends with the session's client, which
 * the RootStore of the session hands down. Changes go one at a time (v0 design 7.7); fetches do not queue.
 */
export class WorkspaceMemberStore implements IWorkspaceMemberStore {
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberMap: {
    [workspaceSlug: string]: Record<string, IWorkspaceMembership>;
  } = {}; // { workspaceSlug: { userId: userDetails } }
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>> = {};
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  memberRoot: IMemberRootStore;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  /** The users the stores read, which the members' profiles join. */
  memberRoot: Pick<IMemberRootStore, "memberMap">;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceService;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceService;
  private readonly service: WorkspaceMembersService;
  /** The changes of the memberships, sent one at a time. */
  private readonly changes = oneAtATime();
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  constructor(_memberRoot: IMemberRootStore, _rootStore: RootStore) {
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  constructor(_memberRoot: Pick<IMemberRootStore, "memberMap">, _rootStore: RootStore, api: ApiClient) {
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    this.workspaceService = new WorkspaceService();
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    this.workspaceService = new WorkspaceService();
    this.service = new WorkspaceMembersService(api);
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    let members = Object.values(this.workspaceMemberMap?.[workspaceSlug] ?? {});
    members = sortBy(members, [
      (m) => m.member !== this.userStore?.data?.id,
      (m) => this.memberRoot?.memberMap?.[m.member]?.display_name?.toLowerCase(),
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    const members = sortBy(Object.values(this.workspaceMemberMap?.[workspaceSlug] ?? {}), [
      (m) => m.member.id !== this.userStore?.data?.id,
      (m) => m.member.display_name.toLowerCase(),
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    //filter out bots
    const memberIds = members.filter((m) => !this.memberRoot?.memberMap?.[m.member]?.is_bot).map((m) => m.member);
    return memberIds;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    return members.map((m) => m.member.id);
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  getFilteredWorkspaceMemberIds = computedFn((workspaceSlug: string) => {
    let members = Object.values(this.workspaceMemberMap?.[workspaceSlug] ?? {});
    //filter out bots and inactive members
    members = members.filter((m) => !this.memberRoot?.memberMap?.[m.member]?.is_bot);

    // Use filters store to get filtered member ids
    const memberIds = this.filtersStore.getFilteredMemberIds(
      members,
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  getFilteredWorkspaceMemberIds = computedFn((workspaceSlug: string) =>
    this.filtersStore.getFilteredMemberIds(
      Object.values(this.workspaceMemberMap?.[workspaceSlug] ?? {}),
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
      (member) => member.member
    );

    return memberIds;
  });
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
      (member) => member.member.id
    )
  );
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    const workspaceMember = this.workspaceMemberMap?.[workspaceSlug]?.[userId];
    if (!workspaceMember) return null;

    const memberDetails: IWorkspaceMember = {
      id: workspaceMember.id,
      role: workspaceMember.role,
      member: this.memberRoot?.memberMap?.[workspaceMember.member],
      is_active: workspaceMember.is_active,
    };
    return memberDetails;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    return this.workspaceMemberMap?.[workspaceSlug]?.[userId] ?? null;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description fetch all the members of a workspace
   * @param workspaceSlug
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description fetches a workspace's memberships, those that ended too, and gives them; each member's profile
   * joins the users the other stores read (memberRoot.memberMap). A change of session while they load is no
   * failure: the new session's store fetches its own (store-context.tsx), and this one gives undefined.
   * @returns {Promise<WorkspaceMember[] | undefined>}
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMembers = async (workspaceSlug: string) =>
    await this.workspaceService.fetchWorkspaceMembers(workspaceSlug).then((response) => {
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMembers = async (workspaceSlug: string): Promise<WorkspaceMember[] | undefined> => {
    try {
      const memberships = await this.service.list(workspaceSlug);
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
        response.forEach((member) => {
          set(this.memberRoot?.memberMap, member.member.id, { ...member.member, joining_date: member.created_at });
          set(this.workspaceMemberMap, [workspaceSlug, member.member.id], {
            id: member.id,
            member: member.member.id,
            role: member.role,
            is_active: member.is_active,
          });
        });
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
        for (const membership of memberships) set(this.memberRoot.memberMap, membership.member.id, membership.member);
        this.workspaceMemberMap[workspaceSlug] = Object.fromEntries(memberships.map((m) => [m.member.id, m]));
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
      return response;
    });

  /**
   * @description update the role of a workspace member
   * @param workspaceSlug
   * @param userId
   * @param data
   */
  updateMember = async (workspaceSlug: string, userId: string, data: { role: EUserPermissions }) => {
    const memberDetails = this.getWorkspaceMemberDetails(userId);
    if (!memberDetails) throw new Error("Member not found");
    // original data to revert back in case of error
    const originalProjectMemberData = { ...this.workspaceMemberMap?.[workspaceSlug]?.[userId] };
    try {
      runInAction(() => {
        set(this.workspaceMemberMap, [workspaceSlug, userId, "role"], data.role);
      });
      await this.workspaceService.updateWorkspaceMember(workspaceSlug, memberDetails.id, data);
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
      return memberships;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
      // revert back to original members in case of error
      runInAction(() => {
        set(this.workspaceMemberMap, [workspaceSlug, userId], originalProjectMemberData);
      });
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
      if (error instanceof SessionChangedError) return undefined;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description remove a member from workspace
   * @param workspaceSlug
   * @param userId
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description changes the role of a member of the workspace; the store then has nerve's answer. Fails,
   * changing nothing, when nerve refuses or the store has no membership of his.
   * @returns {Promise<WorkspaceMember>}
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  removeMemberFromWorkspace = async (workspaceSlug: string, userId: string) => {
    const memberDetails = this.getWorkspaceMemberDetails(userId);
    if (!memberDetails) throw new Error("Member not found");
    // oxlint-disable-next-line promise/always-return
    await this.workspaceService.deleteWorkspaceMember(workspaceSlug, memberDetails?.id).then(() => {
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  updateMember = (workspaceSlug: string, userId: string, data: WorkspaceMemberUpdate): Promise<WorkspaceMember> =>
    this.changes(async () => {
      const membership = await this.service.update(this.membership(workspaceSlug, userId).id, data);
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
        set(this.workspaceMemberMap, [workspaceSlug, userId, "is_active"], false);
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
        set(this.workspaceMemberMap, [workspaceSlug, userId], membership);
      });
      return membership;
    });

  /**
   * @description ends the membership of a member of the workspace, which the store then keeps as ended
   * (is_active false), as nerve lists it. Fails, changing nothing, when nerve refuses or the store has no
   * membership of his.
   * @returns {Promise<void>}
   */
  removeMemberFromWorkspace = (workspaceSlug: string, userId: string): Promise<void> =>
    this.changes(async () => {
      const membership = this.membership(workspaceSlug, userId);
      await this.service.remove(membership.id);
      runInAction(() => {
        set(this.workspaceMemberMap, [workspaceSlug, userId], { ...membership, is_active: false });
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    });
  };
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    });

  /** The membership of the member userId names in the workspace, as the store has it; fails when it has none. */
  private membership(workspaceSlug: string, userId: string): WorkspaceMember {
    const membership = this.workspaceMemberMap[workspaceSlug]?.[userId];
    if (!membership) throw new Error("Member not found");
    return membership;
  }
````

`web/apps/web/core/store/root.store.ts`（修改，1 处）：

````old web/apps/web/core/store/root.store.ts
    this.memberRoot = new MemberRootStore(this);
````
````new web/apps/web/core/store/root.store.ts
    this.memberRoot = new MemberRootStore(this, api);
````

- [ ] **Step 4: `MemberUser`、`WorkspaceMember` 的使用方**

`web/apps/web/core/components/modules/links/list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/modules/links/list-item.tsx
          Added {calculateTimeAgo(link.created_at)}{" "}
          {createdByDetails && (
            <>by {createdByDetails?.is_bot ? createdByDetails?.first_name + " Bot" : createdByDetails?.display_name}</>
          )}
````
````new web/apps/web/core/components/modules/links/list-item.tsx
          Added {calculateTimeAgo(link.created_at)} {createdByDetails && <>by {createdByDetails.display_name}</>}
````

`web/apps/web/core/components/profile/sidebar.tsx`（修改，1 处）：

````old web/apps/web/core/components/profile/sidebar.tsx
          <div className="w-3/5 font-medium break-words">{renderFormattedDate(userData.joining_date ?? "")}</div>
````
````new web/apps/web/core/components/profile/sidebar.tsx
          <div className="w-3/5 font-medium break-words">{renderFormattedDate(profileMember.joinedAt)}</div>
````

`web/apps/web/core/components/profile/use-profile-member.ts`（修改，4 处）：

````old web/apps/web/core/components/profile/use-profile-member.ts
  | { status: "member"; member: MemberUser };
````
````new web/apps/web/core/components/profile/use-profile-member.ts
  | { status: "member"; member: MemberUser; joinedAt: string };
````
````old web/apps/web/core/components/profile/use-profile-member.ts
  // subscribing here is what tells whether the members are still loading or failed to load. A failed load
  // shows as a settled request without a list, not as SWR's `error`: the service rethrows only the response
  // body, which is undefined when the request got no response at all.
````
````new web/apps/web/core/components/profile/use-profile-member.ts
  // subscribing here is what tells whether the members are still loading or failed to load: a failed load
  // leaves the request settled without a list.
````
````old web/apps/web/core/components/profile/use-profile-member.ts
  const member = memberDetails?.is_active === false ? undefined : memberDetails?.member;
````
````new web/apps/web/core/components/profile/use-profile-member.ts
  const membership = memberDetails?.is_active === false ? undefined : memberDetails;
````
````old web/apps/web/core/components/profile/use-profile-member.ts
  if (member) return { status: "member", member };
````
````new web/apps/web/core/components/profile/use-profile-member.ts
  if (membership) return { status: "member", member: membership.member, joinedAt: membership.created_at };
````

`web/apps/web/core/components/project/add-project-members-modal.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/add-project-members-modal.tsx
    await bulkAddMembersToProject(workspaceSlug, projectId, payload)
      .then(() => {
        if (onSuccess) onSuccess();
        onClose();
        setToast({
          title: "Success!",
          type: TOAST_TYPE.SUCCESS,
          message: "Members added successfully.",
        });
      })
      .catch((error) => {
        console.error(error);
      })
      .finally(() => {
        reset(defaultValues);
      });
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
    try {
      await bulkAddMembersToProject(workspaceSlug, projectId, payload);
      if (onSuccess) onSuccess();
      onClose();
      setToast({
        title: "Success!",
        type: TOAST_TYPE.SUCCESS,
        message: "Members added successfully.",
      });
    } catch (error) {
      console.error(error);
    } finally {
      reset(defaultValues);
    }
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
      Object.entries(ROLE).filter(([key]) => !isGuestOROwner || [currentMemberWorkspaceRole].includes(parseInt(key)))
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
      Object.entries(ROLE).filter(([key]) => !isGuestOROwner || parseInt(key) === currentMemberWorkspaceRole)
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                      render={({ field }) => (
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                      render={({ field: roleField }) => (
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                          {...field}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                          {...roleField}
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                              <span className="capitalize">{field.value ? ROLE[field.value] : "Select role"}</span>
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                              <span className="capitalize">
                                {roleField.value ? ROLE[roleField.value] : "Select role"}
                              </span>
````

`web/apps/web/core/components/project/settings/member-columns.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/settings/member-columns.tsx
import type { User } from "@nerve/api-client";
import type { EUserProjectRoles, IWorkspaceMember, TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
import type { MemberUser, User } from "@nerve/api-client";
import type { EUserProjectRoles, TProjectMembership } from "@nerve/types";
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
interface RowData extends Pick<TProjectMembership, "original_role"> {
  member: IWorkspaceMember;
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
interface RowData extends Pick<TProjectMembership, "original_role" | "created_at"> {
  member: MemberUser;
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
                      alt={display_name || email}
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
                      alt={display_name || (email ?? undefined)}
````

`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`（修改，3 处）：

````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import type { IWorkspaceMember, TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import type { MemberUser } from "@nerve/api-client";
import type { TProjectMembership } from "@nerve/types";
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
export interface RowData extends Pick<TProjectMembership, "original_role"> {
  member: IWorkspaceMember;
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
export interface RowData extends Pick<TProjectMembership, "original_role" | "created_at"> {
  member: MemberUser;
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: RowData) => <div>{renderFormattedDate(rowData?.member?.joining_date)}</div>,
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: RowData) => <div>{renderFormattedDate(rowData.created_at)}</div>,
````

`web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx
  const renderTriggerName = () => (
    <span className="font-medium text-primary">
      {triggeredBy?.is_bot ? triggeredBy.first_name : triggeredBy?.display_name}{" "}
    </span>
  );
````
````new web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx
  const renderTriggerName = () => <span className="font-medium text-primary">{triggeredBy?.display_name} </span>;
````

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/settings/member-columns.tsx
import type { User } from "@nerve/api-client";
import type { IWorkspaceMember } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
import type { User, WorkspaceMember } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx

export interface RowData {
  member: IWorkspaceMember;
  role: EUserPermissions;
  is_active: boolean;
}

type NameProps = {
  rowData: RowData;
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx

type NameProps = {
  rowData: WorkspaceMember;
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
  setRemoveMemberModal: (rowData: RowData) => void;
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
  setRemoveMemberModal: (rowData: WorkspaceMember) => void;
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
type AccountTypeProps = {
  rowData: RowData;
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
type AccountTypeProps = {
  rowData: WorkspaceMember;
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
                      alt={display_name || email}
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
                      alt={display_name || (email ?? undefined)}
````

`web/apps/web/core/components/workspace/settings/members-list-item.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
import type { IWorkspaceMember } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
import type { WorkspaceMember } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
import type { RowData } from "@/components/workspace/settings/member-columns";
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
  memberDetails: (IWorkspaceMember | null)[];
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
  memberDetails: (WorkspaceMember | null)[];
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
      <Table<RowData>
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
      <Table<WorkspaceMember>
````
````old web/apps/web/core/components/workspace/settings/members-list-item.tsx
        data={
          (memberDetails?.filter((member): member is IWorkspaceMember => member !== null) ?? []) as unknown as RowData[]
        }
````
````new web/apps/web/core/components/workspace/settings/members-list-item.tsx
        data={memberDetails.filter((member): member is WorkspaceMember => member !== null)}
````

`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`（修改，8 处）：

````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
import type { RowData } from "@/components/workspace/settings/member-columns";
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
import type { WorkspaceMember } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  const [removeMemberModal, setRemoveMemberModal] = useState<RowData | null>(null);
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  const [removeMemberModal, setRemoveMemberModal] = useState<WorkspaceMember | null>(null);
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  const isSuspended = (rowData: RowData) => rowData.is_active === false;
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  const isSuspended = (rowData: WorkspaceMember) => rowData.is_active === false;
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      ),
      tdRender: (rowData: RowData) =>
        workspaceSlug && (
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      ),
      tdRender: (rowData: WorkspaceMember) =>
        workspaceSlug && (
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      content: t("workspace_settings.settings.members.details.display_name"),
      tdRender: (rowData: RowData) => (
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      content: t("workspace_settings.settings.members.details.display_name"),
      tdRender: (rowData: WorkspaceMember) => (
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      content: t("workspace_settings.settings.members.details.email_address"),
      tdRender: (rowData: RowData) => (
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      content: t("workspace_settings.settings.members.details.email_address"),
      tdRender: (rowData: WorkspaceMember) => (
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      ),
      tdRender: (rowData: RowData) =>
        workspaceSlug && <AccountTypeColumn rowData={rowData} workspaceSlug={workspaceSlug} />,
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      ),
      tdRender: (rowData: WorkspaceMember) =>
        workspaceSlug && <AccountTypeColumn rowData={rowData} workspaceSlug={workspaceSlug} />,
````
````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      tdRender: (rowData: RowData) =>
        isSuspended(rowData) ? null : <div>{renderFormattedDate(rowData?.member?.joining_date)}</div>,
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
      tdRender: (rowData: WorkspaceMember) =>
        isSuspended(rowData) ? null : <div>{renderFormattedDate(rowData.created_at)}</div>,
````

`web/packages/types/src/users.ts`（修改，1 处）：

````old web/packages/types/src/users.ts

export interface IUserLite {
  avatar_url: string;
  display_name: string;
  email?: string;
  first_name: string;
  id: string;
  is_bot: boolean;
  last_name: string;
  joining_date?: string;
}

````
````new web/packages/types/src/users.ts

````

`web/packages/types/src/workspace.ts`（修改，2 处）：

````old web/packages/types/src/workspace.ts
import type { TUserPermissions } from "./enums";
import type { MemberUser } from "@nerve/api-client";
````
````new web/packages/types/src/workspace.ts
import type { TUserPermissions } from "./enums";
````
````old web/packages/types/src/workspace.ts
  emails: { email: string; role: TUserPermissions }[];
}

export interface IWorkspaceMember {
  id: string;
  member: MemberUser;
  role: TUserPermissions | EUserWorkspaceRoles;
  created_at?: string;
  avatar_url?: string;
  email?: string;
  first_name?: string;
  last_name?: string;
  joining_date?: string;
  display_name?: string;
  is_active?: boolean;
````
````new web/packages/types/src/workspace.ts
  emails: { email: string; role: TUserPermissions }[];
````

`web/packages/utils/src/file.ts`（修改，2 处）：

````old web/packages/utils/src/file.ts
 * an empty path means there is no file
 * @param {string} path
````
````new web/packages/utils/src/file.ts
 * a null or empty path means there is no file
 * @param {string | null} path
````
````old web/packages/utils/src/file.ts
export const getFileURL = (path: string): string | undefined => path || undefined;
````
````new web/packages/utils/src/file.ts
export const getFileURL = (path: string | null): string | undefined => path || undefined;
````

- [ ] **Step 5: 上限、关键词规则**

`tools/keywords.json`（修改，3 处）：

````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/leave/|/workspace-members/me/",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/",
````
````old tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/workspace-members/me/`)"
````
````new tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/workspace-members/me/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/members/`)",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/members/${memberId}/`, data)"
````
````old tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/project-members/me/`)"
````
````new tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/project-members/me/`)",
          "    return unwrap(await this.api.GET(\"/api/v0/workspaces/{slug}/members\", { params: { path: { slug } } })).data;",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`, data)"
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 448",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 445",
````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则，没有命中；web 的 oxlint 445 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过（W4 的成员接口版本照旧）。

- [ ] **Step 7: 提交**

```bash
git add tools/keywords.json web/apps/web/core/components/dropdowns/member/base.tsx web/apps/web/core/components/dropdowns/member/member-options.tsx web/apps/web/core/components/modules/links/list-item.tsx web/apps/web/core/components/navigation/tab-navigation-root.tsx web/apps/web/core/components/navigation/use-navigation-items.ts web/apps/web/core/components/profile/sidebar.tsx web/apps/web/core/components/profile/use-profile-member.ts web/apps/web/core/components/project/add-project-members-modal.tsx web/apps/web/core/components/project/confirm-project-member-remove.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/project/settings/member-columns.tsx web/apps/web/core/components/projects/settings/useProjectColumns.tsx web/apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx web/apps/web/core/components/workspace/settings/member-columns.tsx web/apps/web/core/components/workspace/settings/members-list-item.tsx web/apps/web/core/components/workspace/settings/useMemberColumns.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/services/workspace.service.ts web/apps/web/core/services/workspace/workspace-members.service.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/member/index.ts web/apps/web/core/store/member/project/project-member-filters.store.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/member/utils.ts web/apps/web/core/store/member/workspace/fake-members.ts web/apps/web/core/store/member/workspace/workspace-member-filters.store.ts web/apps/web/core/store/member/workspace/workspace-member.store.test.ts web/apps/web/core/store/member/workspace/workspace-member.store.ts web/apps/web/core/store/notifications/notification.ts web/apps/web/core/store/root.store.ts web/apps/web/package.json web/packages/types/src/project/projects.ts web/packages/types/src/search.ts web/packages/types/src/users.ts web/packages/types/src/workspace-notifications.ts web/packages/types/src/workspace.ts web/packages/typescript-config/react-router.json web/packages/utils/src/file.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts
```
```bash
git commit -m "feat(M3/P8a): a workspace's members come from /api/v0; MemberUser replaces IUserLite

WorkspaceMembersService wraps the generated client, and the workspace
member store, built per session, keeps every membership nerve lists,
ended ones too, and shares the members' public profiles. Role changes
and removals go one at a time, take nerve's answer, and ask nothing for
a member the store has not listed. MemberUser replaces IUserLite in
every consumer, eighteen files by a mechanical rename: the nullable
avatar and email read as null, and a member joined when his membership
began. The web app's TypeScript library is ES2023, so the members' sort
and the two navigation sorts are toSorted, with no suppression.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t7-members-unqueued` | 成员的修改不互相等待 | `workspace-member.store.test.ts` | vitest |
| `t7-members-merged` | 取数把成员关系并进 store 原有的，不是换成 nerve 给的列表 | `workspace-member.store.test.ts` | vitest |
| `t7-members-no-users` | 成员的公开资料不进各 store 共用的用户表 | `workspace-member.store.test.ts` | vitest |
| `t7-members-session` | 成员加载时换了会话，取数失败 | oxlint（`check:lint`）、`workspace-member.store.test.ts` | 静态；vitest |
| `t7-role-early` | 成员的角色在 nerve 应答之前就改了 | `workspace-member.store.test.ts` | vitest |
| `t7-remove-kept` | 移出的成员在 store 里仍是有效成员 | `workspace-member.store.test.ts` | vitest |
| `t7-remove-early` | 成员在 nerve 应答之前就移出了 | `workspace-member.store.test.ts` | vitest |
| `t7-unknown-member` | store 没有列出的成员，修改照样发出 | `workspace-member.store.test.ts` | vitest |
| `t7-caller-not-first` | 成员只按名字排列，调用者混在其中 | `workspace-member.store.test.ts` | vitest |
| `t7-members-back` | 旧 service 又列出工作区的成员 | 关键词守卫 | 静态 |
| `t7-null-email` | 成员的 `email`（调用者无权看时是 `null`）原样交给图片的 `alt` | `tsc` | 静态 |
| `t7-members-fetch-queued` | 成员的取数等待正在发出的修改 | `workspace-member.store.test.ts` | vitest |
| `t7-sort-shared` | 成员排序就地排 store 交出的数组（`sort`，不是 `toSorted`） | oxlint（`check:lint`） | 静态 |
| `t7-members-swallows` | 工作区成员的列表取不到时给出 `undefined`，不失败 | `workspace-member.store.test.ts` | vitest |

---

### Task 8: 邀请的 service 和 store；成员页的邀请只为管理员取；工作区的页面按会话取成员

**Files:**
- Create: `web/apps/web/core/components/workspace/settings/use-members-settings-fetch.test.ts`、`web/apps/web/core/components/workspace/settings/use-members-settings-fetch.ts`、`web/apps/web/core/services/workspace/workspace-invitations.service.ts`、`web/apps/web/core/store/member/workspace/workspace-invitations.test.ts`
- Modify: `.oxlintrc.json`、`tools/keywords.json`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`、`web/apps/web/core/components/onboarding/steps/team/root.tsx`、`web/apps/web/core/components/profile/use-profile-member.ts`、`web/apps/web/core/components/workspace/members/invite-modal.tsx`、`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`、`web/apps/web/core/components/workspace/settings/members-list.tsx`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/member/workspace/workspace-member.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/types/src/workspace.ts`

**Interfaces:**
- Produces（spec 2.8；M3 设计 3.8、5.1、7.1、7.3；决策点 4）：
  - `core/services/workspace/workspace-invitations.service.ts`：`class WorkspaceInvitationsService { list(slug); create(slug, data: WorkspaceInvitationsCreate): Promise<WorkspaceInvitation[]>; update(invitationId, data: WorkspaceInvitationUpdate); delete(invitationId) }`。
  - `WorkspaceMemberStore` 的邀请：`workspaceMemberInvitations[slug]` 是 nerve 列给管理员的邀请（待接受的和已忽略的，最新的在前），每次取数整份换掉，只在 store 的类里，不在接口上（没有组件读它）；`fetchWorkspaceMemberInvitations(slug): Promise<WorkspaceInvitation[] | undefined>`（`SessionChangedError` 给出 `undefined`；被拒绝时原样失败，不留任何邀请）；`inviteMembersToWorkspace(slug, data: WorkspaceInvitationsCreate): Promise<WorkspaceInvitation[]>`（全部或全不，回答的新邀请放在已取的列表最前，没有取过时不建列表）、`updateMemberInvitation(slug, id, data: WorkspaceInvitationUpdate): Promise<WorkspaceInvitation>`（写入回答）、`deleteMemberInvitation(slug, id)`（移出列表）：都经同一个 `changes = oneAtATime()`，被拒绝时什么都不改。`IWorkspaceMemberInvitation`、`IWorkspaceBulkInviteFormData` 删除（7.2），旧 `WorkspaceService` 的四个邀请方法删除。
  - `core/components/workspace/settings/use-members-settings-fetch.ts`：`useMembersSettingsFetch(workspaceSlug)`：`useSessionSWR(["WORKSPACE_MEMBERS", slug], …)` 给每个看得到这一页的人；`useSessionSWR(isAdmin ? ["WORKSPACE_INVITATIONS", slug] : null, …)` 只给工作区管理员（7.1；`members-list.tsx` 原来不分角色先取邀请）。`isAdmin` 由 hook 自己算出：`useWorkspace().getWorkspaceBySlug(slug)?.role === EUserWorkspaceRoles.ADMIN`，即调用者的工作区列表给出的角色（H1：条件不由调用方传入，写错的调用方不能绕过它）。`members-list.tsx` 调用 `useMembersSettingsFetch(workspaceSlug)`，它的 `isAdmin` prop 只决定显示。
  - `useWorkspaceFetch(workspaceSlug)`（整个文件）：列表之外，列表说明这是调用者的工作区时（hook 自己算出 `isMember = workspaceSlug !== undefined && getWorkspaceBySlug(workspaceSlug) !== null`）才取它的成员（`["WORKSPACE_MEMBERS", slug]`，与成员页同一个键）；`WorkspaceAuthWrapper` 调用 `useWorkspaceFetch(workspaceSlug)`，它原来按 slug 取成员的 SWR 删除。
  - 根目录 `.oxlintrc.json` 的 `overrides`（裁定 A6、L4；路径都从仓库根写起，oxlint 按根配置所在的目录匹配，从 `web/apps/web` 写起的路径什么都不匹配）：
    - 会话的取数只经 `useSessionSWR`：`no-restricted-imports` 禁止从 `swr` 导入值（`allowTypeImports: true`，`use-workspace-fetch.ts` 的 `import type { SWRResponse }` 照旧），范围是工作区一侧的 store（`core/store/workspace/**`、`core/store/member/workspace/**`）、`core/components/workspace/settings/**`、`use-profile-member.ts`、`use-workspace-fetch.ts`、`core/lib/use-landing.ts` 和 `core/lib/wrappers/authentication-wrapper.tsx`（`core/lib/wrappers/` 按文件列，`instance-wrapper.tsx` 取公开的实例信息）。不在范围内的：`use-session-swr.ts` 本身、`fake-session-swr.ts`、唯一的例外 `use-invitation-preview.ts`（公开操作，键只有链接，Task 9），以及仍用 `useSWR` 取项目一侧的 `workspace-wrapper.tsx`、`project-wrapper.tsx`（P8b 把它们加进来，spec 附录 A.6）。
    - P8a 的 M3 路径不用非空断言：`typescript/no-non-null-assertion` 为 `error`，范围是上面的 store 和组件，加上 `core/store/user/permissions.store.ts`、`core/services/workspace/**`、`use-workspace-fetch.ts` 和它的测试、`core/lib` 中 P8a 新写的 13 个文件（逐个列出；`core/lib/**` 不行：M2 的 `token-manager.tabs.test.ts` 有 25 处）。
    - 这一步放在 Task 8，因为这里出现最后一个在范围内的文件（`use-members-settings-fetch.ts`、成员一侧的取数）；两条规则在同一个 `overrides` 数组里，各有自己的文件列表（`use-session-swr.ts` 在第二条的范围内、不在第一条的）。
  - 使用方：邀请弹窗（`invite-modal.tsx`）、新手引导的邀请一步（`team/root.tsx`）改调 `inviteMembersToWorkspace`，`invitations-list-item.tsx` 用 `WorkspaceInvitation`；个人主页的成员（`use-profile-member.ts`）改用会话的键取成员。
- Adds：`plane-workspace-urls` 加工作区邀请的旧地址；不命中样例加新的邀请地址和项目邀请（P8b）。

**Tests:**（vitest）
- `core/store/member/workspace/workspace-invitations.test.ts`，`WorkspaceMemberStore, the invitations`：`keeps a workspace's invitations as nerve lists them to an admin, declined ones too`；`fails when nerve refuses the list, keeping none`；`gives nothing, and does not fail, when the session changes as it fetches`；`puts the new invitations first in the list it has, once nerve gives them, and in none it has not fetched`；`changes an invitation's role to nerve's answer, and takes a deleted one off the list`；`fails, changing nothing, when nerve refuses $change`；`sends each change once nerve has answered the one before it, refused or not`。
- `use-members-settings-fetch.test.ts`（9.5 的成员页的取数条件；工作区 store 换成桩，`acme` 的角色是列表给的）：`fetches an admin the members and the invitations`；`it.each` 的 `fetches $who the members alone`（成员、访客、列表中没有这个工作区的调用者，3 行）（W5）。
- `use-workspace-fetch.test.ts`（桩的列表只有 `acme`）：`fetches the caller's workspaces and, in one of his, its members`（`acme`）；`fetches only the caller's workspaces where the address names none of his`（`elsewhere`）（原来的 `fetches the caller's workspaces` 改成这两个）。

- [ ] **Step 1: 邀请的 service 和 store**

`web/apps/web/core/services/workspace.service.ts`（修改，4 处）：

````old web/apps/web/core/services/workspace.service.ts
  IWorkspaceSearchResults,
  IWorkspaceBulkInviteFormData,
````
````new web/apps/web/core/services/workspace.service.ts
  IWorkspaceSearchResults,
````
````old web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
  async inviteWorkspace(workspaceSlug: string, data: IWorkspaceBulkInviteFormData): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/invitations/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
````
````old web/apps/web/core/services/workspace.service.ts

  async workspaceInvitations(workspaceSlug: string): Promise<IWorkspaceMemberInvitation[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/invitations/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts

````
````old web/apps/web/core/services/workspace.service.ts
    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateWorkspaceInvitation(
    workspaceSlug: string,
    invitationId: string,
    data: Partial<IWorkspaceMemberInvitation>
  ): Promise<any> {
    return this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteWorkspaceInvitations(workspaceSlug: string, invitationId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`)
````
````new web/apps/web/core/services/workspace.service.ts
    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })
````

`web/apps/web/core/services/workspace/workspace-invitations.service.ts`（新文件，49 行）：

````file web/apps/web/core/services/workspace/workspace-invitations.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type {
  ApiClient,
  WorkspaceInvitation,
  WorkspaceInvitationUpdate,
  WorkspaceInvitationsCreate,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A workspace's invitations, as its admins manage them (M3 design 5.1, 7.3). */
export class WorkspaceInvitationsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The invitations, pending and declined, newest first, each with the token of its link. */
  async list(slug: string): Promise<WorkspaceInvitation[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/invitations", { params: { path: { slug } } })).data;
  }

  /** Invites the addresses data lists, all of them or none; the answer is the new invitations, in data's order. */
  async create(slug: string, data: WorkspaceInvitationsCreate): Promise<WorkspaceInvitation[]> {
    return unwrap(
      await this.api.POST("/api/v0/workspaces/{slug}/invitations", { params: { path: { slug } }, body: data })
    ).data;
  }

  /** Changes an invitation's role; the answer is the invitation as nerve now holds it. */
  async update(invitationId: string, data: WorkspaceInvitationUpdate): Promise<WorkspaceInvitation> {
    return unwrap(
      await this.api.PATCH("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: { invitation_id: invitationId } },
        body: data,
      })
    );
  }

  /** Deletes an invitation: its link stops working. */
  async delete(invitationId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: { invitation_id: invitationId } },
      })
    );
  }
}
````

`web/apps/web/core/store/member/workspace/workspace-invitations.test.ts`（新文件，188 行）：

````file web/apps/web/core/store/member/workspace/workspace-invitations.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import { memberStore } from "@/store/member/workspace/fake-members";
import type { WorkspaceMemberStore } from "@/store/member/workspace/workspace-member.store";

// A workspace's invitations, as its admins manage them (M3 design 7.3), against a fake nerve that answers each
// request when the test says. Who may fetch them is use-members-settings-fetch.test.ts.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const INVITATIONS = "/api/v0/workspaces/acme/invitations";

/** An invitation of acme as nerve lists it to an admin: the name names the address; pending, unless fields say not. */
function invitationOf(name: string, fields: Partial<WorkspaceInvitation> = {}): WorkspaceInvitation {
  return {
    id: `i-${name}`,
    workspace_id: "id-acme",
    email: `${name}@example.com`,
    role: 15,
    accepted: false,
    responded_at: null,
    created_at: "2026-10-02T09:00:00Z",
    created_by_id: "u-ann",
    token: `nrv_inv_${name}`,
    ...fields,
  };
}
const dan = invitationOf("dan");
/** An invitation its address declined: nerve lists it until an admin deletes it. */
const eve = invitationOf("eve", { role: 5, responded_at: "2026-10-03T09:00:00Z" });

/** The store fetches acme's invitations, and nerve lists these. */
async function load(nerve: FakeNerve, store: WorkspaceMemberStore, invitations: WorkspaceInvitation[]) {
  const at = nerve.calls.length;
  const fetched = store.fetchWorkspaceMemberInvitations("acme");
  await until(() => nerve.calls.length === at + 1, "the invitations");
  nerve.calls[at]?.answer(json(200, { data: invitations }));
  return settle(fetched, "the invitations");
}

/** A store whose invitations of acme nerve gave as dan's and eve's. */
async function loaded() {
  const tab = memberStore();
  await load(tab.nerve, tab.store, [dan, eve]);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspaceMemberStore, the invitations", () => {
  it("keeps a workspace's invitations as nerve lists them to an admin, declined ones too", async () => {
    const { nerve, store } = memberStore();
    const fetched = await load(nerve, store, [dan, eve]);
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: INVITATIONS });
    expect(fetched.value).toEqual([dan, eve]);
    expect(store.workspaceMemberInvitations).toEqual({ acme: [dan, eve] });
    expect(store.workspaceMemberInvitationIds).toEqual(["i-dan", "i-eve"]);
    expect(store.getWorkspaceInvitationDetails("i-eve")).toEqual(eve);
    expect(store.getSearchedWorkspaceInvitationIds("EVE@")).toEqual(["i-eve"]);
  });

  it("fails when nerve refuses the list, keeping none", async () => {
    const { nerve, store } = memberStore();
    const refused = track(store.fetchWorkspaceMemberInvitations("acme"));
    await until(() => nerve.calls.length === 1, "the invitations");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberInvitations).toEqual({});
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = memberStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchWorkspaceMemberInvitations("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.workspaceMemberInvitations).toEqual({});
  });

  it("puts the new invitations first in the list it has, once nerve gives them, and in none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const body: WorkspaceInvitationsCreate = { invitations: [{ email: "fay@example.com", role: 15 }] };
    const fay = invitationOf("fay");
    const invited = track(store.inviteMembersToWorkspace("acme", body));
    await until(() => nerve.calls.length === 2, "the invitation");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: INVITATIONS, body });
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
    nerve.calls[1]?.answer(json(201, { data: [fay] }));
    await until(() => invited.settled, "the new invitations");
    expect(invited.value).toEqual([fay]);
    expect(store.workspaceMemberInvitations.acme).toEqual([fay, dan, eve]);

    const fresh = memberStore();
    const alone = track(fresh.store.inviteMembersToWorkspace("acme", body));
    await until(() => fresh.nerve.calls.length === 1, "the invitation");
    fresh.nerve.calls[0]?.answer(json(201, { data: [fay] }));
    await until(() => alone.settled, "the new invitations");
    // a list of the new invitations alone would show as the whole list
    expect(fresh.store.workspaceMemberInvitations).toEqual({});
  });

  it("changes an invitation's role to nerve's answer, and takes a deleted one off the list", async () => {
    const { nerve, store } = await loaded();
    const promoted = invitationOf("dan", { role: 20 });
    const changed = track(store.updateMemberInvitation("acme", "i-dan", { role: 20 }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: "/api/v0/workspace-invitations/i-dan",
      body: { role: 20 },
    });
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
    nerve.calls[1]?.answer(json(200, promoted));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(promoted);
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted, eve]);

    const deleted = track(store.deleteMemberInvitation("acme", "i-eve"));
    await until(() => nerve.calls.length === 3, "the deletion");
    expect(nerve.calls[2]).toMatchObject({ method: "DELETE", path: "/api/v0/workspace-invitations/i-eve" });
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted, eve]);
    nerve.calls[2]?.answer(noContent());
    await until(() => deleted.settled, "the deletion");
    expect(deleted.error).toBeUndefined();
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted]);
  });

  const refusals: { change: string; send: (store: WorkspaceMemberStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "invitations",
      send: (store) =>
        store.inviteMembersToWorkspace("acme", { invitations: [{ email: "dan@example.com", role: 15 }] }),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a role change",
      send: (store) => store.updateMemberInvitation("acme", "i-eve", { role: 15 }),
      refusal: problem(409, "workspace.invitation_responded"),
    },
    {
      change: "a deletion",
      send: (store) => store.deleteMemberInvitation("acme", "i-dan"),
      refusal: problem(404, "workspace.invitation_not_found"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const sent = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => sent.settled, "the refusal");
    expect(sent.error).toBeInstanceOf(ApiError);
    expect(store.workspaceMemberInvitations.acme).toEqual([dan, eve]);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const promoted = invitationOf("dan", { role: 20 });
    const invited = track(
      store.inviteMembersToWorkspace("acme", { invitations: [{ email: "fay@example.com", role: 15 }] })
    );
    const changed = track(store.updateMemberInvitation("acme", "i-dan", { role: 20 }));
    const deleted = track(store.deleteMemberInvitation("acme", "i-eve"));
    await inTurn(nerve, 1, ["POST", INVITATIONS], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", "/api/v0/workspace-invitations/i-dan"], json(200, promoted));
    await inTurn(nerve, 3, ["DELETE", "/api/v0/workspace-invitations/i-eve"], noContent());
    await until(() => deleted.settled, "the last change");
    expect(invited.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(promoted);
    expect(store.workspaceMemberInvitations.acme).toEqual([promoted]);
  });
});
````

`web/apps/web/core/store/member/workspace/workspace-member.store.ts`（修改，19 处）：

````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
import type { ApiClient, WorkspaceMember, WorkspaceMemberUpdate } from "@nerve/api-client";
import type { IWorkspaceBulkInviteFormData, IWorkspaceMemberInvitation } from "@nerve/types";
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
import type {
  ApiClient,
  WorkspaceInvitation,
  WorkspaceInvitationUpdate,
  WorkspaceInvitationsCreate,
  WorkspaceMember,
  WorkspaceMemberUpdate,
} from "@nerve/api-client";
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
import { WorkspaceInvitationsService } from "@/services/workspace/workspace-invitations.service";
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>>;
  workspaceMemberInvitations: Record<string, IWorkspaceMemberInvitation[]>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberMap: Record<string, Record<string, WorkspaceMember>>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  getWorkspaceInvitationDetails: (invitationId: string) => IWorkspaceMemberInvitation | null;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  getWorkspaceInvitationDetails: (invitationId: string) => WorkspaceInvitation | null;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMemberInvitations: (workspaceSlug: string) => Promise<IWorkspaceMemberInvitation[]>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMemberInvitations: (workspaceSlug: string) => Promise<WorkspaceInvitation[] | undefined>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  inviteMembersToWorkspace: (workspaceSlug: string, data: IWorkspaceBulkInviteFormData) => Promise<void>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  inviteMembersToWorkspace: (workspaceSlug: string, data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    data: Partial<IWorkspaceMemberInvitation>
  ) => Promise<void>;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    data: WorkspaceInvitationUpdate
  ) => Promise<WorkspaceInvitation>;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
 * The members of the workspaces of a session (M3 design 7.3): its service sends with the session's client, which
 * the RootStore of the session hands down. Changes go one at a time (v0 design 7.7); fetches do not queue.
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
 * The members and the invitations of the workspaces of a session (M3 design 7.3): its services send with the
 * session's client, which the RootStore of the session hands down. Changes go one at a time (v0 design 7.7);
 * fetches do not queue.
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberInvitations: Record<string, IWorkspaceMemberInvitation[]> = {}; // { workspaceSlug: [invitations] }
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  /** Each workspace's invitations, pending and declined, newest first: an admin's to fetch, whom nerve shows them. */
  workspaceMemberInvitations: Record<string, WorkspaceInvitation[]> = {};
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  // services
  workspaceService;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  // services
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  /** The changes of the memberships, sent one at a time. */
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  private readonly invitationsService: WorkspaceInvitationsService;
  /** The changes of the memberships and the invitations, sent one at a time. */
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    // services
    this.workspaceService = new WorkspaceService();
    this.service = new WorkspaceMembersService(api);
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    // services
    this.service = new WorkspaceMembersService(api);
    this.invitationsService = new WorkspaceInvitationsService(api);
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description fetch all the member invitations of a workspace
   * @param workspaceSlug
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description fetches a workspace's invitations and gives them: an admin's to fetch, as nerve refuses anyone
   * else. A change of session while they load is no failure: the new session's store fetches its own
   * (store-context.tsx), and this one gives undefined.
   * @returns {Promise<WorkspaceInvitation[] | undefined>}
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMemberInvitations = async (workspaceSlug: string) =>
    await this.workspaceService.workspaceInvitations(workspaceSlug).then((response) => {
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  fetchWorkspaceMemberInvitations = async (workspaceSlug: string): Promise<WorkspaceInvitation[] | undefined> => {
    try {
      const invitations = await this.invitationsService.list(workspaceSlug);
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
        set(this.workspaceMemberInvitations, workspaceSlug, response);
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
        this.workspaceMemberInvitations[workspaceSlug] = invitations;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
      return response;
    });

  /**
   * @description bulk invite members to a workspace
   * @param workspaceSlug
   * @param data
   */
  inviteMembersToWorkspace = async (workspaceSlug: string, data: IWorkspaceBulkInviteFormData) => {
    const response = await this.workspaceService.inviteWorkspace(workspaceSlug, data);
    await this.fetchWorkspaceMemberInvitations(workspaceSlug);
    return response;
  };

  /**
   * @description update the role of a member invitation
   * @param workspaceSlug
   * @param invitationId
   * @param data
   */
  updateMemberInvitation = async (
    workspaceSlug: string,
    invitationId: string,
    data: Partial<IWorkspaceMemberInvitation>
  ) => {
    const originalMemberInvitations = [...(this.workspaceMemberInvitations?.[workspaceSlug] ?? [])]; // in case of error, we will revert back to original members
    try {
      const memberInvitations = originalMemberInvitations?.map((invitation) => ({
        ...invitation,
        ...(invitation.id === invitationId && data),
      }));
      // optimistic update
      runInAction(() => {
        set(this.workspaceMemberInvitations, workspaceSlug, memberInvitations);
      });
      await this.workspaceService.updateWorkspaceInvitation(workspaceSlug, invitationId, data);
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
      return invitations;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
      // revert back to original members in case of error
      runInAction(() => {
        set(this.workspaceMemberInvitations, workspaceSlug, originalMemberInvitations);
      });
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
      if (error instanceof SessionChangedError) return undefined;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description delete a member invitation
   * @param workspaceSlug
   * @param memberId
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
   * @description invites the addresses data lists, all of them or none; the new invitations go first in the list
   * the store has, until the next fetch puts them in nerve's order. Fails, changing nothing, when nerve refuses.
   * @returns {Promise<WorkspaceInvitation[]>}
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  deleteMemberInvitation = async (workspaceSlug: string, invitationId: string) =>
    // oxlint-disable-next-line promise/always-return
    await this.workspaceService.deleteWorkspaceInvitations(workspaceSlug, invitationId).then(() => {
      runInAction(() => {
        this.workspaceMemberInvitations[workspaceSlug] = this.workspaceMemberInvitations[workspaceSlug].filter(
          (inv) => inv.id !== invitationId
        );
      });
    });

````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  inviteMembersToWorkspace = (
    workspaceSlug: string,
    data: WorkspaceInvitationsCreate
  ): Promise<WorkspaceInvitation[]> =>
    this.changes(async () => {
      const created = await this.invitationsService.create(workspaceSlug, data);
      this.changeInvitations(workspaceSlug, (invitations) => [...created, ...invitations]);
      return created;
    });

  /**
   * @description changes an invitation's role; the store then has nerve's answer. Fails, changing nothing, when
   * nerve refuses (a declined invitation, 409).
   * @returns {Promise<WorkspaceInvitation>}
   */
  updateMemberInvitation = (
    workspaceSlug: string,
    invitationId: string,
    data: WorkspaceInvitationUpdate
  ): Promise<WorkspaceInvitation> =>
    this.changes(async () => {
      const changed = await this.invitationsService.update(invitationId, data);
      this.changeInvitations(workspaceSlug, (invitations) =>
        invitations.map((i) => (i.id === changed.id ? changed : i))
      );
      return changed;
    });

  /**
   * @description deletes an invitation, which then leaves the list; fails, changing nothing, when nerve refuses
   * @returns {Promise<void>}
   */
  deleteMemberInvitation = (workspaceSlug: string, invitationId: string): Promise<void> =>
    this.changes(async () => {
      await this.invitationsService.delete(invitationId);
      this.changeInvitations(workspaceSlug, (invitations) => invitations.filter((i) => i.id !== invitationId));
    });

  /** The workspace's invitations as change makes them, when the store has fetched them: it makes up no list. */
  private changeInvitations(
    workspaceSlug: string,
    change: (invitations: WorkspaceInvitation[]) => WorkspaceInvitation[]
  ): void {
    runInAction(() => {
      const listed = this.workspaceMemberInvitations[workspaceSlug];
      if (listed) this.workspaceMemberInvitations[workspaceSlug] = change(listed);
    });
  }

````

`web/packages/types/src/workspace.ts`（修改，1 处）：

````old web/packages/types/src/workspace.ts
  };
}

export interface IWorkspaceBulkInviteFormData {
  emails: { email: string; role: TUserPermissions }[];
````
````new web/packages/types/src/workspace.ts
  };
````

- [ ] **Step 2: 成员页的取数按角色；工作区的页面按会话取成员**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IWorkspaceBulkInviteFormData } from "@nerve/types";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { useUserPermissions } from "@/hooks/store/user";
import type { InvitationFormValues } from "@/hooks/use-workspace-invitation";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const handleWorkspaceInvite = async (data: IWorkspaceBulkInviteFormData) => {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  const handleWorkspaceInvite = async (data: InvitationFormValues) => {
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
      await inviteMembersToWorkspace(workspaceSlug, data);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
      await inviteMembersToWorkspace(workspaceSlug, { invitations: data.emails });
````

`web/apps/web/core/components/profile/use-profile-member.ts`（修改，3 处）：

````old web/apps/web/core/components/profile/use-profile-member.ts

import useSWR from "swr";
// nerve imports
import { WORKSPACE_MEMBERS } from "@nerve/constants";
````
````new web/apps/web/core/components/profile/use-profile-member.ts

// nerve imports
````
````old web/apps/web/core/components/profile/use-profile-member.ts
import { useMember } from "@/hooks/store/use-member";
````
````new web/apps/web/core/components/profile/use-profile-member.ts
import { useMember } from "@/hooks/store/use-member";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
````
````old web/apps/web/core/components/profile/use-profile-member.ts
  const { data: members, isLoading } = useSWR(
    workspaceSlug ? WORKSPACE_MEMBERS(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceMembers(workspaceSlug) : null,
````
````new web/apps/web/core/components/profile/use-profile-member.ts
  const { data: members, isLoading } = useSessionSWR(
    workspaceSlug ? ["WORKSPACE_MEMBERS", workspaceSlug] : null,
    (slug) => fetchWorkspaceMembers(slug),
````

`web/apps/web/core/components/workspace/settings/members-list.tsx`（修改，6 处）：

````old web/apps/web/core/components/workspace/settings/members-list.tsx
import { useParams } from "react-router";
import useSWR from "swr";
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
import { useParams } from "react-router";
````
````old web/apps/web/core/components/workspace/settings/members-list.tsx
import { WorkspaceMembersListItem } from "./members-list-item";
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
import { WorkspaceMembersListItem } from "./members-list-item";
import { useMembersSettingsFetch } from "./use-members-settings-fetch";
````
````old web/apps/web/core/components/workspace/settings/members-list.tsx
    workspace: {
      fetchWorkspaceMembers,
      fetchWorkspaceMemberInvitations,
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
    workspace: {
````
````old web/apps/web/core/components/workspace/settings/members-list.tsx
  // fetching workspace invitations
  useSWR(
    workspaceSlug ? `WORKSPACE_MEMBERS_AND_MEMBER_INVITATIONS_${workspaceSlug}` : null,
    workspaceSlug
      ? async () => {
          await fetchWorkspaceMemberInvitations(workspaceSlug);
          await fetchWorkspaceMembers(workspaceSlug);
        }
      : null
  );
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
  useMembersSettingsFetch(workspaceSlug);
````
````old web/apps/web/core/components/workspace/settings/members-list.tsx
  const memberDetails = searchedMemberIds
    ?.map((memberId) => getWorkspaceMemberDetails(memberId))
    .sort((a, b) => {
      if (a?.is_active && !b?.is_active) return -1;
      if (!a?.is_active && b?.is_active) return 1;
      return 0;
    });
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
  const searchedMembers = searchedMemberIds?.map((memberId) => getWorkspaceMemberDetails(memberId)) ?? [];
  // the active members first, then those whose membership ended, each in the order the filters give
  const memberDetails = [
    ...searchedMembers.filter((member) => member?.is_active),
    ...searchedMembers.filter((member) => !member?.is_active),
  ];
````
````old web/apps/web/core/components/workspace/settings/members-list.tsx
        {searchedMemberIds?.length !== 0 && <WorkspaceMembersListItem memberDetails={memberDetails ?? []} />}
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
        {searchedMemberIds?.length !== 0 && <WorkspaceMembersListItem memberDetails={memberDetails} />}
````

`web/apps/web/core/components/workspace/settings/use-members-settings-fetch.test.ts`（新文件，59 行）：

````file web/apps/web/core/components/workspace/settings/use-members-settings-fetch.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { EUserWorkspaceRoles } from "@nerve/types";
import { handed } from "@/lib/fake-session-swr";

// The members settings fetch what the caller may read (M3 design 7.1, 9.5), with fake-session-swr.ts's stand-in for
// useSessionSWR and stand-ins for the stores: the caller's role in acme is the one his workspaces list gives.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
const state = vi.hoisted((): { calls: string[]; role: number | undefined } => ({ calls: [], role: undefined }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) =>
      slug === "acme" && state.role !== undefined ? { slug, role: state.role } : null,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      fetchWorkspaceMembers: (slug: string) => Promise.resolve(state.calls.push(`members of ${slug}`)),
      fetchWorkspaceMemberInvitations: (slug: string) => Promise.resolve(state.calls.push(`invitations of ${slug}`)),
    },
  }),
}));

const { useMembersSettingsFetch } = await import("./use-members-settings-fetch");

beforeEach(() => {
  handed.length = 0;
  state.calls = [];
  state.role = undefined;
});

describe("useMembersSettingsFetch", () => {
  it("fetches an admin the members and the invitations", async () => {
    state.role = EUserWorkspaceRoles.ADMIN;
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["WORKSPACE_MEMBERS", "acme"],
      ["WORKSPACE_INVITATIONS", "acme"],
    ]);
    await Promise.all(handed.map(([, fetcher]) => fetcher("acme")));
    expect(state.calls).toEqual(["members of acme", "invitations of acme"]);
  });

  it.each([
    { who: "a member", role: EUserWorkspaceRoles.MEMBER },
    { who: "a guest", role: EUserWorkspaceRoles.GUEST },
    { who: "a caller whose list does not have the workspace", role: undefined },
  ])("fetches $who the members alone", ({ role }) => {
    state.role = role;
    useMembersSettingsFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACE_MEMBERS", "acme"], null]);
  });
});
````

`web/apps/web/core/components/workspace/settings/use-members-settings-fetch.ts`（新文件，28 行）：

````file web/apps/web/core/components/workspace/settings/use-members-settings-fetch.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { EUserWorkspaceRoles } from "@nerve/types";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * The fetches of a workspace's members settings (M3 design 7.1: a page fetches what its caller may read): the
 * members, for anyone the page is shown to; the invitations for an admin alone, as nerve shows them to no one else.
 * The caller's role is the one his workspaces list gives (Workspace.role).
 */
export function useMembersSettingsFetch(workspaceSlug: string | undefined): void {
  const { getWorkspaceBySlug } = useWorkspace();
  const {
    workspace: { fetchWorkspaceMembers, fetchWorkspaceMemberInvitations },
  } = useMember();
  const isAdmin = workspaceSlug !== undefined && getWorkspaceBySlug(workspaceSlug)?.role === EUserWorkspaceRoles.ADMIN;
  useSessionSWR(["WORKSPACE_MEMBERS", workspaceSlug], (slug) => fetchWorkspaceMembers(slug));
  useSessionSWR(isAdmin ? ["WORKSPACE_INVITATIONS", workspaceSlug] : null, (slug) =>
    fetchWorkspaceMemberInvitations(slug)
  );
}
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`（修改，3 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
// useSessionSWR and a stand-in for the store.
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
// useSessionSWR and stand-ins for the stores: the caller's list has acme alone.
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    fetchWorkspaces: () => Promise.resolve(fetched.calls.push("the workspaces")),
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    fetchWorkspaces: () => Promise.resolve(fetched.calls.push("the workspaces")),
    getWorkspaceBySlug: (slug: string) => (slug === "acme" ? { slug } : null),
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      fetchWorkspaceMembers: (slug: string) => Promise.resolve(fetched.calls.push(`the members of ${slug}`)),
    },
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces", async () => {
    useWorkspaceFetch();
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher()));
    expect(fetched.calls).toEqual(["the workspaces"]);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members", async () => {
    useWorkspaceFetch("acme");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], ["WORKSPACE_MEMBERS", "acme"]]);
    await Promise.all(handed.map(([, fetcher]) => fetcher("acme")));
    expect(fetched.calls).toEqual(["the workspaces", "the members of acme"]);
  });

  it("fetches only the caller's workspaces where the address names none of his", () => {
    useWorkspaceFetch("elsewhere");
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null]);
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`（整个文件，32 行）：

````whole web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { SWRResponse } from "swr";
import type { Workspace } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/** Fetched once a session: a page that mounts again shows what the stores have. */
const ONCE = { revalidateIfStale: false, revalidateOnFocus: false };

/**
 * The workspace side of what a page of a workspace fetches as it mounts (M3 design 3.1, 7.1): the caller's
 * workspaces, which decide whether he may see the address's one; once his list has it, its members. Gives the
 * list's response, whose failure the page shows.
 */
export function useWorkspaceFetch(workspaceSlug: string | undefined): SWRResponse<Workspace[] | undefined> {
  const { fetchWorkspaces, getWorkspaceBySlug } = useWorkspace();
  const {
    workspace: { fetchWorkspaceMembers },
  } = useMember();
  // the address's workspace is the caller's once his list has it
  const isMember = workspaceSlug !== undefined && getWorkspaceBySlug(workspaceSlug) !== null;
  const listed = useSessionSWR(["WORKSPACES"], () => fetchWorkspaces(), { ...ONCE, shouldRetryOnError: false });
  useSessionSWR(isMember ? ["WORKSPACE_MEMBERS", workspaceSlug] : null, (slug) => fetchWorkspaceMembers(slug), ONCE);
  return listed;
}
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，5 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import {
  WORKSPACE_MEMBERS,
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import {
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { useFavorite } from "@/hooks/store/use-favorite";
import { useMember } from "@/hooks/store/use-member";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { useFavorite } from "@/hooks/store/use-favorite";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchFavorite } = useFavorite();
  const {
    workspace: { fetchWorkspaceMembers },
  } = useMember();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchFavorite } = useFavorite();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const listed = useWorkspaceFetch();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const listed = useWorkspaceFetch(workspaceSlug);
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
    workspaceSlug && currentWorkspace ? () => fetchPartialProjects(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  // fetch workspace members
  useSWR(
    workspaceSlug && currentWorkspace ? WORKSPACE_MEMBERS(workspaceSlug) : null,
    workspaceSlug && currentWorkspace ? () => fetchWorkspaceMembers(workspaceSlug) : null,
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
    workspaceSlug && currentWorkspace ? () => fetchPartialProjects(workspaceSlug) : null,
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts
  `WORKSPACE_PARTIAL_PROJECTS_${workspaceSlug.toUpperCase()}`;

export const WORKSPACE_MEMBERS = (workspaceSlug: string) => `WORKSPACE_MEMBERS_${workspaceSlug.toUpperCase()}`;
````
````new web/packages/constants/src/fetch-keys.ts
  `WORKSPACE_PARTIAL_PROJECTS_${workspaceSlug.toUpperCase()}`;
````

- [ ] **Step 3: 邀请的使用方**

`web/apps/web/core/components/onboarding/steps/team/root.tsx`（修改，4 处）：

````old web/apps/web/core/components/onboarding/steps/team/root.tsx
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// services
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/components/onboarding/steps/team/root.tsx
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````old web/apps/web/core/components/onboarding/steps/team/root.tsx

// services
const workspaceService = new WorkspaceService();
````
````new web/apps/web/core/components/onboarding/steps/team/root.tsx

````
````old web/apps/web/core/components/onboarding/steps/team/root.tsx
  const { workspaces } = useWorkspace();
````
````new web/apps/web/core/components/onboarding/steps/team/root.tsx
  const { workspaces } = useWorkspace();
  const {
    workspace: { inviteMembersToWorkspace },
  } = useMember();
````
````old web/apps/web/core/components/onboarding/steps/team/root.tsx
    await workspaceService
      .inviteWorkspace(workspace.slug, {
        emails: payload.emails.map((email) => ({
          email: email.email,
          role: email.role,
        })),
      })
````
````new web/apps/web/core/components/onboarding/steps/team/root.tsx
    await inviteMembersToWorkspace(workspace.slug, {
      invitations: payload.emails.map((email) => ({
        email: email.email,
        role: email.role,
      })),
    })
````

`web/apps/web/core/components/workspace/members/invite-modal.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/members/invite-modal.tsx
import { useTranslation } from "@nerve/i18n";
import type { IWorkspaceBulkInviteFormData } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/workspace/members/invite-modal.tsx
// hooks
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
// hooks
import type { InvitationFormValues } from "@/hooks/use-workspace-invitation";
````
````old web/apps/web/core/components/workspace/members/invite-modal.tsx
  onSubmit: (data: IWorkspaceBulkInviteFormData) => Promise<void> | undefined;
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
  onSubmit: (data: InvitationFormValues) => Promise<void> | undefined;
````

`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
      const inviteLink = new URL(invitationDetails.invite_link, window.location.origin).href;
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
      // the invitation's link (M3 design 7.4): its id and the token nerve gives an admin with it
      const path = `/workspace-invitations?invitation_id=${invitationDetails.id}&token=${invitationDetails.token}`;
      const inviteLink = new URL(path, window.location.origin).href;
````
````old web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
      icon: LinkOutline,
      shouldRender: !!invitationDetails.invite_link,
````
````new web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
      icon: LinkOutline,
````

- [ ] **Step 4: 取数和非空断言的静态检查**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
    ]
  }
````
````new .oxlintrc.json
    ]
  },
  "overrides": [
    {
      "files": [
        "web/apps/web/core/store/workspace/**",
        "web/apps/web/core/store/member/workspace/**",
        "web/apps/web/core/components/workspace/settings/**",
        "web/apps/web/core/components/profile/use-profile-member.ts",
        "web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts",
        "web/apps/web/core/lib/use-landing.ts",
        "web/apps/web/core/lib/wrappers/authentication-wrapper.tsx"
      ],
      "rules": {
        "no-restricted-imports": [
          "error",
          {
            "paths": [
              {
                "name": "swr",
                "allowTypeImports": true,
                "message": "A fetch of the session goes through useSessionSWR, whose key carries the loginId (v0 design 7.7)."
              }
            ]
          }
        ]
      }
    },
    {
      "files": [
        "web/apps/web/core/store/workspace/**",
        "web/apps/web/core/store/member/workspace/**",
        "web/apps/web/core/store/user/permissions.store.ts",
        "web/apps/web/core/services/workspace/**",
        "web/apps/web/core/components/workspace/settings/**",
        "web/apps/web/core/components/profile/use-profile-member.ts",
        "web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts",
        "web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts",
        "web/apps/web/core/lib/error-messages.test.ts",
        "web/apps/web/core/lib/error-messages.ts",
        "web/apps/web/core/lib/fake-session-swr.ts",
        "web/apps/web/core/lib/in-session.test.ts",
        "web/apps/web/core/lib/in-session.ts",
        "web/apps/web/core/lib/landing.test.ts",
        "web/apps/web/core/lib/landing.ts",
        "web/apps/web/core/lib/session-key.test.ts",
        "web/apps/web/core/lib/session-key.ts",
        "web/apps/web/core/lib/use-landing.test.ts",
        "web/apps/web/core/lib/use-landing.ts",
        "web/apps/web/core/lib/use-session-swr.test.ts",
        "web/apps/web/core/lib/use-session-swr.ts"
      ],
      "rules": {
        "typescript/no-non-null-assertion": "error"
      }
    }
  ]
````

- [ ] **Step 5: 上限、关键词规则**

`tools/keywords.json`（修改，3 处）：

````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/(?:`|\\$\\{[^}]+\\}/`)",
````
````old tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/members/${memberId}/`, data)"
````
````new tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/members/${memberId}/`, data)",
          "    return this.post(`/api/workspaces/${workspaceSlug}/invitations/`, data)",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data)"
````
````old tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`, data)"
````
````new tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`, data)",
          "    return unwrap(await this.api.GET(\"/api/v0/workspaces/{slug}/invitations\", { params: { path: { slug } } })).data;",
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/invitations/`, data)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })"
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 445",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 444",
````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则，没有命中；web 的 oxlint 444 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过（A9 的邀请链接、W4 的邀请接口版本照旧）。

- [ ] **Step 7: 提交**

```bash
git add .oxlintrc.json tools/keywords.json 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx' web/apps/web/core/components/onboarding/steps/team/root.tsx web/apps/web/core/components/profile/use-profile-member.ts web/apps/web/core/components/workspace/members/invite-modal.tsx web/apps/web/core/components/workspace/settings/invitations-list-item.tsx web/apps/web/core/components/workspace/settings/members-list.tsx web/apps/web/core/components/workspace/settings/use-members-settings-fetch.test.ts web/apps/web/core/components/workspace/settings/use-members-settings-fetch.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/workspace.service.ts web/apps/web/core/services/workspace/workspace-invitations.service.ts web/apps/web/core/store/member/workspace/workspace-invitations.test.ts web/apps/web/core/store/member/workspace/workspace-member.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/types/src/workspace.ts
```
```bash
git commit -m "feat(M3/P8a): invitations come from /api/v0; the members page fetches them for admins alone

WorkspaceInvitationsService wraps the generated client, and the member
store keeps a workspace's invitations as nerve lists them to an admin;
creating, changing and deleting them go one at a time with the
members' changes and take nerve's answer. The members page fetches the
members for anyone it shows them to and the invitations for an admin
alone. A workspace's pages fetch its members by the session, and only
once the caller's list says the workspace is his. Both hooks work out
these conditions from the caller's list themselves. The root oxlint
config keeps P8a's session fetches off SWR itself, type imports aside,
and P8a's M3 paths free of non-null assertions.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t8-invite-unqueued` | 邀请不等待前一个修改就发出 | `workspace-invitations.test.ts` | vitest |
| `t8-invite-made-up` | 没有取过列表时，新的邀请自成一个列表 | `workspace-invitations.test.ts` | vitest |
| `t8-invite-last` | 新的邀请排在最后，不是像 nerve 那样最新的在前 | `workspace-invitations.test.ts` | vitest |
| `t8-invitation-early` | 邀请的角色在 nerve 应答之前就改了 | `workspace-invitations.test.ts` | vitest |
| `t8-delete-kept` | 删除的邀请留在列表里 | `workspace-invitations.test.ts` | vitest |
| `t8-invitations-session` | 邀请加载时换了会话，取数失败 | `workspace-invitations.test.ts` | vitest |
| `t8-invitations-for-all` | 成员页对成员、访客也取邀请 | oxlint（`check:lint`）、`use-members-settings-fetch.test.ts` | 静态；vitest |
| `t8-members-for-admins` | 成员页只对管理员取成员 | `use-members-settings-fetch.test.ts` | vitest |
| `t8-settings-fetcher-drops` | 成员页的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |
| `t8-invitations-back` | 旧 service 又列出工作区的邀请 | 关键词守卫 | 静态 |
| `t8-profile-drops` | 个人主页取成员的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |
| `t8-wrapper-members-ungated` | 工作区的页面在列表说明它是调用者的工作区之前就取成员 | `use-workspace-fetch.test.ts` | vitest |
| `t8-wrapper-members-drops` | 工作区的页面取成员的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |
| `t8-settings-any-role` | 成员页对任何角色都取邀请（`role` 有值即可） | oxlint（`check:lint`）、`use-members-settings-fetch.test.ts` | 静态；vitest |
| `t8-settings-member-up` | 成员页对成员也取邀请（成员及以上） | `use-members-settings-fetch.test.ts` | vitest |
| `t8-wrapper-any-slug` | 工作区的页面对任何地址都取成员和显示设置，不看列表里有没有这个工作区 | oxlint（`check:lint`）、`use-workspace-fetch.test.ts` | 静态；vitest |
| `t8-raw-swr-hook` | `useWorkspaceFetch` 绕过 `useSessionSWR`，直接用 SWR 本身 | oxlint（`check:lint`）、`use-workspace-fetch.test.ts` | 静态；vitest |
| `t8-raw-swr-profile` | 个人主页取成员时绕过 `useSessionSWR`，直接用 SWR 本身 | oxlint（`check:lint`） | 静态 |
| `t8-nonnull-email` | 成员列的图片 `alt` 用 `email!` 断言非空 | oxlint（`check:lint`） | 静态 |
| `t8-invitations-swallows` | 邀请的列表被拒绝时给出 `undefined`，不失败 | `workspace-invitations.test.ts` | vitest |
| `t8-invitations-pending` | store 只留待接受的邀请，丢掉 nerve 列出的已忽略的 | `workspace-invitations.test.ts` | vitest |
| `t8-invite-unread` | service 不读 nerve 的 problem：被拒绝的邀请当作没有发出 | `workspace-invitations.test.ts` | vitest |
| `t8-invitation-delete-unread` | service 不读 nerve 的 problem：被拒绝的删除邀请当作成功 | `workspace-invitations.test.ts` | vitest |
| `t8-invitations-not-admins` | 成员页为成员和访客取邀请，不为管理员取 | `use-members-settings-fetch.test.ts` | vitest |

---

### Task 9: 邀请链接：公开的查看、接受与忽略

**Files:**
- Create: `web/apps/web/core/hooks/use-invitation-preview.ts`、`web/apps/web/core/services/workspace/invitation-preview.service.test.ts`、`web/apps/web/core/services/workspace/invitation-preview.service.ts`
- Modify: `tools/keywords.json`、`web/apps/web/app/(all)/workspace-invitations/page.tsx`、`web/apps/web/core/components/account/auth-forms/auth-header.tsx`、`web/apps/web/core/components/account/auth-forms/auth-root.tsx`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/services/workspace/workspaces.service.ts`、`web/apps/web/core/store/workspace/index.test.ts`、`web/apps/web/core/store/workspace/index.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/types/src/workspace.ts`

**Interfaces:**
- Produces（spec 2.9；M3 设计 3.8、5.1、7.1、7.4；决策点 1、2）：
  - `core/services/workspace/invitation-preview.service.ts`：`previewInvitation(client: ApiClient, invitationId, token): Promise<InvitationPreview>`（`GET /api/v0/workspace-invitations/{id}?token=`）。它是公开的操作，调用方传 `publicClient`，所以是一个函数，不是按会话建的 service（7.1："只调公开操作的可以用 `publicClient`"）。
  - `core/hooks/use-invitation-preview.ts`：`useInvitationPreview(invitationId: string | null, token: string | null)`：`useSWR(["INVITATION_PREVIEW", id, token] | null, …)`。它是 P8a 写的唯一不带 `loginId` 的键，也是"会话的取数只经 `useSessionSWR`"的唯一例外：它是公开操作，回答对任何会话、没有会话都一样，只由链接决定（spec 第 3 节）。所以它不在 Task 8 的 `no-restricted-imports` 的范围内。
  - `WorkspacesService.accept(invitationId, token): Promise<Workspace>`、`decline(invitationId, token): Promise<void>`；`IWorkspaceRootStore.acceptInvitation(id, token)`：经 `changes`，回答的工作区加入已取的列表，调用者本来就是成员时换掉列表中它的那一项（角色不变，3.8）；`declineInvitation(id, token)`：经 `changes`，列表不变。被拒绝（发给另一个邮箱的 403、已忽略、已不存在）时什么都不改。
  - 邀请页 `/workspace-invitations?invitation_id=…&token=…`（整个文件）：用 `useInvitationPreview` 查看，接受、忽略经 `WorkspaceRootStore`；Plane 按 `slug` 和 `email` 的查看、"已接受"一支和它的 `joinWorkspace` 删除（P9 照 7.4 重写这一页的行为）。登录、注册页的标题（`auth-header.tsx`、`auth-root.tsx`）读 `invitation_id` 和 `token`，用同一个 hook 取工作区名。
  - 删除：旧 `WorkspaceService` 的 `joinWorkspace`、`getWorkspaceInvitation`、取数键 `WORKSPACE_INVITATION`、`types/src/workspace.ts` 中最后的邀请类型。之后旧 `WorkspaceService` 中 M3 的方法只剩显示设置的两个（Task 10），模块级的实例在工作区一侧都已消失（M2 交接第 3 节的 `git grep`）。
- Adds：`plane-workspace-urls` 加邀请链接的旧地址；不命中样例加新的查看、接受地址。

**Tests:**（vitest）
- `invitation-preview.service.test.ts`：`previewInvitation` / `asks nerve, without a token of a session, what the link shows, and fails for a link that names none`（请求没有 `Authorization`，路径和 `token` 照链接；404 时失败）。
- `index.test.ts` 加 `adds an accepted invitation's workspace to the list it has, in its own place when listed, and to none it has not fetched`；`declines an invitation, the caller's workspaces staying as they are`；拒绝的 `it.each` 加接受、忽略两行；`sends each change once nerve has answered the one before it, refused or not` 加上接受和忽略（第 5、6 个修改），核对接受的回答和忽略的成功。

- [ ] **Step 1: 公开的查看**

`web/apps/web/core/hooks/use-invitation-preview.ts`（新文件，23 行）：

````file web/apps/web/core/hooks/use-invitation-preview.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import useSWR from "swr";
// lib
import { publicClient } from "@/lib/auth/api-client";
// services
import { previewInvitation } from "@/services/workspace/invitation-preview.service";

/**
 * The invitation a link names, as the link shows it, for the pages a link opens (sign-in, sign-up, the invitation's
 * own). It is public and the same for every session and for none, so it is the one M3 fetch keyed by the link
 * alone, not by a session (M3 design 7.1, 7.4); a link that names no invitation is SWR's error.
 */
export function useInvitationPreview(invitationId: string | null, token: string | null) {
  return useSWR(
    invitationId && token ? ["INVITATION_PREVIEW", invitationId, token] : null,
    ([, id, linkToken]: [string, string, string]) => previewInvitation(publicClient, id, linkToken),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );
}
````

`web/apps/web/core/services/workspace/invitation-preview.service.test.ts`（新文件，51 行）：

````file web/apps/web/core/services/workspace/invitation-preview.service.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { InvitationPreview } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { previewInvitation } from "./invitation-preview.service";

// What an invitation's link shows whoever holds it (M3 design 7.4), against a fake nerve.

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("previewInvitation", () => {
  it("asks nerve, without a token of a session, what the link shows, and fails for a link that names none", async () => {
    const nerve = new FakeNerve();
    const preview: InvitationPreview = {
      id: "i-dan",
      role: 15,
      declined: false,
      workspace_name: "Acme",
      workspace_slug: "acme",
      workspace_logo_url: null,
    };
    const shown = track(previewInvitation(nerve.client(), "i-dan", "nrv_inv_dan"));
    await until(() => nerve.calls.length === 1, "the preview");
    expect(nerve.calls[0]).toMatchObject({
      method: "GET",
      path: "/api/v0/workspace-invitations/i-dan",
      query: { token: "nrv_inv_dan" },
      authorization: null,
    });
    nerve.calls[0]?.answer(json(200, preview));
    await until(() => shown.settled, "the preview");
    expect(shown.value).toEqual(preview);

    const unknown = track(previewInvitation(nerve.client(), "i-zed", "nrv_inv_zed"));
    await until(() => nerve.calls.length === 2, "the preview");
    nerve.calls[1]?.answer(problem(404, "workspace.invitation_not_found"));
    await until(() => unknown.settled, "the failure");
    expect(unknown.error).toBeInstanceOf(ApiError);
  });
});
````

`web/apps/web/core/services/workspace/invitation-preview.service.ts`（新文件，23 行）：

````file web/apps/web/core/services/workspace/invitation-preview.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, InvitationPreview } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/**
 * An invitation as its link shows it to whoever holds the link (M3 design 5.1, 7.4): public, the same with a session
 * or without one, so the web app asks with publicClient.
 */
export async function previewInvitation(
  client: ApiClient,
  invitationId: string,
  token: string
): Promise<InvitationPreview> {
  return unwrap(
    await client.GET("/api/v0/workspace-invitations/{invitation_id}", {
      params: { path: { invitation_id: invitationId }, query: { token } },
    })
  );
}
````

- [ ] **Step 2: 接受与忽略**

`web/apps/web/core/services/workspace/workspaces.service.ts`（修改，1 处）：

````old web/apps/web/core/services/workspace/workspaces.service.ts
    return unwrap(await this.api.GET("/api/v0/workspace-slugs/{slug}", { params: { path: { slug } } }));
  }
````
````new web/apps/web/core/services/workspace/workspaces.service.ts
    return unwrap(await this.api.GET("/api/v0/workspace-slugs/{slug}", { params: { path: { slug } } }));
  }

  /** Accepts an invitation sent to the caller's address; the answer is its workspace, with the caller's role. */
  async accept(invitationId: string, token: string): Promise<Workspace> {
    return unwrap(
      await this.api.POST("/api/v0/workspace-invitations/{invitation_id}/accept", {
        params: { path: { invitation_id: invitationId } },
        body: { token },
      })
    );
  }

  /** Declines an invitation sent to the caller's address: its link shows so from then on. */
  async decline(invitationId: string, token: string): Promise<void> {
    unwrap(
      await this.api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
        params: { path: { invitation_id: invitationId } },
        body: { token },
      })
    );
  }
````

`web/apps/web/core/store/workspace/index.test.ts`（修改，6 处）：

````old web/apps/web/core/store/workspace/index.test.ts
  });

  const refusals: { change: string; send: (store: WorkspaceRootStore) => Promise<unknown>; refusal: Response }[] = [
````
````new web/apps/web/core/store/workspace/index.test.ts
  });

  it("adds an accepted invitation's workspace to the list it has, in its own place when listed, and to none it has not fetched", async () => {
    const { nerve, store } = await loaded();
    const gamma = workspaceOf("gamma");
    const accepted = track(store.acceptInvitation("i-gamma", "nrv_inv_gamma"));
    await until(() => nerve.calls.length === 2, "the acceptance");
    expect(nerve.calls[1]).toMatchObject({
      method: "POST",
      path: "/api/v0/workspace-invitations/i-gamma/accept",
      body: { token: "nrv_inv_gamma" },
    });
    expect(store.workspaces).toEqual([acme, beta]);
    nerve.calls[1]?.answer(json(200, gamma));
    await until(() => accepted.settled, "the workspace");
    expect(accepted.value).toEqual(gamma);
    expect(store.workspaces).toEqual([acme, beta, gamma]);

    // a member accepts too, his role as it was (M3 design 3.8)
    const renamed = { ...beta, name: "Beta Inc" };
    const again = track(store.acceptInvitation("i-beta", "nrv_inv_beta"));
    await until(() => nerve.calls.length === 3, "the acceptance");
    nerve.calls[2]?.answer(json(200, renamed));
    await until(() => again.settled, "the workspace");
    expect(store.workspaces).toEqual([acme, renamed, gamma]);

    const fresh = setUp();
    const alone = track(fresh.store.acceptInvitation("i-gamma", "nrv_inv_gamma"));
    await until(() => fresh.nerve.calls.length === 1, "the acceptance");
    fresh.nerve.calls[0]?.answer(json(200, gamma));
    await until(() => alone.settled, "the workspace");
    expect(fresh.store.workspaces).toBeUndefined();
  });

  it("declines an invitation, the caller's workspaces staying as they are", async () => {
    const { nerve, store } = await loaded();
    const declined = track(store.declineInvitation("i-gamma", "nrv_inv_gamma"));
    await until(() => nerve.calls.length === 2, "the decline");
    expect(nerve.calls[1]).toMatchObject({
      method: "POST",
      path: "/api/v0/workspace-invitations/i-gamma/decline",
      body: { token: "nrv_inv_gamma" },
    });
    nerve.calls[1]?.answer(noContent());
    await until(() => declined.settled, "the decline");
    expect(declined.error).toBeUndefined();
    expect(store.workspaces).toEqual([acme, beta]);
  });

  const refusals: { change: string; send: (store: WorkspaceRootStore) => Promise<unknown>; refusal: Response }[] = [
````
````old web/apps/web/core/store/workspace/index.test.ts
      refusal: problem(409, "workspace.sole_admin"),
    },
````
````new web/apps/web/core/store/workspace/index.test.ts
      refusal: problem(409, "workspace.sole_admin"),
    },
    {
      change: "an acceptance",
      send: (store) => store.acceptInvitation("i-gamma", "nrv_inv_gamma"),
      refusal: problem(403, "workspace.invitation_email_mismatch"),
    },
    {
      change: "a decline",
      send: (store) => store.declineInvitation("i-gamma", "nrv_inv_gamma"),
      refusal: problem(409, "workspace.invitation_responded"),
    },
````
````old web/apps/web/core/store/workspace/index.test.ts
    const gamma = workspaceOf("gamma", { role: 20 });
````
````new web/apps/web/core/store/workspace/index.test.ts
    const gamma = workspaceOf("gamma", { role: 20 });
    const delta = workspaceOf("delta");
````
````old web/apps/web/core/store/workspace/index.test.ts
    const left = track(store.leaveWorkspace("acme"));
````
````new web/apps/web/core/store/workspace/index.test.ts
    const left = track(store.leaveWorkspace("acme"));
    const accepted = track(store.acceptInvitation("i-delta", "nrv_inv_delta"));
    const declined = track(store.declineInvitation("i-gamma", "nrv_inv_gamma"));
````
````old web/apps/web/core/store/workspace/index.test.ts
    await until(() => left.settled, "the last change");
````
````new web/apps/web/core/store/workspace/index.test.ts
    await inTurn(nerve, 5, ["POST", "/api/v0/workspace-invitations/i-delta/accept"], json(200, delta));
    await inTurn(nerve, 6, ["POST", "/api/v0/workspace-invitations/i-gamma/decline"], noContent());
    await until(() => declined.settled, "the last change");
````
````old web/apps/web/core/store/workspace/index.test.ts
    expect(store.workspaces).toEqual([gamma]);
````
````new web/apps/web/core/store/workspace/index.test.ts
    expect(left.error).toBeUndefined();
    expect(accepted.value).toEqual(delta);
    expect(declined.error).toBeUndefined();
    expect(store.workspaces).toEqual([gamma, delta]);
````

`web/apps/web/core/store/workspace/index.ts`（修改，3 处）：

````old web/apps/web/core/store/workspace/index.ts
  leaveWorkspace: (workspaceSlug: string) => Promise<void>;
````
````new web/apps/web/core/store/workspace/index.ts
  leaveWorkspace: (workspaceSlug: string) => Promise<void>;
  acceptInvitation: (invitationId: string, token: string) => Promise<Workspace>;
  declineInvitation: (invitationId: string, token: string) => Promise<void>;
````
````old web/apps/web/core/store/workspace/index.ts
      leaveWorkspace: action,
````
````new web/apps/web/core/store/workspace/index.ts
      leaveWorkspace: action,
      acceptInvitation: action,
      declineInvitation: action,
````
````old web/apps/web/core/store/workspace/index.ts
    });

  /** The workspace leaves the caller's list: it was deleted, or he is no longer a member. */
````
````new web/apps/web/core/store/workspace/index.ts
    });

  /**
   * @description accepts an invitation sent to the caller's address: its workspace joins the list, or takes its own
   * place there when the caller was a member already (his role stays). Fails, changing nothing, when nerve refuses
   * (another address's invitation, 403; one declined, or no longer there).
   * @returns {Promise<Workspace>}
   */
  acceptInvitation = (invitationId: string, token: string): Promise<Workspace> =>
    this.changes(async () => {
      const workspace = await this.service.accept(invitationId, token);
      runInAction(() => {
        const listed = this.workspaces?.some((w) => w.id === workspace.id);
        if (listed) this.workspaces = this.workspaces?.map((w) => (w.id === workspace.id ? workspace : w));
        else if (this.workspaces) this.workspaces = [...this.workspaces, workspace];
      });
      return workspace;
    });

  /**
   * @description declines an invitation sent to the caller's address; his workspaces stay as they are. Fails when
   * nerve refuses.
   * @returns {Promise<void>}
   */
  declineInvitation = (invitationId: string, token: string): Promise<void> =>
    this.changes(() => this.service.decline(invitationId, token));

  /** The workspace leaves the caller's list: it was deleted, or he is no longer a member. */
````

- [ ] **Step 3: 邀请页、登录和注册页**

`web/apps/web/app/(all)/workspace-invitations/page.tsx`（整个文件，95 行）：

````whole web/apps/web/app/(all)/workspace-invitations/page.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate, useSearchParams } from "react-router";
import { BoxesOutline, CloseOutline, TickOutline, UserOutline } from "@makeplane/propel/icons";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { EmptySpace, EmptySpaceItem } from "@/components/ui/empty-space";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser } from "@/hooks/store/user";
import { useInvitationPreview } from "@/hooks/use-invitation-preview";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

function WorkspaceInvitationPage() {
  // router
  const navigate = useNavigate();
  // query params
  const [searchParams] = useSearchParams();
  const invitation_id = searchParams.get("invitation_id");
  const token = searchParams.get("token");
  // store hooks
  const { data: currentUser } = useUser();
  const { acceptInvitation, declineInvitation } = useWorkspace();

  const { data: invitationDetail, error } = useInvitationPreview(invitation_id, token);

  const handleAccept = async () => {
    if (!invitationDetail || !token) return;
    try {
      // nerve accepts the invitation of the caller's own address alone
      await acceptInvitation(invitationDetail.id, token);
      navigate(`/${invitationDetail.workspace_slug}`);
    } catch (err: unknown) {
      console.error(err);
    }
  };

  const handleReject = async () => {
    if (!invitationDetail || !token) return;
    try {
      await declineInvitation(invitationDetail.id, token);
      navigate("/");
    } catch (err: unknown) {
      console.error(err);
    }
  };

  return (
    <AuthenticationWrapper pageType={EPageTypes.PUBLIC}>
      <div className="flex h-full w-full flex-col items-center justify-center px-3">
        {invitationDetail && !invitationDetail.declined ? (
          error ? (
            <div className="shadow-2xl flex w-full flex-col space-y-4 rounded-sm border border-subtle bg-surface-1 px-4 py-8 text-center md:w-1/3">
              <h2 className="text-18 uppercase">INVITATION NOT FOUND</h2>
            </div>
          ) : (
            <EmptySpace
              title={`You have been invited to ${invitationDetail.workspace_name}`}
              description="Your workspace is where you'll create projects, collaborate on your work items, and organize different streams of work in your Nerve account."
            >
              <EmptySpaceItem Icon={TickOutline} title="Accept" action={handleAccept} />
              <EmptySpaceItem Icon={CloseOutline} title="Ignore" action={handleReject} />
            </EmptySpace>
          )
        ) : error || invitationDetail?.declined ? (
          <EmptySpace
            title="This invitation link is not active anymore."
            description="Your workspace is where you'll create projects, collaborate on your work items, and organize different streams of work in your Nerve account."
            link={{ text: "Or start from an empty project", href: "/" }}
          >
            {!currentUser ? (
              <EmptySpaceItem Icon={UserOutline} title="Sign in to continue" href="/" />
            ) : (
              <EmptySpaceItem Icon={BoxesOutline} title="Continue to home" href="/" />
            )}
          </EmptySpace>
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <LogoSpinner />
          </div>
        )}
      </div>
    </AuthenticationWrapper>
  );
}

export default observer(WorkspaceInvitationPage);
````

`web/apps/web/core/components/account/auth-forms/auth-header.tsx`（修改，9 处）：

````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
import useSWR from "swr";
import { useTranslation } from "@nerve/i18n";
import type { IWorkspaceMemberInvitation } from "@nerve/types";
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
import type { InvitationPreview } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
// services
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
// hooks
import { useInvitationPreview } from "@/hooks/use-invitation-preview";
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
  workspaceSlug: string | undefined;
  invitationId: string | undefined;
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
  invitationId: string | null;
  token: string | null;
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx

const workSpaceService = new WorkspaceService();

````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx

````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const { workspaceSlug, invitationId, authMode } = props;
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const { invitationId, token, authMode } = props;
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const { data: invitation, isLoading } = useSWR(
    workspaceSlug && invitationId ? `WORKSPACE_INVITATION_${workspaceSlug}_${invitationId}` : null,
    async () => workspaceSlug && invitationId && workSpaceService.getWorkspaceInvitation(workspaceSlug, invitationId),
    {
      revalidateOnFocus: false,
      shouldRetryOnError: false,
    }
  );
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const { data: invitation, isLoading } = useInvitationPreview(invitationId, token);
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const getHeaderSubHeader = (mode: EAuthModes, current: IWorkspaceMemberInvitation | undefined) => {
    if (current?.workspace) {
      const workspace = current.workspace;
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const getHeaderSubHeader = (mode: EAuthModes, current: InvitationPreview | undefined) => {
    if (current) {
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
            <WorkspaceLogo logo={workspace?.logo_url} name={workspace?.name} classNames="size-9 flex-shrink-0" />{" "}
            {workspace.name}
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
            <WorkspaceLogo
              logo={current.workspace_logo_url}
              name={current.workspace_name}
              classNames="size-9 flex-shrink-0"
            />{" "}
            {current.workspace_name}
````
````old web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const { header, subHeader } = getHeaderSubHeader(authMode, invitation || undefined);
````
````new web/apps/web/core/components/account/auth-forms/auth-header.tsx
  const { header, subHeader } = getHeaderSubHeader(authMode, invitation);
````

`web/apps/web/core/components/account/auth-forms/auth-root.tsx`（修改，3 处）：

````old web/apps/web/core/components/account/auth-forms/auth-root.tsx
  // query params: a workspace invitation's, for M3's "join the workspace" title (M2 design 7.3)
````
````new web/apps/web/core/components/account/auth-forms/auth-root.tsx
  // query params: a workspace invitation's link, for M3's "join the workspace" title (M2 design 7.3)
````
````old web/apps/web/core/components/account/auth-forms/auth-root.tsx
  const workspaceSlug = searchParams.get("slug");
````
````new web/apps/web/core/components/account/auth-forms/auth-root.tsx
  const token = searchParams.get("token");
````
````old web/apps/web/core/components/account/auth-forms/auth-root.tsx
        <AuthHeader
          workspaceSlug={workspaceSlug || undefined}
          invitationId={invitation_id || undefined}
          authMode={authMode}
        />
````
````new web/apps/web/core/components/account/auth-forms/auth-root.tsx
        <AuthHeader invitationId={invitation_id} token={token} authMode={authMode} />
````

- [ ] **Step 4: 旧的方法、键、类型；上限、关键词规则**

`tools/keywords.json`（修改，3 处）：

````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/(?:`|\\$\\{[^}]+\\}/`)",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/",
````
````old tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data)"
````
````new tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })"
````
````old tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/invitations/`, data)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })"
````
````new tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/invitations/`, data)"
````

`web/apps/web/core/services/workspace.service.ts`（修改，3 处）：

````old web/apps/web/core/services/workspace.service.ts
import type {
  IWorkspaceMemberInvitation,
````
````new web/apps/web/core/services/workspace.service.ts
import type {
````
````old web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
  async joinWorkspace(workspaceSlug: string, invitationId: string, data: any): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, data, {
      headers: {},
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
````
````old web/apps/web/core/services/workspace.service.ts
    return this.post(`/api/workspaces/${workspaceSlug}/workspace-views/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getWorkspaceInvitation(workspaceSlug: string, invitationId: string): Promise<IWorkspaceMemberInvitation> {
    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })
````
````new web/apps/web/core/services/workspace.service.ts
    return this.post(`/api/workspaces/${workspaceSlug}/workspace-views/`, data)
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 444",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 442",
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_INVITATION = (invitationId: string) => `WORKSPACE_INVITATION_${invitationId}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/types/src/workspace.ts`（修改，2 处）：

````old web/packages/types/src/workspace.ts

import type { TUserPermissions } from "./enums";

````
````new web/packages/types/src/workspace.ts

````
````old web/packages/types/src/workspace.ts
  GUEST = 5,
}

export interface IWorkspaceMemberInvitation {
  accepted: boolean;
  email: string;
  id: string;
  message: string;
  responded_at: Date;
  role: TUserPermissions;
  token: string;
  invite_link: string;
  workspace: {
    id: string;
    logo_url: string;
    name: string;
    slug: string;
  };
````
````new web/packages/types/src/workspace.ts
  GUEST = 5,
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则，没有命中；web 的 oxlint 442 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过（A9 打开邀请链接的注册页：标题取自公开的查看）。

- [ ] **Step 6: 提交**

```bash
git add tools/keywords.json 'web/apps/web/app/(all)/workspace-invitations/page.tsx' web/apps/web/core/components/account/auth-forms/auth-header.tsx web/apps/web/core/components/account/auth-forms/auth-root.tsx web/apps/web/core/hooks/use-invitation-preview.ts web/apps/web/core/services/workspace.service.ts web/apps/web/core/services/workspace/invitation-preview.service.test.ts web/apps/web/core/services/workspace/invitation-preview.service.ts web/apps/web/core/services/workspace/workspaces.service.ts web/apps/web/core/store/workspace/index.test.ts web/apps/web/core/store/workspace/index.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/types/src/workspace.ts
```
```bash
git commit -m "feat(M3/P8a): an invitation's link is previewed in public and accepted through the workspaces store

previewInvitation asks nerve with the public client what a link shows,
and useInvitationPreview keys it by the link alone, the same for every
session and for none. The invitation page and the sign-in and sign-up
titles read it. Accepting and declining go through the workspaces
store, one at a time: an accepted invitation's workspace joins the list,
in its own place when the caller was a member already.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t9-accept-made-up` | 没有取过列表时，接受的邀请的工作区自成一个列表 | `index.test.ts` | vitest |
| `t9-accept-twice` | 调用者已有的工作区，接受之后又加一次 | `index.test.ts` | vitest |
| `t9-accept-unqueued` | 接受不等待前一个修改 | `index.test.ts` | vitest |
| `t9-decline-unqueued` | 忽略不等待前一个修改 | `index.test.ts` | vitest |
| `t9-preview-no-token` | 查看邀请时不带链接的令牌 | `invitation-preview.service.test.ts` | vitest |
| `t9-join-back` | 旧 service 又查看邀请的链接 | 关键词守卫 | 静态 |
| `t9-accept-unread` | service 不读 nerve 的 problem：被拒绝的接受当作成功 | `index.test.ts` | vitest |
| `t9-decline-unread` | service 不读 nerve 的 problem：被拒绝的忽略当作成功 | `index.test.ts` | vitest |
| `t9-decline-no-token` | 忽略邀请时不带链接的令牌 | `index.test.ts` | vitest |

---

### Task 10: 工作区的显示设置

**Files:**
- Create: `web/apps/web/core/services/workspace/workspace-preferences.service.ts`、`web/apps/web/core/store/workspace/preferences.store.test.ts`、`web/apps/web/core/store/workspace/preferences.store.ts`
- Modify: `tools/keywords.json`、`web/apps/web/core/hooks/use-navigation-preferences.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/issue_filter.service.ts`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/workspace/index.ts`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/types/src/navigation-preferences.ts`、`web/packages/types/src/view-props.ts`

**Interfaces:**
- Produces（spec 2.10；M3 设计 3.18、7.1、7.2、7.3）：
  - `core/services/workspace/workspace-preferences.service.ts`：`class WorkspacePreferencesService { get(slug): Promise<WorkspacePreferences>; update(slug, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> }`（`/api/v0/me/workspaces/{slug}/preferences`）。
  - `core/store/workspace/preferences.store.ts`：`IWorkspacePreferencesStore { getPreferences(slug); fetchPreferences(slug): Promise<WorkspacePreferences | undefined>; updatePreferences(slug, data): Promise<WorkspacePreferences> }`；`WorkspacePreferencesStore(api)` 把每个工作区的设置存在类的 `preferencesMap`（不在接口上），取数遇到 `SessionChangedError` 给出 `undefined`，被拒绝时原样失败、不存任何设置；修改经 `changes = oneAtATime()`，写入 nerve 的回答，不乐观（Plane 先改再回滚，spec 第 3 节）。它是 `WorkspaceRootStore` 的子 store `preferences`；`WorkspaceRootStore` 的三个显示设置的方法、`projectNavigationPreferencesMap` 和它对旧 `WorkspaceService` 的引用删除。
  - `useWorkspaceFetch`：hook 算出的 `isMember` 为真（列表中有这个工作区，Task 8）时再取调用者在这个工作区的显示设置（`["WORKSPACE_PREFERENCES", slug]`）；`WorkspaceAuthWrapper` 原来取显示设置的 SWR 删除。
  - `core/hooks/use-navigation-preferences.ts`（整个文件）：侧边栏的项目导航读 `preferences.getPreferences(slug)`、写 `updatePreferences`；视图模型 `TProjectNavigationPreferences`（`navigationMode: NavigationControlPreference`、`limitedProjectsCount`、`showLimitedProjects`：`limit > 0`）留在 `packages/types`，模式的类型改用生成的 `NavigationControlPreference`，`TProjectNavigationMode` 删除（spec 第 3 节）。
  - 删除：旧 `WorkspaceService` 的 `fetchWorkspaceFilters`、`patchWorkspaceFilters`，`IWorkspaceUserPropertiesResponse`，取数键 `WORKSPACE_PROJECT_NAVIGATION_PREFERENCES`，`issue_filter.service.ts` 中注释掉的同一组方法。
- Adds：`plane-workspace-urls` 加工作区的 `user-properties/`（只到工作区一级：`/api/workspaces/${…}/user-properties/`）；不命中样例加迭代的 `user-properties`（M6）和新的显示设置地址。项目的 `user-properties/` 由 P8b 加进模式，连同 M4 的例外（7.10）。

**Tests:**（vitest）
- `preferences.store.test.ts`，`WorkspacePreferencesStore`：`keeps the caller's settings in a workspace as nerve gives them`；`fails when nerve refuses them, keeping none`；`gives nothing, and does not fail, when the session changes as it fetches`；`has nerve's answer to a change, not the change, and only once nerve answers`（回答与请求不同）；`fails, changing nothing, when nerve refuses a change`；`sends each change once nerve has answered the one before it, refused or not`；`fetches the settings while a change is out: a fetch does not wait for it`（Task 4 的 `fetchedWhileChangeIsOut`）。
- `use-workspace-fetch.test.ts`：`fetches the caller's workspaces and, in one of his, its members and his settings`（取代 Task 8 的同名测试）；`fetches only the caller's workspaces where the address names none of his` 照旧（W5）。

- [ ] **Step 1: 显示设置的 service 和 store**

`web/apps/web/core/services/workspace/workspace-preferences.service.ts`（新文件，25 行）：

````file web/apps/web/core/services/workspace/workspace-preferences.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's own settings of the sidebar's project navigation in a workspace (M3 design 3.18, 5.1). */
export class WorkspacePreferencesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The caller's settings, or the defaults until he first changes them. */
  async get(slug: string): Promise<WorkspacePreferences> {
    return unwrap(await this.api.GET("/api/v0/me/workspaces/{slug}/preferences", { params: { path: { slug } } }));
  }

  /** Changes the settings data names; the answer is all of them as nerve now holds them. */
  async update(slug: string, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> {
    return unwrap(
      await this.api.PATCH("/api/v0/me/workspaces/{slug}/preferences", { params: { path: { slug } }, body: data })
    );
  }
}
````

`web/apps/web/core/store/workspace/index.ts`（修改，13 处）：

````old web/apps/web/core/store/workspace/index.ts

import { clone } from "lodash-es";
import { action, computed, observable, makeObservable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
````
````new web/apps/web/core/store/workspace/index.ts

import { action, computed, observable, makeObservable, runInAction } from "mobx";
````
````old web/apps/web/core/store/workspace/index.ts
import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
import type { IWorkspaceUserPropertiesResponse } from "@nerve/types";
````
````new web/apps/web/core/store/workspace/index.ts
import type { ApiClient, SlugAvailability, Workspace, WorkspaceCreate, WorkspaceUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/store/workspace/index.ts
// services
import { WorkspaceService } from "@/services/workspace.service";
````
````new web/apps/web/core/store/workspace/index.ts
// services
````
````old web/apps/web/core/store/workspace/index.ts
import type { RootStore } from "@/store/root.store";
// sub-stores
````
````new web/apps/web/core/store/workspace/index.ts
import type { RootStore } from "@/store/root.store";
// sub-stores
import type { IWorkspacePreferencesStore } from "./preferences.store";
import { WorkspacePreferencesStore } from "./preferences.store";
````
````old web/apps/web/core/store/workspace/index.ts
  workspaces: Workspace[] | undefined;
  projectNavigationPreferencesMap: Record<string, IWorkspaceUserPropertiesResponse>;
````
````new web/apps/web/core/store/workspace/index.ts
  workspaces: Workspace[] | undefined;
````
````old web/apps/web/core/store/workspace/index.ts
  declineInvitation: (invitationId: string, token: string) => Promise<void>;
  getProjectNavigationPreferences: (workspaceSlug: string) => IWorkspaceUserPropertiesResponse | undefined;
  fetchProjectNavigationPreferences: (workspaceSlug: string) => Promise<void>;
  updateProjectNavigationPreferences: (
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ) => Promise<void>;
  // sub-stores
````
````new web/apps/web/core/store/workspace/index.ts
  declineInvitation: (invitationId: string, token: string) => Promise<void>;
  // sub-stores
  preferences: IWorkspacePreferencesStore;
````
````old web/apps/web/core/store/workspace/index.ts
  workspaces: Workspace[] | undefined = undefined;
  projectNavigationPreferencesMap: Record<string, IWorkspaceUserPropertiesResponse> = {};
  // services
  workspaceService;
````
````new web/apps/web/core/store/workspace/index.ts
  workspaces: Workspace[] | undefined = undefined;
  // services
````
````old web/apps/web/core/store/workspace/index.ts
  router;
  // sub-stores
````
````new web/apps/web/core/store/workspace/index.ts
  router;
  // sub-stores
  preferences: IWorkspacePreferencesStore;
````
````old web/apps/web/core/store/workspace/index.ts
      workspaces: observable.ref,
      projectNavigationPreferencesMap: observable,
````
````new web/apps/web/core/store/workspace/index.ts
      workspaces: observable.ref,
````
````old web/apps/web/core/store/workspace/index.ts
      declineInvitation: action,
      fetchProjectNavigationPreferences: action,
      updateProjectNavigationPreferences: action,
````
````new web/apps/web/core/store/workspace/index.ts
      declineInvitation: action,
````
````old web/apps/web/core/store/workspace/index.ts
    // services
    this.workspaceService = new WorkspaceService();
````
````new web/apps/web/core/store/workspace/index.ts
    // services
````
````old web/apps/web/core/store/workspace/index.ts
    // sub-stores
````
````new web/apps/web/core/store/workspace/index.ts
    // sub-stores
    this.preferences = new WorkspacePreferencesStore(api);
````
````old web/apps/web/core/store/workspace/index.ts
  }

  getProjectNavigationPreferences = computedFn(
    (workspaceSlug: string): IWorkspaceUserPropertiesResponse | undefined =>
      this.projectNavigationPreferencesMap[workspaceSlug]
  );

  fetchProjectNavigationPreferences = async (workspaceSlug: string) => {
    try {
      const response = await this.workspaceService.fetchWorkspaceFilters(workspaceSlug);

      runInAction(() => {
        this.projectNavigationPreferencesMap[workspaceSlug] = response;
      });
    } catch (error) {
      console.error("Failed to fetch project navigation preferences:", error);
      throw error;
    }
  };

  updateProjectNavigationPreferences = async (
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ) => {
    const beforeUpdateData = clone(this.projectNavigationPreferencesMap[workspaceSlug]);

    try {
      // Optimistically update store
      runInAction(() => {
        this.projectNavigationPreferencesMap[workspaceSlug] = {
          ...this.projectNavigationPreferencesMap[workspaceSlug],
          ...data,
        };
      });

      // Call API to persist changes
      await this.workspaceService.patchWorkspaceFilters(workspaceSlug, data);
    } catch (error) {
      // Rollback on failure
      runInAction(() => {
        this.projectNavigationPreferencesMap[workspaceSlug] = beforeUpdateData;
      });
      console.error("Failed to update project navigation preferences:", error);
      throw error;
    }
  };
````
````new web/apps/web/core/store/workspace/index.ts
  }
````

`web/apps/web/core/store/workspace/preferences.store.test.ts`（新文件，126 行）：

````file web/apps/web/core/store/workspace/preferences.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiClient, WorkspacePreferences } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { WorkspacePreferencesStore } from "@/store/workspace/preferences.store";

// The caller's navigation settings in his workspaces (M3 design 3.18, 7.3), against a fake nerve that answers each
// request when the test says.

const PREFERENCES = "/api/v0/me/workspaces/acme/preferences";
/** nerve's defaults, which it gives until the caller changes one. */
const defaults: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 };
const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 10 };
/** nerve's answer to the change to tabs, once another tab of his has set the limit to 3. */
const elsewhere: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };

/** The store of a tab; client builds its client. */
function preferencesStore(client: (nerve: FakeNerve) => ApiClient = (nerve) => nerve.client()) {
  const nerve = new FakeNerve();
  return { nerve, store: new WorkspacePreferencesStore(client(nerve)) };
}

/** A store whose settings in acme nerve gave as its defaults. */
async function loaded() {
  const tab = preferencesStore();
  const fetched = tab.store.fetchPreferences("acme");
  await until(() => tab.nerve.calls.length === 1, "the settings");
  tab.nerve.calls[0]?.answer(json(200, defaults));
  await settle(fetched, "the settings");
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspacePreferencesStore", () => {
  it("keeps the caller's settings in a workspace as nerve gives them", async () => {
    const { nerve, store } = preferencesStore();
    const fetched = store.fetchPreferences("acme");
    await until(() => nerve.calls.length === 1, "the settings");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: PREFERENCES });
    expect(store.getPreferences("acme")).toBeUndefined();
    nerve.calls[0]?.answer(json(200, defaults));
    expect(await settle(fetched, "the settings")).toEqual({ settled: true, value: defaults });
    expect(store.getPreferences("acme")).toEqual(defaults);
    expect(store.getPreferences("beta")).toBeUndefined();
  });

  it("fails when nerve refuses them, keeping none", async () => {
    const { nerve, store } = preferencesStore();
    const refused = track(store.fetchPreferences("acme"));
    await until(() => nerve.calls.length === 1, "the settings");
    nerve.calls[0]?.answer(problem(404, "workspace.not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.preferencesMap).toEqual({});
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = preferencesStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchPreferences("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.preferencesMap).toEqual({});
  });

  it("has nerve's answer to a change, not the change, and only once nerve answers", async () => {
    const { nerve, store } = await loaded();
    const changed = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: PREFERENCES,
      body: { navigation_control_preference: "TABBED" },
    });
    expect(store.getPreferences("acme")).toEqual(defaults);
    nerve.calls[1]?.answer(json(200, elsewhere));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(elsewhere);
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });

  it("fails, changing nothing, when nerve refuses a change", async () => {
    const { nerve, store } = await loaded();
    const refused = track(store.updatePreferences("acme", { navigation_project_limit: -1 }));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(problem(422, "validation_failed"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toEqual(defaults);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updatePreferences("acme", { navigation_project_limit: 3 }));
    const second = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, tabbed));
    await until(() => second.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toEqual(tabbed);
  });

  it("fetches the settings while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      () => store.updatePreferences("acme", { navigation_project_limit: 3 }),
      () => store.fetchPreferences("acme"),
      ["GET", PREFERENCES],
      json(200, elsewhere)
    );
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });
});
````

`web/apps/web/core/store/workspace/preferences.store.ts`（新文件，77 行）：

````file web/apps/web/core/store/workspace/preferences.store.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { ApiClient, WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
// services
import { WorkspacePreferencesService } from "@/services/workspace/workspace-preferences.service";

export interface IWorkspacePreferencesStore {
  getPreferences: (workspaceSlug: string) => WorkspacePreferences | undefined;
  fetchPreferences: (workspaceSlug: string) => Promise<WorkspacePreferences | undefined>;
  updatePreferences: (workspaceSlug: string, data: WorkspacePreferencesUpdate) => Promise<WorkspacePreferences>;
}

/**
 * The caller's own settings of the sidebar's project navigation in his workspaces (M3 design 3.18, 7.3): its
 * service sends with the session's client, which the RootStore of the session hands down. Changes go one at a time
 * (v0 design 7.7); fetches do not queue.
 */
export class WorkspacePreferencesStore implements IWorkspacePreferencesStore {
  /** The caller's settings of the sidebar's project navigation, by workspace slug, as nerve last gave them. */
  preferencesMap: Record<string, WorkspacePreferences> = {};
  private readonly service: WorkspacePreferencesService;
  /** The changes of the settings, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(api: ApiClient) {
    makeObservable(this, {
      preferencesMap: observable,
      fetchPreferences: action,
      updatePreferences: action,
    });
    this.service = new WorkspacePreferencesService(api);
  }

  /** The caller's settings in the workspace, once fetched. */
  getPreferences = (workspaceSlug: string): WorkspacePreferences | undefined => this.preferencesMap[workspaceSlug];

  /**
   * @description fetches the caller's settings in a workspace, nerve's defaults until he changes one, and gives
   * them. A change of session while they load is no failure: the new session's store fetches its own
   * (store-context.tsx), and this one gives undefined.
   * @returns {Promise<WorkspacePreferences | undefined>}
   */
  fetchPreferences = async (workspaceSlug: string): Promise<WorkspacePreferences | undefined> => {
    try {
      const preferences = await this.service.get(workspaceSlug);
      runInAction(() => {
        this.preferencesMap[workspaceSlug] = preferences;
      });
      return preferences;
    } catch (error) {
      if (error instanceof SessionChangedError) return undefined;
      throw error;
    }
  };

  /**
   * @description changes the settings data names; the store then has nerve's answer, all of them. Fails, changing
   * nothing, when nerve refuses.
   * @returns {Promise<WorkspacePreferences>}
   */
  updatePreferences = (workspaceSlug: string, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> =>
    this.changes(async () => {
      const preferences = await this.service.update(workspaceSlug, data);
      runInAction(() => {
        this.preferencesMap[workspaceSlug] = preferences;
      });
      return preferences;
    });
}
````

- [ ] **Step 2: 工作区的页面按会话取显示设置；侧边栏读写新 store**

`web/apps/web/core/hooks/use-navigation-preferences.ts`（整个文件，79 行）：

````whole web/apps/web/core/hooks/use-navigation-preferences.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useParams } from "react-router";
import type { NavigationControlPreference } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
import { DEFAULT_PROJECT_PREFERENCES } from "@nerve/types";
import { useWorkspace } from "./store/use-workspace";

export const useProjectNavigationPreferences = () => {
  const { workspaceSlug } = useParams();
  const {
    preferences: { getPreferences, updatePreferences },
  } = useWorkspace();

  // The caller's settings as nerve gave them: its defaults until he changes one (M3 design 3.18)
  const storePreferences = getPreferences(workspaceSlug || "");

  // Until they load, the sidebar uses the defaults; a limit of 0 shows every project
  const preferences: TProjectNavigationPreferences = useMemo(() => {
    if (!storePreferences) return DEFAULT_PROJECT_PREFERENCES;
    const limit = storePreferences.navigation_project_limit;
    return {
      navigationMode: storePreferences.navigation_control_preference,
      limitedProjectsCount: limit > 0 ? limit : DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount,
      showLimitedProjects: limit > 0,
    };
  }, [storePreferences]);

  // Update navigation mode
  const updateNavigationMode = useCallback(
    async (mode: NavigationControlPreference) => {
      if (!workspaceSlug) return;

      await updatePreferences(workspaceSlug, {
        navigation_control_preference: mode,
      });
    },
    [workspaceSlug, updatePreferences]
  );

  // Update show limited projects
  const updateShowLimitedProjects = useCallback(
    async (show: boolean) => {
      if (!workspaceSlug) return;

      // When toggling off, set to 0; when toggling on, use current count or default
      const newLimit = show ? preferences.limitedProjectsCount || DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount : 0;

      await updatePreferences(workspaceSlug, {
        navigation_project_limit: newLimit,
      });
    },
    [workspaceSlug, updatePreferences, preferences.limitedProjectsCount]
  );

  // Update limited projects count
  const updateLimitedProjectsCount = useCallback(
    async (count: number) => {
      if (!workspaceSlug) return;

      await updatePreferences(workspaceSlug, {
        navigation_project_limit: count,
      });
    },
    [workspaceSlug, updatePreferences]
  );

  return {
    preferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  };
};
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`（修改，5 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    getWorkspaceBySlug: (slug: string) => (slug === "acme" ? { slug } : null),
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    getWorkspaceBySlug: (slug: string) => (slug === "acme" ? { slug } : null),
    preferences: {
      fetchPreferences: (slug: string) => Promise.resolve(fetched.calls.push(`the settings in ${slug}`)),
    },
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members", async () => {
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members and his settings", async () => {
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], ["WORKSPACE_MEMBERS", "acme"]]);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["WORKSPACES"],
      ["WORKSPACE_MEMBERS", "acme"],
      ["WORKSPACE_PREFERENCES", "acme"],
    ]);
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(fetched.calls).toEqual(["the workspaces", "the members of acme"]);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(fetched.calls).toEqual(["the workspaces", "the members of acme", "the settings in acme"]);
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null]);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null]);
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`（修改，3 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
 * workspaces, which decide whether he may see the address's one; once his list has it, its members. Gives the
 * list's response, whose failure the page shows.
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
 * workspaces, which decide whether he may see the address's one; once his list has it, its members and his
 * navigation settings in it. Gives the list's response, whose failure the page shows.
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  const { fetchWorkspaces, getWorkspaceBySlug } = useWorkspace();
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  const {
    fetchWorkspaces,
    getWorkspaceBySlug,
    preferences: { fetchPreferences },
  } = useWorkspace();
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  useSessionSWR(isMember ? ["WORKSPACE_MEMBERS", workspaceSlug] : null, (slug) => fetchWorkspaceMembers(slug), ONCE);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  useSessionSWR(isMember ? ["WORKSPACE_MEMBERS", workspaceSlug] : null, (slug) => fetchWorkspaceMembers(slug), ONCE);
  useSessionSWR(isMember ? ["WORKSPACE_PREFERENCES", workspaceSlug] : null, (slug) => fetchPreferences(slug), ONCE);
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，3 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  WORKSPACE_STATES,
  WORKSPACE_PROJECT_NAVIGATION_PREFERENCES,
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  WORKSPACE_STATES,
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { workspaces, getWorkspaceBySlug, fetchProjectNavigationPreferences } = useWorkspace();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { workspaces, getWorkspaceBySlug } = useWorkspace();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
    workspaceSlug ? () => fetchWorkspaceStates(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetch workspace project navigation preferences
  useSWR(
    workspaceSlug ? WORKSPACE_PROJECT_NAVIGATION_PREFERENCES(workspaceSlug) : null,
    workspaceSlug ? () => fetchProjectNavigationPreferences(workspaceSlug) : null,
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
    workspaceSlug ? () => fetchWorkspaceStates(workspaceSlug) : null,
````

`web/packages/types/src/navigation-preferences.ts`（修改，2 处）：

````old web/packages/types/src/navigation-preferences.ts
export type TProjectNavigationMode = "ACCORDION" | "TABBED";
````
````new web/packages/types/src/navigation-preferences.ts
import type { NavigationControlPreference } from "@nerve/api-client";
````
````old web/packages/types/src/navigation-preferences.ts
  navigationMode: TProjectNavigationMode;
````
````new web/packages/types/src/navigation-preferences.ts
  navigationMode: NavigationControlPreference;
````

- [ ] **Step 3: 旧的方法、类型、键；关键词规则**

`tools/keywords.json`（修改，3 处）：

````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/|/api/workspaces/\\$\\{[^}]+\\}/user-properties/",
````
````old tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })"
````
````new tools/keywords.json
          "    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })",
          "    return this.get(`/api/workspaces/${workspaceSlug}/user-properties/`)",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data)"
````
````old tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/invitations/`, data)"
````
````new tools/keywords.json
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/invitations/`, data)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/user-properties/`)",
          "    return unwrap(await this.api.GET(\"/api/v0/me/workspaces/{slug}/preferences\", { params: { path: { slug } } }));"
````

`web/apps/web/core/services/issue_filter.service.ts`（修改，1 处）：

````old web/apps/web/core/services/issue_filter.service.ts
export class IssueFiltersService extends APIService {
  // // workspace issue filters
  // async fetchWorkspaceFilters(workspaceSlug: string): Promise<IIssueFiltersResponse> {
  //   return this.get(`/api/workspaces/${workspaceSlug}/user-properties/`)
  //     .then((response) => response?.data)
  //     .catch((error) => {
  //       throw error?.response?.data;
  //     });
  // }
  // async patchWorkspaceFilters(
  //   workspaceSlug: string,
  //   data: Partial<IIssueFiltersResponse>
  // ): Promise<IIssueFiltersResponse> {
  //   return this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data)
  //     .then((response) => response?.data)
  //     .catch((error) => {
  //       throw error?.response?.data;
  //     });
  // }

````
````new web/apps/web/core/services/issue_filter.service.ts
export class IssueFiltersService extends APIService {
````

`web/apps/web/core/services/workspace.service.ts`（修改，2 处）：

````old web/apps/web/core/services/workspace.service.ts
  TActivityEntityData,
  IWorkspaceUserPropertiesResponse,
````
````new web/apps/web/core/services/workspace.service.ts
  TActivityEntityData,
````
````old web/apps/web/core/services/workspace.service.ts
  }

  async fetchWorkspaceFilters(workspaceSlug: string): Promise<IWorkspaceUserPropertiesResponse> {
    return this.get(`/api/workspaces/${workspaceSlug}/user-properties/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async patchWorkspaceFilters(
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ): Promise<IWorkspaceUserPropertiesResponse> {
    return this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
````
````new web/apps/web/core/services/workspace.service.ts
  }
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_PROJECT_NAVIGATION_PREFERENCES = (workspaceSlug: string) =>
  `WORKSPACE_PROJECT_NAVIGATION_PREFERENCES_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/types/src/view-props.ts`（修改，1 处）：

````old web/packages/types/src/view-props.ts

export interface IWorkspaceUserPropertiesResponse extends IIssueFiltersResponse {
  navigation_project_limit?: number;
  navigation_control_preference?: "ACCORDION" | "TABBED";
  // Note: show_limited_projects is derived from navigation_project_limit (0 = false, >0 = true)
}

````
````new web/packages/types/src/view-props.ts

````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则，没有命中；web 的 oxlint 442 条。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add tools/keywords.json web/apps/web/core/hooks/use-navigation-preferences.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/issue_filter.service.ts web/apps/web/core/services/workspace.service.ts web/apps/web/core/services/workspace/workspace-preferences.service.ts web/apps/web/core/store/workspace/index.ts web/apps/web/core/store/workspace/preferences.store.test.ts web/apps/web/core/store/workspace/preferences.store.ts web/packages/constants/src/fetch-keys.ts web/packages/types/src/navigation-preferences.ts web/packages/types/src/view-props.ts
```
```bash
git commit -m "feat(M3/P8a): the caller's settings in a workspace come from /api/v0

WorkspacePreferencesService wraps the generated client, and a
preferences store under the workspaces store keeps the caller's
settings per workspace. Changes go one at a time and take nerve's
answer instead of a guess rolled back on failure. A workspace's pages
fetch them by the session once the list says the workspace is the
caller's, and the sidebar's project navigation reads and writes them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t10-prefs-early` | 显示设置的修改在 nerve 应答之前就显示 | `preferences.store.test.ts` | vitest |
| `t10-prefs-request` | 显示设置保留请求的值，不是 nerve 的应答 | `preferences.store.test.ts` | vitest |
| `t10-prefs-unqueued` | 显示设置的修改不等待前一个 | `preferences.store.test.ts` | vitest |
| `t10-prefs-session` | 显示设置加载时换了会话，取数失败 | `preferences.store.test.ts` | vitest |
| `t10-prefs-swallows` | 显示设置被拒绝时给出 `undefined`，不失败 | oxlint（`check:lint`）、`preferences.store.test.ts` | 静态；vitest |
| `t10-prefs-fetch-queued` | 显示设置的取数等待正在发出的修改 | `preferences.store.test.ts` | vitest |
| `t10-wrapper-prefs-ungated` | 工作区的页面在列表说明它是调用者的工作区之前就取他在其中的显示设置 | `use-workspace-fetch.test.ts` | vitest |
| `t10-wrapper-prefs-drops` | 工作区的页面取显示设置的 fetcher 丢掉 store 的 Promise | `tsc` | 静态 |
| `t10-user-properties-back` | 旧 service 又读调用者在工作区的显示设置 | 关键词守卫 | 静态 |
| `t10-prefs-unkept` | 取到的显示设置交回调用方但不存下：侧边栏读不到 | `preferences.store.test.ts` | vitest |

---

### Task 11: 挂载时取 M6、M7 的数据删除；首页的"最近"小部件删除

**Files:**
- Modify: `web/apps/web/core/components/core/list/list-item.tsx`、`web/apps/web/core/components/home/home-body.tsx`、`web/apps/web/core/components/home/widgets/empty-states/index.ts`、`web/apps/web/core/components/home/widgets/index.ts`、`web/apps/web/core/components/home/widgets/loaders/index.ts`、`web/apps/web/core/components/navigation/top-navigation-root.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/favorite/favorite.service.ts`、`web/apps/web/core/services/view.service.ts`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/favorite.store.ts`、`web/apps/web/core/store/module.store.ts`、`web/apps/web/core/store/notifications/workspace-notifications.store.ts`、`web/apps/web/core/store/project-view.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/home.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/home.json`、`web/packages/types/src/index.ts`
- Delete: `web/apps/web/core/components/core/content-overflow-HOC.tsx`、`web/apps/web/core/components/home/widgets/empty-states/recents.tsx`、`web/apps/web/core/components/home/widgets/loaders/recent-activity.tsx`、`web/apps/web/core/components/home/widgets/recents/filters.tsx`、`web/apps/web/core/components/home/widgets/recents/index.tsx`、`web/apps/web/core/components/home/widgets/recents/issue.tsx`、`web/apps/web/core/components/home/widgets/recents/project.tsx`、`web/packages/types/src/home.ts`

**Interfaces:**
- Removes（spec 2.11；M3 设计 3.1、7.5、7.6、7.8；裁定 R1 的包装层一句）：
  - `WorkspaceAuthWrapper` 取收藏（M7）的 SWR；`TopNavigationRoot` 取未读通知数（M7）的 SWR：通知按钮没有数字，注释写明到 M7；`ProjectAuthWrapper` 取迭代、两次模块（M6）和视图、分诊状态（M7）的 SWR，连同它们的 hook 和键（`WORKSPACE_FAVORITE`、`PROJECT_ALL_CYCLES`、`PROJECT_MODULES`、`PROJECT_VIEWS`、`PROJECT_INTAKE_STATE`）。`ProjectAuthWrapper` 中 M3 自己的取数（项目详情、成员信息、显示设置、标签、成员、状态）不动，属于 P8b。
  - 首页的"最近"小部件：`HomeBody` 不再渲染它，`home/widgets/recents/` 四个文件、只为它存在的空状态和骨架、`content-overflow-HOC.tsx`（knip 报未使用）、`packages/types/src/home.ts`、旧 `WorkspaceService.fetchWorkspaceRecents`、两种语言的 `home.recents.*`（7 条）和 `show_all`、`show_less`。首页在 M7 之前只有问候和"还没有项目"的引导（P9）。
  - 上面的删除留下的、没有调用方的代码一并删除（M3 设计 3.2 的写法：`favoriteProjectIds` 没有读者，删除，M7 加回）：`FavoriteStore.fetchFavorite`、`FavoriteService.getFavorites`、`ModulesStore.fetchModulesSlim`、`ProjectViewStore.fetchViews`、`ViewService.getViews`，`IWorkspaceNotificationStore.getUnreadNotificationsCount`（类里的方法留下：取通知列表时它自己调用），`ListItem` 只有"最近"小部件传的 `id`、`disableLink`、`itemClassName`、`preventDefaultProgress`；旧 `WorkspaceService` 没有调用方的 `updateWorkspaceView`。
  - 这些文件因此有了手改，按 7.9 清零：`module.store.ts` 的两条 `no-useless-catch`、三条 `always-return`，`project-view.store.ts` 的一条 `always-return`（`then` 回调改为 `await`，`.catch` 改为 `try`，行为不变），`list-item.tsx` 的一条 `const-comparisons`（`{quickActionElement && quickActionElement}` 即 `{quickActionElement}`）。web 的上限 442 → 435。
- 本 Task 不加关键词规则：收藏、通知、迭代、模块、视图、分诊状态、"最近"的地址是 M6、M7 的，取它们的方法（例如 `fetchModules`）多数仍在 store 上。它们中任何一个重回挂载路径都由 P8b 改写的 S2 发现：S2 断言每个账户的每张挂载清单上都没有 M6、M7 的地址（spec 第 5 节；附录 A 的两个存活的变异是这一类的样例）。

**Tests:** 没有新的 vitest：删除的是取数和代码，`make lint-web`（`tsc`、oxlint 的上限）、`make knip` 发现漏删的使用方；`make e2e` 的工作区页面（W1、W3、W10）照旧通过，挂载时不再有这些请求（spec 附录 A 的 W2 清单）。

- [ ] **Step 1: 包装层和顶部导航的 M6、M7 取数**

`web/apps/web/core/components/navigation/top-navigation-root.tsx`（修改，2 处）：

````old web/apps/web/core/components/navigation/top-navigation-root.tsx
import { InboxOutline } from "@makeplane/propel/icons";
import useSWR from "swr";
````
````new web/apps/web/core/components/navigation/top-navigation-root.tsx
import { InboxOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/navigation/top-navigation-root.tsx
  // store hooks
  const { unreadNotificationsCount, getUnreadNotificationsCount } = useWorkspaceNotifications();

  // Fetch notification count
  useSWR(
    workspaceSlug ? "WORKSPACE_UNREAD_NOTIFICATION_COUNT" : null,
    workspaceSlug ? () => getUnreadNotificationsCount(workspaceSlug) : null
  );
````
````new web/apps/web/core/components/navigation/top-navigation-root.tsx
  // store hooks: no page M3 reaches fetches the unread count until M7 (M3 design 3.1), so the button has no dot
  const { unreadNotificationsCount } = useWorkspaceNotifications();
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，7 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  PROJECT_STATES,
  PROJECT_ALL_CYCLES,
  PROJECT_MODULES,
  PROJECT_VIEWS,
  PROJECT_INTAKE_STATE,
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  PROJECT_STATES,
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
// hooks
import { useCycle } from "@/hooks/store/use-cycle";
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
// hooks
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
import { useMember } from "@/hooks/store/use-member";
import { useModule } from "@/hooks/store/use-module";
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
import { useMember } from "@/hooks/store/use-member";
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
import { useProjectState } from "@/hooks/store/use-project-state";
import { useProjectView } from "@/hooks/store/use-project-view";
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
import { useProjectState } from "@/hooks/store/use-project-state";
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { joinProject } = useUserPermissions();
  const { fetchAllCycles } = useCycle();
  const { fetchModulesSlim, fetchModules } = useModule();
  const { fetchViews } = useProjectView();
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { joinProject } = useUserPermissions();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { fetchProjectStates, fetchProjectIntakeState } = useProjectState();
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { fetchProjectStates } = useProjectState();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  });
  // fetching project intake state
  useSWR(PROJECT_INTAKE_STATE(projectId, currentProjectRole), () => fetchProjectIntakeState(workspaceSlug, projectId), {
    revalidateIfStale: false,
    revalidateOnFocus: false,
  });
  // fetching project cycles
  useSWR(PROJECT_ALL_CYCLES(projectId, currentProjectRole), () => fetchAllCycles(workspaceSlug, projectId), {
    revalidateIfStale: false,
    revalidateOnFocus: false,
  });
  // fetching project modules
  useSWR(
    PROJECT_MODULES(projectId, currentProjectRole),
    async () => {
      await Promise.all([fetchModulesSlim(workspaceSlug, projectId), fetchModules(workspaceSlug, projectId)]);
    },
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  // fetching project views
  useSWR(PROJECT_VIEWS(projectId, currentProjectRole), () => fetchViews(workspaceSlug, projectId), {
    revalidateIfStale: false,
    revalidateOnFocus: false,
  });
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  });
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，5 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { LogOutOutline } from "@makeplane/propel/icons";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { LogOutOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import {
  WORKSPACE_PARTIAL_PROJECTS,
  WORKSPACE_PROJECTS_ROLES_INFORMATION,
  WORKSPACE_FAVORITE,
  WORKSPACE_STATES,
} from "@nerve/constants";
// hooks
import { useFavorite } from "@/hooks/store/use-favorite";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { WORKSPACE_PARTIAL_PROJECTS, WORKSPACE_PROJECTS_ROLES_INFORMATION, WORKSPACE_STATES } from "@nerve/constants";
// hooks
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchPartialProjects } = useProject();
  const { fetchFavorite } = useFavorite();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchPartialProjects } = useProject();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchUserProjectPermissions, allowPermissions } = useUserPermissions();
  const { fetchWorkspaceStates } = useProjectState();
  // derived values
  const canPerformWorkspaceMemberActions = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE
  );
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { fetchUserProjectPermissions } = useUserPermissions();
  const { fetchWorkspaceStates } = useProjectState();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
    workspaceSlug && currentWorkspace ? () => fetchPartialProjects(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  // fetch workspace favorite
  useSWR(
    workspaceSlug && currentWorkspace && canPerformWorkspaceMemberActions ? WORKSPACE_FAVORITE(workspaceSlug) : null,
    workspaceSlug && currentWorkspace && canPerformWorkspaceMemberActions ? () => fetchFavorite(workspaceSlug) : null,
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
    workspaceSlug && currentWorkspace ? () => fetchPartialProjects(workspaceSlug) : null,
````

`web/packages/constants/src/fetch-keys.ts`（修改，2 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_FAVORITE = (workspaceSlug: string) => `WORKSPACE_FAVORITE_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````
````old web/packages/constants/src/fetch-keys.ts

export const PROJECT_INTAKE_STATE = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_INTAKE_STATE_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_ALL_CYCLES = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_ALL_CYCLES_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_MODULES = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_MODULES_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_VIEWS = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_VIEWS_${projectId.toUpperCase()}_${projectRole}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

- [ ] **Step 2: "最近"小部件**

`web/apps/web/core/components/core/content-overflow-HOC.tsx`（删除）：

````delete web/apps/web/core/components/core/content-overflow-HOC.tsx
````

`web/apps/web/core/components/home/home-body.tsx`（修改，3 处）：

````old web/apps/web/core/components/home/home-body.tsx
import { HomeLoader, NoProjectsEmptyState, RecentActivityWidget } from "./widgets";
````
````new web/apps/web/core/components/home/home-body.tsx
import { HomeLoader, NoProjectsEmptyState } from "./widgets";
````
````old web/apps/web/core/components/home/home-body.tsx
/** The home body is a fixed list (M1 design 3.14): the no-projects empty state and the recent visits. */
````
````new web/apps/web/core/components/home/home-body.tsx
/**
 * The home body is a fixed list (M1 design 3.14): the no-projects empty state. The recent visits come back with
 * M7 (M3 design 3.1).
 */
````
````old web/apps/web/core/components/home/home-body.tsx
      <NoProjectsEmptyState />
      <div className="py-4">
        <RecentActivityWidget workspaceSlug={workspaceSlug} />
      </div>
````
````new web/apps/web/core/components/home/home-body.tsx
      <NoProjectsEmptyState />
````

`web/apps/web/core/components/home/widgets/empty-states/index.ts`（修改，1 处）：

````old web/apps/web/core/components/home/widgets/empty-states/index.ts
export * from "./no-projects";
export * from "./recents";
````
````new web/apps/web/core/components/home/widgets/empty-states/index.ts
export * from "./no-projects";
````

`web/apps/web/core/components/home/widgets/empty-states/recents.tsx`（删除）：

````delete web/apps/web/core/components/home/widgets/empty-states/recents.tsx
````

`web/apps/web/core/components/home/widgets/index.ts`（修改，1 处）：

````old web/apps/web/core/components/home/widgets/index.ts
export * from "./loaders";
export * from "./recents";
````
````new web/apps/web/core/components/home/widgets/index.ts
export * from "./loaders";
````

`web/apps/web/core/components/home/widgets/loaders/index.ts`（修改，1 处）：

````old web/apps/web/core/components/home/widgets/loaders/index.ts
export * from "./home-loader";
export * from "./recent-activity";
````
````new web/apps/web/core/components/home/widgets/loaders/index.ts
export * from "./home-loader";
````

`web/apps/web/core/components/home/widgets/loaders/recent-activity.tsx`（删除）：

````delete web/apps/web/core/components/home/widgets/loaders/recent-activity.tsx
````

`web/apps/web/core/components/home/widgets/recents/filters.tsx`（删除）：

````delete web/apps/web/core/components/home/widgets/recents/filters.tsx
````

`web/apps/web/core/components/home/widgets/recents/index.tsx`（删除）：

````delete web/apps/web/core/components/home/widgets/recents/index.tsx
````

`web/apps/web/core/components/home/widgets/recents/issue.tsx`（删除）：

````delete web/apps/web/core/components/home/widgets/recents/issue.tsx
````

`web/apps/web/core/components/home/widgets/recents/project.tsx`（删除）：

````delete web/apps/web/core/components/home/widgets/recents/project.tsx
````

`web/apps/web/core/services/workspace.service.ts`（修改，4 处）：

````old web/apps/web/core/services/workspace.service.ts
  IWorkspaceSearchResults,
  IWorkspaceViewProps,
````
````new web/apps/web/core/services/workspace.service.ts
  IWorkspaceSearchResults,
````
````old web/apps/web/core/services/workspace.service.ts
  TSearchEntityRequestPayload,
  TActivityEntityData,
````
````new web/apps/web/core/services/workspace.service.ts
  TSearchEntityRequestPayload,
````
````old web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
  async updateWorkspaceView(workspaceSlug: string, data: { view_props: IWorkspaceViewProps }): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/workspace-views/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts
export class WorkspaceService extends APIService {
````
````old web/apps/web/core/services/workspace.service.ts
  }

  // recents
  async fetchWorkspaceRecents(workspaceSlug: string, entity_name?: string): Promise<TActivityEntityData[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/recent-visits/`, {
      params: {
        entity_name,
      },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }
````
````new web/apps/web/core/services/workspace.service.ts
  }
````

`web/packages/i18n/src/locales/en/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/common.json
  "evening": "evening",
  "show_all": "Show all",
  "show_less": "Show less",
````
````new web/packages/i18n/src/locales/en/common.json
  "evening": "evening",
````

`web/packages/i18n/src/locales/en/home.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/home.json
    },
    "recents": {
      "title": "Recents",
      "empty": {
        "project": "Your recent projects will appear here once you visit one.",
        "issue": "Your recent work items will appear here once you visit one.",
        "default": "You don't have any recents yet."
      },
      "filters": {
        "all": "All",
        "projects": "Projects",
        "issues": "Work items"
      }
    },
````
````new web/packages/i18n/src/locales/en/home.json
    },
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "evening": "晚上",
  "show_all": "显示全部",
  "show_less": "显示更少",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "evening": "晚上",
````

`web/packages/i18n/src/locales/zh-CN/home.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/home.json
    },
    "recents": {
      "title": "最近",
      "empty": {
        "project": "访问项目后，您的最近项目将显示在这里。",
        "issue": "访问工作项后，您的最近工作项将显示在这里。",
        "default": "您还没有任何最近项目。"
      },
      "filters": {
        "all": "所有",
        "projects": "项目",
        "issues": "工作项"
      }
    },
````
````new web/packages/i18n/src/locales/zh-CN/home.json
    },
````

`web/packages/types/src/home.ts`（删除）：

````delete web/packages/types/src/home.ts
````

`web/packages/types/src/index.ts`（修改，1 处）：

````old web/packages/types/src/index.ts
export * from "./file";
export * from "./home";
````
````new web/packages/types/src/index.ts
export * from "./file";
````

- [ ] **Step 3: 留下的没有调用方的代码，和这些文件的 oxlint**

`web/apps/web/core/components/core/list/list-item.tsx`（修改，9 处）：

````old web/apps/web/core/components/core/list/list-item.tsx
interface IListItemProps {
  id?: string;
````
````new web/apps/web/core/components/core/list/list-item.tsx
interface IListItemProps {
````
````old web/apps/web/core/components/core/list/list-item.tsx
  parentRef: React.RefObject<HTMLDivElement | null>;
  disableLink?: boolean;
  className?: string;
  itemClassName?: string;
````
````new web/apps/web/core/components/core/list/list-item.tsx
  parentRef: React.RefObject<HTMLDivElement | null>;
  className?: string;
````
````old web/apps/web/core/components/core/list/list-item.tsx
  quickActionElement?: React.ReactNode;
  preventDefaultProgress?: boolean;
````
````new web/apps/web/core/components/core/list/list-item.tsx
  quickActionElement?: React.ReactNode;
````
````old web/apps/web/core/components/core/list/list-item.tsx
  const {
    id,
````
````new web/apps/web/core/components/core/list/list-item.tsx
  const {
````
````old web/apps/web/core/components/core/list/list-item.tsx
    parentRef,
    disableLink = false,
````
````new web/apps/web/core/components/core/list/list-item.tsx
    parentRef,
````
````old web/apps/web/core/components/core/list/list-item.tsx
    quickActionElement,
    itemClassName = "",
    preventDefaultProgress = false,
````
````new web/apps/web/core/components/core/list/list-item.tsx
    quickActionElement,
````
````old web/apps/web/core/components/core/list/list-item.tsx
        <div className={cn("relative flex w-full items-center justify-between gap-3 truncate", itemClassName)}>
          <ControlLink
            id={id}
````
````new web/apps/web/core/components/core/list/list-item.tsx
        <div className="relative flex w-full items-center justify-between gap-3 truncate">
          <ControlLink
````
````old web/apps/web/core/components/core/list/list-item.tsx
            onClick={handleControlLinkClick}
            disabled={disableLink}
            data-prevent-progress={preventDefaultProgress}
````
````new web/apps/web/core/components/core/list/list-item.tsx
            onClick={handleControlLinkClick}
````
````old web/apps/web/core/components/core/list/list-item.tsx
          {quickActionElement && quickActionElement}
````
````new web/apps/web/core/components/core/list/list-item.tsx
          {quickActionElement}
````

`web/apps/web/core/services/favorite/favorite.service.ts`（修改，1 处）：

````old web/apps/web/core/services/favorite/favorite.service.ts

  async getFavorites(workspaceSlug: string): Promise<IFavorite[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/user-favorites/`, {
      params: {
        all: true,
      },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/favorite/favorite.service.ts

````

`web/apps/web/core/services/view.service.ts`（修改，1 处）：

````old web/apps/web/core/services/view.service.ts
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/${viewId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getViews(workspaceSlug: string, projectId: string): Promise<IProjectView[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/`)
````
````new web/apps/web/core/services/view.service.ts
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/${viewId}/`)
````

`web/apps/web/core/store/favorite.store.ts`（修改，3 处）：

````old web/apps/web/core/store/favorite.store.ts
  groupedFavorites: { [favoriteId: string]: IFavorite };
  // actions
  fetchFavorite: (workspaceSlug: string) => Promise<IFavorite[]>;
````
````new web/apps/web/core/store/favorite.store.ts
  groupedFavorites: { [favoriteId: string]: IFavorite };
````
````old web/apps/web/core/store/favorite.store.ts
      groupedFavorites: computed,
      // action
      fetchFavorite: action,
````
````new web/apps/web/core/store/favorite.store.ts
      groupedFavorites: computed,
````
````old web/apps/web/core/store/favorite.store.ts
  };

  /**
   * get Workspace favorite using workspace slug
   * @param workspaceSlug
   * @returns Promise<IFavorite[]>
   *
   */
  fetchFavorite = async (workspaceSlug: string) => {
    try {
      const favorites = await this.favoriteService.getFavorites(workspaceSlug);
      runInAction(() => {
        favorites.forEach((favorite) => {
          set(this.favoriteMap, [favorite.id], favorite);
          this.favoriteIds.push(favorite.id);
          if (favorite.entity_identifier) {
            set(this.entityMap, [favorite.entity_identifier], favorite);
          }
        });
      });
      return favorites;
    } catch (error) {
      console.error("Failed to fetch favorites from workspace store");
      throw error;
    }
  };
````
````new web/apps/web/core/store/favorite.store.ts
  };
````

`web/apps/web/core/store/module.store.ts`（修改，7 处）：

````old web/apps/web/core/store/module.store.ts
  fetchModules: (workspaceSlug: string, projectId: string) => Promise<undefined | IModule[]>;
  fetchModulesSlim: (workspaceSlug: string, projectId: string) => Promise<undefined | IModule[]>;
````
````new web/apps/web/core/store/module.store.ts
  fetchModules: (workspaceSlug: string, projectId: string) => Promise<undefined | IModule[]>;
````
````old web/apps/web/core/store/module.store.ts
  /**
   * @description fetch all modules
   * @param workspaceSlug
   * @param projectId
   * @returns IModule[]
   */
  fetchModulesSlim = async (workspaceSlug: string, projectId: string) => {
    try {
      this.loader = true;
      await this.moduleService.getWorkspaceModules(workspaceSlug).then((response) => {
        const projectModules = response.filter((module) => module.project_id === projectId);
        runInAction(() => {
          projectModules.forEach((module) => {
            set(this.moduleMap, [module.id], { ...this.moduleMap[module.id], ...module });
          });
          set(this.fetchedMap, projectId, true);
          this.loader = false;
        });
        return projectModules;
      });
    } catch {
      this.loader = false;
      return undefined;
    }
  };

  /**
````
````new web/apps/web/core/store/module.store.ts
  /**
````
````old web/apps/web/core/store/module.store.ts
    await this.moduleService.deleteModule(workspaceSlug, projectId, moduleId).then(() => {
      runInAction(() => {
        delete this.moduleMap[moduleId];
        if (this.rootStore.favorite.entityMap[moduleId]) this.rootStore.favorite.removeFavoriteFromStore(moduleId);
      });
````
````new web/apps/web/core/store/module.store.ts
    await this.moduleService.deleteModule(workspaceSlug, projectId, moduleId);
    runInAction(() => {
      delete this.moduleMap[moduleId];
      if (this.rootStore.favorite.entityMap[moduleId]) this.rootStore.favorite.removeFavoriteFromStore(moduleId);
````
````old web/apps/web/core/store/module.store.ts
    try {
      const moduleLink = await this.moduleService.createModuleLink(workspaceSlug, projectId, moduleId, data);
      runInAction(() => {
        update(this.moduleMap, [moduleId, "link_module"], (moduleLinks = []) => concat(moduleLinks, moduleLink));
      });
      return moduleLink;
    } catch (error) {
      throw error;
    }
````
````new web/apps/web/core/store/module.store.ts
    const moduleLink = await this.moduleService.createModuleLink(workspaceSlug, projectId, moduleId, data);
    runInAction(() => {
      update(this.moduleMap, [moduleId, "link_module"], (moduleLinks = []) => concat(moduleLinks, moduleLink));
    });
    return moduleLink;
````
````old web/apps/web/core/store/module.store.ts
    try {
      const moduleLink = await this.moduleService.deleteModuleLink(workspaceSlug, projectId, moduleId, linkId);
      runInAction(() => {
        update(this.moduleMap, [moduleId, "link_module"], (moduleLinks = []) =>
          moduleLinks.filter((link: ILinkDetails) => link.id !== linkId)
        );
      });
      return moduleLink;
    } catch (error) {
      throw error;
    }
````
````new web/apps/web/core/store/module.store.ts
    const moduleLink = await this.moduleService.deleteModuleLink(workspaceSlug, projectId, moduleId, linkId);
    runInAction(() => {
      update(this.moduleMap, [moduleId, "link_module"], (moduleLinks = []) =>
        moduleLinks.filter((link: ILinkDetails) => link.id !== linkId)
      );
    });
    return moduleLink;
````
````old web/apps/web/core/store/module.store.ts
    await this.moduleArchiveService
      .archiveModule(workspaceSlug, projectId, moduleId)
      .then((response) => {
        runInAction(() => {
          set(this.moduleMap, [moduleId, "archived_at"], response.archived_at);
          if (this.rootStore.favorite.entityMap[moduleId]) this.rootStore.favorite.removeFavoriteFromStore(moduleId);
        });
      })
      .catch((error) => {
        console.error("Failed to archive module in module store", error);
      });
````
````new web/apps/web/core/store/module.store.ts
    try {
      const response = await this.moduleArchiveService.archiveModule(workspaceSlug, projectId, moduleId);
      runInAction(() => {
        set(this.moduleMap, [moduleId, "archived_at"], response.archived_at);
        if (this.rootStore.favorite.entityMap[moduleId]) this.rootStore.favorite.removeFavoriteFromStore(moduleId);
      });
    } catch (error) {
      console.error("Failed to archive module in module store", error);
    }
````
````old web/apps/web/core/store/module.store.ts
    await this.moduleArchiveService
      .restoreModule(workspaceSlug, projectId, moduleId)
      .then(() => {
        runInAction(() => {
          set(this.moduleMap, [moduleId, "archived_at"], null);
        });
      })
      .catch((error) => {
        console.error("Failed to restore module in module store", error);
      });
````
````new web/apps/web/core/store/module.store.ts
    try {
      await this.moduleArchiveService.restoreModule(workspaceSlug, projectId, moduleId);
      runInAction(() => {
        set(this.moduleMap, [moduleId, "archived_at"], null);
      });
    } catch (error) {
      console.error("Failed to restore module in module store", error);
    }
````

`web/apps/web/core/store/notifications/workspace-notifications.store.ts`（修改，1 处）：

````old web/apps/web/core/store/notifications/workspace-notifications.store.ts
  setUnreadNotificationsCount: (type: "increment" | "decrement", newCount?: number) => void;
  getUnreadNotificationsCount: (workspaceSlug: string) => Promise<TUnreadNotificationsCount | undefined>;
````
````new web/apps/web/core/store/notifications/workspace-notifications.store.ts
  setUnreadNotificationsCount: (type: "increment" | "decrement", newCount?: number) => void;
````

`web/apps/web/core/store/project-view.store.ts`（修改，4 处）：

````old web/apps/web/core/store/project-view.store.ts
  // fetch actions
  fetchViews: (workspaceSlug: string, projectId: string) => Promise<undefined | IProjectView[]>;
````
````new web/apps/web/core/store/project-view.store.ts
  // fetch actions
````
````old web/apps/web/core/store/project-view.store.ts
      // fetch actions
      fetchViews: action,
````
````new web/apps/web/core/store/project-view.store.ts
      // fetch actions
````
````old web/apps/web/core/store/project-view.store.ts
  /**
   * Fetches views for current project
   * @param workspaceSlug
   * @param projectId
   * @returns Promise<IProjectView[]>
   */
  fetchViews = async (workspaceSlug: string, projectId: string) => {
    try {
      this.loader = true;
      await this.viewService.getViews(workspaceSlug, projectId).then((response) => {
        runInAction(() => {
          response.forEach((view) => {
            set(this.viewMap, [view.id], view);
          });
          set(this.fetchedMap, projectId, true);
          this.loader = false;
        });
        return response;
      });
    } catch (_error) {
      this.loader = false;
      return undefined;
    }
  };

  /**
````
````new web/apps/web/core/store/project-view.store.ts
  /**
````
````old web/apps/web/core/store/project-view.store.ts
    await this.viewService.deleteView(workspaceSlug, projectId, viewId).then(() => {
      runInAction(() => {
        delete this.viewMap[viewId];
        if (this.rootStore.favorite.entityMap[viewId]) this.rootStore.favorite.removeFavoriteFromStore(viewId);
      });
````
````new web/apps/web/core/store/project-view.store.ts
    await this.viewService.deleteView(workspaceSlug, projectId, viewId);
    runInAction(() => {
      delete this.viewMap[viewId];
      if (this.rootStore.favorite.entityMap[viewId]) this.rootStore.favorite.removeFavoriteFromStore(viewId);
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 442",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 435",
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；web 的 oxlint 435 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

完成时：本 Task 删除的 9 个文案键（`home.json` 的 `home.recents.title`、`home.recents.filters.all`、`.issues`、`.projects`、`home.recents.empty.default`、`.issue`、`.project`，`common.json` 的 `show_all`、`show_less`）在 `en`、`zh-CN` 中都已不在（spec 附录 A.7 的键表；`check:sync` 只核对两种语言一致，两种语言都留下的键要评审照键表核对）。

- [ ] **Step 5: 提交**

```bash
git add web/apps/web/core/components/core/content-overflow-HOC.tsx web/apps/web/core/components/core/list/list-item.tsx web/apps/web/core/components/home/home-body.tsx web/apps/web/core/components/home/widgets/empty-states/index.ts web/apps/web/core/components/home/widgets/empty-states/recents.tsx web/apps/web/core/components/home/widgets/index.ts web/apps/web/core/components/home/widgets/loaders/index.ts web/apps/web/core/components/home/widgets/loaders/recent-activity.tsx web/apps/web/core/components/home/widgets/recents/filters.tsx web/apps/web/core/components/home/widgets/recents/index.tsx web/apps/web/core/components/home/widgets/recents/issue.tsx web/apps/web/core/components/home/widgets/recents/project.tsx web/apps/web/core/components/navigation/top-navigation-root.tsx web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/favorite/favorite.service.ts web/apps/web/core/services/view.service.ts web/apps/web/core/services/workspace.service.ts web/apps/web/core/store/favorite.store.ts web/apps/web/core/store/module.store.ts web/apps/web/core/store/notifications/workspace-notifications.store.ts web/apps/web/core/store/project-view.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/home.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/home.json web/packages/types/src/home.ts web/packages/types/src/index.ts
```
```bash
git commit -m "feat(M3/P8a): no page M3 reaches fetches M6's or M7's data as it mounts

The workspace wrapper stops fetching the favourites, the top navigation
the unread count, and the project wrapper the cycles, modules, views and
the intake's state; the home page loses its recent-visits widget, until
M6 and M7 bring them back with their APIs. What these deletions leave
without a caller goes too, as the design does with the favourite
project ids, and the files that lose it end with no oxlint warning.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p8a.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫和 i18n 的 `check:sync`，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t11-cycles-back` | 项目包装层又取项目的迭代（M6 的，每个项目页都取） | 本 Phase 没有（P8b 改写的 S2：每张挂载清单上都没有 M6、M7 的地址） | 存活 |
| `t11-unread-back` | 顶部导航又取未读通知数（M7 的，每一页都取），通知 store 的接口加回这个方法 | 本 Phase 没有（P8b 改写的 S2：每张挂载清单上都没有 M6、M7 的地址） | 存活 |

---

### Task 12: 3.20 中 P8a 的行：总体设计 7.7、前端改动清单

**Files:**
- Modify: `docs/v0/frontend-changes.md`、`docs/v0/v0-design.md`

**Interfaces:**
- 总体设计 7.7（3.20 中 P8a 的行）：SWR 键一条改写为原则（裁定 A6）：会话的每个取数都经 `useSessionSWR`，键只有 `sessionKey` 一种写法 `[取数的名称, loginId, ...取数的参数]`（fetcher 收到参数、返回 store 的 Promise）；唯一的例外是经 `publicClient` 的公开操作，键只由它的输入组成（`useInvitationPreview`）；静态检查是根目录 `.oxlintrc.json` 的 `no-restricted-imports`（Task 8），之后的 M 把接上新接口的取数文件加进它的范围。加一条"页面按权限决定取数，不只决定显示"（7.1）；释放一条改写为 `RootStore.dispose()` 和 `store-context.tsx` 的调用；`inSession()` 的引用改为 `core/lib/in-session.ts` 的 `sessionGuard()`；加一条 problem 码的文案表：`PROBLEM_MESSAGES` 在 `core/lib/error-messages.ts`，新码与两份 `errors.json` 同一个提交加入（M3 设计第 12 节约束 4，Task 2）；代码列表加 `session-key.ts`、`use-session-swr.ts`、`in-session.ts`。
- 前端改动清单（3.20 中 P8a–P11 的行在 P8a 的部分）：3.1 中 M3 一行改为"进行中"（工作区一侧在 P8a，项目一侧在 P8b）；错误格式一行的文案位置改为 `core/lib/error-messages.ts`；3.2 加五行：系统内接受、slug 检查与 `RESTRICTED_URLS`、用户设置和落点（`useLanding`、`landingPath`）、挂载时的取数（M6、M7 加回）、工作区图标的上传（M5 加回）。

**Tests:** 没有代码改动。每一句与它描述的代码或测试核对过（扫描 50，spec 附录 A）。

- [ ] **Step 1: 文档**

`docs/v0/frontend-changes.md`（修改，3 处）：

````old docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 计划中 |
````
````new docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：工作区、成员、邀请、工作区的显示设置已对接（M3/P8a）；项目一侧在 M3/P8b |
````
````old docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`helpers/authentication.helper.ts` 的 `fieldErrorKeys`、`errorMessageKey`）。M3–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````new docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````old docs/v0/frontend-changes.md
| 迭代和模块的归属 | 通过工作项的 `cycle_id`、`module_ids` 字段修改，不再调用单独的接口 | 接口设计 | 计划中 | |
````
````new docs/v0/frontend-changes.md
| 迭代和模块的归属 | 通过工作项的 `cycle_id`、`module_ids` 字段修改，不再调用单独的接口 | 接口设计 | 计划中 | |
| 系统内接受邀请 | `/invitations` 页、它的入口和路由、新手引导的"加入工作区"一步、按邮箱列出和批量接受的 service、store 方法删除；邀请只凭链接接受（`/workspace-invitations`） | [M3 设计](M3-workspace-project/M3-design.md) 决策点 2 | 已完成 | M3/P8a |
| 创建工作区的 slug 检查 | `RESTRICTED_URLS` 删除，创建表单改问 nerve（`GET /api/v0/workspace-slugs/{slug}`）；保留名单"应用"一段由 vitest 核对与路由一致 | M3 设计 3.10 | 已完成 | M3/P8a |
| 用户设置和落点 | `settings.store.ts`、`IUserSettings`、`currentUserSettings` 删除；登录后的落点由 `useLanding`（`core/lib/use-landing.ts`）决定，落到哪个工作区由纯函数按工作区列表算出（`core/lib/landing.ts`） | M3 设计 3.14 | 已完成 | M3/P8a |
| 挂载时的取数 | 工作区包装层取收藏、顶部导航取未读通知数、首页的"最近"小部件、项目包装层取迭代、模块、视图、分诊状态，全部删除，各自的 M 随新接口加回 | M3 设计 3.1 | 已完成（M6、M7 加回） | M3/P8a |
| 工作区图标的上传 | 工作区设置 general 页的图标上传弹窗删除：接口不能设置 `logo_url`，它恒为 `null`，显示图标的地方照旧显示首字母；M5 随文件的接口加回 | M3 设计 3.2 | 已完成（M5 加回） | M3/P8a |
````

`docs/v0/v0-design.md`（修改，3 处）：

````old docs/v0/v0-design.md
- **填充 stores 的 SWR 键带上 `loginId`**（例如 `["CURRENT_USER", loginId]`）：换了会话，新的一代取自己的数据。
- **一代退役时释放它的反应**：注册在跨代沿用的对象（`router` 等）上的 `reaction`、`autorun` 不会随旧的一代回收。M2 结束时 `RootStore` 还没有释放的方法：第一个接上这类 store 的 M 给 `RootStore` 加上它，由 `store-context.tsx` 在换代时调用，之后的 M 照做（要释放的反应列在各 M 的交接里）。只观察本代对象的反应不用释放。
````
````new docs/v0/v0-design.md
- **填充 stores 的 SWR 键带上 `loginId`**：会话的每个取数都经 `useSessionSWR`（`core/lib/use-session-swr.ts`），键只有 `sessionKey`（`core/lib/session-key.ts`）一种写法：`[取数的名称, loginId, ...取数的参数]`（例如 `["WORKSPACE_MEMBERS", loginId, slug]`）；没有会话或有参数未知时是 `null`，不取。换了会话，新的一代取自己的数据。fetcher 收到键的参数，返回 store 的 Promise：取数失败是 SWR 的 `error`，不是未处理的拒绝（M3/P8a）。唯一的例外是不属于会话的公开操作：经 `publicClient` 调用，键只由它的输入组成（查看邀请的链接，`core/hooks/use-invitation-preview.ts`，M3 设计 7.1）。根目录 `.oxlintrc.json` 的 `overrides` 静态地看住这一条：列出的文件不能从 `swr` 导入值，只能导入类型（M3/P8a 起是工作区一侧的 store、取数的 hook、`use-landing.ts` 和 `authentication-wrapper.tsx`）；之后的 M 把接上新接口的取数文件加进去（M3/P8b 加上工作区和项目的包装层）。
- **页面按权限决定取数，不只决定显示**：页面只请求调用者有权读的资源；按权限隐藏的区域，它的取数同样按权限启用（没有权限时 `useSessionSWR` 的取数是 `null`）。例如工作区设置的成员页只在调用者是工作区的管理员时取邀请列表（M3 设计 7.1）。
- **一代退役时释放它的反应**：注册在跨代沿用的对象（`router` 等）上的 `reaction`、`autorun` 不会随旧的一代回收。`RootStore.dispose()` 释放它们，`store-context.tsx` 在换代时对退役的一代调用它（M3/P8a 起：`project_filter` 跟随 `router` 的反应）；之后注册这类反应的 store 把它的释放加进 `dispose()`。只观察本代对象的反应不用释放。
````
````old docs/v0/v0-design.md
- **页面级的状态跟随当前的会话**：界面语言由 `StoreWrapper`（`core/lib/wrappers/store-wrapper.tsx`）按当前一代的资料设置，store 不设。主题同样由 `StoreWrapper` 按会话设置，别处不设：没有会话时（本标签页或别的标签页退出、账户停用、续期被拒）是默认的"跟随系统"，每个会话第一次取到资料时按资料设置一次；之后由改它的组件在 nerve 应答成功之后设置，设置之前先核对标签页仍在发出修改时的会话（`core/components/appearance/theme-switcher.tsx` 的 `inSession()`，M2/P5 spec 2.7）。store 不直接改全局状态；组件在一个 `await` 之后改页面级的状态时，都先核对会话没变。这样旧一代迟到的应答就改不到新会话的页面（M2/P5 spec 第 3 节第 16 条）。
````
````new docs/v0/v0-design.md
- **页面级的状态跟随当前的会话**：界面语言由 `StoreWrapper`（`core/lib/wrappers/store-wrapper.tsx`）按当前一代的资料设置，store 不设。主题同样由 `StoreWrapper` 按会话设置，别处不设：没有会话时（本标签页或别的标签页退出、账户停用、续期被拒）是默认的"跟随系统"，每个会话第一次取到资料时按资料设置一次；之后由改它的组件在 nerve 应答成功之后设置，设置之前先核对标签页仍在发出修改时的会话（`core/lib/in-session.ts` 的 `sessionGuard()`：发出修改时取得核对，应答之后调用；M2/P5 spec 2.7 中 `theme-switcher.tsx` 自己的 `inSession()` 在 M3/P8a 移到这里）。store 不直接改全局状态；组件在一个 `await` 之后改页面级的状态时，都先核对会话没变。这样旧一代迟到的应答就改不到新会话的页面（M2/P5 spec 第 3 节第 16 条）。
````
````old docs/v0/v0-design.md
- 代码在 `web/apps/web/core/lib/store-context.tsx`、`core/store/root.store.ts`、`core/lib/auth/api-client.ts`、`core/lib/auth/auth-middleware.ts`（`SessionChangedError`）和 `core/lib/wrappers/store-wrapper.tsx`；照这些规则写的 store 可以看 `core/store/user/api-token.store.ts`；测试和变异见 M2/P4 spec 2.8、M2/P5 spec 2.4。
````
````new docs/v0/v0-design.md
- **nerve 的 problem 码的文案只有一张表**：`PROBLEM_MESSAGES`（`core/lib/error-messages.ts`，M3/P8a 起）把每个码对到 `errors` 命名空间的文案键，`en`、`zh-CN` 两份 `errors.json` 各有这条文案。接口新声明一个码（3.5 的 `x-problem-codes`）时，同一个提交把它加进这三处：`error-messages.test.ts` 对照 `api/dist/openapi.yaml` 核对表中的码不缺不多、每个文案键都在，i18n 的 `check:sync` 核对两种语言的键相同（M3 设计第 12 节约束 4）。
- 代码在 `web/apps/web/core/lib/store-context.tsx`、`core/store/root.store.ts`、`core/lib/session-key.ts`、`core/lib/use-session-swr.ts`、`core/lib/in-session.ts`、`core/lib/auth/api-client.ts`、`core/lib/auth/auth-middleware.ts`（`SessionChangedError`）和 `core/lib/wrappers/store-wrapper.tsx`；照这些规则写的 store 可以看 `core/store/user/api-token.store.ts`；测试和变异见 M2/P4 spec 2.8、M2/P5 spec 2.4。
````

- [ ] **Step 2: 运行检查**

Run: `make lint-web`
Expected: 通过（关键词守卫读 `docs/` 的规则照旧没有命中）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 3: 提交**

```bash
git add docs/v0/frontend-changes.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M3/P8a): the overall design's session rules and the front-end change list follow P8a

Overall design 7.7 says every fetch of the session goes through
useSessionSWR, keyed one way, with the public preview as its one
exception and the root oxlint config as its check; adds that a page's
permissions decide what it fetches and where the problem codes'
messages live; and points at RootStore.dispose() and sessionGuard(). The
front-end change list marks
M3 in progress and records the in-app accept, the reserved slugs, the
user settings and landing, the mount-time fetches and the workspace
icon's upload.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有变异（它只改文档）。
