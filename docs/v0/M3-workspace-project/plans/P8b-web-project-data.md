# M3/P8b 项目一侧的数据层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 项目一侧的类型、service、store（项目的两份列表和详情、项目成员、状态、标签、项目的显示设置、权限 store 的项目一半）迁到 `/api/v0`，按会话分代，修改一个接一个、写入 nerve 的回答，取数与修改经 `reconciled.ts` 调和。两个包装层项目一侧的挂载时取数改经 `useSessionSWR`、按权限启用：工作区包装层在调用者的列表有这个工作区之后取项目列表和工作区的状态；项目包装层先取项目，nerve 说调用者是它的有效成员（`member_role` 不为 `null`）之后才取显示设置、标签、成员和状态。生成的 `Project` 没有的（封面的上传、项目的收藏、`intake_count`）和 M7 的分诊状态、工作区级的标签从页面删除。使用方改到能编译、行为不变（spec 第 3 节列出的除外）。关键词规则完整（7.10）；S2 改写为四个账户登录之后的挂载清单。

**Architecture:** 只改 web（另有 `tools/keywords.json`、`.oxlintrc.json`、S2 和两份上级文档）。`core/services/project/`：`projects`、`project-preferences`、`project-members`、`states`、`labels` 五个 service，每个都是生成的客户端之上的薄封装，由 store 用这一代的 `ApiClient` 构造；旧 `ProjectService` 只留 M4、M6、M7 的三个方法（7.3）。store：`ProjectStore`（重写：未归档、已归档两份列表按工作区的 id，详情按项目的 id，三者都是 `ReconciledByKey`；加入、离开、侧边栏的顺序也在这里）、`ProjectPreferencesStore`（新：调用者在项目里的标签栏）、`ProjectMemberStore`（重写）、`StateStore`（重写：按项目和按工作区两份）、`LabelStore`（重写：按项目，两层由 `parent_id` 算出，移动的位置在轮到它发出时才算）、`UserPermissionStore` 的项目一半（读 `Project.member_role`）。`reconciled.ts` 加 `ReconciledByKey.values()`（扩展，不复制）。挂载时的取数：`useWorkspaceFetch`（工作区包装层，P8a 的 hook 加两个取数）和新的 `useProjectFetch`（项目包装层；它也做出包装层的全部判断：`kind` 是 `unavailable`、`loading`、`not-found`、`not-member`、`member` 之一）。测试的共用部分：`core/store/project/fake-projects.ts`（`projectOf`、`loadProjects`、`loadArchivedProjects`），`fake-store-hooks.ts` 加项目一侧的 hook。静态检查：根目录 `.oxlintrc.json` 的两组 `overrides`（P8a 的 A6 和非空断言）加上 P8b 的文件，两个包装层都在其中。契约不变；不加 npm 包，`@nerve/propel` 加工作区内的依赖 `@nerve/api-client`（Task 2，锁文件一处）。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、openapi-fetch 0.17.0、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1（Go 代码不变）。版本都由 `pnpm-lock.yaml` 固定。

**Spec:** `docs/v0/M3-workspace-project/specs/P8b-web-project-data.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 包或 Go 模块，不执行 `go get`、`pnpm add`。`pnpm-lock.yaml` 只在 Task 2 改一处（`@nerve/propel` 依赖工作区内的 `@nerve/api-client`：项目的图标是生成的 `LogoProps`），之后执行 `pnpm install --frozen-lockfile`。`git diff --stat 7390e012 -- server/ api/` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-web`（关键词守卫、`tsc`、oxlint 在上限、格式、`en` 与 `zh-CN` 的键一致）、`make knip`、`make test-web`、`make e2e` 都通过。本 plan 不改 Go 代码和接口描述，不生成任何文件，不执行 `make gen`。
- **e2e 的已知失败**：只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（P4b spec F4）；在 worktree 中 `make e2e` 必须全部通过（Task 1–10 是 70 个，Task 11 的 S2 加 4 个之后是 74 个）。
- **容器**：`make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**（总体设计 7.7，P8a 的裁定 F-1–F-10）：store 每个资源一个，按会话分代，由 `RootStore` 用这一代的 `ApiClient` 建；没有模块级的 service 实例（M2 交接第 3 节的 `git grep` 中 M3 的两处在本 Phase 消失）；修改经 `oneAtATime()` 排队，取数不排队；修改不乐观，store 写入 nerve 的回答；取数和修改的应答只经 `core/lib/reconciled.ts` 对齐（扩展它，不复制）；每个工作区、每个项目的状态按 id 存，项目一侧的 store 只给出项目 store 仍给出的项目的东西（离开、删除的项目，和不再是调用者的工作区里的项目，什么都不给）；取数遇到 `SessionChangedError` 给出 `undefined`、不改本代的状态，它不是认证失败。会话的每个取数都经 `useSessionSWR`（键 `[取数的名称, loginId, ...取数的参数]`，不传配置），没有权限时键为 `null`，权限的条件由取数的 hook 自己从 store 算出、不由调用方传入。这一条有静态检查：根目录 `.oxlintrc.json` 的 `no-restricted-imports` 不让范围内的文件从 `swr` 导入值（P8b 的 store、两个包装层、`use-project-fetch.ts` 和读它们的设置组件，spec 附录 A.6）。类型只来自生成的客户端：不写重述契约的类型，不留旧名字的别名，不写把新接口映射成 Plane 形状的适配层。删除的代码删干净（store 和 service 的方法、组件、两种语言的文案、常量、键、它们的测试），不加 knip 的忽略、开关或桩。没有新的 `as`、`any`、`!`（spec 附录 A.7 的 W12），P8b 的 M3 路径上 `typescript/no-non-null-assertion` 是错误（同一组 `overrides`）。本 plan 写或重写的文件都在约 400 行以内（最终原型上量的，spec 附录 A.10：最长的是 `core/store/project/project.store.test.ts` 422 行、`core/store/label.store.test.ts` 362 行、`core/store/project/project.store.ts` 312 行）；只为使用方改到的 Plane 文件不拆，也不变长，例外三个，原因见 spec 附录 A.10：`issues/issue-layouts/utils.tsx` 808 → 812（状态图标读组中的位置，调用比 `state.order` 长，格式化把一行拆成五行）、`power-k/config/navigation/commands.ts` 465 → 466（两个不读闭包的条件移出 hook，清掉 `consistent-function-scoping`，多一个空行）、只经机械步骤改到的 `power-k/ui/pages/context-based/work-item/commands.ts` 414 → 415（`rename_types.py` 把 `Label` 另起一行导入）。
- **oxlint**（M3 设计 7.9，裁定 R3）：有手改的文件（每个有块的 TS 文件）在它的 Task 提交时没有 oxlint 警告；只经本 plan 的机械步骤改到的文件（Task 2、8、9 的 Step 1 的文件中没有块的那些）不在此列，它们的警告留给 P11 的第 4 个任务，spec 附录 A.8 列出它们。没有新的 oxlint 抑制。上限在改变条数的 Task 里调到新的条数：web 435 → 421（Task 1）→ 414（Task 2）→ 411（Task 3）→ 409（Task 4）→ 408（Task 5）→ 406（Task 6）→ 370（Task 7）→ 367（Task 8）；`utils` 12 → 10（Task 2）→ 9（Task 8）→ 7（Task 9）；`types` 0、`constants` 1、`propel` 16、`ui` 19 不变。
- **注释**：TS 代码、测试、JSON 的说明用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的；二进制的图片也这样删除）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `7390e012` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A.12）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **机械步骤**（brief 的规则）：Task 2、8、9 的 Step 1 只改类型名和它的导入，由命令对写明的文件列表执行，不写成块：`Run（机械步骤）:` 的命令照原样从仓库根执行，它们用的一次性脚本在下面"一次性脚本"一节全文给出，写到 `$P8BTMP`（实现者自己的临时目录，在仓库之外；每条命令之前 `export P8BTMP=<那个目录>`）。步骤之后 `git diff --numstat` 必须恰好是步骤给出的那几行，`shasum -a 256` 的结果与表中的散列和行数相同；对不上时停下来：说明某个输入与本 plan 不一致。各 Task 的块写在机械步骤之后的文件上（有几个文件既经机械步骤改名、又有手改）。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：`core/store/project/project.store.ts` 和它的测试（Task 3 列表、详情和修改，Task 4 加入、离开，Task 5 标识检查，Task 6 `confirmProject`）、`core/store/user/permissions.store.ts`（Task 3、4）、`core/layouts/auth-layout/project-wrapper.tsx`（Task 3–9 随各自的 store 改调，Task 10 交给 `useProjectFetch`）、`use-workspace-fetch.ts` 和它的测试（Task 3 项目，Task 8 状态）、`core/store/member/project/project-member.store.ts`（Task 2–5 跟随项目的类型和 store，Task 6 重写）、`core/hooks/store/fake-store-hooks.ts`（Task 3、8、10）、`core/services/project/projects.service.ts`（Task 3、4、5）、`packages/constants/src/fetch-keys.ts`、`packages/types/src/project/projects.ts`、`.oxlintrc.json`、`web/apps/web/package.json` 的上限，以及 spec 第 2 节逐个列出的使用方。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（静态：`make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，和 `make knip`；vitest：`make test-web`；端到端：`make e2e` 的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p8btools/mutants_p8b.py`，`mut.py` 在它写的每个检查上各跑一次；spec 附录 A.2）；一个变异列在它守的性质出现的 Task：改 P8a 已有的代码的几个（`reconciled.ts` 的，和读回生成的 `Project` 没有的字段的）列在第一个依赖这个性质的 Task；只从之后的 Task 起才失败的检查，表中标"（之后的 Task 起）"（只有 S2，它在 Task 11 改写）。读 `@nerve/utils` 的构建产物的两个先构建那个包。**会话、权限或取数的性质只由评审才能发现的，算缺口**（brief 的缺陷类别）：表中每一条这类性质都有一个会失败的检查。
- **评审敏感**（M3 设计 12 节 P8b 的评审重点）：不是有效的项目成员（`member_role` 为 `null`）时不取项目的子资源（Task 10 的 `useProjectFetch`，条件由 hook 从 nerve 对项目的回答和项目 store 算出）；权限 store 的项目角色与 3.4 的服务端规则一致（Task 4，与 9.2 同一组身份，含 PM+WA、WA-）；拖动排序一个接一个发出（状态 Task 8、标签 Task 9、侧边栏的项目 Task 3，标签的位置在轮到它发出时才算）。改动这些取数的条件、键或 store 的会话处理之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性脚本

Task 2、8、9 的机械步骤用下面的脚本，照原样写到 `$P8BTMP`（不提交）。

`$P8BTMP/rename_types.py`（Task 2、8、9：把一个或几个 Plane 类型换成一个生成的类型）：

```python
# -*- coding: utf-8 -*-
# M3/P8b's mechanical step for type names (the plan gives this script whole): in each file named, the Plane types
# <old>... (comma-separated), imported from @nerve/types (or, with --from, from that module: the types package's own
# files import them by a relative path), become the one generated type <new> of @nerve/api-client. Every whole-word
# use of an old name, outside strings and comments as well as inside them, becomes <new>; the old names leave the
# file's `import type { … } from "<module>";` (the line goes when nothing is left in it), and <new> joins the file's
# `import type { … } from "@nerve/api-client";`, or such a line is added where the old import was. It checks first
# that each file imports at least one old name from the module, in one `import type` line, and neither imports nor
# declares <new> already, and changes nothing unless every file passes.
# usage: python3 rename_types.py <root> <old>[,<old>...] <new> [--from <module>] <file>...   (files relative to <root>)
import re
import sys

root, olds, new, rest = sys.argv[1], sys.argv[2].split(','), sys.argv[3], sys.argv[4:]
module = '@nerve/types'
if rest[:1] == ['--from']:
    module, rest = rest[1], rest[2:]
files = rest
# an `import type { … }` on one line, or over several (one name a line, as oxfmt writes a long one); the step writes
# it back on one line, and where that line is too long, oxfmt lays it out again (Task 9's step runs it after this
# script; the files of Task 2's and Task 8's steps need it nowhere)
TYPES = re.compile(r'^import type \{\s*([^}]*?)\s*\} from "' + re.escape(module) + r'";\n', re.M)
CLIENT = re.compile(r'^import type \{\s*([^}]*?)\s*\} from "@nerve/api-client";\n', re.M)
words = re.compile(r'\b(?:' + '|'.join(re.escape(o) for o in olds) + r')\b')
taken = re.compile(r'^import [^;]*\b' + re.escape(new) + r'\b[^;]*;|^(?:export )?(?:const|let|function|class|type|interface|enum) '
                   + re.escape(new) + r'\b', re.M)


def names(group):
    return [n.strip() for n in group.split(',') if n.strip()]


results = {}
for rel in files:
    path = f'{root}/{rel}'
    text = open(path, encoding='utf-8').read()
    found = [m for m in TYPES.finditer(text) if set(olds) & set(names(m.group(1)))]
    if len(found) != 1:
        sys.exit(f'{rel}: imports {olds} from {module} in {len(found)} import type lines')
    if taken.search(text):
        sys.exit(f'{rel}: imports or declares {new} already')
    line = found[0]
    others = [n for n in names(line.group(1)) if n not in olds]
    client = CLIENT.search(text)
    replacement = f'import type {{ {", ".join(others)} }} from "{module}";\n' if others else ''
    if client is None:
        replacement = f'import type {{ {new} }} from "@nerve/api-client";\n' + replacement
    text = text[:line.start()] + replacement + text[line.end():]
    if client is not None:
        client = CLIENT.search(text)
        listed = names(client.group(1)) + [new]
        text = text[:client.start()] + f'import type {{ {", ".join(listed)} }} from "@nerve/api-client";\n' + text[client.end():]
    results[path] = words.sub(new, text)
for path, text in results.items():
    open(path, 'w', encoding='utf-8').write(text)
print(f'{",".join(olds)} -> {new} in {len(results)} files')
```

## 文件结构

路径相对于仓库根目录；"web/"之下的应用写作 `web/apps/web/…`。一个文件由多个 Task 修改时，"Task"一列都列出；"机械"表示只经那个 Task 的机械步骤改到。

| 文件 | 职责 | Task |
|---|---|---|
| `web/apps/web/core/components/core/image-picker-popover.tsx`、`web/apps/web/app/assets/cover-images/image_2`–`image_29` 的 `.svg` 和 `.webp`（56 个） | 封面的选择器和 28 张预设封面（删除） | 1 |
| `web/apps/web/app/assets/cover-images/SOURCES.md`、`web/apps/web/helpers/cover-image.helper.ts`、`web/packages/constants/src/file.ts`、`web/packages/constants/src/tab-indices.ts`、`web/packages/types/src/enums.ts`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json` | 只留默认封面（整个文件）；封面的上传常量、标签页序号、资源类型和 `change_cover` 删除 | 1 |
| `web/apps/web/core/components/issues/issue-layouts/list/block.tsx` | 工作项的键宽不读 `next_work_item_sequence` | 1 |
| `web/apps/web/core/store/favorite.store.ts`、`web/apps/web/core/store/inbox/inbox-issue.store.ts`、`web/apps/web/core/store/inbox/project-inbox.store.ts` | 收藏 store 的项目一支、收集箱 store 改写 `intake_count` 的两段删除 | 1 |
| `web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/create-project-modal.tsx`、`web/apps/web/core/components/projects/create/root.tsx`、`web/apps/web/core/components/projects/create/utils.ts` | 封面只显示、收藏和创建时的封面删除（Task 1）；`Project`、`ProjectCreate`（Task 2） | 1、2 |
| `web/apps/web/core/components/project/create/header.tsx` | 封面的选择删除（Task 1）；`ProjectCreate`（Task 2 机械） | 1、2 机械 |
| `web/apps/web/core/components/workspace/sidebar/project-navigation.tsx` | 收集箱的数字删除（Task 1）；`Project`（Task 2） | 1、2 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx` | `setToFavorite` 删除、清零（Task 1）；`Project`（Task 2）；侧边栏的顺序改调新 store（Task 3） | 1、2、3 |
| `web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx` | 注释掉的收藏一段删除（Task 1）；`Project`（Task 2）；标签栏只传项目的 id（Task 5） | 1、2、5 |
| `web/apps/web/core/components/project/form.tsx` | 封面的选择删除（Task 1）；`Project`（Task 2）；新 store（Task 3）；标识检查改问 nerve，模块级的 `ProjectService` 删除（Task 5） | 1、2、3、5 |
| `web/apps/web/package.json` | web 的 oxlint 上限 | 1–8 |
| `pnpm-lock.yaml`、`web/packages/propel/package.json`、`web/packages/propel/src/emoji-icon-picker/logo.tsx`、`web/apps/web/core/components/common/switcher-label.tsx` | `@nerve/propel` 依赖 `@nerve/api-client`；项目的图标是生成的 `LogoProps` | 2 |
| `web/packages/utils/src/project.ts` | 项目的筛选和排序读生成的字段名，倒序用 `toReversed()`（Task 2 手改；Step 1 的改名在同一个 Task） | 2 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`、`web/apps/web/core/components/projects/settings/intake/header.tsx`、`web/apps/web/core/components/inbox/content/inbox-issue-header.tsx`、`web/apps/web/core/components/breadcrumbs/project.tsx`、`web/apps/web/core/components/navigation/use-navigation-items.ts`、`web/apps/web/core/components/power-k/config/creation/command.ts`、`web/apps/web/core/components/power-k/config/navigation/commands.ts`、`web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx`、`web/apps/web/core/components/projects/create/attributes.tsx`、`web/apps/web/core/components/settings/project/sidebar/header.tsx`、`web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx` | `Project` 的使用方：生成的字段名（`intake_view` 等）、`getProjectById` 代替 `getPartialProjectById` | 2 |
| `web/apps/web/core/components/archives/archive-tabs-list.tsx`、`web/apps/web/core/components/automation/auto-archive-automation.tsx`、`web/apps/web/core/components/automation/select-month-modal.tsx`、`web/apps/web/core/components/dropdowns/project/base.tsx`、`web/apps/web/core/components/issues/filters.tsx`、`web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx`、`web/apps/web/core/components/navigation/project-header-button.tsx`、`web/apps/web/core/components/power-k/menus/projects.tsx`、`web/apps/web/core/components/project/create/common-attributes.tsx`、`web/apps/web/core/components/project/create/project-create-buttons.tsx`、`web/packages/types/src/search.ts`、`web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts` | Plane 的项目类型 → `Project`、`ProjectCreate` | 2 机械 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/core/components/project/delete-project-modal.tsx`、`web/apps/web/core/components/settings/project/content/feature-control-item.tsx` | `Project`（Task 2 机械）；新 store 的签名（Task 3） | 2 机械、3 |
| `web/apps/web/core/components/project/join-project-modal.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx` | `Project`（Task 2 机械）；加入、离开改调项目 store（Task 4） | 2 机械、4 |
| `web/apps/web/core/services/project/project.service.ts` | `Project`（Task 2 机械）；只留 M4、M6、M7 的三个方法（Task 5，整个文件） | 2 机械、5 |
| `web/apps/web/core/store/issue/root.store.ts` | 类型的改名（Task 2、8、9 机械）；`projectMap` 删除（Task 3） | 2 机械、3、8 机械、9 机械 |
| `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/packages/utils/src/work-item/modal.ts` | 类型的改名 | 2 机械、8 机械（`use-work-item-filters-config.tsx` 另有 9 机械） |
| `web/apps/web/core/components/project/settings/features-list.tsx` | `Project`（Task 2）；新 store 的签名（Task 3） | 2、3 |
| `web/apps/web/core/components/project/project-settings-member-defaults.tsx` | 只改两个字段的表单（Task 2）；新 store（Task 3）；不再自己取项目（Task 10） | 2、3、10 |
| `web/apps/web/core/store/project/project.store.ts` | Plane 的收藏、序号、`getPartialProjectById` 删除（Task 2）；重写（Task 3，整个文件）；加入、离开（Task 4）；标识检查（Task 5）；`confirmProject`（Task 6） | 2、3、4、5、6 |
| `web/apps/web/core/store/member/project/project-member.store.ts` | 跟随项目的类型和 store（Task 2–5）；重写（Task 6，整个文件） | 2、3、4、5、6 |
| `web/apps/web/core/components/navigation/project-header.tsx`、`web/apps/web/core/components/navigation/tab-navigation-root.tsx` | `getProjectById`（Task 2）；标签栏读新 store（Task 5） | 2、5 |
| `web/packages/types/src/project/projects.ts` | 只留 M3 之外的类型（Task 2，整个文件）；`IProjectMemberNavigationPreferences`（Task 5）、`TProjectMembership`、`IProjectBulkAddFormData`（Task 6）删除；`StateGroup`（Task 8 机械） | 2、5、6、8 机械 |
| `web/apps/web/core/services/project/projects.service.ts` | 项目：列出、创建、读、修改、删除、归档、恢复（Task 3）；加入、离开（Task 4）；标识检查（Task 5） | 3、4、5 |
| `web/apps/web/core/services/project/project-preferences.service.ts` | 调用者在项目里的设置：修改（Task 3，侧边栏的顺序）；读（Task 5） | 3、5 |
| `web/apps/web/core/services/project/project-archive.service.ts` | 旧的归档 service（删除） | 3 |
| `web/apps/web/core/store/project/project.store.test.ts`、`web/apps/web/core/store/project/fake-projects.ts` | 项目 store 的测试（Task 3；加入、离开 Task 4；标识检查 Task 5）；`projectOf`、`loadProjects`、`loadArchivedProjects` | 3、4、5 |
| `web/apps/web/core/store/project/index.ts`、`web/apps/web/core/store/root.store.ts` | 项目 store 用这一代的客户端建（Task 3）；显示设置的子 store（Task 5，整个文件）；状态、标签 store 同样（Task 8、9） | 3、5、8、9 |
| `web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts` | 工作区包装层挂载时的取数：项目列表（Task 3）、工作区的状态（Task 8） | 3、8 |
| `web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx` | 取项目（Task 3）、项目角色（Task 4）、状态（Task 8）的 `useSWR` 删除 | 3、4、8 |
| `web/apps/web/core/layouts/auth-layout/project-wrapper.tsx` | 过渡：随各自的 store 改调（Task 3–9）；只照 `useProjectFetch` 渲染（Task 10，整个文件） | 3、4、5、6、8、9、10 |
| `web/apps/web/core/hooks/store/fake-store-hooks.ts` | hook 测试中代替项目一侧的 store hook | 3、8、10 |
| `.oxlintrc.json` | P8b 的文件加进两条 `overrides` 的范围，两个包装层都在其中 | 3、4、5、6、8、9、10 |
| `web/packages/constants/src/fetch-keys.ts` | 不再用的取数键删除 | 3、4、7、8、10 |
| `web/apps/web/core/components/project/archive-restore-modal.tsx`、`web/apps/web/core/components/project/card-list.tsx`、`web/apps/web/core/components/projects/page.tsx`、`web/apps/web/core/store/issue/archived/issue.store.ts`、`web/apps/web/core/store/issue/project/issue.store.ts`、`web/packages/types/src/issues/issue-identifier.ts` | 项目 store 的使用方；项目页取已归档的列表（`page.tsx`，整个文件） | 3 |
| `web/apps/web/core/components/issues/issue-layouts/utils.tsx` | 项目列表（Task 3）；工作区级的标签分组（Task 7）；状态图标的组中位置（Task 8） | 3、7、8 |
| `web/apps/web/core/services/project/index.ts` | service 的出口 | 3、6、8 |
| `web/apps/web/core/store/user/permissions.store.ts`、`web/apps/web/core/store/user/permissions.store.test.ts` | 过渡（Task 3）；项目一半取自 `member_role`（Task 4，整个文件）和 9.2 的测试；测试改用 `ProjectRootStore`（Task 5） | 3、4、5 |
| `web/apps/web/core/components/workspace-notifications/root.tsx`、`web/apps/web/core/services/user.service.ts`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/user/index.ts`、`web/packages/types/src/users.ts` | 旧的加入、离开、项目角色的取数和类型删除；使用方改读项目 store | 4 |
| `web/apps/web/core/components/project/member-list-item.tsx` | 离开改调项目 store（Task 4）；成员的新签名（Task 6） | 4、6 |
| `web/apps/web/core/services/project/project-member.service.ts` | `projectMemberMe` 和模块级的实例删除（Task 4）；整个旧 service 删除（Task 6） | 4、6 |
| `tools/keywords.json` | `project-invitations` 的例外删除（Task 4）；`plane-workspace-urls` 补全、`plane-user-urls` 收紧、M4 的例外（Task 11） | 4、11 |
| `web/apps/web/core/store/project/preferences.store.ts`、`web/apps/web/core/store/project/preferences.store.test.ts`、`web/apps/web/core/hooks/store/use-project-preferences.ts` | 调用者在项目里的标签栏 | 5 |
| `web/apps/web/core/components/navigation/use-tab-preferences.ts`、`web/apps/web/core/components/navigation/tab-navigation-utils.ts`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`、`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`、`web/packages/types/src/view-props.ts` | 标签栏读写新 store（`use-tab-preferences.ts` 整个文件），键是生成的 `ProjectTab`；`IProjectUserPropertiesResponse` 删除 | 5 |
| `web/apps/web/core/services/project/project-members.service.ts`、`web/apps/web/core/store/member/project/project-member.store.test.ts` | 项目成员：列出、加入、改角色、移出；store 的测试 | 6 |
| `web/apps/web/core/components/dropdowns/member/dropdown.tsx`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/member-list.tsx`、`web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`、`web/apps/web/core/store/member/index.ts`、`web/apps/web/core/store/member/project/project-member-filters.store.ts`、`web/apps/web/core/store/member/utils.ts` | 成员 store 的使用方：`ProjectMember`、`IProjectMemberDetails`、`ProjectMembersAdd` | 6 |
| `web/apps/web/core/store/issue/issue-details/sub_issues.store.ts` | 成员（Task 6）、状态（Task 8）、标签（Task 9）的新签名 | 6、8、9 |
| `web/apps/web/core/components/dropdowns/intake-state/base.tsx`、`web/apps/web/core/components/dropdowns/intake-state/dropdown.tsx`、`web/packages/propel/src/icons/state/intake-state-group-icon.tsx`、`web/packages/propel/src/icons/state/triage-group-icon.tsx`、`web/packages/types/src/intake/index.ts`、`web/packages/types/src/intake/state.ts` | 收集箱的分诊状态（M7，删除） | 7 |
| `web/apps/web/core/components/inbox/content/issue-properties.tsx`、`web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx`、`web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx`、`web/apps/web/core/hooks/use-workspace-issue-properties.ts`、`web/packages/propel/src/icons/state/index.ts`、`web/packages/types/src/index.ts` | 分诊状态和工作区级的标签的使用方 | 7 |
| `web/packages/ui/src/dropdowns/combo-box.tsx` | `ComboBox` 的根可以给 `role`（标签选择器清零） | 7 |
| `web/apps/web/core/store/state.store.ts`、`web/packages/propel/src/icons/state/helper.tsx` | 分诊状态删除（Task 7）；重写、`StateGroup`（Task 8） | 7、8 |
| `web/apps/web/core/services/project/project-state.service.ts` | 分诊状态的方法删除（Task 7）；整个旧 service 删除（Task 8） | 7、8 |
| `web/apps/web/core/store/label.store.ts` | 工作区级的标签删除（Task 7）；重写（Task 9，整个文件） | 7、9 |
| `web/apps/web/core/services/issue/issue_label.service.ts` | 工作区级的标签删除（Task 7）；整个旧 service 删除（Task 9） | 7、9 |
| `web/apps/web/core/components/issues/issue-detail/label/root.tsx`、`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`、`web/apps/web/core/components/issues/select/base.tsx`、`web/apps/web/core/components/labels/create-update-label-inline.tsx`、`web/apps/web/core/components/labels/delete-label-modal.tsx`、`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`、`web/apps/web/core/components/labels/project-setting-label-item.tsx`、`web/apps/web/core/components/labels/project-setting-label-list.tsx` | 清零（Task 7）；`Label`（Task 9 机械）和新 store（Task 9） | 7、9 |
| `web/apps/web/core/services/project/states.service.ts`、`web/apps/web/core/store/state.store.test.ts` | 状态：列出（项目、工作区）、创建、修改、删除、设默认；store 的测试 | 8 |
| `web/apps/web/core/lib/reconciled.ts` | `ReconciledByKey.values()` | 8 |
| `web/packages/types/src/state.ts`、`web/packages/utils/src/work-item/state.ts` | 只留页面的回调类型（整个文件）；`sortStates` 用 `toSorted` | 8 |
| `web/apps/web/core/components/dropdowns/state/base.tsx`、`web/apps/web/core/components/dropdowns/state/dropdown.tsx`、`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx`、`web/apps/web/core/components/project-states/create-update/create.tsx`、`web/apps/web/core/components/project-states/create-update/form.tsx`、`web/apps/web/core/components/project-states/create-update/update.tsx`、`web/apps/web/core/components/project-states/options/delete.tsx`、`web/apps/web/core/components/project-states/root.tsx`、`web/apps/web/core/components/project-states/state-item.tsx`、`web/apps/web/core/store/issue/issue-details/issue.store.ts` | 状态 store 的使用方：组中的位置、移动经 `updateState`（`dropdown.tsx`、`root.tsx` 整个文件） | 8 |
| `web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx`、`web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx`、`web/apps/web/core/components/issues/preview-card/date.tsx`、`web/apps/web/core/components/issues/preview-card/root.tsx`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx`、`web/apps/web/core/components/project-states/group-item.tsx`、`web/apps/web/core/components/project-states/group-list.tsx`、`web/apps/web/core/components/project-states/state-item-title.tsx`、`web/apps/web/core/components/project-states/state-list.tsx`、`web/packages/constants/src/state.ts`、`web/packages/types/src/issues/issue.ts`、`web/packages/utils/src/distribution-update.ts`、`web/packages/utils/src/progress.test.ts`、`web/packages/utils/src/work-item-filters/configs/filters/state.ts`、`web/packages/utils/src/work-item/base.ts` | `IState`、`TStateGroups` → `State`、`StateGroup` | 8 机械 |
| `web/packages/utils/package.json` | `utils` 的 oxlint 上限 | 2、8、9 |
| `web/apps/web/core/services/project/labels.service.ts`、`web/apps/web/core/store/label.store.test.ts` | 标签：列出、创建、修改、删除；store 的测试 | 9 |
| `web/apps/web/core/components/issues/select/dropdown.tsx`、`web/apps/web/core/components/labels/label-utils.ts`、`web/apps/web/core/services/issue/index.ts`、`web/packages/types/src/issues.ts`、`web/packages/utils/src/array.ts` | 标签 store 的使用方；`IIssueLabel`、`IIssueLabelTree`、`buildTree` 删除；`toSorted` | 9 |
| `web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx`、`web/apps/web/core/components/labels/label-block/label-item-block.tsx`、`web/apps/web/core/components/labels/project-setting-label-group.tsx`、`web/apps/web/core/components/power-k/menus/labels.tsx`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx`、`web/apps/web/core/components/ui/labels-list.tsx`、`web/packages/utils/src/work-item-filters/configs/filters/label.ts` | `IIssueLabel` → `Label` | 9 机械 |
| `web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`、`web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts` | 项目包装层挂载时的取数和它的全部判断 | 10 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx`、`web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx`、`web/packages/i18n/src/locales/en/empty-state.json`、`web/packages/i18n/src/locales/zh-CN/empty-state.json` | 包装层不收 `workspaceSlug`；"无权访问"的界面和文案删除，加入的界面收 `canJoin` | 10 |
| `e2e/stories/smoke/s2-web-app.spec.ts` | S2：四个账户登录之后的挂载清单 | 11 |
| `web/apps/web/core/store/project/project_filter.store.ts`、`web/apps/web/core/lib/store-context.test.ts` | 项目筛选的死行删除 | 11 |
| `docs/v0/v0-design.md`、`docs/v0/frontend-changes.md` | 3.20 中 P8b 的行 | 11 |

---

### Task 1: 生成的 `Project` 没有的：封面的上传和预设、项目的收藏、`intake_count`、`next_work_item_sequence`

**Files:**
- Modify: `web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/app/assets/cover-images/SOURCES.md`、`web/apps/web/core/components/issues/issue-layouts/list/block.tsx`、`web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/create-project-modal.tsx`、`web/apps/web/core/components/project/create/header.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/projects/create/root.tsx`、`web/apps/web/core/components/projects/create/utils.ts`、`web/apps/web/core/components/workspace/sidebar/project-navigation.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/core/store/favorite.store.ts`、`web/apps/web/core/store/inbox/inbox-issue.store.ts`、`web/apps/web/core/store/inbox/project-inbox.store.ts`、`web/apps/web/helpers/cover-image.helper.ts`、`web/apps/web/package.json`、`web/packages/constants/src/file.ts`、`web/packages/constants/src/tab-indices.ts`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/types/src/enums.ts`
- Delete: `web/apps/web/core/components/core/image-picker-popover.tsx`、`web/apps/web/app/assets/cover-images/` 的 `image_2`–`image_29` 的 `.svg` 和 `.webp`（56 个）

**Interfaces:**
- Removes（spec 2.1；M3 设计 3.2 的规则 2、7.8；生成的 `Project` 没有这些字段，`ProjectUpdate` 不能设封面）：
  - 封面的上传和选择：`core/components/core/image-picker-popover.tsx`（删除）；`helpers/cover-image.helper.ts` 只留 `DEFAULT_COVER_IMAGE_URL`（`image_1.webp`）和 `getCoverImageDisplayURL`（整个文件）；其余 28 张预设封面（`image_2`–`image_29` 的 `.svg`、`.webp`）删除，`SOURCES.md` 照改（整个文件）；`EFileAssetType.PROJECT_COVER`、`ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE`、`TAB_INDEX` 的 `cover_image`、文案 `change_cover`（两种语言）删除。项目的卡片、设置页、创建弹窗只显示封面：没有自己的封面时显示默认的那一张（`CoverImage` 的 `showDefaultWhenEmpty`）；创建时不再随机给封面、不再上传。
  - 项目的收藏（M7）：卡片的星标、创建弹窗的 `setToFavorite`、侧边栏菜单里注释掉的收藏一段、`FavoriteStore` 中项目的一支（它写 `projectMap[…].is_favorite`）删除。收藏本身（工作区的收藏列表）不动，M7 照新接口加回。
  - `intake_count`：侧边栏"收集箱"一项的数字、收集箱 store 在状态变化时改写项目的 `intake_count` 的两段删除。
  - `next_work_item_sequence`：列表视图的工作项的键宽按一位数字算（`currentProjectNextSequenceId` 不再读；M4 加回）。
- 顺手清零（7.9，R3）：这些文件的 oxlint 警告清零（`then` 的回调不再返回值、`div` 的点击改为 `button`、`autoFocus` 改为侧边栏打开时 `focus()`、加载行的键不用下标）。

**Tests:** 没有新的 vitest：本 Task 只删除，`tsc` 和 knip 核对没有留下读者。

- [ ] **Step 1: 封面**

`web/apps/web/app/assets/cover-images/SOURCES.md`（整个文件，21 行）：

````whole web/apps/web/app/assets/cover-images/SOURCES.md
# Sources of the default project cover

Copyright (c) 2026-present OpenNerve
SPDX-License-Identifier: AGPL-3.0-only

The picture in this directory is an original Nerve drawing: an abstract pattern on a 1600 × 900 canvas, with no
photograph, person, name or product in it. The SVG is the source, and carries this notice in an XML comment; the
WebP file cannot, so it is listed here.

`helpers/cover-image.helper.ts` imports the WebP file: it is the cover a project shows while it has none of its own
(M3 design 3.2). `image_1.webp` is its SVG rendered at the SVG's size, 1920 × 1080, and encoded as lossy WebP at
quality 0.9 (in Chromium, with `canvas.toBlob(callback, "image/webp", 0.9)`). To change it, edit the SVG and render
the WebP again at the same size; the code reads the file by name only. The 28 other preset covers went with the
cover picker (M3/P8b): M5 brings covers back with uploads.

The app crops a cover to a wide band (the project card, the project settings) and writes white text and icons on it,
so the drawing keeps its detail across the middle of the canvas and stays mid to dark.

| Files                         | Drawing                     |
| ----------------------------- | --------------------------- |
| `image_1.svg`, `image_1.webp` | Soft fields of colour, teal |
````

`web/apps/web/app/assets/cover-images/image_10.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_10.svg
````

`web/apps/web/app/assets/cover-images/image_10.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_10.webp
````

`web/apps/web/app/assets/cover-images/image_11.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_11.svg
````

`web/apps/web/app/assets/cover-images/image_11.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_11.webp
````

`web/apps/web/app/assets/cover-images/image_12.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_12.svg
````

`web/apps/web/app/assets/cover-images/image_12.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_12.webp
````

`web/apps/web/app/assets/cover-images/image_13.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_13.svg
````

`web/apps/web/app/assets/cover-images/image_13.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_13.webp
````

`web/apps/web/app/assets/cover-images/image_14.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_14.svg
````

`web/apps/web/app/assets/cover-images/image_14.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_14.webp
````

`web/apps/web/app/assets/cover-images/image_15.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_15.svg
````

`web/apps/web/app/assets/cover-images/image_15.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_15.webp
````

`web/apps/web/app/assets/cover-images/image_16.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_16.svg
````

`web/apps/web/app/assets/cover-images/image_16.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_16.webp
````

`web/apps/web/app/assets/cover-images/image_17.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_17.svg
````

`web/apps/web/app/assets/cover-images/image_17.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_17.webp
````

`web/apps/web/app/assets/cover-images/image_18.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_18.svg
````

`web/apps/web/app/assets/cover-images/image_18.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_18.webp
````

`web/apps/web/app/assets/cover-images/image_19.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_19.svg
````

`web/apps/web/app/assets/cover-images/image_19.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_19.webp
````

`web/apps/web/app/assets/cover-images/image_2.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_2.svg
````

`web/apps/web/app/assets/cover-images/image_2.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_2.webp
````

`web/apps/web/app/assets/cover-images/image_20.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_20.svg
````

`web/apps/web/app/assets/cover-images/image_20.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_20.webp
````

`web/apps/web/app/assets/cover-images/image_21.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_21.svg
````

`web/apps/web/app/assets/cover-images/image_21.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_21.webp
````

`web/apps/web/app/assets/cover-images/image_22.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_22.svg
````

`web/apps/web/app/assets/cover-images/image_22.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_22.webp
````

`web/apps/web/app/assets/cover-images/image_23.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_23.svg
````

`web/apps/web/app/assets/cover-images/image_23.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_23.webp
````

`web/apps/web/app/assets/cover-images/image_24.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_24.svg
````

`web/apps/web/app/assets/cover-images/image_24.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_24.webp
````

`web/apps/web/app/assets/cover-images/image_25.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_25.svg
````

`web/apps/web/app/assets/cover-images/image_25.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_25.webp
````

`web/apps/web/app/assets/cover-images/image_26.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_26.svg
````

`web/apps/web/app/assets/cover-images/image_26.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_26.webp
````

`web/apps/web/app/assets/cover-images/image_27.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_27.svg
````

`web/apps/web/app/assets/cover-images/image_27.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_27.webp
````

`web/apps/web/app/assets/cover-images/image_28.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_28.svg
````

`web/apps/web/app/assets/cover-images/image_28.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_28.webp
````

`web/apps/web/app/assets/cover-images/image_29.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_29.svg
````

`web/apps/web/app/assets/cover-images/image_29.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_29.webp
````

`web/apps/web/app/assets/cover-images/image_3.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_3.svg
````

`web/apps/web/app/assets/cover-images/image_3.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_3.webp
````

`web/apps/web/app/assets/cover-images/image_4.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_4.svg
````

`web/apps/web/app/assets/cover-images/image_4.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_4.webp
````

`web/apps/web/app/assets/cover-images/image_5.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_5.svg
````

`web/apps/web/app/assets/cover-images/image_5.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_5.webp
````

`web/apps/web/app/assets/cover-images/image_6.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_6.svg
````

`web/apps/web/app/assets/cover-images/image_6.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_6.webp
````

`web/apps/web/app/assets/cover-images/image_7.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_7.svg
````

`web/apps/web/app/assets/cover-images/image_7.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_7.webp
````

`web/apps/web/app/assets/cover-images/image_8.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_8.svg
````

`web/apps/web/app/assets/cover-images/image_8.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_8.webp
````

`web/apps/web/app/assets/cover-images/image_9.svg`（删除）：

````delete web/apps/web/app/assets/cover-images/image_9.svg
````

`web/apps/web/app/assets/cover-images/image_9.webp`（删除）：

````delete web/apps/web/app/assets/cover-images/image_9.webp
````

`web/apps/web/core/components/core/image-picker-popover.tsx`（删除）：

````delete web/apps/web/core/components/core/image-picker-popover.tsx
````

`web/apps/web/helpers/cover-image.helper.ts`（整个文件，32 行）：

````whole web/apps/web/helpers/cover-image.helper.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { getFileURL } from "@nerve/utils";

import DefaultCoverImage from "@/app/assets/cover-images/image_1.webp?url";

/**
 * The cover shown where a project has none of its own: nerve gives cover_image_url as null until uploads arrive
 * (M5; M3 design 3.2).
 */
export const DEFAULT_COVER_IMAGE_URL = DefaultCoverImage;

/**
 * Gets the correct display URL for a cover image
 * - The default cover: returned as-is (served from assets folder)
 * - Uploaded assets: processed through getFileURL (adds backend URL)
 */
export function getCoverImageDisplayURL(imageUrl: string | null | undefined, fallbackUrl: string): string {
  if (!imageUrl) {
    return fallbackUrl;
  }

  if (imageUrl === DEFAULT_COVER_IMAGE_URL) {
    return imageUrl;
  }

  return getFileURL(imageUrl) || imageUrl;
}
````

`web/packages/constants/src/file.ts`（修改，1 处）：

````old web/packages/constants/src/file.ts
export const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5MB

export const ACCEPTED_COVER_IMAGE_MIME_TYPES_FOR_REACT_DROPZONE = {
  "image/jpeg": [],
  "image/jpg": [],
  "image/png": [],
  "image/webp": [],
};
````
````new web/packages/constants/src/file.ts
export const MAX_FILE_SIZE = 5 * 1024 * 1024; // 5MB
````

`web/packages/constants/src/tab-indices.ts`（修改，1 处）：

````old web/packages/constants/src/tab-indices.ts
  "close",
  "cover_image",
````
````new web/packages/constants/src/tab-indices.ts
  "close",
````

`web/packages/i18n/src/locales/en/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/common.json
  "password": "Password",
  "change_cover": "Change cover",
````
````new web/packages/i18n/src/locales/en/common.json
  "password": "Password",
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "password": "密码",
  "change_cover": "更改封面",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "password": "密码",
````

`web/packages/types/src/enums.ts`（修改，1 处）：

````old web/packages/types/src/enums.ts
  DRAFT_ISSUE_DESCRIPTION = "DRAFT_ISSUE_DESCRIPTION",
  PROJECT_COVER = "PROJECT_COVER",
````
````new web/packages/types/src/enums.ts
  DRAFT_ISSUE_DESCRIPTION = "DRAFT_ISSUE_DESCRIPTION",
````

- [ ] **Step 2: 使用方**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`（修改，7 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
import { useCallback, useRef, useState } from "react";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
import { useCallback, useEffect, useRef, useState } from "react";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const extendedProjectSidebarRef = useRef<HTMLDivElement | null>(null);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const extendedProjectSidebarRef = useRef<HTMLDivElement | null>(null);
  const searchInputRef = useRef<HTMLInputElement | null>(null);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const handleClose = useCallback(() => toggleExtendedProjectSidebar(false), [toggleExtendedProjectSidebar]);

````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const handleClose = useCallback(() => toggleExtendedProjectSidebar(false), [toggleExtendedProjectSidebar]);

  // the search takes the focus each time the sidebar opens
  useEffect(() => {
    if (isExtendedProjectSidebarOpened) searchInputRef.current?.focus();
  }, [isExtendedProjectSidebarOpened]);

````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() => {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() =>
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
      });
    });
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
      })
    );
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
          onClose={() => setIsProjectModalOpen(false)}
          setToFavorite={false}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
          onClose={() => setIsProjectModalOpen(false)}
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
              autoFocus
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
              ref={searchInputRef}
````

`web/apps/web/core/components/issues/issue-layouts/list/block.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/issue-layouts/list/block.tsx
  const { getProjectIdentifierById, currentProjectNextSequenceId } = useProject();
````
````new web/apps/web/core/components/issues/issue-layouts/list/block.tsx
  const { getProjectIdentifierById } = useProject();
````
````old web/apps/web/core/components/issues/issue-layouts/list/block.tsx
  // Calculate width for: projectIdentifier + "-" + dynamic sequence number digits
  // Use next_work_item_sequence from backend (static value from project endpoint)
  const maxSequenceId = currentProjectNextSequenceId ?? 1;
  const keyMinWidth = displayProperties?.key
    ? calculateIdentifierWidth(projectIdentifier?.length ?? 0, maxSequenceId)
    : 0;
````
````new web/apps/web/core/components/issues/issue-layouts/list/block.tsx
  // Calculate width for: projectIdentifier + "-" + the sequence number's digits: one, as nerve's project has no
  // count of its work items until M4 adds it (M3 design 3.2)
  const keyMinWidth = displayProperties?.key ? calculateIdentifierWidth(projectIdentifier?.length ?? 0, 1) : 0;
````

`web/apps/web/core/components/project/card.tsx`（修改，12 处）：

````old web/apps/web/core/components/project/card.tsx
import { EUserPermissions, EUserPermissionsLevel, IS_FAVORITE_MENU_OPEN } from "@nerve/constants";
import { useLocalStorage } from "@nerve/hooks";
````
````new web/apps/web/core/components/project/card.tsx
import { EUserPermissions } from "@nerve/constants";
````
````old web/apps/web/core/components/project/card.tsx
import { setPromiseToast, setToast, TOAST_TYPE } from "@nerve/propel/toast";
````
````new web/apps/web/core/components/project/card.tsx
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
````
````old web/apps/web/core/components/project/card.tsx
import { ContextMenu, FavoriteStar } from "@nerve/ui";
````
````new web/apps/web/core/components/project/card.tsx
import { ContextMenu } from "@nerve/ui";
````
````old web/apps/web/core/components/project/card.tsx
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/project/card.tsx
import { useMember } from "@/hooks/store/use-member";
````
````old web/apps/web/core/components/project/card.tsx
  const { getUserDetails } = useMember();
  const { addProjectToFavorites, removeProjectFromFavorites } = useProject();
  const { allowPermissions } = useUserPermissions();
````
````new web/apps/web/core/components/project/card.tsx
  const { getUserDetails } = useMember();
````
````old web/apps/web/core/components/project/card.tsx
  const projectMembersIds = project.members;
  const shouldRenderFavorite = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE
  );
````
````new web/apps/web/core/components/project/card.tsx
  const projectMembersIds = project.members;
````
````old web/apps/web/core/components/project/card.tsx
  const isArchived = !!project.archived_at;
  // local storage
  const { setValue: toggleFavoriteMenu, storedValue: isFavoriteMenuOpen } = useLocalStorage<boolean>(
    IS_FAVORITE_MENU_OPEN,
    false
  );

  const handleAddToFavorites = () => {
    if (!workspaceSlug) return;

    const addToFavoritePromise = addProjectToFavorites(workspaceSlug, project.id);
    setPromiseToast(addToFavoritePromise, {
      loading: "Adding project to favorites...",
      success: {
        title: "Success!",
        message: () => "Project added to favorites.",
        actionItems: () => {
          if (!isFavoriteMenuOpen) toggleFavoriteMenu(true);
          return <></>;
        },
      },
      error: {
        title: "Error!",
        message: () => "Couldn't add the project to favorites. Please try again.",
      },
    });
  };

  const handleRemoveFromFavorites = () => {
    if (!workspaceSlug) return;

    const removeFromFavoritePromise = removeProjectFromFavorites(workspaceSlug, project.id);
    setPromiseToast(removeFromFavoritePromise, {
      loading: "Removing project from favorites...",
      success: {
        title: "Success!",
        message: () => "Project removed from favorites.",
      },
      error: {
        title: "Error!",
        message: () => "Couldn't remove the project from favorites. Please try again.",
      },
    });
  };
````
````new web/apps/web/core/components/project/card.tsx
  const isArchived = !!project.archived_at;
````
````old web/apps/web/core/components/project/card.tsx
            src={project.cover_image_url}
````
````new web/apps/web/core/components/project/card.tsx
            src={project.cover_image_url}
            showDefaultWhenEmpty
````
````old web/apps/web/core/components/project/card.tsx
                </button>
                {shouldRenderFavorite && (
                  <FavoriteStar
                    buttonClassName="h-6 w-6 bg-white/10 rounded-sm"
                    iconClassName={cn("h-3 w-3", {
                      "text-on-color": !project.is_favorite,
                    })}
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      if (project.is_favorite) handleRemoveFromFavorites();
                      else handleAddToFavorites();
                    }}
                    selected={!!project.is_favorite}
                  />
                )}
````
````new web/apps/web/core/components/project/card.tsx
                </button>
````
````old web/apps/web/core/components/project/card.tsx
                <div className="flex items-center justify-center gap-2">
                  <div
````
````new web/apps/web/core/components/project/card.tsx
                <div className="flex items-center justify-center gap-2">
                  <button
                    type="button"
````
````old web/apps/web/core/components/project/card.tsx
                  </div>
                  <div
````
````new web/apps/web/core/components/project/card.tsx
                  </button>
                  <button
                    type="button"
````
````old web/apps/web/core/components/project/card.tsx
                    <DeleteOutline className="h-3.5 w-3.5" />
                  </div>
````
````new web/apps/web/core/components/project/card.tsx
                    <DeleteOutline className="h-3.5 w-3.5" />
                  </button>
````

`web/apps/web/core/components/project/create-project-modal.tsx`（修改，7 处）：

````old web/apps/web/core/components/project/create-project-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { getAssetIdFromUrl, checkURLValidity } from "@nerve/utils";
````
````new web/apps/web/core/components/project/create-project-modal.tsx
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
````
````old web/apps/web/core/components/project/create-project-modal.tsx
import type { TProject } from "@nerve/types";
// services
import { FileService } from "@/services/file.service";
const fileService = new FileService();
````
````new web/apps/web/core/components/project/create-project-modal.tsx
import type { TProject } from "@nerve/types";
````
````old web/apps/web/core/components/project/create-project-modal.tsx
  onClose: () => void;
  setToFavorite?: boolean;
````
````new web/apps/web/core/components/project/create-project-modal.tsx
  onClose: () => void;
````
````old web/apps/web/core/components/project/create-project-modal.tsx
  const { isOpen, onClose, setToFavorite = false, workspaceSlug, data } = props;
````
````new web/apps/web/core/components/project/create-project-modal.tsx
  const { isOpen, onClose, workspaceSlug, data } = props;
````
````old web/apps/web/core/components/project/create-project-modal.tsx

  const handleCoverImageStatusUpdate = async (projectId: string, coverImage: string) => {
    if (!checkURLValidity(coverImage)) {
      await fileService.updateBulkProjectAssetsUploadStatus(workspaceSlug, projectId, projectId, {
        asset_ids: [getAssetIdFromUrl(coverImage)],
      });
    }
  };

````
````new web/apps/web/core/components/project/create-project-modal.tsx

````
````old web/apps/web/core/components/project/create-project-modal.tsx
        <CreateProjectForm
          setToFavorite={setToFavorite}
````
````new web/apps/web/core/components/project/create-project-modal.tsx
        <CreateProjectForm
````
````old web/apps/web/core/components/project/create-project-modal.tsx
          onClose={onClose}
          updateCoverImageStatus={handleCoverImageStatusUpdate}
````
````new web/apps/web/core/components/project/create-project-modal.tsx
          onClose={onClose}
````

`web/apps/web/core/components/project/create/header.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/create/header.tsx
import { ImagePickerPopover } from "@/components/core/image-picker-popover";
````
````new web/apps/web/core/components/project/create/header.tsx
import { DEFAULT_COVER_IMAGE_URL } from "@/helpers/cover-image.helper";
````
````old web/apps/web/core/components/project/create/header.tsx
  const { watch, control, setValue } = useFormContext<IProject>();
  const { t } = useTranslation();
  // derived values
  const coverImage = watch("cover_image_url");
````
````new web/apps/web/core/components/project/create/header.tsx
  const { control, setValue } = useFormContext<IProject>();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project/create/header.tsx
        src={coverImage}
````
````new web/apps/web/core/components/project/create/header.tsx
        src={DEFAULT_COVER_IMAGE_URL}
````
````old web/apps/web/core/components/project/create/header.tsx
        </button>
      </div>
      <div className="absolute right-2 bottom-2">
        <Controller
          name="cover_image_url"
          control={control}
          render={({ field: { value, onChange } }) => (
            <ImagePickerPopover
              label={t("change_cover")}
              onChange={onChange}
              value={value ?? null}
              tabIndex={getIndex("cover_image")}
            />
          )}
        />
````
````new web/apps/web/core/components/project/create/header.tsx
        </button>
````

`web/apps/web/core/components/project/form.tsx`（修改，6 处）：

````old web/apps/web/core/components/project/form.tsx
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { EFileAssetType } from "@nerve/types";
````
````new web/apps/web/core/components/project/form.tsx
import { Tooltip } from "@makeplane/propel/components/tooltip";
````
````old web/apps/web/core/components/project/form.tsx
import { CoverImage } from "@/components/common/cover-image";
import { ImagePickerPopover } from "@/components/core/image-picker-popover";
import { TimezoneSelect } from "@/components/global";
// helpers
import { handleCoverImageChange } from "@/helpers/cover-image.helper";
````
````new web/apps/web/core/components/project/form.tsx
import { CoverImage } from "@/components/common/cover-image";
import { TimezoneSelect } from "@/components/global";
````
````old web/apps/web/core/components/project/form.tsx
  const currentNetwork = NETWORK_CHOICES.find((n) => n.key === project?.network);
  const coverImage = watch("cover_image_url");
````
````new web/apps/web/core/components/project/form.tsx
  const currentNetwork = NETWORK_CHOICES.find((n) => n.key === project?.network);
````
````old web/apps/web/core/components/project/form.tsx

    // Handle cover image changes
    try {
      const coverImagePayload = await handleCoverImageChange(project.cover_image_url, formData.cover_image_url, {
        workspaceSlug,
        entityIdentifier: project.id,
        entityType: EFileAssetType.PROJECT_COVER,
      });

      if (coverImagePayload) {
        Object.assign(payload, coverImagePayload);
      }
    } catch (error) {
      console.error("Error handling cover image:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: error instanceof Error ? error.message : "Failed to process cover image",
      });
      setIsLoading(false);
      return;
    }

````
````new web/apps/web/core/components/project/form.tsx

````
````old web/apps/web/core/components/project/form.tsx
        <CoverImage src={coverImage} alt="Project cover image" className="h-44 w-full rounded-md" />
````
````new web/apps/web/core/components/project/form.tsx
        <CoverImage
          src={project.cover_image_url}
          showDefaultWhenEmpty
          alt="Project cover image"
          className="h-44 w-full rounded-md"
        />
````
````old web/apps/web/core/components/project/form.tsx
              </span>
            </div>
          </div>
          <div className="flex flex-shrink-0 justify-center">
            <div>
              <Controller
                control={control}
                name="cover_image_url"
                render={({ field: { value, onChange } }) => (
                  <ImagePickerPopover
                    label={t("change_cover")}
                    onChange={onChange}
                    value={value ?? null}
                    disabled={!isAdmin}
                    projectId={project.id}
                  />
                )}
              />
````
````new web/apps/web/core/components/project/form.tsx
              </span>
````

`web/apps/web/core/components/projects/create/root.tsx`（修改，9 处）：

````old web/apps/web/core/components/projects/create/root.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EFileAssetType } from "@nerve/types";
````
````new web/apps/web/core/components/projects/create/root.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````old web/apps/web/core/components/projects/create/root.tsx
// hooks
import { getCoverImageType, uploadCoverImage } from "@/helpers/cover-image.helper";
````
````new web/apps/web/core/components/projects/create/root.tsx
// hooks
````
````old web/apps/web/core/components/projects/create/root.tsx
export type TCreateProjectFormProps = {
  setToFavorite?: boolean;
````
````new web/apps/web/core/components/projects/create/root.tsx
export type TCreateProjectFormProps = {
````
````old web/apps/web/core/components/projects/create/root.tsx
  data?: Partial<TProject>;
  updateCoverImageStatus: (projectId: string, coverImage: string) => Promise<void>;
````
````new web/apps/web/core/components/projects/create/root.tsx
  data?: Partial<TProject>;
````
````old web/apps/web/core/components/projects/create/root.tsx
  const { setToFavorite, workspaceSlug, data, onClose, handleNextStep, updateCoverImageStatus } = props;
````
````new web/apps/web/core/components/projects/create/root.tsx
  const { workspaceSlug, data, onClose, handleNextStep } = props;
````
````old web/apps/web/core/components/projects/create/root.tsx
  const { addProjectToFavorites, createProject, updateProject } = useProject();
````
````new web/apps/web/core/components/projects/create/root.tsx
  const { createProject } = useProject();
````
````old web/apps/web/core/components/projects/create/root.tsx
  const { isMobile } = usePlatformOS();
  const handleAddToFavorites = (projectId: string) => {
    if (!workspaceSlug) return;

    addProjectToFavorites(workspaceSlug, projectId).catch(() => {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("toast.error"),
        message: t("failed_to_remove_project_from_favorites"),
      });
    });
  };
````
````new web/apps/web/core/components/projects/create/root.tsx
  const { isMobile } = usePlatformOS();
````
````old web/apps/web/core/components/projects/create/root.tsx
    formData.identifier = formData.identifier?.toUpperCase();
    const coverImage = formData.cover_image_url;
    let uploadedAssetUrl: string | null = null;

    if (coverImage) {
      const imageType = getCoverImageType(coverImage);

      if (imageType === "local_static") {
        try {
          uploadedAssetUrl = await uploadCoverImage(coverImage, {
            workspaceSlug,
            entityIdentifier: "",
            entityType: EFileAssetType.PROJECT_COVER,
          });
        } catch (error) {
          console.error("Error uploading cover image:", error);
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: error instanceof Error ? error.message : "Failed to upload cover image",
          });
          return Promise.reject(error);
        }
      } else {
        formData.cover_image = coverImage;
        formData.cover_image_asset = null;
      }
    }

    return createProject(workspaceSlug, formData)
      .then(async (res) => {
        if (uploadedAssetUrl) {
          await updateCoverImageStatus(res.id, uploadedAssetUrl);
          await updateProject(workspaceSlug, res.id, { cover_image_url: uploadedAssetUrl });
        } else if (coverImage && coverImage.startsWith("http")) {
          await updateCoverImageStatus(res.id, coverImage);
          await updateProject(workspaceSlug, res.id, { cover_image_url: coverImage });
        }
````
````new web/apps/web/core/components/projects/create/root.tsx
    return createProject(workspaceSlug, { ...formData, identifier: formData.identifier?.toUpperCase() })
      .then((res) => {
````
````old web/apps/web/core/components/projects/create/root.tsx
        });

        if (setToFavorite) {
          handleAddToFavorites(res.id);
        }
````
````new web/apps/web/core/components/projects/create/root.tsx
        });
````

`web/apps/web/core/components/projects/create/utils.ts`（修改，2 处）：

````old web/apps/web/core/components/projects/create/utils.ts
import type { IProject } from "@nerve/types";
import { getRandomCoverImage } from "@/helpers/cover-image.helper";
````
````new web/apps/web/core/components/projects/create/utils.ts
import type { IProject } from "@nerve/types";
````
````old web/apps/web/core/components/projects/create/utils.ts
export const getProjectFormValues = (): Partial<IProject> => ({
  cover_image_url: getRandomCoverImage(),
````
````new web/apps/web/core/components/projects/create/utils.ts
export const getProjectFormValues = (): Partial<IProject> => ({
````

`web/apps/web/core/components/workspace/sidebar/project-navigation.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
    (workspaceSlug: string, projectId: string): TNavigationItem[] => [
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
    (): TNavigationItem[] => [
````
````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
    [project]
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
    [project, workspaceSlug, projectId]
````
````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
    () => baseNavigation(workspaceSlug, projectId).sort((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0)),
    [workspaceSlug, projectId, baseNavigation]
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
    () => baseNavigation().sort((a, b) => (a.sortOrder || 0) - (b.sortOrder || 0)),
    [baseNavigation]
````
````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx

        const shouldShowCount = item.key === "intake" && (project.intake_count ?? 0) > 0;

````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx

````
````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
                </div>
                {shouldShowCount && <span className="text-11 font-medium text-tertiary">{project.intake_count}</span>}
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
                </div>
````

`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                >
                  {/* TODO: Removed is_favorite logic due to the optimization in projects API */}
                  {/* {isAuthorized && (
                    <CustomMenu.MenuItem
                      onClick={project.is_favorite ? handleRemoveFromFavorites : handleAddToFavorites}
                    >
                      <span className="flex items-center justify-start gap-2">
                        <Star
                          className={cn("h-3.5 w-3.5 ", {
                            "fill-yellow-500 stroke-yellow-500": project.is_favorite,
                          })}
                        />
                        <span>{project.is_favorite ? t("remove_from_favorites") : t("add_to_favorites")}</span>
                      </span>
                    </CustomMenu.MenuItem>
                  )} */}

````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
                >
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，4 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() => {
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() =>
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
      });
    });
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
      })
    );
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
          onClose={() => setIsProjectModalOpen(false)}
          setToFavorite={false}
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
          onClose={() => setIsProjectModalOpen(false)}
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                  {Array.from({ length: 4 }).map((_, index) => (
                    <Loader.Item key={index} height="28px" />
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                  {["first", "second", "third", "fourth"].map((row) => (
                    <Loader.Item key={row} height="28px" />
````

`web/apps/web/core/store/favorite.store.ts`（修改，3 处）：

````old web/apps/web/core/store/favorite.store.ts
  viewStore;
  projectStore;
````
````new web/apps/web/core/store/favorite.store.ts
  viewStore;
````
````old web/apps/web/core/store/favorite.store.ts
    this.viewStore = _rootStore.projectView;
    this.projectStore = _rootStore.projectRoot.project;
````
````new web/apps/web/core/store/favorite.store.ts
    this.viewStore = _rootStore.projectView;
````
````old web/apps/web/core/store/favorite.store.ts
        );
      case "project":
        return (
          this.projectStore.projectMap[entity_identifier] &&
          (this.projectStore.projectMap[entity_identifier].is_favorite = false)
        );
````
````new web/apps/web/core/store/favorite.store.ts
        );
````

`web/apps/web/core/store/inbox/inbox-issue.store.ts`（修改，6 处）：

````old web/apps/web/core/store/inbox/inbox-issue.store.ts
      status: this.status,
    };
    const previousStatus = this.status;
````
````new web/apps/web/core/store/inbox/inbox-issue.store.ts
      status: this.status,
    };
````
````old web/apps/web/core/store/inbox/inbox-issue.store.ts
        set(this, "status", inboxIssue?.status);

        // Handle intake_count transitions
        if (previousStatus === EInboxIssueStatus.PENDING && inboxIssue.status !== EInboxIssueStatus.PENDING) {
          // Changed from PENDING to something else: decrement
          const currentCount = this.store.projectRoot.project.projectMap[this.projectId]?.intake_count ?? 0;
          set(
            this.store.projectRoot.project.projectMap,
            [this.projectId, "intake_count"],
            Math.max(0, currentCount - 1)
          );
        } else if (previousStatus !== EInboxIssueStatus.PENDING && inboxIssue.status === EInboxIssueStatus.PENDING) {
          // Changed from something else to PENDING: increment
          const currentCount = this.store.projectRoot.project.projectMap[this.projectId]?.intake_count ?? 0;
          set(this.store.projectRoot.project.projectMap, [this.projectId, "intake_count"], currentCount + 1);
        }
````
````new web/apps/web/core/store/inbox/inbox-issue.store.ts
        set(this, "status", inboxIssue?.status);
````
````old web/apps/web/core/store/inbox/inbox-issue.store.ts
    };
    const wasPending = this.status === EInboxIssueStatus.PENDING;
````
````new web/apps/web/core/store/inbox/inbox-issue.store.ts
    };
````
````old web/apps/web/core/store/inbox/inbox-issue.store.ts
        set(this, "duplicate_issue_detail", inboxIssue?.duplicate_issue_detail);
        // Decrement intake_count if the issue was PENDING
        if (wasPending) {
          const currentCount = this.store.projectRoot.project.projectMap[this.projectId]?.intake_count ?? 0;
          set(
            this.store.projectRoot.project.projectMap,
            [this.projectId, "intake_count"],
            Math.max(0, currentCount - 1)
          );
        }
````
````new web/apps/web/core/store/inbox/inbox-issue.store.ts
        set(this, "duplicate_issue_detail", inboxIssue?.duplicate_issue_detail);
````
````old web/apps/web/core/store/inbox/inbox-issue.store.ts
      snoozed_till: this.snoozed_till,
    };
    const previousStatus = this.status;
````
````new web/apps/web/core/store/inbox/inbox-issue.store.ts
      snoozed_till: this.snoozed_till,
    };
````
````old web/apps/web/core/store/inbox/inbox-issue.store.ts
        set(this, "snoozed_till", inboxIssue?.snoozed_till);
        // Handle intake_count transitions
        if (previousStatus === EInboxIssueStatus.PENDING && inboxIssue.status === EInboxIssueStatus.SNOOZED) {
          const currentCount = this.store.projectRoot.project.projectMap[this.projectId]?.intake_count ?? 0;
          set(
            this.store.projectRoot.project.projectMap,
            [this.projectId, "intake_count"],
            Math.max(0, currentCount - 1)
          );
        } else if (previousStatus !== EInboxIssueStatus.PENDING && inboxIssue.status === EInboxIssueStatus.PENDING) {
          const currentCount = this.store.projectRoot.project.projectMap[this.projectId]?.intake_count ?? 0;
          set(this.store.projectRoot.project.projectMap, [this.projectId, "intake_count"], currentCount + 1);
        }
````
````new web/apps/web/core/store/inbox/inbox-issue.store.ts
        set(this, "snoozed_till", inboxIssue?.snoozed_till);
````

`web/apps/web/core/store/inbox/project-inbox.store.ts`（修改，6 处）：

````old web/apps/web/core/store/inbox/project-inbox.store.ts
    this.inboxFilters != undefined &&
````
````new web/apps/web/core/store/inbox/project-inbox.store.ts
    if (this.inboxFilters != undefined)
````
````old web/apps/web/core/store/inbox/project-inbox.store.ts
        if (this.inboxFilters[filterKey] && this.inboxFilters?.[filterKey])
          count = count + (this.inboxFilters?.[filterKey]?.length ?? 0);
````
````new web/apps/web/core/store/inbox/project-inbox.store.ts
        if (this.inboxFilters[filterKey]) count = count + (this.inboxFilters?.[filterKey]?.length ?? 0);
````
````old web/apps/web/core/store/inbox/project-inbox.store.ts
    !isEmpty(inboxFilters) &&
````
````new web/apps/web/core/store/inbox/project-inbox.store.ts
    if (!isEmpty(inboxFilters))
````
````old web/apps/web/core/store/inbox/project-inbox.store.ts
          );
          // Increment intake_count if the new issue is PENDING
          if (inboxIssueResponse.status === EInboxIssueStatus.PENDING) {
            const currentCount = this.store.projectRoot.project.projectMap[projectId]?.intake_count ?? 0;
            set(this.store.projectRoot.project.projectMap, [projectId, "intake_count"], currentCount + 1);
          }
````
````new web/apps/web/core/store/inbox/project-inbox.store.ts
          );
````
````old web/apps/web/core/store/inbox/project-inbox.store.ts
    const currentIssue = this.inboxIssues?.[inboxIssueId];
    const wasPending = currentIssue?.status === EInboxIssueStatus.PENDING;
````
````new web/apps/web/core/store/inbox/project-inbox.store.ts
    const currentIssue = this.inboxIssues?.[inboxIssueId];
````
````old web/apps/web/core/store/inbox/project-inbox.store.ts
      await this.inboxIssueService.destroy(workspaceSlug, projectId, inboxIssueId).then(() => {
        runInAction(() => {
          set(
            this,
            ["inboxIssuePaginationInfo", "total_results"],
            (this.inboxIssuePaginationInfo?.total_results || 0) - 1
          );
          set(this, "inboxIssues", omit(this.inboxIssues, inboxIssueId));
          set(
            this,
            ["inboxIssueIds"],
            this.inboxIssueIds.filter((id) => id !== inboxIssueId)
          );
          // Decrement intake_count if the deleted issue was PENDING
          if (wasPending) {
            const currentCount = this.store.projectRoot.project.projectMap[projectId]?.intake_count ?? 0;
            set(this.store.projectRoot.project.projectMap, [projectId, "intake_count"], Math.max(0, currentCount - 1));
          }
        });
````
````new web/apps/web/core/store/inbox/project-inbox.store.ts
      await this.inboxIssueService.destroy(workspaceSlug, projectId, inboxIssueId);
      runInAction(() => {
        set(
          this,
          ["inboxIssuePaginationInfo", "total_results"],
          (this.inboxIssuePaginationInfo?.total_results || 0) - 1
        );
        set(this, "inboxIssues", omit(this.inboxIssues, inboxIssueId));
        set(
          this,
          ["inboxIssueIds"],
          this.inboxIssueIds.filter((id) => id !== inboxIssueId)
        );
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 435",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 421",
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 421 条，等于新的上限。

Run: `make knip`
Expected: 通过（删除的图片、组件、常量都不再被引用）。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

完成时：本 Task 删除的文案键（`common.json` 的 `change_cover`）在 `en`、`zh-CN` 中都已不在（spec 附录 A.7 的键表）。

- [ ] **Step 4: 提交**

```bash
git add 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' web/apps/web/app/assets/cover-images/SOURCES.md web/apps/web/app/assets/cover-images/image_10.svg web/apps/web/app/assets/cover-images/image_10.webp web/apps/web/app/assets/cover-images/image_11.svg web/apps/web/app/assets/cover-images/image_11.webp web/apps/web/app/assets/cover-images/image_12.svg web/apps/web/app/assets/cover-images/image_12.webp web/apps/web/app/assets/cover-images/image_13.svg web/apps/web/app/assets/cover-images/image_13.webp web/apps/web/app/assets/cover-images/image_14.svg web/apps/web/app/assets/cover-images/image_14.webp web/apps/web/app/assets/cover-images/image_15.svg web/apps/web/app/assets/cover-images/image_15.webp web/apps/web/app/assets/cover-images/image_16.svg web/apps/web/app/assets/cover-images/image_16.webp web/apps/web/app/assets/cover-images/image_17.svg web/apps/web/app/assets/cover-images/image_17.webp web/apps/web/app/assets/cover-images/image_18.svg web/apps/web/app/assets/cover-images/image_18.webp web/apps/web/app/assets/cover-images/image_19.svg web/apps/web/app/assets/cover-images/image_19.webp web/apps/web/app/assets/cover-images/image_2.svg web/apps/web/app/assets/cover-images/image_2.webp web/apps/web/app/assets/cover-images/image_20.svg web/apps/web/app/assets/cover-images/image_20.webp web/apps/web/app/assets/cover-images/image_21.svg web/apps/web/app/assets/cover-images/image_21.webp web/apps/web/app/assets/cover-images/image_22.svg web/apps/web/app/assets/cover-images/image_22.webp web/apps/web/app/assets/cover-images/image_23.svg web/apps/web/app/assets/cover-images/image_23.webp web/apps/web/app/assets/cover-images/image_24.svg web/apps/web/app/assets/cover-images/image_24.webp web/apps/web/app/assets/cover-images/image_25.svg web/apps/web/app/assets/cover-images/image_25.webp web/apps/web/app/assets/cover-images/image_26.svg web/apps/web/app/assets/cover-images/image_26.webp web/apps/web/app/assets/cover-images/image_27.svg web/apps/web/app/assets/cover-images/image_27.webp web/apps/web/app/assets/cover-images/image_28.svg web/apps/web/app/assets/cover-images/image_28.webp web/apps/web/app/assets/cover-images/image_29.svg web/apps/web/app/assets/cover-images/image_29.webp web/apps/web/app/assets/cover-images/image_3.svg web/apps/web/app/assets/cover-images/image_3.webp web/apps/web/app/assets/cover-images/image_4.svg web/apps/web/app/assets/cover-images/image_4.webp web/apps/web/app/assets/cover-images/image_5.svg web/apps/web/app/assets/cover-images/image_5.webp web/apps/web/app/assets/cover-images/image_6.svg web/apps/web/app/assets/cover-images/image_6.webp web/apps/web/app/assets/cover-images/image_7.svg web/apps/web/app/assets/cover-images/image_7.webp web/apps/web/app/assets/cover-images/image_8.svg web/apps/web/app/assets/cover-images/image_8.webp web/apps/web/app/assets/cover-images/image_9.svg web/apps/web/app/assets/cover-images/image_9.webp web/apps/web/core/components/core/image-picker-popover.tsx web/apps/web/core/components/issues/issue-layouts/list/block.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/create-project-modal.tsx web/apps/web/core/components/project/create/header.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/projects/create/root.tsx web/apps/web/core/components/projects/create/utils.ts web/apps/web/core/components/workspace/sidebar/project-navigation.tsx web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/store/favorite.store.ts web/apps/web/core/store/inbox/inbox-issue.store.ts web/apps/web/core/store/inbox/project-inbox.store.ts web/apps/web/helpers/cover-image.helper.ts web/apps/web/package.json web/packages/constants/src/file.ts web/packages/constants/src/tab-indices.ts web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/types/src/enums.ts
```
```bash
git commit -m "refactor(M3/P8b): the project pages drop what nerve's project lacks

The generated Project has no favourite, no intake count, no next work
item sequence, and its cover cannot be set until M5's uploads. The
cover picker, its 28 preset covers and their upload go; a project shows
its cover or the default one. The project card's star, the create
form's favourite and the favourite store's project branch go (M7 brings
favourites back), as do the intake count in the sidebar and the stores
that kept it. The files touched end without oxlint warnings.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有变异：它只删除，删除由 `tsc`、knip 核对（W13 的 grep 见 spec 附录 A.7）。

---

### Task 2: 生成的 `Project`、`ProjectCreate` 取代 Plane 的四个项目类型

**Files:**
- Modify: `pnpm-lock.yaml`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`、`web/apps/web/core/components/breadcrumbs/project.tsx`、`web/apps/web/core/components/common/switcher-label.tsx`、`web/apps/web/core/components/inbox/content/inbox-issue-header.tsx`、`web/apps/web/core/components/navigation/project-header.tsx`、`web/apps/web/core/components/navigation/tab-navigation-root.tsx`、`web/apps/web/core/components/navigation/use-navigation-items.ts`、`web/apps/web/core/components/power-k/config/creation/command.ts`、`web/apps/web/core/components/power-k/config/navigation/commands.ts`、`web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx`、`web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/create-project-modal.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/components/project/settings/features-list.tsx`、`web/apps/web/core/components/projects/create/attributes.tsx`、`web/apps/web/core/components/projects/create/root.tsx`、`web/apps/web/core/components/projects/create/utils.ts`、`web/apps/web/core/components/projects/settings/intake/header.tsx`、`web/apps/web/core/components/settings/project/sidebar/header.tsx`、`web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx`、`web/apps/web/core/components/workspace/sidebar/project-navigation.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/core/store/member/project/project-member.store.ts`、`web/apps/web/core/store/project/project.store.ts`、`web/apps/web/package.json`、`web/packages/propel/package.json`、`web/packages/propel/src/emoji-icon-picker/logo.tsx`、`web/packages/types/src/project/projects.ts`、`web/packages/utils/package.json`、`web/packages/utils/src/project.ts`
- 机械步骤（Step 1）改到：`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/core/components/archives/archive-tabs-list.tsx`、`web/apps/web/core/components/automation/auto-archive-automation.tsx`、`web/apps/web/core/components/automation/select-month-modal.tsx`、`web/apps/web/core/components/dropdowns/project/base.tsx`、`web/apps/web/core/components/issues/filters.tsx`、`web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx`、`web/apps/web/core/components/navigation/project-header-button.tsx`、`web/apps/web/core/components/power-k/menus/projects.tsx`、`web/apps/web/core/components/project/create/common-attributes.tsx`、`web/apps/web/core/components/project/create/header.tsx`、`web/apps/web/core/components/project/create/project-create-buttons.tsx`、`web/apps/web/core/components/project/delete-project-modal.tsx`、`web/apps/web/core/components/project/join-project-modal.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx`、`web/apps/web/core/components/settings/project/content/feature-control-item.tsx`、`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/services/project/project.service.ts`、`web/apps/web/core/store/issue/root.store.ts`、`web/packages/types/src/search.ts`、`web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts`、`web/packages/utils/src/work-item/modal.ts`

**Interfaces:**
- Produces（spec 2.2；M3 设计 7.2）：
  - `Project` 取代 `TProject`、`IProject`、`TPartialProject`、`IPartialProject`；创建表单的值是它发出的 `ProjectCreate`（`create-project-modal.tsx`、`project/create/*`、`projects/create/*`）。字段改名照生成的类型：`workspace` → `workspace_id`、`members` → `member_ids`、`project_lead` → `project_lead_id`、`default_assignee` → `default_assignee_id`、`inbox_view` → `intake_view`；`logo_props` 是生成的 `LogoProps`，`@nerve/propel` 的 `Logo` 收它（`@nerve/propel` 加工作区内的依赖 `@nerve/api-client`，锁文件一处）；`cover_image_url` 可空，读作 `null`（W20）。
  - `packages/types/src/project/projects.ts`（整个文件）只留 M3 之外的：`EUserProjectRoles`、项目成员的过渡类型（Task 6 删除）、`IProjectMemberNavigationPreferences`（Task 5 删除）、M4 的工作项搜索参数和回答。
  - `ProjectStore`：读者都已在 Task 1 删除的 `favoriteProjectIds`、`currentProjectNextSequenceId`、`addProjectToFavorites`、`removeProjectFromFavorites` 删除；`getPartialProjectById` 删除，使用方改调 `getProjectById`（两者一直给出同一个对象）。store 的其余部分仍是 Plane 的写法，Task 3 重写。
  - 项目设置的成员默认值（`project-settings-member-defaults.tsx`）的表单只有它改的两个字段（`Pick<Project, "project_lead_id" | "default_assignee_id">`），Plane 读嵌套对象的 `as MemberUser`、`as Workspace` 随之删除。
- 顺手清零（7.9，R3）：有块的文件的 oxlint 警告清零（`@nerve/utils` 的 `project.ts` 中两处 `sortBy(…).reverse()` 改为 `toReversed()`：`sortBy` 交回新的数组，顺序不变；`utils` 的上限 12 → 10）；只经 Step 1 改名的文件的警告留给 P11（spec 附录 A.8）。

**Tests:** 没有新的 vitest：类型的替换由 `tsc` 核对，行为不变。

- [ ] **Step 1: 机械步骤：Plane 的项目类型换成生成的 `Project`、`ProjectCreate`**

`$P8BTMP/rename_types.py` 见"一次性脚本"。38 个文件；其中 16 个在本 Task 另有手改（它们的块写在这一步之后的文件上），其余 22 个只经这一步改到。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . TProject,IProject,TPartialProject,IPartialProject Project 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx' web/apps/web/core/components/archives/archive-tabs-list.tsx web/apps/web/core/components/automation/auto-archive-automation.tsx web/apps/web/core/components/automation/select-month-modal.tsx web/apps/web/core/components/breadcrumbs/project.tsx web/apps/web/core/components/dropdowns/project/base.tsx web/apps/web/core/components/issues/filters.tsx web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx web/apps/web/core/components/navigation/project-header-button.tsx web/apps/web/core/components/navigation/use-navigation-items.ts web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/projects.tsx web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/delete-project-modal.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/join-project-modal.tsx web/apps/web/core/components/project/leave-project-modal.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/project/settings/features-list.tsx web/apps/web/core/components/settings/project/content/feature-control-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/services/project/project.service.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/project/project.store.ts web/packages/utils/src/project.ts web/packages/utils/src/work-item/modal.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts`
Expected: `TProject,IProject,TPartialProject,IPartialProject -> Project in 30 files`。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . IProject Project --from ./project web/packages/types/src/search.ts`
Expected: `IProject -> Project in 1 files`。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . TProject,IProject ProjectCreate web/apps/web/core/components/project/create-project-modal.tsx web/apps/web/core/components/project/create/common-attributes.tsx web/apps/web/core/components/project/create/header.tsx web/apps/web/core/components/project/create/project-create-buttons.tsx web/apps/web/core/components/projects/create/attributes.tsx web/apps/web/core/components/projects/create/root.tsx web/apps/web/core/components/projects/create/utils.ts`
Expected: `TProject,IProject -> ProjectCreate in 7 files`。

`git diff --numstat` 必须恰好是：

```text
2	2	web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
2	2	web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
2	2	web/apps/web/core/components/archives/archive-tabs-list.tsx
3	3	web/apps/web/core/components/automation/auto-archive-automation.tsx
5	5	web/apps/web/core/components/automation/select-month-modal.tsx
2	2	web/apps/web/core/components/breadcrumbs/project.tsx
3	3	web/apps/web/core/components/dropdowns/project/base.tsx
2	2	web/apps/web/core/components/issues/filters.tsx
3	2	web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx
2	2	web/apps/web/core/components/navigation/project-header-button.tsx
3	2	web/apps/web/core/components/navigation/use-navigation-items.ts
3	3	web/apps/web/core/components/power-k/config/navigation/commands.ts
3	3	web/apps/web/core/components/power-k/menus/projects.tsx
2	2	web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx
2	2	web/apps/web/core/components/project/card.tsx
2	2	web/apps/web/core/components/project/create-project-modal.tsx
3	3	web/apps/web/core/components/project/create/common-attributes.tsx
2	2	web/apps/web/core/components/project/create/header.tsx
2	2	web/apps/web/core/components/project/create/project-create-buttons.tsx
2	2	web/apps/web/core/components/project/delete-project-modal.tsx
6	7	web/apps/web/core/components/project/form.tsx
2	2	web/apps/web/core/components/project/join-project-modal.tsx
2	2	web/apps/web/core/components/project/leave-project-modal.tsx
4	5	web/apps/web/core/components/project/project-settings-member-defaults.tsx
3	3	web/apps/web/core/components/project/settings/features-list.tsx
2	2	web/apps/web/core/components/projects/create/attributes.tsx
4	4	web/apps/web/core/components/projects/create/root.tsx
2	2	web/apps/web/core/components/projects/create/utils.ts
2	2	web/apps/web/core/components/settings/project/content/feature-control-item.tsx
2	2	web/apps/web/core/components/workspace/sidebar/projects-list.tsx
3	11	web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
6	6	web/apps/web/core/services/project/project.service.ts
4	4	web/apps/web/core/store/issue/root.store.ts
23	23	web/apps/web/core/store/project/project.store.ts
8	9	web/packages/types/src/search.ts
9	8	web/packages/utils/src/project.ts
5	5	web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts
3	2	web/packages/utils/src/work-item/modal.ts
```

Run（机械步骤）: `shasum -a 256 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx' web/apps/web/core/components/archives/archive-tabs-list.tsx web/apps/web/core/components/automation/auto-archive-automation.tsx web/apps/web/core/components/automation/select-month-modal.tsx web/apps/web/core/components/breadcrumbs/project.tsx web/apps/web/core/components/dropdowns/project/base.tsx web/apps/web/core/components/issues/filters.tsx web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx web/apps/web/core/components/navigation/project-header-button.tsx web/apps/web/core/components/navigation/use-navigation-items.ts web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/projects.tsx web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/create-project-modal.tsx web/apps/web/core/components/project/create/common-attributes.tsx web/apps/web/core/components/project/create/header.tsx web/apps/web/core/components/project/create/project-create-buttons.tsx web/apps/web/core/components/project/delete-project-modal.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/join-project-modal.tsx web/apps/web/core/components/project/leave-project-modal.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/project/settings/features-list.tsx web/apps/web/core/components/projects/create/attributes.tsx web/apps/web/core/components/projects/create/root.tsx web/apps/web/core/components/projects/create/utils.ts web/apps/web/core/components/settings/project/content/feature-control-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/services/project/project.service.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/project/project.store.ts web/packages/types/src/search.ts web/packages/utils/src/project.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts web/packages/utils/src/work-item/modal.ts`
Expected: 每个文件的散列和行数与下表相同：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `eb12d4e04d947729ea5572d458c871a8e8e539d14a4b9c27cddb18c6c992437f` | 179 | `web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx` |
| `3e131d4cc18a2999d29013856c998e2fec25640aa4fb816eadfdd138caa49231` | 74 | `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx` |
| `eeaa9d3870f19e3a3fb0e5bab92a2b7b609de41422060450b042dd788d94fc0f` | 69 | `web/apps/web/core/components/archives/archive-tabs-list.tsx` |
| `cdad4a9ef3cd73b55967cda35a641631957dea0fe617539fbff089018217d897` | 134 | `web/apps/web/core/components/automation/auto-archive-automation.tsx` |
| `07e2cb2028721c3a192805298fb22ce7538dfdd539344facc467d4da8692c8ef` | 101 | `web/apps/web/core/components/automation/select-month-modal.tsx` |
| `26acf07ed81d8b6bf35de5f7e087f59ddff070e46cf0d0d65cddc57b2a35c951` | 88 | `web/apps/web/core/components/breadcrumbs/project.tsx` |
| `8b3ca6163bf964954bd53136e2950e49c495866e91a25af058fd966883cdd82f` | 293 | `web/apps/web/core/components/dropdowns/project/base.tsx` |
| `23309725244df86598749f2843f6f29c879f832e262067b89ee91c4e2ee38026` | 110 | `web/apps/web/core/components/issues/filters.tsx` |
| `9ebdb76399f70f8e23d362715b675de9a99aa2b73b163ef51c6808334e5da5d8` | 169 | `web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx` |
| `0463999c209992364b2e1508bc245eb30bff6a804b5cbcf12d4253be78f90b36` | 36 | `web/apps/web/core/components/navigation/project-header-button.tsx` |
| `e3444883779e4befbc5259e4ce9119c77f9b4420244676095ae92627301a19f9` | 107 | `web/apps/web/core/components/navigation/use-navigation-items.ts` |
| `cfeeee7ab6aab3b9db6fddefb955196f87c1a9653afb6798350d81897cfb9fb1` | 465 | `web/apps/web/core/components/power-k/config/navigation/commands.ts` |
| `d3502dbb4d3d3ea8687637c075a198e3536d289e2a01775f88118ee442e06d4f` | 36 | `web/apps/web/core/components/power-k/menus/projects.tsx` |
| `895b0e1323767cb7102dfe857e5f45df10e2f8690b8f9074152cc65ceaf066f7` | 32 | `web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx` |
| `d437770f88347e647b926c53ec5940785e532f3be77bcebe55f1afe116f710c7` | 321 | `web/apps/web/core/components/project/card.tsx` |
| `67fe0fbbda45bfe964dff22069462d61e136e7d64997569c1d030ece17b5c7cc` | 67 | `web/apps/web/core/components/project/create-project-modal.tsx` |
| `0511cef61967f7382c6f65ff36e1db70edd8cf3dae61753728d3491a2d433faf` | 168 | `web/apps/web/core/components/project/create/common-attributes.tsx` |
| `4b5ffc71ca3532a48f0a6567e1abf7512c8efec8ca4af7e670830d3cce39d2dc` | 94 | `web/apps/web/core/components/project/create/header.tsx` |
| `0d8ed4d07d6f95af4748ee8494339a7345f398104e7f0fd7688b3ce0bdeedd8c` | 42 | `web/apps/web/core/components/project/create/project-create-buttons.tsx` |
| `3794e19f24e82a0dc8f53a0471d027e80d1fe0744677774a326b74062467fbdc` | 159 | `web/apps/web/core/components/project/delete-project-modal.tsx` |
| `c41c44f1d98f53edfa9554714b084364337bcdc5e6e52d8ef2b0a046cd6f3638` | 422 | `web/apps/web/core/components/project/form.tsx` |
| `6ebbed7deb0f1bca651dda6d1741066192c15c51c8da0d8c67c4188f62a5094b` | 71 | `web/apps/web/core/components/project/join-project-modal.tsx` |
| `4c11a180658e3f4a0e5d0228356ac7687f8636e6a91296a07200ed6482bf26b8` | 183 | `web/apps/web/core/components/project/leave-project-modal.tsx` |
| `19f614a825b4abb55ebbaa7b51fd2e812e903bb91fdc3c616daeddd29047a46d` | 203 | `web/apps/web/core/components/project/project-settings-member-defaults.tsx` |
| `baa2c90776dc7b2555943b4f6bcc94c6f75b7266ae782a9a7628d7715ee1623d` | 103 | `web/apps/web/core/components/project/settings/features-list.tsx` |
| `73b8fe6e4e103051dc9557e6a4336ca172e0c6276c460565ddf93fe6cb1c1bf2` | 98 | `web/apps/web/core/components/projects/create/attributes.tsx` |
| `2f297bf1294f031b226da935b54bd37b5e64ddd39c1fe01b374e39dd2cac1925` | 136 | `web/apps/web/core/components/projects/create/root.tsx` |
| `04791674aad5db7f1035cbd534b2fe80a5eb8eb8ed16033b51920c5a186326e4` | 22 | `web/apps/web/core/components/projects/create/utils.ts` |
| `1f2f4fabddadafed60ad5613fbe3229a08b25e445b938b552ac196f5c681df66` | 74 | `web/apps/web/core/components/settings/project/content/feature-control-item.tsx` |
| `3f928d99163727de11d6bcf3e52f86b6b5365a6c62dc15c8c5e0e0375eac116f` | 290 | `web/apps/web/core/components/workspace/sidebar/projects-list.tsx` |
| `e22b5d10fbc5df350835971aa007921e0b264274144989498d4eadffcf0aa9b6` | 396 | `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx` |
| `3e385606115d822076e4e1fd6db64b1e5fe9546c40d413e46e3a612196fcb4f2` | 132 | `web/apps/web/core/services/project/project.service.ts` |
| `c45cb999c430a7ed198935cd251917368fa78de21aa81e390b7dc72909d69656` | 217 | `web/apps/web/core/store/issue/root.store.ts` |
| `e6432888186797415a10ada4dc15e3b23438d56306e3b60eb3f32347b66b1571` | 558 | `web/apps/web/core/store/project/project.store.ts` |
| `9729a9f9c0e41654595dbc38312d29c6e6ff070b49b0220b008a899ac61f0a84` | 71 | `web/packages/types/src/search.ts` |
| `c5db847bf3762cad4e40877947fb174059fd4cfb61aae127d914de3d57cb07b9` | 111 | `web/packages/utils/src/project.ts` |
| `0a576f2822e21427d700226535e6a90ca30c2b6278158f6653b8deba20ead566` | 93 | `web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts` |
| `26edcd745dd2268e3b189ccdf9bbafd880af77f424d9d42f1d8df77da4bc20d8` | 53 | `web/packages/utils/src/work-item/modal.ts` |

- [ ] **Step 2: 依赖与类型**

`pnpm-lock.yaml`（修改，1 处）：

````old pnpm-lock.yaml
        version: 1.0.0-beta.3(@types/react@19.2.17)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)
      '@makeplane/propel':
        specifier: 'catalog:'
        version: 0.3.0(@date-fns/tz@1.4.1)(@types/react@19.2.17)(date-fns@4.1.0)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)(tailwindcss@4.1.17)
      '@nerve/constants':
        specifier: workspace:*
````
````new pnpm-lock.yaml
        version: 1.0.0-beta.3(@types/react@19.2.17)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)
      '@makeplane/propel':
        specifier: 'catalog:'
        version: 0.3.0(@date-fns/tz@1.4.1)(@types/react@19.2.17)(date-fns@4.1.0)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)(tailwindcss@4.1.17)
      '@nerve/api-client':
        specifier: workspace:*
        version: link:../api-client
      '@nerve/constants':
        specifier: workspace:*
````

`web/packages/propel/package.json`（修改，1 处）：

````old web/packages/propel/package.json
    "@makeplane/propel": "catalog:",
````
````new web/packages/propel/package.json
    "@makeplane/propel": "catalog:",
    "@nerve/api-client": "workspace:*",
````

`web/packages/propel/src/emoji-icon-picker/logo.tsx`（修改，2 处）：

````old web/packages/propel/src/emoji-icon-picker/logo.tsx
import type { TLogoProps } from "@nerve/types";
````
````new web/packages/propel/src/emoji-icon-picker/logo.tsx
import type { LogoProps } from "@nerve/api-client";
````
````old web/packages/propel/src/emoji-icon-picker/logo.tsx
  logo?: TLogoProps;
````
````new web/packages/propel/src/emoji-icon-picker/logo.tsx
  logo?: LogoProps;
````

`web/packages/types/src/project/projects.ts`（整个文件，65 行）：

````whole web/packages/types/src/project/projects.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TUserPermissions } from "../enums";
import type { TStateGroups } from "../state";

export enum EUserProjectRoles {
  ADMIN = 20,
  MEMBER = 15,
  GUEST = 5,
}

export type TProjectMembership = {
  member: string;
  role: TUserPermissions | EUserProjectRoles;
} & (
  | {
      id: string;
      original_role: EUserProjectRoles;
      created_at: string;
    }
  | {
      id: null;
      original_role: null;
      created_at: null;
    }
);

export interface IProjectBulkAddFormData {
  members: { role: TUserPermissions | EUserProjectRoles; member_id: string }[];
}

export type IProjectMemberNavigationPreferences = {
  default_tab: string;
  hide_in_more_menu: string[];
};

export type TProjectIssuesSearchParams = {
  search: string;
  parent?: boolean;
  issue_relation?: boolean;
  cycle?: boolean;
  module?: string;
  sub_issue?: boolean;
  issue_id?: string;
  workspace_search: boolean;
  target_date?: string;
};

export interface ISearchIssueResponse {
  id: string;
  name: string;
  project_id: string;
  project__identifier: string;
  project__name: string;
  sequence_id: number;
  start_date: string | null;
  state__color: string;
  state__group: TStateGroups;
  state__name: string;
  workspace__slug: string;
}
````

`web/packages/utils/package.json`（修改，1 处）：

````old web/packages/utils/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 12",
````
````new web/packages/utils/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 10",
````

`web/packages/utils/src/project.ts`（修改，4 处）：

````old web/packages/utils/src/project.ts
      fallsInFilters = fallsInFilters && filters.lead.includes(`${project.project_lead}`);
````
````new web/packages/utils/src/project.ts
      fallsInFilters = fallsInFilters && filters.lead.includes(`${project.project_lead_id}`);
````
````old web/packages/utils/src/project.ts
      const memberIds = project.members;
      fallsInFilters = fallsInFilters && filters.members.some((memberId) => memberIds?.includes(memberId));
````
````new web/packages/utils/src/project.ts
      fallsInFilters = fallsInFilters && filters.members.some((memberId) => project.member_ids.includes(memberId));
````
````old web/packages/utils/src/project.ts
  if (orderByKey === "-name") orderedProjects = sortBy(projects, [(p) => p.name.toLowerCase()]).reverse();
````
````new web/packages/utils/src/project.ts
  if (orderByKey === "-name") orderedProjects = sortBy(projects, [(p) => p.name.toLowerCase()]).toReversed();
````
````old web/packages/utils/src/project.ts
  if (orderByKey === "members_length") orderedProjects = sortBy(projects, [(p) => p.members?.length]);
  if (orderByKey === "-members_length") orderedProjects = sortBy(projects, [(p) => p.members?.length]).reverse();
````
````new web/packages/utils/src/project.ts
  if (orderByKey === "members_length") orderedProjects = sortBy(projects, [(p) => p.member_ids.length]);
  if (orderByKey === "-members_length") orderedProjects = sortBy(projects, [(p) => p.member_ids.length]).toReversed();
````

- [ ] **Step 3: store**

`web/apps/web/core/store/member/project/project-member.store.ts`（修改，3 处）：

````old web/apps/web/core/store/member/project/project-member.store.ts
      update(this.projectRoot.projectMap, [projectId, "members"], (memberIds) =>
````
````new web/apps/web/core/store/member/project/project-member.store.ts
      update(this.projectRoot.projectMap, [projectId, "member_ids"], (memberIds) =>
````
````old web/apps/web/core/store/member/project/project-member.store.ts
      this.projectRoot.projectMap[projectId].members = this.projectRoot.projectMap?.[projectId]?.members?.concat(
````
````new web/apps/web/core/store/member/project/project-member.store.ts
      this.projectRoot.projectMap[projectId].member_ids = this.projectRoot.projectMap[projectId].member_ids.concat(
````
````old web/apps/web/core/store/member/project/project-member.store.ts
      [projectId, "members"],
      this.projectRoot.projectMap?.[projectId]?.members?.filter((memberId) => memberId !== userId)
````
````new web/apps/web/core/store/member/project/project-member.store.ts
      [projectId, "member_ids"],
      this.projectRoot.projectMap?.[projectId]?.member_ids.filter((memberId) => memberId !== userId)
````

`web/apps/web/core/store/project/project.store.ts`（修改，15 处）：

````old web/apps/web/core/store/project/project.store.ts
  joinedProjectIds: string[];
  favoriteProjectIds: string[];
  currentProjectDetails: Project | undefined;
  currentProjectNextSequenceId: number | undefined;
````
````new web/apps/web/core/store/project/project.store.ts
  joinedProjectIds: string[];
  currentProjectDetails: Project | undefined;
````
````old web/apps/web/core/store/project/project.store.ts
  getProjectById: (projectId: string | undefined | null) => Project | undefined;
  getPartialProjectById: (projectId: string | undefined | null) => Project | undefined;
````
````new web/apps/web/core/store/project/project.store.ts
  getProjectById: (projectId: string | undefined | null) => Project | undefined;
````
````old web/apps/web/core/store/project/project.store.ts
  fetchProjectDetails: (workspaceSlug: string, projectId: string) => Promise<Project>;
  // favorites actions
  addProjectToFavorites: (workspaceSlug: string, projectId: string) => Promise<any>;
  removeProjectFromFavorites: (workspaceSlug: string, projectId: string) => Promise<any>;
````
````new web/apps/web/core/store/project/project.store.ts
  fetchProjectDetails: (workspaceSlug: string, projectId: string) => Promise<Project>;
````
````old web/apps/web/core/store/project/project.store.ts
      joinedProjectIds: computed,
      favoriteProjectIds: computed,
      currentProjectNextSequenceId: computed,
````
````new web/apps/web/core/store/project/project.store.ts
      joinedProjectIds: computed,
````
````old web/apps/web/core/store/project/project.store.ts
      fetchProjectDetails: action,
      // favorites actions
      addProjectToFavorites: action,
      removeProjectFromFavorites: action,
````
````new web/apps/web/core/store/project/project.store.ts
      fetchProjectDetails: action,
````
````old web/apps/web/core/store/project/project.store.ts
        p.workspace === workspaceDetails.id &&
````
````new web/apps/web/core/store/project/project.store.ts
        p.workspace_id === workspaceDetails.id &&
````
````old web/apps/web/core/store/project/project.store.ts
      (p) => p.workspace === workspaceDetails.id && !p.archived_at
````
````new web/apps/web/core/store/project/project.store.ts
      (p) => p.workspace_id === workspaceDetails.id && !p.archived_at
````
````old web/apps/web/core/store/project/project.store.ts
      .filter((project) => project.workspace === currentWorkspace.id && !!project.archived_at)
````
````new web/apps/web/core/store/project/project.store.ts
      .filter((project) => project.workspace_id === currentWorkspace.id && !!project.archived_at)
````
````old web/apps/web/core/store/project/project.store.ts
  /**
   * Returns the next sequence ID for the current project
   * Used for calculating identifier width in list layouts
   */
  get currentProjectNextSequenceId() {
    if (!this.rootStore.router.projectId) return undefined;
    return this.currentProjectDetails?.next_work_item_sequence;
  }

  /**
````
````new web/apps/web/core/store/project/project.store.ts
  /**
````
````old web/apps/web/core/store/project/project.store.ts
    const projectIds = projects
      .filter((project) => project.workspace === currentWorkspace.id && !!project.member_role && !project.archived_at)
      .map((project) => project.id);
    return projectIds;
  }

  /**
   * Returns favorite project IDs belong to the current workspace
   */
  get favoriteProjectIds() {
    const currentWorkspace = this.rootStore.workspaceRoot.currentWorkspace;
    if (!currentWorkspace) return [];

    let projects = Object.values(this.projectMap ?? {});
    projects = sortBy(projects, "created_at");

    const projectIds = projects
````
````new web/apps/web/core/store/project/project.store.ts
    const projectIds = projects
````
````old web/apps/web/core/store/project/project.store.ts
        (project) =>
          project.workspace === currentWorkspace.id &&
          !!project.member_role &&
          project.is_favorite &&
          !project.archived_at
````
````new web/apps/web/core/store/project/project.store.ts
        (project) => project.workspace_id === currentWorkspace.id && !!project.member_role && !project.archived_at
````
````old web/apps/web/core/store/project/project.store.ts
  /**
   * Returns project lite using project id
   * This method is used just for type safety
   * @param projectId
   * @returns Project | null
   */
  getPartialProjectById = computedFn((projectId: string | undefined | null) => {
    const projectInfo = this.projectMap[projectId ?? ""] || undefined;
    return projectInfo;
  });

  /**
````
````new web/apps/web/core/store/project/project.store.ts
  /**
````
````old web/apps/web/core/store/project/project.store.ts
  });

  /**
   * Adds project to favorites and updates project favorite status in the store
   * @param workspaceSlug
   * @param projectId
   * @returns
   */
  addProjectToFavorites = async (workspaceSlug: string, projectId: string) => {
    try {
      const currentProject = this.getProjectById(projectId);
      if (currentProject.is_favorite) return;
      runInAction(() => {
        set(this.projectMap, [projectId, "is_favorite"], true);
      });
      const response = await this.rootStore.favorite.addFavorite(workspaceSlug, {
        entity_type: "project",
        entity_identifier: projectId,
        project_id: projectId,
        entity_data: { name: this.projectMap[projectId].name || "" },
      });
      return response;
    } catch (error) {
      console.log("Failed to add project to favorite");
      runInAction(() => {
        set(this.projectMap, [projectId, "is_favorite"], false);
      });
      throw error;
    }
  };

  /**
   * Removes project from favorites and updates project favorite status in the store
   * @param workspaceSlug
   * @param projectId
   * @returns
   */
  removeProjectFromFavorites = async (workspaceSlug: string, projectId: string) => {
    try {
      const currentProject = this.getProjectById(projectId);
      if (!currentProject.is_favorite) return;
      runInAction(() => {
        set(this.projectMap, [projectId, "is_favorite"], false);
      });
      const response = await this.rootStore.favorite.removeFavoriteEntity(workspaceSlug, projectId);

      return response;
    } catch (error) {
      console.log("Failed to add project to favorite");
      runInAction(() => {
        set(this.projectMap, [projectId, "is_favorite"], true);
      });
      throw error;
    }
  };
````
````new web/apps/web/core/store/project/project.store.ts
  });
````
````old web/apps/web/core/store/project/project.store.ts
    await this.projectArchiveService
      .archiveProject(workspaceSlug, projectId)
      .then((response) => {
        runInAction(() => {
          set(this.projectMap, [projectId, "archived_at"], response.archived_at);
          this.rootStore.favorite.removeFavoriteFromStore(projectId);
        });
      })
      .catch((error) => {
        console.log("Failed to archive project from project store");
        throw error;
      });
````
````new web/apps/web/core/store/project/project.store.ts
    const response = await this.projectArchiveService.archiveProject(workspaceSlug, projectId);
    runInAction(() => {
      set(this.projectMap, [projectId, "archived_at"], response.archived_at);
      this.rootStore.favorite.removeFavoriteFromStore(projectId);
    });
````
````old web/apps/web/core/store/project/project.store.ts
    await this.projectArchiveService
      .restoreProject(workspaceSlug, projectId)
      .then(() => {
        runInAction(() => {
          set(this.projectMap, [projectId, "archived_at"], null);
        });
      })
      .catch((error) => {
        console.log("Failed to restore project from project store");
        throw error;
      });
````
````new web/apps/web/core/store/project/project.store.ts
    await this.projectArchiveService.restoreProject(workspaceSlug, projectId);
    runInAction(() => {
      set(this.projectMap, [projectId, "archived_at"], null);
    });
````

- [ ] **Step 4: 使用方**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const { getPartialProjectById, joinedProjectIds: joinedProjects, updateProjectView } = useProject();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const { getProjectById, joinedProjectIds: joinedProjects, updateProjectView } = useProject();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
      const projectDetails = getPartialProjectById(projectId);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
      const projectDetails = getProjectById(projectId);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    const project = getPartialProjectById(projectId);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    const project = getProjectById(projectId);
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx
  if (currentProjectDetails?.inbox_view === false)
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx
  if (currentProjectDetails?.intake_view === false)
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx
          inboxAccessible={currentProjectDetails?.inbox_view || false}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx
          inboxAccessible={currentProjectDetails?.intake_view || false}
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
            featureProperty="inbox_view"
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
            featureProperty="intake_view"
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
            value={!!currentProjectDetails?.inbox_view}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx
            value={!!currentProjectDetails?.intake_view}
````

`web/apps/web/core/components/breadcrumbs/project.tsx`（修改，5 处）：

````old web/apps/web/core/components/breadcrumbs/project.tsx
  projectId: string;
  handleOnClick?: () => void;
````
````new web/apps/web/core/components/breadcrumbs/project.tsx
  projectId: string;
````
````old web/apps/web/core/components/breadcrumbs/project.tsx
  const { workspaceSlug, projectId, handleOnClick } = props;
````
````new web/apps/web/core/components/breadcrumbs/project.tsx
  const { workspaceSlug, projectId } = props;
````
````old web/apps/web/core/components/breadcrumbs/project.tsx
  const { joinedProjectIds, getPartialProjectById } = useProject();
  const currentProjectDetails = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/breadcrumbs/project.tsx
  const { joinedProjectIds, getProjectById } = useProject();
  const currentProjectDetails = getProjectById(projectId);
````
````old web/apps/web/core/components/breadcrumbs/project.tsx
      const project = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/breadcrumbs/project.tsx
      const project = getProjectById(projectId);
````
````old web/apps/web/core/components/breadcrumbs/project.tsx
            handleOnClick={() => {
              if (handleOnClick) handleOnClick();
              else navigate(`/${workspaceSlug}/projects/${currentProjectDetails.id}/issues`);
            }}
````
````new web/apps/web/core/components/breadcrumbs/project.tsx
            handleOnClick={() => navigate(`/${workspaceSlug}/projects/${currentProjectDetails.id}/issues`)}
````

`web/apps/web/core/components/common/switcher-label.tsx`（修改，4 处）：

````old web/apps/web/core/components/common/switcher-label.tsx
import type { FC } from "react";
````
````new web/apps/web/core/components/common/switcher-label.tsx
import type { FC } from "react";
import type { LogoProps } from "@nerve/api-client";
````
````old web/apps/web/core/components/common/switcher-label.tsx
import type { ISvgIcons } from "@nerve/propel/icons";
import type { TLogoProps } from "@nerve/types";
````
````new web/apps/web/core/components/common/switcher-label.tsx
import type { ISvgIcons } from "@nerve/propel/icons";
````
````old web/apps/web/core/components/common/switcher-label.tsx
type TSwitcherIconProps = {
  logo_props?: TLogoProps;
````
````new web/apps/web/core/components/common/switcher-label.tsx
type TSwitcherIconProps = {
  logo_props?: LogoProps;
````
````old web/apps/web/core/components/common/switcher-label.tsx
type TSwitcherLabelProps = {
  logo_props?: TLogoProps;
````
````new web/apps/web/core/components/common/switcher-label.tsx
type TSwitcherLabelProps = {
  logo_props?: LogoProps;
````

`web/apps/web/core/components/inbox/content/inbox-issue-header.tsx`（修改，3 处）：

````old web/apps/web/core/components/inbox/content/inbox-issue-header.tsx
  const { getPartialProjectById } = useProject();
  const currentProjectDetails = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/inbox/content/inbox-issue-header.tsx
  const { getProjectById } = useProject();
  const currentProjectDetails = getProjectById(projectId);
````
````old web/apps/web/core/components/inbox/content/inbox-issue-header.tsx
  const navigate = useNavigate();
  const { getProjectById } = useProject();
````
````new web/apps/web/core/components/inbox/content/inbox-issue-header.tsx
  const navigate = useNavigate();
````
````old web/apps/web/core/components/inbox/content/inbox-issue-header.tsx
    await deleteInboxIssue(workspaceSlug, projectId, currentInboxIssueId).then(() => {
      if (!isNotificationEmbed) navigate(`/${workspaceSlug}/projects/${projectId}/intake`);
    });
````
````new web/apps/web/core/components/inbox/content/inbox-issue-header.tsx
    await deleteInboxIssue(workspaceSlug, projectId, currentInboxIssueId);
    if (!isNotificationEmbed) navigate(`/${workspaceSlug}/projects/${projectId}/intake`);
````

`web/apps/web/core/components/navigation/project-header.tsx`（修改，4 处）：

````old web/apps/web/core/components/navigation/project-header.tsx
  const { joinedProjectIds, getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/navigation/project-header.tsx
  const { joinedProjectIds, getProjectById } = useProject();
````
````old web/apps/web/core/components/navigation/project-header.tsx
  const currentProjectDetails = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/navigation/project-header.tsx
  const currentProjectDetails = getProjectById(projectId);
````
````old web/apps/web/core/components/navigation/project-header.tsx
          const project = getPartialProjectById(id);
````
````new web/apps/web/core/components/navigation/project-header.tsx
          const project = getProjectById(id);
````
````old web/apps/web/core/components/navigation/project-header.tsx
    [joinedProjectIds, getPartialProjectById]
````
````new web/apps/web/core/components/navigation/project-header.tsx
    [joinedProjectIds, getProjectById]
````

`web/apps/web/core/components/navigation/tab-navigation-root.tsx`（修改，2 处）：

````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const { getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const { getProjectById } = useProject();
````
````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const project = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const project = getProjectById(projectId);
````

`web/apps/web/core/components/navigation/use-navigation-items.ts`（修改，1 处）：

````old web/apps/web/core/components/navigation/use-navigation-items.ts
        shouldRender: !!project?.inbox_view,
````
````new web/apps/web/core/components/navigation/use-navigation-items.ts
        shouldRender: !!project?.intake_view,
````

`web/apps/web/core/components/power-k/config/creation/command.ts`（修改，2 处）：

````old web/apps/web/core/components/power-k/config/creation/command.ts
  const { workspaceProjectIds, getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/power-k/config/creation/command.ts
  const { workspaceProjectIds, getProjectById } = useProject();
````
````old web/apps/web/core/components/power-k/config/creation/command.ts
    ctx.params.projectId ? getPartialProjectById(ctx.params.projectId) : undefined;
````
````new web/apps/web/core/components/power-k/config/creation/command.ts
    ctx.params.projectId ? getProjectById(ctx.params.projectId) : undefined;
````

`web/apps/web/core/components/power-k/config/navigation/commands.ts`（修改，4 处）：

````old web/apps/web/core/components/power-k/config/navigation/commands.ts
  | "nav_project_settings";

````
````new web/apps/web/core/components/power-k/config/navigation/commands.ts
  | "nav_project_settings";

const baseWorkspaceConditions = (ctx: TPowerKContext) => Boolean(ctx.params.workspaceSlug);
const baseProjectConditions = (ctx: TPowerKContext) => Boolean(ctx.params.workspaceSlug && ctx.params.projectId);

````
````old web/apps/web/core/components/power-k/config/navigation/commands.ts
  const { getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/power-k/config/navigation/commands.ts
  const { getProjectById } = useProject();
````
````old web/apps/web/core/components/power-k/config/navigation/commands.ts
  const baseWorkspaceConditions = (ctx: TPowerKContext) => Boolean(ctx.params.workspaceSlug);
  const baseProjectConditions = (ctx: TPowerKContext) => Boolean(ctx.params.workspaceSlug && ctx.params.projectId);
  const getContextProject = (ctx: TPowerKContext) => getPartialProjectById(ctx.params.projectId);
````
````new web/apps/web/core/components/power-k/config/navigation/commands.ts
  const getContextProject = (ctx: TPowerKContext) => getProjectById(ctx.params.projectId);
````
````old web/apps/web/core/components/power-k/config/navigation/commands.ts
      isEnabled: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.inbox_view,
      isVisible: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.inbox_view,
````
````new web/apps/web/core/components/power-k/config/navigation/commands.ts
      isEnabled: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.intake_view,
      isVisible: (ctx) => baseProjectConditions(ctx) && !!getContextProject(ctx)?.intake_view,
````

`web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx`（修改，2 处）：

````old web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx
  const { loader, joinedProjectIds, getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx
  const { loader, joinedProjectIds, getProjectById } = useProject();
````
````old web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx
    ? joinedProjectIds.map((id) => getPartialProjectById(id)).filter((project) => project !== undefined)
````
````new web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx
    ? joinedProjectIds.map((id) => getProjectById(id)).filter((project) => project !== undefined)
````

`web/apps/web/core/components/project/card.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/card.tsx
  const projectMembersIds = project.members;
````
````new web/apps/web/core/components/project/card.tsx
  const projectMembersIds = project.member_ids;
````
````old web/apps/web/core/components/project/card.tsx
            src={project.cover_image_url}
````
````new web/apps/web/core/components/project/card.tsx
            src={project.cover_image_url ?? undefined}
````
````old web/apps/web/core/components/project/card.tsx
                label={project.members?.length ? `Members: ${project.members.length}` : "No members"}
````
````new web/apps/web/core/components/project/card.tsx
                label={projectMembersIds.length ? `Members: ${projectMembersIds.length}` : "No members"}
````
````old web/apps/web/core/components/project/card.tsx
                {projectMembersIds && projectMembersIds.length > 0 ? (
````
````new web/apps/web/core/components/project/card.tsx
                {projectMembersIds.length > 0 ? (
````

`web/apps/web/core/components/project/create-project-modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/create-project-modal.tsx
import { CreateProjectForm } from "@/components/projects/create/root";
// nerve imports
import type { ProjectCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/create-project-modal.tsx
import { CreateProjectForm } from "@/components/projects/create/root";
````
````old web/apps/web/core/components/project/create-project-modal.tsx
  workspaceSlug: string;
  data?: Partial<ProjectCreate>;
````
````new web/apps/web/core/components/project/create-project-modal.tsx
  workspaceSlug: string;
````
````old web/apps/web/core/components/project/create-project-modal.tsx
  const { isOpen, onClose, workspaceSlug, data } = props;
````
````new web/apps/web/core/components/project/create-project-modal.tsx
  const { isOpen, onClose, workspaceSlug } = props;
````
````old web/apps/web/core/components/project/create-project-modal.tsx
        <CreateProjectForm
          workspaceSlug={workspaceSlug}
          onClose={onClose}
          handleNextStep={handleNextStep}
          data={data}
        />
````
````new web/apps/web/core/components/project/create-project-modal.tsx
        <CreateProjectForm workspaceSlug={workspaceSlug} onClose={onClose} handleNextStep={handleNextStep} />
````

`web/apps/web/core/components/project/form.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/form.tsx
import type { Workspace, Project } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/form.tsx
import type { Project } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/form.tsx
  } = useForm<Project>({
    defaultValues: {
      ...project,
      workspace: (project.workspace as Workspace).id,
    },
  });
````
````new web/apps/web/core/components/project/form.tsx
  } = useForm<Project>({ defaultValues: project });
````
````old web/apps/web/core/components/project/form.tsx
      reset({
        ...project,
        workspace: (project.workspace as Workspace).id,
      });
````
````new web/apps/web/core/components/project/form.tsx
      reset(project);
````
````old web/apps/web/core/components/project/form.tsx
          src={project.cover_image_url}
````
````new web/apps/web/core/components/project/form.tsx
          src={project.cover_image_url ?? undefined}
````

`web/apps/web/core/components/project/project-settings-member-defaults.tsx`（修改，12 处）：

````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
import type { Workspace, MemberUser, Project } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
import type { Project } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
const defaultValues: Partial<Project> = {
  project_lead: null,
  default_assignee: null,
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
type TMemberDefaults = Pick<Project, "project_lead_id" | "default_assignee_id">;

const defaultValues: TMemberDefaults = {
  project_lead_id: null,
  default_assignee_id: null,
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { reset, control } = useForm<Project>({ defaultValues });
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { reset, control } = useForm<TMemberDefaults>({ defaultValues });
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
      ...currentProjectDetails,
      default_assignee:
        (currentProjectDetails.default_assignee as MemberUser)?.id ?? currentProjectDetails.default_assignee,
      project_lead: (currentProjectDetails.project_lead as MemberUser)?.id ?? currentProjectDetails.project_lead,
      workspace: (currentProjectDetails.workspace as Workspace).id,
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
      project_lead_id: currentProjectDetails.project_lead_id,
      default_assignee_id: currentProjectDetails.default_assignee_id,
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const submitChanges = async (formData: Partial<Project>) => {
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const submitChanges = async (formData: Partial<TMemberDefaults>) => {
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
      ...currentProjectDetails,
      default_assignee:
        (currentProjectDetails?.default_assignee as MemberUser)?.id ?? currentProjectDetails?.default_assignee,
      project_lead: (currentProjectDetails?.project_lead as MemberUser)?.id ?? currentProjectDetails?.project_lead,
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
      project_lead_id: currentProjectDetails?.project_lead_id ?? null,
      default_assignee_id: currentProjectDetails?.default_assignee_id ?? null,
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
    await updateProject(workspaceSlug, projectId, {
      default_assignee:
        formData.default_assignee === "none"
          ? null
          : (formData.default_assignee ?? currentProjectDetails?.default_assignee),
      project_lead:
        formData.project_lead === "none" ? null : (formData.project_lead ?? currentProjectDetails?.project_lead),
    })
      .then(() => {
        setToast({
          title: `${t("success")}!`,
          type: TOAST_TYPE.SUCCESS,
          message: t("project_settings.general.toast.success"),
        });
      })
      .catch((err) => {
        console.error(err);
      });
  };

  const toggleGuestViewAllIssues = async (value: boolean) => {
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
    try {
      await updateProject(workspaceSlug, projectId, {
        default_assignee_id:
          formData.default_assignee_id === "none"
            ? null
            : (formData.default_assignee_id ?? currentProjectDetails?.default_assignee_id),
        project_lead_id:
          formData.project_lead_id === "none"
            ? null
            : (formData.project_lead_id ?? currentProjectDetails?.project_lead_id),
      });
      setToast({
        title: `${t("success")}!`,
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      });
    } catch (err) {
      console.error(err);
    }
  };

  const toggleGuestViewAllIssues = async (value: boolean) => {
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
    updateProject(workspaceSlug, projectId, {
      guest_view_all_features: value,
    })
      .then(() => {
        setToast({
          title: `${t("success")}!`,
          type: TOAST_TYPE.SUCCESS,
          message: t("project_settings.general.toast.success"),
        });
      })
      .catch((err) => {
        console.error(err);
      });
  };

  return (
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
    try {
      await updateProject(workspaceSlug, projectId, {
        guest_view_all_features: value,
      });
      setToast({
        title: `${t("success")}!`,
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      });
    } catch (err) {
      console.error(err);
    }
  };

  return (
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
            name="project_lead"
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
            name="project_lead_id"
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
                  submitChanges({ project_lead: val });
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
                  submitChanges({ project_lead_id: val });
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
            name="default_assignee"
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
            name="default_assignee_id"
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
                  submitChanges({ default_assignee: val });
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
                  submitChanges({ default_assignee_id: val });
````

`web/apps/web/core/components/project/settings/features-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/settings/features-list.tsx
    property: "inbox_view",
````
````new web/apps/web/core/components/project/settings/features-list.tsx
    property: "intake_view",
````

`web/apps/web/core/components/projects/create/attributes.tsx`（修改，2 处）：

````old web/apps/web/core/components/projects/create/attributes.tsx
        name="project_lead"
````
````new web/apps/web/core/components/projects/create/attributes.tsx
        name="project_lead_id"
````
````old web/apps/web/core/components/projects/create/attributes.tsx
        render={({ field: { value, onChange } }) => {
          if (value === undefined || value === null || typeof value === "string")
            return (
              <div className="h-7 flex-shrink-0" tabIndex={getIndex("lead")}>
                <MemberDropdown
                  value={value ?? null}
                  onChange={(lead) => onChange(lead === value ? null : lead)}
                  placeholder={t("lead")}
                  multiple={false}
                  buttonVariant="border-with-text"
                  tabIndex={getIndex("lead")}
                />
              </div>
            );
          else return <></>;
        }}
````
````new web/apps/web/core/components/projects/create/attributes.tsx
        render={({ field: { value, onChange } }) => (
          <div className="h-7 flex-shrink-0" tabIndex={getIndex("lead")}>
            <MemberDropdown
              value={value ?? null}
              onChange={(lead) => onChange(lead === value ? undefined : lead)}
              placeholder={t("lead")}
              multiple={false}
              buttonVariant="border-with-text"
              tabIndex={getIndex("lead")}
            />
          </div>
        )}
````

`web/apps/web/core/components/projects/create/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/projects/create/root.tsx
  handleNextStep: (projectId: string) => void;
  data?: Partial<ProjectCreate>;
````
````new web/apps/web/core/components/projects/create/root.tsx
  handleNextStep: (projectId: string) => void;
````
````old web/apps/web/core/components/projects/create/root.tsx
  const { workspaceSlug, data, onClose, handleNextStep } = props;
````
````new web/apps/web/core/components/projects/create/root.tsx
  const { workspaceSlug, onClose, handleNextStep } = props;
````
````old web/apps/web/core/components/projects/create/root.tsx
    defaultValues: { ...getProjectFormValues(), ...data },
````
````new web/apps/web/core/components/projects/create/root.tsx
    defaultValues: getProjectFormValues(),
````
````old web/apps/web/core/components/projects/create/root.tsx
  const onSubmit = async (formData: Partial<ProjectCreate>) => {
````
````new web/apps/web/core/components/projects/create/root.tsx
  const onSubmit = async (formData: ProjectCreate) => {
````
````old web/apps/web/core/components/projects/create/root.tsx
    return createProject(workspaceSlug, { ...formData, identifier: formData.identifier?.toUpperCase() })
````
````new web/apps/web/core/components/projects/create/root.tsx
    return createProject(workspaceSlug, { ...formData, identifier: formData.identifier.toUpperCase() })
````

`web/apps/web/core/components/projects/create/utils.ts`（修改，2 处）：

````old web/apps/web/core/components/projects/create/utils.ts
export const getProjectFormValues = (): Partial<ProjectCreate> => ({
````
````new web/apps/web/core/components/projects/create/utils.ts
export const getProjectFormValues = (): ProjectCreate => ({
````
````old web/apps/web/core/components/projects/create/utils.ts
  network: 2,
  project_lead: null,
````
````new web/apps/web/core/components/projects/create/utils.ts
  network: 2,
````

`web/apps/web/core/components/projects/settings/intake/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/projects/settings/intake/header.tsx
        {currentProjectDetails?.inbox_view && isAuthorized ? (
````
````new web/apps/web/core/components/projects/settings/intake/header.tsx
        {currentProjectDetails?.intake_view && isAuthorized ? (
````

`web/apps/web/core/components/settings/project/sidebar/header.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/project/sidebar/header.tsx
  const { getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/settings/project/sidebar/header.tsx
  const { getProjectById } = useProject();
````
````old web/apps/web/core/components/settings/project/sidebar/header.tsx
  const projectDetails = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/settings/project/sidebar/header.tsx
  const projectDetails = getProjectById(projectId);
````

`web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx
import type { TFavoriteEntityType, TLogoProps } from "@nerve/types";
````
````new web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx
import type { LogoProps } from "@nerve/api-client";
import type { TFavoriteEntityType } from "@nerve/types";
````
````old web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx
  logo?: TLogoProps;
````
````new web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx
  logo?: LogoProps;
````

`web/apps/web/core/components/workspace/sidebar/project-navigation.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
  const { getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
  const { getProjectById } = useProject();
````
````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
  const project = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
  const project = getProjectById(projectId);
````
````old web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
        shouldRender: project?.inbox_view ?? false,
````
````new web/apps/web/core/components/workspace/sidebar/project-navigation.tsx
        shouldRender: project?.intake_view ?? false,
````

`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const { getPartialProjectById } = useProject();
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const { getProjectById } = useProject();
````
````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const project = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const project = getProjectById(projectId);
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
  const { loader, getPartialProjectById, joinedProjectIds: joinedProjects, updateProjectView } = useProject();
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
  const { loader, getProjectById, joinedProjectIds: joinedProjects, updateProjectView } = useProject();
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
      const projectDetails = getPartialProjectById(projectId);
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
      const projectDetails = getProjectById(projectId);
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 421",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 414",
````

- [ ] **Step 5: 安装和运行检查**

Run: `pnpm install --frozen-lockfile`
Expected: 通过，锁文件不变（只有工作区内的一条链接）。

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 414 条，等于新的上限；`utils` 10 条，等于新的上限；`propel` 16 条，不变。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add pnpm-lock.yaml 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/features/intake/page.tsx' web/apps/web/core/components/archives/archive-tabs-list.tsx web/apps/web/core/components/automation/auto-archive-automation.tsx web/apps/web/core/components/automation/select-month-modal.tsx web/apps/web/core/components/breadcrumbs/project.tsx web/apps/web/core/components/common/switcher-label.tsx web/apps/web/core/components/dropdowns/project/base.tsx web/apps/web/core/components/inbox/content/inbox-issue-header.tsx web/apps/web/core/components/issues/filters.tsx web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx web/apps/web/core/components/navigation/project-header-button.tsx web/apps/web/core/components/navigation/project-header.tsx web/apps/web/core/components/navigation/tab-navigation-root.tsx web/apps/web/core/components/navigation/use-navigation-items.ts web/apps/web/core/components/power-k/config/creation/command.ts web/apps/web/core/components/power-k/config/navigation/commands.ts web/apps/web/core/components/power-k/menus/projects.tsx web/apps/web/core/components/power-k/ui/pages/open-entity/projects-menu.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/create-project-modal.tsx web/apps/web/core/components/project/create/common-attributes.tsx web/apps/web/core/components/project/create/header.tsx web/apps/web/core/components/project/create/project-create-buttons.tsx web/apps/web/core/components/project/delete-project-modal.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/join-project-modal.tsx web/apps/web/core/components/project/leave-project-modal.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/project/settings/features-list.tsx web/apps/web/core/components/projects/create/attributes.tsx web/apps/web/core/components/projects/create/root.tsx web/apps/web/core/components/projects/create/utils.ts web/apps/web/core/components/projects/settings/intake/header.tsx web/apps/web/core/components/settings/project/content/feature-control-item.tsx web/apps/web/core/components/settings/project/sidebar/header.tsx web/apps/web/core/components/workspace/sidebar/favorites/favorite-items/common/favorite-item-icon.tsx web/apps/web/core/components/workspace/sidebar/project-navigation.tsx web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/services/project/project.service.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/project/project.store.ts web/apps/web/package.json web/packages/propel/package.json web/packages/propel/src/emoji-icon-picker/logo.tsx web/packages/types/src/project/projects.ts web/packages/types/src/search.ts web/packages/utils/package.json web/packages/utils/src/project.ts web/packages/utils/src/rich-filters/factories/configs/properties/shared.ts web/packages/utils/src/work-item/modal.ts
```
```bash
git commit -m "refactor(M3/P8b): the generated Project replaces Plane's four project types

Every consumer of TProject, IProject, TPartialProject and IPartialProject
reads the generated Project, 38 files of them by a mechanical rename,
and the create form's values are the ProjectCreate it sends. The fields
take the contract's names (workspace_id, member_ids, project_lead_id,
default_assignee_id, intake_view), the logo is the generated LogoProps,
and getPartialProjectById gives way to getProjectById. The project
store's favourite and sequence getters go with their last readers.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t2-favorite-back` | 项目卡片又读项目的收藏（`is_favorite`，M7 的） | `tsc` | 静态 |
| `t2-intake-count-back` | 收集箱的工作项又读项目的 `intake_count` | `tsc` | 静态 |
| `t2-cover-null` | 项目卡片把 `cover_image_url` 当作字符串，不按 `null` 处理（M5 之前它恒为 `null`） | `tsc` | 静态 |

---

### Task 3: 项目的 service 和 store；工作区包装层取项目列表

**Files:**
- Create: `web/apps/web/core/services/project/project-preferences.service.ts`、`web/apps/web/core/services/project/projects.service.ts`、`web/apps/web/core/store/project/fake-projects.ts`、`web/apps/web/core/store/project/project.store.test.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`、`web/apps/web/core/components/issues/issue-layouts/utils.tsx`、`web/apps/web/core/components/project/archive-restore-modal.tsx`、`web/apps/web/core/components/project/card-list.tsx`、`web/apps/web/core/components/project/delete-project-modal.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/components/project/settings/features-list.tsx`、`web/apps/web/core/components/projects/page.tsx`、`web/apps/web/core/components/settings/project/content/feature-control-item.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/core/hooks/store/fake-store-hooks.ts`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/project/index.ts`、`web/apps/web/core/store/issue/archived/issue.store.ts`、`web/apps/web/core/store/issue/project/issue.store.ts`、`web/apps/web/core/store/issue/root.store.ts`、`web/apps/web/core/store/member/project/project-member.store.ts`、`web/apps/web/core/store/project/index.ts`、`web/apps/web/core/store/project/project.store.ts`、`web/apps/web/core/store/root.store.ts`、`web/apps/web/core/store/user/permissions.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/types/src/issues/issue-identifier.ts`
- Delete: `web/apps/web/core/services/project/project-archive.service.ts`

**Interfaces:**
- Produces（spec 2.3；M3 设计 3.19、7.1、7.3；总体设计 7.7）：
  - `core/services/project/projects.service.ts`：`class ProjectsService { constructor(api: ApiClient); list(slug, archived: boolean): Promise<Project[]>; create(slug, data: ProjectCreate); get(projectId); update(projectId, data: ProjectUpdate); delete(projectId): Promise<void>; archive(projectId); unarchive(projectId) }`，每个方法一次调用生成的客户端（`unwrap`），`list` 交回 `ProjectList.data`。`core/services/project/project-preferences.service.ts`：`ProjectPreferencesService(api).update(projectId, data: ProjectPreferencesUpdate)`（侧边栏的顺序；`get` 在 Task 5）。旧的 `project-archive.service.ts` 删除。
  - `core/store/project/project.store.ts`（整个文件）：`ProjectStore(rootStore, filters: IProjectFilterStore, api)`，状态都按 id（总体设计 7.7，P14）：每个工作区的两份列表（未归档、已归档）按工作区的 id，各自的读按项目的 id，三者都是 `ReconciledByKey`。`getProjectById` 先给项目自己的读，再给列表中的；工作区已不在调用者的列表中时什么都不给。`workspaceProjectIds`、`totalProjectIds`、`filteredProjectIds`（项目页的筛选和顺序，两份列表都到了才给）、`joinedProjectIds`（调用者是成员的、未归档的，按他侧边栏的 `sort_order`）、`loader`。取数：`fetchProjects(workspace)`、`fetchArchivedProjects(workspace)`（收 `Pick<Workspace, "id" | "slug">`）、`fetchProject(projectId)`；较旧的、会话已换的给出 `undefined`（`reconciled.ts`）。修改都经 `changes = oneAtATime()`，写入 nerve 的回答，被拒绝时什么都不改：`createProject(slug, data)`（排在已取的列表最前，`prepended`）、`updateProject(projectId, data)`、`deleteProject(project)`、`archiveProject(projectId)`、`restoreProject(projectId)`（在两份列表之间移动）、`updateProjectSortOrder(project, sortOrder)`（侧边栏的位置：发到调用者在项目里的设置，store 写回答的 `sort_order`）。Plane 的 `projectMap`、`fetchStatus`、`fetchPartialProjects`、`fetchProjectDetails` 和按 slug 的签名删除。
  - `ProjectRootStore(root, api)`：先建筛选 store，交给 `ProjectStore`；`RootStore` 传这一代的客户端。
  - `useWorkspaceFetch`：列表中有这个工作区之后再取 `["PROJECTS", id, slug]`（`fetchProjects`）；工作区包装层取 `fetchPartialProjects` 的 `useSWR` 和键 `WORKSPACE_PARTIAL_PROJECTS` 删除。项目页（`components/projects/page.tsx`，整个文件）取已归档的列表 `["ARCHIVED_PROJECTS", id, slug]`。
  - 过渡（Task 10 换掉）：项目包装层仍用原来的 `useSWR`，改调 `fetchProject(projectId)`；`permissions.store.ts` 的项目一半改调 `fetchProject`（Task 4 重写）。项目成员 store 不再写项目的 `member_ids`（`projectMap` 没有了），直到 Task 6 经 `confirmProject` 写回（spec 第 3 节，过渡）。
  - 使用方：归档、恢复、删除、卡片列表、设置页、自动化页、`issue` 的两个 store（`fetchParentStats` 改调 `fetchProject`）、`issue/root.store.ts`（`projectMap` 删除）、`TIssueIdentifierProps.projectIdentifier` 可以是 `undefined`。
  - 测试的共用部分：`core/store/project/fake-projects.ts` 的 `projectOf(identifier, workspaceId, fields?)`、`loadProjects(nerve, store, workspace, projects)`、`loadArchivedProjects(…)`（经 `answered`，核对方法和路径）；`fake-store-hooks.ts` 加 `useProject()`（`fetchProjects`）。
  - `.oxlintrc.json`：`project.store.ts`、`projects/page.tsx` 加进 `no-restricted-imports` 的范围；两个 service、store、它的测试和 `fake-projects.ts` 加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest）`core/store/project/project.store.test.ts`：
- `ProjectStore, the lists and the reads`：`lists the address's projects as nerve gives them, and finds them by id and identifier`；`lists the archived projects apart, and gives the projects page's once both lists are there`；`gives a project's own read before its list's, and holds nothing when nerve refuses it`；`fails when nerve refuses a list, keeping none, and again, keeping the list it had`；`gives nothing of a workspace the caller left, the lists and reads it holds being by id (v0 design 7.7)`；`keeps the lists it had, gives nothing and does not fail, when the session changes as it fetches again`。
- `ProjectStore, the changes`：`puts a created project first in its workspace's list, and in none not fetched`；`shows nerve's answer to a change in the list and in the project's own read`；`moves a project to the archived list and back as nerve archives and unarchives it`；`forgets a deleted project: neither the lists nor its own read give it`；`moves a project in the caller's sidebar to the place nerve gives it`（回答的位置与发出的不同）；`fails, changing nothing, when nerve refuses $change`（`it.each`，Task 4 加齐到 8 行）；`sends each change once nerve has answered the one before it, refused or not: two moves in a row among them`（`inTurn`）。
- `ProjectStore, while a fetch is out`：`fetches the list while a change is out: a fetch does not wait for it`（`fetchedWhileChangeIsOut`，回答的列表与加载的不同）；`keeps the changes nerve confirmed during a refetch on the list the refetch shows, once each`（创建、修改、删除；重取已列出的创建只列一次）；`lets the newer of two fetches write: the older gives nothing when $answer comes last`（答到列表、答到失败两行）。
- `use-workspace-fetch.test.ts`：取数的测试加上项目列表（键、fetcher 的参数；地址不是调用者的工作区时不取）。

- [ ] **Step 1: service、store 和测试的共用部分**

`web/apps/web/core/hooks/store/fake-store-hooks.ts`（修改，2 处）：

````old web/apps/web/core/hooks/store/fake-store-hooks.ts
// A stand-in for the store hooks that the hooks of a page's fetches read (use-workspace.ts, use-member.ts and user's
// useUserProfile), for their tests: a test file mocks each of those modules with this one, for instance
// vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks")), says in `stores` what the
// stores hold, and reads there what they were asked to fetch. The hooks then run as plain functions, outside React,
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
// A stand-in for the store hooks that the hooks of a page's fetches read (use-workspace.ts, use-member.ts,
// use-project.ts and user's useUserProfile), for their tests: a test file mocks each of those modules with this one,
// for instance vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks")), says in
// `stores` what the stores hold, and reads there what they were asked to fetch. The hooks then run as plain functions, outside React,
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
}

export function useUserProfile() {
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
}

export function useProject() {
  return {
    fetchProjects: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the projects of ${named(workspace)}`),
  };
}

export function useUserProfile() {
````

`web/apps/web/core/services/project/index.ts`（修改，1 处）：

````old web/apps/web/core/services/project/index.ts
export * from "./project-state.service";
export * from "./project-archive.service";
````
````new web/apps/web/core/services/project/index.ts
export * from "./project-state.service";
````

`web/apps/web/core/services/project/project-archive.service.ts`（删除）：

````delete web/apps/web/core/services/project/project-archive.service.ts
````

`web/apps/web/core/services/project/project-preferences.service.ts`（新文件，23 行）：

````file web/apps/web/core/services/project/project-preferences.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ProjectPreferences, ProjectPreferencesUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The caller's own display settings in a project: its tab bar and its place in his sidebar (M3 design 3.18, 5.1). */
export class ProjectPreferencesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** Changes the settings data names; the answer is all of them as nerve now holds them. */
  async update(projectId: string, data: ProjectPreferencesUpdate): Promise<ProjectPreferences> {
    return unwrap(
      await this.api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    );
  }
}
````

`web/apps/web/core/services/project/projects.service.ts`（新文件，60 行）：

````file web/apps/web/core/services/project/projects.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, Project, ProjectCreate, ProjectUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The projects the caller sees, each as he sees it (M3 design 3.19, 5.1, 7.3). */
export class ProjectsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /**
   * The workspace's projects that the caller sees: the archived ones alone when archived, else the others; by each
   * one's place in his sidebar, those without a place last, then by name.
   */
  async list(slug: string, archived: boolean): Promise<Project[]> {
    return unwrap(
      await this.api.GET("/api/v0/workspaces/{slug}/projects", { params: { path: { slug }, query: { archived } } })
    ).data;
  }

  /** Creates a project with the caller, and the lead when one is given, as its admins. */
  async create(slug: string, data: ProjectCreate): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/workspaces/{slug}/projects", { params: { path: { slug } }, body: data })
    );
  }

  /** The project as the caller sees it: member_role and sort_order null when he is not its member. */
  async get(projectId: string): Promise<Project> {
    return unwrap(await this.api.GET("/api/v0/projects/{project_id}", { params: { path: { project_id: projectId } } }));
  }

  /** Changes the fields data names; the answer is the project as nerve now holds it. */
  async update(projectId: string, data: ProjectUpdate): Promise<Project> {
    return unwrap(
      await this.api.PATCH("/api/v0/projects/{project_id}", { params: { path: { project_id: projectId } }, body: data })
    );
  }

  async delete(projectId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/projects/{project_id}", { params: { path: { project_id: projectId } } }));
  }

  /** Archives the project; the answer is the project as archived. */
  async archive(projectId: string): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/archive", { params: { path: { project_id: projectId } } })
    );
  }

  /** Unarchives the project; the answer is the project as unarchived. */
  async unarchive(projectId: string): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/unarchive", { params: { path: { project_id: projectId } } })
    );
  }
}
````

`web/apps/web/core/store/project/fake-projects.ts`（新文件，76 行）：

````file web/apps/web/core/store/project/fake-projects.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The caller's projects, for the tests of the stores that read them, against a fake nerve.

import type { Project, Workspace } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { answered } from "@/lib/auth/fake-nerve";
import type { IProjectStore } from "@/store/project/project.store";

/**
 * A project of the workspace workspaceId names, as nerve gives it: the identifier names it; public, with the caller
 * as its member, first in his sidebar, unless fields say otherwise.
 */
export function projectOf(identifier: string, workspaceId: string, fields: Partial<Project> = {}): Project {
  return {
    id: `p-${identifier.toLowerCase()}`,
    workspace_id: workspaceId,
    name: identifier,
    description: "",
    identifier,
    network: 2,
    project_lead_id: null,
    default_assignee_id: null,
    cycle_view: true,
    module_view: true,
    issue_views_view: true,
    intake_view: false,
    guest_view_all_features: false,
    archive_in: 0,
    archived_at: null,
    logo_props: {},
    timezone: "UTC",
    cover_image_url: null,
    member_role: 15,
    sort_order: 1000,
    member_ids: [],
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** The store fetches the workspace's projects that are not archived, and nerve lists these. */
export function loadProjects(
  nerve: FakeNerve,
  store: IProjectStore,
  workspace: Pick<Workspace, "id" | "slug">,
  projects: Project[]
) {
  return answered(
    nerve,
    () => store.fetchProjects(workspace),
    ["GET", `/api/v0/workspaces/${workspace.slug}/projects`],
    { data: projects },
    "the projects"
  );
}

/** The store fetches the workspace's archived projects, and nerve lists these. */
export function loadArchivedProjects(
  nerve: FakeNerve,
  store: IProjectStore,
  workspace: Pick<Workspace, "id" | "slug">,
  projects: Project[]
) {
  return answered(
    nerve,
    () => store.fetchArchivedProjects(workspace),
    ["GET", `/api/v0/workspaces/${workspace.slug}/projects`],
    { data: projects },
    "the archived projects"
  );
}
````

`web/apps/web/core/store/project/index.ts`（修改，3 处）：

````old web/apps/web/core/store/project/index.ts
 */

````
````new web/apps/web/core/store/project/index.ts
 */

import type { ApiClient } from "@nerve/api-client";
````
````old web/apps/web/core/store/project/index.ts
export class ProjectRootStore {
````
````new web/apps/web/core/store/project/index.ts
export class ProjectRootStore implements IProjectRootStore {
````
````old web/apps/web/core/store/project/index.ts
  constructor(_root: RootStore) {
    this.project = new ProjectStore(_root);
    this.projectFilter = new ProjectFilterStore(_root);
````
````new web/apps/web/core/store/project/index.ts
  constructor(_root: RootStore, api: ApiClient) {
    this.projectFilter = new ProjectFilterStore(_root);
    this.project = new ProjectStore(_root, this.projectFilter, api);
````

`web/apps/web/core/store/project/project.store.test.ts`（新文件，372 行）：

````file web/apps/web/core/store/project/project.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectPreferences } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import { ProjectStore } from "@/store/project/project.store";
import { ProjectFilterStore } from "@/store/project/project_filter.store";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The projects the caller sees (M3 design 3.19, 7.3), against a fake nerve that answers each request when the test
// says. The address names acme, where he is an admin; he is a member of web and docs, not of the public ops.

const acme = workspaceOf("acme", { role: 20 });
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id, { sort_order: 2000 });
const ops = projectOf("OPS", acme.id, { member_role: null, sort_order: null });
const docs = projectOf("DOCS", acme.id, { sort_order: 1000 });
const old = projectOf("OLD", acme.id, { archived_at: "2026-10-02T09:00:00Z" });
const lab = projectOf("LAB", beta.id);
/** web as nerve answers its change of name */
const renamed = { ...web, name: "Web App", updated_at: "2026-10-07T09:00:00Z" };
const LIST = "/api/v0/workspaces/acme/projects";
const ids = (projects: Project[]) => projects.map((project) => project.id);
/** The caller's settings in a project as nerve answers a change of its place in his sidebar. */
const placed = (sortOrder: number): ProjectPreferences => ({
  navigation: { default_tab: "work_items", hide_in_more_menu: [] },
  sort_order: sortOrder,
});

function setUp() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const filters = new ProjectFilterStore(fakeRoot({ router }));
  const store = new ProjectStore(fakeRoot({ router, workspaceRoot }), filters, api);
  return { nerve, api, router, workspaceRoot, filters, store };
}

/** A store of a caller whose workspaces are acme and beta, at acme's address, where nerve listed web, ops and docs. */
async function loaded() {
  const tab = setUp();
  await loadWorkspaces(tab.nerve, tab.workspaceRoot, [acme, beta]);
  tab.router.setQuery({ workspaceSlug: "acme" });
  await loadProjects(tab.nerve, tab.store, acme, [web, ops, docs]);
  return tab;
}

/** Sends a change, which nerve gives reply once it is the request out; gives how the change settled. */
async function sent(nerve: FakeNerve, send: () => Promise<unknown>, request: Endpoint, reply: Response) {
  const k = nerve.calls.length;
  const change = track(send());
  await inTurn(nerve, k, request, reply);
  await until(() => change.settled, "the change");
  return change;
}

const readOf =
  (project: Project) =>
  (nerve: FakeNerve, store: ProjectStore, body: Project = project) =>
    answered(nerve, () => store.fetchProject(project.id), ["GET", `/api/v0/projects/${project.id}`], body, "the read");

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectStore, the lists and the reads", () => {
  it("lists the address's projects as nerve gives them, and finds them by id and identifier", async () => {
    const { nerve, router, workspaceRoot, store } = setUp();
    await loadWorkspaces(nerve, workspaceRoot, [acme, beta]);
    router.setQuery({ workspaceSlug: "acme" });
    expect(store.workspaceProjectIds).toBeUndefined();
    expect(store.loader).toBe("init-loader");

    const fetched = await loadProjects(nerve, store, acme, [web, ops, docs]);
    expect(nerve.calls[1]?.query).toEqual({ archived: "false" });
    expect(fetched.value).toEqual([web, ops, docs]);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
    expect(store.loader).toBe("loaded");
    // his own, by their place in his sidebar
    expect(store.joinedProjectIds).toEqual(ids([docs, web]));
    expect(store.getProjectById(ops.id)).toEqual(ops);
    expect(store.getProjectIdentifierById(web.id)).toBe("WEB");
    expect(store.getProjectByIdentifier("DOCS")).toEqual(docs);
    router.setQuery({ workspaceSlug: "acme", projectId: web.id });
    expect(store.currentProjectDetails).toEqual(web);

    // another of his workspaces: its projects are found by id, not by the address's identifiers
    await loadProjects(nerve, store, beta, [lab]);
    expect(store.getProjectById(lab.id)).toEqual(lab);
    expect(store.getProjectByIdentifier("LAB")).toBeUndefined();
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });

  it("lists the archived projects apart, and gives the projects page's once both lists are there", async () => {
    const { nerve, filters, store } = await loaded();
    expect(store.filteredProjectIds).toBeUndefined();
    const archived = await loadArchivedProjects(nerve, store, acme, [old]);
    expect(nerve.calls[2]?.query).toEqual({ archived: "true" });
    expect(archived.value).toEqual([old]);
    expect(store.totalProjectIds).toEqual(ids([web, ops, docs, old]));
    // the page shows the projects that are not archived, unless it asks for the archived ones
    expect(store.filteredProjectIds).toEqual(ids([web, ops, docs]));
    filters.updateDisplayFilters("acme", { archived_projects: true });
    expect(store.filteredProjectIds).toEqual([old.id]);
    filters.updateDisplayFilters("acme", { archived_projects: false });
    filters.updateSearchQuery("do");
    expect(store.filteredProjectIds).toEqual([docs.id]);
  });

  it("gives a project's own read before its list's, and holds nothing when nerve refuses it", async () => {
    const { nerve, store } = await loaded();
    const read = await readOf(web)(nerve, store, renamed);
    expect(read.value).toEqual(renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);

    const missing = track(store.fetchProject("p-gone"));
    await until(() => nerve.calls.length === 4, "the read");
    nerve.calls[3]?.answer(problem(404, "project.not_found"));
    await until(() => missing.settled, "the refusal");
    expect(missing.error).toBeInstanceOf(ApiError);
    expect(store.getProjectById("p-gone")).toBeUndefined();
  });

  it("fails when nerve refuses a list, keeping none, and again, keeping the list it had", async () => {
    const { nerve, store } = await loaded();
    const archived = track(store.fetchArchivedProjects(acme));
    await until(() => nerve.calls.length === 3, "the archived list");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => archived.settled, "the refusal");
    expect(archived.error).toBeInstanceOf(ApiError);
    expect(store.filteredProjectIds).toBeUndefined();
    const again = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 4, "the refetch");
    nerve.calls[3]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });

  it("gives nothing of a workspace the caller left, the lists and reads it holds being by id (v0 design 7.7)", async () => {
    const { nerve, workspaceRoot, store } = await loaded();
    await readOf(web)(nerve, store);
    await loadProjects(nerve, store, beta, [lab]);
    await sent(nerve, () => workspaceRoot.leaveWorkspace(acme), ["POST", "/api/v0/workspaces/acme/leave"], noContent());
    expect(store.getProjectById(web.id)).toBeUndefined();
    expect(store.getProjectById(ops.id)).toBeUndefined();
    expect(store.workspaceProjectIds).toBeUndefined();
    expect(store.joinedProjectIds).toEqual([]);
    expect(store.getProjectById(lab.id)).toEqual(lab);
  });

  it("keeps the lists it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjects(acme), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(2);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });
});

describe("ProjectStore, the changes", () => {
  it("puts a created project first in its workspace's list, and in none not fetched", async () => {
    const { nerve, store } = await loaded();
    const created = projectOf("API", acme.id, { member_role: 20 });
    const body = { name: "API", identifier: "API" };
    const creating = await sent(nerve, () => store.createProject("acme", body), ["POST", LIST], json(201, created));
    expect(nerve.calls[2]?.body).toEqual(body);
    expect(creating.value).toEqual(created);
    // first, as nerve lists it: it is first in his sidebar
    expect(store.workspaceProjectIds).toEqual(ids([created, web, ops, docs]));

    // beta's projects were never fetched: a list of the new project alone would show as the whole list
    const elsewhere = projectOf("NEW", beta.id);
    const request: Endpoint = ["POST", "/api/v0/workspaces/beta/projects"];
    await sent(
      nerve,
      () => store.createProject("beta", { name: "New", identifier: "NEW" }),
      request,
      json(201, elsewhere)
    );
    expect(store.getProjectById(elsewhere.id)).toBeUndefined();
  });

  it("shows nerve's answer to a change in the list and in the project's own read", async () => {
    const { nerve, store } = await loaded();
    await readOf(web)(nerve, store);
    const request: Endpoint = ["PATCH", `/api/v0/projects/${web.id}`];
    const updated = await sent(
      nerve,
      () => store.updateProject(web.id, { name: "Web App" }),
      request,
      json(200, renamed)
    );
    expect(nerve.calls[3]?.body).toEqual({ name: "Web App" });
    expect(updated.value).toEqual(renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);
    expect(store.getProjectByIdentifier("WEB")).toEqual(renamed);
  });

  it("moves a project to the archived list and back as nerve archives and unarchives it", async () => {
    const { nerve, store } = await loaded();
    await loadArchivedProjects(nerve, store, acme, [old]);
    const archived: Project = { ...web, archived_at: "2026-10-08T09:00:00Z" };
    await sent(
      nerve,
      () => store.archiveProject(web.id),
      ["POST", `/api/v0/projects/${web.id}/archive`],
      json(200, archived)
    );
    expect(store.totalProjectIds).toEqual(ids([ops, docs, old, archived]));
    expect(store.joinedProjectIds).toEqual([docs.id]);

    await sent(
      nerve,
      () => store.restoreProject(web.id),
      ["POST", `/api/v0/projects/${web.id}/unarchive`],
      json(200, web)
    );
    expect(store.totalProjectIds).toEqual(ids([ops, docs, web, old]));
  });

  it("forgets a deleted project: neither the lists nor its own read give it", async () => {
    const { nerve, store } = await loaded();
    await readOf(web)(nerve, store);
    const deleted = await sent(
      nerve,
      () => store.deleteProject(web),
      ["DELETE", `/api/v0/projects/${web.id}`],
      noContent()
    );
    expect(deleted.error).toBeUndefined();
    expect(store.getProjectById(web.id)).toBeUndefined();
    expect(store.workspaceProjectIds).toEqual(ids([ops, docs]));
    expect(store.joinedProjectIds).toEqual([docs.id]);
  });

  it("moves a project in the caller's sidebar to the place nerve gives it", async () => {
    const { nerve, store } = await loaded();
    const request: Endpoint = ["PATCH", `/api/v0/me/projects/${web.id}/preferences`];
    // nerve's answer places web first, where the place sent would have left it after docs
    const moved = await sent(nerve, () => store.updateProjectSortOrder(web, 2500), request, json(200, placed(500)));
    expect(nerve.calls[2]?.body).toEqual({ sort_order: 2500 });
    expect(moved.error).toBeUndefined();
    expect(store.joinedProjectIds).toEqual(ids([web, docs]));
    expect(store.getProjectById(web.id)?.sort_order).toBe(500);
  });

  const changes: { change: string; send: (store: ProjectStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createProject("acme", { name: "Web", identifier: "WEB" }),
      refusal: problem(409, "project.identifier_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateProject(web.id, { name: "Ops" }),
      refusal: problem(409, "project.name_taken"),
    },
    { change: "a deletion", send: (store) => store.deleteProject(web), refusal: problem(403, "forbidden") },
    { change: "an archiving", send: (store) => store.archiveProject(web.id), refusal: problem(403, "forbidden") },
    { change: "an unarchiving", send: (store) => store.restoreProject(web.id), refusal: problem(403, "forbidden") },
    {
      change: "a move in the sidebar",
      send: (store) => store.updateProjectSortOrder(web, 500),
      refusal: problem(403, "forbidden"),
    },
  ];
  it.each(changes)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 3, "the change");
    nerve.calls[2]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
    expect([web, ops, docs].map((project) => store.getProjectById(project.id))).toEqual([web, ops, docs]);
  });

  it("sends each change once nerve has answered the one before it, refused or not: two moves in a row among them", async () => {
    const { nerve, store } = await loaded();
    const sending = changes.map(({ send }) => track(send(store)));
    // the first move is refused; the second is sent after it, as each change after the one before
    await inTurn(nerve, 2, ["POST", LIST], json(201, projectOf("API", acme.id)));
    await inTurn(nerve, 3, ["PATCH", `/api/v0/projects/${web.id}`], json(200, renamed));
    await inTurn(nerve, 4, ["DELETE", `/api/v0/projects/${web.id}`], problem(403, "forbidden"));
    await inTurn(nerve, 5, ["POST", `/api/v0/projects/${web.id}/archive`], problem(403, "forbidden"));
    await inTurn(nerve, 6, ["POST", `/api/v0/projects/${web.id}/unarchive`], json(200, renamed));
    await inTurn(nerve, 7, ["PATCH", `/api/v0/me/projects/${web.id}/preferences`], json(200, placed(500)));
    await until(() => sending.every((change) => change.settled), "the last change");
    expect(sending.map((change) => change.error === undefined)).toEqual([true, true, false, false, true, true]);
    expect(store.getProjectById(web.id)).toEqual({ ...renamed, sort_order: 500 });
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it). Of two fetches, only the newer writes.
describe("ProjectStore, while a fetch is out", () => {
  it("fetches the list while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    const listed = [web, ops, docs, projectOf("API", acme.id)];
    await fetchedWhileChangeIsOut(
      nerve,
      {
        send: () => store.updateProject(web.id, { name: "Web App" }),
        request: ["PATCH", `/api/v0/projects/${web.id}`],
      },
      { send: () => store.fetchProjects(acme), request: ["GET", LIST], body: { data: listed } }
    );
    expect(store.workspaceProjectIds).toEqual(ids(listed));
  });

  it("keeps the changes nerve confirmed during a refetch on the list the refetch shows, once each", async () => {
    const { nerve, store } = await loaded();
    const api = projectOf("API", acme.id);
    const refetched = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 3, "the refetch");
    await sent(
      nerve,
      () => store.createProject("acme", { name: "API", identifier: "API" }),
      ["POST", LIST],
      json(201, api)
    );
    await sent(
      nerve,
      () => store.updateProject(web.id, { name: "Web App" }),
      ["PATCH", `/api/v0/projects/${web.id}`],
      json(200, renamed)
    );
    await sent(nerve, () => store.deleteProject(ops), ["DELETE", `/api/v0/projects/${ops.id}`], noContent());
    // the list was read after the creation, which it has where nerve lists it, and before the other two
    nerve.calls[2]?.answer(json(200, { data: [web, ops, docs, api] }));
    await until(() => refetched.settled, "the refetch");

    expect(store.workspaceProjectIds).toEqual(ids([web, docs, api]));
    expect(store.getProjectById(web.id)).toEqual(renamed);
  });

  // The newer fetch decides: an older one answering last gives nothing, whatever nerve answered it, a failure too.
  it.each([
    { answer: "a list", reply: () => json(200, { data: [web] }) },
    { answer: "a failure", reply: () => problem(503, "server_busy") },
  ])("lets the newer of two fetches write: the older gives nothing when $answer comes last", async ({ reply }) => {
    const { nerve, store } = await loaded();
    const older = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 3, "the older list");
    const newer = track(store.fetchProjects(acme));
    await until(() => nerve.calls.length === 4, "the newer list");
    nerve.calls[3]?.answer(json(200, { data: [docs, web] }));
    await until(() => newer.settled, "the newer list");
    nerve.calls[2]?.answer(reply());
    await until(() => older.settled, "the older list");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.workspaceProjectIds).toEqual(ids([docs, web]));
  });
});
````

`web/apps/web/core/store/project/project.store.ts`（整个文件，259 行）：

````whole web/apps/web/core/store/project/project.store.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { sortBy } from "lodash-es";
import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Project, ProjectCreate, ProjectUpdate, Workspace } from "@nerve/api-client";
import type { TLoader } from "@nerve/types";
import { orderProjects, shouldFilterProject } from "@nerve/utils";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, dropped, prepended, replaced, upserted } from "@/lib/reconciled";
// services
import { ProjectPreferencesService } from "@/services/project/project-preferences.service";
import { ProjectsService } from "@/services/project/projects.service";
// store
import type { RootStore } from "../root.store";
import type { IProjectFilterStore } from "./project_filter.store";

/** A workspace as the store fetches its projects: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;
/** A project as the store changes it where no answer of nerve names its workspace. */
type ProjectRef = Pick<Project, "id" | "workspace_id">;

export interface IProjectStore {
  // computed
  /** "init-loader" until the current workspace's projects are fetched, then "loaded". */
  loader: TLoader;
  /** The current workspace's projects that are not archived, in nerve's order; undefined until fetched. */
  workspaceProjectIds: string[] | undefined;
  /** The current workspace's projects, the archived ones last, of the lists fetched. */
  totalProjectIds: string[] | undefined;
  /** The projects page's projects, by its filters and its order; undefined until both lists are fetched. */
  filteredProjectIds: string[] | undefined;
  /** The current workspace's projects the caller is a member of, not archived, by their place in his sidebar. */
  joinedProjectIds: string[];
  currentProjectDetails: Project | undefined;
  // computed actions
  getProjectById: (projectId: string | undefined | null) => Project | undefined;
  getProjectIdentifierById: (projectId: string | undefined | null) => string | undefined;
  getProjectByIdentifier: (projectIdentifier: string) => Project | undefined;
  // fetch actions
  fetchProjects: (workspace: WorkspaceRef) => Promise<Project[] | undefined>;
  fetchArchivedProjects: (workspace: WorkspaceRef) => Promise<Project[] | undefined>;
  fetchProject: (projectId: string) => Promise<Project | null | undefined>;
  // changes
  createProject: (workspaceSlug: string, data: ProjectCreate) => Promise<Project>;
  updateProject: (projectId: string, data: ProjectUpdate) => Promise<Project>;
  deleteProject: (project: ProjectRef) => Promise<void>;
  archiveProject: (projectId: string) => Promise<Project>;
  restoreProject: (projectId: string) => Promise<Project>;
  updateProjectSortOrder: (project: ProjectRef, sortOrder: number) => Promise<void>;
}

/**
 * The projects the caller sees, each as he sees it (M3 design 3.19, 7.3): two lists for each workspace, by its id,
 * the archived projects and the others, and each project's own read, by its id, which the project's pages fetch.
 * What the store holds of a workspace he is no longer a member of, it no longer gives (v0 design 7.7). Its services
 * send with the session's client; changes go one at a time and the store writes nerve's answers (7.7); fetches do not
 * queue.
 */
export class ProjectStore implements IProjectStore {
  /** Each workspace's projects that are not archived, reconciled between fetches and changes (reconciled.ts). */
  private readonly unarchived = new ReconciledByKey<Project[]>();
  /** Each workspace's archived projects. */
  private readonly archived = new ReconciledByKey<Project[]>();
  /** Each project as nerve last read it alone; null once deleted. */
  private readonly details = new ReconciledByKey<Project | null>();
  // services
  private readonly service: ProjectsService;
  private readonly preferences: ProjectPreferencesService;
  /** The changes of the projects, sent one at a time. */
  private readonly changes = oneAtATime();
  // stores
  private readonly rootStore: RootStore;
  /** The projects page's filters, by which filteredProjectIds picks and orders. */
  private readonly filters: IProjectFilterStore;

  constructor(_rootStore: RootStore, filters: IProjectFilterStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      loader: computed,
      workspaceProjectIds: computed,
      totalProjectIds: computed,
      filteredProjectIds: computed,
      joinedProjectIds: computed,
      currentProjectDetails: computed,
      // actions
      fetchProjects: action,
      fetchArchivedProjects: action,
      fetchProject: action,
      createProject: action,
      updateProject: action,
      deleteProject: action,
      archiveProject: action,
      restoreProject: action,
      updateProjectSortOrder: action,
    });
    this.rootStore = _rootStore;
    this.filters = filters;
    this.service = new ProjectsService(api);
    this.preferences = new ProjectPreferencesService(api);
  }

  /** The current workspace's two lists, each undefined until fetched. */
  private get currentLists() {
    const workspace = this.rootStore.workspaceRoot.currentWorkspace;
    return { unarchived: this.unarchived.get(workspace?.id), archived: this.archived.get(workspace?.id) };
  }

  get loader(): TLoader {
    return this.workspaceProjectIds === undefined ? "init-loader" : "loaded";
  }

  get workspaceProjectIds() {
    return this.currentLists.unarchived?.map((project) => project.id);
  }

  get totalProjectIds() {
    if (!this.rootStore.workspaceRoot.currentWorkspace) return undefined;
    const { unarchived, archived } = this.currentLists;
    return [...(unarchived ?? []), ...(archived ?? [])].map((project) => project.id);
  }

  get filteredProjectIds() {
    const {
      currentWorkspaceDisplayFilters: displayFilters,
      currentWorkspaceFilters: filters,
      searchQuery,
    } = this.filters;
    const { unarchived, archived } = this.currentLists;
    if (!displayFilters || !filters || !unarchived || !archived) return undefined;
    const query = searchQuery.toLowerCase();
    const found = [...unarchived, ...archived].filter(
      (project) =>
        (project.name.toLowerCase().includes(query) || project.identifier.toLowerCase().includes(query)) &&
        shouldFilterProject(project, displayFilters, filters)
    );
    return orderProjects(found, displayFilters.order_by).map((project) => project.id);
  }

  get joinedProjectIds() {
    const joined = (this.currentLists.unarchived ?? []).filter((project) => project.member_role !== null);
    return sortBy(joined, "sort_order").map((project) => project.id);
  }

  get currentProjectDetails() {
    return this.getProjectById(this.rootStore.router.projectId);
  }

  /**
   * The project as the store last had it from nerve, its own read first, else from its workspace's lists; nothing
   * once deleted, or when its workspace is no longer among the caller's.
   */
  getProjectById = computedFn((projectId: string | undefined | null): Project | undefined => {
    if (!projectId) return undefined;
    const workspaces = this.rootStore.workspaceRoot.workspaces ?? [];
    const read = this.details.get(projectId);
    if (read !== undefined) {
      return read && workspaces.some((workspace) => workspace.id === read.workspace_id) ? read : undefined;
    }
    for (const { id } of workspaces) {
      const listed = [...(this.unarchived.get(id) ?? []), ...(this.archived.get(id) ?? [])];
      const project = listed.find((held) => held.id === projectId);
      if (project) return project;
    }
    return undefined;
  });

  getProjectIdentifierById = computedFn(
    (projectId: string | undefined | null): string | undefined => this.getProjectById(projectId)?.identifier
  );

  /** The current workspace's project that identifier names. */
  getProjectByIdentifier = computedFn((projectIdentifier: string): Project | undefined => {
    const { unarchived, archived } = this.currentLists;
    return [...(unarchived ?? []), ...(archived ?? [])].find((project) => project.identifier === projectIdentifier);
  });

  /**
   * @description fetches the workspace's projects that are not archived, and shows them with the changes nerve
   * confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a change of session cut
   * (Reconciled.fetch)
   */
  fetchProjects = (workspace: WorkspaceRef): Promise<Project[] | undefined> =>
    this.unarchived.fetch(workspace.id, () => this.service.list(workspace.slug, false));

  /** @description fetches the workspace's archived projects, as fetchProjects */
  fetchArchivedProjects = (workspace: WorkspaceRef): Promise<Project[] | undefined> =>
    this.archived.fetch(workspace.id, () => this.service.list(workspace.slug, true));

  /** @description reads the project alone, as the caller sees it (a page of the project), as fetchProjects */
  fetchProject = (projectId: string): Promise<Project | null | undefined> =>
    this.details.fetch(projectId, () => this.service.get(projectId));

  /**
   * @description creates a project: it comes first in its workspace's list, as nerve lists it, the caller's sidebar
   * having it first. Fails, changing nothing, when nerve refuses.
   */
  createProject = (workspaceSlug: string, data: ProjectCreate): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.create(workspaceSlug, data);
      this.unarchived.confirm(project.workspace_id, prepended([project]));
      return project;
    });

  /** @description changes a project; the store then shows nerve's answer. Fails, changing nothing, when refused. */
  updateProject = (projectId: string, data: ProjectUpdate): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.update(projectId, data);
      this.unarchived.confirm(project.workspace_id, replaced(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /** @description deletes a project, which the store then no longer gives; fails, changing nothing, when refused */
  deleteProject = (project: ProjectRef): Promise<void> =>
    this.changes(async () => {
      await this.service.delete(project.id);
      this.unarchived.confirm(project.workspace_id, dropped(project.id));
      this.archived.confirm(project.workspace_id, dropped(project.id));
      this.details.confirm(project.id, () => null);
    });

  /** @description archives a project: it moves to the end of its workspace's archived list until the next fetch */
  archiveProject = (projectId: string): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.archive(projectId);
      this.unarchived.confirm(project.workspace_id, dropped(project.id));
      this.archived.confirm(project.workspace_id, upserted(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /** @description unarchives a project: it moves to the end of its workspace's list until the next fetch */
  restoreProject = (projectId: string): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.unarchive(projectId);
      this.archived.confirm(project.workspace_id, dropped(project.id));
      this.unarchived.confirm(project.workspace_id, upserted(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /** @description moves a project in the caller's sidebar; the store then shows the place nerve gives it */
  updateProjectSortOrder = (project: ProjectRef, sortOrder: number): Promise<void> =>
    this.changes(async () => {
      const { sort_order } = await this.preferences.update(project.id, { sort_order: sortOrder });
      const placed: Change<Project[]> = (list) =>
        list.map((held) => (held.id === project.id ? { ...held, sort_order } : held));
      this.unarchived.confirm(project.workspace_id, placed);
      this.details.confirm(project.id, (held) => held && { ...held, sort_order });
    });
}
````

`web/apps/web/core/store/root.store.ts`（修改，1 处）：

````old web/apps/web/core/store/root.store.ts
    this.projectRoot = new ProjectRootStore(this);
````
````new web/apps/web/core/store/root.store.ts
    this.projectRoot = new ProjectRootStore(this, api);
````

- [ ] **Step 2: 挂载时的取数**

`web/apps/web/core/components/projects/page.tsx`（整个文件，25 行）：

````whole web/apps/web/core/components/projects/page.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { ProjectRoot } from "@/components/project/root";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

export const ProjectPageRoot = observer(function ProjectPageRoot() {
  // store
  const { currentWorkspace } = useWorkspace();
  const { fetchArchivedProjects } = useProject();
  useSessionSWR(currentWorkspace && ["ARCHIVED_PROJECTS", currentWorkspace.id, currentWorkspace.slug], (id, slug) =>
    fetchArchivedProjects({ id, slug })
  );

  return <ProjectRoot />;
});
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，2 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { fetchProjectDetails } = useProject();
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { fetchProject } = useProject();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    () => fetchProjectDetails(workspaceSlug, projectId)
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    () => fetchProject(projectId)
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`（修改，5 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members and his settings", async () => {
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members, his settings and its projects", async () => {
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      ["WORKSPACE_PREFERENCES", "id-acme", "acme"],
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      ["WORKSPACE_PREFERENCES", "id-acme", "acme"],
      ["PROJECTS", "id-acme", "acme"],
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      "the settings in acme (id-acme)",
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      "the settings in acme (id-acme)",
      "the projects of acme (id-acme)",
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null]);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null, null]);
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`（修改，4 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
// hooks
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
// hooks
import { useProject } from "@/hooks/store/use-project";
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
 * whether the address's workspace is the caller's: his workspaces, which decide it; once his list has it, its members
 * and his navigation settings in it. Gives what the wrapper shows.
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
 * whether the address's workspace is the caller's: his workspaces, which decide it; once his list has it, its members,
 * his navigation settings in it and its projects that are not archived. Gives what the wrapper shows.
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  } = useWorkspace();
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  } = useWorkspace();
  const { fetchProjects } = useProject();
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  );
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  );
  useSessionSWR(workspace && ["PROJECTS", workspace.id, workspace.slug], (id, slug) => fetchProjects({ id, slug }));
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，3 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { WORKSPACE_PARTIAL_PROJECTS, WORKSPACE_PROJECTS_ROLES_INFORMATION, WORKSPACE_STATES } from "@nerve/constants";
// hooks
import { useProject } from "@/hooks/store/use-project";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { WORKSPACE_PROJECTS_ROLES_INFORMATION, WORKSPACE_STATES } from "@nerve/constants";
// hooks
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { signOut, data: currentUser } = useUser();
  const { fetchPartialProjects } = useProject();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { signOut, data: currentUser } = useUser();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx

  // fetching workspace projects
  useSWR(
    workspace ? WORKSPACE_PARTIAL_PROJECTS(workspace.slug) : null,
    workspace ? () => fetchPartialProjects(workspace.slug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx

````

- [ ] **Step 3: 使用方、键和静态检查**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/api-token/token-list.tsx",
        "web/apps/web/core/hooks/use-workspace-members-fetch.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/api-token/token-list.tsx",
        "web/apps/web/core/hooks/use-workspace-members-fetch.ts",
        "web/apps/web/core/store/project/project.store.ts",
        "web/apps/web/core/components/projects/page.tsx"
````
````old .oxlintrc.json
        "web/apps/web/core/components/profile/use-profile-member.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/profile/use-profile-member.test.ts",
        "web/apps/web/core/services/project/projects.service.ts",
        "web/apps/web/core/services/project/project-preferences.service.ts",
        "web/apps/web/core/store/project/project.store.ts",
        "web/apps/web/core/store/project/project.store.test.ts",
        "web/apps/web/core/store/project/fake-projects.ts",
        "web/apps/web/core/components/projects/page.tsx"
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`（修改，3 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const { getProjectById, joinedProjectIds: joinedProjects, updateProjectView } = useProject();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const { getProjectById, joinedProjectIds: joinedProjects, updateProjectSortOrder } = useProject();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    if (joinedProjectsList.length <= 0) return;

````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    if (joinedProjectsList.length <= 0) return;

    const source = getProjectById(sourceId);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    if (updatedSortOrder != undefined)
      updateProjectView(workspaceSlug, sourceId, { sort_order: updatedSortOrder }).catch(() => {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
    if (source && updatedSortOrder != undefined)
      updateProjectSortOrder(source, updatedSortOrder).catch(() => {
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  const { workspaceSlug, projectId } = params;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
  const { projectId } = params;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
      await updateProject(workspaceSlug, projectId, formData);
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx
      await updateProject(projectId, formData);
````

`web/apps/web/core/components/issues/issue-layouts/utils.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/issue-layouts/utils.tsx
  const { joinedProjectIds: projectIds, projectMap } = rootStore.projectRoot.project;
````
````new web/apps/web/core/components/issues/issue-layouts/utils.tsx
  const { joinedProjectIds: projectIds, getProjectById } = rootStore.projectRoot.project;
````
````old web/apps/web/core/components/issues/issue-layouts/utils.tsx
      const project = projectMap[projectId];
````
````new web/apps/web/core/components/issues/issue-layouts/utils.tsx
      const project = getProjectById(projectId);
````

`web/apps/web/core/components/project/archive-restore-modal.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/archive-restore-modal.tsx
    await archiveProject(workspaceSlug, projectId)
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
    await archiveProject(projectId)
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
    await restoreProject(workspaceSlug, projectId)
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
    await restoreProject(projectId)
````
````old web/apps/web/core/components/project/archive-restore-modal.tsx
            size="lg"
            tabIndex={1}
````
````new web/apps/web/core/components/project/archive-restore-modal.tsx
            size="lg"
````

`web/apps/web/core/components/project/card-list.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/card-list.tsx
type TProjectCardListProps = {
  totalProjectIds?: string[];
  filteredProjectIds?: string[];
};

export const ProjectCardList = observer(function ProjectCardList(props: TProjectCardListProps) {
  const { totalProjectIds: totalProjectIdsProps, filteredProjectIds: filteredProjectIdsProps } = props;
````
````new web/apps/web/core/components/project/card-list.tsx
export const ProjectCardList = observer(function ProjectCardList() {
````
````old web/apps/web/core/components/project/card-list.tsx
  const {
    loader,
    fetchStatus,
    workspaceProjectIds: storeWorkspaceProjectIds,
    filteredProjectIds: storeFilteredProjectIds,
    getProjectById,
  } = useProject();
````
````new web/apps/web/core/components/project/card-list.tsx
  const { workspaceProjectIds, filteredProjectIds, getProjectById } = useProject();
````
````old web/apps/web/core/components/project/card-list.tsx
  const { allowPermissions } = useUserPermissions();

  // derived values
  const workspaceProjectIds = totalProjectIdsProps ?? storeWorkspaceProjectIds;
  const filteredProjectIds = filteredProjectIdsProps ?? storeFilteredProjectIds;
````
````new web/apps/web/core/components/project/card-list.tsx
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/core/components/project/card-list.tsx
  if (!filteredProjectIds || !workspaceProjectIds || loader === "init-loader" || fetchStatus !== "complete")
    return <ProjectsLoader />;
````
````new web/apps/web/core/components/project/card-list.tsx
  if (!filteredProjectIds || !workspaceProjectIds) return <ProjectsLoader />;
````

`web/apps/web/core/components/project/delete-project-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/delete-project-modal.tsx
      await deleteProject(workspaceSlug, project.id);
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
      await deleteProject(project);
````

`web/apps/web/core/components/project/form.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/form.tsx
import type { Project } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/form.tsx
import type { Project, ProjectUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/form.tsx
  const handleUpdateChange = async (payload: Partial<Project>) => {
````
````new web/apps/web/core/components/project/form.tsx
  const handleUpdateChange = async (payload: ProjectUpdate) => {
````
````old web/apps/web/core/components/project/form.tsx
    return updateProject(workspaceSlug, project.id, payload)
````
````new web/apps/web/core/components/project/form.tsx
    return updateProject(project.id, payload)
````
````old web/apps/web/core/components/project/form.tsx
    const payload: Partial<Project> = {
````
````new web/apps/web/core/components/project/form.tsx
    const payload: ProjectUpdate = {
````

`web/apps/web/core/components/project/project-settings-member-defaults.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { currentProjectDetails, fetchProjectDetails, updateProject } = useProject();
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { currentProjectDetails, fetchProject, updateProject } = useProject();
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
    workspaceSlug && projectId ? () => fetchProjectDetails(workspaceSlug, projectId) : null
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
    workspaceSlug && projectId ? () => fetchProject(projectId) : null
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
    try {
      await updateProject(workspaceSlug, projectId, {
        default_assignee_id:
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
    try {
      await updateProject(projectId, {
        default_assignee_id:
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
    try {
      await updateProject(workspaceSlug, projectId, {
        guest_view_all_features: value,
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
    try {
      await updateProject(projectId, {
        guest_view_all_features: value,
````

`web/apps/web/core/components/project/settings/features-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/settings/features-list.tsx
    const updateProjectPromise = updateProject(workspaceSlug, projectId, settingsPayload);
````
````new web/apps/web/core/components/project/settings/features-list.tsx
    const updateProjectPromise = updateProject(projectId, settingsPayload);
````

`web/apps/web/core/components/settings/project/content/feature-control-item.tsx`（修改，4 处）：

````old web/apps/web/core/components/settings/project/content/feature-control-item.tsx
  description?: React.ReactNode;
  disabled?: boolean;
````
````new web/apps/web/core/components/settings/project/content/feature-control-item.tsx
  description?: React.ReactNode;
````
````old web/apps/web/core/components/settings/project/content/feature-control-item.tsx
  const { description, disabled, featureProperty, projectId, title, value, workspaceSlug } = props;
````
````new web/apps/web/core/components/settings/project/content/feature-control-item.tsx
  const { description, featureProperty, projectId, title, value, workspaceSlug } = props;
````
````old web/apps/web/core/components/settings/project/content/feature-control-item.tsx
    const updateProjectPromise = updateProject(workspaceSlug, projectId, settingsPayload);
````
````new web/apps/web/core/components/settings/project/content/feature-control-item.tsx
    const updateProjectPromise = updateProject(projectId, settingsPayload);
````
````old web/apps/web/core/components/settings/project/content/feature-control-item.tsx
          onCheckedChange={handleSubmit}
          disabled={disabled}
````
````new web/apps/web/core/components/settings/project/content/feature-control-item.tsx
          onCheckedChange={handleSubmit}
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
  const { loader, getProjectById, joinedProjectIds: joinedProjects, updateProjectView } = useProject();
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
  const { loader, getProjectById, joinedProjectIds: joinedProjects, updateProjectSortOrder } = useProject();
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    if (joinedProjectsList.length <= 0) return;

````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    if (joinedProjectsList.length <= 0) return;

    const source = getProjectById(sourceId);
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    if (updatedSortOrder != undefined)
      updateProjectView(workspaceSlug, sourceId, { sort_order: updatedSortOrder }).catch(() => {
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
    if (source && updatedSortOrder != undefined)
      updateProjectSortOrder(source, updatedSortOrder).catch(() => {
````

`web/apps/web/core/store/issue/archived/issue.store.ts`（修改，1 处）：

````old web/apps/web/core/store/issue/archived/issue.store.ts
  fetchParentStats = async (workspaceSlug: string, projectId?: string) => {
    projectId && this.rootIssueStore.rootStore.projectRoot.project.fetchProjectDetails(workspaceSlug, projectId);
````
````new web/apps/web/core/store/issue/archived/issue.store.ts
  fetchParentStats = async (_workspaceSlug: string, projectId?: string) => {
    if (projectId) await this.rootIssueStore.rootStore.projectRoot.project.fetchProject(projectId);
````

`web/apps/web/core/store/issue/project/issue.store.ts`（修改，1 处）：

````old web/apps/web/core/store/issue/project/issue.store.ts
  fetchParentStats = async (workspaceSlug: string, projectId?: string) => {
    projectId && this.rootIssueStore.rootStore.projectRoot.project.fetchProjectDetails(workspaceSlug, projectId);
````
````new web/apps/web/core/store/issue/project/issue.store.ts
  fetchParentStats = async (_workspaceSlug: string, projectId?: string) => {
    if (projectId) await this.rootIssueStore.rootStore.projectRoot.project.fetchProject(projectId);
````

`web/apps/web/core/store/issue/root.store.ts`（修改，5 处）：

````old web/apps/web/core/store/issue/root.store.ts
import type { ApiClient, MemberUser, Project } from "@nerve/api-client";
````
````new web/apps/web/core/store/issue/root.store.ts
import type { ApiClient, MemberUser } from "@nerve/api-client";
````
````old web/apps/web/core/store/issue/root.store.ts
  memberMap: Record<string, MemberUser> | undefined;
  projectMap: Record<string, Project> | undefined;
````
````new web/apps/web/core/store/issue/root.store.ts
  memberMap: Record<string, MemberUser> | undefined;
````
````old web/apps/web/core/store/issue/root.store.ts
  memberMap: Record<string, MemberUser> | undefined = undefined;
  projectMap: Record<string, Project> | undefined = undefined;
````
````new web/apps/web/core/store/issue/root.store.ts
  memberMap: Record<string, MemberUser> | undefined = undefined;
````
````old web/apps/web/core/store/issue/root.store.ts
      memberMap: observable,
      projectMap: observable,
````
````new web/apps/web/core/store/issue/root.store.ts
      memberMap: observable,
````
````old web/apps/web/core/store/issue/root.store.ts
      if (!isEmpty(rootStore?.memberRoot?.memberMap)) this.memberMap = rootStore?.memberRoot?.memberMap || undefined;
      if (!isEmpty(rootStore?.projectRoot?.project?.projectMap))
        this.projectMap = rootStore?.projectRoot?.project?.projectMap;
````
````new web/apps/web/core/store/issue/root.store.ts
      if (!isEmpty(rootStore?.memberRoot?.memberMap)) this.memberMap = rootStore?.memberRoot?.memberMap || undefined;
````

`web/apps/web/core/store/member/project/project-member.store.ts`（修改，3 处）：

````old web/apps/web/core/store/member/project/project-member.store.ts
import { uniq, unset, set, update, sortBy } from "lodash-es";
````
````new web/apps/web/core/store/member/project/project-member.store.ts
import { unset, set, sortBy } from "lodash-es";
````
````old web/apps/web/core/store/member/project/project-member.store.ts
      });
      update(this.projectRoot.projectMap, [projectId, "member_ids"], (memberIds) =>
        uniq([...memberIds, ...data.members.map((m) => m.member_id)])
      );
      this.projectRoot.projectMap[projectId].member_ids = this.projectRoot.projectMap[projectId].member_ids.concat(
        data.members.map((m) => m.member_id)
      );
````
````new web/apps/web/core/store/member/project/project-member.store.ts
      });
````
````old web/apps/web/core/store/member/project/project-member.store.ts
    unset(this.projectMemberMap, [projectId, userId]);
    set(
      this.projectRoot.projectMap,
      [projectId, "member_ids"],
      this.projectRoot.projectMap?.[projectId]?.member_ids.filter((memberId) => memberId !== userId)
    );
````
````new web/apps/web/core/store/member/project/project-member.store.ts
    unset(this.projectMemberMap, [projectId, userId]);
````

`web/apps/web/core/store/user/permissions.store.ts`（修改，2 处）：

````old web/apps/web/core/store/user/permissions.store.ts
    void this.store.projectRoot.project.fetchProjectDetails(workspaceSlug, projectId);
````
````new web/apps/web/core/store/user/permissions.store.ts
    void this.store.projectRoot.project.fetchProject(projectId);
````
````old web/apps/web/core/store/user/permissions.store.ts
        unset(this.projectUserInfo, [workspaceSlug, projectId]);
        unset(this.store.projectRoot.project.projectMap, [projectId]);
````
````new web/apps/web/core/store/user/permissions.store.ts
        unset(this.projectUserInfo, [workspaceSlug, projectId]);
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 414",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 411",
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_PARTIAL_PROJECTS = (workspaceSlug: string) =>
  `WORKSPACE_PARTIAL_PROJECTS_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/types/src/issues/issue-identifier.ts`（修改，1 处）：

````old web/packages/types/src/issues/issue-identifier.ts
  projectIdentifier: string;
````
````new web/packages/types/src/issues/issue-identifier.ts
  projectIdentifier: string | undefined;
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 411 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/automations/page.tsx' web/apps/web/core/components/issues/issue-layouts/utils.tsx web/apps/web/core/components/project/archive-restore-modal.tsx web/apps/web/core/components/project/card-list.tsx web/apps/web/core/components/project/delete-project-modal.tsx web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/components/project/settings/features-list.tsx web/apps/web/core/components/projects/page.tsx web/apps/web/core/components/settings/project/content/feature-control-item.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/hooks/store/fake-store-hooks.ts web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/project/index.ts web/apps/web/core/services/project/project-archive.service.ts web/apps/web/core/services/project/project-preferences.service.ts web/apps/web/core/services/project/projects.service.ts web/apps/web/core/store/issue/archived/issue.store.ts web/apps/web/core/store/issue/project/issue.store.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/project/fake-projects.ts web/apps/web/core/store/project/index.ts web/apps/web/core/store/project/project.store.test.ts web/apps/web/core/store/project/project.store.ts web/apps/web/core/store/root.store.ts web/apps/web/core/store/user/permissions.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/types/src/issues/issue-identifier.ts
```
```bash
git commit -m "feat(M3/P8b): the projects come from /api/v0, by a store of the session

ProjectsService wraps the generated client, and ProjectStore, built per
session with that session's client, keeps each workspace's two project
lists and each project's own read by id, reconciled with the changes
nerve confirms; a workspace the caller left gives nothing. Changes go one
at a time and take nerve's answers, the sidebar's order among them. The
workspace wrapper fetches the projects once the caller's list has the
workspace, and the projects page the archived ones, both through
useSessionSWR.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t3-svc-archived-query` | service 不论哪份列表都只问未归档的项目 | `project.store.test.ts` | vitest |
| `t3-ps-list-archived` | store 取未归档的列表时问的是已归档的 | `project.store.test.ts` | vitest |
| `t3-ps-details-unread` | `getProjectById` 不先给项目自己的读 | `project.store.test.ts` | vitest |
| `t3-ps-left-workspace` | `getProjectById` 不看调用者的工作区就给出项目自己的读（P14） | `project.store.test.ts` | vitest |
| `t3-ps-filtered-early` | 已归档的列表还没到，项目页的列表就给出了 | `project.store.test.ts` | vitest |
| `t3-ps-create-last` | 创建的项目排在列表最后，不是最前 | `project.store.test.ts` | vitest |
| `t3-ps-update-list` | 修改不写进工作区的列表 | `project.store.test.ts` | vitest |
| `t3-ps-update-read` | 修改不写进项目自己的读 | `project.store.test.ts` | vitest |
| `t3-ps-archive-stays` | 归档的项目仍留在未归档的列表中 | `project.store.test.ts` | vitest |
| `t3-ps-restore-stays` | 恢复的项目仍留在已归档的列表中 | `project.store.test.ts` | vitest |
| `t3-ps-forget-read` | 删除或离开的项目，它自己的读仍在 | `project.store.test.ts` | vitest |
| `t3-ps-no-queue` | 项目的修改立即发出，不一个接一个 | `project.store.test.ts` | vitest |
| `t3-wf-no-projects` | 工作区包装层不取工作区的项目 | `use-workspace-fetch.test.ts` | vitest |
| `t3-wf-projects-unlisted` | 调用者的列表还没有这个工作区，包装层就按地址取它的项目 | `use-workspace-fetch.test.ts` | vitest |
| `t3-ps-joined-order` | 侧边栏的项目不按它们的位置排 | `project.store.test.ts` | vitest |
| `t3-ps-change-swallows` | 项目的修改被拒绝时给出空值，不失败 | `project.store.test.ts` | vitest |
| `t3-ps-fetch-swallows` | 项目列表被拒绝时给出 `undefined`，不失败 | `project.store.test.ts` | vitest |
| `t3-ps-fetch-queued` | 项目列表的取数排进修改的队列，等在外的修改 | `project.store.test.ts` | vitest |
| `t3-rc-overtaken-writes` | 被较新的取数超过的取数也写入它的回答（P8b 的 store 都依赖的 `reconciled.ts`） | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t3-rc-overtaken-fails` | 被超过的取数遇到拒绝时照样失败 | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t3-rc-session-fails` | 会话已换的取数失败 | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t3-rc-no-replay` | 取数在外时 nerve 确认的修改不重放到它的回答上 | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t3-rc-prepended-twice` | 重取的列表已有的创建列出两次（`prepended`） | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t3-ps-archived-swallows` | 已归档的列表被拒绝时给出 `undefined`，不失败 | `project.store.test.ts` | vitest |
| `t3-ps-identifier-name` | 按名称而不是标识找项目 | `project.store.test.ts` | vitest |
| `t3-nonnull-project` | 项目 store 断言项目自己的读一定在（`!`） | oxlint（`check:lint`） | 静态 |

---

### Task 4: 加入、离开项目；权限 store 的项目一半取自 `member_role`

**Files:**
- Modify: `.oxlintrc.json`、`tools/keywords.json`、`web/apps/web/core/components/project/join-project-modal.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx`、`web/apps/web/core/components/project/member-list-item.tsx`、`web/apps/web/core/components/workspace-notifications/root.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/services/project/project-member.service.ts`、`web/apps/web/core/services/project/projects.service.ts`、`web/apps/web/core/services/user.service.ts`、`web/apps/web/core/services/workspace.service.ts`、`web/apps/web/core/store/member/project/project-member.store.ts`、`web/apps/web/core/store/project/project.store.test.ts`、`web/apps/web/core/store/project/project.store.ts`、`web/apps/web/core/store/user/index.ts`、`web/apps/web/core/store/user/permissions.store.test.ts`、`web/apps/web/core/store/user/permissions.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/types/src/users.ts`

**Interfaces:**
- Produces（spec 2.4；M3 设计 3.4、7.2、7.3；M2 交接第 11 节；P8a review 第 6 节 P14、P18）：
  - `ProjectsService`：`join(projectId): Promise<Project>`（`POST /api/v0/projects/{id}/join`）、`leave(projectId): Promise<void>`。
  - `ProjectStore`：`joinProject(projectId)`（回答是调用者此刻看到的项目，带他的 `member_role`：写进它所在的列表（在原位，否则在最后）和它自己的读）、`leaveProject(project: ProjectRef)`（之后 store 不再给出它：两份列表和自己的读都去掉；公开项目 nerve 仍会列出，下一次取数时显示为不是成员）。两者与其余修改同一个队列，被拒绝时什么都不改（`project.sole_admin` 等）。
  - `core/store/user/permissions.store.ts`（整个文件）：`UserPermissionStore(rootStore)` 只读两个 store：工作区角色是列表给出的 `Workspace.role`（P8a），项目角色是项目 store 给出的 `Project.member_role`：`getProjectRoleByWorkspaceSlugAndProjectId(slug, projectId)` 在项目属于这个工作区、调用者是成员（`member_role` 不为 `null`）时给出角色，工作区管理员给出管理员（3.4；PM+WA 是管理员，WA- 什么都没有，P18）；项目、工作区不在 store 中时什么都不给，所以离开或删除工作区之后它的项目没有角色，同一个 slug 重建的工作区也没有旧项目的角色（P14：状态都在按 id 的 store 中，权限 store 自己什么都不存）。`projectUserInfo`、`workspaceProjectsPermissions`、`fetchUserProjectInfo`、`fetchUserProjectPermissions`、`getProjectRolesByWorkspaceSlug`、`joinProject`、`leaveProject` 和它的 `WorkspaceService` 字段删除（P8a spec 第 3 节第 4 条在这里关闭）。
  - `UserStore.projectsWithCreatePermissions` 从 `joinedProjectIds` 和项目角色算出。
  - 删除：`UserService.joinProject`、`leaveProject`；`WorkspaceService.getWorkspaceUserProjectsRole`（`/project-roles/`）；`ProjectMemberService.projectMemberMe` 和模块级的 `projectMemberService`（M2 交接第 3 节的一处）；`IUserProjectsRole`；键 `WORKSPACE_PROJECTS_ROLES_INFORMATION`、`PROJECT_ME_INFORMATION`；工作区包装层取项目角色、项目包装层取 `project-members/me` 的两个 `useSWR`。
  - 使用方：加入、离开的两个弹窗和成员行改调项目 store；通知页的收集箱项改为读项目（`fetchProject`，M7 的页面，仍是原来的 `useSWR`，spec 第 5 节）。
  - 关键词：`project-invitations` 的例外删除（M1-P3 的项目成员一条在这里关闭），规则的 `why` 照改。`.oxlintrc.json`：权限 store 的测试加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest）
- `core/store/project/project.store.test.ts`：`forgets a deleted project and one the caller left: neither the lists nor its own read give it`（取代 Task 3 的删除一条）；`shows a project the caller joined as nerve now gives it, in its place and as its own read`（先读了它自己的读，加入之后两处都是回答）；拒绝的 `it.each` 加上加入、离开两行（8 行）；排队的测试加上这两个修改。
- `core/store/user/permissions.store.test.ts`：`UserPermissionStore, in a project`：`allows $who what nerve allows in it, in the project named or in the address's`（9.2 的项目一列：PA、PM、PG、PM+WA、WA-、WM-公、WM-私、WG-、P-前，9 行；每一行核对管理员、成员、任何人、只有访客四组，在给出的项目和地址的项目中各一次）；`gives no role in a project through another workspace than its own`；`gives no role in the projects of a workspace the caller left, nor of one made again under its slug`（P14）。

- [ ] **Step 1: 加入、离开**

`web/apps/web/core/services/project/projects.service.ts`（修改，1 处）：

````old web/apps/web/core/services/project/projects.service.ts
    );
  }
}
````
````new web/apps/web/core/services/project/projects.service.ts
    );
  }

  /** Makes the caller a member of the project; the answer is the project as he now sees it. */
  async join(projectId: string): Promise<Project> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/join", { params: { path: { project_id: projectId } } })
    );
  }

  /** Ends the caller's own membership of the project. */
  async leave(projectId: string): Promise<void> {
    unwrap(await this.api.POST("/api/v0/projects/{project_id}/leave", { params: { path: { project_id: projectId } } }));
  }
}
````

`web/apps/web/core/store/project/project.store.test.ts`（修改，5 处）：

````old web/apps/web/core/store/project/project.store.test.ts
  it("forgets a deleted project: neither the lists nor its own read give it", async () => {
````
````new web/apps/web/core/store/project/project.store.test.ts
  it("forgets a deleted project and one the caller left: neither the lists nor its own read give it", async () => {
````
````old web/apps/web/core/store/project/project.store.test.ts
    expect(store.workspaceProjectIds).toEqual(ids([ops, docs]));
    expect(store.joinedProjectIds).toEqual([docs.id]);
````
````new web/apps/web/core/store/project/project.store.test.ts

    const left = await sent(
      nerve,
      () => store.leaveProject(docs),
      ["POST", `/api/v0/projects/${docs.id}/leave`],
      noContent()
    );
    expect(left.error).toBeUndefined();
    expect(store.workspaceProjectIds).toEqual([ops.id]);
    expect(store.joinedProjectIds).toEqual([]);
  });

  it("shows a project the caller joined as nerve now gives it, in its place and as its own read", async () => {
    const { nerve, store } = await loaded();
    await readOf(ops)(nerve, store);
    const joined: Project = { ...ops, member_role: 20, sort_order: 3000 };
    const joining = await sent(
      nerve,
      () => store.joinProject(ops.id),
      ["POST", `/api/v0/projects/${ops.id}/join`],
      json(200, joined)
    );
    expect(joining.value).toEqual(joined);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
    expect(store.joinedProjectIds).toEqual(ids([docs, web, ops]));
    expect(store.getProjectById(ops.id)).toEqual(joined);
````
````old web/apps/web/core/store/project/project.store.test.ts
    { change: "an unarchiving", send: (store) => store.restoreProject(web.id), refusal: problem(403, "forbidden") },
````
````new web/apps/web/core/store/project/project.store.test.ts
    { change: "an unarchiving", send: (store) => store.restoreProject(web.id), refusal: problem(403, "forbidden") },
    { change: "a join", send: (store) => store.joinProject(ops.id), refusal: problem(403, "forbidden") },
    { change: "a leave", send: (store) => store.leaveProject(web), refusal: problem(409, "project.sole_admin") },
````
````old web/apps/web/core/store/project/project.store.test.ts
    await inTurn(nerve, 7, ["PATCH", `/api/v0/me/projects/${web.id}/preferences`], json(200, placed(500)));
````
````new web/apps/web/core/store/project/project.store.test.ts
    await inTurn(nerve, 7, ["POST", `/api/v0/projects/${ops.id}/join`], json(200, { ...ops, member_role: 20 }));
    await inTurn(nerve, 8, ["POST", `/api/v0/projects/${web.id}/leave`], problem(409, "project.sole_admin"));
    await inTurn(nerve, 9, ["PATCH", `/api/v0/me/projects/${web.id}/preferences`], json(200, placed(500)));
````
````old web/apps/web/core/store/project/project.store.test.ts
    expect(sending.map((change) => change.error === undefined)).toEqual([true, true, false, false, true, true]);
````
````new web/apps/web/core/store/project/project.store.test.ts
    expect(sending.map((change) => change.error === undefined)).toEqual([
      true,
      true,
      false,
      false,
      true,
      true,
      false,
      true,
    ]);
````

`web/apps/web/core/store/project/project.store.ts`（修改，7 处）：

````old web/apps/web/core/store/project/project.store.ts
  restoreProject: (projectId: string) => Promise<Project>;
````
````new web/apps/web/core/store/project/project.store.ts
  restoreProject: (projectId: string) => Promise<Project>;
  joinProject: (projectId: string) => Promise<Project>;
  leaveProject: (project: ProjectRef) => Promise<void>;
````
````old web/apps/web/core/store/project/project.store.ts
  /** Each project as nerve last read it alone; null once deleted. */
````
````new web/apps/web/core/store/project/project.store.ts
  /** Each project as nerve last read it alone; null once deleted or left. */
````
````old web/apps/web/core/store/project/project.store.ts
      restoreProject: action,
````
````new web/apps/web/core/store/project/project.store.ts
      restoreProject: action,
      joinProject: action,
      leaveProject: action,
````
````old web/apps/web/core/store/project/project.store.ts
   * once deleted, or when its workspace is no longer among the caller's.
````
````new web/apps/web/core/store/project/project.store.ts
   * once deleted or left, or when its workspace is no longer among the caller's.
````
````old web/apps/web/core/store/project/project.store.ts
      this.unarchived.confirm(project.workspace_id, dropped(project.id));
      this.archived.confirm(project.workspace_id, dropped(project.id));
      this.details.confirm(project.id, () => null);
````
````new web/apps/web/core/store/project/project.store.ts
      this.forget(project);
````
````old web/apps/web/core/store/project/project.store.ts
    });

  /** @description moves a project in the caller's sidebar; the store then shows the place nerve gives it */
````
````new web/apps/web/core/store/project/project.store.ts
    });

  /**
   * @description makes the caller a member of a project; the store then shows it as he now sees it, in its place in
   * its list, else last. Fails, changing nothing, when nerve refuses.
   */
  joinProject = (projectId: string): Promise<Project> =>
    this.changes(async () => {
      const project = await this.service.join(projectId);
      (project.archived_at ? this.archived : this.unarchived).confirm(project.workspace_id, upserted(project));
      this.details.confirm(project.id, () => project);
      return project;
    });

  /**
   * @description ends the caller's membership of a project, which the store then no longer gives (nerve may still
   * list a public one to him: the next fetch shows it). Fails, changing nothing, when nerve refuses (its only admin).
   */
  leaveProject = (project: ProjectRef): Promise<void> =>
    this.changes(async () => {
      await this.service.leave(project.id);
      this.forget(project);
    });

  /** @description moves a project in the caller's sidebar; the store then shows the place nerve gives it */
````
````old web/apps/web/core/store/project/project.store.ts
      this.details.confirm(project.id, (held) => held && { ...held, sort_order });
    });
````
````new web/apps/web/core/store/project/project.store.ts
      this.details.confirm(project.id, (held) => held && { ...held, sort_order });
    });

  /** The project leaves both lists, and its own read is gone. */
  private forget(project: ProjectRef) {
    this.unarchived.confirm(project.workspace_id, dropped(project.id));
    this.archived.confirm(project.workspace_id, dropped(project.id));
    this.details.confirm(project.id, () => null);
  }
````

- [ ] **Step 2: 权限 store 的项目一半**

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，4 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  PROJECT_DETAILS,
  PROJECT_ME_INFORMATION,
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  PROJECT_DETAILS,
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { fetchUserProjectInfo, allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { fetchProject } = useProject();
  const { joinProject } = useUserPermissions();
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  const { allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { fetchProject, joinProject } = useProject();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  );
  // fetching user project member information
  useSWR(PROJECT_ME_INFORMATION(workspaceSlug, projectId), () => fetchUserProjectInfo(workspaceSlug, projectId));
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  );
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    joinProject(workspaceSlug, projectId).finally(() => setIsJoiningProject(false));
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    joinProject(projectId).finally(() => setIsJoiningProject(false));
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，4 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { WORKSPACE_PROJECTS_ROLES_INFORMATION, WORKSPACE_STATES } from "@nerve/constants";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { WORKSPACE_STATES } from "@nerve/constants";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { useUser } from "@/hooks/store/user";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { isMobile } = usePlatformOS();
  const { fetchUserProjectPermissions } = useUserPermissions();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { isMobile } = usePlatformOS();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const access = useWorkspaceFetch(workspaceSlug);
  const workspace = access.kind === "ready" ? access.workspace : null;
  useSWR(
    workspace ? WORKSPACE_PROJECTS_ROLES_INFORMATION(workspace.slug) : null,
    workspace ? () => fetchUserProjectPermissions(workspace.slug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const access = useWorkspaceFetch(workspaceSlug);
````

`web/apps/web/core/services/project/project-member.service.ts`（修改，2 处）：

````old web/apps/web/core/services/project/project-member.service.ts
        throw error?.response?.data;
      });
  }

  async projectMemberMe(workspaceSlug: string, projectId: string): Promise<TProjectMembership> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/project-members/me/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
````
````new web/apps/web/core/services/project/project-member.service.ts
        throw error?.response?.data;
````
````old web/apps/web/core/services/project/project-member.service.ts

const projectMemberService = new ProjectMemberService();

export default projectMemberService;

````
````new web/apps/web/core/services/project/project-member.service.ts

````

`web/apps/web/core/services/user.service.ts`（修改，1 处）：

````old web/apps/web/core/services/user.service.ts
  }

  async joinProject(workspaceSlug: string, project_ids: string[]): Promise<any> {
    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveProject(workspaceSlug: string, projectId: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
````
````new web/apps/web/core/services/user.service.ts
  }
````

`web/apps/web/core/services/workspace.service.ts`（修改，2 处）：

````old web/apps/web/core/services/workspace.service.ts
  IWorkspaceSearchResults,
  IUserProjectsRole,
````
````new web/apps/web/core/services/workspace.service.ts
  IWorkspaceSearchResults,
````
````old web/apps/web/core/services/workspace.service.ts

  async getWorkspaceUserProjectsRole(workspaceSlug: string): Promise<IUserProjectsRole> {
    return this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/workspace.service.ts

````

`web/apps/web/core/store/member/project/project-member.store.ts`（修改，3 处）：

````old web/apps/web/core/store/member/project/project-member.store.ts
    // original data to revert back in case of error
    const isCurrentUser = this.rootStore.user.data?.id === userId;
    const membershipBeforeUpdate = { ...this.getProjectMembershipByUserId(userId, projectId) };
    const permissionBeforeUpdate = isCurrentUser
      ? this.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)
      : undefined;
````
````new web/apps/web/core/store/member/project/project-member.store.ts
    // original data to revert back in case of error
    const membershipBeforeUpdate = { ...this.getProjectMembershipByUserId(userId, projectId) };
````
````old web/apps/web/core/store/member/project/project-member.store.ts
        set(this.projectMemberMap, [projectId, userId, "role"], updatedProjectRole);
        if (isCurrentUser) {
          set(
            this.rootStore.user.permission.workspaceProjectsPermissions,
            [workspaceSlug, projectId],
            updatedProjectRole
          );
        }
        set(this.rootStore.user.permission.projectUserInfo, [workspaceSlug, projectId, "role"], updatedProjectRole);
````
````new web/apps/web/core/store/member/project/project-member.store.ts
        set(this.projectMemberMap, [projectId, userId, "role"], updatedProjectRole);
````
````old web/apps/web/core/store/member/project/project-member.store.ts
        set(this.projectMemberMap, [projectId, userId, "role"], membershipBeforeUpdate?.role);
        if (isCurrentUser) {
          set(
            this.rootStore.user.permission.workspaceProjectsPermissions,
            [workspaceSlug, projectId],
            membershipBeforeUpdate?.original_role
          );
          set(
            this.rootStore.user.permission.projectUserInfo,
            [workspaceSlug, projectId, "role"],
            permissionBeforeUpdate
          );
        }
````
````new web/apps/web/core/store/member/project/project-member.store.ts
        set(this.projectMemberMap, [projectId, userId, "role"], membershipBeforeUpdate?.role);
````

`web/apps/web/core/store/user/index.ts`（修改，2 处）：

````old web/apps/web/core/store/user/index.ts
    this.permission = new UserPermissionStore(store, api);
````
````new web/apps/web/core/store/user/index.ts
    this.permission = new UserPermissionStore(store);
````
````old web/apps/web/core/store/user/index.ts
    const { workspaceSlug } = this.store.router;

    const allWorkspaceProjectRoles = this.permission.getProjectRolesByWorkspaceSlug(workspaceSlug || "");

    const userPermissions =
      (allWorkspaceProjectRoles &&
        Object.keys(allWorkspaceProjectRoles)
          .filter((key) => allWorkspaceProjectRoles[key] >= EUserPermissions.MEMBER)
          .reduce(
            (res: { [projectId: string]: number }, key: string) => ((res[key] = allWorkspaceProjectRoles[key]), res),
            {}
          )) ||
      null;

    return userPermissions;
````
````new web/apps/web/core/store/user/index.ts
    const workspaceSlug = this.store.router.workspaceSlug ?? "";
    const roles: { [projectId: string]: TUserPermissions } = {};
    for (const projectId of this.store.projectRoot.project.joinedProjectIds) {
      const role = this.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
      if (role !== undefined && role >= EUserPermissions.MEMBER) roles[projectId] = role;
    }
    return roles;
````

`web/apps/web/core/store/user/permissions.store.test.ts`（修改，9 处）：

````old web/apps/web/core/store/user/permissions.store.test.ts
import type { WorkspaceRole } from "@nerve/api-client";
````
````new web/apps/web/core/store/user/permissions.store.test.ts
import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";
````
````old web/apps/web/core/store/user/permissions.store.test.ts
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
````
````new web/apps/web/core/store/user/permissions.store.test.ts
import { FakeNerve, json, noContent } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { projectOf } from "@/store/project/fake-projects";
import { ProjectStore } from "@/store/project/project.store";
import { ProjectFilterStore } from "@/store/project/project_filter.store";
````
````old web/apps/web/core/store/user/permissions.store.test.ts
// and the page's permissions follow it as nerve's do (9.2's workspace columns). On the web the caller who was never
// a member, the one removed and the one whose workspace was deleted are alike: nerve does not list the workspace.
````
````new web/apps/web/core/store/user/permissions.store.test.ts
// his role in a project the one nerve gives with the project (Project.member_role), and the page's permissions follow
// them as nerve's do (9.2's columns). On the web the caller who was never a member, the one removed and the one whose
// workspace was deleted are alike: nerve does not list the workspace.
````
````old web/apps/web/core/store/user/permissions.store.test.ts
const WORKSPACE = EUserPermissionsLevel.WORKSPACE;
````
````new web/apps/web/core/store/user/permissions.store.test.ts
const WORKSPACE = EUserPermissionsLevel.WORKSPACE;
const PROJECT = EUserPermissionsLevel.PROJECT;
````
````old web/apps/web/core/store/user/permissions.store.test.ts
];

/** The caller's workspaces as nerve lists them: one in which he has each role. */
````
````new web/apps/web/core/store/user/permissions.store.test.ts
];

/**
 * 9.2's project columns but X (the workspace's, above): the caller's role in the workspace, and the project as nerve
 * gives it to him, with his member_role (null: he sees it, no member), or not at all (undefined: nerve lists it not).
 */
const IN_PROJECTS: { who: string; role: WorkspaceRole; memberRole?: ProjectRole | null; allowed: Check[] }[] = [
  { who: "PA, its admin", role: 15, memberRole: 20, allowed: ["admin", "member", "anyone"] },
  { who: "PM, its member", role: 15, memberRole: 15, allowed: ["member", "anyone"] },
  { who: "PG, its guest", role: 5, memberRole: 5, allowed: ["anyone", "guest alone"] },
  {
    who: "PM+WA, its member and the workspace's admin",
    role: 20,
    memberRole: 15,
    allowed: ["admin", "member", "anyone"],
  },
  { who: "WA-, the workspace's admin", role: 20, memberRole: null, allowed: [] },
  { who: "WM-公, a member of the workspace, of a public project", role: 15, memberRole: null, allowed: [] },
  { who: "WM-私, a member of the workspace, of a private project", role: 15, allowed: [] },
  { who: "WG-, a guest of the workspace", role: 5, allowed: [] },
  { who: "P-前, once a member of a private project", role: 15, allowed: [] },
];

/** The caller's workspaces as nerve lists them: one in which he has each role. */
````
````old web/apps/web/core/store/user/permissions.store.test.ts
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }), nerve.client());
````
````new web/apps/web/core/store/user/permissions.store.test.ts
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }));
````
````old web/apps/web/core/store/user/permissions.store.test.ts
const slugOf = (role: WorkspaceRole | undefined) => (role ? `ws-${role}` : "ws-elsewhere");
````
````new web/apps/web/core/store/user/permissions.store.test.ts
const slugOf = (role: WorkspaceRole | undefined) => (role ? `ws-${role}` : "ws-elsewhere");

/** For each identity of IN_PROJECTS, a workspace of the caller's (ws-0 …) and its project as nerve gives it (p-p0 …). */
async function inProjects() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectFilter = new ProjectFilterStore(fakeRoot({ router }));
  const project = new ProjectStore(fakeRoot({ router, workspaceRoot }), projectFilter, api);
  const permissions = new UserPermissionStore(
    fakeRoot({ router, workspaceRoot, projectRoot: { project, projectFilter } })
  );
  const workspaces = IN_PROJECTS.map(({ role }, i) => workspaceOf(`ws-${i}`, { role }));
  await loadWorkspaces(nerve, workspaceRoot, workspaces);
  // each workspace's list, by its address: the project, or none when nerve does not list it to him
  const lists = new Map(
    IN_PROJECTS.map(({ memberRole }, i) => [
      `/api/v0/workspaces/ws-${i}/projects`,
      memberRole === undefined ? [] : [projectOf(`P${i}`, `id-ws-${i}`, { member_role: memberRole })],
    ])
  );
  const fetched = Promise.all(workspaces.map((workspace) => project.fetchProjects(workspace)));
  await until(() => nerve.calls.length === 1 + workspaces.length, "the projects");
  for (const call of nerve.calls.slice(1)) call.answer(json(200, { data: lists.get(call.path) }));
  await fetched;
  return { nerve, router, workspaceRoot, permissions };
}
````
````old web/apps/web/core/store/user/permissions.store.test.ts
    const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }), new FakeNerve().client());
````
````new web/apps/web/core/store/user/permissions.store.test.ts
    const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot }));
````
````old web/apps/web/core/store/user/permissions.store.test.ts
  });
});

````
````new web/apps/web/core/store/user/permissions.store.test.ts
  });
});

describe("UserPermissionStore, in a project", () => {
  it.each(IN_PROJECTS)(
    "allows $who what nerve allows in it, in the project named or in the address's",
    async (identity) => {
      const { router, permissions } = await inProjects();
      const i = IN_PROJECTS.indexOf(identity);
      const [slug, projectId, allowed] = [`ws-${i}`, `p-p${i}`, identity.allowed];
      for (const check of CHECKED) {
        const roles = CHECKS[check];
        const allows = allowed.includes(check);
        expect(permissions.allowPermissions(roles, PROJECT, slug, projectId), `${check}, named`).toBe(allows);
        router.setQuery({ workspaceSlug: slug, projectId });
        expect(permissions.allowPermissions(roles, PROJECT), `${check}, the address's`).toBe(allows);
        router.setQuery({});
      }
    }
  );

  it("gives no role in a project through another workspace than its own", async () => {
    const { permissions } = await inProjects();
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-0", "p-p0")).toBe(ADMIN);
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-3", "p-p0")).toBeUndefined();
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-elsewhere", "p-p0")).toBeUndefined();
  });

  it("gives no role in the projects of a workspace the caller left, nor of one made again under its slug", async () => {
    const { nerve, workspaceRoot, permissions } = await inProjects();
    const k = nerve.calls.length;
    const left = workspaceRoot.leaveWorkspace(workspaceOf("ws-0"));
    await inTurn(nerve, k, ["POST", "/api/v0/workspaces/ws-0/leave"], noContent());
    await left;
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-0", "p-p0")).toBeUndefined();

    // another tab of his deleted ws-1 and made it again: the projects of the old one are not his in the new one
    const again = workspaceOf("ws-1", { id: "id-ws-1-again", role: 20 });
    await loadWorkspaces(nerve, workspaceRoot, [again]);
    expect(permissions.getProjectRoleByWorkspaceSlugAndProjectId("ws-1", "p-p1")).toBeUndefined();
  });
});

````

`web/apps/web/core/store/user/permissions.store.ts`（整个文件，117 行）：

````whole web/apps/web/core/store/user/permissions.store.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { computedFn } from "mobx-utils";
// nerve imports
import type { WorkspaceRole } from "@nerve/api-client";
import type { TUserPermissions, TUserPermissionsLevel } from "@nerve/constants";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { EUserProjectRoles } from "@nerve/types";
import { EUserWorkspaceRoles } from "@nerve/types";
// store
import type { RootStore } from "@/store/root.store";

type ETempUserRole = TUserPermissions | EUserWorkspaceRoles | EUserProjectRoles; // TODO: Remove this once user permissions are enums in @nerve/constants

export interface IUserPermissionStore {
  getWorkspaceRoleByWorkspaceSlug: (workspaceSlug: string) => WorkspaceRole | undefined;
  getProjectRoleByWorkspaceSlugAndProjectId: (
    workspaceSlug: string,
    projectId?: string
  ) => EUserPermissions | undefined;
  allowPermissions: (
    allowPermissions: ETempUserRole[],
    level: TUserPermissionsLevel,
    workspaceSlug?: string,
    projectId?: string,
    onPermissionAllowed?: () => boolean
  ) => boolean;
}

/**
 * @description The caller's permissions, as his pages check them (M3 design 7.3): his role in a workspace is the one
 * nerve gives with the workspace (Workspace.role), his role in a project the one nerve gives with the project
 * (Project.member_role, 7.2). Both are read from the stores of his workspaces and projects, which hold them by id and
 * no longer give a workspace he left, or its projects (v0 design 7.7).
 */
export class UserPermissionStore implements IUserPermissionStore {
  constructor(protected store: RootStore) {}

  /**
   * @description Returns the caller's role in the workspace, from the caller's workspaces; undefined while they are
   * not fetched, or for a workspace the caller is not a member of
   * @param { string } workspaceSlug
   * @returns { WorkspaceRole | undefined }
   */
  getWorkspaceRoleByWorkspaceSlug = computedFn(
    (workspaceSlug: string): WorkspaceRole | undefined =>
      this.store.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.role
  );

  /**
   * @description Returns the caller's role in a project of the workspace: none until he is its member (nerve gives
   * member_role null to one who only sees it), and then the admin's for a workspace admin, as nerve decides (3.4)
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @returns { EUserPermissions | undefined }
   */
  getProjectRoleByWorkspaceSlugAndProjectId = computedFn(
    (workspaceSlug: string, projectId?: string): EUserPermissions | undefined => {
      const workspace = this.store.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
      const project = this.store.projectRoot.project.getProjectById(projectId);
      if (!workspace || project?.workspace_id !== workspace.id || project.member_role === null) return undefined;
      return workspace.role === EUserWorkspaceRoles.ADMIN ? EUserPermissions.ADMIN : project.member_role;
    }
  );

  // action helpers
  /**
   * @description Returns whether the user has the permission to perform an action
   * @param { TUserPermissions[] } allowPermissions
   * @param { TUserPermissionsLevel } level
   * @param { string } workspaceSlug
   * @param { string } projectId
   * @param { () => boolean } onPermissionAllowed
   * @returns { boolean }
   */
  allowPermissions = (
    allowPermissions: ETempUserRole[],
    level: TUserPermissionsLevel,
    workspaceSlug?: string,
    projectId?: string,
    onPermissionAllowed?: () => boolean
  ): boolean => {
    const { workspaceSlug: currentWorkspaceSlug, projectId: currentProjectId } = this.store.router;
    if (!workspaceSlug) workspaceSlug = currentWorkspaceSlug;
    if (!projectId) projectId = currentProjectId;

    let currentUserRole: TUserPermissions | undefined = undefined;

    if (level === EUserPermissionsLevel.WORKSPACE) {
      currentUserRole = workspaceSlug ? this.getWorkspaceRoleByWorkspaceSlug(workspaceSlug) : undefined;
    }

    if (level === EUserPermissionsLevel.PROJECT) {
      currentUserRole = (workspaceSlug &&
        projectId &&
        this.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)) as EUserPermissions | undefined;
    }

    if (typeof currentUserRole === "string") {
      currentUserRole = parseInt(currentUserRole);
    }

    if (currentUserRole && typeof currentUserRole === "number" && allowPermissions.includes(currentUserRole)) {
      if (onPermissionAllowed) {
        return onPermissionAllowed();
      } else {
        return true;
      }
    }

    return false;
  };
}
````

- [ ] **Step 3: 使用方、键、类型、关键词和静态检查**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/store/user/permissions.store.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/store/user/permissions.store.ts",
        "web/apps/web/core/store/user/permissions.store.test.ts",
````

`tools/keywords.json`（修改，2 处）：

````old tools/keywords.json
      "why": "项目邀请（M1 设计 2.2、3.15）：web 里没有按邮件邀请进项目的流程，\"邀请成员\"弹窗是从工作区成员中直接添加，改名为添加成员；离开项目、私有项目的文案不再说\"受邀\"。工作区邀请保留，不搜单独的 invite、invitation。精确例外：joinProject 调用的 Plane 地址 /projects/invitations/，到 M3",
````
````new tools/keywords.json
      "why": "项目邀请（M1 设计 2.2、3.15）：web 里没有按邮件邀请进项目的流程，\"邀请成员\"弹窗是从工作区成员中直接添加，改名为添加成员；离开项目、私有项目的文案不再说\"受邀\"。工作区邀请保留，不搜单独的 invite、invitation。joinProject 调用的 Plane 地址 /projects/invitations/ 曾是精确例外，M3/P8b 改调 POST /api/v0/projects/{project_id}/join 时删除",
````
````old tools/keywords.json
    {
      "rule": "project-invitations",
      "path": "web/apps/web/core/services/user.service.ts",
      "match": "projects/invitation",
      "count": 1,
      "reason": "joinProject 调用的 Plane 地址 /projects/invitations/（加入公开项目），M3 定义项目成员接口时替换",
      "until": "M3"
    },
    {
````
````new tools/keywords.json
    {
````

`web/apps/web/core/components/project/join-project-modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/join-project-modal.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/project/join-project-modal.tsx
import { useProject } from "@/hooks/store/use-project";
````
````old web/apps/web/core/components/project/join-project-modal.tsx
  const { joinProject } = useUserPermissions();
````
````new web/apps/web/core/components/project/join-project-modal.tsx
  const { joinProject } = useProject();
````
````old web/apps/web/core/components/project/join-project-modal.tsx
    await joinProject(workspaceSlug, project.id)
````
````new web/apps/web/core/components/project/join-project-modal.tsx
    await joinProject(project.id)
````
````old web/apps/web/core/components/project/join-project-modal.tsx
        <Button variant="primary" size="lg" tabIndex={1} type="submit" onClick={handleJoin} loading={isJoiningLoading}>
````
````new web/apps/web/core/components/project/join-project-modal.tsx
        <Button variant="primary" size="lg" type="submit" onClick={handleJoin} loading={isJoiningLoading}>
````

`web/apps/web/core/components/project/leave-project-modal.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/leave-project-modal.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
import { useProject } from "@/hooks/store/use-project";
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
  const { leaveProject } = useUserPermissions();
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
  const { leaveProject } = useProject();
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
          return leaveProject(workspaceSlug, project.id)
            .then(() => {
              handleClose();
            })
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
          return leaveProject(project)
            .then(() => handleClose())
````

`web/apps/web/core/components/project/member-list-item.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/member-list-item.tsx
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/project/member-list-item.tsx
import { useProject } from "@/hooks/store/use-project";
import { useUser } from "@/hooks/store/user";
````
````old web/apps/web/core/components/project/member-list-item.tsx
  const { leaveProject } = useUserPermissions();
````
````new web/apps/web/core/components/project/member-list-item.tsx
  const { getProjectById, leaveProject } = useProject();
````
````old web/apps/web/core/components/project/member-list-item.tsx
    if (memberId === currentUser?.id) {
      await leaveProject(workspaceSlug, projectId)
````
````new web/apps/web/core/components/project/member-list-item.tsx
    const project = getProjectById(projectId);
    if (memberId === currentUser?.id && project) {
      await leaveProject(project)
````

`web/apps/web/core/components/workspace-notifications/root.tsx`（修改，3 处）：

````old web/apps/web/core/components/workspace-notifications/root.tsx
import { useWorkspaceNotifications } from "@/hooks/store/notifications";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace-notifications/root.tsx
import { useWorkspaceNotifications } from "@/hooks/store/notifications";
import { useProject } from "@/hooks/store/use-project";
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````old web/apps/web/core/components/workspace-notifications/root.tsx
  const { fetchUserProjectInfo } = useUserPermissions();
````
````new web/apps/web/core/components/workspace-notifications/root.tsx
  const { fetchProject } = useProject();
````
````old web/apps/web/core/components/workspace-notifications/root.tsx
    workspace_slug && project_id && is_inbox_issue ? () => fetchUserProjectInfo(workspace_slug, project_id) : null
````
````new web/apps/web/core/components/workspace-notifications/root.tsx
    workspace_slug && project_id && is_inbox_issue ? () => fetchProject(project_id) : null
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 411",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 409",
````

`web/packages/constants/src/fetch-keys.ts`（修改，2 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_PROJECTS_ROLES_INFORMATION = (workspaceSlug: string) =>
  `WORKSPACE_PROJECTS_ROLES_INFORMATION_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````
````old web/packages/constants/src/fetch-keys.ts

export const PROJECT_ME_INFORMATION = (_workspaceSlug: string, projectId: string) =>
  `PROJECT_ME_INFORMATION_${projectId.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/types/src/users.ts`（修改，2 处）：

````old web/packages/types/src/users.ts
 */

import type { TUserPermissions } from "./enums";
````
````new web/packages/types/src/users.ts
 */
````
````old web/packages/types/src/users.ts

export interface IUserProjectsRole {
  [projectId: string]: TUserPermissions;
}

````
````new web/packages/types/src/users.ts

````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外（`project-invitations` 的例外删除），没有命中；web 的 oxlint 409 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json tools/keywords.json web/apps/web/core/components/project/join-project-modal.tsx web/apps/web/core/components/project/leave-project-modal.tsx web/apps/web/core/components/project/member-list-item.tsx web/apps/web/core/components/workspace-notifications/root.tsx web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/services/project/project-member.service.ts web/apps/web/core/services/project/projects.service.ts web/apps/web/core/services/user.service.ts web/apps/web/core/services/workspace.service.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/project/project.store.test.ts web/apps/web/core/store/project/project.store.ts web/apps/web/core/store/user/index.ts web/apps/web/core/store/user/permissions.store.test.ts web/apps/web/core/store/user/permissions.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/types/src/users.ts
```
```bash
git commit -m "feat(M3/P8b): joining and leaving a project, and the caller's role in it, from /api/v0

The project store joins and leaves through the generated client, in its
queue, and shows nerve's answer. The permission store keeps nothing of
its own: a project role is the member_role nerve gives with the project,
an admin's for a workspace admin, and none once the project or its
workspace is gone from the stores, which hold them by id. The old
project-roles and project-members/me fetches, the user service's join
and leave and the project-invitations keyword exception go.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t4-ps-join-list` | 加入的项目在列表中仍是旧的那一份，没有 `member_role` | `project.store.test.ts` | vitest |
| `t4-ps-join-read` | 加入的项目自己的读仍是旧的那一份 | `project.store.test.ts` | vitest |
| `t4-ps-leave-stays` | 离开的项目仍在 store 显示它的地方 | `project.store.test.ts` | vitest |
| `t4-pf-wa-admin` | 是项目成员的工作区管理员得到他的项目角色，不是管理员（PM+WA，P18） | `permissions.store.test.ts` | vitest |
| `t4-pf-wa-no-member` | 工作区管理员在他不是成员的项目中得到管理员的角色（WA-，P18） | `permissions.store.test.ts` | vitest |
| `t4-pf-public-guest` | 看得到公开项目的人在其中得到访客的角色（WM-公） | `permissions.store.test.ts` | vitest |
| `t4-pf-admin-member` | 项目管理员得到成员的角色（PA） | `permissions.store.test.ts` | vitest |
| `t4-pf-member-guest` | 项目成员得到访客的角色（PM） | `permissions.store.test.ts` | vitest |
| `t4-pf-guest-member` | 项目访客得到成员的角色（PG） | `permissions.store.test.ts` | vitest |
| `t4-pf-other-workspace` | 经不是项目所在的工作区也给出项目角色 | `permissions.store.test.ts` | vitest |
| `t4-pf-admin-any` | 任何工作区角色都让项目成员成为项目的管理员（P8a 的 `pf-project-admin-any`，P18） | `permissions.store.test.ts` | vitest |
| `t4-pf-unlisted-guest` | nerve 没有列出的项目也给出访客的角色（Plane 对公开项目的兜底） | `permissions.store.test.ts` | vitest |
| `t4-pf-old-workspace` | 调用者离开的工作区的项目，在同一个 slug 重建的工作区中给出角色 | `permissions.store.test.ts` | vitest |

---

### Task 5: 调用者在项目里的标签栏；标识的检查；旧 `ProjectService` 只留三个方法

**Files:**
- Create: `web/apps/web/core/hooks/store/use-project-preferences.ts`、`web/apps/web/core/store/project/preferences.store.test.ts`、`web/apps/web/core/store/project/preferences.store.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/navigation/project-header.tsx`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`、`web/apps/web/core/components/navigation/tab-navigation-root.tsx`、`web/apps/web/core/components/navigation/tab-navigation-utils.ts`、`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`、`web/apps/web/core/components/navigation/use-tab-preferences.ts`、`web/apps/web/core/components/project/form.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/services/project/project-preferences.service.ts`、`web/apps/web/core/services/project/project.service.ts`、`web/apps/web/core/services/project/projects.service.ts`、`web/apps/web/core/store/member/project/project-member.store.ts`、`web/apps/web/core/store/project/index.ts`、`web/apps/web/core/store/project/project.store.test.ts`、`web/apps/web/core/store/project/project.store.ts`、`web/apps/web/core/store/user/permissions.store.test.ts`、`web/apps/web/package.json`、`web/packages/types/src/project/projects.ts`、`web/packages/types/src/view-props.ts`

**Interfaces:**
- Produces（spec 2.5；M3 设计 3.18、7.3；P4b 的交接；Codex 设计评审 M-3）：
  - `ProjectPreferencesService.get(projectId): Promise<ProjectPreferences>`（`GET /api/v0/me/projects/{id}/preferences`，没有改过时是 nerve 的默认）。
  - `core/store/project/preferences.store.ts`：`ProjectPreferencesStore(projectOf, api)`，`ProjectRootStore.preferences`（`core/store/project/index.ts`，整个文件）；每个项目的标签栏按项目的 id（`ReconciledByKey`，一份文档：取数在外时确认的修改胜过它较旧的回答）；`getNavigation(projectId)`（项目 store 不再给出这个项目时什么都不给）、`fetchNavigation(projectId)`、`updateNavigation(projectId, navigation)`（经 `changes = oneAtATime()`，nerve 整份替换，store 写入回答）。设置的另一个字段 `sort_order` 由项目 store 持有、经 Task 3 的 `updateProjectSortOrder` 修改（`Project.sort_order` 也带着它）。`core/hooks/store/use-project-preferences.ts`：`useProjectPreferences()`。
  - `use-tab-preferences.ts`（整个文件）：`useTabPreferences(projectId)` 读写新 store，键是生成的 `ProjectTab`；导航的四个组件、侧边栏的项目行、项目页头只传项目的 id。Plane 先改再回滚的写法删除（W17：修改在 nerve 回答之后才显示）。
  - 项目包装层取标签栏改调 `fetchNavigation`（过渡：仍是原来的 `useSWR`，Task 10 换掉）；成员 store 的 `fetchProjectUserProperties`、`updateProjectUserProperties`、`projectUserPropertiesMap` 和它的两个旧 service 删除。
  - `ProjectStore.checkProjectIdentifier(slug, identifier): Promise<IdentifierAvailability>`（`ProjectsService.checkIdentifier`，`GET …/project-identifiers/{identifier}`）；项目设置页的标识检查改调它，`project/form.tsx` 的模块级 `ProjectService` 删除（M2 交接第 3 节的一处）。
  - `core/services/project/project.service.ts`（整个文件）：旧 `ProjectService` 只留 M4、M6、M7 的三个方法（7.3）：`getProjectUserProperties`、`updateProjectUserProperties`（M4 的工作项筛选）、`projectIssuesSearch`（M4、M6、M7 的工作项搜索）。
  - 删除：`IProjectMemberNavigationPreferences`、`IProjectUserPropertiesResponse`。
  - `.oxlintrc.json`：两条规则的范围中 `core/store/project/` 改为 `web/apps/web/core/store/project/**`；`use-project-preferences.ts`、`use-tab-preferences.ts` 加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest）
- `core/store/project/preferences.store.test.ts`：`ProjectPreferencesStore`：`keeps the caller's tab bar in a project as nerve gives it, and none of a project the store no longer gives`；`fails when nerve refuses it, keeping none, and again, keeping the tab bar it had`；`keeps the tab bar it had, gives nothing and does not fail, when the session changes as it fetches it again`；`sends the whole tab bar, and has nerve's answer only once nerve answers`（回答与请求不同）；`fails, changing nothing, when nerve refuses a change`；`keeps each project's tab bar apart: a change or a fetch in one leaves the other's`；`sends each change once nerve has answered the one before it, refused or not`；`fetches the tab bar while a change is out: a fetch does not wait for it`。`ProjectPreferencesStore, while a fetch is out`：`shows a change nerve confirmed during a refetch, not the refetch's older tab bar`；`lets the newer fetch write: an older one answering last writes nothing`（一份文档没有"已列出的又被创建"，文件里说明）。
- `core/store/project/project.store.test.ts`：`asks nerve whether an identifier is free, and fails when nerve refuses`。
- `permissions.store.test.ts`：改用 `ProjectRootStore` 建项目 store（它现在有子 store）。

- [ ] **Step 1: service、store 和 hook**

`web/apps/web/core/hooks/store/use-project-preferences.ts`（新文件，15 行）：

````file web/apps/web/core/hooks/store/use-project-preferences.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useContext } from "react";
// mobx store
import { StoreContext } from "@/lib/store-context";
import type { IProjectPreferencesStore } from "@/store/project/preferences.store";

export const useProjectPreferences = (): IProjectPreferencesStore => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useProjectPreferences must be used within StoreProvider");
  return context.projectRoot.preferences;
};
````

`web/apps/web/core/services/project/project-preferences.service.ts`（修改，1 处）：

````old web/apps/web/core/services/project/project-preferences.service.ts
  constructor(private readonly api: ApiClient) {}

````
````new web/apps/web/core/services/project/project-preferences.service.ts
  constructor(private readonly api: ApiClient) {}

  /** The caller's settings in the project, nerve's defaults while he has changed none. */
  async get(projectId: string): Promise<ProjectPreferences> {
    return unwrap(
      await this.api.GET("/api/v0/me/projects/{project_id}/preferences", {
        params: { path: { project_id: projectId } },
      })
    );
  }

````

`web/apps/web/core/services/project/project.service.ts`（整个文件，51 行）：

````whole web/apps/web/core/services/project/project.service.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueFiltersResponse, ISearchIssueResponse, TProjectIssuesSearchParams } from "@nerve/types";
// services
import { APIService } from "@/services/api.service";

/**
 * The old addresses that later Ms replace (M3 design 7.3): a project's work item filters, which M4's filter store
 * reads and writes, and the search of its work items, which M4, M6 and M7 call. The service goes when M4 replaces the
 * last of them.
 */
export class ProjectService extends APIService {
  // User Properties
  async getProjectUserProperties(workspaceSlug: string, projectId: string): Promise<IIssueFiltersResponse> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateProjectUserProperties(
    workspaceSlug: string,
    projectId: string,
    data: Partial<IIssueFiltersResponse>
  ): Promise<IIssueFiltersResponse> {
    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async projectIssuesSearch(
    workspaceSlug: string,
    projectId: string,
    params: TProjectIssuesSearchParams
  ): Promise<ISearchIssueResponse[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/search-issues/`, {
      params,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
````

`web/apps/web/core/services/project/projects.service.ts`（修改，2 处）：

````old web/apps/web/core/services/project/projects.service.ts
import type { ApiClient, Project, ProjectCreate, ProjectUpdate } from "@nerve/api-client";
````
````new web/apps/web/core/services/project/projects.service.ts
import type { ApiClient, IdentifierAvailability, Project, ProjectCreate, ProjectUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/services/project/projects.service.ts
      await this.api.POST("/api/v0/workspaces/{slug}/projects", { params: { path: { slug } }, body: data })
````
````new web/apps/web/core/services/project/projects.service.ts
      await this.api.POST("/api/v0/workspaces/{slug}/projects", { params: { path: { slug } }, body: data })
    );
  }

  /** Whether identifier, once upper-cased, can name a new project of the workspace. */
  async checkIdentifier(slug: string, identifier: string): Promise<IdentifierAvailability> {
    return unwrap(
      await this.api.GET("/api/v0/workspaces/{slug}/project-identifiers/{identifier}", {
        params: { path: { slug, identifier } },
      })
````

`web/apps/web/core/store/project/index.ts`（整个文件，32 行）：

````whole web/apps/web/core/store/project/index.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ApiClient } from "@nerve/api-client";
import type { RootStore } from "../root.store";
import type { IProjectPreferencesStore } from "./preferences.store";
import { ProjectPreferencesStore } from "./preferences.store";
import type { IProjectStore } from "./project.store";
import { ProjectStore } from "./project.store";
import type { IProjectFilterStore } from "./project_filter.store";
import { ProjectFilterStore } from "./project_filter.store";

export interface IProjectRootStore {
  project: IProjectStore;
  projectFilter: IProjectFilterStore;
  preferences: IProjectPreferencesStore;
}

export class ProjectRootStore implements IProjectRootStore {
  project: IProjectStore;
  projectFilter: IProjectFilterStore;
  preferences: IProjectPreferencesStore;

  constructor(_root: RootStore, api: ApiClient) {
    this.projectFilter = new ProjectFilterStore(_root);
    this.project = new ProjectStore(_root, this.projectFilter, api);
    this.preferences = new ProjectPreferencesStore(this.project.getProjectById, api);
  }
}
````

`web/apps/web/core/store/project/preferences.store.test.ts`（新文件，199 行）：

````file web/apps/web/core/store/project/preferences.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, ProjectNavigation } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { projectOf } from "@/store/project/fake-projects";
import { ProjectPreferencesStore } from "@/store/project/preferences.store";

// The caller's tab bar in each project's header (M3 design 3.18, 7.3), against a fake nerve that answers each request
// when the test says.

const web = projectOf("WEB", "id-acme");
const ops = projectOf("OPS", "id-acme");
const PREFERENCES = `/api/v0/me/projects/${web.id}/preferences`;
/** nerve's default, which it gives until the caller changes it. */
const defaults: ProjectNavigation = { default_tab: "work_items", hide_in_more_menu: [] };
const modules: ProjectNavigation = { default_tab: "modules", hide_in_more_menu: ["views"] };
/** nerve's answer to the change to modules, once another tab of his has hidden the cycles too. */
const elsewhere: ProjectNavigation = { default_tab: "modules", hide_in_more_menu: ["views", "cycles"] };
/** His tab bar in ops, which opens on its cycles. */
const cycles: ProjectNavigation = { default_tab: "cycles", hide_in_more_menu: [] };

/** nerve's answer: the settings, the tab bar with his place for the project in the sidebar. */
const settings = (navigation: ProjectNavigation) => ({ navigation, sort_order: 1000 });

/** The store of a tab and its client; the projects the project store gives, web and ops, until a test takes one. */
function setUp() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const given = new Map<string, Project>([web, ops].map((project) => [project.id, project]));
  const store = new ProjectPreferencesStore((projectId) => given.get(projectId), api);
  return { nerve, api, given, store };
}

/** The store fetches the caller's tab bar in the project (web unless it says), and nerve gives this one. */
function load(nerve: FakeNerve, store: ProjectPreferencesStore, navigation: ProjectNavigation, project = web) {
  const fetch = () => store.fetchNavigation(project.id);
  const path = `/api/v0/me/projects/${project.id}/preferences`;
  return answered(nerve, fetch, ["GET", path], settings(navigation), "the tab bar");
}

/** A store whose tab bar in web nerve gave as its default. */
async function loaded() {
  const tab = setUp();
  await load(tab.nerve, tab.store, defaults);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectPreferencesStore", () => {
  it("keeps the caller's tab bar in a project as nerve gives it, and none of a project the store no longer gives", async () => {
    const { nerve, given, store } = setUp();
    const fetched = store.fetchNavigation(web.id);
    await until(() => nerve.calls.length === 1, "the tab bar");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: PREFERENCES });
    expect(store.getNavigation(web.id)).toBeUndefined();
    nerve.calls[0]?.answer(json(200, settings(modules)));
    expect(await settle(fetched, "the tab bar")).toEqual({ settled: true, value: modules });
    expect(store.getNavigation(web.id)).toEqual(modules);
    expect(store.getNavigation(ops.id)).toBeUndefined();

    // he left web, or it was deleted, or its workspace is no longer his
    given.delete(web.id);
    expect(store.getNavigation(web.id)).toBeUndefined();
  });

  it("fails when nerve refuses it, keeping none, and again, keeping the tab bar it had", async () => {
    const { nerve, store } = setUp();
    const refused = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 1, "the tab bar");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getNavigation(web.id)).toBeUndefined();

    await load(nerve, store, defaults);
    const again = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 3, "the tab bar again");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getNavigation(web.id)).toEqual(defaults);
  });

  it("keeps the tab bar it had, gives nothing and does not fail, when the session changes as it fetches it again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchNavigation(web.id), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getNavigation(web.id)).toEqual(defaults);
  });

  it("sends the whole tab bar, and has nerve's answer only once nerve answers", async () => {
    const { nerve, store } = await loaded();
    const changed = track(store.updateNavigation(web.id, modules));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: PREFERENCES, body: { navigation: modules } });
    expect(store.getNavigation(web.id)).toEqual(defaults);
    nerve.calls[1]?.answer(json(200, settings(elsewhere)));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(elsewhere);
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });

  it("fails, changing nothing, when nerve refuses a change", async () => {
    const { nerve, store } = await loaded();
    const hidden: ProjectNavigation = { default_tab: "work_items", hide_in_more_menu: ["work_items"] };
    const refused = track(store.updateNavigation(web.id, hidden));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(problem(422, "validation_failed"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getNavigation(web.id)).toEqual(defaults);
  });

  it("keeps each project's tab bar apart: a change or a fetch in one leaves the other's", async () => {
    const { nerve, store } = await loaded();
    await load(nerve, store, cycles, ops);
    const changed = track(store.updateNavigation(web.id, modules));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(modules)));
    await until(() => changed.settled, "the answer");
    expect(store.getNavigation(web.id)).toEqual(modules);
    expect(store.getNavigation(ops.id)).toEqual(cycles);
    await load(nerve, store, defaults, ops);
    expect(store.getNavigation(ops.id)).toEqual(defaults);
    expect(store.getNavigation(web.id)).toEqual(modules);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updateNavigation(web.id, cycles));
    const second = track(store.updateNavigation(web.id, modules));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(elsewhere)));
    await until(() => second.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(nerve.calls[2]?.body).toEqual({ navigation: modules });
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });

  it("fetches the tab bar while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateNavigation(web.id, cycles), request: ["PATCH", PREFERENCES] },
      { send: () => store.fetchNavigation(web.id), request: ["GET", PREFERENCES], body: settings(elsewhere) }
    );
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: the change wins (the order is forced:
// the fetch waits until the test answers it, after the change has finished). Of two fetches of one project's tab bar,
// only the newer writes. A tab bar is one document, nothing in it listed: none can be listed twice.
describe("ProjectPreferencesStore, while a fetch is out", () => {
  it("shows a change nerve confirmed during a refetch, not the refetch's older tab bar", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    const changed = track(store.updateNavigation(web.id, modules));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(modules)));
    await until(() => changed.settled, "the answer");
    // read before the change
    nerve.calls[1]?.answer(json(200, settings(defaults)));
    await until(() => refetched.settled, "the refetch");

    expect(store.getNavigation(web.id)).toEqual(modules);
    expect(refetched.value).toEqual(modules);
  });

  it("lets the newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = setUp();
    const older = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 1, "the older tab bar");
    const newer = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 2, "the newer tab bar");
    nerve.calls[1]?.answer(json(200, settings(elsewhere)));
    await until(() => newer.settled, "the newer tab bar");
    // read before another tab of his hid the cycles
    nerve.calls[0]?.answer(json(200, settings(modules)));
    await until(() => older.settled, "the older tab bar");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });
});
````

`web/apps/web/core/store/project/preferences.store.ts`（新文件，75 行）：

````file web/apps/web/core/store/project/preferences.store.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { action, makeObservable } from "mobx";
// nerve imports
import type { ApiClient, Project, ProjectNavigation } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { ProjectPreferencesService } from "@/services/project/project-preferences.service";

export interface IProjectPreferencesStore {
  getNavigation: (projectId: string) => ProjectNavigation | undefined;
  fetchNavigation: (projectId: string) => Promise<ProjectNavigation | undefined>;
  updateNavigation: (projectId: string, navigation: ProjectNavigation) => Promise<ProjectNavigation>;
}

/**
 * The tab bar of each project's header as the caller has it, the navigation of his settings in the project (M3 design
 * 3.18, 7.3). Their other field, the project's place in his sidebar, nerve also gives with the project
 * (Project.sort_order): the project store holds and changes it. The service sends with the session's client, which
 * the RootStore of the session hands down; changes go one at a time and the store writes nerve's answers (v0 design
 * 7.7); fetches do not queue.
 */
export class ProjectPreferencesStore implements IProjectPreferencesStore {
  /**
   * The tab bar in each project, by the project's id, reconciled between its fetches and the changes nerve confirmed
   * (reconciled.ts): it is one document, so a change nerve confirmed wins over a fetch that was out.
   */
  private readonly navigation = new ReconciledByKey<ProjectNavigation>();
  private readonly service: ProjectPreferencesService;
  /** The changes of the tab bars, sent one at a time. */
  private readonly changes = oneAtATime();

  /** projectOf: the project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  constructor(
    private readonly projectOf: (projectId: string) => Project | undefined,
    api: ApiClient
  ) {
    makeObservable(this, {
      fetchNavigation: action,
      updateNavigation: action,
    });
    this.service = new ProjectPreferencesService(api);
  }

  /**
   * The caller's tab bar in the project, once fetched: nothing of a project the project store no longer gives, one
   * deleted or left, or of a workspace no longer his.
   */
  getNavigation = (projectId: string): ProjectNavigation | undefined =>
    this.projectOf(projectId) ? this.navigation.get(projectId) : undefined;

  /**
   * @description fetches the caller's tab bar in a project, nerve's default until he changes it, and shows it with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchNavigation = (projectId: string): Promise<ProjectNavigation | undefined> =>
    this.navigation.fetch(projectId, async () => (await this.service.get(projectId)).navigation);

  /**
   * @description replaces the caller's tab bar in a project (nerve replaces it whole); the store then has nerve's
   * answer. Fails, changing nothing, when nerve refuses.
   */
  updateNavigation = (projectId: string, navigation: ProjectNavigation): Promise<ProjectNavigation> =>
    this.changes(async () => {
      const preferences = await this.service.update(projectId, { navigation });
      this.navigation.confirm(projectId, () => preferences.navigation);
      return preferences.navigation;
    });
}
````

`web/apps/web/core/store/project/project.store.test.ts`（修改，1 处）：

````old web/apps/web/core/store/project/project.store.test.ts
    expect(nerve.calls).toHaveLength(2);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
````
````new web/apps/web/core/store/project/project.store.test.ts
    expect(nerve.calls).toHaveLength(2);
    expect(store.workspaceProjectIds).toEqual(ids([web, ops, docs]));
  });

  it("asks nerve whether an identifier is free, and fails when nerve refuses", async () => {
    const { nerve, store } = setUp();
    const check = () => store.checkProjectIdentifier("acme", "web");
    const request: Endpoint = ["GET", "/api/v0/workspaces/acme/project-identifiers/web"];
    const checked = await answered(nerve, check, request, { available: false }, "the check");
    expect(checked).toEqual({ settled: true, value: { available: false } });
    const refused = track(check());
    await until(() => nerve.calls.length === 2, "the second check");
    nerve.calls[1]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
````

`web/apps/web/core/store/project/project.store.ts`（修改，3 处）：

````old web/apps/web/core/store/project/project.store.ts
import type { ApiClient, Project, ProjectCreate, ProjectUpdate, Workspace } from "@nerve/api-client";
````
````new web/apps/web/core/store/project/project.store.ts
import type {
  ApiClient,
  IdentifierAvailability,
  Project,
  ProjectCreate,
  ProjectUpdate,
  Workspace,
} from "@nerve/api-client";
````
````old web/apps/web/core/store/project/project.store.ts
  fetchProject: (projectId: string) => Promise<Project | null | undefined>;
````
````new web/apps/web/core/store/project/project.store.ts
  fetchProject: (projectId: string) => Promise<Project | null | undefined>;
  checkProjectIdentifier: (workspaceSlug: string, identifier: string) => Promise<IdentifierAvailability>;
````
````old web/apps/web/core/store/project/project.store.ts
    this.details.fetch(projectId, () => this.service.get(projectId));

````
````new web/apps/web/core/store/project/project.store.ts
    this.details.fetch(projectId, () => this.service.get(projectId));

  /** @description whether identifier can name a new project of the workspace */
  checkProjectIdentifier = (workspaceSlug: string, identifier: string): Promise<IdentifierAvailability> =>
    this.service.checkIdentifier(workspaceSlug, identifier);

````

`web/apps/web/core/store/user/permissions.store.test.ts`（修改，3 处）：

````old web/apps/web/core/store/user/permissions.store.test.ts
import { fakeRoot } from "@/store/fake-root";
import { projectOf } from "@/store/project/fake-projects";
import { ProjectStore } from "@/store/project/project.store";
import { ProjectFilterStore } from "@/store/project/project_filter.store";
````
````new web/apps/web/core/store/user/permissions.store.test.ts
import { fakeRoot } from "@/store/fake-root";
import { ProjectRootStore } from "@/store/project";
import { projectOf } from "@/store/project/fake-projects";
````
````old web/apps/web/core/store/user/permissions.store.test.ts
  const projectFilter = new ProjectFilterStore(fakeRoot({ router }));
  const project = new ProjectStore(fakeRoot({ router, workspaceRoot }), projectFilter, api);
  const permissions = new UserPermissionStore(
    fakeRoot({ router, workspaceRoot, projectRoot: { project, projectFilter } })
  );
````
````new web/apps/web/core/store/user/permissions.store.test.ts
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  const permissions = new UserPermissionStore(fakeRoot({ router, workspaceRoot, projectRoot }));
````
````old web/apps/web/core/store/user/permissions.store.test.ts
  const fetched = Promise.all(workspaces.map((workspace) => project.fetchProjects(workspace)));
````
````new web/apps/web/core/store/user/permissions.store.test.ts
  const fetched = Promise.all(workspaces.map((workspace) => projectRoot.project.fetchProjects(workspace)));
````

- [ ] **Step 2: 标签栏和标识检查的使用方**

`web/apps/web/core/components/navigation/project-header.tsx`（修改，1 处）：

````old web/apps/web/core/components/navigation/project-header.tsx
  const { tabPreferences } = useTabPreferences(workspaceSlug, projectId);
````
````new web/apps/web/core/components/navigation/project-header.tsx
  const { tabPreferences } = useTabPreferences(projectId);
````

`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`（修改，2 处）：

````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
import { Link } from "react-router";
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
import { Link } from "react-router";
import type { ProjectTab } from "@nerve/api-client";
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
  onToggleDefault: (tabKey: string) => void;
  onShow: (tabKey: string) => void;
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
  onToggleDefault: (tabKey: ProjectTab) => void;
  onShow: (tabKey: ProjectTab) => void;
````

`web/apps/web/core/components/navigation/tab-navigation-root.tsx`（修改，3 处）：

````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
import { useParams, useLocation, Link } from "react-router";
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
import { useParams, useLocation, Link } from "react-router";
import type { ProjectTab } from "@nerve/api-client";
````
````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
  key: string;
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
  key: ProjectTab;
````
````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const { tabPreferences, handleToggleDefaultTab, handleHideTab, handleShowTab } = useTabPreferences(
    workspaceSlug,
    projectId
  );
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const { tabPreferences, handleToggleDefaultTab, handleHideTab, handleShowTab } = useTabPreferences(projectId);
````

`web/apps/web/core/components/navigation/tab-navigation-utils.ts`（修改，3 处）：

````old web/apps/web/core/components/navigation/tab-navigation-utils.ts
 */

// Tab preferences type
````
````new web/apps/web/core/components/navigation/tab-navigation-utils.ts
 */

import type { ProjectTab } from "@nerve/api-client";

// Tab preferences type
````
````old web/apps/web/core/components/navigation/tab-navigation-utils.ts
  defaultTab: string;
  hiddenTabs: string[];
````
````new web/apps/web/core/components/navigation/tab-navigation-utils.ts
  defaultTab: ProjectTab;
  hiddenTabs: ProjectTab[];
````
````old web/apps/web/core/components/navigation/tab-navigation-utils.ts
export const DEFAULT_TAB_KEY = "work_items";
````
````new web/apps/web/core/components/navigation/tab-navigation-utils.ts
export const DEFAULT_TAB_KEY: ProjectTab = "work_items";
````

`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
import { Link } from "react-router";
````
````new web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
import { Link } from "react-router";
import type { ProjectTab } from "@nerve/api-client";
````
````old web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
  onToggleDefault: (tabKey: string) => void;
  onHide: (tabKey: string) => void;
````
````new web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
  onToggleDefault: (tabKey: ProjectTab) => void;
  onHide: (tabKey: ProjectTab) => void;
````

`web/apps/web/core/components/navigation/use-tab-preferences.ts`（整个文件，115 行）：

````whole web/apps/web/core/components/navigation/use-tab-preferences.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
import { DEFAULT_TAB_KEY } from "./tab-navigation-utils";
import type { TTabPreferences } from "./tab-navigation-utils";

export type TTabPreferencesHook = {
  tabPreferences: TTabPreferences;
  handleToggleDefaultTab: (tabKey: ProjectTab) => void;
  handleHideTab: (tabKey: ProjectTab) => void;
  handleShowTab: (tabKey: ProjectTab) => void;
};

/**
 * The caller's tab bar in the project's header (ProjectPreferences.navigation): the tab the project opens on and the
 * tabs under "more", nerve's default (work items, none hidden) until the project wrapper has fetched his. A change
 * shows once nerve has answered it; one nerve refuses leaves the tab bar as it was.
 *
 * @param projectId - The project ID
 * @returns Tab preferences state and handlers
 */
export const useTabPreferences = (projectId: string): TTabPreferencesHook => {
  const { getNavigation, updateNavigation } = useProjectPreferences();
  const navigation = getNavigation(projectId);

  const tabPreferences: TTabPreferences = useMemo(
    () => ({
      defaultTab: navigation?.default_tab ?? DEFAULT_TAB_KEY,
      hiddenTabs: navigation?.hide_in_more_menu ?? [],
    }),
    [navigation]
  );

  /** Replaces the tab bar; nerve replaces it whole. */
  const updatePreferences = (newPreferences: TTabPreferences): Promise<ProjectNavigation> =>
    updateNavigation(projectId, {
      default_tab: newPreferences.defaultTab,
      hide_in_more_menu: newPreferences.hiddenTabs,
    });

  /**
   * Toggle default tab setting
   * If tab is already default, resets to work_items; otherwise sets as default
   */
  const handleToggleDefaultTab = (tabKey: ProjectTab) => {
    const newDefaultTab = tabKey === tabPreferences.defaultTab ? DEFAULT_TAB_KEY : tabKey;
    const newPreferences = { ...tabPreferences, defaultTab: newDefaultTab };
    updatePreferences(newPreferences)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Default tab updated successfully.",
        });
        return;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Failed to update default tab. Please try again later.",
        });
      });
  };

  /**
   * Hide a tab (moves to overflow menu with "Show" option)
   */
  const handleHideTab = (tabKey: ProjectTab) => {
    const newPreferences = {
      ...tabPreferences,
      hiddenTabs: [...tabPreferences.hiddenTabs, tabKey],
    };
    updatePreferences(newPreferences).catch((error: unknown) => {
      console.error("Error hiding tab:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to hide tab. Please try again later.",
      });
    });
  };

  /**
   * Show a previously hidden tab (returns to visible pool)
   */
  const handleShowTab = (tabKey: ProjectTab) => {
    const newPreferences = {
      ...tabPreferences,
      hiddenTabs: tabPreferences.hiddenTabs.filter((key) => key !== tabKey),
    };
    updatePreferences(newPreferences).catch((error: unknown) => {
      console.error("Error showing tab:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again later.",
      });
    });
  };

  return {
    tabPreferences,
    handleToggleDefaultTab,
    handleHideTab,
    handleShowTab,
  };
};
````

`web/apps/web/core/components/project/form.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/form.tsx
import { usePlatformOS } from "@/hooks/use-platform-os";
// services
import { ProjectService } from "@/services/project";
````
````new web/apps/web/core/components/project/form.tsx
import { usePlatformOS } from "@/hooks/use-platform-os";
````
````old web/apps/web/core/components/project/form.tsx
}
const projectService = new ProjectService();
````
````new web/apps/web/core/components/project/form.tsx
}
````
````old web/apps/web/core/components/project/form.tsx
  const { updateProject } = useProject();
````
````new web/apps/web/core/components/project/form.tsx
  const { updateProject, checkProjectIdentifier } = useProject();
````
````old web/apps/web/core/components/project/form.tsx
      const res = await projectService.checkProjectIdentifierAvailability(workspaceSlug, payload.identifier ?? "");
      if (res.exists) setError("identifier", { message: t("common.identifier_already_exists") });
````
````new web/apps/web/core/components/project/form.tsx
      const { available } = await checkProjectIdentifier(workspaceSlug, payload.identifier ?? "");
      if (!available) setError("identifier", { message: t("common.identifier_already_exists") });
````

`web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const { tabPreferences } = useTabPreferences(workspaceSlug, projectId);
````
````new web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx
  const { tabPreferences } = useTabPreferences(projectId);
````

- [ ] **Step 3: 包装层、成员 store、类型和静态检查**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/hooks/use-workspace-members-fetch.ts",
        "web/apps/web/core/store/project/project.store.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/hooks/use-workspace-members-fetch.ts",
        "web/apps/web/core/store/project/**",
````
````old .oxlintrc.json
        "web/apps/web/core/store/project/project.store.ts",
        "web/apps/web/core/store/project/project.store.test.ts",
        "web/apps/web/core/store/project/fake-projects.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/store/project/**",
        "web/apps/web/core/hooks/store/use-project-preferences.ts",
        "web/apps/web/core/components/navigation/use-tab-preferences.ts",
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，4 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
import { useProject } from "@/hooks/store/use-project";
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
import { useProject } from "@/hooks/store/use-project";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    project: { fetchProjectMembers, fetchProjectUserProperties },
  } = useMember();
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    project: { fetchProjectMembers },
  } = useMember();
  const { fetchNavigation } = useProjectPreferences();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  // fetching project member preferences
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  // fetching the caller's tab bar in the project
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    currentUserData?.id ? () => fetchProjectUserProperties(workspaceSlug, projectId) : null,
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
    currentUserData?.id ? () => fetchNavigation(projectId) : null,
````

`web/apps/web/core/store/member/project/project-member.store.ts`（修改，11 处）：

````old web/apps/web/core/store/member/project/project-member.store.ts
import type {
  EUserProjectRoles,
  IProjectBulkAddFormData,
  IProjectUserPropertiesResponse,
  TProjectMembership,
} from "@nerve/types";
````
````new web/apps/web/core/store/member/project/project-member.store.ts
import type { EUserProjectRoles, IProjectBulkAddFormData, TProjectMembership } from "@nerve/types";
````
````old web/apps/web/core/store/member/project/project-member.store.ts
import { ProjectMemberService, ProjectService } from "@/services/project";
````
````new web/apps/web/core/store/member/project/project-member.store.ts
import { ProjectMemberService } from "@/services/project";
````
````old web/apps/web/core/store/member/project/project-member.store.ts
  };
  projectUserPropertiesMap: {
    [projectId: string]: IProjectUserPropertiesResponse;
  };
````
````new web/apps/web/core/store/member/project/project-member.store.ts
  };
````
````old web/apps/web/core/store/member/project/project-member.store.ts
  getFilteredProjectMemberDetails: (userId: string, projectId: string) => IProjectMemberDetails | null;
  getProjectUserProperties: (projectId: string) => IProjectUserPropertiesResponse | null;
````
````new web/apps/web/core/store/member/project/project-member.store.ts
  getFilteredProjectMemberDetails: (userId: string, projectId: string) => IProjectMemberDetails | null;
````
````old web/apps/web/core/store/member/project/project-member.store.ts
  ) => Promise<TProjectMembership[]>;
  fetchProjectUserProperties: (workspaceSlug: string, projectId: string) => Promise<IProjectUserPropertiesResponse>;
  // update actions
  updateProjectUserProperties: (
    workspaceSlug: string,
    projectId: string,
    data: Partial<IProjectUserPropertiesResponse>
  ) => Promise<IProjectUserPropertiesResponse>;
````
````new web/apps/web/core/store/member/project/project-member.store.ts
  ) => Promise<TProjectMembership[]>;
````
````old web/apps/web/core/store/member/project/project-member.store.ts
  } = {};
  projectUserPropertiesMap: {
    [projectId: string]: IProjectUserPropertiesResponse;
  } = {};
````
````new web/apps/web/core/store/member/project/project-member.store.ts
  } = {};
````
````old web/apps/web/core/store/member/project/project-member.store.ts
  projectMemberService;
  projectService;
````
````new web/apps/web/core/store/member/project/project-member.store.ts
  projectMemberService;
````
````old web/apps/web/core/store/member/project/project-member.store.ts
      projectMemberMap: observable,
      projectUserPropertiesMap: observable,
````
````new web/apps/web/core/store/member/project/project-member.store.ts
      projectMemberMap: observable,
````
````old web/apps/web/core/store/member/project/project-member.store.ts
      fetchProjectMembers: action,
      fetchProjectUserProperties: action,
      updateProjectUserProperties: action,
````
````new web/apps/web/core/store/member/project/project-member.store.ts
      fetchProjectMembers: action,
````
````old web/apps/web/core/store/member/project/project-member.store.ts
    this.projectMemberService = new ProjectMemberService();
    this.projectService = new ProjectService();
````
````new web/apps/web/core/store/member/project/project-member.store.ts
    this.projectMemberService = new ProjectMemberService();
````
````old web/apps/web/core/store/member/project/project-member.store.ts
  };

  /**
   * @description get project member preferences
   * @param projectId
   */
  getProjectUserProperties = computedFn(
    (projectId: string): IProjectUserPropertiesResponse | null => this.projectUserPropertiesMap[projectId] || null
  );

  /**
   * @description fetch project member preferences
   * @param workspaceSlug
   * @param projectId
   * @param data
   */
  fetchProjectUserProperties = async (
    workspaceSlug: string,
    projectId: string
  ): Promise<IProjectUserPropertiesResponse> => {
    const response = await this.projectService.getProjectUserProperties(workspaceSlug, projectId);
    runInAction(() => {
      set(this.projectUserPropertiesMap, [projectId], response);
    });
    return response;
  };

  /**
   * @description update project member preferences
   * @param workspaceSlug
   * @param projectId
   * @param data
   */
  updateProjectUserProperties = async (
    workspaceSlug: string,
    projectId: string,
    data: Partial<IProjectUserPropertiesResponse>
  ): Promise<IProjectUserPropertiesResponse> => {
    const previousProperties = this.projectUserPropertiesMap[projectId];
    try {
      // Optimistically update the store
      runInAction(() => {
        set(this.projectUserPropertiesMap, [projectId], data);
      });
      const response = await this.projectService.updateProjectUserProperties(workspaceSlug, projectId, data);
      return response;
    } catch (error) {
      // Revert on error
      runInAction(() => {
        if (previousProperties) {
          set(this.projectUserPropertiesMap, [projectId], previousProperties);
        } else {
          unset(this.projectUserPropertiesMap, [projectId]);
        }
      });
      throw error;
    }
  };
````
````new web/apps/web/core/store/member/project/project-member.store.ts
  };
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 409",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 408",
````

`web/packages/types/src/project/projects.ts`（修改，1 处）：

````old web/packages/types/src/project/projects.ts

export type IProjectMemberNavigationPreferences = {
  default_tab: string;
  hide_in_more_menu: string[];
};

````
````new web/packages/types/src/project/projects.ts

````

`web/packages/types/src/view-props.ts`（修改，2 处）：

````old web/packages/types/src/view-props.ts
import type { FC } from "react";
import type { IProjectMemberNavigationPreferences } from "./project";
````
````new web/packages/types/src/view-props.ts
import type { FC } from "react";
````
````old web/packages/types/src/view-props.ts

export interface IProjectUserPropertiesResponse extends IIssueFiltersResponse {
  sort_order: number;
  preferences: {
    navigation: IProjectMemberNavigationPreferences;
  };
}

````
````new web/packages/types/src/view-props.ts

````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外，没有命中；web 的 oxlint 408 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/navigation/project-header.tsx web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx web/apps/web/core/components/navigation/tab-navigation-root.tsx web/apps/web/core/components/navigation/tab-navigation-utils.ts web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx web/apps/web/core/components/navigation/use-tab-preferences.ts web/apps/web/core/components/project/form.tsx web/apps/web/core/components/workspace/sidebar/projects-list-item.tsx web/apps/web/core/hooks/store/use-project-preferences.ts web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/services/project/project-preferences.service.ts web/apps/web/core/services/project/project.service.ts web/apps/web/core/services/project/projects.service.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/project/index.ts web/apps/web/core/store/project/preferences.store.test.ts web/apps/web/core/store/project/preferences.store.ts web/apps/web/core/store/project/project.store.test.ts web/apps/web/core/store/project/project.store.ts web/apps/web/core/store/user/permissions.store.test.ts web/apps/web/package.json web/packages/types/src/project/projects.ts web/packages/types/src/view-props.ts
```
```bash
git commit -m "feat(M3/P8b): the caller's tab bar in a project, and the identifier check, from /api/v0

ProjectPreferencesStore keeps each project's tab bar as nerve gives it,
by the project's id, and replaces it whole through its queue, showing
nerve's answer; the tab bar's hooks read it. The project settings ask
the project store whether an identifier is free, so the form's
module-level ProjectService goes. The old ProjectService keeps the three
methods M4, M6 and M7 still call.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t5-pp-left` | 项目 store 不再给出的项目，它的标签栏仍给出 | `preferences.store.test.ts` | vitest |
| `t5-pp-request` | store 存发出的标签栏，不是 nerve 的回答 | `preferences.store.test.ts` | vitest |
| `t5-pp-before-answer` | nerve 回答之前 store 就显示标签栏（乐观） | `preferences.store.test.ts` | vitest |
| `t5-pp-no-queue` | 标签栏的修改立即发出 | `preferences.store.test.ts` | vitest |
| `t5-ps-identifier` | 标识检查问的是工作区的 slug，不是标识 | `project.store.test.ts` | vitest |
| `t5-pp-change-swallows` | 标签栏的修改被拒绝时给出空值，不失败 | `preferences.store.test.ts` | vitest |
| `t5-pp-fetch-swallows` | 标签栏被拒绝时给出 `undefined`，不失败 | `preferences.store.test.ts` | vitest |
| `t5-pp-fetch-queued` | 标签栏的取数排进修改的队列 | `preferences.store.test.ts` | vitest |
| `t5-rc-one-entry` | 每个键都取进同一个值：一个项目的取数或修改写到别的项目上 | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t5-ps-identifier-swallows` | 被拒绝的标识检查说标识可用，不失败 | `project.store.test.ts` | vitest |

---

### Task 6: 项目成员的 service 和 store

**Files:**
- Create: `web/apps/web/core/services/project/project-members.service.ts`、`web/apps/web/core/store/member/project/project-member.store.test.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/dropdowns/member/dropdown.tsx`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/member-list-item.tsx`、`web/apps/web/core/components/project/member-list.tsx`、`web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/services/project/index.ts`、`web/apps/web/core/store/issue/issue-details/sub_issues.store.ts`、`web/apps/web/core/store/member/index.ts`、`web/apps/web/core/store/member/project/project-member-filters.store.ts`、`web/apps/web/core/store/member/project/project-member.store.ts`、`web/apps/web/core/store/member/utils.ts`、`web/apps/web/core/store/project/project.store.ts`、`web/apps/web/package.json`、`web/packages/types/src/project/projects.ts`
- Delete: `web/apps/web/core/services/project/project-member.service.ts`

**Interfaces:**
- Produces（spec 2.6；M3 设计 7.2、7.3；M2 交接第 3 节）：
  - `core/services/project/project-members.service.ts`：`class ProjectMembersService { list(projectId): Promise<ProjectMember[]>; add(projectId, data: ProjectMembersAdd): Promise<ProjectMember[]>; update(membershipId, data: ProjectMemberUpdate): Promise<ProjectMember>; remove(membershipId): Promise<void> }`（`/api/v0/projects/{id}/members`、`/api/v0/project-members/{id}`）。旧的 `project-member.service.ts` 删除。
  - `core/store/member/project/project-member.store.ts`（整个文件）：`ProjectMemberStore(memberRoot: Pick<IMemberRootStore, "memberMap">, rootStore, api)`；每个项目的成员关系按成员的账户 id 存，项目按它的 id 存（`ReconciledByKey`）；项目 store 不再给出这个项目时什么都不给（离开、删除，或工作区已不是调用者的）。`projectMemberIds`（地址的项目，按成员页的筛选和顺序）、`getProjectMemberDetails(userId, projectId)`（成员关系连同工作区成员给出的公开资料，`IProjectMemberDetails = ProjectMember & { member: MemberUser }`）、`getProjectMemberIds(projectId, includeGuestUsers)`（调用者在最前，其余按显示名）、`getFilteredProjectMemberDetails`。`fetchProjectMembers(projectId)`；修改经 `changes = oneAtATime()`，写入回答：`bulkAddMembersToProject(projectId, data)`、`updateMemberRole(projectId, userId, role)`、`removeMemberFromProject(projectId, userId)`；store 没有列出这个人时不问 nerve、直接失败（`Member not found`）。修改改到的项目一侧（`member_ids`、调用者自己的 `member_role`）经 `ProjectStore.confirmProject` 写到项目 store 显示这个项目的每一处（列表和它自己的读），也重放到取数在外的回答上（Task 3 的过渡在这里结束）。Plane 的乐观修改和回滚删除。
  - `ProjectStore.confirmProject(project, change)`。
  - `TProjectMembership`、`IProjectBulkAddFormData` 删除；成员行（`member-columns.tsx`、`useProjectColumns.tsx`）直接用 `IProjectMemberDetails`，`RowData` 删除；角色是 `ProjectMember.role`（Plane 的 `original_role` 没有了）；加成员的弹窗的表单值是生成的 `ProjectMembersAdd`，角色取工作区成员的角色。成员下拉框没有调用方传的 `memberIds` 删除。项目包装层取成员改调新签名（过渡：仍是原来的 `useSWR`，Task 10 换掉）。
  - `.oxlintrc.json`：`core/store/member/project/**` 加进两条规则的范围，`project-members.service.ts` 加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest）`core/store/member/project/project-member.store.test.ts`：
- `ProjectMemberStore, the members`：`keeps a project's memberships as nerve lists them, each with its member's profile`；`shows none of the members of a project the caller left`；`fails when nerve refuses them, keeping none, and again, keeping the members it had`；`keeps the members it had, gives nothing and does not fail, when the session changes as it fetches again`。
- `ProjectMemberStore, the changes`：`adds the members nerve answers, whom the project then has among its members`；`changes a role to nerve's answer, and the caller's own role in the project with his`（回答的 `created_at` 与列表的不同）；`removes a member, whom the project then no longer has among its members`；`fails, changing nothing, when nerve refuses $change`（3 行）；`fails, asking nerve nothing, for $change of a member it has not listed`（2 行）；`sends each change once nerve has answered the one before it, refused or not`；`fetches the members while a change is out: a fetch does not wait for it`。
- `ProjectMemberStore, while a fetch is out`：`keeps an addition, a role change and a removal nerve confirmed during a refetch`；`lets each project's newer fetch write: an older one answering last writes nothing`（成员关系按成员存，"已列出的又被加入"不会列两次，文件里说明）。

- [ ] **Step 1: service 和 store**

`web/apps/web/core/services/project/index.ts`（修改，1 处）：

````old web/apps/web/core/services/project/index.ts
export * from "./project.service";
export * from "./project-member.service";
````
````new web/apps/web/core/services/project/index.ts
export * from "./project.service";
````

`web/apps/web/core/services/project/project-member.service.ts`（删除）：

````delete web/apps/web/core/services/project/project-member.service.ts
````

`web/apps/web/core/services/project/project-members.service.ts`（新文件，49 行）：

````file web/apps/web/core/services/project/project-members.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, ProjectMember, ProjectMembersAdd, ProjectMemberUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A project's memberships (M3 design 5.1, 7.3); the caller's own joining and leaving are the projects service's. */
export class ProjectMembersService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's active memberships, in the order they began. */
  async list(projectId: string): Promise<ProjectMember[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/members", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** Adds members of the workspace to the project, all of them or none; the answer is their memberships. */
  async add(projectId: string, data: ProjectMembersAdd): Promise<ProjectMember[]> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/members", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    ).data;
  }

  /** Changes a member's role; the answer is the membership with its new role. */
  async update(membershipId: string, data: ProjectMemberUpdate): Promise<ProjectMember> {
    return unwrap(
      await this.api.PATCH("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membershipId } },
        body: data,
      })
    );
  }

  /** Ends a member's membership of the project. */
  async remove(membershipId: string): Promise<void> {
    unwrap(
      await this.api.DELETE("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membershipId } },
      })
    );
  }
}
````

`web/apps/web/core/store/member/index.ts`（修改，1 处）：

````old web/apps/web/core/store/member/index.ts
    this.project = new ProjectMemberStore(this, _rootStore);
````
````new web/apps/web/core/store/member/index.ts
    this.project = new ProjectMemberStore(this, _rootStore, api);
````

`web/apps/web/core/store/member/project/project-member-filters.store.ts`（修改，4 处）：

````old web/apps/web/core/store/member/project/project-member-filters.store.ts
import type { MemberUser } from "@nerve/api-client";
import type { TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/store/member/project/project-member-filters.store.ts
import type { MemberUser, ProjectMember } from "@nerve/api-client";
````
````old web/apps/web/core/store/member/project/project-member-filters.store.ts
  getFilteredMemberIds: (
    members: TProjectMembership[],
    memberDetailsMap: Record<string, MemberUser>,
    getMemberKey: (member: TProjectMembership) => string,
````
````new web/apps/web/core/store/member/project/project-member-filters.store.ts
  getFilteredMemberIds: (
    members: ProjectMember[],
    memberDetailsMap: Record<string, MemberUser>,
    getMemberKey: (member: ProjectMember) => string,
````
````old web/apps/web/core/store/member/project/project-member-filters.store.ts
      members: TProjectMembership[],
````
````new web/apps/web/core/store/member/project/project-member-filters.store.ts
      members: ProjectMember[],
````
````old web/apps/web/core/store/member/project/project-member-filters.store.ts
      getMemberKey: (member: TProjectMembership) => string,
````
````new web/apps/web/core/store/member/project/project-member-filters.store.ts
      getMemberKey: (member: ProjectMember) => string,
````

`web/apps/web/core/store/member/project/project-member.store.test.ts`（新文件，322 行）：

````file web/apps/web/core/store/member/project/project-member.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectMember, ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { ProjectMemberStore } from "@/store/member/project/project-member.store";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { ProjectRootStore } from "@/store/project";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RouterStore } from "@/store/router.store";
import { UserStore } from "@/store/user";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The members of a project (M3 design 7.3), against a fake nerve that answers each request when the test says. The
// caller is ann, the admin of acme and of web, whose address the tab has.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const acme = workspaceOf("acme", { role: 20 });
const web = projectOf("WEB", acme.id, { member_role: 20, member_ids: ["u-ann", "u-bob", "u-cat"] });
const ops = projectOf("OPS", acme.id);
const MEMBERS = `/api/v0/projects/${web.id}/members`;

/** A membership of a project (web unless it says) as nerve lists it: the name names the member and the membership. */
const memberOf = (name: string, role: ProjectRole = 15, projectId = web.id): ProjectMember => ({
  id: `pm-${projectId}-${name}`,
  project_id: projectId,
  member_id: `u-${name}`,
  role,
  created_at: "2026-10-01T09:00:00Z",
});
const ann = memberOf("ann", 20);
const bob = memberOf("bob");
const cat = memberOf("cat", 5);
const dee = memberOf("dee");
/** Bob's membership as nerve answers his change to a guest: its created_at, which only nerve gives, is not the list's. */
const demoted: ProjectMember = { ...memberOf("bob", 5), created_at: "2026-10-03T09:00:00Z" };
/** Dee, a member of the workspace, added to web. */
const adding: ProjectMembersAdd = { members: [{ member_id: "u-dee", role: 15 }] };
/** The profiles the workspace's members gave. */
const profile = (name: string) => membershipOf(name).member;
const membership = (name: string) => `/api/v0/project-members/pm-${web.id}-${name}`;

/** The store of a tab at web's address, whose caller is ann, and whose projects nerve listed: web and ops. */
async function memberStore() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  router.setQuery({ workspaceSlug: "acme", projectId: web.id });
  const user = new UserStore(fakeRoot({ router }), api);
  runInAction(() => {
    const account = { email: "ann@example.com", user_timezone: "UTC", cover_image_url: null };
    user.data = { ...profile("ann"), ...account, created_at: "2026-09-01T09:00:00Z" };
  });
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  await loadWorkspaces(nerve, workspaceRoot, [acme]);
  await loadProjects(nerve, projectRoot.project, acme, [web, ops]);
  nerve.calls.length = 0;
  const users = Object.fromEntries(["ann", "bob", "cat", "dee"].map((name) => [`u-${name}`, profile(name)]));
  const store = new ProjectMemberStore({ memberMap: users }, fakeRoot({ router, user, projectRoot }), api);
  return { nerve, api, projects: projectRoot.project, store };
}

/** The store fetches the members of a project (web unless it says), and nerve lists these. */
function load(nerve: FakeNerve, store: ProjectMemberStore, memberships: ProjectMember[], projectId = web.id) {
  const fetch = () => store.fetchProjectMembers(projectId);
  return answered(nerve, fetch, ["GET", `/api/v0/projects/${projectId}/members`], { data: memberships }, "members");
}

/** A store whose members of web nerve gave as ann, bob and cat. */
async function loaded() {
  const tab = await memberStore();
  await load(tab.nerve, tab.store, [ann, bob, cat]);
  return tab;
}

/** Sends a change, which nerve gives reply once it is the request out; gives how the change settled. */
async function sent(nerve: FakeNerve, send: () => Promise<unknown>, request: Endpoint, reply: Response) {
  const k = nerve.calls.length;
  const change = track(send());
  await inTurn(nerve, k, request, reply);
  await until(() => change.settled, "the change");
  return change;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectMemberStore, the members", () => {
  it("keeps a project's memberships as nerve lists them, each with its member's profile", async () => {
    const { nerve, store } = await memberStore();
    const fetched = await load(nerve, store, [bob, cat, ann]);
    expect(fetched.value).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...bob, member: profile("bob") });
    expect(store.getProjectMemberDetails("u-dee", web.id)).toBeNull();
    // the caller first, then by display name; the guest when asked for
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
    expect(store.getProjectMemberIds(web.id, false)).toEqual(["u-ann", "u-bob"]);
    expect(store.getProjectMemberIds(ops.id, true)).toBeNull();
    // the address's project's, by the members page's order and filters
    store.filters.updateFilters(web.id, { order_by: "-display_name" });
    expect(store.projectMemberIds).toEqual(["u-cat", "u-bob", "u-ann"]);
    store.filters.updateFilters(web.id, { roles: ["5"] });
    expect(store.getFilteredProjectMemberDetails("u-cat", web.id)).toEqual({ ...cat, member: profile("cat") });
    expect(store.getFilteredProjectMemberDetails("u-bob", web.id)).toBeNull();
  });

  it("shows none of the members of a project the caller left", async () => {
    const { nerve, projects, store } = await loaded();
    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    expect(store.getProjectMemberIds(web.id, true)).toBeNull();
    expect(store.getProjectMemberDetails("u-bob", web.id)).toBeNull();
    expect(store.projectMemberIds).toBeNull();
  });

  it("fails when nerve refuses them, keeping none, and again, keeping the members it had", async () => {
    const { nerve, store } = await memberStore();
    const refused = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 1, "the members");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectMemberIds(web.id, true)).toBeNull();

    await load(nerve, store, [ann, bob]);
    const again = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob"]);
  });

  it("keeps the members it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjectMembers(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
  });
});

describe("ProjectMemberStore, the changes", () => {
  it("adds the members nerve answers, whom the project then has among its members", async () => {
    const { nerve, projects, store } = await loaded();
    const added = track(store.bulkAddMembersToProject(web.id, adding));
    await until(() => nerve.calls.length === 2, "the addition");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: MEMBERS, body: adding });
    expect(store.getProjectMemberDetails("u-dee", web.id)).toBeNull();
    nerve.calls[1]?.answer(json(201, { data: [dee] }));
    await until(() => added.settled, "the answer");
    expect(added.value).toEqual([dee]);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat", "u-dee"]);
    expect(projects.getProjectById(web.id)?.member_ids).toEqual(["u-ann", "u-bob", "u-cat", "u-dee"]);
  });

  it("changes a role to nerve's answer, and the caller's own role in the project with his", async () => {
    const { nerve, projects, store } = await loaded();
    const changed = track(store.updateMemberRole(web.id, "u-bob", 5));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: membership("bob"), body: { role: 5 } });
    expect(store.getProjectMemberDetails("u-bob", web.id)?.role).toBe(15);
    nerve.calls[1]?.answer(json(200, demoted));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(demoted);
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...demoted, member: profile("bob") });
    expect(projects.getProjectById(web.id)?.member_role).toBe(20);

    // ann, an admin of the workspace, makes herself a member of the project
    const own: ProjectMember = { ...ann, role: 15 };
    await sent(nerve, () => store.updateMemberRole(web.id, "u-ann", 15), ["PATCH", membership("ann")], json(200, own));
    expect(store.getProjectMemberDetails("u-ann", web.id)?.role).toBe(15);
    expect(projects.getProjectById(web.id)?.member_role).toBe(15);
  });

  it("removes a member, whom the project then no longer has among its members", async () => {
    const { nerve, projects, store } = await loaded();
    const removed = await sent(
      nerve,
      () => store.removeMemberFromProject(web.id, "u-bob"),
      ["DELETE", membership("bob")],
      noContent()
    );
    expect(removed.error).toBeUndefined();
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-cat"]);
    expect(projects.getProjectById(web.id)?.member_ids).toEqual(["u-ann", "u-cat"]);
  });

  const refusals: { change: string; send: (store: ProjectMemberStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "an addition",
      send: (store) => store.bulkAddMembersToProject(web.id, adding),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a role change",
      send: (store) => store.updateMemberRole(web.id, "u-bob", 20),
      refusal: problem(409, "project.role_too_high"),
    },
    {
      change: "a removal",
      send: (store) => store.removeMemberFromProject(web.id, "u-ann"),
      refusal: problem(409, "project.own_membership"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, projects, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
    expect(store.getProjectMemberDetails("u-bob", web.id)?.role).toBe(15);
    expect(projects.getProjectById(web.id)).toEqual(web);
  });

  it.each(refusals.slice(1))(
    "fails, asking nerve nothing, for $change of a member it has not listed",
    async ({ send }) => {
      const { nerve, store } = await memberStore();
      const refused = await settle(send(store), "the change");
      expect(refused).toMatchObject({ settled: true, error: new Error("Member not found") });
      expect(nerve.calls).toEqual([]);
    }
  );

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const promoted = track(store.updateMemberRole(web.id, "u-bob", 20));
    const changed = track(store.updateMemberRole(web.id, "u-bob", 5));
    const removed = track(store.removeMemberFromProject(web.id, "u-bob"));
    await inTurn(nerve, 1, ["PATCH", membership("bob")], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", membership("bob")], json(200, demoted));
    await inTurn(nerve, 3, ["DELETE", membership("bob")], noContent());
    await until(() => removed.settled, "the last change");
    expect(promoted.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(demoted);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-cat"]);
  });

  it("fetches the members while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateMemberRole(web.id, "u-bob", 5), request: ["PATCH", membership("bob")] },
      { send: () => store.fetchProjectMembers(web.id), request: ["GET", MEMBERS], body: { data: [ann] } }
    );
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann"]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one project's members, only the newer writes. An addition the refetch lists already cannot show twice:
// the memberships are kept by member.
describe("ProjectMemberStore, while a fetch is out", () => {
  it("keeps an addition, a role change and a removal nerve confirmed during a refetch", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(
      nerve,
      () => store.bulkAddMembersToProject(web.id, adding),
      ["POST", MEMBERS],
      json(201, { data: [dee] })
    );
    await sent(
      nerve,
      () => store.updateMemberRole(web.id, "u-bob", 5),
      ["PATCH", membership("bob")],
      json(200, demoted)
    );
    await sent(nerve, () => store.removeMemberFromProject(web.id, "u-cat"), ["DELETE", membership("cat")], noContent());
    // the list was read before all three
    nerve.calls[1]?.answer(json(200, { data: [ann, bob, cat] }));
    await until(() => refetched.settled, "the refetch");

    expect(refetched.value).toEqual({ "u-ann": ann, "u-bob": demoted, "u-dee": dee });
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-dee"]);
  });

  it("lets each project's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await memberStore();
    const older = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 1, "the older members");
    const elsewhere = track(store.fetchProjectMembers(ops.id));
    await until(() => nerve.calls.length === 2, "ops's members");
    const newer = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 3, "the newer members");
    nerve.calls[2]?.answer(json(200, { data: [ann, demoted] }));
    await until(() => newer.settled, "the newer members");
    // a newer fetch of web's members does not overtake one of ops's
    nerve.calls[1]?.answer(json(200, { data: [memberOf("dee", 15, ops.id)] }));
    await until(() => elsewhere.settled, "ops's members");
    // read before bob's change
    nerve.calls[0]?.answer(json(200, { data: [ann, bob] }));
    await until(() => older.settled, "the older members");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob"]);
    expect(store.getProjectMemberDetails("u-bob", web.id)?.role).toBe(5);
    expect(store.getProjectMemberIds(ops.id, true)).toEqual(["u-dee"]);
  });
});
````

`web/apps/web/core/store/member/project/project-member.store.ts`（整个文件，213 行）：

````whole web/apps/web/core/store/member/project/project-member.store.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { omit, sortBy, union, without } from "lodash-es";
import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import type { ApiClient, MemberUser, Project, ProjectMember, ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { ProjectMembersService } from "@/services/project/project-members.service";
// store
import type { IProjectStore } from "@/store/project/project.store";
import type { RootStore } from "@/store/root.store";
import type { IRouterStore } from "@/store/router.store";
import type { IUserStore } from "@/store/user";
// local imports
import type { IMemberRootStore } from "../index";
import { sortProjectMembers } from "../utils";
import type { IProjectMemberFiltersStore } from "./project-member-filters.store";
import { ProjectMemberFiltersStore } from "./project-member-filters.store";

/** A project's active memberships by the member's account id. */
type Memberships = Record<string, ProjectMember>;

/** A membership of a project with its member's profile, which the workspace's members give (M3 design 7.2). */
export type IProjectMemberDetails = ProjectMember & { member: MemberUser };

export interface IProjectMemberStore {
  // filters store
  filters: IProjectMemberFiltersStore;
  // computed
  projectMemberIds: string[] | null;
  // computed actions
  getProjectMemberDetails: (userId: string, projectId: string) => IProjectMemberDetails | null;
  getProjectMemberIds: (projectId: string, includeGuestUsers: boolean) => string[] | null;
  getFilteredProjectMemberDetails: (userId: string, projectId: string) => IProjectMemberDetails | null;
  // fetch actions
  fetchProjectMembers: (projectId: string) => Promise<Memberships | undefined>;
  // changes
  bulkAddMembersToProject: (projectId: string, data: ProjectMembersAdd) => Promise<ProjectMember[]>;
  updateMemberRole: (projectId: string, userId: string, role: ProjectRole) => Promise<ProjectMember>;
  removeMemberFromProject: (projectId: string, userId: string) => Promise<void>;
}

/** The memberships by their members' account ids. */
const byMember = (memberships: ProjectMember[]): Memberships =>
  Object.fromEntries(memberships.map((membership) => [membership.member_id, membership]));

/**
 * The members of the projects of a session (M3 design 7.3), each project's by its id: its service sends with the
 * session's client, which the RootStore of the session hands down. Changes go one at a time and the store writes
 * nerve's answers (v0 design 7.7); fetches do not queue. The members of a project the project store no longer gives
 * (deleted, left, or of a workspace no longer the caller's) do not show. What a change makes of the project, its
 * member_ids and the caller's own role (member_role), the project store shows too.
 */
export class ProjectMemberStore implements IProjectMemberStore {
  /** Each project's memberships, reconciled between their fetches and the changes nerve confirmed (reconciled.ts). */
  private readonly members = new ReconciledByKey<Memberships>();
  // filters store
  filters: IProjectMemberFiltersStore;
  // stores
  private readonly routerStore: IRouterStore;
  private readonly userStore: IUserStore;
  /** The users the stores read, which the workspace's members' profiles fill. */
  private readonly memberRoot: Pick<IMemberRootStore, "memberMap">;
  /** The projects the caller sees, which show what the changes of their members make of them. */
  private readonly projects: Pick<IProjectStore, "getProjectById" | "confirmProject">;
  // services
  private readonly service: ProjectMembersService;
  /** The changes of the memberships, sent one at a time. */
  private readonly changes = oneAtATime();

  constructor(_memberRoot: Pick<IMemberRootStore, "memberMap">, _rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      projectMemberIds: computed,
      // actions
      fetchProjectMembers: action,
      bulkAddMembersToProject: action,
      updateMemberRole: action,
      removeMemberFromProject: action,
    });
    this.routerStore = _rootStore.router;
    this.userStore = _rootStore.user;
    this.memberRoot = _memberRoot;
    this.projects = _rootStore.projectRoot.project;
    this.filters = new ProjectMemberFiltersStore();
    this.service = new ProjectMembersService(api);
  }

  /** The project's memberships, once fetched, while the project store gives the project. */
  private membershipsOf(projectId: string): Memberships | undefined {
    return this.projects.getProjectById(projectId) ? this.members.get(projectId) : undefined;
  }

  /**
   * @description the members of the address's project, by the members page's filters and order; null until fetched
   * or while it has none
   */
  get projectMemberIds() {
    const projectId = this.routerStore.projectId;
    if (!projectId) return null;
    const members = Object.values(this.membershipsOf(projectId) ?? {});
    if (members.length === 0) return null;
    const sortedMembers = sortProjectMembers(
      members,
      this.memberRoot.memberMap,
      (member) => member.member_id,
      this.filters.filtersMap[projectId]
    );
    return sortedMembers.map((member) => member.member_id);
  }

  /** @description a member's membership of the project with his profile; null without either */
  getProjectMemberDetails = computedFn((userId: string, projectId: string): IProjectMemberDetails | null => {
    const membership = this.membershipsOf(projectId)?.[userId];
    const member = this.memberRoot.memberMap[userId];
    return membership && member ? { ...membership, member } : null;
  });

  /** @description the project's members, the caller first, then by display name; null until fetched */
  getProjectMemberIds = computedFn((projectId: string, includeGuestUsers: boolean): string[] | null => {
    const memberships = this.membershipsOf(projectId);
    if (!memberships) return null;
    const members = Object.values(memberships).filter(
      (membership) => includeGuestUsers || membership.role !== EUserPermissions.GUEST
    );
    return sortBy(members, [
      (membership) => membership.member_id !== this.userStore.data?.id,
      (membership) => this.memberRoot.memberMap[membership.member_id]?.display_name.toLowerCase(),
    ]).map((membership) => membership.member_id);
  });

  /** @description as getProjectMemberDetails, for a member the members page's filters let through */
  getFilteredProjectMemberDetails = computedFn((userId: string, projectId: string): IProjectMemberDetails | null => {
    const memberships = Object.values(this.membershipsOf(projectId) ?? {});
    const shown = this.filters.getFilteredMemberIds(
      memberships,
      this.memberRoot.memberMap,
      (membership) => membership.member_id,
      projectId
    );
    return shown.includes(userId) ? this.getProjectMemberDetails(userId, projectId) : null;
  });

  /**
   * @description fetches a project's members, a member's to fetch as nerve refuses anyone else, and shows them with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchProjectMembers = (projectId: string): Promise<Memberships | undefined> =>
    this.members.fetch(projectId, async () => byMember(await this.service.list(projectId)));

  /**
   * @description adds members of the workspace to a project, all of them or none; the store then has their
   * memberships as nerve answers them. Fails, changing nothing, when nerve refuses.
   */
  bulkAddMembersToProject = (projectId: string, data: ProjectMembersAdd): Promise<ProjectMember[]> =>
    this.changes(async () => {
      const added = await this.service.add(projectId, data);
      this.members.confirm(projectId, (memberships) => ({ ...memberships, ...byMember(added) }));
      const ids = added.map((membership) => membership.member_id);
      this.confirmOnProject(projectId, (project) => ({ ...project, member_ids: union(project.member_ids, ids) }));
      return added;
    });

  /**
   * @description changes a member's role; the store then has nerve's answer, and the project the caller's own role
   * when it is his. Fails, changing nothing, when nerve refuses or the store has no membership of his.
   */
  updateMemberRole = (projectId: string, userId: string, role: ProjectRole): Promise<ProjectMember> =>
    this.changes(async () => {
      const membership = await this.service.update(this.membership(projectId, userId).id, { role });
      this.members.confirm(projectId, (memberships) => ({ ...memberships, [userId]: membership }));
      if (userId === this.userStore.data?.id) {
        this.confirmOnProject(projectId, (project) => ({ ...project, member_role: membership.role }));
      }
      return membership;
    });

  /**
   * @description ends a member's membership of a project, which the store then no longer has. Fails, changing
   * nothing, when nerve refuses (the caller's own: he leaves the project instead) or the store has no membership of
   * his.
   */
  removeMemberFromProject = (projectId: string, userId: string): Promise<void> =>
    this.changes(async () => {
      await this.service.remove(this.membership(projectId, userId).id);
      this.members.confirm(projectId, (memberships) => omit(memberships, userId));
      this.confirmOnProject(projectId, (project) => ({ ...project, member_ids: without(project.member_ids, userId) }));
    });

  /** The membership of the member userId names in the project, as the store has it; fails when it has none. */
  private membership(projectId: string, userId: string): ProjectMember {
    const membership = this.membershipsOf(projectId)?.[userId];
    if (!membership) throw new Error("Member not found");
    return membership;
  }

  /** What a change nerve confirmed makes of the project, wherever the project store shows it. */
  private confirmOnProject(projectId: string, change: Change<Project>): void {
    const project = this.projects.getProjectById(projectId);
    if (project) this.projects.confirmProject(project, change);
  }
}
````

`web/apps/web/core/store/member/utils.ts`（修改，6 处）：

````old web/apps/web/core/store/member/utils.ts
import type { MemberUser } from "@nerve/api-client";
import type { TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/store/member/utils.ts
import type { MemberUser, ProjectMember } from "@nerve/api-client";
````
````old web/apps/web/core/store/member/utils.ts
const filterProjectMembersByRole = (members: TProjectMembership[], roleFilters: string[]): TProjectMembership[] => {
````
````new web/apps/web/core/store/member/utils.ts
const filterProjectMembersByRole = (members: ProjectMember[], roleFilters: string[]): ProjectMember[] => {
````
````old web/apps/web/core/store/member/utils.ts
  return members.filter((member) => {
    const memberRole = String(member.role ?? member.original_role ?? "");
    return roleFilters.includes(memberRole);
  });
````
````new web/apps/web/core/store/member/utils.ts
  return members.filter((member) => roleFilters.includes(String(member.role)));
````
````old web/apps/web/core/store/member/utils.ts
  members: TProjectMembership[],
````
````new web/apps/web/core/store/member/utils.ts
  members: ProjectMember[],
````
````old web/apps/web/core/store/member/utils.ts
  getMemberKey: (member: TProjectMembership) => string,
  filters?: IMemberFilters
): TProjectMembership[] => {
````
````new web/apps/web/core/store/member/utils.ts
  getMemberKey: (member: ProjectMember) => string,
  filters?: IMemberFilters
): ProjectMember[] => {
````
````old web/apps/web/core/store/member/utils.ts
    (member) => String(member.role ?? member.original_role ?? ""),
````
````new web/apps/web/core/store/member/utils.ts
    (member) => String(member.role),
````

`web/apps/web/core/store/project/project.store.ts`（修改，2 处）：

````old web/apps/web/core/store/project/project.store.ts
  updateProjectSortOrder: (project: ProjectRef, sortOrder: number) => Promise<void>;
````
````new web/apps/web/core/store/project/project.store.ts
  updateProjectSortOrder: (project: ProjectRef, sortOrder: number) => Promise<void>;
  confirmProject: (project: ProjectRef, change: Change<Project>) => void;
````
````old web/apps/web/core/store/project/project.store.ts
      const placed: Change<Project[]> = (list) =>
        list.map((held) => (held.id === project.id ? { ...held, sort_order } : held));
      this.unarchived.confirm(project.workspace_id, placed);
      this.details.confirm(project.id, (held) => held && { ...held, sort_order });
    });
````
````new web/apps/web/core/store/project/project.store.ts
      this.confirmProject(project, (held) => ({ ...held, sort_order }));
    });

  /**
   * @description a change nerve confirmed to what the caller sees of a project, by another store (its members, his
   * role in it) or this one: it is made wherever the store shows the project, and on the answers of the fetches out
   * (reconciled.ts)
   */
  confirmProject = (project: ProjectRef, change: Change<Project>): void => {
    const inPlace: Change<Project[]> = (list) => list.map((held) => (held.id === project.id ? change(held) : held));
    this.unarchived.confirm(project.workspace_id, inPlace);
    this.archived.confirm(project.workspace_id, inPlace);
    this.details.confirm(project.id, (held) => held && change(held));
  };
````

- [ ] **Step 2: 使用方、类型和静态检查**

`.oxlintrc.json`（修改，3 处）：

````old .oxlintrc.json
        "web/apps/web/core/store/workspace/**",
        "web/apps/web/core/store/member/workspace/**",
        "web/apps/web/core/components/workspace/settings/**",
````
````new .oxlintrc.json
        "web/apps/web/core/store/workspace/**",
        "web/apps/web/core/store/member/workspace/**",
        "web/apps/web/core/store/member/project/**",
        "web/apps/web/core/components/workspace/settings/**",
````
````old .oxlintrc.json
        "web/apps/web/core/store/workspace/**",
        "web/apps/web/core/store/member/workspace/**",
        "web/apps/web/core/store/user/permissions.store.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/store/workspace/**",
        "web/apps/web/core/store/member/workspace/**",
        "web/apps/web/core/store/member/project/**",
        "web/apps/web/core/store/user/permissions.store.ts",
````
````old .oxlintrc.json
        "web/apps/web/core/services/project/project-preferences.service.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/services/project/project-preferences.service.ts",
        "web/apps/web/core/services/project/project-members.service.ts",
````

`web/apps/web/core/components/dropdowns/member/dropdown.tsx`（修改，5 处）：

````old web/apps/web/core/components/dropdowns/member/dropdown.tsx
import { observer } from "mobx-react";
import { useParams } from "react-router";
````
````new web/apps/web/core/components/dropdowns/member/dropdown.tsx
import { observer } from "mobx-react";
````
````old web/apps/web/core/components/dropdowns/member/dropdown.tsx
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  memberIds?: string[];
````
````new web/apps/web/core/components/dropdowns/member/dropdown.tsx
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
````
````old web/apps/web/core/components/dropdowns/member/dropdown.tsx
  const { memberIds: propsMemberIds, projectId } = props;
  // router params
  const { workspaceSlug } = useParams();
````
````new web/apps/web/core/components/dropdowns/member/dropdown.tsx
  const { projectId } = props;
````
````old web/apps/web/core/components/dropdowns/member/dropdown.tsx
  const memberIds = propsMemberIds
    ? propsMemberIds
    : projectId
      ? getProjectMemberIds(projectId, false)
      : workspaceMemberIds;
````
````new web/apps/web/core/components/dropdowns/member/dropdown.tsx
  const memberIds = projectId ? getProjectMemberIds(projectId, false) : workspaceMemberIds;
````
````old web/apps/web/core/components/dropdowns/member/dropdown.tsx
    if (!memberIds && projectId && workspaceSlug) fetchProjectMembers(workspaceSlug, projectId);
````
````new web/apps/web/core/components/dropdowns/member/dropdown.tsx
    if (!memberIds && projectId) fetchProjectMembers(projectId);
````

`web/apps/web/core/components/project/add-project-members-modal.tsx`（修改，11 处）：

````old web/apps/web/core/components/project/add-project-members-modal.tsx
// nerve imports
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
// nerve imports
import type { ProjectMembersAdd } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  onClose: () => void;
  onSuccess?: () => void;
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  onClose: () => void;
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
type member = {
  role: EUserPermissions;
  member_id: string;
};

type FormValues = {
  members: member[];
};
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
type FormValues = ProjectMembersAdd;
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  const { isOpen, onClose, onSuccess, projectId, workspaceSlug } = props;
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  const { isOpen, onClose, projectId, workspaceSlug } = props;
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  const nonProjectMemberIds = workspaceMemberIds?.filter((userId) => {
    const projectMemberDetails = getProjectMemberDetails(userId, projectId);
    const isProjectMember = projectMemberDetails?.member.id && projectMemberDetails?.original_role;
    return !isProjectMember;
  });
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  const nonProjectMemberIds = workspaceMemberIds?.filter(
    (userId) => getProjectMemberDetails(userId, projectId) === null
  );
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx

    const payload = { ...formData };

````
````new web/apps/web/core/components/project/add-project-members-modal.tsx

````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
      await bulkAddMembersToProject(workspaceSlug, projectId, payload);
      if (onSuccess) onSuccess();
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
      await bulkAddMembersToProject(projectId, formData);
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  const options = nonProjectMemberIds
    ?.map((userId) => {
      const memberDetails = getWorkspaceMemberDetails(userId);
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  const options = nonProjectMemberIds?.flatMap((userId) => {
    const memberDetails = getWorkspaceMemberDetails(userId);
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
      if (!memberDetails?.member) return;
      return {
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
    if (!memberDetails?.member) return [];
    return [
      {
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
      };
    })
    .filter((option) => !!option) as
    | {
        value: string;
        query: string;
        content: React.ReactNode;
      }[]
    | undefined;
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
      },
    ];
  });
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                            const workspaceMemberDetails = getWorkspaceMemberDetails(val);
                            const workspaceRole = workspaceMemberDetails?.role ?? 5;
                            const newValue = ROLE[workspaceRole].toUpperCase();
                            setValue(
                              `members.${index}.role`,
                              EUserPermissions[newValue as keyof typeof EUserPermissions]
                            );
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                            setValue(`members.${index}.role`, getWorkspaceMemberDetails(val)?.role ?? 5);
````

`web/apps/web/core/components/project/member-list-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/project/member-list-item.tsx
      await removeMemberFromProject(workspaceSlug, projectId, memberId).catch((err) =>
````
````new web/apps/web/core/components/project/member-list-item.tsx
      await removeMemberFromProject(projectId, memberId).catch((err) =>
````
````old web/apps/web/core/components/project/member-list-item.tsx
        data={(memberDetails?.filter((member): member is IProjectMemberDetails => member !== null) ?? []) as any}
````
````new web/apps/web/core/components/project/member-list-item.tsx
        data={memberDetails.filter((member): member is IProjectMemberDetails => member !== null)}
````

`web/apps/web/core/components/project/member-list.tsx`（修改，2 处）：

````old web/apps/web/core/components/project/member-list.tsx
    if (!memberDetails?.member || !memberDetails.original_role) return false;
````
````new web/apps/web/core/components/project/member-list.tsx
    if (!memberDetails) return false;
````
````old web/apps/web/core/components/project/member-list.tsx
              value={searchQuery}
              autoFocus
````
````new web/apps/web/core/components/project/member-list.tsx
              value={searchQuery}
````

`web/apps/web/core/components/project/settings/member-columns.tsx`（修改，8 处）：

````old web/apps/web/core/components/project/settings/member-columns.tsx
import type { MemberUser, User } from "@nerve/api-client";
import type { EUserProjectRoles, TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
import type { ProjectRole, User } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/settings/member-columns.tsx

interface RowData extends Pick<TProjectMembership, "original_role" | "created_at"> {
  member: MemberUser;
}
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
type NameProps = {
  rowData: RowData;
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
type NameProps = {
  rowData: IProjectMemberDetails;
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
  setRemoveMemberModal: (rowData: RowData) => void;
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
  setRemoveMemberModal: (rowData: IProjectMemberDetails) => void;
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
type AccountTypeProps = {
  rowData: RowData;
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
type AccountTypeProps = {
  rowData: IProjectMemberDetails;
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
  const roleLabel = ROLE[rowData.original_role ?? EUserPermissions.GUEST];
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
  const roleLabel = ROLE[rowData.role];
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
              value={rowData.original_role}
              onChange={async (value: EUserProjectRoles) => {
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
              value={rowData.role}
              onChange={async (value: ProjectRole) => {
````
````old web/apps/web/core/components/project/settings/member-columns.tsx
                await updateMemberRole(workspaceSlug, projectId, rowData.member.id, value).catch((err) => {
````
````new web/apps/web/core/components/project/settings/member-columns.tsx
                await updateMemberRole(projectId, rowData.member.id, value).catch((err) => {
````

`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`（修改，8 处）：

````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import type { MemberUser } from "@nerve/api-client";
import type { TProjectMembership } from "@nerve/types";
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { useUser, useUserPermissions } from "@/hooks/store/user";
import type { IMemberFilters } from "@/store/member/utils";

export interface RowData extends Pick<TProjectMembership, "original_role" | "created_at"> {
  member: MemberUser;
}
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { useUser, useUserPermissions } from "@/hooks/store/user";
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
import type { IMemberFilters } from "@/store/member/utils";
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
  const [removeMemberModal, setRemoveMemberModal] = useState<RowData | null>(null);
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
  const [removeMemberModal, setRemoveMemberModal] = useState<IProjectMemberDetails | null>(null);
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      ),
      tdRender: (rowData: RowData) => (
        <NameColumn
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      ),
      tdRender: (rowData: IProjectMemberDetails) => (
        <NameColumn
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: RowData) => <div className="w-32">{rowData.member.display_name}</div>,
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: IProjectMemberDetails) => <div className="w-32">{rowData.member.display_name}</div>,
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: RowData) => <div className="w-48 text-secondary">{rowData.member.email}</div>,
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: IProjectMemberDetails) => <div className="w-48 text-secondary">{rowData.member.email}</div>,
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      ),
      tdRender: (rowData: RowData) => (
        <AccountTypeColumn
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      ),
      tdRender: (rowData: IProjectMemberDetails) => (
        <AccountTypeColumn
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: RowData) => <div>{renderFormattedDate(rowData.created_at)}</div>,
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      tdRender: (rowData: IProjectMemberDetails) => <div>{renderFormattedDate(rowData.created_at)}</div>,
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，1 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  useSWR(PROJECT_MEMBERS(projectId, currentProjectRole), () => fetchProjectMembers(workspaceSlug, projectId), {
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  useSWR(PROJECT_MEMBERS(projectId, currentProjectRole), () => fetchProjectMembers(projectId), {
````

`web/apps/web/core/store/issue/issue-details/sub_issues.store.ts`（修改，3 处）：

````old web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
      const issueIds = subIssues.map((issue) => issue.id);
````
````new web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
      const subIssueIds = subIssues.map((issue) => issue.id);
````
````old web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        if (!issues) return issueIds;
        return concat(issues, issueIds);
````
````new web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        if (!issues) return subIssueIds;
        return concat(issues, subIssueIds);
````
````old web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        this.rootIssueDetailStore.rootIssueStore.rootStore.memberRoot.project.fetchProjectMembers(
          workspaceSlug,
          projectId
        );
````
````new web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        this.rootIssueDetailStore.rootIssueStore.rootStore.memberRoot.project.fetchProjectMembers(projectId);
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 408",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 406",
````

`web/packages/types/src/project/projects.ts`（修改，2 处）：

````old web/packages/types/src/project/projects.ts

import type { TUserPermissions } from "../enums";
````
````new web/packages/types/src/project/projects.ts

````
````old web/packages/types/src/project/projects.ts
  GUEST = 5,
}

export type TProjectMembership = {
  member: string;
  role: TUserPermissions | EUserProjectRoles;
} & (
  | {
      id: string;
      original_role: EUserProjectRoles;
      created_at: string;
    }
  | {
      id: null;
      original_role: null;
      created_at: null;
    }
);

export interface IProjectBulkAddFormData {
  members: { role: TUserPermissions | EUserProjectRoles; member_id: string }[];
````
````new web/packages/types/src/project/projects.ts
  GUEST = 5,
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外，没有命中；web 的 oxlint 406 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/dropdowns/member/dropdown.tsx web/apps/web/core/components/project/add-project-members-modal.tsx web/apps/web/core/components/project/member-list-item.tsx web/apps/web/core/components/project/member-list.tsx web/apps/web/core/components/project/settings/member-columns.tsx web/apps/web/core/components/projects/settings/useProjectColumns.tsx web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/services/project/index.ts web/apps/web/core/services/project/project-member.service.ts web/apps/web/core/services/project/project-members.service.ts web/apps/web/core/store/issue/issue-details/sub_issues.store.ts web/apps/web/core/store/member/index.ts web/apps/web/core/store/member/project/project-member-filters.store.ts web/apps/web/core/store/member/project/project-member.store.test.ts web/apps/web/core/store/member/project/project-member.store.ts web/apps/web/core/store/member/utils.ts web/apps/web/core/store/project/project.store.ts web/apps/web/package.json web/packages/types/src/project/projects.ts
```
```bash
git commit -m "feat(M3/P8b): the project's members come from /api/v0, by a store of the session

ProjectMembersService wraps the generated client, and ProjectMemberStore
keeps each project's memberships by its id, with each member's profile
from the workspace's members, and gives none of a project the project
store no longer gives. Additions, role changes and removals go one at a
time and take nerve's answers; what they make of the project, its
member_ids and the caller's own role, the project store shows through
confirmProject. Plane's optimistic updates and the old service go.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t6-ps-sort-request` | 侧边栏的位置是发出的，不是 nerve 回答的 | `project.store.test.ts` | vitest |
| `t6-pm-left` | 调用者离开的项目，它的成员仍给出 | `project-member.store.test.ts` | vitest |
| `t6-pm-add-none` | 加入的成员不存 | `project-member.store.test.ts` | vitest |
| `t6-pm-add-project` | 项目的成员中没有加入的成员 | `project-member.store.test.ts` | vitest |
| `t6-pm-role-request` | 改角色存的是请求，不是 nerve 的回答 | `project-member.store.test.ts` | vitest |
| `t6-pm-own-role` | 调用者自己的角色变化不写到项目上 | `project-member.store.test.ts` | vitest |
| `t6-pm-remove-keeps` | 移出的成员仍在 | `project-member.store.test.ts` | vitest |
| `t6-pm-remove-project` | 项目的成员中仍有移出的成员 | `project-member.store.test.ts` | vitest |
| `t6-pm-unlisted` | store 没有列出的成员也照样发出修改 | `project-member.store.test.ts` | vitest |
| `t6-pm-no-queue` | 成员关系的修改立即发出 | `project-member.store.test.ts` | vitest |
| `t6-pm-change-swallows` | 成员关系的修改被拒绝时给出空值，不失败 | `project-member.store.test.ts` | vitest |
| `t6-pm-fetch-swallows` | 成员列表被拒绝时给出 `undefined`，不失败 | `project-member.store.test.ts` | vitest |
| `t6-pm-fetch-queued` | 成员列表的取数排进修改的队列 | `project-member.store.test.ts` | vitest |
| `t6-pm-guests` | 页面要不含访客的成员时也给出访客 | `project-member.store.test.ts` | vitest |

---

### Task 7: M7 的部分先走：收集箱的分诊状态、工作区级的标签；标签的页面和选择器清零

**Files:**
- Modify: `web/apps/web/core/components/inbox/content/issue-properties.tsx`、`web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx`、`web/apps/web/core/components/issues/issue-detail/label/root.tsx`、`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`、`web/apps/web/core/components/issues/issue-layouts/utils.tsx`、`web/apps/web/core/components/issues/select/base.tsx`、`web/apps/web/core/components/labels/create-update-label-inline.tsx`、`web/apps/web/core/components/labels/delete-label-modal.tsx`、`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`、`web/apps/web/core/components/labels/project-setting-label-item.tsx`、`web/apps/web/core/components/labels/project-setting-label-list.tsx`、`web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx`、`web/apps/web/core/hooks/use-workspace-issue-properties.ts`、`web/apps/web/core/services/issue/issue_label.service.ts`、`web/apps/web/core/services/project/project-state.service.ts`、`web/apps/web/core/store/label.store.ts`、`web/apps/web/core/store/state.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/propel/src/icons/state/helper.tsx`、`web/packages/propel/src/icons/state/index.ts`、`web/packages/types/src/index.ts`、`web/packages/ui/src/dropdowns/combo-box.tsx`
- Delete: `web/apps/web/core/components/dropdowns/intake-state/base.tsx`、`web/apps/web/core/components/dropdowns/intake-state/dropdown.tsx`、`web/packages/propel/src/icons/state/intake-state-group-icon.tsx`、`web/packages/propel/src/icons/state/triage-group-icon.tsx`、`web/packages/types/src/intake/index.ts`、`web/packages/types/src/intake/state.ts`

**Interfaces:**
- Removes（spec 2.7；M3 设计 3.1、3.16、3.17、7.3：分诊状态和工作区跨项目的标签是 M7 的，契约中没有它们）：
  - 收集箱的分诊状态：`IntakeStateDropdown`（`dropdowns/intake-state/` 删除）、`StateStore` 的 `intakeStateMap`、`fetchedIntakeMap`、`getIntakeStateById`、`getProjectIntakeState`、`getProjectIntakeStateIds`、`fetchProjectIntakeState`，旧状态 service 的 `getIntakeState`，`packages/types/src/intake/`，`@nerve/propel` 的 `IntakeStateGroupIcon`、`TriageGroupIcon`、`IIntakeStateGroupIcon`、`INTAKE_STATE_GROUP_COLORS`。收集箱的工作项：已接受的显示状态下拉框，未接受的不显示状态；创建弹窗没有状态一项（W17）。
  - 工作区级的标签：`LabelStore` 的 `workspaceLabels`、`getWorkspaceLabels`、`getWorkspaceLabelIds`、`fetchWorkspaceLabels`，旧标签 service 的 `getWorkspaceIssueLabels`（`/api/workspaces/{slug}/labels/`），键 `WORKSPACE_LABELS`；`useWorkspaceIssueProperties` 不再取它（整个文件）；工作区级的工作项筛选没有标签一项，按标签分组只有"无"（W17，M7 加回）。
- 清零（7.9，R3）：标签的设置页和工作项的标签选择器有 Task 9 的手改，它们的 oxlint 警告在这里先清零，Task 9 的块因此只有换 store 的改动：`then` 的回调不再返回值、遮蔽的变量改名、点击的 `p`/`li` 改为 `button`、只收集按键的根加 `role="presentation"`（`@nerve/ui` 的 `ComboBox` 加 `role` 这个 prop）、`useCallback` 的依赖补齐、`createLabel` 的回调不再收 slug 和项目（调用方已有它们）。行为不变。
- web 的 oxlint 上限 406 → 370。

**Tests:** 没有新的 vitest：本 Task 只删除和清零。

- [ ] **Step 1: 分诊状态和工作区级的标签**

`web/apps/web/core/components/dropdowns/intake-state/base.tsx`（删除）：

````delete web/apps/web/core/components/dropdowns/intake-state/base.tsx
````

`web/apps/web/core/components/dropdowns/intake-state/dropdown.tsx`（删除）：

````delete web/apps/web/core/components/dropdowns/intake-state/dropdown.tsx
````

`web/apps/web/core/components/inbox/content/issue-properties.tsx`（修改，3 处）：

````old web/apps/web/core/components/inbox/content/issue-properties.tsx
import { DateDropdown } from "@/components/dropdowns/date";
import { IntakeStateDropdown } from "@/components/dropdowns/intake-state/dropdown";
````
````new web/apps/web/core/components/inbox/content/issue-properties.tsx
import { DateDropdown } from "@/components/dropdowns/date";
````
````old web/apps/web/core/components/inbox/content/issue-properties.tsx
  });
  const DropdownComponent = isIntakeAccepted ? StateDropdown : IntakeStateDropdown;
````
````new web/apps/web/core/components/inbox/content/issue-properties.tsx
  });
````
````old web/apps/web/core/components/inbox/content/issue-properties.tsx
              {issue?.state_id && (
                <DropdownComponent
````
````new web/apps/web/core/components/inbox/content/issue-properties.tsx
              {issue?.state_id && isIntakeAccepted && (
                <StateDropdown
````

`web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx`（修改，2 处）：

````old web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx
import { DateDropdown } from "@/components/dropdowns/date";
import { IntakeStateDropdown } from "@/components/dropdowns/intake-state/dropdown";
````
````new web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx
import { DateDropdown } from "@/components/dropdowns/date";
````
````old web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx
    <div className="relative flex flex-wrap items-center gap-2">
      {/* intake state */}
      <div className="h-7">
        <IntakeStateDropdown
          value={data?.state_id}
          onChange={(stateId) => handleData("state_id", stateId)}
          projectId={projectId}
          buttonVariant="border-with-text"
          tabIndex={getIndex("state_id")}
        />
      </div>

````
````new web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx
    <div className="relative flex flex-wrap items-center gap-2">
````

`web/apps/web/core/components/issues/issue-layouts/utils.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/issue-layouts/utils.tsx
  const { workspaceLabels, projectLabels } = rootStore.label;
  // map labels to group by columns
````
````new web/apps/web/core/components/issues/issue-layouts/utils.tsx
  // the workspace's labels across its projects are M7's (M3 design 7.3): a workspace level groups none of them
````
````old web/apps/web/core/components/issues/issue-layouts/utils.tsx
    ...(isWorkspaceLevel ? workspaceLabels || [] : projectLabels || []),
````
````new web/apps/web/core/components/issues/issue-layouts/utils.tsx
    ...(isWorkspaceLevel ? [] : (rootStore.label.projectLabels ?? [])),
````

`web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx`（修改，5 处）：

````old web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
import { useGlobalView } from "@/hooks/store/use-global-view";
import { useLabel } from "@/hooks/store/use-label";
````
````new web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
import { useGlobalView } from "@/hooks/store/use-global-view";
````
````old web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
  } = useMember();
  const { getWorkspaceLabelIds } = useLabel();
````
````new web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
  } = useMember();
````
````old web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
        .then(() => {
````
````new web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
        .then(() =>
````
````old web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
            message: "Your view has been updated successfully.",
          });
        })
````
````new web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
            message: "Your view has been updated successfully.",
          })
        )
````
````old web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
        memberIds={getWorkspaceMemberIds(workspaceSlug)}
        labelIds={getWorkspaceLabelIds(workspaceSlug)}
````
````new web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx
        memberIds={getWorkspaceMemberIds(workspaceSlug)}
````

`web/apps/web/core/hooks/use-workspace-issue-properties.ts`（整个文件，32 行）：

````whole web/apps/web/core/hooks/use-workspace-issue-properties.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
// nerve imports
import { WORKSPACE_CYCLES, WORKSPACE_MODULES } from "@nerve/constants";
// local imports
import { useCycle } from "./store/use-cycle";
import { useModule } from "./store/use-module";

export const useWorkspaceIssueProperties = (workspaceSlug: string | undefined) => {
  const { fetchWorkspaceModules } = useModule();

  const { fetchWorkspaceCycles } = useCycle();

  // fetch workspace Modules
  useSWR(
    workspaceSlug ? WORKSPACE_MODULES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceModules(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetch workspace Cycles
  useSWR(
    workspaceSlug ? WORKSPACE_CYCLES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceCycles(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
};
````

`web/apps/web/core/services/issue/issue_label.service.ts`（修改，1 处）：

````old web/apps/web/core/services/issue/issue_label.service.ts
export class IssueLabelService extends APIService {
  async getWorkspaceIssueLabels(workspaceSlug: string): Promise<IIssueLabel[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/labels/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

````
````new web/apps/web/core/services/issue/issue_label.service.ts
export class IssueLabelService extends APIService {
````

`web/apps/web/core/services/project/project-state.service.ts`（修改，2 处）：

````old web/apps/web/core/services/project/project-state.service.ts
import type { IIntakeState, IState } from "@nerve/types";
````
````new web/apps/web/core/services/project/project-state.service.ts
import type { IState } from "@nerve/types";
````
````old web/apps/web/core/services/project/project-state.service.ts
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getIntakeState(workspaceSlug: string, projectId: string): Promise<IIntakeState> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/intake-state/`)
````
````new web/apps/web/core/services/project/project-state.service.ts
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/`)
````

`web/apps/web/core/store/label.store.ts`（修改，8 处）：

````old web/apps/web/core/store/label.store.ts
  projectLabelsTree: IIssueLabelTree[] | undefined;
  workspaceLabels: IIssueLabel[] | undefined;
  //computed actions
  getWorkspaceLabels: (workspaceSlug: string) => IIssueLabel[] | undefined;
  getWorkspaceLabelIds: (workspaceSlug: string) => string[] | undefined;
````
````new web/apps/web/core/store/label.store.ts
  projectLabelsTree: IIssueLabelTree[] | undefined;
  //computed actions
````
````old web/apps/web/core/store/label.store.ts
  // fetch actions
  fetchWorkspaceLabels: (workspaceSlug: string) => Promise<IIssueLabel[]>;
````
````new web/apps/web/core/store/label.store.ts
  // fetch actions
````
````old web/apps/web/core/store/label.store.ts
  /**
   * Returns the labelMap belongs to a specific workspace
   */
  get workspaceLabels() {
    const currentWorkspaceDetails = this.rootStore.workspaceRoot.currentWorkspace;
    if (!currentWorkspaceDetails) return;
    return this.getWorkspaceLabels(currentWorkspaceDetails.slug);
  }

  /**
````
````new web/apps/web/core/store/label.store.ts
  /**
````
````old web/apps/web/core/store/label.store.ts
    const projectId = this.rootStore.router.projectId;
    const workspaceSlug = this.rootStore.router.workspaceSlug || "";
    if (!projectId || !(this.fetchedMap[projectId] || this.fetchedMap[workspaceSlug])) return;
````
````new web/apps/web/core/store/label.store.ts
    const projectId = this.rootStore.router.projectId;
    if (!projectId || !this.fetchedMap[projectId]) return;
````
````old web/apps/web/core/store/label.store.ts

  getWorkspaceLabels = computedFn((workspaceSlug: string) => {
    const workspaceDetails = this.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
    if (!workspaceDetails || !this.fetchedMap[workspaceSlug]) return;
    return sortBy(
      Object.values(this.labelMap).filter((label) => label.workspace_id === workspaceDetails.id),
      "sort_order"
    );
  });

  getWorkspaceLabelIds = computedFn(
    (workspaceSlug: string) => this.getWorkspaceLabels(workspaceSlug)?.map((label) => label.id) ?? undefined
  );

  getProjectLabels = computedFn((projectId: string | undefined | null) => {
    const workspaceSlug = this.rootStore.router.workspaceSlug || "";
    if (!projectId || !(this.fetchedMap[projectId] || this.fetchedMap[workspaceSlug])) return;
````
````new web/apps/web/core/store/label.store.ts

  getProjectLabels = computedFn((projectId: string | undefined | null) => {
    if (!projectId || !this.fetchedMap[projectId]) return;
````
````old web/apps/web/core/store/label.store.ts
    const workspaceSlug = this.rootStore.router.workspaceSlug;
    if (!workspaceSlug || !projectId || !(this.fetchedMap[projectId] || this.fetchedMap[workspaceSlug]))
      return undefined;
````
````new web/apps/web/core/store/label.store.ts
    if (!projectId || !this.fetchedMap[projectId]) return undefined;
````
````old web/apps/web/core/store/label.store.ts
        set(this.fetchedMap, projectId, true);
      });
      return response;
    });

  /**
   * Fetches all the labelMap belongs to a specific project
   * @param workspaceSlug
   * @param projectId
   * @returns Promise<IIssueLabel[]>
   */
  fetchWorkspaceLabels = async (workspaceSlug: string) =>
    await this.issueLabelService.getWorkspaceIssueLabels(workspaceSlug).then((response) => {
      runInAction(() => {
        response.forEach((label) => {
          set(this.labelMap, [label.id], label);
        });
        set(this.fetchedMap, workspaceSlug, true);
````
````new web/apps/web/core/store/label.store.ts
        set(this.fetchedMap, projectId, true);
````
````old web/apps/web/core/store/label.store.ts
    await this.issueLabelService.deleteIssueLabel(workspaceSlug, projectId, labelId).then(() => {
      runInAction(() => {
        delete this.labelMap[labelId];
      });
````
````new web/apps/web/core/store/label.store.ts
    await this.issueLabelService.deleteIssueLabel(workspaceSlug, projectId, labelId);
    runInAction(() => {
      delete this.labelMap[labelId];
````

`web/apps/web/core/store/state.store.ts`（修改，13 处）：

````old web/apps/web/core/store/state.store.ts
import type { IIntakeState, IState } from "@nerve/types";
````
````new web/apps/web/core/store/state.store.ts
import type { IState } from "@nerve/types";
````
````old web/apps/web/core/store/state.store.ts
  fetchedMap: Record<string, boolean>;
  fetchedIntakeMap: Record<string, boolean>;
````
````new web/apps/web/core/store/state.store.ts
  fetchedMap: Record<string, boolean>;
````
````old web/apps/web/core/store/state.store.ts
  stateMap: Record<string, IState>;
  intakeStateMap: Record<string, IIntakeState>;
````
````new web/apps/web/core/store/state.store.ts
  stateMap: Record<string, IState>;
````
````old web/apps/web/core/store/state.store.ts
  getStateById: (stateId: string | null | undefined) => IState | undefined;
  getIntakeStateById: (intakeStateId: string | null | undefined) => IIntakeState | undefined;
  getProjectStates: (projectId: string | null | undefined) => IState[] | undefined;
  getProjectIntakeState: (projectId: string | null | undefined) => IIntakeState | undefined;
  getProjectStateIds: (projectId: string | null | undefined) => string[] | undefined;
  getProjectIntakeStateIds: (projectId: string | null | undefined) => string[] | undefined;
````
````new web/apps/web/core/store/state.store.ts
  getStateById: (stateId: string | null | undefined) => IState | undefined;
  getProjectStates: (projectId: string | null | undefined) => IState[] | undefined;
  getProjectStateIds: (projectId: string | null | undefined) => string[] | undefined;
````
````old web/apps/web/core/store/state.store.ts
  fetchProjectStates: (workspaceSlug: string, projectId: string) => Promise<IState[]>;
  fetchProjectIntakeState: (workspaceSlug: string, projectId: string) => Promise<IIntakeState>;
````
````new web/apps/web/core/store/state.store.ts
  fetchProjectStates: (workspaceSlug: string, projectId: string) => Promise<IState[]>;
````
````old web/apps/web/core/store/state.store.ts
  stateMap: Record<string, IState> = {};
  intakeStateMap: Record<string, IIntakeState> = {};
````
````new web/apps/web/core/store/state.store.ts
  stateMap: Record<string, IState> = {};
````
````old web/apps/web/core/store/state.store.ts
  fetchedMap: Record<string, boolean> = {};
  fetchedIntakeMap: Record<string, boolean> = {};
````
````new web/apps/web/core/store/state.store.ts
  fetchedMap: Record<string, boolean> = {};
````
````old web/apps/web/core/store/state.store.ts
      stateMap: observable,
      intakeStateMap: observable,
      fetchedMap: observable,
      fetchedIntakeMap: observable,
````
````new web/apps/web/core/store/state.store.ts
      stateMap: observable,
      fetchedMap: observable,
````
````old web/apps/web/core/store/state.store.ts
      fetchProjectStates: action,
      fetchProjectIntakeState: action,
````
````new web/apps/web/core/store/state.store.ts
      fetchProjectStates: action,
````
````old web/apps/web/core/store/state.store.ts
  /**
   * @description returns intake state details using intake state id
   * @param intakeStateId
   */
  getIntakeStateById = computedFn((intakeStateId: string | null | undefined) => {
    if (!this.intakeStateMap || !intakeStateId) return;
    return this.intakeStateMap[intakeStateId] ?? undefined;
  });

  /**
````
````new web/apps/web/core/store/state.store.ts
  /**
````
````old web/apps/web/core/store/state.store.ts
    return sortStates(Object.values(this.stateMap).filter((state) => state.project_id === projectId));
  });

  /**
   * Returns the intake state for a project by projectId
   * @param projectId
   * @returns IIntakeState | undefined
   */
  getProjectIntakeState = computedFn((projectId: string | null | undefined) => {
    if (!projectId || !this.fetchedIntakeMap[projectId]) return;
    return Object.values(this.intakeStateMap).find((state) => state.project_id === projectId);
````
````new web/apps/web/core/store/state.store.ts
    return sortStates(Object.values(this.stateMap).filter((state) => state.project_id === projectId));
````
````old web/apps/web/core/store/state.store.ts
    return projectStates?.map((state) => state.id) ?? [];
  });

  /**
   * Returns the intake state ids for a project by projectId
   * @param projectId
   * @returns string[]
   */
  getProjectIntakeStateIds = computedFn((projectId: string | null | undefined) => {
    const workspaceSlug = this.router.workspaceSlug;
    if (!workspaceSlug || !projectId || !this.fetchedIntakeMap[projectId]) return undefined;
    const projectIntakeState = this.getProjectIntakeState(projectId);
    return projectIntakeState?.id ? [projectIntakeState.id] : [];
````
````new web/apps/web/core/store/state.store.ts
    return projectStates?.map((state) => state.id) ?? [];
````
````old web/apps/web/core/store/state.store.ts
    return statesResponse;
  };

  /**
   * fetches the intakeStateMap of a project
   * @param workspaceSlug
   * @param projectId
   * @returns
   */
  fetchProjectIntakeState = async (workspaceSlug: string, projectId: string) => {
    const intakeStateResponse = await this.stateService.getIntakeState(workspaceSlug, projectId);
    runInAction(() => {
      set(this.intakeStateMap, [intakeStateResponse.id], intakeStateResponse);
      set(this.fetchedIntakeMap, projectId, true);
    });
    return intakeStateResponse;
````
````new web/apps/web/core/store/state.store.ts
    return statesResponse;
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_LABELS = (workspaceSlug: string) => `WORKSPACE_LABELS_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/propel/src/icons/state/helper.tsx`（修改，3 处）：

````old web/packages/propel/src/icons/state/helper.tsx
import { EIconSize } from "@nerve/constants";
import type { TIntakeStateGroups } from "@nerve/types";
````
````new web/packages/propel/src/icons/state/helper.tsx
import { EIconSize } from "@nerve/constants";
````
````old web/packages/propel/src/icons/state/helper.tsx
  stateGroup: TStateGroups;
  size?: EIconSize;
  percentage?: number;
}

export interface IIntakeStateGroupIcon {
  className?: string;
  color?: string;
  stateGroup: TIntakeStateGroups;
````
````new web/packages/propel/src/icons/state/helper.tsx
  stateGroup: TStateGroups;
````
````old web/packages/propel/src/icons/state/helper.tsx

export const INTAKE_STATE_GROUP_COLORS: { [key in TIntakeStateGroups]: string } = { triage: "#4E5355" };

````
````new web/packages/propel/src/icons/state/helper.tsx

````

`web/packages/propel/src/icons/state/index.ts`（修改，1 处）：

````old web/packages/propel/src/icons/state/index.ts
export * from "./unstarted-group-icon";
export * from "./intake-state-group-icon";
````
````new web/packages/propel/src/icons/state/index.ts
export * from "./unstarted-group-icon";
````

`web/packages/propel/src/icons/state/intake-state-group-icon.tsx`（删除）：

````delete web/packages/propel/src/icons/state/intake-state-group-icon.tsx
````

`web/packages/propel/src/icons/state/triage-group-icon.tsx`（删除）：

````delete web/packages/propel/src/icons/state/triage-group-icon.tsx
````

`web/packages/types/src/index.ts`（修改，1 处）：

````old web/packages/types/src/index.ts
export * from "./project";
export * from "./intake";
````
````new web/packages/types/src/index.ts
export * from "./project";
````

`web/packages/types/src/intake/index.ts`（删除）：

````delete web/packages/types/src/intake/index.ts
````

`web/packages/types/src/intake/state.ts`（删除）：

````delete web/packages/types/src/intake/state.ts
````

- [ ] **Step 2: 标签的页面和选择器清零**

`web/apps/web/core/components/issues/issue-detail/label/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
  createLabel: (workspaceSlug: string, projectId: string, data: Partial<IIssueLabel>) => Promise<any>;
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
  createLabel: (data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
      updateIssue: async (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => {
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
      updateIssue: async (slug: string, issueProjectId: string, workItemId: string, data: Partial<TIssue>) => {
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
          else await updateIssue(workspaceSlug, projectId, issueId, data);
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
          else await updateIssue(slug, issueProjectId, workItemId, data);
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
      createLabel: async (workspaceSlug: string, projectId: string, data: Partial<IIssueLabel>) => {
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
      createLabel: async (data: Partial<IIssueLabel>) => {
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
    [updateIssue, createLabel, onLabelUpdate]
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
    [updateIssue, createLabel, onLabelUpdate, isInboxIssue, workspaceSlug, projectId, t]
````

`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`（修改，4 处）：

````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
  onAddLabel: (workspaceSlug: string, projectId: string, data: Partial<IIssueLabel>) => Promise<any>;
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
  onAddLabel: (data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
````
````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
    const label = await onAddLabel(workspaceSlug, projectId, { name: labelName, color: getRandomLabelColor() });
    onSelect([...values, label.id]);
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
    const created = await onAddLabel({ name: labelName, color: getRandomLabelColor() });
    onSelect([...values, created.id]);
````
````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
                <ul className="space-y-1">
                  <Combobox.Option
                    as="li"
                    value={query}
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
                <div className="space-y-1">
                  <button
                    type="button"
````
````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
                  </Combobox.Option>
                </ul>
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
                  </button>
                </div>
````

`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`（修改，9 处）：

````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
}

export function LabelDropdown(props: ILabelDropdownProps) {
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
}

const preventPropagation = (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
  e.stopPropagation();
  e.preventDefault();
};

export function LabelDropdown(props: ILabelDropdownProps) {
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    fullHeight = false,
    label,
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    fullHeight = false,
    label: buttonLabel,
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
        {label}
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
        {buttonLabel}
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
      label,
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
      buttonLabel,
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx

  const preventPropagation = (e: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
    e.stopPropagation();
    e.preventDefault();
  };

````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx

````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    <div className={`${fullHeight ? "h-full" : "h-5"}`} onClick={preventPropagation}>
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    <div className={`${fullHeight ? "h-full" : "h-5"}`} onClick={preventPropagation} role="presentation">
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
        as="div"
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
        as="div"
        role="presentation"
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
                ) : canCreateLabel ? (
                  <p
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
                ) : canCreateLabel ? (
                  <button
                    type="button"
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
                  </p>
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
                  </button>
````

`web/apps/web/core/components/issues/select/base.tsx`（修改，5 处）：

````old web/apps/web/core/components/issues/select/base.tsx
    label,
````
````new web/apps/web/core/components/issues/select/base.tsx
    label: buttonLabel,
````
````old web/apps/web/core/components/issues/select/base.tsx
      as="div"
````
````new web/apps/web/core/components/issues/select/base.tsx
      as="div"
      role="presentation"
````
````old web/apps/web/core/components/issues/select/base.tsx
        {label ? (
          label
````
````new web/apps/web/core/components/issues/select/base.tsx
        {buttonLabel ? (
          buttonLabel
````
````old web/apps/web/core/components/issues/select/base.tsx
                ) : createLabelEnabled ? (
                  <p
````
````new web/apps/web/core/components/issues/select/base.tsx
                ) : createLabelEnabled ? (
                  <button
                    type="button"
````
````old web/apps/web/core/components/issues/select/base.tsx
                  </p>
````
````new web/apps/web/core/components/issues/select/base.tsx
                  </button>
````

`web/apps/web/core/components/labels/create-update-label-inline.tsx`（修改，6 处）：

````old web/apps/web/core/components/labels/create-update-label-inline.tsx
        .then((_res) => {
          handleClose();
          reset(defaultValues);
        })
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
        .then(() => handleClose())
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
        .then((_res) => {
          reset(defaultValues);
          handleClose();
        })
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
        .then(() => handleClose())
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
                            onChange={(value) => onChange(value.hex)}
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
                            onChange={(picked) => onChange(picked.hex)}
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
              render={({ field: { value, onChange, ref } }) => (
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
              render={({ field: { value, onChange, ref: inputRef } }) => (
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
                      type="text"
                      autoFocus
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
                      type="text"
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
                      ref={ref}
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
                      ref={inputRef}
````

`web/apps/web/core/components/labels/delete-label-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/labels/delete-label-modal.tsx
      .then(() => {
        handleClose();
      })
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
      .then(() => handleClose())
````

`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`（修改，10 处）：

````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    const element = labelRef.current;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    const labelElement = labelRef.current;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    if (!element || !isEditable) return;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    if (!labelElement || !isEditable) return;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
      draggable({
        element,
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
      draggable({
        element: labelElement,
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
      dropTargetForElements({
        element,
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
      dropTargetForElements({
        element: labelElement,
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const instruction = getInstructionFromPayload(self, source, location);
          setInstruction(instruction);
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          setInstruction(getInstructionFromPayload(self, source, location));
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const instruction = getInstructionFromPayload(dropTarget, source, location);
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const dropInstruction = getInstructionFromPayload(dropTarget, source, location);
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          parentId = instruction === "make-child" ? dropTargetData.id : dropTargetData.parentId;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          parentId = dropInstruction === "make-child" ? dropTargetData.id : dropTargetData.parentId;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const droppedLabelId = instruction !== "make-child" ? dropTargetData.id : undefined;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const droppedLabelId = dropInstruction !== "make-child" ? dropTargetData.id : undefined;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          if (instruction === "reorder-below") dropAtEndOfList = true;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          if (dropInstruction === "reorder-below") dropAtEndOfList = true;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
  }, [labelRef?.current, dragHandleRef?.current, label, isChild, isGroup, isLastChild, onDrop]);
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
  }, [label, isChild, isGroup, isLastChild, onDrop, isEditable]);
````

`web/apps/web/core/components/labels/project-setting-label-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/labels/project-setting-label-item.tsx
  const removeFromGroup = (label: IIssueLabel) => {
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
  const removeFromGroup = () => {
````

`web/apps/web/core/components/labels/project-setting-label-list.tsx`（修改，2 处）：

````old web/apps/web/core/components/labels/project-setting-label-list.tsx
                    handleLabelDelete={(label: IIssueLabel) => setSelectDeleteLabel(label)}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
                    handleLabelDelete={setSelectDeleteLabel}
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
                  handleLabelDelete={(label) => setSelectDeleteLabel(label)}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
                  handleLabelDelete={setSelectDeleteLabel}
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 406",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 370",
````

`web/packages/ui/src/dropdowns/combo-box.tsx`（修改，2 处）：

````old web/packages/ui/src/dropdowns/combo-box.tsx
import type { ElementType, KeyboardEventHandler, ReactNode, Ref } from "react";
````
````new web/packages/ui/src/dropdowns/combo-box.tsx
import type { AriaRole, ElementType, KeyboardEventHandler, ReactNode, Ref } from "react";
````
````old web/packages/ui/src/dropdowns/combo-box.tsx
  onKeyDown?: KeyboardEventHandler<HTMLDivElement> | undefined;
````
````new web/packages/ui/src/dropdowns/combo-box.tsx
  onKeyDown?: KeyboardEventHandler<HTMLDivElement> | undefined;
  /** The root's role: "presentation" for a root that only gathers its parts' keys (onKeyDown). */
  role?: AriaRole | undefined;
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外，没有命中；web 的 oxlint 370 条，等于新的上限；`propel` 16 条、`ui` 19 条，不变。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add web/apps/web/core/components/dropdowns/intake-state/base.tsx web/apps/web/core/components/dropdowns/intake-state/dropdown.tsx web/apps/web/core/components/inbox/content/issue-properties.tsx web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx web/apps/web/core/components/issues/issue-detail/label/root.tsx web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx web/apps/web/core/components/issues/issue-layouts/utils.tsx web/apps/web/core/components/issues/select/base.tsx web/apps/web/core/components/labels/create-update-label-inline.tsx web/apps/web/core/components/labels/delete-label-modal.tsx web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx web/apps/web/core/components/labels/project-setting-label-item.tsx web/apps/web/core/components/labels/project-setting-label-list.tsx web/apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx web/apps/web/core/hooks/use-workspace-issue-properties.ts web/apps/web/core/services/issue/issue_label.service.ts web/apps/web/core/services/project/project-state.service.ts web/apps/web/core/store/label.store.ts web/apps/web/core/store/state.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/propel/src/icons/state/helper.tsx web/packages/propel/src/icons/state/index.ts web/packages/propel/src/icons/state/intake-state-group-icon.tsx web/packages/propel/src/icons/state/triage-group-icon.tsx web/packages/types/src/index.ts web/packages/types/src/intake/index.ts web/packages/types/src/intake/state.ts web/packages/ui/src/dropdowns/combo-box.tsx
```
```bash
git commit -m "refactor(M3/P8b): M7's intake triage state and workspace labels leave first

nerve has no intake triage state and no workspace-wide labels: they
are M7's. The intake state dropdown, its icons and types, the state
store's intake half, the label store's workspace half and their old
service methods go; intake items show a state once accepted, and the
workspace views have no label filter or grouping until M7. The label
pages and pickers that Task 9 rewires end without oxlint warnings
first, without changing what they do.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有变异：它只删除和清零，删除由 `tsc`、knip 核对，清零由 oxlint 的上限核对（spec 附录 A.7、A.8）。

---

### Task 8: 状态的 service 和 store：顺序和组中的位置；工作区包装层取工作区的状态

**Files:**
- Create: `web/apps/web/core/services/project/states.service.ts`、`web/apps/web/core/store/state.store.test.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/dropdowns/state/base.tsx`、`web/apps/web/core/components/dropdowns/state/dropdown.tsx`、`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx`、`web/apps/web/core/components/issues/issue-layouts/utils.tsx`、`web/apps/web/core/components/project-states/create-update/create.tsx`、`web/apps/web/core/components/project-states/create-update/form.tsx`、`web/apps/web/core/components/project-states/create-update/update.tsx`、`web/apps/web/core/components/project-states/options/delete.tsx`、`web/apps/web/core/components/project-states/root.tsx`、`web/apps/web/core/components/project-states/state-item.tsx`、`web/apps/web/core/hooks/store/fake-store-hooks.ts`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/lib/reconciled.ts`、`web/apps/web/core/services/project/index.ts`、`web/apps/web/core/store/issue/issue-details/issue.store.ts`、`web/apps/web/core/store/issue/issue-details/sub_issues.store.ts`、`web/apps/web/core/store/root.store.ts`、`web/apps/web/core/store/state.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/propel/src/icons/state/helper.tsx`、`web/packages/types/src/state.ts`、`web/packages/utils/package.json`、`web/packages/utils/src/work-item/state.ts`
- Delete: `web/apps/web/core/services/project/project-state.service.ts`
- 机械步骤（Step 1）改到：`web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx`、`web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx`、`web/apps/web/core/components/issues/preview-card/date.tsx`、`web/apps/web/core/components/issues/preview-card/root.tsx`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx`、`web/apps/web/core/components/project-states/group-item.tsx`、`web/apps/web/core/components/project-states/group-list.tsx`、`web/apps/web/core/components/project-states/state-item-title.tsx`、`web/apps/web/core/components/project-states/state-list.tsx`、`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/store/issue/root.store.ts`、`web/packages/constants/src/state.ts`、`web/packages/types/src/issues/issue.ts`、`web/packages/types/src/project/projects.ts`、`web/packages/utils/src/distribution-update.ts`、`web/packages/utils/src/progress.test.ts`、`web/packages/utils/src/work-item-filters/configs/filters/state.ts`、`web/packages/utils/src/work-item/base.ts`、`web/packages/utils/src/work-item/modal.ts`

**Interfaces:**
- Produces（spec 2.8；M3 设计 3.17、7.1、7.2、7.3；P7a 的交接）：
  - 生成的 `State`、`StateGroup` 取代 `IState`、`TStateGroups`（Step 1 的机械步骤，28 个文件；`@nerve/propel` 的状态图标收 `StateGroup`）；`packages/types/src/state.ts` 只留页面的回调类型 `TStateOperationsCallbacks`（整个文件，修改用生成的 `StateCreate`、`StateUpdate`）。
  - `core/services/project/states.service.ts`：`class StatesService { list(projectId); listInWorkspace(slug); create(projectId, data: StateCreate); update(stateId, data: StateUpdate); delete(stateId); markDefault(stateId) }`。旧的 `project-state.service.ts` 删除。
  - `core/store/state.store.ts`（整个文件）：`StateStore(rootStore, api)`；每个项目的状态按项目的 id，每个工作区的（调用者是成员的项目的状态）按工作区的 id，两者都是 `ReconciledByKey`；项目 store 不再给出的项目的状态什么都不给。顺序是 `sortStates`（组在 `STATE_GROUPS` 中的顺序，再按 `sequence`）；nerve 不给 `order`，组中的位置由顺序算出：`getStatePercentageInGroup(stateId)`（组中最后一个是 100）。`stateMap`、`workspaceStates`、`projectStates`、`groupedProjectStates`（每个组都在）、`getStateById`、`getProjectStates`（先给项目自己的列表，没有时给工作区列表中这个项目的）、`getProjectStateIds`；`fetchProjectStates(projectId)`、`fetchWorkspaceStates(workspace)`；修改经 `changes = oneAtATime()`，写入回答，在项目和工作区两份列表中都改：`createState(projectId, data)`、`updateState(stateId, data)`（移动就是改 `group`、`sequence`）、`deleteState(stateId)`、`markStateAsDefault(stateId)`（原来的默认不再是默认）；store 没有这个状态时删除和设默认不问 nerve、直接失败。
  - `ReconciledByKey.values()`：已取的每个键的值（`stateMap` 在两份列表中找状态；扩展 `reconciled.ts`，不复制）。
  - `sortStates`（`@nerve/utils`）用 `toSorted`、不改给它的数组，总是交回数组；它的 oxlint 警告随之清零（`utils` 的上限 10 → 9）。
  - 页面：状态设置页的根（`project-states/root.tsx`，整个文件）经 `useSessionSWR(["PROJECT_STATES", projectId])` 取，回调直接是 store 的方法；拖动排序发出 `updateState(stateId, { group, sequence })`，在回答之后显示（W17，P7a）；表单的值是 `StateCreate` 去掉 `group`。状态图标的百分比改读 `getStatePercentageInGroup`（下拉框、筛选、分组的表头；`issues/issue-layouts/utils.tsx` 多 4 行，spec 附录 A.10）。状态下拉框（整个文件）打开时仍按需取（M4，spec 第 5 节）。
  - `useWorkspaceFetch`：列表中有这个工作区之后再取 `["WORKSPACE_STATES", id, slug]`（`fetchWorkspaceStates`）；工作区包装层原来按 slug 取状态的 `useSWR` 和键 `WORKSPACE_STATES` 删除，包装层从此不从 `swr` 导入值，加进 `no-restricted-imports` 的范围（P8a review 第 6 节）。项目包装层取状态改调新签名（过渡：仍是原来的 `useSWR`，Task 10 换掉）。
  - `.oxlintrc.json`：`state.store.ts`、`components/project-states/**`、`workspace-wrapper.tsx` 加进 `no-restricted-imports` 的范围；`states.service.ts`、`state.store.ts` 和它的测试加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest）`core/store/state.store.test.ts`：
- `StateStore, the states`：`keeps a project's states by group, then sequence, and each one's place in its group`（nerve 列出的次序打乱；完成组的 `sequence` 小于进行中的两个，只按 `sequence` 排就不对）；`gives a project's states from its workspace's list until its own is fetched, and none of a project left`；`fails when nerve refuses a project's or the workspace's, keeping none, and again, keeping the states it had`；`keeps the states it had, gives nothing and does not fail, when the session changes as it fetches again`。
- `StateStore, the changes`：`adds a created state as nerve answers it, last of its project's`；`moves a state to the place nerve answers, in its project's list and in its workspace's`；`deletes a state, and makes another its project's default, the one that was no longer`；`fails, changing nothing, when nerve refuses $change`（4 行）；`fails, asking nerve nothing, for $change of a state it does not have`（2 行）；`sends each change once nerve has answered the one before it, refused or not: two moves in a row among them`；`fetches the states while a change is out: a fetch does not wait for it`。
- `StateStore, while a fetch is out`：`shows once a state created during a refetch whose list has it, and the changes confirmed meanwhile`；`lets each project's newer fetch write: an older one answering last writes nothing`。
- `use-workspace-fetch.test.ts`：取数的测试加上工作区的状态。

- [ ] **Step 1: 机械步骤：Plane 的状态类型换成生成的 `State`、`StateGroup`**

`$P8BTMP/rename_types.py` 见"一次性脚本"。28 个文件；其中 9 个在本 Task 另有手改（它们的块写在这一步之后的文件上），其余 19 个只经这一步改到。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . IState State web/apps/web/core/components/project-states/group-list.tsx web/apps/web/core/components/project-states/state-item.tsx web/apps/web/core/components/project-states/state-item-title.tsx web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/options/delete.tsx web/apps/web/core/components/project-states/root.tsx web/apps/web/core/components/project-states/create-update/create.tsx web/apps/web/core/components/project-states/state-list.tsx web/apps/web/core/components/project-states/create-update/form.tsx web/apps/web/core/components/project-states/create-update/update.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx web/apps/web/core/components/dropdowns/state/base.tsx web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/packages/utils/src/progress.test.ts web/packages/utils/src/distribution-update.ts web/packages/utils/src/work-item-filters/configs/filters/state.ts web/packages/utils/src/work-item/state.ts web/packages/utils/src/work-item/modal.ts`
Expected: `IState -> State in 21 files`。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . TStateGroups StateGroup web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx web/apps/web/core/components/project-states/group-list.tsx web/apps/web/core/components/project-states/state-item.tsx web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/create-update/create.tsx web/apps/web/core/components/project-states/state-list.tsx web/apps/web/core/components/issues/preview-card/date.tsx web/apps/web/core/components/issues/preview-card/root.tsx web/packages/constants/src/state.ts web/packages/utils/src/work-item-filters/configs/filters/state.ts web/packages/utils/src/work-item/base.ts`
Expected: `TStateGroups -> StateGroup in 11 files`。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . TStateGroups StateGroup --from ../state web/packages/types/src/project/projects.ts web/packages/types/src/issues/issue.ts`
Expected: `TStateGroups -> StateGroup in 2 files`。

`git diff --numstat` 必须恰好是：

```text
2	2	web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx
2	2	web/apps/web/core/components/dropdowns/state/base.tsx
3	2	web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx
2	2	web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
2	2	web/apps/web/core/components/issues/preview-card/date.tsx
3	2	web/apps/web/core/components/issues/preview-card/root.tsx
2	2	web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx
4	3	web/apps/web/core/components/project-states/create-update/create.tsx
7	7	web/apps/web/core/components/project-states/create-update/form.tsx
4	3	web/apps/web/core/components/project-states/create-update/update.tsx
8	7	web/apps/web/core/components/project-states/group-item.tsx
7	6	web/apps/web/core/components/project-states/group-list.tsx
3	2	web/apps/web/core/components/project-states/options/delete.tsx
5	4	web/apps/web/core/components/project-states/root.tsx
3	2	web/apps/web/core/components/project-states/state-item-title.tsx
7	6	web/apps/web/core/components/project-states/state-item.tsx
6	5	web/apps/web/core/components/project-states/state-list.tsx
4	4	web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
8	8	web/apps/web/core/store/issue/root.store.ts
4	4	web/packages/constants/src/state.ts
2	2	web/packages/types/src/issues/issue.ts
2	2	web/packages/types/src/project/projects.ts
4	3	web/packages/utils/src/distribution-update.ts
7	6	web/packages/utils/src/progress.test.ts
7	6	web/packages/utils/src/work-item-filters/configs/filters/state.ts
3	8	web/packages/utils/src/work-item/base.ts
3	3	web/packages/utils/src/work-item/modal.ts
3	3	web/packages/utils/src/work-item/state.ts
```

Run（机械步骤）: `shasum -a 256 web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx web/apps/web/core/components/dropdowns/state/base.tsx web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx web/apps/web/core/components/issues/preview-card/date.tsx web/apps/web/core/components/issues/preview-card/root.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx web/apps/web/core/components/project-states/create-update/create.tsx web/apps/web/core/components/project-states/create-update/form.tsx web/apps/web/core/components/project-states/create-update/update.tsx web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/group-list.tsx web/apps/web/core/components/project-states/options/delete.tsx web/apps/web/core/components/project-states/root.tsx web/apps/web/core/components/project-states/state-item-title.tsx web/apps/web/core/components/project-states/state-item.tsx web/apps/web/core/components/project-states/state-list.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/packages/constants/src/state.ts web/packages/types/src/issues/issue.ts web/packages/types/src/project/projects.ts web/packages/utils/src/distribution-update.ts web/packages/utils/src/progress.test.ts web/packages/utils/src/work-item-filters/configs/filters/state.ts web/packages/utils/src/work-item/base.ts web/packages/utils/src/work-item/modal.ts web/packages/utils/src/work-item/state.ts`
Expected: 每个文件的散列和行数与下表相同：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `066b22fbf2d1c2a94f8eab26e5b9aa297dc103a759e6245be3c527776b247949` | 52 | `web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx` |
| `8e319330012739626738e36539e4291c8a8f6e5ef23414f7cc3026691c4bbd09` | 256 | `web/apps/web/core/components/dropdowns/state/base.tsx` |
| `e48ea416b22294a9d4b0ff3c431cf03a4bbc0ddfb20744bdf4009ec3b8b13825` | 162 | `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx` |
| `4aedab1c2402d14b49e5d276f9de38f3e06359096d39f23fa62ca48a37969e0c` | 98 | `web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx` |
| `89c7868fdd6e0cb67305375d85ba04807ad52a657a9b2dd7cf5042383572fbb4` | 56 | `web/apps/web/core/components/issues/preview-card/date.tsx` |
| `af95776c4684ad1cce3dcbbcb276bd524effedcfcdb350777e449236b9266131` | 69 | `web/apps/web/core/components/issues/preview-card/root.tsx` |
| `fddcab2f516f6f92efbef51424998005d461f6ac33ef3056a93c034a23ef1e0a` | 37 | `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx` |
| `7ebd840ab375ca212261c985b8e2d9686b2ad30362af0bcb8a3dc2ae0812204a` | 75 | `web/apps/web/core/components/project-states/create-update/create.tsx` |
| `26e0c8f08cc6c86e0c777b003becc057194250869fbcb8727f3e11a2eda54d11` | 122 | `web/apps/web/core/components/project-states/create-update/form.tsx` |
| `71317cafdf5da207977d56bb70d6b8175c1c83c27742ba697b4a1ca646288f15` | 72 | `web/apps/web/core/components/project-states/create-update/update.tsx` |
| `6d517962873c39e33a93eca5bc75a09f436ffa608f8b2b864826a6ad1075e3af` | 132 | `web/apps/web/core/components/project-states/group-item.tsx` |
| `15c09d419fef34b6d241263cec2e4b75538b07929bbbf0a11c6c937164cdca27` | 83 | `web/apps/web/core/components/project-states/group-list.tsx` |
| `797b7c369931ce75533c760de5db763ba1a77e91dcec34fb86d4bc6e6dca9c83` | 105 | `web/apps/web/core/components/project-states/options/delete.tsx` |
| `a4528a8d9de2cc7a9399cddaed5228dc69b740a42ff89b9a715dd99c755bcbd3` | 77 | `web/apps/web/core/components/project-states/root.tsx` |
| `cb02eda939301ec565699c9e3f976a8fbcacaa8b0b9dcceb7f4659f6c8737ebd` | 92 | `web/apps/web/core/components/project-states/state-item-title.tsx` |
| `0f11168b291450cdfb9cd8f0c032d11f083037bcfce2925792366ea981c4bfa1` | 159 | `web/apps/web/core/components/project-states/state-item.tsx` |
| `1c94b92661308ad607779299a4b93972abce6f43d8c51142a597cab223d58f98` | 41 | `web/apps/web/core/components/project-states/state-list.tsx` |
| `ba28882c03adb7aa6177da6b1597972cf933e31d3f7714ee4772ff9aa18cd0cf` | 396 | `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx` |
| `b55363b8f0af8935ca3d0bb8c4c9a3baa01ba4af53b7db33a21142b9ba092fa0` | 212 | `web/apps/web/core/store/issue/root.store.ts` |
| `e283cd5a1afb02effb8db1a0438729a0e05f27ff7a9fd44a440db01e67e05e06` | 101 | `web/packages/constants/src/state.ts` |
| `39f6ba6a10e1d618235d35b68866ae4725ba5783dee3974ccffbecf72bb56dc6` | 132 | `web/packages/types/src/issues/issue.ts` |
| `fbecc90efb063d5ee1cb57043b0864fed57f4224ae06848e63f843fbe045d0dc` | 39 | `web/packages/types/src/project/projects.ts` |
| `425fa38f51812c183986db986d1bcdb2b9b91930f57f6fb8d721851a3a6d5dc3` | 220 | `web/packages/utils/src/distribution-update.ts` |
| `a6455eb43dc410a68a6c2416825146a2639108bdde19cb4e2cce1b95818ceed8` | 170 | `web/packages/utils/src/progress.test.ts` |
| `3f8a8d523bc2d028860ac3a54b912eaddce2372e2dacdeb46adc1b3332e9e6c0` | 122 | `web/packages/utils/src/work-item-filters/configs/filters/state.ts` |
| `bf5ec20b4c8d16060e067bb09dfb292017da6a4551115b43128bdd06f4d49244` | 188 | `web/packages/utils/src/work-item/base.ts` |
| `0dac872d71fbec7bb428d973c23770fb10f937777e9f4603b27991900cff34bf` | 53 | `web/packages/utils/src/work-item/modal.ts` |
| `71991c16b5402147d47587b41b8e910935b88d2ef2e998bf77d4ba4b497d944b` | 51 | `web/packages/utils/src/work-item/state.ts` |

- [ ] **Step 2: service、store、`reconciled.ts` 和排序**

`web/apps/web/core/hooks/store/fake-store-hooks.ts`（修改，2 处）：

````old web/apps/web/core/hooks/store/fake-store-hooks.ts
// A stand-in for the store hooks that the hooks of a page's fetches read (use-workspace.ts, use-member.ts,
// use-project.ts and user's useUserProfile), for their tests: a test file mocks each of those modules with this one,
// for instance vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks")), says in
// `stores` what the stores hold, and reads there what they were asked to fetch. The hooks then run as plain functions, outside React,
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
// A stand-in for the store hooks that the hooks of a page's fetches read (the modules of core/hooks/store that have
// the hooks below), for their tests: a test file mocks each of those modules with this one, for instance
// vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks")), says in `stores` what the
// stores hold, and reads there what they were asked to fetch. The hooks then run as plain functions, outside React,
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
}

export function useUserProfile() {
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
}

export function useProjectState() {
  return {
    fetchWorkspaceStates: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the states of ${named(workspace)}`),
  };
}

export function useUserProfile() {
````

`web/apps/web/core/lib/reconciled.ts`（修改，1 处）：

````old web/apps/web/core/lib/reconciled.ts
  }

  /** Fetches the key's value with read (Reconciled.fetch). */
````
````new web/apps/web/core/lib/reconciled.ts
  }

  /** The values of the keys fetched, whatever their keys (for one, to find an item in any of them). */
  values(): V[] {
    return [...this.entries.values()].flatMap((entry) => (entry.value === undefined ? [] : [entry.value]));
  }

  /** Fetches the key's value with read (Reconciled.fetch). */
````

`web/apps/web/core/services/project/index.ts`（修改，1 处）：

````old web/apps/web/core/services/project/index.ts
export * from "./project.service";
export * from "./project-state.service";
````
````new web/apps/web/core/services/project/index.ts
export * from "./project.service";
````

`web/apps/web/core/services/project/project-state.service.ts`（删除）：

````delete web/apps/web/core/services/project/project-state.service.ts
````

`web/apps/web/core/services/project/states.service.ts`（新文件，52 行）：

````file web/apps/web/core/services/project/states.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, State, StateCreate, StateUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The states of a project's work items (M3 design 3.17, 5.1); the intake's triage state is none of them. */
export class StatesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's states by sequence, the lowest first; none of an archived project. */
  async list(projectId: string): Promise<State[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/states", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** The states of the workspace's projects the caller is a member of, the archived ones' left out. */
  async listInWorkspace(slug: string): Promise<State[]> {
    return unwrap(await this.api.GET("/api/v0/workspaces/{slug}/states", { params: { path: { slug } } })).data;
  }

  /** Creates a state, last of the project's: not its default. */
  async create(projectId: string, data: StateCreate): Promise<State> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/states", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    );
  }

  /** Changes the fields data names; the answer is the state as nerve now holds it. */
  async update(stateId: string, data: StateUpdate): Promise<State> {
    return unwrap(
      await this.api.PATCH("/api/v0/states/{state_id}", { params: { path: { state_id: stateId } }, body: data })
    );
  }

  /** Deletes a state that is neither its project's default nor the last of its group. */
  async delete(stateId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/states/{state_id}", { params: { path: { state_id: stateId } } }));
  }

  /** Makes the state its project's default; the one that was is no longer. */
  async markDefault(stateId: string): Promise<void> {
    unwrap(await this.api.POST("/api/v0/states/{state_id}/mark-default", { params: { path: { state_id: stateId } } }));
  }
}
````

`web/apps/web/core/store/state.store.test.ts`（新文件，323 行）：

````file web/apps/web/core/store/state.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { State, StateCreate, StateGroup, StateUpdate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { ProjectRootStore } from "@/store/project";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RouterStore } from "@/store/router.store";
import { StateStore } from "@/store/state.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The states of the caller's projects (M3 design 3.17, 7.3), against a fake nerve that answers each request when the
// test says. The tab's address is web's, a project of acme; ops is acme's other.

const acme = workspaceOf("acme");
const web = projectOf("WEB", acme.id);
const ops = projectOf("OPS", acme.id);
const LIST = `/api/v0/projects/${web.id}/states`;
const ACME = "/api/v0/workspaces/acme/states";

/** A state of a project (web unless it says) as nerve gives it: the name names it in its project. */
function stateOf(name: string, group: StateGroup, sequence: number, fields: Partial<State> = {}, project = web): State {
  return {
    id: `s-${project.identifier}-${name}`,
    workspace_id: project.workspace_id,
    project_id: project.id,
    name,
    description: "",
    color: "#60646C",
    group,
    default: false,
    sequence,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}
const backlog = stateOf("Backlog", "backlog", 15000, { default: true });
const todo = stateOf("Todo", "unstarted", 30000);
const doing = stateOf("Doing", "started", 45000);
const review = stateOf("Review", "started", 50000);
const done = stateOf("Done", "completed", 40000);
/** web's states by group, then sequence: the started group has two, whose sequences are above done's. */
const listed = [backlog, todo, doing, review, done];
const opsBacklog = stateOf("Backlog", "backlog", 15000, { default: true }, ops);
/** Doing moved before Todo, as nerve answers it: a change the request does not make shows its updated_at. */
const moved: State = { ...doing, group: "unstarted", sequence: 20000, updated_at: "2026-10-08T09:00:00Z" };
/** A state created in web's started group, as the page sends it and as nerve answers it. */
const blocking: StateCreate = { name: "Blocked", color: "#60646C", group: "started" };
const blocked = stateOf("Blocked", "started", 75000);
const ids = (states: State[]) => states.map((state) => state.id);
const at = (state: State) => `/api/v0/states/${state.id}`;

/** The store of a tab at web's address, whose caller's acme nerve listed with its web and ops. */
async function stateStore() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  router.setQuery({ workspaceSlug: "acme", projectId: web.id });
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  await loadWorkspaces(nerve, workspaceRoot, [acme]);
  await loadProjects(nerve, projectRoot.project, acme, [web, ops]);
  nerve.calls.length = 0;
  const store = new StateStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  return { nerve, api, projects: projectRoot.project, store };
}

/** The store fetches web's states, and nerve lists these. */
const load = (nerve: FakeNerve, store: StateStore, states: State[]) =>
  answered(nerve, () => store.fetchProjectStates(web.id), ["GET", LIST], { data: states }, "the states");

/** A store whose states of web nerve listed. */
async function loaded() {
  const tab = await stateStore();
  await load(tab.nerve, tab.store, listed);
  return tab;
}

/** Sends a change, which nerve gives reply once it is the request out; gives how the change settled. */
async function sent(nerve: FakeNerve, send: () => Promise<unknown>, request: Endpoint, reply: Response) {
  const k = nerve.calls.length;
  const change = track(send());
  await inTurn(nerve, k, request, reply);
  await until(() => change.settled, "the change");
  return change;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("StateStore, the states", () => {
  it("keeps a project's states by group, then sequence, and each one's place in its group", async () => {
    const { nerve, store } = await stateStore();
    const fetched = await load(nerve, store, [doing, done, review, backlog, todo]);
    expect(fetched.value).toEqual([doing, done, review, backlog, todo]);
    expect(store.getProjectStates(web.id)).toEqual(listed);
    expect(store.projectStates).toEqual(listed);
    expect(store.getProjectStateIds(web.id)).toEqual(ids(listed));
    expect(store.groupedProjectStates).toEqual({
      backlog: [backlog],
      unstarted: [todo],
      started: [doing, review],
      completed: [done],
      cancelled: [],
    });
    expect(store.getStatePercentageInGroup(doing.id)).toBe(50);
    expect(store.getStatePercentageInGroup(review.id)).toBe(100);
    expect(store.getStateById(todo.id)).toEqual(todo);
    expect(store.getProjectStates(ops.id)).toBeUndefined();
  });

  it("gives a project's states from its workspace's list until its own is fetched, and none of a project left", async () => {
    const { nerve, projects, store } = await stateStore();
    const fetch = () => store.fetchWorkspaceStates(acme);
    await answered(nerve, fetch, ["GET", ACME], { data: [opsBacklog, ...listed] }, "acme's states");
    // by group, then sequence; nerve's order where both are equal
    expect(store.workspaceStates).toEqual([opsBacklog, backlog, todo, doing, review, done]);
    expect(store.getProjectStates(ops.id)).toEqual([opsBacklog]);
    expect(store.getProjectStates(web.id)).toEqual(listed);
    await load(nerve, store, [backlog, done]);
    expect(store.getProjectStates(web.id)).toEqual([backlog, done]);

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectStates(ops.id)).toBeUndefined();
    expect(store.getStateById(opsBacklog.id)).toBeUndefined();
    expect(store.workspaceStates).toEqual([backlog, todo, doing, review, done]);
  });

  it("fails when nerve refuses a project's or the workspace's, keeping none, and again, keeping the states it had", async () => {
    const { nerve, store } = await stateStore();
    const refused = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 1, "the states");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectStates(web.id)).toBeUndefined();
    const workspace = track(store.fetchWorkspaceStates(acme));
    await until(() => nerve.calls.length === 2, "acme's states");
    nerve.calls[1]?.answer(problem(403, "forbidden"));
    await until(() => workspace.settled, "the refusal");
    expect(workspace.error).toBeInstanceOf(ApiError);
    expect(store.workspaceStates).toBeUndefined();

    await load(nerve, store, listed);
    const again = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 4, "the refetch");
    nerve.calls[3]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getProjectStates(web.id)).toEqual(listed);
  });

  it("keeps the states it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjectStates(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectStates(web.id)).toEqual(listed);
  });
});

describe("StateStore, the changes", () => {
  it("adds a created state as nerve answers it, last of its project's", async () => {
    const { nerve, store } = await loaded();
    const created = await sent(nerve, () => store.createState(web.id, blocking), ["POST", LIST], json(201, blocked));
    expect(nerve.calls[1]?.body).toEqual(blocking);
    expect(created.value).toEqual(blocked);
    expect(store.groupedProjectStates?.started).toEqual([doing, review, blocked]);
    expect(store.getStatePercentageInGroup(review.id)).toBeCloseTo(66.67, 2);
  });

  it("moves a state to the place nerve answers, in its project's list and in its workspace's", async () => {
    const { nerve, store } = await loaded();
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: listed }, "acme's states");
    const data: StateUpdate = { group: "unstarted", sequence: 20000 };
    const changed = track(store.updateState(doing.id, data));
    await until(() => nerve.calls.length === 3, "the move");
    expect(nerve.calls[2]).toMatchObject({ method: "PATCH", path: at(doing), body: data });
    expect(store.getStateById(doing.id)).toEqual(doing);
    nerve.calls[2]?.answer(json(200, moved));
    await until(() => changed.settled, "the answer");
    expect(store.getProjectStates(web.id)).toEqual([backlog, moved, todo, review, done]);
    expect(store.workspaceStates).toEqual([backlog, moved, todo, review, done]);
    expect(store.getStatePercentageInGroup(review.id)).toBe(100);
  });

  it("deletes a state, and makes another its project's default, the one that was no longer", async () => {
    const { nerve, store } = await loaded();
    await answered(nerve, () => store.fetchWorkspaceStates(acme), ["GET", ACME], { data: [opsBacklog] }, "acme");
    const deleted = await sent(nerve, () => store.deleteState(review.id), ["DELETE", at(review)], noContent());
    expect(deleted.error).toBeUndefined();
    expect(store.getProjectStateIds(web.id)).toEqual(ids([backlog, todo, doing, done]));

    await sent(nerve, () => store.markStateAsDefault(todo.id), ["POST", `${at(todo)}/mark-default`], noContent());
    expect(store.getProjectStates(web.id)?.filter((state) => state.default)).toEqual([{ ...todo, default: true }]);
    // ops's default is its own
    expect(store.getStateById(opsBacklog.id)?.default).toBe(true);
  });

  const refusals: { change: string; send: (store: StateStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createState(web.id, { name: "Todo", color: "#60646C", group: "unstarted" }),
      refusal: problem(409, "project.state_name_taken"),
    },
    {
      change: "a move",
      send: (store) => store.updateState(done.id, { group: "started" }),
      refusal: problem(409, "project.state_last_in_group"),
    },
    {
      change: "a deletion",
      send: (store) => store.deleteState(backlog.id),
      refusal: problem(409, "project.state_default"),
    },
    {
      change: "a new default",
      send: (store) => store.markStateAsDefault(todo.id),
      refusal: problem(403, "forbidden"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectStates(web.id)).toEqual(listed);
  });

  it.each(refusals.slice(2))(
    "fails, asking nerve nothing, for $change of a state it does not have",
    async ({ send }) => {
      const { nerve, store } = await stateStore();
      const refused = await settle(send(store), "the change");
      expect(refused).toMatchObject({ settled: true, error: new Error("State not found") });
      expect(nerve.calls).toEqual([]);
    }
  );

  it("sends each change once nerve has answered the one before it, refused or not: two moves in a row among them", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updateState(doing.id, { sequence: 1 }));
    const second = track(store.updateState(doing.id, { group: "unstarted", sequence: 20000 }));
    const marked = track(store.markStateAsDefault(todo.id));
    await inTurn(nerve, 1, ["PATCH", at(doing)], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", at(doing)], json(200, moved));
    await inTurn(nerve, 3, ["POST", `${at(todo)}/mark-default`], noContent());
    await until(() => marked.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(second.value).toEqual(moved);
    expect(store.getStateById(todo.id)?.default).toBe(true);
  });

  it("fetches the states while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateState(doing.id, { sequence: 1 }), request: ["PATCH", at(doing)] },
      { send: () => store.fetchProjectStates(web.id), request: ["GET", LIST], body: { data: [backlog, done] } }
    );
    expect(store.getProjectStates(web.id)).toEqual([backlog, done]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one project's states, only the newer writes.
describe("StateStore, while a fetch is out", () => {
  it("shows once a state created during a refetch whose list has it, and the changes confirmed meanwhile", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(nerve, () => store.createState(web.id, blocking), ["POST", LIST], json(201, blocked));
    await sent(nerve, () => store.updateState(doing.id, { sequence: 20000 }), ["PATCH", at(doing)], json(200, moved));
    await sent(nerve, () => store.deleteState(done.id), ["DELETE", at(done)], noContent());
    // read after the creation, before the move and the deletion
    nerve.calls[1]?.answer(json(200, { data: [...listed, blocked] }));
    await until(() => refetched.settled, "the refetch");

    const shown = [backlog, moved, todo, review, blocked];
    expect(store.getProjectStates(web.id)).toEqual(shown);
    expect(refetched.value && ids(refetched.value)).toEqual(ids([backlog, todo, moved, review, blocked]));
  });

  it("lets each project's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await stateStore();
    const older = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 1, "the older states");
    const opsStates = track(store.fetchProjectStates(ops.id));
    await until(() => nerve.calls.length === 2, "ops's states");
    const newer = track(store.fetchProjectStates(web.id));
    await until(() => nerve.calls.length === 3, "the newer states");
    nerve.calls[2]?.answer(json(200, { data: [backlog, moved, done] }));
    await until(() => newer.settled, "the newer states");
    // a newer fetch of web's states does not overtake one of ops's
    nerve.calls[1]?.answer(json(200, { data: [opsBacklog] }));
    await until(() => opsStates.settled, "ops's states");
    // read before Doing moved
    nerve.calls[0]?.answer(json(200, { data: listed }));
    await until(() => older.settled, "the older states");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getProjectStates(web.id)).toEqual([backlog, moved, done]);
    expect(store.getProjectStates(ops.id)).toEqual([opsBacklog]);
  });
});
````

`web/apps/web/core/store/state.store.ts`（整个文件，214 行）：

````whole web/apps/web/core/store/state.store.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Project, State, StateCreate, StateUpdate, Workspace } from "@nerve/api-client";
import { STATE_GROUPS } from "@nerve/constants";
import { sortStates } from "@nerve/utils";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, dropped, replaced, upserted } from "@/lib/reconciled";
// services
import { StatesService } from "@/services/project/states.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IRouterStore } from "@/store/router.store";

/** A workspace as the store fetches its states: by its slug, kept under its id. */
type WorkspaceRef = Pick<Workspace, "id" | "slug">;

export interface IStateStore {
  // computed
  /** Every state the store shows, by id, which the work items' stores look up. */
  stateMap: Record<string, State>;
  /** The current workspace's states, by group, then sequence; undefined until fetched. */
  workspaceStates: State[] | undefined;
  /** The address's project's states, by group, then sequence; undefined until fetched. */
  projectStates: State[] | undefined;
  /** The address's project's states in each group, every group there; undefined until fetched. */
  groupedProjectStates: Record<string, State[]> | undefined;
  // computed actions
  getStateById: (stateId: string | null | undefined) => State | undefined;
  getProjectStates: (projectId: string | null | undefined) => State[] | undefined;
  getProjectStateIds: (projectId: string | null | undefined) => string[] | undefined;
  getStatePercentageInGroup: (stateId: string | null | undefined) => number | undefined;
  // fetch actions
  fetchProjectStates: (projectId: string) => Promise<State[] | undefined>;
  fetchWorkspaceStates: (workspace: WorkspaceRef) => Promise<State[] | undefined>;
  // changes
  createState: (projectId: string, data: StateCreate) => Promise<State>;
  updateState: (stateId: string, data: StateUpdate) => Promise<State>;
  deleteState: (stateId: string) => Promise<void>;
  markStateAsDefault: (stateId: string) => Promise<void>;
}

/**
 * The states of the projects of a session (M3 design 3.17, 7.3): each project's list, by its id, which its pages
 * fetch, and each workspace's, by its id, the states of its projects the caller is a member of. Their order is
 * sortStates' (group, then sequence), and a state's place in its group is computed from it: nerve gives no order. The
 * states of a project the project store no longer gives (deleted, left, or of a workspace no longer the caller's) do
 * not show. Its service sends with the session's client; changes go one at a time and the store writes nerve's
 * answers (v0 design 7.7); fetches do not queue.
 */
export class StateStore implements IStateStore {
  /** Each project's states, reconciled between fetches and changes (reconciled.ts). */
  private readonly projects = new ReconciledByKey<State[]>();
  /** Each workspace's states. */
  private readonly workspaces = new ReconciledByKey<State[]>();
  // services
  private readonly service: StatesService;
  /** The changes of the states, sent one at a time. */
  private readonly changes = oneAtATime();
  // stores
  private readonly router: IRouterStore;
  private readonly rootStore: RootStore;
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      stateMap: computed,
      workspaceStates: computed,
      projectStates: computed,
      groupedProjectStates: computed,
      // actions
      fetchProjectStates: action,
      fetchWorkspaceStates: action,
      createState: action,
      updateState: action,
      deleteState: action,
      markStateAsDefault: action,
    });
    this.service = new StatesService(api);
    this.router = _rootStore.router;
    this.rootStore = _rootStore;
    this.projectOf = _rootStore.projectRoot.project.getProjectById;
  }

  get stateMap() {
    const held = [...this.projects.values(), ...this.workspaces.values()].flat();
    return Object.fromEntries(
      held.filter((state) => this.projectOf(state.project_id)).map((state) => [state.id, state])
    );
  }

  get workspaceStates() {
    const states = this.workspaces.get(this.rootStore.workspaceRoot.currentWorkspace?.id);
    return states && sortStates(states.filter((state) => this.projectOf(state.project_id)));
  }

  get projectStates() {
    return this.getProjectStates(this.router.projectId);
  }

  get groupedProjectStates() {
    const states = this.projectStates;
    if (!states) return undefined;
    return Object.fromEntries(
      Object.keys(STATE_GROUPS).map((group) => [group, states.filter((state) => state.group === group)])
    );
  }

  /** @description the state, of a project the caller sees, as the store last had it from nerve */
  getStateById = computedFn((stateId: string | null | undefined): State | undefined =>
    stateId ? this.stateMap[stateId] : undefined
  );

  /**
   * @description the project's states, by group, then sequence: its own list, else its workspace's; undefined until
   * fetched, or once the project store no longer gives the project
   */
  getProjectStates = computedFn((projectId: string | null | undefined): State[] | undefined => {
    const project = this.projectOf(projectId);
    if (!project) return undefined;
    const listed =
      this.projects.get(project.id) ??
      this.workspaces.get(project.workspace_id)?.filter((state) => state.project_id === project.id);
    return listed && sortStates(listed);
  });

  getProjectStateIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.getProjectStates(projectId)?.map((state) => state.id)
  );

  /** @description the state's place in its group, in its project's order: 100 for the group's last, as a percentage */
  getStatePercentageInGroup = computedFn((stateId: string | null | undefined): number | undefined => {
    const state = this.getStateById(stateId);
    if (!state) return undefined;
    const group = this.getProjectStates(state.project_id)?.filter((held) => held.group === state.group) ?? [];
    const place = group.findIndex((held) => held.id === state.id);
    return place === -1 ? undefined : ((place + 1) / group.length) * 100;
  });

  /**
   * @description fetches a project's states, a member's to fetch as nerve refuses anyone else, and shows them with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchProjectStates = (projectId: string): Promise<State[] | undefined> =>
    this.projects.fetch(projectId, () => this.service.list(projectId));

  /** @description fetches the states of the workspace's projects the caller is a member of, as fetchProjectStates */
  fetchWorkspaceStates = (workspace: WorkspaceRef): Promise<State[] | undefined> =>
    this.workspaces.fetch(workspace.id, () => this.service.listInWorkspace(workspace.slug));

  /** @description creates a state, last of its project's; fails, changing nothing, when nerve refuses */
  createState = (projectId: string, data: StateCreate): Promise<State> =>
    this.changes(async () => {
      const state = await this.service.create(projectId, data);
      this.confirm(state, upserted(state));
      return state;
    });

  /**
   * @description changes a state, its place among its project's (group, sequence) too; the store then shows nerve's
   * answer. Fails, changing nothing, when nerve refuses.
   */
  updateState = (stateId: string, data: StateUpdate): Promise<State> =>
    this.changes(async () => {
      const state = await this.service.update(stateId, data);
      this.confirm(state, replaced(state));
      return state;
    });

  /** @description deletes a state; fails, changing nothing, when nerve refuses or the store does not have it */
  deleteState = (stateId: string): Promise<void> =>
    this.changes(async () => {
      const state = this.held(stateId);
      await this.service.delete(state.id);
      this.confirm(state, dropped(state.id));
    });

  /**
   * @description makes a state its project's default, and the one that was no longer; fails, changing nothing, when
   * nerve refuses or the store does not have it
   */
  markStateAsDefault = (stateId: string): Promise<void> =>
    this.changes(async () => {
      const state = this.held(stateId);
      await this.service.markDefault(state.id);
      this.confirm(state, (list) =>
        list.map((held) => (held.project_id === state.project_id ? { ...held, default: held.id === state.id } : held))
      );
    });

  /** The state as the store has it; fails when it has none. */
  private held(stateId: string): State {
    const state = this.getStateById(stateId);
    if (!state) throw new Error("State not found");
    return state;
  }

  /** A change nerve confirmed to a state's project: made on its project's list and on its workspace's. */
  private confirm(state: Pick<State, "project_id" | "workspace_id">, change: Change<State[]>): void {
    this.projects.confirm(state.project_id, change);
    this.workspaces.confirm(state.workspace_id, change);
  }
}
````

`web/packages/types/src/state.ts`（整个文件，14 行）：

````whole web/packages/types/src/state.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { State, StateCreate, StateUpdate } from "@nerve/api-client";

export type TStateOperationsCallbacks = {
  createState: (data: StateCreate) => Promise<State>;
  updateState: (stateId: string, data: StateUpdate) => Promise<State>;
  deleteState: (stateId: string) => Promise<void>;
  markStateAsDefault: (stateId: string) => Promise<void>;
};
````

`web/packages/utils/package.json`（修改，1 处）：

````old web/packages/utils/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 10",
````
````new web/packages/utils/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 9",
````

`web/packages/utils/src/work-item/state.ts`（修改，2 处）：

````old web/packages/utils/src/work-item/state.ts
export const sortStates = (states: State[]) => {
  if (!states || states.length === 0) return;

  return states.sort((stateA, stateB) => {
````
````new web/packages/utils/src/work-item/state.ts
/** The states by group, in STATE_GROUPS' order, then by sequence, the lowest first; the list given is left as it is. */
export const sortStates = (states: State[]): State[] =>
  states.toSorted((stateA, stateB) => {
````
````old web/packages/utils/src/work-item/state.ts
  });
};
````
````new web/packages/utils/src/work-item/state.ts
  });
````

- [ ] **Step 3: 挂载时的取数**

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，1 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  useSWR(PROJECT_STATES(projectId, currentProjectRole), () => fetchProjectStates(workspaceSlug, projectId), {
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  useSWR(PROJECT_STATES(projectId, currentProjectRole), () => fetchProjectStates(projectId), {
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts`（修改，5 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-state", () => import("@/hooks/store/fake-store-hooks"));
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members, his settings and its projects", async () => {
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
  it("fetches the caller's workspaces and, in one of his, its members, his settings, its projects, their states", async () => {
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      ["PROJECTS", "id-acme", "acme"],
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      ["PROJECTS", "id-acme", "acme"],
      ["WORKSPACE_STATES", "id-acme", "acme"],
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      "the projects of acme (id-acme)",
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
      "the projects of acme (id-acme)",
      "the states of acme (id-acme)",
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null, null]);
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts
    expect(handed.map(([fetch]) => fetch)).toEqual([["WORKSPACES"], null, null, null, null]);
````

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`（修改，4 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
import { useProject } from "@/hooks/store/use-project";
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
 * his navigation settings in it and its projects that are not archived. Gives what the wrapper shows.
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
 * his navigation settings in it, its projects that are not archived and the states of those he is a member of. Gives
 * what the wrapper shows.
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  const { fetchProjects } = useProject();
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  const { fetchProjects } = useProject();
  const { fetchWorkspaceStates } = useProjectState();
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  useSessionSWR(workspace && ["PROJECTS", workspace.id, workspace.slug], (id, slug) => fetchProjects({ id, slug }));
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  useSessionSWR(workspace && ["PROJECTS", workspace.id, workspace.slug], (id, slug) => fetchProjects({ id, slug }));
  useSessionSWR(workspace && ["WORKSPACE_STATES", workspace.id, workspace.slug], (id, slug) =>
    fetchWorkspaceStates({ id, slug })
  );
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，4 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { useParams, Link } from "react-router";
import useSWR from "swr";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { useParams, Link } from "react-router";
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { NerveLogo } from "@/components/common/nerve-logo";
// constants
import { WORKSPACE_STATES } from "@nerve/constants";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
import { NerveLogo } from "@/components/common/nerve-logo";
// hooks
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { isMobile } = usePlatformOS();
  const { fetchWorkspaceStates } = useProjectState();
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const { isMobile } = usePlatformOS();
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const access = useWorkspaceFetch(workspaceSlug);

  // fetch workspace states
  useSWR(
    workspaceSlug ? WORKSPACE_STATES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceStates(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
  const access = useWorkspaceFetch(workspaceSlug);
````

- [ ] **Step 4: 使用方、键和静态检查**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/store/project/**",
        "web/apps/web/core/components/projects/page.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/store/project/**",
        "web/apps/web/core/components/projects/page.tsx",
        "web/apps/web/core/store/state.store.ts",
        "web/apps/web/core/components/project-states/**",
        "web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx"
````
````old .oxlintrc.json
        "web/apps/web/core/services/project/project-members.service.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/services/project/project-members.service.ts",
        "web/apps/web/core/services/project/states.service.ts",
        "web/apps/web/core/store/state.store.ts",
        "web/apps/web/core/store/state.store.test.ts",
````

`web/apps/web/core/components/dropdowns/state/base.tsx`（修改，4 处）：

````old web/apps/web/core/components/dropdowns/state/base.tsx
  getStateById: (stateId: string | null | undefined) => State | undefined;
````
````new web/apps/web/core/components/dropdowns/state/base.tsx
  getStateById: (stateId: string | null | undefined) => State | undefined;
  getStatePercentageInGroup: (stateId: string | null | undefined) => number | undefined;
````
````old web/apps/web/core/components/dropdowns/state/base.tsx
    getStateById,
````
````new web/apps/web/core/components/dropdowns/state/base.tsx
    getStateById,
    getStatePercentageInGroup,
````
````old web/apps/web/core/components/dropdowns/state/base.tsx
          percentage={state?.order}
````
````new web/apps/web/core/components/dropdowns/state/base.tsx
          percentage={getStatePercentageInGroup(state?.id)}
````
````old web/apps/web/core/components/dropdowns/state/base.tsx
                percentage={selectedState?.order}
````
````new web/apps/web/core/components/dropdowns/state/base.tsx
                percentage={getStatePercentageInGroup(selectedState?.id)}
````

`web/apps/web/core/components/dropdowns/state/dropdown.tsx`（整个文件，48 行）：

````whole web/apps/web/core/components/dropdowns/state/dropdown.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
// local imports
import type { TWorkItemStateDropdownBaseProps } from "./base";
import { WorkItemStateDropdownBase } from "./base";

type TWorkItemStateDropdownProps = Omit<
  TWorkItemStateDropdownBaseProps,
  "stateIds" | "getStateById" | "getStatePercentageInGroup" | "onDropdownOpen" | "isInitializing"
>;

export const StateDropdown = observer(function StateDropdown(props: TWorkItemStateDropdownProps) {
  const { projectId } = props;
  // states
  const [stateLoader, setStateLoader] = useState(false);
  // store hooks
  const { fetchProjectStates, getProjectStateIds, getStateById, getStatePercentageInGroup } = useProjectState();
  // derived values
  const stateIds = getProjectStateIds(projectId);

  // fetch states if not provided
  const onDropdownOpen = async () => {
    if ((stateIds === undefined || stateIds.length === 0) && projectId) {
      setStateLoader(true);
      await fetchProjectStates(projectId);
      setStateLoader(false);
    }
  };

  return (
    <WorkItemStateDropdownBase
      {...props}
      getStateById={getStateById}
      getStatePercentageInGroup={getStatePercentageInGroup}
      isInitializing={stateLoader}
      stateIds={stateIds ?? []}
      onDropdownOpen={onDropdownOpen}
    />
  );
});
````

`web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx`（修改，3 处）：

````old web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";
````
````new web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";
import { useProjectState } from "@/hooks/store/use-project-state";
````
````old web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
  const { appliedFilters, handleUpdate, searchQuery, states } = props;
````
````new web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
  const { appliedFilters, handleUpdate, searchQuery, states } = props;
  const { getStatePercentageInGroup } = useProjectState();
````
````old web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
                        percentage={state?.order}
````
````new web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx
                        percentage={getStatePercentageInGroup(state.id)}
````

`web/apps/web/core/components/issues/issue-layouts/utils.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/issue-layouts/utils.tsx
  const { getProjectStates, projectStates } = rootStore.state;
````
````new web/apps/web/core/components/issues/issue-layouts/utils.tsx
  const { getProjectStates, projectStates, getStatePercentageInGroup } = rootStore.state;
````
````old web/apps/web/core/components/issues/issue-layouts/utils.tsx
        <StateGroupIcon stateGroup={state.group} color={state.color} size={EIconSize.LG} percentage={state.order} />
````
````new web/apps/web/core/components/issues/issue-layouts/utils.tsx
        <StateGroupIcon
          stateGroup={state.group}
          color={state.color}
          size={EIconSize.LG}
          percentage={getStatePercentageInGroup(state.id)}
        />
````

`web/apps/web/core/components/project-states/create-update/create.tsx`（修改，6 处）：

````old web/apps/web/core/components/project-states/create-update/create.tsx
import type { State, StateGroup } from "@nerve/api-client";
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
import type { StateGroup } from "@nerve/api-client";
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
import { StateForm } from "@/components/project-states";
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
import { StateForm } from "@/components/project-states";
import type { TStateFormData } from "./form";
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
  const onSubmit = async (formData: Partial<State>) => {
    if (!groupKey) return { status: "error" };
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
  const onSubmit = async (formData: TStateFormData) => {
    if (!groupKey) return;
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
      handleClose();
      return { status: "success" };
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
      handleClose();
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
        });
        return { status: "already_exists" };
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
        });
````
````old web/apps/web/core/components/project-states/create-update/create.tsx
        });
        return { status: "error" };
````
````new web/apps/web/core/components/project-states/create-update/create.tsx
        });
````

`web/apps/web/core/components/project-states/create-update/form.tsx`（修改，6 处）：

````old web/apps/web/core/components/project-states/create-update/form.tsx
import type { State } from "@nerve/api-client";
import { Popover } from "@nerve/ui";
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
import type { StateCreate } from "@nerve/api-client";
import { Popover } from "@nerve/ui";

/** What the form edits of a state, as a creation sends it. */
export type TStateFormData = Omit<StateCreate, "group">;

````
````old web/apps/web/core/components/project-states/create-update/form.tsx
  data: Partial<State>;
  onSubmit: (formData: Partial<State>) => Promise<{ status: string }>;
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
  data: TStateFormData;
  onSubmit: (formData: TStateFormData) => Promise<void>;
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
  const [formData, setFromData] = useState<Partial<State> | undefined>(undefined);
  const [errors, setErrors] = useState<Partial<Record<keyof State, string>> | undefined>(undefined);
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
  const [formData, setFromData] = useState<TStateFormData | undefined>(undefined);
  const [errors, setErrors] = useState<Partial<Record<keyof TStateFormData, string>> | undefined>(undefined);
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
  const handleFormData = <T extends keyof State>(key: T, value: State[T]) => {
    setFromData((prev) => ({ ...prev, [key]: value }));
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
  const handleFormData = <T extends keyof TStateFormData>(key: T, value: TStateFormData[T]) => {
    setFromData((prev) => prev && { ...prev, [key]: value });
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
      let currentErrors: Partial<Record<keyof State, string>> = {};
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
      let currentErrors: Partial<Record<keyof TStateFormData, string>> = {};
````
````old web/apps/web/core/components/project-states/create-update/form.tsx
              maxLength={100}
              autoFocus
````
````new web/apps/web/core/components/project-states/create-update/form.tsx
              maxLength={100}
````

`web/apps/web/core/components/project-states/create-update/update.tsx`（修改，5 处）：

````old web/apps/web/core/components/project-states/create-update/update.tsx
import { StateForm } from "@/components/project-states";
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
import { StateForm } from "@/components/project-states";
import type { TStateFormData } from "./form";
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
  const onSubmit = async (formData: Partial<State>) => {
    if (!state.id) return { status: "error" };
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
  const onSubmit = async (formData: TStateFormData) => {
    if (!state.id) return;
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
      handleClose();
      return { status: "success" };
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
      handleClose();
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
        });
        return { status: "already_exists" };
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
        });
````
````old web/apps/web/core/components/project-states/create-update/update.tsx
        });
        return { status: "error" };
````
````new web/apps/web/core/components/project-states/create-update/update.tsx
        });
````

`web/apps/web/core/components/project-states/options/delete.tsx`（修改，1 处）：

````old web/apps/web/core/components/project-states/options/delete.tsx
      const errorStatus = error as { status: number; data: { error: string } };
````
````new web/apps/web/core/components/project-states/options/delete.tsx
      const errorStatus = error as { status: number };
````

`web/apps/web/core/components/project-states/root.tsx`（整个文件，63 行）：

````whole web/apps/web/core/components/project-states/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { observer } from "mobx-react";
// components
import { EUserPermissionsLevel } from "@nerve/constants";
import type { StateCreate } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { EUserProjectRoles } from "@nerve/types";
import { ProjectStateLoader, GroupList } from "@/components/project-states";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
import { useUserPermissions } from "@/hooks/store/user";
import { useSessionSWR } from "@/lib/use-session-swr";

type TProjectState = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectStateRoot = observer(function ProjectStateRoot(props: TProjectState) {
  const { workspaceSlug, projectId } = props;
  // hooks
  const { groupedProjectStates, fetchProjectStates, createState, updateState, deleteState, markStateAsDefault } =
    useProjectState();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const isEditable = allowPermissions(
    [EUserProjectRoles.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  // Fetching all project states
  useSessionSWR(["PROJECT_STATES", projectId], (id) => fetchProjectStates(id));

  // State operations callbacks
  const stateOperationsCallbacks: TStateOperationsCallbacks = useMemo(
    () => ({
      createState: (data: StateCreate) => createState(projectId, data),
      updateState,
      deleteState,
      markStateAsDefault,
    }),
    [projectId, createState, updateState, deleteState, markStateAsDefault]
  );

  // Loader
  if (!groupedProjectStates) return <ProjectStateLoader />;

  return (
    <GroupList
      groupedStates={groupedProjectStates}
      stateOperationsCallbacks={stateOperationsCallbacks}
      isEditable={isEditable}
    />
  );
});
````

`web/apps/web/core/components/project-states/state-item.tsx`（修改，3 处）：

````old web/apps/web/core/components/project-states/state-item.tsx
    async (payload: Partial<State>) => {
````
````new web/apps/web/core/components/project-states/state-item.tsx
    async (stateId: string, group: StateGroup, sequence: number | undefined) => {
````
````old web/apps/web/core/components/project-states/state-item.tsx
        if (!payload.id) return;
        await stateOperationsCallbacks.moveStatePosition(payload.id, payload);
````
````new web/apps/web/core/components/project-states/state-item.tsx
        await stateOperationsCallbacks.updateState(stateId, { group, sequence });
````
````old web/apps/web/core/components/project-states/state-item.tsx
              const payload: Partial<State> = {
                id: sourceData.id,
                group: destinationGroupKey,
                sequence: getCurrentStateSequence(groupedStates[destinationGroupKey], destinationData, edge),
              };
              handleStateSequence(payload);
````
````new web/apps/web/core/components/project-states/state-item.tsx
              handleStateSequence(
                sourceData.id,
                destinationGroupKey,
                getCurrentStateSequence(groupedStates[destinationGroupKey], destinationData, edge)
              );
````

`web/apps/web/core/store/issue/issue-details/issue.store.ts`（修改，4 处）：

````old web/apps/web/core/store/issue/issue-details/issue.store.ts
      this.issueService.retrieve(workspaceSlug, issue.parent.project_id, issue?.parent?.id).then((res) => {
        this.rootIssueDetailStore.rootIssueStore.issues.addIssue([res]);
      });
````
````new web/apps/web/core/store/issue/issue-details/issue.store.ts
      this.issueService
        .retrieve(workspaceSlug, issue.parent.project_id, issue?.parent?.id)
        .then((res) => this.rootIssueDetailStore.rootIssueStore.issues.addIssue([res]));
````
````old web/apps/web/core/store/issue/issue-details/issue.store.ts
    this.rootIssueDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(workspaceSlug, projectId);
````
````new web/apps/web/core/store/issue/issue-details/issue.store.ts
    this.rootIssueDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(projectId);
````
````old web/apps/web/core/store/issue/issue-details/issue.store.ts
      this.issueService.retrieve(workspaceSlug, issue.parent.project_id, issue.parent.id).then((res) => {
        this.rootIssueDetailStore.rootIssueStore.issues.addIssue([res]);
      });
````
````new web/apps/web/core/store/issue/issue-details/issue.store.ts
      this.issueService
        .retrieve(workspaceSlug, issue.parent.project_id, issue.parent.id)
        .then((res) => this.rootIssueDetailStore.rootIssueStore.issues.addIssue([res]));
````
````old web/apps/web/core/store/issue/issue-details/issue.store.ts
    rootWorkItemDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(workspaceSlug, projectId);
````
````new web/apps/web/core/store/issue/issue-details/issue.store.ts
    rootWorkItemDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(projectId);
````

`web/apps/web/core/store/issue/issue-details/sub_issues.store.ts`（修改，1 处）：

````old web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        this.rootIssueDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(workspaceSlug, projectId);
````
````new web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        this.rootIssueDetailStore.rootIssueStore.rootStore.state.fetchProjectStates(projectId);
````

`web/apps/web/core/store/root.store.ts`（修改，1 处）：

````old web/apps/web/core/store/root.store.ts
    this.state = new StateStore(this);
````
````new web/apps/web/core/store/root.store.ts
    this.state = new StateStore(this, api);
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 370",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 367",
````

`web/packages/constants/src/fetch-keys.ts`（修改，1 处）：

````old web/packages/constants/src/fetch-keys.ts

export const WORKSPACE_STATES = (workspaceSlug: string) => `WORKSPACE_STATES_${workspaceSlug.toUpperCase()}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

`web/packages/propel/src/icons/state/helper.tsx`（修改，4 处）：

````old web/packages/propel/src/icons/state/helper.tsx
 */

````
````new web/packages/propel/src/icons/state/helper.tsx
 */

import type { StateGroup } from "@nerve/api-client";
````
````old web/packages/propel/src/icons/state/helper.tsx
  stateGroup: TStateGroups;
````
````new web/packages/propel/src/icons/state/helper.tsx
  stateGroup: StateGroup;
````
````old web/packages/propel/src/icons/state/helper.tsx

type TStateGroups = "backlog" | "unstarted" | "started" | "completed" | "cancelled";

````
````new web/packages/propel/src/icons/state/helper.tsx

````
````old web/packages/propel/src/icons/state/helper.tsx
  [key in TStateGroups]: string;
````
````new web/packages/propel/src/icons/state/helper.tsx
  [key in StateGroup]: string;
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外，没有命中；web 的 oxlint 367 条、`utils` 9 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx web/apps/web/core/components/dropdowns/state/base.tsx web/apps/web/core/components/dropdowns/state/dropdown.tsx web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/filters.tsx web/apps/web/core/components/issues/issue-layouts/filters/header/filters/state.tsx web/apps/web/core/components/issues/issue-layouts/utils.tsx web/apps/web/core/components/issues/preview-card/date.tsx web/apps/web/core/components/issues/preview-card/root.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/state-menu-item.tsx web/apps/web/core/components/project-states/create-update/create.tsx web/apps/web/core/components/project-states/create-update/form.tsx web/apps/web/core/components/project-states/create-update/update.tsx web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/group-list.tsx web/apps/web/core/components/project-states/options/delete.tsx web/apps/web/core/components/project-states/root.tsx web/apps/web/core/components/project-states/state-item-title.tsx web/apps/web/core/components/project-states/state-item.tsx web/apps/web/core/components/project-states/state-list.tsx web/apps/web/core/hooks/store/fake-store-hooks.ts web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/layouts/auth-layout/use-workspace-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/lib/reconciled.ts web/apps/web/core/services/project/index.ts web/apps/web/core/services/project/project-state.service.ts web/apps/web/core/services/project/states.service.ts web/apps/web/core/store/issue/issue-details/issue.store.ts web/apps/web/core/store/issue/issue-details/sub_issues.store.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/root.store.ts web/apps/web/core/store/state.store.test.ts web/apps/web/core/store/state.store.ts web/apps/web/package.json web/packages/constants/src/fetch-keys.ts web/packages/constants/src/state.ts web/packages/propel/src/icons/state/helper.tsx web/packages/types/src/issues/issue.ts web/packages/types/src/project/projects.ts web/packages/types/src/state.ts web/packages/utils/package.json web/packages/utils/src/distribution-update.ts web/packages/utils/src/progress.test.ts web/packages/utils/src/work-item-filters/configs/filters/state.ts web/packages/utils/src/work-item/base.ts web/packages/utils/src/work-item/modal.ts web/packages/utils/src/work-item/state.ts
```
```bash
git commit -m "feat(M3/P8b): the states come from /api/v0, by a store of the session

The generated State and StateGroup replace Plane's state types, 28 files
of them by a mechanical rename. StatesService wraps the generated
client, and StateStore keeps each project's states and each workspace's
by id, ordered by group, then sequence, with each state's place in its
group computed from that order since nerve gives none. Changes, moves
among them, go one at a time and take nerve's answers. The workspace
wrapper fetches the workspace's states once the caller's list has it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t8-sort-sequence` | 同一组的状态按 `sequence` 从大到小排 | `state.store.test.ts` | vitest |
| `t8-sort-group` | 状态不按组的顺序排 | `state.store.test.ts` | vitest |
| `t8-st-place` | 状态在组中的位置从低一位算起 | `state.store.test.ts` | vitest |
| `t8-st-left` | 项目 store 不再给出的项目，它的状态仍能按 id 找到 | `state.store.test.ts` | vitest |
| `t8-st-no-fallback` | 项目自己的状态没取到时不从工作区的列表给出 | `state.store.test.ts` | vitest |
| `t8-st-create-none` | 创建的状态不存 | `state.store.test.ts` | vitest |
| `t8-st-workspace-list` | 修改不写进工作区的列表 | `state.store.test.ts` | vitest |
| `t8-st-delete-keeps` | 删除的状态仍在 | `state.store.test.ts` | vitest |
| `t8-st-default-old` | 原来的默认状态仍是默认 | `state.store.test.ts` | vitest |
| `t8-st-unheld` | store 没有的状态也照样发出修改 | `state.store.test.ts` | vitest |
| `t8-st-no-queue` | 状态的修改立即发出 | `state.store.test.ts` | vitest |
| `t8-wf-no-states` | 工作区包装层不取工作区的状态 | `use-workspace-fetch.test.ts` | vitest |
| `t8-st-change-swallows` | 状态的修改被拒绝时给出空值，不失败 | `state.store.test.ts` | vitest |
| `t8-st-fetch-swallows` | 状态列表被拒绝时给出 `undefined`，不失败 | `state.store.test.ts` | vitest |
| `t8-st-fetch-queued` | 状态列表的取数排进修改的队列 | `state.store.test.ts` | vitest |
| `t8-rc-upserted-twice` | 重取的列表已有的创建列出两次（`upserted`） | `project.store.test.ts`、`preferences.store.test.ts`、`project-member.store.test.ts`、`state.store.test.ts`、`label.store.test.ts` | vitest |
| `t8-st-workspace-fetch-swallows` | 工作区的状态被拒绝时给出 `undefined`，不失败 | `state.store.test.ts` | vitest |
| `t8-raw-swr-states` | 状态设置页经 `useSWR` 取项目的状态 | oxlint（`check:lint`） | 静态 |
| `t8-raw-swr-workspace-wrapper` | 工作区包装层又经 `useSWR` 取工作区的状态 | oxlint（`check:lint`） | 静态 |
| `t8-nonnull-state` | 状态 store 断言工作区的列表一定在（`!`） | oxlint（`check:lint`） | 静态 |

---

### Task 9: 标签的 service 和 store：两层、nerve 的顺序、排队的移动

**Files:**
- Create: `web/apps/web/core/services/project/labels.service.ts`、`web/apps/web/core/store/label.store.test.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/issues/issue-detail/label/root.tsx`、`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`、`web/apps/web/core/components/issues/select/base.tsx`、`web/apps/web/core/components/issues/select/dropdown.tsx`、`web/apps/web/core/components/labels/create-update-label-inline.tsx`、`web/apps/web/core/components/labels/delete-label-modal.tsx`、`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`、`web/apps/web/core/components/labels/label-utils.ts`、`web/apps/web/core/components/labels/project-setting-label-item.tsx`、`web/apps/web/core/components/labels/project-setting-label-list.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/services/issue/index.ts`、`web/apps/web/core/store/issue/issue-details/sub_issues.store.ts`、`web/apps/web/core/store/label.store.ts`、`web/apps/web/core/store/root.store.ts`、`web/packages/types/src/issues.ts`、`web/packages/utils/package.json`、`web/packages/utils/src/array.ts`
- Delete: `web/apps/web/core/services/issue/issue_label.service.ts`
- 机械步骤（Step 1）改到：`web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx`、`web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx`、`web/apps/web/core/components/labels/label-block/label-item-block.tsx`、`web/apps/web/core/components/labels/project-setting-label-group.tsx`、`web/apps/web/core/components/power-k/menus/labels.tsx`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx`、`web/apps/web/core/components/ui/labels-list.tsx`、`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/store/issue/root.store.ts`、`web/packages/utils/src/work-item-filters/configs/filters/label.ts`

**Interfaces:**
- Produces（spec 2.9；M3 设计 3.16、7.2、7.3；P7b 的交接）：
  - 生成的 `Label` 取代 `IIssueLabel`（Step 1 的机械步骤，22 个文件；`IIssueLabel`、`IIssueLabelTree` 和 `@nerve/utils` 的 `buildTree` 删除）；父标签是 `parent_id`。
  - `core/services/project/labels.service.ts`：`class LabelsService { list(projectId); create(projectId, data: LabelCreate); update(labelId, data: LabelUpdate); delete(labelId) }`（`/api/v0/projects/{id}/labels`、`/api/v0/labels/{id}`；归档的项目也列得出，P7b）。旧的 `issue_label.service.ts` 删除。
  - `core/store/label.store.ts`（整个文件）：`LabelStore(rootStore, api)`；每个项目的标签按项目的 id（`ReconciledByKey`），项目 store 不再给出的项目的标签什么都不给；标签只按项目取，工作区跨项目的标签是 M7 的（Task 7）。顺序是 nerve 的：`sort_order`，再按 id；两层由 `parent_id` 算出：`projectLabelsTree` 是顶层的标签，各带它下面的标签（按同一顺序）。`labelMap`、`projectLabels`、`getLabelById`、`getProjectLabels`、`getProjectLabelIds`；`fetchProjectLabels(projectId)`；修改经 `changes = oneAtATime()`，写入回答：`createLabel(projectId, data)`（不发 `sort_order`，顶层不发 `parent_id: null`，P7b：nerve 排在最后）、`updateLabel(labelId, data)`、`updateLabelPosition(labelId, parentId, droppedOnId, dropAtEndOfList)`（位置在轮到它发出时，从 nerve 最近一次回答的标签算出：在新的同级之间取两邻的中点，放在第一个之前取它的一半，放到最后取最后一个加 10000；新父标签下没有同级时只发 `parent_id`；放在原父标签下、没有放在某个标签上时什么都不发）、`deleteLabel(labelId)`（它下面的标签一并去掉，nerve 也一并删除）；store 没有这个标签时移动和删除不问 nerve、直接失败。
  - 使用方：标签的设置页和工作项的标签选择器（Task 7 已清零）改调新签名：`createLabel(projectId, data)`、`updateLabel(labelId, data)`、`deleteLabel(labelId)`；被拒绝的移动和"移出分组"提示失败（原来什么都不提示）；取标签的使用方只传项目的 id。`@nerve/utils` 的 `array.ts` 两处 `[...options].sort` 改为 `toSorted`（`utils` 的上限 9 → 7）。项目包装层取标签改调新签名（过渡：仍是原来的 `useSWR`，Task 10 换掉）。
  - `.oxlintrc.json`：`label.store.ts`、`components/labels/**` 加进 `no-restricted-imports` 的范围；`labels.service.ts`、`label.store.ts` 和它的测试加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest）`core/store/label.store.test.ts`：
- `LabelStore, the labels`：`keeps a project's labels by sort order, then id, and the labels under each label at the top`；`keeps each project's labels apart, and none of a project left`；`fails when nerve refuses them, keeping none, and again, keeping the labels it had`；`keeps the labels it had, gives nothing and does not fail, when the session changes as it fetches again`。
- `LabelStore, the changes`：`adds a created label as nerve answers it: last at the top, or under the parent the page names`；`changes a label as nerve answers it, and moves one under a parent to the top with a null parent`；`moves a label dropped %s, as nerve answers it`（`it.each` 5 行：父标签下的第一个之前、顶层的两个之间、顶层标签下的最后、列表的末尾、没有同级的父标签下）；`sends nothing for a label dropped under its own parent on no label`；`deletes a label and the labels under it`；`fails, changing nothing, when nerve refuses $change`（4 行）；`fails, asking nerve nothing, for $change of a label it does not have`（2 行）；`sends each change once nerve has answered the one before it, a move's place reckoned from that answer`；`fetches the labels while a change is out: a fetch does not wait for it`。
- `LabelStore, while a fetch is out`：`shows once a label created during a refetch whose list has it, and the changes confirmed meanwhile`；`lets each project's newer fetch write: an older one answering last writes nothing`。

- [ ] **Step 1: 机械步骤：Plane 的 `IIssueLabel` 换成生成的 `Label`**

`$P8BTMP/rename_types.py` 见"一次性脚本"。22 个文件；其中 11 个在本 Task 另有手改（它们的块写在这一步之后的文件上），其余 11 个只经这一步改到。改名之后 oxfmt 重排这些文件：较短的名字可以放进原来折行的一行。

Run（机械步骤）: `python3 $P8BTMP/rename_types.py . IIssueLabel Label web/apps/web/core/components/ui/labels-list.tsx web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx web/apps/web/core/components/labels/project-setting-label-group.tsx web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx web/apps/web/core/components/labels/project-setting-label-item.tsx web/apps/web/core/components/labels/create-update-label-inline.tsx web/apps/web/core/components/labels/label-utils.ts web/apps/web/core/components/labels/delete-label-modal.tsx web/apps/web/core/components/labels/label-block/label-item-block.tsx web/apps/web/core/components/labels/project-setting-label-list.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx web/apps/web/core/components/power-k/menus/labels.tsx web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx web/apps/web/core/components/issues/issue-detail/label/root.tsx web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx web/apps/web/core/components/issues/select/base.tsx web/apps/web/core/components/issues/select/dropdown.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/packages/utils/src/work-item-filters/configs/filters/label.ts`
Expected: `IIssueLabel -> Label in 22 files`。

Run（机械步骤）: `pnpm -C web/apps/web exec oxfmt core/components/ui/labels-list.tsx core/components/inbox/inbox-filter/filters/labels.tsx core/components/labels/project-setting-label-group.tsx core/components/labels/label-drag-n-drop-HOC.tsx core/components/labels/project-setting-label-item.tsx core/components/labels/create-update-label-inline.tsx core/components/labels/label-utils.ts core/components/labels/delete-label-modal.tsx core/components/labels/label-block/label-item-block.tsx core/components/labels/project-setting-label-list.tsx core/components/power-k/ui/pages/context-based/work-item/commands.ts core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx core/components/power-k/menus/labels.tsx core/components/issues/issue-layouts/properties/labels.tsx core/components/issues/issue-layouts/properties/label-dropdown.tsx core/components/issues/issue-detail/label/root.tsx core/components/issues/issue-detail/label/select/label-select.tsx core/components/issues/select/base.tsx core/components/issues/select/dropdown.tsx core/hooks/work-item-filters/use-work-item-filters-config.tsx core/store/issue/root.store.ts`
Expected: 退出码 0。

Run（机械步骤）: `pnpm -C web/packages/utils exec oxfmt src/work-item-filters/configs/filters/label.ts`
Expected: 退出码 0。

`git diff --numstat` 必须恰好是：

```text
2	2	web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx
4	3	web/apps/web/core/components/issues/issue-detail/label/root.tsx
2	2	web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
2	2	web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
4	4	web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx
3	3	web/apps/web/core/components/issues/select/base.tsx
2	2	web/apps/web/core/components/issues/select/dropdown.tsx
9	9	web/apps/web/core/components/labels/create-update-label-inline.tsx
2	2	web/apps/web/core/components/labels/delete-label-modal.tsx
4	4	web/apps/web/core/components/labels/label-block/label-item-block.tsx
4	3	web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
3	2	web/apps/web/core/components/labels/label-utils.ts
4	4	web/apps/web/core/components/labels/project-setting-label-group.tsx
3	3	web/apps/web/core/components/labels/project-setting-label-item.tsx
4	4	web/apps/web/core/components/labels/project-setting-label-list.tsx
3	3	web/apps/web/core/components/power-k/menus/labels.tsx
3	2	web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
3	2	web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx
2	2	web/apps/web/core/components/ui/labels-list.tsx
4	6	web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
4	4	web/apps/web/core/store/issue/root.store.ts
4	3	web/packages/utils/src/work-item-filters/configs/filters/label.ts
```

Run（机械步骤）: `shasum -a 256 web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx web/apps/web/core/components/issues/issue-detail/label/root.tsx web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx web/apps/web/core/components/issues/select/base.tsx web/apps/web/core/components/issues/select/dropdown.tsx web/apps/web/core/components/labels/create-update-label-inline.tsx web/apps/web/core/components/labels/delete-label-modal.tsx web/apps/web/core/components/labels/label-block/label-item-block.tsx web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx web/apps/web/core/components/labels/label-utils.ts web/apps/web/core/components/labels/project-setting-label-group.tsx web/apps/web/core/components/labels/project-setting-label-item.tsx web/apps/web/core/components/labels/project-setting-label-list.tsx web/apps/web/core/components/power-k/menus/labels.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx web/apps/web/core/components/ui/labels-list.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/issue/root.store.ts web/packages/utils/src/work-item-filters/configs/filters/label.ts`
Expected: 每个文件的散列和行数与下表相同：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ea2068a0e316eecca819935b888821fb1a4a122bc33afd1df9848e078c340a15` | 94 | `web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx` |
| `acc5fdb4c96a2062d00494810348f571de2e8d98f4bc13e6d8cd3d8592607dc2` | 111 | `web/apps/web/core/components/issues/issue-detail/label/root.tsx` |
| `345c185d64ca11f6c5bc56b08a887261ab07ef281b1567e1c1f9f0dacb64a5f0` | 224 | `web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx` |
| `5fda45f0fa66a36f6fb37806cef3043932810179320651968611ad2240d12d8e` | 338 | `web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx` |
| `ee60950fcd81f1ac4a6445d96ae81006a78e8280a49be622fdb3e68c3f25bcce` | 259 | `web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx` |
| `b7f70f78ca6e0a47f32883a43f08f559e0800fa2df72d56720cd360f60101b85` | 324 | `web/apps/web/core/components/issues/select/base.tsx` |
| `a5ea8651b29ec559dcecd0b891c0c2ecb133cbb0ff20a1e504f0cfdb796a4fb0` | 58 | `web/apps/web/core/components/issues/select/dropdown.tsx` |
| `32af67d7e30dbddbaec2eadc2d234bd3b76b2d87f3d6ec817d4e6416d926a291` | 252 | `web/apps/web/core/components/labels/create-update-label-inline.tsx` |
| `8e05d7ad1aa997e7998d0b607f6368eb7251db44b30b00f48dc970de9ee3db45` | 71 | `web/apps/web/core/components/labels/delete-label-modal.tsx` |
| `9a4ddd1d1c37f4a686445883a0e868016b193d30cf87f812cedd8ae1cdc78416` | 112 | `web/apps/web/core/components/labels/label-block/label-item-block.tsx` |
| `d75e48c5c84f1a31dbfb19c90f31cdcf8273d7f63da3cab575fa4a945f0bc1c0` | 174 | `web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx` |
| `c5f509133a63f2cc13d48b9302b94c1d146cf9dab3818e6b4a8fb5a819dfaacb` | 74 | `web/apps/web/core/components/labels/label-utils.ts` |
| `06ea22933ad586a2837135dc555ff9e2ebd8a430c458cdca9a87d6bf0f2a937e` | 167 | `web/apps/web/core/components/labels/project-setting-label-group.tsx` |
| `45e28470f339fcda245344a167032db6466014ac37d5304eaf3e88eff16e2b66` | 124 | `web/apps/web/core/components/labels/project-setting-label-item.tsx` |
| `216f2b1e377ac6a5c5ceaf09dcc68de4c17d33612700b3ac202779516ce2b9a4` | 166 | `web/apps/web/core/components/labels/project-setting-label-list.tsx` |
| `facb8753b10ab0cd96295d7c3fb985ed88bbec876e2e26371f2113a8301a60b9` | 37 | `web/apps/web/core/components/power-k/menus/labels.tsx` |
| `ca8e1086e71ecc42db5423a7d537cf7397cc1fc4456cef5fcbf598e4d83bd2cd` | 415 | `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts` |
| `52da5497369fa34ee360d375e4d28c70bafad357845bac8fefb72a192b002f1a` | 34 | `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx` |
| `66ab1381d8155aec73575916d3eb111e568431bc761f011cf1980fce27fd28c7` | 37 | `web/apps/web/core/components/ui/labels-list.tsx` |
| `8ee9fb569ce525a1d731bc2270828c7b2331ab0ac7ce86ff4074004e89f107d2` | 394 | `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx` |
| `461f136984b522db2aee6b6fffcbe01b2511eb01b104bb829d1c7b107f7612f0` | 212 | `web/apps/web/core/store/issue/root.store.ts` |
| `213164821196ab064c0b1306c431f8dcb29e3fb1a360776fa71f97fe8cc5497c` | 65 | `web/packages/utils/src/work-item-filters/configs/filters/label.ts` |

- [ ] **Step 2: service 和 store**

`web/apps/web/core/services/issue/index.ts`（修改，1 处）：

````old web/apps/web/core/services/issue/index.ts
export * from "./issue_reaction.service";
export * from "./issue_label.service";
````
````new web/apps/web/core/services/issue/index.ts
export * from "./issue_reaction.service";
````

`web/apps/web/core/services/issue/issue_label.service.ts`（删除）：

````delete web/apps/web/core/services/issue/issue_label.service.ts
````

`web/apps/web/core/services/project/labels.service.ts`（新文件，42 行）：

````file web/apps/web/core/services/project/labels.service.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** The labels of a project's work items, in two levels: at the top, or under a label at the top (M3 design 3.16). */
export class LabelsService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's labels, those at the top and those under them alike, by sort order, then id; an archived one's too. */
  async list(projectId: string): Promise<Label[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/labels", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** Creates a label after the project's others; data names a parent only for a label under one. */
  async create(projectId: string, data: LabelCreate): Promise<Label> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/labels", {
        params: { path: { project_id: projectId } },
        body: data,
      })
    );
  }

  /** Changes the fields data names (a null parent moves the label to the top); the answer is the label as nerve holds it. */
  async update(labelId: string, data: LabelUpdate): Promise<Label> {
    return unwrap(
      await this.api.PATCH("/api/v0/labels/{label_id}", { params: { path: { label_id: labelId } }, body: data })
    );
  }

  /** Deletes a label and the labels under it. */
  async delete(labelId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/labels/{label_id}", { params: { path: { label_id: labelId } } }));
  }
}
````

`web/apps/web/core/store/label.store.test.ts`（新文件，362 行）：

````file web/apps/web/core/store/label.store.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { LabelStore } from "@/store/label.store";
import { ProjectRootStore } from "@/store/project";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// The labels of the caller's projects (M3 design 3.16, 7.3), against a fake nerve that answers each request when the
// test says. The tab's address is web's, a project of acme; ops is acme's other.

const acme = workspaceOf("acme");
const web = projectOf("WEB", acme.id);
const ops = projectOf("OPS", acme.id);
const LIST = `/api/v0/projects/${web.id}/labels`;

/** A label of a project (web unless it says) as nerve gives it: the name names it in its project. */
function labelOf(name: string, sortOrder: number, fields: Partial<Label> = {}, project = web): Label {
  return {
    id: `l-${project.identifier}-${name}`,
    workspace_id: project.workspace_id,
    project_id: project.id,
    parent_id: null,
    name,
    color: "#F59E0B",
    sort_order: sortOrder,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}
const bug = labelOf("bug", 65535);
const feature = labelOf("feature", 75535);
const frontend = labelOf("frontend", 80000, { parent_id: feature.id });
const backend = labelOf("backend", 85000, { parent_id: feature.id });
/** web's labels as nerve lists them, by sort order: two at the top, two under feature. */
const listed = [bug, feature, frontend, backend];
const opsBug = labelOf("bug", 65535, {}, ops);
/** A label created in web, as the page sends it and as nerve answers it: last, at the top. */
const docs: LabelCreate = { name: "docs", color: "#3F76FF" };
const created = labelOf("docs", 95000, { color: "#3F76FF" });
/** nerve's answer to a change: the fields given, and a change the request does not make, its updated_at. */
const changed = (label: Label, fields: Partial<Label>): Label => ({
  ...label,
  ...fields,
  updated_at: "2026-10-08T09:00:00Z",
});
const ids = (labels: Label[]) => labels.map((label) => label.id);
const at = (label: Label) => `/api/v0/labels/${label.id}`;

/** The store of a tab at web's address, whose caller's acme nerve listed with its web and ops. */
async function labelStore() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const router = new RouterStore();
  router.setQuery({ workspaceSlug: "acme", projectId: web.id });
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router }), api);
  const projectRoot = new ProjectRootStore(fakeRoot({ router, workspaceRoot }), api);
  await loadWorkspaces(nerve, workspaceRoot, [acme]);
  await loadProjects(nerve, projectRoot.project, acme, [web, ops]);
  nerve.calls.length = 0;
  const store = new LabelStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  return { nerve, api, projects: projectRoot.project, store };
}

/** The store fetches web's labels, and nerve lists these. */
const load = (nerve: FakeNerve, store: LabelStore, labels: Label[]) =>
  answered(nerve, () => store.fetchProjectLabels(web.id), ["GET", LIST], { data: labels }, "the labels");

/** A store whose labels of web nerve listed. */
async function loaded() {
  const tab = await labelStore();
  await load(tab.nerve, tab.store, listed);
  return tab;
}

/** Sends a change, which nerve gives reply once it is the request out; gives how the change settled. */
async function sent(nerve: FakeNerve, send: () => Promise<unknown>, request: Endpoint, reply: Response) {
  const k = nerve.calls.length;
  const change = track(send());
  await inTurn(nerve, k, request, reply);
  await until(() => change.settled, "the change");
  return change;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("LabelStore, the labels", () => {
  it("keeps a project's labels by sort order, then id, and the labels under each label at the top", async () => {
    const { nerve, store } = await labelStore();
    const tied = labelOf("api", 85000, { parent_id: feature.id });
    const fetched = await load(nerve, store, [backend, frontend, bug, tied, feature]);
    expect(fetched.value).toEqual([backend, frontend, bug, tied, feature]);
    const ordered = [bug, feature, frontend, tied, backend];
    expect(store.getProjectLabels(web.id)).toEqual(ordered);
    expect(store.projectLabels).toEqual(ordered);
    expect(store.getProjectLabelIds(web.id)).toEqual(ids(ordered));
    expect(store.projectLabelsTree).toEqual([
      { ...bug, children: [] },
      { ...feature, children: [frontend, tied, backend] },
    ]);
    expect(store.getLabelById(frontend.id)).toEqual(frontend);
    expect(Object.keys(store.labelMap)).toEqual(ids([backend, frontend, bug, tied, feature]));
    expect(store.getProjectLabels(ops.id)).toBeUndefined();
  });

  it("keeps each project's labels apart, and none of a project left", async () => {
    const { nerve, projects, store } = await loaded();
    await answered(
      nerve,
      () => store.fetchProjectLabels(ops.id),
      ["GET", `/api/v0/projects/${ops.id}/labels`],
      {
        data: [opsBug],
      },
      "ops's labels"
    );
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
    expect(store.getProjectLabels(web.id)).toEqual(listed);

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectLabels(ops.id)).toBeUndefined();
    expect(store.getLabelById(opsBug.id)).toBeUndefined();
    expect(Object.keys(store.labelMap)).toEqual(ids(listed));
  });

  it("fails when nerve refuses them, keeping none, and again, keeping the labels it had", async () => {
    const { nerve, store } = await labelStore();
    const refused = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 1, "the labels");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectLabels(web.id)).toBeUndefined();

    await load(nerve, store, listed);
    const again = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });

  it("keeps the labels it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjectLabels(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });
});

describe("LabelStore, the changes", () => {
  it("adds a created label as nerve answers it: last at the top, or under the parent the page names", async () => {
    const { nerve, store } = await loaded();
    const made = await sent(nerve, () => store.createLabel(web.id, docs), ["POST", LIST], json(201, created));
    expect(nerve.calls[1]?.body).toEqual(docs);
    expect(made.value).toEqual(created);
    expect(store.getProjectLabels(web.id)).toEqual([...listed, created]);

    const under: LabelCreate = { name: "infra", parent_id: feature.id };
    const infra = labelOf("infra", 105000, { parent_id: feature.id, color: "" });
    await sent(nerve, () => store.createLabel(web.id, under), ["POST", LIST], json(201, infra));
    expect(nerve.calls[2]?.body).toEqual(under);
    expect(store.projectLabelsTree?.find((top) => top.id === feature.id)?.children).toEqual([frontend, backend, infra]);
  });

  it("changes a label as nerve answers it, and moves one under a parent to the top with a null parent", async () => {
    const { nerve, store } = await loaded();
    const renamed = changed(bug, { name: "defect" });
    await sent(nerve, () => store.updateLabel(bug.id, { name: "defect" }), ["PATCH", at(bug)], json(200, renamed));
    expect(nerve.calls[1]?.body).toEqual({ name: "defect" });
    expect(store.getLabelById(bug.id)).toEqual(renamed);

    const top = changed(frontend, { parent_id: null });
    const freed = track(store.updateLabel(frontend.id, { parent_id: null }));
    await until(() => nerve.calls.length === 3, "the move");
    expect(nerve.calls[2]).toMatchObject({ method: "PATCH", path: at(frontend), body: { parent_id: null } });
    expect(store.getLabelById(frontend.id)).toEqual(frontend);
    nerve.calls[2]?.answer(json(200, top));
    await until(() => freed.settled, "the answer");
    expect(store.projectLabelsTree).toEqual([
      { ...renamed, children: [] },
      { ...feature, children: [backend] },
      { ...top, children: [] },
    ]);
  });

  // a drop: what it is, the label dropped, updateLabelPosition's arguments after the label, and what it sends nerve
  const drops: [string, Label, [string | null, string | undefined, boolean], LabelUpdate][] = [
    [
      "before a parent's first",
      backend,
      [feature.id, frontend.id, false],
      { parent_id: feature.id, sort_order: 40000 },
    ],
    ["between two at the top", frontend, [null, feature.id, false], { parent_id: null, sort_order: 70535 }],
    ["last under a top label", bug, [feature.id, undefined, false], { parent_id: feature.id, sort_order: 95000 }],
    ["at the list's end, at the top", backend, [null, bug.id, true], { parent_id: null, sort_order: 85535 }],
    ["under a top label with none: the parent alone", frontend, [bug.id, undefined, false], { parent_id: bug.id }],
  ];
  it.each(drops)("moves a label dropped %s, as nerve answers it", async (_drop, label, to, body) => {
    const { nerve, store } = await loaded();
    const moved = changed(label, body);
    const sentMove = await sent(
      nerve,
      () => store.updateLabelPosition(label.id, ...to),
      ["PATCH", at(label)],
      json(200, moved)
    );
    expect(nerve.calls[1]?.body).toEqual(body);
    expect(sentMove.value).toEqual(moved);
    expect(store.getLabelById(label.id)).toEqual(moved);
  });

  it("sends nothing for a label dropped under its own parent on no label", async () => {
    const { nerve, store } = await loaded();
    const kept = await settle(store.updateLabelPosition(frontend.id, feature.id, undefined, false), "the drop");
    expect(kept).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
  });

  it("deletes a label and the labels under it", async () => {
    const { nerve, store } = await loaded();
    const deleted = await sent(nerve, () => store.deleteLabel(feature.id), ["DELETE", at(feature)], noContent());
    expect(deleted.error).toBeUndefined();
    expect(store.getProjectLabels(web.id)).toEqual([bug]);
    expect(store.getLabelById(frontend.id)).toBeUndefined();
  });

  const refusals: { change: string; send: (store: LabelStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createLabel(web.id, { name: "BUG" }),
      refusal: problem(409, "project.label_name_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateLabel(feature.id, { parent_id: bug.id }),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a move",
      send: (store) => store.updateLabelPosition(feature.id, bug.id, undefined, false),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a deletion",
      send: (store) => store.deleteLabel(bug.id),
      refusal: problem(404, "project.label_not_found"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });

  it.each(refusals.slice(2))(
    "fails, asking nerve nothing, for $change of a label it does not have",
    async ({ send }) => {
      const { nerve, store } = await labelStore();
      const refused = await settle(send(store), "the change");
      expect(refused).toMatchObject({ settled: true, error: new Error("Label not found") });
      expect(nerve.calls).toEqual([]);
    }
  );

  it("sends each change once nerve has answered the one before it, a move's place reckoned from that answer", async () => {
    const { nerve, store } = await loaded();
    const refused = track(store.updateLabel(bug.id, { name: "defect" }));
    const first = track(store.updateLabelPosition(backend.id, feature.id, frontend.id, false));
    const second = track(store.updateLabelPosition(bug.id, feature.id, frontend.id, false));
    await inTurn(nerve, 1, ["PATCH", at(bug)], problem(503, "server_busy"));
    const backendFirst = changed(backend, { sort_order: 40000 });
    await inTurn(nerve, 2, ["PATCH", at(backend)], json(200, backendFirst));
    // frontend's neighbour before it is now backend, as nerve answered the first move
    const bugBetween = changed(bug, { parent_id: feature.id, sort_order: 60000 });
    await inTurn(nerve, 3, ["PATCH", at(bug)], json(200, bugBetween));
    await until(() => second.settled, "the last change");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(first.value).toEqual(backendFirst);
    expect(nerve.calls[3]?.body).toEqual({ parent_id: feature.id, sort_order: 60000 });
    expect(store.projectLabelsTree?.[0]).toEqual({ ...feature, children: [backendFirst, bugBetween, frontend] });
  });

  it("fetches the labels while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateLabel(bug.id, { name: "defect" }), request: ["PATCH", at(bug)] },
      { send: () => store.fetchProjectLabels(web.id), request: ["GET", LIST], body: { data: [bug, feature] } }
    );
    expect(store.getProjectLabels(web.id)).toEqual([bug, feature]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one project's labels, only the newer writes.
describe("LabelStore, while a fetch is out", () => {
  it("shows once a label created during a refetch whose list has it, and the changes confirmed meanwhile", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(nerve, () => store.createLabel(web.id, docs), ["POST", LIST], json(201, created));
    const renamed = changed(bug, { name: "defect" });
    await sent(nerve, () => store.updateLabel(bug.id, { name: "defect" }), ["PATCH", at(bug)], json(200, renamed));
    await sent(nerve, () => store.deleteLabel(feature.id), ["DELETE", at(feature)], noContent());
    // read after the creation, before the change and the deletion
    nerve.calls[1]?.answer(json(200, { data: [...listed, created] }));
    await until(() => refetched.settled, "the refetch");

    expect(store.getProjectLabels(web.id)).toEqual([renamed, created]);
    expect(refetched.value && ids(refetched.value)).toEqual(ids([renamed, created]));
  });

  it("lets each project's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await labelStore();
    const older = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 1, "the older labels");
    const opsLabels = track(store.fetchProjectLabels(ops.id));
    await until(() => nerve.calls.length === 2, "ops's labels");
    const newer = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 3, "the newer labels");
    nerve.calls[2]?.answer(json(200, { data: [bug, feature] }));
    await until(() => newer.settled, "the newer labels");
    // a newer fetch of web's labels does not overtake one of ops's
    nerve.calls[1]?.answer(json(200, { data: [opsBug] }));
    await until(() => opsLabels.settled, "ops's labels");
    // read before frontend and backend were made
    nerve.calls[0]?.answer(json(200, { data: listed }));
    await until(() => older.settled, "the older labels");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getProjectLabels(web.id)).toEqual([bug, feature]);
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
  });
});
````

`web/apps/web/core/store/label.store.ts`（整个文件，222 行）：

````whole web/apps/web/core/store/label.store.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, computed, makeObservable } from "mobx";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Label, LabelCreate, LabelUpdate, Project } from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change } from "@/lib/reconciled";
import { ReconciledByKey, replaced, upserted } from "@/lib/reconciled";
// services
import { LabelsService } from "@/services/project/labels.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IRouterStore } from "@/store/router.store";

/** A label at the top with the labels under it, in their project's order: labels have two levels (M3 design 3.16). */
type LabelTree = Label & { children: Label[] };

export interface ILabelStore {
  // computed
  /** Every label the store shows, by id, which the work items' stores look up. */
  labelMap: Record<string, Label>;
  /** The address's project's labels, by sort order, then id; undefined until fetched. */
  projectLabels: Label[] | undefined;
  /** The address's project's labels at the top, each with the labels under it; undefined until fetched. */
  projectLabelsTree: LabelTree[] | undefined;
  // computed actions
  getLabelById: (labelId: string | null | undefined) => Label | undefined;
  getProjectLabels: (projectId: string | null | undefined) => Label[] | undefined;
  getProjectLabelIds: (projectId: string | null | undefined) => string[] | undefined;
  // fetch actions
  fetchProjectLabels: (projectId: string) => Promise<Label[] | undefined>;
  // changes
  createLabel: (projectId: string, data: LabelCreate) => Promise<Label>;
  updateLabel: (labelId: string, data: LabelUpdate) => Promise<Label>;
  updateLabelPosition: (
    labelId: string,
    parentId: string | null,
    droppedOnId: string | undefined,
    dropAtEndOfList: boolean
  ) => Promise<Label | undefined>;
  deleteLabel: (labelId: string) => Promise<void>;
}

/** nerve's order of a project's labels: by sort order, the lowest first, then by id. */
const inOrder = (labels: Label[]): Label[] =>
  labels.toSorted((a, b) => a.sort_order - b.sort_order || Number(a.id > b.id) - Number(a.id < b.id));

/** A label at the top, a copy, with the labels of the list under it, in the list's order. */
const treeOf = (top: Label, labels: Label[]): LabelTree => ({
  ...top,
  children: labels.filter((label) => label.parent_id === top.id),
});

/**
 * The sort order that puts a label at index among siblings (the labels under its new parent, in order): halfway
 * between its neighbours there, half the first's before the first, 10000 past the last; none among no siblings,
 * where the label takes the parent alone.
 */
const sortOrderAt = (siblings: Label[], index: number): number | undefined => {
  const before = index > 0 ? siblings[index - 1].sort_order : undefined;
  const after = index < siblings.length ? siblings[index].sort_order : undefined;
  if (before === undefined) return after === undefined ? undefined : after / 2;
  return after === undefined ? before + 10000 : (before + after) / 2;
};

/** The list without the label id names and the labels under it, as nerve deletes them. */
const deleted =
  (id: string): Change<Label[]> =>
  (list) =>
    list.filter((label) => label.id !== id && label.parent_id !== id);

/**
 * The labels of the projects of a session (M3 design 3.16, 7.3): each project's list, by its id, which its pages
 * fetch; labels are a project's only, the workspace's across its projects are M7's. Their order is nerve's (sort
 * order, then id), and the labels under a label at the top come from parent_id. The labels of a project the project
 * store no longer gives (deleted, left, or of a workspace no longer the caller's) do not show. Its service sends with
 * the session's client; changes go one at a time and the store writes nerve's answers (v0 design 7.7); fetches do
 * not queue.
 */
export class LabelStore implements ILabelStore {
  /** Each project's labels, reconciled between fetches and changes (reconciled.ts). */
  private readonly projects = new ReconciledByKey<Label[]>();
  // services
  private readonly service: LabelsService;
  /** The changes of the labels, sent one at a time. */
  private readonly changes = oneAtATime();
  // stores
  private readonly router: IRouterStore;
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;

  constructor(_rootStore: RootStore, api: ApiClient) {
    makeObservable(this, {
      // computed
      labelMap: computed,
      projectLabels: computed,
      projectLabelsTree: computed,
      // actions
      fetchProjectLabels: action,
      createLabel: action,
      updateLabel: action,
      updateLabelPosition: action,
      deleteLabel: action,
    });
    this.service = new LabelsService(api);
    this.router = _rootStore.router;
    this.projectOf = _rootStore.projectRoot.project.getProjectById;
  }

  get labelMap() {
    const shown = this.projects
      .values()
      .flat()
      .filter((label) => this.projectOf(label.project_id));
    return Object.fromEntries(shown.map((label) => [label.id, label]));
  }

  get projectLabels() {
    return this.getProjectLabels(this.router.projectId);
  }

  get projectLabelsTree() {
    const labels = this.projectLabels;
    return labels?.filter((label) => label.parent_id === null).map((top) => treeOf(top, labels));
  }

  /** @description the label, of a project the caller sees, as the store last had it from nerve */
  getLabelById = computedFn((labelId: string | null | undefined): Label | undefined =>
    labelId ? this.labelMap[labelId] : undefined
  );

  /**
   * @description the project's labels, by sort order, then id; undefined until fetched, or once the project store
   * no longer gives the project
   */
  getProjectLabels = computedFn((projectId: string | null | undefined): Label[] | undefined => {
    const project = this.projectOf(projectId);
    const labels = project && this.projects.get(project.id);
    return labels && inOrder(labels);
  });

  getProjectLabelIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.getProjectLabels(projectId)?.map((label) => label.id)
  );

  /**
   * @description fetches a project's labels, a member's to fetch as nerve refuses anyone else, and shows them with
   * the changes nerve confirmed meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a
   * change of session cut (Reconciled.fetch)
   */
  fetchProjectLabels = (projectId: string): Promise<Label[] | undefined> =>
    this.projects.fetch(projectId, () => this.service.list(projectId));

  /**
   * @description creates a label, last of its project's: at the top unless data names a parent; fails, changing
   * nothing, when nerve refuses
   */
  createLabel = (projectId: string, data: LabelCreate): Promise<Label> =>
    this.changes(async () => {
      const label = await this.service.create(projectId, data);
      this.projects.confirm(label.project_id, upserted(label));
      return label;
    });

  /** @description changes a label; the store then shows nerve's answer. Fails, changing nothing, when nerve refuses. */
  updateLabel = (labelId: string, data: LabelUpdate): Promise<Label> => this.changes(() => this.send(labelId, data));

  /**
   * @description moves a label where it was dropped: under parentId (null for the top), before the label droppedOnId
   * names, or last there for none or at the end of the list. Its place is reckoned when the change goes out, from
   * the labels as nerve last answered them; dropped under its own parent on no label, it stays and nothing is sent.
   * Fails, changing nothing, when nerve refuses or the store does not have it.
   */
  updateLabelPosition = (
    labelId: string,
    parentId: string | null,
    droppedOnId: string | undefined,
    dropAtEndOfList: boolean
  ): Promise<Label | undefined> =>
    this.changes(async () => {
      const label = this.held(labelId);
      if (label.parent_id === parentId && !droppedOnId) return undefined;
      const siblings = (this.getProjectLabels(label.project_id) ?? []).filter((held) => held.parent_id === parentId);
      const droppedOn = siblings.findIndex((held) => held.id === droppedOnId);
      const sortOrder = sortOrderAt(siblings, dropAtEndOfList || droppedOn === -1 ? siblings.length : droppedOn);
      return this.send(
        label.id,
        sortOrder === undefined ? { parent_id: parentId } : { parent_id: parentId, sort_order: sortOrder }
      );
    });

  /**
   * @description deletes a label and the labels under it; fails, changing nothing, when nerve refuses or the store
   * does not have it
   */
  deleteLabel = (labelId: string): Promise<void> =>
    this.changes(async () => {
      const label = this.held(labelId);
      await this.service.delete(label.id);
      this.projects.confirm(label.project_id, deleted(label.id));
    });

  /** Sends a change of a label, its turn come, and makes nerve's answer on its project's list. */
  private async send(labelId: string, data: LabelUpdate): Promise<Label> {
    const label = await this.service.update(labelId, data);
    this.projects.confirm(label.project_id, replaced(label));
    return label;
  }

  /** The label as the store has it; fails when it has none. */
  private held(labelId: string): Label {
    const label = this.getLabelById(labelId);
    if (!label) throw new Error("Label not found");
    return label;
  }
}
````

`web/apps/web/core/store/root.store.ts`（修改，1 处）：

````old web/apps/web/core/store/root.store.ts
    this.label = new LabelStore(this);
````
````new web/apps/web/core/store/root.store.ts
    this.label = new LabelStore(this, api);
````

`web/packages/types/src/issues.ts`（修改，1 处）：

````old web/packages/types/src/issues.ts
  url: string;
}

export interface IIssueLabel {
  id: string;
  name: string;
  color: string;
  project_id: string;
  workspace_id: string;
  parent: string | null;
  sort_order: number;
}

export interface IIssueLabelTree extends IIssueLabel {
  children: IIssueLabel[] | undefined;
````
````new web/packages/types/src/issues.ts
  url: string;
````

`web/packages/utils/package.json`（修改，1 处）：

````old web/packages/utils/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 9",
````
````new web/packages/utils/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 7",
````

`web/packages/utils/src/array.ts`（修改，3 处）：

````old web/packages/utils/src/array.ts
import { isEmpty } from "lodash-es";
import type { IIssueLabel, IIssueLabelTree } from "@nerve/types";

/**
 * @description Builds a tree structure from an array of labels
 * @param {IIssueLabel[]} array Array of labels
 * @param {any} parent Parent ID
 * @returns {IIssueLabelTree[]} Tree structure
 */
export const buildTree = (array: IIssueLabel[], parent = null) => {
  const tree: IIssueLabelTree[] = [];

  array.forEach((item: any) => {
    if (item.parent === parent) {
      const children = buildTree(array, item.id);
      item.children = children;
      tree.push(item);
    }
  });

  return tree;
};
````
````new web/packages/utils/src/array.ts
import { isEmpty } from "lodash-es";
````
````old web/packages/utils/src/array.ts

  // Create a shallow copy to avoid mutating the original array
  return [...options].sort((a, b) => {
    const aSelected = a.value !== null && selectedSet.has(a.value);
````
````new web/packages/utils/src/array.ts

  return options.toSorted((a, b) => {
    const aSelected = a.value !== null && selectedSet.has(a.value);
````
````old web/packages/utils/src/array.ts

  // Create a shallow copy to avoid mutating the original array
  return [...options].sort((a, b) => {
    const aIsCurrent = currentUserId && a.value === currentUserId;
````
````new web/packages/utils/src/array.ts

  return options.toSorted((a, b) => {
    const aIsCurrent = currentUserId && a.value === currentUserId;
````

- [ ] **Step 3: 使用方和静态检查**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx",
        "web/apps/web/core/store/label.store.ts",
        "web/apps/web/core/components/labels/**"
````
````old .oxlintrc.json
        "web/apps/web/core/store/state.store.test.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/store/state.store.test.ts",
        "web/apps/web/core/services/project/labels.service.ts",
        "web/apps/web/core/store/label.store.ts",
        "web/apps/web/core/store/label.store.test.ts",
````

`web/apps/web/core/components/issues/issue-detail/label/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
import type { Label, LabelCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
  createLabel: (data: Partial<Label>) => Promise<Label>;
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
  createLabel: (data: LabelCreate) => Promise<Label>;
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
      createLabel: async (data: Partial<Label>) => {
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
      createLabel: async (data: LabelCreate) => {
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
          const labelResponse = await createLabel(workspaceSlug, projectId, data);
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
          const labelResponse = await createLabel(projectId, data);
````
````old web/apps/web/core/components/issues/issue-detail/label/root.tsx
    [updateIssue, createLabel, onLabelUpdate, isInboxIssue, workspaceSlug, projectId, t]
````
````new web/apps/web/core/components/issues/issue-detail/label/root.tsx
    [updateIssue, createLabel, onLabelUpdate, isInboxIssue, projectId, t]
````

`web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx`（修改，3 处）：

````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
import type { Label, LabelCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
  onAddLabel: (data: Partial<Label>) => Promise<Label>;
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
  onAddLabel: (data: LabelCreate) => Promise<Label>;
````
````old web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
    if (!projectLabels && workspaceSlug && projectId)
      fetchProjectLabels(workspaceSlug, projectId).then(() => setIsLoading(false));
````
````new web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx
    if (!projectLabels && projectId) fetchProjectLabels(projectId).then(() => setIsLoading(false));
````

`web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx`（修改，4 处）：

````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    if (!storeLabels && workspaceSlug && projectId)
      fetchProjectLabels(workspaceSlug, projectId)
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    if (!storeLabels && projectId)
      fetchProjectLabels(projectId)
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
  }, [storeLabels, workspaceSlug, projectId, fetchProjectLabels, setIsLoading]);
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
  }, [storeLabels, projectId, fetchProjectLabels, setIsLoading]);
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    if (!workspaceSlug || !projectId) return;
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    if (!projectId) return;
````
````old web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    const label = await createLabel(workspaceSlug, projectId, { name: labelName, color: getRandomLabelColor() });
````
````new web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx
    const label = await createLabel(projectId, { name: labelName, color: getRandomLabelColor() });
````

`web/apps/web/core/components/issues/select/base.tsx`（修改，5 处）：

````old web/apps/web/core/components/issues/select/base.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/issues/select/base.tsx
import type { Label, LabelCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/issues/select/base.tsx
  getLabelById: (labelId: string) => Label | null;
````
````new web/apps/web/core/components/issues/select/base.tsx
  getLabelById: (labelId: string) => Label | undefined;
````
````old web/apps/web/core/components/issues/select/base.tsx
  createLabel?: (data: Partial<Label>) => Promise<Label>;
````
````new web/apps/web/core/components/issues/select/base.tsx
  createLabel?: (data: LabelCreate) => Promise<Label>;
````
````old web/apps/web/core/components/issues/select/base.tsx
                      const children = labelsList?.filter((l) => l.parent === label.id);
````
````new web/apps/web/core/components/issues/select/base.tsx
                      const children = labelsList?.filter((l) => l.parent_id === label.id);
````
````old web/apps/web/core/components/issues/select/base.tsx
                        if (!label.parent)
````
````new web/apps/web/core/components/issues/select/base.tsx
                        if (!label.parent_id)
````

`web/apps/web/core/components/issues/select/dropdown.tsx`（修改，4 处）：

````old web/apps/web/core/components/issues/select/dropdown.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/issues/select/dropdown.tsx
import type { LabelCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/issues/select/dropdown.tsx
    if (projectLabelIds === undefined && workspaceSlug && projectId) fetchProjectLabels(workspaceSlug, projectId);
````
````new web/apps/web/core/components/issues/select/dropdown.tsx
    if (projectLabelIds === undefined && projectId) fetchProjectLabels(projectId);
````
````old web/apps/web/core/components/issues/select/dropdown.tsx
  const handleCreateLabel = (data: Partial<Label>) => {
    if (!workspaceSlug || !projectId) {
      throw new Error("Workspace slug or project ID is missing");
````
````new web/apps/web/core/components/issues/select/dropdown.tsx
  const handleCreateLabel = (data: LabelCreate) => {
    if (!projectId) {
      throw new Error("Project ID is missing");
````
````old web/apps/web/core/components/issues/select/dropdown.tsx
    return createLabel(workspaceSlug, projectId, data);
````
````new web/apps/web/core/components/issues/select/dropdown.tsx
    return createLabel(projectId, data);
````

`web/apps/web/core/components/labels/create-update-label-inline.tsx`（修改，7 处）：

````old web/apps/web/core/components/labels/create-update-label-inline.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
import type { Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
  createLabel: (data: Partial<Label>) => Promise<Label>;
  updateLabel: (labelId: string, data: Partial<Label>) => Promise<Label>;
};
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
  createLabel: (data: LabelCreate) => Promise<Label>;
  updateLabel: (labelId: string, data: LabelUpdate) => Promise<Label>;
};

/** What the form sends, a creation (at the top) or a change alike. */
type TLabelFormValues = Pick<Label, "name" | "color">;
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
const defaultValues: Partial<Label> = {
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
const defaultValues: TLabelFormValues = {
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    } = useForm<Label>({
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    } = useForm<TLabelFormValues>({
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    const handleLabelCreate: SubmitHandler<Label> = async (formData) => {
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    const handleLabelCreate: SubmitHandler<TLabelFormValues> = async (formData) => {
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    const handleLabelUpdate: SubmitHandler<Label> = async (formData) => {
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    const handleLabelUpdate: SubmitHandler<TLabelFormValues> = async (formData) => {
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    const handleFormSubmit = (formData: Label) => {
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    const handleFormSubmit = (formData: TLabelFormValues) => {
````

`web/apps/web/core/components/labels/delete-label-modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/labels/delete-label-modal.tsx
import { observer } from "mobx-react";
import { useParams } from "react-router";
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
import { observer } from "mobx-react";
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
  const { isOpen, onClose, data } = props;
  // router
  const { workspaceSlug, projectId } = useParams();
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
  const { isOpen, onClose, data } = props;
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
    if (!workspaceSlug || !projectId || !data) return;
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
    if (!data) return;
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
    await deleteLabel(workspaceSlug, projectId, data.id)
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
    await deleteLabel(data.id)
````

`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`（修改，2 处）：

````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
        getInitialData: () => ({ id: label?.id, parentId: label?.parent, isGroup, isChild }),
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
        getInitialData: () => ({ id: label?.id, parentId: label?.parent_id, isGroup, isChild }),
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const data = { id: label?.id, parentId: label?.parent, isGroup, isChild };
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
          const data = { id: label?.id, parentId: label?.parent_id, isGroup, isChild };
````

`web/apps/web/core/components/labels/label-utils.ts`（修改，1 处）：

````old web/apps/web/core/components/labels/label-utils.ts
  if (sourceData.id === label?.id || sourceData.id === label?.parent) return false;
````
````new web/apps/web/core/components/labels/label-utils.ts
  if (sourceData.id === label?.id || sourceData.id === label?.parent_id) return false;
````

`web/apps/web/core/components/labels/project-setting-label-item.tsx`（修改，5 处）：

````old web/apps/web/core/components/labels/project-setting-label-item.tsx
import { useState } from "react";
import { useParams } from "react-router";
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
import { useState } from "react";
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
import type { Label } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
  // router
  const { workspaceSlug, projectId } = useParams();
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
  // nerve hooks
  const { t } = useTranslation();
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
    if (!workspaceSlug || !projectId) return;

    updateLabel(workspaceSlug, projectId, label.id, {
      parent: null,
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
    updateLabel(label.id, { parent_id: null }).catch(() => {
      setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("something_went_wrong") });
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
      isVisible: !!label.parent,
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
      isVisible: label.parent_id !== null,
````

`web/apps/web/core/components/labels/project-setting-label-list.tsx`（修改，7 处）：

````old web/apps/web/core/components/labels/project-setting-label-list.tsx
import type { Label } from "@nerve/api-client";
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Label, LabelCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { workspaceSlug, projectId } = useParams();
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { projectId } = useParams();
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  if (!workspaceSlug || !projectId) return null;
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  if (!projectId) return null;
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
    createLabel: (data: Partial<Label>) => createLabel(workspaceSlug, projectId, data),
    updateLabel: (labelId: string, data: Partial<Label>) => updateLabel(workspaceSlug, projectId, labelId, data),
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
    createLabel: (data: LabelCreate) => createLabel(projectId, data),
    updateLabel,
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
    updateLabelPosition(workspaceSlug, projectId, draggingLabelId, droppedParentId, droppedLabelId, dropAtEndOfList);
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
    updateLabelPosition(draggingLabelId, droppedParentId, droppedLabelId, dropAtEndOfList).catch(() => {
      setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("something_went_wrong") });
    });
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
              if (label.children && label.children.length) {
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
              if (label.children.length > 0) {
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
                    labelChildren={label.children || []}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
                    labelChildren={label.children}
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（修改，1 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  useSWR(PROJECT_LABELS(projectId, currentProjectRole), () => fetchProjectLabels(workspaceSlug, projectId), {
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
  useSWR(PROJECT_LABELS(projectId, currentProjectRole), () => fetchProjectLabels(projectId), {
````

`web/apps/web/core/store/issue/issue-details/sub_issues.store.ts`（修改，1 处）：

````old web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        this.rootIssueDetailStore.rootIssueStore.rootStore.label.fetchProjectLabels(workspaceSlug, projectId);
````
````new web/apps/web/core/store/issue/issue-details/sub_issues.store.ts
        this.rootIssueDetailStore.rootIssueStore.rootStore.label.fetchProjectLabels(projectId);
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外，没有命中；web 的 oxlint 367 条、`utils` 7 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/inbox/inbox-filter/filters/labels.tsx web/apps/web/core/components/issues/issue-detail/label/root.tsx web/apps/web/core/components/issues/issue-detail/label/select/label-select.tsx web/apps/web/core/components/issues/issue-layouts/properties/label-dropdown.tsx web/apps/web/core/components/issues/issue-layouts/properties/labels.tsx web/apps/web/core/components/issues/select/base.tsx web/apps/web/core/components/issues/select/dropdown.tsx web/apps/web/core/components/labels/create-update-label-inline.tsx web/apps/web/core/components/labels/delete-label-modal.tsx web/apps/web/core/components/labels/label-block/label-item-block.tsx web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx web/apps/web/core/components/labels/label-utils.ts web/apps/web/core/components/labels/project-setting-label-group.tsx web/apps/web/core/components/labels/project-setting-label-item.tsx web/apps/web/core/components/labels/project-setting-label-list.tsx web/apps/web/core/components/power-k/menus/labels.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx web/apps/web/core/components/ui/labels-list.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/services/issue/index.ts web/apps/web/core/services/issue/issue_label.service.ts web/apps/web/core/services/project/labels.service.ts web/apps/web/core/store/issue/issue-details/sub_issues.store.ts web/apps/web/core/store/issue/root.store.ts web/apps/web/core/store/label.store.test.ts web/apps/web/core/store/label.store.ts web/apps/web/core/store/root.store.ts web/packages/types/src/issues.ts web/packages/utils/package.json web/packages/utils/src/array.ts web/packages/utils/src/work-item-filters/configs/filters/label.ts
```
```bash
git commit -m "feat(M3/P8b): the labels come from /api/v0, by a store of the session

The generated Label replaces IIssueLabel, 22 files of it by a mechanical
rename. LabelsService wraps the generated client, and LabelStore keeps
each project's labels by id in nerve's order, two levels from parent_id.
Changes go one at a time and take nerve's answers; a move's place is
reckoned when its turn comes, from the labels as nerve last answered
them, and a creation sends no sort order and no null parent. A refused
move now says so.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t9-lb-order` | 标签只按 id 排，不按 `sort_order` | `label.store.test.ts` | vitest |
| `t9-lb-order-tie` | `sort_order` 相同的标签按 id 从大到小排 | `label.store.test.ts` | vitest |
| `t9-lb-tree-children` | 顶层的标签下面没有标签 | `label.store.test.ts` | vitest |
| `t9-lb-tree-top` | 子标签也在顶层 | `label.store.test.ts` | vitest |
| `t9-lb-left` | 项目 store 不再给出的项目，它的标签仍能按 id 找到 | `label.store.test.ts` | vitest |
| `t9-lb-left-list` | 项目 store 不再给出的项目，它的标签列表仍给出 | `label.store.test.ts` | vitest |
| `t9-lb-create-none` | 创建的标签不存 | `label.store.test.ts` | vitest |
| `t9-lb-update-none` | 标签的修改不写入 | `label.store.test.ts` | vitest |
| `t9-lb-place-middle` | 放在两个标签之间的标签紧跟在第一个之后 | `label.store.test.ts` | vitest |
| `t9-lb-place-first` | 放在第一个之前的标签紧挨在它之前 | `label.store.test.ts` | vitest |
| `t9-lb-place-last` | 放在最后的标签紧跟在最后一个之后 | `label.store.test.ts` | vitest |
| `t9-lb-place-none` | 放进没有子标签的父标签下的标签也发出 `sort_order` | `label.store.test.ts` | vitest |
| `t9-lb-end-of-list` | 放在列表末尾的标签排到它放在其上的标签之前 | `label.store.test.ts` | vitest |
| `t9-lb-same-parent` | 放在原父标签下、没有放在某个标签上，也照样发出 | `label.store.test.ts` | vitest |
| `t9-lb-siblings` | 标签的位置在项目的全部标签中算，不在新的同级中算 | `label.store.test.ts` | vitest |
| `t9-lb-delete-children` | 删除的标签下面的标签仍在 | `label.store.test.ts` | vitest |
| `t9-lb-unheld` | store 没有的标签也照样发出移动或删除 | `label.store.test.ts` | vitest |
| `t9-lb-no-queue` | 标签的修改立即发出：移动的位置在前一个修改有回答之前就算出 | `label.store.test.ts` | vitest |
| `t9-lb-create-null-parent` | 在顶层创建的标签发出 `parent_id: null`（P7b） | `tsc` | 静态 |
| `t9-lb-create-sort-order` | 创建的标签发出 `sort_order`（P7b） | `tsc` | 静态 |
| `t9-lb-change-swallows` | 标签的修改被拒绝时给出空值，不失败 | `label.store.test.ts` | vitest |
| `t9-lb-fetch-swallows` | 标签列表被拒绝时给出 `undefined`，不失败 | `label.store.test.ts` | vitest |
| `t9-lb-fetch-queued` | 标签列表的取数排进修改的队列 | `label.store.test.ts` | vitest |

---

### Task 10: 项目包装层：`useProjectFetch` 做出全部判断，子资源只给成员取

**Files:**
- Create: `web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx`、`web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/hooks/store/fake-store-hooks.ts`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/packages/constants/src/fetch-keys.ts`、`web/packages/i18n/src/locales/en/empty-state.json`、`web/packages/i18n/src/locales/zh-CN/empty-state.json`

**Interfaces:**
- Produces（spec 2.10；M3 设计 3.19、7.1、7.6、8.3；Codex 设计评审 4.3 第 1 条的后一半；P8a review 第 6 节）：
  - `core/layouts/auth-layout/use-project-fetch.ts`：`useProjectFetch(projectId): ProjectAccess`，项目一侧挂载时的取数和"这个项目对调用者是什么"这唯一的判断。先取 nerve 对项目的读（`["PROJECT", projectId]`，`fetchProject`）；判断按这个顺序：读被拒绝（码是 `project.not_found` → `not-found`，其余 → `unavailable`，带重取的 `retry`；nerve 先前答过也一样）、还没有回答（`loading`）、项目 store 此刻不再给出它（删除、离开 → `not-found`）、`member_role` 为 `null`（看得到、不是成员 → `not-member`，带项目）、其余（`member`，带项目）。只在 `member` 时再取调用者在项目里的标签栏、项目的标签、成员、状态（`["PROJECT_PREFERENCES", id]`、`["PROJECT_LABELS", id]`、`["PROJECT_MEMBERS", id]`、`["PROJECT_STATES", id]`），条件由 hook 从 nerve 的读和项目 store 算出，调用方传不进来；不是成员的人一个子资源的请求都不发（7.1，W5）。
  - `core/layouts/auth-layout/project-wrapper.tsx`（整个文件）：`ProjectAuthWrapper({ projectId, children })` 只照 `useProjectFetch` 渲染：`loading` 什么都不显示；`unavailable` 显示 `SessionUnavailable`（重取）；`not-member` 显示加入的界面（`joinProject` 经项目 store 的队列）；`not-found` 显示"找不到项目"；`member` 渲染页面。Plane 按 HTTP 状态码（403、409）和工作区管理员判断的分支、"无权访问"的界面和它的文案 `project_empty_state.no_access.restricted_description`（两种语言）删除；`ProjectAccessRestriction` 收 `canJoin`。包装层原来的 `useSWR`、键 `PROJECT_DETAILS`、`PROJECT_LABELS`、`PROJECT_MEMBERS`、`PROJECT_STATES`、`PROJECT_MEMBER_PREFERENCES` 删除；三个使用它的布局不再传 `workspaceSlug`。
  - 项目设置的成员默认值不再自己取项目（包装层的读就是它，W17：少一个重复的请求）。
  - `fake-store-hooks.ts` 加项目一侧的 hook（`useProject`、`useProjectPreferences`、`useLabel`、`useMember().project`、`useProjectState`）。
  - `.oxlintrc.json`：`project-wrapper.tsx`、`use-project-fetch.ts`、`project-settings-member-defaults.tsx` 加进 `no-restricted-imports` 的范围；`use-project-fetch.ts` 和它的测试加进 `typescript/no-non-null-assertion` 的范围。

**Tests:**（vitest，`fake-session-swr.ts` 代替 `useSessionSWR`，`fake-store-hooks.ts` 代替 store 的 hook）`core/layouts/auth-layout/use-project-fetch.test.ts`：`fetches the project and, for a member, his tab bar in it, its labels, its members and its states`（键和 fetcher 的参数）；`fetches the project alone when $when`（`it.each` 4 行：看得到不是成员、nerve 还没有读而工作区的列表有它、找不到、对成员重取时取不到）；`shows the wrapper what to render when $when`（`it.each` 8 行：第一次就取不到、重取时取不到而 store 仍有它、找不到、还没有回答、store 不再给出、看得到不是成员、是访客、是成员）；`reads the project again when the page's retry asks`。

- [ ] **Step 1: 判断和取数的 hook**

`web/apps/web/core/hooks/store/fake-store-hooks.ts`（修改，7 处）：

````old web/apps/web/core/hooks/store/fake-store-hooks.ts
import type { Profile, Workspace, WorkspaceMember } from "@nerve/api-client";
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
import type { Profile, Project, Workspace, WorkspaceMember } from "@nerve/api-client";
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
  profile: Profile | undefined;
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
  profile: Profile | undefined;
  /** The projects the project store gives (getProjectById). */
  projects: Project[];
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
} = { workspaces: undefined, address: undefined, members: {}, profile: undefined, fetched: [] };
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
} = { workspaces: undefined, address: undefined, members: {}, profile: undefined, projects: [], fetched: [] };
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
  Object.assign(stores, { workspaces: undefined, address: undefined, members: {}, profile: undefined, fetched: [] });
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
  Object.assign(stores, {
    workspaces: undefined,
    address: undefined,
    members: {},
    profile: undefined,
    projects: [],
    fetched: [],
  });
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
        (stores.address === undefined ? undefined : stores.members[stores.address]?.[userId]) ?? null,
    },
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
        (stores.address === undefined ? undefined : stores.members[stores.address]?.[userId]) ?? null,
    },
    project: {
      fetchProjectMembers: (projectId: string) => fetching(`the members of ${projectId}`),
    },
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
export function useProject() {
  return {
    fetchProjects: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the projects of ${named(workspace)}`),
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
export function useProject() {
  return {
    getProjectById: (projectId: string) => stores.projects.find((project) => project.id === projectId),
    fetchProjects: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the projects of ${named(workspace)}`),
    fetchProject: (projectId: string) => fetching(`the project ${projectId}`),
  };
}

export function useProjectPreferences() {
  return {
    fetchNavigation: (projectId: string) => fetching(`the tab bar in ${projectId}`),
  };
}

export function useLabel() {
  return {
    fetchProjectLabels: (projectId: string) => fetching(`the labels of ${projectId}`),
````
````old web/apps/web/core/hooks/store/fake-store-hooks.ts
    fetchWorkspaceStates: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the states of ${named(workspace)}`),
````
````new web/apps/web/core/hooks/store/fake-store-hooks.ts
    fetchWorkspaceStates: (workspace: Pick<Workspace, "id" | "slug">) => fetching(`the states of ${named(workspace)}`),
    fetchProjectStates: (projectId: string) => fetching(`the states of ${projectId}`),
````

`web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts`（新文件，105 行）：

````file web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { ApiError } from "@/lib/api-error";
import { fetchHanded, handed, response } from "@/lib/fake-session-swr";
import { projectOf } from "@/store/project/fake-projects";

// A page of a project fetches what its caller may read (M3 design 3.1, 7.1): the project, and its own reads only once
// nerve's read says he is a member; its wrapper shows what that read decides (3.19, 7.6). fake-session-swr.ts stands in
// for useSessionSWR and fake-store-hooks.ts for the stores.

vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/store/use-project", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-preferences", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-label", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-member", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/hooks/store/use-project-state", () => import("@/hooks/store/fake-store-hooks"));

const { useProjectFetch } = await import("./use-project-fetch");

const member = projectOf("WEB", "id-acme");
const guest = projectOf("WEB", "id-acme", { member_role: 5 });
const seen = projectOf("WEB", "id-acme", { member_role: null });
const notFound = new ApiError(404, { status: 404, code: "project.not_found", title: "Not Found" });
const unreachable = new ApiError(503, { status: 503, code: "server_busy", title: "Service Unavailable" });

beforeEach(() => {
  handed.length = 0;
  response.current = {};
  emptyStores();
});

describe("useProjectFetch", () => {
  it("fetches the project and, for a member, his tab bar in it, its labels, its members and its states", async () => {
    stores.projects = [member];
    response.current = { data: member };
    expect(useProjectFetch(member.id)).toEqual({ kind: "member", project: member });
    expect(handed.map(([fetch]) => fetch)).toEqual([
      ["PROJECT", "p-web"],
      ["PROJECT_PREFERENCES", "p-web"],
      ["PROJECT_LABELS", "p-web"],
      ["PROJECT_MEMBERS", "p-web"],
      ["PROJECT_STATES", "p-web"],
    ]);
    await fetchHanded();
    expect(stores.fetched).toEqual([
      "the project p-web",
      "the tab bar in p-web",
      "the labels of p-web",
      "the members of p-web",
      "the states of p-web",
    ]);
  });

  it.each<{ when: string; projects: Project[]; data?: Project; error?: ApiError }>([
    { when: "he sees it and is no member", projects: [seen], data: seen },
    { when: "nerve has not read it yet, though its workspace's list has it", projects: [member] },
    { when: "nerve does not find it", projects: [], error: notFound },
    { when: "nerve cannot read it again for a member", projects: [member], data: member, error: unreachable },
  ])("fetches the project alone when $when", ({ projects, data, error }) => {
    stores.projects = projects;
    response.current = { data, error };
    useProjectFetch(member.id);
    expect(handed.map(([fetch]) => fetch)).toEqual([["PROJECT", "p-web"], null, null, null, null]);
  });

  it.each<{ when: string; projects: Project[]; data?: Project; error?: ApiError; shows: object }>([
    {
      when: "nerve cannot read it the first time",
      projects: [],
      error: unreachable,
      shows: { kind: "unavailable", retry: expect.any(Function) },
    },
    {
      when: "nerve cannot read it again",
      projects: [member],
      data: member,
      error: unreachable,
      shows: { kind: "unavailable", retry: expect.any(Function) },
    },
    { when: "nerve does not find it", projects: [member], error: notFound, shows: { kind: "not-found" } },
    { when: "nerve has not answered yet", projects: [member], shows: { kind: "loading" } },
    { when: "the store no longer gives it", projects: [], data: member, shows: { kind: "not-found" } },
    { when: "he sees it, no member", projects: [seen], data: seen, shows: { kind: "not-member", project: seen } },
    { when: "he is its guest", projects: [guest], data: guest, shows: { kind: "member", project: guest } },
    { when: "he is its member", projects: [member], data: member, shows: { kind: "member", project: member } },
  ])("shows the wrapper what to render when $when", ({ projects, data, error, shows }) => {
    stores.projects = projects;
    response.current = { data, error };
    expect(useProjectFetch(member.id)).toEqual(shows);
  });

  it("reads the project again when the page's retry asks", () => {
    const mutate = vi.fn(() => Promise.resolve(undefined));
    response.current = { error: unreachable, mutate };
    const access = useProjectFetch(member.id);
    if (access.kind === "unavailable") access.retry();
    expect(mutate).toHaveBeenCalledOnce();
  });
});
````

`web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`（新文件，70 行）：

````file web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
import { useProjectState } from "@/hooks/store/use-project-state";
// lib
import { ApiError } from "@/lib/api-error";
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * What ProjectAuthWrapper shows for the address's project (M3 design 3.19, 7.6): that nerve cannot be reached, with a
 * retry; a wait for nerve's read of the project; that the project is not found (it does not exist, is deleted, or the
 * caller does not see it); that he sees it and is no member of it, which he may join; or its pages, a member's.
 */
export type ProjectAccess =
  | { kind: "unavailable"; retry: () => void }
  | { kind: "loading" }
  | { kind: "not-found" }
  | { kind: "not-member"; project: Project }
  | { kind: "member"; project: Project };

/**
 * The project side of what a page of a project fetches as it mounts (M3 design 3.1, 7.1), and the one decision what
 * the project is to the caller: nerve's read of the project, which decides it; once it says he is a member, his tab
 * bar in the project, its labels, its members and its states, which nerve gives its members alone. Gives what the
 * wrapper shows.
 */
export function useProjectFetch(projectId: string): ProjectAccess {
  const { getProjectById, fetchProject } = useProject();
  const { fetchNavigation } = useProjectPreferences();
  const { fetchProjectLabels } = useLabel();
  const {
    project: { fetchProjectMembers },
  } = useMember();
  const { fetchProjectStates } = useProjectState();
  const read = useSessionSWR(["PROJECT", projectId], (id) => fetchProject(id));
  const access = decide(read, getProjectById(projectId));
  // the project's own reads, a member's alone: for anyone else they are nothing to fetch
  const member = access.kind === "member" ? access.project.id : undefined;
  useSessionSWR(member ? ["PROJECT_PREFERENCES", member] : null, (id) => fetchNavigation(id));
  useSessionSWR(member ? ["PROJECT_LABELS", member] : null, (id) => fetchProjectLabels(id));
  useSessionSWR(member ? ["PROJECT_MEMBERS", member] : null, (id) => fetchProjectMembers(id));
  useSessionSWR(member ? ["PROJECT_STATES", member] : null, (id) => fetchProjectStates(id));
  return access;
}

/** Of the SWR answer to the read of the project, what the decision takes. */
type Read = { data?: unknown; error?: unknown; mutate: () => Promise<unknown> };

/**
 * The decision, in this order: nerve's refusal of the read (the project not found, else nerve not reached, even for a
 * project it gave before); no answer yet; the project as the store now gives it (none once deleted or left), whose
 * member_role says whether he is a member.
 */
function decide(read: Read, project: Project | undefined): ProjectAccess {
  if (read.error) {
    const notFound = read.error instanceof ApiError && read.error.problem?.code === "project.not_found";
    return notFound ? { kind: "not-found" } : { kind: "unavailable", retry: () => void read.mutate() };
  }
  if (read.data === undefined) return { kind: "loading" };
  if (project === undefined) return { kind: "not-found" };
  return project.member_role === null ? { kind: "not-member", project } : { kind: "member", project };
}
````

- [ ] **Step 2: 包装层和它的界面**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx
        <ProjectAuthWrapper workspaceSlug={workspaceSlug} projectId={projectId}>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx
        <ProjectAuthWrapper projectId={projectId}>
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
      <ProjectAuthWrapper workspaceSlug={workspaceSlug} projectId={projectId}>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx
      <ProjectAuthWrapper projectId={projectId}>
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx`（修改，2 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
  const { workspaceSlug, projectId } = params;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
  const { projectId } = params;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
          <ProjectAuthWrapper workspaceSlug={workspaceSlug} projectId={projectId}>
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx
          <ProjectAuthWrapper projectId={projectId}>
````

`web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx`（修改，5 处）：

````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  isWorkspaceAdmin: boolean;
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  /** Whether the caller sees the project and is no member of it (M3 design 3.19): he may join it. */
  canJoin: boolean;
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  isJoinButtonDisabled: boolean;
  errorStatusCode: number | undefined;
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  isJoinButtonDisabled: boolean;
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  const { isWorkspaceAdmin, handleJoinProject, isJoinButtonDisabled, errorStatusCode } = props;
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  const { canJoin, handleJoinProject, isJoinButtonDisabled } = props;
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  // Show join project screen if:
  // - User lacks project membership (409 Conflict)
  // - User lacks permission to access the private project (403 Forbidden) but is a workspace admin (can join any project)
  if (errorStatusCode === 409 || (errorStatusCode === 403 && isWorkspaceAdmin))
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  // the caller sees the project and is no member of it: he may join it
  if (canJoin)
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  // Show no access screen if:
  // - User lacks permission to access the private project (403 Forbidden)
  if (errorStatusCode === 403) {
    return (
      <div className="grid h-full w-full place-items-center bg-surface-1">
        <EmptyStateDetailed
          title={t("project_empty_state.no_access.title")}
          description={t("project_empty_state.no_access.restricted_description")}
          assetKey="no-access"
          assetClassName="size-40"
        />
      </div>
    );
  }

  // Show empty state screen if:
  // - Project not found (404 Not Found)
  // - Any other error status code
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  // the project is not found to him: it does not exist, is deleted, or he does not see it
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（整个文件，58 行）：

````whole web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState } from "react";
import { observer } from "mobx-react";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { ProjectAccessRestriction } from "@/components/auth-screens/project/project-access-restriction";
// hooks
import { useProject } from "@/hooks/store/use-project";
// local imports
import { useProjectFetch } from "./use-project-fetch";

interface IProjectAuthWrapper {
  projectId: string;
  children: ReactNode;
}

export const ProjectAuthWrapper = observer(function ProjectAuthWrapper(props: IProjectAuthWrapper) {
  const { projectId, children } = props;
  // states
  const [isJoiningProject, setIsJoiningProject] = useState(false);
  // store hooks
  const { joinProject } = useProject();

  // the project side of what every page of a project fetches (M3 design 7.1), and what nerve's read of the project
  // decides it is to the caller (3.19)
  const access = useProjectFetch(projectId);

  // handle join project
  const handleJoinProject = () => {
    setIsJoiningProject(true);
    joinProject(projectId).finally(() => setIsJoiningProject(false));
  };

  // nerve's read of the project has not answered yet
  if (access.kind === "loading") return null;

  // nerve could not be reached: the page says so, and tries again when asked (M2 design 7.1)
  if (access.kind === "unavailable") return <SessionUnavailable autoRetry={false} onRetry={access.retry} />;

  // a project the caller sees and is no member of, which he may join; or one not found to him
  if (access.kind !== "member") {
    return (
      <ProjectAccessRestriction
        canJoin={access.kind === "not-member"}
        handleJoinProject={handleJoinProject}
        isJoinButtonDisabled={isJoiningProject}
      />
    );
  }

  return <>{children}</>;
});
````

`web/packages/i18n/src/locales/en/empty-state.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/empty-state.json
      "title": "Seems like you don’t have access to this Project",
      "restricted_description": "Contact admin to request for access and you can continue here.",
````
````new web/packages/i18n/src/locales/en/empty-state.json
      "title": "Seems like you don’t have access to this Project",
````

`web/packages/i18n/src/locales/zh-CN/empty-state.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/empty-state.json
      "title": "您似乎无权访问该项目",
      "restricted_description": "请联系管理员申请访问权限，通过后您可以在此继续。",
````
````new web/packages/i18n/src/locales/zh-CN/empty-state.json
      "title": "您似乎无权访问该项目",
````

- [ ] **Step 3: 重复的取数、键和静态检查**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/labels/**"
````
````new .oxlintrc.json
        "web/apps/web/core/components/labels/**",
        "web/apps/web/core/layouts/auth-layout/project-wrapper.tsx",
        "web/apps/web/core/layouts/auth-layout/use-project-fetch.ts",
        "web/apps/web/core/components/project/project-settings-member-defaults.tsx"
````
````old .oxlintrc.json
        "web/apps/web/core/store/label.store.test.ts",
````
````new .oxlintrc.json
        "web/apps/web/core/store/label.store.test.ts",
        "web/apps/web/core/layouts/auth-layout/use-project-fetch.ts",
        "web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts",
````

`web/apps/web/core/components/project/project-settings-member-defaults.tsx`（修改，4 处）：

````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
import { Controller, useForm } from "react-hook-form";
import useSWR from "swr";
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
import { Controller, useForm } from "react-hook-form";
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
import { Loader } from "@nerve/ui";
// constants
import { PROJECT_DETAILS } from "@nerve/constants";
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
import { Loader } from "@nerve/ui";
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { currentProjectDetails, fetchProject, updateProject } = useProject();
````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { currentProjectDetails, updateProject } = useProject();
````
````old web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { reset, control } = useForm<TMemberDefaults>({ defaultValues });
  // fetching user members
  useSWR(
    workspaceSlug && projectId ? PROJECT_DETAILS(workspaceSlug, projectId) : null,
    workspaceSlug && projectId ? () => fetchProject(projectId) : null
  );

````
````new web/apps/web/core/components/project/project-settings-member-defaults.tsx
  const { reset, control } = useForm<TMemberDefaults>({ defaultValues });
````

`web/packages/constants/src/fetch-keys.ts`（修改，2 处）：

````old web/packages/constants/src/fetch-keys.ts
 */

import type { EUserPermissions } from "@nerve/types";
````
````new web/packages/constants/src/fetch-keys.ts
 */
````
````old web/packages/constants/src/fetch-keys.ts

// project level keys
export const PROJECT_DETAILS = (_workspaceSlug: string, projectId: string) =>
  `PROJECT_DETAILS_${projectId.toUpperCase()}`;

export const PROJECT_LABELS = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_LABELS_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_MEMBERS = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_MEMBERS_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_STATES = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_STATES_${projectId.toUpperCase()}_${projectRole}`;

export const PROJECT_MEMBER_PREFERENCES = (projectId: string, projectRole: EUserPermissions | undefined) =>
  `PROJECT_MEMBER_PREFERENCES_${projectId.toUpperCase()}_${projectRole}`;

````
````new web/packages/constants/src/fetch-keys.ts

````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、2 个例外，没有命中；web 的 oxlint 367 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

完成时：本 Task 删除的文案键（`empty-state.json` 的 `project_empty_state.no_access.restricted_description`）在 `en`、`zh-CN` 中都已不在（spec 附录 A.7 的键表）。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/layout.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/layout.tsx' web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/hooks/store/fake-store-hooks.ts web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-project-fetch.ts web/packages/constants/src/fetch-keys.ts web/packages/i18n/src/locales/en/empty-state.json web/packages/i18n/src/locales/zh-CN/empty-state.json
```
```bash
git commit -m "feat(M3/P8b): the project wrapper decides by nerve's read, and fetches a member's reads alone

useProjectFetch reads the project through useSessionSWR and makes the one
decision what it is to the caller: nerve not reached, not answered yet,
not found, seen by one who is no member, or a member's. Only then does
it fetch his tab bar, the labels, the members and the states, so no one
who is no member asks nerve for them. ProjectAuthWrapper renders what it
decides; Plane's status-code branches, the no-access screen and the
duplicate read of the member defaults go.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t10-pf-non-member` | 看得到项目、不是成员（`member_role` 为 `null`）的人也取项目自己的四个读 | `use-project-fetch.test.ts`、故事 S2（之后的 Task 起） | vitest；端到端 |
| `t10-pf-before-read` | nerve 读到项目之前就取项目自己的四个读 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-no-states` | 不取项目的状态 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-stale-member` | nerve 重取不到的项目按 store 的旧副本显示 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-any-refusal-not-found` | 读被拒绝，不论什么码都显示找不到项目 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-store-gone-waits` | 项目 store 不再给出的项目一直等，不显示找不到 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-guest-not-member` | 项目的访客看到加入的界面 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-no-retry` | 页面的重取什么都不重读 | `use-project-fetch.test.ts` | vitest |
| `t10-pw-children-non-member` | 包装层给不是成员的人渲染项目的页面 | 故事 S2（之后的 Task 起） | 端到端 |
| `t10-pf-code` | nerve 找不到的项目显示为连不上（读的码不是 nerve 的） | `use-project-fetch.test.ts` | vitest |
| `t10-pf-no-loading` | nerve 回答读之前就按 store 的副本显示 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-all-members` | 看得到项目、不是成员的人也得到它的页面 | `use-project-fetch.test.ts` | vitest |
| `t10-pf-no-members` | 项目的成员看到加入的界面 | `use-project-fetch.test.ts` | vitest |
| `t10-raw-swr-project` | 项目的 hook 经 `useSWR` 取标签，键不带 `loginId` | oxlint（`check:lint`） | 静态 |
| `t10-raw-swr-wrapper` | 项目包装层自己经 `useSWR` 取项目 | oxlint（`check:lint`） | 静态 |
| `t10-nonnull-project-fetch` | 项目的 hook 断言项目 store 一定给出这个项目（`!`） | oxlint（`check:lint`） | 静态 |

---

### Task 11: 关键词规则补全；S2 改写为登录之后的挂载清单；项目筛选的死行；文档

**Files:**
- Modify: `docs/v0/frontend-changes.md`、`docs/v0/v0-design.md`、`e2e/stories/smoke/s2-web-app.spec.ts`、`tools/keywords.json`、`web/apps/web/core/lib/store-context.test.ts`、`web/apps/web/core/store/project/project_filter.store.ts`

**Interfaces:**
- 关键词规则（spec 2.11；M3 设计 7.10；M2 交接第 11 节）：
  - `plane-workspace-urls` 的模式补上项目一侧的旧地址：`/project-roles/`、`/api/workspaces/${…}/projects/` 的列出和创建、`projects/details/`、一个项目的地址和它的 `archive/`、`members/`、`project-members/me/`、`states/`、`issue-labels/`、`user-properties/`，以及工作区的 `project-identifiers`、`states/`、`labels/`；`why` 照改。原来项目一侧的不命中样例改为命中样例，另加六个命中样例；`/search-issues/`（M4）、迭代的 `user-properties/`（M6）、`archived-cycles`、工作项的标签和 v0 的标签地址是不命中样例。
  - 精确例外一条（7.10 写明的 `until: "M4"`）：`core/services/project/project.service.ts` 中项目的 `user-properties/` 两处（旧 `ProjectService` 留给 M4 的工作项筛选的两个方法，Task 5）。
  - `plane-user-urls` 收紧为整个 `/api/users/`（和 `/api/instances/`）：原来的三个不命中样例改为命中样例，加 `project-roles` 的命中样例和 v0 加入项目的不命中样例。
- S2（`e2e/stories/smoke/s2-web-app.spec.ts`；M3 设计第 2 节 S2；P8a review 第 6 节）：未登录的两个测试不变；加四个账户各一个测试：工作区的管理员、成员、访客，和工作区的成员而不是项目成员的人。每个账户登录、落点到工作区的首页，再打开项目的设置页；每一页加载时的 API 请求（去重、排序，slug 和项目的 id 换成名字）必须恰好等于给它的清单：应用启动的、工作区包装层的、项目的读，和只有项目成员才有的项目自己的四个（不是项目成员的人没有）。清单上没有 M6、M7 的地址（迭代、模块、视图、分诊状态、收藏、未读通知数、"最近"），也没有 `/api/v0` 之外的；两张清单都齐了之后再没有请求；没有失败的请求、没有发往旧接口的请求、没有 CSP 违规、没有页面错误，控制台只有两次加载各一条已知的警告。
- `project_filter.store.ts` 的死行（P8a spec 第 5 节）：没有读者的 `getDisplayFiltersByWorkspaceSlug`、`getFiltersByWorkspaceSlug` 和接口上的 `displayFilters`、`filters` 删除；`store-context.test.ts` 改读 `currentWorkspaceDisplayFilters`。
- 文档（3.20 中 P8b 的行）：总体设计 7.7 的三处（`.oxlintrc.json` 的范围加上 P8b 的文件；"页面按权限决定取数"加项目包装层的例子；`reconciled.ts` 的使用者加上 P8b 的 store 和 `values()`）；前端改动清单 3.1 的 M3 一行和错误格式一行，3.2 加五行（项目封面的上传、项目的收藏、收集箱的分诊状态、工作区级的标签、项目一侧的挂载时取数）。

**Tests:** S2 的四个新测试（`make e2e`，74 个）；`store-context.test.ts` 照旧 7 个。没有新的 vitest。

- [ ] **Step 1: 关键词规则**

`tools/keywords.json`（修改，11 处）：

````old tools/keywords.json
      "why": "Plane 的当前用户、资料、新手引导、导览和实例地址（M2 设计 7.9）；前端改调 /api/v0 下的 /me、/me/profile 和 /instance。当前用户只禁止 /api/users/me/ 本身（后面紧跟引号），不禁止整个 /api/users/me/：M3 的 joinProject、工作区列表和邀请等旧调用还在用它下面的地址",
````
````new tools/keywords.json
      "why": "Plane 的当前用户、资料、新手引导、导览和实例地址（M2 设计 7.9），以及它们之下 M3 的工作区列表、项目角色、加入项目等地址；前端改调 /api/v0 下的 /me、/me/profile、/instance 和 M3 的接口。M3/P8b 起禁止整个 /api/users/（M3 设计 7.10，M2 交接第 11 节）：它下面的旧调用都已迁走",
````
````old tools/keywords.json
        "source": "/api/users/me/(?:profile|onboard|tour-completed|(?=[\"'`]))|/api/instances/",
````
````new tools/keywords.json
        "source": "/api/users/|/api/instances/",
````
````old tools/keywords.json
          "    return this.get(\"/api/instances/\")"
````
````new tools/keywords.json
          "    return this.get(\"/api/instances/\")",
          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })",
          "    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : \"/api/users/me/settings/\";",
          "    return this.get(\"/api/users/me/workspaces/\")",
          "    return this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`)"
````
````old tools/keywords.json
          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })",
          "    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : \"/api/users/me/settings/\";",
          "    return this.get(\"/api/users/me/workspaces/\")"
        ],
````
````new tools/keywords.json
          "    return unwrap(await this.api.POST(\"/api/v0/projects/{project_id}/join\", { params: { path: { project_id: projectId } } }));"
        ],
````
````old tools/keywords.json
      "why": "M3 换掉的 Plane 地址（M3 设计 7.10）：工作区一侧的 store 改调 /api/v0 之后，旧 service 中这些地址的方法都已删除。模式只写到 P8a 已换掉的地址为止；项目一侧的地址（项目、项目成员、项目角色、状态、标签、显示设置）P8b 换掉时补进来，在那之前它们不命中。/search-issues/ 不是 M3 的地址",
````
````new tools/keywords.json
      "why": "M3 换掉的 Plane 地址（M3 设计 7.10）：工作区、项目两侧的 store 改调 /api/v0 之后，旧 service 中这些地址的方法都已删除（P8a 工作区一侧；P8b 项目一侧：项目、项目成员、项目角色、归档、标识检查、状态、标签、显示设置）。项目的 /user-properties/ 是 M4 的工作项筛选读写的，留一条例外到 M4。/search-issues/ 和迭代、模块、工作项的地址不是 M3 的，模式不宽到它们",
````
````old tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/|/api/workspaces/\\$\\{[^}]+\\}/user-properties/",
````
````new tools/keywords.json
        "source": "[\"'`]/api/users/me/workspaces/[\"'`]|[\"'`]/api/workspaces/[\"'`]|/api/workspaces/\\$\\{[^}]+\\}/`|/api/users/last-visited-workspace/|/api/workspace-slug-check/|/api/workspaces/\\$\\{[^}]+\\}/members/|/workspace-members/me/|/api/workspaces/\\$\\{[^}]+\\}/invitations/|/api/workspaces/\\$\\{[^}]+\\}/user-properties/|/project-roles/|/api/workspaces/\\$\\{[^}]+\\}/projects/(?:`|details/|\\$\\{[^}]+\\}/(?:`|archive/|members/|project-members/me/|states/|issue-labels/|user-properties/))|/api/workspaces/\\$\\{[^}]+\\}/(?:project-identifiers|states/|labels/)",
````
````old tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data)"
````
````new tools/keywords.json
          "    return this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data)",
          "    return this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/states/`)",
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/project-members/me/`)",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`, data)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/details/`)",
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/archive/`, {})",
          "    return this.get(`/api/workspaces/${workspaceSlug}/project-identifiers`, {",
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/${stateId}/mark-default/`, {})",
          "    return this.get(`/api/workspaces/${workspaceSlug}/labels/`)",
          "    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/${labelId}/`)"
````
````old tools/keywords.json
          "    return unwrap(await this.api.DELETE(\"/api/v0/workspaces/{slug}\", { params: { path: { slug } } }));",
          "    return this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`)",
          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/search-issues/`, {",
          "    return this.get(`/api/workspaces/${workspaceSlug}/states/`)",
````
````new tools/keywords.json
          "    return unwrap(await this.api.DELETE(\"/api/v0/workspaces/{slug}\", { params: { path: { slug } } }));",
          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/search-issues/`, {",
````
````old tools/keywords.json
          "    unwrap(await this.api.POST(\"/api/v0/workspaces/{slug}/leave\", { params: { path: { slug } } }));",
          "    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/project-members/me/`)",
          "    return unwrap(await this.api.GET(\"/api/v0/workspaces/{slug}/members\", { params: { path: { slug } } })).data;",
          "    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`, data)",
````
````new tools/keywords.json
          "    unwrap(await this.api.POST(\"/api/v0/workspaces/{slug}/leave\", { params: { path: { slug } } }));",
          "    return unwrap(await this.api.GET(\"/api/v0/workspaces/{slug}/members\", { params: { path: { slug } } })).data;",
````
````old tools/keywords.json
          "    return unwrap(await this.api.GET(\"/api/v0/me/workspaces/{slug}/preferences\", { params: { path: { slug } } }));"
````
````new tools/keywords.json
          "    return unwrap(await this.api.GET(\"/api/v0/me/workspaces/{slug}/preferences\", { params: { path: { slug } } }));",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/archived-cycles/`)",
          "    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/labels/`)",
          "    return unwrap(await this.api.GET(\"/api/v0/projects/{project_id}/labels\", { params: { path: { project_id: projectId } } }));"
````
````old tools/keywords.json
      "until": "M9"
````
````new tools/keywords.json
      "until": "M9"
    },
    {
      "rule": "plane-workspace-urls",
      "path": "web/apps/web/core/services/project/project.service.ts",
      "match": "/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/",
      "count": 2,
      "reason": "M3/P8b 起，M4 的工作项筛选改用新接口时删除：项目的工作项筛选（getProjectUserProperties、updateProjectUserProperties）由 M4 的筛选 store 读写（M3 设计 7.3）",
      "until": "M4"
````

- [ ] **Step 2: S2**

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，3 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
import type { Page, Response } from "@playwright/test";

````
````new e2e/stories/smoke/s2-web-app.spec.ts
import type { Page, Request, Response, TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Project,
} from "../../fixtures/api";
import { accountId, emailFor, type AuthTokens } from "../../fixtures/auth";
````
````old e2e/stories/smoke/s2-web-app.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
````
````new e2e/stories/smoke/s2-web-app.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  await expectQuietSignedOut(page, watch, [EMOJI_CHECK_WARNING]);
});

````
````new e2e/stories/smoke/s2-web-app.spec.ts
  await expectQuietSignedOut(page, watch, [EMOJI_CHECK_WARNING]);
});

/** The accounts of the workspace S2 signs in to: its admin; a member and a guest of it in its project too; a member of it who is not. */
const ACCOUNTS = ["admin", "member", "guest", "project non-member"] as const;
type Account = (typeof ACCOUNTS)[number];

/**
 * What a page asks nerve for as it loads, signed in, as "<method> <path>", {slug} for the workspace's slug and
 * {project} for the project's id: the app's start, on every page (M2 design 7.1, M3 design 7.4); the workspace
 * wrapper's, on every page of a workspace, whatever the role; the project's read, on every page of a project; and the
 * project's own resources, only once that read says the caller is a member of the project (M3 design 7.1). No list
 * has an address of M6's or M7's (cycles, modules, views, the intake's triage state, favourites, the unread
 * notifications, recents; M3 design 3.1), nor one outside /api/v0 (M2 design 3.1).
 */
const APP = [
  "POST /api/v0/auth/refresh",
  "GET /api/v0/instance",
  "GET /api/v0/me",
  "GET /api/v0/me/profile",
  "GET /api/v0/workspaces",
];
const WORKSPACE = [
  "GET /api/v0/me/workspaces/{slug}/preferences",
  "GET /api/v0/workspaces/{slug}/members",
  "GET /api/v0/workspaces/{slug}/projects",
  "GET /api/v0/workspaces/{slug}/states",
];
const PROJECT = ["GET /api/v0/projects/{project}"];
const PROJECT_MEMBER = [
  "GET /api/v0/me/projects/{project}/preferences",
  "GET /api/v0/projects/{project}/labels",
  "GET /api/v0/projects/{project}/members",
  "GET /api/v0/projects/{project}/states",
];
/** The project's general settings, the page S2 opens, list the time zones for its member (M2 design 5.3). */
const GENERAL = ["GET /api/v0/timezones"];

/** The requests of the two pages each account opens, each once, sorted: the workspace's home, then the project's settings. */
const REQUESTS: Record<Account, { home: string[]; settings: string[] }> = {
  admin: { home: [...APP, ...WORKSPACE], settings: [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL] },
  member: { home: [...APP, ...WORKSPACE], settings: [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL] },
  guest: { home: [...APP, ...WORKSPACE], settings: [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL] },
  "project non-member": { home: [...APP, ...WORKSPACE], settings: [...APP, ...WORKSPACE, ...PROJECT] },
};

/**
 * Makes S2's workspace through the API: its admin creates it and a public project; a member and a guest of the
 * workspace join the project with the same roles; another member of the workspace does not. Returns the tokens of
 * each account, which have not been used in a browser yet.
 */
async function acme(
  api: Api,
  testInfo: TestInfo
): Promise<{ slug: string; project: Project; tokens: Record<Account, AuthTokens> }> {
  const signUp = async (label: string) => {
    const email = emailFor(testInfo, label);
    return { email, tokens: await registerOnboarded(api, email) };
  };
  const admin = await signUp("admin");
  const member = await signUp("member");
  const guest = await signUp("guest");
  const outsider = await signUp("outsider");
  const token = admin.tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const project = await createProject(api, token, slug, { name: "Web", identifier: "WEB", network: 2 });
  const join = (who: { email: string; tokens: AuthTokens }, role: 5 | 15) =>
    inviteAndAccept(api, token, slug, { email: who.email, token: who.tokens.access_token }, role);
  await join(member, 15);
  await join(guest, 5);
  await join(outsider, 15);
  await addProjectMembers(api, token, project.id, [
    { member_id: await accountId(api, member.tokens.access_token), role: 15 },
    { member_id: await accountId(api, guest.tokens.access_token), role: 5 },
  ]);
  const tokens = {
    admin: admin.tokens,
    member: member.tokens,
    guest: guest.tokens,
    "project non-member": outsider.tokens,
  };
  return { slug, project, tokens };
}

function apiPath(request: Request): string | undefined {
  const { pathname } = new URL(request.url());
  return pathname.startsWith("/api/") ? pathname : undefined;
}

/**
 * Follows the API requests of page from now on, each as "<method> <path>" with the names of names in place of their
 * values. between(from, to) gives the requests from the index from to the index to (the end when it is left out),
 * each once and sorted, once every request so far has ended, answered or failed, and null until then; next() is the
 * index of the next request.
 */
function followRequests(page: Page, names: Record<string, string>) {
  const requests: string[] = [];
  let ended = 0;
  page.on("request", (request) => {
    const path = apiPath(request);
    if (path !== undefined) {
      const named = Object.entries(names).reduce((shown, [value, name]) => shown.replaceAll(value, name), path);
      requests.push(`${request.method()} ${named}`);
    }
  });
  const end = (request: Request) => {
    if (apiPath(request) !== undefined) {
      ended += 1;
    }
  };
  page.on("requestfinished", end);
  page.on("requestfailed", end);
  return {
    next: () => requests.length,
    between: (from: number, to?: number) =>
      ended === requests.length ? [...new Set(requests.slice(from, to))].toSorted() : null,
  };
}

for (const account of ACCOUNTS) {
  test(`S2: the workspace's ${account} signs in, lands on its home and opens the project's settings`, async ({
    api,
    signedInPage,
  }, testInfo) => {
    const { slug, project, tokens } = await acme(api, testInfo);
    const page = await signedInPage(tokens[account]);
    const watch = await watchPage(page);
    const requests = followRequests(page, { [slug]: "{slug}", [project.id]: "{project}" });
    const expected = REQUESTS[account];

    await page.goto("/");
    await expect(page).toHaveURL(`/${slug}`);
    await expect.poll(() => requests.between(0)).toEqual(expected.home.toSorted());
    const settings = requests.next();
    await page.goto(`/${slug}/settings/projects/${project.id}`);
    await expect.poll(() => requests.between(settings)).toEqual(expected.settings.toSorted());

    expect(watch.apiFailures).toEqual([]);
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    // The hint is logged once for each of the two loads (EMOJI_CHECK_WARNING).
    await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
    // Nothing came after either list was whole.
    expect(requests.between(0, settings)).toEqual(expected.home.toSorted());
    expect(requests.between(settings)).toEqual(expected.settings.toSorted());
  });
}

````

- [ ] **Step 3: 项目筛选的死行**

`web/apps/web/core/lib/store-context.test.ts`（修改，2 处）：

````old web/apps/web/core/lib/store-context.test.ts
    expect(y.projectRoot.projectFilter.getDisplayFiltersByWorkspaceSlug("acme")).toBeDefined();
````
````new web/apps/web/core/lib/store-context.test.ts
    expect(y.projectRoot.projectFilter.currentWorkspaceDisplayFilters).toBeDefined();
````
````old web/apps/web/core/lib/store-context.test.ts
    expect(x.projectRoot.projectFilter.getDisplayFiltersByWorkspaceSlug("acme")).toBeUndefined();
````
````new web/apps/web/core/lib/store-context.test.ts
    expect(x.projectRoot.projectFilter.currentWorkspaceDisplayFilters).toBeUndefined();
````

`web/apps/web/core/store/project/project_filter.store.ts`（修改，5 处）：

````old web/apps/web/core/store/project/project_filter.store.ts
import { action, computed, observable, makeObservable, runInAction, reaction } from "mobx";
import { computedFn } from "mobx-utils";
````
````new web/apps/web/core/store/project/project_filter.store.ts
import { action, computed, observable, makeObservable, runInAction, reaction } from "mobx";
````
````old web/apps/web/core/store/project/project_filter.store.ts
  // observables
  displayFilters: Record<string, TProjectDisplayFilters>;
  filters: Record<string, TProjectFilters>;
````
````new web/apps/web/core/store/project/project_filter.store.ts
  // observables
````
````old web/apps/web/core/store/project/project_filter.store.ts
  currentWorkspaceFilters: TProjectFilters | undefined;
  // computed functions
  getDisplayFiltersByWorkspaceSlug: (workspaceSlug: string) => TProjectDisplayFilters | undefined;
  getFiltersByWorkspaceSlug: (workspaceSlug: string) => TProjectFilters | undefined;
````
````new web/apps/web/core/store/project/project_filter.store.ts
  currentWorkspaceFilters: TProjectFilters | undefined;
````
````old web/apps/web/core/store/project/project_filter.store.ts
  /**
   * @description get display filters of a workspace by workspaceSlug
   * @param {string} workspaceSlug
   */
  getDisplayFiltersByWorkspaceSlug = computedFn((workspaceSlug: string) => this.displayFilters[workspaceSlug]);

  /**
   * @description get filters of a workspace by workspaceSlug
   * @param {string} workspaceSlug
   */
  getFiltersByWorkspaceSlug = computedFn((workspaceSlug: string) => this.filters[workspaceSlug]);

  /**
````
````new web/apps/web/core/store/project/project_filter.store.ts
  /**
````
````old web/apps/web/core/store/project/project_filter.store.ts
    const displayFilters = this.getDisplayFiltersByWorkspaceSlug(workspaceSlug);
````
````new web/apps/web/core/store/project/project_filter.store.ts
    const displayFilters = this.displayFilters[workspaceSlug];
````

- [ ] **Step 4: 文档**

`docs/v0/frontend-changes.md`（修改，3 处）：

````old docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：工作区、成员、邀请、工作区的显示设置已对接（M3/P8a）；项目一侧在 M3/P8b |
````
````new docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；页面在 M3/P9–P11 |
````
````old docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`，它们的页面按 `code` 显示错误在 M3/P9；M3 的项目一侧和 M4–M8 的领域在各自的 Phase 和 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````new docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`，它们的页面按 `code` 显示错误在 M3/P9–P11；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````old docs/v0/frontend-changes.md
| 工作区图标的上传 | 工作区设置 general 页的图标上传弹窗删除：接口不能设置 `logo_url`，它恒为 `null`，显示图标的地方照旧显示首字母；M5 随文件的接口加回 | M3 设计 3.2 | 已完成（M5 加回） | M3/P8a |
````
````new docs/v0/frontend-changes.md
| 工作区图标的上传 | 工作区设置 general 页的图标上传弹窗删除：接口不能设置 `logo_url`，它恒为 `null`，显示图标的地方照旧显示首字母；M5 随文件的接口加回 | M3 设计 3.2 | 已完成（M5 加回） | M3/P8a |
| 项目封面的上传 | 新建项目时的封面一步、默认的随机封面和项目设置的封面选择器删除，预设的封面图只留显示用的一张：接口不能设置 `cover_image_url`，它恒为 `null`，显示封面的地方用这张默认图（`helpers/cover-image.helper.ts`）；M5 随文件的接口加回 | M3 设计 3.2 | 已完成（M5 加回） | M3/P8b |
| 项目的收藏 | 项目卡片的收藏按钮、建项目之后加入收藏的一步、项目 store 的两个收藏方法和 `favoriteProjectIds`、收藏 store 移除收藏时改项目的一支删除；`Project` 没有 `is_favorite`；M7 随收藏的接口加回 | M3 设计 3.2 | 已完成（M7 加回） | M3/P8b |
| 收集箱的分诊状态 | state store 的分诊状态、取它的旧 service 方法、分诊状态的下拉框、类型和图标删除；收集箱中已接受的工作项照旧显示状态，新建弹窗不再选分诊状态；M7 随收集箱的接口加回 | M3 设计 3.1、第 12 节 P8b | 已完成（M7 加回） | M3/P8b |
| 工作区级的标签 | 工作区内全部项目的标签列表、它的取数和旧地址删除（nerve 没有工作区级的标签，标签按项目）；工作区一级的工作项列表按标签分组只有"无"，标签筛选没有选项；M7 的视图决定工作区一级怎样取标签 | M3 设计 3.16、7.3 | 已完成（M7 决定） | M3/P8b |
| 项目一侧的挂载时取数 | 项目包装层的取数由 `useProjectFetch`（`core/layouts/auth-layout/use-project-fetch.ts`）决定：先取项目，nerve 说调用者是有效成员之后才取显示设置、标签、成员和状态；不是成员的看得到的项目显示"加入项目"，看不到的显示"找不到项目"（nerve 对看不到的项目答 404，Plane 的 403 界面删除）；工作区包装层取项目列表和工作区的状态，项目角色取自 `Project.member_role`，`project-roles` 的取数删除 | M3 设计 7.1、7.3 | 已完成 | M3/P8b |
````

`docs/v0/v0-design.md`（修改，3 处）：

````old docs/v0/v0-design.md
- **填充 stores 的 SWR 键带上 `loginId`**：会话的每个经 SWR 的取数都经 `useSessionSWR`（`core/lib/use-session-swr.ts`），键只有 `sessionKey`（`core/lib/session-key.ts`）一种写法：`[取数的名称, loginId, ...取数的参数]`（例如 `["WORKSPACE_MEMBERS", loginId, workspaceId, slug]`）；标签页不在已登录的状态（`signed-in`）或有参数未知时是 `null`，不取。换了会话，新的一代取自己的数据。fetcher 收到键的参数，返回 store 的 Promise：取数失败是 SWR 的 `error`，不是未处理的拒绝（M3/P8a）。会话的取数只有一份 SWR 配置，在 `useSessionSWR` 里，调用方不传：`{ revalidateOnFocus: false, shouldRetryOnError: false }`，挂载和键变化的事不写，由应用的配置（`WEB_SWR_CONFIG`）决定：页面每次挂载时取，靠的是 `revalidateOnMount: true`；已挂载的页面换了键时取，SWR 没有这个键的回答时总是取，有时靠的是 `revalidateIfStale: true`；窗口聚焦时不取，被拒绝不重试（M3/P8a；同一个键不会有两份不同的配置）。唯一的例外是不属于会话的公开操作：经 `publicClient` 调用，回答对任何会话和没有会话都一样，键是取数的名称和它的输入，不带 `loginId`（例如查看邀请的链接，键是 `["INVITATION_PREVIEW", invitationId, token]`，`core/hooks/use-invitation-preview.ts`，M3 设计 7.1；M2 的实例信息、时区列表也是这样的公开操作，没有输入，键只有名称）。根目录 `.oxlintrc.json` 的 `overrides` 中 `no-restricted-imports` 的一条静态地看住这一条：列出的文件不能从 `swr` 或 `swr/*`（`swr/immutable`、`swr/infinite` 等）导入值，只能导入类型（M3/P8a 起是工作区一侧的 store、工作区设置的组件（`core/components/workspace/settings/`，成员页的取数 `use-members-settings-fetch.ts` 在其中）、取数的 hook `use-workspace-fetch.ts`、`use-workspace-members-fetch.ts`、`use-profile-member.ts`、`use-landing.ts`，`authentication-wrapper.tsx`，令牌列表 `token-list.tsx`，以及读邀请查看的邀请页 `workspace-invitations/page.tsx` 和 `auth-header.tsx`）；之后的 M 把接上新接口的取数文件加进去（M3/P8b 加上工作区和项目的包装层）。
````
````new docs/v0/v0-design.md
- **填充 stores 的 SWR 键带上 `loginId`**：会话的每个经 SWR 的取数都经 `useSessionSWR`（`core/lib/use-session-swr.ts`），键只有 `sessionKey`（`core/lib/session-key.ts`）一种写法：`[取数的名称, loginId, ...取数的参数]`（例如 `["WORKSPACE_MEMBERS", loginId, workspaceId, slug]`）；标签页不在已登录的状态（`signed-in`）或有参数未知时是 `null`，不取。换了会话，新的一代取自己的数据。fetcher 收到键的参数，返回 store 的 Promise：取数失败是 SWR 的 `error`，不是未处理的拒绝（M3/P8a）。会话的取数只有一份 SWR 配置，在 `useSessionSWR` 里，调用方不传：`{ revalidateOnFocus: false, shouldRetryOnError: false }`，挂载和键变化的事不写，由应用的配置（`WEB_SWR_CONFIG`）决定：页面每次挂载时取，靠的是 `revalidateOnMount: true`；已挂载的页面换了键时取，SWR 没有这个键的回答时总是取，有时靠的是 `revalidateIfStale: true`；窗口聚焦时不取，被拒绝不重试（M3/P8a；同一个键不会有两份不同的配置）。唯一的例外是不属于会话的公开操作：经 `publicClient` 调用，回答对任何会话和没有会话都一样，键是取数的名称和它的输入，不带 `loginId`（例如查看邀请的链接，键是 `["INVITATION_PREVIEW", invitationId, token]`，`core/hooks/use-invitation-preview.ts`，M3 设计 7.1；M2 的实例信息、时区列表也是这样的公开操作，没有输入，键只有名称）。根目录 `.oxlintrc.json` 的 `overrides` 中 `no-restricted-imports` 的一条静态地看住这一条：列出的文件不能从 `swr` 或 `swr/*`（`swr/immutable`、`swr/infinite` 等）导入值，只能导入类型（M3/P8a 起是工作区一侧的 store、工作区设置的组件（`core/components/workspace/settings/`，成员页的取数 `use-members-settings-fetch.ts` 在其中）、取数的 hook `use-workspace-fetch.ts`、`use-workspace-members-fetch.ts`、`use-profile-member.ts`、`use-landing.ts`，`authentication-wrapper.tsx`，令牌列表 `token-list.tsx`，以及读邀请查看的邀请页 `workspace-invitations/page.tsx` 和 `auth-header.tsx`；M3/P8b 起加上项目一侧的 store（`core/store/project/` 中的项目和项目的显示设置，`core/store/member/project/` 的项目成员，`state.store.ts`、`label.store.ts`）、两个包装层 `workspace-wrapper.tsx`、`project-wrapper.tsx` 和后者的取数 hook `use-project-fetch.ts`、项目列表 `core/components/projects/page.tsx`、状态和标签的设置组件（`core/components/project-states/`、`core/components/labels/`）和项目设置的成员默认值 `project-settings-member-defaults.tsx`）；之后的 M 把接上新接口的取数文件加进去。
````
````old docs/v0/v0-design.md
- **页面按权限决定取数，不只决定显示**：页面只请求调用者有权读的资源；按权限隐藏的区域，它的取数同样按权限启用（没有权限时 `useSessionSWR` 的取数是 `null`）。例如工作区设置的成员页只在调用者是工作区的管理员时取邀请列表（M3 设计 7.1）。
````
````new docs/v0/v0-design.md
- **页面按权限决定取数，不只决定显示**：页面只请求调用者有权读的资源；按权限隐藏的区域，它的取数同样按权限启用（没有权限时 `useSessionSWR` 的取数是 `null`）。例如工作区设置的成员页只在调用者是工作区的管理员时取邀请列表；项目包装层只在 nerve 对项目的回答说调用者是它的有效成员（`member_role` 不为 `null`）之后才取项目的显示设置、标签、成员和状态（M3 设计 7.1，`use-project-fetch.ts`）。
````
````old docs/v0/v0-design.md
- **取数和修改的应答对齐**：取数不排队，所以一个取数可以在修改的应答之前或之后答到。store 保存的每个值（按键的，每个键一个）都有这样的性质：每个键只有最新的取数写入，较旧的取数不论答到什么（回答或失败）都什么都不写、给出 `undefined`，由较新的那个决定；取数在外时 nerve 确认的修改按确认的顺序重放到它的回答上；修改是这个值的幂等函数（按 id 写），重放到已经含有它的回答上不重复；还没有取过的值，修改不写，留给取数；取数遇到 `SessionChangedError` 给出 `undefined`，不改这一代的状态。它不假定队列（排队是上一条的事）。这个性质只有一份实现：`core/lib/reconciled.ts`（`Reconciled`、按键的 `ReconciledByKey`，和按 id 的修改 `prepended`、`replaced`、`upserted`、`dropped`），M3/P8a 的工作区、成员、邀请、显示设置和令牌的 store 是它最先的使用者。之后的 M 用它，不另写一份；一个集合的形状不是"一次取数写整个值"时（例如 M4 按游标分页、按组返回的列表），扩展这份实现，不复制它。
````
````new docs/v0/v0-design.md
- **取数和修改的应答对齐**：取数不排队，所以一个取数可以在修改的应答之前或之后答到。store 保存的每个值（按键的，每个键一个）都有这样的性质：每个键只有最新的取数写入，较旧的取数不论答到什么（回答或失败）都什么都不写、给出 `undefined`，由较新的那个决定；取数在外时 nerve 确认的修改按确认的顺序重放到它的回答上；修改是这个值的幂等函数（按 id 写），重放到已经含有它的回答上不重复；还没有取过的值，修改不写，留给取数；取数遇到 `SessionChangedError` 给出 `undefined`，不改这一代的状态。它不假定队列（排队是上一条的事）。这个性质只有一份实现：`core/lib/reconciled.ts`（`Reconciled`、按键的 `ReconciledByKey`，和按 id 的修改 `prepended`、`replaced`、`upserted`、`dropped`），M3/P8a 的工作区、成员、邀请、显示设置和令牌的 store 是它最先的使用者，M3/P8b 的项目、项目成员、状态、标签和项目的显示设置的 store 也用它（按项目的值用 `ReconciledByKey`，在任一个项目的值里找一项用它的 `values()`）。之后的 M 用它，不另写一份；一个集合的形状不是"一次取数写整个值"时（例如 M4 按游标分页、按组返回的列表），扩展这份实现，不复制它。
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外（加了 `plane-workspace-urls` 的 `until: "M4"`），没有命中；web 的 oxlint 367 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 74 个全部通过（S2 6 个）。

- [ ] **Step 6: 提交**

```bash
git add docs/v0/frontend-changes.md docs/v0/v0-design.md e2e/stories/smoke/s2-web-app.spec.ts tools/keywords.json web/apps/web/core/lib/store-context.test.ts web/apps/web/core/store/project/project_filter.store.ts
```
```bash
git commit -m "test(M3/P8b): S2 checks every signed-in mount list, and the keyword rules cover the project side

S2 signs in as the workspace's admin, member, guest and a member who is
not in its project, and checks that the home and the project's settings
ask nerve for exactly their lists: no failed or old request, no address
of M6's or M7's, and nothing of the project's own for one who is not
its member. plane-workspace-urls now names the project side's old
addresses, with the M4 exception 7.10 gives, and plane-user-urls covers
the whole of /api/users/. The project filters' dead getters go, and the
overall design and the front-end change list follow P8b.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p8b.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `t11-cycles-back` | 项目包装层又取项目的迭代（M6 的，在项目的每一页） | 故事 S2 | 端到端 |
| `t11-unread-back` | 顶部导航又取未读通知数（M7 的，在每一页） | 故事 S2 | 端到端 |
| `t11-old-members-back` | 旧的项目成员地址回到项目的挂载路径上 | 关键词守卫、故事 S2 | 静态；端到端 |
| `t11-user-urls-narrow` | `plane-user-urls` 又只禁止 `/api/users/me/` 本身，不是整个 `/api/users/` | 关键词守卫 | 静态 |
| `t11-workspace-urls-narrow` | `plane-workspace-urls` 不再写项目角色的地址 | 关键词守卫 | 静态 |
| `t11-search-issues` | `plane-workspace-urls` 的项目地址放宽到项目的任何地址（`/search-issues/` 也命中） | 关键词守卫 | 静态 |
| `t11-filter-dispose` | `RootStore.dispose()` 不释放项目筛选的反应 | `store-context.test.ts` | vitest |
